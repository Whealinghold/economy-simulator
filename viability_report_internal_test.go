package economysim

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
)

// This file tests the jobs, the census and the report of SolveSteadyState on
// configuration literals, each case breaking one thing of viaToy
// (viability_internal_test.go): ten households, 4 grain a tick each, 1 eaten,
// grain at 100, a wage of 400 and a basket of 100.

// twoGoods turns viaToy into a chain of grain and water, both eaten 1 a tick,
// each made by its own recipe of the given duration and output; the recipes are
// not on any role's list yet.
func twoGoods(grainOut, waterOut Quantity, duration int64) Config {
	c := viaToy()
	c.Commodities = append(c.Commodities, viaCommodity("water", 100, nil))
	c.World.Upkeep = append(c.World.Upkeep, Item{Commodity: "water", Quantity: 1})
	c.Regions[0].ReferenceBasket = 200
	c.Recipes = []Recipe{
		viaRecipe("farm", nil, []Output{viaOut("grain", grainOut)}, func(r *Recipe) { r.Duration = duration }),
		viaRecipe("well", nil, []Output{viaOut("water", waterOut)}, func(r *Recipe) { r.Duration = duration }),
	}
	return c
}

// TestSolveSteadyState_JobsAreScaledDownWhenTheyOverfillAGroup verifies D121
// and the clamp: a group of one household that must run both recipes of a chain
// that 100 households need has its two shares clamped to 10 000 and then scaled
// down in proportion to sum to 10 000, each rounded down, and the census lists
// the group as short.
//
// Requirement: FR-VIA-028, FR-VIA-034
func TestSolveSteadyState_JobsAreScaledDownWhenTheyOverfillAGroup(t *testing.T) {
	t.Parallel()
	c := twoGoods(8, 8, 2)
	c.Roles[0].Recipes = []string{"farm", "well"}
	idle := DefaultRole()
	idle.ID = "idle"
	c.Roles = append(c.Roles, idle)
	c.Groups = []Group{{Region: "north", Role: "farmer", Households: 1}, {Region: "north", Role: "idle", Households: 99}}
	st := solveToy(t, c)
	// Each good: 100 households eat 100 a tick, one execution makes 8, so
	// x = 100 / 8 = 12.5 at lambda 1 and 25 executions are in progress (x * d);
	// over the 1 household that lists the recipe that is 25 = 250 000 basis
	// points, clamped to 10 000 for each of the two recipes. The sum 20 000 is
	// scaled down to 10 000 * 10 000 / 20 000 = 5 000 each.
	wantJobs := []JobSpec{{Recipe: "farm", InProgressBP: 5_000}, {Recipe: "well", InProgressBP: 5_000}}
	if got := st.Table.Groups[0].Jobs; !reflect.DeepEqual(got, wantJobs) {
		t.Errorf("Jobs of the farmer = %v, want %v", got, wantJobs)
	}
	if got := st.Table.Groups[1].Jobs; got != nil {
		t.Errorf("Jobs of the idle role = %v, want none: it lists no recipe", got)
	}
	// The farmer's need is the labour of both recipes, 2 * 12.5 * 2 = 50 + 50 = 100
	// households; the idle role's is 0, so all 100 households balance to the farmer.
	//
	// IncomeBP of the farmer: the seed wage 400 times its labour, 25 + 25 = 50
	// (executions in progress, x * d, of both recipes), over 1 household times the
	// upkeep of 200 (one grain and one water at 100): 400 * 50 * 10 000 / 200 =
	// 1 000 000, that is 100 times its upkeep. The idle role has no active recipe,
	// so no income figure.
	wage := st.Table.Wages[0].Wage
	want := []CensusRow{
		{Region: "north", Role: "farmer", Households: 1, Balanced: 100, Imbalance: -99, Short: true, IncomeBP: 1_000_000, HasIncome: true},
		{Region: "north", Role: "idle", Households: 99, Balanced: 0, Imbalance: 99},
	}
	if wage != 400 {
		t.Errorf("seed wage = %d, want 400", wage)
	}
	if !reflect.DeepEqual(st.Report.Census, want) {
		t.Errorf("Census = %+v, want %+v", st.Report.Census, want)
	}
}

