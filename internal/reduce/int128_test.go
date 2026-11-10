package reduce_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// TestInt128_Checked verifies that Add, Sub, Neg and Mul agree with big.Int when
// the result fits 128 bits and report false when it does not: 2^126 + 2^126 and
// 2^64 * 2^64 * 2^... carry across the words, the minimum has no negation.
//
// Requirement: FR-EXP-040
//
// Guard: a wrapped sum looks like a small one. A reduction that wrapped at 2^127
// would report a plausible mean for readings that overflowed.
func TestInt128_Checked(t *testing.T) {
	t.Parallel()
	pow := func(e uint) *big.Int { return new(big.Int).Lsh(big.NewInt(1), e) }
	cases := []struct {
		name string
		a, b *big.Int
	}{
		{"carry across the words", new(big.Int).SetUint64(math.MaxUint64), big.NewInt(1)},
		{"2^126 plus 2^126 is 2^127, out of range", pow(126), pow(126)},
		{"minimum minus one", min128, big.NewInt(1)},
		{"maximum plus one", max128, big.NewInt(1)},
		{"minimum times minus one", min128, big.NewInt(-1)},
		{"2^63 times 2^63 fits", pow(63), pow(63)},
		{"2^64 times 2^64 does not", pow(64), pow(64)},
		{"negative times positive", big.NewInt(-3), pow(100)},
		{"zero", big.NewInt(0), pow(100)},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a, b := fromBig(tt.a), fromBig(tt.b)
			check := func(op string, got reduce.Int128, ok bool, want *big.Int) {
				if ok != fits(want) {
					t.Errorf("%s(%v, %v) ok = %v, want %v", op, tt.a, tt.b, ok, fits(want))
				} else if ok && bigOf(got).Cmp(want) != 0 {
					t.Errorf("%s(%v, %v) = %v, want %v", op, tt.a, tt.b, got, want)
				}
			}
			s, ok := a.Add(b)
			check("Add", s, ok, new(big.Int).Add(tt.a, tt.b))
			d, ok := a.Sub(b)
			check("Sub", d, ok, new(big.Int).Sub(tt.a, tt.b))
			p, ok := a.Mul(b)
			check("Mul", p, ok, new(big.Int).Mul(tt.a, tt.b))
			n, ok := a.Neg()
			check("Neg", n, ok, new(big.Int).Neg(tt.a))
			if got, want := a.Cmp(b), tt.a.Cmp(tt.b); got != want {
				t.Errorf("Cmp(%v, %v) = %d, want %d", tt.a, tt.b, got, want)
			}
		})
	}
}

// TestInt128_RandomAgainstBig verifies the four operations and Cmp on 20 000
// random pairs of every width against big.Int, seed 7.
//
// Requirement: FR-EXP-040
func TestInt128_RandomAgainstBig(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(7, 7))
	for i := range 20_000 {
		a, b := randInt128(r), randInt128(r)
		ba, bb := bigOf(a), bigOf(b)
		for _, op := range []struct {
			name string
			got  func() (reduce.Int128, bool)
			want *big.Int
		}{
			{"Add", func() (reduce.Int128, bool) { return a.Add(b) }, new(big.Int).Add(ba, bb)},
			{"Sub", func() (reduce.Int128, bool) { return a.Sub(b) }, new(big.Int).Sub(ba, bb)},
			{"Mul", func() (reduce.Int128, bool) { return a.Mul(b) }, new(big.Int).Mul(ba, bb)},
			{"Neg", func() (reduce.Int128, bool) { return a.Neg() }, new(big.Int).Neg(ba)},
		} {
			got, ok := op.got()
			if ok != fits(op.want) || (ok && bigOf(got).Cmp(op.want) != 0) {
				t.Fatalf("case %d: %s(%v, %v) = %v, %v; want %v (fits %v)", i, op.name, ba, bb, got, ok, op.want, fits(op.want))
			}
		}
		if got, want := a.Cmp(b), ba.Cmp(bb); got != want {
			t.Fatalf("case %d: Cmp(%v, %v) = %d, want %d", i, ba, bb, got, want)
		}
	}
}

// TestInt128_Int64AndSign verifies the conversions at the int64 edges and Sign.
//
// Requirement: FR-EXP-040
func TestInt128_Int64AndSign(t *testing.T) {
	t.Parallel()
	for _, v := range []int64{0, 1, -1, math.MaxInt64, math.MinInt64} {
		got, ok := reduce.FromInt64(v).Int64()
		if !ok || got != v {
			t.Errorf("FromInt64(%d).Int64() = %d, %v, want the same value", v, got, ok)
		}
	}
	over, _ := reduce.FromInt64(math.MaxInt64).Add(reduce.FromInt64(1))
	if _, ok := over.Int64(); ok {
		t.Errorf("(MaxInt64 + 1).Int64() ok = true, want false")
	}
	under, _ := reduce.FromInt64(math.MinInt64).Sub(reduce.FromInt64(1))
	if _, ok := under.Int64(); ok {
		t.Errorf("(MinInt64 - 1).Int64() ok = true, want false")
	}
	for v, want := range map[int64]int{-5: -1, 0: 0, 5: 1} {
		if got := reduce.FromInt64(v).Sign(); got != want {
			t.Errorf("FromInt64(%d).Sign() = %d, want %d", v, got, want)
		}
	}
	hi, lo := reduce.FromWords(-3, 9).Words()
	if hi != -3 || lo != 9 {
		t.Errorf("FromWords(-3, 9).Words() = %d, %d, want -3, 9", hi, lo)
	}
}

