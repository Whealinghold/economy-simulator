package economysim

import (
	"errors"
	"log/slog"
	"reflect"
	"slices"
	"testing"
)

// edgeMill is threeRegionMill with five edges. IDs sort away (0), home (1),
// zone (2); the EdgeIDs are in ascending (From, To): 0 away>home 30, 1 away>zone 90
// (longer than the 60 through home), 2 home>away 30, 3 home>zone 30, 4 zone>home 30.
func edgeMill() Config {
	c := threeRegionMill()
	c.Edges = []Edge{
		{From: "away", To: "home", Distance: 30}, {From: "away", To: "zone", Distance: 90},
		{From: "home", To: "away", Distance: 30}, {From: "home", To: "zone", Distance: 30},
		{From: "zone", To: "home", Distance: 30},
	}
	return c
}

// TestEdgeStatus_AtomicAndRejected verifies that a batch with one refused change
// applies none of it: each refusal is its sentinel by errors.Is, the valid change
// listed before the bad one is not applied, and the graph, the statuses, the
// rebuild count and the events are as they were.
//
// Requirement: FR-REG-050
// Guard: applying the changes one at a time would leave the first one in force
// when the second fails.
func TestEdgeStatus_AtomicAndRejected(t *testing.T) {
	t.Parallel()
	good := EdgeStatus{From: 0, To: 1, Closed: true, FactorBP: 10_000}
	tests := []struct {
		name string
		bad  EdgeStatus
		want error
	}{
		{"an edge that is not configured", EdgeStatus{From: 0, To: 0, FactorBP: 10_000}, ErrUnknownEdge},
		{"a region that is not one", EdgeStatus{From: 1, To: 9, FactorBP: 10_000}, ErrUnknownEdge},
		{"a negative region", EdgeStatus{From: -1, To: 1, FactorBP: 10_000}, ErrUnknownEdge},
		{"a factor of 0", EdgeStatus{From: 1, To: 2}, ErrOutOfBounds},
		{"a factor past the bound", EdgeStatus{From: 1, To: 2, FactorBP: MaxEdgeFactorBP + 1}, ErrOutOfBounds},
		{"an edge twice", EdgeStatus{From: 0, To: 1, FactorBP: 20_000}, ErrDuplicateID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &evLog{}
			w := newMill(t, edgeMill(), WithObserver(rec))
			before, held := w.Graph(), w.Graph()
			rec.events = nil
			err := w.setEdgeStatuses([]EdgeStatus{good, tt.bad}, CauseConsumer)
			if !errors.Is(err, tt.want) {
				t.Fatalf("setEdgeStatuses(good, %+v) = %v, want %v", tt.bad, err, tt.want)
			}
			if w.geo.status != nil || w.geo.rebuilds != 0 || len(rec.events) != 0 {
				t.Errorf("after a refused batch: status %v, %d rebuilds, %d events, want none of them", w.geo.status, w.geo.rebuilds, len(rec.events))
			}
			if after := w.Graph(); !reflect.DeepEqual(before, after) || !reflect.DeepEqual(before, held) {
				t.Errorf("Graph() after a refused batch differs from the one before")
			}
		})
	}
}

