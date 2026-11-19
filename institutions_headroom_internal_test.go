package economysim

import (
	"testing"
)

// The tests of the headroom a body gives a role and of the held completion (plan 3-org
// Phase 9, spec-institutions.md 3.4, 4.5). The views and the births read the same fields as
// the class limit of the settlements module (RoleView.Headroom and NoLimit), so what is
// tested here is that a body is a second source of the limit and the least of the two wins.

// orgHeadroomWorld is the world of exitConfig (home: three millers in two agents, one
// weaver) with a body guild over the miller of home with the given places, the settlements
// module when settle is true (the miller bounded by the class limit "places": 4 in class
// small, 50 in class big, the world's class being big when bigAt is 4) and the settlements
// fields cleared otherwise.
func orgHeadroomWorld(t *testing.T, bigAt, places int64, settle bool) *World {
	t.Helper()
	return readerWorld(t, bigAt, func(c *Config) {
		if !settle {
			c.Modules.Settlements = nil
			c.Roles[0].Needs, c.Roles[0].Ceiling = "", ""
		}
		c.Modules.Institutions = &Institutions{Bodies: []Body{{ID: "guild", Roles: []string{"miller"}, Regions: []string{"home"}, Places: places}}}
	})
}

// TestRoleView_BodyHeadroom verifies the view of the miller: a body with places is a limit of
// its own (places less the households of the body's roles), the least of it and the class
// limit wins, a body with no places gives no figure, and the weaver, in no body, is untouched.
//
// Arithmetic: the miller holds 3 households. The class small limits to 4: headroom 1; class
// big to 50: 47. The body's headroom is places - 3: 1 for 4 places, 97 for 100. So: body alone
// of 4 places, 1; class big 47 with body 4 places, min(47, 1) = 1; class big 47 with body 100
// places, min(47, 97) = 47; class small 1 with body 100 places, min(1, 97) = 1; a body of 0
// places leaves the class's 47 and, with no class, no limit.
//
// Requirement: FR-ORG-007
func TestRoleView_BodyHeadroom(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		bigAt, places int64
		settle        bool
		headroom      int64
		noLimit       bool
	}{
		{"a body alone", 1_000, 4, false, 1, false},
		{"a body with no places and no class", 1_000, 0, false, 0, true},
		{"the body is the least", 4, 4, true, 1, false},
		{"the class is the least", 4, 100, true, 47, false},
		{"the small class is the least", 1_000, 100, true, 1, false},
		{"a body with no places leaves the class", 4, 0, true, 47, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			roles := orgHeadroomWorld(t, tc.bigAt, tc.places, tc.settle).buildRegionView(0).Roles
			if roles[0].Headroom != tc.headroom || roles[0].NoLimit != tc.noLimit {
				t.Errorf("miller Headroom %d NoLimit %v, want %d and %v", roles[0].Headroom, roles[0].NoLimit, tc.headroom, tc.noLimit)
			}
			if !roles[1].NoLimit {
				t.Errorf("weaver NoLimit = false, want true (in no body, with no ceiling)")
			}
		})
	}
}

// TestRegion_NoBodyRoleUntouched verifies that with the module off, or present with no body
// over the role, the view of every role is what it was without the module: the same
// Headroom and NoLimit as the world of the class alone.
//
// Requirement: FR-ORG-001, FR-ORG-007
func TestRegion_NoBodyRoleUntouched(t *testing.T) {
	t.Parallel()
	base := readerWorld(t, 4, nil)
	with := readerWorld(t, 4, func(c *Config) { c.Modules.Institutions = &Institutions{} })
	a, b := base.buildRegionView(0).Roles, with.buildRegionView(0).Roles
	for r := range a {
		if a[r].Headroom != b[r].Headroom || a[r].NoLimit != b[r].NoLimit {
			t.Errorf("role %d: Headroom %d NoLimit %v with the empty module, want %d and %v without it", r, b[r].Headroom, b[r].NoLimit, a[r].Headroom, a[r].NoLimit)
		}
	}
}

