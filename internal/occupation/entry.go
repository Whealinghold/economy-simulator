package occupation

import "github.com/Whealinghold/economy-sim/internal/checked"

// Host is an agent of the entrant's role in the region that can take
// households: Ref is the caller's index, Weight its weight. The caller lists
// hosts in ascending agent ID order and leaves out an agent that is not
// eligible (the one on a migration journey is MIG's).
type Host struct {
	Ref    int
	Weight int64
}

// Renewal advances the renewal carry of a role: carry + removed * renewalBP is
// split into n whole households (div 10 000) and the new carry (mod 10 000), so
// a rate below 10 000 loses no fraction (spec-spawner.md 7.1, 15.7). ok is false
// on an overflow or a negative argument.
func Renewal(carry, removed, renewalBP int64) (n, carryOut int64, ok bool) {
	if carry < 0 || removed < 0 || renewalBP < 0 {
		return 0, 0, false
	}
	add, ok := checked.Mul(removed, renewalBP)
	if !ok {
		return 0, 0, false
	}
	total, ok := checked.Add(carry, add)
	if !ok {
		return 0, 0, false
	}
	return total / Basis, total % Basis, true
}

// Funded returns min(n, floor(budget / amount)), the entrants the budget buys;
// the others are not created (spec-spawner.md 4.3, 15.5). An amount of 0 or
// below is not limited, and a negative budget buys none at a positive amount.
func Funded(n, budget, amount int64) int64 {
	switch {
	case n <= 0:
		return 0
	case amount <= 0:
		return n
	case budget < 0:
		return 0
	}
	return min(n, budget/amount)
}

// Place chooses the host of an entrant: the agent of the smallest weight below
// scale, a tie going to the earlier host, which is the lower ID (spec-spawner.md
// 4.3). room is the households it can take before reaching scale; the caller
// takes min(remaining, room) and calls again. ok is false when no host is below
// scale, and the entrant then becomes a new agent of weight 1.
func Place(hosts []Host, scale int64) (ref int, room int64, ok bool) {
	best := -1
	for i, h := range hosts {
		if h.Weight < scale && (best < 0 || h.Weight < hosts[best].Weight) {
			best = i
		}
	}
	if best < 0 {
		return 0, 0, false
	}
	return hosts[best].Ref, scale - hosts[best].Weight, true
}

// Retarget carries an agent's target ratio over a role change, with no draw:
// newRole + (target - oldRole) clamped to [lo, hi], so the agent keeps its
// offset from its role's centre (spec-spawner.md 4.2). lo, hi come from
// welfare.TargetRange of the new role through the caller and lo is at most hi.
// ok is false on an overflow.
func Retarget(target, oldRole, newRole, lo, hi int64) (int64, bool) {
	off, ok := checked.Sub(target, oldRole)
	if !ok {
		return 0, false
	}
	v, ok := checked.Add(newRole, off)
	if !ok {
		return 0, false
	}
	return min(max(v, lo), hi), true
}
