# Spec: Calendar, tick length and seasons

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CAL-NNN` and
`NFR-CAL-NNN`, which tests cite (testing.md section 4). They are in section 10.

Layer: core, module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
The structure of time (the length of a tick, a year, a month, a week and a day) is
core: every world has a calendar and the default is Earth's. Seasons are optional:
an empty list means none. Needs nothing.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

The simulation steps in **ticks**. A calendar says what a tick, a year and the
parts between them are, so a consumer can model Earth or another planet by
changing configuration only. This spec covers:

- the calendar's fields and their Earth defaults (section 3);
- the quantities derived from them, above all `TicksPerYear` (3.2);
- the mapping from a tick to a **moment**: year, month, day, week, hour and
  season (section 4);
- seasons (section 5);
- what other specs may and may not assume about time (section 6);
- validation (section 7).

Out of scope: what a season does to a recipe's yield (`spec-production.md` 6 holds
the yield profile, `spec-resources.md` the values), the tick phases (`spec-world.md`), names for months, weekdays
and hours (the consumer's concern, as denominations are, `spec-currency.md` 2.2),
and leap years (open question 1).

## 2. Principles

2.1. A calendar is **data**. Nothing in the core branches on a month, a weekday or a
season (architecture.md 9.1). A different planet is a different configuration.

2.2. The tick is the only clock. The simulation never reads wall-clock time
(architecture.md 7.3). A moment is a pure function of the calendar and the tick, so
it needs no stored state and a checkpoint resumes it exactly.

2.3. All arithmetic is integer with a stated rounding rule, truncation toward zero
unless stated (architecture.md 3.5), and overflow is checked.

2.4. Earth is a default, not an opinion. It is a placeholder for the engine in the
sense of `spec-config.md` 5.5; a setting that is not Earth replaces it.

2.5. In the other specs a "day" means a tick (`spec-currency.md` 2.4). In this spec
a **calendar day** is the calendar's unit and a tick may be shorter or longer. The
two are the same only when `TicksPerDay` and `DaysPerTick` are both 1, the default.

## 3. The calendar

### 3.1 Fields

`Calendar` is a section of `Config` (`spec-config.md` 3.1). Months and seasons are
**ordered lists**: their order is data, so they are not tables keyed by ID and are not
sorted (`spec-config.md` 3.4).

| Scope | Field | Meaning | Default |
|-------|-------|---------|---------|
| Calendar | `HoursPerDay` | hours in a calendar day; at least 1 and at most `MaxHoursPerDay` | 24 |
| Calendar | `DaysPerWeek` | calendar days in a week; at least 1 and at most `MaxDaysPerWeek` | 7 |
| Calendar | `Months` | the months of a year in order, each an `ID` and `Days` (calendar days, at least 1 and at most `MaxDaysPerMonth`); at least 1 and at most `MaxMonths` of them | 12 months, `m01` to `m12`, of 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31 days |
| Calendar | `TicksPerDay` | ticks in a calendar day; at least 1 and at most `MaxTicksPerDay` | 1 |
| Calendar | `DaysPerTick` | calendar days in a tick; at least 1 | 1 |
| Calendar | `StartDay` | the zero-based day of the year on which tick 0 falls | 0 |
| Calendar | `Seasons` | the seasons in order, each an `ID` and `Days`; may be empty; at most `MaxSeasons` | 4 seasons: `spring` 92, `summer` 92, `autumn` 91, `winter` 90 |
| Calendar | `SeasonStart` | the zero-based day of the year on which the first season begins | 59 |

A tick is either a part of a day (`TicksPerDay` above 1) or a number of whole days
(`DaysPerTick` above 1), never both: with both above 1, the day of a tick would
depend on a fraction that has no meaning. The default is one tick a day.

The Earth defaults are 365 days, a 7-day week, 24 hours and seasons that begin on 1
March (day 59 of a year that begins on 1 January: 31 + 28), so winter wraps across
the new year. There is no leap day (open question 1). Month IDs are neutral
(`m01`); a consumer attaches names.

### 3.2 Derived quantities

These are computed, not configured:

```
DaysPerYear  = sum of the days of the months
TicksPerYear = DaysPerYear * TicksPerDay / DaysPerTick       (must divide exactly)
```

**`TicksPerYear` is no longer a world parameter.** `spec-currency.md` 2.4 defined it
with a default of 365; it is now derived, and 365 is what the Earth default gives.
Every formula that uses `TicksPerYear` (`spec-welfare-and-wages.md` sections 3 and
5) is unchanged, and so are the numbers in their worked examples at the default.

The division is exact or the calendar is rejected (7.2): if a year were not a whole
number of ticks, the year in the wage formulas would not be a whole number either.
Earth with a 7-day tick (52.14 ticks) is therefore not allowed; a 5-day or a 73-day
tick is (open question 2).

## 4. From a tick to a moment

`Calendar.At(tick)` returns a **moment**, a plain value. It is a pure function:
equal calendar and tick always give an equal moment.

```go
type Moment struct {
	Tick        int64
	Year        int64 // 0 for the year in which tick 0 falls
	DayOfYear   int   // zero-based
	Month       int   // index into Months
	DayOfMonth  int   // zero-based
	DayOfWeek   int   // zero-based
	TickOfDay   int   // zero-based; 0 unless TicksPerDay is above 1
	Hour        int   // zero-based
	Season      int   // index into Seasons, or -1 if there are none
	DayOfSeason int   // zero-based; 0 if there are no seasons
}
```

4.1. The **absolute day** `d` of a tick `t` is:

- `StartDay + t / TicksPerDay` (integer division) when `TicksPerDay` is above 1, with
  `TickOfDay = t % TicksPerDay`;
- `StartDay + t * DaysPerTick` otherwise, with `TickOfDay = 0`.

4.2. `Year = d / DaysPerYear` and `DayOfYear = d % DaysPerYear`. `Month` and
`DayOfMonth` are found by walking the months' days in order.

4.3. `DayOfWeek = d % DaysPerWeek`. The week does not restart with the year, and the
weekday of absolute day 0 is 0. The calendar needs no weekday offset: a setting that
wants another first weekday shifts `StartDay` or its own labels.

4.4. `Hour = TickOfDay * HoursPerDay / TicksPerDay`, truncated. It is exact when
`TicksPerDay` divides `HoursPerDay` (a warning otherwise, 7.3) and 0 for a tick of
whole days. Hours exist for the consumer's clock and for rules that depend on the
time of day; nothing in the core does yet.

4.5. A negative tick, or a tick for which `t * DaysPerTick` or `StartDay + d` would
overflow `int64`, is rejected with an error and no moment. The overflow is
unreachable in a real run and is checked anyway (architecture.md 3.5).

4.5.1. **The periodic extension.** A warm start places jobs that began before tick 0,
and reads the yield by season at the virtual start tick `-o`
(`spec-viability.md` 8.5.2). For that, `Calendar.SeasonAt(tick)` accepts any `int64`,
negative included, and returns the season index (or -1 with no seasons) of the tick
congruent to `tick` modulo `TicksPerYear` in `[0, TicksPerYear)`:
`At(floorMod(tick, TicksPerYear)).Season`, with `floorMod` non-negative and computed
without negating `tick`, so the smallest `int64` is safe. The year-level fields of
the calendar are periodic in `TicksPerYear` ticks (FR-CAL-006), so this is the natural
backward extension; the season of tick `-o` is that of `(-o) mod TicksPerYear`. It is
a separate helper because `At` rejects a negative tick (FR-CAL-010) and returns a
whole moment; `SeasonAt` returns the season only, since `DayOfWeek` does not repeat
with the year (FR-CAL-007) and `Year` has no use before the start. It is derived, uses
the table of 4.6, allocates nothing and adds no state.

4.6. `At` allocates nothing and does not walk more than the months and seasons of a
year. A world precomputes a table of month and season by day of year at
construction. The table is derived from the calendar, not state, and is rebuilt on
restore.

## 5. Seasons

5.1. The seasons are one **cycle** that partitions the year. Season `i` begins
`SeasonStart` plus the days of seasons `0` to `i-1` into the year, counted modulo
`DaysPerYear`, and lasts its `Days`. The days must add up to `DaysPerYear`
exactly, so every day belongs to one season. A cycle may wrap past the year end,
which is why the Earth default can start in March and still cover a year that
starts in January.

5.2. With no seasons, `Season` is -1 and a year is one undivided period. A setting
with no seasons loses nothing; every rule that depends on a season treats -1 as
"all year".

5.3. The cycle is global: one list for the world, not one for each region. A region
differs through what it does in each season, not through its own cycle. The
five-region world of the first consumer needs different climates, and that is a
table of yields by region and season (`spec-resources.md`, N-030):

- a region whose seasons are the reverse of another's (the opposite hemisphere)
  gives its yields to the same seasons the other way round;
- a region with two seasons (wet and dry) where the world has four gives the same
  yield to two of them each;
- a world whose regions need seasons that fit none of these defines a finer cycle.

A cycle that is shared and fine-grained is simpler and cheaper than a cycle per
region, and it keeps `At` a function of the tick alone. A per-region shift in days is
left out (open question 3).

5.4. A season has an `ID` and nothing else; what a season changes is owned by the
specs that use it. Season IDs are neutral by the same rule as every other ID
(`spec-config.md` 3.3). The Earth names `spring` to `winter` are the one default
that carries setting vocabulary, because Earth is the stated default (2.4); a pack
that is not Earth replaces them.

## 6. What other specs may assume

6.1. Durations in the configuration are in **ticks**: `ShelfLife`, `TrainingTicks`,
`ExpectedWorkTicks`, a welfare window. Changing the calendar does not rescale
them. A pack for another planet states them in the ticks of its own calendar. The
file format takes integers only (`spec-config.md` 2.6), so there is no "two weeks".

6.2. Quantities stated per tick (a household's upkeep, the reference basket `B_r`, the
speed of a transport mode in distance units per tick, `spec-transport.md` 3.1, and a money flow stated
`Per` tick, `spec-money-supply.md` 4.2)
are data for the chosen tick length and are not rescaled (`spec-currency.md` 2.4). A quantity stated per
year follows `TicksPerYear`: a money flow with `Per` `year` and the annual birth and death rates of
`spec-population.md` 3.8 are converted exactly from it.
This is the reason `TicksPerDay` and `DaysPerTick` change results and are hashed.

6.3. `ExpectedWorkTicks` must be at most `TicksPerYear`
(`spec-welfare-and-wages.md` 4.2). With the role default of 250, a calendar with a
year of fewer than 250 ticks fails validation until the role sets its own value. This
is deliberate (`spec-config.md` 5.2, open question 5). It is an error, not a warning
(FR-CFG-048).

6.4. Calendar fields all affect results, directly (the length of a year in the wage
formulas) or through seasons, so they are all in the configuration hash. None is
presentation-only.

6.5. The calendar adds no state to a world or to a checkpoint. The tick is already
state, and a moment is derived from it (2.2).

## 7. Validation

### 7.1 Bounds

The bounds are constants in `bounds.go` (`spec-config.md` 4.1). Provisional until
the slice runs.

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxHoursPerDay` | 1 000 | a day of many hours is legal; the bound keeps `Hour` arithmetic small |
| `MaxDaysPerWeek` | 100 | |
| `MaxMonths` | 100 | validation and the day-of-year table are bounded by the configuration |
| `MaxDaysPerMonth` | 10 000 | |
| `MaxDaysPerYear` | 100 000 | the sum of the months; the table of days has this many entries |
| `MaxSeasons` | 100 | |
| `MaxTicksPerDay` | 1 000 | |

