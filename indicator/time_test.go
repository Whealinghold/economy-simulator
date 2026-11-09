package indicator_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// calendar365 is the calendar of spec-indicators.md 18.10: 365 one-day ticks, a
// world that starts on day 100, and two seasons, warm (days 59 to 258) and cold
// (days 259 to 364 and 0 to 58, which wraps the year end).
func calendar365() economysim.Calendar {
	return economysim.Calendar{
		HoursPerDay: 24, DaysPerWeek: 7,
		Months:      []economysim.Month{{ID: "m", Days: 365}},
		TicksPerDay: 1, DaysPerTick: 1,
		StartDay:    100,
		Seasons:     []economysim.Season{{ID: "warm", Days: 200}, {ID: "cold", Days: 165}},
		SeasonStart: 59,
	}
}

// TestYearWindow_PartialFirstYear verifies the calendar-year windows of
// spec-indicators.md 18.10: with StartDay 100 year 0 is [0, 265) and partial, and
// year 1 is [265, 630) and whole; a calendar that starts on day 0 has no partial
// year; a negative year or an invalid calendar is ErrRange.
//
// Requirement: FR-IND-052
func TestYearWindow_PartialFirstYear(t *testing.T) {
	t.Parallel()
	cal := calendar365()
	whole := cal
	whole.StartDay = 0
	tests := []struct {
		name string
		cal  economysim.Calendar
		year int64
		want indicator.YearSpan
	}{
		{"first year is partial", cal, 0, indicator.YearSpan{From: 0, To: 265, Partial: true}},
		{"second year is whole", cal, 1, indicator.YearSpan{From: 265, To: 630}},
		{"a later year", cal, 5, indicator.YearSpan{From: 265 + 4*365, To: 265 + 5*365}},
		{"a year starting at day 0", whole, 0, indicator.YearSpan{From: 0, To: 365}},
		{"the next", whole, 3, indicator.YearSpan{From: 3 * 365, To: 4 * 365}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := indicator.YearWindow(tt.cal, tt.year)
			if err != nil || got != tt.want {
				t.Errorf("YearWindow(%d) = %+v, %v, want %+v", tt.year, got, err, tt.want)
			}
		})
	}
	if _, err := indicator.YearWindow(cal, -1); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("YearWindow(-1) error = %v, want ErrRange", err)
	}
	if _, err := indicator.YearWindow(economysim.Calendar{}, 0); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("YearWindow of the zero calendar error = %v, want ErrRange", err)
	}
	if _, err := indicator.YearWindow(cal, 1<<62); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("YearWindow(2^62) error = %v, want ErrOverflow", err)
	}
}

// TestSeasonWindows_WrapsTheYear verifies the season windows of
// spec-indicators.md 18.10: warm is [0, 159) and [324, 524); cold is [159, 324),
// which crosses the year boundary at 265 and belongs to year 0, and [524, 630),
// clipped at the end of the range and so flagged; a range that starts inside a
// run clips it at the start; a season no tick carries has no run.
//
// Requirement: FR-IND-052
func TestSeasonWindows_WrapsTheYear(t *testing.T) {
	t.Parallel()
	cal := calendar365()
	tests := []struct {
		name     string
		from, to int64
		season   int
		want     []indicator.SeasonRun
	}{
		{"warm", 0, 630, 0, []indicator.SeasonRun{{From: 0, To: 159, Year: 0}, {From: 324, To: 524, Year: 1}}},
		{"cold wraps and is clipped at the end", 0, 630, 1, []indicator.SeasonRun{{From: 159, To: 324, Year: 0}, {From: 524, To: 630, Year: 1, Clipped: true}}},
		{"a range starting inside a run", 200, 300, 1, []indicator.SeasonRun{{From: 200, To: 300, Year: 0, Clipped: true}}},
		{"a range ending where a run ends is not clipped", 0, 159, 0, []indicator.SeasonRun{{From: 0, To: 159, Year: 0}}},
		{"a range of no ticks", 5, 5, 0, nil},
		{"a season the calendar lacks", 0, 630, 2, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := indicator.SeasonWindows(cal, tt.from, tt.to, tt.season)
			if err != nil || !slices.Equal(got, tt.want) {
				t.Errorf("SeasonWindows(%d, %d, %d) = %+v, %v, want %+v", tt.from, tt.to, tt.season, got, err, tt.want)
			}
		})
	}
	for _, bad := range []struct {
		from, to int64
		s        int
	}{{0, 10, -1}, {-1, 10, 0}, {10, 5, 0}} {
		if _, err := indicator.SeasonWindows(cal, bad.from, bad.to, bad.s); !errors.Is(err, indicator.ErrRange) {
			t.Errorf("SeasonWindows(%d, %d, %d) error = %v, want ErrRange", bad.from, bad.to, bad.s, err)
		}
	}
	if _, err := indicator.SeasonWindows(economysim.Calendar{}, 0, 5, 0); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("SeasonWindows of the zero calendar error = %v, want ErrRange", err)
	}
}

// TestYearWindow_LastYearIsOverflowNotWrapped verifies that the year past which
// the next year's first tick cannot be formed is ErrOverflow and not a wrapped
// span.
//
// Requirement: FR-IND-009
func TestYearWindow_LastYearIsOverflowNotWrapped(t *testing.T) {
	t.Parallel()
	if _, err := indicator.YearWindow(calendar365(), 1<<63-1); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("YearWindow(year MaxInt64) error = %v, want ErrOverflow", err)
	}
}
