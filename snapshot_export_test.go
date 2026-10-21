package economysim

// This file is the test-only seam of the snapshot (architecture.md 10.2): the
// role signals of a (region, role) as the spawner's view and a hand-built set of
// agents give them, so the external test compares the snapshot with a second
// reading of the saved state. Nothing here is exported from production code.

// RoleSignalsView is the independent reading of one role row.
type RoleSignalsView struct {
	ExpectedRatioBP         int64
	Viable, HasExpected     bool
	MaxPrivation, MaxHunger int64
	HeldByFloor             int64
	Households              int64
}

// RoleSignalsOf reads the signals of role in region from the world as it
// stands: the expected ratio and viability from roleSignals over a fresh view,
// the largest privation and hunger over the agents, and the households the work
// floor holds over the agents with no job in progress, in a set built here
// agent by agent.
func RoleSignalsOf(w *World, region RegionID, role RoleID) RoleSignalsView {
	v := w.buildRegionView(region)
	var out RoleSignalsView
	out.ExpectedRatioBP, _, out.HasExpected, out.Viable = roleSignals(&v.Roles[role], v.TicksPerYear, v.EntryRatio)
	var set roleSet
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 || a.role != role || w.regionOf(a) != region {
			continue
		}
		out.Households += a.weight
		out.MaxPrivation, out.MaxHunger = max(out.MaxPrivation, a.privation), max(out.MaxHunger, a.hunger)
		set.rings, set.weights = append(set.rings, &a.history), append(set.weights, a.weight)
		set.targets, set.free = append(set.targets, a.target), append(set.free, !a.hasJob())
	}
	nc := len(w.cfg.Commodities)
	prices := make([]int64, nc)
	for c := range prices {
		prices[c] = int64(w.markets.reports[int(region)*nc+c].Price)
	}
	out.HeldByFloor = w.reads.heldByFloor(set, paramsOf(&w.cfg, role, &w.roles), w.roles.row(role).upkeepIn(region), prices)
	return out
}
