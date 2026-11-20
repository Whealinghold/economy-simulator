package economysim_test

import (
	"errors"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// capSiteTicks is the length of S120: a mill that serves the one execution a tick of a
// household of weight 1 loses 6 + 2 = 8 units a tick of the 1 000 it has, so an unrepaired
// one is ruined in 125 ticks and 200 ticks show a repaired one holding.
const capSiteTicks = 200

// capRepairBelow is the condition under which the repairing owner asks for a repair.
const capRepairBelow = 300

// siteRun is what S120 keeps of one run.
type siteRun struct {
	prepared, repaired int64
	kinds              map[string]int
	mill               economysim.FacilityID
	last               economysim.FacilityState
	lastErr            error
}

// runSite runs one directed cook with a mill for capSiteTicks, asking for one execution of
// `prepare` a tick and, if repair is set, for a repair when the mill is under capRepairBelow.
func runSite(t *testing.T, seed uint64, repair bool) siteRun {
	t.Helper()
	cfg := capOnePlain(t, "")
	rec := &recorder{}
	w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
	cook := capAddCook(t, w, cfg, 0)
	mill := capAdd(t, w, capClass(cfg, "mill"), 0, economysim.NoRegion, cook)
	prepare, fix := capRecipe(cfg, "prepare"), capRecipe(cfg, "repair_mill")
	out := siteRun{mill: mill}
	for range capSiteTicks {
		out.last, out.lastErr = w.Facility(mill)
		if repair && out.lastErr == nil && out.last.Condition < capRepairBelow {
			capDirect(t, w, cook, fix, 1)
		} else {
			capDirect(t, w, cook, prepare, 1)
		}
		capStep(t, w)
	}
	out.last, out.lastErr = w.Facility(mill)
	out.prepared, out.repaired = capExecuted(rec.events, cook, prepare), capExecuted(rec.events, cook, fix)
	out.kinds = capKinds(rec.events)
	return out
}

// TestScenarioS120_SiteAndRepair verifies CAP-S2 of spec-infrastructure.md 16.2: pack one with
// overlay/infrastructure.toml, one directed cook with one mill (Durability 1 000, Decay 2,
// WearPerUse 6, Capacity 3), over 200 ticks with the audit on, with a repair when the
// condition falls under 300 and without. Exact: the unrepaired mill is ruined once, with the
// cause use or decay, and the query then says it is unknown; the repaired one is never
// ruined, is repaired at least once (one event of kind repaired and cause recipe for each repair execution) and holds a condition above
// 0; the repaired owner runs more executions of `prepare` than the other. The
// unrepaired owner's starts are refused after the ruin, which is why its executions stop at
// 125 or fewer.
//
// Requirement: FR-CAP-007, FR-CAP-008, FR-CAP-015, FR-CAP-019
//
// Guard: wear and decay that did not reach the ruin, or a repair that did not restore, would
// leave the two runs the same and the module's one loop (use, wear, repair) unproved.
func TestScenarioS120_SiteAndRepair(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			with, without := runSite(t, seed, true), runSite(t, seed, false)
			ruins := without.kinds["ruined/use"] + without.kinds["ruined/decay"]
			if ruins != 1 || !errors.Is(without.lastErr, economysim.ErrUnknownFacility) {
				t.Errorf("unrepaired: %d ruins (%v), the query says %v, want one ruin and ErrUnknownFacility", ruins, without.kinds, without.lastErr)
			}
			if n := with.kinds["ruined/use"] + with.kinds["ruined/decay"]; n != 0 || with.lastErr != nil || with.last.Condition <= 0 {
				t.Errorf("repaired: %d ruins, condition %d, error %v, want none, above 0 and nil", n, with.last.Condition, with.lastErr)
			}
			if with.repaired < 1 || with.kinds["repaired/recipe"] != int(with.repaired) {
				t.Errorf("repaired: %d repair executions, %d repaired events, want at least one, and one event for each", with.repaired, with.kinds["repaired/recipe"])
			}
			if without.prepared > 125 {
				t.Errorf("unrepaired: %d executions of prepare, want 125 or fewer (starts refused after the ruin)", without.prepared)
			}
			if with.prepared <= without.prepared {
				t.Errorf("executions of prepare: %d with repair, %d without, want more with", with.prepared, without.prepared)
			}
			t.Logf("seed %d: with repair %d prepared, %d repairs, condition %d; without %d prepared (events %v)", seed, with.prepared, with.repaired, with.last.Condition, without.prepared, without.kinds)
		})
	}
}
