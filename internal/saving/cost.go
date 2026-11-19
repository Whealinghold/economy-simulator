package saving

// Inputs returns m * sum(qty[i] * centre[i]): the recipe's inputs and the wear
// of its tools at the agent's centres for m executions (spec 5.4). The slices
// are read to the shorter length; a negative term counts as 0; the sum
// saturates at int64.
func Inputs(m int64, qty, centre []int64) int64 {
	n := min(len(qty), len(centre))
	var sum int64
	for i := 0; i < n; i++ {
		sum = satAdd(sum, satMul(nonneg(qty[i]), nonneg(centre[i])))
	}
	return satMul(nonneg(m), sum)
}

// Opportunity returns jNow * duration / 10 000: what the best job would earn in
// the ticks of the project (spec 5.4). jNow is J(c, cap_now).
func Opportunity(jNow, duration int64) int64 {
	return satMulDiv(nonneg(jNow), nonneg(duration), scale)
}

// Cost returns inputs + opportunity saturated at maxMoney (spec 5.4,
// FR-SAV-013 and 014). Upkeep is never a term of the cost: it is in the outlay.
func Cost(inputs, opportunity, maxMoney int64) int64 {
	return capAt(satAdd(nonneg(inputs), nonneg(opportunity)), maxMoney)
}
