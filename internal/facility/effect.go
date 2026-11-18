package facility

import "github.com/Whealinghold/economy-sim/internal/checked"

// Eff is the gain a route or ground facility gives at its condition,
// floor(boostBP * condition / durability), in basis points
// (spec-infrastructure.md 3.3.2). The product is taken in 128 bits. It rounds
// down, so a facility whose gain has rounded to 0 still gives 0 and stands. ok
// is false when durability is below 1, condition is outside [0, durability] or
// boostBP is negative.
func Eff(boostBP, condition, durability int64) (int64, bool) {
	if durability < 1 || condition < 0 || condition > durability || boostBP < 0 {
		return 0, false
	}
	return checked.MulDivFloor(boostBP, condition, durability)
}

// Cap is the executions of a tick a site serves at its condition,
// ceil(capacity * condition / durability) (spec-infrastructure.md 3.3.2). It
// rounds up, so a site whose condition is above 0 serves at least one
// execution, and it is 0 at condition 0 (a ruined site). ok is false when
// durability is below 1, condition is outside [0, durability] or capacity is
// negative.
func Cap(capacity, condition, durability int64) (int64, bool) {
	if durability < 1 || condition < 0 || condition > durability || capacity < 0 {
		return 0, false
	}
	return checked.MulDivCeil(capacity, condition, durability)
}

// Gain is the sum of the effect fractions of the facilities that apply to one
// pair or one endowment, capped at maxTotal (spec-infrastructure.md 5.2.2,
// 5.3.2: a capped sum, never a product, so ten roads do not multiply). A
// negative term counts as 0 and a negative maxTotal as 0. The sum saturates at
// the cap and cannot overflow.
func Gain(effs []int64, maxTotal int64) int64 {
	maxTotal = max(maxTotal, 0)
	var sum int64
	for _, e := range effs {
		if e <= 0 {
			continue
		}
		if e >= maxTotal-sum {
			return maxTotal
		}
		sum += e
	}
	return sum
}

// Layer is the effective transit factor of a pair, floor(slotBP * (10 000 +
// gainBP) / 10 000) capped at maxFactorBP (spec-infrastructure.md 5.2.2). A
// slot of 0 (a closed edge) stays 0 whatever the gain. The 128-bit product is
// exact and a factor beyond int64 is capped like any other. ok is false only
// when slotBP, gainBP or maxFactorBP is negative.
func Layer(slotBP, gainBP, maxFactorBP int64) (int64, bool) {
	if slotBP < 0 || gainBP < 0 || maxFactorBP < 0 {
		return 0, false
	}
	if slotBP == 0 {
		return 0, true
	}
	f, ok := checked.MulDivFloor(slotBP, Basis+gainBP, Basis)
	if !ok {
		// The factor exceeds int64, so it exceeds any cap too.
		return maxFactorBP, true
	}
	return min(f, maxFactorBP), true
}
