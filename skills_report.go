package economysim

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/skill"
)

// This file is the reports of the skill barriers: the logs and the snapshot rows
// of spec-skill-barriers.md 8, and the checkpoint's encoder and decoder of tag 10
// with its cross-check (spec-checkpoint.md 4.2 row 10, F7; spec-skill-barriers.md
// 9). The three events are built where the transitions are (skills_queue.go),
// under w.obs.supplied. Nothing here draws a random number, and nothing above
// Debug is logged except the one bounded Warn.

// reportPlaces advances the run of the Warn "no places open" of (region g, role r)
// and logs what stopped the filling of its places (spec-skill-barriers.md 8.3,
// FR-SKL-022). The run counts the consecutive ticks in which an applicant was
// kept waiting by a negative gain, and warns at the tenth and every tenth after,
// once per region and role, never per applicant; a stop for want of a free place
// is a Debug line only, with its counterpart in the snapshot (Places, Queue,
// Trainers). The Debug line "traineeship refused" is once per region and role and
// tick, naming the applicant it ended on.
func (w *World) reportPlaces(g RegionID, r RoleID, stop placeStop, head AgentID) {
	cell := w.skills.cell(g, r)
	run, warn := skill.NextRun(w.skills.warnRun[cell], stop == placeGain)
	w.skills.warnRun[cell] = run
	if stop == placeOpen {
		return
	}
	reason := reasonNoTrainer
	if stop == placeGain {
		reason = reasonGainNegative
	}
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "traineeship refused", slog.Int64(logkeys.Agent, int64(head)),
			slog.Int64(logkeys.Region, int64(g)), slog.Int64(logkeys.Role, int64(r)), slog.String(logkeys.Reason, string(reason)))
	}
	if warn && w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "no places open", slog.Int64(logkeys.Region, int64(g)),
			slog.Int64(logkeys.Role, int64(r)), slog.String(logkeys.Reason, string(reason)))
	}
}

// reportEnded reports the end of the traineeship rec of trainee id without a
// completion: the event TraineeshipEnded, which carries the reason and the
// region, and the Debug line "traineeship ended", which carries what the event
// does not (the trainer, the role trained for and the ticks served), so that a
// run that never completes can be told from one whose trainers leave late
// (spec-skill-barriers.md 8.3).
func (w *World) reportEnded(id AgentID, region RegionID, rec traineeRec, reason Reason) {
	if w.obs.supplied {
		w.obs.deliver(TraineeshipEnded{Tick: w.Tick(), Agent: id, Region: region, Reason: reason})
	}
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "traineeship ended", slog.Int64(logkeys.Agent, int64(id)),
			slog.Int64(logkeys.Trainer, int64(rec.trainer)), slog.Int64(logkeys.Region, int64(region)),
			slog.Int64(logkeys.Role, int64(rec.role)), slog.Int64(logkeys.Served, rec.served),
			slog.String(logkeys.Reason, string(reason)))
	}
}

// TraineeRow is a traineeship in progress, one per trainee in ascending agent ID
// (spec-skill-barriers.md 8.2): the trainee, its trainer, the role it trains for,
// the region, the ticks served and the tick it applied. Empty while no role
// trains or nobody is a trainee.
type TraineeRow struct {
	Agent, Trainer  AgentID
	Role            RoleID
	Region          RegionID
	Served, Applied int64
}

// snapshotSkills fills the trainee rows of a snapshot. It is a no-op while
// w.skills is nil.
func (w *World) snapshotSkills(s *Snapshot) {
	sk := w.skills
	if sk == nil {
		return
	}
	for id, rec := range sk.trainee {
		if rec.live {
			s.Trainees = append(s.Trainees, TraineeRow{Agent: AgentID(id), Trainer: rec.trainer, Role: rec.role,
				Region: w.regionOf(&w.agents[id]), Served: rec.served, Applied: rec.applied})
		}
	}
}

