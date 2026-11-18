package institution

import "github.com/Whealinghold/economy-sim/internal/checked"

// basisPoints is the divisor of a level in basis points (architecture.md 3.5).
const basisPoints = 10_000

// Range returns the legal range in minor units of a price whose effective
// reference price is ref, from a cap and a floor in basis points of it
// (spec-institutions.md 6.2.1, FR-ORG-028):
//
//	cap   = min(maxPrice, max(1, ref * capBP   / 10 000))   if capBP   > 0, else 0 (none)
//	floor = min(maxPrice, max(1, ref * floorBP / 10 000))   if floorBP > 0, else 0 (none)
//
// The division truncates toward zero and the product goes through internal/checked.
// A cap or floor of 0 is none and is returned as 0. ok is false when ref or maxPrice
// is below 1, a level is negative, or a product does not fit int64; the caller then
// has no range for the pair. Equal levels give equal prices and the result is
// monotone in each level, so a floor at most the cap in basis points is at most it in
// minor units.
func Range(ref, capBP, floorBP, maxPrice int64) (capPrice, floorPrice int64, ok bool) {
	if ref < 1 || maxPrice < 1 || capBP < 0 || floorBP < 0 {
		return 0, 0, false
	}
	if capPrice, ok = level(ref, capBP, maxPrice); !ok {
		return 0, 0, false
	}
	if floorPrice, ok = level(ref, floorBP, maxPrice); !ok {
		return 0, 0, false
	}
	return capPrice, floorPrice, true
}

// level is one side of Range: 0 for no level, else the truncated price bounded to
// [1, maxPrice].
func level(ref, bp, maxPrice int64) (int64, bool) {
	if bp == 0 {
		return 0, true
	}
	p, ok := checked.MulDiv(ref, bp, basisPoints)
	if !ok {
		return 0, false
	}
	return min(maxPrice, max(1, p)), true
}
