package economysim

import (
	"slices"
	"testing"
)

// The commodities of the mill of spec-saving-and-investment.md 16.1 in canonical order, at the
// belief centres of the example, and its recipes.
const (
	mFood CommodityID = iota
	mTimber
	mStone
	mGrain
	mFlour
	mCarry
	mTool
)

const (
	rCarry RecipeID = iota
	rGrind
	rBuild
	rFix
)

// millView is the household of example A of 16.1 as a view the rules read: a miller of weight 1
// with an upkeep cost of 30 (one food at 30), ReserveTicks 3 (reserve 90), the recipes carry
// (net 8 a tick, a source), grind (needs the class mill; 4 grain at 10 into 3 flour at 20, net
// 20), build_mill (30 timber at 8 and 10 stone at 12, duration 5, builds a mill) and fix_mill
// (4 timber a execution, restores 100), and the block HurdleBP 15 000, PaybackTicks 180,
// RepairBelowBP 5 000 with a SafetyTicks of 2. The mill is a site of Durability 600, Decay 1,
// WearPerUse 1 and Capacity 6. The ring holds 30 records with a cumulative surplus of 420 (a
// mean of 14 a tick, 16.5). Every number is a placeholder (N-015).
func millView(money int64) *AgentView {
	t := testTunables()
	v := &AgentView{
		Agent: 1, Weight: 1, Money: money, StartsIn: 1, Tunables: t,
		Upkeep: []UpkeepRow{{Commodity: mFood, Quantity: 1}},
		Commodities: []CommodityView{
			cview(mFood, 0, 30, 0), cview(mTimber, 0, 8, 0), cview(mStone, 0, 12, 0), cview(mGrain, 0, 10, 0),
			cview(mFlour, 0, 20, 0), cview(mCarry, 0, 8, 0), cview(mTool, 0, 50, 0),
		},
		Recipes: []RecipeView{
			{Recipe: rCarry, Duration: 1, Effective: 10_000, MaxExecutions: 5, Outputs: []RecipeOutput{{Commodity: mCarry, Quantity: 1}}},
			{Recipe: rGrind, Duration: 1, Effective: 10_000, Needs: []FacilityClassID{0},
				Inputs:  []RecipeLine{{Commodity: mGrain, Quantity: 4}},
				Outputs: []RecipeOutput{{Commodity: mFlour, Quantity: 3}}},
			{Recipe: rBuild, Duration: 5, Effective: 10_000, Builds: BuildView{On: true, Class: 0, Toward: NoRegion},
				Inputs: []RecipeLine{{Commodity: mTimber, Quantity: 30}, {Commodity: mStone, Quantity: 10}}},
			{Recipe: rFix, Duration: 1, Effective: 10_000, Repairs: RepairView{On: true, Class: 0, Units: 100},
				Inputs: []RecipeLine{{Commodity: mTimber, Quantity: 4}}},
		},
		Saving: &SavingView{
			Block:       Saver{PrecautionCapTicks: 30, HurdleBP: 15_000, PaybackTicks: 180, RepairBelowBP: 5_000},
			SafetyTicks: 2, Records: 30, Surplus: 420,
			Classes: []SavingClass{{Class: 0, Name: "mill", Kind: "site", Durability: 600, Decay: 1, WearPerUse: 1, Capacity: 6}},
		},
	}
	// The world derives nothing_to_repair for a repair of a class of which the household holds
	// no worn facility (spec-infrastructure.md 4.2.2); a test that gives it a worn mill clears it.
	v.Recipes[rFix].Standing = ReasonNothingToRepair
	return v
}

// wornMill is the mill of 16.2: condition 120 of 600, so cap = ceil(6 * 120 / 600) = 2.
func wornMill(v *AgentView) {
	v.Saving.Facilities = []FacilityView{{ID: 1, Class: 0, To: NoRegion, Condition: 120, Durability: 600, Capacity: 2}}
	v.Recipes[rFix].Standing = ""
}

