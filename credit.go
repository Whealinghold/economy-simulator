package economysim

import "github.com/Whealinghold/economy-sim/internal/ledger"

// This file is the credit module (spec-credit.md; architecture.md 2.10.1): what
// the module reads of its configuration and its validator. The warnings are in
// credit_report.go. At this stage the module is configured, validated and hashed
// and does nothing: it has no state, no position and no event (plan 3-crd Phase 3);
// its arithmetic is in internal/credit. Outside this file and its continuation the
// module appears only in the catalogue files (config.go, hash.go, bounds.go,
// defaults.go, errors.go, validate.go), in ledger.go (the reasons and the claims
// limit) and at the named hooks of ADR 0010.

// claimsLimit is the most open claims the ledger takes: the module's MaxClaims, and
// 0 for the module off, which makes the ledger refuse any claim (nil-safe, so the
// call in newLedger holds no branch).
func (m *Credit) claimsLimit() int {
	if m == nil {
		return 0
	}
	return int(m.MaxClaims)
}

// creditReasons are the ledger reasons the credit module declares, in the order of
// spec-credit.md 3.3: the loan, the repayment of principal and the payment of
// interest. All three are internal and money only (FR-CRD-002), so none opens a
// boundary account.
func creditReasons() []ledger.Reason {
	return []ledger.Reason{ledger.LoanOut, ledger.LoanRepay, ledger.LoanInterest}
}

// validateCredit checks the credit module and the credit blocks of the roles
// (spec-credit.md 15.1, FR-CRD-010, FR-CRD-011). It runs in three phases, in the
// order of 15.1, and a phase that finds an error ends the run, so the first error
// is the same for the same mistake: the bounds of the module and of every block,
// then the shape of every block (ErrCreditShape), then the dependency (a block with
// the module off). The reason collision is the money flows' rule (moneyflow.go).
// A rate absent from a file is ErrMissingField, reported by the loader.
func (v *validator) validateCredit() {
	steps := []func() bool{v.creditBounds, v.creditShapes, v.creditDependencies}
	for _, step := range steps {
		if !step() {
			return
		}
	}
}

// creditBounds checks the four fields of the module and every field of every block
// against its bound of spec-credit.md 14, and reports whether all are inside.
func (v *validator) creditBounds() bool {
	n := len(v.rep.errs)
	if m := v.c.Modules.Credit; m != nil {
		v.ranges([]string{"modules", "credit"}, []fieldRange{
			{"min_principal", m.MinPrincipal, 1, MaxMoney},
			{"shield_ticks", m.ShieldTicks, 0, MaxCreditTerm},
			{"write_off_ticks", m.WriteOffTicks, 0, MaxCreditTerm},
			{"max_claims", m.MaxClaims, 1, MaxOpenClaims},
		})
	}
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		if ro.Credit != nil {
			v.creditBlockBounds(path([]string{"roles", ro.ID}, "credit"), ro.Credit)
		}
	}
	return len(v.rep.errs) == n
}

// creditBlockBounds checks the fields of the sub-blocks of one role's block.
func (v *validator) creditBlockBounds(base []string, t *CreditTerms) {
	if l := t.Lend; l != nil {
		v.ranges(path(base, "lend"), []fieldRange{
			{"rate_bp", l.RateBP, 0, MaxCreditRateBP},
			{"max_term_ticks", l.MaxTermTicks, 1, MaxCreditTerm},
			{"keep_ticks", l.KeepTicks, 0, MaxCreditTerm},
			{"share_bp", l.ShareBP, 1, wholeBP},
		})
	}
	if b := t.Borrow; b != nil {
		v.ranges(path(base, "borrow"), []fieldRange{
			{"max_rate_bp", b.MaxRateBP, 0, MaxCreditRateBP},
			{"max_debt_ticks", b.MaxDebtTicks, 1, MaxCreditTerm},
			{"slack_ticks", b.SlackTicks, 0, MaxCreditTerm},
		})
	}
}

// creditShapes reports ErrCreditShape for a block with both Lend and Borrow or with
// neither, and whether every block is right.
func (v *validator) creditShapes() bool {
	n := len(v.rep.errs)
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		if t := ro.Credit; t != nil && (t.Lend == nil) == (t.Borrow == nil) {
			v.rep.add(ErrCreditShape, path([]string{"roles", ro.ID}, "credit"), creditShapeValue(t), "1")
		}
	}
	return len(v.rep.errs) == n
}

// creditShapeValue names what a bad block holds, "both" or "neither", for the
// Value of the error.
func creditShapeValue(t *CreditTerms) string {
	if t.Lend != nil {
		return "both"
	}
	return "neither"
}

// creditDependencies reports ErrModuleDependency for a role with a credit block
// while Modules.Credit is nil, and whether every block is right.
func (v *validator) creditDependencies() bool {
	if v.c.Modules.Credit != nil {
		return true
	}
	n := len(v.rep.errs)
	for i := range v.c.Roles {
		if ro := &v.c.Roles[i]; ro.Credit != nil {
			v.rep.add(ErrModuleDependency, path([]string{"roles", ro.ID}, "credit"), "credit block", "modules.credit")
		}
	}
	return len(v.rep.errs) == n
}
