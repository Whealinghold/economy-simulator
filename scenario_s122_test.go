package economysim_test

import (
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// capThinTicks is the length of S122: the build takes 3 ticks and the failure ladder of a thin
// household acts within a few more.
const capThinTicks = 40

// builtBy counts the facilities of the events that a build made for the owner.
func builtBy(events []economysim.Event, owner economysim.AgentID) (n int) {
	for _, e := range events {
		if fc, ok := e.(economysim.FacilityChanged); ok && fc.Kind == economysim.FacilityBuilt && fc.Owner == owner {
			n++
		}
	}
	return n
}

// abandonedBy sums the executions of a job that ended before it completed (RecipeAbandoned).
func abandonedBy(events []economysim.Event, agent economysim.AgentID) (n int64) {
	for _, e := range events {
		if x, ok := e.(economysim.RecipeAbandoned); ok && x.Agent == agent {
			n += x.Executions
		}
	}
	return n
}

// TestScenarioS122_ThinBuilder verifies CAP-S4 of spec-infrastructure.md 16.2 in the part that
// holds with no relief or migration layer: pack one with overlay/infrastructure.toml, a
// buffered directed cook (the stock and money of the other scenarios) and a thin mortal one (the
// two units of staple that one build takes, no money, no other goods) each directed to build a
// mill (3 ticks, 1 execution), 40 ticks, the audit on. Exact: the buffered builder finishes and
// owns one mill (kind built, cause recipe); the thin one ends with its one execution accounted
// for exactly once, as a mill it owns or as an abandoned execution (RecipeAbandoned), never
// both and never neither, so a build that the household could not carry is lost with its
// inputs and makes no facility; the audit passes at every step. Reported: which of the two the
// thin builder met. The relief and migration modules of the scenario are not layered here: the
// void completion (FR-CAP-017) is proved at the unit level by TestComplete_Voided.
//
// Requirement: FR-CAP-017, FR-CAP-014
//
// Guard: a build that made a facility and also counted as abandoned, or neither, would create or
// lose capital without a record.
func TestScenarioS122_ThinBuilder(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			cfg := capOnePlain(t, "")
			rec := &recorder{}
			w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
			fat := capAddCook(t, w, cfg, 0)
			thin, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: capRole(cfg, "cook"), Strategy: "directed", Mortal: true,
				Stock: []economysim.StockItem{{Commodity: capCommodity(cfg, "staple"), Quantity: 2}}})
			if err != nil {
				t.Fatalf("AddAgent(thin) error = %v, want nil", err)
			}
			build := capRecipe(cfg, "build_mill")
			capDirect(t, w, fat, build, 1)
			capDirect(t, w, thin, build, 1)
			for range capThinTicks {
				capStep(t, w)
			}
			if n := builtBy(rec.events, fat); n != 1 {
				t.Errorf("the buffered builder owns %d built mills, want 1", n)
			}
			made, lost := int64(builtBy(rec.events, thin)), abandonedBy(rec.events, thin)
			if made+lost != 1 {
				t.Errorf("the thin builder made %d mills and abandoned %d executions, want exactly one of the two", made, lost)
			}
			t.Logf("seed %d: the thin builder made %d mill(s) and abandoned %d execution(s)", seed, made, lost)
		})
	}
}
