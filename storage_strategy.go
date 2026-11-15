package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// This file is the part of DefaultStrategy that reads the room (spec-storage.md 6.2,
// 6.3, FR-STO-022 to 025): the scaling of the targets, the sizing of the bids in the
// order of the money stack and the discard that makes room for tier 1. Every method
// is a no-op for an agent that is not limited, which costs one branch. It reads the
// view alone, takes no random word and logs nothing: the counts and the Debug lines
// of what it cut are the world's, from TradePlan.Cuts (checkRoomPlan).

// RoomCut is the units of a bid that the default strategy took off for room: a bid
// sized down, or dropped when the room left held none (spec-storage.md 6.3.1).
type RoomCut struct {
	// Commodity is the commodity of the bid.
	Commodity CommodityID
	// Quantity is the units cut.
	Quantity Quantity
}

// roomScratch is the working memory of the room part of Trade, grown once.
type roomScratch struct {
	items []room.Item
	cands []room.Candidate
	lots  []int64
	rows  []int
}

// tradeRoom is the room of a limited agent's Trade: its free room as the bids are
// sized (it falls with each unit bought and rises with each unit discarded) and the
// scaling f of the targets, in basis points (spec-storage.md 6.2.2).
type tradeRoom struct {
	on         bool
	free, room room.Pair
	f          int64
}

// scaleToRoom sets t.room for a limited agent and the scaling f of its targets
// (spec-storage.md 6.2.2): per dimension the room less the pledge and the upkeep
// eaten this tick is what the targets of every commodity, the job's inputs included,
// may take, pro rata. An agent with no room, or whose load does not overflow, has
// f = 10 000 and nothing scaled.
func (t *tradeCtx) scaleToRoom(sc *roomScratch) {
	v := t.v
	if !v.Room.Limited {
		return
	}
	tr := &t.room
	*tr = tradeRoom{on: true, free: pairOf(v.Room.Free), room: pairOf(v.Room.Room), f: room.Denominator}
	sc.items = sc.items[:0]
	var res room.Pair
	for i := range v.Commodities {
		c := &v.Commodities[i]
		u, _, ok1 := t.upkeepUnits(c.Commodity)
		job, out, ok2 := t.jobAndOutput(c.Commodity)
		tg, ok3 := household.ComputeTargets(household.Need{Upkeep: u, Job: job, OutTgt: out, Want: t.wantOf(c.Commodity)}, t.par)
		sz := room.Pair{Mass: c.Mass, Bulk: c.Bulk}
		if !ok1 || !ok2 || !ok3 {
			tr.f = 0 // a view whose targets overflow targets nothing, the safe side
			return
		}
		sc.items = append(sc.items, room.Item{Qty: tg.Target, Size: sz})
		var okm, okb bool
		res.Mass, okm = checked.Add(res.Mass, mulOrMax(tg.Reserve, sz.Mass))
		res.Bulk, okb = checked.Add(res.Bulk, mulOrMax(tg.Reserve, sz.Bulk))
		if !okm || !okb {
			tr.f = 0
			return
		}
	}
	tr.f = room.Scale(sc.items, tr.room, pairOf(v.Room.Pledge), res)
}

// mulOrMax is a*b saturated at the largest int64, for a sum the caller checks.
func mulOrMax(a, b int64) int64 {
	if p, ok := checked.Mul(a, b); ok {
		return p
	}
	return 1<<63 - 1
}

// pairOf is a Capacity as the pair of the room rules.
func pairOf(c Capacity) room.Pair { return room.Pair{Mass: c.Mass, Bulk: c.Bulk} }

// roomTargets scales the targets of a commodity by f: Target' and UseTgt' are the
// floors of Target and UseTgt times f over 10 000, and Keep' is Reserve + UseTgt' +
// Want (spec-storage.md 6.2.2). With f at its full value they are unchanged.
func (t *tradeCtx) roomTargets(tg household.Targets, want int64) household.Targets {
	if !t.room.on || t.room.f >= room.Denominator {
		return tg
	}
	use := room.ScaleQty(tg.UseTgt, t.room.f)
	return household.Targets{Reserve: tg.Reserve, UseTgt: use, Target: room.ScaleQty(tg.Target, t.room.f), Keep: tg.Reserve + use + max(want, 0)}
}

// roomRow records on the row of a commodity the size of its unit and its surplus
// above Keep', which the sizing and the discard read.
func (t *tradeCtx) roomRow(r *tradeRow, c *CommodityView, surplus int64) {
	if !t.room.on {
		return
	}
	r.commodity, r.size, r.surplus = c.Commodity, room.Pair{Mass: c.Mass, Bulk: c.Bulk}, surplus
}

