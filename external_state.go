package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/outside"
)

// This file is the continuation of external.go (architecture.md 2.6, EXT): what a
// consumer and a checkpoint see of the module's state, the snapshot rows
// (spec-external-sector.md 15.4, spec-observability.md 5.3.9), the internal shock
// setter (6.5), tag 18 (16) and the pure read of the lines (12.1, FR-EXT-047).

// OutsideState is the state of the external module in a snapshot, with slices in the
// order of the configuration (spec-observability.md 5.3.9); the zero value while the
// module is off. There is no per-node money balance: the world's is Boundary.
type OutsideState struct {
	// Lines has one row per (node, commodity); Sides one per (node, gate, commodity,
	// side) with buy before sell; Trade one per commodity with a line; Counterparties
	// one per pseudo-agent, by agent ID.
	Lines          []OutsideLineRow
	Sides          []OutsideSideRow
	Trade          []OutsideTradeRow
	Counterparties []CounterpartyRow
}

// OutsideLineRow is the state of one line: its mid now and the state of its walk and
// shock.
type OutsideLineRow struct {
	Node      RegionID
	Commodity CommodityID
	Mid       Price
	// DriftBP is the walk's drift, 0 without a walk; ShockBP the shock slot, 10 000
	// until set.
	DriftBP, ShockBP int64
}

// OutsideSideRow is the state of one side at one gate: its quote now, the carry of
// its wanted units, the cumulative units and list value, and whether it ended.
type OutsideSideRow struct {
	Node, Gate RegionID
	Commodity  CommodityID
	// Side is Bid for a buy side and Ask for a sell side.
	Side      Side
	Quote     Price
	Carry     int64
	Units     int64
	ListValue Money
	Ended     bool
}

// OutsideTradeRow is the cumulative trade of one commodity over all nodes: the units
// of the sell sides (imports) and of the buy sides (exports).
type OutsideTradeRow struct {
	Commodity        CommodityID
	Imports, Exports Quantity
}

// CounterpartyRow is a counterparty pseudo-agent: its agent ID, node, gate, side and
// ledger account.
type CounterpartyRow struct {
	Agent      AgentID
	Node, Gate RegionID
	Side       Side
	Account    int64
}

// snapshotOutside fills the outside state of a snapshot from the module's state
// and the mid of the tick about to run. Every slice is new (a deep copy).
func (w *World) snapshotOutside(s *Snapshot) {
	x := w.external
	if x == nil {
		return
	}
	o := &s.Outside
	o.Lines = make([]OutsideLineRow, len(x.lines))
	for i := range x.lines {
		l := &x.lines[i]
		mid, _ := x.midOf(w, l)
		o.Lines[i] = OutsideLineRow{Node: l.node, Commodity: l.commodity, Mid: Price(mid), DriftBP: l.drift, ShockBP: l.shock}
	}
	o.Sides = make([]OutsideSideRow, 0, len(x.rows))
	trade := map[CommodityID]*OutsideTradeRow{}
	x.eachRowOrdered(func(cp *counterparty, row *sideRow) {
		l := &x.lines[row.line]
		side := &l.cfg.Sell
		if cp.buy {
			side = &l.cfg.Buy
		}
		mid, _ := x.midOf(w, l)
		quote, _ := outside.Quote(mid, side.MarginBP, cp.scaleBP, MaxPrice, !cp.buy)
		o.Sides = append(o.Sides, OutsideSideRow{Node: cp.node, Gate: cp.gate, Commodity: l.commodity, Side: cp.sideOf(),
			Quote: Price(quote), Carry: row.carry, Units: row.units, ListValue: Money(row.listValue), Ended: row.ended})
		t := trade[l.commodity]
		if t == nil {
			t = &OutsideTradeRow{Commodity: l.commodity}
			trade[l.commodity] = t
		}
		if cp.buy {
			t.Exports = Quantity(satAdd(int64(t.Exports), row.units))
		} else {
			t.Imports = Quantity(satAdd(int64(t.Imports), row.units))
		}
	})
	for _, t := range trade {
		o.Trade = append(o.Trade, *t)
	}
	slices.SortFunc(o.Trade, func(a, b OutsideTradeRow) int { return int(a.Commodity) - int(b.Commodity) })
	o.Counterparties = make([]CounterpartyRow, len(x.cps))
	for i := range x.cps {
		cp := &x.cps[i]
		o.Counterparties[i] = CounterpartyRow{Agent: cp.agent, Node: cp.node, Gate: cp.gate, Side: cp.sideOf(), Account: int64(cp.account)}
	}
}

