package skill

// MaxAccumulator is the largest value a teaching accumulator holds between
// ticks, one below a full tick of teaching (spec-skill-barriers.md 4.2.2).
const MaxAccumulator int64 = Basis - 1

// Teach steps a trainer's teaching accumulator by one tick (spec-skill-barriers.md
// 4.2, 4.2.2, FR-SKL-015): it adds sumLoads, the sum of the loads of the
// trainer's trainees in basis points, and when the total reaches 10 000 it
// subtracts 10 000 once and the tick is a teaching tick. A trainer teaches at
// most every tick, so the load beyond one full tick a tick is not carried: the
// result is cut to MaxAccumulator. acc2 is always in [0, MaxAccumulator]. A
// negative acc or sumLoads (outside validation) counts as 0. The loads are capped at
// 20 000 before the sum, so it cannot overflow.
func Teach(acc, sumLoads int64) (acc2 int64, teaching bool) {
	// Loads of 20 000 or more cut the result to MaxAccumulator whatever they are,
	// so they are capped before the sum, which then cannot overflow.
	total := max(acc, 0) + min(max(sumLoads, 0), 2*Basis)
	if total < Basis {
		return total, false
	}
	return min(total-Basis, MaxAccumulator), true
}