// TestSolveSteadyState_JobShareIsClamped verifies that the share of a recipe
// that more executions are in progress for than the group has households is
// clamped to 10 000 basis points before the shares of the group are scaled down
// together, where the table would otherwise break the rule of 8.7.
//
// Requirement: FR-VIA-033
func TestSolveSteadyState_JobShareIsClamped(t *testing.T) {
	t.Parallel()
	c := twoGoods(8, 8, 2)
	c.Roles[0].Recipes = []string{"farm", "well"}
	driller := DefaultRole()
	driller.ID, driller.Recipes = "driller", []string{"well"}
	c.Roles = append(c.Roles, driller)
	c.Groups = []Group{{Region: "north", Role: "farmer", Households: 1}, {Region: "north", Role: "driller", Households: 99}}
	st := solveToy(t, c)
	// 100 households eat 100 of each; an execution makes 8, so x = 12.5 and
	// x * d = 25 executions are in progress for each recipe. The farm is listed by
	// the 1 farmer: 25 = 250 000 basis points, clamped to 10 000. The well is
	// listed by all 100 households: 25 / 100 = 2 500 for both groups. The farmer's
	// 10 000 + 2 500 = 12 500 is scaled down to 10 000: farm 10 000 * 10 000 /
	// 12 500 = 8 000 and well 2 500 * 10 000 / 12 500 = 2 000 (without the clamp
	// they would be 9 900 and 99).
	want := map[string][]JobSpec{
		"farmer":  {{Recipe: "farm", InProgressBP: 8_000}, {Recipe: "well", InProgressBP: 2_000}},
		"driller": {{Recipe: "well", InProgressBP: 2_500}},
	}
	for _, g := range st.Table.Groups {
		if !reflect.DeepEqual(g.Jobs, want[g.Role]) {
			t.Errorf("Jobs of %s = %v, want %v", g.Role, g.Jobs, want[g.Role])
		}
	}
}

// TestSolveSteadyState_SharedRecipe verifies D121: a recipe on the lists of two
// roles of a region has its executions in progress divided by the households of
// both groups, each group gets the same share, and the census reports the recipe
// as ambiguous.
//
// Requirement: FR-VIA-028
func TestSolveSteadyState_SharedRecipe(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Recipes[0].Duration, c.Recipes[0].Outputs[0].Quantity = 2, 8
	miller := DefaultRole()
	miller.ID, miller.Recipes = "miller", []string{"farm"}
	c.Roles = append(c.Roles, miller)
	c.Groups = []Group{{Region: "north", Role: "farmer", Households: 6}, {Region: "north", Role: "miller", Households: 4}}
	st := solveToy(t, c)
	// 10 households eat 10 a tick, an execution makes 8: x = 1.25, so x * d = 2.5
	// executions are in progress, over the 6 + 4 households that list the recipe:
	// 0.25 = 2 500 basis points for each group.
	want := []JobSpec{{Recipe: "farm", InProgressBP: 2_500}}
	for _, g := range st.Table.Groups {
		if !reflect.DeepEqual(g.Jobs, want) {
			t.Errorf("Jobs of %s = %v, want %v", g.Role, g.Jobs, want)
		}
	}
	// One row for the recipe, flagged: its need is d * x = 2.5 households, rounded
	// up to 3, against the 10 that list it.
	row := CensusRow{Region: "north", Recipe: "farm", Households: 10, Balanced: 3, Imbalance: 7, Ambiguous: true}
	if !slices.Contains(st.Report.Census, row) {
		t.Errorf("Census = %+v, want it to hold %+v", st.Report.Census, row)
	}
}

