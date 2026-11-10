package otelobserver

import (
	"context"
	"fmt"
	"time"

	economysim "github.com/Whealinghold/economy-sim"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// scope is the instrumentation scope name of the meter and the tracer: the
// module path (doc/rules/observability-and-logging.md 6.3).
const scope = "github.com/Whealinghold/economy-sim/otelobserver"

// Option configures an Observer.
type Option func(*options) error

type options struct {
	clock func() time.Time
}

// WithClock sets the clock that times a step. It is read twice per step, at
// TickStarted and at TickEnded, and nowhere else: the duration of a step is
// the one wall-clock reading the adapter makes. The default is time.Now; a nil
// clock keeps the default.
func WithClock(clock func() time.Time) Option {
	return func(o *options) error {
		if clock != nil {
			o.clock = clock
		}
		return nil
	}
}

// instrument holds the SDK instrument of one row of the catalogue; the field
// that is set is the one of the row's kind.
type instrument struct {
	counter   metric.Int64Counter
	upDown    metric.Int64UpDownCounter
	gauge     metric.Int64Gauge
	histogram metric.Float64Histogram
}

// Observer implements economysim.Observer on OpenTelemetry. Give it to a world
// with economysim.WithObserver. It is not safe for concurrent use.
type Observer struct {
	clock  func() time.Time
	tracer trace.Tracer
	inst   [nInstruments]instrument

	names names
	hot   hot
	cells map[cell]*state

	// touched is set when a series changed since the last flush.
	touched bool

	inStep    bool
	stepStart time.Time
}

var _ economysim.Observer = (*Observer)(nil)

// New builds an Observer that records to the instruments of mp and traces with
// tp. A nil provider is replaced by the no-op provider of the API; the global
// providers are never read. New returns an error only when the meter refuses
// to create an instrument; after it nothing returns an error or panics.
func New(mp metric.MeterProvider, tp trace.TracerProvider, opts ...Option) (*Observer, error) {
	cfg := options{clock: time.Now}
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}
	if mp == nil {
		mp = metricnoop.NewMeterProvider()
	}
	if tp == nil {
		tp = tracenoop.NewTracerProvider()
	}
	o := &Observer{clock: cfg.clock, tracer: tp.Tracer(scope), cells: map[cell]*state{}}
	meter := mp.Meter(scope)
	for i := range catalogue {
		in, err := build(meter, &catalogue[i])
		if err != nil {
			return nil, fmt.Errorf("create instrument %s: %w", catalogue[i].name, err)
		}
		o.inst[i] = in
	}
	return o, nil
}

// build creates the instrument of one catalogue row.
func build(m metric.Meter, s *spec) (in instrument, err error) {
	unit, desc := metric.WithUnit(s.unit), metric.WithDescription(s.desc)
	switch s.kind {
	case kCounter:
		in.counter, err = m.Int64Counter(s.name, unit, desc)
	case kUpDown:
		in.upDown, err = m.Int64UpDownCounter(s.name, unit, desc)
	case kGauge:
		in.gauge, err = m.Int64Gauge(s.name, unit, desc)
	case kHistogram:
		in.histogram, err = m.Float64Histogram(s.name, unit, desc)
	}
	return in, err
}

// bg is the context of every SDK call: the adapter has no context of its own
// and the world's calls carry none.
var bg = context.Background()
