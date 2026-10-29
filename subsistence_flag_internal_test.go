package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// forageConfig is the small world of the flag and Work tests: one region "home",
// one role "forager" whose upkeep is 2 "grain" a tick, a market recipe "tend"
// (2 grain a tick, no input, so the pack is viable) and the own-use recipe "gather" (2 grain a tick,
// sited, yield 10 000), so Cov(gather, grain) = 2 * 10 000 / (1 * 2) = 10 000. The
// module is on with EnterTicks 2, ExitTicks 3 and RetryJobs 0. The households
// start with no grain and no money, so Secure is 0 and every one enters at the
// first flag step; the tests set the stock and money they need.
func forageConfig(households int64) Config {
	c := shellConfig()
	c.World.Upkeep = []Item{{Commodity: "grain", Quantity: 2}}
	c.World.OutputCoverTicks = 0
	c.Recipes = []Recipe{
		{ID: "gather", Duration: 1, Sited: true, TraineeYield: 5_000, Outputs: []Output{{Commodity: "grain", Quantity: 2, CostShare: 10_000}}},
		{ID: "tend", Duration: 1, TraineeYield: 5_000, Outputs: []Output{{Commodity: "grain", Quantity: 2, CostShare: 10_000}}},
	}
	c.Yields = []YieldProfile{{Region: "home", Recipe: "gather", HasAll: true, All: 10_000}}
	c.Regions[0].ReferenceBasket = 2_000 // the upkeep: 2 grain at 1 000
	c.Roles[0].ID, c.Roles[0].Recipes, c.Roles[0].StartMoney = "forager", []string{"tend"}, 0
	c.Groups = []Group{{Region: "home", Role: "forager", Households: households}}
	m := DefaultSubsistence()
	m.Recipes, m.EnterTicks, m.ExitTicks, m.RetryJobs = []string{"gather"}, 2, 3, 0
	c.Modules.Subsistence = &m
	return c
}

// newForage builds the world of forageConfig, with edit applied to the
// configuration first.
func newForage(t *testing.T, households int64, edit func(*Config), opts ...Option) *World {
	t.Helper()
	c := forageConfig(households)
	if edit != nil {
		edit(&c)
	}
	return newMill(t, c, opts...)
}

// holdGrain sets the grain an agent holds to n units, through the ledger's
// boundary as a test fixture does.
func holdGrain(t *testing.T, w *World, a *agent, n int64) {
	t.Helper()
	have, _, err := w.ledger.Holding(a.account, 0)
	if err != nil {
		t.Fatalf("Holding(agent %d) error = %v, want nil", a.id, err)
	}
	if n > have {
		giveGrain(t, w, a, n-have)
		return
	}
	if n == have {
		return
	}
	takeGrain(t, w, a, have-n)
}

// takeGrain takes n units of grain from an agent to the Disposal boundary.
func takeGrain(t *testing.T, w *World, a *agent, n int64) {
	t.Helper()
	to, ok := w.ledger.Boundary(ledger.Disposal, 0)
	if !ok {
		t.Fatalf("no Disposal boundary for grain")
	}
	if err := w.ledger.Move(a.account, to, 0, n, ledger.Disposal); err != nil {
		t.Fatalf("take %d grain from agent %d: %v", n, a.id, err)
	}
}

// TestFlag_EnterLeaveHysteresis verifies the mode of one household step by step:
// it enters when Secure is below EnterTicks, stays while Cover is below ExitTicks
// whatever its money, and leaves only when Cover reaches ExitTicks.
//
// Arithmetic: upkeep 2 a tick at the price 1 000, so Cover is stock / 2 and a
// million minor units afford 500 ticks. EnterTicks 2, ExitTicks 3. Stock 0 and
// no money: Secure 0 < 2, enters. Stock 4 (Cover 2) and money: the exit test is
// on Cover, 2 < 3, so it stays. Stock 6 (Cover 3): leaves, and stays out with
// Secure 3 + 500 whether or not it is hungry (Afford 0 leaves Secure 3 >= 2).
//
// Requirement: FR-SUB-013, FR-SUB-014
func TestFlag_EnterLeaveHysteresis(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	a := &w.agents[1]
	steps := []struct {
		name         string
		stock, money int64
		hunger       int64
		wantOn       bool
	}{
		{"nothing held enters", 0, 0, 0, true},
		{"cover below exit stays whatever the money", 4, 1_000_000, 0, true},
		{"cover at exit leaves", 6, 1_000_000, 0, false},
		{"a household with the exit cover stays out", 6, 1_000_000, 0, false},
		{"hunger alone does not take a household with the exit cover in", 6, 1_000_000, 1, false},
	}
	for _, s := range steps {
		holdGrain(t, w, a, s.stock)
		setMoney(t, w, a, s.money)
		a.hunger = s.hunger
		w.sub.grow(len(w.agents))
		w.p24OwnUse()
		if got := w.sub.ownUse[a.id]; got != s.wantOn {
			t.Errorf("%s: flag = %v, want %v", s.name, got, s.wantOn)
		}
	}
}

