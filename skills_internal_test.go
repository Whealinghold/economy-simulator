package economysim

import (
	"bytes"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// skillsConfig is exitConfig on two regions with every household an agent of
// weight 1: in home three millers and one weaver, in yonder two millers and one
// weaver. The weaver role trains (4 ticks), so the state exists; the millers do
// not.
func skillsConfig() Config {
	c := exitTrainingConfig()
	r := DefaultRegion()
	r.ID = "yonder"
	c.Regions = append(c.Regions, r)
	c.Groups = []Group{
		{Region: "home", Role: "miller", Households: 3, Scale: 1},
		{Region: "home", Role: "weaver", Households: 1},
		{Region: "yonder", Role: "miller", Households: 2, Scale: 1},
		{Region: "yonder", Role: "weaver", Households: 1},
	}
	return c
}

// skillCast is the agents of skillsConfig by region and role, in ascending ID.
type skillCast struct {
	homeMillers, yonderMillers []AgentID
	homeWeaver, yonderWeaver   AgentID
}

// castOf reads the agents of a skillsConfig world, failing if one is gone or has
// a weight other than 1.
func castOf(t testing.TB, w *World) skillCast {
	t.Helper()
	var c skillCast
	w.eachLive(func(a *agent) {
		home := w.regionOf(a) == 0
		switch {
		case a.weight != 1:
			t.Fatalf("agent %d has weight %d, want 1", a.id, a.weight)
		case home && a.role == 0:
			c.homeMillers = append(c.homeMillers, a.id)
		case home:
			c.homeWeaver = a.id
		case a.role == 0:
			c.yonderMillers = append(c.yonderMillers, a.id)
		default:
			c.yonderWeaver = a.id
		}
	})
	if len(c.homeMillers) != 3 || len(c.yonderMillers) != 2 || c.homeWeaver == 0 || c.yonderWeaver == 0 {
		t.Fatalf("cast = %+v, want 3 home millers, 2 yonder millers and a weaver in each region", c)
	}
	return c
}

// skillsWorld builds the world of skillsConfig and steps it for ticks, so that
// application ticks and runs up to ticks are legal in a section.
func skillsWorld(t testing.TB, ticks int) (*World, skillCast) {
	t.Helper()
	w := newMill(t, skillsConfig(), WithSeed(exitSeed), WithAudit())
	step(t, w, ticks)
	return w, castOf(t, w)
}

// plantSkills plants a state with something in each of the four counts: home
// miller 0 is a trainee of the home weaver (2 ticks served, applied at tick 3);
// home millers 1 and 2 queue for home weaver (applied 4, 1 tick served, and
// applied 4); the first yonder miller queues for yonder weaver (3 ticks served,
// applied at 2); and the Warn run of (yonder, weaver) is 7.
func plantSkills(w *World, c skillCast) {
	s := w.skills
	s.start(c.homeMillers[0], c.homeWeaver, 1, 2, 3)
	s.trainer[c.homeWeaver].acc = 4_500
	s.queue[s.cell(0, 1)] = []applicant{{c.homeMillers[1], 4, 1}, {c.homeMillers[2], 4, 0}}
	s.queue[s.cell(1, 1)] = []applicant{{c.yonderMillers[0], 2, 3}}
	s.warnRun[s.cell(1, 1)] = 7
}

// section10 returns the body tag 10 saves for w.
func section10(t testing.TB, w *World) []byte {
	t.Helper()
	cw := codec.NewWriter(nil)
	if err := w.saveSkills(cw); err != nil {
		t.Fatalf("saveSkills() error = %v, want nil", err)
	}
	return cw.Out()
}

// TestWorld_SkillsNilWithoutTrainingRole verifies that the state exists exactly
// when some role has TrainingTicks above 0, and registers its obligation only
// then.
//
// Guard: the skill barriers have no Modules pointer (spec-config.md 3.7.2), so
// "on" is read from the roles; a state built for every world would cost the worlds
// of the other packs and register an obligation they do not have.
//
// Requirement: FR-REG-060
func TestWorld_SkillsNilWithoutTrainingRole(t *testing.T) {
	t.Parallel()
	off := newMill(t, panConfig(), WithSeed(1))
	if off.skills != nil || len(off.bounds.rows) != 0 {
		t.Errorf("world with no training role: skills = %v, obligations = %d, want nil and 0", off.skills, len(off.bounds.rows))
	}
	on := newMill(t, skillsConfig(), WithSeed(1))
	if on.skills == nil || len(on.skills.queue) != 2*2 || len(on.skills.warnRun) != 2*2 {
		t.Fatalf("world with a training role: skills = %+v, want a state with 2 regions x 2 roles cells", on.skills)
	}
	if n := len(on.bounds.rows); n != 1 || on.bounds.rows[0].err.Obligation != inTraineeshipName {
		t.Errorf("obligations of a world with a training role = %d, want only %q", n, inTraineeshipName)
	}
}

// TestRefusal_InTraineeshipBlocksRelocate verifies that a trainee, and a trainer
// with trainees, are refused a relocation with the region_bound obligation
// in_traineeship, that a skip by name or by instance key lifts it, that another
// agent is not refused, and that the refusal ends with the traineeship.
//
// Requirement: FR-REG-060, FR-REG-048, FR-SKL-010
func TestRefusal_InTraineeshipBlocksRelocate(t *testing.T) {
	t.Parallel()
	w, c := skillsWorld(t, 1)
	trainee, trainer, other := c.homeMillers[0], c.homeWeaver, c.homeMillers[1]
	w.skills.start(trainee, trainer, 1, 0, 0)
	for _, tt := range []struct {
		name    string
		agent   AgentID
		skip    []string
		refused bool
	}{
		{"trainee", trainee, nil, true},
		{"trainer with a trainee", trainer, nil, true},
		{"another agent", other, nil, false},
		{"trainee, skipped by name", trainee, []string{inTraineeshipName}, false},
		{"trainee, skipped by its own key", trainee, []string{instance(trainee)}, false},
		{"trainee, skipped by another agent's key", trainee, []string{instance(other)}, true},
		{"trainer, skipped by its own key", trainer, []string{instance(trainer)}, false},
	} {
		reason, err := w.RelocationRefusal(tt.agent, 1, tt.skip...)
		var bound *RegionBoundError
		got := errors.As(err, &bound) && bound.Obligation == inTraineeshipName && reason == string(ReasonRegionBound)
		if got != tt.refused {
			t.Errorf("%s: RelocationRefusal(%d, 1, %q) = %q, %v, want refused %v", tt.name, tt.agent, tt.skip, reason, err, tt.refused)
		}
	}
	w.skills.end(trainee)
	if reason, err := w.RelocationRefusal(trainee, 1); reason != "" || err != nil {
		t.Errorf("after the end, RelocationRefusal(trainee) = %q, %v, want none", reason, err)
	}
	if reason, err := w.RelocationRefusal(trainer, 1); reason != "" || err != nil {
		t.Errorf("after the last trainee left, RelocationRefusal(trainer) = %q, %v, want none", reason, err)
	}
}

// instance is the skip key of an in_traineeship instance (spec-regions.md 6.1.6).
func instance(id AgentID) string { return inTraineeshipName + ":" + strconv.Itoa(int(id)) }

// TestSkills_StartEndKeepTrainersInOrder verifies the state operations the codec
// and the queue share: trainees join a trainer in ascending ID, the accumulator
// of a trainer is 0 at its first trainee and dropped with its last, and a second
// traineeship for a trainee is a violated invariant.
//
// Requirement: FR-SKL-015
func TestSkills_StartEndKeepTrainersInOrder(t *testing.T) {
	t.Parallel()
	w, c := skillsWorld(t, 1)
	s := w.skills
	s.start(c.homeMillers[2], c.homeWeaver, 1, 0, 1)
	s.start(c.homeMillers[0], c.homeWeaver, 1, 0, 1)
	if got, want := s.trainer[c.homeWeaver].trainees, []AgentID{c.homeMillers[0], c.homeMillers[2]}; !slices.Equal(got, want) {
		t.Errorf("trainees = %v, want %v in ascending ID", got, want)
	}
	if tr, ok := s.trainerOf(c.homeMillers[0]); !ok || tr != c.homeWeaver {
		t.Errorf("trainerOf = %d, %v, want %d, true", tr, ok, c.homeWeaver)
	}
	if _, ok := s.trainerOf(c.homeMillers[1]); ok {
		t.Errorf("trainerOf(a non-trainee) = true, want false")
	}
	s.trainer[c.homeWeaver].acc = 6_000
	if _, ok := s.end(c.homeMillers[0]); !ok || s.trainer[c.homeWeaver].acc != 6_000 {
		t.Errorf("after the first trainee left, acc = %d, want 6 000 kept", s.trainer[c.homeWeaver].acc)
	}
	if _, ok := s.end(c.homeMillers[2]); !ok || s.trainer[c.homeWeaver].acc != 0 || s.isTrainer(c.homeWeaver) {
		t.Errorf("after the last trainee left, trainer = %+v, want the zero record", s.trainer[c.homeWeaver])
	}
	if _, ok := s.end(c.homeMillers[2]); ok {
		t.Errorf("end of a non-trainee = true, want false")
	}
	s.start(c.homeMillers[0], c.homeWeaver, 1, 0, 1)
	for name, f := range map[string]func(){
		"a second traineeship": func() { s.start(c.homeMillers[0], c.homeWeaver, 1, 0, 1) },
		"its own trainer":      func() { s.start(c.homeWeaver, c.homeWeaver, 1, 0, 1) },
		"a trainee as trainer": func() { s.start(c.homeMillers[1], c.homeMillers[0], 1, 0, 1) },
	} {
		if msg := invariantMessage(f); msg == "" {
			t.Errorf("start with %s did not panic, want a violated invariant", name)
		}
	}
}

// TestCheckpoint_SkillsRoundTrip verifies tag 10: a world with nothing held, and a
// world with no training role, write four counts of 0; a planted world writes the
// layout of spec-checkpoint.md 4.2 row 10 (count, then its rows, four times); and
// a restored world holds the same state and writes the same bytes.
//
// Arithmetic: the planted state has one trainee (home miller 0), one trainer, two
// queue cells and one Warn row. The expected bytes are written field by field below
// with the codec: 1 trainee row (agent, trainer, role 1, served 2, applied 3), 1
// trainer row (agent, acc 4 500, 1 trainee), 2 cells ((0, 1) with 2 applicants and
// (1, 1) with 1) and 1 Warn row (1, 1, 7).
//
// Requirement: FR-SKL-027, FR-CHK-023
func TestCheckpoint_SkillsRoundTrip(t *testing.T) {
	t.Parallel()
	zero := []byte{0, 0, 0, 0}
	if got := section10(t, newMill(t, panConfig(), WithSeed(1))); !bytes.Equal(got, zero) {
		t.Errorf("tag 10 of a world with no training role = %v, want four counts of 0", got)
	}
	w, c := skillsWorld(t, 8)
	if got := section10(t, w); !bytes.Equal(got, zero) {
		t.Errorf("tag 10 of a world with nothing held = %v, want four counts of 0", got)
	}
	plantSkills(w, c)
	cw := codec.NewWriter(nil)
	cw.Count(1)
	for _, v := range []uint64{uint64(c.homeMillers[0]), uint64(c.homeWeaver), 1} {
		cw.U(v)
	}
	cw.U(2)
	cw.U(3)
	cw.Count(1)
	cw.U(uint64(c.homeWeaver))
	cw.U(4_500)
	cw.Count(1)
	cw.U(uint64(c.homeMillers[0]))
	cw.Count(2)
	cw.U(0)
	cw.U(1)
	cw.Count(2)
	for _, ap := range []applicant{{c.homeMillers[1], 4, 1}, {c.homeMillers[2], 4, 0}} {
		cw.U(uint64(ap.agent))
		cw.U(uint64(ap.applied))
		cw.U(uint64(ap.served))
	}
	cw.U(1)
	cw.U(1)
	cw.Count(1)
	cw.U(uint64(c.yonderMillers[0]))
	cw.U(2)
	cw.U(3)
	cw.Count(1)
	cw.U(1)
	cw.U(1)
	cw.U(7)
	if got := section10(t, w); !bytes.Equal(got, cw.Out()) {
		t.Fatalf("tag 10 of the planted world = %v, want %v", got, cw.Out())
	}
	got := restoreWith(t, w, skillsConfig())
	if g, want := section10(t, got), section10(t, w); !bytes.Equal(g, want) {
		t.Errorf("tag 10 after the restore = %v, want %v", g, want)
	}
	if tr, ok := got.skills.trainerOf(c.homeMillers[0]); !ok || tr != c.homeWeaver || got.skills.trainee[c.homeMillers[0]].served != 2 {
		t.Errorf("restored trainee = %+v, want the planted one", got.skills.trainee[c.homeMillers[0]])
	}
	if !slices.Equal(got.skills.queue[got.skills.cell(0, 1)], w.skills.queue[w.skills.cell(0, 1)]) || got.skills.warnRun[got.skills.cell(1, 1)] != 7 || got.skills.trainer[c.homeWeaver].acc != 4_500 {
		t.Errorf("restored queue, run or accumulator differ from the planted ones")
	}
}

// skillSection is the pieces of a tag 10 body for the corrupt cases, written in
// the layout of spec-checkpoint.md 4.2 row 10.
type skillSection struct {
	trainees [][5]int64 // agent, trainer, role, served, applied
	trainers []trainerRow
	queues   []queueRow
	warns    [][3]int64 // region, role, run
}
type trainerRow struct {
	agent, acc int64
	trainees   []int64
}
type queueRow struct {
	region, role int64
	apps         [][3]int64 // agent, applied, served
}

func (s skillSection) bytes() []byte {
	cw := codec.NewWriter(nil)
	cw.Count(len(s.trainees))
	for _, r := range s.trainees {
		cw.U(uint64(r[0]))
		cw.U(uint64(r[1]))
		cw.U(uint64(r[2]))
		cw.U(uint64(r[3]))
		cw.U(uint64(r[4]))
	}
	cw.Count(len(s.trainers))
	for _, r := range s.trainers {
		cw.U(uint64(r.agent))
		cw.U(uint64(r.acc))
		cw.Count(len(r.trainees))
		for _, t := range r.trainees {
			cw.U(uint64(t))
		}
	}
	cw.Count(len(s.queues))
	for _, q := range s.queues {
		cw.U(uint64(q.region))
		cw.U(uint64(q.role))
		cw.Count(len(q.apps))
		for _, a := range q.apps {
			cw.U(uint64(a[0]))
			cw.U(uint64(a[1]))
			cw.U(uint64(a[2]))
		}
	}
	cw.Count(len(s.warns))
	for _, r := range s.warns {
		cw.U(uint64(r[0]))
		cw.U(uint64(r[1]))
		cw.U(uint64(r[2]))
	}
	return cw.Out()
}

// TestCheckpoint_SkillsRefusesCorrupt verifies that the decoder of tag 10 and the
// cross-check of F7 refuse each way the section can be wrong: rows with no role
// that trains, an agent that is not a live agent of weight 1, a role out of range
// or with no training, a row not above the previous, ticks served above the
// course, an accumulator outside [0, 9 999], a trainee in another region than its
// trainer, an edge in one of the two sections and not the other, a queued agent
// that is a trainee, an agent queued twice, and a Warn run of a role that does not
// train.
//
// Requirement: FR-SKL-027, FR-CHK-023, FR-CHK-021
func TestCheckpoint_SkillsRefusesCorrupt(t *testing.T) {
	t.Parallel()
	_, c := skillsWorld(t, 8)
	hm, hw, ym, yw := c.homeMillers, int64(c.homeWeaver), c.yonderMillers, int64(c.yonderWeaver)
	a := func(id AgentID) int64 { return int64(id) }
	good := skillSection{
		trainees: [][5]int64{{a(hm[0]), hw, 1, 2, 3}},
		trainers: []trainerRow{{hw, 100, []int64{a(hm[0])}}},
	}
	with := func(f func(*skillSection)) skillSection { s := good; f(&s); return s }
	tests := []struct {
		name string
		sec  skillSection
		cfg  *Config // nil: the world of skillsConfig
	}{
		{"trainee with an unknown role", with(func(s *skillSection) { s.trainees = [][5]int64{{a(hm[0]), hw, 9, 2, 3}} }), nil},
		{"trainee for a role with no training", with(func(s *skillSection) { s.trainees = [][5]int64{{a(hm[0]), a(hm[1]), 0, 2, 3}} }), nil},
		{"trainee that is not an agent", with(func(s *skillSection) { s.trainees = [][5]int64{{999, hw, 1, 2, 3}} }), nil},
		{"trainee that is the estate", with(func(s *skillSection) { s.trainees = [][5]int64{{0, hw, 1, 2, 3}} }), nil},
		{"trainee with a trainer that is not a weaver", with(func(s *skillSection) { s.trainees = [][5]int64{{a(hm[0]), a(hm[1]), 1, 2, 3}} }), nil},
		{"trainee in the role it trains for", with(func(s *skillSection) { s.trainees = [][5]int64{{hw, yw, 1, 2, 3}} }), nil},
		{"trainee of itself", with(func(s *skillSection) { s.trainees = [][5]int64{{hw, hw, 1, 2, 3}} }), nil},
		{"ticks served above the course", with(func(s *skillSection) { s.trainees = [][5]int64{{a(hm[0]), hw, 1, 5, 3}} }), nil},
		{"application tick in the future", with(func(s *skillSection) { s.trainees = [][5]int64{{a(hm[0]), hw, 1, 2, 9}} }), nil},
		{"trainee in another region than its trainer", with(func(s *skillSection) { s.trainees = [][5]int64{{a(ym[0]), hw, 1, 2, 3}} }), nil},
		{"trainees not ascending", with(func(s *skillSection) {
			s.trainees = [][5]int64{{a(hm[1]), hw, 1, 0, 1}, {a(hm[0]), hw, 1, 0, 1}}
			s.trainers = []trainerRow{{hw, 0, []int64{a(hm[0]), a(hm[1])}}}
		}), nil},
		{"accumulator at 10 000", with(func(s *skillSection) { s.trainers = []trainerRow{{hw, 10_000, []int64{a(hm[0])}}} }), nil},
		{"trainer with no trainee", with(func(s *skillSection) { s.trainers = []trainerRow{{hw, 0, nil}} }), nil},
		{"trainee without its trainer row", with(func(s *skillSection) { s.trainers = nil }), nil},
		{"trainer lists a non-trainee", with(func(s *skillSection) { s.trainers = []trainerRow{{hw, 0, []int64{a(hm[0]), a(hm[1])}}} }), nil},
		{"trainer lists another trainer's trainee", with(func(s *skillSection) {
			s.trainers = []trainerRow{{hw, 0, []int64{a(hm[0])}}, {yw, 0, []int64{a(hm[0])}}}
		}), nil},
		{"trainers not ascending", with(func(s *skillSection) {
			s.trainees = [][5]int64{{a(hm[0]), hw, 1, 0, 1}, {a(ym[0]), yw, 1, 0, 1}}
			s.trainers = []trainerRow{{yw, 0, []int64{a(ym[0])}}, {hw, 0, []int64{a(hm[0])}}}
		}), nil},
		// Defensive: an ID of 2^63 or more is a negative AgentID once converted; the
		// cross-check indexed with it and panicked.
		// Two real trainees keep the binary search of the first cross-check pass, so
		// only the last element is wrong.
		{"trainer lists an agent ID above int64", with(func(s *skillSection) {
			s.trainees = [][5]int64{{a(hm[0]), hw, 1, 2, 3}, {a(hm[1]), hw, 1, 2, 3}}
			s.trainers = []trainerRow{{hw, 0, []int64{a(hm[0]), a(hm[1]), -1}}}
		}), nil},
		{"trainer lists an agent beyond the agents", with(func(s *skillSection) { s.trainers = []trainerRow{{hw, 0, []int64{a(hm[0]), 1 << 40}}} }), nil},
		{"trainee listed twice by its trainer", with(func(s *skillSection) { s.trainers = []trainerRow{{hw, 0, []int64{a(hm[0]), a(hm[0])}}} }), nil},
		{"queued trainee", with(func(s *skillSection) { s.queues = []queueRow{{0, 1, [][3]int64{{a(hm[0]), 3, 0}}}} }), nil},
		{"agent queued twice in one queue", with(func(s *skillSection) { s.queues = []queueRow{{0, 1, [][3]int64{{a(hm[1]), 3, 0}, {a(hm[1]), 4, 0}}}} }), nil},
		{"agent queued in two cells", with(func(s *skillSection) {
			s.queues = []queueRow{{0, 1, [][3]int64{{a(hm[1]), 3, 0}}}, {1, 1, [][3]int64{{a(hm[1]), 3, 0}}}}
		}), nil},
		{"queued agent of another region", with(func(s *skillSection) { s.queues = []queueRow{{0, 1, [][3]int64{{a(ym[0]), 3, 0}}}} }), nil},
		{"queued agent already in the role", with(func(s *skillSection) { s.queues = []queueRow{{0, 1, [][3]int64{{hw, 3, 0}}}} }), nil},
		{"queue of a role with no training", with(func(s *skillSection) { s.queues = []queueRow{{0, 0, [][3]int64{{a(hm[1]), 3, 0}}}} }), nil},
		{"queue of an unknown region", with(func(s *skillSection) { s.queues = []queueRow{{5, 1, [][3]int64{{a(hm[1]), 3, 0}}}} }), nil},
		{"empty queue row", with(func(s *skillSection) { s.queues = []queueRow{{0, 1, nil}} }), nil},
		{"queue rows not ascending", with(func(s *skillSection) {
			s.queues = []queueRow{{1, 1, [][3]int64{{a(ym[0]), 3, 0}}}, {0, 1, [][3]int64{{a(hm[1]), 3, 0}}}}
		}), nil},
		{"applicant with ticks served above the course", with(func(s *skillSection) { s.queues = []queueRow{{0, 1, [][3]int64{{a(hm[1]), 3, 5}}}} }), nil},
		{"applicant that applied in the future", with(func(s *skillSection) { s.queues = []queueRow{{0, 1, [][3]int64{{a(hm[1]), 9, 0}}}} }), nil},
		{"warn run of a role with no training", with(func(s *skillSection) { s.warns = [][3]int64{{0, 0, 3}} }), nil},
		{"warn run above the ticks", with(func(s *skillSection) { s.warns = [][3]int64{{0, 1, 9}} }), nil},
		{"warn run at 0", with(func(s *skillSection) { s.warns = [][3]int64{{0, 1, 0}} }), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fresh := newMill(t, skillsConfig(), WithSeed(exitSeed))
			step(t, fresh, 8)
			r := codec.NewReader(tt.sec.bytes())
			err := fresh.loadSkills(&restoreCtx{tick: 8}, r)
			if err == nil {
				err = r.Done()
			}
			if err == nil {
				err = fresh.skillsCross()
			}
			if err == nil {
				t.Errorf("loadSkills + skillsCross accepted %+v, want a refusal", tt.sec)

			}
		})
	}
	t.Run("the good section is accepted", func(t *testing.T) {
		t.Parallel()
		fresh := newMill(t, skillsConfig(), WithSeed(exitSeed))
		step(t, fresh, 8)
		r := codec.NewReader(good.bytes())
		if err := fresh.loadSkills(&restoreCtx{tick: 8}, r); err != nil {
			t.Fatalf("loadSkills error = %v, want nil", err)
		}
		if err := r.Done(); err != nil {
			t.Fatalf("Done error = %v, want nil", err)
		}
		if err := fresh.skillsCross(); err != nil {
			t.Errorf("skillsCross error = %v, want nil", err)
		}
	})
	// Fix: iteration 1 of the PR A review. readApplicant accepted a queued agent that
	// is also a trainer with trainees, a state no live world reaches. The agent here
	// holds the role it is queued for, so the older check would also refuse it, but
	// with the "already holds role" message: the message pins which check refused.
	t.Run("queued trainer with trainees is refused as a trainer", func(t *testing.T) {
		t.Parallel()
		sec := with(func(s *skillSection) { s.queues = []queueRow{{0, 1, [][3]int64{{hw, 3, 0}}}} })
		fresh := newMill(t, skillsConfig(), WithSeed(exitSeed))
		step(t, fresh, 8)
		r := codec.NewReader(sec.bytes())
		err := fresh.loadSkills(&restoreCtx{tick: 8}, r)
		if err == nil {
			err = r.Done()
		}
		if err == nil {
			t.Fatalf("loadSkills accepted a queued trainer, want a refusal")
		}
		if got := err.Error(); !strings.Contains(got, "trainer with trainees") {
			t.Errorf("loadSkills error = %q, want it to name a trainer with trainees", got)
		}
	})
	t.Run("rows in a world with no training role", func(t *testing.T) {
		t.Parallel()
		off := newMill(t, panConfig(), WithSeed(1))
		r := codec.NewReader(skillSection{warns: [][3]int64{{0, 0, 1}}}.bytes())
		if err := off.loadSkills(&restoreCtx{tick: 8}, r); err == nil {
			err = r.Done()
			if err == nil {
				t.Errorf("loadSkills accepted a row with no training role, want a refusal")
			}
		}
	})
}

