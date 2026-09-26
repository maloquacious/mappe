// Mappe - consolidated map and world generators
// Copyright (c) 2026 Michael D Henderson
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"

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
	"github.com/spf13/cobra"
)

func main() {
	cmd := newCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "mappe",
		Short:         "Generate and render maps",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newFlatCartographicPNGCommand())
	cmd.AddCommand(newFlatMonochromePNGCommand())
	cmd.AddCommand(newFlatTerrainPNGCommand())
	cmd.AddCommand(newOlssonCartographicPNGCommand())
	cmd.AddCommand(newOlssonMonochromePNGCommand())
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the Mappe version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), mappe.Version().Core())
		},
	})
	return cmd
}

func newFlatCartographicPNGCommand() *cobra.Command {
	seed := int64(42)
	outputPath := ""
	generatorConfig := flat.Config{
		Width:      640,
		Height:     320,
		Iterations: 100,
	}
	rendererConfig := cartographicpng.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "flat-cartographic-png",
		Short: "Run the flat generator and cartographic PNG renderer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			generatorConfig.Source = rand.NewPCG(uint64(seed), 0)
			return writeFlatCartographicPNG(outputPath, generatorConfig, rendererConfig)
		},
	}
	cmd.Flags().Int64Var(&seed, "seed", seed, "pseudorandom seed")
	cmd.Flags().IntVar(&generatorConfig.Width, "width", generatorConfig.Width, "map width")
	cmd.Flags().IntVar(&generatorConfig.Height, "height", generatorConfig.Height, "map height")
	cmd.Flags().IntVar(&generatorConfig.Iterations, "iterations", generatorConfig.Iterations, "number of circular fractures")
	cmd.Flags().BoolVar(&generatorConfig.Wrap, "wrap", generatorConfig.Wrap, "wrap circles across both map axes")
	cmd.Flags().IntVar(&rendererConfig.OceanPercent, "ocean-percent", rendererConfig.OceanPercent, "percentage of map allocated to ocean")
	cmd.Flags().IntVar(&rendererConfig.IcePercent, "ice-percent", rendererConfig.IcePercent, "percentage of map allocated to ice")
	cmd.Flags().StringVarP(&outputPath, "output", "o", outputPath, "PNG output path")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func writeFlatCartographicPNG(path string, generatorConfig flat.Config, rendererConfig cartographicpng.Config) error {
	return writePNG(path, func(output io.Writer) error {
		return flatcartographicpng.Run(output, generatorConfig, rendererConfig)
	})
}

func newFlatMonochromePNGCommand() *cobra.Command {
	seed := int64(42)
	outputPath := ""
	cfg := flat.Config{
		Width:      640,
		Height:     320,
		Iterations: 100,
	}

	cmd := &cobra.Command{
		Use:   "flat-monochrome-png",
		Short: "Run the flat generator and monochrome PNG renderer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg.Source = rand.NewPCG(uint64(seed), 0)
			return writeFlatMonochromePNG(outputPath, cfg)
		},
	}
	cmd.Flags().Int64Var(&seed, "seed", seed, "pseudorandom seed")
	cmd.Flags().IntVar(&cfg.Width, "width", cfg.Width, "map width")
	cmd.Flags().IntVar(&cfg.Height, "height", cfg.Height, "map height")
	cmd.Flags().IntVar(&cfg.Iterations, "iterations", cfg.Iterations, "number of circular fractures")
	cmd.Flags().BoolVar(&cfg.Wrap, "wrap", cfg.Wrap, "wrap circles across both map axes")
	cmd.Flags().StringVarP(&outputPath, "output", "o", outputPath, "PNG output path")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func writeFlatMonochromePNG(path string, cfg flat.Config) error {
	return writePNG(path, func(output io.Writer) error {
		return flatmonochromepng.Run(output, cfg)
	})
}

