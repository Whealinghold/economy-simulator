package measure

// IsqrtNearest returns the integer nearest to the square root of n
// (spec-indicators.md 5.3.4): r0 = floor(sqrt(n)), and r0 + 1 when n - r0*r0 is
// above r0. There is no tie to break, because (r0 + 1/2)^2 is never an integer.
func IsqrtNearest(n uint64) uint64 {
	lo, hi := uint64(0), uint64(1<<32-1)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if mid*mid <= n {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if n-lo*lo > lo {
		return lo + 1
	}
	return lo
}
