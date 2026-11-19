package economysim_test

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// scriptedPlan is the plan the next step of a scriptSpawner appends: the test sets it
// between two steps and the spawner empties it, so it runs once. The spawner is a
// consumer extension written through the public API; the plan is the only state, and
// no checkpoint is taken of a world that holds one.
type scriptedPlan struct{ next []economysim.Decision }

// scriptSpawner appends the scripted plan to the plan of the first region.
type scriptSpawner struct{ plan *scriptedPlan }

func (s scriptSpawner) Decide(v *economysim.RegionView, _ *rand.Rand, d *economysim.Decisions) {
	if v.Region != 0 {
		return
	}
	d.List = append(d.List, s.plan.next...)
	s.plan.next = nil
}

// interruptWorld is the hire pack with 3-tick jobs (hireForkConfig), a second region
// "south" to move a worker to, the scripted spawner in place of the default one and
// the audit on, which checks the escrow identity at the end of every step.
func interruptWorld(t *testing.T, seed uint64) (*economysim.World, *hireTape, *scriptedPlan, hirePackStrategy) {
	t.Helper()
	cfg, s := hireForkConfig(t)
	south := economysim.DefaultRegion()
	south.ID, south.ReferenceBasket = "south", cfg.Regions[0].ReferenceBasket
	cfg.Regions = append(cfg.Regions, south)
	cfg.World.Spawner = "script"
	plan, tape := &scriptedPlan{}, &hireTape{}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(tape),
		economysim.WithStrategy("hire_pack", s), economysim.WithSpawner("script", scriptSpawner{plan}))
	if err != nil {
		t.Fatalf("NewWorld(hire pack + south) = %v, want nil", err)
	}
	return w, tape, plan, s
}

// moneyAndEscrow returns the money of every sector of the world and the money held in
// the Escrow accounts, and fails the test when the escrow of the sector differs from the
// sum of the open hires' accounts: the escrow identity, read from the public snapshot.
func moneyAndEscrow(t *testing.T, w *economysim.World) (total, escrow economysim.Money) {
	t.Helper()
	sn, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() at tick %d = %v, want nil", w.Tick(), err)
	}
	for _, r := range sn.Regions {
		total += r.MoneyHouseholds + r.MoneyPool + r.MoneyContract + r.MoneyTreasury
		escrow += r.MoneyContract
	}
	var held economysim.Money
	for _, h := range sn.Hires {
		held += h.Held
	}
	if held != escrow {
		t.Fatalf("tick %d: Hires hold %d, the contract sector %d, want equal", w.Tick(), held, escrow)
	}
	return total, escrow
}

// stepWorld steps w n ticks; the audit runs in each Step, and the escrow identity is
// read from the snapshot after each.
func stepWorld(t *testing.T, w *economysim.World, n int) {
	t.Helper()
	for range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d = %v, want nil", w.Tick(), err)
		}
		moneyAndEscrow(t, w)
	}
}

// wantOpenHire fails the test unless err is a refusal ErrRegionBound that names open_hire.
func wantOpenHire(t *testing.T, err error, employer economysim.AgentID) {
	t.Helper()
	var rb *economysim.RegionBoundError
	if !errors.Is(err, economysim.ErrRegionBound) || !errors.As(err, &rb) || rb.Obligation != "open_hire" {
		t.Fatalf("Relocate(employer %d) = %v, want ErrRegionBound naming open_hire", employer, err)
	}
}

// stepUntilHire steps until a hire is in progress with a tick of work left and returns
// its row; a world that never forms one fails the test.
func stepUntilHire(t *testing.T, w *economysim.World) economysim.HireRow {
	t.Helper()
	for range 12 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d = %v, want nil", w.Tick(), err)
		}
		sn, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		for _, h := range sn.Hires {
			if h.TicksLeft >= 2 {
				return h
			}
		}
	}
	t.Fatalf("no hire in progress after 12 ticks, want one")
	return economysim.HireRow{}
}

// endedOf returns the HireEnded of a hire, and whether there is one.
func endedOf(tape *hireTape, id economysim.HireID) (economysim.HireEnded, bool) {
	for _, e := range tape.events {
		if h, ok := e.(economysim.HireEnded); ok && h.Hire == id {
			return h, true
		}
	}
	return economysim.HireEnded{}, false
}

// interruptRoleOf returns the role of an agent.
func interruptRoleOf(t *testing.T, w *economysim.World, id economysim.AgentID) economysim.RoleID {
	t.Helper()
	st, err := w.Agent(id)
	if err != nil {
		t.Fatalf("Agent(%d) = %v, want nil", id, err)
	}
	return st.Role
}