// TestInt128_String verifies the decimal text at the widths where its 19-digit
// chunks change: zero, one word, exactly 10^19, the extremes.
//
// Requirement: FR-EXP-040
func TestInt128_String(t *testing.T) {
	t.Parallel()
	for _, s := range []string{
		"0", "-1", "18446744073709551615", "18446744073709551616", "10000000000000000000",
		"9999999999999999999", "100000000000000000000000000000000000000",
		min128.String(), max128.String(), "-340282366920938463463374607431768211",
	} {
		b, _ := new(big.Int).SetString(s, 10)
		if got := fromBig(b).String(); got != s {
			t.Errorf("String of %s = %s, want it unchanged", s, got)
		}
	}
}

// TestInt128_MinimumByMultiplication verifies the one magnitude that fits only as
// a negative number: -2^63 * 2^64 is exactly the minimum, and one more in
// magnitude is refused: -3 * ((2^127 + 1) / 3) = -(2^127 + 1).
//
// Requirement: FR-EXP-040
func TestInt128_MinimumByMultiplication(t *testing.T) {
	t.Parallel()
	pow64 := fromBig(new(big.Int).Lsh(big.NewInt(1), 64))
	got, ok := reduce.FromInt64(math.MinInt64).Mul(pow64)
	if !ok || bigOf(got).Cmp(min128) != 0 {
		t.Errorf("(-2^63) * 2^64 = %v, %v, want %v", got, ok, min128)
	}
	third := new(big.Int).Div(new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1)), big.NewInt(3))
	if p, ok := reduce.FromInt64(-3).Mul(fromBig(third)); ok {
		t.Errorf("-3 * (2^127 + 1)/3 = %v, true, want false: -(2^127 + 1) is below the minimum", p)
	}
	if p, ok := reduce.FromInt64(3).Mul(fromBig(third)); ok {
		t.Errorf("3 * (2^127 + 1)/3 = %v, true, want false: 2^127 + 1 is above the maximum", p)
	}
}

// TestRDiv_WideDivisors verifies the two-word divisor path on the cases the
// random test reaches rarely: exact multiples (the remainder equals the divisor
// when the step is off by one), an exact half, the minimum numerator (whose
// magnitude 2^127 uses the top bit of the loop), and a divisor of 2^127.
//
// Requirement: FR-EXP-040
func TestRDiv_WideDivisors(t *testing.T) {
	t.Parallel()
	two64 := new(big.Int).Lsh(big.NewInt(1), 64)
	d := new(big.Int).Add(two64, big.NewInt(1)) // 2^64 + 1: two words
	cases := []struct {
		name string
		n, d *big.Int
	}{
		{"exact multiple", new(big.Int).Mul(d, big.NewInt(3)), d},
		{"exact multiple, negative", new(big.Int).Mul(d, big.NewInt(-3)), d},
		{"just under a multiple", new(big.Int).Sub(new(big.Int).Mul(d, big.NewInt(3)), big.NewInt(1)), d},
		{"just over a multiple", new(big.Int).Add(new(big.Int).Mul(d, big.NewInt(3)), big.NewInt(1)), d},
		// 2^64 * 5 + 2^63 over 2^65: 2.5 + 0.25 ... use d = 2^65 and n = 5 * 2^64 = 2.5 d: an exact half.
		{"exact half, even neighbour below", new(big.Int).Mul(big.NewInt(5), two64), new(big.Int).Lsh(big.NewInt(1), 65)},
		{"exact half, even neighbour above", new(big.Int).Mul(big.NewInt(7), two64), new(big.Int).Lsh(big.NewInt(1), 65)},
		{"minimum over a wide divisor", min128, d},
		{"minimum over minus a wide divisor", min128, new(big.Int).Neg(d)},
		{"minimum over 2^127 magnitude", min128, min128},
		{"maximum over the minimum", max128, min128},
		{"wide over larger", d, new(big.Int).Lsh(big.NewInt(1), 100)},
		{"numerator of one word, divisor of two", big.NewInt(12345), d},
	}
	for _, tt := range cases {
		want := refRDiv(tt.n, tt.d)
		got, ok := reduce.RDiv(fromBig(tt.n), fromBig(tt.d))
		if ok != fits(want) || (ok && bigOf(got).Cmp(want) != 0) {
			t.Errorf("%s: RDiv(%v, %v) = %v, %v, want %v", tt.name, tt.n, tt.d, got, ok, want)
		}
	}
}
