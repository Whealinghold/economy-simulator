package skill

import "math"

// Outcome is what the entry step of a region does with a trainee
// (spec-skill-barriers.md 4.2.1).
type Outcome int

// The outcomes of Complete.
const (
	// Serving: the ticks served are below TrainingTicks; the course goes on.
	Serving Outcome = iota
	// Done: the trainee completes and changes role.
	Done
	// Held: the ticks served have reached TrainingTicks but the target role has no
	// headroom for the trainee; it stays a trainee and keeps its place.
	Held
)

// Complete returns the outcome for a trainee with served ticks served of ticks
// (spec-skill-barriers.md 4.2.1, FR-SKL-015): Serving below ticks; at ticks, Done
// when the target role has headroom of at least weight or noLimit is true, else
// Held.
func Complete(served, ticks, headroom, weight int64, noLimit bool) Outcome {
	switch {
	case served < ticks:
		return Serving
	case noLimit || headroom >= weight:
		return Done
	}
	return Held
}

// WarnEvery is the period of the warning "no places open": the tenth
// consecutive blocked tick and every tenth after (spec-skill-barriers.md 8.3,
// FR-SKL-022).
const WarnEvery int64 = 10

// NextRun advances the run of consecutive ticks in which a role had applicants
// and no place because the gain is negative. blocked false resets it to 0. warn
// is true at the tenth tick and every tenth after. The run saturates at the
// largest int64 and then no longer warns.
func NextRun(run int64, blocked bool) (run2 int64, warn bool) {
	if !blocked {
		return 0, false
	}
	if run == math.MaxInt64 {
		return run, false
	}
	run++
	return run, run%WarnEvery == 0
}