`TicksPerYear` is also bounded by `MaxTicksPerYear` (`spec-config.md` 4.1).

### 7.2 Rules

Each has a sentinel and a failing-case test (testing.md 3.9). Errors carry the path,
for example `calendar.months[3].days`.

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field outside its bound in 7.1, `TicksPerYear` outside `[1, MaxTicksPerYear]`, `StartDay` outside `[0, DaysPerYear)`, `SeasonStart` outside `[0, DaysPerYear)` when there are seasons |
| `ErrInvalidID`, `ErrDuplicateID` | a month or season ID that breaks `spec-config.md` 3.3, or that repeats within its list |
| `ErrCalendarTick` | both `TicksPerDay` and `DaysPerTick` above 1, or `DaysPerYear * TicksPerDay` not divisible by `DaysPerTick` |
| `ErrCalendarSeasons` | seasons whose days do not add up to `DaysPerYear`, or a season of fewer than 1 day |
| `ErrOverflow` | `DaysPerYear * TicksPerDay` or `MaxDaysPerYear` products that do not fit (`spec-config.md` 4.2) |

A file that sets the months but not the seasons keeps the Earth seasons, and fails
`ErrCalendarSeasons` unless the lengths happen to match. This is intended: the author
restates both. `seasons = []` says "none".

