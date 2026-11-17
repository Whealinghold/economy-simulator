package economysim

import (
	"reflect"
	"slices"
	"testing"
)

// The tests of the start, the view and the hooks of the tenure module: the reservation of
// rent, the exemption, the cost anchor, the strategy, the welfare, the demand and the
// principal. The worlds and the replay of a tick are in tenure_ration_internal_test.go.

// TestStart_RentReservedAgainstOtherStarts verifies that a start reserves its rent in a
// scratch balance of its principal, which the next start of the same principal reads
// (spec-land.md 5.4.1), and that the reservation is dropped at the next snapshot.
//
// Requirement: FR-LND-016
func TestStart_RentReservedAgainstOtherStarts(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	tn := w.tenure
	if got := tn.spendable(1); got != 10_000 {
		t.Fatalf("spendable before a reservation = %d, want the start money 10 000", got)
	}
	tn.reserve(1, 600)
	tn.reserve(1, 400)
	if got := tn.spendable(1); got != 9_000 {
		t.Errorf("spendable after reserving 1 000 = %d, want 9 000", got)
	}
	tn.reserve(1, 100_000)
	if got := tn.spendable(1); got != 0 {
		t.Errorf("spendable with more reserved than held = %d, want 0", got)
	}
	tn.entitlements()
	if got := tn.reservedOf(1); got != 0 {
		t.Errorf("reserved after the next snapshot = %d, want 0", got)
	}
}

// TestOwnUseExempt_NoRentNoPayee verifies spec-land.md 5.3.4: with OwnUseExempt a request
// for a recipe of the principal's own-use list draws let units and pays no rent, and the
// lessor receives nothing, though the units still come out of its unused entitlement. The
// own-use list is planted on the role, as the subsistence module builds it.
//
// Requirement: FR-LND-018
func TestOwnUseExempt_NoRentNoPayee(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithAudit())
	w.tenure.ownUseExempt = true
	row := w.roles.row(0)
	row.ownUse = []RecipeID{0}
	pool := w.estates.account(0)
	before, _ := w.ledger.Account(pool)
	toStart(t, w)
	c := w.tenure.rt.counts[0]
	if c.letUnits != 3 || c.exemptUnits != 3 || c.paid != 0 {
		t.Errorf("counts = %+v, want 3 let units, all exempt, and no rent paid", c)
	}
	after, _ := w.ledger.Account(pool)
	if after.Balance != before.Balance {
		t.Errorf("pool balance = %d, want %d (an exempt unit pays its lessor nothing)", after.Balance, before.Balance)
	}
}

// TestCostAnchor_AddsRentForHolderUnitsToo_ZeroRentNoChange verifies spec-land.md 5.7.1:
// the cost anchor of the output gains the rent of the recipe's draws (3 units at the
// rent in force), so it is above the anchor with no rent, and with a rent of 0 it equals
// the anchor of a world without the module.
//
// Requirement: FR-LND-020
func TestCostAnchor_AddsRentForHolderUnitsToo_ZeroRentNoChange(t *testing.T) {
	t.Parallel()
	anchor := func(c Config) int64 {
		w := newMill(t, c, WithSeed(1))
		w.seedWage[0] = 100 // a wage of 100 a tick: the anchor is ceil((2 * 1 000 + 5 * 100 + rent of 3 units) / 3)
		w.p12Anchors()
		return w.prod.unitCost(0, flour)
	}
	off := anchor(oreMill(5, 10, 1_000)) // ceil(2 500 / 3) = 834
	zero := rationConfig(1_000, 5_000, false)
	zero.Modules.Tenure.Terms[0].Rent = 0
	if got := anchor(zero); got != off {
		t.Errorf("anchor with a rent of 0 = %d, want %d (the world without the module)", got, off)
	}
	paid := rationConfig(1_000, 5_000, false)
	paid.Modules.Tenure.Terms[0].Rent = 30
	if got, want := anchor(paid), int64(864); got != want { // ceil((2 500 + 3 * 30) / 3)
		t.Errorf("anchor with a rent of 30 on 3 units = %d, want %d", got, want)
	}
}

// TestDefaultStrategy_RecipeNetPaysPostedRent_SourceStaysEligible verifies spec-land.md
// 4.6.2: the net value of one execution falls by the rent of the recipe's draws, and a
// source recipe (no inputs) stays eligible at a net of 0 or below, so rent moves the
// choice among recipes and does not stop a household that must work to eat.
//
// Requirement: FR-LND-020
func TestDefaultStrategy_RecipeNetPaysPostedRent_SourceStaysEligible(t *testing.T) {
	t.Parallel()
	var v AgentView
	for _, tt := range []struct {
		name string
		rent int64
		want int64
	}{{"no rent", 0, 0}, {"rent 7", 7, -7}} {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := v.recipeNet(&RecipeView{Rent: tt.rent})
			if !ok || got != tt.want {
				t.Errorf("recipeNet(Rent %d) = %d, %v, want %d, true", tt.rent, got, ok, tt.want)
			}
		})
	}
}

