package economysim_test

import (
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestConfig_PlainStructs verifies that every type reachable from Config is a
// plain struct of integers, bools, strings, slices and pointers to structs,
// with no struct tag and no embedded field, so that a world can be built from Go
// alone and a transport can never be implied by a tag.
//
// Requirement: FR-CFG-001
func TestConfig_PlainStructs(t *testing.T) {
	t.Parallel()
	seen := map[reflect.Type]bool{}
	var visit func(reflect.Type)
	visit = func(typ reflect.Type) {
		if seen[typ] {
			return
		}
		seen[typ] = true
		for i := range typ.NumField() {
			f := typ.Field(i)
			if f.Tag != "" {
				t.Errorf("%s.%s has the struct tag %q, want none", typ.Name(), f.Name, f.Tag)
			}
			if f.Anonymous {
				t.Errorf("%s embeds %s, want a named field", typ.Name(), f.Type)
			}
			if !f.IsExported() {
				t.Errorf("%s.%s is unexported, want plain data", typ.Name(), f.Name)
			}
			ft := f.Type
			if ft.Kind() == reflect.Slice || ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			switch ft.Kind() {
			case reflect.Int64, reflect.Bool, reflect.String:
			case reflect.Struct:
				visit(ft)
			default:
				t.Errorf("%s.%s has the type %s, want an integer, bool, string or struct", typ.Name(), f.Name, f.Type)
			}
		}
	}
	visit(reflect.TypeFor[economysim.Config]())
	if len(seen) < 20 {
		t.Errorf("visited %d struct types, want the whole configuration (at least 20)", len(seen))
	}
}

// TestDefaults_EarthCalendar verifies that DefaultCalendar is the Earth
// calendar of spec-calendar.md 3.1: twelve months and four seasons that each
// sum to 365 days, one tick a day.
//
// Requirement: FR-CFG-007
func TestDefaults_EarthCalendar(t *testing.T) {
	t.Parallel()
	c := economysim.DefaultCalendar()
	var months, seasons int64
	for _, m := range c.Months {
		months += m.Days
	}
	for _, s := range c.Seasons {
		seasons += s.Days
	}
	// 31+28+31+30+31+30+31+31+30+31+30+31 = 365, and 92+92+91+90 = 365.
	if len(c.Months) != 12 || months != 365 {
		t.Errorf("DefaultCalendar().Months = %d months of %d days, want 12 months of 365 days", len(c.Months), months)
	}
	if len(c.Seasons) != 4 || seasons != 365 {
		t.Errorf("DefaultCalendar().Seasons = %d seasons of %d days, want 4 seasons of 365 days", len(c.Seasons), seasons)
	}
	if c.HoursPerDay != 24 || c.DaysPerWeek != 7 || c.TicksPerDay != 1 || c.DaysPerTick != 1 || c.StartDay != 0 || c.SeasonStart != 59 {
		t.Errorf("DefaultCalendar() = %+v, want 24 hours, 7-day week, one tick a day, start day 0, season start 59", c)
	}
}

// TestConfig_HoldsNoWageOrPremiumSetting verifies that no type reachable from
// Config has a field named for a wage multiplier or a skill premium, and that
// the only field named for a wage is the warm start's starting wage, a seed
// that the first observed working household-tick replaces (WEL 4.3), so that a
// wage and a premium are measured and never set.
//
// Guard: FR-WEL-015 is a negative requirement. A convenient SkillPremium or
// WageBP field added later would pass every other test and quietly turn the
// measured premium into a setting.
//
// Requirement: FR-WEL-015
func TestConfig_HoldsNoWageOrPremiumSetting(t *testing.T) {
	t.Parallel()
	seen := map[reflect.Type]bool{}
	var visit func(reflect.Type)
	visit = func(typ reflect.Type) {
		if seen[typ] {
			return
		}
		seen[typ] = true
		for i := range typ.NumField() {
			f := typ.Field(i)
			name := strings.ToLower(f.Name)
			if strings.Contains(name, "premium") || strings.Contains(name, "multiplier") {
				t.Errorf("%s.%s names a premium or a multiplier, want none (FR-WEL-015)", typ.Name(), f.Name)
			}
			seedWage := (typ.Name() == "WarmStart" && f.Name == "Wages") || (typ.Name() == "SeedWage" && f.Name == "Wage")
			if strings.Contains(name, "wage") && !seedWage {
				t.Errorf("%s.%s names a wage, want only the warm start's starting wage (FR-WEL-015)", typ.Name(), f.Name)
			}
			ft := f.Type
			if ft.Kind() == reflect.Slice || ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				visit(ft)
			}
		}
	}
	visit(reflect.TypeOf(economysim.Config{}))
}

// TestDefaults_Values verifies the defaults the owner specs state, one row per
// constructor, so that a change to a default is a visible, explained change
// (architecture.md 9.8).
//
// Requirement: FR-CFG-007, FR-CFG-049, FR-CFG-050, FR-CFG-057, FR-WEL-016, FR-WEL-032
func TestDefaults_Values(t *testing.T) {
	t.Parallel()
	w := economysim.DefaultWorld()
	r := economysim.DefaultRole()
	tests := []struct {
		name      string
		got, want any
	}{
		{"world scale", w.Scale, int64(1)},
		{"world welfare window", w.WelfareWindow, int64(30)},
		{"world window cycles", w.WindowCycles, int64(2)},
		{"world basket tolerance", w.BasketTolerance, int64(1_000)},
		{"world belief evidence share", w.BeliefEvidenceShare, int64(5_000)},
		{"world cost floor", w.CostFloor, int64(7_000)},
		{"world spawner", w.Spawner, "default"},
		{"world clearing", w.Clearing, "average"},
		{"world upkeep", len(w.Upkeep), 0},
		{"commodity mass", economysim.DefaultCommodity().Mass, int64(1)},
		{"commodity bulk", economysim.DefaultCommodity().Bulk, int64(1)},
		{"commodity reference price is required", economysim.DefaultCommodity().ReferencePrice, economysim.Price(0)},
		{"recipe duration", economysim.DefaultRecipe().Duration, int64(1)},
		{"recipe trainee yield", economysim.DefaultRecipe().TraineeYield, int64(5_000)},
		{"role start money", r.StartMoney, economysim.Money(10_000)},
		{"role strategy", r.Strategy, "default"},
		{"role target ratio", r.TargetRatio, int64(15_000)},
		{"role expected work ticks", r.ExpectedWorkTicks, int64(250)},
		{"role min work share", r.MinWorkShare, int64(4_000)},
		{"region reference basket", economysim.DefaultRegion().ReferenceBasket, economysim.Money(1_000)},
		{"region capital return", economysim.DefaultRegion().CapitalReturn, int64(1_000)},
		{"region estate markdown", economysim.DefaultRegion().EstateMarkdown, int64(500)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestDefaults_ReturnFreshValues verifies that the constructors share nothing
// between calls: changing a slice of one result leaves the next result alone.
//
// Guard: a default held in a package variable and returned by reference would be
// package-level mutable state (architecture.md 7.1); one pack that edited its
// calendar in place would change the Earth default of every later world.
func TestDefaults_ReturnFreshValues(t *testing.T) {
	t.Parallel()
	first := economysim.DefaultCalendar()
	first.Months[0].Days = 99
	first.Seasons[0].ID = "changed"
	second := economysim.DefaultCalendar()
	if second.Months[0].Days != 31 || second.Seasons[0].ID != "spring" {
		t.Errorf("DefaultCalendar() after a change to an earlier result = %+v, want the Earth calendar", second)
	}
}
