package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/fiscal"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the taxes of the fiscal module, P1.7b (spec-fiscal.md 5, 10.1;
// a continuation file of fiscal.go, architecture.md 2.6): the taxes on households
// and pools, then the remittances of subordinate treasuries. The payouts of the
// position are fiscal_payout.go. Nothing is drawn (FR-FIS-062).

// p17bFiscal is P1.7b (spec-world.md 6, spec-fiscal.md 5.3.1): the taxes whose
// target is households or a pool in ascending flow ID, then the taxes whose target
// is a treasury in ascending flow ID, so a subordinate authority remits what it
// collected from its own households in the same tick (FR-FIS-022, 023), then the
// payouts, from what the treasuries hold after them (fiscal_payout.go). A flow
// outside its window or at scale 0 is skipped whole. The events follow, per flow in
// the order it ran and per region ascending. A ledger refusal of an amount checked
// against the balance and the headroom is a violated invariant.
func (w *World) p17bFiscal() {
	s := w.fiscal
	s.ran = s.ran[:0]
	tick, season := w.Tick(), w.tk.moment.Season
	for _, remit := range [...]bool{false, true} {
		for i := range s.flows {
			f := &s.flows[i]
			if f.kind == fkTax && (f.target.kind == TargetTreasury) == remit && f.active(tick, season) {
				s.ran = append(s.ran, i)
				w.runTax(s, i)
			}
		}
	}
	w.runPayouts(s)
	w.reportFiscal(s, tick)
}

// runTax applies tax i to the accounts of its class: the dues over the carries,
// the cut to the treasury's headroom if the collections would pass it, then the
// transfers, in ascending account ID.
func (w *World) runTax(s *fiscalState, i int) {
	f := &s.flows[i]
	clear(s.agg[i*s.regions : (i+1)*s.regions])
	s.gather(w, f)
	w.dueTax(s, f)
	to := s.pol.tr[f.treasury].account
	v, err := w.ledger.Account(to)
	if err != nil {
		invariant("the account of a treasury is unreadable: " + err.Error())
	}
	s.capCollection(w.Tick(), fiscal.Headroom(v.Balance, MaxMoney))
	for k, p := range s.payers {
		if a := s.applied[k]; a > 0 {
			if err := w.ledger.Transfer(p.account, to, a, f.reason); err != nil {
				invariant("the ledger refused a tax's transfer: " + err.Error())
			}
			w.netCharge(f, p, a)
		}
		s.count(i, k)
	}
}

// dueTax computes the due and the collection of each payer and advances its carry,
// merging the payers (ascending account ID) with the flow's pairs (ascending too):
// a payer with no pair starts at the flow's initial carry, a pair of an account that
// is no payer now is kept while the account is open, and a pair of a closed account
// is dropped (plan 3-fis D3, FR-FIS-021, 024). The carry advances by the due and not
// by what was collected: what a payer could not pay is forgiven.
func (w *World) dueTax(s *fiscalState, f *fiscalFlowState) {
	rule := f.rule(w.cfgFiscalTax(f))
	walk := w.walkCarries(f.carry, s.next[:0])
	s.dues, s.take = s.dues[:0], s.take[:0]
	s.checkAscending(f)
	for _, p := range s.payers {
		res, ok := rule.Due(p.weight, w.available(p.account), walk.start(p.account, rule.StartCarry(int64(p.account))))
		if !ok {
			invariant("a tax's arithmetic left its validated bounds: " + f.id)
		}
		walk.put(p.account, res.Carry)
		s.dues, s.take = append(s.dues, res.Due), append(s.take, res.Collected)
	}
	f.carry, s.next = walk.finish()
}

// checkAscending refuses payers that are not in ascending account ID, which the merge
// with the saved carries needs.
func (s *fiscalState) checkAscending(f *fiscalFlowState) {
	for k := 1; k < len(s.payers); k++ {
		if s.payers[k].account <= s.payers[k-1].account {
			invariant("a fiscal flow's payers are not in ascending account order: " + f.id)
		}
	}
}

