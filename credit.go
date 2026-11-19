package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

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
	// one (spec-credit.md 5.2.3, FR-CRD-028): the ORG seam. It takes the region of the
	// borrower and the limit of the request and returns the limit the request is admitted
	// with and whether it was lowered to the cap, so the module that sets it can count the
	// replacement and the round's own count of it does not have to outlive the round.
	rateCap func(g RegionID, limit int64) (replaced int64, lowered bool)
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

// SetRateScales sets the scale of the posted rates of regions (the credit module,
// spec-credit.md 13.1): the effective rate of an offer is its rate times the scale,
// read at P3.3b from the next step and for new loans only, since a claim's rate is
// fixed at formation. A scale of 0 makes credit interest-free (it is not a pause) and
// 10 000 clears the key. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for
// the first entry that fails ErrUnknownRegion, ErrOutsideNode, ErrOutOfBounds or
// ErrDuplicateID; a refused call changes nothing. It emits CreditRateScaled for each
// changed region in ascending order, after the whole call is applied. FR-CRD-060.
func (w *World) SetRateScales(scales []RegionScale) error {
	if err := w.enter("SetRateScales", "credit", w.credit != nil); err != nil {
		return err
	}
	keys, err := slotKeys(scales, func(i int, c RegionScale) (int64, error) {
		if err := w.slotRegion(i, c.Region, "region", true); err != nil {
			return 0, err
		}
		return int64(c.Region), slotBP(i, c.BP, 0, MaxShockBP, "MaxShockBP")
	}, "region")
	if err != nil {
		return fmt.Errorf("SetRateScales: %w", err)
	}
	var events []CreditRateScaled
	for _, i := range inKeyOrder(keys) {
		c := scales[i]
		if before := w.credit.scale[c.Region]; before != c.BP {
			w.credit.scale[c.Region] = c.BP
			events = append(events, CreditRateScaled{Tick: w.Tick(), Region: c.Region, BeforeBP: before, AfterBP: c.BP})
		}
	}
	slotNote(w, "rate scales set", events)
	return nil
}

// creditSlots adds the rate scales that are not 10 000 to the slot list, in ascending
// region order; it is empty with the module off.
func (w *World) creditSlots(s *SlotState) {
	if w.credit == nil {
		return
	}
	for g, bp := range w.credit.scale {
		if bp != wholeBP {
			s.RateScales = append(s.RateScales, RegionScale{Region: RegionID(g), BP: bp})
		}
	}
}

// ClaimFilter names the claims to forgive by the debtor's region and role at the call
// (spec-credit.md 13.2). A claim matches when its debtor's region is Region, or
// AllRegions is set, and its debtor's role is Role, or AllRoles is set; a field the
// All flag overrides is ignored.
type ClaimFilter struct {
	// Region is the debtor's region, when AllRegions is false.
	Region RegionID
	// Role is the debtor's role, when AllRoles is false.
	Role RoleID
	// AllRegions and AllRoles match any region and any role.
	AllRegions, AllRoles bool
}

// matches reports whether a debtor in region g with role r is named by the filter.
func (f ClaimFilter) matches(g RegionID, r RoleID) bool {
	return (f.AllRegions || f.Region == g) && (f.AllRoles || f.Role == r)
}

// same reports whether two filters name the same claims, so that a repeat is a
// duplicate whatever a set All flag leaves in the field it overrides.
func (f ClaimFilter) same(o ClaimFilter) bool {
	return f.AllRegions == o.AllRegions && f.AllRoles == o.AllRoles &&
		(f.AllRegions || f.Region == o.Region) && (f.AllRoles || f.Role == o.Role)
}

// ForgiveClaims ends every open claim of origin loan whose debtor matches any filter,
// with no transfer, and returns how many it ended (the credit module, spec-credit.md
// 13.2). A claim of origin payout is never ended by it. Each is written off at the
// call in ascending claim ID with the cause forgiven: no money moves and the
// creditors lose what was owed, so the lost totals of the ledger rise by it. It
// returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the first filter that
// fails ErrUnknownRegion, ErrOutsideNode, ErrUnknownRole or ErrDuplicateID, with 0
// ended and nothing changed; a filter that matches nothing is legal. It emits one
// LoanEnded per claim, at the call, with the Tick the next step runs, and draws
// nothing. FR-CRD-061.
func (w *World) ForgiveClaims(filters []ClaimFilter) (ended int64, err error) {
	if err := w.enter("ForgiveClaims", "credit", w.credit != nil); err != nil {
		return 0, err
	}
	if err := w.checkClaimFilters(filters); err != nil {
		return 0, fmt.Errorf("ForgiveClaims: %w", err)
	}
	t := w.Tick()
	for _, c := range w.ledger.AppendClaims(nil) {
		if c.Origin != ledger.OriginLoan || !w.forgiven(&c, filters) {
			continue
		}
		lp, li, werr := w.ledger.WriteOff(c.ID)
		mustClaim(werr, "the forgiveness")
		w.credit.idx.remove(c)
		w.reportLoanEnded(t, &c, endRec{cause: CauseForgiven, lostP: lp, lostI: li})
		ended++
	}
	return ended, nil
}

