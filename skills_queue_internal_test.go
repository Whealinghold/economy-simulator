package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// trainingConfig is skillsConfig with a weaver whose trainee pays: the weaver is
// open, runs "knead" (1 grain into 3 flour, trainee-eligible at the full yield of
// 10 000, so a trainee's output is worth 3 * 3 000 - 1 000 = 8 000 a tick at the
// reference prices against an upkeep of 1 000) and takes maxTrainees at once.
// Agents: 1 to 3 the millers of home, 4 the weaver of home, 5 and 6 the millers of
// yonder, 7 the weaver of yonder.
func trainingConfig(maxTrainees int64) Config {
	c := skillsConfig()
	c.Recipes = append(c.Recipes, Recipe{
		ID: "knead", Duration: 1, TraineeEligible: true, TraineeYield: 10_000,
		Inputs:  []Item{{Commodity: "grain", Quantity: 1}},
		Outputs: []Output{{Commodity: "flour", Quantity: 3, CostShare: 10_000}},
	})
	c.Roles[1].Recipes, c.Roles[1].Closed, c.Roles[1].MaxTrainees = []string{"knead"}, false, maxTrainees
	c.Roles[1].StartStock = []Item{{Commodity: "grain", Quantity: 10}}
	return c
}

// trainingWorld builds the world of trainingConfig after one tick.
func trainingWorld(t testing.TB, maxTrainees int64, opts ...Option) (*World, skillCast) {
	t.Helper()
	w := newMill(t, trainingConfig(maxTrainees), append([]Option{WithSeed(exitSeed), WithAudit()}, opts...)...)
	step(t, w, 1)
	return w, castOf(t, w)
}

// queueIDs is the agents of the queue of (region, role), in queue order.
func queueIDs(s *skills, region RegionID, role RoleID) []AgentID {
	var ids []AgentID
	for _, ap := range s.queue[s.cell(region, role)] {
		ids = append(ids, ap.agent)
	}
	return ids
}

// TestApply_QueueOrderApplicationTickThenID verifies that the Apply executor puts
// agents in the queue of the role in their region in the order of ticks served,
// then application tick, then agent ID: new applicants by the tick they applied and
// then by ID, and a returned trainee, which has served, ahead of them.
//
// Requirement: FR-SKL-017, FR-SKL-020
func TestApply_QueueOrderApplicationTickThenID(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	m := c.homeMillers
	// Applied at ticks 5, 3, 3 and 9 (the last has served 2): the order is the
	// returned trainee (served 2), then the two of tick 3 by ID, then tick 5.
	for _, ap := range []applicant{{m[2], 5, 0}, {m[1], 3, 0}, {m[0], 3, 0}} {
		w.skills.enqueue(w.skills.cell(0, 1), ap)
	}
	if got, want := queueIDs(w.skills, 0, 1), []AgentID{m[0], m[1], m[2]}; !slices.Equal(got, want) {
		t.Errorf("queue after three applications = %v, want %v (tick 3 by ID, then tick 5)", got, want)
	}
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: c.yonderMillers[0], applied: 9, served: 2})
	if got, want := queueIDs(w.skills, 0, 1), []AgentID{c.yonderMillers[0], m[0], m[1], m[2]}; !slices.Equal(got, want) {
		t.Errorf("queue with a returned trainee = %v, want %v (served 2 first)", got, want)
	}
	if !w.skills.queued(m[1]) || w.skills.queued(c.homeWeaver) {
		t.Errorf("queued(%d), queued(%d) = %v, %v, want true, false", m[1], c.homeWeaver, w.skills.queued(m[1]), w.skills.queued(c.homeWeaver))
	}
}

