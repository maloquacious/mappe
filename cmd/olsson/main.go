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
	"encoding/json"
	"fmt"
	"os"

	"github.com/maloquacious/mappe"
	"github.com/maloquacious/mappe/olsson"
	"github.com/spf13/cobra"
)

type output struct {
	Seed       int64 `json:"seed"`
	Width      int   `json:"width"`
	Height     int   `json:"height"`
	Faults     int   `json:"faults"`
	Elevations []int `json:"elevations"`
}

func main() {
	cmd := newCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	cfg := olsson.Config{
		Seed:   0x638bb317ac47a6ba,
		Width:  640,
		Height: 320,
		Faults: 100,
	}

	cmd := &cobra.Command{
		Use:           "olsson",
		Short:         "Generate an Olsson world height map",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			world, err := olsson.Generate(cfg)
			if err != nil {
				return err
			}

			return json.NewEncoder(cmd.OutOrStdout()).Encode(output{
				Seed:       cfg.Seed,
				Width:      world.Width(),
				Height:     world.Height(),
				Faults:     cfg.Faults,
				Elevations: world.Elevations(),
			})
		},
	}
	cmd.Flags().Int64Var(&cfg.Seed, "seed", cfg.Seed, "pseudorandom seed")
	cmd.Flags().IntVar(&cfg.Width, "width", cfg.Width, "map width (must be twice height)")
	cmd.Flags().IntVar(&cfg.Height, "height", cfg.Height, "map height")
	cmd.Flags().IntVar(&cfg.Faults, "faults", cfg.Faults, "number of faults")

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
