package economysim

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// viewConfig is the configuration of the filler tests. Commodities sort to ash
// 0, bread 1 (spoils in 20 ticks), flour 2, grain 3, oven 4 (durability 20).
// Recipes sort to bake 0 (grain-free: flour 2 and the tool oven make bread 3),
// burn 1 (makes ash), farm 2 (makes grain and throws ash away), forge 3 (makes
// the tool) and mill 4. Roles sort to baker 0, burner 1, clerk 2 (no recipe;
// trades oven and grain), farmer 3 (trades ash, which only its discard makes),
// miller 4 and smith 5. The world upkeep is bread 1; the region south adds grain
// 1. Regions sort to north 0 and south 1.
func viewConfig() Config {
	w := DefaultWorld()
	w.Upkeep = []Item{{Commodity: "bread", Quantity: 1}}
	return Config{
		World:       w,
		Regions:     []Region{{ID: "north"}, {ID: "south", ExtraUpkeep: []Item{{Commodity: "grain", Quantity: 1}}}},
		Commodities: []Commodity{{ID: "ash", ReferencePrice: 20}, {ID: "bread", ReferencePrice: 100, ShelfLife: 20}, {ID: "flour", ReferencePrice: 40}, {ID: "grain", ReferencePrice: 30}, {ID: "oven", ReferencePrice: 900, Durability: 20}},
		Recipes: []Recipe{
			{ID: "bake", Duration: 1, Inputs: []Item{{Commodity: "flour", Quantity: 2}},
				Outputs: []Output{{Commodity: "bread", Quantity: 3, CostShare: 10_000}}, Requires: []Tool{{Commodity: "oven", WearPerUse: 5}}},
			{ID: "burn", Duration: 1, Outputs: []Output{{Commodity: "ash", Quantity: 1, CostShare: 10_000}}},
			{ID: "farm", Duration: 3, Outputs: []Output{{Commodity: "grain", Quantity: 6, CostShare: 10_000}, {Commodity: "ash", Quantity: 2, Discard: true}}},
			{ID: "forge", Duration: 1, Outputs: []Output{{Commodity: "oven", Quantity: 1, CostShare: 10_000}}},
			{ID: "mill", Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 3}}, Outputs: []Output{{Commodity: "flour", Quantity: 2, CostShare: 10_000}}},
		},
		Roles: []Role{
			{ID: "baker", Recipes: []string{"bake"}, TargetRatio: 15_000, DefaultRatio: 12_000, MinWorkShare: 2_000},
			{ID: "burner", Recipes: []string{"burn"}, TargetRatio: 15_000},
			{ID: "clerk", Trades: []string{"oven", "grain"}, TargetRatio: 15_000},
			{ID: "farmer", Recipes: []string{"farm"}, Trades: []string{"ash"}, TargetRatio: 15_000},
			{ID: "miller", Recipes: []string{"mill"}, TargetRatio: 15_000},
			{ID: "smith", Recipes: []string{"forge"}, TargetRatio: 15_000},
		},
	}
}

