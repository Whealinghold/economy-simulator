package checked_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// refMulDiv computes a*b/c exactly with math/big under the named rounding
// rule, and reports whether c is non-zero and the result fits in int64. It is
// the oracle of these tests and is written independently of the package: it
// takes the truncated quotient and remainder of big.Int.QuoRem and adjusts.
func refMulDiv(a, b, c int64, rule string) (q, r int64, ok bool) {
	if c == 0 {
		return 0, 0, false
	}
	p := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	bc := big.NewInt(c)
	bq, br := new(big.Int).QuoRem(p, bc, new(big.Int))
	rem := new(big.Int).Set(br)
	if br.Sign() != 0 {
		// The exact quotient lies strictly between bq and bq+away, where
		// away is the direction of the quotient's sign (+1 or -1).
		away := int64(1)
		if p.Sign()*bc.Sign() < 0 {
			away = -1
		}
		up := false
		switch rule {
		case "floor":
			up = away < 0
		case "ceil":
			up = away > 0
		case "half-even":
			twice := new(big.Int).Abs(br)
			twice.Lsh(twice, 1)
			cmp := twice.Cmp(new(big.Int).Abs(bc))
			up = cmp > 0 || (cmp == 0 && bq.Bit(0) == 1)
		}
		if up {
			bq.Add(bq, big.NewInt(away))
		}
	}
	if !bq.IsInt64() {
		return 0, 0, false
	}
	return bq.Int64(), rem.Int64(), true
}

var rules = []string{"trunc", "floor", "ceil", "half-even"}

func call(rule string, a, b, c int64) (q int64, ok bool) {
	switch rule {
	case "floor":
		return checked.MulDivFloor(a, b, c)
	case "ceil":
		return checked.MulDivCeil(a, b, c)
	case "half-even":
		return checked.MulDivHalfEven(a, b, c)
	}
	return checked.MulDiv(a, b, c)
}

// TestMulDiv_RoundingRules verifies each rounding variant on small values
// whose results are worked by hand, for every sign combination of dividend and
// divisor.
//
// Requirement: FR-CUR-003
func TestMulDiv_RoundingRules(t *testing.T) {
	t.Parallel()
	// Each row is a*b/c. 7*1/2 = 3.5, 5*1/2 = 2.5, 7*1/3 = 2.33, 8*1/3 =
	// 2.67; the signed rows are the same magnitudes with a sign.
	tests := []struct {
		name                         string
		a, b, c                      int64
		trunc, floor, ceil, halfEven int64
	}{
		{"3.5", 7, 1, 2, 3, 3, 4, 4},
		{"2.5", 5, 1, 2, 2, 2, 3, 2},
		{"-3.5", -7, 1, 2, -3, -4, -3, -4},
		{"-2.5", -5, 1, 2, -2, -3, -2, -2},
		{"3.5 by negative divisor", 7, 1, -2, -3, -4, -3, -4},
		{"-3.5 by negative divisor", -7, 1, -2, 3, 3, 4, 4},
		{"2.33", 7, 1, 3, 2, 2, 3, 2},
		{"2.67", 8, 1, 3, 2, 2, 3, 3},
		{"-2.67", -8, 1, 3, -2, -3, -2, -3},
		{"exact", 6, 2, 4, 3, 3, 3, 3},
		{"zero product", 0, 99, 7, 0, 0, 0, 0},
		{"negative zero quotient", -1, 1, 3, 0, -1, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			want := map[string]int64{"trunc": tt.trunc, "floor": tt.floor, "ceil": tt.ceil, "half-even": tt.halfEven}
			for _, rule := range rules {
				got, ok := call(rule, tt.a, tt.b, tt.c)
				if !ok || got != want[rule] {
					t.Errorf("%s MulDiv(%d, %d, %d) = %d, %v, want %d, true", rule, tt.a, tt.b, tt.c, got, ok, want[rule])
				}
			}
		})
	}
}

