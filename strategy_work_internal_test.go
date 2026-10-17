package economysim

import (
	"slices"
	"testing"
)

// The commodities of the work fixture.
const (
	cStaple CommodityID = iota
	cItem
	cOre
	cTool
)

// cview is a CommodityView with a belief of the given centre and width and the
// widest bounds.
func cview(c CommodityID, stock, centre, width int64) CommodityView {
	return CommodityView{
		Commodity: c, Stock: stock, Belief: Belief{Centre: Price(centre), Width: width},
		Bounds: Bounds{Floor: 1, Ceiling: MaxPrice},
	}
}

// testTunables are the defaults of spec-agents.md 12 that the examples of
// section 16 use, with a belief width limit of 0 to 5 000 so that a width of 0
// gives a range of one price.
func testTunables() Tunables {
	return Tunables{
		CoverTicks: 3, InputJobs: 1, OutputCoverTicks: 1, FavourFloor: 2000, ReserveTicks: 3,
		SpendMaxShare: 5000, SpendHalfTicks: 10, RestBand: 500, ProduceHaltRatio: 30000,
		Beliefs: BeliefTunables{WidthMin: 0, WidthMax: 5000},
	}
}

// workView is the fixture of the Work tests: one household that eats 3 staple a
// tick (cost 3 * 300 = 900, so the reserve money is 3 * 900 = 2 700), holds 10
// staple and 5 000 money, and has one source recipe, gather, that makes 2 item
// at a belief centre of 100 (net 200). Its ratio is below its target, so it
// works.
func workView() *AgentView {
	return &AgentView{
		Weight: 1, Money: 5000, StartsIn: 1,
		HasRatio: true, Ratio: 10000, TargetRatio: 15000, Window: 10, WorkingTicks: 10,
		Upkeep: []UpkeepRow{{Commodity: cStaple, Quantity: 3}},
		Commodities: []CommodityView{
			cview(cStaple, 10, 300, 0), cview(cItem, 0, 100, 0), cview(cOre, 0, 50, 0), cview(cTool, 0, 100, 0),
		},
		Recipes: []RecipeView{{
			Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Outputs: []RecipeOutput{{Commodity: cItem, Quantity: 2, CostShare: 10000}},
		}},
		Tunables: testTunables(),
	}
}

