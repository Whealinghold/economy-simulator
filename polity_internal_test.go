package economysim

import (
	"errors"
	"reflect"
	"testing"
)

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

// TestPolity_ReadableOnAFailedWorld verifies that World.Polity answers on a world a
// step failed, with the same value as before the failure: the tree is a derived
// table, not state that the failure could have left half written.
//
// Requirement: FR-POL-013
func TestPolity_ReadableOnAFailedWorld(t *testing.T) {
	t.Parallel()
	c := regionMill()
	c.Modules.Polity = &PolityConfig{
		Jurisdictions: []Jurisdiction{{ID: "realm"}, {ID: "march", Parent: "realm"}},
		Members:       []Member{{Region: "home", Jurisdiction: "realm"}, {Region: "away", Jurisdiction: "march"}},
	}
	w := newMill(t, c)
	before := w.Polity()
	w.failed = errors.New("synthetic failure")
	after := w.Polity()
	if !reflect.DeepEqual(after, before) || after.Len() != 2 {
		t.Errorf("Polity() on a failed world = %+v, want the value before the failure %+v", after, before)
	}
	if got := after.Chain(0); len(got) != 2 || after.Name(got[0]) != "march" || after.Name(got[1]) != "realm" {
		t.Errorf("Chain(away) (region 0, ascending IDs) on a failed world = %v, want [march realm]", got)
	}
}
