package domains

import (
	"math"
	"testing"
)

func TestEnvironmentalMapUsesCartesianTopologyAtEdges(t *testing.T) {
	for _, tt := range []struct {
		name     string
		topology GridTopology
		waterX   bool
		waterY   bool
		want     Terrain
	}{
		{name: "bounded", topology: GridTopology{}, waterX: true, waterY: true, want: TerrainGrassland},
		{name: "east-west", topology: GridTopology{WrapEastWest: true}, waterX: true, want: TerrainCoast},
		{name: "north-south", topology: GridTopology{WrapNorthSouth: true}, waterY: true, want: TerrainCoast},
		{name: "both", topology: GridTopology{WrapEastWest: true, WrapNorthSouth: true}, waterX: true, want: TerrainCoast},
	} {
		t.Run(tt.name, func(t *testing.T) {
			samples := repeatedSamples(3, 3, ordinarySample())
			if tt.waterX {
				samples[0*3+2].Elevation = 0.4 // west of (0, 0) only when east-west wraps
			}
			if tt.waterY {
				samples[2*3+0].Elevation = 0.4 // north of (0, 0) only when north-south wraps
			}
			m, err := NewEnvironmentalMap(3, 3, tt.topology, samples, testLevels(t), noVolcanism(t, 3, 3), DefaultClassificationConfig())
			if err != nil {
				t.Fatal(err)
			}
			if got := m.Cell(0, 0).Terrain; got != tt.want {
				t.Fatalf("terrain at (0, 0) = %v, want %v", got, tt.want)
			}
			if m.Topology() != tt.topology {
				t.Fatalf("Topology = %+v, want %+v", m.Topology(), tt.topology)
			}
		})
	}
}

func TestEnvironmentalMapIsRowMajorAndDefensivelyCopied(t *testing.T) {
	samples := repeatedSamples(2, 2, ordinarySample())
	samples[2].Heat = 0.9
	m, err := NewEnvironmentalMap(2, 2, GridTopology{}, samples, testLevels(t), noVolcanism(t, 2, 2), DefaultClassificationConfig())
	if err != nil {
		t.Fatal(err)
	}
	if m.Width() != 2 || m.Height() != 2 || m.Cell(0, 1).Climate.Heat != HeatHot {
		t.Fatalf("map data = %dx%d %+v, want 2x2 hot at (0, 1)", m.Width(), m.Height(), m.Cell(0, 1))
	}

	samples[2].Heat = 0
	cells := m.Cells()
	cells[2].Terrain = TerrainLake
	if m.Cell(0, 1).Climate.Heat != HeatHot || m.Cell(0, 1).Terrain == TerrainLake {
		t.Fatal("changing constructor input or Cells result changed the map")
	}
}

func TestNewEnvironmentalMapRejectsInvalidInput(t *testing.T) {
	valid := ordinarySample()
	badSample := valid
	badSample.Heat = math.NaN()
	badConfig := DefaultClassificationConfig()
	badConfig.Heat.Cold = badConfig.Heat.Polar

	levels := testLevels(t)
	volcanic := noVolcanism(t, 1, 1)

	for _, tt := range []struct {
		name     string
		width    int
		height   int
		samples  []EnvironmentalSample
		levels   *ElevationLevels
		volcanic *VolcanicFeatures
		cfg      ClassificationConfig
		want     error
	}{
		{name: "width", height: 1, levels: levels, volcanic: volcanic, cfg: DefaultClassificationConfig(), want: ErrInvalidWidth},
		{name: "height", width: 1, levels: levels, volcanic: volcanic, cfg: DefaultClassificationConfig(), want: ErrInvalidHeight},
		{name: "count", width: 2, height: 1, samples: []EnvironmentalSample{valid}, levels: levels, volcanic: volcanic, cfg: DefaultClassificationConfig(), want: ErrInvalidEnvironmentalSampleCount},
		{name: "levels", width: 1, height: 1, samples: []EnvironmentalSample{valid}, volcanic: volcanic, cfg: DefaultClassificationConfig(), want: ErrNilElevationLevels},
		{name: "volcanic features", width: 1, height: 1, samples: []EnvironmentalSample{valid}, levels: levels, cfg: DefaultClassificationConfig(), want: ErrNilVolcanicFeatures},
		{name: "volcanic features size", width: 1, height: 1, samples: []EnvironmentalSample{valid}, levels: levels, volcanic: noVolcanism(t, 2, 1), cfg: DefaultClassificationConfig(), want: ErrVolcanicFeaturesSize},
		{name: "sample", width: 1, height: 1, samples: []EnvironmentalSample{badSample}, levels: levels, volcanic: volcanic, cfg: DefaultClassificationConfig(), want: ErrInvalidEnvironmentalSample},
		{name: "config", width: 1, height: 1, samples: []EnvironmentalSample{valid}, levels: levels, volcanic: volcanic, cfg: badConfig, want: ErrInvalidClassificationConfig},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewEnvironmentalMap(tt.width, tt.height, GridTopology{}, tt.samples, tt.levels, tt.volcanic, tt.cfg); err != tt.want {
				t.Fatalf("NewEnvironmentalMap error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestEnvironmentalMapClassifiesSuppliedVolcanicFeatures(t *testing.T) {
	samples := repeatedSamples(3, 1, ordinarySample())
	samples[0].Elevation = 0.4
	samples[2].Heat = DefaultClassificationConfig().Terrain.IceHeat
	kinds := []VolcanicKind{VolcanicHighland, VolcanicVolcano, VolcanicHighland}
	volcanic, err := NewVolcanicFeatures(3, 1, []VolcanoSite{{X: 1, Y: 0}}, kinds)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewEnvironmentalMap(3, 1, GridTopology{}, samples, testLevels(t), volcanic, DefaultClassificationConfig())
	if err != nil {
		t.Fatal(err)
	}
	want := []Terrain{TerrainCoastalWater, TerrainVolcano, TerrainVolcanicHighland}
	for x, terrain := range want {
		if got := m.Cell(x, 0).Terrain; got != terrain {
			t.Errorf("terrain at (%d, 0) = %v, want %v", x, got, terrain)
		}
	}
}

func TestEnvironmentalMapCellPanicsOutsideMap(t *testing.T) {
	m, err := NewEnvironmentalMap(1, 1, GridTopology{}, []EnvironmentalSample{ordinarySample()}, testLevels(t), noVolcanism(t, 1, 1), DefaultClassificationConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("Cell did not panic")
		}
	}()
	m.Cell(1, 0)
}

func ordinarySample() EnvironmentalSample {
	return EnvironmentalSample{
		Elevation: 0.55,
		Heat:      0.5,
		Moisture:  0.5,
		Relief:    0.1,
		Basin:     0.5,
	}
}

func repeatedSamples(width, height int, sample EnvironmentalSample) []EnvironmentalSample {
	samples := make([]EnvironmentalSample, width*height)
	for i := range samples {
		samples[i] = sample
	}
	return samples
}

func noVolcanism(t *testing.T, width, height int) *VolcanicFeatures {
	t.Helper()
	features, err := NewVolcanicFeatures(width, height, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return features
}
