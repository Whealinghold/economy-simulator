package economysim

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// TestViaModel_AddMeansOverflow verifies that a weighted sum that does not fit
// int64 switches the nominal warnings off for the region instead of wrapping, and
// that the sums of a region that fits are exact.
//
// Requirement: FR-VIA-009
func TestViaModel_AddMeansOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		households int64
		role       Role
		wantOK     bool
	}{
		// 10 households * 250 expected ticks = 2 500 and * 15 000 target = 150 000.
		{"fits", 10, Role{ExpectedWorkTicks: 250, TargetRatio: 15_000}, true},
		// MaxInt64 households * 3 ticks does not fit.
		{"expected work overflows", math.MaxInt64, Role{ExpectedWorkTicks: 3, TargetRatio: 1}, false},
		// MaxInt64 / 2 households * 1 tick fits, * 3 target does not.
		{"target overflows", math.MaxInt64 / 2, Role{ExpectedWorkTicks: 1, TargetRatio: 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &viaModel{expected: make([]int64, 1), targets: make([]int64, 1), sumsOK: []bool{true}}
			m.addMeans(0, tt.households, &tt.role)
			if m.sumsOK[0] != tt.wantOK {
				t.Errorf("addMeans(%d households, %+v) ok = %v, want %v", tt.households, tt.role, m.sumsOK[0], tt.wantOK)
			}
			if tt.wantOK && (m.expected[0] != 2_500 || m.targets[0] != 150_000) {
				t.Errorf("sums = %d, %d, want 2 500 and 150 000", m.expected[0], m.targets[0])
			}
			// A second group that fits does not undo the switch-off.
			m.addMeans(0, 1, &Role{ExpectedWorkTicks: 1, TargetRatio: 1})
			if m.sumsOK[0] != tt.wantOK {
				t.Errorf("addMeans after the first: ok = %v, want %v", m.sumsOK[0], tt.wantOK)
			}
		})
	}
}

// TestCheckViability_NominalWarningsStopOnOverflow verifies that a region whose
// sums or work-share denominator do not fit int64 raises neither nominal warning
// and does not panic.
//
// Requirement: FR-VIA-009
func TestCheckViability_NominalWarningsStopOnOverflow(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Roles[0].ExpectedWorkTicks, c.Roles[0].MinWorkShare = 20, 0 // would draw subsistence_exceeds_expected_work
	c.Recipes[0].Outputs[0].Quantity = 2                          // and target_above_surplus
	c = c.canonical()
	base := newViaModel(&c)
	var o viabilityOutcome
	base.regions(&o)
	for _, rule := range []string{"subsistence_exceeds_expected_work", "target_above_surplus"} {
		if !slices.ContainsFunc(o.warnings, func(w Warning) bool { return w.Rule == rule }) {
			t.Fatalf("regions(base) has no %s in %v, want it: the case must be able to fire", rule, o.warnings)
		}
	}
	for _, tt := range []struct {
		name string
		edit func(m *viaModel)
	}{
		{"sums do not fit", func(m *viaModel) { m.sumsOK[0] = false }},
		{"denominator does not fit", func(m *viaModel) { m.households[0] = math.MaxInt64 / 100 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newViaModel(&c)
			tt.edit(m)
			var o viabilityOutcome
			m.nominal(&o, 0)
			if tt.name == "sums do not fit" {
				m.autarky(&o, 0) // the sums gate the nominal programme too
			}
			if slices.ContainsFunc(o.warnings, func(w Warning) bool { return w.Rule == "target_above_surplus" && tt.name != "sums do not fit" }) {
				t.Errorf("nominal(%s) warned target_above_surplus, want none", tt.name)
			}
			if tt.name == "sums do not fit" && slices.ContainsFunc(o.warnings, func(w Warning) bool { return w.Rule == "subsistence_exceeds_expected_work" }) {
				t.Errorf("autarky(%s) warned subsistence_exceeds_expected_work, want none", tt.name)
			}
		})
	}
}

// TestCheckViability_SolverFailureIsAFault verifies that an error of the solver
// that is not a limit (here a total upkeep that does not fit int64 once the
// regions are pooled) is kept as a fault and not as a finding: the programme is
// not a verdict on the configuration.
//
// Requirement: FR-VIA-014
func TestCheckViability_SolverFailureIsAFault(t *testing.T) {
	t.Parallel()
	c := viaToy()
	viaTwoRegions(&c)
	c = c.canonical()
	m := newViaModel(&c)
	m.upkeep[0][0], m.upkeep[1][0] = math.MaxInt64, math.MaxInt64
	var o viabilityOutcome
	m.labour(&o)
	if !errors.Is(o.fault, ErrSolveInconsistent) || len(o.errs) != 0 {
		t.Errorf("labour with an upkeep total of 2^64 - 2: fault = %v, errs = %v, want a fault and no finding", o.fault, o.errs)
	}
	if _, _, err := m.solve(m.model(m.withHouseholds, steady.Ratio{Num: 1, Den: 1})); !errors.Is(err, steady.ErrOverflow) {
		t.Errorf("solve = %v, want steady.ErrOverflow", err)
	}
}

