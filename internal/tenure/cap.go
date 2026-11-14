package tenure

import "github.com/Whealinghold/economy-sim/internal/checked"

// HolderShare is the share, in basis points, a role holder of weight w gives its
// agent for a row of shareBP: w * shareBP (spec-land.md 6.1.1). ok is false for a
// negative number or a product beyond int64.
func HolderShare(weight, shareBP int64) (int64, bool) {
	if weight < 0 || shareBP < 0 {
		return 0, false
	}
	return checked.Mul(weight, shareBP)
}

// OverCap reports whether a share held by one agent passes the cap
// (spec-land.md 6.1.2, 6.3.4). A cap of BasisPoints or more bounds nothing,
// which is the default. The pool is never checked: its caller does not ask.
func OverCap(shareBP, capBP int64) bool {
	return capBP < BasisPoints && shareBP > capBP
}

// LargestOverCap returns the index of the largest weight whose share
// weight * shareBP passes capBP, the agent that decides the check at
// construction, since the largest agent of a group holds the most (6.1.2). found
// is false when no agent passes the cap. ok is false for a negative number or a
// product beyond int64.
func LargestOverCap(weights []int64, shareBP, capBP int64) (idx int, found, ok bool) {
	for i, w := range weights {
		share, valid := HolderShare(w, shareBP)
		if !valid {
			return 0, false, false
		}
		if OverCap(share, capBP) && (!found || w > weights[idx]) {
			idx, found = i, true
		}
	}
	return idx, found, true
}