// TestFillCore_Groups verifies, with the numbers worked out below, every core
// group of a view for a cohort of four farmers in the region south: identity,
// money, state, the merged upkeep, the job, the tradable set in canonical order
// with stocks, beliefs, bounds, reference prices, the market report and the
// price window, the recipes with their lines and the executions allowed.
//
// The farmer's tradable set is bread (upkeep), grain (output) and ash (trade).
// South's upkeep is bread 1 plus grain 1 per household. The window holds 100
// and 140 (mean 120, low 100, high 140) and the bread price 120 lies at
// (120-100)*10 000/40 = 5 000 of it. The welfare ring holds two ticks (income 300 and 100, cost 500 and
// 200) and its other 28 are padded at DefaultRatio, so the ratio is
// welfare.Ratio(400, 700, 28, cost, default) with cost = 4*(120 + 30) = 600 at the published prices.
//
// Requirement: FR-AGT-024, FR-AGT-071, FR-AGT-073, FR-AGT-038
func TestFillCore_Groups(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "farmer", "south", 4)
	h.Give(t, i, "grain", 10)
	h.Give(t, i, "bread", 7)
	h.Fund(t, i, 5000)
	h.SetHunger(i, 2, 1)
	h.PushHistory(i, 300, 500, false)
	h.PushHistory(i, 100, 200, false)
	h.SetReport(t, "south", "bread", MarketReport{Price: 120, Age: 3, BestBid: 110, HasBestBid: true})
	h.PushWindow(t, "south", "bread", 100)
	h.PushWindow(t, "south", "bread", 140)
	v, err := h.Fill(i)
	if err != nil {
		t.Fatalf("fillCore error = %v, want nil", err)
	}
	a := h.agents[i]
	bread, grain, ash := h.commodity(t, "bread"), h.commodity(t, "grain"), h.commodity(t, "ash")
	ratio, present := welfare.Ratio(400, 700, 28, 600, 0)
	if !present {
		t.Fatalf("welfare.Ratio is absent, want present")
	}
	bp := h.ctx.bp
	limits := func(ref int64) Bounds {
		b := beliefs.Limits(ref, 0, bp)
		return Bounds{Floor: Price(b.Floor), Ceiling: Price(b.Ceiling)}
	}
	if v.Agent != 0 || v.Role != a.role || v.Region != 1 || v.Weight != 4 || v.External || !reflect.DeepEqual(v.Tick, h.ctx.moment) {
		t.Errorf("identity = %+v, want agent 0, role %d, region 1, weight 4", v, a.role)
	}
	if v.Money != 5000 || v.Hunger != 2 || v.Unfed != 1 || v.TargetRatio != 15_000 || v.Window != h.roles.row(a.role).window ||
		v.WorkingTicks != 0 || v.StartsIn != 1 || v.HasJob || v.MinWorkShare != 0 {
		t.Errorf("money and state = %+v, want money 5000, hunger 2, unfed 1, target 15000, no job, starts in 1", v)
	}
	if v.Ratio != ratio || !v.HasRatio {
		t.Errorf("ratio = %d (present %v), want %d", v.Ratio, v.HasRatio, ratio)
	}
	if want := []UpkeepRow{{Commodity: bread, Quantity: 1}, {Commodity: grain, Quantity: 1}}; !slices.Equal(v.Upkeep, want) {
		t.Errorf("Upkeep = %v, want %v", v.Upkeep, want)
	}
	want := []CommodityView{
		{Commodity: ash, Stock: 0, Ref: 20, Belief: Belief{Centre: 20, Width: bp.WidthInit}, Bounds: limits(20),
			Market: MarketReport{Price: 20}},
		{Commodity: bread, Stock: 7, Ref: 100, Belief: Belief{Centre: 100, Width: bp.WidthInit}, Bounds: limits(100),
			Market: MarketReport{Price: 120, Age: 3, BestBid: 110, HasBestBid: true},
			Window: PriceWindowView{Mean: 120, Low: 100, High: 140, Position: 5000, HasWindow: true}},
		{Commodity: grain, Stock: 10, Ref: 30, Belief: Belief{Centre: 30, Width: bp.WidthInit}, Bounds: limits(30),
			Market: MarketReport{Price: 30}},
	}
	if !slices.Equal(v.Commodities, want) {
		t.Errorf("Commodities =\n%+v\nwant\n%+v", v.Commodities, want)
	}
	if len(v.Recipes) != 1 {
		t.Fatalf("len(Recipes) = %d, want 1", len(v.Recipes))
	}
	r := v.Recipes[0]
	if r.Recipe != 2 || r.Duration != 3 || len(r.Inputs) != 0 || len(r.Tools) != 0 || r.Effective != 10_000 || r.MaxExecutions != 4 || r.Reason != "" {
		t.Errorf("recipe = %+v, want farm, duration 3, no lines, yield 10000, 4 executions (a source runs the weight)", r)
	}
	if want := []RecipeOutput{{Commodity: ash, Quantity: 2, Discard: true}, {Commodity: grain, Quantity: 6, CostShare: 10_000}}; !slices.Equal(r.Outputs, want) {
		t.Errorf("Outputs = %v, want %v", r.Outputs, want)
	}
	if got, want := v.Tunables.Beliefs, beliefTunablesOf(bp); got != want {
		t.Errorf("belief tunables = %+v, want %+v", got, want)
	}
	if v.Tunables.CoverTicks != h.Cfg.World.CoverTicks || v.Tunables.RestBand != h.Cfg.World.RestBand {
		t.Errorf("tunables = %+v, want the World section", v.Tunables)
	}
}

