package saving

// ToolBuy returns sum(max(0, m - stock[i]) * centre[i]): the price at the
// centres of the tools the household does not hold (spec 5.6.1).
func ToolBuy(m int64, stock, centre []int64) int64 {
	n := min(len(stock), len(centre))
	var sum int64
	for i := 0; i < n; i++ {
		sum = satAdd(sum, satMul(nonneg(satSub(m, nonneg(stock[i]))), nonneg(centre[i])))
	}
	return sum
}

// HeldValue returns sum(min(stock[i], qty[i] * m) * centre[i]): the value at
// the centres of the inputs, or of the tools with a quantity of 1, that the
// household already holds, up to what m executions need (spec 5.5.3, 5.6.1).
func HeldValue(m int64, qty, stock, centre []int64) int64 {
	n := min(len(qty), len(stock), len(centre))
	var sum int64
	for i := 0; i < n; i++ {
		held := min(nonneg(stock[i]), satMul(nonneg(qty[i]), nonneg(m)))
		sum = satAdd(sum, satMul(held, nonneg(centre[i])))
	}
	return sum
}

// Outlay returns inputs + toolBuy + upkeep * (duration + 1 + safety): the money
// the household needs in hand to buy the project and eat through it
// (spec 5.6.1, FR-SAV-019).
func Outlay(inputs, toolBuy, upkeep, duration, safety int64) int64 {
	ticks := satAdd(satAdd(nonneg(duration), 1), nonneg(safety))
	return satAdd(satAdd(nonneg(inputs), nonneg(toolBuy)), satMul(nonneg(upkeep), ticks))
}

// Goal returns max(0, outlay - held - tier2): the money still to set aside for
// the project in this tick (spec 5.6.3, FR-SAV-019). tier2 is the plan's own
// tier 2 spend on the project's inputs and tools in the tick, 0 when the goal
// is read from saved state.
func Goal(outlay, held, tier2 int64) int64 {
	return nonneg(satSub(satSub(outlay, nonneg(held)), nonneg(tier2)))
}

// Keep returns upkeep * (remaining + 1 + safety): the goal while a job of a
// project recipe is in progress (spec 5.5.6, FR-SAV-046).
func Keep(upkeep, remaining, safety int64) int64 {
	return satMul(nonneg(upkeep), satAdd(satAdd(nonneg(remaining), 1), nonneg(safety)))
}

// Funded reports money + held >= outlay + floor (spec 5.6.2).
func Funded(money, held, outlay, floor int64) bool {
	return satAdd(money, nonneg(held)) >= satAdd(outlay, nonneg(floor))
}
