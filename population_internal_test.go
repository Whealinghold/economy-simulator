package economysim

import (
	"slices"
	"testing"
)

// TestCanonical_PopulationCopies verifies that the canonical form of the module
// copies both schedules (a later change of the caller's slice does not reach it),
// keeps the knots in the order written, sorts the region rows by region, and
// keeps a nil module nil and an empty module on.
//
// Requirement: FR-CFG-069
func TestCanonical_PopulationCopies(t *testing.T) {
	t.Parallel()
	src := DefaultPopulation()
	src.Regions = []RegionPopulation{DefaultRegionPopulation("b"), DefaultRegionPopulation("a")}
	got := src.canonical()
	src.BirthSchedule[0].Rate = 1
	src.DeathSchedule[1].Ratio = 1
	if got.BirthSchedule[0].Rate != 250 || got.DeathSchedule[1].Ratio != 16_000 {
		t.Errorf("canonical schedules = %v, %v, want copies the caller's change does not reach", got.BirthSchedule, got.DeathSchedule)
	}
	if got.Regions[0].Region != "a" || got.Regions[1].Region != "b" || src.Regions[0].Region != "b" {
		t.Errorf("canonical rows = %v from %v, want sorted copies by region", got.Regions, src.Regions)
	}
	if !slices.Equal(got.BirthSchedule, []Knot{{8_000, 250}, {16_000, 250}}) {
		t.Errorf("canonical birth schedule = %v, want the knots in order", got.BirthSchedule)
	}
	var nilModule *Population
	if nilModule.canonical() != nil {
		t.Errorf("canonical of a nil module is not nil")
	}
	if empty := (&Population{}).canonical(); empty == nil {
		t.Errorf("canonical of an empty module is nil, want it on")
	}
}
