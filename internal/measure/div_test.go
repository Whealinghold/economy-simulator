package measure_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// halfEven returns n / d rounded half to even, symmetric for negative values,
// computed with math/big: the oracle of RoundDiv.
func halfEven(n, d *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(new(big.Int).Abs(n), new(big.Int).Abs(d), new(big.Int))
	switch c := new(big.Int).Lsh(r, 1).Cmp(new(big.Int).Abs(d)); {
	case c > 0, c == 0 && q.Bit(0) == 1:
		q.Add(q, big.NewInt(1))
	}
	if n.Sign()*d.Sign() < 0 {
		q.Neg(q)
	}
	return q
}

// TestRoundDiv_HalfEvenBothSigns verifies the rounding rule of
// spec-indicators.md 5.2.2 on its own examples and on the edges, and that a zero
// divisor reports not ok and never panics.
//
// Requirement: FR-IND-012
func TestRoundDiv_HalfEvenBothSigns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		n, d   int64
		want   int64
		wantOK bool
	}{
		{"minus five over two is minus two", -5, 2, -2, true},
		{"minus three over two is minus two", -3, 2, -2, true},
		{"twenty five over ten is two", 25, 10, 2, true},
		{"thirty five over ten is four", 35, 10, 4, true},
		{"below half rounds down", 14, 10, 1, true},
		{"above half rounds up", 16, 10, 2, true},
		{"negative divisor flips the sign", 5, -2, -2, true},
		{"both negative", -7, -2, 4, true},
		{"exact", 40, 8, 5, true},
		{"zero numerator", 0, 7, 0, true},
		{"zero divisor", 7, 0, 0, false},
		{"zero over zero", 0, 0, 0, false},
		{"min over minus one is out of int64 but exact in 128 bits", math.MinInt64, -1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q, ok := measure.RoundDiv(measure.FromInt64(tt.n), measure.FromInt64(tt.d))
			if ok != tt.wantOK {
				t.Fatalf("RoundDiv(%d, %d) ok = %v, want %v", tt.n, tt.d, ok, tt.wantOK)
			}
			if tt.n == math.MinInt64 {
				if got := bigOf(q); got.Cmp(new(big.Int).Neg(big.NewInt(math.MinInt64))) != 0 {
					t.Errorf("RoundDiv(MinInt64, -1) = %v, want 2^63", got)
				}
				return
			}
			if got, _ := measure.ToInt64(q); got != tt.want {
				t.Errorf("RoundDiv(%d, %d) = %d, want %d", tt.n, tt.d, got, tt.want)
			}
		})
	}
}

// TestRoundDiv_AgainstBig verifies RoundDiv on random 128-bit numerators and
// divisors of every size against math/big, and RoundDivU against the signed form.
//
// Requirement: FR-IND-012
func TestRoundDiv_AgainstBig(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(5, 6))
	for range 4000 {
		n, d := randInt128(r), randInt128(r)
		if r.IntN(3) == 0 {
			d = measure.FromInt64(randInt64(r) % 1000) // small divisors make exact halves common
		}
		got, ok := measure.RoundDiv(n, d)
		if ok != (d.Sign() != 0) {
			t.Fatalf("RoundDiv(%v, %v) ok = %v", bigOf(n), bigOf(d), ok)
		}
		if ok {
			if want := halfEven(bigOf(n), bigOf(d)); bigOf(got).Cmp(want) != 0 {
				t.Errorf("RoundDiv(%v, %v) = %v, want %v", bigOf(n), bigOf(d), bigOf(got), want)
			}
		}
	}
	a, b := measure.Uint128{Hi: 3, Lo: 9}, measure.Uint128{Lo: 7}
	if u, ok := measure.RoundDivU(a, b); !ok || bigOfU(u).Cmp(halfEven(bigOfU(a), bigOfU(b))) != 0 {
		t.Errorf("RoundDivU = %v, %v", bigOfU(u), ok)
	}
}

