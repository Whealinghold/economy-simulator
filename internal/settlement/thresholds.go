package settlement

import "github.com/Whealinghold/economy-sim/internal/checked"

// Denominator is the basis-point denominator of the scale and of the band.
const Denominator = 10_000

// Thresholds returns the effective enter thresholds e and the leave thresholds l
// of the classes of one region (spec-settlements.md 4.2.1). enter holds the Enter
// of each class in ascending order, enter[0] being 0; scaleBP is the region's
// ThresholdBP (10 000 for no row) and bandBP the band, 0 to Denominator.
//
//	e[0] = 0                                  l[0] = 0
//	e[k] = ceil(enter[k] * scaleBP / 10 000)  l[k] = floor(e[k] * (10 000 - bandBP) / 10 000)   k >= 1
//
// The enter threshold rounds up, so a region is never given a class earlier than
// the scale says, and the leave threshold rounds down, so for a band of at least
// 1 basis point and e[k] >= 1 the leave threshold is below the enter threshold
// (4.2.2). ok is false for no class, a negative enter or scale, a band outside
// 0 to Denominator, or a product that does not fit int64 (it cannot for validated
// input: the largest is MaxHouseholds * MaxThresholdBP).
func Thresholds(enter []int64, scaleBP, bandBP int64) (e, l []int64, ok bool) {
	if len(enter) == 0 || scaleBP < 1 || bandBP < 0 || bandBP > Denominator {
		return nil, nil, false
	}
	e = make([]int64, len(enter))
	l = make([]int64, len(enter))
	for k := 1; k < len(enter); k++ {
		if enter[k] < 0 {
			return nil, nil, false
		}
		var okE, okL bool
		e[k], okE = checked.MulDivCeil(enter[k], scaleBP, Denominator)
		l[k], okL = checked.MulDivFloor(e[k], Denominator-bandBP, Denominator)
		if !okE || !okL {
			return nil, nil, false
		}
	}
	return e, l, true
}

// Initial returns the class a world is built with (spec-settlements.md 4.5.1):
// the highest k with h >= e[k], never below floor, with no band and no memory.
// A floor outside 0 to len(e)-1 is clamped. e must have its first entry 0, as
// Thresholds returns it; an empty e gives 0.
func Initial(h int64, e []int64, floor int) int {
	c := clampClass(floor, len(e))
	for k := len(e) - 1; k > c; k-- {
		if h >= e[k] {
			return k
		}
	}
	return c
}
