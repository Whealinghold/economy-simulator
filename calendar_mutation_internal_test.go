package economysim

import (
	"errors"
	"math"
	"testing"
)

// The tests in this file pin edges of calendar.go that a mutation run (gomu,
// inside Docker) showed calendar_test.go did not reach.
//
// Equivalent mutants (cannot be killed, each reasoned from the code):
//   - line 67 `len(c.Months) == 0` -> `<` or `<=`: a length is never negative.
//   - lines 91, 201 and 222 `len(c.Seasons) == 0` -> `<` or `<=`: same.
//   - line 201 (`== 0` -> `false`, `0` -> `1` is killed below, empty block):
//     with no seasons the loop over Seasons is empty, so Season stays -1.
//   - line 101 `MaxDaysPerYear+1` -> `+2`: any capped season already exceeds the
//     year, so the sum differs from days either way.
//   - lines 114 and 178 `TicksPerDay > 1` / `DaysPerTick > 1` -> `!= 1`: checkFields
//     has already rejected values below 1.
//   - line 121 `prod%DaysPerTick != 0` -> `> 0`: a remainder of positive operands
//     is never negative.
//   - line 125 `ticks < 1` -> `< 0`: a positive product divided exactly is >= 1.
//   - lines 136 and 230 (`if err != nil` blocks and conditions): derive returns 0
//     ticks on error, and At cannot refuse a reduced tick.
//   - line 226 `r < 0` -> `true`, `<=`, `!=`, `< 1`: adding one year to r moves
//     the day by whole years, so the season is the same.
//   - line 233 `-1` -> `+1`, `2`, `0`: the unreachable branch after line 230.

// TestCalendar_DeriveErrorsReturnZeroes verifies that derive returns zero days
// and zero ticks together with the sentinel of the rule each calendar breaks.
//
// Requirement: FR-CAL-022
func TestCalendar_DeriveErrorsReturnZeroes(t *testing.T) {
	t.Parallel()
	tooMany := DefaultCalendar()
	tooMany.Months = make([]Month, MaxMonths+1)
	badFields := DefaultCalendar()
	badFields.DaysPerWeek = 0
	badDays := DefaultCalendar()
	badDays.StartDay = 365
	badSeasons := DefaultCalendar()
	badSeasons.Seasons = []Season{{ID: "s", Days: 364}}
	badTicks := DefaultCalendar()
	badTicks.TicksPerDay, badTicks.DaysPerTick = 2, 2
	tests := []struct {
		name string
		cal  Calendar
		want error
	}{
		{"too many months", tooMany, ErrTooMany},
		{"bad scalar field", badFields, ErrOutOfBounds},
		{"start day past the year", badDays, ErrOutOfBounds},
		// One season of 364 days against a 365-day year.
		{"seasons do not tile", badSeasons, ErrCalendarSeasons},
		{"tick of both kinds", badTicks, ErrCalendarTick},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			days, ticks, err := tt.cal.derive()
			if !errors.Is(err, tt.want) || days != 0 || ticks != 0 {
				t.Errorf("derive() = %d, %d, %v, want 0, 0, %v", days, ticks, err, tt.want)
			}
		})
	}
}

// TestCalendar_CheckFieldsRejectsZero verifies that a zero in each scalar field
// that checkFields bounds from below is refused on its own.
//
// Requirement: FR-CAL-022
func TestCalendar_CheckFieldsRejectsZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		set  func(*Calendar)
	}{
		{"hours per day", func(c *Calendar) { c.HoursPerDay = 0 }},
		{"days per week", func(c *Calendar) { c.DaysPerWeek = 0 }},
		{"ticks per day", func(c *Calendar) { c.TicksPerDay = 0 }},
		{"days per tick", func(c *Calendar) { c.DaysPerTick = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := DefaultCalendar()
			tt.set(&c)
			if err := c.checkFields(); !errors.Is(err, ErrOutOfBounds) {
				t.Errorf("checkFields() = %v, want %v", err, ErrOutOfBounds)
			}
		})
	}
}

// TestCalendar_StepsReturnZeroOnRefusal verifies that yearDays, yearTicks and
// absoluteDay return zero values with the sentinel or the false they refuse
// with.
//
// Requirement: FR-CAL-022
func TestCalendar_StepsReturnZeroOnRefusal(t *testing.T) {
	t.Parallel()
	t.Run("year days, month out of bounds", func(t *testing.T) {
		t.Parallel()
		c := DefaultCalendar()
		c.Months = []Month{{ID: "m", Days: 0}}
		if days, err := c.yearDays(); days != 0 || !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("yearDays() = %d, %v, want 0, %v", days, err, ErrOutOfBounds)
		}
	})
	t.Run("year days, start day past the year", func(t *testing.T) {
		t.Parallel()
		c := DefaultCalendar()
		c.StartDay = 365
		if days, err := c.yearDays(); days != 0 || !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("yearDays() = %d, %v, want 0, %v", days, err, ErrOutOfBounds)
		}
	})
	ticks := []struct {
		name         string
		days         int64
		perDay, perT int64
		want         error
	}{
		{"both tick kinds", 365, 2, 2, ErrCalendarTick},
		{"product overflows", math.MaxInt64, 2, 1, ErrOverflow},
		// 7 * 1 = 7 is not a multiple of 2.
		{"not a whole number of ticks", 7, 1, 2, ErrCalendarTick},
		// 100 001 days at one tick a day is one over MaxTicksPerYear.
		{"too many ticks", MaxTicksPerYear + 1, 1, 1, ErrOutOfBounds},
	}
	for _, tt := range ticks {
		t.Run("year ticks, "+tt.name, func(t *testing.T) {
			t.Parallel()
			c := DefaultCalendar()
			c.TicksPerDay, c.DaysPerTick = tt.perDay, tt.perT
			if got, err := c.yearTicks(tt.days); got != 0 || !errors.Is(err, tt.want) {
				t.Errorf("yearTicks(%d) = %d, %v, want 0, %v", tt.days, got, err, tt.want)
			}
		})
	}
	t.Run("absolute day overflows", func(t *testing.T) {
		t.Parallel()
		c := DefaultCalendar()
		c.DaysPerTick = 2
		// MaxInt64 * 2 does not fit.
		if day, tod, ok := c.absoluteDay(math.MaxInt64); day != 0 || tod != 0 || ok {
			t.Errorf("absoluteDay(MaxInt64) = %d, %d, %v, want 0, 0, false", day, tod, ok)
		}
	})
}

// TestCalendar_SingleSeasonYear verifies the moment and the season of a year
// that is one season long.
//
// Requirement: FR-CAL-007, FR-CAL-019
func TestCalendar_SingleSeasonYear(t *testing.T) {
	t.Parallel()
	c := DefaultCalendar()
	c.SeasonStart = 0
	c.Seasons = []Season{{ID: "all", Days: 365}}
	m, err := c.At(100)
	if err != nil || m.Season != 0 || m.DayOfSeason != 100 {
		t.Errorf("At(100) = season %d day %d, %v, want season 0 day 100", m.Season, m.DayOfSeason, err)
	}
	if got := c.SeasonAt(100); got != 0 {
		t.Errorf("SeasonAt(100) = %d, want 0", got)
	}
}