// TestEdgeStatus_ClosedAndFactor verifies the examples of spec-regions.md 12.1 and
// 12.2 on the world's own tables: a closed edge is absent from Out, In, Distance,
// NextHop, Path, Within and the components and present in Edge with Closed set, and
// a factor lengthens an open edge by the rule of 7.2.1.
//
// Arithmetic: open, away to zone is 60 (away>home 30 + home>zone 30). Close
// away>home and put home>zone at 15 000: its length is ceil(30 * 15 000 / 10 000) =
// 45. Then away reaches home only through zone, 90 + 30 = 120, and zone directly at
// 90; home reaches zone at 45. The cycle away>zone>home>away is still open, so the
// graph keeps one component.
//
// Requirement: FR-REG-049
func TestEdgeStatus_ClosedAndFactor(t *testing.T) {
	t.Parallel()
	w := newMill(t, edgeMill())
	if d, ok := w.geo.distance(0, 2); !ok || d != 60 {
		t.Fatalf("setup: distance(away, zone) = %d, %v, want 60, true", d, ok)
	}
	if err := w.setEdgeStatuses([]EdgeStatus{{From: 0, To: 1, Closed: true, FactorBP: 10_000}, {From: 1, To: 2, FactorBP: 15_000}}, CauseConsumer); err != nil {
		t.Fatalf("setEdgeStatuses() = %v, want nil", err)
	}
	for _, g := range []Graph{w.Graph(), w.geo.g} {
		if d, ok := g.Distance(0, 1); !ok || d != 120 {
			t.Errorf("Distance(away, home) = %d, %v, want 120, true", d, ok)
		}
		if d, ok := g.Distance(0, 2); !ok || d != 90 {
			t.Errorf("Distance(away, zone) = %d, %v, want 90, true", d, ok)
		}
		if d, ok := g.Distance(1, 2); !ok || d != 45 {
			t.Errorf("Distance(home, zone) = %d, %v, want 45, true", d, ok)
		}
		if next, ok := g.NextHop(0, 1); !ok || next != 2 {
			t.Errorf("NextHop(away, home) = %d, %v, want zone (2), true", next, ok)
		}
		if p, ok := g.Path(0, 1); !ok || !slices.Equal(p, []RegionID{0, 2, 1}) {
			t.Errorf("Path(away, home) = %v, %v, want [away zone home]", p, ok)
		}
		if got, want := g.Out(0), []Hop{{Region: 2, Distance: 90}}; !slices.Equal(got, want) {
			t.Errorf("Out(away) = %v, want %v: away>home is closed", got, want)
		}
		if got, want := g.In(1), []Hop{{Region: 2, Distance: 30}}; !slices.Equal(got, want) {
			t.Errorf("In(home) = %v, want %v: away>home is closed", got, want)
		}
		if got, want := g.Within(0, 100), []Hop{{Region: 2, Distance: 90}}; !slices.Equal(got, want) {
			t.Errorf("Within(away, 100) = %v, want %v: home is 120 away", got, want)
		}
		if g.Components() != 1 {
			t.Errorf("Components() = %d, want 1", g.Components())
		}
		e, ok := g.Edge(0, 1)
		if want := (EdgeView{ID: 0, From: 0, To: 1, Base: 30, Distance: 0, Closed: true, FactorBP: 10_000}); !ok || e != want {
			t.Errorf("Edge(away, home) = %+v, %v, want %+v", e, ok, want)
		}
		if e, ok := g.EdgeAt(3); !ok || e.Distance != 45 || e.Base != 30 || e.Closed || e.FactorBP != 15_000 {
			t.Errorf("EdgeAt(3) = %+v, %v, want base 30, length 45, open, factor 15 000", e, ok)
		}
	}
}

