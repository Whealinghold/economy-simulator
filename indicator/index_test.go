package indicator_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// foodWindow returns a window [10, 11) of the valley in which alpha consumed
// the quantities q1 of grain and loaf and every market traded, with the base
// of 18.4 (prices (8, 20), quantities q0 = (12, 18) consumed) and the current
// prices (10, 24). Wood and cart carry a quantity too, so that the all-goods
// index is a different number from the food index.
func foodWindow(t *testing.T, curGrain, curLoaf economysim.Price, volume economysim.Quantity) (indicator.Window, indicator.Base, indicator.Prices) {
	t.Helper()
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{})
	events := []economysim.Event{
		goodsFlow(10, alpha, economysim.ReasonConsumption, grain, 10), goodsFlow(10, alpha, economysim.ReasonConsumption, loaf, 20),
		goodsFlow(10, alpha, economysim.ReasonConsumption, wood, 4),
	}
	for c := economysim.CommodityID(0); c < 4; c++ {
		events = append(events, marketDay(10, alpha, c, 1, volume))
	}
	step(tl, 10, events...)
	w, err := tl.Close(valleySnap(11, [4]int64{}))
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	basePrices := valleyTable(t, w, [4]economysim.Price{8, 20, 5, 5}, [4]economysim.Price{1, 1, 1, 1})
	base := indicator.Base{Prices: basePrices, Consumed: make([]int64, 8), Traded: make([]int64, 8), Produced: make([]int64, 8)}
	base.Consumed[int(grain)], base.Consumed[int(loaf)], base.Consumed[int(wood)] = 12, 18, 3
	return w, base, valleyTable(t, w, [4]economysim.Price{curGrain, curLoaf, 6, 5}, [4]economysim.Price{1, 1, 1, 1})
}

// foodSel is the class food of the valley at alpha.
var foodSel = indicator.CellSel{Regions: []economysim.RegionID{alpha}, Commodities: []economysim.CommodityID{grain, loaf}}

// TestPriceIndex_Food verifies 18.4: base prices (8, 20), current (10, 24), base
// quantities (12, 18) and current (10, 20) give Laspeyres rdiv(10 000 * 552, 456)
// = 12 105, Paasche rdiv(10 000 * 580, 480) = 12 083 and Fisher the integer
// nearest to the square root of 12 105 * 12 083 = 146 264 715, 12 094; against
// its own base an index is 10 000; a zero denominator is undefined per index; a
// class and the all-goods index are separate calls.
//
// Requirement: FR-IND-020, FR-IND-021, FR-IND-032
func TestPriceIndex_Food(t *testing.T) {
	w, base, cur := foodWindow(t, 10, 24, 5)
	got, err := indicator.PriceIndex(w, base, indicator.WeightsConsumed, cur, foodSel)
	if err != nil {
		t.Fatalf("PriceIndex() error = %v, want nil", err)
	}
	want := map[string]indicator.Index{
		"laspeyres": {Value: 12105, OK: true}, "paasche": {Value: 12083, OK: true}, "fisher": {Value: 12094, OK: true},
	}
	for name, have := range map[string]indicator.Index{"laspeyres": got.Laspeyres, "paasche": got.Paasche, "fisher": got.Fisher} {
		if have != want[name] {
			t.Errorf("PriceIndex().%s = %+v, want %+v", name, have, want[name])
		}
	}
	if got.Weights != indicator.WeightsConsumed || got.ConfigHash != valleyHash {
		t.Errorf("PriceIndex() = weights %v hash %x, want consumed and the valley's", got.Weights, got.ConfigHash[:2])
	}

	own, err := indicator.PriceIndex(w, base, indicator.WeightsConsumed, base.Prices, foodSel)
	if err != nil || own.Laspeyres.Value != 10_000 || own.Paasche.Value != 10_000 || own.Fisher.Value != 10_000 {
		t.Errorf("PriceIndex(base prices) = %+v, %v, want 10 000 for each", own, err)
	}

	all, err := indicator.PriceIndex(w, base, indicator.WeightsConsumed, cur, indicator.CellSel{})
	if err != nil || all.Laspeyres.Value == got.Laspeyres.Value {
		t.Errorf("PriceIndex(all goods) Laspeyres = %d, %v, want a value different from the food index %d", all.Laspeyres.Value, err, got.Laspeyres.Value)
	}

	// Must not hide: a zero base value makes Laspeyres undefined without making Paasche so.
	zero := base
	zero.Consumed = make([]int64, 8)
	undefined, err := indicator.PriceIndex(w, zero, indicator.WeightsConsumed, cur, foodSel)
	if err != nil || undefined.Laspeyres.OK || undefined.Laspeyres.Value != 0 || !undefined.Paasche.OK || undefined.Fisher.OK {
		t.Errorf("PriceIndex(zero base quantities) = %+v, %v, want Laspeyres and Fisher undefined, Paasche defined", undefined, err)
	}
}

