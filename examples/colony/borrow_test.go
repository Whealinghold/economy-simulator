package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestBorrower_BorrowsFallsDueAndAsksAgain verifies the borrowing player of borrow.go over
// the colony with the credit layer: after its first tick the player owes the 2 000 it asked
// for and the 8 of interest the lender's posted rate fixed at formation (2 008 owed, due 12
// ticks later), the claim is settled by the module and leaves nothing owed at some later tick, and the player then asks again and owes again. The books pass the
// audit at every step, because the world is built with it.
//
// Requirement: FR-CRD-055, FR-CRD-001
func TestBorrower_BorrowsFallsDueAndAsksAgain(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	var out bytes.Buffer
	if err := Borrower(&out, 1, 60); err != nil {
		t.Fatalf("Borrower(seed 1, 60 ticks) = %v, want nil", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 60 {
		t.Fatalf("Borrower printed %d lines, want one a tick (60)", len(lines))
	}
	if want := "tick 1 player money 2119 debt 2000 owed 2008 next due 12"; lines[0] != want {
		t.Errorf("first line = %q, want %q (the loan formed at the first round)", lines[0], want)
	}
	repaid := -1
	for i, l := range lines {
		if strings.HasSuffix(l, "debt 0 owed 0 next due -1") {
			repaid = i
			break
		}
	}
	if repaid < 0 {
		t.Fatalf("no line with nothing owed in 60 ticks, want the claim settled once: %q", lines)
	}
	if !strings.Contains(strings.Join(lines[repaid+1:], "\n"), "debt 2000 owed 2008") {
		t.Errorf("after the repayment at line %d the player owes nothing again, want a second loan", repaid+1)
	}
}

// TestRun_BorrowFlagRunsThePlayer verifies that -borrow runs the player in place of the
// report, with the same seed giving the same bytes twice, and that a bad run is exit code 1.
//
// Requirement: FR-CRD-055
//
// Guard: the flag is the only way the command reaches borrow.go; a determinism loss in the
// directed path would show as two different outputs.
func TestRun_BorrowFlagRunsThePlayer(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	var a, b, errs bytes.Buffer
	if code := Run([]string{"-borrow", "-ticks", "15"}, &a, &errs); code != 0 {
		t.Fatalf("Run(-borrow) = %d, want 0 (stderr %q)", code, errs.String())
	}
	if code := Run([]string{"-borrow", "-ticks", "15"}, &b, &errs); code != 0 || a.String() != b.String() {
		t.Errorf("Run(-borrow) twice = %d and different outputs, want 0 and the same bytes", code)
	}
	if !strings.HasPrefix(a.String(), "tick 1 player money") {
		t.Errorf("Run(-borrow) printed %q, want the player's lines", a.String())
	}
}
