package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// TestStartWindows_WrapsAcrossTheYearEnd verifies that a job started late in the
// year and completed early in the next is counted in the window it started in.
//
// Arithmetic: a year of 10 ticks, open on ticks 6 to 9 (the window [6, 10)). A
// job of duration 3 completing at tick 1 started at 1 - 3 + 1 = -1, that is tick
// 9 of the year, inside the window; a completion at tick 8 started at 6.
//
// Fix: stage-2-9c review. The start was not taken modulo the year, so the first
// completion matched no window and the window's draw was undercounted.
func TestStartWindows_WrapsAcrossTheYearEnd(t *testing.T) {
	t.Parallel()
	a := viaActivity{
		runs:  []steady.Run{{Length: 6, FactorBP: 0}, {Length: 4, FactorBP: 10_000}},
		count: steady.StartCount{Completions: []int64{1, 8}},
		act:   steady.Activity{Duration: 3},
	}
	want := []steady.Window{{Start: 6, End: 10, Starts: 2}}
	if got := startWindows(a); !slices.Equal(got, want) {
		t.Errorf("startWindows(year 10, window [6,10), completions 1 and 8, duration 3) = %v, want %v", got, want)
	}
}

// TestWarnOnce_SkipsARepeatedRuleAndPath verifies that the same rule at the same
// path is reported once, and that a different path or rule is kept.
//
// Guard: endowment_season_peak is raised per recipe draw, so two recipes drawing
// one resource would otherwise report the resource twice.
func TestWarnOnce_SkipsARepeatedRuleAndPath(t *testing.T) {
	t.Parallel()
	var o viabilityOutcome
	o.warnOnce("rule", "a", "b")
	o.warnOnce("rule", "a", "b")
	o.warnOnce("rule", "a", "c")
	o.warnOnce("other", "a", "b")
	if got, want := len(o.warnings), 3; got != want {
		t.Errorf("warnOnce(rule/a/b twice, rule/a/c, other/a/b) kept %d warnings %v, want %d", got, o.warnings, want)
	}
}

// TestSupplies_UnknownResourceIsNotSupplied verifies that a draw naming a
// resource the module does not hold is not supplied, and that a nil supply (the
// module off) and a recipe with no draw are.
//
// Guard: Validate rejects the unknown name first (ErrUnknownResource), so the
// branch is defensive; without the test a change to the order of the checks
// could let a draw of nothing count as sourced.
//
// Requirement: FR-RES-037
func TestSupplies_UnknownResourceIsNotSupplied(t *testing.T) {
	t.Parallel()
	s := &resSupply{c: &Config{}, idx: map[string]int{}}
	draws := []Draw{{Resource: "ghost", Quantity: 1}}
	if s.supplies("north", draws) {
		t.Errorf("supplies(north, draw of an unknown resource) = true, want false")
	}
	if !s.supplies("north", nil) {
		t.Errorf("supplies(north, no draw) = false, want true")
	}
	if !(*resSupply)(nil).supplies("north", draws) {
		t.Errorf("supplies on a nil supply (module off) = false, want true")
	}
}

// TestSeasonFactors_GapAndUnlistedSeason verifies that a stretch outside every
// season, and a season the profile does not list, both read as factor 0, and a
// listed season reads its own factor.
//
// Arithmetic: the profile lists "wet" at 5 000; the year is a gap of 3 ticks, "wet"
// for 2, "dry" (unlisted) for 4: factors 0, 5 000, 0.
//
// Requirement: FR-RES-012, FR-RES-039
func TestSeasonFactors_GapAndUnlistedSeason(t *testing.T) {
	t.Parallel()
	cfg := &Config{Calendar: Calendar{Seasons: []Season{{ID: "wet"}, {ID: "dry"}}}}
	s := &resSupply{c: cfg}
	r := &Resource{Seasons: []SeasonRegen{{Season: "wet", BP: 5_000}}}
	got := s.seasonFactors(r, []seasonRun{{season: -1, length: 3}, {season: 0, length: 2}, {season: 1, length: 4}})
	want := []steady.Run{{Length: 3}, {Length: 2, FactorBP: 5_000}, {Length: 4}}
	if !slices.Equal(got, want) {
		t.Errorf("seasonFactors(profile wet=5000, runs gap 3, wet 2, dry 4) = %v, want %v", got, want)
	}
	if got := s.seasonFactors(&Resource{}, []seasonRun{{season: 0, length: 1}}); got != nil {
		t.Errorf("seasonFactors(no profile) = %v, want nil", got)
	}
}

// TestEndowments_SaturateTheReport verifies that a rent past MaxMoney is
// reported as MaxMoney, and that a headroom or a runway past int64 is reported as
// the largest int64, not wrapped.
//
// Arithmetic: a rent of 2 * MaxMoney is above the bound. A deposit (no rate) of
// 2^62 units drawn at 1/4 a tick has a runway of 2^64 ticks and a headroom of
// 2^62 / (1/4) * 10 000 basis points, both past 2^63 - 1.
//
// Guard: money and quantities are never wrapped (architecture.md 3.5); an
// overflow in the report would show a small, wrong number to the operator.
//
// Requirement: FR-RES-040
func TestEndowments_SaturateTheReport(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Regions: []Region{{ID: "north"}},
		Modules: Modules{Resources: &Resources{Resources: []Resource{{ID: "deposit", Shape: ShapeConstant}}}},
	}
	s := &viaSolve{m: &viaModel{c: cfg}}
	p := &viaProgramme{
		regions: []RegionID{0},
		md:      steady.Model{Resources: []steady.Endow{{Resource: 0, Region: 0, Stock: 1 << 62}}},
		out: steady.Outcome{
			ResRent: []steady.Frac{steady.Rat(2*MaxMoney, 1)},
			ResPhi:  []steady.Frac{steady.Rat(1<<62, 1)},
			ResDraw: []steady.Frac{steady.Rat(1, 4)},
		},
	}
	s.endowments(p, steady.Rat(1, 1))
	if len(s.rep.Rent) != 1 || s.rep.Rent[0].Rent != MaxMoney {
		t.Errorf("endowments rent 2*MaxMoney = %v, want one row at MaxMoney %d", s.rep.Rent, int64(MaxMoney))
	}
	if len(s.rep.Headroom) != 1 || s.rep.Headroom[0].BP != maxInt64 {
		t.Errorf("endowments headroom past int64 = %v, want one row at %d", s.rep.Headroom, int64(maxInt64))
	}
	if len(s.rep.Runway) != 1 || s.rep.Runway[0].Ticks != maxInt64 {
		t.Errorf("endowments runway past int64 = %v, want one row at %d", s.rep.Runway, int64(maxInt64))
	}
}
