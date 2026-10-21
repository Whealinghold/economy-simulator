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
