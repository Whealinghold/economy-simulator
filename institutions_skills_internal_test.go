package economysim

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"
)

// The tests of what a body does to the traineeship (plan 3-org Phase 9, spec-institutions.md
// 5.7, 4.5): the trainer who is a member, the body's own maximum of trainees, the count of
// the places a right closes and the held completion. The world is trainingConfig, whose
// role weaver (RoleID 1) trains and whose home region has three millers (RoleID 0), with
// three weavers of home as households of their own and the body guild over them.

// The roles of trainingConfig.
const (
	skMiller RoleID = iota
	skWeaver
)

// orgTrainConfig is trainingConfig(3) with three weavers in home, each a household of its
// own, and the body guild over the weaver of home with the given places and trainee limit.
// The weavers are agents 4, 5 and 6 (the millers are 1 to 3). A right of the activity train
// on the weaver in home is declared when right is "member" (held by guild) or "prohibition"
// (no holder); the first round admits the places' worth of weavers in the order of the key.
func orgTrainConfig(places, bodyMax int64, right string) Config {
	c := trainingConfig(3)
	c.Groups[1].Households, c.Groups[1].Scale = 3, 1
	m := &Institutions{Bodies: []Body{{ID: "guild", Roles: []string{"weaver"}, Regions: []string{"home"}, Places: places, MaxTrainees: bodyMax}}}
	switch right {
	case "member":
		m.Rights = []Right{{ID: "teach", Body: "guild", Activity: ActivityTrain, Subject: "weaver", Region: "home"}}
	case "prohibition":
		m.Rights = []Right{{ID: "teach", Activity: ActivityTrain, Subject: "weaver", Region: "home"}}
	}
	c.Modules.Institutions = m
	return c
}

// orgTrainWorld builds the world of orgTrainConfig with an event log and a capturing
// logger and runs one step, so the round of P1.9a has admitted the members.
func orgTrainWorld(t *testing.T, places, bodyMax int64, right string) (*World, *evLog, capture) {
	t.Helper()
	rec, lg := &evLog{}, newCapture()
	w := newMill(t, orgTrainConfig(places, bodyMax, right), WithSeed(exitSeed), WithObserver(rec), WithLogger(slog.New(lg)))
	step(t, w, 1)
	return w, rec, lg
}

// weaverIDs returns the weavers of home, ascending, split into members and outsiders.
func weaverIDs(w *World) (members, outsiders []AgentID) {
	for _, i := range orgAgents(w, 0, skWeaver) {
		if w.org.isMember(i, skWeaver, 0, 0) {
			members = append(members, AgentID(i))
		} else {
			outsiders = append(outsiders, AgentID(i))
		}
	}
	return members, outsiders
}

// trainerIDs returns the IDs of the candidate trainers of the weaver in home.
func trainerIDs(w *World) []AgentID {
	var ids []AgentID
	for _, s := range w.trainersOf(0, skWeaver) {
		ids = append(ids, s.id)
	}
	return ids
}

// TestTrain_NonMemberHasNoPlace verifies that where a train right is in force a household of
// the role that is not a member of the holder has no place for a trainee, that a prohibition
// opens none at all, and that with no right every holder has places.
//
// Arithmetic: guild has 1 place for 3 weavers, so one is a member and two wait. With the right
// held by guild the trainers are the one member; with the prohibition there are none; with no
// right all three.
//
// Requirement: FR-ORG-025
func TestTrain_NonMemberHasNoPlace(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, right string
		want        func(members, all []AgentID) []AgentID
	}{
		{"a right held by the body", "member", func(m, _ []AgentID) []AgentID { return m }},
		{"a prohibition", "prohibition", func(_, _ []AgentID) []AgentID { return nil }},
		{"no right", "", func(_, all []AgentID) []AgentID { return all }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w, _, _ := orgTrainWorld(t, 1, 0, tc.right)
			members, outsiders := weaverIDs(w)
			if len(members) != 1 || len(outsiders) != 2 {
				t.Fatalf("members %v, outsiders %v, want 1 member and 2 outsiders of 1 place", members, outsiders)
			}
			all := slices.Sorted(slices.Values(slices.Concat(members, outsiders)))
			if got, want := trainerIDs(w), tc.want(members, all); !slices.Equal(got, want) {
				t.Errorf("trainersOf(home, weaver) = %v, want %v", got, want)
			}
		})
	}
}

