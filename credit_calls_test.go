package economysim_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditRun is a world of pack one with creditLiveDoc, seed 1 and the audit on, and
// the events it has emitted so far (see creditLiveDoc for the probed ticks of the run).
type creditRun struct {
	w      *economysim.World
	events []economysim.Event
}

// newCreditRun builds the run, with a recording observer when observe is set.
func newCreditRun(t *testing.T, observe bool) *creditRun {
	t.Helper()
	r := &creditRun{}
	opts := []economysim.Option{economysim.WithAudit()}
	if observe {
		opts = append(opts, economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) { r.events = append(r.events, e) })))
	}
	r.w = newPackWorld(t, loadPackBytes(t, []byte(creditLiveDoc), packOne...), 1, opts...)
	return r
}

// step runs n steps.
func (r *creditRun) step(t *testing.T, n int) {
	t.Helper()
	for range n {
		if err := r.w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", r.w.Tick(), err)
		}
	}
}

// snapshot reads the snapshot at the boundary.
func (r *creditRun) snapshot(t *testing.T) economysim.Snapshot {
	t.Helper()
	s, err := r.w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	return s
}

// loansFormed returns the LoanFormed events from index from on.
func (r *creditRun) loansFormed(from int) []economysim.LoanFormed {
	var out []economysim.LoanFormed
	for _, e := range r.events[from:] {
		if l, ok := e.(economysim.LoanFormed); ok {
			out = append(out, l)
		}
	}
	return out
}

// loansEnded returns the LoanEnded events from index from on.
func (r *creditRun) loansEnded(from int) []economysim.LoanEnded {
	var out []economysim.LoanEnded
	for _, e := range r.events[from:] {
		if l, ok := e.(economysim.LoanEnded); ok {
			out = append(out, l)
		}
	}
	return out
}

// foldClaims folds the claim rows into the credit group each agent should hold: the
// sums over its rows as debtor (Debt, Owed, Overdue, NextDue) and as creditor (Lent,
// Receivable). CreditRoom is not a fold of the rows and is left 0.
func foldClaims(claims []economysim.ClaimRow) map[economysim.AgentID]economysim.CreditState {
	out := map[economysim.AgentID]economysim.CreditState{}
	group := func(id economysim.AgentID) economysim.CreditState {
		g, ok := out[id]
		if !ok {
			g.NextDue = -1
		}
		return g
	}
	for _, c := range claims {
		owed := int64(c.PrincipalOwed + c.InterestOwed)
		d := group(c.Debtor)
		d.Debt += int64(c.PrincipalOwed)
		d.Owed += owed
		if c.Overdue >= 0 {
			d.Overdue += owed
		}
		if d.NextDue < 0 || c.Due < d.NextDue {
			d.NextDue = c.Due
		}
		out[c.Debtor] = d
		cr := group(c.Creditor)
		cr.Lent += int64(c.PrincipalOwed)
		cr.Receivable += owed
		out[c.Creditor] = cr
	}
	return out
}

// rateScaled returns the CreditRateScaled events so far.
func (r *creditRun) rateScaled() []economysim.CreditRateScaled {
	var out []economysim.CreditRateScaled
	for _, e := range r.events {
		if v, ok := e.(economysim.CreditRateScaled); ok {
			out = append(out, v)
		}
	}
	return out
}

// bearsInterestAt600 reports whether a loan was formed at the posted rate of the cooks
// of creditLiveDoc, 600, with interest.
func bearsInterestAt600(l economysim.LoanFormed) bool { return l.RateBP == 600 && l.Interest > 0 }

