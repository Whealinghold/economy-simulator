package economysim

import (
	"slices"
	"testing"
)

// TestScenarioS99_EmployerAndWorkerRemovedMidJob is EMP-S3 (plan-3-emp.md Phase 8, built
// with PR 4 because it is the exit check of the interruptions): two farms each hire a
// hand for a job of 3 ticks; after the first working tick the worker of the first hire
// is removed and the employer of the second is removed. Exact, for seeds 1 to 10 with
// the audit on at every tick: the money of the world is the same before the removals and at
// every tick after them; the remainder of the escrow (140) returns to the
// employer's own account, so the first farm holds 10 000 - 210 + 140 and the estate
// receives the second farm's 10 000 - 210 + 140; the surviving worker keeps its wage
// (70) and has no job; no Escrow money is left; the hires end with worker_left and
// employer_left; the events of each removal come in the order of 6.5.2.
//
// Requirement: FR-EMP-031, FR-EMP-034, FR-EMP-036, FR-EMP-038
func TestScenarioS99_EmployerAndWorkerRemovedMidJob(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 10; seed++ {
		log := &hireLog{}
		w := hireWorld(t, defaultHire(hireOpts{farms: 2, hands: 2, handRecipe: true}), farmWants(1, 100, 40), WithObserver(log), WithSeed(seed))
		stepN(t, w, 1)
		if len(w.employment.hires) != 2 {
			t.Fatalf("seed %d: hires after the first tick = %d, want 2", seed, len(w.employment.hires))
		}
		a, b := w.employment.hires[0], w.employment.hires[1]
		money := errVal(w.Snapshot()).Money
		s99Remove(t, seed, w, log, a.worker, []string{"abandon:agent_removed", "end:0:worker_left", "bankrupt"})
		s99Remove(t, seed, w, log, b.employer, []string{"abandon:principal_left", "end:1:employer_left", "bankrupt"})
		if got := bankruptMoney(log.all[len(log.all)-3:]); got != 10_000-210+140 {
			t.Errorf("seed %d: the estate received %d from the employer, want %d", seed, got, 10_000-210+140)
		}
		if got := balanceOf(t, w, int(a.employer)); got != 10_000-210+140 {
			t.Errorf("seed %d: the first farm holds %d, want %d: the unpaid wages came back to its own account", seed, got, 10_000-210+140)
		}
		if got := balanceOf(t, w, int(b.worker)); got != 70 || w.agents[b.worker].hasJob() {
			t.Errorf("seed %d: the surviving worker holds %d with job %v, want its wage 70 and no job", seed, got, w.agents[b.worker].hasJob())
		}
		if got := escrowHeld(t, w); got != 0 {
			t.Errorf("seed %d: escrow held = %d, want 0", seed, got)
		}
		stepN(t, w, 4)
		if got := errVal(w.Snapshot()).Money; got != money {
			t.Errorf("seed %d: the money of the world = %d after the removals, want %d", seed, got, money)
		}
	}
}

// s99Remove removes agent id and checks the events it caused against want.
func s99Remove(t *testing.T, seed uint64, w *World, log *hireLog, id AgentID, want []string) {
	t.Helper()
	from := len(log.all)
	if err := w.removeAgent(&w.agents[id], CauseConsumer); err != nil {
		t.Fatalf("seed %d: removing agent %d = %v, want nil", seed, id, err)
	}
	if got := seen(log.all[from:]); !slices.Equal(got, want) {
		t.Errorf("seed %d: events of the removal of agent %d = %v, want %v", seed, id, got, want)
	}
}
