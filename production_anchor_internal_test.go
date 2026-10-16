package economysim

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

// flourConfig is the flour recipe of spec-production.md 9.2 in four regions.
// Commodities sort to bran 0, flour 1, grain 2, millstone 3 (durability
// 2 000). The recipe mill_flour takes 3 grain, makes 2 flour (share 10 000)
// and a bran byproduct (share 0), lasts 1 tick and wears the millstone by 1.
// Regions sort to a_rich 0, b_poor 1 (profile all = 8 000), c_nowage 2 and
// d_shock 3.
func flourConfig() *Config {
	c := Config{
		Regions:     []Region{{ID: "a_rich"}, {ID: "b_poor"}, {ID: "c_nowage"}, {ID: "d_shock"}},
		Commodities: []Commodity{{ID: "bran"}, {ID: "flour"}, {ID: "grain"}, {ID: "millstone", Durability: 2_000}},
		Recipes: []Recipe{{ID: "mill_flour", Duration: 1,
			Inputs:   []Item{{Commodity: "grain", Quantity: 3}},
			Outputs:  []Output{{Commodity: "flour", Quantity: 2, CostShare: 10_000}, {Commodity: "bran", Quantity: 1}},
			Requires: []Tool{{Commodity: "millstone", WearPerUse: 1}}}},
		Yields: []YieldProfile{{Region: "b_poor", Recipe: "mill_flour", HasAll: true, All: 8_000}},
	}
	cc := c.canonical()
	return &cc
}

// anchorRig is the tables and scratch of a configuration for the anchor step.
type anchorRig struct {
	cfg *Config
	rt  recipeTable
	yt  yieldTable
	sh  yieldShocks
	s   *productionScratch
}

func newAnchorRig(cfg *Config) *anchorRig {
	r := &anchorRig{cfg: cfg, rt: newRecipeTable(cfg), yt: newYieldTable(cfg)}
	r.s = newProductionScratch(cfg, &r.rt)
	return r
}

func (r *anchorRig) run(prices, wages []int64, runnable [][]RecipeID) []int64 {
	r.s.anchors(prices, wages, runnable, &r.yt, &r.sh, -1)
	return slices.Clone(r.s.costs)
}

// repeat returns one price row n times.
func repeat(row []int64, n int) []int64 {
	var out []int64
	for range n {
		out = append(out, row...)
	}
	return out
}

// TestAnchors_FlourTwoRegions verifies the worked example of PRO 9.2 through
// the table: the same recipe, grain at 200 and the millstone at 20 000, gives
// 455 at E = 10 000, 569 in the poor region at E = 8 000, 0 where the wage is
// 0, and 910 where a shock of 5 000 halves the yield; the bran byproduct and
// the raw grain and millstone get no anchor; unitCost reads the same cells and
// 0 outside the table.
//
// Arithmetic: cost = 3*200 + ceil(1*20000/2000) + 1*300 = 600 + 10 + 300 =
// 910. a_rich: ceil(910*10000 / (2*10000)) = 455. b_poor: ceil(9100000 /
// 16000) = ceil(568.75) = 569. c_nowage: wage 0 gives 0. d_shock: E =
// 10000*5000/10000 = 5000, ceil(9100000 / 10000) = 910.
//
// Requirement: FR-PRO-035, FR-PRO-037, FR-PRO-049
func TestAnchors_FlourTwoRegions(t *testing.T) {
	r := newAnchorRig(flourConfig())
	if err := r.sh.set(3, 0, 5_000); err != nil {
		t.Fatalf("set shock: %v", err)
	}
	prices := repeat([]int64{0, 0, 200, 20_000}, 4)
	wages := []int64{300, 300, 0, 300}
	all := [][]RecipeID{{0}, {0}, {0}, {0}}
	got := r.run(prices, wages, all)
	want := []int64{
		0, 455, 0, 0,
		0, 569, 0, 0,
		0, 0, 0, 0,
		0, 910, 0, 0,
	}
	if !slices.Equal(got, want) {
		t.Errorf("anchors(grain 200, millstone 20000, wages %v) = %v, want %v", wages, got, want)
	}
	tests := []struct {
		name string
		g    RegionID
		c    CommodityID
		want int64
	}{
		{"rich flour", 0, 1, 455},
		{"poor flour", 1, 1, 569},
		{"shocked flour", 3, 1, 910},
		{"byproduct", 0, 0, 0},
		{"region beyond", 4, 1, 0},
		{"negative region", -1, 1, 0},
		{"commodity beyond", 0, 4, 0},
		{"negative commodity", 0, -1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.s.unitCost(tt.g, tt.c); got != tt.want {
				t.Errorf("unitCost(%d, %d) = %d, want %d", tt.g, tt.c, got, tt.want)
			}
		})
	}
}

