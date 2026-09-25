package flatcartographicpng

import (
	"bytes"
	"errors"
	"image/png"
	"math/rand/v2"
	"testing"

	"github.com/maloquacious/mappe/internal/generators/flat"
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
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 6 {
		t.Fatalf("image size = %dx%d, want 10x6", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestRunIdentifiesFailingStage(t *testing.T) {
	if err := Run(&bytes.Buffer{}, flat.Config{}, cartographicpng.DefaultConfig()); !errors.Is(err, flat.ErrNilSource) {
		t.Fatalf("generator error = %v, want error wrapping %v", err, flat.ErrNilSource)
	}
	if err := Run(nil, testConfig(42), cartographicpng.DefaultConfig()); !errors.Is(err, cartographicpng.ErrNilWriter) {
		t.Fatalf("renderer error = %v, want error wrapping %v", err, cartographicpng.ErrNilWriter)
	}
}

func testConfig(seed uint64) flat.Config {
	return flat.Config{Source: rand.NewPCG(seed, 0), Width: 10, Height: 6, Iterations: 20, Wrap: true}
}
