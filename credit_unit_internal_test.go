package economysim

import (
	"math"
	"slices"
	"strings"
	"testing"

	crd "github.com/Whealinghold/economy-sim/internal/credit"
)

// TestCredit_ShapeRefusal verifies the three shape rows of the table of
// spec-credit.md 5.2 at their exact boundaries: the principal against the module's
// minimum and MaxMoney, the term against 1 and MaxCreditTerm, the rate against 0 and
// MaxCreditRateBP. One step inside each bound is admitted, one step outside is refused
// with that row's reason, and the rows are checked in the order principal, term, rate.
//
// Arithmetic: loanConfig sets the minimum principal to 100, so 99 is the last refused
// and 100 the first admitted.
//
// Requirement: FR-CRD-013
func TestCredit_ShapeRefusal(t *testing.T) {
	t.Parallel()
	w, _, _ := loanWorld(t, newLoanScript())
	tests := []struct {
		name                    string
		principal, term, rateBP int64
		want                    Reason
	}{
		{"all at the lower bounds", 100, 1, 0, ""},
		{"all at the upper bounds", MaxMoney, MaxCreditTerm, MaxCreditRateBP, ""},
		{"principal one below the minimum", 99, 30, 500, ReasonInvalidPrincipal},
		{"principal one above MaxMoney", MaxMoney + 1, 30, 500, ReasonInvalidPrincipal},
		{"negative principal", -1, 30, 500, ReasonInvalidPrincipal},
		{"term 0", 100, 0, 500, ReasonInvalidTerm},
		{"term one above the largest", 100, MaxCreditTerm + 1, 500, ReasonInvalidTerm},
		{"negative rate", 100, 30, -1, ReasonInvalidRate},
		{"rate one above the largest", 100, 30, MaxCreditRateBP + 1, ReasonInvalidRate},
		{"principal is checked before term", 99, 0, 500, ReasonInvalidPrincipal},
		{"term is checked before rate", 100, 0, -1, ReasonInvalidTerm},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := w.shapeRefusal(tt.principal, tt.term, tt.rateBP); got != tt.want {
				t.Errorf("shapeRefusal(%d, %d, %d) = %q, want %q", tt.principal, tt.term, tt.rateBP, got, tt.want)
			}
		})
	}
}

// TestCredit_RefusalPrecedence verifies the order of the refusal rows for a request
// and for an offer when more than one row fails: the role's block first, then the
// shape, then a second submission of the same agent, and last the money (the exposure
// room of a request, the available money of an offer). A request exactly at the room
// and an offer exactly at the available money are admitted.
//
// Requirement: FR-CRD-013, FR-CRD-014, FR-CRD-015, FR-CRD-016, FR-CRD-017
func TestCredit_RefusalPrecedence(t *testing.T) {
	t.Parallel()
	w, _, _ := loanWorld(t, newLoanScript())
	w.creditPrices()
	room := w.creditRoomAt(w.cfg.Roles[w.agents[hMillA].role].Credit.Borrow, w.creditUpkeep(&w.agents[hMillA]), 0)
	if room < 1 || room >= MaxMoney {
		t.Fatalf("exposure room of agent %d = %d, want a room the test can step over", hMillA, room)
	}
	held := heldMoney(t, w, hLendX)
	rd := &w.credit.round

	t.Run("request", func(t *testing.T) {
		okReq := CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 500}
		tests := []struct {
			name  string
			agent AgentID
			req   CreditRequest
			prior bool // the agent's request is already admitted
			want  Reason
		}{
			{"a lender is no borrower, whatever the shape", hLendX, CreditRequest{Principal: 1}, false, ReasonNotBorrower},
			{"a bad shape beats a duplicate", hMillA, CreditRequest{Principal: 1, Term: 30}, true, ReasonInvalidPrincipal},
			{"a duplicate beats the exposure", hMillA, CreditRequest{Principal: room + 1, Term: 30, MaxRateBP: 500}, true, ReasonDuplicate},
			{"a second request of another agent is no duplicate", hMillB, okReq, true, ""},
			{"exactly the room is admitted", hMillA, CreditRequest{Principal: room, Term: 30, MaxRateBP: 500}, false, ""},
			{"one above the room is refused whole", hMillA, CreditRequest{Principal: room + 1, Term: 30, MaxRateBP: 500}, false, ReasonExposure},
		}
		for _, tt := range tests {
			rd.rq = rd.rq[:0]
			if tt.prior {
				rd.rq = append(rd.rq, loanRequest{agent: hMillA})
			}
			s := subRequest{agent: tt.agent, r: tt.req}
			if got := w.creditRequestRefusal(&w.agents[tt.agent], &s); got != tt.want {
				t.Errorf("%s: creditRequestRefusal(agent %d, %+v) = %q, want %q", tt.name, tt.agent, tt.req, got, tt.want)
			}
		}
		rd.rq = rd.rq[:0]
	})
	t.Run("offer", func(t *testing.T) {
		okOff := CreditOffer{Principal: 1_000, MaxTerm: 30, RateBP: 500}
		tests := []struct {
			name  string
			agent AgentID
			off   CreditOffer
			prior bool
			want  Reason
		}{
			{"a borrower is no lender, whatever the shape", hMillA, CreditOffer{Principal: 1}, false, ReasonNotLender},
			{"a bad shape beats a duplicate", hLendX, CreditOffer{Principal: 1, MaxTerm: 30}, true, ReasonInvalidPrincipal},
			{"a duplicate beats the funds", hLendX, CreditOffer{Principal: held + 1, MaxTerm: 30, RateBP: 500}, true, ReasonDuplicate},
			{"a second offer of another agent is no duplicate", hLendY, okOff, true, ""},
			{"exactly the available money is admitted", hLendX, CreditOffer{Principal: held, MaxTerm: 30, RateBP: 500}, false, ""},
			{"one above the available money is refused whole", hLendX, CreditOffer{Principal: held + 1, MaxTerm: 30, RateBP: 500}, false, ReasonInsufficientFunds},
		}
		for _, tt := range tests {
			rd.of = rd.of[:0]
			if tt.prior {
				rd.of = append(rd.of, loanOffer{agent: hLendX})
			}
			s := subOffer{agent: tt.agent, o: tt.off}
			if got := w.creditOfferRefusal(&w.agents[tt.agent], &s); got != tt.want {
				t.Errorf("%s: creditOfferRefusal(agent %d, %+v) = %q, want %q", tt.name, tt.agent, tt.off, got, tt.want)
			}
		}
		rd.of = rd.of[:0]
	})
}

