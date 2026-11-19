package policy

import "github.com/Whealinghold/economy-sim/internal/checked"

// Hold says why a rule returned no entry (spec-policy.md 8.3.1).
type Hold int

const (
	// Moved: the rule returned an entry.
	Moved Hold = iota
	// InBand: the reading is inside the rest band, Target plus or minus Deadband.
	InBand
	// NoReading: the gauge is undefined.
	NoReading
)

// Reaction is the numbers of one reactive rule on one lever (spec-policy.md
// 8.3.1): the reading at which it rests, the half-width of its rest band, the
// error Span that moves the lever by Move basis points, and the direction.
type Reaction struct {
	Target   int64
	Deadband int64
	Span     int64
	Move     int64
	Raise    bool
}

// RuleStep returns the value a rule asks for, given the reading, whether the
// gauge is defined and the lever's current value v (spec-policy.md 8.3.1).
// With e = reading - Target: undefined gives NoReading; |e| <= Deadband gives
// InBand; otherwise d = trunc(e * Move / Span) in 128 bits, negated when Raise
// is false, and the result is v + d. A sum, a difference or a quotient outside
// int64 is cut to the bound of int64 in its sign, so no wrap occurs; the
// checks then cut it to the lever's bounds. A Span below 1 is an input outside
// the domain (validation excludes it) and reads as NoReading.
func RuleStep(r Reaction, reading int64, defined bool, v int64) (next int64, hold Hold) {
	if !defined || r.Span < 1 {
		return v, NoReading
	}
	e := satSub(reading, r.Target)
	if abs(e) <= r.Deadband {
		return v, InBand
	}
	raw, ok := checked.MulDiv(e, r.Move, r.Span)
	if !ok {
		raw = satBound(e < 0 != (r.Move < 0))
	}
	if !r.Raise {
		raw = satNeg(raw)
	}
	return SatAdd(v, raw), Moved
}

// SatAdd returns a + b, cut to the bound of int64 in the sign of the overflow.
func SatAdd(a, b int64) int64 {
	if s, ok := checked.Add(a, b); ok {
		return s
	}
	return satBound(b < 0)
}

func satSub(a, b int64) int64 {
	if d, ok := checked.Sub(a, b); ok {
		return d
	}
	return satBound(b > 0)
}

func satNeg(a int64) int64 {
	if a == -1<<63 {
		return 1<<63 - 1
	}
	return -a
}

// satBound is the lowest int64 when negative and the highest otherwise.
func satBound(negative bool) int64 {
	if negative {
		return -1 << 63
	}
	return 1<<63 - 1
}

// abs returns |a| cut to the highest int64 for the lowest (which has no
// positive counterpart), so a comparison with a deadband never wraps.
func abs(a int64) int64 {
	switch {
	case a >= 0:
		return a
	case a == -1<<63:
		return 1<<63 - 1
	}
	return -a
}
