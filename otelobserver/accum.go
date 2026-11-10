package otelobserver

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// series is a dense array of counts, one cell per series of an instrument, with
// the measurement option of each cell built the first time the cell is flushed.
// The SDK is called once per non-zero cell per flush, however many events the
// step had (spec-observability.md 6.6.2).
type series struct {
	n     []int64
	opt   []metric.MeasurementOption
	attrs func(i int) []attribute.KeyValue
}

func newSeries(size int, attrs func(i int) []attribute.KeyValue) series {
	return series{n: make([]int64, size), opt: make([]metric.MeasurementOption, size), attrs: attrs}
}

// add adds v to cell i; a cell outside the array drops the sample.
func (s *series) add(i int, v int64) {
	if i >= 0 && i < len(s.n) {
		s.n[i] = satAdd(s.n[i], v)
	}
}

// flush adds each non-zero cell to the counter once and zeroes it.
func (s *series) flush(c metric.Int64Counter) {
	for i, v := range s.n {
		if v == 0 {
			continue
		}
		if s.opt[i] == nil {
			s.opt[i] = metric.WithAttributeSet(attribute.NewSet(s.attrs(i)...))
		}
		c.Add(bg, v, s.opt[i])
		s.n[i] = 0
	}
}

// hot holds the accumulators of the instruments fed by tens of thousands of
// events a tick (plan 2-22 D646).
type hot struct {
	orders, rejected, trades, volume, estateSold series
	nCommodity, nRegion                          int
}

// newHot sizes the accumulators from the names table; the radix of a cell is
// the size of each of its attribute tables.
func newHot(n *names) hot {
	nc, nr := len(n.commodity), len(n.region)
	return hot{
		nCommodity: nc,
		nRegion:    nr,
		orders: newSeries(nc*nr*2, func(i int) []attribute.KeyValue {
			return []attribute.KeyValue{n.commodity[i/(nr*2)], n.region[i/2%nr], sideAttr[i%2]}
		}),
		rejected: newSeries(nc*maxReasons, func(i int) []attribute.KeyValue {
			return []attribute.KeyValue{n.commodity[i/maxReasons], n.reasons.kv[i%maxReasons]}
		}),
		trades: newSeries(nc, func(i int) []attribute.KeyValue { return []attribute.KeyValue{n.commodity[i]} }),
		volume: newSeries(nc, func(i int) []attribute.KeyValue { return []attribute.KeyValue{n.commodity[i]} }),
		estateSold: newSeries(nc*nr, func(i int) []attribute.KeyValue {
			return []attribute.KeyValue{n.commodity[i/nr], n.region[i%nr]}
		}),
	}
}

// flushHot sends every non-zero hot cell to its instrument.
func (o *Observer) flushHot() {
	o.hot.orders.flush(o.inst[iMarketOrders].counter)
	o.hot.rejected.flush(o.inst[iMarketOrdersRejected].counter)
	o.hot.trades.flush(o.inst[iMarketTrades].counter)
	o.hot.volume.flush(o.inst[iMarketVolume].counter)
	o.hot.estateSold.flush(o.inst[iEstateSold].counter)
}

// total is the sum of the cells that have not been flushed.
func (s *series) total() (t int64) {
	for _, v := range s.n {
		t = satAdd(t, v)
	}
	return t
}
