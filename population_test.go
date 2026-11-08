package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the population module's configuration surface: its bounds, its
// region rows, its warnings and its hash (spec-population.md 8, 9, FR-POP-025 to
// 030). The module has no world effect in this file.

// populationConfig returns referenceConfig (one region, north) with the
// population module on at its defaults and no region row.
func populationConfig() economysim.Config {
	c := referenceConfig()
	withPopulation(&c)
	return c
}

// withPopulation switches the module on in c at its defaults.
func withPopulation(c *economysim.Config) {
	m := economysim.DefaultPopulation()
	c.Modules.Population = &m
}

// populationFieldCases are the failing cases of the rows of the population module
// in the parameter-owner table (spec-population.md 8, 9.1, FR-CFG-031): each
// breaks one field of a valid configuration. buildFieldCases appends them to the
// table that the registry test reads.
func populationFieldCases() []fieldCase {
	base := []string{"modules", "population"}
	pop := func(field, name string, rule error, mutate func(m *economysim.Population), path ...string) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{append(slices.Clone(base), path...)},
			mutate: func(c *economysim.Config) { withPopulation(c); mutate(c.Modules.Population) }}
	}
	row := func(field, name string, rule error, mutate func(r *economysim.RegionPopulation), path ...string) fieldCase {
		return pop(field, name, rule, func(m *economysim.Population) {
			r := economysim.DefaultRegionPopulation("north")
			mutate(&r)
			m.Regions = append(m.Regions, r)
		}, append([]string{"region", "north"}, path...)...)
	}
	bounds := economysim.ErrOutOfBounds
	nine := make([]economysim.Knot, economysim.MaxKnots+1)
	for i := range nine {
		nine[i] = economysim.Knot{Ratio: int64(i) * 1_000, Rate: 100}
	}
	return []fieldCase{
		pop("Population.BirthSchedule", "no knots", bounds, func(m *economysim.Population) { m.BirthSchedule = nil }, "birth_schedule"),
		pop("Population.BirthSchedule", "more than MaxKnots", bounds, func(m *economysim.Population) { m.BirthSchedule = nine }, "birth_schedule"),
		pop("Population.DeathSchedule", "no knots", bounds, func(m *economysim.Population) { m.DeathSchedule = nil }, "death_schedule"),
		pop("Population.DeathSchedule", "more than MaxKnots", bounds, func(m *economysim.Population) { m.DeathSchedule = nine }, "death_schedule"),
		pop("Knot.Ratio", "not above the one before", bounds, func(m *economysim.Population) { m.BirthSchedule[1].Ratio = m.BirthSchedule[0].Ratio }, "birth_schedule", "1", "ratio"),
		pop("Knot.Ratio", "below the one before", bounds, func(m *economysim.Population) { m.DeathSchedule[1].Ratio = 1 }, "death_schedule", "1", "ratio"),
		pop("Knot.Ratio", "above MaxBasisPoints", bounds, func(m *economysim.Population) { m.BirthSchedule[1].Ratio = economysim.MaxBasisPoints + 1 }, "birth_schedule", "1", "ratio"),
		pop("Knot.Ratio", "below 0", bounds, func(m *economysim.Population) { m.DeathSchedule[0].Ratio = -1 }, "death_schedule", "0", "ratio"),
		pop("Knot.Rate", "above 10 000", bounds, func(m *economysim.Population) { m.BirthSchedule[0].Rate = 10_001 }, "birth_schedule", "0", "rate"),
		pop("Knot.Rate", "below 0", bounds, func(m *economysim.Population) { m.DeathSchedule[1].Rate = -1 }, "death_schedule", "1", "rate"),
		pop("Population.MinHouseholds", "below 0", bounds, func(m *economysim.Population) { m.MinHouseholds = -1 }, "min_households"),
		pop("Population.MinHouseholds", "above MaxHouseholds", bounds, func(m *economysim.Population) { m.MinHouseholds = economysim.MaxHouseholds + 1 }, "min_households"),
		pop("Population.BirthStake", "above 10 000", bounds, func(m *economysim.Population) { m.BirthStake = 10_001 }, "birth_stake"),
		pop("Population.BirthStake", "below 0", bounds, func(m *economysim.Population) { m.BirthStake = -1 }, "birth_stake"),
		pop("Population.Regions", "more than MaxRegions", bounds, func(m *economysim.Population) {
			m.Regions = make([]economysim.RegionPopulation, economysim.MaxRegions+1)
		}, "regions"),
		pop("RegionPopulation.Region", "a region that is not defined", economysim.ErrUnknownRegion, func(m *economysim.Population) {
			m.Regions = []economysim.RegionPopulation{economysim.DefaultRegionPopulation("nowhere")}
		}, "region", "nowhere", "region"),
		pop("RegionPopulation.Region", "a region with two rows", economysim.ErrDuplicateID, func(m *economysim.Population) {
			m.Regions = []economysim.RegionPopulation{economysim.DefaultRegionPopulation("north"), economysim.DefaultRegionPopulation("north")}
		}, "region", "north", "region"),
		row("RegionPopulation.BirthScale", "above MaxBasisPoints", bounds, func(r *economysim.RegionPopulation) { r.BirthScale = economysim.MaxBasisPoints + 1 }, "birth_scale"),
		row("RegionPopulation.BirthScale", "below 0", bounds, func(r *economysim.RegionPopulation) { r.BirthScale = -1 }, "birth_scale"),
		row("RegionPopulation.DeathScale", "above MaxBasisPoints", bounds, func(r *economysim.RegionPopulation) { r.DeathScale = economysim.MaxBasisPoints + 1 }, "death_scale"),
		row("RegionPopulation.DeathScale", "below 0", bounds, func(r *economysim.RegionPopulation) { r.DeathScale = -1 }, "death_scale"),
		row("RegionPopulation.HouseholdCap", "above MaxHouseholds", bounds, func(r *economysim.RegionPopulation) { r.HouseholdCap = economysim.MaxHouseholds + 1 }, "household_cap"),
		row("RegionPopulation.HouseholdCap", "below 0", bounds, func(r *economysim.RegionPopulation) { r.HouseholdCap = -1 }, "household_cap"),
	}
}

