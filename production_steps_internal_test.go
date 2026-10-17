package economysim

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

type seedSpec struct {
	commodity string
	n         int64
	reserved  int64
}

// TestStart_Table verifies the count of a start over the reasons and limits of
// spec-production.md 4.2: the least of the weight, the maximum, and the caps of
// the yield pair, the tools and the inputs; the first applicable reason in the
// master order when it is 0; and that the read maxExecutions returns the same
// k0 and reason as the start that follows it, changing nothing. Available units
// are held - reserved (D58).
//
// Requirement: FR-PRO-009, FR-PRO-010, FR-PRO-011, FR-PRO-013
//
// Guard: FR-PRO-009, a read that drifts from the start would show the strategy
// executions the start then refuses, and a refused start that consumes
// anything breaks the conservation of the ledger.
func TestStart_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		recipe     string
		region     string
		season     int
		weight     int64
		maximum    int64
		seeds      []seedSpec
		shockFarm0 bool // set the yield shock of farm in the north to 0
		wantK      int64
		wantReason Reason
	}{
		// bake: grain 1 and flour 2 per execution, oven as the tool.
		{name: "least of the stocks", recipe: "bake", region: "north", weight: 5,
			seeds: []seedSpec{{"grain", 10, 0}, {"flour", 6, 0}, {"oven", 5, 0}}, wantK: 3}, // min(5, 10/1, 6/2, 5)
		{name: "weight binds", recipe: "bake", region: "north", weight: 2,
			seeds: []seedSpec{{"grain", 10, 0}, {"flour", 20, 0}, {"oven", 5, 0}}, wantK: 2},
		{name: "strategy maximum binds", recipe: "bake", region: "north", weight: 5, maximum: 1,
			seeds: []seedSpec{{"grain", 10, 0}, {"flour", 20, 0}, {"oven", 5, 0}}, wantK: 1},
		{name: "maximum above weight changes nothing", recipe: "bake", region: "north", weight: 2, maximum: 9,
			seeds: []seedSpec{{"grain", 10, 0}, {"flour", 20, 0}, {"oven", 5, 0}}, wantK: 2},
		{name: "reserved inputs do not count", recipe: "bake", region: "north", weight: 10,
			seeds: []seedSpec{{"grain", 5, 2}, {"flour", 100, 0}, {"oven", 100, 0}}, wantK: 3}, // (5 - 2) / 1, not 5
		{name: "reserved tool items do not count", recipe: "bake", region: "north", weight: 10,
			seeds: []seedSpec{{"grain", 10, 0}, {"flour", 20, 0}, {"oven", 4, 1}}, wantK: 3}, // 4 - 1 items
		{name: "source recipe in a sited region", recipe: "farm", region: "north", season: spring, weight: 4, wantK: 4},
		{name: "tool-only recipe", recipe: "polish", region: "north", weight: 5,
			seeds: []seedSpec{{"peel", 2, 0}}, wantK: 2},
		{name: "commodity both input and tool", recipe: "rework", region: "north", weight: 10,
			seeds: []seedSpec{{"oven", 7, 0}}, wantK: 2}, // 7 / (2 + 1), not 7 / 2
		{name: "no input", recipe: "bake", region: "north", weight: 5,
			seeds: []seedSpec{{"flour", 6, 0}, {"oven", 5, 0}}, wantReason: ReasonNoInput},
		{name: "reserved input leaves none", recipe: "bake", region: "north", weight: 5,
			seeds: []seedSpec{{"grain", 4, 4}, {"flour", 6, 0}, {"oven", 5, 0}}, wantReason: ReasonNoInput},
		{name: "no tool", recipe: "bake", region: "north", weight: 5,
			seeds: []seedSpec{{"grain", 10, 0}, {"flour", 6, 0}}, wantReason: ReasonNoTool},
		{name: "tool and input both missing is no_tool first", recipe: "bake", region: "north", weight: 5,
			wantReason: ReasonNoTool},
		{name: "commodity both input and tool, none held", recipe: "rework", region: "north", weight: 5,
			wantReason: ReasonNoTool},
		{name: "no profile of a sited recipe", recipe: "farm", region: "arid", season: spring, weight: 4, wantReason: ReasonNoSite},
		{name: "profile without the season", recipe: "farm", region: "north", season: summer, weight: 4, wantReason: ReasonZeroYield},
		{name: "profile of all = 0", recipe: "farm", region: "crater", season: spring, weight: 4, wantReason: ReasonZeroYield},
		{name: "shock of 0", recipe: "farm", region: "north", season: spring, weight: 4, shockFarm0: true, wantReason: ReasonZeroYield},
		{name: "weight 0 starts nothing and has no reason", recipe: "bake", region: "north", weight: 0,
			seeds: []seedSpec{{"grain", 10, 0}, {"flour", 20, 0}, {"oven", 5, 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newStepFixture(t)
			for _, s := range tt.seeds {
				f.seed(s.commodity, s.n)
				if s.reserved > 0 {
					f.reserve(s.commodity, s.reserved)
				}
			}
			if tt.shockFarm0 {
				if err := f.sh.set(f.region("north"), f.recipe("farm"), 0); err != nil {
					t.Fatalf("set shock: %v", err)
				}
			}
			q := f.req(tt.recipe, tt.region, tt.season, tt.weight, tt.maximum)
			before, residBefore := f.state(), slices.Clone(f.resid)

			k0, reason, err := f.s.maxExecutions(f.l, q)
			if err != nil || k0 != tt.wantK || reason != tt.wantReason {
				t.Errorf("maxExecutions = (%d, %q, %v), want (%d, %q, nil)", k0, reason, err, tt.wantK, tt.wantReason)
			}
			if got := f.state(); got != before || !slices.Equal(f.resid, residBefore) || f.job.executions != 0 {
				t.Errorf("maxExecutions changed the ledger, the residues or the job")
			}

			res, err := f.start(q)
			if err != nil || res.k != tt.wantK || res.reason != tt.wantReason {
				t.Fatalf("start = (%+v, %v), want k %d, reason %q, nil", res, err, tt.wantK, tt.wantReason)
			}
			if tt.wantK == 0 {
				if got := f.state(); got != before || !slices.Equal(f.resid, residBefore) || f.job.executions != 0 {
					t.Errorf("a refused start changed the ledger, the residues or the job")
				}
				return
			}
			if f.job.executions != tt.wantK || f.job.recipe != q.recipe || f.job.principal != 7 || f.job.yield != q.effective {
				t.Errorf("job = %+v, want recipe %d, executions %d, principal 7, yield %d", f.job, q.recipe, tt.wantK, q.effective)
			}
			if res.effective != q.effective {
				t.Errorf("start effective = %d, want %d", res.effective, q.effective)
			}
			// Earmarked units are never taken or worn.
			for _, s := range tt.seeds {
				if _, reserved := f.held(s.commodity); reserved != s.reserved {
					t.Errorf("%s reserved after start = %d, want %d", s.commodity, reserved, s.reserved)
				}
			}
		})
	}
}

