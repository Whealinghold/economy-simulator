package indicator

import (
	"sort"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// YearSpan is a calendar-year window [From, To) in ticks. Partial is true when
// it has fewer than TicksPerYear ticks, as the first year of a world does when
// the calendar's StartDay is not 0.
type YearSpan struct {
	From, To int64
	Partial  bool
}

// yearOf returns the calendar year of a tick, or ErrOverflow when the tick is
// outside what the calendar can read.
func yearOf(cal economysim.Calendar, tick int64) (int64, error) {
	m, err := cal.At(tick)
	if err != nil {
		return 0, wrap("YearWindow", "calendar", ErrOverflow)
	}
	return m.Year, nil
}

// firstTickOfYear returns the first tick whose year is at least y, by binary
// search over the monotone cal.At: tick (y+1) * TicksPerYear is already in a
// later year, because StartDay is not negative.
func firstTickOfYear(cal economysim.Calendar, y, perYear int64) (int64, error) {
	hi, ok := measure.ToInt64(measure.MulInt64(y+1, perYear))
	if !ok {
		return 0, wrap("YearWindow", "year", ErrOverflow)
	}
	var failed error
	t := int64(sort.Search(int(min(hi, 1<<62)), func(i int) bool {
		if failed != nil {
			return true
		}
		var yr int64
		yr, failed = yearOf(cal, int64(i))
		return yr >= y
	}))
	return t, failed
}

// YearWindow returns the ticks whose Moment.Year is y: [first tick of year y,
// first tick of year y+1), found by a binary search over cal.At
// (spec-indicators.md 10.2.1). Partial is set when the window has fewer than
// TicksPerYear ticks. err is ErrRange for a negative year or a calendar that
// fails its rules and ErrOverflow for a year the calendar cannot read.
func YearWindow(cal economysim.Calendar, y int64) (YearSpan, error) {
	perYear := cal.TicksPerYear()
	if y < 0 || perYear < 1 {
		return YearSpan{}, wrap("YearWindow", "year", ErrRange)
	}
	from, err := firstTickOfYear(cal, y, perYear)
	if err != nil {
		return YearSpan{}, err
	}
	to, err := firstTickOfYear(cal, y+1, perYear)
	if err != nil {
		return YearSpan{}, err
	}
	return YearSpan{From: from, To: to, Partial: to-from < perYear}, nil
}

// SeasonRun is a maximal run of consecutive ticks of one season, clipped to the
// range asked. Year is the calendar year in which the run starts, so a season
// that wraps the year end belongs to the year it starts in. Clipped is set when
// the range cut the run: it started before From or goes on at To.
type SeasonRun struct {
	From, To int64
	Year     int64
	Clipped  bool
}

// SeasonWindows returns the runs of the season s (an index into the calendar's
// seasons) inside [from, to), in order (spec-indicators.md 10.2.2). A season
// that wraps the year end is one run across the boundary. err is ErrRange for a
// negative season, an empty or negative range or a calendar that fails its
// rules, and ErrOverflow for a tick the calendar cannot read.
//
// shortcut: one cal.At per tick of the range, O(to - from); upgrade to a step
// by whole days when a consumer reads seasons over decades of fine ticks.
func SeasonWindows(cal economysim.Calendar, from, to int64, s int) ([]SeasonRun, error) {
	if s < 0 || from < 0 || to < from || cal.TicksPerYear() < 1 {
		return nil, wrap("SeasonWindows", "range", ErrRange)
	}
	var runs []SeasonRun
	for t := from; t < to; t++ {
		m, err := cal.At(t)
		if err != nil {
			return nil, wrap("SeasonWindows", "calendar", ErrOverflow)
		}
		runs = stepSeason(cal, runs, m, from, s)
	}
	return closeSeason(cal, runs, to, s), nil
}

// stepSeason folds one tick into the runs: it opens a run (To -1 while open) at
// the first tick of the season, and closes it at the first tick that is not.
func stepSeason(cal economysim.Calendar, runs []SeasonRun, m economysim.Moment, from int64, s int) []SeasonRun {
	open := len(runs) > 0 && runs[len(runs)-1].To < 0
	switch {
	case m.Season == s && !open:
		clipped := m.Tick == from && from > 0 && seasonAt(cal, m.Tick-1) == s
		return append(runs, SeasonRun{From: m.Tick, To: -1, Year: m.Year, Clipped: clipped})
	case m.Season != s && open:
		runs[len(runs)-1].To = m.Tick
	}
	return runs
}

// closeSeason ends a run still open at the end of the range, which clips it
// when the season goes on at the tick after.
func closeSeason(cal economysim.Calendar, runs []SeasonRun, to int64, s int) []SeasonRun {
	if n := len(runs); n > 0 && runs[n-1].To < 0 {
		runs[n-1].To = to
		runs[n-1].Clipped = runs[n-1].Clipped || seasonAt(cal, to) == s
	}
	return runs
}

// seasonAt returns the season of a tick, or -1 when the calendar cannot read it.
func seasonAt(cal economysim.Calendar, tick int64) int {
	m, err := cal.At(tick)
	if err != nil {
		return -1
	}
	return m.Season
}