// TestFillCore_Executions verifies the per-recipe executions and their reason
// against the closed form and against a direct count: a miller needs 3 grain
// for each execution, so 7 grain allow 2 of a weight of 5, none without grain
// (no_input), and a baker needs 2 flour and an oven (no_tool without one).
//
// Requirement: FR-AGT-071, FR-AGT-028
func TestFillCore_Executions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		role       string
		weight     int64
		gives      map[string]int64
		wantK      int64
		wantReason Reason
	}{
		{"grain for two of five", "miller", 5, map[string]int64{"grain": 7}, 2, ""},
		{"grain for more than the weight", "miller", 2, map[string]int64{"grain": 30}, 2, ""},
		{"no grain", "miller", 5, nil, 0, ReasonNoInput},
		{"flour and an oven for each", "baker", 4, map[string]int64{"flour": 6, "oven": 4}, 3, ""},
		{"flour and no oven", "baker", 4, map[string]int64{"flour": 6}, 0, ReasonNoTool},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewViewHarness(t, viewConfig())
			i := h.Spawn(t, tt.role, "north", tt.weight)
			for _, name := range slices.Sorted(mapKeys(tt.gives)) {
				h.Give(t, i, name, tt.gives[name])
			}
			v, err := h.Fill(i)
			if err != nil {
				t.Fatalf("fillCore error = %v, want nil", err)
			}
			direct, dreason := h.MaxExecutions(t, i, v.Recipes[0].Recipe.name(h.Cfg))
			if r := v.Recipes[0]; r.MaxExecutions != tt.wantK || r.Reason != tt.wantReason || direct != tt.wantK || dreason != tt.wantReason {
				t.Errorf("fillCore: executions = %d (%q), direct = %d (%q), want %d (%q)", r.MaxExecutions, r.Reason, direct, dreason, tt.wantK, tt.wantReason)
			}
		})
	}
}

func mapKeys[M ~map[K]V, K comparable, V any](m M) func(yield func(K) bool) {
	return func(yield func(K) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// name is the string ID of a recipe.
func (r RecipeID) name(c *Config) string { return c.Recipes[r].ID }

// TestFillCore_StartsIn verifies that StartsIn is 1 without a job and
// remaining + 1 with one (AGT 6.4.3), with the job fields of the view.
//
// Requirement: FR-AGT-038
func TestFillCore_StartsIn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		remaining  int64
		executions int64
		wantJob    bool
		wantStarts int64
	}{
		{"no job", 0, 0, false, 1},
		{"three ticks to go", 3, 2, true, 4},
		{"one tick to go", 1, 5, true, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewViewHarness(t, viewConfig())
			i := h.Spawn(t, "farmer", "north", 5)
			if tt.wantJob {
				h.SetJob(t, i, "farm", tt.remaining, tt.executions)
			}
			v, err := h.Fill(i)
			if err != nil {
				t.Fatalf("fillCore error = %v, want nil", err)
			}
			if v.StartsIn != tt.wantStarts || v.HasJob != tt.wantJob || (tt.wantJob && (v.Recipe != 2 || v.Remaining != tt.remaining || v.Executions != tt.executions)) {
				t.Errorf("job = (has %v, recipe %d, remaining %d, executions %d, starts in %d), want has %v, starts in %d",
					v.HasJob, v.Recipe, v.Remaining, v.Executions, v.StartsIn, tt.wantJob, tt.wantStarts)
			}
		})
	}
}

