package domains

import "testing"

func TestNewVolcanicFeaturesValidatesSitesAndKinds(t *testing.T) {
	kinds := func(values ...VolcanicKind) []VolcanicKind { return values }
	for _, tt := range []struct {
		name  string
		width int
		sites []VolcanoSite
		kinds []VolcanicKind
		want  error
	}{
		{name: "no volcanism", width: 3},
		{name: "site with footprint", width: 3, sites: []VolcanoSite{{X: 1}}, kinds: kinds(VolcanicHighland, VolcanicVolcano, VolcanicHighland)},
		{name: "width", width: 0, want: ErrInvalidWidth},
		{name: "kind count", width: 3, kinds: kinds(VolcanicNone), want: ErrInvalidVolcanicKindCount},
		{name: "undeclared kind", width: 3, kinds: kinds(VolcanicNone, 9, VolcanicNone), want: ErrInvalidVolcanicKind},
		{name: "site out of bounds", width: 3, sites: []VolcanoSite{{X: 3}}, kinds: kinds(VolcanicNone, VolcanicNone, VolcanicVolcano), want: ErrInvalidVolcanoSite},
		{name: "site not marked", width: 3, sites: []VolcanoSite{{X: 0}}, kinds: kinds(VolcanicHighland, VolcanicNone, VolcanicNone), want: ErrInvalidVolcanoSite},
		{name: "duplicate site", width: 3, sites: []VolcanoSite{{X: 1}, {X: 1}}, kinds: kinds(VolcanicNone, VolcanicVolcano, VolcanicNone), want: ErrInvalidVolcanoSite},
		{name: "unsited volcano", width: 3, sites: []VolcanoSite{{X: 1}}, kinds: kinds(VolcanicVolcano, VolcanicVolcano, VolcanicNone), want: ErrUnsitedVolcano},
		{name: "sites without kinds", width: 3, sites: []VolcanoSite{{X: 1}}, want: ErrInvalidVolcanicKindCount},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewVolcanicFeatures(tt.width, 1, tt.sites, tt.kinds); err != tt.want {
				t.Fatalf("NewVolcanicFeatures error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVolcanicFeaturesAreDefensivelyCopied(t *testing.T) {
	sites := []VolcanoSite{{X: 0, Y: 1, Hotspot: true}, {X: 1, Y: 0}}
	kinds := []VolcanicKind{VolcanicHighland, VolcanicVolcano, VolcanicVolcano, VolcanicNone}
	features, err := NewVolcanicFeatures(2, 2, sites, kinds)
	if err != nil {
		t.Fatal(err)
	}
	sites[0].X = 1
	kinds[0] = VolcanicNone
	features.Sites()[1].Hotspot = true
	if features.Kind(0, 0) != VolcanicHighland || features.Kind(0, 1) != VolcanicVolcano || features.Sites()[0].X != 0 {
		t.Fatal("changing constructor input or Sites result changed the features")
	}
	if land, hotspot := features.SiteCounts(); land != 1 || hotspot != 1 {
		t.Fatalf("SiteCounts = %d land, %d hotspot, want 1 and 1", land, hotspot)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("Kind did not panic outside the grid")
		}
	}()
	features.Kind(2, 0)
}
