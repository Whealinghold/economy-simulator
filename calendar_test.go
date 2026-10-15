package economysim_test

import (
	"errors"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// planetCalendar is the 320-day planet of spec-calendar.md 7.4: a 20-hour day,
// a 5-day week, two ticks a day, eight months of 40 days and two seasons of 120
// and 200 days that start on day 0. TicksPerYear is 320 * 2 / 1 = 640.
func planetCalendar() economysim.Calendar {
	c := economysim.DefaultCalendar()
	c.HoursPerDay, c.DaysPerWeek, c.TicksPerDay = 20, 5, 2
	c.Months = nil
	for i := range 8 {
		c.Months = append(c.Months, economysim.Month{ID: "m" + strconv.Itoa(i+1), Days: 40})
	}
	c.Seasons = []economysim.Season{{ID: "wet", Days: 120}, {ID: "dry", Days: 200}}
	c.SeasonStart = 0
	return c
}

// calWith returns the Earth default changed by f.
func calWith(f func(*economysim.Calendar)) economysim.Calendar {
	c := economysim.DefaultCalendar()
	f(&c)
	return c
}

// TestCalendar_AtMoments verifies the moment of a tick for the Earth default
// and for calendars with other ticks, a wrapping season cycle and no seasons.
// Each expectation carries its arithmetic.
//
// Requirement: FR-CAL-004, FR-CAL-005, FR-CAL-007, FR-CAL-008, FR-CAL-009
func TestCalendar_AtMoments(t *testing.T) {
	t.Parallel()
	earth := economysim.DefaultCalendar()
	// Earth: months end at day 31, 59, 90, 120, 151, 181, 212, 243, 273, 304,
	// 334, 365; the seasons start on day 59 and have 92, 92, 91, 90 days, so
	// season offset = (doy - 59) mod 365 and winter is offsets 275 to 364.
	tests := []struct {
		name string
		cal  economysim.Calendar
		tick int64
		want economysim.Moment
	}{
		// tick 0: doy 0, offset (0-59+365) = 306, winter day 306-275 = 31.
		{"earth tick 0 is in a winter that wraps", earth, 0, economysim.Moment{Tick: 0, Season: 3, DayOfSeason: 31}},
		// tick 58: doy 58 is day 27 of Feb (31+27), offset 364, winter day 89,
		// weekday 58 mod 7 = 2.
		{"earth tick 58 is the last winter day", earth, 58, economysim.Moment{Tick: 58, DayOfYear: 58, Month: 1, DayOfMonth: 27, DayOfWeek: 2, Season: 3, DayOfSeason: 89}},
		// tick 59: first day of March (doy 59 = 31+28), offset 0, spring, weekday 59 mod 7 = 3.
		{"earth tick 59 is the first spring day", earth, 59, economysim.Moment{Tick: 59, DayOfYear: 59, Month: 2, DayOfMonth: 0, DayOfWeek: 3, Season: 0, DayOfSeason: 0}},
		// tick 364: doy 364 is day 30 of Dec (334+30), weekday 364 mod 7 = 0
		// (52 weeks), offset 305, winter day 305-275 = 30.
		{"earth tick 364 is the last day of the year", earth, 364, economysim.Moment{Tick: 364, DayOfYear: 364, Month: 11, DayOfMonth: 30, DayOfWeek: 0, Season: 3, DayOfSeason: 30}},
		// tick 365: year 1, doy 0, weekday 365 mod 7 = 1 (the week does not
		// restart), winter day 31 as at tick 0.
		{"earth tick 365 starts year 1 without restarting the week", earth, 365, economysim.Moment{Tick: 365, Year: 1, DayOfWeek: 1, Season: 3, DayOfSeason: 31}},
		// tick 1 is absolute day 0, tick of day 1, hour 1 * 20 / 2 = 10.
		{"planet tick 1 is the second tick of day 0", planetCalendar(), 1, economysim.Moment{Tick: 1, TickOfDay: 1, Hour: 10, Season: 0}},
		// tick 639: day 319, tick of day 1, hour 10, month 319/40 = 7 day 39,
		// weekday 319 mod 5 = 4, offset 319, dry day 319-120 = 199.
		{"planet last tick of the year", planetCalendar(), 639, economysim.Moment{Tick: 639, DayOfYear: 319, Month: 7, DayOfMonth: 39, DayOfWeek: 4, TickOfDay: 1, Hour: 10, Season: 1, DayOfSeason: 199}},
		// tick 640 is day 320: year 1, doy 0, weekday 320 mod 5 = 0.
		{"planet tick 640 is the first tick of year 1", planetCalendar(), 640, economysim.Moment{Tick: 640, Year: 1, Season: 0}},
		// tick 240: day 120, month 3 day 0, weekday 0, offset 120, dry day 0.
		{"planet tick 240 starts the dry season", planetCalendar(), 240, economysim.Moment{Tick: 240, DayOfYear: 120, Month: 3, DayOfWeek: 0, Season: 1, DayOfSeason: 0}},
		// 5-day tick: tick 1 is day 5, month 0 day 5, weekday 5, offset
		// 5-59+365 = 311, winter day 311-275 = 36.
		{"five-day tick 1", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 5 }), 1, economysim.Moment{Tick: 1, DayOfYear: 5, DayOfMonth: 5, DayOfWeek: 5, Season: 3, DayOfSeason: 36}},
		// tick 72: day 360, Dec day 26 (334+26), weekday 360 mod 7 = 3, offset
		// 301, winter day 26.
		{"five-day tick 72", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 5 }), 72, economysim.Moment{Tick: 72, DayOfYear: 360, Month: 11, DayOfMonth: 26, DayOfWeek: 3, Season: 3, DayOfSeason: 26}},
		// tick 73: day 365, year 1, doy 0, weekday 1.
		{"five-day tick 73 starts year 1", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 5 }), 73, economysim.Moment{Tick: 73, Year: 1, DayOfWeek: 1, Season: 3, DayOfSeason: 31}},
		// 73-day tick: tick 4 is day 292, Oct (273) day 19, weekday 292 mod 7 =
		// 5, offset 233, autumn (184) day 49.
		{"73-day tick 4", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 73 }), 4, economysim.Moment{Tick: 4, DayOfYear: 292, Month: 9, DayOfMonth: 19, DayOfWeek: 5, Season: 2, DayOfSeason: 49}},
		// no seasons: season -1 and day of season 0; doy 100 is Apr (index 3) day
		// 100-90 = 10, weekday 100 mod 7 = 2.
		{"no seasons", calWith(func(c *economysim.Calendar) { c.Seasons = nil }), 100, economysim.Moment{Tick: 100, DayOfYear: 100, Month: 3, DayOfMonth: 10, DayOfWeek: 2, Season: -1}},
		// StartDay 364: tick 0 is the last day of the year (Dec day 30,
		// weekday 364 mod 7 = 0); tick 1 is day 365, year 1, weekday 1.
		{"start day at the last day of the year", calWith(func(c *economysim.Calendar) { c.StartDay = 364 }), 0, economysim.Moment{Tick: 0, DayOfYear: 364, Month: 11, DayOfMonth: 30, DayOfWeek: 0, Season: 3, DayOfSeason: 30}},
		{"start day rolls into year 1", calWith(func(c *economysim.Calendar) { c.StartDay = 364 }), 1, economysim.Moment{Tick: 1, Year: 1, DayOfWeek: 1, Season: 3, DayOfSeason: 31}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.cal.At(tt.tick)
			if err != nil {
				t.Fatalf("At(%d) error = %v, want nil", tt.tick, err)
			}
			if got != tt.want {
				t.Errorf("At(%d) = %+v, want %+v", tt.tick, got, tt.want)
			}
			if again, _ := tt.cal.At(tt.tick); again != got {
				t.Errorf("At(%d) twice = %+v then %+v, want equal", tt.tick, got, again)
			}
		})
	}
}

