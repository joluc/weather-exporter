package main

import (
	"io"
	"log/slog"
	"testing"
)

func TestEnabledProvidersWithDWDPollenFlag(t *testing.T) {
	t.Setenv("USER_AGENT", "weather-exporter-test")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Test with dwd-pollen enabled
	providers := enabledProviders(logger, false, true, "", "")
	if !hasProvider(providers, "dwd-pollen") {
		t.Fatal("expected dwd-pollen provider when dwd-pollen-enabled is true")
	}
	if hasProvider(providers, "dwd") {
		t.Fatal("did not expect dwd provider when dwd-enabled is false")
	}
}

func TestEnabledProvidersWithBothDWDProviders(t *testing.T) {
	t.Setenv("USER_AGENT", "weather-exporter-test")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Test with both dwd and dwd-pollen enabled
	providers := enabledProviders(logger, true, true, "", "")
	if !hasProvider(providers, "dwd") {
		t.Fatal("expected dwd provider when dwd-enabled is true")
	}
	if !hasProvider(providers, "dwd-pollen") {
		t.Fatal("expected dwd-pollen provider when dwd-pollen-enabled is true")
	}
}
