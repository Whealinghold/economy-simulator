package economysim

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/skill"
)

// This file is the skill barriers' traineeship (spec-skill-barriers.md, SKL).
// The barriers are core data, not a module: there is no Modules pointer, no hash
// field and no modulesOn name, and TrainingTicks 0 is "no traineeship"
// (spec-config.md 3.7.2). The state below exists only when some role has
// TrainingTicks above 0, and w.skills is nil otherwise, so every call into this
// file is guarded as a module's is (architecture.md 2.10.1, plan 3-skl SK1,
// SK2). It holds the state, the guard and the codec's data; the tick positions
// and the queue are skills_queue.go, the reports and the checkpoint are
// skills_report.go. The arithmetic is internal/skill; this file converts at the
// call (architecture.md 2.9).

// inTraineeshipName is the name of the region_bound obligation this module
// registers (spec-regions.md 6.5.2, spec-skill-barriers.md 4.6.1).
const inTraineeshipName = string(ReasonInTraineeship)

// traineeRec is a trainee's row, indexed by agent slot: its trainer, the role it
// is training for, the ticks served and the tick it applied (spec-skill-barriers.md
// 9). live is false for an agent that is not a trainee.
type traineeRec struct {
	trainer         AgentID
	role            RoleID
	served, applied int64
	live            bool
}

// trainerRec is a trainer's row, indexed by agent slot: the teaching accumulator
// and the trainees in ascending agent ID. A trainer is live exactly while it has
// a trainee, so its accumulator is 0 when it takes its first trainee and is
// dropped with the last (spec-skill-barriers.md 4.2.2).
type trainerRec struct {
	acc      int64
	trainees []AgentID
	// teaching is whether this tick is a teaching tick of the trainer, set at P2.2
	// and read by the starts of the tick. It is derived and never saved
	// (spec-skill-barriers.md 9).
	teaching bool
}

// applicant is a queue entry: the agent, the tick it applied and the ticks it has
// served (a trainee returned by a trainer's removal keeps them).
type applicant struct {
	agent           AgentID
	applied, served int64
}

// skills is the state of the traineeships, nil unless some role has TrainingTicks
// above 0 (SK2). Per-agent state is in the slices below, indexed by agent slot and
// grown on demand, never a field of the agent type (architecture.md 2.10.1,
// N-244). queue and warnRun are per (region, role) cell, region * roles + role.
// The places, the throttle, the expected wait and the entry decision are derived
// and not saved (spec-skill-barriers.md 9).
type skills struct {
	roles   int
	trainee []traineeRec
	trainer []trainerRec
	queue   [][]applicant
	warnRun []int64
	// queuedIn is, per agent slot, the queue cell the agent waits in plus 1, or 0
	// when it is in no queue (an agent is in at most one, spec-skill-barriers.md
	// 4.6.2). It is derived from queue and never saved.
	queuedIn []int32
	// board is the scratch of fillBoard, valid until the next call.
	board []UpkeepRow
	// headroom is the headroom a role has for the completion of a trainee in a
	// region (spec-institutions.md 4.5.2); nil means no limit. The settlement and
	// institutions modules set it at construction, so it is derived from the
	// configuration and never saved. ready is the scratch of the entry step.
	headroom func(RegionID, RoleID) (headroom int64, noLimit bool)
	ready    []AgentID
}

// newSkills returns the state of the traineeships, or nil when no role of the
// configuration has TrainingTicks above 0.
func newSkills(c *Config) *skills {
	for i := range c.Roles {
		if c.Roles[i].TrainingTicks > 0 {
			cells := len(c.Regions) * len(c.Roles)
			return &skills{roles: len(c.Roles), queue: make([][]applicant, cells), warnRun: make([]int64, cells)}
		}
	}
	return nil
}

// cell is the index of the (region, role) pair in queue and warnRun.
func (s *skills) cell(region RegionID, role RoleID) int { return int(region)*s.roles + int(role) }

// isTrainee reports whether the agent is a trainee.
func (s *skills) isTrainee(id AgentID) bool {
	return id >= 0 && int(id) < len(s.trainee) && s.trainee[id].live
}

