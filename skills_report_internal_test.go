package economysim

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"
)

// loseConfig is trainingConfig with a trainee yield of 1 basis point: the trainee's
// output rounds to nothing and the recipe's input costs 1 000, so the gain of a new
// place is negative whatever the trainer's wage (the case of
// TestPlaceOpening_NegativeGainKeepsStarted).
func loseConfig() Config {
	c := trainingConfig(3)
	c.Recipes[1].TraineeYield = 1
	return c
}

// linesOf returns the lines of the capture and forgets them.
func linesOf(h capture) []logRec {
	out := *h.lines
	*h.lines = nil
	return out
}

// TestWarn_NoPlacesOpenTenthTick verifies the bounded Warn of the incentive
// cliff: with two applicants queued and a negative gain, "no places open" is
// logged once per region and role at the tenth consecutive tick and at the twentieth,
// never per applicant and never in between; the Debug line "traineeship refused"
// is once a tick for the role and names the head of the queue; nothing above
// Debug is logged but that Warn; the snapshot holds the counterpart (Queue 2, Places
// 3, the trainer); and a tick in which the gain turns positive and the queue is
// placed resets the run, so the next Warn is ten ticks after.
//
// Arithmetic: the weaver's one trainer holds 0 of MaxTrainees 3, so 3 places are
// open for 2 applicants, and only the gain keeps them waiting (loseConfig).
//
// Requirement: FR-SKL-022, FR-OBS-051
func TestWarn_NoPlacesOpenTenthTick(t *testing.T) {
	t.Parallel()
	h := newCapture()
	w := newMill(t, loseConfig(), WithSeed(exitSeed), WithAudit(), WithLogger(slog.New(h)))
	step(t, w, 1)
	c := castOf(t, w)
	a, b := c.homeMillers[0], c.homeMillers[1]
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: a, applied: 1})
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: b, applied: 1})
	linesOf(h)
	var warnAt []int
	for call := 1; call <= 25; call++ {
		if err := w.skillsEntry(0); err != nil {
			t.Fatalf("call %d: skillsEntry error = %v, want nil", call, err)
		}
		refused, warns := 0, 0
		for _, l := range linesOf(h) {
			switch {
			case l.msg == "traineeship refused":
				refused++
				if l.attrs["agent"] != int64(a) || l.attrs["region"] != 0 || l.attrs["role"] != 1 || l.strs["reason"] != "gain_negative" {
					t.Errorf("call %d: Debug line = %+v, want agent %d, region 0, role 1, reason gain_negative", call, l, a)
				}
			case l.msg == "no places open":
				warns++
				if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["role"] != 1 || l.strs["reason"] != "gain_negative" {
					t.Errorf("call %d: Warn = %+v, want level Warn, region 0, role 1, reason gain_negative", call, l)
				}
			case l.level > slog.LevelDebug:
				t.Errorf("call %d: line %q at level %v, want nothing above Debug but the one Warn", call, l.msg, l.level)
			}
		}
		if refused != 1 {
			t.Errorf("call %d: %d Debug lines of \"traineeship refused\", want 1 per region and role, not per applicant", call, refused)
		}
		if warns > 1 {
			t.Errorf("call %d: %d Warns, want at most 1 per region and role", call, warns)
		}
		if warns == 1 {
			warnAt = append(warnAt, call)
		}
	}
	if want := []int{10, 20}; !slices.Equal(warnAt, want) {
		t.Errorf("Warn at calls %v, want %v: the tenth tick and every tenth after", warnAt, want)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if row := s.Roles[1]; row.Queue != 2 || row.Places != 3 || row.Trainers != 1 || row.PlacesTaken != 0 {
		t.Errorf("snapshot row (home, weaver) = queue %d, places %d, trainers %d, taken %d, want 2, 3, 1, 0: the counterpart of the Warn", row.Queue, row.Places, row.Trainers, row.PlacesTaken)
	}
	if got := w.skills.warnRun[w.skills.cell(0, 1)]; got != 25 {
		t.Errorf("run after 25 blocked calls = %d, want 25", got)
	}
	w.cfg.Recipes[kneadOf(t, w)].TraineeYield = 10_000
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry error = %v, want nil", err)
	}
	if got := w.skills.warnRun[w.skills.cell(0, 1)]; got != 0 || len(w.skills.queue[w.skills.cell(0, 1)]) != 0 {
		t.Errorf("after the gain turned positive: run %d, queue %v, want 0 and the queue placed", got, queueIDs(w.skills, 0, 1))
	}
}

