package economysim_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// runSettlementsWorld builds a world from c with the seed, runs ticks steps and
// returns every event after WorldCreated (which carries the configuration hash and
// so differs by design) and the final snapshot with its hash zeroed.
func runSettlementsWorld(t *testing.T, c economysim.Config, seed uint64, ticks int) ([]economysim.Event, economysim.Snapshot) {
	t.Helper()
	rec := &recorder{}
	w, err := economysim.NewWorld(c, economysim.WithSeed(seed), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step(seed %d) error = %v, want nil", seed, err)
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot(seed %d) error = %v, want nil", seed, err)
	}
	snap.ConfigHash = [32]byte{}
	var events []economysim.Event
	for _, e := range rec.events {
		if _, created := e.(economysim.WorldCreated); !created {
			events = append(events, e)
		}
	}
	return events, snap
}

// TestScenario_SettlementsOnIsInert verifies that, in this change, a configuration
// with the module on (three classes, a flag, a limit, a gated grower and refiner, a
// region row) builds a world, runs 40 ticks and gives the same events and the same
// final snapshot as the same world with the module off, for three seeds. The module
// is data only until its review is built; a world that already reacted to a class
// would move the golden of the reference chain.
//
// Requirement: FR-SET-001, FR-SET-015
//
// Guard: PR A adds a configuration surface and no behaviour. If a later change makes
// the module do something, this test must be replaced by its behaviour tests; until
// then it fails if the surface leaks into a result (a gate, a draw, an event).
func TestScenario_SettlementsOnIsInert(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	on := settlementsConfig()
	if err := on.Validate(); err != nil {
		t.Fatalf("Validate(settlementsConfig) = %v, want nil", err)
	}
	for _, seed := range []uint64{1, 2, 3} {
		wantEvents, wantSnap := runSettlementsWorld(t, referenceConfig(), seed, 40)
		gotEvents, gotSnap := runSettlementsWorld(t, on, seed, 40)
		if len(wantEvents) == 0 {
			t.Fatalf("seed %d: the off world emitted no event, want a run that means something", seed)
		}
		if !reflect.DeepEqual(gotEvents, wantEvents) {
			t.Errorf("seed %d: events with the module on differ from the module off (%d against %d events)", seed, len(gotEvents), len(wantEvents))
		}
		if !reflect.DeepEqual(gotSnap, wantSnap) {
			t.Errorf("seed %d: final snapshot with the module on differs from the module off", seed)
		}
	}
}

// TestConfigOnly_SettlementsHashDiffersFromOff verifies that the world reports the
// hash of the configuration with the module, which differs from the same world with
// the module off, so a checkpoint of one is refused by the other.
//
// Requirement: FR-SET-030
func TestConfigOnly_SettlementsHashDiffersFromOff(t *testing.T) {
	t.Parallel()
	off, err := economysim.NewWorld(referenceConfig())
	if err != nil {
		t.Fatalf("NewWorld(off) error = %v, want nil", err)
	}
	on, err := economysim.NewWorld(settlementsConfig())
	if err != nil {
		t.Fatalf("NewWorld(on) error = %v, want nil", err)
	}
	if on.ConfigHash() == off.ConfigHash() {
		t.Errorf("ConfigHash with the module on = %x, the same as off, want different", on.ConfigHash())
	}
}
