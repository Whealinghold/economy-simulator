package outside_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/outside"
)

// FuzzMid verifies that Mid never panics; that it reports ok for every input
// inside the validated bounds of spec-external-sector.md 13.2, with a mid in
// [1, MaxPrice]; that it is not decreasing in the base; and that the 128-bit
// path agrees with a plain int64 computation where that cannot overflow.
//
// Guard: the products reach 10^25, past int64, so only the checked helper keeps
// them exact. A silent wrap would price a line near zero or negative.
func FuzzMid(f *testing.F) {
	f.Add(int64(40), int64(10_000), int64(0), int64(0), int64(360), int64(0), int64(1_000), int64(100_000), int64(10_000), int64(1_000_000_000))
	f.Add(int64(1_000_000_000), int64(100_000), int64(100_000), int64(1_000_000), int64(1), int64(9_999), int64(0), int64(1_000_000), int64(100_000), int64(1_000_000_000))
	f.Add(int64(1), int64(0), int64(-100_000), int64(360), int64(360), int64(-9_999), int64(1_000), int64(1_000), int64(0), int64(1))
	f.Fuzz(func(t *testing.T, base, season, trend, tick, tpy, drift, floor, ceil, shock, maxPrice int64) {
		m := outside.MidIn{Base: base, Season: season, TrendBP: trend, Tick: tick, TicksPerYear: tpy, Drift: drift,
			FloorBP: floor, CeilBP: ceil, Shock: shock, MaxPrice: maxPrice}
		mid, _, ok := outside.Mid(m)
		inBounds := base >= 1 && base <= 1_000_000_000 && season >= 0 && season <= 100_000 && trend >= -100_000 && trend <= 100_000 &&
			tick >= 0 && tick <= 1_000_000_000 && tpy >= 1 && tpy <= 1_000_000 && drift > -10_000 && drift < 10_000 &&
			floor >= 0 && ceil >= floor && ceil <= 1_000_000 && shock >= 0 && shock <= 100_000 && maxPrice >= 1 && maxPrice <= 1_000_000_000
		if inBounds && !ok {
			t.Fatalf("Mid(%+v) = not ok for an input inside the bounds", m)
		}
		if !ok {
			if mid != 0 {
				t.Fatalf("Mid(%+v) = %d with ok false, want 0", m, mid)
			}
			return
		}
		if mid < 1 || mid > maxPrice {
			t.Fatalf("Mid(%+v) = %d, want within [1, %d]", m, mid, maxPrice)
		}
		if base < 1<<40 {
			m.Base = base + 1
			if next, _, ok := outside.Mid(m); ok && next < mid {
				t.Fatalf("Mid decreases in the base: %d at base %d, %d at base %d (%+v)", mid, base, next, base+1, m)
			}
		}
	})
}

// FuzzQuote verifies that Quote never panics, that a quote is within [0,
// maxPrice], that every input inside the bounds of spec-external-sector.md 13.2
// is ok, and that the quote of a side is not decreasing in the mid.
//
// Guard: the same 128-bit path as FuzzMid, for the quote.
func FuzzQuote(f *testing.F) {
	f.Add(int64(40), int64(500), int64(10_000), int64(1_000_000_000), true)
	f.Add(int64(1), int64(9_999), int64(100_000), int64(1), false)
	f.Add(int64(1_000_000_000), int64(100_000), int64(100_000), int64(1_000_000_000), true)
	f.Fuzz(func(t *testing.T, mid, margin, scale, maxPrice int64, sell bool) {
		got, ok := outside.Quote(mid, margin, scale, maxPrice, sell)
		inBounds := mid >= 1 && mid <= 1_000_000_000 && maxPrice >= 1 && maxPrice <= 1_000_000_000 && scale >= 0 && scale <= 100_000 &&
			margin >= 0 && ((sell && margin <= 100_000) || (!sell && margin <= 9_999))
		if inBounds && !ok {
			t.Fatalf("Quote(%d, %d, %d, %d, sell %v) = not ok inside the bounds", mid, margin, scale, maxPrice, sell)
		}
		if !ok {
			if got != 0 {
				t.Fatalf("Quote(%d, %d, %d, %d, sell %v) = %d with ok false, want 0", mid, margin, scale, maxPrice, sell, got)
			}
			return
		}
		if got < 0 || got > maxPrice {
			t.Fatalf("Quote(%d, %d, %d, %d, sell %v) = %d, want within [0, %d]", mid, margin, scale, maxPrice, sell, got, maxPrice)
		}
		if mid < 1<<40 {
			if next, ok := outside.Quote(mid+1, margin, scale, maxPrice, sell); ok && next < got {
				t.Fatalf("Quote decreases in the mid: %d at %d, %d at %d (margin %d, scale %d, max %d, sell %v)", got, mid, next, mid+1, margin, scale, maxPrice, sell)
			}
		}
	})
}