// TestFillCore_UnitCostAndTool verifies that UnitCost is the region's cost
// anchor only for a commodity the role makes (a non-discard output of one of its
// recipes) and 0 otherwise or while the wage is unknown, that the bounds use the
// same zeroed cost, and that Tool is true for every required tool of any recipe
// whatever the role. The anchor table is nonzero for the oven and the ash, so
// the zeroes are the filler's doing: the clerk trades the oven but cannot make
// it, the farmer trades ash but only throws it away.
//
// Requirement: FR-AGT-048, FR-AGT-071
func TestFillCore_UnitCostAndTool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		wage      int64
		role      string
		commodity string
		wantCost  bool
		wantTool  bool
	}{
		{"the smith makes the oven", 100, "smith", "oven", true, true},
		{"the clerk trades the oven but cannot make it", 100, "clerk", "oven", false, true},
		{"the baker uses the oven as a tool", 100, "baker", "oven", false, true},
		{"the farmer's ash is a discard", 100, "farmer", "ash", false, false},
		{"the burner makes the ash", 100, "burner", "ash", true, false},
		{"an unknown wage makes the oven unknown", 0, "smith", "oven", false, true},
		{"a commodity that is no tool", 100, "miller", "flour", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewViewHarness(t, viewConfig())
			i := h.Spawn(t, tt.role, "north", 1)
			h.Anchors(tt.wage)
			c := h.commodity(t, tt.commodity)
			anchor := h.AnchorOf(0, c)
			if tt.wage > 0 && anchor == 0 {
				t.Fatalf("anchor of %s = 0, want nonzero (a precondition of the test)", tt.commodity)
			}
			v, err := h.Fill(i)
			if err != nil {
				t.Fatalf("fillCore error = %v, want nil", err)
			}
			k := slices.IndexFunc(v.Commodities, func(cv CommodityView) bool { return cv.Commodity == c })
			if k < 0 {
				t.Fatalf("commodity %s not in the tradable set of %s", tt.commodity, tt.role)
			}
			cv := v.Commodities[k]
			wantCost := int64(0)
			if tt.wantCost {
				wantCost = anchor
			}
			bd := beliefs.Limits(int64(cv.Ref), wantCost, h.ctx.bp)
			if int64(cv.UnitCost) != wantCost || cv.Tool != tt.wantTool || cv.Bounds != (Bounds{Floor: Price(bd.Floor), Ceiling: Price(bd.Ceiling)}) {
				t.Errorf("%s of %s: UnitCost = %d, Tool = %v, Bounds = %+v, want %d, %v, %+v",
					tt.commodity, tt.role, cv.UnitCost, cv.Tool, cv.Bounds, wantCost, tt.wantTool, Bounds{Floor: Price(bd.Floor), Ceiling: Price(bd.Ceiling)})
			}
		})
	}
}

// TestFillCore_MoneyIsAvailableCash verifies that the Money of a view is the
// balance less the money earmarked by an admitted bid.
//
// Requirement: FR-AGT-071
func TestFillCore_MoneyIsAvailableCash(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "clerk", "north", 1)
	h.Fund(t, i, 1000)
	grain := h.commodity(t, "grain")
	plan := &TradePlan{Intents: []Intent{{Commodity: grain, Side: Bid, Quantity: 4, Limit: 50}}}
	if res := h.Admit(t, i, plan); !res[0].OK {
		t.Fatalf("Admit = %+v, want admitted", res[0])
	}
	v, err := h.Fill(i)
	if err != nil {
		t.Fatalf("fillCore error = %v, want nil", err)
	}
	if v.Money != 800 {
		t.Errorf("Money = %d, want 1000 - 4*50 = 800", v.Money)
	}
}