// TestStart_LedgerEffects verifies what a start of bake at weight 5 over a
// stock for 3 executions does to the ledger: 3 times each input moves to
// recipe_input, three distinct oven items are worn by 5 each (none twice), and
// the pending output, the wear facts and the inputs spent are what the facts
// of D51 say.
//
// Requirement: FR-PRO-010, FR-PRO-011, FR-PRO-013
func TestStart_LedgerEffects(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	f.seed("grain", 10)
	f.seed("flour", 6)
	f.seed("oven", 5)
	res, err := f.start(f.req("bake", "north", spring, 5, 0))
	if err != nil || res.k != 3 {
		t.Fatalf("start = (%+v, %v), want k 3", res, err)
	}
	if g, _ := f.held("grain"); g != 7 { // 10 - 3 * 1
		t.Errorf("grain held = %d, want 7", g)
	}
	if fl, _ := f.held("flour"); fl != 0 { // 6 - 3 * 2
		t.Errorf("flour held = %d, want 0", fl)
	}
	if got := f.flow(ledger.RecipeInput, "grain"); got != 3 {
		t.Errorf("recipe_input grain = %d, want 3", got)
	}
	if got := f.flow(ledger.RecipeInput, "flour"); got != 6 {
		t.Errorf("recipe_input flour = %d, want 6", got)
	}
	// Five new ovens at durability 20; three worn by 5 leave 3 at 15 and 2 at 20.
	lots, err := f.l.AppendLots(nil, f.acct)
	if err != nil {
		t.Fatalf("AppendLots: %v", err)
	}
	var ovens []ledger.Lot
	for _, lt := range lots {
		if lt.Commodity == int(f.com("oven")) {
			ovens = append(ovens, lt)
		}
	}
	if len(ovens) != 2 || ovens[0].Durability != 15 || ovens[0].Quantity != 3 || ovens[1].Durability != 20 || ovens[1].Quantity != 2 {
		t.Errorf("oven lots = %+v, want 3 at 15 and 2 at 20", ovens)
	}
	if want := []wearFact{{f.com("oven"), 3, 5}}; !slices.Equal(f.s.wear, want) {
		t.Errorf("wear facts = %+v, want %+v", f.s.wear, want)
	}
	wantSpent := []Amount{{f.com("flour"), 6}, {f.com("grain"), 3}} // commodity order
	if !slices.Equal(f.s.spent, wantSpent) {
		t.Errorf("inputs spent = %+v, want %+v", f.s.spent, wantSpent)
	}
	// Outputs are in commodity order: ash 3 * 1 = 3 and bread 3 * 3 = 9, at the neutral factor.
	if want := []int64{3, 9}; !slices.Equal(f.job.pending, want) || f.job.remaining != 1 {
		t.Errorf("job pending = %v, remaining %d, want %v, 1", f.job.pending, f.job.remaining, want)
	}
}

// TestStart_ToolDestroyedAtDurability verifies that a tool at durability equal
// to WearPerUse is destroyed to the wear boundary by the start (polish wears a
// peel of durability 7 by 7), that a tool-only recipe makes no input Transform
// and that a source recipe creates from nothing.
//
// Requirement: FR-PRO-011, FR-PRO-003
func TestStart_ToolDestroyedAtDurability(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	f.seed("peel", 2)
	res, err := f.start(f.req("polish", "north", spring, 2, 0))
	if err != nil || res.k != 2 {
		t.Fatalf("start = (%+v, %v), want k 2", res, err)
	}
	if h, _ := f.held("peel"); h != 0 {
		t.Errorf("peel held = %d, want 0 (both destroyed)", h)
	}
	if got := f.flow(ledger.Wear, "peel"); got != 2 {
		t.Errorf("wear flow of peel = %d, want 2", got)
	}
	for _, name := range []string{"ash", "grain", "peel"} {
		if got := f.flow(ledger.RecipeInput, name); got != 0 {
			t.Errorf("recipe_input %s = %d, want 0 (no input Transform)", name, got)
		}
	}
	// A source recipe: no input and no tool, nothing in the ledger changes.
	g := newStepFixture(t)
	before := g.state()
	if res, err := g.start(g.req("farm", "north", spring, 4, 0)); err != nil || res.k != 4 {
		t.Fatalf("farm start = (%+v, %v), want k 4", res, err)
	}
	if g.state() != before {
		t.Errorf("a source start changed the ledger")
	}
	// grain 4 * 6 = 24 and slag 4 * 2 = 8 at 10 000.
	if want := []int64{24, 8}; !slices.Equal(g.job.pending, want) {
		t.Errorf("farm pending = %v, want %v", g.job.pending, want)
	}
}

