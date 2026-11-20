package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packThreeRoad is the pack three with its trade and transport overlays, the layers a road
// needs (the edges, and the mode it speeds), and the extra layer of S121: one route class
// that speeds foot by 6 000 basis points, and a build recipe on the cook role.
var packThreeRoad = []string{"base.toml", "three.toml", tradeOverlay("three"), "overlay/transport-three.toml"}

// capRoadLayer is the extra configuration of S121: the road class and its build recipe. The
// numbers are placeholders (N-018, N-027). The road is built by a cook of the north toward the
// south, and a new road is at full condition.
const capRoadLayer = `
[modules.infrastructure.class.road]
kind = "route"
durability = 2000
decay = 1
boost_bp = 6000
modes = [ "foot" ]

[recipe.build_road]
duration = 3
inputs = [ { commodity = "staple", quantity = 2 } ]
[recipe.build_road.builds]
class = "road"
toward = "south"

[role.cook]
recipes = [ "prepare", "prepare_basic", "build_road" ]
`

// roadWorld builds the three-region world with the road class and one directed cook of the north.
func roadWorld(t *testing.T) (*economysim.World, economysim.Config, economysim.AgentID) {
	t.Helper()
	cfg := loadPackBytes(t, []byte(capRoadLayer), packThreeRoad...)
	w := mustWorld(t, cfg, economysim.WithSeed(1), economysim.WithAudit())
	return w, cfg, capAddCook(t, w, cfg, capRegion(cfg, "north"))
}

// footTicks is the ticks a journey on foot takes between two regions.
func footTicks(t *testing.T, w *economysim.World, cfg economysim.Config, from, to string) int64 {
	t.Helper()
	foot := economysim.ModeID(slices.IndexFunc(cfg.Modules.Transport.Modes, func(m economysim.Mode) bool { return m.ID == "foot" }))
	k, _, ok := w.Travel(capRegion(cfg, from), capRegion(cfg, to), foot)
	if !ok {
		t.Fatalf("Travel(%s, %s, foot) has no route", from, to)
	}
	return k
}

// TestScenarioS121_Road verifies CAP-S3 of spec-infrastructure.md 16.2 in the part the public
// API proves: pack three with the trade and transport overlays, a road class of 6 000 basis
// points on foot between the north and the south, in three worlds: none, planted by
// AddFacilities and built by a directed cook of the north with a 3-tick recipe. Exact: the
// 100-long edge takes 5 ticks on foot with no road and 4 with it (F = 16 000), in both
// directions; the edge between the south and the far south is not speeded; the built road
// changes no journey before its completion (a 3-tick recipe, so the ticks 0 and 1 read 5) and
// gives the planted road's journey from the completing tick on: the effect enters the table at
// that tick's P8.3b, after every position that could read it (FR-CAP-016); the world with no
// road never changes. The free-rider comparison and a shipment in flight are not in this test: the
// shipment in flight is proved by TestRoute_BothEdges and the price gap is a report of the
// calibration platform.
//
// Requirement: FR-CAP-021, FR-CAP-022
//
// Guard: a layer that is the identity, one that speeds the wrong edge or one that takes
// effect in the tick of the completion each fail one of these journeys.
func TestScenarioS121_Road(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	none, cfg, _ := roadWorld(t)
	planted, _, owner := roadWorld(t)
	capAdd(t, planted, capClass(cfg, "road"), capRegion(cfg, "north"), capRegion(cfg, "south"), owner)
	built, _, builder := roadWorld(t)
	capDirect(t, built, builder, capRecipe(cfg, "build_road"), 1)
	if k := footTicks(t, built, cfg, "north", "south"); k != 5 {
		t.Fatalf("journey before the build = %d, want 5", k)
	}
	for tick := range 8 {
		for _, w := range []*economysim.World{none, planted, built} {
			capStep(t, w)
		}
		got := footTicks(t, built, cfg, "north", "south")
		if want := int64(4); tick < 2 {
			want = 5
			if got != want {
				t.Errorf("tick %d: journey %d with the road not completed, want 5", tick, got)
			}
		} else if got != want {
			t.Errorf("tick %d: journey %d after the completing tick's P8.3b, want 4", tick, got)
		}
	}
	checkJourneys(t, cfg, none, planted, built)
	snap, err := built.Snapshot()
	if err != nil || len(snap.Facilities) != 1 || snap.Facilities[0].Facilities != 1 {
		t.Errorf("rows after the build = %+v, %v, want one row of one road", snap.Facilities, err)
	}
}

// checkJourneys asserts the foot journeys of S121 in the three worlds at the end of the run.
func checkJourneys(t *testing.T, cfg economysim.Config, none, planted, built *economysim.World) {
	t.Helper()
	for name, c := range map[string]struct {
		w        *economysim.World
		from, to string
		want     int64
	}{
		"none":              {none, "north", "south", 5},
		"planted there":     {planted, "north", "south", 4},
		"planted back":      {planted, "south", "north", 4},
		"planted elsewhere": {planted, "south", "far_south", 5},
		"built there":       {built, "north", "south", 4},
		"built back":        {built, "south", "north", 4},
	} {
		if got := footTicks(t, c.w, cfg, c.from, c.to); got != c.want {
			t.Errorf("%s: %s to %s on foot = %d ticks, want %d", name, c.from, c.to, got, c.want)
		}
	}
}