// trainerOf returns the trainer of the agent when it is a trainee.
func (s *skills) trainerOf(id AgentID) (AgentID, bool) {
	if !s.isTrainee(id) {
		return 0, false
	}
	return s.trainee[id].trainer, true
}

// held is the number of trainees of the agent.
func (s *skills) held(id AgentID) int64 {
	if id < 0 || int(id) >= len(s.trainer) {
		return 0
	}
	return int64(len(s.trainer[id].trainees))
}

// isTrainer reports whether the agent has at least one trainee.
func (s *skills) isTrainer(id AgentID) bool {
	return id >= 0 && int(id) < len(s.trainer) && len(s.trainer[id].trainees) > 0
}

// bound reports whether the agent is in a traineeship or waits in a queue, so its
// weight must stay 1. It is false while the receiver is nil.
func (s *skills) bound(id AgentID) bool {
	return s != nil && (s.isTrainee(id) || s.isTrainer(id) || s.queued(id))
}

// holds is the test of the region_bound obligation in_traineeship: a trainee, or a
// trainer with trainees, unless skip names the instance by key
// "in_traineeship:<agent>" or the obligation by name (spec-regions.md 6.5.2).
func (s *skills) holds(a *agent, skip []string) bool {
	return (s.isTrainee(a.id) || s.isTrainer(a.id)) && !skipHas(skip, inTraineeshipName, int64(a.id))
}

// start records a traineeship: the trainee joins the trainer's list in ascending
// agent ID and the trainer, if it had no trainee, starts with a teaching
// accumulator of 0. A trainee that already is one, a trainer with trainees that
// would become one (a trainer is never a trainee, and the default spawner and
// the executor refuse its application), a trainer that is a trainee or an agent
// that is its own trainer is a violated invariant of the caller: the queue
// admits an agent once and takes it out of the queue first.
func (s *skills) start(trainee, trainer AgentID, role RoleID, served, applied int64) {
	if trainee == trainer || s.isTrainee(trainee) || s.isTrainer(trainee) || s.isTrainee(trainer) {
		invariant(fmt.Sprintf("a traineeship was started for an agent that is a trainee, a trainer with trainees or its own trainer (trainee %d, trainer %d)", trainee, trainer))
	}
	s.grow(max(trainee, trainer))
	s.trainee[trainee] = traineeRec{trainer: trainer, role: role, served: served, applied: applied, live: true}
	at, _ := slices.BinarySearch(s.trainer[trainer].trainees, trainee)
	s.trainer[trainer].trainees = slices.Insert(s.trainer[trainer].trainees, at, trainee)
}

// end deletes the traineeship of a trainee and reports its row. The trainer's
// accumulator is reset to 0 with its last trainee (spec-skill-barriers.md 4.2.2).
func (s *skills) end(trainee AgentID) (traineeRec, bool) {
	if !s.isTrainee(trainee) {
		return traineeRec{}, false
	}
	rec := s.trainee[trainee]
	s.trainee[trainee] = traineeRec{}
	tr := &s.trainer[rec.trainer]
	if at, found := slices.BinarySearch(tr.trainees, trainee); found {
		tr.trainees = slices.Delete(tr.trainees, at, at+1)
	}
	if len(tr.trainees) == 0 {
		*tr = trainerRec{}
	}
	return rec, true
}

// grow makes the per-agent slices cover agent slot id.
func (s *skills) grow(id AgentID) {
	if n := int(id) + 1; n > len(s.trainee) {
		s.trainee = slices.Grow(s.trainee, n-len(s.trainee))[:n]
		s.trainer = slices.Grow(s.trainer, n-len(s.trainer))[:n]
		s.queuedIn = slices.Grow(s.queuedIn, n-len(s.queuedIn))[:n]
	}
}

// queued reports whether the agent waits in a traineeship queue.
func (s *skills) queued(id AgentID) bool {
	return id >= 0 && int(id) < len(s.queuedIn) && s.queuedIn[id] != 0
}

// rule is the entry of the queue as the pure rules read it.
func (ap applicant) rule() skill.Applicant {
	return skill.Applicant{Agent: int64(ap.agent), Applied: ap.applied, Served: ap.served}
}

