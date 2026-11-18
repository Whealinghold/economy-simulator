package fiscal_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

func baseOf(t *testing.T, vs ...int64) fiscal.Base {
	t.Helper()
	var b fiscal.Base
	for _, v := range vs {
		if !b.Add(v) {
			t.Fatalf("Base.Add(%d) = false", v)
		}
	}
	return b
}

// TestDutyDue_Carry verifies the sales duty of spec-fiscal.md 18.3: 300 bp at
// scale 10 000. Seller A's base of 530 + 820 = 1 350 gives
// 1 350 * 300 * 10^4 = 4.05 * 10^9, due 40 and carry 5 * 10^7; seller B's base
// of 208 gives 6.24 * 10^8, due 6 and carry 2.4 * 10^7; a second tick of 100 for
// A gives 100 * 300 * 10^4 + 5 * 10^7 = 3.5 * 10^8, due 3 and carry 5 * 10^7.
// Over the two ticks A paid 43 of the exact 43.5.
//
// Requirement: FR-FIS-032
func TestDutyDue_Carry(t *testing.T) {
	t.Parallel()
	due, carry, ok := fiscal.DutyDue(baseOf(t, 530, 820), 300, 10_000, 0, maxMoney)
	if !ok || due != 40 || carry != 50_000_000 {
		t.Errorf("DutyDue(1350) = %d, %d, %v; want 40, 50000000, true", due, carry, ok)
	}
	if due, carry, ok = fiscal.DutyDue(baseOf(t, 208), 300, 10_000, 0, maxMoney); !ok || due != 6 || carry != 24_000_000 {
		t.Errorf("DutyDue(208) = %d, %d, %v; want 6, 24000000, true", due, carry, ok)
	}
	due2, carry2, ok := fiscal.DutyDue(baseOf(t, 100), 300, 10_000, 50_000_000, maxMoney)
	if !ok || due2 != 3 || carry2 != 50_000_000 {
		t.Errorf("DutyDue(100, carry 5e7) = %d, %d, %v; want 3, 50000000, true", due2, carry2, ok)
	}
}

// TestDutyDue_ExactOverRuns verifies that, with an unchanged rate and scale, the
// total paid over a run of ticks is floor((sum of bases * rate * scale + c0) /
// 10^8) exactly, however the value splits into ticks, over 2 000 random runs
// (the model of spec-fiscal.md 20). Each run is seeded with its own seed, in the
// message.
//
// Requirement: FR-FIS-032
func TestDutyDue_ExactOverRuns(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 2_000; seed++ {
		rng := rand.New(rand.NewPCG(seed, 7))
		rate, scale := rng.Int64N(10_000)+1, rng.Int64N(25_001)
		var carry, paid, sum int64
		for range rng.IntN(20) + 1 {
			v := rng.Int64N(10_000_000)
			sum += v
			due, next, ok := fiscal.DutyDue(baseOf(t, v), rate, scale, carry, maxMoney)
			if !ok {
				t.Fatalf("seed %d: DutyDue(%d, %d, %d, %d) not ok", seed, v, rate, scale, carry)
			}
			paid, carry = paid+due, next
		}
		want := new(big.Int).Mul(big.NewInt(sum), big.NewInt(rate*scale))
		want.Div(want, big.NewInt(100_000_000))
		if big.NewInt(paid).Cmp(want) != 0 {
			t.Errorf("seed %d: paid %d over the run, want %v", seed, paid, want)
		}
	}
}

// TestDutyDue_Wide verifies the 192-bit intermediate against big integers on a
// base beyond int64: the sum of many int64 values, a rate and scale at their
// largest, and a quotient above maxMoney, which is cut to maxMoney.
//
// Requirement: FR-FIS-032, NFR-FIS-003
func TestDutyDue_Wide(t *testing.T) {
	t.Parallel()
	var b fiscal.Base
	for range 1_000 {
		b.Add(math.MaxInt64)
	}
	due, carry, ok := fiscal.DutyDue(b, 1, 1, 12_345, math.MaxInt64)
	num := new(big.Int).Mul(big.NewInt(math.MaxInt64), big.NewInt(1_000))
	num.Add(num, big.NewInt(12_345))
	q, r := new(big.Int).DivMod(num, big.NewInt(100_000_000), new(big.Int))
	if !ok || big.NewInt(due).Cmp(q) != 0 || big.NewInt(carry).Cmp(r) != 0 {
		t.Errorf("DutyDue(wide) = %d, %d, %v; want %v, %v, true", due, carry, ok, q, r)
	}
	if due, _, _ = fiscal.DutyDue(b, 10_000, 100_000, 0, 1_000); due != 1_000 {
		t.Errorf("DutyDue(wide, maxMoney 1000) = %d, want the cut to 1000", due)
	}
}

// TestDutyDue_Domain verifies that a negative argument, a carry of 10^8 or more
// and a rate times scale that leaves int64 are reported, and that a negative
// value is left out of a base.
//
// Requirement: NFR-FIS-003
func TestDutyDue_Domain(t *testing.T) {
	t.Parallel()
	var b fiscal.Base
	if b.Add(-1) || !b.IsZero() {
		t.Errorf("Base.Add(-1) must be refused and leave the base empty")
	}
	b.Add(5)
	if b.IsZero() {
		t.Errorf("Base.IsZero after Add(5) = true")
	}
	tests := []struct{ rate, scale, carry, max int64 }{
		{-1, 1, 0, 1}, {1, -1, 0, 1}, {1, 1, -1, 1}, {1, 1, 100_000_000, 1}, {1, 1, 0, -1},
		{math.MaxInt64, 2, 0, 1},
	}
	for _, tt := range tests {
		if due, _, ok := fiscal.DutyDue(b, tt.rate, tt.scale, tt.carry, tt.max); ok {
			t.Errorf("DutyDue(rate %d, scale %d, carry %d, max %d) = %d, true; want ok false", tt.rate, tt.scale, tt.carry, tt.max, due)
		}
	}
}