// TestValidate_PopulationBounds verifies that the defaults validate and that each
// case of populationFieldCases breaks exactly one rule: the error matches the
// sentinel and sits at the path of the field, and no other error is reported.
//
// Requirement: FR-POP-025, FR-POP-026
func TestValidate_PopulationBounds(t *testing.T) {
	t.Parallel()
	if err := populationConfig().Validate(); err != nil {
		t.Fatalf("Validate(defaults) = %v, want nil", err)
	}
	for _, tt := range populationFieldCases() {
		t.Run(tt.field+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tt.rules[0]) {
				t.Fatalf("Validate() = %v, want %v", err, tt.rules[0])
			}
			vs, other := violations(err)
			if len(other) != 0 || len(vs) != 1 || !slices.Equal(vs[0].Path, tt.paths[0]) {
				t.Errorf("Validate() = %v, want only %v at %v", err, tt.rules[0], tt.paths[0])
			}
		})
	}
}

// TestValidate_PopulationRegionRows verifies the region-keyed mechanism of
// spec-regions.md 5.3: a row for a defined region validates, a region with no row
// is the default and draws no warning, and rows are accepted in any order.
//
// Requirement: FR-POP-027, FR-REG-035
func TestValidate_PopulationRegionRows(t *testing.T) {
	t.Parallel()
	c := populationConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(no rows) = %v, want nil: absence is the default", err)
	}
	for _, w := range c.Warnings() {
		if len(w.Path) > 2 && w.Path[0] == "modules" && w.Path[1] == "population" && w.Path[2] == "region" {
			t.Errorf("Warnings() has %v for a region with no row, want none", w)
		}
	}
	c.Modules.Population.Regions = []economysim.RegionPopulation{{Region: "north", BirthScale: 20_000, DeathScale: 5_000, HouseholdCap: 600}}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(one row) = %v, want nil", err)
	}
}