// TestCalendar_TicksPerYearAndDefault verifies the derived year of the default
// and of calendars with another tick, and that the default has the values of
// FR-CAL-001.
//
// Requirement: FR-CAL-001, FR-CAL-002, FR-CAL-003
func TestCalendar_TicksPerYearAndDefault(t *testing.T) {
	t.Parallel()
	d := economysim.DefaultCalendar()
	if got := d.TicksPerYear(); got != 365 {
		t.Errorf("DefaultCalendar().TicksPerYear() = %d, want 365", got)
	}
	var days []int64
	for _, m := range d.Months {
		days = append(days, m.Days)
	}
	if want := []int64{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}; !reflect.DeepEqual(days, want) {
		t.Errorf("DefaultCalendar() month days = %v, want %v", days, want)
	}
	var seasons []int64
	for _, s := range d.Seasons {
		seasons = append(seasons, s.Days)
	}
	if want := []int64{92, 92, 91, 90}; !reflect.DeepEqual(seasons, want) {
		t.Errorf("DefaultCalendar() season days = %v, want %v", seasons, want)
	}
	if d.HoursPerDay != 24 || d.DaysPerWeek != 7 || d.TicksPerDay != 1 || d.DaysPerTick != 1 || d.StartDay != 0 || d.SeasonStart != 59 {
		t.Errorf("DefaultCalendar() = %+v, want 24 hours, 7 days a week, one tick a day, start day 0, season start 59", d)
	}
	tests := []struct {
		name string
		cal  economysim.Calendar
		want int64
	}{
		{"planet: 320 * 2 / 1", planetCalendar(), 640},
		{"five-day tick: 365 / 5", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 5 }), 73},
		{"73-day tick: 365 / 73", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 73 }), 5},
	}
	for _, tt := range tests {
		if got := tt.cal.TicksPerYear(); got != tt.want {
			t.Errorf("%s: TicksPerYear() = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// TestCalendar_AtRefusals verifies the refusals of At for a tick: a negative
// tick is ErrOutOfBounds and a day number that does not fit is ErrOverflow,
// each with no moment.
//
// Requirement: FR-CAL-010
func TestCalendar_AtRefusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cal  economysim.Calendar
		tick int64
		want error
	}{
		{"negative tick", economysim.DefaultCalendar(), -1, economysim.ErrOutOfBounds},
		{"smallest tick", economysim.DefaultCalendar(), math.MinInt64, economysim.ErrOutOfBounds},
		// MaxInt64 * 5 does not fit in int64.
		{"product overflows", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 5 }), math.MaxInt64, economysim.ErrOverflow},
		// StartDay 364 + MaxInt64 does not fit; the product MaxInt64 * 1 does.
		{"start day add overflows", calWith(func(c *economysim.Calendar) { c.StartDay = 364 }), math.MaxInt64, economysim.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := tt.cal.At(tt.tick)
			if !errors.Is(err, tt.want) {
				t.Errorf("At(%d) error = %v, want %v", tt.tick, err, tt.want)
			}
			if m != (economysim.Moment{}) {
				t.Errorf("At(%d) = %+v with an error, want no moment", tt.tick, m)
			}
			if err != nil && !strings.Contains(err.Error(), strconv.FormatInt(tt.tick, 10)) {
				t.Errorf("At(%d) error = %q, want it to carry the tick", tt.tick, err)
			}
		})
	}
}

