package measure_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/measure"
)

// bigOf returns the exact value of a 128-bit integer as a big.Int, the oracle
// of the tests below (a test file may use math/big; the package may not).
func bigOf(v measure.Int128) *big.Int {
	m := bigOfU(v.Mag())
	if v.Sign() < 0 {
		m.Neg(m)
	}
	return m
}

// bigOfU returns the exact value of an unsigned 128-bit integer.
func bigOfU(u measure.Uint128) *big.Int {
	m := new(big.Int).SetUint64(u.Hi)
	m.Lsh(m, 64)
	return m.Or(m, new(big.Int).SetUint64(u.Lo))
}

// randInt64 draws an int64 that is often an edge value, so the carries and the
// extremes are exercised and not only the middle of the range.
func randInt64(r *rand.Rand) int64 {
	switch r.IntN(6) {
	case 0:
		return []int64{0, 1, -1, math.MaxInt64, math.MinInt64, math.MaxInt64 - 1, math.MinInt64 + 1}[r.IntN(7)]
	case 1:
		return int64(r.IntN(2000)) - 1000
	}
	return int64(r.Uint64())
}

// randInt128 draws a signed 128-bit value built from products and sums, so its
// magnitude reaches the upper words.
func randInt128(r *rand.Rand) measure.Int128 {
	v := measure.MulInt64(randInt64(r), randInt64(r))
	if r.IntN(2) == 0 {
		v, _ = v.Add(measure.FromInt64(randInt64(r)))
	}
	return v
}

// TestInt128_ArithmeticLaws verifies Add, Sub, Neg, Cmp, Sign and MulInt64
// against math/big for random signed values, that addition is commutative and
// associative where it does not overflow, and that a sum past 2^127 is
// reported and not wrapped.
//
// Requirement: FR-IND-010, FR-IND-009
func TestInt128_ArithmeticLaws(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(1, 2))
	limit := new(big.Int).Lsh(big.NewInt(1), 127)
	for range 3000 {
		a, b, c := randInt128(r), randInt128(r), randInt128(r)
		sum, ok := a.Add(b)
		want := new(big.Int).Add(bigOf(a), bigOf(b))
		if ok != (want.CmpAbs(limit) < 0) {
			t.Fatalf("Add(%v, %v) ok = %v, want %v", bigOf(a), bigOf(b), ok, !ok)
		}
		if ok && bigOf(sum).Cmp(want) != 0 {
			t.Errorf("Add(%v, %v) = %v, want %v", bigOf(a), bigOf(b), bigOf(sum), want)
		}
		if diff, ok := a.Sub(b); ok && bigOf(diff).Cmp(new(big.Int).Sub(bigOf(a), bigOf(b))) != 0 {
			t.Errorf("Sub(%v, %v) = %v", bigOf(a), bigOf(b), bigOf(diff))
		}
		if got, want := a.Cmp(b), bigOf(a).Cmp(bigOf(b)); got != want {
			t.Errorf("Cmp(%v, %v) = %d, want %d", bigOf(a), bigOf(b), got, want)
		}
		if got, want := a.Sign(), bigOf(a).Sign(); got != want {
			t.Errorf("Sign(%v) = %d, want %d", bigOf(a), got, want)
		}
		if bigOf(a.Neg()).Cmp(new(big.Int).Neg(bigOf(a))) != 0 {
			t.Errorf("Neg(%v) = %v", bigOf(a), bigOf(a.Neg()))
		}
		x, y := randInt64(r), randInt64(r)
		if got, want := bigOf(measure.MulInt64(x, y)), new(big.Int).Mul(big.NewInt(x), big.NewInt(y)); got.Cmp(want) != 0 {
			t.Errorf("MulInt64(%d, %d) = %v, want %v", x, y, got, want)
		}
		// associativity and commutativity where nothing overflows
		ab, ok1 := a.Add(b)
		bc, ok2 := b.Add(c)
		left, ok3 := ab.Add(c)
		right, ok4 := a.Add(bc)
		if ok1 && ok2 && ok3 && ok4 && left.Cmp(right) != 0 {
			t.Errorf("(a+b)+c = %v, a+(b+c) = %v", bigOf(left), bigOf(right))
		}
		if ba, ok := b.Add(a); ok && ba.Cmp(sum) != 0 {
			t.Errorf("b+a = %v, a+b = %v", bigOf(ba), bigOf(sum))
		}
	}
}