// warnRules returns the rules of the warnings of c whose path starts with the
// given segments, sorted as Warnings sorts them.
func warnRules(c economysim.Config, prefix ...string) []string {
	var out []string
	for _, w := range c.Warnings() {
		if len(w.Path) >= len(prefix) && slices.Equal(w.Path[:len(prefix)], prefix) {
			out = append(out, w.Rule)
		}
	}
	return out
}

// TestWarnings_Population verifies each warning and diagnostic of
// spec-population.md 9.2, 5.7 and 4.6.1 on its condition at its path, and that
// none of them fails Validate.
//
// Requirement: FR-POP-020, FR-POP-028, FR-POP-029, FR-POP-030
func TestWarnings_Population(t *testing.T) {
	t.Parallel()
	modPath := []string{"modules", "population"}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		path   []string
		want   bool
	}{
		// r* of the defaults is 12 000 on the segment that starts at knot 0, above the closed ceiling.
		{"replacement ratio is reported", func(c *economysim.Config) {}, "replacement_ratio", []string{"modules", "population", "birth_schedule", "0"}, true},
		{"above the closed ceiling without demand", func(c *economysim.Config) {}, "replacement_above_closed_ceiling", modPath, true},
		{"demand on lifts the ceiling", func(c *economysim.Config) { c.Modules.Demand = &economysim.Demand{} }, "replacement_above_closed_ceiling", modPath, false},
		{"r* under 10 000 is under the ceiling", func(c *economysim.Config) {
			c.Modules.Population.DeathSchedule = []economysim.Knot{{Ratio: 4_000, Rate: 450}, {Ratio: 12_000, Rate: 50}}
		}, "replacement_above_closed_ceiling", modPath, false},
		{"births never below deaths", func(c *economysim.Config) {
			c.Modules.Population.BirthSchedule = []economysim.Knot{{Ratio: 0, Rate: 500}}
		}, "no_replacement_ratio", modPath, true},
		{"deaths above births everywhere", func(c *economysim.Config) {
			c.Modules.Population.DeathSchedule = []economysim.Knot{{Ratio: 0, Rate: 500}}
		}, "no_replacement_ratio", modPath, true},
		{"a crossing is not no_replacement_ratio", func(c *economysim.Config) {}, "no_replacement_ratio", modPath, false},
		{"a falling birth schedule", func(c *economysim.Config) {
			c.Modules.Population.BirthSchedule = []economysim.Knot{{Ratio: 8_000, Rate: 300}, {Ratio: 16_000, Rate: 200}}
		}, "schedule_direction", []string{"modules", "population", "birth_schedule"}, true},
		{"a rising death schedule", func(c *economysim.Config) {
			c.Modules.Population.DeathSchedule = []economysim.Knot{{Ratio: 8_000, Rate: 50}, {Ratio: 16_000, Rate: 450}}
		}, "schedule_direction", []string{"modules", "population", "death_schedule"}, true},
		{"the defaults keep their direction", func(c *economysim.Config) {}, "schedule_direction", []string{"modules", "population", "death_schedule"}, false},
		// A segment of one basis point of ratio: 11 999 * 400 * 30 = 143 988 000 against 10 000 * 1 * 365 = 3 650 000.
		{"a steep crossing fires the gain test", func(c *economysim.Config) {
			c.Modules.Population.DeathSchedule = []economysim.Knot{{Ratio: 11_999, Rate: 450}, {Ratio: 12_000, Rate: 50}}
		}, "population_gain", modPath, true},
		{"the defaults pass the gain test", func(c *economysim.Config) {}, "population_gain", modPath, false},
		{"no draw and no cap", func(c *economysim.Config) {}, "no_fixed_factor", modPath, true},
		{"a household cap is a fixed factor", func(c *economysim.Config) {
			c.Modules.Population.Regions = []economysim.RegionPopulation{{Region: "north", BirthScale: 10_000, DeathScale: 10_000, HouseholdCap: 600}}
		}, "no_fixed_factor", modPath, false},
		{"a recipe that draws is a fixed factor", func(c *economysim.Config) { withResources(c) }, "no_fixed_factor", modPath, false},
		{"no upkeep anywhere", func(c *economysim.Config) {
			c.World.Upkeep = nil
			for i := range c.Roles {
				c.Roles[i].Upkeep = nil
			}
			for i := range c.Regions {
				c.Regions[i].ReferenceBasket = 0
			}
		}, "population_without_upkeep", modPath, true},
		{"an upkeep keeps the module live", func(c *economysim.Config) {}, "population_without_upkeep", modPath, false},
		{"renewal at its default is ignored", func(c *economysim.Config) { c.World.Renewal = 10_000 }, "renewal_ignored", []string{"world", "renewal"}, true},
		{"renewal at its default is not a conflict", func(c *economysim.Config) { c.World.Renewal = 10_000 }, "renewal_with_population", []string{"world", "renewal"}, false},
		{"renewal off the default conflicts", func(c *economysim.Config) { c.World.Renewal = 5_000 }, "renewal_with_population", []string{"world", "renewal"}, true},
		{"renewal off the default is not ignored", func(c *economysim.Config) { c.World.Renewal = 5_000 }, "renewal_ignored", []string{"world", "renewal"}, false},
		{"renewal 0 is silent about conflict", func(c *economysim.Config) { c.World.Renewal = 0 }, "renewal_with_population", []string{"world", "renewal"}, false},
		{"renewal 0 is silent about ignoring", func(c *economysim.Config) { c.World.Renewal = 0 }, "renewal_ignored", []string{"world", "renewal"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := populationConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil: a warning never fails validation", err)
			}
			if got := hasWarning(c.Warnings(), tt.rule, tt.path...); got != tt.want {
				t.Errorf("Warnings() has %s at %v = %v, want %v", tt.rule, tt.path, got, tt.want)
			}
		})
	}
	off := referenceConfig()
	if got := warnRules(off, "modules", "population"); len(got) != 0 {
		t.Errorf("Warnings() with the module off has %v under modules.population, want none", got)
	}
	if hasWarning(off.Warnings(), "renewal_ignored", "world", "renewal") {
		t.Errorf("Warnings() with the module off has renewal_ignored, want none")
	}
}

