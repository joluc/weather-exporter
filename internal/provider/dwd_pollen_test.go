package provider

import "testing"

func TestGetDWDRegionByCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		lat       float64
		lon       float64
		wantFound bool
	}{
		{
			name:      "Leipzig",
			lat:       51.33,
			lon:       12.37,
			wantFound: true,
		},
		{
			name:      "Berlin",
			lat:       52.52,
			lon:       13.40,
			wantFound: true,
		},
		{
			name:      "Hamburg",
			lat:       53.55,
			lon:       10.00,
			wantFound: true,
		},
		{
			name:      "Munich",
			lat:       48.13,
			lon:       11.57,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region, found := GetDWDRegionByCoordinates(tt.lat, tt.lon)
			if found != tt.wantFound {
				t.Errorf("GetDWDRegionByCoordinates() found = %v, want %v", found, tt.wantFound)
				return
			}
			if found {
				// Just verify that we got a valid region
				if region.ID == 0 || region.Name == "" {
					t.Errorf("GetDWDRegionByCoordinates() returned invalid region: ID=%d, Name=%s", region.ID, region.Name)
				}
				t.Logf("Coordinates %.2f,%.2f mapped to region %d (%s)", tt.lat, tt.lon, region.ID, region.Name)
			}
		})
	}
}

func TestMapPollenValue(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0", 0.0},
		{"0-1", 0.5},
		{"1", 1.0},
		{"1-2", 1.5},
		{"2", 2.0},
		{"2-3", 2.5},
		{"3", 3.0},
		{"-1", -1.0},
		{"invalid", -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapPollenValue(tt.input)
			if got != tt.want {
				t.Errorf("mapPollenValue(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
