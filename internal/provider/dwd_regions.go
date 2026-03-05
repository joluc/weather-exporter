package provider

import "math"

// DWDRegion represents a DWD pollen forecast region
type DWDRegion struct {
	ID   int
	Name string
}

// GetDWDRegionByCoordinates returns the DWD region ID for given coordinates
// Based on approximate boundaries of the 26 DWD pollen forecast regions
func GetDWDRegionByCoordinates(lat, lon float64) (DWDRegion, bool) {
	// DWD divides Germany into 26 regions for pollen forecasting
	// This mapping uses approximate geographic boundaries

	// Helper function to calculate distance
	distance := func(lat1, lon1, lat2, lon2 float64) float64 {
		return math.Sqrt(math.Pow(lat1-lat2, 2) + math.Pow(lon1-lon2, 2))
	}

	// Region center points (approximate)
	regions := []struct {
		id     int
		name   string
		lat    float64
		lon    float64
		weight float64 // Used for boundary regions
	}{
		// Schleswig-Holstein und Hamburg
		{11, "Inseln und Marschen", 54.5, 8.5, 1.0},
		{12, "Geest,Schleswig-Holstein und Hamburg", 53.8, 10.0, 1.0},

		// Mecklenburg-Vorpommern
		{20, "Mecklenburg-Vorpommern", 53.5, 12.5, 1.0},

		// Niedersachsen und Bremen
		{31, "Westl. Niedersachsen/Bremen", 53.0, 8.5, 1.0},
		{32, "Östl. Niedersachsen", 52.5, 11.0, 1.0},

		// Nordrhein-Westfalen
		{41, "Rhein.-Westfäl. Tiefland", 51.5, 7.0, 1.0},
		{42, "Ostwestfalen", 51.8, 8.8, 1.0},
		{43, "Mittelgebirge NRW", 51.0, 7.5, 1.0},

		// Sachsen-Anhalt
		{61, "Tiefland Sachsen-Anhalt", 52.2, 11.8, 1.0},
		{62, "Harz", 51.7, 10.8, 1.0},

		// Thüringen
		{71, "Tiefland Thüringen", 51.0, 11.5, 1.0},
		{72, "Mittelgebirge Thüringen", 50.5, 11.0, 1.0},

		// Sachsen
		{81, "Tiefland Sachsen", 51.3, 13.5, 1.0},
		{82, "Mittelgebirge Sachsen", 50.5, 13.0, 1.0},

		// Hessen
		{91, "Nordhessen und hess. Mittelgebirge", 51.0, 9.5, 1.0},
		{92, "Rhein-Main", 50.0, 8.7, 1.0},

		// Rheinland-Pfalz und Saarland
		{101, "Rhein, Pfalz, Nahe und Mosel", 49.8, 7.8, 1.0},
		{102, "Mittelgebirgsbereich Rheinland-Pfalz", 49.5, 7.0, 1.0},
		{103, "Saarland", 49.4, 7.0, 1.0},

		// Baden-Württemberg
		{111, "Oberrhein und unteres Neckartal", 49.0, 8.4, 1.0},
		{112, "Hohenlohe/mittlerer Neckar/Oberschwaben", 48.5, 9.5, 1.0},
		{113, "Mittelgebirge Baden-Württemberg", 48.0, 8.3, 1.0},

		// Bayern
		{121, "Allgäu/Oberbayern/Bay. Wald", 47.8, 11.5, 1.0},
		{122, "Donauniederungen", 48.5, 12.0, 1.0},
		{123, "Bayern nördl. der Donau, o. Bayr. Wald, o. Mainfranken", 49.2, 11.5, 1.0},
		{124, "Mainfranken", 49.8, 10.0, 1.0},
	}

	// Find the nearest region
	var closestRegion *DWDRegion
	minDistance := math.MaxFloat64

	for _, r := range regions {
		d := distance(lat, lon, r.lat, r.lon) * r.weight
		if d < minDistance {
			minDistance = d
			closestRegion = &DWDRegion{
				ID:   r.id,
				Name: r.name,
			}
		}
	}

	if closestRegion == nil {
		return DWDRegion{}, false
	}

	return *closestRegion, true
}
