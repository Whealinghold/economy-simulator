package economysim

import (
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

// The call, the events, the snapshot rows and the logs of the technology module (plan
// 3-tec, Phase 6).

// callWorld is extConfig with one field "making" on the recipe make (cap 2 000, trend
// off, learning on): the outside node coast is region 0 and has no cell, lowland is
// region 1 and holds the one cell.
func callWorld(t *testing.T, opts ...Option) (*World, *evLog) {
	t.Helper()
	c := extConfig()
	c.Modules.Technology = &Technology{Fields: []TechField{{ID: "making", Recipes: []string{"make"}, CapBP: 2_000, Base: 100, DoublingBP: 300}}}
	w, _, rec := extWorld(t, c, opts...)
	return w, rec
}

// TestAdjustTechnology_AtomicAndErrors verifies the errors of the call in the order of
// spec-technology.md 7.1.3 and that a refusal changes nothing: the module off is
// ErrModuleOff; a region outside the range, an outside node, a field outside the
// range, a delta above MaxGainBP in magnitude and two entries for one cell are
// ErrUnknownRegion, ErrOutsideNode, ErrUnknownTechField, ErrOutOfBounds and
// ErrDuplicateID, each a *ValidationError whose path names the entry; and a valid entry
// before a bad one is not applied and emits nothing.
//
// Requirement: FR-TEC-021, FR-TEC-022
func TestAdjustTechnology_AtomicAndErrors(t *testing.T) {
	t.Parallel()
	off, _, _ := extWorld(t, extConfig())
	if err := off.AdjustTechnology(nil); !errors.Is(err, ErrModuleOff) {
		t.Errorf("AdjustTechnology with the module off = %v, want ErrModuleOff", err)
	}
	ok := TechnologyAdjustment{Region: 1, Field: 0, DeltaBP: 100}
	tests := []struct {
		name string
		bad  TechnologyAdjustment
		want error
		path string
	}{
		{"region above range", TechnologyAdjustment{Region: 9}, ErrUnknownRegion, "region"},
		{"region below range", TechnologyAdjustment{Region: -1}, ErrUnknownRegion, "region"},
		{"an outside node", TechnologyAdjustment{Region: 0}, ErrOutsideNode, "region"},
		{"field above range", TechnologyAdjustment{Region: 1, Field: 1}, ErrUnknownTechField, "field"},
		{"field below range", TechnologyAdjustment{Region: 1, Field: -1}, ErrUnknownTechField, "field"},
		{"delta above the bound", TechnologyAdjustment{Region: 1, DeltaBP: MaxGainBP + 1}, ErrOutOfBounds, "delta_bp"},
		{"delta below the bound", TechnologyAdjustment{Region: 1, DeltaBP: -MaxGainBP - 1}, ErrOutOfBounds, "delta_bp"},
	}
	for _, tt := range tests {
		w, rec := callWorld(t)
		err := w.AdjustTechnology([]TechnologyAdjustment{ok, tt.bad})
		var ve *ValidationError
		if !errors.Is(err, tt.want) || !errors.As(err, &ve) || ve.Path[len(ve.Path)-2] != "changes[1]" || ve.Path[len(ve.Path)-1] != tt.path {
			t.Errorf("%s: error = %v (path %v), want %v at changes[1].%s", tt.name, err, ve, tt.want, tt.path)
		}
		if w.tec.cells[0].Advance != 0 || len(of[TechnologyAdjusted](rec)) != 0 {
			t.Errorf("%s: a refused call moved the advance to %d and emitted %d events, want neither", tt.name, w.tec.cells[0].Advance, len(of[TechnologyAdjusted](rec)))
		}
	}
	w, _ := callWorld(t)
	if err := w.AdjustTechnology([]TechnologyAdjustment{ok, ok}); !errors.Is(err, ErrDuplicateID) {
		t.Errorf("two entries for one cell = %v, want ErrDuplicateID", err)
	}
	if err := w.AdjustTechnology(nil); err != nil {
		t.Errorf("an empty call = %v, want nil", err)
	}
}

// TestAdjustTechnology_ClampAndLost verifies example B of spec-technology.md 16.2: -500
// on an advance of 900 gives 400 with nothing lost; -500 on 300 gives 0 and loses 200; +500
// on 1 900 under a cap of 2 000 gives 2 000 and loses 400. The event carries the gain
// before and after and the call changes the gain at once.
//
// Requirement: FR-TEC-021, FR-TEC-023
func TestAdjustTechnology_ClampAndLost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		start, delta, wantA, wantLost int64
	}{
		{900, -500, 400, 0},
		{300, -500, 0, 200},
		{1_900, 500, 2_000, 400},
	}
	for _, tt := range tests {
		w, rec := callWorld(t)
		if err := w.AdjustTechnology([]TechnologyAdjustment{{Region: 1, Field: 0, DeltaBP: tt.start}}); err != nil {
			t.Fatalf("AdjustTechnology(%d) = %v, want nil", tt.start, err)
		}
		rec.events = nil
		if err := w.AdjustTechnology([]TechnologyAdjustment{{Region: 1, Field: 0, DeltaBP: tt.delta}}); err != nil {
			t.Fatalf("AdjustTechnology(%d) = %v, want nil", tt.delta, err)
		}
		got := of[TechnologyAdjusted](rec)
		want := TechnologyAdjusted{Tick: w.Tick(), Region: 1, Field: 0, DeltaBP: tt.delta, BeforeBP: tt.start, AfterBP: tt.wantA, LostBP: tt.wantLost}
		if len(got) != 1 || got[0] != want || w.tec.cells[0].Advance != tt.wantA || w.tec.gain[0] != tt.wantA {
			t.Errorf("start %d, delta %d: events %+v, advance %d, gain %d, want %+v and both %d", tt.start, tt.delta, got, w.tec.cells[0].Advance, w.tec.gain[0], want, tt.wantA)
		}
	}
	w, rec := callWorld(t)
	for _, d := range []int64{0, -50} { // nothing to cut, and a cut of everything: no change, no event
		if err := w.AdjustTechnology([]TechnologyAdjustment{{Region: 1, DeltaBP: d}}); err != nil {
			t.Fatalf("AdjustTechnology(%d) = %v, want nil", d, err)
		}
	}
	if got := of[TechnologyAdjusted](rec); len(got) != 0 {
		t.Errorf("events for a call that changed nothing = %+v, want none", got)
	}
}

