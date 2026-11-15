package settlement_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/settlement"
)

// spec121 are the Enter values of the classes of spec-settlements.md 12.1 (the
// invented names are not used: a class is an index here).
var spec121 = []int64{0, 300, 800, 2000}

// reviewTables returns the thresholds of the four-class example at the band.
func reviewTables(t *testing.T, band int64) (e, l []int64) {
	t.Helper()
	e, l, ok := settlement.Thresholds(spec121, 10_000, band)
	if !ok {
		t.Fatalf("Thresholds(%v, 10000, %d) not ok", spec121, band)
	}
	return e, l
}

// TestReview_UpThenDownExclusive verifies the two directions of the review
// (spec-settlements.md 4.3.1): a count that meets an enter threshold moves up,
// one below a leave threshold moves down, one in between keeps the class.
//
// Requirement: FR-SET-010
func TestReview_UpThenDownExclusive(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000) // e 0 300 800 2000, l 0 270 720 1800
	tests := []struct {
		name  string
		c     int
		h     int64
		want  int
		floor int
	}{
		{"meets the enter threshold", 0, 300, 1, 0},
		{"one short of the enter threshold", 0, 299, 0, 0},
		{"inside the band keeps the class", 1, 275, 1, 0},
		{"at the leave threshold keeps the class", 1, 270, 1, 0},
		{"below the leave threshold gives it up", 1, 269, 0, 0},
		{"between leave and enter of the next keeps the lower", 2, 750, 2, 0},
		{"top class kept", 3, 5000, 3, 0},
		{"down only one step per threshold passed", 3, 1000, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := settlement.Review(tt.c, tt.floor, tt.h, e, l); got != tt.want {
				t.Errorf("Review(c %d, floor %d, h %d) = %d, want %d", tt.c, tt.floor, tt.h, got, tt.want)
			}
		})
	}
}

// TestReview_MovesMoreThanOneClass verifies that a flood of households moves a
// region up more than one class in one review, and a plague down more than one
// (spec-settlements.md 4.3.1).
//
// Requirement: FR-SET-010
func TestReview_MovesMoreThanOneClass(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000)
	if got := settlement.Review(0, 0, 2500, e, l); got != 3 {
		t.Errorf("Review(0, 0, 2500) = %d, want 3 (up through 1, 2, 3)", got)
	}
	if got := settlement.Review(3, 0, 10, e, l); got != 0 {
		t.Errorf("Review(3, 0, 10) = %d, want 0 (down through 2, 1, 0)", got)
	}
}

// TestReview_NeverBelowFloor verifies that the down loop stops at the floor, that
// a class below the floor is raised to it and that Target reports the floor.
//
// Requirement: FR-SET-007
func TestReview_NeverBelowFloor(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000)
	// north of 12.1: floor 1, 150 households, L of class 1 is 270: the count alone gives class 0.
	if got := settlement.Review(1, 1, 150, e, l); got != 1 {
		t.Errorf("Review(1, floor 1, 150) = %d, want 1", got)
	}
	if got := settlement.Review(0, 1, 150, e, l); got != 1 {
		t.Errorf("Review(0, floor 1, 150) = %d, want 1 (raised to the floor)", got)
	}
}

// TestTarget_AtFloor verifies the target and the floor flag of a snapshot row
// (spec-settlements.md 4.3.3, 10.5): the flag is true only at a floor above the
// first class when the count alone is below the leave threshold.
//
// Requirement: FR-SET-026
func TestTarget_AtFloor(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000)
	tests := []struct {
		name        string
		c, floor    int
		h           int64
		want        int
		wantAtFloor bool
	}{
		{"north: the count alone gives the first class", 1, 1, 150, 1, true},
		{"inside the band", 1, 1, 275, 1, false},
		{"floor 0 is no floor", 0, 0, 10, 0, false},
		{"count above the floor", 1, 1, 400, 1, false},
		{"a floor below the class is not at the floor", 2, 1, 100, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, at := settlement.Target(tt.c, tt.floor, tt.h, e, l); got != tt.want || at != tt.wantAtFloor {
				t.Errorf("Target(%d, %d, %d) = %d, %v, want %d, %v", tt.c, tt.floor, tt.h, got, at, tt.want, tt.wantAtFloor)
			}
		})
	}
}

// TestReview_InsideBandUnlessFloor verifies, for every count from 0 to 3 000, bands
// 0 and 1 000, no floor and floors 0, 1 and 2, from every starting class, that the
// result is inside its band: below the next enter threshold, and at or above its own
// leave threshold unless it is the floor (spec-settlements.md 4.3.2, the model of
// 14.1 as a Go test).
//
// Requirement: FR-SET-010
func TestReview_InsideBandUnlessFloor(t *testing.T) {
	t.Parallel()
	for _, band := range []int64{0, 1_000} {
		e, l := reviewTables(t, band)
		for floor := 0; floor <= 2; floor++ {
			for c := 0; c < len(e); c++ {
				for h := int64(0); h <= 3_000; h++ {
					got := settlement.Review(c, floor, h, e, l)
					if got < floor {
						t.Fatalf("Review(c %d, floor %d, h %d, band %d) = %d, below the floor", c, floor, h, band, got)
					}
					if got+1 < len(e) && h >= e[got+1] {
						t.Fatalf("Review(c %d, floor %d, h %d, band %d) = %d, count meets the next enter threshold %d", c, floor, h, band, got, e[got+1])
					}
					if got != floor && h < l[got] {
						t.Fatalf("Review(c %d, floor %d, h %d, band %d) = %d, count below its leave threshold %d", c, floor, h, band, got, l[got])
					}
				}
			}
		}
	}
}

