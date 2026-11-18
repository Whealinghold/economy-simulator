package economysim

import (
	"testing"
)

// facSeed is the seed of the infrastructure tests of the root package; the failure
// messages name it (testing.md 6.2).
const facSeed = 20261118

// facConfig is a world of two regions (home and yonder, joined by an edge each way) with
// the transport, resources and storage modules on and the infrastructure module with one
// class of each kind: a mill (site, capacity 2, decay 1, wear 2 a use), a road (route,
// the mode foot), a drain (ground, on ore) and a barn (store, installing a bin). The
// home region has five millers of weight 1 and a weaver, yonder two millers and a weaver.
// No recipe uses a class and no initial row exists: a test adds its own rows through mut.
// The numbers are placeholders (N-015).
func facConfig(mut func(m *Infrastructure)) Config {
	c := tenureWorldConfig(nil)
	c.Modules.Tenure = nil
	c = withTransport(c)
	c.Edges = []Edge{{From: "home", To: "yonder", Distance: 40}, {From: "yonder", To: "home", Distance: 40}}
	c.Groups = []Group{
		{Region: "home", Role: "miller", Households: 5, Scale: 1},
		{Region: "home", Role: "weaver", Households: 1},
		{Region: "yonder", Role: "miller", Households: 2, Scale: 1},
		{Region: "yonder", Role: "weaver", Households: 1},
	}
	c.Modules.Storage = &Storage{
		Roles:   []RoleRoom{{Role: "miller", Base: &Capacity{Mass: 2_000, Bulk: 2_000}}},
		Classes: []StoreClass{{ID: "bin", Mass: 400, Bulk: 300}},
	}
	m := &Infrastructure{Classes: []FacilityClass{
		{ID: "barn", Kind: "store", Durability: 800, Store: "bin"},
		{ID: "drain", Kind: "ground", Durability: 500, BoostBP: 3_000, Resource: "ore"},
		{ID: "mill", Kind: "site", Durability: 1_000, Decay: 1, WearPerUse: 2, Capacity: 2},
		{ID: "road", Kind: "route", Durability: 2_000, BoostBP: 6_000, Modes: []string{"foot"}},
	}}
	if mut != nil {
		mut(m)
	}
	c.Modules.Infrastructure = m
	return c
}

// facWorld builds the world of facConfig with the audit on and the options.
func facWorld(t testing.TB, mut func(m *Infrastructure), opts ...Option) *World {
	t.Helper()
	return newMill(t, facConfig(mut), append([]Option{WithSeed(facSeed), WithAudit()}, opts...)...)
}

// facAgents are the live agents of the role in the region, in ascending agent ID.
func facAgents(w *World, region RegionID, role string) []AgentID {
	ri := -1
	for i := range w.cfg.Roles {
		if w.cfg.Roles[i].ID == role {
			ri = i
		}
	}
	var out []AgentID
	w.eachLive(func(a *agent) {
		if int(a.role) == ri && w.regionOf(a) == region {
			out = append(out, a.id)
		}
	})
	return out
}
