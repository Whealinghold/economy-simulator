package insure_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/insure"
)

// FuzzPremium checks the premium for any input: it never panics, a result that is
// ok is within the limit and non-negative, and the maximum payout never exceeds the
// sum it is a share of.
//
// Requirement: FR-INS-022, FR-INS-023, NFR-INS-005
func FuzzPremium(f *testing.F) {
	f.Add(int64(1134), int64(1305), int64(8000), int64(13000))
	f.Add(maxMoney, int64(10_000), int64(10_000), int64(100_000))
	f.Add(int64(-1), int64(0), int64(0), int64(0))
	f.Add(int64(1<<63-1), int64(1<<63-1), int64(1<<63-1), int64(1<<63-1))
	f.Fuzz(func(t *testing.T, sum, loss, cover, loading int64) {
		p, rate, ok := insure.Premium(sum, loss, cover, loading, maxMoney)
		if ok && (p < 0 || p > maxMoney || rate < 0) {
			t.Errorf("Premium(%d, %d, %d, %d) = %d, %d, true, want within [0, MaxMoney]", sum, loss, cover, loading, p, rate)
		}
		if m, ok := insure.MaxPayout(cover, sum); ok && cover <= insure.Basis && m > sum {
			t.Errorf("MaxPayout(%d, %d) = %d, want at most the sum", cover, sum, m)
		}
	})
}

// FuzzSum checks the sums for any input: they never panic and a result that is ok
// is within the limit and non-negative, and the store sum never exceeds the held
// value it is a share of.
//
// Requirement: FR-INS-021, NFR-INS-005
func FuzzSum(f *testing.F) {
	f.Add(int64(40), int64(12), int64(5), int64(90), int64(5000))
	f.Add(int64(1<<62), int64(4), int64(1), int64(1), int64(10_000))
	f.Add(int64(-1), int64(1), int64(1), int64(1), int64(-1))
	f.Fuzz(func(t *testing.T, q1, v1, q2, v2, share int64) {
		qty, value := []int64{q1, q2}, []int64{v1, v2}
		if s, ok := insure.SumLines(qty, value, maxMoney); ok && (s < 0 || s > maxMoney) {
			t.Errorf("SumLines(%v, %v) = %d, true, want within [0, MaxMoney]", qty, value, s)
		}
		held, ok := insure.SumAccount(qty, value, -1, maxMoney)
		s, sok := insure.SumStore(qty, value, share, maxMoney)
		if sok && (!ok || s < 0 || (share <= insure.Basis && s > held)) {
			t.Errorf("SumStore(%v, %v, %d) = %d, true, with held %d, %v, want a share of held", qty, value, share, s, held, ok)
		}
	})
}

// FuzzUnitValue checks the unit value for any input: it never panics and a result
// that is ok is neither negative nor above the published price.
//
// Requirement: FR-INS-020, NFR-INS-005
func FuzzUnitValue(f *testing.F) {
	f.Add(int64(12), int64(10), int64(12_000))
	f.Add(int64(1<<63-1), int64(1<<63-1), int64(1<<63-1))
	f.Add(int64(-1), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, published, reference, capBP int64) {
		v, ok := insure.UnitValue(published, reference, capBP)
		if ok && (v < 0 || v > published) {
			t.Errorf("UnitValue(%d, %d, %d) = %d, true, want within [0, published]", published, reference, capBP, v)
		}
	})
}

// FuzzDue checks the due for any input: it never panics and is never above the
// maximum payout or negative.
//
// Requirement: FR-INS-042, NFR-INS-005
func FuzzDue(f *testing.F) {
	f.Add(int64(907), int64(8000), int64(246))
	f.Add(int64(1<<63-1), int64(1<<63-1), int64(1<<63-1))
	f.Fuzz(func(t *testing.T, maxPayout, cover, loss int64) {
		d, _ := insure.Due(maxPayout, cover, loss)
		if d < 0 || (maxPayout >= 0 && d > maxPayout) {
			t.Errorf("Due(%d, %d, %d) = %d, want within [0, maxPayout]", maxPayout, cover, loss, d)
		}
	})
}
