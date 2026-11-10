package economysim_test

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// TestCatalogueOtelCopy_MatchesSpec verifies that the copy of the metric tables
// kept by the otelobserver module (which cannot read doc/) is the text of
// spec-observability.md from 6.2.1 up to 6.3.
//
// Requirement: FR-OBS-050
// Guard: otelobserver's catalogue test compares its table with the copy; without
// this test the copy could drift from the spec and the adapter with it.
func TestCatalogueOtelCopy_MatchesSpec(t *testing.T) {
	t.Parallel()
	spec, err := os.ReadFile("doc/specs/spec-observability.md")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	copyText, err := os.ReadFile("otelobserver/testdata/catalogue_copy.md")
	if err != nil {
		t.Fatalf("read copy: %v", err)
	}
	lines := strings.Split(string(spec), "\n")
	from := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "#### 6.2.1") })
	to := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "### 6.3") })
	if from < 0 || to <= from {
		t.Fatalf("spec headings 6.2.1 and 6.3 at lines %d, %d, want both in order", from, to)
	}
	if want := strings.Join(lines[from:to], "\n") + "\n"; string(copyText) != want {
		t.Errorf("otelobserver/testdata/catalogue_copy.md differs from spec-observability.md 6.2 (%d bytes, want %d): copy the spec text again", len(copyText), len(want))
	}
}

// TestAllEvents_MatchEventGo verifies that the list of event names kept by the
// otelobserver module is the set of types of event.go that implement Event.
//
// Requirement: FR-OBS-052
// Guard: otelobserver's exhaustiveness test sorts every event into handled or
// ignored; a new event missing from its list would be in neither.
func TestAllEvents_MatchEventGo(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("otelobserver/testdata/all_events.txt")
	if err != nil {
		t.Fatalf("read list: %v", err)
	}
	if got, want := strings.Fields(string(data)), sourceEvents(t); !slices.Equal(got, want) {
		t.Errorf("otelobserver/testdata/all_events.txt = %v, want the isEvent types of event.go %v", got, want)
	}
}
