package saving

// ConditionBP returns condition * 10 000 / durability, the share of its
// durability a facility has left, in basis points (spec 5.8.1). A durability
// below 1 gives 0.
func ConditionBP(condition, durability int64) int64 {
	if durability < 1 {
		return 0
	}
	return satMulDiv(nonneg(condition), scale, durability)
}

// RepairDue is the gate of spec 5.8.1 (FR-SAV-022): the lowest share of
// durability among the facilities of the class is at or below belowBP. A
// belowBP of 0 never repairs, whatever the condition.
func RepairDue(lowestBP, belowBP int64) bool {
	return belowBP > 0 && lowestBP <= belowBP
}

// MPlan returns ceil(missing / units): the executions of a repair recipe that
// restore what is missing, planned ignoring stock (spec 5.8.1). A units below 1
// is taken as 1.
func MPlan(missing, units int64) int64 {
	if units < 1 {
		units = 1
	}
	return satMulDivCeil(nonneg(missing), 1, units)
}

// Restored returns min(missing, mPlan * units) (spec 5.8.1).
func Restored(missing, mPlan, units int64) int64 {
	return min(nonneg(missing), satMul(nonneg(mPlan), nonneg(units)))
}

// RepairTime returns T_rep = min(floor(restored / rate), payback * nFac): the
// ticks of service a repair buys (spec 5.8.1). A rate of 0 gives the bound.
func RepairTime(restored, rate, payback, nFac int64) int64 {
	bound := satMul(nonneg(payback), nonneg(nFac))
	if rate <= 0 {
		return bound
	}
	return min(nonneg(restored)/rate, bound)
}