// TestSolveSteadyState_UnlistedRecipe verifies that an active recipe no group
// of the region lists gets no job and is listed in the census, and that the
// seed over the full unit cost reads only the recipes a role lists.
//
// Requirement: FR-VIA-027, FR-VIA-028
func TestSolveSteadyState_UnlistedRecipe(t *testing.T) {
	t.Parallel()
	c := viaToy()
	// wild makes 8 grain a tick on its own, twice the farm, and no role lists it.
	c.Recipes = append(c.Recipes, viaRecipe("wild", nil, []Output{viaOut("grain", 8)}, nil))
	st := solveToy(t, c)
	// x = 10 / 8 = 1.25 executions of wild at d = 1: a need of 1.25, rounded up to 2.
	row := CensusRow{Region: "north", Recipe: "wild", Balanced: 2, Imbalance: -2, Unlisted: true}
	if !slices.Contains(st.Report.Census, row) {
		t.Errorf("Census = %+v, want it to hold %+v", st.Report.Census, row)
	}
	if got := st.Table.Groups[0].Jobs; got != nil {
		t.Errorf("Jobs = %v, want none: the only active recipe is a one-tick recipe nobody lists", got)
	}
	// The wage is that of wild, 8 * 0.1 * 1 000 = 800; the farm the role lists would
	// cost 800 * 1 / 4 = 200 a unit, and the seed 100 is below 0.7 * 200.
	want := []CostRecord{{Region: "north", Commodity: "grain", Seed: 100, UnitCost: 200}}
	if !reflect.DeepEqual(st.Report.SeedBelowCostFloor, want) {
		t.Errorf("SeedBelowCostFloor = %+v, want %+v", st.Report.SeedBelowCostFloor, want)
	}
}

// TestSolveSteadyState_InactiveLongRecipe verifies that a long recipe the basis
// leaves unused (the `salvage` shape) changes neither the main recipe nor the
// longest duration of the money.
//
// Requirement: FR-VIA-032
func TestSolveSteadyState_InactiveLongRecipe(t *testing.T) {
	t.Parallel()
	c := viaToy()
	// salvage takes 50 ticks for 1 grain: far worse than the farm, never active.
	c.Recipes = append(c.Recipes, viaRecipe("salvage", nil, []Output{viaOut("grain", 1)}, func(r *Recipe) { r.Duration = 50 }))
	c.Roles[0].Recipes = []string{"farm", "salvage"}
	// U * (d_max + buffer) = 100 * (1 + 10) = 1 100; d_max 50 would give 6 000.
	if got := solveToy(t, c).Table.Groups[0].Money; got != 1_100 {
		t.Errorf("Money with an inactive 50-tick recipe = %d, want 1 100", got)
	}
}

// TestSolveSteadyState_Imbalance verifies the balanced census (5.2, D132): the
// households of the region are apportioned in proportion to need, and the
// imbalance is the configured households less the balanced ones; a balanced
// configuration has none.
//
// Requirement: FR-VIA-028
func TestSolveSteadyState_Imbalance(t *testing.T) {
	t.Parallel()
	build := func(farmers, drillers int64) Config {
		c := twoGoods(8, 4, 1)
		c.Roles[0].Recipes = []string{"farm"}
		dr := DefaultRole()
		dr.ID, dr.Recipes = "driller", []string{"well"}
		c.Roles = append(c.Roles, dr)
		c.Groups = []Group{{Region: "north", Role: "farmer", Households: farmers}, {Region: "north", Role: "driller", Households: drillers}}
		return c
	}
	tests := []struct {
		name               string
		farmers, drillers  int64
		wantFarm, wantWell int64
		wantFarmIncome     int64
		wantWellIncome     int64
	}{
		// Needs per tick: grain 30 / 8 = 3.75 households, water 30 / 4 = 7.5; of the
		// 30 households the quotas are 30 * 3.75 / 11.25 = 10 and 20.
		{"imbalanced", 20, 10, 10, 20, 4_997, 19_988},
		{"balanced", 10, 20, 10, 20, 9_994, 9_994},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st := solveToy(t, build(tt.farmers, tt.drillers))
			// Rows are sorted by role: driller, then farmer.
			// IncomeBP is wage * labour * 10 000 / (households * upkeep) with the wage
			// 533, the upkeep 67 + 133 = 200 and the labour of a household count at
			// lambda 1: the farm 3.75, the well 7.5 (the needs above). Balanced, the
			// farmers' 533 * 3.75 * 10 000 / (10 * 200) = 9 993.75 and the drillers'
			// 533 * 7.5 * 10 000 / (20 * 200) = 9 993.75 both round to 9 994; with
			// 20 farmers 19 987.5 / 2 = 4 996.875 rounds to 4 997, and with 10
			// drillers 39 975 000 / 2 000 = 19 987.5 rounds half to even to 19 988.
			want := []CensusRow{
				{Region: "north", Role: "driller", Households: tt.drillers, Balanced: tt.wantWell, Imbalance: tt.drillers - tt.wantWell, IncomeBP: tt.wantWellIncome, HasIncome: true},
				{Region: "north", Role: "farmer", Households: tt.farmers, Balanced: tt.wantFarm, Imbalance: tt.farmers - tt.wantFarm, IncomeBP: tt.wantFarmIncome, HasIncome: true},
			}
			if !reflect.DeepEqual(st.Report.Census, want) {
				t.Errorf("Census(%d farmers, %d drillers) = %+v, want %+v", tt.farmers, tt.drillers, st.Report.Census, want)
			}
			// Prices 67 and 133 (a basket of 200 split 1 : 2 by the labour of an
			// execution), wage 533: the farm adds 8 * 67 = 536, 3 above 533, which is
			// 3 / 533 = 56 basis points; the well adds 4 * 133 = 532, 1 below, 18. The
			// largest deviation counts whichever way it goes.
			if st.Report.ResidualBP != 56 {
				t.Errorf("ResidualBP = %d, want 56", st.Report.ResidualBP)
			}
		})
	}
}

