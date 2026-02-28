package provider

import "context"

type Provider interface {
	Init(apiKey string) error
	Fetch(ctx context.Context, lat, lon float64) (WeatherData, error)
	Name() string
}

type WeatherData struct {
	TemperatureCelsius   *float64
	HumidityRelative     *float64
	PressureHPA          *float64
	WindSpeedMPS         *float64
	WindDirectionDegrees *float64
	PrecipitationMM      *float64
	CloudCoverPercent    *float64
	VisibilityMeters     *float64
}
