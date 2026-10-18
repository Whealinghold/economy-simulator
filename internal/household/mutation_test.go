package household_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// The tests in this file pin the boundaries that a mutation run (gomu, inside
// Docker) showed the other tests of the package did not: each case sits on
// the value where a comparison or a constant would change the result.

// TestSpendCap_Boundaries verifies the cap at the edges of its three guards:
// an excess of 1, an upkeep cost of 1 or below 0, and a share of 0 or below.
//
// 1 * 10 000 / 10 000 = 1 at a full share. With an upkeep cost of 1, a half-life
// of 10 and an excess of 1 000: H = 10, x = 1 000 * 10 / 1 010 = 9, and 9 *
// 5 000 / 10 000 = 4; without the half-life the cap would be 1 000 * 5 000 /
// 10 000 = 500.
//
// Requirement: FR-AGT-042
func TestSpendCap_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		e    int64
		uc   int64
		p    household.Params
		want int64
	}{
		{"an excess of 1 at a full share buys 1", 1, 0, household.Params{SpendMaxShare: 10_000}, 1},
		{"an upkeep cost of 1 applies the half-life", 1_000, 1, household.Params{SpendMaxShare: 5_000, SpendHalfTicks: 10}, 4},
		{"a negative upkeep cost counts as none", 1_000, -5, household.Params{SpendMaxShare: 5_000, SpendHalfTicks: 10}, 500},
		{"a share of 0 caps at 0", 100_000, 0, household.Params{SpendMaxShare: 0}, 0},
		{"a negative share caps at 0", 100_000, 0, household.Params{SpendMaxShare: -5}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.SpendCap(tt.e, tt.uc, tt.p)
			if !ok || got != tt.want {
				t.Errorf("SpendCap(%d, %d, %+v) = %d, %v, want %d, true", tt.e, tt.uc, tt.p, got, ok, tt.want)
			}
		})
	}
}

// TestBuyShare_FloorOfZeroAtTheTopOfTheWindow verifies that with a favour floor
// of 0 a price at the top of the window buys nothing, and that EmptyPos is the
// middle of the window.
//
// With floor 0 and pos 10 000: 0 + 10 000 * 0 / 10 000 = 0. At EmptyPos 5 000
// and floor 0: 10 000 * 5 000 / 10 000 = 5 000.
//
// Requirement: FR-AGT-043
func TestBuyShare_FloorOfZeroAtTheTopOfTheWindow(t *testing.T) {
	t.Parallel()
	p := household.Params{FavourFloor: 0}
	if got := household.BuyShare(household.Basis, p); got != 0 {
		t.Errorf("BuyShare(%d, floor 0) = %d, want 0", household.Basis, got)
	}
	if household.EmptyPos != 5_000 {
		t.Errorf("EmptyPos = %d, want 5 000", household.EmptyPos)
	}
	if got := household.BuyShare(household.EmptyPos, p); got != 5_000 {
		t.Errorf("BuyShare(EmptyPos, floor 0) = %d, want 5 000", got)
	}
}

// TestAfford_OneUnitEdges verifies that a bid of exactly one unit, one unit of
// money or a limit of 1 is still bought, one edge at a time.
//
// Requirement: FR-AGT-040, FR-AGT-041
func TestAfford_OneUnitEdges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		avail, limit, want int64
		wantLimit, wantQty int64
	}{
		{"money for exactly one unit at a limit of 1", 1, 1, 1, 1, 1},
		{"a limit of 1 buys the whole want", 10, 1, 5, 1, 5},
		{"a want of 1 buys one unit", 10, 2, 1, 2, 1},
		{"money below the limit buys one unit at the money", 1, 5, 5, 1, 1},
		{"no money buys nothing", 0, 5, 5, 5, 0},
		{"a limit of 0 buys nothing", 10, 0, 5, 0, 0},
		{"a want of 0 buys nothing", 10, 2, 0, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, q := household.Afford(tt.avail, tt.limit, tt.want)
			if l != tt.wantLimit || q != tt.wantQty {
				t.Errorf("Afford(%d, %d, %d) = %d, %d, want %d, %d", tt.avail, tt.limit, tt.want, l, q, tt.wantLimit, tt.wantQty)
			}
		})
	}
}

