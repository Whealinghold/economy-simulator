package economysim

import (
	"fmt"
	"math"
	"slices"
	"testing"
)

// TestEntryTally_Limited verifies that the value is omitted when every count is
// 0, carries each count otherwise with the executor's and the plan's added, and
// ignores a region outside the tally.
//
// Requirement: FR-SPN-035
func TestEntryTally_Limited(t *testing.T) {
	t.Parallel()
	d := &Decisions{regionCap: 1, roleCap: 2, cooldown: 3, hysteresis: 4, exitCap: 5, noViableRole: 6, stranded: 7}
	tl := newEntryTally(2)
	if _, ok := tl.limited(7, 0); ok {
		t.Fatal("limited with all counts 0 = true, want false")
	}
	tl.addPlan(1, d)
	tl.addPlan(1, d) // two Decide calls add: 1+1, 2+2, ...
	tl.addExecuted(1, 10, 20, 30, 40, 50)
	tl.addExecuted(1, 1, 0, 0, 0, 0)
	got, ok := tl.limited(7, 1)
	want := EntryLimited{
		Tick: 7, Region: 1, Changed: 11, Entered: 20, Removed: 30, Unfunded: 40, Refused: 50,
		RegionCap: 2, RoleCap: 4, Cooldown: 6, Hysteresis: 8, ExitCap: 10, NoViableRole: 12, Stranded: 14,
	}
	if !ok || got != want {
		t.Errorf("limited = %+v, %v, want %+v, true", got, ok, want)
	}
	if _, ok := tl.limited(7, 0); ok {
		t.Error("limited for an untouched region = true, want false")
	}
	tl.addPlan(5, d)
	tl.addExecuted(-1, 1, 1, 1, 1, 1)
	if _, ok := tl.limited(7, 5); ok {
		t.Error("limited outside the tally = true, want false")
	}
	tl.reset()
	if _, ok := tl.limited(7, 1); ok {
		t.Error("limited after reset = true, want false")
	}
}

// TestEntryTally_Single verifies that each count alone makes the value present.
//
// Requirement: FR-SPN-035
func TestEntryTally_Single(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		d    Decisions
		exec [5]int64
	}{
		{"regionCap", Decisions{regionCap: 1}, [5]int64{}},
		{"roleCap", Decisions{roleCap: 1}, [5]int64{}},
		{"cooldown", Decisions{cooldown: 1}, [5]int64{}},
		{"hysteresis", Decisions{hysteresis: 1}, [5]int64{}},
		{"exitCap", Decisions{exitCap: 1}, [5]int64{}},
		{"noViableRole", Decisions{noViableRole: 1}, [5]int64{}},
		{"stranded", Decisions{stranded: 1}, [5]int64{}},
		{"changed", Decisions{}, [5]int64{1}},
		{"entered", Decisions{}, [5]int64{0, 1}},
		{"removed", Decisions{}, [5]int64{0, 0, 1}},
		{"unfunded", Decisions{}, [5]int64{0, 0, 0, 1}},
		{"refused", Decisions{}, [5]int64{0, 0, 0, 0, 1}},
	}
	for _, tt := range tests {
		tl := newEntryTally(1)
		tl.addPlan(0, &tt.d)
		tl.addExecuted(0, tt.exec[0], tt.exec[1], tt.exec[2], tt.exec[3], tt.exec[4])
		if _, ok := tl.limited(0, 0); !ok {
			t.Errorf("limited(%s) = false, want true", tt.name)
		}
	}
}

// TestEntryTally_Saturates verifies that counts add with saturation.
//
// Guard: a wrapped count would turn a huge limit into a negative one in the
// event; without this test a plain + could replace satAdd unnoticed.
func TestEntryTally_Saturates(t *testing.T) {
	t.Parallel()
	tl := newEntryTally(1)
	tl.addExecuted(0, math.MaxInt64, 0, 0, 0, 0)
	tl.addExecuted(0, 1, 0, 0, 0, 0)
	if got, _ := tl.limited(0, 0); got.Changed != math.MaxInt64 {
		t.Errorf("Changed = %d, want MaxInt64", got.Changed)
	}
}

