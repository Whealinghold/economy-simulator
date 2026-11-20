package economysim

import (
	"math"
	"reflect"
	"testing"
)

// TestExports_NilView verifies that the two exported functions of the rules return the empty
// result for a nil view and do not panic.
//
// Guard: a consumer that calls them with a view it failed to build would otherwise crash the
// run through a nil dereference in upkeepCost.
func TestExports_NilView(t *testing.T) {
	t.Parallel()
	if got := SavingTarget(nil, 0); got != (SavingTargetResult{}) {
		t.Errorf("SavingTarget(nil, 0) = %+v, want the zero result", got)
	}
	if pc, ok := ChooseProject(nil); ok || pc != (ProjectChoice{}) {
		t.Errorf("ChooseProject(nil) = %+v, %v, want the zero choice and false", pc, ok)
	}
}

// TestPick_UpkeepOverflowChoosesNothing verifies that an upkeep cost that overflows int64 is
// read as the largest value by every reader of the rules, so a household with the largest
// possible balance, which the survival bar does not catch (its reserve is also the largest
// value), has an outlay that saturates and no project: the choice of the exports, the request
// of Work and the Debug line agree.
//
// Guard: the upkeep was read as 0 on overflow, which made the outlay of 360 and the project
// of example A (16.1) chosen and requested, while the default's Trade made no plan at all.
func TestPick_UpkeepOverflowChoosesNothing(t *testing.T) {
	t.Parallel()
	v := millView(math.MaxInt64)
	v.Upkeep = []UpkeepRow{{Commodity: mFood, Quantity: math.MaxInt64}}
	if _, ok := v.upkeepCost(); ok {
		t.Fatal("upkeepCost() ok = true, want an overflow for the setup")
	}
	if pc, ok := ChooseProject(v); ok {
		t.Errorf("ChooseProject(upkeep overflow) = %+v, true, want no project", pc)
	}
	if got := SavingTarget(v, 0); got.Chosen || got.Goal != 0 {
		t.Errorf("SavingTarget(upkeep overflow) = %+v, want no project and no goal", got)
	}
	if req, ok := savingRequest(v); ok {
		t.Errorf("savingRequest(upkeep overflow) = %+v, true, want the default's request", req)
	}
}

// TestPick_ToolWearOverflowIsNotFree verifies that a tool whose wear cost overflows makes the
// project cost saturate, so the project does not pass, where a wear read as 0 made the tool free.
//
// Guard: the ok of production.WearCost was discarded, so a tool with an enormous WearPerUse
// cost nothing to wear and the project passed the test of 5.5.1.
func TestPick_ToolWearOverflowIsNotFree(t *testing.T) {
	t.Parallel()
	v := millView(100)
	v.Recipes[rBuild].Tools = []RecipeTool{{Commodity: mTool, WearPerUse: math.MaxInt64 / 2, Durability: 1}}
	if pc, ok := ChooseProject(v); ok {
		t.Errorf("ChooseProject(tool wear overflow) = %+v, true, want no project", pc)
	}
}

// TestPick_OversizedRecipeIsNotWeighed verifies that a project recipe of a hand-built view with
// more lines than a validated recipe can have is not weighed and does not panic.
//
// Guard: the line arrays of the weighing hold MaxRecipeItems entries of each side, and a view
// built by hand is not validated; an out-of-range index would crash the caller.
func TestPick_OversizedRecipeIsNotWeighed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mut  func(r *RecipeView)
	}{
		{"inputs", func(r *RecipeView) { r.Inputs = make([]RecipeLine, MaxRecipeItems+1) }},
		{"tools", func(r *RecipeView) { r.Tools = make([]RecipeTool, MaxRecipeItems+1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := millView(100)
			tt.mut(&v.Recipes[rBuild])
			if pc, ok := ChooseProject(v); ok {
				t.Errorf("ChooseProject(%d %s) = %+v, true, want no project", MaxRecipeItems+1, tt.name, pc)
			}
		})
	}
}

// TestPick_RepeatedEvaluationIsStable verifies that the choice is a pure function of the view:
// the exports, the request of Work and the plan of Trade evaluate it three times in a tick, and
// repeating it neither changes the result nor writes to the view.
//
// Guard: the three evaluations per tick stand until the pick is cached (plan 3-sav, decision
// 4); a scratch written by one evaluation and read by the next would make them disagree.
func TestPick_RepeatedEvaluationIsStable(t *testing.T) {
	t.Parallel()
	for _, money := range []int64{100, 700} {
		v, fresh := millView(money), millView(money)
		first := SavingTarget(v, 0)
		for i := 0; i < 3; i++ {
			if got := SavingTarget(v, 0); got != first {
				t.Errorf("SavingTarget(money %d) evaluation %d = %+v, want %+v", money, i+2, got, first)
			}
		}
		if !reflect.DeepEqual(v, fresh) {
			t.Errorf("view after repeated evaluation (money %d) differs from a fresh one", money)
		}
	}
}

// TestPick_EqualProjectsChooseTheLowerRecipe verifies that two projects with the same figures
// are decided by the lower recipe ID whatever their order in the view, so the choice does not
// depend on the order of the slice.
//
// Guard: the tie-break of 5.5.4 is the last rule; without it the order of the view would pick.
func TestPick_EqualProjectsChooseTheLowerRecipe(t *testing.T) {
	t.Parallel()
	for _, swap := range []bool{false, true} {
		v := millView(100)
		twin := v.Recipes[rBuild]
		twin.Recipe = 9
		v.Recipes = append(v.Recipes, twin)
		if swap {
			n := len(v.Recipes)
			v.Recipes[rBuild], v.Recipes[n-1] = v.Recipes[n-1], v.Recipes[rBuild]
		}
		if pc, ok := ChooseProject(v); !ok || pc.Recipe != rBuild {
			t.Errorf("ChooseProject(twin builds, swapped %v) = %+v, %v, want recipe %d", swap, pc, ok, rBuild)
		}
	}
}