// TestFed_EdgesOfTheRows verifies the row loop of Fed: a per-household quantity
// of 1 is read, a consumed of 0 feeds nobody, and the shorter slice sets the
// prefix in either direction without reading past it.
//
// Requirement: FR-AGT-013, FR-AGT-014
func TestFed_EdgesOfTheRows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		consumed []int64
		per      []int64
		weight   int64
		wantFed  int64
	}{
		{"a quantity of 1 is read", []int64{2}, []int64{1}, 5, 2},
		{"nothing consumed feeds nobody", []int64{0}, []int64{1}, 5, 0},
		{"consumed shorter than the rows", []int64{5}, []int64{1, 1}, 3, 3},
		{"rows shorter than consumed", []int64{5, 0}, []int64{1}, 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.Fed(tt.consumed, tt.per, tt.weight); got != tt.wantFed {
				t.Errorf("Fed(%v, %v, %d) = %d, want %d", tt.consumed, tt.per, tt.weight, got, tt.wantFed)
			}
		})
	}
}

// TestCost_AddReportsAnOverflowOfTheProduct verifies that a product of total
// and centre that does not fit makes the cost not ok, even when the sum so far
// would accept the (zero) value a failed product leaves behind.
//
// Guard: a cost that wrapped would set the reserve money, and with it what a
// household spends on a buffer.
func TestCost_AddReportsAnOverflowOfTheProduct(t *testing.T) {
	t.Parallel()
	var c household.Cost
	c.Add(math.MaxInt64, 2)
	if got, ok := c.Value(); ok {
		t.Errorf("Cost after Add(MaxInt64, 2) = %d, true, want not ok", got)
	}
}

// TestComputeTargets_ZeroEdges verifies the targets where a term is exactly 0
// or 1: no cover ticks, a negative cover, a use of 1 and a want of 1 each decide
// the target, and none falls back to the output target.
//
// Use = upkeep * cover ticks + job. With cover 0 and job 2 the use is 2; with
// upkeep 1, cover 1 and no job it is 1; a want of 1 alone makes the target 1.
//
// Requirement: FR-AGT-040
func TestComputeTargets_ZeroEdges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		n    household.Need
		p    household.Params
		want household.Targets
	}{
		{"no cover ticks leaves the job", household.Need{Upkeep: 5, Job: 2}, household.Params{CoverTicks: 0}, household.Targets{Reserve: 5, UseTgt: 2, Target: 2, Keep: 7}},
		{"negative cover ticks count as 0", household.Need{Upkeep: 5, Job: 2}, household.Params{CoverTicks: -3}, household.Targets{Reserve: 5, UseTgt: 2, Target: 2, Keep: 7}},
		{"a use of 1 beats the output target", household.Need{Upkeep: 1, OutTgt: 7}, household.Params{CoverTicks: 1}, household.Targets{Reserve: 1, UseTgt: 1, Target: 1, Keep: 2}},
		{"a want of 1 beats the output target", household.Need{Want: 1, OutTgt: 7}, household.Params{}, household.Targets{Reserve: 0, UseTgt: 0, Target: 1, Keep: 1}},
		{"no use and no want takes the output target", household.Need{OutTgt: 7}, household.Params{}, household.Targets{Target: 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.ComputeTargets(tt.n, tt.p)
			if !ok || got != tt.want {
				t.Errorf("ComputeTargets(%+v, %+v) = %+v, %v, want %+v, true", tt.n, tt.p, got, ok, tt.want)
			}
		})
	}
}

// TestNet_OutputOverflowIsNotOK verifies that a value whose scaled product does
// not fit in an int64 makes the net not ok.
//
// q * effective = 10^9 * 10^9 = 10^18 fits, and 10^18 * centre 10^18 / 10^4 =
// 10^32 does not.
//
// Guard: a wrapped value could make a recipe look like the best one to run.
func TestNet_OutputOverflowIsNotOK(t *testing.T) {
	t.Parallel()
	var n household.Net
	n.Output(1_000_000_000, 1_000_000_000, 1_000_000_000_000_000_000)
	if got, ok := n.Value(); ok {
		t.Errorf("Net after an overflowing Output = %d, true, want not ok", got)
	}
}

