package saving_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
	"github.com/Whealinghold/economy-sim/internal/saving"
)

// TestCost_OpportunityNotUpkeep checks the cost of 16.1 (360 + 40) and 16.4
// (200 + 20): inputs plus the opportunity term, and nothing for upkeep.
//
// Requirement: FR-SAV-013
func TestCost_OpportunityNotUpkeep(t *testing.T) {
	t.Parallel()
	in := saving.Inputs(1, []int64{30, 10}, []int64{8, 12})
	if in != 360 {
		t.Fatalf("Inputs = %d, want 360", in)
	}
	if got := saving.Cost(in, saving.Opportunity(80_000, 5), maxMoney); got != 400 {
		t.Errorf("Cost of 16.1 = %d, want 400", got)
	}
	if got := saving.Cost(200, saving.Opportunity(50_000, 4), maxMoney); got != 220 {
		t.Errorf("Cost of 16.4 = %d, want 220", got)
	}
	if got := saving.Inputs(5, []int64{4}, []int64{8}); got != 160 {
		t.Errorf("Inputs of the repair of 16.2 = %d, want 160", got)
	}
}

// TestPasses_128BitBoundary checks the test of 5.5.1 at its edge, with the
// examples of 16.1 and 16.4, and a product that leaves int64.
//
// Requirement: FR-SAV-014
func TestPasses_128BitBoundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                          string
		benefit, cost, hurdle, expect int64
		want                          bool
	}{
		{"16.1 passes", 2_160, 400, 15_000, 0, true},
		{"16.4 passes", 1_500, 220, 15_000, 0, true},
		{"16.4 at PerTick 3 fails", 300, 220, 15_000, 0, false},
		{"exactly at the hurdle", 330, 220, 15_000, 0, true},
		{"one below the hurdle", 329, 220, 15_000, 0, false},
		{"benefit zero never passes", 0, 0, 10_000, 0, false},
		{"product above int64", 900_000_000_000_000_000, 600_000_000_000_000_000, 15_000, 0, true},
		{"just below at 128 bits", 899_999_999_999_999_999, 600_000_000_000_000_000, 15_000, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := saving.Passes(tc.benefit, tc.cost, tc.hurdle, maxMoney); got != tc.want {
				t.Errorf("Passes(%d, %d, %d) = %v, want %v", tc.benefit, tc.cost, tc.hurdle, got, tc.want)
			}
		})
	}
}

// TestHurdle_WeylSpread checks the hurdles of 16.1 at a spread of 2 000: agent
// 3 has 16 708, agent 7 15 652 and agent 11 16 596, each in [base, base + spread).
//
// Requirement: FR-SAV-015
func TestHurdle_WeylSpread(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ id, want int64 }{{3, 16_708}, {7, 15_652}, {11, 16_596}} {
		got := saving.Hurdle(15_000, 2_000, ordering.Weyl32(tc.id))
		if got != tc.want {
			t.Errorf("Hurdle for agent %d = %d, want %d", tc.id, got, tc.want)
		}
	}
	if got := saving.Hurdle(15_000, 0, 123); got != 15_000 {
		t.Errorf("Hurdle with no spread = %d, want 15000", got)
	}
}

// TestCommit_HalfByValue checks that a household holding half of its inputs by
// value is committed, and the lowered hurdle of 16.1 (12 000).
//
// Requirement: FR-SAV-016
func TestCommit_HalfByValue(t *testing.T) {
	t.Parallel()
	if !saving.Committed(180, 360) {
		t.Error("half by value must be committed")
	}
	if saving.Committed(179, 360) {
		t.Error("just under half must not be committed")
	}
	if got := saving.Lowered(15_000, 2_000); got != 12_000 {
		t.Errorf("Lowered = %d, want 12000", got)
	}
}

// TestCommit_NoFlapWithBand checks that a price at which the test fails for an
// uncommitted household still passes for a committed one, so a household that
// holds the inputs does not sell them back (SAV-A4).
//
// Requirement: FR-SAV-016
func TestCommit_NoFlapWithBand(t *testing.T) {
	t.Parallel()
	const benefit, cost = 2_000, 400 // a ratio of 5
	if saving.Passes(benefit, cost, 52_000, maxMoney) {
		t.Fatal("setup: the full hurdle must fail")
	}
	if !saving.Passes(benefit, cost, saving.Lowered(52_000, 2_000), maxMoney) {
		t.Error("the committed household must still pass at the lowered hurdle")
	}
}

// TestChoose_BestRatioTiesRepairThenID checks the choice of 5.5.4: the greater
// benefit over cost, then a repair before a build, then the lower ID.
//
// Requirement: FR-SAV-017
func TestChoose_BestRatioTiesRepairThenID(t *testing.T) {
	t.Parallel()
	build := saving.Candidate{Benefit: 2_160, Cost: 400, ID: 3}
	repair := saving.Candidate{Benefit: 2_160, Cost: 180, Repair: true, ID: 9}
	if !saving.Better(repair, build) || saving.Better(build, repair) {
		t.Error("the repair of 16.2 (ratio 12) must beat the build of 16.1 (ratio 5.4)")
	}
	tieB := saving.Candidate{Benefit: 100, Cost: 10, ID: 1}
	tieR := saving.Candidate{Benefit: 200, Cost: 20, Repair: true, ID: 5}
	if !saving.Better(tieR, tieB) || saving.Better(tieB, tieR) {
		t.Error("an equal ratio must go to the repair")
	}
	lo := saving.Candidate{Benefit: 100, Cost: 10, ID: 2}
	hi := saving.Candidate{Benefit: 300, Cost: 30, ID: 7}
	if !saving.Better(lo, hi) || saving.Better(hi, lo) {
		t.Error("an equal ratio of one kind must go to the lower ID")
	}
	if saving.Better(lo, lo) {
		t.Error("a candidate must not be better than itself")
	}
}

// TestBounds_EveryProductInside128Bits checks the largest products of spec
// 11.3 (a benefit of MaxMoney against a cost of MaxMoney at the largest
// hurdle) without a wrap.
//
// Requirement: FR-SAV-043
func TestBounds_EveryProductInside128Bits(t *testing.T) {
	t.Parallel()
	if saving.Passes(maxMoney-1, maxMoney-1, 200_000, maxMoney) {
		t.Error("a benefit equal to the cost cannot pass a hurdle of 20")
	}
	if !saving.Passes(maxMoney-1, 1, 200_000, maxMoney) {
		t.Error("a benefit of nearly MaxMoney against a cost of 1 must pass")
	}
	a := saving.Candidate{Benefit: maxMoney - 1, Cost: maxMoney - 2, ID: 1}
	b := saving.Candidate{Benefit: maxMoney - 2, Cost: maxMoney - 1, ID: 2}
	if !saving.Better(a, b) {
		t.Error("cross-multiplication at the bound must order the two")
	}
}
