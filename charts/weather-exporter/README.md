# Weather Exporter Helm Chart

Deploys weather-exporter to Kubernetes.

## Installation

```bash
helm install weather-exporter ./helm/weather-exporter \
  --set config.cities[0]="Leipzig:51.33,12.37"
```

## Configuration

### Basic Example

```yaml
config:
  cities:
    - "Leipzig:51.33,12.37"
    - "Copenhagen:55.67,12.56"
```

### Per-City Providers

```yaml
config:
  cities:
    - "Leipzig:51.33,12.37@yr,dwd"
    - "Copenhagen:55.67,12.56@openweathermap"

  providers:
    dwd:
      enabled: true
    openweathermap:
      enabled: true
      apiKey: "your-api-key"
```

### Using Existing Secret

```bash
kubectl create secret generic my-secret \
  --from-literal=openweathermap-api-key="your-key"
```

```yaml
existingSecret: "my-secret"
config:
  providers:
    openweathermap:
      enabled: true
```

## Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.cities` | List of cities | `["Leipzig:51.33,12.37"]` |
| `config.providers.dwd.enabled` | Enable DWD | `false` |
| `config.providers.openweathermap.enabled` | Enable OpenWeatherMap | `false` |
| `config.providers.openweathermap.apiKey` | API key | `""` |
| `existingSecret` | Use existing secret | `""` |
| `resources.limits.cpu` | CPU limit | `200m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
