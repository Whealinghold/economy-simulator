package verdict

import (
	"slices"
	"strconv"
)

// SeedResult is the outcome of one of the seed rules of STY 8.3.
type SeedResult struct {
	// Rule is "SR1", "SR2" or "SR3".
	Rule string
	// Pass is the verdict of the rule.
	Pass bool
	// N is the number of seeds planned, not the number completed (8.3.3).
	N int
	// Need is the number of seeds that must be inside (SR1, SR2).
	Need int
	// Outside lists the seeds outside the band, ascending and without repeats.
	// A failed world is an outside seed (8.3.3).
	Outside []int64
	// Min, Median and Max are recorded for SR3 (HasStats is false when no
	// seed produced a value).
	Min, Median, Max int64
	HasStats         bool
}

// String renders the result as one line of integers and words for a failure
// message and the verdict record (FR-STY-086).
func (r SeedResult) String() string {
	s := r.Rule + " "
	if r.Pass {
		s += "pass"
	} else {
		s += "fail"
	}
	s += " n " + strconv.Itoa(r.N) + " need " + strconv.Itoa(r.Need) + " outside"
	for _, o := range r.Outside {
		s += " " + strconv.FormatInt(o, 10)
	}
	if r.HasStats {
		s += " min " + strconv.FormatInt(r.Min, 10) + " median " + strconv.FormatInt(r.Median, 10) + " max " + strconv.FormatInt(r.Max, 10)
	}
	return s
}

// outsideList returns the outside seeds ascending and without repeats; the
// argument is not changed, so the order the caller gives is irrelevant.
func outsideList(outside []int64) []int64 {
	out := slices.Clone(outside)
	slices.Sort(out)
	return slices.Compact(out)
}

// ceil9n returns ceil(9n/10) as n - floor(n/10), which cannot overflow.
func ceil9n(n int) int { return n - n/10 }

// SR1 passes only when every one of the n planned seeds is inside: no seed is
// outside (STY 8.3, FR-STY-076). A plan of fewer than one seed never passes.
func SR1(n int, outside []int64) SeedResult {
	out := outsideList(outside)
	return SeedResult{Rule: "SR1", Pass: n >= 1 && len(out) == 0, N: n, Need: max(n, 0), Outside: out}
}

// SR2 passes when at least ceil(9n/10) of the n planned seeds are inside; the
// seeds outside are listed, not forgiven (8.3.2). A plan of fewer than one
// seed never passes.
func SR2(n int, outside []int64) SeedResult {
	out := outsideList(outside)
	need := ceil9n(max(n, 0))
	return SeedResult{Rule: "SR2", Pass: n >= 1 && n-len(out) >= need, N: n, Need: need, Outside: out}
}

// SR3 passes when the median of the values lies in [lo, hi] and every one of
// the n planned seeds produced a value: a failed world is a seed outside and
// the run does not pass by having fewer values to judge (8.3.3). The median of
// an even count is the half-to-even mean of the middle two (8.3.4). The
// minimum and maximum are recorded; Outside stays empty, because the seeds
// are not known here and the caller lists them against the band.
func SR3(n int, values []int64, lo, hi int64) SeedResult {
	r := SeedResult{Rule: "SR3", N: n, Need: max(n, 0)}
	if len(values) == 0 {
		return r
	}
	s := slices.Clone(values)
	slices.Sort(s)
	r.Min, r.Max, r.HasStats = s[0], s[len(s)-1], true
	if len(s)%2 == 1 {
		r.Median = s[len(s)/2]
	} else {
		r.Median = meanHalfEven(s[len(s)/2-1], s[len(s)/2])
	}
	r.Pass = n >= 1 && len(s) >= n && r.Median >= lo && r.Median <= hi
	return r
}