// TestTarget_Example16_1 verifies the target of the mill of 16.1 at a money of 100: the floor
// is the reserve, 90; the build passes with a benefit of 2 160 against a cost of 400 and an
// outlay of 600, so the goal is 600 and S* = 690. The household is not funded (100 < 690).
//
// Arithmetic: pv(grind) = 20 * 10 000 = 200 000, pv(carry) = 80 000, dJ = 200 000 - 80 000 =
// 120 000 (K = 6 against 0), rate = 1 + ceil(1 * 1 / 5) ... the best recipe that needs the
// mill, grind, lasts 1 tick, so rate = 1 + 1 = 2, T_p = min(600 / 2, 180) = 180, Benefit =
// 120 000 * 180 / 10 000 = 2 160; Inputs = 30 * 8 + 10 * 12 = 360, Opp = 80 000 * 5 / 10 000 =
// 40, Cost = 400; Outlay = 360 + 30 * (5 + 1 + 2) = 600.
//
// Requirement: FR-SAV-003, FR-SAV-009, FR-SAV-010, FR-SAV-011, FR-SAV-012, FR-SAV-013, FR-SAV-019
func TestTarget_Example16_1(t *testing.T) {
	t.Parallel()
	v := millView(100)
	got := SavingTarget(v, 0)
	want := ProjectChoice{Recipe: rBuild, Class: 0, Executions: 1, Benefit: 2_160, Cost: 400, Outlay: 600}
	if !got.Chosen || got.Project != want {
		t.Errorf("SavingTarget(example A).Project = %+v (chosen %v), want %+v", got.Project, got.Chosen, want)
	}
	if got.Floor != 90 || got.Goal != 600 || got.Target != 690 || got.Bar != "" || got.Building {
		t.Errorf("SavingTarget(example A) floor, goal, target = %d, %d, %d (bar %q), want 90, 600, 690", got.Floor, got.Goal, got.Target, got.Bar)
	}
	if pc, ok := ChooseProject(v); !ok || pc != want {
		t.Errorf("ChooseProject(example A) = %+v, %v, want %+v, true", pc, ok, want)
	}
}

// TestTarget_NoBlockIsReserve verifies that a view with no Saving group has the flat reserve
// as its target and no project, and that a role with a block and no project recipe has the
// floor alone with no bar.
//
// Requirement: FR-SAV-003, FR-SAV-004, FR-SAV-024
func TestTarget_NoBlockIsReserve(t *testing.T) {
	t.Parallel()
	v := millView(100)
	v.Saving = nil
	if got := SavingTarget(v, 0); got.Target != 90 || got.Floor != 90 || got.Goal != 0 || got.Chosen {
		t.Errorf("SavingTarget(no group) = %+v, want the reserve 90 and no project", got)
	}
	none := millView(100)
	none.Recipes = none.Recipes[:2] // carry and grind: no clause
	if got := SavingTarget(none, 0); got.Target != 90 || got.Bar != "" || got.Chosen {
		t.Errorf("SavingTarget(no project recipe) = %+v, want 90 and no bar", got)
	}
}

// TestGoal_NetOfTier2FundingTick verifies the goal of the funding tick (16.5): with money 700
// the household is funded (700 >= 600 + 90), and in the tick in which the plan buys the
// timber and the stone as tier 2, 360, the goal is net of it so the inputs are counted once.
//
// Arithmetic: tier2 0: goal 600, S* 90 + 600 = 690; tier2 360: goal 600 - 360 = 240, S* 330.
//
// Requirement: FR-SAV-019, FR-SAV-020, FR-SAV-042
func TestGoal_NetOfTier2FundingTick(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier2, goal, target int64
	}{{0, 600, 690}, {360, 240, 330}, {600, 0, 90}, {9_000, 0, 90}}
	for _, tt := range tests {
		got := SavingTarget(millView(700), tt.tier2)
		if !got.Project.Funded || got.Goal != tt.goal || got.Target != tt.target {
			t.Errorf("SavingTarget(money 700, tier2 %d) funded, goal, target = %v, %d, %d, want true, %d, %d",
				tt.tier2, got.Project.Funded, got.Goal, got.Target, tt.goal, tt.target)
		}
	}
}

// TestOutlay_Example16_5 verifies the outlay with a tool (16.5): a build that requires a tool
// of 50 the household does not hold has an outlay of 360 + 50 + 30 * 8 = 650, and the
// household is funded at a money of 740 and not at 739.
//
// Requirement: FR-SAV-019
func TestOutlay_Example16_5(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		money  int64
		funded bool
	}{{739, false}, {740, true}} {
		v := millView(tt.money)
		v.Recipes[rBuild].Tools = []RecipeTool{{Commodity: mTool, WearPerUse: 0, Durability: 100}}
		pc, ok := ChooseProject(v)
		if !ok || pc.Outlay != 650 || pc.Funded != tt.funded {
			t.Errorf("ChooseProject(money %d, tool of 50) = outlay %d, funded %v (chosen %v), want 650, %v", tt.money, pc.Outlay, pc.Funded, ok, tt.funded)
		}
	}
}

