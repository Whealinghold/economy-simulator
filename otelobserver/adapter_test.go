package otelobserver_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// sets reads the attribute sets of every point the SDK holds, by metric name.
func (h *harness) sets() map[string][]attribute.Set {
	h.t.Helper()
	var rm metricdata.ResourceMetrics
	if err := h.read.Collect(context.Background(), &rm); err != nil {
		h.t.Fatalf("Collect() error = %v, want nil", err)
	}
	out := map[string][]attribute.Set{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch d := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, p := range d.DataPoints {
					out[m.Name] = append(out[m.Name], p.Attributes)
				}
			case metricdata.Gauge[int64]:
				for _, p := range d.DataPoints {
					out[m.Name] = append(out[m.Name], p.Attributes)
				}
			case metricdata.Histogram[float64]:
				for _, p := range d.DataPoints {
					out[m.Name] = append(out[m.Name], p.Attributes)
				}
			}
		}
	}
	return out
}

// closedSets are the constants of spec-observability.md 6.5.1, copied here so
// that the test does not trust the adapter's own table.
var closedSets = map[string][]string{
	"economysim.bound":        {"floor", "ceiling"},
	"economysim.belief_limit": {"conflict", "clamped"},
	"economysim.work_decline": {"rest", "no_recipe"},
	"economysim.cause": {"insolvent", "starved", "consumer", "population", "transport", "migration", "delivered", "short",
		"payer_removed", "deliverer_removed"},
	"economysim.entry_limit": {"unfunded", "no_viable_role", "region_cap", "role_cap", "role_ceiling", "cooldown",
		"hysteresis", "exit_cap", "refused", "stranded"},
	"economysim.kind":    {"faucet", "sink"},
	"economysim.side":    {"bid", "ask"},
	"economysim.outcome": {"offered", "no_path", "too_old", "not_yet"},
	"economysim.limit": {"no_estimate", "no_destination", "no_rumour", "too_far", "no_viable", "no_gain", "bound",
		"out_cap", "unprovisioned", "in_cap", "held", "ended"},
	"economysim.decision": {"no_rumour", "stale", "locked", "perishable", "no_margin", "unaffordable", "below_share",
		"planned", "rested", "waiting", "lot_capped"},
	"economysim.holder":           {"agents", "estate", "treasury"},
	"economysim.sector":           {"households", "pool", "contract", "treasury"},
	"economysim.population_limit": {"deaths_cut", "deaths_held", "births_capped", "births_unfunded", "rate_clamped"},
	"economysim.flow_limit":       {"short", "headroom", "limit"},
	"economysim.own_use_change":   {"entered", "left"},
	"economysim.external_cut":     {"closed", "cap", "price"},
	"economysim.contract_failure": {"no_taker", "payer_funds", "too_many", "dispatch_refused"},
	"economysim.contract_request": {"offer", "take"},
	"economysim.direction":        {"from", "to"},
}

// openReasonCap is the intern cap of spec-observability.md 6.5.1 for an open
// string, the bound of economysim.reason.
const openReasonCap = 1024

// allowed is the values each attribute key may take for a world with names n.
func allowed(n economysim.Names) map[string][]string {
	m := map[string][]string{
		"economysim.commodity": n.Commodities, "economysim.region": n.Regions, "economysim.region.from": n.Regions,
		"economysim.region.to": n.Regions, "economysim.node": n.Regions, "economysim.role": n.Roles,
		"economysim.recipe": n.Recipes, "economysim.want": n.Wants, "economysim.resource": n.Resources,
		"economysim.mode": n.Modes,
	}
	for k, v := range closedSets {
		m[k] = v
	}
	return m
}

