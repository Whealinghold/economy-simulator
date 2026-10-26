package household_test

import (
	"math"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestPriceRatio verifies the ratio of an item's price to its reference, floor,
// and the share-weighted mean of the item ratios, floor.
//
// Requirement: FR-DEM-007, FR-DEM-009
func TestPriceRatio(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		centre, ref int64
		want        int64
		ok          bool
	}{
		// 165 * 10 000 / 150 = 11 000 exactly.
		{"above the reference", 165, 150, 11_000, true},
		// 380 * 10 000 / 400 = 9 500.
		{"below the reference", 380, 400, 9_500, true},
		// 1 * 10 000 / 3 = 3 333.33, rounded down.
		{"rounds down", 1, 3, 3_333, true},
		{"negative centre counts as 0", -5, 100, 0, true},
		{"zero reference", 100, 0, 0, false},
		{"negative reference", 100, -1, 0, false},
		{"quotient overflows", 1 << 62, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.PriceRatio(tt.centre, tt.ref)
			if got != tt.want || ok != tt.ok {
				t.Errorf("PriceRatio(%d, %d) = %d, %v, want %d, %v", tt.centre, tt.ref, got, ok, tt.want, tt.ok)
			}
		})
	}

	// Ratios 9 500 and 11 000 with shares 5 728 and 4 272:
	// (5 728 * 9 500 + 4 272 * 11 000) / 10 000 = 101 408 000 / 10 000 = 10 140.8,
	// rounded down to 10 140.
	items := []household.WantItem{{Centre: 380, Ref: 400, Weight: 10_000}, {Centre: 550, Ref: 500, Weight: 10_000}}
	mean := []struct {
		name   string
		items  []household.WantItem
		shares []int64
		want   int64
		ok     bool
	}{
		{"weighted mean rounds down", items, []int64{5_728, 4_272}, 10_140, true},
		{"one item is its own ratio", items[:1], []int64{10_000}, 9_500, true},
		{"reference below 1", []household.WantItem{{Centre: 5, Ref: 0}}, []int64{10_000}, 0, false},
		// Only the shorter prefix is read: one share for two items, two shares for one item.
		{"fewer shares than items", items, []int64{10_000}, 9_500, true},
		{"fewer items than shares", items[:1], []int64{10_000, 10_000}, 9_500, true},
		// A negative share counts as 0: (0 * 9 500 + 10 000 * 11 000) / 10 000 = 11 000.
		{"negative share counts as 0", items, []int64{-5, 10_000}, 11_000, true},
		{"ratio overflows", []household.WantItem{{Centre: 1 << 50, Ref: 1}}, []int64{10_000}, 0, false},
		// 2^49 * 10 000 = 5.6e18 fits, times the share 10 000 does not.
		{"product overflows", []household.WantItem{{Centre: 1 << 49, Ref: 1}}, []int64{10_000}, 0, false},
		// 9e10 * 10 000 = 9e14 per item, times 10 000 = 9e18 each, 1.8e19 together.
		{"sum overflows", []household.WantItem{{Centre: 9e10, Ref: 1}, {Centre: 9e10, Ref: 1}}, []int64{10_000, 10_000}, 0, false},
	}
	for _, tt := range mean {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.WantRatio(tt.items, tt.shares)
			if got != tt.want || ok != tt.ok {
				t.Errorf("WantRatio(%v, %v) = %d, %v, want %d, %v", tt.items, tt.shares, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestShares_SumTo10000AndSigmaZero verifies that the substitution shares add to
// 10 000, follow the weights alone at sigma 0, move toward the cheaper item at
// a higher sigma, and are the worked example 15.1 at sigma 2.
//
// Requirement: FR-DEM-008
func TestShares_SumTo10000AndSigmaZero(t *testing.T) {
	t.Parallel()
	a := household.WantItem{Centre: 380, Ref: 400, Weight: 10_000}
	b := household.WantItem{Centre: 550, Ref: 500, Weight: 10_000}
	tests := []struct {
		name  string
		items []household.WantItem
		sigma int64
		want  []int64
		ok    bool
	}{
		// x = 400 * 10 000 / 380 = 10 526 and 500 * 10 000 / 550 = 9 090; two
		// steps: 10 526 * 10 526 / 10 000 = 11 079 and 9 090 * 9 090 / 10 000 =
		// 8 262; sum 19 341; 11 079 * 10 000 / 19 341 = 5 728.2 and 8 262 * 10 000
		// / 19 341 = 4 271.7, so the unit left goes to the larger remainder, .7.
		{"worked example 15.1", []household.WantItem{a, b}, 2, []int64{5_728, 4_272}, true},
		// Sigma 0: t = weight, 3 000 and 1 000, shares 7 500 and 2 500 at any price.
		{"sigma 0 follows the weights", []household.WantItem{{Centre: 1, Ref: 900, Weight: 3_000}, {Centre: 900, Ref: 1, Weight: 1_000}}, 0, []int64{7_500, 2_500}, true},
		{"one item takes all", []household.WantItem{a}, 3, []int64{10_000}, true},
		// Equal terms 1, 1, 1: 3 333 each, 1 left, equal remainders, lower index.
		{"tie goes to the lower index", []household.WantItem{{Centre: 5, Ref: 5, Weight: 1}, {Centre: 5, Ref: 5, Weight: 1}, {Centre: 5, Ref: 5, Weight: 1}}, 2, []int64{3_334, 3_333, 3_333}, true},
		// Sigma 4 at a 2:1 price gap: x = 20 000 and 10 000; pow 160 000 000 and 10 000
		// (v = v * x / 10 000 four times from 10 000: 20 000, 40 000, 80 000, 160 000).
		// Terms 16 and 1 (160 000 * 10 000 / 10 000 and 10 000 * 10 000 / 10 000
		// scaled by the weight 10 000): shares 160 000 / 170 000 and 10 000 / 170 000.
		{"high sigma moves to the cheap item", []household.WantItem{{Centre: 50, Ref: 100, Weight: 10_000}, {Centre: 100, Ref: 100, Weight: 10_000}}, 4, []int64{9_412, 588}, true},
		// Three terms 3 000, 3 000, 3 000 and 1 000 at sigma 0 are 3/10 each and 1/10: floors
		// 3 000, 3 000, 3 000, 1 000 add to 10 000 with nothing left.
		{"four items", []household.WantItem{{Centre: 1, Ref: 1, Weight: 3_000}, {Centre: 1, Ref: 1, Weight: 3_000}, {Centre: 1, Ref: 1, Weight: 3_000}, {Centre: 1, Ref: 1, Weight: 1_000}}, 0, []int64{3_000, 3_000, 3_000, 1_000}, true},
		// Terms 1, 1, 1, 1, 1, 1, 1: 1 428 each (1.428 57), 4 left over to the first four
		// remainders, which are equal (10 000 mod 7 = 4): the lower indexes.
		{"seven items, four left over", []household.WantItem{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}, {1, 1, 1}, {1, 1, 1}, {1, 1, 1}, {1, 1, 1}}, 0, []int64{1_429, 1_429, 1_429, 1_429, 1_428, 1_428, 1_428}, true},
		// A single term of 1: the whole 10 000 goes to it.
		{"sum of terms is 1", []household.WantItem{{Centre: 1, Ref: 1, Weight: 1}}, 0, []int64{10_000}, true},
		// A negative weight counts as 0, a negative reference gives x = 0 and so a term of 0 from sigma 1.
		{"negative weight counts as 0", []household.WantItem{{Centre: 1, Ref: 1, Weight: 5_000}, {Centre: 1, Ref: 1, Weight: -7_000}}, 0, []int64{10_000, 0}, true},
		{"negative reference counts as 0", []household.WantItem{{Centre: 1, Ref: 1, Weight: 5_000}, {Centre: 1, Ref: -9, Weight: 5_000}}, 1, []int64{10_000, 0}, true},
		{"second item has a centre below 1", []household.WantItem{{Centre: 1, Ref: 1, Weight: 5_000}, {Centre: 0, Ref: 1, Weight: 5_000}}, 2, nil, false},
		{"second item power overflows", []household.WantItem{{Centre: 1, Ref: 1, Weight: 5_000}, {Centre: 1, Ref: 1 << 40, Weight: 5_000}}, 4, nil, false},
		{"sum of terms overflows", []household.WantItem{{Centre: 1, Ref: 1, Weight: math.MaxInt64}, {Centre: 1, Ref: 1, Weight: math.MaxInt64}}, 0, nil, false},
		{"no items", nil, 2, nil, false},
		{"all weights 0", []household.WantItem{{Centre: 5, Ref: 5}, {Centre: 5, Ref: 5}}, 2, nil, false},
		{"centre below 1", []household.WantItem{{Centre: 0, Ref: 5, Weight: 10_000}}, 2, nil, false},
		{"power overflows", []household.WantItem{{Centre: 1, Ref: 1 << 40, Weight: 10_000}}, 4, nil, false},
		// ref * 10 000 / centre = 2^62 * 10 000 does not fit in int64 before any power is taken.
		{"price ratio overflows", []household.WantItem{{Centre: 1, Ref: 1 << 62, Weight: 10_000}}, 1, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// A dst that already holds values is overwritten, not appended to.
			got, ok := household.Shares([]int64{7, 7, 7}, tt.items, tt.sigma)
			if ok != tt.ok || !slices.Equal(got, tt.want) {
				t.Errorf("Shares(%v, %d) = %v, %v, want %v, %v", tt.items, tt.sigma, got, ok, tt.want, tt.ok)
			}
		})
	}

	t.Run("always add to 10 000", func(t *testing.T) {
		t.Parallel()
		for sigma := int64(0); sigma <= 4; sigma++ {
			for _, c := range []int64{1, 37, 99, 100, 250, 1_000} {
				items := []household.WantItem{{Centre: c, Ref: 100, Weight: 3_333}, {Centre: 100, Ref: 100, Weight: 10_000}, {Centre: 1_000 / c, Ref: 90, Weight: 7}}
				got, ok := household.Shares(nil, items, sigma)
				var sum int64
				for _, s := range got {
					sum += s
				}
				if !ok || sum != 10_000 {
					t.Errorf("Shares(%v, %d) = %v, %v: sum %d, want 10 000", items, sigma, got, ok, sum)
				}
			}
		}
	})
}

// TestSchedule_Table verifies the schedule at the reference, below it up to the
// ceiling, above it down to the choke price, and the elasticity table of 15.2.
//
// Requirement: FR-DEM-010, FR-DEM-011
func TestSchedule_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		ratio, elast, ceiling int64
		want                  int64
	}{
		{"reference is 10 000", 10_000, 15_000, 15_000, 10_000},
		// 10 000 + 15 000 * 2 000 / 10 000 = 13 000.
		{"cheap raises", 8_000, 15_000, 15_000, 13_000},
		// 10 000 + 15 000 * 4 000 / 10 000 = 16 000, cut to the ceiling.
		{"ceiling cuts", 6_000, 15_000, 15_000, 15_000},
		// 10 000 + 15 000 * (-1 000) / 10 000 = 8 500 (worked example 15.1).
		{"dear lowers", 11_000, 15_000, 15_000, 8_500},
		// Elasticity 20 000: choke price 10 000 + 10^8 / 20 000 = 15 000. At 14 999
		// the slope term is trunc(20 000 * (-4 999) / 10 000) = trunc(-9 998) so g = 2.
		{"just below the choke", 14_999, 20_000, 15_000, 2},
		{"at the choke price", 15_000, 20_000, 15_000, 0},
		// Elasticity 15 000: choke 16 666.67. At 16 666 the term is trunc(-9 999)
		// so g = 1; at 16 667 it is trunc(-10 000.5) = -10 000 so g = 0.
		{"fractional choke, below", 16_666, 15_000, 15_000, 1},
		{"fractional choke, above", 16_667, 15_000, 15_000, 0},
		// The worked choke of 15.1: want ratio 16 859, 10 000 + 20 000 * (-6 859) / 10 000 < 0.
		{"worked example choke", 16_859, 20_000, 15_000, 0},
		{"far above the choke", 1_000_000, 15_000, 15_000, 0},
		{"elasticity 0 is flat", 3_000, 0, 15_000, 10_000},
		{"ceiling 10 000 never rises", 5_000, 15_000, 10_000, 10_000},
		{"negative ceiling counts as 0", 5_000, 15_000, -1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.Schedule(tt.ratio, tt.elast, tt.ceiling)
			if !ok || got != tt.want {
				t.Errorf("Schedule(%d, %d, %d) = %d, %v, want %d, true", tt.ratio, tt.elast, tt.ceiling, got, ok, tt.want)
			}
		})
	}

	// The point elasticity of 15.2, Elasticity * r / g in basis points, for
	// Elasticity 15 000 and Ceiling 15 000 (floor).
	t.Run("elasticity table of 15.2", func(t *testing.T) {
		t.Parallel()
		table := []struct{ ratio, g, e int64 }{
			// g = 10 000 + 15 000 * (10 000 - r) / 10 000; e = 15 000 * r / g.
			{8_000, 13_000, 9_230},  // 15 000 * 8 000 / 13 000 = 9 230.8
			{8_333, 12_500, 9_999},  // g = 10 000 + trunc(2 500.5) = 12 500; 9 999.6
			{9_000, 11_500, 11_739}, // 11 739.1
			{10_000, 10_000, 15_000},
			{11_000, 8_500, 19_411}, // 19 411.8
			{12_000, 7_000, 25_714}, // 25 714.3
			{14_000, 4_000, 52_500},
			{16_000, 1_000, 240_000},
		}
		for _, row := range table {
			g, ok := household.Schedule(row.ratio, 15_000, 15_000)
			if !ok || g != row.g || 15_000*row.ratio/g != row.e {
				t.Errorf("Schedule(%d, 15000, 15000) = %d, %v, elasticity %d, want g %d, elasticity %d",
					row.ratio, g, ok, 15_000*row.ratio/max(g, 1), row.g, row.e)
			}
		}
	})

	t.Run("never rises with the ratio", func(t *testing.T) {
		t.Parallel()
		for _, el := range []int64{0, 5_000, 15_000, 100_000} {
			prev := int64(math.MaxInt64)
			for r := int64(0); r <= 60_000; r += 37 {
				g, ok := household.Schedule(r, el, 15_000)
				if !ok || g > prev || g < 0 || g > 15_000 {
					t.Fatalf("Schedule(%d, %d, 15000) = %d, %v after %d: want non-increasing in [0, 15000]", r, el, g, ok, prev)
				}
				prev = g
			}
		}
	})

	t.Run("overflow", func(t *testing.T) {
		t.Parallel()
		if g, ok := household.Schedule(math.MinInt64, 15_000, 15_000); ok || g != 0 {
			t.Errorf("Schedule(MinInt64, ...) = %d, true, want ok false", g)
		}
		if g, ok := household.Schedule(-1_000_000, math.MaxInt64, 15_000); ok || g != 0 {
			t.Errorf("Schedule(-1000000, MaxInt64, ...) = %d, %v, want 0, false", g, ok)
		}
		if g, ok := household.Schedule(0, math.MaxInt64, 15_000); ok || g != 0 {
			t.Errorf("Schedule(0, MaxInt64, ...) = %d, true, want ok false", g)
		}
		if g, ok := household.Schedule(-1, math.MaxInt64/10_000, math.MaxInt64); ok && g < 0 {
			t.Errorf("Schedule near the top = %d, %v", g, ok)
		}
	})
}