// TestWarnings_PopulationUnreadable verifies that the warnings read a
// configuration that does not validate without a panic and without a figure: a
// schedule with no knot, nine knots or ratios that do not rise reports nothing
// about r*.
//
// Guard: Warnings is safe to call on an invalid configuration (spec-config.md
// 7.2); a schedule the rules cannot read must not reach Replacement.
func TestWarnings_PopulationUnreadable(t *testing.T) {
	t.Parallel()
	bad := [][]economysim.Knot{nil, {{Ratio: 5, Rate: 5}, {Ratio: 5, Rate: 6}}, {{Ratio: -1, Rate: 5}}, {{Ratio: 1, Rate: 20_000}}, make([]economysim.Knot, economysim.MaxKnots+1)}
	for i, ks := range bad {
		c := populationConfig()
		c.Modules.Population.BirthSchedule = ks
		for _, w := range c.Warnings() {
			if w.Rule == "replacement_ratio" || w.Rule == "no_replacement_ratio" || w.Rule == "population_gain" {
				t.Errorf("case %d: Warnings() has %v for an unreadable schedule, want no figure", i, w)
			}
		}
	}
}

// TestWarnings_ResourcePlan verifies that with the resources module and the
// population module on, the schedules fill endowment.Plan and
// resource_overshoot and resource_unstable_plan fire on the inequalities of
// spec-resources.md 8.3.1, and that they are silent with the module off.
//
// The defaults give r* 12 000, dR 400, dX 8 000. The arable refill at the plan
// share 6 000 has eps_bp 1 304 on its curve (6 000 * 1 000 * 10 000 / (5 000 *
// 9 200) = 1 304.3), so the overshoot condition is RateBP_eff * 4.8e11 <
// 4 * 12 000 * 400 * 1 304 * 4 000 = 1.0014720e14, true up to a rate of 208.
//
// Requirement: FR-RES-044
func TestWarnings_ResourcePlan(t *testing.T) {
	t.Parallel()
	arable := []string{"modules", "resources", "resources", "arable"}
	stand := []string{"modules", "resources", "resources", "stand"}
	with := func(mutate func(c *economysim.Config)) economysim.Config {
		c := resourcesConfig()
		withPopulation(&c)
		arable := resourceOf(&c, "arable")
		arable.Seasons = nil // a profile would scale the rate by the mean season factor
		mutate(&c)
		return c
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		path   []string
		want   bool
	}{
		{"refill at 208 overshoots", func(c *economysim.Config) { resourceOf(c, "arable").RateBP = 208 }, "resource_overshoot", arable, true},
		{"refill at 209 does not", func(c *economysim.Config) { resourceOf(c, "arable").RateBP = 209 }, "resource_overshoot", arable, false},
		{"a fast refill does not", func(c *economysim.Config) {}, "resource_overshoot", arable, false},
		{"a refill is never unstable", func(c *economysim.Config) {}, "resource_unstable_plan", arable, false},
		{"logistic above half on a flat curve", func(c *economysim.Config) {}, "resource_overshoot", stand, false},
		{"a constant flow with a curve is unstable", func(c *economysim.Config) {
			r := resourceOf(c, "arable")
			r.Shape, r.RateBP, r.RatePerYear = economysim.ShapeConstant, 0, 5
		}, "resource_unstable_plan", arable, true},
		{"logistic at half with a curve is unstable", func(c *economysim.Config) {
			r := resourceOf(c, "arable")
			r.Shape, r.RateBP, r.Refuge, r.PlanShareBP = economysim.ShapeLogistic, 3_000, 20, 5_000
		}, "resource_unstable_plan", arable, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := with(tt.mutate)
			if got := hasWarning(c.Warnings(), tt.rule, tt.path...); got != tt.want {
				t.Errorf("Warnings() has %s at %v = %v, want %v", tt.rule, tt.path, got, tt.want)
			}
		})
	}
	t.Run("silent with the population module off", func(t *testing.T) {
		t.Parallel()
		c := with(func(c *economysim.Config) {
			r := resourceOf(c, "arable")
			r.Shape, r.RateBP, r.RatePerYear = economysim.ShapeConstant, 0, 5
			resourceOf(c, "arable").RateBP = 0
		})
		c.Modules.Population = nil
		for _, rule := range []string{"resource_overshoot", "resource_unstable_plan"} {
			if hasWarning(c.Warnings(), rule, arable...) {
				t.Errorf("Warnings() has %s with the population module off, want none", rule)
			}
		}
	})
}