// TestApply_AlreadyQueued verifies that an agent is in at most one queue: a second
// Apply is refused already_queued, leaves the queue as it was, and so a consumer
// that repeats an Apply for an agent adds no entry (the queue is bounded by the
// agents, NFR-SKL-002).
//
// Requirement: FR-SKL-017, NFR-SKL-002
func TestApply_AlreadyQueued(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	d := Decision{Kind: DecideApply, Agent: c.homeMillers[0], Role: 1}
	for i := range 3 {
		if err := w.execute(0, d); err != nil {
			t.Fatalf("execute #%d error = %v, want nil", i, err)
		}
	}
	if got, want := queueIDs(w.skills, 0, 1), []AgentID{c.homeMillers[0]}; !slices.Equal(got, want) {
		t.Errorf("queue after three Apply decisions of one agent = %v, want %v once", got, want)
	}
	if got := w.refusal(0, &d); got != ReasonAlreadyQueued {
		t.Errorf("refusal(second Apply) = %q, want %q", got, ReasonAlreadyQueued)
	}
	// The applicant applies to the role of the other region's cell by no means: a
	// different region's agent is unknown to region 0.
	other := Decision{Kind: DecideApply, Agent: c.yonderMillers[0], Role: 1}
	if got := w.refusal(0, &other); got != ReasonUnknownAgent {
		t.Errorf("refusal(Apply of an agent of another region) = %q, want %q", got, ReasonUnknownAgent)
	}
}

// TestApply_RefusedNeedsWeightOne verifies that an Apply for an agent of weight
// above 1 is refused needs_weight_one and queues nothing, and an agent of weight 1
// is not.
//
// Requirement: FR-SKL-012, FR-SPN-009
func TestApply_RefusedNeedsWeightOne(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	w.agents[c.homeMillers[0]].weight = 2
	heavy := Decision{Kind: DecideApply, Agent: c.homeMillers[0], Role: 1}
	if got := w.refusal(0, &heavy); got != ReasonNeedsWeightOne {
		t.Errorf("refusal(Apply of weight 2) = %q, want %q", got, ReasonNeedsWeightOne)
	}
	if err := w.execute(0, heavy); err != nil || len(queueIDs(w.skills, 0, 1)) != 0 {
		t.Errorf("execute(Apply of weight 2) = %v, queue %v, want nil and an empty queue", err, queueIDs(w.skills, 0, 1))
	}
	light := Decision{Kind: DecideApply, Agent: c.homeMillers[1], Role: 1}
	if got := w.refusal(0, &light); got != "" {
		t.Errorf("refusal(Apply of weight 1) = %q, want none", got)
	}
}

// TestApply_RefusedInTraineeship verifies that an Apply of a trainee, and of a
// trainer with trainees, is refused in_traineeship (a trainer is never a trainee, so
// the apply of one is refused as a switch of one is), and a removal or a reduction of
// one is not refused for it.
//
// Requirement: FR-SPN-011
func TestApply_RefusedInTraineeship(t *testing.T) {
	t.Parallel()
	// The miller trains too, so the weaver can apply to a role other than its own.
	cfg := trainingConfig(3)
	cfg.Roles[0].TrainingTicks = 4
	w := newMill(t, cfg, WithSeed(exitSeed), WithAudit())
	step(t, w, 1)
	c := castOf(t, w)
	w.skills.start(c.homeMillers[0], c.homeWeaver, 1, 0, 0)
	for _, tt := range []struct {
		name string
		d    Decision
		want Reason
	}{
		{"apply of a trainee", Decision{Kind: DecideApply, Agent: c.homeMillers[0], Role: 1}, ReasonInTraineeship},
		{"apply of a trainer with trainees", Decision{Kind: DecideApply, Agent: c.homeWeaver, Role: 0}, ReasonInTraineeship},
		{"switch of a trainee", Decision{Kind: DecideSwitch, Agent: c.homeMillers[0], Role: 1}, ReasonInTraineeship},
		{"removal of a trainee", Decision{Kind: DecideRemove, Agent: c.homeMillers[0]}, ""},
		{"removal of a trainer", Decision{Kind: DecideRemove, Agent: c.homeWeaver}, ""},
	} {
		if got := w.refusal(0, &tt.d); got != tt.want {
			t.Errorf("%s: refusal(%+v) = %q, want %q", tt.name, tt.d, got, tt.want)
		}
	}
	sw := Decision{Kind: DecideSwitch, Agent: c.homeWeaver, Role: 0}
	if got := w.refusal(0, &sw); got != ReasonInTraineeship {
		t.Errorf("refusal(switch of a trainer with trainees) = %q, want %q", got, ReasonInTraineeship)
	}
}