// need is the independent form of the rate of 4.4 for quantity 1, taste and
// region 10 000 and period 4: w * 1 * 10^4 * 10^4 * g / (4 * 10^8) = w * g / 4.
func need(w, g int64) int64 { return w * g / 4 }

// TestUnits_CarryExactOverLongRun verifies that over 400 ticks of changing
// schedules the units wanted are exactly the floor of the carried sum of the
// rates, that the carry stays below 10 000, that the error against the rate is
// under one unit, and that a cohort of weight w wants w times the units of one
// household over a whole number of periods.
//
// Requirement: FR-DEM-012, FR-DEM-014, FR-DEM-015, FR-DEM-027
func TestUnits_CarryExactOverLongRun(t *testing.T) {
	t.Parallel()
	scheds := []int64{10_000, 8_500, 12_000, 0, 15_000, 9_720}
	for _, w := range []int64{1, 7, 1000} {
		t.Run("weight "+strconv.FormatInt(w, 10), func(t *testing.T) {
			t.Parallel()
			carry, units, sum := int64(4_000), int64(0), int64(0)
			for tick := 0; tick < 400; tick++ {
				g := scheds[tick%len(scheds)]
				u, c, ok := household.Units(carry, 1, w, 10_000, 10_000, g, 4)
				if !ok || c < 0 || c >= 10_000 {
					t.Fatalf("tick %d: Units(%d, 1, %d, 10000, 10000, %d, 4) = %d, %d, %v", tick, carry, w, g, u, c, ok)
				}
				carry, units, sum = c, units+u, sum+need(w, g)
			}
			// units * 10 000 + carry = 4 000 + the sum of the rates: nothing is lost.
			if units*10_000+carry != 4_000+sum {
				t.Errorf("weight %d: units %d carry %d, want units*10000+carry = %d", w, units, carry, 4_000+sum)
			}
			if d := 4_000 + sum - units*10_000; d < 0 || d >= 10_000 {
				t.Errorf("weight %d: error against the rate is %d ten-thousandths, want in [0, 10000)", w, d)
			}
		})
	}

	t.Run("cohort wants w times the units", func(t *testing.T) {
		t.Parallel()
		// Schedule 10 000, period 4: 2 500 per household per tick, so 100 ticks
		// are 25 units each and 400 ticks 100 units, from a carry of 0.
		run := func(w int64) (units int64) {
			carry := int64(0)
			for range 400 {
				u, c, _ := household.Units(carry, 1, w, 10_000, 10_000, 10_000, 4)
				carry, units = c, units+u
			}
			return units
		}
		one := run(1)
		for _, w := range []int64{7, 1000} {
			if got := run(w); got != w*one || one != 100 {
				t.Errorf("units over 400 ticks: weight %d wants %d, weight 1 wants %d, want %d and 100", w, got, one, w*100)
			}
		}
	})

	tests := []struct {
		name                                                   string
		carry, q, w, taste, region, sched, period, units, left int64
		ok                                                     bool
	}{
		// Worked example 15.1: need 1 * 1 * 10^4 * 10^4 * 8 500 / 10^8 = 8 500; 4 000 + 8 500 = 12 500.
		{"comfort of 15.1", 4_000, 1, 1, 10_000, 10_000, 8_500, 1, 1, 2_500, true},
		// need 1 * 10^8 * 9 720 / (5 * 10^8) = 1 944; 9 000 + 1 944 = 10 944.
		{"treat of 15.1", 9_000, 1, 1, 10_000, 10_000, 9_720, 5, 1, 944, true},
		{"choked want keeps the carry", 6_543, 1, 3, 10_000, 10_000, 0, 1, 0, 6_543, true},
		// Taste 5 000 and region 15 000 scale: 2 * 3 * 5 000 * 15 000 * 10 000 / 10^8 = 45 000: 4 units, carry 5 000.
		{"scales", 0, 2, 3, 5_000, 15_000, 10_000, 1, 4, 5_000, true},
		{"negative carry counts as 0", -5, 1, 1, 10_000, 10_000, 5_000, 1, 0, 5_000, true},
		{"period 0", 0, 1, 1, 10_000, 10_000, 10_000, 0, 0, 0, false},
		{"negative period", 0, 1, 1, 10_000, 10_000, 10_000, -1, 0, 0, false},
		// With a negative factor, the others large: a mutant that counts it as 1 would want units.
		{"negative quantity counts as 0", 0, -3, 1_000_000, 10_000, 10_000, 10_000, 1, 0, 0, true},
		{"negative weight counts as 0", 0, 1_000, -3, 10_000, 10_000, 10_000, 1, 0, 0, true},
		{"negative taste counts as 0", 0, 1_000, 1_000, -3, 10_000, 10_000, 1, 0, 0, true},
		{"negative region scale counts as 0", 0, 1_000, 1_000, 10_000, -3, 10_000, 1, 0, 0, true},
		{"negative schedule counts as 0", 0, 1_000, 1_000, 10_000, 10_000, -3, 1, 0, 0, true},
		{"first product overflows", 0, 1 << 40, 1 << 30, 10_000, 10_000, 10_000, 1, 0, 0, false},
		{"taste product overflows", 0, 1 << 32, 1 << 20, 1 << 20, 10_000, 10_000, 1, 0, 0, false},
		{"second product overflows", 0, 1, 1, 10_000, 1 << 40, 1 << 40, 1, 0, 0, false},
		// n1 = 2^62 and n2 = 2^31 * 2^31 = 2^62 fit; 2^124 / 10^8 does not.
		{"quotient overflows", 0, 1 << 62, 1, 1, 1 << 31, 1 << 31, 1, 0, 0, false},
		{"carry sum overflows", math.MaxInt64, 1, 1, 10_000, 10_000, 10_000, 1, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, c, ok := household.Units(tt.carry, tt.q, tt.w, tt.taste, tt.region, tt.sched, tt.period)
			if ok != tt.ok || u != tt.units || c != tt.left {
				t.Errorf("Units(%d, %d, %d, %d, %d, %d, %d) = %d, %d, %v, want %d, %d, %v",
					tt.carry, tt.q, tt.w, tt.taste, tt.region, tt.sched, tt.period, u, c, ok, tt.units, tt.left, tt.ok)
			}
		})
	}
}

