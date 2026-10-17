package economysim

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// TestNormalise verifies the clamps of spec-agents.md 6.5.2 on an intent: a
// Target or Reserve above MaxQuantity or below 0 is read as 0, a negative
// UnitCost as 0, a Purpose outside the enumeration as PurposeOther, each with
// the changed result, and a clean intent, or one exactly at a bound, is left as
// it is. A Target above MaxQuantity is what an output target can be at the
// bound of OutputCoverTicks (FR-AGT-059 bounds the use targets only).
//
// Requirement: FR-AGT-048
func TestNormalise(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		in, want    Intent
		wantChanged bool
	}{
		{"clean", Intent{Target: 5, Reserve: 3, UnitCost: 7, Purpose: PurposeInput}, Intent{Target: 5, Reserve: 3, UnitCost: 7, Purpose: PurposeInput}, false},
		{"target at the bound is kept", Intent{Target: MaxQuantity}, Intent{Target: MaxQuantity}, false},
		{"reserve at the bound is kept", Intent{Reserve: MaxQuantity}, Intent{Reserve: MaxQuantity}, false},
		{"target above the bound", Intent{Target: MaxQuantity + 1, Reserve: 2}, Intent{Reserve: 2}, true},
		{"reserve above the bound", Intent{Reserve: MaxQuantity + 1, Target: 2}, Intent{Target: 2}, true},
		{"negative target", Intent{Target: -1}, Intent{}, true},
		{"negative reserve", Intent{Reserve: -4}, Intent{}, true},
		{"negative unit cost", Intent{UnitCost: -9}, Intent{}, true},
		{"purpose above the set", Intent{Purpose: PurposeTool + 1}, Intent{Purpose: PurposeOther}, true},
		{"negative purpose", Intent{Purpose: -1}, Intent{Purpose: PurposeOther}, true},
		{"last purpose is kept", Intent{Purpose: PurposeTool}, Intent{Purpose: PurposeTool}, false},
		{"everything wrong at once", Intent{Target: -1, Reserve: MaxQuantity + 1, UnitCost: -1, Purpose: 9}, Intent{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := tt.in
			if got := normalise(&in); got != tt.wantChanged || in != tt.want {
				t.Errorf("normalise(%+v) = %v, intent %+v, want %v, %+v", tt.in, got, in, tt.wantChanged, tt.want)
			}
		})
	}
}

// TestOrderOf verifies that the intent's Limit is the order's Price and its
// Quantity the order's, with the agent's ID, the commodity and the side, and
// that the fields of the belief rule do not reach the order.
//
// Requirement: FR-AGT-025
func TestOrderOf(t *testing.T) {
	t.Parallel()
	a := &agent{id: 42}
	in := Intent{Commodity: 3, Side: Ask, Quantity: 8, Limit: 55, Target: 9, Reserve: 2, UnitCost: 7, Purpose: PurposeInput}
	want := Order{Agent: 42, Commodity: 3, Side: Ask, Price: 55, Quantity: 8}
	if got := orderOf(a, in); got != want {
		t.Errorf("orderOf(agent 42, %+v) = %+v, want %+v", in, got, want)
	}
}

// TestOutcomeStock verifies the Stock of an outcome view: the units held less
// those reserved for the upkeep, not below 0, below, at and above the reserve.
//
// Requirement: FR-AGT-049
func TestOutcomeStock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		held, reserve, want int64
	}{
		{"held below the reserve", 3, 5, 0},
		{"held equal to the reserve", 5, 5, 0},
		{"held above the reserve", 9, 5, 4},
		{"no reserve", 9, 0, 9},
		{"nothing held", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := outcomeStock(tt.held, tt.reserve); got != tt.want {
				t.Errorf("outcomeStock(%d, %d) = %d, want %d", tt.held, tt.reserve, got, tt.want)
			}
		})
	}
}

// TestViewScratch_OutcomeViewAndPlanAreReset verifies that the scratch hands
// out the same plan and outcome view for every call, emptied, with the belief
// group set on the outcome.
//
// Requirement: NFR-AGT-001
func TestViewScratch_OutcomeViewAndPlanAreReset(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	sc := h.Scratch()
	p := sc.tradePlan()
	p.Intents = append(p.Intents, Intent{Quantity: 1})
	p.Discards = append(p.Discards, Discard{Quantity: 1})
	if q := sc.tradePlan(); q != p || len(q.Intents) != 0 || len(q.Discards) != 0 {
		t.Errorf("tradePlan() = %+v (same plan %v), want the same plan, empty", *q, q == p)
	}
	params := beliefTunablesOf(h.ctx.bp)
	o := sc.outcomeView(params)
	o.Filled = 4
	if o2 := sc.outcomeView(params); o2 != o || o2.Filled != 0 || o2.Params != params {
		t.Errorf("outcomeView() = %+v (same view %v), want the same view, zeroed, with the belief group", *o2, o2 == o)
	}
}

