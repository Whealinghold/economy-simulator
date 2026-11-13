package room_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/room"
)

// example183 is the account of 18.3: A of mass 1 and bulk 3 in three lots of
// 10, B of mass 3 and bulk 1 in four lots of 10, a new room of (100, 100).
func example183() (load, rm room.Pair, cands []room.Candidate) {
	return room.Pair{Mass: 150, Bulk: 130}, room.Pair{Mass: 100, Bulk: 100}, []room.Candidate{
		{Commodity: 1, Size: room.Pair{Mass: 1, Bulk: 3}, Lots: []int64{10, 10, 10}},
		{Commodity: 2, Size: room.Pair{Mass: 3, Bulk: 1}, Lots: []int64{10, 10, 10, 10}},
	}
}

// TestExcessOrder_BindingDimensionFirst verifies example 18.3 pass by pass:
// mass binds first (50 of 100 against 30 of 100), B has the greater mass load
// and two lots go; then bulk binds, A has the greater bulk load and one lot
// goes. The moves are B lot 0, B lot 1, A lot 0.
//
// Requirement: FR-STO-018
func TestExcessOrder_BindingDimensionFirst(t *testing.T) {
	t.Parallel()
	load, rm, cands := example183()
	steps, met := room.ExcessOrder(load, rm, cands)
	want := []room.Step{{Cand: 1, Lot: 0, Units: 10}, {Cand: 1, Lot: 1, Units: 10}, {Cand: 0, Lot: 0, Units: 10}}
	if !met || !slices.Equal(steps, want) {
		t.Errorf("ExcessOrder(%v, %v) = %v, %v, want %v, true", load, rm, steps, met, want)
	}
}

// TestExcessOrder_ReevaluatedBeforeEachCommodity verifies that the ranking is
// taken again after a commodity, not continued: once B's two lots are gone the
// bulk dimension binds, and A, not B's remaining lots, goes (18.3 pass 2). An
// absolute comparison of a mass count with a bulk count would have kept to B.
//
// Requirement: FR-STO-018
func TestExcessOrder_ReevaluatedBeforeEachCommodity(t *testing.T) {
	t.Parallel()
	load, rm, cands := example183()
	steps, _ := room.ExcessOrder(load, rm, cands)
	var order []int
	for _, s := range steps {
		order = append(order, s.Cand)
	}
	if want := []int{1, 1, 0}; !slices.Equal(order, want) {
		t.Errorf("commodity order = %v, want %v (B, B, then A)", order, want)
	}
}

// TestExcessOrder_WholeLotsOldestFirst verifies that a lot goes whole even when
// less would do, and in lot order: a shortfall of 1 mass takes the whole first
// lot of 10.
//
// Requirement: FR-STO-018
func TestExcessOrder_WholeLotsOldestFirst(t *testing.T) {
	t.Parallel()
	cands := []room.Candidate{{Commodity: 7, Size: room.Pair{Mass: 1}, Lots: []int64{10, 5}}}
	steps, met := room.ExcessOrder(room.Pair{Mass: 101}, room.Pair{Mass: 100, Bulk: 100}, cands)
	if want := []room.Step{{Cand: 0, Lot: 0, Units: 10}}; !met || !slices.Equal(steps, want) {
		t.Errorf("ExcessOrder = %v, %v, want %v, true", steps, met, want)
	}
	// A shortfall of 11 takes both lots: the second goes whole too.
	steps, _ = room.ExcessOrder(room.Pair{Mass: 111}, room.Pair{Mass: 100, Bulk: 100}, cands)
	if want := []room.Step{{Cand: 0, Lot: 0, Units: 10}, {Cand: 0, Lot: 1, Units: 5}}; !slices.Equal(steps, want) {
		t.Errorf("ExcessOrder(shortfall 11) = %v, want %v", steps, want)
	}
}

