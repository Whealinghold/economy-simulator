package economysim

import (
	"math"
	"testing"
)

// breadConfig is trainingConfig with a recipe "bake" that sorts before "knead" and
// is not trainee-eligible, on the weaver's list ahead of knead, so a loop over the
// trainer's recipes meets a recipe it must skip.
func breadConfig(stock []Item) Config {
	c := trainingConfig(3)
	c.Recipes = append(c.Recipes, Recipe{
		ID: "bake", Duration: 1, TraineeYield: 10_000,
		Inputs:  []Item{{Commodity: "grain", Quantity: 1}},
		Outputs: []Output{{Commodity: "flour", Quantity: 9, CostShare: 10_000}},
	})
	c.Roles[1].Recipes, c.Roles[1].StartStock = []string{"bake", "knead"}, stock
	return c
}

// TestSkillsWork_SkipsRecipesNotEligibleOrNotAffordable verifies that a trainee asks
// for the first trainee-eligible recipe of its trainer that the trainer's account
// can run: a recipe that is not eligible is skipped however profitable (bake makes
// 9 flour of 1 grain), and when the trainer holds none of the inputs the trainee
// asks for nothing and the step still counts as handled.
//
// Arithmetic: the trainer holds 10 grain (the start stock) and bake and knead each
// take 1, so with stock the trainee's ask is knead for one execution (weight 1);
// with no stock the count of both is 0 and no ask is recorded.
//
// Requirement: FR-SKL-014
func TestSkillsWork_SkipsRecipesNotEligibleOrNotAffordable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stock []Item
		want  bool
	}{
		{"trainer holds the input", []Item{{Commodity: "grain", Quantity: 10}}, true},
		{"trainer holds no input", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, breadConfig(tt.stock), WithSeed(exitSeed), WithAudit())
			c := castOf(t, w)
			trainee, trainer := c.homeMillers[0], c.homeWeaver
			w.skills.start(trainee, trainer, 1, 0, 0)
			w.pr.asks = make([]jobAsk, len(w.agents))
			if !w.skillsWork(&w.agents[trainee], 0) {
				t.Fatalf("skillsWork(trainee) = false, want true: the trainer's trainee is always handled")
			}
			ask := w.pr.asks[trainee]
			if ask.set != tt.want {
				t.Fatalf("ask of the trainee = %+v, want set = %v", ask, tt.want)
			}
			if tt.want && (ask.recipe != kneadOf(t, w) || ask.maximum != 1) {
				t.Errorf("ask of the trainee = %+v, want one execution of knead, not of the ineligible bake", ask)
			}
		})
	}
}

// TestEligibleOutput_ReportsOverflowAndSkipsDiscards verifies the value of a
// trainee's recipe at the region's prices: the discarded output has no value, a
// price that makes a product or a sum overflow reports not ok (so the gain test
// refuses the place rather than wrap), and the ordinary value is the net per tick.
//
// Arithmetic: knead is 1 grain into 3 flour at yield 10 000. At flour 3 000 and
// grain 1 000 it is 3 * 3 000 - 1 000 = 8 000. A flour price of MaxInt64 overflows
// the product with 3 units; two outputs of 1 unit at MaxInt64 overflow the sum; a
// grain price of MaxInt64 with 2 units overflows the input product and with two
// inputs of 1 unit the input sum.
//
// Requirement: FR-SKL-016
func TestEligibleOutput_ReportsOverflowAndSkipsDiscards(t *testing.T) {
	t.Parallel()
	const huge = math.MaxInt64
	tests := []struct {
		name   string
		edit   func(row *recipeRow, prices []int64)
		want   int64
		wantOK bool
	}{
		{"the ordinary value", func(*recipeRow, []int64) {}, 8_000, true},
		{"a discarded output has no value", func(row *recipeRow, _ []int64) { row.outputs[0].discard = true }, -1_000, true},
		{"an output product overflows", func(_ *recipeRow, p []int64) { p[flour] = huge }, 0, false},
		{"an output sum overflows", func(row *recipeRow, p []int64) {
			p[flour], row.outputs[0].quantity = huge, 1
			row.outputs = append(row.outputs, row.outputs[0])
		}, 0, false},
		{"an input product overflows", func(row *recipeRow, p []int64) { p[grain], row.inputs[0].quantity = huge, 2 }, 0, false},
		{"an input sum overflows", func(row *recipeRow, p []int64) {
			p[grain] = huge
			row.inputs = append(row.inputs, row.inputs[0])
		}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := trainingWorld(t, 3)
			rid := kneadOf(t, w)
			row := w.recipes.row(rid)
			w.ex.prices[flour], w.ex.prices[grain] = 3_000, 1_000
			tt.edit(row, w.ex.prices)
			got, ok := w.eligibleOutput(rid, &w.cfg.Recipes[rid])
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("eligibleOutput(knead) = %d, %v, want %d, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestGainer_ReadsOnlyEligibleRecipes verifies that the output a trainer's gain test
// weighs is that of the trainee-eligible recipes alone: adding a more profitable
// recipe that is not eligible to the role changes nothing.
//
// Arithmetic: knead gives 3 * flour price - grain price per tick at yield 10 000
// (8 000 at the reference prices of the comment of trainingConfig); bake would give
// 9 * flour price - grain price but is not eligible.
//
// Requirement: FR-SKL-016
func TestGainer_ReadsOnlyEligibleRecipes(t *testing.T) {
	t.Parallel()
	stock := []Item{{Commodity: "grain", Quantity: 10}}
	with := newMill(t, breadConfig(stock), WithSeed(exitSeed))
	without := newMill(t, trainingConfig(3), WithSeed(exitSeed))
	step(t, with, 1)
	step(t, without, 1)
	a, b := with.newGainer(0, 1), without.newGainer(0, 1)
	a.load()
	b.load()
	if a.output != b.output || a.output <= 0 {
		t.Errorf("gain output with an ineligible recipe = %d, without = %d, want equal and above 0", a.output, b.output)
	}
}

// TestTeaching_LoadSumBeyondInt64CountsAsAFullTick verifies that a trainer whose
// trainees' summed load overflows int64 is taken to teach every tick, with the
// accumulator kept in [0, 9 999], instead of wrapping to a small or negative sum.
//
// Arithmetic: two trainees at a TeachingLoad of MaxInt64 sum beyond int64, which
// counts as one full tick (10 000): the accumulator 0 + 10 000 reaches the
// threshold, so the tick teaches, and 10 000 - 10 000 = 0 is left.
//
// Requirement: FR-SKL-015
func TestTeaching_LoadSumBeyondInt64CountsAsAFullTick(t *testing.T) {
	t.Parallel()
	w, c := trainingWorld(t, 3)
	w.cfg.Roles[1].TeachingLoad = math.MaxInt64
	w.skills.start(c.homeMillers[0], c.homeWeaver, 1, 0, 0)
	w.skills.start(c.homeMillers[1], c.homeWeaver, 1, 0, 0)
	for tick := 1; tick <= 3; tick++ {
		w.p22Teaching()
		tr := w.skills.trainer[c.homeWeaver]
		if !tr.teaching || tr.acc < 0 || tr.acc > 9_999 {
			t.Errorf("tick %d: teaching = %v, acc = %d, want teaching and acc in [0, 9 999]", tick, tr.teaching, tr.acc)
		}
	}
}
