package otelobserver

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestIntern_CapDropsBeyond verifies that an open string gets a dense index up to
// the cap, the same string the same index, and every new value beyond the cap
// -1, so a flood of distinct values cannot grow the series (plan 2-22 D645).
//
// Requirement: FR-OBS-052
func TestIntern_CapDropsBeyond(t *testing.T) {
	in := newIntern("economysim.reason", 3)
	for i, s := range []string{"a", "b", "c"} {
		if got := in.id(s); got != i {
			t.Errorf("id(%q) = %d, want %d", s, got, i)
		}
	}
	if got := in.id("b"); got != 1 {
		t.Errorf("id(\"b\") again = %d, want 1", got)
	}
	if got := in.id("d"); got != -1 {
		t.Errorf("id(\"d\") beyond the cap = %d, want -1", got)
	}
	if got := len(in.kv); got != 3 {
		t.Errorf("len(kv) = %d, want 3", got)
	}
}

// TestIntern_CapOfRejectionReasons verifies the production cap is the width of
// the dense rejection array.
//
// Guard: the array index is commodity*maxReasons+reason, so a larger cap would
// make two series share a cell.
func TestIntern_CapOfRejectionReasons(t *testing.T) {
	n := newNames(economysim.Names{})
	if n.reasons.max != maxReasons {
		t.Errorf("reasons cap = %d, want %d", n.reasons.max, maxReasons)
	}
}
