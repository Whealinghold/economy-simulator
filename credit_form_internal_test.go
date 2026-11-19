package economysim

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	crd "github.com/Whealinghold/economy-sim/internal/credit"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestCredit_Formation verifies the formation of a loan and the end of a round at the
// claims limit: the first fill forms with Due = tick + term, the next fill meets
// MaxClaims and ends the round with TooMany counted, the later lane is not matched and
// its request is unfilled, and the ledger holds the one loan.
//
// Arithmetic: MaxClaims is 1. Agent 2 (limit 1 500) is filled 1 000 by agent 5 for 30
// ticks; agent 3 (limit 1 400) meets the limit; yonder's request of 1 000 is unmatched.
//
// Requirement: FR-CRD-025, FR-CRD-026
func TestCredit_Formation(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	s.plan[hMillB] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_400})
	s.plan[hLendX] = offers(CreditOffer{Principal: 5_000, MaxTerm: 90, RateBP: 600})
	s.plan[yMillA], s.plan[yLend] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500}), offers(CreditOffer{Principal: 2_000, MaxTerm: 90, RateBP: 600})
	c := loanConfig()
	c.Modules.Credit.MaxClaims = 1
	w, rec, _ := loanWorldFor(t, c, s)
	step(t, w, 1)
	interest := Money(interestOf(t, w, 1_000, 600, 30))
	wantLoan := LoanFormed{Tick: 0, Claim: 0, Creditor: hLendX, Debtor: hMillA, Region: 0, Principal: 1_000, Interest: interest, RateBP: 600, Term: 30, Due: 30}
	if got := formedOf(rec); !slices.Equal(got, []LoanFormed{wantLoan}) {
		t.Errorf("LoanFormed = %+v, want %+v", got, wantLoan)
	}
	home := CreditRound{Tick: 0, Region: 0, Requests: 2, Offers: 1, Formed: 1, Principal: 1_000, Interest: interest, Unfilled: 1_000, TooMany: 1}
	yonder := CreditRound{Tick: 0, Region: 1, Requests: 1, Offers: 1, Unfilled: 1_000}
	if got := of[CreditRound](rec); !slices.Equal(got, []CreditRound{home, yonder}) {
		t.Errorf("CreditRound = %+v, want %+v", got, []CreditRound{home, yonder})
	}
	if got := heldMoney(t, w, hMillA); got != 2_000 {
		t.Errorf("the borrower holds %d, want 2 000", got)
	}
}

// TestCredit_FormationDropsOverflow verifies that a fill whose principal and interest
// pass MaxMoney is dropped and counted as Overflow, and that the ledger and the
// balances are unchanged: the matching goes on.
//
// Requirement: FR-CRD-025
func TestCredit_FormationDropsOverflow(t *testing.T) {
	t.Parallel()
	w, _, _ := loanWorld(t, newLoanScript())
	before := moneyTotal(w)
	rd := &w.credit.round
	rd.rq, rd.of = []loanRequest{{agent: hMillA, region: 0}}, []loanOffer{{agent: hLendX, region: 0}}
	got := w.formLoan(crd.Fill{Request: 0, Offer: 0, Principal: MaxMoney, RateBP: MaxCreditRateBP, Term: MaxCreditTerm})
	if got != crd.Dropped || rd.counts[0].overflow != 1 || w.ledger.OpenClaims() != 0 || moneyTotal(w) != before {
		t.Errorf("formLoan(MaxMoney at the largest rate and term) = %v with %+v, %d claims, want Dropped, one overflow and no change", got, rd.counts[0], w.ledger.OpenClaims())
	}
}

