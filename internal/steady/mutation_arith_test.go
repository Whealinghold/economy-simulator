package steady

import (
	"math/big"
	"slices"
	"testing"
)

// panicText runs f and returns what it panicked with, or "" when it did not.
func panicText(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg, _ = r.(string)
			if msg == "" {
				msg = "non-string panic"
			}
		}
	}()
	f()
	return ""
}

// TestInvariantPanicsMutation pins each programmer-invariant panic: the guard,
// its boundary and its message.
//
// Guard: gomu survivors in rational.go, activity.go and start.go (guard
// conditions replaced by false, boundaries moved, panics removed or blanked).
func TestInvariantPanicsMutation(t *testing.T) {
	t.Parallel()
	okTool := []Tool{{Commodity: 0, Wear: 1, Durability: 1}}
	badTool := []Tool{{Commodity: 0, Wear: 1, Durability: 0}}
	tests := []struct {
		name string
		call func()
		want string
	}{
		{"Rat zero den", func() { Rat(1, 0) }, "steady: invariant violated: Rat with a denominator below 1"},
		{"Rat negative den", func() { Rat(1, -1) }, "steady: invariant violated: Rat with a denominator below 1"},
		{"Rat den one", func() { Rat(1, 1) }, ""},
		{"Mul zero den", func() { Rat(1, 1).Mul(1, 0) }, "steady: invariant violated: Mul with a denominator below 1"},
		{"Mul den one", func() { Rat(1, 1).Mul(1, 1) }, ""},
		{"Apportion negative total", func() { Apportion(-1, []Frac{Rat(1, 1)}) }, "steady: invariant violated: Apportion of a negative total"},
		{"Apportion zero total", func() { Apportion(0, []Frac{Rat(1, 1)}) }, ""},
		{"Apportion negative weight", func() { Apportion(1, []Frac{Rat(-1, 1)}) }, "steady: invariant violated: Apportion with a negative weight"},
		{"Apportion zero weight", func() { Apportion(1, []Frac{Rat(0, 1)}) }, ""},
		{"Uses durability zero", func() { Activity{Tools: badTool}.Uses(0) }, "steady: invariant violated: tool durability below 1"},
		{"Uses durability one", func() { Activity{Tools: okTool}.Uses(0) }, ""},
		{"Yield no start", func() { Activity{}.Yield(0) }, "steady: invariant violated: activity with no start"},
		{"Yield one start", func() { Activity{Starts: 1}.Yield(0) }, ""},
		{"Net no start", func() { Activity{}.Net() }, "steady: invariant violated: activity with no start"},
		{"Net durability zero", func() { Activity{Starts: 1, Tools: badTool}.Net() }, "steady: invariant violated: tool durability below 1"},
		{"Net durability one", func() { Activity{Starts: 1, Tools: okTool}.Net() }, ""},
		{"Starts duration zero", func() { Starts(0, nil) }, "steady: invariant violated: duration below 1"},
		{"Starts duration one", func() { Starts(1, nil) }, ""},
		{"Starts negative length", func() { Starts(1, []Run{{Length: -1, FactorBP: 1}}) }, "steady: invariant violated: negative run"},
		{"Starts negative factor", func() { Starts(1, []Run{{Length: 1, FactorBP: -1}}) }, "steady: invariant violated: negative run"},
		{"Starts zero run", func() { Starts(1, []Run{{Length: 0, FactorBP: 0}}) }, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := panicText(tc.call); got != tc.want {
				t.Errorf("panic = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFracArithmeticMutation pins the sign and ordering cases of Frac.Quo,
// Apportion, Uses, Yield and ValueAdded that the happy-path tests do not reach.
//
// Guard: gomu survivors in rational.go (Quo by a negative, zero result shape,
// largest-remainder start index, commodity comparisons, accumulation).
func TestFracArithmeticMutation(t *testing.T) {
	t.Parallel()
	t.Run("Quo by a negative", func(t *testing.T) {
		t.Parallel()
		// 1/2 divided by -1/4 is -2.
		got, ok := Rat(1, 2).Quo(Rat(-1, 4))
		if !ok || got.Cmp(Rat(-2, 1)) != 0 {
			t.Errorf("Quo = %v, %v, want -2, true", got, ok)
		}
	})
	t.Run("Quo by zero", func(t *testing.T) {
		t.Parallel()
		got, ok := Rat(1, 2).Quo(Rat(0, 1))
		if ok || got.Num.Sign() != 0 || got.Den.Int64() != 1 {
			t.Errorf("Quo = %v/%v, %v, want 0/1, false", got.Num, got.Den, ok)
		}
	})
	t.Run("Apportion", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			total   int64
			weights []Frac
			want    []int64
		}{
			// 10 over three equal weights: quotas 10/3 each, floors 3, one left, tie to the earlier.
			{"tie to earlier", 10, []Frac{Rat(1, 1), Rat(1, 1), Rat(1, 1)}, []int64{4, 3, 3}},
			// 2 over three equal weights: quotas 2/3, floors 0, two left to indexes 0 and 1.
			{"two left", 2, []Frac{Rat(1, 1), Rat(1, 1), Rat(1, 1)}, []int64{1, 1, 0}},
			// 5 over 1:2: quotas 5/3 and 10/3, remainders 2/3 and 1/3, the one left to index 0.
			{"first largest", 5, []Frac{Rat(1, 1), Rat(2, 1)}, []int64{2, 3}},
			// 5 over 2:1: quotas 10/3 and 5/3, remainders 1/3 and 2/3, the one left to index 1.
			{"second largest", 5, []Frac{Rat(2, 1), Rat(1, 1)}, []int64{3, 2}},
			// 5 over 0:1:2:... the zero weight never wins a unit it did not earn.
			{"zero weight first", 2, []Frac{Rat(0, 1), Rat(1, 1), Rat(1, 1), Rat(1, 1)}, []int64{0, 1, 1, 0}},
			{"all zero", 5, []Frac{Rat(0, 1), Rat(0, 1)}, []int64{0, 0}},
			{"empty", 5, nil, []int64{}},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				if got := Apportion(tc.total, tc.weights); !slices.Equal(got, tc.want) {
					t.Errorf("Apportion = %v, want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("Uses", func(t *testing.T) {
		t.Parallel()
		a := Activity{
			Inputs: []Item{{Commodity: 2, Qty: 5}, {Commodity: 1, Qty: 3}, {Commodity: 1, Qty: 4}},
			Tools:  []Tool{{Commodity: 0, Wear: 1, Durability: 2}, {Commodity: 1, Wear: 1, Durability: 4}, {Commodity: 2, Wear: 3, Durability: 4}},
		}
		// Commodity 1: inputs 3 + 4 plus tool 1/4 = 29/4. The lower and higher
		// commodities must not leak in, and the tool before it must be skipped.
		if got := a.Uses(1); got.Cmp(Rat(29, 4)) != 0 {
			t.Errorf("Uses(1) = %v/%v, want 29/4", got.Num, got.Den)
		}
		// Commodity 2: input 5 plus tool 3/4.
		if got := a.Uses(2); got.Cmp(Rat(23, 4)) != 0 {
			t.Errorf("Uses(2) = %v/%v, want 23/4", got.Num, got.Den)
		}
		// Commodity 0: no input, tool 1/2.
		if got := a.Uses(0); got.Cmp(Rat(1, 2)) != 0 {
			t.Errorf("Uses(0) = %v/%v, want 1/2", got.Num, got.Den)
		}
	})
	t.Run("Yield", func(t *testing.T) {
		t.Parallel()
		a := Activity{
			Starts: 2, FactorSum: 20000,
			Outputs: []Item{{Commodity: 2, Qty: 7}, {Commodity: 1, Qty: 3}, {Commodity: 1, Qty: 4}, {Commodity: 1, Qty: 100, Discard: true}},
		}
		// Mean factor 20000/2 = 10 000 bp = 1, so commodity 1 yields 3 + 4 = 7.
		if got := a.Yield(1); got.Cmp(Rat(7, 1)) != 0 {
			t.Errorf("Yield(1) = %v/%v, want 7", got.Num, got.Den)
		}
		if got := a.Yield(0); got.Sign() != 0 {
			t.Errorf("Yield(0) = %v/%v, want 0", got.Num, got.Den)
		}
	})
	t.Run("ValueAdded", func(t *testing.T) {
		t.Parallel()
		a := Activity{Starts: 1, FactorSum: 10000, Outputs: []Item{{Commodity: 1, Qty: 2}, {Commodity: 2, Qty: 5}}, Inputs: []Item{{Commodity: 0, Qty: 1}}}
		// Net: -1 of commodity 0, +2 of commodity 1, +5 of commodity 2. Prices
		// cover commodities 0 and 1 only, so commodity 2 is priced 0:
		// -1*3 + 2*4 = 5.
		if got := a.ValueAdded([]int64{3, 4}); got.Cmp(Rat(5, 1)) != 0 {
			t.Errorf("ValueAdded = %v/%v, want 5", got.Num, got.Den)
		}
	})
}

// TestNetMutation pins that discarded outputs are skipped without stopping the
// scan and that Net reports in commodity order without zero entries.
//
// Guard: gomu survivor activity.go:62 (continue -> break).
func TestNetMutation(t *testing.T) {
	t.Parallel()
	a := Activity{
		Starts: 1, FactorSum: 10000,
		Outputs: []Item{{Commodity: 0, Qty: 9, Discard: true}, {Commodity: 3, Qty: 2}},
		Inputs:  []Item{{Commodity: 1, Qty: 4}},
	}
	got := a.Net()
	if len(got) != 2 || got[0].Commodity != 1 || got[1].Commodity != 3 {
		t.Fatalf("Net commodities = %v, want [1 3]", got)
	}
	if got[0].Val.Cmp(big.NewRat(-4, 1)) != 0 || got[1].Val.Num().Int64() != 2 {
		t.Errorf("Net values = %v, %v, want -4, 2", got[0].Val, got[1].Val)
	}
}

// Equivalent or undetectable mutants of the steady package that no test can
// kill (gomu, one-line why each):
//
//   - round.go 13, 42, 59: floorDiv's d is never 0 here (checked above), the
//     remainder is never negative and q.Bit(0)==1 only matters on cmp == 0, so
//     the widened comparisons select the same branch.
//   - start.go 44, 47, 52: a run reached at t == end adds n = 0 starts either
//     way; a factor is never negative (it panics above); i < n and i != n agree
//     for an i that counts up from 0 to n >= 0.
//   - simplex.go 46-47: the certificate cannot fail on a correct pivot, so the
//     error branch after certify is defence in depth with no input that reaches
//     it. 80, 94, 105 (== to <=), 132, 136, 137, 142: loop bounds and
//     comparisons on quantities that are never negative at that point (the
//     right-hand side, the reduced costs of an optimal tableau); the -1 sentinels
//     of choose and ratio are only tested with < 0.
//   - tableau.go 173, 175, 184, 199: an extra zero row or column, a doubled
//     objective scale (divided out again in result) and i != m agree with the
//     original; the T survivors (222, 223, 229, i++, j++) loop or corrupt the
//     tableau without an assertion that finishes inside the test timeout.
//   - results.go 78, 85, 88, 90, 100, 117, 125: duals, flows and the surplus are
//     non-negative, a capacity row is never row 0 or 1, x <= RHS so x >= RHS
//     means x == RHS, and the labour share is at least 1.
//   - rational.go 42 (0/1 against 0/2 is the same Rat), 85 (a sum of
//     non-negative weights is never negative), 98-100 and 106 (the largest
//     remainder loop: the sentinel -1 against 0 or -2 stays below every
//     remainder that can still win, and i != len agrees with i < len).
//   - program.go 162, 183: c != Commodities agrees with c < Commodities; a cost
//     of 0/2 is 0.
//   - components.go 100-168: the same loop-bound and low <= index reasons;
//     242, 238, 251 scale the cost of a column or a zero right-hand side, which
//     leaves the optimal s unchanged (only the pivot count could differ).
//   - cycles.go 62, 77, 82, 101, 112: len >= 0 always; an outside need never has
//     a maker in comp; needs order does not reach the order of succ; w.over is
//     sticky; c == start is handled by the first case before c >= start.
//   - gap.go 55, 59, 139, 141: len == 0 against len <= 0; a duplicate tick or cut
//     adds a zero-length interval; i != len agrees with i < len.
//   - producible.go 41, 120: n = 2 for a pooled run writes slot 0 only; the
//     commodity loop bound as above.
//   - certify.go 27: weak duality gives primal <= dual for a feasible pair, so
//     != 0 and < 0 agree on the primal-against-dual comparison.