// TestCreditCount_RefuseRequest verifies that a refused request is counted in the
// total, and that the arrears and exposure reasons are also counted in their own
// subset while every other reason is in the total only.
//
// Requirement: FR-CRD-074
func TestCreditCount_RefuseRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		reason                 Reason
		wantTotal, wantArrears int64
		wantExposure           int64
	}{
		{ReasonInArrears, 1, 1, 0},
		{ReasonExposure, 1, 0, 1},
		{ReasonDuplicate, 1, 0, 0},
		{ReasonNotBorrower, 1, 0, 0},
		{ReasonInvalidRate, 1, 0, 0},
	}
	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()
			var c creditCount
			c.refuseRequest(tt.reason)
			if c.requestsRefused != tt.wantTotal || c.inArrears != tt.wantArrears || c.exposure != tt.wantExposure {
				t.Errorf("refuseRequest(%q) gave refused %d, in arrears %d, exposure %d, want %d, %d, %d",
					tt.reason, c.requestsRefused, c.inArrears, c.exposure, tt.wantTotal, tt.wantArrears, tt.wantExposure)
			}
		})
	}
}

// TestCreditCount_Quiet verifies that a region whose every reported count is 0 is
// quiet whatever its price-control counter says, since that counter is in no event,
// and that any other count makes the region not quiet, so CreditRound is emitted.
//
// Requirement: FR-CRD-070
func TestCreditCount_Quiet(t *testing.T) {
	t.Parallel()
	if !(creditCount{priceControlled: 7}).quiet() {
		t.Errorf("quiet() of a count with only priceControlled = false, want true")
	}
	tests := map[string]creditCount{
		"requests":        {requests: 1},
		"offers":          {offers: 1},
		"requestsRefused": {requestsRefused: 1},
		"offersRefused":   {offersRefused: 1},
		"inArrears":       {inArrears: 1},
		"exposure":        {exposure: 1},
		"formed":          {formed: 1},
		"principal":       {principal: 1},
		"interest":        {interest: 1},
		"unfilled":        {unfilled: 1},
		"tooMany":         {tooMany: 1},
		"overflow":        {overflow: 1},
	}
	for name, c := range tests {
		if c.quiet() {
			t.Errorf("quiet() of a count with %s = 1 is true, want false", name)
		}
	}
}