// TestStart_LedgerRefusal verifies that an input Transform the ledger refuses
// is returned with the job unstarted, the residues untouched and the ledger
// unchanged. The ledger is built with the recipe_input boundary account opened
// for every commodity but flour, so Transform gives ErrUnknownAccount.
//
// Requirement: FR-PRO-010
func TestStart_LedgerRefusal(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	cfg := *f.cfg
	l, err := ledger.New(ledger.Config{
		Commodities: []ledger.Commodity{{}, {ShelfLife: 5}, {}, {}, {}, {Durability: 20}, {Durability: 7}},
		Regions:     len(cfg.Regions), MaxMoney: MaxMoney, MaxQuantity: MaxQuantity,
	})
	if err != nil {
		t.Fatalf("ledger.New: %v", err)
	}
	flour := f.com("flour")
	for thing := ledger.Money; thing < len(cfg.Commodities); thing++ {
		for r := ledger.Reason(0); int(r) < numBuiltInReasons; r++ {
			if class, ok := ledger.ClassOf(r, thing != ledger.Money); !ok || class == ledger.Internal {
				continue
			}
			if r == ledger.RecipeInput && thing == int(flour) {
				continue
			}
			if _, err := l.OpenBoundary(r, thing); err != nil {
				t.Fatalf("OpenBoundary(%d, %d): %v", r, thing, err)
			}
		}
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	f.l = l
	if f.acct, err = l.Open(ledger.KindAgent, 0, 0); err != nil {
		t.Fatalf("Open: %v", err)
	}
	f.seed("grain", 10)
	f.seed("flour", 6)
	f.seed("oven", 5)
	before, residBefore := f.state(), slices.Clone(f.resid)
	res, err := f.start(f.req("bake", "north", spring, 5, 0))
	if !errors.Is(err, ledger.ErrUnknownAccount) || res.k != 0 {
		t.Fatalf("start = (%+v, %v), want k 0 and ErrUnknownAccount", res, err)
	}
	if f.job.executions != 0 || len(f.job.pending) != 0 || !slices.Equal(f.resid, residBefore) || f.state() != before {
		t.Errorf("a refused Transform changed the job %+v, the residues or the ledger", f.job)
	}
}

// TestStart_NoPartialStateOnOverflow verifies that a pending output that does
// not fit int64 refuses the start with ErrOverflow before the ledger is
// touched. The configuration is not validated, so the quantity can exceed the
// bound a pack could carry.
//
// Requirement: FR-PRO-010
func TestStart_NoPartialStateOnOverflow(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	f.cfg.Recipes[f.recipe("scrap")].Outputs[0].Quantity = 1 << 62
	f.rt = newRecipeTable(f.cfg)
	f.s = newProductionScratch(f.cfg, &f.rt)
	before := f.state()
	// 4 * 2^62 does not fit.
	res, err := f.start(f.req("scrap", "north", spring, 4, 0))
	if !errors.Is(err, ErrOverflow) || res.k != 0 || f.job.executions != 0 || f.state() != before {
		t.Errorf("start = (%+v, %v), want ErrOverflow, nothing changed", res, err)
	}
}

// TestSteps_Panics verifies the preconditions of the steps, each a violated
// invariant: a start for an agent that holds a job, an advance with no job, a
// completion before the end, and residues that do not match the outputs.
//
// Requirement: FR-PRO-013, FR-PRO-014
func TestSteps_Panics(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	q := f.req("farm", "north", spring, 1, 0)
	mustPanicInvariant(t, "advance without a job", func() { f.job.advance() })
	mustPanicInvariant(t, "complete without a job", func() { f.s.complete(f.l, &f.job, f.acct, MaxQuantity) })
	mustPanicInvariant(t, "residues of the wrong length", func() { f.s.start(f.l, &f.job, nil, q) })
	if _, err := f.start(q); err != nil {
		t.Fatalf("start: %v", err)
	}
	mustPanicInvariant(t, "start with a job in progress", func() { f.start(q) })
	mustPanicInvariant(t, "complete before the end", func() { f.s.complete(f.l, &f.job, f.acct, MaxQuantity) })
}

// TestTiming verifies the tick arithmetic of spec-production.md 5.2: a
// duration-1 job started and advanced in one tick completes in it; a job of
// duration 3 started in tick 10 is done at the advance of tick 12 and has
// created nothing in 10 and 11; and the pending output is fixed at the start:
// a farm job started in spring (factor 10 000) and completed after a summer
// (site factor 0) produces the quantity of the start, while a start made in
// summer is zero_yield and one in the arid region is no_site.
//
// Requirement: FR-PRO-013, FR-PRO-014, FR-PRO-021
func TestTiming(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	if res, err := f.start(f.req("scrap", "north", spring, 1, 0)); err != nil || res.k != 1 {
		t.Fatalf("scrap start = (%+v, %v)", res, err)
	}
	if done, _ := f.job.advance(); !done {
		t.Errorf("duration-1 job after the start-tick advance: done = false, want true")
	}
	f.s.complete(f.l, &f.job, f.acct, MaxQuantity)

	if res, err := f.start(f.req("farm", "north", spring, 1, 0)); err != nil || res.k != 1 {
		t.Fatalf("farm start (tick 10) = (%+v, %v)", res, err)
	}
	for tick := int64(10); tick <= 12; tick++ {
		done, _ := f.job.advance()
		if want := tick == 12; done != want {
			t.Errorf("tick %d: done = %v, want %v", tick, done, want)
		}
		if g, _ := f.held("grain"); g != 0 && tick < 12 {
			t.Errorf("tick %d: grain held = %d, want 0 before completion", tick, g)
		}
	}
	// Summer at completion: the site factor is 0 now, the pending was fixed in spring.
	if k0, _, err := f.s.maxExecutions(f.l, f.req("farm", "north", summer, 1, 0)); k0 != 0 || err != nil {
		t.Errorf("a farm start in summer has k0 %d, err %v, want 0", k0, err)
	}
	f.s.complete(f.l, &f.job, f.acct, MaxQuantity)
	if g, _ := f.held("grain"); g != 6 { // 1 * 6 * 10 000 / 10 000
		t.Errorf("grain after completion = %d, want 6", g)
	}
	for _, c := range []struct {
		region string
		season int
		want   Reason
	}{{"north", summer, ReasonZeroYield}, {"arid", spring, ReasonNoSite}} {
		if _, got, _ := f.s.maxExecutions(f.l, f.req("farm", c.region, c.season, 1, 0)); got != c.want {
			t.Errorf("farm start in %s season %d: reason %q, want %q", c.region, c.season, got, c.want)
		}
	}
}

// TestComplete_BornAndFraction verifies that a completion creates a spoiling
// output born in the completing tick and a durable output at full durability
// (FR-PRO-012), and that the fractional yield of spec-production.md 6.6 over
// four jobs of fract (bread 5 at a shock of 5 000) produces 2, 3, 2, 3 units:
// 5 * 5 000 / 10 000 = 2 remainder 5 000, then 2 + the carried 5 000 gives 3.
//
// Requirement: FR-PRO-012, FR-PRO-024
func TestComplete_BornAndFraction(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	if err := f.sh.set(f.region("north"), f.recipe("fract"), 5_000); err != nil {
		t.Fatalf("set shock: %v", err)
	}
	for range 7 {
		f.l.AdvanceTick()
	}
	var got []int64
	for range 4 {
		if res, err := f.start(f.req("fract", "north", spring, 1, 0)); err != nil || res.k != 1 {
			t.Fatalf("fract start = (%+v, %v)", res, err)
		}
		got = append(got, f.job.pending[0])
		f.finish()
	}
	if want := []int64{2, 3, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("units of four jobs = %v, want %v", got, want)
	}
	lots, err := f.l.AppendLots(nil, f.acct)
	if err != nil {
		t.Fatalf("AppendLots: %v", err)
	}
	if len(lots) != 1 || lots[0].Commodity != int(f.com("bread")) || lots[0].Quantity != 10 || lots[0].Born != 7 {
		t.Errorf("lots = %+v, want one lot of 10 bread born in tick 7", lots)
	}
	if _, err := f.start(f.req("forge", "north", spring, 1, 0)); err != nil {
		t.Fatalf("forge start: %v", err)
	}
	f.finish()
	lots, _ = f.l.AppendLots(nil, f.acct)
	var oven *ledger.Lot
	for i := range lots {
		if lots[i].Commodity == int(f.com("oven")) {
			oven = &lots[i]
		}
	}
	if oven == nil || oven.Quantity != 2 || oven.Durability != 20 {
		t.Errorf("oven lot = %+v, want 2 items at durability 20", oven)
	}
}

// TestComplete_Values verifies the values of Outputs and Disposed and the flow
// totals (D60, D54, D58) for six cases, each with its arithmetic. farm in the
// north in spring at weight 1 fixes grain 6 and a slag discard of 2.
//
//	(a) room for all: fit 6; Outputs [grain 6, slag 2], Disposed [slag 2];
//	    recipe_output 6 + 2 = 8, disposal 2.
//	(b) held MaxQuantity - 4: fit 4; Disposed [grain 2, slag 2]; held MaxQuantity.
//	(c) held MaxQuantity: fit 0, no Transform; Disposed [grain 6, slag 2].
//	(d) MaxQuantity - 2 held, all earmarked: available 0 but room 2; fit 2;
//	    Disposed [grain 4, slag 2] (a room of MaxQuantity - (held - reserved)
//	    would offer 6 and Transform would refuse).
//	(e) scrap, a discard output only: no Transform; both slices [slag 2].
//	Overflow is Disposed less the discard output slag: (b) grain 2, (c) grain 6,
//	(d) grain 4, and none in (a), (e) and (f).
//	(f) trace at a shock of 5 000 and residue 0: 1 * 1 * 5 000 / 10 000 = 0 of
//	    bread and of slag; no row in either slice and no ledger call.
//
// Requirement: FR-PRO-029, FR-PRO-030, FR-PRO-031, FR-PRO-041, FR-PRO-053
func TestComplete_Values(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		recipe       string
		preheld      int64 // grain held before the completion, as MaxQuantity - preheld; -1 for none
		earmarkAll   bool
		shock        int64
		wantOut      []Amount
		wantDisposed []Amount
		wantOverflow []Amount
		wantGrain    int64
		wantRecOut   int64
		wantDisposal int64
	}{
		{name: "a: room for all", recipe: "farm", preheld: -1,
			wantOut: []Amount{{3, 6}, {6, 2}}, wantDisposed: []Amount{{6, 2}}, wantOverflow: nil, wantGrain: 6, wantRecOut: 8, wantDisposal: 2},
		{name: "b: partial room", recipe: "farm", preheld: 4,
			wantOut: []Amount{{3, 6}, {6, 2}}, wantDisposed: []Amount{{3, 2}, {6, 2}}, wantOverflow: []Amount{{3, 2}}, wantGrain: MaxQuantity, wantRecOut: 4 + 2 + 2, wantDisposal: 4},
		{name: "c: no room", recipe: "farm", preheld: 0,
			wantOut: []Amount{{3, 6}, {6, 2}}, wantDisposed: []Amount{{3, 6}, {6, 2}}, wantOverflow: []Amount{{3, 6}}, wantGrain: MaxQuantity, wantRecOut: 8, wantDisposal: 8},
		{name: "d: reserved units count against the room", recipe: "farm", preheld: 2, earmarkAll: true,
			wantOut: []Amount{{3, 6}, {6, 2}}, wantDisposed: []Amount{{3, 4}, {6, 2}}, wantOverflow: []Amount{{3, 4}}, wantGrain: MaxQuantity, wantRecOut: 2 + 2 + 4, wantDisposal: 6},
		{name: "e: discard output only", recipe: "scrap", preheld: -1,
			wantOut: []Amount{{6, 2}}, wantDisposed: []Amount{{6, 2}}, wantGrain: 0, wantRecOut: 2, wantDisposal: 2},
		{name: "f: zero quantities are no row", recipe: "trace", preheld: -1, shock: 5_000,
			wantOut: nil, wantDisposed: nil, wantGrain: 0, wantRecOut: 0, wantDisposal: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newStepFixture(t)
			if tt.preheld >= 0 {
				f.seed("grain", MaxQuantity-tt.preheld)
				if tt.earmarkAll {
					f.reserve("grain", MaxQuantity-tt.preheld)
				}
			}
			if tt.shock != 0 {
				if err := f.sh.set(f.region("north"), f.recipe(tt.recipe), tt.shock); err != nil {
					t.Fatalf("set shock: %v", err)
				}
			}
			if res, err := f.start(f.req(tt.recipe, "north", spring, 1, 0)); err != nil || res.k != 1 {
				t.Fatalf("start = (%+v, %v)", res, err)
			}
			base := f.flow(ledger.RecipeOutput, "grain") + f.flow(ledger.RecipeOutput, "slag") + f.flow(ledger.RecipeOutput, "bread")
			c := f.finish()
			ev := f.s.recipeExecuted(nil, nil, nil, 12, 7, 2, &c, f.cfg.Recipes[c.recipe].Duration)
			if !slices.Equal(ev.Overflow, tt.wantOverflow) {
				t.Errorf("Overflow = %v, want %v", ev.Overflow, tt.wantOverflow)
			}
			if !slices.Equal(ev.Outputs, tt.wantOut) || !slices.Equal(ev.Disposed, tt.wantDisposed) {
				t.Errorf("Outputs, Disposed = %v, %v, want %v, %v", ev.Outputs, ev.Disposed, tt.wantOut, tt.wantDisposed)
			}
			if g, _ := f.held("grain"); g != tt.wantGrain && tt.recipe == "farm" {
				t.Errorf("grain held = %d, want %d", g, tt.wantGrain)
			}
			gotOut := f.flow(ledger.RecipeOutput, "grain") + f.flow(ledger.RecipeOutput, "slag") + f.flow(ledger.RecipeOutput, "bread") - base
			// (b) also seeded MaxQuantity - 4 under Initial, which is not recipe_output.
			if gotOut != tt.wantRecOut {
				t.Errorf("recipe_output total = %d, want %d", gotOut, tt.wantRecOut)
			}
			disp := f.flow(ledger.Disposal, "grain") + f.flow(ledger.Disposal, "slag") + f.flow(ledger.Disposal, "bread")
			if disp != tt.wantDisposal {
				t.Errorf("disposal total = %d, want %d", disp, tt.wantDisposal)
			}
			// The slag never appears in an account.
			if s, _ := f.held("slag"); s != 0 {
				t.Errorf("slag held = %d, want 0", s)
			}
			if err := f.l.Audit(); err != nil {
				t.Errorf("Audit: %v", err)
			}
		})
	}
}

