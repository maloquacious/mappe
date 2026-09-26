// Mappe - consolidated map and world generators
// Copyright (c) 2026 Michael D Henderson
//
// Portions of this file were adapted from github.com/mdhender/wgva,
// Copyright (c) 2026 Michael D Henderson, under the MIT License.
// See docs/licenses/wgva-MIT.txt.

package domains

// Terrain is the categorical terrain or biome classification of a cell.
type Terrain uint8

// Terrain values have stable IDs because they are serialized domain values.
// Inland sea and lake are reserved but are not produced by the classifier.
const (
	TerrainDeepOcean        Terrain = 0
	TerrainOcean            Terrain = 1
	TerrainShallowSea       Terrain = 2
	TerrainCoastalWater     Terrain = 3
	TerrainInlandSea        Terrain = 4
	TerrainLake             Terrain = 5
	TerrainGlacialIce       Terrain = 6
	TerrainTundra           Terrain = 7
	TerrainMarsh            Terrain = 8
	TerrainSwamp            Terrain = 9
	TerrainBog              Terrain = 10
	TerrainDesert           Terrain = 11
	TerrainBadlands         Terrain = 12
	TerrainScrubland        Terrain = 13
	TerrainPlains           Terrain = 14
	TerrainGrassland        Terrain = 15
	TerrainSteppe           Terrain = 16
	TerrainSavanna          Terrain = 17
	TerrainBorealForest     Terrain = 18
	TerrainTemperateForest  Terrain = 19
	TerrainRainforest       Terrain = 20
	TerrainHills            Terrain = 21
	TerrainMountain         Terrain = 22
	TerrainAlpine           Terrain = 23
	TerrainVolcano          Terrain = 24
	TerrainVolcanicHighland Terrain = 25
	TerrainCoast            Terrain = 26
)

// Terrains returns every declared terrain in stable ID order.
func Terrains() []Terrain {
	return []Terrain{
		TerrainDeepOcean, TerrainOcean, TerrainShallowSea, TerrainCoastalWater,
		TerrainInlandSea, TerrainLake,
		TerrainGlacialIce, TerrainTundra,
		TerrainMarsh, TerrainSwamp, TerrainBog,
		TerrainDesert, TerrainBadlands, TerrainScrubland,
		TerrainPlains, TerrainGrassland, TerrainSteppe, TerrainSavanna,
		TerrainBorealForest, TerrainTemperateForest, TerrainRainforest,
		TerrainHills, TerrainMountain, TerrainAlpine,
		TerrainVolcano, TerrainVolcanicHighland,
		TerrainCoast,
	}
}

// String returns the stable name of the terrain.
func (t Terrain) String() string {
	switch t {
	case TerrainDeepOcean:
		return "deep-ocean"
	case TerrainOcean:
		return "ocean"
	case TerrainShallowSea:
		return "shallow-sea"
	case TerrainCoastalWater:
		return "coastal-water"
	case TerrainInlandSea:
		return "inland-sea"
	case TerrainLake:
		return "lake"
	case TerrainGlacialIce:
		return "glacial-ice"
	case TerrainTundra:
		return "tundra"
	case TerrainMarsh:
		return "marsh"
	case TerrainSwamp:
		return "swamp"
	case TerrainBog:
		return "bog"
	case TerrainDesert:
		return "desert"
	case TerrainBadlands:
		return "badlands"
	case TerrainScrubland:
		return "scrubland"
	case TerrainPlains:
		return "plains"
	case TerrainGrassland:
		return "grassland"
	case TerrainSteppe:
		return "steppe"
	case TerrainSavanna:
		return "savanna"
	case TerrainBorealForest:
		return "boreal-forest"
	case TerrainTemperateForest:
		return "temperate-forest"
	case TerrainRainforest:
		return "rainforest"
	case TerrainHills:
		return "hills"
	case TerrainMountain:
		return "mountain"
	case TerrainAlpine:
		return "alpine"
	case TerrainVolcano:
		return "volcano"
	case TerrainVolcanicHighland:
		return "volcanic-highland"
	case TerrainCoast:
		return "coast"
	default:
		return "unknown"
	}
}

// Valid reports whether t is a declared terrain.
func (t Terrain) Valid() bool { return t <= TerrainCoast }

