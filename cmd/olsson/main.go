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
	"flag"
	"fmt"
	"os"

	"github.com/maloquacious/mappe/olsson"
)

type output struct {
	Seed       int64 `json:"seed"`
	Width      int   `json:"width"`
	Height     int   `json:"height"`
	Faults     int   `json:"faults"`
	Elevations []int `json:"elevations"`
}

func main() {
	seed := flag.Int64("seed", 0x638bb317ac47a6ba, "pseudorandom seed")
	width := flag.Int("width", 640, "map width (must be twice height)")
	height := flag.Int("height", 320, "map height")
	faults := flag.Int("faults", 100, "number of faults")
	flag.Parse()

	world, err := olsson.Generate(olsson.Config{
		Seed:   *seed,
		Width:  *width,
		Height: *height,
		Faults: *faults,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = json.NewEncoder(os.Stdout).Encode(output{
		Seed:       *seed,
		Width:      world.Width(),
		Height:     world.Height(),
		Faults:     *faults,
		Elevations: world.Elevations(),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