// TestComplete_NoOutputAndSameCommodityRoom verifies that a recipe with no
// output makes no Transform at completion, and that two outputs of one
// commodity share the room: the second line is cut by what the first took.
//
// Requirement: FR-PRO-029, FR-PRO-030
func TestComplete_NoOutputAndSameCommodityRoom(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	f.seed("grain", 4)
	if res, err := f.start(f.req("burn", "north", spring, 2, 0)); err != nil || res.k != 2 {
		t.Fatalf("burn start = (%+v, %v)", res, err)
	}
	before := f.state()
	c := f.finish()
	if f.state() != before || len(c.created) != 0 {
		t.Errorf("a completion with no output changed the ledger or created %v", c.created)
	}

	g := newStepFixture(t)
	r := &g.cfg.Recipes[g.recipe("fract")]
	r.Outputs = []Output{{Commodity: "bread", Quantity: 6, CostShare: 5_000}, {Commodity: "bread", Quantity: 6, CostShare: 5_000}}
	g.rt = newRecipeTable(g.cfg)
	g.s = newProductionScratch(g.cfg, &g.rt)
	g.all = g.all[:0]
	for i := range g.cfg.Recipes {
		g.all = append(g.all, RecipeID(i))
	}
	g.resid = make([]int64, residueLen(g.all, &g.rt))
	g.seed("bread", MaxQuantity-8)
	if res, err := g.start(g.req("fract", "north", spring, 1, 0)); err != nil || res.k != 1 {
		t.Fatalf("start = (%+v, %v)", res, err)
	}
	c = g.finish()
	// Room 8: the first line takes 6, the second 2; 4 are disposed.
	if want := []int64{0, 4}; !slices.Equal(c.disposed, want) {
		t.Errorf("disposed = %v, want %v", c.disposed, want)
	}
	if h, _ := g.held("bread"); h != MaxQuantity {
		t.Errorf("bread held = %d, want MaxQuantity", h)
	}
}

