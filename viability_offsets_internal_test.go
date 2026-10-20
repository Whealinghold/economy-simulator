package economysim

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// viaSeasonal returns the toy with a farm that runs a job of d ticks, sited with
// the given yield profile in the north.
func viaSeasonal(d int64, y YieldProfile) Config {
	c := viaToy()
	c.Recipes[0].Duration = d
	c.Recipes[0].Sited = true
	c.Recipes[0].Outputs[0].Quantity = Quantity(10 * d) // 10 grain a tick of the job
	y.Region, y.Recipe = "north", "farm"
	c.Yields = []YieldProfile{y}
	return c
}

// viaBruteOffsets is the definition of 8.5.2 spelled out: the offsets o in 0 .. d-1
// whose virtual tick -o has a positive yield factor by Calendar.SeasonAt.
func viaBruteOffsets(c *Config, m *viaModel, region RegionID, recipe RecipeID) []int64 {
	var out []int64
	rc := c.Recipes[recipe]
	for o := range rc.Duration {
		f, _, _ := m.yt.effective(&yieldShocks{}, region, recipe, c.Calendar.SeasonAt(-o), rc.Sited)
		if f > 0 {
			out = append(out, o)
		}
	}
	return out
}

// TestVirtualStartOffsets verifies the offsets of 8.5.2 on the default calendar
// (spring is days 59 to 150 of 365): a 60-tick spring job has none at tick 0,
// because the 60 ticks before it are winter and autumn, and a year-long one has
// the 92 offsets 215 to 306 (tick -215 is day 150, tick -306 is day 59).
//
// Requirement: FR-VIA-053, FR-VIA-033
func TestVirtualStartOffsets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		d     int64
		y     YieldProfile
		first int64 // first offset, -1 for none
		n     int
	}{
		{"60 ticks, spring only", 60, viaSpring("", ""), -1, 0},
		{"100 ticks, spring only", 100, viaSpring("", ""), -1, 0},
		{"a year, spring only", 365, viaSpring("", ""), 215, 92}, // days 59 to 150: o = 365 - day, 215 to 306
		{"all year", 12, YieldProfile{HasAll: true, All: 10_000}, 0, 12},
		{"summer only, 200 ticks", 200, YieldProfile{Seasons: []SeasonYield{{Season: "summer", BP: 10_000}}}, 123, 77}, // summer is days 151 to 242: o from 365-242=123 to 199
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := viaSeasonal(tt.d, tt.y).canonical()
			m := newViaModel(&c)
			got := virtualStartOffsets(&c, &m.yt, 0, 0)
			if len(got) != tt.n || (tt.n > 0 && got[0] != tt.first) {
				t.Errorf("virtualStartOffsets(%s) = %d offsets from %v, want %d from %d", tt.name, len(got), got[:min(3, len(got))], tt.n, tt.first)
			}
			if !slices.Equal(got, viaBruteOffsets(&c, m, 0, 0)) {
				t.Errorf("virtualStartOffsets(%s) differs from the definition by SeasonAt", tt.name)
			}
			if empty := noVirtualStart(m.runsFor(0, 0), m.year, tt.d); empty != (tt.n == 0) {
				t.Errorf("noVirtualStart(%s) = %v, want %v", tt.name, empty, tt.n == 0)
			}
		})
	}
}

// TestNoVirtualStart_AgreesWithTheOffsets verifies, for every duration up to
// 400 and several yield profiles and calendars, that the run-based test of an
// empty set agrees with the list of offsets, and that the offsets are ascending
// and inside [0, d).
//
// Requirement: FR-VIA-053
//
// Guard: Validate uses the cheap test and the world will use the list; if they
// disagreed a job could pass validation and have nowhere to start.
func TestNoVirtualStart_AgreesWithTheOffsets(t *testing.T) {
	t.Parallel()
	profiles := map[string]YieldProfile{
		"spring":       viaSpring("", ""),
		"winter":       {Seasons: []SeasonYield{{Season: "winter", BP: 10_000}}},
		"summer+autum": {Seasons: []SeasonYield{{Season: "summer", BP: 5_000}, {Season: "autumn", BP: 10_000}}},
		"all":          {HasAll: true, All: 10_000},
	}
	calendars := map[string]func(*Calendar){
		"default":    func(*Calendar) {},
		"late start": func(c *Calendar) { c.StartDay, c.SeasonStart = 200, 10 },
	}
	for pn, p := range profiles {
		for cn, edit := range calendars {
			t.Run(pn+"/"+cn, func(t *testing.T) {
				t.Parallel()
				for d := int64(1); d <= 400; d++ {
					c := viaSeasonal(d, p)
					edit(&c.Calendar)
					c = c.canonical()
					m := newViaModel(&c)
					got := virtualStartOffsets(&c, &m.yt, 0, 0)
					if empty := noVirtualStart(m.runsFor(0, 0), m.year, d); empty != (len(got) == 0) {
						t.Fatalf("noVirtualStart(d=%d, %s, %s) = %v with %d offsets, want them to agree", d, pn, cn, empty, len(got))
					}
					if !slices.IsSorted(got) || (len(got) > 0 && (got[0] < 0 || got[len(got)-1] >= d)) {
						t.Fatalf("offsets (d=%d, %s, %s) = %v, want ascending and in [0, %d)", d, pn, cn, got, d)
					}
				}
			})
		}
	}
}

