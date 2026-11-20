package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// This file is the duties of the fiscal module, P4.1a (spec-fiscal.md 6, 10.1; a
// continuation file of fiscal.go, architecture.md 2.6): a share of the value of the
// trades of the markets an authority covers, taken after the trade from the party the
// flow names. The clearing hands the trades of each market over as it settles them
// (noteDutyTrades, a sub-step of P4.1), and P4.1a sums them per (duty, payer) and
// takes the dues once every market has settled. Nothing is drawn (FR-FIS-062).

// dutyScope says which trades a duty takes: by whether a party is a counterparty of
// the outside world (spec-fiscal.md 6.1.1).
type dutyScope uint8

const (
	scopeAny dutyScope = iota
	scopeDomestic
	scopeImport
	scopeExport
)

// dutyTrade is one trade a duty may take: the payer and the value, price times
// quantity. It is scratch of the tick, never saved.
type dutyTrade struct {
	payer AgentID
	value int64
}

// fiscalDuty is what a duty derives from its configuration once, and the trades the
// clearing handed over in the tick.
type fiscalDuty struct {
	seller    bool // the seller pays, else the buyer
	scope     dutyScope
	commodity int // the commodity taxed, -1 for every commodity
	rateBP    int64
	trades    []dutyTrade
}

// newDuty resolves a duty's payer, scope and commodity. A commodity that is not
// defined is a violated invariant: the validator rejects it, and -1 would tax every
// commodity in its place.
func newDuty(c *Config, d *Duty) fiscalDuty {
	out := fiscalDuty{seller: d.Payer == DutySeller, commodity: -1, rateBP: d.RateBP}
	switch d.Scope {
	case ScopeDomestic:
		out.scope = scopeDomestic
	case ScopeImport:
		out.scope = scopeImport
	case ScopeExport:
		out.scope = scopeExport
	}
	if d.Commodity != "" {
		out.commodity = slices.IndexFunc(c.Commodities, func(x Commodity) bool { return x.ID == d.Commodity })
		if out.commodity < 0 {
			invariant("a duty names a commodity that is not defined: " + d.ID)
		}
	}
	return out
}

// isCounterparty reports whether the agent is a counterparty of the external module:
// the range of IDs that follows the estates' (spec-world.md 5.3).
func (w *World) isCounterparty(id AgentID) bool {
	lo := len(w.cfg.Regions)
	return int(id) >= lo && int(id) < lo+w.external.counterpartyCount()
}

// matches reports whether the trade is in the scope of the duty (spec-fiscal.md 6.1.1).
func (w *World) matches(sc dutyScope, t *Trade) bool {
	switch sc {
	case scopeDomestic:
		return !w.isCounterparty(t.Buyer) && !w.isCounterparty(t.Seller)
	case scopeImport:
		return w.isCounterparty(t.Seller)
	case scopeExport:
		return w.isCounterparty(t.Buyer)
	}
	return true
}

// noteDutyTrades is the hand-over of the clearing (spec-fiscal.md 6.2.1): the trades
// a market of the region settled, kept for every duty that is in its window, covers
// the region and is not paused. A trade whose payer side is a pseudo-agent is in no
// base and is counted in the duty's Exempt (6.4). The trades are the data the events
// carry, kept whether or not an observer is present.
func (s *fiscalState) noteDutyTrades(w *World, region RegionID, trades []Trade) {
	tick, season := w.Tick(), w.tk.moment.Season
	for i := range s.flows {
		if f := &s.flows[i]; f.kind == fkDuty && f.target.in[region] && f.active(tick, season) {
			w.noteDuty(f, &s.agg[i*s.regions+int(region)], trades)
		}
	}
}

