package economysim

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/fiscal"
	"github.com/Whealinghold/economy-sim/internal/moneyflow"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the payouts of the fiscal module, the third pass of P1.7b
// (spec-fiscal.md 8, 12; a continuation file of fiscal.go, architecture.md 2.6): the
// dues of every recipient of one authority and one resource are computed first, the
// treasury's stock is shared among them pro rata when it is short, and the transfers
// follow. A payout never takes the treasury below zero (FR-FIS-063) and draws nothing
// (FR-FIS-062).

// payEntry is one recipient of one payout flow in the tick: its due, what the pro
// rata cut left it (share) and what it received (applied, after the headroom).
type payEntry struct {
	flow                int
	p                   fiscalPayer
	due, share, applied int64
}

// fiscalReceipt is the units of a commodity a payout gave to a household: valued at
// the prices of the tick at P6.3 and added to its unearned income (spec-fiscal.md
// 12.1, 12.3). It is scratch, never saved.
type fiscalReceipt struct {
	slot      int
	region    RegionID
	commodity int
	units     int64
}

// runPayouts applies the payouts that run in the tick: one group per authority in
// ascending JurisdictionID, and within it money first, then the commodities in
// ascending order; the flows of a group in ascending flow ID (spec-fiscal.md 5.3.1,
// 8.4.1).
func (w *World) runPayouts(s *fiscalState) {
	tick, season := w.Tick(), w.tk.moment.Season
	s.order = s.order[:0]
	for i := range s.flows {
		if f := &s.flows[i]; f.kind == fkPayout && f.active(tick, season) {
			s.order = append(s.order, i)
		}
	}
	s.sortGroups()
	for lo := 0; lo < len(s.order); {
		hi := lo + 1
		for hi < len(s.order) && s.sameGroup(s.order[lo], s.order[hi]) {
			hi++
		}
		w.payGroup(s, s.order[lo:hi])
		lo = hi
	}
}

// sameGroup reports whether two payouts draw on the same treasury and resource.
func (s *fiscalState) sameGroup(a, b int) bool {
	return s.flows[a].authority == s.flows[b].authority && s.flows[a].resource == s.flows[b].resource
}

// sortGroups orders s.order by authority, then resource, keeping the flow order within
// a group: an insertion sort, stable, with no allocation, over a handful of flows.
func (s *fiscalState) sortGroups() {
	for i := 1; i < len(s.order); i++ {
		for j := i; j > 0 && s.groupBefore(s.order[j], s.order[j-1]); j-- {
			s.order[j], s.order[j-1] = s.order[j-1], s.order[j]
		}
	}
}

// groupBefore reports whether payout a belongs to an earlier group than payout b.
func (s *fiscalState) groupBefore(a, b int) bool {
	fa, fb := &s.flows[a], &s.flows[b]
	if fa.authority != fb.authority {
		return fa.authority < fb.authority
	}
	return fa.resource < fb.resource
}

// payGroup pays one group: the dues of all its flows first, then the share of the
// stock each is left, then the transfers in the order of the entries.
func (w *World) payGroup(s *fiscalState, group []int) {
	s.ent = s.ent[:0]
	for _, pos := range group {
		f := &s.flows[pos]
		clear(s.agg[pos*s.regions : (pos+1)*s.regions])
		s.ran = append(s.ran, pos)
		s.gather(w, f)
		w.duePayout(s, f, pos)
	}
	f := &s.flows[group[0]]
	from := s.pol.tr[f.treasury].account
	s.share(w.Tick(), w.stock(from, f.resource))
	for k := range s.ent {
		w.pay(s, from, &s.ent[k])
	}
}

// duePayout appends the entry of each recipient of the flow with its due, advancing
// the carry of a flow that has one by the due (spec-fiscal.md 8.3).
func (w *World) duePayout(s *fiscalState, f *fiscalFlowState, pos int) {
	po := &w.cfg.Modules.Fiscal.Payouts[f.at]
	var walk carryWalk
	if po.Amount != 0 {
		walk = w.walkCarries(f.carry, s.next[:0])
	}
	s.checkAscending(f)
	room := int64(MaxMoney)
	if f.resource >= 0 {
		room = MaxQuantity
	}
	for _, p := range s.payers {
		var due int64
		ok := true
		if po.Amount == 0 {
			due, ok = moneyflow.TopUp(p.weight, po.Level, f.scale, w.balance(p.account), MaxMoney)
		} else {
			var next int64
			c := walk.start(p.account, moneyflow.StartCarry(int64(p.account), f.period))
			due, next, ok = moneyflow.Spread(p.weight, po.Amount, f.scale, c, f.period, room)
			walk.put(p.account, next)
		}
		if !ok {
			invariant("a payout's arithmetic left its validated bounds: " + f.id)
		}
		s.ent = append(s.ent, payEntry{flow: pos, p: p, due: due})
	}
	if po.Amount != 0 {
		f.carry, s.next = walk.finish()
	}
}

