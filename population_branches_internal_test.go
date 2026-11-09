package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/population"
)

// TestHasUpkeep_EachSource verifies that a configuration counts as having upkeep
// when the world, a role or a region lists any, and not otherwise, because the
// warning population_without_upkeep keys on it.
//
// Requirement: FR-POP-028
func TestHasUpkeep_EachSource(t *testing.T) {
	t.Parallel()
	item := []Item{{Commodity: "x", Quantity: 1}}
	for _, tt := range []struct {
		name string
		edit func(c *Config)
		want bool
	}{
		{"none", func(c *Config) {}, false},
		{"world", func(c *Config) { c.World.Upkeep = item }, true},
		{"role", func(c *Config) { c.Roles[0].Upkeep = item }, true},
		{"region", func(c *Config) { c.Regions[0].ExtraUpkeep = item }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := exitConfig()
			c.World.Upkeep, c.Roles[0].Upkeep = nil, nil
			for i := range c.Regions {
				c.Regions[i].ExtraUpkeep = nil
			}
			tt.edit(&c)
			if got := hasUpkeep(&c); got != tt.want {
				t.Errorf("hasUpkeep(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestResourcePlan_Refusals verifies that a resource whose plan share or rate is
// outside its bound, or whose abundance curve has no readable elasticity, gives no
// plan (so the plan warnings skip it), and that a readable one maps its shape.
//
// Requirement: FR-POP-030
func TestResourcePlan_Refusals(t *testing.T) {
	t.Parallel()
	c := exitConfig()
	good := Resource{PlanShareBP: 5_000, RateBP: 1_000, Shape: ShapeRefill,
		Abundance: []AbundanceKnot{{Share: 0, Factor: 0}, {Share: 10_000, Factor: 10_000}}}
	for _, tt := range []struct {
		name string
		edit func(r *Resource)
		want bool
	}{
		{"readable refill", func(r *Resource) {}, true},
		{"readable logistic", func(r *Resource) { r.Shape = ShapeLogistic }, true},
		{"share 0", func(r *Resource) { r.PlanShareBP = 0 }, false},
		{"share above whole", func(r *Resource) { r.PlanShareBP = wholeBP + 1 }, false},
		{"negative rate", func(r *Resource) { r.RateBP = -1 }, false},
		{"rate above bound", func(r *Resource) { r.RateBP = MaxRegenBP + 1 }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := good
			tt.edit(&r)
			if _, ok := resourcePlan(&c, &r); ok != tt.want {
				t.Errorf("resourcePlan(%s) ok = %v, want %v", tt.name, ok, tt.want)
			}
		})
	}
}

// TestCarryOf_UnseenAgent verifies that the death carry of an agent the step has
// not seen yet is the deterministic offset of its id, not zero and not a draw.
//
// Requirement: FR-POP-010
func TestCarryOf_UnseenAgent(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	id := AgentID(len(w.pop.deathCarry) + 7)
	if got, want := w.pop.carryOf(id), population.Offset(int64(id), w.pop.denom); got != want {
		t.Errorf("carryOf(%d) = %d, want %d", id, got, want)
	}
}

// TestPopRates_External verifies that an external agent has no rates, because it
// is never exposed to births or deaths.
//
// Requirement: FR-POP-023
func TestPopRates_External(t *testing.T) {
	t.Parallel()
	w := popWorld(t, DefaultPopulation())
	if b, d, ok := w.popRates(0, &RegionAgent{External: true}); ok || b != 0 || d != 0 {
		t.Errorf("popRates(external) = %d, %d, %v, want 0, 0, false", b, d, ok)
	}
}
