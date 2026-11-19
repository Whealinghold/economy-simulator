package policy_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/policy"
)

// TestSchedule_Value verifies the step function of spec-policy.md 8.2.1 on the
// head-tax schedule of 14.1 (10 000 from 0, 15 000 from 120, 10 000 from 240),
// where the step at 120 is the value at tick 126 and the step at 240 at 246.
//
// Requirement: FR-PLC-061
func TestSchedule_Value(t *testing.T) {
	t.Parallel()
	steps := []policy.Step{
		{Lever: 0, At: 0, BP: 10_000},
		{Lever: 0, At: 120, BP: 15_000},
		{Lever: 0, At: 240, BP: 10_000},
	}
	tests := []struct {
		tick int64
		want int64
	}{{0, 10_000}, {6, 10_000}, {119, 10_000}, {120, 15_000}, {126, 15_000}, {239, 15_000}, {246, 10_000}}
	for _, tt := range tests {
		got, closed, ok := policy.ScheduleValue(steps, 0, 0, tt.tick)
		if !ok || closed || got != tt.want {
			t.Errorf("ScheduleValue(tick %d) = %d, %v, %v; want %d, false, true", tt.tick, got, closed, ok, tt.want)
		}
	}
}

// TestSchedule_HeldBeforeFirstStep verifies that a lever with no step at or
// below the tick is held, and so is a lever with no step at all.
//
// Requirement: FR-PLC-061
func TestSchedule_HeldBeforeFirstStep(t *testing.T) {
	t.Parallel()
	steps := []policy.Step{{Lever: 1, At: 50, BP: 7}}
	for _, tt := range []struct {
		lever int
		tick  int64
	}{{1, 49}, {1, 0}, {0, 100}, {2, 100}} {
		if bp, _, ok := policy.ScheduleValue(steps, tt.lever, 0, tt.tick); ok {
			t.Errorf("ScheduleValue(lever %d, tick %d) = %d, true; want held", tt.lever, tt.tick, bp)
		}
	}
}

// TestSchedule_WrapAndCarriedLast verifies the cycle of 8.2.1: with a cycle of
// 100 and steps at 20 (5) and 60 (7), the lever is held before 20 in the first
// cycle, takes its last step (7) at ticks whose u is below 20 once the cycle
// has wrapped, and repeats the steps after that.
//
// Requirement: FR-PLC-011, FR-PLC-061
func TestSchedule_WrapAndCarriedLast(t *testing.T) {
	t.Parallel()
	steps := []policy.Step{
		{Lever: 0, At: 20, BP: 5, Closed: false},
		{Lever: 0, At: 60, BP: 7, Closed: true},
	}
	tests := []struct {
		tick       int64
		want       int64
		wantClosed bool
		wantOK     bool
	}{
		{10, 0, false, false},  // first cycle, before the first step: held
		{20, 5, false, true},   // first step
		{99, 7, true, true},    // last step of the first cycle
		{100, 7, true, true},   // u = 0: no step at or below, the last is carried
		{110, 7, true, true},   // u = 10
		{120, 5, false, true},  // u = 20: the first step again
		{170, 7, true, true},   // u = 70
		{1_000, 7, true, true}, // u = 0 after ten cycles
	}
	for _, tt := range tests {
		got, closed, ok := policy.ScheduleValue(steps, 0, 100, tt.tick)
		if ok != tt.wantOK || (ok && (got != tt.want || closed != tt.wantClosed)) {
			t.Errorf("ScheduleValue(tick %d, cycle 100) = %d, %v, %v; want %d, %v, %v",
				tt.tick, got, closed, ok, tt.want, tt.wantClosed, tt.wantOK)
		}
	}
}

// TestSchedule_OtherLeversIgnored verifies that the steps of another lever are
// not read.
//
// Requirement: FR-PLC-061
func TestSchedule_OtherLeversIgnored(t *testing.T) {
	t.Parallel()
	steps := []policy.Step{
		{Lever: 0, At: 0, BP: 1},
		{Lever: 1, At: 0, BP: 2},
		{Lever: 1, At: 10, BP: 3},
	}
	if got, _, ok := policy.ScheduleValue(steps, 1, 0, 12); !ok || got != 3 {
		t.Errorf("ScheduleValue(lever 1, tick 12) = %d, %v; want 3, true", got, ok)
	}
	if got, _, ok := policy.ScheduleValue(steps, 0, 0, 12); !ok || got != 1 {
		t.Errorf("ScheduleValue(lever 0, tick 12) = %d, %v; want 1, true", got, ok)
	}
}

// TestSchedule_OrderDoesNotMatter verifies that the value of a lever is the same for
// the steps in any order, so a caller that does not sort them gets the contract of
// 8.2.1 and not the last step in its list.
//
// Requirement: FR-PLC-061
func TestSchedule_OrderDoesNotMatter(t *testing.T) {
	t.Parallel()
	sorted := []policy.Step{{Lever: 0, At: 0, BP: 1}, {Lever: 0, At: 10, BP: 2}, {Lever: 0, At: 20, BP: 3}}
	shuffled := []policy.Step{sorted[2], sorted[0], sorted[1]}
	for _, cycle := range []int64{0, 30} {
		for tick := int64(-1); tick < 70; tick++ {
			bp, closed, ok := policy.ScheduleValue(sorted, 0, cycle, tick)
			bp2, closed2, ok2 := policy.ScheduleValue(shuffled, 0, cycle, tick)
			if bp != bp2 || closed != closed2 || ok != ok2 {
				t.Errorf("cycle %d tick %d: sorted (%d, %v, %v), shuffled (%d, %v, %v)", cycle, tick, bp, closed, ok, bp2, closed2, ok2)
			}
		}
	}
}
