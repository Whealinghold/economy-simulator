package economysim

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/skill"
)

// crowdConfig is trainingConfig with enough households for a queue: in home eight
// millers and three weavers, in yonder two millers and one weaver, all of weight 1,
// and two places a trainer, so applicants wait, trainers fill and course end.
func crowdConfig() Config {
	c := trainingConfig(2)
	c.Groups = []Group{
		{Region: "home", Role: "miller", Households: 8, Scale: 1},
		{Region: "home", Role: "weaver", Households: 3},
		{Region: "yonder", Role: "miller", Households: 2, Scale: 1},
		{Region: "yonder", Role: "weaver", Households: 1},
	}
	return c
}

// ledgerTotals sums the balances of every account of the ledger, boundaries
// included, and the units of every commodity held by the open ones: the figures a
// move between two accounts must leave alone.
func ledgerTotals(t testing.TB, w *World) (money int64, goods []int64) {
	t.Helper()
	goods = make([]int64, len(w.cfg.Commodities))
	for id := range w.ledger.Len() {
		v, err := w.ledger.Account(id)
		if err != nil {
			t.Fatalf("Account(%d) error = %v, want nil", id, err)
		}
		money += v.Balance
		if v.Closed {
			continue
		}
		for c := range goods {
			if n, _, err := w.ledger.Holding(id, c); err == nil {
				goods[c] += n
			}
		}
	}
	return money, goods
}

// skillsStateErr returns the first violated invariant of the traineeship state of
// w, or "" when it holds: an agent is exactly one of free, queued, trainee and
// trainer; a trainee is in the region and the role of its trainer, trains for the
// trainer's role and has served at most TrainingTicks; a trainer holds at most
// MaxTrainees in ascending ID; every queue is in the order of skill.Requeue and of
// agents that are neither trainee nor trainer.
func skillsStateErr(w *World) string {
	for _, check := range []func(*World) string{agentsErr, traineesErr, trainersErr, queuesErr} {
		if msg := check(w); msg != "" {
			return msg
		}
	}
	return ""
}

// agentsErr checks that each household is in at most one of the queue, a
// traineeship as trainee and one as trainer, that a removed agent is in none and
// that an agent bound to one has weight 1.
func agentsErr(w *World) string {
	s := w.skills
	for i := w.firstHousehold; i < len(w.agents); i++ {
		id, in := AgentID(i), 0
		for _, is := range []bool{s.queued(id), s.isTrainee(id), s.isTrainer(id)} {
			if is {
				in++
			}
		}
		switch {
		case in > 1:
			return fmt.Sprintf("agent %d is queued %v, trainee %v and trainer %v, want at most one", id, s.queued(id), s.isTrainee(id), s.isTrainer(id))
		case in == 1 && w.agents[i].weight != 1:
			return fmt.Sprintf("agent %d in a traineeship or queue has weight %d (0 is a removed agent), want 1", id, w.agents[i].weight)
		}
	}
	return ""
}

// traineesErr checks each trainee against its trainer: the same region, the
// trainer's role as the one trained for (not the trainee's own) and the ticks served
// within the course.
func traineesErr(w *World) string {
	for id, rec := range w.skills.trainee {
		if !rec.live {
			continue
		}
		tr, a := &w.agents[rec.trainer], &w.agents[id]
		switch {
		case w.regionOf(tr) != w.regionOf(a):
			return fmt.Sprintf("trainee %d and its trainer %d are in different regions", id, rec.trainer)
		case tr.role != rec.role || a.role == rec.role:
			return fmt.Sprintf("trainee %d (role %d) trains for role %d under trainer %d of role %d", id, a.role, rec.role, rec.trainer, tr.role)
		case rec.served < 0 || rec.served > w.cfg.Roles[rec.role].TrainingTicks:
			return fmt.Sprintf("trainee %d has served %d of %d ticks", id, rec.served, w.cfg.Roles[rec.role].TrainingTicks)
		}
	}
	return ""
}

// trainersErr checks each trainer's places, the order of its list and its
// accumulator, which is 0 for a record with no trainee and below 10 000 always.
func trainersErr(w *World) string {
	for id, tr := range w.skills.trainer {
		max := w.cfg.Roles[w.agents[id].role].MaxTrainees
		switch {
		case int64(len(tr.trainees)) > max:
			return fmt.Sprintf("trainer %d holds %d trainees, above MaxTrainees %d", id, len(tr.trainees), max)
		case !slices.IsSorted(tr.trainees) || (len(tr.trainees) == 0 && tr.acc != 0) || tr.acc < 0 || tr.acc >= skill.Basis:
			return fmt.Sprintf("trainer %d has trainees %v and accumulator %d", id, tr.trainees, tr.acc)
		}
	}
	return ""
}

