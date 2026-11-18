package economysim

import (
	"testing"
)

// This file tests the seam the settlements module fills in the skill barriers
// (spec-settlements.md 6.3, plan 3-set SET11 b): the headroom of a held completion of a
// traineeship (skills.headroom).

// seamWorld is the world of trainingConfig (home: three millers and a weaver; yonder: two
// millers and a weaver) with the module on: class small has places places and class big,
// which no region reaches, 5. The weaver is bounded by places.
func seamWorld(t *testing.T, places int64) (*World, skillCast) {
	t.Helper()
	c := trainingConfig(3)
	m := DefaultSettlements()
	m.Classes = []SettlementClass{
		{ID: "small", Enter: 0, Limits: []ClassLimit{{Name: "places", Value: places}}},
		{ID: "big", Enter: 1_000, Limits: []ClassLimit{{Name: "places", Value: 5}}},
	}
	c.Modules.Settlements = &m
	c.Roles[1].Ceiling = "places"
	w := newMill(t, c, WithSeed(exitSeed), WithAudit())
	step(t, w, 1)
	return w, castOf(t, w)
}

func seamEntry(t *testing.T, w *World) {
	t.Helper()
	if err := w.skillsEntry(0); err != nil {
		t.Fatalf("skillsEntry error = %v, want nil", err)
	}
}

// TestSkillHeadroom_TwoTraineesOneSlot verifies that the class limit is the slot of the
// completion and that two held trainees share it in order: the first completes, the
// second stays held although the slot was read before the first completed, and it
// completes the first entry step after the limit rises.
//
// Arithmetic: the weaver holds 1 household and the limit is 2, so the headroom is 1. The
// first trainee (applied at tick 2) takes it (weaver households 2, headroom 0) and the
// second (applied at tick 5) is held. A class change to big, 5 places, leaves 3.
//
// Requirement: FR-SKL-015, FR-SET-023
func TestSkillHeadroom_TwoTraineesOneSlot(t *testing.T) {
	t.Parallel()
	w, c := seamWorld(t, 2)
	first, second := c.homeMillers[1], c.homeMillers[0]
	w.skills.start(second, c.homeWeaver, 1, 4, 5)
	w.skills.start(first, c.homeWeaver, 1, 4, 2)
	seamEntry(t, w)
	if w.skills.isTrainee(first) || !w.skills.isTrainee(second) || w.agents[first].role != 1 {
		t.Errorf("after the first entry step: first trainee %v (role %d), second trainee %v, want first completed and second held",
			w.skills.isTrainee(first), w.agents[first].role, w.skills.isTrainee(second))
	}
	w.set.class[0] = 1
	seamEntry(t, w)
	if w.skills.isTrainee(second) || w.agents[second].role != 1 {
		t.Errorf("after the limit rose: second trainee %v (role %d), want completed", w.skills.isTrainee(second), w.agents[second].role)
	}
}

// TestSkillHeadroom_HeldCompletionWithClassLimit verifies the held path of plan 3-skl SK9
// with a real headroom: a trainee at TrainingTicks waits while the class limit of the
// target role is at or below its households and completes the first entry step after the
// limit rises.
//
// Arithmetic: a limit of 0 against 1 household leaves max(0, 0 - 1) = 0.
//
// Requirement: FR-SKL-015, FR-SET-023
func TestSkillHeadroom_HeldCompletionWithClassLimit(t *testing.T) {
	t.Parallel()
	w, c := seamWorld(t, 0)
	a := c.homeMillers[0]
	w.skills.start(a, c.homeWeaver, 1, 4, 2)
	seamEntry(t, w)
	if !w.skills.isTrainee(a) || w.skills.trainee[a].served != 4 {
		t.Fatalf("with a limit of 0: trainee %v served %d, want held at 4", w.skills.isTrainee(a), w.skills.trainee[a].served)
	}
	w.set.class[0] = 1
	seamEntry(t, w)
	if w.skills.isTrainee(a) {
		t.Error("after the limit rose the trainee is still held, want completed")
	}
}

// TestSkillHeadroom_NoCeilingNoLimit verifies that a role with no ceiling in the class,
// and a role of no ceiling at all, have no limit.
//
// Requirement: FR-SET-005
func TestSkillHeadroom_NoCeilingNoLimit(t *testing.T) {
	t.Parallel()
	w, _ := seamWorld(t, 2)
	if _, noLimit := w.roleHeadroom(0, 0); !noLimit {
		t.Error("roleHeadroom of a role with no Ceiling is limited, want no limit")
	}
	if h, noLimit := w.roleHeadroom(0, 1); noLimit || h != 1 {
		t.Errorf("roleHeadroom(home, weaver) = %d, noLimit %v, want 1, false", h, noLimit)
	}
}
