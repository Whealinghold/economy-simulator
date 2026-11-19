package economysim

import (
	"errors"
	"math"
	"testing"
)

// This file holds one test for each check of the restore of tag 24 that
// TestCheckpoint_EmploymentRestoreRejects (the bodies of the wrong shape and the wrong
// party) does not name: the checks of employmentCross on the worker's other hire, the
// regions of the parties and of the Escrow account, the hire list, the job, the
// formation tick and the two caps. Each starts from a body that loads and breaks one
// field of the world or of the body.

// rejectFixture is a hire world of one farm and two hands, stepped for ticks ticks with
// a job of 4 ticks and two hires open (when ticks is 2), and the saved hires.
func rejectFixture(t *testing.T, o hireOpts, wants func(v *AgentView) JobRequest, ticks int) *World {
	t.Helper()
	w := hireWorld(t, defaultHire(o), wants)
	stepN(t, w, ticks)
	if len(w.employment.hires) < 1 {
		t.Fatalf("hires open after %d ticks = 0, want at least one", ticks)
	}
	return w
}

// rowOf is the body row of a hire, with the Escrow account esc.
func rowOf(h hireRec, esc int) [8]uint64 {
	return [8]uint64{uint64(h.id), uint64(h.employer), uint64(h.worker), uint64(h.wage), uint64(esc), uint64(h.formed), uint64(h.region), uint64(h.paid)}
}

// wantCorrupt loads body into w and fails the test unless the load is refused with
// ErrCheckpointCorrupt.
func wantCorrupt(t *testing.T, w *World, body []byte) {
	t.Helper()
	if err := loadHireBody(w, body); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("loadEmployment(body) = %v, want ErrCheckpointCorrupt", err)
	}
}

// twoHires is the fixture of two hires of one employer, with the saved rows.
func twoHires(t *testing.T) (*World, hireRec, hireRec) {
	t.Helper()
	w := rejectFixture(t, hireOpts{hands: 2, handRecipe: true, duration: 4}, farmWants(2, 100, 40), 2)
	if len(w.employment.hires) != 2 {
		t.Fatalf("hires open = %d, want 2", len(w.employment.hires))
	}
	a, b := w.employment.hires[0], w.employment.hires[1]
	if err := loadHireBody(w, hireBody(2, rowOf(a, a.escrow), rowOf(b, b.escrow))); err != nil {
		t.Fatalf("the saved hires = %v, want nil", err)
	}
	return w, a, b
}

// TestCheckpoint_RestoreRejects_WorkerHasAnotherHire verifies that two hires that name
// one worker are refused as ErrCheckpointCorrupt: a worker works for one employer.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_WorkerHasAnotherHire(t *testing.T) {
	t.Parallel()
	w, a, b := twoHires(t)
	row := rowOf(b, b.escrow)
	row[2] = uint64(a.worker)
	wantCorrupt(t, w, hireBody(2, rowOf(a, a.escrow), row))
}

// TestCheckpoint_RestoreRejects_PartyInOtherRegion verifies that a hire that names a
// region other than the one its employer and worker stand in is refused.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_PartyInOtherRegion(t *testing.T) {
	t.Parallel()
	w := rejectFixture(t, hireOpts{hands: 1, handRecipe: true, duration: 4, regions: 2}, farmWants(1, 100, 40), 1)
	h := w.employment.hires[0]
	row := rowOf(h, h.escrow)
	row[6] = uint64(1 - h.region)
	wantCorrupt(t, w, hireBody(uint64(h.id)+1, row))
}

// TestCheckpoint_RestoreRejects_EscrowOfOtherRegion verifies that an Escrow account of
// another region than the hire's is refused: the account is open and an Escrow account,
// but its region is not the hire's.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_EscrowOfOtherRegion(t *testing.T) {
	t.Parallel()
	w := rejectFixture(t, hireOpts{hands: 1, handRecipe: true, duration: 4, regions: 2}, farmWants(1, 100, 40), 1)
	var h0, h1 hireRec
	for _, h := range w.employment.hires {
		if h.region == 0 {
			h0 = h
		} else {
			h1 = h
		}
	}
	if h1.escrow == 0 || h1.escrow == h0.escrow {
		t.Fatalf("hires = %+v, %+v, want one open hire in each region", h0, h1)
	}
	wantCorrupt(t, w, hireBody(uint64(h0.id)+1, rowOf(h0, h1.escrow)))
}

// TestCheckpoint_RestoreRejects_RecipeNotOnHireList verifies that a hire whose worker's
// job runs a recipe that is not on the employer's hire list is refused.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_RecipeNotOnHireList(t *testing.T) {
	t.Parallel()
	w := rejectFixture(t, hireOpts{hands: 1, handRecipe: true, duration: 4}, farmWants(1, 100, 40), 1)
	h := w.employment.hires[0]
	w.employment.roles[w.agents[h.employer].role].list = nil
	wantCorrupt(t, w, hireBody(uint64(h.id)+1, rowOf(h, h.escrow)))
}