// checkClaimFilters is the checks of spec-integration.md 4.10.1, filter by filter in
// the order given: the region (known, and not an outside node unless AllRegions),
// the role (known unless AllRoles), and that no earlier filter names the same claims.
func (w *World) checkClaimFilters(filters []ClaimFilter) error {
	for i, f := range filters {
		if err := w.filterRegion(i, f); err != nil {
			return err
		}
		if !f.AllRoles && (f.Role < 0 || int(f.Role) >= len(w.cfg.Roles)) {
			return argError(ErrUnknownRole, itoa(int(f.Role)), "", entry("filters", i), "role")
		}
		for _, earlier := range filters[:i] {
			if earlier.same(f) {
				return argError(ErrDuplicateID, fmt.Sprintf("%+v", f), "", entry("filters", i))
			}
		}
	}
	return nil
}

// filterRegion is the check of the region of filter i: known, and not an outside node,
// unless AllRegions says the field is not read.
func (w *World) filterRegion(i int, f ClaimFilter) error {
	switch {
	case f.AllRegions:
		return nil
	case f.Region < 0 || int(f.Region) >= len(w.cfg.Regions):
		return argError(ErrUnknownRegion, itoa(int(f.Region)), "", entry("filters", i), "region")
	case w.isOutsideNode(f.Region):
		return argError(ErrOutsideNode, itoa(int(f.Region)), "", entry("filters", i), "region")
	}
	return nil
}

// forgiven reports whether the debtor of claim c, at the call, matches a filter.
func (w *World) forgiven(c *ledger.Claim, filters []ClaimFilter) bool {
	v := w.accountView(c.Debtor)
	role := w.agents[v.Owner].role
	for _, f := range filters {
		if f.matches(RegionID(v.Region), role) {
			return true
		}
	}
	return false
}

// cloneCredit is a copy of the Borrow and Lend of a directive, so that the mailbox
// shares no pointer with the caller (nil for a field not set).
func (d *Directive) cloneCredit() (b *CreditRequest, l *CreditOffer) {
	if d.Borrow != nil {
		c := *d.Borrow
		b = &c
	}
	if d.Lend != nil {
		c := *d.Lend
		l = &c
	}
	return b, l
}

// checkCreditModule is row 1 of spec-integration.md 4.3.2 for the credit fields: a
// directive that sets Borrow or Lend needs the module on (ErrModuleOff).
func (w *World) checkCreditModule(d *Directive) error {
	if (d.Borrow != nil || d.Lend != nil) && w.cfg.Modules.Credit == nil {
		return fmt.Errorf("module credit: %w", ErrModuleOff)
	}
	return nil
}

// checkCreditDirective is row 7 of spec-integration.md 4.3.2 (spec-credit.md 10.5):
// the Borrow and the Lend of a directive are inside their bounds (ErrOutOfBounds),
// and the agent's role has a block of the kind it asks for (ErrNotOnRole). Whether
// the request is below MinPrincipal or over the room is the round's admission, a
// counted refusal, so it is not checked here.
func (w *World) checkCreditDirective(a *agent, d *Directive) error {
	if r := d.Borrow; r != nil {
		if err := directBounds("borrow", r.Principal, r.Term, r.MaxRateBP, "term", "max_rate_bp"); err != nil {
			return err
		}
	}
	if o := d.Lend; o != nil {
		if err := directBounds("lend", o.Principal, o.MaxTerm, o.RateBP, "max_term", "rate_bp"); err != nil {
			return err
		}
	}
	return w.directedBlock(a, d)
}

// directedBlock reports ErrNotOnRole for a Borrow or a Lend of a directive that the
// agent's role has no block for.
func (w *World) directedBlock(a *agent, d *Directive) error {
	t := w.cfg.Roles[a.role].Credit
	switch {
	case d.Borrow != nil && (t == nil || t.Borrow == nil):
		return argError(ErrNotOnRole, itoa(int(a.role)), "", "d", "borrow")
	case d.Lend != nil && (t == nil || t.Lend == nil):
		return argError(ErrNotOnRole, itoa(int(a.role)), "", "d", "lend")
	}
	return nil
}

