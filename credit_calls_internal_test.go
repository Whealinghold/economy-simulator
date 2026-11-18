package economysim

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

// TestSetRateScales_Errors verifies the checks of spec-credit.md 13.1 in their order,
// each with its sentinel and, for an argument, the path of the entry: the module off
// first, then for the first entry that fails an unknown region, an outside node, a
// scale out of 0 to MaxShockBP, a repeated region (a repeat that comes first beats a
// later error); that a refused call changes no scale and emits nothing; and that a
// scale of 0 (interest-free) and of MaxShockBP are accepted.
//
// Requirement: FR-CRD-060
func TestSetRateScales_Errors(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := creditWorld(t, 1, 0, WithObserver(rec))
	w.outside = []bool{false, true}
	tests := []struct {
		name   string
		scales []RegionScale
		want   error
	}{
		{"unknown region", []RegionScale{{Region: 5, BP: 100}}, ErrUnknownRegion},
		{"negative region", []RegionScale{{Region: -1, BP: 100}}, ErrUnknownRegion},
		{"outside node", []RegionScale{{Region: 1, BP: 100}}, ErrOutsideNode},
		{"above MaxShockBP", []RegionScale{{Region: 0, BP: MaxShockBP + 1}}, ErrOutOfBounds},
		{"below 0", []RegionScale{{Region: 0, BP: -1}}, ErrOutOfBounds},
		{"a region twice", []RegionScale{{Region: 0, BP: 100}, {Region: 0, BP: 200}}, ErrDuplicateID},
		{"the first failure is the one named", []RegionScale{{Region: 5, BP: 100}, {Region: 0, BP: -1}}, ErrUnknownRegion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := w.SetRateScales(tt.scales)
			var ve *ValidationError
			if !errors.Is(err, tt.want) || !errors.As(err, &ve) || len(ve.Path) < 1 || ve.Path[0] == "" {
				t.Errorf("SetRateScales(%v) = %v, want %v with the path of the entry", tt.scales, err, tt.want)
			}
		})
	}
	if s, _ := w.Slots(); len(s.RateScales) != 0 || len(of[CreditRateScaled](rec)) != 0 {
		t.Errorf("after refused calls Slots().RateScales = %v and %d events, want none", s.RateScales, len(of[CreditRateScaled](rec)))
	}
	off := creditConfig(1, 0)
	off.Modules.Credit = nil
	if err := newMill(t, off).SetRateScales(nil); !errors.Is(err, ErrModuleOff) {
		t.Errorf("SetRateScales with the module off = %v, want ErrModuleOff", err)
	}
	if err := w.SetRateScales([]RegionScale{{Region: 0, BP: 0}, {Region: 0, BP: 0}}); !errors.Is(err, ErrDuplicateID) {
		t.Errorf("a region repeated with equal values = %v, want ErrDuplicateID", err)
	}
	if err := w.SetRateScales([]RegionScale{{Region: 0, BP: 0}}); err != nil || w.credit.scale[0] != 0 {
		t.Errorf("a scale of 0 = %v, scale %d, want nil and 0 (interest-free, not a pause)", err, w.credit.scale[0])
	}
}