// TestMulDiv_BoundaryAgainstBig verifies every rounding variant, the
// remainder and the ok result against math/big at the edges of int64, for all
// sign combinations: products that need 128 bits, quotients at exactly
// MaxInt64 and MinInt64, and quotients one past each.
//
// Requirement: FR-CUR-003
func TestMulDiv_BoundaryAgainstBig(t *testing.T) {
	t.Parallel()
	edges := []int64{0, 1, 2, 3, -1, -2, -3, math.MaxInt64, math.MaxInt64 - 1, math.MinInt64, math.MinInt64 + 1, 1 << 32, -(1 << 32), 1<<62 + 1}
	for _, a := range edges {
		for _, b := range edges {
			for _, c := range edges {
				for _, rule := range rules {
					wq, _, wok := refMulDiv(a, b, c, rule)
					gq, gok := call(rule, a, b, c)
					if gok != wok || gq != wq {
						t.Errorf("%s MulDiv(%d, %d, %d) = %d, %v, want %d, %v", rule, a, b, c, gq, gok, wq, wok)
					}
				}
				wq, wr, wok := refMulDiv(a, b, c, "trunc")
				gq, gr, gok := checked.MulDivRem(a, b, c)
				if gok != wok || gq != wq || gr != wr {
					t.Errorf("MulDivRem(%d, %d, %d) = %d, %d, %v, want %d, %d, %v", a, b, c, gq, gr, gok, wq, wr, wok)
				}
			}
		}
	}
}

// TestMulDiv_RoundUpFromLargestMagnitude verifies that a quotient whose
// truncated magnitude is 2^64-1 and that must round away from zero reports not
// ok, in every rounding variant and for both signs, instead of wrapping to 0.
//
// The witness: with c = 2^62-2^33, a = 2^63-i and b = 2^63-j where
// i+j = 2^34+1, a*b = 2^64*c - 2^63 + i*j. For i = 5e8, i*j lies between
// 2^63-c/2 and 2^63, so the remainder is c-2^63+i*j, strictly between c/2 and
// c, and the truncated quotient is exactly 2^64-1.
//
// Guard: the carry out of the round-up addition is the only thing that stops
// the magnitude 2^64-1+1 wrapping to 0, which signed then accepts as a valid
// result of 0. Without this test, deleting the carry check passes every other
// test in the package (found by mutation testing).
func TestMulDiv_RoundUpFromLargestMagnitude(t *testing.T) {
	t.Parallel()
	const i = 500_000_000
	const j = 1<<34 + 1 - i
	c := int64(1<<62 - 1<<33)
	a := int64(math.MaxInt64) - (i - 1) // 2^63 - i
	b := int64(math.MaxInt64) - (j - 1) // 2^63 - j

	// The witness is what the comment says it is: truncated quotient 2^64-1,
	// remainder above c/2.
	p := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	bq, br := new(big.Int).QuoRem(p, big.NewInt(c), new(big.Int))
	if want := new(big.Int).SetUint64(math.MaxUint64); bq.Cmp(want) != 0 {
		t.Fatalf("witness quotient = %v, want %v", bq, want)
	}
	if twice := new(big.Int).Lsh(br, 1); twice.Cmp(big.NewInt(c)) <= 0 {
		t.Fatalf("witness remainder = %v, want more than c/2 = %d", br, c/2)
	}

	for _, rule := range rules {
		for _, neg := range []bool{false, true} {
			bb := b
			if neg {
				bb = -b
			}
			// Every variant must report the same result as the oracle: the
			// quotient does not fit in int64, so not ok.
			wq, _, wok := refMulDiv(a, bb, c, rule)
			gq, gok := call(rule, a, bb, c)
			if gok != wok || gq != wq {
				t.Errorf("%s MulDiv(%d, %d, %d) = %d, %v, want %d, %v", rule, a, bb, c, gq, gok, wq, wok)
			}
			if gok {
				t.Errorf("%s MulDiv(%d, %d, %d) is ok, want not ok: the quotient needs 65 bits", rule, a, bb, c)
			}
		}
	}
}

// TestMulDiv_ZeroDivisor verifies that a zero divisor reports not ok instead
// of panicking, for every variant.
//
// Guard: Go panics on integer division by zero; a caller that passes a
// computed divisor would crash the world without this result.
func TestMulDiv_ZeroDivisor(t *testing.T) {
	t.Parallel()
	for _, rule := range rules {
		if q, ok := call(rule, 5, 5, 0); ok || q != 0 {
			t.Errorf("%s MulDiv(5, 5, 0) = %d, %v, want 0, false", rule, q, ok)
		}
	}
	if q, r, ok := checked.MulDivRem(5, 5, 0); ok || q != 0 || r != 0 {
		t.Errorf("MulDivRem(5, 5, 0) = %d, %d, %v, want 0, 0, false", q, r, ok)
	}
}

