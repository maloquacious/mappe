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

// Package monochromepng renders normalized height maps as grayscale PNGs.
package monochromepng

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/cerrs"
)

// Input errors returned by Render.
const (
	ErrNilWriter    cerrs.Error = "monochromepng: writer must not be nil"
	ErrNilHeightMap cerrs.Error = "monochromepng: height map must not be nil"
)

// Render writes heightMap as an 8-bit grayscale PNG. Elevation zero maps to
// black, elevation one maps to white, and intermediate elevations map to the
// nearest grayscale value.
func Render(w io.Writer, heightMap *domains.NormalizedHeightMap) error {
	if w == nil {
		return ErrNilWriter
	}
	if heightMap == nil {
		return ErrNilHeightMap
	}

	img := image.NewGray(image.Rect(0, 0, heightMap.Width(), heightMap.Height()))
	for y := 0; y < heightMap.Height(); y++ {
		for x := 0; x < heightMap.Width(); x++ {
			shade := uint8(math.Round(heightMap.Elevation(x, y) * 255))
			img.SetGray(x, y, color.Gray{Y: shade})
		}
	}

	return png.Encode(w, img)
}
