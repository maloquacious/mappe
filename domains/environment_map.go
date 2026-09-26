// Mappe - consolidated map and world generators
// Copyright (c) 2026 Michael D Henderson
//
// Portions of this file were adapted from github.com/mdhender/wgva,
// Copyright (c) 2026 Michael D Henderson, under the MIT License.
// See docs/licenses/wgva-MIT.txt.

package domains

import (
	"math"

	"github.com/maloquacious/mappe/internal/cerrs"
)

// Environmental map construction errors.
const (
	ErrInvalidEnvironmentalSampleCount cerrs.Error = "domains: environmental sample count must equal width times height"
	ErrInvalidEnvironmentalSample      cerrs.Error = "domains: environmental samples must contain finite normalized values in [0, 1]"
	ErrInvalidClassificationConfig     cerrs.Error = "domains: classification thresholds must be finite, normalized, and ordered"
	ErrNilElevationLevels              cerrs.Error = "domains: elevation levels must not be nil"
	ErrNilVolcanicFeatures             cerrs.Error = "domains: volcanic features must not be nil"
	ErrVolcanicFeaturesSize            cerrs.Error = "domains: volcanic features must match the environmental map size"
)

// EnvironmentalSample contains the normalized physical values used to classify
// one Cartesian cell. All values are in [0, 1]. Sea level and the other
// elevation boundaries come from ElevationLevels rather than a fixed value;
// heat, moisture, and basin values use 0.5 as their neutral value.
type EnvironmentalSample struct {
	Elevation float64
	Heat      float64
	Moisture  float64
	Relief    float64
	Basin     float64
}

// EnvironmentalCell preserves a cell's physical values beside its derived
// elevation, climate, and terrain classifications.
type EnvironmentalCell struct {
	Sample        EnvironmentalSample
	ElevationBand ElevationBand
	Climate       Climate
	Terrain       Terrain
}

// ClassificationConfig contains the non-elevation thresholds used to classify
// normalized environmental samples. Elevation boundaries are supplied
// separately as ElevationLevels.
type ClassificationConfig struct {
	Heat                HeatThresholds
	Moisture            MoistureThresholds
	BasinMoistureWeight float64
	Terrain             TerrainThresholds
}

// DefaultClassificationConfig returns wgva's classification defaults converted
// linearly from signed [-1, 1] values to normalized [0, 1] values.
func DefaultClassificationConfig() ClassificationConfig {
	return ClassificationConfig{
		Heat: HeatThresholds{
			Polar: 0.2, Cold: 0.4, Temperate: 0.6, Warm: 0.8,
		},
		Moisture: MoistureThresholds{
			Arid: 0.2, Dry: 0.4, Moderate: 0.6, Humid: 0.8,
		},
		BasinMoistureWeight: 0.6,
		Terrain: TerrainThresholds{
			IceHeat:        0.09,
			AlpineHeat:     0.35,
			HillsRelief:    0.6,
			WetlandWetness: 0.7,
			WetlandRelief:  0.3,
			BogHeat:        0.375,
			SwampHeat:      0.65,
			BadlandsRelief: 0.4,
		},
	}
}

// Validate reports whether the classification thresholds are normalized and
// ordered so every declared band and rule remains reachable.
func (c ClassificationConfig) Validate() error {
	for _, values := range [][]float64{
		{c.Heat.Polar, c.Heat.Cold, c.Heat.Temperate, c.Heat.Warm},
		{c.Moisture.Arid, c.Moisture.Dry, c.Moisture.Moderate, c.Moisture.Humid},
	} {
		if !strictlyAscendingOpenUnit(values) {
			return ErrInvalidClassificationConfig
		}
	}
	for _, value := range []float64{
		c.BasinMoistureWeight,
		c.Terrain.HillsRelief,
		c.Terrain.WetlandRelief, c.Terrain.BadlandsRelief,
	} {
		if !normalized(value) {
			return ErrInvalidClassificationConfig
		}
	}
	for _, value := range []float64{
		c.Terrain.IceHeat, c.Terrain.AlpineHeat,
		c.Terrain.WetlandWetness, c.Terrain.BogHeat, c.Terrain.SwampHeat,
	} {
		if value <= 0 || value >= 1 || !normalized(value) {
			return ErrInvalidClassificationConfig
		}
	}
	if c.Terrain.IceHeat >= c.Terrain.AlpineHeat ||
		c.Terrain.BogHeat >= c.Terrain.SwampHeat {
		return ErrInvalidClassificationConfig
	}
	return nil
}

// EnvironmentalMap is an immutable Cartesian, row-major environmental grid.
type EnvironmentalMap struct {
	width    int
	height   int
	topology GridTopology
	cells    []EnvironmentalCell
}

