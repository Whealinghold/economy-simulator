package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// demandRun builds a world from cfg with a recorder, steps it ticks times with
// the audit on and returns the events after construction (the WorldCreated and
// the AgentSpawned of the initial agents left out) and the final snapshot.
func demandRun(t *testing.T, cfg economysim.Config, seed uint64, ticks int) ([]economysim.Event, economysim.Snapshot) {
	t.Helper()
	rec := &recorder{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("seed %d: NewWorld error = %v, want nil", seed, err)
	}
	rec.forget()
	for i := range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step at tick %d error = %v, want nil", seed, i, err)
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("seed %d: Snapshot error = %v, want nil", seed, err)
	}
	return rec.events, snap
}

// TestDemand_NoTasteIsInert verifies that the reference chain with the demand
// module on and no taste runs, tick for tick, exactly as it does with the module
// off: the same events and the same markets, regions and roles in the snapshot.
// With no taste no role has a ladder, so no filler, checker or consumption has
// anything to do; all the module may change is its own announcement (the Modules
// and Names rows and the want_consumption boundary rows).
//
// Requirement: FR-DEM-001, FR-DEM-004
//
// Guard: a module that is on but has no taste is a world with no ladder; a want
// bid, a want_consumption posting or a changed draw order there would be a leak of
// the module into a world that did not ask for it.
func TestDemand_NoTasteIsInert(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	const ticks = 40
	noTastes := referenceConfig()
	withDemand(&noTastes)
	noTastes.Modules.Demand.Tastes, noTastes.Modules.Demand.RegionTastes = nil, nil
	for seed := uint64(1); seed <= uint64(invariantSeeds(3)); seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			offEv, offSnap := demandRun(t, referenceConfig(), seed, ticks)
			onEv, onSnap := demandRun(t, noTastes, seed, ticks)
			if !reflect.DeepEqual(offEv, onEv) {
				t.Errorf("seed %d: events with the module on and no taste differ from the module off (%d and %d events)", seed, len(onEv), len(offEv))
			}
			// The announcement rows are the module's own; everything else must agree.
			if !reflect.DeepEqual(offSnap.Markets, onSnap.Markets) || !reflect.DeepEqual(offSnap.Regions, onSnap.Regions) || !reflect.DeepEqual(offSnap.Roles, onSnap.Roles) {
				t.Errorf("seed %d: markets, regions or roles of the snapshot differ with the module on and no taste", seed)
			}
		})
	}
}

// TestDemand_OneDemandIsDeterministic verifies that the pack one-demand, with
// the module on, gives the same events and the same snapshot on two runs of one
// seed and a different run for another seed, so the module's presence does not
// add a source of nondeterminism (architecture.md 7.4).
//
// Requirement: FR-DEM-040
func TestDemand_OneDemandIsDeterministic(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemand...)
	aEv, aSnap := demandRun(t, cfg, 1, 60)
	bEv, bSnap := demandRun(t, cfg, 1, 60)
	if !reflect.DeepEqual(aEv, bEv) || !reflect.DeepEqual(aSnap, bSnap) {
		t.Errorf("two runs of one-demand at seed 1 differ: %d and %d events", len(aEv), len(bEv))
	}
	cEv, _ := demandRun(t, cfg, 2, 60)
	if reflect.DeepEqual(aEv, cEv) {
		t.Errorf("one-demand at seeds 1 and 2 gave the same %d events, want different runs", len(aEv))
	}
}
