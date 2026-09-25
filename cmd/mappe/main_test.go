package main

import (
	"bytes"
	"image/png"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/maloquacious/mappe"
	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/olsson"
	"github.com/maloquacious/mappe/pipelines/flatmonochromepng"
	"github.com/maloquacious/mappe/pipelines/olssonmonochromepng"
)

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	cmd := newCommand()
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), mappe.Version().Core()+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestFlatMonochromePNGCommandWritesSelectedArtifact(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "flat.png")
	cmd := newCommand()
	cmd.SetArgs([]string{
		"flat-monochrome-png",
		"--output", outputPath,
		"--seed", "42",
		"--width", "10",
		"--height", "6",
		"--iterations", "20",
		"--wrap",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	output, err := os.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	img, err := png.Decode(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Dx(); got != 10 {
		t.Fatalf("image width = %d, want 10", got)
	}
	if got := img.Bounds().Dy(); got != 6 {
		t.Fatalf("image height = %d, want 6", got)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if err := flatmonochromepng.Run(&want, flat.Config{
		Source:     rand.NewPCG(42, 0),
		Width:      10,
		Height:     6,
		Iterations: 20,
		Wrap:       true,
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatal("command flags did not produce the selected pipeline output")
	}
}

func TestFlatMonochromePNGCommandRequiresOutput(t *testing.T) {
	cmd := newCommand()
	cmd.SetArgs([]string{"flat-monochrome-png"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("command succeeded without --output")
	}
}

func TestOlssonMonochromePNGCommandWritesSelectedArtifact(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "world.png")
	cmd := newCommand()
	cmd.SetArgs([]string{
		"olsson-monochrome-png",
		"--output", outputPath,
		"--seed", "42",
		"--width", "10",
		"--height", "5",
		"--faults", "100",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	output, err := os.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	img, err := png.Decode(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Dx(); got != 10 {
		t.Fatalf("image width = %d, want 10", got)
	}
	if got := img.Bounds().Dy(); got != 5 {
		t.Fatalf("image height = %d, want 5", got)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if err := olssonmonochromepng.Run(&want, olsson.Config{
		Source: rand.NewPCG(42, 0),
		Width:  10,
		Height: 5,
		Faults: 100,
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatal("command flags did not produce the selected pipeline output")
	}
}

func TestOlssonMonochromePNGCommandRequiresOutput(t *testing.T) {
	cmd := newCommand()
	cmd.SetArgs([]string{"olsson-monochrome-png"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("command succeeded without --output")
	}
}