// TestSeasonRuns verifies that the runs of the year agree with SeasonAt tick by
// tick, add up to the year, merge neighbours of one season, and are one run of
// season -1 for a calendar with no seasons.
//
// Requirement: FR-VIA-053
func TestSeasonRuns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cal  func() Calendar
		runs int
	}{
		// SeasonStart 59: winter days 334 to 364 and 0 to 58 wrap around the year end,
		// so the year is winter (59 days), spring, summer, autumn, winter (31 days).
		{"default", DefaultCalendar, 5},
		{"no seasons", func() Calendar { c := DefaultCalendar(); c.Seasons = nil; return c }, 1},
		{"four ticks a day", func() Calendar {
			c := DefaultCalendar()
			c.TicksPerDay = 4
			return c
		}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cal := tt.cal()
			runs := seasonRuns(cal)
			if len(runs) != tt.runs {
				t.Errorf("seasonRuns(%s) has %d runs %v, want %d", tt.name, len(runs), runs, tt.runs)
			}
			var tick int64
			for _, r := range runs {
				for range r.length {
					if got := cal.SeasonAt(tick); got != r.season {
						t.Fatalf("seasonRuns(%s): tick %d is in run season %d, SeasonAt gives %d", tt.name, tick, r.season, got)
					}
					tick++
				}
			}
			if tick != cal.TicksPerYear() {
				t.Errorf("seasonRuns(%s) covers %d ticks, want the year of %d", tt.name, tick, cal.TicksPerYear())
			}
		})
	}
}

// TestViaModel_Activities verifies the activities the model builds for the
// default calendar: a 60-tick spring job has J = 2 starts at 59 and 119 with a
// factor sum of 20 000 and completions at 118 and 178 (spec-viability.md 3.1,
// 4.5), an all-year recipe is open on every tick, a recipe with no yield does not
// exist, and the households and upkeep of a region are summed over its groups.
//
// Requirement: FR-VIA-001, FR-VIA-002
func TestViaModel_Activities(t *testing.T) {
	t.Parallel()
	c := viaSeasonal(60, viaSpring("", ""))
	c.Recipes = append(c.Recipes, viaRecipe("rest", nil, []Output{viaOut("grain", 1)}, func(r *Recipe) { r.Sited = true }), // sited, no yield: J = 0
		viaRecipe("grind", []Item{{Commodity: "grain", Quantity: 3}}, []Output{viaOut("grain", 4)}, func(r *Recipe) { r.Duration = 2 }))
	c.Roles[0].Recipes = []string{"farm", "grind", "rest"}
	// A second group of the region: 7 households of a role that eats 2 more grain in
	// the region (the region's extra upkeep applies to every group).
	trader := DefaultRole()
	trader.ID = "trader"
	c.Roles = append(c.Roles, trader)
	c.Groups = append(c.Groups, Group{Region: "north", Role: "trader", Households: 7})
	c.Regions[0].ExtraUpkeep = []Item{{Commodity: "grain", Quantity: 1}}
	c.Regions[0].ReferenceBasket = 200
	c = c.canonical()
	m := newViaModel(&c)
	if m.households[0] != 17 || m.upkeep[0][0] != 34 { // 17 households * (1 + 1) grain
		t.Errorf("households, upkeep = %d, %d, want 17 and 34", m.households[0], m.upkeep[0][0])
	}
	if !m.listed[0][0] || !m.listed[0][1] || !slices.Equal(m.withHouseholds, []RegionID{0}) {
		t.Errorf("listed = %v, withHouseholds = %v, want farm and grind listed in region 0", m.listed[0], m.withHouseholds)
	}
	byRecipe := map[string]viaActivity{}
	for _, a := range m.acts[0] {
		byRecipe[c.Recipes[a.recipe].ID] = a
	}
	if _, ok := byRecipe["rest"]; ok || len(byRecipe) != 2 {
		t.Fatalf("activities = %d, want farm and grind and no rest", len(byRecipe))
	}
	farm := byRecipe["farm"]
	if farm.count.Starts != 2 || farm.count.FactorSum != 20_000 || !slices.Equal(farm.count.Completions, []int64{118, 178}) || farm.allYear {
		t.Errorf("farm = %+v, want 2 starts, factor sum 20 000, completions [118 178], not all year", farm.count)
	}
	// The activity handed to the solver carries the counts, the recipe's lists as
	// commodity indexes (grain is 0) and, for a tool, the commodity's durability.
	wantAct := steady.Activity{Duration: 60, Starts: 2, FactorSum: 20_000, Outputs: []steady.Item{{Commodity: 0, Qty: 600}}}
	if !reflect.DeepEqual(farm.act, wantAct) {
		t.Errorf("farm activity = %+v, want %+v", farm.act, wantAct)
	}
	grind := byRecipe["grind"]
	wantGrind := steady.Activity{Duration: 2, Starts: 183, FactorSum: 183 * 10_000,
		Inputs: []steady.Item{{Commodity: 0, Qty: 3}}, Outputs: []steady.Item{{Commodity: 0, Qty: 4}}}
	if !reflect.DeepEqual(grind.act, wantGrind) {
		t.Errorf("grind activity = %+v, want %+v", grind.act, wantGrind)
	}
	if !grind.allYear || grind.count.Starts != 183 { // ceil(365 / 2) starts of a 2-tick job
		t.Errorf("grind = %+v allYear %v, want 183 starts and open all year", grind.count, grind.allYear)
	}
}