// TestPriceIndex_Misuse verifies the preconditions as errors: a table of another
// world, a base without the weights asked for, and unknown weights.
//
// Requirement: FR-IND-006
func TestPriceIndex_Misuse(t *testing.T) {
	w, base, cur := foodWindow(t, 10, 24, 5)
	other := cur
	other.ConfigHash = [32]byte{5}
	short := base
	short.Consumed = nil
	for name, tt := range map[string]struct {
		base indicator.Base
		k    indicator.Weights
		cur  indicator.Prices
		want error
	}{
		"current table of another world": {base, indicator.WeightsConsumed, other, indicator.ErrConfigMismatch},
		"base with no quantities":        {short, indicator.WeightsConsumed, cur, indicator.ErrRange},
		"unknown weights":                {base, indicator.Weights(9), cur, indicator.ErrRange},
	} {
		if _, err := indicator.PriceIndex(w, tt.base, tt.k, tt.cur, foodSel); !errors.Is(err, tt.want) {
			t.Errorf("PriceIndex(%s) error = %v, want %v", name, err, tt.want)
		}
	}
	if _, err := w.Quantities(indicator.Weights(9)); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Quantities(unknown weights) error = %v, want ErrRange", err)
	}
}

// TestPriceIndex_ChainFollowsTheRule verifies C_k = rdiv(C_{k-1} * I_k, 10 000)
// from C_0 = 10 000, each link rounded once half to even (10 001 * 5 000 =
// 50 005 000 is 5 000.5 and rounds to 5 000), that a clamped link makes every
// later point Clamped, and that an undefined link leaves the chain undefined.
//
// Requirement: FR-IND-022
func TestPriceIndex_ChainFollowsTheRule(t *testing.T) {
	ok := func(v indicator.BP) indicator.Index { return indicator.Index{Value: v, OK: true} }
	tests := []struct {
		name  string
		links []indicator.Index
		want  []indicator.Index
	}{
		{"links of 1.05, 0.98 and 1.2", []indicator.Index{ok(10500), ok(9800), ok(12000)}, []indicator.Index{ok(10500), ok(10290), ok(12348)}},
		{"an exact half goes to the even integer", []indicator.Index{ok(10001), ok(5000)}, []indicator.Index{ok(10001), ok(5000)}},
		{"a clamped link is sticky", []indicator.Index{{Value: 20000, OK: true, Clamped: true}, ok(10000), ok(5000)},
			[]indicator.Index{{Value: 20000, OK: true, Clamped: true}, {Value: 20000, OK: true, Clamped: true}, {Value: 10000, OK: true, Clamped: true}}},
		{"an undefined link breaks the chain", []indicator.Index{ok(10500), {}, ok(10000)}, []indicator.Index{ok(10500), {}, {}}},
		{"a chain past the bound is clamped", []indicator.Index{ok(indicator.MaxIndexBP), ok(20000)},
			[]indicator.Index{{Value: indicator.MaxIndexBP, OK: true, Clamped: true}, {Value: indicator.MaxIndexBP, OK: true, Clamped: true}}},
		{"no link", nil, []indicator.Index{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indicator.Chain(tt.links)
			if len(got) != len(tt.want) {
				t.Fatalf("Chain() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Chain()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestPriceIndex_StaleShareAndClamp verifies that an index reports the share of
// its base value whose prices were carried forward (the grain market traded
// nothing: 8*12 = 96 of the base value 96 + 360 = 456, 2 105 bp) with the Stale
// warning, and that an index at or above 2^31 - 1 is clamped and flagged, not an
// error.
//
// Requirement: FR-IND-023
func TestPriceIndex_StaleShareAndClamp(t *testing.T) {
	// Every market of foodWindow trades the same volume; build one where grain does not.
	tl := indicator.NewTally()
	openValley(t, tl, 10, [4]int64{})
	step(tl, 10, marketDay(10, alpha, grain, 10, 0), marketDay(10, alpha, loaf, 24, 6),
		goodsFlow(10, alpha, economysim.ReasonConsumption, grain, 10), goodsFlow(10, alpha, economysim.ReasonConsumption, loaf, 20))
	w, err := tl.Close(valleySnap(11, [4]int64{}))
	if err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	base := indicator.Base{Prices: valleyTable(t, w, [4]economysim.Price{8, 20, 1, 1}, [4]economysim.Price{1, 1, 1, 1}), Consumed: make([]int64, 8)}
	base.Consumed[int(grain)], base.Consumed[int(loaf)] = 12, 18
	cur := valleyTable(t, w, [4]economysim.Price{10, 24, 1, 1}, [4]economysim.Price{1, 1, 1, 1})
	got, err := indicator.PriceIndex(w, base, indicator.WeightsConsumed, cur, foodSel)
	if err != nil {
		t.Fatalf("PriceIndex() error = %v, want nil", err)
	}
	if got.Stale != 2105 || !got.StaleOK || len(got.Warnings) != 1 || got.Warnings[0] != indicator.WarnStale {
		t.Errorf("PriceIndex() stale = %d (%v) warnings %v, want 2105, true, [stale]", got.Stale, got.StaleOK, got.Warnings)
	}

	// Grain from 8 to 2 400 000: 300 000 times the base, past 2^31 - 1 = 214 748 times.
	huge := valleyTable(t, w, [4]economysim.Price{2_400_000, 24, 1, 1}, [4]economysim.Price{1, 1, 1, 1})
	clamped, err := indicator.PriceIndex(w, base, indicator.WeightsConsumed, huge, indicator.CellSel{Regions: []economysim.RegionID{alpha}, Commodities: []economysim.CommodityID{grain}})
	if err != nil {
		t.Fatalf("PriceIndex(huge) error = %v, want nil: a clamp is not an error", err)
	}
	if !clamped.Laspeyres.Clamped || clamped.Laspeyres.Value != indicator.MaxIndexBP || !clamped.Fisher.Clamped {
		t.Errorf("PriceIndex(huge) = Laspeyres %+v Fisher %+v, want clamped at MaxIndexBP", clamped.Laspeyres, clamped.Fisher)
	}
	if !contains(clamped.Warnings, indicator.WarnClamped) {
		t.Errorf("PriceIndex(huge).Warnings = %v, want Clamped", clamped.Warnings)
	}
}

// contains reports whether a warning is in the list.
func contains(ws []indicator.Warning, k indicator.Warning) bool {
	for _, w := range ws {
		if w == k {
			return true
		}
	}
	return false
}

// TestRealAndDeflator_Identity verifies that the nominal total equals its real
// value times the deflator up to the half basis point of the deflator's
// rounding: |N - rdiv(R * deflator, 10 000)| <= ceil(R / 20 000), over the 18.4
// totals (580 and 480) and a series of generated ones.
//
// Requirement: FR-IND-013
func TestRealAndDeflator_Identity(t *testing.T) {
	pair := func(basePrice, curPrice economysim.Price) (indicator.Prices, indicator.Prices) {
		b, _ := indicator.NewPrices(valleyHash, 0, 0, 1, 1, []economysim.Price{basePrice})
		c, _ := indicator.NewPrices(valleyHash, 0, 0, 1, 1, []economysim.Price{curPrice})
		return b, c
	}
	b, c := pair(8, 10) // the grain of 18.4 alone: R = 8 * 10, N = 10 * 10
	got, err := indicator.Real([]int64{10}, b, c, indicator.CellSel{})
	if err != nil || got.Nominal != 100 || got.Real != 80 || got.Deflator.Value != 12500 {
		t.Fatalf("Real() = %+v, %v, want nominal 100, real 80, deflator 12 500", got, err)
	}
	seed := uint64(7)
	next := func(n uint64) int64 {
		seed = seed*6364136223846793005 + 1442695040888963407
		return int64(seed >> 33 % n)
	}
	for i := 0; i < 500; i++ {
		b, c := pair(economysim.Price(1+next(5000)), economysim.Price(1+next(9000)))
		q := 1 + next(100_000)
		r, err := indicator.Real([]int64{q}, b, c, indicator.CellSel{})
		if err != nil || !r.Deflator.OK {
			t.Fatalf("Real(q=%d) = %+v, %v, want a deflator", q, r, err)
		}
		diff, bound := r.Nominal-evenDiv(r.Real*int64(r.Deflator.Value), 10_000), (r.Real+19_999)/20_000
		if diff < 0 {
			diff = -diff
		}
		if diff > bound {
			t.Fatalf("Real(q=%d) N=%d R=%d deflator=%d: |N - R*d/10 000| = %d, want <= %d", q, r.Nominal, r.Real, r.Deflator.Value, diff, bound)
		}
	}
	if _, err := indicator.Real([]int64{1, 2}, b, c, indicator.CellSel{}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Real(quantities of another shape) error = %v, want ErrRange", err)
	}
	other := c
	other.ConfigHash = [32]byte{1}
	if _, err := indicator.Real([]int64{1}, b, other, indicator.CellSel{}); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("Real(tables of two worlds) error = %v, want ErrConfigMismatch", err)
	}
	zero, _ := indicator.NewPrices(valleyHash, 0, 0, 1, 1, []economysim.Price{0})
	if u, err := indicator.Real([]int64{5}, zero, c, indicator.CellSel{}); err != nil || u.Deflator.OK {
		t.Errorf("Real(base price 0) = %+v, %v, want the deflator undefined", u, err)
	}
}

// evenDiv is n / d rounded half to even, for positive n and d: the oracle of the
// rule of 5.2.2 written for the test.
func evenDiv(n, d int64) int64 {
	q, r := n/d, n%d
	if 2*r > d || (2*r == d && q%2 == 1) {
		q++
	}
	return q
}
