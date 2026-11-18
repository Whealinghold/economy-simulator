package fiscal_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// FuzzDue verifies, for any weight, balance, carry and rule, that Due never
// panics, never reports a collection above the due or above what is held above the
// exemption, and a carry outside [0, period*10^4 or 10^8) only when it was given
// outside it.
//
// Requirement: NFR-FIS-003
func FuzzDue(f *testing.F) {
	f.Add(int64(1), int64(3_000), int64(60), int64(0), int64(10_000), int64(1), int64(0))
	f.Add(int64(1), int64(2_940), int64(101), int64(2_000), int64(10_000), int64(1), int64(0))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(10_001), int64(0), int64(100_000), int64(365), int64(-1))
	f.Fuzz(func(t *testing.T, w, bal, a, exempt, scale, period, carry int64) {
		// The low bit of a picks the shape and the rest is its amount or its rate.
		r := fiscal.Rule{Shape: fiscal.Shape(a & 1), Amount: a >> 1, RateBP: a >> 1, Exempt: exempt,
			Scale: scale, Period: period, MaxMoney: maxMoney}
		got, ok := r.Due(w, bal, carry)
		if !ok {
			return
		}
		if got.Due < 0 || got.Collected < 0 || got.Collected > got.Due || got.Collected > bal || got.Carry < 0 {
			t.Fatalf("Due(%d, %d, %d) = %+v for %+v: out of range", w, bal, carry, got, r)
		}
	})
}

// FuzzCut verifies that Cut never panics and, on any input it accepts, uses
// exactly min(avail, total due), gives no recipient above its due and none a
// negative amount.
//
// Requirement: NFR-FIS-003
func FuzzCut(f *testing.F) {
	f.Add(int64(4_135), int64(500), int64(500), int64(300), uint64(1), uint64(2), uint64(3))
	f.Add(int64(0), int64(1), int64(0), int64(math.MaxInt64), uint64(0), uint64(0), uint64(0))
	f.Fuzz(func(t *testing.T, avail, d0, d1, d2 int64, k0, k1, k2 uint64) {
		rs := []fiscal.Recipient{{Due: d0, Key: k0, ID: 0}, {Due: d1, Key: k1, ID: 1}, {Due: d2, Key: k2, ID: 2}}
		var s fiscal.Scratch
		got, ok := s.Cut(nil, avail, rs)
		if !ok {
			return
		}
		total, sum := new(big.Int), new(big.Int)
		for i, a := range got {
			total.Add(total, big.NewInt(rs[i].Due))
			sum.Add(sum, big.NewInt(a))
			if a < 0 || a > rs[i].Due {
				t.Fatalf("Cut(%d, %v)[%d] = %d outside [0, due]", avail, rs, i, a)
			}
		}
		want := new(big.Int).Set(total)
		if total.Cmp(big.NewInt(avail)) > 0 {
			want.SetInt64(avail)
		}
		if sum.Cmp(want) != 0 {
			t.Fatalf("Cut(%d, %v) shares sum to %v, want %v", avail, rs, sum, want)
		}
	})
}

// FuzzToll verifies that Toll never panics, reports an overflow instead of
// wrapping, and equals the big-integer quotient when it is ok.
//
// Requirement: NFR-FIS-003
func FuzzToll(f *testing.F) {
	f.Add(int64(3), int64(5_000), int64(3))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64))
	f.Fuzz(func(t *testing.T, amount, scale, n int64) {
		got, ok := fiscal.Toll(amount, scale, n)
		if amount < 0 || scale < 0 || n < 0 {
			if ok {
				t.Fatalf("Toll(%d, %d, %d) = %d, true for a negative argument", amount, scale, n, got)
			}
			return
		}
		want := new(big.Int).Mul(big.NewInt(amount), big.NewInt(scale))
		want.Mul(want, big.NewInt(max(1, n)))
		want.Div(want, big.NewInt(10_000))
		if ok != want.IsInt64() || (ok && big.NewInt(got).Cmp(want) != 0) {
			t.Fatalf("Toll(%d, %d, %d) = %d, %v; exact quotient %v", amount, scale, n, got, ok, want)
		}
	})
}
