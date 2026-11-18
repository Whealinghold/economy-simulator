package economysim

import "slices"

// This file is the construction report of the infrastructure module
// (spec-infrastructure.md 13.2; the continuation file of infrastructure.go,
// architecture.md 2.6): the warnings of the configuration. They never fail
// validation, they are raised once each with a rule and a path, and none is logged
// inside a tick (FR-CAP-040).

// facilityUsage is what the roles' recipe lists do with the classes: which are needed,
// built and repaired by a recipe that some role lists, and which have an initial row.
type facilityUsage struct {
	needed, built, repaired, initial map[string]bool // membership tests only
	modes                            map[string]bool // transport modes of the merchant blocks
	listed                           map[string]bool // recipes some role lists
}

// facilityUsageOf scans the roles once.
func facilityUsageOf(c *Config, lk lookup) facilityUsage {
	u := facilityUsage{
		needed: map[string]bool{}, built: map[string]bool{}, repaired: map[string]bool{},
		initial: map[string]bool{}, modes: map[string]bool{}, listed: map[string]bool{},
	}
	for _, r := range c.Modules.Infrastructure.Initial {
		u.initial[r.Class] = true
	}
	for i := range c.Roles {
		ro := &c.Roles[i]
		if ro.Merchant != nil {
			for _, id := range ro.Merchant.Modes {
				u.modes[id] = true
			}
		}
		for _, id := range ro.Recipes {
			rc := lk.recipe[id]
			if rc == nil {
				continue
			}
			u.listed[id] = true
			for _, n := range rc.Needs {
				u.needed[n] = true
			}
			if rc.Builds != nil {
				u.built[rc.Builds.Class] = true
			}
			if rc.Repairs != nil {
				u.repaired[rc.Repairs.Class] = true
			}
		}
	}
	return u
}

// infrastructureWarnings are the warnings of spec-infrastructure.md 13.2: facility_unused,
// facility_unbuildable, unrunnable_recipe (extended), decay_without_repair,
// capacity_never_binds, route_unused_mode and facility_gate_closed. It reads the
// configuration as given and reports nothing with the module off.
func infrastructureWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Infrastructure
	if m == nil {
		return nil
	}
	u := facilityUsageOf(c, lk)
	var out []Warning
	for _, cl := range m.Classes {
		out = append(out, facilityClassWarnings(c, lk, u, cl)...)
	}
	for i := range c.Recipes {
		rc := &c.Recipes[i]
		// A recipe no role lists never runs, so it is unused_recipe's and not this
		// rule's; one warning a recipe, whatever number of its classes are missing.
		if !u.listed[rc.ID] {
			continue
		}
		if slices.ContainsFunc(rc.Needs, func(id string) bool { return !u.initial[id] && !u.built[id] }) {
			out = append(out, Warning{Path: []string{"recipes", rc.ID, "needs"}, Rule: "unrunnable_recipe"})
		}
	}
	return append(out, facilityGateWarnings(c, m)...)
}

// facilityClassWarnings are the warnings about one class.
func facilityClassWarnings(c *Config, lk lookup, u facilityUsage, cl FacilityClass) []Warning {
	p := []string{"modules", "infrastructure", "class", cl.ID}
	var out []Warning
	add := func(rule string) { out = append(out, Warning{Path: p, Rule: rule}) }
	if cl.Kind == kindSite && !u.needed[cl.ID] {
		add("facility_unused")
	}
	if !u.initial[cl.ID] && !u.built[cl.ID] {
		add("facility_unbuildable")
	}
	if (cl.Decay > 0 || cl.WearPerUse > 0) && !u.repaired[cl.ID] {
		add("decay_without_repair")
	}
	for _, rule := range facilityKindWarnings(c, lk, u, cl) {
		add(rule)
	}
	return out
}

// facilityKindWarnings are the warnings of a site and of a route class: the rules
// capacity_never_binds and route_unused_mode.
func facilityKindWarnings(c *Config, lk lookup, u facilityUsage, cl FacilityClass) []string {
	switch {
	case cl.Kind == kindSite && facilityCapacityNeverBinds(c, lk, cl):
		return []string{"capacity_never_binds"}
	case cl.Kind == kindRoute && !facilityModeUsed(c, u, cl):
		return []string{"route_unused_mode"}
	}
	return nil
}

// facilityCapacityNeverBinds reports whether the capacity of a site class is at least
// the largest weight of a group whose role lists a recipe that needs it, and some
// such group exists: the capacity never cuts a start.
func facilityCapacityNeverBinds(c *Config, lk lookup, cl FacilityClass) bool {
	var heaviest int64
	for _, g := range c.Groups {
		ro := lk.role[g.Role]
		if ro == nil || !roleNeeds(lk, ro, cl.ID) {
			continue
		}
		if _, w, _, ok := groupShape(c.World, g); ok {
			heaviest = max(heaviest, w)
		}
	}
	return heaviest > 0 && cl.Capacity >= heaviest
}

// roleNeeds reports whether a recipe on the role's list needs the class.
func roleNeeds(lk lookup, ro *Role, class string) bool {
	for _, id := range ro.Recipes {
		if rc := lk.recipe[id]; rc != nil {
			for _, n := range rc.Needs {
				if n == class {
					return true
				}
			}
		}
	}
	return false
}

// facilityModeUsed reports whether a mode of a route class is listed by a merchant
// block or is a hired mode with a haul commodity: the road is speeded for somebody.
func facilityModeUsed(c *Config, u facilityUsage, cl FacilityClass) bool {
	hired := map[string]bool{}
	if t := c.Modules.Transport; t != nil {
		for i := range t.Modes {
			if t.Modes[i].hired() {
				hired[t.Modes[i].ID] = true
			}
		}
	}
	for _, id := range cl.Modes {
		if u.modes[id] || hired[id] {
			return true
		}
	}
	return false
}

// facilityGateWarnings is facility_gate_closed: an initial row in a region whose
// settlement class at construction lacks the class's Needs flag (spec-infrastructure.md
// 8.4); the facility is created all the same.
func facilityGateWarnings(c *Config, m *Infrastructure) []Warning {
	st := c.Modules.Settlements.canonical()
	if st == nil || len(st.Classes) == 0 {
		return nil
	}
	census, ok := newSettlementCensus(c, st)
	if !ok {
		return nil
	}
	needs := make(map[string]string, len(m.Classes)) // lookup only
	for _, cl := range m.Classes {
		needs[cl.ID] = cl.Needs
	}
	at := make(map[string]int, len(c.Regions)) // lookup only
	for i := range c.Regions {
		at[c.Regions[i].ID] = i
	}
	var out []Warning
	for i, r := range m.Initial {
		flag, ri := needs[r.Class], at[r.Region]
		if _, known := at[r.Region]; !known || flag == "" {
			continue
		}
		if !slices.Contains(st.Classes[census.class[ri]].Flags, flag) {
			out = append(out, Warning{Path: facilityRowPath(i), Rule: "facility_gate_closed"})
		}
	}
	return out
}
