package economysim

import (
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// The edges of the default strategy under a room: the dimension that is not the one
// the tests of storage_strategy_internal_test.go bind (bulk), and the arithmetic that
// does not fit int64, which every part of the strategy takes on the safe side (a view
// whose load cannot be added up plans no purchase and discards nothing, 6.2.2, 6.3).

// bulkView is roomView with the room in bulk: each unit has a bulk of 1 and no mass,
// and the mass room is never binding.
func bulkView(money, roomBulk, freeBulk int64) *AgentView {
	v := exampleView(money)
	v.Room = RoomState{
		Limited: true,
		Room:    Capacity{Mass: 1 << 40, Bulk: roomBulk},
		Free:    Capacity{Mass: 1 << 40, Bulk: freeBulk},
	}
	for i := range v.Commodities {
		v.Commodities[i].Bulk = 1
	}
	return v
}

// TestDefault_BulkBindsLikeMass verifies that the two dimensions of the room are read
// by the same rules: a view whose room, free room and unit sizes are in bulk and
// whose mass is unlimited is planned exactly as the view with the same figures in
// mass, for the scaling of the targets (rooms 14 and 19), the sizing of the bids
// (free room 3) and the discard for tier 1 (free room 0 with a surplus of 8 ore).
//
// Requirement: FR-STO-022, FR-STO-023, FR-STO-024
//
// Guard: the tests of the strategy bind mass alone; a root file that read Mass where
// it should read Bulk (or added the two) would pass all of them.
func TestDefault_BulkBindsLikeMass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		room, free int64
		surplusOre int64
	}{
		{"targets scaled to a room of 14", 14, 1 << 30, 0},
		{"targets held by a room of 19", 19, 1 << 30, 0},
		{"bids sized to a free room of 3", 1000, 3, 0},
		{"tier 1 makes room by a discard", 1000, 0, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, b := roomView(8000, tt.room, tt.free), bulkView(8000, tt.room, tt.free)
			m.Commodities[2].Stock, b.Commodities[2].Stock = tt.surplusOre, tt.surplusOre
			pm, nm := runTrade(m, exampleWords())
			pb, nb := runTrade(b, exampleWords())
			if !reflect.DeepEqual(pm.Intents, pb.Intents) || !reflect.DeepEqual(pm.Discards, pb.Discards) || !reflect.DeepEqual(pm.Cuts, pb.Cuts) {
				t.Errorf("Trade(bulk room %d, free %d) = %+v, want the plan of the same figures in mass %+v", tt.room, tt.free, pb, pm)
			}
			if nm != nb {
				t.Errorf("Trade(bulk room) drew %d words, want %d, those of the same figures in mass", nb, nm)
			}
		})
	}
}

// TestDefault_ReserveLoadTooLargeToAddUpTargetsNothing verifies the safe side of the
// arithmetic of 6.2.2: with every unit of mass 2^62, the reserve load of the fuel (1
// unit) and the staple (3 units) is 2^64, which does not fit int64, so the scaling is 0
// and every target is 0 (the plan holds nothing it could not size). What the strategy
// still does is tier 1: the fuel reserve is bid for, 1 unit of 2^62, and the free room
// of 2^40 holds none of it, so 1 unit of the staple above its keep' of 3 (stock 4) is
// discarded to make room for it; nothing else is bought and the staple is not asked.
//
// Requirement: FR-STO-022, FR-STO-023, FR-STO-024
func TestDefault_ReserveLoadTooLargeToAddUpTargetsNothing(t *testing.T) {
	t.Parallel()
	v := roomView(8000, 1<<62, 1<<40)
	for i := range v.Commodities {
		v.Commodities[i].Mass = 1 << 62
	}
	p, _ := runTrade(v, exampleWords())
	for _, in := range p.Intents {
		if in.Target != 0 {
			t.Errorf("intent %+v has a target, want 0 where the reserve load does not add up", in)
		}
	}
	if in, ok := intentFor(p, xFuel); !ok || in.Side != Bid || in.Quantity != 1 {
		t.Errorf("fuel intent = %+v (found %v), want the 1 unit of its tier 1 reserve", in, ok)
	}
	if got := bidLoad(p); got != 1 {
		t.Errorf("units bid = %d, want only the 1 unit of the fuel reserve", got)
	}
	if want := []Discard{{Commodity: xStaple, Quantity: 1}}; !reflect.DeepEqual(p.Discards, want) {
		t.Errorf("Discards = %+v, want %+v, the surplus unit that makes room for tier 1", p.Discards, want)
	}
	if _, ok := intentFor(p, xStaple); ok {
		t.Errorf("the staple has an intent, want none: its one surplus unit was discarded")
	}
}