// invalidCalendars is the set of calendars that break one numeric rule each.
func invalidCalendars() []struct {
	name string
	cal  economysim.Calendar
	want error
} {
	many := economysim.DefaultCalendar()
	many.Months = nil
	for range economysim.MaxMonths + 1 {
		many.Months = append(many.Months, economysim.Month{ID: "m", Days: 1})
	}
	manySeasons := economysim.DefaultCalendar()
	manySeasons.Seasons = nil
	for range economysim.MaxSeasons + 1 {
		manySeasons.Seasons = append(manySeasons.Seasons, economysim.Season{ID: "s", Days: 1})
	}
	return []struct {
		name string
		cal  economysim.Calendar
		want error
	}{
		{"zero value", economysim.Calendar{}, economysim.ErrOutOfBounds},
		{"no months", calWith(func(c *economysim.Calendar) { c.Months, c.Seasons = nil, nil }), economysim.ErrOutOfBounds},
		{"days per tick 0", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 0 }), economysim.ErrOutOfBounds},
		{"ticks per day 0", calWith(func(c *economysim.Calendar) { c.TicksPerDay = 0 }), economysim.ErrOutOfBounds},
		{"hours per day over bound", calWith(func(c *economysim.Calendar) { c.HoursPerDay = economysim.MaxHoursPerDay + 1 }), economysim.ErrOutOfBounds},
		{"days per week over bound", calWith(func(c *economysim.Calendar) { c.DaysPerWeek = economysim.MaxDaysPerWeek + 1 }), economysim.ErrOutOfBounds},
		{"ticks per day over bound", calWith(func(c *economysim.Calendar) { c.TicksPerDay = economysim.MaxTicksPerDay + 1 }), economysim.ErrOutOfBounds},
		// 11 months of 10 000 days = 110 000 days; DaysPerTick 10 makes the year
		// 11 000 ticks, so only the days rule can refuse it.
		{"days per year over bound", func() economysim.Calendar {
			c := flatCalendar(11, economysim.MaxDaysPerMonth, 1)
			c.DaysPerTick = 10
			return c
		}(), economysim.ErrOutOfBounds},
		{"start day past the year end", calWith(func(c *economysim.Calendar) { c.StartDay = 366 }), economysim.ErrOutOfBounds},
		{"season start past the year end", calWith(func(c *economysim.Calendar) { c.SeasonStart = 366 }), economysim.ErrOutOfBounds},
		// One season of twice the year's cap, over a 100 000-day year: a cap
		// that is not one past the year would make it add up.
		{"season longer than any year", func() economysim.Calendar {
			c := flatCalendar(10, economysim.MaxDaysPerMonth, 1)
			c.Seasons = []economysim.Season{{ID: "s", Days: 2 * economysim.MaxDaysPerYear}}
			return c
		}(), economysim.ErrCalendarSeasons},
		{"days per week 0", calWith(func(c *economysim.Calendar) { c.DaysPerWeek = 0 }), economysim.ErrOutOfBounds},
		{"month of 0 days", calWith(func(c *economysim.Calendar) { c.Months[0].Days = 0 }), economysim.ErrOutOfBounds},
		{"start day outside the year", calWith(func(c *economysim.Calendar) { c.StartDay = 365 }), economysim.ErrOutOfBounds},
		{"negative start day", calWith(func(c *economysim.Calendar) { c.StartDay = -1 }), economysim.ErrOutOfBounds},
		{"season start outside the year", calWith(func(c *economysim.Calendar) { c.SeasonStart = 365 }), economysim.ErrOutOfBounds},
		{"year of ticks over bound", calWith(func(c *economysim.Calendar) {
			c.Months = []economysim.Month{{ID: "a", Days: economysim.MaxDaysPerMonth}}
			c.TicksPerDay, c.StartDay, c.Seasons = economysim.MaxTicksPerDay, 0, nil
		}), economysim.ErrOutOfBounds},
		{"both ticks above 1", calWith(func(c *economysim.Calendar) { c.TicksPerDay, c.DaysPerTick = 2, 2 }), economysim.ErrCalendarTick},
		{"year not a whole number of ticks", calWith(func(c *economysim.Calendar) { c.DaysPerTick = 7 }), economysim.ErrCalendarTick},
		{"seasons do not add up", calWith(func(c *economysim.Calendar) { c.Seasons[0].Days = 91 }), economysim.ErrCalendarSeasons},
		{"season of 0 days", calWith(func(c *economysim.Calendar) { c.Seasons[0].Days, c.Seasons[1].Days = 0, 184 }), economysim.ErrCalendarSeasons},
		{"101 months", many, economysim.ErrTooMany},
		{"101 seasons", manySeasons, economysim.ErrTooMany},
	}
}

// TestCalendar_InvalidCalendarNeverPanics verifies that a hand-built calendar
// that breaks a numeric rule gets the error of the rule from At, 0 from
// TicksPerYear and -1 from SeasonAt, and never panics or divides by zero.
//
// Requirement: FR-CAL-011, FR-CAL-012, FR-CAL-013, FR-CAL-022
func TestCalendar_InvalidCalendarNeverPanics(t *testing.T) {
	t.Parallel()
	for _, tt := range invalidCalendars() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := tt.cal.At(0)
			if !errors.Is(err, tt.want) || m != (economysim.Moment{}) {
				t.Errorf("At(0) = %+v, %v, want no moment and %v", m, err, tt.want)
			}
			if got := tt.cal.TicksPerYear(); got != 0 {
				t.Errorf("TicksPerYear() = %d, want 0", got)
			}
			for _, tick := range []int64{0, 1, -1, math.MaxInt64, math.MinInt64} {
				if got := tt.cal.SeasonAt(tick); got != -1 {
					t.Errorf("SeasonAt(%d) = %d, want -1", tick, got)
				}
			}
		})
	}
}