// TestWarn_NoTrainerIsDebugOnly verifies that a queue kept waiting for want of a
// trainer is a Debug line with the reason no_trainer and never the Warn, which
// FR-SKL-022 limits to a negative gain: the run stays 0.
//
// Guard: the spec's Warn covers the gain only; a queue stuck for want of a trainer
// is seen in the snapshot (Trainers 0, Queue above 0) and on this line, and widening
// the Warn is the owner's decision (plan 3-skl, Question list of PR B2).
//
// Requirement: FR-SKL-022, FR-SKL-019
func TestWarn_NoTrainerIsDebugOnly(t *testing.T) {
	t.Parallel()
	h := newCapture()
	w := newMill(t, trainingConfig(3), WithSeed(exitSeed), WithAudit(), WithLogger(slog.New(h)))
	step(t, w, 1)
	c := castOf(t, w)
	if err := w.removeAgent(&w.agents[c.homeWeaver], CauseConsumer); err != nil {
		t.Fatalf("removeAgent(weaver) error = %v, want nil", err)
	}
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: c.homeMillers[0], applied: 1})
	linesOf(h)
	for call := 1; call <= 12; call++ {
		if err := w.skillsEntry(0); err != nil {
			t.Fatalf("call %d: skillsEntry error = %v, want nil", call, err)
		}
		for _, l := range linesOf(h) {
			if l.level > slog.LevelDebug {
				t.Errorf("call %d: line %q at level %v, want Debug only for a queue with no trainer", call, l.msg, l.level)
			}
			if l.msg == "traineeship refused" && (l.strs["reason"] != "no_trainer" || l.attrs["agent"] != int64(c.homeMillers[0])) {
				t.Errorf("call %d: Debug line = %+v, want reason no_trainer for agent %d", call, l, c.homeMillers[0])
			}
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if row := s.Roles[1]; row.Trainers != 0 || row.Places != 0 || row.Queue != 1 {
		t.Errorf("snapshot row = trainers %d, places %d, queue %d, want 0, 0, 1: the only signal of a queue with no trainer", row.Trainers, row.Places, row.Queue)
	}
	if got := w.skills.warnRun[w.skills.cell(0, 1)]; got != 0 {
		t.Errorf("run = %d, want 0: no trainer is not a negative gain", got)
	}
}

// TestLog_TraineeshipEnded verifies the Debug line "traineeship ended" of both ways
// a traineeship ends without a completion: the removal of the trainer (reason
// trainer_left) and of the trainee (agent_left), with the trainee, the trainer, the
// role trained for, the region and the ticks served, which the event does not
// carry, and nothing above Debug.
//
// Requirement: FR-SKL-020, FR-OBS-020
func TestLog_TraineeshipEnded(t *testing.T) {
	t.Parallel()
	h := newCapture()
	w := newMill(t, trainingConfig(3), WithSeed(exitSeed), WithAudit(), WithLogger(slog.New(h)))
	step(t, w, 1)
	c := castOf(t, w)
	a, b, trainer := c.homeMillers[0], c.homeMillers[1], c.homeWeaver
	w.skills.start(a, trainer, 1, 2, 3)
	w.skills.start(b, trainer, 1, 3, 4)
	linesOf(h)
	if err := w.removeAgent(&w.agents[b], CauseConsumer); err != nil {
		t.Fatalf("removeAgent(trainee) error = %v, want nil", err)
	}
	if err := w.removeAgent(&w.agents[trainer], CauseConsumer); err != nil {
		t.Fatalf("removeAgent(trainer) error = %v, want nil", err)
	}
	var ended []logRec
	for _, l := range linesOf(h) {
		if l.msg == "traineeship ended" {
			ended = append(ended, l)
		}
		if l.level > slog.LevelDebug {
			t.Errorf("line %q at level %v, want Debug only", l.msg, l.level)
		}
	}
	want := []logRec{
		{msg: "traineeship ended", attrs: map[string]int64{"agent": int64(b), "trainer": int64(trainer), "region": 0, "role": 1, "served": 3}, strs: map[string]string{"reason": "agent_left"}},
		{msg: "traineeship ended", attrs: map[string]int64{"agent": int64(a), "trainer": int64(trainer), "region": 0, "role": 1, "served": 2}, strs: map[string]string{"reason": "trainer_left"}},
	}
	if len(ended) != 2 {
		t.Fatalf("lines of \"traineeship ended\" = %d, want 2", len(ended))
	}
	for i := range want {
		if ended[i].level != slog.LevelDebug || !reflect.DeepEqual(ended[i].attrs, want[i].attrs) || !reflect.DeepEqual(ended[i].strs, want[i].strs) {
			t.Errorf("line %d = %+v, want %+v at Debug", i, ended[i], want[i])
		}
	}
}

// TestSnapshot_SkillRows verifies the snapshot's view of the traineeships
// (spec-skill-barriers.md 8.2): per region and role the places open (the trainers'
// MaxTrainees less the trainees they hold), the places taken, the queue and the
// trainers, 0 for a role with no training and for Held while no module limits the
// headroom; the per-trainee rows by agent ID; and that Region(g) returns the same
// role rows as the full snapshot.
//
// Arithmetic: MaxTrainees is 3 and the home weaver holds 1 trainee, so 2 places are
// open; the yonder weaver holds none, so 3.
//
// Requirement: FR-SKL-017, FR-OBS-044
func TestSnapshot_SkillRows(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	w.skills.start(c.homeMillers[0], c.homeWeaver, 1, 2, 3)
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: c.homeMillers[1], applied: 4})
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	type cols struct{ places, taken, queue, trainers, held int64 }
	for _, tt := range []struct {
		name string
		row  RoleRow
		want cols
	}{
		{"home weaver", s.Roles[1], cols{2, 1, 1, 1, 0}},
		{"home miller (no training)", s.Roles[0], cols{}},
		{"yonder weaver", s.Roles[3], cols{3, 0, 0, 1, 0}},
		{"yonder miller (no training)", s.Roles[2], cols{}},
	} {
		got := cols{tt.row.Places, tt.row.PlacesTaken, tt.row.Queue, tt.row.Trainers, tt.row.Held}
		if got != tt.want {
			t.Errorf("%s: places, taken, queue, trainers, held = %+v, want %+v", tt.name, got, tt.want)
		}
	}
	want := []TraineeRow{{Agent: c.homeMillers[0], Trainer: c.homeWeaver, Role: 1, Region: 0, Served: 2, Applied: 3}}
	if !reflect.DeepEqual(s.Trainees, want) {
		t.Errorf("Trainees = %+v, want %+v", s.Trainees, want)
	}
	rs, err := w.Region(0)
	if err != nil {
		t.Fatalf("Region(0) error = %v, want nil", err)
	}
	if !reflect.DeepEqual(rs.Roles, s.Roles[:2]) {
		t.Errorf("Region(0).Roles = %+v, want the rows of the full snapshot %+v", rs.Roles, s.Roles[:2])
	}
}