// enqueue puts the applicant in the queue of the cell at its place in the queue
// order: ticks served first, then the application tick, then the agent
// (spec-skill-barriers.md 4.4, 4.6; skill.Requeue, which is the order of new
// applicants when none has served a tick). The caller has checked that the agent
// is in no queue.
func (s *skills) enqueue(cell int, ap applicant) {
	s.grow(ap.agent)
	q := s.queue[cell]
	at, _ := slices.BinarySearchFunc(q, ap, func(e, t applicant) int {
		switch {
		case skill.Requeue(e.rule(), t.rule()):
			return -1
		case skill.Requeue(t.rule(), e.rule()):
			return 1
		}
		return 0
	})
	s.queue[cell] = slices.Insert(q, at, ap)
	s.queuedIn[ap.agent] = int32(cell) + 1
}

// dequeue takes the agent out of the queue it waits in and reports whether it
// was in one.
func (s *skills) dequeue(id AgentID) bool {
	if !s.queued(id) {
		return false
	}
	cell := int(s.queuedIn[id]) - 1
	s.queuedIn[id] = 0
	q := s.queue[cell]
	for i := range q {
		if q[i].agent == id {
			s.queue[cell] = slices.Delete(q, i, i+1)
			return true
		}
	}
	invariant(fmt.Sprintf("agent %d is marked as queued but is in no queue (cell %d)", id, cell))
	return false
}

// teachingTick reports whether the agent is a trainer and this tick is a teaching
// tick of it (spec-skill-barriers.md 4.2).
func (s *skills) teachingTick(id AgentID) bool {
	return s.isTrainer(id) && s.trainer[id].teaching
}

// p22Teaching is P2.2 (spec-world.md 6, ruling C5): each trainer's teaching
// accumulator advances by the sum of its trainees' loads, in ascending agent ID,
// and a trainer whose accumulator reaches 10 000 is teaching this tick (4.2,
// 4.2.2; skill.Teach). The flag bars a start in P2.5 and P2.7, never a job in
// progress. It is called only while w.skills is not nil, and draws nothing.
func (w *World) p22Teaching() {
	s := w.skills
	for id := range s.trainer {
		tr := &s.trainer[id]
		if len(tr.trainees) == 0 {
			continue
		}
		sum, ok := checked.Mul(int64(len(tr.trainees)), w.cfg.Roles[w.agents[id].role].TeachingLoad)
		if !ok {
			sum = skill.Basis // beyond one full tick whatever the loads
		}
		tr.acc, tr.teaching = skill.Teach(tr.acc, sum)
	}
}

// skillsWork is the P2.5 step of an agent that the traineeships change, and reports
// whether it handled the agent. A trainer on a teaching tick starts nothing (a job
// in progress is not touched: P2.1 came first). A trainee runs no recipe of its
// own role: it asks for the first of its trainer's trainee-eligible recipes (in
// ascending recipe ID) that its trainer's account can run, one execution of its
// weight, the trainer as principal (spec-skill-barriers.md 4.2, spec-production.md
// 8.1, spec-agents.md 7.1.8). It draws nothing.
func (w *World) skillsWork(a *agent, region RegionID) bool {
	s := w.skills
	if s.teachingTick(a.id) {
		return true
	}
	trainer, ok := s.trainerOf(a.id)
	if !ok {
		return false
	}
	for _, rid := range w.roles.row(w.agents[trainer].role).recipes {
		if !w.cfg.Recipes[rid].TraineeEligible {
			continue
		}
		q := w.startReq(a, region, rid, a.weight)
		k0, _, err := w.prod.maxExecutions(w.ledger, &q)
		if err != nil {
			invariant("the ledger refused the count of a trainee's request: " + err.Error())
		}
		if k0 < 1 {
			continue
		}
		ask := jobAsk{recipe: rid, maximum: a.weight, set: true}
		ask.first, ask.draws = w.recordDraws(a, region, rid, k0)
		w.pr.asks[a.id] = ask
		return true
	}
	return true
}