// calendarSet is a spread of valid calendars for the periodicity tests.
func calendarSet() []economysim.Calendar {
	return []economysim.Calendar{
		economysim.DefaultCalendar(),
		planetCalendar(),
		calWith(func(c *economysim.Calendar) { c.DaysPerTick = 5 }),
		calWith(func(c *economysim.Calendar) { c.DaysPerTick = 73 }),
		calWith(func(c *economysim.Calendar) { c.StartDay = 364 }),
		calWith(func(c *economysim.Calendar) { c.Seasons = nil }),
		calWith(func(c *economysim.Calendar) { c.SeasonStart = 200; c.StartDay = 17 }),
		calWith(func(c *economysim.Calendar) { c.TicksPerDay, c.HoursPerDay = 3, 24 }),
	}
}

// TestCalendar_PeriodicInTicksPerYear verifies that TicksPerYear ticks later
// the year advances by 1 and every year-level field returns, while the weekday
// is not required to.
//
// Requirement: FR-CAL-006, FR-CAL-007
func TestCalendar_PeriodicInTicksPerYear(t *testing.T) {
	t.Parallel()
	for i, cal := range calendarSet() {
		per := cal.TicksPerYear()
		for _, tick := range []int64{0, 1, per / 2, per - 1, per, 3*per + 7, 1_000_000} {
			a, errA := cal.At(tick)
			b, errB := cal.At(tick + per)
			if errA != nil || errB != nil {
				t.Fatalf("calendar %d At(%d) / At(%d) errors = %v, %v, want nil", i, tick, tick+per, errA, errB)
			}
			b.Year--
			b.Tick = a.Tick
			b.DayOfWeek = a.DayOfWeek
			if a != b {
				t.Errorf("calendar %d: At(%d) = %+v, At(%d) with the year less one = %+v, want equal", i, tick, a, tick+per, b)
			}
		}
	}
}

// TestCalendar_SeasonAt verifies SeasonAt on the whole int64 range: it equals
// At's season where At accepts the tick, repeats every TicksPerYear ticks
// including for negative ticks, is safe at both ends, and is -1 with no seasons.
//
// Requirement: FR-CAL-019, FR-CAL-020
func TestCalendar_SeasonAt(t *testing.T) {
	t.Parallel()
	for i, cal := range calendarSet() {
		per := cal.TicksPerYear()
		for _, tick := range []int64{0, 5, 59, per - 1, per, 2 * per, 12345} {
			m, err := cal.At(tick)
			if err != nil {
				t.Fatalf("calendar %d At(%d) error = %v, want nil", i, tick, err)
			}
			if got := cal.SeasonAt(tick); got != m.Season {
				t.Errorf("calendar %d: SeasonAt(%d) = %d, want At's %d", i, tick, got, m.Season)
			}
		}
		// MinInt64 + per and MaxInt64 - per cannot wrap; the other direction
		// would wrap in the test itself.
		if a, b := cal.SeasonAt(math.MinInt64), cal.SeasonAt(math.MinInt64+per); a != b {
			t.Errorf("calendar %d: SeasonAt(MinInt64) = %d, SeasonAt(MinInt64+%d) = %d, want equal", i, a, per, b)
		}
		if a, b := cal.SeasonAt(math.MaxInt64), cal.SeasonAt(math.MaxInt64-per); a != b {
			t.Errorf("calendar %d: SeasonAt(MaxInt64) = %d, SeasonAt(MaxInt64-%d) = %d, want equal", i, a, per, b)
		}
		for _, tick := range []int64{-1, -per, -per - 1, -1_000_003, 0, 17, 1_000_000} {
			if a, b := cal.SeasonAt(tick), cal.SeasonAt(tick-per); a != b {
				t.Errorf("calendar %d: SeasonAt(%d) = %d, SeasonAt(%d) = %d, want equal", i, tick, a, tick-per, b)
			}
		}
	}
	noSeasons := calWith(func(c *economysim.Calendar) { c.Seasons = nil })
	for _, tick := range []int64{math.MinInt64, -1, 0, 99, math.MaxInt64} {
		if got := noSeasons.SeasonAt(tick); got != -1 {
			t.Errorf("no seasons: SeasonAt(%d) = %d, want -1", tick, got)
		}
	}
	// A negative tick is the virtual start of a warm-start job: tick -1 is day
	// 364 of the year, offset 305, winter.
	if got := economysim.DefaultCalendar().SeasonAt(-1); got != 3 {
		t.Errorf("SeasonAt(-1) = %d, want 3 (winter)", got)
	}
}

// calErrors returns the ValidationErrors of err under the calendar section.
func calErrors(err error) []*economysim.ValidationError {
	var out []*economysim.ValidationError
	j, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return nil
	}
	for _, e := range j.Unwrap() {
		var ve *economysim.ValidationError
		if errors.As(e, &ve) && len(ve.Path) > 0 && ve.Path[0] == "calendar" {
			out = append(out, ve)
		}
	}
	return out
}