// TestCredit_AgentStateMatchesTheClaimRows verifies, through the public API of a
// running world, that the credit group of Agent is the fold of the claim rows of
// Snapshot: for every debtor, Debt, Owed and Overdue are the sums over its rows as
// debtor and NextDue is the earliest due tick, and for every creditor Lent and
// Receivable are the sums over its rows as creditor. The two readers have separate
// code, so a drift between them is a bug in one.
//
// Setup: pack one with creditLiveDoc, seed 1, 100 ticks: 649 open claims, all of
// origin loan (the first fall due at tick 123).
//
// Requirement: FR-CRD-050, FR-CRD-078
func TestCredit_AgentStateMatchesTheClaimRows(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	r := newCreditRun(t, false)
	r.step(t, 100)
	s := r.snapshot(t)
	if len(s.Claims) == 0 {
		t.Fatal("Snapshot() holds no claims at tick 100, want open claims to fold")
	}
	want := foldClaims(s.Claims)
	roles, checked := r.w.Config().Roles, 0
	for id, w := range want {
		st, err := r.w.Agent(id)
		if err != nil {
			t.Fatalf("Agent(%d) error = %v, want nil", id, err)
		}
		if !st.HasCredit {
			// A debtor that changed role since it borrowed has no block now, so no group.
			if roles[st.Role].Credit != nil {
				t.Errorf("Agent(%d) has no credit group, want one: its role %d has a block", id, st.Role)
			}
			continue
		}
		checked++
		got := st.Credit
		if got.CreditRoom < 0 {
			t.Errorf("Agent(%d).Credit.CreditRoom = %d, want 0 or more", id, got.CreditRoom)
		}
		got.CreditRoom = 0 // the room is the exposure cap less the debt, not a fold of the rows
		if got != w {
			t.Errorf("Agent(%d).Credit = %+v, want the fold of the claim rows %+v", id, got, w)
		}
	}
	if checked < len(want)/2 {
		t.Errorf("checked %d of %d agents with claims, want most of them to hold a credit group", checked, len(want))
	}
}

// TestCredit_RateScaleOfZeroMakesNewLoansInterestFree verifies SetRateScales end to
// end against a control: with the scale of the only region set to 0 before the first
// round, every loan of the run is formed at rate 0 with no interest, the call emits one
// CreditRateScaled and lists the scale in Slots, while the unscaled control run forms
// loans at the posted 600 with interest.
//
// Setup: pack one with creditLiveDoc, seed 1, 70 ticks (the first loans form at tick
// 58 and 59).
//
// Requirement: FR-CRD-060, FR-CRD-024
func TestCredit_RateScaleOfZeroMakesNewLoansInterestFree(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	control := newCreditRun(t, true)
	control.step(t, 70)
	if posted := control.loansFormed(0); !slices.ContainsFunc(posted, bearsInterestAt600) {
		t.Fatalf("control run formed %d loans, want some at the posted 600 with interest", len(posted))
	}

	r := newCreditRun(t, true)
	if err := r.w.SetRateScales([]economysim.RegionScale{{Region: 0, BP: 0}}); err != nil {
		t.Fatalf("SetRateScales error = %v, want nil", err)
	}
	r.step(t, 70)
	got := r.loansFormed(0)
	if len(got) == 0 {
		t.Fatal("scaled run formed no loan, want loans at rate 0")
	}
	if i := slices.IndexFunc(got, func(l economysim.LoanFormed) bool { return l.RateBP != 0 || l.Interest != 0 }); i >= 0 {
		t.Errorf("LoanFormed %+v with the scale at 0, want rate 0 and no interest", got[i])
	}
	if want := []economysim.CreditRateScaled{{Tick: 0, Region: 0, BeforeBP: 10_000, AfterBP: 0}}; !slices.Equal(r.rateScaled(), want) {
		t.Errorf("CreditRateScaled = %+v, want %+v", r.rateScaled(), want)
	}
	if s, err := r.w.Slots(); err != nil || !slices.Equal(s.RateScales, []economysim.RegionScale{{Region: 0, BP: 0}}) {
		t.Errorf("Slots().RateScales = %v, %v, want region 0 at 0 (a scale of 0 is held, not cleared)", s.RateScales, err)
	}
}