// TestWorkReport_FloorChangedIsHeldByFloor verifies over a grid of ratios,
// working ticks and floors that the work report's floor flag equals
// welfare.HeldByFloor, the predicate the world counts (FR-AGT-072), with a
// MinWorkShare of 0 turning the floor off.
//
// Requirement: FR-AGT-072
func TestWorkReport_FloorChangedIsHeldByFloor(t *testing.T) {
	t.Parallel()
	const window, target, band = 10, 15_000, 500
	for _, present := range []bool{false, true} {
		for _, ratio := range []int64{9_000, 15_000, 15_200, 15_750, 20_000} {
			for _, working := range []int64{0, 3, 5, 10} {
				for _, floor := range []int64{0, 3_000, 5_000, 10_000} {
					name := fmt.Sprintf("present %v ratio %d working %d floor %d", present, ratio, working, floor)
					t.Run(name, func(t *testing.T) {
						t.Parallel()
						v := workView()
						v.HasRatio, v.Ratio, v.WorkingTicks, v.Window, v.MinWorkShare = present, ratio, working, window, floor
						v.TargetRatio, v.Tunables.RestBand = target, band
						var buf workReportBuf
						workReport(v, &buf)
						if want := welfare.HeldByFloor(ratio, present, working, window, floor, target, band); buf.floorChanged != want {
							t.Errorf("floorChanged = %v, want HeldByFloor %v", buf.floorChanged, want)
						}
					})
				}
			}
		}
	}
}

// TestWorkReport_AWritingStrategyDoesNotChangeIt verifies that the work report
// is a function of the view alone and that a strategy's writes to the view do
// not reach the world's state, so a consumer's strategy cannot hide the halt or
// the floor from the world's counts (D88). On one agent:
//
//  1. fill the view and take the report R1;
//  2. scribble every field of the view: the report taken from the scribbled
//     view differs from R1 (so the report really reads the view), while a
//     refill from the live state gives R2 equal to R1 (the writes did not reach
//     the live state);
//  3. change the live state (hunger) without refilling: the report taken again
//     from the same view still equals R1 (it reads the view, not the live
//     agent), and a refill does see the change.
//
// A report that read live state would fail step 2 (scribbled view, same report)
// and step 3 (live change, same view); one that read a slice shared with live
// state would fail the refill of step 2.
//
// Requirement: FR-AGT-024, FR-AGT-072
func TestWorkReport_AWritingStrategyDoesNotChangeIt(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "farmer", "north", 3)
	h.Fund(t, i, 5000)
	report := func(v *AgentView) workReportBuf {
		var b workReportBuf
		workReport(v, &b)
		b.skipped = append([]RecipeID(nil), b.skipped...)
		return b
	}
	v, err := h.Fill(i)
	if err != nil {
		t.Fatalf("fill error = %v, want nil", err)
	}
	r1 := report(v)
	scribble(reflect.ValueOf(v))
	if rs := report(v); reflect.DeepEqual(rs, r1) {
		t.Errorf("report from a scribbled view = %+v, same as R1: the report does not read the view", rs)
	}
	v, err = h.Fill(i)
	if err != nil {
		t.Fatalf("refill error = %v, want nil", err)
	}
	if r2 := report(v); !reflect.DeepEqual(r2, r1) {
		t.Errorf("report after a refill = %+v, want R1 %+v: the scribble reached the live state", r2, r1)
	}
	h.SetHunger(i, 4, 1)
	if r3 := report(v); !reflect.DeepEqual(r3, r1) {
		t.Errorf("report from the unchanged view after a live change = %+v, want R1 %+v: it read live state", r3, r1)
	}
	if fresh, err := h.Fill(i); err != nil || fresh.Hunger != 4 {
		t.Errorf("refill after the live change = Hunger %d, err %v, want 4, nil", fresh.Hunger, err)
	}
}