// TestSnapshot_NoSkillRowsWhenNoRoleTrains verifies that a world with no role that
// trains has no trainee rows and 0 in every traineeship column.
//
// Guard: the columns are appended to a row that every pack has, so a world with
// none of the state must render as it did (architecture.md 2.10.5).
func TestSnapshot_NoSkillRowsWhenNoRoleTrains(t *testing.T) {
	t.Parallel()
	c := skillsConfig()
	c.Roles[1].TrainingTicks = 0
	w := newMill(t, c, WithSeed(exitSeed))
	step(t, w, 2)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if s.Trainees != nil {
		t.Errorf("Trainees = %+v, want nil", s.Trainees)
	}
	for i, row := range s.Roles {
		if row.Places != 0 || row.PlacesTaken != 0 || row.Queue != 0 || row.Trainers != 0 || row.Held != 0 {
			t.Errorf("row %d = %+v, want the traineeship columns at 0", i, row)
		}
	}
}

// TestEvents_TraineeshipFields verifies that the three events carry identifiers, the
// tick and the reason only: a trainee placed, completed and ended by its removal
// are seen with the region of the agent, after the transition (the state the event
// describes holds when the observer is called), and no event holds a map, a
// pointer, a slice or a function (FR-SKL-021, FR-OBS-018).
//
// Requirement: FR-SKL-021, FR-OBS-018
func TestEvents_TraineeshipFields(t *testing.T) {
	t.Parallel()
	var w *World
	seenAfter := map[string]bool{}
	var rec evLog
	obs := ObserverFunc(func(e Event) {
		rec.Observe(e)
		switch x := e.(type) {
		case TraineeshipStarted:
			seenAfter["started"] = w.skills.isTrainee(x.Trainee)
		case TraineeshipCompleted:
			seenAfter["completed"] = !w.skills.isTrainee(x.Agent)
		case TraineeshipEnded:
			seenAfter["ended"] = !w.skills.isTrainee(x.Agent)
		}
	})
	var c skillCast
	w, c = trainingWorld(t, 3, WithObserver(obs))
	a, b := c.homeMillers[0], c.homeMillers[1]
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: a, applied: 1})
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: b, applied: 1})
	for range 5 { // placed at call 1, served 4 at call 4, completed at call 5
		if err := w.skillsEntry(0); err != nil {
			t.Fatalf("skillsEntry error = %v, want nil", err)
		}
	}
	w.skills.start(c.homeMillers[2], c.homeWeaver, 1, 1, 2)
	if err := w.removeAgent(&w.agents[c.homeMillers[2]], CauseConsumer); err != nil {
		t.Fatalf("removeAgent error = %v, want nil", err)
	}
	if got := len(of[TraineeshipStarted](&rec)); got != 2 {
		t.Errorf("TraineeshipStarted events = %d, want 2", got)
	}
	st := of[TraineeshipStarted](&rec)[0]
	if st.Trainee != a || st.Trainer != c.homeWeaver || st.Role != 1 || st.Region != 0 || st.Tick != 1 {
		t.Errorf("TraineeshipStarted = %+v, want trainee %d, trainer %d, role 1, region 0, tick 1", st, a, c.homeWeaver)
	}
	if got := of[TraineeshipCompleted](&rec); len(got) != 2 || got[0].Agent != a || got[0].Region != 0 || got[0].From != 0 || got[0].To != 1 {
		t.Errorf("TraineeshipCompleted = %+v, want two, the first of agent %d in region 0 from role 0 to role 1", got, a)
	}
	if got := of[TraineeshipEnded](&rec); len(got) != 1 || got[0].Agent != c.homeMillers[2] || got[0].Region != 0 || got[0].Reason != ReasonAgentLeft {
		t.Errorf("TraineeshipEnded = %+v, want one, agent %d region 0 reason agent_left", got, c.homeMillers[2])
	}
	for name, ok := range seenAfter {
		if !ok {
			t.Errorf("event %s: the state it describes did not hold when the observer was called", name)
		}
	}
	for _, typ := range []reflect.Type{reflect.TypeFor[TraineeshipStarted](), reflect.TypeFor[TraineeshipCompleted](), reflect.TypeFor[TraineeshipEnded]()} {
		for i := range typ.NumField() {
			switch k := typ.Field(i).Type.Kind(); k {
			case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Func, reflect.Interface, reflect.Chan:
				t.Errorf("%s.%s is a %v, want plain data (observability-and-logging.md 4.1)", typ.Name(), typ.Field(i).Name, k)
			}
		}
	}
}

