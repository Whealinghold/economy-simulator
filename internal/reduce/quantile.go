package reduce

// Quantile returns the nearest-rank quantile at bp basis points of sorted, which
// is ascending (sorting is the caller's duty): the value at rank
// max(1, ceil(bp * n / 10 000)), clamped to n. There is no interpolation, so the
// result is a value of the input (spec-experiments.md 8.3; the rule of
// spec-indicators.md 5.5.2). An empty slice returns 0; the callers test their
// used count first.
func Quantile(sorted []int64, bp int) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	// bp * n cannot overflow for bp in [0, 10 000] and any slice that fits memory.
	rank := min(max(1, (bp*n+9_999)/10_000), n)
	return sorted[rank-1]
}
