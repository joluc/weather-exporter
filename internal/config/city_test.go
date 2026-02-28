package config

import "testing"

func TestParseCity(t *testing.T) {
	t.Parallel()

	city, err := ParseCity("Leipzig:51.33,12.37")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if city.Name != "Leipzig" {
		t.Fatalf("expected city name Leipzig, got %s", city.Name)
	}
	if city.Lat != 51.33 {
		t.Fatalf("expected latitude 51.33, got %v", city.Lat)
	}
	if city.Lon != 12.37 {
		t.Fatalf("expected longitude 12.37, got %v", city.Lon)
	}
	if len(city.Providers) != 0 {
		t.Fatalf("expected no providers (backward compatibility), got %v", city.Providers)
	}
}

func TestParseCityWithSingleProvider(t *testing.T) {
	t.Parallel()

	city, err := ParseCity("Leipzig:51.33,12.37@yr")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if city.Name != "Leipzig" {
		t.Fatalf("expected city name Leipzig, got %s", city.Name)
	}
	if len(city.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(city.Providers))
	}
	if city.Providers[0] != "yr" {
		t.Fatalf("expected provider yr, got %s", city.Providers[0])
	}
}

func TestParseCityWithMultipleProviders(t *testing.T) {
	t.Parallel()

	city, err := ParseCity("Leipzig:51.33,12.37@yr,dwd,openweathermap")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(city.Providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(city.Providers))
	}
	if city.Providers[0] != "yr" || city.Providers[1] != "dwd" || city.Providers[2] != "openweathermap" {
		t.Fatalf("expected providers [yr, dwd, openweathermap], got %v", city.Providers)
	}
}

func TestParseCityWithProvidersWhitespace(t *testing.T) {
	t.Parallel()

	city, err := ParseCity("Leipzig:51.33,12.37@yr , dwd , openweathermap")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(city.Providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(city.Providers))
	}
	if city.Providers[0] != "yr" || city.Providers[1] != "dwd" || city.Providers[2] != "openweathermap" {
		t.Fatalf("expected providers [yr, dwd, openweathermap], got %v", city.Providers)
	}
}

func TestParseCityInvalid(t *testing.T) {
	t.Parallel()

	cases := []string{
		"Leipzig",
		":51.33,12.37",
		"Leipzig:51.33",
		"Leipzig:abc,12.37",
		"Leipzig:51.33,xyz",
		"Leipzig:51.33,12.37@",        // Empty provider list
		"Leipzig:51.33,12.37@yr,",     // Trailing comma
		"Leipzig:51.33,12.37@yr,,dwd", // Empty provider name
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseCity(tc); err == nil {
				t.Fatalf("expected error for %q", tc)
			}
		})
	}
}