// TestFlag_CounterResetsOnChange verifies that the counter is 0 on entry and on
// exit and is kept while the flag does not change.
//
// Requirement: FR-SUB-013, FR-SUB-014, FR-SUB-022
func TestFlag_CounterResetsOnChange(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	a := &w.agents[1]
	w.sub.grow(len(w.agents))
	w.sub.ownUseJobs[a.id] = 7
	w.p24OwnUse() // enters: the counter was 7 and the flag false
	if !w.sub.ownUse[a.id] || w.sub.ownUseJobs[a.id] != 0 {
		t.Fatalf("after entry: flag = %v, counter = %d, want true and 0", w.sub.ownUse[a.id], w.sub.ownUseJobs[a.id])
	}
	w.sub.ownUseJobs[a.id] = 4
	w.p24OwnUse() // still short: the counter stays
	if w.sub.ownUseJobs[a.id] != 4 {
		t.Errorf("counter while in the mode = %d, want 4", w.sub.ownUseJobs[a.id])
	}
	holdGrain(t, w, a, 6)
	w.p24OwnUse()
	if w.sub.ownUse[a.id] || w.sub.ownUseJobs[a.id] != 0 {
		t.Errorf("after exit: flag = %v, counter = %d, want false and 0", w.sub.ownUse[a.id], w.sub.ownUseJobs[a.id])
	}
}

// TestFlag_AffordZeroWhenHungry verifies that a household that went unfed does
// not count its money: with 100 000 minor units it stays out of the mode, with
// Hunger 1 it enters.
//
// Requirement: FR-SUB-012
func TestFlag_AffordZeroWhenHungry(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	a := &w.agents[1]
	setMoney(t, w, a, 100_000)
	w.sub.grow(len(w.agents))
	w.p24OwnUse()
	if w.sub.ownUse[a.id] {
		t.Fatalf("flag with money and no hunger = true, want false")
	}
	a.hunger = 1
	w.p24OwnUse()
	if !w.sub.ownUse[a.id] {
		t.Errorf("flag with money and hunger = false, want true")
	}
}

// TestFlag_IndependentOfOthers verifies that the flag of a household is the same
// whatever the other households hold: two worlds that differ only in the state
// of agent 2 give agent 1 the same flag, counter and view figures.
//
// Requirement: FR-SUB-015
func TestFlag_IndependentOfOthers(t *testing.T) {
	t.Parallel()
	a, b := newForage(t, 2, nil), newForage(t, 2, nil)
	holdGrain(t, b, &b.agents[2], 1000)
	setMoney(t, b, &b.agents[2], 1_000_000)
	b.agents[2].hunger = 3
	for _, w := range []*World{a, b} {
		w.sub.grow(len(w.agents))
		w.p24OwnUse()
	}
	if a.sub.ownUse[1] != b.sub.ownUse[1] || a.sub.ownUseJobs[1] != b.sub.ownUseJobs[1] {
		t.Errorf("flag of agent 1 = (%v, %d) with another household changed, want (%v, %d)",
			b.sub.ownUse[1], b.sub.ownUseJobs[1], a.sub.ownUse[1], a.sub.ownUseJobs[1])
	}
	if !a.sub.ownUse[1] || b.sub.ownUse[2] {
		t.Errorf("flags = agent 1 %v, agent 2 (rich, fed) %v, want true and false", a.sub.ownUse[1], b.sub.ownUse[2])
	}
}

