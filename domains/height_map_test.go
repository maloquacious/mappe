package domains

import (
	"math"
	"slices"
	"testing"
)

func TestNewNormalizedHeightMap(t *testing.T) {
	elevations := []float64{0, 0.25, 0.75, 1}
	m, err := NewNormalizedHeightMap(2, 2, elevations)
	if err != nil {
		t.Fatal(err)
	}
	if m.Width() != 2 || m.Height() != 2 {
		t.Fatalf("size = %dx%d, want 2x2", m.Width(), m.Height())
	}
	if got := m.Elevation(0, 1); got != 0.75 {
		t.Fatalf("Elevation(0, 1) = %v, want 0.75", got)
	}

	elevations[2] = 0
	if got := m.Elevation(0, 1); got != 0.75 {
		t.Fatalf("changing constructor input changed elevation to %v", got)
	}
	got := m.Elevations()
	got[2] = 0
	if slices.Equal(got, m.Elevations()) {
		t.Fatal("changing Elevations result changed map")
	}
}

func TestNormalizedHeightMapElevationPanicsOutsideMap(t *testing.T) {
	m, err := NewNormalizedHeightMap(2, 2, []float64{0, 0.25, 0.75, 1})
	if err != nil {
		t.Fatal(err)
	}

	for _, point := range []struct {
		name string
		x    int
		y    int
	}{
		{name: "left", x: -1, y: 0},
		{name: "right", x: 2, y: 0},
		{name: "above", x: 0, y: -1},
		{name: "below", x: 0, y: 2},
	} {
		t.Run(point.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("Elevation did not panic")
				}
			}()
			m.Elevation(point.x, point.y)
		})
	}
}

func TestNewNormalizedHeightMapRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		elevations []float64
		want       error
	}{
		{name: "width", width: 0, height: 1, want: ErrInvalidWidth},
		{name: "height", width: 1, height: 0, want: ErrInvalidHeight},
		{name: "count", width: 2, height: 1, elevations: []float64{0}, want: ErrInvalidElevationCount},
		{name: "negative", width: 1, height: 1, elevations: []float64{-0.01}, want: ErrInvalidElevation},
		{name: "above one", width: 1, height: 1, elevations: []float64{1.01}, want: ErrInvalidElevation},
		{name: "NaN", width: 1, height: 1, elevations: []float64{math.NaN()}, want: ErrInvalidElevation},
		{name: "infinity", width: 1, height: 1, elevations: []float64{math.Inf(1)}, want: ErrInvalidElevation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewNormalizedHeightMap(tt.width, tt.height, tt.elevations); err != tt.want {
				t.Fatalf("NewNormalizedHeightMap error = %v, want %v", err, tt.want)
			}
		})
	}
}