// skillsRoleRows fills the traineeship columns of the role rows of region g, by
// RoleID: the places the trainers have open, the trainees in training for the
// role, the queue and the trainers that can take a trainee (spec-skill-barriers.md
// 8.2). A role with no training keeps 0, and so does Held until the headroom of
// a role is limited by a module (completionHeadroom).
func (w *World) skillsRoleRows(rows []RoleRow, g RegionID) {
	s := w.skills
	for r := range rows {
		if w.cfg.Roles[r].TrainingTicks <= 0 {
			continue
		}
		slots := w.trainersOf(g, RoleID(r))
		s.open = s.open[:0]
		for _, t := range slots {
			s.open = append(s.open, skill.Trainer{Held: t.held, Max: t.max})
		}
		rows[r].Places, rows[r].Trainers = skill.Open(s.open), int64(len(slots))
		rows[r].Queue = int64(len(s.queue[s.cell(g, RoleID(r))]))
	}
	for id, rec := range s.trainee {
		if rec.live && w.regionOf(&w.agents[id]) == g {
			rows[rec.role].PlacesTaken++
		}
	}
}

// saveSkills is tag 10 (spec-checkpoint.md 4.2): four counts, each followed by its
// rows. The trainees in ascending agent ID (agent, trainer, target role, ticks
// served, application tick); the trainers that have a trainee, in ascending agent
// ID (agent, teaching accumulator, then a count and the trainees in ascending
// ID); the (region, role) cells that have a queue, in region then role order
// (region, role, then a count and each applicant in queue order: agent,
// application tick, ticks served); and the cells with a Warn run (region, role,
// run). A world with no role that trains, or with nothing held, writes four
// counts of 0.
func (w *World) saveSkills(cw *codec.Writer) error {
	s := w.skills
	if s == nil {
		for range 4 {
			cw.Count(0)
		}
		return nil
	}
	s.saveTrainees(cw)
	s.saveTrainers(cw)
	s.saveQueues(cw)
	writeSparse(cw, s.warnRun, s.roles)
	return nil
}

// saveTrainees writes the first count of tag 10: the trainees in ascending agent ID.
func (s *skills) saveTrainees(cw *codec.Writer) {
	n := 0
	for i := range s.trainee {
		if s.trainee[i].live {
			n++
		}
	}
	cw.Count(n)
	for i, rec := range s.trainee {
		if rec.live {
			cw.U(uint64(i))
			cw.U(uint64(rec.trainer))
			cw.U(uint64(rec.role))
			cw.U(uint64(rec.served))
			cw.U(uint64(rec.applied))
		}
	}
}

// saveTrainers writes the second count of tag 10: the trainers that have a trainee.
func (s *skills) saveTrainers(cw *codec.Writer) {
	n := 0
	for i := range s.trainer {
		if len(s.trainer[i].trainees) > 0 {
			n++
		}
	}
	cw.Count(n)
	for i, tr := range s.trainer {
		if len(tr.trainees) == 0 {
			continue
		}
		cw.U(uint64(i))
		cw.U(uint64(tr.acc))
		cw.Count(len(tr.trainees))
		for _, t := range tr.trainees {
			cw.U(uint64(t))
		}
	}
}

// saveQueues writes the third count of tag 10: the cells with a queue and their
// applicants.
func (s *skills) saveQueues(cw *codec.Writer) {
	n := 0
	for _, q := range s.queue {
		if len(q) > 0 {
			n++
		}
	}
	cw.Count(n)
	for c, q := range s.queue {
		if len(q) == 0 {
			continue
		}
		cw.U(uint64(c / s.roles))
		cw.U(uint64(c % s.roles))
		cw.Count(len(q))
		for _, ap := range q {
			cw.U(uint64(ap.agent))
			cw.U(uint64(ap.applied))
			cw.U(uint64(ap.served))
		}
	}
}

// loadSkills is tag 10. A world with no role that trains must have four counts of
// 0. Otherwise it reads the four counts in order, checking every reference
// against the agents and the configuration; the edges between the trainees and
// the trainers are checked together in step 7 (skillsCross, F7).
func (w *World) loadSkills(rs *restoreCtx, r *codec.Reader) error {
	s := w.skills
	if s == nil {
		for range 4 {
			if r.Count(1) != 0 {
				r.Failf("rows in the skills section, whose roles have no training")
			}
		}
		return nil
	}
	w.readTrainees(rs, r)
	w.readTrainers(r)
	w.readQueues(rs, r)
	readSparse(r, s.warnRun, len(w.cfg.Regions), s.roles, rs.tick)
	if r.Err() == nil {
		for c, run := range s.warnRun {
			if run != 0 && w.cfg.Roles[c%s.roles].TrainingTicks <= 0 {
				r.Failf("a Warn run for role %d, which has no training", c%s.roles)
			}
		}
	}
	return nil
}

