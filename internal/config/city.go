package config

import (
	"fmt"
	"strconv"
	"strings"
)

type City struct {
	Name      string
	Lat       float64
	Lon       float64
	Providers []string // Optional provider filter, nil/empty = all providers
}

type CityFlags []City

func (c *CityFlags) String() string {
	parts := make([]string, 0, len(*c))
	for _, city := range *c {
		s := fmt.Sprintf("%s:%f,%f", city.Name, city.Lat, city.Lon)
		if len(city.Providers) > 0 {
			s += "@" + strings.Join(city.Providers, ",")
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ";")
}

func (c *CityFlags) Set(value string) error {
	city, err := ParseCity(value)
	if err != nil {
		return err
	}
	*c = append(*c, city)
	return nil
}

func ParseCity(value string) (City, error) {
	// Split on @ to separate location from optional providers
	atParts := strings.SplitN(value, "@", 2)
	locationPart := atParts[0]
	var providers []string

	if len(atParts) == 2 {
		// Parse provider list
		providerPart := strings.TrimSpace(atParts[1])
		if providerPart == "" {
			return City{}, fmt.Errorf("invalid city format %q, provider list cannot be empty after @", value)
		}
		for _, p := range strings.Split(providerPart, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				return City{}, fmt.Errorf("invalid city format %q, provider names cannot be empty", value)
			}
			providers = append(providers, p)
		}
	}

	// Parse Name:lat,lon part
	parts := strings.SplitN(locationPart, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return City{}, fmt.Errorf("invalid city format %q, expected Name:lat,lon[@provider1,provider2]", value)
	}

	coords := strings.SplitN(parts[1], ",", 2)
	if len(coords) != 2 {
		return City{}, fmt.Errorf("invalid coordinates format %q, expected lat,lon", parts[1])
	}

	lat, err := strconv.ParseFloat(strings.TrimSpace(coords[0]), 64)
	if err != nil {
		return City{}, fmt.Errorf("invalid latitude %q: %w", coords[0], err)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(coords[1]), 64)
	if err != nil {
		return City{}, fmt.Errorf("invalid longitude %q: %w", coords[1], err)
	}

	return City{
		Name:      strings.TrimSpace(parts[0]),
		Lat:       lat,
		Lon:       lon,
		Providers: providers,
	}, nil
}
