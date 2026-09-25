package olsson

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/maloquacious/mappe/domains"
)

const (
	sourceWidth  = 640
	sourceHeight = 320
	sourceFaults = 100
)

var goldenTests = []struct {
	seed int64
	want string
}{
	{seed: 0, want: "ee524377b9b1f398f728ecee3abef06163991e1f8979fedaf2635556f79813e6"},
	{seed: 0x638bb317ac47a6ba, want: "f2117b23bc908767c610bccab2b8361dc2a2e906f96592fc4f2621352acbb04a"},
}

func TestGenerateRawGolden(t *testing.T) {
	for _, tt := range goldenTests {
		t.Run(fmt.Sprintf("seed_%d", tt.seed), func(t *testing.T) {
			elevations, err := generateRaw(testConfig(tt.seed))
			if err != nil {
				t.Fatal(err)
			}
			if got := rawChecksum(elevations); got != tt.want {
				t.Fatalf("checksum = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGenerateNormalizedHeightMapIsIndependentAcrossConcurrentCalls(t *testing.T) {
	wants := make(map[int64]string, len(goldenTests))
	for _, tt := range goldenTests {
		m, err := GenerateNormalizedHeightMap(testConfig(tt.seed))
		if err != nil {
			t.Fatal(err)
		}
		wants[tt.seed] = normalizedChecksum(m)
	}

	var wg sync.WaitGroup
	for _, tt := range goldenTests {
		tt := tt
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := GenerateNormalizedHeightMap(testConfig(tt.seed))
			if err != nil {
				t.Error(err)
				return
			}
			if got := normalizedChecksum(m); got != wants[tt.seed] {
				t.Errorf("seed %d checksum = %s, want %s", tt.seed, got, wants[tt.seed])
			}
		}()
	}
	wg.Wait()
}

func TestNormalize(t *testing.T) {
	got := normalize([]int{-4, -1, 8})
	want := []float64{0, 0.25, 1}
	if !slices.Equal(got, want) {
		t.Fatalf("normalize() = %v, want %v", got, want)
	}
}

func TestGenerateNormalizedHeightMapUsesFullRange(t *testing.T) {
	m, err := GenerateNormalizedHeightMap(testConfig(42))
	if err != nil {
		t.Fatal(err)
	}

	minimum, maximum := 1.0, 0.0
	for _, elevation := range m.Elevations() {
		if elevation < 0 || elevation > 1 {
			t.Fatalf("elevation = %v, want value in [0, 1]", elevation)
		}
		minimum = min(minimum, elevation)
		maximum = max(maximum, elevation)
	}
	if minimum != 0 || maximum != 1 {
		t.Fatalf("elevation range = [%v, %v], want [0, 1]", minimum, maximum)
	}
}

func TestGenerateNormalizedHeightMapWithoutFaultsIsFlat(t *testing.T) {
	m, err := GenerateNormalizedHeightMap(Config{
		Source: rand.NewPCG(7, 0),
		Width:  10,
		Height: 5,
		Faults: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.Width() != 10 || m.Height() != 5 {
		t.Fatalf("size = %dx%d, want 10x5", m.Width(), m.Height())
	}
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.Width(); x++ {
			if got := m.Elevation(x, y); got != 0 {
				t.Fatalf("elevation at (%d, %d) = %v, want 0", x, y, got)
			}
		}
	}
}

func TestGenerateNormalizedHeightMapRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want error
	}{
		{name: "nil source", cfg: Config{Width: 10, Height: 5}, want: ErrNilSource},
		{name: "zero height", cfg: Config{Source: rand.NewPCG(1, 0), Width: 2, Height: 0}, want: ErrInvalidHeight},
		{name: "wrong aspect ratio", cfg: Config{Source: rand.NewPCG(1, 0), Width: 9, Height: 5}, want: ErrInvalidWidth},
		{name: "negative faults", cfg: Config{Source: rand.NewPCG(1, 0), Width: 10, Height: 5, Faults: -1}, want: ErrInvalidFaults},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := GenerateNormalizedHeightMap(tt.cfg); err != tt.want {
				t.Fatalf("GenerateNormalizedHeightMap error = %v, want %v", err, tt.want)
			}
		})
	}
}

func testConfig(seed int64) Config {
	return Config{
		Source: rand.NewPCG(uint64(seed), 0),
		Width:  sourceWidth,
		Height: sourceHeight,
		Faults: sourceFaults,
	}
}

func rawChecksum(elevations []int) string {
	h := sha256.New()
	var buf [8]byte
	for _, elevation := range elevations {
		binary.BigEndian.PutUint64(buf[:], uint64(int64(elevation)))
		_, _ = h.Write(buf[:])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func normalizedChecksum(m *domains.NormalizedHeightMap) string {
	h := sha256.New()
	var buf [8]byte
	for _, elevation := range m.Elevations() {
		binary.BigEndian.PutUint64(buf[:], math.Float64bits(elevation))
		_, _ = h.Write(buf[:])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