// TestReach_Example16_5 verifies the reach test (16.5): with money 100 the household lacks
// 600 + 90 - 100 = 590; with a cumulative surplus of 420 over 30 records it is reachable
// (590 * 30 = 17 700 <= 420 * 180 = 75 600) and saves, with a surplus of 60 it is not (17 700 >
// 10 800), has no goal, a target of the floor and the bar Unreachable; a funded household
// lacks nothing and is reachable whatever its surplus.
//
// Requirement: FR-SAV-045
func TestReach_Example16_5(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		money   int64
		surplus int64
		goal    int64
		bar     SavingBar
	}{
		{"a surplus of 14 a tick reaches it", 100, 420, 600, ""},
		{"a surplus of 2 a tick does not", 100, 60, 0, BarUnreachable},
		{"a deficit does not", 100, -10, 0, BarUnreachable},
		{"a funded household needs nothing", 690, 0, 600, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := millView(tt.money)
			v.Saving.Surplus = tt.surplus
			got := SavingTarget(v, 0)
			if got.Goal != tt.goal || got.Bar != tt.bar || got.Target != 90+tt.goal {
				t.Errorf("SavingTarget(money %d, surplus %d) goal, bar, target = %d, %q, %d, want %d, %q, %d",
					tt.money, tt.surplus, got.Goal, got.Bar, got.Target, tt.goal, tt.bar, 90+tt.goal)
			}
		})
	}
	none := millView(100)
	none.Saving.Records = 0
	if got := SavingTarget(none, 0); got.Bar != BarUnreachable {
		t.Errorf("SavingTarget(no records, a need) bar = %q, want unreachable (with no ring only a need of 0 is reachable)", got.Bar)
	}
}

// TestKeep_Example16_5 verifies the keep of a job in progress (16.5): while the build has 3
// ticks remaining, no project is chosen and the goal is 30 * (3 + 1 + 2) = 180, so S* = 270.
//
// Requirement: FR-SAV-046
func TestKeep_Example16_5(t *testing.T) {
	t.Parallel()
	v := millView(100)
	v.HasJob, v.Recipe, v.Remaining, v.Executions, v.StartsIn = true, rBuild, 3, 1, 4
	got := SavingTarget(v, 0)
	if got.Chosen || !got.Building || got.Goal != 180 || got.Target != 270 || got.Bar != "" {
		t.Errorf("SavingTarget(job in progress) = %+v, want building, goal 180, target 270, no project", got)
	}
	v.Recipe = rCarry
	if got := SavingTarget(v, 0); got.Building || !got.Chosen {
		t.Errorf("SavingTarget(job of carry) building = %v, chosen = %v, want a job that is not a project's to leave the choice open", got.Building, got.Chosen)
	}
}

// TestBars_SurvivalOwnUseStanding verifies the bars of 5.5: hunger, money below the reserve
// and the own-use mode bar the household whatever its projects (a bar that comes first, also
// for the keep of a job in progress); a standing refusal of every project leaves it counted
// Standing with no goal; a project that does not pass is NoReturn. Each leaves the floor.
//
// Requirement: FR-SAV-018, FR-SAV-047
func TestBars_SurvivalOwnUseStanding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(v *AgentView)
		bar   SavingBar
	}{
		{"hunger", func(v *AgentView) { v.Hunger = 1 }, BarSurvival},
		{"money below the reserve", func(v *AgentView) { v.Money = 89 }, BarSurvival},
		{"own-use mode", func(v *AgentView) { v.OwnUse = true }, BarSurvival},
		{"survival also bars the keep", func(v *AgentView) {
			v.Hunger, v.HasJob, v.Recipe, v.Remaining = 1, true, rBuild, 3
		}, BarSurvival},
		{"every project refused for a standing reason", func(v *AgentView) { v.Recipes[rBuild].Standing = ReasonFacilityCap }, BarStanding},
		{"a project that does not pass", func(v *AgentView) { v.Saving.Block.HurdleBP = 60_000 }, BarNoReturn},
		{"a class with no row and no clause data", func(v *AgentView) { v.Saving.Classes = nil }, BarNoReturn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := millView(100)
			tt.setup(v)
			got := SavingTarget(v, 0)
			if got.Bar != tt.bar || got.Goal != 0 || got.Chosen || got.Building || got.Target != got.Floor {
				t.Errorf("SavingTarget() = %+v, want bar %q, no goal, no project and S* = floor", got, tt.bar)
			}
		})
	}
}

