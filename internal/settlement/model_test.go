package settlement_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/settlement"
)

// modelReview is the review of spec-settlements.md 4.3 written a second way, so a
// test can compare it with settlement.Review over a whole domain. Up: the highest
// class whose enter threshold the count meets, if that is above the start. Down,
// only when nothing moved up: the highest class at or below the start that is the
// floor or whose leave threshold the count meets. It relies on the thresholds being
// non-decreasing, which Thresholds guarantees for sorted Enter values.
func modelReview(c, floor int, h int64, e, l []int64) int {
	start := max(c, floor)
	top := start
	for k := start + 1; k < len(e); k++ {
		if h >= e[k] {
			top = k
		}
	}
	if top > start {
		return top
	}
	for k := start; k > floor; k-- {
		if h >= l[k] {
			return k
		}
	}
	return floor
}

// TestReview_AgreesWithModel verifies that Review equals an independent
// formulation of the rule for every count from 0 to 3 000, bands 0, 1 000 and 5 000,
// floors 0 to 3 and every starting class. It pins what the invariant test cannot:
// that the class is the right one inside its band, not only some class in it, and
// that a class already inside its band is kept (no flip-flop).
//
// Requirement: FR-SET-010
//
// Guard: the invariant "inside its band" is also met by a review that jumps to the
// lowest or highest class that fits; only an exact comparison catches that.
func TestReview_AgreesWithModel(t *testing.T) {
	t.Parallel()
	for _, band := range []int64{0, 1_000, 5_000} {
		e, l := reviewTables(t, band)
		for floor := 0; floor < len(e); floor++ {
			for c := 0; c < len(e); c++ {
				for h := int64(0); h <= 3_000; h++ {
					got, want := settlement.Review(c, floor, h, e, l), modelReview(c, floor, h, e, l)
					if got != want {
						t.Fatalf("Review(c %d, floor %d, h %d, band %d) = %d, want %d (e %v, l %v)", c, floor, h, band, got, want, e, l)
					}
				}
			}
		}
	}
}

// TestReview_StableInsideItsBand verifies that a class whose band holds the count is
// kept by a review, for every class and every count in [leave, next enter), and that
// a second review of any result returns the result (the review is idempotent).
//
// Requirement: FR-SET-010
//
// Guard: hysteresis is the point of the band; a review that moved a class whose count
// sits in its band would make a region flap at a threshold.
func TestReview_StableInsideItsBand(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000)
	for c := 0; c < len(e); c++ {
		hi := int64(3_000)
		if c+1 < len(e) {
			hi = e[c+1] - 1
		}
		for h := l[c]; h <= hi; h++ {
			if got := settlement.Review(c, 0, h, e, l); got != c {
				t.Fatalf("Review(c %d, floor 0, h %d) = %d, want %d (count inside [%d, %d])", c, h, got, c, l[c], hi)
			}
		}
	}
	for c := 0; c < len(e); c++ {
		for h := int64(0); h <= 3_000; h += 7 {
			first := settlement.Review(c, 1, h, e, l)
			if again := settlement.Review(first, 1, h, e, l); again != first {
				t.Fatalf("Review(Review(c %d, h %d)) = %d, want %d (idempotent)", c, h, again, first)
			}
		}
	}
}

// TestDecide_AppliesTargetOfReview verifies the pair as the world uses it: a count
// that crosses an enter threshold is Changed when the dwell has run and Held, with
// the ticks left, when it has not; a count that falls back inside the band before the
// release is Same (the held change is forgotten).
//
// Requirement: FR-SET-011, FR-SET-012
func TestDecide_AppliesTargetOfReview(t *testing.T) {
	t.Parallel()
	e, l := reviewTables(t, 1_000)
	const dwell, since = 30, 10
	// At tick 25, 25-10 = 15 ticks have run of 30, so 30-15 = 15 are left.
	up := settlement.Review(1, 0, 800, e, l)
	if got, ok := settlement.Decide(1, up, since, 25, dwell); !ok || got != (settlement.Outcome{Kind: settlement.Held, TicksLeft: 15}) {
		t.Errorf("Decide(1, %d, since %d, t 25, dwell %d) = %v, %v, want Held with 15 left", up, since, dwell, got, ok)
	}
	if got, ok := settlement.Decide(1, up, since, 40, dwell); !ok || got.Kind != settlement.Changed {
		t.Errorf("Decide(1, %d, since %d, t 40, dwell %d) = %v, %v, want Changed (30 elapsed)", up, since, dwell, got, ok)
	}
	back := settlement.Review(1, 0, 500, e, l)
	if got, ok := settlement.Decide(1, back, since, 25, dwell); !ok || got.Kind != settlement.Same {
		t.Errorf("Decide(1, %d, ...) after the count fell back = %v, %v, want Same", back, got, ok)
	}
}
