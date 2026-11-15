package economysim_test

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// settlementsLines are the records of the settlements module's Debug lines, by message.
var settlementsLines = []string{"settlement class changed", "settlement class held"}

// TestLogs_SettlementsConstructionLineAndDebugLines verifies the logs of the module
// (spec-settlements.md 10.2): the "world created" line carries settlement_classes and
// lists settlements in modules; the founding, each held tick and the change are Debug
// lines with constant messages and the keys of the spec; nothing of the module is
// written at Info or above, and no Error.
//
// Arithmetic: 150 external agents make 750 households in north, a town (2) at
// construction, so the target is city (3) from tick 0; with Dwell 10 and Since 0 the
// change is held in ticks 0 to 9 (ten "held" lines) and applies at tick 10, so a run of 12
// steps logs one founding line, ten held lines and one changed line.
//
// Requirement: FR-SET-027
func TestLogs_SettlementsConstructionLineAndDebugLines(t *testing.T) {
	t.Parallel()
	h := newCaptureHandler()
	w := newPackWorld(t, settlementsRunConfig(), 1, economysim.WithLogger(slog.New(h)))
	addExternals(150)(w)
	step(t, w, 12)
	var created *logLine
	counts := map[string]int{}
	records := h.records()
	for i, l := range records {
		switch {
		case l.msg == "world created":
			created = &records[i]
		case slices.Contains(settlementsLines, l.msg):
			counts[l.msg]++
			if l.level != slog.LevelDebug {
				t.Errorf("%q written at %v, want Debug", l.msg, l.level)
			}
		case l.level >= slog.LevelError:
			t.Errorf("record %q at %v, want no Error", l.msg, l.level)
		}
	}
	checkCreatedLine(t, created)
	if want := map[string]int{"settlement class changed": 2, "settlement class held": 10}; !reflect.DeepEqual(counts, want) {
		t.Errorf("Debug lines of the module = %v, want %v (the founding and the change, ten holds)", counts, want)
	}
	checkClassLines(t, records)
}

// checkCreatedLine checks the settlements part of the construction line: the count of
// classes and the module in the list.
func checkCreatedLine(t *testing.T, created *logLine) {
	t.Helper()
	if created == nil {
		t.Fatal("no \"world created\" line")
	}
	if v, ok := created.attr("settlement_classes"); !ok || v.Int64() != 4 {
		t.Errorf("settlement_classes of the construction line = %v, %v, want 4, true", v, ok)
	}
	if v, ok := created.attr("modules"); !ok || !slices.Contains(v.Any().([]string), "settlements") {
		t.Errorf("modules of the construction line = %v, %v, want a list with settlements", v, ok)
	}
}

// checkClassLines checks the keys of the first "settlement class changed" line, the
// founding of north: from NoSettlementClass (-1) to the town (2), 600 households, cause
// founded.
func checkClassLines(t *testing.T, records []logLine) {
	t.Helper()
	i := slices.IndexFunc(records, func(l logLine) bool { return l.msg == "settlement class changed" })
	if i < 0 {
		return // the count above has already failed
	}
	l := records[i]
	from, _ := l.attr("from_settlement_class")
	to, _ := l.attr("to_settlement_class")
	hh, _ := l.attr("households")
	cause, _ := l.attr("cause")
	if from.Int64() != -1 || to.Int64() != 2 || hh.Int64() != 600 || cause.String() != "founded" {
		t.Errorf("founding line: from %v to %v households %v cause %v, want -1, 2, 600, founded", from, to, hh, cause)
	}
}

// TestLogs_SettlementsDebugBehindEnabled verifies that a logger that leaves Debug off
// receives no record of the module: the lines are built only when Debug is enabled.
//
// Requirement: FR-SET-027
func TestLogs_SettlementsDebugBehindEnabled(t *testing.T) {
	t.Parallel()
	all := newCaptureHandler()
	run := func(l *slog.Logger) {
		w := newPackWorld(t, settlementsRunConfig(), 1, economysim.WithLogger(l))
		addExternals(150)(w)
		step(t, w, 12)
	}
	run(slog.New(all))
	atInfo := 0
	for _, l := range all.records() {
		if l.level >= slog.LevelInfo {
			atInfo++
		}
	}
	info := &levelHandler{min: slog.LevelInfo}
	run(slog.New(info))
	if info.handled != atInfo {
		t.Errorf("records handled with Debug off = %d, want %d (those at Info and above of the full run)", info.handled, atInfo)
	}
}

// TestEvents_SettlementMetricsDerivable verifies that the two instruments of
// spec-settlements.md 10.4 can be folded from the events alone: the last To of
// SettlementClassChanged of each region, which the construction events set, is the
// class of the snapshot, and the changes other than the founding are counted by region.
//
// Requirement: FR-SET-031
func TestEvents_SettlementMetricsDerivable(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := newPackWorld(t, settlementsRunConfig(), 1, economysim.WithObserver(rec))
	addExternals(150)(w)
	step(t, w, 12)
	gauge, changes := map[economysim.RegionID]economysim.SettlementClassID{}, map[economysim.RegionID]int{}
	for _, e := range rec.events {
		if c, ok := e.(economysim.SettlementClassChanged); ok {
			gauge[c.Region] = c.To
			if c.Cause != economysim.CauseFounded {
				changes[c.Region]++
			}
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if got := snap.Regions[0].Class; gauge[0] != got || got != 3 || changes[0] != 1 {
		t.Errorf("gauge %d and %d changes from the events, class %d in the snapshot, want 3, 1 and 3", gauge[0], changes[0], got)
	}
}

// TestNames_SettlementClassesFollowTheOrderOfEnter verifies the names table: the string
// ID of each class in the order of the SettlementClassID, in the snapshot and in
// WorldCreated, and nil with the module off.
//
// Requirement: FR-SET-002, FR-OBS-002
func TestNames_SettlementClassesFollowTheOrderOfEnter(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := newPackWorld(t, settlementsRunConfig(), 1, economysim.WithObserver(rec))
	want := []string{"hamlet", "village", "town", "city"}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	created, _ := rec.events[0].(economysim.WorldCreated)
	if !slices.Equal(snap.Names.SettlementClasses, want) || !slices.Equal(created.Names.SettlementClasses, want) {
		t.Errorf("names of the classes = %v and %v, want %v in both", snap.Names.SettlementClasses, created.Names.SettlementClasses, want)
	}
	off, err := newPackWorld(t, referenceConfig(), 1).Snapshot()
	if err != nil || off.Names.SettlementClasses != nil {
		t.Errorf("names of the classes with the module off = %v, %v, want nil, nil", off.Names.SettlementClasses, err)
	}
}

// TestDeterminism_SettlementsNonInterference verifies that a logger and an observer
// change nothing: the same seed and configuration give the same snapshot and the same
// checkpoint bytes with and without them, through a hold and a change.
//
// Requirement: FR-SET-016, FR-SET-033
func TestDeterminism_SettlementsNonInterference(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	run := func(opts ...economysim.Option) (economysim.Snapshot, []byte) {
		w := newPackWorld(t, settlementsRunConfig(), 1, opts...)
		addExternals(150)(w)
		step(t, w, 15)
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		return snap, mustMarshal(t, mustCheckpoint(t, w))
	}
	bareSnap, bareCk := run()
	obsSnap, obsCk := run(economysim.WithObserver(&recorder{}), economysim.WithLogger(slog.New(newCaptureHandler())))
	if !reflect.DeepEqual(bareSnap, obsSnap) || !slices.Equal(bareCk, obsCk) {
		t.Errorf("snapshot or checkpoint differs with a logger and an observer attached, want identical")
	}
}
