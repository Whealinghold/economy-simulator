package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Moment is the calendar's reading of a tick: a plain value that Calendar.At
// returns (spec-calendar.md 4). Every field is derived from the calendar and
// the tick alone, so a moment is never stored or checkpointed.
type Moment struct {
	// Tick is the tick the moment reads.
	Tick int64
	// Year is 0 for the year in which tick 0 falls.
	Year int64
	// DayOfYear is the zero-based calendar day of the year.
	DayOfYear int
	// Month is the index into Calendar.Months.
	Month int
	// DayOfMonth is the zero-based calendar day of the month.
	DayOfMonth int
	// DayOfWeek is the zero-based weekday of the absolute day; it does not
	// restart with the year.
	DayOfWeek int
	// TickOfDay is the zero-based tick of the day; 0 unless TicksPerDay is
	// above 1.
	TickOfDay int
	// Hour is the zero-based hour of the day, truncated.
	Hour int
	// Season is the index into Calendar.Seasons, or -1 if there are none.
	Season int
	// DayOfSeason is the zero-based day of the season; 0 with no seasons.
	DayOfSeason int
}

// derive returns the calendar days of a year and the ticks of a year of a
// calendar, or the sentinel of the numeric rule it breaks (spec-calendar.md
// 3.2.2): the counts first, so no walk below is longer than MaxMonths plus
// MaxSeasons entries, then the field bounds, the seasons and the tick shape.
// The ID rules are the validator's alone. It allocates nothing and cannot
// panic or divide by zero.
func (c Calendar) derive() (days, ticks int64, err error) {
	if len(c.Months) > MaxMonths || len(c.Seasons) > MaxSeasons {
		return 0, 0, ErrTooMany
	}
	if c.HoursPerDay < 1 || c.HoursPerDay > MaxHoursPerDay ||
		c.DaysPerWeek < 1 || c.DaysPerWeek > MaxDaysPerWeek ||
		c.TicksPerDay < 1 || c.TicksPerDay > MaxTicksPerDay ||
		c.DaysPerTick < 1 || len(c.Months) == 0 {
		return 0, 0, ErrOutOfBounds
	}
	for _, m := range c.Months {
		if m.Days < 1 || m.Days > MaxDaysPerMonth {
			return 0, 0, ErrOutOfBounds
		}
		// At most MaxMonths months of at most MaxDaysPerMonth days: no overflow.
		days += m.Days
	}
	if days > MaxDaysPerYear || c.StartDay < 0 || c.StartDay >= days {
		return 0, 0, ErrOutOfBounds
	}
	if len(c.Seasons) > 0 {
		var sum int64
		for _, s := range c.Seasons {
			if s.Days < 1 {
				return 0, 0, ErrCalendarSeasons
			}
			// Capped one past the year, so the sum of at most MaxSeasons cannot
			// overflow and a season longer than the year still breaks the rule.
			sum += min(s.Days, MaxDaysPerYear+1)
		}
		if sum != days {
			return 0, 0, ErrCalendarSeasons
		}
		if c.SeasonStart < 0 || c.SeasonStart >= days {
			return 0, 0, ErrOutOfBounds
		}
	}
	if c.TicksPerDay > 1 && c.DaysPerTick > 1 {
		return 0, 0, ErrCalendarTick
	}
	prod, ok := checked.Mul(days, c.TicksPerDay)
	if !ok {
		return 0, 0, ErrOverflow
	}
	if prod%c.DaysPerTick != 0 {
		return 0, 0, ErrCalendarTick
	}
	ticks = prod / c.DaysPerTick
	if ticks < 1 || ticks > MaxTicksPerYear {
		return 0, 0, ErrOutOfBounds
	}
	return days, ticks, nil
}

// TicksPerYear returns the ticks in a year, DaysPerYear * TicksPerDay /
// DaysPerTick (spec-calendar.md 3.2, FR-CAL-002). It returns 0, never a valid
// year, for a calendar that fails a numeric rule (FR-CAL-022).
func (c Calendar) TicksPerYear() int64 {
	_, ticks, err := c.derive()
	if err != nil {
		return 0
	}
	return ticks
}