// TestDefaultStrategyWork_Cases verifies the work decision of the default
// strategy for a table of states: no recipe, a source at a negative net, a
// recipe with inputs at a negative net, a tie, the production halt and a
// byproduct, the keep-the-food cap and a cap of 0, the survival rule, the floor,
// an absent ratio and a rest. Each row also checks that the work report the
// world reads gives the same share, rest, skipped recipes, reason and request
// as Work, since they are one function.
//
// Requirement: FR-AGT-010, FR-AGT-033, FR-AGT-034, FR-AGT-035, FR-AGT-036, FR-AGT-037, FR-AGT-070, FR-AGT-072
func TestDefaultStrategyWork_Cases(t *testing.T) {
	t.Parallel()
	gather := func(id RecipeID, q int64) RecipeView {
		return RecipeView{
			Recipe: id, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Outputs: []RecipeOutput{{Commodity: cItem, Quantity: q, CostShare: 10000}},
		}
	}
	tests := []struct {
		name         string
		mutate       func(v *AgentView)
		want         JobRequest
		share        int64
		floorChanged bool
		skipped      []RecipeID
		reason       Reason
	}{
		{"works at a ratio below its target", func(*AgentView) {}, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"no recipes requests none", func(v *AgentView) { v.Recipes = nil }, JobRequest{NoJob: true}, 10000, false, nil, ReasonNoRecipe},
		{"a source at a negative net is chosen", func(v *AgentView) {
			// Wear of the tool: ceil(10 * 100 / 1) = 1 000 against a value of 200.
			v.Recipes[0].Tools = []RecipeTool{{Commodity: cTool, WearPerUse: 10, Durability: 1}}
		}, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"a recipe with inputs at a negative net is not", func(v *AgentView) {
			// An input of 1 ore at 50 and an output of 2 item at 10: 20 - 50 < 0.
			v.Commodities[cItem].Belief.Centre = 10
			v.Recipes[0].Inputs = []RecipeLine{{Commodity: cOre, Quantity: 1}}
		}, JobRequest{NoJob: true}, 10000, false, nil, ReasonNoRecipe},
		{"equal value per tick goes to the lower ID", func(v *AgentView) {
			v.Recipes = []RecipeView{gather(0, 2), gather(1, 2)}
		}, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"a better recipe wins", func(v *AgentView) {
			v.Recipes = []RecipeView{gather(0, 2), gather(1, 3)}
		}, JobRequest{Recipe: 1, Executions: 1}, 10000, false, nil, ""},
		{"value per tick not per execution", func(v *AgentView) {
			// Recipe 0: 2 * 100 = 200 over 3 ticks, 66 a tick; recipe 1: 1 * 100 over 1 tick.
			r0 := gather(0, 2)
			r0.Duration = 3
			v.Recipes = []RecipeView{r0, gather(1, 1)}
		}, JobRequest{Recipe: 1, Executions: 1}, 10000, false, nil, ""},
		{"an overstocked output halts the recipe", func(v *AgentView) {
			// OutputTarget = ceil(1 * 2 * 10000 * 1 / 10000) = 2; halted at stock * 10000 >= 30000 * 2, stock >= 6.
			v.Commodities[cItem].Stock = 6
		}, JobRequest{NoJob: true}, 10000, false, []RecipeID{0}, ReasonHalted},
		{"just below the halt works", func(v *AgentView) { v.Commodities[cItem].Stock = 5 }, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"a byproduct never halts", func(v *AgentView) {
			v.Commodities[cItem].Stock = 1000
			v.Recipes[0].Outputs[0].CostShare = 0
		}, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"a halted recipe leaves the other", func(v *AgentView) {
			v.Commodities[cItem].Stock = 6
			other := gather(1, 1)
			other.Outputs[0].Commodity = cOre
			v.Recipes = append(v.Recipes, other)
		}, JobRequest{Recipe: 1, Executions: 1}, 10000, false, []RecipeID{0}, ""},
		{"the production reason is reported", func(v *AgentView) {
			v.Recipes[0].MaxExecutions, v.Recipes[0].Reason = 0, ReasonNoInput
		}, JobRequest{NoJob: true}, 10000, false, nil, ReasonNoInput},
		{"halted comes before a recipe that cannot run", func(v *AgentView) {
			v.Recipes[0].MaxExecutions, v.Recipes[0].Reason = 0, ReasonNoInput
			halted := gather(1, 2)
			halted.Outputs[0].Commodity = cOre
			v.Commodities[cOre].Stock = 6
			v.Recipes = append(v.Recipes, halted)
		}, JobRequest{NoJob: true}, 10000, false, []RecipeID{1}, ReasonHalted},
		{"the first recipe that cannot run gives the reason", func(v *AgentView) {
			v.Recipes[0].MaxExecutions, v.Recipes[0].Reason = 0, ReasonNoInput
			b := gather(1, 1)
			b.MaxExecutions, b.Reason = 0, ReasonNoTool
			v.Recipes = append(v.Recipes, b)
		}, JobRequest{NoJob: true}, 10000, false, nil, ReasonNoInput},
		{"the max executions cut the food cap", func(v *AgentView) {
			v.Weight, v.Money = 5, 100000
			v.Commodities[cStaple].Stock = 1000
			v.Recipes[0].MaxExecutions = 2
			v.Recipes[0].Inputs = []RecipeLine{{Commodity: cStaple, Quantity: 1}}
			v.Recipes[0].Outputs[0].Quantity = 10
		}, JobRequest{Recipe: 0, Executions: 2}, 10000, false, nil, ""},
		{"the smaller food cap of two inputs wins", func(v *AgentView) {
			v.Upkeep = append(v.Upkeep, UpkeepRow{Commodity: cOre, Quantity: 1})
			v.Weight, v.Money = 5, 100000
			v.Commodities[cStaple].Stock = 15 + 8 // 8 spare
			v.Commodities[cOre].Stock = 5 + 2     // 2 spare
			v.Recipes[0].MaxExecutions = 50
			v.Recipes[0].Inputs = []RecipeLine{{Commodity: cStaple, Quantity: 1}, {Commodity: cOre, Quantity: 1}}
			v.Recipes[0].Outputs[0].Quantity = 10
		}, JobRequest{Recipe: 0, Executions: 2}, 10000, false, nil, ""},
		{"keep the food shortens the request", func(v *AgentView) {
			// Weight 10 eats 30; 35 held less 30 = 5 left, one staple an execution.
			v.Weight, v.Money = 10, 100000
			v.Commodities[cStaple].Stock = 35
			v.Recipes[0].MaxExecutions = 10
			v.Recipes[0].Inputs = []RecipeLine{{Commodity: cStaple, Quantity: 1}}
			v.Recipes[0].Outputs[0].Quantity = 10
		}, JobRequest{Recipe: 0, Executions: 5}, 10000, false, nil, ""},
		{"a food cap of 0 never requests 0 executions", func(v *AgentView) {
			v.Commodities[cStaple].Stock = 3
			v.Recipes[0].Inputs = []RecipeLine{{Commodity: cStaple, Quantity: 1}}
			v.Recipes[0].Outputs[0].Quantity = 10
		}, JobRequest{NoJob: true}, 10000, false, nil, ReasonNoRecipe},
		{"a food cap of 0 tries the next best", func(v *AgentView) {
			v.Commodities[cStaple].Stock = 3
			v.Recipes[0].Inputs = []RecipeLine{{Commodity: cStaple, Quantity: 1}}
			v.Recipes[0].Outputs[0].Quantity = 10
			v.Recipes = append(v.Recipes, gather(1, 1))
		}, JobRequest{Recipe: 1, Executions: 1}, 10000, false, nil, ""},
		{"hunger works whatever the ratio says", func(v *AgentView) { v.Hunger, v.Ratio = 1, 99999 }, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"money below the reserve works whatever the ratio says", func(v *AgentView) { v.Money, v.Ratio = 2699, 99999 }, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"money at the reserve is not below it", func(v *AgentView) { v.Money, v.Ratio = 2700, 99999 }, JobRequest{NoJob: true}, 0, false, nil, ReasonRest},
		{"the floor works whatever the ratio says", func(v *AgentView) {
			// 2 working ticks of 10 is below 4 000 * 10 / 10 000 = 4.
			v.Ratio, v.WorkingTicks, v.MinWorkShare = 99999, 2, 4000
		}, JobRequest{Recipe: 0, Executions: 1}, 10000, true, nil, ""},
		{"the floor met does not act", func(v *AgentView) {
			v.Ratio, v.WorkingTicks, v.MinWorkShare = 99999, 4, 4000
		}, JobRequest{NoJob: true}, 0, false, nil, ReasonRest},
		{"the floor does not change a share already 10 000", func(v *AgentView) {
			v.WorkingTicks, v.MinWorkShare = 0, 4000
		}, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"an absent ratio works", func(v *AgentView) { v.HasRatio, v.Ratio = false, 99999 }, JobRequest{Recipe: 0, Executions: 1}, 10000, false, nil, ""},
		{"a share of 0 rests", func(v *AgentView) { v.Ratio = 15750 }, JobRequest{NoJob: true}, 0, false, nil, ReasonRest},
		{"rest comes before halted", func(v *AgentView) {
			v.Ratio = 15750
			v.Commodities[cItem].Stock = 6
		}, JobRequest{NoJob: true}, 0, false, nil, ReasonRest},
		{"a halt target above 1 still halts", func(v *AgentView) {
			// FR-AGT-070: every role's TargetRatio is above 1 (10 000) and the stock at the ratio halts.
			v.TargetRatio = 12000
			v.Commodities[cItem].Stock = 6
		}, JobRequest{NoJob: true}, 10000, false, []RecipeID{0}, ReasonHalted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := workView()
			tt.mutate(v)
			got := DefaultStrategy{}.Work(v, nil)
			if got != tt.want {
				t.Errorf("Work() = %+v, want %+v", got, tt.want)
			}
			var rep workReportBuf
			workReport(v, &rep)
			if rep.share != tt.share || rep.floorChanged != tt.floorChanged || rep.rest != (tt.share == 0) ||
				!slices.Equal(rep.skipped, tt.skipped) || rep.reason != tt.reason {
				t.Errorf("workReport() = share %d, floor %v, rest %v, skipped %v, reason %q, want %d, %v, %v, %v, %q",
					rep.share, rep.floorChanged, rep.rest, rep.skipped, rep.reason, tt.share, tt.floorChanged, tt.share == 0, tt.skipped, tt.reason)
			}
			if rep.halted != len(tt.skipped) {
				t.Errorf("workReport() halted = %d, want %d (one for each skipped recipe)", rep.halted, len(tt.skipped))
			}
			counted := workReportBuf{noList: true}
			workReport(v, &counted)
			if counted.halted != rep.halted || len(counted.skipped) != 0 || counted.reason != rep.reason || counted.chosen != rep.chosen {
				t.Errorf("workReport() with noList = halted %d, skipped %v, reason %q, chosen %d, want the count %d, no list, %q, %d",
					counted.halted, counted.skipped, counted.reason, counted.chosen, rep.halted, rep.reason, rep.chosen)
			}
			if chosen := rep.chosen >= 0; chosen == got.NoJob || chosen && (v.Recipes[rep.chosen].Recipe != got.Recipe || rep.executions != got.Executions) {
				t.Errorf("workReport() chose %d with %d executions, but Work() = %+v", rep.chosen, rep.executions, got)
			}
		})
	}
}