// TestSolveSteadyState_SeedAgainstCost verifies the seed reports on a chain
// with a joint output: grain carries 10% of the cost and chaff 90%, and chaff is
// a free good. Chaff is unpriced, floored, and below the cost band; grain is
// above the ceiling; the drift of chaff is reported; and the byproduct revenue is
// a residual above the threshold, which a higher threshold does not list.
//
// Requirement: FR-VIA-023, FR-VIA-025, FR-VIA-026 (dual-0 half; non-unique half N-311), FR-VIA-027, FR-VIA-054
func TestSolveSteadyState_SeedAgainstCost(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities = append(c.Commodities, viaCommodity("chaff", 80, nil))
	c.Recipes[0].Outputs = []Output{{Commodity: "grain", Quantity: 4, CostShare: 1_000}, {Commodity: "chaff", Quantity: 2, CostShare: 9_000}}
	st := solveToy(t, c)
	r := st.Report
	chaff := RegionCommodity{"north", "chaff"}
	// Nobody eats chaff, so its dual is 0: it is a free good, priced at the
	// resolution floor 50 (the band of ref 80 starts at 8), listed unpriced and floored.
	if !reflect.DeepEqual(r.UnpricedCommodities, []RegionCommodity{chaff}) || !reflect.DeepEqual(r.Floored, []RegionCommodity{chaff}) {
		t.Errorf("UnpricedCommodities, Floored = %v, %v, want [%v] for both", r.UnpricedCommodities, r.Floored, chaff)
	}
	// Unit cost of chaff = ceil(400 * 9 000 / (2 * 10 000)) = 180; 0.7 * 180 = 126
	// is above its seed 50. Grain: ceil(400 * 1 000 / (4 * 10 000)) = 10; 3 * 10 =
	// 30 is below its seed 100.
	if want := []CostRecord{{"north", "chaff", 50, 180}}; !reflect.DeepEqual(r.SeedBelowCostFloor, want) {
		t.Errorf("SeedBelowCostFloor = %+v, want %+v", r.SeedBelowCostFloor, want)
	}
	if want := []CostRecord{{"north", "grain", 100, 10}}; !reflect.DeepEqual(r.SeedAboveCostCeiling, want) {
		t.Errorf("SeedAboveCostCeiling = %+v, want %+v", r.SeedAboveCostCeiling, want)
	}
	// (50 - 80) * 10 000 / 80 = -3 750 for chaff; grain is at its reference price.
	if want := []DriftRow{{"north", "chaff", -3_750}, {"north", "grain", 0}}; !reflect.DeepEqual(r.Drift, want) {
		t.Errorf("Drift = %+v, want %+v", r.Drift, want)
	}
	// Value added of the farm = 4 * 100 + 2 * 50 = 500 against 400: 100 / 400 = 2 500.
	if want := []ResidualRow{{"north", "farm", 2_500}}; r.ResidualBP != 2_500 || !reflect.DeepEqual(r.ResidualOver, want) {
		t.Errorf("ResidualBP, ResidualOver = %d, %+v, want 2 500, %+v", r.ResidualBP, r.ResidualOver, want)
	}
	if got := solveToy(t, c, WithResidualWarnBP(2_500)).Report.ResidualOver; got != nil {
		t.Errorf("ResidualOver with a threshold of 2 500 = %+v, want none: a residual equal to it is not above it", got)
	}
}