// TestSetRateScales_EventsAndSlots verifies that a call emits CreditRateScaled once
// per changed region in ascending order, after the whole call, with the tick of the
// next step; that a key set to the value it holds emits nothing; that Slots lists the
// scales that are not 10 000, ascending, and drops a key set back to 10 000; and that
// the configuration hash does not move (the scales are state, not configuration).
//
// Requirement: FR-CRD-060, FR-CRD-012
func TestSetRateScales_EventsAndSlots(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := creditWorld(t, 1, 0, WithObserver(rec))
	step(t, w, 2)
	hash := w.ConfigHash()
	if err := w.SetRateScales([]RegionScale{{Region: 1, BP: 3_000}, {Region: 0, BP: 5_000}}); err != nil {
		t.Fatalf("SetRateScales error = %v, want nil", err)
	}
	want := []CreditRateScaled{{Tick: 2, Region: 0, BeforeBP: 10_000, AfterBP: 5_000}, {Tick: 2, Region: 1, BeforeBP: 10_000, AfterBP: 3_000}}
	if got := of[CreditRateScaled](rec); !slices.Equal(got, want) {
		t.Errorf("CreditRateScaled = %+v, want %+v (ascending region, the tick of the next step)", got, want)
	}
	if s, _ := w.Slots(); !slices.Equal(s.RateScales, []RegionScale{{Region: 0, BP: 5_000}, {Region: 1, BP: 3_000}}) {
		t.Errorf("Slots().RateScales = %v, want regions 0 and 1 at 5 000 and 3 000", s.RateScales)
	}
	if err := w.SetRateScales([]RegionScale{{Region: 0, BP: 5_000}, {Region: 1, BP: 10_000}}); err != nil {
		t.Fatalf("second SetRateScales error = %v, want nil", err)
	}
	if got := of[CreditRateScaled](rec); len(got) != 3 || got[2] != (CreditRateScaled{Tick: 2, Region: 1, BeforeBP: 3_000, AfterBP: 10_000}) {
		t.Errorf("CreditRateScaled after the second call = %+v, want one more, for region 1 only (region 0 kept its value)", got)
	}
	if s, _ := w.Slots(); !slices.Equal(s.RateScales, []RegionScale{{Region: 0, BP: 5_000}}) {
		t.Errorf("Slots().RateScales after clearing region 1 = %v, want region 0 only", s.RateScales)
	}
	if w.ConfigHash() != hash {
		t.Error("ConfigHash moved after SetRateScales, want the scales out of the hash")
	}
}

