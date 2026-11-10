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

// TestSaturate_NoWrap verifies that a sum or a difference at the limits of int64
// is held there and does not wrap to the other sign, and that a job row whose
// IDs are outside the names is not remembered between snapshots.
//
// Requirement: FR-OBS-052
func TestSaturate_NoWrap(t *testing.T) {
	const hi, lo = int64(1<<63 - 1), int64(-1 << 63)
	for _, c := range []struct{ got, want int64 }{
		{satAdd(hi, 1), hi}, {satAdd(lo, -1), lo}, {satAdd(hi, lo), -1}, {satAdd(2, 3), 5},
		{satSub(lo, 1), lo}, {satSub(hi, -1), hi}, {satSub(0, lo), hi}, {satSub(5, 3), 2},
	} {
		if c.got != c.want {
			t.Errorf("got %d, want %d", c.got, c.want)
		}
	}
	o := &Observer{}
	o.recordJobs(snapNames{recipe: []string{"r"}, region: []string{"g"}},
		[]economysim.JobRow{{Recipe: 7, Region: 0, Executions: 1}, {Recipe: 0, Region: -1, Executions: 1}})
	if len(o.jobPairs) != 0 {
		t.Errorf("jobPairs = %v, want none", o.jobPairs)
	}
}
