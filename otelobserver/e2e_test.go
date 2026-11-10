package otelobserver_test

import (
	"context"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/otelobserver"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// e2e builds the way a consumer wires the adapter: a real world, the SDK's
// meter and tracer providers on in-memory sinks, and the adapter given to the
// world with WithObserver.
func e2e(t *testing.T, extra ...economysim.Option) (*economysim.World, *harness, *tracetest.SpanRecorder) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	rec := tracetest.NewSpanRecorder()
	obs, err := otelobserver.New(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)),
		sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	opts := append([]economysim.Option{economysim.WithSeed(5), economysim.WithObserver(obs)}, extra...)
	w, err := economysim.NewWorld(miniConfig(), opts...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	return w, &harness{t: t, obs: obs, read: reader}, rec
}

// TestEndToEnd_RunTracesAndMetersAgree verifies, on a real world run for 60
// ticks with World.Run, that the trace and the metrics tell the same story: one
// run span with every step span as its child, the step spans' trade and volume
// counts sum to the counters, one duration sample and no failure per step, and
// the snapshot gauges appear once a snapshot is passed.
//
// Requirement: FR-OBS-052, FR-OBS-055, FR-OBS-056
// Guard: the unit tests feed hand-built events; only this run proves the order
// and the content of the events a real world emits drive both signals without
// drift between them.
func TestEndToEnd_RunTracesAndMetersAgree(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const ticks = 60
	w, h, rec := e2e(t)
	done, err := w.Run(context.Background(), ticks)
	if err != nil || done != ticks {
		t.Fatalf("Run(%d) = %d, %v, want %d, nil", ticks, done, err, ticks)
	}
	runs, steps := byName(rec, "economysim.run"), byName(rec, "economysim.step")
	if len(runs) != 1 || len(steps) != ticks {
		t.Fatalf("spans = %d runs, %d steps, want 1, %d", len(runs), len(steps), ticks)
	}
	if g, ok := attrOf(runs[0], "economysim.completed"); !ok || g != ticks {
		t.Errorf("run economysim.completed = %d, %v, want %d, true", g, ok, ticks)
	}
	if runs[0].Status().Code == codes.Error {
		t.Errorf("run status = %v, want not an error", runs[0].Status())
	}
	var trades, volume int64
	for i, s := range steps {
		if s.Parent().SpanID() != runs[0].SpanContext().SpanID() {
			t.Errorf("step %d parent = %v, want the run span", i, s.Parent().SpanID())
		}
		if s.Status().Code == codes.Error {
			t.Errorf("step %d status = %v, want not an error", i, s.Status())
		}
		a, _ := attrOf(s, "economysim.trades")
		b, _ := attrOf(s, "economysim.volume")
		trades, volume = trades+a, volume+b
	}
	p := h.collect()
	if trades == 0 {
		t.Fatal("the run traded nothing, so the agreement below proves nothing")
	}
	if got := p.sum("market.trades"); got != trades {
		t.Errorf("market.trades = %d, step spans sum = %d, want equal", got, trades)
	}
	if got := p.sum("market.volume"); got != volume {
		t.Errorf("market.volume = %d, step spans sum = %d, want equal", got, volume)
	}
	if got := p.sum("world.step.duration"); got != ticks {
		t.Errorf("world.step.duration samples = %d, want %d", got, ticks)
	}
	if got := p.sum("world.step.failed"); got != 0 {
		t.Errorf("world.step.failed = %d, want 0", got)
	}
	if _, ok := p["economysim.role.agents"]; ok {
		t.Error("role.agents present before any snapshot, want absent")
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	h.obs.RecordSnapshot(snap)
	if p = h.collect(); p.sum("role.agents") == 0 || p.sum("money.supply") == 0 {
		t.Errorf("after RecordSnapshot role.agents = %d, money.supply = %d, want both above 0", p.sum("role.agents"), p.sum("money.supply"))
	}
}

// TestEndToEnd_CancelledRunEndsItsSpans verifies that a Run cancelled between
// ticks ends the run span with cancelled true and no error status, leaves every
// completed step as a closed child, and leaves no span open.
//
// Requirement: FR-OBS-055
// Guard: a cancelled run is not a failure; a span left open or marked as an error
// here would page an operator for a normal shutdown.
func TestEndToEnd_CancelledRunEndsItsSpans(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ended := 0
	// The canceller is a second observer, so the adapter sees the events exactly
	// as in a run that a consumer cancels from its own observer.
	canceller := economysim.ObserverFunc(func(e economysim.Event) {
		if _, ok := e.(economysim.TickEnded); ok {
			if ended++; ended == 3 {
				cancel()
			}
		}
	})
	w, _, rec := e2e(t, economysim.WithObserver(canceller))
	done, err := w.Run(ctx, 20)
	if err == nil || done != 3 {
		t.Fatalf("Run(cancelled after 3) = %d, %v, want 3 and a context error", done, err)
	}
	runs, steps := byName(rec, "economysim.run"), byName(rec, "economysim.step")
	if len(runs) != 1 || len(steps) != 3 {
		t.Fatalf("spans = %d runs, %d steps, want 1, 3", len(runs), len(steps))
	}
	if runs[0].Status().Code == codes.Error {
		t.Errorf("cancelled run status = %v, want not an error", runs[0].Status())
	}
	if c := runs[0].Attributes(); !hasBool(c, "economysim.cancelled", true) {
		t.Errorf("run attributes = %v, want economysim.cancelled true", c)
	}
	if open := len(rec.Started()) - len(rec.Ended()); open != 0 {
		t.Errorf("open spans = %d, want 0", open)
	}
}

// hasBool reports whether the attributes hold key with the boolean value want.
func hasBool(kvs []attribute.KeyValue, key string, want bool) bool {
	for _, kv := range kvs {
		if string(kv.Key) == key {
			return kv.Value.AsBool() == want
		}
	}
	return false
}
