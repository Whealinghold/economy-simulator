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

// Output returns the net value of a trainee's eligible output per tick
// (spec-skill-barriers.md 4.3, spec-production.md 8.3): the value of a full
// execution's outputs times the trainee's share yieldBP over 10 000, less the
// value of its inputs, over the duration of the recipe in ticks, each quotient
// truncated toward zero. ok is false when the duration is not above 0, an input
// is negative, or a step overflows.
func Output(value, inputCost, yieldBP, duration int64) (o int64, ok bool) {
	if duration <= 0 || value < 0 || inputCost < 0 || yieldBP < 0 {
		return 0, false
	}
	got, ok := checked.MulDiv(value, yieldBP, Basis)
	if !ok {
		return 0, false
	}
	net, ok := checked.Sub(got, inputCost)
	if !ok {
		return 0, false
	}
	return net / duration, true
}

// Surplus returns S_u, the observed income per tick above the agent's own
// upkeep, floored at 0 (spec-skill-barriers.md 4.5): the ratio of the welfare
// window is income over cost in basis points, so the surplus is (ratioBP -
// 10 000) * upkeep / 10 000 with the quotient truncated toward zero. It is 0
// when the window has no ratio yet (present false) or the ratio is at most
// 10 000. ok is false on overflow.
func Surplus(ratioBP int64, present bool, upkeep int64) (s int64, ok bool) {
	if !present || ratioBP <= Basis || upkeep <= 0 {
		return 0, true
	}
	return checked.MulDiv(ratioBP-Basis, upkeep, Basis)
}