// TestPopulation_BirthsCappedByBody verifies that the births of the population module are
// capped by a body's headroom as by a class's, under the same count: the dropped birth is
// in BirthsCeiling and not in BirthsCapped.
//
// Arithmetic: as TestPopulation_BirthsCeilingCounted, with the class replaced by a body of 4
// places over the miller: headroom 4 - 3 = 1, two births due, one kept and one dropped.
//
// Requirement: FR-ORG-007, FR-POP-041
func TestPopulation_BirthsCappedByBody(t *testing.T) {
	t.Parallel()
	w := readerWorld(t, 1_000, func(c *Config) {
		c.Modules.Settlements = nil
		c.Roles[0].Needs, c.Roles[0].Ceiling = "", ""
		p := DefaultPopulation()
		c.Modules.Population = &p
		c.World.EntryPoolShare = 10_000
		c.Modules.Institutions = &Institutions{Bodies: []Body{{ID: "guild", Roles: []string{"miller"}, Regions: []string{"home"}, Places: 4}}}
	})
	for id := 1; id <= 3; id++ {
		setRatio(w, AgentID(id), 14_000)
	}
	popView(w, 0)
	setCarries(w, nil, map[RoleID]int64{0: 2*w.pop.denom - 1})
	v := popView(w, 0)
	c := w.pop.cnt[0]
	if v.Roles[0].PopBirths != 1 || c.birthsDue != 2 || c.birthsCeiling != 1 || c.birthsCapped != 0 {
		t.Errorf("PopBirths %d due %d ceiling %d capped %d, want 1, 2, 1, 0", v.Roles[0].PopBirths, c.birthsDue, c.birthsCeiling, c.birthsCapped)
	}
}

// orgSkilledConfig is orgTrainConfig with the settlements module on and the weaver bounded by
// the class limit "places" (4 in the only class), when settle is true, and a body over the
// weaver with the given places. The weavers are three households of their own, so the class
// headroom of the weaver is 4 - 3 = 1.
func orgSkilledConfig(places int64, settle, body bool) Config {
	c := orgTrainConfig(places, 0, "")
	if settle {
		m := DefaultSettlements()
		m.Classes = []SettlementClass{{ID: "small", Enter: 0, Limits: []ClassLimit{{Name: "places", Value: 4}}}}
		c.Modules.Settlements = &m
		c.Roles[skWeaver].Ceiling = "places"
	}
	if !body {
		c.Modules.Institutions = nil
	}
	return c
}

// TestOrgHeadroom_LeastOfTwo verifies the slot of the completion with both modules on: it is
// the least of the class limit's headroom and the body's, a body alone gives its own, the
// class alone gives its own, and with neither a completion is never held.
//
// Arithmetic: the weaver holds 3 households. The class limit is 4: headroom 1. A body of 5
// places has headroom 2, so min(1, 2) = 1; a body of 3 places has 0, so min(1, 0) = 0. A body
// alone of 5 places gives 2; the class alone gives 1.
//
// Requirement: FR-ORG-007
func TestOrgHeadroom_LeastOfTwo(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		places       int64
		settle, body bool
		headroom     int64
		noLimit      bool
	}{
		{"the class is the least", 5, true, true, 1, false},
		{"the body is the least", 3, true, true, 0, false},
		{"a body alone", 5, false, true, 2, false},
		{"a body with no places and the class", 0, true, true, 1, false},
		{"the class alone", 5, true, false, 1, false},
		{"neither", 5, false, false, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, orgSkilledConfig(tc.places, tc.settle, tc.body), WithSeed(exitSeed))
			if got, none := w.completionHeadroom(0, skWeaver); got != tc.headroom || none != tc.noLimit {
				t.Errorf("completionHeadroom(home, weaver) = %d, noLimit %v, want %d, %v", got, none, tc.headroom, tc.noLimit)
			}
		})
	}
}