// TestFillCore_ExpiringUnits verifies the units that spoil this tick:
// bread has a shelf life of 20, so bread born at tick 0 expires at the end of
// tick 19 and not before.
//
// Requirement: FR-AGT-071
func TestFillCore_ExpiringUnits(t *testing.T) {
	t.Parallel()
	for _, tick := range []int64{0, 19} {
		t.Run(fmt.Sprintf("tick %d", tick), func(t *testing.T) {
			t.Parallel()
			h := NewViewHarness(t, viewConfig())
			i := h.Spawn(t, "miller", "north", 1)
			h.Give(t, i, "bread", 5)
			h.ctx.moment.Tick = tick
			v, err := h.Fill(i)
			if err != nil {
				t.Fatalf("fillCore error = %v, want nil", err)
			}
			bread := h.commodity(t, "bread")
			k := slices.IndexFunc(v.Commodities, func(cv CommodityView) bool { return cv.Commodity == bread })
			want := int64(0)
			if tick == 19 {
				want = 5
			}
			if got := v.Commodities[k].Expiring; got != want || v.Commodities[k].Stock != 5 {
				t.Errorf("tick %d: Expiring = %d, Stock = %d, want %d, 5", tick, got, v.Commodities[k].Stock, want)
			}
		})
	}
}

// TestFillCore_DirectiveIsACopy verifies that a pending directive is copied into
// the view by value, with its slices and its job in the scratch, so that writing
// through the view cannot change the agent's directive (D97), and that an agent
// with no directive has none in the view.
//
// Requirement: FR-AGT-024, FR-AGT-056
func TestFillCore_DirectiveIsACopy(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "farmer", "north", 1)
	if v, err := h.Fill(i); err != nil || v.HasDirective {
		t.Fatalf("no directive: fillCore error = %v, HasDirective = %v, want nil, false", err, v.HasDirective)
	}
	d := &Directive{
		Job:      &JobRequest{Recipe: 2, Executions: 1},
		Intents:  []Intent{{Commodity: 3, Side: Bid, Quantity: 2, Limit: 30}},
		Discards: []Discard{{Commodity: 0, Quantity: 1}},
		Hold:     []CommodityID{1},
		Sticky:   true,
	}
	h.SetDirective(i, d)
	v, err := h.Fill(i)
	if err != nil || !v.HasDirective {
		t.Fatalf("directive: fillCore error = %v, HasDirective = %v, want nil, true", err, v.HasDirective)
	}
	if !reflect.DeepEqual(&v.Directive, d) {
		t.Errorf("view directive = %+v, want %+v", v.Directive, *d)
	}
	v.Directive.Job.Executions, v.Directive.Intents[0].Quantity, v.Directive.Hold[0] = 99, 99, 99
	if d.Job.Executions != 1 || d.Intents[0].Quantity != 2 || d.Hold[0] != 1 {
		t.Errorf("the agent's directive = %+v after a write through the view, want it unchanged", *d)
	}
}

// scribble overwrites every exported value reachable from v, as a hostile
// strategy might: numbers, flags and strings, through pointers and slices.
func scribble(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			scribble(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				scribble(v.Field(i))
			}
		}
	case reflect.Slice:
		for i := range v.Len() {
			scribble(v.Index(i))
		}
	case reflect.Int, reflect.Int64:
		v.SetInt(-7777)
	case reflect.Bool:
		v.SetBool(!v.Bool())
	case reflect.String:
		v.SetString("scribbled")
	}
}

