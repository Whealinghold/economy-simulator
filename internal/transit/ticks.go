package transit

import "github.com/Whealinghold/economy-sim/internal/checked"

// Ticks returns the travel time k = max(1, ceil(distance * 10 000 / (speed *
// factorBP))) of spec-transport.md 4.1. The product distance * 10 000 is held in
// 128 bits and the rounding is up, so a journey never takes less than the
// distance needs. ok is false when distance is negative, speed or factorBP is
// not positive (a factor of 0 is a blocked pair, which the caller tests first),
// or the result does not fit in int64.
func Ticks(distance, speed, factorBP int64) (k int64, ok bool) {
	if distance < 0 || speed <= 0 || factorBP <= 0 {
		return 0, false
	}
	vf, ok := checked.Mul(speed, factorBP)
	if !ok {
		return 0, false
	}
	k, ok = checked.MulDivCeil(distance, 10_000, vf)
	if !ok {
		return 0, false
	}
	return max(1, k), true
}
