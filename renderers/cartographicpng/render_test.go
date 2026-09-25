package cartographicpng

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/maloquacious/mappe/domains"
)

func TestRenderAssignsOceanLandAndIceWithoutTransposingCoordinates(t *testing.T) {
	heightMap, err := domains.NewNormalizedHeightMap(4, 2, []float64{
		0, 1.0 / 255, 2.0 / 255, 3.0 / 255,
		4.0 / 255, 5.0 / 255, 6.0 / 255, 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	oceanDark, oceanLight := rgba(0, 0, 10), rgba(0, 0, 20)
	landLow, landHigh := rgba(0, 10, 0), rgba(20, 10, 0)
	iceLow, iceHigh := rgba(200, 200, 200), rgba(255, 255, 255)
	cfg := Config{
		OceanPercent: 25,
		IcePercent:   25,
		OceanPalette: []color.RGBA{oceanDark, oceanLight},
		LandPalette:  []color.RGBA{landLow, landHigh},
		IcePalette:   []color.RGBA{iceLow, iceHigh},
	}

	var output bytes.Buffer
	if err := Render(&output, heightMap, cfg); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := img.Bounds(), image.Rect(0, 0, 4, 2); got != want {
		t.Fatalf("bounds = %v, want %v", got, want)
	}

	want := [][]color.RGBA{
		{oceanDark, oceanLight, landLow, landLow},
		{landHigh, landHigh, iceLow, iceHigh},
	}
	for y := range want {
		for x := range want[y] {
			if got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA); got != want[y][x] {
				t.Errorf("pixel (%d, %d) = %v, want %v", x, y, got, want[y][x])
			}
		}
	}
}

func TestBandAllocationPreservesWholeBinsAndSparseRanges(t *testing.T) {
	histogram := [256]int{}
	histogram[0] = 3
	histogram[10] = 1
	height := 0

	if got := consumeLevels(histogram, &height, 2); got != 1 {
		t.Fatalf("ocean levels = %d, want 1 after boundary-bin overshoot", got)
	}
	if got := consumeLevels(histogram, &height, 1); got != 10 {
		t.Fatalf("land levels = %d, want 10 including sparse bins", got)
	}
	if height != 11 {
		t.Fatalf("next height = %d, want 11", height)
	}
}

func TestColorMapSamplesPalettesDiscretely(t *testing.T) {
	histogram := [256]int{1, 1}
	palette := []color.RGBA{rgba(1, 0, 0), rgba(2, 0, 0), rgba(3, 0, 0), rgba(4, 0, 0)}
	colors := colorMap(histogram, Config{
		LandPalette:  palette,
		OceanPalette: []color.RGBA{rgba(0, 0, 1)},
		IcePalette:   []color.RGBA{rgba(9, 9, 9)},
	})

	if got, want := colors[0], palette[0]; got != want {
		t.Fatalf("first land color = %v, want %v", got, want)
	}
	if got, want := colors[1], palette[2]; got != want {
		t.Fatalf("second land color = %v, want skipped palette color %v", got, want)
	}
}

func TestConstantMapUsesWholeBinAndSmallTargetsTruncate(t *testing.T) {
	histogram := [256]int{2}
	cfg := DefaultConfig()
	colors := colorMap(histogram, cfg)
	if got, want := colors[0], cfg.OceanPalette[0]; got != want {
		t.Fatalf("two-pixel constant map color = %v, want ocean %v", got, want)
	}

	histogram[0] = 1
	colors = colorMap(histogram, cfg)
	if got, want := colors[0], cfg.IcePalette[0]; got != want {
		t.Fatalf("one-pixel constant map color = %v, want ice %v after percentage truncation", got, want)
	}
}

func TestHeightIndexTruncatesBoundaries(t *testing.T) {
	for _, tt := range []struct {
		elevation float64
		want      uint8
	}{
		{elevation: 0, want: 0},
		{elevation: 0.5, want: 127},
		{elevation: 128.0 / 255, want: 128},
		{elevation: 1, want: 255},
	} {
		if got := heightIndex(tt.elevation); got != tt.want {
			t.Errorf("heightIndex(%v) = %d, want %d", tt.elevation, got, tt.want)
		}
	}
}

func TestRenderRejectsInvalidInput(t *testing.T) {
	heightMap, err := domains.NewNormalizedHeightMap(1, 1, []float64{0})
	if err != nil {
		t.Fatal(err)
	}
	valid := DefaultConfig()
	tests := []struct {
		name string
		w    io.Writer
		m    *domains.NormalizedHeightMap
		cfg  Config
		want error
	}{
		{name: "nil writer", m: heightMap, cfg: valid, want: ErrNilWriter},
		{name: "nil map", w: &bytes.Buffer{}, cfg: valid, want: ErrNilHeightMap},
		{name: "negative ocean", w: &bytes.Buffer{}, m: heightMap, cfg: Config{OceanPercent: -1}, want: ErrInvalidOceanPercent},
		{name: "ocean above 100", w: &bytes.Buffer{}, m: heightMap, cfg: Config{OceanPercent: 101}, want: ErrInvalidOceanPercent},
		{name: "negative ice", w: &bytes.Buffer{}, m: heightMap, cfg: Config{IcePercent: -1}, want: ErrInvalidIcePercent},
		{name: "ice above 100", w: &bytes.Buffer{}, m: heightMap, cfg: Config{IcePercent: 101}, want: ErrInvalidIcePercent},
		{name: "total above 100", w: &bytes.Buffer{}, m: heightMap, cfg: Config{OceanPercent: 60, IcePercent: 41}, want: ErrInvalidTotalPercent},
		{name: "empty ocean", w: &bytes.Buffer{}, m: heightMap, cfg: Config{OceanPalette: []color.RGBA{}}, want: ErrEmptyOceanPalette},
		{name: "empty land", w: &bytes.Buffer{}, m: heightMap, cfg: Config{LandPalette: []color.RGBA{}}, want: ErrEmptyLandPalette},
		{name: "empty ice", w: &bytes.Buffer{}, m: heightMap, cfg: Config{IcePalette: []color.RGBA{}}, want: ErrEmptyIcePalette},
		{name: "transparent", w: &bytes.Buffer{}, m: heightMap, cfg: Config{LandPalette: []color.RGBA{{A: 254}}}, want: ErrTransparentPalette},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Render(tt.w, tt.m, tt.cfg); err != tt.want {
				t.Fatalf("Render error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRenderReturnsWriterError(t *testing.T) {
	heightMap, err := domains.NewNormalizedHeightMap(1, 1, []float64{0})
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("write failed")
	if err := Render(errorWriter{err: want}, heightMap, DefaultConfig()); !errors.Is(err, want) {
		t.Fatalf("Render error = %v, want error wrapping %v", err, want)
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func rgba(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
