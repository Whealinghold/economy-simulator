package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// outsideRegions adds the region "coast", joined to north both ways, and an external
// module whose one node sits there, so the region is an outside node.
func outsideRegions(c *economysim.Config) {
	r := economysim.DefaultRegion()
	r.ID = "coast"
	c.Regions = append(c.Regions, r)
	c.Edges = append(c.Edges, economysim.Edge{From: "north", To: "coast", Distance: 30}, economysim.Edge{From: "coast", To: "north", Distance: 30})
	ore := economysim.DefaultOutsideLine("mineral", 837)
	ore.Sell = economysim.OutsideSide{MarginBP: 500, Capacity: 10}
	c.Modules.External = &economysim.External{Nodes: []economysim.OutsideNode{{Region: "coast", Lines: []economysim.OutsideLine{ore}}}}
}

// TestValidateInfrastructure_RowAndClauseReferences verifies two rules of
// spec-infrastructure.md 13.1 that the field table does not reach: an initial row
// whose region or whose toward region is an outside node is ErrOutsideNode (a
// facility is built on the world's own ground), and a recipe with a builds or repairs
// clause while the module is off is ErrModuleDependency at the clause.
//
// Requirement: FR-CAP-003, FR-CAP-010
//
// Guard: the clause check and the outside-node check each have a branch per field; a
// branch that was dropped would accept a facility in an outside node, or a clause that
// nothing can run, and the other cases of the table would still pass.
func TestValidateInfrastructure_RowAndClauseReferences(t *testing.T) {
	t.Parallel()
	moduleOff := func(c *economysim.Config) {
		c.Modules.Infrastructure = nil
		recipeOf(c, "grind").Needs = nil
	}
	tests := []struct {
		name   string
		change func(c *economysim.Config)
		rule   error
		path   []string
		value  string
	}{
		{"a row in an outside node", func(c *economysim.Config) {
			outsideRegions(c)
			c.Modules.Infrastructure.Initial[0].Region = "coast"
		}, economysim.ErrOutsideNode, facilityRowPathT(0, "region"), "coast"},
		{"a route toward an outside node", func(c *economysim.Config) {
			outsideRegions(c)
			c.Modules.Infrastructure.Initial[3].Toward = "coast"
		}, economysim.ErrOutsideNode, facilityRowPathT(3, "toward"), "coast"},
		{"a builds clause with the module off", moduleOff, economysim.ErrModuleDependency, facilityRecipePath("build_mill", "builds"), "mill"},
		{"a repairs clause with the module off", moduleOff, economysim.ErrModuleDependency, facilityRecipePath("repair_mill", "repairs"), "mill"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := infrastructureConfig()
			tt.change(&c)
			err := c.Validate()
			if !errors.Is(err, tt.rule) {
				t.Fatalf("Validate() = %v, want %v", err, tt.rule)
			}
			fc := fieldCase{rules: []error{tt.rule}, paths: [][]string{tt.path}, value: tt.value}
			if !errorAt(err, fc) {
				t.Errorf("Validate() = %v, want %v at %v with value %q", err, tt.rule, tt.path, tt.value)
			}
		})
	}
}

// TestInfrastructureWarnings_RouteModeOfAMerchant verifies that a route class whose
// only mode is carried by a merchant block is not warned as route_unused_mode, and
// that the same class is warned once no merchant lists the mode.
//
// Requirement: FR-CAP-040
//
// Guard: the warning reads the merchant blocks and the hired modes; if it read the
// hired modes alone, a road that serves a merchant's own mode would be reported as
// useless.
func TestInfrastructureWarnings_RouteModeOfAMerchant(t *testing.T) {
	t.Parallel()
	withMerchant := func(modes ...string) economysim.Config {
		c := infrastructureConfig()
		facilityOf(&c, "road").Modes = []string{"foot"}
		c.Modules.Information = &economysim.Information{InfoSpeed: 40, RumourMaxAge: 20}
		m := economysim.DefaultMerchants()
		c.Modules.Merchants = &m
		r := economysim.DefaultRole()
		r.ID, r.Informed, r.Trades = "broker", true, []string{"staple", "seed"}
		r.MerchantFeeShare = economysim.DefaultMerchantFeeShare
		b := economysim.DefaultMerchant()
		b.Modes, b.TakeUnits = modes, 10
		r.Merchant = &b
		c.Roles = append(c.Roles, r)
		c.Groups = append(c.Groups, economysim.Group{Region: "north", Role: "broker", Households: 1, Scale: 1})
		return c
	}
	const warned = "route_unused_mode@modules.infrastructure.class.road"
	tests := []struct {
		name  string
		modes []string
		want  bool
	}{
		{"a merchant carries the mode", []string{"foot"}, false},
		{"no merchant carries the mode", []string{"coach"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := withMerchant(tt.modes...)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
			if got := slices.Contains(facilityWarningsOf(c), warned); got != tt.want {
				t.Errorf("Warnings() has %s = %v, want %v (%v)", warned, got, tt.want, facilityWarningsOf(c))
			}
		})
	}
}
