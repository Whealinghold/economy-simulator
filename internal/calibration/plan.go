package calibration

import "github.com/Whealinghold/economy-sim/internal/checked"

// The plan of a stage (spec-calibration.md 7.2): the screen over the base and the
// two neighbours of each knob, the explore over a spread of the screened-in
// knobs, and the confirm of the chosen configuration against the base.
const (
	// exploreMin and explorePerKnob give max(16, 10k) points for k screened-in
	// knobs (FR-CLB-023), placeholders of 7.2.2.
	exploreMin     = 16
	explorePerKnob = 10
	// widenLevels is the two kept levels by which an explore interval is widened
	// when its feasible points touch an end that is not an edge (5.3, FR-CLB-026).
	widenLevels = 2
	// exploreSpan is the levels each side of the base that the explore covers: the
	// second level below to the second above (5.3).
	exploreSpan = 2
)

// ErrCount is returned for a negative number of knobs or one whose point count
// leaves int.
const ErrCount = calError("calibration: knob count negative or too large")

// ScreenLevels returns the levels of the screen of a knob: the base and the
// neighbours it has: ascending for `geo` and `lin`, in the order the levels were
// given for a `set` (FR-CLB-022). A side that is an edge gives none.
func ScreenLevels(g Grid) []int64 {
	if g.Base < 0 || g.Base >= len(g.Levels) { // the zero or a forged Grid: no base to screen around
		return nil
	}
	var out []int64
	if g.Lower.Present {
		out = append(out, g.Lower.Level)
	}
	out = append(out, g.Levels[g.Base])
	if g.Upper.Present {
		out = append(out, g.Upper.Level)
	}
	return out
}

// ScreenPoints returns the number of points of the one-at-a-time screen of the
// given knobs: the base once and the neighbours each knob has, 1+2k for k knobs
// with both, fewer at an edge (FR-CLB-022).
func ScreenPoints(grids ...Grid) int {
	n := 1
	for _, g := range grids {
		if l := ScreenLevels(g); len(l) > 0 {
			n += len(l) - 1
		}
	}
	return n
}

// ExploreSize returns the points of the explore for k screened-in knobs,
// max(16, 10k) (FR-CLB-023): 16 for one knob, 20 for two.
func ExploreSize(k int) (int, error) {
	if k < 0 {
		return 0, ErrCount
	}
	n, ok := checked.Mul(int64(k), explorePerKnob)
	if !ok || int64(int(n)) != n {
		return 0, ErrCount
	}
	return max(exploreMin, int(n)), nil
}

// Replicas returns the replicas of a point: 10 in a search, and in a confirm 50,
// or 10 at tier L (FR-CLB-022, FR-CLB-021).
func Replicas(search, tierL bool) int {
	if search {
		return searchReplicas
	}
	_, last := ConfirmSeeds(tierL)
	return int(last)
}

// Span is an interval of a grid, as inclusive indices into Grid.Levels.
type Span struct{ Lo, Hi int }

// Explore returns the interval the explore covers: the second kept valid level
// below the base to the second above, shorter where an edge is in the way
// (5.3).
func Explore(g Grid) Span {
	if g.Base < 0 || g.Base >= len(g.Levels) { // the zero or a forged Grid: an empty interval
		return Span{Lo: 0, Hi: -1}
	}
	return Span{Lo: max(0, g.Base-exploreSpan), Hi: min(len(g.Levels)-1, g.Base+exploreSpan)}
}

// AtEdge reports whether the end of the interval is the end of the grid, an
// edge that cannot be widened (a bound, the off value or an invalid level lies
// beyond it), as opposed to an end that is not an edge.
func AtEdge(g Grid, s Span, upper bool) bool {
	if upper {
		return s.Hi >= len(g.Levels)-1
	}
	return s.Lo <= 0
}

// Widen returns the interval widened by two kept valid levels on one side, never
// beyond the grid, so never outside [Lo, Hi] and never to a cut off value
// (FR-CLB-026). It reports false when that end is an edge.
func Widen(g Grid, s Span, upper bool) (Span, bool) {
	if AtEdge(g, s, upper) {
		return s, false
	}
	if upper {
		s.Hi = min(len(g.Levels)-1, s.Hi+widenLevels)
	} else {
		s.Lo = max(0, s.Lo-widenLevels)
	}
	return s, true
}

// Confirm is the plan of a confirm: the chosen configuration and the base as the
// control, on the same seeds, measures and windows (FR-CLB-025).
type Confirm struct {
	Arms        [2]string // the chosen configuration, then the control (the base)
	First, Last uint64    // the confirm seeds
	Replicas    int
	TierL       bool
}

// ConfirmPlan returns the confirm of a choice against its base.
func ConfirmPlan(chosen, base string, tierL bool) Confirm {
	first, last := ConfirmSeeds(tierL)
	return Confirm{Arms: [2]string{chosen, base}, First: first, Last: last, Replicas: Replicas(false, tierL), TierL: tierL}
}
