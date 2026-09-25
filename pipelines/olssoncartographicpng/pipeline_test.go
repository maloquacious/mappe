package olssoncartographicpng

import (
	"bytes"
	"errors"
	"image/png"
	"math/rand/v2"
	"testing"

	"github.com/maloquacious/mappe/olsson"
	"github.com/maloquacious/mappe/renderers/cartographicpng"
)

func TestRunProducesDeterministicPNG(t *testing.T) {
	var first, second bytes.Buffer
	rendererConfig := cartographicpng.DefaultConfig()
	if err := Run(&first, testConfig(42), rendererConfig); err != nil {
		t.Fatal(err)
	}
	if err := Run(&second, testConfig(42), rendererConfig); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("same configuration produced different PNG output")
	}
	img, err := png.Decode(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 5 {
		t.Fatalf("image size = %dx%d, want 10x5", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestRunIdentifiesFailingStage(t *testing.T) {
	if err := Run(&bytes.Buffer{}, olsson.Config{}, cartographicpng.DefaultConfig()); !errors.Is(err, olsson.ErrNilSource) {
		t.Fatalf("generator error = %v, want error wrapping %v", err, olsson.ErrNilSource)
	}
	if err := Run(nil, testConfig(42), cartographicpng.DefaultConfig()); !errors.Is(err, cartographicpng.ErrNilWriter) {
		t.Fatalf("renderer error = %v, want error wrapping %v", err, cartographicpng.ErrNilWriter)
	}
}

func testConfig(seed uint64) olsson.Config {
	return olsson.Config{Source: rand.NewPCG(seed, 0), Width: 10, Height: 5, Faults: 100}
}
