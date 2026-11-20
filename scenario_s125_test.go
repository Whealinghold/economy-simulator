package economysim_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// capTail is the ticks that S125 runs after the checkpoint.
const capTail = 30

// TestScenarioS125_Checkpoint verifies CAP-S7 of spec-infrastructure.md 16.2: pack one with
// overlay/infrastructure.toml (60 initial mills decaying) and a directed cook that was told to
// build a mill (3 ticks) one tick ago, so the checkpoint is taken with the job in progress. The
// world restored from it (RestoreWorld with the audit and an observer) and the original run 30
// more ticks: the events of the tail, the snapshot, the audit (every Step) and the checkpoint
// bytes are the same, and the built mill completes in both. The case of a store facility
// pending is proved at the unit level (TestFork_Infrastructure_Build, TestStoreFacility_Pending):
// no pack of the reference chain gives a household that cannot take the store, and an owner
// that cannot is not reached by the public calls.
//
// Requirement: FR-CAP-041
//
// Guard: a record, a table or a job that the checkpoint did not carry would make the
// continuation differ from the uninterrupted run, and only a facility in the middle of its
// making can show it.
func TestScenarioS125_Checkpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	cfg := loadValidPack(t, packOneInfrastructure...)
	recA, recB := &recorder{}, &recorder{}
	a := mustWorld(t, cfg, economysim.WithSeed(3), economysim.WithAudit(), economysim.WithObserver(recA))
	cook := capAddCook(t, a, cfg, 0)
	capDirect(t, a, cook, capRecipe(cfg, "build_mill"), 1)
	capStep(t, a)
	ck := mustCheckpoint(t, a)
	ckBytes := mustMarshal(t, ck)
	b, err := economysim.RestoreWorld(cfg, ck, economysim.WithAudit(), economysim.WithObserver(recB))
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if !bytes.Equal(ckBytes, mustMarshal(t, mustCheckpoint(t, b))) {
		t.Fatalf("the restored world's checkpoint differs from the one it was restored from")
	}
	mark := len(recA.events)
	for range capTail {
		capStep(t, a)
		capStep(t, b)
	}
	if x, y := fmt.Sprintf("%+v", recA.events[mark:]), fmt.Sprintf("%+v", recB.events[1:]); x != y { // event 0 of the restored world is WorldRestored
		t.Errorf("the events of the tail differ between the original and the restored world")
	}
	capSameState(t, a, b)
	if n := capKinds(recB.events)["built/recipe"]; n != 1 {
		t.Errorf("the restored world completed %d builds in the tail, want 1 (the job was in progress)", n)
	}
}

// capSameState fails t unless the two worlds have equal snapshots and byte-equal checkpoints.
func capSameState(t *testing.T, a, b *economysim.World) {
	t.Helper()
	sa, errA := a.Snapshot()
	sb, errB := b.Snapshot()
	if errA != nil || errB != nil {
		t.Fatalf("Snapshot() errors = %v, %v, want nil", errA, errB)
	}
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("the snapshots differ after %d ticks", capTail)
	}
	if !bytes.Equal(mustMarshal(t, mustCheckpoint(t, a)), mustMarshal(t, mustCheckpoint(t, b))) {
		t.Errorf("the checkpoints differ after %d ticks", capTail)
	}
}
