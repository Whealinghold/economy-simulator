package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the helpers of the settlement tests (plan 3-ins Phase 6): a world of
// insConfig with route policies formed at tick 0 for shipments that arrive at tick 2, and
// readers of the events and of the welfare records.

// bigOffer is the offer of the settlement tests: the fair offer with a liability that
// covers two shipments of the tests.
var bigOffer = CoverOffer{MaxLiability: 20_000, LoadingBP: 10_000, StoreRateBP: 100, CoverBP: 8_000}

// settleWorld builds the world of c, dispatches at tick 0 one coach shipment of each of the
// quantities of grain, the first for insHome and the second for insHome2, and forms a
// route policy for each with insBacker as the underwriter. The shipments arrive at tick 2.
func settleWorld(t *testing.T, c Config, qtys ...int64) (*World, *evLog) {
	t.Helper()
	w, rec := newInsWorld(t, c, nil)
	for i, q := range qtys {
		id := []AgentID{insHome, insHome2}[i]
		insDispatch(t, w, id, q)
		planCover(w, id, coachLane)
	}
	planOffer(w, insBacker, bigOffer)
	w.p33cCoverRound()
	if got := len(w.insurance.open); got != len(qtys) {
		t.Fatalf("policies formed = %d, want %d", got, len(qtys))
	}
	return w, rec
}

// moneyOf sets the money of an agent: what it has too much goes to the estate of its
// region, what it lacks comes through the Initial boundary.
func moneyOf(t *testing.T, w *World, id AgentID, n int64) {
	t.Helper()
	a := &w.agents[id]
	have := heldMoney(t, w, id)
	if have > n {
		if err := w.ledger.Transfer(a.account, w.estates.account(w.regionOf(a)), have-n, ledger.EstateMoney); err != nil {
			t.Fatalf("moneyOf(%d, %d): %v", id, n, err)
		}
		return
	}
	setMoney(t, w, a, n)
}

// welfareRecords returns the welfare records of an agent, oldest first.
func welfareRecords(w *World, id AgentID) []welfare.Record {
	return w.agents[id].history.AppendRecords(nil)
}

// openClaim returns the open claim with an ID.
func openClaim(t *testing.T, w *World, id int64) ledger.Claim {
	t.Helper()
	c, ok := w.ledger.Claim(id)
	if !ok {
		t.Fatalf("claim %d is not open", id)
	}
	return c
}

// coverEndedAt and coverSettledAt return the events of a tick, in the order told.
func coverEndedAt(rec *evLog, tick int64) []CoverEnded {
	return slices.DeleteFunc(of[CoverEnded](rec), func(e CoverEnded) bool { return e.Tick != tick })
}

func coverSettledAt(rec *evLog, tick int64) []CoverSettled {
	return slices.DeleteFunc(of[CoverSettled](rec), func(e CoverSettled) bool { return e.Tick != tick })
}

// only returns the one element of a list, failing the test when there is not exactly one.
func only[T any](t *testing.T, name string, got []T) T {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("%s = %+v, want exactly one", name, got)
	}
	return got[0]
}

// figures are the money figures of a CoverEnded: due, paid, short, dropped and cut.
func figures(e CoverEnded) [5]Money { return [5]Money{e.Due, e.Paid, e.Short, e.Dropped, e.Cut} }

// inOrder reports whether the last events of the three types are in that order in the log.
func inOrder[A, B, C Event](rec *evLog) bool {
	a, b, c := indexOfEvent[A](rec), indexOfEvent[B](rec), indexOfEvent[C](rec)
	return a < b && b < c
}

// moneyOfAgents reads the money of the agents, in the order given.
func moneyOfAgents(t *testing.T, w *World, ids ...AgentID) []int64 {
	t.Helper()
	out := make([]int64, len(ids))
	for i, id := range ids {
		out[i] = heldMoney(t, w, id)
	}
	return out
}

// shape is the state of the module's lists for an underwriter and an insured: open route
// policies, store policies, the underwriter's liability and the insured's policies.
func shape(w *World, under, insured AgentID) [4]int64 {
	in := w.insurance
	return [4]int64{int64(len(in.open)), int64(len(in.stores)), in.liabilityOf(under), in.policiesOf(insured)}
}
