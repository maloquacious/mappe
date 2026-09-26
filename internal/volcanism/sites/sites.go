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

// Package sites places volcanoes as point features and builds their cones.
//
// The number of sites is a density per land tile, shared between land
// volcanoes and ocean hotspot islands. Land sites are drawn with weights that
// favor local peaks, coasts near deep ocean, and high ground; hotspot sites are
// drawn uniformly from open ocean. Accepted sites keep a minimum spacing. Every
// site raises a cone on a copy of the heights, so the input height field is
// never modified. One pixel is one tile, and distances follow the field's
// GridTopology.
package sites

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/cerrs"
)

// Configuration errors returned by Place.
const (
	ErrNilHeightField        cerrs.Error = "sites: height field must not be nil"
	ErrNilElevationLevels    cerrs.Error = "sites: elevation levels must not be nil"
	ErrNilSource             cerrs.Error = "sites: source must not be nil"
	ErrInvalidDensity        cerrs.Error = "sites: land tiles per site must be positive"
	ErrInvalidHotspotPercent cerrs.Error = "sites: hotspot percent must be in [0, 100]"
	ErrInvalidDistance       cerrs.Error = "sites: spacing and radii must be positive and distances must not be negative"
)

// Suitability weights for land sites. Every land tile starts with the base
// weight, so volcanoes remain possible in lowland away from coasts.
const (
	baseWeight       = 1
	prominenceWeight = 4
	arcWeight        = 3
	elevationWeight  = 2
)

// Config controls site count, spacing, and cone size. Distances are in tiles.
type Config struct {
	// Source supplies the randomness for choosing sites. The caller owns it.
	Source rand.Source
	// LandTilesPerSite sets the density: one site per this many land tiles,
	// counted before volcanism and rounded down.
	LandTilesPerSite int
	// HotspotPercent is the share of sites placed in ocean as islands.
	HotspotPercent int
	// MinimumSpacing is the smallest distance allowed between two sites.
	MinimumSpacing int
	// ConeRadius is the distance at which a cone descends to sea level.
	ConeRadius int
	// ProminenceRadius is the radius of the ring a tile is compared with.
	ProminenceRadius int
	// ArcDistance is how near a land tile must be to deep ocean to count as
	// a subduction-arc coast.
	ArcDistance int
	// HotspotLandDistance is the distance within which a hotspot site may
	// have no land.
	HotspotLandDistance int
}

// DefaultConfig returns one site per 2,500 land tiles, 20 percent hotspots,
// 8-tile spacing, 4-tile cones, a 6-tile prominence ring, a 6-tile arc
// distance, and hotspots at least 3 tiles offshore. The caller supplies the
// Source.
func DefaultConfig() Config {
	return Config{
		LandTilesPerSite:    2500,
		HotspotPercent:      20,
		MinimumSpacing:      8,
		ConeRadius:          4,
		ProminenceRadius:    6,
		ArcDistance:         6,
		HotspotLandDistance: 3,
	}
}

// Result is the output of Place.
type Result struct {
	// HeightField holds the heights after cone building, with the input
	// field's topology.
	HeightField *domains.HeightField
	// Features marks each volcano site and its volcanic-highland footprint.
	Features *domains.VolcanicFeatures
	// RequestedLand and RequestedHotspot are the site counts the density
	// asked for. Features reports how many were placed; fewer are placed
	// only when spacing leaves no candidates.
	RequestedLand    int
	RequestedHotspot int
}