// TestAdjustTechnology_EventOrderAndTick verifies that the events of one call come in
// ascending (region, field) order whatever the order of the entries, each with the
// tick of the next step, and that the call is not listed by Slots.
//
// Requirement: FR-TEC-023
func TestAdjustTechnology_EventOrderAndTick(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, techTwoByTwo(), WithObserver(rec))
	step(t, w, 2)
	rec.events = nil
	err := w.AdjustTechnology([]TechnologyAdjustment{{Region: 1, Field: 1, DeltaBP: 100}, {Region: 0, Field: 0, DeltaBP: 100}, {Region: 1, Field: 0, DeltaBP: 100}})
	if err != nil {
		t.Fatalf("AdjustTechnology() = %v, want nil", err)
	}
	var got [][3]int64
	for _, e := range of[TechnologyAdjusted](rec) {
		got = append(got, [3]int64{e.Tick, int64(e.Region), int64(e.Field)})
	}
	if want := [][3]int64{{2, 0, 0}, {2, 1, 0}, {2, 1, 1}}; !reflect.DeepEqual(got, want) {
		t.Errorf("events (tick, region, field) = %v, want %v", got, want)
	}
	if s, err := w.Slots(); err != nil || !reflect.DeepEqual(s, SlotState{}) {
		t.Errorf("Slots() = %+v, %v, want the empty state: an adjustment holds no value", s, err)
	}
}

// TestTechnologyAdvanced_PerCellOnlyWhenAccrued verifies that the event comes for a cell
// in which something accrued: with learning alone, only in the ticks in which a job
// completed (ticks 1 and 3 of a recipe of two ticks); with a trend, in every tick.
//
// Requirement: FR-TEC-024
func TestTechnologyAdvanced_PerCellOnlyWhenAccrued(t *testing.T) {
	t.Parallel()
	ticks := func(f TechField) (out []int64) {
		rec := &evLog{}
		w := newMill(t, techMill(2, 2, f), WithObserver(rec))
		step(t, w, 4)
		for _, e := range of[TechnologyAdvanced](rec) {
			out = append(out, e.Tick)
		}
		return out
	}
	if got := ticks(techField()); !reflect.DeepEqual(got, []int64{1, 3}) {
		t.Errorf("learning alone: events at ticks %v, want [1 3]", got)
	}
	trend := techField()
	trend.TrendBP = 10_000
	if got := ticks(trend); !reflect.DeepEqual(got, []int64{0, 1, 2, 3}) {
		t.Errorf("with a trend: events at ticks %v, want every tick", got)
	}
}

