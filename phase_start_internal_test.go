package economysim

import (
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// panConfig is millConfig with a tool: mill requires one "pan" (durability 10,
// wear 1 per use), and every household holds 1. The commodity IDs are the
// canonical order: flour 0, grain 1, pan 2.
func panConfig() Config {
	c := millConfig(1, 2)
	c.Commodities = append(c.Commodities, Commodity{ID: "pan", ReferencePrice: 5_000, Durability: 10, Raw: true})
	c.Recipes[0].Requires = []Tool{{Commodity: "pan", WearPerUse: 1}}
	c.Roles[0].StartStock = append(c.Roles[0].StartStock, Item{Commodity: "pan", Quantity: 1})
	return c
}

const pan = 2

// TestBeginStep_ConstructionBucketIsEmittedOnceBeforeTheFirstTickStarted
// verifies P0.2a: the construction bucket comes as BoundaryFlow rows and then
// WearApplied rows with Tick -1, once, after the events of construction and
// before the first TickStarted, equal to what the ledger holds for the bucket;
// the second step emits none; and a world that never steps emits none.
//
// Arithmetic: the two agents of panConfig are funded 10 000 money each, 10 grain
// and 1 pan: the initial faucet is 20 000 money, 20 grain and 2 pans. One of the
// pans is worn by 3 units of durability in the construction phase, so the bucket
// holds 3 units of wear for the pan.
//
// Requirement: FR-OBS-015, FR-WLD-035, FR-LED-052
func TestBeginStep_ConstructionBucketIsEmittedOnceBeforeTheFirstTickStarted(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, panConfig(), WithObserver(rec))
	if err := w.ledger.WearItems(w.agents[1].account, pan, 1, 3); err != nil {
		t.Fatalf("WearItems() error = %v, want nil", err)
	}
	for _, e := range rec.events {
		if _, ok := e.(BoundaryFlow); ok {
			t.Fatalf("a BoundaryFlow was emitted by a world that never stepped, want none")
		}
	}
	wantFlows := boundaryFlows(nil, w.ledger.AppendFlows(nil, true), constructionTick, nil)
	wantWear := wearApplied(nil, w.ledger, len(w.cfg.Commodities), constructionTick, true)
	if len(wantFlows) < 3 || !slices.Equal(wantWear, []WearApplied{{Tick: -1, Commodity: pan, Units: 3}}) {
		t.Fatalf("bucket = %v and %v, want at least the money and two goods flows, and 3 units of wear of the pan", wantFlows, wantWear)
	}
	before := len(rec.events)
	step(t, w, 2)
	var seenFlows []BoundaryFlow
	var seenWear []WearApplied
	firstTick := -1
	for i, e := range rec.events[before:] {
		switch e := e.(type) {
		case BoundaryFlow:
			if e.Tick == constructionTick {
				seenFlows = append(seenFlows, e)
			}
		case WearApplied:
			if e.Tick == constructionTick {
				seenWear = append(seenWear, e)
			}
		case TickStarted:
			if firstTick < 0 {
				firstTick = i
			}
		}
	}
	if !slices.Equal(seenFlows, wantFlows) || !slices.Equal(seenWear, wantWear) {
		t.Errorf("bucket events = %v and %v, want %v and %v, each once", seenFlows, seenWear, wantFlows, wantWear)
	}
	// The bucket comes first: BoundaryFlow rows, then WearApplied rows, then TickStarted.
	want := len(wantFlows) + len(wantWear)
	if firstTick != want {
		t.Errorf("TickStarted is event %d of the first step, want %d: the %d bucket events come before it", firstTick, want, want)
	}
	if _, ok := rec.events[before+want].(TickStarted); !ok {
		t.Errorf("event %d of the first step is %T, want TickStarted", want, rec.events[before+want])
	}
}

