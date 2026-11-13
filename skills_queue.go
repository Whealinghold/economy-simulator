package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/occupation"
	"github.com/Whealinghold/economy-sim/internal/skill"
)

// This file is the entry side of the traineeship (spec-skill-barriers.md 4.2.1,
// 4.4, 4.5; plan 3-skl Phases 4 to 6): the application and the queue, the
// places, the entry step of a region (P7.1(a): completion, placement, advance),
// the completion and the step of a removal. The state is skills.go, the reports
// skills_report.go. Nothing here draws a random number.

// completionHeadroom is the headroom a role has for the completion of a trainee in
// region g, the least of the class limit and the body's (spec-institutions.md
// 4.5.2). Neither exists before the settlement and institutions modules, so
// skills.headroom is nil and there is no limit: noLimit is true and a completion
// is never held. It is the slot those modules fill (plan 3-skl SK9), read again
// after each completion, so the headroom shared by the held trainees is spent in
// order.
func (w *World) completionHeadroom(g RegionID, r RoleID) (headroom int64, noLimit bool) {
	if w.skills.headroom != nil {
		return w.skills.headroom(g, r)
	}
	return 0, true
}

// skillsEntry is P7.1(a) of region g (spec-skill-barriers.md 4.2.1, spec-world.md
// ruling C5), in this order: every trainee whose ticks served have reached
// TrainingTicks completes, the places are filled from the queues, and the ticks
// served of every trainee rise by 1. The error is the ledger refusal of a
// completion's role change, which fails the step.
func (w *World) skillsEntry(g RegionID) error {
	s := w.skills
	if err := w.completeReady(g); err != nil {
		return err
	}
	for r := range w.cfg.Roles {
		if w.cfg.Roles[r].TrainingTicks > 0 && len(s.queue[s.cell(g, RoleID(r))]) > 0 {
			w.fillPlaces(g, RoleID(r))
		}
	}
	for id := range s.trainee {
		rec := &s.trainee[id]
		if rec.live && w.regionOf(&w.agents[id]) == g {
			rec.served = min(rec.served+1, w.cfg.Roles[rec.role].TrainingTicks)
		}
	}
	return nil
}

// completeReady completes the trainees of region g whose ticks served have reached
// TrainingTicks, in the order of application tick and then agent ID, each when its
// target role has headroom of at least its weight; the others stay trainees and
// are held (spec-skill-barriers.md 4.2.1).
func (w *World) completeReady(g RegionID) error {
	s := w.skills
	s.ready = s.ready[:0]
	for id := range s.trainee {
		if rec := &s.trainee[id]; rec.live && rec.served >= w.cfg.Roles[rec.role].TrainingTicks && w.regionOf(&w.agents[id]) == g {
			s.ready = append(s.ready, AgentID(id))
		}
	}
	slices.SortFunc(s.ready, func(a, b AgentID) int {
		return cmp.Or(cmp.Compare(s.trainee[a].applied, s.trainee[b].applied), cmp.Compare(a, b))
	})
	for _, id := range s.ready {
		rec, a := &s.trainee[id], &w.agents[id]
		headroom, noLimit := w.completionHeadroom(g, rec.role)
		if skill.Complete(rec.served, w.cfg.Roles[rec.role].TrainingTicks, headroom, a.weight, noLimit) != skill.Done {
			continue
		}
		d := Decision{Kind: DecideSwitch, Agent: a.id, Role: rec.role, Cause: CauseTraineeship}
		if err := w.changeRole(g, a, d, true); err != nil {
			return err
		}
	}
	return nil
}

// traineeshipCompleted ends the traineeship of agent a, which changes role to the
// target at the completion, and emits TraineeshipCompleted. It runs between the
// abandonment of the job (role_changed) and the RoleChanged it causes, which
// is the order of spec-observability.md 4.4.8.
func (w *World) traineeshipCompleted(a *agent, g RegionID, to RoleID) {
	if _, ok := w.skills.end(a.id); !ok {
		invariant("a traineeship completed for an agent that is no trainee")
	}
	if w.obs.supplied {
		w.obs.deliver(TraineeshipCompleted{Tick: w.Tick(), Agent: a.id, Region: g, From: a.role, To: to})
	}
}

// trainerSlot is a candidate trainer of a role in a region: its ID and the
// trainees it holds, in ascending ID order.
type trainerSlot struct {
	id   AgentID
	held int64
}

// canTrain reports whether agent a may be the trainer of a place of its role: a
// live household of weight 1 that is no trainee and is neither on a migration
// journey nor away (spec-skill-barriers.md 4.1.1, 4.6.1).
func (w *World) canTrain(a *agent) bool {
	return a.weight == 1 && !w.skills.isTrainee(a.id) && !w.onJourney(a) && !w.offStage(a)
}