// TestFlag_CohortEqualsUnits verifies that a cohort of weight 3 and three agents
// of weight 1 in the same circumstances have the same Cover, Secure and flag.
//
// Arithmetic: upkeep 2 a tick; a cohort of 3 holding 12 grain has U = 6 and Cover
// 2; three households holding 4 each have U = 2 and Cover 2.
//
// Requirement: FR-SUB-016
func TestFlag_CohortEqualsUnits(t *testing.T) {
	t.Parallel()
	cohort := newForage(t, 3, func(c *Config) { c.World.Scale = 3 })
	units := newForage(t, 3, nil)
	if cohort.agents[1].weight != 3 || units.agents[1].weight != 1 {
		t.Skipf("weights = %d and %d, want 3 and 1: the scale does not make a cohort here", cohort.agents[1].weight, units.agents[1].weight)
	}
	holdGrain(t, cohort, &cohort.agents[1], 12)
	for i := 1; i <= 3; i++ {
		holdGrain(t, units, &units.agents[i], 4)
	}
	var v1, v2 AgentView
	for _, w := range []*World{cohort, units} {
		w.sub.grow(len(w.agents))
		w.p24OwnUse()
		w.refreshViewCtx()
	}
	if err := cohort.fillAgentView(&v1, &cohort.agents[1]); err != nil {
		t.Fatal(err)
	}
	if err := units.fillAgentView(&v2, &units.agents[1]); err != nil {
		t.Fatal(err)
	}
	if v1.Cover != v2.Cover || v1.Secure != v2.Secure || v1.OwnUse != v2.OwnUse {
		t.Errorf("cohort (cover %d, secure %d, flag %v), units (cover %d, secure %d, flag %v), want equal",
			v1.Cover, v1.Secure, v1.OwnUse, v2.Cover, v2.Secure, v2.OwnUse)
	}
}

// TestFlagStep_OrderIndependent verifies that the flag step is a function of the
// state alone: a second pass over unchanged state changes no flag, so the result
// does not depend on the order or the number of the visits.
//
// Requirement: FR-SUB-017
func TestFlagStep_OrderIndependent(t *testing.T) {
	t.Parallel()
	w := newForage(t, 4, nil)
	for i := 1; i <= 4; i++ {
		holdGrain(t, w, &w.agents[i], int64(i)*2)
	}
	w.sub.grow(len(w.agents))
	w.p24OwnUse()
	first := slices.Clone(w.sub.ownUse)
	w.p24OwnUse()
	if !slices.Equal(first, w.sub.ownUse) {
		t.Errorf("flags after a second pass = %v, want %v: the step is a function of the state", w.sub.ownUse, first)
	}
	if want := []bool{false, true, false, false, false}; !slices.Equal(first[:5], want) {
		t.Errorf("flags = %v, want %v: stocks 2, 4, 6, 8 give covers 1, 2, 3, 4 and no money, so only cover 1 is below EnterTicks 2", first[:5], want)
	}
}

// TestFlag_Eligibility verifies who is evaluated: an agent with no covered good
// in its region (the role lists no own-use recipe) never has the flag, and an
// estate pseudo-agent never does.
//
// Requirement: FR-SUB-010
func TestFlag_Eligibility(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, func(c *Config) { c.Modules.Subsistence.Roles = []RoleOwnUse{{Role: "forager"}} })
	w.sub.grow(len(w.agents))
	w.p24OwnUse()
	if w.sub.ownUse[0] || w.sub.ownUse[1] {
		t.Errorf("flags = %v, want none: the role has an empty row and the estate is no household", w.sub.ownUse[:2])
	}
}

// TestFlag_CoveredSetIsUpkeepAndOutputs verifies that the covered set of a
// role is the intersection of its upkeep and the outputs of its own-use list: an
// own-use recipe of a commodity the role does not eat covers nothing.
//
// Requirement: FR-SUB-010
func TestFlag_CoveredSetIsUpkeepAndOutputs(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, func(c *Config) {
		c.Commodities = append(c.Commodities, Commodity{ID: "ore", ReferencePrice: 1_000, Raw: true})
		c.Recipes = append(c.Recipes, Recipe{ID: "dig", Duration: 1, Sited: true, TraineeYield: 5_000, Outputs: []Output{{Commodity: "ore", Quantity: 1, CostShare: 10_000}}})
		c.Yields = append(c.Yields, YieldProfile{Region: "home", Recipe: "dig", HasAll: true, All: 10_000})
		c.Modules.Subsistence.Recipes = []string{"dig", "gather"}
	})
	if got, want := w.sub.covered[0], []CommodityID{0}; !slices.Equal(got, want) {
		t.Errorf("covered set = %v, want %v (grain is commodity 0, ore 1 is not eaten)", got, want)
	}
}

