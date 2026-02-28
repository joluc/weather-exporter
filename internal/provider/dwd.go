package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const dwdEndpoint = "https://api.open-meteo.com/v1/forecast"

type DWDProvider struct {
	client *http.Client
}

func NewDWDProvider() *DWDProvider {
	return &DWDProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *DWDProvider) Init(_ string) error {
	return nil
}

func (p *DWDProvider) Name() string {
	return "dwd"
}

func (p *DWDProvider) Fetch(ctx context.Context, lat, lon float64) (WeatherData, error) {
	params := url.Values{}
	params.Set("latitude", strconv.FormatFloat(lat, 'f', 6, 64))
	params.Set("longitude", strconv.FormatFloat(lon, 'f', 6, 64))
	params.Set("timezone", "UTC")
	params.Set("current", "temperature_2m,relative_humidity_2m,pressure_msl,wind_speed_10m,wind_direction_10m,precipitation,cloud_cover,visibility")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dwdEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return WeatherData{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return WeatherData{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var payload struct {
		Current struct {
			Temperature2M      *float64 `json:"temperature_2m"`
			RelativeHumidity2M *float64 `json:"relative_humidity_2m"`
			PressureMSL        *float64 `json:"pressure_msl"`
			WindSpeed10M       *float64 `json:"wind_speed_10m"`
			WindDirection10M   *float64 `json:"wind_direction_10m"`
			Precipitation      *float64 `json:"precipitation"`
			CloudCover         *float64 `json:"cloud_cover"`
			Visibility         *float64 `json:"visibility"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return WeatherData{}, fmt.Errorf("decode response: %w", err)
	}

	return WeatherData{
		TemperatureCelsius:   payload.Current.Temperature2M,
		HumidityRelative:     payload.Current.RelativeHumidity2M,
		PressureHPA:          payload.Current.PressureMSL,
		WindSpeedMPS:         payload.Current.WindSpeed10M,
		WindDirectionDegrees: payload.Current.WindDirection10M,
		PrecipitationMM:      payload.Current.Precipitation,
		CloudCoverPercent:    payload.Current.CloudCover,
		VisibilityMeters:     payload.Current.Visibility,
	}, nil
}