// TestEdgeStatus_RebuildOnceAndEventOrder verifies that a batch of five changes,
// listed against the order of the edges, rebuilds the tables once and emits five
// events in ascending EdgeID after the tables are in place; that a batch equal to
// the statuses in force rebuilds nothing and emits nothing; that of a batch with one
// equal and one different status only the different one is an event; and that the
// EdgeIDs do not change.
//
// Arithmetic: the factors 11 000 to 15 000 are given to EdgeIDs 0 to 4 and listed
// 4 to 0.
//
// Requirement: FR-REG-014, FR-REG-051
func TestEdgeStatus_RebuildOnceAndEventOrder(t *testing.T) {
	t.Parallel()
	var w *World
	var atDelivery []int
	var lengthAtDelivery []int64
	rec := &evLog{}
	obs := ObserverFunc(func(e Event) {
		if ev, ok := e.(EdgeStatusChanged); ok {
			atDelivery = append(atDelivery, w.geo.rebuilds)
			view, _ := w.geo.edge(ev.From, ev.To)
			lengthAtDelivery = append(lengthAtDelivery, view.Distance)
		}
		rec.Observe(e)
	})
	w = newMill(t, edgeMill(), WithObserver(obs))
	rec.events = nil
	pairs := [][2]RegionID{{0, 1}, {0, 2}, {1, 0}, {1, 2}, {2, 1}}
	var batch []EdgeStatus
	for i := len(pairs) - 1; i >= 0; i-- {
		batch = append(batch, EdgeStatus{From: pairs[i][0], To: pairs[i][1], FactorBP: int64(11_000 + 1_000*i)})
	}
	if err := w.setEdgeStatuses(batch, CauseConsumer); err != nil {
		t.Fatalf("setEdgeStatuses(5 changes) = %v, want nil", err)
	}
	var want []Event
	for i, p := range pairs {
		want = append(want, EdgeStatusChanged{From: p[0], To: p[1], FactorBP: int64(11_000 + 1_000*i), Cause: CauseConsumer})
	}
	if !slices.Equal(rec.events, want) {
		t.Errorf("events = %+v, want %+v: one per changed edge in ascending EdgeID", rec.events, want)
	}
	if w.geo.rebuilds != 1 || !slices.Equal(atDelivery, []int{1, 1, 1, 1, 1}) {
		t.Errorf("rebuilds = %d, and at each delivery %v, want 1 and [1 1 1 1 1]: one rebuild, before the first event", w.geo.rebuilds, atDelivery)
	}
	// ceil(30 * 11 000 / 10 000) = 33, 90 * 12 000 / 10 000 = 108, 30 at 13 000 = 39, 30 at 14 000 = 42, 30 at 15 000 = 45.
	if !slices.Equal(lengthAtDelivery, []int64{33, 108, 39, 42, 45}) {
		t.Errorf("effective lengths at delivery = %v, want [33 108 39 42 45]: the tables are in place before the events", lengthAtDelivery)
	}
	for i := range pairs {
		if e, ok := w.Graph().EdgeAt(EdgeID(i)); !ok || e.From != pairs[i][0] || e.To != pairs[i][1] {
			t.Errorf("EdgeAt(%d) = %+v, %v, want the edge %v: the EdgeIDs do not change", i, e, ok, pairs[i])
		}
	}

	rec.events = nil
	if err := w.setEdgeStatuses(batch, CauseConsumer); err != nil || w.geo.rebuilds != 1 || len(rec.events) != 0 {
		t.Errorf("the same batch again = %v with %d rebuilds and %d events, want nil, 1 and 0", err, w.geo.rebuilds, len(rec.events))
	}
	mixed := []EdgeStatus{{From: 1, To: 2, FactorBP: 14_000}, {From: 2, To: 1, FactorBP: 20_000}}
	if err := w.setEdgeStatuses(mixed, CauseConsumer); err != nil {
		t.Fatalf("setEdgeStatuses(one equal, one different) = %v, want nil", err)
	}
	if want := []Event{EdgeStatusChanged{From: 2, To: 1, FactorBP: 20_000, Cause: CauseConsumer}}; !slices.Equal(rec.events, want) || w.geo.rebuilds != 2 {
		t.Errorf("events = %+v with %d rebuilds, want only %+v and 2 rebuilds", rec.events, w.geo.rebuilds, want)
	}
}

