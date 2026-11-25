package economysim

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// savWorld is the world of bldConfig with the saving module on and a Saver block on the
// miller role (role 0): the hurdle and the horizon of example A of the spec, a repair gate of
// 5 000 and one Worth row for the barn. The weaver (role 1) has no block. The numbers are
// placeholders (N-015, N-027); mut changes the configuration before the world is built.
func savWorld(t testing.TB, mut func(c *Config), opts ...Option) *World {
	t.Helper()
	c := bldConfig(func(c *Config) {
		c.Roles[0].Strategy = "default" // the default strategy, which a saver role runs
		c.Modules.Saving = &Saving{SafetyTicks: 2, HurdleSpreadBP: 2_000, CommitBandBP: 2_000}
		c.Roles[0].Saver = &Saver{
			PrecautionShareBP: 10_000, PrecautionCapTicks: 30, HurdleBP: 15_000, PaybackTicks: 180, RepairBelowBP: 5_000,
			Worth: []ProjectWorth{{Class: "barn", PerTick: 15, Count: 4}},
		}
		if mut != nil {
			mut(c)
		}
	})
	return newMill(t, c, append([]Option{WithSeed(facSeed), WithAudit()}, opts...)...)
}

// savView builds the view of an agent for a call, as P2.5 and P3.2 do, and returns it. The
// group and the recipes point into the world's scratch, so the next call rewrites them.
func savView(t testing.TB, w *World, id AgentID, kind viewKind) AgentView {
	t.Helper()
	w.refreshViewCtx()
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[id], kind); err != nil {
		t.Fatalf("fillAgentView(agent %d, kind %d) with seed %d error = %v, want nil", id, kind, facSeed, err)
	}
	return v
}

// recipeOf is the view of the recipe of the configuration.
func recipeOf(t testing.TB, w *World, v *AgentView, id string) *RecipeView {
	t.Helper()
	want := bldRecipe(t, w, id)
	for i := range v.Recipes {
		if v.Recipes[i].Recipe == want {
			return &v.Recipes[i]
		}
	}
	t.Fatalf("recipe %s is not in the view of agent %d", id, v.Agent)
	return nil
}

// TestView_SavingInWorkAndTrade verifies that an agent whose role has a Saver block has the
// Saving group in the Work view and in the Trade view, with a copy of the block (the Worth
// rows included) and the module's three values.
//
// Requirement: FR-SAV-028
func TestView_SavingInWorkAndTrade(t *testing.T) {
	t.Parallel()
	w := savWorld(t, nil)
	m := bldMiller(w, 0)
	cfg := w.cfg.Roles[0].Saver
	for _, kind := range []viewKind{viewWork, viewTrade} {
		sv := savView(t, w, m, kind).Saving
		if sv == nil {
			t.Fatalf("kind %d: Saving = nil for an agent with a block, want a group", kind)
		}
		if !reflect.DeepEqual(sv.Block, *cfg) {
			t.Errorf("kind %d: Block = %+v, want %+v", kind, sv.Block, *cfg)
		}
		if sv.SafetyTicks != 2 || sv.HurdleSpreadBP != 2_000 || sv.CommitBandBP != 2_000 {
			t.Errorf("kind %d: module values = %d, %d, %d, want 2, 2000, 2000", kind, sv.SafetyTicks, sv.HurdleSpreadBP, sv.CommitBandBP)
		}
	}
}

