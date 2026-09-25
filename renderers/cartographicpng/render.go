// Mappe - consolidated map and world generators
// Copyright (c) 2023-2026 Michael D Henderson
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

// Package cartographicpng renders normalized height maps as land, ocean, and
// ice PNGs. Its histogram-derived color bands and palettes were migrated from
// github.com/mdhender/worldgen, which was derived from John Olsson's generator.
package cartographicpng

import (
	"image"
	"image/color"
	"image/png"
	"io"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/cerrs"
)

// Configuration and input errors returned by Render.
const (
	ErrNilWriter           cerrs.Error = "cartographicpng: writer must not be nil"
	ErrNilHeightMap        cerrs.Error = "cartographicpng: height map must not be nil"
	ErrInvalidOceanPercent cerrs.Error = "cartographicpng: ocean percent must be in [0, 100]"
	ErrInvalidIcePercent   cerrs.Error = "cartographicpng: ice percent must be in [0, 100]"
	ErrInvalidTotalPercent cerrs.Error = "cartographicpng: ocean and ice percentages must total at most 100"
	ErrEmptyOceanPalette   cerrs.Error = "cartographicpng: ocean palette must not be empty"
	ErrEmptyLandPalette    cerrs.Error = "cartographicpng: land palette must not be empty"
	ErrEmptyIcePalette     cerrs.Error = "cartographicpng: ice palette must not be empty"
	ErrTransparentPalette  cerrs.Error = "cartographicpng: palette colors must be opaque"
)

// Config controls histogram band allocation and coloring. Nil palettes select
// the palettes preserved from mdhender/worldgen. Non-nil palettes must contain
// at least one opaque color.
type Config struct {
	OceanPercent int
	IcePercent   int
	OceanPalette []color.RGBA
	LandPalette  []color.RGBA
	IcePalette   []color.RGBA
}

// DefaultConfig returns the source renderer's 55 percent ocean, 8 percent ice,
// and built-in palettes. Land receives the remaining 37 percent.
func DefaultConfig() Config {
	return Config{
		OceanPercent: 55,
		IcePercent:   8,
		OceanPalette: defaultOceanPalette(),
		LandPalette:  defaultLandPalette(),
		IcePalette:   defaultIcePalette(),
	}
}

// Render writes heightMap as an opaque cartographic PNG. Elevations are
// truncated into 256 bins. Complete histogram bins are allocated to ocean and
// land in ascending order; ice receives every remaining height index.
func Render(w io.Writer, heightMap *domains.NormalizedHeightMap, cfg Config) error {
	if w == nil {
		return ErrNilWriter
	}
	if heightMap == nil {
		return ErrNilHeightMap
	}
	cfg = withDefaultPalettes(cfg)
	if err := validate(cfg); err != nil {
		return err
	}

	histogram := [256]int{}
	for _, elevation := range heightMap.Elevations() {
		histogram[heightIndex(elevation)]++
	}
	colors := colorMap(histogram, cfg)
	img := image.NewRGBA(image.Rect(0, 0, heightMap.Width(), heightMap.Height()))
	for y := 0; y < heightMap.Height(); y++ {
		for x := 0; x < heightMap.Width(); x++ {
			img.SetRGBA(x, y, colors[heightIndex(heightMap.Elevation(x, y))])
		}
	}
	return png.Encode(w, img)
}

func heightIndex(elevation float64) uint8 {
	return uint8(elevation * 255)
}

func withDefaultPalettes(cfg Config) Config {
	if cfg.OceanPalette == nil {
		cfg.OceanPalette = defaultOceanPalette()
	}
	if cfg.LandPalette == nil {
		cfg.LandPalette = defaultLandPalette()
	}
	if cfg.IcePalette == nil {
		cfg.IcePalette = defaultIcePalette()
	}
	return cfg
}