// TestDrops_RemovalRoleChangeWeightChange verifies the drops of 5.8: a removal
// zeroes the slot, a role change resets the counter and keeps the flag, and a
// weight change leaves both.
//
// Requirement: FR-SUB-032
func TestDrops_RemovalRoleChangeWeightChange(t *testing.T) {
	t.Parallel()
	w := newForage(t, 2, nil)
	w.sub.grow(len(w.agents))
	a, b := &w.agents[1], &w.agents[2]
	for _, x := range []*agent{a, b} {
		w.sub.ownUse[x.id], w.sub.ownUseJobs[x.id] = true, 4
	}
	w.sub.roleChanged(a)
	if !w.sub.ownUse[a.id] || w.sub.ownUseJobs[a.id] != 0 {
		t.Errorf("after a role change: flag = %v, counter = %d, want true and 0", w.sub.ownUse[a.id], w.sub.ownUseJobs[a.id])
	}
	a.weight++ // a weight change touches neither
	if !w.sub.ownUse[a.id] {
		t.Errorf("after a weight change the flag was lost")
	}
	if err := w.removeAgent(b, CauseStarved); err != nil {
		t.Fatalf("removeAgent error = %v, want nil", err)
	}
	if w.sub.ownUse[b.id] || w.sub.ownUseJobs[b.id] != 0 {
		t.Errorf("after a removal: flag = %v, counter = %d, want false and 0", w.sub.ownUse[b.id], w.sub.ownUseJobs[b.id])
	}
}

// TestDrops_Relocation verifies that a relocation sets the flag false and the
// counter 0.
//
// Requirement: FR-SUB-031
func TestDrops_Relocation(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, func(c *Config) {
		r := DefaultRegion()
		r.ID, r.ReferenceBasket = "away", 2_000
		c.Regions = append(c.Regions, r)
	})
	w.sub.grow(len(w.agents))
	a := &w.agents[2] // the first household: two estates come first
	w.sub.ownUse[a.id], w.sub.ownUseJobs[a.id] = true, 3
	if err := w.moveAgent(a, 1-w.regionOf(a), CauseConsumer); err != nil {
		t.Fatalf("moveAgent error = %v, want nil", err)
	}
	if w.sub.ownUse[a.id] || w.sub.ownUseJobs[a.id] != 0 {
		t.Errorf("after a relocation: flag = %v, counter = %d, want false and 0", w.sub.ownUse[a.id], w.sub.ownUseJobs[a.id])
	}
}

// giveGrain gives an agent n units of grain from the Initial boundary, as a
// fixture does.
func giveGrain(t *testing.T, w *World, a *agent, n int64) {
	t.Helper()
	from, ok := w.ledger.Boundary(ledger.Initial, 0)
	if !ok {
		t.Fatalf("no Initial boundary for grain")
	}
	if err := w.ledger.Move(from, a.account, 0, n, ledger.Initial); err != nil {
		t.Fatalf("give %d grain to agent %d: %v", n, a.id, err)
	}
}

// setMoney sets an agent's money to n through the Initial boundary.
func setMoney(t *testing.T, w *World, a *agent, n int64) {
	t.Helper()
	b, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatalf("no Initial boundary for money")
	}
	acct, err := w.ledger.Account(a.account)
	if err != nil {
		t.Fatalf("Account(%d): %v", a.account, err)
	}
	from, to, amt := b, a.account, n-acct.Balance
	if amt < 0 {
		from, to, amt = a.account, b, -amt
	}
	if amt == 0 {
		return
	}
	if err := w.ledger.Transfer(from, to, amt, ledger.Initial); err != nil {
		t.Fatalf("set money of agent %d to %d: %v", a.id, n, err)
	}
}
