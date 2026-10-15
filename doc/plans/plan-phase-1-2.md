# Plan: Stage 1, row 2: the goods, recipe, role and calendar tables

**Status: Built (2026-10-14), all six phases; uncommitted.** Sequences the second row
of Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`commodity.go`, `recipe.go`,
`role.go`, `calendar.go` (the root types and their derived tables)", into six
phases. It implements [spec-goods.md](../specs/spec-goods.md) (GDS),
[spec-production.md](../specs/spec-production.md) section 3 (PRO),
[spec-agents.md](../specs/spec-agents.md) section 3, with the effective upkeep
of 5.2 that the tradable set reads (AGT), and
[spec-calendar.md](../specs/spec-calendar.md) (CAL) for the slice. It is built
against `doc/rules/architecture.md` (the layout of 2.6, the `internal/`
boundary of 2.9, the module-file rule of 2.10, the fixed-point and overflow
rule of 3.5, the determinism rules of section 7), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR
or the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md)
(D3 and D5 above all). Where a decision is still owed, Phase 1 makes it in the
document that owns it, before the code that depends on it, and where it is not
this row's to make it is an open item.

No PRD exists for this row. The requirements source is the roadmap row and the
four specs above. The row is built after the ledger row
([plan-phase-1-1.md](plan-phase-1-1.md), in progress at the time of writing)
and shares no code with it: the two touch disjoint files, and the ledger's
`ledger.go` reads `Commodity.ShelfLife` and `Commodity.Durability` from
`config.go` directly (A8 there), so neither waits for the other.

Spec codes: GDS goods, PRO production, AGT agents, CAL calendar, CFG config,
VIA viability, WLD world, BEL beliefs, CLR clearing, DEM demand, SUB
subsistence, TRN transport, STO storage, REG regions, IND indicators, RES
resources, SET settlements. `FR-`/`NFR-` IDs are the requirement IDs of those
specs.

## What Stage 0 already built, and what is left

The row's wording ("the root types") is older than the code. Stage 0 is done
(commit `023cf97`), and it built:

| Concern | Where it already is |
|---------|---------------------|
| `Commodity`, `Item`, `Output`, `Tool`, `Recipe`, `Role`, `Calendar`, `Month`, `Season` | `config.go` (architecture.md 2.6 homes the section structs there; the `config.go` deliverable of plan 0, Phase 2) |
| `DefaultCommodity`, `DefaultRecipe`, `DefaultRole`, `DefaultCalendar` | `defaults.go` |
| `CommodityID`, `RecipeID`, `RoleID`, `Price`, `Quantity` | `value.go` |
| The bounds of GDS 8, PRO 10.1, AGT 12 and CAL 7.1 | `bounds.go` |
| The validation rules and warnings of GDS 8, 6.4, 7.2, PRO 10.2 to 10.4, AGT 13 and CAL 7, including the derived `TicksPerYear` (`calendarFacts`), the tradable sets (`tradableSets`) and the effective upkeep (`effectiveUpkeep`) | `validate.go` |
| The canonical order and the typed-ID resolvers `commodityID`, `recipeID`, `roleID`, `regionID` | `config.go` (`canonical`, binary search over the sorted slices) |
| `ErrCalendarTick`, `ErrCalendarSeasons`, `ErrOutOfBounds`, `ErrOverflow`, `ErrUnknown*` | `errors.go` |

What is left for this row is what the specs name as behaviour of those types and
as derived data that a world needs and that is not state:

- **Calendar** (CAL 3.2, 4, 5): `Moment`, `At`, `SeasonAt`, the derived year,
  and the day-of-year table of 4.6. None exists. `SeasonAt` is also the first
  work of `viability.go` (the deferred `ErrWarmStartJob` sub-rule of plan 0's
  rule ledger and FR-VIA-053).
- **Goods** (GDS 3): the load pair, the chargeable size and the chargeable
  capacity (FR-GDS-005 to 007).
- **Recipes** (PRO 3): the recipe in typed-ID form, which is what production,
  the default strategy and the viability solver read in the tick (a string key
  and a map lookup per item would be the alternative), and the index of the
  recipes that make a commodity (PRO 9.3, the cost anchor).
- **Roles** (AGT 3.3, 3.4, 5.2): the role in typed-ID form, its tradable set,
  the commodities that are tradable in the world, and its effective upkeep per
  region.

Everything else of the four specs is either built (validation), another row's
(the yield table, `UnitCost` and jobs are `production.go`; the strategy is
`strategy.go`; beliefs are `beliefs.go`) or a module's (the want items, the own
use outputs, the settlement flags and the other fields appended to `Role`).

## Scope

**In scope.**

- `calendar.go`: `Moment`, `Calendar.At`, `Calendar.SeasonAt`,
  `Calendar.TicksPerYear`, the unexported derivation of `DaysPerYear` and
  `TicksPerYear` with checked arithmetic, and the unexported day-of-year table
  (CAL 3.2, 4.1 to 4.6, 5.1, 5.2).
- `commodity.go`: the load of a set of goods, whether it fits a container, the
  chargeable size of a unit and the chargeable capacity of a container
  (GDS 3.2 to 3.4), unexported, with the checked arithmetic of 3.5 of the
  rules.
- `recipe.go`: the recipe table (items, outputs and tools by typed ID, the
  source flag) and the index of the makers of a commodity (PRO 3.1.1, 3.1.2,
  3.2.2, 9.3).
- `role.go`: the role table (recipes, trades, base upkeep, tradable set, and the
  effective upkeep per region) and the per-commodity "tradable in some role"
  flag (AGT 3.3, 3.4, 5.2; GDS 6.4).
- The glossary terms these files use, in `doc.go`, before they use them
  (architecture.md 3.2).
