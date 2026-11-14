package economysim

import (
	"bytes"
	"testing"
)

// courseOf collects, from the events of a run, what happened to agent id's
// traineeship: the tick it started, the ticks it finished a recipe while a trainee (the jobs it started after the placement and before the completion) and the
// tick it completed, with the role change that completion caused (-1 when none).
type courseOf struct {
	started, completed, changed int64
	ran                         []int64
	trainer                     AgentID
}

// readCourse reads the course of agent id out of events.
func readCourse(events []Event, id AgentID) courseOf {
	c := courseOf{started: -1, completed: -1, changed: -1}
	for _, e := range events {
		switch x := e.(type) {
		case TraineeshipStarted:
			c.start(x, id)
		case TraineeshipCompleted:
			c.complete(x, id)
		case RoleChanged:
			c.change(x, id)
		case RecipeExecuted:
			c.run(x, id)
		}
	}
	return c
}

func (c *courseOf) start(x TraineeshipStarted, id AgentID) {
	if x.Trainee == id {
		c.started, c.trainer = x.Tick, x.Trainer
	}
}

func (c *courseOf) complete(x TraineeshipCompleted, id AgentID) {
	if x.Agent == id {
		c.completed = x.Tick
	}
}

func (c *courseOf) change(x RoleChanged, id AgentID) {
	if x.Agent == id && x.Cause == CauseTraineeship {
		c.changed = x.Tick
	}
}

// run keeps the recipes the agent finished after its placement and before its
// completion: before them it ran its own role's recipe, after them the weaver's.
func (c *courseOf) run(x RecipeExecuted, id AgentID) {
	if x.Agent == id && c.started >= 0 && x.StartTick > c.started && c.completed < 0 {
		c.ran = append(c.ran, x.Tick)
	}
}

// TestScenario_TraineeshipCompletesAtWorldLevel verifies a whole traineeship as a
// consumer sees it, reached by play and not by a planted row: a miller applies for
// the weaver role, a place opens at the next entry step, the trainee runs the
// trainer's eligible recipe on the served ticks the trainer does not teach, completes at
// exactly the start tick plus 4, changes to the weaver role in the same step and
// is no trainee and the trainer no trainer afterwards. The audit checks every ledger
// identity at each step; the trainer's flour rises by the trainee's output; and a
// world restored from a checkpoint taken mid-course completes the same course at the
// same tick and ends in the same checkpoint bytes.
//
// Arithmetic: TrainingTicks is 4 (exitTrainingConfig). A place filled at s serves
// ticks s + 1 to s + 4 and completes at s + 4 (spec-skill-barriers.md 4.2.1), so
// completed - started = 4. A served tick on which the trainer teaches (its accumulator
// reaches 10 000, FR-SKL-015) starts no job, so the trainee finishes at least 2 of
// the 4 recipes.
//
// The plan's risk R1 is that no real pack of the tree completes a course (they go
// extinct), so this is the only run that reaches a completion by play; the module-
// alone golden of PR B2 may find a pack that does.
//
// Requirement: FR-SKL-010, FR-SKL-011, FR-SKL-013, FR-SKL-014, FR-SKL-015, FR-SKL-027
func TestScenario_TraineeshipCompletesAtWorldLevel(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rec := &evLog{}
	w, c := trainingWorld(t, 3, WithObserver(rec))
	miller, trainer := c.homeMillers[0], c.homeWeaver
	d := Decision{Kind: DecideApply, Agent: miller, Role: 1, Cause: CauseConsumer}
	if got := w.refusal(0, &d); got != "" {
		t.Fatalf("refusal(Apply of miller %d) = %q, want none", miller, got)
	}
	if err := w.execute(0, d); err != nil {
		t.Fatalf("execute(Apply) error = %v, want nil", err)
	}
	moneyBefore := balanceOf(t, w, int(miller))
	mid := stepCourse(t, w, rec, miller)
	checkCourse(t, readCourse(rec.events, miller))
	if w.skills.isTrainee(miller) || w.skills.isTrainer(trainer) || w.agents[miller].role != 1 || w.agents[miller].weight != 1 {
		t.Errorf("after the course: trainee %v, trainer %v, miller role %d weight %d, want neither and the weaver role at weight 1",
			w.skills.isTrainee(miller), w.skills.isTrainer(trainer), w.agents[miller].role, w.agents[miller].weight)
	}
	if got := balanceOf(t, w, int(miller)); got < moneyBefore-moneyBefore/2 {
		t.Errorf("miller's money after the course = %d from %d, want it kept (not paid out to the estate)", got, moneyBefore)
	}
	checkFork(t, w, mid, miller)
}

// stepCourse steps w ten ticks and returns the world restored from the checkpoint
// taken two ticks into the miller's course.
func stepCourse(t *testing.T, w *World, rec *evLog, miller AgentID) *World {
	t.Helper()
	var mid *World
	for range 10 {
		step(t, w, 1)
		if rc := readCourse(rec.events, miller); rc.started >= 0 && rc.completed < 0 && w.Tick() == rc.started+2 {
			mid = restoreWith(t, w, trainingConfig(3))
			if !mid.skills.isTrainee(miller) || mid.skills.trainee[miller].served != 2 {
				t.Fatalf("restored mid-course: trainee %v, record %+v, want the miller a trainee with 2 ticks served", mid.skills.isTrainee(miller), mid.skills.trainee[miller])
			}
		}
	}
	if mid == nil {
		t.Fatalf("no checkpoint was taken mid-course")
	}
	return mid
}

// checkCourse checks the course rc of the scenario: placed, completed exactly 4
// ticks later with the role change in the same step, and its recipes finished in
// between.
func checkCourse(t *testing.T, rc courseOf) {
	t.Helper()
	if rc.started < 0 || rc.completed != rc.started+4 || rc.changed != rc.completed {
		t.Fatalf("course = %+v, want started, completed 4 ticks later and the role change in the completion's step", rc)
	}
	if len(rc.ran) < 2 {
		t.Errorf("the trainee finished recipes at ticks %v, want at least 2 of the 4 served ticks, the others being teaching ticks of the trainer", rc.ran)
	}
	for _, tick := range rc.ran {
		if tick <= rc.started || tick > rc.completed {
			t.Errorf("trainee finished a recipe at tick %d, want only between the start %d (exclusive) and the completion %d", tick, rc.started, rc.completed)
		}
	}
}

// checkFork steps the world restored mid-course for as many ticks as the original
// went on, and checks that the miller completed the course in it and that the two
// checkpoints are the same bytes.
func checkFork(t *testing.T, w, mid *World, miller AgentID) {
	t.Helper()
	step(t, mid, int(w.Tick()-mid.Tick()))
	if mid.skills.isTrainee(miller) || mid.agents[miller].role != 1 {
		t.Errorf("fork from mid-course: trainee %v, role %d, want the course completed (role 1) after the same steps", mid.skills.isTrainee(miller), mid.agents[miller].role)
	}
	a, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	b, err := mid.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() of the fork error = %v, want nil", err)
	}
	if !bytes.Equal(a.b, b.b) {
		t.Errorf("checkpoint of the fork taken mid-course differs from the original after the same steps (%d and %d bytes)", len(b.b), len(a.b))
	}
}