// TestCredit_RateCapReplaced verifies the seam of the rate limit (spec-credit.md
// 5.2.3): a request's limit above the cap in force for its region is replaced by the
// cap and counted, never refused, so the request is admitted and finds no lender at
// the price; a region with no cap and a limit under it are untouched.
//
// Requirement: FR-CRD-028
func TestCredit_RateCapReplaced(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	s.plan[hMillB] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 400})
	s.plan[yMillA] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	w, rec, _ := loanWorld(t, s)
	w.credit.rateCap = func(g RegionID, limit int64) (int64, bool) { return 500, g == 0 && limit > 500 }
	rd := &w.credit.round
	rd.reqs = []subRequest{
		{hMillA, CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500}},
		{hMillB, CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 400}},
		{yMillA, CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500}},
	}
	w.creditPrices()
	w.admitCreditRequests()
	if rd.counts[0].priceControlled != 1 || rd.counts[1].priceControlled != 0 || rd.counts[0].requestsRefused != 0 {
		t.Errorf("counts = %+v and %+v, want one replacement in region 0, none in region 1 and no refusal", rd.counts[0], rd.counts[1])
	}
	limits := []int64{rd.lanes[0].requests[0].RateBP, rd.lanes[0].requests[1].RateBP, rd.lanes[1].requests[0].RateBP}
	if !slices.Equal(limits, []int64{500, 400, 1_500}) {
		t.Errorf("limits in the lanes = %v, want [500 400 1500]", limits)
	}
	rd.reset()
	s.plan[hLendX] = offers(CreditOffer{Principal: 5_000, MaxTerm: 90, RateBP: 600})
	step(t, w, 1)
	if r := of[CreditRound](rec); len(r) != 2 || r[0].Requests != 2 || r[0].RequestsRefused != 0 || r[0].Formed != 0 || w.ledger.OpenClaims() != 0 {
		t.Errorf("CreditRound = %+v, want 2 requests admitted in region 0 and no loan at the cap 500 (the offer posts 600)", r)
	}
}

// TestCredit_NoRenewal verifies that a due claim is paid at P1.7a before a request of
// the same tick is admitted, so a loan repaid on its due tick lets its borrower ask
// again at once, and that a debtor in arrears is refused in_arrears.
//
// Arithmetic: agent 2 holds 5 000 and borrows 500 owing 520, due at tick 1; at tick 1
// its pot is 5 500 - 1 000 = 4 500, the claim ends repaid and its request of tick 1 is
// admitted and filled. Agent 3 holds nothing and owes a payout claim of 300 due at
// tick 1 that the pot 0 cannot pay: it is overdue and its request is refused.
//
// Requirement: FR-CRD-016, FR-CRD-037
func TestCredit_NoRenewal(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	w, rec, _ := loanWorld(t, s)
	setBalances(t, w, map[AgentID]int64{hMillA: 5_000, hLendX: 10_000, hLendY: 5_000})
	plantLoan(t, w, hLendX, hMillA, 500, 20, 1)
	plantPayout(t, w, hLendY, hMillB, 300, 1)
	at1 := func(f func(*TradePlan)) func(*TradePlan) {
		return func(p *TradePlan) {
			if w.Tick() == 1 {
				f(p)
			}
		}
	}
	again := at1(asks(CreditRequest{Principal: 400, Term: 30, MaxRateBP: 1_000}))
	s.plan[hMillA], s.plan[hMillB] = again, again
	s.plan[hLendX] = at1(offers(CreditOffer{Principal: 1_000, MaxTerm: 90, RateBP: 600}))
	step(t, w, 2)
	if got := endCauses(rec); !slices.Equal(got, []string{"2:repaid"}) {
		t.Errorf("LoanEnded = %v, want the loan of agent 2 repaid", got)
	}
	want := CreditRound{Tick: 1, Region: 0, Requests: 1, Offers: 1, RequestsRefused: 1, InArrears: 1, Formed: 1, Principal: 400, Interest: Money(interestOf(t, w, 400, 600, 30))}
	if got := of[CreditRound](rec); !slices.Equal(got, []CreditRound{want}) {
		t.Errorf("CreditRound = %+v, want %+v: one request admitted and filled, agent 3's refused in_arrears", got, want)
	}
	if got := rowsOf(rec); !slices.Equal(got, []loanRow{{hLendX, hMillA, 400, 600}}) {
		t.Errorf("loans formed = %+v, want one new loan to agent 2 at tick 1", got)
	}
}

// endCauses lists the LoanEnded events of the log as debtor:cause.
func endCauses(rec *evLog) []string {
	var out []string
	for _, e := range of[LoanEnded](rec) {
		out = append(out, fmt.Sprintf("%d:%s", e.Debtor, e.Cause))
	}
	return out
}

// TestCredit_EventsPhase3 verifies the order of the events of a tick with a payment and
// a loan: the LoanEnded and CreditSettled of P1.7a, then the LoanFormed events in
// formation order, then CreditRound per region in ascending order, and that a tick with
// no request and no offer emits no CreditRound.
//
// Requirement: FR-CRD-070, FR-CRD-074
func TestCredit_EventsPhase3(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	w, rec, _ := loanWorld(t, s)
	setBalances(t, w, map[AgentID]int64{hMillA: 5_000, hLendX: 10_000, yLend: 10_000})
	plantLoan(t, w, hLendX, hMillA, 500, 20, 1)
	at1 := func(f func(*TradePlan)) func(*TradePlan) {
		return func(p *TradePlan) {
			if w.Tick() == 1 {
				f(p)
			}
		}
	}
	s.plan[hMillA] = at1(asks(CreditRequest{Principal: 400, Term: 30, MaxRateBP: 1_000}))
	s.plan[yMillA] = at1(asks(CreditRequest{Principal: 400, Term: 30, MaxRateBP: 1_000}))
	s.plan[hLendX] = at1(offers(CreditOffer{Principal: 1_000, MaxTerm: 90, RateBP: 600}))
	s.plan[yLend] = at1(offers(CreditOffer{Principal: 1_000, MaxTerm: 90, RateBP: 600}))
	step(t, w, 3)
	order := []int{firstOf[LoanEnded](rec), firstOf[CreditSettled](rec), firstOf[LoanFormed](rec), lastOf[LoanFormed](rec), firstOf[CreditRound](rec)}
	if !slices.IsSorted(order) || slices.Contains(order, -1) {
		t.Errorf("event positions of LoanEnded, CreditSettled, first and last LoanFormed, CreditRound = %v, want them found and in that order", order)
	}
	if got := regionsOf(of[CreditRound](rec)); !slices.Equal(got, []string{"0@1", "1@1"}) {
		t.Errorf("CreditRound regions@ticks = %v, want regions 0 and 1 at tick 1 only", got)
	}
	f := formedOf(rec)
	if len(f) != 2 || f[0].Region >= f[1].Region || f[0].Claim >= f[1].Claim {
		t.Errorf("LoanFormed = %+v, want the home loan, then the yonder loan, in claim order", f)
	}
}

// firstOf and lastOf are the positions of the first and the last event of type T in
// the log, or -1.
func firstOf[T Event](rec *evLog) int {
	return slices.IndexFunc(rec.events, func(e Event) bool { _, ok := e.(T); return ok })
}

func lastOf[T Event](rec *evLog) int {
	last := -1
	for i, e := range rec.events {
		if _, ok := e.(T); ok {
			last = i
		}
	}
	return last
}

// regionsOf lists the CreditRound events as region@tick.
func regionsOf(rounds []CreditRound) []string {
	var out []string
	for _, r := range rounds {
		out = append(out, fmt.Sprintf("%d@%d", r.Region, r.Tick))
	}
	return out
}

// TestCredit_Warns verifies the Warn "claim cap reached" (spec-credit.md 16.2, pattern
// A): a fill dropped for want of room in each of ten consecutive ticks logs one line
// at the tenth tick and another at the twentieth, with the count of claims, and no
// line per fill or per agent.
//
// Requirement: FR-CRD-076
func TestCredit_Warns(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillB] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	s.plan[hLendY] = offers(CreditOffer{Principal: 2_000, MaxTerm: 90, RateBP: 600})
	c := loanConfig()
	c.Modules.Credit.MaxClaims = 1
	w, _, lc := loanWorldFor(t, c, s)
	plantLoan(t, w, hLendX, hMillA, 500, 0, 100)
	for _, tt := range []struct {
		ticks int
		want  int
	}{{9, 0}, {1, 1}, {9, 0}, {1, 1}} {
		step(t, w, tt.ticks)
		lines := lc.take("claim cap reached")
		if len(lines) != tt.want || (tt.want == 1 && (lines[0].attrs["claims"] != 1 || lines[0].level < 4)) {
			t.Errorf("at tick %d: lines = %+v, want %d Warn line with claims = 1", w.Tick(), lines, tt.want)
		}
	}
}

// TestCredit_ObservationRound verifies that the round gives the same claims and the
// same snapshot with and without a logger, an observer and the audit: neither
// changes a result, and nothing builds an event when none is supplied.
//
// Requirement: FR-CRD-075
func TestCredit_ObservationRound(t *testing.T) {
	t.Parallel()
	build := func(observed bool) *World {
		s := newLoanScript()
		s.plan[hMillA] = askOnce(CreditRequest{Principal: 3_000, Term: 3, MaxRateBP: 1_500})
		s.plan[hLendX] = offers(CreditOffer{Principal: 4_000, MaxTerm: 90, RateBP: 600})
		if observed {
			w, _, _ := loanWorld(t, s)
			return w
		}
		w := newMill(t, loanConfig(), WithSeed(exitSeed), WithStrategy("script", s))
		setBalances(t, w, loanBalances)
		return w
	}
	a, b := build(true), build(false)
	step(t, a, 6)
	step(t, b, 6)
	sa, ea := a.Snapshot()
	sb, eb := b.Snapshot()
	if ea != nil || eb != nil || !reflect.DeepEqual(sa, sb) || !slices.Equal(a.ledger.AppendClaims(nil), b.ledger.AppendClaims(nil)) || a.ledger.ClaimTotals() != b.ledger.ClaimTotals() {
		t.Errorf("the observed and the bare world differ: snapshots equal %v, claims %+v and %+v, errors %v and %v",
			reflect.DeepEqual(sa, sb), a.ledger.ClaimTotals(), b.ledger.ClaimTotals(), ea, eb)
	}
	if a.ledger.ClaimTotals().Lent != 3_000 {
		t.Errorf("Lent = %d, want the one loan of 3 000", a.ledger.ClaimTotals().Lent)
	}
}

// TestCredit_RoundCost verifies the cost of the round in counts (spec-credit.md 6.5):
// the tentative fills of a round are at most its requests plus its offers, so the
// matching is linear in the agents with a block and never in the claims or households.
//
// Requirement: NFR-CRD-002
func TestCredit_RoundCost(t *testing.T) {
	s := newLoanScript()
	reqs := []AgentID{hMillA, hMillB, hMillC}
	for i, a := range reqs {
		s.plan[a] = asks(CreditRequest{Principal: 1_500, Term: 30, MaxRateBP: 1_500 - int64(i)*100})
	}
	s.plan[hLendX] = offers(CreditOffer{Principal: 2_000, MaxTerm: 90, RateBP: 600})
	s.plan[hLendY] = offers(CreditOffer{Principal: 2_000, MaxTerm: 90, RateBP: 400})
	w, _, _ := loanWorld(t, s)
	fills := 0
	w.credit.round.form = func(f crd.Fill) crd.Outcome { fills++; return w.formLoan(f) }
	step(t, w, 1)
	if fills == 0 || fills > len(reqs)+2 {
		t.Errorf("tentative fills = %d, want between 1 and requests + offers = %d", fills, len(reqs)+2)
	}
}

// TestCredit_RoundAllocsPerRun verifies that admission, order and matching allocate nothing
// once the scratch has grown, with the default logger and no observer, on a round whose
// offer is above every limit so that no loan forms.
//
// Requirement: NFR-CRD-001
func TestCredit_RoundAllocsPerRun(t *testing.T) {
	quiet := newMill(t, loanConfig(), WithSeed(exitSeed), WithStrategy("script", newLoanScript()))
	setBalances(t, quiet, loanBalances)
	rd := &quiet.credit.round
	round := func() {
		rd.reqs = append(rd.reqs, subRequest{hMillA, CreditRequest{Principal: 500, Term: 30, MaxRateBP: 100}}, subRequest{hMillB, CreditRequest{Principal: 500, Term: 30, MaxRateBP: 50}})
		rd.offs = append(rd.offs, subOffer{hLendX, CreditOffer{Principal: 500, MaxTerm: 90, RateBP: 600}})
		quiet.p33bRound()
	}
	round()
	if n := testing.AllocsPerRun(50, round); n != 0 {
		t.Errorf("a round with no fill allocates %v times, want 0", n)
	}
}

