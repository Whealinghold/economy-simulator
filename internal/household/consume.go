package household

import "github.com/Whealinghold/economy-sim/internal/checked"

// Fed returns the number of whole households that received their full basket:
// the least over the upkeep rows of floor(consumed[i] / perHousehold[i]),
// capped at weight (spec-agents.md 5.4, example 16.3). It rounds down, never up
// in favour of the household. With no rows it returns weight (an agent with no
// upkeep is never unfed, FR-AGT-014). The slices are parallel and only the
// shorter prefix is read. A row with a per-household quantity of 0 or less is
// skipped: validation forbids it, and it asks for nothing. A negative consumed
// counts as 0. A weight below 0 gives 0.
func Fed(consumed, perHousehold []int64, weight int64) int64 {
	fed := max(weight, 0)
	for i := 0; i < len(consumed) && i < len(perHousehold); i++ {
		if perHousehold[i] <= 0 {
			continue
		}
		fed = min(fed, max(consumed[i], 0)/perHousehold[i])
	}
	return fed
}

// NextHunger returns Hunger after a tick: before + 1 if unfed is above 0, else 0
// (spec-agents.md 5.4.1). It saturates at the maximum of int64 instead of
// wrapping; a counter that rises one a tick cannot reach it in a run, so the
// saturation is a guard and not a rule.
func NextHunger(before, unfed int64) int64 {
	if unfed <= 0 {
		return 0
	}
	return satInc(before)
}

// NextPrivation returns Privation after a tick: before + 1 if unfed is above 0,
// else max(0, before - 1) (spec-agents.md 5.4.1, the leaky count the failure
// ladder reads). It is the one implementation of the step. It saturates at the
// maximum of int64, as NextHunger does.
func NextPrivation(before, unfed int64) int64 {
	if unfed > 0 {
		return satInc(before)
	}
	if before <= 0 {
		return 0
	}
	return before - 1
}

func satInc(v int64) int64 {
	if v == 1<<63-1 {
		return v
	}
	return v + 1
}

// Cost accumulates the upkeep cost of one tick, the sum over the upkeep
// commodities of U_c * Centre_c (spec-agents.md 7.4.2). The zero value is an
// empty sum. Once an operation overflows, ok stays false.
type Cost struct {
	sum int64
	bad bool
}

// Add adds total * centre, total being the units the agent needs this tick and
// centre its belief's centre for the commodity, both with checked arithmetic.
func (c *Cost) Add(total, centre int64) {
	if c.bad {
		return
	}
	p, ok := checked.Mul(total, centre)
	if ok {
		c.sum, ok = checked.Add(c.sum, p)
	}
	c.bad = !ok
}

// Value returns the sum and false if any Add overflowed.
func (c *Cost) Value() (int64, bool) {
	if c.bad {
		return 0, false
	}
	return c.sum, true
}

// ReserveMoney returns reserveTicks * upkeepCost, the flat reserve of
// spec-agents.md 7.4.2, and false on overflow.
func ReserveMoney(upkeepCost, reserveTicks int64) (int64, bool) {
	return checked.Mul(reserveTicks, upkeepCost)
}
