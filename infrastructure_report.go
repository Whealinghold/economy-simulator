package economysim

import (
	"context"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the report of the infrastructure module (spec-infrastructure.md 13.2 and
// 14; the continuation file of infrastructure.go, architecture.md 2.6): the warnings of
// the configuration, which never fail validation, are raised once each with a rule and a
// path and none of which is logged inside a tick (FR-CAP-040); and the events, the
// Debug lines and the snapshot rows.

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

// FacilityRow is the facilities of one class in one region at the boundary
// (spec-infrastructure.md 14.5): one row for each (region, class) with a facility, in
// ascending region then class. All are functions of the saved state.
type FacilityRow struct {
	Region RegionID
	Class  FacilityClassID
	// Facilities is the count of facilities, never of agents or households.
	Facilities int64
	// ConditionBP is the sum of Condition * 10 000 / Durability over the facilities, so
	// the mean is it over Facilities.
	ConditionBP int64
	// Effect is the sum of eff for a route or ground class and of cap for a site, 0 for a
	// store.
	Effect int64
	// Pending is the store facilities whose store has not been given.
	Pending int64
}

// snapshotFacilities fills the facility rows of the snapshot, nothing with the module off.
func (w *World) snapshotFacilities(s *Snapshot) {
	if w.infra != nil {
		s.Facilities = w.facilityRows()
	}
}

// facilityRows builds the snapshot rows (14.5), a deep copy in ascending (region, class).
func (w *World) facilityRows() []FacilityRow {
	n := w.infra
	nc := len(n.cls)
	cells := make([]FacilityRow, n.regions*nc)
	for i := range n.fac {
		f := &n.fac[i]
		row := &cells[int(f.region)*nc+int(f.class)]
		row.Facilities++
		row.ConditionBP += f.condition * wholeBP / n.cls[f.class].durability
		row.Effect += n.eff(f)
		if n.cls[f.class].kind == kindStore && !f.installed {
			row.Pending++
		}
	}
	var out []FacilityRow
	for k := range cells {
		if cells[k].Facilities > 0 {
			cells[k].Region, cells[k].Class = RegionID(k/nc), FacilityClassID(k%nc)
			out = append(out, cells[k])
		}
	}
	return out
}

// announce emits the FacilityChanged events of the initial facilities (kind added, cause
// initial), in ascending ID, after the last AgentSpawned (spec-infrastructure.md 3.4.3).
// It is nil-safe and builds nothing without an observer.
func (n *infrastructure) announce(w *World) {
	if n == nil || !w.obs.supplied {
		return
	}
	for i := range n.fac {
		w.reportFacility(&n.fac[i], FacilityAdded, CauseInitial, 0, 0)
	}
}

// reportFacility delivers FacilityChanged for a transition of f, after it is applied,
// and logs the Debug line of a build, repair, install or ruin. f is the record after the
// transition (for a ruin, a copy with condition 0), before is the condition before it and
// goods the lots a forced removal of a store listed to the estate. The event is built only with an observer (observability-and-logging.md 4.5).
func (w *World) reportFacility(f *facility, kind FacilityKind, cause Cause, before, goods int64) {
	n := w.infra
	after := f.condition // 0 for a ruin: the caller passes the record as it ends
	if w.obs.supplied {
		w.obs.deliver(FacilityChanged{Tick: w.Tick(), Facility: f.id, Class: f.class, Region: f.region, To: f.to,
			Owner: f.owner, Kind: kind, Cause: cause, Before: before, After: after, Effect: n.eff(f), GoodsListed: goods})
	}
	ctx := context.Background()
	if msg := facilityMessage(kind); msg != "" && w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, msg, slog.Int64(logkeys.Tick, w.Tick()),
			slog.Int64(logkeys.Facility, int64(f.id)), slog.String(logkeys.FacilityClass, w.cfg.Modules.Infrastructure.Classes[f.class].ID),
			slog.Int64(logkeys.Agent, int64(f.owner)), slog.Int64(logkeys.Condition, after),
			slog.String(logkeys.Cause, string(cause)))
	}
}

// logVoided is the Debug line of executions of a build or repair that did nothing at the
// completion (4.3.4): the inputs are spent and nothing was made, which only FacilityLimited
// counted, and only under an observer.
func (n *infrastructure) logVoided(region RegionID, class FacilityClassID, executions int64) {
	w := n.world
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "facility executions voided", slog.Int64(logkeys.Tick, w.Tick()),
			slog.Int64(logkeys.Region, int64(region)), slog.String(logkeys.FacilityClass, w.cfg.Modules.Infrastructure.Classes[class].ID),
			slog.Int64(logkeys.Executions, executions))
	}
}

// facilityMessage is the constant message of the Debug line of 14.3; a kind with no line
// (a facility added is reported by its event) returns "". A function, not a package-level
// map, so no mutable package state exists.
func facilityMessage(k FacilityKind) string {
	switch k {
	case FacilityBuilt:
		return "facility built"
	case FacilityRepaired:
		return "facility repaired"
	case FacilityRuined:
		return "facility ruined"
	case FacilityInstalled:
		return "facility installed"
	}
	return ""
}

// cell is the index of the counters of a (region, class).
func (n *infrastructure) cell(region RegionID, class FacilityClassID) int {
	return int(region)*len(n.cls) + int(class)
}

// countPending counts a store facility left pending by the install step. Counters exist
// only to build FacilityLimited, so they are kept only under an observer
// (observability-and-logging.md 4.5).
func (n *infrastructure) countPending(f *facility) {
	if n.world.obs.supplied {
		n.limits[n.cell(f.region, f.class)].pending++
	}
}

// reportLimits is step 5 of 6.3.2: FacilityLimited for each (region, class) with a count,
// in ascending region then class, never when all five are 0. It builds nothing without an
// observer.
func (n *infrastructure) reportLimits() {
	w := n.world
	if !w.obs.supplied {
		return
	}
	nc := len(n.cls)
	for k := range n.limits {
		if l := n.limits[k]; l.cut+l.refused+l.voided+l.pending != 0 {
			w.obs.deliver(FacilityLimited{Tick: w.Tick(), Region: RegionID(k / nc), Class: FacilityClassID(k % nc),
				Served: l.served, Cut: l.cut, Refused: l.refused, Voided: l.voided, Pending: l.pending})
		}
	}
}

// clearTick clears the counters of the tick (P8.6). It is nil-safe: with the module off
// it does nothing.
func (n *infrastructure) clearTick() {
	if n != nil {
		clear(n.limits)
		clear(n.used)
		clear(n.ordered)
		n.orderedAll = 0
	}
}
