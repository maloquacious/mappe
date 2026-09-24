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
	"fmt"
	"math"
	"math/rand/v2"
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

// Map is an equirectangular height map.
type Map struct {
	width      int
	height     int
	elevations []int
}

// Width returns the number of columns in the map.
func (m *Map) Width() int { return m.width }

// Height returns the number of rows in the map.
func (m *Map) Height() int { return m.height }

// Elevation returns the generated elevation at (x, y). It panics when either
// coordinate is outside the map.
func (m *Map) Elevation(x, y int) int {
	if x < 0 || x >= m.width || y < 0 || y >= m.height {
		panic("olsson: coordinates outside map")
	}
	return m.elevations[y*m.width+x]
}

// Elevations returns the map's elevations in row-major order. The returned
// slice is a copy and may be modified by the caller.
func (m *Map) Elevations() []int {
	elevations := make([]int, len(m.elevations))
	copy(elevations, m.elevations)
	return elevations
}

// Generate creates a deterministic height map from cfg.
func Generate(cfg Config) (*Map, error) {
	if cfg.Source == nil {
		return nil, fmt.Errorf("olsson: source must not be nil")
	}
	if cfg.Height < 1 {
		return nil, fmt.Errorf("olsson: height must be positive")
	}
	if cfg.Width != 2*cfg.Height {
		return nil, fmt.Errorf("olsson: width must be twice height")
	}
	if cfg.Faults < 0 {
		return nil, fmt.Errorf("olsson: faults must not be negative")
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

	return &Map{
		width:      cfg.Width,
		height:     cfg.Height,
		elevations: elevations,
	}, nil
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
