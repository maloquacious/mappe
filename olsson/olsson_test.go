package olsson

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"
	"testing"
)

const (
	sourceWidth  = 640
	sourceHeight = 320
	sourceFaults = 100
)

func TestGenerateMatchesWorldgen(t *testing.T) {
	tests := []struct {
		seed int64
		want string
	}{
		{seed: 0, want: "87783ea7670f02c18612fea0b93e5958748e93e004a448d3ea2fdbf3d5e7b3ac"},
		{seed: 0x638bb317ac47a6ba, want: "8ad30d5ff876ca922ec1262c1f7cdafd91b084e7816e71bfd789e6d97c08ad31"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("seed_%d", tt.seed), func(t *testing.T) {
			m, err := Generate(Config{
				Seed:   tt.seed,
				Width:  sourceWidth,
				Height: sourceHeight,
				Faults: sourceFaults,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := checksum(m); got != tt.want {
				t.Fatalf("checksum = %s, want source checksum %s", got, tt.want)
			}
		})
	}
}

func TestGenerateIsIndependentAcrossConcurrentCalls(t *testing.T) {
	tests := []struct {
		seed int64
		want string
	}{
		{seed: 0, want: "87783ea7670f02c18612fea0b93e5958748e93e004a448d3ea2fdbf3d5e7b3ac"},
		{seed: 0x638bb317ac47a6ba, want: "8ad30d5ff876ca922ec1262c1f7cdafd91b084e7816e71bfd789e6d97c08ad31"},
	}

	var wg sync.WaitGroup
	for _, tt := range tests {
		tt := tt
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := Generate(Config{
				Seed:   tt.seed,
				Width:  sourceWidth,
				Height: sourceHeight,
				Faults: sourceFaults,
			})
			if err != nil {
				t.Error(err)
				return
			}
			if got := checksum(m); got != tt.want {
				t.Errorf("seed %d checksum = %s, want %s", tt.seed, got, tt.want)
			}
		}()
	}
	wg.Wait()
}

func TestGenerateWithoutFaultsIsFlat(t *testing.T) {
	m, err := Generate(Config{Seed: 7, Width: 10, Height: 5, Faults: 0})
	if err != nil {
		t.Fatal(err)
	}
	if m.Width() != 10 || m.Height() != 5 {
		t.Fatalf("size = %dx%d, want 10x5", m.Width(), m.Height())
	}
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.Width(); x++ {
			if got := m.Elevation(x, y); got != 0 {
				t.Fatalf("elevation at (%d, %d) = %d, want 0", x, y, got)
			}
		}
	}
}

func TestGenerateRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "zero height", cfg: Config{Width: 2, Height: 0}},
		{name: "wrong aspect ratio", cfg: Config{Width: 9, Height: 5}},
		{name: "negative faults", cfg: Config{Width: 10, Height: 5, Faults: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Generate(tt.cfg); err == nil {
				t.Fatal("Generate returned nil error")
			}
		})
	}
}

func checksum(m *Map) string {
	h := sha256.New()
	var buf [8]byte
	for _, elevation := range m.Elevations() {
		binary.BigEndian.PutUint64(buf[:], uint64(int64(elevation)))
		_, _ = h.Write(buf[:])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