// TestViaModel_ActivitiesOfUnprofiledRecipesAreSharedAcrossRegions verifies that
// a recipe with no yield row reads the same in every region: an unsited one
// exists in all of them with equal counts, a sited one in none, and neither is
// recomputed per region.
//
// Requirement: FR-VIA-002, FR-PRO-020
func TestViaModel_ActivitiesOfUnprofiledRecipesAreSharedAcrossRegions(t *testing.T) {
	t.Parallel()
	c := viaToy()
	for _, id := range []string{"east", "south"} {
		r := DefaultRegion()
		r.ID, r.ReferenceBasket = id, 100
		c.Regions = append(c.Regions, r)
		c.Groups = append(c.Groups, Group{Region: id, Role: "farmer", Households: 3})
	}
	c.Commodities = append(c.Commodities, viaCommodity("rock", 50, nil))
	c.Recipes = append(c.Recipes, viaRecipe("quarry", nil, []Output{viaOut("rock", 1)}, func(r *Recipe) { r.Sited = true }))
	c.Roles[0].Recipes = []string{"farm", "quarry"}
	c = c.canonical()
	m := newViaModel(&c)
	if len(m.withHouseholds) != 3 {
		t.Fatalf("withHouseholds = %v, want three regions", m.withHouseholds)
	}
	first := m.acts[0]
	if len(first) != 1 || c.Recipes[first[0].recipe].ID != "farm" {
		t.Fatalf("activities of the first region = %v, want farm only", first)
	}
	for _, r := range m.withHouseholds[1:] {
		got := m.acts[r]
		if len(got) != 1 || got[0].region != r || got[0].count.Starts != first[0].count.Starts || !slices.Equal(got[0].runs, first[0].runs) {
			t.Errorf("activities of region %d = %+v, want the farm of region 0 with its own region", r, got)
		}
	}
}

// TestShareAboveMean verifies the comparison of the labour share to subsist with
// the mean expected work share: strictly above only, at the rounded-down mean
// exactly, and false when the products do not fit int64.
//
// Requirement: FR-VIA-009
func TestShareAboveMean(t *testing.T) {
	t.Parallel()
	// 10 households expecting 250 of 365 ticks: expected 2 500, mean
	// 2 500 * 10 000 / (365 * 10) = 6 849.3, floored to 6 849.
	tests := []struct {
		name                     string
		share, expected, year, h int64
		want                     bool
	}{
		{"above", 6_850, 2_500, 365, 10, true},
		{"at the floor of the mean", 6_849, 2_500, 365, 10, false},
		{"below", 2_500, 2_500, 365, 10, false},
		{"year times households overflows", 1, 1, math.MaxInt64, 2, false},
		{"expected times 10 000 overflows", 1, math.MaxInt64, 1, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shareAboveMean(tt.share, tt.expected, tt.year, tt.h); got != tt.want {
				t.Errorf("shareAboveMean(%d, %d, %d, %d) = %v, want %v", tt.share, tt.expected, tt.year, tt.h, got, tt.want)
			}
		})
	}
}

// TestTargetAboveSurplus verifies the comparison of the mean target ratio with
// the nominal surplus factor: strictly above only, and false when the product
// does not fit.
//
// Requirement: FR-VIA-009
func TestTargetAboveSurplus(t *testing.T) {
	t.Parallel()
	// 10 households with a target of 15 000 sum to 150 000; a surplus of 15 000 over
	// 10 households is a limit of 150 000.
	tests := []struct {
		name                string
		targets, surplus, h int64
		want                bool
	}{
		{"above", 150_001, 15_000, 10, true},
		{"equal", 150_000, 15_000, 10, false},
		{"below", 150_000, 15_001, 10, false},
		{"product overflows", math.MaxInt64, math.MaxInt64, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := targetAboveSurplus(tt.targets, tt.surplus, tt.h); got != tt.want {
				t.Errorf("targetAboveSurplus(%d, %d, %d) = %v, want %v", tt.targets, tt.surplus, tt.h, got, tt.want)
			}
		})
	}
}

// TestValidate_LabourShortNeedsASurplusBelowOne verifies the boundary of V3: a
// pooled surplus factor of exactly 1 (a farmer makes the 1 grain a household
// eats) is viable, and 1 less in basis points is not.
//
// Requirement: FR-VIA-007
func TestValidate_LabourShortNeedsASurplusBelowOne(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		duration int64
		out      Quantity
		bad      bool
	}{
		{"exactly 1", 1, 1, false},
		// 3 grain in 4 ticks is 0.75, and 2 in 2 ticks is 1: the first is short.
		{"0.75", 4, 3, true},
		{"two in two", 2, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := viaToy()
			c.Recipes[0].Duration, c.Recipes[0].Outputs[0].Quantity = tt.duration, tt.out
			err := c.Validate()
			if bad := errors.Is(err, ErrNotViable); bad != tt.bad {
				t.Errorf("Validate(%d in %d ticks) = %v, want not viable = %v", tt.out, tt.duration, err, tt.bad)
			}
		})
	}
}

// TestViaModel_SolveGateIsInclusive verifies the size gate at its edge: a
// tableau of exactly the gate's cells runs, and one cell fewer allowed skips it.
//
// Requirement: FR-VIA-010, FR-CFG-043
func TestViaModel_SolveGateIsInclusive(t *testing.T) {
	t.Parallel()
	c := viaToy().canonical()
	m := newViaModel(&c)
	md := m.model(m.withHouseholds, steady.Ratio{Num: 1, Den: 1})
	rows, cols := steady.Size(md)
	cells := int64(rows) * int64(cols) // 2 rows (grain, labour) by 4 columns (farm, lambda, two slacks): 8 cells
	for _, tt := range []struct {
		name    string
		limit   int64
		skipped bool
	}{
		{"exactly the size", cells, false},
		{"one cell fewer", cells - 1, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mm := *m
			mm.maxCells = tt.limit
			_, skipped, err := mm.solve(md)
			if err != nil || skipped != tt.skipped {
				t.Errorf("solve with a gate of %d cells (the tableau has %d) = skipped %v, error %v, want skipped %v", tt.limit, cells, skipped, err, tt.skipped)
			}
		})
	}
}
