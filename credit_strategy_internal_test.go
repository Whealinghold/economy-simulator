package economysim

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// creditBorrowView is the view of a borrower of example A of spec-credit.md 20.1 as
// the default rule reads it: upkeep cost 100, minimum principal 100, MaxRateBP 1 500,
// room 9 000 (MaxDebtTicks 90), SlackTicks 4.
func creditBorrowView(money int64) *AgentView {
	return &AgentView{Money: money, Credit: &CreditView{
		Borrow:       &BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4},
		MinPrincipal: 100, UpkeepCost: 100, CreditRoom: 9_000, NextDue: -1,
	}}
}

// TestDefaultBorrow_Rule verifies the rule of spec-credit.md 10.2 on its worked
// example and on each branch: a job to start and a job in progress, the Late tick,
// the room, the minimum principal and the figures that give no request.
//
// Arithmetic: example A, J = 60 and H = 60 + 1 + 0 + 4 = 65, need = 100 * 65 + 200 -
// 1 000 = 5 700 under the room 9 000, so the request is 5 700 over 65 ticks at 1 500.
// With 150 held, 150 - 100 = 50 is below the inputs 200, so Late is 1, H = 66 and
// need = 6 600 + 200 - 150 = 6 650. A job in progress with 10 ticks left has
// H = 10 + 1 + 4 = 15 and no inputs, need = 1 500 - 400 = 1 100.
//
// Requirement: FR-CRD-051
func TestDefaultBorrow_Rule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		money          int64
		tier2, dur     int64
		job            int64 // ticks left of a job in progress, 0 for none
		room           int64
		want           CreditRequest
		wantOK         bool
		noBlockOrLimit bool
	}{
		{"example A", 1_000, 200, 60, 0, 9_000, CreditRequest{Principal: 5_700, Term: 65, MaxRateBP: 1_500}, true, false},
		{"late: the inputs cannot be bought at once", 150, 200, 60, 0, 9_000, CreditRequest{Principal: 6_650, Term: 66, MaxRateBP: 1_500}, true, false},
		{"a job in progress", 400, 0, 0, 10, 9_000, CreditRequest{Principal: 1_100, Term: 15, MaxRateBP: 1_500}, true, false},
		{"a job in progress has no late tick", 0, 500, 0, 10, 9_000, CreditRequest{Principal: 2_000, Term: 15, MaxRateBP: 1_500}, true, false},
		{"the room caps the ask", 1_000, 200, 60, 0, 3_000, CreditRequest{Principal: 3_000, Term: 65, MaxRateBP: 1_500}, true, false},
		{"no job and no planned recipe", 1_000, 0, 0, 0, 9_000, CreditRequest{}, false, false},
		{"no gap", 100_000, 200, 60, 0, 9_000, CreditRequest{}, false, false},
		{"the ask is below the minimum principal", 1_000, 200, 60, 0, 99, CreditRequest{}, false, false},
		{"no room at all", 1_000, 200, 60, 0, 0, CreditRequest{}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := creditBorrowView(tt.money)
			v.Credit.CreditRoom = tt.room
			if tt.job > 0 {
				v.HasJob, v.Remaining = true, tt.job
			}
			got, ok := DefaultBorrow(v, tt.tier2, tt.dur)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("DefaultBorrow(money %d, tier2 %d, duration %d, job %d, room %d) = %+v, %v, want %+v, %v",
					tt.money, tt.tier2, tt.dur, tt.job, tt.room, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestDefaultBorrow_NoRequest verifies that a view with no credit group, or one with
// a lender's block, and a view whose figures overflow, give no request and no panic.
//
// Requirement: FR-CRD-051, FR-CRD-053
func TestDefaultBorrow_NoRequest(t *testing.T) {
	t.Parallel()
	if r, ok := DefaultBorrow(&AgentView{Money: 1}, 100, 60); ok {
		t.Errorf("DefaultBorrow(no group) = %+v, true, want none", r)
	}
	lender := &AgentView{Credit: &CreditView{Lend: &LendTerms{ShareBP: 5_000}, MinPrincipal: 1}}
	if r, ok := DefaultBorrow(lender, 100, 60); ok {
		t.Errorf("DefaultBorrow(lender) = %+v, true, want none", r)
	}
	big := creditBorrowView(0)
	big.Credit.UpkeepCost = MaxMoney
	if r, ok := DefaultBorrow(big, MaxMoney, 3_650); ok {
		t.Errorf("DefaultBorrow(overflowing upkeep) = %+v, true, want none", r)
	}
}

// TestDefaultBorrow_TermClamped verifies that a horizon past MaxCreditTerm (a slack
// near the bound added to a long job) gives a term of MaxCreditTerm, never above: the
// module refuses a request whose term is out of bounds, so a default plan must not make
// one. The principal is still sized by the whole horizon.
//
// Requirement: FR-CRD-051
func TestDefaultBorrow_TermClamped(t *testing.T) {
	t.Parallel()
	v := creditBorrowView(0)
	v.Credit.Borrow.SlackTicks = MaxCreditTerm
	v.Credit.CreditRoom = MaxMoney
	v.HasJob, v.Remaining = true, 100
	got, ok := DefaultBorrow(v, 0, 0)
	if !ok || got.Term != MaxCreditTerm {
		t.Errorf("DefaultBorrow(job 100, slack %d) = %+v, %v, want a request with Term %d", MaxCreditTerm, got, ok, MaxCreditTerm)
	}
}

// TestDefaultLend_Rule verifies the rule of spec-credit.md 10.3 on lenders A and B of
// example A and on its branches: the bids of the plan and the kept upkeep come off,
// the share is floored, and an offer below the minimum principal is not made.
//
// Arithmetic: A holds 20 000 and keeps 3 * 80, so lendable 19 760 and the offer is
// 5 000 * 19 760 / 10 000 = 9 880; B holds 6 000 and offers 2 880; with bids of 5 000
// A has 14 760 and offers 7 380. 5 000 of 199 lendable is 99, below 100.
//
// Requirement: FR-CRD-052
func TestDefaultLend_Rule(t *testing.T) {
	t.Parallel()
	lend := &LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}
	tests := []struct {
		name         string
		money, bids  int64
		want         CreditOffer
		wantOK       bool
		keep, upkeep int64
	}{
		{"lender A", 20_000, 0, CreditOffer{Principal: 9_880, MaxTerm: 90, RateBP: 600}, true, 3, 80},
		{"lender B", 6_000, 0, CreditOffer{Principal: 2_880, MaxTerm: 90, RateBP: 600}, true, 3, 80},
		{"the bids come off", 20_000, 5_000, CreditOffer{Principal: 7_380, MaxTerm: 90, RateBP: 600}, true, 3, 80},
		{"the share is floored", 20_001, 0, CreditOffer{Principal: 9_880, MaxTerm: 90, RateBP: 600}, true, 3, 80},
		{"below the minimum principal", 439, 0, CreditOffer{}, false, 3, 80},
		{"nothing lendable", 200, 0, CreditOffer{}, false, 3, 80},
		{"the bids exceed the money", 1_000, 5_000, CreditOffer{}, false, 3, 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := &AgentView{Money: tt.money, Credit: &CreditView{Lend: lend, MinPrincipal: 100, UpkeepCost: tt.upkeep}}
			got, ok := DefaultLend(v, tt.bids)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("DefaultLend(money %d, bids %d) = %+v, %v, want %+v, %v", tt.money, tt.bids, got, ok, tt.want, tt.wantOK)
			}
		})
	}
	if o, ok := DefaultLend(&AgentView{Money: 1}, 0); ok {
		t.Errorf("DefaultLend(no group) = %+v, true, want none", o)
	}
}

