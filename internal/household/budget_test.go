package household_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestSpendCap_Examples verifies the cap of AGT 16.2 and its boundaries.
//
// Requirement: FR-AGT-042
func TestSpendCap_Examples(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		e, uc  int64
		p      household.Params
		want   int64
		wantOK bool
	}{
		// e 4 505, H = 10 * 1 000 = 10 000: 4 505 * 10 000 / 14 505 = 3 105 (truncated),
		// cap = 5 000 * 3 105 / 10 000 = 1 552.
		{"example 16.2", 4505, 1000, params, 1552, true},
		{"no excess", 0, 1000, params, 0, true},
		{"negative excess", -5, 1000, params, 0, true},
		// No upkeep cost: 5 000 * 100 / 10 000 = 50.
		{"no upkeep cost", 100, 0, params, 50, true},
		// A share above 10 000 is cut to 10 000, so the cap is e / 2 at most of e: e * H / (e + H) with e = H.
		{"share clamped", 100, 10, household.Params{SpendMaxShare: 50_000, SpendHalfTicks: 10}, 50, true},
		// SpendHalfTicks 0 gives H 0 and a cap of 0.
		{"no half ticks", 100, 10, household.Params{SpendMaxShare: 5000}, 0, true},
		{"H overflows", 100, math.MaxInt64 / 2, params, 0, false},
		// e + H = 10 + (MaxInt64 - 9) = MaxInt64 + 1.
		{"e plus H overflows", 10, math.MaxInt64 - 9, household.Params{SpendMaxShare: 5000, SpendHalfTicks: 1}, 0, false},
		// e * share overflows? e = Max, share 10 000 -> Max * 10 000 / 10 000 = Max: fits.
		{"largest excess without upkeep", math.MaxInt64, 0, household.Params{SpendMaxShare: 10_000}, math.MaxInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.SpendCap(tt.e, tt.uc, tt.p)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("SpendCap(%d, %d, %+v) = %d, %v, want %d, %v", tt.e, tt.uc, tt.p, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestSpendCap_MonotoneAndBounded verifies over a range of excess values that
// the cap never decreases and never exceeds the excess.
//
// Requirement: FR-AGT-042
func TestSpendCap_MonotoneAndBounded(t *testing.T) {
	t.Parallel()
	for _, uc := range []int64{0, 1, 7, 1000} {
		var prev int64
		for e := int64(0); e < 5000; e += 7 {
			got, ok := household.SpendCap(e, uc, params)
			if !ok || got < prev || got > e {
				t.Fatalf("SpendCap(%d, %d) = %d, %v after %d: want non-decreasing and at most e", e, uc, got, ok, prev)
			}
			prev = got
		}
	}
}

// TestExcess verifies the money above the savings target: AGT 16.1 has
// 2 005 - 3 000 below zero, AGT 16.2 has 7 505 - 3 000 = 4 505.
//
// Requirement: FR-AGT-041
func TestExcess(t *testing.T) {
	t.Parallel()
	tests := []struct{ avail, saving, want int64 }{
		{2005, 3000, 0}, {7505, 3000, 4505}, {3000, 3000, 0}, {5, -4, 5},
		{math.MaxInt64, -1, math.MaxInt64}, {-5, 3, 0},
	}
	for _, tt := range tests {
		if got := household.Excess(tt.avail, tt.saving); got != tt.want {
			t.Errorf("Excess(%d, %d) = %d, want %d", tt.avail, tt.saving, got, tt.want)
		}
	}
}

// TestBuyShare verifies the favourability of AGT 7.5.2 and AGT 16.2.
//
// Requirement: FR-AGT-043
func TestBuyShare(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pos  int64
		p    household.Params
		want int64
	}{
		// 2 000 + 8 000 * 5 000 / 10 000 = 6 000 (AGT 16.2).
		{"neutral window", household.EmptyPos, params, 6000},
		{"price at the low", 0, params, 10_000},
		{"price at the high", 10_000, params, 2000},
		{"position cut above", 20_000, params, 2000},
		{"position cut below", -5, params, 10_000},
		// 1 000 + 9 000 * (10 000 - 2 500) / 10 000 = 1 000 + 6 750 = 7 750.
		{"quarter position", 2500, household.Params{FavourFloor: 1000}, 7750},
		{"floor cut to the basis", 3000, household.Params{FavourFloor: 20_000}, 10_000},
		{"floor of zero", 5000, household.Params{}, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.BuyShare(tt.pos, tt.p); got != tt.want {
				t.Errorf("BuyShare(%d, %+v) = %d, want %d", tt.pos, tt.p, got, tt.want)
			}
		})
	}
}

// TestAfford verifies the quantity one tier buys and the lowered limit.
//
// Requirement: FR-AGT-041, FR-AGT-044
func TestAfford(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		avail, limit, want int64
		wantLimit, wantQty int64
	}{
		// AGT 16.1 tier 1: fuel 1 at 105 with 2 500 left.
		{"fuel tier 1", 2500, 105, 1, 105, 1},
		// AGT 16.2 tier 3: staple 4 at 310 within the cap 1 447: 1 447 / 310 = 4.
		{"staple within the cap", 1447, 310, 4, 310, 4},
		// Money for 2 of 5 wanted: 250 / 105 = 2 (AGT 7.5.3 example).
		{"money binds", 250, 105, 5, 105, 2},
		// 50 cannot buy one unit at 105: one unit at 50.
		{"limit lowered to the money", 50, 105, 3, 50, 1},
		{"no money", 0, 105, 3, 105, 0},
		{"no want", 500, 105, 0, 105, 0},
		{"no want, money below one unit", 50, 105, 0, 105, 0},
		{"no valid limit", 500, 0, 3, 0, 0},
		{"exactly one unit", 105, 105, 3, 105, 1},
		{"want binds", 10_000, 105, 3, 105, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			limit, qty := household.Afford(tt.avail, tt.limit, tt.want)
			if limit != tt.wantLimit || qty != tt.wantQty {
				t.Errorf("Afford(%d, %d, %d) = %d, %d, want %d, %d", tt.avail, tt.limit, tt.want, limit, qty, tt.wantLimit, tt.wantQty)
			}
			if qty > 0 && qty > tt.avail/limit {
				t.Errorf("Afford(%d, %d, %d) spends %d * %d, more than the money", tt.avail, tt.limit, tt.want, qty, limit)
			}
		})
	}
}