// TestFillCore_AStrategyThatWritesTheViewChangesNothing verifies that a
// strategy that overwrites every field of a view, and appends to its slices,
// leaves the agent, the ledger, the configuration, the tables and the next fill
// unchanged: the slices are scratch copies with a capped capacity.
//
// Requirement: FR-AGT-024
//
// Guard: the view is the only channel from the world to a strategy and it holds
// the strategy's own copies; if a field pointed into live state, a faulty
// strategy would corrupt the world and the run would not be reproducible.
func TestFillCore_AStrategyThatWritesTheViewChangesNothing(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "farmer", "south", 3)
	h.Give(t, i, "grain", 9)
	h.Fund(t, i, 777)
	d := &Directive{Job: &JobRequest{Recipe: 2}, Intents: []Intent{{Commodity: 3, Side: Bid, Quantity: 2, Limit: 30}}, Hold: []CommodityID{1}}
	h.SetDirective(i, d)
	a := h.agents[i]
	before, err := h.Fill(i)
	if err != nil {
		t.Fatalf("fillCore error = %v, want nil", err)
	}
	snap := fmt.Sprintf("%+v", *before)
	bel, res := slices.Clone(a.beliefs), slices.Clone(a.residues)
	dir := *d
	dir.Job, dir.Intents, dir.Hold = new(JobRequest), slices.Clone(d.Intents), slices.Clone(d.Hold)
	*dir.Job = *d.Job
	cfg, rt, roles := h.Cfg.canonical(), newRecipeTable(h.Cfg), newRoleTable(h.Cfg)
	bal, held := h.balance(t, i)

	v, _ := h.Fill(i)
	scribble(reflect.ValueOf(v))
	v.Commodities = append(v.Commodities, CommodityView{})
	v.Recipes = append(v.Recipes[:0], RecipeView{Inputs: []RecipeLine{{Quantity: 5}}})

	if !slices.Equal(a.beliefs, bel) || !slices.Equal(a.residues, res) || !reflect.DeepEqual(d, &dir) {
		t.Errorf("the agent changed: beliefs %v residues %v directive %+v, want %v %v %+v", a.beliefs, a.residues, d, bel, res, dir)
	}
	if b2, h2 := h.balance(t, i); b2 != bal || h2 != held {
		t.Errorf("the ledger changed: balance %d holding %d, want %d %d", b2, h2, bal, held)
	}
	if !reflect.DeepEqual(*h.Cfg, cfg) || !reflect.DeepEqual(h.rt.rows, rt.rows) || !reflect.DeepEqual(h.roles.rows, roles.rows) {
		t.Errorf("the configuration or a table changed after a write through the view")
	}
	after, err := h.Fill(i)
	if err != nil {
		t.Fatalf("second fillCore error = %v, want nil", err)
	}
	if got := fmt.Sprintf("%+v", *after); got != snap {
		t.Errorf("next fill =\n%s\nwant\n%s", got, snap)
	}
}

func (h *ViewHarness) balance(t *testing.T, i int) (balance, held int64) {
	t.Helper()
	b, _ := h.Balance(t, i)
	return b, h.Held(t, i, "grain")
}

// TestFillCore_Errors verifies that a ledger refusal and a belief list that does
// not match the tradable set are returned, not panicked, and wrap the cause.
//
// Requirement: FR-AGT-071
func TestFillCore_Errors(t *testing.T) {
	t.Parallel()
	t.Run("unknown account", func(t *testing.T) {
		t.Parallel()
		h := NewViewHarness(t, viewConfig())
		i := h.Spawn(t, "farmer", "north", 1)
		h.agents[i].account = 1 << 20
		if _, err := h.Fill(i); !errors.Is(err, ledger.ErrUnknownAccount) {
			t.Errorf("fillCore error = %v, want ErrUnknownAccount", err)
		}
	})
	t.Run("beliefs do not match the tradable set", func(t *testing.T) {
		t.Parallel()
		h := NewViewHarness(t, viewConfig())
		i := h.Spawn(t, "farmer", "north", 1)
		h.agents[i].beliefs = h.agents[i].beliefs[:1]
		if _, err := h.Fill(i); err == nil {
			t.Errorf("fillCore error = nil, want an error")
		}
	})
}

// TestFillCore_AllocatesNothingWarmed verifies that with a warmed scratch the
// fill, and Work and Trade on the filled view, allocate nothing.
//
// Requirement: NFR-AGT-001
func TestFillCore_AllocatesNothingWarmed(t *testing.T) {
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "baker", "south", 7)
	h.Give(t, i, "flour", 20)
	h.Give(t, i, "oven", 1)
	h.Fund(t, i, 4000)
	rng := rand.New(rand.NewPCG(1, 2))
	plan := h.Plan()
	run := func() {
		v, err := h.Fill(i)
		if err != nil {
			panic(err)
		}
		DefaultStrategy{}.Work(v, rng)
		plan = h.Scratch().tradePlan()
		DefaultStrategy{}.Trade(v, rng, plan)
	}
	run()
	if got := testing.AllocsPerRun(50, run); got != 0 {
		t.Errorf("fill, Work and Trade allocate %v times per run, want 0", got)
	}
	if len(plan.Intents) == 0 {
		t.Errorf("Trade planned nothing, want at least one order for the allocation test to mean something")
	}
}

