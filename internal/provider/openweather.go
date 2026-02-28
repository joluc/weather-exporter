package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const openWeatherEndpoint = "https://api.openweathermap.org/data/2.5/weather"

type OpenWeatherProvider struct {
	client *http.Client
	apiKey string
}

func NewOpenWeatherProvider() *OpenWeatherProvider {
	return &OpenWeatherProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *OpenWeatherProvider) Init(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("OPENWEATHER_API_KEY is required")
	}
	p.apiKey = apiKey
	return nil
}

func (p *OpenWeatherProvider) Name() string {
	return "openweathermap"
}

func (p *OpenWeatherProvider) Fetch(ctx context.Context, lat, lon float64) (WeatherData, error) {
	params := url.Values{}
	params.Set("lat", strconv.FormatFloat(lat, 'f', 6, 64))
	params.Set("lon", strconv.FormatFloat(lon, 'f', 6, 64))
	params.Set("units", "metric")
	params.Set("appid", p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openWeatherEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return WeatherData{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return WeatherData{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody []byte
		if resp.Body != nil {
			errBody, _ = io.ReadAll(resp.Body)
		}
		return WeatherData{}, fmt.Errorf("unexpected status: %s (body: %s)", resp.Status, string(errBody))
	}

	var payload struct {
		Main struct {
			Temp     float64 `json:"temp"`
			Humidity float64 `json:"humidity"`
			Pressure float64 `json:"pressure"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   float64 `json:"deg"`
		} `json:"wind"`
		Clouds struct {
			All float64 `json:"all"`
		} `json:"clouds"`
		Visibility *float64           `json:"visibility"`
		Rain       map[string]float64 `json:"rain"`
		Snow       map[string]float64 `json:"snow"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return WeatherData{}, fmt.Errorf("decode response: %w", err)
	}

	precip := 0.0
	hasPrecip := false
	if v, ok := payload.Rain["1h"]; ok {
		precip += v
		hasPrecip = true
	}
	if v, ok := payload.Snow["1h"]; ok {
		precip += v
		hasPrecip = true
	}

	data := WeatherData{
		TemperatureCelsius:   new(payload.Main.Temp),
		HumidityRelative:     new(payload.Main.Humidity),
		PressureHPA:          new(payload.Main.Pressure),
		WindSpeedMPS:         new(payload.Wind.Speed),
		WindDirectionDegrees: new(payload.Wind.Deg),
		CloudCoverPercent:    new(payload.Clouds.All),
		VisibilityMeters:     payload.Visibility,
	}
	if hasPrecip {
		data.PrecipitationMM = new(precip)
	}

	return data, nil
}
