package provider

import "testing"

func TestOpenWeatherInitRequiresAPIKey(t *testing.T) {
	t.Parallel()

	p := NewOpenWeatherProvider()
	if err := p.Init(""); err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestOpenWeatherInitWithAPIKey(t *testing.T) {
	t.Parallel()

	p := NewOpenWeatherProvider()
	if err := p.Init("test-key"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