// TestSetRateScales_AppliesToNewLoansOnly verifies that a scale is read at the next
// round for the loans it forms, and that a claim already formed keeps its rate: a
// loan formed at the posted 600 stays at 600 when the scale falls to 5 000, the next
// loan is formed at half(600 * 5 000 / 10 000) = 300, and with the scale at 0 a
// loan is formed at 0 with no interest.
//
// Requirement: FR-CRD-060, FR-CRD-024
func TestSetRateScales_AppliesToNewLoansOnly(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = askOnce(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	s.plan[hLendX] = offers(CreditOffer{Principal: 5_000, MaxTerm: 90, RateBP: 600})
	w, rec, _ := loanWorld(t, s)
	step(t, w, 1)
	if err := w.SetRateScales([]RegionScale{{Region: 0, BP: 5_000}}); err != nil {
		t.Fatalf("SetRateScales error = %v, want nil", err)
	}
	s.plan[hMillB] = askOnce(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	step(t, w, 1)
	if err := w.SetRateScales([]RegionScale{{Region: 0, BP: 0}}); err != nil {
		t.Fatalf("SetRateScales(0) error = %v, want nil", err)
	}
	s.plan[hMillC] = askOnce(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	step(t, w, 1)
	var rates []int64
	for _, f := range formedOf(rec) {
		rates = append(rates, f.RateBP)
	}
	if !slices.Equal(rates, []int64{600, 300, 0}) {
		t.Errorf("rates of the loans formed = %v, want [600 300 0]", rates)
	}
	if got := formedOf(rec)[2].Interest; got != 0 {
		t.Errorf("interest of the loan formed with the scale at 0 = %d, want 0", got)
	}
	first, ok := w.ledger.Claim(int64(formedOf(rec)[0].Claim))
	if !ok || first.RateBP != 600 {
		t.Errorf("the first claim = %+v (open %v), want its rate still 600 after the scale fell", first, ok)
	}
}

// forgiveWorld is the credit world with five planted claims, in ID order: loans of the
// home weaver to homeA and homeB (role 0, region 0), a loan of the yonder weaver to
// yonderA (role 0, region 1), a loan of homeC to the home weaver (role 1, region 0) and
// a payout claim of the home weaver on homeC (due at 50). The loans are due at 50.
func forgiveWorld(t *testing.T) (*World, *evLog, [5]int64) {
	t.Helper()
	rec := &evLog{}
	w := creditWorld(t, 1, 0, WithObserver(rec), WithAudit())
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 5_000, yonderWeaver: 1_000, homeC: 500})
	ids := [5]int64{
		plantLoan(t, w, homeWeaver, homeA, 400, 40, 50),
		plantLoan(t, w, homeWeaver, homeB, 300, 30, 50),
		plantLoan(t, w, yonderWeaver, yonderA, 200, 20, 50),
		plantLoan(t, w, homeC, homeWeaver, 100, 10, 50),
		plantPayout(t, w, homeWeaver, homeC, 70, 50),
	}
	return w, rec, ids
}

// TestForgiveClaims_Filters verifies the call of spec-credit.md 13.2: each loan whose
// debtor matches a filter at the call is written off with no transfer, in ascending
// claim ID, and the count is returned; a filter by region, by role, by both and "all"
// each select the claims of their debtors; a payout claim is never ended; no money
// moves; a filter that matches nothing is legal and returns 0; and the ledger's lost
// totals rise by what was owed, so the audit holds.
//
// Requirement: FR-CRD-061
func TestForgiveClaims_Filters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		filters []ClaimFilter
		want    []int
	}{
		{"every claim", []ClaimFilter{{AllRegions: true, AllRoles: true}}, []int{0, 1, 2, 3}},
		{"one region", []ClaimFilter{{Region: 0, AllRoles: true}}, []int{0, 1, 3}},
		{"the other region", []ClaimFilter{{Region: 1, AllRoles: true}}, []int{2}},
		{"one role in every region", []ClaimFilter{{Role: 0, AllRegions: true}}, []int{0, 1, 2}},
		{"the other role", []ClaimFilter{{Role: 1, AllRegions: true}}, []int{3}},
		{"a region and a role", []ClaimFilter{{Region: 1, Role: 0}}, []int{2}},
		{"two filters", []ClaimFilter{{Region: 1, AllRoles: true}, {Region: 0, Role: 0}}, []int{0, 1, 2}},
		{"no filter", nil, nil},
		{"a role no debtor of the region holds", []ClaimFilter{{Region: 1, Role: 1}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec, ids := forgiveWorld(t)
			money, before := moneyTotal(w), w.ledger.ClaimTotals()
			var wantIDs []int64
			var lostOwed int64
			for _, i := range tt.want {
				c, _ := w.ledger.Claim(ids[i])
				lostOwed += c.PrincipalOwed + c.InterestOwed
				wantIDs = append(wantIDs, ids[i])
			}
			rec.events = nil
			n, err := w.ForgiveClaims(tt.filters)
			if err != nil || n != int64(len(tt.want)) {
				t.Fatalf("ForgiveClaims(%+v) = %d, %v, want %d, nil", tt.filters, n, err, len(tt.want))
			}
			wantForgivenEvents(t, rec, wantIDs)
			if _, open := w.ledger.Claim(ids[4]); !open {
				t.Error("the payout claim ended, want it never ended by ForgiveClaims")
			}
			totals := w.ledger.ClaimTotals()
			if got := totals.LostPrincipal + totals.LostInterest - before.LostPrincipal - before.LostInterest; got != lostOwed {
				t.Errorf("lost totals rose by %d, want what was owed, %d", got, lostOwed)
			}
			if moneyTotal(w) != money {
				t.Errorf("money total = %d, want %d: forgiveness moves no money", moneyTotal(w), money)
			}
			auditNow(t, w)
		})
	}
}

// wantForgivenEvents fails the test unless the LoanEnded events of the log are those
// of the claims in want, in that order, each of origin loan, with the cause forgiven
// and the tick 1 the next step runs.
func wantForgivenEvents(t *testing.T, rec *evLog, want []int64) {
	t.Helper()
	var ended []int64
	for _, e := range of[LoanEnded](rec) {
		if e.Cause != CauseForgiven || e.Tick != 1 || e.Origin != OriginLoan {
			t.Errorf("LoanEnded %+v, want the cause forgiven, the tick 1 of the next step and origin loan", e)
		}
		ended = append(ended, int64(e.Claim))
	}
	if !slices.Equal(ended, want) {
		t.Errorf("LoanEnded claims = %v, want %v in ascending ID", ended, want)
	}
}

