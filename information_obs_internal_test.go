package economysim

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"testing"
)

// This file tests the observability of the information module (plan 2-13 Phase 5):
// the aggregate RumoursServed and the Debug line, over the fixture of
// information_helpers_internal_test.go.

// servedAt returns the RumoursServed events of a tick, in emission order.
func servedAt(rec *evLog, tick int64) []RumoursServed {
	var out []RumoursServed
	for _, e := range of[RumoursServed](rec) {
		if e.Tick == tick {
			out = append(out, e)
		}
	}
	return out
}

// TestRumoursServed_PerRegionAscending verifies that the event is emitted once per
// reader region per tick, in ascending RegionID, after the last order of phase 3
// and before the markets of phase 4, and never for a region with no informed
// view: with the carrier not informed, region c has none.
//
// Requirement: FR-INF-040
func TestRumoursServed_PerRegionAscending(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, _ := newInfoWorld(t, func(c *Config) { c.Roles[1].Informed = false }, WithObserver(rec))
	step(t, w, 4)
	for tick := range int64(4) {
		got := servedAt(rec, tick)
		var regions []RegionID
		for _, e := range got {
			regions = append(regions, e.Region)
			if e.Readers != 2 || int(e.Region) >= len(w.cfg.Regions) {
				t.Errorf("tick %d: event %+v, want 2 readers in a region of the configuration", tick, e)
			}
		}
		if want := []RegionID{infoA, infoB}; !slices.Equal(regions, want) {
			t.Errorf("tick %d: RumoursServed regions = %v, want %v (ascending, none for a region with no informed view)", tick, regions, want)
		}
	}
	lastOrder, firstServed, firstCleared := -1, -1, -1
	for i, e := range rec.events {
		switch e := e.(type) {
		case OrderPlaced:
			if e.Tick == 3 {
				lastOrder = i
			}
		case RumoursServed:
			if e.Tick == 3 && firstServed < 0 {
				firstServed = i
			}
		case MarketCleared:
			if e.Tick == 3 && firstCleared < 0 {
				firstCleared = i
			}
		}
	}
	if !(lastOrder >= 0 && lastOrder < firstServed && firstServed < firstCleared) {
		t.Errorf("event positions: last OrderPlaced %d, first RumoursServed %d, first MarketCleared %d, want that order", lastOrder, firstServed, firstCleared)
	}
	none := &evLog{}
	off, _ := newInfoWorld(t, func(c *Config) { c.Roles[0].Informed, c.Roles[1].Informed = false, false }, WithObserver(none))
	step(t, off, 2)
	if n := len(of[RumoursServed](none)); n != 0 {
		t.Errorf("a world with no informed role emitted %d RumoursServed events, want none", n)
	}
}

// TestP36_RumoursBeforeWants verifies by parsing phase_trade.go that P3.6 calls
// the information module's emitter before the demand module's, so RumoursServed
// precedes WantShortfall (spec-world.md 6, observability 4.3).
//
// Guard: the order of the two events of one position is part of the golden
// rendering; the call order is the only thing that fixes it.
//
// Requirement: FR-INF-040
func TestP36_RumoursBeforeWants(t *testing.T) {
	t.Parallel()
	f, err := parser.ParseFile(token.NewFileSet(), "phase_trade.go", nil, 0)
	if err != nil {
		t.Fatalf("ParseFile() = %v, want nil", err)
	}
	var calls []string
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "p36Aggregates" {
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					if s, ok := c.Fun.(*ast.SelectorExpr); ok {
						calls = append(calls, s.Sel.Name)
					}
				}
				return true
			})
		}
	}
	if want := []string{"p36Rumours", "p36Wants"}; !slices.Equal(calls, want) {
		t.Errorf("p36Aggregates calls %v, want %v", calls, want)
	}
}

// TestRumoursServed_CountsAddUp verifies the four counts, the readers and the
// largest lag of the events of a tick in which every result occurs.
//
// Arithmetic: the edge c to b is removed and the horizon is 3, speed 40, at tick
// 2 with two households in each region. Region a (miller, candidates b flour, b
// grain, c grain): b is lag 2, offered, twice; c has no path, so per agent 2
// offered and 1 no path: 4, 2. Region b: a is lag 2, offered twice, c no path: 4,
// 2. Region c (carrier, candidates a grain, b grain): a is lag 4, above the
// horizon (too old); b is lag 3, above the tick 2 (not yet): 2 and 2 for the two
// agents. The largest lag offered is 2 in a and b, none in c.
//
// Requirement: FR-INF-041
func TestRumoursServed_CountsAddUp(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, s := newInfoWorld(t, func(c *Config) {
		c.Edges = c.Edges[:3]
		c.Modules.Information.RumourMaxAge = 3
	}, WithObserver(rec))
	step(t, w, 3)
	want := []RumoursServed{
		{Tick: 2, Region: infoA, Readers: 2, Offered: 4, NoPath: 2, MaxLag: 2},
		{Tick: 2, Region: infoB, Readers: 2, Offered: 4, NoPath: 2, MaxLag: 2},
		{Tick: 2, Region: infoC, Readers: 2, TooOld: 2, NotYet: 2},
	}
	if got := servedAt(rec, 2); !slices.Equal(got, want) {
		t.Errorf("RumoursServed at tick 2 = %+v, want %+v", got, want)
	}
	placed := map[RegionID]int64{}
	for _, v := range viewsAt(s, 2) {
		placed[v.region] += int64(len(v.rumours))
	}
	for _, e := range servedAt(rec, 2) {
		if e.Offered != placed[e.Region] {
			t.Errorf("region %d: Offered = %d, rumours placed in the views = %d, want equal", e.Region, e.Offered, placed[e.Region])
		}
	}
}