// TestDefaultStrategyWork_CohortShare verifies the example of spec-agents.md
// 16.4: a cohort of 40 with a target of 15 000 and a rest band of 500 at a ratio
// of 15 300 has a share of (15 750 - 15 300) * 10 000 / (15 750 - 15 000) = 6 000
// and requests ceil(40 * 6 000 / 10 000) = 24 executions; below the target all 40
// work and at the top of the band none does.
//
// Requirement: FR-AGT-034, FR-AGT-052
func TestDefaultStrategyWork_CohortShare(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		ratio int64
		want  JobRequest
	}{
		{"between the target and the top of the band", 15300, JobRequest{Recipe: 0, Executions: 24}},
		{"below the target", 14999, JobRequest{Recipe: 0, Executions: 40}},
		{"at the target", 15000, JobRequest{Recipe: 0, Executions: 40}},
		{"at the top of the band", 15750, JobRequest{NoJob: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := workView()
			v.Weight, v.Money, v.Ratio = 40, 1_000_000, tt.ratio
			v.Commodities[cStaple].Stock = 1000
			v.Recipes[0].MaxExecutions = 40
			if got := (DefaultStrategy{}).Work(v, nil); got != tt.want {
				t.Errorf("Work() at ratio %d = %+v, want %+v", tt.ratio, got, tt.want)
			}
		})
	}
}

