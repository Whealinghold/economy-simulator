package economysim_test

import (
	economysim "github.com/Whealinghold/economy-sim"
)

// The world the subsistence scenarios run on (plan 2-10 Phase 8). The scenarios
// need a household that can be starved by an outage of its market recipe and
// rescued by an own-use recipe, so they use one small region rather than pack
// one: pack one is large and its S-scenarios take minutes (the cost matters, the
// F run has a limit of 120 minutes), and its households hold stock for hundreds
// of ticks before a shortage reaches them.

// forageSpec says how forageWorld is built.
type forageSpec struct {
	Households int64
	// StartGrain is the grain each household holds at the start; the upkeep is 2 a tick.
	StartGrain int64
	// OwnUseBP is the yield of the own-use recipe in basis points: Cov = OwnUseBP.
	OwnUseBP int64
	// Duration is the ticks of the own-use recipe.
	Duration int64
	// Module switches the subsistence module on; RetryJobs, EnterTicks and
	// ExitTicks are its tunables.
	Module                          bool
	RetryJobs, EnterTicks, ExitTick int64
	// Weight is the households of one agent (0 means 1).
	Weight int64
}

// forageWorld is one region "home" of foragers whose upkeep is 2 "grain" a tick.
// The market recipe "tend" makes 2 grain a tick at a yield of 10 000, so with
// nothing wrong a household feeds itself; a script that sets its yield to 0 is an
// outage. The own-use recipe "gather" (sited, 2 grain, spec.Duration ticks) is on
// the list of the module when it is on.
func forageWorld(spec forageSpec) economysim.Config {
	world := economysim.DefaultWorldParams()
	world.Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 2}}
	world.OutputCoverTicks = 0
	grain := economysim.DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	region := economysim.DefaultRegion()
	region.ID, region.ReferenceBasket = "home", 2_000
	role := economysim.DefaultRole()
	role.ID, role.Recipes, role.StartMoney = "forager", []string{"tend"}, 0
	if spec.StartGrain > 0 {
		role.StartStock = []economysim.Item{{Commodity: "grain", Quantity: economysim.Quantity(spec.StartGrain)}}
	}
	dur := max(spec.Duration, 1)
	cfg := economysim.Config{
		World: world, Calendar: economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{grain}, Roles: []economysim.Role{role}, Regions: []economysim.Region{region},
		Groups: []economysim.Group{{Region: "home", Role: "forager", Households: spec.Households}},
		Recipes: []economysim.Recipe{
			{ID: "gather", Duration: dur, Sited: true, TraineeYield: 5_000, Outputs: []economysim.Output{{Commodity: "grain", Quantity: 2 * economysim.Quantity(dur), CostShare: 10_000}}},
			{ID: "tend", Duration: 1, TraineeYield: 5_000, Outputs: []economysim.Output{{Commodity: "grain", Quantity: 2, CostShare: 10_000}}},
		},
		Yields: []economysim.YieldProfile{{Region: "home", Recipe: "gather", HasAll: true, All: spec.OwnUseBP}},
	}
	if spec.Module {
		m := economysim.DefaultSubsistence()
		m.Recipes, m.RetryJobs = []string{"gather"}, spec.RetryJobs
		if spec.EnterTicks > 0 {
			m.EnterTicks = spec.EnterTicks
		}
		if spec.ExitTick > 0 {
			m.ExitTicks = spec.ExitTick
		}
		cfg.Modules.Subsistence = &m
	}
	return cfg
}
