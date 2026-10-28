package economysim

import "testing"

// TestRowCapacity_FallsBackToTheResource verifies the capacity a row's initial
// stock is checked against: the row's own, else the resource's default, else 0 when
// the resource is unknown (spec-resources.md 3.3).
//
// Requirement: FR-RES-004
func TestRowCapacity_FallsBackToTheResource(t *testing.T) {
	t.Parallel()
	res := &Resource{Capacity: 700}
	for _, tt := range []struct {
		name string
		row  Endowment
		res  *Resource
		want int64
	}{
		{"own capacity wins", Endowment{HasCapacity: true, Capacity: 300}, res, 300},
		{"own capacity of 0 wins", Endowment{HasCapacity: true}, res, 0},
		{"the resource's default", Endowment{}, res, 700},
		{"unknown resource", Endowment{}, nil, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rowCapacity(&tt.row, tt.res); got != tt.want {
				t.Errorf("rowCapacity(%+v, %+v) = %d, want %d", tt.row, tt.res, got, tt.want)
			}
		})
	}
}

// TestSiteAt_Precedence verifies the site factor of a recipe at a season in the
// order of spec-production.md 6.2: the entry of the profile, else All, else 0 when
// the profile lists seasons, else 0 for a Sited recipe and 10 000 for one that is not.
//
// Requirement: FR-RES-035
func TestSiteAt_Precedence(t *testing.T) {
	t.Parallel()
	seasons := []Season{{ID: "spring", Days: 100}, {ID: "summer", Days: 100}}
	listed := &YieldProfile{Seasons: []SeasonYield{{Season: "summer", BP: 4_000}}}
	withAll := &YieldProfile{HasAll: true, All: 6_000, Seasons: []SeasonYield{{Season: "summer", BP: 4_000}}}
	for _, tt := range []struct {
		name  string
		y     *YieldProfile
		sited bool
		s     int
		want  int64
	}{
		{"entry of the season", listed, true, 1, 4_000},
		{"all for a season without an entry", withAll, true, 0, 6_000},
		{"listed seasons and no all give 0", listed, false, 0, 0},
		{"no season uses all", withAll, true, len(seasons), 6_000},
		{"no profile, sited", nil, true, 0, 0},
		{"no profile, not sited", nil, false, 0, wholeBP},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := siteAt(tt.y, tt.sited, seasons, tt.s); got != tt.want {
				t.Errorf("siteAt(%+v, sited %v, season %d) = %d, want %d", tt.y, tt.sited, tt.s, got, tt.want)
			}
		})
	}
}

// TestNoComparativeAdvantage_InvalidInputsAreFalse verifies that the cross-ratio
// condition is false, not a panic, for a configuration it cannot evaluate: fewer
// than two sited recipes, fewer than two regions, and a calendar with no valid year.
//
// Guard: the condition reads the calendar and the region graph of a configuration
// that has not been validated when the warnings run on a rejected one, so a zero
// calendar must read as "no".
func TestNoComparativeAdvantage_InvalidInputsAreFalse(t *testing.T) {
	t.Parallel()
	region := func(id string) Region { r := DefaultRegion(); r.ID = id; return r }
	sited := []Recipe{{ID: "a", Sited: true}, {ID: "b", Sited: true}}
	for _, tt := range []struct {
		name string
		c    Config
	}{
		{"one sited recipe", Config{Recipes: sited[:1], Regions: []Region{region("x"), region("y")}}},
		{"one region", Config{Recipes: sited, Regions: []Region{region("x")}}},
		{"zero calendar", Config{Recipes: sited, Regions: []Region{region("x"), region("y")}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if noComparativeAdvantage(&tt.c) {
				t.Errorf("noComparativeAdvantage(%s) = true, want false", tt.name)
			}
		})
	}
	if m := meanSites(&Config{}, nil); m != nil {
		t.Errorf("meanSites(zero calendar) is not nil, want nil")
	}
}
