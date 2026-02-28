package collector

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joluc/weather-exporter/internal/config"
	"github.com/joluc/weather-exporter/internal/provider"
)

type MetricSample struct {
	Name   string
	Labels map[string]string
	Value  float64
}

type WeatherCollector struct {
	providers []provider.Provider
	cities    []config.City
	logger    *slog.Logger

	// Cache
	mu          sync.Mutex
	cacheTTL    time.Duration
	lastScrape  time.Time
	cachedData  []MetricSample
}

func NewWeatherCollector(providers []provider.Provider, cities []config.City, logger *slog.Logger, cacheTTL time.Duration) *WeatherCollector {
	return &WeatherCollector{
		providers:  providers,
		cities:     cities,
		logger:     logger,
		cacheTTL:   cacheTTL,
		cachedData: []MetricSample{},
	}
}

func (c *WeatherCollector) filterProvidersForCity(city config.City) []provider.Provider {
	// If no provider filter specified, use all providers (backward compatible)
	if len(city.Providers) == 0 {
		return c.providers
	}

	// Build a map of requested provider names for O(1) lookup
	requested := make(map[string]bool, len(city.Providers))
	for _, name := range city.Providers {
		requested[name] = true
	}

	// Filter providers to only those requested for this city
	filtered := make([]provider.Provider, 0, len(city.Providers))
	for _, p := range c.providers {
		if requested[p.Name()] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (c *WeatherCollector) Collect(ctx context.Context) []MetricSample {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache is still valid
	if time.Since(c.lastScrape) < c.cacheTTL && len(c.cachedData) > 0 {
		c.logger.Debug("serving from cache",
			slog.Duration("age", time.Since(c.lastScrape)),
			slog.Duration("ttl", c.cacheTTL))

		// Add cache age metric
		cacheAge := time.Since(c.lastScrape).Seconds()
		result := append([]MetricSample{}, c.cachedData...)
		result = append(result, MetricSample{
			Name:   "weather_cache_age_seconds",
			Labels: map[string]string{},
			Value:  cacheAge,
		})
		return result
	}

	// Cache miss or expired - fetch new data
	c.logger.Debug("cache miss, fetching fresh data")

	type result struct {
		ProviderName string
		CityName     string
		Data         provider.WeatherData
		Err          error
	}

	var wg sync.WaitGroup
	resultCh := make(chan result, len(c.providers)*len(c.cities))
	samples := make([]MetricSample, 0, len(c.providers)*len(c.cities)*5)

	for _, city := range c.cities {
		filteredProviders := c.filterProvidersForCity(city)
		for _, p := range filteredProviders {
			wg.Add(1)
			go func(p provider.Provider, city config.City) {
				defer wg.Done()
				data, err := p.Fetch(ctx, city.Lat, city.Lon)
				resultCh <- result{
					ProviderName: p.Name(),
					CityName:     city.Name,
					Data:         data,
					Err:          err,
				}
			}(p, city)
		}
	}

	wg.Wait()
	close(resultCh)

	for r := range resultCh {
		if r.Err != nil {
			c.logger.Error("provider fetch failed",
				slog.String("provider", r.ProviderName),
				slog.String("city", r.CityName),
				slog.Any("error", r.Err))
			samples = append(samples, sample("weather_provider_up", 0, r.ProviderName, r.CityName))
			continue
		}
		samples = append(samples, sample("weather_provider_up", 1, r.ProviderName, r.CityName))

		emitIfSet(&samples, "weather_temperature_celsius", r.Data.TemperatureCelsius, r.ProviderName, r.CityName)
		emitIfSet(&samples, "weather_humidity_relative", r.Data.HumidityRelative, r.ProviderName, r.CityName)
		emitIfSet(&samples, "weather_pressure_hpa", r.Data.PressureHPA, r.ProviderName, r.CityName)
		emitIfSet(&samples, "weather_wind_speed_mps", r.Data.WindSpeedMPS, r.ProviderName, r.CityName)
		emitIfSet(&samples, "weather_wind_direction_degrees", r.Data.WindDirectionDegrees, r.ProviderName, r.CityName)
		emitIfSet(&samples, "weather_precipitation_mm", r.Data.PrecipitationMM, r.ProviderName, r.CityName)
		emitIfSet(&samples, "weather_cloud_cover_percent", r.Data.CloudCoverPercent, r.ProviderName, r.CityName)
		emitIfSet(&samples, "weather_visibility_meters", r.Data.VisibilityMeters, r.ProviderName, r.CityName)
	}

	// Update cache
	c.cachedData = samples
	c.lastScrape = time.Now()

	// Add cache age metric (0 for fresh data)
	output := append([]MetricSample{}, samples...)
	output = append(output, MetricSample{
		Name:   "weather_cache_age_seconds",
		Labels: map[string]string{},
		Value:  0,
	})

	return output
}

func emitIfSet(samples *[]MetricSample, metricName string, value *float64, providerName, cityName string) {
	if value == nil {
		return
	}
	*samples = append(*samples, sample(metricName, *value, providerName, cityName))
}

func sample(name string, value float64, providerName, cityName string) MetricSample {
	return MetricSample{
		Name: name,
		Labels: map[string]string{
			"provider": providerName,
			"city":     cityName,
		},
		Value: value,
	}
}

var metricHelp = map[string]string{
	"weather_temperature_celsius":    "Air temperature at 2m height.",
	"weather_humidity_relative":      "Relative humidity percentage.",
	"weather_pressure_hpa":           "Atmospheric pressure at sea level in hPa.",
	"weather_wind_speed_mps":         "Wind speed in meters per second.",
	"weather_wind_direction_degrees": "Wind direction in degrees.",
	"weather_precipitation_mm":       "Precipitation amount in millimeters.",
	"weather_cloud_cover_percent":    "Cloud cover percentage.",
	"weather_visibility_meters":      "Horizontal visibility in meters.",
	"weather_provider_up":            "Whether a provider/city fetch was successful.",
	"weather_cache_age_seconds":      "Age of cached data in seconds.",
}

var metricOrder = []string{
	"weather_temperature_celsius",
	"weather_humidity_relative",
	"weather_pressure_hpa",
	"weather_wind_speed_mps",
	"weather_wind_direction_degrees",
	"weather_precipitation_mm",
	"weather_cloud_cover_percent",
	"weather_visibility_meters",
	"weather_provider_up",
	"weather_cache_age_seconds",
}

func RenderPrometheus(samples []MetricSample) string {
	grouped := make(map[string][]MetricSample, len(metricOrder))
	for _, s := range samples {
		grouped[s.Name] = append(grouped[s.Name], s)
	}

	var b strings.Builder
	for _, metricName := range metricOrder {
		items := grouped[metricName]
		if len(items) == 0 {
			continue
		}

		sort.Slice(items, func(i, j int) bool {
			left := items[i].Labels["provider"] + "\x00" + items[i].Labels["city"]
			right := items[j].Labels["provider"] + "\x00" + items[j].Labels["city"]
			return left < right
		})

		fmt.Fprintf(&b, "# HELP %s %s\n", metricName, metricHelp[metricName])
		fmt.Fprintf(&b, "# TYPE %s gauge\n", metricName)
		for _, item := range items {
			fmt.Fprintf(
				&b,
				"%s{provider=\"%s\",city=\"%s\"} %s\n",
				metricName,
				escapeLabel(item.Labels["provider"]),
				escapeLabel(item.Labels["city"]),
				strconv.FormatFloat(item.Value, 'f', -1, 64),
			)
		}
	}
	return b.String()
}

func escapeLabel(v string) string {
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "\n", "\\n")
	v = strings.ReplaceAll(v, "\"", "\\\"")
	return v
}