func validate(cfg Config) error {
	if cfg.OceanPercent < 0 || cfg.OceanPercent > 100 {
		return ErrInvalidOceanPercent
	}
	if cfg.IcePercent < 0 || cfg.IcePercent > 100 {
		return ErrInvalidIcePercent
	}
	if cfg.OceanPercent+cfg.IcePercent > 100 {
		return ErrInvalidTotalPercent
	}
	if len(cfg.OceanPalette) == 0 {
		return ErrEmptyOceanPalette
	}
	if len(cfg.LandPalette) == 0 {
		return ErrEmptyLandPalette
	}
	if len(cfg.IcePalette) == 0 {
		return ErrEmptyIcePalette
	}
	for _, palette := range [][]color.RGBA{cfg.OceanPalette, cfg.LandPalette, cfg.IcePalette} {
		for _, entry := range palette {
			if entry.A != 255 {
				return ErrTransparentPalette
			}
		}
	}
	return nil
}

func colorMap(histogram [256]int, cfg Config) [256]color.RGBA {
	points := 0
	for _, count := range histogram {
		points += count
	}

	height := 0
	oceanLevels := consumeLevels(histogram, &height, percentage(points, cfg.OceanPercent))
	landLevels := consumeLevels(histogram, &height, percentage(points, 100-cfg.OceanPercent-cfg.IcePercent))
	iceLevels := len(histogram) - height

	var colors [256]color.RGBA
	height = assignColors(colors[:], 0, cfg.OceanPalette, oceanLevels)
	height = assignColors(colors[:], height, cfg.LandPalette, landLevels)
	assignColors(colors[:], height, cfg.IcePalette, iceLevels)
	return colors
}

func percentage(points, percent int) int {
	return points/100*percent + points%100*percent/100
}

func consumeLevels(histogram [256]int, height *int, target int) int {
	start := *height
	for target > 0 && *height < len(histogram) {
		target -= histogram[*height]
		*height = *height + 1
	}
	return *height - start
}

func assignColors(colors []color.RGBA, height int, palette []color.RGBA, levels int) int {
	for i := 0; i < levels; i++ {
		colors[height] = palette[i*len(palette)/levels]
		height++
	}
	return height
}

func defaultOceanPalette() []color.RGBA {
	return []color.RGBA{
		{R: 0, G: 0, B: 0, A: 255},
		{R: 0, G: 0, B: 68, A: 255},
		{R: 0, G: 17, B: 102, A: 255},
		{R: 0, G: 51, B: 136, A: 255},
		{R: 0, G: 85, B: 170, A: 255},
		{R: 0, G: 119, B: 187, A: 255},
		{R: 0, G: 153, B: 221, A: 255},
		{R: 0, G: 204, B: 255, A: 255},
		{R: 34, G: 221, B: 255, A: 255},
		{R: 68, G: 238, B: 255, A: 255},
		{R: 102, G: 255, B: 255, A: 255},
		{R: 119, G: 255, B: 255, A: 255},
		{R: 136, G: 255, B: 255, A: 255},
		{R: 153, G: 255, B: 255, A: 255},
		{R: 170, G: 255, B: 255, A: 255},
		{R: 187, G: 255, B: 255, A: 255},
	}
}

func defaultLandPalette() []color.RGBA {
	return []color.RGBA{
		{R: 0, G: 68, B: 0, A: 255},
		{R: 34, G: 102, B: 0, A: 255},
		{R: 34, G: 136, B: 0, A: 255},
		{R: 119, G: 170, B: 0, A: 255},
		{R: 187, G: 221, B: 0, A: 255},
		{R: 255, G: 187, B: 34, A: 255},
		{R: 238, G: 170, B: 34, A: 255},
		{R: 221, G: 136, B: 34, A: 255},
		{R: 204, G: 136, B: 34, A: 255},
		{R: 187, G: 102, B: 34, A: 255},
		{R: 170, G: 85, B: 34, A: 255},
		{R: 153, G: 85, B: 34, A: 255},
		{R: 136, G: 68, B: 34, A: 255},
		{R: 119, G: 51, B: 34, A: 255},
		{R: 85, G: 51, B: 17, A: 255},
		{R: 68, G: 34, B: 0, A: 255},
	}
}

func defaultIcePalette() []color.RGBA {
	palette := make([]color.RGBA, 17)
	for i := range palette {
		shade := uint8(175 + 5*i)
		palette[i] = color.RGBA{R: shade, G: shade, B: shade, A: 255}
	}
	return palette
}
