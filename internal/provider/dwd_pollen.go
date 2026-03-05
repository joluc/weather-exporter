package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const dwdPollenEndpoint = "https://opendata.dwd.de/climate_environment/health/alerts/s31fg.json"

// PollenData holds pollen risk index values for different types
type PollenData struct {
	Region     string
	RegionID   int
	LastUpdate time.Time
	Values     map[string]float64 // pollen type -> risk index
}

type DWDPollenProvider struct {
	client *http.Client
}

func NewDWDPollenProvider() *DWDPollenProvider {
	return &DWDPollenProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *DWDPollenProvider) Init(_ string) error {
	return nil
}

func (p *DWDPollenProvider) Name() string {
	return "dwd-pollen"
}

// dwdPollenResponse represents the structure of the DWD pollen API response
type dwdPollenResponse struct {
	LastUpdate string `json:"last_update"`
	NextUpdate string `json:"next_update"`
	Content    []struct {
		PartregionID   int    `json:"partregion_id"`
		PartregionName string `json:"partregion_name"`
		Pollen         struct {
			Hazel    map[string]string `json:"Hasel"`
			Alder    map[string]string `json:"Erle"`
			Ash      map[string]string `json:"Esche"`
			Birch    map[string]string `json:"Birke"`
			Grasses  map[string]string `json:"Graeser"`
			Rye      map[string]string `json:"Roggen"`
			Mugwort  map[string]string `json:"Beifuss"`
			Ambrosia map[string]string `json:"Ambrosia"`
		} `json:"Pollen"`
	} `json:"content"`
}

// mapPollenValue converts DWD string values to numeric risk indices
func mapPollenValue(value string) float64 {
	switch value {
	case "0":
		return 0.0
	case "0-1":
		return 0.5
	case "1":
		return 1.0
	case "1-2":
		return 1.5
	case "2":
		return 2.0
	case "2-3":
		return 2.5
	case "3":
		return 3.0
	case "-1":
		return -1.0
	default:
		return -1.0
	}
}

// getTodayValue extracts today's pollen value from the daily map
func getTodayValue(dailyMap map[string]string) float64 {
	// The API returns "today", "tomorrow", and sometimes "dayafter_to"
	// We prioritize "today"
	if val, ok := dailyMap["today"]; ok {
		return mapPollenValue(val)
	}
	return -1.0
}

func (p *DWDPollenProvider) FetchPollen(ctx context.Context, lat, lon float64) (PollenData, error) {
	// Map coordinates to DWD region
	region, found := GetDWDRegionByCoordinates(lat, lon)
	if !found {
		return PollenData{}, fmt.Errorf("coordinates %.2f,%.2f not within DWD coverage area", lat, lon)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dwdPollenEndpoint, nil)
	if err != nil {
		return PollenData{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return PollenData{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PollenData{}, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var payload dwdPollenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return PollenData{}, fmt.Errorf("decode response: %w", err)
	}

	// Find the matching region in the response
	var regionIndex = -1
	for i := range payload.Content {
		if payload.Content[i].PartregionID == region.ID {
			regionIndex = i
			break
		}
	}

	if regionIndex == -1 {
		return PollenData{}, fmt.Errorf("region %d (%s) not found in API response", region.ID, region.Name)
	}

	regionData := &payload.Content[regionIndex]

	// Parse last_update timestamp
	lastUpdate := time.Now() // fallback
	if payload.LastUpdate != "" {
		if t, err := time.Parse(time.RFC3339, payload.LastUpdate); err == nil {
			lastUpdate = t
		}
	}

	// Build pollen values map
	values := map[string]float64{
		"hazel":    getTodayValue(regionData.Pollen.Hazel),
		"alder":    getTodayValue(regionData.Pollen.Alder),
		"ash":      getTodayValue(regionData.Pollen.Ash),
		"birch":    getTodayValue(regionData.Pollen.Birch),
		"grasses":  getTodayValue(regionData.Pollen.Grasses),
		"rye":      getTodayValue(regionData.Pollen.Rye),
		"mugwort":  getTodayValue(regionData.Pollen.Mugwort),
		"ambrosia": getTodayValue(regionData.Pollen.Ambrosia),
	}

	return PollenData{
		Region:     region.Name,
		RegionID:   region.ID,
		LastUpdate: lastUpdate,
		Values:     values,
	}, nil
}

// PollenProvider interface for providers that support pollen data
type PollenProvider interface {
	Provider
	FetchPollen(ctx context.Context, lat, lon float64) (PollenData, error)
}

// Fetch is required by the Provider interface but not used for pollen provider
// Pollen data is fetched via FetchPollen instead
func (p *DWDPollenProvider) Fetch(_ context.Context, _, _ float64) (WeatherData, error) {
	return WeatherData{}, fmt.Errorf("DWDPollenProvider does not support weather data, use FetchPollen instead")
}
