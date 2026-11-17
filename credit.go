package economysim

import "github.com/Whealinghold/economy-sim/internal/ledger"

// This file is the credit module (spec-credit.md; architecture.md 2.10.1): its
// state, what it reads of its configuration and its validator. The continuation
// files are credit_state.go (the derived indexes, the per-agent figures, the
// checkpoint and the audit), credit_settle.go (P1.7a and the removal of an agent's
// claims) and credit_report.go (the warnings, the events and the logs); the
// arithmetic is in internal/credit. The module is built in claims-in-life order
// (plan 3-crd Phases 4 and 5): the settlement, the waterfall, the passing, the
// audit and tag 26 exist, and nothing forms a claim yet. Outside this file and its
// continuation the module appears only in the catalogue files (config.go, hash.go,
// bounds.go, defaults.go, errors.go, validate.go, event.go, tick.go), in ledger.go
// (the reasons and the claims limit) and at the named hooks of ADR 0010.

// credit is the state of the credit module, nil when Config.Modules.Credit is nil.
// The claims are the ledger's list and are saved there (spec-credit.md 17.1); what
// is here is derived from it and rebuilt at a restore (the indexes), scratch (the
// settlement), or the module's own saved state (the scales and the runs).
type credit struct {
	// cfg is the module's configuration of the world's canonical Config.
	cfg *Credit
	// idx is the derived indexes of the claim list (spec-credit.md 17.3).
	idx claimIndex
	// scale is the rate scale of each region in basis points, 10 000 by default
	// (spec-credit.md 13.1). It is state, not configuration, and is not hashed.
	scale []int64
	// overdueRun is the run of the Warn "claims overdue" of each region, and capRun
	// the run of the Warn "claim cap reached" (spec-credit.md 16.2). Both are saved.
	overdueRun []int64
	capRun     int64
	// pass is the scratch of the settlement and of a waterfall.
	pass settleScratch
	// vs is the storage of the credit group of the view, valuedAt the tick plus one at
	// which the prices were last valued for the upkeep cost (0: never), and round the
	// requests, offers and counts of the round at P3.3b. All are scratch, not saved.
	vs       creditScratch
	valuedAt int64
	round    roundScratch
	// rateCap is the rate limit in force in a region, or nil when no module supplies
	// one (spec-credit.md 5.2.3, FR-CRD-028): the ORG seam. It reports the cap and
	// whether the region has one.
	rateCap func(RegionID) (int64, bool)
}

// newCredit builds the state of the module for a canonical configuration, or nil
// when the module is off (a nil receiver is the off state everywhere it is read).
func newCredit(c *Config) *credit {
	if c.Modules.Credit == nil {
		return nil
	}
	n := len(c.Regions)
	cr := &credit{
		cfg:        c.Modules.Credit,
		idx:        newClaimIndex(),
		scale:      make([]int64, n),
		overdueRun: make([]int64, n),
	}
	for i := range cr.scale {
		cr.scale[i] = wholeBP
	}
	cr.pass.rows = make([]settleRow, n)
	cr.round.lanes, cr.round.counts = make([]crdLane, n), make([]creditCount, n)
	return cr
}

// claimsLimit is the most open claims the ledger takes: the module's MaxClaims, and
// 0 for the module off, which makes the ledger refuse any claim (nil-safe, so the
// call in newLedger holds no branch). Validate holds MaxClaims to MaxOpenClaims; the
// cut here keeps the narrowing to int from wrapping on a value it refused, and a
// negative one reaches the ledger, which refuses it.
func (m *Credit) claimsLimit() int {
	if m == nil {
		return 0
	}
	return int(min(m.MaxClaims, MaxOpenClaims))
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