// TestStanding_NotWeighedWhateverTheBenefit verifies 5.5.1: a project with a Standing reason
// is not weighed, whatever its benefit, for each reason of CAP's order that buying inputs does
// not cure and for ORG's not_permitted; and that the reason of MaxExecutions (no_input here)
// does not mask it, because the rule reads Standing and not Reason.
//
// Requirement: FR-SAV-018, FR-SAV-047
func TestStanding_NotWeighedWhateverTheBenefit(t *testing.T) {
	t.Parallel()
	for _, r := range []Reason{ReasonFacilityCap, ReasonNoTarget, ReasonNotUnlocked, ReasonRentFunds, ReasonNoFacility, ReasonNotPermitted} {
		v := millView(100)
		v.Recipes[rBuild].Standing, v.Recipes[rBuild].Reason, v.Recipes[rBuild].MaxExecutions = r, ReasonNoInput, 0
		if got := SavingTarget(v, 0); got.Bar != BarStanding || got.Chosen || got.Goal != 0 {
			t.Errorf("SavingTarget(Standing %q, Reason no_input) = %+v, want standing, no project, no goal", r, got)
		}
	}
	v := millView(100)
	v.Recipes[rBuild].Reason, v.Recipes[rBuild].MaxExecutions = ReasonNoInput, 0
	if got := SavingTarget(v, 0); got.Bar != "" || !got.Chosen {
		t.Errorf("SavingTarget(Reason no_input, no Standing) = %+v, want the project chosen: buying the inputs cures a transient reason", got)
	}
}

// TestHurdle_AgentSpread verifies that the hurdle is a function of the agent ID (5.5.2): the
// build's ratio is 2 160 * 10 000 / 400 = 54 000 basis points; with HurdleBP 53 000 and a
// spread of 2 000 the hurdle is 53 000 + floor(Weyl32(agent) * 2 000 / 2^32): agent 1 has
// 54 236 and agent 3 has 54 708 and neither passes, agent 7 has 53 652 and passes.
//
// Requirement: FR-SAV-015
func TestHurdle_AgentSpread(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		agent  AgentID
		passes bool
	}{{1, false}, {3, false}, {7, true}} {
		v := millView(100)
		v.Agent, v.Saving.Block.HurdleBP, v.Saving.HurdleSpreadBP = tt.agent, 53_000, 2_000
		if _, ok := ChooseProject(v); ok != tt.passes {
			t.Errorf("ChooseProject(agent %d, hurdle 53 000 + spread 2 000) chosen = %v, want %v", tt.agent, ok, tt.passes)
		}
	}
}

// TestCommit_BandKeepsAHeldProject verifies the commit band (5.5.3): at a hurdle of 55 000
// the build (54 000) does not pass, but a household that already holds half its inputs by
// value (30 timber at 8 = 240, and 2 * 240 >= 360) continues at 55 000 * 8 000 / 10 000 =
// 44 000 with a band of 2 000, and one that holds less (100 < 180) does not.
//
// Requirement: FR-SAV-016
func TestCommit_BandKeepsAHeldProject(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		timber int64
		band   int64
		passes bool
	}{
		{"nothing held", 0, 2_000, false},
		{"less than half held", 12, 2_000, false},
		{"half held and a band", 30, 2_000, true},
		{"half held and no band", 30, 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := millView(100)
			v.Commodities[mTimber].Stock = tt.timber
			v.Saving.Block.HurdleBP, v.Saving.CommitBandBP = 55_000, tt.band
			if _, ok := ChooseProject(v); ok != tt.passes {
				t.Errorf("ChooseProject(timber %d, band %d) chosen = %v, want %v", tt.timber, tt.band, ok, tt.passes)
			}
		})
	}
}