// TestCheckpoint_RestoreRejects_EmployerRoleHasNoBlock verifies that a hire whose
// employer's role has no employer block is refused.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_EmployerRoleHasNoBlock(t *testing.T) {
	t.Parallel()
	w := rejectFixture(t, hireOpts{hands: 1, handRecipe: true, duration: 4}, farmWants(1, 100, 40), 1)
	h := w.employment.hires[0]
	w.employment.roles[w.agents[h.employer].role].on = false
	wantCorrupt(t, w, hireBody(uint64(h.id)+1, rowOf(h, h.escrow)))
}

// TestCheckpoint_RestoreRejects_WorkerRoleNotHireable verifies that a hire whose
// worker's role is not Hireable is refused.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_WorkerRoleNotHireable(t *testing.T) {
	t.Parallel()
	w := rejectFixture(t, hireOpts{hands: 1, handRecipe: true, duration: 4}, farmWants(1, 100, 40), 1)
	h := w.employment.hires[0]
	w.cfg.Roles[w.agents[h.worker].role].Hireable = false
	wantCorrupt(t, w, hireBody(uint64(h.id)+1, rowOf(h, h.escrow)))
}

// TestCheckpoint_RestoreRejects_JobWithNoTickAhead verifies that a hire whose worker's
// job has no working tick behind it (all its ticks are left) is refused: a hire is
// formed in the tick before the checkpoint at the latest, so a tick is worked.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_JobWithNoTickAhead(t *testing.T) {
	t.Parallel()
	w := rejectFixture(t, hireOpts{hands: 1, handRecipe: true, duration: 4}, farmWants(1, 100, 40), 1)
	h := w.employment.hires[0]
	w.agents[h.worker].job.remaining = w.cfg.Recipes[w.agents[h.worker].job.recipe].Duration
	wantCorrupt(t, w, hireBody(uint64(h.id)+1, rowOf(h, h.escrow)))
}

// TestCheckpoint_RestoreRejects_FormationTickDisagreesWithTicksWorked verifies that a
// hire whose formation tick plus the ticks its worker has worked is not the tick of the
// checkpoint is refused: one wage is paid a tick from the formation tick.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_FormationTickDisagreesWithTicksWorked(t *testing.T) {
	t.Parallel()
	w := rejectFixture(t, hireOpts{hands: 1, handRecipe: true, duration: 4}, farmWants(1, 100, 40), 2)
	h := w.employment.hires[0]
	row := rowOf(h, h.escrow)
	row[5] = uint64(h.formed + 1) // below the tick (2), so the bound passes and only the sum fails
	wantCorrupt(t, w, hireBody(uint64(h.id)+1, row))
}

// TestCheckpoint_RestoreRejects_EmployerOverMaxHires verifies that two hires of one
// employer whose executions together pass the MaxHires of its role are refused.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_EmployerOverMaxHires(t *testing.T) {
	t.Parallel()
	w, a, b := twoHires(t)
	w.employment.roles[w.agents[a.employer].role].maxHires = 1
	wantCorrupt(t, w, hireBody(2, rowOf(a, a.escrow), rowOf(b, b.escrow)))
}

// TestCheckpoint_RestoreRejects_RegionOverMaxOpen verifies that two hires of one
// region past the MaxOpen of the module are refused.
//
// Requirement: FR-EMP-044
func TestCheckpoint_RestoreRejects_RegionOverMaxOpen(t *testing.T) {
	t.Parallel()
	w, a, b := twoHires(t)
	w.employment.maxOpen = 1
	wantCorrupt(t, w, hireBody(2, rowOf(a, a.escrow), rowOf(b, b.escrow)))
}

// TestCheckpoint_RestoreRejects_WagesPaidAndOwedOverflow verifies that a hire whose
// wages paid, with the wages still owed for the ticks left, do not fit in an int64 is
// refused: releaseWage adds each tick's wage to the total without a check.
//
// Fix: the restore checked only a lower bound of the wages paid, so a crafted section
// with the wages paid near the int64 limit loaded and the next wage wrapped the total.
func TestCheckpoint_RestoreRejects_WagesPaidAndOwedOverflow(t *testing.T) {
	t.Parallel()
	w := rejectFixture(t, hireOpts{hands: 1, handRecipe: true, duration: 4}, farmWants(1, 100, 40), 1)
	h := w.employment.hires[0]
	row := rowOf(h, h.escrow)
	// The largest multiple of the wage that an int64 holds: a whole number of wages, so
	// the divisibility check passes and only the sum with the wages owed fails.
	row[7] = uint64(math.MaxInt64 - math.MaxInt64%int64(h.wage))
	wantCorrupt(t, w, hireBody(uint64(h.id)+1, row))
}
