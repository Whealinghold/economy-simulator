package saving_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/saving"
)

// FuzzProject checks, over the bounds of the spec, that no function panics, that
// a job value and a benefit do not fall as the capacity rises, that a gain is
// never negative and that the test never passes a project of benefit 0 and
// never fails a larger benefit where a smaller one passed.
//
// Requirement: NFR-SAV-003, FR-SAV-043
func FuzzProject(f *testing.F) {
	f.Add(int64(200_000), int64(80_000), int64(1), int64(0), int64(6), int64(15_000), int64(400))
	f.Add(int64(1)<<62, int64(1)<<62, int64(1_000_000), int64(3), int64(4), int64(200_000), int64(maxMoney))
	f.Fuzz(func(t *testing.T, pvB, pvA, w, k, more, hurdle, cost int64) {
		if k < 0 || more < 0 || k+more < k {
			t.Skip("capacity outside the domain")
		}
		lo := saving.JobValue(pvB, pvA, w, k)
		hi := saving.JobValue(pvB, pvA, w, k+more)
		if hi < lo {
			t.Fatalf("J fell as the capacity rose: %d then %d", lo, hi)
		}
		gain := saving.BuildGain(lo, hi)
		checkBenefit(t, gain)
		checkTest(t, saving.SiteBenefit(gain, 180, maxMoney), cost, hurdle)
		_ = saving.Outlay(cost, gain, w, k, more)
		_ = saving.Reach(cost, k, gain, more)
	})
}

// checkBenefit checks that a benefit stays within [0, MaxMoney] and does not
// fall with the service time.
func checkBenefit(t *testing.T, gain int64) {
	t.Helper()
	b1 := saving.SiteBenefit(gain, 180, maxMoney)
	b2 := saving.SiteBenefit(gain, 181, maxMoney)
	if gain < 0 || b1 < 0 || b1 > maxMoney || b2 < b1 {
		t.Fatalf("gain %d, benefit %d then %d, want within [0, MaxMoney] and not falling with the service time", gain, b1, b2)
	}
}

// checkTest checks that a benefit of 0 never passes and that a larger benefit
// never fails where a smaller one passed.
func checkTest(t *testing.T, benefit, cost, hurdle int64) {
	t.Helper()
	if saving.Passes(0, cost, hurdle, maxMoney) {
		t.Fatal("a project of benefit 0 passed")
	}
	if benefit < maxMoney && saving.Passes(benefit, cost, hurdle, maxMoney) && !saving.Passes(benefit+1, cost, hurdle, maxMoney) {
		t.Fatal("a larger benefit failed where a smaller one passed")
	}
}
