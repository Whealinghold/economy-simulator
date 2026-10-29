package ownuse_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ownuse"
)

// TestCover_MinOverGoods verifies that the cover is the least of
// floor(Stock / Upkeep) over the covered goods, so the scarcest good decides,
// and that a good that cannot be divided (no upkeep, a negative stock) or no
// good at all is refused.
//
// Requirement: FR-SUB-011
func TestCover_MinOverGoods(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		goods  []ownuse.Good
		want   int64
		wantOK bool
	}{
		// floor(7 / 3) = 2.
		{"one good", []ownuse.Good{{Stock: 7, Upkeep: 3, Price: 1}}, 2, true},
		// floor(10 / 2) = 5, floor(5 / 4) = 1: the scarcer good decides.
		{"least of two", []ownuse.Good{{Stock: 10, Upkeep: 2, Price: 1}, {Stock: 5, Upkeep: 4, Price: 1}}, 1, true},
		{"empty stock", []ownuse.Good{{Stock: 0, Upkeep: 3, Price: 1}}, 0, true},
		{"truncates toward zero", []ownuse.Good{{Stock: 8, Upkeep: 3, Price: 1}}, 2, true},
		{"no good", nil, 0, false},
		{"zero upkeep", []ownuse.Good{{Stock: 7, Upkeep: 0, Price: 1}}, 0, false},
		{"negative stock", []ownuse.Good{{Stock: -1, Upkeep: 3, Price: 1}}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ownuse.Cover(tt.goods)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Cover(%v) = %d, %v, want %d, %v", tt.goods, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestCost_Sum verifies that the cost of a tick of the covered upkeep is the
// sum of Upkeep * Price, and that an overflow of a product or of the sum, a
// price or upkeep below 1 and an empty set are refused rather than wrapped.
//
// Requirement: FR-SUB-011, NFR-SUB-002
func TestCost_Sum(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		goods  []ownuse.Good
		want   int64
		wantOK bool
	}{
		// 3 * 100 = 300 (spec-subsistence.md 16.1).
		{"one good", []ownuse.Good{{Stock: 7, Upkeep: 3, Price: 100}}, 300, true},
		// 3 * 100 + 1 * 40 = 340.
		{"two goods", []ownuse.Good{{Stock: 7, Upkeep: 3, Price: 100}, {Stock: 1, Upkeep: 1, Price: 40}}, 340, true},
		{"product overflows", []ownuse.Good{{Upkeep: math.MaxInt64, Price: 2}}, 0, false},
		{"sum overflows", []ownuse.Good{{Upkeep: 1, Price: math.MaxInt64}, {Upkeep: 1, Price: 1}}, 0, false},
		{"largest product fits", []ownuse.Good{{Upkeep: 3_037_000_499, Price: 3_037_000_499}}, 3_037_000_499 * 3_037_000_499, true},
		{"price below 1", []ownuse.Good{{Upkeep: 1, Price: 0}}, 0, false},
		{"upkeep below 1", []ownuse.Good{{Upkeep: 0, Price: 1}}, 0, false},
		{"no good", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ownuse.Cost(tt.goods)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Cost(%v) = %d, %v, want %d, %v", tt.goods, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestSecure_Saturates verifies that the secure ticks are cover plus afford,
// cut at the limit, and that a sum beyond int64 is cut as well.
//
// Requirement: FR-SUB-011
func TestSecure_Saturates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                       string
		cover, afford, limit, want int64
	}{
		{"below the limit", 2, 3, 1000, 5},
		{"at the limit", 400, 600, 1000, 1000},
		{"above the limit", 900, 900, 1000, 1000},
		{"sum overflows int64", math.MaxInt64, 1, 1000, 1000},
		{"afford alone above", 0, 5000, 1000, 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ownuse.Secure(tt.cover, tt.afford, tt.limit); got != tt.want {
				t.Errorf("Secure(%d, %d, %d) = %d, want %d", tt.cover, tt.afford, tt.limit, got, tt.want)
			}
		})
	}
}

// TestAfford_ZeroWhenHungry verifies that a household that went unfed affords
// nothing whatever its money (the market did not deliver, 5.2.1), that a fed
// one affords floor(money / cost), and that a bad cost or a negative money is
// refused for a fed household.
//
// Requirement: FR-SUB-012
func TestAfford_ZeroWhenHungry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		hungry      bool
		money, cost int64
		want        int64
		wantOK      bool
	}{
		{"hungry with money", true, 1_000_000, 300, 0, true},
		// floor(150 / 300) = 0, floor(900 / 300) = 3, floor(1000 / 300) = 3.
		{"fed cannot buy a tick", false, 150, 300, 0, true},
		{"fed buys three", false, 900, 300, 3, true},
		{"fed truncates", false, 1000, 300, 3, true},
		{"hungry ignores a bad cost", true, 5, 0, 0, true},
		{"fed zero cost", false, 5, 0, 0, false},
		{"fed negative money", false, -1, 300, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ownuse.Afford(tt.hungry, tt.money, tt.cost)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Afford(%v, %d, %d) = %d, %v, want %d, %v", tt.hungry, tt.money, tt.cost, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

var p23 = ownuse.Params{Enter: 2, Exit: 3, Max: 1000}

func one(stock int64) []ownuse.Good { return []ownuse.Good{{Stock: stock, Upkeep: 3, Price: 100}} }

// TestStep_EnterBelowEnter verifies that an agent out of the mode enters it when
// its secure ticks are below Enter (strictly), with the counter at 0, and does
// not when they equal Enter; and that a fed agent whose money buys ticks of
// upkeep stays out (the entry test of 4.4).
//
// Requirement: FR-SUB-013
func TestStep_EnterBelowEnter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		stock      int64
		hungry     bool
		money      int64
		jobs       int64
		wantSecure int64
		wantEnter  bool
	}{
		// Cover floor(7/3) = 2, afford 0: secure 2 is not below Enter 2.
		{"secure equals enter stays out", 7, false, 150, 5, 2, false},
		// Cover floor(4/3) = 1: secure 1 < 2.
		{"secure below enter enters", 4, false, 150, 5, 1, true},
		// Cover 1, afford floor(600/300) = 2: secure 3, the money buys the rest.
		{"money keeps a fed agent out", 4, false, 600, 5, 3, false},
		// The same money, but the agent went unfed: afford 0, secure 1.
		{"hungry agent ignores its money", 4, true, 600, 5, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, ok := ownuse.Step(p23, false, tt.jobs, one(tt.stock), tt.hungry, tt.money)
			if !ok || r.Secure != tt.wantSecure || r.Entered != tt.wantEnter || r.On != tt.wantEnter {
				t.Fatalf("Step(out, stock %d, hungry %v, money %d) = %+v, %v, want secure %d, entered %v", tt.stock, tt.hungry, tt.money, r, ok, tt.wantSecure, tt.wantEnter)
			}
			if tt.wantEnter && r.Jobs != 0 {
				t.Errorf("Step entered with Jobs = %d, want 0", r.Jobs)
			}
			if !tt.wantEnter && r.Jobs != tt.jobs {
				t.Errorf("Step stayed out with Jobs = %d, want %d unchanged", r.Jobs, tt.jobs)
			}
		})
	}
}

