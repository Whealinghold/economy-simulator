package steady_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// Commodity indexes of the V1 cases: 0 seed, 1 staple, 2 ore, 3 hoe.
const (
	seed = iota
	staple
	ore
	hoe
)

// TestUnproducible verifies V1 on hand-made chains: a seed cycle with no source,
// a Raw marker that sources it, a crop that grows nowhere, the producer of a
// tool's input and the difference between the realm and a region.
//
// Requirement: FR-VIA-004
func TestUnproducible(t *testing.T) {
	t.Parallel()
	grow := steady.Maker{Region: 0, Needs: []int{seed}, Makes: []int{staple}, Listed: true}
	save := steady.Maker{Region: 0, Needs: []int{staple}, Makes: []int{seed}, Listed: true}
	tests := []struct {
		name   string
		realm  steady.Realm
		upkeep []int
		want   []steady.Missing
	}{
		// grow needs seed, save needs staple: neither can start, so seed (needed by
		// grow, index 0) and staple (upkeep, and needed by save, index 1) are missing.
		{"seed cycle with no source", steady.Realm{Commodities: 2, Regions: 1, Makers: []steady.Maker{grow, save}}, []int{staple},
			[]steady.Missing{{Commodity: seed, By: 0}, {Commodity: staple, By: 1}}},
		// A Raw marker on seed sources it, and then staple follows.
		{"raw marker sources the cycle", steady.Realm{Commodities: 2, Regions: 1, Sourced: []bool{true, false}, Makers: []steady.Maker{grow, save}}, []int{staple}, nil},
		// A maker with no needs is a source by itself.
		{"no-input activity is a source", steady.Realm{Commodities: 2, Regions: 1, Makers: []steady.Maker{grow, save, {Region: 0, Makes: []int{seed}}}}, []int{staple}, nil},
		// The crop grows nowhere: the caller leaves its activity out, so staple has
		// no maker; only the upkeep needs it (By -1).
		{"crop that grows nowhere", steady.Realm{Commodities: 2, Regions: 1}, []int{staple}, []steady.Missing{{Commodity: staple, By: -1}}},
		// hoe is made from ore, ore from nothing, and staple needs the hoe as a tool:
		// the producer of the tool's input counts.
		{"tool input is produced", steady.Realm{Commodities: 4, Regions: 1, Makers: []steady.Maker{
			{Region: 0, Makes: []int{ore}}, {Region: 0, Needs: []int{ore}, Makes: []int{hoe}},
			{Region: 0, Needs: []int{hoe}, Makes: []int{staple}, Listed: true}}}, []int{staple}, nil},
		// Same chain with no ore source: hoe (a tool of the listed maker 1) and staple
		// (upkeep) are missing; ore is needed only by the unlisted maker 0, so it is not.
		{"tool input is not produced", steady.Realm{Commodities: 4, Regions: 1, Makers: []steady.Maker{
			{Region: 0, Needs: []int{ore}, Makes: []int{hoe}},
			{Region: 0, Needs: []int{hoe}, Makes: []int{staple}, Listed: true}}}, []int{staple},
			[]steady.Missing{{Commodity: staple, By: -1}, {Commodity: hoe, By: 1}}},
		// An unlisted activity's needs are not required (no role runs it).
		{"unlisted maker needs nothing", steady.Realm{Commodities: 2, Regions: 1, Makers: []steady.Maker{{Region: 0, Needs: []int{seed}, Makes: []int{staple}}}}, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := steady.Unproducible(tt.realm, tt.upkeep)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Unproducible(%+v, %v) = %v, want %v", tt.realm, tt.upkeep, got, tt.want)
			}
		})
	}
}

// TestProducible_RealmIsPooled verifies that the realm reading is optimistic
// (free trade, spec-viability.md 4.6) while the region reading is autarky.
//
// Requirement: FR-VIA-004
// Region 0 makes seed from nothing; region 1 makes staple from seed. Staple is
// producible in the realm (seed travels) and in no region alone: region 0 cannot
// make it, region 1 has no seed.
func TestProducible_RealmIsPooled(t *testing.T) {
	t.Parallel()
	r := steady.Realm{Commodities: 2, Regions: 2, Makers: []steady.Maker{
		{Region: 0, Makes: []int{seed}},
		{Region: 1, Needs: []int{seed}, Makes: []int{staple}},
	}}
	got := steady.Producible(r)
	if want := []bool{true, true}; !slices.Equal(got.Realm, want) {
		t.Errorf("Producible().Realm = %v, want %v", got.Realm, want)
	}
	if want := []bool{true, false}; !slices.Equal(got.Region[0], want) {
		t.Errorf("Producible().Region[0] = %v, want %v", got.Region[0], want)
	}
	if want := []bool{false, false}; !slices.Equal(got.Region[1], want) {
		t.Errorf("Producible().Region[1] = %v, want %v", got.Region[1], want)
	}
}

// TestCannotWork verifies the runnable check: a role with households where none
// of its recipes exists, the first such region, and the exemption of a role with
// no recipes.
//
// Requirement: FR-VIA-005
func TestCannotWork(t *testing.T) {
	t.Parallel()
	// exists[region][recipe]: recipe 0 runs in region 0 only, recipe 1 nowhere.
	exists := [][]bool{{true, false}, {false, false}, {false, false}}
	tests := []struct {
		name  string
		roles []steady.RoleUse
		want  []steady.Stuck
	}{
		{"runs where it lives", []steady.RoleUse{{Regions: []int{0}, Recipes: []int{0, 1}}}, nil},
		{"none in region 1", []steady.RoleUse{{Regions: []int{0, 1, 2}, Recipes: []int{0}}}, []steady.Stuck{{Role: 0, Region: 1}}},
		{"recipe that exists nowhere", []steady.RoleUse{{Regions: []int{0}, Recipes: []int{1}}}, []steady.Stuck{{Role: 0, Region: 0}}},
		{"role with no recipes is exempt", []steady.RoleUse{{Regions: []int{1, 2}}}, nil},
		{"two roles, second stuck", []steady.RoleUse{{Regions: []int{0}, Recipes: []int{0}}, {Regions: []int{2}, Recipes: []int{0}}}, []steady.Stuck{{Role: 1, Region: 2}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := steady.CannotWork(tt.roles, exists)
			if !slices.Equal(got, tt.want) {
				t.Errorf("CannotWork(%+v) = %v, want %v", tt.roles, got, tt.want)
			}
		})
	}
}