// selectConfig has flour made by five recipes, to test which ones set the
// anchor. Commodities: flour 0, grain 1, salt 2. Recipes sort to a_slow 0,
// b_fast 1, c_nosalt 2, d_idle 3, e_sited 4. At grain 200 and wage 300 with
// duration 1: a_slow (3 grain, 2 flour) costs (600+300)/2 = 450; b_fast (1
// grain, 2 flour) costs (200+300)/2 = 250; c_nosalt needs salt, price 0, so it
// is unknown; d_idle (1 grain, 10 flour) costs 500/10 = 50; e_sited (1 grain,
// 20 flour) is Sited with no profile, so its factor is 0.
func selectConfig() *Config {
	out := func(q int64) []Output {
		return []Output{{Commodity: "flour", Quantity: Quantity(q), CostShare: 10_000}}
	}
	c := Config{
		Regions:     []Region{{ID: "r0"}, {ID: "r1"}, {ID: "r2"}},
		Commodities: []Commodity{{ID: "flour"}, {ID: "grain"}, {ID: "salt"}},
		Recipes: []Recipe{
			{ID: "a_slow", Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 3}}, Outputs: out(2)},
			{ID: "b_fast", Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 1}}, Outputs: out(2)},
			{ID: "c_nosalt", Duration: 1, Inputs: []Item{{Commodity: "salt", Quantity: 1}}, Outputs: out(2)},
			{ID: "d_idle", Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 1}}, Outputs: out(10)},
			{ID: "e_sited", Duration: 1, Sited: true, Inputs: []Item{{Commodity: "grain", Quantity: 1}}, Outputs: out(20)},
		},
	}
	cc := c.canonical()
	return &cc
}

// TestAnchors_WhichRecipeSetsIt verifies FR-PRO-037 one rule per case: the
// least non-zero cost wins; a recipe nobody of the region has on a role list
// (not in the runnable list), one with an unknown cost and one with a factor
// of 0 are ignored; duplicates in the list change nothing; no runnable recipe
// gives 0.
//
// Arithmetic (see selectConfig): runnable {a,b,c,e}: 450, 250, unknown and
// ignored, 0-factor and ignored, so min(450, 250) = 250. {a}: 450. {d} only:
// 50. {}: 0. {a,a,b,b}: 250.
//
// Requirement: FR-PRO-037
func TestAnchors_WhichRecipeSetsIt(t *testing.T) {
	r := newAnchorRig(selectConfig())
	prices := repeat([]int64{0, 200, 0}, 3)
	wages := []int64{300, 300, 300}
	tests := []struct {
		name     string
		runnable []RecipeID
		want     int64
	}{
		{"least of runnable", []RecipeID{0, 1, 2, 4}, 250},
		{"only the slow one", []RecipeID{0}, 450},
		{"cheaper recipes not on the list are ignored", []RecipeID{0, 2, 4}, 450},
		{"idle recipe when listed", []RecipeID{3}, 50},
		{"none runnable", nil, 0},
		{"unknown cost only", []RecipeID{2}, 0},
		{"zero factor only", []RecipeID{4}, 0},
		{"duplicates", []RecipeID{0, 0, 1, 1}, 250},
		{"out of range ignored", []RecipeID{-1, 9, 0}, 450},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.run(prices, wages, [][]RecipeID{tt.runnable, nil, nil})
			if got[0] != tt.want {
				t.Errorf("anchor(flour) with runnable %v = %d, want %d", tt.runnable, got[0], tt.want)
			}
			if got[3] != 0 || got[6] != 0 {
				t.Errorf("anchors of regions with an empty list = (%d, %d), want 0", got[3], got[6])
			}
		})
	}
}

