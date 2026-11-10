package calibration

import "testing"

// TestIndex_FindByPath verifies that a configuration path is matched segment by
// segment: a wildcard stands for one instance, a path of another length or with
// another fixed segment does not match, and a registry key is found directly.
//
// Requirement: FR-CLB-050
func TestIndex_FindByPath(t *testing.T) {
	t.Parallel()
	x := newIndex(checkKnown())
	tests := []struct {
		key  string
		want string
		ok   bool
	}{
		{"World.CostFloor", "K-BEL12", true},
		{"role.baker.target_ratio", "K-AGT01", true},
		{"role.target_ratio", "", false},         // one segment short
		{"role.baker.x.target_ratio", "", false}, // one segment long
		{"region.baker.target_ratio", "", false}, // another fixed segment
		{"role.baker.expected_ratio", "", false}, // another last segment
	}
	for _, tt := range tests {
		got, ok := x.find(tt.key)
		if ok != tt.ok || got.ID != tt.want {
			t.Errorf("find(%q) = %q, %v, want %q, %v", tt.key, got.ID, ok, tt.want, tt.ok)
		}
	}
	// A pattern whose key is no row finds nothing, and two rows with one key keep the lowest ID.
	k := checkKnown()
	k.Paths = map[string]string{"a.*.b": "No.Row"}
	k.Knobs["K-AAA01"] = KnobInfo{Class: 'K', Key: "World.CostFloor"}
	y := newIndex(k)
	if _, ok := y.find("a.x.b"); ok {
		t.Errorf("find(a.x.b) ok = true, want false for a pattern with no row")
	}
	if got, _ := y.find("World.CostFloor"); got.ID != "K-AAA01" {
		t.Errorf("find(World.CostFloor) = %s, want the lowest ID K-AAA01", got.ID)
	}
}

// TestIndex_MostSpecificPatternWins verifies that when two path patterns match one
// configuration path, the one with fewer wildcards is tried first, whatever the map
// order, and that equal wildcard counts fall back to byte order.
//
// Requirement: FR-CLB-050
func TestIndex_MostSpecificPatternWins(t *testing.T) {
	t.Parallel()
	k := checkKnown()
	k.Knobs["K-ZZZ01"] = KnobInfo{Class: 'K', Key: "Gen.Row"}
	k.Knobs["K-ZZZ02"] = KnobInfo{Class: 'K', Key: "Spec.Row"}
	// "a.*.*" and "a.x.*" both match a.x.b; the second has one wildcard, so it is tried first.
	k.Paths = map[string]string{"a.*.*": "Gen.Row", "a.x.*": "Spec.Row"}
	for range 20 { // map iteration order is random; the result must not depend on it
		if got, ok := newIndex(k).find("a.x.b"); !ok || got.ID != "K-ZZZ02" {
			t.Fatalf("find(a.x.b) = %q, %v, want K-ZZZ02 (the pattern with fewer wildcards)", got.ID, ok)
		}
	}
	if x := newIndex(k); x.patterns[0] != "a.x.*" || x.patterns[1] != "a.*.*" {
		t.Errorf("patterns = %v, want [a.x.* a.*.*]", x.patterns)
	}
}
