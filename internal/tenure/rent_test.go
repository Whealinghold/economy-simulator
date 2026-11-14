package tenure_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// TestRentInForce_ScaleTruncatedMinOneZeroScale verifies the rent in force
// (spec-land.md 5.2.2): posted * scale / 10 000 truncated toward zero, at least
// 1 when both are positive, 0 for a scale of 0, and a refusal for a negative
// number or a product beyond int64.
//
// Requirement: FR-LND-015
func TestRentInForce_ScaleTruncatedMinOneZeroScale(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		posted, scale int64
		want          int64
		ok            bool
	}{
		{"unit scale", 40, 10_000, 40, true},
		{"half", 41, 5_000, 20, true},  // 41 * 5000 / 10000 = 20.5, truncated to 20
		{"minimum one", 3, 1, 1, true}, // 3 * 1 / 10000 = 0, raised to 1
		{"zero scale", 40, 0, 0, true},
		{"zero posted", 0, 10_000, 0, true},
		{"double", 40, 20_000, 80, true},
		{"negative", -1, 10_000, 0, false},
		{"overflow", math.MaxInt64, math.MaxInt64, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tenure.RentInForce(tt.posted, tt.scale); got != tt.want || ok != tt.ok {
				t.Errorf("RentInForce(%d, %d) = %d, %v, want %d, %v", tt.posted, tt.scale, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestLeastRent_ZeroWhenOwnEntitlementCovers verifies the bound of a hire
// (spec-land.md 4.7.2): the rent of max(0, m * d - own - commons) units. A job of
// 3 executions of 2 units is 6 units; an own entitlement of 6 covers it (0), and
// an own 2 with a commons of 1 leaves 3 units, at a rent of 4, 12.
//
// Requirement: FR-LND-014
func TestLeastRent_ZeroWhenOwnEntitlementCovers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		m, d, own, commons, rent int64
		want                     int64
		ok                       bool
	}{
		{"own covers", 3, 2, 6, 0, 4, 0, true},
		{"own and commons cover", 3, 2, 4, 2, 4, 0, true},
		{"remainder let", 3, 2, 2, 1, 4, 12, true},
		{"no entitlement", 3, 2, 0, 0, 4, 24, true},
		{"negative", -1, 2, 0, 0, 4, 0, false},
		{"product overflows", math.MaxInt64, 2, 0, 0, 4, 0, false},
		{"rent overflows", 1 << 40, 1, 0, 0, 1 << 40, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tenure.LeastRent(tt.m, tt.d, tt.own, tt.commons, tt.rent); got != tt.want || ok != tt.ok {
				t.Errorf("LeastRent(%d, %d, %d, %d, %d) = %d, %v, want %d, %v", tt.m, tt.d, tt.own, tt.commons, tt.rent, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestCharge_LargestKBalanceCovers_ReservedAgainstOtherStarts_ZeroIsRentFunds
// verifies the cut for rent (spec-land.md 5.4). One draw of 2 units at a rent of
// 5 with 1 own and 1 commons execution: k executions cost 10 * max(0, k - 2).
// With 35 available and 8 wanted, the largest k is 5 (rent 30). The balance 50
// less 20 reserved by earlier starts leaves 30, the same. With 9 available no
// execution beyond the free two is paid for, so k is 2, not 0; with no free
// execution (Own 0, Commons 0) 9 available is below the 10 of one execution, so
// k is 0, the refusal rent_funds.
//
// Requirement: FR-LND-016
func TestCharge_LargestKBalanceCovers_ReservedAgainstOtherStarts_ZeroIsRentFunds(t *testing.T) {
	t.Parallel()
	free := []tenure.Line{{Rent: 5, Draw: 2, Own: 1, Commons: 1}}
	tests := []struct {
		name      string
		available int64
		other     int64
		lines     []tenure.Line
		wantK     int64
		wantRent  int64
	}{
		{"covers some", 35, 8, free, 5, 30},
		{"reserved against other starts", tenure.Available(50, 20), 8, free, 5, 30},
		{"covers all", 1000, 8, free, 8, 60},
		{"free executions are free", 9, 8, free, 2, 0},
		{"zero is rent funds", 9, 8, []tenure.Line{{Rent: 5, Draw: 2}}, 0, 0},
		{"two draws", 40, 8, []tenure.Line{{Rent: 5, Draw: 2}, {Rent: 1, Draw: 1, Own: 1}}, 3, 32}, // k=3: 5*2*3 + 1*1*(3-1) = 32; k=4 costs 40 + 3 = 43
		{"nothing wanted", 40, 0, free, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k, rent, ok := tenure.Charge(tt.available, tt.other, tt.lines)
			if !ok || k != tt.wantK || rent != tt.wantRent {
				t.Errorf("Charge(%d, %d, %+v) = %d, %d, %v, want %d, %d", tt.available, tt.other, tt.lines, k, rent, ok, tt.wantK, tt.wantRent)
			}
		})
	}
}

// TestCharge_OverflowIsUnaffordableAndRefusals verifies a total rent beyond int64
// is unaffordable (the largest k is the one before it), and that a negative number
// is refused.
//
// Requirement: FR-LND-016
func TestCharge_OverflowIsUnaffordableAndRefusals(t *testing.T) {
	t.Parallel()
	lines := []tenure.Line{{Rent: 1 << 40, Draw: 1 << 20}} // one execution costs 2^60; eight overflow int64
	if k, rent, ok := tenure.Charge(math.MaxInt64, 100, lines); !ok || k != 7 || rent != 7<<60 {
		t.Errorf("Charge(MaxInt64, 100, %+v) = %d, %d, %v, want 7, %d", lines, k, rent, ok, int64(7)<<60)
	}
	for _, bad := range []struct {
		available, other int64
		lines            []tenure.Line
	}{
		{-1, 1, nil}, {1, -1, nil}, {1, 1, []tenure.Line{{Rent: -1, Draw: 1}}},
		{1, 1, []tenure.Line{{Rent: 1, Draw: -1}}}, {1, 1, []tenure.Line{{Rent: 1, Draw: 1, Own: -1}}},
		{1, 1, []tenure.Line{{Rent: 1, Draw: 1, Commons: -1}}},
	} {
		if k, rent, ok := tenure.Charge(bad.available, bad.other, bad.lines); ok || k != 0 || rent != 0 {
			t.Errorf("Charge(%d, %d, %+v) = %d, %d, %v, want refused", bad.available, bad.other, bad.lines, k, rent, ok)
		}
	}
	// free entitlements beyond int64 in sum still cover k: the line is free, not unaffordable
	if total, ok := tenure.TotalRent([]tenure.Line{{Rent: 1, Draw: 1, Own: math.MaxInt64, Commons: 1}}, 5); !ok || total != 0 {
		t.Errorf("TotalRent(free beyond int64) = %d, %v, want 0, true", total, ok)
	}
}

// TestAvailable verifies the money a start can spend: the balance less the
// reservation, 0 when the reservation reaches the balance or the balance is not
// positive.
//
// Requirement: FR-LND-016
func TestAvailable(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ balance, reserved, want int64 }{
		{50, 20, 30}, {50, 50, 0}, {50, 70, 0}, {0, 0, 0}, {-5, 0, 0}, {50, -1, 0},
	} {
		if got := tenure.Available(tt.balance, tt.reserved); got != tt.want {
			t.Errorf("Available(%d, %d) = %d, want %d", tt.balance, tt.reserved, got, tt.want)
		}
	}
}

// TestCharge_PayeeAtMaxMoneyForfeits verifies a payee near the bound receives
// what it can and the rest is forfeited, not charged: balance 90 of a limit of
// 100 receiving 25 takes 10 and forfeits 15 (spec-land.md 5.3.3).
//
// Requirement: FR-LND-016
func TestCharge_PayeeAtMaxMoneyForfeits(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		balance, amount, limit, paid, forfeited int64
		ok                                      bool
	}{
		{90, 25, 100, 10, 15, true}, {0, 25, 100, 25, 0, true}, {100, 25, 100, 0, 25, true},
		{101, 1, 100, 0, 0, false}, {-1, 1, 100, 0, 0, false}, {1, -1, 100, 0, 0, false}, {1, 1, -1, 0, 0, false},
	} {
		paid, forfeited, ok := tenure.Credit(tt.balance, tt.amount, tt.limit)
		if paid != tt.paid || forfeited != tt.forfeited || ok != tt.ok {
			t.Errorf("Credit(%d, %d, %d) = %d, %d, %v, want %d, %d, %v", tt.balance, tt.amount, tt.limit, paid, forfeited, ok, tt.paid, tt.forfeited, tt.ok)
		}
	}
}

// TestStep_RisesFallsHolds is the table of the rent step (spec-land.md 5.5.1) for
// a configured rent of 40, a step of 250 basis points (a step of 1), a floor of
// 5 000 (20) and a ceiling of 20 000 (80), from a posted rent of 40: unserved
// units with no cut raise it, unserved units with a cut and idle units lower it,
// idle units alone lower it, nothing leaves it, priced-out units appear in no
// field and so move it neither way, a zero StepBP and a closed endowment never
// move it, and the band bounds both directions.
//
// Requirement: FR-LND-019
func TestStep_RisesFallsHolds(t *testing.T) {
	t.Parallel()
	base := tenure.StepIn{Posted: 40, Base: 40, StepBP: 250, FloorBP: 5_000, CeilBP: 20_000}
	tests := []struct {
		name string
		edit func(*tenure.StepIn)
		want int64
	}{
		{"rises when unserved and no cut", func(in *tenure.StepIn) { in.Unserved = 3 }, 41},
		{"unserved with a cut falls", func(in *tenure.StepIn) { in.Unserved, in.Cut, in.Idle = 3, 2, 2 }, 39},
		{"idle falls", func(in *tenure.StepIn) { in.Idle = 4 }, 39},
		{"unserved and idle: unserved first", func(in *tenure.StepIn) { in.Unserved, in.Idle = 3, 4 }, 41},
		{"priced out moves neither way", func(in *tenure.StepIn) {}, 40},
		{"zero step never moves", func(in *tenure.StepIn) { in.StepBP, in.Unserved = 0, 9 }, 40},
		{"closed never moves", func(in *tenure.StepIn) { in.Closed, in.Unserved = true, 9 }, 40},
		{"ceiling bounds the rise", func(in *tenure.StepIn) { in.Posted, in.Unserved = 80, 3 }, 80},
		{"floor bounds the fall", func(in *tenure.StepIn) { in.Posted, in.Idle = 20, 3 }, 20},
		{"above the ceiling comes down to it", func(in *tenure.StepIn) { in.Posted, in.Unserved = 95, 3 }, 80},
		{"large step is a fraction of the base", func(in *tenure.StepIn) { in.StepBP, in.Unserved = 1_000, 3 }, 44},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := base
			tt.edit(&in)
			if got, ok := tenure.Step(in); !ok || got != tt.want {
				t.Errorf("Step(%+v) = %d, %v, want %d", in, got, ok, tt.want)
			}
		})
	}
}

// TestStep_ZeroBaseStaysZero verifies a configured rent of 0 has a ceiling of 0
// and a posted rent that stays 0 whatever is unserved, and that the step is at
// least 1 for a small base.
//
// Requirement: FR-LND-019
func TestStep_ZeroBaseStaysZero(t *testing.T) {
	t.Parallel()
	if got, ok := tenure.Step(tenure.StepIn{StepBP: 500, FloorBP: 0, CeilBP: 20_000, Unserved: 9}); !ok || got != 0 {
		t.Errorf("Step(base 0, unserved) = %d, %v, want 0", got, ok)
	}
	// A base of 3 with a step of 250 basis points is 0 truncated, raised to 1.
	if got, ok := tenure.Step(tenure.StepIn{Posted: 3, Base: 3, StepBP: 250, FloorBP: 0, CeilBP: 20_000, Unserved: 1}); !ok || got != 4 {
		t.Errorf("Step(base 3, unserved) = %d, %v, want 4", got, ok)
	}
}

// TestStep_Refusals verifies a negative field or a product beyond int64 is
// refused and returns the posted rent.
//
// Requirement: FR-LND-019
func TestStep_Refusals(t *testing.T) {
	t.Parallel()
	for _, in := range []tenure.StepIn{
		{Posted: -1, Base: 1, StepBP: 1}, {Posted: 1, Base: -1, StepBP: 1}, {Posted: 1, Base: 1, StepBP: 1, Unserved: -1},
		{Posted: 7, Base: math.MaxInt64, StepBP: 1_000_000, CeilBP: 1},
		{Posted: math.MaxInt64, Base: math.MaxInt64, StepBP: 1, CeilBP: 10_000, Unserved: 1},
	} {
		if got, ok := tenure.Step(in); ok || got != in.Posted {
			t.Errorf("Step(%+v) = %d, %v, want refused with the posted rent", in, got, ok)
		}
	}
}

// TestCap_LargestRoleHolderAboveCap verifies the construction check of the cap
// (spec-land.md 6.1.2): the largest agent of a group holds the most, weight *
// shareBP. Weights 2, 7 and 5 at 300 basis points hold 600, 2 100 and 1 500; a
// cap of 2 000 is passed by the weight 7 only, a cap of 10 000 or more bounds
// nothing, and the pool is never asked.
//
// Requirement: FR-LND-041
func TestCap_LargestRoleHolderAboveCap(t *testing.T) {
	t.Parallel()
	weights := []int64{2, 7, 5}
	for _, tt := range []struct {
		capBP int64
		idx   int
		found bool
	}{{2_000, 1, true}, {1_000, 1, true}, {2_100, 0, false}, {10_000, 0, false}} {
		if idx, found, ok := tenure.LargestOverCap(weights, 300, tt.capBP); !ok || found != tt.found || (found && idx != tt.idx) {
			t.Errorf("LargestOverCap(%v, 300, %d) = %d, %v, %v, want %d, %v", weights, tt.capBP, idx, found, ok, tt.idx, tt.found)
		}
	}
	if _, _, ok := tenure.LargestOverCap([]int64{math.MaxInt64}, 2, 100); ok {
		t.Errorf("LargestOverCap(overflow) ok, want refused")
	}
	if share, ok := tenure.HolderShare(-1, 5); ok || share != 0 {
		t.Errorf("HolderShare(-1, 5) = %d, %v, want refused", share, ok)
	}
}

// TestCap_OverCapIsStrictAndOffAtBasisPoints verifies the cap is a strict bound
// and that a cap of 10 000 or more bounds nothing, whatever the share.
//
// Requirement: FR-LND-041
func TestCap_OverCapIsStrictAndOffAtBasisPoints(t *testing.T) {
	t.Parallel()
	if tenure.OverCap(10_001, 10_000) || !tenure.OverCap(2_001, 2_000) || tenure.OverCap(2_000, 2_000) {
		t.Errorf("OverCap: the cap is a strict bound, and off at 10 000")
	}
}
