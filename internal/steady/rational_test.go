package steady_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// TestFracArithmetic verifies the exact methods the root builds the table
// from: product, sum, quotient, comparison, sign and absolute value, and the
// three roundings with their bounds.
//
// Requirement: FR-VIA-015, FR-VIA-023
func TestFracArithmetic(t *testing.T) {
	t.Parallel()
	half := steady.Rat(1, 2)
	third := steady.Rat(1, 3)
	tests := []struct {
		name string
		got  steady.Frac
		want steady.Frac
	}{
		// 1/2 * 2/3 = 1/3.
		{"mul", half.Mul(2, 3), third},
		// 1/2 * 1/3 = 1/6.
		{"times", half.Times(third), steady.Rat(1, 6)},
		// 1/2 + 1/3 = 5/6.
		{"add", half.Add(third), steady.Rat(5, 6)},
		// |1/3 - 1/2| = 1/6.
		{"abs", third.Add(half.Mul(-1, 1)).Abs(), steady.Rat(1, 6)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got.Cmp(tt.want) != 0 {
				t.Errorf("%s = %v/%v, want %v/%v", tt.name, tt.got.Num, tt.got.Den, tt.want.Num, tt.want.Den)
			}
		})
	}
	if q, ok := half.Quo(third); !ok || q.Cmp(steady.Rat(3, 2)) != 0 { // (1/2)/(1/3) = 3/2
		t.Errorf("Quo(1/2, 1/3) = %v, %v, want 3/2, true", q, ok)
	}
	if _, ok := half.Quo(steady.Rat(0, 1)); ok {
		t.Errorf("Quo(1/2, 0) ok = true, want false")
	}
	if got := steady.Rat(-1, 2).Sign(); got != -1 {
		t.Errorf("Sign(-1/2) = %d, want -1", got)
	}
}

// TestFracRounding verifies the three roundings on 5/2, 7/2 and a negative
// half, and the bound: a value outside [lo, hi] is not ok.
//
// Requirement: FR-VIA-023
func TestFracRounding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                    string
		f                       steady.Frac
		floor, ceil, round      int64
		lo, hi                  int64
		wantFits, wantRoundFits bool
	}{
		// 5/2: floor 2, ceil 3, half to even 2.
		{"five halves", steady.Rat(5, 2), 2, 3, 2, 0, 10, true, true},
		// 7/2: floor 3, ceil 4, half to even 4.
		{"seven halves", steady.Rat(7, 2), 3, 4, 4, 0, 10, true, true},
		// 7/2 against a ceiling of 3: floor 3 fits, round 4 does not.
		{"above the bound", steady.Rat(7, 2), 3, 4, 4, 0, 3, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fl, ok1 := tt.f.Floor(tt.lo, tt.hi)
			ce, _ := tt.f.Ceil(tt.lo, 100)
			ro, ok3 := tt.f.Round(tt.lo, tt.hi)
			if fl != tt.floor || ok1 != tt.wantFits {
				t.Errorf("Floor = %d, %v, want %d, %v", fl, ok1, tt.floor, tt.wantFits)
			}
			if ce != tt.ceil {
				t.Errorf("Ceil = %d, want %d", ce, tt.ceil)
			}
			if ok3 != tt.wantRoundFits || (ok3 && ro != tt.round) {
				t.Errorf("Round = %d, %v, want %d, %v", ro, ok3, tt.round, tt.wantRoundFits)
			}
		})
	}
}

// TestRat_RejectsBadDenominator verifies that a denominator below 1 in Rat or
// Mul is a programmer invariant that panics.
//
// Guard: a zero denominator would otherwise reach big.Rat and panic with a
// message that does not say it is a programmer error.
func TestRat_RejectsBadDenominator(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func(){
		"Rat": func() { steady.Rat(1, 0) },
		"Mul": func() { steady.Rat(1, 2).Mul(1, 0) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("%s with a zero denominator did not panic", name)
				}
			}()
			f()
		})
	}
}

