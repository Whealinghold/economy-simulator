package economysim_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

// TestScenarioS119_InertIsOff verifies CAP-S1 of spec-infrastructure.md 16.2 end to end: pack
// one with an infrastructure module that carries no class and no initial row against pack one
// without it, seeds 1 to the seed count, 400 ticks, Step loop with the observer and the audit.
// The module present and empty draws no word and moves nothing: every event after WorldCreated,
// every sampled snapshot (the last of them the final one) equal the run without it, except what the
// module owns (the configuration hash, Snapshot.Modules and the names of its classes), and it emits no facility event and
// no snapshot row.
//
// Requirement: FR-CAP-001, FR-CAP-002
//
// Guard: a module that is on but idle must leave no trace in a run; a draw, an event or a
// changed tie-break would move every later result of a world that merely listed the module.
func TestScenarioS119_InertIsOff(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	off := loadValidPack(t, packOne...)
	idle := loadPackBytes(t, []byte("[modules.infrastructure]\n"), packOne...)
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			spec := runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample}
			a, b := runPack(t, off, spec), runPack(t, idle, spec)
			if len(a.Events) != len(b.Events) {
				t.Fatalf("seed %d: %d events with the module idle, %d without, want equal", seed, len(b.Events), len(a.Events))
			}
			for i := 1; i < len(a.Events); i++ { // event 0 is WorldCreated: it carries the hash
				if x, y := fmt.Sprintf("%+v", a.Events[i]), fmt.Sprintf("%+v", b.Events[i]); x != y {
					t.Fatalf("seed %d: event %d is %s with the module idle, %s without, want equal", seed, i, y, x)
				}
			}
			if len(a.Samples) != len(b.Samples) {
				t.Fatalf("seed %d: %d sampled snapshots with the module idle, %d without, want equal", seed, len(b.Samples), len(a.Samples))
			}
			for tick, sa := range a.Samples {
				sb := b.Samples[tick]
				if len(sb.Facilities) != 0 {
					t.Errorf("seed %d: the snapshot after tick %d has %d facility rows, want none", seed, tick, len(sb.Facilities))
				}
				sb.Modules, sb.ConfigHash, sb.Names = sa.Modules, sa.ConfigHash, sa.Names
				if !reflect.DeepEqual(sa, sb) {
					t.Errorf("seed %d: the snapshot after tick %d differs with the module idle", seed, tick)
				}
			}
		})
	}
}