// planTap wraps the default strategy and keeps what it appended for credit and the
// view it saw, by agent.
type planTap struct {
	DefaultStrategy
	borrow map[AgentID][]CreditRequest
	lend   map[AgentID][]CreditOffer
	views  map[AgentID]AgentView
	bids   map[AgentID]int64
}

func newPlanTap() *planTap {
	return &planTap{borrow: map[AgentID][]CreditRequest{}, lend: map[AgentID][]CreditOffer{}, views: map[AgentID]AgentView{}, bids: map[AgentID]int64{}}
}

// Trade runs the default Trade and records the plan's credit fields.
func (s *planTap) Trade(v *AgentView, rng *rand.Rand, p *TradePlan) {
	s.DefaultStrategy.Trade(v, rng, p)
	s.borrow[v.Agent] = append(s.borrow[v.Agent], p.Borrow...)
	s.lend[v.Agent] = append(s.lend[v.Agent], p.Lend...)
	cp := *v
	s.views[v.Agent], s.bids[v.Agent] = cp, planBids(p)
}

// TestDefault_NoBlockNoPlanFields verifies that the default strategy appends no
// request and no offer for a role with no credit block, with the module on and with
// it off, and that no view carries the group.
//
// Requirement: FR-CRD-053
func TestDefault_NoBlockNoPlanFields(t *testing.T) {
	t.Parallel()
	off := creditConfig(1, 0)
	off.Modules.Credit = nil
	for name, c := range map[string]Config{"module on, no block": creditConfig(1, 0), "module off": off} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tap := newPlanTap()
			c.Roles[0].Strategy, c.Roles[1].Strategy = "tap", "tap"
			w := newMill(t, c, WithSeed(exitSeed), WithStrategy("tap", tap))
			step(t, w, 3)
			if len(tap.views) == 0 {
				t.Fatal("the tap saw no view, want one for each agent")
			}
			for id, v := range tap.views {
				if v.Credit != nil || len(tap.borrow[id]) != 0 || len(tap.lend[id]) != 0 {
					t.Errorf("agent %d got group %v, requests %v, offers %v, want none", id, v.Credit, tap.borrow[id], tap.lend[id])
				}
			}
		})
	}
}