### 7.3 Warnings

A warning never fails validation (`spec-config.md` 7.2):

- `inexact_hour`: `TicksPerDay` does not divide `HoursPerDay`, so `Hour` is
  approximate;
- `season_shorter_than_tick`: `DaysPerTick` is above 1 and a season is shorter than
  a tick, so some ticks skip the season entirely.

The names are those of `spec-config.md` 7.4.2.

### 7.4 Examples

Earth, which is the default, written out:

```toml
[calendar]
hours_per_day = 24
days_per_week = 7
months = [
  { id = "m01", days = 31 }, { id = "m02", days = 28 }, { id = "m03", days = 31 },
  { id = "m04", days = 30 }, { id = "m05", days = 31 }, { id = "m06", days = 30 },
  { id = "m07", days = 31 }, { id = "m08", days = 31 }, { id = "m09", days = 30 },
  { id = "m10", days = 31 }, { id = "m11", days = 30 }, { id = "m12", days = 31 },
]
seasons = [
  { id = "spring", days = 92 }, { id = "summer", days = 92 },
  { id = "autumn", days = 91 }, { id = "winter", days = 90 },
]
season_start = 59
```

A planet with a 320-day year, a 20-hour day, a 5-day week and two ticks a day:
`TicksPerYear` is 320 * 2 / 1 = 640, and the seasons add up (120 + 200 = 320):