// TestInt128_NegativeZeroIsZero verifies that a result of 0 has no sign, so a
// sum that cancels compares equal to 0 and to FromInt64(0).
//
// Guard: a negative zero would make Cmp and Sign disagree about equal values
// and make a flow that nets to nothing look like a loss.
func TestInt128_NegativeZeroIsZero(t *testing.T) {
	t.Parallel()
	a := measure.MulInt64(-7, 11)
	z, ok := a.Add(a.Neg())
	if !ok || z.Sign() != 0 || z.Cmp(measure.FromInt64(0)) != 0 || z.Neg().Cmp(z) != 0 {
		t.Errorf("-77 + 77 = %v (sign %d, ok %v), want a signless 0", bigOf(z), z.Sign(), ok)
	}
	if got := measure.MulInt64(-5, 0); got.Sign() != 0 || got.Cmp(measure.FromInt64(0)) != 0 {
		t.Errorf("MulInt64(-5, 0) = %v, want 0", bigOf(got))
	}
}

// TestFromUint128_Bound verifies the largest value an Int128 holds, 2^127 - 1.
//
// Requirement: FR-IND-009
func TestFromUint128_Bound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   measure.Uint128
		want bool
	}{
		{"zero", measure.Uint128{}, true},
		{"largest", measure.Uint128{Hi: 1<<63 - 1, Lo: math.MaxUint64}, true},
		{"one past", measure.Uint128{Hi: 1 << 63}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, ok := measure.FromUint128(tt.in); ok != tt.want {
				t.Errorf("FromUint128(%v) ok = %v, want %v", bigOfU(tt.in), ok, tt.want)
			}
		})
	}
}

// TestUint128_AgainstBig verifies Add, Sub, Mul, MulU64, Mul64, Cmp and DivRem
// against math/big, with divisors below and above 2^64 (the two code paths of
// DivRem), including the carry out of the shift when the divisor is above 2^127.
//
// Requirement: FR-IND-010
func TestUint128_AgainstBig(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(3, 4))
	two128 := new(big.Int).Lsh(big.NewInt(1), 128)
	draw := func() measure.Uint128 {
		switch r.IntN(4) {
		case 0:
			return measure.Uint128{Lo: r.Uint64()}
		case 1:
			return measure.Uint128{Hi: 1<<63 | r.Uint64(), Lo: r.Uint64()}
		case 2:
			return measure.Uint128{Hi: uint64(r.IntN(3)), Lo: r.Uint64()}
		}
		return measure.Uint128{Hi: r.Uint64(), Lo: r.Uint64()}
	}
	for range 4000 {
		a, b := draw(), draw()
		x, y := bigOfU(a), bigOfU(b)
		if s, ok := a.Add(b); ok != (new(big.Int).Add(x, y).Cmp(two128) < 0) || (ok && bigOfU(s).Cmp(new(big.Int).Add(x, y)) != 0) {
			t.Errorf("Add(%v, %v) = %v, %v", x, y, bigOfU(s), ok)
		}
		if d, ok := a.Sub(b); ok != (x.Cmp(y) >= 0) || (ok && bigOfU(d).Cmp(new(big.Int).Sub(x, y)) != 0) {
			t.Errorf("Sub(%v, %v) = %v, %v", x, y, bigOfU(d), ok)
		}
		if p, ok := a.Mul(b); ok != (new(big.Int).Mul(x, y).Cmp(two128) < 0) || (ok && bigOfU(p).Cmp(new(big.Int).Mul(x, y)) != 0) {
			t.Errorf("Mul(%v, %v) = %v, %v", x, y, bigOfU(p), ok)
		}
		if p, ok := a.MulU64(b.Lo); ok != (new(big.Int).Mul(x, new(big.Int).SetUint64(b.Lo)).Cmp(two128) < 0) || (ok && bigOfU(p).Cmp(new(big.Int).Mul(x, new(big.Int).SetUint64(b.Lo))) != 0) {
			t.Errorf("MulU64(%v, %d) = %v, %v", x, b.Lo, bigOfU(p), ok)
		}
		if got, want := bigOfU(measure.Mul64(a.Lo, b.Lo)), new(big.Int).Mul(new(big.Int).SetUint64(a.Lo), new(big.Int).SetUint64(b.Lo)); got.Cmp(want) != 0 {
			t.Errorf("Mul64(%d, %d) = %v, want %v", a.Lo, b.Lo, got, want)
		}
		if got, want := a.Cmp(b), x.Cmp(y); got != want {
			t.Errorf("Cmp(%v, %v) = %d, want %d", x, y, got, want)
		}
		q, rem, ok := a.DivRem(b)
		if ok != !b.IsZero() {
			t.Fatalf("DivRem(%v, %v) ok = %v", x, y, ok)
		}
		if ok {
			wq, wr := new(big.Int).DivMod(x, y, new(big.Int))
			if bigOfU(q).Cmp(wq) != 0 || bigOfU(rem).Cmp(wr) != 0 {
				t.Errorf("DivRem(%v, %v) = %v r %v, want %v r %v", x, y, bigOfU(q), bigOfU(rem), wq, wr)
			}
		}
	}
}