// Place chooses volcano sites on field and builds their cones. The elevation
// levels stay fixed: land is measured before volcanism, and cones rise toward a
// summit in the mountain band.
//
// Each cone has elevation S - (S - seaLevel)·d/R at distance d <= R from its
// site, where S = max(site elevation, (mountain + 1) / 2). A tile keeps the
// higher of its elevation and every cone's, so cones never lower terrain and
// overlapping cones merge. In ocean, the same profile raises an island. The
// site tile is a volcano; land within R of a site is volcanic highland.
func Place(field *domains.HeightField, levels *domains.ElevationLevels, cfg Config) (Result, error) {
	if field == nil {
		return Result{}, ErrNilHeightField
	}
	if levels == nil {
		return Result{}, ErrNilElevationLevels
	}
	if err := validate(cfg); err != nil {
		return Result{}, err
	}

	g := grid{width: field.Width(), height: field.Height(), topology: field.Topology()}
	elevations := field.Elevations()
	level := levels.Values()

	landTiles := 0
	for _, elevation := range elevations {
		if elevation > level.SeaLevel {
			landTiles++
		}
	}
	total := landTiles / cfg.LandTilesPerSite
	hotspots := total * cfg.HotspotPercent / 100
	result := Result{RequestedLand: total - hotspots, RequestedHotspot: hotspots}

	rnd := rand.New(cfg.Source)
	landCandidates := weightedOrder(rnd, landWeights(g, elevations, levels, cfg))
	hotspotCandidates := weightedOrder(rnd, hotspotWeights(g, elevations, level.SeaLevel, cfg))
	var accepted []int
	accepted = accept(g, accepted, landCandidates, result.RequestedLand, cfg.MinimumSpacing)
	accepted = accept(g, accepted, hotspotCandidates, len(accepted)+result.RequestedHotspot, cfg.MinimumSpacing)

	raised := buildCones(g, elevations, accepted, level, cfg.ConeRadius)
	features, err := footprints(g, elevations, raised, accepted, level.SeaLevel, cfg.ConeRadius)
	if err != nil {
		return Result{}, err
	}
	heightMap, err := domains.NewNormalizedHeightMap(g.width, g.height, raised)
	if err != nil {
		return Result{}, err
	}
	result.HeightField, err = domains.NewHeightField(heightMap, g.topology)
	if err != nil {
		return Result{}, err
	}
	result.Features = features
	return result, nil
}

func validate(cfg Config) error {
	switch {
	case cfg.Source == nil:
		return ErrNilSource
	case cfg.LandTilesPerSite < 1:
		return ErrInvalidDensity
	case cfg.HotspotPercent < 0 || cfg.HotspotPercent > 100:
		return ErrInvalidHotspotPercent
	case cfg.MinimumSpacing < 1 || cfg.ConeRadius < 1 || cfg.ProminenceRadius < 1 ||
		cfg.ArcDistance < 0 || cfg.HotspotLandDistance < 0:
		return ErrInvalidDistance
	}
	return nil
}

// landWeights returns a suitability weight for every land tile and zero for
// water.
func landWeights(g grid, elevations []float64, levels *domains.ElevationLevels, cfg Config) []float64 {
	level := levels.Values()
	ring := ringOffsets(cfg.ProminenceRadius)
	arc := diskOffsets(cfg.ArcDistance)
	weights := make([]float64, len(elevations))
	for index, elevation := range elevations {
		if elevation <= level.SeaLevel {
			continue
		}
		x, y := index%g.width, index/g.width
		weight := float64(baseWeight)
		if level.Mountain > level.SeaLevel {
			prominence := elevation - meanAt(g, elevations, x, y, ring, elevation)
			weight += prominenceWeight * max(0, min(1, prominence/(level.Mountain-level.SeaLevel)))
		}
		if anyAt(g, x, y, arc, func(i int) bool { return elevations[i] <= level.Abyss }) {
			weight += arcWeight
		}
		weight += elevationWeight * float64(levels.Classify(elevation)-domains.ElevationLowland) / 3
		weights[index] = weight
	}
	return weights
}

// hotspotWeights returns a uniform weight for every ocean tile with no land
// within the hotspot land distance, and zero elsewhere.
func hotspotWeights(g grid, elevations []float64, seaLevel float64, cfg Config) []float64 {
	offshore := diskOffsets(cfg.HotspotLandDistance)
	weights := make([]float64, len(elevations))
	for index, elevation := range elevations {
		x, y := index%g.width, index/g.width
		if elevation <= seaLevel && !anyAt(g, x, y, offshore, func(i int) bool { return elevations[i] > seaLevel }) {
			weights[index] = 1
		}
	}
	return weights
}

// weightedOrder returns the indices of positive weights in a random order in
// which higher weights tend to come first (Efraimidis-Spirakis sampling).
func weightedOrder(rnd *rand.Rand, weights []float64) []int {
	type keyed struct {
		index int
		key   float64
	}
	var candidates []keyed
	for index, weight := range weights {
		if weight > 0 {
			candidates = append(candidates, keyed{index: index, key: math.Log(1-rnd.Float64()) / weight})
		}
	}
	slices.SortFunc(candidates, func(a, b keyed) int {
		return cmp.Or(cmp.Compare(b.key, a.key), cmp.Compare(a.index, b.index))
	})
	order := make([]int, len(candidates))
	for i, candidate := range candidates {
		order[i] = candidate.index
	}
	return order
}

// accept appends candidates in order until accepted holds want sites,
// skipping any candidate nearer than spacing to an accepted site.
func accept(g grid, accepted, candidates []int, want, spacing int) []int {
	for _, candidate := range candidates {
		if len(accepted) >= want {
			break
		}
		if !slices.ContainsFunc(accepted, func(site int) bool { return g.distanceSquared(site, candidate) < spacing*spacing }) {
			accepted = append(accepted, candidate)
		}
	}
	return accepted
}