// noteDuty keeps the trades of one market that the duty takes.
func (w *World) noteDuty(f *fiscalFlowState, g *fiscalAgg, trades []Trade) {
	d := &f.duty
	for k := range trades {
		t := &trades[k]
		if (d.commodity >= 0 && int(t.Commodity) != d.commodity) || !w.matches(d.scope, t) {
			continue
		}
		payer := t.Buyer
		if d.seller {
			payer = t.Seller
		}
		if int(payer) < w.firstHousehold {
			g.exempt = satAdd(g.exempt, 1)
			continue
		}
		v, ok := checked.Mul(int64(t.Price), int64(t.Quantity))
		if !ok {
			invariant("the value of a settled trade overflows")
		}
		d.trades = append(d.trades, dutyTrade{payer: payer, value: v})
	}
}

// p41aDuties is P4.1a (spec-world.md 6, spec-fiscal.md 6.5): after every market has
// cleared and settled and before the sweep of P4.2, the duties in ascending flow ID,
// each from its payers in ascending account ID. A flow outside its window or at scale
// 0 noted nothing and is skipped whole. The events follow, per flow and per region
// ascending. The trades of the tick and the counters are cleared at the end.
func (w *World) p41aDuties() {
	s := w.fiscal
	s.ran = s.ran[:0]
	tick, season := w.Tick(), w.tk.moment.Season
	for i := range s.flows {
		if f := &s.flows[i]; f.kind == fkDuty && f.active(tick, season) {
			s.ran = append(s.ran, i)
			w.runDuty(s, i)
		}
	}
	w.reportFiscal(s, tick)
	for _, pos := range s.ran {
		clear(s.agg[pos*s.regions : (pos+1)*s.regions])
	}
	for i := range s.flows {
		s.flows[i].duty.trades = s.flows[i].duty.trades[:0]
	}
}

// runDuty applies duty i: the bases of its payers, the dues over the carries, the cut
// to the treasury's headroom, then the transfers, in ascending account ID.
func (w *World) runDuty(s *fiscalState, i int) {
	f := &s.flows[i]
	s.gatherDuty(w, f)
	w.dueDuty(s, f)
	w.collect(s, i, f)
}

// gatherDuty lists the payers of the duty in ascending account ID with the base of
// each: the sum of the values of its trades, in 128 bits, so that the due does not
// depend on the order of the trades (spec-fiscal.md 6.2.2, FR-FIS-032).
func (s *fiscalState) gatherDuty(w *World, f *fiscalFlowState) {
	d := &f.duty
	slices.SortFunc(d.trades, func(a, b dutyTrade) int { return int(a.payer) - int(b.payer) })
	s.payers, s.bases = s.payers[:0], s.bases[:0]
	for k := range d.trades {
		t := d.trades[k]
		if n := len(s.payers); n == 0 || s.payers[n-1].owner != t.payer {
			a := &w.agents[t.payer]
			s.payers = append(s.payers, fiscalPayer{account: a.account, weight: a.weight, region: w.regionOf(a), owner: a.id})
			s.bases = append(s.bases, fiscal.Base{})
		}
		if !s.bases[len(s.bases)-1].Add(t.value) {
			invariant("the value of a duty's trade is negative")
		}
	}
}

// dueDuty computes the due and the collection of each payer and advances its carry:
// num = base * RateBP * S + carry, due = num / 10^8, the carry advancing by the due
// and not by what the payer could pay, which is forgiven (spec-fiscal.md 6.3). No
// reservation is open at P4.1a, so the payer's available money is its balance.
func (w *World) dueDuty(s *fiscalState, f *fiscalFlowState) {
	walk := w.walkCarries(f.carry, s.next[:0])
	s.dues, s.take = s.dues[:0], s.take[:0]
	s.checkAscending(f)
	for k, p := range s.payers {
		due, next, ok := fiscal.DutyDue(s.bases[k], f.duty.rateBP, f.scale, walk.start(p.account, 0), MaxMoney)
		if !ok {
			invariant("a duty's arithmetic left its validated bounds: " + f.id)
		}
		walk.put(p.account, next)
		s.dues, s.take = append(s.dues, due), append(s.take, min(due, w.available(p.account)))
	}
	f.carry, s.next = walk.finish()
}