// TestSolveSteadyState_DriftOfAReferencePrice verifies the drift when the
// configured reference prices are inconsistent with the technology: grain
// priced 80 with a basket of 85 is seeded at 85, 625 basis points above.
//
// Requirement: FR-VIA-024, FR-VIA-025
func TestSolveSteadyState_DriftOfAReferencePrice(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities[0].ReferencePrice, c.Regions[0].ReferenceBasket = 80, 85
	st := solveToy(t, c)
	// The scale makes the basket equal the configured 85: 10 * 85 / (10 * 0.1) = 850.
	if got := st.Table.Prices[0].Price; got != 85 {
		t.Errorf("seed = %d, want 85: the basket is the configured one", got)
	}
	if want := []DriftRow{{"north", "grain", 625}}; !reflect.DeepEqual(st.Report.Drift, want) {
		t.Errorf("Drift = %+v, want %+v ((85 - 80) * 10 000 / 80)", st.Report.Drift, want)
	}
}

// capacityToy is a chain whose farm is open in spring only and takes 100 ticks:
// one start a year, so the capacity row (10 * 1 / 365 executions a tick) binds
// and labour is slack. It makes 4 000 grain, so it is viable (lambda 10.96).
func capacityToy() Config {
	c := viaToy()
	c.Recipes[0].Duration, c.Recipes[0].Sited = 100, true
	c.Recipes[0].Outputs[0].Quantity = 4_000
	c.Yields = []YieldProfile{viaSpring("north", "farm")}
	return c
}

// TestSolveSteadyState_CapacityTight verifies the labour share and the nominal
// fallback on a chain whose capacity row is tight: the labour share is absent,
// the row is listed, the region is listed in NominalSeed, and because its nominal
// wage is also 0 it stays unpriced and has no wage entry.
//
// Requirement: FR-VIA-012, FR-VIA-050, FR-VIA-058
func TestSolveSteadyState_CapacityTight(t *testing.T) {
	t.Parallel()
	c := capacityToy()
	st := solveToy(t, c)
	// lambda = x * 4 000 / 10 with x = 10 / 365: 10.9589, SurplusBP 109 589.
	p := st.Programmes[0]
	if p.SurplusBP != 109_589 || p.HasLabourShare || p.LabourShareBP != 0 {
		t.Errorf("Programme = %+v, want SurplusBP 109 589 and no labour share", p)
	}
	r := st.Report
	if want := []RegionRecipe{{"north", "farm"}}; !reflect.DeepEqual(r.CapacityTight, want) {
		t.Errorf("CapacityTight = %+v, want %+v", r.CapacityTight, want)
	}
	if want := []string{"north"}; !slices.Equal(r.NominalSeed, want) || !slices.Equal(r.UnpricedWages, want) {
		t.Errorf("NominalSeed, UnpricedWages = %v, %v, want [north] for both: the nominal wage is 0 too", r.NominalSeed, r.UnpricedWages)
	}
	if st.Table.Wages != nil {
		t.Errorf("Wages = %v, want none: the wage is 0 (FR-VIA-050)", st.Table.Wages)
	}
	// The lean reserve: the first completion is at 59 + 100 - 1 = 158, so the 10
	// households use 10 a tick for 158 ticks = 1 580 grain, 158 each, plus the cover 5.
	if got := st.Table.Groups[0].Stock[0].Quantity; got != 163 {
		t.Errorf("grain stock = %d, want 163 = 158 + 5", got)
	}
	// The same chain with the farm open all year has a slack capacity row and a share.
	if p := solveToy(t, viaToy()).Programmes[0]; !p.HasLabourShare || p.LabourShareBP != 2_500 {
		t.Errorf("Programme of a slack chain = %+v, want a labour share of 2 500", p)
	}
}

