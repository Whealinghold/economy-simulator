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
package otelobserver