// trainersOf lists the agents of role r in region g that can take a trainee, in
// ascending ID, with the trainees each holds.
func (w *World) trainersOf(g RegionID, r RoleID) []trainerSlot {
	var trainers []trainerSlot
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && a.role == r && w.regionOf(a) == g && w.canTrain(a) {
			trainers = append(trainers, trainerSlot{id: a.id, held: w.skills.held(a.id)})
		}
	}
	return trainers
}

// freest returns the index of the trainer with the fewest trainees, the lower ID
// first, and false when every trainer holds max.
func freest(trainers []trainerSlot, max int64) (int, bool) {
	best := -1
	for i, t := range trainers {
		if t.held < max && (best < 0 || t.held < trainers[best].held) {
			best = i
		}
	}
	return best, best >= 0
}

// fillPlaces fills the places of role r in region g from its queue, in queue order
// (spec-skill-barriers.md 4.2.1 step 2, 4.3.1). An applicant that is away or on a
// journey waits and the next is tried; the first applicant whose gain is
// negative, or the lack of a free place, ends the filling. The gain decides a new
// place only, never a traineeship in progress.
func (w *World) fillPlaces(g RegionID, r RoleID) {
	s, cell := w.skills, w.skills.cell(g, r)
	trainers, gain := w.trainersOf(g, r), w.newGainer(g, r)
	for i := 0; i < len(s.queue[cell]); {
		at, free := freest(trainers, w.cfg.Roles[r].MaxTrainees)
		if !free {
			return
		}
		ap := s.queue[cell][i]
		a := &w.agents[ap.agent]
		if w.onJourney(a) || w.offStage(a) {
			i++
			continue
		}
		if !gain.ok(a) {
			return
		}
		trainers[at].held++
		s.dequeue(ap.agent)
		s.start(ap.agent, trainers[at].id, r, ap.served, ap.applied)
		if w.obs.supplied {
			w.obs.deliver(TraineeshipStarted{Tick: w.Tick(), Trainee: ap.agent, Trainer: trainers[at].id, Role: r, Region: g})
		}
	}
}

// gainer decides whether a trainer takes a trainee in role r of region g
// (spec-skill-barriers.md 4.3): output_a - U_a - TeachingLoad * W_trainer /
// 10 000 >= 0, with output_a the best net value per tick of the role's
// trainee-eligible recipes at the region's published prices and TraineeYield
// (spec-production.md 8.3), U_a the trainee's upkeep at those prices and
// W_trainer the observed wage of the role. The prices and the wage are read once
// per call of fillPlaces, on the first applicant.
type gainer struct {
	w      *World
	g      RegionID
	r      RoleID
	loaded bool
	output int64
	wage   int64
}

// newGainer returns the gain test of role r in region g; it reads nothing yet.
func (w *World) newGainer(g RegionID, r RoleID) *gainer { return &gainer{w: w, g: g, r: r} }

// load reads the prices of the region, the observed wage of the role and the best
// eligible output.
func (gn *gainer) load() {
	w := gn.w
	gn.loaded = true
	w.pricesOf(gn.g)
	w.scanRegion(gn.g)
	w.fillRoles(gn.g)
	if sum := w.ex.sums[gn.r]; sum.HasWage {
		gn.wage = sum.Wage
	}
	found := false
	for _, rid := range w.roles.row(gn.r).recipes {
		cr := &w.cfg.Recipes[rid]
		if !cr.TraineeEligible {
			continue
		}
		if o, ok := w.eligibleOutput(rid, cr); ok && (!found || o > gn.output) {
			gn.output, found = o, true
		}
	}
}

// eligibleOutput is the net value per tick of a trainee running recipe rid at the
// prices of the region in w.ex.prices (skill.Output).
func (w *World) eligibleOutput(rid RecipeID, cr *Recipe) (int64, bool) {
	row := w.recipes.row(rid)
	var value, inputs int64
	for _, o := range row.outputs {
		if o.discard {
			continue
		}
		v, ok := checked.Mul(int64(o.quantity), w.ex.prices[o.commodity])
		if !ok {
			return 0, false
		}
		if value, ok = checked.Add(value, v); !ok {
			return 0, false
		}
	}
	for _, in := range row.inputs {
		v, ok := checked.Mul(int64(in.quantity), w.ex.prices[in.commodity])
		if !ok {
			return 0, false
		}
		if inputs, ok = checked.Add(inputs, v); !ok {
			return 0, false
		}
	}
	return skill.Output(value, inputs, cr.TraineeYield, cr.Duration)
}

// ok reports whether the gain of taking applicant a is not negative.
func (gn *gainer) ok(a *agent) bool {
	if !gn.loaded {
		gn.load()
	}
	w := gn.w
	g, ok := skill.Gain(gn.output, int64(w.ex.upkeep[a.role]), w.cfg.Roles[gn.r].TeachingLoad, gn.wage)
	return ok && g >= 0
}