// TestAnchors_Deterministic verifies architecture.md 7.5 for the table: two
// builds from the same inputs are equal, a shuffled runnable list (with a
// duplicate) gives the same table, and a call with other inputs in between
// leaves nothing behind.
//
// Requirement: FR-PRO-037, FR-PRO-048
func TestAnchors_Deterministic(t *testing.T) {
	r := newAnchorRig(selectConfig())
	prices := repeat([]int64{0, 200, 0}, 3)
	wages := []int64{300, 300, 0}
	base := [][]RecipeID{{0, 1, 2, 3, 4}, {0, 1, 3}, {0, 1}}
	want := r.run(prices, wages, base)
	if got := r.run(prices, wages, base); !slices.Equal(got, want) {
		t.Errorf("second build = %v, want %v", got, want)
	}
	rng := rand.New(rand.NewPCG(7, 11))
	for i := range 20 {
		shuffled := make([][]RecipeID, len(base))
		for g := range base {
			shuffled[g] = append(slices.Clone(base[g]), base[g][0])
			rng.Shuffle(len(shuffled[g]), func(a, b int) { shuffled[g][a], shuffled[g][b] = shuffled[g][b], shuffled[g][a] })
		}
		r.run(prices, []int64{0, 0, 0}, [][]RecipeID{nil, nil, nil})
		if got := r.run(prices, wages, shuffled); !slices.Equal(got, want) {
			t.Errorf("shuffle %d %v: table = %v, want %v", i, shuffled, got, want)
		}
	}
	// Derived: r0 flour 50 (d_idle), r1 50, r2 wage 0.
	if want[0] != 50 || want[3] != 50 || want[6] != 0 {
		t.Errorf("table flour cells = (%d, %d, %d), want (50, 50, 0)", want[0], want[3], want[6])
	}
}

// fiveConfig is the size of the first consumer's world: 5 regions, 30
// commodities and 45 recipes, each with two inputs, a tool and two outputs.
func fiveConfig() *Config {
	c := Config{}
	for i := range 5 {
		c.Regions = append(c.Regions, Region{ID: fmt.Sprintf("g%02d", i)})
	}
	id := func(i int) string { return fmt.Sprintf("c%02d", i%30) }
	for i := range 30 {
		c.Commodities = append(c.Commodities, Commodity{ID: id(i), Durability: 1_000})
	}
	for i := range 45 {
		c.Recipes = append(c.Recipes, Recipe{ID: fmt.Sprintf("r%02d", i), Duration: 1 + int64(i%3),
			Inputs:   []Item{{Commodity: id(i), Quantity: 2}, {Commodity: id(i + 7), Quantity: 1}},
			Outputs:  []Output{{Commodity: id(i + 1), Quantity: 2, CostShare: 10_000}, {Commodity: id(i + 2), Quantity: 1}},
			Requires: []Tool{{Commodity: id(i + 3), WearPerUse: 1}}})
	}
	cc := c.canonical()
	return &cc
}

func fiveInputs(cfg *Config) (prices, wages []int64, runnable [][]RecipeID) {
	for range len(cfg.Regions) {
		for range cfg.Commodities {
			prices = append(prices, 100)
		}
		wages = append(wages, 300)
		var all []RecipeID
		for i := range cfg.Recipes {
			all = append(all, RecipeID(i))
		}
		runnable = append(runnable, all)
	}
	return prices, wages, runnable
}

// TestAnchors_AllocatesNothing verifies NFR-PRO-001 for the table at the size
// of the first consumer's world: a call allocates nothing, and the table is
// not all zero (the step did work).
//
// Requirement: NFR-PRO-001, FR-PRO-037
func TestAnchors_AllocatesNothing(t *testing.T) {
	// not parallel: AllocsPerRun panics in a parallel test
	r := newAnchorRig(fiveConfig())
	prices, wages, runnable := fiveInputs(r.cfg)
	if allocs := testing.AllocsPerRun(50, func() { r.s.anchors(prices, wages, runnable, &r.yt, &r.sh, -1) }); allocs != 0 {
		t.Errorf("AllocsPerRun(anchors) = %v, want 0", allocs)
	}
	if !slices.ContainsFunc(r.s.costs, func(v int64) bool { return v > 0 }) {
		t.Errorf("table = %v, want some non-zero anchor", r.s.costs)
	}
}

// BenchmarkAnchors measures the whole table at the size `five` (5 regions, 45
// recipes, 30 commodities).
//
// Requirement: NFR-PRO-003
func BenchmarkAnchors(b *testing.B) {
	r := newAnchorRig(fiveConfig())
	prices, wages, runnable := fiveInputs(r.cfg)
	b.ReportAllocs()
	for b.Loop() {
		r.s.anchors(prices, wages, runnable, &r.yt, &r.sh, -1)
	}
}
