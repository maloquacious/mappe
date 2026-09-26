// Mappe - consolidated map and world generators
// Copyright (c) 2022-2026 Michael D Henderson
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

// Package olsson generates equirectangular world height maps using John
// Olsson's spherical great-circle fault algorithm.
//
// This implementation was migrated from github.com/mdhender/worldgen, which
// preserved and later ported John Olsson's original C generator. It retains
// the source generator's map construction while replacing its process-global
// state, legacy pseudorandom source, and file output with an explicit library
// API.
package olsson

import (
	"math"
	"math/rand/v2"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/cerrs"
)

// Configuration errors returned by GenerateNormalizedHeightMap.
const (
	ErrNilSource     cerrs.Error = "olsson: source must not be nil"
	ErrInvalidHeight cerrs.Error = "olsson: height must be positive"
	ErrInvalidWidth  cerrs.Error = "olsson: width must be twice height"
	ErrInvalidFaults cerrs.Error = "olsson: faults must not be negative"
)

// Config controls generation. Source must not be nil and is advanced during
// generation. Width must be twice Height, matching the equirectangular layout
// of the source generator.
type Config struct {
	Source rand.Source
	Width  int
	Height int
	Faults int
}

// GenerateNormalizedHeightMap creates a deterministic normalized height map
// from cfg. Non-flat maps span the full [0, 1] elevation range. Flat maps
// contain only zero elevations.
func GenerateNormalizedHeightMap(cfg Config) (*domains.NormalizedHeightMap, error) {
	elevations, err := generateRaw(cfg)
	if err != nil {
		return nil, err
	}

	return domains.NewNormalizedHeightMap(cfg.Width, cfg.Height, normalize(elevations))
}

// GenerateHeightField creates an equirectangular height field. Longitude wraps
// east-west; the north and south edges are distinct polar boundaries.
func GenerateHeightField(cfg Config) (*domains.HeightField, error) {
	heightMap, err := GenerateNormalizedHeightMap(cfg)
	if err != nil {
		return nil, err
	}
	return domains.NewHeightField(heightMap, domains.GridTopology{WrapEastWest: true})
}

func generateRaw(cfg Config) ([]int, error) {
	if cfg.Source == nil {
		return nil, ErrNilSource
	}
	if cfg.Height < 1 {
		return nil, ErrInvalidHeight
	}
	if cfg.Width != 2*cfg.Height {
		return nil, ErrInvalidWidth
	}
	if cfg.Faults < 0 {
		return nil, ErrInvalidFaults
	}

	rows := make([][]int, cfg.Height)
	elevations := make([]int, cfg.Width*cfg.Height)
	for y := range rows {
		rows[y] = elevations[y*cfg.Width : (y+1)*cfg.Width]
		for x := 1; x < cfg.Width; x++ {
			rows[y][x] = math.MinInt
		}
	}

	sinPhi := make([]float64, 2*cfg.Width)
	for x := 0; x < cfg.Width; x++ {
		sinPhi[x] = math.Sin(float64(x) * 2 * math.Pi / float64(cfg.Width))
		sinPhi[x+cfg.Width] = sinPhi[x]
	}

	rnd := rand.New(cfg.Source)
	for fault := 0; fault < cfg.Faults; fault++ {
		generateFault(rows, sinPhi, rnd, rnd.IntN(2) == 0)
	}

	// The source computes half the map and mirrors it around the seam.
	for y := range rows {
		for x := 1; x < cfg.Width/2; x++ {
			rows[y][cfg.Width-x] = rows[y][x]
		}
	}

	// Faults are stored as sparse deltas. Integrating each row reconstructs
	// the elevation at every point.
	for y := range rows {
		elevation := rows[y][0]
		for x := 1; x < cfg.Width; x++ {
			if rows[y][x] != math.MinInt {
				elevation += rows[y][x]
			}
			rows[y][x] = elevation
		}
	}

	return elevations, nil
}

func normalize(elevations []int) []float64 {
	normalized := make([]float64, len(elevations))
	minimum, maximum := elevations[0], elevations[0]
	for _, elevation := range elevations[1:] {
		if elevation < minimum {
			minimum = elevation
		}
		if elevation > maximum {
			maximum = elevation
		}
	}
	if minimum == maximum {
		return normalized
	}

	span := float64(maximum) - float64(minimum)
	for i, elevation := range elevations {
		normalized[i] = (float64(elevation) - float64(minimum)) / span
	}
	return normalized
}

func generateFault(rows [][]int, sinPhi []float64, rnd *rand.Rand, lower bool) {
	height, width := len(rows), len(rows[0])
	bump := 1
	if lower {
		bump = -1
	}

	alpha := (rnd.Float64() - 0.5) * math.Pi
	beta := (rnd.Float64() - 0.5) * math.Pi
	tanB := math.Tan(math.Acos(math.Cos(alpha) * math.Cos(beta)))
	xsi := int((float64(width)/2 - float64(width)/math.Pi) * beta)
	heightDiv2 := float64(height) / 2
	heightDivPI := float64(height) / math.Pi

	for row, phi := 0, 0; phi < width/2; row, phi = row+1, phi+1 {
		theta := int(heightDivPI*math.Atan(sinPhi[xsi-phi+width]*tanB) + heightDiv2)
		if rows[row][theta] == math.MinInt {
			rows[row][theta] = 0
		} else {
			rows[row][theta] += bump
		}
	}
}