// TestFund_LadderOrder verifies the allocation of 4.5: the budget is the cap less
// tier 3, floored at 0 and at most the money left; wants are funded in order so
// a higher rank gets only what the lower ones leave; a want splits among its
// items by largest remainder with ties to the lower index; and the money spent
// never exceeds the budget.
//
// Requirement: FR-DEM-016, FR-DEM-017, FR-DEM-018, FR-DEM-020
func TestFund_LadderOrder(t *testing.T) {
	t.Parallel()
	ladder := []household.Rung{
		{Units: 3, Shares: []int64{10_000}, Limits: []int64{100}},
		{Units: 2, Shares: []int64{6_000, 4_000}, Limits: []int64{50, 30}},
	}
	unit := []household.Rung{{Units: 4, Shares: []int64{10_000}, Limits: []int64{1}}}
	four := []household.Rung{{Units: 9, Shares: []int64{3_000, 3_000, 3_000, 1_000}, Limits: []int64{1, 1, 1, 1}}}
	tests := []struct {
		name               string
		cap, spent3, avail int64
		ladder             []household.Rung
		wantOut            []int64
		wantLeft           int64
		ok                 bool
	}{
		// B = 1 000 - 600 = 400. Rung 1: min(3, 400 / 100) = 3, B 100. Rung 2: 2 units
		// split 1.2 and 0.8, floors 1 and 0, the unit left to the larger remainder
		// (8 000 against 2 000): 1 and 1. Item 1: min(1, 100 / 50) = 1, B 50; item 2:
		// min(1, 50 / 30) = 1, B 20.
		{"everything funded", 1_000, 600, 10_000, ladder, []int64{3, 1, 1}, 20, true},
		// B = 250. Rung 1: min(3, 2) = 2, B 50. Rung 2: item 1 buys 1 (B 0), item 2 nothing.
		{"higher rank gets what is left", 1_000, 750, 10_000, ladder, []int64{2, 1, 0}, 0, true},
		// B = 99: rung 1 buys 0 (99 / 100), rung 2 item 1 buys 1 (B 49), item 2 min(1, 1) = 1 (B 19).
		{"a dear want is skipped, a cheaper one above it is bought", 1_000, 901, 10_000, ladder, []int64{0, 1, 1}, 19, true},
		{"tier 3 spent the cap", 1_000, 1_000, 10_000, ladder, []int64{0, 0, 0}, 0, true},
		{"tier 3 above the cap floors at 0", 1_000, 1_500, 10_000, ladder, []int64{0, 0, 0}, 0, true},
		// The money left after tiers 1 to 3 is 150, below the cap 1 000: B = 150.
		// Rung 1: min(3, 1) = 1, B 50; item 1 of rung 2: min(1, 1) = 1, B 0.
		{"money limits the budget", 1_000, 0, 150, ladder, []int64{1, 1, 0}, 0, true},
		{"no money", 1_000, 0, -3, ladder, []int64{0, 0, 0}, 0, true},
		// Limit 1 so that a budget of 1 would buy: a negative cap less tier 3, and
		// negative money, count as a budget of 0.
		{"negative cap", -5, 0, 10_000, unit, []int64{0}, 0, true},
		{"negative tier 3 spending counts as 0", 5, -9, 10_000, unit, []int64{4}, 1, true},
		{"negative money with a cheap limit", 1_000, 0, -9, unit, []int64{0}, 0, true},
		// The budget 4 buys exactly 4 units at the limit 1 (limit 1 is a valid price).
		{"limit 1 buys", 4, 0, 10_000, unit, []int64{4}, 0, true},
		// 9 units split 3 ways: 2.7, 2.7, 2.7 and 0.9 -> floors 2, 2, 2, 0, three left with
		// remainders 7 000, 7 000, 7 000, 9 000: the 9 000 first, then the ties by index.
		{"three left over, remainders descending then index", 1_000, 0, 10_000, four, []int64{3, 3, 2, 1}, 991, true},
		// 3 units over 6 000, 2 000, 2 000 -> floors 1, 0, 0, two left with remainders 8 000, 6 000, 6 000:
		// item 0 first, then item 1 (not item 0 again, not item 2).
		{"two left over, first item biggest", 1_000, 0, 10_000, []household.Rung{{Units: 3, Shares: []int64{6_000, 2_000, 2_000}, Limits: []int64{1, 1, 1}}}, []int64{2, 1, 0}, 997, true},
		// 7 units over 1 000, 2 500, 3 200, 3 300 -> floors 0, 1, 2, 2, two left with remainders
		// 7 000, 7 500, 2 400, 3 100: item 1 then item 0.
		{"two left over, order by remainder", 1_000, 0, 10_000, []household.Rung{{Units: 7, Shares: []int64{1_000, 2_500, 3_200, 3_300}, Limits: []int64{1, 1, 1, 1}}}, []int64{1, 2, 2, 2}, 993, true},
		// Rungs of 1, 2 and 1 items: the third rung's units go to its own slot.
		{"three rungs", 1_000, 0, 10_000, []household.Rung{{Units: 1, Shares: []int64{10_000}, Limits: []int64{1}}, {Units: 2, Shares: []int64{10_000, 0}, Limits: []int64{1, 1}}, {Units: 3, Shares: []int64{10_000}, Limits: []int64{1}}}, []int64{1, 2, 0, 3}, 994, true},
		// Equal remainders (5 000 each of 1 unit): the lower index takes the unit.
		{"tie goes to the lower index", 1_000, 0, 10_000, []household.Rung{{Units: 1, Shares: []int64{5_000, 5_000}, Limits: []int64{10, 10}}}, []int64{1, 0}, 990, true},
		{"a choked want has 0 units", 1_000, 0, 10_000, []household.Rung{{Units: 0, Shares: []int64{10_000}, Limits: []int64{10}}, {Units: 1, Shares: []int64{10_000}, Limits: []int64{10}}}, []int64{0, 1}, 990, true},
		{"limit below 1 buys nothing", 1_000, 0, 10_000, []household.Rung{{Units: 4, Shares: []int64{10_000}, Limits: []int64{0}}}, []int64{0}, 1_000, true},
		{"empty ladder", 1_000, 0, 10_000, nil, []int64{}, 1_000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := make([]int64, len(tt.wantOut))
			left, ok := household.Fund(tt.cap, tt.spent3, tt.avail, tt.ladder, out)
			if !ok || left != tt.wantLeft || !slices.Equal(out, tt.wantOut) {
				t.Errorf("Fund(%d, %d, %d, %v) = %v, left %d, %v, want %v, left %d, true",
					tt.cap, tt.spent3, tt.avail, tt.ladder, out, left, ok, tt.wantOut, tt.wantLeft)
			}
			// The money spent is the budget less what is left, and never negative.
			budget := min(max(0, max(tt.cap, 0)-max(tt.spent3, 0)), max(0, tt.avail))
			var spent, i int64
			for _, r := range tt.ladder {
				for j := range r.Limits {
					spent += out[i] * r.Limits[j]
					i++
				}
			}
			if left < 0 || spent != budget-left {
				t.Errorf("Fund(%d, %d, %d, ...) spent %d with budget %d, left %d: spent must be budget less left", tt.cap, tt.spent3, tt.avail, spent, budget, left)
			}
		})
	}

	bad := []struct {
		name   string
		ladder []household.Rung
		out    int
	}{
		{"out too short", ladder, 2},
		{"more limits than shares", []household.Rung{{Units: 1, Shares: []int64{10_000}, Limits: []int64{1, 2}}}, 2},
		// Shares 6 000 + 6 000 add to more than 10 000: invalid even with one unit, where
		// both floors are 0 and apportion alone would not notice.
		{"shares above 10 000", []household.Rung{{Units: 1, Shares: []int64{6_000, 6_000}, Limits: []int64{1, 1}}}, 2},
		{"more shares than limits", []household.Rung{{Units: 1, Shares: []int64{5_000, 5_000}, Limits: []int64{1}}}, 2},
		{"out too short for the second rung", []household.Rung{{Units: 1, Shares: []int64{10_000}, Limits: []int64{1}}, {Units: 1, Shares: []int64{10_000}, Limits: []int64{1}}}, 1},
		// 10 units with shares 6 000 and 6 000: floors 6 and 6 give 12 above 10.
		{"shares above 10 000", []household.Rung{{Units: 10, Shares: []int64{6_000, 6_000}, Limits: []int64{1, 1}}}, 2},
		{"units overflow", []household.Rung{{Units: math.MaxInt64, Shares: []int64{math.MaxInt64}, Limits: []int64{1}}}, 1},
		// Shares add to 2 000, under 10 000 and so tolerated by the share check, but 3 units
		// give floor(3 * 2 000 / 10 000) = 0 and the one item takes a single unit left over:
		// 2 units have no item to take them, so the rung cannot be funded.
		{"shares under-sum leave units unplaced", []household.Rung{{Units: 3, Shares: []int64{2_000}, Limits: []int64{1}}}, 1},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if left, ok := household.Fund(1_000, 0, 1_000, tt.ladder, make([]int64, tt.out)); ok || left != 0 {
				t.Errorf("Fund(..., %v) = %d, %v, want 0, false", tt.ladder, left, ok)
			}
		})
	}
}

