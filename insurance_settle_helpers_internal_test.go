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
