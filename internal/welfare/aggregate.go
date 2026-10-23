package welfare

import "github.com/Whealinghold/economy-sim/internal/checked"

// sumParts adds the numerators and the denominators of a set in the slice
// order, saturating (a saturating sum is not associative, so the order is the
// rule; the caller passes ascending agent slot, D63).
func sumParts(nums, dens []int64) (num, den int64) {
	if len(nums) != len(dens) {
		panic("welfare: numerators and denominators differ in length (violated programmer invariant)")
	}
	for i := range nums {
		num, den = satAdd(num, nums[i]), satAdd(den, dens[i])
	}
	return num, den
}

// SetWage is the observed wage of a set of agents (a role in a region, the
// unskilled agents of a region): the summed numerators over the summed
// denominators, one truncating division. It is present whenever the
// denominator is above 0 and may be zero or negative (4.3, FR-WEL-037).
func SetWage(nums, dens []int64) (wage int64, present bool) {
	return quotient(sumParts(nums, dens))
}

// PooledWage is the pooled regional wage over every agent of a region: the
// same sums, known only when the denominator and the quotient are above 0;
// otherwise the caller uses the starting wage (4.3).
func PooledWage(nums, dens []int64) (wage int64, known bool) {
	w, present := SetWage(nums, dens)
	if !present || w <= 0 {
		return 0, false
	}
	return w, true
}

// FloorWage raises wage to floorBP basis points of the starting wage seed,
// truncated toward zero, and returns wage unchanged when the floor is off
// (floorBP 0), there is no starting wage (seed 0 or less) or the wage is already
// above the floor. It is a pure function of its arguments (4.3, the wage the cost
// anchor reads): a price crash lowers the observed wage, and the floor stops
// the anchor, and so the belief floor, from following it down.
func FloorWage(wage, seed, floorBP int64) int64 {
	if seed <= 0 || floorBP <= 0 {
		return wage
	}
	floor, ok := checked.MulDiv(seed, floorBP, bp)
	if !ok { // seed times floorBP cannot leave int64 for floorBP of at most bp
		return wage
	}
	return max(wage, floor)
}

// MeanRatio is the household-weighted mean ratio of the agents whose ratio
// exists: the sum of weight times ratio over the sum of those weights, rounded
// toward zero (FR-WEL-036). households is the sum of those weights; the mean is
// absent when it is not above 0.
func MeanRatio(weights, ratios []int64, present []bool) (meanBP, households int64, ok bool) {
	if len(weights) != len(ratios) || len(weights) != len(present) {
		panic("welfare: MeanRatio slices differ in length (violated programmer invariant)")
	}
	var sum int64
	for i, w := range weights {
		if present[i] {
			sum, households = satAdd(sum, satMul(w, ratios[i])), satAdd(households, w)
		}
	}
	if households <= 0 {
		return 0, households, false
	}
	return sum / households, households, true
}

// Premium is the observed premium of a role in basis points,
// roleWage * 10 000 / unskilledWage in one quotient (4.3). It is absent when
// the role has no wage, the set of unskilled agents has none, or the unskilled
// wage is not positive.
func Premium(roleWage int64, rolePresent bool, unskilledWage int64, unskilledPresent bool) (premiumBP int64, ok bool) {
	if !rolePresent || !unskilledPresent || unskilledWage <= 0 {
		return 0, false
	}
	return satMulDiv(roleWage, bp, unskilledWage), true
}

// HeldByFloor reports whether the work floor holds a free agent to work (D69,
// 11.3): the floor condition of 5.3 holds and the share its ratio alone gives is
// below 10 000, so an absent ratio and a ratio below targetLo are not held. That
// the agent is free is the caller's.
func HeldByFloor(ratioBP int64, present bool, workingTicks, window, minShareBP, targetLo, restBand int64) bool {
	return present && FloorBinds(workingTicks, window, minShareBP) && WorkShare(ratioBP, targetLo, restBand) < bp
}