// TestWorkedExample_15_1 verifies every number of spec-demand.md 15.1: the cap,
// the comfort want (ratio, schedule, rate, unit, carry, budget), the treat want
// (ratios, inverse ratios, powers, shares, want ratio, schedule, rate, unit,
// carry, the item it goes to, budget) and the choke.
//
// Requirement: FR-DEM-007, FR-DEM-008, FR-DEM-009, FR-DEM-010, FR-DEM-011, FR-DEM-012, FR-DEM-016, FR-DEM-017, FR-DEM-018
func TestWorkedExample_15_1(t *testing.T) {
	t.Parallel()
	// Money 3 200, upkeep cost 300, reserve 900: e = 2 300, H = 10 * 300 = 3 000,
	// e * H / (e + H) = 6 900 000 / 5 300 = 1 301, cap = 5 000 * 1 301 / 10 000 = 650.
	spendCap, ok := household.SpendCap(2_300, 300, params)
	if !ok || spendCap != 650 {
		t.Fatalf("SpendCap(2300, 300) = %d, %v, want 650, true", spendCap, ok)
	}

	// Comfort: centre 165, ref 150: r = 11 000; g = 10 000 + 15 000 * (-1 000) / 10 000 = 8 500;
	// need 8 500; carry 4 000 + 8 500 = 12 500: one unit, carry 2 500.
	comfort := household.WantItem{Centre: 165, Ref: 150, Weight: 10_000}
	r, _ := household.PriceRatio(comfort.Centre, comfort.Ref)
	g, _ := household.Schedule(r, 15_000, 15_000)
	cu, cc, _ := household.Units(4_000, 1, 1, 10_000, 10_000, g, 1)
	if r != 11_000 || g != 8_500 || cu != 1 || cc != 2_500 {
		t.Errorf("comfort: ratio %d, schedule %d, units %d, carry %d, want 11000, 8500, 1, 2500", r, g, cu, cc)
	}

	// Treat: ratios 9 500 and 11 000; shares 5 728 and 4 272; want ratio 10 140;
	// g = 10 000 + 20 000 * (-140) / 10 000 = 9 720; need 1 944; carry 9 000 + 1 944 = 10 944.
	sa := household.WantItem{Centre: 380, Ref: 400, Weight: 10_000}
	sb := household.WantItem{Centre: 550, Ref: 500, Weight: 10_000}
	ra, _ := household.PriceRatio(sa.Centre, sa.Ref)
	rb, _ := household.PriceRatio(sb.Centre, sb.Ref)
	shares, _ := household.Shares(nil, []household.WantItem{sa, sb}, 2)
	rk, _ := household.WantRatio([]household.WantItem{sa, sb}, shares)
	gk, _ := household.Schedule(rk, 20_000, 15_000)
	tu, tc, _ := household.Units(9_000, 1, 1, 10_000, 10_000, gk, 5)
	if ra != 9_500 || rb != 11_000 || !slices.Equal(shares, []int64{5_728, 4_272}) || rk != 10_140 || gk != 9_720 || tu != 1 || tc != 944 {
		t.Errorf("treat: ratios %d %d, shares %v, want ratio %d, schedule %d, units %d, carry %d, want 9500 11000 [5728 4272] 10140 9720 1 944",
			ra, rb, shares, rk, gk, tu, tc)
	}

	// The ladder: comfort buys its unit at the limit 160 (B 650 -> 490); the treat unit
	// goes to sweet_a (5 728 against 4 272), limit 410 within 490: B 80. The limit
	// of sweet_b is not in the example; 450 is used and does not matter.
	out := make([]int64, 3)
	left, ok := household.Fund(spendCap, 0, 3_200, []household.Rung{
		{Units: cu, Shares: []int64{10_000}, Limits: []int64{160}},
		{Units: tu, Shares: shares, Limits: []int64{410, 450}},
	}, out)
	if !ok || left != 80 || !slices.Equal(out, []int64{1, 1, 0}) {
		t.Errorf("Fund = %v, left %d, %v, want [1 1 0], left 80, true (650 - 160 - 410)", out, left, ok)
	}
	// After the comfort unit alone the budget is 490.
	out = out[:1]
	if left, ok = household.Fund(spendCap, 0, 3_200, []household.Rung{{Units: 1, Shares: []int64{10_000}, Limits: []int64{160}}}, out); !ok || left != 490 {
		t.Errorf("Fund(comfort only) left %d, %v, want 490 (650 - 160)", left, ok)
	}

	// Choke: centres 700 and 800 give ratios 17 500 and 16 000. The example takes the
	// want ratio 16 859 from the shares above: (5 728 * 17 500 + 4 272 * 16 000) / 10 000
	// = 168 592 000 / 10 000 = 16 859.2. The choke price is 10 000 + 10^8 / 20 000 = 15 000.
	ha, _ := household.PriceRatio(700, 400)
	hb, _ := household.PriceRatio(800, 500)
	hk, _ := household.WantRatio([]household.WantItem{{Centre: 700, Ref: 400}, {Centre: 800, Ref: 500}}, []int64{5_728, 4_272})
	hg, _ := household.Schedule(hk, 20_000, 15_000)
	if ha != 17_500 || hb != 16_000 || hk != 16_859 || hg != 0 {
		t.Errorf("choke: ratios %d %d, want ratio %d, schedule %d, want 17500 16000 16859 0", ha, hb, hk, hg)
	}
	// The actual shares at those prices (4 552 and 5 448) give 16 682, also choked.
	real, _ := household.Shares(nil, []household.WantItem{{Centre: 700, Ref: 400, Weight: 10_000}, {Centre: 800, Ref: 500, Weight: 10_000}}, 2)
	rr, _ := household.WantRatio([]household.WantItem{{Centre: 700, Ref: 400}, {Centre: 800, Ref: 500}}, real)
	if rg, _ := household.Schedule(rr, 20_000, 15_000); rg != 0 || rr != 16_682 {
		t.Errorf("choke with its own shares %v: want ratio %d, schedule %d, want 16682 and 0", real, rr, rg)
	}
}

