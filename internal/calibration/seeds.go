package calibration

import "github.com/Whealinghold/economy-sim/internal/checked"

// The seed sets of spec-calibration.md 7.1. Every number here is a placeholder
// (P) of that section except the stylised range of 1 to 60, which is REF 8.2's.
const (
	// blockSize is the number of seeds of a search block (7.1.1).
	blockSize = 100
	// confirmSeeds and confirmSeedsL are the last confirm seed: 1 to 50, and 1 to 10
	// at tier L (7.1.3), STY's own setting range.
	confirmSeeds  = 50
	confirmSeedsL = 10
	// stylisedLast is the last seed of the stylised seeds 1 to 60 (the confirm
	// seeds 1 to 50 and the hold-out 51 to 60, REF 8.2): no search seed is in it.
	stylisedLast = 60
	// searchReplicas is the replicas of a point in a search (7.1.3): a band is
	// never set from fewer than 10 seeds (FR-STY-078).
	searchReplicas = 10
	// passDenominator is the 10 of the pass threshold R - floor(R/10) (7.1.2).
	passDenominator = 10
)

// ErrBlock is returned for a search block below 1 or one whose seeds leave int64.
const ErrBlock = calError("calibration: search block below 1 or too large")

// Block returns the first and last seed of search block b, 100b+1 to 100b+100
// for b at least 1 (FR-CLB-020). The block is disjoint from the seeds 1 to 60.
func Block(b int) (first, last uint64, err error) {
	if b < 1 {
		return 0, 0, ErrBlock
	}
	base, ok := checked.Mul(int64(b), blockSize)
	if !ok {
		return 0, 0, ErrBlock
	}
	end, ok := checked.Add(base, blockSize)
	if !ok {
		return 0, 0, ErrBlock
	}
	return uint64(base) + 1, uint64(end), nil
}

// ConfirmSeeds returns the first and last seed of a confirm: 1 to 50, and 1 to
// 10 at tier L (FR-CLB-021).
func ConfirmSeeds(tierL bool) (first, last uint64) {
	if tierL {
		return 1, confirmSeedsL
	}
	return 1, confirmSeeds
}

// Threshold returns the number of the R seeds on which a B criterion must pass,
// R - floor(R/10), which is ceil(9R/10) in exact integer arithmetic (FR-CLB-106,
// FR-CLB-013): 23 of 25, where half to even would give 22. It is 0 for R below 1.
func Threshold(r int) int {
	if r < 1 {
		return 0
	}
	return r - r/passDenominator
}

// InSTYRange reports whether a seed is one of the stylised seeds 1 to 60, which
// no search uses (FR-CLB-020, FR-CLB-021).
func InSTYRange(seed uint64) bool {
	return seed >= 1 && seed <= stylisedLast
}
