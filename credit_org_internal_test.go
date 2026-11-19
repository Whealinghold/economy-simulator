package economysim

import (
	"slices"
	"testing"
)

// The tests of the rate limit of the institutions module in the credit round of P3.3b
// (spec-institutions.md 6.6, spec-credit.md 5.2.3, 13.3). The fixture is loanConfig, whose
// region home has three millers that borrow and two weavers that lend.

// loanLimit is loanConfig with a rate limit in home of the cap in basis points.
func loanLimit(capBP int64) Config {
	c := loanConfig()
	c.Modules.Institutions = &Institutions{RateLimits: []RateLimit{{ID: "usury", Region: "home", CapBP: capBP}}}
	return c
}

// TestRateLimit_RequestLimitReplaced verifies example C of spec-institutions.md 13.3: with a
// cap of 1 200 the requests of limit 2 000 and 1 500 are admitted with 1 200 and the one of 1 000
// as it is; the lender at 1 100 matches the first two and not the third, and the lender at 1 400
// matches none. PriceControlled counts two requests lowered, with no commodity and no units.
//
// Arithmetic: three requests of 200 against the lender at 1 100 with a principal of 400: the first
// two (limits 1 200 after the cap, both at or above 1 100) take 200 each; the third (1 000) is
// below 1 100 and finds nothing, and the lender at 1 400 is above all three limits.
//
// Requirement: FR-ORG-034, FR-CRD-028
func TestRateLimit_RequestLimitReplaced(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = asks(CreditRequest{Principal: 200, Term: 30, MaxRateBP: 2_000})
	s.plan[hMillB] = asks(CreditRequest{Principal: 200, Term: 30, MaxRateBP: 1_500})
	s.plan[hMillC] = asks(CreditRequest{Principal: 200, Term: 30, MaxRateBP: 1_000})
	s.plan[hLendX] = offers(CreditOffer{Principal: 400, MaxTerm: 90, RateBP: 1_100})
	s.plan[hLendY] = offers(CreditOffer{Principal: 600, MaxTerm: 90, RateBP: 1_400})
	w, rec, _ := loanWorldFor(t, loanLimit(1_200), s)
	step(t, w, 1)
	want := []loanRow{{hLendX, hMillA, 200, 1_100}, {hLendX, hMillB, 200, 1_100}}
	got := rowsOf(rec)
	// The two requests tie at 1 200 after the cap, so the keyed tie-break orders them: compare
	// the rows as a set by borrower.
	slices.SortFunc(got, func(a, b loanRow) int { return int(a.borrower) - int(b.borrower) })
	if !slices.Equal(got, want) {
		t.Errorf("loans formed = %+v, want %+v", got, want)
	}
	pc := of[PriceControlled](rec)
	wantEvent := PriceControlled{Tick: 0, Region: 0, Control: 0, Commodity: NoCommodity, Bids: 2}
	if len(pc) != 1 || pc[0] != wantEvent {
		t.Errorf("PriceControlled = %+v, want [%+v]", pc, wantEvent)
	}
}

// TestRateLimit_ClaimInForceUnchanged verifies that a cap alters no claim in force: a rate is
// fixed at formation (spec-credit.md 7.1.3), so a loan at 600 under a cap of 300 keeps its rate
// and its amounts, and no request is refused for the cap (spec-institutions.md 6.6.3).
//
// Requirement: FR-ORG-034
func TestRateLimit_ClaimInForceUnchanged(t *testing.T) {
	t.Parallel()
	w, rec, _ := loanWorldFor(t, loanLimit(300), newLoanScript())
	id := plantLoan(t, w, hLendX, hMillA, 500, 20, 40)
	before, ok := w.ledger.Claim(id)
	if !ok {
		t.Fatalf("Claim(%d) is not open after plantLoan", id)
	}
	step(t, w, 3)
	after, ok := w.ledger.Claim(id)
	if !ok || after != before {
		t.Errorf("claim after 3 ticks under the cap = %+v, %v, want %+v unchanged", after, ok, before)
	}
	if got := of[PriceControlled](rec); len(got) != 0 {
		t.Errorf("PriceControlled = %+v, want none: no request was made", got)
	}
}

// TestRateLimit_NoRequestRefusedForTheCap verifies that a request is never refused for the
// cap: a cap of 0 (interest-free credit only) replaces the limit and the request is admitted
// and counted, with no lender at 0 to match it (spec-institutions.md 6.6.1).
//
// Requirement: FR-ORG-034
func TestRateLimit_NoRequestRefusedForTheCap(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	s.plan[hMillA] = asks(CreditRequest{Principal: 200, Term: 30, MaxRateBP: 1_500})
	s.plan[hLendX] = offers(CreditOffer{Principal: 400, MaxTerm: 90, RateBP: 600})
	w, rec, _ := loanWorldFor(t, loanLimit(0), s)
	step(t, w, 1)
	r := of[CreditRound](rec)
	if len(r) != 1 || r[0].Requests != 1 || r[0].RequestsRefused != 0 || r[0].Formed != 0 {
		t.Errorf("CreditRound = %+v, want 1 request admitted, none refused and no loan", r)
	}
	if pc := of[PriceControlled](rec); len(pc) != 1 || pc[0].Bids != 1 {
		t.Errorf("PriceControlled = %+v, want one request lowered to the cap of 0", pc)
	}
}