// TestView_SavingClassesAndFacilities verifies that the group carries the classes a clause of
// the role names in ascending ID, with the numbers of the class, and the facilities of the
// agent, in the Work view and the Trade view alike.
//
// Arithmetic: the miller's recipes build the barn, the mill (build and repair) and the road,
// so the classes are barn, mill, road: IDs 0, 2 and 3 (the order of the configuration). A mill
// at condition 600 of 1 000 has cap = ceil(2 * 600 / 1 000) = 2.
//
// Requirement: FR-SAV-028
func TestView_SavingClassesAndFacilities(t *testing.T) {
	t.Parallel()
	w := savWorld(t, nil)
	m := bldMiller(w, 0)
	bldAdd(t, w, clsMill, m, 1, 600)
	for _, kind := range []viewKind{viewWork, viewTrade} {
		sv := savView(t, w, m, kind).Saving
		var names []string
		for _, c := range sv.Classes {
			names = append(names, c.Name)
		}
		if want := []string{"barn", "mill", "road"}; !slices.Equal(names, want) {
			t.Errorf("kind %d: class names = %v, want %v", kind, names, want)
		}
		mill := sv.class(clsMill)
		if mill == nil || *mill != (SavingClass{Class: clsMill, Name: "mill", Kind: "site", Durability: 1_000, Decay: 1, WearPerUse: 2, Capacity: 2}) {
			t.Errorf("kind %d: mill class = %+v, want the site of capacity 2", kind, mill)
		}
		if len(sv.Facilities) != 1 || sv.Facilities[0].Class != clsMill || sv.Facilities[0].Capacity != 2 {
			t.Errorf("kind %d: Facilities = %+v, want one mill of capacity 2 at condition 600", kind, sv.Facilities)
		}
	}
}

// TestView_FacilitiesInTradeView verifies that the Trade view of an agent with a block carries
// the Facilities of the view too, which the infrastructure module alone leaves to the Work
// view, and that an agent of a role with no block has none in the Trade view.
//
// Requirement: FR-SAV-028
func TestView_FacilitiesInTradeView(t *testing.T) {
	t.Parallel()
	w := savWorld(t, nil)
	m, wv := bldMiller(w, 0), facAgents(w, 0, "weaver")[0]
	bldAdd(t, w, clsMill, m, 2)
	if _, err := w.AddFacilities([]NewFacility{{Class: clsMill, Region: 0, Toward: NoRegion, Owner: wv}}); err != nil {
		t.Fatalf("AddFacilities(weaver mill) error = %v, want nil", err)
	}
	work, trade := savView(t, w, m, viewWork), AgentView{}
	wf := slices.Clone(work.Facilities)
	trade = savView(t, w, m, viewTrade)
	if len(trade.Facilities) != 2 || !slices.Equal(trade.Facilities, wf) {
		t.Errorf("miller Trade view Facilities = %+v, want the Work view's %+v", trade.Facilities, wf)
	}
	if !slices.Equal(trade.Saving.Facilities, trade.Facilities) {
		t.Errorf("Saving.Facilities = %+v, want the view's %+v", trade.Saving.Facilities, trade.Facilities)
	}
	if other := savView(t, w, wv, viewTrade); other.Facilities != nil || other.Saving != nil {
		t.Errorf("weaver Trade view: Facilities = %+v, Saving = %+v, want none (no block)", other.Facilities, other.Saving)
	}
}

// TestView_NoBlockHasNoGroup verifies that a role with no block, a world with the saving module
// on and a world with it off give a view with no Saving group and a recipe with none of the
// clause fields: a strategy of such a role sees nothing new.
//
// Requirement: FR-SAV-001, FR-SAV-028
// Guard: every new field is set only for a block; one set for all roles would show a clause to
// strategies that never asked and would move no golden only by luck.
//
// Fix: the cases share one world, so its subtests must not run in parallel; they did, and the race
// detector failed this test and any test running beside it, intermittently (plan 3-fis PR 7 stage 6).
func TestView_NoBlockHasNoGroup(t *testing.T) {
	t.Parallel()
	on := savWorld(t, nil)
	off, _, _ := bldWorld(t, nil)
	tests := []struct {
		name string
		w    *World
		id   AgentID
	}{
		{"role with no block, module on", on, facAgents(on, 0, "weaver")[0]},
		{"role with a block, module off", off, bldMiller(off, 0)},
		{"role with no block, module off", off, facAgents(off, 0, "weaver")[0]},
	}
	for _, tt := range tests {
		// Not parallel: two cases share the world off, and a view call rewrites the world's view context
		// and scratch (refreshViewCtx, fillAgentView), so parallel subtests raced under -race.
		t.Run(tt.name, func(t *testing.T) {
			for _, kind := range []viewKind{viewWork, viewTrade} {
				v := savView(t, tt.w, tt.id, kind)
				if v.Saving != nil {
					t.Errorf("kind %d: Saving = %+v, want nil", kind, v.Saving)
				}
				for _, r := range v.Recipes {
					if r.Needs != nil || r.Builds.On || r.Repairs.On || r.Standing != "" {
						t.Errorf("kind %d: recipe %d has clause fields %+v, want none", kind, r.Recipe, r)
					}
				}
			}
		})
	}
}