```toml
[calendar]
hours_per_day = 20
days_per_week = 5
ticks_per_day = 2
months = [
  { id = "m1", days = 40 }, { id = "m2", days = 40 }, { id = "m3", days = 40 },
  { id = "m4", days = 40 }, { id = "m5", days = 40 }, { id = "m6", days = 40 },
  { id = "m7", days = 40 }, { id = "m8", days = 40 },
]
seasons = [ { id = "wet", days = 120 }, { id = "dry", days = 200 } ]
season_start = 0
```

Here `Hour` is exact (20 hours over 2 ticks is 10 hours a tick). Tick 1 is absolute
day 0, tick of day 1, hour 10. Tick 640 is the first tick of year 1.

## 8. Observability

8.1. This spec adds no event, log line or metric: a moment is a pure function of a
tick (2.2), and every event already carries its `Tick`. A consumer converts with
`Calendar.At`.

8.2. Whether the world emits an event when a season begins is open (open question
4). A game that wants "winter has come" can derive it from the tick today.

## 9. Stability register entries

Merged into `spec-stability.md` section 4: seasons as a periodic forcing is STB-L23
and a coarse tick hiding a season is STB-L24; "a season shorter than a delay" is a
note on STB-L23.

For `spec-stability.md`:

- **Seasons are a periodic forcing.** Seasonal yields make supply swing with a
  fixed period, which excites any loop with a delay close to the period (production
  and storage delays, `spec-goods.md` 4.5 whole-lot spoilage). The reference chain
  needs a seasonal scenario in the acceptance bands of `spec-stylised-facts.md`.
- **A season shorter than a delay.** Does not arise for production: the season and
  the yield are read when a job starts, so a recipe longer than its season completes
  (`spec-production.md` 6.4; N-031). It would arise only for a mechanism that reads the
  season at completion, which production defers (`spec-production.md` 6.7); a harvest
  window is a perishable intermediate. Storage across a season is checked by V4
  (`spec-viability.md` 4.5).
- **A coarse tick hides a season.** With `DaysPerTick` above 1, a short season may
  not be sampled (7.3).

