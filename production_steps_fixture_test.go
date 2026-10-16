package economysim

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// stepsConfig is the canonical configuration of the production steps tests.
// Commodities sort to ash 0, bread 1 (spoils in 5 ticks), flour 2, grain 3,
// oven 4 (durability 20), peel 5 (durability 7), slag 6. Recipes sort to bake
// 0, burn 1, farm 2, forge 3, fract 4, grind 5, mill 6, polish 7, rework 8,
// scrap 9, trace 10. Regions sort to arid 0, crater 1, north 2, south 3. The
// seasons are spring 0 and summer 1. The yield profiles are those of
// spec-production.md 12.1: farm in the north has a spring entry only, in the
// south spring 12 000 and summer 6 000, in the crater all = 0, and in the arid
// region no profile.
func stepsConfig() *Config {
	c := Config{
		Calendar: Calendar{Seasons: []Season{{ID: "spring"}, {ID: "summer"}}},
		Regions:  []Region{{ID: "north"}, {ID: "south"}, {ID: "crater"}, {ID: "arid"}},
		Commodities: []Commodity{{ID: "grain"}, {ID: "flour"}, {ID: "bread", ShelfLife: 5}, {ID: "ash"},
			{ID: "slag"}, {ID: "oven", Durability: 20}, {ID: "peel", Durability: 7}},
		Recipes: []Recipe{
			{ID: "bake", Duration: 1,
				Inputs:   []Item{{Commodity: "grain", Quantity: 1}, {Commodity: "flour", Quantity: 2}},
				Outputs:  []Output{{Commodity: "bread", Quantity: 3, CostShare: 10_000}, {Commodity: "ash", Quantity: 1}},
				Requires: []Tool{{Commodity: "oven", WearPerUse: 5}}},
			{ID: "burn", Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 2}}},
			{ID: "farm", Duration: 3, Sited: true,
				Outputs: []Output{{Commodity: "grain", Quantity: 6, CostShare: 10_000}, {Commodity: "slag", Quantity: 2, Discard: true}}},
			{ID: "forge", Duration: 1, Outputs: []Output{{Commodity: "oven", Quantity: 2, CostShare: 10_000}}},
			{ID: "fract", Duration: 1, Outputs: []Output{{Commodity: "bread", Quantity: 5, CostShare: 10_000}}},
			{ID: "grind", Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 5}},
				Outputs: []Output{{Commodity: "flour", Quantity: 3, CostShare: 10_000}}},
			{ID: "mill", Duration: 3, Inputs: []Item{{Commodity: "grain", Quantity: 3}},
				Outputs: []Output{{Commodity: "flour", Quantity: 2, CostShare: 10_000}, {Commodity: "ash", Quantity: 1, Discard: true}}},
			{ID: "polish", Duration: 1, Outputs: []Output{{Commodity: "ash", Quantity: 1, CostShare: 10_000}},
				Requires: []Tool{{Commodity: "peel", WearPerUse: 7}}},
			{ID: "rework", Duration: 1, Inputs: []Item{{Commodity: "oven", Quantity: 2}},
				Outputs:  []Output{{Commodity: "ash", Quantity: 1, CostShare: 10_000}},
				Requires: []Tool{{Commodity: "oven", WearPerUse: 5}}},
			{ID: "scrap", Duration: 1, Outputs: []Output{{Commodity: "slag", Quantity: 2, Discard: true}}},
			{ID: "trace", Duration: 1,
				Outputs: []Output{{Commodity: "bread", Quantity: 1, CostShare: 10_000}, {Commodity: "slag", Quantity: 1, Discard: true}}},
		},
		Yields: []YieldProfile{
			{Region: "north", Recipe: "farm", Seasons: []SeasonYield{{Season: "spring", BP: 10_000}}},
			{Region: "south", Recipe: "farm", Seasons: []SeasonYield{{Season: "spring", BP: 12_000}, {Season: "summer", BP: 6_000}}},
			{Region: "crater", Recipe: "farm", HasAll: true, All: 0},
		},
	}
	cc := c.canonical()
	return &cc
}

// The seasons of stepsConfig.
const (
	spring = 0
	summer = 1
)

// stepFixture is a real ledger with one agent account and the production
// scratch over a small configuration (plan-phase-1-5.md Phase 6, A8).
type stepFixture struct {
	t     *testing.T
	cfg   *Config
	rt    recipeTable
	yt    yieldTable
	sh    yieldShocks
	s     *productionScratch
	l     *ledger.Ledger
	acct  int
	job   job
	resid []int64
	all   []RecipeID
}

