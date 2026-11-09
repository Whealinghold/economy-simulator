package indicator_test

import (
	"errors"
	"math"
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
