package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s58ChangePrivation is the privation at which rung 1 of the failure ladder acts, in
// the scenario: the three durations 1, 3 and 8 have Duration - 1 of 0, 2 and 7, below,
// below and above it.
const s58ChangePrivation int64 = 3

// s58Run is what S58 keeps of one run: the largest privation a forager reached before
// its first own-use output.
type s58Run struct {
	firstOutput, peakPrivation int64
	warned                     bool
}

// runS58 runs the forage world with an own-use recipe of the given duration through
// the outage and returns the measures of S58.
func runS58(t testing.TB, seed uint64, duration int64) s58Run {
	t.Helper()
	cfg := forageWorld(forageSpec{Households: 20, StartGrain: 6, OwnUseBP: 10_000, Duration: duration, Module: true, RetryJobs: 10, ChangePrivation: s58ChangePrivation})
	rec := &recorder{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	script, run := outage(t), s58Run{firstOutput: -1}
	byTick := map[int64]int64{} // the largest privation after each tick
	for range 40 {
		applyScript(t, w, s0.Names, script)
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() error = %v, want nil", seed, err)
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		for _, r := range s.Roles {
			byTick[w.Tick()-1] = max(byTick[w.Tick()-1], r.MaxPrivation)
		}
	}
	for _, e := range rec.events {
		if o, ok := e.(economysim.OwnUseChanged); ok && len(o.Output) > 0 && o.Tick >= 10 && (run.firstOutput < 0 || o.Tick < run.firstOutput) {
			run.firstOutput = o.Tick
		}
	}
	for tick := int64(10); tick <= run.firstOutput; tick++ {
		run.peakPrivation = max(run.peakPrivation, byTick[tick])
	}
	for _, wn := range cfg.Warnings() {
		run.warned = run.warned || wn.Rule == "own_use_slower_than_ladder"
	}
	return run
}

// TestScenarioS58_LadderAgainstDuration runs scenario S58 (SUB-S3, plan 2-10 D417) on
// the forage world with own-use recipes of duration 1, 3 and 8 against ChangePrivation
// 3, seeds 1 to scenarioSeedCount(), with the audit on. For each duration it
// measures the largest privation of the households before the first own-use output
// of the outage. It asserts that the ladder can reach the running job exactly when
// Duration - 1 reaches ChangePrivation (the duration 8: a privation of 6, which is
// at least 3; the durations 1 and 3: 0 and 1, which are not) and that the Warn
// own_use_slower_than_ladder is on the configuration of exactly those. It reports the
// measure. The outcomes are written to
// testdata/chains/expected/subsistence.scenarios.txt with -update.
//
// Setup: the forage world has one role, so rung 1 has no role to move a household
// to and no job is abandoned; the scenario measures the privation, which is what
// decides whether rung 1 can abandon a job. A second role moves households through
// the review and its cooldown as well, and a household that has been through
// several cycles carries the leaky privation of the earlier ones, so the duration 3
// abandons jobs in such a world; that is not what the model's condition says and
// is an open item of the report.
//
// Requirement: FR-SUB-032, FR-SUB-038
//
// Guard: a ladder that ignores a running own-use job, or a warning that fires for
// a recipe the ladder does not reach, fails here.
func TestScenarioS58_LadderAgainstDuration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S58", "forage"
	rep := newPackScenReport("subsistence", fmt.Sprintf("S58 forage ladder seeds 1-%d, ChangePrivation %d", scenarioSeedCount(), s58ChangePrivation))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				for _, d := range []int64{1, 3, 8} {
					r := runS58(t, seed, d)
					rep.num(seed, fmt.Sprintf("duration %d peak privation before the first output", d), r.peakPrivation)
					reaches := d-1 >= s58ChangePrivation
					checkExact(t, rep, scenario, pack, seed, "rung 1 reaches the job exactly when Duration - 1 reaches ChangePrivation",
						(r.peakPrivation >= s58ChangePrivation) == reaches,
						"duration %d: peak privation %d, want rung 1 reached = %v", d, r.peakPrivation, reaches)
					checkExact(t, rep, scenario, pack, seed, "the warning fires for exactly those", r.warned == reaches,
						"duration %d: warning = %v, want %v", d, r.warned, reaches)
				}
			})
		}
	})
	rep.flush(t)
}
