package economysim

// The Stage 1 shock seam of the scenario tests (plan 1-12, phase 7, O3):
// S01 (a) and S04 apply their yield shock through the world's shock slot, the
// way the public SetYieldShocks of INT will (spec-production.md 6.8). Stage 2
// replaces this file with the public setter. Nothing is exported from
// production code (architecture.md 10.2).

// SetYieldShockForTest sets the yield shock of a (region, recipe) pair in basis
// points between steps; 10 000 clears it.
func SetYieldShockForTest(w *World, region RegionID, recipe RecipeID, bp int64) error {
	return w.shocks.set(region, recipe, bp)
}

// RoleMoneyForTest returns the money held by the agents of each role, by RoleID
// (S04 with and without the demand module reads the cooks' share; the snapshot
// has no per-role money). A removed agent keeps its slot with weight 0 and no
// account of its own, so only agents of weight above 0 count.
func RoleMoneyForTest(w *World, roles int) []int64 {
	out := make([]int64, roles)
	for i := range w.agents {
		a := &w.agents[i]
		if v, err := w.ledger.Account(a.account); err == nil && a.weight > 0 && int(a.role) < roles {
			out[a.role] += int64(v.Balance)
		}
	}
	return out
}
