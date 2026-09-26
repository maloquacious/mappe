package percentile

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/generators/flat"
)

func TestDeriveMeetsEveryRequestedShareOnDistinctElevations(t *testing.T) {
	elevations := make([]float64, 100)
	for i := range elevations {
		elevations[i] = float64(i) / 99
	}
	field := testField(t, elevations)

	levels, err := Derive(field, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	// 48 ocean and 52 land tiles. Each share rounds up to whole tiles:
	// shelf ceil(7.2) = 8, abyss ceil(28.8) = 29, upland ceil(26) = 26,
	// highland ceil(41.6) = 42, and mountain ceil(48.36) = 49.
	want := domains.ElevationComposition{
		Tiles: 100, DeepOcean: 19, Ocean: 21, ShallowSea: 8,
		Lowland: 26, Upland: 16, Highland: 7, Mountain: 3,
	}
	if got := levels.Composition(field); got != want {
		t.Fatalf("Composition = %+v, want %+v", got, want)
	}
	if got := levels.Values().SeaLevel; got != elevations[47] {
		t.Fatalf("sea level = %v, want highest ocean tile %v", got, elevations[47])
	}
}

func TestDeriveKeepsTiedElevationsTogether(t *testing.T) {
	field := testField(t, []float64{0, 0, 0, 0.2, 0.2, 0.2, 0.2, 0.6, 0.8, 1})
	cfg := DefaultConfig()
	cfg.OceanPercent = 40

	levels, err := Derive(field, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := levels.Values().SeaLevel; got != 0.2 {
		t.Fatalf("sea level = %v, want 0.2", got)
	}
	if got := levels.Composition(field).WaterTiles(); got != 7 {
		t.Fatalf("water tiles = %d, want all 7 tiles at or below the tied level", got)
	}
}

func TestDeriveHandlesOceanExtremes(t *testing.T) {
	for _, tt := range []struct {
		name       string
		elevations []float64
		percent    int
		wantWater  int
	}{
		{name: "no ocean above zero", elevations: []float64{0.3, 0.5, 0.7, 1}, percent: 0, wantWater: 0},
		{name: "no ocean keeps zero-elevation tiles wet", elevations: []float64{0, 0, 0.5, 1}, percent: 0, wantWater: 2},
		{name: "all ocean", elevations: []float64{0, 0.5, 0.7, 1}, percent: 100, wantWater: 4},
		{name: "uniform", elevations: []float64{0.4, 0.4, 0.4, 0.4}, percent: 48, wantWater: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			field := testField(t, tt.elevations)
			cfg := DefaultConfig()
			cfg.OceanPercent = tt.percent
			levels, err := Derive(field, cfg)
			if err != nil {
				t.Fatal(err)
			}
			composition := levels.Composition(field)
			if composition.WaterTiles() != tt.wantWater {
				t.Fatalf("water tiles = %d, want %d (%+v)", composition.WaterTiles(), tt.wantWater, levels.Values())
			}
			v := levels.Values()
			if composition.LandTiles() == 0 && (v.Upland != v.SeaLevel || v.Mountain != v.SeaLevel) {
				t.Fatalf("land levels without land = %+v, want sea level", v)
			}
			if composition.WaterTiles() == 0 && (v.Shelf != v.SeaLevel || v.Abyss != v.SeaLevel) {
				t.Fatalf("depth levels without ocean = %+v, want sea level", v)
			}
		})
	}
}

func TestDeriveMergesLevelsThatTiesCannotSeparate(t *testing.T) {
	field := testField(t, []float64{0, 0.1, 0.2, 0.3, 0.9, 0.9, 0.9, 0.9})
	cfg := DefaultConfig()
	cfg.OceanPercent = 50

	levels, err := Derive(field, cfg)
	if err != nil {
		t.Fatal(err)
	}
	v := levels.Values()
	if v.Upland != v.Highland || v.Highland != v.Mountain || v.Upland <= 0.9 {
		t.Fatalf("land levels = %+v, want one merged level above the tied land", v)
	}
	if got := levels.Composition(field).Lowland; got != 4 {
		t.Fatalf("lowland tiles = %d, want every tied land tile", got)
	}
}

func TestDeriveProducesPopulatedOrderedBandsForGeneratedTerrain(t *testing.T) {
	field, err := flat.GenerateHeightField(flat.Config{
		Source: rand.NewPCG(42, 0), Width: 128, Height: 64, Iterations: 1000, Wrap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := field.Elevations()
	levels, err := Derive(field, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(field.Elevations(), before) {
		t.Fatal("Derive changed the height field")
	}

	c := levels.Composition(field)
	for name, count := range map[string]int{
		"deep ocean": c.DeepOcean, "ocean": c.Ocean, "shallow sea": c.ShallowSea,
		"lowland": c.Lowland, "upland": c.Upland, "highland": c.Highland, "mountain": c.Mountain,
	} {
		if count == 0 {
			t.Errorf("%s band is empty: %+v", name, c)
		}
	}
	if requested := share(c.Tiles, 48); c.WaterTiles() < requested {
		t.Fatalf("water tiles = %d, want at least %d", c.WaterTiles(), requested)
	}
	if mountains := float64(c.Mountain) / float64(c.LandTiles()); math.Abs(mountains-0.07) > 0.03 {
		t.Fatalf("mountain share of land = %.3f, want about 0.07", mountains)
	}
}

func TestDeriveRejectsInvalidInput(t *testing.T) {
	field := testField(t, []float64{0, 1})
	for _, tt := range []struct {
		name   string
		change func(*Config)
		want   error
	}{
		{name: "negative ocean", change: func(c *Config) { c.OceanPercent = -1 }, want: ErrInvalidOceanPercent},
		{name: "ocean above 100", change: func(c *Config) { c.OceanPercent = 101 }, want: ErrInvalidOceanPercent},
		{name: "equal land percents", change: func(c *Config) { c.HighlandPercent = c.UplandPercent }, want: ErrInvalidLandPercents},
		{name: "mountain above 100", change: func(c *Config) { c.MountainPercent = 101 }, want: ErrInvalidLandPercents},
		{name: "negative upland", change: func(c *Config) { c.UplandPercent = -1 }, want: ErrInvalidLandPercents},
		{name: "abyss not deeper than shelf", change: func(c *Config) { c.AbyssPercent = c.ShelfPercent }, want: ErrInvalidDepthPercents},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.change(&cfg)
			if _, err := Derive(field, cfg); err != tt.want {
				t.Fatalf("Derive error = %v, want %v", err, tt.want)
			}
		})
	}
	if _, err := Derive(nil, DefaultConfig()); err != ErrNilHeightField {
		t.Fatalf("Derive(nil) error = %v, want %v", err, ErrNilHeightField)
	}
}

func testField(t *testing.T, elevations []float64) *domains.HeightField {
	t.Helper()
	heightMap, err := domains.NewNormalizedHeightMap(len(elevations), 1, elevations)
	if err != nil {
		t.Fatal(err)
	}
	field, err := domains.NewHeightField(heightMap, domains.GridTopology{})
	if err != nil {
		t.Fatal(err)
	}
	return field
}
