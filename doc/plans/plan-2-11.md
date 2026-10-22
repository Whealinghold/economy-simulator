# Plan: Stage 2, row 11: POP, `population.go` and `internal/population`

**Status: plan written (2026-10-22), against the contract `plan-2-0.md`. Not yet audited. Plans 1 to 10 were being written at the same time: every line of Consumes that names them is checked against their text in Phase 1 before any code.**
Sequences the POP row of [roadmap-v1.md](../roadmap-v1.md) Stage 2 ("`population.go` and `internal/population`") into eight phases: births and deaths as the formation and dissolution of households, driven by the household's own welfare ratio; the exact carried remainder; the death carry and its Weyl offset; the stake that funds a birth; the Malthusian fixed point as warnings, a snapshot figure and scenarios S37 to S41.

Rule tags read in full for this plan: architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1 (the tags the contract quotes; the files show the same).
Specs implemented: [spec-population.md](../specs/spec-population.md) (POP) FR-POP-001 to 041 and NFR-POP-001 to 004; read with [spec-estates.md](../specs/spec-estates.md) 2.1, 2.3, 2.6, 2.8 (the pool, the one draw function) and [spec-welfare-and-wages.md](../specs/spec-welfare-and-wages.md) 2.1 to 2.3 (the signal and its window); the touch points in `spec-spawner.md` 3.2, 8, 10.2, `spec-world.md` 6 (P7.1(b)), `spec-ledger.md` 7.2, `spec-config.md` 3.1 and 8.2.2, `spec-checkpoint.md` 4.2 (tags 3 and 14), `spec-observability.md` 4.1 and 5.3.
Notes read (index of `doc/notes.md`, tagged POP and ALL, with their updates; the file was not read in full and is not edited): N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217 (ALL); N-096, N-097, N-109, N-132, N-153, N-192, N-268, N-244, N-290 (POP). Dispositions are in Risks and open items.
Decision-ID block of this plan: **D420 to D439** (all used).

## Scope

**Builds.**

| Item | Where |
| --- | --- |
| The module file: state, `validatePopulation`, `populationWarnings`, the P7.1(b) step, the views' filler, the executor hooks, the two plan helpers of the default spawner, the snapshot rows, the report and the three `Warn`s, the checkpoint encoder and decoder, the scale slot | `population.go` (root; continuation files only if N-243 forces a split, E13) |
| Pure rules: schedule evaluation, effective rate, carry step, Weyl offset, floor and cap cuts, replacement ratio, gain test, mean rates | `internal/population` |
| Config types `Population`, `Knot`, `RegionPopulation`, `Modules.Population`, `DefaultPopulation`, `DefaultRegionPopulation`; hash id 3; the file table `[modules.population]` | `config.go`, `defaults.go`, `hash.go`, `config/mapping.go` |
| Events `PopulationChanged`, `PopulationScaled` (+ `ScaleKind`); causes `population`, `births`; reason `deaths` | `event.go`, `internal/ledger/registry.go`, `ledger.go` |
| Hook fields `RoleView.PopBirths`, `RegionAgent.PopDeaths`; the plan order; the stake; the effective `Renewal` of 0 | `spawner.go`, `phase_exit.go` (one guarded line each) |
| Overlays `population.toml`, `population-low.toml`; module-alone golden `testdata/golden/population/`; scenario tests S37 to S41 | `testdata/`, root `_test.go` files |

**Leaves out, with the owner.** The setters `SetBirthScales` and `SetDeathScales` and their events' emission (INT, plan 17: this plan keeps the state and the event type); the role ceiling `Role.Ceiling`, `Role.Needs` and `PopulationChanged.BirthsCeiling` (SET, Stage 3: FR-POP-041 and the `Needs` half of FR-POP-007 are tested then); a body's places (ORG, Stage 3); migration (MIG, plan 14: no stake, no pool draw); the fixed factor itself (RES, plan 9); the bands of S37 to S41 and tier L (STY, plan 19); the metrics `household.births` and `household.deaths` (OBS adapter, plan 22: the core defines events only, observability 5); the maturity pipeline, per-role schedules and seasonal rates (`spec-population.md` 18 items 1, 2, 6); the margin of `no_fixed_factor` (18 item 9).

## Decisions

Each cites its source. A plan decides nothing: where one below resolves a gap it says so and the owner is named in Risks.

