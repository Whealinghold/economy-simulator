package economysim_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// skillRun is what a run of a pack with the skill overlay counted.
type skillRun struct {
	started, completed, ended int
	snapshot                  economysim.Snapshot
}

// runSkillPack runs the pack with overlay/skill.toml for ticks with the audit, so
// every identity of the ledger is checked at each tick, and counts the events of
// the traineeships.
func runSkillPack(t *testing.T, pack string, ticks int, seed uint64) skillRun {
	t.Helper()
	cfg := loadValidPack(t, append(append([]string{}, packLayers[pack]...), "overlay/skill.toml")...)
	var r skillRun
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		switch e.(type) {
		case economysim.TraineeshipStarted:
			r.started++
		case economysim.TraineeshipCompleted:
			r.completed++
		case economysim.TraineeshipEnded:
			r.ended++
		}
	})
	w, err := economysim.NewWorld(cfg, economysim.WithAudit(), economysim.WithSeed(seed), economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("seed %d: NewWorld(%s + skill) error = %v, want nil", seed, pack, err)
	}
	step(t, w, ticks)
	if r.snapshot, err = w.Snapshot(); err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
	}
	return r
}

// TestScenario_SkillOverlayRunsWithAudit verifies the exit check of plan 3-skl
// Phase 4: the pack one with the traineeship overlay (120 ticks of training on
// cook and crafter_a) runs 200 ticks with the audit on, so every identity of the
// ledger holds at each tick whatever the default spawner and the queue did, the
// default spawner applies and places trainees (the module is not inert), and two
// runs of one seed end in the same snapshot.
//
// Requirement: FR-SKL-010, FR-SKL-016, FR-SKL-017, FR-SKL-027
func TestScenario_SkillOverlayRunsWithAudit(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	a, b := runSkillPack(t, "one", 200, 7), runSkillPack(t, "one", 200, 7)
	if a.started == 0 {
		t.Errorf("seed 7: no traineeship started in 200 ticks, want the default spawner to apply and the queue to place")
	}
	if !reflect.DeepEqual(a.snapshot, b.snapshot) || a.started != b.started {
		t.Errorf("seed 7: two runs ended in different snapshots or counts (%d, %d started), want the same", a.started, b.started)
	}
}
