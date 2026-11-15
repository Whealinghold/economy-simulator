package credit

import "math"

// WarnEvery is the run length at which a bounded Warn is logged and the interval
// of the lines after it (spec-observability.md 7.3): the tenth consecutive tick and
// every tenth after. It is a rule constant that changes no result.
const WarnEvery int64 = 10

// NextRun advances a run of consecutive ticks in which a condition held: active
// false resets it to 0, and warn is true at the tenth tick and at every tenth after
// it. The run saturates at the largest int64 and then no longer warns
// (spec-credit.md 16.2).
func NextRun(run int64, active bool) (next int64, warn bool) {
	if !active {
		return 0, false
	}
	if run == math.MaxInt64 {
		return run, false
	}
	run++
	return run, run%WarnEvery == 0
}