// TestP11Moment_IsTheMomentOfTheTick verifies that P1.1 gives the calendar's
// moment of the tick, once per tick, and that P1.2 reads its season.
//
// Requirement: FR-WLD-004, FR-CAL-014
func TestP11Moment_IsTheMomentOfTheTick(t *testing.T) {
	t.Parallel()
	w := newMill(t, millConfig(1, 2))
	for tick := range int64(4) {
		if err := w.beginStep(); err != nil {
			t.Fatalf("beginStep() error = %v, want nil", err)
		}
		if err := w.p11Moment(); err != nil {
			t.Fatalf("p11Moment() at tick %d error = %v, want nil", tick, err)
		}
		want, err := w.cfg.Calendar.At(tick)
		if err != nil {
			t.Fatalf("At(%d) error = %v, want nil", tick, err)
		}
		if w.tk.moment != want {
			t.Errorf("moment at tick %d = %+v, want %+v", tick, w.tk.moment, want)
		}
		if err := w.endStep(nil); err != nil {
			t.Fatalf("endStep() error = %v, want nil", err)
		}
	}
}

// TestP12Anchors_WageAndPrices verifies the cost anchor of P1.2 for the mill:
// unknown (0) while neither the pooled wage nor the starting wage is known, the
// starting wage while the pooled one is unknown, and the pooled wage as soon as
// the welfare rings give one (a function of the rings, not a latch).
//
// Arithmetic (spec-production.md 9.1): the cost of one execution is 2 grain * 1 000
// + 1 tick * wage, and 3 flour come out at E = 10 000, so the anchor is
// ceil(cost / 3). With no wage it is 0. With the starting wage 100 it is
// ceil(2 100 / 3) = 700. Two agents whose records are an income of 500 over one
// working household-tick pool to (500 + 500) / (1 + 1) = 500, which is above 0 and
// so known, and the anchor is ceil(2 500 / 3) = ceil(833.33) = 834. The grain,
// which no recipe makes, has no anchor.
//
// Requirement: FR-PRO-037, FR-PRO-047, FR-PRO-049, FR-WLD-058, FR-VIA-052
func TestP12Anchors_WageAndPrices(t *testing.T) {
	t.Parallel()
	w := newMill(t, millConfig(1, 2))
	anchors := func() (int64, int64) {
		w.p12Anchors()
		return w.prod.unitCost(0, flour), w.prod.unitCost(0, grain)
	}
	if f, g := anchors(); f != 0 || g != 0 {
		t.Errorf("anchors with no wage = %d (flour), %d (grain), want 0 and 0", f, g)
	}
	w.seedWage[0] = 100
	if f, g := anchors(); f != 700 || g != 0 {
		t.Errorf("anchors with the starting wage 100 = %d (flour), %d (grain), want 700 and 0", f, g)
	}
	for _, id := range []AgentID{1, 2} {
		w.agents[id].history.Push(welfare.Record{Income: 500, Working: true, WorkHH: 1})
	}
	if f, _ := anchors(); f != 834 {
		t.Errorf("anchor of flour with the pooled wage 500 = %d, want 834", f)
	}
}

// TestStep_WearFactsAreRecordedAtTheStart verifies that a start that wears a
// tool puts the wear fact in the welfare scratch of the agent at P2.7, once, and
// that P8.6 clears it.
//
// Arithmetic: one execution wears the pan by WearPerUse 1 of durability 10, so
// the scratch holds one row (pan, 1 execution, 1 unit of wear per use) per agent.
//
// Requirement: FR-WEL-003
func TestStep_WearFactsAreRecordedAtTheStart(t *testing.T) {
	t.Parallel()
	w := newMill(t, panConfig())
	tickInspecting(t, w, func() {
		for _, id := range []AgentID{1, 2} {
			f := w.welfare.slots[id]
			if f.nWear != 1 || f.wear[0] != (wearRow{commodity: pan, executions: 1, wearPerUse: 1}) {
				t.Errorf("wear scratch of agent %d = %d rows, %+v, want one row (pan, 1, 1)", id, f.nWear, f.wear[0])
			}
		}
	})
	if got := w.welfare.slots[1]; got.nWear != 0 {
		t.Errorf("wear scratch after the step = %+v, want cleared at P8.6", got)
	}
}