func newStepFixture(t *testing.T) *stepFixture {
	t.Helper()
	cfg := stepsConfig()
	f := &stepFixture{t: t, cfg: cfg, rt: newRecipeTable(cfg), yt: newYieldTable(cfg)}
	f.s = newProductionScratch(cfg, &f.rt)
	l, err := newLedger(*cfg)
	if err != nil {
		t.Fatalf("newLedger: %v", err)
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	f.l = l
	if f.acct, err = l.Open(ledger.KindAgent, 0, 0); err != nil {
		t.Fatalf("Open: %v", err)
	}
	for i := range cfg.Recipes {
		f.all = append(f.all, RecipeID(i))
	}
	f.resid = make([]int64, residueLen(f.all, &f.rt))
	return f
}

func (f *stepFixture) recipe(name string) RecipeID {
	f.t.Helper()
	id, ok := f.cfg.recipeID(name)
	if !ok {
		f.t.Fatalf("recipe %q not in the configuration", name)
	}
	return id
}

func (f *stepFixture) com(name string) CommodityID {
	f.t.Helper()
	id, ok := f.cfg.commodityID(name)
	if !ok {
		f.t.Fatalf("commodity %q not in the configuration", name)
	}
	return id
}

func (f *stepFixture) region(name string) RegionID {
	f.t.Helper()
	id, ok := f.cfg.regionID(name)
	if !ok {
		f.t.Fatalf("region %q not in the configuration", name)
	}
	return id
}

// seed gives the agent n units of a commodity from the Initial boundary.
func (f *stepFixture) seed(name string, n int64) {
	f.t.Helper()
	c := int(f.com(name))
	from, ok := f.l.Boundary(ledger.Initial, c)
	if !ok {
		f.t.Fatalf("no Initial boundary for %s", name)
	}
	if err := f.l.Move(from, f.acct, c, n, ledger.Initial); err != nil {
		f.t.Fatalf("seed %d %s: %v", n, name, err)
	}
}

// reserve earmarks n units of a commodity of the agent.
func (f *stepFixture) reserve(name string, n int64) {
	f.t.Helper()
	if err := f.l.ReserveGoods(f.acct, int(f.com(name)), n); err != nil {
		f.t.Fatalf("reserve %d %s: %v", n, name, err)
	}
}

// held returns the units of a commodity the agent holds and how many are earmarked.
func (f *stepFixture) held(name string) (held, reserved int64) {
	f.t.Helper()
	h, r, err := f.l.Holding(f.acct, int(f.com(name)))
	if err != nil {
		f.t.Fatalf("Holding(%s): %v", name, err)
	}
	return h, r
}

// flow returns the total of a reason for a commodity this tick, over regions.
func (f *stepFixture) flow(r ledger.Reason, name string) (total int64) {
	f.t.Helper()
	c := int(f.com(name))
	for _, fl := range f.l.AppendFlows(nil, false) {
		if fl.Reason == r && fl.Commodity == c {
			total += fl.Total
		}
	}
	return total
}

// state renders everything observable of the ledger: every account view, the
// lots of every account and both flow tables. The ledger has no clone.
func (f *stepFixture) state() string {
	var b strings.Builder
	for id := range f.l.Len() {
		v, err := f.l.Account(id)
		lots, lerr := f.l.AppendLots(nil, id)
		fmt.Fprintf(&b, "%d %+v %v %v %v;", id, v, err, lots, lerr)
	}
	fmt.Fprintf(&b, "%v %v", f.l.AppendFlows(nil, false), f.l.AppendFlows(nil, true))
	return b.String()
}

// req builds a request with the effective factor of the region and season.
func (f *stepFixture) req(recipe, region string, season int, weight, maximum int64) *startReq {
	f.t.Helper()
	rid := f.recipe(recipe)
	e, has, ok := f.yt.effective(&f.sh, f.region(region), rid, season, f.cfg.Recipes[rid].Sited)
	if !ok {
		f.t.Fatalf("effective factor of %s overflows", recipe)
	}
	return &startReq{recipe: rid, account: f.acct, principal: 7, weight: weight, maximum: maximum, effective: e, hasProfile: has}
}

// residues returns the agent's residues of a recipe's outputs.
func (f *stepFixture) residues(recipe RecipeID) []int64 {
	off, ok := residueOffset(f.all, &f.rt, recipe)
	if !ok {
		f.t.Fatalf("recipe %d has no residue offset", recipe)
	}
	return f.resid[off : off+len(f.rt.row(recipe).outputs)]
}

// start runs the start step for the fixture's job.
func (f *stepFixture) start(q *startReq) (startResult, error) {
	return f.s.start(f.l, &f.job, f.residues(q.recipe), q)
}

// finish advances the job to its end and completes it, returning the completion.
func (f *stepFixture) finish() completion {
	f.t.Helper()
	for {
		if done, _ := f.job.advance(); done {
			break
		}
	}
	return f.s.complete(f.l, &f.job, f.acct, MaxQuantity)
}

// mustPanicInvariant runs fn and fails unless it panics with a message that
// contains "invariant".
func mustPanicInvariant(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		msg, _ := r.(string)
		if !strings.Contains(msg, "invariant") {
			t.Errorf("%s: recovered %v, want a panic whose message contains \"invariant\"", what, r)
		}
	}()
	fn()
}
