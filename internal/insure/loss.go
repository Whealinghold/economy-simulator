package insure

import "github.com/Whealinghold/economy-sim/internal/checked"

// LossValue returns the sum of lost[i] * value[i] (spec-insurance.md 8.2.1,
// FR-INS-041), the value of the units the risk module took, at the unit values of the
// settlement. It saturates at limit and reports saturated true when the sum, or one
// product, passes it (spec-insurance.md 6.3.5): nothing is lost by it, because the
// due is then at most the maximum payout. A negative entry counts as 0. Slices of
// different lengths are read over the shorter.
func LossValue(lost, value []int64, limit int64) (v int64, saturated bool) {
	for i := 0; i < min(len(lost), len(value)); i++ {
		if lost[i] <= 0 || value[i] <= 0 {
			continue
		}
		p, ok := checked.Mul(lost[i], value[i])
		if !ok {
			return limit, true
		}
		if v, ok = checked.Add(v, p); !ok || v > limit {
			return limit, true
		}
	}
	return v, false
}

// Due returns min(maxPayout, halfEven(coverBP * lossValue / 10 000)) and whether the
// payout was cut at maxPayout (spec-insurance.md 8.2.1, 8.2.2, FR-INS-042). A share
// that does not fit int64 is above every maximum payout, so it is capped too. The
// cut is true only when the share is above maxPayout, not when it equals it.
func Due(maxPayout, coverBP, lossValue int64) (due int64, capped bool) {
	if maxPayout <= 0 || coverBP < 0 || lossValue <= 0 {
		return 0, false
	}
	share, ok := checked.MulDivHalfEven(coverBP, lossValue, Basis)
	if !ok || share > maxPayout {
		return maxPayout, true
	}
	return share, false
}