// TestCredit_ForgiveClaimsOverARun verifies ForgiveClaims through a running world: a
// call between ticks ends every open claim of the region at once with the cause
// forgiven, in ascending claim ID, with the tick of the next step; it moves no money
// (the money of the households and the pool is the same before and after); the claim
// rows are gone and the totals LostPrincipal and LostInterest rise by exactly what was
// owed; the audit holds at the call and for forty steps after; and a second call
// finds nothing and ends 0.
//
// Setup: pack one with creditLiveDoc, seed 1, 100 ticks (649 open claims in the one
// region, none yet due).
//
// Requirement: FR-CRD-061, FR-CRD-071, FR-CRD-083
func TestCredit_ForgiveClaimsOverARun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	r := newCreditRun(t, true)
	r.step(t, 100)
	before := r.snapshot(t)
	owedP, owedI := owedOn(before.Claims)
	mark := len(r.events)
	n, err := r.w.ForgiveClaims([]economysim.ClaimFilter{{Region: 0, AllRoles: true}})
	if err != nil || n != int64(len(before.Claims)) {
		t.Fatalf("ForgiveClaims = %d, %v, want %d (every open claim) and nil", n, err, len(before.Claims))
	}
	ended := r.loansEnded(mark)
	if int64(len(ended)) != n {
		t.Fatalf("LoanEnded at the call = %d events, want %d (one per claim)", len(ended), n)
	}
	if i := firstNotForgiven(ended, 100); i >= 0 {
		t.Fatalf("LoanEnded[%d] = %+v, want cause forgiven at tick 100 with nothing paid, in ascending claim ID", i, ended[i])
	}
	if lostP, lostI := lostBy(ended); lostP != owedP || lostI != owedI {
		t.Errorf("forgiven losses = %d principal and %d interest, want the owed %d and %d", lostP, lostI, owedP, owedI)
	}
	b, a := before.Credit, r.snapshot(t).Credit
	if a.LostPrincipal-b.LostPrincipal != owedP || a.LostInterest-b.LostInterest != owedI {
		t.Errorf("Snapshot().Credit lost rose by %d and %d, want exactly the owed %d and %d", a.LostPrincipal-b.LostPrincipal, a.LostInterest-b.LostInterest, owedP, owedI)
	}
}

// lostBy sums the principal and the interest the events say the creditors lost.
func lostBy(ended []economysim.LoanEnded) (principal, interest economysim.Money) {
	for _, e := range ended {
		principal += e.LostPrincipal
		interest += e.LostInterest
	}
	return principal, interest
}

// TestCredit_ForgiveClaimsLeavesAConsistentWorld verifies that forgiving every claim of
// a run changes no balance (the money of the households and of the pool is the same
// before and after: the creditors' loss is the cost, nothing is created or destroyed),
// that no claim row is left, that the audit holds at the call, that a second call finds nothing and ends 0, and
// that the world runs on for forty steps past the main due tick with no claim left to
// settle and the audit on.
//
// Setup: as TestCredit_ForgiveClaimsOverARun.
//
// Requirement: FR-CRD-061, FR-CRD-083
func TestCredit_ForgiveClaimsLeavesAConsistentWorld(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	r := newCreditRun(t, false)
	r.step(t, 100)
	before := r.snapshot(t).Regions[0]
	if _, err := r.w.ForgiveClaims([]economysim.ClaimFilter{{Region: 0, AllRoles: true}}); err != nil {
		t.Fatalf("ForgiveClaims error = %v, want nil", err)
	}
	snap := r.snapshot(t)
	if len(snap.Claims) != 0 {
		t.Errorf("Snapshot().Claims after the call = %d rows, want none", len(snap.Claims))
	}
	after := snap.Regions[0]
	if after.MoneyHouseholds != before.MoneyHouseholds || after.MoneyPool != before.MoneyPool {
		t.Errorf("money after the call = %d households and %d pool, was %d and %d, want no money moved", after.MoneyHouseholds, after.MoneyPool, before.MoneyHouseholds, before.MoneyPool)
	}
	if err := r.w.Audit(); err != nil {
		t.Errorf("Audit() after the call error = %v, want nil (creditors lose what was owed, identities hold)", err)
	}
	if again, err := r.w.ForgiveClaims([]economysim.ClaimFilter{{AllRegions: true, AllRoles: true}}); err != nil || again != 0 {
		t.Errorf("second ForgiveClaims = %d, %v, want 0 and nil", again, err)
	}
	r.step(t, 40) // WithAudit fails the step on a broken identity
}

// owedOn sums the principal and the interest still owed over the claim rows.
func owedOn(claims []economysim.ClaimRow) (principal, interest economysim.Money) {
	for _, c := range claims {
		principal += c.PrincipalOwed
		interest += c.InterestOwed
	}
	return principal, interest
}

