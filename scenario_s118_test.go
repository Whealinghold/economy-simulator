package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS118_CallsAndCheckpoint verifies CRD-S5 of spec-credit.md 18.3 on the credit
// pack, seed 1, the audit on. At tick 100 (the first loans are open, none due: tick 58 plus
// the term) SetRateScales halves the scale of region 0 and ForgiveClaims ends every open loan
// of it. Exact: the claim rows are gone and nothing is left to settle, no money moves, the
// principal and the interest lost rise by what was owed, and the identities hold before and
// after the calls (the events of the call, their order and their tick are the tests of
// TestCredit_ForgiveClaimsOverARun). The original and two restores of a checkpoint taken just
// after the calls then agree in every event, snapshot, audit and byte for 60 ticks, which
// takes the main due tick (the scale is saved in tag 26). A checkpoint taken at tick 122
// with claims open, a tick before the main due, restores a world that settles them as the
// original does (FR-CRD-087).
//
// Requirement: FR-CRD-060, FR-CRD-061, FR-CRD-071, FR-CRD-087
func TestScenarioS118_CallsAndCheckpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	cfg := loadValidPack(t, packOneCredit...)
	opts := func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }
	t.Run("after the calls", func(t *testing.T) {
		t.Parallel()
		forkAtN(t, forkCase{
			Name: "S118 after the calls", Cfg: cfg, Seed: 1, N: 100, M: 60, Logs: true, Opts: opts,
			Before: func(w *economysim.World) { callsOfS118(t, w) },
		})
	})
	t.Run("claims open at the boundary", func(t *testing.T) {
		t.Parallel()
		forkAtN(t, forkCase{
			Name: "S118 open claims", Cfg: cfg, Seed: 1, N: 122, M: 40, Logs: true, Opts: opts,
			Inspect: func(t *testing.T, _, since []economysim.Event) {
				if !hasCreditEvent(since, "CreditSettled") {
					t.Errorf("S118 open claims: no CreditSettled after the boundary, want the main due tick settled in the fork")
				}
			},
		})
	})
}

// callsOfS118 makes the two calls of S118 on a world at a boundary and checks what they must
// do exactly: the event of the scale, the ended claims and their order, the money, the lost
// totals and the identities.
func callsOfS118(t *testing.T, w *economysim.World) {
	t.Helper()
	before, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() before the calls error = %v, want nil", err)
	}
	if len(before.Claims) == 0 {
		t.Fatalf("no claim open at tick %d, want the loans of the first job starts", before.Tick)
	}
	checkClaimIdentities(t, "S118 before the calls", before)
	if err := w.SetRateScales([]economysim.RegionScale{{Region: 0, BP: 5_000}}); err != nil {
		t.Fatalf("SetRateScales error = %v, want nil", err)
	}
	n, err := w.ForgiveClaims([]economysim.ClaimFilter{{Region: 0, AllRoles: true}})
	if err != nil || n != int64(len(before.Claims)) {
		t.Fatalf("ForgiveClaims = %d, %v, want %d (every open claim) and nil", n, err, len(before.Claims))
	}
	after, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() after the calls error = %v, want nil", err)
	}
	checkClaimIdentities(t, "S118 after the calls", after)
	checkForgiven(t, before, after)
}

// checkForgiven checks that the snapshot after ForgiveClaims has no claim row, the same money
// and lost totals raised by exactly what the rows before it owed.
func checkForgiven(t *testing.T, before, after economysim.Snapshot) {
	t.Helper()
	owedP, owedI := owedOn(before.Claims)
	lostP, lostI := after.Credit.LostPrincipal-before.Credit.LostPrincipal, after.Credit.LostInterest-before.Credit.LostInterest
	if len(after.Claims) != 0 || after.Money != before.Money || lostP != owedP || lostI != owedI {
		t.Errorf("after the calls: %d claim rows, money %d (was %d), lost rose by %d and %d, want 0 rows, no money moved and lost raised by the owed %d and %d",
			len(after.Claims), after.Money, before.Money, lostP, lostI, owedP, owedI)
	}
}
