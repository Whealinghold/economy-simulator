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

// TestDemand_OnHasNoWorldEffectYet verifies that the reference chain with the
// demand module on and no taste runs, tick for tick, exactly as it does with the
// module off: the same events and the same markets, regions and roles in the
// snapshot, and a clean audit. With no taste no role has a ladder, and the module
// has no strategy, production or consumption hook in this slice, so all it may
// change is its own announcement (the Modules and Names rows and the
// want_consumption boundary rows). With the tastes of demandConfig the want items
// join the tradable set of the roles that like them (FR-AGT-003, FR-AGT-004), which
// does change what a role change lists to the estate; that run is held to the
// audit and to a zero want_consumption balance only.
//
// Requirement: FR-DEM-001, FR-DEM-004, FR-LED-050
//
// Guard: the world side of the module is the next PR. Until it lands, a want
// bid, a want_consumption posting or a changed draw order would be a leak of the
// module into a world that cannot yet be asked for it. The successor tests of the
// world side (spec-demand.md FR-DEM-019 to 025) replace this one.
func TestDemand_OnHasNoWorldEffectYet(t *testing.T) {
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
			// The tastes of demandConfig: a valid, audited world that posts no want_consumption.
			_, tasteSnap := demandRun(t, demandConfig(), seed, ticks)
			for _, r := range tasteSnap.Boundary {
				if r.Reason == economysim.ReasonWantConsumption && r.Balance != 0 {
					t.Errorf("seed %d: want_consumption row %+v has a balance, want 0 while the module posts nothing", seed, r)
				}
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

// TestFork_DemandOn verifies that a world with the demand module on survives a
// checkpoint: restored at tick N (at the start and inside the run), it and a
// restore of its restore give the same events, snapshots and checkpoint bytes as
// the world that never stopped. The module's state is derived of the
// configuration in this slice, so the restore must rebuild it and the
// configuration hash must carry the module.
//
// Requirement: FR-DEM-034
//
// Guard: the carries and the choke runs are saved in the checkpoint tag 12 of the
// next PR; until then a restore that lost the module (a hash without it, a world
// built without its state) would pass every other fork test, which run with it off.
func TestFork_DemandOn(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemand...)
	for _, n := range []int{0, 7} {
		t.Run(fmt.Sprintf("N %d", n), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{Name: "one-demand", Cfg: cfg, Seed: 1, N: n, M: 10})
		})
	}
}