// TestView_SavingDrawdownIsTheWorlds verifies that the drawdown of the group is the world's
// function of the agent's ring, with the records and the cumulative income less cost that the
// reach test reads, and that a strategy that writes the group changes nothing: the next view
// has the world's values again.
//
// Arithmetic: five records of net -400, -100, -100, -100, +2 400 give the running sums -400,
// -500, -600, -700, +1 700: the peak stays 0, the deepest fall is 700 and C_n is 1 700.
//
// Requirement: FR-SAV-008
func TestView_SavingDrawdownIsTheWorlds(t *testing.T) {
	t.Parallel()
	w := savWorld(t, nil)
	m := bldMiller(w, 0)
	for _, net := range []int64{-400, -100, -100, -100, 2_400} {
		w.agents[m].history.Push(welfare.Record{Income: max(net, 0), Cost: max(-net, 0)})
	}
	v := savView(t, w, m, viewWork)
	if got := v.Saving; got.Drawdown != 700 || got.Records != 5 || got.Surplus != 1_700 {
		t.Fatalf("Saving drawdown, records, surplus = %d, %d, %d, want 700, 5, 1700", got.Drawdown, got.Records, got.Surplus)
	}
	v.Saving.Drawdown, v.Saving.Block.HurdleBP, v.Saving.Block.Worth[0].PerTick = 0, 1, 0
	v.Saving.Classes[0].Durability = 0
	again := savView(t, w, m, viewTrade)
	if again.Saving.Drawdown != 700 || again.Saving.Block.HurdleBP != 15_000 || again.Saving.Block.Worth[0].PerTick != 15 || again.Saving.Classes[0].Durability != 800 {
		t.Errorf("next view after a strategy wrote the group: %+v, want the world's values again", again.Saving)
	}
	if w.cfg.Roles[0].Saver.Worth[0].PerTick != 15 {
		t.Errorf("configuration Worth PerTick = %d after a write to the view, want 15", w.cfg.Roles[0].Saver.Worth[0].PerTick)
	}
}

// standingCase is one case of TestView_ClausesAndStandingOnRecipes.
type standingCase struct {
	name   string
	setup  func(t *testing.T, w *World, m AgentID)
	recipe string
	want   Reason
}

// standingCases are the refusals of 4.2.2 that the infrastructure fixture can reach, one case
// for each, with the recipe whose Standing is read.
func standingCases() []standingCase {
	atCap := func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent) }
	oneMill := func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, 1) }
	return []standingCase{
		{"nothing refuses a build", nil, "zz_build", ""},
		{"the owner is at the cap: facility_cap", atCap, "zz_build", ReasonFacilityCap},
		{"a build that needs a site the owner has none of: no_facility", nil, "zz_upgrade", ReasonNoFacility},
		{"the same with a site: nothing refuses", oneMill, "zz_upgrade", ""},
		{"nothing worn: nothing_to_repair", oneMill, "zz_fix", ReasonNothingToRepair},
		{"a worn mill: nothing refuses the repair", func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, 1, 700) }, "zz_fix", ""},
		{"a route toward the region itself: no_target", nil, "zz_road_here", ReasonNoTarget},
		{"a route toward yonder: nothing refuses", nil, "zz_road", ""},
		{"a barn for a household with room: nothing refuses", nil, "zz_barn", ""},
		{"no input does not mask a standing refusal", func(t *testing.T, w *World, m AgentID) {
			atCap(t, w, m)
			bldGrain(t, w, m)
		}, "zz_build", ReasonFacilityCap},
	}
}

