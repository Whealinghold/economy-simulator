package economysim

import (
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/room"
)

// The default strategy under a room is tested on the view of exampleView
// (spec-agents.md 16.2) with money 8 000, whose baseline plan is fuel 2, staple 4 and
// ore 2 bid and 1 item asked, targets fuel 3, staple 9, ore 2 and item 1, reserve fuel
// 1 and staple 3, and stock staple 4 and item 1. Each commodity is given a unit mass of
// 1 and a bulk of 0, so a load is a count of units; the bulk room is never binding.

// roomView is exampleView with a room: the room and the free room in mass, a pledge of
// 0 and the sizes set.
func roomView(money, roomMass, freeMass int64) *AgentView {
	v := exampleView(money)
	v.Room = RoomState{
		Limited: true,
		Room:    Capacity{Mass: roomMass, Bulk: 1 << 40},
		Free:    Capacity{Mass: freeMass, Bulk: 1 << 40},
	}
	for i := range v.Commodities {
		v.Commodities[i].Mass = 1
	}
	return v
}

// intentFor returns the intent of the commodity in the plan, if any.
func intentFor(p TradePlan, c CommodityID) (Intent, bool) {
	for _, in := range p.Intents {
		if in.Commodity == c {
			return in, true
		}
	}
	return Intent{}, false
}

// bidLoad is the units bid in the plan, a load of 1 each.
func bidLoad(p TradePlan) (n int64) {
	for _, in := range p.Intents {
		if in.Side == Bid {
			n += int64(in.Quantity)
		}
	}
	return n
}

// TestDefault_ScalesTargetsToRoom verifies the scaling of 6.2.2 on the example: the
// targets are 3, 9, 2 and 1, a load of 15, and the upkeep eaten this tick is 4 (fuel 1,
// staple 3). A room of 14 with nothing pledged leaves Avail = 10, so f = floor(10 * 10 000
// / 15) = 6 666 and the targets are floor(3 * 0.6666) = 1, 5, 1 and 0; a room that holds
// them all (19, Avail = 15) leaves them as they are.
//
// Requirement: FR-STO-022
func TestDefault_ScalesTargetsToRoom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		roomMass   int64
		wantTarget map[CommodityID]Quantity
	}{
		{"binding", 14, map[CommodityID]Quantity{xFuel: 1, xStaple: 5, xOre: 1}},
		{"room holds the targets", 19, map[CommodityID]Quantity{xFuel: 3, xStaple: 9, xOre: 2}},
		{"nothing left after the upkeep", 4, map[CommodityID]Quantity{xFuel: 0, xStaple: 0, xOre: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := roomView(8000, tt.roomMass, 1<<30)
			p, _ := runTrade(v, exampleWords())
			for c, want := range tt.wantTarget {
				if in, ok := intentFor(p, c); ok && in.Target != want {
					t.Errorf("Target of commodity %d = %d, want %d", c, in.Target, want)
				}
			}
		})
	}
	t.Run("f from the pure rule", func(t *testing.T) {
		t.Parallel()
		got := room.Scale([]room.Item{{Qty: 3, Size: room.Pair{Mass: 1}}, {Qty: 9, Size: room.Pair{Mass: 1}}, {Qty: 2, Size: room.Pair{Mass: 1}}, {Qty: 1, Size: room.Pair{Mass: 1}}},
			room.Pair{Mass: 14, Bulk: 1 << 40}, room.Pair{}, room.Pair{Mass: 4})
		if got != 6666 {
			t.Errorf("Scale = %d, want 6666", got)
		}
	})
}

// TestDefault_BidsSizedToFreeRoom_NoRejectedBid verifies the sizing of 6.3.1: with a
// room that holds the targets but a free room of 3, the bids are cut in the order of
// the money stack, tier 1 first (fuel: the reserve of 1), and no plan asks the market
// for more than the free room, so none is rejected for room; the cuts are reported.
//
// Requirement: FR-STO-023
func TestDefault_BidsSizedToFreeRoom_NoRejectedBid(t *testing.T) {
	t.Parallel()
	v := roomView(8000, 1000, 3)
	p, _ := runTrade(v, exampleWords())
	if got := bidLoad(p); got != 3 {
		t.Errorf("units bid = %d, want 3, the free room", got)
	}
	if in, ok := intentFor(p, xFuel); !ok || in.Quantity < 1 {
		t.Errorf("fuel bid = %+v (found %v), want at least its tier 1 unit first", in, ok)
	}
	if len(p.Cuts) == 0 {
		t.Errorf("Cuts = %v, want the bids cut for room reported", p.Cuts)
	}
	free := roomView(8000, 1000, 1<<30)
	q, _ := runTrade(free, exampleWords())
	if len(q.Cuts) != 0 {
		t.Errorf("Cuts with room to spare = %v, want none", q.Cuts)
	}
}