// TestApply_OnlyWayIntoTrainingRole verifies that a role with training is entered
// only by Apply: a Switch or an Enter into it is refused needs_traineeship, an Apply
// is not, an Apply to a role with no training is refused as an unknown role, and an
// Apply to the agent's own role as same_role.
//
// Requirement: FR-SKL-010, FR-SPN-020
func TestApply_OnlyWayIntoTrainingRole(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	m := c.homeMillers[0]
	for _, tt := range []struct {
		name string
		d    Decision
		want Reason
	}{
		{"switch into the role", Decision{Kind: DecideSwitch, Agent: m, Role: 1}, ReasonNeedsTraineeship},
		{"entry into the role", Decision{Kind: DecideEnter, Role: 1, Households: 1}, ReasonNeedsTraineeship},
		{"apply to the role", Decision{Kind: DecideApply, Agent: m, Role: 1}, ""},
		{"apply to a role with no training", Decision{Kind: DecideApply, Agent: c.homeWeaver, Role: 0}, ReasonUnknownRole},
		{"apply to the own role", Decision{Kind: DecideApply, Agent: c.homeWeaver, Role: 1}, ReasonSameRole},
		{"apply to an unknown role", Decision{Kind: DecideApply, Agent: m, Role: 9}, ReasonUnknownRole},
		{"switch to a role with no training", Decision{Kind: DecideSwitch, Agent: c.homeWeaver, Role: 0}, ""},
	} {
		if got := w.refusal(0, &tt.d); got != tt.want {
			t.Errorf("%s: refusal(%+v) = %q, want %q", tt.name, tt.d, got, tt.want)
		}
	}
	// A world with no role that trains refuses the kind of an apply, as before.
	off := newMill(t, panConfig(), WithSeed(1))
	d := Decision{Kind: DecideApply, Agent: 1, Role: 0}
	if got := off.refusal(0, &d); got != reasonUnknownKind {
		t.Errorf("refusal(Apply) in a world with no training role = %q, want %q", got, reasonUnknownKind)
	}
}

// TestPlaces_MaxTraineesAndQueueOrder verifies the entry step's filling of
// places: with one trainer of MaxTrainees 2 and four applicants, the first two in
// queue order become its trainees, a trainer never holds more than MaxTrainees,
// and the other two stay queued in order. A second trainer takes the next, the one
// with the fewest trainees first.
//
// Requirement: FR-SKL-017, FR-SKL-016
func TestPlaces_MaxTraineesAndQueueOrder(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, c := trainingWorld(t, 2, WithObserver(rec))
	m := c.homeMillers
	for i, id := range m {
		w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: id, applied: int64(i)})
	}
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: c.yonderMillers[0], applied: 9}) // another region's agent: no place here
	// The last one is not in region 0, so take it out again as a stale entry would be.
	w.skills.dequeue(c.yonderMillers[0])
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry error = %v, want nil", err)
	}
	t.Run("the first two of the queue fill the places", func(t *testing.T) {
		if got, want := w.skills.trainer[c.homeWeaver].trainees, []AgentID{m[0], m[1]}; !slices.Equal(got, want) {
			t.Errorf("trainees of the weaver = %v, want %v (first two of the queue, MaxTrainees 2)", got, want)
		}
		if got, want := queueIDs(w.skills, 0, 1), []AgentID{m[2]}; !slices.Equal(got, want) {
			t.Errorf("queue after the step = %v, want %v", got, want)
		}
		started := of[TraineeshipStarted](rec)
		if len(started) != 2 || started[0].Trainee != m[0] || started[0].Trainer != c.homeWeaver || started[0].Role != 1 || started[0].Region != 0 || started[1].Trainee != m[1] {
			t.Errorf("TraineeshipStarted = %+v, want m[0] then m[1] for the weaver", started)
		}
	})
	t.Run("a placed trainee has served one tick", func(t *testing.T) {
		// Placed in the step, so 1 tick after the advance.
		if rec := w.skills.trainee[m[0]]; rec.served != 1 || rec.role != 1 || !rec.live {
			t.Errorf("trainee row = %+v, want served 1 of role 1", rec)
		}
	})
	t.Run("a full trainer takes nobody more", func(t *testing.T) {
		if err := w.skillsEntry(0); err != nil || len(w.skills.trainer[c.homeWeaver].trainees) != 2 {
			t.Errorf("second step: error %v, trainees %v, want nil and still 2", err, w.skills.trainer[c.homeWeaver].trainees)
		}
	})
}

