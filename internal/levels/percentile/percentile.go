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

// Package percentile derives elevation levels by checking tile percentages
// against a height field's elevation histogram. Sea level comes from the share
// of all tiles; land levels come from the share of land tiles; depth levels
// come from the share of ocean tiles. One pixel is one tile, and every tile
// counts equally.
package percentile

import (
	"math"
	"slices"
	"sort"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/cerrs"
)

// Configuration errors returned by Derive.
const (
	ErrNilHeightField       cerrs.Error = "percentile: height field must not be nil"
	ErrInvalidOceanPercent  cerrs.Error = "percentile: ocean percent must be in [0, 100]"
	ErrInvalidLandPercents  cerrs.Error = "percentile: upland, highland, and mountain percents must be strictly increasing in [0, 100]"
	ErrInvalidDepthPercents cerrs.Error = "percentile: shelf and abyss percents must be strictly increasing in [0, 100]"
)

// Config holds whole percentages of tiles.
type Config struct {
	// OceanPercent is the share of all tiles at or below sea level.
	OceanPercent int
	// UplandPercent, HighlandPercent, and MountainPercent are the shares of
	// land tiles below the upland, highland, and mountain levels.
	UplandPercent   int
	HighlandPercent int
	MountainPercent int
	// ShelfPercent and AbyssPercent are the shares of ocean tiles above the
	// shelf and abyss levels.
	ShelfPercent int
	AbyssPercent int
}

// DefaultConfig returns 48 percent ocean. Land is 50 percent lowland, 30
// percent upland, 13 percent highland, and 7 percent mountain. Ocean is 15
// percent shallow sea, 45 percent ocean, and 40 percent deep ocean.
func DefaultConfig() Config {
	return Config{
		OceanPercent:    48,
		UplandPercent:   50,
		HighlandPercent: 80,
		MountainPercent: 93,
		ShelfPercent:    15,
		AbyssPercent:    60,
	}
}

// Derive returns the elevation levels for field.
//
// Every level uses the same tie rule: the requested share is a minimum. When
// tiles share an elevation, all of them fall on the same side of a level, so
// the achieved share may exceed the request and a neighboring band may be
// smaller or empty. ElevationLevels.Composition reports the achieved shares.
//
// Normalized height maps usually contain a tile at elevation 0, and no level
// can lie below 0, so a 0 percent ocean still leaves those tiles as water.
func Derive(field *domains.HeightField, cfg Config) (*domains.ElevationLevels, error) {
	if field == nil {
		return nil, ErrNilHeightField
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}

	elevations := field.Elevations()
	slices.Sort(elevations)
	seaLevel := topOfLowest(elevations, cfg.OceanPercent)
	firstLand := sort.Search(len(elevations), func(i int) bool { return elevations[i] > seaLevel })
	ocean, land := elevations[:firstLand], elevations[firstLand:]

	return domains.NewElevationLevels(domains.ElevationLevelValues{
		Abyss:    belowShallowest(ocean, cfg.AbyssPercent, seaLevel),
		Shelf:    belowShallowest(ocean, cfg.ShelfPercent, seaLevel),
		SeaLevel: seaLevel,
		Upland:   aboveLowest(land, cfg.UplandPercent, seaLevel),
		Highland: aboveLowest(land, cfg.HighlandPercent, seaLevel),
		Mountain: aboveLowest(land, cfg.MountainPercent, seaLevel),
	})
}

func validate(cfg Config) error {
	if cfg.OceanPercent < 0 || cfg.OceanPercent > 100 {
		return ErrInvalidOceanPercent
	}
	if !strictlyIncreasingPercents(cfg.UplandPercent, cfg.HighlandPercent, cfg.MountainPercent) {
		return ErrInvalidLandPercents
	}
	if !strictlyIncreasingPercents(cfg.ShelfPercent, cfg.AbyssPercent) {
		return ErrInvalidDepthPercents
	}
	return nil
}

func strictlyIncreasingPercents(percents ...int) bool {
	for i, percent := range percents {
		if percent < 0 || percent > 100 || i > 0 && percent <= percents[i-1] {
			return false
		}
	}
	return true
}

// share returns the smallest tile count that is at least percent of n.
func share(n, percent int) int {
	return (n*percent + 99) / 100
}

// topOfLowest returns the lowest level at or below which at least percent of
// the ascending, non-empty elevations lie.
func topOfLowest(ascending []float64, percent int) float64 {
	k := share(len(ascending), percent)
	if k == 0 {
		return max(0, math.Nextafter(ascending[0], math.Inf(-1)))
	}
	return ascending[k-1]
}

// aboveLowest returns the lowest level below which at least percent of the
// ascending land elevations lie. The level is the elevation of the lowest tile
// left above it. Without land, every land level equals sea level.
func aboveLowest(ascending []float64, percent int, seaLevel float64) float64 {
	if len(ascending) == 0 {
		return seaLevel
	}
	k := share(len(ascending), percent)
	if k == 0 {
		return ascending[0]
	}
	below := ascending[k-1]
	next := sort.Search(len(ascending), func(i int) bool { return ascending[i] > below })
	if next < len(ascending) {
		return ascending[next]
	}
	return min(1, math.Nextafter(below, math.Inf(1)))
}

// belowShallowest returns the highest level above which at least percent of
// the ascending ocean elevations lie. The level is the elevation of the
// shallowest tile left at or below it. Without ocean, every depth level equals
// sea level.
func belowShallowest(ascending []float64, percent int, seaLevel float64) float64 {
	if len(ascending) == 0 {
		return seaLevel
	}
	k := share(len(ascending), percent)
	if k == 0 {
		return ascending[len(ascending)-1]
	}
	above := ascending[len(ascending)-k]
	next, _ := slices.BinarySearch(ascending, above)
	if next > 0 {
		return ascending[next-1]
	}
	return max(0, math.Nextafter(above, math.Inf(-1)))
}