// TestEventValues_StartTickAndYield verifies D67: a farm job of duration 3
// started in tick 10 in spring and completed in tick 12, with the shock set to
// 3 000 in tick 11 and the summer's site factor 0 at completion, reports
// StartTick 10 (12 - 3 + 1) and EffectiveYield 10 000; a duration-1 job
// reports the tick; a weight fall does not move StartTick; the job record
// copied before completion (a restore) reports the same pair; and a warm-start
// job of remaining 2 of duration 3 completing in tick 1 reports StartTick -1
// and the factor of the virtual season (south, summer, 6 000).
//
// Requirement: FR-PRO-021, FR-PRO-041
func TestEventValues_StartTickAndYield(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	if res, err := f.start(f.req("farm", "north", spring, 1, 0)); err != nil || res.k != 1 {
		t.Fatalf("start = (%+v, %v)", res, err)
	}
	restored := f.job
	restored.pending = slices.Clone(f.job.pending)
	f.job.advance() // tick 10
	if err := f.sh.set(f.region("north"), f.recipe("farm"), 3_000); err != nil {
		t.Fatal(err)
	}
	f.job.advance() // tick 11
	f.job.advance() // tick 12
	c := f.s.complete(f.l, &f.job, f.acct, MaxQuantity)
	ev := f.s.recipeExecuted(nil, nil, nil, 12, 7, 2, &c, 3)
	if ev.StartTick != 10 || ev.EffectiveYield != 10_000 || ev.Executions != 1 || ev.Tick != 12 || ev.Agent != 7 || ev.Region != 2 || ev.Recipe != f.recipe("farm") {
		t.Errorf("event = %+v, want StartTick 10, EffectiveYield 10000, executions 1", ev)
	}

	g := newStepFixture(t)
	g.job = restored
	g.job.remaining = 0
	c2 := g.s.complete(g.l, &g.job, g.acct, MaxQuantity)
	ev2 := g.s.recipeExecuted(nil, nil, nil, 12, 7, 2, &c2, 3)
	if !reflect.DeepEqual(ev, ev2) {
		t.Errorf("restored job event = %+v, want %+v", ev2, ev)
	}

	h := newStepFixture(t)
	if _, err := h.start(h.req("scrap", "north", spring, 1, 0)); err != nil {
		t.Fatal(err)
	}
	c3 := h.finish()
	if ev := h.s.recipeExecuted(nil, nil, nil, 5, 7, 2, &c3, 1); ev.StartTick != 5 {
		t.Errorf("duration-1 StartTick = %d, want 5", ev.StartTick)
	}

	w := newStepFixture(t)
	if _, err := w.start(w.req("farm", "north", spring, 4, 0)); err != nil {
		t.Fatal(err)
	}
	if k := w.job.reweigh(2); k != 2 {
		t.Fatalf("reweigh = %d, want 2", k)
	}
	w.finish()
	// The pending was scaled 24 * 2 / 4 = 12 and 8 * 2 / 4 = 4.
	if g, _ := w.held("grain"); g != 12 {
		t.Errorf("grain after the weight fall = %d, want 12", g)
	}

	m := newStepFixture(t)
	if err := m.s.warmJob(&m.job, &m.yt, m.region("south"), summer, m.recipe("farm"), 7, 1, 2); err != nil {
		t.Fatalf("warmJob: %v", err)
	}
	m.job.advance() // tick 0
	done, _ := m.job.advance()
	if !done {
		t.Fatalf("warm job not done at tick 1")
	}
	c5 := m.s.complete(m.l, &m.job, m.acct, MaxQuantity)
	ev5 := m.s.recipeExecuted(nil, nil, nil, 1, 7, 3, &c5, 3)
	// 1 * 6 * 6 000 / 10 000 = 3 grain, 1 * 2 * 6 000 / 10 000 = 1 slag.
	if ev5.StartTick != -1 || ev5.EffectiveYield != 6_000 || !slices.Equal(ev5.Outputs, []Amount{{3, 3}, {6, 1}}) {
		t.Errorf("warm event = %+v, want StartTick -1, EffectiveYield 6000, outputs [grain 3, slag 1]", ev5)
	}
}