// TestPlaceOpening_NegativeGainKeepsStarted verifies that a negative gain closes
// new places and does not end a traineeship in progress: with a recipe that pays
// nothing the head of the queue is not taken, and a trainee already started stays
// a trainee with its ticks served rising.
//
// Arithmetic: knead at a trainee yield of 1 gives 3 * 3 000 * 1 / 10 000 = 0 of
// output and costs 1 000 of grain, so output_a = -1 000, and the gain -1 000 - 1 000
// (upkeep) - teaching cost is below 0.
//
// Requirement: FR-SKL-016
func TestPlaceOpening_NegativeGainKeepsStarted(t *testing.T) {
	t.Parallel()
	c := trainingConfig(3)
	c.Recipes[1].TraineeYield = 1
	w := newMill(t, c, WithSeed(exitSeed), WithAudit())
	step(t, w, 1)
	cast := castOf(t, w)
	w.skills.start(cast.homeMillers[0], cast.homeWeaver, 1, 0, 0)
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: cast.homeMillers[1], applied: 1})
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry error = %v, want nil", err)
	}
	if !w.skills.isTrainee(cast.homeMillers[0]) || w.skills.trainee[cast.homeMillers[0]].served != 1 {
		t.Errorf("trainee in progress = %+v, want still a trainee with 1 tick served", w.skills.trainee[cast.homeMillers[0]])
	}
	if w.skills.isTrainee(cast.homeMillers[1]) || !w.skills.queued(cast.homeMillers[1]) {
		t.Errorf("applicant: trainee %v, queued %v, want queued and no trainee: the gain is negative", w.skills.isTrainee(cast.homeMillers[1]), w.skills.queued(cast.homeMillers[1]))
	}
}

// TestApply_QueuedApplicantLeavesOnRoleChangeOrRemoval verifies that a queued
// applicant stays a candidate for a switch to another role and is dropped from the
// queue when it changes role (spec-spawner.md 6.2), and that a removal or a move to
// another region takes it out too.
//
// Requirement: FR-SPN-020, FR-SKL-017
func TestApply_QueuedApplicantLeavesOnRoleChangeOrRemoval(t *testing.T) {
	t.Parallel()
	c := trainingConfig(3)
	w := newMill(t, c, WithSeed(exitSeed), WithAudit())
	step(t, w, 1)
	cast := castOf(t, w)
	for _, id := range cast.homeMillers[:2] {
		w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: id, applied: 1})
	}
	// A queued applicant is not refused a switch to a role without training.
	sw := Decision{Kind: DecideSwitch, Agent: cast.homeMillers[0], Role: 0}
	if got := w.refusal(0, &sw); got != ReasonSameRole {
		t.Fatalf("refusal(switch to the own role) = %q, want %q", got, ReasonSameRole)
	}
	if err := w.removeAgent(&w.agents[cast.homeMillers[1]], CauseConsumer); err != nil {
		t.Fatalf("removeAgent error = %v, want nil", err)
	}
	if got, want := queueIDs(w.skills, 0, 1), []AgentID{cast.homeMillers[0]}; !slices.Equal(got, want) {
		t.Errorf("queue after a removal = %v, want %v", got, want)
	}
	if err := w.moveAgent(&w.agents[cast.homeMillers[0]], 1, CauseConsumer); err != nil {
		t.Fatalf("moveAgent error = %v, want nil", err)
	}
	if len(queueIDs(w.skills, 0, 1)) != 0 {
		t.Errorf("queue after the applicant moved region = %v, want empty: a queue is a region's", queueIDs(w.skills, 0, 1))
	}
	a := &w.agents[cast.homeMillers[0]]
	w.skills.enqueue(w.skills.cell(1, 1), applicant{agent: a.id, applied: 2})
	w.takeRole(a, 1, 1) // a role change by any path: the applicant leaves its queue
	if len(queueIDs(w.skills, 1, 1)) != 0 || w.skills.queued(a.id) {
		t.Errorf("queue after the applicant changed role = %v, want empty", queueIDs(w.skills, 1, 1))
	}
}

