package indicator_test

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// TestRowsOf_AbsentWeightPastInt64IsOverflow verifies that the weights of the
// agents without the column add up in 128 bits: two absent agents of the largest
// weight are ErrOverflow, not a wrapped count.
//
// Fix: review round 5. Absent was an int64 sum that wrapped silently.
//
// Requirement: FR-IND-009
func TestRowsOf_AbsentWeightPastInt64IsOverflow(t *testing.T) {
	t.Parallel()
	st := []indicator.AgentRow{{Weight: math.MaxInt64}, {Weight: math.MaxInt64}} // no welfare ratio: both are absent
	if _, err := indicator.RowsOf(st, indicator.ColWelfareRatio, indicator.RowOpt{}); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("RowsOf(two absent agents of weight MaxInt64) error = %v, want ErrOverflow", err)
	}
}

// TestTally_PeopleEventsPastInt64AreOverflow verifies that an event whose
// weight cannot be negated, or whose change does not fit int64, makes Close
// ErrOverflow instead of wrapping the households count.
//
// Fix: review round 5. -Weight and To-From wrapped silently in the tally.
//
// Requirement: FR-IND-009
func TestTally_PeopleEventsPastInt64AreOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		e    economysim.Event
	}{
		{"bankruptcy of the minimum weight", economysim.AgentBankrupted{Tick: 10, Region: alpha, Role: grower, Weight: math.MinInt64}},
		{"weight change past int64", economysim.WeightChanged{Tick: 10, Region: alpha, Role: grower, From: -1, To: math.MaxInt64}},
		{"role change of the minimum weight", economysim.RoleChanged{Tick: 10, Region: alpha, From: grower, To: baker, Weight: math.MinInt64}},
		{"relocation of the minimum weight", economysim.AgentRelocated{Tick: 10, Role: grower, From: alpha, To: beta, Weight: math.MinInt64}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tl := indicator.NewTally()
			openValley(t, tl, 10, [4]int64{})
			step(tl, 10, tt.e)
			if _, err := tl.Close(valleySnap(11, [4]int64{})); !errors.Is(err, indicator.ErrOverflow) {
				t.Errorf("Close() after %T error = %v, want ErrOverflow", tt.e, err)
			}
		})
	}
}

// TestBasketResult_RatioWithAZeroReferenceCostIsUndefined verifies that a
// region whose reference cost is 0 has no level, so no ratio against it or from
// it is defined.
//
// Fix: review round 5. A zero reference cost on the against side returned a
// defined ratio of 0.
//
// Requirement: FR-IND-033
func TestBasketResult_RatioWithAZeroReferenceCostIsUndefined(t *testing.T) {
	t.Parallel()
	r := indicator.BasketResult{Rows: []indicator.BasketRow{{Region: 0, Cost: 5}, {Region: 1, Cost: 6, RefCost: 3}}}
	for _, p := range [][2]economysim.RegionID{{1, 0}, {0, 1}} {
		if got, err := r.Ratio(p[0], p[1]); err != nil || got.OK {
			t.Errorf("Ratio(%d, %d) = %+v, %v, want undefined", p[0], p[1], got, err)
		}
	}
}

