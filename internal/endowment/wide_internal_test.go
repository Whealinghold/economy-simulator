package endowment

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func (x u128) big() *big.Int {
	b := new(big.Int).SetUint64(x.hi)
	return b.Lsh(b, 64).Or(b, new(big.Int).SetUint64(x.lo))
}

// wideSamples returns values at the edges of 64 and 128 bits and random ones.
func wideSamples() []u128 {
	edges := []uint64{0, 1, 2, 1 << 31, 1 << 32, 1<<63 - 1, 1 << 63, 1<<64 - 1}
	var out []u128
	for _, h := range edges {
		for _, l := range edges {
			out = append(out, u128{h, l})
		}
	}
	rng := rand.New(rand.NewPCG(4, 2))
	for range 400 {
		out = append(out, u128{rng.Uint64() >> rng.UintN(64), rng.Uint64() >> rng.UintN(64)})
	}
	return out
}

// TestWide_AgainstBig verifies mul, add, divmod64 and cmp of the local 128-bit
// value against math/big over edge values and random ones: the product and the
// sum are exact and report overflow past 128 bits, the division gives the
// quotient and remainder, and cmp orders by the high word first.
//
// Requirement: FR-RES-008
func TestWide_AgainstBig(t *testing.T) {
	t.Parallel()
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	samples := wideSamples()
	for _, x := range samples {
		for _, c := range []uint64{0, 1, 3, 1 << 63, 1<<64 - 1} {
			want := new(big.Int).Mul(x.big(), new(big.Int).SetUint64(c))
			got, ok := x.mul(c)
			if wantOK := want.Cmp(limit) < 0; ok != wantOK || (ok && got.big().Cmp(want) != 0) {
				t.Fatalf("%v.mul(%d) = %v, %v, want %v, %v", x, c, got, ok, want, wantOK)
			}
			if c == 0 {
				continue
			}
			q, r := x.divmod64(c)
			wq, wr := new(big.Int).DivMod(x.big(), new(big.Int).SetUint64(c), new(big.Int))
			if q.big().Cmp(wq) != 0 || new(big.Int).SetUint64(r).Cmp(wr) != 0 {
				t.Fatalf("%v.divmod64(%d) = %v rem %d, want %v rem %v", x, c, q, r, wq, wr)
			}
		}
		for _, y := range samples[:80] {
			want := new(big.Int).Add(x.big(), y.big())
			got, ok := x.add(y)
			if wantOK := want.Cmp(limit) < 0; ok != wantOK || (ok && got.big().Cmp(want) != 0) {
				t.Fatalf("%v.add(%v) = %v, %v, want %v, %v", x, y, got, ok, want, wantOK)
			}
			if got, want := x.cmp(y), x.big().Cmp(y.big()); got != want {
				t.Fatalf("%v.cmp(%v) = %d, want %d", x, y, got, want)
			}
		}
	}
}

// TestWide_Mul64AndProduct verifies mul64 is the full 128-bit product and
// product reports an overflow of any of its three steps.
//
// Requirement: FR-RES-008
func TestWide_Mul64AndProduct(t *testing.T) {
	t.Parallel()
	if got, want := mul64(1<<64-1, 1<<64-1).big(), new(big.Int).Mul(new(big.Int).SetUint64(1<<64-1), new(big.Int).SetUint64(1<<64-1)); got.Cmp(want) != 0 {
		t.Errorf("mul64(max, max) = %v, want %v", got, want)
	}
	if p, ok := product(1<<63, 1<<63, 4, 1); ok {
		t.Errorf("product(2^63, 2^63, 4, 1) = %v, true, want overflow (2^128 does not fit)", p)
	}
	if p, ok := product(1<<63, 1<<63, 1, 4); ok {
		t.Errorf("product(2^63, 2^63, 1, 4) = %v, true, want overflow in the last step", p)
	}
	if p, ok := product(1<<63, 1<<63, 1, 1); !ok || p.big().Cmp(new(big.Int).Lsh(big.NewInt(1), 126)) != 0 {
		t.Errorf("product(2^63, 2^63, 1, 1) = %v, %v, want 2^126, true", p, ok)
	}
}