// skillAgent returns the live agent id of weight 1, or nil after recording the
// failure on r: a trainer, a trainee and an applicant are agents of weight 1
// (spec-skill-barriers.md 4.1.1).
func (w *World) skillAgent(r *codec.Reader, id uint64, what string) *agent {
	if id < uint64(w.firstHousehold) || id >= uint64(len(w.agents)) || w.agents[id].weight != 1 {
		r.Failf("%s %d is not a live agent of weight 1", what, id)
		return nil
	}
	return &w.agents[id]
}

// readTrainees reads the first count of tag 10. A row is strictly above the one
// before; the target role trains, the trainee is not in it and is in its trainer's
// region, and the ticks served do not pass the course.
func (w *World) readTrainees(rs *restoreCtx, r *codec.Reader) {
	s := w.skills
	n := r.Count(5)
	prev := uint64(0)
	for i := range n {
		id := r.Strict(prev, i > 0)
		trainer, role := r.U(), r.Index(uint64(s.roles))
		if r.Err() != nil {
			return
		}
		prev = id
		ticks := w.cfg.Roles[role].TrainingTicks
		served, applied := r.Bounded("ticks served", ticks), r.Bounded("application tick", rs.tick)
		if r.Err() != nil || !w.traineeRowOK(r, id, trainer, RoleID(role), ticks) {
			return
		}
		s.grow(max(AgentID(id), AgentID(trainer)))
		s.trainee[id] = traineeRec{trainer: AgentID(trainer), role: RoleID(role), served: served, applied: applied, live: true}
	}
}

// traineeRowOK checks the references of a trainee row, recording the failure on r:
// both are live agents of weight 1, the course exists, the trainer holds the role
// the trainee trains for, the trainee does not, and they are in one region.
func (w *World) traineeRowOK(r *codec.Reader, id, trainer uint64, role RoleID, ticks int64) bool {
	a, t := w.skillAgent(r, id, "trainee"), w.skillAgent(r, trainer, "trainer")
	if a == nil || t == nil {
		return false
	}
	if ticks <= 0 || a.role == role || t.role != role || id == trainer {
		r.Failf("trainee %d has a role or trainer that cannot hold a traineeship", id)
		return false
	}
	if w.regionOf(a) != w.regionOf(t) {
		r.Failf("trainee %d is not in the region of its trainer %d", id, trainer)
		return false
	}
	return true
}

// readTrainers reads the second count of tag 10: each trainer with at least one
// trainee, at most its role's MaxTrainees (an institution may lower a body's
// maximum under a traineeship, never raise the role's), its accumulator in
// [0, 9 999] and its trainees strictly ascending. The trainer is not itself a
// trainee.
func (w *World) readTrainers(r *codec.Reader) {
	s := w.skills
	n := r.Count(3)
	prev := uint64(0)
	for i := range n {
		id := r.Strict(prev, i > 0)
		acc := r.Bounded("teaching accumulator", skill.MaxAccumulator)
		m := r.Count(1)
		if r.Err() != nil {
			return
		}
		prev = id
		t := w.skillAgent(r, id, "trainer")
		if t == nil {
			return
		}
		if m == 0 || s.isTrainee(AgentID(id)) {
			r.Failf("trainer %d has no trainee or is a trainee", id)
			return
		}
		if int64(m) > w.cfg.Roles[t.role].MaxTrainees {
			r.Failf("trainer %d holds %d trainees, above MaxTrainees %d of its role", id, m, w.cfg.Roles[t.role].MaxTrainees)
			return
		}
		list := make([]AgentID, 0, m)
		last := uint64(0)
		for j := range m {
			t := r.Strict(last, j > 0)
			if t >= uint64(len(w.agents)) {
				r.Failf("trainer %d lists %d, which is not an agent", id, t)
				return
			}
			list, last = append(list, AgentID(t)), t
		}
		if r.Err() != nil {
			return
		}
		s.grow(AgentID(id))
		s.trainer[id] = trainerRec{acc: acc, trainees: list}
	}
}