// carryWalk is the merge of a flow's saved pairs with the accounts it acts on, both
// in ascending account ID: it builds the flow's next table in one pass.
type carryWalk struct {
	w         *World
	old, next []carryPair
	j         int
}

// walkCarries starts a merge of the pairs old into the table next, which is empty.
func (w *World) walkCarries(old, next []carryPair) carryWalk {
	return carryWalk{w: w, old: old, next: next}
}

// start returns the carry of the account: its pair if it has one, else initial. The
// pairs of lower accounts are kept while open and dropped when closed.
func (c *carryWalk) start(account int, initial int64) int64 {
	for ; c.j < len(c.old) && c.old[c.j].account < account; c.j++ {
		c.next = c.w.keepCarry(c.next, c.old[c.j])
	}
	if c.j < len(c.old) && c.old[c.j].account == account {
		c.j++
		return c.old[c.j-1].carry
	}
	return initial
}

// put records the carry of the account after the tick.
func (c *carryWalk) put(account int, carry int64) {
	c.next = append(c.next, carryPair{account: account, carry: carry})
}

// finish keeps the pairs of the accounts above the last one acted on and returns the
// new table and the old one emptied, to be reused as scratch.
func (c *carryWalk) finish() (table, spare []carryPair) {
	for ; c.j < len(c.old); c.j++ {
		c.next = c.w.keepCarry(c.next, c.old[c.j])
	}
	return c.next, c.old[:0]
}

// cfgFiscalTax returns the tax the flow was built from.
func (w *World) cfgFiscalTax(f *fiscalFlowState) *Tax { return &w.cfg.Modules.Fiscal.Taxes[f.at] }

// rule is the part of a tax that does not depend on the payer (spec-fiscal.md 5.2),
// at the flow's scale.
func (f *fiscalFlowState) rule(t *Tax) fiscal.Rule {
	return fiscal.Rule{Shape: f.shape, Amount: t.Amount, RateBP: t.RateBP, Exempt: t.Exempt,
		Scale: f.scale, Period: f.period, MaxMoney: MaxMoney}
}

// keepCarry appends a pair to the new table when its account is still open: a pair
// of a closed account is dropped with the account (spec-fiscal.md 17).
func (w *World) keepCarry(next []carryPair, p carryPair) []carryPair {
	if w.accountOpen(p.account) {
		return append(next, p)
	}
	return next
}

// available is the balance of an account that a transfer may spend: the balance
// less what is reserved (no reservation is open at P1.7b, which follows P4.1 of the
// last tick).
func (w *World) available(account int) int64 {
	v, err := w.ledger.Account(account)
	if err != nil {
		invariant("the account of a payer is unreadable: " + err.Error())
	}
	return max(v.Balance-v.Reserved, 0)
}

// capCollection sets s.applied to what is taken from each payer: its collection,
// cut pro rata by the largest remainder when the collections together would pass
// the treasury's headroom (spec-fiscal.md 5.3.2, FR-FIS-025). The tie is the keyed
// tie of 8.4.2, K(tick, owner agent).
func (s *fiscalState) capCollection(tick, headroom int64) {
	var total int64
	for _, c := range s.take {
		total = satAdd(total, c)
	}
	if total <= headroom {
		s.applied = append(s.applied[:0], s.take...)
		return
	}
	s.rec = s.rec[:0]
	for k, p := range s.payers {
		s.rec = append(s.rec, fiscal.Recipient{Due: s.take[k], Key: ordering.Key(tick, int64(p.owner)), ID: int64(p.account)})
	}
	applied, ok := s.cut.Cut(s.applied, headroom, s.rec)
	if !ok {
		invariant("a tax's collections left their validated bounds")
	}
	s.applied = applied
}