// TestDefaultStrategyWork_CapZeroStartsNoJob verifies, on the real production
// tables, the case the property test cannot reach by sampling: a baker with
// money but no oven or no flour has a cap of 0 on its only recipe, so the
// default Work requests nothing (NoJob; the report gives the production reason
// of the recipe that cannot run, no_tool then no_input, FR-AGT-037), and
// production, asked to start the recipe anyway as the world would after a
// faulty strategy's request, starts no execution. A burner, whose recipe has no
// inputs and no tool, requests its recipe and starts it, so the first half is
// not an artefact of the fixture.
//
// Requirement: FR-AGT-037, FR-AGT-028
func TestDefaultStrategyWork_CapZeroStartsNoJob(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	baker := h.Spawn(t, "baker", "north", 1)
	h.Fund(t, baker, 5000)
	bake, _ := h.Cfg.recipeID("bake")
	for _, step := range []struct {
		name   string
		give   func()
		reason Reason
	}{
		{"no oven, no flour", func() {}, ReasonNoTool},
		{"oven, no flour", func() { h.Give(t, baker, "oven", 1) }, ReasonNoInput},
	} {
		step.give()
		if k, _ := h.MaxExecutions(t, baker, "bake"); k != 0 {
			t.Fatalf("%s: cap of bake = %d, want 0", step.name, k)
		}
		v, err := h.Fill(baker)
		if err != nil {
			t.Fatalf("%s: fill error = %v, want nil", step.name, err)
		}
		var rep workReportBuf
		workReport(v, &rep)
		if req := (DefaultStrategy{}).Work(v, rand.New(rand.NewPCG(1, 2))); !req.NoJob || rep.reason != step.reason {
			t.Errorf("%s: Work = %+v, reason %q, want NoJob with %q", step.name, req, rep.reason, step.reason)
		}
		if k, reason := h.Start(t, baker, bake, 1); k != 0 || reason == "" {
			t.Errorf("%s: start with a cap of 0 = %d executions, reason %q, want 0 and a reason", step.name, k, reason)
		}
	}
	burner := h.Spawn(t, "burner", "north", 1)
	h.Fund(t, burner, 5000)
	v, err := h.Fill(burner)
	if err != nil {
		t.Fatalf("burner fill error = %v, want nil", err)
	}
	req := (DefaultStrategy{}).Work(v, rand.New(rand.NewPCG(1, 2)))
	if req.NoJob {
		t.Fatalf("burner Work = %+v, want a request", req)
	}
	if k, reason := h.Start(t, burner, req.Recipe, req.Executions); k < 1 {
		t.Errorf("burner start = %d executions, reason %q, want at least 1", k, reason)
	}
}

// TestPlan_ThroughTheRealAdmission verifies the pair of calls that connect a
// strategy to the market (FR-AGT-025, 026): an intent for a commodity outside
// the role's tradable set, converted by orderOf and admitted by the real book
// with Tradable from the role row, is refused not_tradable, and a second intent
// for one commodity and side is refused duplicate_order, while the first is
// admitted and the money it earmarks is the limit times the quantity.
//
// The clerk's tradable set is bread, grain and oven (flour is not in it); the
// first grain bid reserves 4*50 = 200 of the 1000.
//
// Requirement: FR-AGT-025, FR-AGT-026
func TestPlan_ThroughTheRealAdmission(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "clerk", "north", 1)
	h.Fund(t, i, 1000)
	flour, grain := h.commodity(t, "flour"), h.commodity(t, "grain")
	plan := &TradePlan{Intents: []Intent{
		{Commodity: grain, Side: Bid, Quantity: 4, Limit: 50},
		{Commodity: flour, Side: Bid, Quantity: 1, Limit: 10},
		{Commodity: grain, Side: Bid, Quantity: 1, Limit: 40},
		{Commodity: grain, Side: Ask, Quantity: 1, Limit: 40},
	}}
	res := h.Admit(t, i, plan)
	wantReasons := []Reason{"", ReasonNotTradable, ReasonDuplicateOrder, ReasonInsufficientStock}
	for k, r := range res {
		if r.Reason != wantReasons[k] || r.OK != (wantReasons[k] == "") {
			t.Errorf("intent %d (%+v): reason %q, admitted %v, want reason %q", k, plan.Intents[k], r.Reason, r.OK, wantReasons[k])
		}
	}
	if _, reserved := h.Balance(t, i); reserved != 200 {
		t.Errorf("reserved = %d, want 4*50 = 200", reserved)
	}
}

// BenchmarkFillCoreAndTrade reports the cost and allocations of the fill and of
// Trade across weights; the cost does not depend on the weight (NFR-AGT-002; a
// benchmark never fails CI, the allocation limit is
// TestFillCore_AllocatesNothingWarmed).
func BenchmarkFillCoreAndTrade(b *testing.B) {
	for _, w := range []int64{1, 7, MaxWeight} {
		b.Run(fmt.Sprintf("weight %d", w), func(b *testing.B) {
			h := NewViewHarness(b, viewConfig())
			i := h.Spawn(b, "baker", "south", w)
			h.Give(b, i, "flour", 20)
			h.Give(b, i, "oven", 1)
			h.Fund(b, i, 4000)
			rng := rand.New(rand.NewPCG(1, 2))
			b.Run("fill", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := h.Fill(i); err != nil {
						b.Fatal(err)
					}
				}
			})
			v, _ := h.Fill(i)
			b.Run("trade", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					DefaultStrategy{}.Trade(v, rng, h.Scratch().tradePlan())
				}
			})
		})
	}
}
