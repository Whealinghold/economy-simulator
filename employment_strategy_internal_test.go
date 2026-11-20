package economysim

import (
	"reflect"
	"testing"
)

// hireView is the fixture of the employer tests: workView (one household, 5 000 money,
// an upkeep cost of 900) as an employer. Its recipe 0 takes 1 ore (belief centre 50)
// and makes 4 item at a centre of 100, so the net of a hired execution at the hired
// yield of 10 000 is 4 * 100 - 50 = 350 and over its 2 ticks a position is bid at
// floor(350 / 2) = 175. It holds 10 ore, so its own request (1 execution, the weight)
// leaves 9 for hires, and MaxHires 4 with none open.
func hireView() *AgentView {
	v := workView()
	v.HasEmployer = true
	v.Commodities[cOre].Stock = 10
	v.Recipes[0] = RecipeView{
		Recipe: 0, Duration: 2, Effective: 10000, Hired: 10000, MaxExecutions: 1,
		Inputs:  []RecipeLine{{Commodity: cOre, Quantity: 1}},
		Outputs: []RecipeOutput{{Commodity: cItem, Quantity: 4, CostShare: 10000}},
	}
	v.Employer = EmployerView{Recipes: []RecipeID{0}, MaxHires: 4, WorkerShareBP: 5000, HireYieldBP: 10000, MaxOpen: 1000}
	return v
}

// TestDefaultOffer_Cases verifies the worker's offer of the default strategy: the
// households are ceil(w * s / 10 000), the reservation is the value per tick of the
// agent's own best recipe (floor of its net over its duration, 0 with none), and an
// agent that rests, is in the own-use mode or is not Hireable offers nothing.
//
// Setup: workView has a target of 15 000 with a rest band of 500, so T_hi = 15 750 and
// a ratio of 15 300 gives a share of (15 750 - 15 300) * 10 000 / 750 = 6 000; at a
// weight of 4 that asks ceil(4 * 6 000 / 10 000) = ceil(2.4) = 3 households. The own
// recipe nets 200 over 1 tick, so it reserves 200; over 3 ticks, floor(200 / 3) = 66.
//
// Requirement: FR-EMP-013, FR-EMP-014
func TestDefaultOffer_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(v *AgentView)
		want   LabourOffer
	}{
		{"a share of 10 000 offers the whole weight at the own value", func(v *AgentView) { v.Weight = 4 }, LabourOffer{Households: 4, Reserve: 200}},
		{"a part share rounds the households up", func(v *AgentView) { v.Weight, v.Ratio = 4, 15_300 }, LabourOffer{Households: 3, Reserve: 200}},
		{"the reservation is the value per tick, floored", func(v *AgentView) {
			v.Weight, v.Recipes[0].Duration = 4, 3
		}, LabourOffer{Households: 4, Reserve: 66}},
		{"a household with no recipe reserves 0", func(v *AgentView) { v.Weight, v.Recipes = 4, nil }, LabourOffer{Households: 4}},
		{"a ratio at the top of the band offers nothing", func(v *AgentView) { v.Weight, v.Ratio = 4, 16_000 }, LabourOffer{}},
		{"the own-use mode offers nothing", func(v *AgentView) { v.Weight, v.OwnUse = 4, true }, LabourOffer{}},
		{"a role that is not Hireable offers nothing", func(v *AgentView) { v.Weight, v.Hireable = 4, false }, LabourOffer{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := workView()
			v.Hireable, v.Money = true, 1_000_000 // the reserve of a weight of 4 is 10 800: the survival rule is off
			tt.mutate(v)
			if got := (DefaultStrategy{}).Work(v, nil).Offer; got != tt.want {
				t.Errorf("DefaultStrategy.Work(%+v).Offer = %+v, want %+v", v, got, tt.want)
			}
		})
	}
}

