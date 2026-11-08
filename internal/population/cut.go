package population

// Cut applies the per-agent cap and the group floor to the deaths due of one
// agent. due is the whole households the carry produced, fed the agent's
// households that are not unfed, group the households of its (region, role)
// group counting the deaths already taken this tick, and floor MinHouseholds.
// take is what the agent loses, cut the part beyond fed (spec-population.md
// 4.4.3) and held the part the floor refuses (4.4.1); take + cut + held = due.
// The caller subtracts take from group before the next agent. Negative inputs
// count as 0.
func Cut(due, fed, group, floor int64) (take, cut, held int64) {
	due, fed, group, floor = max(due, 0), max(fed, 0), max(group, 0), max(floor, 0)
	take = min(due, fed)
	cut = due - take
	room := max(group-floor, 0)
	held = max(take-room, 0)
	return take - held, cut, held
}

// Allow applies the household cap of a region to the births due: households is
// the region's count after the tick's deaths, and a limit of 0 means no cap
// (spec-population.md 4.4.2). allowed + dropped = due. Negative inputs count as
// 0.
func Allow(due, households, limit int64) (allowed, dropped int64) {
	due, households = max(due, 0), max(households, 0)
	if limit <= 0 {
		return due, 0
	}
	allowed = min(due, max(limit-households, 0))
	return allowed, due - allowed
}
