package occupation

// Extinct reports the condition of the "role extinct" warning (FR-SPN-042): the
// role has no agent now and either had at least one (everHad) or has a
// configured group (configured). A role that never had a group and never had an
// agent is not extinct.
func Extinct(agents int64, everHad, configured bool) bool {
	return agents <= 0 && (everHad || configured)
}

// BelowFloor reports the condition of the "group below floor" warning
// (FR-SPN-047): a group of weight above 1 whose agent count is below minAgents.
func BelowFloor(weight, agents, minAgents int64) bool {
	return weight > 1 && agents < minAgents
}

// Unfunded reports the condition of the "entry unfunded" run: entrants were due
// and the pool funded none of them.
func Unfunded(due, funded int64) bool {
	return due > 0 && funded <= 0
}
