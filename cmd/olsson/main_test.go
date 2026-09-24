package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/maloquacious/mappe"
)

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	cmd := newCommand()
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), mappe.Version().Core()+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestGenerateCommandUsesFlags(t *testing.T) {
	var stdout bytes.Buffer
	cmd := newCommand()
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{
		"--seed", "42",
		"--width", "10",
		"--height", "5",
		"--faults", "0",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got output
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Seed != 42 || got.Width != 10 || got.Height != 5 || got.Faults != 0 {
		t.Fatalf("metadata = %+v", got)
	}
	if len(got.Elevations) != 50 {
		t.Fatalf("len(Elevations) = %d, want 50", len(got.Elevations))
	}
	for i, elevation := range got.Elevations {
		if elevation != 0 {
			t.Fatalf("Elevations[%d] = %d, want 0", i, elevation)
		}
	}
}