// TestView_ClausesAndStandingOnRecipes verifies the Standing of 5.2.2 for a recipe with a
// clause: the first reason of CAP's order (4.2.2) other than no_input and no_tool, or empty,
// one case for each reason that the infrastructure fixture can reach; and that the first
// reason no_input does not mask it (the last case: the household holds no grain).
//
// Requirement: FR-SAV-028, FR-SAV-047
func TestView_ClausesAndStandingOnRecipes(t *testing.T) {
	t.Parallel()
	upgrade := Recipe{ID: "zz_upgrade", TraineeYield: 5_000, Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 1}},
		Needs: []string{"mill"}, Builds: &FacilityBuild{Class: "mill"}}
	here := Recipe{ID: "zz_road_here", TraineeYield: 5_000, Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 1}},
		Builds: &FacilityBuild{Class: "road", Toward: "home"}}
	for _, tt := range standingCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := savWorld(t, func(c *Config) {
				c.Recipes = append(c.Recipes, upgrade, here)
				c.Roles[0].Recipes = append(c.Roles[0].Recipes, "zz_road_here", "zz_upgrade")
			})
			m := bldMiller(w, 0)
			if tt.setup != nil {
				tt.setup(t, w, m)
			}
			v := savView(t, w, m, viewTrade)
			if got := recipeOf(t, w, &v, tt.recipe).Standing; got != tt.want {
				t.Errorf("Standing of %s = %q, want %q (seed %d)", tt.recipe, got, tt.want, facSeed)
			}
		})
	}
}

// TestView_ClauseFieldsOnRecipes verifies the per-recipe Needs, Builds and Repairs of 5.2.2 for
// a role with a block: a recipe that needs a class has it in Needs and, having no clause, no
// Standing; a repair and a build carry their clause.
//
// Requirement: FR-SAV-028
func TestView_ClauseFieldsOnRecipes(t *testing.T) {
	t.Parallel()
	w := savWorld(t, nil)
	v := savView(t, w, bldMiller(w, 0), viewWork)
	grind, fix, build := recipeOf(t, w, &v, "zz_grind"), recipeOf(t, w, &v, "zz_fix"), recipeOf(t, w, &v, "zz_build")
	if !slices.Equal(grind.Needs, []FacilityClassID{clsMill}) || grind.Builds.On || grind.Repairs.On || grind.Standing != "" {
		t.Errorf("zz_grind clause fields = %+v, want Needs [mill] and nothing else (no clause, no Standing)", grind)
	}
	if got, want := fix.Repairs, (RepairView{On: true, Class: clsMill, Units: 100}); got != want {
		t.Errorf("zz_fix Repairs = %+v, want %+v", got, want)
	}
	if got, want := build.Builds, (BuildView{On: true, Class: clsMill, Toward: NoRegion}); got != want {
		t.Errorf("zz_build Builds = %+v, want %+v", got, want)
	}
}

// TestConstruct_SavingTablesResolved verifies the tables of the module at construction: the
// state is nil with the module off, a role with a block has its classes resolved to the rows
// of the class table in ascending ID, a role with none has no block, and a saver role of a
// world with no infrastructure module has no class and no project.
//
// Requirement: FR-SAV-026
func TestConstruct_SavingTablesResolved(t *testing.T) {
	t.Parallel()
	if off, _, _ := bldWorld(t, nil); off.saving != nil {
		t.Errorf("saving state with the module off = %v, want nil", off.saving)
	}
	w := savWorld(t, nil)
	if w.saving == nil || len(w.saving.roles) != len(w.cfg.Roles) {
		t.Fatalf("saving state = %v, want one table per role (%d)", w.saving, len(w.cfg.Roles))
	}
	got := w.saving.roles[0]
	want := []SavingClass{
		{Class: clsBarn, Name: "barn", Kind: "store", Durability: 800},
		{Class: clsMill, Name: "mill", Kind: "site", Durability: 1_000, Decay: 1, WearPerUse: 2, Capacity: 2},
		{Class: clsRoad, Name: "road", Kind: "route", Durability: 2_000},
	}
	if got.block != w.cfg.Roles[0].Saver || !slices.Equal(got.classes, want) {
		t.Errorf("miller table = block %v, classes %+v, want the configured block and %+v", got.block, got.classes, want)
	}
	if weaver := w.saving.roles[1]; weaver.block != nil || weaver.classes != nil {
		t.Errorf("weaver table = %+v, want no block and no class", weaver)
	}
	if !slices.Contains(modulesOn(&w.cfg), "saving") {
		t.Errorf("modulesOn = %v, want saving in it", modulesOn(&w.cfg))
	}
}