// TestDefaultStrategyWork_CutNotRefused verifies that a request above what the
// recipe can run is cut to MaxExecutions and not refused, and that the request
// is never 0 executions for a recipe that can run.
//
// Requirement: FR-AGT-037
func TestDefaultStrategyWork_CutNotRefused(t *testing.T) {
	t.Parallel()
	v := workView()
	v.Weight, v.Money = 40, 1_000_000
	v.Commodities[cStaple].Stock = 1000
	v.Recipes[0].MaxExecutions = 7
	if got, want := (DefaultStrategy{}).Work(v, nil), (JobRequest{Recipe: 0, Executions: 7}); got != want {
		t.Errorf("Work() = %+v, want %+v", got, want)
	}
}

// TestDefaultStrategyWork_TakesNoWord verifies that Work draws nothing from the
// random source (spec-world.md 7.3, D3).
//
// Requirement: FR-AGT-020
func TestDefaultStrategyWork_TakesNoWord(t *testing.T) {
	t.Parallel()
	src := &wordSource{}
	(DefaultStrategy{}).Work(workView(), newRand(src))
	if src.n != 0 {
		t.Errorf("Work() took %d words, want 0", src.n)
	}
}

// TestWorkReport_BlockedNamesTheRecipeThatCannotRun verifies that a report that
// declines for a production reason names the first recipe of the view that
// cannot run, so that the world can count the household in the refusal tally
// under that recipe and reason, and that every other outcome leaves it at -1.
//
// Guard: the default strategy never requests a recipe the view says cannot run,
// so the refusal of a start never fires for it. Without blocked the households
// of a stalled chain (no input, no tool) would be counted by nothing, and the
// metric economysim.recipe.refused would stay at 0 for the default strategy.
//
// Requirement: FR-AGT-086
func TestWorkReport_BlockedNamesTheRecipeThatCannotRun(t *testing.T) {
	t.Parallel()
	gather := func(id RecipeID, q int64) RecipeView {
		return RecipeView{
			Recipe: id, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Outputs: []RecipeOutput{{Commodity: cItem, Quantity: q, CostShare: 10000}},
		}
	}
	cannot := func(r RecipeView, why Reason) RecipeView {
		r.MaxExecutions, r.Reason = 0, why
		return r
	}
	tests := []struct {
		name   string
		mutate func(v *AgentView)
		want   int
		reason Reason
	}{
		{"a recipe that works is not blocked", func(*AgentView) {}, -1, ""},
		{"the only recipe cannot run", func(v *AgentView) {
			v.Recipes[0] = cannot(v.Recipes[0], ReasonNoInput)
		}, 0, ReasonNoInput},
		{"the first of two that cannot run", func(v *AgentView) {
			v.Recipes = []RecipeView{cannot(gather(0, 1), ReasonNoTool), cannot(gather(1, 1), ReasonNoInput)}
		}, 0, ReasonNoTool},
		{"one that cannot run beside one that works", func(v *AgentView) {
			v.Recipes = []RecipeView{cannot(gather(0, 1), ReasonNoInput), gather(1, 1)}
		}, -1, ""},
		{"a halted recipe takes the household from the blocked count", func(v *AgentView) {
			v.Recipes = []RecipeView{cannot(gather(0, 1), ReasonNoInput), gather(1, 2)}
			v.Recipes[1].Outputs[0].Commodity = cOre
			v.Commodities[cOre].Stock = 6
		}, -1, ReasonHalted},
		{"a rest is not blocked", func(v *AgentView) {
			v.Ratio = 15750
			v.Recipes[0] = cannot(v.Recipes[0], ReasonNoInput)
		}, -1, ReasonRest},
		{"no eligible recipe is not blocked", func(v *AgentView) { v.Recipes = nil }, -1, ReasonNoRecipe},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := workView()
			tt.mutate(v)
			var rep workReportBuf
			workReport(v, &rep)
			if rep.blocked != tt.want {
				t.Errorf("workReport() blocked = %d, want %d (reason %q)", rep.blocked, tt.want, rep.reason)
			}
			if tt.want >= 0 && rep.reason != tt.reason {
				t.Errorf("workReport() reason = %q, want %q", rep.reason, tt.reason)
			}
			if tt.want < 0 && tt.reason != "" && rep.reason != tt.reason {
				t.Errorf("workReport() reason = %q, want %q", rep.reason, tt.reason)
			}
		})
	}
}