// TestMulDiv_RandomAgainstBig verifies every variant against math/big for a
// fixed-seed sample of full-width operands, mixing magnitudes so that both
// fitting and overflowing quotients occur.
//
// Requirement: FR-CUR-003
func TestMulDiv_RandomAgainstBig(t *testing.T) {
	t.Parallel()
	const seed = 20261012
	rng := rand.New(rand.NewPCG(seed, seed))
	pick := func() int64 {
		v := int64(rng.Uint64())
		// A shift of 0 to 63 bits gives operands of every magnitude.
		return v >> rng.UintN(64)
	}
	for i := 0; i < 20000; i++ {
		a, b, c := pick(), pick(), pick()
		for _, rule := range rules {
			wq, _, wok := refMulDiv(a, b, c, rule)
			gq, gok := call(rule, a, b, c)
			if gok != wok || gq != wq {
				t.Fatalf("seed %d sample %d: %s MulDiv(%d, %d, %d) = %d, %v, want %d, %v", seed, i, rule, a, b, c, gq, gok, wq, wok)
			}
		}
	}
}

// TestAddSubMul verifies the checked sum, difference and product at the edges
// of int64 against math/big.
//
// Requirement: FR-CUR-003
func TestAddSubMul(t *testing.T) {
	t.Parallel()
	edges := []int64{0, 1, -1, 2, math.MaxInt64, math.MinInt64, math.MaxInt64 - 1, math.MinInt64 + 1, 1 << 32, -(1 << 32), 3037000500}
	type op struct {
		name string
		f    func(a, b int64) (int64, bool)
		ref  func(z, x, y *big.Int) *big.Int
	}
	ops := []op{
		{"Add", checked.Add, (*big.Int).Add},
		{"Sub", checked.Sub, (*big.Int).Sub},
		{"Mul", checked.Mul, (*big.Int).Mul},
	}
	for _, o := range ops {
		t.Run(o.name, func(t *testing.T) {
			t.Parallel()
			for _, a := range edges {
				for _, b := range edges {
					w := o.ref(new(big.Int), big.NewInt(a), big.NewInt(b))
					wantOK := w.IsInt64()
					var want int64
					if wantOK {
						want = w.Int64()
					}
					got, ok := o.f(a, b)
					if ok != wantOK || got != want {
						t.Errorf("%s(%d, %d) = %d, %v, want %d, %v", o.name, a, b, got, ok, want, wantOK)
					}
				}
			}
		})
	}
}

// FuzzMulDiv checks that no variant panics for any operands and that each
// agrees with math/big in value and in ok.
//
// Requirement: FR-CUR-003
func FuzzMulDiv(f *testing.F) {
	f.Add(int64(7), int64(1), int64(2))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64))
	f.Add(int64(math.MinInt64), int64(-1), int64(1))
	f.Add(int64(math.MinInt64), int64(math.MinInt64), int64(math.MinInt64))
	f.Add(int64(5), int64(5), int64(0))
	f.Fuzz(func(t *testing.T, a, b, c int64) {
		for _, rule := range rules {
			wq, _, wok := refMulDiv(a, b, c, rule)
			gq, gok := call(rule, a, b, c)
			if gok != wok || gq != wq {
				t.Fatalf("%s MulDiv(%d, %d, %d) = %d, %v, want %d, %v", rule, a, b, c, gq, gok, wq, wok)
			}
		}
		wq, wr, wok := refMulDiv(a, b, c, "trunc")
		gq, gr, gok := checked.MulDivRem(a, b, c)
		if gok != wok || gq != wq || gr != wr {
			t.Fatalf("MulDivRem(%d, %d, %d) = %d, %d, %v, want %d, %d, %v", a, b, c, gq, gr, gok, wq, wr, wok)
		}
		if p, ok := checked.Mul(a, b); ok != new(big.Int).Mul(big.NewInt(a), big.NewInt(b)).IsInt64() || (ok && p != a*b) {
			t.Fatalf("Mul(%d, %d) = %d, %v", a, b, p, ok)
		}
	})
}
