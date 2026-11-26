package economysim_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// savingMidBuildTick returns the tick after the last request of a build that a run of cfg at
// seed makes in 400 ticks, so that a checkpoint taken there holds households with a project job in
// progress, with inputs held and others saving; ok is false when the run makes no request.
func savingMidBuildTick(t testing.TB, cfg economysim.Config, seed uint64) (int64, bool) {
	t.Helper()
	res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true})
	var last int64 = -1
	for _, e := range res.Events {
		if v, ok := e.(economysim.SavingPlanned); ok && v.Builds > 0 {
			last = v.Tick
		}
	}
	return last + 1, last >= 0
}

// savingQueries returns Saving(agent) of every agent of w, rendered, in agent order, and the
// number of agents with a project job in progress.
func savingQueries(t testing.TB, w *economysim.World) (rows []string, building int) {
	t.Helper()
	for id := economysim.AgentID(0); id < savingAgentBound; id++ {
		s, err := w.Saving(id)
		if errors.Is(err, economysim.ErrUnknownAgent) {
			continue // the first IDs are the regions' estates, and a removed agent's is not reused
		}
		rows = append(rows, fmt.Sprintf("%d %+v %v", id, s, err))
		if err == nil && s.Building {
			building++
		}
	}
	if _, err := w.Saving(savingAgentBound); !errors.Is(err, economysim.ErrUnknownAgent) {
		t.Errorf("Saving(%d) error = %v, want ErrUnknownAgent: savingAgentBound is not above every agent", savingAgentBound, err)
	}
	return rows, building
}

// savingAgentBound is above the ID of every agent of pack one over the scenario's 600 ticks.
const savingAgentBound = 2000

// TestScenarioS147_CheckpointWhileSavingAndMidBuild reports SAV-S6 of spec-saving-and-investment.md
// 14.3, a harness test: a checkpoint taken with households saving, holding inputs and in the middle
// of a build, restored and run, equals the uninterrupted run, the plan and the snapshot rows
// included (FR-SAV-040). For seeds 1 to 3 the world runs to the tick after the last build request,
// is checkpointed and restored, and for 30 ticks both give the same snapshot and the same query
// Saving for every agent; at the boundary some household has a project job in progress.
//
// Arithmetic: the module keeps no state, so everything it reports is a function of the saved
// state of the other modules (spec 13); the build job in progress is in the production state and
// the facility is made at its completion, which both worlds reach in the 30 ticks.
//
// Requirement: FR-SAV-026, FR-SAV-037, FR-SAV-040
//
// Guard: a table, scratch or cached choice of the module that the restore did not rebuild would
// make the restored world plan, ask or answer differently from the uninterrupted one.
func TestScenarioS147_CheckpointWhileSavingAndMidBuild(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSaving...)
	savingScenario(t, "S147 one+saving checkpoint mid-build, 30 ticks after", min(3, scenarioSeedCount()), func(t *testing.T, rep *scenReport, seed uint64) {
		n, ok := savingMidBuildTick(t, cfg, seed)
		checkExact(t, rep, "S147", "one", seed, "the run makes a build request", ok, "no SavingPlanned with a build request in 400 ticks")
		if !ok {
			return
		}
		a := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit())
		for a.Tick() < n {
			capStep(t, a)
		}
		ck := mustCheckpoint(t, a)
		b, err := economysim.RestoreWorld(cfg, ck, economysim.WithAudit())
		if err != nil {
			t.Fatalf("seed %d: RestoreWorld() error = %v, want nil", seed, err)
		}
		before, building := savingQueries(t, a)
		after, _ := savingQueries(t, b)
		checkExact(t, rep, "S147", "one", seed, "a job is in progress at the boundary", building > 0, "no agent has Building set at tick %d", n)
		checkExact(t, rep, "S147", "one", seed, "the restored queries equal the original's", reflect.DeepEqual(before, after), "Saving() differs at the boundary")
		same := true
		for range 30 {
			capStep(t, a)
			capStep(t, b)
			sa, errA := a.Snapshot()
			sb, errB := b.Snapshot()
			if errA != nil || errB != nil {
				t.Fatalf("seed %d: Snapshot() = %v, %v, want nil", seed, errA, errB)
			}
			qa, _ := savingQueries(t, a)
			qb, _ := savingQueries(t, b)
			same = same && reflect.DeepEqual(sa, sb) && reflect.DeepEqual(qa, qb)
		}
		checkExact(t, rep, "S147", "one", seed, "the snapshots and queries agree for 30 ticks", same, "the restored run differs from the uninterrupted one")
		rep.num(seed, "boundary tick", n)
		rep.num(seed, "households building at the boundary", int64(building))
	})
}
