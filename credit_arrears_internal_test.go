package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// stepSet sets the balances of the agents named (everyone else holds 0) and steps the
// world one tick, so that P1.7a reads exactly those balances: nothing before it in
// the step moves money in the credit worlds.
func stepSet(t *testing.T, w *World, want map[AgentID]int64) {
	t.Helper()
	setBalances(t, w, want)
	step(t, w, 1)
}

// TestSettle_ArrearsAndWriteOff verifies the life of a claim a debtor cannot pay:
// at its due tick it is marked overdue and counted as defaulted, with its owed left
// as it was (no penalty and no interest on arrears); a later tick pays what the pot
// allows, interest first, and does not default it again; and at Due + WriteOffTicks
// it is written off with the cause written_off and what it still owed as the loss.
//
// Arithmetic: the loan is 1 000 of principal and 100 of interest, due at tick 2, with
// ShieldTicks 0 and WriteOffTicks 2. At tick 2 the debtor holds 0 and pays nothing,
// and the claim is overdue from 2 and owes 1 100. At tick 3 it holds 400: 100 pays
// the interest and 300 the principal, so 700 of principal is owed. At tick 4 (2 + 2)
// it holds 0 and the 700 are written off.
//
// Requirement: FR-CRD-034, FR-CRD-035, FR-CRD-036, FR-CRD-071, FR-CRD-073
func TestSettle_ArrearsAndWriteOff(t *testing.T) {
	t.Parallel()
	w, rec := settleAt(t, 0, 2)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 1_000})
	id := plantLoan(t, w, homeWeaver, homeA, 1_000, 100, 2)
	step(t, w, 1) // tick 2: the claim is due
	rec.events = nil

	stepSet(t, w, map[AgentID]int64{})
	wantOpenClaim(t, w, id, 2, 1_000, 100)
	wantOne(t, rec, CreditSettled{Tick: 2, Region: 0, Due: 1, Defaulted: 1, Arrears: 1_100})

	rec.events = nil
	stepSet(t, w, map[AgentID]int64{homeA: 400})
	wantOpenClaim(t, w, id, 2, 700, 0)
	wantOne(t, rec, CreditSettled{Tick: 3, Region: 0, PrincipalPaid: 300, InterestPaid: 100, Received: 400, Arrears: 700})

	rec.events = nil
	stepSet(t, w, map[AgentID]int64{})
	if _, _, open := claimOwed(t, w, id); open {
		t.Error("claim is open at Due + WriteOffTicks, want written off")
	}
	wantOne(t, rec, LoanEnded{Tick: 4, Claim: ClaimID(id), Origin: OriginLoan, Creditor: homeWeaver, Debtor: homeA, Principal: 1_000,
		LostPrincipal: 700, Cause: CauseWrittenOff})
	wantOne(t, rec, CreditSettled{Tick: 4, Region: 0, WrittenOff: 1, Lost: 700})
	if tot := w.ledger.ClaimTotals(); tot.LostPrincipal != 700 || tot.LostInterest != 0 {
		t.Errorf("lost totals = %d and %d, want 700 and 0", tot.LostPrincipal, tot.LostInterest)
	}
}

// TestSettle_NeverWrittenOffWithZeroWriteOffTicks verifies that with WriteOffTicks 0
// an overdue claim stays in the list however long it is unpaid (here, six ticks, kept out of
// reach by the shield), and owes what it owed.
//
// Requirement: FR-CRD-035, FR-CRD-036
func TestSettle_NeverWrittenOffWithZeroWriteOffTicks(t *testing.T) {
	t.Parallel()
	// The shield is 1 000 ticks of upkeep, so the debtor, which holds 5 000 at every
	// tick, pays nothing and is neither removed for want of food nor repays.
	w, _ := settleAt(t, 1_000, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 1_000})
	id := plantLoan(t, w, homeWeaver, homeA, 1_000, 100, 2)
	for range 6 {
		stepSet(t, w, map[AgentID]int64{homeA: 5_000})
	}
	wantOpenClaim(t, w, id, 2, 1_000, 100)
}