// TestCardinality_AttributeSetsWithinBounds verifies that after a real world and
// one event of every type with IDs in and out of range, each attribute takes only
// a value of its allowed set, no key names an unbounded thing, and the series of
// each instrument are at most the product of the allowed sets (the formula of
// spec-observability.md 6.5.2).
//
// Requirement: FR-OBS-052
// Guard: a label that grew with agents, orders or ticks would be the cost of the
// whole backend; this is the only test that sees every instrument at once.
func TestCardinality_AttributeSetsWithinBounds(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	h := newHarness(t)
	w, err := economysim.NewWorld(miniConfig(), economysim.WithSeed(7), economysim.WithObserver(h.obs))
	if err != nil {
		t.Fatalf("NewWorld() error = %v", err)
	}
	for i := 0; i < 200; i++ {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() i=%d error = %v", i, err)
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	h.obs.RecordSnapshot(snap)
	// The same snapshot with the modules of the deferred gauges on and rows for them.
	on := snap
	on.Names.Modes = []string{"m0", "m1"}
	on.Modules = []string{"population", "transport", "migration"}
	on.Population = economysim.PopulationState{HasReplacement: true, ReplacementRatioBP: 9_000}
	on.Shipments = []economysim.ShipmentRow{{Mode: 1, TicksLeft: 0}, {Mode: 7, TicksLeft: 0}}
	on.Away = []economysim.AgentID{3}
	on.Regions = []economysim.RegionRow{{Region: 0, HouseholdCap: 5, OnJourneyFrom: 1, OnJourneyTo: 2}, {Region: 99}}
	h.obs.RecordSnapshot(on)
	f := &filler{}
	for n, idx := range []int{0, 1, 2, 99, -1, 1, 0} {
		f.idx = idx
		h.send(economysim.TickStarted{Tick: int64(1000 + n)})
		for _, e := range allEvents {
			if !slices.Contains([]string{"WorldCreated", "WorldRestored", "RunStarted", "RunEnded", "TickStarted", "TickEnded"}, eventName(e)[len("economysim."):]) {
				h.send(f.filled(e))
			}
		}
		h.send(economysim.TickEnded{Tick: int64(1000 + n)})
	}
	total := checkSets(t, h.sets(), allowed(on.Names))
	t.Logf("series after 200 ticks and the synthetic events: %d", total)
}

// checkSets checks every series of every instrument against the allowed values
// and the product bound, and returns the number of series.
func checkSets(t *testing.T, all map[string][]attribute.Set, ok map[string][]string) (total int) {
	t.Helper()
	for name, sets := range all {
		bound := 1
		keys := map[string]bool{}
		for _, set := range sets {
			for it := set.Iter(); it.Next(); {
				keys[string(it.Attribute().Key)] = true
				checkAttr(t, name, it.Attribute(), ok)
			}
		}
		for key := range keys {
			bound *= max(len(ok[key]), 1)
			if key == "economysim.reason" {
				bound *= openReasonCap
			}
		}
		if len(sets) > bound {
			t.Errorf("%s has %d series, want at most %d", name, len(sets), bound)
		}
		total += len(sets)
	}
	return total
}

// checkAttr checks one attribute of a series: not an unbounded key, and a value
// of its allowed set (the open reason is bounded by its cap instead).
func checkAttr(t *testing.T, name string, kv attribute.KeyValue, ok map[string][]string) {
	t.Helper()
	key := string(kv.Key)
	if bad := strings.TrimPrefix(key, "economysim."); slices.Contains([]string{"agent", "order", "tick", "price", "quantity", "contract", "shipment"}, bad) {
		t.Errorf("%s has attribute %s, want none of agent, order, tick, price, quantity, contract, shipment", name, key)
	}
	if key != "economysim.reason" && !slices.Contains(ok[key], kv.Value.AsString()) {
		t.Errorf("%s has %s = %q, want one of %v", name, key, kv.Value.AsString(), ok[key])
	}
}

// TestAdapter_HoldsNoWorldAndNoAgentKeys verifies by reflection that the adapter
// holds no pointer to a World and no map keyed by an agent, so it cannot call the
// world and cannot grow with the number of agents.
//
// Requirement: FR-OBS-056
func TestAdapter_HoldsNoWorldAndNoAgentKeys(t *testing.T) {
	h := newHarness(t)
	seen := map[reflect.Type]bool{}
	var walk func(path string, ty reflect.Type)
	walk = func(path string, ty reflect.Type) {
		if seen[ty] {
			return
		}
		seen[ty] = true
		switch ty.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			walk(path+"[]", ty.Elem())
		case reflect.Map:
			if ty.Key() == reflect.TypeOf(economysim.AgentID(0)) {
				t.Errorf("%s is a map keyed by AgentID", path)
			}
			walk(path+"{key}", ty.Key())
			walk(path+"{}", ty.Elem())
		case reflect.Struct:
			if ty == reflect.TypeOf(economysim.World{}) {
				t.Errorf("%s holds a World", path)
			}
			for i := range ty.NumField() {
				walk(path+"."+ty.Field(i).Name, ty.Field(i).Type)
			}
		}
	}
	walk("Observer", reflect.TypeOf(h.obs).Elem())
	if len(seen) < 10 {
		t.Errorf("walked %d types, want the whole Observer", len(seen))
	}
}

