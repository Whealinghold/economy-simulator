package occupation

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// Leaver is an agent that meets the rung-3 trigger, as Exits reads it. Ref is
// the caller's index of the agent, in ascending agent ID order, so that a tie
// in Privation ends in the ID (architecture.md 7.6). Upkeep is one household's
// effective upkeep per tick.
type Leaver struct {
	Ref                                     int
	Privation, Weight, Unfed, Money, Upkeep int64
}

// Exit is one reduction or removal that Exits decided: Households leave the
// agent Ref, which is a removal when they are its whole weight. Insolvent is
// the cause: true for insolvent, false for starved (spec-spawner.md 5.3).
type Exit struct {
	Ref        int
	Households int64
	Remove     bool
	Insolvent  bool
}

// PastCooldown reports whether an agent may act: it took up its role before
// this tick (roleSince < tick, FR-SPN-050) and has held it for at least
// cooldown ticks.
func PastCooldown(tick, roleSince, cooldown int64) bool {
	if roleSince >= tick {
		return false
	}
	d, ok := checked.Sub(tick, roleSince)
	return !ok || d >= cooldown
}

// Rung3 reports whether an agent meets the rung-3 trigger: privation of at
// least removePrivation and removePrivation ticks since it took up its role
// (spec-spawner.md 5.2), which includes roleSince < tick.
func Rung3(privation, tick, roleSince, removePrivation int64) bool {
	return privation >= removePrivation && PastCooldown(tick, roleSince, removePrivation)
}

// ReviewDue reports whether the agent id reviews its role at tick: (tick +
// bucket(id, period)) mod period = 0, with the bucket of ordering.Bucket
// (spec-spawner.md 6.2, 15.2). A period below 1 never reviews.
func ReviewDue(id, tick, period int64) bool {
	if period < 1 {
		return false
	}
	return (tick%period+ordering.Bucket(id, period))%period == 0
}

// Allowance returns ceil(households * shareBP / 10 000), the households that may
// leave (ExitShare) or that bound the changes (EntryShare) in a region or role
// this tick (spec-spawner.md 5.3, 6.3, 15.3). ok is false on an overflow. A
// shareBP of 0 gives 0: the caller treats an ExitShare of 0 as the rung being
// off and does not call Exits.
func Allowance(households, shareBP int64) (int64, bool) {
	return checked.MulDivCeil(households, shareBP, Basis)
}

// Exits decides the reductions and removals of rung 3 (spec-spawner.md 5.3). It
// orders ls by Privation descending then Ref ascending, in place, and for each
// leaver with Unfed above 0 takes drop = min(Unfed, max(1, ceil(Weight *
// deathStepBP / 10 000)), Weight), cut to what is left of the allowance: the
// agent that crosses the allowance is cut, not skipped, and one that finds it
// used up waits. A leaver with Unfed 0 gets no reduction (FR-SPN-059). The cause
// is insolvent when Money < Upkeep * Weight, and also when that product leaves
// int64 (Money is at most MaxMoney). The exits are appended to dst[:0] in that
// order; cut is the households the allowance kept from leaving (ExitCap).
func Exits(dst []Exit, ls []Leaver, allowance, deathStepBP int64) (out []Exit, cut int64) {
	out = dst[:0]
	slices.SortFunc(ls, func(a, b Leaver) int {
		return cmp.Or(cmp.Compare(b.Privation, a.Privation), cmp.Compare(a.Ref, b.Ref))
	})
	left := max(allowance, 0)
	for _, l := range ls {
		if l.Unfed <= 0 || l.Weight <= 0 {
			continue
		}
		want := l.Unfed
		if step, ok := checked.MulDivCeil(l.Weight, deathStepBP, Basis); ok {
			want = min(want, max(1, step))
		}
		want = min(want, l.Weight)
		drop := min(want, left)
		cut += want - drop
		if drop == 0 {
			continue
		}
		left -= drop
		need, ok := checked.Mul(l.Upkeep, l.Weight)
		out = append(out, Exit{
			Ref:        l.Ref,
			Households: drop,
			Remove:     drop == l.Weight,
			Insolvent:  !ok || l.Money < need,
		})
	}
	return out, cut
}

// Remaining returns floor(x * wNew / w), what an agent of weight w keeps of an
// amount x (money, or the units of a good) when its weight falls to wNew; the
// executor sends x less this to the estate (spec-spawner.md 4.1, 15.4). The
// share per remaining household does not fall, since the fed households keep
// what they held. ok is false when w is not above 0, when wNew is outside [0,
// w] or x is negative.
func Remaining(x, w, wNew int64) (int64, bool) {
	if w <= 0 || wNew < 0 || wNew > w || x < 0 {
		return 0, false
	}
	return checked.MulDivFloor(x, wNew, w)
}