// TestConsumed_Guard verifies that a want consumes at most its planned units and
// at most what is held above the tick's upkeep of the same commodity.
//
// Requirement: FR-DEM-025
//
// Guard: D32. Without the subtraction a want would eat the upkeep of the next
// tick and a household would go hungry because of a luxury.
func TestConsumed_Guard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		want, held, upkeep int64
		got                int64
	}{
		// min(3, 10 - 8) = 2.
		{"upkeep guard binds", 3, 10, 8, 2},
		{"held equals upkeep", 3, 8, 8, 0},
		{"held below upkeep", 3, 5, 8, 0},
		// min(3, 10 - 0) = 3.
		{"not an upkeep commodity", 3, 10, 0, 3},
		{"ample stock", 3, 100, 8, 3},
		{"no planned units", 0, 100, 8, 0},
		{"negative arguments count as 0", -2, -5, -1, 0},
		// min(4, 4 - 0) = 4.
		{"negative upkeep counts as 0", 4, 4, -3, 4},
		{"negative stock counts as 0", 4, -5, 0, 0},
		{"extreme values", math.MaxInt64, math.MaxInt64, 0, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.Consumed(tt.want, tt.held, tt.upkeep); got != tt.got {
				t.Errorf("Consumed(%d, %d, %d) = %d, want %d", tt.want, tt.held, tt.upkeep, got, tt.got)
			}
		})
	}
}