- **D420.** The Go types are `Population`, `Knot` and `RegionPopulation` (fields `BirthSchedule`, `DeathSchedule []Knot`, `MinHouseholds`, `BirthStake`, `Regions []RegionPopulation`; `Region`, `BirthScale`, `DeathScale`, `HouseholdCap`), not `PopulationConfig` as `spec-population.md` 19 says. Source: the parameter-owner table of `doc/specs/README.md` and `registry_test.go` name the scope by Go type; the spec wording is corrected (Phase 1, E1).
- **D421.** Build order is not row order: `Population` is id 3 of `Modules` but is built after `Demand` (1), `MoneySupply` (2), `Estates` (25), `Resources` (5) and `Subsistence` (8). Only the hash id follows the row: `pairsModules` writes the pair 3 in id order. Every other line (the `Modules` field, `canonical`, `modulesOn`) is appended in Go order after what plans 5 to 10 added, as plans 9 and 10 do; no existing line moves or changes. Source: `spec-config.md` 3.1 and 8.2.2 ("the order in which modules are built does not change it"); `architecture.md` 2.10.5 (owed clarification, E6). Test: the hash of a configuration with another module on and population off is unchanged by this change.
- **D422.** P7.1(b) is a sub-step of the position P7.1 (owner SPN), not a position. `tick.go` and its position tables do not change. The one call is `w.popFill(g, v)` at the end of `buildRegionView` (`phase_exit.go`, the WLD continuation file of the entry step), guarded by `w.pop != nil`, before `Decide`, once per region in ascending `RegionID`. Source: `spec-world.md` 6 P7.1, 6.9; `architecture.md` 2.10.3 (owed clarification, E6).
- **D423.** The death carry is a slice of the module indexed by `AgentID`, grown at the compute step with `Offset(id, D)` for every id it has not seen. There is no hook in the creation paths (`construct.go`, `spawnEntrant`, INT's `AddAgent`), so a newborn agent "takes the offset of its own ID" by construction (FR-POP-010) and an external agent's unused entry costs one write. The checkpoint writes it in the agent record (tag 3) through `deathCarryOf` (plan 2 D247, which writes the field always). `agent` gains no field. The sixth fan-in `agentJoined` of plan 5 is not used: it would only grow the slice, which the compute step already does lazily. Source: `architecture.md` 2.10.1, 13; N-244; N-128; `spec-population.md` 3.8.3, 11.
- **D424.** The signal is `a.history.Ratio(U, role.DefaultRatio)` with `U = cost.tick(upkeep of the role in the region, region prices, a.weight)`: the valuation `readBuf.fill` uses (`welfare.go`), read from the rings and not from `RoleRow.MeanRatioBP`. A guard test fails if `population.go` mentions a mean-ratio identifier (FR-POP-040). Source: `spec-population.md` 3.4, 3.4.3; `spec-welfare-and-wages.md` 2.1, 2.2.
- **D425.** The hook fields are appended last: `RoleView.PopBirths int64`, `RegionAgent.PopDeaths int64`. `DefaultSpawner.Decide` gains `c.popDeaths()` between `exits()` and `switches()` and `c.popBirths()` after `renewals()`; both are methods of `decider` defined in `population.go` and are no-ops at 0. Deaths are `DecideReduce` with `CausePopulation`, births `DecideEnter` with `CauseBirths`, no `EntryShare` cap and no viability gate. Source: `spec-spawner.md` 3.2, 8.2, 10.2, FR-SPN-028, 031; N-268.
- **D426.** Causes and reasons are appended together (N-290): `CausePopulation`, `CauseBirths` after `CauseRenewal`; `ReasonDeaths` in `event.go`; `ledger.Deaths` after the last declared reason present when this lands (its table position does not affect `BoundaryFlow` order, the reason being internal; plan 5 D311: declared reasons are appended in build order; the rank column gives display order), row money and goods, agent to estate; `newLedger` declares it only with the module on; `reasonNames` gets its row. The executor accepts a `DecideReduce` with `CausePopulation` only when the module is on (else `unsupported_cause`), and the ledger reason of a reduction comes from one function in `population.go`, `reductionReason(cause)`, called by `shed` (`starvation` for the ladder's causes, `deaths` for `population`), so E6 licenses a one-line call in the executor and no logic of the module lies outside its file. Source: `spec-ledger.md` 7.2.2, FR-LED-051, 055; `spec-spawner.md` 4.1.1.
- **D427.** The money a born household brings is `stake = trunc(StartMoney * BirthStake / 10 000)` (checked 128-bit), reason `births`, whether it joins a host or becomes a new agent, no goods. One function in `population.go`, `entryMoney(g, d) (Money, ledger.Reason)`, is called by `execEnter`, `enlarge` and `spawnEntrant`; with the module off it returns exactly what Stage 1 does, so the off golden cannot move. A stake of 0 is not limited by the pool (`occupation.Funded` already says so). Source: `spec-population.md` 4.3; FR-POP-017.
- **D428.** With the module on the effective `Renewal` is 0: `buildRegionView` puts 0 in the view and `advanceRenewal` reads `w.renewalBP()`, both from one function in `population.go`. It is a rule constant (`spec-population.md` 4.6). Births are never netted against renewal.
- **D429.** A birth is an `Enter` like any entry, so its unfunded households are counted in `EntryLimited.Unfunded` and also in `PopulationChanged.BirthsUnfunded`, and they enter the run of the "entry unfunded" `Warn`. Source: FR-SPN-008 covers every `Enter`; `spec-population.md` 4.3. A reading, not a decision of a spec: open item O3.
- **D430.** Order of the counts per region: for each agent in ID order `due = Step(deathCarry)`; `DeathsDue` is the sum of dues; cut to `Weight - Unfed` (`DeathsCut`); cut so that the group (region, role) stays at or above `MinHouseholds` counting earlier takes (`DeathsHeld`); then per role in role order the birth carry gives `BirthsDue`; cut to the region's `HouseholdCap` less the households after the module's deaths (`BirthsCapped`); the funding cut is the executor's (`BirthsUnfunded`). Source: `spec-population.md` 3.8, 4.4, 4.4.4 (the ceiling step is SET's and absent).
- **D431.** A region scale is `effBirth[g]`, `effDeath[g]` (dense by `RegionID`), initialised from the configured row, absent row 10 000, read every tick, set by `setPopulationScale(region RegionID, kind ScaleKind, bp int64)` (state only; INT emits `PopulationScaled`; `g` in this plan's text is the dense region index of that `RegionID`, as in plan 17 D542). A scale is saved only when it differs from the configured value. Source: FR-POP-039; `spec-population.md` 6.6, 11.
- **D432.** The three `Warn`s use `advanceRun(before, active, true)` (pattern A) with runs per region ("region extinct", "births unfunded") and per (region, role) ("deaths held"); "births unfunded" is active when the births handed to the executor are above 0 and `BirthsDone` is 0. The runs are saved. Source: `spec-population.md` 10.3, 11; `spec-observability.md` 7.3 rows 10, 11.
- **D433.** Snapshot: `RoleRow` gains `BirthRateBP`, `DeathRateBP`, `NaturalIncreaseBP` (household-weighted means at the agents' ratios, over the households whose ratio exists, truncated toward zero) and `BirthCarry`; `RegionRow` gains `HouseholdCap`; `Snapshot` gains `Population PopulationState{ReplacementRatioBP int64; HasReplacement bool; GainBP int64}` with `GainBP = trunc(r* * dRate / dRatio)` (the gain `k` of 5.4 at `eta` 1, in basis points a year). All are counts, amounts or bools, zero with the module off, so omitted from the golden rendering. `HasReplacement` and the unit of `GainBP` are not in `spec-observability.md` 5.3.10: E8.
- **D434.** `replacement_ratio` (with `r*`'s segment) and `renewal_ignored` are `Warning` rows like `window_extended` (`validate.go`); `Warning` carries a path and a rule only, so the path ends in the lower knot index of the segment and the value is read from the snapshot. Source: `spec-population.md` 5.7, 4.6.1, 9.2. Open item O6.
- **D435.** The file table is `[modules.population]` with `birth_schedule`, `death_schedule` (arrays of `{ ratio, rate }`, replaced whole by a later layer), `min_households`, `birth_stake`, and `[modules.population.region.<id>]` with `birth_scale`, `death_scale`, `household_cap`. Only `config/` reads it. Source: `spec-population.md` 8; `spec-config.md` 6.2.1.
- **D436.** `Population.Regions` follows the region-keyed mechanism of REG: unknown region `ErrUnknownRegion`, second row `ErrDuplicateID`, absence the default, rows sorted by region in the hash and the checkpoint, resolved once into dense arrays at construction. Bounds are `ErrOutOfBounds` with the path; there is no `ErrModuleDependency` and no new sentinel. Source: `spec-regions.md` 5.3; FR-POP-025 to 027.
- **D437.** The scenarios S37 to S41 are root integration tests that assert the exact checks (conservation, disjoint exposure, the warnings and counters named by the scenario, no extinct-region `Warn` where S38 says so) and record the rest; bands wait for STY (plan 19) and tier L. They skip under `-short` and state their level. Source: `spec-population.md` 14.4; `spec-reference-chain.md` 7.1.5; FR-REF-022, 023.
- **D438.** Fixtures: `overlay/population.toml` is the placeholders of `spec-population.md` 8 with `world.renewal = 0`, used by S37 to S39; `overlay/population-low.toml` crosses at 9 500 (below the closed ceiling, so no `replacement_above_closed_ceiling`) and carries one `HouseholdCap` row, and is the module-alone overlay (births, deaths and the cap all occur) and the second pack of S40. REF names one overlay; the second is E10.
- **D439.** `population.Rate` evaluates the schedule with the piecewise-linear evaluator of `internal/curve`, the one helper `spec-resources.md` 4.7.1 and `spec-population.md` 3.5.1 share; this package adds the scale, the clamp at 10 000 and the checked conversion. If the evaluator's signature differs from what Phase 1 finds, only `Rate` changes.

## Phases

### Phase 1: the documents, and the checks of what is consumed (no code)

Goal: the specs are true before the code depends on them, and every consumed symbol exists. Touches: nothing but the edits below (the supervisor applies them; this plan edits only itself).

| ID | File, section | Edit owed |
| --- | --- | --- |
| E1 | `spec-population.md` 19 | `PopulationConfig` becomes `Population` (D420) |
| E2 | `doc/specs/README.md` Parameter owners | row `Regions` / `Population (the region-keyed rows)` / `spec-population.md` section 8 (N-016; `registry_test.go` needs it) |
| E3 | `spec-regions.md` 5.1.3 | row `Modules.Population.Regions`, key region, owner POP 8, mode scale (`BirthScale`, `DeathScale`) and replace (`HouseholdCap`) |
| E4 | `spec-config.md` 8.2.2 | ids: `Population` BirthSchedule 1, DeathSchedule 2, MinHouseholds 3, BirthStake 4, Regions 5; `Knot` Ratio 1, Rate 2; `RegionPopulation` Region 1, BirthScale 2, DeathScale 3, HouseholdCap 4 |
| E5 | `doc.go` glossary | birth, death, rate schedule, replacement ratio, exposed household, natural increase, birth stake, household cap, death carry, birth carry, fixed factor, gain, and the type and event names `Population`, `RegionPopulation`, `PopulationState`, `PopulationChanged`, `PopulationScaled` (`TestGlossary_DeclaredTypesNamed`; architecture 3.2: before any code uses them). `Knot` is not added: the root type `Knot` is named by the `Knot` entry of plan 9's E-row `doc.go`, and plan 9 is applied first (`TestGlossary_Vocabulary` fails on a duplicate) |
| E6 | `architecture.md` 2.10.3, 2.10.5 | the sub-step of a position is not a position only the 2.10.5 build-order sentence (D421): where build order differs from the `Modules` row, only the hash id follows the row, the Go lines are appended in Go order and nothing is renumbered. The sub-step rule (D422, 2.10.3) is item (e) of plan 5 E1 and needs ADR 0007 decision 2 in plan 5 E11, not repeated here. Same 2.10.5 change as plan 7 edit 3; one fresh tag |
| E7 | plan-2-2 D247 (not a spec) | plan 2 writes `DeathCarry` always and leaves the value source to the module ("the modules' plans replace the value source"); this plan supplies `w.deathCarryOf(a)` (the slice value, or `Offset(id, D)` when the module is off, as R10 says). Confirm the name with plan 2 (O1) |
| E8 | `spec-observability.md` 5.3.10 | add `HasReplacement`, define `GainBP` (D433) |
| E9 | `spec-spawner.md` 13.1.0 or 8.2 | one sentence: the unfunded households of births are counted in `EntryLimited.Unfunded` too (D429) |
| E10 | `spec-reference-chain.md` 7 overlay row | "18.3" becomes 8; add `overlay/population-low.toml` and `testdata/golden/population/` (D438) |
| E13 | `architecture.md` 2.6 and `AGENTS.md` Layout (same files in both lists; fresh rule tag) | conditional on a Lizard failure on `population.go` (N-243) when the code is written: the supervisor adds the continuation files (POP) before the split is made, as plan 15 E4 does |
| E12 | `spec-population.md` 5.2, 9.2 | wire `resource_overshoot` and `resource_unstable_plan` (plan 9 D399, FR-RES-044): state that POP fills `endowment.Plan` (`r*`, `dR`, `dX`) from its schedules and reports both when the module is on |
| E11 | `spec-population.md` 5.7 | state the case `g` = 0 at a knot (the crossing is the first `x1` with `g(x0) < 0 <= g(x1)`) and a crossing from positive to negative only (reported as `no_replacement_ratio`) |

Checks (read-only, recorded in the PR; a failure stops the row and goes to the supervisor): `grep` finds in the tree as it stands: `internal/curve` with a piecewise-linear evaluator; `MaxKnots` in `bounds.go`; the declared-reason plumbing in `newLedger` (`ledger.Config.Declared` set from the modules on); the module-alone golden harness for `testdata/golden/<module>/`; the `[modules]` dispatch in `config/mapping.go`; the section table and the hosted-field value source of D247 in `checkpoint_sections.go`; `Recipe.Draws` and `overlay/resources.toml`; `Modules.Demand`; `Region.PoolRelease`. Also: `grep -n "PopDelta"` finds nothing live (the spawner spec already carries the gross hook).
Exit: E1 to E13 applied or ruled; the check list shows each symbol with its plan.

### Phase 2: `internal/population` (stdlib, primitive structs, no root import)

Files: `doc.go`, `rate.go`, `carry.go`, `cut.go`, `replacement.go` and `*_test.go`. Imports `internal/curve`, `internal/checked`, `internal/ordering` only.
Surface (architecture 2.9; all overflow is reported, never wrapped):
`type Knot struct{ Ratio, Rate int64 }`; `Rate(knots []Knot, ratio, scaleBP int64) (rate int64, clamped, ok bool)`; `Term(exposed, rate int64) (int64, bool)` (the checked product `H_a*b_a`; a role's numerator is the checked sum of its agents' terms); `Step(carry, numerator, denom int64) (events, next int64, ok bool)`; `Offset(id, denom int64) int64`; `Cut(due, fed, group, floor int64) (take, cut, held int64)`; `Allow(due, households, cap int64) (allowed, dropped int64)`; `Replacement(birth, death []Knot) (ratio, lo, hi int64, ok bool)`; `GainWarn(rstar, dRate, dRatio, window, ticksPerYear int64) bool`; `Gain(rstar, dRate, dRatio int64) int64`; `Means(weights, birth, death []int64, present []bool) (b, d, n int64)`.
Tests (external package, table-driven, numbers with their arithmetic):
- `TestRate_Piecewise` FR-POP-004: below, above, between, a single knot, a negative ratio, truncation toward zero on a falling segment (`450 + trunc(-400 * 1 / 8000)` = 450), `14 000` gives 150 (`spec-population.md` 12.1).
- `TestRate_ScaleAndClamp` FR-POP-005: `min(10 000, trunc(700 * 30 000 / 10 000))` = 2 100 (12.2); 9 000 times 2 clamps and says so.
- `TestStep_Carry` FR-POP-008, 009, 011, 012: agent 17, `D` 3 650 000: `Offset(17)` = 1 849 008, carry after tick 1 is 1 854 408, one event at tick 334 leaving 2 608; the role of 1 000 households at 250: one event at tick 15 leaving 100 000; a numerator of 0 keeps the carry; a property over 10 000 ticks of changing exposure and rate: events equal `floor((carry0 + sum) / D) - floor(carry0 / D)` exactly.
- `TestOffset_Range` FR-POP-010: in `[0, D)` for ids 0 to 10^5, and `Offset(id, D)` equals `ordering.Weyl32(id) * D >> 32` through the 128-bit product.
- `TestCut_FloorAndFed` FR-POP-013, 015: the cases of 4.4.1 and 4.4.3 (`MinHouseholds` 1 holds the last household: `held` 1).
- `TestAllow_Cap` FR-POP-014: 598 households, cap 600, 3 due: 2 allowed, 1 dropped (12.2); cap 0 means none.
- `TestReplacement` FR-POP-029: the defaults give 12 000 (12.1) and the habitat 12 000 (12.2); `x0 + trunc(-g0 * (x1 - x0) / (g1 - g0))`; a schedule pair with no negative-to-positive crossing is `ok` false; `g` = 0 at a knot.
- `TestGainWarn` FR-POP-030: 12.1 gives 144 000 000 against 29 200 000 000 (false); 12.2 gives 288 000 000 against 76 800 000 000 (false); equality is false, one more is true.
- `TestMeans` FR-POP-034; `FuzzRate`, `FuzzStep` (never panic, `0 <= rate <= 10 000`, `0 <= next < denom`, `ok` false only on overflow).
Exit: `go test -short -race ./internal/population/`; coverage at least 90%; Lizard clean; `gomu` on the package in Docker (`--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4 --workers 4`, see `~/.claude/CLAUDE.md`), survivors listed.

### Phase 3: the configuration surface

Goal: the module can be configured, hashed, loaded and validated; nothing runs.
Files: `config.go` (types; `Modules.Population` appended in Go order), `defaults.go`, `hash.go` (`pairsModules` id 3 and `pairsPopulation`, `pairsKnot`, `pairsRegionPopulation`), `Config.canonical` (clone both schedules and sort `Regions` by region), `validate.go` (one call in `fields()`, one in `warnings()`), `population.go` (`populationRules`, `populationWarnings`), `construct.go` (`modulesOn` gains `"population"` in Go order), `config/mapping.go` (D435), `registry_test.go` (delete the entries `Modules.Population`, `Population`, `Knot`, `RegionPopulation`), the config integration test (pinned hashes of the two composed packs of Phase 7).
Tests:
- `TestValidate_PopulationBounds` FR-POP-025, 026 (one case per rule, only that rule broken: no knots, nine knots, ratios not strictly increasing, a ratio above `MaxBasisPoints`, a rate above 10 000, `MinHouseholds`, `BirthStake`, both scales and the cap out of bound; each `errors.Is(err, ErrOutOfBounds)` and the path).
- `TestValidate_PopulationRegionRows` FR-POP-027, FR-REG-035 (`ErrUnknownRegion`, `ErrDuplicateID`, absence is the default with no warning).
- `TestWarnings_ResourcePlan` FR-RES-044: with `Resources` and the module on, `popWarnings` fills `endowment.Plan` from the schedules (`r*` from `Replacement`, `dR`, `dX` from the knots at `r*`) and `resource_overshoot` and `resource_unstable_plan` fire on the cases of plan 9's `TestUnstablePlan_Shapes` and are silent with the module off.
- `TestWarnings_Population` FR-POP-020, 028, 029: `no_replacement_ratio`, `schedule_direction`, `replacement_above_closed_ceiling` (fires with `Demand` nil and r* above 10 000; silent with `Demand` on), `population_gain`, `no_fixed_factor` (no recipe `Draws` and no cap), `population_without_upkeep`, `replacement_ratio`, `renewal_ignored` at `Renewal` 10 000, `renewal_with_population` at a non-default value above 0, neither at 0; none fails `Validate`.
- `TestHashConfig_PopulationId3` FR-CFG-069, FR-CFG-105: off leaves `pinnedReferenceHash` unmoved; on writes the pair 3 with schedules in knot order and rows in region order; a configuration with `Demand` on and population off hashes as before this change (D421).
- `TestCanonical_PopulationCopies` (copy on construction, `spec-config.md` 3.5).
- `config`: `TestLoad_Population` (layers merge by key, a schedule is replaced whole, unknown key rejected, a knot with a missing `rate`), `TestFileAndGo_Population`; seeds for `FuzzLoad`.
Exit: `testdata/golden/off/` and `pinnedReferenceHash` unmoved; `registry_test.go` green with the four entries deleted; `go test -short -race ./...`.

### Phase 4: the module: the step, the hook, the executor

Goal: with the module on, births and deaths happen; with it off nothing changes.
Files: `population.go` (state `population{denom, deathCarry, birthCarry, effBirth, effDeath, cap, scratch}`, `newPopulation`, `popFill` split into `popRates`, `popDeaths`, `popBirths` helpers under the Lizard limits, `noteDeaths`, `noteBirths`, `entryMoney`, `renewalBP`, `setPopulationScale`, the two `decider` methods), `world.go` (the field `pop *population`, built in `construct.go`), `phase_exit.go` (the call in `buildRegionView`; `refusal` accepts `CausePopulation` when `w.pop != nil`; `shed` takes a reason; `execEnter`, `enlarge`, `spawnEntrant` call `entryMoney`; `execReduce`, `execRemove`, `execEnter` call the notes), `spawner.go` (the two fields; two lines in `Decide`), `event.go`, `internal/ledger/registry.go`, `ledger.go`.
Tests (root; unit tests build the state by hand in internal tests, packs run in integration tests):
- `TestPopulation_OffLeavesNoTrace` FR-POP-001: no state, no event, `Renewal` keeps its meaning; the off golden unchanged.
- `TestStep_OrderAndHook` FR-POP-002, 019: a hand-written fake spawner in the test records the views: regions ascending, the hooks set before `Decide`, roles in role order; the default's plan lists ladder reductions, then deaths, then switches, then births.
- `TestStep_SignalIsOwnRatio` FR-POP-003, 040 (two agents of one role with different rings get different rates; an absent ratio is `NoRatio`; the source guard on mean-ratio identifiers).
- `TestStep_Exposure` FR-POP-006, 007, 023: weight 40 with `Unfed` 4 exposes 36; an external agent exposes none; a `Closed` role's birth carry does not advance and its deaths run.
- `TestStep_FloorCapAndExtinct` FR-POP-013, 014, 015, 038: the cuts in the order of D430; a region with no households gets no birth.
- `TestExec_Death` FR-POP-016, 018, 022: a reduction sheds `floor(x * w' / w)` of money and each good to the estate with reason `deaths`; a weight-1 agent is removed with `AgentBankrupted.Cause` `population` and the reasons `estate_money`, `estate_goods`; the ledger `Audit` is clean.
- `TestExec_Birth` FR-POP-017, 021: pool 30 000, `EntryPoolShare` 5 000, stake 5 000: funded, pool 25 000 (12.1); pool 3 000: none created, `BirthsUnfunded` 1; stake 0 is unlimited; placement in the host of smallest weight below the scale, else a new agent of weight 1 with the newcomer's state of `spec-spawner.md` 7.4.
- `TestRenewal_ZeroWhileOn` FR-POP-020; `TestExec_PopulationCauseRefusedWhenOff` (N-290: `unsupported_cause`, counted in `Refused`).
- `TestScale_EffectiveReadEachTick` FR-POP-039, 005 (set to 30 000: the rate at 9 000 is 2 100 the next tick; cleared: back to the row).
- `TestPopulation_DrawsNothing` FR-POP-036, NFR-POP-004 (the PCG state marshalled before and after `popFill` is equal; the order of agents and roles is by ID, no map).
- `TestPopulation_Conservation` FR-POP-022, 037: seeds 1 to 4, the `population-low` pack, after every tick money and goods are conserved exactly and the household count rebuilt from `AgentSpawned`, `WeightChanged` and `AgentBankrupted` equals the sum of weights.
- `TestPopulation_NoAllocation` NFR-POP-001 (`testing.AllocsPerRun` on `popFill` with the default observer, after the carry slice has grown); `BenchmarkPopulation_Fill` (`b.Loop`, `b.ReportAllocs`) over 1 000 agents of weight 1 and of weight 10^6 for NFR-POP-002.
- Worked examples as one internal test: agent 17 of 12.1 and the habitat of 12.2 through `popFill` (FR-POP-008, 011).
Exit: `go test -short -race ./...`; coverage; the new packs run in the long suite; `testdata/golden/off/` unmoved.

### Phase 5: observability and the snapshot

Files: `population.go` (`popReport`, the runs, the `Warn`s, the `Debug` line, the snapshot rows), `phase_exit.go` (`entryRegion` calls `w.popReport(g)` after the `EntryLimited` delivery), `snapshot.go` (fields of D433), `internal/logkeys/logkeys.go` (`Births`, `Deaths`; the rows exist in observability 3.3.2), `cardinality_test.go` (cases for `PopulationChanged` and the new causes).
Tests: `TestPopulationChanged_Emission` FR-POP-031 (once per region and role, in ascending role order, after `EntryLimited`, never when every listed count is 0, no map); `TestPopulationChanged_DoneEqualsExecuted` FR-POP-032 (`DeathsDone` equals the households removed with cause `population`, `BirthsDone` the households added with cause `births`, over `WeightChanged`, `AgentSpawned`, `AgentBankrupted`); `TestWarns_Population` FR-POP-033 (capturing handler: level, constant message, keys, the first tick and every tenth, bounded by regions and roles, no `Info` in the tick, the counterpart field of each `Warn`); `TestSnapshot_Population` FR-POP-034 (means, natural increase, `BirthCarry`, `r*`, gain, zero with the module off); `TestObservation_DoesNotChangePopulation` NFR-POP-003; an event-order test for a region with deaths, births and a ladder reduction.
Exit: the staged catalogue test: `PopulationChanged` and `PopulationScaled` leave `notBuiltEvents`; `PopulationChanged.BirthsCeiling` is listed in `notBuiltFields` for SET.

### Phase 6: the checkpoint section and the fork test

Files: `population.go` (`saveX` and `loadX` of tag 14, `spec-population.md` 11; `deathCarryOf` for the agent record's `DeathCarry`), one row in `checkpointSections` and its tag deleted from `notBuiltSections`, a cross-check in `checkpointCross`, the value source of the agent record's `DeathCarry` (D243, D247 of plan 2).
Content (`spec-checkpoint.md` 4.2 tag 14; sparse, fixed order, no map): birth carry per (region, role) in ascending `RegionID` then role; effective scales that differ from the row; the three runs. Tag 3: the death carry of every agent in ascending ID. A value a row's default would give is corrupt (CHK R10); a carry at or above `D` is corrupt.
Tests: `TestCheckpoint_PopulationRoundTrip` FR-POP-035; `TestCheckpoint_PopulationRejectsCorrupt` (carry at `D`, a negative run, an out-of-order row); `TestFork_Population` FR-POP-035 (the `population-low` world, fork at N in {0, 37 (inside a job), 211 (inside a season), after a call between ticks} through `forkAtN` of plan 2 D258, one shock scale set and one run non-zero at the fork, resumed and uninterrupted runs equal in state hash, events and snapshot).
Exit: the fork test passes with `-race`.

### Phase 7: fixtures, goldens and the scenarios

Files: `testdata/chains/overlay/population.toml`, `population-low.toml`; `testdata/golden/population/` (the files the harness of plan 5 writes for a module, recorded once with `-update` on `one` + `population-low`, seed 1, 400 ticks, with the explanation testing.md 7.2 asks); `scenario_s37_test.go` to `scenario_s41_test.go`; the pinned hash of the two composed packs.
Tests: the scenarios take their seeds from `scenarioSeedCount()` of plan 1 (S37 to S41), and the `population-low` world joins the invariants matrix (REF 7.5) and the determinism matrix (REF 8.2). `TestGolden_PopulationAlone` FR-POP-001 (testing.md 7.1.1), `TestPositionHash_PopulationAlone`; the proof that `testdata/golden/off/` and `pinnedReferenceHash` did not move is the existing off tests, unedited and green. Scenario tests, each skipped under `-short`, naming scenario, pack, seed and measure on failure:
- S37 (`one-demand` + `resources-binding` + population; the solver does not report `N*` (spec 18 item 3, N-109), so `N*` is the household count a 2 000-tick pre-run of the same pack reaches, and the two runs start at 0.6 and 1.6 of it by `world.scale`): exact checks FR-POP-022, 037; the ladder's and the schedule's sets are disjoint every tick (FR-POP-006); the end ratio and the crossing count are reported, not asserted (bands are STY's).
- S38 (a yield shock cuts income, relief and ladder on): the extinct-region `Warn` does not fire; households and ratio recorded; the run is reported for the F3 regime (uses plan 7's relief and plan 17's shock only through the test's own `YieldShocked` script if INT is absent: O5).
- S39 (`resources-binding`, closed region, empty pool, `BirthStake` 10 000, 5 000, 0): `BirthsUnfunded` is non-zero when the population stops; the stop is reported against `M / stake` (FR-POP-017).
- S40 (defaults with demand off, r* 12 000; and `population-low`; and `Renewal` set to a non-default value above 0): the warning `replacement_above_closed_ceiling` is reported for the first, the second shows a bounded unfed share in `PopulationChanged.Unfed`, `Renewal` is ignored and warned only at the non-default value (FR-POP-020).
- S41 (RES-S1, population on `resources-binding.toml`, `TestScenarioS41_ResourcePopulationLoop`; owed to this plan by plan 9 D398): exact checks FR-POP-022, 037; the loop of households, draw and yield is reported against the plan of `endowment.Plan`, bands are STY's.
Exit: scenario outcomes recorded in the PR as Stage 1 row 12 did (pass or not, with the measure).

### Phase 8: exit gate

`gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; `go mod tidy` no diff; `lizard -l go -C 10 -a 8 -L 100 -w` on `population.go`, `internal/population/*.go` and the lines added to `phase_exit.go`, `spawner.go`, `validate.go`, `hash.go`; coverage at least 90% for the root and for `internal/population`, and the runs R and F of plan-2-1 D229 R and F exactly as plan-2-1 D229 states them; `gomu` on `internal/population` in Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4 --workers 4`; the registry and catalogue tests with the entries of this row deleted; `testdata/golden/off/` and the pinned hash unmoved. Fills the PR sections "Testing" and "Observability" from this file.

## Catalogue appends

| File | Appends | Position | Id or tag | Fixed by |
| --- | --- | --- | --- | --- |
| `config.go` | `Modules.Population *Population`; types `Population`, `Knot`, `RegionPopulation` | appended in Go order (D421) | hash id 3 | `spec-config.md` 3.1, 8.2.2 |
| `hash.go` | `pairsModules` case; `pairsPopulation` (ids 1 to 5), `pairsKnot` (1, 2), `pairsRegionPopulation` (1 to 4) | between ids 2 and 4 in the function body | 3; E4 | `spec-config.md` 8.2.2, FR-CFG-069 |
| `defaults.go` | `DefaultPopulation()` (schedules of 8, `MinHouseholds` 1, `BirthStake` 10 000), `DefaultRegionPopulation(region)` (scales 10 000, cap 0) | after the other defaults | none | `spec-population.md` 8 |
| `bounds.go` | none: `MaxKnots` is plan 9's (Consumes) | with the other limits | none | `spec-config.md` 4.1 |
| `validate.go` | one call in `fields()`, one in `warnings()` | last of each | none | `spec-population.md` 9 |
| `errors.go` | nothing | none | none | 9.1 (no new sentinel) |
| `event.go` | `CausePopulation`, `CauseBirths`, `ReasonDeaths`, `ScaleKind` (`ScaleBirth`, `ScaleDeath`), `PopulationChanged`, `PopulationScaled` | after the last of each present | none | `spec-observability.md` 4.1, 4.2.3; N-268 |
| `option.go` | nothing | none | none | none |
| `tick.go` | nothing (D422) | none | none | `spec-world.md` 6 P7.1 |
| `snapshot.go` | `RoleRow` fields, `RegionRow.HouseholdCap`, `Snapshot.Population` | last in each struct | none | `spec-observability.md` 5.3.2, 5.3.10 |
| `ledger.go`, `internal/ledger/registry.go` | `ledger.Deaths` (declared), its row and name; It uses the declared-reason mechanism of plan 5, so the walked `off` array does not change and `golden/off/*.positions.txt` does not move. | after the last declared reason | none | `spec-ledger.md` 7.2, FR-LED-055 |
| `construct.go` | `"population"` in `modulesOn` | Go order (D421) | none | `spec-world.md` 5.8 |
| checkpoint | section 14 (carries, scales, three runs); the death carry in the agent record | tag 14; tag 3 | 14, 3 | `spec-checkpoint.md` 4.2 |
| log keys | `births`, `deaths` in `internal/logkeys`; `ledger.Deaths` uses the declared-reason mechanism of plan 5, so the walked `off` array does not change and `golden/off/*.positions.txt` does not move | after the existing keys | none | observability 3.3.2 (row exists) |
| metrics | none in the core (`household.births`, `household.deaths` exist in 5.3 and are fed by `PopulationChanged`) | none | none | observability 5 |
| `config/mapping.go` | the `[modules.population]` branch | under `modules` | none | D435 |

## Cross-plan contract

Consumes (Phase 1 resolves each against the plan named; a line that does not resolve stops the row):
- Modules.Demand (config.go) <- plan-2-5 (the condition of `replacement_above_closed_ceiling`; the demand packs `one-demand` of S37)
- module-alone golden harness for testdata/golden/<module>/ (golden_test.go) <- plan-2-5
- declared-reason plumbing in newLedger (ledger.go) <- plan-2-5
- the [modules] dispatch of the loader (config/mapping.go) <- plan-2-5
- the rebalanced reference pack one and its warm start (testdata/chains/) <- plan-2-8
- Region.PoolRelease, Region.ReliefBelow (estates.go), ReliefPaid (event.go) <- plan-2-7 (S38 only)
- ledger.Relief, ReasonRelief (internal/ledger/registry.go, event.go) <- plan-2-7 (S38 only: the relief transfers the test reads)
- Recipe.Draws, Modules.Resources, overlay/resources.toml (config.go, testdata/chains/overlay/) <- plan-2-9
- MaxKnots (bounds.go) <- plan-2-9
- endowment.Plan, endowment.Overshoot, endowment.UnstablePlan (internal/endowment/plan.go) <- plan-2-9
- overlay/resources-binding.toml (testdata/chains/overlay/) <- plan-2-9 (S37, S39, S41)
- agentCount, groupWeights, groupShape, checkCensus (scale.go) <- plan-2-4 (the weight and household-count checks of the tests)
- internal/curve piecewise-linear evaluator (internal/curve) <- plan-2-9
- checkpointSections, notBuiltSections, checkpointCross, the agent-record hosted fields (D247: `DeathCarry` always written) and `agentTails` (checkpoint_sections.go), forkAtN, forkCase (checkpoint_fork_test.go) <- plan-2-2
- internal/codec primitives (internal/codec) <- plan-2-2
- `invariantSeeds`, `scenarioSeedCount` (`scenario_helpers_test.go`) <- plan-2-1 (the invariants-matrix row of `population-low`)
- Ring.Ratio (internal/welfare/ring.go) <- Stage 1
- ordering.Weyl32 (internal/ordering) <- Stage 0
- checked.MulDiv (internal/checked) <- Stage 0
- advanceRun (welfare.go) <- Stage 1
- estates.grant (estates.go) <- Stage 1
- occupation.Funded, occupation.Place (internal/occupation) <- Stage 1
- DefaultSpawner, Decisions, RegionView (spawner.go) <- Stage 1
- buildRegionView, the plan executor (phase_exit.go) <- Stage 1

Plan 2 and 5 are named beyond the contract's Needs column (7, 9) because the module needs a checkpoint section and the first-module harness; plan 8 because the golden and S37 to S41 need a stable pack.

Provides:
- Modules.Population, Population, Knot, RegionPopulation (config.go) -> plan-2-17
- PopulationScaled, ScaleKind (event.go) -> plan-2-17
- setPopulationScale (population.go) -> plan-2-17
- PopulationChanged, PopulationScaled (event.go) -> plan-2-18
- Snapshot.Population, RoleRow.BirthRateBP, RoleRow.DeathRateBP, RoleRow.NaturalIncreaseBP, RoleRow.BirthCarry, RegionRow.HouseholdCap (snapshot.go) -> plan-2-18
- PopulationChanged, CausePopulation, CauseBirths (event.go) -> plan-2-22
- overlay/population.toml, overlay/population-low.toml (testdata/chains/overlay/) -> plan-2-19
- overlay/population.toml (testdata/chains/overlay/) -> plan-2-17
- the stake rule entryMoney and RoleView.PopBirths, RegionAgent.PopDeaths (population.go, spawner.go)
- testdata/golden/population/ and overlay/population-low.toml -> plan-2-24
- the population region rows (Population.Regions, RegionPopulation) -> plan-2-16
- PopulationChanged.BirthsUnfunded (event.go) -> plan-2-19
- ledger.Deaths, ReasonDeaths (internal/ledger/registry.go, event.go) -> plan-2-22

The death carry is keyed by `AgentID`, so a relocation (plan 3, plan 14) leaves it unchanged (FR-POP-024; FR-MIG-032). INT's `AddAgent` needs no carry call (D423).

## Testing

(Testing.md 11.) Requirement to test, by the names in Phases 2 to 7:

| Requirements | Tests |
| --- | --- |
| FR-RES-044 (the call) | `TestWarnings_ResourcePlan` |
| FR-SCL-012 | the property in `TestStep_Carry` (events equal `floor((carry0 + sum)/D) - floor(carry0/D)`, expected count `w*p`; integer arithmetic and no draw: `TestPopulation_DrawsNothing`) |
| FR-SCL-014 (weight change by population) | `TestExec_Death`, `TestExec_Birth` (the household count equals the sum of weights, `scale.go`'s `checkCensus`) |
| FR-POP-001 | `TestPopulation_OffLeavesNoTrace`, `TestGolden_PopulationAlone`, the off tests unedited |
| FR-POP-002, 019 | `TestStep_OrderAndHook` |
| FR-POP-003, 040 | `TestStep_SignalIsOwnRatio` |
| FR-POP-004, 005 | `TestRate_Piecewise`, `TestRate_ScaleAndClamp`, `TestScale_EffectiveReadEachTick` |
| FR-POP-006, 007, 023 | `TestStep_Exposure` (the `Needs` half of 007 waits for SET) |
| FR-POP-008, 009, 010, 011, 012 | `TestStep_Carry`, `TestOffset_Range`, the worked-example test |
| FR-POP-013, 014, 015, 038 | `TestCut_FloorAndFed`, `TestAllow_Cap`, `TestStep_FloorCapAndExtinct` |
| FR-POP-016, 017, 018, 021, 022, 037 | `TestExec_Death`, `TestExec_Birth`, `TestPopulation_Conservation` |
| FR-POP-020 | `TestRenewal_ZeroWhileOn`, `TestWarnings_Population`, S40 |
| FR-POP-024 | `TestPopulation_Conservation` plus a relocation case in plan 14's tests |
| FR-POP-025, 026, 027 | `TestValidate_PopulationBounds`, `TestValidate_PopulationRegionRows` |
| FR-POP-028, 029, 030 | `TestWarnings_Population`, `TestReplacement`, `TestGainWarn` |
| FR-POP-031, 032, 033, 034 | `TestPopulationChanged_*`, `TestWarns_Population`, `TestSnapshot_Population` |
| FR-POP-035 | `TestCheckpoint_PopulationRoundTrip`, `TestFork_Population` |
| FR-POP-036, 039 | `TestPopulation_DrawsNothing`, `TestScale_EffectiveReadEachTick` |
| FR-POP-041 | not in this row: SET (Stage 3). `BirthsCeiling` is in `notBuiltFields` |
| S41 (RES-S1, RES 19.4) | `TestScenarioS41_ResourcePopulationLoop` |
| FR-OBS-021 | excluded: Traineeship events are SKL (Stage 3), the owner named in the `plan-2-0.md` deferred table for FR-OBS-021; the `PopulationChanged` position at P7.1(b) is asserted by `TestPositions_*` |
| NFR-POP-001, 002, 003, 004 | `TestPopulation_NoAllocation` and the benchmark, `TestObservation_DoesNotChangePopulation`, `TestPopulation_DrawsNothing` |

Goldens: `testdata/golden/off/` and the pinned hash unchanged (proof: the unedited off tests); `testdata/golden/population/` new; `all/` is not touched (it does not exist until the last module). Fork-at-N: `TestFork_Population`. Registry and catalogue rows: the four scope entries and two event entries deleted, one field entry added (Phase 3, 5). Fuzz: `FuzzRate`, `FuzzStep` in `internal/population`; knot-list seeds in `FuzzLoad`; a `FuzzValidatePopulation` over a decoded `Population` (never panics, a valid input has no error). Mutation: `gomu` on `internal/population`. `Guard:` tests: source guard of FR-POP-040; the PCG-state test; the proof the off files did not move. Each test has the doc comment and `Requirement:` line testing.md 4 and 5 ask for; no requirement ID in a name.

## Observability

(Observability 9.) 1. New failure paths: none returned; the module has no new error. The survived conditions have `Warn`s: "region extinct" (`region`), "births unfunded" (`region`, `births`), "deaths held" (`region`, `role`, `deaths`), pattern A, bounded by regions and roles, each with its counter in `PopulationChanged` (`Exposed` and `Unfed` for none; `BirthsUnfunded`; `DeathsHeld`) and region extinct with the region's households in the snapshot. 2. State transitions: a birth and a death are the executed operations' own events (`AgentSpawned`, `WeightChanged` with `Money` and `Units`, `AgentBankrupted` with `Cause` `population`), plus `PopulationChanged` (aggregate, per region and role, bounded by regions times roles, no map) after `EntryLimited`, and `PopulationScaled` (per changed key, emitted by INT's call). 3. Decision points: `Debug` "population step" (`region`, `births`, `deaths`) per region per tick, guarded by `Enabled`. 4. Loops: the step is a loop over a region's agents: no `Info`, no per-agent `Warn`, event construction only when an observer was supplied (observability 4.5), proved by the allocation test. 5. Changed event or metric: none; the metrics `household.births` and `household.deaths` already exist and `household.exits` leaves `population` out (FR-OBS-059). 6. New metric attribute: none. 7. `Info`/`Error`: none. 8. No error is both logged and returned. 9. Tests: Phase 5.

## Risks and open items

Open items (owner in brackets):
- **O1** (plan 2): plan 2 D247 writes `DeathCarry` always and leaves the value source to this plan; with the module off R10 wants `offset(ID)`, so `deathCarryOf` computes it from `ordering.Weyl32` and the calendar (E7). Confirm the name in Phase 1.
- **O2** (supervisor): a sub-step of a position and a build order that differs from the `Modules` row are not covered by `architecture.md` 2.10.2, 2.10.3, 2.10.5 (E6). Every later module built out of row order meets the second.
- **O3** (SPN, POP): whether the unfunded households of births count in `EntryLimited.Unfunded` and the "entry unfunded" `Warn` (D429, E9). If the spawner owner rules they do not, the executor passes a flag and the plan changes in `entryRegion` only.
- **O4** (POP with VIA): the margin of `no_fixed_factor` (`spec-population.md` 18 item 9) is not stated, so the warning here is the first-level test: no recipe `Draws` and no cap. N-109 (`r_full`) stays open; `GainBP` takes `eta` 1.
- **O5** (plans 7, 17, 19): S38 needs a yield shock and relief; INT's slot setters (plan 17) come after this row, so the test scripts the shock through the test seam of the yield slot that exists, or the leg is recorded as waiting for plan 17.
- **O6** (CFG): `Warning` has no value field, so `r*` is carried by the snapshot and the path (D434).
- **O7** (supervisor, plans 5 and 9): symbols named under Consumes that the plans have not been seen to define (`Declared` plumbing, the harness, the `[modules]` dispatch, `internal/curve`'s evaluator, `MaxKnots`, the checkpoint registration). If a plan names them otherwise only the Phase 1 grep list and the call sites change.
- **O8** (supervisor): N-132 stays a warning (`replacement_above_closed_ceiling` is read by acceptance as a rejection); the weakest reading of its note is that CFG may later make it an error.

Note proposals (text only; the supervisor is the editor): (a) N-290: "applied for POP in plan 11: `CausePopulation`, `CauseBirths`, `ReasonDeaths`, `ledger.Deaths` appended together; the executor accepts the cause with the module on"; (b) N-268: "POP fields `PopBirths`, `PopDeaths` and the effective `Renewal` 0 appended in plan 11"; (c) N-128 and N-244: "POP keeps the death carry in the module's slice by `AgentID`; the checkpoint writes it in the agent record (plan 11 D423); R10 of CHK needs O1"; (d) N-096: "births draw the stake from the pool inside `EntryPoolShare` after the module's deaths (built in plan 11); migrants take no stake"; (e) N-016: "rows owed: `Regions` of `Population` (E2)"; (f) N-017: "glossary terms of POP added to `doc.go` (E5)"; (g) a new note, CODE and POP: "a module built out of `Modules` row order appends its Go lines and keeps its hash id (E6)"; (i) N-097: "applied: the POP half is already in `spec-population.md` 6.5 (line 738) and table 7 (line 778), so no spec edit is owed; close at plan 11's gate"; (h) a new note, POP and SET: "`PopulationChanged.BirthsCeiling` and the `Needs` half of FR-POP-007 are tested when SET lands".

Dispositions: apply N-016 (E2), N-017 (E5), N-126 (rows exist in observability 4.3 and `spec-observability.md` 4.1, 7.3; the staged tests are edited), N-128 and N-244 (D423), N-217 (standing: the off files and the pin are the proof), N-268, N-290 (D425, D426). Leave open: N-027 (every default here is a placeholder, no stability claimed), N-103 (not this row's), N-163 (the code says "pool"), N-096 (update after the build), apply N-097 (the POP half is already in `spec-population.md` 6.5 and table 7; no spec edit is owed; closure proposed in (i)), N-109 (O4), N-132 (O8), N-153 and N-192 (SET and ORG: only the `Closed` half is built here).
