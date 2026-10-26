package economysim

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is phases 3 and 4 of the tick (spec-world.md 6.5, 6.6;
// plan-phase-1-11.md Phase 5): the trade decision of every agent (P3.2), the
// estates' asks (P3.4) and the clearing of every market (P4.1). The books are
// emptied at the start of P3.2 and every reservation is settled or released by
// the end of P4.1 (spec-world.md 6.1.6). The orders of a market are in input
// order: the agents' orders in ascending agent ID and plan order, then the
// estates' asks (spec-world.md 6.5.1). The positions of the phase that belong to
// modules are not built.

// p32Trade is P3.2: for each agent in ascending ID, the view from the state after
// P2.7, Trade, the check of the plan and its execution: the discards in plan
// order, then the orders in plan order, each admitted with its reservation or
// rejected with a reason. A pseudo-agent has no strategy and is skipped. The
// stream of the agent is keyed before every call of Trade, because the default
// takes one word for each intent it considers (D155). The books and the receipt
// tally are cleared first.
func (w *World) p32Trade() error {
	w.refreshViewCtx()
	w.receipts.reset()
	w.markets.reset(w.Tick())
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := w.tradeAgent(a); err != nil {
			return err
		}
	}
	return nil
}

// Untested, testing.md 9.5: the branch of p32Trade that skips an agent of weight
// below 1 is Phase 7's (a removed agent keeps its slot), and the error of
// tradeAgent is a ledger refusal of the account read, which a live agent cannot
// cause; both are kept so a defect is not silent.

// tradeAgent is the P3.2 step of one agent.
func (w *World) tradeAgent(a *agent) error {
	v := &w.tk.view
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err := w.fillAgentView(v, a); err != nil {
		return fmt.Errorf("trade at tick %d: %w", w.Tick(), err)
	}
	p := w.views.tradePlan()
	w.rnd.keyed(siteTrade, w.Tick(), int64(a.id))
	w.ext.strategies[a.strategy].Trade(v, w.rnd.rnd, p)
	w.checkTradePlan(a, p)
	w.executeDiscards(a, p)
	region := w.regionOf(a)
	for _, in := range p.Intents {
		w.submit(a, region, in)
	}
	return nil
}

// p36Aggregates is P3.6: the per-region, per-role aggregates of the trade phase,
// a fixed sequence of guarded calls into the module files (spec-world.md 6, D312).
func (w *World) p36Aggregates() {
	if w.demand != nil {
		w.p36Wants()
	}
}

// normalisePlan applies the clamps of the belief inputs to every intent of the
// plan and logs a Debug line for each one it changed (spec-agents.md 6.5.2).
func (w *World) normalisePlan(a *agent, p *TradePlan) {
	ctx := context.Background()
	for i := range p.Intents {
		in := &p.Intents[i]
		if normalise(in) && w.log.Enabled(ctx, slog.LevelDebug) {
			w.log.LogAttrs(ctx, slog.LevelDebug, "intent normalised",
				slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.Commodity, int64(in.Commodity)),
				slog.String(logkeys.Side, in.Side.String()))
		}
	}
}

// checkDiscards drops from the plan, in place and keeping the order, every
// discard the agent cannot make: a quantity below 1, a commodity outside the
// configuration, or more than the agent holds unearmarked once the discards kept
// before it are taken (spec-agents.md 6.5.2). Each refusal is a Debug line.
func (w *World) checkDiscards(a *agent, p *TradePlan) {
	ctx := context.Background()
	kept := p.Discards[:0]
	for _, d := range p.Discards {
		if w.discardable(a, d, kept) {
			kept = append(kept, d)
			continue
		}
		if w.log.Enabled(ctx, slog.LevelDebug) {
			w.log.LogAttrs(ctx, slog.LevelDebug, "discard refused",
				slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.Commodity, int64(d.Commodity)),
				slog.Int64(logkeys.Quantity, int64(d.Quantity)))
		}
	}
	p.Discards = kept
}

// discardable reports whether the agent can discard d on top of the discards
// kept before it. Each kept discard fits what the agent holds, so their sum is
// at most the holding and the subtraction cannot overflow.
func (w *World) discardable(a *agent, d Discard, kept []Discard) bool {
	if d.Quantity < 1 || d.Commodity < 0 || int(d.Commodity) >= len(w.cfg.Commodities) {
		return false
	}
	held, reserved, err := w.ledger.Holding(a.account, int(d.Commodity))
	if err != nil {
		// Excluded from coverage (testing.md 9.5): the account and the commodity
		// were checked above, so the ledger cannot refuse the read.
		invariant("the ledger refused the holding of a live agent: " + err.Error())
	}
	free := held - reserved
	for _, k := range kept {
		if k.Commodity == d.Commodity {
			free -= int64(k.Quantity)
		}
	}
	return int64(d.Quantity) <= free
}

// executeDiscards moves the discards of a checked plan, in plan order, to the
// disposal boundary (spec-agents.md 8.2).
func (w *World) executeDiscards(a *agent, p *TradePlan) {
	for _, d := range p.Discards {
		sink, ok := w.ledger.Boundary(ledger.Disposal, int(d.Commodity))
		if !ok {
			invariant("a commodity has no disposal boundary account")
		}
		if err := w.ledger.Move(a.account, sink, int(d.Commodity), int64(d.Quantity), ledger.Disposal); err != nil {
			// Excluded from coverage (testing.md 9.5): checkDiscards ruled it out.
			invariant("the ledger refused a checked discard: " + err.Error())
		}
	}
}