// TestForgiveClaims_Errors verifies the order of the checks of spec-integration.md
// 4.10.1: the module off, then filter by filter an unknown region, an outside node
// (unless AllRegions), an unknown role (unless AllRoles) and a filter that repeats
// another, each leaving every claim open; a field that AllRegions or AllRoles
// overrides is not read; a repeat is judged on what the filters name.
//
// Requirement: FR-CRD-061
func TestForgiveClaims_Errors(t *testing.T) {
	t.Parallel()
	w, rec, _ := forgiveWorld(t)
	w.outside = []bool{false, true}
	tests := []struct {
		name    string
		filters []ClaimFilter
		want    error
	}{
		{"unknown region", []ClaimFilter{{Region: 7, AllRoles: true}}, ErrUnknownRegion},
		{"negative region", []ClaimFilter{{Region: -1, AllRoles: true}}, ErrUnknownRegion},
		{"outside node", []ClaimFilter{{Region: 1, AllRoles: true}}, ErrOutsideNode},
		{"unknown role", []ClaimFilter{{AllRegions: true, Role: 9}}, ErrUnknownRole},
		{"negative role", []ClaimFilter{{AllRegions: true, Role: -1}}, ErrUnknownRole},
		{"a filter twice", []ClaimFilter{{Region: 0, Role: 0}, {Region: 0, Role: 0}}, ErrDuplicateID},
		{"two filters that name everything", []ClaimFilter{{AllRegions: true, AllRoles: true, Region: 3}, {AllRegions: true, AllRoles: true}}, ErrDuplicateID},
		{"the first failure is named", []ClaimFilter{{Region: 7, AllRoles: true}, {AllRegions: true, Role: 9}}, ErrUnknownRegion},
	}
	for _, tt := range tests {
		n, err := w.ForgiveClaims(tt.filters)
		if n != 0 || !errors.Is(err, tt.want) {
			t.Errorf("%s: ForgiveClaims = %d, %v, want 0 and %v", tt.name, n, err, tt.want)
		}
	}
	if w.ledger.OpenClaims() != 5 || len(of[LoanEnded](rec)) != 0 {
		t.Errorf("after refused calls %d claims open and %d LoanEnded, want 5 and 0", w.ledger.OpenClaims(), len(of[LoanEnded](rec)))
	}
	if n, err := w.ForgiveClaims([]ClaimFilter{{AllRegions: true, Region: 9, AllRoles: true}}); err != nil || n != 4 {
		t.Errorf("ForgiveClaims with AllRegions and a region field out of range = %d, %v, want 4, nil (the field is not read)", n, err)
	}
	off := creditConfig(1, 0)
	off.Modules.Credit = nil
	if _, err := newMill(t, off).ForgiveClaims(nil); !errors.Is(err, ErrModuleOff) {
		t.Errorf("ForgiveClaims with the module off = %v, want ErrModuleOff", err)
	}
}

// TestForgiveClaims_RoleAtTheCall verifies that the debtor's role is read at the
// call, not at the formation: a claim whose debtor changed role since is matched by
// the role it has now.
//
// Requirement: FR-CRD-061, FR-CRD-044
func TestForgiveClaims_RoleAtTheCall(t *testing.T) {
	t.Parallel()
	w, _, _ := forgiveWorld(t)
	w.agents[homeA].role = 1
	if n, err := w.ForgiveClaims([]ClaimFilter{{Role: 0, AllRegions: true}}); err != nil || n != 2 {
		t.Errorf("ForgiveClaims(role 0) after agent %d moved to role 1 = %d, %v, want 2 (homeB and yonderA), nil", homeA, n, err)
	}
}

