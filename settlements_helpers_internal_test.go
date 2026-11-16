package economysim

import (
	"testing"
)

// This file unit-tests the small helpers under the readers and the seams of the
// settlements module (spec-settlements.md 6.1, 6.4.2): the name lookup, the room of a
// view, and the per-region household count with its incremental update.

// TestNameIndex verifies the lookup of a Needs or Ceiling name in the sorted names of the
// module: its index, and -1 for the empty name and for a name the module does not have.
//
// Guard: the index feeds regionHas and regionLimit as is; a missing name that returned 0
// would silently gate every such role on the first flag.
//
// Requirement: FR-SET-005
func TestNameIndex(t *testing.T) {
	t.Parallel()
	sorted := []string{"fair", "market", "port"}
	for _, tt := range []struct {
		name, key string
		want      int
	}{
		{"first", "fair", 0},
		{"last", "port", 2},
		{"empty name is no reader", "", -1},
		{"unknown name", "zoo", -1},
		{"between two names", "gate", -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := nameIndex(sorted, tt.key); got != tt.want {
				t.Errorf("nameIndex(%v, %q) = %d, want %d", sorted, tt.key, got, tt.want)
			}
		})
	}
	if got := nameIndex(nil, "fair"); got != -1 {
		t.Errorf("nameIndex(nil, %q) = %d, want -1", "fair", got)
	}
}

// TestRoomOf verifies the form of the ceiling that the spawner takes: -1 for a role with
// no limit, else the headroom, never negative.
//
// Guard: a negative headroom taken as room would read as "no ceiling" in occupation.Admit.
//
// Requirement: FR-SET-023
func TestRoomOf(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		view RoleView
		want int64
	}{
		{"no limit", RoleView{NoLimit: true, Headroom: 7}, -1},
		{"headroom", RoleView{Headroom: 7}, 7},
		{"full role", RoleView{Headroom: 0}, 0},
		{"negative headroom clamps to 0", RoleView{Headroom: -3}, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := roomOf(&tt.view); got != tt.want {
				t.Errorf("roomOf(%+v) = %d, want %d", tt.view, got, tt.want)
			}
		})
	}
}

// TestRoleCount_ScansOncePerTickAndRegion verifies that the households of a role are
// counted from the agents, kept for the rest of the (tick, region) pair, and counted
// again for another tick or region.
//
// Arithmetic: in region home of exitConfig the miller holds 3 households and the weaver 1.
// A planted value in the cache is returned for the same tick and region (no rescan) and
// replaced after a step.
//
// Requirement: NFR-SET-001
func TestRoleCount_ScansOncePerTickAndRegion(t *testing.T) {
	t.Parallel()
	w := readerWorld(t, 1_000, nil)
	s := w.set
	if got := s.roleCount(w, 0, 0); got != 3 {
		t.Fatalf("roleCount(home, miller) = %d, want 3", got)
	}
	if got := s.roleCount(w, 0, 1); got != 1 {
		t.Errorf("roleCount(home, weaver) = %d, want 1", got)
	}
	s.cnt[0] = 99
	if got := s.roleCount(w, 0, 0); got != 99 {
		t.Errorf("second roleCount in the same tick and region = %d, want the held 99 (no rescan)", got)
	}
	step(t, w, 1)
	if got := s.roleCount(w, 0, 0); got == 99 {
		t.Errorf("roleCount after a tick = 99, want a fresh scan")
	}
}

// TestNoteRoleMove verifies the incremental update after a completed traineeship: it
// moves the weight between the roles of the held region, floors the source at 0, and does
// nothing when no count is held or the count is of another region.
//
// Guard: a count that fell below 0 would inflate the headroom of the next held trainee.
//
// Requirement: FR-SET-023
func TestNoteRoleMove(t *testing.T) {
	t.Parallel()
	w := readerWorld(t, 1_000, nil)
	s := w.set
	s.cntOK = false
	s.noteRoleMove(0, 0, 1, 1)
	if s.cnt[0] != 0 || s.cnt[1] != 0 {
		t.Errorf("noteRoleMove with no count held changed cnt to %v, want unchanged zeros", s.cnt)
	}
	s.roleCount(w, 0, 0)
	s.noteRoleMove(0, 0, 1, 1)
	if s.cnt[0] != 2 || s.cnt[1] != 2 {
		t.Errorf("after moving 1 household miller to weaver: cnt = %v, want [2 2]", s.cnt)
	}
	s.noteRoleMove(0, 0, 1, 5)
	if s.cnt[0] != 0 || s.cnt[1] != 7 {
		t.Errorf("after moving 5 households from a role of 2: cnt = %v, want [0 7] (floored at 0)", s.cnt)
	}
	other := s.cntRegion + 1
	s.noteRoleMove(other, 1, 0, 3)
	if s.cnt[0] != 0 || s.cnt[1] != 7 {
		t.Errorf("a move in region %d changed the count held for region %d: cnt = %v, want [0 7]", other, s.cntRegion, s.cnt)
	}
}

// TestRoleHeadroom_ClassWithoutTheLimit verifies that a role whose ceiling names a limit
// the region's class has no row for is not limited in the slot of a completion, and is
// limited once the region's class carries it.
//
// Arithmetic: class small has no limit, so the slot reads no limit; class big carries 5
// places and the weaver holds 1 household, so after the class changes the headroom is
// 5 - 1 = 4.
//
// Requirement: FR-SET-005, FR-SET-023
func TestRoleHeadroom_ClassWithoutTheLimit(t *testing.T) {
	t.Parallel()
	c := trainingConfig(3)
	m := DefaultSettlements()
	m.Classes = []SettlementClass{
		{ID: "small", Enter: 0},
		{ID: "big", Enter: 1_000, Limits: []ClassLimit{{Name: "places", Value: 5}}},
	}
	c.Modules.Settlements = &m
	c.Roles[1].Ceiling = "places"
	w := newMill(t, c, WithSeed(exitSeed))
	step(t, w, 1)
	if _, noLimit := w.roleHeadroom(0, 1); !noLimit {
		t.Error("roleHeadroom(home, weaver) in a class with no limit is limited, want no limit")
	}
	w.set.class[0] = 1
	if h, noLimit := w.roleHeadroom(0, 1); noLimit || h != 4 {
		t.Errorf("roleHeadroom(home, weaver) in big = %d, noLimit %v, want 4, false", h, noLimit)
	}
}

// TestStoreClassGate_NilWithoutNeeds verifies that the store gate is nil, and lockStores
// takes nothing, when the storage module is off or no store class has a Needs, so every
// class stays open.
//
// Requirement: FR-STO-044, FR-SET-021
func TestStoreClassGate_NilWithoutNeeds(t *testing.T) {
	t.Parallel()
	off, _ := seamWorld(t, 2)
	if off.storeClassGate() != nil {
		t.Error("storeClassGate() with the storage module off is non-nil, want nil")
	}
	w, rec := stoWorld(t, func(c *Config) {
		m := DefaultSettlements()
		m.Classes = []SettlementClass{{ID: "hamlet"}}
		c.Modules.Settlements = &m
	})
	if w.sto.gate != nil {
		t.Error("a world with the module on and no Needs has a store gate, want nil")
	}
	for _, e := range of[AgentSpawned](rec) {
		if e.StoresLocked != 0 {
			t.Errorf("AgentSpawned.StoresLocked of agent %d = %d, want 0 with no gate", e.Agent, e.StoresLocked)
		}
	}
}
