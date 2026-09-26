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

// Package flatterrainpng composes the flat generator, percentile elevation
// levels, deterministic diagnostic environmental fields, and the terrain PNG
// renderer.
package flatterrainpng

import (
	"fmt"
	"io"
	"math"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/internal/levels/percentile"
	"github.com/maloquacious/mappe/renderers/terrainpng"
)

// mountainCooling is the heat removed at the mountain level. Cooling grows
// linearly with height above sea level and continues above the mountain level.
const mountainCooling = 0.1875

// Run generates a flat height field, derives its elevation levels from tile
// percentages, derives deterministic periodic environmental samples, classifies
// their terrain, and writes a PNG to w. The environmental fields are diagnostic
// rather than a climatological model.
func Run(w io.Writer, generatorConfig flat.Config, levelsConfig percentile.Config, classificationConfig domains.ClassificationConfig) error {
	heightField, err := flat.GenerateHeightField(generatorConfig)
	if err != nil {
		return fmt.Errorf("flat-terrain-png: generate height field: %w", err)
	}
	levels, err := percentile.Derive(heightField, levelsConfig)
	if err != nil {
		return fmt.Errorf("flat-terrain-png: derive elevation levels: %w", err)
	}
	environmentalMap, err := classify(heightField, levels, classificationConfig)
	if err != nil {
		return fmt.Errorf("flat-terrain-png: classify environment: %w", err)
	}
	if err := terrainpng.Render(w, environmentalMap); err != nil {
		return fmt.Errorf("flat-terrain-png: render terrain: %w", err)
	}
	return nil
}

func classify(heightField *domains.HeightField, levels *domains.ElevationLevels, cfg domains.ClassificationConfig) (*domains.EnvironmentalMap, error) {
	width, height := heightField.Width(), heightField.Height()
	seaLevel, mountain := levels.Values().SeaLevel, levels.Values().Mountain
	samples := make([]domains.EnvironmentalSample, width*height)
	for y := 0; y < height; y++ {
		phaseY := 2 * math.Pi * float64(y) / float64(height)
		for x := 0; x < width; x++ {
			phaseX := 2 * math.Pi * float64(x) / float64(width)
			elevation := heightField.Elevation(x, y)
			heat := 0.5 + 0.3*math.Sin(phaseY) + 0.2*math.Cos(phaseX+phaseY)
			heat -= mountainCooling * riseAboveSea(elevation, seaLevel, mountain)
			samples[y*width+x] = domains.EnvironmentalSample{
				Elevation: elevation,
				Heat:      clampNormalized(heat),
				Moisture:  clampNormalized(0.5 + 0.3*math.Sin(phaseX-phaseY) + 0.2*math.Cos(phaseX+2*phaseY)),
				Relief:    reliefAt(heightField, x, y),
				Basin:     clampNormalized(0.5 + 0.5*math.Sin(phaseX+2*phaseY)),
				Volcanic:  clampNormalized(0.5 + 0.5*math.Cos(3*phaseX-phaseY)),
			}
		}
	}
	return domains.NewEnvironmentalMap(width, height, heightField.Topology(), samples, levels, cfg)
}

// riseAboveSea returns height above sea level in units of the mountain level's
// height above sea level: 0 at or below the shore and 1 at the mountain level.
func riseAboveSea(elevation, seaLevel, mountain float64) float64 {
	if elevation <= seaLevel {
		return 0
	}
	if mountain <= seaLevel {
		return 1
	}
	return (elevation - seaLevel) / (mountain - seaLevel)
}

func reliefAt(heightField *domains.HeightField, x, y int) float64 {
	topology := heightField.Topology()
	total := 0.0
	neighbors := 0
	for _, offset := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		nx, ny := x+offset[0], y+offset[1]
		if nx < 0 || nx >= heightField.Width() {
			if !topology.WrapEastWest {
				continue
			}
			nx = modulo(nx, heightField.Width())
		}
		if ny < 0 || ny >= heightField.Height() {
			if !topology.WrapNorthSouth {
				continue
			}
			ny = modulo(ny, heightField.Height())
		}
		total += math.Abs(heightField.Elevation(x, y) - heightField.Elevation(nx, ny))
		neighbors++
	}
	if neighbors == 0 {
		return 0
	}
	return clampNormalized(8 * total / float64(neighbors))
}

func clampNormalized(value float64) float64 { return max(0, min(1, value)) }

func modulo(value, modulus int) int {
	value %= modulus
	if value < 0 {
		value += modulus
	}
	return value
}
