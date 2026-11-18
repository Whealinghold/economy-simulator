package risk_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/risk"
)

// FuzzProb verifies the effective probability never panics and stays in 0 to
// 10 000 for every non-negative input, and that a scale of 0 gives 0.
//
// Requirement: FR-RSK-006, FR-RSK-040
func FuzzProb(f *testing.F) {
	f.Add(int64(3_000), int64(10_000))
	f.Add(int64(10_000), int64(100_000))
	f.Add(int64(0), int64(0))
	f.Fuzz(func(t *testing.T, strike, scale int64) {
		p, ok := risk.Prob(strike, scale)
		if strike < 0 || scale < 0 {
			if ok {
				t.Fatalf("Prob(%d, %d) ok, want refused", strike, scale)
			}
			return
		}
		if !ok || p < 0 || p > 10_000 {
			t.Fatalf("Prob(%d, %d) = %d, %v, want 0 to 10000", strike, scale, p, ok)
		}
		if scale == 0 && p != 0 {
			t.Fatalf("Prob(%d, 0) = %d, want 0", strike, p)
		}
	})
}

// FuzzFold verifies the combined loss of up to four severities stays in 0 to
// 10 000, never falls when a row is added and is 10 000 once a total row is
// met.
//
// Requirement: FR-RSK-009, FR-RSK-040
func FuzzFold(f *testing.F) {
	f.Add(int64(2_500), int64(4_000), int64(0), int64(0))
	f.Add(int64(10_000), int64(1), int64(1), int64(1))
	f.Fuzz(func(t *testing.T, a, b, c, d int64) {
		sev := []int64{a % 10_001, b % 10_001, c % 10_001, d % 10_001}
		for i := range sev {
			if sev[i] < 0 {
				sev[i] = -sev[i]
			}
		}
		prev := int64(0)
		for n := 1; n <= len(sev); n++ {
			got, ok := risk.Combine(sev[:n])
			if !ok || got < prev || got > 10_000 {
				t.Fatalf("Combine(%v) = %d, %v after %d, want a loss from %d to 10000", sev[:n], got, ok, prev, prev)
			}
			if sev[n-1] == 10_000 && got != 10_000 {
				t.Fatalf("Combine(%v) = %d, want 10000 with a total row", sev[:n], got)
			}
			prev = got
		}
	})
}

// FuzzPhi verifies the stored fraction never panics, is 0 or refused outside
// its domain and never exceeds 10 000 when the stored load is within the load
// and the capacity within the sum.
//
// Requirement: FR-RSK-017, FR-RSK-040
func FuzzPhi(f *testing.F) {
	f.Add(int64(450), int64(500), int64(200), int64(550))
	f.Add(int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62)
	f.Fuzz(func(t *testing.T, stored, sumCap, capK, load int64) {
		phi, ok := risk.Phi(stored, sumCap, capK, load)
		if !ok {
			if phi != 0 {
				t.Fatalf("Phi refused with a non-zero result %d", phi)
			}
			return
		}
		if stored >= 0 && sumCap > 0 && capK <= sumCap && stored <= load && load > 0 && phi > 10_000 {
			t.Fatalf("Phi(%d, %d, %d, %d) = %d, want at most 10000", stored, sumCap, capK, load, phi)
		}
	})
}