// TestRumoursServed_ExcludesMigrationCalls verifies that a call of the rumour
// function with no tally, the form the migration pass uses, counts nothing: the
// next event is what the views alone made.
//
// Requirement: FR-INF-052
func TestRumoursServed_ExcludesMigrationCalls(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w, _ := newInfoWorld(t, nil, WithObserver(rec))
	step(t, w, 2)
	q := rumourQuery{reader: infoA, all: true, commodities: w.roles.rows[roleOf(t, w, "miller")].tradable, asOf: w.Tick() + 1}
	if rs := w.rumoursFor(nil, q, nil); len(rs) == 0 {
		t.Fatal("rumoursFor gave nothing, want a call that would be counted if it were")
	}
	if w.info.tally[infoA] != (rumourTally{}) {
		t.Errorf("tally after a call with a nil tally = %+v, want zero", w.info.tally[infoA])
	}
	step(t, w, 1)
	if got := servedAt(rec, 2)[0]; got.Readers != 2 || got.Offered+got.NoPath+got.TooOld+got.NotYet != 6 {
		t.Errorf("RumoursServed for region a = %+v, want 2 readers and the 6 candidates of their views", got)
	}
}

// TestRumoursServed_NoMapNoAlloc verifies that the event holds only integers (no
// map, pointer, slice or function) and that the emitter allocates nothing with the
// default observer.
//
// Requirement: FR-INF-042, NFR-INF-001
func TestRumoursServed_NoMapNoAlloc(t *testing.T) {
	typ := reflect.TypeOf(RumoursServed{})
	for i := range typ.NumField() {
		switch k := typ.Field(i).Type.Kind(); k {
		case reflect.Int, reflect.Int64:
		default:
			t.Errorf("RumoursServed.%s has kind %v, want an integer", typ.Field(i).Name, k)
		}
	}
	w, _ := newInfoWorld(t, nil)
	step(t, w, 3)
	if n := testing.AllocsPerRun(50, w.p36Rumours); n != 0 {
		t.Errorf("allocations per p36Rumours with the default observer = %v, want 0", n)
	}
}

// TestInformation_NonInterference verifies that a logger at Debug and an observer
// change no result: the snapshot after 25 ticks of the default strategy equals
// that of a world with neither.
//
// Requirement: FR-INF-042
func TestInformation_NonInterference(t *testing.T) {
	t.Parallel()
	build := func(watched bool) *World {
		c := infoConfig()
		c.Roles[0].Strategy, c.Roles[1].Strategy = "default", "default"
		c.Modules.Information.RumourErrorBP = 500
		opts := []Option{WithSeed(3), WithAudit()}
		if watched {
			opts = append(opts, WithObserver(&evLog{}), WithLogger(slog.New(newCapture())))
		}
		return newMill(t, c, opts...)
	}
	a, b := build(true), build(false)
	step(t, a, 25)
	step(t, b, 25)
	sa, errA := a.Snapshot()
	sb, errB := b.Snapshot()
	if errA != nil || errB != nil {
		t.Fatalf("Snapshot() = %v, %v, want nil", errA, errB)
	}
	if !reflect.DeepEqual(sa, sb) {
		t.Error("the snapshot of a watched world differs from that of an unwatched one, want equal")
	}
}

// TestLogs_RumoursServed verifies the Debug line of an informed view: one per
// informed view with the keys agent, region, rumours and lag, nothing at Info or
// above inside the tick, and nothing built when Debug is off.
//
// Arithmetic: at tick 5 the carrier 7 in region c has two candidates, grain in a
// (lag 4) and grain in b (lag 3), so 2 rumours and the largest lag 4.
//
// Requirement: FR-INF-043
func TestLogs_RumoursServed(t *testing.T) {
	c := newCapture()
	w, _ := newInfoWorld(t, nil, WithLogger(slog.New(c)))
	*c.lines = nil
	step(t, w, 6)
	var served []logRec
	for _, l := range *c.lines {
		if l.level >= slog.LevelInfo {
			t.Errorf("a line at %v inside the tick: %q", l.level, l.msg)
		}
		if l.msg == "rumours served" {
			served = append(served, l)
		}
	}
	if len(served) != 6*6 {
		t.Fatalf("lines %q = %d, want one per informed view, 6 agents over 6 ticks = 36", "rumours served", len(served))
	}
	last := served[len(served)-2] // agent 7, the second to last of tick 5
	if last.level != slog.LevelDebug || last.attrs["agent"] != 7 || last.attrs["region"] != int64(infoC) || last.attrs["rumours"] != 2 || last.attrs["lag"] != 4 {
		t.Errorf("last line but one = %+v, want Debug with agent 7, region 2, rumours 2, lag 4", last)
	}

	quiet := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	q, _ := newInfoWorld(t, nil, WithLogger(quiet))
	step(t, q, 6)
	v := AgentView{Region: infoA}
	if n := testing.AllocsPerRun(50, func() { v.Rumours = nil; q.fillRumours(&v, &q.agents[3]) }); n != 0 {
		t.Errorf("allocations per view with Debug off = %v, want 0", n)
	}
}