// TestTrain_TraineeshipInProgressKept verifies that a traineeship in progress under a trainer
// that is not (or no longer) a member is not thrown out: the right is read when a place opens
// (spec-institutions.md 5.7.1).
//
// Requirement: FR-ORG-025
func TestTrain_TraineeshipInProgressKept(t *testing.T) {
	t.Parallel()
	w, _, _ := orgTrainWorld(t, 1, 0, "member")
	_, outsiders := weaverIDs(w)
	trainee := AgentID(orgAgents(w, 0, skMiller)[0])
	w.skills.start(trainee, outsiders[0], skWeaver, 0, 0)
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry() error = %v, want nil", err)
	}
	if !w.skills.isTrainee(trainee) || !w.skills.isTrainer(outsiders[0]) {
		t.Errorf("trainee %v and trainer %v after the entry step, want the traineeship under the non-member kept", w.skills.isTrainee(trainee), w.skills.isTrainer(outsiders[0]))
	}
	if slices.Contains(trainerIDs(w), outsiders[0]) {
		t.Errorf("trainersOf lists the non-member %d, want it left out", outsiders[0])
	}
}

// TestTrain_TrainersCountEligibleOnly verifies that the view of the applicant counts only the
// trainers that can take a trainee (spec-institutions.md 5.7.2): 1 of 3 with the right, 3
// without it.
//
// Requirement: FR-ORG-025
func TestTrain_TrainersCountEligibleOnly(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		right string
		want  int64
	}{{"member", 1}, {"prohibition", 0}, {"", 3}} {
		w, _, _ := orgTrainWorld(t, 1, 0, tc.right)
		if got := w.buildRegionView(0).Roles[skWeaver].Trainers; got != tc.want {
			t.Errorf("RoleView.Trainers with the right %q = %d, want %d", tc.right, got, tc.want)
		}
	}
}

// TestTrain_BodyMaxTraineesLeast verifies the body's own limit: a member holds at most the
// least of the role's MaxTrainees (3) and its body's, a body's 0 is no limit of its own, the
// places the snapshot counts are exact for each trainer, and the maximum of the view is the
// least of the two where a right makes every trainer a member.
//
// Arithmetic: with a body limit of 2 the member's maximum is min(3, 2) = 2; with 5 it is
// min(3, 5) = 3; with 0 it is 3. With no train right the member has 2 places and the two
// outsiders 3 each: 2 + 3 + 3 = 8 places for 3 trainers; with the right the member alone has 2.
//
// Requirement: FR-ORG-026
func TestTrain_BodyMaxTraineesLeast(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, right    string
		bodyMax        int64
		member, outsid int64
		places, view   int64
	}{
		{"a lower body limit", "", 2, 2, 3, 8, 3},
		{"a higher body limit", "", 5, 3, 3, 9, 3},
		{"no limit of its own", "", 0, 3, 3, 9, 3},
		{"the right makes every trainer a member", "member", 2, 2, 3, 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w, _, _ := orgTrainWorld(t, 1, tc.bodyMax, tc.right)
			members, outsiders := weaverIDs(w)
			if got := w.org.trainerMax(&w.agents[members[0]], 0, 3); got != tc.member {
				t.Errorf("trainerMax(member) = %d, want %d", got, tc.member)
			}
			if got := w.org.trainerMax(&w.agents[outsiders[0]], 0, 3); got != tc.outsid {
				t.Errorf("trainerMax(outsider) = %d, want %d", got, tc.outsid)
			}
			rows := make([]RoleRow, len(w.cfg.Roles))
			w.skillsRoleRows(rows, 0)
			if rows[skWeaver].Places != tc.places {
				t.Errorf("RoleRow.Places = %d, want %d", rows[skWeaver].Places, tc.places)
			}
			if got := w.buildRegionView(0).Roles[skWeaver].MaxTrainees; got != tc.view {
				t.Errorf("RoleView.MaxTrainees = %d, want %d", got, tc.view)
			}
		})
	}
}

// TestTrain_MemberFillsToTheBodyLimit verifies the filling of places with a body limit: a
// member takes applicants up to its body's limit and no further, though the role's limit is
// higher.
//
// Arithmetic: the body limit is 1 and the right makes the member the only trainer, so of 3
// applicants one is placed and two wait.
//
// Requirement: FR-ORG-026
func TestTrain_MemberFillsToTheBodyLimit(t *testing.T) {
	t.Parallel()
	w, _, _ := orgTrainWorld(t, 1, 1, "member")
	s := w.skills
	for i, id := range orgAgents(w, 0, skMiller) {
		s.enqueue(s.cell(0, skWeaver), applicant{agent: AgentID(id), applied: int64(i)})
	}
	w.placeRole(0, skWeaver)
	if got, want := len(s.queue[s.cell(0, skWeaver)]), 2; got != want {
		t.Errorf("applicants still queued after the filling = %d, want %d (the member holds 1 trainee)", got, want)
	}
}