// firstNotForgiven returns the index of the first event that is not a forgiven claim
// ended at the tick with nothing paid, or that does not come after the claim before it
// in claim ID, and -1 when every event is.
func firstNotForgiven(ended []economysim.LoanEnded, tick int64) int {
	for i, e := range ended {
		if e.Cause != economysim.CauseForgiven || e.Tick != tick || e.PaidPrincipal != 0 || e.PaidInterest != 0 {
			return i
		}
		if i > 0 && e.Claim <= ended[i-1].Claim {
			return i
		}
	}
	return -1
}

// TestCredit_CallsDoNotDependOnObservers verifies the non-interference of the credit
// calls: two runs of the same seed, one bare and one with an observer and a debug
// logger, that make the same SetRateScales and ForgiveClaims calls between the same
// ticks, end with the same count forgiven, the same snapshot and the same checkpoint
// bytes. A call that built its events or its log lines from state that an observer
// changes, or skipped a step without one, would diverge here.
//
// Setup: pack one with creditLiveDoc, seed 1; scale 5 000 at tick 62, forgive at tick
// 100, run to 130.
//
// Requirement: FR-CRD-075, FR-CRD-062
func TestCredit_CallsDoNotDependOnObservers(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	run := func(observe bool, opts ...economysim.Option) (int64, economysim.Snapshot, []byte) {
		cfg := loadPackBytes(t, []byte(creditLiveDoc), packOne...)
		if observe {
			opts = append(opts, economysim.WithObserver(economysim.ObserverFunc(func(economysim.Event) {})))
		}
		w := newPackWorld(t, cfg, 1, append(opts, economysim.WithAudit())...)
		r := &creditRun{w: w}
		r.step(t, 62)
		if err := w.SetRateScales([]economysim.RegionScale{{Region: 0, BP: 5_000}}); err != nil {
			t.Fatalf("SetRateScales error = %v, want nil", err)
		}
		r.step(t, 38)
		n, err := w.ForgiveClaims([]economysim.ClaimFilter{{Region: 0, AllRoles: true}})
		if err != nil {
			t.Fatalf("ForgiveClaims error = %v, want nil", err)
		}
		r.step(t, 30)
		b, err := mustCheckpoint(t, w).MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary error = %v, want nil", err)
		}
		return n, r.snapshot(t), b
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	n0, s0, b0 := run(false)
	n1, s1, b1 := run(true, economysim.WithLogger(logger))
	if n0 == 0 || n0 != n1 {
		t.Errorf("claims forgiven = %d bare and %d observed, want the same and above 0", n0, n1)
	}
	if !reflect.DeepEqual(s0, s1) {
		t.Error("snapshot of the observed run differs from the bare run, want identical")
	}
	if !bytes.Equal(b0, b1) {
		t.Errorf("checkpoint of the observed run has %d bytes and differs from the bare run's %d", len(b1), len(b0))
	}
}

// TestDirect_CreditFieldsThroughTheAPI verifies a consumer's path through the public
// API: an external agent of the borrower role is added with money, directed to
// borrow, and the next round forms a loan to it at no more than its ceiling, after
// which its AgentState holds the debt and the snapshot a claim row for it.
//
// Setup: pack one with creditLiveDoc, seed 1, 61 ticks (the cooks have offered since
// tick 58); the grower is the role with the Borrow block.
//
// Requirement: FR-CRD-055, FR-CRD-050
func TestDirect_CreditFieldsThroughTheAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	r := newCreditRun(t, true)
	r.step(t, 61)
	grower, _ := borrowerRoles(r.w.Config().Roles)
	if grower < 0 {
		t.Fatal("pack has no role with a Borrow block, want one")
	}
	id, err := r.w.AddAgent(economysim.NewAgent{Region: 0, Role: grower, Money: 1_000})
	if err != nil {
		t.Fatalf("AddAgent(borrower) error = %v, want nil", err)
	}
	ask := economysim.CreditRequest{Principal: 10_000, Term: 30, MaxRateBP: 1_500}
	if err := r.w.Direct(id, economysim.Directive{Borrow: &ask}); err != nil {
		t.Fatalf("Direct(Borrow) error = %v, want nil", err)
	}
	mark := len(r.events)
	r.step(t, 1)
	formed, debt, dearest := loansTo(r.loansFormed(mark), id)
	if !withinAsk(formed, debt, dearest, ask) {
		t.Errorf("loans formed to agent %d = %d for %d at up to %d, want at least one, at most the %d asked and a rate within %d", id, formed, debt, dearest, ask.Principal, ask.MaxRateBP)
	}
	st, err := r.w.Agent(id)
	if err != nil {
		t.Fatalf("Agent(%d) error = %v, want nil", id, err)
	}
	if got := creditDebt(st); got != int64(debt) {
		t.Errorf("Agent(%d).Credit debt = %d, want the %d just formed (-1 means no credit group)", id, got, debt)
	}
	if !slices.ContainsFunc(r.snapshot(t).Claims, func(c economysim.ClaimRow) bool { return c.Debtor == id }) {
		t.Errorf("Snapshot().Claims has no row for debtor %d, want the loan", id)
	}
}

