// Mappe - consolidated map and world generators
// Copyright (c) 2026 Michael D Henderson
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package domains

import "github.com/maloquacious/mappe/internal/cerrs"

// ErrInvalidElevationLevels reports elevation levels that are not finite,
// normalized, and non-decreasing.
const ErrInvalidElevationLevels cerrs.Error = "domains: elevation levels must be finite values in [0, 1] ordered from abyss to mountain"

// ElevationLevelValues are the absolute normalized elevations used to build
// ElevationLevels. Water levels are inclusive upper bounds: a tile at or below
// SeaLevel is water, at or below Shelf is below the continental shelf, and at
// or below Abyss is deep ocean. Land levels are inclusive lower bounds: a land
// tile at or above Upland is upland, and likewise for Highland and Mountain.
type ElevationLevelValues struct {
	Abyss    float64
	Shelf    float64
	SeaLevel float64
	Upland   float64
	Highland float64
	Mountain float64
}

// ElevationLevels is an immutable, validated set of absolute elevations that
// divide a height field into water depths and land elevation bands. It records
// only the elevations, not how they were derived, so any stage may produce it.
type ElevationLevels struct {
	values ElevationLevelValues
}

// NewElevationLevels validates values. Every level must be finite and in
// [0, 1], ordered Abyss <= Shelf <= SeaLevel <= Upland <= Highland <= Mountain.
// Equal adjacent levels are allowed and leave the band between them empty.
func NewElevationLevels(values ElevationLevelValues) (*ElevationLevels, error) {
	ordered := []float64{values.Abyss, values.Shelf, values.SeaLevel, values.Upland, values.Highland, values.Mountain}
	for i, value := range ordered {
		if !normalized(value) || i > 0 && value < ordered[i-1] {
			return nil, ErrInvalidElevationLevels
		}
	}
	return &ElevationLevels{values: values}, nil
}

// Values returns the absolute elevation of every level.
func (l *ElevationLevels) Values() ElevationLevelValues { return l.values }

// Classify returns the band containing elevation. Deep water lies at or below
// the shelf level and shallow water between the shelf and sea level.
func (l *ElevationLevels) Classify(elevation float64) ElevationBand {
	switch {
	case elevation <= l.values.Shelf:
		return ElevationDeepWater
	case elevation <= l.values.SeaLevel:
		return ElevationShallowWater
	case elevation < l.values.Upland:
		return ElevationLowland
	case elevation < l.values.Highland:
		return ElevationUpland
	case elevation < l.values.Mountain:
		return ElevationHighland
	default:
		return ElevationMountain
	}
}

// ElevationComposition counts the tiles of a height field in each depth zone
// and land band defined by ElevationLevels.
type ElevationComposition struct {
	Tiles      int
	DeepOcean  int
	Ocean      int
	ShallowSea int
	Lowland    int
	Upland     int
	Highland   int
	Mountain   int
}

// WaterTiles returns the number of tiles at or below sea level.
func (c ElevationComposition) WaterTiles() int { return c.DeepOcean + c.Ocean + c.ShallowSea }

// LandTiles returns the number of tiles above sea level.
func (c ElevationComposition) LandTiles() int { return c.Tiles - c.WaterTiles() }

// Composition measures how the levels divide field. It reports the achieved
// share of each zone and band, which histogram ties can move away from the
// share a stage requested.
func (l *ElevationLevels) Composition(field *HeightField) ElevationComposition {
	composition := ElevationComposition{}
	for _, elevation := range field.Elevations() {
		composition.Tiles++
		switch {
		case elevation <= l.values.Abyss:
			composition.DeepOcean++
		case elevation <= l.values.Shelf:
			composition.Ocean++
		case elevation <= l.values.SeaLevel:
			composition.ShallowSea++
		case elevation < l.values.Upland:
			composition.Lowland++
		case elevation < l.values.Highland:
			composition.Upland++
		case elevation < l.values.Mountain:
			composition.Highland++
		default:
			composition.Mountain++
		}
	}
	return composition
}