// TestForgiveClaims_ResumesAfterACall verifies that the claims a call removed are not
// in the next ticks: the indexes agree with the list (the audit runs every tick) and
// the forgiven loans are never visited at their due tick, while the payout claim the
// call spared is.
//
// Requirement: FR-CRD-061
func TestForgiveClaims_ResumesAfterACall(t *testing.T) {
	t.Parallel()
	w, rec, ids := forgiveWorld(t)
	if _, err := w.ForgiveClaims([]ClaimFilter{{AllRegions: true, AllRoles: true}}); err != nil {
		t.Fatalf("ForgiveClaims error = %v, want nil", err)
	}
	if err := w.credit.idx.check(w.ledger.AppendClaims(nil)); err != nil {
		t.Errorf("indexes after the call: %v, want them equal to the list", err)
	}
	rec.events = nil
	step(t, w, 50)
	got := of[LoanEnded](rec)
	if len(got) != 1 || got[0].Origin != OriginPayout || got[0].Claim != ClaimID(ids[4]) {
		t.Errorf("LoanEnded after the call = %+v, want only the payout claim %d (the agents of this world are removed on the way, so its cause is the debtor's)", got, ids[4])
	}
}

// TestDirectCredit_Replaces verifies that the Borrow and the Lend of a directive
// replace the request and the offer the fallback appended, leave a plan with none
// untouched, add the directive's when the fallback made none, and keep what the plan
// held before the fallback ran.
//
// Requirement: FR-CRD-055
func TestDirectCredit_Replaces(t *testing.T) {
	t.Parallel()
	pre := CreditRequest{Principal: 1}
	fb := CreditRequest{Principal: 100, Term: 5, MaxRateBP: 10}
	fo := CreditOffer{Principal: 100, MaxTerm: 5, RateBP: 10}
	dr := CreditRequest{Principal: 900, Term: 9, MaxRateBP: 90}
	do := CreditOffer{Principal: 900, MaxTerm: 9, RateBP: 90}
	tests := []struct {
		name  string
		dir   Directive
		fromB []CreditRequest // the fallback's requests
		fromL []CreditOffer
		wantB []CreditRequest
		wantL []CreditOffer
	}{
		{"neither set", Directive{}, []CreditRequest{fb}, []CreditOffer{fo}, []CreditRequest{fb}, []CreditOffer{fo}},
		{"a request replaces the fallback's", Directive{Borrow: &dr}, []CreditRequest{fb}, []CreditOffer{fo}, []CreditRequest{dr}, []CreditOffer{fo}},
		{"an offer replaces the fallback's", Directive{Lend: &do}, []CreditRequest{fb}, []CreditOffer{fo}, []CreditRequest{fb}, []CreditOffer{do}},
		{"added when the fallback made none", Directive{Borrow: &dr, Lend: &do}, nil, nil, []CreditRequest{dr}, []CreditOffer{do}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := TradePlan{Borrow: []CreditRequest{pre}}
			nb, nl := len(p.Borrow), len(p.Lend)
			p.Borrow, p.Lend = append(p.Borrow, tt.fromB...), append(p.Lend, tt.fromL...)
			directCredit(&p, &tt.dir, nb, nl)
			if !slices.Equal(p.Borrow, append([]CreditRequest{pre}, tt.wantB...)) || !slices.Equal(p.Lend, tt.wantL) {
				t.Errorf("plan after directCredit = %+v and %+v, want %+v after the earlier %+v, and %+v", p.Borrow, p.Lend, tt.wantB, pre, tt.wantL)
			}
		})
	}
}

// directWorldCredit is the loan world with a directed external miller (role 0, the
// Borrow block) and an external weaver (role 1, the Lend block) in region 0, funded to
// hold the offer, and the ID of each.
func directWorldCredit(t *testing.T) (w *World, rec *evLog, miller, weaver AgentID) {
	t.Helper()
	w, rec, _ = loanWorld(t, newLoanScript())
	var err error
	if miller, err = w.AddAgent(NewAgent{Region: 0, Role: 0, Money: 1_000}); err != nil {
		t.Fatalf("AddAgent(miller) error = %v, want nil", err)
	}
	if weaver, err = w.AddAgent(NewAgent{Region: 0, Role: 1, Money: 10_000}); err != nil {
		t.Fatalf("AddAgent(weaver) error = %v, want nil", err)
	}
	return w, rec, miller, weaver
}

