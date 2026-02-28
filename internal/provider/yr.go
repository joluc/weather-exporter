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

const (
	yrEndpoint         = "https://api.met.no/weatherapi/locationforecast/2.0/compact"
	defaultYRUserAgent = "weather-exporter/1.0"
)

type YRProvider struct {
	client    *http.Client
	userAgent string
}

func NewYRProvider(userAgent string) *YRProvider {
	if userAgent == "" {
		userAgent = defaultYRUserAgent
	}
	return &YRProvider{
		client:    &http.Client{Timeout: 10 * time.Second},
		userAgent: userAgent,
	}
}

func (p *YRProvider) Init(_ string) error {
	if p.userAgent == "" {
		return fmt.Errorf("user agent must not be empty")
	}
	return nil
}

func (p *YRProvider) Name() string {
	return "yr"
}

func (p *YRProvider) Fetch(ctx context.Context, lat, lon float64) (WeatherData, error) {
	params := url.Values{}
	params.Set("lat", strconv.FormatFloat(lat, 'f', 6, 64))
	params.Set("lon", strconv.FormatFloat(lon, 'f', 6, 64))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, yrEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return WeatherData{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", p.userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return WeatherData{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var payload struct {
		Properties struct {
			Timeseries []struct {
				Data struct {
					Instant struct {
						Details struct {
							AirTemperature        float64 `json:"air_temperature"`
							RelativeHumidity      float64 `json:"relative_humidity"`
							AirPressureAtSeaLevel float64 `json:"air_pressure_at_sea_level"`
							WindSpeed             float64 `json:"wind_speed"`
							WindFromDirection     float64 `json:"wind_from_direction"`
							CloudAreaFraction     float64 `json:"cloud_area_fraction"`
						} `json:"details"`
					} `json:"instant"`
					Next1Hours *struct {
						Details struct {
							PrecipitationAmount float64 `json:"precipitation_amount"`
						} `json:"details"`
					} `json:"next_1_hours"`
				} `json:"data"`
			} `json:"timeseries"`
		} `json:"properties"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return WeatherData{}, fmt.Errorf("decode response: %w", err)
	}
	if len(payload.Properties.Timeseries) == 0 {
		return WeatherData{}, fmt.Errorf("timeseries is empty")
	}

	details := payload.Properties.Timeseries[0].Data.Instant.Details
	data := WeatherData{
		TemperatureCelsius:   new(details.AirTemperature),
		HumidityRelative:     new(details.RelativeHumidity),
		PressureHPA:          new(details.AirPressureAtSeaLevel),
		WindSpeedMPS:         new(details.WindSpeed),
		WindDirectionDegrees: new(details.WindFromDirection),
		CloudCoverPercent:    new(details.CloudAreaFraction),
	}

	if payload.Properties.Timeseries[0].Data.Next1Hours != nil {
		precip := payload.Properties.Timeseries[0].Data.Next1Hours.Details.PrecipitationAmount
		data.PrecipitationMM = new(precip)
	}

	return data, nil
}