// TestStep_LeaveAtExitCoverOnly verifies that an agent in the mode leaves when
// its cover (stock alone) reaches Exit, whatever its money, and in no other
// case: lots of money with a short stock does not let it leave (5.3.2).
//
// Requirement: FR-SUB-014
func TestStep_LeaveAtExitCoverOnly(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		stock     int64
		money     int64
		wantLeave bool
	}{
		// floor(9/3) = 3 reaches Exit 3.
		{"cover reaches exit", 9, 0, true},
		{"cover above exit", 30, 0, true},
		// floor(8/3) = 2 is short, and 10^6 of money does not matter.
		{"rich but short of stock stays", 8, 1_000_000, false},
		{"empty stock stays", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, ok := ownuse.Step(p23, true, 4, one(tt.stock), false, tt.money)
			if !ok || r.Left != tt.wantLeave || r.On == tt.wantLeave || r.Entered {
				t.Fatalf("Step(in, stock %d, money %d) = %+v, %v, want left %v", tt.stock, tt.money, r, ok, tt.wantLeave)
			}
			wantJobs := int64(4)
			if tt.wantLeave {
				wantJobs = 0
			}
			if r.Jobs != wantJobs {
				t.Errorf("Step(in, stock %d) Jobs = %d, want %d", tt.stock, r.Jobs, wantJobs)
			}
		})
	}
}

// TestStep_BandAndCounterReset verifies the hysteresis band: an agent that
// entered with cover 1 stays in at cover 2 (between Enter and Exit) with its
// counter kept, leaves at 3 with the counter reset, and the same agent out of
// the mode at cover 2 does not enter.
//
// Requirement: FR-SUB-013, FR-SUB-014, FR-SUB-022
func TestStep_BandAndCounterReset(t *testing.T) {
	t.Parallel()
	in, _ := ownuse.Step(p23, true, 7, one(7), false, 0) // cover 2: in the band
	if !in.On || in.Left || in.Entered || in.Jobs != 7 {
		t.Errorf("Step(in, cover 2) = %+v, want still in with Jobs 7", in)
	}
	out, _ := ownuse.Step(p23, false, 7, one(7), false, 0) // cover 2, secure 2
	if out.On || out.Entered {
		t.Errorf("Step(out, cover 2) = %+v, want still out", out)
	}
	left, _ := ownuse.Step(p23, true, 7, one(9), false, 0) // cover 3
	if left.On || !left.Left || left.Jobs != 0 {
		t.Errorf("Step(in, cover 3) = %+v, want left with Jobs 0", left)
	}
}