// TestDirect_CreditFields verifies the credit fields of a directive
// (spec-credit.md 10.5, spec-integration.md 4.3.2 rows 1 and 7): the module off is
// ErrModuleOff, a field out of its bound is ErrOutOfBounds naming it, a Borrow for a
// role with only a Lend block (and the reverse) is ErrNotOnRole, and a refused call
// leaves the mailbox as it was.
//
// Requirement: FR-CRD-055, FR-INT-025
func TestDirect_CreditFields(t *testing.T) {
	t.Parallel()
	w, _, miller, weaver := directWorldCredit(t)
	good := CreditRequest{Principal: 500, Term: 30, MaxRateBP: 1_500}
	goodOffer := CreditOffer{Principal: 2_000, MaxTerm: 90, RateBP: 600}
	tests := []struct {
		name string
		to   AgentID
		d    Directive
		want error
		path string
	}{
		{"principal 0", miller, Directive{Borrow: &CreditRequest{Term: 30}}, ErrOutOfBounds, "principal"},
		{"principal above MaxMoney", miller, Directive{Borrow: &CreditRequest{Principal: MaxMoney + 1, Term: 30}}, ErrOutOfBounds, "principal"},
		{"term 0", miller, Directive{Borrow: &CreditRequest{Principal: 5}}, ErrOutOfBounds, "term"},
		{"term above the bound", miller, Directive{Borrow: &CreditRequest{Principal: 5, Term: MaxCreditTerm + 1}}, ErrOutOfBounds, "term"},
		{"rate above the bound", miller, Directive{Borrow: &CreditRequest{Principal: 5, Term: 5, MaxRateBP: MaxCreditRateBP + 1}}, ErrOutOfBounds, "max_rate_bp"},
		{"offer term 0", weaver, Directive{Lend: &CreditOffer{Principal: 5}}, ErrOutOfBounds, "max_term"},
		{"offer rate below 0", weaver, Directive{Lend: &CreditOffer{Principal: 5, MaxTerm: 5, RateBP: -1}}, ErrOutOfBounds, "rate_bp"},
		{"a Borrow for a lender", weaver, Directive{Borrow: &good}, ErrNotOnRole, "borrow"},
		{"a Lend for a borrower", miller, Directive{Lend: &goodOffer}, ErrNotOnRole, "lend"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := w.Direct(tt.to, tt.d)
			var ve *ValidationError
			if !errors.Is(err, tt.want) || !errors.As(err, &ve) || ve.Path[len(ve.Path)-1] != tt.path {
				t.Errorf("Direct(%+v) = %v, want %v on %q", tt.d, err, tt.want, tt.path)
			}
			if st, _ := w.Agent(tt.to); st.HasDirective {
				t.Error("a refused Direct left a directive in the mailbox, want it as it was")
			}
		})
	}
}

// TestDirect_CreditReachesTheRound verifies that a good directive is copied into the
// mailbox and reaches the round: the directed miller's request is admitted and
// filled by the directed weaver's offer, at the weaver's posted rate, for the term
// the directive asked.
//
// Requirement: FR-CRD-055
func TestDirect_CreditReachesTheRound(t *testing.T) {
	t.Parallel()
	w, rec, miller, weaver := directWorldCredit(t)
	good := CreditRequest{Principal: 500, Term: 30, MaxRateBP: 1_500}
	if err := w.Direct(miller, Directive{Borrow: &good}); err != nil {
		t.Fatalf("Direct(Borrow) error = %v, want nil", err)
	}
	st, _ := w.Agent(miller)
	if box := st.Directive.Borrow; box == nil || *box != good || box == &good {
		t.Errorf("mailbox = %+v, want a copy of the request", st.Directive)
	}
	if err := w.Direct(weaver, Directive{Lend: &CreditOffer{Principal: 2_000, MaxTerm: 90, RateBP: 600}}); err != nil {
		t.Fatalf("Direct(Lend) error = %v, want nil", err)
	}
	step(t, w, 1)
	want := []loanRow{{lender: weaver, borrower: miller, principal: 500, rate: 600}}
	if got := rowsOf(rec); !slices.Equal(got, want) {
		t.Errorf("loans formed = %+v, want %+v from the directed weaver to the directed miller", got, want)
	}
	if f := formedOf(rec); len(f) != 1 || f[0].Term != 30 {
		t.Errorf("LoanFormed = %+v, want one for 30 ticks", f)
	}
}