// TestAdapter_RunWithAndWithoutGivesSameWorld verifies that the adapter changes
// nothing in the world: after 100 ticks a world with it and one with a bare
// recording observer have equal snapshots and checkpoints.
//
// Requirement: FR-OBS-056
func TestAdapter_RunWithAndWithoutGivesSameWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	run := func(o economysim.Observer) (economysim.Snapshot, []byte) {
		w, err := economysim.NewWorld(miniConfig(), economysim.WithSeed(11), economysim.WithObserver(o))
		if err != nil {
			t.Fatalf("NewWorld() error = %v", err)
		}
		for i := 0; i < 100; i++ {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() i=%d error = %v", i, err)
			}
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v", err)
		}
		c, err := w.Checkpoint()
		if err != nil {
			t.Fatalf("Checkpoint() error = %v", err)
		}
		b, err := c.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary() error = %v", err)
		}
		return s, b
	}
	events := 0
	sa, ba := run(newHarness(t).obs)
	sb, bb := run(economysim.ObserverFunc(func(economysim.Event) { events++ }))
	if events == 0 {
		t.Fatal("the recording observer saw no event")
	}
	if !reflect.DeepEqual(sa, sb) || !bytes.Equal(ba, bb) {
		t.Errorf("snapshots equal = %v, checkpoints equal = %v, want both true", reflect.DeepEqual(sa, sb), bytes.Equal(ba, bb))
	}
}

// handled, ignored: the sorting of every event type (plan 2-22 D650). An event in
// neither list fails the test, so a new event is a decision, not an omission.
var (
	handledEvents = []string{"WorldCreated", "WorldRestored", "RunStarted", "RunEnded", "TickStarted", "TickEnded",
		"OrderPlaced", "OrderRejected", "TradeExecuted", "MarketCleared", "BeliefsUpdated", "RecipeExecuted",
		"RecipeRefused", "RecipeAbandoned", "ProductionHalted", "UpkeepShortfall", "WantShortfall", "AgentSpawned",
		"AgentBankrupted", "WeightChanged", "RoleChanged", "EntryLimited", "BoundaryFlow", "WearApplied",
		"AgentRelocated", "MoneyFlowLimited", "EndowmentChanged", "EndowmentAdjusted", "OwnUseChanged",
		"PopulationChanged", "ShipmentDispatched", "ShipmentArrived", "RumoursServed", "MigrationStarted",
		"MigrationReviewed", "MerchantsPlanned", "ContractFormed", "ContractSettled", "OutsideTraded", "ShipmentRejected",
		"ContractRound"}
	ignoredEvents = []string{"EstateOpened", "EdgeStatusChanged", "ReliefPaid", "MoneyFlowEnded", "MoneyFlowScaled",
		"PopulationScaled", "TransitFactorChanged", "OutsideEnded", "OutsideShocked"}
)

// TestAdapter_ExhaustiveOverEvents verifies that every event type is on the
// handled list or the ignored list and on one only, that the list of types is the
// one the root test ties to event.go, and that an ignored event records nothing.
//
// Requirement: FR-OBS-052
func TestAdapter_ExhaustiveOverEvents(t *testing.T) {
	data, err := os.ReadFile("testdata/all_events.txt")
	if err != nil {
		t.Fatalf("read list: %v", err)
	}
	var names []string
	for _, e := range allEvents {
		names = append(names, eventName(e)[len("economysim."):])
	}
	slices.Sort(names)
	if want := strings.Fields(string(data)); !slices.Equal(names, want) {
		t.Errorf("allEvents = %v, want the names of testdata/all_events.txt %v", names, want)
	}
	sorted := append(slices.Clone(handledEvents), ignoredEvents...)
	slices.Sort(sorted)
	if !slices.Equal(sorted, names) {
		t.Errorf("handled + ignored = %v, want every event type %v exactly once", sorted, names)
	}
	f := &filler{idx: 1}
	for _, e := range allEvents {
		if !slices.Contains(ignoredEvents, eventName(e)[len("economysim."):]) {
			continue
		}
		h := newHarness(t)
		h.send(created(), economysim.TickStarted{}, f.filled(e), economysim.TickEnded{})
		if got := h.collect(); len(got) != 1 {
			t.Errorf("ignored %T recorded %v, want only the step duration", e, got)
		}
	}
}