// TestRoundDiv_AgreesWithChecked verifies that RoundDiv and PV give the answers
// of internal/checked (MulDivHalfEven and Acc.DivHalfEven) on the domain the two
// share, so the rounding rule has one definition and not two.
//
// Guard: spec-indicators.md 5.2.1 and plan 2-18 D562 say the 128-bit helper is
// one. If measure drifted from checked, a price published by clearing and a
// price an indicator derives from the same trades would round differently.
func TestRoundDiv_AgreesWithChecked(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(7, 8))
	for range 4000 {
		a, b := int64(r.Uint64()>>uint(r.IntN(63))), int64(r.Uint64()>>uint(r.IntN(63)))
		c := int64(r.Uint64() >> uint(1+r.IntN(62)))
		if c == 0 {
			continue
		}
		want, wantOK := checked.MulDivHalfEven(a, b, c)
		q, ok := measure.RoundDiv(measure.MulInt64(a, b), measure.FromInt64(c))
		got, fits := measure.ToInt64(q)
		if wantOK != (ok && fits) || (wantOK && got != want) {
			t.Fatalf("RoundDiv(%d*%d, %d) = %d (ok %v), checked.MulDivHalfEven = %d (ok %v)", a, b, c, got, ok && fits, want, wantOK)
		}
		var pv measure.PV
		var acc checked.Acc
		if pv.Add(a, b) != acc.AddMul(a, b) {
			t.Fatalf("PV.Add(%d, %d) disagrees with checked.Acc.AddMul", a, b)
		}
		pg, pok := pv.Div(c)
		cg, cok := acc.DivHalfEven(c)
		if pg != cg || pok != cok {
			t.Errorf("PV.Div(%d) = %d, %v; checked.Acc.DivHalfEven = %d, %v", c, pg, pok, cg, cok)
		}
	}
}

// TestPV_RejectsNegativeAndBadDivisor verifies the edges of the price-times-
// volume sum: a negative factor leaves it unchanged, and a divisor below 1 is
// not ok.
//
// Requirement: FR-IND-028
func TestPV_RejectsNegativeAndBadDivisor(t *testing.T) {
	t.Parallel()
	var pv measure.PV
	// 10*30 + 11*20 = 520 over 50 units is 10.4, which rounds to 10 (spec 18.1)
	if !pv.Add(10, 30) || !pv.Add(11, 20) {
		t.Fatal("PV.Add of non-negative factors = false, want true")
	}
	if pv.Add(-1, 5) || pv.Add(5, -1) {
		t.Error("PV.Add of a negative factor = true, want false")
	}
	if got, ok := pv.Div(50); !ok || got != 10 {
		t.Errorf("PV.Div(50) = %d, %v, want 10, true", got, ok)
	}
	for _, v := range []int64{0, -1} {
		if _, ok := pv.Div(v); ok {
			t.Errorf("PV.Div(%d) ok = true, want false", v)
		}
	}
}

// FuzzRoundDiv verifies that RoundDiv never panics and agrees with math/big for
// any numerator and divisor.
//
// Requirement: NFR-IND-003
func FuzzRoundDiv(f *testing.F) {
	f.Add(int64(-5), int64(2), int64(1), int64(0))
	f.Add(int64(math.MaxInt64), int64(math.MinInt64), int64(7), int64(3))
	f.Add(int64(0), int64(0), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, a, b, c, d int64) {
		n := measure.MulInt64(a, b)
		den := measure.MulInt64(c, d)
		got, ok := measure.RoundDiv(n, den)
		if ok != (den.Sign() != 0) {
			t.Fatalf("RoundDiv(%d*%d, %d*%d) ok = %v", a, b, c, d, ok)
		}
		if ok && bigOf(got).Cmp(halfEven(bigOf(n), bigOf(den))) != 0 {
			t.Errorf("RoundDiv(%d*%d, %d*%d) = %v, want %v", a, b, c, d, bigOf(got), halfEven(bigOf(n), bigOf(den)))
		}
	})
}
