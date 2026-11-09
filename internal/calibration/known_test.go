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