// TestDefault_DrawsEveryIntent verifies 6.3.1: a bid cut to nothing is not placed but
// its draw is taken, so the stream position is the same with a free room of 0 and with
// room to spare.
//
// Requirement: FR-STO-023
func TestDefault_DrawsEveryIntent(t *testing.T) {
	t.Parallel()
	_, base := runTrade(roomView(8000, 1000, 1<<30), exampleWords())
	p, n := runTrade(roomView(8000, 1000, 0), exampleWords())
	if n != base {
		t.Errorf("draws with a free room of 0 = %d, want %d, those with room to spare", n, base)
	}
	// The one unit bid is the fuel reserve of tier 1, made room by discarding the
	// item (a surplus above its keep): nothing else fits.
	if got := bidLoad(p); got != 1 || len(p.Discards) != 1 {
		t.Errorf("units bid with no free room = %d after discards %+v, want the 1 tier 1 unit and 1 unit discarded", got, p.Discards)
	}
	if len(p.Cuts) == 0 {
		t.Errorf("Cuts = %v, want every dropped bid reported", p.Cuts)
	}
}

// TestDefault_DiscardsOnlyForTierOne verifies 6.3.2: the household holds 10 ore (a
// surplus of 8 above its keep of 2) in a room with no free mass, and its tier 1 bid
// (the fuel reserve, 1 unit) needs room, so 1 unit of ore is discarded and the ask for
// the ore falls by it; when only a tier 2 bid lacks room (the fuel is held, the ore
// input is needed, the surplus is of another commodity) nothing is discarded.
//
// Requirement: FR-STO-024
func TestDefault_DiscardsOnlyForTierOne(t *testing.T) {
	t.Parallel()
	t.Run("tier 1 makes room", func(t *testing.T) {
		t.Parallel()
		v := roomView(8000, 1000, 0)
		v.Commodities[2].Stock = 10
		p, _ := runTrade(v, exampleWords())
		if len(p.Discards) != 1 || p.Discards[0] != (Discard{Commodity: xOre, Quantity: 1}) {
			t.Errorf("Discards = %+v, want 1 unit of ore", p.Discards)
		}
		if in, ok := intentFor(p, xFuel); !ok || in.Quantity != 1 {
			t.Errorf("fuel bid = %+v (found %v), want the tier 1 unit", in, ok)
		}
		if in, ok := intentFor(p, xOre); !ok || in.Side != Ask || in.Quantity != 7 {
			t.Errorf("ore intent = %+v (found %v), want an ask of 7: 8 of surplus less the 1 discarded", in, ok)
		}
	})
	t.Run("tier 2 does not", func(t *testing.T) {
		t.Parallel()
		v := roomView(8000, 1000, 0)
		v.Commodities[0].Stock = 1 // the fuel reserve is held: no tier 1
		v.Commodities[3].Stock = 6 // a surplus of the output
		p, _ := runTrade(v, exampleWords())
		if len(p.Discards) != 0 {
			t.Errorf("Discards = %+v, want none: the ore input is tier 2", p.Discards)
		}
	})
}

// TestDefault_TierOneAlwaysFits verifies 6.3.3 and FR-STO-025 as a property: for rooms
// that hold one tick's upkeep (4 units), holdings that fit the room and money to pay,
// the tier 1 bids (the units needed to reach the reserve) are placed whole after the
// discard, and the plan never asks for more than the free room and the discard make.
//
// Requirement: FR-STO-025
func TestDefault_TierOneAlwaysFits(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(7, 11))
	for range 500 {
		v := randomRoomView(rng)
		p, _ := runTrade(v, exampleWords())
		var discarded int64
		for _, d := range p.Discards {
			discarded += int64(d.Quantity)
		}
		if got, limit := bidLoad(p), v.Room.Free.Mass+discarded; got > limit {
			t.Fatalf("room %d stock %v: bid %d units, want at most the free room %d plus the %d discarded", v.Room.Room.Mass, stocks(v), got, v.Room.Free.Mass, discarded)
		}
		for c, reserve := range map[CommodityID]int64{xFuel: 1, xStaple: 3} {
			need := max(0, reserve-v.Commodities[c].Stock)
			in, ok := intentFor(p, c)
			if need > 0 && (!ok || int64(in.Quantity) < need) {
				t.Fatalf("room %d stock %v: commodity %d bid %+v (found %v), want its tier 1 %d units", v.Room.Room.Mass, stocks(v), c, in, ok, need)
			}
		}
	}
}