// TestDefault_OverflowingNeedPlansNothing verifies that a view whose upkeep does not
// fit int64 (a weight of 2^62 and 4 units a household) plans nothing, limited or not:
// the scaling reads the need as unresolved and does not panic on it.
//
// Requirement: FR-STO-022
func TestDefault_OverflowingNeedPlansNothing(t *testing.T) {
	t.Parallel()
	v := roomView(8000, 1<<62, 1<<62)
	v.Weight = 1 << 62
	p, _ := runTrade(v, exampleWords())
	if len(p.Intents) != 0 || len(p.Discards) != 0 {
		t.Errorf("Trade(upkeep of 2^64 units) = %+v, want an empty plan", p)
	}
}

// TestMulOrMax verifies that the product saturates at the largest int64 where it does
// not fit, and is exact where it does.
//
// Requirement: FR-STO-022
func TestMulOrMax(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b int64
		want int64
	}{
		{"fits", 3, 5, 15},
		{"zero", 0, math.MaxInt64, 0},
		{"exactly the largest", 1, math.MaxInt64, math.MaxInt64},
		{"one over", 2, 1 << 62, math.MaxInt64},
		{"far over", math.MaxInt64, math.MaxInt64, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mulOrMax(tt.a, tt.b); got != tt.want {
				t.Errorf("mulOrMax(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// roomCtx is a trade of a limited agent with money 100 and the rows given: a free
// room of 0 and a room of 10 in each dimension.
func roomCtx(rows ...tradeRow) *tradeCtx {
	return &tradeCtx{v: &AgentView{Money: 100}, rows: rows, room: tradeRoom{on: true, room: room.Pair{Mass: 10, Bulk: 10}}}
}

// TestDiscardForRoom verifies 6.3.2 on rows built by hand: the tier 1 bid of 4 units of
// mass 1 and a free room of 0 is a shortfall of 4, met by the surplus of the other
// commodity (5 units) for 4 units, which are recorded and added to the free room; a
// load that does not fit int64 is no shortfall that can be named, so nothing is
// discarded; and a trade with no room does nothing.
//
// Requirement: FR-STO-024
func TestDiscardForRoom(t *testing.T) {
	t.Parallel()
	bid := func(size int64) tradeRow {
		return tradeRow{kind: bidOrder, limit: 1, t1: 4, commodity: 0, size: room.Pair{Mass: size}}
	}
	surplus := tradeRow{kind: askOrder, commodity: 1, surplus: 5, size: room.Pair{Mass: 1}}
	tests := []struct {
		name         string
		bidSize      int64
		off          bool
		wantDiscard  int64
		wantFreeMass int64
	}{
		{"a shortfall of 4 is made room for", 1, false, 4, 4},
		{"a load that overflows discards nothing", 1 << 62, false, 0, 0},
		{"no room discards nothing", 1, true, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := roomCtx(bid(tt.bidSize), surplus)
			c.room.on = !tt.off
			c.discardForRoom(&roomScratch{})
			if got := c.rows[1].discard; got != tt.wantDiscard {
				t.Errorf("discardForRoom() discarded %d units of the surplus, want %d", got, tt.wantDiscard)
			}
			if got := c.room.free.Mass; got != tt.wantFreeMass {
				t.Errorf("discardForRoom() left a free room of %d, want %d", got, tt.wantFreeMass)
			}
		})
	}
}

// TestRoomPlan_IsEmptyWithNoRoom verifies that a trade with no room adds nothing to the
// plan, however the rows read: no discard, no cut, and no discarded units counted.
//
// Requirement: FR-STO-023
func TestRoomPlan_IsEmptyWithNoRoom(t *testing.T) {
	t.Parallel()
	c := roomCtx(tradeRow{kind: bidOrder, commodity: 0, cut: 3, discard: 2})
	c.room.on = false
	var p TradePlan
	c.roomPlan(&p)
	if len(p.Discards) != 0 || len(p.Cuts) != 0 || p.roomDiscarded != 0 {
		t.Errorf("roomPlan() with no room = %+v, want it to add nothing", p)
	}
}

// TestRoomLimited_AnUnlimitedAgentIsNotCounted verifies that checkRoomPlan counts the
// cuts and discards of a limited agent and not of one with no room, so a plan that a
// consumer's strategy fills with cuts for an agent without a room leaves the counters
// at 0 and triggers no RoomLimited.
//
// Requirement: FR-STO-036
//
// Guard: the counters are summed into the cell of the agent; without the check an
// unlimited role's agent would raise the count of its cell for a limit that does not
// apply to it.
func TestRoomLimited_AnUnlimitedAgentIsNotCounted(t *testing.T) {
	t.Parallel()
	c := storageMillConfig()
	c.Modules.Storage.Base = nil
	w := newMill(t, c, WithSeed(1))
	a := &w.agents[1]
	if _, _, ok := w.sto.roomState(w.ledger, a); ok {
		t.Fatalf("the agent has a room with no base, want an unlimited one")
	}
	w.checkRoomPlan(a, &TradePlan{Cuts: []RoomCut{{Commodity: 0, Quantity: 1}}, roomDiscarded: 2})
	if got := w.sto.counts[w.cellOf(w.regionOf(a), a)]; got != (roomCount{}) {
		t.Errorf("counts after a plan with cuts for an unlimited agent = %+v, want none", got)
	}
}

// TestRoomTargets verifies the scaling of one commodity's targets (6.2.2): Target' and
// UseTgt' are the floors of Target and UseTgt times f over 10 000, the reserve is not
// scaled, and Keep' is Reserve + UseTgt' + Want (a negative want counts as 0). With f at
// 10 000, or no room, the targets come back as they were.
//
// Arithmetic: f = 6 666 on Target 9 and UseTgt 6 gives floor(9 * 6666 / 10000) = 5 and
// floor(6 * 6666 / 10000) = 3; Keep' = 3 + 3 + 2 = 8 with a want of 2.
//
// Requirement: FR-STO-022
func TestRoomTargets(t *testing.T) {
	t.Parallel()
	tg := household.Targets{Reserve: 3, UseTgt: 6, Target: 9, Keep: 11}
	tests := []struct {
		name string
		on   bool
		f    int64
		want int64
		out  household.Targets
	}{
		{"scaled with a want", true, 6666, 2, household.Targets{Reserve: 3, UseTgt: 3, Target: 5, Keep: 8}},
		{"scaled with a negative want", true, 6666, -4, household.Targets{Reserve: 3, UseTgt: 3, Target: 5, Keep: 6}},
		{"scaled to nothing keeps the reserve", true, 0, 0, household.Targets{Reserve: 3, UseTgt: 0, Target: 0, Keep: 3}},
		{"full scale is unchanged", true, room.Denominator, 2, tg},
		{"no room is unchanged", false, 0, 2, tg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &tradeCtx{room: tradeRoom{on: tt.on, f: tt.f}}
			if got := c.roomTargets(tg, tt.want); got != tt.out {
				t.Errorf("roomTargets(%+v, want %d) at f = %d = %+v, want %+v", tg, tt.want, tt.f, got, tt.out)
			}
		})
	}
}

// TestRoomBound verifies the cut of a bid to the free room (6.3.1): the units that fit
// are the free room divided by the size of a unit in each dimension, the smaller wins,
// a dimension in which the unit has no size does not bind, a negative free room is read
// as 0, the units cut are recorded on the row, and a trade with no room cuts nothing.
//
// Arithmetic: a unit of (mass 2, bulk 3) in a free room of (9, 6) fits min(9/2, 6/3) = 2
// units, so a want of 5 is cut by 3.
//
// Requirement: FR-STO-023
func TestRoomBound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		on       bool
		size     room.Pair
		free     room.Pair
		want     int64
		fit, cut int64
	}{
		{"bulk binds", true, room.Pair{Mass: 2, Bulk: 3}, room.Pair{Mass: 9, Bulk: 6}, 5, 2, 3},
		{"mass binds", true, room.Pair{Mass: 2, Bulk: 3}, room.Pair{Mass: 3, Bulk: 60}, 5, 1, 4},
		{"no bulk in the unit", true, room.Pair{Mass: 2}, room.Pair{Mass: 9, Bulk: 0}, 5, 4, 1},
		{"no mass in the unit", true, room.Pair{Bulk: 2}, room.Pair{Mass: 0, Bulk: 9}, 5, 4, 1},
		{"a negative free room fits nothing", true, room.Pair{Mass: 1}, room.Pair{Mass: -5, Bulk: 9}, 3, 0, 3},
		{"everything fits", true, room.Pair{Mass: 1, Bulk: 1}, room.Pair{Mass: 9, Bulk: 9}, 4, 4, 0},
		{"a want below 1 is not cut", true, room.Pair{Mass: 1}, room.Pair{}, 0, 0, 0},
		{"no room cuts nothing", false, room.Pair{Mass: 1}, room.Pair{}, 4, 4, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &tradeCtx{room: tradeRoom{on: tt.on, free: tt.free}}
			r := &tradeRow{size: tt.size}
			if got := c.roomBound(r, tt.want); got != tt.fit || r.cut != tt.cut {
				t.Errorf("roomBound(size %+v, free %+v, want %d) = %d with %d cut, want %d with %d cut", tt.size, tt.free, tt.want, got, r.cut, tt.fit, tt.cut)
			}
		})
	}
}