// TestDefaultHire_Cases verifies the employer's vacancies of the default strategy: the
// bid is the net value per tick of the best recipe of the hire list at the hired yield,
// and the positions are the least of the establishment left and of what the cash, the
// stock (the food kept out) and the tools allow after the agent's own request.
//
// Setup: hireView bids 175 on a recipe of 2 ticks, so a position costs 350. Money 5 000
// less the upkeep cost 900 buys floor(4 100 / 350) = 11 positions; the stock of 10 ore
// less the 1 execution of the own request leaves 9; MaxHires 4 cuts both. Each row
// moves one limit below the others.
//
// Requirement: FR-EMP-015, FR-EMP-016
func TestDefaultHire_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(v *AgentView)
		want   HireRequest
	}{
		{"the establishment limits the positions", func(*AgentView) {}, HireRequest{Recipe: 0, Positions: 4, Bid: 175}},
		{"hires already open use the establishment", func(v *AgentView) { v.Employer.Hired = 3 }, HireRequest{Recipe: 0, Positions: 1, Bid: 175}},
		{"the establishment full posts nothing", func(v *AgentView) { v.Employer.Hired = 4 }, HireRequest{}},
		{"stock less the own request limits the positions", func(v *AgentView) {
			v.Commodities[cOre].Stock = 3 // 3 ore - 1 execution of the own request = 2
		}, HireRequest{Recipe: 0, Positions: 2, Bid: 175}},
		{"cash less the upkeep limits the positions", func(v *AgentView) {
			v.Money = 1_600 // (1 600 - 900) / 350 = 2
		}, HireRequest{Recipe: 0, Positions: 2, Bid: 175}},
		{"cash that leaves no position posts nothing", func(v *AgentView) { v.Money = 900 }, HireRequest{}},
		{"a tool limits the positions", func(v *AgentView) {
			v.Recipes[0].Tools = []RecipeTool{{Commodity: cTool, Durability: 1_000_000}}
			v.Commodities[cTool].Stock = 2 // 2 tools - 1 = 1
		}, HireRequest{Recipe: 0, Positions: 1, Bid: 175}},
		{"the hired yield is the value of the bid", func(v *AgentView) {
			v.Recipes[0].Hired = 5_000 // 4 * 5 000 / 10 000 * 100 - 50 = 150, over 2 ticks = 75
		}, HireRequest{Recipe: 0, Positions: 4, Bid: 75}},
		{"a bid below 1 posts nothing", func(v *AgentView) { v.Recipes[0].Hired = 1_000 }, HireRequest{}},
		{"a recipe off the hire list posts nothing", func(v *AgentView) { v.Employer.Recipes = nil }, HireRequest{}},
		{"a full lane posts nothing", func(v *AgentView) { v.Employer.Open = 1000 }, HireRequest{}},
		{"a busy employer posts nothing", func(v *AgentView) { v.StartsIn = 2 }, HireRequest{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := hireView()
			tt.mutate(v)
			if got := (DefaultStrategy{}).Work(v, nil).Hire; got != tt.want {
				t.Errorf("DefaultStrategy.Work(hireView).Hire = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestDefaultHire_PositionsNotScaledByWorkShare verifies that an employer whose ratio is
// above its target rests its own labour and still posts its vacancies (spec-employment.md
// 4.3.4, 8.2.2): the establishment is not governed by the target-income rule.
//
// Requirement: FR-EMP-016
func TestDefaultHire_PositionsNotScaledByWorkShare(t *testing.T) {
	t.Parallel()
	v := hireView()
	v.Ratio = 16_000 // above T_hi = 15 750: the share is 0
	got := DefaultStrategy{}.Work(v, nil)
	if !got.NoJob {
		t.Errorf("DefaultStrategy.Work(ratio above the band) = %+v, want NoJob: its own labour rests", got)
	}
	// The own request is 0 here, so stock allows 10 and the establishment 4 positions.
	if want := (HireRequest{Recipe: 0, Positions: 4, Bid: 175}); got.Hire != want {
		t.Errorf("DefaultStrategy.Work(ratio above the band).Hire = %+v, want %+v", got.Hire, want)
	}
}

// TestDefaultByStock_KeepsTheFoodOut verifies the stock bound of a vacancy for an input
// that is also an upkeep commodity: the units the agent eats this tick stay out of it,
// as spec-agents.md 7.1.6 keeps them out of its own job.
//
// Setup: 10 staple held, 3 eaten (a weight of 1 and an upkeep of 3), the recipe takes
// 2 an execution: floor((10 - 3) / 2) = 3.
//
// Requirement: FR-EMP-015
func TestDefaultByStock_KeepsTheFoodOut(t *testing.T) {
	t.Parallel()
	v := hireView()
	r := &v.Recipes[0]
	r.Inputs = []RecipeLine{{Commodity: cStaple, Quantity: 2}}
	if got, want := v.byStock(r), int64(3); got != want {
		t.Errorf("byStock(2 staple an execution, 10 held, 3 eaten) = %d, want %d", got, want)
	}
}

// TestDefaultWork_ZeroOfferHireUnchanged verifies that a view with no employment group
// (the module off, the payroll form, a role with neither field) gets the same request
// as before the module: no offer and no vacancy, whatever the rest of the view says.
//
// Requirement: FR-EMP-009
func TestDefaultWork_ZeroOfferHireUnchanged(t *testing.T) {
	t.Parallel()
	for _, v := range []*AgentView{workView(), hireView()} {
		v.HasEmployer = false
		got := DefaultStrategy{}.Work(v, nil)
		if got.Offer != (LabourOffer{}) || got.Hire != (HireRequest{}) {
			t.Errorf("DefaultStrategy.Work(no employment group) = %+v, want no offer and no vacancy", got)
		}
	}
	d := Directed{Fallback: DefaultStrategy{}}
	v := workView()
	v.HasDirective = true
	v.Directive = Directive{Job: &JobRequest{NoJob: true, Offer: LabourOffer{Households: 2, Reserve: 7}}}
	if got := d.Work(v, nil).Offer; got != (LabourOffer{Households: 2, Reserve: 7}) {
		t.Errorf("Directed.Work(directive with an offer).Offer = %+v, want the directive's offer unchanged", got)
	}
}

// TestDefaultWork_NoStateNoDraw verifies that the default's offer and vacancy are a
// function of the view alone: two calls give the same request and neither takes a word
// (the random source passed is nil, which would panic if it were read).
//
// Requirement: FR-EMP-013, NFR-EMP-001
func TestDefaultWork_NoStateNoDraw(t *testing.T) {
	t.Parallel()
	v := hireView()
	v.Hireable = true
	first, second := DefaultStrategy{}.Work(v, nil), DefaultStrategy{}.Work(v, nil)
	if first != second || first.Hire.Positions == 0 || first.Offer.Households == 0 {
		t.Errorf("Work twice = %+v then %+v, want equal requests with an offer and a vacancy", first, second)
	}
}

// TestPlannedHire_TargetsOfTheNextTick verifies H* of spec-employment.md 9.2: the
// positions the default would post next tick, bounded by the establishment and the cash
// left after tier 1 and not by the stock it is about to buy, none for a busy employer,
// and that the input targets then add H* * q of the recipe to the planned job's.
//
// Setup: hireView with no ore held: cash buys 11, MaxHires cuts H* to 4. The own job is
// 1 execution (1 ore), so Job_ore = InputJobs * (1 * 1 + 4 * 1) = 5.
//
// Requirement: FR-EMP-015
func TestPlannedHire_TargetsOfTheNextTick(t *testing.T) {
	t.Parallel()
	v := hireView()
	v.Commodities[cOre].Stock = 0
	cost, _ := v.upkeepCost()
	if got := v.plannedHire(cost); got.n != 4 {
		t.Fatalf("plannedHire(no stock) = %+v, want H* = 4 (the stock is not a bound)", got)
	}
	tc := tradeCtx{v: v, par: householdParams(&v.Tunables)}
	tc.rstar = tc.plannedRecipe()
	tc.hire = v.plannedHire(cost)
	job, _, ok := tc.jobAndOutput(cOre)
	if !ok || job != 5 {
		t.Errorf("jobAndOutput(ore) = %d, %v, want 5, true (1 own + 4 hired executions of 1 ore)", job, ok)
	}
	if p := tc.purposeOf(cOre, job, false); p != PurposeInput {
		t.Errorf("purposeOf(ore) = %v, want %v", p, PurposeInput)
	}
	v.StartsIn = 2
	if got := v.plannedHire(cost); got.n != 0 {
		t.Errorf("plannedHire(busy employer) = %+v, want none (spec-employment.md 9.2.2)", got)
	}
	v.StartsIn, v.HasEmployer = 1, false
	tc.hire = v.plannedHire(cost)
	if job, _, _ := tc.jobAndOutput(cOre); job != 1 {
		t.Errorf("jobAndOutput(ore) with no employer group = %d, want 1: the own job alone", job)
	}
}

// TestPlannedHire_NoOwnRecipeStillTargetsTheHires verifies that an employer with no job
// of its own to plan still buys the inputs of its expected hires: the planned recipe is
// none and the target is H* * q.
//
// Requirement: FR-EMP-015
func TestPlannedHire_NoOwnRecipeStillTargetsTheHires(t *testing.T) {
	t.Parallel()
	v := hireView()
	v.Commodities[cOre].Stock = 0
	tc := tradeCtx{v: v, par: householdParams(&v.Tunables), rstar: -1}
	cost, _ := v.upkeepCost()
	tc.hire = v.plannedHire(cost)
	if job, out, ok := tc.jobAndOutput(cOre); !ok || job != 4 || out != 0 {
		t.Errorf("jobAndOutput(ore, no planned recipe) = %d, %d, %v, want 4, 0, true", job, out, ok)
	}
}

// TestView_NoHoldingsOfOthers verifies that the employer group of the view carries only
// the agent's own counts and the block's fields: no field of it is an agent, a hire or
// a slice other than the shared hire list of recipe IDs, so a view cannot show another
// agent's hires (spec-employment.md 9.1).
//
// Guard: the view is the information the agent has. A field naming a worker or a hire
// would leak another household's state to a strategy.
func TestView_NoHoldingsOfOthers(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(EmployerView{})
	for i := range typ.NumField() {
		f := typ.Field(i)
		switch {
		case f.Type.Kind() == reflect.Int64:
		case f.Type == reflect.TypeOf([]RecipeID(nil)):
		default:
			t.Errorf("EmployerView.%s has type %v, want int64 or []RecipeID", f.Name, f.Type)
		}
	}
}
