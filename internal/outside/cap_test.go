package outside_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/outside"
)

// TestOffer_Caps verifies the offered and the cut units: the wanted units less
// what the unit cap and the value cap leave, over the caps that are set, never
// below 0, and the degenerate inputs (no wanted unit, a limit below 1).
//
// Requirement: FR-EXT-030
func TestOffer_Caps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		c                outside.Cap
		wanted, limit    int64
		offered, wantCut int64
	}{
		{"no cap", outside.Cap{}, 200, 96, 200, 0},
		// 100 units left of a want of 200: 100 offered, 100 cut (18.3, ore, tick 7).
		{"unit cap", outside.Cap{UnitsLeft: 100, UnitCap: true}, 200, 96, 100, 100},
		// 2 280 left at a limit of 324: floor(2 280 / 324) = floor(7.04) = 7 of 10, 3 cut (18.3, reagent, tick 3).
		{"value cap", outside.Cap{ValueLeft: 2_280, ValueCap: true}, 10, 324, 7, 3},
		// The tighter of the two: units allow 5, the value allows floor(1 000 / 100) = 10.
		{"units tighter", outside.Cap{UnitsLeft: 5, ValueLeft: 1_000, UnitCap: true, ValueCap: true}, 8, 100, 5, 3},
		{"value tighter", outside.Cap{UnitsLeft: 50, ValueLeft: 250, UnitCap: true, ValueCap: true}, 8, 100, 2, 6},
		// A cap that is not set is not read, whatever its remainder holds.
		{"unset caps ignored", outside.Cap{UnitsLeft: 0, ValueLeft: 0}, 4, 10, 4, 0},
		// An ended side (remainder 0) offers nothing.
		{"nothing left", outside.Cap{UnitsLeft: 0, UnitCap: true}, 4, 10, 0, 4},
		{"a negative remainder offers nothing", outside.Cap{UnitsLeft: -3, UnitCap: true}, 4, 10, 0, 4},
		{"limit below one", outside.Cap{}, 4, 0, 0, 4},
		{"limit of one", outside.Cap{ValueLeft: 3, ValueCap: true}, 4, 1, 3, 1},
		{"one unit wanted", outside.Cap{}, 1, 5, 1, 0},
		{"nothing wanted", outside.Cap{}, 0, 10, 0, 0},
		{"negative wanted", outside.Cap{}, -2, 10, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			offered, cut := outside.Offer(tt.c, tt.wanted, tt.limit)
			if offered != tt.offered || cut != tt.wantCut {
				t.Errorf("Offer(%+v, %d, %d) = %d, %d, want %d, %d", tt.c, tt.wanted, tt.limit, offered, cut, tt.offered, tt.wantCut)
			}
		})
	}
}