// TestPositions_P22Listed verifies that P2.2 is a position of the straight line
// owned by SKL, between P2.1 and P2.3, and is no longer listed as not built.
//
// Guard: the position table test compares tick.go with spec-world.md 6, and a
// position that lost its owner or its place would pass it only by a matching edit
// of the spec; this pins the entry this module appended.
//
// Requirement: FR-SKL-015
func TestPositions_P22Listed(t *testing.T) {
	t.Parallel()
	at := slices.IndexFunc(positionList[:], func(p position) bool { return p.id == "P2.2" })
	if at < 1 || positionList[at].owner != "SKL" || positionList[at].where != line ||
		positionList[at-1].id != "P2.1" || positionList[at+1].id != "P2.3" {
		t.Errorf("P2.2 at %d of positionList, want an SKL line entry between P2.1 and P2.3", at)
	}
	if slices.Contains(notBuiltPositions[:], "P2.2") {
		t.Errorf("notBuiltPositions lists P2.2, which has a call")
	}
}

// FuzzLoadSkillsSection verifies that the decoder of tag 10 and the cross-check of
// F7 never panic on an arbitrary body, and that a body they accept is a fixed
// point: saving the restored state gives the same bytes.
//
// The seeds are the empty section, the planted state of TestCheckpoint_SkillsRoundTrip
// and one case of each refusal of TestCheckpoint_SkillsRefusesCorrupt.
//
// Requirement: FR-SKL-027, FR-CHK-016
func FuzzLoadSkillsSection(f *testing.F) {
	probe := newMill(f, skillsConfig(), WithSeed(exitSeed))
	step(f, probe, 8)
	c := castOf(f, probe)
	a := func(id AgentID) int64 { return int64(id) }
	hm, hw, ym, yw := c.homeMillers, a(c.homeWeaver), c.yonderMillers, a(c.yonderWeaver)
	good := skillSection{
		trainees: [][5]int64{{a(hm[0]), hw, 1, 2, 3}},
		trainers: []trainerRow{{hw, 4_500, []int64{a(hm[0])}}},
		queues:   []queueRow{{0, 1, [][3]int64{{a(hm[1]), 4, 1}, {a(hm[2]), 4, 0}}}, {1, 1, [][3]int64{{a(ym[0]), 2, 3}}}},
		warns:    [][3]int64{{1, 1, 7}},
	}
	f.Add([]byte{0, 0, 0, 0})
	f.Add(good.bytes())
	f.Add(skillSection{trainees: [][5]int64{{a(hm[0]), hw, 9, 2, 3}}}.bytes())                         // role out of range
	f.Add(skillSection{trainees: [][5]int64{{a(hm[0]), hw, 1, 5, 3}}}.bytes())                         // served above the course
	f.Add(skillSection{trainers: []trainerRow{{hw, 10_000, nil}}}.bytes())                             // accumulator out of range
	f.Add(skillSection{queues: []queueRow{{0, 1, [][3]int64{{hw, 3, 0}}}}}.bytes())                    // queued trainer
	f.Add(skillSection{trainees: [][5]int64{{a(hm[0]), yw, 1, 0, 0}}}.bytes())                         // other region
	f.Add(skillSection{warns: [][3]int64{{0, 0, 1}}}.bytes())                                          // run of a role with no training
	f.Add(skillSection{queues: []queueRow{{0, 1, [][3]int64{{a(hm[1]), 1, 0}}}, {0, 1, nil}}}.bytes()) // cell not above the previous
	f.Fuzz(func(t *testing.T, body []byte) {
		w := newMill(t, skillsConfig(), WithSeed(exitSeed))
		step(t, w, 8)
		r := codec.NewReader(body)
		err := w.loadSkills(&restoreCtx{tick: 8}, r)
		if err == nil {
			err = r.Done()
		}
		if err == nil {
			err = w.skillsCross()
		}
		if err != nil {
			return
		}
		if got := section10(t, w); !bytes.Equal(got, body) {
			t.Errorf("accepted tag 10 body %v saves as %v, want the same bytes", body, got)
		}
	})
}

