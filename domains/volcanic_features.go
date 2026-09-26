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

// Volcanic feature construction errors.
const (
	ErrInvalidVolcanicKindCount cerrs.Error = "domains: volcanic kind count must equal width times height"
	ErrInvalidVolcanicKind      cerrs.Error = "domains: volcanic kinds must be declared values"
	ErrInvalidVolcanoSite       cerrs.Error = "domains: volcano sites must be distinct in-bounds tiles marked as volcanoes"
	ErrUnsitedVolcano           cerrs.Error = "domains: every volcano tile must be a volcano site"
)

// VolcanicKind marks a tile's role in a volcanic feature.
type VolcanicKind uint8

// Volcanic kinds.
const (
	VolcanicNone     VolcanicKind = 0
	VolcanicHighland VolcanicKind = 1
	VolcanicVolcano  VolcanicKind = 2
)

// String returns the name of the volcanic kind.
func (k VolcanicKind) String() string {
	switch k {
	case VolcanicNone:
		return "none"
	case VolcanicHighland:
		return "volcanic-highland"
	case VolcanicVolcano:
		return "volcano"
	default:
		return "unknown"
	}
}

// Valid reports whether k is a declared volcanic kind.
func (k VolcanicKind) Valid() bool { return k <= VolcanicVolcano }

// VolcanoSite is the tile holding a volcano's summit. Hotspot is true for a
// site placed in ocean that rose as an island.
type VolcanoSite struct {
	X, Y    int
	Hotspot bool
}

// VolcanicFeatures is an immutable, validated, row-major grid of volcanic
// kinds together with the volcano sites that produced them. It records nothing
// about how the sites were chosen, so any stage may produce it.
type VolcanicFeatures struct {
	width  int
	height int
	sites  []VolcanoSite
	kinds  []VolcanicKind
}

// NewVolcanicFeatures validates and copies sites and row-major kinds. A nil
// kinds slice with no sites describes a grid without volcanism. Every site
// must be a distinct in-bounds tile marked VolcanicVolcano, and every
// VolcanicVolcano tile must be a site.
func NewVolcanicFeatures(width, height int, sites []VolcanoSite, kinds []VolcanicKind) (*VolcanicFeatures, error) {
	if width < 1 {
		return nil, ErrInvalidWidth
	}
	if height < 1 {
		return nil, ErrInvalidHeight
	}
	if width > int(^uint(0)>>1)/height {
		return nil, ErrInvalidVolcanicKindCount
	}
	if kinds == nil && len(sites) == 0 {
		kinds = make([]VolcanicKind, width*height)
	}
	if len(kinds) != width*height {
		return nil, ErrInvalidVolcanicKindCount
	}
	volcanoes := 0
	for _, kind := range kinds {
		if !kind.Valid() {
			return nil, ErrInvalidVolcanicKind
		}
		if kind == VolcanicVolcano {
			volcanoes++
		}
	}
	seen := make(map[int]bool, len(sites))
	for _, site := range sites {
		index := site.Y*width + site.X
		if site.X < 0 || site.X >= width || site.Y < 0 || site.Y >= height || seen[index] || kinds[index] != VolcanicVolcano {
			return nil, ErrInvalidVolcanoSite
		}
		seen[index] = true
	}
	if volcanoes != len(sites) {
		return nil, ErrUnsitedVolcano
	}
	return &VolcanicFeatures{
		width:  width,
		height: height,
		sites:  append([]VolcanoSite(nil), sites...),
		kinds:  append([]VolcanicKind(nil), kinds...),
	}, nil
}

// Width returns the number of columns in the grid.
func (f *VolcanicFeatures) Width() int { return f.width }

// Height returns the number of rows in the grid.
func (f *VolcanicFeatures) Height() int { return f.height }

// Sites returns a copy of the volcano sites.
func (f *VolcanicFeatures) Sites() []VolcanoSite { return append([]VolcanoSite(nil), f.sites...) }

// SiteCounts returns the number of land volcanoes and hotspot islands.
func (f *VolcanicFeatures) SiteCounts() (land, hotspot int) {
	for _, site := range f.sites {
		if site.Hotspot {
			hotspot++
		} else {
			land++
		}
	}
	return land, hotspot
}

// Kind returns the volcanic kind at (x, y). It panics for coordinates outside
// the grid.
func (f *VolcanicFeatures) Kind(x, y int) VolcanicKind {
	if x < 0 || x >= f.width || y < 0 || y >= f.height {
		panic("domains: coordinates outside volcanic features")
	}
	return f.kinds[y*f.width+x]
}