// TestSettle_BothOrigins verifies that a payout claim is settled by the same pot and
// the same pro rata share as a loan, ends with its own LoanEnded of origin payout,
// and is not counted in CreditSettled, which counts loans only.
//
// Arithmetic: the debtor holds 1 500 with ShieldTicks 0 and owes a loan of 900 of
// principal and 100 of interest (1 000) and a payout claim of 1 000. The pot 1 500
// is shared 750 and 750: the loan pays its 100 of interest and 650 of principal, the
// payout claim 750 of principal. CreditSettled holds the loan's 650 and 100 only.
//
// Requirement: FR-CRD-031, FR-CRD-032, FR-CRD-071, FR-CRD-073
func TestSettle_BothOrigins(t *testing.T) {
	t.Parallel()
	w, rec := settleAt(t, 0, 0)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 900})
	loan := plantLoan(t, w, homeWeaver, homeA, 900, 100, 2)
	payout := plantPayout(t, w, homeB, homeA, 1_000, 2)
	step(t, w, 1)
	rec.events = nil
	setBalances(t, w, map[AgentID]int64{homeA: 1_500})
	w.p17aSettle()
	if p, i, _ := claimOwed(t, w, loan); p != 250 || i != 0 {
		t.Errorf("loan owes %d and %d, want 250 and 0", p, i)
	}
	if p, _, _ := claimOwed(t, w, payout); p != 250 {
		t.Errorf("payout claim owes %d, want 250", p)
	}
	wantOne(t, rec, CreditSettled{Tick: 2, Region: 0, Due: 1, Defaulted: 1, PrincipalPaid: 650, InterestPaid: 100, Received: 750, Arrears: 250}) // loans only
	if got := heldMoney(t, w, homeB); got != 750 {
		t.Errorf("creditor of the payout claim holds %d, want 750", got)
	}
}

// TestSettle_HeadroomCutsAPaymentAtTheCreditor verifies that a payment that would take
// the creditor above MaxMoney is cut to its headroom, that the part not paid stays
// owed, and that CreditSettled counts the cut.
//
// Arithmetic: the weaver starts with MaxMoney - 100 and lends 3 000, so it holds
// MaxMoney - 3 100 and has 3 100 of room. The debtor owes 3 000 of principal and 300
// of interest and holds more, so it would pay 3 300; the payment is cut to 3 100, interest first
// (300) and then 2 800 of principal, and 200 are cut and stay owed.
//
// Requirement: FR-CRD-033, FR-CRD-074
func TestSettle_HeadroomCutsAPaymentAtTheCreditor(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, creditConfig(0, 0), WithSeed(exitSeed), WithObserver(rec), WithAudit())
	// A role's start money is bound far below MaxMoney, so the weaver is raised from
	// the pool of its region, seeded while the ledger is in construction.
	if err := w.estates.seedPool(w.ledger, 0, MaxMoney-1_000); err != nil {
		t.Fatalf("setup: seedPool error = %v, want nil", err)
	}
	putWeaver := func(target int64) {
		t.Helper()
		weaver, pool := w.agents[homeWeaver].account, w.estates.account(0)
		var err error
		if delta := target - heldMoney(t, w, homeWeaver); delta > 0 {
			err = w.ledger.Transfer(pool, weaver, delta, ledger.StartMoney)
		} else if delta < 0 {
			err = w.ledger.Transfer(weaver, pool, -delta, ledger.EstateMoney)
		}
		if err != nil {
			t.Fatalf("setup: set the weaver to %d: %v", target, err)
		}
	}
	step(t, w, 1)
	putWeaver(3_000)
	id := plantLoan(t, w, homeWeaver, homeA, 3_000, 300, 2)
	step(t, w, 1)
	rec.events = nil
	if have := heldMoney(t, w, homeA); have < 3_300 {
		t.Fatalf("debtor holds %d, want at least the 3 300 it owes", have)
	}
	putWeaver(MaxMoney - 3_100) // 3 100 of room: what the 3 000 lent and 100 more left it
	before := heldMoney(t, w, homeWeaver)
	w.p17aSettle()
	if p, i, open := claimOwed(t, w, id); !open || p != 200 || i != 0 {
		t.Errorf("claim owes %d and %d (open %v), want 200 and 0: the 200 that did not fit", p, i, open)
	}
	if got := heldMoney(t, w, homeWeaver); got != MaxMoney || got-before != 3_100 {
		t.Errorf("creditor holds %d (a rise of %d), want MaxMoney and a rise of 3 100", got, got-before)
	}
	wantOne(t, rec, CreditSettled{Tick: 2, Region: 0, Due: 1, Defaulted: 1, PrincipalPaid: 2_800, InterestPaid: 300, Headroom: 200, Received: 3_100, Arrears: 200})
	auditNow(t, w)
}

// TestSettle_AwayDebtor verifies that a debtor that is away on a trip is settled at
// P1.7a like any other: the pass reads no flag of the agent, only its account.
//
// Requirement: FR-CRD-045
func TestSettle_AwayDebtor(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, withTransport(creditConfig(0, 0)), WithSeed(exitSeed), WithObserver(rec), WithAudit())
	step(t, w, 1)
	setBalances(t, w, map[AgentID]int64{homeWeaver: 500})
	id := plantLoan(t, w, homeWeaver, homeA, 500, 0, 2)
	step(t, w, 1)
	w.transport.setAway(homeA, true)
	setBalances(t, w, map[AgentID]int64{homeA: 500})
	w.p17aSettle()
	if _, _, open := claimOwed(t, w, id); open {
		t.Error("the claim of an away debtor is open after its due tick, want repaid")
	}
}
