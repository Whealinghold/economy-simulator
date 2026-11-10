package otelobserver_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/otelobserver"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// failingMeterProvider records that it was asked for a meter.
type failingMeterProvider struct {
	metricnoop.MeterProvider
	used *bool
}

func (p failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	*p.used = true
	return metricnoop.NewMeterProvider().Meter("")
}

type failingTracerProvider struct {
	tracenoop.TracerProvider
	used *bool
}

func (p failingTracerProvider) Tracer(string, ...trace.TracerOption) trace.Tracer {
	*p.used = true
	return tracenoop.NewTracerProvider().Tracer("")
}

// TestNew_NilProvidersAreNoop verifies that nil providers give a working no-op
// adapter and that the global providers are never read: the test installs
// global providers that record any use.
//
// Requirement: FR-OBS-056
// Guard: architecture.md 11.2 bars the global providers; the global one is the
// easy mistake of a library that wants "the" provider.
func TestNew_NilProvidersAreNoop(t *testing.T) {
	var usedMeter, usedTracer bool
	otel.SetMeterProvider(failingMeterProvider{used: &usedMeter})
	otel.SetTracerProvider(failingTracerProvider{used: &usedTracer})
	t.Cleanup(func() {
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
	})
	obs, err := otelobserver.New(nil, nil)
	if err != nil {
		t.Fatalf("New(nil, nil) error = %v, want nil", err)
	}
	obs.Observe(allEvents[0])
	if usedMeter || usedTracer {
		t.Errorf("New(nil, nil) used the global meter provider = %v, tracer provider = %v, want neither", usedMeter, usedTracer)
	}
}

// failingMeter fails to create the instrument named failName.
type failingMeter struct {
	metric.Meter
	failName string
}

var errInstrument = errors.New("instrument refused")

func (m failingMeter) Int64Counter(name string, o ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	if name == m.failName {
		return nil, errInstrument
	}
	return m.Meter.Int64Counter(name, o...)
}

type failingProvider struct {
	metric.MeterProvider
	failName string
}

func (p failingProvider) Meter(name string, o ...metric.MeterOption) metric.Meter {
	return failingMeter{p.MeterProvider.Meter(name, o...), p.failName}
}

// TestNew_InstrumentErrorIsReturned verifies that New returns the error of an
// instrument the meter refuses, wrapped so that errors.Is finds it, and no
// adapter.
//
// Requirement: FR-OBS-050
func TestNew_InstrumentErrorIsReturned(t *testing.T) {
	mp := failingProvider{metricnoop.NewMeterProvider(), "economysim.market.trades"}
	obs, err := otelobserver.New(mp, nil)
	if !errors.Is(err, errInstrument) || obs != nil {
		t.Errorf("New(refusing meter) = %v, %v, want nil and %v", obs, err, errInstrument)
	}
}
