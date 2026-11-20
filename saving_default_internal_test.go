package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// spendOf is the money a plan's bids reserve: the sum of limit times quantity.
func spendOf(p *TradePlan) int64 {
	var s int64
	for _, in := range p.Intents {
		if in.Side == Bid {
			s += int64(in.Limit) * int64(in.Quantity)
		}
	}
	return s
}

// bidOf is the quantity the plan bids for the commodity, 0 when it has no bid for it.
func bidOf(p *TradePlan, c CommodityID) int64 {
	for _, in := range p.Intents {
		if in.Side == Bid && in.Commodity == c {
			return int64(in.Quantity)
		}
	}
	return 0
}

// TestWork_RequestsBuildOneExecution verifies the request of 5.7: a household with a project
// chosen whose recipe can run now (MaxExecutions at least 1) asks for one execution of a
// build, and for MaxExecutions of a repair; with MaxExecutions 0 (the inputs are not held) the
// default's request stands, and so it does for a household that survival bars.
//
// Requirement: FR-SAV-021, FR-SAV-022, FR-SAV-018
func TestWork_RequestsBuildOneExecution(t *testing.T) {
	t.Parallel()
	build := millView(100)
	build.Recipes[rBuild].MaxExecutions = 5
	repair := millView(100)
	wornMill(repair)
	repair.Recipes[rFix].MaxExecutions = 5
	cannot, barred := millView(100), millView(100)
	cannot.Recipes[rBuild].MaxExecutions, cannot.Recipes[rBuild].Reason = 0, ReasonNoInput
	barred.Recipes[rBuild].MaxExecutions, barred.Hunger = 5, 1
	tests := []struct {
		name string
		v    *AgentView
		want JobRequest
	}{
		{"a build asks for one execution", build, JobRequest{Recipe: rBuild, Executions: 1}},
		{"a repair asks for MaxExecutions", repair, JobRequest{Recipe: rFix, Executions: 5}},
		{"inputs not held: the default's request", cannot, JobRequest{Recipe: rCarry, Executions: 1}},
		{"survival binds: the default's request", barred, JobRequest{Recipe: rCarry, Executions: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := (DefaultStrategy{}).Work(tt.v, nil); got != tt.want {
				t.Errorf("Work() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestWork_IgnoresWorkShare verifies that the request does not depend on the work share
// (5.7): a household whose ratio is far above its target, which the default would rest, still
// asks for its build, because a build is the use of idle labour; and that the goal does not
// enter the survival rule or the work share, which read the flat reserve (3.2.3).
//
// Requirement: FR-SAV-021, FR-SAV-003
func TestWork_IgnoresWorkShare(t *testing.T) {
	t.Parallel()
	rich := func(v *AgentView) {
		v.HasRatio, v.Ratio, v.TargetRatio, v.Window, v.WorkingTicks = true, 30_000, 10_000, 30, 30
		v.Recipes[rBuild].MaxExecutions = 5
	}
	saver, plain := millView(100), millView(100)
	plain.Saving = nil
	rich(saver)
	rich(plain)
	if got := (DefaultStrategy{}).Work(plain, nil); !got.NoJob {
		t.Fatalf("Work(no group, ratio above target) = %+v, want NoJob: the default rests it", got)
	}
	if got := (DefaultStrategy{}).Work(saver, nil); got != (JobRequest{Recipe: rBuild, Executions: 1}) {
		t.Errorf("Work(saver, ratio above target) = %+v, want the build", got)
	}
	// Money 100 is above the reserve of 90 but far below S* = 690: the goal is not read by the
	// survival rule, so the work share is the ratio's and the saver without a project rests.
	saver.Recipes[rBuild].MaxExecutions = 0
	if got := (DefaultStrategy{}).Work(saver, nil); !got.NoJob {
		t.Errorf("Work(saver, money 100 below its goal, no start possible) = %+v, want NoJob: the goal is not in the survival rule", got)
	}
	par := householdParams(&saver.Tunables)
	if got := saver.reserveMoney(par); got != 90 {
		t.Errorf("reserveMoney(saver) = %d, want the flat reserve 90", got)
	}
}

// TestPlannedRecipe_FundedIsTheProject verifies 5.6.4: a funded household's planned recipe is
// the project's, with m executions of inputs and not InputJobs of them (InputJobs is 2 here,
// the project buys 30 timber and 10 stone once, as tier 2, and none of the default's grain).
//
// Requirement: FR-SAV-020
func TestPlannedRecipe_FundedIsTheProject(t *testing.T) {
	t.Parallel()
	funded := millView(700)
	funded.Tunables.InputJobs = 2
	p, _ := runTrade(funded, nil)
	if bidOf(&p, mTimber) != 30 || bidOf(&p, mStone) != 10 || bidOf(&p, mGrain) != 0 {
		t.Errorf("Trade(funded) bids timber %d, stone %d, grain %d, want 30, 10, 0", bidOf(&p, mTimber), bidOf(&p, mStone), bidOf(&p, mGrain))
	}
	for _, in := range p.Intents {
		if in.Commodity == mTimber && in.Purpose != PurposeInput {
			t.Errorf("Trade(funded) timber purpose = %v, want PurposeInput (tier 2, not subject to saving)", in.Purpose)
		}
	}
}

// TestPlannedRecipe_UnfundedIsTheDefaults verifies that an unfunded household's planned recipe
// is the default's (grind: grain, as far as its money goes) and it buys none of the project's
// inputs, and that the saving rule takes no random word: the same words are drawn and the same
// intents made as without the group (FR-SAV-002).
//
// Requirement: FR-SAV-020, FR-SAV-002
func TestPlannedRecipe_UnfundedIsTheDefaults(t *testing.T) {
	t.Parallel()
	unfunded, plain := millView(100), millView(100)
	unfunded.Tunables.InputJobs, plain.Tunables.InputJobs = 2, 2
	plain.Saving = nil
	pu, nu := runTrade(unfunded, nil)
	pp, np := runTrade(plain, nil)
	if g := bidOf(&pp, mGrain); bidOf(&pu, mTimber) != 0 || bidOf(&pu, mGrain) != g || g < 1 {
		t.Errorf("Trade(unfunded) bids timber %d, grain %d, want 0 and the %d of the default's planned recipe (grind)", bidOf(&pu, mTimber), bidOf(&pu, mGrain), g)
	}
	if nu != np || len(pu.Intents) != len(pp.Intents) {
		t.Errorf("Trade(unfunded) took %d words and %d intents, want %d and %d, as without the group", nu, len(pu.Intents), np, len(pp.Intents))
	}
}

// TestTarget_NeverRaisesASpendBound verifies FR-SAV-005 for the plan: with the group the bids
// never reserve more than the money the household has, and for a household that is not funded
// (so its planned recipe is the default's) the target only lowers the discretionary spending,
// by at most the cap itself, SpendMaxShare * SpendHalfTicks * UpkeepCost / 10 000 = 5 000 *
// 10 * 30 / 10 000 = 150.
//
// Requirement: FR-SAV-005
func TestTarget_NeverRaisesASpendBound(t *testing.T) {
	t.Parallel()
	for _, money := range []int64{90, 100, 300, 600, 689, 700, 1_500, 20_000} {
		saver, plain := millView(money), millView(money)
		plain.Saving = nil
		ps, _ := runTrade(saver, nil)
		pp, _ := runTrade(plain, nil)
		if spendOf(&ps) > money || spendOf(&pp) > money {
			t.Errorf("money %d: bids reserve %d with the group and %d without, want at most the money", money, spendOf(&ps), spendOf(&pp))
		}
		if money >= 690 {
			continue // funded: the plan buys the project's inputs, which is not a cut of discretion
		}
		if cut := spendOf(&pp) - spendOf(&ps); cut < 0 || cut > 150 {
			t.Errorf("money %d: spending with the group is %d against %d without, a cut of %d, want 0 to 150", money, spendOf(&ps), spendOf(&pp), cut)
		}
	}
}

// TestDefault_NoCreditForAProject verifies FR-SAV-023: the default borrower asks for nothing
// to finance a project. A household funded for its project (money 700, SafetyTicks 2 against
// SlackTicks 2) finds no gap: the horizon is 5 + 1 + 2 ticks of 30 = 240 plus tier 2 of 360
// against 700, so there is none; and an unfunded saver's request is the request of the same
// household without the group.
//
// Requirement: FR-SAV-023
func TestDefault_NoCreditForAProject(t *testing.T) {
	t.Parallel()
	credit := &CreditView{Borrow: &BorrowTerms{SlackTicks: 2, MaxDebtTicks: 10}, MinPrincipal: 1, UpkeepCost: 30, CreditRoom: 1_000_000}
	for _, money := range []int64{700, 100} {
		saver, plain := millView(money), millView(money)
		plain.Saving = nil
		saver.Credit, plain.Credit = credit, credit
		ps, _ := runTrade(saver, nil)
		pp, _ := runTrade(plain, nil)
		switch {
		case money == 700 && len(ps.Borrow) != 0:
			t.Errorf("Trade(funded saver, money 700) Borrow = %+v, want none: a funded household finds no gap", ps.Borrow)
		case money == 100 && (len(ps.Borrow) != len(pp.Borrow) || (len(ps.Borrow) == 1 && ps.Borrow[0] != pp.Borrow[0])):
			t.Errorf("Trade(unfunded saver) Borrow = %+v, want the request of the household without the group %+v", ps.Borrow, pp.Borrow)
		}
	}
}

// TestExports_MatchDefault verifies FR-SAV-042: the exported pair returns what the default
// uses, in the funding tick too: the target the default's cap reads is SavingTarget(v, tier2)
// with the plan's own tier 2, and its chosen project is ChooseProject(v).
//
// Requirement: FR-SAV-042
func TestExports_MatchDefault(t *testing.T) {
	t.Parallel()
	for _, money := range []int64{100, 700} {
		v := millView(money)
		cost, _ := v.upkeepCost()
		reserve, _ := household.ReserveMoney(cost, v.Tunables.ReserveTicks)
		tc := &tradeCtx{v: v, par: householdParams(&v.Tunables), rstar: -1}
		tc.planProject(reserve, cost)
		for _, tier2 := range []int64{0, 360} {
			if got, want := tc.savingTarget(tier2), SavingTarget(v, tier2).Target; got != want {
				t.Errorf("money %d, tier2 %d: the default's target = %d, SavingTarget = %d, want equal", money, tier2, got, want)
			}
		}
		if pc, ok := ChooseProject(v); ok != tc.pk.chosen || pc != tc.pk.choice {
			t.Errorf("money %d: the default's project = %+v (%v), ChooseProject = %+v (%v), want equal", money, tc.pk.choice, tc.pk.chosen, pc, ok)
		}
	}
}

// TestStart_TransientRefusalKeepsInputs verifies FR-SAV-044 at the rule: a start that the world
// would refuse for a transient reason (no_input, no_tool) does not end the weighing of the
// project. The household funded for the project holds none of its inputs (MaxExecutions 0, the
// default's request), buys them as tier 2 in the plan, and the next tick, with the inputs on
// hand, asks for the build; the project is chosen in both ticks.
//
// Requirement: FR-SAV-044
func TestStart_TransientRefusalKeepsInputs(t *testing.T) {
	t.Parallel()
	v := millView(700)
	v.Recipes[rBuild].MaxExecutions, v.Recipes[rBuild].Reason = 0, ReasonNoInput
	if pc, ok := ChooseProject(v); !ok || !pc.Funded {
		t.Fatalf("ChooseProject(inputs not held) = %+v, %v, want the project, funded", pc, ok)
	}
	if got := (DefaultStrategy{}).Work(v, nil); got.Recipe == rBuild {
		t.Errorf("Work(inputs not held) = %+v, want the default's request", got)
	}
	v.Commodities[mTimber].Stock, v.Commodities[mStone].Stock = 30, 10
	v.Recipes[rBuild].MaxExecutions, v.Recipes[rBuild].Reason = 5, ""
	if pc, ok := ChooseProject(v); !ok || pc.Held != 360 || pc.Recipe != rBuild {
		t.Errorf("ChooseProject(inputs held) = %+v, %v, want the project with 360 held", pc, ok)
	}
	if got := (DefaultStrategy{}).Work(v, nil); got != (JobRequest{Recipe: rBuild, Executions: 1}) {
		t.Errorf("Work(inputs held) = %+v, want one execution of the build", got)
	}
}

// TestNoBlock_TargetPlanAndRequestAreTheDefaults verifies FR-SAV-024: a view with no group is
// planned and asked as before the module, though its recipes name clauses: clearing the
// clauses of the recipes changes neither the plan nor the request.
//
// Requirement: FR-SAV-024
func TestNoBlock_TargetPlanAndRequestAreTheDefaults(t *testing.T) {
	t.Parallel()
	with, without := millView(700), millView(700)
	with.Saving, without.Saving = nil, nil
	for i := range without.Recipes {
		without.Recipes[i].Builds, without.Recipes[i].Repairs, without.Recipes[i].Standing = BuildView{}, RepairView{}, ""
	}
	a, na := runTrade(with, nil)
	b, nb := runTrade(without, nil)
	if na != nb || spendOf(&a) != spendOf(&b) || len(a.Intents) != len(b.Intents) {
		t.Errorf("Trade(no group) differs with the clauses cleared: %d words, %d intents, %d spent against %d, %d, %d", na, len(a.Intents), spendOf(&a), nb, len(b.Intents), spendOf(&b))
	}
	if x, y := (DefaultStrategy{}).Work(with, nil), (DefaultStrategy{}).Work(without, nil); x != y {
		t.Errorf("Work(no group) = %+v with the clauses and %+v without, want equal", x, y)
	}
}
