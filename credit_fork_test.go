package economysim_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditLiveDoc is creditPackDoc with the smallest principal 1, so that the growers of
// pack one borrow from the cooks in the run: the default borrower asks for the gap of
// its job and the default lender offers what its own bids leave. The numbers below are
// probed on seed 1 over 200 ticks: the first loans are formed at tick 58 and 59, a
// handful fall due at the ticks 66 to 86, and the main wave of 202 claims falls due at
// tick 123; the removal of debtors from tick 115 passes through the waterfall.
var creditLiveDoc = strings.Replace(creditPackDoc, "min_principal = 5000", "min_principal = 1", 1)

// creditForgive is the call made between ticks in the case "after the calls" of
// TestFork_Credit: it halves the scale of region 0 and forgives every loan of its
// debtors.
func creditForgive(t *testing.T) func(w *economysim.World) {
	return func(w *economysim.World) {
		t.Helper()
		if err := w.SetRateScales([]economysim.RegionScale{{Region: 0, BP: 5_000}}); err != nil {
			t.Fatalf("SetRateScales error = %v, want nil", err)
		}
		n, err := w.ForgiveClaims([]economysim.ClaimFilter{{Region: 0, AllRoles: true}})
		if err != nil || n == 0 {
			t.Fatalf("ForgiveClaims = %d, %v, want some claims ended and nil", n, err)
		}
	}
}

// TestFork_Credit verifies that a world forked while claims are open runs on exactly
// as an uninterrupted one in every event, in the snapshot (the claim rows and the
// totals), in the audit and in the checkpoint bytes, at the ticks that matter: before
// any loan, with loans open, a tick before and at the first due tick, a tick before
// and at the main due tick, and after a SetRateScales and a ForgiveClaims call made
// between ticks (the scale is saved in tag 26 and the forgiven claims left the list).
//
// Setup: pack one, warm, with the credit tables of creditLiveDoc, seed 1 (probed, see
// creditLiveDoc), M = 30 ticks after the boundary (60 from the start, to cover the first loans). The original and two restores must
// agree.
//
// Requirement: FR-CRD-087, FR-CHK-050
func TestFork_Credit(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(creditLiveDoc), packOne...)
	opts := func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }
	cells := []struct {
		name   string
		n, m   int
		before func(*testing.T) func(*economysim.World)
		// want names the event the credit module must emit after the boundary, so the
		// fork carries claims through it.
		want string
	}{
		{"before any loan", 0, 60, nil, "LoanFormed"},
		{"loans open", 61, forkM, nil, "LoanEnded"},
		{"a tick before the first due", 65, forkM, nil, "LoanEnded"},
		{"the first due tick", 66, forkM, nil, "LoanEnded"},
		{"a tick before the main due", 122, forkM, nil, "CreditSettled"},
		{"the main due tick", 123, forkM, nil, "CreditSettled"},
		{"after the calls", 100, forkM, creditForgive, "LoanEnded"},
	}
	for _, c := range cells {
		fc := forkCase{
			Name: "credit " + c.name, Cfg: cfg, Seed: 1, N: c.n, M: c.m, Logs: true, Opts: opts,
			Inspect: func(t *testing.T, _, since []economysim.Event) {
				if !hasCreditEvent(since, c.want) {
					t.Errorf("credit %s: no %s after the boundary, want one so the fork carries the claims through it", c.name, c.want)
				}
			},
		}
		if c.before != nil {
			fc.Before = c.before(t)
		}
		forkAtN(t, fc)
	}
}

// hasCreditEvent reports whether the events hold one of the named credit event type.
func hasCreditEvent(events []economysim.Event, name string) bool {
	for _, e := range events {
		switch e.(type) {
		case economysim.LoanFormed:
			if name == "LoanFormed" {
				return true
			}
		case economysim.LoanEnded:
			if name == "LoanEnded" {
				return true
			}
		case economysim.CreditSettled:
			if name == "CreditSettled" {
				return true
			}
		}
	}
	return false
}

// TestCheckpoint_CreditRoundTripOpenClaims verifies that a checkpoint taken while claims
// are open restores to a world with the same claim rows and totals in its snapshot and
// the same bytes in its own checkpoint.
//
// Setup: pack one with creditLiveDoc, seed 1, 100 ticks (loans open, the main due tick
// 123 ahead).
//
// Requirement: FR-CRD-087
func TestCheckpoint_CreditRoundTripOpenClaims(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(creditLiveDoc), packOne...)
	w := newPackWorld(t, cfg, 1, economysim.WithAudit())
	for i := range 100 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", i, err)
		}
	}
	want, err := w.Snapshot()
	if err != nil || len(want.Claims) == 0 {
		t.Fatalf("Snapshot() = %d claims, %v, want open claims and nil", len(want.Claims), err)
	}
	cp := mustCheckpoint(t, w)
	r, err := economysim.RestoreWorld(cfg, cp, economysim.WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	wantSameClaims(t, r, want)
	a, _ := cp.MarshalBinary()
	b, _ := mustCheckpoint(t, r).MarshalBinary()
	if !bytes.Equal(a, b) {
		t.Errorf("checkpoint of the restored world has %d bytes and differs from the original's %d", len(b), len(a))
	}
}

// wantSameClaims fails the test unless the snapshot of w holds the claim rows and the
// totals of want.
func wantSameClaims(t *testing.T, w *economysim.World, want economysim.Snapshot) {
	t.Helper()
	got, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() of the restored world error = %v, want nil", err)
	}
	if !reflect.DeepEqual(got.Claims, want.Claims) || got.Credit != want.Credit {
		t.Errorf("restored claims (%d rows) and totals %+v differ from the original's (%d rows, %+v)", len(got.Claims), got.Credit, len(want.Claims), want.Credit)
	}
}
