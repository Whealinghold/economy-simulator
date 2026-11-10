package otelobserver_test

import (
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/otelobserver"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// spanHarness is an Observer that traces to an in-memory recorder.
func spanHarness(t *testing.T) (*otelobserver.Observer, *tracetest.SpanRecorder) {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	obs, err := otelobserver.New(nil, sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	obs.Observe(created())
	return obs, rec
}

// attrOf reads an int attribute of a finished span.
func attrOf(s sdktrace.ReadOnlySpan, key string) (int64, bool) {
	for _, kv := range s.Attributes() {
		if string(kv.Key) == key {
			return kv.Value.AsInt64(), true
		}
	}
	return 0, false
}

func byName(rec *tracetest.SpanRecorder, name string) (out []sdktrace.ReadOnlySpan) {
	for _, s := range rec.Ended() {
		if s.Name() == name {
			out = append(out, s)
		}
	}
	return out
}

// TestSpans_RunAndStepParenting verifies that steps in a run are children of the
// run span, and a step with no run is a root span.
//
// Requirement: rules 6.1
func TestSpans_RunAndStepParenting(t *testing.T) {
	obs, rec := spanHarness(t)
	obs.Observe(economysim.TickStarted{Tick: 0})
	obs.Observe(economysim.TickEnded{Tick: 0})
	obs.Observe(economysim.RunStarted{Ticks: 2})
	for i := int64(1); i <= 2; i++ {
		obs.Observe(economysim.TickStarted{Tick: i})
		obs.Observe(economysim.TickEnded{Tick: i})
	}
	obs.Observe(economysim.RunEnded{Completed: 2})
	steps, runs := byName(rec, "economysim.step"), byName(rec, "economysim.run")
	if len(steps) != 3 || len(runs) != 1 {
		t.Fatalf("spans = %d steps, %d runs, want 3, 1", len(steps), len(runs))
	}
	if steps[0].Parent().IsValid() {
		t.Errorf("step outside a run has parent %v, want none", steps[0].Parent())
	}
	for _, s := range steps[1:] {
		if s.Parent().SpanID() != runs[0].SpanContext().SpanID() {
			t.Errorf("step parent = %v, want the run span %v", s.Parent().SpanID(), runs[0].SpanContext().SpanID())
		}
	}
	if tk, ok := attrOf(steps[2], "economysim.tick"); !ok || tk != 2 {
		t.Errorf("last step economysim.tick = %d, %v, want 2, true", tk, ok)
	}
}

// TestSpans_ErrorStatusOnFailed verifies that a failed step and a failed run end
// with an error status, and that a step or run left open by a panic is ended
// with an error when the next one starts.
//
// Requirement: FR-OBS-055
func TestSpans_ErrorStatusOnFailed(t *testing.T) {
	obs, rec := spanHarness(t)
	obs.Observe(economysim.RunStarted{})
	obs.Observe(economysim.TickStarted{Tick: 1})
	obs.Observe(economysim.TickEnded{Tick: 1, Failed: true})
	obs.Observe(economysim.TickStarted{Tick: 2}) // never ended: a panicked step
	obs.Observe(economysim.TickStarted{Tick: 3})
	obs.Observe(economysim.TickEnded{Tick: 3})
	obs.Observe(economysim.RunEnded{Failed: true})
	obs.Observe(economysim.RunStarted{}) // never ended
	obs.Observe(economysim.RunStarted{})
	obs.Observe(economysim.RunEnded{})
	steps, runs := byName(rec, "economysim.step"), byName(rec, "economysim.run")
	if d := steps[0].Status().Description; d != "step failed" {
		t.Errorf("failed step status description = %q, want %q", d, "step failed")
	}
	if d := runs[0].Status().Description; d != "run failed" {
		t.Errorf("failed run status description = %q, want %q", d, "run failed")
	}
	wantSteps := []codes.Code{codes.Error, codes.Error, codes.Unset}
	for i, w := range wantSteps {
		if g := steps[i].Status().Code; g != w {
			t.Errorf("step %d status = %v, want %v", i, g, w)
		}
	}
	wantRuns := []codes.Code{codes.Error, codes.Error, codes.Unset}
	for i, w := range wantRuns {
		if g := runs[i].Status().Code; g != w {
			t.Errorf("run %d status = %v, want %v", i, g, w)
		}
	}
}

// TestSpans_StepAttributesAreCounts verifies the step span carries the counts of
// orders placed, rejected, trades and volume, and 0 for an empty step.
//
// Requirement: FR-OBS-055
func TestSpans_StepAttributesAreCounts(t *testing.T) {
	obs, rec := spanHarness(t)
	obs.Observe(economysim.TickStarted{Tick: 0})
	for i := 0; i < 5; i++ {
		obs.Observe(economysim.OrderPlaced{Commodity: 1, Region: 1, Side: economysim.Bid})
	}
	obs.Observe(economysim.OrderRejected{Commodity: 0, Reason: "no_cash"})
	obs.Observe(economysim.TradeExecuted{Commodity: 2, Quantity: 4})
	obs.Observe(economysim.TradeExecuted{Commodity: 2, Quantity: 3})
	obs.Observe(economysim.TickEnded{Tick: 0})
	obs.Observe(economysim.TickStarted{Tick: 1})
	obs.Observe(economysim.TickEnded{Tick: 1})
	steps := byName(rec, "economysim.step")
	for i, want := range [][4]int64{{5, 1, 2, 7}, {0, 0, 0, 0}} {
		for j, key := range []string{"orders.placed", "orders.rejected", "trades", "volume"} {
			if g, ok := attrOf(steps[i], "economysim."+key); !ok || g != want[j] {
				t.Errorf("step %d economysim.%s = %d, %v, want %d, true", i, key, g, ok, want[j])
			}
		}
	}
}

// TestSpans_ClassEEventIsSpanEvent verifies that a bankruptcy in a step is a span
// event of the step with its fields, and that one outside a step goes to the run
// span or nowhere.
//
// Requirement: rules 6.2
func TestSpans_ClassEEventIsSpanEvent(t *testing.T) {
	obs, rec := spanHarness(t)
	bankrupt := economysim.AgentBankrupted{Region: 1, Role: 0, Weight: 3, Cause: economysim.CauseInsolvent}
	obs.Observe(bankrupt) // no run, no step: dropped
	obs.Observe(economysim.RunStarted{})
	obs.Observe(bankrupt) // run span
	obs.Observe(economysim.TickStarted{Tick: 0})
	obs.Observe(bankrupt)
	obs.Observe(economysim.AgentRelocated{From: 0, To: 9, Weight: 1, Cause: economysim.CauseMigration})
	obs.Observe(economysim.TickEnded{Tick: 0})
	obs.Observe(economysim.RunEnded{})
	step, run := byName(rec, "economysim.step")[0], byName(rec, "economysim.run")[0]
	if n := len(run.Events()); n != 1 || run.Events()[0].Name != "economysim.agent_bankrupted" {
		t.Fatalf("run events = %v, want one economysim.agent_bankrupted", run.Events())
	}
	if len(step.Events()) != 2 {
		t.Fatalf("step events = %v, want 2", step.Events())
	}
	want := map[attribute.Key]string{"economysim.region": "r1", "economysim.role": "k0", "economysim.cause": "insolvent"}
	for _, kv := range step.Events()[0].Attributes {
		if w, ok := want[kv.Key]; ok && kv.Value.AsString() != w {
			t.Errorf("event attribute %s = %q, want %q", kv.Key, kv.Value.AsString(), w)
		}
		delete(want, kv.Key)
	}
	if len(want) != 0 {
		t.Errorf("event lacks attributes %v", want)
	}
	if rel := step.Events()[1]; len(rel.Attributes) != 3 {
		t.Errorf("relocation event attributes = %v, want 3 (the destination outside the names is left out)", rel.Attributes)
	}
}

// TestSpans_NoSpanPerOrderTradeOrAgent verifies that orders, rejections and
// trades make no span and no span event.
//
// Requirement: rules 6.1, 6.2
func TestSpans_NoSpanPerOrderTradeOrAgent(t *testing.T) {
	obs, rec := spanHarness(t)
	obs.Observe(economysim.TickStarted{Tick: 0})
	for i := 0; i < 10_000; i++ {
		obs.Observe(economysim.OrderPlaced{Commodity: 0, Region: 0, Side: economysim.Ask})
		obs.Observe(economysim.OrderRejected{Commodity: 0, Reason: "no_cash"})
		obs.Observe(economysim.TradeExecuted{Commodity: 0, Quantity: 1})
		obs.Observe(economysim.RecipeExecuted{Recipe: 0, Executions: 1})
	}
	obs.Observe(economysim.TickEnded{Tick: 0})
	spans := rec.Ended()
	if len(spans) != 1 || len(spans[0].Events()) != 0 {
		t.Errorf("spans = %d, events of the step = %d, want 1 span with 0 events", len(spans), len(spans[0].Events()))
	}
}

// TestSpans_EveryEntityEventIsOneSpanEvent verifies that each rare entity event
// the adapter traces becomes exactly one span event of the step with the name
// economysim.<event>, and that no other event type does.
//
// Requirement: rules 6.2
func TestSpans_EveryEntityEventIsOneSpanEvent(t *testing.T) {
	obs, rec := spanHarness(t)
	traced := map[string]string{"AgentSpawned": "agent_spawned", "AgentBankrupted": "agent_bankrupted",
		"WeightChanged": "weight_changed", "RoleChanged": "role_changed", "AgentRelocated": "agent_relocated",
		"ShipmentDispatched": "shipment_dispatched", "ShipmentArrived": "shipment_arrived",
		"ContractFormed": "contract_formed", "ContractSettled": "contract_settled", "MigrationStarted": "migration_started"}
	f := &filler{idx: 1}
	obs.Observe(economysim.TickStarted{})
	for _, e := range allEvents {
		switch eventName(e) {
		case "economysim.WorldCreated", "economysim.WorldRestored", "economysim.RunStarted", "economysim.RunEnded",
			"economysim.TickStarted", "economysim.TickEnded":
		default:
			obs.Observe(f.filled(e))
		}
	}
	obs.Observe(economysim.TickEnded{})
	var got []string
	for _, ev := range rec.Ended()[0].Events() {
		got = append(got, ev.Name)
	}
	var want []string
	for _, n := range traced {
		want = append(want, "economysim."+n)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("span events = %v, want %v", got, want)
	}
}

// TestSpans_OpenStepEndsWithRunBoundary verifies that a step left open when the
// run ends, or when the next run starts, is ended with an error and is not
// carried into the next run, and that the run span records how it ended.
//
// Requirement: FR-OBS-055
func TestSpans_OpenStepEndsWithRunBoundary(t *testing.T) {
	obs, rec := spanHarness(t)
	obs.Observe(economysim.RunStarted{Ticks: 4})
	obs.Observe(economysim.TickStarted{Tick: 1}) // a panic: no TickEnded
	obs.Observe(economysim.RunEnded{Completed: 3, Cancelled: true})
	obs.Observe(economysim.RunStarted{})
	obs.Observe(economysim.TickStarted{Tick: 5})
	obs.Observe(economysim.RunStarted{})
	obs.Observe(economysim.TickStarted{Tick: 6})
	obs.Observe(economysim.TickEnded{Tick: 6})
	obs.Observe(economysim.RunEnded{})
	steps, runs := byName(rec, "economysim.step"), byName(rec, "economysim.run")
	if len(steps) != 3 || len(runs) != 3 {
		t.Fatalf("spans = %d steps, %d runs, want 3, 3", len(steps), len(runs))
	}
	checkOpenBoundary(t, steps, runs)
}

// checkOpenBoundary checks the spans of TestSpans_OpenStepEndsWithRunBoundary.
func checkOpenBoundary(t *testing.T, steps, runs []sdktrace.ReadOnlySpan) {
	t.Helper()
	for i, s := range steps[:2] {
		if s.Status().Code != codes.Error || s.Status().Description != "step did not end" {
			t.Errorf("open step %d status = %v, want an error", i, s.Status())
		}
	}
	if steps[2].Parent().SpanID() != runs[2].SpanContext().SpanID() {
		t.Errorf("step of the last run has parent %v, want that run %v", steps[2].Parent().SpanID(), runs[2].SpanContext().SpanID())
	}
	for key, want := range map[string]int64{"economysim.completed": 3, "economysim.ticks": 4} {
		if c, ok := attrOf(runs[0], key); !ok || c != want {
			t.Errorf("run %s = %d, %v, want %d, true", key, c, ok, want)
		}
	}
	if runs[1].Status().Description != "run did not end" || len(runs[0].Attributes()) != 3 {
		t.Errorf("open run status = %v, first run attributes = %v, want %q and 3 attributes", runs[1].Status(), runs[0].Attributes(), "run did not end")
	}
}

// TestSpans_EventAttributeKeys verifies the attribute keys of each span event,
// and that an index equal to the length of its table leaves its attribute out.
//
// Guard: the keys are this module's own (spec 6.6.5 fixes the content, not the
// names, plan 2-22 D653); a dashboard on a span event breaks if one is renamed.
func TestSpans_EventAttributeKeys(t *testing.T) {
	want := map[string]string{
		"AgentSpawned":       "region role weight",
		"AgentBankrupted":    "cause region role weight",
		"WeightChanged":      "cause from region role to",
		"RoleChanged":        "cause region role.from role.to weight",
		"AgentRelocated":     "cause region.from region.to weight",
		"ShipmentDispatched": "mode region.from region.to units",
		"ShipmentArrived":    "mode region.from region.to",
		"ContractFormed":     "commodity quantity region.from region.to",
		"ContractSettled":    "cause commodity quantity region.from region.to",
		"MigrationStarted":   "region.from region.to weight",
	}
	for _, idx := range []int{1, 3} { // 3 is the length of the region table: out of range
		obs, rec := spanHarness(t)
		f := &filler{idx: idx}
		obs.Observe(economysim.TickStarted{})
		for _, e := range allEvents {
			if _, ok := want[eventName(e)[len("economysim."):]]; ok {
				obs.Observe(f.filled(e))
			}
		}
		obs.Observe(economysim.TickEnded{})
		for i, ev := range rec.Ended()[0].Events() {
			var keys []string
			for _, kv := range ev.Attributes {
				keys = append(keys, strings.TrimPrefix(string(kv.Key), "economysim."))
			}
			slices.Sort(keys)
			exp := want[camel(strings.TrimPrefix(ev.Name, "economysim."))]
			if idx == 3 {
				exp = dropIDKeys(exp)
			}
			if got := strings.Join(keys, " "); got != exp {
				t.Errorf("idx %d event %d %s attributes = %q, want %q", idx, i, ev.Name, got, exp)
			}
		}
	}
}

// camel turns a snake case event name into its type name.
func camel(s string) string {
	var b strings.Builder
	for _, w := range strings.Split(s, "_") {
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

// dropIDKeys removes the attributes read from a table when every ID is out of
// range: regions, roles, modes and commodities.
func dropIDKeys(keys string) string {
	var kept []string
	for _, k := range strings.Fields(keys) {
		if !strings.HasPrefix(k, "region") && !strings.HasPrefix(k, "role") && k != "mode" && k != "commodity" {
			kept = append(kept, k)
		}
	}
	return strings.Join(kept, " ")
}

// TestSpans_ResetMidStepEndsStepSpan verifies that a WorldCreated that arrives
// while a step is open ends that step's span with an error status.
//
// Guard: the span would otherwise stay open until the next step began.
func TestSpans_ResetMidStepEndsStepSpan(t *testing.T) {
	obs, rec := spanHarness(t)
	obs.Observe(economysim.TickStarted{Tick: 1})
	obs.Observe(economysim.WorldCreated{})
	steps := byName(rec, "economysim.step")
	if len(steps) != 1 || steps[0].Status().Description != "step did not end" {
		t.Errorf("spans after a reset mid-step = %d, want 1 ended with %q", len(steps), "step did not end")
	}
}