// derivationOwns reports whether e is one of the sentinels the derivation
// owns.
func derivationOwns(e error) bool {
	for _, s := range []error{economysim.ErrTooMany, economysim.ErrOutOfBounds, economysim.ErrCalendarTick, economysim.ErrCalendarSeasons, economysim.ErrOverflow} {
		if errors.Is(e, s) {
			return true
		}
	}
	return false
}

// checkAgreement fails the test when the derivation and Validate disagree on
// the calendar of cfg.
func checkAgreement(t *testing.T, label string, cfg economysim.Config) {
	t.Helper()
	days, ticks, derr := economysim.DeriveCalendar(cfg.Calendar)
	var owned []string
	for _, ve := range calErrors(cfg.Validate()) {
		if derivationOwns(ve.Rule) {
			owned = append(owned, ve.Error())
		}
	}
	switch {
	case derr == nil && len(owned) > 0:
		t.Errorf("%s: derivation accepts but Validate raises %v", label, owned)
	case derr != nil && len(owned) == 0:
		t.Errorf("%s: derivation refuses with %v but Validate raises no derivation-owned error under calendar", label, derr)
	}
	if derr == nil {
		var want int64
		for _, m := range cfg.Calendar.Months {
			want += m.Days
		}
		if days != want || ticks != want*cfg.Calendar.TicksPerDay/cfg.Calendar.DaysPerTick {
			t.Errorf("%s: derivation = %d days, %d ticks, want %d days, %d ticks", label, days, ticks, want, want*cfg.Calendar.TicksPerDay/cfg.Calendar.DaysPerTick)
		}
	}
}

// TestCalendar_DerivationAgreesWithValidator verifies that over the Stage 0
// packs, their defect overlays and generated calendars the derivation accepts
// exactly the calendars that Validate accepts over the rules the derivation
// owns, with the same TicksPerYear, and that the ID rules are excluded on
// purpose: a duplicate month ID is accepted by the derivation and rejected by
// Validate.
//
// Requirement: FR-CAL-011, FR-CAL-012, FR-CAL-013, FR-CAL-022
//
// Guard: the validator carries paths and the derivation does not, so the two
// are separate code for one set of rules. If they drifted, At would accept a
// calendar the world refuses or the reverse, and no test would say so.
func TestCalendar_DerivationAgreesWithValidator(t *testing.T) {
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	layers := map[string][]string{
		"one":    {"base.toml", "one.toml", "warm-one.toml"},
		"second": {"second/second.toml", "second/warm-second.toml"},
	}
	for _, d := range []string{"d1_seed_cycle", "d2_upkeep_unmade", "d3_idle_role", "d4_long_job", "d5_storage_gap", "d6_floor_full"} {
		layers["defect "+d] = []string{"base.toml", "one.toml", "defects/" + d + ".toml"}
	}
	for name, l := range layers {
		cfg, err := config.LoadFS(fsys, l...)
		if err != nil {
			t.Fatalf("LoadFS(%q) error = %v, want nil", l, err)
		}
		checkAgreement(t, name, cfg)
	}

	for _, tt := range invalidCalendars() {
		cfg := referenceConfig()
		cfg.Calendar = tt.cal
		checkAgreement(t, "invalid "+tt.name, cfg)
	}

	// Generated calendars: small values around every bound, so a good share
	// are valid and the rest break one rule or several.
	for seed := uint64(1); seed <= 400; seed++ {
		r := &splitmix{state: seed}
		cal := economysim.Calendar{
			HoursPerDay: int64(r.IntN(30)),
			DaysPerWeek: int64(r.IntN(9)),
			TicksPerDay: int64(1 + r.IntN(3)),
			DaysPerTick: int64(r.IntN(8)),
		}
		if r.IntN(5) == 0 {
			cal.HoursPerDay, cal.DaysPerWeek = 24, 7
		}
		var days int64
		for i := range 1 + r.IntN(4) {
			d := int64(r.IntN(40)) - 1
			if r.IntN(3) > 0 {
				d = int64(1 + r.IntN(40))
			}
			cal.Months = append(cal.Months, economysim.Month{ID: "m" + strconv.Itoa(i), Days: d})
			days += d
		}
		cal.StartDay = int64(r.IntN(int(max(days, 1))+2)) - 1
		if n := r.IntN(4); n > 0 && days > 0 {
			left := days
			for i := range n {
				d := int64(1 + r.IntN(int(days)))
				if i == n-1 && r.IntN(4) > 0 {
					d = left
				}
				left -= d
				cal.Seasons = append(cal.Seasons, economysim.Season{ID: "s" + strconv.Itoa(i), Days: d})
			}
			cal.SeasonStart = int64(r.IntN(int(days)+2)) - 1
		}
		cfg := referenceConfig()
		cfg.Calendar = cal
		checkAgreement(t, "seed "+strconv.FormatUint(seed, 10), cfg)
	}

	// The ID rules are the validator's alone.
	cfg := referenceConfig()
	cfg.Calendar.Months[1].ID = cfg.Calendar.Months[0].ID
	if _, _, err := economysim.DeriveCalendar(cfg.Calendar); err != nil {
		t.Errorf("derivation of a duplicate month ID error = %v, want nil: the ID rules are not numeric", err)
	}
	if err := cfg.Validate(); !errors.Is(err, economysim.ErrDuplicateID) {
		t.Errorf("Validate() of a duplicate month ID error = %v, want ErrDuplicateID", err)
	}
}

