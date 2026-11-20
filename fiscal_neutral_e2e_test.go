package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalEmptyRun steps cfg for ticks under the audit with an observer and returns the events and
// the final snapshot.
func fiscalEmptyRun(t *testing.T, cfg economysim.Config, seed uint64, ticks int, opts ...economysim.Option) ([]economysim.Event, economysim.Snapshot) {
	t.Helper()
	rec := &recorder{}
	w := mustWorld(t, cfg, append(opts, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))...)
	stepBoth(t, ticks, w)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	// What tells the two worlds apart is the configuration: its hash, in WorldCreated and
	// in the snapshot, and the list of modules that are on, in the snapshot.
	s.ConfigHash, s.Modules = [32]byte{}, nil
	events := slices.DeleteFunc(rec.events, func(e economysim.Event) bool { _, ok := e.(economysim.WorldCreated); return ok })
	return events, s
}

// TestFiscal_PackWithoutDutiesOrTollsLeavesTheWorldAlone verifies, end to end, that a world
// with the fiscal module on and no flow configured is the world with no fiscal module: the
// same events, in the same order, and the same snapshot, on the governed warm five (where
// the module would act at P4.1a) and on pack two with the scripted shipper (where it would
// act at dispatch, and every ShipmentDispatched reports a Toll of 0).
//
// Requirement: FR-FIS-001, FR-FIS-005, NFR-FIS-005
//
// Guard: the hooks of the clearing and of the dispatch are guarded calls; one that took
// money, drew from the stream or emitted an event with no flow configured would move
// every world that turns the module on before it has a rule, and the off goldens would
// not see it.
func TestFiscal_PackWithoutDutiesOrTollsLeavesTheWorldAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 1, 30
	t.Run("governed warm five", func(t *testing.T) {
		t.Parallel()
		off := loadValidPack(t, fivePolityGoverned...)
		on := off
		on.Modules.Fiscal = &economysim.Fiscal{}
		evOff, sOff := fiscalEmptyRun(t, off, seed, ticks)
		evOn, sOn := fiscalEmptyRun(t, on, seed, ticks)
		if !reflect.DeepEqual(evOff, evOn) {
			t.Errorf("events with an empty fiscal module (%d) differ from those with none (%d) (seed %d), want equal", len(evOn), len(evOff), seed)
		}
		if !reflect.DeepEqual(sOff, sOn) {
			t.Errorf("snapshot with an empty fiscal module differs from the one with none (seed %d), want equal", seed)
		}
	})
	t.Run("pack two with a shipper", func(t *testing.T) {
		t.Parallel()
		off, opts := tollPack(t, fiscalTollDoc(3))
		off.Modules.Fiscal = nil
		on := off
		on.Modules.Fiscal = &economysim.Fiscal{}
		evOff, sOff := fiscalEmptyRun(t, off, seed, ticks, opts...)
		evOn, sOn := fiscalEmptyRun(t, on, seed, ticks, opts...)
		var shipped int
		for _, e := range evOn {
			if d, ok := e.(economysim.ShipmentDispatched); ok {
				shipped++
				if d.Toll != 0 {
					t.Errorf("ShipmentDispatched %+v has a Toll with no toll configured (seed %d), want 0", d, seed)
				}
			}
		}
		if shipped == 0 {
			t.Errorf("shipments dispatched over %d ticks (seed %d) = 0, want some: the test would prove nothing", ticks, seed)
		}
		if !reflect.DeepEqual(evOff, evOn) || !reflect.DeepEqual(sOff, sOn) {
			t.Errorf("run with an empty fiscal module differs from the one with none (seed %d): %d and %d events, want equal", seed, len(evOn), len(evOff))
		}
	})
}
