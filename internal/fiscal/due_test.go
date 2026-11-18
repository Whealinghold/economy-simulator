package fiscal_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

const maxMoney = int64(1) << 60

// TestDueFixed_Example verifies the head tax of spec-fiscal.md 18.1: a fixed tax
// of 60 a tick at scale 10 000 collects 60 from a household holding 3 000, and
// from a household holding 40 it collects 40 and forgives 20.
//
// The carry starts at (id mod period) * 10^4 = 0 for a tick period, so
// num = 60 * 10 000 + 0 = 600 000 and due = 600 000 / 10 000 = 60.
//
// Requirement: FR-FIS-021, FR-FIS-024
func TestDueFixed_Example(t *testing.T) {
	t.Parallel()
	rule := fiscal.Rule{Shape: fiscal.Fixed, Amount: 60, Scale: 10_000, Period: 1, MaxMoney: maxMoney}
	tests := []struct {
		name        string
		bal         int64
		wantDue     int64
		wantCollect int64
		wantShort   int64
	}{
		{"rich household pays in full", 3_000, 60, 60, 0},
		{"household holding 40 pays 40 and 20 is forgiven", 40, 60, 40, 20},
		{"household holding 25 pays 25 and 35 is forgiven", 25, 60, 25, 35},
		{"empty household pays nothing", 0, 60, 0, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := rule.Due(1, tt.bal, rule.StartCarry(7))
			if !ok || got.Due != tt.wantDue || got.Collected != tt.wantCollect || got.Short() != tt.wantShort {
				t.Errorf("Due(1, %d, 0) = %+v, %v; want due %d, collected %d, short %d",
					tt.bal, got, ok, tt.wantDue, tt.wantCollect, tt.wantShort)
			}
		})
	}
}

// TestDueRate_Example verifies the holding tax of spec-fiscal.md 18.1: 100 bp a
// tick of the balance above an exemption of 2 000. A household holding 2 940
// has a base of 940; 940 * 100 * 10^4 = 9.4 * 10^8, so the due is 9 and the
// carry 4 * 10^7.
//
// Requirement: FR-FIS-021
func TestDueRate_Example(t *testing.T) {
	t.Parallel()
	rule := fiscal.Rule{Shape: fiscal.Rate, RateBP: 100, Exempt: 2_000, Scale: 10_000, Period: 1, MaxMoney: maxMoney}
	got, ok := rule.Due(1, 2_940, rule.StartCarry(12))
	want := fiscal.Result{Due: 9, Collected: 9, Carry: 40_000_000}
	if !ok || got != want {
		t.Errorf("Due(1, 2940, 0) = %+v, %v; want %+v", got, ok, want)
	}
	if got, _ := rule.Due(1, 1_500, 0); got.Due != 0 || got.Collected != 0 || got.Carry != 0 {
		t.Errorf("Due(1, 1500, 0) below the exemption = %+v, want nothing due", got)
	}
}

// TestDue_CarryAdvancesByDue verifies that the carry advances by the due and not
// by what was collected: a payer that could pay nothing still moves the carry,
// so what it could not pay is forgiven, not owed.
//
// A fixed tax of 5 000 over a period of 4 ticks, for the account 3, starts at
// (3 mod 4) * 10^4 = 30 000; a payer holding nothing has nothing above its
// exemption, so it collects 0, yet the carry advances as for a payer that paid.
//
// Requirement: FR-FIS-021, FR-FIS-024
func TestDue_CarryAdvancesByDue(t *testing.T) {
	t.Parallel()
	rule := fiscal.Rule{Shape: fiscal.Fixed, Amount: 5_000, Scale: 10_000, Period: 4, MaxMoney: maxMoney}
	carry := rule.StartCarry(3) // (3 mod 4) * 10^4 = 30 000
	if carry != 30_000 {
		t.Fatalf("StartCarry(3) = %d, want 30000", carry)
	}
	// num = 5000 * 10000 + 30000 = 50 030 000, den = 4 * 10^4 = 40 000:
	// due 1 250, carry 50 030 000 - 1250 * 40 000 = 30 000.
	got, ok := rule.Due(1, 0, carry)
	want := fiscal.Result{Due: 1_250, Collected: 0, Carry: 30_000}
	if !ok || got != want {
		t.Errorf("Due(1, 0, %d) = %+v, %v; want %+v: a payer holding nothing still advances the carry by the due", carry, got, ok, want)
	}
}

// TestRule_StartCarry verifies the carry a payer starts with: a fixed tax starts
// at (id mod period) * 10^4 and a rate tax at 0.
//
// Requirement: FR-FIS-021
func TestRule_StartCarry(t *testing.T) {
	t.Parallel()
	fixed := fiscal.Rule{Shape: fiscal.Fixed, Period: 365}
	rate := fiscal.Rule{Shape: fiscal.Rate, Period: 365}
	if got := fixed.StartCarry(366); got != 10_000 {
		t.Errorf("fixed StartCarry(366) = %d, want 10000 ((366 mod 365) * 10^4)", got)
	}
	if got := rate.StartCarry(366); got != 0 {
		t.Errorf("rate StartCarry(366) = %d, want 0", got)
	}
}

// TestDue_Domain verifies that a negative argument, a period below 1 and a
// product that leaves int64 are reported and never wrapped.
//
// Requirement: NFR-FIS-003
func TestDue_Domain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rule fiscal.Rule
		w    int64
		bal  int64
	}{
		{"negative weight", fiscal.Rule{Amount: 1, Scale: 1, Period: 1}, -1, 0},
		{"negative balance", fiscal.Rule{Amount: 1, Scale: 1, Period: 1}, 1, -1},
		{"negative exemption", fiscal.Rule{Amount: 1, Scale: 1, Period: 1, Exempt: -1}, 1, 0},
		{"period below 1", fiscal.Rule{Amount: 1, Scale: 1}, 1, 0},
		{"exemption times weight overflows", fiscal.Rule{Amount: 1, Scale: 1, Period: 1, Exempt: math.MaxInt64}, 2, 0},
		{"rate times scale overflows", fiscal.Rule{Shape: fiscal.Rate, RateBP: math.MaxInt64, Scale: 2, Period: 1}, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tt.rule.Due(tt.w, tt.bal, 0); ok {
				t.Errorf("Due(%d, %d, 0) = %+v, true; want ok false", tt.w, tt.bal, got)
			}
		})
	}
}

// TestHeadroom verifies the room a treasury has below MaxMoney, never negative.
//
// Requirement: FR-FIS-025
func TestHeadroom(t *testing.T) {
	t.Parallel()
	tests := []struct{ bal, max, want int64 }{
		{100, 1_000, 900},
		{1_000, 1_000, 0},
		{1_200, 1_000, 0},
	}
	for _, tt := range tests {
		if got := fiscal.Headroom(tt.bal, tt.max); got != tt.want {
			t.Errorf("Headroom(%d, %d) = %d, want %d", tt.bal, tt.max, got, tt.want)
		}
	}
}
