package economysim

import "cmp"

// This file is the derivation of the agents of a world from its census
// (spec-scale.md 3.2, 13): how many agents a group has and the weight of each,
// the plan of every group, and the invariant that the census adds up. It is the
// one home of the formula; validation and construction both call it (D281).

// agentCount returns the number of agents of a group of households households
// at per-agent scale scale and floor minAgents (spec-scale.md 3.2, FR-SCL-002):
// ceil(households / scale), raised to min(minAgents, households) when below the
// floor. It returns 0 for a group with no household or a scale below 1, which
// validation rejects.
func agentCount(households, scale, minAgents int64) int64 {
	if households < 1 || scale < 1 {
		return 0
	}
	n := households / scale
	if households%scale != 0 {
		n++
	}
	return max(n, min(minAgents, households))
}

// groupWeights returns the weights of the agents of a group of households
// households, per-agent scale scale and floor minAgents (spec-scale.md 3.2,
// FR-SCL-001 to 005): agentCount agents with weights that sum to households and
// differ by at most 1, the first households mod agents of them taking the extra
// household. It takes no random number. It returns nil for a group with no
// household or a scale below 1, which validation rejects.
func groupWeights(households, scale, minAgents int64) []int64 {
	n := agentCount(households, scale, minAgents)
	if n < 1 {
		return nil
	}
	base, extra := households/n, households%n
	out := make([]int64, n)
	for i := range out {
		out[i] = base
		if int64(i) < extra {
			out[i]++
		}
	}
	return out
}

// groupShape returns the number of agents of a group and the largest weight of
// one (spec-scale.md 3.2): agentCount agents, and weights of H / n and one more
// for the first H mod n, so the largest is ceil(H / n). The last result is the
// scale in force: the group's, or the world's when the group gives none. ok is
// false when a household count, a scale or the floor is outside its bound, where
// the shape is not defined.
func groupShape(w WorldParams, g Group) (agents, maxWeight, scale int64, ok bool) {
	h, s := g.Households, g.Scale
	if s == 0 {
		s = w.Scale
	}
	if h < 1 || h > MaxHouseholds || s < 1 || s > MaxHouseholds || w.MinAgents < 1 || w.MinAgents > MaxHouseholds {
		return 0, 0, 0, false
	}
	n := agentCount(h, s, w.MinAgents)
	return n, (h + n - 1) / n, s, true
}

// groupPlan is one group of the configuration resolved to IDs, with the weights
// of its agents.
type groupPlan struct {
	region  RegionID
	role    RoleID
	weights []int64
	// warm is the group's entry in the warm start, nil when it has none; first is
	// the agent ID of its first agent, set when the agents are created.
	warm  *WarmGroup
	first int
}

// groupPlans resolves the groups of a canonical, validated configuration, in
// canonical order (region, then role), and derives their weights. The scale of
// a group is its own, or the world's when it gives none (FR-SCL-006).
func groupPlans(c *Config) []groupPlan {
	plans := make([]groupPlan, len(c.Groups))
	for i, g := range c.Groups {
		r, rok := c.regionID(g.Region)
		ro, ook := c.roleID(g.Role)
		if !rok || !ook {
			invariant("a group names an unknown region or role after validation")
		}
		weights := groupWeights(g.Households, cmp.Or(g.Scale, c.World.Scale), c.World.MinAgents)
		plans[i] = groupPlan{region: r, role: ro, weights: weights, warm: warmGroupOf(c, g.Region, g.Role)}
	}
	return plans
}
