package reduce_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// TestRDiv_HalfEven verifies the rounding rule: an exact half goes to the even
// neighbour, from either sign, and a zero divisor reports false.
//
// Requirement: FR-EXP-040
func TestRDiv_HalfEven(t *testing.T) {
	t.Parallel()
	cases := []struct {
		n, d, want int64
		ok         bool
	}{
		{-5, 2, -2, true},  // -2.5 -> -2
		{-3, 2, -2, true},  // -1.5 -> -2
		{25, 10, 2, true},  // 2.5 -> 2
		{35, 10, 4, true},  // 3.5 -> 4
		{5, -2, -2, true},  // sign of the divisor counts
		{-5, -2, 2, true},  // 2.5 -> 2
		{7, 3, 2, true},    // 2.33 -> 2
		{8, 3, 3, true},    // 2.67 -> 3
		{0, 5, 0, true},    // zero numerator
		{1, 0, 0, false},   // zero divisor
		{-7, 3, -2, true},  // -2.33 -> -2
		{-8, 3, -3, true},  // -2.67 -> -3
		{4, 1, 4, true},    // exact
		{1, 2, 0, true},    // 0.5 -> 0
		{3, 2, 2, true},    // 1.5 -> 2
		{-1, 2, 0, true},   // -0.5 -> 0 (even)
		{-1, 3, 0, true},   // below a half
		{-2, 3, -1, true},  // above a half
		{99, 100, 1, true}, // just under 1
	}
	for _, tt := range cases {
		got, ok := reduce.RDiv(reduce.FromInt64(tt.n), reduce.FromInt64(tt.d))
		v, _ := got.Int64()
		if ok != tt.ok || (ok && v != tt.want) {
			t.Errorf("RDiv(%d, %d) = %d, %v, want %d, %v", tt.n, tt.d, v, ok, tt.want, tt.ok)
		}
	}
	// The one quotient out of range: -2^127 / -1.
	if _, ok := reduce.RDiv(fromBig(min128), reduce.FromInt64(-1)); ok {
		t.Errorf("RDiv(min, -1) ok = true, want false")
	}
}

// TestRDiv_AgainstBig verifies 20 000 random 128-bit quotients against the
// rounding rule written on big.Int, both divisor widths (one word takes the
// hardware path, two words the bit loop), seed 11.
//
// Requirement: FR-EXP-040
func TestRDiv_AgainstBig(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(11, 11))
	for i := range 20_000 {
		n, d := randInt128(r), randInt128(r)
		if i%3 == 0 { // force exact halves and exact quotients to occur
			d = reduce.FromInt64(int64(r.UintN(20)) - 10)
			if p, ok := n.Mul(d); ok && i%2 == 0 {
				n = p
			}
		}
		bn, bd := bigOf(n), bigOf(d)
		got, ok := reduce.RDiv(n, d)
		if bd.Sign() == 0 {
			if ok {
				t.Fatalf("case %d: RDiv(%v, 0) ok = true, want false", i, bn)
			}
			continue
		}
		want := refRDiv(bn, bd)
		if ok != fits(want) || (ok && bigOf(got).Cmp(want) != 0) {
			t.Fatalf("case %d: RDiv(%v, %v) = %v, %v; want %v", i, bn, bd, got, ok, want)
		}
	}
}

// TestIsqrtNearest verifies the nearest root at the edges of each interval:
// s*s, s*s+s (still rounds down), s*s+s+1 (rounds up) for s up to 2^63 - 1, and
// 2^126, against big.Int.Sqrt.
//
// Requirement: FR-EXP-040, FR-EXP-045
func TestIsqrtNearest(t *testing.T) {
	t.Parallel()
	var ss []uint64
	for s := uint64(0); s < 40; s++ {
		ss = append(ss, s)
	}
	ss = append(ss, 1<<32-1, 1<<32, 1<<32+1, math.MaxInt64-1, math.MaxInt64)
	for _, s := range ss {
		bs := new(big.Int).SetUint64(s)
		sq := new(big.Int).Mul(bs, bs)
		for _, c := range []struct {
			name string
			n    *big.Int
			want uint64
		}{
			{"s*s", sq, s},
			{"s*s+s", new(big.Int).Add(sq, bs), s},
			{"s*s+s+1", new(big.Int).Add(new(big.Int).Add(sq, bs), big.NewInt(1)), s + 1},
		} {
			if got := reduce.IsqrtNearest(fromBig(c.n)); got != c.want {
				t.Errorf("IsqrtNearest(%s of %d = %v) = %d, want %d", c.name, s, c.n, got, c.want)
			}
		}
	}
	pow126 := new(big.Int).Lsh(big.NewInt(1), 126)
	if got := reduce.IsqrtNearest(fromBig(pow126)); got != 1<<63 {
		t.Errorf("IsqrtNearest(2^126) = %d, want 2^63", got)
	}
	if got := reduce.IsqrtNearest(fromBig(max128)); got != 13043817825332782212 {
		// floor root 13043817825332782212 with remainder 9119501915260492783, below the root: rounds down.
		t.Errorf("IsqrtNearest(2^127 - 1) = %d, want 13043817825332782212", got)
	}
	if got := reduce.IsqrtNearest(reduce.FromInt64(-4)); got != 0 {
		t.Errorf("IsqrtNearest(-4) = %d, want 0", got)
	}
}

