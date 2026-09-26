package flatterrainpng

import (
	"bytes"
	"errors"
	"image/png"
	"io"
	"math/rand/v2"
	"testing"

	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/renderers/terrainpng"
)

func TestRunProducesDeterministicTerrainPNG(t *testing.T) {
	var first, second bytes.Buffer
	if err := Run(&first, testConfig(42), domains.DefaultClassificationConfig()); err != nil {
		t.Fatal(err)
	}
	if err := Run(&second, testConfig(42), domains.DefaultClassificationConfig()); err != nil {
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
	environmentalMap, err := classify(heightField, domains.DefaultClassificationConfig())
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
			cell.Sample.Relief, cell.Sample.Basin, cell.Sample.Volcanic,
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
	environmentalMap, err := classify(heightField, domains.DefaultClassificationConfig())
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

func TestRunIdentifiesFailingStage(t *testing.T) {
	if err := Run(&bytes.Buffer{}, flat.Config{}, domains.DefaultClassificationConfig()); !errors.Is(err, flat.ErrNilSource) {
		t.Fatalf("generator error = %v, want error wrapping %v", err, flat.ErrNilSource)
	}
	badConfig := domains.DefaultClassificationConfig()
	badConfig.Heat.Cold = badConfig.Heat.Polar
	if err := Run(&bytes.Buffer{}, testConfig(42), badConfig); !errors.Is(err, domains.ErrInvalidClassificationConfig) {
		t.Fatalf("classification error = %v, want error wrapping %v", err, domains.ErrInvalidClassificationConfig)
	}
	want := errors.New("write failed")
	if err := Run(errorWriter{err: want}, testConfig(42), domains.DefaultClassificationConfig()); !errors.Is(err, want) {
		t.Fatalf("renderer error = %v, want error wrapping %v", err, want)
	}
	if err := Run(nil, testConfig(42), domains.DefaultClassificationConfig()); !errors.Is(err, terrainpng.ErrNilWriter) {
		t.Fatalf("nil writer error = %v, want error wrapping %v", err, terrainpng.ErrNilWriter)
	}
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
		{a.Basin, b.Basin}, {a.Volcanic, b.Volcanic},
	} {
		if difference := max(pair[0], pair[1]) - min(pair[0], pair[1]); difference > 0.12 {
			t.Fatalf("wrapped seam difference = %v between %v and %v", difference, pair[0], pair[1])
		}
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

var _ io.Writer = errorWriter{}
