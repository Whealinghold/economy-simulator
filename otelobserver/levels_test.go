package otelobserver_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// unfedLevel is the level of household.unfed for the role k1.
func unfedLevel(h *harness) int64 {
	v, _ := h.collect().get("household.unfed", "role", "k1")
	return v
}

// TestUnfed_Sequence16_4 verifies the unfed level of spec-observability.md 16.4
// at every tick, including the tick with no UpkeepShortfall, where the cell that
// the step did not see goes back to 0.
//
// Requirement: FR-OBS-053
func TestUnfed_Sequence16_4(t *testing.T) {
	short := func(region economysim.RegionID, n int64) economysim.Event {
		return economysim.UpkeepShortfall{Region: region, Role: 1, Unfed: n}
	}
	steps := []struct {
		name   string
		events []economysim.Event
		want   int64
	}{
		{"tick 5: 4 unfed", []economysim.Event{short(0, 4)}, 4},
		{"tick 6: 6 unfed adds the change", []economysim.Event{short(0, 6)}, 6},
		{"tick 7: everyone fed, no event", nil, 0},
		{"tick 8: 1 unfed", []economysim.Event{short(0, 1)}, 1},
		{"tick 9: two regions", []economysim.Event{short(0, 3), short(1, 2)}, 5},
		{"tick 10: one region fed again", []economysim.Event{short(1, 2)}, 2},
		{"tick 11: one unfed household", []economysim.Event{short(1, 1)}, 1},
		{"tick 12: it is fed again", nil, 0},
	}
	h := newHarness(t)
	h.send(created())
	for i, s := range steps {
		tick(h, int64(5+i), s.events...)
		if got := unfedLevel(h); got != s.want {
			t.Errorf("%s: household.unfed{k1} = %d, want %d", s.name, got, s.want)
		}
	}
}

// TestUnfed_FailedStepNoReset verifies that a step that failed makes no
// correction: the level is what the events put, no duration is recorded, one
// failure is counted, and the next complete step resets the cell it did not see.
//
// Requirement: FR-OBS-055, FR-OBS-058
func TestUnfed_FailedStepNoReset(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.UpkeepShortfall{Region: 0, Role: 1, Unfed: 4})
	h.send(economysim.TickStarted{Tick: 1}, economysim.TickEnded{Tick: 1, Failed: true})
	if got := unfedLevel(h); got != 4 {
		t.Errorf("household.unfed{k1} after a failed step = %d, want 4: no reset", got)
	}
	got := h.collect()
	if v := got.sum("world.step.failed"); v != 1 {
		t.Errorf("world.step.failed = %d, want 1", v)
	}
	if v := got.sum("world.step.duration"); v != 1 {
		t.Errorf("world.step.duration count = %d, want 1: the failed step records none", v)
	}
	tick(h, 2)
	if got := unfedLevel(h); got != 0 {
		t.Errorf("household.unfed{k1} after the next complete step = %d, want 0", got)
	}
}

// TestUnfed_FailedStepReportDoesNotMaskNextStep verifies that an
// UpkeepShortfall reported in a step that then failed does not stop the next
// complete step from zeroing the cell it does not report.
//
// Guard: a seen flag left set by the failed step would make the next
// step skip the zeroing and household.unfed would stay too high.
func TestUnfed_FailedStepReportDoesNotMaskNextStep(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	h.send(economysim.TickStarted{Tick: 1}, economysim.UpkeepShortfall{Region: 0, Role: 1, Unfed: 4}, economysim.TickEnded{Tick: 1, Failed: true})
	tick(h, 2)
	if got := unfedLevel(h); got != 0 {
		t.Errorf("household.unfed{k1} after a silent step following a failed one = %d, want 0", got)
	}
}

func rolesSnapshot(rows ...economysim.RoleRow) economysim.Snapshot {
	return economysim.Snapshot{Roles: rows}
}

// TestSeed_ByDifference verifies that a second WorldRestored on the same adapter
// with another names table leaves the SDK levels equal to its snapshot and the
// old series at 0, and that a second WorldCreated leaves every level at 0 until
// AgentSpawned.
//
// Requirement: FR-OBS-054
func TestSeed_ByDifference(t *testing.T) {
	h := newHarness(t)
	h.send(economysim.WorldRestored{Names: miniNames(), Snapshot: rolesSnapshot(
		economysim.RoleRow{Region: 0, Role: 0, Agents: 3, Households: 3},
		economysim.RoleRow{Region: 0, Role: 1, Agents: 5, Households: 7, Unfed: 2},
		economysim.RoleRow{Region: 1, Role: 1, Agents: 1, Households: 1, Unfed: 1})})
	got := h.collect()
	check(t, got, "agent.count", 3, "role", "k0")
	check(t, got, "agent.count", 6, "role", "k1")
	check(t, got, "household.count", 8, "role", "k1")
	check(t, got, "household.unfed", 3, "role", "k1")

	// Role 1 is renamed: a series kept from the first table would carry the old name.
	names2 := economysim.Names{Commodities: []string{"a"}, Regions: []string{"x"}, Roles: []string{"m0", "m1", "m2"}}
	h.send(economysim.WorldRestored{Names: names2, Snapshot: rolesSnapshot(
		economysim.RoleRow{Region: 0, Role: 1, Agents: 2, Households: 2},
		economysim.RoleRow{Region: 0, Role: 2, Agents: 4, Households: 9, Unfed: 6})})
	got = h.collect()
	for _, name := range []string{"agent.count", "household.count"} {
		check(t, got, name, 0, "role", "k0")
	}
	for _, name := range []string{"agent.count", "household.count", "household.unfed"} {
		check(t, got, name, 0, "role", "k1")
	}
	check(t, got, "agent.count", 2, "role", "m1")
	check(t, got, "agent.count", 4, "role", "m2")
	check(t, got, "household.count", 9, "role", "m2")
	check(t, got, "household.unfed", 6, "role", "m2")

	h.send(created())
	got = h.collect()
	for _, name := range []string{"agent.count", "household.count", "household.unfed"} {
		if v := got.sum(name); v != 0 {
			t.Errorf("%s after a second WorldCreated = %d, want 0", name, v)
		}
	}
	h.send(economysim.AgentSpawned{Role: 1, Weight: 2})
	check(t, h.collect(), "household.count", 2, "role", "k1")
}

