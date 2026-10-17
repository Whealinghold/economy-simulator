package welfare

// WearCost returns the wear cost of a row of executions started in a tick,
// executions * ceil(wearPerUse * price / durability): one ceiling per
// execution, then the product, never one ceiling of the product (3 * 10 / 4
// rounds to 8, so 2 executions cost 16 and not ceil(60 / 4) = 15). The weight
// does not enter: an execution is one household's (spec-welfare-and-wages.md
// 2.1.4, D66). It saturates and returns no ok. A zero durability is refused by
// validation, so here it is a violated programmer invariant.
func WearCost(wearPerUse, price, durability, executions int64) int64 {
	if durability == 0 {
		panic("welfare: WearCost with durability 0 (violated programmer invariant)")
	}
	return satMul(executions, satMulDivCeil(wearPerUse, price, durability))
}

// TickCost returns the cost of one tick, the sum over the upkeep rows in slice
// order of quantity * price * weight (2.1). The slices are equal in length.
func TickCost(quantities, prices []int64, weight int64) int64 {
	if len(quantities) != len(prices) {
		panic("welfare: TickCost with slices of different length (violated programmer invariant)")
	}
	var sum int64
	for i, q := range quantities {
		sum = satAdd(sum, satMul(satMul(q, prices[i]), weight))
	}
	return sum
}

// TickIncome returns the income of one tick, the sum of the credits less the
// sum of the debits, each summed in slice order (2.1).
func TickIncome(credits, debits []int64) int64 {
	var c, d int64
	for _, v := range credits {
		c = satAdd(c, v)
	}
	for _, v := range debits {
		d = satAdd(d, v)
	}
	return satSub(c, d)
}
