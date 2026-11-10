// Package otelobserver is an economysim.Observer that reports a world to
// OpenTelemetry: metrics from the events and from the snapshots the consumer
// passes to RecordSnapshot, and a span for each run and each step.
//
// It is a separate module so that the core imports no OpenTelemetry package
// (doc/rules/architecture.md 11.4). The consumer supplies the MeterProvider and
// the TracerProvider; the adapter never reads a global one, and a nil provider
// means the no-op one. The instrument scope name of both is this module's path.
//
// An Observer is not safe for concurrent use, like the world it observes: the
// world calls it on the goroutine that steps. It holds no World, never calls
// back into one, logs nothing and returns no error after New.
//
// # Limitation: the intern caps
//
// A value of economysim.reason beyond the cap of 1024 distinct values (32 for the
// reasons of a rejected order) is dropped, and nothing counts the drops: the
// instrument set is fixed by the catalogue of spec-observability.md, so a drop
// counter would be a change of the spec, not of the adapter.
//
// # Limitation: held gauges
//
// A held gauge (economysim.market.clearing_price, belief.centre, belief.at_bound,
// endowment.stock, household.own_use and external.price) keeps its last value and
// is recorded again at every step. OpenTelemetry has no call that withdraws a
// gauge point, so after a second WorldCreated or WorldRestored on the same
// Observer the series of the old names table stay in the SDK at their last value,
// while the up-down counters are brought back by their difference. Give each world
// its own Observer (and its own reader or resource) if the series of an earlier
// world must not remain.
package otelobserver