// TestDefaultStrategy_CreditThroughTrade verifies the default strategy's call of the
// rules: a borrower with a job to start asks for the gap of its horizon (the term
// is the recipe's duration + 1 + Late + SlackTicks) and a lender offers the share of
// what its bids and the kept upkeep leave, both through DefaultStrategy.Trade.
//
// Requirement: FR-CRD-051, FR-CRD-052
func TestDefaultStrategy_CreditThroughTrade(t *testing.T) {
	t.Parallel()
	tap := newPlanTap()
	c := loanConfig()
	c.Roles[0].Strategy, c.Roles[1].Strategy = "tap", "tap"
	w := newMill(t, c, WithSeed(exitSeed), WithStrategy("tap", tap))
	setBalances(t, w, loanBalances)
	step(t, w, 1)
	dur := w.cfg.Recipes[0].Duration
	if r := tap.borrow[hMillA]; len(r) != 1 || r[0].MaxRateBP != 1_500 || r[0].Term < dur+1+4 || r[0].Term > dur+1+1+4 || r[0].Principal < 100 {
		t.Errorf("the miller's requests = %+v, want one at 1 500 for a term of %d or %d ticks", r, dur+5, dur+6)
	}
	v := tap.views[hLendX]
	want := (v.Money - tap.bids[hLendX] - 3*v.Credit.UpkeepCost) * 5_000 / 10_000
	if o := tap.lend[hLendX]; len(o) != 1 || o[0].Principal != want || o[0].RateBP != 600 || o[0].MaxTerm != 90 {
		t.Errorf("the weaver's offers = %+v, want one of %d at 600 for 90 ticks", o, want)
	}
}

// TestCredit_ViewGroup verifies the credit group of the view: present for an agent
// whose role has a block and for no other, with the figures of 10.1 read from the
// claims (a loan and a payout claim as debtor, a loan as creditor, one claim
// overdue), and nothing of any other agent's.
//
// Arithmetic: the upkeep cost is 1 000 and MaxDebtTicks 90, so the room is 90 000
// less the debt. Agent 2 owes 500 of a loan (interest 20, due 5), so Debt 500, Owed
// 520, NextDue 5 and room 89 500; agent 3 owes a payout claim of 300 due at tick 1,
// which its pot of 1 000 - 1 000 = 0 cannot pay, so at tick 1 it is overdue with
// Overdue 300; agent 5 lent 500, so Lent 500 and Receivable 520.
//
// Requirement: FR-CRD-050
func TestCredit_ViewGroup(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	w, _, _ := loanWorld(t, s)
	plantLoan(t, w, hLendX, hMillA, 500, 20, 5)
	plantPayout(t, w, hLendY, hMillB, 300, 1)
	step(t, w, 1)
	borrower := CreditView{Borrow: &BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}, MinPrincipal: 100, UpkeepCost: 1_000,
		Debt: 500, Owed: 520, NextDue: 5, CreditRoom: 89_500, ScaleBP: wholeBP}
	lender := CreditView{Lend: &LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}, MinPrincipal: 100, UpkeepCost: 1_000,
		NextDue: -1, Lent: 500, Receivable: 520, ScaleBP: wholeBP}
	wantView(t, s, hMillA, borrower)
	wantView(t, s, hLendX, lender)
	other := CreditView{Borrow: borrower.Borrow, MinPrincipal: 100, UpkeepCost: 1_000, Debt: 300, Owed: 300, NextDue: 1, CreditRoom: 89_700, ScaleBP: wholeBP}
	wantView(t, s, hMillB, other)
	step(t, w, 1)
	other.Overdue = 300
	wantView(t, s, hMillB, other)
	if got := s.seen[hMillC]; got.Debt != 0 || got.Owed != 0 || got.NextDue != -1 || got.Lent != 0 {
		t.Errorf("view of agent %d = %+v, want none of the other agents' claims", hMillC, got)
	}
}

