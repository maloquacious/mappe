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

// ErrNilHeightMap is returned when a height field is constructed without map
// data.
const ErrNilHeightMap cerrs.Error = "domains: height field map must not be nil"

// GridTopology describes which opposite edges of a Cartesian grid are joined.
// East-west wrapping joins the left and right edges; north-south wrapping joins
// the top and bottom edges.
type GridTopology struct {
	WrapEastWest   bool
	WrapNorthSouth bool
}

// HeightField carries normalized elevation data and the spatial topology that
// downstream stages need when inspecting neighboring cells.
type HeightField struct {
	topology  GridTopology
	heightMap *NormalizedHeightMap
}

// NewHeightField constructs a height field from immutable normalized map data.
func NewHeightField(heightMap *NormalizedHeightMap, topology GridTopology) (*HeightField, error) {
	if heightMap == nil {
		return nil, ErrNilHeightMap
	}
	return &HeightField{topology: topology, heightMap: heightMap}, nil
}

// Topology returns the field's Cartesian edge topology.
func (f *HeightField) Topology() GridTopology { return f.topology }

// HeightMap returns the field's immutable normalized elevation map.
func (f *HeightField) HeightMap() *NormalizedHeightMap { return f.heightMap }

// Width returns the number of columns in the field.
func (f *HeightField) Width() int { return f.heightMap.Width() }

// Height returns the number of rows in the field.
func (f *HeightField) Height() int { return f.heightMap.Height() }

// Elevation returns the normalized elevation at (x, y).
func (f *HeightField) Elevation(x, y int) float64 { return f.heightMap.Elevation(x, y) }

// Elevations returns a copy of the field's normalized elevations in row-major
// order.
func (f *HeightField) Elevations() []float64 { return f.heightMap.Elevations() }
