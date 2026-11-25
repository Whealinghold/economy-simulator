package economysim

import (
	"context"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the standing orders of the fiscal module (spec-fiscal.md 9; a
// continuation file of fiscal.go, architecture.md 2.6): a purchase is a standing bid
// and a release a standing ask of a treasury in the market of its seat region, placed
// at P3.4a. They are not a strategy and not an extension point. The treasury reads no
// market price, no belief and no window: the limit is a fraction of the effective
// reference price, the quantity a wish with a carry cut by the cash, the hold or the
// stock, and the order is admitted like an estate's ask (clearing.go admit). A
// treasury's order has no outcome and takes no word at P5.4 (spec-fiscal.md 9.5).

// fiscalOrder is what a purchase or a release derives from its configuration once: the
// commodity, the units a tick, the limit in basis points and the bound, which is
// Hold for a purchase and Reserve for a release.
type fiscalOrder struct {
	commodity CommodityID
	qty       int64
	limitBP   int64
	bound     int64
}

// newOrder resolves the commodity of a standing order. A name that is not defined is a
// violated invariant: the validator rejects it, and index 0 would order another good.
func newOrder(c *Config, commodity string, qty, limitBP, bound int64) fiscalOrder {
	i := slices.IndexFunc(c.Commodities, func(x Commodity) bool { return x.ID == commodity })
	if i < 0 {
		invariant("a standing order names a commodity that is not defined: " + commodity)
	}
	return fiscalOrder{commodity: CommodityID(i), qty: qty, limitBP: limitBP, bound: bound}
}

// orderTally counts the orders of one authority in the tick, the fields of
// TreasuryOrdered (spec-fiscal.md 16.1.2). wished is whether a standing order was in
// its window with a wish of at least 1, which is when the event is emitted.
type orderTally struct {
	bids, asks                                   int64
	wantedBuy, placedBuy, wantedSell, placedSell int64
	cutCash, cutHold, cutStock                   int64
	wished                                       bool
}

// p34aOrders is P3.4a: for each authority in ascending JurisdictionID, its purchases
// in ascending flow ID and then its releases in ascending flow ID, each flow in its
// window and above scale 0 giving one order or none; then TreasuryOrdered. The flows
// are in the canonical order of the lists, so the bid of an authority comes before its
// ask, the order of the market's input (spec-fiscal.md 9.4.2).
func (w *World) p34aOrders() {
	s := w.fiscal
	tick, season := w.Tick(), w.tk.moment.Season
	for t := range s.pol.tr {
		var c orderTally
		for pos := s.orders[0]; pos < s.orders[1]; pos++ {
			if f := &s.flows[pos]; f.treasury == t && f.active(tick, season) {
				w.standingOrder(f, &c)
			}
		}
		w.reportOrdered(&s.pol.tr[t], &c)
	}
}

// standingOrder is the step of one purchase or release: the wish over its carry, the
// caps in the order of spec-fiscal.md 9.3.2, and the order if at least one unit is
// left. The carry advances whether or not an order is placed.
func (w *World) standingOrder(f *fiscalFlowState, c *orderTally) {
	tr := &w.fiscal.pol.tr[f.treasury]
	wish, carry, ok := fiscal.Wish(f.order.qty, f.scale, f.single)
	if !ok {
		invariant("a standing order's wish is outside its bounds after validation: " + f.id)
	}
	f.single = carry
	if wish < 1 {
		return
	}
	c.wished = true
	buy := f.kind == fkPurchase
	limit, ok := fiscal.Limit(int64(w.refPrice(tr.seat, f.order.commodity)), f.order.limitBP, MaxPrice)
	if !ok {
		invariant("a standing order's limit is outside its bounds after validation: " + f.id)
	}
	capped := w.capOrder(tr, f, wish, limit)
	c.cutCash, c.cutHold, c.cutStock = satAdd(c.cutCash, capped.CutCash), satAdd(c.cutHold, capped.CutHold), satAdd(c.cutStock, capped.CutStock)
	if buy {
		c.wantedBuy = satAdd(c.wantedBuy, wish)
	} else {
		c.wantedSell = satAdd(c.wantedSell, wish)
	}
	if capped.Qty < 1 || !w.submitTreasury(tr, f, limit, capped.Qty) {
		return
	}
	if buy {
		c.bids, c.placedBuy = c.bids+1, satAdd(c.placedBuy, capped.Qty)
	} else {
		c.asks, c.placedSell = c.asks+1, satAdd(c.placedSell, capped.Qty)
	}
}

// capOrder applies the caps of the flow to a wish: for a purchase the cash and then the
// hold, for a release the unreserved stock less the reserve. It reads the treasury's
// account as it is, so an order placed earlier in the position has reserved its cash or
// units (spec-fiscal.md 9.3.2).
func (w *World) capOrder(tr *treasury, f *fiscalFlowState, wish, limit int64) fiscal.Capped {
	held, reserved, err := w.ledger.Holding(tr.account, int(f.order.commodity))
	if err != nil {
		invariant("the holding of a treasury is unreadable: " + err.Error())
	}
	if f.kind == fkRelease {
		return fiscal.CapRelease(wish, held-reserved, f.order.bound)
	}
	v, err := w.ledger.Account(tr.account)
	if err != nil {
		invariant("the account of a treasury is unreadable: " + err.Error())
	}
	return fiscal.CapPurchase(wish, v.Balance-v.Reserved, limit, f.order.bound, held)
}

// submitTreasury offers one order of a treasury to the market of its seat region and
// reports whether the book admitted it, with OrderPlaced, or refused it, with
// OrderRejected and the reason. A treasury may place either side, is in no role and so
// has every commodity tradable (spec-fiscal.md 9.1.1), and its order has no record of
// an intent, like an estate's.
func (w *World) submitTreasury(tr *treasury, f *fiscalFlowState, limit, qty int64) bool {
	side := Bid
	if f.kind == fkRelease {
		side = Ask
	}
	o := Order{Agent: tr.agent, Commodity: f.order.commodity, Side: side, Price: Price(limit), Quantity: Quantity(qty)}
	mi := w.markets.index(tr.seat, o.Commodity)
	b := w.markets.books[mi]
	facts := orderFacts{Account: tr.account, AgentKnown: true, SideAllowed: true, CommodityKnown: true, Tradable: true, Tally: &w.receipts}
	reason, ok := b.admit(w.ledger, facts, o)
	if !ok {
		b.emitRejected(&w.obs, w.log, o, reason)
		return false
	}
	b.emitPlaced(&w.obs, o)
	w.markets.record(mi, intentRec{})
	return true
}

// reportOrdered emits TreasuryOrdered for the authority, after its orders, when a
// standing order of it was in its window with a wish of at least 1 (FR-FIS-045), and
// its Debug line. The counters are kept whether or not an observer was supplied; the
// event is built only with one (observability-and-logging.md 4.5).
func (w *World) reportOrdered(tr *treasury, c *orderTally) {
	if !c.wished {
		return
	}
	if w.obs.supplied {
		w.obs.deliver(TreasuryOrdered{Tick: w.Tick(), Authority: tr.j, Region: tr.seat, Bids: c.bids, Asks: c.asks,
			WantedBuy: c.wantedBuy, PlacedBuy: c.placedBuy, WantedSell: c.wantedSell, PlacedSell: c.placedSell,
			CutCash: c.cutCash, CutHold: c.cutHold, CutStock: c.cutStock})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "treasury ordered", slog.String(logkeys.Authority, w.fiscal.pol.v.Name(tr.j)),
			slog.Int64(logkeys.Region, int64(tr.seat)))
	}
}