// TestWarnings_ResourcePlanSeasons verifies the effective rate of the plan: the
// rate times the mean season factor over the calendar's seasons, so a profile
// that gives a season 20 000 and the others nothing moves the threshold of the
// overshoot.
//
// Requirement: FR-RES-044
func TestWarnings_ResourcePlanSeasons(t *testing.T) {
	t.Parallel()
	arable := []string{"modules", "resources", "resources", "arable"}
	seasons := int64(len(referenceConfig().Calendar.Seasons))
	if seasons == 0 {
		t.Skip("the reference calendar has no season, so no profile applies")
	}
	mean := 20_000 / seasons
	fires := func(rate int64) bool {
		c := resourcesConfig()
		withPopulation(&c)
		resourceOf(&c, "arable").RateBP = rate
		return hasWarning(c.Warnings(), "resource_overshoot", arable...)
	}
	// Smallest rate at which the effective rate reaches 209 and the overshoot ends.
	first := int64(1)
	for fires(first) {
		first++
	}
	if got := first * mean / 10_000; got < 209 || (first-1)*mean/10_000 > 208 {
		t.Errorf("overshoot ends at RateBP %d, effective %d, want the first rate whose effective rate is 209", first, got)
	}
}

// TestHashConfig_PopulationId3 verifies that the module is hashed as id 3 of
// Modules only when it is on: off leaves the pinned reference hash, on moves it,
// the region rows hash alike in any order, and the order of the knots matters.
//
// Requirement: FR-CFG-069, FR-CFG-105
func TestHashConfig_PopulationId3(t *testing.T) {
	t.Parallel()
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Fatalf("hash of the all-off reference = %s, want the pin %s", got, pinnedReferenceHash)
	}
	on := populationConfig()
	if hexHash(on) == pinnedReferenceHash {
		t.Errorf("hash with the module on = the all-off hash, want a different one")
	}
	empty := referenceConfig()
	empty.Modules.Population = &economysim.Population{}
	if hexHash(empty) == hexHash(on) || hexHash(empty) == pinnedReferenceHash {
		t.Errorf("hash of an empty module on = %s, want one that differs from off and from the defaults", hexHash(empty))
	}
	rows := []economysim.RegionPopulation{economysim.DefaultRegionPopulation("a"), economysim.DefaultRegionPopulation("b")}
	rows[1].DeathScale = 20_000
	one, two := populationConfig(), populationConfig()
	one.Modules.Population.Regions = rows
	two.Modules.Population.Regions = []economysim.RegionPopulation{rows[1], rows[0]}
	if hexHash(one) != hexHash(two) {
		t.Errorf("hash with rows in two orders differs, want equal: rows hash by region")
	}
	swapped := populationConfig()
	swapped.Modules.Population.DeathSchedule = []economysim.Knot{{Ratio: 8_000, Rate: 50}, {Ratio: 16_000, Rate: 450}}
	if hexHash(swapped) == hexHash(on) {
		t.Errorf("hash with the death rates swapped = the defaults', want different: knot order is the schedule")
	}
	demandOnly := referenceConfig()
	demandOnly.Modules.Demand = &economysim.Demand{}
	withBoth := demandOnly
	withPopulation(&withBoth)
	if hexHash(demandOnly) == hexHash(withBoth) {
		t.Errorf("hash with demand alone = hash with demand and population, want different")
	}
}

