package monochromepng

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"testing"

	"github.com/maloquacious/mappe/domains"
)

func TestRenderMapsElevationsToPixelsWithoutTransposingCoordinates(t *testing.T) {
	heightMap, err := domains.NewNormalizedHeightMap(3, 2, []float64{
		0, 0.5, 1,
		0.25, 0.75, 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := Render(&output, heightMap); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := img.Bounds(), image.Rect(0, 0, 3, 2); got != want {
		t.Fatalf("bounds = %v, want %v", got, want)
	}

	want := [][]uint8{
		{0, 128, 255},
		{64, 191, 26},
	}
	for y := range want {
		for x := range want[y] {
			gray, _, _, _ := img.At(x, y).RGBA()
			if got := uint8(gray >> 8); got != want[y][x] {
				t.Errorf("pixel (%d, %d) = %d, want %d", x, y, got, want[y][x])
			}
		}
	}
}

func TestRenderRejectsNilInputs(t *testing.T) {
	heightMap, err := domains.NewNormalizedHeightMap(1, 1, []float64{0})
	if err != nil {
		t.Fatal(err)
	}

	if err := Render(nil, heightMap); err != ErrNilWriter {
		t.Fatalf("nil writer error = %v, want %v", err, ErrNilWriter)
	}
	if err := Render(&bytes.Buffer{}, nil); err != ErrNilHeightMap {
		t.Fatalf("nil height map error = %v, want %v", err, ErrNilHeightMap)
	}
}

func TestRenderReturnsWriterError(t *testing.T) {
	heightMap, err := domains.NewNormalizedHeightMap(1, 1, []float64{0})
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("write failed")

	if err := Render(errorWriter{err: want}, heightMap); !errors.Is(err, want) {
		t.Fatalf("Render error = %v, want error wrapping %v", err, want)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }
