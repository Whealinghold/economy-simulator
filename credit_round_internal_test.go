package economysim

import (
	"slices"
	"testing"

	crd "github.com/Whealinghold/economy-sim/internal/credit"
)

// loanTerms are a request and an offer the checks of 5.2 accept.
var (
	goodRequest = CreditRequest{Principal: 500, Term: 30, MaxRateBP: 1_000}
	goodOffer   = CreditOffer{Principal: 500, MaxTerm: 90, RateBP: 600}
)

// asks makes a script that appends the request.
func asks(r CreditRequest) func(*TradePlan) {
	return func(p *TradePlan) { p.Borrow = append(p.Borrow, r) }
}

// offers makes a script that appends the offer.
func offers(o CreditOffer) func(*TradePlan) {
	return func(p *TradePlan) { p.Lend = append(p.Lend, o) }
}

// askOnce and offerOnce make scripts that submit at the first tick only.
func askOnce(r CreditRequest) func(*TradePlan) {
	done := false
	return func(p *TradePlan) {
		if !done {
			done = true
			asks(r)(p)
		}
	}
}

func offerOnce(o CreditOffer) func(*TradePlan) {
	done := false
	return func(p *TradePlan) {
		if !done {
			done = true
			offers(o)(p)
		}
	}
}

// refusalCase is one row of the table of spec-credit.md 5.2: the request or offer an
// agent submits and the reason of its refusal.
type refusalCase struct {
	name   string
	agent  AgentID
	req    []CreditRequest
	off    []CreditOffer
	reason Reason
}

func (c refusalCase) msg() string {
	if len(c.off) > 0 {
		return "offer refused"
	}
	return "request refused"
}

// refusalCases are the rows of 5.2 in order, and the cases where two checks fail and
// the earlier one is the reason.
func refusalCases() []refusalCase {
	req := func(f func(*CreditRequest)) []CreditRequest { r := goodRequest; f(&r); return []CreditRequest{r} }
	off := func(f func(*CreditOffer)) []CreditOffer { o := goodOffer; f(&o); return []CreditOffer{o} }
	return []refusalCase{
		{"not a borrower", hLendX, req(func(*CreditRequest) {}), nil, ReasonNotBorrower},
		{"not a borrower beats a bad principal", hLendX, req(func(r *CreditRequest) { r.Principal, r.Term = 1, 0 }), nil, ReasonNotBorrower},
		{"principal below the minimum", hMillA, req(func(r *CreditRequest) { r.Principal = 99 }), nil, ReasonInvalidPrincipal},
		{"principal above MaxMoney", hMillA, req(func(r *CreditRequest) { r.Principal = MaxMoney + 1 }), nil, ReasonInvalidPrincipal},
		{"a bad principal beats a bad term", hMillA, req(func(r *CreditRequest) { r.Principal, r.Term = 1, 0 }), nil, ReasonInvalidPrincipal},
		{"term below 1", hMillA, req(func(r *CreditRequest) { r.Term = 0 }), nil, ReasonInvalidTerm},
		{"term above MaxCreditTerm", hMillA, req(func(r *CreditRequest) { r.Term = MaxCreditTerm + 1 }), nil, ReasonInvalidTerm},
		{"a bad term beats a bad rate", hMillA, req(func(r *CreditRequest) { r.Term, r.MaxRateBP = 0, -1 }), nil, ReasonInvalidTerm},
		{"rate below 0", hMillA, req(func(r *CreditRequest) { r.MaxRateBP = -1 }), nil, ReasonInvalidRate},
		{"rate above MaxCreditRateBP", hMillA, req(func(r *CreditRequest) { r.MaxRateBP = MaxCreditRateBP + 1 }), nil, ReasonInvalidRate},
		{"a second request", hMillA, []CreditRequest{goodRequest, goodRequest}, nil, ReasonDuplicate},
		{"above the exposure room", hMillA, req(func(r *CreditRequest) { r.Principal = 90_001 }), nil, ReasonExposure},
		{"not a lender", hMillA, nil, off(func(*CreditOffer) {}), ReasonNotLender},
		{"offer principal below the minimum", hLendX, nil, off(func(o *CreditOffer) { o.Principal = 99 }), ReasonInvalidPrincipal},
		{"offer term below 1", hLendX, nil, off(func(o *CreditOffer) { o.MaxTerm = 0 }), ReasonInvalidTerm},
		{"offer rate below 0", hLendX, nil, off(func(o *CreditOffer) { o.RateBP = -1 }), ReasonInvalidRate},
		{"a second offer", hLendX, nil, []CreditOffer{goodOffer, goodOffer}, ReasonDuplicate},
		{"offer above the available money", hLendX, nil, off(func(o *CreditOffer) { o.Principal = 10_001 }), ReasonInsufficientFunds},
	}
}

// TestCredit_Admission_Order verifies the table of spec-credit.md 5.2 one row at a
// time: a request or an offer that fails a check is refused whole with the first
// failing reason, counted in CreditRound and logged at Debug, forms no loan and is
// no error of Step; a second request of an agent is refused duplicate and the first
// is kept.
//
// Arithmetic: every upkeep cost is 1 000 and MaxDebtTicks 90, so the room is 90 000
// and 90 001 passes it by 1; the lender holds 10 000 and no order reserves any, so
// 10 001 passes its available money by 1.
//
// Requirement: FR-CRD-013, FR-CRD-014, FR-CRD-016, FR-CRD-017, FR-CRD-018
func TestCredit_Admission_Order(t *testing.T) {
	t.Parallel()
	for _, tt := range refusalCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newLoanScript()
			s.plan[tt.agent] = func(p *TradePlan) {
				p.Borrow, p.Lend = append(p.Borrow, tt.req...), append(p.Lend, tt.off...)
			}
			w, rec, lc := loanWorld(t, s)
			step(t, w, 1)
			lines := lc.take(tt.msg())
			if len(lines) != 1 || lines[0].strs["reason"] != string(tt.reason) || lines[0].attrs["agent"] != int64(tt.agent) {
				t.Errorf("%q lines = %+v, want one for agent %d with reason %s", tt.msg(), lines, tt.agent, tt.reason)
			}
			if n := w.ledger.OpenClaims(); n != 0 {
				t.Errorf("open claims = %d, want 0 (nothing forms with no counterpart)", n)
			}
			wantRefusalCounted(t, tt, of[CreditRound](rec))
		})
	}
}

// wantRefusalCounted fails the test unless the one CreditRound of region 0 counts the
// one refusal of the case: the duplicate keeps the first of its two, and the exposure
// and arrears refusals are also counted in their own subset.
func wantRefusalCounted(t *testing.T, tt refusalCase, rounds []CreditRound) {
	t.Helper()
	if len(rounds) != 1 {
		t.Fatalf("CreditRound events = %+v, want one", rounds)
	}
	r := rounds[0]
	want := CreditRound{Tick: 0, Region: 0}
	if len(tt.off) > 0 {
		want.OffersRefused = 1
	} else {
		want.RequestsRefused = 1
	}
	if tt.reason == ReasonDuplicate && len(tt.off) > 0 {
		want.Offers = 1
	}
	if tt.reason == ReasonDuplicate && len(tt.req) > 0 {
		want.Requests, want.Unfilled = 1, Money(goodRequest.Principal)
	}
	if tt.reason == ReasonExposure {
		want.Exposure = 1
	}
	if r != want {
		t.Errorf("CreditRound = %+v, want %+v", r, want)
	}
}

// TestCredit_ExposureNotClamped verifies that a request one above its room is
// refused whole, not cut to the room, and one equal to the room is admitted whole.
//
// Requirement: FR-CRD-015
func TestCredit_ExposureNotClamped(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		principal                     int64
		wantRequests, wantRefused, un int64
	}{{90_000, 1, 0, 90_000}, {90_001, 0, 1, 0}} {
		s := newLoanScript()
		s.plan[hMillA] = asks(CreditRequest{Principal: tt.principal, Term: 30, MaxRateBP: 1_000})
		w, rec, _ := loanWorld(t, s)
		step(t, w, 1)
		r := of[CreditRound](rec)
		if len(r) != 1 || r[0].Requests != tt.wantRequests || r[0].RequestsRefused != tt.wantRefused || r[0].Unfilled != Money(tt.un) {
			t.Errorf("principal %d: CreditRound = %+v, want %d admitted, %d refused and %d unfilled", tt.principal, r, tt.wantRequests, tt.wantRefused, tt.un)
		}
	}
}

// TestCredit_PlanAtMostOne verifies that a plan with two requests and two offers is
// kept whole for the round, which admits the first of each and refuses the second as
// duplicate: the plan check refuses nothing.
//
// Requirement: FR-CRD-017
func TestCredit_PlanAtMostOne(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = func(p *TradePlan) {
		asks(goodRequest)(p)
		asks(CreditRequest{Principal: 200, Term: 10, MaxRateBP: 900})(p)
	}
	s.plan[hLendX] = func(p *TradePlan) { offers(goodOffer)(p); offers(goodOffer)(p) }
	w, rec, _ := loanWorld(t, s)
	step(t, w, 1)
	claims := w.ledger.AppendClaims(nil)
	r := of[CreditRound](rec)
	if len(claims) != 1 || claims[0].Principal != 500 || len(r) != 1 || r[0].RequestsRefused != 1 || r[0].OffersRefused != 1 || r[0].Requests != 1 || r[0].Offers != 1 {
		t.Errorf("claims = %+v, rounds = %+v, want the first request filled by the first offer and the second of each refused", claims, r)
	}
}

// TestCredit_RoundMovesOnlyAvailable verifies that an offer is admitted against the
// lender's balance less what its orders of the tick reserved: an offer of the whole
// balance is refused insufficient_funds, and one of what is left forms its loan.
//
// Arithmetic: the lender holds 10 000 and bids 2 grain at 1 000, so 2 000 is reserved
// and 8 000 is available.
//
// Requirement: FR-CRD-020
func TestCredit_RoundMovesOnlyAvailable(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		offer     int64
		wantLoans int
	}{{10_000, 0}, {8_000, 1}} {
		s := newLoanScript()
		s.plan[hLendX] = func(p *TradePlan) {
			p.Intents = append(p.Intents, Intent{Commodity: 1, Side: Bid, Quantity: 2, Limit: 1_000, Purpose: PurposeUpkeep})
			offers(CreditOffer{Principal: tt.offer, MaxTerm: 90, RateBP: 600})(p)
		}
		s.plan[hMillA] = asks(CreditRequest{Principal: 8_000, Term: 30, MaxRateBP: 1_000})
		w, _, _ := loanWorld(t, s)
		step(t, w, 1)
		if n := w.ledger.OpenClaims(); n != tt.wantLoans {
			t.Errorf("offer %d: open claims = %d, want %d", tt.offer, n, tt.wantLoans)
		}
	}
}

// formedOf returns the LoanFormed events of the log, in order.
func formedOf(rec *evLog) []LoanFormed { return of[LoanFormed](rec) }

// loanRow is a formed loan in the terms of the tests.
type loanRow struct {
	lender, borrower AgentID
	principal, rate  int64
}

func rowsOf(rec *evLog) []loanRow {
	var out []loanRow
	for _, f := range formedOf(rec) {
		out = append(out, loanRow{f.Creditor, f.Debtor, int64(f.Principal), f.RateBP})
	}
	return out
}

// TestCredit_RoundOrder verifies the order of the round: the lanes in ascending
// region, the offers by effective rate ascending and the requests by limit
// descending, so the borrower that pays most is served first and the one whose limit
// is below every posted rate gets nothing.
//
// Arithmetic: home lenders post 600 (agent 5, 4 000) and 400 (agent 6, 4 000); the
// requests are agent 2 for 3 000 at 1 500, agent 3 for 3 000 at 500 and agent 4 for
// 3 000 at 300. Agent 2 is served first by the 400 lender (3 000), agent 3 by what is
// left of it, 1 000, and not by the 600 lender above its limit, and agent 4's limit 300
// is below both rates: unfilled 2 000 + 3 000 = 5 000. Yonder is its own lane.
//
// Requirement: FR-CRD-021, FR-CRD-022
func TestCredit_RoundOrder(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	req := func(p, limit int64) func(*TradePlan) {
		return asks(CreditRequest{Principal: p, Term: 30, MaxRateBP: limit})
	}
	s.plan[hMillA], s.plan[hMillB], s.plan[hMillC] = req(3_000, 1_500), req(3_000, 500), req(3_000, 300)
	s.plan[hLendX] = offers(CreditOffer{Principal: 4_000, MaxTerm: 90, RateBP: 600})
	s.plan[hLendY] = offers(CreditOffer{Principal: 4_000, MaxTerm: 90, RateBP: 400})
	s.plan[yMillA], s.plan[yLend] = req(1_000, 1_500), offers(CreditOffer{Principal: 2_000, MaxTerm: 90, RateBP: 600})
	w, rec, _ := loanWorld(t, s)
	step(t, w, 1)
	want := []loanRow{{hLendY, hMillA, 3_000, 400}, {hLendY, hMillB, 1_000, 400}, {yLend, yMillA, 1_000, 600}}
	if got := rowsOf(rec); !slices.Equal(got, want) {
		t.Errorf("loans formed = %+v, want %+v", got, want)
	}
	home := CreditRound{Tick: 0, Region: 0, Requests: 3, Offers: 2, Formed: 2, Principal: 4_000, Unfilled: 5_000}
	home.Interest = Money(interestOf(t, w, 3_000, 400, 30) + interestOf(t, w, 1_000, 400, 30))
	yonder := CreditRound{Tick: 0, Region: 1, Requests: 1, Offers: 1, Formed: 1, Principal: 1_000, Interest: Money(interestOf(t, w, 1_000, 600, 30))}
	if got := of[CreditRound](rec); !slices.Equal(got, []CreditRound{home, yonder}) {
		t.Errorf("CreditRound events = %+v, want %+v", got, []CreditRound{home, yonder})
	}
}

// interestOf is the interest of a loan by the arithmetic package at the world's calendar.
func interestOf(t testing.TB, w *World, x, rate, term int64) int64 {
	t.Helper()
	i, ok := crd.Interest(x, rate, term, w.cfg.Calendar.TicksPerYear())
	if !ok {
		t.Fatalf("Interest(%d, %d, %d) is not ok, want a value", x, rate, term)
	}
	return i
}

// TestCredit_DiscriminatoryFill verifies that each fill is at its lender's own posted
// rate: one request of 5 000 is filled 4 000 by the cheaper lender at 400 and 1 000
// by the other at 600, each claim carrying its own rate and interest.
//
// Requirement: FR-CRD-023
func TestCredit_DiscriminatoryFill(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = asks(CreditRequest{Principal: 5_000, Term: 30, MaxRateBP: 1_500})
	s.plan[hLendX] = offers(CreditOffer{Principal: 4_000, MaxTerm: 90, RateBP: 600})
	s.plan[hLendY] = offers(CreditOffer{Principal: 4_000, MaxTerm: 90, RateBP: 400})
	w, rec, _ := loanWorld(t, s)
	step(t, w, 1)
	want := []loanRow{{hLendY, hMillA, 4_000, 400}, {hLendX, hMillA, 1_000, 600}}
	if got := rowsOf(rec); !slices.Equal(got, want) {
		t.Errorf("loans formed = %+v, want %+v", got, want)
	}
	claims := w.ledger.AppendClaims(nil)
	if len(claims) != 2 || claims[0].Interest != interestOf(t, w, 4_000, 400, 30) || claims[1].Interest != interestOf(t, w, 1_000, 600, 30) {
		t.Errorf("claims = %+v, want interest at each lender's own rate", claims)
	}
}

// TestCredit_MinPrincipalFill verifies the two cases of the matching below the
// minimum principal: a request whose remainder after a fill is below it ends, and an
// offer whose remainder is below it is skipped for the next request.
//
// Arithmetic: MinPrincipal is 100. Agent 6 offers 1 000 at 400 and agent 5 offers
// 5 000 at 600; agent 2 asks 1 050 at 1 500 and agent 3 asks 500 at 1 000. Agent 2 takes
// 1 000 from agent 6, its remainder 50 is below 100 and the request ends (50 unfilled);
// agent 3 finds agent 6 spent and takes 500 from agent 5.
//
// Requirement: FR-CRD-024
func TestCredit_MinPrincipalFill(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = asks(CreditRequest{Principal: 1_050, Term: 30, MaxRateBP: 1_500})
	s.plan[hMillB] = asks(CreditRequest{Principal: 500, Term: 30, MaxRateBP: 1_000})
	s.plan[hLendX] = offers(CreditOffer{Principal: 5_000, MaxTerm: 90, RateBP: 600})
	s.plan[hLendY] = offers(CreditOffer{Principal: 1_000, MaxTerm: 90, RateBP: 400})
	w, rec, _ := loanWorld(t, s)
	step(t, w, 1)
	want := []loanRow{{hLendY, hMillA, 1_000, 400}, {hLendX, hMillB, 500, 600}}
	if got := rowsOf(rec); !slices.Equal(got, want) {
		t.Errorf("loans formed = %+v, want %+v", got, want)
	}
	if r := of[CreditRound](rec); len(r) != 1 || r[0].Unfilled != 50 {
		t.Errorf("CreditRound = %+v, want 50 unfilled", r)
	}
}

// TestCredit_ScaleApplies verifies that the lane's rate scale makes the effective
// rate of an offer: 600 at a scale of 15 000 is 900, so a limit of 800 gets no loan; a
// scale of 0 lends at no interest; and the effective rate is clamped to MaxCreditRateBP.
//
// Requirement: FR-CRD-024
func TestCredit_ScaleApplies(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name            string
		scale, posted   int64
		limit           int64
		wantRate        int64
		wantLoan        bool
		wantZeroInteres bool
	}{
		{"scaled up", 15_000, 600, 1_000, 900, true, false},
		{"a limit equal to the rate is filled", 15_000, 600, 900, 900, true, false},
		{"a limit one below the rate is not", 15_000, 600, 899, 0, false, false},
		{"scaled above the limit", 15_000, 600, 800, 0, false, false},
		{"scaled to zero", 0, 600, 1_000, 0, true, true},
		{"clamped", 20_000, 90_000, MaxCreditRateBP, MaxCreditRateBP, true, false},
	} {
		s := newLoanScript()
		s.plan[hMillA] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: tt.limit})
		s.plan[hLendX] = offers(CreditOffer{Principal: 1_000, MaxTerm: 90, RateBP: tt.posted})
		w, rec, _ := loanWorld(t, s)
		w.credit.scale[0] = tt.scale
		step(t, w, 1)
		got := formedOf(rec)
		switch {
		case tt.wantLoan && (len(got) != 1 || got[0].RateBP != tt.wantRate || (got[0].Interest == 0) != tt.wantZeroInteres):
			t.Errorf("%s: LoanFormed = %+v, want one at rate %d (interest-free %v)", tt.name, got, tt.wantRate, tt.wantZeroInteres)
		case !tt.wantLoan && len(got) != 0:
			t.Errorf("%s: LoanFormed = %+v, want none", tt.name, got)
		}
	}
}
