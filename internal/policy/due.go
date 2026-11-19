package policy

import "github.com/Whealinghold/economy-sim/internal/ordering"

// Due reports whether the actor with index actor is due at tick, for an actor
// that decides every every ticks: (tick + bucket(actor, every)) mod every = 0,
// with bucket the rotation of internal/ordering (spec-policy.md 7.2.1,
// FR-PLC-051). An actor of index 0 is due at every multiple of every, tick 0
// included. An every below 1 has no rotation and is never due. The sum is taken
// on tick mod every, so no tick, however near the end of int64, wraps it.
func Due(tick, actor, every int64) bool {
	if every < 1 {
		return false
	}
	return (tick%every+ordering.Bucket(actor, every))%every == 0
}