// TestEvents_OrderInEntryStepAndRemoval verifies the two orders of the traineeship
// events in one script: in a region's entry step a completion emits the abandon of
// its job, TraineeshipCompleted and the RoleChanged it causes, before the
// TraineeshipStarted of a place filled; and a trainer's removal emits, for each
// trainee in ascending agent ID, RecipeAbandoned (principal_left) then
// TraineeshipEnded (trainer_left), before the trainer's own events, and
// AgentBankrupted last.
//
// Requirement: FR-OBS-020, FR-OBS-021, FR-SKL-021
func TestEvents_OrderInEntryStepAndRemoval(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, c := trainingWorld(t, 3, WithObserver(rec))
	a, b, trainer := c.homeMillers[0], c.homeMillers[1], c.homeWeaver
	w.skills.start(a, trainer, 1, 4, 1)
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: b, applied: 2})
	before := len(rec.events)
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry error = %v, want nil", err)
	}
	var entry []string
	for _, e := range rec.events[before:] {
		switch x := e.(type) {
		case TraineeshipCompleted:
			entry = append(entry, "completed:"+idStr(int64(x.Agent)))
		case RoleChanged:
			entry = append(entry, "changed:"+idStr(int64(x.Agent)))
		case TraineeshipStarted:
			entry = append(entry, "started:"+idStr(int64(x.Trainee)))
		}
	}
	if want := []string{"completed:" + idStr(int64(a)), "changed:" + idStr(int64(a)), "started:" + idStr(int64(b))}; !slices.Equal(entry, want) {
		t.Errorf("entry step = %v, want %v", entry, want)
	}
	t.Run("a trainer's removal", func(t *testing.T) {
		rec := &evLog{}
		w, c := trainingWorld(t, 3, WithObserver(rec))
		m, trainer := c.homeMillers, c.homeWeaver
		for _, id := range []AgentID{m[2], m[0], m[1]} {
			w.skills.start(id, trainer, 1, 1, 3)
		}
		before := len(rec.events)
		if err := w.removeAgent(&w.agents[trainer], CauseConsumer); err != nil {
			t.Fatalf("removeAgent(trainer) error = %v, want nil", err)
		}
		var want []string
		for _, id := range m {
			want = append(want, "ended:"+idStr(int64(id))+":trainer_left")
		}
		want = append(want, "bankrupted:"+idStr(int64(trainer)), "estate:"+idStr(int64(trainer)))
		if got := endOrder(rec.events[before:]); !slices.Equal(got, want) {
			t.Errorf("removal of the trainer = %v, want %v: trainees in ascending ID, the trainer's own events after, AgentBankrupted the only event that removes the agent", got, want)
		}
	})
}

