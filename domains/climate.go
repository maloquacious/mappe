// Mappe - consolidated map and world generators
// Copyright (c) 2026 Michael D Henderson
//
// Portions of this file were adapted from github.com/mdhender/wgva,
// Copyright (c) 2026 Michael D Henderson, under the MIT License.
// See docs/licenses/wgva-MIT.txt.

package domains

// HeatBand is one of five ordered temperature classifications.
type HeatBand uint8

// Heat bands have stable IDs because they are serialized domain values.
const (
	HeatPolar     HeatBand = 0
	HeatCold      HeatBand = 1
	HeatTemperate HeatBand = 2
	HeatWarm      HeatBand = 3
	HeatHot       HeatBand = 4
)

// HeatBands returns all declared heat bands from coldest to hottest.
func HeatBands() []HeatBand {
	return []HeatBand{HeatPolar, HeatCold, HeatTemperate, HeatWarm, HeatHot}
}

// String returns the stable name of the heat band.
func (h HeatBand) String() string {
	switch h {
	case HeatPolar:
		return "polar"
	case HeatCold:
		return "cold"
	case HeatTemperate:
		return "temperate"
	case HeatWarm:
		return "warm"
	case HeatHot:
		return "hot"
	default:
		return "unknown"
	}
}

// Valid reports whether h is a declared heat band.
func (h HeatBand) Valid() bool { return h <= HeatHot }

// MoistureBand is one of five ordered moisture classifications.
type MoistureBand uint8

// Moisture bands have stable IDs because they are serialized domain values.
const (
	MoistureArid      MoistureBand = 0
	MoistureDry       MoistureBand = 1
	MoistureModerate  MoistureBand = 2
	MoistureHumid     MoistureBand = 3
	MoistureSaturated MoistureBand = 4
)

// MoistureBands returns all declared moisture bands from driest to wettest.
func MoistureBands() []MoistureBand {
	return []MoistureBand{MoistureArid, MoistureDry, MoistureModerate, MoistureHumid, MoistureSaturated}
}

// String returns the stable name of the moisture band.
func (m MoistureBand) String() string {
	switch m {
	case MoistureArid:
		return "arid"
	case MoistureDry:
		return "dry"
	case MoistureModerate:
		return "moderate"
	case MoistureHumid:
		return "humid"
	case MoistureSaturated:
		return "saturated"
	default:
		return "unknown"
	}
}

// Valid reports whether m is a declared moisture band.
func (m MoistureBand) Valid() bool { return m <= MoistureSaturated }

// Climate is a pair of independent heat and moisture bands.
type Climate struct {
	Heat     HeatBand
	Moisture MoistureBand
}

// String returns the stable heat/moisture name of the climate.
func (c Climate) String() string { return c.Heat.String() + "/" + c.Moisture.String() }

// Valid reports whether both climate axes are declared bands.
func (c Climate) Valid() bool { return c.Heat.Valid() && c.Moisture.Valid() }

// HeatThresholds divide a normalized heat scalar into five bands. Each field
// is the inclusive upper boundary of the named band.
type HeatThresholds struct {
	Polar     float64
	Cold      float64
	Temperate float64
	Warm      float64
}

// Classify returns the band containing normalized heat.
func (t HeatThresholds) Classify(heat float64) HeatBand {
	switch {
	case heat <= t.Polar:
		return HeatPolar
	case heat <= t.Cold:
		return HeatCold
	case heat <= t.Temperate:
		return HeatTemperate
	case heat <= t.Warm:
		return HeatWarm
	default:
		return HeatHot
	}
}

// MoistureThresholds divide a normalized moisture scalar into five bands.
// Each field is the inclusive upper boundary of the named band.
type MoistureThresholds struct {
	Arid     float64
	Dry      float64
	Moderate float64
	Humid    float64
}

// Classify returns the band containing normalized moisture.
func (t MoistureThresholds) Classify(moisture float64) MoistureBand {
	switch {
	case moisture <= t.Arid:
		return MoistureArid
	case moisture <= t.Dry:
		return MoistureDry
	case moisture <= t.Moderate:
		return MoistureModerate
	case moisture <= t.Humid:
		return MoistureHumid
	default:
		return MoistureSaturated
	}
}
