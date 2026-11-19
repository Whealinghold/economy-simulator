package economysim

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestRound_TakesNoWord verifies that the labour round leaves the world's random
// stream where it was: the state of the PCG after a round that matches and forms a
// hire is the state it was keyed to before it.
//
// Requirement: FR-EMP-018, FR-EMP-061
//
// Guard: the module is the only one switched on; a draw in the round would move every
// later site's words only in worlds with the module on, which no off golden can see.
func TestRound_TakesNoWord(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{hands: 2}), farmWants(2, 100, 10))
	i := slicesIndexRow(t, "P2.5a")
	if err := runRows(w, w.Tick(), replayRows[:i], func(string) {}); err != nil {
		t.Fatalf("positions up to P2.5: %v", err)
	}
	w.rnd.keyed(siteWork, 3, 4)
	before, err := w.rnd.pcg.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	w.p25aLabour()
	if n := len(w.employment.hires); n != 2 {
		t.Fatalf("hires after the round = %d, want 2: the round did not run, the test would prove nothing", n)
	}
	after, err := w.rnd.pcg.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("PCG state after P2.5a = %x, want %x: the round took a word", after, before)
	}
}

// slicesIndexRow returns the index of the replay row of a position.
func slicesIndexRow(t *testing.T, id string) int {
	t.Helper()
	for i := range replayRows {
		if replayRows[i].id == id {
			return i
		}
	}
	t.Fatalf("no replay row %q", id)
	return 0
}

// TestRound_OffersWithTheModuleOffAreRefusedAtDebug verifies that an offer and a
// vacancy returned with the module off are refused employment_off at Debug only: no
// state, no hire, no error.
//
// Requirement: FR-EMP-001, FR-EMP-010, FR-EMP-011
func TestRound_OffersWithTheModuleOffAreRefusedAtDebug(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	lg := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	w := hireWorld(t, defaultHire(hireOpts{off: true}), farmWants(1, 100, 40), WithLogger(lg))
	stepN(t, w, 2)
	if w.employment != nil {
		t.Errorf("employment state = %v, want nil with the module off", w.employment)
	}
	if got := strings.Count(buf.String(), "reason=employment_off"); got != 4 {
		t.Errorf("Debug lines with employment_off = %d, want 4 (an offer and a vacancy a tick for two ticks)", got)
	}
}

// TestHire_PayrollRuleDoesNotRunInTheHireForm verifies that P4.1b pays nothing in the
// hire form: a farm with receipts and a hand with a working tick are left as they
// were, so a hire is not paid twice (spec-employment.md 24.3, one form per world).
//
// Requirement: FR-EMP-001, FR-EMP-064
func TestHire_PayrollRuleDoesNotRunInTheHireForm(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{handRecipe: true}), nil)
	farm, hand := slotsOf(w, "farm")[0], slotsOf(w, "hand")[0]
	w.welfare.addReceipt(farm, 1_000)
	w.welfare.markWorking(hand, 1, false)
	if err := w.p41bPayroll(); err != nil {
		t.Fatalf("p41bPayroll() error = %v, want nil", err)
	}
	if got := balanceOf(t, w, hand); got != 0 {
		t.Errorf("worker balance after P4.1b in the hire form = %d, want 0", got)
	}
}

// TestEscrow_AccountIsReusedAndSeenByTheFreeCheck verifies the Escrow account of a
// hire: the account of a hire in progress is the lowest-ID open Escrow account of the
// region and is not a "free" account for the merchants module's audit (markEscrow), the
// next hire takes the account the first one released rather than opening another, and
// no account is ever closed.
//
// Requirement: FR-EMP-024, FR-EMP-040
func TestEscrow_AccountIsReusedAndSeenByTheFreeCheck(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{duration: 1}), farmWants(1, 100, 40))
	stepN(t, w, 1)
	first := w.ledger.Len()
	// The hire of a one-tick job has ended at P2.7; the account is free and empty.
	if n := len(w.employment.hires); n != 0 {
		t.Fatalf("hires open after a one-tick job = %d, want 0", n)
	}
	stepN(t, w, 1)
	if got := w.ledger.Len(); got != first {
		t.Errorf("accounts after the second hire = %d, want %d: the released Escrow account is reused", got, first)
	}
	w2 := hireWorld(t, defaultHire(hireOpts{}), farmWants(1, 100, 40))
	stepN(t, w2, 1)
	used := map[int]bool{}
	if msg := w2.freeEscrowFault(used); msg == "" {
		t.Errorf("freeEscrowFault with no account marked = %q, want a fault: the hire's account holds money", msg)
	}
	w2.employment.markEscrow(used)
	if msg := w2.freeEscrowFault(used); msg != "" {
		t.Errorf("freeEscrowFault with the hire's account marked = %q, want none", msg)
	}
}

// TestSnapshot_HireEscrowIsInTheContractSector verifies that the money held in a
// hire's Escrow account is counted in the contract sector of the snapshot, in the
// total in escrow and in the money of the world, so the sectoral identity holds with a
// hire in progress (spec-employment.md 7.4.1).
//
// Arithmetic: the farm holds 10 000 and the bill is 210, of which 70 was paid at P2.7:
// 140 is in escrow after tick 0, and the household money 10 000 - 210 + 70 = 9 860, so
// the money of the world is 9 860 + 140 = 10 000.
//
// Requirement: FR-EMP-039
func TestSnapshot_HireEscrowIsInTheContractSector(t *testing.T) {
	t.Parallel()
	w := hireWorld(t, defaultHire(hireOpts{}), farmWants(1, 100, 40))
	stepN(t, w, 1)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	got := []Money{s.Escrow, s.Regions[0].MoneyContract, s.Regions[0].MoneyHouseholds, s.Money}
	if want := []Money{140, 140, 9_860, 10_000}; !equalMoney(got, want) {
		t.Errorf("snapshot escrow, contract sector, households, world money = %v, want %v", got, want)
	}
}

// equalMoney reports whether two lists of money are equal.
func equalMoney(a, b []Money) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// FuzzFormation verifies that no scripted offer and vacancy, however wrong, creates
// or loses money or stops a step: the audit is on, so a hire that broke an identity
// fails the run, and the household money plus the escrow is the money the world
// started with.
//
// Requirement: FR-EMP-036, FR-EMP-042, NFR-EMP-003
func FuzzFormation(f *testing.F) {
	f.Add(uint8(2), int64(2), int64(100), int64(40), uint8(3), uint16(1_000))
	f.Add(uint8(3), int64(5), int64(7), int64(7), uint8(1), uint16(10))
	f.Add(uint8(1), int64(-1), int64(0), int64(-5), uint8(2), uint16(0))
	f.Fuzz(func(t *testing.T, hands uint8, positions, bid, reserve int64, duration uint8, money uint16) {
		o := defaultHire(hireOpts{hands: int64(hands%4) + 1, duration: int64(duration%5) + 1, farmMoney: Money(money) + 1})
		w := hireWorld(t, o, farmWants(positions, Price(bid), Price(reserve)))
		total := totalMoney(t, w)
		for range 6 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
			if got := totalMoney(t, w) + Money(escrowHeld(t, w)); got != total {
				t.Fatalf("household money plus escrow = %d, want %d", got, total)
			}
		}
	})
}
