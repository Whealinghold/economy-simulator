package economysim_test

import (
	"log/slog"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// loseYield is a layer over pack one with the skill overlay that cuts the trainee
// yield of the one eligible recipe of the cook to 1 basis point, so that the gain of a
// new place is negative and the applicants wait: the Warn "no places open" runs.
//
// Arithmetic: prepare_basic makes 4 prepared (628 each at the seeds of pack one)
// at a yield of 1 over 10 000, which is 0, against inputs of 2 staple and 1 fuel
// (2 * 96 + 179 = 371): the output is -371 and the gain below 0 whatever the wage.
const loseYield = "[recipe.prepare_basic]\ntrainee_yield = 1\n"

// TestFork_Skills verifies that a world forked inside the traineeships runs on exactly as
// an uninterrupted one in every event, in the snapshot (the places, the queue and the
// trainee rows), in the checkpoint bytes and in the log records, at the ticks that
// matter: before the first step, with applicants queued and no place yet filled, in
// the middle of the courses, just before and just after the first completions, with a
// queue at tick 150, and, in a second pack whose gain is negative, inside a run of the
// Warn "no places open" whose tenth tick falls after the boundary, so the saved run
// decides the log line.
//
// Setup: pack one, warm, with overlay/skill.toml, seed 1 (probed: queue of 4 at the
// fifth step, 4 trainees placed at tick 5 and completing at tick 125, 192 trainees
// at tick 80, a queue of 12 at tick 150), M = 30 ticks after the boundary.
//
// Requirement: FR-SKL-027, FR-CHK-050, FR-SKL-022
func TestFork_Skills(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	opts := func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }
	traineeship := func(name string) func(t *testing.T, restored, since []economysim.Event) {
		return func(t *testing.T, restored, since []economysim.Event) {
			n := 0
			for _, e := range since {
				switch e.(type) {
				case economysim.TraineeshipStarted, economysim.TraineeshipCompleted, economysim.TraineeshipEnded:
					n++
				}
			}
			if n == 0 {
				t.Errorf("%s: no Traineeship event after the boundary, want some so the fork carries the traineeships", name)
			}
		}
	}
	for _, c := range []struct {
		name string
		n    int
	}{
		{"before the first step", 0},
		{"applicants queued", 5},
		{"inside the first courses", 60},
		{"just before the first completions", 125},
		{"just after the first completions", 126},
		{"with a queue", 150},
	} {
		forkAtN(t, forkCase{Name: "skills " + c.name, Cfg: cfg, Seed: 1, N: c.n, M: forkM, Logs: true, Opts: opts, Inspect: traineeship(c.name)})
	}
	lose := loadPackBytes(t, []byte(loseYield), packOneSkills...)
	forkAtN(t, forkCase{Name: "skills inside a run of the Warn", Cfg: lose, Seed: 1, N: 12, M: forkM, Logs: true, Opts: opts})
}

// TestFork_SkillsWarnRunsAfterTheBoundary verifies the design of the last cell of
// TestFork_Skills: with the yield cut to 1 basis point the world logs the Warn "no
// places open" at the tenth consecutive tick after the first application and again ten
// ticks after, and nothing of the module above Warn, so a fork at tick 12, inside the
// run, is decided by the saved run.
//
// Requirement: FR-SKL-022
//
// Guard: without the Warn in the window after the boundary the cell would compare
// two runs that never log it, and the saved run counter would be untested.
func TestFork_SkillsWarnRunsAfterTheBoundary(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(loseYield), packOneSkills...)
	h := newCaptureHandler()
	w := newPackWorld(t, cfg, 1, economysim.WithLogger(slog.New(h)))
	var warns []int
	for i := range 42 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", i, err)
		}
		for _, l := range h.sink.lines {
			if l.msg == "no places open" {
				warns = append(warns, i)
			}
		}
		h.sink.lines = nil
	}
	if len(warns) < 2 || warns[0] > 12+forkM || warns[0] <= 12 {
		t.Errorf("Warn \"no places open\" at ticks %v, want the first after tick 12 (inside the window of the fork) and a second ten ticks after", warns)
	}
	if len(warns) >= 2 && warns[1]-warns[0] != 10 {
		t.Errorf("Warns at ticks %v, want them ten ticks apart", warns)
	}
}