// IsWater reports whether t represents open or inland water.
func (t Terrain) IsWater() bool {
	switch t {
	case TerrainDeepOcean, TerrainOcean, TerrainShallowSea, TerrainCoastalWater,
		TerrainInlandSea, TerrainLake:
		return true
	default:
		return false
	}
}

// TerrainThresholds contains the normalized thresholds used by the ordered
// terrain and biome classifier.
type TerrainThresholds struct {
	DeepOceanDepth            float64
	OceanDepth                float64
	IceHeat                   float64
	AlpineHeat                float64
	VolcanicElevation         float64
	VolcanoThreshold          float64
	VolcanoRelief             float64
	VolcanicHighlandThreshold float64
	HillsRelief               float64
	WetlandWetness            float64
	WetlandElevation          float64
	WetlandRelief             float64
	BogHeat                   float64
	SwampHeat                 float64
	BadlandsRelief            float64
}

func classifyTerrain(sample EnvironmentalSample, elevationBand ElevationBand, heatBand HeatBand, wetness float64, wetnessBand MoistureBand, landNeighbor, waterNeighbor bool, cfg ClassificationConfig) Terrain {
	t := cfg.Terrain
	if sample.Elevation <= cfg.Elevation.SeaLevel {
		switch {
		case landNeighbor:
			return TerrainCoastalWater
		case sample.Elevation <= t.DeepOceanDepth:
			return TerrainDeepOcean
		case sample.Elevation <= t.OceanDepth:
			return TerrainOcean
		default:
			return TerrainShallowSea
		}
	}
	if sample.Heat <= t.IceHeat {
		return TerrainGlacialIce
	}
	if sample.Elevation >= t.VolcanicElevation {
		switch {
		case sample.Volcanic >= t.VolcanoThreshold && sample.Relief >= t.VolcanoRelief:
			return TerrainVolcano
		case sample.Volcanic >= t.VolcanicHighlandThreshold:
			return TerrainVolcanicHighland
		}
	}
	if elevationBand == ElevationMountain {
		if sample.Heat <= t.AlpineHeat {
			return TerrainAlpine
		}
		return TerrainMountain
	}
	if elevationBand == ElevationHighland || sample.Relief >= t.HillsRelief {
		return TerrainHills
	}
	if wetness >= t.WetlandWetness && sample.Elevation <= t.WetlandElevation && sample.Relief <= t.WetlandRelief {
		switch {
		case sample.Heat <= t.BogHeat:
			return TerrainBog
		case sample.Heat >= t.SwampHeat:
			return TerrainSwamp
		default:
			return TerrainMarsh
		}
	}
	if waterNeighbor {
		return TerrainCoast
	}
	return classifyBiome(heatBand, wetnessBand, sample.Relief, t.BadlandsRelief)
}

func classifyBiome(heat HeatBand, moisture MoistureBand, relief, badlandsRelief float64) Terrain {
	if !heat.Valid() || !moisture.Valid() {
		return TerrainPlains
	}
	terrain := biomeTable[heat][moisture]
	if terrain == TerrainDesert && relief >= badlandsRelief {
		return TerrainBadlands
	}
	return terrain
}

var biomeTable = [5][5]Terrain{
	HeatPolar: {
		MoistureArid: TerrainTundra, MoistureDry: TerrainTundra,
		MoistureModerate: TerrainTundra, MoistureHumid: TerrainTundra,
		MoistureSaturated: TerrainTundra,
	},
	HeatCold: {
		MoistureArid: TerrainTundra, MoistureDry: TerrainSteppe,
		MoistureModerate: TerrainPlains, MoistureHumid: TerrainBorealForest,
		MoistureSaturated: TerrainBorealForest,
	},
	HeatTemperate: {
		MoistureArid: TerrainDesert, MoistureDry: TerrainScrubland,
		MoistureModerate: TerrainGrassland, MoistureHumid: TerrainTemperateForest,
		MoistureSaturated: TerrainTemperateForest,
	},
	HeatWarm: {
		MoistureArid: TerrainDesert, MoistureDry: TerrainScrubland,
		MoistureModerate: TerrainSavanna, MoistureHumid: TerrainTemperateForest,
		MoistureSaturated: TerrainRainforest,
	},
	HeatHot: {
		MoistureArid: TerrainDesert, MoistureDry: TerrainDesert,
		MoistureModerate: TerrainSavanna, MoistureHumid: TerrainRainforest,
		MoistureSaturated: TerrainRainforest,
	},
}
