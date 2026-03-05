# weather-exporter

Prometheus exporter for weather data with dynamic provider activation.

## Features

- Multi-provider collection with a unified metric schema.
- Dynamic provider activation based on flags and credentials.
- Per-city provider selection to optimize API usage.
- Built-in caching (default 5m) to reduce provider API calls while allowing frequent Prometheus scrapes.
- Repeatable `--city` flag for multiple target locations.
- Concurrent fetches across providers and cities.
- Endpoints:
  - `/metrics`
  - `/health`

## Providers

- `yr` (MET Norway): enabled by default, optionally uses `--yr-user-agent` flag or `USER_AGENT` env var.
- `dwd` (Open-Meteo backend): enabled with `--dwd-enabled` flag.
- `dwd-pollen` (DWD Pollen Forecast): enabled with `--dwd-pollen-enabled` flag. Provides pollen risk indices for 8 allergen types across 26 German regions. Automatically maps lat/lon coordinates to DWD regions.
- `openweathermap`: enabled with `--openweathermap-api-key` flag or `OPENWEATHER_API_KEY` env var.

## Metrics

### Weather Metrics

All weather metrics are gauges with labels `provider` and `city`.

- `weather_temperature_celsius`
- `weather_humidity_relative`
- `weather_pressure_hpa`
- `weather_wind_speed_mps`
- `weather_wind_direction_degrees`
- `weather_precipitation_mm`
- `weather_cloud_cover_percent`
- `weather_visibility_meters`
- `weather_provider_up` (1 when fetch succeeds, 0 on failure)
- `weather_cache_age_seconds` (age of cached data, 0 for fresh data)

### Pollen Metrics

Pollen metrics have labels `provider`, `city`, `region`, and `pollen_type`.

- `pollen_risk_index`: Pollen risk index (0=none, 0.5=low-medium, 1=low, 1.5=low-medium, 2=medium, 2.5=medium-high, 3=high, -1=no data)
  - Pollen types: `hazel`, `alder`, `ash`, `birch`, `grasses`, `rye`, `mugwort`, `ambrosia`

## Run locally

Basic usage with all enabled providers:

```bash
go run ./cmd/weather-exporter \
  --city="Leipzig:51.33,12.37" \
  --city="Copenhagen:55.67,12.56"
```

### Per-city provider selection

Specify which providers to use for each city with the `@provider1,provider2` suffix:

```bash
go run ./cmd/weather-exporter \
  --dwd-enabled \
  --city="Leipzig:51.33,12.37@yr,dwd" \
  --city="Copenhagen:55.67,12.56@yr"
```

This fetches Leipzig data from both `yr` and `dwd`, while Copenhagen only uses `yr`.

**Backward compatibility**: Cities without the `@` suffix use all enabled providers (same behavior as before).

### Provider configuration

Enable OpenWeatherMap with flag:

```bash
go run ./cmd/weather-exporter \
  --openweathermap-api-key="your-key" \
  --city="Leipzig:51.33,12.37"
```

Or use environment variable (backward compatible):

```bash
export OPENWEATHER_API_KEY="your-key"
go run ./cmd/weather-exporter --city="Leipzig:51.33,12.37"
```

Enable all providers:

```bash
go run ./cmd/weather-exporter \
  --dwd-enabled \
  --dwd-pollen-enabled \
  --openweathermap-api-key="your-key" \
  --yr-user-agent="weather-exporter/1.0" \
  --city="Leipzig:51.33,12.37"
```

Mix and match providers per city:

```bash
go run ./cmd/weather-exporter \
  --dwd-enabled \
  --dwd-pollen-enabled \
  --openweathermap-api-key="your-key" \
  --city="Leipzig:51.33,12.37@yr,dwd,dwd-pollen" \
  --city="Berlin:52.52,13.40@openweathermap,dwd-pollen"
```

Enable pollen data only for German cities:

```bash
go run ./cmd/weather-exporter \
  --dwd-pollen-enabled \
  --city="Leipzig:51.33,12.37@dwd-pollen" \
  --city="Berlin:52.52,13.40@dwd-pollen" \
  --city="Munich:48.13,11.57@dwd-pollen"
```

## Test

```bash
go test ./...
```

## Kubernetes Deployment

Deploy using Helm:

```bash
helm install weather-exporter ./charts/weather-exporter \
  --set config.cities[0]="Leipzig:51.33,12.37@yr,dwd" \
  --set config.providers.dwd.enabled=true \
  --set config.providers.openweathermap.enabled=true \
  --set config.providers.openweathermap.apiKey="your-key"
```

Or with a values file:

```yaml
# values.yaml
config:
  cities:
    - "Leipzig:51.33,12.37@yr,dwd"
    - "Copenhagen:55.67,12.56@openweathermap"
  providers:
    dwd:
      enabled: true
    openweathermap:
      enabled: true
      apiKey: "your-key"
```

```bash
helm install weather-exporter ./charts/weather-exporter -f values.yaml
```

See [charts/weather-exporter/README.md](charts/weather-exporter/README.md) for full documentation.

## Troubleshooting

### Provider not available error

If you see an error like:

```
city "Leipzig" requests provider "dwd" which is not available (available: yr)
```

This means you specified a provider for a city that isn't enabled. Solutions:
- Enable the provider: use `--dwd-enabled` flag for DWD, or `--openweathermap-api-key=KEY` for OpenWeatherMap
- Remove the provider from the city's `@` list
- Remove the `@` suffix entirely to use all enabled providers