// TestView_OwnRegionOnly_NoHolderShare verifies spec-land.md 4.6.1: the endowment of a
// view carries the rent in force, the agent's own share and entitlement and whether it
// is closed, and no type of the view carries a list of holdings or another holder's share.
//
// Requirement: FR-LND-022
func TestView_OwnRegionOnly_NoHolderShare(t *testing.T) {
	t.Parallel()
	c := rationConfig(100, 0, false)
	c.Modules.Tenure.Holdings = []Holding{{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: 500}}
	w := newMill(t, c, WithSeed(1))
	step(t, w, 1)
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[1], viewWork); err != nil {
		t.Fatalf("fillAgentView error = %v, want nil", err)
	}
	if len(v.Endowments) != 1 {
		t.Fatalf("Endowments = %+v, want one row", v.Endowments)
	}
	e := v.Endowments[0]
	if e.Posted != 2 || e.ShareBP != 500 || e.Closed || e.Entitled != 5 {
		t.Errorf("endowment view = %+v, want rent 2, share 500, entitled 5 (500 bp of 100 units) and open", e)
	}
	if got, want := v.Recipes[0].Rent, int64(6); got != want {
		t.Errorf("recipe rent = %d, want %d (3 units at 2)", got, want)
	}
}

// TestView_CarriesNoListOfHoldings verifies by reflection that no view type has a field
// named for a list of holdings, so a strategy cannot read another holder's share.
//
// Requirement: FR-LND-022
func TestView_CarriesNoListOfHoldings(t *testing.T) {
	t.Parallel()
	for _, typ := range []any{EndowmentView{}, AgentView{}} {
		tv := reflect.TypeOf(typ)
		for i := range tv.NumField() {
			if f := tv.Field(i); slices.Contains([]string{"Holdings", "Holders", "Titles"}, f.Name) {
				t.Errorf("%s has a field %s, want none: a view carries no list of holdings", tv.Name(), f.Name)
			}
		}
	}
}

// TestWelfare_RentPaidNettedRentReceivedUnearned verifies spec-land.md 5.7.3 at the start:
// the principal's rent paid is an input spend of the tick, and a holder that is an agent
// receives it as an unearned receipt (the pool, which has no welfare record, as nothing).
//
// Requirement: FR-LND-021
func TestWelfare_RentPaidNettedRentReceivedUnearned(t *testing.T) {
	t.Parallel()
	c := rationConfig(10, 0, false)
	c.Modules.Tenure.Holdings = []Holding{{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: 500}}
	c.Modules.Tenure.Terms[0].Rent = 5
	w := newMill(t, c, WithSeed(1), WithAudit())
	toStart(t, w)
	var spent, unearned int64
	for slot := 1; slot <= 10; slot++ {
		f := w.welfare.at(slot)
		spent += f.inputSpend
		unearned += f.unearned
	}
	cnt := w.tenure.rt.counts[0]
	if cnt.paid == 0 {
		t.Fatalf("rent paid = 0, want a tenant that paid (counts %+v)", cnt)
	}
	if spent != cnt.paid {
		t.Errorf("input spend of the households = %d, want the rent paid %d", spent, cnt.paid)
	}
	if unearned > cnt.paid {
		t.Errorf("unearned receipts = %d, want at most the rent paid %d (a tenant that is its own lessor receives none)", unearned, cnt.paid)
	}
}

// TestDemand_HolderRoleNoClassWithoutIncome verifies that a role named by a holding is
// not raised as a class without income (spec-land.md 5.7.3, spec-demand.md 10.2), through
// the function the demand index reads.
//
// Requirement: FR-LND-021
func TestDemand_HolderRoleNoClassWithoutIncome(t *testing.T) {
	t.Parallel()
	c := rationConfig(10, 5_000, false)
	if holdsTitle(&c, "miller") {
		t.Errorf("holdsTitle(miller) = true with only the pool holding, want false")
	}
	c.Modules.Tenure.Holdings = append(slices.Clone(c.Modules.Tenure.Holdings),
		Holding{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: 100})
	if !holdsTitle(&c, "miller") {
		t.Errorf("holdsTitle(miller) = false with a holding of the role, want true")
	}
	c.Modules.Tenure = nil
	if holdsTitle(&c, "miller") {
		t.Errorf("holdsTitle(miller) = true with the module off, want false")
	}
}

