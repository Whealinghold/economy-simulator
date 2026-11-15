package economysim

import (
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// claimOf builds a claim record for the index tests, which need no world.
func claimOf(id int64, creditor, debtor int, due, overdue int64) ledger.Claim {
	return ledger.Claim{ID: id, Creditor: creditor, Debtor: debtor, Principal: 10, PrincipalOwed: 10, Formed: 0, Due: due, Overdue: overdue}
}

// TestClaimIndex_FollowsTheClaimList verifies that the indexes of the claim list
// (spec-credit.md 17.3) keep up with a claim formed, marked overdue, passed to
// another creditor and ended, that rebuild gives the indexes of the list, and that
// check names the first claim that is missing from an index or is not in the list.
//
// Requirement: FR-CRD-082, NFR-CRD-003
func TestClaimIndex_FollowsTheClaimList(t *testing.T) {
	t.Parallel()
	a, b, c := claimOf(0, 1, 10, 5, -1), claimOf(1, 1, 11, 5, -1), claimOf(2, 2, 10, 9, -1)
	x := newClaimIndex()
	list := []ledger.Claim{a, b, c}
	for _, cl := range list {
		x.add(cl)
	}
	wantIndex(t, &x, list, "three claims formed")
	x.add(a) // an index of IDs lists a claim once, so a second add changes nothing
	wantIndex(t, &x, list, "a claim indexed twice")
	if got := *x.bucket(5); !slices.Equal(got, []int64{0, 1}) {
		t.Errorf("due bucket of tick 5 = %v, want [0 1]", got)
	}

	x.markOverdue(a)
	list[0].Overdue = 5
	wantIndex(t, &x, list, "a claim gone overdue")
	if !slices.Equal(x.overdue, []int64{0}) || len(*x.bucket(5)) != 1 {
		t.Errorf("overdue list %v and bucket %v, want [0] and the one claim left in the bucket", x.overdue, *x.bucket(5))
	}

	x.reassign(b, 3)
	list[1].Creditor = 3
	wantIndex(t, &x, list, "a claim passed")
	if got := x.byCreditor[3]; !slices.Equal(got, []int64{1}) || len(x.byCreditor[1]) != 1 {
		t.Errorf("creditor lists: account 3 %v, account 1 %v, want [1] and one claim left", got, x.byCreditor[1])
	}

	x.remove(list[0]) // the overdue claim ends
	list = list[1:]
	wantIndex(t, &x, list, "a claim ended")

	again := newClaimIndex()
	again.rebuild(list)
	wantIndex(t, &again, list, "a rebuild")
	x.rebuild(list)
	wantIndex(t, &x, list, "a rebuild over the old content")
}

// wantIndex fails the test unless the indexes are those of the list.
func wantIndex(t *testing.T, x *claimIndex, list []ledger.Claim, when string) {
	t.Helper()
	if err := x.check(list); err != nil {
		t.Errorf("check after %s = %v, want nil", when, err)
	}
}

// TestClaimIndex_CheckNamesTheClaim verifies that check fails, naming the claim, for
// a claim missing from the index of its debtor, of its creditor, from the overdue
// list or from its due bucket, and for an index that holds more than the list.
//
// Requirement: FR-CRD-082
func TestClaimIndex_CheckNamesTheClaim(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		damage  func(x *claimIndex)
		overdue int64
		want    string
	}{
		{"debtor", func(x *claimIndex) { x.byDebtor[10] = nil }, -1, "claim 7 is not in the index of its debtor account 10"},
		{"creditor", func(x *claimIndex) { x.byCreditor[1] = nil }, -1, "claim 7 is not in the index of its creditor account 1"},
		{"bucket", func(x *claimIndex) { *x.bucket(5) = nil }, -1, "claim 7 is not in the bucket of its due tick 5"},
		{"overdue list", func(x *claimIndex) { x.overdue = nil }, 5, "claim 7 is overdue and not in the overdue list"},
		{"a claim the list does not hold", func(x *claimIndex) { x.add(claimOf(9, 1, 11, 6, -1)) }, -1, "the indexes hold"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cl := claimOf(7, 1, 10, 5, tt.overdue)
			x := newClaimIndex()
			x.add(cl)
			tt.damage(&x)
			if err := x.check([]ledger.Claim{cl}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("check = %v, want an error containing %q", err, tt.want)
			}
		})
	}
}

