package economysim

import (
	"math"
	"slices"
	"testing"
)

// TestWantUnits_Table verifies the most units a cohort wants in a tick,
// weight * Quantity * Ceiling * taste * region / (Period * 10^12), truncated, and
// that a product that does not fit int64 reports ok false instead of wrapping.
//
// Requirement: FR-DEM-003, FR-DEM-041
func TestWantUnits_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		w                   Want
		weight, taste, area int64
		want                int64
		wantOK              bool
	}{
		// 10 * 2 * 15 000 * 10 000 * 10 000 = 3 * 10^13; over 4 * 10^12 that is 7.5, truncated to 7.
		{"truncates", Want{Quantity: 2, Ceiling: 15_000, Period: 4}, 10, 10_000, 10_000, 7, true},
		{"zero taste", Want{Quantity: 2, Ceiling: 15_000, Period: 4}, 10, 0, 10_000, 0, true},
		{"first product overflows", Want{Quantity: math.MaxInt64 / 4, Ceiling: 50_000, Period: 1}, 1, 10_000, 10_000, 0, false},
		{"quotient overflows", Want{Quantity: 1_000_000, Ceiling: 50_000, Period: 1}, 1_000_000_000_000, 100_000, 100_000, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := wantUnits(&tt.w, tt.weight, tt.taste, tt.area)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("wantUnits(%+v, %d, %d, %d) = %d, %v, want %d, %v", tt.w, tt.weight, tt.taste, tt.area, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestOverflowOrValue verifies the Value of an ErrOverflow: the product as text,
// or the word "overflow" when it did not fit.
//
// Requirement: FR-DEM-003
func TestOverflowOrValue(t *testing.T) {
	t.Parallel()
	if got := overflowOrValue(42, true); got != "42" {
		t.Errorf("overflowOrValue(42, true) = %q, want %q", got, "42")
	}
	if got := overflowOrValue(0, false); got != overflowValue {
		t.Errorf("overflowOrValue(0, false) = %q, want %q", got, overflowValue)
	}
}

// TestWantSpend_Table verifies the per-household spending of a want at the mean
// reference price, ref = price * Quantity * taste * region / (Period * 10^8) and
// top the same times Ceiling / 10 000, and that an overflow in any product is
// reported rather than wrapped.
//
// Requirement: FR-DEM-033
func TestWantSpend_Table(t *testing.T) {
	t.Parallel()
	w := Want{Quantity: 2, Period: 4, Ceiling: 15_000}
	// 100 * 2 * 10 000 * 10 000 / (4 * 10^8) = 50; the top is 50 * 15 000 / 10 000 = 75.
	ref, top, ok := wantSpend(&w, 100, 10_000, 10_000)
	if ref != 50 || top != 75 || !ok {
		t.Errorf("wantSpend(price 100) = %d, %d, %v, want 50, 75, true", ref, top, ok)
	}
	if _, _, ok := wantSpend(&w, math.MaxInt64, 10_000, 10_000); ok {
		t.Errorf("wantSpend(price MaxInt64) ok = true, want false (the product of price and quantity overflows)")
	}
}

// demandLookup is a lookup of two commodities at the given reference prices, for
// the pure price rules. Its override map is nil, so every price is the reference.
func demandLookup(a, b Price) lookup {
	return lookup{commodity: map[string]*Commodity{"a": {ID: "a", ReferencePrice: a}, "b": {ID: "b", ReferencePrice: b}}}
}

// TestMeanRef_Table verifies the weighted mean of the reference prices of a
// want's items, truncated, and its refusals: no item with a weight and a declared
// commodity, and a product that overflows.
//
// Requirement: FR-DEM-033
func TestMeanRef_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		lk     lookup
		items  []WantItem
		want   int64
		wantOK bool
	}{
		// (10 000 * 100 + 5 000 * 301) / 15 000 = 2 505 000 / 15 000 = 167.0, truncated to 167.
		{"weighted mean truncates", demandLookup(100, 301), []WantItem{{Commodity: "a", Weight: 10_000}, {Commodity: "b", Weight: 5_000}}, 167, true},
		{"undeclared commodity is skipped", demandLookup(100, 301), []WantItem{{Commodity: "zz", Weight: 10_000}, {Commodity: "a", Weight: 10_000}}, 100, true},
		{"no usable item", demandLookup(100, 301), []WantItem{{Commodity: "zz", Weight: 10_000}, {Commodity: "a", Weight: 0}}, 0, false},
		{"line overflows", demandLookup(math.MaxInt64, 1), []WantItem{{Commodity: "a", Weight: 10_000}}, 0, false},
		{"sum overflows", demandLookup(math.MaxInt64/10_000, math.MaxInt64/10_000), []WantItem{{Commodity: "a", Weight: 10_000}, {Commodity: "b", Weight: 10_000}}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := meanRef(tt.lk, &Want{Items: tt.items}, "north")
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("meanRef(%v) = %d, %v, want %d, %v", tt.items, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestLadderSpend_Table verifies the spending of one household over its ladder:
// wref with every schedule at 10 000 and wmax at the ceilings, a want with no
// price skipped, no priced want not ok, and an overflow not ok.
//
// Requirement: FR-DEM-033
func TestLadderSpend_Table(t *testing.T) {
	t.Parallel()
	build := func(price Price) (lookup, *Role) {
		c := viaWanting(2, 4)
		c.Commodities[0].ReferencePrice = price
		cc := c.canonical()
		lk := newLookup(&cc)
		return lk, &cc.Roles[0]
	}
	t.Run("priced want", func(t *testing.T) {
		t.Parallel()
		lk, role := build(100)
		// 100 * 2 * 10 000 * 10 000 / (4 * 10^8) = 50; the default ceiling 15 000 makes the top 75.
		wref, wmax, ok := ladderSpend(lk, role, "north")
		if wref != 50 || wmax != 75 || !ok {
			t.Errorf("ladderSpend = %d, %d, %v, want 50, 75, true", wref, wmax, ok)
		}
	})
	t.Run("role without a ladder", func(t *testing.T) {
		t.Parallel()
		lk, role := build(100)
		other := *role
		other.ID = "nobody"
		if wref, wmax, ok := ladderSpend(lk, &other, "north"); ok || wref != 0 || wmax != 0 {
			t.Errorf("ladderSpend(no tastes) = %d, %d, %v, want 0, 0, false", wref, wmax, ok)
		}
	})
	t.Run("unpriced want", func(t *testing.T) {
		t.Parallel()
		lk, role := build(100)
		delete(lk.commodity, "grain") // the item no longer resolves, so the want has no price
		if _, _, ok := ladderSpend(lk, role, "north"); ok {
			t.Errorf("ladderSpend(unpriced) ok = true, want false")
		}
	})
	t.Run("overflow", func(t *testing.T) {
		t.Parallel()
		lk, role := build(100)
		lk.commodity["grain"].ReferencePrice = math.MaxInt64
		if _, _, ok := ladderSpend(lk, role, "north"); ok {
			t.Errorf("ladderSpend(overflow) ok = true, want false")
		}
	})
}

// TestMergeItems_SameCommodityAcrossWants verifies that two wants whose items
// name one commodity share one merged row whose units add, that rows come out in
// ascending commodity ID, and that an item planned at 0 units leaves no row.
//
// Requirement: FR-DEM-021
func TestMergeItems_SameCommodityAcrossWants(t *testing.T) {
	t.Parallel()
	d := &demandState{
		wants: []wantRT{
			{items: []wantItemRT{{commodity: 2}, {commodity: 1}}},
			{items: []wantItemRT{{commodity: 2}, {commodity: 3}}},
		},
		cur: wantCur{items: []wantUse{{commodity: 2, units: 4}, {commodity: 1, units: 0}, {commodity: 2, units: 5}, {commodity: 3, units: 7}}},
	}
	d.mergeItems(9, []rung{{want: 0}, {want: 1}})
	want := []wantUse{{agent: 9, commodity: 2, units: 9}, {agent: 9, commodity: 3, units: 7}}
	if !slices.Equal(d.merged, want) {
		t.Errorf("mergeItems merged = %v, want %v", d.merged, want)
	}
}
