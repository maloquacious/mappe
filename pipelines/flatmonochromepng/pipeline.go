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

// Package flatmonochromepng composes the flat generator with the monochrome
// PNG renderer.
package flatmonochromepng

import (
	"fmt"
	"io"

	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/renderers/monochromepng"
)

// Run generates a flat normalized height map and writes it as a monochrome PNG
// to w.
func Run(w io.Writer, cfg flat.Config) error {
	heightField, err := flat.GenerateHeightField(cfg)
	if err != nil {
		return fmt.Errorf("flat-monochrome-png: generate height map: %w", err)
	}
	if err := monochromepng.Render(w, heightField.HeightMap()); err != nil {
		return fmt.Errorf("flat-monochrome-png: render height map: %w", err)
	}
	return nil
}
