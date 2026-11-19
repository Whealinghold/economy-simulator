package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/fiscal"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the state of the fiscal module in a checkpoint, tag 30 (spec-fiscal.md
// 17, spec-checkpoint.md 4.2; a continuation file of fiscal.go, architecture.md 2.6),
// and its audit. The section is written only when the module is on. It holds, for
// each flow that keeps a carry in the canonical order (taxes, duties, payouts, then
// purchases, releases), the carries as (account ID, carry) pairs in ascending account
// ID, then the sparse list of the scales that are not 10 000.

// keepsCarry reports whether the flow keeps a carry per account: a tax, a duty and a
// payout with an amount do (a top-up has none). A purchase and a release keep one
// carry in all; a toll keeps none.
func (f *fiscalFlowState) keepsCarry(c *Config) bool {
	switch f.kind {
	case fkTax, fkDuty:
		return true
	case fkPayout:
		return c.Modules.Fiscal.Payouts[f.at].Amount != 0
	}
	return false
}

// carryDen is the bound of a carry: the period times 10^4 for a fixed tax and a rate
// payout, times 10^8 for a rate tax and a duty (spec-fiscal.md 5.2, 6.3).
func (f *fiscalFlowState) carryDen() int64 {
	den := f.period * wholeBP
	if f.kind == fkDuty || (f.kind == fkTax && f.shape == fiscal.Rate) {
		den *= wholeBP
	}
	return den
}

// accountKind is the kind of ledger account a flow's pairs name: a pool's Estate
// account, a treasury's Treasury account and an Agent account otherwise.
func (f *fiscalFlowState) accountKind() ledger.Kind {
	switch {
	case f.kind == fkDuty:
		return ledger.KindAgent
	case f.target.kind == TargetPool:
		return ledger.KindEstate
	case f.target.kind == TargetTreasury:
		return ledger.KindTreasury
	}
	return ledger.KindAgent
}

// saveFiscal is tag 30: per flow that keeps a carry the pairs of the open accounts,
// per purchase and release its one carry, then the scales that differ from 10 000 as
// (position, scale) pairs in ascending position. The carry of a closed account is
// not saved (FR-FIS-060).
func (w *World) saveFiscal(cw *codec.Writer) error {
	s := w.fiscal
	for i := range s.flows {
		f := &s.flows[i]
		switch {
		case f.kind == fkPurchase || f.kind == fkRelease:
			cw.U(uint64(f.single))
		case f.keepsCarry(&w.cfg):
			w.saveCarryPairs(cw, f)
		}
	}
	n := 0
	for i := range s.flows {
		if s.flows[i].scale != wholeBP {
			n++
		}
	}
	cw.Count(n)
	for i := range s.flows {
		if s.flows[i].scale != wholeBP {
			cw.U(uint64(i))
			cw.U(uint64(s.flows[i].scale))
		}
	}
	return nil
}

// saveCarryPairs writes a count and the pairs of one flow, skipping the pairs of
// closed accounts.
func (w *World) saveCarryPairs(cw *codec.Writer, f *fiscalFlowState) {
	n := 0
	for _, p := range f.carry {
		if w.accountOpen(p.account) {
			n++
		}
	}
	cw.Count(n)
	for _, p := range f.carry {
		if w.accountOpen(p.account) {
			cw.U(uint64(p.account))
			cw.U(uint64(p.carry))
		}
	}
}

// accountOpen reports whether the account exists and is open.
func (w *World) accountOpen(id int) bool {
	v, err := w.ledger.Account(id)
	return err == nil && !v.Closed
}

// loadFiscal is tag 30: every carry inside its denominator on an open account of the
// right kind, strictly ascending by account, then the scale list with each position
// a flow, strictly ascending, and no scale of 10 000 (FR-FIS-061). A world restored
// from it applies the same payments as an uninterrupted one.
func (w *World) loadFiscal(_ *restoreCtx, r *codec.Reader) error {
	s := w.fiscal
	for i := range s.flows {
		f := &s.flows[i]
		switch {
		case r.Err() != nil:
		case f.kind == fkPurchase || f.kind == fkRelease:
			f.single = r.Bounded("flow carry", wholeBP-1)
		case f.keepsCarry(&w.cfg):
			w.loadCarryPairs(r, f)
		}
	}
	w.loadScales(r)
	return nil
}

// loadScales reads the sparse list of scales: a count, then (position, scale) with
// the position strictly ascending and a flow, and the scale in its bound and not
// 10 000, which a writer leaves out (spec-checkpoint.md 4.1.4).
func (w *World) loadScales(r *codec.Reader) {
	s := w.fiscal
	prev := uint64(0)
	for i := range r.Count(2) {
		pos := r.Strict(prev, i > 0)
		if r.Err() != nil || pos >= uint64(len(s.flows)) {
			r.Failf("flow scale names flow %d, outside the %d flows", pos, len(s.flows))
			return
		}
		bp := r.Bounded("flow scale", MaxFlowScale)
		if r.Err() != nil {
			return
		}
		if bp == wholeBP {
			r.Failf("flow scale of flow %d is 10 000, which is the default and is not saved", pos)
			return
		}
		prev, s.flows[pos].scale = pos, bp
	}
}

// loadCarryPairs reads the pairs of one flow into its table: the account strictly
// ascending, open and of the kind the flow acts on, the carry inside its denominator.
func (w *World) loadCarryPairs(r *codec.Reader, f *fiscalFlowState) {
	f.carry = f.carry[:0]
	den, prev := f.carryDen(), uint64(0)
	for i := range r.Count(2) {
		acct := r.Strict(prev, i > 0)
		carry := r.Bounded("flow carry", den-1)
		if r.Err() != nil {
			return
		}
		prev = acct
		if !w.accountIs(int(acct), f.accountKind()) {
			r.Failf("flow carry names account %d, which is not an open %v account", acct, f.accountKind())
			return
		}
		f.carry = append(f.carry, carryPair{account: int(acct), carry: carry})
	}
}

// accountIs reports whether the account is open and of the kind.
func (w *World) accountIs(id int, k ledger.Kind) bool {
	v, err := w.ledger.Account(id)
	return err == nil && !v.Closed && v.Kind == k
}

// accountFits reports whether the account exists and is of the kind, open or not: a
// pair of an account that closed since the flow last ran is dropped at its next run
// and is not saved, so the audit allows it.
func (w *World) accountFits(id int, k ledger.Kind) bool {
	v, err := w.ledger.Account(id)
	return err == nil && v.Kind == k
}

// audit is the module's part of the audit (spec-fiscal.md 17): every carry names an
// account of the kind the flow acts on, in strictly ascending order and inside its
// denominator. The identities of the money are the ledger's.
func (s *fiscalState) audit(w *World) error {
	for i := range s.flows {
		f := &s.flows[i]
		den := f.carryDen()
		for k, p := range f.carry {
			switch {
			case !w.accountFits(p.account, f.accountKind()):
				return fmt.Errorf("fiscal flow %q: carry for account %d, which is not an account of the %v kind", f.id, p.account, f.accountKind())
			case k > 0 && p.account <= f.carry[k-1].account:
				return fmt.Errorf("fiscal flow %q: carries are not in ascending account order at account %d", f.id, p.account)
			case p.carry < 0 || p.carry >= den:
				return fmt.Errorf("fiscal flow %q: carry %d of account %d is outside [0, %d)", f.id, p.carry, p.account, den)
			}
		}
	}
	return nil
}
