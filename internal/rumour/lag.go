package rumour

import "github.com/Whealinghold/economy-sim/internal/checked"

// Lag returns 1 + ceil(distance / speed), the age in ticks of the news of a
// price that travelled distance at speed distance units a tick
// (spec-information.md 5.2.1). The 1 is the tick of publication, so for a
// distance of at least 1 the lag is at least 2 (5.2.2). The ceiling is the
// quotient plus one for a remainder, not (distance + speed - 1) / speed, whose
// sum can overflow when the lag fits. ok is false when distance is below 1 (the
// same region, or no path), speed is below 1, or the lag itself does not fit
// int64.
func Lag(distance, speed int64) (lag int64, ok bool) {
	if distance < 1 || speed < 1 {
		return 0, false
	}
	ceil := distance / speed
	if distance%speed != 0 {
		ceil++
	}
	return checked.Add(1, ceil)
}
