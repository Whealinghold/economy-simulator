package economysim

import "testing"

// TestPolityCovers_UnknownJurisdictionCoversNothing checks that a membership row
// naming a jurisdiction that is not defined is not read as jurisdiction 0: here
// index 0 ("a") has parent "b", so a lookup that defaulted to 0 would climb to "b"
// and report that "b" covers the region.
//
// Requirement: FR-POL-005
func TestPolityCovers_UnknownJurisdictionCoversNothing(t *testing.T) {
	t.Parallel()
	m := &PolityConfig{Jurisdictions: []Jurisdiction{{ID: "a", Parent: "b"}, {ID: "b"}}}
	idx := polityIndex(m)
	direct := map[string]string{"r": "ghost"}
	if polityCovers(m, idx, direct, "b", "r") {
		t.Error("polityCovers(b, r) = true for a region whose jurisdiction is not defined, want false")
	}
	direct["r"] = "a"
	if !polityCovers(m, idx, direct, "b", "r") {
		t.Error("polityCovers(b, r) = false for a region in a child of b, want true")
	}
}
