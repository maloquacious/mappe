package domains

import (
	"math"
	"testing"
)

func TestTerrainVocabularyHasStableIDsAndNames(t *testing.T) {
	want := []string{
		"deep-ocean", "ocean", "shallow-sea", "coastal-water",
		"inland-sea", "lake", "glacial-ice", "tundra", "marsh",
		"swamp", "bog", "desert", "badlands", "scrubland", "plains",
		"grassland", "steppe", "savanna", "boreal-forest",
		"temperate-forest", "rainforest", "hills", "mountain", "alpine",
		"volcano", "volcanic-highland", "coast",
	}
	terrains := Terrains()
	if len(terrains) != len(want) {
		t.Fatalf("len(Terrains()) = %d, want %d", len(terrains), len(want))
	}
	for id, terrain := range terrains {
		if int(terrain) != id || terrain.String() != want[id] || !terrain.Valid() {
			t.Errorf("terrain %d = %d/%q/valid=%v, want %d/%q/true", id, terrain, terrain, terrain.Valid(), id, want[id])
		}
	}
	if terrain := Terrain(200); terrain.Valid() || terrain.String() != "unknown" {
		t.Fatalf("undeclared terrain = %q/valid=%v, want unknown/false", terrain, terrain.Valid())
	}
}

func TestTerrainWaterIsExactlySixVariants(t *testing.T) {
	want := map[Terrain]bool{
		TerrainDeepOcean: true, TerrainOcean: true, TerrainShallowSea: true,
		TerrainCoastalWater: true, TerrainInlandSea: true, TerrainLake: true,
	}
	for _, terrain := range Terrains() {
		if terrain.IsWater() != want[terrain] {
			t.Errorf("%v.IsWater() = %v, want %v", terrain, terrain.IsWater(), want[terrain])
		}
	}
}

func TestClimateVocabularyHasStableIDsAndNames(t *testing.T) {
	for id, want := range []string{"polar", "cold", "temperate", "warm", "hot"} {
		heat := HeatBand(id)
		if !heat.Valid() || heat.String() != want || HeatBands()[id] != heat {
			t.Errorf("heat %d = %q/valid=%v", id, heat, heat.Valid())
		}
	}
	for id, want := range []string{"arid", "dry", "moderate", "humid", "saturated"} {
		moisture := MoistureBand(id)
		if !moisture.Valid() || moisture.String() != want || MoistureBands()[id] != moisture {
			t.Errorf("moisture %d = %q/valid=%v", id, moisture, moisture.Valid())
		}
	}
	for _, heat := range HeatBands() {
		for _, moisture := range MoistureBands() {
			climate := Climate{Heat: heat, Moisture: moisture}
			if !climate.Valid() || climate.String() != heat.String()+"/"+moisture.String() {
				t.Errorf("climate %+v = %q/valid=%v", climate, climate, climate.Valid())
			}
		}
	}
}

func TestElevationVocabularyHasStableIDsAndNames(t *testing.T) {
	want := []string{"deep-water", "shallow-water", "lowland", "upland", "highland", "mountain"}
	for id, band := range ElevationBands() {
		if int(band) != id || !band.Valid() || band.String() != want[id] {
			t.Errorf("elevation %d = %d/%q/valid=%v, want %d/%q/true", id, band, band, band.Valid(), id, want[id])
		}
		if got := band.IsWater(); got != (band == ElevationDeepWater || band == ElevationShallowWater) {
			t.Errorf("%v.IsWater() = %v", band, got)
		}
	}
}

