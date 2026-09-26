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

// Package flatcartographicpng composes the flat generator with the
// cartographic PNG renderer.
package flatcartographicpng

import (
	"fmt"
	"io"

	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/renderers/cartographicpng"
)

// Run generates a flat normalized height map and writes it as a cartographic
// PNG to w.
func Run(w io.Writer, generatorConfig flat.Config, rendererConfig cartographicpng.Config) error {
	heightField, err := flat.GenerateHeightField(generatorConfig)
	if err != nil {
		return fmt.Errorf("flat-cartographic-png: generate height map: %w", err)
	}
	if err := cartographicpng.Render(w, heightField.HeightMap(), rendererConfig); err != nil {
		return fmt.Errorf("flat-cartographic-png: render height map: %w", err)
	}
	return nil
}