// TestWorkingExecutions verifies the executions that are a working
// household-tick (FR-PRO-015): the start returns k, a refused start 0, the
// advance and the completion return the executions the job holds, and a weight
// fall to 2 makes the later calls return 2 and not the 3 of the start.
//
// Requirement: FR-PRO-015
func TestWorkingExecutions(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	f.seed("grain", 9)
	if res, err := f.start(f.req("bake", "north", spring, 5, 0)); err != nil || res.k != 0 || res.reason != ReasonNoTool {
		t.Fatalf("refused start = (%+v, %v), want k 0, no_tool", res, err)
	}
	f.seed("flour", 6)
	f.seed("oven", 5)
	// Duration 1: start 3, advance 3, complete 3.
	res, err := f.start(f.req("bake", "north", spring, 5, 0))
	if err != nil || res.k != 3 {
		t.Fatalf("start = (%+v, %v), want k 3", res, err)
	}
	done, working := f.job.advance()
	c := f.s.complete(f.l, &f.job, f.acct, MaxQuantity)
	if !done || working != 3 || c.executions != 3 {
		t.Errorf("duration 1: advance (%v, %d), complete %d, want (true, 3), 3", done, working, c.executions)
	}

	// Duration 3 with a weight fall: mill k = 3 over 9 grain.
	f.seed("grain", 9)
	if res, err := f.start(f.req("mill", "north", spring, 3, 0)); err != nil || res.k != 3 {
		t.Fatalf("mill start = (%+v, %v), want k 3", res, err)
	}
	if _, w := f.job.advance(); w != 3 {
		t.Errorf("advance working = %d, want 3", w)
	}
	if k := f.job.reweigh(2); k != 2 {
		t.Errorf("reweigh = %d, want 2", k)
	}
	if _, w := f.job.advance(); w != 2 {
		t.Errorf("advance after the fall working = %d, want 2", w)
	}
	f.job.advance()
	if c := f.s.complete(f.l, &f.job, f.acct, MaxQuantity); c.executions != 2 {
		t.Errorf("complete executions = %d, want 2", c.executions)
	}
}

