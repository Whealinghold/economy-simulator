package ownuse

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// BP is the denominator of a basis point.
const BP = 10_000

// Option is an own-use recipe a household may request: Ref is the caller's
// opaque reference (the recipe ID, which gives the tie-break), Max the
// executions the household could start (MaxExecutions of the view) and Cov the
// coverage of each covered good by the recipe, in the order of the covered
// set, 0 for a good the recipe does not make.
type Option struct {
	Ref int
	Max int64
	Cov []int64
}

// Cov returns the coverage of a commodity by a recipe, q * E / (Duration * u)
// in basis points, truncated, with 128-bit intermediates (spec-subsistence.md
// 4.1, FR-SUB-039): q the units per execution, factor the effective factor E in
// basis points (the abundance included), duration the recipe's ticks and
// upkeep the household's upkeep u of the commodity. ok is false for a
// negative q or factor, a duration or upkeep below 1, or a result beyond
// int64; the value is then math.MaxInt64 for a result that is too large, which
// is the saturated coverage, and 0 otherwise.
func Cov(q, factor, duration, upkeep int64) (cov int64, ok bool) {
	if q < 0 || factor < 0 || duration < 1 || upkeep < 1 {
		return 0, false
	}
	cov, ok = checked.MulDivDiv(q, factor, duration, upkeep)
	if !ok {
		return math.MaxInt64, false
	}
	return cov, true
}

// FloorBP returns the floor figure of a recipe in a region, the greatest of
// its coverages over the covered goods (spec-subsistence.md 4.2, FR-SUB-039),
// and 0 for none.
func FloorBP(covs []int64) int64 {
	var m int64
	for _, c := range covs {
		m = max(m, c)
	}
	return m
}

// Pick returns the reference of the own-use recipe to request: among the
// options with Max of at least 1 and a positive sum of coverage over the goods
// whose cover is short (short[i] is true), the one with the greatest sum, a tie
// going to the lower Ref (spec-subsistence.md 5.5 item 3, FR-SUB-020). The sum
// is of the truncated coverages, compared as integers, and saturates at
// math.MaxInt64. ok is false when no option qualifies, so a recipe with a sum
// of 0 is never returned.
func Pick(opts []Option, short []bool) (ref int, ok bool) {
	var best int64
	for _, o := range opts {
		if o.Max < 1 {
			continue
		}
		sum := shortSum(o.Cov, short)
		if sum > best || (sum == best && sum > 0 && o.Ref < ref) {
			best, ref, ok = sum, o.Ref, true
		}
	}
	return ref, ok
}

// shortSum is the sum of the coverages of the goods that are short, saturating
// at math.MaxInt64.
func shortSum(covs []int64, short []bool) int64 {
	var sum int64
	for i, c := range covs {
		if i >= len(short) || !short[i] {
			continue
		}
		s, ok := checked.Add(sum, c)
		if !ok {
			return math.MaxInt64
		}
		sum = s
	}
	return sum
}

// Ramp returns the executions an agent of weight w requests in the mode,
// min(w, maxExec, max(1, ceil(w * (exit - cover) / exit))) (spec-subsistence.md
// 5.5 item 4, FR-SUB-021): a weight-1 agent runs 1 and a cohort puts in the
// share of its households that is short of cover. The caller passes maxExec of
// at least 1. ok is false for a weight or an exit below 1.
func Ramp(w, maxExec, cover, exit int64) (k int64, ok bool) {
	if w < 1 || exit < 1 {
		return 0, false
	}
	short := max(exit-cover, 0)
	r, ok := checked.MulDivCeil(w, short, exit)
	if !ok {
		return 0, false
	}
	return min(w, maxExec, max(1, r)), true
}

// RetryDue reports whether the default Work should prefer a role recipe to the
// own-use recipe: the retry is on (retryJobs above 0) and the household has
// started that many own-use jobs in a row (spec-subsistence.md 5.5 item 5,
// FR-SUB-022).
func RetryDue(retryJobs, jobs int64) bool {
	return retryJobs > 0 && jobs >= retryJobs
}