// TestChoose_BestRatioTiesRepairThenID verifies the choice of 5.5.4: of two builds that pass,
// the one with the greater benefit over cost (the cheaper inputs), and between two identical
// builds the lower recipe ID.
//
// Arithmetic: the second build needs only 10 timber (cost 80 + 40 = 120 against 400), so its
// ratio is higher.
//
// Requirement: FR-SAV-017
func TestChoose_BestRatioTiesRepairThenID(t *testing.T) {
	t.Parallel()
	v := millView(100)
	cheap := RecipeView{Recipe: 7, Duration: 5, Effective: 10_000, Builds: BuildView{On: true, Class: 0, Toward: NoRegion},
		Inputs: []RecipeLine{{Commodity: mTimber, Quantity: 10}}}
	v.Recipes = append(v.Recipes, cheap)
	if pc, ok := ChooseProject(v); !ok || pc.Recipe != 7 {
		t.Errorf("ChooseProject(a cheaper build) = %+v, %v, want recipe 7", pc, ok)
	}
	twin := millView(100)
	dup := twin.Recipes[rBuild]
	dup.Recipe = 9
	twin.Recipes = append(twin.Recipes, dup)
	if pc, ok := ChooseProject(twin); !ok || pc.Recipe != rBuild {
		t.Errorf("ChooseProject(two identical builds) = %+v, %v, want the lower ID %d", pc, ok, rBuild)
	}
	slices.Reverse(twin.Recipes)
	if pc, ok := ChooseProject(twin); !ok || pc.Recipe != rBuild {
		t.Errorf("ChooseProject(two identical builds in the other order) = %+v, %v, want the lower ID %d", pc, ok, rBuild)
	}
}

// TestRepair_Example16_2 verifies the repair of 16.2: a mill at 120 of 600 (20%, below the gate
// of 50%) with cap 2 gives m_plan = ceil(480 / 100) = 5, restored 480, rate 2, T_rep =
// min(480 / 2, 180 * 1) = 180, dJ_1 = J(mill, 2) - J(mill, 0) = 200 000 - 80 000 = 120 000, a
// benefit of 120 000 * 180 / 10 000 = 2 160 against a cost of 5 * 4 * 8 = 160 plus an
// opportunity of 200 000 * 1 / 10 000 = 20, 180 (a ratio of 12); the build of the same class
// has dJ = J(mill, 8) - J(mill, 2) = 0 and does not pass.
//
// Requirement: FR-SAV-022, FR-SAV-017
func TestRepair_Example16_2(t *testing.T) {
	t.Parallel()
	v := millView(100)
	wornMill(v)
	pc, ok := ChooseProject(v)
	want := ProjectChoice{Recipe: rFix, Class: 0, Repair: true, Executions: 5, Benefit: 2_160, Cost: 180, Outlay: 160 + 30*(1+1+2)}
	if !ok || pc != want {
		t.Errorf("ChooseProject(worn mill) = %+v, %v, want %+v", pc, ok, want)
	}
}

// TestRepair_NeverAtZeroGate verifies that a repair gate of 0 never repairs, a mill above the
// gate is not repaired (condition 400 of 600 is 6 666 basis points) and a household that holds
// no facility of the class has nothing to repair: in each case no project is chosen, because
// the build of a class held at capacity 2 has no gain either.
//
// Requirement: FR-SAV-022
func TestRepair_NeverAtZeroGate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(v *AgentView)
	}{
		{"gate of 0", func(v *AgentView) { wornMill(v); v.Saving.Block.RepairBelowBP = 0 }},
		{"above the gate", func(v *AgentView) { wornMill(v); v.Saving.Facilities[0].Condition = 400 }},
		{"no facility", func(v *AgentView) { v.Recipes = slices.Delete(v.Recipes, int(rBuild), int(rBuild)+1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := millView(100)
			tt.setup(v)
			if pc, ok := ChooseProject(v); ok && pc.Repair {
				t.Errorf("ChooseProject() = %+v, want no repair", pc)
			}
		})
	}
}

// TestCohort_TotalsAndOneFacility verifies 3.3.4 and 5.1.3: the target is a total of the agent,
// weight included, and a build asks for one facility whatever the weight, so the inputs are one
// execution's and only the upkeep term scales.
//
// Arithmetic: weight 3 has an upkeep cost of 90 and a reserve of 270; the outlay is 360 +
// 90 * (5 + 1 + 2) = 1 080. The gain at weight 3 is J(mill, 6) - J(mill, 0) = 200 000 * 3 -
// 80 000 * 3 = 360 000, the rate 1 + ceil(1 * 3 / 1) = 4, T_p = min(600 / 4, 180) = 150, a
// benefit of 360 000 * 150 / 10 000 = 5 400 against a cost of 360 + 240 000 * 5 / 10 000 = 480.
//
// Requirement: FR-SAV-025
func TestCohort_TotalsAndOneFacility(t *testing.T) {
	t.Parallel()
	v := millView(300)
	v.Weight = 3
	got := SavingTarget(v, 0)
	if !got.Chosen || got.Floor != 270 || got.Project.Outlay != 1_080 || got.Project.Executions != 1 || got.Project.Benefit != 5_400 {
		t.Errorf("SavingTarget(weight 3) = %+v, want floor 270, outlay 1080, one execution, benefit 5400", got)
	}
}