// TestInterrupt_EndToEnd verifies, through the public API only and with the audit on at
// every step, each cause that interrupts a hire in progress: the worker removed
// (worker_left), the worker's role changed (worker_role), the worker moved to another
// region (worker_moved, by Relocate), the employer removed (employer_left) and the
// employer's role changed (employer_role). Removals and role changes are decisions of a
// consumer's spawner. For each: the hire ends in that tick with the cause, the escrow
// of the sector equals the sum of the open hires' accounts at every tick, no money is
// held for the ended hire, the money of the world is the same before the interruption
// and ten ticks after it, and the run goes on without an audit failure.
//
// Requirement: FR-EMP-031, FR-EMP-032, FR-EMP-033, FR-EMP-034, FR-EMP-035, FR-EMP-036, FR-EMP-038
func TestInterrupt_EndToEnd(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	tests := []struct {
		name  string
		cause economysim.Cause
		do    func(t *testing.T, w *economysim.World, plan *scriptedPlan, h economysim.HireRow, s hirePackStrategy)
	}{
		{"worker removed", economysim.CauseWorkerLeft, func(_ *testing.T, _ *economysim.World, plan *scriptedPlan, h economysim.HireRow, _ hirePackStrategy) {
			plan.next = []economysim.Decision{{Kind: economysim.DecideRemove, Agent: h.Worker, Cause: economysim.CauseInsolvent}}
		}},
		{"worker role changed", economysim.CauseWorkerRole, func(t *testing.T, w *economysim.World, plan *scriptedPlan, h economysim.HireRow, s hirePackStrategy) {
			to := s.digger
			if interruptRoleOf(t, w, h.Worker) == s.digger {
				to = s.forest
			}
			plan.next = []economysim.Decision{{Kind: economysim.DecideSwitch, Agent: h.Worker, Role: to, Cause: economysim.CauseReview}}
		}},
		{"worker moved", economysim.CauseWorkerMoved, func(t *testing.T, w *economysim.World, _ *scriptedPlan, h economysim.HireRow, _ hirePackStrategy) {
			if err := w.Relocate(h.Worker, 1); err != nil {
				t.Fatalf("Relocate(%d, 1) = %v, want nil: a worker is not bound by its hire", h.Worker, err)
			}
		}},
		{"employer removed", economysim.CauseEmployerLeft, func(_ *testing.T, _ *economysim.World, plan *scriptedPlan, h economysim.HireRow, _ hirePackStrategy) {
			plan.next = []economysim.Decision{{Kind: economysim.DecideRemove, Agent: h.Employer, Cause: economysim.CauseInsolvent}}
		}},
		{"employer role changed", economysim.CauseEmployerRole, func(t *testing.T, w *economysim.World, plan *scriptedPlan, h economysim.HireRow, s hirePackStrategy) {
			plan.next = []economysim.Decision{{Kind: economysim.DecideSwitch, Agent: h.Employer, Role: s.digger, Cause: economysim.CauseReview}}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, seed := range []uint64{1, 2, 3} {
				w, tape, plan, s := interruptWorld(t, seed)
				h := stepUntilHire(t, w)
				before, escrowBefore := moneyAndEscrow(t, w)
				if escrowBefore == 0 {
					t.Fatalf("seed %d: escrow before the interruption = 0, want the hire's bill", seed)
				}
				tc.do(t, w, plan, h, s)
				stepWorld(t, w, 10)
				checkInterrupted(t, w, tape, h, tc.cause, seed)
				if after, _ := moneyAndEscrow(t, w); after != before {
					t.Errorf("seed %d: money of the world = %d, want %d, the same as before the interruption", seed, after, before)
				}
			}
		})
	}
}

// checkInterrupted fails the test unless hire h ended with cause between its two
// parties, refunded part of its escrow and is no longer open.
func checkInterrupted(t *testing.T, w *economysim.World, tape *hireTape, h economysim.HireRow, cause economysim.Cause, seed uint64) {
	t.Helper()
	ended, ok := endedOf(tape, h.Hire)
	if !ok || ended.Cause != cause {
		t.Errorf("seed %d: HireEnded of hire %d = %+v (found %v), want cause %q", seed, h.Hire, ended, ok, cause)
	}
	if ended.Employer != h.Employer || ended.Worker != h.Worker {
		t.Errorf("seed %d: HireEnded parties = %d, %d, want %d, %d", seed, ended.Employer, ended.Worker, h.Employer, h.Worker)
	}
	if ended.Refunded <= 0 || ended.Refunded > h.Held {
		t.Errorf("seed %d: HireEnded refunded %d, want 1 to %d (the escrow held when the hire was read)", seed, ended.Refunded, h.Held)
	}
	if hireOpen(t, w, func(r economysim.HireRow) bool { return r.Hire == h.Hire }) {
		t.Errorf("seed %d: hire %d is still open after %s", seed, h.Hire, cause)
	}
}

// hireOpen reports whether a hire in progress satisfies match.
func hireOpen(t *testing.T, w *economysim.World, match func(economysim.HireRow) bool) bool {
	t.Helper()
	sn, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	return slices.ContainsFunc(sn.Hires, match)
}

// TestInterrupt_EmployerRelocateRefusedWhileHireOpen verifies through the public API
// that Relocate of an employer with a hire in progress is refused with ErrRegionBound
// naming open_hire, that nothing changes (the hire stays open and the audit passes),
// and that the same call succeeds once the hire has ended.
//
// Requirement: FR-EMP-037
func TestInterrupt_EmployerRelocateRefusedWhileHireOpen(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	w, tape, _, _ := interruptWorld(t, 1)
	h := stepUntilHire(t, w)
	err := w.Relocate(h.Employer, 1)
	wantOpenHire(t, err, h.Employer)
	if _, esc := moneyAndEscrow(t, w); esc == 0 {
		t.Errorf("escrow after the refused call = 0, want the hire's bill: a refusal changes nothing")
	}
	stepWorld(t, w, 4)
	if e, ok := endedOf(tape, h.Hire); !ok || e.Cause != economysim.CauseCompleted {
		t.Fatalf("HireEnded of hire %d = %+v (found %v), want completed", h.Hire, e, ok)
	}
	if hireOpen(t, w, func(r economysim.HireRow) bool { return r.Employer == h.Employer }) {
		t.Skipf("the employer holds another hire at tick %d, so the relocation is still bound", w.Tick())
	}
	if err := w.Relocate(h.Employer, 1); err != nil {
		t.Errorf("Relocate(employer %d) after the hire ended = %v, want nil", h.Employer, err)
	}
}