// TestStep_WorkedExampleA replays the nine ticks of spec-subsistence.md 16.1
// (the Made column is the recipe's, not this package's): the stock at the start
// of each tick is 7, 4, 5, 5, 6, 7, 8, 8, 9 (the table's stock after eating of
// the tick before), upkeep 3, price 100, money 150, never hungry, Enter 2 and
// Exit 3. The cover is 2, 1, 1, 1, 2, 2, 2, 2, 3; the household enters at tick
// 2 (secure 1 < 2) and leaves at tick 9 (cover 3).
//
// Requirement: FR-SUB-013, FR-SUB-014
func TestStep_WorkedExampleA(t *testing.T) {
	t.Parallel()
	stocks := []int64{7, 4, 5, 5, 6, 7, 8, 8, 9}
	wantCover := []int64{2, 1, 1, 1, 2, 2, 2, 2, 3}
	wantOn := []bool{false, true, true, true, true, true, true, true, false}
	on := false
	for i, s := range stocks {
		r, ok := ownuse.Step(p23, on, 0, one(s), false, 150)
		if !ok || r.Cover != wantCover[i] || r.Afford != 0 || r.On != wantOn[i] {
			t.Errorf("tick %d: Step(on %v, stock %d) = %+v, %v, want cover %d, afford 0, on %v", i+1, on, s, r, ok, wantCover[i], wantOn[i])
		}
		on = r.On
	}
}

// TestStep_WorkedExampleB replays the cohort of spec-subsistence.md 16.2: 40
// households, ration upkeep 4 each so 160 for the agent, stock 200, money 300,
// price 12. Cover floor(200/160) = 1, cost 160 * 12 = 1 920, afford
// floor(300/1920) = 0, secure 1 < 2: it enters.
//
// Requirement: FR-SUB-011, FR-SUB-013
func TestStep_WorkedExampleB(t *testing.T) {
	t.Parallel()
	goods := []ownuse.Good{{Stock: 200, Upkeep: 160, Price: 12}}
	r, ok := ownuse.Step(p23, false, 0, goods, false, 300)
	want := ownuse.Result{Cover: 1, Afford: 0, Secure: 1, On: true, Entered: true}
	if !ok || r != want {
		t.Errorf("Step(cohort of 40) = %+v, %v, want %+v", r, ok, want)
	}
}

// TestStep_CohortEqualsUnits verifies that a cohort of weight w and an agent of
// weight 1 in the same circumstances have the same cover, afford and secure
// ticks: every term is a total for the agent (5.2). The cohort of 40 has stock
// 240, upkeep 160, money 3 840 at price 12; the unit has 6, 4 and 96:
// cover 1, cost 1 920 (cohort) and 48 (unit), afford 2 in both.
//
// Requirement: FR-SUB-016
func TestStep_CohortEqualsUnits(t *testing.T) {
	t.Parallel()
	cohort, _ := ownuse.Step(p23, false, 0, []ownuse.Good{{Stock: 240, Upkeep: 160, Price: 12}}, false, 3840)
	unit, _ := ownuse.Step(p23, false, 0, []ownuse.Good{{Stock: 6, Upkeep: 4, Price: 12}}, false, 96)
	if cohort != unit || cohort.Cover != 1 || cohort.Afford != 2 || cohort.Secure != 3 {
		t.Errorf("cohort %+v, unit %+v, want equal with cover 1, afford 2, secure 3", cohort, unit)
	}
}

// TestOverflow_ReturnsNotOK verifies that a figure that cannot be formed (a
// cost beyond int64, no good, a bad stock) makes Step report ok false, leave
// the mode and counter as they were and saturate the cover and the secure
// ticks at Max, so the household is never pushed into the mode by it.
//
// Requirement: NFR-SUB-002
func TestOverflow_ReturnsNotOK(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		on    bool
		goods []ownuse.Good
	}{
		{"cost overflows", false, []ownuse.Good{{Stock: 1, Upkeep: math.MaxInt64, Price: 2}}},
		{"cost overflows in the mode", true, []ownuse.Good{{Stock: 1, Upkeep: math.MaxInt64, Price: 2}}},
		{"no good", false, nil},
		{"negative stock", true, []ownuse.Good{{Stock: -5, Upkeep: 1, Price: 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, ok := ownuse.Step(p23, tt.on, 9, tt.goods, false, 100)
			want := ownuse.Result{Cover: 1000, Afford: 1000, Secure: 1000, On: tt.on, Jobs: 9}
			if ok || r != want {
				t.Errorf("Step(on %v, %v) = %+v, %v, want %+v, false", tt.on, tt.goods, r, ok, want)
			}
		})
	}
	if _, ok := ownuse.Step(p23, false, 0, one(7), false, -1); ok {
		t.Errorf("Step with a negative money returned ok")
	}
}
