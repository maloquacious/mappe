// Mappe - consolidated map and world generators
// Copyright (c) 2026 Michael D Henderson
//
// Portions of this file were adapted from github.com/mdhender/wgva,
// Copyright (c) 2026 Michael D Henderson, under the MIT License.
// See docs/licenses/wgva-MIT.txt.

package domains

// ElevationBand is one of six ordered normalized elevation classifications.
type ElevationBand uint8

// Elevation bands have stable IDs because they are serialized domain values.
const (
	ElevationDeepWater    ElevationBand = 0
	ElevationShallowWater ElevationBand = 1
	ElevationLowland      ElevationBand = 2
	ElevationUpland       ElevationBand = 3
	ElevationHighland     ElevationBand = 4
	ElevationMountain     ElevationBand = 5
)

// ElevationBands returns all declared elevation bands from lowest to highest.
func ElevationBands() []ElevationBand {
	return []ElevationBand{
		ElevationDeepWater,
		ElevationShallowWater,
		ElevationLowland,
		ElevationUpland,
		ElevationHighland,
		ElevationMountain,
	}
}

// String returns the stable name of the elevation band.
func (e ElevationBand) String() string {
	switch e {
	case ElevationDeepWater:
		return "deep-water"
	case ElevationShallowWater:
		return "shallow-water"
	case ElevationLowland:
		return "lowland"
	case ElevationUpland:
		return "upland"
	case ElevationHighland:
		return "highland"
	case ElevationMountain:
		return "mountain"
	default:
		return "unknown"
	}
}

// Valid reports whether e is a declared elevation band.
func (e ElevationBand) Valid() bool { return e <= ElevationMountain }

// IsWater reports whether e is an ocean-water elevation band.
func (e ElevationBand) IsWater() bool {
	return e == ElevationDeepWater || e == ElevationShallowWater
}
