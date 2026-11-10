package indicator_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/measure"
	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// toReduce converts a measure.Int128 to a reduce.Int128 through its sign and
// magnitude, so the two implementations are compared by value and not by layout.
func toReduce(v measure.Int128) (reduce.Int128, bool) {
	m := v.Mag()
	if m.Hi>>63 != 0 { // beyond 2^127 - 1: the test's operands never reach it
		return reduce.Int128{}, false
	}
	r := reduce.FromWords(int64(m.Hi), m.Lo)
	if v.Sign() < 0 {
		return r.Neg()
	}
	return r, true
}

// lcg is a deterministic stream of 64-bit words for the cases below: the test
// needs varied inputs, not randomness (architecture.md 7.3).
type lcg uint64

func (s *lcg) next() uint64 {
	*s = *s*6364136223846793005 + 1442695040888963407
	return uint64(*s)
}

// TestReduceAgreesWithMeasure verifies that the harness's integer helpers
// (internal/reduce) and the indicators' (internal/measure) are the same rule, so
// the two implementations cannot drift: the half-to-even division on 128-bit
// numerators, the nearest-rank quantile on weight-1 rows, the nearest integer
// root, and the exact sums.
//
// Requirement: FR-EXP-040, FR-EXP-042
func TestReduceAgreesWithMeasure(t *testing.T) {
	t.Parallel()
	s := lcg(7)
	t.Run("half to even division", func(t *testing.T) {
		for i := range 5000 {
			a, b, d := int64(s.next()), int64(s.next()), int64(s.next()>>(s.next()%60))
			// A product of two int64 is a 128-bit numerator; the ties come from the small ones.
			n := measure.MulInt64(a>>(uint(i)%40), b>>(uint(i)%23))
			if i%7 == 0 {
				n, d = measure.FromInt64(int64(i%50)-25), int64(2+i%4)
			}
			want, okM := measure.RoundDiv(n, measure.FromInt64(d))
			rn, okN := toReduce(n)
			got, okR := reduce.RDiv(rn, reduce.FromInt64(d))
			if okM != okR || !okN {
				t.Fatalf("case %d: RoundDiv(%v, %d) ok = %v, RDiv ok = %v (operand converted: %v)", i, n, d, okM, okR, okN)
			}
			if w, ok := toReduce(want); okM && (!ok || got.Cmp(w) != 0) {
				t.Fatalf("case %d: RDiv(%s, %d) = %s, RoundDiv = %v", i, rn.String(), d, got.String(), want)
			}
		}
	})
	t.Run("nearest rank quantile", func(t *testing.T) {
		for n := 1; n <= 40; n++ {
			vals, rows := make([]int64, n), make([]measure.Row, n)
			for i := range vals {
				vals[i] = int64(s.next()%2001) - 1000
			}
			slices.Sort(vals)
			for i, v := range vals {
				rows[i] = measure.Row{Weight: 1, Total: v}
			}
			for _, q := range []int{0, 1000, 2500, 5000, 7500, 9000, 9999, 10000} {
				want, ok, err := measure.Quantile(rows, int64(q))
				if err != nil || !ok || reduce.Quantile(vals, q) != want {
					t.Fatalf("n %d q %d: Quantile(%v) = %d, measure.Quantile = %d (ok %v, err %v)", n, q, vals, reduce.Quantile(vals, q), want, ok, err)
				}
			}
		}
	})
	t.Run("nearest integer root", func(t *testing.T) {
		edges := []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 1<<32 - 1, 1 << 32, 1<<63 - 1, 1 << 63, 1<<64 - 1}
		for r := uint64(1); r < 1<<32; r = r*3 + 1 { // s*s, s*s + s and s*s + s + 1 for a spread of s
			edges = append(edges, r*r, r*r+r, r*r+r+1)
		}
		for i := 0; i < 3000; i++ {
			edges = append(edges, s.next()>>(s.next()%64))
		}
		for _, n := range edges {
			if got, want := reduce.IsqrtNearest(reduce.FromWords(0, n)), measure.IsqrtNearest(n); got != want {
				t.Fatalf("IsqrtNearest(%d) = %d, measure.IsqrtNearest = %d", n, got, want)
			}
		}
	})
	t.Run("exact sums", func(t *testing.T) {
		var mo reduce.Moments
		sum := measure.FromInt64(0)
		for i := 0; i < 2000; i++ {
			v := int64(s.next()) >> (24 + s.next()%8) // within MaxReading (2^40): Moments squares it in 128 bits
			var ok bool
			if sum, ok = sum.Add(measure.FromInt64(v)); !ok || !mo.Add(v) {
				t.Fatalf("reading %d: a sum overflowed", i)
			}
		}
		if w, ok := toReduce(sum); !ok || mo.Sum().Cmp(w) != 0 {
			t.Errorf("sum of 2000 readings = %s, measure's = %v", mo.Sum().String(), sum)
		}
	})
}
