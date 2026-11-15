package economysim

import (
	"testing"
)

// This file tests the readers of the class (spec-settlements.md 6): the gate and the
// ceiling of a role in the view and in the population module's births, on the small
// world of exitConfig (region home: three millers in two agents, one weaver). The
// region holds 4 households, so with the classes of readerSettlements it is "small" and,
// when bigAt is 4, "big".

// readerSettlements gives classes small (Enter 0, a limit of 4 places) and big (Enter
// bigAt, the flag fair and a limit of 50 places), the default band and a dwell of 30.
func readerSettlements(bigAt int64) *Settlements {
	m := DefaultSettlements()
	m.Classes = []SettlementClass{
		{ID: "small", Enter: 0, Limits: []ClassLimit{{Name: "places", Value: 4}}},
		{ID: "big", Enter: bigAt, Flags: []string{"fair"}, Limits: []ClassLimit{{Name: "places", Value: 50}}},
	}
	return &m
}

// readerWorld is the world of exitConfig with the settlements module at bigAt, the
// miller needing fair and bounded by places, and the weaver closed by its own field.
func readerWorld(t *testing.T, bigAt int64, mut func(c *Config)) *World {
	t.Helper()
	c := exitConfig()
	c.Modules.Settlements = readerSettlements(bigAt)
	c.Roles[0].Needs, c.Roles[0].Ceiling = "fair", "places"
	if mut != nil {
		mut(&c)
	}
	return newMill(t, c, WithSeed(exitSeed))
}

// TestGate_NeedsClosesRoleInRegion verifies H1: a role whose Needs flag the region's
// class lacks is closed in the view of that region, and open when the class carries it.
//
// Arithmetic: the region holds 3 + 1 = 4 households. With big at 1 000 it is small (no
// flag) and the miller is closed; with big at 4 it is big (fair) and the miller is open.
//
// Requirement: FR-SET-022
func TestGate_NeedsClosesRoleInRegion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		bigAt  int64
		closed bool
	}{{"class without the flag", 1_000, true}, {"class with the flag", 4, false}} {
		w := readerWorld(t, tt.bigAt, nil)
		if got := w.buildRegionView(0).Roles[0].Closed; got != tt.closed {
			t.Errorf("%s: Roles[miller].Closed = %v, want %v", tt.name, got, tt.closed)
		}
	}
}

// TestGate_ClosedByEitherField verifies that a role is closed if Role.Closed or the
// gate says so: the weaver (Closed) stays closed where its Needs flag is carried, and
// a role with neither field is open.
//
// Requirement: FR-SET-022
func TestGate_ClosedByEitherField(t *testing.T) {
	t.Parallel()
	w := readerWorld(t, 4, func(c *Config) { c.Roles[1].Needs = "fair" })
	roles := w.buildRegionView(0).Roles
	if !roles[1].Closed || roles[0].Closed {
		t.Errorf("Closed = %v (weaver), %v (miller), want true by Role.Closed with the flag carried, false", roles[1].Closed, roles[0].Closed)
	}
	w = readerWorld(t, 1_000, func(c *Config) { c.Roles[0].Needs = "" })
	if w.buildRegionView(0).Roles[0].Closed {
		t.Error("a role with no Needs is closed in a class without the flag, want open")
	}
}

// TestGate_HouseholdsStay verifies R3: a gated role keeps its households. A step in a
// region whose class lacks the flag removes none of the miller's three households.
//
// Requirement: FR-SET-022
func TestGate_HouseholdsStay(t *testing.T) {
	t.Parallel()
	w := readerWorld(t, 1_000, nil)
	step(t, w, 1)
	var millers int64
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if a := &w.agents[i]; a.role == 0 {
			millers += a.weight
		}
	}
	if millers != 3 {
		t.Errorf("households of the gated miller after a step = %d, want 3", millers)
	}
}

// TestRoleView_HeadroomOnlyWithModule verifies that the view says NoLimit for every
// role while the module is off and for a role with no ceiling, and gives the headroom
// of the class limit less the role's households otherwise.
//
// Arithmetic: the class small has 4 places and the miller holds 3 households, so the
// headroom is 1; in class big it is 50 - 3 = 47. The weaver has no ceiling.
//
// Requirement: FR-SET-018, FR-SET-023
func TestRoleView_HeadroomOnlyWithModule(t *testing.T) {
	t.Parallel()
	for _, r := range newMill(t, exitConfig(), WithSeed(exitSeed)).buildRegionView(0).Roles {
		if !r.NoLimit || r.Headroom != 0 {
			t.Errorf("module off: role %d Headroom %d NoLimit %v, want 0 and true", r.Role, r.Headroom, r.NoLimit)
		}
	}
	for _, tt := range []struct {
		bigAt, headroom int64
	}{{1_000, 1}, {4, 47}} {
		roles := readerWorld(t, tt.bigAt, nil).buildRegionView(0).Roles
		if roles[0].NoLimit || roles[0].Headroom != tt.headroom || !roles[1].NoLimit {
			t.Errorf("big at %d: miller Headroom %d NoLimit %v, weaver NoLimit %v, want %d, false and true",
				tt.bigAt, roles[0].Headroom, roles[0].NoLimit, roles[1].NoLimit, tt.headroom)
		}
	}
}

// TestPopulation_BirthsCeilingCounted verifies H2 in the population module (N-348,
// FR-POP-041): the births a role ceiling drops are counted in BirthsCeiling and not in
// BirthsCapped, after the household cap, and a role closed by Needs does not advance
// its birth carry (FR-POP-007).
//
// Arithmetic: the miller carry is set to 2D - 1 (D - 1 in the gated variant, a carry below D as the step leaves it) with D the carry denominator; the
// miller's exposed households add a numerator that carries it past 2D, so two births
// are due. The headroom is 1 (4 places less 3 households): one birth is kept and one
// dropped by the ceiling, and none by the cap. In the gated variant the numerator is 0,
// so no birth is formed and the carry stays where it was.
//
// Requirement: FR-POP-007, FR-POP-041, FR-SET-023
func TestPopulation_BirthsCeilingCounted(t *testing.T) {
	t.Parallel()
	for _, gated := range []bool{false, true} {
		w := readerWorld(t, 1_000, func(c *Config) {
			p := DefaultPopulation()
			c.Modules.Population = &p
			c.World.EntryPoolShare = 10_000
			if !gated {
				c.Roles[0].Needs = ""
			}
		})
		for id := 1; id <= 3; id++ {
			setRatio(w, AgentID(id), 14_000)
		}
		popView(w, 0)
		d := w.pop.denom
		carry := 2*d - 1
		if gated {
			carry = d - 1 // a carry is always below D between steps
		}
		setCarries(w, nil, map[RoleID]int64{0: carry})
		v := popView(w, 0)
		c := w.pop.cnt[0]
		switch {
		case gated:
			if v.Roles[0].PopBirths != 0 || w.pop.birthCarry[0] != carry {
				t.Errorf("gated: PopBirths %d carry %d, want 0 and the carry %d unmoved", v.Roles[0].PopBirths, w.pop.birthCarry[0], carry)
			}
		case v.Roles[0].PopBirths != 1 || c.birthsDue != 2 || c.birthsCeiling != 1 || c.birthsCapped != 0:
			t.Errorf("ceiling: PopBirths %d due %d ceiling %d capped %d, want 1, 2, 1, 0", v.Roles[0].PopBirths, c.birthsDue, c.birthsCeiling, c.birthsCapped)
		}
	}
}
