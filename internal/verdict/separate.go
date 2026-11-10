package verdict

import "slices"

// Kind is the kind of a threshold key, which decides how its midpoint is
// rounded (STY 6.5.3).
type Kind uint8

// The kinds of threshold key.
const (
	// Ticks is a count of ticks: rounded up to a multiple of 5.
	Ticks Kind = iota
	// Share is a share or a ratio: rounded up to two significant digits.
	Share
)

// roundTicksUp returns x rounded up to a multiple of 5, and false on overflow.
func roundTicksUp(x int64) (int64, bool) {
	r := x % 5
	if r < 0 {
		r += 5
	}
	if r == 0 {
		return x, true
	}
	if x > maxInt64-(5-r) {
		return 0, false
	}
	return x + 5 - r, true
}

// Separate returns the value of the separated key of a detector from the
// statistic of each healthy run and of each provoked run (STY 6.5.3,
// FR-STY-071, FR-STY-074): t_hi is the largest healthy value and t_lo the
// smallest provoked one; the key is the integer midpoint of the two rounded up,
// then up to a multiple of 5 for Ticks or to two significant digits for Share,
// kept within [t_hi+1, t_lo]; when t_lo = t_hi+1 it is t_lo. A detector whose
// bad side is low (R4, R7) is stated with lowIsBad true: the statistics are
// negated first and the key negated back, so that badness at or above the
// threshold is what "separates" means. If t_hi >= t_lo the result is
// ErrNoSeparation and the key stays unset; with no healthy or no provoked
// run it is ErrNoRuns, because no threshold is set from one set alone.
func Separate(healthy, provoked []int64, kind Kind, lowIsBad bool) (int64, error) {
	if len(healthy) == 0 || len(provoked) == 0 {
		return 0, ErrNoRuns
	}
	hs, ps := healthy, provoked
	if lowIsBad {
		var ok bool
		if hs, ok = negated(healthy); !ok {
			return 0, ErrOverflow
		}
		if ps, ok = negated(provoked); !ok {
			return 0, ErrOverflow
		}
	}
	tHi, tLo := slices.Max(hs), slices.Min(ps)
	if tHi >= tLo {
		return 0, ErrNoSeparation
	}
	k, ok := roundMid(midUp(tHi, tLo), kind)
	if !ok || k > tLo {
		k = tLo // the window is [tHi+1, tLo]; a midpoint at or above tLo is tLo
	}
	if lowIsBad {
		return -k, nil
	}
	return k, nil
}

// negated returns the values negated, and false if one is the minimum int64.
func negated(v []int64) ([]int64, bool) {
	out := make([]int64, len(v))
	for i, x := range v {
		if x == minInt64 {
			return nil, false
		}
		out[i] = -x
	}
	return out, true
}

func roundMid(mid int64, kind Kind) (int64, bool) {
	if kind == Ticks {
		return roundTicksUp(mid)
	}
	return roundOutward(mid, true)
}