// execApply is the executor of an Apply (spec-skill-barriers.md 4.5, 4.6.2): the
// agent joins the queue of the role in its region at the tick it applies. The
// refusals were checked before, and an apply changes no role, so it is no entry
// and no change in the counts of the region.
func (w *World) execApply(g RegionID, a *agent, d Decision) {
	w.skills.enqueue(w.skills.cell(g, d.Role), applicant{agent: a.id, applied: w.Tick()})
}

// applyFacts adds to the facts of a decision what the traineeship knows about
// its agent and role: whether the role has a traineeship, whether the agent is a
// trainee or a trainer with trainees, and whether it waits in a queue. It runs
// for every kind, since a switch or an entry into a training role is refused too.
func (w *World) applyFacts(f *occupation.Facts, a *agent, d *Decision) {
	s := w.skills
	if s == nil {
		return
	}
	f.TrainingRole = f.RoleKnown && w.cfg.Roles[d.Role].TrainingTicks > 0
	if a != nil {
		f.InTraineeship = s.isTrainee(a.id) || s.isTrainer(a.id)
		f.Queued = s.queued(a.id)
	}
}

// fillRoleSkills sets the traineeship fields of the view of every role of region g
// from the agents of the view already scanned: the applicants queued, the
// trainers that can take a trainee and the most trainees one may hold. A role
// with no training keeps 0.
func (w *World) fillRoleSkills(g RegionID) {
	s, v := w.skills, &w.ex.view
	for r := range v.Roles {
		if w.cfg.Roles[r].TrainingTicks > 0 {
			v.Roles[r].QueueAhead = int64(len(s.queue[s.cell(g, RoleID(r))]))
			v.Roles[r].MaxTrainees = w.cfg.Roles[r].MaxTrainees
		}
	}
	for i := range v.Agents {
		if a := &w.agents[v.Agents[i].Agent]; w.cfg.Roles[a.role].TrainingTicks > 0 && w.canTrain(a) {
			v.Roles[a.role].Trainers++
		}
	}
}

// fillAgentSkills sets the traineeship fields of the view row of agent a: whether
// it is a trainee or a trainer with trainees, whether it is queued and its
// surplus S_u, the observed income per tick above its upkeep (4.5).
func (w *World) fillAgentSkills(row *RegionAgent, a *agent) {
	s := w.skills
	row.Trainee = s.isTrainee(a.id)
	row.InTraineeship = row.Trainee || s.isTrainer(a.id)
	row.Queued = s.queued(a.id)
	ratio, present := a.history.Ratio(int64(row.Upkeep), w.cfg.Roles[a.role].DefaultRatio)
	if surplus, ok := skill.Surplus(ratio, present, int64(row.Upkeep)); ok {
		row.Surplus = Money(surplus)
	}
}

// dropAgent (a no-op while w.skills is nil) is the step of a removal that belongs to the traineeships
// (spec-agents.md 3.6 step 2, spec-skill-barriers.md 4.6; FR-SPN-011: a removal is
// never refused for a traineeship). It runs before the removal's own events:
//   - an applicant leaves its queue;
//   - a trainer's trainees, in ascending agent ID, each lose their job in progress
//     (RecipeAbandoned, principal_left), end their traineeship (TraineeshipEnded,
//     trainer_left) and rejoin the queue of the role they trained for with the
//     ticks they have served;
//   - a trainee loses its job in progress (agent_removed, which the removal would
//     do next) and its traineeship ends (TraineeshipEnded, agent_left).
//
// It draws nothing.
func (w *World) dropAgent(a *agent) {
	s, region := w.skills, w.regionOf(a)
	if s == nil {
		return
	}
	s.dequeue(a.id)
	if s.isTrainer(a.id) {
		for _, id := range slices.Clone(s.trainer[a.id].trainees) {
			w.returnTrainee(&w.agents[id], region)
		}
	}
	if s.isTrainee(a.id) {
		if a.hasJob() {
			w.abandonJob(a, region, ReasonAgentRemoved)
		}
		s.end(a.id)
		if w.obs.supplied {
			w.obs.deliver(TraineeshipEnded{Tick: w.Tick(), Agent: a.id, Region: region, Reason: ReasonAgentLeft})
		}
	}
}

// returnTrainee ends the traineeship of trainee t because its trainer is removed
// and queues it again for the role it trained for, keeping the ticks served and
// the tick it applied (spec-skill-barriers.md 4.6; the queue order is skill.Requeue).
func (w *World) returnTrainee(t *agent, region RegionID) {
	if t.hasJob() {
		w.abandonJob(t, region, ReasonPrincipalLeft)
	}
	rec, _ := w.skills.end(t.id)
	if w.obs.supplied {
		w.obs.deliver(TraineeshipEnded{Tick: w.Tick(), Agent: t.id, Region: region, Reason: ReasonTrainerLeft})
	}
	w.skills.enqueue(w.skills.cell(region, rec.role), applicant{agent: t.id, applied: rec.applied, served: rec.served})
}
