package otelobserver

import (
	"math"
	"math/bits"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// maxInterned is the most distinct values of an open string (a money flow ID, a
// reason) that the adapter keeps for the attribute economysim.reason; a value
// beyond it drops the sample (plan 2-22 D645).
const maxInterned = 1024

// cell is one series of an instrument: the instrument and the dense index of
// each of its attributes, in the order of the row in the catalogue. An ID is its
// own index, a constant of a closed set is its place in closed, and an open
// string is interned.
type cell struct {
	inst uint8
	k    [4]int32
}

// closed holds the constants of the closed sets of attribute values, by
// attribute key (spec-observability.md 6.5.1). The place of a constant is its
// index. It is read, never written.
var closed = map[string][]string{
	keyBound:       {"floor", "ceiling"},
	keyBeliefLimit: {"conflict", "clamped"},
	keyWorkDecline: {"rest", "no_recipe"},
	keyCause:       {"insolvent", "starved", "consumer", "population", "transport", "migration"},
	keyEntryLimit: {"unfunded", "no_viable_role", "region_cap", "role_cap", "role_ceiling",
		"cooldown", "hysteresis", "exit_cap", "refused", "stranded"},
	keyKind:    {"faucet", "sink"},
	keySide:    {"bid", "ask"},
	keyOutcome: {"offered", "no_path", "too_old", "not_yet"},
	keyLimit: {"no_estimate", "no_destination", "no_rumour", "too_far", "no_viable", "no_gain",
		"bound", "out_cap", "unprovisioned", "in_cap", "held", "ended"},
	keyHolder: {"agents", "estate", "treasury"},
	keySector: {"households", "pool", "contract", "treasury"},
	keyDecision: {"no_rumour", "stale", "locked", "perishable", "no_margin", "unaffordable",
		"below_share", "planned", "rested", "waiting", "lot_capped"},
}

// The places of the causes that the instruments of the earlier modules use.
const (
	causeInsolvent = iota
	causeStarved
	causeConsumer
	causePopulation
	causeTransport
	causeMigration
)

// aliasKey is the attribute key that reads the table of another: the two ends of
// a relocation and the outside node are regions.
var aliasKey = map[string]string{keyRegionFrom: keyRegion, keyRegionTo: keyRegion, keyNode: keyRegion}

// closedIndex is the place of s in the closed set of key, or -1.
func closedIndex(key, s string) int32 {
	for i, c := range closed[key] {
		if c == s {
			return int32(i)
		}
	}
	return -1
}

// ix converts the integer of an ID or a count to a dense index; a negative or
// huge value is -1, which no table holds.
func ix[T ~int | ~int32 | ~int64](v T) int32 {
	if v < 0 || int64(v) > 1<<30 {
		return -1
	}
	return int32(v)
}

// The places of the bounds, in the closed set of economysim.bound.
const (
	boundFloor = iota
	boundCeiling
)

// m0 to m4 build the cell of an instrument with that many attributes.
func m0(inst int) cell                   { return mk(inst, 0, 0, 0, 0) }
func m1(inst int, a int32) cell          { return mk(inst, a, 0, 0, 0) }
func m2(inst int, a, b int32) cell       { return mk(inst, a, b, 0, 0) }
func m3(inst int, a, b, c int32) cell    { return mk(inst, a, b, c, 0) }
func m4(inst int, a, b, c, d int32) cell { return mk(inst, a, b, c, d) }

// ratioBP is n * 10 000 / d, truncated toward zero, in 128 bits so that it cannot
// wrap; 0 for a d that is not above 0 or a negative n, and the largest int64 if
// the ratio does not fit (architecture.md 3.5).
func ratioBP(n, d int64) int64 {
	if n < 0 || d <= 0 {
		return 0
	}
	hi, lo := bits.Mul64(uint64(n), 10_000)
	if hi >= uint64(d) {
		return math.MaxInt64
	}
	q, _ := bits.Div64(hi, lo, uint64(d))
	if q > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(q)
}

// mk builds the cell of an instrument from the indices of its attributes.
func mk(inst int, a, b, c, d int32) cell {
	return cell{inst: uint8(inst), k: [4]int32{a, b, c, d}}
}

// attrTable builds the attribute of every place of key: the names of an ID kind
// (strs, by the key of the kind) and the constants of a closed set.
func attrTable(key string, strs map[string][]string) []attribute.KeyValue {
	if base, ok := aliasKey[key]; ok {
		return kvs(key, strs[base])
	}
	if s, ok := strs[key]; ok {
		return kvs(key, s)
	}
	return kvs(key, closed[key])
}

// attr is the attribute of place i of key, and false if there is none.
func (n *names) attr(key string, i int32) (attribute.KeyValue, bool) {
	t := n.tab[key]
	if key == keyReason {
		t = n.other.kv
	}
	if i < 0 || int(i) >= len(t) {
		return attribute.KeyValue{}, false
	}
	return t[i], true
}

// state is what the adapter keeps for a series that is not a hot accumulator.
// n is the sum a counter has not sent, the level of an up-down counter, or the
// last value of a gauge.
type state struct {
	opt   metric.MeasurementOption
	n     int64
	sent  int64 // up-down counter: what the SDK holds
	dirty bool  // gauge: set since it was last recorded
}

// state returns the state of a series, creating it the first time; it is nil if
// an attribute of the cell is outside its table, so the sample is dropped
// (spec-observability.md 6.6.4).
func (o *Observer) state(c cell) *state {
	if s, ok := o.cells[c]; ok {
		return s
	}
	keys := catalogue[c.inst].attrs
	kv := make([]attribute.KeyValue, len(keys))
	for i, key := range keys {
		a, ok := o.names.attr(key, c.k[i])
		if !ok {
			return nil
		}
		kv[i] = a
	}
	s := &state{opt: metric.WithAttributeSet(attribute.NewSet(kv...))}
	o.cells[c] = s
	return s
}

// count adds v to the counter or the level of a series.
func (o *Observer) count(c cell, v int64) {
	if v == 0 || (v < 0 && catalogue[c.inst].kind == kCounter) { // a counter only increases
		return
	}
	if s := o.state(c); s != nil {
		s.n += v
		o.touched = true
	}
}

// set records the last value of a gauge or sets the level of an up-down counter.
func (o *Observer) set(c cell, v int64) {
	if s := o.state(c); s != nil {
		s.n, s.dirty, o.touched = v, true, true
	}
}

// counts adds a count for each place of the closed set that the attribute at pos
// of the cell takes: vs[i] is the count of place i.
func (o *Observer) counts(c cell, pos int, vs ...int64) {
	for i, v := range vs {
		c.k[pos] = int32(i)
		o.count(c, v)
	}
}

// reason is the place of an open reason string in the table of the attribute
// economysim.reason, or -1 beyond the cap.
func (o *Observer) reason(s string) int32 {
	return int32(o.names.other.id(s))
}

// heldGauge marks the gauges that keep their last value per series and are
// recorded again at every TickEnded, so that a series whose event was silent in
// a tick is not stale (spec-observability.md 6.3.6).
var heldGauge = [nInstruments]bool{
	iMarketClearingPrice: true, iBeliefCentre: true, iBeliefAtBound: true,
	iEndowmentStock: true, iHouseholdOwnUse: true, iExternalPrice: true,
}

// flushCells sends to the SDK what the series hold: each counter's sum once, each
// up-down counter's change since the last send once, and each gauge's value if it
// was set (all=true: and every held gauge). It is called at TickEnded, and after
// an event outside a step.
func (o *Observer) flushCells(all bool) {
	if !o.touched && !all {
		return
	}
	for c, s := range o.cells {
		s.send(&o.inst[c.inst], catalogue[c.inst].kind, all && heldGauge[c.inst])
	}
	o.touched = false
}

// send sends what a series holds to its instrument; again records a gauge that
// was not set since it was last recorded.
func (s *state) send(in *instrument, k kind, again bool) {
	switch k {
	case kCounter:
		if s.n != 0 {
			in.counter.Add(bg, s.n, s.opt)
			s.n = 0
		}
	case kUpDown:
		if d := s.n - s.sent; d != 0 {
			in.upDown.Add(bg, d, s.opt)
			s.sent = s.n
		}
	case kGauge:
		if s.dirty || again {
			in.gauge.Record(bg, s.n, s.opt)
			s.dirty = false
		}
	}
}