// TestCardinality_SkillReasons verifies that every reason a traineeship carries is
// in its closed set: the three events' Ended reasons (trainer_left, agent_left,
// role_removed, which no code reaches yet) and the log reasons of the place filling
// (no_trainer, gain_negative), so a metric attribute built from them is bounded by
// the configuration (observability-and-logging.md 5.4.1).
//
// Requirement: FR-SKL-021
func TestCardinality_SkillReasons(t *testing.T) {
	t.Parallel()
	for name, got := range map[string]Reason{
		"trainer_left": ReasonTrainerLeft, "agent_left": ReasonAgentLeft, "role_removed": ReasonRoleRemoved,
		"no_trainer": reasonNoTrainer, "gain_negative": reasonGainNegative,
	} {
		if string(got) != name {
			t.Errorf("reason constant for %q = %q, want %q", name, got, name)
		}
	}
}

// TestSkills_WorkIsCounted verifies the bound of the entry decision
// (NFR-SKL-002): the queue holds an agent at most once, so its length is bounded by
// the live agents and not by the households they stand for, and the placements and
// the course do not depend on the weight of an agent that takes no part: two
// worlds whose bystander group differs 100-fold in weight give the same
// traineeships, queue and snapshot columns after 20 ticks.
//
// Requirement: NFR-SKL-002
func TestSkills_WorkIsCounted(t *testing.T) {
	t.Parallel()
	run := func(scale int64) ([]TraineeRow, RoleRow, int) {
		c := trainingConfig(3)
		c.Groups[2].Households, c.Groups[2].Scale = 2*scale, scale // the yonder millers: 2 agents of weight scale
		w := newMill(t, c, WithSeed(exitSeed), WithAudit())
		step(t, w, 1)
		for _, id := range castOf2(t, w) {
			w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: id, applied: 1})
		}
		step(t, w, 20)
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		return s.Trainees, s.Roles[1], len(w.skills.queue[w.skills.cell(0, 1)])
	}
	t1, r1, q1 := run(1)
	t2, r2, q2 := run(100)
	if !reflect.DeepEqual(t1, t2) || r1.Places != r2.Places || r1.Queue != r2.Queue || r1.PlacesTaken != r2.PlacesTaken || q1 != q2 {
		t.Errorf("a bystander group of weight 100 changed the traineeships: %+v %+v %d against %+v %+v %d", t1, r1, q1, t2, r2, q2)
	}
	if q1 > 3 {
		t.Errorf("queue length = %d, want at most the 3 home millers: an agent is queued once", q1)
	}
}