// TestCredit_LoopClosed verifies the end to end of the two positions on a small world:
// a borrower forms a loan at P3.3b, the claim falls due, P1.7a repays it, the claim
// leaves the list, and the totals, the balances and the audit agree.
//
// Arithmetic: 9 000 at 20 000 basis points a year for 5 ticks is halfEven(9 000 *
// 20 000 * 5 / (10 000 * TicksPerYear)); the borrower held 2 000, holds 11 000 after the
// loan and 11 000 - 9 000 - interest after the repayment, the lender 10 000 + interest.
//
// Requirement: FR-CRD-030, FR-CRD-082
func TestCredit_LoopClosed(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = askOnce(CreditRequest{Principal: 9_000, Term: 5, MaxRateBP: 25_000})
	s.plan[hLendX] = offerOnce(CreditOffer{Principal: 9_000, MaxTerm: 90, RateBP: 20_000})
	w, rec, _ := loanWorld(t, s)
	setBalances(t, w, map[AgentID]int64{hMillA: 2_000, hLendX: 10_000})
	total := moneyTotal(w)
	step(t, w, 1)
	if got := heldMoney(t, w, hMillA); got != 11_000 {
		t.Errorf("borrower holds %d after the loan, want 11 000", got)
	}
	step(t, w, 6)
	interest := interestOf(t, w, 9_000, 20_000, 5)
	wantTotals := ledger.ClaimTotals{Lent: 9_000, Repaid: 9_000, InterestPaid: interest, InterestFixed: interest}
	if got := w.ledger.ClaimTotals(); got != wantTotals || w.ledger.OpenClaims() != 0 {
		t.Errorf("totals %+v with %d claims open, want %+v and none open", got, w.ledger.OpenClaims(), wantTotals)
	}
	got := []int64{heldMoney(t, w, hMillA), heldMoney(t, w, hLendX), moneyTotal(w)}
	if want := []int64{2_000 - interest, 10_000 + interest, total}; !slices.Equal(got, want) {
		t.Errorf("borrower, lender and total money = %v, want %v", got, want)
	}
	if got := endCauses(rec); !slices.Equal(got, []string{"2:repaid"}) || of[LoanEnded](rec)[0].Tick != 5 {
		t.Errorf("LoanEnded = %v, want one repaid at tick 5", got)
	}
	auditNow(t, w)
}

// moneyFold rebuilds the money of the households and of the pool of each region from
// the events alone, with the three readings of spec-credit.md 16.1.3 and the money
// an agent's removal passes to the estate: it keeps no configuration.
type moneyFold struct{ households, pool []int64 }

// apply folds one event into the readings.
func (m *moneyFold) apply(e Event) {
	switch e := e.(type) {
	case CreditSettled:
		m.households[e.Region] += int64(e.Received) - int64(e.PrincipalPaid+e.InterestPaid)
		m.pool[e.Region] += int64(e.ReceivedByEstate)
	case LoanEnded:
		if e.Cause != CauseDebtorRemoved {
			return
		}
		paid := int64(e.PaidPrincipal + e.PaidInterest)
		m.households[e.Region] -= paid
		if e.ToEstate {
			m.pool[e.CreditorRegion] += paid
		} else {
			m.households[e.CreditorRegion] += paid
		}
	case AgentBankrupted:
		m.households[e.Region] -= int64(e.Money)
		m.pool[e.Region] += int64(e.Money)
	}
}

