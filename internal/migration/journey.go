package migration

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// weylSpan is 2^32, the span of Weyl32.
const weylSpan = 1 << 32

// wholeBP is 10 000 basis points, a ratio of 1.
const wholeBP = 10_000

// Ticks returns the journey time k = max(1, ceil(distance / speed)) of
// spec-migration.md 6.4: an integer ceiling, and at least one tick. ok is false
// when distance is negative or speed is below 1.
func Ticks(distance, speed int64) (k int64, ok bool) {
	if distance < 0 || speed < 1 {
		return 0, false
	}
	k, ok = checked.MulDivCeil(distance, 1, speed)
	if !ok {
		return 0, false
	}
	return max(1, k), true
}

// Discount returns vhat * (horizon - k) / horizon truncated toward zero
// (spec-migration.md 6.4), the estimate less the share of the horizon spent on
// the journey. ok is false when k is not below horizon (the stage 2 test: the
// candidate is too far), when k is negative or horizon is below 1. The 128-bit
// product cannot overflow the quotient, which is no larger than |vhat|.
func Discount(vhat, k, horizon int64) (veff int64, ok bool) {
	if horizon < 1 || k < 0 || k >= horizon {
		return 0, false
	}
	return checked.MulDiv(vhat, horizon-k, horizon)
}

// Due reports whether the turn of the agent comes at tick: (tick +
// Bucket(id, period)) mod period = 0 (spec-migration.md 5.1). A period below 1
// has no turn. It draws nothing.
func Due(tick, id, period int64) bool {
	if period < 1 {
		return false
	}
	return (tick+ordering.Bucket(id, period))%period == 0
}

// Settled reports whether tick - since >= max(cooldown, window)
// (spec-migration.md 5.2). since is the signed tick of RoleSince. A difference past int64
// is the difference of a distant past, which is settled.
func Settled(tick, since, cooldown, window int64) bool {
	d, ok := checked.Sub(tick, since)
	if !ok {
		return since < 0
	}
	return d >= max(cooldown, window)
}

// Pushed reports whether the earned ratio exists and is below the gate
// (spec-migration.md 5.3): the ratio of an agent with no upkeep is absent and
// that agent is not pushed. A negative ratio is below any non-negative gate.
func Pushed(earnedBP int64, present bool, below int64) bool {
	return present && earnedBP < below
}

// Margin returns the agent's own margin gain + floor(Weyl32(id) * width / 2^32)
// in basis points (spec-migration.md 6.5): at least gain, below gain + width,
// spread over the agents by their IDs with no draw. ok is false on overflow.
func Margin(id, gain, width int64) (g int64, ok bool) {
	spread, ok := checked.MulDivFloor(int64(ordering.Weyl32(id)), width, weylSpan)
	if !ok {
		return 0, false
	}
	return checked.Add(gain, spread)
}

// Clears reports the gain test of stage 4, veff * 10 000 >= home * (10 000 +
// margin) (spec-migration.md 6.5). It is evaluated as veff >= ceil(home * (10
// 000 + margin) / 10 000), which is the same for integers of either sign
// (veff * 10 000 is a multiple of 10 000, so it is at least the product if and
// only if veff is at least the ceiling of the product over 10 000), so no
// signed 128-bit product is formed. ok is false on overflow or a negative margin.
func Clears(veff, home, margin int64) (clears, ok bool) {
	if margin < 0 {
		return false, false
	}
	factor, ok := checked.Add(wholeBP, margin)
	if !ok {
		return false, false
	}
	need, ok := checked.MulDivCeil(home, factor, wholeBP)
	if !ok {
		return false, false
	}
	return veff >= need, true
}