// TestCredit_AddSat verifies the saturating sum of a report: an ordinary sum is exact,
// a sum at MaxMoney stays there, and a sum above MaxMoney or past int64 saturates at
// MaxMoney instead of wrapping.
//
// Requirement: FR-CRD-074
func TestCredit_AddSat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b int64
		want int64
	}{
		{"small", 2, 3, 5},
		{"zero", 0, 0, 0},
		{"exactly MaxMoney", MaxMoney - 1, 1, MaxMoney},
		{"one above MaxMoney", MaxMoney, 1, MaxMoney},
		{"two large principals", MaxMoney, MaxMoney, MaxMoney},
		{"past int64", math.MaxInt64, 1, MaxMoney},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := addSat(tt.a, tt.b); got != tt.want {
				t.Errorf("addSat(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestCredit_RateSeamInvariants verifies the two violated-invariant panics of the
// admission: a rate cap below 0 from the rate limit seam, and an offer whose
// effective rate cannot be computed because the region's scale is negative.
//
// Guard: both states are impossible from any configuration or call that validates,
// so the panic is the only sign if a later module's seam or the scale slot breaks
// that; without the test, a silent fall to a wrong rate would go unseen.
func TestCredit_RateSeamInvariants(t *testing.T) {
	t.Parallel()
	const want = "violated programmer invariant"
	t.Run("negative cap", func(t *testing.T) {
		t.Parallel()
		w, _, _ := loanWorld(t, newLoanScript())
		w.credit.rateCap = func(RegionID, int64) (int64, bool) { return -1, true }
		w.credit.round.reqs = []subRequest{{hMillA, CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 500}}}
		w.creditPrices()
		if msg := invariantMessage(w.admitCreditRequests); !strings.Contains(msg, want) {
			t.Errorf("admitCreditRequests() with a cap of -1 panicked with %q, want a message with %q", msg, want)
		}
	})
	t.Run("a cap below 0 that reports no cap is ignored", func(t *testing.T) {
		t.Parallel()
		w, _, _ := loanWorld(t, newLoanScript())
		w.credit.rateCap = func(RegionID, int64) (int64, bool) { return -1, false }
		w.credit.round.reqs = []subRequest{{hMillA, CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 500}}}
		w.creditPrices()
		if msg := invariantMessage(w.admitCreditRequests); msg != "" {
			t.Errorf("admitCreditRequests() with no cap in force panicked with %q, want no panic", msg)
		}
	})
	t.Run("negative scale", func(t *testing.T) {
		t.Parallel()
		w, _, _ := loanWorld(t, newLoanScript())
		w.credit.scale[0] = -1
		w.credit.round.offs = []subOffer{{hLendX, CreditOffer{Principal: 1_000, MaxTerm: 30, RateBP: 500}}}
		if msg := invariantMessage(w.admitCreditOffers); !strings.Contains(msg, want) {
			t.Errorf("admitCreditOffers() with a scale of -1 panicked with %q, want a message with %q", msg, want)
		}
	})
}

// TestCredit_FormLoanBalanceOverflow verifies that a fill the ledger refuses because
// the borrower's balance would pass MaxMoney is dropped and counted as Overflow like
// one whose interest overflows: no claim, no money moved, the matching goes on.
//
// Arithmetic: the borrower is set to MaxMoney - 50 and the fill is for 100, so the
// borrower would hold MaxMoney + 50.
//
// Requirement: FR-CRD-025
func TestCredit_FormLoanBalanceOverflow(t *testing.T) {
	t.Parallel()
	w, _, _ := loanWorld(t, newLoanScript())
	topUp(t, w, w.agents[hMillA].account, 50)
	before, claims := moneyTotal(w), w.ledger.OpenClaims()
	rd := &w.credit.round
	rd.rq, rd.of = []loanRequest{{agent: hMillA, region: 0}}, []loanOffer{{agent: hLendX, region: 0}}
	got := w.formLoan(crd.Fill{Request: 0, Offer: 0, Principal: 100, RateBP: 0, Term: 30})
	if got != crd.Dropped || rd.counts[0].overflow != 1 || rd.counts[0].formed != 0 {
		t.Errorf("formLoan(100 to a borrower at MaxMoney-50) = %v with %+v, want Dropped, one overflow, none formed", got, rd.counts[0])
	}
	if w.ledger.OpenClaims() != claims || moneyTotal(w) != before {
		t.Errorf("formLoan(overflowing balance) left %d claims and %d money, want %d and %d", w.ledger.OpenClaims(), moneyTotal(w), claims, before)
	}
}

// TestDefaultCredit_OverflowGuards verifies the three guards of the default rule
// against a sum that does not fit: a keep that overflows leaves nothing to lend, the
// bids of a plan whose products overflow read as MaxMoney (so nothing is lendable),
// and the tier 2 cost of a plan whose products overflow reads as MaxMoney (so the
// request is capped by the room, never by a wrapped small figure).
//
// Arithmetic: KeepTicks MaxInt64 times an upkeep cost of 2 passes int64; a bid of
// MaxInt64 units at MaxInt64 passes it too; a tier 2 of 2 units at a limit of
// MaxInt64 passes it.
//
// Requirement: FR-CRD-051, FR-CRD-052
func TestDefaultCredit_OverflowGuards(t *testing.T) {
	t.Parallel()
	t.Run("keep overflows", func(t *testing.T) {
		t.Parallel()
		v := &AgentView{Money: MaxMoney, Credit: &CreditView{
			Lend:         &LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: math.MaxInt64, ShareBP: 10_000},
			MinPrincipal: 1, UpkeepCost: 2,
		}}
		if o, ok := DefaultLend(v, 0); ok {
			t.Errorf("DefaultLend(keep past int64) = %+v, true, want none", o)
		}
	})
	t.Run("bids overflow", func(t *testing.T) {
		t.Parallel()
		p := &TradePlan{Intents: []Intent{{Side: Bid, Limit: math.MaxInt64, Quantity: math.MaxInt64}}}
		if got := planBids(p); got != MaxMoney {
			t.Errorf("planBids(overflowing bid) = %d, want MaxMoney %d", got, MaxMoney)
		}
	})
	t.Run("only bids count", func(t *testing.T) {
		t.Parallel()
		p := &TradePlan{Intents: []Intent{
			{Side: Bid, Limit: 10, Quantity: 3}, {Side: Ask, Limit: 99, Quantity: 99}, {Side: Bid, Limit: 5, Quantity: 2},
		}}
		if got := planBids(p); got != 40 { // 10*3 + 5*2
			t.Errorf("planBids(two bids and an ask) = %d, want 40", got)
		}
	})
	t.Run("tier 2 overflows", func(t *testing.T) {
		t.Parallel()
		tc := &tradeCtx{v: &AgentView{Credit: &CreditView{}}, rows: []tradeRow{{kind: bidOrder, limit: math.MaxInt64, t2: 2}}}
		if got := tc.tier2Cost(); got != MaxMoney {
			t.Errorf("tier2Cost(overflowing row) = %d, want MaxMoney %d", got, MaxMoney)
		}
	})
	t.Run("tier 2 is 0 with no credit group", func(t *testing.T) {
		t.Parallel()
		tc := &tradeCtx{v: &AgentView{}, rows: []tradeRow{{kind: bidOrder, limit: 10, t2: 2}}}
		if got := tc.tier2Cost(); got != 0 {
			t.Errorf("tier2Cost(no credit group) = %d, want 0", got)
		}
	})
	t.Run("tier 2 sums the bid rows at their limits", func(t *testing.T) {
		t.Parallel()
		tc := &tradeCtx{v: &AgentView{Credit: &CreditView{}}, rows: []tradeRow{
			{kind: bidOrder, limit: 10, t2: 2}, {kind: bidOrder, limit: 0, t2: 5}, {kind: bidOrder, limit: 7, t2: 1},
		}}
		if got := tc.tier2Cost(); got != 27 { // 10*2 + 7*1; the limit 0 row is no bid
			t.Errorf("tier2Cost(three rows) = %d, want 27", got)
		}
	})
}

