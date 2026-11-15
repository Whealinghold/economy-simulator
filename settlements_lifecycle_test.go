package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// regionClassEvents returns the SettlementClassChanged and SettlementHeld events of
// the events, in order.
func regionClassEvents(events []economysim.Event) (changed []economysim.SettlementClassChanged, held []economysim.SettlementHeld) {
	for _, e := range events {
		switch e := e.(type) {
		case economysim.SettlementClassChanged:
			changed = append(changed, e)
		case economysim.SettlementHeld:
			held = append(held, e)
		}
	}
	return changed, held
}

// TestLifecycle_ConsumerGrowsAndShrinksARegion verifies the module as a consumer meets
// it, through the public API only: the region north is founded as a town; the consumer
// adds external households between two ticks; the class stays a town until the next
// step, which reports the rise to a city with the cause grew, and the snapshot row then
// carries the new class and its tick; the consumer removes those households again, and
// the review holds the fall for the dwell, with the ticks left counting down, and then
// shrinks the class.
//
// Arithmetic: settlementsRunConfig has a dwell of 10, Since 0 at construction, a town at
// 600 households (enter 500) and a city from 700. With 150 added, 750 meets 700, and the
// review at tick 10 (10 - 0 >= 10) applies it, so Since is 10. Removing the 150 leaves
// about 600, below the leave of a city, floor(700 * 9 000 / 10 000) = 630, so the fall
// is held at ticks 11 to 19 (elapsed 1 to 9) and applied at tick 20 (20 - 10 = 10).
//
// Requirement: FR-SET-012, FR-SET-014, FR-SET-017, FR-SET-024, FR-SET-025, FR-SET-026
func TestLifecycle_ConsumerGrowsAndShrinksARegion(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rec := &recorder{}
	w := newPackWorld(t, settlementsRunConfig(), 1, economysim.WithObserver(rec))
	step(t, w, 10) // ticks 0 to 9: the dwell has not run, and nothing is due
	rec.events = nil
	var added []economysim.AgentID
	for range 150 {
		id, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: 0, Money: 100})
		if err != nil {
			t.Fatalf("AddAgent() error = %v, want nil", err)
		}
		added = append(added, id)
	}
	if snap, _ := w.Snapshot(); snap.Regions[0].Class != 2 {
		t.Fatalf("class of north after the additions, before the step = %d, want 2 (a call between ticks changes no class)", snap.Regions[0].Class)
	}
	step(t, w, 1) // tick 10: the dwell has run
	changed, held := regionClassEvents(rec.events)
	if len(changed) != 1 || changed[0].Region != 0 || changed[0].From != 2 || changed[0].To != 3 || changed[0].Cause != economysim.CauseGrew || len(held) != 0 {
		t.Fatalf("events of tick 10 = %+v held %+v, want one SettlementClassChanged of north grew from 2 to 3 and none held", changed, held)
	}
	snap, _ := w.Snapshot()
	if row := snap.Regions[0]; row.Class != 3 || row.ClassSince != 10 || row.Households != changed[0].Households {
		t.Errorf("row of north after tick 10 = class %d since %d households %d, want class 3 since 10 households %d", row.Class, row.ClassSince, row.Households, changed[0].Households)
	}

	rec.events = nil
	for _, id := range added {
		if err := w.RemoveAgent(id); err != nil {
			t.Fatalf("RemoveAgent(%d) error = %v, want nil", id, err)
		}
	}
	step(t, w, 9) // ticks 11 to 19
	changed, held = regionClassEvents(rec.events)
	if len(changed) != 0 || len(held) != 9 {
		t.Fatalf("events of ticks 11 to 19 = %d changed and %d held, want 0 and 9", len(changed), len(held))
	}
	for i, h := range held {
		if want := int64(9 - i); h.Region != 0 || h.Class != 3 || h.Target != 2 || h.TicksLeft != want {
			t.Errorf("held[%d] = %+v, want north holding class 3 for target 2 with %d ticks left", i, h, want)
		}
	}
	step(t, w, 1) // tick 20
	changed, _ = regionClassEvents(rec.events)
	if len(changed) != 1 || changed[0].Cause != economysim.CauseShrank || changed[0].From != 3 || changed[0].To != 2 {
		t.Errorf("events of tick 20 = %+v, want one shrank from 3 to 2", changed)
	}
}

// TestCheckpoint_SettlementsSkipOutsideNodes verifies that tag 23 holds a row for every
// region that can hold households and none for an outside node, and that a world
// restored from it, in a world that has outside nodes, has the same snapshot and takes
// the same next step as the original.
//
// Arithmetic: externalConfig has the reference regions and two outside nodes (coast and
// far); every class and Since is below 128, so a row is two bytes, and the body is two
// bytes per region that is not a node.
//
// Requirement: FR-SET-029, FR-SET-011
func TestCheckpoint_SettlementsSkipOutsideNodes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := externalConfig()
	m := economysim.DefaultSettlements()
	m.Dwell = 3
	m.Classes = []economysim.SettlementClass{{ID: "hamlet"}, {ID: "village", Enter: 300}, {ID: "town", Enter: 500}}
	cfg.Modules.Settlements = &m
	w := newPackWorld(t, cfg, 1)
	step(t, w, 4)
	c := mustCheckpoint(t, w)
	body, ok := settlementsSection(t, mustMarshal(t, c))
	if want := 2 * (len(cfg.Regions) - len(cfg.Modules.External.Nodes)); !ok || len(body) != want {
		t.Fatalf("tag 23 = %v, %v, want %d bytes (2 per region that is not a node)", body, ok, want)
	}
	r, err := economysim.RestoreWorld(cfg, c)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	for _, x := range []*economysim.World{w, r} {
		step(t, x, 1)
	}
	want, _ := w.Snapshot()
	got, _ := r.Snapshot()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("snapshot of the restored world after one step differs from the original: classes %v, want %v",
			classesOf(got), classesOf(want))
	}
}

// classesOf returns the class of each region row of s.
func classesOf(s economysim.Snapshot) []economysim.SettlementClassID {
	var out []economysim.SettlementClassID
	for _, r := range s.Regions {
		out = append(out, r.Class)
	}
	return slices.Clone(out)
}
