package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS124_OwnerDies verifies CAP-S6 of spec-infrastructure.md 16.2: pack one with
// overlay/infrastructure.toml, two directed cooks and one mill that the first owns. The first
// works the mill, is removed (RemoveAgent), and the mill passes to the estate of its region
// (FacilityChanged, kind transferred, cause owner_removed) and idles: the second cook, directed
// to prepare each tick, runs none for five ticks and every start is refused. Then
// TransferFacilities gives the mill to the second, which runs `prepare` from the next tick.
// Exact: the facility count is 1 throughout, the audit passes at every step (no good or money
// is created or lost by the transfers), and the mill's condition never rises.
//
// Requirement: FR-CAP-030, FR-CAP-036
//
// Guard: a removal that deleted the mill, a mill that served a non-owner, or a transfer that
// left the owner unchanged each break one of the three phases.
func TestScenarioS124_OwnerDies(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	cfg := capOnePlain(t, "")
	rec := &recorder{}
	w := mustWorld(t, cfg, economysim.WithSeed(1), economysim.WithAudit(), economysim.WithObserver(rec))
	first, second := capAddCook(t, w, cfg, 0), capAddCook(t, w, cfg, 0)
	mill := capAdd(t, w, capClass(cfg, "mill"), 0, economysim.NoRegion, first)
	prepare := capRecipe(cfg, "prepare")
	run := func(n int, who economysim.AgentID) {
		for range n {
			capDirect(t, w, who, prepare, 1)
			capStep(t, w)
		}
	}
	run(5, first)
	expectRuns(t, rec, first, prepare, true, "before the removal")
	cond := removeOwner(t, w, rec, first, mill)
	run(5, second)
	expectRuns(t, rec, second, prepare, false, "while the estate holds the mill")
	if err := w.TransferFacilities([]economysim.FacilityTransfer{{ID: mill, To: second}}); err != nil {
		t.Fatalf("TransferFacilities() error = %v, want nil", err)
	}
	run(5, second)
	expectRuns(t, rec, second, prepare, true, "after the transfer")
	f, err := w.Facility(mill)
	if err != nil || f.Owner != second || f.Condition > cond {
		t.Errorf("mill at the end = %+v, %v, want owner %d and a condition of at most %d", f, err, second, cond)
	}
	if snap, err := w.Snapshot(); err != nil || len(snap.Facilities) != 1 || snap.Facilities[0].Facilities != 1 {
		t.Errorf("facility rows = %+v, %v, want one row of one mill", snap.Facilities, err)
	}
}

// removeOwner removes the owner of the mill and checks that the mill went to the estate of
// region 0 with one transferred/owner_removed event; it returns the mill's condition.
func removeOwner(t *testing.T, w *economysim.World, rec *recorder, owner economysim.AgentID, mill economysim.FacilityID) int64 {
	t.Helper()
	if err := w.RemoveAgent(owner); err != nil {
		t.Fatalf("RemoveAgent() error = %v, want nil", err)
	}
	estate := economysim.AgentID(0)
	f, err := w.Facility(mill)
	if err != nil || f.Owner != estate {
		t.Fatalf("mill after the removal = %+v, %v, want owner %d (the estate of region 0)", f, err, estate)
	}
	if got := capKinds(rec.events)["transferred/owner_removed"]; got != 1 {
		t.Errorf("transferred/owner_removed events = %d, want 1", got)
	}
	return f.Condition
}

// expectRuns asserts that the agent has run the recipe (or has not) in the events so far.
func expectRuns(t *testing.T, rec *recorder, who economysim.AgentID, recipe economysim.RecipeID, want bool, when string) {
	t.Helper()
	if n := capExecuted(rec.events, who, recipe); (n > 0) != want {
		t.Errorf("agent %d ran %d executions %s, want some: %v", who, n, when, want)
	}
}