// TestExcessOrder_TiesByCommodityID verifies that two commodities with equal
// load in the binding dimension go in the order of the lower ID, and that
// equal relative shortfalls bind mass.
//
// Requirement: FR-STO-018
func TestExcessOrder_TiesByCommodityID(t *testing.T) {
	t.Parallel()
	// Candidate order is deliberately the reverse of the IDs.
	cands := []room.Candidate{
		{Commodity: 9, Size: room.Pair{Mass: 2, Bulk: 2}, Lots: []int64{5}},
		{Commodity: 3, Size: room.Pair{Mass: 2, Bulk: 2}, Lots: []int64{5}},
	}
	steps, met := room.ExcessOrder(room.Pair{Mass: 20, Bulk: 20}, room.Pair{Mass: 10, Bulk: 10}, cands)
	// Shortfall (10, 10) in room (10, 10): a tie binds mass; commodity 3 goes first and its lot of 5 (load 10) meets it.
	if want := []room.Step{{Cand: 1, Lot: 0, Units: 5}}; !met || !slices.Equal(steps, want) {
		t.Errorf("ExcessOrder = %v, %v, want %v, true", steps, met, want)
	}
}

// TestExcessOrder_NotMetWhenNothingCanGo verifies the end of the order: a
// shortfall in a dimension in which no candidate has a size cannot be met, and
// the function says so instead of looping; a load within the room needs no step.
//
// Requirement: FR-STO-018
func TestExcessOrder_NotMetWhenNothingCanGo(t *testing.T) {
	t.Parallel()
	cands := []room.Candidate{{Commodity: 1, Size: room.Pair{Mass: 1}, Lots: []int64{4}}}
	steps, met := room.ExcessOrder(room.Pair{Mass: 1, Bulk: 9}, room.Pair{Mass: 5, Bulk: 5}, cands)
	if met || len(steps) != 0 {
		t.Errorf("ExcessOrder with a bulk shortfall and no bulk = %v, %v, want no steps, false", steps, met)
	}
	steps, met = room.ExcessOrder(room.Pair{Mass: 5, Bulk: 5}, room.Pair{Mass: 5, Bulk: 5}, cands)
	if !met || len(steps) != 0 {
		t.Errorf("ExcessOrder within the room = %v, %v, want no steps, true", steps, met)
	}
}

// TestExcessOrder_BindingByCrossMultiplication verifies the comparison of the
// relative shortfalls with a zero room (max(room, 1)) and with values whose
// products leave int64: the larger relative shortfall binds, no division.
//
// Requirement: FR-STO-018
func TestExcessOrder_BindingByCrossMultiplication(t *testing.T) {
	t.Parallel()
	const big = int64(1) << 62
	cands := []room.Candidate{
		{Commodity: 1, Size: room.Pair{Mass: 1}, Lots: []int64{1}},
		{Commodity: 2, Size: room.Pair{Bulk: 1}, Lots: []int64{1}},
	}
	// S = (big/2, big/2 - 1) in room (big, big): mass has the larger relative shortfall, so commodity 1 goes first.
	steps, _ := room.ExcessOrder(room.Pair{Mass: big + big/2, Bulk: big + big/2 - 1}, room.Pair{Mass: big, Bulk: big}, cands)
	if len(steps) == 0 || steps[0].Cand != 0 {
		t.Errorf("first step = %v, want the mass commodity", steps)
	}
	// Room 0 counts as 1: S = (1, 1) in room (0, 3): mass 1/1 beats bulk 1/3.
	steps, _ = room.ExcessOrder(room.Pair{Mass: 1, Bulk: 4}, room.Pair{Mass: 0, Bulk: 3}, cands)
	if len(steps) == 0 || steps[0].Cand != 0 {
		t.Errorf("first step with a zero mass room = %v, want the mass commodity", steps)
	}
}