// TestSaving_NoMapRange is a text guard (NFR-SAV-001): no file of the saving module ranges over
// a map, because map order is random and an iteration that reached a result would break
// determinism. It finds a variable declared with a map type or made with make(map...) in the
// file and fails on a range statement over that name.
//
// Requirement: NFR-SAV-001
func TestSaving_NoMapRange(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("saving*.go")
	if err != nil {
		t.Fatalf("Glob(saving*.go) error = %v, want nil", err)
	}
	more, err := filepath.Glob("internal/saving/*.go")
	if err != nil {
		t.Fatalf("Glob(internal/saving/*.go) error = %v, want nil", err)
	}
	for _, f := range append(files, more...) {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v, want nil", f, err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s) error = %v, want nil", f, err)
		}
		maps := mapNames(file)
		ast.Inspect(file, func(n ast.Node) bool {
			if rs, ok := n.(*ast.RangeStmt); ok {
				if id, ok := rs.X.(*ast.Ident); ok && maps[id.Name] {
					t.Errorf("%s: range over the map %s", fset.Position(rs.Pos()), id.Name)
				}
			}
			return true
		})
	}
}

// mapNames are the names a file declares as a map: by an explicit map type or by make(map...).
func mapNames(file *ast.File) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			madeMaps(x, out)
		case *ast.ValueSpec:
			if _, isMap := x.Type.(*ast.MapType); isMap {
				for _, id := range x.Names {
					out[id.Name] = true
				}
			}
		}
		return true
	})
	return out
}

// madeMaps records the names an assignment sets to make(map...).
func madeMaps(x *ast.AssignStmt, out map[string]bool) {
	for i, rhs := range x.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 || i >= len(x.Lhs) {
			continue
		}
		if _, isMap := call.Args[0].(*ast.MapType); !isMap {
			continue
		}
		if id, ok := x.Lhs[i].(*ast.Ident); ok {
			out[id.Name] = true
		}
	}
}

// TestCost_SavingViewZeroAllocs verifies that building the Work and Trade views of a saver
// allocates nothing once the scratch has grown: the drawdown is read into scratch, the
// group, the classes, the Worth rows and the per-recipe Needs are copied into scratch, and the
// Standing reasons are derived without a map or a slice of their own (NFR-SAV-002).
//
// Requirement: NFR-SAV-002
func TestCost_SavingViewZeroAllocs(t *testing.T) {
	w := savWorld(t, nil)
	m := bldMiller(w, 0)
	bldAdd(t, w, clsMill, m, 3, 400, 700, 1_000)
	for range 12 {
		w.agents[m].history.Push(welfare.Record{Income: 50, Cost: 100})
	}
	w.refreshViewCtx()
	a := &w.agents[m]
	fill := func() {
		for _, kind := range []viewKind{viewWork, viewTrade} {
			if err := w.fillAgentView(&w.tk.view, a, kind); err != nil {
				t.Fatalf("fillAgentView() error = %v, want nil", err)
			}
		}
	}
	fill() // grows the scratch
	if got := testing.AllocsPerRun(20, fill); got != 0 {
		t.Errorf("AllocsPerRun(fill Work and Trade views of a saver) = %v, want 0", got)
	}
}
