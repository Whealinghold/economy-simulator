package economysim

import "slices"

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
	return int(id) < len(s.trainee) && s.trainee[id].live
}

// trainerOf returns the trainer of the agent when it is a trainee.
func (s *skills) trainerOf(id AgentID) (AgentID, bool) {
	if !s.isTrainee(id) {
		return 0, false
	}
	return s.trainee[id].trainer, true
}

// isTrainer reports whether the agent has at least one trainee.
func (s *skills) isTrainer(id AgentID) bool {
	return int(id) < len(s.trainer) && len(s.trainer[id].trainees) > 0
}

// holds is the test of the region_bound obligation in_traineeship: a trainee, or a
// trainer with trainees, unless skip names the instance by key
// "in_traineeship:<agent>" or the obligation by name (spec-regions.md 6.5.2).
func (s *skills) holds(a *agent, skip []string) bool {
	return (s.isTrainee(a.id) || s.isTrainer(a.id)) && !skipHas(skip, inTraineeshipName, int64(a.id))
}

// start records a traineeship: the trainee joins the trainer's list in ascending
// agent ID and the trainer, if it had no trainee, starts with a teaching
// accumulator of 0. A trainee that already is one, or that is its own trainer, is
// a violated invariant of the caller: the queue admits an agent once and takes it
// out of the queue first.
func (s *skills) start(trainee, trainer AgentID, role RoleID, served, applied int64) {
	if trainee == trainer || s.isTrainee(trainee) || s.isTrainee(trainer) {
		invariant("a traineeship was started for an agent that is a trainee or its own trainer")
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
	}
}

// p22Teaching is P2.2 (spec-world.md 6, ruling C5): each trainer's teaching
// accumulator advances by the sum of its trainees' loads and a trainer whose
// accumulator reaches 10 000 is teaching this tick. It is called only while
// w.skills is not nil. The surface of plan 3-skl Phase 3 declares the position and
// no trainee can exist yet, so the step has nothing to advance; the accumulator
// and the flag join it with the living traineeship.
func (w *World) p22Teaching() {}