// TestSeed_FirstStepAfterRestoreCorrectsUnfed verifies the case of 16.3: an
// UpkeepShortfall of 3 on a cell seeded at 2 adds 1, and a cell that the first
// step did not see is reset, so the level is a function of that step alone.
//
// Requirement: FR-OBS-054
func TestSeed_FirstStepAfterRestoreCorrectsUnfed(t *testing.T) {
	h := newHarness(t)
	h.send(economysim.WorldRestored{Names: miniNames(), Snapshot: rolesSnapshot(
		economysim.RoleRow{Region: 0, Role: 1, Agents: 5, Households: 7, Unfed: 2},
		economysim.RoleRow{Region: 1, Role: 1, Agents: 1, Households: 1, Unfed: 4})})
	tick(h, 50, economysim.UpkeepShortfall{Region: 0, Role: 1, Unfed: 3})
	if got := unfedLevel(h); got != 3 {
		t.Errorf("household.unfed{k1} after the first step = %d, want 3: 2 + 1 for the seen cell, the other reset", got)
	}
}

// TestSeed_ClearingPriceFromMarkets verifies that the clearing price of every
// market in the restored snapshot is a series from the restore on, before any
// MarketCleared arrives.
//
// Requirement: FR-OBS-054
func TestSeed_ClearingPriceFromMarkets(t *testing.T) {
	h := newHarness(t)
	h.send(economysim.WorldRestored{Names: miniNames(), Snapshot: economysim.Snapshot{Markets: []economysim.MarketRow{
		{State: economysim.MarketState{Region: 1, Commodity: 2, Price: 77}},
		{State: economysim.MarketState{Region: 9, Commodity: 2, Price: 5}}}}})
	got := h.collect()
	check(t, got, "market.clearing_price", 77, "commodity", "c2", "region", "r1")
	if n := len(got["economysim.market.clearing_price"]); n != 1 {
		t.Errorf("market.clearing_price series = %d, want 1: the market of region 9 is outside the names", n)
	}
}

// TestUnfed_IDsOutsideTheTableAreDropped verifies that an UpkeepShortfall or a
// restored role row whose region or role is outside the names table moves no
// level and does not panic, and that the last cell of the table still works.
//
// Region 2 and role 1 are the last cell of a 3 x 2 table. Role 2 with region 0 is
// inside the flat array (index 2) but outside the role table, and region 3 with
// role 0 is index 6, one past the array.
//
// Requirement: FR-OBS-052
func TestUnfed_IDsOutsideTheTableAreDropped(t *testing.T) {
	h := newHarness(t)
	h.send(economysim.WorldRestored{Names: miniNames(), Snapshot: rolesSnapshot(
		economysim.RoleRow{Region: 9, Role: 1, Agents: 1, Households: 1, Unfed: 5},
		economysim.RoleRow{Region: 0, Role: 5, Agents: 1, Households: 1, Unfed: 5},
		economysim.RoleRow{Region: 0, Role: 1, Agents: 3, Households: 3})}) // a valid row after the two bad ones
	bad := []economysim.UpkeepShortfall{{Region: -1, Role: 0, Unfed: 3}, {Region: 0, Role: -1, Unfed: 3},
		{Region: 0, Role: 2, Unfed: 3}, {Region: 3, Role: 0, Unfed: 3}}
	for _, ev := range bad {
		tick(h, 0, ev)
	}
	tick(h, 1, economysim.UpkeepShortfall{Region: 2, Role: 1, Unfed: 2}, economysim.UpkeepShortfall{Region: 0, Role: 0, Unfed: 4})
	got := h.collect()
	if v := got.sum("household.unfed"); v != 6 {
		t.Errorf("household.unfed = %d, want 6: only the first and the last cell of the table count", v)
	}
	check(t, got, "household.unfed", 2, "role", "k1")
	check(t, got, "household.unfed", 4, "role", "k0")
	check(t, got, "agent.count", 3, "role", "k1")
}