// TestUint128_DivRemExactAndEdges verifies DivRem where the remainder meets the
// divisor exactly during the shift and subtract (a dividend that is a multiple of
// a divisor with a high word), at the extremes of 128 bits, and that a zero
// divisor is not ok.
//
// Requirement: FR-IND-010
func TestUint128_DivRemExactAndEdges(t *testing.T) {
	t.Parallel()
	max := measure.Uint128{Hi: math.MaxUint64, Lo: math.MaxUint64}
	b := measure.Uint128{Hi: 5, Lo: 7}
	tests := []struct {
		name string
		a, b measure.Uint128
	}{
		{"equal", b, b},
		{"twice the divisor", mulSmall(b, 2), b},
		{"three times plus one", addOne(mulSmall(b, 3)), b},
		{"largest by largest", max, max},
		{"largest by 2^127", max, measure.Uint128{Hi: 1 << 63}},
		{"largest by 2^127 + 1", max, measure.Uint128{Hi: 1 << 63, Lo: 1}},
		{"zero by a high divisor", measure.Uint128{}, b},
		{"below the divisor", measure.Uint128{Lo: 9}, b},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q, r, ok := tt.a.DivRem(tt.b)
			wq, wr := new(big.Int).DivMod(bigOfU(tt.a), bigOfU(tt.b), new(big.Int))
			if !ok || bigOfU(q).Cmp(wq) != 0 || bigOfU(r).Cmp(wr) != 0 {
				t.Errorf("DivRem(%v, %v) = %v r %v (ok %v), want %v r %v", bigOfU(tt.a), bigOfU(tt.b), bigOfU(q), bigOfU(r), ok, wq, wr)
			}
		})
	}
	if _, _, ok := b.DivRem(measure.Uint128{}); ok {
		t.Error("DivRem by zero ok = true, want false")
	}
}

// mulSmall returns u * m for a product that fits.
func mulSmall(u measure.Uint128, m uint64) measure.Uint128 {
	p, _ := u.MulU64(m)
	return p
}

// addOne returns u + 1 for a sum that fits.
func addOne(u measure.Uint128) measure.Uint128 {
	s, _ := u.Add(measure.Uint128{Lo: 1})
	return s
}

// TestInt128_BoundaryAtTwoToThe127 verifies that a sum of the largest magnitude
// below 2^127 is accepted and one past it is refused, in both signs and for the
// difference, so the bound is exact.
//
// Requirement: FR-IND-009
func TestInt128_BoundaryAtTwoToThe127(t *testing.T) {
	t.Parallel()
	top, ok := measure.FromUint128(measure.Uint128{Hi: 1<<63 - 1, Lo: math.MaxUint64})
	if !ok {
		t.Fatal("FromUint128(2^127 - 1) ok = false, want true")
	}
	one, two := measure.FromInt64(1), measure.FromInt64(2)
	tests := []struct {
		name   string
		a, b   measure.Int128
		sub    bool
		wantOK bool
	}{
		{"largest less one plus one", mustAdd(top, one.Neg()), one, false, true},
		{"largest plus zero", top, measure.FromInt64(0), false, true},
		{"largest plus one", top, one, false, false},
		{"negative largest plus negative one", top.Neg(), one.Neg(), false, false},
		{"negative largest less one plus negative one", mustAdd(top.Neg(), one), one.Neg(), false, true},
		{"largest less negative one", top, one.Neg(), true, false},
		{"largest less one", top, one, true, true},
		{"opposite signs never overflow", top, top.Neg(), false, true},
		{"largest less two plus two", mustAdd(top, two.Neg()), two, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got measure.Int128
			var ok bool
			if tt.sub {
				got, ok = tt.a.Sub(tt.b)
			} else {
				got, ok = tt.a.Add(tt.b)
			}
			if ok != tt.wantOK {
				t.Fatalf("%s ok = %v, want %v", tt.name, ok, tt.wantOK)
			}
			if ok {
				want := new(big.Int).Add(bigOf(tt.a), bigOf(tt.b))
				if tt.sub {
					want = new(big.Int).Sub(bigOf(tt.a), bigOf(tt.b))
				}
				if bigOf(got).Cmp(want) != 0 {
					t.Errorf("%s = %v, want %v", tt.name, bigOf(got), want)
				}
			}
		})
	}
}

// mustAdd returns a + b for a sum that fits.
func mustAdd(a, b measure.Int128) measure.Int128 {
	s, _ := a.Add(b)
	return s
}
