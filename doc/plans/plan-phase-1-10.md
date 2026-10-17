# Plan: Stage 1, row 10: the exact solver, the viability rules and the warm-start table

**Status: plan written (2026-10-17), audited in rounds 1 to 6 (round 6: ready, no blocking item). Every open item (O8 to O12) and every judgement call (JC1 to JC9) is ruled by the coordinator (final, reversible; the reason is stated where each is used), so nothing waits for the owner.** Sequences the tenth row of
Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`internal/steady` (the exact
solver, `math/big`) and `viability.go` (`SolveSteadyState`, the reader of the
warm-start table; the `WarmStart` types stay in `config.go`), with the five
viability rules of CFG 7.6, `ViabilityError` and the solver-dependent warm-start
checks in `validate.go` (D4: FR-CFG-033, 041 to 044)", into seven phases. It
implements [spec-viability.md](../specs/spec-viability.md) (VIA) and the parts of
[spec-config.md](../specs/spec-config.md) 7.4.1 and 7.6 that VIA fills. It is built
against `doc/rules/architecture.md` (the layout of 2.6, the `internal/` boundary
of 2.9, the `math/big` exception of 3.5, the determinism rules of section 7),
`doc/rules/testing.md` and `doc/rules/observability-and-logging.md`. Where this
plan and the rules or the specs disagree, they win and this plan is corrected.

Rule tags read for this plan: architecture=905c0e578c4e662908638dd8,
observability=35827912d1e3141a76af3937, testing=b3559bc6d47470c221bc4642.
Notes tag (as read, 2026-10-17, before this plan's edits): 61bd3ab432f88e1b1af8c3a2.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR or
the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md) (D3, D4
and D5 above all). Where a rule is still silent, Phase 1 writes it into the
document that owns it before the code that depends on it, as plans 1-4 to 1-6 do;
each such rule is a numbered decision here, flagged for the owner's override.
Decision IDs D120 to D134 are this plan's (plan 1-6 used D61 to D70; the other
rows planned at the same time use other ranges). Note IDs N-274 to N-282 are this
plan's. Earlier plans' decisions are cited by file and ID.

No PRD exists for this row. The requirements source is the roadmap row, VIA and
the CFG sections named. The row is built after the goods, clearing, beliefs,
production and welfare rows
([plan-phase-1-2.md](plan-phase-1-2.md) to [plan-phase-1-6.md](plan-phase-1-6.md)),
whose tables and functions it reuses, and before the world row, which applies the
table it produces.

Spec codes: VIA viability, CFG config, PRO production, GDS goods, CAL calendar,
AGT agents, WEL welfare, BEL beliefs, CUR currency, REF reference chain, WLD
world, OBS observability, DEM demand, TRN transport, STB stability, CLB
calibration, SCL scale, SKL skill barriers. `FR-`/`NFR-` IDs are the requirement
IDs of those specs.

## What Stage 0 and rows 1 to 6 already built, and what is left

| Concern | Where it already is |
|---------|---------------------|
| The `WarmStart` types (`SeedPrice`, `SeedWage`, `WarmGroup`, `JobSpec`, `WarmPool`), their canonical order, the hash pairs and the hash of a nil against an empty table | `config.go`, `hash.go` (Stage 0) |
| The warm-start rules that need no solver: seed price in `[1, MaxPrice]`, at or above the resolution floor and inside the reference band; wage in `[1, MaxMoney]` and its product with the longest `Duration`; duplicate wage; money, stock and share bounds; a job recipe on the role's list; shares of a group summing above 10 000; overflow of weight times stock; unknown references | `validate.go` (`warmRules`, `warmProducts`), `ErrWarmStartPrice`, `ErrWarmStartJob` in `errors.go` |
| The structural producibility fixpoint of CFG 7.5, `ErrUnproducible` with the named cycle | `validate.go` (`structure`, `witness`) |
| The informational `window_extended`, `cost_floor_at_full_cost`, `raw_without_source` and the other Stage 0 warnings, `Config.Warnings` | `validate.go` (`warnings`) |
| `MaxSeedPrices`, `MaxTicksPerYear`, `MaxRecipeTicks`, `MaxYieldBP`, `MaxUpkeepItems`, `MaxRequires` | `bounds.go` |
| `Calendar.TicksPerYear()` and `Calendar.SeasonAt(tick)` (any `int64`, `-1` with no seasons) | `calendar.go` (row 2) |
| The recipe table (`newRecipeTable`, `makers`), the role table (`newRoleTable`, `upkeepIn(region)`, `recipes`, the effective window) | `recipe.go`, `role.go` (rows 2 and 6) |
| The yield table (`newYieldTable`, `site(region, recipe, season, sited)`, `effective`) and `internal/production` (`Site`, `Effective`, `UnitCost`, `Basis`, `WearCost`) | `production.go`, `internal/production` (row 5) |
| `warmJob`, the pending output of a job made from a virtual start (it takes the remaining ticks, the executions and the season) | `production.go` (row 5, `spec-production.md` 15) |
| The region wage read: pooled, else `WarmStart.Wages` (`seedWages`, `regionWage`) | `welfare.go` (row 6) |
| The calibration formulas `Wage`, `RatioForWage`, `DaysWorked` with `ok` | `internal/welfare/calibration.go` (row 6) |
| The 128-bit helpers `Add`, `Mul`, `MulDiv`, `MulDivCeil`, `MulDivHalfEven` | `internal/checked` |
| The packs `one`, `two`, `three`, `five`, `famine`, `one-demand`, `second`, the six `defects/` overlays and the `warm-*.toml` documents | `testdata/chains/` (Stage 0, REF) |

What is left for this row: the exact solver and its certificate; the start count
and the net vector; the four layers of viability (V1 to V4) and the rules and
warnings that report them; `ViabilityError` and the sentinels `ErrNotViable`,
`ErrNoUpkeep`, `ErrSolveLimit` and `ErrSolveInconsistent`; the two bounds Stage
0 left out (`MaxViabilityPivots`, `MaxViabilityCells`: FR-CFG-046; `MaxLanes` waits for `WithLanes`, D127);
phase 6 of `Validate` and the viability warnings of `Warnings`; the empty-set rule
of `ErrWarmStartJob`; and `SolveSteadyState` with the table, the report and the
census.

## Scope

**In scope.**

- `internal/steady` (standard library only, `math/big` allowed here and only here,
  architecture.md 3.5; imports `internal/checked` for its boundary only; never
  imports the root): a fraction-free exact simplex with Bland's rule, the
  certificate (primal feasible, dual feasible, equal objectives, from the
  original data), the greedy start count with the mean factor, the net vector,
  the reference programme in its autarky, pooled (free trade only: Stage 1 has no other
  lane set, D129) and nominal forms, the strongly connected components of the material graph with the
  productive-cycle programme, the producibility fixpoint on existing activities,
  the supply gap, the lean reserve, the rounding of a result to `int64` at its
  boundary, and the report diagnostics (degenerate, alternative optima, residual,
  census, advantage).
- Root files (standard library and `internal/` only): `viability.go` (the public
  API: `SolveSteadyState`, `SolveOption` and its five options, `SteadyState`,
  `ProgrammeResult`, `Report`, the conversion of a `Config` to the solver's
  primitives, the virtual start offsets), `viability_check.go` (V1 to V4 as
  called by `Validate` and `Warnings`, the size gate), `viability_table.go` (the
  seed prices, the wage, the money, the stock, the jobs and the report bands).
  Splitting into three files is for the Lizard limits of N-243 (about 500 lines,
  complexity 10, 8 parameters); `architecture.md` 2.6 and `AGENTS.md` name the
  files in Phase 1 (N-243 asks for it for `production_report.go`, and this row
  does the same).
- `errors.go` (`ErrNotViable`, `ErrNoUpkeep`, `ErrSolveLimit`,
  `ErrSolveInconsistent`, `ViabilityError`), `bounds.go` (`MaxViabilityPivots`,
  `MaxViabilityCells`), `validate.go` (phase 6 and the virtual-offset
  rule), and `doc.go` (the glossary terms of this row, before they are used).
- The spec, note and roadmap edits these files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| Applying the table at construction: the stagger of 8.4.3, the creation of the jobs, the grant of stock and money by `initial`, the construction bucket, the log line "warm start applied", `WorldCreated.WarmStart`, the pool (FR-VIA-031 to 042, 046, 047) | Roadmap Stage 1, `world.go` and `agent.go` rows. This row gives them `virtualStartOffsets` and the validated table |
| The starting wage as the pooled wage of the cost anchor (FR-VIA-052) | Built by [plan-phase-1-6.md](plan-phase-1-6.md) (`regionWage`); this row only produces the wage |
| The checks of FR-VIA-021 ("`NewWorld` never calls `SolveSteadyState`") and A7, A8 (conservation, stagger) | World row (a world needs to exist); plan 1-12 lists A7/A8 as not tested in Stage 1 and plan 1-11 Phase 3 owns `FuzzApportion` and `FuzzStagger` (NFR-VIA-004, FR-VIA-034, 037) and takes only `virtualStartOffsets` from this row; REF A7/A8 acceptance (conservation, stagger at the world level) stays unowned in Stage 1 (plan 1-12 lists it as not tested) |
| The resource rows, `Headroom`, `Rent`, `endowment_runway`, `endowment_season_peak` (RES), the facility row and `FacilityDecay` (CAP), the open layer (EXT), the wants term and `SurplusWantsBP`, `wants_exceed_labour` (DEM), `own_use_beats_market` (SUB), `rent_off_dual` (LND), the `room_below_warm_job` check (STO), `cap_below_seed_cost` (ORG), `Lane`, `WithLanes`, `MaxLanes` and any lane that is not the free-trade set, the lanes of `transport.Lanes` and the loss of `spec-risk.md` 8.3 | Each module's PR, appended (architecture.md 2.10.5). Every module is off in the slice, so the programme is the closed one of 3.2. The types keep the places (D5) |
| The experiment harness' `Prepare` and the warning `stale_warm_start` (N-134) | `experiment/` (Stage 3); this row publishes nothing new beyond VIA 11's path list |
| A time-expanded programme (seasonal storage), a `BalancedCensus` rewrite, a default pool (VIA open questions 2, 3, 6) | Left open in VIA; not built |
| The byte-identity of `warm-*.toml` with a TOML writer (FR-REF-007) | Phase 7 here proves equality of the loaded table; the writer and the two-architecture run are the golden row's (plan 1-12), see D133 |
| The d1 to d6 file tests (`TestPacks_DefectOverlays` in `config/`) | Plan 1-12 owns them (the rule name per defect, "that rule only", the warm layer set, the wild-source case); **this row's Phase 5 PR flips the Stage 0 line** that expects `Validate` to be nil for d1 to d5 to `errors.Is(err, ErrNotViable)` (coordinator ruling (k); plan 1-12 A8, N-299), so the suite stays green; this row also owns the failing-case unit tests on documents and the contract they read (Provides) |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- The solver is a pure function of `Config` and its options; `NewWorld` never
  calls it; it draws nothing, logs nothing and starts no goroutine (VIA 7.1, 7.4,
  FR-VIA-020, 021; architecture.md 7.1 to 7.3, 11.1, 12.2). It takes a
  `context.Context` first and checks it between pivots (FR-VIA-017;
  architecture.md 12.3).
- Exact integers of unbounded size inside `internal/steady`, Bland's rule, no
  floating point of any kind (`big.Float` and `big.Rat`'s float conversions
  included), and a result leaves as `int64` once, at the boundary, with the
  rounding rule of its field: prices and wages half to even, `SurplusBP` down, the
  labour share up, checked against `int64` and the field's bound (architecture.md
  3.5; VIA 6.1 to 6.4). No `math/big` type appears in anything exported from the
  root, an event or a snapshot.
- The answer is certified in exact arithmetic or not returned; failures return no
  table (VIA 5.4).
- `Validate` runs V1 and V4 always and V2 and V3 below the size gate, as phase 6,
  only on a configuration phases 1 to 5 found clean (CFG 7.1.6, 7.6; FR-VIA-010,
  FR-CFG-043).
- The error checks are necessary and optimistic: pooled free trade, the best
  yield across regions (VIA 4.6). The per-region autarky check, the nominal
  checks and the cycle gain are warnings.
- The solve options are not `Config` fields, are not hashed and are rows of CLB's
  registry, not of the parameter-owner table (VIA 11; FR-CFG-047). The two
  bounds are in `bounds.go` (CFG 4.1; FR-CFG-046).
- The default `TargetRatio` and the closed-economy identity are not changed; the
  role economics report `WageBelowTarget` is a report and never an error
  (WEL 7.1; FR-WEL-031, 038).
- A module that is off contributes nothing to any programme row, any column or
  any report field, and an appended field is zero while off (D5 of plan 0).