// TestTechnologyAdvanced_Fields verifies the identity of the event, over 120 ticks of a
// field with every mechanism on and a cap of 60 that the trend reaches: After = Before +
// Doing + Trend + Discovery + CatchUp - Clamped, Clamped never negative, AtCap exactly
// at the cap, each event's Before the After of the cell's last, and the cap cutting
// something.
//
// Requirement: FR-TEC-017, FR-TEC-024
func TestTechnologyAdvanced_Fields(t *testing.T) {
	t.Parallel()
	f := techAll()
	f.CapBP = 60
	rec := &evLog{}
	w := newMill(t, techMill(2, 4, f), WithObserver(rec))
	step(t, w, 120)
	last, cut := int64(0), false
	events := of[TechnologyAdvanced](rec)
	for _, e := range events {
		sum := e.BeforeBP + e.DoingAddedBP + e.TrendAddedBP + e.DiscoveryAddedBP + e.CatchUpAddedBP - e.ClampedBP
		if e.AfterBP != sum || e.ClampedBP < 0 || e.BeforeBP != last || e.AtCap != (e.AfterBP == 60) || e.AfterBP > 60 {
			t.Fatalf("event %+v: sum %d, last After %d, want the identity, Clamped >= 0, Before = last After and AtCap at the cap", e, sum, last)
		}
		last, cut = e.AfterBP, cut || e.ClampedBP > 0
	}
	if len(events) == 0 || !cut || last != 60 {
		t.Errorf("%d events, cut %v, last gain %d, want events, the cap cutting and the gain at 60", len(events), cut, last)
	}
}

// TestEvents_OrderAfterP83b verifies that the events of a tick keep their order with the
// module on: TechnologyAdvanced after every RecipeExecuted of the tick and before its
// TickEnded, in ascending (region, field) order.
//
// Requirement: FR-TEC-024
func TestEvents_OrderAfterP83b(t *testing.T) {
	t.Parallel()
	c := techTwoByTwo()
	c.Modules.Technology.Fields[0].TrendBP = 10_000
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec))
	step(t, w, 3)
	var lastExec, firstAdv, lastEnd = -1, -1, -1
	prev := [2]int64{-1, -1}
	for i, e := range rec.events {
		switch e := e.(type) {
		case RecipeExecuted:
			lastExec = i
		case TechnologyAdvanced:
			if e.Tick != 1 {
				continue
			}
			if firstAdv < 0 {
				firstAdv = i
			}
			if key := [2]int64{int64(e.Region), int64(e.Field)}; key[0] < prev[0] || key[0] == prev[0] && key[1] <= prev[1] {
				t.Errorf("TechnologyAdvanced (%v) after (%v) in tick 1, want ascending", key, prev)
			} else {
				prev = key
			}
		case TickEnded:
			if e.Tick == 1 {
				lastEnd = i
			}
		}
	}
	if firstAdv < 0 || lastEnd < 0 || firstAdv >= lastEnd || lastExec < 0 {
		t.Fatalf("event indexes: first TechnologyAdvanced %d, TickEnded %d, last RecipeExecuted %d, want the advance before the end", firstAdv, lastEnd, lastExec)
	}
}