// At returns the moment of a tick (spec-calendar.md 4). It is a pure function
// of the calendar and the tick and allocates nothing. A negative tick is
// ErrOutOfBounds and a tick whose day number does not fit in int64 is
// ErrOverflow, each wrapped with the tick; a calendar that fails a numeric
// rule returns the error of the rule (FR-CAL-022). No error comes with a
// moment.
func (c Calendar) At(tick int64) (Moment, error) {
	days, _, err := c.derive()
	if err != nil {
		return Moment{}, err
	}
	if tick < 0 {
		return Moment{}, fmt.Errorf("tick %d: %w", tick, ErrOutOfBounds)
	}
	var (
		d         int64
		tickOfDay int64
		ok        = true
	)
	if c.TicksPerDay > 1 {
		d, ok = checked.Add(c.StartDay, tick/c.TicksPerDay)
		tickOfDay = tick % c.TicksPerDay
	} else {
		var span int64
		if span, ok = checked.Mul(tick, c.DaysPerTick); ok {
			d, ok = checked.Add(c.StartDay, span)
		}
	}
	if !ok {
		return Moment{}, fmt.Errorf("tick %d: %w", tick, ErrOverflow)
	}
	doy := d % days
	m := Moment{
		Tick:      tick,
		Year:      d / days,
		DayOfYear: int(doy),
		DayOfWeek: int(d % c.DaysPerWeek),
		TickOfDay: int(tickOfDay),
		// At most (MaxTicksPerDay - 1) * MaxHoursPerDay: no overflow.
		Hour:   int(tickOfDay * c.HoursPerDay / c.TicksPerDay),
		Season: -1,
	}
	rest := doy
	for i, mo := range c.Months {
		if rest < mo.Days {
			m.Month, m.DayOfMonth = i, int(rest)
			break
		}
		rest -= mo.Days
	}
	if len(c.Seasons) > 0 {
		// Both operands are in [0, days), so the offset is the floor modulus.
		rest = (doy - c.SeasonStart + days) % days
		for i, s := range c.Seasons {
			if rest < s.Days {
				m.Season, m.DayOfSeason = i, int(rest)
				break
			}
			rest -= s.Days
		}
	}
	return m, nil
}

// SeasonAt returns the index of the season of a tick, or -1 with no seasons
// and for a calendar that fails a numeric rule (spec-calendar.md 4.5.1,
// FR-CAL-019, FR-CAL-020, FR-CAL-022). It accepts any int64: the season is that
// of the tick congruent to tick modulo TicksPerYear, the floor modulus taken
// without negating tick so the smallest int64 is safe. It allocates nothing.
func (c Calendar) SeasonAt(tick int64) int {
	_, perYear, err := c.derive()
	if err != nil || len(c.Seasons) == 0 {
		return -1
	}
	r := tick % perYear
	if r < 0 {
		r += perYear
	}
	m, err := c.At(r)
	if err != nil {
		// Unreachable: derive passed and 0 <= r < perYear, so At cannot refuse.
		// Kept because SeasonAt must never panic for a consumer's calendar.
		return -1
	}
	return m.Season
}

// dayTable is the month, day of month and season of every day of the year of
// a calendar (spec-calendar.md 4.6). It is derived, not state: a world builds
// it once at construction and rebuilds it on restore, and it is in neither the
// configuration hash nor a checkpoint.
type dayTable struct {
	month      []int
	dayOfMonth []int64
	season     []int
}

// newDayTable builds the table of a calendar that passed Validate. A calendar
// the derivation refuses is a programmer error, not input, and panics.
func newDayTable(cal Calendar) dayTable {
	days, _, err := cal.derive()
	if err != nil {
		panic("economysim: invariant violated: newDayTable of a calendar that failed validation: " + err.Error())
	}
	t := dayTable{
		month:      make([]int, 0, days),
		dayOfMonth: make([]int64, 0, days),
		season:     make([]int, days),
	}
	for i, m := range cal.Months {
		for d := range m.Days {
			t.month = append(t.month, i)
			t.dayOfMonth = append(t.dayOfMonth, d)
		}
	}
	for i := range t.season {
		t.season[i] = -1
	}
	// The cycle starts at SeasonStart and may wrap past the year end.
	at := cal.SeasonStart
	for i, s := range cal.Seasons {
		for range s.Days {
			t.season[at%days] = i
			at++
		}
	}
	return t
}

// day returns the month index, the zero-based day of the month and the season
// index (-1 with no seasons) of a zero-based day of the year in [0,
// DaysPerYear). It allocates nothing.
func (t dayTable) day(dayOfYear int64) (month int, dayOfMonth int64, season int) {
	return t.month[dayOfYear], t.dayOfMonth[dayOfYear], t.season[dayOfYear]
}