// TestInitialCarry_Formula verifies (AgentIndex * 7 919 + position * 2 777) mod
// 10 000.
//
// Requirement: FR-DEM-013
func TestInitialCarry_Formula(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		idx, pos int64
		want     int64
		ok       bool
	}{
		{"first agent, first want", 0, 0, 0, true},
		{"second agent", 1, 0, 7_919, true},
		// 7 919 + 2 777 = 10 696, mod 10 000 = 696.
		{"wraps", 1, 1, 696, true},
		// 2 * 7 919 = 15 838, mod 10 000 = 5 838.
		{"agent 2", 2, 0, 5_838, true},
		// 3 * 7 919 + 2 * 2 777 = 23 757 + 5 554 = 29 311, mod 10 000 = 9 311.
		{"agent 3 position 2", 3, 2, 9_311, true},
		// 3 * 2 777 = 8 331.
		{"position only", 0, 3, 8_331, true},
		{"negative counts as 0", -4, -1, 0, true},
		{"agent product overflows", 1 << 62, 0, 0, false},
		{"position product overflows", 0, 1 << 62, 0, false},
		{"sum overflows", math.MaxInt64 / 7_919, math.MaxInt64 / 2_777, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.InitialCarry(tt.idx, tt.pos)
			if got != tt.want || ok != tt.ok {
				t.Errorf("InitialCarry(%d, %d) = %d, %v, want %d, %v", tt.idx, tt.pos, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestKeepCarries_RoleChange verifies that on a role change the carries of
// wants in both ladders are kept by want ID, a want only in the new ladder
// starts at the formula with its position in the new ladder, and a want only in
// the old ladder is dropped.
//
// Requirement: FR-DEM-042
func TestKeepCarries_RoleChange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                       string
		oldIDs, oldCarries, newIDs []int64
		idx                        int64
		want                       []int64
		ok                         bool
	}{
		// Wants 9 and 5 are in both ladders (new order 9, 4, 5): 333 and 111 kept; 4 is
		// new at position 1 for agent 2: 2 * 7 919 + 1 * 2 777 = 18 615, mod 10 000 = 8 615.
		// Want 7 is only in the old ladder and is dropped.
		{"keep, add, drop", []int64{5, 7, 9}, []int64{111, 222, 333}, []int64{9, 4, 5}, 2, []int64{333, 8_615, 111}, true},
		// Both new: positions 0 and 1 for agent 0: 0 and 2 777.
		{"no old ladder", nil, nil, []int64{1, 2}, 0, []int64{0, 2_777}, true},
		{"no new ladder", []int64{1}, []int64{5}, nil, 3, []int64{}, true},
		{"same ladder keeps all", []int64{1, 2}, []int64{10, 20}, []int64{1, 2}, 9, []int64{10, 20}, true},
		{"more ids than carries", []int64{1, 2}, []int64{10}, []int64{1}, 0, []int64{}, false},
		{"more carries than ids", []int64{1}, []int64{10, 20}, []int64{1}, 0, []int64{}, false},
		// A repeated old ID is invalid input; the first carry is the one kept.
		{"repeated old id keeps the first", []int64{1, 1}, []int64{10, 20}, []int64{1}, 0, []int64{10}, true},
		{"initial carry overflows", nil, nil, []int64{1}, 1 << 62, []int64{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.KeepCarries([]int64{8, 8, 8, 8}, tt.oldIDs, tt.oldCarries, tt.newIDs, tt.idx)
			if ok != tt.ok || !slices.Equal(got, tt.want) {
				t.Errorf("KeepCarries(%v, %v, %v, %d) = %v, %v, want %v, %v", tt.oldIDs, tt.oldCarries, tt.newIDs, tt.idx, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestTiers_T4 verifies that the want units are tier 4: tier 3 is the gap of the
// keep without the want, tier 4 is the rest, the four tiers add to the gap, and
// tier 4 never exceeds the want.
//
// Requirement: FR-DEM-016, FR-DEM-019
func TestTiers_T4(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                            string
		keep, stock, reserve, job, want int64
		gap, surplus, t1, t2, t3, t4    int64
	}{
		// Keep 16 = reserve 3 + use 9 + want 4. Gap 16 - 4 = 12; t1 = min(12, 0) = 0;
		// left 1, t2 = min(12, max(0, 0 - 1)) = 0; t3 = min(12, 16 - 4 - 4) = 8; t4 = 12 - 8 = 4.
		{"want on top of use", 16, 4, 3, 0, 4, 12, 0, 0, 0, 8, 4},
		// Keep 16 = 3 + 9 + 4 with job 3: gap 15; t1 = min(15, 3 - 1) = 2; left 0;
		// t2 = min(13, 3) = 3; t3 = min(15, 16 - 4 - 1) - 5 = 11 - 5 = 6; t4 = 15 - 11 = 4.
		{"all four tiers", 16, 1, 3, 3, 4, 15, 0, 2, 3, 6, 4},
		// The stock 14 fills the use first: gap 2, keep without want 12 is met, t3 = 0, t4 = 2 of 4.
		{"stock covers the use, half the want", 16, 14, 3, 0, 4, 2, 0, 0, 0, 0, 2},
		// A want alone: keep 4, nothing held: t3 = min(4, 0) = 0, t4 = 4.
		{"want alone", 4, 0, 0, 0, 4, 4, 0, 0, 0, 0, 4},
		// Want above the keep (inconsistent input): t3 = min(2, max(0, 2 - 5 - 0)) = 0, t4 = 2.
		{"want above keep", 2, 0, 0, 0, 5, 2, 0, 0, 0, 0, 2},
		{"want met by stock", 4, 6, 0, 0, 4, 0, 2, 0, 0, 0, 0},
		// Want 0 is the Stage 1 split: t4 = 0, t3 = 12 - 0 - 0.
		{"no want", 12, 0, 0, 0, 0, 12, 0, 0, 0, 12, 0},
		{"negative want counts as 0", 12, 0, 0, 0, -3, 12, 0, 0, 0, 12, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gap, surplus, t1, t2, t3, t4 := household.Tiers(tt.keep, tt.stock, tt.reserve, tt.job, tt.want)
			if gap != tt.gap || surplus != tt.surplus || t1 != tt.t1 || t2 != tt.t2 || t3 != tt.t3 || t4 != tt.t4 {
				t.Errorf("Tiers(%d, %d, %d, %d, %d) = %d, %d, %d, %d, %d, %d, want %d, %d, %d, %d, %d, %d",
					tt.keep, tt.stock, tt.reserve, tt.job, tt.want, gap, surplus, t1, t2, t3, t4,
					tt.gap, tt.surplus, tt.t1, tt.t2, tt.t3, tt.t4)
			}
		})
	}

	t.Run("add to the gap over a grid", func(t *testing.T) {
		t.Parallel()
		vals := []int64{-1, 0, 1, 2, 3, 5, 8, 13}
		for _, keep := range vals {
			for _, stock := range vals {
				for _, reserve := range vals {
					for _, job := range vals {
						for _, want := range vals {
							gap, _, t1, t2, t3, t4 := household.Tiers(keep, stock, reserve, job, want)
							if t1 < 0 || t2 < 0 || t3 < 0 || t4 < 0 || t1+t2+t3+t4 != gap || t4 > max(want, 0) {
								t.Fatalf("Tiers(%d, %d, %d, %d, %d) = gap %d, %d, %d, %d, %d: want non-negative tiers adding to the gap with t4 <= want",
									keep, stock, reserve, job, want, gap, t1, t2, t3, t4)
							}
							// Tiers 1 to 3 do not depend on the want when keep includes it.
							if g0, _, a0, b0, _, _ := household.Tiers(keep, stock, reserve, job, 0); g0 != gap || a0 != t1 || b0 != t2 {
								t.Fatalf("Tiers(%d, %d, %d, %d, %d): t1, t2 = %d, %d differ from the no-want %d, %d", keep, stock, reserve, job, want, t1, t2, a0, b0)
							}
						}
					}
				}
			}
		}
	})
}

// bound reduces v to [lo, hi] without a panic, for any int64.
func bound(v, lo, hi int64) int64 {
	if v < 0 {
		v = -(v + 1)
	}
	return lo + v%(hi-lo+1)
}

// FuzzWants verifies, for inputs within the validation bounds, that nothing
// panics or overflows, that the shares add to 10 000, that the schedule does not
// rise with the price, that the carry stays below 10 000 and that the funded
// units never cost more than the budget.
//
// Requirement: NFR-DEM-004
func FuzzWants(f *testing.F) {
	// Seeds: the worked example 15.1, a choke, and extreme bounds.
	f.Add(int64(380), int64(400), int64(550), int64(500), int64(10_000), int64(10_000), int64(2), int64(20_000), int64(15_000), int64(9_000), int64(1), int64(1), int64(5), int64(650), int64(3_200), int64(410), int64(450))
	f.Add(int64(700), int64(400), int64(800), int64(500), int64(1), int64(10_000), int64(4), int64(100_000), int64(50_000), int64(9_999), int64(1_000), int64(1_000_000), int64(100_000), int64(0), int64(0), int64(1), int64(1))
	f.Add(int64(1), int64(1_000_000_000_000), int64(1_000_000_000_000), int64(1), int64(10_000), int64(1), int64(4), int64(0), int64(10_000), int64(0), int64(1), int64(1), int64(1), int64(1<<40), int64(1<<40), int64(1), int64(1<<40))
	f.Fuzz(func(t *testing.T, c1, r1, c2, r2, w1, w2, sigma, el, ceil, carry, q, w, period, budget, avail, l1, l2 int64) {
		const maxPrice = 1_000_000_000_000
		// The belief centre lies in the reference band (spec-beliefs.md 3.2), so the
		// ratio is bounded; 100 times the reference is far wider than any band.
		ref1, ref2 := bound(r1, 1, maxPrice), bound(r2, 1, maxPrice)
		items := []household.WantItem{
			{Centre: bound(c1, 1, 100*ref1), Ref: ref1, Weight: bound(w1, 1, 10_000)},
			{Centre: bound(c2, 1, 100*ref2), Ref: ref2, Weight: bound(w2, 1, 10_000)},
		}
		shares, ok := household.Shares(nil, items, bound(sigma, 0, 4))
		if !ok {
			// Only a power that leaves int64 may fail within the bounds, and then no shares are given.
			if len(shares) != 0 {
				t.Fatalf("Shares(%v) = %v, false: want no shares on failure", items, shares)
			}
			return
		}
		if sum := shares[0] + shares[1]; sum != 10_000 || shares[0] < 0 || shares[1] < 0 {
			t.Fatalf("Shares(%v, %d) = %v: want non-negative and adding to 10000", items, sigma, shares)
		}
		ratio, ok := household.WantRatio(items, shares)
		if !ok {
			t.Fatalf("WantRatio(%v, %v) overflowed within the bounds", items, shares)
		}
		elast, ceiling := bound(el, 0, 100_000), bound(ceil, 10_000, 50_000)
		g1, ok1 := household.Schedule(ratio, elast, ceiling)
		g2, ok2 := household.Schedule(ratio+bound(q, 0, 1_000_000), elast, ceiling)
		if !ok1 || !ok2 || g1 < 0 || g1 > ceiling || g2 > g1 {
			t.Fatalf("Schedule(%d, %d, %d) = %d, %v then dearer %d, %v: want in [0, ceiling] and not rising", ratio, elast, ceiling, g1, ok1, g2, ok2)
		}
		units, left, ok := household.Units(bound(carry, 0, 9_999), bound(q, 1, 1_000), bound(w, 1, 1_000_000), 100_000, 100_000, g1, bound(period, 1, 100_000))
		if !ok || left < 0 || left >= 10_000 || units < 0 {
			t.Fatalf("Units within the bounds = %d, %d, %v: want units >= 0, carry in [0, 9999], ok", units, left, ok)
		}
		b, a := bound(budget, 0, 1<<50), bound(avail, 0, 1<<50)
		ladder := []household.Rung{{Units: units, Shares: shares, Limits: []int64{bound(l1, 0, 1<<30), bound(l2, 0, 1<<30)}}}
		out := make([]int64, 2)
		rest, ok := household.Fund(b, 0, a, ladder, out)
		spent := out[0]*ladder[0].Limits[0] + out[1]*ladder[0].Limits[1]
		if !ok || out[0]+out[1] > units || rest < 0 || spent != min(b, a)-rest {
			t.Fatalf("Fund(%d, 0, %d, %v) = %v, left %d, %v: want spent %d = budget %d less left", b, a, ladder, out, rest, ok, spent, min(b, a))
		}
	})
}

// BenchmarkWants reports the time and allocations of the pure rules of one
// household with the two wants of the worked example 15.1.
//
// Requirement: NFR-DEM-003
func BenchmarkWants(b *testing.B) {
	b.ReportAllocs()
	items := []household.WantItem{{Centre: 380, Ref: 400, Weight: 10_000}, {Centre: 550, Ref: 500, Weight: 10_000}}
	shares := make([]int64, 0, 2)
	out := make([]int64, 3)
	limits := []int64{410, 450}
	for b.Loop() {
		shares, _ = household.Shares(shares, items, 2)
		r, _ := household.WantRatio(items, shares)
		g, _ := household.Schedule(r, 20_000, 15_000)
		u, _, _ := household.Units(9_000, 1, 1, 10_000, 10_000, g, 5)
		household.Fund(650, 0, 3_200, []household.Rung{{Units: u, Shares: shares, Limits: limits}}, out[:2])
		household.Consumed(u, 10, 8)
	}
}

// TestSplit verifies that Split divides the units among the shares by largest
// remainder, ties to the lower index, so the parts add to the total, and refuses
// shares that add to more than 10 000.
//
// Requirement: FR-DEM-018
func TestSplit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		total  int64
		shares []int64
		want   []int64
		ok     bool
	}{
		{"two thirds", 3, []int64{6_667, 3_333}, []int64{2, 1}, true},
		{"tie goes to the lower index", 1, []int64{5_000, 5_000}, []int64{1, 0}, true},
		{"worked example", 7, []int64{5_728, 4_272}, []int64{4, 3}, true},
		{"one item", 5, []int64{10_000}, []int64{5}, true},
		{"negative total is zero", -4, []int64{10_000}, []int64{0}, true},
		{"over 10 000", 3, []int64{9_000, 9_000}, nil, false},
		// With a total of 1 the floors are 0, so only the guard on the sum refuses.
		{"over 10 000, nothing floored", 1, []int64{9_000, 9_000}, nil, false},
		// Two passes, the first item first: the pass after it must skip it.
		{"two remainders, first item largest", 2, []int64{3_334, 3_333, 3_333}, []int64{1, 1, 0}, true},
		// Floors [0, 1] leave 2 units; the zero share must not take the second.
		{"short weights never feed a zero share", 3, []int64{0, 5_000}, nil, false},
		{"a zero share has no remainder", 1, []int64{0, 1}, []int64{0, 1}, true},
	}
	for _, tt := range tests {
		got, ok := household.Split(nil, tt.total, tt.shares)
		if ok != tt.ok || (ok && !slices.Equal(got, tt.want)) {
			t.Errorf("%s: Split(%d, %v) = %v, %v, want %v, %v", tt.name, tt.total, tt.shares, got, ok, tt.want, tt.ok)
		}
	}
}