// directBounds checks the three figures of a request or an offer of a directive, the
// principal, the term and the rate, against MaxMoney, MaxCreditTerm and
// MaxCreditRateBP; field names the two fields after the principal in the path.
func directBounds(field string, principal, term, rate int64, termName, rateName string) error {
	for _, f := range [...]struct {
		name, bound string
		v, lo, hi   int64
	}{
		{"principal", "MaxMoney", principal, 1, MaxMoney},
		{termName, "MaxCreditTerm", term, 1, MaxCreditTerm},
		{rateName, "MaxCreditRateBP", rate, 0, MaxCreditRateBP},
	} {
		if f.v < f.lo || f.v > f.hi {
			return argError(ErrOutOfBounds, itoa(int(f.v)), f.bound, "d", field, f.name)
		}
	}
	return nil
}

// CreditState is the claims of one agent at the tick boundary, in its AgentState
// (spec-credit.md 10.1 without the block, spec-integration.md 4.7): what it owes as a
// debtor and is owed as a creditor, read from the claim list.
type CreditState struct {
	// Debt is the principal owed as debtor, Owed the principal and interest owed,
	// Overdue the owed on the overdue claims and NextDue the earliest due tick of its
	// claims as debtor (-1 when it has none).
	Debt, Owed, Overdue, NextDue int64
	// CreditRoom is the principal the agent may still ask for under its exposure cap,
	// 0 for a lender.
	CreditRoom int64
	// Lent is the principal owed to the agent as creditor and Receivable the principal
	// and interest owed to it.
	Lent, Receivable int64
}

// fillCreditState sets the credit group of the state of agent a: set only with the
// module on and a role with a block. It reads the indexes and values the upkeep at
// the prices of the boundary into a local slice, never into the scratch the next
// step reads (w.oc.valued is what P1.6 prices relief with), so a call to Agent
// changes no later result.
func (w *World) fillCreditState(st *AgentState, a *agent) {
	t := w.cfg.Roles[a.role].Credit
	if w.credit == nil || t == nil {
		return
	}
	f := w.figuresOf(a.account)
	st.HasCredit = true
	st.Credit = CreditState{Debt: f.Debt, Owed: f.Owed, Overdue: f.Overdue, NextDue: f.NextDue, Lent: f.Lent, Receivable: f.Receivable}
	if t.Borrow != nil {
		st.Credit.CreditRoom = w.creditRoomAt(t.Borrow, w.boundaryUpkeep(a), f.Debt)
	}
}

// boundaryUpkeep is the upkeep cost U of an agent at the prices of the boundary, as
// creditUpkeep is at those of a tick, for a reader between steps: it values the
// prices of its region into a copy and leaves the world's scratch alone.
func (w *World) boundaryUpkeep(a *agent) int64 {
	g, nc := w.regionOf(a), len(w.cfg.Commodities)
	prices := make([]int64, nc)
	for c := range prices {
		i := int(g)*nc + c
		if w.markets.windows[i].Len() > 0 {
			prices[c] = int64(w.markets.reports[i].Price)
		} else {
			prices[c] = int64(w.refs[i])
		}
	}
	return w.welfare.cost.tick(w.roles.row(a.role).upkeepIn(g), prices, a.weight)
}

// saveDirectiveCredit writes the Borrow and the Lend of a mailbox, after the fields
// every directive has, in the record of an agent of a world with the module on
// (spec-checkpoint.md 4.2 tag 3; no byte is written with the module off, so the
// bytes of format 1 do not move): a flag for each, then the principal, the term and
// the rate. A negative amount is written as the unsigned number it converts to, and
// the restore refuses it.
func saveDirectiveCredit(cw *codec.Writer, d *Directive) {
	cw.B(d.Borrow != nil)
	if r := d.Borrow; r != nil {
		cw.U(uint64(r.Principal))
		cw.U(uint64(r.Term))
		cw.U(uint64(r.MaxRateBP))
	}
	cw.B(d.Lend != nil)
	if o := d.Lend; o != nil {
		cw.U(uint64(o.Principal))
		cw.U(uint64(o.MaxTerm))
		cw.U(uint64(o.RateBP))
	}
}

// loadDirectiveCredit reads saveDirectiveCredit's bytes into d, each amount checked
// against its upper bound (the lower bounds and the role's block are checked with
// the rules of Direct after the record is read).
func loadDirectiveCredit(r *codec.Reader, d *Directive) {
	if r.B() {
		d.Borrow = &CreditRequest{
			Principal: r.Bounded("principal", MaxMoney), Term: r.Bounded("term", MaxCreditTerm), MaxRateBP: r.Bounded("rate", MaxCreditRateBP),
		}
	}
	if r.B() {
		d.Lend = &CreditOffer{
			Principal: r.Bounded("principal", MaxMoney), MaxTerm: r.Bounded("term", MaxCreditTerm), RateBP: r.Bounded("rate", MaxCreditRateBP),
		}
	}
}