func newFlatTerrainPNGCommand() *cobra.Command {
	seed := int64(42)
	outputPath := ""
	cfg := flat.Config{
		Width:      640,
		Height:     320,
		Iterations: 10000,
	}
	levelsConfig := percentile.DefaultConfig()
	volcanismConfig := sites.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "flat-terrain-png",
		Short: "Run the flat generator and diagnostic terrain PNG renderer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg.Source = rand.NewPCG(uint64(seed), 0)
			// Volcanism draws from its own stream, so its settings never
			// change the generated terrain.
			volcanismConfig.Source = rand.NewPCG(uint64(seed), 1)
			return writeFlatTerrainPNG(outputPath, cfg, levelsConfig, volcanismConfig)
		},
	}
	cmd.Flags().Int64Var(&seed, "seed", seed, "pseudorandom seed")
	cmd.Flags().IntVar(&cfg.Width, "width", cfg.Width, "map width")
	cmd.Flags().IntVar(&cfg.Height, "height", cfg.Height, "map height")
	cmd.Flags().IntVar(&cfg.Iterations, "iterations", cfg.Iterations, "number of circular fractures")
	cmd.Flags().BoolVar(&cfg.Wrap, "wrap", cfg.Wrap, "wrap terrain and environmental fields across both map axes")
	cmd.Flags().IntVar(&levelsConfig.OceanPercent, "ocean-percent", levelsConfig.OceanPercent, "percentage of map allocated to ocean")
	cmd.Flags().StringVarP(&outputPath, "output", "o", outputPath, "PNG output path")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func writeFlatTerrainPNG(path string, cfg flat.Config, levelsConfig percentile.Config, volcanismConfig sites.Config) error {
	return writePNG(path, func(output io.Writer) error {
		return flatterrainpng.Run(output, cfg, levelsConfig, volcanismConfig, domains.DefaultClassificationConfig())
	})
}

func newOlssonCartographicPNGCommand() *cobra.Command {
	seed := int64(0x638bb317ac47a6ba)
	outputPath := ""
	generatorConfig := olsson.Config{
		Width:  640,
		Height: 320,
		Faults: 100,
	}
	rendererConfig := cartographicpng.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "olsson-cartographic-png",
		Short: "Run the Olsson generator and cartographic PNG renderer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			generatorConfig.Source = rand.NewPCG(uint64(seed), 0)
			return writeOlssonCartographicPNG(outputPath, generatorConfig, rendererConfig)
		},
	}
	cmd.Flags().Int64Var(&seed, "seed", seed, "pseudorandom seed")
	cmd.Flags().IntVar(&generatorConfig.Width, "width", generatorConfig.Width, "map width (must be twice height)")
	cmd.Flags().IntVar(&generatorConfig.Height, "height", generatorConfig.Height, "map height")
	cmd.Flags().IntVar(&generatorConfig.Faults, "faults", generatorConfig.Faults, "number of faults")
	cmd.Flags().IntVar(&rendererConfig.OceanPercent, "ocean-percent", rendererConfig.OceanPercent, "percentage of map allocated to ocean")
	cmd.Flags().IntVar(&rendererConfig.IcePercent, "ice-percent", rendererConfig.IcePercent, "percentage of map allocated to ice")
	cmd.Flags().StringVarP(&outputPath, "output", "o", outputPath, "PNG output path")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func writeOlssonCartographicPNG(path string, generatorConfig olsson.Config, rendererConfig cartographicpng.Config) error {
	return writePNG(path, func(output io.Writer) error {
		return olssoncartographicpng.Run(output, generatorConfig, rendererConfig)
	})
}

func newOlssonMonochromePNGCommand() *cobra.Command {
	seed := int64(0x638bb317ac47a6ba)
	outputPath := ""
	cfg := olsson.Config{
		Width:  640,
		Height: 320,
		Faults: 100,
	}

	cmd := &cobra.Command{
		Use:   "olsson-monochrome-png",
		Short: "Run the Olsson generator and monochrome PNG renderer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg.Source = rand.NewPCG(uint64(seed), 0)
			return writeOlssonMonochromePNG(outputPath, cfg)
		},
	}
	cmd.Flags().Int64Var(&seed, "seed", seed, "pseudorandom seed")
	cmd.Flags().IntVar(&cfg.Width, "width", cfg.Width, "map width (must be twice height)")
	cmd.Flags().IntVar(&cfg.Height, "height", cfg.Height, "map height")
	cmd.Flags().IntVar(&cfg.Faults, "faults", cfg.Faults, "number of faults")
	cmd.Flags().StringVarP(&outputPath, "output", "o", outputPath, "PNG output path")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func writeOlssonMonochromePNG(path string, cfg olsson.Config) error {
	return writePNG(path, func(output io.Writer) error {
		return olssonmonochromepng.Run(output, cfg)
	})
}

func writePNG(path string, run func(io.Writer) error) error {
	output, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if err := run(output); err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	return nil
}
