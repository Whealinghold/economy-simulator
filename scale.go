package economysim

import (
	"cmp"
	"math"
)

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

// scaleOf is the per-agent scale in force for a group: its own, or the world's
// when it gives none (FR-SCL-006). Validation and construction both read it.
func scaleOf(w WorldParams, g Group) int64 { return cmp.Or(g.Scale, w.Scale) }

// groupShape returns the number of agents of a group and the largest weight of
// one (spec-scale.md 3.2): agentCount agents, and weights of H / n and one more
// for the first H mod n, so the largest is ceil(H / n). The last result is the
// scale in force: the group's, or the world's when the group gives none. ok is
// false when a household count, a scale or the floor is outside its bound, where
// the shape is not defined.
func groupShape(w WorldParams, g Group) (agents, maxWeight, scale int64, ok bool) {
	h, s := g.Households, scaleOf(w, g)
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
		weights := groupWeights(g.Households, scaleOf(c.World, g), c.World.MinAgents)
		plans[i] = groupPlan{region: r, role: ro, weights: weights, warm: warmGroupOf(c, g.Region, g.Role)}
	}
	return plans
}

// checkCensus is the census invariant (spec-scale.md 7.1, FR-SCL-003, 005, 019),
// run once the agents of every group are created: for each group the weights sum
// to its Households, differ by at most 1 and are each at least 1, a group of
// weight above 1 has at least min(MinAgents, Households) agents, and the totals
// of the plans equal the live counters agents and households. A validated
// configuration cannot break it, so a violation is a programmer invariant and
// panics, like the other invariants of construction (D283). It is O(groups) and
// allocation-free.
func checkCensus(plans []groupPlan, c *Config, agents, households int64) {
	if len(plans) != len(c.Groups) {
		invariant("the plans of a world do not match its groups")
	}
	var nAgents, nHouseholds int64
	for i := range plans {
		n, sum := checkGroupWeights(plans[i], c, c.Groups[i])
		nAgents += n
		nHouseholds += sum
	}
	if nAgents != agents || nHouseholds != households {
		invariant("the agents or households of the world differ from the census")
	}
}

// checkGroupWeights is the per-group part of checkCensus; it returns the group's
// agent count and the sum of its weights.
func checkGroupWeights(p groupPlan, c *Config, g Group) (n, sum int64) {
	lo, hi := int64(math.MaxInt64), int64(0)
	for _, w := range p.weights {
		sum += w
		lo, hi = min(lo, w), max(hi, w)
	}
	n = int64(len(p.weights))
	switch {
	case sum != g.Households:
		invariant("the weights of a group do not sum to its households")
	case lo < 1:
		invariant("an agent of a group has a weight below 1")
	case hi-lo > 1:
		invariant("the weights of a group differ by more than 1")
	case hi > 1 && n < min(c.World.MinAgents, g.Households):
		invariant("a group of weight above 1 has fewer agents than its floor")
	}
	return n, sum
}
