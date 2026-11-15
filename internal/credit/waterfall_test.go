package credit_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/credit"
)

// TestPay_Shield verifies the pot of a debtor: the balance less the shield, never
// below 0 and never above what is owed, with the shield product saturating at
// MaxMoney. The three rows of the table of spec-credit.md 20.1 are the first
// three cases.
//
// Requirement: FR-CRD-031
func TestPay_Shield(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		balance, ticks, upkeep int64
		owed, want             int64
		ok                     bool
	}{
		{"pays it all", 7_500, 1, 100, 5_761, 5_761, true},
		{"pays the pot", 3_000, 1, 100, 5_761, 2_900, true},
		{"below the shield pays nothing", 80, 1, 100, 5_761, 0, true},
		{"exactly the shield pays nothing", 100, 1, 100, 5_761, 0, true},
		{"no shield", 80, 0, 100, 5_761, 80, true},
		{"no upkeep, no shield", 80, 5, 0, 5_761, 80, true},
		{"nothing owed", 80, 0, 100, 0, 0, true},
		{"no balance", 0, 1, 100, 50, 0, true},
		{"the shield product saturates at MaxMoney", maxMoney, math.MaxInt64, 2, 10, 0, true},
		{"negative balance", -1, 1, 1, 1, 0, false},
		{"negative shield", 1, -1, 1, 1, 0, false},
		{"negative upkeep", 1, 1, -1, 1, 0, false},
		{"negative owed", 1, 1, 1, -1, 0, false},
	}
	for _, tt := range tests {
		got, ok := credit.Pay(tt.balance, tt.ticks, tt.upkeep, tt.owed, maxMoney)
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s: Pay(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.name, tt.balance, tt.ticks, tt.upkeep, tt.owed, got, ok, tt.want, tt.ok)
		}
	}
}

// TestPay_MaxMoney verifies the saturation of the shield at MaxMoney and the
// refusal of a negative MaxMoney: with MaxMoney 100 a shield of 1 000 is 100, so a
// balance of 150 has a pot of 50.
//
// Requirement: FR-CRD-031
func TestPay_MaxMoney(t *testing.T) {
	t.Parallel()
	if _, ok := credit.Pay(1, 0, 0, 1, -1); ok {
		t.Errorf("Pay with a negative MaxMoney = ok, want refused")
	}
	// A shield product that fits int64 and is above MaxMoney is saturated at it: with
	// MaxMoney 100 the shield of 1 000 is 100, so a balance of 150 has a pot of 50.
	if got, ok := credit.Pay(150, 1, 1_000, 500, 100); got != 50 || !ok {
		t.Errorf("Pay(150, 1, 1000, 500, MaxMoney 100) = %d, %v, want 50, true", got, ok)
	}
	if got, ok := credit.Pay(150, 1, 100, 500, 100); got != 50 || !ok {
		t.Errorf("Pay with a shield equal to MaxMoney = %d, %v, want 50, true", got, ok)
	}
	if got, ok := credit.Pay(150, 1, 5, 500, 0); got != 150 || !ok {
		t.Errorf("Pay with a MaxMoney of 0 = %d, %v, want 150, true: a MaxMoney of 0 is accepted and caps the shield at 0", got, ok)
	}
}

// TestWaterfall_Example verifies the waterfall of spec-credit.md 20.2: a balance
// of 800 over claims of 500 and 700 pays 333 and 467 and leaves nothing for the
// estate, and a balance of 1 500 pays both in full and leaves 300.
//
// Requirement: FR-CRD-040, FR-CRD-041
func TestWaterfall_Example(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		balance  int64
		owed     []int64
		want     []int64
		wantLeft int64
	}{
		{"short of the debts", 800, []int64{500, 700}, []int64{333, 467}, 0},
		{"pays both and the rest goes to the estate", 1_500, []int64{500, 700}, []int64{500, 700}, 300},
		{"no debt", 90, nil, []int64{}, 90},
		{"no balance", 0, []int64{500, 700}, []int64{0, 0}, 0},
	}
	var s credit.Scratch
	for _, tt := range tests {
		got, left, ok := s.Waterfall(nil, tt.balance, tt.owed, make([]uint64, len(tt.owed)))
		if !ok || left != tt.wantLeft || !slices.Equal(got, tt.want) {
			t.Errorf("%s: Waterfall(%d, %v) = %v, %d, %v, want %v, %d", tt.name, tt.balance, tt.owed, got, left, ok, tt.want, tt.wantLeft)
		}
	}
}

// TestWaterfall_Refused verifies a negative balance or owed, a short tie and an
// overflowing sum are refused with no shares.
//
// Requirement: FR-CRD-040, NFR-CRD-005
func TestWaterfall_Refused(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		balance int64
		owed    []int64
		tie     []uint64
	}{
		{"negative balance", -1, []int64{1}, []uint64{0}},
		{"negative owed", 1, []int64{-1}, []uint64{0}},
		{"short tie", 1, []int64{1, 1}, []uint64{0}},
		{"sum overflows", 1, []int64{math.MaxInt64, 1}, []uint64{0, 0}},
	}
	var s credit.Scratch
	for _, tt := range tests {
		if got, left, ok := s.Waterfall(nil, tt.balance, tt.owed, tt.tie); ok || got != nil || left != 0 {
			t.Errorf("%s: Waterfall = %v, %d, %v, want nil, 0, false", tt.name, got, left, ok)
		}
	}
}

// TestRoom_CapLessDebt verifies the exposure room of spec-credit.md 5.3.3: the cap
// of MaxDebtTicks times the upkeep cost less the debt, never below 0, saturating at
// maxMoney when the product overflows or passes it, and refused for a negative
// argument.
//
// Requirement: FR-CRD-015
func TestRoom_CapLessDebt(t *testing.T) {
	t.Parallel()
	const maxMoney = 1_000_000
	tests := []struct {
		name                          string
		ticks, upkeep, debt, wantRoom int64
		wantOK                        bool
	}{
		{"nothing owed", 90, 100, 0, 9_000, true},
		{"some debt", 90, 100, 2_500, 6_500, true},
		{"debt at the cap", 90, 100, 9_000, 0, true},
		{"debt above the cap after a fall in weight", 90, 100, 12_000, 0, true},
		{"no upkeep", 90, 0, 0, 0, true},
		{"the cap passes maxMoney", 3_650, 1_000_000, 10, maxMoney - 10, true},
		{"the product overflows int64", math.MaxInt64, math.MaxInt64, 0, maxMoney, true},
		{"negative debt", 1, 1, -1, 0, false},
		{"negative ticks", -1, 1, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			room, ok := credit.Room(tt.ticks, tt.upkeep, tt.debt, maxMoney)
			if room != tt.wantRoom || ok != tt.wantOK {
				t.Errorf("Room(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.ticks, tt.upkeep, tt.debt, maxMoney, room, ok, tt.wantRoom, tt.wantOK)
			}
		})
	}
}