- The spec, note and roadmap edits these files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| Holding the tables: `World`, `NewWorld` step 5, the readers `Calendar()` and `ReferencePrice()`, the moment computed once at P1.1 | Roadmap Stage 1, `world.go` row. This plan gives step 5 the constructors it calls |
| The yield table and its lookup (PRO 6), `UnitCost` and the cost anchor (PRO 9), the execution count, jobs, residues | Roadmap Stage 1, `internal/production` and `production.go` row. This plan gives it the recipe table and the makers index |
| Agents, views, the budget stack and `Fed` (AGT 4 to 7), `AgentView`, strategies | Roadmap Stage 1, `agent.go` and `strategy.go` row |
| `internal/household` of AGT 22: it is not in architecture.md 2.6 | Open item O2: the agent row decides, with 2.1 and 2.6 |
| The belief and the seed price per commodity in a tradable set (BEL 3.1, 8.1) | Roadmap Stage 1, `internal/beliefs` and `beliefs.go` row |
| The `not_tradable`, `not_on_role` and `no_room` refusals (AGT 6.5, CLR 3.2) | Clearing and production rows (plan 0's "sentinel and reason with its first user") |
| The effective reference price with a region override, `RegionPrices` (FR-GDS-020, GDS 6.3) | Roadmap Stage 2, `regions.go` row: the field is Stage 2 (field ids 8 and 9 reserved, N-218) |
| The tradable items of a module: the wants of the demand module (AGT 3.4, DEM 3.2.3), the own-use outputs (SUB 3.6) | Each module's own PR; the touch point they need is open item O1 (note N-228) |
| Spoilage, wear and ageing of lots; shelf life and durability as ledger input (FR-GDS-009 to 015) | `internal/ledger`, [plan-phase-1-1.md](plan-phase-1-1.md) |
| Class aggregators (GDS 6.1, IND) | Roadmap Stage 2, `indicator/` row. No code names a class (FR-GDS-017) |
| The season of a yield, the warm start's virtual start offsets, the viability rules | `viability.go` and `production.go` rows, which call `SeasonAt` |
| Marking a module's field on `Role` (`Informed`, `Fixed`, `Merchant`, `Hireable`, `Needs`, `Ceiling`, ...) | Each module's PR, appended (architecture.md 2.10.5, D3) |
| A `World.Calendar` consumer format, labels for months and weekdays (CAL open questions 5, 6) | Not built; a consumer concern |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- The section structs live in `config.go`, their defaults in `defaults.go` and
  the typed IDs in `value.go` (architecture.md 2.6; the `config.go` deliverable
  of plan 0, Phase 2; D3 of plan 0 fixes which fields they declare). The four
  files of this row therefore hold behaviour and derived tables, not the
  structs. GDS 12, PRO 19 and CAL 12 still say the structs are in
  `commodity.go`, `recipe.go` and `calendar.go`, and AGT 22 lists `Role` under
  `agent.go`; Phase 1 corrects that text (for AGT 22 the correction is that
  `Role` is in `config.go`, not that it moves to `role.go`).
- A typed ID is the index of the entity in the canonical slice of `Config`
  (`spec-config.md` 3.3, `config.go`). A table built from a canonical
  configuration is indexed by the typed ID, and "the order" of a list in a
  table is the canonical order (ascending ID), so no sort in this row can
  produce a different result on another run (architecture.md 7.5, 7.6).
- Commodities, recipes, roles, classes and seasons are data. No code branches
  on one (architecture.md 9.1; FR-GDS-017, FR-PRO-001, FR-PRO-032, AGT 3.3.2).
- A table is derived, not state: it is not in the configuration hash and not in
  a checkpoint, and a restore rebuilds it (CAL 6.5, FR-CAL-018; WLD 12.2).
  Nothing here changes `testdata/golden/off/` or the pinned hash (D5).
- All arithmetic that can overflow goes through `internal/checked`
  (architecture.md 3.5): the absolute day, the loads, the merged upkeep.
- A violated internal invariant found after validation panics with a message
  that says so (architecture.md 8.1; WLD 5.1.1): the table builders take a
  configuration that passed `Validate`, as `NewWorld` does.
- The public methods of `Calendar` are callable by a consumer with a hand-built
  value, so they return an error and do not panic or divide by zero for a
  calendar that fails the rules of CAL 7.2 (architecture.md 8.1, 6.3).

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The structs stay in `config.go`. The four files hold methods,
  unexported table types and their constructors. A spec line, not code, is
  what changes where GDS 12, PRO 19, AGT 22 and CAL 12 say the structs live
  (Phase 1).
- **A2.** A table is an unexported type built by one constructor from a
  `*Config` that is canonical and valid (`newRecipeTable`, `newRoleTable`,
  `newDayTable`), and the world of the `world.go` row holds the results and
  hands them to the rules. The constructors allocate, once, at `NewWorld`;
  every read of a table allocates nothing (NFR-GDS-001). No table has an
  exported type, so nothing here is a compatibility promise except A3.
- **A3.** The exported surface added is `Moment` and three methods of
  `Calendar`: `At`, `SeasonAt` and `TicksPerYear` (`spec-indicators.md` reads
  `TicksPerYear` and `At` from `World.Calendar()`). `DaysPerYear` stays
  unexported, because no spec of the slice or version 1 reads it from outside
  (architecture.md 2.8). Signatures: `At(tick int64) (Moment, error)`,
  `SeasonAt(tick int64) int`, `TicksPerYear() int64`. For a calendar that
  fails the numeric rules, `At` returns the error of the rule,
  `TicksPerYear` returns 0 (never a valid year) and `SeasonAt` returns -1.
  The numeric rules are the rules of CAL 7.2 that carry a number (the four
  sentinels `ErrOutOfBounds`, `ErrCalendarTick`, `ErrCalendarSeasons` and
  `ErrOverflow`) and the two count
  bounds of `spec-config.md` 4.1 that `Validate` checks in its first phase
  (`len(Months) <= MaxMonths`, `len(Seasons) <= MaxSeasons`, `ErrTooMany`,
  `validate.go` `counts`). The counts are checked first, so no walk is longer
  than `MaxMonths` plus `MaxSeasons` entries whatever a consumer hands in. The
  ID rules (`ErrInvalidID`, `ErrDuplicateID`, FR-CAL-014) are not numeric: no
  number depends on an ID, so the derivation does not apply them and the
  validator alone does. `SeasonAt` returning -1 is also the answer for a
  calendar with no seasons (FR-CAL-019), so a caller that must tell the two
  apart reads `TicksPerYear() == 0` or the error of `At`. Phase 1 writes this
  into CAL, with a new requirement FR-CAL-022 for the three behaviours.
- **A4.** The refusals of `At` (FR-CAL-010) reuse the existing sentinels:
  `ErrOutOfBounds` for a negative tick and `ErrOverflow` for a day number that
  does not fit, wrapped with the tick (`%w`, architecture.md 8.2). No new
  public sentinel is added. Phase 1 writes the names into CAL 4.5 and
  FR-CAL-010.
- **A5.** `At` computes by walking the months and seasons (at most
  `MaxMonths` and `MaxSeasons`, 100 each, after the count check of A3), because
  `Calendar` is a plain public struct, plain data hashed field by field, and
  carries no cache (FR-CAL-001). The day-of-year table of CAL 4.6 is the world's:
  `newDayTable` builds it once from a calendar and a `Guard:` test asserts it
  gives the same month and season as `At` for every day of the year.
- **A6.** The tradable set and the effective upkeep are not re-implemented.
  `newRoleTable(c *Config)` builds `lk := newLookup(c)` (the `lookup` of
  `validate.go`, which `tradableSets(c, lk)` needs to resolve a role's recipe
  list) and calls the existing `tradableSets` and `effectiveUpkeep` of
  `validate.go`, and converts their string keys to IDs through
  `Config.commodityID`, `recipeID` and `regionID`, so AGT 3.4 has one
  implementation and the validator and the world cannot disagree. It reads
  nothing of the recipe table, so the role table does not take it as a
  parameter. Maps are used only at build time and are turned into ascending
  slices before they are stored (architecture.md 7.5).
- **A7.** The load, the chargeable size and the chargeable capacity are built
  now although no module of version 1 prices by the size (GDS 3.3, 3.5):
  FR-GDS-005 to 007 are MUST requirements of this row's spec. The load pair
  and `fits` are what TRN 3.2 and STO 3.4 test against; the chargeable size
  and the chargeable capacity are read by no module of version 1 (GDS 3.3,
  3.5; STO 8.4: transport and storage test the (mass, bulk) pair themselves)
  and exist only for FR-GDS-006 and 007. They are unexported, small and
  tested. If the owner prefers to defer them to the transport and storage
  rows, Phase 3 is dropped and the three requirements move there with a note.
- **A8.** Tests of the tables run from an internal test (package
  `economysim`) with small configuration literals whose expectations are
  computed by hand (testing.md 5.5), and from one external test that loads the
  packs `one` and `second` through `config/` and reaches the unexported
  constructors through a test-only `export_test.go`, so the two settings of
  architecture.md 9.4 both pass. The loader's `Config` is not canonical (only
  `hash.go` calls `canonical()`), so the external test passes each pack through
  `canonical()`, exposed by `export_test.go`, before `newRecipeTable` and
  `newRoleTable`. Plans 1-2, 1-3, 1-5 and 1-6 each need `export_test.go`: the
  first to land creates it, the others append, and none replaces it. The file is `_test.go`, so nothing is exported
  from production code (architecture.md 10.2).
- **A2a.** The row structs of the Provides table are append-only for later
  rows: a field a later row needs is appended to the struct and filled in the
  constructor by that row's PR (plan 1-6 Phase 4 appends `window int64` to
  `roleRow`, one `effectiveWindow(c, lk, r)` call in `newRoleTable`). Fields
  and signatures of this table are not otherwise changed.
- **A9.** One PR per phase, in phase order, on a branch from `stage-1-1`.
  Phases 2 to 5 depend only on Phase 1 (docs) and on each other as stated in
  each "Why here", and may be built in parallel by different authors. Phase 6
  is last. The row shares `doc.go`, `doc/notes.md` and the roadmap with the
  ledger row: each PR appends its glossary lines and re-reads the shared file
  before editing it.