// orgHeldWorld is the world of orgSkilledConfig with a body of the given places over the
// weaver and two trainees of a weaver at the end of their training, as a world that stepped
// once would hold them: the miller first applied at tick 2 and the other at tick 5, so the
// first completes first whatever their IDs.
func orgHeldWorld(t *testing.T, places int64) (w *World, rec *evLog, first, second AgentID) {
	t.Helper()
	rec = &evLog{}
	w = newMill(t, orgSkilledConfig(places, false, true), WithSeed(exitSeed), WithObserver(rec))
	step(t, w, 1)
	millers := orgAgents(w, 0, skMiller)
	first, second = AgentID(millers[1]), AgentID(millers[0])
	trainer := AgentID(orgAgents(w, 0, skWeaver)[0])
	w.skills.start(second, trainer, skWeaver, 4, 5)
	w.skills.start(first, trainer, skWeaver, 4, 2)
	return w, rec, first, second
}

// TestOrgCompletion_HeldWithoutBodyHeadroom verifies the held completion of a body
// (spec-institutions.md 4.5.2): a trainee at TrainingTicks whose target role has no headroom
// stays a trainee and is counted Held, and the count reaches MembershipChanged at P7.2b.
//
// Arithmetic: the weaver holds 3 households and the body has 3 places, so headroom 0 and both
// trainees are held: Held 2.
//
// Requirement: FR-ORG-027, FR-ORG-039
func TestOrgCompletion_HeldWithoutBodyHeadroom(t *testing.T) {
	t.Parallel()
	w, rec, first, second := orgHeldWorld(t, 3)
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry() error = %v, want nil", err)
	}
	if !w.skills.isTrainee(first) || !w.skills.isTrainee(second) || w.skills.trainee[first].served != 4 {
		t.Errorf("with no headroom: trainees %v, %v served %d, want both held at 4", w.skills.isTrainee(first), w.skills.isTrainee(second), w.skills.trainee[first].served)
	}
	w.p72bInstitutions()
	var held int64
	for _, e := range of[MembershipChanged](rec) {
		held += e.Held
	}
	if held != 2 {
		t.Errorf("MembershipChanged.Held = %d, want 2", held)
	}
}

// heldNow is the Held of MembershipChanged for (home, guild) as P7.2b would tell it now, and
// clears the count so the next call reads only what came after.
func heldNow(w *World, rec *evLog) int64 {
	rec.events = nil
	w.p72bInstitutions()
	var held int64
	for _, e := range of[MembershipChanged](rec) {
		held += e.Held
	}
	w.org.rd.out[0*w.org.nb+0].Held = 0
	return held
}

// TestOrgCompletion_FirstEntryStepWithHeadroom verifies that a place that opens is taken by
// the first held trainee in the first entry step in which the headroom reaches 1, in the
// order of application, and that the entrant of the same step sees the headroom the
// completion left (spec-institutions.md 4.5.2).
//
// Arithmetic: the weaver holds 3 households. With 3 places the headroom is 0 and both are
// held (Held 2). With 4 places it is 1: the first, which applied at tick 2, completes and the
// weaver count rises to 4, so the headroom is 0 again and the second stays held (Held 1), and
// the view a spawner reads shows Headroom 0. With 5 places the second completes (Held 0).
//
// Requirement: FR-ORG-027
func TestOrgCompletion_FirstEntryStepWithHeadroom(t *testing.T) {
	t.Parallel()
	w, rec, first, second := orgHeldWorld(t, 3)
	entry := func() {
		t.Helper()
		if err := w.skillsEntry(0); err != nil {
			t.Fatalf("skillsEntry() error = %v, want nil", err)
		}
	}
	entry()
	if got := heldNow(w, rec); got != 2 {
		t.Errorf("Held with no headroom = %d, want 2", got)
	}
	w.org.places[0] = 4
	entry()
	if w.skills.isTrainee(first) || w.agents[first].role != skWeaver || !w.skills.isTrainee(second) {
		t.Errorf("with headroom 1: first still a trainee %v (role %d), second a trainee %v, want the first completed and the second held", w.skills.isTrainee(first), w.agents[first].role, w.skills.isTrainee(second))
	}
	if got := heldNow(w, rec); got != 1 {
		t.Errorf("Held with headroom 1 and two trainees = %d, want 1", got)
	}
	if v := w.buildRegionView(0).Roles[skWeaver]; v.NoLimit || v.Headroom != 0 {
		t.Errorf("view of the weaver after the completion: Headroom %d NoLimit %v, want 0 and false (the held trainee took the place before an entrant)", v.Headroom, v.NoLimit)
	}
	w.org.places[0] = 5
	entry()
	if w.skills.isTrainee(second) {
		t.Errorf("second still a trainee with headroom 1 again, want completed")
	}
	if got := heldNow(w, rec); got != 0 {
		t.Errorf("Held after the last completion = %d, want 0", got)
	}
}

