package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// anchorSeries steps the warm pack one for ticks ticks with the seed 1 and
// returns the CostAnchor of every MarketCleared of commodity name, by tick. extra
// is a document layered last (empty for none).
func anchorSeries(t *testing.T, name string, ticks int, extra string) []int64 {
	t.Helper()
	cfg := loadPackBytes(t, []byte(extra), "base.toml", "one.toml", "warm-one.toml")
	var names economysim.Names
	out := make([]int64, 0, ticks)
	w := newPackWorld(t, cfg, 1, economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) {
		switch v := e.(type) {
		case economysim.WorldCreated:
			names = v.Names
		case economysim.MarketCleared:
			if names.Commodities[v.Commodity] == name {
				out = append(out, int64(v.CostAnchor))
			}
		}
	})))
	for i := range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", i, err)
		}
	}
	return out
}

// TestCostAnchor_SeasonalCropKeepsItsAnchorAllYear verifies over the warm pack
// one that the cost anchor of staple, whose only maker grow_staple has a yield
// factor of 0 outside spring, is above 0 at every tick of the first 220, not only
// in spring (days 59 to 150), with the population held fixed so that growers
// exist throughout: a recipe with a factor of 0 this season is costed at its best
// factor of any season.
//
// Requirement: FR-PRO-054
//
// Fix: the anchor of staple was 0 from about tick 160, because the anchor skipped
// a recipe whose factor of the season is 0. With no anchor the belief floor fell
// to the reference band's 10%, and a staple glut sold at 10 against a reference of
// 96, a loop that emptied the grower role.
func TestCostAnchor_SeasonalCropKeepsItsAnchorAllYear(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	const ticks = 220
	// A fixed population keeps the growers in place: with no grower the recipe is
	// on nobody's list and has no anchor, which is a different rule.
	got := anchorSeries(t, "staple", ticks, "[world]\nentry_share = 0\nexit_share = 0\n")
	if len(got) != ticks {
		t.Fatalf("MarketCleared of staple = %d events, want %d", len(got), ticks)
	}
	for tick, a := range got {
		if a <= 0 {
			t.Errorf("CostAnchor of staple at tick %d = %d, want above 0 (grow_staple is spring-only)", tick, a)
			break
		}
	}
}

// TestCostAnchor_AnchorHoldBindsTheObservedWage verifies over the warm pack one
// that with anchor_hold at 10 000 the anchor of fuel (cut_fuel, no input and no
// tool, 14 units a tick: ceil(wage / 14)) never falls below its value at the
// starting wage of 2 512, ceil(2 512 / 14) = 180, and that without the key the
// observed wage pulls it below that within 150 ticks, so the floor is what held it.
//
// Requirement: FR-PRO-055, FR-CFG-107
//
// Fix: the cost anchor read the pooled observed wage alone. The observed wage
// falls with the prices (a wage of 80 was recorded), which lowers the anchor and
// with it the belief floor, so a price crash fed itself.
func TestCostAnchor_AnchorHoldBindsTheObservedWage(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	const ticks, want = 150, 180
	low := func(s []int64) int64 {
		m := s[0]
		for _, a := range s {
			m = min(m, a)
		}
		return m
	}
	if got := low(anchorSeries(t, "fuel", ticks, "")); got >= want {
		t.Fatalf("lowest CostAnchor of fuel without the floor = %d, want below %d (the control must show the loop)", got, want)
	}
	if got := low(anchorSeries(t, "fuel", ticks, "[world]\nanchor_hold = 10000\n")); got < want {
		t.Errorf("lowest CostAnchor of fuel with anchor_hold = 10000 is %d, want at least %d", got, want)
	}
}
