package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The helpers of the scenarios S119 to S125 (plan 3-cap Phase 10). A scenario drives
// external agents with the "directed" strategy: the default strategy builds and repairs
// nothing (spec-infrastructure.md 4.5.1), so a build or a repair is a directive.

// capOnePlain is pack one with overlay/infrastructure.toml and no initial row: the class,
// the build recipe and the repair recipe, and no facility until a scenario adds one.
func capOnePlain(t testing.TB, extra string) economysim.Config {
	t.Helper()
	cfg := loadPackBytes(t, []byte(extra), packOneInfrastructure...)
	cfg.Modules.Infrastructure.Initial = nil
	return cfg
}

// capRecipe, capRole, capClass and capCommodity resolve an identifier to its index.
func capRecipe(cfg economysim.Config, id string) economysim.RecipeID {
	return economysim.RecipeID(slices.IndexFunc(cfg.Recipes, func(r economysim.Recipe) bool { return r.ID == id }))
}

func capRole(cfg economysim.Config, id string) economysim.RoleID {
	return economysim.RoleID(slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == id }))
}

func capClass(cfg economysim.Config, id string) economysim.FacilityClassID {
	return economysim.FacilityClassID(slices.IndexFunc(cfg.Modules.Infrastructure.Classes, func(c economysim.FacilityClass) bool { return c.ID == id }))
}

func capCommodity(cfg economysim.Config, id string) economysim.CommodityID {
	return economysim.CommodityID(slices.IndexFunc(cfg.Commodities, func(c economysim.Commodity) bool { return c.ID == id }))
}

func capRegion(cfg economysim.Config, id string) economysim.RegionID {
	return economysim.RegionID(slices.IndexFunc(cfg.Regions, func(r economysim.Region) bool { return r.ID == id }))
}

// capStock is the goods of a directed cook that runs `prepare` for hundreds of ticks: the
// inputs, and tool_b in a quantity whose wear outlasts the run.
func capStock(cfg economysim.Config) []economysim.StockItem {
	return []economysim.StockItem{
		{Commodity: capCommodity(cfg, "staple"), Quantity: 20_000},
		{Commodity: capCommodity(cfg, "fuel"), Quantity: 20_000},
		{Commodity: capCommodity(cfg, "tool_b"), Quantity: 100},
	}
}

// capAddCook adds a directed cook with the stock of capStock to the region and returns it.
func capAddCook(t testing.TB, w *economysim.World, cfg economysim.Config, region economysim.RegionID) economysim.AgentID {
	t.Helper()
	id, err := w.AddAgent(economysim.NewAgent{Region: region, Role: capRole(cfg, "cook"), Strategy: "directed", Money: 10_000_000, Stock: capStock(cfg)})
	if err != nil {
		t.Fatalf("AddAgent(directed cook) error = %v, want nil", err)
	}
	return id
}

// capDirect asks the agent for k executions of the recipe at the next Step.
func capDirect(t testing.TB, w *economysim.World, id economysim.AgentID, recipe economysim.RecipeID, k int64) {
	t.Helper()
	if err := w.Direct(id, economysim.Directive{Job: &economysim.JobRequest{Recipe: recipe, Executions: k}}); err != nil {
		t.Fatalf("Direct(agent %d, recipe %d) error = %v, want nil", id, recipe, err)
	}
}

// capStep runs one audited Step.
func capStep(t testing.TB, w *economysim.World) {
	t.Helper()
	if err := w.Step(); err != nil {
		t.Fatalf("Step() at tick %d error = %v, want nil (WithAudit)", w.Tick(), err)
	}
}

// capAdd plants one facility of the class in the region and returns its ID.
func capAdd(t testing.TB, w *economysim.World, class economysim.FacilityClassID, region, toward economysim.RegionID, owner economysim.AgentID) economysim.FacilityID {
	t.Helper()
	ids, err := w.AddFacilities([]economysim.NewFacility{{Class: class, Region: region, Toward: toward, Owner: owner}})
	if err != nil || len(ids) != 1 {
		t.Fatalf("AddFacilities = %v, %v, want one ID and nil", ids, err)
	}
	return ids[0]
}

// capKinds counts the FacilityChanged events by "kind/cause".
func capKinds(events []economysim.Event) map[string]int {
	out := map[string]int{}
	for _, e := range events {
		if fc, ok := e.(economysim.FacilityChanged); ok {
			out[string(fc.Kind)+"/"+string(fc.Cause)]++
		}
	}
	return out
}

// capExecuted sums the executions of the recipe that agent ran (RecipeExecuted).
func capExecuted(events []economysim.Event, agent economysim.AgentID, recipe economysim.RecipeID) int64 {
	var n int64
	for _, e := range events {
		if x, ok := e.(economysim.RecipeExecuted); ok && x.Agent == agent && x.Recipe == recipe {
			n += x.Executions
		}
	}
	return n
}