## 10. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CAL-001 | `Calendar` MUST be a plain struct in the root package, and its default MUST be Earth's: 24 hours a day, a 7-day week, 12 months of 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31 days, one tick a day, `StartDay` 0, seasons of 92, 92, 91 and 90 days beginning on day 59. |
| FR-CAL-002 | `DaysPerYear` MUST be the sum of the months' days and `TicksPerYear` MUST be `DaysPerYear * TicksPerDay / DaysPerTick`, and the world MUST have no separately configured `TicksPerYear`. |
| FR-CAL-003 | With the default calendar `TicksPerYear` MUST be 365. |
| FR-CAL-004 | `At` MUST be a pure function: equal calendar and tick MUST give an equal moment, and it MUST NOT read wall-clock time. |
| FR-CAL-005 | The absolute day of a tick MUST be `StartDay + t / TicksPerDay` when `TicksPerDay` is above 1 and `StartDay + t * DaysPerTick` otherwise, and `TickOfDay` MUST be `t % TicksPerDay` or 0 respectively. |
| FR-CAL-006 | `TicksPerYear` consecutive ticks MUST advance the year by exactly 1 and return to the same `DayOfYear`, `Month`, `DayOfMonth` and `Season`. |
| FR-CAL-007 | `DayOfWeek` MUST be the absolute day modulo `DaysPerWeek` and MUST NOT restart at a year boundary. |
| FR-CAL-008 | `Hour` MUST be `TickOfDay * HoursPerDay / TicksPerDay`, truncated toward zero. |
| FR-CAL-009 | Every day of the year MUST belong to exactly one season when seasons are configured, with the first season beginning on `SeasonStart` and the cycle wrapping past the year end; `Season` MUST be -1 when none are configured. |
| FR-CAL-010 | A negative tick, or one whose day number would overflow `int64`, MUST be rejected with an error and no moment. |
| FR-CAL-011 | A calendar with both `TicksPerDay` and `DaysPerTick` above 1, or whose `DaysPerYear * TicksPerDay` is not divisible by `DaysPerTick`, MUST be rejected with `ErrCalendarTick`. |
| FR-CAL-012 | Seasons whose days do not add up to `DaysPerYear`, or a season of fewer than 1 day, MUST be rejected with `ErrCalendarSeasons`. |
| FR-CAL-013 | A field outside its bound, `TicksPerYear` outside `[1, MaxTicksPerYear]`, or a `StartDay` or `SeasonStart` outside the year, MUST be rejected with `ErrOutOfBounds` carrying its path. |
| FR-CAL-014 | A month or season ID that is invalid or repeated within its list MUST be rejected with `ErrInvalidID` or `ErrDuplicateID`. |
| FR-CAL-015 | Months and seasons MUST keep their written order through loading, canonicalisation and the configuration hash, and every calendar field MUST change the hash. |
| FR-CAL-016 | A calendar with `TicksPerDay` not dividing `HoursPerDay`, or with `DaysPerTick` above 1 and a season shorter than a tick, MUST produce a warning and MUST NOT fail validation. |
| FR-CAL-017 | Changing the calendar MUST NOT rescale any quantity stated per tick or any duration stated in ticks. |
| FR-CAL-018 | The calendar MUST add no state to a world or to a checkpoint, and a restored world MUST give the same moment for the same tick. |
| FR-CAL-019 | `SeasonAt` MUST accept any `int64` tick, MUST return -1 when no seasons are configured, and MUST equal `At(tick).Season` for every tick that `At` accepts. |
| FR-CAL-020 | `SeasonAt(tick)` MUST equal `SeasonAt(tick - TicksPerYear)` for every tick, negative included, and MUST NOT overflow or panic for the smallest and largest `int64`. |
| FR-CAL-021 | The warnings of 7.3 MUST be named `inexact_hour` and `season_shorter_than_tick`. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-CAL-001 | `At` MUST NOT allocate. |
| NFR-CAL-003 | `SeasonAt` MUST NOT allocate and MUST be covered by the fuzz test of NFR-CAL-002 (never panics, result in range, periodic). |
| NFR-CAL-002 | The calendar validator and `At` MUST have fuzz tests (testing.md 7.4) asserting that they never panic and that, for every valid calendar and tick, the moment's fields are within their ranges. |

## 11. Open questions

1. **Leap years and irregular years.** Version 1 has a fixed year, so Earth's 365
   is a simplification. A leap rule would make the year length vary, and
   `TicksPerYear` would become a mean, which the wage formulas do not expect. If a
   setting needs it, add an optional rule that is a fixed cycle of years (for
   example 4, with one longer), and keep `TicksPerYear` as the length of the
   shortest or the mean. Recommend leaving it out until a consumer asks.
2. **A tick that does not divide the year.** A weekly tick on a 365-day year is
   rejected (3.2). The alternative is a short final tick of the year, which gives
   the wage formulas a year whose last period is not a whole tick. Recommend
   keeping the rejection and choosing a tick that divides the year.
3. **A season shift per region.** A shift in days would let regions share a cycle
   but start it at different dates (a hemisphere). The table of yields by season
   covers the cases known (5.3), and a shift makes the season a function of the
   region as well as the tick. Add it only if the first consumer's climates need it.
4. **A season event.** Whether the world emits `SeasonStarted` when the season
   changes (8.2). It is derivable, so it is a convenience for a game loop. Decide in
   `spec-observability.md`.
5. **Labels.** Names for months, weekdays and seasons are a consumer concern. If a
   formatter is wanted, it follows the denomination formatter of
   `spec-currency.md` section 4: optional, outside the core and not hashed.
6. **Durations in calendar units.** Whether a pack may state "two weeks" for a
   duration. Recommend not: integers in ticks only, and a converter in the pack
   tooling if it proves useful.

## 12. Where this lives in the code

| Concern | Home |
|---------|------|
| `Calendar`, `Month`, `Season`, `Moment`, `At`, `SeasonAt` | root package, `calendar.go` |
| `DefaultCalendar()` | root package, `defaults.go` |
| The bounds of 7.1 | root package, `bounds.go` |
| Validation and the sentinels | root package, `validate.go`, `errors.go` |
| The day-of-year table | derived at `NewWorld`, held by the world, not in the configuration or the checkpoint |
