package economysim_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestValidateInfrastructure_Errors verifies each rule of spec-infrastructure.md 13.1
// with a configuration that breaks only that rule: the sentinel by errors.Is, the
// path of the offending field and the value.
//
// Requirement: FR-CAP-003, FR-CAP-004, FR-CAP-010, FR-CFG-031
func TestValidateInfrastructure_Errors(t *testing.T) {
	t.Parallel()
	for _, tt := range infrastructureFieldCases() {
		t.Run(tt.field+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			var c economysim.Config
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tt.rules[0]) {
				t.Fatalf("Validate() = %v, want %v", err, tt.rules[0])
			}
			if !errorAt(err, tt) {
				t.Errorf("Validate() = %v, want %v at %v with value %q and bound %q", err, tt.rules[0], tt.paths[0], tt.value, tt.bound)
			}
		})
	}
}

// TestValidateInfrastructure_Order verifies that a phase that finds an error ends the
// run (spec-infrastructure.md 13.1): a configuration with a mistake of an earlier
// phase and one of a later phase returns only the earlier one, so the first error is
// the same for the same mistake.
//
// Requirement: FR-CAP-004
func TestValidateInfrastructure_Order(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		change       func(c *economysim.Config)
		want, absent error
	}{
		{"bounds before shape", func(c *economysim.Config) {
			facilityOf(c, "mill").Durability = 1
			facilityOf(c, "road").BoostBP = 0
		}, economysim.ErrOutOfBounds, economysim.ErrFacilityShape},
		{"shape before references", func(c *economysim.Config) {
			facilityOf(c, "road").BoostBP = 0
			c.Modules.Infrastructure.Initial[1].Class = "nowhere"
		}, economysim.ErrFacilityShape, economysim.ErrUnknownFacilityClass},
		{"references before dependencies", func(c *economysim.Config) {
			c.Modules.Infrastructure.Initial[1].Class = "nowhere"
			c.Modules.Storage = nil
		}, economysim.ErrUnknownFacilityClass, economysim.ErrModuleDependency},
		{"dependencies before counts", func(c *economysim.Config) {
			c.Modules.Storage = nil
			recipeOf(c, "build_mill").Inputs = nil
		}, economysim.ErrModuleDependency, economysim.ErrFacilityNoCost},
		{"a repeated group does not make a holder count of its own", func(c *economysim.Config) {
			c.Modules.Infrastructure.Initial[2].Count = 1_000
			for _, g := range slices.Clone(c.Groups) {
				if g.Role == "grower" {
					c.Groups = append(c.Groups, g)
				}
			}
		}, economysim.ErrDuplicateID, economysim.ErrFacilityCount},
		// Fix: a builds clause on a class that is not defined, with a toward set, got
		// ErrFacilityShape from phase 2 and never reached the unknown class of phase 3.
		{"an unknown class on a clause with a toward is a reference", func(c *economysim.Config) {
			recipeOf(c, "build_mill").Builds = &economysim.FacilityBuild{Class: "nowhere", Toward: "south"}
		}, economysim.ErrUnknownFacilityClass, economysim.ErrFacilityShape},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := infrastructureConfig()
			tt.change(&c)
			err := c.Validate()
			if !errors.Is(err, tt.want) || errors.Is(err, tt.absent) {
				t.Errorf("Validate() = %v, want %v and not %v", err, tt.want, tt.absent)
			}
		})
	}
}

// TestRecipeClauses_Errors verifies the rules B1 to B7 of spec-infrastructure.md
// 4.1.2 on the recipe clauses, with the cases of the field table that name a clause.
//
// Requirement: FR-CAP-010
func TestRecipeClauses_Errors(t *testing.T) {
	t.Parallel()
	n := 0
	for _, tt := range infrastructureFieldCases() {
		if !strings.HasPrefix(tt.field, "Recipe.") && !strings.HasPrefix(tt.field, "Facility") || !strings.HasPrefix(tt.paths[0][0], "recipes") {
			continue
		}
		n++
		t.Run(tt.field+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			var c economysim.Config
			tt.mutate(&c)
			if err := c.Validate(); !errors.Is(err, tt.rules[0]) {
				t.Errorf("Validate() = %v, want %v", err, tt.rules[0])
			}
		})
	}
	if n < 12 {
		t.Errorf("TestRecipeClauses_Errors ran %d cases, want the clause cases of B1 to B7 (at least 12)", n)
	}
}

// TestRecipeClauses_NoOutputIsValid verifies rule B3: a recipe with Builds or
// Repairs and no output at all is valid (ErrRecipeNoOutput is waived), one with only
// discard outputs is valid, and a recipe with neither clause and no output is still
// ErrRecipeNoOutput.
//
// Requirement: FR-CAP-010
func TestRecipeClauses_NoOutputIsValid(t *testing.T) {
	t.Parallel()
	c := infrastructureConfig()
	if len(recipeOf(&c, "build_mill").Outputs) != 0 {
		t.Fatalf("build_mill has outputs, want none")
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil for a build recipe with no output", err)
	}
	recipeOf(&c, "repair_mill").Outputs = []economysim.Output{{Commodity: "waste", Quantity: 1, Discard: true}}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for a repair recipe with a discard output", err)
	}
	none := infrastructureConfig()
	recipeOf(&none, "grind").Outputs = nil
	if err := none.Validate(); !errors.Is(err, economysim.ErrRecipeNoOutput) {
		t.Errorf("Validate() = %v, want ErrRecipeNoOutput for a recipe with no clause and no output", err)
	}
}

// FuzzInfrastructureValidate verifies that Validate, Warnings and HashConfig never
// panic on a configuration whose class, row and clause numbers are arbitrary, that a
// configuration that validates hashes the same twice, and that every error of the
// module carries a path.
//
// Requirement: FR-CAP-003, FR-CAP-004, NFR-CAP-003
func FuzzInfrastructureValidate(f *testing.F) {
	f.Add("site", int64(1000), int64(1), int64(2), int64(0), int64(1), int64(5))
	f.Add("route", int64(-1), int64(1)<<62, int64(0), int64(100_001), int64(0), int64(9))
	f.Add("", int64(0), int64(0), int64(0), int64(0), int64(-5), int64(1)<<62)
	f.Fuzz(func(t *testing.T, kind string, dur, decay, capacity, boost, count, units int64) {
		c := infrastructureConfig()
		m := facilityOf(&c, "mill")
		m.Kind, m.Durability, m.Decay, m.Capacity, m.BoostBP = kind, dur, decay, capacity, boost
		c.Modules.Infrastructure.Initial[2].Count = count
		recipeOf(&c, "repair_mill").Repairs.Units = units
		recipeOf(&c, "grind").Needs = slices.Repeat([]string{"mill"}, int(count&3))
		err := c.Validate()
		_ = c.Warnings()
		first, second := economysim.HashConfig(c), economysim.HashConfig(c)
		if err == nil && first != second {
			t.Fatalf("HashConfig is not deterministic for a valid configuration")
		}
		var ve *economysim.ValidationError
		for _, e := range unwrapAll(err) {
			if errors.As(e, &ve) && len(ve.Path) == 0 {
				t.Fatalf("a validation error has no path: %v", e)
			}
		}
	})
}