// TestSolveSteadyState_DegenerateAndAlternative verifies that a second recipe
// identical to the first makes alternative optima and is reported.
//
// Requirement: FR-VIA-019, FR-VIA-022
func TestSolveSteadyState_DegenerateAndAlternative(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Recipes = append(c.Recipes, viaRecipe("farm2", nil, []Output{viaOut("grain", 4)}, nil))
	c.Roles[0].Recipes = []string{"farm", "farm2"}
	if st := solveToy(t, c); !st.Report.AlternativeOptima {
		t.Errorf("AlternativeOptima of two identical recipes = false, want true")
	}
	if st := solveToy(t, viaToy()); st.Report.AlternativeOptima || st.Report.Degenerate {
		t.Errorf("Degenerate, AlternativeOptima of the toy = %v, %v, want false, false", st.Report.Degenerate, st.Report.AlternativeOptima)
	}
}

// TestSolveSteadyState_WageBelowTarget verifies the role-economics report: the
// seed wage against TargetRatio * B_r * TicksPerYear / (10 000 * ExpectedWorkTicks),
// a report and never an error.
//
// Requirement: FR-WEL-031, FR-WEL-038
func TestSolveSteadyState_WageBelowTarget(t *testing.T) {
	t.Parallel()
	// Basket 100: 15 000 * 100 * 365 / (10 000 * 250) = 219 against a wage of 400.
	if got := solveToy(t, viaToy()).Report.WageBelowTarget; got != nil {
		t.Errorf("WageBelowTarget = %+v, want none: 400 is above 219", got)
	}
	c := viaToy()
	c.Roles[0].TargetRatio = 30_000 // 438 > 400
	want := []RoleWage{{Region: "north", Role: "farmer", Wage: 400, Target: 438}}
	if got := solveToy(t, c).Report.WageBelowTarget; !reflect.DeepEqual(got, want) {
		t.Errorf("WageBelowTarget = %+v, want %+v (30 000 * 100 * 365 / 2 500 000 = 438)", got, want)
	}
}

// TestSolveSteadyState_RegionsWithoutAProgramme verifies D130: a region with an
// empty basket is not a programme, has no table row and is listed in
// UnpricedWages, while the other region is solved; a region with no household is
// not a programme either; and with free trade the pooled programme covers both.
//
// Requirement: FR-VIA-018
func TestSolveSteadyState_RegionsWithoutAProgramme(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.World.Upkeep = nil
	c.Regions[0].ExtraUpkeep = []Item{{Commodity: "grain", Quantity: 1}} // the north eats; nobody else does
	south, empty := DefaultRegion(), DefaultRegion()
	south.ID, south.ReferenceBasket = "south", 0
	empty.ID, empty.ReferenceBasket = "empty", 0
	c.Regions = append(c.Regions, empty, south)
	idle := DefaultRole()
	idle.ID = "idle"
	c.Roles = append(c.Roles, idle)
	c.Groups = append(c.Groups, Group{Region: "south", Role: "idle", Households: 5})
	st := solveToy(t, c)
	if len(st.Programmes) != 1 || !slices.Equal(st.Programmes[0].Regions, []string{"north"}) {
		t.Errorf("Programmes = %+v, want one, of the north", st.Programmes)
	}
	if !slices.Equal(st.Report.UnpricedWages, []string{"south"}) {
		t.Errorf("UnpricedWages = %v, want [south]: the empty region has no household and is not listed", st.Report.UnpricedWages)
	}
	if len(st.Table.Groups) != 1 || st.Table.Groups[0].Region != "north" {
		t.Errorf("Groups = %+v, want the north's only", st.Table.Groups)
	}
	pooled := solveToy(t, c, WithFreeTrade())
	if len(pooled.Programmes) != 1 || !slices.Equal(pooled.Programmes[0].Regions, []string{"north", "south"}) || pooled.Report.UnpricedWages != nil {
		t.Errorf("pooled Programmes, UnpricedWages = %+v, %v, want one over north and south and none unpriced", pooled.Programmes, pooled.Report.UnpricedWages)
	}
}

