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

// Package terrainpng renders classified environmental maps as categorical
// terrain PNGs.
package terrainpng

import (
	"image"
	"image/color"
	"image/png"
	"io"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/cerrs"
)

// Input errors returned by Render.
const (
	ErrNilWriter           cerrs.Error = "terrainpng: writer must not be nil"
	ErrNilEnvironmentalMap cerrs.Error = "terrainpng: environmental map must not be nil"
)

// Render writes environmentalMap as an opaque categorical terrain PNG.
func Render(w io.Writer, environmentalMap *domains.EnvironmentalMap) error {
	if w == nil {
		return ErrNilWriter
	}
	if environmentalMap == nil {
		return ErrNilEnvironmentalMap
	}

	img := image.NewRGBA(image.Rect(0, 0, environmentalMap.Width(), environmentalMap.Height()))
	for y := 0; y < environmentalMap.Height(); y++ {
		for x := 0; x < environmentalMap.Width(); x++ {
			img.SetRGBA(x, y, terrainColor(environmentalMap.Cell(x, y).Terrain))
		}
	}
	return png.Encode(w, img)
}

func terrainColor(terrain domains.Terrain) color.RGBA {
	if terrain.Valid() {
		return palette[terrain]
	}
	return color.RGBA{R: 255, B: 255, A: 255}
}

var palette = [27]color.RGBA{
	domains.TerrainDeepOcean:        rgba(8, 29, 88),
	domains.TerrainOcean:            rgba(34, 94, 168),
	domains.TerrainShallowSea:       rgba(65, 182, 196),
	domains.TerrainCoastalWater:     rgba(127, 205, 187),
	domains.TerrainInlandSea:        rgba(44, 127, 184),
	domains.TerrainLake:             rgba(116, 169, 207),
	domains.TerrainGlacialIce:       rgba(247, 251, 255),
	domains.TerrainTundra:           rgba(203, 213, 168),
	domains.TerrainMarsh:            rgba(90, 143, 89),
	domains.TerrainSwamp:            rgba(47, 107, 69),
	domains.TerrainBog:              rgba(107, 127, 74),
	domains.TerrainDesert:           rgba(232, 198, 106),
	domains.TerrainBadlands:         rgba(166, 95, 60),
	domains.TerrainScrubland:        rgba(181, 155, 99),
	domains.TerrainPlains:           rgba(168, 198, 108),
	domains.TerrainGrassland:        rgba(115, 169, 66),
	domains.TerrainSteppe:           rgba(184, 180, 106),
	domains.TerrainSavanna:          rgba(201, 184, 74),
	domains.TerrainBorealForest:     rgba(56, 102, 65),
	domains.TerrainTemperateForest:  rgba(45, 106, 79),
	domains.TerrainRainforest:       rgba(20, 83, 45),
	domains.TerrainHills:            rgba(139, 125, 87),
	domains.TerrainMountain:         rgba(116, 111, 103),
	domains.TerrainAlpine:           rgba(183, 188, 197),
	domains.TerrainVolcano:          rgba(178, 34, 34),
	domains.TerrainVolcanicHighland: rgba(110, 59, 42),
	domains.TerrainCoast:            rgba(217, 195, 140),
}

func rgba(red, green, blue uint8) color.RGBA {
	return color.RGBA{R: red, G: green, B: blue, A: 255}
}
