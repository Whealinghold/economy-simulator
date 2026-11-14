package settlement_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/settlement"
)

// TestEdges_ThresholdsInputs verifies the boundaries of Thresholds that the other
// tests leave to the guard: a band of exactly 10 000 (the leave threshold is 0), an
// Enter of 0 after the first class, and an Enter written for the first class, which is
// ignored (the first class is always 0).
//
// Requirement: FR-SET-008
func TestEdges_ThresholdsInputs(t *testing.T) {
	t.Parallel()
	e, l, ok := settlement.Thresholds([]int64{0, 10}, 10_000, 10_000)
	if !ok || !slices.Equal(e, []int64{0, 10}) || !slices.Equal(l, []int64{0, 0}) {
		t.Errorf("Thresholds(band 10000) = %v, %v, %v, want [0 10], [0 0], true", e, l, ok)
	}
	if e, l, ok := settlement.Thresholds([]int64{0, 0}, 10_000, 1_000); !ok || !slices.Equal(e, []int64{0, 0}) || !slices.Equal(l, []int64{0, 0}) {
		t.Errorf("Thresholds([0 0]) = %v, %v, %v, want zeros, true", e, l, ok)
	}
	if e, _, ok := settlement.Thresholds([]int64{5, 10}, 10_000, 1_000); !ok || e[0] != 0 {
		t.Errorf("Thresholds([5 10]) e = %v, %v, want e[0] = 0 whatever the first Enter", e, ok)
	}
}

// TestEdges_ReviewTablesOfDifferentLength verifies that Review and Target take the
// guard for thresholds of different lengths in either direction and give the first
// class, so a table that does not match is never indexed past its end.
//
// Guard: the guard keeps the function total; a mutation of it would index a short
// table, or follow the enter thresholds alone, in a state a restore should refuse.
func TestEdges_ReviewTablesOfDifferentLength(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000)
	long := append(slices.Clone(l), 9_999)
	if got := settlement.Review(0, 0, 5_000, e, l[:2]); got != 0 {
		t.Errorf("Review(short leave table, h 5000) = %d, want 0", got)
	}
	if got := settlement.Review(0, 0, 5_000, e, long); got != 0 {
		t.Errorf("Review(long leave table, h 5000) = %d, want 0", got)
	}
	if got, at := settlement.Target(1, 1, 5, e, l[:1]); got != 0 || at {
		t.Errorf("Target(short leave table) = %d, %v, want 0, false", got, at)
	}
	if got, at := settlement.Target(1, 1, 5, e, long); got != 0 || at {
		t.Errorf("Target(long leave table) = %d, %v, want 0, false", got, at)
	}
	if got := settlement.Review(0, 0, 5_000, e[:1], l[:1]); got != 0 {
		t.Errorf("Review(one class) = %d, want 0", got)
	}
}

// TestEdges_TargetAtFloorBoundary verifies the three edges of the floor flag: a count
// exactly at the leave threshold is inside the band, a class below the floor is not
// the floor, and the floor above the list is clamped.
//
// Requirement: FR-SET-026
func TestEdges_TargetAtFloorBoundary(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000) // l[1] = 270
	if got, at := settlement.Target(1, 1, 270, e, l); got != 1 || at {
		t.Errorf("Target(h at the leave threshold) = %d, %v, want 1, false", got, at)
	}
	if got, at := settlement.Target(1, 1, 269, e, l); got != 1 || !at {
		t.Errorf("Target(h one below the leave threshold) = %d, %v, want 1, true", got, at)
	}
	if got, at := settlement.Target(0, 1, 150, e, l); got != 1 || at {
		t.Errorf("Target(class below the floor) = %d, %v, want 1, false", got, at)
	}
	if got, at := settlement.Target(0, 0, -5, e, l); got != 0 || at {
		t.Errorf("Target(floor 0, negative count) = %d, %v, want 0, false (floor 0 is no floor)", got, at)
	}
}

// TestEdges_DecideReleasedAfterDwell verifies that a change due more than Dwell ticks
// after the last change is applied, not only one due exactly at Dwell.
//
// Requirement: FR-SET-011
func TestEdges_DecideReleasedAfterDwell(t *testing.T) {
	t.Parallel()
	if got, ok := settlement.Decide(1, 0, 0, 35, 30); !ok || got.Kind != settlement.Changed {
		t.Errorf("Decide(due 35 ticks after the last change, dwell 30) = %v, %v, want Changed", got, ok)
	}
}

// TestEdges_TableBoundaries verifies the limits of NewTable: no limit names at all,
// and a flag or a limit past the word or the list.
//
// Requirement: FR-SET-005, FR-SET-020
func TestEdges_TableBoundaries(t *testing.T) {
	t.Parallel()
	if tb, ok := settlement.NewTable(0, []settlement.Class{{Flags: []int{1}}}); !ok || !tb.Has(0, 1) {
		t.Errorf("NewTable(no limit names) = %v, %v, want ok with the flag", tb, ok)
	}
	for name, c := range map[string]settlement.Class{
		"flag 65":        {Flags: []int{65}},
		"limit index 5":  {Limits: []settlement.Limit{{Index: 5}}},
		"limit index 2 ": {Limits: []settlement.Limit{{Index: 2}}},
	} {
		if _, ok := settlement.NewTable(2, []settlement.Class{c}); ok {
			t.Errorf("NewTable(%s) ok = true, want false", name)
		}
	}
}

// TestEdges_TableReads verifies Has and Limit on a table with flags twice in one
// class, a class with several flags, the last limit of the last class, and reads far
// past the classes, the word and the limit list.
//
// Requirement: FR-SET-005, FR-SET-020
func TestEdges_TableReads(t *testing.T) {
	t.Parallel()
	tb, ok := settlement.NewTable(2, []settlement.Class{{}, {}, {Flags: []int{3, 3, 5, 0}, Limits: []settlement.Limit{{Index: 1, Value: 7}}}})
	if !ok {
		t.Fatal("NewTable not ok")
	}
	for _, f := range []int{0, 3, 5} {
		if !tb.Has(2, f) {
			t.Errorf("Has(2, %d) = false, want true (flags {3 3 5 0})", f)
		}
	}
	if v, ok := tb.Limit(2, 1); !ok || v != 7 {
		t.Errorf("Limit(2, 1) = %d, %v, want 7, true", v, ok)
	}
	if tb.Has(7, 0) || tb.Has(2, 70) {
		t.Error("Has far past the classes or the word = true, want false")
	}
	if _, ok := tb.Limit(7, 0); ok {
		t.Error("Limit(7, 0) ok = true, want false")
	}
	if _, ok := tb.Limit(2, 5); ok {
		t.Error("Limit(2, 5) ok = true, want false")
	}
}

// TestEdges_HeadroomLargestNegative verifies that the most negative limit does not
// wrap when households are subtracted from it.
//
// Requirement: FR-SET-032
//
// Guard: architecture.md 3.5; limit - households with limit MinInt64 wraps to a huge
// room if the negative limit is not clamped first.
func TestEdges_HeadroomLargestNegative(t *testing.T) {
	t.Parallel()
	if got := settlement.Headroom(math.MinInt64, 1); got != 0 {
		t.Errorf("Headroom(MinInt64, 1) = %d, want 0", got)
	}
}