// share sets the share of every entry: its due if the stock covers them all, else the
// pro rata cut with the largest remainder, the tie broken by K(tick, owner agent)
// (spec-fiscal.md 8.4.2).
func (s *fiscalState) share(tick, stock int64) {
	s.rec = s.rec[:0]
	for _, e := range s.ent {
		s.rec = append(s.rec, fiscal.Recipient{Due: e.due, Key: ordering.Key(tick, int64(e.p.owner)), ID: int64(e.p.account)})
	}
	out, ok := s.cut.Cut(s.applied, stock, s.rec)
	if !ok {
		invariant("a payout's dues left their validated bounds")
	}
	s.applied = out
	for k := range s.ent {
		s.ent[k].share = out[k]
	}
}

// pay moves the share of an entry from the treasury to its recipient, less what the
// recipient has no room for (the unapplied part stays in the treasury, spec-fiscal.md
// 8.4.3), and counts it.
func (w *World) pay(s *fiscalState, from int, e *payEntry) {
	f := &s.flows[e.flow]
	e.applied = min(e.share, w.room(e.p.account, f.resource))
	if e.applied > 0 {
		var err error
		if f.resource < 0 {
			err = w.ledger.Transfer(from, e.p.account, e.applied, f.reason)
		} else {
			err = w.ledger.Move(from, e.p.account, f.resource, e.applied, f.reason)
		}
		if err != nil {
			invariant("the ledger refused a payout's transfer: " + err.Error())
		}
		w.income(s, f, e)
	}
	g := &s.agg[e.flow*s.regions+int(e.p.region)]
	g.due, g.applied = satAdd(g.due, e.due), satAdd(g.applied, e.applied)
	if e.due > 0 {
		g.parties = satAdd(g.parties, e.p.weight)
	}
	if e.share < e.due {
		g.cut = satAdd(g.cut, e.p.weight)
	}
	if e.applied < e.share {
		g.headrm = satAdd(g.headrm, e.p.weight)
	}
}

// income adds what a household received to its unearned income of the tick, unless
// the payout is NotIncome (spec-fiscal.md 12.1): the money now, the goods as a
// receipt valued at P6.3.
func (w *World) income(s *fiscalState, f *fiscalFlowState, e *payEntry) {
	if f.target.kind != TargetHouseholds || w.cfg.Modules.Fiscal.Payouts[f.at].NotIncome {
		return
	}
	if f.resource < 0 {
		w.welfare.addUnearned(int(e.p.owner), e.applied)
		return
	}
	s.receipts = append(s.receipts, fiscalReceipt{slot: int(e.p.owner), region: e.p.region, commodity: f.resource, units: e.applied})
}

// netCharge nets a tax, a duty or a toll a household paid from its income of the tick
// (X_t, spec-fiscal.md 12.1); a pool or a treasury is no household.
func (w *World) netCharge(f *fiscalFlowState, p fiscalPayer, amount int64) {
	if f.target.kind == TargetHouseholds {
		w.welfare.addUnearned(int(p.owner), -amount)
	}
}

// priceFiscalReceipts is the step of P6.3 for the goods the payouts gave to
// households: each is valued at the price of its region and commodity and added to
// the household's unearned income, then the list is cleared (spec-fiscal.md 12.3).
func (w *World) priceFiscalReceipts() {
	s, nc := w.fiscal, len(w.cfg.Commodities)
	for _, r := range s.receipts {
		v, ok := checked.Mul(r.units, w.oc.valued[int(r.region)*nc+r.commodity])
		if !ok {
			v = math.MaxInt64
		}
		w.welfare.addUnearned(r.slot, v)
	}
	s.receipts = s.receipts[:0]
}

// balance reads the money of an account.
func (w *World) balance(account int) int64 {
	v, err := w.ledger.Account(account)
	if err != nil {
		invariant("the account of a recipient is unreadable: " + err.Error())
	}
	return v.Balance
}

// held reads the units of a commodity an account holds and the units earmarked.
func (w *World) held(account, commodity int) (held, reserved int64) {
	h, r, err := w.ledger.Holding(account, commodity)
	if err != nil {
		invariant("the holding of an account is unreadable: " + err.Error())
	}
	return h, r
}

// stock is what a treasury can pay of a resource: its unreserved money or its
// unearmarked units (spec-fiscal.md 8.4.1).
func (w *World) stock(account, resource int) int64 {
	if resource < 0 {
		return w.available(account)
	}
	h, r := w.held(account, resource)
	return max(h-r, 0)
}

// room is how much of a resource an account can still take before its bound: MaxMoney
// of money or MaxQuantity of a commodity.
func (w *World) room(account, resource int) int64 {
	if resource < 0 {
		return max(MaxMoney-w.balance(account), 0)
	}
	h, _ := w.held(account, resource)
	return max(MaxQuantity-h, 0)
}
