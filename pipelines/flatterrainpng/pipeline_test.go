package flatterrainpng

import (
	"bytes"
	"errors"
	"image/png"
	"io"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/internal/levels/percentile"
	"github.com/maloquacious/mappe/internal/volcanism/sites"
	"github.com/maloquacious/mappe/renderers/terrainpng"
)

func TestRunProducesDeterministicTerrainPNG(t *testing.T) {
	var first, second bytes.Buffer
	if err := Run(&first, testConfig(42), percentile.DefaultConfig(), volcanismConfig(42), domains.DefaultClassificationConfig()); err != nil {
		t.Fatal(err)
	}
	if err := Run(&second, testConfig(42), percentile.DefaultConfig(), volcanismConfig(42), domains.DefaultClassificationConfig()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("same configuration produced different PNG output")
	}
	img, err := png.Decode(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 16 {
		t.Fatalf("image size = %dx%d, want 32x16", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestDiagnosticFieldsAreNormalizedAndUseHeightTopology(t *testing.T) {
	heightField, err := flat.GenerateHeightField(testConfig(42))
	if err != nil {
		t.Fatal(err)
	}
	levels, err := percentile.Derive(heightField, percentile.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	environmentalMap, err := classify(heightField, levels, noVolcanism(t, heightField), domains.DefaultClassificationConfig())
	if err != nil {
		t.Fatal(err)
	}
	if environmentalMap.Topology() != heightField.Topology() {
		t.Fatalf("environment topology = %+v, want %+v", environmentalMap.Topology(), heightField.Topology())
	}
	seen := map[domains.Terrain]bool{}
	for _, cell := range environmentalMap.Cells() {
		seen[cell.Terrain] = true
		for _, value := range []float64{
			cell.Sample.Elevation, cell.Sample.Heat, cell.Sample.Moisture,
			cell.Sample.Relief, cell.Sample.Basin,
		} {
			if value < 0 || value > 1 {
				t.Fatalf("sample value %v outside [0, 1]", value)
			}
		}
	}
	if len(seen) < 6 {
		t.Fatalf("diagnostic map produced %d terrains, want at least 6", len(seen))
	}
}

func TestDiagnosticFieldsAreContinuousAcrossWrappedSeams(t *testing.T) {
	const width, height = 128, 64
	heightMap, err := domains.NewNormalizedHeightMap(width, height, make([]float64, width*height))
	if err != nil {
		t.Fatal(err)
	}
	heightField, err := domains.NewHeightField(heightMap, domains.GridTopology{WrapEastWest: true, WrapNorthSouth: true})
	if err != nil {
		t.Fatal(err)
	}
	levels, err := domains.NewElevationLevels(domains.ElevationLevelValues{})
	if err != nil {
		t.Fatal(err)
	}
	environmentalMap, err := classify(heightField, levels, noVolcanism(t, heightField), domains.DefaultClassificationConfig())
	if err != nil {
		t.Fatal(err)
	}

	for y := 0; y < height; y++ {
		assertNearbySamples(t, environmentalMap.Cell(0, y).Sample, environmentalMap.Cell(width-1, y).Sample)
	}
	for x := 0; x < width; x++ {
		assertNearbySamples(t, environmentalMap.Cell(x, 0).Sample, environmentalMap.Cell(x, height-1).Sample)
	}
}

func TestClassifiedWaterMatchesRequestedOceanShare(t *testing.T) {
	heightField, err := flat.GenerateHeightField(flat.Config{
		Source: rand.NewPCG(42, 0), Width: 128, Height: 64, Iterations: 1000, Wrap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, oceanPercent := range []int{25, 48, 70} {
		levelsConfig := percentile.DefaultConfig()
		levelsConfig.OceanPercent = oceanPercent
		levels, err := percentile.Derive(heightField, levelsConfig)
		if err != nil {
			t.Fatal(err)
		}
		environmentalMap, err := classify(heightField, levels, noVolcanism(t, heightField), domains.DefaultClassificationConfig())
		if err != nil {
			t.Fatal(err)
		}
		water := 0
		for _, cell := range environmentalMap.Cells() {
			if cell.Terrain.IsWater() {
				water++
			}
		}
		composition := levels.Composition(heightField)
		if water != composition.WaterTiles() {
			t.Fatalf("ocean %d%%: water terrain tiles = %d, want %d", oceanPercent, water, composition.WaterTiles())
		}
		// The requested share is a minimum, and only tiles tied at sea level
		// may carry the achieved share above it.
		requested := (composition.Tiles*oceanPercent + 99) / 100
		atSeaLevel := 0
		for _, elevation := range heightField.Elevations() {
			if elevation == levels.Values().SeaLevel {
				atSeaLevel++
			}
		}
		if water < requested || water-atSeaLevel >= requested {
			t.Fatalf("ocean %d%%: water tiles = %d with %d at sea level, want the fewest tied tiles reaching %d", oceanPercent, water, atSeaLevel, requested)
		}
	}
}

func TestVolcanoSitesBecomeVolcanoTerrain(t *testing.T) {
	heightField, err := flat.GenerateHeightField(flat.Config{
		Source: rand.NewPCG(42, 0), Width: 128, Height: 64, Iterations: 1000, Wrap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	levels, err := percentile.Derive(heightField, percentile.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	cfg := volcanismConfig(42)
	cfg.LandTilesPerSite = 200
	volcanism, err := sites.Place(heightField, levels, cfg)
	if err != nil {
		t.Fatal(err)
	}
	environmentalMap, err := classify(volcanism.HeightField, levels, volcanism.Features, domains.DefaultClassificationConfig())
	if err != nil {
		t.Fatal(err)
	}
	sitesPlaced := volcanism.Features.Sites()
	if len(sitesPlaced) == 0 {
		t.Fatal("no volcano sites placed")
	}
	for _, site := range sitesPlaced {
		if got := environmentalMap.Cell(site.X, site.Y).Terrain; got != domains.TerrainVolcano {
			t.Errorf("site %+v terrain = %v, want volcano", site, got)
		}
	}
	volcanoes := 0
	for _, cell := range environmentalMap.Cells() {
		if cell.Terrain == domains.TerrainVolcano {
			volcanoes++
		}
	}
	if volcanoes != len(sitesPlaced) {
		t.Fatalf("volcano tiles = %d, want one per site (%d)", volcanoes, len(sitesPlaced))
	}
}

func TestRiseAboveSeaIsMeasuredFromDerivedSeaLevel(t *testing.T) {
	for _, tt := range []struct {
		elevation, seaLevel, mountain, want float64
	}{
		{elevation: 0.2, seaLevel: 0.3, mountain: 0.8, want: 0},
		{elevation: 0.3, seaLevel: 0.3, mountain: 0.8, want: 0},
		{elevation: 0.55, seaLevel: 0.3, mountain: 0.8, want: 0.5},
		{elevation: 0.8, seaLevel: 0.3, mountain: 0.8, want: 1},
		{elevation: 1, seaLevel: 0.3, mountain: 0.8, want: 1.4},
	} {
		if got := riseAboveSea(tt.elevation, tt.seaLevel, tt.mountain); math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("riseAboveSea(%v, %v, %v) = %v, want %v", tt.elevation, tt.seaLevel, tt.mountain, got, tt.want)
		}
	}
}

func TestRunIdentifiesFailingStage(t *testing.T) {
	if err := Run(&bytes.Buffer{}, flat.Config{}, percentile.DefaultConfig(), volcanismConfig(42), domains.DefaultClassificationConfig()); !errors.Is(err, flat.ErrNilSource) {
		t.Fatalf("generator error = %v, want error wrapping %v", err, flat.ErrNilSource)
	}
	badLevels := percentile.DefaultConfig()
	badLevels.OceanPercent = 101
	if err := Run(&bytes.Buffer{}, testConfig(42), badLevels, volcanismConfig(42), domains.DefaultClassificationConfig()); !errors.Is(err, percentile.ErrInvalidOceanPercent) {
		t.Fatalf("levels error = %v, want error wrapping %v", err, percentile.ErrInvalidOceanPercent)
	}
	badVolcanism := volcanismConfig(42)
	badVolcanism.Source = nil
	if err := Run(&bytes.Buffer{}, testConfig(42), percentile.DefaultConfig(), badVolcanism, domains.DefaultClassificationConfig()); !errors.Is(err, sites.ErrNilSource) {
		t.Fatalf("volcanism error = %v, want error wrapping %v", err, sites.ErrNilSource)
	}
	badConfig := domains.DefaultClassificationConfig()
	badConfig.Heat.Cold = badConfig.Heat.Polar
	if err := Run(&bytes.Buffer{}, testConfig(42), percentile.DefaultConfig(), volcanismConfig(42), badConfig); !errors.Is(err, domains.ErrInvalidClassificationConfig) {
		t.Fatalf("classification error = %v, want error wrapping %v", err, domains.ErrInvalidClassificationConfig)
	}
	want := errors.New("write failed")
	if err := Run(errorWriter{err: want}, testConfig(42), percentile.DefaultConfig(), volcanismConfig(42), domains.DefaultClassificationConfig()); !errors.Is(err, want) {
		t.Fatalf("renderer error = %v, want error wrapping %v", err, want)
	}
	if err := Run(nil, testConfig(42), percentile.DefaultConfig(), volcanismConfig(42), domains.DefaultClassificationConfig()); !errors.Is(err, terrainpng.ErrNilWriter) {
		t.Fatalf("nil writer error = %v, want error wrapping %v", err, terrainpng.ErrNilWriter)
	}
}

func volcanismConfig(seed uint64) sites.Config {
	cfg := sites.DefaultConfig()
	cfg.Source = rand.NewPCG(seed, 1)
	return cfg
}

func noVolcanism(t *testing.T, heightField *domains.HeightField) *domains.VolcanicFeatures {
	t.Helper()
	features, err := domains.NewVolcanicFeatures(heightField.Width(), heightField.Height(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return features
}

func testConfig(seed uint64) flat.Config {
	return flat.Config{
		Source: rand.NewPCG(seed, 0), Width: 32, Height: 16, Iterations: 100, Wrap: true,
	}
}

func assertNearbySamples(t *testing.T, a, b domains.EnvironmentalSample) {
	t.Helper()
	for _, pair := range [][2]float64{
		{a.Heat, b.Heat}, {a.Moisture, b.Moisture},
		{a.Basin, b.Basin},
	} {
		if difference := max(pair[0], pair[1]) - min(pair[0], pair[1]); difference > 0.12 {
			t.Fatalf("wrapped seam difference = %v between %v and %v", difference, pair[0], pair[1])
		}
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

var _ io.Writer = errorWriter{}
