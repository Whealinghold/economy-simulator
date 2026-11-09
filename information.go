package economysim

import "github.com/Whealinghold/economy-sim/internal/rumour"

// This file is the information module (spec-information.md): the price record
// of every market, and the delayed, blurred rumours of remote prices that the
// agents of an informed role are shown. The pure rules are in internal/rumour.
// The file holds, in the order of architecture.md 2.10.1: the validator and the
// warnings (the configuration surface). The state, the record's position, the
// filler of the trade view and the checkpoint section follow in the later
// phases of the plan that builds them.

// validateInformation checks the information module by spec-information.md 9.1
// and 9.2: the bounds of its three fields, a role flagged Informed while the
// module is off, and, only when the bounds hold, the two derived products of
// 9.2 (the cells of the record and the candidates of an informed role). It
// reports nothing about the module when it is off beyond the flag.
func (v *validator) validateInformation() {
	c := &v.c
	m := c.Modules.Information
	if m == nil {
		for i := range c.Roles {
			if c.Roles[i].Informed {
				v.rep.add(ErrModuleDependency, path([]string{"roles", c.Roles[i].ID}, "informed"), "true", "")
			}
		}
		return
	}
	base := []string{"modules", "information"}
	ok := v.inRange(path(base, "info_speed"), m.InfoSpeed, 1, MaxInfoSpeed)
	ok = v.inRange(path(base, "rumour_error_bp"), m.RumourErrorBP, 0, MaxRumourErrorBP) && ok
	ok = v.inRange(path(base, "rumour_max_age"), m.RumourMaxAge, 2, MaxRumourAge) && ok
	if !ok {
		return
	}
	sets := tradableSets(c, v.lk)
	cells := int64(0)
	for _, set := range regionTradable(c, v.lk, sets) {
		cells += int64(len(set))
	}
	if cells *= m.RumourMaxAge; cells > MaxRumourCells {
		v.rep.add(ErrTooMany, path(base, "rumour_max_age"), i64(cells), i64(MaxRumourCells))
	}
	others := int64(max(0, len(c.Regions)-1))
	for i := range c.Roles {
		if r := &c.Roles[i]; r.Informed && int64(len(sets[i]))*others > MaxRumoursPerView {
			v.rep.add(ErrTooMany, path([]string{"roles", r.ID}, "informed"), i64(int64(len(sets[i]))*others), i64(MaxRumoursPerView))
		}
	}
}

// regionTradable returns, for each region that has a group of a role of c, the
// commodities of c that the roles of its groups can trade (spec-information.md
// 4.1.1: a market is active when some agent of the region can trade it). The
// pairs (region, commodity) are the sources of the record. sets is
// tradableSets(c, lk). It reads a configuration that may not validate, so a
// group of an unknown role or a commodity that is not defined adds nothing.
func regionTradable(c *Config, lk lookup, sets []map[string]bool) map[string]map[string]bool {
	index := make(map[string]int, len(c.Roles))
	for i := range c.Roles {
		index[c.Roles[i].ID] = i
	}
	out := make(map[string]map[string]bool)
	for _, g := range c.Groups {
		i, ok := index[g.Role]
		if !ok || lk.region[g.Region] == nil {
			continue
		}
		if out[g.Region] == nil {
			out[g.Region] = make(map[string]bool)
		}
		for id := range sets[i] {
			if lk.commodity[id] != nil {
				out[g.Region][id] = true
			}
		}
	}
	return out
}

// informationWarnings returns the three warnings of spec-information.md 9.3
// that read the configuration alone, in no particular order:
// information_without_readers, information_without_routes and
// informed_without_sources. rumour_horizon_short needs the shortest distances
// and is horizonWarnings. They read a configuration that may not validate, so
// each guards its lookups.
func informationWarnings(c *Config, lk lookup) []Warning {
	if c.Modules.Information == nil {
		return nil
	}
	base := []string{"modules", "information"}
	var out []Warning
	informed := false
	for i := range c.Roles {
		informed = informed || c.Roles[i].Informed
	}
	if !informed {
		out = append(out, Warning{Path: base, Rule: "information_without_readers"})
	}
	if len(c.Regions) < 2 || len(c.Edges) == 0 {
		out = append(out, Warning{Path: base, Rule: "information_without_routes"})
	}
	return append(out, unsourcedRoles(c, lk)...)
}

// unsourcedRoles raises informed_without_sources for each informed role that
// would never be shown a rumour: in no region where it can be (the regions of
// its groups, every region when it has none) is there another region with a
// source among the commodities it trades.
func unsourcedRoles(c *Config, lk lookup) []Warning {
	sets := tradableSets(c, lk)
	byRegion := regionTradable(c, lk, sets)
	var out []Warning
	for i := range c.Roles {
		r := &c.Roles[i]
		if r.Informed && !hasCandidate(c, r.ID, sets[i], byRegion) {
			out = append(out, Warning{Path: path([]string{"roles", r.ID}, "informed"), Rule: "informed_without_sources"})
		}
	}
	return out
}

// hasCandidate reports whether an agent of the role, in some region where the
// role has a group (any region when it has none), has a candidate: a commodity
// of its tradable set that is a source in another region.
func hasCandidate(c *Config, role string, tradable map[string]bool, byRegion map[string]map[string]bool) bool {
	var homes []string
	for _, g := range c.Groups {
		if g.Role == role {
			homes = append(homes, g.Region)
		}
	}
	if len(homes) == 0 {
		for i := range c.Regions {
			homes = append(homes, c.Regions[i].ID)
		}
	}
	for _, home := range homes {
		for region, set := range byRegion {
			if region == home {
				continue
			}
			for id := range set {
				if tradable[id] {
					return true
				}
			}
		}
	}
	return false
}

// horizonWarnings returns rumour_horizon_short (spec-information.md 9.3): the
// ordered pairs of regions with a walk in the configured graph, no edge status
// applied, whose lag is above RumourMaxAge, so that no rumour is ever offered
// for them. Count is the number of pairs. It needs the shortest distances, so
// Warnings appends it after the structural phases come back clean, as it
// appends the viability warnings; the module being off or the graph not
// building gives nothing.
func horizonWarnings(c *Config) []Warning {
	m := c.Modules.Information
	if m == nil {
		return nil
	}
	g, err := NewGraph(*c)
	if err != nil {
		return nil
	}
	pairs := int64(0)
	for from := range RegionID(g.Len()) {
		for to := range RegionID(g.Len()) {
			d, ok := g.Distance(from, to)
			if !ok || from == to {
				continue
			}
			if lag, ok := rumour.Lag(d, m.InfoSpeed); ok && lag > m.RumourMaxAge {
				pairs++
			}
		}
	}
	if pairs == 0 {
		return nil
	}
	return []Warning{{Path: []string{"modules", "information", "rumour_max_age"}, Rule: "rumour_horizon_short", Count: pairs}}
}