// TestNewViewScratch_SizedFromTheLargestRole verifies that the scratch has room
// for the largest role's tradable set, recipes, upkeep and lines, so that a
// fill never grows it.
//
// Requirement: NFR-AGT-001
func TestNewViewScratch_SizedFromTheLargestRole(t *testing.T) {
	t.Parallel()
	cfg := viewConfig().canonical()
	rt, rc := newRoleTable(&cfg), newRecipeTable(&cfg)
	sc := newViewScratch(&cfg, &rt, &rc)
	// The most tradable commodities are the baker's: bread, flour, grain (south's
	// upkeep puts grain in every set) and the oven.
	if got := cap(sc.commodities); got != 4 {
		t.Errorf("cap(commodities) = %d, want 4", got)
	}
	// South adds grain to the bread of the world: two upkeep rows at most.
	if got := cap(sc.upkeep); got != 2 {
		t.Errorf("cap(upkeep) = %d, want 2", got)
	}
	// The baker's one recipe has 1 input, 1 output and 1 tool; the farmer's has 2 outputs.
	if cap(sc.recipes) != 1 || cap(sc.inputs) != 1 || cap(sc.outputs) != 2 || cap(sc.tools) != 1 {
		t.Errorf("cap(recipes, inputs, outputs, tools) = %d, %d, %d, %d, want 1, 1, 2, 1",
			cap(sc.recipes), cap(sc.inputs), cap(sc.outputs), cap(sc.tools))
	}
}

// TestRecipeTable_Tool verifies that a commodity is a tool when some recipe of
// the configuration requires it, whichever role has the recipe, and that a
// commodity out of range is not.
//
// Requirement: FR-AGT-071
func TestRecipeTable_Tool(t *testing.T) {
	t.Parallel()
	cfg := viewConfig().canonical()
	rt := newRecipeTable(&cfg)
	for id, want := range map[string]bool{"ash": false, "bread": false, "flour": false, "grain": false, "oven": true} {
		c, _ := cfg.commodityID(id)
		if got := rt.tool(c); got != want {
			t.Errorf("tool(%s) = %v, want %v", id, got, want)
		}
	}
	if rt.tool(-1) || rt.tool(99) {
		t.Errorf("tool of an ID out of range = true, want false")
	}
}

// TestBeliefTunables_EveryFieldIsCarried verifies, over parameters that are all
// different, that the belief group of a view converts back to the same
// beliefs.Params field by field (D82).
//
// Guard: the filler writes the group and the default strategy reads it; a
// parameter dropped or swapped in the copy would give the default a different
// update from the world's own.
func TestBeliefTunables_EveryFieldIsCarried(t *testing.T) {
	t.Parallel()
	var p beliefs.Params
	pv := reflect.ValueOf(&p).Elem()
	for i := range pv.NumField() {
		pv.Field(i).SetInt(int64(100 + i))
	}
	p.MaxPrice = MaxPrice
	bt := beliefTunablesOf(p)
	if got := beliefParamsOf(&bt); got != p {
		t.Errorf("beliefParamsOf(beliefTunablesOf(%+v)) = %+v, want it back (MaxPrice is the root bound %d)", p, got, MaxPrice)
	}
}