// TestCredit_PlanWithModuleOff verifies the plan check with the module off
// (hook 3 of plan 3-crd): a strategy that appends a request and an offer has each
// refused at once as credit_off, with a Debug line and no event, no claim and no
// money moved, and the step is no error. The check is nil-safe, so the fan-in holds
// no branch on the module.
//
// Requirement: FR-CRD-053, FR-CRD-017
func TestCredit_PlanWithModuleOff(t *testing.T) {
	t.Parallel()
	c := loanConfig()
	c.Modules.Credit = nil
	for i := range c.Roles {
		c.Roles[i].Credit = nil
	}
	s := newLoanScript()
	s.plan[hMillA] = asks(CreditRequest{Principal: 1_000, Term: 30, MaxRateBP: 500})
	s.plan[hLendX] = offers(CreditOffer{Principal: 1_000, MaxTerm: 30, RateBP: 500})
	w, rec, lc := loanWorldFor(t, c, s)
	before := moneyTotal(w)
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	all := slices.Clone(*lc.lines) // take forgets every line, so read both messages first
	for _, tt := range []struct {
		msg   string
		agent AgentID
	}{{"request refused", hMillA}, {"offer refused", hLendX}} {
		lines := withMessage(all, tt.msg)
		if len(lines) != 1 || lines[0].strs["reason"] != string(ReasonCreditOff) || lines[0].attrs["agent"] != int64(tt.agent) {
			t.Errorf("Debug lines %q = %+v, want one for agent %d with reason %q", tt.msg, lines, tt.agent, ReasonCreditOff)
		}
	}
	if n := len(of[LoanFormed](rec)) + len(of[CreditRound](rec)); n != 0 {
		t.Errorf("a world with the module off emitted %d credit events, want 0", n)
	}
	if got := moneyTotal(w); got != before {
		t.Errorf("money after a step with refused credit_off plans = %d, want %d", got, before)
	}
}

// withMessage returns the captured lines with the message.
func withMessage(all []logRec, msg string) []logRec {
	return slices.DeleteFunc(slices.Clone(all), func(l logRec) bool { return l.msg != msg })
}