Decided by this plan, to be written into the owning spec in Phase 1 before code
(each is flagged for the owner's override in the review):

- **D120. The flows of the table are those of lambda equal to 1.** The programme
  maximises `lambda`, but a steady state is the programme at `lambda = 1` (VIA
  5.1 (a)): the executions per tick `x` that the jobs (8.5.1), the census (5.2)
  and the lean reserve (8.4.1) use are `x* / lambda*` (the optimum scaled down,
  which keeps every row satisfied), not `x*`. Prices and wages are duals and are
  unchanged by the scaling. VIA 5.1 and 8.5.1 say it in one sentence each. Phase 7
  checks it against `warm-one.toml` (the `refine` and `make_tool_*` shares); if the
  file contradicts the reading, the file is evidence of REF's prototype and the
  owner decides which text moves (O2). Tie: with alternative optima Bland's rule
  picks the vector (FR-VIA-022).
- **D121. A recipe on the lists of several roles of a region.** VIA 3.3 item 2
  makes the activities open to any household and 5.2 flags the census ambiguous in
  that case, and 8.5.1 divides by "the households of the group" without saying
  which. Rule: the executions in progress of the recipe are divided by the sum of
  the households of the groups of the region whose role lists it, and every such
  group gets that same share (each household is equally likely to be the one
  running it). If a group's shares then sum above 10 000 they are scaled down in
  proportion, each rounded down, and the group is listed in `Report.Census` as
  short; a recipe active in a region where no group lists it gets no job and is
  listed there too. Written into VIA 8.5.1 and 5.2.
- **D122. The supply gap is read from the greedy start sequence of `J`** (VIA 4.5:
  "the same start windows as `J`"). The start rule is the prototype's (A3): a start counts
  at every tick of the open window. The audit of round 1 (B1) showed that counting every
  open start tick as a completion does not give REF's numbers: `grow_staple` of `one` is
  open for the 92 ticks of spring (days 59 to 150, `defaults.go`), completions of an
  all-open-ticks reading fall on 118 to 209, and the longest run without one is 365 - 92 =
  **273**, not 304. The greedy sequence of one household (starts 59 and 119, `d` 60,
  completions 118 and 178) gives 365 - 60 - 1 = **304**, and the two-planting pattern
  (spring and autumn) gives 123 (completions 118, 178, 302, 362: the run 178 to 302 is
  123), which are REF's 304, 123 and 184. So: the completion ticks of an activity in a
  region are those of its greedy sequence over the year (start `s`, completion `s + d - 1`,
  `production.go`'s `job`), the sequence repeats every year, and the gap of a commodity is
  the longest cyclic run of ticks with no completion of any existing activity of the region
  that outputs it. **A completion tick at or after `TicksPerYear`** (a job begun late in the
  year, which the start rule counts) **is taken modulo the year for the gap**: the steady
  state is periodic, so that job completes early in the next year (a window on days 340 to
  364 with `d` 60 starts once, at 340, and completes at tick 34: one completion a year, gap
  364). The reserve of D123 does not wrap it (D123 says why). With no seasons, or an
  activity that completes every tick, the gap is 0.
  **An activity whose start is open on every tick of the year** (all-year profile, no season
  at zero yield) **counts as completing on every tick** (option (a) of O10, ruled by the
  coordinator, reversible; this is JC7): the group's households are spread over the `d`
  offsets by 8.5.2 and 8.4.3, so one of them completes at each tick, and a `d` of 2 or 12
  does not make a gap. Reason for ruling (a) over the two alternatives: it is the only
  reading that reproduces all three documents (the greedy sequence of ONE household has runs
  of `d - 1` ticks, and reading it for a staggered group would give `refined`, `tool_a`,
  `tool_b` and `second`'s `biomass` a reserve that `warm-one.toml` and `warm-second.toml`
  do not have, round 2, B2), and it needs no code beyond a test of "open on every tick".
  The greedy sequence is read only for an activity with a closed tick in its year.
  VIA 4.5 already says this; Phase 1 only adds the worked check (starts 59 and 119,
  completions 118 and 178, gap 304 in `one`; 61-day window also 304) and the wrap of a
  completion past the year end. **The all-open-ticks reading is dropped for the gap.** The
  lean reserve (D123) is the one place the plan uses every open start tick, at the rate of
  8.5.1 that spreads the jobs over them (ruled, JC2).
- **D123. The lean reserve is the largest cumulative shortfall over one year from
  tick 0.** VIA 8.4.1 names it without a formula. Rule: let `cons(t)` be the
  region's **whole consumption** of `c` per tick at the flows of lambda 1 (D120): the
  upkeep `D(c, rho)` **plus the inputs and tool wear of every active recipe of the
  region** (round 1, B2: upkeep alone gives 560, not REF's 644; in `one` the realm eats
  1 200 `staple` a tick of upkeep, 150 as the input of `prepare` at 75 executions a
  tick and 30.68 as the input of `save_seed`, 1 380.68 in all, and 1 380.68 * 118 / 253
  growers = 643.95, which rounds up to 644; the 664 of `warm-one.toml` is these 644
  plus the 20 `staple` the grower group holds as the input of one `save_seed`, D134),
  and `prod(t)` the
  output of `c` completing at `t`: each open start tick has `x1 * TicksPerYear /
  N_open` executions starting (`x1` the activity's flow of D120, `N_open` the open
  ticks of the year, the rate of 8.5.1), each yielding its non-discard output at
  the factor of the start tick, credited at the completion tick of D122; **a start whose completion tick is at or after `TicksPerYear` adds nothing (it is not wrapped to the next year), and `N_open` still counts its tick** (round 3, N2; only this reading gives `two` south 789 = ceil(109 959 / 143) 769 + max(cover 10, `save_seed` input 20) in `warm-two.toml`, while wrapping gives 510; `two` north 723 either way). **Rechecked after the revert of the start rule (A3): unchanged.** The production rate uses `N_open`, never `J`, and `x1` of `two` is the same under both start rules (round 3, N5), so the 789 and the 723 do not move; the greedy starts of `two` south (59, 119, 243, 303; completions 118, 178, 302, 362) are the same under both rules. The asymmetry with the cyclic gap of D122 is deliberate: the reserve bridges the first year from tick 0, when nothing begun before tick 0 can complete (jobs in progress at tick 0 are table rows, D134), so a start late in the year has produced nothing by the year's end; the gap describes the periodic steady state. The
  reserve is `max over t in [0, TicksPerYear) of max(0, sum_{u <= t}
  (cons(u) - prod(u)))`, held by the groups that produce `c` in
  proportion to their output (largest remainder in whole units of the total, ties by role ID),
  **then per household `ceil(the group's share / the group's households)`** (round 2, B2: `warm-famine.toml`
  holds 383 + 10 where 1 350 * 118 / 417 = 382.01, so the household's figure rounds up, not the
  total), then cut to the shelf life (8.4.2, FR-VIA-039). It exists only for a commodity with a supply
  gap in the sense of D122 (so none for an all-year activity: `refined`, `tool_a`, `tool_b` of `one`
  and `biomass` of `second` have no reserve, as the documents say); **which commodities carry a gap
  is every commodity the region's activities output, not only upkeep and all-year inputs**
  (`storage_gap` and `storage_gap_input` are the findings that restrict to upkeep and inputs; the
  reserve does not). It reduces to REF's wording (the realm's whole
  use per tick times the 118 ticks to the first harvest of `staple`, shared among the
  producing households) when production is zero up to the first completion. Written
  into VIA 8.4.1; checked in Phase 7 against the five packs.
- **D124. `ViabilityError` is its own type and joins the one sorted list.**
  Fields: `Rule string` (the name of CFG 7.4.2: `unproducible_in_realm`,
  `role_cannot_work`, `cycle_not_productive`, `labour_short`, `storage_gap`),
  `Path []string`, `Value string` (the offending value as text) and `Bound string`
  (the number that decides it, as text: the surplus factor in basis points
  against 10 000, the gap against the shelf life, and so on); `Unwrap` returns
  `ErrNotViable`, so `errors.Is(err, ErrNotViable)` matches every one and
  `errors.As` finds it (FR-VIA-011, FR-CFG-041). It does not embed
  `ValidationError` (architecture.md 4.7). The report keeps one list for both
  types, sorted by path, then by rule name, then value and bound, cut at 100 with
  one `ErrTooManyErrors` (CFG 7.1.2, 7.1.3). The rule names are plain strings with
  no exported constants (architecture.md 2.8); the tests spell them. Written into
  CFG 7.1.1 and VIA 10. **Element type** (round 1, N4): `report.errs` is today
  `[]*ValidationError` and its sort reads `a.Rule.Error()` (`validate.go`); the
  report gains one unexported element type that carries the error to return and the
  common sort key (path, rule name, value, bound), so the two error types sort in one
  list; the helpers that assume `*ValidationError` (the test helper `violations()` and
  `FuzzValidate`) are listed and updated in the Phase 5 PR. **Path, Value and Bound
  of each rule, and how many errors it makes** (VIA 10.1 says only "a path"; plan 1-12
  asserts the counts, so they are fixed here and written into VIA 10.1; flagged O4):
  `unproducible_in_realm`: one error per unproducible commodity, path
  `commodities/<id>`, value the commodity ID, bound empty; `role_cannot_work`: one per
  role, path `roles/<id>`, value the first region where none of its recipes exists,
  bound empty; `cycle_not_productive`: one per component, path `commodities/<id>` of
  the lowest-index commodity of the component, value `s*` in basis points (floor),
  bound `0`; `labour_short`: exactly one for the realm, path `regions`, value
  `SurplusBP` of the pooled programme, bound `10000`; `storage_gap`: one per
  (region, commodity), path `regions/<region>/commodities/<id>`, value the gap in
  ticks, bound the shelf life in ticks. `Error()` text quotes and cuts as
  `ValidationError` does.
- **D125. Phase 6 and `Warnings`.** `Validate` runs phase 6 only if phases 1 to 5
  found nothing (CFG 7.1.6). `Warnings()` is documented "safe on a configuration
  that does not validate" and "linear in size"; the viability warnings are
  neither, so `Warnings` appends them only when the structural phases come back
  clean on a scratch report, and otherwise returns the others (the same gate, so
  the two calls agree). Both reach one unexported
  `checkViability(c *Config, lk lookup) viabilityOutcome` that returns the errors
  and the warnings of V1 to V4 and the cycle gain. Nothing is cached between
  calls; the double cost (`Validate` then `Warnings`) is measured in Phase 7 and
  the world row may call `checkViability` once (N-276). Written into CFG 7.1.6 and
  7.2. **Canonical input** (round 1, B4): `Validate` does not canonicalise (its
  `validator` holds the caller's slice order, and `TestValidate_ReferenceConfigurationIsValid`
  needs reversed lists to validate), but `newRecipeTable`, `newRoleTable` and
  `newYieldTable` index typed IDs into the sorted slices and panic on anything else
  (`recipe.go`, `role.go`, `production.go`). So phase 6 begins with `cc :=
  v.c.canonical()` and `checkViability(&cc, newLookup(&cc))`, `Warnings` does the same
  on its clean-gate path, and the tables are built there only, after phases 1 to 5 found
  nothing. The empty-offset rule of `ErrWarmStartJob` (deliverable 6) therefore runs at
  the entry of phase 6, on the canonical copy, not in phase 4; `virtualStartOffsets`
  keeps the signature of Provides.
- **D126. The size gate.** Before a tableau is built, `rows * cols` is computed
  with `internal/checked` and compared with `MaxViabilityCells`: `rows` is the
  commodities of the programme plus its regions plus one row per activity with
  `J * d < TicksPerYear` (the capacity rows), `cols` is the activities plus one
  for `lambda` plus the slack of each row (Stage 1 has no lane column: the pooled form merges
  the goods rows, D129). The pivots are capped by
  `MaxViabilityPivots`. In `Validate` an over-size tableau, or `ErrSolveLimit`
  during the solve, is not an error: it is the warning `viability_check_skipped`
  naming the part (V2 of a component, V3 pooled, V3 per region, the nominal
  warnings, the cycle gain). V1 and V4 are never skipped and are not
  programmes (FR-VIA-010). In `SolveSteadyState` the same overshoot is
  `ErrSolveLimit` carrying the pivots, rows and columns (`SolveError`). Written
  into VIA 4.3, 4.6 and FR-VIA-010.
- **D127. The solve options** are functional, of the type
  `SolveOption func(*solveOptions) error` (a separate type from the world's
  `Option`, architecture.md 6.2), unexported state, and five of them, named
  `WithStockCoverTicks`, `WithMoneyBufferTicks`, `WithResidualWarnBP`,
  `WithMaxPivots` and `WithFreeTrade` (the last is the name of VIA 9.4). An option outside
  its bound of VIA 11 returns `ErrOutOfBounds` from `SolveSteadyState` before any work; a
  nil option is ignored; the last of a kind wins. **`WithLanes`, the type `Lane` (its
  `Labour` and `LossBP` fields) and `MaxLanes` are not published in Stage 1** (ruled by the
  coordinator, reversible; architecture.md 2.8: export as little as possible, and no
  consumer exists before the transport module of Stage 2, which defines `transport.Lanes`
  and adds the three together, N-279). Because the option set has no way to carry a lane,
  a non-free lane cannot be expressed, so no rejection code is written (it would be
  unreachable); a `Guard:` test (Phase 6) pins that the exported `With*` functions of
  `viability.go` are exactly the five. When TRN adds `WithLanes` its lane validation is an
  error, not a warning (the CFG convention: a configuration that cannot be honoured is
  rejected with a sentinel, and a warning is informational only). VIA 7.1 and 11 gain the
  five names; VIA 9.4 and 11 gain one sentence: Stage 1 realises only the free-trade lane
  set, and `Lane`, `Lanes` and `MaxLanes` are added with the transport module.
- **D128. The exported shape of the result** (the contract the golden row reads,
  see Provides). `SolveSteadyState` returns `SteadyState{Programmes
  []ProgrammeResult, Table WarmStart, Report Report}`. A `ProgrammeResult` is one
  programme solved: `Regions []string` (one region in autarky, all of a
  component in the pooled case), `SurplusBP int64` (floor of `lambda* * 10 000`),
  `LabourShareBP int64` with `HasLabourShare bool` (ceil, absent when a capacity
  row is tight: FR-VIA-012), `NominalSurplusBP int64`, `Pivots int64`. `Report`
  holds the diagnostics of VIA 10.3 that the slice computes (`Degenerate`,
  `AlternativeOptima`, `UnpricedCommodities []RegionCommodity`,
  `UnpricedWages []string`, `Floored`, `OutsideReferenceBand`,
  `SeedBelowCostFloor`, `SeedAboveCostCeiling`, `Drift`, `ResidualBP` and the
  list over `ResidualWarnBP`, `Census`, `CapacityTight`, `Advantage`,
  `CapShortened`, `NominalSeed`, `RegionSurplusBP`, `WageBelowTarget`), every
  list in a stable order (ascending region then commodity or recipe ID) and no
  map. The exact flows are not exported: the census and the table are the only
  flow-derived outputs (architecture.md 2.8). The module-owned fields of 10.3
  (`Headroom`, `Rent`, `FacilityDecay`, `SurplusWantsBP`) are appended with their
  modules and are zero while off. Written into VIA 7.1 and 10.3.
- **D129. Autarky or pooled.** With no `WithFreeTrade`, each region with a group is
  solved alone (its own programme, its own numeraire `B_ref(rho)` of 6.3) and the table's
  rows for the region come from it; with `WithFreeTrade`, all the regions with a group are
  one pooled programme with one scale. The pooled form is the **free-trade set only**: the
  complete set of zero-cost lanes, collapsed to one goods row per commodity (labour and
  capacity rows stay per region); this is the form behind the pivot counts 19, 30 and 202.
  Stage 1 has no partial pooling and no lane column (ruled by the coordinator, reversible;
  D127 gives the reason). A region with no group has no programme and no table row. VIA 3.2
  and 9.4 say "the default is autarky" and "pooled is a comparison tool"; the sentence is
  added to 7.1.
- **D130. The empty basket.** VIA 5.3 item 1 and FR-VIA-018 say `SolveSteadyState`
  returns `ErrNoUpkeep` when no group has an upkeep (the programme is unbounded). That
  is kept as written. What the specs do not say is a programme of an autarky solve
  whose own basket is empty while another region's is not (round 1, N11): the plan
  follows the ruling (JC4): an autarky region with an empty basket is not a programme, with no
  table row, listed in `UnpricedWages`; the call fails with `ErrNoUpkeep` only when no group
  of any region has an upkeep. A programme whose basket goods are all unsourced `Raw` markers (JC5: V3 and the table are skipped, so `minimalConfig()` and `exampleConfig()` still validate) is treated the same way: no table row, its region in `UnpricedWages` and each such commodity in `UnpricedCommodities`, and `ErrNoUpkeep` only when no programme remains. A region with no households is not a programme. **In `Validate`**
  an empty basket is never an error: `no_upkeep` is a warning of CFG 7.4.2 and such a
  configuration validates (a Stage 0 fixture has one, `validate_rules_test.go`), so V3
  and V2's programme are skipped silently for a programme with an empty basket, and
  V1 and V4 still run.
- **D131. The nominal fallback is per programme.** 5.3.4 speaks of "a region"
  whose labour dual at full labour is 0 while a capacity rent is positive. In an
  autarky programme that is the programme itself. In a pooled programme the
  prices are one vector, so the whole pooled table is rebuilt from the nominal
  pooled programme (`phi` the household-weighted mean of the regions), and every
  region whose full-labour wage was 0 with a tight capacity row is listed in
  `NominalSeed` (FR-VIA-058). A region whose nominal wage is also 0 stays in `UnpricedWages`. **Which programme gives which part** (round 2, N1; checked by an exact LP on
  `famine`): the seed prices and the wage come from the nominal programme (lambda 1.11247, wage 1 624),
  but the flows (jobs, reserve, stock inputs, census) come from the full-labour programme (lambda
  96/73 = 1.3151, 9 pivots, `grow_staple` 2.5 and no `save_seed` at lambda 1, because its capacity row is
  tight), which is what `warm-famine.toml` holds (`make_tool_a` 2 272 = 2.5 * 3 / 33; reserve 393 = 383 +
  10 with 1 350 = 1 200 + 150); using the nominal flows would give 2 454 and 433. Written into 5.3.4 and
  8.5.1; a working default pinned against the document in Phase 1.
- **D132. The census formula** is fixed in Phase 1 from `testdata/chains/one.toml`
  (VIA 5.2 and REF 4.3 give the description in prose: the households the
  activities need at lambda 1, the start-capacity row counted, apportioned by
  largest remainder to the configured total) and written into VIA 5.2 as a
  formula with the 600-household figures of `one` as its worked example. It is a
  report only and changes no table (FR-VIA-028). Left to Phase 1 because the prose
  does not fix whether the work share divides the capacity term; O1.
- **D133. Agreement with the warm documents is by loaded equality.** FR-REF-007
  asks the `warm-*.toml` to be byte-identical to the solver's table. The core
  module writes no TOML (architecture.md 5.1, 5.2), so Phase 7 loads each
  document through `config/` and compares it field for field with the table of
  `SolveSteadyState` on the layered configuration without it (`reflect.DeepEqual`
  on the canonical `WarmStart`). Because a file that loads equal to the table and
  was written by one canonical writer is byte-equal, the byte test and the writer
  (a test-only helper under `config/`, used by the `-update` flag of testing.md
  7.2) are plan 1-12's. If a document does not load equal, Phase 7 follows D120:
  the spec text decides, a defect in the code is fixed, and a defect in the file is
  regenerated with the explanation testing.md 7.2 asks (the producer of the
  documents, REF's `lp.py`, is not in the repository: N-274). `DeepEqual(nil,
  []T{})` is false, so the table passes through `Config.canonical()` before the
  comparison (`sortedBy` returns nil for an empty list, `config.go`), and the
  solver leaves absent pools, jobs and stock as nil slices (round 1, N10).
- **D134. The composition rules of the warm table are pinned against the documents
  in Phase 1** (round 1, B2). VIA 8.3.1, 8.4.1 and 8.5.1 do not fix nine things the
  documents `warm-one.toml`, `warm-second.toml` and `warm-famine.toml` fix by example,
  and the code would otherwise choose them. Phase 1 recomputes each against the three
  documents (arithmetic in the spec, as D132 does for the census) and writes the rule
  into VIA 8.3 to 8.5 with `one` as the worked example. The readings that fit `one`
  (U = 2 * 96 + 628 + 179 = 999, 600 households) and are the working defaults of the
  code until Phase 1 confirms them: (1) the reserve counts the whole consumption
  (D123); (2) a commodity's stock per household is the larger of its cover and the sum
  of the one-execution inputs of the group's active recipes (**active** = positive execution in the solver's final basis scaled to lambda 1; the same set decides the main recipe of JC3 and the `d_max` of VIA 8.3.1, so a long recipe the basis leaves unused, such as `salvage` of `second`, does not count), **plus** the reserve and
  the tool items (cook `staple` 10 = max(cover 10, `prepare` input 2); grower `staple`
  664 = 644 + max(10, `save_seed` 20); maker `refined` 5 = 2 + 3, the inputs summed over
  the group's recipes, not their maximum); (3) the "main recipe" of 8.3.1 is the active
  recipe of the group with the LONGEST duration, ties broken by the largest inputs cost, then the lower recipe ID (grower 78 810 = 999 * (60 + 10) + 8
  880, the 8 880 being `grow_staple`'s 40 `seed` at 222, not `save_seed`'s 20 at 96;
  maker 22 557 = 999 * 13 + 9 570, `make_tool_b`'s 3 `refined` at 3 190). In `one` the longest duration and the largest inputs cost pick the same recipe in both groups, so `one` alone does not decide; "largest inputs cost" alone contradicts 16 money values (`second` fabber: `fab_board`, duration 3, gives 19 423 as in the document, `make_fabricator`, duration 1 and dearer inputs, would give 52 189; 15 `crafter_*` groups of `five`), while "longest duration, ties by largest inputs cost" reproduces every group of the five warm documents (round 4, N1). A spec-silent reading, ruled (JC3, revised by the coordinator after round 4; the earlier "largest inputs cost" text is withdrawn; ruled by coordinator, reversible): Phase 1 adds one clarifying sentence to VIA 8.3.1;
  (4) a recipe of duration 1 writes no job (REF 4.4.1: "the one-tick recipes have no job
  in progress at tick 0"; VIA 8.5.2 as written gives `V` = {0} and a positive share,
  which contradicts the document, so 8.5 gains the rule); (5) `InProgressBP` rounds down
  (28.125 / 55 = 5 113.6 gives the file's 5 113; 2.301 / 55 gives 418.4 and 418; the
  scale-down of D121 already says down);   (6) no pool row is written (the documents have
  none); (7) a commodity that appears in no input, tool, non-discard output or upkeep of the
  programme (`waste`, a discard output only) has no row in the table and no `UnpricedCommodities`
  entry (`warm-one.toml` has no `waste` price and REF expects `unpriced = []`; round 2, N2);
  (8) `second`'s `grow_algae` share 5 163 (executions in progress = `x1 * 12`, the recipe being open on
  every tick in both seasons, so both formulas of (9) coincide; share = 10 000 * x1 * 12 / 600, so the
  document's figure needs `x1` = 25.816): the restored start rule (A3, the prototype's) gives `x1` =
  26.160 and a share of 5 232 (measured, round 2; the withdrawn JC1 rule gave 25.641 and 5 128). The
  wage 1 966 and every other row of `warm-second.toml` are the restored rule's (A3), so the document
  stands except possibly this one row. Its producer is not archived (N-274), so the cause is not known (a
  different vertex of an alternative optimum is the first suspect: `Report.AlternativeOptima` for `second`
  is read first). Phase 7 compares the row; if it differs it is the one listed difference, handled as
  D133 says (the solver is the reference for a flow, the row is regenerated with the testing.md 7.2
  explanation). This is a measured fact of the document, not a status of a ruling. (9) **a seasonal job** (round 3, N1; ruled by the coordinator, reversible): `warm-two`, `warm-three` and `warm-five` write for every `grow_staple` job
  row `floor(10 000 * x1 * 60 / growers)`, that is executions in progress = `x1 * d`, with `V` used only for
  emptiness and the offsets (no job where `V` is empty, 8.5.2); VIA 8.5.1 as it stands gives `x1 * TicksPerYear * |V| / N_open`, which
  differs for a recipe that does not run all year: `two` south 3 341 in the document against 3 110.5, `three` south 3 464 against 3 224.7 and
  far_south 1 736 against 3 250.8, `five` centre 3 766 against 3 506.0, east 4 124 against 3 838.9, south
  2 506 against 4 692.0; six job rows in three packs, none visible in `one`, `second` or `famine`, where the recipe runs all year and the two
  formulas coincide. **Ruling: VIA 8.5.1 as written decides.** The spec is normative and the warm documents
  are derived output: FR-REF-007 defines them as the solver's output, so a document that differs from the
  spec's formula is the stale one. Reason: the as-written formula spreads the executions over the open
  start ticks at the rate the steady state has (the same rate D123 uses), so the jobs in progress at tick 0
  match the flow, whereas `x1 * d` assumes a start at every tick. The six rows are regenerated in Phase 7
  (`warm-five.toml` wholesale, O11) with the testing.md 7.2 explanation; Phase 1 writes no change to 8.5.1
  for this beyond the lambda-1 scaling of D120.
  No money difference is expected from the main recipe: under the JC3 rule every group of the five warm documents loads equal, so a money mismatch in Phase 7 is a defect to explain, not a listed difference.
  Written into VIA 8.3.1, 8.4.1, 8.5.1, 8.5.2; Phase 7 compares the result with
  the documents.

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The signatures of `internal/steady` are the implementer's, within VIA
  and architecture.md 2.9. Names in this plan are indicative of the capability, not
  an interface promised to other rows, except the three the world and golden rows
  consume (Cross-plan contract).
- **A2.** The tableau is integer-preserving: one common denominator (the previous
  pivot, the Edmonds/Bareiss form) or one denominator per row, whichever the
  implementer measures smaller on `five`; the certificate (Phase 2) is what makes
  either right. `big.Int` only; `big.Rat` only at the boundary. No pool, no
  package-level cache (architecture.md 7.1). The **canonical column order** is the
  prototype's (`models/via/proto3.py`; the 24 pivots of the three-region pooled chain hold only in the
  prototype's own insertion order, the same chain with regions, recipes and commodities sorted by ID as
  FR-VIA-022 says gives 13, so the `steady_test` case builds its columns in the prototype's order
  explicitly (the package takes opaque indices) and the 24 is never derived through the sorted
  conversion; the pack counts 10, 19, 30, 202, 9 are in ID order and unaffected): the activities (region, then recipe), `lambda`,
  then one slack per row; the entering variable is the lowest index with a negative
  reduced cost and a tie in the ratio test goes to the lowest basis index. Bland's rule
  makes the pivot count representation-independent only for the same indexing, so the
  pivot counts of Phases 3 and 7 (11, 10, 18, 24, 19, 30, 202, 9: the prototype's and REF 7.3's) are claims about this
  order (round 1, N3).
- **A3.** The greedy start count is run-length based over the year's constant
  stretches (the seasons), not a loop over ticks, so a pair with a profile costs
  O(seasons + starts per run) and a pair without one is closed form
  (`ceil(TicksPerYear / d)`, 7 for a 60-day job, or 0 for a sited recipe). **A start counts
  at every tick of the open window** (the prototype's `t < YEAR` loop of `proto3.py`, which
  REF 7.3, FR-VIA-048 and the committed warm documents all use; ruled by the coordinator,
  reversible; the earlier JC1 reading, "a start counts only if `t + d` is at most the year",
  is withdrawn). Reason: the steady state is periodic, so a job begun late in one year
  completes early in the next, and the annual flow of a stationary seasonal activity is its
  starts per year, the year boundary being arbitrary. VIA 3.1's sentence that `J` is
  `TicksPerYear / d` rounded down, and the clause about completing inside the year, are a
  prose error; Phase 1 amends VIA 3.1 to the prototype's reading with this reason. Consequences:
  `J * d` may exceed the year by up to `d - 1` (the property is `(J - 1) * d` below
  `TicksPerYear`, Phase 3), and a recipe with no profile has `J * d` at least the year, so it has no
  capacity row (the row exists only where `J * d` is below `TicksPerYear`, D126). That is the
  prototype's Bland path, so **every number of REF stands**: chain A 2.980, chain B (`second`)
  1.966 with 10 pivots, chain B on two regions pooled 1.911 with 18 pivots, the three regions
  pooled 3.124 with 24 pivots (autarky 11 each), `one` 10, `famine` at full labour 9. The pivot
  counts of `two`, `three` and `five` (19, 30, 202) were measured (round 3, N5) with an
  exact-fraction LP loaded from the packs, same column order and Bland rule, and are asserted in
  Phase 7. The code and the differential test take the rule from one predicate, "does a start at
  tick `t` count": the factor at `t` is positive and the household is free. A differential test
  against the naive tick loop proves the run-length version (Phase 3). The root builds the
  per-season factor from `yieldTable.site` and `Calendar.SeasonAt` and passes primitives (the
  year as runs of `(length, factorBP)`); `internal/steady` knows no season.
- **A4.** `internal/steady` receives the net vector as primitives and multiplies
  in `big`: each activity as `Starts` (J), `FactorSum` (the sum of the factors at
  the starts, in basis points), the duration, the outputs `(commodity, quantity)`,
  the inputs, and the tools `(commodity, wearPerUse, durability)`. A product like
  `FactorSum * quantity` can exceed `int64` (10^10 times 10^9), which is the reason
  the multiplication is inside the package.
- **A5.** The slice's programme has every right-hand side non-negative (goods
  rows `... <= 0`, labour rows `phi * H`, capacity rows `H * J / TicksPerYear`), so
  a basic feasible start exists and no phase one is needed. The wants term of the
  demand module makes a goods right-hand side negative (3.2): `internal/steady`
  takes a per-goods fixed term, rejects a non-zero one with a programmer-invariant
  error in this row, and the demand row adds the two-phase start in its own PR
  (N-277).
- **A6.** Performance is measured, not assumed (N-061): Phase 7 records pivots
  and wall time of `five` pooled in Go against the 202 pivots and 1.7 s of the
  prototype, as the number behind `a10.seconds` and `MaxViabilityCells`.
- **A7.** The prototype of VIA (`doc/specs/models/via/`, `proto3.py`, `sweep.py`,
  `ricardo.py`) is run only in Docker, as the specs say, with
  `PYTHONDONTWRITEBYTECODE=1` (N-103), to cross-check lambda, prices and the
  three-region ratios. The producer of the warm documents is not archived (N-274),
  so the money, stock and jobs are checked against the documents themselves and by
  hand arithmetic.
- **A8.** One PR per phase, in phase order, on a branch from `main` (row 6 is merged, e135fcf) or the then-current Stage 1 head.
  Phases 2 to 4 are pure and may be written while Phase 1 is in review, but they
  implement spec text (D122, D123, D126 and D130), so none is merged before Phase
  1. The row shares `doc.go`, `doc/notes.md`, `doc/documentation_index.md`, the
  roadmap and `AGENTS.md` with the other rows: each PR appends its lines and
  re-reads the shared file just before editing it.
- **A9.** The tests of the root file run from internal tests (package
  `economysim`) on small configuration literals; whatever needs the packs runs
  from one external test (package `economysim_test`) that loads them through
  `config/` and reaches unexported functions through `export_test.go`, as plan
  1-6 A8 says (an internal test cannot import `config/`, which imports the root).

## Phase 1: settle the owed spec lines and the notes

**Why first:** D120 to D134 are rules the code implements, and the result-affecting
ones (the flow scaling, the supply gap, the reserve, the shape of the error) would
otherwise be choices made in code where the configuration hash and the golden files
cannot see them (architecture.md 9.5). This phase changes specs, rules and notes
only, so it starts by reading `doc/notes.md` in full and quoting its tag, handles
every open note tagged VIA, CFG, ALL and CODE (below), and gives `doc/notes.md` a
fresh tag (`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **VIA 5.1, 5.2, 5.3 (items 1 and 4), 7.1, 8.5.1, FR-VIA-018** (D120, D121, D128
   to D131). The flows of the table are those of lambda 1; the shared recipe rule;
   the exported shape of the result and the default autarky; the empty basket per
   programme (FR-VIA-018 itself is not changed; JC4 is ruled, D130); the nominal
   fallback per programme. No requirement is renumbered (testing.md 4.1).
2. **VIA 4.5 and 8.4.1** (D122, D123). The supply gap from the greedy start sequence of
   `J` (VIA 4.5 already says so: add the worked check, starts 59 and 119, completions 118
   and 178, gap 365 - 60 - 1 = 304 for `one`; the longest run of an all-open-ticks
   reading would be 273 and is not the rule) and the formula of the lean reserve over the
   whole consumption, with the worked numbers of `staple` in `one` (first completion at
   tick 118; 1 380.68 a tick of use; 644 per grower plus the 20 of `save_seed`'s input,
   664 in `warm-one.toml`; recompute from `one.toml` before writing).
2a. **VIA 8.3.1, 8.4.1, 8.5.1, 8.5.2** (D134). The nine composition rules pinned against
   `warm-one.toml`, `warm-second.toml` and `warm-famine.toml`, with `one` as the worked
   example; the "main recipe" rule is JC3 (longest duration, ties by largest inputs cost, then the lower recipe ID; ruled by coordinator, reversible); reading (9) (the seasonal job keeps 8.5.1 as written, ruled) is compared against `warm-two`, `warm-three` and `warm-five` too, and the six rows that differ are listed for Phase 7.
2b. **VIA 3.1 and FR-VIA-002** (A3, ruled by the coordinator, reversible; JC1 reverted). Amend
   the sentence "`TicksPerYear / d` rounded down" to "rounded up: a start counts at every tick
   of the open window, so a job begun late in the year counts", with the reason (the steady
   state is periodic: a job begun late in one year completes early in the next, and the
   annual flow of a stationary seasonal activity is its starts per year, the year boundary being
   arbitrary). FR-VIA-002 stays and gains "whether or not the job completes inside the year".
   The numbers of VIA 15, REF 7.3 and FR-VIA-048 are not touched.