// TestBetter_MatchesExactCrossMultiplication verifies Better against the exact
// comparison netA/durA > netB/durB made with math/big, over a grid that reaches
// the 128-bit products (high words that differ, and equal high words with
// different low words), both signs, zero and the extremes of int64, and that a
// duration below 1 gives false.
//
// 2^62 * 8 = 2^65 has a high word of 2 and a low word of 0, while 2^62 * 1 has a
// high word of 0, so the grid separates a comparison of high words from one of
// low words.
//
// Requirement: FR-AGT-035
func TestBetter_MatchesExactCrossMultiplication(t *testing.T) {
	t.Parallel()
	nets := []int64{0, 1, -1, 2, -2, 5, -5, 1 << 31, -(1 << 31), 1 << 62, -(1 << 62), math.MaxInt64, math.MinInt64}
	durs := []int64{1, 2, 3, 8, 1 << 31, 1 << 62, math.MaxInt64}
	for _, na := range nets {
		for _, da := range durs {
			for _, nb := range nets {
				for _, db := range durs {
					// netA/durA > netB/durB  <=>  netA*durB > netB*durA (durations are positive).
					l := new(big.Int).Mul(big.NewInt(na), big.NewInt(db))
					r := new(big.Int).Mul(big.NewInt(nb), big.NewInt(da))
					want := l.Cmp(r) > 0
					if got := household.Better(na, da, nb, db); got != want {
						t.Fatalf("Better(%d, %d, %d, %d) = %v, want %v", na, da, nb, db, got, want)
					}
				}
			}
		}
	}
	// Negative nets with a zero duration on either side: the products would be 0
	// and |net|, and the smaller magnitude would win, so only the duration guard
	// returns false.
	for _, c := range [][4]int64{{5, 0, 1, 1}, {5, 1, 1, 0}, {5, -1, 1, 1}, {5, 1, 1, -1}, {-5, 1, -1, 0}, {-5, 0, -1, 1}} {
		if household.Better(c[0], c[1], c[2], c[3]) {
			t.Errorf("Better(%d, %d, %d, %d) = true, want false for a duration below 1", c[0], c[1], c[2], c[3])
		}
	}
}

// TestOutputTarget_Edges verifies the target at its zero and overflow edges:
// a weight, quantity or yield of 0 gives 0, a negative one counts as 0, and a
// product that does not fit at each of the three steps reports not ok with 0.
//
// Requirement: FR-AGT-036
func TestOutputTarget_Edges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                  string
		weight, q, effective, cover, duration int64
		want                                  int64
		wantOK                                bool
	}{
		{"weight 0", 0, 5, 10_000, 3, 1, 0, true},
		{"negative weight counts as 0", -4, 5, 10_000, 3, 1, 0, true},
		{"quantity 0", 2, 0, 10_000, 3, 1, 0, true},
		{"negative quantity counts as 0", 2, -6, 10_000, 3, 1, 0, true},
		{"effective 0", 2, 5, 0, 3, 1, 0, true},
		{"negative effective counts as 0", 2, 5, -7, 3, 1, 0, true},
		{"weight times quantity overflows", 1 << 40, 1 << 40, 10_000, 3, 1, 0, false},
		{"times effective overflows", 1, 1 << 40, 1 << 40, 3, 1, 0, false},
		{"denominator overflows", 1, 1, 10_000, 3, math.MaxInt64, 0, false},
		{"duration below 1", 1, 1, 10_000, 3, 0, 0, false},
		{"a plain case", 2, 5, 10_000, 3, 1, 30, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.OutputTarget(tt.weight, tt.q, tt.effective, tt.cover, tt.duration)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("OutputTarget(%d, %d, %d, %d, %d) = %d, %v, want %d, %v", tt.weight, tt.q, tt.effective, tt.cover, tt.duration, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestHalted_TargetOfOne verifies that an output target of exactly 1 is
// compared like any other: 1 unit of stock against a ratio of 10 000 halts.
//
// 1 * 10 000 >= 10 000 * 1.
//
// Requirement: FR-AGT-036
func TestHalted_TargetOfOne(t *testing.T) {
	t.Parallel()
	if !household.Halted(1, 1, 10_000) {
		t.Errorf("Halted(1, 1, 10 000) = false, want true")
	}
	if household.Halted(1, 0, 10_000) {
		t.Errorf("Halted(1, 0, 10 000) = true, want false for a target of 0")
	}
}

// TestRequest_OneExecution verifies that a request of one execution is
// returned whichever of the two limits is 1.
//
// Requirement: FR-AGT-037
func TestRequest_OneExecution(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ executions, kcap, want int64 }{{1, 5, 1}, {5, 1, 1}, {1, 1, 1}} {
		if got, ok := household.Request(tt.executions, tt.kcap); !ok || got != tt.want {
			t.Errorf("Request(%d, %d) = %d, %v, want %d, true", tt.executions, tt.kcap, got, ok, tt.want)
		}
	}
}