// queuesErr checks the queues of w: each entry is a household of the cell's region
// that is not in the cell's role, the entries are in the order of skill.Requeue and
// the agent's queue marker names its cell.
func queuesErr(w *World) string {
	s := w.skills
	for cell, q := range s.queue {
		region, role := RegionID(cell/s.roles), RoleID(cell%s.roles)
		for i, ap := range q {
			a := &w.agents[ap.agent]
			switch {
			case s.queuedIn[ap.agent] != int32(cell)+1:
				return fmt.Sprintf("agent %d is in cell %d but marked for %d", ap.agent, cell, s.queuedIn[ap.agent]-1)
			case w.regionOf(a) != region || a.role == role:
				return fmt.Sprintf("agent %d (region %d, role %d) waits in cell (region %d, role %d)", ap.agent, w.regionOf(a), a.role, region, role)
			case i > 0 && !skill.Requeue(q[i-1].rule(), ap.rule()):
				return fmt.Sprintf("cell %d: agent %d is not after agent %d in queue order", cell, ap.agent, q[i-1].agent)
			}
		}
	}
	return ""
}

// crowdOps is what a seeded run of the crowd did, for the checks over all seeds.
type crowdOps struct {
	started, completed, trainerLeft, agentLeft, removedTrainer, removedTrainee, removedQueued int
}

// runCrowd steps the crowd world 40 ticks under seed. Before each step it applies a
// few random agents to the weaver role and, now and again, removes one agent,
// preferring one in a traineeship or a queue, and checks that the removal moved money and goods
// and created none. After each step it checks the invariants of skillsStateErr and
// that a checkpoint restores to a world that writes the same bytes of tag 10 and of
// the whole checkpoint.
func runCrowd(t *testing.T, seed uint64) crowdOps {
	t.Helper()
	var got crowdOps
	obs := ObserverFunc(func(e Event) {
		switch x := e.(type) {
		case TraineeshipStarted:
			got.started++
		case TraineeshipCompleted:
			got.completed++
		case TraineeshipEnded:
			if x.Reason == ReasonTrainerLeft {
				got.trainerLeft++
			} else {
				got.agentLeft++
			}
		}
	})
	w := newMill(t, crowdConfig(), WithSeed(seed), WithAudit(), WithObserver(obs))
	rng := rand.New(rand.NewPCG(seed, 0x5eed))
	for range 40 {
		crowdApply(w, rng)
		if rng.IntN(4) == 0 {
			crowdRemove(t, w, rng, seed, &got)
		}
		step(t, w, 1)
		if msg := skillsStateErr(w); msg != "" {
			t.Fatalf("seed %d, tick %d: %s", seed, w.Tick(), msg)
		}
		fork := restoreWith(t, w, crowdConfig())
		if a, b := section10(t, w), section10(t, fork); !bytes.Equal(a, b) {
			t.Fatalf("seed %d, tick %d: tag 10 after restore = %v, want %v", seed, w.Tick(), b, a)
		}
		if msg := skillsStateErr(fork); msg != "" {
			t.Fatalf("seed %d, tick %d: restored world: %s", seed, w.Tick(), msg)
		}
	}
	return got
}

// crowdApply directs up to three random live agents to apply for the weaver role,
// through the executor's own refusal check, as a consumer's spawner would.
func crowdApply(w *World, rng *rand.Rand) {
	for range rng.IntN(4) {
		a := &w.agents[w.firstHousehold+rng.IntN(len(w.agents)-w.firstHousehold)]
		d := Decision{Kind: DecideApply, Agent: a.id, Role: 1, Cause: CauseConsumer}
		if a.weight == 0 || a.role == 1 {
			continue
		}
		if g := w.regionOf(a); w.refusal(g, &d) == "" {
			if err := w.execute(g, d); err != nil {
				panic(err)
			}
		}
	}
}

// crowdPick returns the live agent to remove: one in a traineeship or a queue half
// of the time when there is one, else any; nil when too few agents are left.
func crowdPick(w *World, rng *rand.Rand) *agent {
	var bound, live []*agent
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight > 0 {
			live = append(live, a)
			if w.skills.bound(a.id) {
				bound = append(bound, a)
			}
		}
	}
	if len(bound) > 0 && rng.IntN(2) == 0 {
		live = bound
	}
	if len(live) <= 3 {
		return nil
	}
	return live[rng.IntN(len(live))]
}

