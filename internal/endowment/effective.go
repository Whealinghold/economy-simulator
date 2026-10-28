package endowment

import "github.com/Whealinghold/economy-sim/internal/checked"

// EffectiveCapacity returns C = min(maxCap, trunc(row * scaleBP * (BP + facilityBP) /
// BP^2)), the capacity of an endowment after its capacity scale and the
// capacity layer of its ground facilities (spec-resources.md 3.4). The three
// factors are multiplied in 128 bits. ok is false for a negative input or a
// product beyond 128 bits; a quotient above int64 is cut to maxCap.
func EffectiveCapacity(row, scaleBP, facilityBP, maxCap int64) (c int64, ok bool) {
	if row < 0 || scaleBP < 0 || facilityBP < 0 || maxCap < 0 {
		return 0, false
	}
	x, ok := mul64(uint64(row), uint64(scaleBP)).mul(uint64(BP) + uint64(facilityBP))
	if !ok {
		return 0, false
	}
	q, _ := x.divmod64(BP * BP)
	if q.hi != 0 || q.lo > uint64(maxCap) {
		return maxCap, true
	}
	return int64(q.lo), true
}

// EffectiveScale returns m = trunc(rowBP * scaleBP / BP), the regeneration scale
// of an endowment row times the scale of its shock slot (spec-resources.md 3.4).
// ok is false for a negative input or a result beyond int64.
func EffectiveScale(rowBP, scaleBP int64) (m int64, ok bool) {
	if rowBP < 0 || scaleBP < 0 {
		return 0, false
	}
	return checked.MulDiv(rowBP, scaleBP, BP)
}
