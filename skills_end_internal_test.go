package economysim

import (
	"slices"
	"strconv"
	"testing"
)

// endOrder names the events of the traineeships and of a removal, in order, for the
// tests of how a traineeship ends.
func endOrder(events []Event) []string {
	var out []string
	for _, e := range events {
		switch x := e.(type) {
		case RecipeAbandoned:
			out = append(out, "abandoned:"+idStr(int64(x.Agent))+":"+string(x.Reason))
		case TraineeshipEnded:
			out = append(out, "ended:"+idStr(int64(x.Agent))+":"+string(x.Reason))
		case AgentBankrupted:
			out = append(out, "bankrupted:"+idStr(int64(x.Agent)))
		case EstateOpened:
			out = append(out, "estate:"+idStr(int64(x.Agent)))
		}
	}
	return out
}

// idStr formats an agent ID for an expected event list.
func idStr(n int64) string { return strconv.Itoa(int(n)) }

// TestTrainerRemoved_TraineesRequeue verifies the removal of a trainer
// (spec-skill-barriers.md 4.6): in ascending trainee ID each trainee loses its job
// (RecipeAbandoned, principal_left) and its traineeship ends (TraineeshipEnded,
// trainer_left), before the events of the removal; the trainees keep their role and
// the ticks they served and rejoin the queue of the role they trained for ordered by
// ticks served, then application tick; the trainer's record is gone.
//
// Arithmetic: trainee A (applied at tick 2) has served 1 tick and trainee B (applied
// at tick 5) 3, so B is ahead of A in the queue although it applied later; C was
// queued with no ticks served at tick 1, so it is behind both.
//
// Requirement: FR-SKL-020, FR-SPN-011, FR-OBS-020, FR-OBS-021
func TestTrainerRemoved_TraineesRequeue(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, c := trainingWorld(t, 3, WithObserver(rec))
	a, b, queued := c.homeMillers[0], c.homeMillers[1], c.homeMillers[2]
	trainer := c.homeWeaver
	w.skills.start(a, trainer, 1, 1, 2)
	w.skills.start(b, trainer, 1, 3, 5)
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: queued, applied: 1})
	w.skills.trainer[trainer].acc = 4_000
	// A has a job in progress, which the end of its traineeship abandons.
	w.agents[a].job = job{recipe: kneadOf(t, w), remaining: 2, executions: 1, principal: trainer, yield: 5_000, pending: []int64{1}}
	before := len(rec.events)
	if err := w.removeAgent(&w.agents[trainer], CauseConsumer); err != nil {
		t.Fatalf("removeAgent error = %v, want nil", err)
	}
	ida, idb, idt := int64(a), int64(b), int64(trainer)
	want := []string{
		"abandoned:" + idStr(ida) + ":principal_left", "ended:" + idStr(ida) + ":trainer_left",
		"ended:" + idStr(idb) + ":trainer_left",
		"bankrupted:" + idStr(idt), "estate:" + idStr(idt),
	}
	t.Run("events in ascending trainee ID, then the removal's", func(t *testing.T) {
		if got := endOrder(rec.events[before:]); !slices.Equal(got, want) {
			t.Errorf("events of the removal = %v, want %v", got, want)
		}
	})
	t.Run("the trainees rejoin the queue with what they served", func(t *testing.T) {
		if got, want := queueIDs(w.skills, 0, 1), []AgentID{b, a, queued}; !slices.Equal(got, want) {
			t.Errorf("queue after the removal = %v, want %v (ticks served 3, 1, 0)", got, want)
		}
		if q := w.skills.queue[w.skills.cell(0, 1)]; q[0].served != 3 || q[0].applied != 5 || q[1].served != 1 || q[1].applied != 2 {
			t.Errorf("returned trainees = %+v, want their ticks served and application ticks kept", q[:2])
		}
	})
	t.Run("the trainees keep their role and the trainer's record is gone", func(t *testing.T) {
		if w.skills.isTrainee(a) || w.skills.isTrainee(b) || w.agents[a].role != 0 || w.agents[b].role != 0 {
			t.Errorf("returned trainees: trainee %v, %v, roles %d, %d, want none and their own role 0", w.skills.isTrainee(a), w.skills.isTrainee(b), w.agents[a].role, w.agents[b].role)
		}
		if len(w.skills.trainer) > int(trainer) && (w.skills.trainer[trainer].acc != 0 || len(w.skills.trainer[trainer].trainees) != 0) {
			t.Errorf("record of the removed trainer = %+v, want zero", w.skills.trainer[trainer])
		}
	})
}