// TestEdgeStatus_EmptyAndReopened verifies that an empty batch is accepted and
// changes nothing, and that setting every edge back to open at 10 000 leaves the
// world as one that never had a status: the status table is nil again and the graph
// equals the first, so a checkpoint writes no row for it.
//
// Requirement: FR-REG-050, FR-REG-052
// Guard: a status table that stays allocated after a reopening would differ from a
// world that never had one in the position hash and in the checkpoint.
func TestEdgeStatus_EmptyAndReopened(t *testing.T) {
	t.Parallel()
	w := newMill(t, edgeMill())
	first := w.Graph()
	if err := w.setEdgeStatuses(nil, CauseConsumer); err != nil || w.geo.rebuilds != 0 {
		t.Errorf("setEdgeStatuses(nil) = %v with %d rebuilds, want nil and 0", err, w.geo.rebuilds)
	}
	closed := []EdgeStatus{{From: 0, To: 1, Closed: true, FactorBP: 10_000}}
	open := []EdgeStatus{{From: 0, To: 1, FactorBP: 10_000}}
	for _, step := range [][]EdgeStatus{closed, open} {
		if err := w.setEdgeStatuses(step, CauseConsumer); err != nil {
			t.Fatalf("setEdgeStatuses(%+v) = %v, want nil", step, err)
		}
	}
	if w.geo.status != nil || w.geo.rebuilds != 2 || !reflect.DeepEqual(first, w.Graph()) {
		t.Errorf("after close and reopen: status %v, %d rebuilds, graph equal %v, want nil, 2 and true", w.geo.status, w.geo.rebuilds, reflect.DeepEqual(first, w.Graph()))
	}
}

// TestEdgeStatus_PartitionsComponents verifies that closing the two edges that
// leave away splits the graph into two components, away alone (number 0) and home
// with zone (number 1), and that the answers of RelocationRefusal for every agent
// and region are as before: a closure does not touch the relocation of an account
// (spec-regions.md 4.5.4, 7.4).
//
// Requirement: FR-REG-049, FR-REG-027
func TestEdgeStatus_PartitionsComponents(t *testing.T) {
	t.Parallel()
	w := newMill(t, edgeMill())
	refusals := func() []string {
		var out []string
		for a := range w.state.nextAgent {
			for r := range RegionID(3) {
				reason, err := w.RelocationRefusal(a, r)
				out = append(out, reason+":"+errText(err))
			}
		}
		return out
	}
	before := refusals()
	if c := w.geo.components(); c != 1 {
		t.Fatalf("setup: components() = %d, want 1", c)
	}
	cut := []EdgeStatus{{From: 0, To: 1, Closed: true, FactorBP: 10_000}, {From: 0, To: 2, Closed: true, FactorBP: 10_000}}
	if err := w.setEdgeStatuses(cut, CauseConsumer); err != nil {
		t.Fatalf("setEdgeStatuses() = %v, want nil", err)
	}
	g := w.Graph()
	if w.geo.components() != 2 || g.Components() != 2 {
		t.Errorf("components = %d (world), %d (Graph), want 2", w.geo.components(), g.Components())
	}
	for r, want := range []int{0, 1, 1} {
		if got, ok := g.Component(RegionID(r)); !ok || got != want {
			t.Errorf("Component(%d) = %d, %v, want %d, true", r, got, ok, want)
		}
	}
	if d, ok := g.Distance(0, 1); ok {
		t.Errorf("Distance(away, home) = %d, %v, want unreachable", d, ok)
	}
	if after := refusals(); !slices.Equal(before, after) {
		t.Errorf("RelocationRefusal answers changed with the closure: %v, want %v", after, before)
	}
}