// TestFillCore_AppendingToAViewSliceChangesNothing verifies that no slice of a
// view has spare capacity, so that an append to it copies and cannot write into
// the neighbouring scratch region (the lines of the next recipe) or into the
// scratch beyond it, and that the other recipes' lines are unchanged after an
// append to every slice of the first.
//
// Requirement: FR-AGT-024
//
// Guard: an uncapped slice would let a faulty strategy's append overwrite the
// lines of the next recipe in the same view, or the scratch the next fill reads.
func TestFillCore_AppendingToAViewSliceChangesNothing(t *testing.T) {
	t.Parallel()
	cfg := viewConfig()
	cfg.Regions[1].ExtraUpkeep = []Item{{Commodity: "ash", Quantity: 1}, {Commodity: "flour", Quantity: 1}, {Commodity: "grain", Quantity: 1}, {Commodity: "oven", Quantity: 1}}
	cfg.Commodities = append(cfg.Commodities, Commodity{ID: "zinc", ReferencePrice: 10})
	bake2 := cfg.Recipes[0]
	bake2.ID = "bake2"
	cfg.Recipes = append(cfg.Recipes, bake2)
	cfg.Roles = append(cfg.Roles, Role{ID: "zebra", Recipes: []string{"bake", "bake2", "mill"}, Trades: []string{"ash", "oven", "zinc"}, TargetRatio: 15_000})
	h := NewViewHarness(t, cfg)
	i := h.Spawn(t, "zebra", "south", 2)
	h.Fund(t, i, 1000)
	d := &Directive{
		Intents: make([]Intent, 5), Discards: make([]Discard, 5), Hold: make([]CommodityID, 5),
	}
	h.SetDirective(i, d)
	v, err := h.Fill(i)
	if err != nil {
		t.Fatalf("fillCore error = %v, want nil", err)
	}
	if len(v.Recipes) != 3 || len(v.Recipes[0].Inputs) == 0 || len(v.Recipes[0].Outputs) == 0 || len(v.Recipes[0].Tools) == 0 {
		t.Fatalf("recipes = %+v, want three, the first with inputs, outputs and a tool", v.Recipes)
	}
	rest := fmt.Sprintf("%+v", v.Recipes[1:])
	v.Recipes[0].Inputs = append(v.Recipes[0].Inputs, RecipeLine{Commodity: 9, Quantity: 9})
	v.Recipes[0].Outputs = append(v.Recipes[0].Outputs, RecipeOutput{Commodity: 9, Quantity: 9})
	v.Recipes[0].Tools = append(v.Recipes[0].Tools, RecipeTool{Commodity: 9})
	if got := fmt.Sprintf("%+v", v.Recipes[1:]); got != rest {
		t.Errorf("recipes after the first = %s, want %s unchanged by appends to the first", got, rest)
	}
	// A smaller view filled after the larger one reuses its scratch, so every
	// slice would have spare capacity if it were not capped.
	j := h.Spawn(t, "burner", "north", 1)
	h.SetDirective(j, &Directive{Intents: make([]Intent, 1), Discards: make([]Discard, 1), Hold: make([]CommodityID, 1)})
	w, err := h.Fill(j)
	if err != nil {
		t.Fatalf("second fillCore error = %v, want nil", err)
	}
	if len(w.Recipes) != 1 || len(w.Commodities) == 0 || len(w.Upkeep) == 0 {
		t.Fatalf("view = %+v, want one recipe, some commodities and upkeep", w)
	}
	r := w.Recipes[0]
	for name, lc := range map[string][2]int{
		"Commodities": {len(w.Commodities), cap(w.Commodities)}, "Recipes": {len(w.Recipes), cap(w.Recipes)},
		"Upkeep": {len(w.Upkeep), cap(w.Upkeep)}, "Intents": {len(w.Directive.Intents), cap(w.Directive.Intents)},
		"Discards": {len(w.Directive.Discards), cap(w.Directive.Discards)}, "Hold": {len(w.Directive.Hold), cap(w.Directive.Hold)},
		"Inputs": {len(r.Inputs), cap(r.Inputs)}, "Outputs": {len(r.Outputs), cap(r.Outputs)}, "Tools": {len(r.Tools), cap(r.Tools)},
	} {
		if lc[0] != lc[1] {
			t.Errorf("%s has length %d and capacity %d, want equal", name, lc[0], lc[1])
		}
	}
}