// TestPrice_ExhaustedWarnConditionWithTenure verifies spec-land.md 13.1.4: with tenure
// data the Warn "endowment exhausted" reads Short less the units of unused entitlement
// held or idle, and an endowment with no title reads Short as RES does.
//
// Requirement: FR-LND-032
func TestPrice_ExhaustedWarnConditionWithTenure(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, true), WithSeed(1))
	tn := w.tenure
	tn.entitlements() // sizes the counts of the tick
	tn.rt.counts[0] = tenureCount{held: 5}
	for _, tt := range []struct{ short, want int64 }{{20, 15}, {5, 0}, {0, 0}} {
		if got := tn.shortAdjusted(0, tt.short); got != tt.want {
			t.Errorf("shortAdjusted(short %d) with 5 held = %d, want %d", tt.short, got, tt.want)
		}
	}
	tn.holds[0] = nil
	if got := tn.shortAdjusted(0, 20); got != 20 {
		t.Errorf("shortAdjusted(20) with no title = %d, want 20 (RES's condition)", got)
	}
}

// TestStart_LargestKTheBalanceCovers_NothingTakenBeyond verifies the cut of spec-land.md
// 5.4.1, which the clip of the ration cannot see: one household of weight 2 with a balance
// of 14 draws on two titled endowments, ore (3 units an execution, rent 2: 6) and wood
// (2 units, rent 2: 4). The clip, endowment by endowment, leaves floor(14 / 6) = 2 and
// floor(14 / 4) = 3 executions, so the ration grants 2, but 2 executions cost
// 2 * (6 + 4) = 20 > 14 and 1 costs 10 <= 14: the start takes 1 execution, reserves 10,
// and the 6 units of the second execution (3 of ore, 2 of wood, as let units) are cut:
// 5 units. The rent paid is 10, from the household to the two pools.
//
// Requirement: FR-LND-016
func TestStart_LargestKTheBalanceCovers_NothingTakenBeyond(t *testing.T) {
	t.Parallel()
	c := twoResourceMill(2, 100, 100)
	c.Groups[0].Scale = 2 // one agent of weight 2
	c.World.MinAgents = 1
	c.Roles[0].StartMoney = 7 // per household: the agent of weight 2 holds 14
	c.Recipes[0].Duration = 5 // the job outlives its start tick, so its executions can be read
	c.Modules.Estates = &Estates{}
	pool := HolderRef{Kind: HolderKindPool}
	c.Modules.Tenure = &TenureConfig{
		Terms: []TenureTerm{{Region: "home", Resource: "ore", HasRent: true, Rent: 2}, {Region: "home", Resource: "wood", HasRent: true, Rent: 2}},
		// the pool holds every unit: the commons and the holders' own tiers are empty
		Holdings: []Holding{{Region: "home", Resource: "ore", Holder: pool, ShareBP: 10_000}, {Region: "home", Resource: "wood", Holder: pool, ShareBP: 10_000}},
	}
	w := newMill(t, c, WithSeed(1), WithAudit())
	if len(w.agents) <= 1 || w.agents[1].weight != 2 {
		t.Fatalf("agents = %d, weight of the first household = %d, want one household of weight 2", len(w.agents), w.agents[1].weight)
	}
	toStart(t, w)
	if got := w.agents[1].job.executions; got != 1 {
		t.Errorf("executions started = %d, want 1 (the largest k whose rent 10 the balance 14 covers)", got)
	}
	ore, wood := w.tenure.rt.counts[0], w.tenure.rt.counts[1]
	if ore.paid != 6 || wood.paid != 4 {
		t.Errorf("rent paid on ore, wood = %d, %d, want 6 and 4", ore.paid, wood.paid)
	}
	if ore.cutUnits != 3 || wood.cutUnits != 2 {
		t.Errorf("let units cut on ore, wood = %d, %d, want 3 and 2 (the second execution)", ore.cutUnits, wood.cutUnits)
	}
	if v, _ := w.ledger.Account(w.agents[1].account); v.Balance != 4 {
		t.Errorf("balance after the rent = %d, want 4 (14 less 10)", v.Balance)
	}
}