// TestSnapshot_TechRows verifies the rows: one for each cell with experience or advance
// above 0, in (region, field) order, with the gain derived from them; none with the module
// off; and a copy, so a consumer that edits a row changes nothing of the world.
//
// Requirement: FR-TEC-026
func TestSnapshot_TechRows(t *testing.T) {
	t.Parallel()
	w := newMill(t, techTwoByTwo())
	if s, _ := w.Snapshot(); len(s.Tech) != 0 {
		t.Errorf("rows of a world that has done nothing = %+v, want none", s.Tech)
	}
	if err := w.AdjustTechnology([]TechnologyAdjustment{{Region: 1, Field: 1, DeltaBP: 400}, {Region: 0, Field: 1, DeltaBP: 100}}); err != nil {
		t.Fatalf("AdjustTechnology() = %v, want nil", err)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	want := []TechRow{{Region: 0, Field: 1, AdvanceBP: 100, GainBP: 100}, {Region: 1, Field: 1, AdvanceBP: 400, GainBP: 400}}
	if !reflect.DeepEqual(s.Tech, want) {
		t.Errorf("Tech = %+v, want %+v", s.Tech, want)
	}
	s.Tech[0].GainBP = 99_999
	if again, _ := w.Snapshot(); !reflect.DeepEqual(again.Tech, want) {
		t.Errorf("Tech after an edit of a row = %+v, want %+v (a copy)", again.Tech, want)
	}
	off := newMill(t, millConfig(2, 2))
	if s, _ := off.Snapshot(); s.Tech != nil {
		t.Errorf("Tech with the module off = %+v, want nil", s.Tech)
	}
}

// TestLogs_TechnologyDebugOnly verifies that the module logs at Debug only: over 10 ticks
// and a call, every line after construction is "technology advanced" or "technology
// adjusted" at Debug, with the region, the field ID and the gain, and a logger that is
// not enabled for Debug receives none (the Enabled guard).
//
// Requirement: FR-TEC-037
func TestLogs_TechnologyDebugOnly(t *testing.T) {
	t.Parallel()
	f := techField()
	f.TrendBP = 10_000
	lg := newCapture()
	w := newMill(t, techMill(2, 2, f), WithLogger(slog.New(lg)))
	lg.take("world created")
	if err := w.AdjustTechnology([]TechnologyAdjustment{{Region: 0, Field: 0, DeltaBP: 100}}); err != nil {
		t.Fatalf("AdjustTechnology() = %v, want nil", err)
	}
	step(t, w, 10)
	var adjusted, advanced []logRec
	for _, l := range *lg.lines {
		switch {
		case l.msg == "technology adjusted":
			adjusted = append(adjusted, l)
		case l.msg == "technology advanced":
			advanced = append(advanced, l)
		case l.level > slog.LevelDebug && strings.Contains(l.msg, "technolog"):
			t.Errorf("line %q at level %v, want the module to log nothing above Debug in a tick", l.msg, l.level)
		}
	}
	if len(adjusted) != 1 || len(advanced) != 10 {
		t.Fatalf("%d adjusted and %d advanced lines, want 1 and 10", len(adjusted), len(advanced))
	}
	var quiet bytes.Buffer
	q := newMill(t, techMill(2, 2, f), WithLogger(slog.New(slog.NewTextHandler(&quiet, &slog.HandlerOptions{Level: slog.LevelInfo}))))
	step(t, q, 5)
	if strings.Contains(quiet.String(), "technology advanced") || strings.Contains(quiet.String(), "technology adjusted") {
		t.Errorf("a logger at Info received %q, want no tick line of the module", quiet.String())
	}
	for _, l := range append(adjusted, advanced...) {
		if l.level != slog.LevelDebug || l.strs["field"] != "milling" || l.attrs["region"] != 0 || l.attrs["gain"] <= 0 {
			t.Errorf("line %+v, want Debug with field milling, region 0 and a gain above 0", l)
		}
	}
}

// TestNonInterference_Technology verifies that an observer and a Debug logger change
// nothing of the world: the same snapshot after 60 ticks with a field of every
// mechanism, with and without them.
//
// Requirement: NFR-TEC-001
func TestNonInterference_Technology(t *testing.T) {
	t.Parallel()
	c := techMill(2, 4, techAll())
	plain := newMill(t, c, WithSeed(3))
	loud := newMill(t, c, WithSeed(3), WithObserver(&evLog{}), WithLogger(slog.New(newCapture())))
	step(t, plain, 60)
	step(t, loud, 60)
	a, _ := plain.Snapshot()
	b, _ := loud.Snapshot()
	if !reflect.DeepEqual(a, b) || len(a.Tech) == 0 {
		t.Errorf("snapshots differ with an observer and a logger (rows %d and %d), want equal and with rows", len(a.Tech), len(b.Tech))
	}
}
