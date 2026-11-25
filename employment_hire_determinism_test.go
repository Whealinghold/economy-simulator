package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestDeterminism_EmploymentHire verifies that the pack one-demand with the hire form on gives the same
// final snapshot, the same state hash and the same events whatever the way it is driven and
// observed: a second run, Run in chunks, a logger, a nil observer, the audit (detCases, the cases
// of the race run when under -race), for seeds 1 and 2, and that the two seeds end differently. The
// default strategy offers and posts, so thousands of hires form in each run.
//
// Requirement: FR-EMP-018, FR-EMP-061, NFR-EMP-004
//
// Guard: the round and the report read state and draw no word; a logger, an observer or the audit
// that changed what the module counted would change the run, and only a run with hires in it can
// show it.
func TestDeterminism_EmploymentHire(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandHire...)
	cases := detCases(true)
	seeds := seedList(2)
	finals := make([]economysim.Snapshot, len(seeds))
	for i, seed := range seeds {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
			finals[i] = base.Final
			ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s", seed, d)
			}
			for _, tc := range cases {
				spec := tc.spec()
				spec.Seed, spec.Ticks = seed, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
				}
				if spec.Observe {
					if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
						t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
	if len(seeds) > 1 && reflect.DeepEqual(finals[0], finals[1]) {
		t.Errorf("seeds %d and %d end in the same snapshot, want different runs", seeds[0], seeds[1])
	}
}

// TestFork_EmploymentScenarios verifies FR-EMP-043 on the pack of the scenarios, with the default
// strategy: a world with the hire form on, forked at tick 5 (the first hires have formed) and at tick
// 150 (the money has circulated), runs on as an uninterrupted one in every event, in the snapshot, in
// the checkpoint bytes and in the log records, and so does the restore of the restore. The jobs of
// the pack take one tick, so no hire is open at a boundary; the fork with a hire in progress is
// TestFork_EmploymentHire and S101.
//
// Requirement: FR-EMP-043, FR-EMP-044
//
// Guard: the wage counters of the tick and the Warn runs of the round are state; a restore that
// dropped them would differ at the next report.
func TestFork_EmploymentScenarios(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandHire...)
	for _, n := range []int{5, 150} {
		t.Run(fmt.Sprintf("N %d", n), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "hire pack with the default strategy", Cfg: cfg, Seed: 3, N: n, M: 20, Logs: true,
				Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} },
			})
		})
	}
}