// FuzzValidatePopulation verifies that Validate and Warnings never panic on a
// population module built from the fuzzed numbers, and that Validate accepts it
// exactly when every number is within its bound and the ratios rise.
//
// Requirement: FR-POP-025, FR-POP-026
func FuzzValidatePopulation(f *testing.F) {
	f.Add(int64(8_000), int64(250), int64(16_000), int64(250), int64(1), int64(10_000), int64(10_000))
	f.Add(int64(-1), int64(20_000), int64(-1), int64(-5), int64(-1), int64(-1), int64(1<<62))
	f.Fuzz(func(t *testing.T, r0, a0, r1, a1, minHouseholds, stake, scale int64) {
		c := referenceConfig()
		m := economysim.DefaultPopulation()
		m.BirthSchedule = []economysim.Knot{{Ratio: r0, Rate: a0}, {Ratio: r1, Rate: a1}}
		m.DeathSchedule = slices.Clone(m.BirthSchedule)
		m.MinHouseholds, m.BirthStake = minHouseholds, stake
		m.Regions = []economysim.RegionPopulation{{Region: "north", BirthScale: scale, DeathScale: scale}}
		c.Modules.Population = &m
		err := c.Validate()
		_ = c.Warnings()
		valid := r0 >= 0 && r0 < r1 && r1 <= economysim.MaxBasisPoints &&
			a0 >= 0 && a0 <= 10_000 && a1 >= 0 && a1 <= 10_000 &&
			minHouseholds >= 0 && minHouseholds <= economysim.MaxHouseholds &&
			stake >= 0 && stake <= 10_000 && scale >= 0 && scale <= economysim.MaxBasisPoints
		if valid != (err == nil) {
			t.Errorf("Validate() = %v for %+v, want an error exactly when a number is out of its bound", err, m)
		}
	})
}
