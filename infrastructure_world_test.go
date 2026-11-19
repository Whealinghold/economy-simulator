package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// infrastructureOverlay is a layer for pack one in the file form of
// spec-infrastructure.md 12.1: one class of the kind site, two initial rows of it held
// by the growers of the north, and a recipe that needs it (named to sort after every recipe of the pack, so no recipe
// ID of the pack moves). No role lists the recipe, so
// the run has nothing to do with the module until its steps exist (plan 3-cap, PR 2).
const infrastructureOverlay = `
[modules.infrastructure.class.mill]
kind = "site"
durability = 1000
decay = 1
wear_per_use = 2
capacity = 4

[[modules.infrastructure.initial]]
region = "north"
class = "mill"
role = "grower"
count = 2

[recipe.zz_grind]
duration = 1
needs = ["mill"]
inputs = [ { commodity = "seed", quantity = 1 } ]
outputs = [ { commodity = "staple", quantity = 2, cost_share = 10000 } ]
`

// quietRun runs a world of c for ticks steps with the audit on and returns the events
// after WorldCreated and the final snapshot with the fields the module's presence
// changes by design zeroed: the configuration hash, the names of the IDs (the extra
// recipe) and the module list.
func quietRun(t *testing.T, c economysim.Config, seed uint64, ticks int64) ([]economysim.Event, economysim.Snapshot) {
	t.Helper()
	res := runPack(t, c, runSpec{Seed: seed, Ticks: ticks, Drive: driveStep, Observe: true, Audit: true})
	var events []economysim.Event
	for _, e := range res.Events {
		if _, ok := e.(economysim.WorldCreated); !ok {
			events = append(events, e)
		}
	}
	snap := res.Final
	snap.ConfigHash, snap.Names, snap.Modules = [32]byte{}, economysim.Names{}, nil
	return events, snap
}

// TestInfrastructure_ModuleOnWorldRuns verifies, for the configuration of
// infrastructureConfig (all four kinds, three recipes with a clause, the transport,
// resources and storage modules it depends on) and three seeds, that a world with the
// module on is built, steps 40 ticks with the audit on, lists the module among the
// modules it runs, carries the hash of its configuration in WorldCreated,
// and gives the same events and the same final snapshot when run twice.
//
// Requirement: FR-CAP-001, FR-CAP-003
//
// Guard: a world that failed to build with the block present, or one whose run
// changed with the module's presence alone, would break every later step of the
// module.
func TestInfrastructure_ModuleOnWorldRuns(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	c := infrastructureConfig()
	want := economysim.HashConfig(c)
	for _, seed := range []uint64{1, 2, 3} {
		a, b := runTwice(t, c, seed)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("seed %d: two runs of the same world differ in their final snapshot", seed)
		}
		res := runPack(t, c, runSpec{Seed: seed, Ticks: 40, Drive: driveStep, Observe: true, Audit: true})
		created, ok := res.Events[0].(economysim.WorldCreated)
		if !ok || created.ConfigHash != want {
			t.Errorf("seed %d: first event = %T with hash %x, want WorldCreated with the hash of the configuration %x", seed, res.Events[0], created.ConfigHash, want)
		}
		if !slices.Contains(res.Final.Modules, "infrastructure") {
			t.Errorf("seed %d: Snapshot.Modules = %v, want infrastructure listed", seed, res.Final.Modules)
		}
	}
}

// runTwice runs the world of c twice for 40 ticks and returns the final snapshot of
// each, with the run's events compared inside.
func runTwice(t *testing.T, c economysim.Config, seed uint64) (economysim.Snapshot, economysim.Snapshot) {
	t.Helper()
	ea, a := quietRun(t, c, seed, 40)
	eb, b := quietRun(t, c, seed, 40)
	if !reflect.DeepEqual(ea, eb) {
		t.Errorf("seed %d: two runs of the same world emitted different events", seed)
	}
	return a, b
}

// withoutFacilityEvents drops the FacilityChanged events and counts them.
func withoutFacilityEvents(events []economysim.Event) ([]economysim.Event, int) {
	added := 0
	out := slices.DeleteFunc(events, func(e economysim.Event) bool {
		_, ok := e.(economysim.FacilityChanged)
		if ok {
			added++
		}
		return ok
	})
	return out, added
}

// TestInfrastructure_LoadedPackRunsLikeTheModuleOff is the end-to-end proof of the
// module's neutrality: the file form of the module is layered over pack one, loaded,
// validated, built into a world and run for 400 ticks with the audit on, and it gives
// the same events as pack one alone but for WorldCreated and the FacilityChanged
// events of the two initial mills, and the same final snapshot but for the hash, the
// names of the IDs, the module list and the facility rows, for two seeds, while its
// configuration hash differs. The overlay's recipe is listed by no role, so no start
// uses the mills; the step only decays them, which the facility rows alone show.
//
// Requirement: FR-CAP-001, FR-CAP-045, FR-CAP-003, FR-CAP-009
//
// Guard: a class or a row that leaked into a result (a draw, a ledger move, a gate)
// would move the golden of the reference chain; this test fails first.
func TestInfrastructure_LoadedPackRunsLikeTheModuleOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	off := loadValidPack(t, packOne...)
	on := loadPackBytes(t, []byte(infrastructureOverlay), packOne...)
	if on.Modules.Infrastructure == nil || len(on.Modules.Infrastructure.Classes) != 1 {
		t.Fatalf("loaded configuration .Modules.Infrastructure = %+v, want one class", on.Modules.Infrastructure)
	}
	if economysim.HashConfig(on) == economysim.HashConfig(off) {
		t.Error("HashConfig(pack one with the module) = HashConfig(pack one), want them to differ")
	}
	for _, seed := range []uint64{1, 2} {
		offEvents, offSnap := quietRun(t, off, seed, 400)
		onEvents, onSnap := quietRun(t, on, seed, 400)
		var added int
		onEvents, added = withoutFacilityEvents(onEvents)
		if added != 2 {
			t.Errorf("seed %d: %d FacilityChanged events, want 2 (the initial mills)", seed, added)
		}
		onSnap = withoutMillRow(t, seed, onSnap)
		if !reflect.DeepEqual(offEvents, onEvents) {
			t.Errorf("seed %d: the events of the pack with the module differ from those without it (%d and %d events)", seed, len(onEvents), len(offEvents))
		}
		if !reflect.DeepEqual(offSnap, onSnap) {
			t.Errorf("seed %d: the final snapshot of the pack with the module differs from the one without it", seed)
		}
	}
}

// withoutMillRow checks the one facility row of the overlay (two mills) and returns the
// snapshot without it.
func withoutMillRow(t *testing.T, seed uint64, s economysim.Snapshot) economysim.Snapshot {
	t.Helper()
	if len(s.Facilities) != 1 || s.Facilities[0].Facilities != 2 {
		t.Errorf("seed %d: Snapshot.Facilities = %+v, want one row of two mills", seed, s.Facilities)
	}
	s.Facilities = nil
	return s
}