// TestCap_Ended verifies the end of a side after settlement over the worked
// example B of spec-external-sector.md 18.3: the ore side, 1 500 units at 200 a
// tick and a limit of 96, ends in tick 7 with the cause units, 1 500 units and a
// list value of 144 000; the reagent side, a list value of 12 000 at a limit of
// 324 and 10 units a tick, ends in tick 3 with the cause value, 37 units and a
// list value of 11 988.
//
// Requirement: FR-EXT-030, FR-EXT-031
func TestCap_Ended(t *testing.T) {
	t.Parallel()
	run := func(c outside.Cap, wanted, limit int64) (endTick int, cause int, units, list, cutTotal int64) {
		for tick := range 50 {
			offered, cut := outside.Offer(c, wanted, limit)
			cutTotal += cut
			var lv int64
			var ok bool
			c, lv, ok = c.Settle(offered, limit)
			if !ok {
				t.Fatalf("Settle(%d, %d) refused a trade the cap allowed", offered, limit)
			}
			units += offered
			list += lv
			if cs, ended := c.Ended(limit); ended {
				return tick, cs, units, list, cutTotal
			}
		}
		t.Fatalf("the side never ended")
		return 0, 0, 0, 0, 0
	}
	// Ticks 0 to 6 take 1 400 units; in tick 7 the want of 200 is cut to the remaining 100.
	tick, cause, units, list, cut := run(outside.Cap{UnitsLeft: 1_500, UnitCap: true}, 200, 96)
	if tick != 7 || cause != outside.EndUnits || units != 1_500 || list != 144_000 || cut != 100 {
		t.Errorf("ore side: ended in tick %d cause %d with %d units, list value %d, cut %d, want tick 7, units, 1500, 144000, 100", tick, cause, units, list, cut)
	}
	// Ticks 0 to 2 offer 10 (3 240 each); tick 3 has 2 280 left and offers floor(2 280 / 324) = 7, cutting 3.
	tick, cause, units, list, cut = run(outside.Cap{ValueLeft: 12_000, ValueCap: true}, 10, 324)
	if tick != 3 || cause != outside.EndValue || units != 37 || list != 11_988 || cut != 3 {
		t.Errorf("reagent side: ended in tick %d cause %d with %d units, list value %d, cut %d, want tick 3, value, 37, 11988, 3", tick, cause, units, list, cut)
	}

	for _, tt := range []struct {
		name  string
		c     outside.Cap
		limit int64
		cause int
		ended bool
	}{
		{"no cap never ends", outside.Cap{UnitsLeft: 0, ValueLeft: 0}, 10, 0, false},
		{"units left", outside.Cap{UnitsLeft: 1, UnitCap: true}, 10, 0, false},
		{"value equal to the limit is enough", outside.Cap{ValueLeft: 10, ValueCap: true}, 10, 0, false},
		{"value below the limit ends", outside.Cap{ValueLeft: 9, ValueCap: true}, 10, outside.EndValue, true},
		// Both reached: the units are named first.
		{"units win a tie", outside.Cap{UnitsLeft: 0, ValueLeft: 0, UnitCap: true, ValueCap: true}, 10, outside.EndUnits, true},
		{"a negative remainder ends", outside.Cap{UnitsLeft: -1, UnitCap: true}, 10, outside.EndUnits, true},
	} {
		if cause, ended := tt.c.Ended(tt.limit); cause != tt.cause || ended != tt.ended {
			t.Errorf("%s: Ended(%d) of %+v = %d, %v, want %d, %v", tt.name, tt.limit, tt.c, cause, ended, tt.cause, tt.ended)
		}
	}
}

// TestCap_SettleRefuses verifies that Settle leaves the cap unchanged and reports
// ok false for a trade the cap did not allow, a negative quantity or limit, and a
// list value that does not fit in int64.
//
// Requirement: FR-EXT-031
func TestCap_SettleRefuses(t *testing.T) {
	t.Parallel()
	c := outside.Cap{UnitsLeft: 5, ValueLeft: 100, UnitCap: true, ValueCap: true}
	for _, tt := range []struct {
		name         string
		units, limit int64
	}{
		{"too many units", 6, 1},
		{"too much value", 5, 21},
		{"negative units", -1, 1},
		{"negative limit", 1, -1},
		{"overflow", 1 << 62, 1 << 62},
	} {
		next, list, ok := c.Settle(tt.units, tt.limit)
		if ok || next != c || list != 0 {
			t.Errorf("%s: Settle(%d, %d) = %+v, %d, %v, want the cap unchanged, 0, false", tt.name, tt.units, tt.limit, next, list, ok)
		}
	}
	// A trade of no unit, or at a limit of 0, is allowed and moves nothing.
	if next, list, ok := c.Settle(0, 20); !ok || list != 0 || next != c {
		t.Errorf("Settle(0, 20) = %+v, %d, %v, want the cap unchanged, 0, true", next, list, ok)
	}
	if next, list, ok := c.Settle(5, 0); !ok || list != 0 || next.UnitsLeft != 0 || next.ValueLeft != 100 {
		t.Errorf("Settle(5, 0) = %+v, %d, %v, want the units used and the value kept, 0, true", next, list, ok)
	}
	next, list, ok := c.Settle(5, 20)
	if !ok || list != 100 || next.UnitsLeft != 0 || next.ValueLeft != 0 {
		t.Errorf("Settle(5, 20) = %+v, %d, %v, want everything used, list value 100", next, list, ok)
	}
	// A cap that is not set keeps its remainder untouched.
	free := outside.Cap{UnitsLeft: 7, ValueLeft: 9}
	if next, _, ok := free.Settle(3, 2); !ok || next != free {
		t.Errorf("Settle on unset caps = %+v, %v, want the cap unchanged, true", next, ok)
	}
}