// TestCalendar_DayTableAgreesWithAt verifies that the day-of-year table gives
// the month, day of month and season that At gives, for every day of the
// year, for the default, the 320-day planet, a calendar whose season cycle
// wraps and one with no seasons.
//
// Requirement: FR-CAL-009, FR-CAL-018
//
// Guard: the table is built by a different walk than At, for the world's
// precomputation (spec-calendar.md 4.6); a disagreement would give two
// different seasons for one day.
func TestCalendar_DayTableAgreesWithAt(t *testing.T) {
	t.Parallel()
	cals := append(calendarSet(), calWith(func(c *economysim.Calendar) { c.SeasonStart = 300 }))
	for i, cal := range cals {
		days, _, err := economysim.DeriveCalendar(cal)
		if err != nil {
			t.Fatalf("calendar %d derivation error = %v, want nil", i, err)
		}
		table := economysim.NewDayTable(cal)
		// Day d of the year is reached at the tick whose absolute day is d:
		// with TicksPerDay and DaysPerTick 1 the tick is d - StartDay mod days.
		unit := cal
		unit.TicksPerDay, unit.DaysPerTick = 1, 1
		for d := range days {
			tick := (d - cal.StartDay + days) % days
			m, err := unit.At(tick)
			if err != nil {
				t.Fatalf("calendar %d At(%d) error = %v, want nil", i, tick, err)
			}
			if int64(m.DayOfYear) != d {
				t.Fatalf("calendar %d: At(%d).DayOfYear = %d, want %d", i, tick, m.DayOfYear, d)
			}
			month, dom, season := economysim.DayTableDay(table, d)
			if month != m.Month || dom != int64(m.DayOfMonth) || season != m.Season {
				t.Errorf("calendar %d day %d: table = month %d day %d season %d, At = month %d day %d season %d",
					i, d, month, dom, season, m.Month, m.DayOfMonth, m.Season)
			}
		}
	}
}

// TestCalendar_NewDayTablePanicsOnInvalid verifies that a calendar the
// derivation refuses is a programmer error with a message that says so.
//
// Guard: architecture.md 8.1 allows a panic only for a violated invariant and
// requires the message to say so; the table builder takes a validated calendar.
func TestCalendar_NewDayTablePanicsOnInvalid(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		msg, _ := r.(string)
		if !strings.Contains(msg, "invariant") {
			t.Errorf("NewDayTable(Calendar{}) panic = %v, want a message that says invariant", r)
		}
	}()
	economysim.NewDayTable(economysim.Calendar{})
	t.Errorf("NewDayTable(Calendar{}) returned, want a panic")
}

// TestCalendar_PacksUnchangedAndCanonical verifies that building moments does
// not change the configuration it reads (two packs with different calendars)
// and that a canonicalised copy gives equal moments for every tick of a year.
//
// Requirement: FR-CAL-017, FR-CAL-018
func TestCalendar_PacksUnchangedAndCanonical(t *testing.T) {
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	for name, l := range map[string][]string{
		"one":    {"base.toml", "one.toml", "warm-one.toml"},
		"second": {"second/second.toml", "second/warm-second.toml"},
	} {
		cfg, err := config.LoadFS(fsys, l...)
		if err != nil {
			t.Fatalf("LoadFS(%q) error = %v, want nil", l, err)
		}
		before := economysim.Canonical(cfg)
		copyCfg := economysim.Canonical(cfg)
		per := cfg.Calendar.TicksPerYear()
		if per < 1 {
			t.Fatalf("pack %s TicksPerYear() = %d, want at least 1", name, per)
		}
		for tick := range per {
			a, errA := cfg.Calendar.At(tick)
			b, errB := copyCfg.Calendar.At(tick)
			if errA != nil || errB != nil || a != b {
				t.Fatalf("pack %s At(%d) = %+v, %v and canonical copy %+v, %v, want equal", name, tick, a, errA, b, errB)
			}
			cfg.Calendar.SeasonAt(tick)
		}
		economysim.NewDayTable(cfg.Calendar)
		if !reflect.DeepEqual(economysim.Canonical(cfg), before) {
			t.Errorf("pack %s: Config after At, SeasonAt and NewDayTable differs from before, want unchanged", name)
		}
	}
}

// TestCalendar_NoAllocations verifies that At, SeasonAt, TicksPerYear and a
// table read allocate nothing.
//
// Requirement: NFR-CAL-001, NFR-CAL-003
func TestCalendar_NoAllocations(t *testing.T) {
	cal := economysim.DefaultCalendar()
	table := economysim.NewDayTable(cal)
	var sink int
	tests := []struct {
		name string
		f    func()
	}{
		{"At", func() { m, _ := cal.At(1234); sink += m.Month }},
		{"SeasonAt", func() { sink += cal.SeasonAt(-98765) }},
		{"TicksPerYear", func() { sink += int(cal.TicksPerYear()) }},
		{"table read", func() { m, _, s := economysim.DayTableDay(table, 200); sink += m + s }},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, tt.f); got != 0 {
			t.Errorf("%s allocates %v per run, want 0", tt.name, got)
		}
	}
	_ = sink
}

