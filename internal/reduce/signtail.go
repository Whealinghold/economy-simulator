package reduce

import "math/bits"

// MaxSignN is the largest number of untied pairs the exact sign test takes: a
// cap of choice, so that 2^n fits an int64 (spec-experiments.md 8.3.2).
const MaxSignN = 62

// signScale is the parts per million of a probability.
const signScale = 1_000_000

// SignTailPPM returns the exact two-sided tail of the sign test in parts per
// million, ties already dropped: min(10^6, RDiv(10^6 * 2 * sum_{j >= m} C(n, j),
// 2^n)) with n = wins + losses and m = max(wins, losses), rounded half to even
// (spec-experiments.md 8.4.2). It returns false for n of 0, above MaxSignN, or a
// negative count.
func SignTailPPM(wins, losses int) (int64, bool) {
	if wins < 0 || losses < 0 {
		return 0, false
	}
	n := wins + losses
	if n == 0 || n > MaxSignN {
		return 0, false
	}
	tail := uint64(0) // sum of C(n, j) for j >= m, at most 2^n
	c := uint64(1)    // C(n, j)
	for j := 0; j <= n; j++ {
		if j >= max(wins, losses) {
			tail += c
		}
		hi, lo := bits.Mul64(c, uint64(n-j))
		c, _ = bits.Div64(hi, lo, uint64(j+1)) // exact: C(n, j+1) is an integer, and hi < j+1
	}
	hi, lo := bits.Mul64(2*signScale, tail)
	num, _ := fromMag(u128{hi, lo}, false) // at most 2^83: fits
	q, _ := RDiv(num, FromInt64(int64(1)<<uint(n)))
	v, _ := q.Int64() // at most 2 * 10^6
	return min(v, signScale), true
}
