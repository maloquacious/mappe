package main

import (
	"bytes"
	"image/png"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/maloquacious/mappe"
	"github.com/maloquacious/mappe/domains"
	"github.com/maloquacious/mappe/internal/generators/flat"
	"github.com/maloquacious/mappe/internal/levels/percentile"
	"github.com/maloquacious/mappe/internal/volcanism/sites"
	"github.com/maloquacious/mappe/olsson"
	"github.com/maloquacious/mappe/pipelines/flatcartographicpng"
	"github.com/maloquacious/mappe/pipelines/flatmonochromepng"
	"github.com/maloquacious/mappe/pipelines/flatterrainpng"
	"github.com/maloquacious/mappe/pipelines/olssoncartographicpng"
	"github.com/maloquacious/mappe/pipelines/olssonmonochromepng"
	"github.com/maloquacious/mappe/renderers/cartographicpng"
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

func TestFlatCartographicPNGCommandWritesSelectedArtifact(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "flat-color.png")
	cmd := newCommand()
	cmd.SetArgs([]string{
		"flat-cartographic-png",
		"--output", outputPath,
		"--seed", "42",
		"--width", "10",
		"--height", "6",
		"--iterations", "20",
		"--wrap",
		"--ocean-percent", "25",
		"--ice-percent", "15",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 6 {
		t.Fatalf("image size = %dx%d, want 10x6", img.Bounds().Dx(), img.Bounds().Dy())
	}

	var want bytes.Buffer
	rendererConfig := cartographicpng.DefaultConfig()
	rendererConfig.OceanPercent = 25
	rendererConfig.IcePercent = 15
	if err := flatcartographicpng.Run(&want, flat.Config{
		Source:     rand.NewPCG(42, 0),
		Width:      10,
		Height:     6,
		Iterations: 20,
		Wrap:       true,
	}, rendererConfig); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatal("command flags did not produce the selected pipeline output")
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

func TestFlatTerrainPNGCommandWritesSelectedArtifact(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "flat-terrain.png")
	cmd := newCommand()
	cmd.SetArgs([]string{
		"flat-terrain-png",
		"--output", outputPath,
		"--seed", "42",
		"--width", "32",
		"--height", "16",
		"--iterations", "100",
		"--wrap",
		"--ocean-percent", "30",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 16 {
		t.Fatalf("image size = %dx%d, want 32x16", img.Bounds().Dx(), img.Bounds().Dy())
	}

	var want bytes.Buffer
	levelsConfig := percentile.DefaultConfig()
	levelsConfig.OceanPercent = 30
	volcanismConfig := sites.DefaultConfig()
	volcanismConfig.Source = rand.NewPCG(42, 1)
	if err := flatterrainpng.Run(&want, flat.Config{
		Source: rand.NewPCG(42, 0), Width: 32, Height: 16, Iterations: 100, Wrap: true,
	}, levelsConfig, volcanismConfig, domains.DefaultClassificationConfig()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatal("command flags did not produce the selected pipeline output")
	}
}

func TestOceanPercentDefaultsTo48InEveryPipeline(t *testing.T) {
	for _, name := range []string{"flat-cartographic-png", "flat-terrain-png", "olsson-cartographic-png"} {
		cmd, _, err := newCommand().Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		flag := cmd.Flags().Lookup("ocean-percent")
		if flag == nil {
			t.Fatalf("%s has no --ocean-percent flag", name)
		}
		if flag.DefValue != "48" {
			t.Errorf("%s --ocean-percent default = %s, want 48", name, flag.DefValue)
		}
	}
}

func TestFlatTerrainPNGCommandRequiresOutput(t *testing.T) {
	cmd := newCommand()
	cmd.SetArgs([]string{"flat-terrain-png"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("command succeeded without --output")
	}
}

func TestOlssonCartographicPNGCommandWritesSelectedArtifact(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "world-color.png")
	cmd := newCommand()
	cmd.SetArgs([]string{
		"olsson-cartographic-png",
		"--output", outputPath,
		"--seed", "42",
		"--width", "10",
		"--height", "5",
		"--faults", "100",
		"--ocean-percent", "25",
		"--ice-percent", "15",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 5 {
		t.Fatalf("image size = %dx%d, want 10x5", img.Bounds().Dx(), img.Bounds().Dy())
	}

	var want bytes.Buffer
	rendererConfig := cartographicpng.DefaultConfig()
	rendererConfig.OceanPercent = 25
	rendererConfig.IcePercent = 15
	if err := olssoncartographicpng.Run(&want, olsson.Config{
		Source: rand.NewPCG(42, 0),
		Width:  10,
		Height: 5,
		Faults: 100,
	}, rendererConfig); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatal("command flags did not produce the selected pipeline output")
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

func TestCartographicPNGCommandsRequireOutput(t *testing.T) {
	for _, name := range []string{"flat-cartographic-png", "olsson-cartographic-png"} {
		t.Run(name, func(t *testing.T) {
			cmd := newCommand()
			cmd.SetArgs([]string{name})
			if err := cmd.Execute(); err == nil {
				t.Fatal("command succeeded without --output")
			}
		})
	}
}