// crowdRemove removes one agent chosen by crowdPick and checks that the move of its
// money and goods to the estate and the return of its trainees changed no total of
// the ledger.
func crowdRemove(t *testing.T, w *World, rng *rand.Rand, seed uint64, got *crowdOps) {
	t.Helper()
	a := crowdPick(w, rng)
	if a == nil {
		return
	}
	trainer, trainee, queued := w.skills.isTrainer(a.id), w.skills.isTrainee(a.id), w.skills.queued(a.id)
	money, goods := ledgerTotals(t, w)
	if err := w.execute(w.regionOf(a), Decision{Kind: DecideRemove, Agent: a.id, Cause: CauseConsumer}); err != nil {
		t.Fatalf("seed %d, tick %d: remove agent %d error = %v, want nil", seed, w.Tick(), a.id, err)
	}
	money2, goods2 := ledgerTotals(t, w)
	if money != money2 || !slices.Equal(goods, goods2) {
		t.Errorf("seed %d, tick %d: removing agent %d (trainer %v, trainee %v, queued %v): money %d -> %d, goods %v -> %v, want both unchanged",
			seed, w.Tick(), a.id, trainer, trainee, queued, money, money2, goods, goods2)
	}
	got.removedTrainer += int(b2i(trainer))
	got.removedTrainee += int(b2i(trainee))
	got.removedQueued += int(b2i(queued))
}

// TestSkills_SeededOpSequencesKeepTheInvariants verifies, over five seeds of a 40
// tick world with queues, full trainers and courses of 4 ticks, that whatever
// applications and removals come between steps: an agent is at most one of queued,
// trainee and trainer; the trainer lists, the places, the queue order and the ticks
// served stay within their bounds; the ledger audit holds at every step; removing a
// trainee or a trainer moves no money and no goods out of the ledger's total; and
// every state the world reaches is one a checkpoint restores to the same bytes. It
// also requires the runs to have started, completed and ended courses, by a trainer's
// and by a trainee's removal, so the properties were exercised.
//
// Guard: the unit tests plant their states; this test reaches them by play, so a
// state restore refuses but the world can reach (or the reverse) is found here.
//
// Requirement: FR-SKL-017, FR-SKL-020, FR-SKL-027, FR-CHK-023
func TestSkills_SeededOpSequencesKeepTheInvariants(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	var all crowdOps
	for seed := uint64(1); seed <= 5; seed++ {
		r := runCrowd(t, seed)
		all.started += r.started
		all.completed += r.completed
		all.trainerLeft += r.trainerLeft
		all.agentLeft += r.agentLeft
		all.removedTrainer += r.removedTrainer
		all.removedTrainee += r.removedTrainee
		all.removedQueued += r.removedQueued
	}
	if all.started == 0 || all.completed == 0 || all.trainerLeft == 0 || all.agentLeft == 0 || all.removedTrainer == 0 || all.removedTrainee == 0 || all.removedQueued == 0 {
		t.Errorf("over 5 seeds: %+v, want each of started, completed, trainer_left and agent_left endings and the removal of a trainer, of a trainee and of an applicant above 0", all)
	}
}

// TestSkills_QueueOrderIsInsertionIndependent verifies, over 50 seeded shuffles of
// 12 applicants with ties in ticks served and application tick, that enqueue gives
// the order of skill.Requeue whatever the insertion order, that the queue marker of
// each agent names its cell, and that taking agents out in a random order leaves the
// rest in that order.
//
// Arithmetic: ticks served come from {0, 2}, application ticks from {1, 2, 3}, so
// with 12 agents many entries tie on both and only the agent ID separates them.
//
// Requirement: FR-SKL-017, FR-SKL-020
func TestSkills_QueueOrderIsInsertionIndependent(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 50; seed++ {
		rng := rand.New(rand.NewPCG(seed, 0x0de5))
		s := &skills{roles: 2, queue: make([][]applicant, 4)}
		aps := make([]applicant, 12)
		for i := range aps {
			aps[i] = applicant{agent: AgentID(i + 1), applied: int64(1 + rng.IntN(3)), served: int64(2 * rng.IntN(2))}
		}
		want := slices.Clone(aps)
		slices.SortFunc(want, func(a, b applicant) int {
			switch {
			case skill.Requeue(a.rule(), b.rule()):
				return -1
			case skill.Requeue(b.rule(), a.rule()):
				return 1
			}
			return 0
		})
		rng.Shuffle(len(aps), func(i, j int) { aps[i], aps[j] = aps[j], aps[i] })
		for _, ap := range aps {
			s.enqueue(3, ap)
		}
		if !slices.Equal(s.queue[3], want) {
			t.Fatalf("seed %d: queue = %+v, want %+v", seed, s.queue[3], want)
		}
		for _, i := range rng.Perm(len(want))[:5] {
			s.dequeue(want[i].agent)
			if s.queued(want[i].agent) {
				t.Errorf("seed %d: agent %d is still marked as queued after dequeue", seed, want[i].agent)
			}
		}
		if !slices.IsSortedFunc(s.queue[3], func(a, b applicant) int {
			if skill.Requeue(a.rule(), b.rule()) {
				return -1
			}
			return 1
		}) || len(s.queue[3]) != 7 {
			t.Errorf("seed %d: queue after 5 removals = %+v, want 7 entries in queue order", seed, s.queue[3])
		}
	}
}
