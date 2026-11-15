package economysim

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// creditConfig is the world of the credit tests: two regions, home with three
// millers and a weaver and yonder with two millers and a weaver, every household an
// agent of weight 1, and the credit module on with the shield and the write-off the
// test gives. The agents are 2 to 4 (home millers), 5 (home weaver), 6 and 7 (yonder
// millers) and 8 (yonder weaver): the estates are agents 0 and 1. No role has a
// credit block, so no request is made and the tests plant their claims.
func creditConfig(shield, writeOff int64) Config {
	c := skillsConfig()
	c.Roles[1].TrainingTicks = 0
	c.Modules.Credit = &Credit{MinPrincipal: 1, ShieldTicks: shield, WriteOffTicks: writeOff, MaxClaims: 100}
	return c
}

// creditWorld builds the world of creditConfig with the options.
func creditWorld(t testing.TB, shield, writeOff int64, opts ...Option) *World {
	t.Helper()
	return newMill(t, creditConfig(shield, writeOff), append([]Option{WithSeed(exitSeed)}, opts...)...)
}

// heldMoney reads the money of an agent.
func heldMoney(t testing.TB, w *World, id AgentID) int64 {
	t.Helper()
	v, err := w.ledger.Account(w.agents[id].account)
	if err != nil {
		t.Fatalf("Account(agent %d) error = %v, want nil", id, err)
	}
	return v.Balance
}

// setBalances gives every live agent the money in want and 0 to the others. The money
// of an agent that has too much goes to the estate of its region and what an agent
// needs comes from it (reason start_money), so no money is made and the region's pool
// holds the rest. It fails the test when a pool cannot pay.
func setBalances(t testing.TB, w *World, want map[AgentID]int64) {
	t.Helper()
	for pass := range 2 {
		w.eachLive(func(a *agent) {
			g := w.regionOf(a)
			have, target := heldMoney(t, w, a.id), want[a.id]
			var err error
			switch {
			case pass == 0 && have > 0:
				err = w.ledger.Transfer(a.account, w.estates.account(g), have, ledger.EstateMoney)
			case pass == 1 && target > 0:
				err = w.ledger.Transfer(w.estates.account(g), a.account, target, ledger.StartMoney)
			}
			if err != nil {
				t.Fatalf("setBalances: agent %d: %v", a.id, err)
			}
		})
	}
}

// plantLoan forms a loan of the principal and the interest by the ledger's Lend,
// the way the round will, and indexes it. The creditor must hold the principal.
func plantLoan(t testing.TB, w *World, creditor, debtor AgentID, principal, interest, due int64) int64 {
	t.Helper()
	id, err := w.ledger.Lend(w.agents[creditor].account, w.agents[debtor].account, principal, interest, 600, due)
	if err != nil {
		t.Fatalf("plantLoan(%d to %d): Lend error = %v, want nil", creditor, debtor, err)
	}
	indexClaim(t, w, id)
	return id
}

// plantPayout forms a payout claim of an amount owed by debtor to creditor, with no
// transfer, as the insurance module will, and indexes it.
func plantPayout(t testing.TB, w *World, creditor, debtor AgentID, amount, due int64) int64 {
	t.Helper()
	id, err := w.ledger.Obligate(w.agents[creditor].account, w.agents[debtor].account, amount, due)
	if err != nil {
		t.Fatalf("plantPayout(%d to %d): Obligate error = %v, want nil", creditor, debtor, err)
	}
	indexClaim(t, w, id)
	return id
}

// indexClaim adds the claim with the ID to the derived indexes.
func indexClaim(t testing.TB, w *World, id int64) {
	t.Helper()
	c, ok := w.ledger.Claim(id)
	if !ok {
		t.Fatalf("claim %d is not open, want a claim just formed", id)
	}
	w.credit.idx.add(c)
}

// claimOwed reads the owed parts of a claim, and whether it is open.
func claimOwed(t testing.TB, w *World, id int64) (principal, interest int64, open bool) {
	t.Helper()
	c, ok := w.ledger.Claim(id)
	return c.PrincipalOwed, c.InterestOwed, ok
}

// auditNow runs the world's audit and fails the test with its error.
func auditNow(t testing.TB, w *World) {
	t.Helper()
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit() error = %v, want nil", err)
	}
}

// moneyTotal is the money over every account that is not a boundary.
func moneyTotal(w *World) int64 {
	var sum int64
	for id := range w.ledger.Len() {
		if v, err := w.ledger.Account(id); err == nil && v.Kind != ledger.KindBoundary {
			sum += v.Balance
		}
	}
	return sum
}

// home and yonder are the agents of creditConfig by role and region.
const (
	homeA, homeB, homeC, homeWeaver = AgentID(2), AgentID(3), AgentID(4), AgentID(5)
	yonderA, yonderB                = AgentID(6), AgentID(7)
	yonderWeaver                    = AgentID(8)
)

// wantOpenClaim fails the test unless the claim is open, overdue from the tick and
// owes the principal and the interest.
func wantOpenClaim(t testing.TB, w *World, id, overdue, principal, interest int64) {
	t.Helper()
	c, open := w.ledger.Claim(id)
	if !open || c.Overdue != overdue || c.PrincipalOwed != principal || c.InterestOwed != interest {
		t.Errorf("claim %d = %+v (open %v), want overdue from %d and owing %d and %d", id, c, open, overdue, principal, interest)
	}
}

// wantOne fails the test unless the log holds exactly one event of the type of want
// and it equals want.
func wantOne[T Event](t testing.TB, rec *evLog, want T) {
	t.Helper()
	if got := of[T](rec); len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Errorf("%T events = %+v, want [%+v]", want, got, want)
	}
}