// outsideShock is one change of the shock slot, in the internal form that the
// public SetOutsideShocks (plan 17) builds from its validated input.
type outsideShock struct {
	node      RegionID
	commodity CommodityID
	bp        int64
}

// applyShocks sets the shock slot of each (node, commodity) to bp, applying from the
// next step, and returns an OutsideShocked for each slot whose value changed, in the
// order given; a value that is unchanged changes nothing and has no event
// (spec-external-sector.md 6.5, D535). The input is assumed validated: a key with no
// line is a programmer error. The caller delivers the events.
func (x *external) applyShocks(tick int64, changes []outsideShock) []OutsideShocked {
	var events []OutsideShocked
	for _, c := range changes {
		i := slices.IndexFunc(x.lines, func(l outsideLine) bool { return l.node == c.node && l.commodity == c.commodity })
		if i < 0 {
			invariant("a shock names a line the node does not have, which validation rules out")
		}
		l := &x.lines[i]
		if l.shock == c.bp {
			continue
		}
		events = append(events, OutsideShocked{Tick: tick, Node: c.node, Commodity: c.commodity, From: l.shock, To: c.bp})
		l.shock = c.bp
	}
	return events
}

// saveExternal is tag 18 (spec-checkpoint.md 4.2, spec-external-sector.md 16): for
// each line in (node, commodity) order the drift when it walks and the shock, then
// for each side row in (node, gate, commodity, buy before sell) order the carry, the
// cumulative units, the cumulative list value and the ended flag. The mid, the
// quotes, the gates and the counterparties are derived; the accounts are empty.
func (w *World) saveExternal(cw *codec.Writer) error {
	x := w.external
	for i := range x.lines {
		l := &x.lines[i]
		if l.cfg.WalkStepBP > 0 {
			cw.I(l.drift)
		}
		cw.U(unsigned(l.shock))
	}
	x.eachRowOrdered(func(_ *counterparty, row *sideRow) {
		cw.U(unsigned(row.carry))
		cw.U(unsigned(row.units))
		cw.U(unsigned(row.listValue))
		cw.B(row.ended)
	})
	return nil
}

// loadExternal reads tag 18 into the lines and rows that the configuration built:
// a drift within its walk bound, a shock in 1 to MaxShockBP, a carry below 10 000
// and units and list value within the caps of their side.
func (w *World) loadExternal(_ *restoreCtx, r *codec.Reader) error {
	x := w.external
	for i := range x.lines {
		l := &x.lines[i]
		if l.cfg.WalkStepBP > 0 {
			l.drift = r.I()
			if l.drift < -l.cfg.WalkBoundBP || l.drift > l.cfg.WalkBoundBP {
				r.Failf("drift %d of line %d is outside the walk bound %d", l.drift, i, l.cfg.WalkBoundBP)
			}
		}
		l.shock = r.Bounded("shock", MaxShockBP)
		if r.Err() == nil && l.shock < 1 {
			r.Failf("shock of line %d is 0, want 1 to %d", i, MaxShockBP)
		}
	}
	x.eachRowOrdered(func(cp *counterparty, row *sideRow) {
		if r.Err() != nil {
			return
		}
		side := &x.lines[row.line].cfg.Sell
		if cp.buy {
			side = &x.lines[row.line].cfg.Buy
		}
		row.carry = r.Bounded("carry", wholeBP-1)
		units := int64(MaxLineTotal)
		if side.TotalUnits > 0 {
			units = min(units, side.TotalUnits)
		}
		value := int64(MaxLineTotal)
		if side.TotalValue > 0 {
			value = min(value, side.TotalValue)
		}
		row.units = r.Bounded("units", units)
		row.listValue = r.Bounded("list value", value)
		row.ended = r.B()
		if r.Err() == nil && !row.ended && side.TotalUnits > 0 && units == side.TotalUnits && row.units >= side.TotalUnits {
			r.Failf("a side that has traded all its %d units has not ended", side.TotalUnits)
		}
		if r.Err() == nil && row.ended && side.TotalUnits == 0 && side.TotalValue == 0 {
			r.Failf("a side with no cap has ended")
		}
	})
	return nil
}