// roomBound cuts want to the units whose load fits the free room that the units
// already allocated left, and records the units cut on the row (spec-storage.md 6.3.1).
func (t *tradeCtx) roomBound(r *tradeRow, want int64) int64 {
	if !t.room.on || want < 1 {
		return want
	}
	fit := want
	if r.size.Mass > 0 {
		fit = min(fit, max(t.room.free.Mass, 0)/r.size.Mass)
	}
	if r.size.Bulk > 0 {
		fit = min(fit, max(t.room.free.Bulk, 0)/r.size.Bulk)
	}
	r.cut += want - fit
	return fit
}

// roomTake takes the load of q units off the free room. q was bounded by roomBound,
// so the product is at most the free room and does not overflow.
func (t *tradeCtx) roomTake(r *tradeRow, q int64) {
	if t.room.on {
		t.room.free.Mass -= q * r.size.Mass
		t.room.free.Bulk -= q * r.size.Bulk
	}
}

// buy spends avail on up to want units of one tier of the order, as combineTiers
// does, bounded by the free room (the money stack's order is the room's, 6.3.1), and
// returns the money left.
func (t *tradeCtx) buy(r *tradeRow, avail, want int64) int64 {
	if !t.room.on {
		return r.combineTiers(avail, want)
	}
	was := r.qty
	avail = r.combineTiers(avail, t.roomBound(r, want))
	t.roomTake(r, r.qty-was)
	return avail
}

// tierOneLoad is the load of the tier 1 quantities the money buys, per dimension,
// as the first pass of the stack would (the first tier of an order to buy may lower
// its limit to the money). ok is false when the load overflows.
func (t *tradeCtx) tierOneLoad(avail int64) (load room.Pair, ok bool) {
	for i := range t.rows {
		r := &t.rows[i]
		if r.kind != bidOrder || r.limit < 1 {
			continue
		}
		limit, q := household.Afford(avail, r.limit, r.t1)
		if q < 1 {
			continue
		}
		avail -= limit * q
		var okm, okb bool
		load.Mass, okm = addMul(load.Mass, q, r.size.Mass)
		load.Bulk, okb = addMul(load.Bulk, q, r.size.Bulk)
		if !okm || !okb {
			return room.Pair{}, false
		}
	}
	return load, true
}

// addMul is sum + a*b, checked.
func addMul(sum, a, b int64) (int64, bool) {
	p, ok := checked.Mul(a, b)
	if !ok {
		return 0, false
	}
	return checked.Add(sum, p)
}

// discardForRoom is the discard that makes room to eat (spec-storage.md 6.3.2,
// FR-STO-024): the load of the tier 1 bids above the free room is the shortfall, and
// the surplus above Keep' goes by the rule of the discard order until the tier 1 bids
// fit. It discards nothing for tier 2 or beyond, and nothing out of Keep'. The units
// are recorded on the rows; emit appends the Discards and the asks are reduced by
// them. The free room grows by what was discarded.
func (t *tradeCtx) discardForRoom(sc *roomScratch) {
	tr := &t.room
	if !tr.on {
		return
	}
	load, ok := t.tierOneLoad(max(t.v.Money, 0))
	if !ok {
		return
	}
	short := room.Pair{Mass: max(load.Mass-tr.free.Mass, 0), Bulk: max(load.Bulk-tr.free.Bulk, 0)}
	if short.Mass == 0 && short.Bulk == 0 {
		return
	}
	sc.cands, sc.lots, sc.rows = sc.cands[:0], sc.lots[:0], sc.rows[:0]
	for i := range t.rows {
		if r := &t.rows[i]; r.surplus > 0 {
			sc.rows = append(sc.rows, i)
			sc.lots = append(sc.lots, r.surplus)
		}
	}
	for k, i := range sc.rows {
		sc.cands = append(sc.cands, room.Candidate{Commodity: int(t.rows[i].commodity), Size: t.rows[i].size, Lots: sc.lots[k : k+1]})
	}
	steps, _ := room.DiscardOrder(short, tr.room, sc.cands)
	for _, st := range steps {
		r := &t.rows[sc.rows[st.Cand]]
		r.discard += st.Units
		tr.free.Mass += st.Units * r.size.Mass
		tr.free.Bulk += st.Units * r.size.Bulk
	}
}

// roomPlan appends what the room did to the plan: the discards, in commodity order,
// and the bids cut, for the world to execute, count and log (checkRoomPlan).
func (t *tradeCtx) roomPlan(p *TradePlan) {
	if !t.room.on {
		return
	}
	for i := range t.rows {
		r := &t.rows[i]
		if r.discard > 0 {
			p.Discards = append(p.Discards, Discard{Commodity: r.commodity, Quantity: Quantity(r.discard)})
		}
		if r.cut > 0 && r.kind == bidOrder {
			p.Cuts = append(p.Cuts, RoomCut{Commodity: r.commodity, Quantity: Quantity(min(r.cut, MaxQuantity))})
		}
	}
}