// TestEntryTally_Allocs verifies that using a tally allocates nothing.
//
// Requirement: NFR-SPN-001
func TestEntryTally_Allocs(t *testing.T) {
	d := &Decisions{regionCap: 1}
	tl := newEntryTally(3)
	if got := testing.AllocsPerRun(100, func() {
		tl.reset()
		tl.addPlan(2, d)
		tl.addExecuted(2, 1, 1, 1, 1, 1)
		tl.limited(1, 2)
	}); got != 0 {
		t.Errorf("AllocsPerRun = %v, want 0", got)
	}
}

// TestSpawnerRuns verifies the three runs over twenty ticks of a holding
// condition: the pattern-A runs warn at ticks 1, 10 and 20, the pattern-B run at
// 10 and 20; each resets when the condition fails; the run saturates.
//
// Requirement: FR-SPN-036, FR-SPN-042, FR-SPN-047
func TestSpawnerRuns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		step  func(before int64, hold bool) (int64, bool)
		warns []int
	}{
		{"extinct", func(b int64, h bool) (int64, bool) {
			if h {
				return extinctRun(b, 0, true, false)
			}
			return extinctRun(b, 1, true, false)
		}, []int{1, 10, 20}},
		{"extinct configured", func(b int64, h bool) (int64, bool) { return extinctRun(b, 0, false, h) }, []int{1, 10, 20}},
		{"extinct never had", func(b int64, _ bool) (int64, bool) { return extinctRun(b, 0, false, false) }, nil},
		{"unfunded", func(b int64, h bool) (int64, bool) {
			if h {
				return unfundedRun(b, 3, 0)
			}
			return unfundedRun(b, 3, 1)
		}, []int{10, 20}},
		{"unfunded none due", func(b int64, _ bool) (int64, bool) { return unfundedRun(b, 0, 0) }, nil},
		{"below floor", func(b int64, h bool) (int64, bool) {
			if h {
				return belowFloorRun(b, 5, 1, 2)
			}
			return belowFloorRun(b, 5, 2, 2)
		}, []int{1, 10, 20}},
		{"below floor weight 1", func(b int64, _ bool) (int64, bool) { return belowFloorRun(b, 1, 0, 2) }, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var run int64
			var warns []int
			for i := 1; i <= 20; i++ {
				var w bool
				if run, w = tt.step(run, true); w {
					warns = append(warns, i)
				}
			}
			if !slices.Equal(warns, tt.warns) {
				t.Errorf("%s warned at %v, want %v", tt.name, warns, tt.warns)
			}
			if run, w := tt.step(run, false); run != 0 || w {
				t.Errorf("%s after the condition fails = %d, %v, want 0, false", tt.name, run, w)
			}
		})
	}
	if run, w := unfundedRun(math.MaxInt64, 1, 0); run != math.MaxInt64 || w {
		t.Errorf("unfundedRun at the saturated run = %d, %v, want MaxInt64, false", run, w)
	}
}

// TestMaxPrivation verifies the largest privation is taken over one role's
// agents only, and is 0 for a role with none.
//
// Requirement: FR-SPN-046
func TestMaxPrivation(t *testing.T) {
	t.Parallel()
	ag := []RegionAgent{{Role: 1, Privation: 3}, {Role: 2, Privation: 9}, {Role: 1, Privation: 7}, {Role: 1, Privation: 5}}
	tests := []struct {
		role RoleID
		want int64
	}{{1, 7}, {2, 9}, {3, 0}}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("role %d", tt.role), func(t *testing.T) {
			t.Parallel()
			if got := maxPrivation(ag, tt.role); got != tt.want {
				t.Errorf("maxPrivation(role %d) = %d, want %d", tt.role, got, tt.want)
			}
		})
	}
}