// TestValidate_EmptyVirtualStartOffsets verifies the rule of FR-VIA-033 that runs
// at the entry of phase 6: a job row on a recipe that has no virtual start offset
// in the group's region is ErrWarmStartJob, at the path of the job, and a recipe
// that has one is accepted. Two groups of the same pair are both refused.
//
// Requirement: FR-VIA-033, FR-CFG-037
func TestValidate_EmptyVirtualStartOffsets(t *testing.T) {
	t.Parallel()
	job := func(c *Config, recipe string) {
		c.WarmStart = &WarmStart{Groups: []WarmGroup{{Region: "north", Role: "farmer", Jobs: []JobSpec{{Recipe: recipe, InProgressBP: 1_000}}}}}
	}
	tests := []struct {
		name   string
		config func() Config
		bad    bool
	}{
		{"spring job of 60 ticks", func() Config {
			c := viaSeasonal(60, viaSpring("", ""))
			job(&c, "farm")
			return c
		}, true},
		{"spring job of a year", func() Config {
			c := viaSeasonal(365, viaSpring("", ""))
			job(&c, "farm")
			return c
		}, false},
		{"all-year job", func() Config {
			c := viaToy()
			c.Recipes[0].Duration = 12
			c.Recipes[0].Outputs[0].Quantity = 48
			job(&c, "farm")
			return c
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.config()
			err := c.Validate()
			if !tt.bad {
				if err != nil {
					t.Fatalf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			var ve *ValidationError
			if !errors.Is(err, ErrWarmStartJob) || !errors.As(err, &ve) ||
				!slices.Equal(ve.Path, []string{"warm_start", "groups", "north", "farmer", "jobs", "farm"}) {
				t.Errorf("Validate(%s) = %v, want ErrWarmStartJob at warm_start.groups.north.farmer.jobs.farm", tt.name, err)
			}
		})
	}
	t.Run("two groups of one pair", func(t *testing.T) {
		t.Parallel()
		c := viaSeasonal(60, viaSpring("", ""))
		other := DefaultRole()
		other.ID, other.Recipes = "grower", []string{"farm"}
		c.Roles = append(c.Roles, other)
		c.Groups = append(c.Groups, Group{Region: "north", Role: "grower", Households: 4})
		c.WarmStart = &WarmStart{Groups: []WarmGroup{
			{Region: "north", Role: "farmer", Jobs: []JobSpec{{Recipe: "farm", InProgressBP: 100}}},
			{Region: "north", Role: "grower", Jobs: []JobSpec{{Recipe: "farm", InProgressBP: 100}}},
		}}
		err := c.Validate()
		n := 0
		for _, e := range err.(interface{ Unwrap() []error }).Unwrap() {
			if errors.Is(e, ErrWarmStartJob) {
				n++
			}
		}
		if n != 2 {
			t.Errorf("Validate = %d ErrWarmStartJob, want 2 (one for each group): %v", n, err)
		}
	})
}

// TestViabilityFiles_NoFloatingPoint verifies that the root files of the
// viability rules use no floating-point type, literal or function (FR-VIA-015
// extends to them), and that the guard sees them.
//
// Requirement: FR-VIA-015
//
// Guard: the determinism guard does not detect float types; a float in these
// files would make the verdict of Validate depend on the platform.
func TestViabilityFiles_NoFloatingPoint(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("viability*.go")
	if err != nil {
		t.Fatalf("Glob error = %v", err)
	}
	files = slices.DeleteFunc(files, func(f string) bool { return strings.HasSuffix(f, "_test.go") })
	if len(files) < 2 {
		t.Fatalf("Glob(viability*.go) = %v, want at least viability.go and viability_check.go", files)
	}
	for _, name := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				if slices.Contains([]string{"float32", "float64", "complex64", "complex128"}, n.Name) || strings.Contains(n.Name, "Float") {
					t.Errorf("%s uses %s, want no floating point", name, n.Name)
				}
			case *ast.BasicLit:
				if n.Kind == token.FLOAT {
					t.Errorf("%s has the floating literal %s", name, n.Value)
				}
			case *ast.ImportSpec:
				if p := n.Path.Value; p == `"math"` || p == `"math/big"` {
					t.Errorf("%s imports %s, want it in internal/steady only (architecture.md 3.5)", name, p)
				}
			}
			return true
		})
	}
}
