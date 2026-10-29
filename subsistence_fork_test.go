package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestFork_Subsistence verifies that a world forked at a boundary with households
// in the own-use mode runs on exactly as an uninterrupted one in every event, in
// the snapshot, in the checkpoint bytes and in the log records (the Warns of the
// two runs): at tick 0, inside an own-use job of three ticks, five ticks into a
// run of the Warn "role on own use" (whose tenth tick falls after the boundary,
// so the saved run decides the log line), and after a call between ticks.
//
// Setup: 12 households start with no grain and enter the mode at the first flag
// step. ExitTicks is 40, which the cover of a household that makes exactly its
// upkeep never reaches, so they stay in the mode for the whole fork and start a
// job every three ticks.
//
// Requirement: FR-SUB-044
func TestFork_Subsistence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := forageWorld(forageSpec{Households: 12, OwnUseBP: 10_000, Duration: 3, Module: true, RetryJobs: 10, ExitTick: 40})
	for _, c := range []struct {
		name   string
		n      int
		before func(*economysim.World)
	}{
		{"tick 0", 0, nil},
		{"inside a job", 1, nil},
		{"inside a run of the Warn", 5, nil},
		{"after a call between ticks", 12, func(w *economysim.World) {
			if err := economysim.SetYieldShockForTest(w, 0, 1, 0); err != nil {
				t.Fatalf("SetYieldShockForTest() error = %v, want nil", err)
			}
		}},
	} {
		forkAtN(t, forkCase{Name: "forage " + c.name, Cfg: cfg, Seed: 3, N: c.n, M: forkM, Before: c.before, Logs: true,
			Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} },
			Inspect: func(t *testing.T, restored, since []economysim.Event) {
				changed := 0
				for _, e := range since {
					if _, ok := e.(economysim.OwnUseChanged); ok {
						changed++
					}
				}
				if changed == 0 {
					t.Errorf("%s: no OwnUseChanged after the boundary, want some so the fork carries the mode", c.name)
				}
			}})
	}
}