- **A10.** The reference price of a commodity is read from the world's
  canonical copy of the configuration by `CommodityID`
  (`cfg.Commodities[id].ReferencePrice`). No function wraps that read until
  the region override exists (Stage 2).

## Cross-plan contract

The exact names the siblings consume. They are unexported, inside package
`economysim`, so a sibling row that builds in the same package calls them by
these names; a rename in review changes this table and the "ASSUMED" cells of
the sibling plans in the same change. Row types are small plain structs, built
once at `NewWorld` and read by position (A2), never copied by a reader.

### Provides

| Name and signature | What it is | Used by |
|---|---|---|
| `type itemRow struct{ commodity CommodityID; quantity Quantity }` | one `(commodity, quantity)` pair, the row type of inputs and of an upkeep list | plans 1-5 (Phases 4, 5, 7), 1-6, the agent row |
| `type recipeRow struct{ inputs []itemRow; outputs []outputRow; tools []toolRow; source bool }` with `type outputRow struct{ commodity CommodityID; quantity Quantity; costShare int64; discard bool }` and `type toolRow struct{ commodity CommodityID; wearPerUse int64 }` | one recipe, every list in canonical commodity order; `source` is "no inputs" (PRO 3.1.2) | plans 1-5 (Phases 4, 5, 7: executions, wear, the cost anchor), 1-3 (none: it takes the facts it needs as arguments), the agent row (the recipe choice), the viability row |
| `type recipeTable struct{ rows []recipeRow; byOutput [][]RecipeID }`; `func newRecipeTable(c *Config) recipeTable`; `func (t *recipeTable) row(id RecipeID) *recipeRow`; `func (t *recipeTable) makers(id CommodityID) []RecipeID` | the rows are indexed by `RecipeID`; `makers` is the recipes with the commodity among their non-discard outputs, ascending `RecipeID` (PRO 9.3); a commodity no recipe makes has an empty list | plan 1-5 Phases 4, 5 and 7 (the cost anchor reads `makers`), the agent row, the viability row |
| `type roleRow struct{ recipes []RecipeID; trades []CommodityID; upkeep []itemRow; tradable []CommodityID; member []bool; regionUpkeep []regionUpkeep }` with `type regionUpkeep struct{ region RegionID; items []itemRow }` | one role: `recipes` ascending `RecipeID` (empty is legal); `trades` ascending; `upkeep` is the base list of AGT 5.2 (the role's, or the world's when the role's is empty); `tradable` ascending and unique; `member[c]` is the membership flag by `CommodityID`, `len(member)` the commodity count; `regionUpkeep` holds a row only for a region with an `ExtraUpkeep`, ascending `RegionID`, the base list merged with it, quantities added with checked arithmetic, per household (the weight multiplication is the agent's and the world's) | plans 1-5 (the role's recipe list), 1-6 (the "has recipes" test is `len(recipes) > 0`; the effective upkeep per region), 1-3 (the membership flag, which the caller passes as a `bool`), the agent row (views, targets), the spawner row |
| `type roleTable struct{ rows []roleRow; anyTradable []bool }`; `func newRoleTable(c *Config) roleTable`; `func (t *roleTable) row(id RoleID) *roleRow`; `func (r *roleRow) isTradable(c CommodityID) bool`; `func (r *roleRow) upkeepIn(g RegionID) []itemRow`; `func (t *roleTable) tradableSomewhere(c CommodityID) bool` | `rows` indexed by `RoleID`; `upkeepIn` returns the merged row of the region when it has one and `upkeep` otherwise (a binary search, no allocation); `tradableSomewhere` is the union of the sets (AGT 3.4, GDS 6.4) | plans 1-6 (`upkeepIn`), 1-3 (`isTradable`, as a fact the caller resolves), the agent row, the world row |
| `type dayTable struct{ month []int; dayOfMonth []int64; season []int }`; `func newDayTable(cal Calendar) dayTable`; `func (t dayTable) day(dayOfYear int64) (month int, dayOfMonth int64, season int)` | the day-of-year table of CAL 4.6, built once from a valid calendar, read by position, allocation-free; `season` is -1 with no seasons | the world row (the moment is computed once at P1.1) |
| `type Moment struct{...}` of CAL 4; `func (c Calendar) At(tick int64) (Moment, error)`; `func (c Calendar) SeasonAt(tick int64) int`; `func (c Calendar) TicksPerYear() int64` | the exported surface (A3, A4); `SeasonAt` accepts any `int64` | plans 1-5 (the season read once at P1.1), 1-6 (`TicksPerYear` for the calibration formulas), the viability row (`SeasonAt` for the virtual start offsets, FR-VIA-053), the world row (`At` at P1.1), `spec-indicators.md` |
| `func load(c *Config, qty []Quantity) (mass, bulk int64, ok bool)`; `func fits(mass, bulk, capMass, capBulk int64) bool`; `func chargeableSize(c *Config, id CommodityID, qty Quantity, bulkFactor int64) (size int64, ok bool)`; `func chargeableCapacity(capMass, capBulk, bulkFactor int64) (int64, bool)` | Phase 3, primitives in and out, checked (`ok` false is overflow); `fits` is the boolean of FR-GDS-005 only: GDS 3.2's "binding dimension" has no rule for a tie, a load that fits or a zero capacity, and TRN 3.2.1 (binding fill) and STO 3.4 define their own, so each computes it from `load`'s pair and its capacities | `load` and `fits`: the transport and storage rows (TRN 3.2, STO 3.4), not Stage 1; `chargeableSize` and `chargeableCapacity`: no module of version 1 (GDS 3.3, 3.5; STO 8.4), they exist only for FR-GDS-006 and 007 |

Read from the configuration by ID, with no wrapper (A2, A10), so there is no
second copy to drift: `cfg.Recipes[id].Duration`, `.Sited`, `.TraineeEligible`
and `.TraineeYield`; `cfg.Commodities[id].ReferencePrice`, `.Mass`, `.Bulk`,
`.ShelfLife` and `.Durability`; `cfg.Calendar`. Plan 1-4 reads the reference
price this way (its `ref` parameter), and plan 1-5 reads `Duration` and
`Sited`. The world row holds the canonical `Config` the tables were built from
and hands readers the tables and that value.

### Consumes (Stage 0, built)

| Symbol | Where | Use here |
|---|---|---|
| `lookup`, `newLookup(c *Config) lookup` | `validate.go` | `newRoleTable` builds it once and passes it to `tradableSets` (A6) |
| `tradableSets(c *Config, lk lookup) []map[string]bool` | `validate.go` | one tradable set per role in `c.Roles` order; the one implementation of AGT 3.4's core terms |
| `effectiveUpkeep(c *Config, r *Role) []Item` | `validate.go` | the base list of a role |
| `Config.commodityID`, `roleID`, `recipeID`, `regionID` (`(id string) (TypedID, bool)`, binary search over the canonical slices; the receiver is canonical) | `config.go` | string to typed ID, once, at construction; a false result is an invariant panic after `Validate` |
| `internal/checked`: `Add`, `Mul`, `MulDiv`, `MulDivCeil` (all with an `ok` result) | `internal/checked` | the absolute day, `DaysPerYear * TicksPerDay`, the loads, the merged upkeep |
| The bounds `MaxMonths`, `MaxSeasons`, `MaxDaysPerYear`, `MaxTicksPerYear`, the calendar field bounds | `bounds.go` | the derivation of Phase 2 |
| `ErrTooMany`, `ErrOutOfBounds`, `ErrCalendarTick`, `ErrCalendarSeasons`, `ErrOverflow` | `errors.go` | the sentinels of the derivation and of `At` (A3, A4); no new sentinel |
| `CommodityID`, `RecipeID`, `RoleID`, `RegionID`, `Quantity`, `Price` | `value.go` | the table row types |

