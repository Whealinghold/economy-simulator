package otelobserver_test

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/otelobserver"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// fakeClock advances 1 ms at every read, so a step that reads it twice lasts
// exactly 1 ms and no test reads the wall clock (testing.md 10.2).
type fakeClock struct {
	now   time.Time
	reads int
}

func (c *fakeClock) read() time.Time {
	c.reads++
	c.now = c.now.Add(time.Millisecond)
	return c.now
}

// countProvider wraps a MeterProvider so that every instrument counts its Add
// and Record calls by name: the SDK aggregates, so only a count of calls shows
// that the adapter calls it once per series per tick (plan 2-22 D657).
type countProvider struct {
	metric.MeterProvider
	calls map[string]int
}

func (p countProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	return countMeter{p.MeterProvider.Meter(name, opts...), p.calls}
}

type countMeter struct {
	metric.Meter
	calls map[string]int
}

func (m countMeter) Int64Counter(name string, o ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	c, err := m.Meter.Int64Counter(name, o...)
	return countCounter{c, name, m.calls}, err
}

func (m countMeter) Int64UpDownCounter(name string, o ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) {
	c, err := m.Meter.Int64UpDownCounter(name, o...)
	return countUpDown{c, name, m.calls}, err
}

func (m countMeter) Int64Gauge(name string, o ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	g, err := m.Meter.Int64Gauge(name, o...)
	return countGauge{g, name, m.calls}, err
}

func (m countMeter) Float64Histogram(name string, o ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	h, err := m.Meter.Float64Histogram(name, o...)
	return countHistogram{h, name, m.calls}, err
}

type countCounter struct {
	metric.Int64Counter
	name  string
	calls map[string]int
}

func (c countCounter) Add(ctx context.Context, v int64, o ...metric.AddOption) {
	c.calls[c.name]++
	c.Int64Counter.Add(ctx, v, o...)
}

type countUpDown struct {
	metric.Int64UpDownCounter
	name  string
	calls map[string]int
}

func (c countUpDown) Add(ctx context.Context, v int64, o ...metric.AddOption) {
	c.calls[c.name]++
	c.Int64UpDownCounter.Add(ctx, v, o...)
}

type countGauge struct {
	metric.Int64Gauge
	name  string
	calls map[string]int
}

func (g countGauge) Record(ctx context.Context, v int64, o ...metric.RecordOption) {
	g.calls[g.name]++
	g.Int64Gauge.Record(ctx, v, o...)
}

type countHistogram struct {
	metric.Float64Histogram
	name  string
	calls map[string]int
}

func (h countHistogram) Record(ctx context.Context, v float64, o ...metric.RecordOption) {
	h.calls[h.name]++
	h.Float64Histogram.Record(ctx, v, o...)
}

// harness is an Observer on the SDK's in-memory reader with a fake clock and a
// call counter.
type harness struct {
	t     *testing.T
	obs   *otelobserver.Observer
	read  *sdkmetric.ManualReader
	clock *fakeClock
	calls map[string]int
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	h := &harness{t: t, read: reader, clock: &fakeClock{now: time.Unix(0, 0)}, calls: map[string]int{}}
	mp := countProvider{sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), h.calls}
	obs, err := otelobserver.New(mp, nil, otelobserver.WithClock(h.clock.read))
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	h.obs = obs
	return h
}

// send gives the events to the adapter in order.
func (h *harness) send(es ...economysim.Event) {
	for _, e := range es {
		h.obs.Observe(e)
	}
}

// points maps a metric name to its points: the attribute set, encoded, to the
// value (a histogram's value is its count).
type points map[string]map[string]int64

// collect reads every point the SDK holds.
func (h *harness) collect() points {
	h.t.Helper()
	var rm metricdata.ResourceMetrics
	if err := h.read.Collect(context.Background(), &rm); err != nil {
		h.t.Fatalf("Collect() error = %v, want nil", err)
	}
	out := points{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch d := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, p := range d.DataPoints {
					out.put(m.Name, p.Attributes, p.Value)
				}
			case metricdata.Gauge[int64]:
				for _, p := range d.DataPoints {
					out.put(m.Name, p.Attributes, p.Value)
				}
			case metricdata.Histogram[float64]:
				for _, p := range d.DataPoints {
					out.put(m.Name, p.Attributes, int64(p.Count))
				}
			}
		}
	}
	return out
}

func (p points) put(name string, set attribute.Set, v int64) {
	if p[name] == nil {
		p[name] = map[string]int64{}
	}
	p[name][set.Encoded(attribute.DefaultEncoder())] = v
}

// sum is the sum of the values of a metric over its series.
func (p points) sum(name string) int64 {
	var s int64
	for _, v := range p["economysim."+name] {
		s += v
	}
	return s
}

// get is the value of the series of a metric with the given attributes, in the
// key order of the catalogue.
func (p points) get(name string, kv ...string) (int64, bool) {
	var parts []string
	for i := 0; i+1 < len(kv); i += 2 {
		parts = append(parts, "economysim."+kv[i]+"="+kv[i+1])
	}
	sort.Strings(parts)
	v, ok := p["economysim."+name][strings.Join(parts, ",")]
	return v, ok
}

// miniNames is a names table of three commodities and three regions, two roles,
// two wants, two resources and two transport modes.
func miniNames() economysim.Names {
	return economysim.Names{
		Commodities: []string{"c0", "c1", "c2"},
		Regions:     []string{"r0", "r1", "r2"},
		Roles:       []string{"k0", "k1"},
		Recipes:     []string{"p0"},
		Wants:       []string{"w0", "w1"},
		Resources:   []string{"s0", "s1"},
		Modes:       []string{"m0", "m1"},
	}
}
