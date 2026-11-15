package economysim

import (
	"reflect"
	"testing"
)

// The worlds of the tenure tests: skillsConfig (two regions, home with the estate agent 0
// and yonder with the estate agent 1; home millers 2 to 4, home weaver 5, yonder millers 6
// and 7, yonder weaver 8, every household an agent of weight 1) with the resources module
// holding one resource "ore" in each region and the tenure module on.

// tenureHolders is the holdings of tenureWorldConfig: each home miller holds 100 basis
// points of the ore of home and the pool of home 1 000; each yonder miller holds 50 and the
// pool of yonder 500.
func tenureHoldings() []Holding {
	miller := HolderRef{Kind: HolderKindRole, Role: "miller"}
	pool := HolderRef{Kind: HolderKindPool}
	return []Holding{
		{Region: "home", Resource: "ore", Holder: pool, ShareBP: 1_000},
		{Region: "home", Resource: "ore", Holder: miller, ShareBP: 100},
		{Region: "yonder", Resource: "ore", Holder: pool, ShareBP: 500},
		{Region: "yonder", Resource: "ore", Holder: miller, ShareBP: 50},
	}
}

// tenureWorldConfig returns the config of the tenure tests with the module's rows mut
// edits.
func tenureWorldConfig(mut func(m *TenureConfig)) Config {
	c := skillsConfig()
	c.Roles[1].TrainingTicks = 0
	ore := DefaultResource()
	ore.ID, ore.Capacity = "ore", 1_000
	c.Modules.Resources = &Resources{
		HorizonYears: 50, Resources: []Resource{ore},
		Endowments: []Endowment{
			{Region: "home", Resource: "ore", HasCapacity: true, Capacity: 1_000, HasInitial: true, Initial: 1_000},
			{Region: "yonder", Resource: "ore", HasCapacity: true, Capacity: 1_000, HasInitial: true, Initial: 1_000},
		},
	}
	c.Modules.Estates = &Estates{}
	m := &TenureConfig{
		Terms:    []TenureTerm{{Region: "home", Resource: "ore", HasRent: true, Rent: 4}},
		Holdings: tenureHoldings(),
	}
	if mut != nil {
		mut(m)
	}
	c.Modules.Tenure = m
	return c
}

// tenureWorld builds the world of tenureWorldConfig with the options.
func tenureWorld(t testing.TB, mut func(m *TenureConfig), opts ...Option) *World {
	t.Helper()
	return newMill(t, tenureWorldConfig(mut), append([]Option{WithSeed(exitSeed), WithAudit()}, opts...)...)
}

// tnHolders reads the tenure of the endowment of ore in a region, failing the test on an error.
func tnHolders(t testing.TB, w *World, region RegionID) HoldingsState {
	t.Helper()
	st, err := w.Holders(region, 0)
	if err != nil {
		t.Fatalf("Holders(%d, 0) error = %v, want nil", region, err)
	}
	return st
}

// tnPairs is the holders and the shares of a HoldingsState as pairs in order.
func tnPairs(st HoldingsState) [][2]int64 {
	out := make([][2]int64, len(st.Titles))
	for i, x := range st.Titles {
		out[i] = [2]int64{int64(x.Holder), x.ShareBP}
	}
	return out
}

// tnMoney is the balance of an agent or an estate pseudo-agent.
func tnMoney(t testing.TB, w *World, id AgentID) int64 {
	t.Helper()
	acct := 0
	if int(id) < len(w.cfg.Regions) {
		acct = w.estates.account(RegionID(id))
	} else {
		acct = w.agents[id].account
	}
	v, err := w.ledger.Account(acct)
	if err != nil {
		t.Fatalf("Account of agent %d error = %v, want nil", id, err)
	}
	return v.Balance
}

// tnTotalMoney is the money of every non-boundary account, which no title sale changes.
func tnTotalMoney(t testing.TB, w *World) int64 {
	t.Helper()
	var sum int64
	for g := range w.cfg.Regions {
		sum += tnMoney(t, w, AgentID(g))
	}
	w.eachLive(func(a *agent) { sum += tnMoney(t, w, a.id) })
	return sum
}

// tnEq fails the test, naming what, when got is not deeply equal to want.
func tnEq[T any](t testing.TB, what string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %+v, want %+v", what, got, want)
	}
}