// TestAbandonAndReweigh verifies that an abandoned job moves nothing, creates
// no output and returns the recipe and executions; that a weight fall scales
// the pending output as 5.4 says (mill k 3: flour 6 and ash 3 (output order ash, flour) become 6 * 2 / 3
// = 4 and 3 * 2 / 3 = 2) and the completion creates the scaled amount; that a
// rise changes nothing; and that a fall to 0 ends the job.
//
// Requirement: FR-PRO-018, FR-PRO-019
func TestAbandonAndReweigh(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	f.seed("grain", 9)
	if _, err := f.start(f.req("mill", "north", spring, 3, 0)); err != nil {
		t.Fatal(err)
	}
	if want := []int64{3, 6}; !slices.Equal(f.job.pending, want) { // ash, flour
		t.Fatalf("pending = %v, want %v", f.job.pending, want)
	}
	if k := f.job.reweigh(5); k != 3 || !slices.Equal(f.job.pending, []int64{3, 6}) {
		t.Errorf("a rise gave %d, %v, want 3, [3 6]", k, f.job.pending)
	}
	if k := f.job.reweigh(2); k != 2 || !slices.Equal(f.job.pending, []int64{2, 4}) {
		t.Errorf("a fall to 2 gave %d, %v, want 2, [2 4]", k, f.job.pending)
	}
	f.finish()
	if fl, _ := f.held("flour"); fl != 4 {
		t.Errorf("flour after the scaled completion = %d, want 4", fl)
	}

	f.seed("grain", 9)
	if res, err := f.start(f.req("mill", "north", spring, 3, 0)); err != nil || res.k != 3 {
		t.Fatalf("second mill start = (%+v, %v), want k 3", res, err)
	}
	before := f.state()
	recipe, k := f.job.abandon()
	if recipe != f.recipe("mill") || k != 3 || f.job.executions != 0 || len(f.job.pending) != 0 || f.state() != before {
		t.Errorf("abandon = (%d, %d), job %+v, want mill, 3, cleared, ledger unchanged", recipe, k, f.job)
	}
	if got := f.flow(ledger.RecipeInput, "grain"); got != 18 { // two starts of 9
		t.Errorf("recipe_input grain = %d, want 18 (the inputs stay in recipe_input)", got)
	}
	if r, k := f.job.abandon(); r != f.recipe("mill") || k != 0 {
		t.Errorf("abandon of a cleared job = (%d, %d), want the stale recipe and 0", r, k)
	}
	if k := f.job.reweigh(1); k != 0 {
		t.Errorf("reweigh of a cleared job = %d, want 0", k)
	}
	if _, err := f.start(f.req("farm", "north", spring, 2, 0)); err != nil {
		t.Fatal(err)
	}
	if k := f.job.reweigh(0); k != 0 || f.job.executions != 0 || len(f.job.pending) != 0 {
		t.Errorf("a fall to 0 gave %d, job %+v, want the job ended", k, f.job)
	}
	mustPanicInvariant(t, "reweigh to a negative weight", func() {
		f.start(f.req("farm", "north", spring, 2, 0))
		f.job.reweigh(-1)
	})
}

// TestWarmJob verifies that the warm-start job fixes its pending output as
// floor(k * q * E / 10 000) with the factor of the virtual season, the shock at
// its default and residue 0 (3 * 6 = 18 grain and 3 * 2 = 6 slag in the north
// in spring; 0 in summer, when the profile has no entry), leaves the
// ledger and the agent's residues alone, and refuses executions or remaining
// below 1.
//
// Requirement: FR-PRO-026
func TestWarmJob(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	if err := f.sh.set(f.region("north"), f.recipe("farm"), 3_000); err != nil { // not used by a warm job
		t.Fatal(err)
	}
	for i := range f.resid {
		f.resid[i] = 4_321
	}
	before, resid := f.state(), slices.Clone(f.resid)
	tests := []struct {
		name        string
		season      int
		wantPending []int64
		wantYield   int64
	}{
		{"spring", spring, []int64{18, 6}, 10_000},
		{"summer has no entry", summer, []int64{0, 0}, 0},
	}
	for _, tt := range tests {
		var j job
		if err := f.s.warmJob(&j, &f.yt, f.region("north"), tt.season, f.recipe("farm"), 7, 3, 2); err != nil {
			t.Fatalf("%s: warmJob: %v", tt.name, err)
		}
		if !slices.Equal(j.pending, tt.wantPending) || j.yield != tt.wantYield || j.executions != 3 || j.remaining != 2 || j.principal != 7 || j.recipe != f.recipe("farm") {
			t.Errorf("%s: job = %+v, want pending %v, yield %d, executions 3, remaining 2", tt.name, j, tt.wantPending, tt.wantYield)
		}
	}
	if f.state() != before || !slices.Equal(f.resid, resid) {
		t.Errorf("a warm job changed the ledger or the residues")
	}
	for _, c := range []struct{ k, rem int64 }{{0, 2}, {3, 0}, {-1, 1}} {
		var j job
		if err := f.s.warmJob(&j, &f.yt, f.region("north"), spring, f.recipe("farm"), 7, c.k, c.rem); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("warmJob(k %d, remaining %d) error = %v, want ErrOutOfBounds", c.k, c.rem, err)
		}
	}
	f.cfg.Recipes[f.recipe("farm")].Outputs[0].Quantity = 1 << 62
	f.rt = newRecipeTable(f.cfg)
	var j job
	if err := f.s.warmJob(&j, &f.yt, f.region("north"), spring, f.recipe("farm"), 7, 4, 2); !errors.Is(err, ErrOverflow) {
		t.Errorf("warmJob with a pending that does not fit: error = %v, want ErrOverflow", err)
	}
}