// TestP12Anchors_RunnableRecipesAndWeightedAgents verifies, on the two-region
// world of removalConfig, three things about P1.2 that a one-region world of
// weight-1 agents cannot show: the recipes listed as runnable in a region are
// those of the roles that have an agent there and no others (the weaver has an
// agent in home only), the list is rebuilt each tick and does not grow, and an
// agent of weight above 1 counts in the wage pool of its region.
//
// Arithmetic: the roles are miller and weaver. Home holds both, so its list is the
// miller's recipes then the weaver's; yonder holds a miller only (agent 5, weight
// 2), so its list is the miller's recipes. One record of income 500 over one working
// household-tick in agent 5's history pools to 500 / 1 = 500 in yonder; the wage of
// yonder is 0 before the push (no starting wage).
//
// Requirement: FR-PRO-047, FR-WLD-058
// Guard: a list that grows each tick leaks memory without changing a result; a
// weaver recipe listed for yonder would give it a cost anchor for goods it cannot
// make; a weight-2 agent left out of the pool would understate the wage.
func TestP12Anchors_RunnableRecipesAndWeightedAgents(t *testing.T) {
	t.Parallel()
	c := removalConfig()
	c.Recipes = append(c.Recipes, Recipe{
		ID: "weave", Duration: 1, TraineeYield: 5_000,
		Inputs:  []Item{{Commodity: "grain", Quantity: 1}},
		Outputs: []Output{{Commodity: "flour", Quantity: 1, CostShare: 10_000}},
	})
	c.Roles[1].Recipes = []string{"weave"}
	w := newMill(t, c)
	miller, weaver := w.roles.row(0).recipes, w.roles.row(1).recipes
	if len(miller) == 0 || len(weaver) == 0 || slices.Equal(miller, weaver) {
		t.Fatalf("recipes of the miller %v and the weaver %v, want two different non-empty lists", miller, weaver)
	}
	want := [][]RecipeID{append(slices.Clone(miller), weaver...), miller}
	for i := range 3 {
		w.p12Anchors()
		for g, wantG := range want {
			if !slices.Equal(w.runnable[g], wantG) {
				t.Errorf("runnable[%d] after %d calls of p12Anchors = %v, want %v", g, i+1, w.runnable[g], wantG)
			}
		}
	}
	if w.agents[5].weight < 2 || w.tk.wages[1] != 0 {
		t.Fatalf("agent 5 weight %d, wage of yonder %d, want weight 2 or more and a wage of 0", w.agents[5].weight, w.tk.wages[1])
	}
	w.agents[5].history.Push(welfare.Record{Income: 500, Working: true, WorkHH: 1})
	w.p12Anchors()
	if got := w.tk.wages[1]; got != 500 {
		t.Errorf("wage of yonder = %d, want 500: the weight-2 agent counts in the pool", got)
	}
}

// TestRegionOf verifies that the region of an agent is the region of its ledger
// account, and that an agent whose account the ledger cannot read is a violated
// invariant that panics with the ledger's reason.
//
// Requirement: FR-AGT-002
func TestRegionOf(t *testing.T) {
	t.Parallel()
	w := newMill(t, removalConfig())
	for id, want := range map[AgentID]RegionID{2: 0, 4: 0, 5: 1} {
		if got := w.regionOf(&w.agents[id]); got != want {
			t.Errorf("regionOf(agent %d) = %d, want %d", id, got, want)
		}
	}
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, "invariant") || !strings.Contains(msg, "the account of a recipient is unreadable: ") {
			t.Errorf("regionOf(unknown account) panicked with %q, want an invariant panic naming the unreadable account", msg)
		}
	}()
	w.regionOf(&agent{account: 1 << 30})
	t.Errorf("regionOf(unknown account) returned, want a panic")
}