// traineeRequest turns the request q of agent a into a trainee's: the inputs are
// taken from the trainer's account and the output goes to it, the trainer is the
// principal, and the effective factor carries TraineeYield (spec-production.md
// 4.3.2, 6.3, 8.1). It leaves the request of any other agent as it is.
func (w *World) traineeRequest(q *startReq, a *agent, region RegionID) {
	trainer, ok := w.skills.trainerOf(a.id)
	if !ok {
		return
	}
	cr := &w.cfg.Recipes[q.recipe]
	e, has, ok := w.yields.effectiveFor(&w.shocks, region, q.recipe, w.tk.moment.Season, cr.Sited, w.prod.abundance(region, q.recipe, false), cr.TraineeYield)
	if !ok {
		invariant("the effective yield of a trainee does not fit after validation")
	}
	t := &w.agents[trainer]
	q.account, q.principal, q.effective, q.hasProfile = t.account, t.id, e, has
}

// residueOwner is the agent whose residues a job of agent a carries: a trainee's job
// is the trainer's recipe, so it takes the trainer's residues of it, and the
// fraction of a unit stays with the principal (spec-production.md 6.6).
func (w *World) residueOwner(a *agent) *agent {
	if trainer, ok := w.skills.trainerOf(a.id); ok {
		return &w.agents[trainer]
	}
	return a
}

// jobAccount is the ledger account the output of the job of agent a goes to: the
// principal's, which is the agent's own except for a trainee's.
func (w *World) jobAccount(a *agent) int {
	if p := a.job.principal; p != a.id {
		return w.agents[p].account
	}
	return a.account
}

// boardAccount is the account the upkeep of agent a is consumed from at P6.1: the
// trainer's for a trainee (spec-agents.md 5.7, FR-AGT-068), else its own.
func (w *World) boardAccount(a *agent) int {
	if trainer, ok := w.skills.trainerOf(a.id); ok {
		return w.agents[trainer].account
	}
	return a.account
}

// valueBoard is the first step of P6.3 for the traineeships: the goods a trainee
// took from its trainer's account this tick, valued at the prices of the tick, are
// its in-kind receipt K_t and the same value is netted from the income of its
// trainer as an input is (spec-welfare-and-wages.md 2.1.6, 6.3, FR-WEL-030). The
// board is the upkeep of the households that were fed, so a boarded trainee's
// ratio is 10 000 and a shortfall of the trainer shows in both.
func (w *World) valueBoard() {
	s, nc := w.skills, len(w.cfg.Commodities)
	for id := range s.trainee {
		if !s.trainee[id].live {
			continue
		}
		a := &w.agents[id]
		g := w.regionOf(a)
		value := w.welfare.cost.tick(w.roles.row(a.role).upkeepIn(g), w.oc.valued[int(g)*nc:(int(g)+1)*nc], a.weight-a.unfed)
		w.welfare.addInKind(id, value)
		w.welfare.addInputSpend(int(s.trainee[id].trainer), value)
	}
}

// fillBoard sets the Board of the view of agent a: the upkeep of its trainees in
// the units of one tick, per commodity in commodity order (spec-agents.md 5.7).
// It is a no-op while w.skills is nil.
func (w *World) fillBoard(v *AgentView, a *agent) {
	s := w.skills
	if s == nil || !s.isTrainer(a.id) {
		return
	}
	s.board = s.board[:0]
	for _, id := range s.trainer[a.id].trainees {
		t := &w.agents[id]
		for _, it := range w.roles.row(t.role).upkeepIn(v.Region) {
			n, ok := checked.Mul(t.weight, int64(it.quantity))
			if !ok {
				invariant("a trainee's upkeep times its weight overflows after validation")
			}
			at, found := slices.BinarySearchFunc(s.board, it.commodity, func(r UpkeepRow, c CommodityID) int { return cmp.Compare(r.Commodity, c) })
			if !found {
				s.board = slices.Insert(s.board, at, UpkeepRow{Commodity: it.commodity})
			}
			if s.board[at].Quantity, ok = checked.Add(s.board[at].Quantity, n); !ok {
				invariant("the board of a trainer overflows after validation")
			}
		}
	}
	v.Board = s.board[:len(s.board):len(s.board)]
}