// TestSettle_OneBucketServesTwoTicks verifies that a claim whose due tick is a ring
// away from another (a payout claim of any term, spec-credit.md 17.3) is not visited
// at the earlier tick, and that the pass visits only the claims due and overdue, not
// the open claims: with seven claims open, one due and one overdue, two are visited.
//
// Arithmetic: the ring has MaxCreditTerm + 1 = 3 651 buckets, so a claim due at tick
// 2 + 3 651 shares the bucket of tick 2.
//
// Requirement: FR-CRD-030, NFR-CRD-003
func TestSettle_OneBucketServesTwoTicks(t *testing.T) {
	t.Parallel()
	w, _ := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 4_000, homeB: 100})
	due := plantLoan(t, w, homeWeaver, homeA, 100, 0, 2)
	late := plantPayout(t, w, homeB, homeA, 50, 2+MaxCreditTerm+1)
	plantLoan(t, w, homeWeaver, homeB, 100, 0, 40)
	plantLoan(t, w, homeWeaver, homeC, 100, 0, 41)
	plantLoan(t, w, homeWeaver, yonderA, 100, 0, 42)
	plantPayout(t, w, homeWeaver, yonderB, 100, 43)
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeA: 1})
	w.p17aSettle()
	p := &w.credit.pass
	if len(p.visits) != 1 || p.visits[0].c.ID != due || len(p.debtors) != 1 {
		t.Errorf("visits at tick 2 = %d claims of %d debtors (first %d), want only claim %d", len(p.visits), len(p.debtors), p.visits[0].c.ID, due)
	}
	if c, _ := w.ledger.Claim(late); c.Overdue != -1 {
		t.Errorf("claim sharing the bucket is overdue from %d at tick 2, want -1: it is due at tick %d", c.Overdue, c.Due)
	}
	// The due claim went overdue unpaid: it and nothing else is visited at tick 3.
	step(t, w, 1)
	w.p17aSettle()
	if len(p.visits) != 1 || p.visits[0].c.ID != due {
		t.Errorf("visits at tick 3 = %d claims, want only the overdue claim %d", len(p.visits), due)
	}
}

// TestFigures_OfAnAgentsClaims verifies the figures of an agent's claims that the
// Credit group of the view will carry (spec-credit.md 10.1): the principal and the
// owed as debtor, the owed on overdue claims, the earliest due tick, and what it is
// owed as creditor; and the exposure room of a borrower, with the cap saturating and
// the room never below 0.
//
// Arithmetic: agent 3 owes 100 + 20 due at tick 40 and 300 + 0 due at tick 30, the
// second overdue, and is owed 50 + 5. Its debt is 400, its owed 420, 300 of it
// overdue, its next due tick 30, and it lent 50 and holds a receivable of 55. The role
// borrows up to 10 ticks of upkeep, so the cap is 10 * U and the room 10 * U - 400.
//
// Requirement: FR-CRD-050, FR-CRD-015
func TestFigures_OfAnAgentsClaims(t *testing.T) {
	t.Parallel()
	c := creditConfig(0, 0)
	c.Roles[0].Credit = &CreditTerms{Borrow: &BorrowTerms{MaxRateBP: 1_000, MaxDebtTicks: 10, SlackTicks: 1}}
	rec := &evLog{}
	w := newMill(t, c, WithSeed(exitSeed), WithObserver(rec))
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 1_000, homeB: 100})
	plantLoan(t, w, homeWeaver, homeB, 100, 20, 40)
	overdue := plantLoan(t, w, homeWeaver, homeB, 300, 0, 30)
	plantLoan(t, w, homeB, homeC, 50, 5, 90)
	w.credit.idx.markOverdue(ledger.Claim{ID: overdue, Due: 30})
	mustClaim(w.ledger.MarkOverdue(overdue, 30), "setup: mark overdue")
	got := w.figuresOf(w.agents[homeB].account)
	want := claimFigures{Debt: 400, Owed: 420, Overdue: 300, NextDue: 30, Lent: 50, Receivable: 55}
	if got != want {
		t.Errorf("figuresOf(agent 3) = %+v, want %+v", got, want)
	}
	if none := w.figuresOf(w.agents[yonderB].account); none != (claimFigures{NextDue: -1}) {
		t.Errorf("figuresOf(an agent with no claim) = %+v, want zeros and NextDue -1", none)
	}
	w.valuePrices()
	u := w.creditUpkeep(&w.agents[homeB])
	if room, wantRoom := w.creditRoom(&w.agents[homeB], 400), max(0, 10*u-400); room != wantRoom {
		t.Errorf("creditRoom(debt 400) = %d, want max(0, 10 * %d - 400) = %d", room, u, wantRoom)
	}
	if room := w.creditRoom(&w.agents[homeB], 10*u+1); room != 0 {
		t.Errorf("creditRoom(debt above the cap) = %d, want 0", room)
	}
	if room := w.creditRoom(&w.agents[yonderWeaver], 0); room != 0 {
		t.Errorf("creditRoom(a role with no Borrow block) = %d, want 0", room)
	}
}
