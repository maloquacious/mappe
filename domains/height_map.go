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

// Package domains defines values shared by Mappe generators and consumers.
package domains

import (
	"math"

	"github.com/maloquacious/mappe/internal/cerrs"
)

// Construction errors returned by NewNormalizedHeightMap.
const (
	ErrInvalidWidth          cerrs.Error = "domains: height map width must be positive"
	ErrInvalidHeight         cerrs.Error = "domains: height map height must be positive"
	ErrInvalidElevationCount cerrs.Error = "domains: height map elevation count must equal width times height"
	ErrInvalidElevation      cerrs.Error = "domains: height map elevations must be finite values in [0, 1]"
)

// NormalizedHeightMap is a rectangular height map whose elevations are finite
// float64 values in the inclusive range [0, 1].
type NormalizedHeightMap struct {
	width      int
	height     int
	elevations []float64
}

// NewNormalizedHeightMap validates and copies elevations into a height map.
func NewNormalizedHeightMap(width, height int, elevations []float64) (*NormalizedHeightMap, error) {
	if width < 1 {
		return nil, ErrInvalidWidth
	}
	if height < 1 {
		return nil, ErrInvalidHeight
	}
	if len(elevations) != width*height {
		return nil, ErrInvalidElevationCount
	}
	for _, elevation := range elevations {
		if math.IsNaN(elevation) || math.IsInf(elevation, 0) || elevation < 0 || elevation > 1 {
			return nil, ErrInvalidElevation
		}
	}

	return &NormalizedHeightMap{
		width:      width,
		height:     height,
		elevations: append([]float64(nil), elevations...),
	}, nil
}

// Width returns the number of columns in the map.
func (m *NormalizedHeightMap) Width() int { return m.width }

// Height returns the number of rows in the map.
func (m *NormalizedHeightMap) Height() int { return m.height }

// Elevation returns the elevation at (x, y). It panics when either coordinate
// is outside the map.
func (m *NormalizedHeightMap) Elevation(x, y int) float64 {
	if x < 0 || x >= m.width || y < 0 || y >= m.height {
		panic("domains: coordinates outside normalized height map")
	}
	return m.elevations[y*m.width+x]
}

// Elevations returns the map's elevations in row-major order. The returned
// slice is a copy and may be modified by the caller.
func (m *NormalizedHeightMap) Elevations() []float64 {
	return append([]float64(nil), m.elevations...)
}