func TestClassificationThresholdBoundaries(t *testing.T) {
	cfg := DefaultClassificationConfig()
	for _, tt := range []struct {
		name string
		at   float64
		next float64
		got  func(float64) string
		want string
	}{
		{name: "polar", at: cfg.Heat.Polar, next: math.Nextafter(cfg.Heat.Polar, 1), got: func(v float64) string { return cfg.Heat.Classify(v).String() }, want: "polar/cold"},
		{name: "cold", at: cfg.Heat.Cold, next: math.Nextafter(cfg.Heat.Cold, 1), got: func(v float64) string { return cfg.Heat.Classify(v).String() }, want: "cold/temperate"},
		{name: "arid", at: cfg.Moisture.Arid, next: math.Nextafter(cfg.Moisture.Arid, 1), got: func(v float64) string { return cfg.Moisture.Classify(v).String() }, want: "arid/dry"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got(tt.at) + "/" + tt.got(tt.next); got != tt.want {
				t.Fatalf("boundary classifications = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBiomeTableCoversEveryClimate(t *testing.T) {
	want := [5][5]Terrain{
		{TerrainTundra, TerrainTundra, TerrainTundra, TerrainTundra, TerrainTundra},
		{TerrainTundra, TerrainSteppe, TerrainPlains, TerrainBorealForest, TerrainBorealForest},
		{TerrainDesert, TerrainScrubland, TerrainGrassland, TerrainTemperateForest, TerrainTemperateForest},
		{TerrainDesert, TerrainScrubland, TerrainSavanna, TerrainTemperateForest, TerrainRainforest},
		{TerrainDesert, TerrainDesert, TerrainSavanna, TerrainRainforest, TerrainRainforest},
	}
	for _, heat := range HeatBands() {
		for _, moisture := range MoistureBands() {
			if got := classifyBiome(heat, moisture, 0, 0.4); got != want[heat][moisture] {
				t.Errorf("classifyBiome(%v, %v) = %v, want %v", heat, moisture, got, want[heat][moisture])
			}
		}
	}
	if got := classifyBiome(HeatHot, MoistureArid, 0.4, 0.4); got != TerrainBadlands {
		t.Fatalf("steep hot/arid biome = %v, want badlands", got)
	}
}

func TestBasinInfluenceDeepensExistingMoisture(t *testing.T) {
	const weight = 0.6
	for _, tt := range []struct {
		moisture float64
		basin    float64
		want     float64
	}{
		{moisture: 0.5, basin: 1, want: 0.5},
		{moisture: 0.75, basin: 1, want: 0.9},
		{moisture: 0.25, basin: 1, want: 0.1},
		{moisture: 0.75, basin: 0.5, want: 0.75},
	} {
		if got := adjustedWetness(tt.moisture, tt.basin, weight); math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("adjustedWetness(%v, %v, %v) = %v, want %v", tt.moisture, tt.basin, weight, got, tt.want)
		}
	}
}

func TestTerrainRulesPreserveSourcePrecedence(t *testing.T) {
	cfg := DefaultClassificationConfig()
	levels := testLevels(t)
	values := levels.Values()
	base := EnvironmentalSample{Elevation: 0.55, Heat: 0.5, Moisture: 0.5, Relief: 0.1, Basin: 0.5}
	test := func(sample EnvironmentalSample, landNeighbor, waterNeighbor bool, volcanic VolcanicKind) Terrain {
		elevation := levels.Classify(sample.Elevation)
		heat := cfg.Heat.Classify(sample.Heat)
		wetness := adjustedWetness(sample.Moisture, sample.Basin, cfg.BasinMoistureWeight)
		return classifyTerrain(sample, elevation, heat, wetness, cfg.Moisture.Classify(wetness), landNeighbor, waterNeighbor, volcanic, values, cfg)
	}
	with := func(change func(*EnvironmentalSample)) EnvironmentalSample {
		sample := base
		change(&sample)
		return sample
	}

	for _, tt := range []struct {
		name          string
		sample        EnvironmentalSample
		landNeighbor  bool
		waterNeighbor bool
		volcanic      VolcanicKind
		want          Terrain
	}{
		{name: "coastal water outranks depth", sample: with(func(s *EnvironmentalSample) { s.Elevation = 0.1 }), landNeighbor: true, want: TerrainCoastalWater},
		{name: "abyss level is deep ocean", sample: with(func(s *EnvironmentalSample) { s.Elevation = values.Abyss }), want: TerrainDeepOcean},
		{name: "above abyss is ocean", sample: with(func(s *EnvironmentalSample) { s.Elevation = math.Nextafter(values.Abyss, 1) }), want: TerrainOcean},
		{name: "shelf level is ocean", sample: with(func(s *EnvironmentalSample) { s.Elevation = values.Shelf }), want: TerrainOcean},
		{name: "above shelf is shallow sea", sample: with(func(s *EnvironmentalSample) { s.Elevation = math.Nextafter(values.Shelf, 1) }), want: TerrainShallowSea},
		{name: "sea level is water", sample: with(func(s *EnvironmentalSample) { s.Elevation = values.SeaLevel }), want: TerrainShallowSea},
		{name: "ice outranks mountain", sample: with(func(s *EnvironmentalSample) { s.Elevation, s.Heat = 0.9, cfg.Terrain.IceHeat }), want: TerrainGlacialIce},
		{name: "volcanism ignores water", sample: with(func(s *EnvironmentalSample) { s.Elevation = values.SeaLevel }), volcanic: VolcanicVolcano, want: TerrainShallowSea},
		{name: "volcano outranks ice", sample: with(func(s *EnvironmentalSample) { s.Elevation, s.Heat = 0.9, cfg.Terrain.IceHeat }), volcanic: VolcanicVolcano, want: TerrainVolcano},
		{name: "volcanic highland outranks ice", sample: with(func(s *EnvironmentalSample) { s.Heat = cfg.Terrain.IceHeat }), volcanic: VolcanicHighland, want: TerrainVolcanicHighland},
		{name: "volcano outranks mountain", sample: with(func(s *EnvironmentalSample) { s.Elevation = 0.9 }), volcanic: VolcanicVolcano, want: TerrainVolcano},
		{name: "volcanic highland outranks hills", sample: with(func(s *EnvironmentalSample) { s.Elevation = values.Highland }), volcanic: VolcanicHighland, want: TerrainVolcanicHighland},
		{name: "volcanic highland reaches lowland", sample: base, volcanic: VolcanicHighland, want: TerrainVolcanicHighland},
		{name: "volcano outranks coast", sample: base, waterNeighbor: true, volcanic: VolcanicVolcano, want: TerrainVolcano},
		{name: "cold mountain is alpine", sample: with(func(s *EnvironmentalSample) { s.Elevation, s.Heat = 0.9, cfg.Terrain.AlpineHeat }), want: TerrainAlpine},
		{name: "relief makes hills", sample: with(func(s *EnvironmentalSample) { s.Relief = cfg.Terrain.HillsRelief }), want: TerrainHills},
		{name: "wetland outranks coast", sample: with(func(s *EnvironmentalSample) { s.Moisture = cfg.Terrain.WetlandWetness }), waterNeighbor: true, want: TerrainMarsh},
		{name: "cold wetland is bog", sample: with(func(s *EnvironmentalSample) { s.Moisture, s.Heat = cfg.Terrain.WetlandWetness, cfg.Terrain.BogHeat }), want: TerrainBog},
		{name: "warm wetland is swamp", sample: with(func(s *EnvironmentalSample) { s.Moisture, s.Heat = cfg.Terrain.WetlandWetness, cfg.Terrain.SwampHeat }), want: TerrainSwamp},
		{name: "wetland stops at the upland level", sample: with(func(s *EnvironmentalSample) { s.Elevation, s.Moisture = values.Upland, cfg.Terrain.WetlandWetness }), want: TerrainTemperateForest},
		{name: "shoreline is coast", sample: base, waterNeighbor: true, want: TerrainCoast},
		{name: "ordinary ground uses biome", sample: base, want: TerrainGrassland},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := test(tt.sample, tt.landNeighbor, tt.waterNeighbor, tt.volcanic); got != tt.want {
				t.Fatalf("terrain = %v, want %v", got, tt.want)
			}
		})
	}
}
