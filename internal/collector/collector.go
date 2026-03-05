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

	type weatherResult struct {
		ProviderName string
		CityName     string
		Data         provider.WeatherData
		Err          error
	}

	type pollenResult struct {
		ProviderName string
		CityName     string
		Data         provider.PollenData
		Err          error
	}

	var wg sync.WaitGroup
	weatherResultCh := make(chan weatherResult, len(c.providers)*len(c.cities))
	pollenResultCh := make(chan pollenResult, len(c.providers)*len(c.cities))
	samples := make([]MetricSample, 0, len(c.providers)*len(c.cities)*5)

	for _, city := range c.cities {
		filteredProviders := c.filterProvidersForCity(city)
		for _, p := range filteredProviders {
			// Check if this is a pollen provider
			if pollenProvider, ok := p.(provider.PollenProvider); ok {
				wg.Add(1)
				go func(p provider.PollenProvider, city config.City) {
					defer wg.Done()
					data, err := p.FetchPollen(ctx, city.Lat, city.Lon)
					pollenResultCh <- pollenResult{
						ProviderName: p.Name(),
						CityName:     city.Name,
						Data:         data,
						Err:          err,
					}
				}(pollenProvider, city)
			} else {
				wg.Add(1)
				go func(p provider.Provider, city config.City) {
					defer wg.Done()
					data, err := p.Fetch(ctx, city.Lat, city.Lon)
					weatherResultCh <- weatherResult{
						ProviderName: p.Name(),
						CityName:     city.Name,
						Data:         data,
						Err:          err,
					}
				}(p, city)
			}
		}
	}

	go func() {
		wg.Wait()
		close(weatherResultCh)
		close(pollenResultCh)
	}()

	// Process weather results
	for r := range weatherResultCh {
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

	// Process pollen results
	for r := range pollenResultCh {
		if r.Err != nil {
			c.logger.Error("pollen provider fetch failed",
				slog.String("provider", r.ProviderName),
				slog.String("city", r.CityName),
				slog.Any("error", r.Err))
			samples = append(samples, sample("weather_provider_up", 0, r.ProviderName, r.CityName))
			continue
		}
		samples = append(samples, sample("weather_provider_up", 1, r.ProviderName, r.CityName))

		// Emit pollen metrics
		for pollenType, value := range r.Data.Values {
			if value >= 0 { // Skip -1 (no data) values
				samples = append(samples, MetricSample{
					Name: "pollen_risk_index",
					Labels: map[string]string{
						"provider":    r.ProviderName,
						"city":        r.CityName,
						"region":      r.Data.Region,
						"pollen_type": pollenType,
					},
					Value: value,
				})
			}
		}
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
	"pollen_risk_index":              "Pollen risk index (0=none, 1=low, 2=medium, 3=high, -1=no data).",
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
	"pollen_risk_index",
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

		// Build sort key based on all labels
		sort.Slice(items, func(i, j int) bool {
			// Sort by provider, then city, then all other labels
			left := items[i].Labels["provider"] + "\x00" + items[i].Labels["city"]
			right := items[j].Labels["provider"] + "\x00" + items[j].Labels["city"]
			if left != right {
				return left < right
			}
			// For pollen metrics, also sort by region and pollen_type
			if region, ok := items[i].Labels["region"]; ok {
				left += "\x00" + region
			}
			if region, ok := items[j].Labels["region"]; ok {
				right += "\x00" + region
			}
			if pollenType, ok := items[i].Labels["pollen_type"]; ok {
				left += "\x00" + pollenType
			}
			if pollenType, ok := items[j].Labels["pollen_type"]; ok {
				right += "\x00" + pollenType
			}
			return left < right
		})

		fmt.Fprintf(&b, "# HELP %s %s\n", metricName, metricHelp[metricName])
		fmt.Fprintf(&b, "# TYPE %s gauge\n", metricName)
		for _, item := range items {
			// Build label string dynamically based on available labels
			var labelPairs []string

			// Always include provider and city if they exist
			if provider, ok := item.Labels["provider"]; ok {
				labelPairs = append(labelPairs, fmt.Sprintf("provider=\"%s\"", escapeLabel(provider)))
			}
			if city, ok := item.Labels["city"]; ok {
				labelPairs = append(labelPairs, fmt.Sprintf("city=\"%s\"", escapeLabel(city)))
			}

			// Add pollen-specific labels if present
			if region, ok := item.Labels["region"]; ok {
				labelPairs = append(labelPairs, fmt.Sprintf("region=\"%s\"", escapeLabel(region)))
			}
			if pollenType, ok := item.Labels["pollen_type"]; ok {
				labelPairs = append(labelPairs, fmt.Sprintf("pollen_type=\"%s\"", escapeLabel(pollenType)))
			}

			labelsStr := strings.Join(labelPairs, ",")
			if labelsStr != "" {
				fmt.Fprintf(&b, "%s{%s} %s\n",
					metricName,
					labelsStr,
					strconv.FormatFloat(item.Value, 'f', -1, 64),
				)
			} else {
				fmt.Fprintf(&b, "%s %s\n",
					metricName,
					strconv.FormatFloat(item.Value, 'f', -1, 64),
				)
			}
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