// TestReview_Total verifies the guard clauses: no class, thresholds of different
// lengths and a class outside the list give a class in range and never panic.
//
// Guard: the root passes validated input, so these only keep the function total; a
// panic here would be a crash of the tick on a state a restore should have refused.
func TestReview_Total(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000)
	if got := settlement.Review(0, 0, 5, nil, nil); got != 0 {
		t.Errorf("Review with no class = %d, want 0", got)
	}
	if got := settlement.Review(0, 0, 5, e, l[:2]); got != 0 {
		t.Errorf("Review with unequal tables = %d, want 0", got)
	}
	if got := settlement.Review(99, 0, 5000, e, l); got != 3 {
		t.Errorf("Review(99, ...) = %d, want 3 (clamped to the top class)", got)
	}
	if got := settlement.Review(-4, 0, 5, e, l); got != 0 {
		t.Errorf("Review(-4, ...) = %d, want 0", got)
	}
	if got, at := settlement.Target(0, 0, 5, nil, nil); got != 0 || at {
		t.Errorf("Target with no class = %d, %v, want 0, false", got, at)
	}
}

// FuzzReview verifies that Review never panics and returns a class inside
// [floor, K) for any thresholds the constructor accepts, and inside its band.
//
// Requirement: NFR-SET-004
func FuzzReview(f *testing.F) {
	f.Add(1, 1, int64(150), int64(1_000), int64(10_000))
	f.Add(0, 0, int64(5000), int64(0), int64(5_000))
	f.Fuzz(func(t *testing.T, c, floor int, h, band, scale int64) {
		e, l, ok := settlement.Thresholds(spec121, scale, band)
		if !ok {
			return
		}
		got := settlement.Review(c, floor, h, e, l)
		f0 := max(0, min(floor, len(e)-1))
		if got < f0 || got >= len(e) {
			t.Fatalf("Review(%d, %d, %d) = %d, want in [%d, %d)", c, floor, h, got, f0, len(e))
		}
		if got+1 < len(e) && h >= e[got+1] {
			t.Fatalf("Review(%d, %d, %d) = %d, count meets the next enter threshold", c, floor, h, got)
		}
	})
}

// TestInitial_ClassIsHighestReached verifies the construction class
// (spec-settlements.md 4.5.1): the highest class whose enter threshold the count
// meets, never below the floor, with no band.
//
// Requirement: FR-SET-013
func TestInitial_ClassIsHighestReached(t *testing.T) {
	t.Parallel()
	e, _ := reviewTables(t, 1_000)
	tests := []struct {
		name  string
		h     int64
		floor int
		want  int
	}{
		{"empty region", 0, 0, 0},
		{"just below a threshold", 299, 0, 0},
		{"exactly the threshold has no band", 300, 0, 1},
		{"in the top class", 2000, 0, 3},
		{"floor above the count", 150, 1, 1},
		{"floor below the count", 900, 1, 2},
		{"floor above the list is clamped", 5, 9, 3},
		{"negative floor is clamped", 5, -2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := settlement.Initial(tt.h, e, tt.floor); got != tt.want {
				t.Errorf("Initial(%d, %v, %d) = %d, want %d", tt.h, e, tt.floor, got, tt.want)
			}
		})
	}
	if got := settlement.Initial(5, nil, 0); got != 0 {
		t.Errorf("Initial(5, nil, 0) = %d, want 0", got)
	}
}

// TestDecide_HoldsAndForgets verifies the dwell (spec-settlements.md 4.4): a change
// due before Dwell ticks since the last change is held with the ticks left, one due
// at or after Dwell is applied, and no change due is Same.
//
// Requirement: FR-SET-011, FR-SET-012
func TestDecide_HoldsAndForgets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		c, target          int
		since, tick, dwell int64
		want               settlement.Outcome
	}{
		{"nothing due", 1, 1, 0, 5, 30, settlement.Outcome{Kind: settlement.Same}},
		// 30 - (5 - 0) = 25 ticks left, as the first tick of the plague example.
		{"held at tick 5", 1, 0, 0, 5, 30, settlement.Outcome{Kind: settlement.Held, TicksLeft: 25}},
		{"held at tick 29 has 1 left", 1, 0, 0, 29, 30, settlement.Outcome{Kind: settlement.Held, TicksLeft: 1}},
		{"released at tick Dwell", 1, 0, 0, 30, 30, settlement.Outcome{Kind: settlement.Changed}},
		{"runs from the last change", 0, 1, 40, 60, 30, settlement.Outcome{Kind: settlement.Held, TicksLeft: 10}},
		{"no dwell changes at once", 0, 1, 7, 7, 0, settlement.Outcome{Kind: settlement.Changed}},
		{"tick before since is held with the full dwell", 0, 1, 10, 4, 30, settlement.Outcome{Kind: settlement.Held, TicksLeft: 30}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := settlement.Decide(tt.c, tt.target, tt.since, tt.tick, tt.dwell)
			if !ok || got != tt.want {
				t.Errorf("Decide(%d, %d, since %d, t %d, dwell %d) = %v, %v, want %v, true", tt.c, tt.target, tt.since, tt.tick, tt.dwell, got, ok, tt.want)
			}
		})
	}
}

// TestDecide_OverflowIsReported verifies that a tick difference that does not fit
// int64 is reported, not wrapped.
//
// Guard: architecture.md 3.5; a wrapped difference could release a held change.
func TestDecide_OverflowIsReported(t *testing.T) {
	t.Parallel()
	const minInt64, maxInt64 = -1 << 63, 1<<63 - 1
	if _, ok := settlement.Decide(0, 1, minInt64, maxInt64, 30); ok {
		t.Error("Decide(since MinInt64, t MaxInt64) ok = true, want false")
	}
}