// TestCredit_MoneyRebuild verifies that an observer that keeps no configuration
// rebuilds the money of the households and of the pool of every region from the
// events alone through a loan, a payment to an estate and a waterfall: a loan moves no
// money between regions, a payment moves it from the debtors' region to the creditors'
// households or pool, and a waterfall payment from the removed debtor's region.
//
// Arithmetic: agent 2 borrows 3 000 for 3 ticks and agent 3 1 000 for 30 from the two
// lenders; agent 5 is removed, its claims pass to the estate, and agent 2 pays the
// estate at tick 3; agent 3 is removed and its claim is paid from its balance.
//
// Requirement: FR-CRD-079
func TestCredit_MoneyRebuild(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = askOnce(CreditRequest{Principal: 3_000, Term: 3, MaxRateBP: 1_500})
	s.plan[hMillB] = askOnce(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 1_500})
	s.plan[hLendX] = offerOnce(CreditOffer{Principal: 3_000, MaxTerm: 90, RateBP: 600})
	s.plan[hLendY] = offerOnce(CreditOffer{Principal: 1_000, MaxTerm: 90, RateBP: 600})
	w, rec, _ := loanWorld(t, s)
	setBalances(t, w, map[AgentID]int64{hMillA: 2_000, hMillB: 1_000, hLendX: 10_000, hLendY: 10_000, yLend: 10_000})
	fold := newMoneyFold(t, w)
	from := len(rec.events)
	step(t, w, 1)
	for _, id := range []AgentID{hLendX, hMillB} {
		if err := w.removeAgent(&w.agents[id], CauseConsumer); err != nil {
			t.Fatalf("removeAgent(%d) error = %v, want nil", id, err)
		}
	}
	step(t, w, 3)
	for _, e := range rec.events[from:] {
		fold.apply(e)
	}
	wantFold(t, w, fold)
	causes := endCauses(rec)
	if len(of[LoanPassed](rec)) == 0 || len(of[CreditSettled](rec)) == 0 || !slices.Contains(causes, "3:"+string(CauseDebtorRemoved)) {
		t.Errorf("events: %d passed, %d settled, ended %v, want a passing, a payment and the waterfall of agent 3", len(of[LoanPassed](rec)), len(of[CreditSettled](rec)), causes)
	}
}

// wantFold fails the test unless the money the snapshot reports in each region equals
// the money the events rebuild.
func wantFold(t testing.TB, w *World, fold *moneyFold) {
	t.Helper()
	got, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	for i, r := range got.Regions {
		if int64(r.MoneyHouseholds) != fold.households[i] || int64(r.MoneyPool) != fold.pool[i] {
			t.Errorf("region %d holds %d and %d, the events rebuild %d and %d", i, r.MoneyHouseholds, r.MoneyPool, fold.households[i], fold.pool[i])
		}
	}
}

// newMoneyFold starts the readings at the money the snapshot of the world holds.
func newMoneyFold(t testing.TB, w *World) *moneyFold {
	t.Helper()
	base, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	m := &moneyFold{households: make([]int64, len(base.Regions)), pool: make([]int64, len(base.Regions))}
	for i, r := range base.Regions {
		m.households[i], m.pool[i] = int64(r.MoneyHouseholds), int64(r.MoneyPool)
	}
	return m
}

// TestCredit_DefaultRulesEndToEnd verifies the default strategy on both sides of a
// small world with the audit on: the millers ask for the gap of their horizon, the
// weavers offer a share of their money, loans form at the lenders' posted rate with the
// borrowers' term, and the run goes through the due ticks with every audit passing.
//
// Requirement: FR-CRD-051, FR-CRD-052, FR-CRD-082
func TestCredit_DefaultRulesEndToEnd(t *testing.T) {
	t.Parallel()
	c := loanConfig()
	c.Roles[0].Strategy, c.Roles[1].Strategy = "default", "default"
	rec := &evLog{}
	w := newMill(t, c, WithSeed(exitSeed), WithObserver(rec), WithAudit())
	setBalances(t, w, loanBalances)
	step(t, w, 14)
	formed := formedOf(rec)
	if len(formed) == 0 {
		t.Fatal("no loan formed in 14 ticks of a world of default borrowers and lenders, want at least one")
	}
	dur := w.cfg.Recipes[0].Duration
	for _, f := range formed {
		if f.RateBP != 600 || f.Principal < 100 || f.Term < dur+5 || f.Term > dur+6 {
			t.Errorf("loan %+v, want the lender's rate 600, at least the minimum 100 and a term of %d or %d", f, dur+5, dur+6)
		}
	}
	if ended := of[LoanEnded](rec); len(ended) == 0 {
		t.Errorf("no loan ended in 14 ticks, want the first due ticks to be reached (%d formed)", len(formed))
	}
}
