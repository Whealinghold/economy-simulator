package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// commodityIndex returns the CommodityID of a commodity string ID of a
// canonical configuration, which is its index.
func commodityIndex(t *testing.T, cfg *economysim.Config, id string) int {
	t.Helper()
	for i := range cfg.Commodities {
		if cfg.Commodities[i].ID == id {
			return i
		}
	}
	t.Fatalf("commodity %q is not in the configuration", id)
	return -1
}

// TestStep_WarmJobsCompleteWithTheirConstructionBooks verifies over warm-one the
// halves of the warm start that need the tick (spec-viability.md 8.5.3, FR-VIA-036):
// the construction Transform took the inputs of the warm jobs to recipe_input, in
// the bucket, and the first completion of each warm job creates exactly the
// pending output that construction fixed, so the recipe_input goods of the books
// and the recipe_output goods of the completions pair; the audit holds at every
// tick.
//
// Arithmetic: for each warm job the inputs are executions * the recipe's input
// quantity (make_tool_a takes 2 refined per execution, make_tool_b 3, refine 5
// mineral and 2 fuel), summed per commodity over the jobs, and the pending output
// of a job is executions * the output quantity * the yield / 10 000.
//
// Requirement: FR-VIA-036, FR-WLD-004, FR-PRO-029
func TestStep_WarmJobsCompleteWithTheirConstructionBooks(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	rec := &recorder{}
	w, cfg := buildWarm(t, []string{"base.toml", "one.toml", "warm-one.toml"}, economysim.WithObserver(rec))
	rows := economysim.ConstructionRows(w)
	jobs := map[economysim.AgentID]economysim.AgentJobView{}
	wantInput := map[int]int64{}
	for _, r := range rows {
		j := economysim.AgentJob(w, r.ID)
		if !j.Has {
			continue
		}
		jobs[r.ID] = j
		for _, in := range cfg.Recipes[j.Recipe].Inputs {
			wantInput[commodityIndex(t, &cfg, in.Commodity)] += j.Executions * int64(in.Quantity)
		}
	}
	if len(jobs) == 0 {
		t.Fatalf("no agent of warm-one has a job at construction, want some")
	}
	bucket, _ := economysim.ConstructionFlows(w)
	gotInput := map[int]int64{}
	for _, f := range bucket {
		if f.Reason == ledger.RecipeInput {
			gotInput[f.Commodity] = f.Total
		}
	}
	if len(gotInput) != len(wantInput) {
		t.Errorf("recipe_input commodities in the bucket = %v, want %v", gotInput, wantInput)
	}
	for c, want := range wantInput {
		if gotInput[c] != want {
			t.Errorf("bucket recipe_input of commodity %d = %d, want %d (the inputs of the warm jobs)", c, gotInput[c], want)
		}
	}
	for tick := range 8 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", tick, err)
		}
		if err := w.Audit(); err != nil {
			t.Fatalf("Audit() after tick %d = %v, want nil", tick, err)
		}
	}
	completed := 0
	for _, e := range rec.events {
		x, ok := e.(economysim.RecipeExecuted)
		if !ok || jobs[x.Agent].Pending == nil {
			continue
		}
		j := jobs[x.Agent]
		if x.Recipe != j.Recipe || x.Executions != j.Executions || x.EffectiveYield != j.Yield {
			continue
		}
		var want []economysim.Amount
		for i, o := range cfg.Recipes[j.Recipe].Outputs {
			if j.Pending[i] > 0 {
				want = append(want, economysim.Amount{Commodity: economysim.CommodityID(commodityIndex(t, &cfg, o.Commodity)), Quantity: economysim.Quantity(j.Pending[i])})
			}
		}
		if !slices.Equal(x.Outputs, want) {
			t.Errorf("first completion of the warm job of agent %d: Outputs = %v, want the pending output %v", x.Agent, x.Outputs, want)
		}
		completed++
		delete(jobs, x.Agent)
	}
	if completed == 0 {
		t.Errorf("no warm job completed in 8 ticks, want at least one")
	}
}

// TestStep_FirstTickCostAnchorsReadTheStartingWage verifies over warm-one that at
// tick 0, when no household has a working tick in its record, the cost anchors
// of P1.2 read the starting wage of the warm start (FR-VIA-052): the anchor of a
// commodity that a local recipe makes with no input and with one tick of labour
// is the wage over the units it gives, and the least cost where two recipes make it.
//
// Arithmetic (spec-production.md 9.1) with the wage 2 512: cut_fuel gives 14 fuel
// at E = 10 000 (all = 10 000 in region north, and its yield is not shocked): cost
// 2 512 and ceil(2 512 * 10 000 / (14 * 10 000)) = ceil(179.43) = 180. The mineral,
// made by dig_mineral at E = 10 000 for 3 units, is ceil(2 512 / 3) = ceil(837.33)
// = 838. The seed is made by gather_seed, 2 units, at ceil(2 512 / 2) = 1 256, and by
// save_seed, 20 units from 20 staple at 96 and one tick: ceil((20 * 96 + 2 512) / 20)
// = ceil(221.6) = 222; the anchor is the least, 222.
//
// Requirement: FR-VIA-052, FR-PRO-037, FR-PRO-047, FR-WLD-058
func TestStep_FirstTickCostAnchorsReadTheStartingWage(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	w, cfg := buildWarm(t, []string{"base.toml", "one.toml", "warm-one.toml"})
	if got := economysim.SeedWages(w); len(got) != 1 || got[0] != 2_512 {
		t.Fatalf("starting wages = %v, want [2512]", got)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	for _, tt := range []struct {
		commodity string
		want      int64
	}{{"fuel", 180}, {"seed", 222}, {"mineral", 838}} {
		c := commodityIndex(t, &cfg, tt.commodity)
		if got := economysim.UnitCostOf(w, 0, economysim.CommodityID(c)); got != tt.want {
			t.Errorf("anchor of %s at tick 0 = %d, want %d", tt.commodity, got, tt.want)
		}
	}
}

// TestStep_RegionOfRecipeExecutedOverTwoRegions verifies over the pack two, with
// two regions, that the Region of every RecipeExecuted is the region of the
// agent's account, that the audit holds and that every agent of a region
// completes jobs there.
//
// Requirement: FR-OBS-018, FR-WLD-043
func TestStep_RegionOfRecipeExecutedOverTwoRegions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	rec := &recorder{}
	w, err := economysim.NewWorld(loadPack(t, "base.toml", "two.toml"), economysim.WithObserver(rec), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(two) error = %v, want nil", err)
	}
	region := map[economysim.AgentID]economysim.RegionID{}
	for _, r := range economysim.ConstructionRows(w) {
		region[r.ID] = r.Region
	}
	for range 5 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	seen := map[economysim.RegionID]int{}
	for _, e := range rec.events {
		if x, ok := e.(economysim.RecipeExecuted); ok {
			seen[x.Region]++
			if x.Region != region[x.Agent] {
				t.Errorf("RecipeExecuted of agent %d has Region %d, want %d, the region of its account", x.Agent, x.Region, region[x.Agent])
			}
		}
	}
	if len(seen) != 2 {
		t.Errorf("regions with a RecipeExecuted = %v, want both regions of the pack", seen)
	}
}
