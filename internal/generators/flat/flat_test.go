package flat

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
)

func TestGenerateRawPreservesDrawOrderAndCoordinates(t *testing.T) {
	source := &sequenceSource{values: []uint64{0, 1, 5, 1}}
	got, err := generateRaw(Config{
		Source:     source,
		Width:      8,
		Height:     4,
		Iterations: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if source.index != 4 {
		t.Fatalf("source draws = %d, want 4", source.index)
	}

	// The scripted draws mean bump +1, radius 2, and center (5, 1).
	wantRaised := map[point]bool{
		{4, 0}: true, {5, 0}: true, {6, 0}: true,
		{4, 1}: true, {5, 1}: true, {6, 1}: true,
		{4, 2}: true, {5, 2}: true, {6, 2}: true,
	}
	assertRawPoints(t, got, 8, 4, wantRaised, 1)
	if got[1*8+3] != 0 || got[1*8+7] != 0 || got[3*8+5] != 0 {
		t.Fatal("points on or outside the radius were raised")
	}
}

func TestGenerateRawSupportsBothSignsAndRadiusEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		values     []uint64
		wantCenter int
		wantCount  int
	}{
		{name: "positive radius one", values: []uint64{0, 0, 3, 2}, wantCenter: 1, wantCount: 1},
		{name: "negative maximum radius", values: []uint64{1, 1, 3, 2}, wantCenter: -1, wantCount: 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateRaw(Config{
				Source:     &sequenceSource{values: tt.values},
				Width:      8,
				Height:     4,
				Iterations: 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got[2*8+3] != tt.wantCenter {
				t.Fatalf("center elevation = %d, want %d", got[2*8+3], tt.wantCenter)
			}
			count := 0
			for _, elevation := range got {
				if elevation == tt.wantCenter {
					count++
				}
			}
			if count != tt.wantCount {
				t.Fatalf("changed cells = %d, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestGenerateRawClipsOrWrapsAtEveryCorner(t *testing.T) {
	centers := []struct {
		name string
		x    uint64
		y    uint64
	}{
		{name: "top left", x: 0, y: 0},
		{name: "top right", x: 7, y: 0},
		{name: "bottom left", x: 0, y: 3},
		{name: "bottom right", x: 7, y: 3},
	}
	for _, center := range centers {
		t.Run(center.name, func(t *testing.T) {
			clipped, err := generateRaw(Config{
				Source:     &sequenceSource{values: []uint64{0, 1, center.x, center.y}},
				Width:      8,
				Height:     4,
				Iterations: 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := countValue(clipped, 1); got != 4 {
				t.Fatalf("clipped changed cells = %d, want 4", got)
			}

			wrapped, err := generateRaw(Config{
				Source:     &sequenceSource{values: []uint64{0, 1, center.x, center.y}},
				Width:      8,
				Height:     4,
				Iterations: 1,
				Wrap:       true,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := countValue(wrapped, 1); got != 9 {
				t.Fatalf("wrapped changed cells = %d, want 9", got)
			}
			oppositeX := (int(center.x) + 7) % 8
			oppositeY := (int(center.y) + 3) % 4
			if got := wrapped[oppositeY*8+oppositeX]; got != 1 {
				t.Fatalf("diagonally wrapped elevation = %d, want 1", got)
			}
		})
	}
}

func TestGenerateRawAccumulatesAndCancelsOverlappingCircles(t *testing.T) {
	positive, err := generateRaw(Config{
		Source:     &sequenceSource{values: []uint64{0, 0, 3, 2, 0, 0, 3, 2}},
		Width:      8,
		Height:     4,
		Iterations: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := positive[2*8+3]; got != 2 {
		t.Fatalf("overlapping elevation = %d, want 2", got)
	}

	cancelled, err := generateRaw(Config{
		Source:     &sequenceSource{values: []uint64{0, 0, 3, 2, 1, 0, 3, 2}},
		Width:      8,
		Height:     4,
		Iterations: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := countValue(cancelled, 0); got != len(cancelled) {
		t.Fatalf("zero elevations = %d, want %d", got, len(cancelled))
	}
}

func TestNormalizeUsesFullRangeAndFlatMapsAreZero(t *testing.T) {
	if got, want := normalize([]int{-2, 0, 2}), []float64{0, 0.5, 1}; !slices.Equal(got, want) {
		t.Fatalf("normalize() = %v, want %v", got, want)
	}
	if got, want := normalize([]int{7, 7, 7}), []float64{0, 0, 0}; !slices.Equal(got, want) {
		t.Fatalf("normalize(flat) = %v, want %v", got, want)
	}
}

func TestGenerateNormalizedHeightMapWithoutIterationsIsFlatAndConsumesNoRandomness(t *testing.T) {
	source := &sequenceSource{}
	m, err := GenerateNormalizedHeightMap(Config{Source: source, Width: 7, Height: 5})
	if err != nil {
		t.Fatal(err)
	}
	if source.index != 0 {
		t.Fatalf("source draws = %d, want 0", source.index)
	}
	if m.Width() != 7 || m.Height() != 5 {
		t.Fatalf("size = %dx%d, want 7x5", m.Width(), m.Height())
	}
	for i, elevation := range m.Elevations() {
		if elevation != 0 {
			t.Fatalf("elevation %d = %v, want 0", i, elevation)
		}
	}
}

func TestGenerateNormalizedHeightMapRejectsInvalidConfigWithoutUsingSource(t *testing.T) {
	source := panicSource{}
	maxInt := int(^uint(0) >> 1)
	tests := []struct {
		name string
		cfg  Config
		want error
	}{
		{name: "nil source", cfg: Config{Width: 2, Height: 2}, want: ErrNilSource},
		{name: "width", cfg: Config{Source: source, Width: 1, Height: 2}, want: ErrInvalidWidth},
		{name: "height", cfg: Config{Source: source, Width: 2, Height: 1}, want: ErrInvalidHeight},
		{name: "iterations", cfg: Config{Source: source, Width: 2, Height: 2, Iterations: -1}, want: ErrInvalidIterations},
		{name: "overflow", cfg: Config{Source: source, Width: maxInt, Height: 2}, want: ErrMapTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := GenerateNormalizedHeightMap(tt.cfg); err != tt.want {
				t.Fatalf("GenerateNormalizedHeightMap error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestGenerateNormalizedHeightMapGolden(t *testing.T) {
	m, err := GenerateNormalizedHeightMap(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	const want = "c5b53da77e57890170cb02a1e6141ee375b8b061454bc93c0ef82f777769134a"
	if got := normalizedChecksum(m.Elevations()); got != want {
		t.Fatalf("checksum = %s, want %s", got, want)
	}
}

func TestGenerateNormalizedHeightMapIsDeterministicAcrossConcurrentCalls(t *testing.T) {
	const goroutines = 4
	want, err := GenerateNormalizedHeightMap(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	wantElevations := want.Elevations()

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := GenerateNormalizedHeightMap(testConfig())
			if err != nil {
				t.Error(err)
				return
			}
			if !slices.Equal(got.Elevations(), wantElevations) {
				t.Error("concurrent generation differed from expected output")
			}
		}()
	}
	wg.Wait()
}

func assertRawPoints(t *testing.T, elevations []int, width, height int, changed map[point]bool, want int) {
	t.Helper()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			got := elevations[y*width+x]
			if changed[point{x, y}] && got != want {
				t.Errorf("elevation at (%d, %d) = %d, want %d", x, y, got, want)
			} else if !changed[point{x, y}] && got != 0 {
				t.Errorf("elevation at (%d, %d) = %d, want 0", x, y, got)
			}
		}
	}
}

func countValue(values []int, want int) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}

func normalizedChecksum(elevations []float64) string {
	h := sha256.New()
	var buf [8]byte
	for _, elevation := range elevations {
		binary.BigEndian.PutUint64(buf[:], math.Float64bits(elevation))
		_, _ = h.Write(buf[:])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func testConfig() Config {
	return Config{
		Source:     rand.NewPCG(42, 0),
		Width:      64,
		Height:     32,
		Iterations: 100,
		Wrap:       true,
	}
}

type point struct {
	x int
	y int
}

type sequenceSource struct {
	values []uint64
	index  int
}

func (s *sequenceSource) Uint64() uint64 {
	if s.index >= len(s.values) {
		panic("sequence source exhausted")
	}
	value := s.values[s.index]
	s.index++
	return value
}

type panicSource struct{}

func (panicSource) Uint64() uint64 { panic("source used for invalid config") }

var _ rand.Source = (*sequenceSource)(nil)
var _ rand.Source = panicSource{}