func buildCones(g grid, elevations []float64, sites []int, level domains.ElevationLevelValues, radius int) []float64 {
	raised := slices.Clone(elevations)
	disk := diskOffsets(radius)
	for _, site := range sites {
		summit := max(elevations[site], (level.Mountain+1)/2)
		x, y := site%g.width, site/g.width
		for _, offset := range disk {
			index, ok := g.index(x+offset.dx, y+offset.dy)
			if !ok {
				continue
			}
			cone := summit - (summit-level.SeaLevel)*offset.distance()/float64(radius)
			raised[index] = max(raised[index], cone)
		}
	}
	return raised
}

func footprints(g grid, original, raised []float64, sites []int, seaLevel float64, radius int) (*domains.VolcanicFeatures, error) {
	kinds := make([]domains.VolcanicKind, len(raised))
	volcanoes := make([]domains.VolcanoSite, len(sites))
	for i, site := range sites {
		kinds[site] = domains.VolcanicVolcano
		volcanoes[i] = domains.VolcanoSite{X: site % g.width, Y: site / g.width, Hotspot: original[site] <= seaLevel}
	}
	disk := diskOffsets(radius)
	for _, site := range sites {
		x, y := site%g.width, site/g.width
		for _, offset := range disk {
			index, ok := g.index(x+offset.dx, y+offset.dy)
			if ok && kinds[index] == domains.VolcanicNone && raised[index] > seaLevel {
				kinds[index] = domains.VolcanicHighland
			}
		}
	}
	return domains.NewVolcanicFeatures(g.width, g.height, volcanoes, kinds)
}

// meanAt returns the mean elevation at the offsets from (x, y) that lie on
// the grid, or fallback when none do.
func meanAt(g grid, elevations []float64, x, y int, offsets []offset, fallback float64) float64 {
	total, count := 0.0, 0
	for _, offset := range offsets {
		if index, ok := g.index(x+offset.dx, y+offset.dy); ok {
			total += elevations[index]
			count++
		}
	}
	if count == 0 {
		return fallback
	}
	return total / float64(count)
}

// anyAt reports whether match holds for any on-grid offset from (x, y).
func anyAt(g grid, x, y int, offsets []offset, match func(index int) bool) bool {
	for _, offset := range offsets {
		if index, ok := g.index(x+offset.dx, y+offset.dy); ok && match(index) {
			return true
		}
	}
	return false
}

type offset struct{ dx, dy int }

func (o offset) distance() float64 { return math.Sqrt(float64(o.dx*o.dx + o.dy*o.dy)) }

// diskOffsets returns every offset within radius tiles, including the origin.
func diskOffsets(radius int) []offset {
	var offsets []offset
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				offsets = append(offsets, offset{dx: dx, dy: dy})
			}
		}
	}
	return offsets
}

// ringOffsets returns the offsets whose distance rounds to radius.
func ringOffsets(radius int) []offset {
	var offsets []offset
	inner, outer := (2*radius-1)*(2*radius-1), (2*radius+1)*(2*radius+1)
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if d := 4 * (dx*dx + dy*dy); d >= inner && d < outer {
				offsets = append(offsets, offset{dx: dx, dy: dy})
			}
		}
	}
	return offsets
}

// grid resolves coordinates and distances under a Cartesian topology.
type grid struct {
	width, height int
	topology      domains.GridTopology
}

// index returns the row-major index of (x, y), wrapping wrapped axes. It
// reports false for coordinates beyond a bounded edge.
func (g grid) index(x, y int) (int, bool) {
	if x < 0 || x >= g.width {
		if !g.topology.WrapEastWest {
			return 0, false
		}
		x = modulo(x, g.width)
	}
	if y < 0 || y >= g.height {
		if !g.topology.WrapNorthSouth {
			return 0, false
		}
		y = modulo(y, g.height)
	}
	return y*g.width + x, true
}

// distanceSquared returns the squared distance between two indices, measured
// the shorter way around wrapped axes.
func (g grid) distanceSquared(a, b int) int {
	dx := abs(a%g.width - b%g.width)
	dy := abs(a/g.width - b/g.width)
	if g.topology.WrapEastWest {
		dx = min(dx, g.width-dx)
	}
	if g.topology.WrapNorthSouth {
		dy = min(dy, g.height-dy)
	}
	return dx*dx + dy*dy
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func modulo(value, modulus int) int {
	value %= modulus
	if value < 0 {
		value += modulus
	}
	return value
}
