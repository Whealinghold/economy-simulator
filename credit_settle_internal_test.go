package economysim

import (
	"testing"
)

// settleAt builds the credit world with the shield and the write-off, steps it one
// tick and returns it at tick 1 with a recording observer and the audit on. A claim
// planted there with the due tick 2 falls due at the next boundary, which the test
// settles by hand with p17aSettle, so no other position moves a balance.
func settleAt(t *testing.T, shield, writeOff int64) (*World, *evLog) {
	t.Helper()
	rec := &evLog{}
	w := creditWorld(t, shield, writeOff, WithObserver(rec), WithAudit())
	step(t, w, 1)
	return w, rec
}

// upkeepOf is the upkeep cost U of an agent at the prices of the tick, the unit of
// the shield.
func upkeepOf(t *testing.T, w *World, id AgentID) int64 {
	t.Helper()
	w.valuePrices()
	u := w.creditUpkeep(&w.agents[id])
	if u < 1 {
		t.Fatalf("creditUpkeep(agent %d) = %d, want at least 1: the shield needs an upkeep", id, u)
	}
	return u
}

// TestSettle_Pot verifies the pot of a debtor at P1.7a: it pays the balance recorded
// at the entry less ShieldTicks of upkeep, never below 0 and never more than it owes,
// interest first; that what the shield kept out is counted in Shielded; that a claim
// still owed at its due tick is marked overdue and counted as defaulted; and that a
// claim paid in full ends repaid with LoanEnded before CreditSettled.
//
// Arithmetic: the claim owes 3 000 of principal and 300 of interest, 3 300 in all,
// ShieldTicks is 2, so the shield is 2 * U. A balance of shield + k leaves a pot of
// k, so it pays min(k, 3 300): the first 300 are interest. The creditor, the home
// weaver, holds 10 000 before the pass.
//
// Requirement: FR-CRD-031, FR-CRD-034, FR-CRD-071, FR-CRD-073, FR-CRD-074
func TestSettle_Pot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		above        int64 // the balance less the shield
		pay          int64
		wantInterest int64
	}{
		{"below the shield pays nothing", -1, 0, 0},
		{"at the shield pays nothing", 0, 0, 0},
		{"a unit above the shield pays a unit of interest", 1, 1, 1},
		{"a pot that covers the interest only", 300, 300, 300},
		{"a pot that pays some principal", 1_500, 1_500, 300},
		{"a pot that pays it all", 3_300, 3_300, 300},
		{"a pot above what is owed pays what is owed", 5_000, 3_300, 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := settleAt(t, 2, 0)
			setBalances(t, w, map[AgentID]int64{homeWeaver: 3_000})
			id := plantLoan(t, w, homeWeaver, homeA, 3_000, 300, 2)
			step(t, w, 1)
			rec.events = nil
			shield := 2 * upkeepOf(t, w, homeA)
			bal := max(shield+tt.above, 0)
			setBalances(t, w, map[AgentID]int64{homeA: bal, homeWeaver: 10_000})
			w.p17aSettle()
			wantP, wantI := 3_000-(tt.pay-tt.wantInterest), 300-tt.wantInterest
			if p, i, open := claimOwed(t, w, id); open == (tt.pay == 3_300) || (open && (p != wantP || i != wantI)) {
				t.Errorf("claim owes %d and %d (open %v) after the pass, want %d and %d", p, i, open, wantP, wantI)
			}
			if got := heldMoney(t, w, homeA); got != bal-tt.pay {
				t.Errorf("debtor holds %d, want %d: the balance %d less the %d paid", got, bal-tt.pay, bal, tt.pay)
			}
			if got := heldMoney(t, w, homeWeaver); got != 10_000+tt.pay {
				t.Errorf("creditor holds %d, want %d", got, 10_000+tt.pay)
			}
			want := CreditSettled{Tick: 2, Region: 0, Due: 1, PrincipalPaid: Money(tt.pay - tt.wantInterest), InterestPaid: Money(tt.wantInterest),
				Shielded: Money(min(bal, 3_300) - tt.pay), Received: Money(tt.pay)}
			if tt.pay == 3_300 {
				want.Repaid = 1
			} else {
				want.Defaulted, want.Arrears = 1, Money(3_300-tt.pay)
			}
			wantOne(t, rec, want)
			auditNow(t, w)
		})
	}
}