// marketOf returns the index of the market an order of a household of region
// goes to, and the reason the world rejects it for before the book sees it: a
// side that is neither bid nor ask (side_not_allowed) or a commodity outside the
// configuration (unknown_commodity). A rejected order of an unknown commodity is
// reported on the first market of the region, which has the right region and
// tick; no book holds it.
func (w *World) marketOf(region RegionID, o Order) (int, Reason) {
	known := o.Commodity >= 0 && int(o.Commodity) < len(w.cfg.Commodities)
	c := o.Commodity
	if !known {
		c = 0
	}
	mi := w.markets.index(region, c)
	switch {
	case o.Side != Bid && o.Side != Ask:
		return mi, ReasonSideNotAllowed
	case !known:
		return mi, ReasonUnknownCommodity
	}
	return mi, ""
}

// submit turns one intent of agent a into an order and offers it to its market:
// admitted (OrderPlaced, and the facts of the intent kept at the entry's index) or
// rejected with a reason (OrderRejected). The book builds both events and the
// Debug line of a rejection.
func (w *World) submit(a *agent, region RegionID, in Intent) {
	o := orderOf(a, in)
	mi, reason := w.marketOf(region, o)
	b := w.markets.books[mi]
	if reason == "" {
		f := orderFacts{
			Account: a.account, AgentKnown: true, SideAllowed: true, CommodityKnown: true,
			Tradable: w.roles.row(a.role).isTradable(o.Commodity), Tally: &w.receipts,
		}
		var ok bool
		if reason, ok = b.admit(w.ledger, f, o); ok {
			b.emitPlaced(&w.obs, o)
			w.markets.record(mi, intentRec{purpose: in.Purpose, target: in.Target, reserve: in.Reserve, unitCost: in.UnitCost})
			return
		}
	}
	b.emitRejected(&w.obs, w.log, o, reason)
}

// p34Estates is P3.4: for each region in ascending order, one named-lot ask for
// each lot its estate holds, admitted like any other order after the agents'
// (spec-world.md 6.5.1). An estate's ask has no outcome, so its record is zero.
func (w *World) p34Estates() error {
	for r := range w.cfg.Regions {
		region := RegionID(r)
		orders, err := w.estates.asks(w.ledger, &w.cfg, w.refs, region, w.markets.asks[:0])
		if err != nil { // excluded from coverage (testing.md 9.5): asks returns a nil error today
			return fmt.Errorf("estate asks at tick %d: %w", w.Tick(), err)
		}
		w.markets.asks = orders
		for _, o := range orders {
			w.submitEstate(region, o)
		}
	}
	return nil
}

// submitEstate offers one ask of the estate of region to its market.
func (w *World) submitEstate(region RegionID, o Order) {
	mi := w.markets.index(region, o.Commodity)
	b := w.markets.books[mi]
	reason, ok := b.admit(w.ledger, w.estates.facts(region, Ask, &w.receipts), o)
	if !ok { // excluded from coverage (testing.md 9.5): an estate's ask is for units it holds at a price of at least the floor, so only a receipt limit refuses it
		b.emitRejected(&w.obs, w.log, o, reason)
		return
	}
	w.estates.admitted(region, o)
	b.emitPlaced(&w.obs, o)
	w.markets.record(mi, intentRec{})
}

// p41Clear is P4.1: every market in ascending (RegionID, CommodityID), clearing
// it, settling its trades and publishing its report. A market with no order is
// cleared too, for the report. An invalid list fails the step at once and the
// markets after it are not cleared (spec-world.md 6.6.1).
func (w *World) p41Clear() error {
	tick := w.Tick()
	for i := range w.markets.books {
		if err := w.clearOne(i, tick); err != nil {
			return err
		}
	}
	return nil
}

// clearOne clears market i: the cost anchor in force on the book, the stream of
// the market keyed by (tick, region, commodity), the slots of its buyers, the
// clearing (which emits TradeExecuted and MarketCleared), then the welfare feeds
// of the settled trades and the published report.
func (w *World) clearOne(i int, tick int64) error {
	region, commodity := RegionID(i/w.markets.commodities), CommodityID(i%w.markets.commodities)
	b := w.markets.books[i]
	b.anchor = w.prod.unitCost(region, commodity)
	w.rnd.keyedPair(siteClearing, tick, int64(region), int64(commodity))
	w.markets.loadSlots(i)
	defer w.markets.clearSlots(i)
	rep, trades, err := clearMarket(w.ledger, b, w.ext.clearing, w.rnd.rnd, w.markets.reports[i], &w.obs, w.log, w.markets.trades[:0])
	w.markets.trades = trades
	if err != nil {
		return fmt.Errorf("clearing at tick %d: %w", tick, err)
	}
	w.markets.reports[i] = rep
	w.feedWelfare(i, trades)
	return nil
}

// feedWelfare is the welfare feeds of P4.1 (spec-world.md 6.1.13) for the trades
// a market settled: the money each seller received, the money a buyer spent on a
// bid of the purpose Input, and the working tick of both sides of a trade. A
// pseudo-agent has no welfare slot.
func (w *World) feedWelfare(mi int, trades []Trade) {
	pseudo := AgentID(len(w.cfg.Regions))
	for _, t := range trades {
		pay, ok := checked.Mul(int64(t.Price), int64(t.Quantity))
		if !ok {
			invariant("the value of a settled trade overflows")
		}
		if t.Seller >= pseudo {
			w.welfare.addReceipt(int(t.Seller), pay)
			w.welfare.markTraded(int(t.Seller))
		}
		// Excluded from coverage (testing.md 9.5): only an estate is a pseudo-agent and
		// an estate never bids, so a buyer is never below pseudo.
		if t.Buyer < pseudo {
			continue
		}
		w.welfare.markTraded(int(t.Buyer))
		if w.markets.intentOf(mi, t.Buyer).purpose == PurposeInput {
			w.welfare.addInputSpend(int(t.Buyer), pay)
		}
	}
}