// decodeCalendar turns bytes into a calendar of small values, some of which
// break a rule, so the fuzzer reaches every refusal and many valid calendars.
func decodeCalendar(data []byte) economysim.Calendar {
	i := 0
	next := func() int64 {
		if i >= len(data) {
			return 0
		}
		i++
		return int64(data[i-1])
	}
	cal := economysim.Calendar{
		HoursPerDay: next() % 40,
		DaysPerWeek: next() % 9,
		TicksPerDay: next() % 4,
		DaysPerTick: next() % 8,
	}
	cal.StartDay = next() - 8
	cal.SeasonStart = next() - 8
	n := int(next() % 6)
	if next() == 255 {
		n = economysim.MaxMonths + 1
	}
	var days int64
	for j := range n {
		d := next() % 50
		cal.Months = append(cal.Months, economysim.Month{ID: "m" + strconv.Itoa(j), Days: d})
		days += d
	}
	ns := int(next() % 4)
	exact := next()&1 == 1
	for j := range ns {
		d := next() % 80
		if exact && j == ns-1 && days-sumSeasons(cal.Seasons) > 0 {
			d = days - sumSeasons(cal.Seasons)
		}
		cal.Seasons = append(cal.Seasons, economysim.Season{ID: "s" + strconv.Itoa(j), Days: d})
	}
	return cal
}

func sumSeasons(s []economysim.Season) int64 {
	var n int64
	for _, e := range s {
		n += e.Days
	}
	return n
}

// FuzzCalendarAt verifies that for any calendar and tick At, SeasonAt and
// TicksPerYear never panic, that the derivation and At agree on whether the
// calendar is usable, that a moment's fields are within their ranges, that
// SeasonAt is in range, equals At's season and is periodic. The validator half
// of NFR-CAL-002 is FuzzValidate.
//
// Requirement: NFR-CAL-002, NFR-CAL-003, FR-CAL-019, FR-CAL-020, FR-CAL-022
func FuzzCalendarAt(f *testing.F) {
	earth := []byte{24, 7, 1, 1, 8, 67, 4, 0, 31, 28, 31, 30, 0, 3, 92, 92, 91, 90}
	f.Add(earth, int64(59))
	f.Add(earth, int64(-1))
	f.Add(earth, int64(math.MaxInt64))
	f.Add(earth, int64(math.MinInt64))
	f.Add([]byte{20, 5, 2, 1, 8, 8, 2, 0, 40, 40, 0, 2, 120, 200}, int64(639))
	f.Add([]byte{}, int64(0))
	f.Add([]byte{24, 7, 1, 5, 8, 8, 3, 255}, int64(7))
	f.Fuzz(func(t *testing.T, data []byte, tick int64) {
		cal := decodeCalendar(data)
		days, per, derr := economysim.DeriveCalendar(cal)
		if got := cal.TicksPerYear(); (derr == nil) != (got > 0) || (derr == nil && got != per) {
			t.Fatalf("TicksPerYear() = %d with derivation error %v, want %d only when valid", got, derr, per)
		}
		m, err := cal.At(tick)
		if derr != nil {
			if !errors.Is(err, derr) || m != (economysim.Moment{}) {
				t.Fatalf("At(%d) = %+v, %v on a calendar the derivation refuses with %v, want that error and no moment", tick, m, err, derr)
			}
			if got := cal.SeasonAt(tick); got != -1 {
				t.Fatalf("SeasonAt(%d) = %d on an invalid calendar, want -1", tick, got)
			}
			return
		}
		season := cal.SeasonAt(tick)
		if season < -1 || season >= len(cal.Seasons) || (len(cal.Seasons) == 0 && season != -1) {
			t.Fatalf("SeasonAt(%d) = %d with %d seasons, want in [-1, %d)", tick, season, len(cal.Seasons), len(cal.Seasons))
		}
		if tick >= math.MinInt64+per {
			if other := cal.SeasonAt(tick - per); other != season {
				t.Fatalf("SeasonAt(%d) = %d but SeasonAt(%d) = %d, want equal", tick, season, tick-per, other)
			}
		}
		if err != nil {
			if tick >= 0 && !errors.Is(err, economysim.ErrOverflow) {
				t.Fatalf("At(%d) error = %v on a valid calendar, want nil or ErrOverflow", tick, err)
			}
			if tick < 0 && !errors.Is(err, economysim.ErrOutOfBounds) {
				t.Fatalf("At(%d) error = %v, want ErrOutOfBounds", tick, err)
			}
			return
		}
		if m.Season != season {
			t.Fatalf("At(%d).Season = %d, SeasonAt = %d, want equal", tick, m.Season, season)
		}
		if int64(m.DayOfYear) >= days || m.DayOfYear < 0 || m.Month < 0 || m.Month >= len(cal.Months) ||
			m.DayOfMonth < 0 || int64(m.DayOfMonth) >= cal.Months[m.Month].Days ||
			m.DayOfWeek < 0 || int64(m.DayOfWeek) >= cal.DaysPerWeek ||
			m.TickOfDay < 0 || int64(m.TickOfDay) >= cal.TicksPerDay ||
			m.Hour < 0 || int64(m.Hour) >= cal.HoursPerDay || m.Year < 0 || m.Tick != tick {
			t.Fatalf("At(%d) = %+v out of range for %+v", tick, m, cal)
		}
		if len(cal.Seasons) > 0 && (m.DayOfSeason < 0 || int64(m.DayOfSeason) >= cal.Seasons[m.Season].Days) {
			t.Fatalf("At(%d).DayOfSeason = %d out of range for season %d of %+v", tick, m.DayOfSeason, m.Season, cal)
		}
	})
}

