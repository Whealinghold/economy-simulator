package calibration

import (
	"errors"
	"math"
	"slices"
	"testing"
)

func mustGrid(t *testing.T, r Rule, base int64, b Bounds, off Off) Grid {
	t.Helper()
	g, err := Build(r, base, b, off, nil)
	if err != nil {
		t.Fatalf("Build(%v, %d, %v) error = %v", r, base, b, err)
	}
	return g
}

// TestPlan_Screen verifies that the screen takes the base and the two neighbours
// of each knob, 1+2k points for k knobs, fewer where a side is an edge.
//
// Requirement: FR-CLB-022
func TestPlan_Screen(t *testing.T) {
	t.Parallel()
	both := mustGrid(t, geo(), 7000, bnds(1, 1_000_000), noOff()) // 3500, 7000, 14000
	if got, want := ScreenLevels(both), []int64{3500, 7000, 14000}; !slices.Equal(got, want) {
		t.Errorf("ScreenLevels(CostFloor) = %v, want %v", got, want)
	}
	edge := mustGrid(t, lin(500), 0, bnds(0, 9999), offAt(0)) // base 0: one neighbour, 500
	if got, want := ScreenLevels(edge), []int64{0, 500}; !slices.Equal(got, want) {
		t.Errorf("ScreenLevels(TargetSpread) = %v, want %v", got, want)
	}
	tests := []struct {
		name  string
		grids []Grid
		want  int
	}{
		{"no knob", nil, 1},                       // the base alone
		{"one knob", []Grid{both}, 3},             // 1 + 2
		{"two knobs", []Grid{both, both}, 5},      // 1 + 2*2
		{"one at an edge", []Grid{both, edge}, 4}, // 1 + 2 + 1
	}
	for _, tt := range tests {
		if got := ScreenPoints(tt.grids...); got != tt.want {
			t.Errorf("ScreenPoints(%s) = %d, want %d", tt.name, got, tt.want)
		}
	}
	if got := Replicas(true, false); got < 10 {
		t.Errorf("Replicas(search) = %d, want at least 10: a band is never set from fewer than 10 seeds", got)
	}
}

// TestPlan_Explore verifies the size of the explore, max(16, 10k) points for k
// screened-in knobs, its interval over the kept valid levels and the replicas of
// a search.
//
// Requirement: FR-CLB-023, FR-CLB-022
func TestPlan_Explore(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ k, want int }{{0, 16}, {1, 16}, {2, 20}, {3, 30}, {6, 60}} {
		if got, err := ExploreSize(tt.k); err != nil || got != tt.want {
			t.Errorf("ExploreSize(%d) = %d, %v, want %d", tt.k, got, err, tt.want) // max(16, 10k)
		}
	}
	for _, k := range []int{-1, math.MaxInt} {
		if _, err := ExploreSize(k); !errors.Is(err, ErrCount) {
			t.Errorf("ExploreSize(%d) error = %v, want ErrCount", k, err)
		}
	}
}

// TestPlan_ExploreSpan verifies the replicas of a search and the interval the
// explore covers over the kept valid levels.
//
// Requirement: FR-CLB-023, FR-CLB-022
func TestPlan_ExploreSpan(t *testing.T) {
	t.Parallel()
	for _, tierL := range []bool{false, true} {
		if got := Replicas(true, tierL); got != 10 {
			t.Errorf("Replicas(search, tier L %v) = %d, want 10", tierL, got)
		}
	}
	g := mustGrid(t, geo(), 7000, bnds(1, 1_000_000), noOff())
	// The second level below 7000 is 1750 and the second above is 28000: five levels.
	s := Explore(g)
	if got := g.Levels[s.Lo : s.Hi+1]; !slices.Equal(got, []int64{1750, 3500, 7000, 14000, 28000}) {
		t.Errorf("Explore(CostFloor) covers %v, want 1750..28000", got)
	}
	// A base with one neighbour has a shorter interval: 0, 500, 1000.
	e := mustGrid(t, lin(500), 0, bnds(0, 9999), offAt(0))
	if s := Explore(e); s.Lo != 0 || s.Hi != 2 {
		t.Errorf("Explore(TargetSpread) = %v, want {0 2}", s)
	}
}

// TestPlan_Confirm verifies that the confirm runs the chosen configuration and
// the base as a control, on the same seeds, with the replicas of the confirm: 50
// on seeds 1 to 50, and 10 on 1 to 10 at tier L.
//
// Requirement: FR-CLB-025, FR-CLB-021
func TestPlan_Confirm(t *testing.T) {
	t.Parallel()
	c := ConfirmPlan("chosen", "base", false)
	if c.Arms != [2]string{"chosen", "base"} || c.First != 1 || c.Last != 50 || c.Replicas != 50 {
		t.Errorf("ConfirmPlan(chosen, base) = %+v, want arms chosen and base on seeds 1..50, 50 replicas", c)
	}
	if c.Arms[1] != "base" {
		t.Errorf("ConfirmPlan control = %q, want the base", c.Arms[1])
	}
	if l := ConfirmPlan("a", "b", true); l.Last != 10 || l.Replicas != 10 || !l.TierL {
		t.Errorf("ConfirmPlan(tier L) = %+v, want seeds 1..10 and 10 replicas", l)
	}
}