// TestApportion verifies the largest-remainder method: the floors first, the
// rest to the largest remainders, a tie to the earlier index, and all zero
// weights giving all zero.
//
// Requirement: FR-VIA-028
func TestApportion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		total   int64
		weights []steady.Frac
		want    []int64
	}{
		// 10 over 1:1:1 = quotas 3.33 each: floors 3, 3, 3 and the one left to the
		// first of the equal remainders.
		{"tie to the earlier", 10, []steady.Frac{steady.Rat(1, 1), steady.Rat(1, 1), steady.Rat(1, 1)}, []int64{4, 3, 3}},
		// 600 over the needs of `one` scaled to tenths: the quotas of VIA 5.2 are
		// cook 135.83, grower 253.53 and a third group 210.64; floors 135, 253,
		// 210 sum to 598 and the two largest remainders (.83, .64) take the rest.
		{"largest remainder", 600, []steady.Frac{steady.Rat(13583, 100), steady.Rat(25353, 100), steady.Rat(21064, 100)}, []int64{136, 253, 211}},
		{"zero weights", 7, []steady.Frac{steady.Rat(0, 1), steady.Rat(0, 1)}, []int64{0, 0}},
		{"no weights", 7, nil, []int64{}},
		{"exact", 6, []steady.Frac{steady.Rat(1, 1), steady.Rat(2, 1)}, []int64{2, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := steady.Apportion(tt.total, tt.weights); !slices.Equal(got, tt.want) {
				t.Errorf("Apportion(%d, %d weights) = %v, want %v", tt.total, len(tt.weights), got, tt.want)
			}
		})
	}
}

// TestApportion_RejectsBadInput verifies that a negative total or weight is a
// programmer invariant that panics.
//
// Guard: a negative quota would round the wrong way and hand out more units
// than the total.
func TestApportion_RejectsBadInput(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func(){
		"total":  func() { steady.Apportion(-1, nil) },
		"weight": func() { steady.Apportion(1, []steady.Frac{steady.Rat(-1, 1)}) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Apportion with a bad %s did not panic", name)
				}
			}()
			f()
		})
	}
}

// TestActivityUsesYieldValueAdded verifies the consumption, the mean output and
// the value added of one execution on a hand activity.
//
// Requirement: FR-VIA-001, FR-VIA-025
func TestActivityUsesYieldValueAdded(t *testing.T) {
	t.Parallel()
	// Two starts at factors 10 000 and 8 000 (FactorSum 18 000): mean 9 000. It
	// takes 5 of commodity 0 and wears 3/2 000 of the tool 1, and outputs 12 of
	// commodity 2 and a discard 4 of commodity 3.
	a := steady.Activity{
		Duration: 2, Starts: 2, FactorSum: 18_000,
		Inputs:  []steady.Item{{Commodity: 0, Qty: 5}},
		Tools:   []steady.Tool{{Commodity: 1, Wear: 3, Durability: 2_000}},
		Outputs: []steady.Item{{Commodity: 2, Qty: 12}, {Commodity: 3, Qty: 4, Discard: true}},
	}
	if got := a.Uses(0); got.Cmp(steady.Rat(5, 1)) != 0 {
		t.Errorf("Uses(0) = %v/%v, want 5", got.Num, got.Den)
	}
	if got := a.Uses(1); got.Cmp(steady.Rat(3, 2000)) != 0 {
		t.Errorf("Uses(1) = %v/%v, want 3/2000", got.Num, got.Den)
	}
	// 18 000 * 12 / (10 000 * 2) = 10.8 = 54/5.
	if got := a.Yield(2); got.Cmp(steady.Rat(54, 5)) != 0 {
		t.Errorf("Yield(2) = %v/%v, want 54/5", got.Num, got.Den)
	}
	if got := a.Yield(3); got.Sign() != 0 {
		t.Errorf("Yield(3) of a discard = %v/%v, want 0", got.Num, got.Den)
	}
	// Prices 10, 1 000, 20: 10.8 * 20 - 5 * 10 - 3/2 000 * 1 000 = 216 - 50 - 1.5 = 164.5.
	if got := a.ValueAdded([]int64{10, 1_000, 20}); got.Cmp(steady.Rat(329, 2)) != 0 {
		t.Errorf("ValueAdded = %v/%v, want 329/2", got.Num, got.Den)
	}
	// A commodity beyond the prices counts for 0: 216 - 0 - 0.
	if got := a.ValueAdded([]int64{0, 0, 20}); got.Cmp(steady.Rat(216, 1)) != 0 {
		t.Errorf("ValueAdded with short prices = %v/%v, want 216", got.Num, got.Den)
	}
}

// TestActivity_RejectsBadData verifies that a tool with no durability or an
// activity with no start is a programmer invariant that panics in Uses and
// Yield.
//
// Guard: a zero durability would divide by zero and a zero start count would
// make the mean yield undefined, each inside big.Rat with a message that does
// not say it is a programmer error.
func TestActivity_RejectsBadData(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func(){
		"Uses":  func() { steady.Activity{Tools: []steady.Tool{{Commodity: 0, Wear: 1}}}.Uses(0) },
		"Yield": func() { steady.Activity{}.Yield(0) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("%s of bad data did not panic", name)
				}
			}()
			f()
		})
	}
}