// TestStart_RefusesATrainerWithTrainees verifies that a trainer with trainees
// cannot become a trainee, and that a decoder refuses a trainer with more trainees
// than its role's MaxTrainees.
//
// Guard: a trainer is never a trainee (the decoder and the executor refuse it), so
// start holds the same line; without it a planted traineeship would build a state
// the checkpoint then refuses to restore.
//
// Requirement: FR-SKL-027
func TestStart_RefusesATrainerWithTrainees(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	w.skills.start(c.homeMillers[0], c.homeWeaver, 1, 0, 0)
	if msg := invariantMessage(func() { w.skills.start(c.homeWeaver, c.homeMillers[1], 1, 0, 0) }); msg == "" {
		t.Errorf("start of a trainer with trainees as a trainee did not panic, want a violated invariant")
	}
	// Two trainees for a role that holds one.
	one, cs := trainingWorld(t, 1)
	sec := skillSection{
		trainees: [][5]int64{{int64(cs.homeMillers[0]), int64(cs.homeWeaver), 1, 0, 0}, {int64(cs.homeMillers[1]), int64(cs.homeWeaver), 1, 0, 0}},
		trainers: []trainerRow{{int64(cs.homeWeaver), 0, []int64{int64(cs.homeMillers[0]), int64(cs.homeMillers[1])}}},
	}
	r := codec.NewReader(sec.bytes())
	err := one.loadSkills(&restoreCtx{tick: 1}, r)
	if err == nil {
		err = r.Done()
	}
	if err == nil {
		t.Errorf("loadSkills accepted two trainees for a trainer of MaxTrainees 1, want a refusal")
	}
}

// TestFillPlaces_TrainerApplicantWaits verifies that a queued agent that is itself
// a trainer with trainees is skipped, not started: start refuses a trainer, so the
// filling must not reach it, and the next applicant takes the place.
//
// Requirement: FR-SKL-017, FR-SKL-027
func TestFillPlaces_TrainerApplicantWaits(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	s := w.skills
	// A miller trains a trainee of its own, then is queued for the weaver role.
	s.start(c.homeMillers[1], c.homeMillers[0], 0, 0, 0)
	s.enqueue(s.cell(0, 1), applicant{agent: c.homeMillers[0], applied: 1})
	s.enqueue(s.cell(0, 1), applicant{agent: c.homeMillers[2], applied: 2})
	w.fillPlaces(0, 1)
	if _, ok := s.trainerOf(c.homeMillers[0]); ok {
		t.Errorf("trainer %d became a trainee, want it to wait", c.homeMillers[0])
	}
	if !s.queued(c.homeMillers[0]) {
		t.Errorf("trainer applicant left the queue, want it queued")
	}
}

