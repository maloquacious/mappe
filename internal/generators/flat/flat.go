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

// Package flat generates normalized planar height maps by repeatedly raising
// or lowering circular regions. It was migrated from
// github.com/mdhender/mapgen/pkg/generators/flat.
package flat

import (
	"math/rand/v2"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/cerrs"
)

// Configuration errors returned by GenerateNormalizedHeightMap.
const (
	ErrNilSource         cerrs.Error = "flat: source must not be nil"
	ErrInvalidWidth      cerrs.Error = "flat: width must be at least two"
	ErrInvalidHeight     cerrs.Error = "flat: height must be at least two"
	ErrInvalidIterations cerrs.Error = "flat: iterations must not be negative"
	ErrMapTooLarge       cerrs.Error = "flat: map dimensions are too large"
)

// Config controls generation. Source must not be nil and is advanced during
// generation. When Wrap is true, circles wrap across both map axes; otherwise
// they are clipped at the map edges.
type Config struct {
	Source     rand.Source
	Width      int
	Height     int
	Iterations int
	Wrap       bool
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

func generateRaw(cfg Config) ([]int, error) {
	if err := validate(cfg); err != nil {
		return nil, err
	}

	elevations := make([]int, cfg.Width*cfg.Height)
	rnd := rand.New(cfg.Source)
	maxRadius := min(cfg.Width, cfg.Height) / 2
	for range cfg.Iterations {
		bump := 1
		if rnd.IntN(2) == 1 {
			bump = -1
		}
		radius := rnd.IntN(maxRadius) + 1
		centerX, centerY := rnd.IntN(cfg.Width), rnd.IntN(cfg.Height)
		applyCircle(elevations, cfg.Width, cfg.Height, centerX, centerY, radius, bump, cfg.Wrap)
	}
	return elevations, nil
}

func validate(cfg Config) error {
	if cfg.Source == nil {
		return ErrNilSource
	}
	if cfg.Width < 2 {
		return ErrInvalidWidth
	}
	if cfg.Height < 2 {
		return ErrInvalidHeight
	}
	if cfg.Iterations < 0 {
		return ErrInvalidIterations
	}
	if cfg.Width > int(^uint(0)>>1)/cfg.Height {
		return ErrMapTooLarge
	}
	return nil
}

func applyCircle(elevations []int, width, height, centerX, centerY, radius, bump int, wrap bool) {
	minimumX, maximumX := centerX-radius, centerX+radius
	minimumY, maximumY := centerY-radius, centerY+radius
	if !wrap {
		minimumX = max(minimumX, 0)
		maximumX = min(maximumX, width)
		minimumY = max(minimumY, 0)
		maximumY = min(maximumY, height)
	}

	radiusSquared := radius * radius
	for x := minimumX; x < maximumX; x++ {
		for y := minimumY; y < maximumY; y++ {
			dx, dy := x-centerX, y-centerY
			if dx*dx+dy*dy >= radiusSquared {
				continue
			}
			px, py := x, y
			if wrap {
				px = modulo(px, width)
				py = modulo(py, height)
			}
			elevations[py*width+px] += bump
		}
	}
}

func modulo(value, modulus int) int {
	value %= modulus
	if value < 0 {
		value += modulus
	}
	return value
}

func normalize(elevations []int) []float64 {
	normalized := make([]float64, len(elevations))
	minimum, maximum := elevations[0], elevations[0]
	for _, elevation := range elevations[1:] {
		minimum = min(minimum, elevation)
		maximum = max(maximum, elevation)
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
