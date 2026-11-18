package economysim_test

import (
	"bytes"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// quietFacilityConfig is infrastructureConfig with no recipe that uses a class and no
// store row, with the module on or off, so that nothing in the world reads a facility
// and the only difference between the two worlds is the records themselves.
func quietFacilityConfig(on bool) economysim.Config {
	c := infrastructureConfig()
	clause := func(id string) bool { return id == "build_mill" || id == "repair_mill" || id == "grind" }
	c.Recipes = slices.DeleteFunc(c.Recipes, func(r economysim.Recipe) bool { return clause(r.ID) })
	m := roleOf(&c, "grower")
	m.Recipes = slices.DeleteFunc(m.Recipes, clause)
	if !on {
		c.Modules.Infrastructure = nil
		return c
	}
	c.Modules.Infrastructure.Initial = slices.DeleteFunc(c.Modules.Infrastructure.Initial, func(r economysim.InitialFacility) bool { return r.Class == "barn" })
	return c
}

// TestInfrastructure_NoDraws verifies that the module takes no random number and moves
// no money or goods (spec-infrastructure.md 10, FR-CAP-002, FR-CAP-044): after 30 ticks
// with initial facilities of a site, a route and a ground kind, every section of the
// checkpoint but the module's own (tag 27) equals the section of the same
// world with the module off, byte for byte, the ledger, the agents and the seed among
// them.
//
// Requirement: FR-CAP-002, FR-CAP-044
func TestInfrastructure_NoDraws(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	has27 := map[bool]bool{}
	sections := func(c economysim.Config) map[uint64][]byte {
		w := newPackWorld(t, c, 3)
		step(t, w, 30)
		out := map[uint64][]byte{}
		for _, s := range decodeFrame(t, mustMarshal(t, mustCheckpoint(t, w))).sections {
			if s.tag == 27 {
				has27[c.Modules.Infrastructure != nil] = true
				continue
			}
			out[s.tag] = s.body
		}
		return out
	}
	want, got := sections(quietFacilityConfig(false)), sections(quietFacilityConfig(true))
	if has27[false] || !has27[true] {
		t.Errorf("section 27 present with the module off = %t and on = %t, want false and true", has27[false], has27[true])
	}
	if len(want) != len(got) {
		t.Fatalf("sections with the module = %d, off = %d, want the same set apart from tag 27", len(got), len(want))
	}
	for tag, body := range want {
		if !bytes.Equal(body, got[tag]) {
			t.Errorf("section %d differs between the world with initial facilities and the module off", tag)
		}
	}
}