2c. **VIA 3.2, 4.2.1, 4.4** (JC5, round 1 B3; the nominal warnings, ruled). What the programme does with a commodity
   marked `Raw` that no existing activity sources (ruled, JC5): V3 and the table are skipped
   for a programme whose basket goods are all unsourced `Raw` markers (a `Raw` good is sourced
   everywhere, 4.2.1, and the pack loads with the warning `raw_without_source`). VIA 4.4 gains the
   basis of the two nominal warnings (ruled by the coordinator, reversible: no invented quantity):
   both are per region, from the region's autarky programme; `target_above_surplus` compares the
   household-weighted mean `TargetRatio` of the region's groups with the `SurplusBP` of the region's
   **nominal** autarky programme (`phi` = the expected work share); `subsistence_exceeds_expected_work`
   compares the region's `LabourShareBP` with the mean expected work share, and **when the labour
   share is absent (FR-VIA-012: a capacity row is tight) the warning is not raised**; a region with
   no programme (empty basket, no households) raises neither.
   **V3 gate (JC9, O12):** VIA 4.1 and FR-VIA-007 gain the condition: V3 (`labour_short`) is not run when V1
   found `unproducible_in_realm` or V2 found `cycle_not_productive` (the programme is then lambda 0 for a
   structural reason, REF 5's d1 and d2), so FR-VIA-007's "whenever pooled lambda* < 1" reads "... and V1
   and V2 found nothing"; the FR-VIA-007 test and the implementation use that one text. REF 5's `famine` expectation
   (`subsistence_exceeds_expected_work` at 76.0%, `target_above_surplus` at 1.112) is a Stage 2
   matter: `famine` does not load in Stage 1, and its programme at full labour has a tight capacity
   row, so the labour share is absent there; REF's warnings paragraph gets a sentence saying so.
2d. **REF 5 defect table, row d2** (O12, ruled). The Defect column gains "and `fuel` is not marked
   `raw` in the overlay": `base.toml` marks `fuel` `raw`, a `Raw` good is sourced (VIA 4.2.1,
   FR-VIA-059), and d2 must be rejected by `unproducible_in_realm` as REF 5 says, so the overlay
   changes (Phase 5), not the rule. VIA 4.2.1 and the Stage 0 rule are unchanged.
2e. **VIA 6.4 and FR-VIA-023, 026** (O11, ruled). A seed price of the table is lifted into the reference
   band: a price floored by the resolution floor, or a free good (dual 0), is raised to the lower
   bound of the reference band that `Validate` enforces (`warmRules`), and a price above the upper
   bound is capped to it; each lift is listed in `Floored` and the unclamped price is recorded in
   `OutsideReferenceBand`, so VIA 6.4's record is kept while the table passes `Validate`. The
   validator's band rule is unchanged. **This also amends VIA 8.2.4 and FR-VIA-026 for a free good:** a
   dual-0 good takes max(resolution floor, band lower bound) instead of the reference price (a
   non-unique price keeps the reference price); it stays listed in `UnpricedCommodities`, and the
   `Floored` and `OutsideReferenceBand` records are written for it as for any lift.