// withinAsk reports whether n loans for the principal at up to the rate answer the
// request: at least one, for no more than it asked, at no more than its ceiling.
func withinAsk(n int, principal economysim.Money, dearest int64, ask economysim.CreditRequest) bool {
	return n > 0 && int64(principal) <= ask.Principal && dearest <= ask.MaxRateBP
}

// creditDebt is the debt in the credit group of a state, or -1 when it has no group.
func creditDebt(st economysim.AgentState) int64 {
	if !st.HasCredit {
		return -1
	}
	return st.Credit.Debt
}

// loansTo counts the loans formed to a debtor and returns the principal lent to it and
// the highest rate among them.
func loansTo(loans []economysim.LoanFormed, debtor economysim.AgentID) (n int, principal economysim.Money, dearest int64) {
	for _, l := range loans {
		if l.Debtor == debtor {
			n++
			principal += l.Principal
			dearest = max(dearest, l.RateBP)
		}
	}
	return n, principal, dearest
}

// TestDirect_CreditRefusedOnARoleWithNoBlock verifies through the public API that a
// directive with a Borrow or a Lend for an agent whose role has no credit block is
// refused with ErrNotOnRole, and that the refused call leaves the agent with no
// mailbox.
//
// Setup: pack one with creditLiveDoc, seed 1, 61 ticks; an external agent of a role
// that has no block.
//
// Requirement: FR-CRD-055
func TestDirect_CreditRefusedOnARoleWithNoBlock(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	r := newCreditRun(t, false)
	r.step(t, 61)
	_, other := borrowerRoles(r.w.Config().Roles)
	if other < 0 {
		t.Fatal("pack has no role without a credit block, want one")
	}
	bare, err := r.w.AddAgent(economysim.NewAgent{Region: 0, Role: other, Money: 1_000})
	if err != nil {
		t.Fatalf("AddAgent(no block) error = %v, want nil", err)
	}
	for name, d := range map[string]economysim.Directive{
		"Borrow": {Borrow: &economysim.CreditRequest{Principal: 10_000, Term: 30, MaxRateBP: 1_500}},
		"Lend":   {Lend: &economysim.CreditOffer{Principal: 10_000, MaxTerm: 30, RateBP: 600}},
	} {
		if err := r.w.Direct(bare, d); !errors.Is(err, economysim.ErrNotOnRole) {
			t.Errorf("Direct(%s) on a role with no block = %v, want ErrNotOnRole", name, err)
		}
	}
	if st, _ := r.w.Agent(bare); st.HasDirective {
		t.Error("Agent after the refused Direct calls holds a directive, want none")
	}
}

// borrowerRoles returns the last role with a Borrow block and the first role with no
// credit block, or -1 for one that the configuration lacks.
func borrowerRoles(roles []economysim.Role) (borrower, bare economysim.RoleID) {
	borrower, bare = -1, -1
	for i, role := range roles {
		switch {
		case role.Credit != nil && role.Credit.Borrow != nil:
			borrower = economysim.RoleID(i)
		case role.Credit == nil && bare < 0:
			bare = economysim.RoleID(i)
		}
	}
	return borrower, bare
}
