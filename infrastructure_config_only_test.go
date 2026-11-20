package economysim_test

import (
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// secondKilnLayer is a site, a build recipe and a repair recipe written for the second setting
// (the crater pack, which has no commodity or role of the first): a kiln that `melt` needs,
// held by 100 of the drillers from the start, and the two recipes that make and mend one. It is
// data only; the numbers are placeholders (N-018, N-027).
const secondKilnLayer = `
[modules.infrastructure.class.kiln]
kind = "site"
durability = 600
decay = 1
wear_per_use = 3
capacity = 2

[[modules.infrastructure.initial]]
region = "crater_rim"
class = "kiln"
role = "driller"
count = 100

[recipe.melt]
needs = [ "kiln" ]

[recipe.build_kiln]
duration = 2
inputs = [ { commodity = "ice", quantity = 2 } ]
[recipe.build_kiln.builds]
class = "kiln"

[recipe.repair_kiln]
inputs = [ { commodity = "ice", quantity = 1 } ]
[recipe.repair_kiln.repairs]
class = "kiln"
units = 200

[role.driller]
recipes = [ "cut_ice", "melt", "build_kiln", "repair_kiln" ]
`

// TestConfigOnly_Infrastructure verifies FR-CAP-045: a site, a build recipe and a repair
// recipe written from a file alone, on a setting that shares no commodity, recipe or role
// with pack one, load, validate and run 300 ticks with the audit on, seeds 1 to 3. The kilns
// exist from the start, the recipe that needs one runs, they wear and are ruined, and no step
// fails: nothing in the module names the identifier of a commodity, a recipe or a role.
//
// Requirement: FR-CAP-045
//
// Guard: a branch on one of the first pack's identifiers in the module would not find the
// kiln or its recipes on another setting, and the world would not run, or would run with no
// facility.
func TestConfigOnly_Infrastructure(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	cfg := loadPackBytes(t, []byte(secondKilnLayer), packSecond...)
	for _, seed := range seedList(min(3, scenarioSeedCount())) {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
			for range 300 {
				capStep(t, w)
			}
			kinds := capKinds(rec.events)
			melt := capRecipe(cfg, "melt")
			var melted int64
			for _, e := range rec.events {
				if x, ok := e.(economysim.RecipeExecuted); ok && x.Recipe == melt {
					melted += x.Executions
				}
			}
			if kinds["added/initial"] != 100 || melted == 0 || kinds["ruined/use"]+kinds["ruined/decay"] == 0 {
				t.Errorf("seed %d: %d kilns added, %d executions of melt, events %v, want 100 added, some melting and some ruin", seed, kinds["added/initial"], melted, kinds)
			}
		})
	}
}
