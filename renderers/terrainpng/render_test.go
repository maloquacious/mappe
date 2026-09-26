package terrainpng

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"

	"github.com/maloquacious/mappe/domains"
)

func TestRenderMapsTerrainToPixelsWithoutTransposingCoordinates(t *testing.T) {
	samples := []domains.EnvironmentalSample{
		sample(0.1, 0.5, 0.5), sample(0.55, 0.5, 0.5),
		sample(0.55, 0.9, 0.1), sample(0.9, 0.5, 0.5),
	}
	environmentalMap, err := domains.NewEnvironmentalMap(2, 2, domains.GridTopology{}, samples, domains.DefaultClassificationConfig())
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := Render(&output, environmentalMap); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			want := terrainColor(environmentalMap.Cell(x, y).Terrain)
			got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if got != want {
				t.Errorf("pixel (%d, %d) = %v, want %v for %v", x, y, got, want, environmentalMap.Cell(x, y).Terrain)
			}
		}
	}
}

func TestPaletteCoversEveryTerrainWithOpaqueColors(t *testing.T) {
	seen := map[color.RGBA]domains.Terrain{}
	for _, terrain := range domains.Terrains() {
		got := terrainColor(terrain)
		if got.A != 255 {
			t.Errorf("%v color = %v, want opaque", terrain, got)
		}
		if other, exists := seen[got]; exists {
			t.Errorf("%v and %v share color %v", other, terrain, got)
		}
		seen[got] = terrain
	}
}

func TestRenderRejectsNilInputs(t *testing.T) {
	environmentalMap, err := domains.NewEnvironmentalMap(1, 1, domains.GridTopology{}, []domains.EnvironmentalSample{sample(0.55, 0.5, 0.5)}, domains.DefaultClassificationConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := Render(nil, environmentalMap); err != ErrNilWriter {
		t.Fatalf("nil writer error = %v, want %v", err, ErrNilWriter)
	}
	if err := Render(&bytes.Buffer{}, nil); err != ErrNilEnvironmentalMap {
		t.Fatalf("nil map error = %v, want %v", err, ErrNilEnvironmentalMap)
	}
}

func sample(elevation, heat, moisture float64) domains.EnvironmentalSample {
	return domains.EnvironmentalSample{
		Elevation: elevation,
		Heat:      heat,
		Moisture:  moisture,
		Relief:    0.1,
		Basin:     0.5,
		Volcanic:  0.5,
	}
}