// TestPriceGap_DeliveredResultOfAnotherWindowIsRejected verifies that lanes of
// another world or another window are not mixed into the gaps.
//
// Guard: review round 5. The lanes were read without a check, so freight of a
// different world gave a gap after freight with no error.
//
// Requirement: FR-IND-035
func TestPriceGap_DeliveredResultOfAnotherWindowIsRejected(t *testing.T) {
	t.Parallel()
	w := indicator.Window{ConfigHash: valleyHash, Regions: 2, Commodities: 1, Regional: make([]indicator.RegionTally, 2)}
	p, err := indicator.NewPrices(valleyHash, 0, 0, 2, 1, []economysim.Price{10, 13})
	if err != nil {
		t.Fatalf("NewPrices() error = %v, want nil", err)
	}
	g := graphOf(t, []string{"a", "b"}, economysim.Edge{From: "a", To: "b", Distance: 1}, economysim.Edge{From: "b", To: "a", Distance: 1})
	lane := []indicator.Lane{{Origin: 0, Destination: 1, Delivered: 11}}
	tests := []struct {
		name string
		dl   indicator.DeliveredResult
		want error
	}{
		{"another world", indicator.DeliveredResult{ConfigHash: [32]byte{1}, Lanes: lane}, indicator.ErrConfigMismatch},
		{"another window", indicator.DeliveredResult{ConfigHash: valleyHash, From: 5, To: 6, Lanes: lane}, indicator.ErrWindowOrder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := indicator.PriceGap(w, p, tt.dl, g); !errors.Is(err, tt.want) {
				t.Errorf("PriceGap(%s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestMarketPresets_HandBuiltWindowIsRangeNotPanic verifies that a window built
// by hand with market rows but no commodity count, or with fewer belief rows
// than markets, is ErrRange and not a division by zero or an index panic.
//
// Fix: review round 5. The presets divided by Commodities and indexed Beliefs
// by the market index without a check.
//
// Requirement: FR-IND-046, FR-IND-047
func TestMarketPresets_HandBuiltWindowIsRangeNotPanic(t *testing.T) {
	t.Parallel()
	noCommodities := indicator.Window{Regions: 1, Markets: make([]indicator.MarketTally, 2)}
	noBeliefs := indicator.Window{Regions: 1, Commodities: 1, Markets: make([]indicator.MarketTally, 1)}
	if _, err := indicator.PriceRange(noCommodities, indicator.CellSel{}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("PriceRange(markets, no commodity count) error = %v, want ErrRange", err)
	}
	if _, err := indicator.Pinned(noBeliefs, indicator.CellSel{}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Pinned(no belief rows) error = %v, want ErrRange", err)
	}
	if _, err := indicator.Dispersion(noBeliefs, indicator.CellSel{}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Dispersion(no belief rows) error = %v, want ErrRange", err)
	}
}

// TestHandBuiltWindow_ShortTablesAreRangeNotPanic verifies that a window whose
// regional, people or recipe table is shorter than its names table is ErrRange
// in every preset that indexes it, not an index panic or a division by Roles 0.
//
// Fix: review round 6. TradeBalance, GDP, Sectors, Households, Unfed,
// RoleShares and ValueAdded indexed the tables of a hand-built window unchecked.
//
// Requirement: FR-IND-038, FR-IND-041, FR-IND-042, FR-IND-045
func TestHandBuiltWindow_ShortTablesAreRangeNotPanic(t *testing.T) {
	t.Parallel()
	p, err := indicator.NewPrices([32]byte{}, 0, 0, 1, 1, []economysim.Price{1})
	if err != nil {
		t.Fatalf("NewPrices() error = %v, want nil", err)
	}
	noRegional := indicator.Window{Regions: 1, Commodities: 1}
	noRoles := indicator.Window{People: make([]indicator.PeopleTally, 1)}
	tests := map[string]error{}
	_, tests["TradeBalance"] = indicator.TradeBalance(noRegional, p, indicator.GDPResult{})
	_, tests["GDP"] = indicator.GDP(noRegional, economysim.Snapshot{}, economysim.Snapshot{}, p)
	_, tests["RoleShares"] = indicator.RoleShares(noRoles)
	_, tests["Unfed"] = indicator.Unfed(noRoles, economysim.Snapshot{})
	_, tests["Households"] = indicator.Households(noRoles, economysim.Snapshot{}, economysim.Snapshot{}, 1)
	for name, err := range tests {
		if !errors.Is(err, indicator.ErrRange) {
			t.Errorf("%s(short table) error = %v, want ErrRange", name, err)
		}
	}
}

// TestPriceRange_ClampedAndNonPositiveLow verifies that a clamped range is
// flagged on the result and that a low at or below 0 has no range.
//
// Fix: review round 6. The range was clamped without WarnClamped, and a negative
// low gave a signed range.
//
// Requirement: FR-IND-023, FR-IND-046
func TestPriceRange_ClampedAndNonPositiveLow(t *testing.T) {
	t.Parallel()
	w := indicator.Window{Regions: 1, Commodities: 2, Markets: []indicator.MarketTally{
		{Ticks: 1, Active: 1, Low: 1, High: 1 << 40}, {Ticks: 1, Active: 1, Low: -5, High: 7},
	}}
	got, err := indicator.PriceRange(w, indicator.CellSel{})
	if err != nil {
		t.Fatalf("PriceRange() error = %v, want nil", err)
	}
	if !slices.Contains(got.Warnings, indicator.WarnClamped) || !got.Rows[0].Range.Clamped {
		t.Errorf("PriceRange(high 2^40, low 1) = %+v, want WarnClamped and a clamped row", got)
	}
	if got.Rows[1].Range.OK {
		t.Errorf("PriceRange(low -5) range = %+v, want undefined", got.Rows[1].Range)
	}
}

// TestRowsOf_WealthHoldingOutsideTheTableIsRange verifies that a holding the
// price table does not cover is ErrRange, not valued at 0.
//
// Fix: review round 6. A missing table valued every holding at 0 silently.
//
// Requirement: FR-IND-026
func TestRowsOf_WealthHoldingOutsideTheTableIsRange(t *testing.T) {
	t.Parallel()
	st := []indicator.AgentRow{{Weight: 1, Money: 5, Holdings: []indicator.Holding{{Commodity: 0, Quantity: 2}}}}
	if _, err := indicator.RowsOf(st, indicator.ColWealth, indicator.RowOpt{}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("RowsOf(wealth, no table) error = %v, want ErrRange", err)
	}
}

// TestPrices_MisshapenWindowOrTableIsRange verifies that PricesOfWindow refuses
// a market table of another shape or a negative carried price, and BaseOf a table
// whose cells do not fit its shape.
//
// Fix: review round 6. Both built a table that read as 0 everywhere.
//
// Requirement: FR-IND-028
func TestPrices_MisshapenWindowOrTableIsRange(t *testing.T) {
	t.Parallel()
	if _, err := indicator.PricesOfWindow(indicator.Window{Regions: 2, Commodities: 1, Markets: make([]indicator.MarketTally, 1)}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("PricesOfWindow(1 market of 2 cells) error = %v, want ErrRange", err)
	}
	if _, err := indicator.PricesOfWindow(indicator.Window{Regions: 1, Commodities: 1, Markets: []indicator.MarketTally{{Price: -1}}}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("PricesOfWindow(negative carried price) error = %v, want ErrRange", err)
	}
	if _, err := indicator.BaseOf(indicator.Window{Regions: 1, Commodities: 1}, indicator.Prices{Regions: 1, Commodities: 1}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("BaseOf(table with no cells) error = %v, want ErrRange", err)
	}
}

// TestValueAdded_ShortRecipeTableIsRange verifies that a window whose recipe
// table is shorter than regions * recipes is ErrRange, not an index panic.
//
// Fix: review round 6. recipeRow indexed Recipe unchecked.
//
// Requirement: FR-IND-031
func TestValueAdded_ShortRecipeTableIsRange(t *testing.T) {
	t.Parallel()
	cfg := loadOne(t)
	h := economysim.HashConfig(cfg)
	snap := economysim.Snapshot{ConfigHash: h}
	cells := make([]economysim.Price, len(cfg.Regions)*len(cfg.Commodities))
	p, err := indicator.NewPrices(h, 0, 0, len(cfg.Regions), len(cfg.Commodities), cells)
	if err != nil {
		t.Fatalf("NewPrices() error = %v, want nil", err)
	}
	w := indicator.Window{ConfigHash: h, Regions: len(cfg.Regions), Commodities: len(cfg.Commodities), Recipes: len(cfg.Recipes),
		Regional: make([]indicator.RegionTally, len(cfg.Regions))}
	_, err = indicator.ValueAdded(w, cfg, snap, snap, p, indicator.NewClasses(cfg))
	if !errors.Is(err, indicator.ErrRange) || !strings.Contains(err.Error(), "recipe") {
		t.Errorf("ValueAdded(no recipe table) error = %v, want ErrRange on recipe", err)
	}
}

// Requirement: FR-IND-009 (a hand-built Window or row set returns an error or a
// result, never a panic or a wrapped value).
func TestNegativeDimsAndWideTotals(t *testing.T) {
	bad := indicator.Window{Regions: -1, Commodities: 2, Roles: 2, To: 1}
	if _, err := bad.Quantities(indicator.WeightsConsumed); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Quantities: %v, want indicator.ErrRange", err)
	}
	if _, err := indicator.StockCover(bad, economysim.Snapshot{Tick: 1}, indicator.CellSel{}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("StockCover: %v, want indicator.ErrRange", err)
	}
	if _, err := indicator.Unfed(bad, economysim.Snapshot{Tick: 1}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Unfed: %v, want indicator.ErrRange", err)
	}
	if _, err := indicator.Households(bad, economysim.Snapshot{}, economysim.Snapshot{Tick: 1}, 1); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("Households: %v, want indicator.ErrRange", err)
	}
	rows := []indicator.Household{{1, 1<<63 - 1}, {1, 1<<63 - 1}}
	if r, err := indicator.Stat(rows, indicator.StatQuantile, 5000); err != nil || !r.OK || r.Value != 1<<63-1 {
		t.Errorf("Quantile of totals past int64: %+v, %v", r, err)
	}
	if _, err := indicator.Stat(rows, indicator.StatSum, 0); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("Sum past int64: %v, want indicator.ErrOverflow", err)
	}
}
