package domains

import "testing"

func TestHeightFieldCarriesMapAndTopology(t *testing.T) {
	heightMap, err := NewNormalizedHeightMap(2, 1, []float64{0.25, 0.75})
	if err != nil {
		t.Fatal(err)
	}
	topology := GridTopology{WrapEastWest: true}
	field, err := NewHeightField(heightMap, topology)
	if err != nil {
		t.Fatal(err)
	}

	if field.HeightMap() != heightMap {
		t.Fatal("HeightMap did not return the immutable source map")
	}
	if field.Topology() != topology {
		t.Fatalf("Topology = %+v, want %+v", field.Topology(), topology)
	}
	if field.Width() != 2 || field.Height() != 1 || field.Elevation(1, 0) != 0.75 {
		t.Fatalf("field data = %dx%d %v, want 2x1 0.75", field.Width(), field.Height(), field.Elevation(1, 0))
	}

	elevations := field.Elevations()
	elevations[1] = 0
	if field.Elevation(1, 0) != 0.75 {
		t.Fatal("changing Elevations result changed the field")
	}
}

func TestNewHeightFieldRejectsNilMap(t *testing.T) {
	if _, err := NewHeightField(nil, GridTopology{}); err != ErrNilHeightMap {
		t.Fatalf("NewHeightField error = %v, want %v", err, ErrNilHeightMap)
	}
}