// TestOrgCompletion_NoBodyNeverHolds verifies that a body with no places never holds a
// completion (spec-institutions.md 4.5.4): both trainees complete in the first entry step and
// nothing is counted Held.
//
// Requirement: FR-ORG-027
func TestOrgCompletion_NoBodyNeverHolds(t *testing.T) {
	t.Parallel()
	w, rec, first, second := orgHeldWorld(t, 0)
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry() error = %v, want nil", err)
	}
	if w.skills.isTrainee(first) || w.skills.isTrainee(second) {
		t.Errorf("trainees %v, %v after the entry step of a body with no places, want both completed", w.skills.isTrainee(first), w.skills.isTrainee(second))
	}
	if got := heldNow(w, rec); got != 0 {
		t.Errorf("Held = %d, want 0", got)
	}
}

// TestOrgHeadroom_WiredAfterRestore verifies that a restored world gives the same headroom
// for a completion as the original: the slot is derived at construction and not saved, so the
// restore wires it as a new world does.
//
// Arithmetic: 3 weavers and a body of 4 places: headroom 1; with the class limit of 4 as well,
// min(1, 1) = 1.
//
// Requirement: FR-ORG-007, FR-ORG-033
func TestOrgHeadroom_WiredAfterRestore(t *testing.T) {
	t.Parallel()
	cfg := orgSkilledConfig(4, true, true)
	w := newMill(t, cfg, WithSeed(exitSeed))
	step(t, w, 2)
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	restored, err := RestoreWorld(cfg, ck)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	wantH, wantNone := w.completionHeadroom(0, skWeaver)
	if h, none := restored.completionHeadroom(0, skWeaver); h != wantH || none != wantNone || wantNone {
		t.Errorf("completionHeadroom of the restored world = %d, %v, want %d, %v (a limit)", h, none, wantH, wantNone)
	}
}

// TestAlloc_InstitutionsSkillsNoOp verifies NFR-ORG-001 for the readers of the traineeship
// and the headroom: with no observer and no log level that writes, the trainer test, the
// body's maximum, the count of closed places, the held count, the headroom of a completion,
// the view fill and the move notice allocate nothing once the scratch exists.
//
// Not parallel: testing.AllocsPerRun is process-wide.
//
// Requirement: NFR-ORG-001
func TestAlloc_InstitutionsSkillsNoOp(t *testing.T) {
	w := newMill(t, orgSkilledConfig(4, true, true), WithSeed(exitSeed))
	step(t, w, 1)
	a := &w.agents[orgAgents(w, 0, skWeaver)[0]]
	w.skills.enqueue(w.skills.cell(0, skWeaver), applicant{agent: AgentID(orgAgents(w, 0, skMiller)[0])})
	w.buildRegionView(0) // grows the scratch of the view once
	for name, f := range map[string]func(){
		"mayTrain":        func() { w.org.mayTrain(a, 0) },
		"trainerMax":      func() { w.org.trainerMax(a, 0, 3) },
		"viewMaxTrainees": func() { w.org.viewMaxTrainees(0, skWeaver, 3) },
		"noteTrain":       func() { w.org.noteTrain(w, 0, skWeaver) },
		"noteHeld":        func() { w.org.noteHeld(w, 0, skWeaver, 1) },
		"noteRoleMove":    func() { w.org.noteRoleMove(0, skMiller, skWeaver, 0) },
		"orgHeadroom":     func() { w.orgHeadroom(0, skWeaver) },
		"fillRoleBodies":  func() { w.fillRoleBodies(0) },
		"closedPlaces":    func() { w.org.closedPlaces(w, 0, skWeaver) },
	} {
		if n := testing.AllocsPerRun(50, f); n != 0 {
			t.Errorf("AllocsPerRun(%s) = %v, want 0", name, n)
		}
	}
}
