package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS101_InvariantsOverSeedsAndForkMidHire is EMP-S5 (spec-employment.md 17) and the
// harness test of NFR-EMP-005, over seeds 1 to 10 with the audit on, in a world whose jobs take
// 3 ticks and whose cooks, diggers and foresters follow a consumer's strategy that offers and
// posts, so that hires are in progress at the end of 60 ticks. Exact: the hires conform (the
// bill, the order of the events, the bill back whole whatever the cause that ended them), no
// hire has no bill, the escrow is the money of the hires in progress, the money moves only by the
// boundary flows, and a second run of the seed gives the same hires and the same final
// snapshot. A checkpoint and a fork at tick 3, with a hire in progress, give the original, the
// restore and the restore of the restore the same events, snapshot, audit and checkpoint bytes
// (FR-EMP-043). The causes that ended the hires are recorded. That the module takes no word is
// the other test of S101, which reads the PCG of seeds 1 to 10.
//
// Requirement: FR-EMP-043, NFR-EMP-005
//
// Guard: a hire lost by a checkpoint, or a wage that moved money outside the boundary flows,
// would appear only in a world with the module on, jobs of more than one tick and a hire in
// progress, which no off golden and no single seed would see.
func TestScenarioS101_InvariantsOverSeedsAndForkMidHire(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg, s := hireForkConfig(t)
	opts := []economysim.Option{economysim.WithStrategy("hire_pack", s)}
	hireScenario(t, "one-demand", "S101 one-demand+hire invariants over seeds with jobs of 3 ticks, 60 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		tape := newLabourTape()
		tape.anyCause = true
		r := runEmpTape(t, tape, cfg, seed, 60, nil, opts...)
		r.invariants(t, rep, "S101", "one-demand", seed, false)
		r2 := runEmp(t, cfg, seed, 60, nil, opts...)
		checkExact(t, rep, "S101", "one-demand", seed, "a second run of the seed gives the same hires and the same final snapshot",
			tape.formed == r2.tape.formed && reflect.DeepEqual(r.run.Final, r2.run.Final), "%d and %d hires formed", tape.formed, r2.tape.formed)
		checkExact(t, rep, "S101", "one-demand", seed, "hires are in progress at the end of the run", len(r.run.Final.Hires) > 0, "no hire open at tick 60")
		rep.num(seed, "hires formed", tape.formed)
		rep.num(seed, "hires open at the end", int64(len(r.run.Final.Hires)))
		for _, c := range []economysim.Cause{economysim.CauseCompleted, economysim.CauseWorkerRole, economysim.CauseWorkerLeft, economysim.CauseEmployerLeft, economysim.CauseEmployerRole, economysim.CauseNotStarted} {
			rep.num(seed, "hires ended, cause "+string(c), tape.byCause[c])
		}
	})
	t.Run("fork mid-hire", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				held := 0
				forkAtN(t, forkCase{
					Name: "S101 hire pack", Cfg: cfg, Seed: seed, N: 3, M: 8, Logs: true,
					Opts: func() []economysim.Option {
						return append([]economysim.Option{economysim.WithAudit()}, opts...)
					},
					Before: func(w *economysim.World) {
						snap, err := w.Snapshot()
						if err != nil {
							t.Fatalf("Snapshot() = %v, want nil", err)
						}
						held = len(snap.Hires)
					},
				})
				if held == 0 {
					t.Errorf("seed %d: no hire in progress at the boundary, want at least one", seed)
				}
			})
		}
	})
}