// NewEnvironmentalMap validates and classifies row-major samples. Elevation
// bands, water, and depth come from levels, and volcanism from volcanic,
// whichever stages produced them. Volcanic kinds apply only to land. Neighbor
// classification uses the supplied topology at all four Cartesian edges.
func NewEnvironmentalMap(width, height int, topology GridTopology, samples []EnvironmentalSample, levels *ElevationLevels, volcanic *VolcanicFeatures, cfg ClassificationConfig) (*EnvironmentalMap, error) {
	if width < 1 {
		return nil, ErrInvalidWidth
	}
	if height < 1 {
		return nil, ErrInvalidHeight
	}
	if width > int(^uint(0)>>1)/height || len(samples) != width*height {
		return nil, ErrInvalidEnvironmentalSampleCount
	}
	if levels == nil {
		return nil, ErrNilElevationLevels
	}
	if volcanic == nil {
		return nil, ErrNilVolcanicFeatures
	}
	if volcanic.width != width || volcanic.height != height {
		return nil, ErrVolcanicFeaturesSize
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	for _, sample := range samples {
		if !sample.valid() {
			return nil, ErrInvalidEnvironmentalSample
		}
	}

	cells := make([]EnvironmentalCell, len(samples))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			sample := samples[index]
			elevationBand := levels.Classify(sample.Elevation)
			climate := Climate{
				Heat:     cfg.Heat.Classify(sample.Heat),
				Moisture: cfg.Moisture.Classify(sample.Moisture),
			}
			wetness := adjustedWetness(sample.Moisture, sample.Basin, cfg.BasinMoistureWeight)
			landNeighbor, waterNeighbor := neighborKinds(samples, width, height, x, y, topology, levels.values.SeaLevel)
			cells[index] = EnvironmentalCell{
				Sample:        sample,
				ElevationBand: elevationBand,
				Climate:       climate,
				Terrain: classifyTerrain(
					sample,
					elevationBand,
					climate.Heat,
					wetness,
					cfg.Moisture.Classify(wetness),
					landNeighbor,
					waterNeighbor,
					volcanic.kinds[index],
					levels.values,
					cfg,
				),
			}
		}
	}

	return &EnvironmentalMap{width: width, height: height, topology: topology, cells: cells}, nil
}

// Width returns the number of columns in the map.
func (m *EnvironmentalMap) Width() int { return m.width }

// Height returns the number of rows in the map.
func (m *EnvironmentalMap) Height() int { return m.height }

// Topology returns the map's Cartesian edge topology.
func (m *EnvironmentalMap) Topology() GridTopology { return m.topology }

// Cell returns the environmental cell at (x, y). It panics for coordinates
// outside the map.
func (m *EnvironmentalMap) Cell(x, y int) EnvironmentalCell {
	if x < 0 || x >= m.width || y < 0 || y >= m.height {
		panic("domains: coordinates outside environmental map")
	}
	return m.cells[y*m.width+x]
}

// Cells returns a copy of all cells in row-major order.
func (m *EnvironmentalMap) Cells() []EnvironmentalCell {
	return append([]EnvironmentalCell(nil), m.cells...)
}

func (s EnvironmentalSample) valid() bool {
	return normalized(s.Elevation) && normalized(s.Heat) && normalized(s.Moisture) &&
		normalized(s.Relief) && normalized(s.Basin)
}

func normalized(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func strictlyAscendingNormalized(values []float64) bool {
	for i, value := range values {
		if !normalized(value) || i > 0 && value <= values[i-1] {
			return false
		}
	}
	return true
}

func strictlyAscendingOpenUnit(values []float64) bool {
	return strictlyAscendingNormalized(values) && values[0] > 0 && values[len(values)-1] < 1
}

func adjustedWetness(moisture, basin, weight float64) float64 {
	signedMoisture := 2*moisture - 1
	signedBasin := 2*basin - 1
	signedWetness := signedMoisture * (1 + weight*signedBasin)
	signedWetness = max(-1, min(1, signedWetness))
	return (signedWetness + 1) / 2
}

func neighborKinds(samples []EnvironmentalSample, width, height, x, y int, topology GridTopology, seaLevel float64) (land, water bool) {
	for _, offset := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		nx, ny := x+offset[0], y+offset[1]
		if nx < 0 || nx >= width {
			if !topology.WrapEastWest {
				continue
			}
			nx = moduloCoordinate(nx, width)
		}
		if ny < 0 || ny >= height {
			if !topology.WrapNorthSouth {
				continue
			}
			ny = moduloCoordinate(ny, height)
		}
		if samples[ny*width+nx].Elevation <= seaLevel {
			water = true
		} else {
			land = true
		}
	}
	return land, water
}

func moduloCoordinate(value, modulus int) int {
	value %= modulus
	if value < 0 {
		value += modulus
	}
	return value
}
