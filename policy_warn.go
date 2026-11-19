package economysim

import "slices"

// This file holds the warnings of the policy module (spec-policy.md 11.2; the
// continuation file of policy.go, architecture.md 2.6). A warning never fails
// validation, and a warning about a thing that is not declared is skipped.

// policyWarnings returns the warnings of spec-policy.md 11.2; it reports nothing when
// the module is off.
func policyWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Policy
	if m == nil {
		return nil
	}
	if len(m.Actors) == 0 {
		return []Warning{{Path: path(policyBase), Rule: "no_actors"}}
	}
	var out []Warning
	ticksPerYear := c.Calendar.TicksPerYear()
	for i, a := range m.Actors {
		out = append(out, policyActorWarnings(i, a, ticksPerYear)...)
		out = append(out, policyRuleWarnings(i, a)...)
		out = append(out, policyDenseWarnings(i, a)...)
	}
	return append(out, policyNoKeyWarnings(c, lk)...)
}

// policyActorWarnings is actor_no_levers, cadence_long, reads_unused and lever_fixed.
func policyActorWarnings(i int, a PolicyActor, ticksPerYear int64) []Warning {
	var out []Warning
	warn := func(rule string, p []string) { out = append(out, Warning{Path: p, Rule: rule}) }
	if len(a.Levers) == 0 {
		warn("actor_no_levers", policyActorPath(i))
	}
	if ticksPerYear > 0 && a.Every > ticksPerYear {
		warn("cadence_long", policyActorPath(i, "every"))
	}
	if a.ReadsRegions && !policyReadsRegions(a) {
		warn("reads_unused", policyActorPath(i, "reads_regions"))
	}
	for j, l := range a.Levers {
		if l.Min == l.Max {
			warn("lever_fixed", policyListPath(i, "levers", j, "min"))
		}
	}
	return out
}

// policyReadsRegions says the actor's policy reads a region: a rule with a gauge other
// than the treasury. A hold, a schedule and a directed policy read none, and a name that
// is not built in is a consumer's, which may.
func policyReadsRegions(a PolicyActor) bool {
	switch a.Policy {
	case policyHold, policySchedule, policyDirected:
		return false
	case policyRule:
		return slices.ContainsFunc(a.Rules, func(r Reaction) bool { return r.Gauge != GaugeTreasury })
	}
	return true
}

// policyRuleWarnings is rule_undamped (a reaction whose lever has MaxStep 0 and whose
// Deadband is 0, 8.3.6) and rule_inert (Move 0).
func policyRuleWarnings(i int, a PolicyActor) []Warning {
	var out []Warning
	for j, r := range a.Rules {
		if r.Move == 0 {
			out = append(out, Warning{Path: policyListPath(i, "rules", j, "move"), Rule: "rule_inert"})
		}
		if l, ok := actorLever(a, r.Lever); ok && l.MaxStep == 0 && r.Deadband == 0 {
			out = append(out, Warning{Path: policyListPath(i, "rules", j, "deadband"), Rule: "rule_undamped"})
		}
	}
	return out
}

// policyDenseWarnings is schedule_dense: with MaxStep above 0, two steps of one lever
// closer than Every ticks (with a cycle, the last and the first plus the cycle as well).
func policyDenseWarnings(i int, a PolicyActor) []Warning {
	var out []Warning
	for li, l := range a.Levers {
		if l.MaxStep == 0 {
			continue
		}
		var ats []int64
		for _, s := range a.Schedule {
			if s.Lever == l.ID {
				ats = append(ats, s.At)
			}
		}
		slices.Sort(ats)
		if scheduleDense(ats, a.Cycle, a.Every) {
			out = append(out, Warning{Path: policyListPath(i, "levers", li, "max_step"), Rule: "schedule_dense"})
		}
	}
	return out
}

// scheduleDense says two consecutive ticks of a sorted list are less than every apart,
// the last and the first plus the cycle included when there is a cycle.
func scheduleDense(ats []int64, cycle, every int64) bool {
	for x := 1; x < len(ats); x++ {
		if ats[x]-ats[x-1] < every {
			return true
		}
	}
	return cycle > 0 && len(ats) > 1 && ats[0]+cycle-ats[len(ats)-1] < every
}

// policyNoKeyWarnings is lever_no_keys: an Against lever that expands to no edge. It is
// skipped when the polity or the graph is not sound.
func policyNoKeyWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Policy
	pol, ok := soundPolity(c, lk)
	if !ok || graphRulesError(*c) != nil {
		return nil
	}
	cc := c.canonical()
	g, err := newGraph(&cc, nil)
	if err != nil {
		return nil
	}
	var out []Warning
	for i, a := range m.Actors {
		j, known := pol.Lookup(a.Jurisdiction)
		for li, l := range a.Levers {
			other, found := pol.Lookup(l.Against)
			if l.Against == "" || !known || !found {
				continue
			}
			if len(policyLeverEdges(pol, g, j, other)) == 0 {
				out = append(out, Warning{Path: policyListPath(i, "levers", li, "against"), Rule: "lever_no_keys"})
			}
		}
	}
	return out
}
