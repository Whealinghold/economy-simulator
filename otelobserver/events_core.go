package otelobserver

import (
	economysim "github.com/Whealinghold/economy-sim"
)

// Observe implements economysim.Observer. It type-switches on the three hot
// events first, then on the groups of handlers in a fixed chain; an event it
// does not know is ignored, so a new event is not a breaking change.
func (o *Observer) Observe(e economysim.Event) {
	switch ev := e.(type) {
	case economysim.OrderPlaced:
		o.orderPlaced(ev)
	case economysim.OrderRejected:
		o.orderRejected(ev)
	case economysim.TradeExecuted:
		o.tradeExecuted(ev)
	default:
		o.observeLifecycle(e)
	}
}

// observeLifecycle handles the events of the world and the step, and reports
// whether it handled e.
func (o *Observer) observeLifecycle(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.WorldCreated:
		o.reset(ev.Names)
	case economysim.WorldRestored:
		o.reset(ev.Names)
	case economysim.TickStarted:
		o.tickStarted()
	case economysim.TickEnded:
		o.tickEnded(ev)
	default:
		return false
	}
	return true
}

// reset rebuilds the names table and the accumulators for a world, which
// forgets what a previous world left.
func (o *Observer) reset(n economysim.Names) {
	o.names = newNames(n)
	o.hot = newHot(&o.names)
	o.inStep = false
}

// settle sends the accumulators at once when the event arrived outside a step:
// a call between ticks has no TickEnded to flush it (plan 2-22 D647).
func (o *Observer) settle() {
	if !o.inStep {
		o.flushHot()
	}
}

func (o *Observer) orderPlaced(ev economysim.OrderPlaced) {
	if ev.Side != economysim.Bid && ev.Side != economysim.Ask {
		return
	}
	nr := o.hot.nRegion
	if ev.Commodity < 0 || int(ev.Commodity) >= o.hot.nCommodity || ev.Region < 0 || int(ev.Region) >= nr {
		return
	}
	o.hot.orders.add((int(ev.Commodity)*nr+int(ev.Region))*2+int(ev.Side), 1)
	o.settle()
}

func (o *Observer) orderRejected(ev economysim.OrderRejected) {
	if ev.Commodity < 0 || int(ev.Commodity) >= o.hot.nCommodity {
		return
	}
	r := o.names.reasons.id(string(ev.Reason))
	if r < 0 {
		return
	}
	o.hot.rejected.add(int(ev.Commodity)*maxReasons+r, 1)
	o.settle()
}

// tradeExecuted counts a trade and its volume, and the volume again as an
// estate's sale where the seller is below the region count: an estate's account
// number is its region's (spec-observability.md 4.3.3, N-310).
func (o *Observer) tradeExecuted(ev economysim.TradeExecuted) {
	nr := o.hot.nRegion
	if ev.Commodity < 0 || int(ev.Commodity) >= o.hot.nCommodity || ev.Quantity <= 0 {
		return
	}
	c := int(ev.Commodity)
	o.hot.trades.add(c, 1)
	o.hot.volume.add(c, int64(ev.Quantity))
	if ev.Seller >= 0 && int(ev.Seller) < nr && ev.Region >= 0 && int(ev.Region) < nr {
		o.hot.estateSold.add(c*nr+int(ev.Region), int64(ev.Quantity))
	}
	o.settle()
}

func (o *Observer) tickStarted() {
	o.inStep = true
	o.stepStart = o.clock()
}

// tickEnded flushes what the step accumulated. A failed step records no
// duration and counts one failure; the other records its duration (FR-OBS-058).
func (o *Observer) tickEnded(ev economysim.TickEnded) {
	switch {
	case ev.Failed:
		o.inst[iWorldStepFailed].counter.Add(bg, 1)
	case o.inStep:
		o.inst[iWorldStepDuration].histogram.Record(bg, o.clock().Sub(o.stepStart).Seconds())
	}
	o.inStep = false
	o.flushHot()
}