// TestDirect_CreditFieldsModuleOff verifies row 1 of spec-integration.md 4.3.2: a
// directive that sets Borrow or Lend is ErrModuleOff with the module off, and an empty
// one is not refused.
//
// Requirement: FR-CRD-055, FR-INT-025
func TestDirect_CreditFieldsModuleOff(t *testing.T) {
	t.Parallel()
	c := creditConfig(1, 0)
	c.Modules.Credit = nil
	w := newMill(t, c)
	id, err := w.AddAgent(NewAgent{Region: 0, Role: 0, Money: 100})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	for _, d := range []Directive{{Borrow: &CreditRequest{Principal: 1, Term: 1}}, {Lend: &CreditOffer{Principal: 1, MaxTerm: 1}}} {
		if err := w.Direct(id, d); !errors.Is(err, ErrModuleOff) {
			t.Errorf("Direct(%+v) with the module off = %v, want ErrModuleOff", d, err)
		}
	}
	if err := w.Direct(id, Directive{}); err != nil {
		t.Errorf("Direct(empty) = %v, want nil", err)
	}
}

// TestCredit_AgentState verifies the credit group of AgentState (spec-credit.md 10.1,
// spec-integration.md 4.7): an agent whose role has a block has the figures of its
// claims as debtor and as creditor, a lender has no room and a borrower the room of
// its exposure cap, an agent with no block has no group, and reading it changes no
// saved state and no scratch a later step reads (the checkpoint bytes and the valued
// prices are as they were).
//
// Arithmetic: the loan is 1 000 at 600 for 30 ticks (interest I); the borrower's upkeep
// cost is U = 1 000 and its cap 90 * U, so the room is 90 000 - 1 000 = 89 000.
//
// Requirement: FR-CRD-050, FR-CRD-055
func TestCredit_AgentState(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = askOnce(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	s.plan[hLendX] = offers(CreditOffer{Principal: 5_000, MaxTerm: 90, RateBP: 600})
	w, _, _ := loanWorld(t, s)
	step(t, w, 2)
	interest := interestOf(t, w, 1_000, 600, 30)
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	valued := slices.Clone(w.oc.valued)
	debtor, err := w.Agent(hMillA)
	if err != nil {
		t.Fatalf("Agent(miller) error = %v, want nil", err)
	}
	wantDebtor := CreditState{Debt: 1_000, Owed: 1_000 + interest, NextDue: 30, CreditRoom: 89_000}
	if !debtor.HasCredit || debtor.Credit != wantDebtor {
		t.Errorf("Agent(borrower).Credit = %+v (has %v), want %+v", debtor.Credit, debtor.HasCredit, wantDebtor)
	}
	if u := w.boundaryUpkeep(&w.agents[hMillA]); u != 1_000 {
		t.Errorf("boundaryUpkeep = %d, want 1 000", u)
	}
	lender, _ := w.Agent(hLendX)
	wantLender := CreditState{NextDue: -1, Lent: 1_000, Receivable: 1_000 + interest}
	if !lender.HasCredit || lender.Credit != wantLender {
		t.Errorf("Agent(lender).Credit = %+v (has %v), want %+v", lender.Credit, lender.HasCredit, wantLender)
	}
	again, _ := w.Checkpoint()
	if !reflect.DeepEqual(again, cp) || !slices.Equal(valued, w.oc.valued) {
		t.Error("reading Agent changed the checkpoint or the valued prices, want neither")
	}
}

// TestCredit_AgentStateWithoutABlock verifies that an agent whose role has no credit
// block, or a world with the module off, has no credit group in its state.
//
// Requirement: FR-CRD-050
func TestCredit_AgentStateWithoutABlock(t *testing.T) {
	t.Parallel()
	off := creditConfig(1, 0)
	off.Modules.Credit = nil
	for name, w := range map[string]*World{"a role with no block": creditWorld(t, 1, 0), "the module off": newMill(t, off)} {
		if st, _ := w.Agent(homeA); st.HasCredit || st.Credit != (CreditState{}) {
			t.Errorf("Agent with %s = %+v (has %v), want no credit group", name, st.Credit, st.HasCredit)
		}
	}
}

// TestCredit_Snapshot verifies the rows of spec-credit.md 16.4: Claims holds every
// open claim in ascending claim ID with the fields of the ledger's record, of both
// origins, Credit the seven totals of the ledger, and both are deep copies (a change
// of one snapshot is not in the next).
//
// Requirement: FR-CRD-078
func TestCredit_Snapshot(t *testing.T) {
	t.Parallel()
	w, _, ids := forgiveWorld(t)
	step(t, w, 3)
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	want := claimRowsOf(w)
	if len(want) != 5 || !slices.Equal(snap.Claims, want) {
		t.Errorf("Snapshot().Claims = %+v, want the 5 open claims %+v", snap.Claims, want)
	}
	first := ClaimRow{Claim: ClaimID(ids[0]), Origin: OriginLoan, Creditor: homeWeaver, Debtor: homeA, Principal: 400, Interest: 40,
		PrincipalOwed: 400, InterestOwed: 40, RateBP: 600, Due: 50, Overdue: -1, Formed: 1}
	if snap.Claims[0] != first {
		t.Errorf("first claim row = %+v, want %+v", snap.Claims[0], first)
	}
	if got := snap.Claims[4]; got.Origin != OriginPayout || got.Principal != 70 {
		t.Errorf("last claim row = %+v, want the payout claim of 70", got)
	}
	tot := w.ledger.ClaimTotals()
	wantTot := CreditTotals{Lent: Money(tot.Lent), Awarded: 70, InterestFixed: Money(tot.InterestFixed)}
	if snap.Credit != wantTot {
		t.Errorf("Snapshot().Credit = %+v, want %+v", snap.Credit, wantTot)
	}
	snap.Claims[0].Principal = 1
	if again, _ := w.Snapshot(); again.Claims[0].Principal != 400 {
		t.Error("a change to a returned row is in the next snapshot, want a deep copy")
	}
}

// TestCredit_SnapshotModuleOff verifies that a world with the module off has no claim
// rows and no totals in its snapshot.
//
// Requirement: FR-CRD-078, FR-CRD-001
func TestCredit_SnapshotModuleOff(t *testing.T) {
	t.Parallel()
	off := creditConfig(1, 0)
	off.Modules.Credit = nil
	if s, _ := newMill(t, off).Snapshot(); len(s.Claims) != 0 || s.Credit != (CreditTotals{}) {
		t.Errorf("snapshot with the module off has %+v and %+v, want none", s.Claims, s.Credit)
	}
}

// claimRowsOf is the claim rows of a world built from the ledger's list, by the
// definition of spec-credit.md 16.4, for a test to compare the snapshot with.
func claimRowsOf(w *World) []ClaimRow {
	var rows []ClaimRow
	for _, c := range w.ledger.AppendClaims(nil) {
		rows = append(rows, ClaimRow{
			Claim: ClaimID(c.ID), Origin: originOf(c.Origin), Creditor: AgentID(w.accountView(c.Creditor).Owner),
			Debtor: AgentID(w.accountView(c.Debtor).Owner), Region: RegionID(w.accountView(c.Debtor).Region),
			Principal: Money(c.Principal), Interest: Money(c.Interest), PrincipalOwed: Money(c.PrincipalOwed),
			InterestOwed: Money(c.InterestOwed), RateBP: c.RateBP, Formed: c.Formed, Due: c.Due, Overdue: c.Overdue,
		})
	}
	return rows
}