// TestDiscardOrder_Tier1Only verifies 6.3.2: only the surplus offered as a
// candidate is touched, in whole units and the least that meets the shortfall
// in the binding dimension. A commodity not offered (no surplus: it is Keep'
// or a tier 2 need) is never a step, and the units discarded are the ceiling
// of the shortfall over the size, capped at the surplus.
//
// Surplus: 7 of C (size (2, 1)) and 20 of D (size (1, 1)). Shortfall (5, 0):
// mass binds, C has the greater mass load (14 against 20? D's 20 is greater), so D
// goes: ceil(5/1) = 5 units.
//
// Requirement: FR-STO-024
func TestDiscardOrder_Tier1Only(t *testing.T) {
	t.Parallel()
	cands := []room.Candidate{
		{Commodity: 4, Size: room.Pair{Mass: 2, Bulk: 1}, Lots: []int64{7}},
		{Commodity: 5, Size: room.Pair{Mass: 1, Bulk: 1}, Lots: []int64{20}},
	}
	steps, met := room.DiscardOrder(room.Pair{Mass: 5}, room.Pair{Mass: 50, Bulk: 50}, cands)
	if want := []room.Step{{Cand: 1, Lot: 0, Units: 5}}; !met || !slices.Equal(steps, want) {
		t.Errorf("DiscardOrder = %v, %v, want %v, true", steps, met, want)
	}
	// A shortfall the surplus cannot meet takes all of it and says not met.
	steps, met = room.DiscardOrder(room.Pair{Mass: 100}, room.Pair{Mass: 50, Bulk: 50}, cands)
	if met {
		t.Errorf("DiscardOrder(shortfall 100) met = true, want false: the surplus is 34 mass")
	}
	var taken int64
	for _, s := range steps {
		taken += s.Units
	}
	if taken != 27 {
		t.Errorf("DiscardOrder(shortfall 100) took %d units, want all 27 of the surplus", taken)
	}
}

// TestDiscardOrder_CeilingAndRerank verifies the least whole units: a
// shortfall of 5 mass with a size of 2 takes 3 units (ceil(5/2)), and the other
// dimension is re-ranked before the next commodity: after the first, only the
// bulk shortfall remains and the commodity with the greater bulk load goes.
//
// Requirement: FR-STO-024
func TestDiscardOrder_CeilingAndRerank(t *testing.T) {
	t.Parallel()
	cands := []room.Candidate{
		{Commodity: 1, Size: room.Pair{Mass: 2, Bulk: 0}, Lots: []int64{10}},
		{Commodity: 2, Size: room.Pair{Mass: 0, Bulk: 3}, Lots: []int64{10}},
	}
	steps, met := room.DiscardOrder(room.Pair{Mass: 5, Bulk: 4}, room.Pair{Mass: 100, Bulk: 100}, cands)
	// Both shortfalls: 5/100 > 4/100 -> mass first: commodity 1, ceil(5/2) = 3 units;
	// then bulk binds: commodity 2, ceil(4/3) = 2 units.
	want := []room.Step{{Cand: 0, Lot: 0, Units: 3}, {Cand: 1, Lot: 0, Units: 2}}
	if !met || !slices.Equal(steps, want) {
		t.Errorf("DiscardOrder = %v, %v, want %v, true", steps, met, want)
	}
}

// TestExcessOrder_LotLoadOverflowMeetsTheShortfall verifies a lot whose load
// does not fit int64 lowers every shortfall: nothing else goes.
//
// Requirement: FR-STO-018
func TestExcessOrder_LotLoadOverflowMeetsTheShortfall(t *testing.T) {
	t.Parallel()
	cands := []room.Candidate{
		{Commodity: 1, Size: room.Pair{Mass: 1 << 40}, Lots: []int64{1 << 40}},
		{Commodity: 2, Size: room.Pair{Mass: 1}, Lots: []int64{5}},
	}
	steps, met := room.ExcessOrder(room.Pair{Mass: 100}, room.Pair{Mass: 10}, cands)
	want := []room.Step{{Cand: 0, Lot: 0, Units: 1 << 40}}
	if !met || !slices.Equal(steps, want) {
		t.Errorf("ExcessOrder = %v, %v, want %v, true", steps, met, want)
	}
}
