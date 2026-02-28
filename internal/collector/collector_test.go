package collector

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/joluc/weather-exporter/internal/config"
	"github.com/joluc/weather-exporter/internal/provider"
)

type fakeProvider struct {
	name string
	data provider.WeatherData
	err  error
}

func (p fakeProvider) Init(_ string) error {
	return nil
}

func (p fakeProvider) Fetch(_ context.Context, _, _ float64) (provider.WeatherData, error) {
	if p.err != nil {
		return provider.WeatherData{}, p.err
	}
	return p.data, nil
}

func (p fakeProvider) Name() string {
	return p.name
}

func TestCollectorEmitsUpAndDataMetrics(t *testing.T) {
	t.Parallel()

	successProvider := fakeProvider{
		name: "ok-provider",
		data: provider.WeatherData{
			TemperatureCelsius: new(10.5),
		},
	}
	failedProvider := fakeProvider{
		name: "bad-provider",
		err:  errors.New("fetch failed"),
	}

	c := NewWeatherCollector(
		[]provider.Provider{successProvider, failedProvider},
		[]config.City{{Name: "Leipzig", Lat: 51.33, Lon: 12.37}},
		slog.Default(),
		0, // No caching for tests
	)

	samples := c.Collect(context.Background())
	if countMetric(samples, "weather_provider_up") != 2 {
		t.Fatalf("expected 2 up metrics, got %d", countMetric(samples, "weather_provider_up"))
	}
	if countMetric(samples, "weather_temperature_celsius") != 1 {
		t.Fatalf("expected 1 temperature metric, got %d", countMetric(samples, "weather_temperature_celsius"))
	}
}

func TestRenderPrometheus(t *testing.T) {
	t.Parallel()

	text := RenderPrometheus([]MetricSample{
		{
			Name: "weather_temperature_celsius",
			Labels: map[string]string{
				"provider": "yr",
				"city":     "Leipzig",
			},
			Value: 12.3,
		},
	})
	if !strings.Contains(text, "# HELP weather_temperature_celsius") {
		t.Fatalf("expected HELP line, got:\n%s", text)
	}
	if !strings.Contains(text, `weather_temperature_celsius{provider="yr",city="Leipzig"} 12.3`) {
		t.Fatalf("expected metric line, got:\n%s", text)
	}
}

func countMetric(samples []MetricSample, metricName string) int {
	count := 0
	for _, sample := range samples {
		if sample.Name == metricName {
			count++
		}
	}
	return count
}

func findSample(samples []MetricSample, metricName, providerName, cityName string) *MetricSample {
	for _, s := range samples {
		if s.Name == metricName && s.Labels["provider"] == providerName && s.Labels["city"] == cityName {
			return &s
		}
	}
	return nil
}

func TestCollectorRespectsPerCityProviderFilter(t *testing.T) {
	t.Parallel()

	provider1 := fakeProvider{name: "provider1", data: provider.WeatherData{TemperatureCelsius: new(10.0)}}
	provider2 := fakeProvider{name: "provider2", data: provider.WeatherData{TemperatureCelsius: new(20.0)}}

	cities := []config.City{
		{Name: "City1", Lat: 51.33, Lon: 12.37, Providers: []string{"provider1"}},
		{Name: "City2", Lat: 55.67, Lon: 12.56, Providers: []string{"provider2"}},
	}

	c := NewWeatherCollector(
		[]provider.Provider{provider1, provider2},
		cities,
		slog.Default(),
		0, // No caching for tests
	)

	samples := c.Collect(context.Background())

	// City1 should only have provider1 metrics
	if s := findSample(samples, "weather_temperature_celsius", "provider1", "City1"); s == nil {
		t.Fatal("expected temperature metric for provider1 in City1")
	}
	if s := findSample(samples, "weather_temperature_celsius", "provider2", "City1"); s != nil {
		t.Fatal("did not expect temperature metric for provider2 in City1")
	}

	// City2 should only have provider2 metrics
	if s := findSample(samples, "weather_temperature_celsius", "provider2", "City2"); s == nil {
		t.Fatal("expected temperature metric for provider2 in City2")
	}
	if s := findSample(samples, "weather_temperature_celsius", "provider1", "City2"); s != nil {
		t.Fatal("did not expect temperature metric for provider1 in City2")
	}
}

func TestCollectorBackwardCompatibilityNoFilter(t *testing.T) {
	t.Parallel()

	provider1 := fakeProvider{name: "provider1", data: provider.WeatherData{TemperatureCelsius: new(10.0)}}
	provider2 := fakeProvider{name: "provider2", data: provider.WeatherData{TemperatureCelsius: new(20.0)}}

	cities := []config.City{
		{Name: "City1", Lat: 51.33, Lon: 12.37, Providers: nil}, // No filter = use all
	}

	c := NewWeatherCollector(
		[]provider.Provider{provider1, provider2},
		cities,
		slog.Default(),
		0, // No caching for tests
	)

	samples := c.Collect(context.Background())

	// City1 should have metrics from both providers
	if s := findSample(samples, "weather_temperature_celsius", "provider1", "City1"); s == nil {
		t.Fatal("expected temperature metric for provider1 in City1")
	}
	if s := findSample(samples, "weather_temperature_celsius", "provider2", "City1"); s == nil {
		t.Fatal("expected temperature metric for provider2 in City1")
	}
}
