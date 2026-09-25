package flatmonochromepng

import (
	"bytes"
	"errors"
	"image/png"
	"math/rand/v2"
	"testing"

	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/renderers/monochromepng"
)

func TestRunProducesDeterministicPNG(t *testing.T) {
	var first, second bytes.Buffer
	if err := Run(&first, testConfig(42)); err != nil {
		t.Fatal(err)
	}
	if err := Run(&second, testConfig(42)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("same configuration produced different PNG output")
	}

	img, err := png.Decode(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Dx(); got != 10 {
		t.Fatalf("image width = %d, want 10", got)
	}
	if got := img.Bounds().Dy(); got != 6 {
		t.Fatalf("image height = %d, want 6", got)
	}
}

func TestRunIdentifiesFailingStage(t *testing.T) {
	if err := Run(&bytes.Buffer{}, flat.Config{}); !errors.Is(err, flat.ErrNilSource) {
		t.Fatalf("generator error = %v, want error wrapping %v", err, flat.ErrNilSource)
	}
	if err := Run(nil, testConfig(42)); !errors.Is(err, monochromepng.ErrNilWriter) {
		t.Fatalf("renderer error = %v, want error wrapping %v", err, monochromepng.ErrNilWriter)
	}
}

func testConfig(seed uint64) flat.Config {
	return flat.Config{
		Source:     rand.NewPCG(seed, 0),
		Width:      10,
		Height:     6,
		Iterations: 20,
		Wrap:       true,
	}
}