// TestCost_TenureTiersAllocs verifies NFR-LND-001: with the default logger and observer the
// entitlements of P2.3 and the four tiers of P2.6 for a titled endowment allocate nothing
// once the scratch of the module has grown.
//
// Requirement: NFR-LND-001
func TestCost_TenureTiersAllocs(t *testing.T) {
	// No t.Parallel: AllocsPerRun changes GOMAXPROCS while it measures.
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1))
	toStart(t, w) // grows the scratch of the tick
	r := w.res
	order := slices.Clone(r.order)
	got := testing.AllocsPerRun(20, func() {
		w.tenure.entitlements()
		if !w.tenure.ration(order) {
			t.Fatalf("ration() = false, want the tiers of a titled endowment")
		}
	})
	if got != 0 {
		t.Errorf("allocs of the entitlements and the tiers = %v, want 0", got)
	}
}

// TestPrincipal_HiredOrTrainedJobUsesPrincipal verifies spec-land.md 4.3, 4.7.2 and D8 of
// the plan with a planted traineeship (the hire form is EMP's): the request of a trainee
// carries its trainer as principal, so the trainee's let units are charged to the trainer's
// account and the trainee, whose balance is not touched, pays nothing. The pool holds the
// whole endowment, so every unit is let at the posted rent of 4: the trainee's one
// execution of 3 units costs its trainer 12.
//
// Requirement: FR-LND-012
func TestPrincipal_HiredOrTrainedJobUsesPrincipal(t *testing.T) {
	t.Parallel()
	c := tenureWorldConfig(func(m *TenureConfig) {
		m.Holdings = []Holding{{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindPool}, ShareBP: 10_000}}
	})
	c.Recipes[0].Draws = []Draw{{Resource: "ore", Quantity: 3}}
	c.Recipes[0].Requires = nil // a trainee-eligible recipe requires no tool (ErrEligibleHasTool)
	c.Commodities = slices.DeleteFunc(c.Commodities, func(x Commodity) bool { return x.ID == "pan" })
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 10}}
	c.Recipes[0].TraineeEligible = true
	c.Roles[1].TrainingTicks = 4 // a role that trains gives the world the state of the traineeships
	c.Roles[1].Recipes = []string{"mill"}
	c.Roles[1].StartStock = []Item{{Commodity: "grain", Quantity: 10}}
	w := newMill(t, c, WithSeed(exitSeed), WithAudit())
	cast := castOf(t, w)
	trainee, trainer := cast.homeMillers[0], cast.homeWeaver
	w.skills.start(trainee, trainer, 1, 0, 100)
	start := tnMoney(t, w, trainee)
	trainerBefore := tnMoney(t, w, trainer)
	toStart(t, w)
	found := false
	for _, q := range w.res.reqs {
		if q.agent == trainee {
			found = true
			if q.principal != trainer {
				t.Errorf("principal of the trainee's request = %d, want its trainer %d", q.principal, trainer)
			}
		}
	}
	if !found {
		t.Fatalf("no request of the trainee %d on the endowment, want one", trainee)
	}
	if got := tnMoney(t, w, trainee); got != start {
		t.Errorf("trainee balance = %d, want %d (its principal pays the rent)", got, start)
	}
	if got := trainerBefore - tnMoney(t, w, trainer); got < 12 {
		t.Errorf("trainer paid %d, want at least 12 (4 per unit of the trainee's 3 units)", got)
	}
}

// TestLeastRent_ZeroWhenOwnEntitlementCovers verifies the bound of spec-land.md 4.7.2: the
// least rent of a job is the rent in force on the units left after the principal's
// entitlement and the whole commons, and 0 when its entitlement covers the draw.
//
// Arithmetic: a stock of 100 with each of ten households holding 500 basis points and the
// commons 5 000: each is entitled to 5 units, the commons to 50. Three executions of 3
// units are 9 units: the entitlement covers 5, the commons the other 4, so the least rent
// is 0. Forty executions are 120 units: 120 - 5 - 50 = 65 units at the rent of 2, 130.
//
// Requirement: FR-LND-014
func TestLeastRent_ZeroWhenOwnEntitlementCovers(t *testing.T) {
	t.Parallel()
	c := rationConfig(100, 0, false)
	c.Modules.Tenure.Holdings = []Holding{{Region: "home", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: "miller"}, ShareBP: 500}}
	w := newMill(t, c, WithSeed(1))
	toStart(t, w)
	for _, tt := range []struct {
		name       string
		executions int64
		want       int64
	}{{"covered by the entitlement and the commons", 3, 0}, {"above both", 40, 130}} {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := w.tenure.leastRent(1, 0, tt.executions, 3)
			if !ok || got != tt.want {
				t.Errorf("leastRent(1, 0, %d, 3) = %d, %v, want %d, true", tt.executions, got, ok, tt.want)
			}
		})
	}
}
