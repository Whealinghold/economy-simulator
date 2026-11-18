package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestInfrastructureHash verifies the hash of the module (spec-config.md 8.2.2,
// FR-CAP-001): off leaves the pinned hash of the reference configuration, an empty
// block that is present differs from a nil one, a change of a class field, an initial
// row or a recipe clause changes the hash, a clause with the module off changes
// nothing, and two spellings of one configuration hash alike (a field at zero is not
// written, so `decay = 0` and no decay are the same class; the classes, rows, modes
// and needs are canonical).
//
// Requirement: FR-CAP-001, FR-CFG-104
func TestInfrastructureHash(t *testing.T) {
	t.Parallel()
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Fatalf("hash of the reference configuration = %s, want the pinned %s", got, pinnedReferenceHash)
	}
	on := referenceConfig()
	on.Modules.Infrastructure = &economysim.Infrastructure{}
	if hexHash(on) == pinnedReferenceHash {
		t.Errorf("an empty module that is on hashes as the module off")
	}
	base := hexHash(infrastructureConfig())
	changes := map[string]func(c *economysim.Config){
		"a class field":      func(c *economysim.Config) { facilityOf(c, "mill").Capacity++ },
		"a class kind":       func(c *economysim.Config) { facilityOf(c, "barn").Kind = "site" },
		"a class mode":       func(c *economysim.Config) { facilityOf(c, "road").Modes = []string{"foot"} },
		"a class resource":   func(c *economysim.Config) { facilityOf(c, "drain").Resource = "stand" },
		"an initial count":   func(c *economysim.Config) { c.Modules.Infrastructure.Initial[0].Count++ },
		"an initial role":    func(c *economysim.Config) { c.Modules.Infrastructure.Initial[1].Role = "cook" },
		"an initial toward":  func(c *economysim.Config) { c.Modules.Infrastructure.Initial[3].Toward = "" },
		"a build class":      func(c *economysim.Config) { recipeOf(c, "build_mill").Builds.Class = "road" },
		"a build toward":     func(c *economysim.Config) { recipeOf(c, "build_mill").Builds.Toward = "south" },
		"a repair unit":      func(c *economysim.Config) { recipeOf(c, "repair_mill").Repairs.Units++ },
		"a need":             func(c *economysim.Config) { recipeOf(c, "grind").Needs = nil },
		"the module removed": func(c *economysim.Config) { c.Modules.Infrastructure = nil },
	}
	for name, change := range changes {
		c := infrastructureConfig()
		change(&c)
		if hexHash(c) == base {
			t.Errorf("hash after changing %s = the unchanged hash, want it to move", name)
		}
	}
}

// TestInfrastructureHash_Spellings verifies that two spellings of one
// configuration hash alike: a class with `decay` 0 written and one that omits it, the
// classes, rows, modes and needs in another order, and a clause with the module off
// against no clause (a field of a module that is off is not written).
//
// Requirement: FR-CAP-001, FR-CFG-104
func TestInfrastructureHash_Spellings(t *testing.T) {
	t.Parallel()
	base := hexHash(infrastructureConfig())
	reordered := infrastructureConfig()
	m := reordered.Modules.Infrastructure
	m.Classes[0], m.Classes[3] = m.Classes[3], m.Classes[0]
	m.Initial[0], m.Initial[2] = m.Initial[2], m.Initial[0]
	if hexHash(reordered) != base {
		t.Errorf("hash with the classes and rows reordered differs from the same configuration in order")
	}
	foot, coach := infrastructureConfig(), infrastructureConfig()
	facilityOf(&foot, "road").Modes = []string{"foot", "coach"}
	facilityOf(&coach, "road").Modes = []string{"coach", "foot"}
	if hexHash(foot) != hexHash(coach) {
		t.Errorf("hash with the modes of a route in two orders differs")
	}
	zero := infrastructureConfig()
	facilityOf(&zero, "mill").Decay = 0
	if hexHash(zero) == base {
		t.Errorf("hash with decay 0 = the hash with decay 1, want them to differ")
	}
	if hexHash(zero) != hexHash(zeroSpelled()) {
		t.Errorf("hash with decay written as 0 differs from the same class with no decay")
	}
	off := referenceConfig()
	clause := referenceConfig()
	recipeOf(&clause, "cut_fuel").Builds = &economysim.FacilityBuild{Class: "mill"}
	recipeOf(&clause, "cut_fuel").Needs = []string{"mill"}
	if hexHash(off) != hexHash(clause) {
		t.Errorf("hash with a clause and the module off differs from the same configuration with none")
	}
}

// zeroSpelled is infrastructureConfig with the mill's decay left out, which the
// struct spells as 0.
func zeroSpelled() economysim.Config {
	c := infrastructureConfig()
	*facilityOf(&c, "mill") = economysim.FacilityClass{ID: "mill", Kind: "site", Durability: 1_000, WearPerUse: 2, Capacity: 2}
	return c
}