// randomRoomView is a view with a room of 4 to 43 units and random holdings that fit it.
func randomRoomView(rng *rand.Rand) *AgentView {
	roomMass := 4 + rng.Int64N(40)
	v := roomView(1<<40, roomMass, 0)
	var load int64
	for i := range v.Commodities {
		s := min(rng.Int64N(12), roomMass-load)
		v.Commodities[i].Stock, load = s, load+s
	}
	v.Room.Free.Mass = roomMass - load
	return v
}

// stocks lists the stocks of a view for a failure message.
func stocks(v *AgentView) []int64 {
	out := make([]int64, len(v.Commodities))
	for i, c := range v.Commodities {
		out[i] = c.Stock
	}
	return out
}

// TestDefault_UnlimitedViewIsUnchanged verifies that an agent with no room is planned
// as before: the plan equals the worked example 16.2 with no Cuts or Discards.
//
// Requirement: FR-STO-001
func TestDefault_UnlimitedViewIsUnchanged(t *testing.T) {
	t.Parallel()
	p, n := runTrade(exampleView(8000), exampleWords())
	if len(p.Cuts) != 0 || len(p.Discards) != 0 || n != 4 || len(p.Intents) != 4 {
		t.Errorf("plan = %+v after %d draws, want 4 intents, no cuts or discards, 4 draws", p, n)
	}
}

// TestView_RoomGroup verifies the room group of the view of a limited agent (6.1.1): the
// room, load, pledge, free room, stores by class and the size of a unit of every
// commodity of the tradable set; an agent of an unlimited role has none.
//
// Requirement: FR-STO-021
func TestView_RoomGroup(t *testing.T) {
	t.Parallel()
	w, _ := stoWorld(t, nil)
	stoAdd(t, w, 3, 0)
	stoGrain(t, w, 3, 10)
	w.refreshViewCtx()
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[3], viewTrade); err != nil {
		t.Fatalf("fillAgentView() = %v, want nil", err)
	}
	st := roomOfAgent(t, w, 3)
	if !v.Room.Limited || v.Room.Room != st.Room || v.Room.Load != st.Load || v.Room.Pledge != st.Pledge || v.Room.Free != st.Free {
		t.Errorf("view room group = %+v, want the agent query's %+v", v.Room, st)
	}
	if len(v.Room.Stores) != 1 || v.Room.Stores[0] != (StoreCount{Class: 0, Count: 1}) {
		t.Errorf("view Stores = %+v, want one barn", v.Room.Stores)
	}
	checkViewSizes(t, w, &v)
}

// checkViewSizes verifies the size of a unit of each commodity of the view.
func checkViewSizes(t *testing.T, w *World, v *AgentView) {
	t.Helper()
	for _, c := range v.Commodities {
		if sz := w.sto.sizes[c.Commodity]; c.Mass != sz.Mass || c.Bulk != sz.Bulk {
			t.Errorf("size of commodity %d = (%d, %d), want (%d, %d)", c.Commodity, c.Mass, c.Bulk, sz.Mass, sz.Bulk)
		}
	}
}

// TestDraws_CountUnchangedWhileNothingActs verifies FR-STO-037: while no scaling, cut or
// discard acts, a limited agent's plan is the plan of an agent with no room and takes the
// same draws.
//
// Requirement: FR-STO-037
func TestDraws_CountUnchangedWhileNothingActs(t *testing.T) {
	t.Parallel()
	want, nWant := runTrade(exampleView(8000), exampleWords())
	got, n := runTrade(roomView(8000, 1000, 1<<30), exampleWords())
	if n != nWant || !reflect.DeepEqual(got.Intents, want.Intents) {
		t.Errorf("limited plan %+v after %d draws, want %+v after %d, those of an agent with no room", got.Intents, n, want.Intents, nWant)
	}
}