// TestBudget_Example162 verifies the whole allocation of AGT 16.2 from the
// functions of this package: excess 4 505, cap 1 552, buy share 6 000, wants
// fuel 1 and staple 4, spent 105 then 1 240, 207 of the cap left.
//
// Requirement: FR-AGT-041, FR-AGT-042, FR-AGT-043
func TestBudget_Example162(t *testing.T) {
	t.Parallel()
	e := household.Excess(8000-105-390, 3000) // 7 505 left after tiers 1 and 2
	if e != 4505 {
		t.Fatalf("Excess = %d, want 4505", e)
	}
	capMoney, ok := household.SpendCap(e, 1000, params)
	if !ok || capMoney != 1552 {
		t.Fatalf("SpendCap = %d, %v, want 1552, true", capMoney, ok)
	}
	share := household.BuyShare(household.EmptyPos, params)
	if share != 6000 {
		t.Fatalf("BuyShare = %d, want 6000", share)
	}
	left := capMoney
	for _, c := range []struct{ t3, limit, wantQty int64 }{{3, 105, 1}, {8, 310, 4}} {
		want := c.t3 * share / household.Basis // floor(3 * 0.6) = 1, floor(8 * 0.6) = 4
		limit, qty := household.Afford(left, c.limit, want)
		if qty != c.wantQty || limit != c.limit {
			t.Fatalf("tier 3 of limit %d: Afford = %d, %d, want %d at the same limit", c.limit, limit, qty, c.wantQty)
		}
		left -= limit * qty
	}
	if left != 207 {
		t.Errorf("cap left = %d, want 207 (1552 - 105 - 1240)", left)
	}
}

// TestBudget_AllocateNothing verifies that the money functions allocate
// nothing.
//
// Requirement: NFR-AGT-001
func TestBudget_AllocateNothing(t *testing.T) {
	var sink int64
	allocs := testing.AllocsPerRun(100, func() {
		c, _ := household.SpendCap(4505, 1000, params)
		l, q := household.Afford(c, 310, 4)
		sink += c + l + q + household.Excess(7505, 3000) + household.BuyShare(5000, params)
	})
	if allocs != 0 {
		t.Errorf("money functions allocated %v times per run, want 0 (sink %d)", allocs, sink)
	}
}

func BenchmarkSpendCap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		household.SpendCap(4505, 1000, params)
	}
}

// TestSpendCap_OverflowIsNotOK verifies that SpendCap reports not ok, and not a
// wrapped cap, when the half-life times the upkeep cost, or the excess plus
// that product, does not fit in an int64.
//
// Guard: a validated configuration cannot reach either overflow, and the
// caller reads not ok as "spend nothing on a buffer"; a wrapped cap would let a
// household spend its reserve.
func TestSpendCap_OverflowIsNotOK(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		e    int64
		uc   int64
		p    household.Params
	}{
		{"half-life times upkeep cost past int64", 100, math.MaxInt64, household.Params{SpendMaxShare: 5000, SpendHalfTicks: 2}},
		// h = 1 * (MaxInt64 - 5) fits; e + h = MaxInt64 + 5 does not.
		{"excess plus the product past int64", 10, math.MaxInt64 - 5, household.Params{SpendMaxShare: 5000, SpendHalfTicks: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := household.SpendCap(tt.e, tt.uc, tt.p); ok {
				t.Errorf("SpendCap(%d, %d, %+v) = %d, true, want not ok", tt.e, tt.uc, tt.p, got)
			}
		})
	}
}
