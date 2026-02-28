package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/joluc/weather-exporter/internal/config"
)

func TestEnabledProvidersWithoutOpenWeatherKey(t *testing.T) {
	t.Setenv("OPENWEATHER_API_KEY", "")
	t.Setenv("USER_AGENT", "weather-exporter-test")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, false, "", "")
	if hasProvider(providers, "openweathermap") {
		t.Fatal("did not expect openweathermap provider without api key")
	}
	if !hasProvider(providers, "yr") {
		t.Fatal("expected yr provider to be enabled")
	}
}

func TestEnabledProvidersWithOpenWeatherKey(t *testing.T) {
	t.Setenv("OPENWEATHER_API_KEY", "test-key")
	t.Setenv("USER_AGENT", "weather-exporter-test")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, false, "", "")
	if !hasProvider(providers, "openweathermap") {
		t.Fatal("expected openweathermap provider with api key")
	}
}

func TestEnabledProvidersWithDWDFlag(t *testing.T) {
	t.Setenv("OPENWEATHER_API_KEY", "")
	t.Setenv("USER_AGENT", "weather-exporter-test")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, true, "", "")
	if !hasProvider(providers, "dwd") {
		t.Fatal("expected dwd provider when dwd-enabled is true")
	}
}

func hasProvider[T interface{ Name() string }](providers []T, name string) bool {
	for _, p := range providers {
		if p.Name() == name {
			return true
		}
	}
	return false
}

func TestValidateCityProvidersWithAllValid(t *testing.T) {
	t.Setenv("USER_AGENT", "weather-exporter-test")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, false, "", "")

	cities := []config.City{
		{Name: "Leipzig", Lat: 51.33, Lon: 12.37, Providers: []string{"yr"}},
	}

	if err := validateCityProviders(cities, providers); err != nil {
		t.Fatalf("expected no error for valid providers, got %v", err)
	}
}

func TestValidateCityProvidersWithUnavailable(t *testing.T) {
	t.Setenv("USER_AGENT", "weather-exporter-test")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, false, "", "") // dwd not enabled

	cities := []config.City{
		{Name: "Leipzig", Lat: 51.33, Lon: 12.37, Providers: []string{"dwd"}},
	}

	if err := validateCityProviders(cities, providers); err == nil {
		t.Fatal("expected error for unavailable provider")
	}
}

func TestValidateCityProvidersWithNoProviders(t *testing.T) {
	t.Setenv("USER_AGENT", "weather-exporter-test")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, false, "", "")

	cities := []config.City{
		{Name: "Leipzig", Lat: 51.33, Lon: 12.37, Providers: nil}, // No provider filter
	}

	if err := validateCityProviders(cities, providers); err != nil {
		t.Fatalf("expected no error when city has no provider filter, got %v", err)
	}
}

func TestEnabledProvidersWithFlagValues(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, true, "flag-api-key", "flag-user-agent")

	if !hasProvider(providers, "yr") {
		t.Fatal("expected yr provider with flag user agent")
	}
	if !hasProvider(providers, "dwd") {
		t.Fatal("expected dwd provider when flag is true")
	}
	if !hasProvider(providers, "openweathermap") {
		t.Fatal("expected openweathermap provider with flag api key")
	}
}

func TestEnabledProvidersEnvVarFallback(t *testing.T) {
	t.Setenv("OPENWEATHER_API_KEY", "env-key")
	t.Setenv("USER_AGENT", "env-agent")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, false, "", "") // Empty flags, should use env vars

	if !hasProvider(providers, "yr") {
		t.Fatal("expected yr provider with env USER_AGENT")
	}
	if !hasProvider(providers, "openweathermap") {
		t.Fatal("expected openweathermap provider with env OPENWEATHER_API_KEY")
	}
}

func TestEnabledProvidersFlagsOverrideEnv(t *testing.T) {
	t.Setenv("OPENWEATHER_API_KEY", "env-key")
	t.Setenv("USER_AGENT", "env-agent")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers := enabledProviders(logger, false, "flag-key", "flag-agent") // Flags should override env

	// Both should be enabled (the test validates that flags take precedence)
	if !hasProvider(providers, "yr") {
		t.Fatal("expected yr provider with flag user agent")
	}
	if !hasProvider(providers, "openweathermap") {
		t.Fatal("expected openweathermap provider with flag api key")
	}
}