// TestAddPending verifies the aggregates of the snapshot (FR-PRO-042): mill
// (ash, flour) and grind (flour). A holds a mill job pending [2, 6], B mill
// [1, 4], C grind [5], D no job (abandoned) and E a grind job [8] at weight 4
// whose weight fell to 2 (8 * 2 / 4 = 4). Flour is 6 + 4 + 5 + 4 = 19 and ash
// 2 + 1 = 3; mill has 2 jobs and grind 2. A second call into the same slices
// doubles the figures (the caller zeroes), and a slice with room allocates
// nothing.
//
// Requirement: FR-PRO-042, NFR-PRO-001
func TestAddPending(t *testing.T) {
	// not parallel: AllocsPerRun panics in a parallel test
	f := newStepFixture(t)
	mill, grind := f.recipe("mill"), f.recipe("grind")
	jobs := []job{
		{recipe: mill, executions: 3, pending: []int64{2, 6}},
		{recipe: mill, executions: 2, pending: []int64{1, 4}},
		{recipe: grind, executions: 1, pending: []int64{5}},
		{recipe: mill},
		{recipe: grind, executions: 4, pending: []int64{8}},
	}
	jobs[4].reweigh(2)
	pending := make([]int64, len(f.cfg.Commodities))
	count := make([]int64, len(f.cfg.Recipes))
	for i := range jobs {
		if !f.s.addPending(&jobs[i], pending, count) {
			t.Fatalf("addPending(job %d) = false", i)
		}
	}
	flour, ash := f.com("flour"), f.com("ash")
	if pending[flour] != 19 || pending[ash] != 3 || count[mill] != 2 || count[grind] != 2 {
		t.Errorf("flour %d, ash %d, mill jobs %d, grind jobs %d, want 19, 3, 2, 2", pending[flour], pending[ash], count[mill], count[grind])
	}
	for i := range jobs {
		f.s.addPending(&jobs[i], pending, count)
	}
	if pending[flour] != 38 || count[mill] != 4 {
		t.Errorf("a second call gave flour %d, mill jobs %d, want 38, 4", pending[flour], count[mill])
	}
	// A job completed between two calls leaves the figures of the others.
	jobs[0].clear()
	clear(pending)
	clear(count)
	for i := range jobs {
		f.s.addPending(&jobs[i], pending, count)
	}
	if pending[flour] != 13 || count[mill] != 1 { // 19 - 6
		t.Errorf("after a completion flour %d, mill jobs %d, want 13, 1", pending[flour], count[mill])
	}
	if allocs := testing.AllocsPerRun(100, func() {
		for i := range jobs {
			f.s.addPending(&jobs[i], pending, count)
		}
	}); allocs != 0 {
		t.Errorf("AllocsPerRun(addPending) = %v, want 0", allocs)
	}
	big := job{recipe: grind, executions: 1, pending: []int64{1 << 62}}
	pending[flour] = 1 << 62
	if ok := f.s.addPending(&big, pending, count); ok {
		t.Errorf("addPending past int64 = true, want false")
	}
}

// TestJob_PlainOrderedState verifies by reflect that no field of the job is a
// pointer, map, function, interface or channel and that the only slice is the
// pending quantities, of int64 (FR-PRO-043).
//
// Requirement: FR-PRO-043
//
// Guard: a map or a pointer in the job would make a restored world complete on
// other ticks or give two checkpoints of one state different bytes.
func TestJob_PlainOrderedState(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(job{})
	for i := range typ.NumField() {
		fld := typ.Field(i)
		switch fld.Type.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Func, reflect.Interface, reflect.Chan, reflect.UnsafePointer:
			t.Errorf("job field %s has kind %v, want plain data", fld.Name, fld.Type.Kind())
		case reflect.Slice:
			if fld.Name != "pending" || fld.Type.Elem().Kind() != reflect.Int64 {
				t.Errorf("job slice field %s of %v, want only pending []int64", fld.Name, fld.Type)
			}
		}
	}
}

// TestSteps_Allocations verifies that, after the first, a start, an advance, a
// completion without overflow, the read of the limit and the event builders
// into destination slices with room allocate nothing (NFR-PRO-001).
//
// Requirement: NFR-PRO-001, NFR-PRO-003
func TestSteps_Allocations(t *testing.T) {
	// not parallel: AllocsPerRun panics in a parallel test
	f := newStepFixture(t)
	q := f.req("farm", "north", spring, 2, 0)
	if _, err := f.start(q); err != nil { // the first start allocates the pending slice
		t.Fatal(err)
	}
	f.finish()
	res := f.residues(q.recipe)
	f.seed("grain", 100)
	var c completion
	var abandoned RecipeAbandoned
	outs, disp, over := make([]Amount, 0, 4), make([]Amount, 0, 4), make([]Amount, 0, 4)
	checks := []struct {
		name string
		fn   func()
	}{
		{"maxExecutions", func() { f.s.maxExecutions(f.l, q) }},
		{"start", func() { f.s.start(f.l, &f.job, res, q); f.job = job{recipe: f.job.recipe, pending: f.job.pending[:0]} }},
		{"start, advance and complete", func() {
			f.s.start(f.l, &f.job, res, q)
			f.job.advance()
			f.job.advance()
			f.job.advance()
			c = f.s.complete(f.l, &f.job, f.acct, MaxQuantity)
		}},
		{"event builders", func() {
			f.s.recipeExecuted(outs[:0], disp[:0], over[:0], 12, 7, 2, &c, 3)
			abandoned = recipeAbandoned(12, 7, 2, q.recipe, 2, ReasonNoSite)
		}},
	}
	_ = abandoned
	for _, tc := range checks {
		if allocs := testing.AllocsPerRun(50, tc.fn); allocs != 0 {
			t.Errorf("AllocsPerRun(%s) = %v, want 0", tc.name, allocs)
		}
	}
}

// TestSteps_UnknownAccount verifies that a stock the ledger cannot read is an
// error of the read and of the start (for a tool and for an input), with
// nothing started, and that a completion into an account it cannot read is a
// violated invariant.
//
// Requirement: FR-PRO-010
func TestSteps_UnknownAccount(t *testing.T) {
	t.Parallel()
	f := newStepFixture(t)
	for _, recipe := range []string{"bake", "burn", "polish"} {
		q := f.req(recipe, "north", spring, 1, 0)
		q.account = 1 << 20
		if _, _, err := f.s.maxExecutions(f.l, q); !errors.Is(err, ledger.ErrUnknownAccount) {
			t.Errorf("%s: maxExecutions error = %v, want ErrUnknownAccount", recipe, err)
		}
		if res, err := f.start(q); !errors.Is(err, ledger.ErrUnknownAccount) || res.k != 0 || f.job.executions != 0 {
			t.Errorf("%s: start = (%+v, %v), want k 0 and ErrUnknownAccount", recipe, res, err)
		}
	}
	if _, err := f.start(f.req("farm", "north", spring, 1, 0)); err != nil {
		t.Fatal(err)
	}
	f.job.remaining = 0
	mustPanicInvariant(t, "completion into an unknown account", func() { f.s.complete(f.l, &f.job, 1<<20, MaxQuantity) })
}