// readQueues reads the third count of tag 10: the cells in ascending (region,
// role) order, each with at least one applicant. An applicant is a live agent of
// weight 1 in the cell's region and not in its role, not a trainee, and queued
// once in the whole section; its ticks served do not pass the course. The queue
// is in the order skill.Requeue gives, as enqueue keeps it.
func (w *World) readQueues(rs *restoreCtx, r *codec.Reader) {
	s := w.skills
	regions := len(w.cfg.Regions)
	seen := make([]bool, len(w.agents))
	n := r.Count(4)
	prev := uint64(0)
	for i := range n {
		region, role := r.Index(uint64(regions)), r.Index(uint64(s.roles))
		key := region*uint64(s.roles) + role
		m := r.Count(3)
		if r.Err() != nil {
			return
		}
		if (i > 0 && key <= prev) || m == 0 || w.cfg.Roles[role].TrainingTicks <= 0 {
			r.Failf("queue row (region %d, role %d) is out of order, empty or for a role with no training", region, role)
			return
		}
		prev = key
		ticks := w.cfg.Roles[role].TrainingTicks
		for range m {
			ap, ok := w.readApplicant(rs, r, seen, RegionID(region), RoleID(role), ticks)
			if !ok {
				return
			}
			if !queueOrdered(r, s.queue[key], ap, RegionID(region), RoleID(role)) {
				return
			}
			s.queue[key] = append(s.queue[key], ap)
			s.grow(ap.agent)
			s.queuedIn[ap.agent] = int32(key) + 1
		}
	}
}

// readApplicant reads one queue entry (agent, application tick, ticks served) of
// the cell (region, role) and checks it, marking the agent in seen. It reports
// false after recording the failure on r.
func (w *World) readApplicant(rs *restoreCtx, r *codec.Reader, seen []bool, region RegionID, role RoleID, ticks int64) (applicant, bool) {
	id := r.U()
	applied, served := r.Bounded("application tick", rs.tick), r.Bounded("ticks served", ticks)
	if r.Err() != nil {
		return applicant{}, false
	}
	a := w.skillAgent(r, id, "applicant")
	switch {
	case a == nil:
		return applicant{}, false
	case seen[id] || w.skills.isTrainee(AgentID(id)) || w.skills.isTrainer(AgentID(id)):
		r.Failf("applicant %d is queued twice, is a trainee or is a trainer with trainees", id)
		return applicant{}, false
	case w.regionOf(a) != region || a.role == role:
		r.Failf("applicant %d is not in region %d or already holds role %d", id, region, role)
		return applicant{}, false
	}
	seen[id] = true
	return applicant{agent: AgentID(id), applied: applied, served: served}, true
}

// skillsCross is the cross-check of F7 (spec-checkpoint.md 4.2, 6.2 step 7): a
// trainee's trainer lists it, and every trainee a trainer lists is a trainee of
// that trainer. An edge in one section and not the other is corrupt
// (FR-CHK-023).
func (w *World) skillsCross() error {
	s := w.skills
	for id, rec := range s.trainee {
		if !rec.live {
			continue
		}
		if _, found := slices.BinarySearch(s.trainer[rec.trainer].trainees, AgentID(id)); !found {
			return fmt.Errorf("%w: trainee %d is not listed by its trainer %d", ErrCheckpointCorrupt, id, rec.trainer)
		}
	}
	for id, tr := range s.trainer {
		for _, t := range tr.trainees {
			if !s.isTrainee(t) || s.trainee[t].trainer != AgentID(id) {
				return fmt.Errorf("%w: trainer %d lists %d, which is not its trainee", ErrCheckpointCorrupt, id, t)
			}
		}
	}
	return nil
}

// queueOrdered reports whether applicant ap comes after the last of q in the order
// of skill.Requeue, recording the failure on r when it does not.
func queueOrdered(r *codec.Reader, q []applicant, ap applicant, region RegionID, role RoleID) bool {
	if len(q) == 0 || skill.Requeue(q[len(q)-1].rule(), ap.rule()) {
		return true
	}
	r.Failf("applicant %d is not after applicant %d in the queue order of cell (region %d, role %d)", ap.agent, q[len(q)-1].agent, region, role)
	return false
}
