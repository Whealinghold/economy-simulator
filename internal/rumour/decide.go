package rumour

// Outcome is the result of deciding one candidate (spec-information.md 5.3.1).
// The zero value is NoPath, the outcome that offers nothing.
type Outcome uint8

// The outcomes, in the order of the cases of spec-information.md 5.3.1.
const (
	// NoPath means no walk over open edges leads from the source to the reader.
	NoPath Outcome = iota
	// TooOld means the lag is above the horizon RumourMaxAge.
	TooOld
	// NotYet means the vintage tick would be before tick 0.
	NotYet
	// Offered means the candidate gets a rumour.
	Offered
)

// Decide returns the first case of spec-information.md 5.3.1 that applies to a
// candidate: NoPath when hasPath is false, TooOld when lag is above maxAge,
// NotYet when lag is above asOf (the tick of the reading, or the next tick for
// the migration pass, 4.3.2), else Offered. The cases are tested in that order,
// so a pair with no path and a lag above the horizon is NoPath. lag is not read
// when hasPath is false.
func Decide(hasPath bool, lag, maxAge, asOf int64) Outcome {
	switch {
	case !hasPath:
		return NoPath
	case lag > maxAge:
		return TooOld
	case lag > asOf:
		return NotYet
	default:
		return Offered
	}
}