// TestSettle_PotIsTheBalanceAtTheEntry verifies that every pot of the pass is
// computed from the balances recorded at its entry: an agent that is a debtor and a
// creditor has the same pot whatever the order of the visits, so what it receives in
// the pass does not raise what it pays.
//
// Arithmetic: ShieldTicks is 0. Agent 3 owes agent 2 4 000 and holds 1 000 at the
// entry; agent 4 owes agent 3 500 and holds 5 000. The debtors are visited in
// ascending ID, so agent 3 is first, and it pays 1 000, not 1 500: whether agent 4's
// 500 has arrived by then does not matter, and in either order the pot is the
// recorded 1 000.
//
// Requirement: FR-CRD-031
func TestSettle_PotIsTheBalanceAtTheEntry(t *testing.T) {
	t.Parallel()
	w, rec := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeA: 4_000, homeB: 500, homeC: 0})
	owedToA := plantLoan(t, w, homeA, homeB, 4_000, 0, 2)
	owedToB := plantLoan(t, w, homeB, homeC, 500, 0, 2)
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeA: 0, homeB: 1_000, homeC: 5_000})
	rec.events = nil
	w.p17aSettle()
	if p, _, open := claimOwed(t, w, owedToA); !open || p != 3_000 {
		t.Errorf("the claim of agent 2 owes %d (open %v), want 3 000: agent 3 paid its recorded 1 000", p, open)
	}
	if _, _, open := claimOwed(t, w, owedToB); open {
		t.Error("the claim of agent 3 is open, want repaid by agent 4's 500")
	}
	if got := heldMoney(t, w, homeB); got != 500 {
		t.Errorf("agent 3 holds %d, want 500: its 1 000 less the 1 000 paid plus the 500 received", got)
	}
}

// TestSettle_Allocation verifies that a payment short of what a debtor owes is shared
// pro rata by what each claim owes with the largest remainder, with the worked case
// of spec-credit.md 20.2: a pot of 1 001 over claims that owe 600 and 900 pays 400
// and 601, and a pot of 1 000 pays exactly 400 and 600; that each share pays interest
// first; and that the money of the two creditors rises by their shares.
//
// Arithmetic: the claims are 550 of principal with 50 of interest, and 800 with 100.
// With 1 001, floor(1 001 * 600 / 1 500) = 400 and floor(1 001 * 900 / 1 500) = 600,
// one unit left that goes to the larger fractional part, 600.6 against 400.4, so the
// shares are 400 and 601: the first pays 50 of interest and 350 of principal and
// owes 200 of principal, the second pays 100 and 501 and owes 299. With 1 000 the
// shares are 400 and 600 and the second owes 300.
//
// Requirement: FR-CRD-032
func TestSettle_Allocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		pot            int64
		shareA, shareB int64
		owedA, owedB   int64
	}{
		{"a unit more than the even split", 1_001, 400, 601, 200, 299},
		{"the exact split", 1_000, 400, 600, 200, 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := settleAt(t, 0, 0)
			setBalances(t, w, map[AgentID]int64{homeWeaver: 550, homeB: 800})
			a := plantLoan(t, w, homeWeaver, homeA, 550, 50, 2)
			b := plantLoan(t, w, homeB, homeA, 800, 100, 2)
			step(t, w, 1)
			setBalances(t, w, map[AgentID]int64{homeA: tt.pot, homeWeaver: 0, homeB: 0})
			money := moneyTotal(w)
			w.p17aSettle()
			if got := moneyTotal(w); got != money {
				t.Errorf("money over the accounts = %d after the pass, want %d: a payment moves money and makes none", got, money)
			}
			if p, i, _ := claimOwed(t, w, a); p != tt.owedA || i != 0 {
				t.Errorf("claim A owes %d and %d, want %d and 0", p, i, tt.owedA)
			}
			if p, i, _ := claimOwed(t, w, b); p != tt.owedB || i != 0 {
				t.Errorf("claim B owes %d and %d, want %d and 0", p, i, tt.owedB)
			}
			if got, want := heldMoney(t, w, homeWeaver), tt.shareA; got != want {
				t.Errorf("creditor A holds %d, want its share %d", got, want)
			}
			if got, want := heldMoney(t, w, homeB), tt.shareB; got != want {
				t.Errorf("creditor B holds %d, want its share %d", got, want)
			}
			auditNow(t, w)
		})
	}
}