// TestIsqrtNearest_RandomAgainstBig verifies 20 000 random values against
// big.Int.Sqrt and the nearest rule, seed 13.
//
// Requirement: FR-EXP-040, FR-EXP-045
func TestIsqrtNearest_RandomAgainstBig(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(13, 13))
	for i := range 20_000 {
		n := randInt128(r)
		if n.IsNeg() {
			n, _ = n.Neg()
		}
		if got, want := reduce.IsqrtNearest(n), refIsqrt(bigOf(n)); new(big.Int).SetUint64(got).Cmp(want) != 0 {
			t.Fatalf("case %d: IsqrtNearest(%v) = %d, want %v", i, n, got, want)
		}
	}
}

func refIsqrt(n *big.Int) *big.Int {
	f := new(big.Int).Sqrt(n)
	rem := new(big.Int).Sub(n, new(big.Int).Mul(f, f))
	if rem.Cmp(f) > 0 {
		f.Add(f, big.NewInt(1))
	}
	return f
}

// FuzzRDiv checks, for any n and d, that RDiv does not panic and that its result
// is within half of d of the exact quotient (|n - q*d| <= |d|/2).
//
// Requirement: NFR-EXP-005
func FuzzRDiv(f *testing.F) {
	f.Add(int64(0), uint64(25), int64(0), uint64(10))
	f.Add(int64(-1), uint64(1<<63), int64(0), uint64(2))
	f.Add(int64(1<<62), uint64(7), int64(3), uint64(1<<63))
	f.Fuzz(func(t *testing.T, nh int64, nl uint64, dh int64, dl uint64) {
		n, d := reduce.FromWords(nh, nl), reduce.FromWords(dh, dl)
		q, ok := reduce.RDiv(n, d)
		bn, bd := bigOf(n), bigOf(d)
		if bd.Sign() == 0 {
			if ok {
				t.Fatalf("RDiv(%v, 0) ok = true", bn)
			}
			return
		}
		if !ok {
			if fits(refRDiv(bn, bd)) {
				t.Fatalf("RDiv(%v, %v) ok = false, but the quotient fits", bn, bd)
			}
			return
		}
		diff := new(big.Int).Sub(bn, new(big.Int).Mul(bigOf(q), bd))
		if new(big.Int).Lsh(new(big.Int).Abs(diff), 1).Cmp(new(big.Int).Abs(bd)) > 0 {
			t.Fatalf("RDiv(%v, %v) = %v: |n - q*d| = %v is above |d|/2", bn, bd, q, diff)
		}
	})
}

// FuzzIsqrtNearest checks, for any non-negative n, that the result r satisfies
// (r - 1/2)^2 < n < (r + 1/2)^2, which is the nearest rule without a tie.
//
// Requirement: NFR-EXP-005
func FuzzIsqrtNearest(f *testing.F) {
	f.Add(int64(0), uint64(0))
	f.Add(int64(0), uint64(1<<63))
	f.Add(int64(1<<62), uint64(12345))
	f.Fuzz(func(t *testing.T, hi int64, lo uint64) {
		n := reduce.FromWords(hi, lo)
		if n.IsNeg() {
			return
		}
		r := new(big.Int).SetUint64(reduce.IsqrtNearest(n))
		// (2r - 1)^2 < 4n < (2r + 1)^2
		four := new(big.Int).Lsh(bigOf(n), 2)
		two := new(big.Int).Lsh(r, 1)
		lowSq := new(big.Int).Mul(new(big.Int).Sub(two, big.NewInt(1)), new(big.Int).Sub(two, big.NewInt(1)))
		highSq := new(big.Int).Mul(new(big.Int).Add(two, big.NewInt(1)), new(big.Int).Add(two, big.NewInt(1)))
		if r.Sign() == 0 {
			lowSq.SetInt64(0) // the lower bound at 0 is 0, not 1/4
		}
		if lowSq.Cmp(four) > 0 || four.Cmp(highSq) >= 0 {
			t.Fatalf("IsqrtNearest(%v) = %v is not the nearest root", bigOf(n), r)
		}
	})
}