// castOf2 returns the home millers of a trainingConfig world.
func castOf2(t testing.TB, w *World) []AgentID {
	t.Helper()
	var ids []AgentID
	w.eachLive(func(a *agent) {
		if w.regionOf(a) == 0 && a.role == 0 {
			ids = append(ids, a.id)
		}
	})
	return ids
}

// TestAlloc_SkillsNoOp verifies NFR-SKL-001: with the default logger and observer,
// the accumulator, the entry step with a queue kept waiting by the gain, the
// queue operations, the view fillers and a removal that touches no traineeship
// allocate nothing beyond the scratch the world holds.
//
// Not parallel: testing.AllocsPerRun is process-wide.
//
// Requirement: NFR-SKL-001, NFR-OBS-002
func TestAlloc_SkillsNoOp(t *testing.T) {
	w := newMill(t, loseConfig(), WithSeed(exitSeed))
	step(t, w, 1)
	c := castOf(t, w)
	s := w.skills
	s.start(c.homeMillers[0], c.homeWeaver, 1, 1, 1)
	s.enqueue(s.cell(0, 1), applicant{agent: c.homeMillers[1], applied: 1})
	s.enqueue(s.cell(0, 1), applicant{agent: c.homeMillers[2], applied: 2})
	s.dequeue(c.homeMillers[2]) // grows the slices once: the scratch of the world
	s.enqueue(s.cell(0, 1), applicant{agent: c.homeMillers[2], applied: 2})
	a := &w.agents[c.yonderMillers[0]]
	var row RegionAgent
	for _, tt := range []struct {
		name string
		f    func()
	}{
		{"the teaching accumulator", func() { w.p22Teaching() }},
		{"the entry step with a queue held by the gain", func() {
			if err := w.skillsEntry(0); err != nil {
				t.Fatalf("skillsEntry error = %v, want nil", err)
			}
			s.trainee[c.homeMillers[0]].served = 1 // the advance rose it: keep the state
		}},
		{"enqueue and dequeue", func() {
			s.enqueue(s.cell(1, 1), applicant{agent: a.id, applied: 3})
			s.dequeue(a.id)
		}},
		{"the view rows", func() { w.fillRoleSkills(0); w.fillAgentSkills(&row, a) }},
		{"the removal step of an agent in no traineeship", func() { w.dropAgent(a) }},
		{"the places of a role", func() { _ = w.trainersOf(0, 1) }},
	} {
		if n := testing.AllocsPerRun(50, tt.f); n != 0 {
			t.Errorf("AllocsPerRun(%s) = %v, want 0", tt.name, n)
		}
	}
}

// BenchmarkTeachingAndQueue reports the cost of the accumulator and of the entry
// step of a region with a queue held by the gain. It reports and never fails
// (testing.md 8.2): TestAlloc_SkillsNoOp holds the limit.
func BenchmarkTeachingAndQueue(b *testing.B) {
	w := newMill(b, loseConfig(), WithSeed(exitSeed))
	step(b, w, 1)
	c := castOf(b, w)
	w.skills.start(c.homeMillers[0], c.homeWeaver, 1, 1, 1)
	w.skills.enqueue(w.skills.cell(0, 1), applicant{agent: c.homeMillers[1], applied: 1})
	b.ReportAllocs()
	for b.Loop() {
		w.p22Teaching()
		if err := w.skillsEntry(0); err != nil {
			b.Fatalf("skillsEntry error = %v, want nil", err)
		}
		w.skills.trainee[c.homeMillers[0]].served = 1
	}
}