// lineRead is one line of the configuration as the pure read of FR-EXT-047 gives it,
// for the open layer of the solver: the quotes at tick 0 and the mean capacity per
// tick of each side present, with the gates the side serves.
type lineRead struct {
	Node, Commodity string
	// Sell and Buy are the sides present; their gates are in ascending order.
	Sell, Buy sideRead
}

// sideRead is one side of a lineRead: whether it is present, its mean capacity in
// units a tick over the seasons (weighted by days) and, per gate, the quote at tick 0.
type sideRead struct {
	Present      bool
	MeanCapacity int64
	Gates        []gateRead
}

// gateRead is a gate of a side with the quote at tick 0 with the gate's scale.
type gateRead struct {
	Gate  string
	Quote int64
}

// externalLines is the pure read of the configured lines: no world, no state, in
// the order of the configuration (node, then commodity). A cell is zero when the
// side is absent. The quote is of the mid at tick 0 (the base times the season of
// tick 0, which is the first season's factor).
func externalLines(c *Config) []lineRead {
	m := c.Modules.External
	if m == nil {
		return nil
	}
	m = m.canonical(c.Calendar.Seasons)
	var out []lineRead
	for ni := range m.Nodes {
		n := &m.Nodes[ni]
		sell, buy := edgeGates(c, n.Region)
		for li := range n.Lines {
			l := &n.Lines[li]
			out = append(out, lineRead{Node: n.Region, Commodity: l.Commodity,
				Sell: readSide(c, n, l, &l.Sell, sell, true), Buy: readSide(c, n, l, &l.Buy, buy, false)})
		}
	}
	return out
}

// readSide reads one side of a line over its gates.
func readSide(c *Config, n *OutsideNode, l *OutsideLine, s *OutsideSide, gates []string, sell bool) sideRead {
	if !s.present() {
		return sideRead{}
	}
	var first string // the season of tick 0: StartDay and SeasonStart decide it, not the list order
	if m, err := c.Calendar.At(0); err == nil && m.Season >= 0 && m.Season < len(c.Calendar.Seasons) {
		first = c.Calendar.Seasons[m.Season].ID
	}
	mid, _, ok := outside.Mid(outside.MidIn{Base: int64(l.Base), Season: seasonBP(l.PriceBySeason, first, wholeBP),
		FloorBP: l.FloorBP, CeilBP: l.CeilingBP, MaxPrice: MaxPrice, Tick: 0, TicksPerYear: c.Calendar.TicksPerYear(), Shock: wholeBP})
	if !ok {
		invariant("the mid at tick 0 of a validated line is out of its domain")
	}
	r := sideRead{Present: true, MeanCapacity: meanCapacity(s, c.Calendar.Seasons)}
	for _, g := range gates {
		scale := int64(DefaultGateScaleBP)
		if i := slices.IndexFunc(n.Gates, func(o OutsideGate) bool { return o.Region == g }); i >= 0 {
			scale = n.Gates[i].ScaleBP
		}
		q, _ := outside.Quote(mid, s.MarginBP, scale, MaxPrice, sell)
		r.Gates = append(r.Gates, gateRead{Gate: g, Quote: q})
	}
	return r
}

// meanCapacity is the capacity of a side times its window averaged over the seasons
// by their days, truncated; the whole capacity when the calendar has no season.
func meanCapacity(s *OutsideSide, seasons []Season) int64 {
	if len(seasons) == 0 {
		return s.Capacity
	}
	var weighted, days int64
	for _, se := range seasons {
		weighted += se.Days * seasonBP(s.Window, se.ID, wholeBP)
		days += se.Days
	}
	return s.Capacity * weighted / (days * wholeBP) // a valid calendar's seasons sum to its days, at least 1
}