// TestSkills_PlantedStateSurvivesTicksAndFork verifies, with real components, that
// a world holding a traineeship, two queues and a Warn run keeps them untouched
// through further steps (PR A adds no behaviour, so nothing may move them), passes
// its audit, and that a world restored from a checkpoint taken then stays equal to
// the original, checkpoint byte for byte, after the same further steps.
//
// Guard: the state is carried by tag 10 only, and the fork of a world with skills
// must replay the original; a field of the state that the codec drops, or a hook
// that mutates it during a tick, shows up as a different checkpoint here.
//
// Requirement: FR-SKL-027, FR-CHK-050
func TestSkills_PlantedStateSurvivesTicksAndFork(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, c := skillsWorld(t, 8)
	plantSkills(w, c)
	before := section10(t, w)
	fork := restoreWith(t, w, skillsConfig())
	step(t, w, 6)
	step(t, fork, 6)
	if got := section10(t, w); !bytes.Equal(got, before) {
		t.Errorf("tag 10 after 6 steps = %v, want the planted %v", got, before)
	}
	if got := section10(t, fork); !bytes.Equal(got, before) {
		t.Errorf("tag 10 of the fork after 6 steps = %v, want the planted %v", got, before)
	}
	a, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	b, err := fork.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() of the fork error = %v, want nil", err)
	}
	if !bytes.Equal(a.b, b.b) {
		t.Errorf("checkpoint of the fork differs from the original after the same 6 steps (%d and %d bytes)", len(b.b), len(a.b))
	}
}
