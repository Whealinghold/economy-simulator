package otelobserver

import (
	"context"

	economysim "github.com/Whealinghold/economy-sim"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// spans holds the open run span, the context that parents the steps under it,
// and the open step span (spec-observability.md 6.6.5). The adapter keeps one of
// each at most; a span that is still open when the next of its kind starts
// belonged to a panicked step and is ended with an error.
type spans struct {
	runCtx   context.Context
	run, stp trace.Span
}

// endSpan ends a span; an error status is set first when failed.
func endSpan(s trace.Span, failed bool, why string) {
	if s == nil {
		return
	}
	if failed {
		s.SetStatus(codes.Error, why)
	}
	s.End()
}

func (o *Observer) runStarted(ev economysim.RunStarted) {
	endSpan(o.sp.stp, true, "step did not end")
	endSpan(o.sp.run, true, "run did not end")
	o.sp.stp = nil
	o.inStep = false // a step that panicked has no TickEnded; its accumulators must flush again
	o.sp.runCtx, o.sp.run = o.tracer.Start(bg, "economysim.run",
		trace.WithAttributes(attribute.Int64("economysim.ticks", ev.Ticks)))
}

func (o *Observer) runEnded(ev economysim.RunEnded) {
	endSpan(o.sp.stp, true, "step did not end")
	o.sp.stp = nil
	o.inStep = false // settle then flushes the accumulators of a step that panicked
	if o.sp.run != nil {
		o.sp.run.SetAttributes(attribute.Int64("economysim.completed", ev.Completed),
			attribute.Bool("economysim.cancelled", ev.Cancelled))
	}
	endSpan(o.sp.run, ev.Failed, "run failed")
	o.sp.run, o.sp.runCtx = nil, nil
}

// stepStarted opens the step span, a child of the run span if a run is open.
func (o *Observer) stepStarted(tick int64) {
	endSpan(o.sp.stp, true, "step did not end")
	ctx := bg
	if o.sp.run != nil {
		ctx = o.sp.runCtx
	}
	_, o.sp.stp = o.tracer.Start(ctx, "economysim.step", trace.WithAttributes(attribute.Int64("economysim.tick", tick)))
}

// stepEnded gives the step span the counts of the step, which are still in the
// hot accumulators, and ends it. It runs before the accumulators are flushed.
func (o *Observer) stepEnded(failed bool) {
	if o.sp.stp == nil {
		return
	}
	h := &o.hot
	o.sp.stp.SetAttributes(
		attribute.Int64("economysim.orders.placed", h.orders.total()),
		attribute.Int64("economysim.orders.rejected", h.rejected.total()),
		attribute.Int64("economysim.trades", h.trades.total()),
		attribute.Int64("economysim.volume", h.volume.total()))
	endSpan(o.sp.stp, failed, "step failed")
	o.sp.stp = nil
}

// total is the sum of the cells that have not been flushed.
func (s *series) total() (t int64) {
	for _, v := range s.n {
		t += v
	}
	return t
}

// spanEvent adds a rare entity event to the step span, or to the run span if it
// came outside a step, or drops it. Orders, trades and recipe executions are
// never span events: they are too many (rules 6.2).
func (o *Observer) spanEvent(e economysim.Event) {
	target := o.sp.run
	if o.inStep && o.sp.stp != nil {
		target = o.sp.stp
	}
	if target == nil || !target.IsRecording() {
		return
	}
	name, a, ok := o.agentSpan(e)
	if !ok {
		name, a, ok = o.flowSpan(e)
	}
	if ok {
		target.AddEvent(name, trace.WithAttributes(a...))
	}
}

// attrs builds the attributes of a span event: a name from a table by index, a
// count or a string. An index outside its table leaves the attribute out.
type attrs []attribute.KeyValue

func (a attrs) name(tab []attribute.KeyValue, key string, id int) attrs {
	if id < 0 || id >= len(tab) {
		return a
	}
	return append(a, attribute.String(key, tab[id].Value.AsString()))
}

func (a attrs) num(key string, v int64) attrs { return append(a, attribute.Int64(key, v)) }

func (a attrs) str(key, v string) attrs { return append(a, attribute.String(key, v)) }

func (o *Observer) agentSpan(e economysim.Event) (string, attrs, bool) {
	n := &o.names
	switch ev := e.(type) {
	case economysim.AgentSpawned:
		return "economysim.agent_spawned", attrs{}.name(n.region, keyRegion, int(ev.Region)).
			name(n.role, keyRole, int(ev.Role)).num("economysim.weight", ev.Weight), true
	case economysim.AgentBankrupted:
		return "economysim.agent_bankrupted", attrs{}.name(n.region, keyRegion, int(ev.Region)).
			name(n.role, keyRole, int(ev.Role)).num("economysim.weight", ev.Weight).str(keyCause, string(ev.Cause)), true
	case economysim.WeightChanged:
		return "economysim.weight_changed", attrs{}.name(n.region, keyRegion, int(ev.Region)).
			name(n.role, keyRole, int(ev.Role)).num("economysim.from", ev.From).num("economysim.to", ev.To).
			str(keyCause, string(ev.Cause)), true
	case economysim.RoleChanged:
		return "economysim.role_changed", attrs{}.name(n.region, keyRegion, int(ev.Region)).
			name(n.role, "economysim.role.from", int(ev.From)).name(n.role, "economysim.role.to", int(ev.To)).
			num("economysim.weight", ev.Weight).str(keyCause, string(ev.Cause)), true
	case economysim.AgentRelocated:
		return "economysim.agent_relocated", attrs{}.name(n.region, keyRegionFrom, int(ev.From)).
			name(n.region, keyRegionTo, int(ev.To)).num("economysim.weight", ev.Weight).str(keyCause, string(ev.Cause)), true
	}
	return "", nil, false
}

func (o *Observer) flowSpan(e economysim.Event) (string, attrs, bool) {
	n := &o.names
	switch ev := e.(type) {
	case economysim.ShipmentDispatched:
		return "economysim.shipment_dispatched", attrs{}.name(n.region, keyRegionFrom, int(ev.Region)).
			name(n.region, keyRegionTo, int(ev.To)).name(n.mode, keyMode, int(ev.Mode)).num("economysim.units", ev.Units), true
	case economysim.ShipmentArrived:
		return "economysim.shipment_arrived", attrs{}.name(n.region, keyRegionFrom, int(ev.Region)).
			name(n.region, keyRegionTo, int(ev.To)).name(n.mode, keyMode, int(ev.Mode)), true
	case economysim.ContractFormed:
		return "economysim.contract_formed", attrs{}.name(n.region, keyRegionFrom, int(ev.Region)).
			name(n.region, keyRegionTo, int(ev.To)).name(n.commodity, keyCommodity, int(ev.Commodity)).
			num("economysim.quantity", int64(ev.Quantity)), true
	case economysim.ContractSettled:
		return "economysim.contract_settled", attrs{}.name(n.region, keyRegionFrom, int(ev.Region)).
			name(n.region, keyRegionTo, int(ev.To)).name(n.commodity, keyCommodity, int(ev.Commodity)).
			num("economysim.quantity", int64(ev.Reached)).str(keyCause, string(ev.Cause)), true
	case economysim.MigrationStarted:
		return "economysim.migration_started", attrs{}.name(n.region, keyRegionFrom, int(ev.From)).
			name(n.region, keyRegionTo, int(ev.To)).num("economysim.weight", ev.Weight), true
	}
	return "", nil, false
}