3. **VIA 5.2 and FR-VIA-028** (D132). The census formula, with the worked example of
   `one` (the 600 households of `one.toml`, not the prototype's).
4. **VIA 4.3, 4.6, FR-VIA-010 and CFG 7.6 item 1** (D126). The size gate arithmetic
   and what a skipped part is called in the warning.
5. **VIA 7.1, 9.4, 11 and CFG 7.1.1, 7.1.6, 7.2, 7.6 item 2** (D124, D125, D127). The
   five option names and bounds, the sentence that Stage 1 realises only the free-trade lane set and
   that `Lane`, `Lanes` and `MaxLanes` come with the transport module (and that the collapsed single
   goods row is the realisation of FR-VIA-043's and 044's complete free-trade set, whose lane columns are
   TRN's), `ViabilityError`'s fields, the path, value, bound and count
   of each rule (D124, into VIA 10.1) and its place in the one sorted list, and the gate
   of `Warnings`. CFG 7.2 says that `Warnings` appends the
   viability warnings only on a configuration whose structural phases pass.
6. **CFG 7.4.1 and VIA 8.7** (the empty-offset rule). Say where in the order of
   phases the rule "the set of virtual start offsets of a job is empty" runs
   (`ErrWarmStartJob`, FR-VIA-033): the entry of phase 6, on the canonical copy of a
   configuration that phases 1 to 5 found clean (D125, round 1 B4), because it needs the
   recipe, yield and role tables, which exist only for a canonical, validated
   configuration. The sub-rule "shares of
   a group above 10 000" is already built in `warmRules` (Stage 0 listed it as
   deferred; check the test and correct the Stage 0 table in a note, not the plan).
7. **`bounds.go` rows in CFG 4.1** are already specified (`MaxViabilityPivots`
   100 000, `MaxViabilityCells` 4 000 000); check the text and add
   nothing. Stage 0 did not build these two of FR-CFG-046's list; `MaxLanes` waits for `WithLanes` (D127, N-281).
8. **`architecture.md` 2.6 and `AGENTS.md` layout.** Add `viability_check.go` and
   `viability_table.go` beside `viability.go` (one concept, three files for the
   Lizard limits: N-243), and `internal/steady` is already listed. Give
   `architecture.md` a fresh RULE-TAG in the same change (its own rule). This edit
   is shared with the other rows of Stage 1: re-read the layout block just before
   the edit and make it a small anchored `Edit`.
9. **`doc.go` glossary list in `doc/notes.md`** (N-017): the terms this row's code
   uses go to the `doc.go` glossary at the start of Phase 2, before use: viability,
   steady state, warm start (already there), surplus factor, activity, start
   capacity, mean factor, net vector, reference programme, labour share to
   subsist, available work share, nominal programme, census (balanced),
   certificate, seed price (as the solver's), drift, residual, supply gap, lean
   reserve, virtual start offset, lane. Phase 1 lists them in N-017's update.
10. **The roadmap and `doc/documentation_index.md`.** The roadmap row reads "plan
    audited" with the link (this plan's first edit); the index has one row for this
    plan, written like the 1-6 row, at the end of the plans group.
11. **Notes.** The disposition of every open note tagged VIA, CFG, ALL or CODE is in
    the table below and is written into `doc/notes.md` in this change, with the nine
    new notes (N-274 to N-282, listed in "New notes"). Phase 1 re-reads the index
    for notes raised since this plan was written and adds a row for each.

**Notes disposition** (apply, reject or leave open, with what was learned):

| Note | Disposition |
|------|-------------|
| N-015 Placeholders to calibrate once something has run (VIA, STY, CLB, EXP) | **Leave open, with a measurement.** `StockCoverTicks` 5, `MoneyBufferTicks` 10, `ResidualWarnBP` 1 000, `MaxViabilityPivots` 100 000, `MaxViabilityCells` 4 000 000 stay placeholders; Phase 7 measures the Go speed of the solver (the first measurement ever) and records pivots and time of `five` in the note. No number is claimed stable |
| N-016 Keep the parameter registry true (ALL, CFG, WEL) | **Leave open.** This row adds no `Config` field and no row of the parameter-owner table: the solve options are rows of CLB's registry (VIA 11, FR-CFG-047) and `WarmStart` has its row. Phase 5 checks the registry test is unchanged |
| N-017 Words and names (ALL) | **Apply.** The terms of deliverable 9 go to the `doc.go` glossary before use; "lane" is qualified (the lane of the solver is not the lane of a credit round or of a merchant's lane); "census" is qualified (balanced census). The note stays open |
| N-027 No default macro constant is stable (ALL, REF, STY) | **Leave open.** The table is a start, not a stability claim (VIA 13.1); no test of this row asserts a default as stable |
| N-061 The load-time size gate is a guess (VIA, CFG) | **Apply in part.** Phase 7 measures `five` pooled; the gate constants stay placeholders; the note gets the measurement. NFR-CFG-001 is checked on the packs with the programmes on and skipped |
| N-071 One wage per region and skilled recipes (VIA, SKL) | **Leave open** for SKL. The solver reports one wage per region (VIA 8.2.5); no per-class wage is built. Learned: `Report` appends a per-class field if SKL decides |
| N-075 Mapping the ledger's sentinels (CFG) | **Apply for this row's sentinels.** `ErrNotViable`, `ErrNoUpkeep`, `ErrSolveLimit`, `ErrSolveInconsistent` and the type `ViabilityError` are root sentinels in `errors.go` (D124); `ErrWarmStartPrice` and `ErrWarmStartJob` exist. `errors_test.go` and the declared-sentinel test gain the four. The note stays open for the other owners |
| N-080 Wants in the programme use equal-price shares (VIA, DEM) | **Leave open** for DEM. Learned: the slice has no wants term (A5); the demand row adds the fixed term and the two-phase start (N-277) |
| N-085 Nominal table against the full-labour table (VIA, STY) | **Apply in part.** The solver builds both (D131) and reports `NominalSeed`; the comparison on `famine` is STY's run. Phase 7 asserts the nominal table of `famine` as `famine.toml` gives it: staple yield `spring = 3000`, nominal wage **1 624**, which is the value of `warm-famine.toml` (the 1 826 of VIA 15 is the variant with a yield of 3 800; VIA 15 lists 1 868, 1 826, 1 624 and 755) |
| N-093 Inputs the wave 2 specs offer the viability check (VIA, CFG) | **Leave open** for TRN, REG and MON. Item (1): the pooled check uses free trade across all regions; the use of strongly connected components waits for the region graph. (2) and (3) are lanes and a hoarder class, both module inputs |
| N-109 Closed forms need the welfare ratio at a full draw (POP, VIA) | **Leave open** for POP and RES. `r_full` is not computed in the slice (no endowment, no population module) |
| N-114 The open layer of the steady state (VIA, REF) | **Leave open** for EXT. The programme is closed; the type keeps no import column |
| N-123 A cold start is a lockstep (CFG, VIA, REF, STY) | **Leave open.** The warm start's stagger is the world row's (8.4.3); this row gives the offsets. The warning `cold_start_lockstep` exists (Stage 0) |
| N-132 A population with no demand module has no healthy start (CFG, POP, STY) | **Leave open** for POP |
| N-134 A sweep that moves the steady state needs a regenerated warm start (VIA, EXP, CLB) | **Leave open** for EXP. The list of paths in VIA 11 is unchanged by this row; Phase 7 reads it against the inputs the code reads (a path the code reads that the list lacks is a defect of one of them). The open question of `world.belief_step` is answered in the code: only the resolution floor of a seed price reads it (VIA 6.4) |
| N-146, N-158, N-165 The warm start has no claim, no hire, no rent column (VIA, REF, CLB, CRD, CFG) | **Leave open** for CRD, EMP and LND. Learned: the table type carries none, and this row adds none (D5) |
| N-180 The expected loss is in the shipper's margin and the lanes (VIA, TRN, MER) | **Leave open** for TRN and RSK. `Lane` and its `LossBP` are not published in Stage 1 (D127); the programme reads a loss only when TRN adds them |
| N-103 Running several spec writers in parallel (ALL) | **Leave open.** Process note; this plan was written beside five others and reads only their contract |
| N-126 A new event or `Warn` adds a row (ALL) | **Apply.** This row adds no event and no `Warn`: the solver logs nothing and the warnings are the existing "configuration warning" line with `rule` and `path` (CFG 7.2). Each new warning rule (`region_not_self_sufficient`, `subsistence_exceeds_expected_work`, `target_above_surplus`, `storage_gap_input`, `viability_check_skipped`, `cycle_cost_gain`; `raw_without_source` already exists) is a row of CFG 7.4.2 already |
| N-128 A hosted per-agent field is saved once (ALL, CHK) | **Leave open.** The table is configuration; the solver has no state (VIA 18) |
| N-163 The word "treasury" (MON, FIS, ALL) | **Reject for this row.** The word is not used |
| N-217 D5 binds every module's change (ALL, CODE) | **Apply.** The bounds and the sentinels are appended in order; no `Config` field, event or snapshot row is appended; `TestHashConfig_ReferencePinned` must not move; Phase 7 checks it |
| N-235 One combined architecture.md edit for the fan-in and admission touch points (ALL) | **Leave open.** Not this row's: no fan-in, no admission. The layout edit of deliverable 8 is separate and small |
| N-019 Housekeeping for the first code change (CODE) | **Leave open.** The glossary, the sentinels and the layout of this row are done as above; the CI and the ADR acceptances stay plan 0's |
| N-089 Model scripts of the demand, spawner and reference-chain specs (DEM, SPN, REF, CODE) | **Leave open**, with a learning: the REF prototype that wrote the warm documents is the one that is not archived (N-274) |
| N-127 Event classes at the observer (OBS, EXP, CODE) | **Leave open.** No event here |
| N-198 The drawdown is `O(T)` per agent and tick (SCL, CODE) | **Reject for this row.** Not a solver concern |
| N-214 Keyed random streams: the edits owed (..., CODE) | **Reject for this row.** The solver draws nothing (FR-VIA-020, FR-VIA-040) |
| N-215, N-216, N-222 (CHK/WLD/SCL, CODE) | **Leave open.** No site, no reseed, no per-agent key |
| N-218 The Stage 0 field set and the hash rule (CFG, CODE) | **Leave open.** `WarmStart` is hashed as Stage 0 built it; this row changes no hash |
| N-224 Stage 0 guards and what they pin (CODE) | **Apply.** `validate_counts_test.go` and `errors_test.go` gain the two bounds' checks (they are not count bounds of a `Config` collection: the test reads them as constants) and the four sentinels; `import_guard_test.go` gains the rule that only `internal/steady` imports `math/big` **in its non-test files** (seven existing test files use it as an exact oracle: `internal/welfare/fuzz_test.go`, `ring_test.go`, `internal/checked/checked_test.go`, `acc_test.go`, `internal/clearing/admit_test.go`, `internal/beliefs/update_test.go`, `internal/ordering/ordering_test.go`; architecture.md 3.5 restricts computing, not testing); `determinism_guard_test.go` must pass for the new files |
| N-228 A module's items in the tradable set have no touch point (AGT, DEM, SUB, CODE) | **Leave open.** The solver reads the role table's tradable set only through V1's producibility, with no new touch point |
| N-232 The job and the residues live on the agent (PRO, AGT, CODE) | **Leave open** for the agent row. `virtualStartOffsets` and `warmJob` are what the world row calls |
| N-243 Follow-up of the Codacy gate on plan 1-5 (CODE, PRO) | **Apply.** The Lizard limits are a design input of Phases 2 to 6 (functions of complexity at most 10, at most 8 parameters, files of about 500 lines); `lizard -l go -C 10 -a 8 -L 100 -w` runs before each push; the layout edit above names the new files; markdown tables of this plan use the spaced separator. `internal/checked.muldiv` (CCN 13) is untouched unless this row edits it |
| N-299 The defect-file tests flip with the viability rules (CFG, REF) | **Apply.** Phase 5 flips the d1 to d5 line of `TestPacks_DefectOverlays` from nil to `errors.Is(err, ErrNotViable)` in the PR that adds phase 6; plan 1-12 adds the rule names and the "that rule only" assertions. The note stays open for row 12 |
| N-284 The type `World` is the configuration section (CFG, CODE) | **Leave open, with what this row learned.** Row 11 renames the section to `WorldParams`; this row reads it through `Config.World` only (see the Consumes table) |
| N-244, N-245, N-247, N-248, N-254, N-258, N-285, N-286, N-287, N-290, N-294, N-295, N-297, N-300 (CODE tag, raised after this plan) | **Left open, not read as touched.** They concern the agent, estate, world, checkpoint and golden rows; none changes a symbol this row consumes or provides. N-290 (a module that adds a reduction cause appends its ledger reason with it) is the reduction cause set's, not this row's. N-300 (the first measurement of a `Step`) is not this row's: the solver is outside the tick path |
| The other open notes tagged CFG (N-075 above, N-182, N-221) and CODE (N-224 above) | **Left open, not read as touched.** N-182 (region-keyed tables across two modules) and N-221 (STO layout) concern module rules |

**New notes** (IDs N-274 to N-282; they already stand in the working-tree `doc/notes.md`, so Phase 1
verifies each against the statements here, completes the index and fresh-tags the file as AGENTS.md
requires, quoting the tag current when it starts; their text is the one-line statements here):

| ID | Areas | Statement |
|----|-------|-----------|
| N-274 | VIA, REF, CODE | The producer of the `warm-*.toml` documents (REF's prototype `lp.py`, `/tmp/economy-boss/ref/`) is not in the repository; the Go solver is the reference once it passes Phase 7, and a document that differs is regenerated with the testing.md 7.2 explanation. Archive nothing from `/tmp`; the specs' `models/via/` cross-checks lambda and prices only |
| N-275 | VIA, REF | The supply gap is read from the greedy start sequence of `J` as VIA 4.5 says (D122: `one` gives 304, an all-open-ticks reading would give 273; a start counts at every tick of the open window, A3); the lean reserve is the largest cumulative shortfall of the whole consumption, upkeep plus recipe inputs (D123: 644 plus 20 in `warm-one.toml`); REF's wording of the reserve is the special case |
| N-276 | VIA, CFG, WLD, CODE | `Validate`, `Warnings` and `NewWorld` each reach the programmes; with `NewWorld` calling both the viability work is done twice. The world row may call the unexported `checkViability` once; Phase 7 measures the cost |
| N-277 | VIA, DEM, CODE | The programme with the wants has a negative right-hand side and needs a two-phase start; `internal/steady` takes the fixed term and rejects a non-zero one until the demand row adds it |
| N-278 | CODE | The viability files (`viability.go`, `viability_check.go`, `viability_table.go`) and the internal files are split for the Lizard limits; a module that appends a column or a row appends to these files and keeps the limits |
| N-279 | VIA, TRN | Stage 1 supports only the free-trade lane set (`WithFreeTrade`); `Lane`, `WithLanes` and `MaxLanes` are not published here and come with `transport.Lanes` (Stage 2), whose lane validation is an error; a lane with `Labour` 0 and `LossBP` 0 is the free-trade case |
| N-280 | VIA, WLD, AGT | `virtualStartOffsets(...)` is the one function that gives the open offsets of 8.5.2; `validate.go` (the empty-set rule) and the world row (the stagger of jobs) both call it, so the rule and the construction agree (FR-VIA-053) |
| N-281 | VIA, CFG | Stage 0 built `MaxSeedPrices` but not `MaxViabilityPivots`, `MaxViabilityCells` and `MaxLanes` of FR-CFG-046; this row adds the first two to `bounds.go` (`MaxLanes` waits for `WithLanes`), and the Stage 0 table that lists the viability rows as deferred is corrected |
| N-282 | VIA, REF | A recipe on the lists of several roles of a region divides its executions equally per household across those groups (D121); the census flags it ambiguous. REF's packs have no such recipe (checked in Phase 5); the rule is for later packs |

**Tests:** none; documents only, except that the notes index and `doc.go` do not
move code.

**Observability and rollback:** none, because no code changes. A wrong edit is
reverted like any document change; nothing depends on it until Phase 2.

**Acceptance:** VIA, CFG, `architecture.md` and `AGENTS.md` carry the lines above;
every open note tagged VIA, CFG, ALL or CODE at that time has a row (the closing
row names the ones left open untouched); the nine new notes are in the index; the
roadmap row reads "plan audited" with the link and the index has the plan row; the
notes carry a fresh tag; a text search finds no stray shell-quoting sequence
(N-103).

## Phase 2: the exact simplex and its certificate (`internal/steady`)

**Why here:** every layer after V1 is a programme over this core, and it is a closed
pure function that can be fuzzed alone with hand-made programmes.

- **First, `doc.go`**: the glossary terms of Phase 1 deliverable 9, before any code
  or comment uses them (architecture.md 3.2). The glossary guard
  (`glossary_test.go`) must pass at the end of this commit.
- The package `internal/steady`, with a doc comment that says it holds no state,
  knows no commodity, region or role (only indexes), imports the standard library
  (`math/big`) and `internal/checked`, and returns primitives (architecture.md
  2.9, 3.5).
- **The programme type**: `Programme{Rows, Cols}` with a signed fixed term per
  goods row (zero in the slice: A5), the columns as sparse primitive vectors, and a
  right-hand side per row. Entries are rationals given as `(num, den)` `int64` pairs
  or as the structured activity form of A4 for Phase 3; this phase tests the core
  on plain pairs. `Solve(ctx, p, maxPivots)` returns the primal values, the dual
  values, the objective and the pivot count as exact fractions in an internal type
  (a `*big.Int` numerator and denominator), never exported to the root as big.
- **Bland's rule** over a tableau in canonical order (rows then columns by index),
  the entering variable the lowest index with a negative reduced cost, the leaving
  variable the lowest-index among ties of the minimum ratio (FR-VIA-016, 022);
  integer-preserving (A2). The context is checked between pivots (FR-VIA-017); the
  cap is `maxPivots`; both return an error and no partial result. `ErrSolveLimit`
  and the wrapped `ctx.Err()` are the only non-defect failures.
- **The certificate** (FR-VIA-014, FR-VIA-013): recomputed from the original
  programme, not from the tableau: primal feasibility (every row, every variable
  non-negative), dual feasibility (every column's reduced cost of the right sign, the
  duals non-negative) and equality of the primal and dual objectives. A failure is
  `ErrInconsistent` (the root maps it to `ErrSolveInconsistent`; it cannot be caused
  by input and is a defect).
- **The boundary**: `RoundHalfEven(num, den)`, `Floor`, `Ceil` over big to `int64`
  with the check against `int64` and a caller's bound, returning `ok` (architecture.md
  3.5: "reports overflow to the caller"). The only functions that return an `int64`
  from a big value are these three.
- **Lizard**: each function at complexity 10 or less, at most 8 parameters; the pivot
  loop is split into `choose`, `ratio`, `pivot` and `certify` helpers; the file split is
  `doc.go`, `tableau.go`, `simplex.go`, `certify.go`, `round.go`.

**Tests** (unit, external package `steady_test` for the exported functions; internal
only for the pivot helpers that need unexported access, architecture.md 10.2):

- Hand-made programmes with the arithmetic of each expected value in a comment
  (testing.md 5.5): a two-variable programme, a degenerate one (a basic variable 0),
  one with alternative optima, an unbounded one reported as an error, an infeasible
  start refused, and the one-activity-one-commodity programme whose answer is `1 / n`.
- Bland's rule: the classic cycling example (Beale's) terminates; the answer is the
  same for two column orders only when the columns are the canonical order (the test
  states that the order is part of the input, FR-VIA-022).
- The certificate: a tableau with a deliberately corrupted entry fails with
  `ErrInconsistent`, so the certificate does catch a wrong pivot (a `Guard:` test:
  without it a bug in the pivot would return a plausible table).
- Cap and context: `maxPivots` reached returns `ErrSolveLimit` with the counts and no
  result; a cancelled context returns the wrapped error; both at the first pivot and
  at a later one.
- The boundary: half-to-even on the exact halves (`5 / 2 = 2`, `7 / 2 = 4`, `-5 / 2 =
  -2`), a value one past `MaxInt64`, a negative value, a zero denominator (a
  programmer invariant, not a result).
- **Fuzz**: `FuzzSolve` over small random programmes (`f.Add` seeds from the hand-made
  cases): never panics, every returned result passes the certificate, and a pure
  integer programme with a known optimum (a unit-cost transport case) matches it
  (testing.md 7.4; NFR-VIA-004). `FuzzRound` against `big.Int` division.
- **Determinism**: the same programme twice gives equal bytes (FR-VIA-020).
- Allocation: none required; the solver is outside the tick path (NFR-VIA-001) and
  the test comment says so.

**Observability and rollback:** none (observability entry: none, because the package
is pure and has no loop in the tick). Rollback is reverting the PR; nothing depends on
it until Phase 3.

**Acceptance:** the package passes its tests with `-race`; coverage of `internal/steady`
is at least 90% (testing.md 9.1); the import guard allows `math/big` in this package
only; Lizard is clean on the files.

## Phase 3: the activities and the reference programme (`internal/steady`)

**Why here:** it needs Phase 2's `Solve` and gives Phase 5 the start counts and the
programmes V3 reads, and Phase 6 the duals the table is made from.

- **The start count** (3.1, FR-VIA-001, 002, 053): `Starts(duration, year)` over the
  year as a slice of runs `(length, factorBP)` (A3) returns `J` and `FactorSum` by the
  greedy rule from tick 0 (start at the first tick whose factor is positive, be busy for
  `d` ticks, repeat; **the one predicate of A3: a start counts at every tick of the open
  window**, so a start whose job ends after the year end counts, and `ceil(TicksPerYear / d)`
  for no profile: 7 starts, at 0, 60, ..., 360, for a 60-day job);
  a pair with `J = 0` does not exist. The closed forms for no profile are tested
  against the loop under the same predicate. The function also returns the
  completion ticks of the greedy sequence (start plus `d` minus 1, not reduced: D122 takes them
  modulo the year for the gap, D123 drops those at or after the year end), which D122's gap
  reads.
- **The activity** (A4): `Activity{Region, Duration, Starts, FactorSum, Inputs, Outputs,
  Tools, Discard}` and its net vector built in `big` as rationals (denominators dividing
  `10 000 * Starts * Durability`); a discard output counts for nothing; a wear flow is
  `WearPerUse / Durability`.
- **The reference programme** (3.2): rows for the goods (one per commodity of the
  programme), the labour of each region (`phi * H`, `phi` an exact fraction: 1, or the
  household-weighted mean of the nominal case), the capacity of each activity with
  `J * d < TicksPerYear` (`H * J / TicksPerYear`),
  and `lambda`. It builds the autarky, pooled (the free-trade set collapsed to one goods row per
  commodity, no lane column or lane input: D127, D129) and nominal forms (D131), the
  order canonical (commodities by index, activities by region then recipe: FR-VIA-022),
  the size known before building (D126: `Size(p)` returns rows and columns).
- **The results**: `lambda*`, `SurplusBP` (floor), `LabourShareBP` (ceil, absent when a
  capacity row is tight: FR-VIA-012), the dual per commodity (price, with the
  normalisation `sum p D = 1`), per region (wage), per capacity row (rent), the flow `x1
  = x* / lambda*` as an exact internal value (D120), `Degenerate`,
  `AlternativeOptima`, the tight capacity rows and the activities active in each region
  (`Advantage`, FR-VIA-045, 019).
- **The family `W` hook**: the fixed term per goods row is accepted and must be zero
  (A5).
- **Lizard**: `program.go`, `activity.go`, `start.go`, `results.go`.

**Tests:** unit, in `steady_test`, with the arithmetic in comments:

- `Starts`: no profile (`d` 60 gives `J` 7, the last start at 360 completing at 419, past the
  year end), a sited recipe, an all-year profile, spring-only starting on day
  59 of 365 with `d` 60 (`J` 2: starts 59 and 119, completions 118 and 178), a window on days 340 to
  364 with `d` 60 (`J` 1, the completion at 399 is past the year end and still counts), two plantings a year, a
  profile whose first open tick is the last; the run-length count equals the naive
  tick loop on 200 random profiles and durations (a differential test with a fixed seed
  and the seed in the failure message, testing.md 6.2). FR-VIA-002.
- The net vector: one hand activity with `FactorSum` 18 000, `Starts` 2, an output of
  12 and a tool with `WearPerUse` 3 and `Durability` 2 000 gives the exact fractions
  (FR-VIA-001); the multiplication that exceeds `int64` is a case.
- The programme on the prototype's cases, with the values of VIA 15 and REF 7.3 **as the committed
  `proto3.py` yields them** (the start rule of A3 is the prototype's; run it in Docker with
  `PYTHONDONTWRITEBYTECODE=1`, take the exact fractions, and write the floored digits and the pivot
  counts into the tests): the rescaled chain A gives `lambda*` 2.980 (`SurplusBP` 29 804), 11 pivots;
  chain B rescaled **1.966**, 10 pivots; chain B on two regions pooled **1.911**, **18** pivots; the three
  regions in autarky 2.980, 2.517, 3.124 (11 pivots each) and pooled 3.124 with **24** pivots (in the prototype's listed order, not the ID-sorted one, which gives 13: A2); the
  unrescaled chain A `1/73` (from `proto.py`/`proto2.py`, "as written", 11 pivots to be read from that run;
  `proto3.py`'s "as written" keeps the rescaled steps and gives 116640000/52135357 = 2.237, so it cannot
  produce 1/73). FR-VIA-003, 043.
- Dual feasibility on every active activity in exact arithmetic (FR-VIA-013); the
  nominal programme with `phi` 250 / 365 differs from the full one (FR-VIA-058 as a
  unit: a capacity-tight region gives wage 0 at full labour and positive nominal).
- `Degenerate` and `AlternativeOptima` on a hand programme with a tie (A6 of VIA 13.2;
  FR-VIA-019); the report lists are in a stable order.
- Canonical order: the same programme with the activities given in another order
  returns the same result because the input is sorted by the caller's keys (the test
  sorts, and a second test shows an unsorted input is a programmer invariant: it panics
  with a message containing "invariant"; the determinism guard allows it).
- An AST guard in `internal/steady` (FR-VIA-015, round 4, N4): no `float32`/`float64`, no `big.Float`, no `Rat.Float64`, no `math` function in the non-test files of the package (the determinism guard does not detect float types; the guard is written here and extends to the root files this row adds in Phase 5 and 6).
- Fuzz `FuzzStarts` (never panics, the start count agrees with the naive loop under the
  predicate of A3; the property that replaces "`J * d` at most the year" (which no longer holds, a
  start near the year end counts): the starts are strictly increasing, each at least `d` after the previous,
  each at a tick of positive factor and before `TicksPerYear`, the first is the first positive tick, no further
  start is possible before the year end (so `(J - 1) * d` is below `TicksPerYear`), and the factor sum is at
  most `J * MaxYieldBP`) and `FuzzProgramme` (the certificate passes for every result).

**Observability and rollback:** none (pure); rollback is reverting the PR.

**Acceptance:** the prototype's lambda values and pivot counts (A3) are
reproduced to the digit; the differential test of the start count passes on all seeds; coverage at least 90%.

## Phase 4: V1, V2, V4 and the supply gap (`internal/steady`)

**Why here:** these are the cheap layers (graph, calendar scan, one small programme per
component) that `Validate` runs always; they do not need the table and unblock Phase 5.

- **V1** (4.2, FR-VIA-004, 005, 059): the producibility fixpoint on the existing
  activities (a tool is an input; a `Raw` commodity and a recipe with no inputs and no
  draws are sourced; an own-use-only recipe is no activity), realm-level (producible in
  some region) and region-level; it returns the unproducible commodities with one
  recipe naming each, and the roles that list recipes yet have households where none of
  their recipes exists. The module sources (RES draws, EXT import lines) are inputs the
  caller adds to the sourced set; none is built here.
- **The raw-without-source** finding (4.2.1, FR-VIA-057, 059): a commodity marked `Raw`
  with no existing no-input activity outputting it and no module source. **Stage 0 already emits it** (round 3, N4: `validate.go` `warnings()`, tests in `validate_mutation_test.go`, `ExampleConfig_Warnings`): this row reuses that rule, ungated and with its tests green, and emits nothing second; the V1 producer set only supplies the "existing activity" reading (Stage 0 tests "a role with households runs it") and a difference from the 4.2.1 wording is listed in the Phase 4 PR. Phase 5 adds no `Warning` for it.
- **V2** (4.3, FR-VIA-006): Tarjan's components over the material graph in canonical
  order (iterative, so the depth is not the number of commodities), and for each
  component with a cycle the programme `max s` with `sum x = 1` over the activities of
  the component (membership, round 3, N7: every activity with a non-discard output in the component, its net vector restricted to the component's commodities, so a source activity such as `gather_seed` belongs, and `d1` with the wild source passes V2 as REF 5 and plan 1-12 expect; Phase 1 writes it into VIA 4.3) at their best mean factor across regions, solved by Phase 2 and
  Phase 3's builder; `s* > 0` is productive. The cycle gain input for the warning
  `cycle_cost_gain` (`q_in / q_out` per cycle in exact integers, 7.6 item 3, FR-CFG-044; `q_in` and `q_out` are the
  **configured quantities**, with no yield factor, because 7.6 item 3 says the ratio "has no prices"
  and the factor depends on region and season; a working default, round 2, N5. Worked case for Phase 1
  and the `cycle_cost_gain` test: the seed cycle of `one`, 40 `seed` in and 1 800 `staple` out, then
  20 `staple` in and 20 `seed` out, is `(40 / 1 800) * (20 / 20)` = 1/45, so it never warns inside
  the `CostFloor` bound; the test uses a hand cycle with `q_in / q_out` 4/5).
  A component can hold exponentially many cycles and joint outputs, and 7.6 item 3 defines the
  ratio only "around the cycle (a product over the recipes of a longer cycle)"; which cycles
  of a component are enumerated and under what bound is ruled (JC6, ruled by the coordinator,
  reversible; reason: it makes the ratio well defined and the enumeration bounded): the simple
  cycles whose every commodity has **one maker in the component, where a maker is a non-source
  activity (one with inputs) that outputs the commodity**. Source activities (no inputs) are not
  counted, because a source supplies the commodity from outside the cycle, so the seed cycle of
  `one` (`gather_seed` is a source; `save_seed` is the one maker of `seed`, `grow_staple` the one
  maker of `staple`) has one maker per commodity and its 1/45 case is enumerated. The enumeration
  runs up to `MaxViabilityCells` steps, and a component past the bound is the
  part "cycle gain" of `viability_check_skipped` (D126); a unit case has a two-cycle component; a mini-chain with a no-input source of the cycle's commodity (the `d1`-with-wild-source shape: the cycle alone has `s* = 0`, with the source it is productive) pins the membership of V2, and a second unit case pins that the same source is not a maker for the ratio (a commodity with a source and one non-source maker is enumerated; one with two non-source makers is not).
  A component whose tableau exceeds the gate is reported skipped (D126).
- **V4 and the gap** (4.5, 8.4.1, D122, D123, FR-VIA-008, 039): the completion ticks of the
  greedy sequence of each activity, kept as runs over the season stretches (once per calendar,
  then O(seasons) per pair, not a mask of `TicksPerYear` ticks per pair: the bounds allow
  `MaxTicksPerYear` 100 000 times `MaxRegions` 1 000 times `MaxCommodities` 1 000, which V4
  must not multiply because it is never skipped, FR-VIA-010; round 1, N1), the longest cyclic run per
  (region, commodity) for **every commodity an existing activity of the region outputs** (the lean
  reserve needs the gap of any produced commodity, round 2, B2), the `storage_gap` and
  `storage_gap_input` findings (which look only at the pairs that are in an upkeep or an all-year
  input), and the lean reserve of D123 as an exact
  value (the flows come from Phase 3; this phase takes them as input; the cut of the cover to the shelf life, FR-VIA-039, is applied by the table in Phase 6).
- **Lizard**: `producible.go`, `components.go`, `cycles.go`, `gap.go`.

**Tests:** unit with the arithmetic in comments:

- V1: a seed cycle with no source is unproducible (and reports the cycle's commodities);
  a `Raw` marker makes it sourced; a crop that grows nowhere (a yield of 0 everywhere)
  is producible for CFG 7.5 and not for V1 (the difference of 4.2); a role with recipes
  and households where none can run; a role with no recipes exempt (FR-VIA-005); an own-use
  only recipe is no source; the producer of a tool's input counts.
- V2: a one-recipe seed cycle `1 seed in, 1 seed out` has `s* = 0` (not productive); `1
  in, 2 out` is productive; the best yield across regions is taken. The `s*` figures of
  VIA 15 round 5 (0.92 as written, 1.92 with the wild source, 0.5 and 0.20 for the board
  and fabricator) come from `lp.py`, which is not archived (N-274; none of
  `models/via/*.py` produces them), so a test asserts them only for mini-chains whose
  fractions Phase 1 derives by hand and writes into VIA 15 (the arithmetic in the test
  comment, testing.md 5.5), and otherwise asserts what `proto3.py` yields (round 1, N3).
  Tarjan's components on a graph of 1 000 nodes in a chain do
  not overflow the stack.
- V4: the gap of a spring-only crop in the default calendar (spring days 59 to 150, `d`
  60: greedy starts 59 and 119, completions 118 and 178) is 365 - 60 - 1 = 304, where the
  all-open-ticks reading would give 365 - 92 = 273 (the case that pins D122; a 61-day
  window gives 304 under both readings and does not tell them apart); two plantings (spring and autumn) give 123 (completions 118, 178,
  302, 362); a window on days 340 to 364 with `d` 60 starts once and completes at tick 34 of the next year, so the gap is 364 (the wrap of D122; the reserve credits nothing for it, D123); cross-check with REF 5: 304, 123, 184; 0 for an all-year recipe, 0 with no seasons; a
  shelf life of 200 against a gap of 304 is `storage_gap` (the d5 case) and 700 passes;
  the cyclic run wraps the year end; the reserve for `staple` in the `one` numbers
  (D123, 118 ticks), in `famine` (393) and in `two` south, two windows (these cases are `internal/steady` unit tests on hand-built primitive activities with the numbers of the packs typed in, since the packs load only through `config/`, which the internal package cannot import; the file-level agreement is Phase 6 and 7; two south 789: the autumn starts complete after the year end and add nothing, N2); a reserve above the shelf life is cut and listed. An all-year recipe of `d` 2
  or 3 or 12 (`refine`, `make_tool_a`, `make_tool_b`, `second`'s `grow_algae`) has gap 0 and no reserve
  (the three documents have none for `refined`, `tool_a`, `tool_b`, `biomass`); the per-household
  figure rounds up (`warm-famine.toml`: 1 350 * 118 / 417 = 382.01 gives 383, plus the 10 of the input;
  `warm-one.toml`: 664 = 644 + 20).
- Fuzz `FuzzChecks`: never panics and the three results are deterministic on any small
  random graph.

**Observability and rollback:** none (pure); rollback is reverting the PR.

**Acceptance:** the V1 to V4 findings of REF's tables (REF 7.3: `one` and the five other
packs pass, d1 to d5 fail with their rules) are reproduced at the unit level on hand-made
mini-chains that mirror them (the file tests are plan 1-12's); coverage at least 90%.

## Phase 5: the root: the rules in `Validate`, the sentinels and the warnings

**Why here:** it needs Phases 3 and 4 and is what makes the row visible to `Validate`
and to the world row. It touches `validate.go` and `errors.go`, which are catalogue
files; the conversion is inline at the call (architecture.md 2.9).

- **First step: the inventory of what phase 6 breaks** (round 1, B3, N12). Before the rule
  is written, grep every `Validate() == nil`, `Validate()` `err != nil` and `Warnings()`
  expectation in the root and in `config/`, and list in the PR what changes and the fix per
  fixture. Known at the time of writing: `minimalConfig()` (`validate_rules_test.go`; about 24
  uses in `validate_branches_test.go`, `validate_rules_test.go`, `validate_mutation_test.go` and
  `registry_test.go`) and `exampleConfig()` (`example_test.go`, with `ExampleConfig_Validate`
  printing "valid: true" and `ExampleConfig_Warnings` whose `// Output:` lists exactly two
  warnings) are one `Raw` upkeep good and a role with no recipes: V1 passes (the marker is
  sourced, VIA 4.2.1), but no activity exists, the pooled `lambda*` is 0 and `labour_short`
  would fire. The treatment (ruled, JC5) is that V3 and the table are skipped for a programme whose
  basket goods are all unsourced `Raw` markers, written into VIA 3.2 and 4.2.1 in Phase 1;
  this PR implements it, so these fixtures keep validating, and still lists every fixture
  phase 6 affects. The `referenceConfig()` mutation tables ("breaks only this
  rule", `validate_test.go`, `validate_mutation_test.go`) are exposed too: a mutation that
  removes a recipe or a yield can now add an `ErrNotViable`, so each such case is re-read and
  gets the exact-rule assertion it had. The loops that skip a pack that stops validating
  (`role_pack_test.go` and `welfare_pack_test.go`: `if raw.Validate() != nil { continue }`)
  would silently drop a defect or overlay that becomes non-viable, so the external test below
  asserts the list.
- **`errors.go`**: `ErrNotViable`, `ErrNoUpkeep`, `ErrSolveLimit`, `ErrSolveInconsistent`
  and the type `ViabilityError` (D124) with `Error()` text built like `ValidationError`'s
  (path and value cut and quoted by the same helpers, so a hostile ID cannot flood a log),
  and `SolveError` carrying the pivots, rows and columns. Appended in order (architecture.md
  2.10.5).
- **`bounds.go`**: `MaxViabilityPivots` 100 000, `MaxViabilityCells` 4 000 000
  (FR-CFG-046), appended, each with its derivation in a comment (`MaxLanes` waits for `WithLanes`, D127). The count test of N-224 reads them.
- **`viability.go` (part 1)**: the conversion of a `Config` to the primitives of
  `internal/steady` inline at the call (the activities of each region from the recipe table,
  the yield table and `Calendar`; the year as runs from `SeasonAt` and `site`; `H(rho)` as
  the sum of `Group.Households` of the region (VIA 11: `scale` is not read, so not
  `groupShape`'s weights); the upkeep rows from the role table's `upkeepIn`),
  and `virtualStartOffsets(cfg, calendar, yieldTable, region, recipe)` for 8.5.2 (N-280).
- **`viability_check.go`**: `checkViability(c *Config, lk lookup) viabilityOutcome` (D125)
  running V1 and V4 always, then V2, then V3 **unless V1 found `unproducible_in_realm` or V2 found `cycle_not_productive`** (O12 option (a), ruled by the coordinator, reversible, with the V1/V2/V3 sequencing of the Stage 0 and VIA 4.2.1 rules unchanged; JC9, round 3, B1: a chain with an unproducible good or a dead cycle is 0 by construction and
would repeat the cause as `labour_short`; REF 5 lists no `labour_short` for d1 or d2. A `role_cannot_work` alone does not gate V3: d3 runs V3, `lambda*` 2.512, no `labour_short`) (pooled free trade,
the best yield: one pooled programme with `WithFreeTrade`'s form), then per region in autarky and the two nominal warnings (basis in Phase 1, deliverable 2c), each part under the size gate (D126); it returns `[]*ViabilityError` for the
  five error rules (and `resource_short` is a module's) and the warnings
  `region_not_self_sufficient` (with the commodities it imports), `subsistence_exceeds_expected_work`,
  `target_above_surplus`, `storage_gap_input`, `viability_check_skipped` (naming the part),
  and `cycle_cost_gain` (`raw_without_source` is Stage 0's and is not re-emitted, Phase 4). The inputs come only from the configuration; the
  work is deterministic. It takes a **canonical** configuration (D125: `Validate` and
  `Warnings` pass `c.canonical()` and a lookup built from it; the tables index typed IDs into the sorted
  slices and panic on any other order) that phases 1 to 5 found clean, builds
  `newRecipeTable`, `newRoleTable` and `newYieldTable` once per call and passes them down.
- **`validate.go`**: phase 6 after `overflow()` in `Validate`, only when the report is empty
  (CFG 7.1.6); the `ViabilityError`s join the one sorted list (D124); the rule of the empty set
  of virtual offsets of a job (`ErrWarmStartJob`, FR-VIA-033) at the entry of phase 6, on
  the canonical copy, after the tables are built (it calls `virtualStartOffsets`; the
  Stage 0 `warmRules` keep every other warm-start rule in phase 4); the report gains the
  common element type of D124; the gated append of the viability warnings in `warnings` (D125). The
  Stage 0 warning `cost_floor_at_full_cost` and the others are unchanged.
- **`config/integration_test.go`**: in `TestPacks_DefectOverlays`, the Stage 0 expectation "`Validate` is nil"
  for d1 to d5 becomes `errors.Is(err, ErrNotViable)` (the minimal edit; d6 stays nil with its warning), so the
  suite is green in the PR that adds phase 6 of `Validate`. Plan 1-12 Phase 6 then adds the rule names (N-299).
- **`testdata/chains/defects/d2_upkeep_unmade.toml`** (O12, ruled by the coordinator, reversible): the overlay
  gains `[commodity.fuel]` with `raw = false`, so that d2 triggers its documented rule. Reason: REF 5 expects
  `unproducible_in_realm` for d2 (fuel, prepared, refined, staple, tool_a, tool_b, plus `role_cannot_work`
  for `forester`), but `base.toml` marks `fuel` `raw` and a `Raw` good is sourced (VIA 4.2.1, FR-VIA-059), so
  with the unchanged overlay V1 passes and the pooled programme is 0 (`labour_short`); the rule stays (VIA
  4.2.1 and Stage 0 unchanged) and the fixture moves. The structural pass of CFG 7.5 ignores yield, so
  `fuel` (`cut_fuel` is a source recipe) is still producible there and the file loads; V3 is then skipped
  because V1 found `unproducible_in_realm`, so d2 gives six `unproducible_in_realm` errors and one
  `role_cannot_work` and no `labour_short`. A first test checks the overlay mechanism accepts an explicit
  `raw = false` over a base `raw = true` (if the loader's merge ignores a false, the PR reports it and uses
  the table form that works, e.g. replacing the commodity table). Plan 1-12 asserts the same set. A `Raw`
  good in a mixed basket (one unsourced `Raw` good, the others produced) is the documented consequence of
  4.2.1: pooled `lambda*` 0 and `labour_short`, with `raw_without_source` naming the good.
- **The registry test** is run and must not move (no `Config` field); `TestHashConfig_ReferencePinned`
  must not move.

**Tests:** unit and internal, on configuration literals (A9), each rule with a case that
breaks only that rule (testing.md 3.9):

- Nominal warnings (ruled, Phase 1 deliverable 2c): `target_above_surplus` on a hand region whose mean `TargetRatio` is above its nominal autarky `SurplusBP` and not on one below; `subsistence_exceeds_expected_work` on a region whose labour share is above the mean expected share, and **not raised when the labour share is absent** (a tight capacity row, FR-VIA-012); a region with an empty basket raises neither; the numbers of `three` (south nominal 1.411, `target_above_surplus`) and `five` (nominal 1.12 to 1.31 under the mean target 1.5) are asserted in the external test below, and `famine`'s expectation is Stage 2's.
- Each of the five error rules returns `ErrNotViable` and a `*ViabilityError` with the expected
  `Rule`, `Path`, `Value` and `Bound` via `errors.Is` and `errors.As` (FR-VIA-011, FR-CFG-041);
  a message shows the cut and quoted path; two rules in one configuration both appear, sorted;
  more than 100 errors still end in one `ErrTooManyErrors` (the common cap).
- The warnings never fail `Validate` (FR-VIA-009, FR-CFG-042); `viability_check_skipped` appears
  when `MaxViabilityCells` is lowered by an internal test constant (the gate is a function of
  the constants, so the test passes small numbers to the function that takes them) and V1 and V4
  still run (FR-VIA-010, FR-CFG-043); `cycle_cost_gain` for a productive cycle with `CostFloor`
  above `10 000 * q_out / q_in` (FR-CFG-044); `d6`'s `cost_floor_at_full_cost` is still present
  and the chain still validates (the contract plan 1-12 reads).
- The empty-set rule of `ErrWarmStartJob` for a spring-only crop at tick 0 (the `grow_staple`
  case of `one`: V is empty, FR-VIA-033) and a valid one for a year-round recipe; the function
  `virtualStartOffsets` returns the offsets of 8.5.2 for a hand calendar (FR-VIA-053: it calls
  `Calendar.SeasonAt` and no other rule).
- `Warnings()` on a configuration that does not validate returns the structural warnings and no
  viability warning and does not panic (D125); on a valid one it returns all.
- One named case for each warning FR-VIA-009 and FR-CFG-042 list that has none above
  (round 2, N4; testing.md 4.5), each breaking only its rule and never failing `Validate`:
  `region_not_self_sufficient` (the `two` + `a5` shape on a literal), `subsistence_exceeds_expected_work`,
  `target_above_surplus`, `storage_gap_input` (an all-year input with a gap against its shelf life)
  (`raw_without_source` is Stage 0's, already tested); `wants_exceed_labour` waits for DEM.
- Fuzz: the existing `FuzzValidate` gains the viability phase (never panics, an error is a
  `*ValidationError`, a `*ViabilityError` or the cap), run for a minute by hand on one
  architecture; its committed corpus is kept. `FuzzCheck` over small random configurations built
  from bytes; `FuzzVirtualStartOffsets` (never panics, offsets inside `[0, d)`, deterministic, empty where `V` is empty; round 3, N8: this row owns the function, so NFR-VIA-004's offsets half has an owner here; the stagger and apportionment fuzz is the world row's, plan 1-11).
- External test (A9): the packs `one`, `two`, `three`, `second` and the cold `five` validate
  (REF 7.3 table: V1, V2, V4 pass; no pooled `labour_short`). **Not in this list, and why**
  (round 2, B3, checked by loading them today): `famine` does not load (`pool_release` and
  `relief_below` are unknown region keys, and `overlay/estates.toml` needs `modules.estates`, which plan
  1-12 defers to Stage 2) and neither does `one-demand`; `five` + `warm-five.toml` is rejected by
  `ErrWarmStartPrice` (`semi_a` and `semi_d` at 50 under the band floors 51 and 52 of
  `RefBandLow`; the validator's band rule stays and `warm-five.toml` is regenerated in Phase 7, O11) and `famine` + `warm-famine.toml` would be (maker shares 2 272 + 8 522 = 10 794
  above 10 000), so the warm documents of those two are compared without `Validate` (Phase 7). The
  `famine` of this row is `one` loaded through `config/` and mutated in the test (staple yield
  `spring = 3000`, the six group sizes of `famine.toml`: 68, 17, 45, 417, 33, 20 households; no
  estates); `famine.toml` differs from `one.toml` in exactly those lines and the two estates keys,
  and the estates keys change no programme, and the six defect overlays load
  and are rejected with the rule REF names (a first check; plan 1-12 owns the file tests); a
  recipe on two roles' lists does not occur in the packs (N-282); every file under
  `overlay/` except those that name an unbuilt module still validates, and d1 to d5 are the
  only layered files that do not (N12: the skip loops above would hide a change); the same
  packs with their entity lists reversed validate and give the same viability outcome, which
  proves phase 6 canonicalises (round 1, B4: a unit case with reversed commodity, recipe and
  region lists goes through phase 6 and must not panic); a second case with an unknown
  region in a yield profile is stopped by phase 3 before any table is built. FR-CFG-043's
  second half (programmes run below the gate and are skipped above it) has its own case.

**Observability and rollback:** No new event, metric or log key; a rejected configuration is
returned as an error, never logged (architecture.md 8.3), and the warnings use the existing
"configuration warning" line with `rule` and `path`, one per warning, bounded by the
configuration (CFG 7.2); the solver logs nothing. Rollback is reverting the PR: `Validate` then
returns to the Stage 0 behaviour, and nothing else reads the new symbols yet.

**Acceptance:** `Validate` rejects d1 to d5 with their rules and accepts d6 with its warning;
the packs validate; the registry and the pinned hash do not move; coverage at least 90% for the
root and for `internal/steady`; Lizard clean.

## Phase 6: `SolveSteadyState`, the table and the report

**Why here:** it needs the programmes of Phase 3, the checks of Phase 4 and the conversion of
Phase 5, and it is the last piece before the agreement run.

- **`viability.go` (part 2)**: `SolveOption` and the five options of D127, `SteadyState`,
  `ProgrammeResult`, `Report` and its row types (D128); `SolveSteadyState(ctx, cfg, opts...)`:
  `cfg.Validate()` first, **run on a copy whose `WarmStart` is nil** (a shipped but stale table such as
  `warm-five` must not make the solve that replaces it fail with `ErrWarmStartPrice` or
  `ErrWarmStartJob`; FR-REF-007 and `experiment/` `Prepare` need that, and a caller who wants the
  loaded table checked calls `Validate` itself; tested with `five` + `warm-five.toml`, which solves,
  and with `five` cold, which gives the same table) (a failure returns that error, which wraps `ErrNotViable` for a
  viability rule, FR-CFG-033; the solver is the same code at its non-gated size), `ErrNoUpkeep`
  (D130), the programmes of D129 and the nominal fallback of D131, the wrapped `ctx.Err()`, no
  partial result on any error.
- **`viability_table.go`**: the price scale of 6.3 (autarky `B_ref(rho) / sum u(c) p(c)`;
  pooled one scale such that the household-weighted baskets equal the household-weighted
  reference baskets), the seed prices rounded half to even in `big` and raised to the resolution
  floor, **lifted into the reference band** (O11, ruled by the coordinator, reversible; the bounds are the ones `warmRules` checks, so a floored or free good such as `semi_a` and `semi_d` of `five` takes the band's lower bound, 51 and 52, and the table passes `Validate` with its band rule unchanged) and capped at `MaxPrice`, with the changes listed (`Floored`, the unclamped price in `OutsideReferenceBand`, FR-VIA-023, 024); the wage
  (the labour dual scaled the same way, rounded half to even, no entry for a region at 0, listed
  in `UnpricedWages`, FR-VIA-050); the commodities with a non-unique price take the
  reference price and are listed (`UnpricedCommodities`, FR-VIA-026); a free good (dual 0) is listed
  the same way but its price is the band lift of O11 (max of the resolution floor and the band's lower
  bound, never the reference price: the one change of VIA 8.2.4 and FR-VIA-026 that 2e writes), except a commodity that is in no
  input, tool, non-discard output or upkeep, which has no row at all (D134 (7)); the money of 8.3.1 capped
  at `MaxMoney` and the stock of 8.4.1 and the jobs of 8.5.1 by the composition rules of D134
  as Phase 1 writes them (the "main recipe" of the money is the one with the longest duration, ties by the largest inputs cost, then the lower recipe ID, JC3; the stock per commodity is
  the larger of cover and summed one-execution inputs plus tool items plus the reserve of
  D123, cut to the shelf life with the cut listed, FR-VIA-039; a recipe of duration 1 writes
  no job; `InProgressBP` rounds down), with D120 and D121, sorted; no pool row (the table's
  `Pools` is nil); absent lists are nil, and the table passes through `Config.canonical()` so
  `DeepEqual` against a loaded document is meaningful (D133). The household count `H(rho)` is
  `Group.Households` summed over the groups of the region, not `groupShape`'s weights (VIA 3.2
  and 11 do not read `scale`), and the per-household upkeep of a role is `u(c) = D(c, rho) /
  H(rho)` as an exact fraction; `B_ref(rho)` is `Region.ReferenceBasket` (FR-VIA-024). Every conversion to `int64` is `ok`-checked against `int64` and the field's bound and
  an overflow is `ErrOverflow` with the field named (architecture.md 3.5).
- **The report**: the drift of each seed from the effective reference price; the residual of 6.5
  (the value added of each active activity at the integer prices against the wage times the
  duration, in basis points, using the integer prices and `internal/production`'s `UnitCost` and
  `Basis` where it computes the same rounding up, so the check and the run agree: FR-VIA-025, 027,
  054); `SeedBelowCostFloor` and `SeedAboveCostCeiling` against the full unit cost at the solver's
  wage; `OutsideReferenceBand`; `Census` (D121, D132) and `CapacityTight`; `Advantage` and
  `ratio_outside_autarky` (FR-VIA-045); `RegionSurplusBP`; `WageBelowTarget` via
  `welfare.Wage` (FR-WEL-031, 038: a report with a number per role, never an error).
- **Lizard**: the table build is split into `prices`, `wages`, `money`, `stock`, `jobs` and
  `report` helpers, each at complexity 10 or less, and none takes more than 8 parameters (pass a
  struct of the programme's results).

**Tests:** unit (internal test on literals) and one external test on the packs:

- The scale and rounding on the prototype's example: chain A rescaled, `B_ref = 1 000`, floor 50,
  half to even: seed 3 353, grain 373, flour 342, dough 277, loaf 376, fuel 248, iron 166, hoe 3 146,
  millstone and oven 3 312, straw and bran at the floor 50 (VIA 15; recompute and write the exact
  digits); this bullet tests the rounding and floor stage as a function that runs before the band lift, so 50 is the floored value there; the lifted value (the band's lower bound, O11) is asserted in the next bullet; the exact wage 2 980.47 rounds to 2 980; a value exactly half rounds to the even one.
- A region at the floor, a free good, dual 0 (priced at the band's lower bound, listed in `UnpricedCommodities`), a non-unique price (reference price and listing), a floored or free price below the band (lifted to the band bound, listed in `Floored` and `OutsideReferenceBand`, and the table passes `Validate`), a price above `MaxPrice` and
  an overflow of the money product each have a case that breaks only that rule.
- The money of 8.3.1 on a hand group with two recipes where the longest duration and the largest inputs cost disagree (the `second` fabber, `fab_board` against `make_fabricator`, 19 423 not 52 189; a tie in duration falls to the larger inputs cost, then the lower recipe ID), and the stock of 8.4.1 (the arithmetic in the comment),
  including the cut to the shelf life (`prepared` 4 in `one`) and the lean reserve of D123.
- The jobs: D120's scaling, D121's shared recipe and the over-10 000 scale-down, a recipe whose
  offsets are empty writes no job and holds the reserve (the `grower` of `one`); and the as-written
  seasonal formula of D134 (9): a seasonal recipe (|V| 28, N_open 183, as `two` south, giving 3 110.5
  rounded down to 3 110 in the formula's own units) beside an all-year recipe (x1 * d), so the six
  document rows are explained by a named case and not only by Phase 7.
- The nominal fallback (`famine` built as above from `one`: full-labour wage 0 with a tight
  capacity row, nominal wage positive; FR-VIA-058) and a region whose nominal wage is also 0.
- The residual and the seed over the full unit cost within 1% of 1 for the rescaled chains
  (FR-VIA-054; VIA 15: 0.991 to 1.000), and the violating cases (round 4, N4): a seed under the full-cost floor is listed in `SeedBelowCostFloor`, one over the ceiling in `SeedAboveCostCeiling` (FR-VIA-027), and a drift over the stated value is reported (FR-VIA-025), each on a hand chain with the arithmetic in the comment.
- The labour share is absent when a capacity row is tight (FR-VIA-012, round 4, N4): the `famine` programme at full labour (`grow_staple`'s capacity row is tight) reports `HasLabourShare` false, and a slack case reports the share.
- `Census` imbalance and the ambiguous flag (FR-VIA-028): a hand group with an imbalance gives the flagged census, a balanced one does not (the formula is D132; Phase 7 compares `one` against REF).
- Options: each bound, a nil option, the last wins, free trade turns two autarky programmes into
  one pooled programme with one scale (FR-VIA-043, 044); a `Guard:` test that the exported `With*` functions of `viability.go` are exactly the five of D127 (no `WithLanes`, no `Lane` type: a lane cannot be expressed, so no rejection code exists); `WithMaxPivots(1)` returns `ErrSolveLimit`
  with the counts; a cancelled context returns the wrapped error and no table; `NoUpkeep` returns
  `ErrNoUpkeep` (FR-VIA-018); a non-viable configuration returns `ErrNotViable`; a configuration whose only basket goods are unsourced `Raw` markers (`minimalConfig()`) returns no table row, the region in `UnpricedWages`, the commodities in `UnpricedCommodities`, and `ErrNoUpkeep` when no other programme remains (D130, JC5); `FuzzSolveSteadyState` covers it; a unit case has a long inactive recipe in the group (the `salvage` shape) that changes neither the main recipe nor `d_max`.
- Purity: the same call twice gives `reflect.DeepEqual` results; the function takes its arguments
  by value and does not change `cfg` (compared before and after); a `Guard:` text search that
  nothing in the solver's files calls `time.`, `rand` or `slog` (the determinism guard covers it).
- Fuzz: `FuzzSolveSteadyState` over small configurations from bytes (never panics; every result
  is an error or a table that passes `Validate` and the table checks of 8.7 (the band lift of O11 makes this unconditional); NFR-VIA-004) and
  no stagger fuzz (the stagger is the world row's; `FuzzVirtualStartOffsets` is Phase 5's).

**Observability and rollback:** none: the solver is pure and logs nothing; its diagnostics are
the `Report` (VIA 17.2). The one `Warn` path is the configuration warnings of CFG 7.2, already
counted in Phase 5. Rollback is reverting the PR; no other row calls the function yet.

**Acceptance:** `SolveSteadyState` returns for `one` a table that passes `Validate` with the
warm start present (the validator's own rules accept it, FR-VIA-030, 051, 041); the VIA 15
numbers are reproduced; coverage at least 90%; Lizard clean.

## Phase 7: agreement, cost and the exit gate

**Why last:** it checks the row as a whole against the references and hands over to the
world and golden rows.

- **Agreement with REF** (A1, A3, A5, A6, A9; S21): for each pack of REF's table (`one`, `two`,
  `three`, `five`, `famine`, `second`) the surplus factors (2.512; 2.229 and 2.216; 2.512, 2.060,
  2.666; 1.915, 1.704, 1.642, 1.805, 1.904; 1.315; **1.966 for `second`**, all REF 7.3's, which
  the restored start rule of A3 reproduces), the pooled factors and pivots (`one` 10, `famine` 9 and
  `second` 10; the two-region chain B 1.911 with 18 and the three-region chain 3.124 with 24 are
  Phase 3's; `two` 19, `three` 30 and `five` 202 are REF's numbers and are asserted; `proto3.py`
  holds no `two`, `three` or `five`, so they were measured with an exact LP loaded from the packs (round
  3, N5); a difference is recorded, not a failure), the labour shares and the nominal factors, the supply
  gaps (304, 123, 184), the census, and the loaded `warm-*.toml` against the table the solver produces
  for the layered configuration without it (D133). Differences are handled as D133 says (the spec text
  decides, O2). **The differences expected, all from rulings (nothing else is expected; any other
  difference, a money one above all, is a defect to explain):** (1) the six seasonal job rows of
  `warm-two` (south 3 341), `warm-three` (south 3 464, far_south 1 736) and `warm-five` (centre 3 766,
  east 4 124, south 2 506), which VIA 8.5.1 as written changes (D134 (9)); (2) `warm-five.toml`, regenerated
  as a whole (no writer exists in this row, D133 puts it in plan 1-12: a throwaway generator, not
  committed, prints the canonical document from the solver's table and the file is replaced by its
  output; the three job rows of `warm-two` and `warm-three` are edited by hand): `semi_a` and `semi_d` lifted into the band (O11: 51 and 52), the six job rows, its prices and
  wage (a constant 1.0039 for the centre and 1.0024 for the east times the exact dual scale of VIA 6.3)
  and its centre grower staple reserve (1 087 against 1 092), its producer being unarchived (N-274;
  `one`, `two`, `three` match to the unit apart from item (1)); (3) possibly the one `grow_algae` row of
  `warm-second.toml` (5 163 against the solver's, D134 (8); if it is regenerated, the pinned hash of
  `second/second.toml` + `second/warm-second.toml` in `config/integration_test.go` (line 42) is updated
  in the same change with the testing.md 7.2 explanation, while the warm-one pin and
  `TestHashConfig_ReferencePinned` do not move); (4) the two maker rows of `warm-famine.toml` (recorded, the file not edited in Stage 1)
  (shares 2 272 + 8 522 = 10 794 are above 10 000, which `Validate` rejects, so the solver scales them
  down, D121), compared on the `famine` built from `one`. The prototype of `models/via/` is run in Docker
  for lambda, prices and the three-region ratios (A7). `warm-second.toml` is otherwise unchanged.
- **The documents that do not validate.** `warm-five.toml` (`semi_a` and `semi_d` at 50 against the band
  floors 51 and 52; FR-VIA-026 also lists these free goods unpriced where the document says
  `unpriced = []`) and `warm-famine.toml` (maker shares 10 794) cannot equal a table that passes
  `Validate`. The comparison is on the loaded document without `Validate` (`config.LoadFS` then
  `canonical()`), the differences are the ones listed above, and `warm-five.toml` is regenerated here with
  the testing.md 7.2 explanation (N-274); `warm-famine.toml` is left UNCHANGED in Stage 1 (ruled:
  its regeneration is Stage 2's; no Stage 1 test loads it through `Validate`, since `famine` needs the
  estates keys, so an edit here would have no consumer to prove it). The 10 794 of its two maker rows
  is recorded as an expected difference only, and plan 1-12's `famine` byte test and the regeneration
  of those two rows are Stage 2's (a note to that effect in the PR). **Validation check of the regeneration:** after `warm-five.toml` is regenerated, `Validate(five + warm-five.toml)` returns nil (the independent proof that the band lift, shares and products are right; it flips the rejection recorded in Phase 5), and a hash pin of `five` with the warm table, if plan 1-12's golden has one, is recorded after it. The agreement of `famine` with `warm-famine.toml` is on the `famine` built from `one`
  (Phase 6); the loaded estates pack and N-085's STY run are Stage 2.
- **A5** (`two` + `overlay/a5`): autarky `south` has `lambda*` 0, pooled passes, and the warning
  `region_not_self_sufficient` names the staple (A5 of REF). **A3** (`three`): each region's autarky
  `lambda*` differs, the pooled activities are active in a strict subset, the pooled ratios lie in the
  autarky range or are listed (`ratio_outside_autarky`). **A6** is the unit test of Phase 3.
- **FR-REF-009 to 013** (round 3, N8; in part, see the Testing summary: the pack-shape and estates parts belong to the Stage 0 pack tests and Stage 2; the solver outputs this phase already computes): 009 `one` has `lambda*` at least 2; 010 `two`: the autarky staple/fuel price ratios of the two regions differ by a factor of at least 2; 011 `three`: the pooled programme runs an activity in a strict subset of the regions; 012 `five`: the pooled `lambda*` is above every autarky one; 013 `famine` (built from `one`): `lambda*` below 2 with a positive wage. Read the exact assertions from REF before writing them.
- **A9 determinism**: two solves in one process give equal tables and hashes; the two-architecture
  run is the golden row's (CI-less, O3); the test is written to pass there.
- **A10 cost**: `BenchmarkSolveFive` with `b.ReportAllocs()` (NFR-VIA-003), the pivots (202 expected)
  and the time recorded in N-061 and N-015; and the cost of `Validate` on `five` with the
  programmes on, and the double cost of `Validate` plus `Warnings` (N-276); if `Validate` on `five`
  exceeds one second (NFR-VIA-002), the gate constants stay and the note says so (a placeholder is
  not tuned to pass).
- **Exit gate**, on the whole module: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`,
  `go test -short -race ./...`, the full run with `-covermode=atomic -coverpkg=./...`,
  `go mod tidy` (no diff, no new dependency), the per-scope coverage read and recorded;
  `lizard -l go -C 10 -a 8 -L 100 -w` on the new files; each fuzz target run for a minute by hand on
  one architecture, corpora committed; the pinned hash of the all-off configuration did not move
  (`TestHashConfig_ReferencePinned`); `testdata/golden/off/` does not exist yet (plan 1-12 records it).
- **Traceability** (testing.md 4.6) by text search: every ID in a `Requirement:` line exists in VIA or
  CFG, and every requirement the table below lists as covered in full is cited.
- The roadmap row set to **done** with its commit (at the end of the row); the notes of Phase 1
  updated with what the code settled; N-061, N-015 and N-276 get the measurements.
- Not gating, if time allows: a mutation pass over `internal/steady` (testing.md 9.4, a SHOULD) with
  `gomu` inside Docker only, `--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS` and `--workers`
  equal to the CPU cap, never on the host.

**Tests:** the gate itself, plus the external tests above (integration level: the packs through
`config/`).

**Observability and rollback:** none; rollback is reverting the PRs in reverse order, and the world
and golden rows, which start after this row, are not yet built.

**Acceptance:** every command is clean; the REF numbers are reproduced or each difference is one of the listed ones and has an
explained resolution (D133); the traceability check finds no ID without a spec row and no
"covered in full" requirement without a test; the roadmap row reads done.

## Cross-plan contract

### Consumes

| Symbol, type, sentinel, spec edit or field | Provided by | Status |
|---|---|---|
| `Config`, `WarmStart`, `SeedPrice`, `SeedWage`, `WarmGroup`, `JobSpec`, `WarmPool`, `World.CostFloor`, `CostCeiling`, `RefBandLow`, `RefBandHigh`, `BeliefStep`, `Role.ExpectedWorkTicks`, `TargetRatio`, `Region.ReferenceBasket` | Stage 0, `config.go` | Built. **N-284:** the section type `World` is renamed `WorldParams` (and `DefaultWorld` to `DefaultWorldParams`) by row 11, Phase 2; this row is written against whichever name is in the tree when it starts, reads the section only through `Config.World` and the helpers `groupShape` and `resolutionFloor`, and takes the rename as a mechanical edit of its own tests |
| `validate.go`'s `lookup`, `groupShape`, `resolutionFloor`, `effectiveUpkeep`, `warmRules`, `warmProducts`, `ErrWarmStartPrice`, `ErrWarmStartJob`, `ErrOverflow`, `ErrOutOfBounds`, the report type | Stage 0 | Built |
| `Calendar.TicksPerYear()`, `Calendar.SeasonAt(tick)` | [plan-phase-1-2.md](plan-phase-1-2.md) (row 2) | Built |
| `newRecipeTable`, `recipeTable.makers`, `recipeRow` (`inputs`, `outputs`, `tools`, `source`), `newRoleTable`, `roleRow.recipes`, `(*roleRow).upkeepIn(RegionID)` | [plan-phase-1-2.md](plan-phase-1-2.md) (row 2) | Built |
| `newYieldTable`, `yieldTable.site(region, recipe, season, sited)`; `internal/production.UnitCost`, `Basis`, `WearCost`, `Term`, `Wear` | [plan-phase-1-5.md](plan-phase-1-5.md) (row 5) | Built |
| `internal/welfare.Wage` (the calibration formula for `WageBelowTarget`) and `seedWages` | [plan-phase-1-6.md](plan-phase-1-6.md) (row 6) | Built |
| `internal/checked`: `Add`, `Mul`, `MulDiv`, `MulDivCeil`, `MulDivHalfEven` | Stage 0 | Built |
| `testdata/chains/` packs, `defects/`, `warm-*.toml` and REF 7.3's expected numbers | Stage 0 and REF | Built |
| The roadmap, documentation index, notes and `AGENTS.md` shared edits | This row, Phase 1 | Owed here |
| Row 11 (world) calls `virtualStartOffsets` and `warmJob` to create the jobs and applies the table in the order of `spec-world.md` 5.6.1 | Row 11 (world), [plan-phase-1-11.md](plan-phase-1-11.md) (piece 3e) | **ASSUMED**: this row provides the offsets; the world row owns the stagger and the application |
| Row 12 (golden) reads `SteadyState.Programmes[i]` for the header (Provides) and the d1 to d6 contract | Row 12 (plan-phase-1-12.md) | **ASSUMED** by row 12; named in Provides below |
| Rows 7 to 9 (agent, estates, spawner): none | | Nothing consumed from them; the agent row reads the role table only |

### Provides

| What | For | Contract |
|---|---|---|
| `Config.Validate` gains phase 6; `ErrNotViable`, `*ViabilityError` (`Rule`, `Path`, `Value`, `Bound`, `Unwrap` to `ErrNotViable`) | `config/` tests, `NewWorld`, plan 1-12's d1 to d6 file tests | `errors.Is(err, ErrNotViable)` and `errors.As(err, &ve)` on the joined error of `Validate`; `ve.Rule` is one of `unproducible_in_realm`, `role_cannot_work`, `cycle_not_productive`, `labour_short`, `storage_gap` (plain strings; the tests spell them); a configuration with several defects returns several, sorted by path then rule; **d6 loads with no error and `Warnings()` holds `cost_floor_at_full_cost`** (unchanged from Stage 0). The documents d1 to d5 give (O12, ruled): d1 `cycle_not_productive` and `unproducible_in_realm` (REF 5: "rejected by `cycle_not_productive` and its consequences"; V3 skipped because V1 found `unproducible_in_realm`), **d2 `unproducible_in_realm` (fuel, prepared, refined, staple, tool_a, tool_b: six errors) and `role_cannot_work` (forester), no `labour_short`, as REF 5 says, because Phase 5 edits the overlay so that `fuel` is not marked `raw`** (with `raw` a `Raw` good is sourced, VIA 4.2.1; the rule stays and the fixture moves), d3 `role_cannot_work` only, d4 `labour_short` only, d5 `storage_gap` only. Plan 1-12's d2 assertion is this set |
| `SolveSteadyState(ctx, cfg, opts...) (SteadyState, error)` and the five options of D127 | consumers, `viability.go`'s own tests, plan 1-12 (the header), `experiment/` (`Prepare`) | Pure; the sentinels `ErrNotViable`, `ErrNoUpkeep`, `ErrSolveLimit`, `ErrSolveInconsistent` and the wrapped `ctx.Err()`; no table with any error; the solve validates a copy with `WarmStart` nil, so the result never depends on the loaded `Config.WarmStart`; the table is in canonical order and passes `Validate` (a seed raised to the resolution floor is also lifted into the reference band, O11 ruled) |
| **The pure read for the golden header (REF 8.5 item 1)**: `SteadyState.Programmes[i]` with `Regions`, `SurplusBP`, `LabourShareBP`, `HasLabourShare` and `Pivots`, and `SteadyState.Report.UnpricedCommodities` (`[]RegionCommodity{Region, Commodity string}`) and `UnpricedWages` (`[]string`) | Plan 1-12's golden renderer (`surplus_bp`, labour share, pivots, unpriced) | All `int64` or strings, no map and no float, lists in ascending region then commodity order; for a one-region pack `Programmes` has one entry (the autarky programme of the region) and the four header values are read from it and from the two lists; an absent labour share is `HasLabourShare` false and is omitted from the rendering; the values of `one` are 25 119 for `SurplusBP` (exact lambda 29568/11771 = 2.51194, floored; REF: 2.512), 3 982 for `LabourShareBP` (`ceil(10^8 / 25 119)` of the integer `SurplusBP`, as FR-VIA-012 and VIA 3.2 define it; round 3, N3: ceil of the exact lambda would give 3 981 and is not the rule) and 10 pivots (REF 7.3), to be confirmed by Phase 7. A read, not a function call into the world: the renderer calls `SolveSteadyState` itself on the layered configuration, so no world is needed |
| `virtualStartOffsets(...)` (unexported) | `validate.go`, row 11 (stagger and jobs) | The ascending offsets `o` in `0 .. d-1` with a positive factor at tick `-o`, by `Calendar.SeasonAt` (FR-VIA-053); empty means `ErrWarmStartJob` |
| `checkViability(c *Config, lk lookup) viabilityOutcome` (unexported) | `NewWorld` (row 11), if it wants the work once (N-276) | Returns the errors and the warnings of V1 to V4 and the cycle gain with no side effect |
| `WithFreeTrade` (the only lane-related option); **not provided in Stage 1: `Lane`, `WithLanes`, `MaxLanes`** (ruled, D127) | tests; `transport.Lanes` (Stage 2) adds the three | N-279 |
| `MaxViabilityPivots`, `MaxViabilityCells` in `bounds.go` | `validate_counts_test.go`, CLB's registry | FR-CFG-046 |
| `Report.WageBelowTarget`, `Census`, `RegionSurplusBP` | CLB, STY (S21, S22), SUB later | Reports, never errors |
| The glossary terms of Phase 1 deliverable 9 in `doc.go` | all later rows | Added before use |

## Security review

The module is a library with no network, file access, credentials or persistence. The
trust boundary of this row is the configuration (a loaded pack, a hostile or merely
huge one) and, for `SolveSteadyState`, a `context` and the options.

- **Who can reach it.** Any caller of `Validate` (the solver runs inside it) and of
  `SolveSteadyState`. The input is attacker-shaped in a service that accepts packs. The
  worst case is work, not corruption.
- **Resource exhaustion.** The whole of the tableau is the risk: big integers grow
  with the pivots and the size is rows times columns. The gates are the defence: counts
  first (CFG 7.1.6, `ErrTooMany`), then `MaxViabilityCells` before any tableau is built,
  `MaxViabilityPivots` as the pivot cap, and the context checked between pivots. V1 and V4
  are never skipped and are proportional to commodities times recipes, plus one scan per
  profiled (region, recipe) pair that is O(seasons) (A3). The memory of a tableau of 4 000 000
  cells of big integers can be hundreds of megabytes if the entries grow; Phase 7 measures
  it, and the gate stays a placeholder until the number is known (N-061). The size of a
  single entry is bounded by the determinant bound of the integer-preserving form (A2); a
  test on `five` records the largest entry in bits. **Residual risk, stated** (round 1, N1):
  the product `MaxViabilityPivots` times `MaxViabilityCells` is 4 x 10^11 big-integer cell
  updates, which is not a second, and `Validate` takes no `context.Context`; NFR-VIA-002 and
  NFR-CFG-003 are SHOULD and provisional (N-015), so the gate bounds the tableau and the pivots
  separately and a hostile pack that sits just under both can still cost minutes. Phase 7 adds a
  work budget (cells times pivots) to the gate if the measurement of N-061 says the two caps are
  not enough, or records the risk in the note.
- **Integer overflow.** Inside the solver big integers cannot overflow (architecture.md
  3.5). At the boundary every conversion to `int64` is checked, and the products the root
  forms (a weight times a stock, a wage times a duration) go through `internal/checked`.
  A value that does not fit is `ErrOverflow` with the field, never a wrap.
- **Division by zero.** A zero denominator (no upkeep, `J` of 0, a zero labour dual,
  `SurplusBP` of 0 in the labour share) is an absent result or a branch, never a panic
  (FR-VIA-012, 018); each has a test. A programmer-invariant panic exists only for an
  unsorted or non-zero-W input from the root, which cannot be built from a validated
  configuration.
- **Error text.** A `ViabilityError` carries a path of IDs from the configuration; its
  `Error()` cuts and quotes each segment as `ValidationError` does, so a control character
  or a long ID cannot flood a log (CFG 6.3.2).
- **Determinism as a guarantee.** The table is configuration that is hashed; a wrong or
  platform-dependent table would silently change every run. The solver is pure, uses no map
  iteration for an ordered result and no float, and its output is compared with the
  documents on every run.
- **Manipulation.** A consumer cannot pass a table the validator does not check: the table is
  data validated by `Validate` (the rules of 8.7), not trusted because the solver produced it.
- **Not applicable:** authentication, authorisation, injection, secrets, cross-tenant
  exposure, supply chain (no dependency is added; `math/big` is the standard library).

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer. The new
  exported identifiers are `SolveSteadyState`, `SolveOption` and its five options,
  `SteadyState`, `ProgrammeResult`, `Report` and its row types, `RegionCommodity`,
  `ViabilityError`, `SolveError` and the four sentinels; each is a promise for ever
  (architecture.md 2.8, 6.6), so the shape of D128 is the thing to review hardest. The rest
  is unexported or under `internal/`.
- **Behaviour change of `Validate`.** From Phase 5 a configuration that Stage 0 accepted can
  be rejected with `ErrNotViable`. The packs are the evidence they pass; the changed rows are
  listed in the Phase 5 PR as CFG 7.4.1 and 7.6 say. The pinned hash and the registry do not
  move (D5).
- **Document contracts.** Phase 1 adds text to VIA, CFG, `architecture.md` 2.6 and `AGENTS.md`
  and renumbers nothing (testing.md 4.1). `architecture.md` gets a fresh RULE-TAG.
- **What becomes hard to change, and when.** The shape of `SteadyState` and the reading of D120
  (the flows of the table) enter the golden files at the end of Stage 1 (the header of plan 1-12
  and the warm documents) and the hash never (the table is data). Until the first golden file is
  recorded they may be corrected in a normal PR; after it, a change is a behaviour change with
  the explanation of testing.md 7.2.
- **Append-only stays true.** A module appends its row, its column or its report field in its own
  PR (architecture.md 2.10.5); with the module off it adds nothing (D5).
- **Rollout** is one PR per phase (A8), each leaving the branch passing every command of
  `AGENTS.md`. A phase is rolled back by reverting its PR; later phases depend only on earlier
  ones, and no other row of Stage 1 calls the new symbols until row 11.
- **Dependency failure.** None at run time. The schedule risks are (1) the world row and the
  golden row planned beside this one, which read Provides: the three names are fixed here and
  changed only with a note; (2) a REF document that does not load equal (D133): resolved by the
  owner under O2 without blocking Phases 2 to 6, because they do not read the documents.
- **The Go toolchain.** `go.mod` says `go 1.25`. Nothing here depends on a version-specific
  order: no map iteration and no unstable sort that affects a result; every sort has a total
  order ending in a unique tie-break (architecture.md 7.6).

## Testing (summary, testing.md section 11)

The pyramid for this row is a wide base of unit tests in `internal/steady` and the root's internal
tests, with differential, fuzz and determinism tests over the pure functions, and a few integration
tests on the packs through `config/`. There is no end-to-end test because there is no world.

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-VIA-013, 014, 015, 016, 017 (the solver core); NFR-VIA-004 (the solver fuzz); NFR-VIA-001 | FR-VIA-020, 022 (the core is deterministic; the canonical build is Phase 3) |
| 3 | FR-VIA-001, 002, 003, 012, 019, 022, 053 (the offsets function is Phase 5), 058 (the unit); FR-VIA-045 (the report fields) | FR-VIA-013 (on the programme), FR-VIA-043 (the collapsed single goods row is the realisation of the complete free-trade set; the lane columns come with TRN), FR-VIA-044 (the free-trade pooled form; the option is Phase 6; the per-pair lane columns come with TRN) |
| 4 | FR-VIA-004, 005, 006, 008, 039, 059 | FR-VIA-057 (`raw_without_source` is Stage 0's rule, reused ungated; Phase 4) |
| 5 | FR-VIA-007, 009, 010, 011, 033 (the empty-set rule), 053 (the offsets); FR-CFG-033, 041, 042, 043 (with an explicit case for the second half: programmes run below the gate and are skipped above it), 044; NFR-CFG-003 | FR-VIA-044 (`Validate` uses free trade, in the collapsed form; the lane columns come with TRN), FR-CFG-046 (`MaxViabilityPivots`, `MaxViabilityCells` built; `MaxLanes` waits for `WithLanes`, D127, N-281), FR-VIA-057 (the other warnings; `raw_without_source` is Stage 0's; `wants_exceed_labour` waits for the demand module, completed by the demand row), FR-CFG-037 (built at Stage 0; extended by the empty-set rule) |
| 6 | FR-VIA-018, 020, 023, 024, 025, 026, 027, 028, 030 (the seed rules hold for the solver's table), 050, 051, 054; FR-CFG-047 (the five options exist and are not `Config` fields, not hashed; built here, round 3, N9); FR-VIA-049 (types built at Stage 0) | FR-VIA-029 (Stage 0), FR-VIA-045 (the pooled ratios), FR-VIA-055 and 056 (module off only: the upkeep-only programme; completed by the demand row, which adds the wants term) |
| 7 | REF A1, A3, A5, A6, A9, A10; S21; NFR-VIA-002, 003 | FR-REF-009 to 013 (in part: the solver outputs here; the pack shape (six roles, nine commodities, ten recipes, 5 000 households, 30 commodities, 45 recipes, each region viable alone) is the Stage 0 pack tests' and the estates and positive starting wage part waits for `famine` in Stage 2), FR-VIA-048 (in part: chain A is a unit programme, not a pack, and "run" is the world row's, plan 1-12 X4), FR-VIA-031 to 047 (the world row builds the construction; this row proves the table loads and validates) |

Built in Stage 0 and not re-tested: FR-VIA-029, 030 (the validator half), 041, 051 (the bounds), FR-CFG-037.
Deferred with their code: FR-VIA-021, 031 to 038, 040, 042, 046, 047, 052 (the world row; 052 was plan 1-6's);
the resource, wants, facility and open-layer terms (FR-VIA-055 and 056 with the module on, the resource and
facility rows, `resource_short`, `wants_exceed_labour`).

"Covered in full" means at the lowest level that can verify it (testing.md 2.1). The second half of
testing.md 2.3, the same behaviour through the public API, is the external pack test of Phases 5 and 7 and
plan 1-12's golden header.

Boundaries and error paths: every `ok` false of the boundary, every sentinel (`ErrNotViable`,
`ErrNoUpkeep`, `ErrSolveLimit`, `ErrSolveInconsistent`, the wrapped `ctx.Err()`), every option bound,
every gate, every absent result has a case that breaks only its rule. Golden files: none (plan 1-12).
Fuzz: the solver, the start count, the checks, `FuzzValidate` extended and `FuzzSolveSteadyState`. Each PR
states its coverage and that its tests were run against a broken version (flip a comparison in the ratio test,
corrupt a pivot entry for the certificate: testing.md 9.3).

## Observability (summary, observability-and-logging.md section 9)

None new, because the solver is a pure function that logs nothing and emits no event (VIA 17), and the tick
path is untouched. Walking the nine questions: (1) a new failure path or rejected input: every one is a
returned error (`ErrNotViable`, `ErrNoUpkeep`, `ErrSolveLimit`, `ErrSolveInconsistent`, the wrapped
`ctx.Err()`) or a warning of CFG 7.2, with no log of an error that is also returned; (2) no new state
transition: the table is applied by the world row, whose `WorldCreated.WarmStart` bool and the "warm start
applied" `Info` line (FR-VIA-046) are its; (3) no decision point in a strategy, clearing rule or spawner;
(4) no loop in the tick; the loops of the solver run at validation or on demand, outside the allocation limits
of NFR-OBS-001 (NFR-VIA-001); (5) no event, metric, unit or attribute changes; (6) no new metric attribute;
(7) neither `Error` nor `Info` is used here; (8) no error is logged and returned; (9) the tests of section 8
that apply are none (the warnings are asserted by their rule and path at the root, and the counterpart rule of
OBS 7.3 needs a world). The new warning rules are rows of CFG 7.4.2, the `Warn` line "configuration warning"
and its keys `rule` and `path` exist (OBS 3.3.2).

## Open items

- **O1.** D132: the census formula is not fixed by the prose of VIA 5.2 and REF 4.3 (whether the work share
  divides the start-capacity term). Phase 1 pins it against the 600 households of `one.toml`; if two readings fit
  the file, the owner chooses.
- **O2.** D120 (the flows of lambda 1), D121 (the shared recipe), D122 and D123 (the supply gap and the reserve)
  are spec-silent or change a reading, are made in Phase 1 and are flagged for the owner's override before the
  phase that implements each (D122 and D123 before Phase 4, D120 and D121 before Phase 6). If a REF document does
  not load equal to the table in Phase 7, the owner decides whether the spec text or the document moves; the
  producer of the documents is not in the repository (N-274).
- **O3.** No CI exists (O3 of plan 0). The gate of Phase 7, the timed fuzz runs and the two-architecture
  determinism run are done by hand, on one architecture.
- **O4.** D124 (the shape and the place of `ViabilityError` in the sorted list), D125 (`Warnings` gated like phase
  6), D128 (the exported shape of the result, the largest new public surface of Stage 1) and D127 (the option
  names) are public-surface decisions; the owner may override before Phase 5 (D124, D125) and Phase 6 (D127,
  D128).
- **O5.** The speed of the Go solver is unmeasured (N-061); Phase 7 measures it. If `five` pooled is far slower
  than the prototype, the integer-preserving form (A2) is revisited before the gate constants are touched.
- **O6.** The double cost of `Validate` then `Warnings` then `NewWorld` (N-276) and whether the world row calls
  `checkViability` once; decided by the world row's plan with the measurement of Phase 7.
- **O8 to O12 (all closed: ruled by the coordinator, final, reversible).** Nothing in this plan waits for
  the owner. The reasons are stated where each ruling is used; this is the index.
  - **JC1 and O9: the start count is the prototype's.** A start counts at every tick of the open window
    (`t < YEAR`: 7 starts for a 60-day job). The `t + d <= year` reading is withdrawn. Reason: the
    prototype, REF 7.3, FR-VIA-048 and the committed warm documents all use it, and the steady state is
    periodic (a job begun late in one year completes early in the next; the annual flow of a stationary
    seasonal activity is its starts per year, the year boundary being arbitrary). VIA 3.1's sentence is
    a prose error that Phase 1 amends. REF's numbers stand (A3); no regeneration of VIA 15, REF 7.3 or
    `warm-second.toml` for it (A3, Phase 1 deliverable 2b, Phase 3).
  - **JC2: the lean reserve** uses every open start tick at the rate of 8.5.1 (D123), and a start whose
    completion is at or after the year end adds nothing there (rechecked after the revert: unchanged).
  - **JC3: the main recipe** is the one with the longest duration, ties by the largest inputs cost,
    then the lower recipe ID, with one clarifying sentence added to VIA 8.3.1 (D134).
  - **JC4:** an autarky region with an empty basket is not a programme and is listed in
    `UnpricedWages`; `Validate` never fails on it (D130).
  - **JC5:** V3 and the table are skipped when every basket good is an unsourced `Raw` marker
    (Phase 1 deliverable 2c, Phase 5).
  - **JC6:** simple cycles of single-maker commodities, a maker being a non-source activity, bounded,
    with a skipped-part warning past the bound (Phase 4).
  - **O10 (JC7):** an activity open on every tick counts as completing every tick, because 8.5.2 staggers
    the group (D122); it reproduces all three documents and needs the least code.
  - **O11 (JC8):** the solver lifts a floored or free price into the reference band and `warm-five.toml`
    is regenerated in Phase 7 (it is regenerated wholesale anyway); the validator keeps its band rule
    (Phase 1 deliverable 2e, Phase 6, Phase 7). Reason: the table must pass `Validate`, and the band
    rule is Stage 0's.
  - **O12 (JC9):** the Stage 0 and VIA 4.2.1 rules are unchanged; V3 runs unless V1 found
    `unproducible_in_realm` or V2 found `cycle_not_productive`. REF 5 expects d2 to give
    `unproducible_in_realm` (its table lists fuel, prepared, refined, staple, tool_a, tool_b and
    `role_cannot_work` for `forester`), which the unchanged overlay cannot give because `base.toml`
    marks `fuel` `raw`; so the d2 overlay changes in Phase 5 (a fixture edit) and REF's expectation
    stands. Plan 1-12 asserts the same set.
  - **Seasonal job (D134 (9)):** VIA 8.5.1 as written, the six document rows regenerated in Phase 7.
  - **Lanes (D127, D129):** Stage 1 supports only the free-trade set; `Lane`, `WithLanes` and
    `MaxLanes` are not published.
  - **Nominal warnings:** per region from the autarky programme; no warning without a quantity (Phase 1
    deliverable 2c).
- **O7.** The demand module's two-phase start (A5, N-277) and the module columns and rows are appended by their
  modules; whether `internal/steady` should be built with the phase one now to avoid a rewrite is the owner's
  (the plan recommends not: it is untestable without a pack with wants, and `one-demand` is DEM's).

## Decisions table

| ID | Decision | Written into | Before phase | Owner override |
|----|----------|--------------|--------------|----------------|
| D120 | The table's flows are those of lambda 1 (`x* / lambda*`) | VIA 5.1, 8.5.1 | 6 | O2 |
| D121 | A shared recipe: equal per-household share across the groups that list it; over-10 000 scaled down | VIA 5.2, 8.5.1 | 6 | O2 |
| D122 | The supply gap is read from the greedy start sequence of `J` (VIA 4.5 as written; the all-open-ticks reading gives 273, not 304; a completion past the year end wraps for the gap; an all-year activity has no gap, O10 (a)) | VIA 4.5 (worked check) | 4 | O2 |
| D123 | The lean reserve is the largest cumulative shortfall of the whole consumption (upkeep plus recipe inputs) over one year | VIA 8.4.1 | 4 | O2 |
| D124 | `ViabilityError` is its own type in the one sorted list; element type and per-rule path, value, bound, count fixed | CFG 7.1.1, VIA 10.1 | 5 | O4 |
| D125 | Phase 6 only on a clean configuration, on its canonical copy; `Warnings` gated the same way; one `checkViability`; the empty-offset rule at phase 6 entry | CFG 7.1.6, 7.2 | 5 | O4, O6 |
| D126 | The size gate arithmetic (no lane column in Stage 1) and the skipped-part warning | VIA 4.3, 4.6, FR-VIA-010; CFG 7.6 | 3 | O5 |
| D127 | The five functional solve options and their bounds; `Lane`, `WithLanes`, `MaxLanes` not published in Stage 1 | VIA 7.1, 9.4, 11 | 6 | O4 |
| D128 | The exported shape of `SteadyState`, `ProgrammeResult`, `Report` | VIA 7.1, 10.3 | 6 | O4 |
| D129 | Default autarky per region; pooled with free trade only (the complete zero-cost set, collapsed) | VIA 7.1 | 6 | |
| D130 | `ErrNoUpkeep` as FR-VIA-018 says; an empty basket never fails `Validate`; the autarky programme with an empty basket is JC4 | VIA 5.3 | 6 | JC4 ruled |
| D131 | The nominal fallback is per programme; the pooled table is rebuilt whole | VIA 5.3.4 | 6 | O2 |
| D132 | The census formula, pinned against `one.toml` | VIA 5.2, FR-VIA-028 | 6 | O1 |
| D133 | Agreement with the warm documents is by loaded equality (through `canonical()`, absent lists nil); the byte test is plan 1-12's | Plan 1-12, VIA 13.2 note | 7 | O2 |
| D134 | The composition rules of the warm table (whole-consumption reserve, stock combination, no job for a one-tick recipe, rounding down, no pool row) pinned against the documents; the main recipe is the longest duration, ties by the largest inputs cost (JC3); the seasonal job keeps 8.5.1 as written (six document rows regenerated in Phase 7) | VIA 8.3.1, 8.4.1, 8.5.1, 8.5.2 | 6 | O2 |

Rulings (coordinator, final, reversible): JC1 start count, the prototype's (A3, D122), JC2 reserve start
windows (D123), JC3 main recipe, longest duration (D134), JC4 empty autarky basket (D130), JC5 unsourced `Raw`
basket (Phase 5), JC6 cycle enumeration (Phase 4), JC7 all-year completion (D122), JC8 band lift (Phase 6), JC9 V3 sequencing and the d2 overlay (Phase 5).