// TestSolveSteadyState_UnsourcedRawBasket verifies JC5: a basket of unsourced
// Raw markers is no programme (its region is unpriced, each good listed), and
// ErrNoUpkeep is returned when no programme remains, where Validate passes.
//
// Requirement: FR-VIA-007, FR-VIA-018
func TestSolveSteadyState_UnsourcedRawBasket(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities[0].Raw = true
	c.Recipes, c.Roles[0].Recipes = nil, nil
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(raw-only basket) = %v, want nil", err)
	}
	if _, err := SolveSteadyState(context.Background(), c); !errors.Is(err, ErrNoUpkeep) {
		t.Errorf("SolveSteadyState(raw-only basket) = %v, want ErrNoUpkeep", err)
	}
	cc := c.canonical()
	s := &viaSolve{m: newViaModel(&cc)}
	if sets := s.sets(); sets != nil {
		t.Errorf("sets() = %v, want none", sets)
	}
	if !slices.Equal(s.rep.UnpricedWages, []string{"north"}) || !reflect.DeepEqual(s.rep.UnpricedCommodities, []RegionCommodity{{"north", "grain"}}) {
		t.Errorf("UnpricedWages, UnpricedCommodities = %v, %v, want [north] and [{north grain}]", s.rep.UnpricedWages, s.rep.UnpricedCommodities)
	}
}

// TestSolveSteadyState_RegionThatCannotFeedItself verifies that a region whose
// own activities cannot make its basket (lambda 0 alone, so no price vector) has
// no table in autarky and is listed unpriced, and that it has a table in the
// pooled programme, with no wage while its labour is slack.
//
// Requirement: FR-VIA-018, FR-VIA-050
func TestSolveSteadyState_RegionThatCannotFeedItself(t *testing.T) {
	t.Parallel()
	c := viaToy()
	// The farm is sited and has a yield row in the north only; the south's miners
	// dig ore nobody wants, and its grain would have to come from the north.
	c.Recipes[0].Sited = true
	c.Yields = []YieldProfile{{Region: "north", Recipe: "farm", HasAll: true, All: 10_000}}
	c.Commodities = append(c.Commodities, viaCommodity("ore", 50, nil))
	c.Recipes = append(c.Recipes, viaRecipe("dig", nil, []Output{viaOut("ore", 1)}, nil))
	miner := DefaultRole()
	miner.ID, miner.Recipes = "miner", []string{"dig"}
	c.Roles = append(c.Roles, miner)
	south := DefaultRegion()
	south.ID, south.ReferenceBasket = "south", 100
	c.Regions = append(c.Regions, south)
	c.Groups = append(c.Groups, Group{Region: "south", Role: "miner", Households: 10})
	st := solveToy(t, c)
	if len(st.Programmes) != 2 || st.Programmes[1].SurplusBP != 0 {
		t.Fatalf("Programmes = %+v, want two, the south's with SurplusBP 0", st.Programmes)
	}
	if !slices.Equal(st.Report.UnpricedWages, []string{"south"}) {
		t.Errorf("UnpricedWages = %v, want [south]", st.Report.UnpricedWages)
	}
	for _, g := range st.Table.Groups {
		if g.Region == "south" {
			t.Errorf("Groups holds the south's %+v, want none: it has no price", g)
		}
	}
	if want := []RegionSurplus{{Region: "north", AutarkyBP: 40_000}, {Region: "south", AutarkyBP: 0}}; !reflect.DeepEqual(st.Report.RegionSurplusBP, want) {
		t.Errorf("RegionSurplusBP = %+v, want %+v", st.Report.RegionSurplusBP, want)
	}
	// Pooled, the south has the realm's prices and a table; its miners' labour is
	// slack (they make nothing anyone eats), so its wage is 0 and it is unpriced
	// there, as the deep crater of spec-viability.md 15 is.
	pooled := solveToy(t, c, WithFreeTrade())
	if pooled.Report.NominalSeed != nil {
		t.Errorf("pooled NominalSeed = %v, want none: the south's labour is slack but no capacity row is tight", pooled.Report.NominalSeed)
	}
	if !slices.Equal(pooled.Report.UnpricedWages, []string{"south"}) || len(pooled.Table.Groups) != 2 {
		t.Errorf("pooled UnpricedWages, Groups = %v, %+v, want [south] unpriced and both groups", pooled.Report.UnpricedWages, pooled.Table.Groups)
	}
}