// TestCanTrain_QueuedAgentIsNoTrainer verifies that an applicant is not a trainer
// candidate: a queued trainer with trainees is a state the checkpoint refuses and
// fillPlaces can only make it wait, so no world may reach it. The weaver is queued
// for the miller role (a planted entry) and is then not listed by trainersOf.
//
// Requirement: FR-SKL-017, FR-CHK-023
func TestCanTrain_QueuedAgentIsNoTrainer(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	s := w.skills
	if got := w.trainersOf(0, 1); len(got) != 1 || got[0].id != c.homeWeaver {
		t.Fatalf("trainersOf before the application = %v, want the weaver", got)
	}
	s.enqueue(s.cell(0, 0), applicant{agent: c.homeWeaver, applied: 1})
	if got := w.trainersOf(0, 1); len(got) != 0 {
		t.Errorf("trainersOf with the weaver queued = %v, want none", got)
	}
	s.dequeue(c.homeWeaver)
	if got := w.trainersOf(0, 1); len(got) != 1 {
		t.Errorf("trainersOf after it left the queue = %v, want the weaver again", got)
	}
}

// TestGatherHosts_SkipsAgentsOfTheTraineeships verifies that an entry into a role
// never enlarges a trainee, a trainer with trainees or a queued applicant: their
// weight must stay 1 (FR-SKL-012) or the checkpoint refuses the world.
//
// Requirement: FR-SKL-012, FR-SPN-008, FR-CHK-023
func TestGatherHosts_SkipsAgentsOfTheTraineeships(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	m, s := c.homeMillers, w.skills
	s.start(m[0], c.homeWeaver, 1, 0, 0)
	s.enqueue(s.cell(0, 1), applicant{agent: m[1], applied: 1})
	w.gatherHosts(0, 0)
	var got []AgentID
	for _, i := range w.ex.hostAt {
		got = append(got, AgentID(i))
	}
	if want := []AgentID{m[2]}; !slices.Equal(got, want) {
		t.Errorf("hosts of the miller role = %v, want only the free miller %v", got, want)
	}
	w.gatherHosts(0, 1)
	if len(w.ex.hostAt) != 0 {
		t.Errorf("hosts of the weaver role = %v, want none (a trainer with trainees)", w.ex.hostAt)
	}
}

// TestView_SkillFields verifies the traineeship fields of the region view of a
// world that holds some: the queue ahead of a role, the trainers that can still take
// a trainee and the places each holds, and for each agent whether it is a trainee, in
// a traineeship (a trainee, or a trainer with trainees) or queued.
//
// Arithmetic: home has the weaver (4) with the trainee miller 1 and the queued
// millers 2 and 3, so for the weaver role QueueAhead is 2, MaxTrainees 3 and Trainers
// 1 (the weaver; a trainee or a queued agent is no trainer). A weaver with trainees
// still counts as a trainer, its places being what it has left.
//
// Requirement: FR-SPN-002
func TestView_SkillFields(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	m := c.homeMillers
	w.skills.start(m[0], c.homeWeaver, 1, 0, 0)
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: m[1], applied: 1})
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: m[2], applied: 2})
	v := w.buildRegionView(0)
	if r := v.Roles[1]; r.QueueAhead != 2 || r.MaxTrainees != 3 || r.Trainers != 1 {
		t.Errorf("weaver role view = queue ahead %d, max trainees %d, trainers %d, want 2, 3, 1", r.QueueAhead, r.MaxTrainees, r.Trainers)
	}
	if r := v.Roles[0]; r.QueueAhead != 0 || r.MaxTrainees != 0 || r.Trainers != 0 {
		t.Errorf("miller role view = queue ahead %d, max trainees %d, trainers %d, want none: no training", r.QueueAhead, r.MaxTrainees, r.Trainers)
	}
	want := map[AgentID][3]bool{ // trainee, in a traineeship, queued
		m[0]: {true, true, false}, m[1]: {false, false, true}, m[2]: {false, false, true},
		c.homeWeaver: {false, true, false},
	}
	for _, a := range v.Agents {
		if got := [3]bool{a.Trainee, a.InTraineeship, a.Queued}; got != want[a.Agent] {
			t.Errorf("agent %d: trainee, in traineeship, queued = %v, want %v", a.Agent, got, want[a.Agent])
		}
	}
}