// TestRight_TrainCountIsMinFA verifies the count of the places a right closes
// (spec-institutions.md 5.6.3): min(f, a), f the free places of the households the right
// keeps from training and a the applicants queued, one count a place, none for a closed craft
// with no queue and none without a right; and that it is told by RightRefused and logged at
// Debug as "place not open" with the reason not_member.
//
// Arithmetic: two non-members at the role's MaxTrainees 3 have f = 2 * 3 = 6 free places. With
// 3 applicants the count is min(6, 3) = 3; with 1 applicant, min(6, 1) = 1.
//
// Requirement: FR-ORG-025, FR-ORG-040, FR-ORG-054
func TestRight_TrainCountIsMinFA(t *testing.T) {
	t.Parallel()
	w, rec, lg := orgTrainWorld(t, 1, 0, "member")
	s, o := w.skills, w.org
	cell := s.cell(0, skWeaver)
	counted := func() int64 {
		o.reportRights(w)
		var n int64
		for _, e := range of[RightRefused](rec) {
			n += e.Train
		}
		rec.events = nil
		return n
	}
	o.noteTrain(w, 0, skWeaver)
	if got := counted(); got != 0 {
		t.Errorf("RightRefused.Train with no queue = %d, want 0 (a closed craft with no queue counts none)", got)
	}
	for i, id := range orgAgents(w, 0, skMiller) {
		s.enqueue(cell, applicant{agent: AgentID(id), applied: int64(i)})
	}
	o.noteTrain(w, 0, skWeaver)
	if got := counted(); got != 3 {
		t.Errorf("RightRefused.Train with 3 applicants and 6 free places = %d, want 3", got)
	}
	s.dequeue(s.queue[cell][1].agent)
	s.dequeue(s.queue[cell][1].agent)
	o.noteTrain(w, 0, skWeaver)
	if got := counted(); got != 1 {
		t.Errorf("RightRefused.Train with 1 applicant and 6 free places = %d, want 1", got)
	}
	lines := 0
	for _, l := range *lg.lines {
		if l.msg == "place not open" && l.strs["reason"] == string(reasonNotMember) {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("Debug lines \"place not open\" with the reason not_member = %d, want 2 (one for each count above 0)", lines)
	}
	open, _, _ := orgTrainWorld(t, 1, 0, "")
	open.skills.enqueue(open.skills.cell(0, skWeaver), applicant{agent: AgentID(orgAgents(open, 0, skMiller)[0])})
	open.org.noteTrain(open, 0, skWeaver)
	if len(open.org.rights.refused) != 0 {
		t.Errorf("a world with no right holds counts %v after noteTrain with a queue, want none (no right is in force)", open.org.rights.refused)
	}
}

// TestSkillsWorld_InstitutionsRunsAndDoesNotInterfere verifies the traineeship under a body
// over a run: the world with a train right, a body limit and a body bound on its places runs
// 120 ticks under the audit, the members never exceed the places, a logger and an observer
// change nothing (the same snapshot and state hash as the bare world), and the two runs of a
// seed agree.
//
// Requirement: FR-ORG-006, FR-ORG-025, FR-ORG-053, NFR-ORG-003
//
// Guard: the readers of the body sit in the entry step of every region and in the view of
// every tick; a read that changed a result only with an observer would show here.
func TestSkillsWorld_InstitutionsRunsAndDoesNotInterfere(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	const ticks = 120
	cfg := orgTrainConfig(2, 1, "member")
	bare := newMill(t, cfg, WithSeed(exitSeed), WithAudit())
	rec, lg := &evLog{}, newCapture()
	watched := newMill(t, cfg, WithSeed(exitSeed), WithAudit(), WithObserver(rec), WithLogger(slog.New(lg)))
	step(t, bare, ticks)
	step(t, watched, ticks)
	for _, e := range of[MembershipChanged](rec) {
		if e.Members > e.Places {
			t.Errorf("tick %d: MembershipChanged members %d above places %d", e.Tick, e.Members, e.Places)
		}
	}
	a, errA := bare.Snapshot()
	b, errB := watched.Snapshot()
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) || HashWorld(bare) != HashWorld(watched) {
		t.Errorf("the world with a logger and an observer differs from the bare one after %d ticks (errors %v, %v)", ticks, errA, errB)
	}
}