// TestCalendar_WrittenOrderKept verifies that the months and seasons of a
// calendar keep their written order through canonicalisation, so the moment
// derived from a canonical copy equals the one from the original, and that
// the day table lists months in that order.
//
// Requirement: FR-CAL-015
func TestCalendar_WrittenOrderKept(t *testing.T) {
	t.Parallel()
	for i, cal := range calendarSet() {
		cfg := economysim.Config{Calendar: cal}
		got := economysim.Canonical(cfg).Calendar
		if !reflect.DeepEqual(got.Months, cal.Months) || !reflect.DeepEqual(got.Seasons, cal.Seasons) {
			t.Errorf("calendar %d: canonical months/seasons = %v / %v, want %v / %v", i, got.Months, got.Seasons, cal.Months, cal.Seasons)
		}
		tbl := economysim.NewDayTable(cal)
		days, _, _ := economysim.DeriveCalendar(cal)
		last := -1
		for d := int64(0); d < days; d++ {
			m, _, _ := economysim.DayTableDay(tbl, d)
			if m < last {
				t.Fatalf("calendar %d: day %d is in month %d after month %d, want written order", i, d, m, last)
			}
			last = m
		}
	}
}

// flatCalendar is a calendar of n months of days days each, no seasons,
// ticksPerDay ticks a day and one day a tick.
func flatCalendar(n int, days, ticksPerDay int64) economysim.Calendar {
	c := economysim.DefaultCalendar()
	c.Months, c.Seasons, c.SeasonStart, c.TicksPerDay = nil, nil, 0, ticksPerDay
	for range n {
		c.Months = append(c.Months, economysim.Month{ID: "m", Days: days})
	}
	return c
}

// TestCalendar_BoundsAreInclusive verifies that exactly MaxMonths months,
// exactly MaxSeasons seasons and exactly MaxTicksPerYear ticks are accepted,
// and one more of each is refused.
//
// Requirement: FR-CAL-013, FR-CAL-022
func TestCalendar_BoundsAreInclusive(t *testing.T) {
	t.Parallel()
	seasons := flatCalendar(1, economysim.MaxSeasons, 1)
	seasons.Seasons = nil
	for range economysim.MaxSeasons {
		seasons.Seasons = append(seasons.Seasons, economysim.Season{ID: "s", Days: 1})
	}
	tests := []struct {
		name    string
		cal     economysim.Calendar
		perYear int64
		want    error
	}{
		// 100 months of 1 day: 100 days, 100 ticks.
		{"max months", flatCalendar(economysim.MaxMonths, 1, 1), economysim.MaxMonths, nil},
		// one month of 100 days split in 100 seasons of 1 day.
		{"max seasons", seasons, economysim.MaxSeasons, nil},
		// 5 months of 10 000 days = 50 000 days, 2 ticks a day = 100 000.
		{"max ticks per year", flatCalendar(5, 10_000, 2), economysim.MaxTicksPerYear, nil},
		// 6 months of 8 334 days = 50 004 days, 2 ticks a day = 100 008 ticks,
		// over the bound with every month inside its own.
		{"over max ticks per year", flatCalendar(6, 8_334, 2), 0, economysim.ErrOutOfBounds},
		{"max hours per day", calWith(func(c *economysim.Calendar) { c.HoursPerDay = economysim.MaxHoursPerDay }), 365, nil},
		{"max days per week", calWith(func(c *economysim.Calendar) { c.DaysPerWeek = economysim.MaxDaysPerWeek }), 365, nil},
		// 1 month of 100 days at 1 000 ticks a day = 100 000 ticks.
		{"max ticks per day", flatCalendar(1, 100, economysim.MaxTicksPerDay), economysim.MaxTicksPerYear, nil},
		// 10 months of 10 000 days = MaxDaysPerYear, one tick a day.
		{"max days per year", flatCalendar(10, economysim.MaxDaysPerMonth, 1), economysim.MaxTicksPerYear, nil},
		// One season that is the whole 100 000-day year: the cap on a season's
		// days must not cut it below the year.
		{"season as long as the longest year", func() economysim.Calendar {
			c := flatCalendar(10, economysim.MaxDaysPerMonth, 1)
			c.Seasons = []economysim.Season{{ID: "s", Days: economysim.MaxDaysPerYear}}
			return c
		}(), economysim.MaxTicksPerYear, nil},
		// 1 day, 1 tick: the smallest year.
		{"one tick year", flatCalendar(1, 1, 1), 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tt.cal.At(0)
			if !errors.Is(err, tt.want) {
				t.Errorf("At(0) error = %v, want %v", err, tt.want)
			}
			if got := tt.cal.TicksPerYear(); got != tt.perYear {
				t.Errorf("TicksPerYear() = %d, want %d", got, tt.perYear)
			}
		})
	}
}

// TestCalendar_AtHourTruncates verifies that an hour that is not a whole
// number of ticks is truncated toward zero.
//
// Requirement: FR-CAL-008
func TestCalendar_AtHourTruncates(t *testing.T) {
	t.Parallel()
	c := economysim.DefaultCalendar()
	c.HoursPerDay, c.TicksPerDay = 24, 5
	// Hour = tickOfDay * 24 / 5: 0, 4.8, 9.6, 14.4, 19.2 truncated.
	for tick, want := range []int{0, 4, 9, 14, 19} {
		m, err := c.At(int64(tick))
		if err != nil || m.Hour != want || m.TickOfDay != tick {
			t.Errorf("At(%d) = hour %d tickOfDay %d err %v, want hour %d", tick, m.Hour, m.TickOfDay, err, want)
		}
	}
}

// splitmix is a fixed-seed SplitMix64 generator for the generated calendars:
// the same seed gives the same calendar on every platform, so a failing seed
// reproduces.
type splitmix struct{ state uint64 }

// IntN returns a value in [0, n) for n >= 1; the modulo bias is irrelevant to
// a generator of test inputs.
func (s *splitmix) IntN(n int) int {
	s.state += 0x9e3779b97f4a7c15
	z := s.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return int((z ^ (z >> 31)) % uint64(n))
}
