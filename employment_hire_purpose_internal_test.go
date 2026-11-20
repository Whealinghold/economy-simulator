package economysim

import (
	"math"
	"testing"
)

// TestHirePurpose_InputsAndToolsOfThePlannedHire verifies the purpose of an order for a
// good the planned hires need (spec-employment.md 9.2.1): an input of the planned
// recipe with a job to buy for is PurposeInput, a tool is PurposeTool, any other good or a
// plan of no hires is not a hire purpose, and an employer with no job of its own to plan
// still classes the order by the hire. The tool target is one item for each position, H*.
//
// Setup: hireView with no ore held plans H* = 4 positions of a recipe that takes 1 ore
// (the input) and needs the tool, so the tool target is 4 and the input target is
// InputJobs * 4 * 1 = 4 (InputJobs is 1 in workView).
//
// Requirement: FR-EMP-015
func TestHirePurpose_InputsAndToolsOfThePlannedHire(t *testing.T) {
	t.Parallel()
	v := hireView()
	v.Commodities[cOre].Stock = 0
	v.Recipes[0].Tools = []RecipeTool{{Commodity: cTool, Durability: 1_000_000}}
	cost, _ := v.upkeepCost()
	tc := tradeCtx{v: v, par: householdParams(&v.Tunables), rstar: -1}
	tc.hire = v.plannedHire(cost)
	if tc.hire.n != 4 {
		t.Fatalf("plannedHire(no stock) = %+v, want H* = 4", tc.hire)
	}
	var tools sum
	tc.hireTools(&tools, cTool)
	if tools.v != 4 || tools.bad {
		t.Errorf("hireTools(tool) = %d (bad %v), want H* = 4", tools.v, tools.bad)
	}
	var other sum
	tc.hireTools(&other, cOre)
	if other.v != 0 {
		t.Errorf("hireTools(ore, not a tool) = %d, want 0", other.v)
	}
	tests := []struct {
		name string
		c    CommodityID
		job  int64
		hire plannedHire
		want Purpose
		ok   bool
	}{
		{"an input with a job to buy for", cOre, 4, tc.hire, PurposeInput, true},
		{"an input with no job is no hire purpose", cOre, 0, tc.hire, PurposeOther, false},
		{"a tool", cTool, 0, tc.hire, PurposeTool, true},
		{"a good the recipe does not use", cItem, 4, tc.hire, PurposeOther, false},
		{"no planned hire", cOre, 4, plannedHire{}, PurposeOther, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tc
			c.hire = tt.hire
			got, ok := c.hirePurpose(tt.c, tt.job)
			if got != tt.want || ok != tt.ok {
				t.Errorf("hirePurpose(%d, job %d) = %v, %v, want %v, %v", tt.c, tt.job, got, ok, tt.want, tt.ok)
			}
		})
	}
	// With no planned recipe of its own, purposeOf still reads the hire.
	if got := tc.purposeOf(cOre, 4, false); got != PurposeInput {
		t.Errorf("purposeOf(ore, no own recipe, H* = 4) = %v, want %v", got, PurposeInput)
	}
	if got := tc.purposeOf(cTool, 0, false); got != PurposeTool {
		t.Errorf("purposeOf(tool, no own recipe, H* = 4) = %v, want %v", got, PurposeTool)
	}
}

// TestHireBound_UntitledAndOverflowCases verifies the branches of check 8 that
// TestFormation_NoTitleRentFundsBounds does not reach (spec-employment.md 5.5,
// spec-land.md 4.7.2): a draw on an endowment with no title is no bound, a rent that
// does not fit in an int64 is rent_funds, and a block the principal's entitlement covers
// owes none whatever it has spent.
//
// Setup: the farm draws 3 ore an execution. The ore is let at 1 000 a unit and held by the
// hands, so the farm's least rent for one execution is 3 * 1 000 = 3 000, over its money of
// 1 000; for math.MaxInt64 / 2 executions the product does not fit. Held by the farm
// itself, the least rent is 0.
//
// Requirement: FR-EMP-023
func TestHireBound_UntitledAndOverflowCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		holder string // "" leaves the ore with no title
		m      int64
		spent  int64
		want   Reason
	}{
		{"no title on the endowment is no bound", "", 1, 0, ""},
		{"a rent the money cannot pay", "hand", 1, 0, ReasonRentFunds},
		{"a rent that does not fit", "hand", math.MaxInt64 / 2, 0, ReasonRentFunds},
		{"the entitlement covers the draw, whatever is spent", "farm", 1, 1_000_000, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			edit := func(c *Config) {
				c.Modules.Estates = &Estates{}
				c.Modules.Tenure = &TenureConfig{
					Terms: []TenureTerm{{Region: "r0", Resource: "ore", HasRent: true, Rent: 1_000, HasClosed: true}},
				}
				if tt.holder != "" {
					c.Modules.Tenure.Holdings = []Holding{{Region: "r0", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: tt.holder}, ShareBP: 10_000}}
				}
			}
			w := hireWorld(t, defaultHire(hireOpts{ore: 1_000, farmMoney: 1_000, edit: edit}), farmWants(1, 100, 0))
			stepN(t, w, 1)
			farm := w.agents[slotsOf(w, "farm")[0]].id
			if got := w.tenure.hireBound(farm, 0, 0, tt.m, tt.spent); got != tt.want {
				t.Errorf("hireBound(farm, r0, tend, m %d, spent %d) with the ore held by %q = %q, want %q", tt.m, tt.spent, tt.holder, got, tt.want)
			}
		})
	}
}
