package domains

import (
	"math"
	"testing"
)

func TestNewElevationLevelsValidatesOrderAndRange(t *testing.T) {
	valid := testLevelValues()
	for _, tt := range []struct {
		name   string
		change func(*ElevationLevelValues)
		want   error
	}{
		{name: "strictly ordered", change: func(*ElevationLevelValues) {}},
		{name: "equal adjacent levels", change: func(v *ElevationLevelValues) { v.Shelf, v.Upland = v.SeaLevel, v.SeaLevel }},
		{name: "every level zero", change: func(v *ElevationLevelValues) { *v = ElevationLevelValues{} }},
		{name: "every level one", change: func(v *ElevationLevelValues) {
			*v = ElevationLevelValues{Abyss: 1, Shelf: 1, SeaLevel: 1, Upland: 1, Highland: 1, Mountain: 1}
		}},
		{name: "shelf above sea level", change: func(v *ElevationLevelValues) { v.Shelf = math.Nextafter(v.SeaLevel, 1) }, want: ErrInvalidElevationLevels},
		{name: "abyss above shelf", change: func(v *ElevationLevelValues) { v.Abyss = math.Nextafter(v.Shelf, 1) }, want: ErrInvalidElevationLevels},
		{name: "upland below sea level", change: func(v *ElevationLevelValues) { v.Upland = math.Nextafter(v.SeaLevel, 0) }, want: ErrInvalidElevationLevels},
		{name: "mountain below highland", change: func(v *ElevationLevelValues) { v.Mountain = math.Nextafter(v.Highland, 0) }, want: ErrInvalidElevationLevels},
		{name: "negative", change: func(v *ElevationLevelValues) { v.Abyss = -0.1 }, want: ErrInvalidElevationLevels},
		{name: "above one", change: func(v *ElevationLevelValues) { v.Mountain = 1.1 }, want: ErrInvalidElevationLevels},
		{name: "NaN", change: func(v *ElevationLevelValues) { v.Highland = math.NaN() }, want: ErrInvalidElevationLevels},
	} {
		t.Run(tt.name, func(t *testing.T) {
			values := valid
			tt.change(&values)
			levels, err := NewElevationLevels(values)
			if err != tt.want {
				t.Fatalf("NewElevationLevels error = %v, want %v", err, tt.want)
			}
			if err == nil && levels.Values() != values {
				t.Fatalf("Values = %+v, want %+v", levels.Values(), values)
			}
		})
	}
}

func TestElevationLevelsClassifyAtEveryBoundary(t *testing.T) {
	levels := testLevels(t)
	v := levels.Values()
	for _, tt := range []struct {
		name  string
		at    float64
		above float64
		want  string
	}{
		{name: "shelf", at: v.Shelf, above: math.Nextafter(v.Shelf, 1), want: "deep-water/shallow-water"},
		{name: "sea level", at: v.SeaLevel, above: math.Nextafter(v.SeaLevel, 1), want: "shallow-water/lowland"},
		{name: "upland", at: math.Nextafter(v.Upland, 0), above: v.Upland, want: "lowland/upland"},
		{name: "highland", at: math.Nextafter(v.Highland, 0), above: v.Highland, want: "upland/highland"},
		{name: "mountain", at: math.Nextafter(v.Mountain, 0), above: v.Mountain, want: "highland/mountain"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := levels.Classify(tt.at).String() + "/" + levels.Classify(tt.above).String(); got != tt.want {
				t.Fatalf("boundary classifications = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestElevationLevelsWithEqualNeighborsLeaveBandEmpty(t *testing.T) {
	values := testLevelValues()
	values.Upland = values.Highland
	levels, err := NewElevationLevels(values)
	if err != nil {
		t.Fatal(err)
	}
	if got := levels.Classify(values.Highland); got != ElevationHighland {
		t.Fatalf("Classify(highland level) = %v, want highland", got)
	}
	if got := levels.Classify(math.Nextafter(values.Highland, 0)); got != ElevationLowland {
		t.Fatalf("Classify(just below merged level) = %v, want lowland", got)
	}
}

func TestElevationLevelsCompositionCountsEveryZone(t *testing.T) {
	levels := testLevels(t)
	v := levels.Values()
	elevations := []float64{
		0, v.Abyss, // deep ocean
		math.Nextafter(v.Abyss, 1), v.Shelf, 0.4, // ocean
		v.SeaLevel,                    // shallow sea
		math.Nextafter(v.SeaLevel, 1), // lowland
		v.Upland, 0.7,                 // upland
		v.Highland,    // highland
		v.Mountain, 1, // mountain
	}
	heightMap, err := NewNormalizedHeightMap(len(elevations), 1, elevations)
	if err != nil {
		t.Fatal(err)
	}
	field, err := NewHeightField(heightMap, GridTopology{})
	if err != nil {
		t.Fatal(err)
	}
	want := ElevationComposition{
		Tiles: 12, DeepOcean: 2, Ocean: 3, ShallowSea: 1,
		Lowland: 1, Upland: 2, Highland: 1, Mountain: 2,
	}
	got := levels.Composition(field)
	if got != want {
		t.Fatalf("Composition = %+v, want %+v", got, want)
	}
	if got.WaterTiles() != 6 || got.LandTiles() != 6 {
		t.Fatalf("water/land tiles = %d/%d, want 6/6", got.WaterTiles(), got.LandTiles())
	}
}

// testLevelValues mirrors the fixed thresholds used before elevation levels
// were derived, so classification tests keep their original sample values.
func testLevelValues() ElevationLevelValues {
	return ElevationLevelValues{
		Abyss:    0.325,
		Shelf:    0.45,
		SeaLevel: 0.5,
		Upland:   0.625,
		Highland: 0.75,
		Mountain: 0.875,
	}
}

func testLevels(t *testing.T) *ElevationLevels {
	t.Helper()
	levels, err := NewElevationLevels(testLevelValues())
	if err != nil {
		t.Fatal(err)
	}
	return levels
}