// TestTraineeRemoved_EndsAndGoesToEstate verifies the removal of a trainee: its job
// is abandoned (agent_removed) and TraineeshipEnded (agent_left) comes before
// AgentBankrupted; the trainer loses it, and its accumulator is reset with its last
// trainee; the money and goods of the trainee go to the estate as for any removal.
//
// Requirement: FR-SPN-011, FR-AGT-008, FR-OBS-020
func TestTraineeRemoved_EndsAndGoesToEstate(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, c := trainingWorld(t, 3, WithObserver(rec))
	a, b, trainer := c.homeMillers[0], c.homeMillers[1], c.homeWeaver
	w.skills.start(a, trainer, 1, 2, 3)
	w.skills.start(b, trainer, 1, 2, 3)
	w.skills.trainer[trainer].acc = 3_000
	w.agents[a].job = job{recipe: kneadOf(t, w), remaining: 2, executions: 1, principal: trainer, yield: 5_000, pending: []int64{1}}
	before := len(rec.events)
	if err := w.removeAgent(&w.agents[a], CauseInsolvent); err != nil {
		t.Fatalf("removeAgent(a) error = %v, want nil", err)
	}
	want := []string{"abandoned:" + idStr(int64(a)) + ":agent_removed", "ended:" + idStr(int64(a)) + ":agent_left", "bankrupted:" + idStr(int64(a)), "estate:" + idStr(int64(a))}
	if got := endOrder(rec.events[before:]); !slices.Equal(got, want) {
		t.Errorf("events of the removal of a trainee = %v, want %v", got, want)
	}
	if got, want := w.skills.trainer[trainer].trainees, []AgentID{b}; !slices.Equal(got, want) || w.skills.trainer[trainer].acc != 3_000 {
		t.Errorf("trainer = %+v, want only %v with its accumulator kept at 3 000", w.skills.trainer[trainer], want)
	}
	if err := w.removeAgent(&w.agents[b], CauseInsolvent); err != nil {
		t.Fatalf("removeAgent(b) error = %v, want nil", err)
	}
	if w.skills.isTrainer(trainer) || w.skills.trainer[trainer].acc != 0 {
		t.Errorf("trainer after its last trainee left = %+v, want the zero record", w.skills.trainer[trainer])
	}
	step(t, w, 2) // the audit of two steps finds the ledger whole
}

// TestTrainerSwitchRefused verifies at the executor that a switch of a trainer with
// trainees, and of a trainee, is refused in_traineeship and changes nothing, while
// the removal of either goes through (a removal is never refused for a traineeship).
//
// Requirement: FR-SPN-011
func TestTrainerSwitchRefused(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	a, trainer := c.homeMillers[0], c.homeWeaver
	w.skills.start(a, trainer, 1, 0, 0)
	for _, d := range []Decision{
		{Kind: DecideSwitch, Agent: trainer, Role: 0, Cause: CauseReview},
		{Kind: DecideSwitch, Agent: a, Role: 1, Cause: CauseReview},
	} {
		role := w.agents[d.Agent].role
		if err := w.execute(0, d); err != nil || w.agents[d.Agent].role != role || !w.skills.isTrainee(a) {
			t.Errorf("execute(%+v) = %v, role %d, want nil, role %d and the traineeship kept", d, err, w.agents[d.Agent].role, role)
		}
		if got := w.refusal(0, &d); got != ReasonInTraineeship {
			t.Errorf("refusal(%+v) = %q, want %q", d, got, ReasonInTraineeship)
		}
	}
	if err := w.execute(0, Decision{Kind: DecideRemove, Agent: trainer, Cause: CauseStarved}); err != nil {
		t.Fatalf("execute(remove trainer) error = %v, want nil", err)
	}
	if w.agents[trainer].weight != 0 || w.skills.isTrainee(a) {
		t.Errorf("after the removal of the trainer: weight %d, trainee %v, want removed and the traineeship ended", w.agents[trainer].weight, w.skills.isTrainee(a))
	}
}

// TestExternalTrainee_Directed verifies that a consumer's spawner may direct an
// external agent that the world may remove (mortal) into a traineeship by Apply, that
// the default spawner never applies for it, and that the default ladder never
// removes a trainee (its board comes from its trainer) but does remove a trainer.
//
// Requirement: FR-SPN-013, FR-SPN-010
func TestExternalTrainee_Directed(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	ext := c.homeMillers[0]
	w.agents[ext].external, w.agents[ext].mortal = true, true
	d := Decision{Kind: DecideApply, Agent: ext, Role: 1}
	if got := w.refusal(0, &d); got != "" {
		t.Fatalf("refusal(Apply of a mortal external agent) = %q, want none", got)
	}
	if err := w.execute(0, d); err != nil || !w.skills.queued(ext) {
		t.Fatalf("execute error = %v, queued %v, want nil and queued", err, w.skills.queued(ext))
	}
	w.agents[ext].mortal = false
	if got := w.refusal(0, &d); got != ReasonExternalAgent && got != ReasonAlreadyQueued {
		t.Errorf("refusal(Apply of an external agent the world may not remove) = %q, want external_agent", got)
	}
	// The ladder: a trainee is exempt, a trainer is not.
	v := spwView()
	v.Roles = []RoleView{spwRole(0, wLow, 4), spwRole(1, wMid, 1)}
	trainee, trainer := spwUnfed(spwAgent(1, 0, 1), 12, 1), spwUnfed(spwAgent(2, 1, 1), 12, 1)
	trainee.Trainee, trainee.InTraineeship, trainer.InTraineeship = true, true, true
	v.Agents = []RegionAgent{trainee, trainer}
	plan := spwDecide(v)
	var removed []AgentID
	for _, x := range plan.List {
		if x.Kind == DecideRemove || x.Kind == DecideReduce {
			removed = append(removed, x.Agent)
		}
	}
	if !slices.Equal(removed, []AgentID{2}) {
		t.Errorf("ladder removed %v, want only the trainer 2 (the trainee is exempt)", removed)
	}
}
