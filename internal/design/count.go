// Package design holds the pure functions behind the experiment harness's
// designs (spec-experiments.md 3.2, 3.3.4, 4): the grid and one-at-a-time point
// orders, the counter-based Latin hypercube, the replica seeds, the overlay text
// of a change and the path syntax. It works on primitive ints and strings and
// knows nothing of a configuration, a world or a measure (architecture.md 2.9).
// It holds no variable, draws nothing and keeps no state: each result is a
// function of its arguments alone (FR-EXP-009).
package design

import "math/bits"

// CheckedCount returns the product of factors, and false when a factor is
// negative or the product does not fit an int. The product of no factors is 1,
// the single point of a factor-free design (spec-experiments.md 4.1.3).
func CheckedCount(factors ...int) (int, bool) {
	p := uint64(1)
	for _, f := range factors {
		// Tested apart from the range test below: after a zero factor p is 0 and a
		// negative f would otherwise multiply to 0 and pass.
		if f < 0 {
			return 0, false
		}
		hi, lo := bits.Mul64(p, uint64(f))
		if hi != 0 || lo > maxInt {
			return 0, false
		}
		p = lo
	}
	return int(p), true
}

const maxInt = uint64(^uint(0) >> 1)
