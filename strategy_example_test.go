package economysim_test

import (
	"fmt"
	"math/rand/v2"

	economysim "github.com/Whealinghold/economy-sim"
)

// exampleWorkView is a household with ample money, stock of its food and one
// recipe it can run, built from plain data as a consumer's own strategy sees it.
func exampleWorkView() *economysim.AgentView {
	return &economysim.AgentView{
		Weight: 1, Money: 5_000, StartsIn: 1,
		HasRatio: true, Ratio: 10_000, TargetRatio: 15_000, Window: 10, WorkingTicks: 10,
		Upkeep: []economysim.UpkeepRow{{Commodity: 0, Quantity: 3}},
		Commodities: []economysim.CommodityView{
			{Commodity: 0, Stock: 10, Belief: economysim.Belief{Centre: 300}, Bounds: economysim.Bounds{Floor: 1, Ceiling: economysim.MaxPrice}},
			{Commodity: 1, Belief: economysim.Belief{Centre: 100}, Bounds: economysim.Bounds{Floor: 1, Ceiling: economysim.MaxPrice}},
		},
		Recipes: []economysim.RecipeView{{
			Recipe: 0, Duration: 1, Effective: 10_000, MaxExecutions: 1,
			Outputs: []economysim.RecipeOutput{{Commodity: 1, Quantity: 2, CostShare: 10_000}},
		}},
		Tunables: economysim.Tunables{
			CoverTicks: 3, InputJobs: 1, OutputCoverTicks: 1, FavourFloor: 2_000, ReserveTicks: 3,
			SpendMaxShare: 5_000, SpendHalfTicks: 10, RestBand: 500, ProduceHaltRatio: 30_000,
			Beliefs: economysim.BeliefTunables{WidthMax: 5_000},
		},
	}
}

// The default strategy works when the household's earned ratio is below its
// target and the recipe pays: here one execution of the only recipe (2 units
// at 100 each is worth more than it costs).
func ExampleDefaultStrategy_Work() {
	v := exampleWorkView()
	rng := rand.New(rand.NewPCG(1, 2))
	req := economysim.DefaultStrategy{}.Work(v, rng)
	fmt.Println("no job:", req.NoJob, "recipe:", req.Recipe, "executions:", req.Executions)

	v.Ratio = 20_000 // above the target: the household rests
	req = economysim.DefaultStrategy{}.Work(v, rng)
	fmt.Println("no job:", req.NoJob)
	// Output:
	// no job: false recipe: 0 executions: 1
	// no job: true
}

// Directed does what a consumer's directive says and leaves the rest to its
// fallback. The directive's job wins; without a directive the fallback decides,
// and with no fallback the household requests nothing.
func ExampleDirected_Work() {
	v := exampleWorkView()
	rng := rand.New(rand.NewPCG(1, 2))
	v.HasDirective = true
	v.Directive = economysim.Directive{Job: &economysim.JobRequest{Recipe: 3, Executions: 2}}

	d := economysim.Directed{Fallback: economysim.DefaultStrategy{}}
	req := d.Work(v, rng)
	fmt.Println("directive: recipe", req.Recipe, "executions", req.Executions)

	v.HasDirective = false
	fmt.Println("fallback no job:", d.Work(v, rng).NoJob)
	fmt.Println("no fallback no job:", economysim.Directed{}.Work(v, rng).NoJob)
	// Output:
	// directive: recipe 3 executions 2
	// fallback no job: false
	// no fallback no job: true
}