Nothing here consumes another Stage 1 row. The ledger row's `ledger.go` reads
`Commodity.ShelfLife` and `Commodity.Durability` from `config.go`, not from
these tables.

## Phase 1: settle the owed spec lines

**Why first:** three spec texts disagree with the built code and with each
other, and two lines the code needs are not written (the sentinel of `At`, and
what `At` does with a calendar that fails the rules). This phase changes specs
and notes, so it starts by reading `doc/notes.md` in full and quoting its tag,
it handles the open notes tagged GDS, PRO, AGT, CAL and `ALL` (the table
below), and `doc/notes.md` gets a fresh tag (`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **The "where this lives" tables** of `spec-goods.md` 12, `spec-production.md`
   19, `spec-agents.md` 22 and `spec-calendar.md` 12 (A1): the structs are in
   `config.go`, the defaults in `defaults.go`, the IDs in `value.go`; the file
   of this row holds the derived tables and the rules of the concern. Correct
   the names `RecipeOutput`, `Requirement` and `Draw` of PRO 19 to the built
   `Output` and `Tool` (`Draw` is the resources module's). AGT 22 lists `Role`
   under `agent.go`: correct it to `config.go`, and move the home of the
   tradable set from `internal/household` to `role.go` (architecture.md 2.1: a
   set union needs no package; the layout of 2.6 has no `internal/household`,
   see O2).
2. **CAL 3.2, 4.5 and FR-CAL-010, FR-CAL-002** (A3, A4): the signatures of
   `At`, `SeasonAt` and `TicksPerYear`; that `DaysPerYear` is derived and not
   exported; the sentinels `ErrOutOfBounds` for a negative tick and
   `ErrOverflow` for a day number that does not fit; and the behaviour for a
   calendar that fails the numeric rules (7.2 and the two count bounds of
   `ErrTooMany`; an error from `At`, 0 from `TicksPerYear`, -1 from
   `SeasonAt`), as a new requirement **FR-CAL-022** (the next free ID of the
   area; no ID is renumbered, testing.md 4.1). 3.2 also says that `SeasonAt`
   returns -1 both for no seasons and for an invalid calendar, and that the ID
   rules of FR-CAL-014 are the validator's alone.
3. **GDS 12** (A7): one row for the load pair and the chargeable size, in
   `commodity.go`, unexported. `load` and `fits` are what `transport.go` and
   `storage.go` test against (TRN 3.2, STO 3.4); `chargeableSize` and
   `chargeableCapacity` are read by no module of version 1 (GDS 3.3, 3.5;
   STO 8.4) and exist only for FR-GDS-006 and 007.
4. **AGT 3.4 and 5.2, PRO 3.3**: one sentence each that the tradable set and
   the effective upkeep are derived once at construction into the role table
   (a table, not a function called in the tick), the list in canonical
   commodity order, and that a module adds its items inside the constructor,
   before it returns (after that the table is a plain value no code mutates,
   which is all "frozen" means here; O1).
5. **The roadmap.** Link this plan from Stage 1, set the row to **in progress**
   when Phase 2 starts, and write in its description that the structs are
   Stage 0's. In the row of `agent.go` and `strategy.go`, add one remark that
   the weight multiplication of the effective upkeep (FR-AGT-011), the
   "requests no job" half of FR-AGT-010 and, with the demand module, the want
   items of FR-AGT-003 are its requirements (this row's tables cover the
   rest); the same three lines are note N-236.
6. **Notes** (the table below). The new note **N-228**, tagged AGT, DEM, SUB and
   CODE, was raised with this plan (2026-10-13, `doc/notes.md` carries it and a
   fresh tag): a module's contribution to the tradable set has no touch point
   in architecture.md 2.10.1 or in the four fan-ins of 2.10.4 (O1), and AGT
   22's `internal/household` is not in 2.6 (O2). This deliverable updates it
   with what the spec edits settle. **N-236** (area AGT, raised with the audit
   of this plan, 2026-10-13) records the three halves of FR-AGT-003, 010 and
   011 that this row does not cover and the agent row must; this deliverable
   keeps it true. Any further gap found is a note in the same change.
7. **`doc/documentation_index.md`**: the row of this plan; correct it if the
   phases change.

### Disposition of the open notes

Every open note tagged GDS, PRO, AGT, CAL or `ALL` in the index of
`doc/notes.md` (read in full on 2026-10-13, tag `6cc73d3abf2fd311fb034a21`;
the tag and the index move whenever any agent edits the notes, so Phase 1
re-reads the file and restates this table against the live index before it
edits anything), each applied, rejected or left open.

| Note | Tag | Disposition |
|------|-----|-------------|
| N-031 What the calendar hands to other specs | GDS | **Left open, with what this row learned.** The calendar's parts are applied (the moment is derived and adds nothing to a checkpoint, `SeasonAt` exists). What remains for GDS is that durations are in ticks and never rescaled: this row adds the test that no table converts a duration (FR-CAL-017) and changes no text |
| N-039 A hard cost floor | GDS, PRO | **Left open.** It is BEL's rule and STB's scenario S07; this row builds no belief or cost. The shelf life it depends on is read by the ledger row |
| N-030 Five regions with different climates | CAL, PRO | **Left open.** CAL's part is done in spec (one global cycle, 5.3) and in code by `At` and `SeasonAt` here; the yield values are RES's and the table lookup is the production row's |
| N-043 Nothing anchors the price level | AGT, PRO | **Left open.** Nothing in this row prices or halts |
| N-067 The production halt has no counter | AGT | **Left open.** `ProductionHalted` is declared (Stage 0); the halt is the strategy and world rows |
| N-087 Wants against the production halt | AGT | **Left open** (scenario S24) |
| N-090 Relocation and the specs that tie obligations to a region | PRO | **Left open.** The recipe table keeps no region state |
| N-101 Does the default `Target` count `Inbound` | AGT | **Left open** for the strategy row |
| N-111 A view takes draws | AGT | **Left open.** No view and no draw here |
| N-118 One job per agent limits the subsistence floor | PRO | **Left open** for the first scenario that shows it |
| N-130 What IND cannot measure exactly | PRO | **Left open.** No snapshot field or event is added |
| N-137 Room is committed at the decision | PRO, AGT | **Left open** for STO; the recipe table gives it the inputs and outputs by ID |
| N-139 The default strategy discards, for upkeep only, when STO is on | AGT | **Left open** for STO |
| N-142 A job start leaves the upkeep room free | PRO, AGT | **Left open** for STO |
| N-144 A busy employer cannot post vacancies | AGT | **Left open** for EMP |
| N-153 Gates by settlement class: `Role.Needs`, `Role.Ceiling` | AGT | **Left open.** The role table has no field for them; SET appends them with its module (D3) |
| N-156 A title or a facility bought needs an observable return | AGT | **Left open** (SAV, CAP, LND) |
| N-164 Credit requests and offers are plan fields | AGT | **Left open** for CRD |
| N-188 Incidence is statutory | AGT | **Left open** for FIS |
| N-191 A right is read at a start, an order and a place | PRO | **Left open** for ORG |
| N-196 A `Facilities` group in the `Trade` view | AGT | **Left open** for CAP |
| N-200 A gain is not a recipe | PRO | **Left open.** The recipe table has no unlock clause; TEC adds one if it is built |
| N-214 Keyed random streams: the edits owed | AGT | **Left open.** This row has no draw site; D3, D5 and D10 are the agent row's |
| N-228 A module's items in the tradable set have no touch point | AGT, DEM, SUB, CODE | **Left open, with what this row learned.** The role constructor of Phase 5 leaves the seam (a module's items are appended inside it, before it returns); the fifth fan-in and the `internal/household` question are O1 and O2, and the edit is held in N-235 |
| N-235 One combined `architecture.md` edit for the fan-in and admission touch points | ALL | **Left open.** Not this row's to make (the sequencing owner is the first Stage 2 module PR); nothing in Stage 1 needs it |
| N-236 The halves of FR-AGT-003, 010 and 011 this row does not cover | AGT | **Left open.** The tables cover their part; the weight multiplication, the "requests no job" half and the want items are the agent row's (deliverable 6 keeps the note true) |
| N-016 Keep the parameter registry true | ALL | **Applied.** This row adds no configuration field, default or bound, so the registry test and its `notBuilt` list do not change; the "no new row" statement is in the PR |
| N-017 Words and names | ALL | **Applied.** The terms the four files use are added to `doc.go` before the code (the list is in Phases 2 to 5). Setting-neutral names throughout |
| N-027 No default macro constant is stable yet | ALL | **Applied (standing).** The row sets no default and claims no stability |
| N-103 Running several spec writers in parallel | ALL | **Applied in its lesson.** This plan was written while the ledger row was being built: shared files are edited with targeted edits after a re-read (A9) |
| N-126 A new event or `Warn` adds a row | ALL | **Applied: nothing to add.** No event and no `Warn` (the warnings of GDS, PRO, AGT and CAL are Stage 0's and are rows already) |
| N-128 A hosted per-agent field is saved once | ALL | **Applied: nothing to add.** The tables hold no per-agent state |
| N-163 The word "treasury" | ALL | **Applied: not used.** No file of the row mentions it |
| N-217 D5 binds every module's change | ALL | **Applied.** The tables are derived and unhashed, no event or snapshot field is appended, and `testdata/golden/off/` and the pinned hash do not move (the Phase 6 gate checks it) |

N-019 and N-224 (CODE) are read for what the first change that touches the area
must do: the glossary guard, the import guard and the determinism guard run
over the new files unchanged (Phases 2 to 6); the glossary guard's file list
gains `calendar.go` so that `Moment` is checked (Phase 2).

**Tests:** none; documents only.

**Observability and rollback:** none, because no code changes. A wrong edit is
reverted like any document change; nothing depends on it until Phase 2.

**Acceptance:** the four "where this lives" tables name the built files; CAL
states the signatures, the two sentinels and the behaviour for an invalid
calendar (FR-CAL-022, counts included); N-228 and N-236 exist and the index of
`doc/notes.md` lists them under AGT (and N-228 under DEM, SUB and CODE); the
notes carry a fresh tag; the roadmap links this plan.

## Phase 2: `calendar.go`

**Why here:** it needs nothing of the other phases and unblocks the rows with
the most waiting on it: `viability.go` needs `SeasonAt` for the virtual starts
of the warm-start jobs (FR-VIA-053; the deferred `ErrWarmStartJob` sub-rule of
plan 0), and the world needs the moment at P1.1.

- The unexported derivation of `daysPerYear` (the sum of the months) and
  `ticksPerYear` (`daysPerYear * TicksPerDay / DaysPerTick`), with
  `internal/checked`, returning the sentinel of the rule the calendar breaks.
  The counts come first, so the sums and the walks that follow are bounded:
  `ErrTooMany` (`len(Months) > MaxMonths` or `len(Seasons) > MaxSeasons`, the
  first phase of `Validate`, `validate.go` `counts`), then `ErrOutOfBounds` (a
  field out of its bound, `TicksPerYear` outside `[1, MaxTicksPerYear]`, a
  `StartDay` or `SeasonStart` outside the year), `ErrCalendarTick`,
  `ErrCalendarSeasons` and `ErrOverflow` (CAL 7.2). The ID rules
  (`ErrInvalidID`, `ErrDuplicateID`, FR-CAL-014) are out of its scope (A3). It
  reuses the bounds of `bounds.go`. The validator keeps its own path-carrying
  rules; a test asserts that the two accept and reject the same calendars over
  the rules the derivation has.
- `Moment` exactly as CAL 4 gives it.
- `Calendar.At(tick)` (CAL 4.1 to 4.5): the absolute day, `Year`, `DayOfYear`,
  `Month` and `DayOfMonth` by walking the months, `DayOfWeek` (not restarting
  with the year), `TickOfDay`, `Hour`, `Season` and `DayOfSeason` with the
  cycle that may wrap past the year end and `-1` and 0 with no seasons.
- `Calendar.SeasonAt(tick)` (CAL 4.5.1): any `int64`, the season of the tick
  congruent to `tick` modulo `TicksPerYear`, computed without negating `tick`
  so the smallest `int64` is safe.
- `Calendar.TicksPerYear()` (A3).
- `newDayTable(Calendar)` (A5): month, day of month and season by day of year,
  built once from a valid calendar, read by position, allocation-free. It takes
  a calendar that passed `Validate`, as the other table builders do: given one
  the derivation refuses, it panics with a message that says "invariant" (it
  is a programmer error, not input, architecture.md 8.1), and a test asserts
  the panic message.
- `doc.go`: the glossary gains moment, absolute day, calendar day, tick of day
  and ticks per year, before the code uses them. The glossary guard's file list
  gains `calendar.go`.

**Tests:** unit, in the external package for the public API, and one internal
test for the derivation and the table.

- `At`, a table whose expected moments carry their arithmetic in a comment
  (testing.md 5.5): the Earth default at tick 0, tick 58 and 59 (the first day
  of the first season is day 59), tick 364 and 365 (the year advances, the
  weekday does not restart: 365 mod 7 = 1), a winter that wraps across the new
  year; the 320-day planet of CAL 7.4 (tick 1 is day 0, tick of day 1, hour 10;
  tick 640 is the first tick of year 1); `DaysPerTick` 5 and 73; no seasons gives
  -1 and day of season 0; `StartDay` at the last day of the year
  (FR-CAL-004 to 009).
- Refusals: tick -1, tick `MaxInt64` with `DaysPerTick` above 1, a `StartDay`
  that overflows on the add, each `errors.Is` `ErrOutOfBounds` or
  `ErrOverflow` with no moment (FR-CAL-010); a hand-built `Calendar{}`, no
  months, `DaysPerTick` 0, `TicksPerDay` and `DaysPerTick` both 2, seasons that
  do not add up, 101 months and 101 seasons each return the sentinel of their
  rule (`ErrTooMany` for the two counts) and never panic or divide by zero (A3;
  FR-CAL-011, 012, 013 in part, FR-CAL-022).
- Periodicity (FR-CAL-006): for a spread of calendars and ticks,
  `At(t + TicksPerYear)` has the next `Year` and the same year-level fields.
  FR-CAL-020 is stated on the largest and the smallest `int64`, where
  `tick - TicksPerYear` wraps in the test itself, so the test compares
  `SeasonAt(MinInt64)` with `SeasonAt(MinInt64 + TicksPerYear)` and
  `SeasonAt(MaxInt64)` with `SeasonAt(MaxInt64 - TicksPerYear)` (the
  subtraction cannot wrap there), and in the middle of the range `SeasonAt(t)
  == SeasonAt(t - TicksPerYear)` with negative ticks included. `SeasonAt(t)
  == At(t).Season` wherever `At` accepts `t` (FR-CAL-019); -1 with no seasons.
- Agreement with the validator (`Guard:`): over the Stage 0 packs, the defect
  overlays and a set of generated calendars (a fixed seed range, the failing
  seed in the message, testing.md 6.2), the derivation succeeds exactly when
  `Validate` raises none of the sentinels the derivation owns, `ErrTooMany`,
  `ErrOutOfBounds`, `ErrCalendarTick`, `ErrCalendarSeasons` and `ErrOverflow`
  at a path under `calendar`, with the same `TicksPerYear`. The ID rules are
  excluded explicitly: a generated calendar with a duplicate month ID is
  accepted by the derivation and rejected by `Validate` with `ErrDuplicateID`,
  and the test asserts that, so the exclusion is a checked fact and not an
  omission.
- The table: month, day of month and season by day of year equal `At`'s for
  every day of the default, the 320-day planet and a calendar with a wrapping
  cycle (`Guard:` for A5).
- FR-CAL-003: `TicksPerYear()` of `DefaultCalendar()` is 365. FR-CAL-001: the
  default equals the values of the requirement field by field. FR-CAL-017: the
  derivation, `At` and the table builders of this row do not mutate the
  configuration they read, asserted by a deep-equal of a `Config` (the packs
  `one` and `second` with two different calendars) before and after, which is
  what "no code converts a duration" can mean for code that only reads.
- FR-CAL-018: a canonicalised deep copy of a configuration gives equal moments
  for every tick of a year (a restore rebuilds the same derived value).
- `FuzzCalendarAt` (testing.md 7.4; NFR-CAL-002, 003): a byte string decoded
  into a calendar and a tick; asserts no panic, every field of the moment in
  its range, `SeasonAt` in `[-1, len(Seasons))` and periodic. Seed corpus with
  `f.Add`; corpus directory `testdata/fuzz/FuzzCalendarAt/` in the root.
  NFR-CAL-002 also names the validator: its fuzz test is Stage 0's
  `FuzzValidate` (which asserts no panic over any `Config`, calendar fields
  included), so that half is cited there and is not re-built; the fuzz test of
  this row adds the derivation and `At`, and asserts that the derivation and
  `At` agree on whether the calendar is usable (`At` returns an error exactly
  when the derivation does, for a tick in range).
- An `Example` test for `Calendar.At`, `SeasonAt` and `TicksPerYear` in the
  existing `example_test.go` (architecture.md 6.5: a runnable `Example` for a
  public entry point), showing the Earth default at tick 0 and tick 59.
- Cost, ordinary tests with `testing.AllocsPerRun` (testing.md 8.2): `At`,
  `SeasonAt` and a table read allocate nothing (NFR-CAL-001, 003).

**Observability and rollback:** none, because the methods are pure, emit
nothing and hold no state; nothing in the module calls them yet. A pure
function with no event answers question 2 of
observability-and-logging.md section 9 with "no state transition", as CAL 8.1
says. Rollback is reverting the PR.

**Acceptance:** `gofmt -l .` prints nothing; `go vet`, `staticcheck` and
`go test -race ./...` pass; the import, determinism and glossary guards of
Stage 0 pass over the new file; `FuzzCalendarAt` runs with its seed corpus and
for five minutes by hand with no crasher (a crasher is committed with a `Fix:`
comment, testing.md 7.5); the allocation tests pass at zero; root statement
coverage does not fall below Stage 0's (testing.md 9.2).

## Phase 3: `commodity.go`

**Why here:** it is the smallest and independent; it comes before the recipe and
role tables only because they all index commodities and a reader should meet
the goods first.

- The load of a set of goods: the pair of the sum of quantity times `Mass` and
  the sum of quantity times `Bulk`, with checked arithmetic, over the quantities
  indexed by commodity (GDS 3.2, FR-GDS-005).
- Whether a load fits a container of a mass capacity and a bulk capacity: both
  sums within their capacity (FR-GDS-005). It returns the boolean only; the
  binding dimension is not computed here (TRN 3.2.1 and STO 3.4 each define
  their own).
- The chargeable size of one unit, `max(mass, ceil(bulk * BulkFactor / 10 000))`,
  and of a quantity, checked (GDS 3.3, FR-GDS-006); the chargeable capacity,
  `min(Cm, floor(Cb * BulkFactor / 10 000))` (3.4, FR-GDS-007). Both use the
  rounding the spec states, in the doc comment (architecture.md 3.5).
- `doc.go`: the glossary gains mass, bulk, load, container, chargeable size and
  chargeable capacity.

All of it is pure and takes primitive values; it does not read the world. The
`World.BulkFactor` default and bounds are Stage 0's.

**Tests:** unit.

- The worked example of GDS 3.7, one case per bullet with the arithmetic in a
  comment: 20 ingots fit and a 21st breaks mass, 10 fleeces fit and an 11th
  breaks bulk, 16 ingots and 6 fleeces fit and their chargeable size is 140
  against a chargeable capacity of 100 (FR-GDS-005, 006, 007).
- Boundaries (testing.md 3.8): an empty load, a commodity with `Mass` 0 and
  `Bulk` 0 (a service) adds nothing, capacity exactly met, one over on each
  dimension, `BulkFactor` 1 and `MaxBasisPoints`. The overflow refusal is
  proved on the primitives, which take `int64`: with in-bound values it is
  unreachable (`MaxQuantity` 10^9 times the mass bound 10^6 of
  `bounds.go` times `MaxCommodities` 10^3 is 10^18, which fits `int64`), so the test feeds out-of-bound `int64`
  values (a mass near `MaxInt64`, a sum that crosses it) and asserts that the
  result is `ok` false and not a wrapped number.
- Rounding: `ceil` on the unit's size and `floor` on the capacity, with a
  case where they differ from truncation (FR-GDS-006, 007).
- Property (`Guard:`, a loop over a fixed seed range, `1..200`, with the
  failing seed in the message, testing.md 6.2): any load of a single commodity whose chargeable
  size is within the chargeable capacity fits (FR-GDS-007).

**Observability and rollback:** none, for the reasons of Phase 2. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds.

## Phase 4: `recipe.go`

**Why here:** the recipe table is what production, the default strategy and the
viability solver read by ID in the tick, and the role table lists recipes by
the same IDs. It needs only the typed IDs and the resolvers of Stage 0. The
role table (Phase 5) does not read it (A6: the tradable set comes from
`tradableSets` over the configuration), so Phases 4 and 5 are independent and
only the order of reading is a courtesy.

- `recipeTable`, built by `newRecipeTable(*Config)`: one row per recipe in
  `RecipeID` order, holding the inputs, the outputs (with `CostShare` and
  `Discard`) and the tools (with `WearPerUse`) as typed-ID rows in canonical
  commodity order, and whether the recipe is a source (no inputs, PRO 3.1.2).
  `Duration`, `Sited`, `TraineeEligible` and `TraineeYield` stay in the
  configuration and are read by ID (no second copy to drift).
- The makers index: for each commodity, the recipes with it among their
  non-discard outputs, in ascending `RecipeID` (PRO 9.3; PRO 3.2.2: a byproduct
  and a co-product are the same row, told apart only by `CostShare`, which the
  cost allocation of section 9 reads).
- A reference that does not resolve is a programmer error after `Validate` and
  panics with a message that says "invariant" (the determinism guard checks
  the word).
- `doc.go`: the glossary gains source (of a recipe), maker and recipe table.

**Tests:** unit, internal, with literals.

- A recipe with two inputs, two outputs (one byproduct with `CostShare` 0, one
  discard) and a tool: every row resolved, in canonical order, quantities
  equal, `Discard` and `CostShare` carried (FR-PRO-002).
- A recipe with no inputs is a source and a recipe with inputs is not
  (FR-PRO-003).
- Makers: a commodity made by two recipes lists both in ascending ID; a
  discard-only output has no maker; a byproduct has its recipe as a maker like
  any other output (FR-PRO-032); a raw commodity with no recipe has none.
- `Guard:` FR-PRO-001: by reflection, `Recipe` has no function field and no
  embedded field, so a later field cannot add one silently (architecture.md
  4.7).
- Reads allocate nothing (`testing.AllocsPerRun`; NFR-GDS-001 for the
  commodity side).
- Two settings (A8): over `one` and `second`, every row's commodity IDs name
  the same strings the configuration lists, and the makers of every commodity
  are exactly the recipes that name it as a non-discard output.

**Observability and rollback:** none, for the reasons of Phase 2: the table is
not state and makes no decision an operator needs explained. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds.

## Phase 5: `role.go`

**Why here:** the tradable set is the join of the upkeep, the recipes and the
trades, so it is the last of the tables to read. The constructor takes the
configuration only (A6): it calls `tradableSets`, whose `lookup` it builds
itself, and reads nothing of the recipe table of Phase 4.

- `roleTable`, built by `newRoleTable(*Config)`: one row per role
  in `RoleID` order with:
  - the recipes of the role as ascending `RecipeID` (AGT 3.3; an empty list is
    legal, 3.3.1, FR-AGT-010);
  - the `Trades` as ascending `CommodityID`;
  - the base upkeep: the role's list, or the world's when the role's is empty
    (`effectiveUpkeep` of `validate.go`, AGT 5.2);
  - the tradable set as ascending `CommodityID` with no duplicate and a
    membership flag per commodity (AGT 3.4, FR-AGT-003): the upkeep of the
    effective list in any region, the inputs, tools and non-discard outputs of
    the role's recipes, and `Trades`, from `tradableSets` (A6);
  - the effective upkeep per region: the base list merged with the region's
    `ExtraUpkeep`, quantities of one commodity added with checked arithmetic,
    in ascending commodity order, per household (AGT 5.2; the multiplication
    by weight is the agent's and the world's). A row is stored only for a
    region that has an `ExtraUpkeep` (`regionUpkeep`, ascending `RegionID`);
    a region without one reads the base list (`upkeepIn`), so the table is
    roles times regions-with-extra rows and not roles times regions, which
    would be about 0.5 GB at the bounds (1 000 by 1 000 rows of up to 32
    items of 16 bytes).
- The "tradable in some role" flag per commodity, the union of the sets
  (AGT 3.4, GDS 6.4, FR-GDS-002 and the warning `commodity_untraded`).
- The seam of O1, in code terms: the constructor builds each role's set in a
  local slice from the core terms, then appends, in the order of the
  `Modules` pointers, the items of any module that is on (none exist in Stage
  1; each module's PR adds one guarded call there), then sorts and
  deduplicates it and stores it. The table is never written after the
  constructor returns; that is the whole content of "frozen" in N-228
  and in O1 below, and no mutex, flag or second pass exists for it.
- `doc.go`: the glossary gains role table, tradable set and effective upkeep.

**Tests:** unit, internal, with literals; one external test over the packs.

- A hand-computed role (testing.md 5.5): the world's upkeep 3 staple and 1
  fuel; a role with its own upkeep of 2 staple replaces it; a region with an
  extra upkeep of 1 fuel merges to 2 staple and 1 fuel; a role with recipes
  whose inputs, tool and non-discard output join the set and whose discard
  output does not; a `Trades` entry joins it; the set is sorted and has no
  duplicate (FR-AGT-003, FR-AGT-011).
- A role with no recipes has an empty recipe list and still has its upkeep and
  trades in its set (FR-AGT-010).
- The merge adds, never replaces, and an overflowing sum is not wrapped: the
  case that Validate would already reject panics with the invariant message
  when the table is built from it directly.
- Agreement (`Guard:`): for every pack of `testdata/chains/` and every defect
  overlay that validates, each role's tradable set equals the set that
  `tradableSets` returns, and a commodity is flagged tradable exactly when it
  is in some set; a commodity that is in no set and is not a discard output
  only is exactly one `commodity_untraded` is raised for (the warning leaves
  out a commodity that exists only as a discard output, AGT 13)
  (FR-GDS-002, FR-AGT-003).
- Two settings (A8): the tables of `one` and `second` build and hold no ID out
  of range; the role count and recipe count match the configuration.
- Reads allocate nothing (NFR-GDS-001).

**Observability and rollback:** none, for the reasons of Phase 4. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds.

## Phase 6: the exit gate and the traceability

**Why last:** it checks the row as a whole and closes the documents.

- The exit gate on the whole module: `gofmt -l .`, `go vet ./...`,
  `staticcheck ./...`, `go test -short -race ./...`, the full run with
  `-covermode=atomic -coverpkg=./...`, `go mod tidy`; the per-scope coverage
  read from `go tool cover -func=coverage.out` and recorded, with the root at
  or above its Stage 0 figure.
- The traceability check of testing.md 4.6 by text search: every ID in a
  `Requirement:` line of the new tests exists in GDS, PRO, AGT or CAL, every
  requirement the table below lists as covered in full is cited, and every
  requirement it lists as covered in part names the row or module that
  completes it (the three agent halves are in N-236).
- The proof of D5: `testdata/golden/off/` (once the world row records it) and
  the pinned hash of the all-off reference configuration
  (`TestHashConfig_ReferencePinned`) are unchanged; today the pinned hash is
  the check.
- A configuration-only check (architecture.md 9.4 in the form this row can
  carry): a new commodity, recipe and role added to a pack by configuration
  alone appear in the three tables with no Go change (the `second` pack is the
  fixture).
- The roadmap row set to **done** with its commit; `documentation_index.md`
  corrected; N-031 and N-030 updated with what this row applied and what stays
  open.

Not gating, worth doing if time allows: a mutation pass over `calendar.go`,
`commodity.go`, `recipe.go` and `role.go` (testing.md 9.4, a SHOULD), with
`gomu` inside Docker only, `--cpus=4 --memory=16g --memory-swap=16g`,
`GOMAXPROCS` and `--workers` equal to the CPU cap.

**Tests:** the tests of Phases 2 to 5 run under the full gate; the
configuration-only check above is an internal test. No integration or
end-to-end test: there is no world, so the half of testing.md 2.3 that
exercises a requirement through the public API at the integration level is the
world row's (the moment at P1.1, FR-WLD-110 for the derived tables of a failed
world), and the table below says so.

**Observability and rollback:** the checklist of
observability-and-logging.md section 9 is answered in each PR and in the
summary below. No log line, metric or event is added. Rollback is reverting the
PR; nothing outside the four files calls them until the world and production
rows.

**Acceptance:** every command of the gate is clean; the traceability check
finds no ID without a spec row and no "covered in full" requirement without a
test; the roadmap row reads done.

## Security review

The module is a library with no network, no file access, no credentials and no
persistence. This row adds no trust boundary and one new public input: the
tick and the calendar of `Calendar.At` and `Calendar.SeasonAt`, which a
consumer may call with any value.

- **Who can reach it.** A consumer calls `World.Calendar().At(t)` with any
  `int64` and, with a hand-built `Calendar`, with any field values. The table
  constructors are unexported and are called by `NewWorld` after `Validate`.
- **Integer overflow.** `t * DaysPerTick`, `StartDay + d`, `daysPerYear *
  TicksPerDay` and every load sum go through `internal/checked`. The result is
  an error and no moment, never a wrapped value (FR-CAL-010, FR-GDS-005). The
  smallest `int64` is handled in `SeasonAt` by a floor modulus that does not
  negate (FR-CAL-020).
- **Panics and division by zero.** A consumer's calendar can have zero days per
  tick or no month. The methods check the numeric shape first and return the
  sentinel, so no input panics (architecture.md 8.1; the fuzz test and the
  hand-built cases). A panic remains only for an unresolved reference in a
  table built from an unvalidated configuration, which no consumer path reaches
  and whose message says it is an invariant.
- **Resource exhaustion.** `At` and `SeasonAt` check the two counts first
  (`ErrTooMany`, A3) and so walk at most `MaxMonths` plus `MaxSeasons` entries
  even for a hand-built calendar: without that check a consumer's calendar of
  100 000 one-day months satisfies every numeric rule and would make each call
  a 100 000-step walk. The day table has at most `MaxDaysPerYear` (100 000)
  entries and is built once per world. The role table is `MaxRoles *
  MaxCommodities` membership flags (10^6 bytes at the bounds) and, for upkeep,
  one row per (role, region that has an `ExtraUpkeep`) of at most
  up to 2 * `MaxUpkeepItems` (16) = 32 merged items of 16 bytes: the worst case is every region
  having an extra, 1 000 by 1 000 rows, about 0.5 GB per world, which is why
  a region with none reads the base list and stores nothing. A pack that sets
  `ExtraUpkeep` on every region of a 1 000-region, 1 000-role world is far
  from a real one; the figure is stated so that a bound is a choice and not a
  surprise (the cost model of `spec-scale.md` 6 and `spec-experiments.md`
  budget one table per world, and an experiment that builds thousands of
  worlds holds one each). All counts are bounded by the count bounds Stage 0
  checks (`ErrTooMany`) before a table is built.
- **Not applicable:** authentication, authorisation, injection, secrets,
  cross-tenant exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **Exported surface added.** `Moment` and the three methods of `Calendar`
  (A3) are new exported identifiers of the root package, a compatibility
  promise (architecture.md 2.8, 6.6). They are the surface `World.Calendar()`
  and `spec-indicators.md` need, with the field set and types fixed by CAL 4.
  There is no release and no consumer yet, so a correction before the first
  release is a normal PR. Nothing else is exported.
- **No existing contract changes.** The structs, defaults, bounds, sentinels
  and validation rules are untouched; no field is added to `Config`, so the
  hash and `testdata/golden/off/` do not move (D5, architecture.md 2.10.5).
- **Document contracts.** Phase 1 corrects four "where this lives" tables and
  adds lines to CAL; it renumbers nothing and rewords no requirement ID
  (testing.md 4.1). `doc/notes.md` gets a fresh tag.
- **What becomes hard to change, and when.** The order of a tradable set and of
  an effective upkeep list is canonical commodity order, which the golden files
  of the world row will depend on (the order of beliefs and of bids); the
  month and season walk is the spec's. Until the first golden file is recorded
  they may be corrected in a normal PR.
- **Rollout** is one PR per phase (A9). Each leaves the branch passing every
  command of `AGENTS.md`. A phase is rolled back by reverting its PR; later
  phases depend only on earlier ones, and no other row of Stage 1 calls the
  files yet except as stated in the table of "Out of scope".
- **Dependency failure.** None: no dependency, no run-time service.
- **The Go toolchain.** `go.mod` says `go 1.25`. No unstable sort is used where
  equal keys can occur: tables are built in index order and sets are turned
  into ascending slices of unique IDs (architecture.md 7.6).

## Testing (summary, testing.md section 11)

The pyramid for this row is all base: table-driven unit tests, a fuzz test for
the calendar (the one function of the row that takes an arbitrary consumer
input), `Guard:` tests for agreement with the validator and for the day table,
and allocation tests. One external test runs the two packs through the
tables. There is no integration or end-to-end test because there is no world.

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-CAL-001, 002, 003, 004, 005, 006, 007, 008, 009, 010, 017, 018, 019, 020, 022 (new, Phase 1); NFR-CAL-001, 003 | FR-CAL-011, 012, 013 (the validator's path-carrying rules are Stage 0's; here, agreement with the derivation); FR-CAL-015 (order kept through the hash is Stage 0's; here, the table keeps the written order); NFR-CAL-002 (`FuzzCalendarAt` is the `At` and derivation half; the validator half is Stage 0's `FuzzValidate`) |
| 3 | FR-GDS-005, 006, 007 | FR-GDS-004 (the bounds are Stage 0's; here, zero mass and bulk in a load) |
| 4 | FR-PRO-001, 002, 003, 032 | none here for FR-PRO-028: the yield lookup and the run-in-a-new-region-by-table-change check are plan-phase-1-5's (no half of it lives in `recipe.go`) |
| 5 | none | FR-AGT-003 (the core terms of the set, in canonical order with no duplicate and no discard output; the demand module's want items are deferred, O1 and N-228, and the module PR completes it); FR-AGT-010 (the table half: a role with no recipes is valid and keeps its upkeep and trades in its set; that an agent of such a role **requests no job** is the agent and strategy row's); FR-AGT-011 (the per-household merge with checked addition, canonical order; that **each quantity is multiplied by the weight**, with checked arithmetic, is the agent's and the world's, `agent.go` and `world.go`); FR-GDS-002 (the flag; the floor check is Stage 0's); FR-AGT-004 (the set; a belief per commodity is the beliefs row's) |
| 6 | none | NFR-GDS-001 (reads of the tables here; the commodity attributes are read from the world's copy in the world row) |

"Covered in full" means at the lowest level that can verify it (testing.md
2.1). The second half of testing.md 2.3, the same behaviour through the public
API at the integration level, is the world row's for every consumer-visible
requirement above: the moment through `World.Calendar().At` and the derived
tables of a failed world (FR-WLD-110).

Already built and not re-tested here: FR-GDS-002, 003, 016, 018, 019, 021, 023
(validation); FR-CAL-014, 016, 021 and the bounds of FR-CAL-013 (validation);
FR-PRO-005, 038, 039, 040 and the rules of PRO 10.2 (validation); FR-AGT-012
(validation).

Deferred with their code: FR-GDS-009 to 015 (`internal/ledger`); FR-GDS-020
(`regions.go`); FR-PRO-008 to 031, 033 to 037, 041 to 051 (`production.go`,
`internal/production` and the agent and strategy rows); FR-AGT-005 to 009 and
013 to 040 (`agent.go`, `strategy.go`, `spawner.go`); FR-CAL-018's world half.
The three halves of the agent requirements that the tables above do not
cover, and that would otherwise fall through every plan because the agent row
has no plan yet (testing.md 4.5): the weight multiplication of FR-AGT-011, the
"requests no job" half of FR-AGT-010 and the want items of FR-AGT-003 (the
demand module's). Each is recorded in note N-236 (area AGT) and in the
roadmap row of `agent.go` and `strategy.go` (Phase 1, deliverable 5), so the
agent row's plan picks them up.

Boundaries and error paths: every refusal of `At` and every branch of the
derivation has a case that breaks only its rule and is asserted with
`errors.Is` (testing.md 3.9). Golden files: none exists yet. Fuzz:
`FuzzCalendarAt`. Each PR states its coverage and that its tests were run
against a broken version (testing.md 9.3).

## Observability (summary, observability-and-logging.md section 9)

None added, because: no new failure path is survived rather than returned: an
invalid tick or calendar is a returned error and is handled once by the
caller (1, 8); no state transition exists, since a table and a moment are pure
derived values, so no event is added (2); no strategy, clearing or spawner
decision is made (3); no loop runs in a tick and nothing is logged in the table
builders, which run once at construction (4); no event, metric, unit or
attribute changes (5, 6); neither `Error` nor `Info` is used (7); the tests of
section 8 that apply here are the allocation tests of Phases 2 to 5, and the
rest need a world (9).

## Open items

- **O1.** Owed to the owner, recorded as note N-228 in Phase 1: a module's
  contribution to the tradable set (the wants of DEM 3.2.3, the own-use outputs
  of SUB 3.6) has no touch point in architecture.md 2.10.1 or among the four
  fan-ins of 2.10.4, and 2.10.2 allows a module to appear outside its file only
  in the catalogue files. Recommended default: a fifth fan-in, one entry point
  `role.go` calls, a fixed-order sequence of guarded calls into the module
  files that append items before the set is frozen, added to 2.10.4 with a fresh
  tag. Ruled by the coordinator (2026-10-13): not blocking Stage 1; the edit is
  not made in this plan but held in note **N-235** (area ALL) for one combined
  `architecture.md` edit, with the admission-table touch point of plan 1-3
  D38/O2, before the first Stage 2 module PR (`demand.go`). The constructor of
  Phase 5 leaves the seam.
- **O2.** AGT 22 homes the tradable set and other household rules in
  `internal/household`, which is not in architecture.md 2.6 or `AGENTS.md`.
  This row does not need the package (the tradable set is a union of lists,
  `role.go`). The agent row decides whether the targets, tiers and spend cap
  need it (architecture.md 2.1) and then adds it to 2.6 and `AGENTS.md`.
- **O3.** Owed to the owner, A7: build the load and the chargeable size now, or
  defer FR-GDS-005 to 007 to the transport and storage rows. Recommended: now,
  because it is small and is requirements of this row's spec; dropping Phase 3
  loses nothing else. **Coordinator-approved (2026-10-13), owner may override:**
  build now, FR-GDS-005 to 007 stay in Phase 3.
- **O4.** No CI exists (O3 of plan 0). The gate of Phase 6 and the timed fuzz
  run are done by hand, on one architecture.
