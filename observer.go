package economysim

import "slices"

// Observer receives the events of a world. It has one method, so that adding
// an event is not a breaking change for an implementer.
//
// The calling contract:
//
//   - Observe is called synchronously on the goroutine that steps the world,
//     once per event and per observer, in the order the observers were given to
//     WithObserver. Events are emitted after the transition they describe is
//     complete, in a deterministic order.
//   - An observer MUST NOT call back into the World: no step, query, snapshot
//     or call of any kind. The world is in the middle of a call.
//   - An observer MUST ignore event types it does not recognise, because a new
//     version of the library may add one.
//   - An observer MAY keep an event: its slices are owned by the event.
//   - The library does not recover a panicking observer. The panic propagates
//     to the caller of Step.
//   - A slow observer is the client's problem. Observation is synchronous, so a
//     step lasts as long as every observer takes. Return quickly, and hand any
//     work to a goroutine of your own if it needs one.
//   - An observer never influences the simulation: it must not draw from a
//     random source or depend on wall-clock time to decide anything the world
//     sees.
type Observer interface {
	Observe(Event)
}

// ObserverFunc adapts a function to the Observer interface. A nil ObserverFunc
// given to WithObserver is ignored, as a nil Observer is.
type ObserverFunc func(Event)

// Observe calls f(e).
func (f ObserverFunc) Observe(e Event) { f(e) }

// discardObserver is the default observer: it discards every event. It is not
// exported because no consumer needs a name for "no observer", which is no
// option or a nil one (architecture.md 4.5). The world builds no event for it:
// with no observer supplied the observers value reports supplied false.
type discardObserver struct{}

// Observe discards e.
func (discardObserver) Observe(Event) {}

var _ Observer = discardObserver{}

// warnRun is the run length of a bounded Warn: a condition that persists is
// logged when its run of consecutive ticks reaches 10 and at every multiple of
// 10 after (spec-observability.md 7.3). It is a rule constant, not a tunable:
// it changes no result.
const warnRun = 10

// observers is the observers of a world, in the order they were given, and the
// flag that says whether any was supplied. The world builds it once at
// construction and never changes it, so the flag is read, not computed, at
// every site that would build an event (observability-and-logging.md 4.5).
type observers struct {
	list     []Observer
	supplied bool
}

// newObservers builds the observers of a world from the observers given to
// WithObserver, in order. A nil observer, and a nil ObserverFunc, is ignored
// and never stored. The given slice is copied. supplied is true if any
// observer remains.
func newObservers(given []Observer) observers {
	var o observers
	for _, g := range given {
		if g == nil {
			continue
		}
		if f, ok := g.(ObserverFunc); ok && f == nil {
			continue
		}
		o.list = append(o.list, g)
	}
	o.list = slices.Clip(o.list)
	o.supplied = len(o.list) > 0
	return o
}

// deliver passes e to each observer in order. It does not allocate; a caller
// that builds e does so only when o.supplied, since boxing a struct into an
// Event allocates. A panic of an observer propagates.
func (o *observers) deliver(e Event) {
	for _, ob := range o.list {
		ob.Observe(e)
	}
}