// wantView fails the test unless the credit group an agent saw equals want.
func wantView(t *testing.T, s *loanScript, id AgentID, want CreditView) {
	t.Helper()
	got := s.seen[id]
	gotBlocks := [2]any{got.Lend, got.Borrow}
	if (got.Lend == nil) != (want.Lend == nil) || (got.Borrow == nil) != (want.Borrow == nil) {
		t.Errorf("view of agent %d has blocks %v, want lend %v borrow %v", id, gotBlocks, want.Lend, want.Borrow)
		return
	}
	g, wnt := got, want
	if g.Lend != nil {
		if *g.Lend != *wnt.Lend {
			t.Errorf("view of agent %d has Lend %+v, want %+v", id, *g.Lend, *wnt.Lend)
		}
		g.Lend, wnt.Lend = nil, nil
	}
	if g.Borrow != nil {
		if *g.Borrow != *wnt.Borrow {
			t.Errorf("view of agent %d has Borrow %+v, want %+v", id, *g.Borrow, *wnt.Borrow)
		}
		g.Borrow, wnt.Borrow = nil, nil
	}
	if g != wnt {
		t.Errorf("view of agent %d = %+v, want %+v", id, g, wnt)
	}
}

// TestCredit_ViewGroupAbsent verifies that no agent of a world whose roles have no
// credit block is given the group, with the module on.
//
// Requirement: FR-CRD-050
func TestCredit_ViewGroupAbsent(t *testing.T) {
	t.Parallel()
	s := newLoanScript()
	c := creditConfig(1, 0)
	c.Roles[0].Strategy, c.Roles[1].Strategy = "script", "script"
	w := newMill(t, c, WithSeed(exitSeed), WithStrategy("script", s))
	step(t, w, 2)
	if len(s.missing) == 0 {
		t.Fatal("the script saw no view, want one for each agent")
	}
	for id, none := range s.missing {
		if !none {
			t.Errorf("agent %d was given the credit group, want none for a role with no block", id)
		}
	}
}

// TestDefaultCredit_WrappingFigures verifies that a hand-built view whose figures
// would wrap an int64 gives no request and no offer, instead of a wrapped one.
//
// Arithmetic: a duration of MaxInt64 plus the one of the phase order wraps H; a
// Money of -MaxMoney less a keep of MaxInt64 wraps the lendable figure to a positive.
//
// Requirement: FR-CRD-051, FR-CRD-052
func TestDefaultCredit_WrappingFigures(t *testing.T) {
	t.Parallel()
	if r, ok := DefaultBorrow(creditBorrowView(0), 0, math.MaxInt64); ok {
		t.Errorf("DefaultBorrow(duration MaxInt64) = %+v, true, want none", r)
	}
	v := &AgentView{Money: -MaxMoney, Credit: &CreditView{
		Lend:         &LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 1, ShareBP: 10_000},
		MinPrincipal: 1, UpkeepCost: math.MaxInt64,
	}}
	if o, ok := DefaultLend(v, MaxMoney); ok {
		t.Errorf("DefaultLend(wrapping lendable) = %+v, true, want none", o)
	}
}

// TestDirectCredit_ShrunkPlan verifies that a fallback that cut the plan's requests
// and offers below the index it was called at does not make the directive's entry
// reslice past what is left, and that the entry it adds is the only one after it.
//
// Requirement: FR-CRD-055
func TestDirectCredit_ShrunkPlan(t *testing.T) {
	t.Parallel()
	dr, do := CreditRequest{Principal: 900, Term: 9, MaxRateBP: 90}, CreditOffer{Principal: 900, MaxTerm: 9, RateBP: 90}
	p := TradePlan{Borrow: []CreditRequest{{Principal: 1}, {Principal: 2}}, Lend: []CreditOffer{{Principal: 3}}}
	nb, nl := len(p.Borrow), len(p.Lend)
	p.Borrow, p.Lend = p.Borrow[:0], p.Lend[:0]
	directCredit(&p, &Directive{Borrow: &dr, Lend: &do}, nb, nl)
	if len(p.Borrow) != 1 || p.Borrow[0] != dr || len(p.Lend) != 1 || p.Lend[0] != do {
		t.Errorf("plan after directCredit = %+v and %+v, want only the directive's %+v and %+v", p.Borrow, p.Lend, dr, do)
	}
}

// TestOriginOf_Unknown verifies that a ledger origin that is neither a loan nor a
// payout is a violated invariant, not a loan.
//
// Requirement: FR-CRD-078
func TestOriginOf_Unknown(t *testing.T) {
	t.Parallel()
	if got := originOf(ledger.OriginPayout); got != OriginPayout {
		t.Errorf("originOf(payout) = %v, want OriginPayout", got)
	}
	if got := originOf(ledger.OriginLoan); got != OriginLoan {
		t.Errorf("originOf(loan) = %v, want OriginLoan", got)
	}
	if msg := invariantMessage(func() { originOf(7) }); msg == "" {
		t.Error("originOf(7) did not panic, want a violated invariant")
	}
}