// errText is the text of an error, "" for nil.
func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TestEdgeStatus_EventAndLog verifies the event and the Debug line of a change:
// the fields are the new status and the cause given, the Tick is the world's, and
// there is one Debug line per changed edge with from_region, to_region, closed and
// cause, and no line at any other level.
//
// Requirement: FR-REG-051, FR-REG-056, FR-REG-057
func TestEdgeStatus_EventAndLog(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	cap := newCapture()
	w := newMill(t, edgeMill(), WithObserver(rec), WithLogger(slog.New(cap)))
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	rec.events = nil
	*cap.lines = nil
	if err := w.setEdgeStatuses([]EdgeStatus{{From: 0, To: 1, Closed: true, FactorBP: 12_000}}, "policy"); err != nil {
		t.Fatalf("setEdgeStatuses() = %v, want nil", err)
	}
	want := EdgeStatusChanged{Tick: 3, From: 0, To: 1, Closed: true, FactorBP: 12_000, Cause: "policy"}
	if got := of[EdgeStatusChanged](rec); !slices.Equal(got, []EdgeStatusChanged{want}) {
		t.Errorf("events = %+v, want exactly %+v", got, want)
	}
	lines := cap.take("edge status changed")
	if len(lines) != 1 || lines[0].level != slog.LevelDebug || lines[0].attrs["from_region"] != 0 || lines[0].attrs["to_region"] != 1 ||
		lines[0].attrs["closed"] != 1 || lines[0].strs["cause"] != "policy" {
		t.Errorf("\"edge status changed\" lines = %+v, want one Debug line with from_region 0, to_region 1, closed, cause policy", lines)
	}
	if len(*cap.lines) != 0 {
		t.Errorf("other lines = %+v, want none", *cap.lines)
	}
}

// TestEdgeStatus_NoEffectOnRun verifies that in a world no module reads the graph
// of, a status set before the run changes the answers of the queries and the
// event and nothing else (spec-regions.md 7.4): two worlds on one seed, one with
// the middle link closed, step 30 ticks to the same state hash outside the region
// state and to the same events but the one.
//
// Requirement: FR-REG-049, FR-REG-057
// Guard: a module that read the graph before its spec lands would move the run of
// every pack with edges.
func TestEdgeStatus_NoEffectOnRun(t *testing.T) {
	t.Parallel()
	run := func(shock bool) ([32]byte, []Event) {
		rec := &evLog{}
		w := newMill(t, edgeMill(), WithSeed(7), WithObserver(rec))
		if shock {
			if err := w.setEdgeStatuses([]EdgeStatus{{From: 0, To: 1, Closed: true, FactorBP: 10_000}, {From: 1, To: 2, FactorBP: 15_000}}, CauseConsumer); err != nil {
				t.Fatalf("setEdgeStatuses() = %v, want nil", err)
			}
		}
		for range 30 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
		}
		sum, err := hashValue(reflect.ValueOf(w), "World.geo")
		if err != nil {
			t.Fatalf("hashValue: %v", err)
		}
		return sum, slices.DeleteFunc(rec.events, func(e Event) bool { _, ok := e.(EdgeStatusChanged); return ok })
	}
	plainHash, plainEvents := run(false)
	shockHash, shockEvents := run(true)
	if plainHash != shockHash {
		t.Errorf("state hash with a status set differs from the world without one, outside World.geo")
	}
	if !reflect.DeepEqual(plainEvents, shockEvents) {
		t.Errorf("events with a status set differ from those without, outside EdgeStatusChanged (%d against %d)", len(shockEvents), len(plainEvents))
	}
}

// TestEdgeStatusChanged_NoMapNoObserverNoAlloc verifies that the event holds no
// map, pointer, slice or function (observability 4.1) and that, with no observer
// and the default logger, reporting a changed edge allocates nothing (4.5).
//
// Requirement: FR-REG-057
func TestEdgeStatusChanged_NoMapNoObserverNoAlloc(t *testing.T) {
	typ := reflect.TypeFor[EdgeStatusChanged]()
	for i := range typ.NumField() {
		switch k := typ.Field(i).Type.Kind(); k {
		case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Func, reflect.Interface, reflect.Chan:
			t.Errorf("EdgeStatusChanged.%s is a %v, want a plain value", typ.Field(i).Name, k)
		}
	}
	w := newMill(t, edgeMill())
	view, _ := w.geo.edge(0, 1)
	if got := testing.AllocsPerRun(100, func() { w.edgeStatusChanged(view, edgeState{closed: true, factorBP: 10_000}, CauseConsumer) }); got != 0 {
		t.Errorf("edgeStatusChanged with no observer allocates %v times, want 0", got)
	}
}
