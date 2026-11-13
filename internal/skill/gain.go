package skill

import "github.com/Whealinghold/economy-sim/internal/checked"

// Gain returns the trainer's net gain per trainee-tick of spec-skill-barriers.md
// 4.3, output - upkeep - load * wTrainer / 10 000, with the product truncated
// toward zero before the subtraction. ok is false, and g 0, when a step
// overflows int64. The caller takes a trainee only when g is not negative; a
// negative g never ends a traineeship in progress (FR-SKL-016).
func Gain(output, upkeep, load, wTrainer int64) (g int64, ok bool) {
	cost, ok := checked.MulDiv(load, wTrainer, Basis)
	if !ok {
		return 0, false
	}
	net, ok := checked.Sub(output, upkeep)
	if !ok {
		return 0, false
	}
	return checked.Sub(net, cost)
}

// CTrain returns the training cost of an applicant, ticks * surplus, with the
// surplus floored at 0: the observed income per tick above the applicant's own
// upkeep that it gives up while in training (spec-skill-barriers.md 4.5,
// FR-SKL-018). ok is false when the product overflows.
func CTrain(ticks, surplus int64) (c int64, ok bool) {
	if surplus < 0 {
		surplus = 0
	}
	return checked.Mul(ticks, surplus)
}
