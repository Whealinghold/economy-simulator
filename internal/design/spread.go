package design

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// MaxSpread is the largest spread the functions of this file take: 2^62 points,
// so that 2n, 2*cell+1 and 2^(2w) stay inside 64 bits. The harness bound
// (MaxPoints, spec-experiments.md 13.2) is far below it.
const MaxSpread = 1 << 62

// The rule constants of the spread (spec-experiments.md 4.4.2, architecture.md
// 9.6). Changing either moves every spread and the worked points (FR-EXP-008).
const (
	// designSalt is xored into the experiment's DesignSalt. It is the design's
	// own constant and not a reseed constant (spec-checkpoint.md 6.8.2).
	designSalt uint64 = 0x243F6A8885A308D3
	// gamma is the 64-bit golden ratio increment, the same value as the
	// SplitMix64 increment but kept here, so that a change to one does not move
	// the other.
	gamma uint64 = 0x9E3779B97F4A7C15
	// rounds is the number of Feistel rounds.
	rounds = 4
)

// SpreadCell returns the cell of point i on axis axis of a spread of n points:
// P_k(i) with k = fin(fin(S + (axis+1)*G) + n) and S = salt xor designSalt, where
// P_k is a four-round balanced Feistel permutation over 2w bits, walked back into
// [0, n) (spec-experiments.md 4.4.2). It is a permutation of [0, n) for every
// n >= 1, and a pure function of (salt, axis, n, i): point i is the same whatever
// other points exist (FR-EXP-007, FR-EXP-009). It returns false for n outside [1, MaxSpread], i
// outside [0, n) or a negative axis.
func SpreadCell(salt uint64, axis, n, i int) (int, bool) {
	if n < 1 || n > MaxSpread || i < 0 || i >= n || axis < 0 {
		return 0, false
	}
	if n == 1 {
		return 0, true
	}
	s := salt ^ designSalt
	key := ordering.Finalise(ordering.Finalise(s+uint64(axis+1)*gamma) + uint64(n))
	w := halfBits(n)
	x := uint64(i)
	for {
		x = feistel(x, w, key)
		if x < uint64(n) {
			return int(x), true
		}
	}
}

// halfBits returns the smallest w >= 1 with 2^(2w) >= n.
func halfBits(n int) uint {
	w := uint(1)
	for uint64(1)<<(2*w) < uint64(n) {
		w++
	}
	return w
}

// feistel is one pass of the network over 2w bits: (l, r) -> (r, l xor
// (fin(key + t*G + r) mod 2^w)) for t = 1 .. 4. All sums and products are modulo
// 2^64.
func feistel(x uint64, w uint, key uint64) uint64 {
	mask := uint64(1)<<w - 1
	l, r := x>>w, x&mask
	for t := uint64(1); t <= rounds; t++ {
		l, r = r, l^(ordering.Finalise(key+t*gamma+r)&mask)
	}
	return l<<w | r
}

// SpreadLevel returns the level of an explicit factor of levels levels used by a
// point whose cell is cell in a spread of n: floor((2*cell+1)*levels / (2*n))
// (spec-experiments.md 4.4.3). It returns false for arguments out of range
// (n or levels below 1, cell outside [0, n)).
func SpreadLevel(cell, n, levels int) (int, bool) {
	if n < 1 || n > MaxSpread || levels < 1 || cell < 0 || cell >= n {
		return 0, false
	}
	hi, lo := bits.Mul64(uint64(2*cell+1), uint64(levels))
	q, _ := bits.Div64(hi, lo, uint64(2*n)) // 2*cell+1 < 2*n, so hi < 2*n
	return int(q), true
}

// SpreadValue returns the value of a range factor [lo, hi] used by a point whose
// cell is cell in a spread of n: lo + floor((2*cell+1)*(hi-lo+1) / (2*n)), the
// midpoint of its cell in 128-bit arithmetic with no rounding but the floor
// (spec-experiments.md 4.4.3). hi-lo+1 may be 2^64, which a 64-bit product would
// wrap. It returns false for n below 1, cell outside [0, n) or lo above hi.
func SpreadValue(cell, n int, lo, hi int64) (int64, bool) {
	if n < 1 || n > MaxSpread || cell < 0 || cell >= n || lo > hi {
		return 0, false
	}
	width := uint64(hi) - uint64(lo) // hi - lo, exact in uint64
	odd := uint64(2*cell + 1)
	ph, pl := bits.Mul64(odd, width)
	pl, c := bits.Add64(pl, odd, 0) // odd * (width + 1)
	ph += c
	q, _ := bits.Div64(ph, pl, uint64(2*n)) // odd < 2*n, so the quotient is below width + 1
	return int64(uint64(lo) + q), true
}
