package sites

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/internal/levels/percentile"
)

func TestPlaceIsDeterministicAndLeavesInputUnchanged(t *testing.T) {
	field, levels := generatedWorld(t, 128, 64)
	before := field.Elevations()
	cfg := testConfig(42)
	cfg.LandTilesPerSite = 100

	first, err := Place(field, levels, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(field.Elevations(), before) {
		t.Fatal("Place changed the input height field")
	}
	cfg.Source = rand.NewPCG(42, 1)
	second, err := Place(field, levels, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(first.Features.Sites(), second.Features.Sites()) || !slices.Equal(first.HeightField.Elevations(), second.HeightField.Elevations()) {
		t.Fatal("same source produced different volcanism")
	}
	cfg.Source = rand.NewPCG(43, 1)
	third, err := Place(field, levels, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Equal(first.Features.Sites(), third.Features.Sites()) {
		t.Fatal("different sources produced the same sites")
	}
	if first.HeightField.Topology() != field.Topology() {
		t.Fatalf("topology = %+v, want %+v", first.HeightField.Topology(), field.Topology())
	}
}

func TestPlaceMeetsDensityWithSpacedLandAndHotspotSites(t *testing.T) {
	field, levels := generatedWorld(t, 128, 64)
	cfg := testConfig(42)
	cfg.LandTilesPerSite = 100
	result, err := Place(field, levels, cfg)
	if err != nil {
		t.Fatal(err)
	}

	landTiles := levels.Composition(field).LandTiles()
	total := landTiles / 100
	if result.RequestedLand+result.RequestedHotspot != total || result.RequestedHotspot != total*20/100 {
		t.Fatalf("requested %d land and %d hotspot sites for %d land tiles, want %d in total with 20%% hotspots",
			result.RequestedLand, result.RequestedHotspot, landTiles, total)
	}
	land, hotspot := result.Features.SiteCounts()
	if land != result.RequestedLand || hotspot != result.RequestedHotspot {
		t.Fatalf("placed %d land and %d hotspot sites, want %d and %d", land, hotspot, result.RequestedLand, result.RequestedHotspot)
	}

	g := grid{width: field.Width(), height: field.Height(), topology: field.Topology()}
	sites := result.Features.Sites()
	for i, a := range sites {
		original := field.Elevation(a.X, a.Y)
		if a.Hotspot != (original <= levels.Values().SeaLevel) {
			t.Errorf("site %+v at original elevation %v has wrong hotspot flag", a, original)
		}
		for _, b := range sites[i+1:] {
			if d := g.distanceSquared(a.Y*g.width+a.X, b.Y*g.width+b.X); d < cfg.MinimumSpacing*cfg.MinimumSpacing {
				t.Errorf("sites %+v and %+v are %.2f tiles apart, want at least %d", a, b, math.Sqrt(float64(d)), cfg.MinimumSpacing)
			}
		}
	}
}

func TestPlaceBuildsConeToTheMountainBand(t *testing.T) {
	const size = 21
	field := uniformField(t, size, size, 0.6, domains.GridTopology{})
	levels := testLevels(t)
	cfg := testConfig(7)
	cfg.LandTilesPerSite = size * size
	result, err := Place(field, levels, cfg)
	if err != nil {
		t.Fatal(err)
	}
	sites := result.Features.Sites()
	if len(sites) != 1 {
		t.Fatalf("sites = %+v, want exactly one", sites)
	}
	site := sites[0]
	const summit = 0.9 // halfway between the 0.8 mountain level and 1
	for y := range size {
		for x := range size {
			d := math.Hypot(float64(x-site.X), float64(y-site.Y))
			want, wantKind := 0.6, domains.VolcanicNone
			if d <= 4 {
				want = max(0.6, summit-(summit-0.5)*d/4)
				wantKind = domains.VolcanicHighland
			}
			if x == site.X && y == site.Y {
				wantKind = domains.VolcanicVolcano
			}
			if got := result.HeightField.Elevation(x, y); math.Abs(got-want) > 1e-12 {
				t.Errorf("elevation at (%d, %d), %.2f from the site = %v, want %v", x, y, d, got, want)
			}
			if got := result.Features.Kind(x, y); got != wantKind {
				t.Errorf("kind at (%d, %d), %.2f from the site = %v, want %v", x, y, d, got, wantKind)
			}
		}
	}
	if band := levels.Classify(result.HeightField.Elevation(site.X, site.Y)); band != domains.ElevationMountain {
		t.Fatalf("summit band = %v, want mountain", band)
	}
}

func TestPlaceKeepsHigherTerrainUnderACone(t *testing.T) {
	elevations := make([]float64, 9*9)
	for i := range elevations {
		elevations[i] = 0.6
	}
	elevations[4*9+4] = 0.95 // a peak above the cone summit
	field := fieldFrom(t, 9, 9, elevations, domains.GridTopology{})
	cfg := testConfig(7)
	cfg.LandTilesPerSite = 81
	result, err := Place(field, testLevels(t), cfg)
	if err != nil {
		t.Fatal(err)
	}
	for y := range 9 {
		for x := range 9 {
			if got, original := result.HeightField.Elevation(x, y), field.Elevation(x, y); got < original {
				t.Fatalf("elevation at (%d, %d) fell from %v to %v", x, y, original, got)
			}
		}
	}
	if site := result.Features.Sites()[0]; site.X == 4 && site.Y == 4 && result.HeightField.Elevation(4, 4) != 0.95 {
		t.Fatalf("summit on a 0.95 peak = %v, want the peak kept", result.HeightField.Elevation(4, 4))
	}
}

func TestPlaceRaisesHotspotIslandsOffshore(t *testing.T) {
	const size = 30
	elevations := make([]float64, size*size)
	for i := range elevations {
		elevations[i] = 0.3
		if i%size < 2 {
			elevations[i] = 0.6 // a two-column strip of land along the west edge
		}
	}
	field := fieldFrom(t, size, size, elevations, domains.GridTopology{})
	cfg := testConfig(11)
	cfg.LandTilesPerSite = 2 * size
	cfg.HotspotPercent = 100
	result, err := Place(field, testLevels(t), cfg)
	if err != nil {
		t.Fatal(err)
	}
	sites := result.Features.Sites()
	if result.RequestedHotspot != 1 || len(sites) != 1 || !sites[0].Hotspot {
		t.Fatalf("requested %d hotspots and placed %+v, want one hotspot", result.RequestedHotspot, sites)
	}
	site := sites[0]
	if site.X-1 < cfg.HotspotLandDistance {
		t.Fatalf("hotspot at x=%d is within %d tiles of land at x=1", site.X, cfg.HotspotLandDistance)
	}
	island := 0
	for y := range size {
		for x := 2; x < size; x++ {
			d := math.Hypot(float64(x-site.X), float64(y-site.Y))
			elevation := result.HeightField.Elevation(x, y)
			if d < 4 {
				island++
				if elevation <= 0.5 || result.Features.Kind(x, y) == domains.VolcanicNone {
					t.Errorf("island tile (%d, %d) = %v/%v, want volcanic land", x, y, elevation, result.Features.Kind(x, y))
				}
			} else if elevation > 0.5 || result.Features.Kind(x, y) != domains.VolcanicNone {
				t.Errorf("ocean tile (%d, %d), %.2f from the site = %v/%v, want unchanged water", x, y, d, elevation, result.Features.Kind(x, y))
			}
		}
	}
	if island < 2 {
		t.Fatalf("island covers %d tiles, want a cone", island)
	}
}

func TestPlaceUsesTopologyAtEveryEdge(t *testing.T) {
	const size = 12
	for _, topology := range allTopologies() {
		t.Run(fmt.Sprintf("%+v", topology), func(t *testing.T) {
			// The only land is the corner tile, so it must be the site.
			elevations := make([]float64, size*size)
			for i := range elevations {
				elevations[i] = 0.3
			}
			elevations[0] = 0.6
			field := fieldFrom(t, size, size, elevations, topology)
			cfg := testConfig(3)
			cfg.LandTilesPerSite = 1
			cfg.HotspotPercent = 0
			result, err := Place(field, testLevels(t), cfg)
			if err != nil {
				t.Fatal(err)
			}
			for _, tt := range []struct {
				x, y int
				want bool
			}{
				{x: 1, y: 0, want: true},
				{x: size - 1, y: 0, want: topology.WrapEastWest},
				{x: 0, y: size - 1, want: topology.WrapNorthSouth},
				{x: size - 1, y: size - 1, want: topology.WrapEastWest && topology.WrapNorthSouth},
			} {
				raised := result.HeightField.Elevation(tt.x, tt.y) > 0.5
				highland := result.Features.Kind(tt.x, tt.y) == domains.VolcanicHighland
				if raised != tt.want || highland != tt.want {
					t.Errorf("(%d, %d) raised=%v highland=%v, want %v", tt.x, tt.y, raised, highland, tt.want)
				}
			}
		})
	}
}

func TestPlaceMeasuresSpacingAcrossWrappedSeams(t *testing.T) {
	const size = 12
	for _, topology := range allTopologies() {
		for _, tt := range []struct {
			name    string
			x, y    int
			wrapped bool
		}{
			{name: "east-west", x: size - 1, y: 0, wrapped: topology.WrapEastWest},
			{name: "north-south", x: 0, y: size - 1, wrapped: topology.WrapNorthSouth},
		} {
			t.Run(fmt.Sprintf("%+v/%s", topology, tt.name), func(t *testing.T) {
				elevations := make([]float64, size*size)
				for i := range elevations {
					elevations[i] = 0.3
				}
				elevations[0], elevations[tt.y*size+tt.x] = 0.6, 0.6
				cfg := testConfig(5)
				cfg.LandTilesPerSite = 1
				cfg.HotspotPercent = 0
				result, err := Place(fieldFrom(t, size, size, elevations, topology), testLevels(t), cfg)
				if err != nil {
					t.Fatal(err)
				}
				want := 2
				if tt.wrapped {
					want = 1 // the two tiles are neighbors across the seam
				}
				if got := len(result.Features.Sites()); got != want {
					t.Fatalf("placed %d sites, want %d", got, want)
				}
			})
		}
	}
}

func TestLandWeightsFavorPeaksArcsAndHighGround(t *testing.T) {
	const size = 15
	elevations := make([]float64, size*size)
	for i := range elevations {
		elevations[i] = 0.55 // lowland
	}
	elevations[7*size+7] = 0.85  // a mountain peak in the middle
	elevations[0*size+14] = 0.1  // deep ocean in the north-east corner
	elevations[14*size+0] = 0.55 // lowland far from both
	field := fieldFrom(t, size, size, elevations, domains.GridTopology{})
	weights := landWeights(grid{width: size, height: size}, field.Elevations(), testLevels(t), DefaultConfig())

	if got := weights[14*size+0]; math.Abs(got-baseWeight) > 1e-9 {
		t.Errorf("isolated lowland weight = %v, want the base weight %v", got, float64(baseWeight))
	}
	if got := weights[1*size+13]; math.Abs(got-(baseWeight+arcWeight)) > 1e-9 {
		t.Errorf("lowland beside deep ocean weight = %v, want %v", got, float64(baseWeight+arcWeight))
	}
	// The peak rises 0.3 above its 0.55 ring; the land relief below the
	// 0.8 mountain level is 0.3, so prominence saturates.
	if got, want := weights[7*size+7], float64(baseWeight+prominenceWeight+elevationWeight); math.Abs(got-want) > 1e-9 {
		t.Errorf("mountain peak weight = %v, want %v", got, want)
	}
	if weights[0*size+14] != 0 {
		t.Error("ocean tile has a land weight")
	}
}

func TestPlaceRejectsInvalidInput(t *testing.T) {
	field := uniformField(t, 4, 4, 0.6, domains.GridTopology{})
	levels := testLevels(t)
	for _, tt := range []struct {
		name   string
		change func(*Config)
		want   error
	}{
		{name: "source", change: func(c *Config) { c.Source = nil }, want: ErrNilSource},
		{name: "density", change: func(c *Config) { c.LandTilesPerSite = 0 }, want: ErrInvalidDensity},
		{name: "negative hotspot", change: func(c *Config) { c.HotspotPercent = -1 }, want: ErrInvalidHotspotPercent},
		{name: "hotspot above 100", change: func(c *Config) { c.HotspotPercent = 101 }, want: ErrInvalidHotspotPercent},
		{name: "spacing", change: func(c *Config) { c.MinimumSpacing = 0 }, want: ErrInvalidDistance},
		{name: "cone radius", change: func(c *Config) { c.ConeRadius = 0 }, want: ErrInvalidDistance},
		{name: "prominence radius", change: func(c *Config) { c.ProminenceRadius = 0 }, want: ErrInvalidDistance},
		{name: "arc distance", change: func(c *Config) { c.ArcDistance = -1 }, want: ErrInvalidDistance},
		{name: "hotspot distance", change: func(c *Config) { c.HotspotLandDistance = -1 }, want: ErrInvalidDistance},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig(1)
			tt.change(&cfg)
			if _, err := Place(field, levels, cfg); err != tt.want {
				t.Fatalf("Place error = %v, want %v", err, tt.want)
			}
		})
	}
	if _, err := Place(nil, levels, testConfig(1)); err != ErrNilHeightField {
		t.Fatalf("nil field error = %v, want %v", err, ErrNilHeightField)
	}
	if _, err := Place(field, nil, testConfig(1)); err != ErrNilElevationLevels {
		t.Fatalf("nil levels error = %v, want %v", err, ErrNilElevationLevels)
	}
}

func TestDefaultDensityKeepsVolcanismRare(t *testing.T) {
	field, levels := generatedWorld(t, 320, 160)
	result, err := Place(field, levels, testConfig(42))
	if err != nil {
		t.Fatal(err)
	}
	landTiles := levels.Composition(field).LandTiles()
	land, hotspot := result.Features.SiteCounts()
	if land+hotspot != landTiles/2500 || land+hotspot == 0 {
		t.Fatalf("placed %d sites for %d land tiles, want %d", land+hotspot, landTiles, landTiles/2500)
	}
	volcanic := 0
	for y := range field.Height() {
		for x := range field.Width() {
			if result.Features.Kind(x, y) != domains.VolcanicNone {
				volcanic++
			}
		}
	}
	footprint := len(diskOffsets(DefaultConfig().ConeRadius))
	if volcanic > (land+hotspot)*footprint {
		t.Fatalf("%d volcanic tiles exceed %d sites × %d-tile footprints", volcanic, land+hotspot, footprint)
	}
	if share := float64(volcanic) / float64(field.Width()*field.Height()); share > 0.01 {
		t.Fatalf("volcanic share = %.4f, want at most 1%%", share)
	}
}

func generatedWorld(t *testing.T, width, height int) (*domains.HeightField, *domains.ElevationLevels) {
	t.Helper()
	field, err := flat.GenerateHeightField(flat.Config{
		Source: rand.NewPCG(42, 0), Width: width, Height: height, Iterations: 10000, Wrap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	levels, err := percentile.Derive(field, percentile.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return field, levels
}

func testConfig(seed uint64) Config {
	cfg := DefaultConfig()
	cfg.Source = rand.NewPCG(seed, 1)
	return cfg
}

// testLevels puts sea level at 0.5 and the mountain level at 0.8, so every
// cone summit is 0.9.
func testLevels(t *testing.T) *domains.ElevationLevels {
	t.Helper()
	levels, err := domains.NewElevationLevels(domains.ElevationLevelValues{
		Abyss: 0.2, Shelf: 0.4, SeaLevel: 0.5, Upland: 0.6, Highland: 0.7, Mountain: 0.8,
	})
	if err != nil {
		t.Fatal(err)
	}
	return levels
}

func uniformField(t *testing.T, width, height int, elevation float64, topology domains.GridTopology) *domains.HeightField {
	t.Helper()
	elevations := make([]float64, width*height)
	for i := range elevations {
		elevations[i] = elevation
	}
	return fieldFrom(t, width, height, elevations, topology)
}

func fieldFrom(t *testing.T, width, height int, elevations []float64, topology domains.GridTopology) *domains.HeightField {
	t.Helper()
	heightMap, err := domains.NewNormalizedHeightMap(width, height, elevations)
	if err != nil {
		t.Fatal(err)
	}
	field, err := domains.NewHeightField(heightMap, topology)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

func allTopologies() []domains.GridTopology {
	return []domains.GridTopology{
		{},
		{WrapEastWest: true},
		{WrapNorthSouth: true},
		{WrapEastWest: true, WrapNorthSouth: true},
	}
}
