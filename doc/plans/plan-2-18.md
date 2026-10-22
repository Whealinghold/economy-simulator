# Plan: Stage 2, row 18: IND, `indicator/` and `internal/measure`

**Status: written (2026-10-22), not audited. Contract: [plan-2-0.md](plan-2-0.md).**
This plan builds the public package `indicator/` (derived measures as pure functions of snapshots and events: a tally, price tables, index numbers, the national-accounts-style presets, household-row statistics) and `internal/measure` (the checked 128-bit arithmetic under it). It is the measure step of "macro is measured, never set" (ADR 0001 decision 7). Nothing in the world reads an indicator; the plan adds no event, no configuration field, no state and no checkpoint section.

Rule tags read (all three files read in full; the files show the same tags as `plan-2-0.md`): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1.

Specs implemented: `spec-indicators.md` (IND) sections 3 to 10 and 12 to 15, the presets of 9.1 rows 1 to 21 (D560), and the requirements FR-IND-001 to FR-IND-055 (FR-IND-050 first clause only: `OK` false; second clause deferred, plan-2-0 deferred table, owner Stage 3 (the plan that adds the RoleRow income fields `IncomeLast`, `ReliefLast`, `CostLast`, N-130 item 1)) and NFR-IND-001 to NFR-IND-004 (all exist in section 20). Read for the seams: `spec-observability.md` (event table 4.1, snapshot 5.3, FR-OBS-040, 042), `spec-scale.md` (2.2, 5.1, FR-SCL-015), `spec-regions.md` (4.3 `Graph`, 6.4, 10.1), `spec-integration.md` (4.7.2 `AgentState`), `spec-checkpoint.md` (`Checkpoint`, `RestoreWorld`), `spec-experiments.md` 3.4.0 and `spec-stylised-facts.md` 11 (who uses the package).

Notes read (index, then the entries tagged IND or ALL; the file is not edited by this plan):
- N-130 (IND asks of OBS and others): items 1, 3, 4, 5, 6 left open and not adopted; item 2 (`WearApplied`) exists in `event.go` and is not read by a preset in version 1 (D574, D575).
- N-159 (fiscal and treasury reconciling item): leave open (FIS is Stage 3; the `Sectors` treasury and `FiscalIn`/`FiscalOut` fields are appended with FIS).
- N-178 (a small line's loss is a step function): leave open (the `Losses` preset is Stage 3, D560).
- N-201 (R9's volume clause is on units): apply. `Turnover.Volume` is units (`SumV`), tested as units in `TestTurnover_VolumeIsUnitsNotValue`.
- N-206 (the engine cannot call an indicator): apply. The import guard enforces it for every package (Phase 1).
- N-208 (gauge set and presets agree): leave open for PLC (Stage 3). A region set is already the `Regions` argument of `FlowSel`; `PriceLevelOf` is not adopted.
- ALL: N-016 apply (no `Config` field, no registry row), N-017 apply (terms in Phase 1), N-027 apply (no band, every example number is a placeholder), N-103 leave open (this plan edits one file), N-126 apply (no event, no `Warn`, no row), N-128 not applicable (no per-agent field), N-163 apply (only "pool" for the estate account; `M_t` is the treasury sector and is 0 until POL), N-217 apply (the pinned hash and `testdata/golden/off/` do not move).

Decision block: D560 to D579 (all used).

## Scope

**Builds.**
- `internal/measure/` (D562): `int128.go` (signed 128-bit sum and product), `div.go` (`RoundDiv`, the half-to-even quotient of two 128-bit values), `isqrt.go` (`IsqrtNearest`), `gini.go`, `rank.go` (`Quantile`, `ShareBelow`, `TopBottom`), `conv.go` (`ToInt64`, `ClampBP`). Primitive types only.
- `indicator/` (D561): `doc.go`, `errors.go`, `bp.go`, `series.go`, `time.go`, `classes.go`, `prices.go`, `tally.go`, `tally_observe.go`, `window.go`, `flows.go`, `accounts.go`, `value_added.go`, `sectors.go`, `index.go`, `levels.go`, `market.go`, `turnover.go`, `trade.go`, `people.go`, `rows.go`, `distribution.go`, `income.go`.
- Public identifiers (provisional until the code, spec 3.6): `Window`, `Tally`, `NewTally`, `Prices`, `PricesOfWindow`, `PricesOfSnapshot`, `PricesOfReference`, `Base`, `BaseOf`, `Classes`, `NewClasses`, `FlowSel`, `Household`, `RowsOf`, `Stat`, `StatOp`, `Point`, `Stats`, `Growth`, `Change`, `YearWindow`, `SeasonWindows`, `BP`, `Quotient`, `Annualise`, the warning constants, the six sentinels of 3.5 and the 21 presets `GDP`, `ValueAdded`, `PriceIndex`, `BasketLevel`, `DeliveredPrice`, `PriceGap`, `Sectors`, `Turnover`, `Households`, `WelfareDistribution`, `Inequality`, `Unfed`, `TradeBalance`, `RealWage`, `StockCover`, `RoleShares`, `PriceRange`, `Pinned`, `Dispersion`, `Offset`, `HouseholdIncome`.
- Test code and fixtures: `indicator/*_test.go` (external package `indicator_test`), `internal/measure/*_test.go`, the extension of `import_guard_test.go`, `testdata/golden/indicator/` (D577).

**Leaves out, with the owner.**

| Item | Owner |
| --- | --- |
| Presets 22 to 25 (`Unemployment`, `WageBill`, `Employment`, `Losses`) and FR-IND-056 to 059 | Stage 3 plans of EMP (`LabourRound`), RSK and STO (`StoreLoss`, `Lost`, reason `loss`); D560, owed edit 3 |
| The treasury term of `Sectors`, `FiscalIn`/`FiscalOut`, the `Loss` term of `Written`, `Jurisdictions` | the Stage 3 plans of POL, FIS, RSK; each appends its case to `Observe` and its field to the result |
| The income fields of `RoleRow` and so `HouseholdIncome` with a value | D574: deferred, owner Stage 3 (the plan that adds the RoleRow income fields `IncomeLast`, `ReliefLast`, `CostLast`, N-130 item 1); this plan returns `OK` false |
| The accrual form of depreciation (`WearApplied`), a bulk row read, goods lines on `AgentRelocated`, per-line carrying units, wear on `RecipeExecuted` | not adopted for version 1 (D575); N-130 stays open |
| Any use of an indicator by the world, a policy or a damper | forbidden (spec 3.1); PLC's `Policy` is a consumer (N-206) |
| Bands and thresholds on a preset | STY, plan 19 |
| Series storage, sweeps, reductions | the consumer, EXP (plan 20); `internal/reduce` does not import `internal/measure` |
| An `otelobserver` gauge from a preset | plan 22 / OBS 18 question 8 |

## Decisions

- **D560** Version 1 builds rows 1 to 21 of spec 9.1 (row 21, `HouseholdIncome`, returns `OK` false, D574). Rows 22 to 25 need events and a reason that do not exist before Stage 3 (`LabourRound` EMP, `StoreLoss` and the `Lost` fields and reason `loss` RSK and STO; grep `event.go`: none of them is declared). A preset is built with the module that makes its input, so FR-IND-056 to FR-IND-059 stay open until those plans, and each such plan adds its tally rows, its preset and its tests in the same change. The accounts' identities hold in Stage 2 because the reason `loss` is never booked. Source: spec 9 intro ("the four rows 22 to 25 are the presets of the employment and risk modules"), `roadmap-v1.md` Stage 3, architecture.md 2.10.6 (no module file before its slice runs).
- **D561** One file per concern, cut to the Lizard limits of N-243 (`-C 10 -a 8 -L 100`), so `Observe` is a dispatch of one switch over per-event fold methods (`tally_observe.go`) and each preset is a short function over the aggregators (spec 3.6, 9.2). Names do not repeat the package (spec 3.2.4): `indicator.Window`, not `indicator.IndicatorWindow`. Source: spec 3.2, 23; architecture.md 2.4, 6.1; N-243.
- **D562** `internal/measure` imports the standard library (`math/bits`) and `internal/checked`, and is the only package that does: `indicator` calls `measure` only. `measure` wraps `checked.Acc` and `DivHalfEven` for the one sum that is unsigned (`PricesOfWindow`: `sum(Price * Volume)`; the `SumPV` field of a window row is a `measure` type) and uses `checked.MulDivHalfEven` where the operands fit `int64`. It adds only what `checked` lacks: a signed `Int128` (flows and changes of stock are signed), a 128-by-128 half-to-even quotient (the Gini divisor `W * S` is 128-bit), the nearest integer square root, the Gini accumulation and nearest rank. No second copy of an existing rule: `TestRoundDiv_AgreesWithChecked` runs both on the shared domain. Spec 3.2.1, 5.2.1 and NFR-IND-004 say "standard library only"; spec 5.2.1 itself says to use the shared helper, so the three sentences are corrected (owed edit 1). Source: spec 5.2.1, N-019, architecture.md 2.6 (`checked/` is the one 128-bit helper), N-019 update of 2026-10-11 ("the first code decides whether a shared `internal/checked` is worth it": it exists).
- **D563** A result is plain data: exported fields, no map, no tag, slices in canonical order, `ConfigHash` and the `Window` it covers, an `OK bool` per field that can be undefined (a zero divisor is `OK` false, never a panic), `Clamped` where it is an index, a ratio or a growth, and `Warnings []Warning` in ascending order from the closed set `Stale`, `Clamped`, `Thin`, `Cohort`, `Absent`, `NoRoute`, `Basis`, `Binding`. A value with `OK` false is zero. Source: spec 5.1, 9.1.2, FR-IND-012, 023.
- **D564** `Tally` is a value with an `Observe(Event)` method (it satisfies `Observer`); it keeps running sums in slices sorted by key and found by binary search, pre-sized at `Open` from the snapshot's `Markets` and `Roles` rows, so `Observe` of an event of an existing row allocates nothing (NFR-IND-002) and its size is a function of the configuration, not of agents, orders, trades or ticks (FR-IND-004). It ignores every event while no window is open, every event type it does not use, and the volume classes O and R (`OrderPlaced`, `OrderRejected`, `TradeExecuted`). The `Observe` switch ignores `ShipmentRejected`, `MigrationStarted`, `OwnUseChanged`, `PopulationScaled`, `OutsideEnded` and the other events no preset reads. The set of cases is the set of event types that exist when the code is written: those of Stage 1 plus `AgentRelocated` (plan 3), `MoneyFlowLimited` (6), `ReliefPaid` (7), `PopulationChanged` (11), `ShipmentDispatched`, `ShipmentArrived` (12), `MigrationReviewed` (14), `ContractFormed`, `ContractSettled`, `ContractRound` (15), `OutsideTraded` (16), `WorldRestored` (2). A Stage 3 event adds one case. Source: spec 4.2, FR-IND-004, 005, NFR-IND-002.
- **D565** `Close(s)` returns a `Window` that is a deep copy of the running sums (the tally may be reopened at once), with `From`, `To`, `ConfigHash`, the step count, the household counts by (region, role) at both snapshots, `OpenPending`, `ClosePending` and the rows of spec 4.2.2 in canonical order. The flows of a window are the rows less `OpenPending` plus `ClosePending` (spec 7.1.5) and the construction bucket (`Tick` -1) is a separate memo row (FR-IND-007). `Close` returns `ErrWindowOrder` for a `TickEnded` count other than `Close.Tick - Open.Tick`, a failed step, or an event outside the window other than the bucket, and `ErrConfigMismatch` for snapshots of different `ConfigHash`. The window of a tally started by `WorldRestored` opens at that event's `Snapshot` (spec 10.5.2, FR-IND-055). Source: spec 4.1, 4.2.5, 4.2.6, 7.1.4, 7.1.5, FR-IND-006, 007, 055.
- **D566** A preset that needs stocks takes the two snapshots as arguments beside the `Window`: `GDP(w Window, open, close Snapshot, p Prices)`. It checks `open.Tick == w.From`, `close.Tick == w.To` and both `ConfigHash` against the window's, and returns `ErrWindowOrder` or `ErrConfigMismatch`. `Snapshot` is a deep copy (architecture.md 12.4), so no live reference crosses the boundary and a snapshot may be marshalled on another goroutine. Spec 4.2.6 gives `Window` no stocks, so keeping the snapshots with the caller avoids copying every market row into every window. Source: spec 3.3, 4.2.6, 9.1.3.
- **D567** A price is always an argument (spec 4.3). `Prices` is a table per (region, commodity) with `At(r, c)` and the `ConfigHash` and window of its origin; a mismatch is `ErrConfigMismatch`. `PricesOfWindow(w)` is `rdiv(SumPV, SumV)` through `checked.Acc` with the last published price carried forward where `SumV` is 0 (FR-IND-028); `PricesOfSnapshot(s)` is `Markets[].State.Price`; `PricesOfReference(cfg)` is `Commodities[].ReferencePrice` with the region's `RegionPrices` row (plan 3 D265, D266). An outside node has no row. `Base` freezes prices and quantities of a closed window (`BaseOf(w, p)`). Source: spec 4.3, 3.3, FR-IND-028.
- **D568** `Classes` is built from `Config` (`NewClasses(cfg)`; the field is `Commodity.Classes`, `config.go:264`, canonical order), not from `Names`, which carries no class. `Classes.Of(name)` returns the sorted commodity set or `ErrUnknownClass`; a commodity in two classes counts once in a whole-economy measure and in each class in a class measure. No code names a class: every preset takes the class or the set as an argument. Source: spec 4.4, FR-IND-029, FR-GDS-017.
- **D569** The calendar is a value argument: `Config().Calendar` (type `economysim.Calendar`, `calendar.go`: `TicksPerYear`, `At`). Spec 3.3 names `World.Calendar()`, which does not exist in the code; owed edit 2 corrects the spec rather than adding a method to the world. `YearWindow` and `SeasonWindows` are binary searches over the monotone `At`. Source: spec 10.2, `calendar.go:134,148`, `world.go:262`.
- **D570** Household rows are `Household{Weight, Total int64}`. `RowsOf(states []AgentState, col Column, opt RowOpt)` takes the states a consumer already holds, never a world (FR-IND-002), so the dependency on INT is the type `AgentState` and its fields (`Weight`, `External`, `Money`, the welfare and earned ratios and whether they exist, `Hunger`, holdings), not the methods `Agents`/`Agent`. A row whose value does not exist is left out and counted in `Absent`. `Wealth` takes a `Prices` table and values holdings at the region's prices with checked arithmetic. Source: spec 4.5, 6.1, 6.5, INT 4.7.2, FR-IND-026, 027.
- **D571** Sums are `Int128`; a sum later multiplied by 10 000 is checked against `2^127 / 10 000` first (`ErrOverflow` beyond); a quotient is rounded half to even at the last division only (the Fisher mean and a chain link are the two named exceptions); an index, ratio or growth at or above `MaxIndexBP = 2^31 - 1` is clamped and flagged, not an error. `indicator` and `internal/measure` import no float type and not `math` (FR-IND-008). Sorts are total orders that end in the row position, no map is iterated, and the result is independent of row order (FR-IND-010). Source: spec 5.2 to 5.7, architecture.md 3.5.
- **D572** The Gini, `Top`, `Bottom` and quantiles are the exact 128-bit rules of spec 5.4 and 5.5 over rows sorted by cross multiplication (`T_i * w_j < T_j * w_i`, ties by row position); the Gini is the population coefficient (no `n / (n - 1)`); a negative value is `ErrNegativeValue`. Every result over rows carries `Households`, `Agents`, `Absent`, `Cohort` and `CohortShare`, and a world of weight above 1 is never an error (spec 6.2). Source: spec 5.4, 5.5, 6.2, FR-IND-014, 015, 024 to 026.
- **D573** The tests of the accounts are written first against the worked example of spec 18, built as `Window` and `Snapshot` literals in one shared fixture (`indicator/worked_test.go`), with the figures that `doc/specs/models/ind/worked.py` asserts (checked by reading `worked.out`, not by running Python on the host). The run-level identities (FR-IND-016 to 019) are then asserted over real runs in Phase 8. Source: spec 17, 18; testing.md 3, 6.3.
- **D574** `HouseholdIncome` returns `OK` false in version 1 (FR-IND-050): the fields it needs (`IncomeLast`, `ReliefLast`, `CostLast` on `RoleRow`, N-130 item 1) are not in `snapshot.go` (grep: none). No Stage 2 plan adds them (plan 8 measures realised income in test helpers and `CensusRow.IncomeBP` only); deferred to Stage 3, the plan that adds the RoleRow income fields. Phase 7 ships the `OK` false result and its test. Source: spec 7.5.1, 9 row 21, 21 question 2, N-130 item 1, N-054.
- **D575** The other asks of N-130: (2) `WearApplied` exists, the tally ignores it, depreciation is the retirement basis with `Basis` in the result and `NetProduct = GDP - sum p * Wear` (spec 7.6.2); (3) no bulk read is asked, but Phase 8 measures a `RowsOf` pass at the first consumer's size (spec 21 question 1); (4) `AgentRelocated` keeps total `Units`, so the regional goods identity holds in total and per commodity up to the reported `RelocatedUnits` residual (spec 8.2); (5) freight per unit of cargo is the stated convention, exact for a one-line shipment (spec 8.4, question 8); (6) `ValueAdded` by recipe overstates a tool-using recipe by its wear and says so in its `Must not hide` test. Source: spec 19, 21, 22.
- **D576** Fact tests of STY and the equivalence test of EXP live in the external test package of `indicator` (spec-stylised-facts 11, spec-experiments 3.4.0, FR-IND-001). Plan 19 writes its own recording observer in `stylised_helpers_test.go` (its D594); this plan keeps its run helper private to `indicator/run_test.go` and provides none. The flag `-update` and `raceEnabled` are mirrored once in `indicator/race_test.go` and `indicator/norace_test.go` (the functions of plan 1 are not importable from `indicator_test`) and plan 19 reuses them (its O5). Source: spec 11.1, 11.2; `config.LoadFS` (`config/config.go:100`).
- **D577** One golden, `testdata/golden/indicator/one.txt`: the rendering of presets 1 to 21 for a window of pack `one` warm (seed 1, the length the golden of Stage 1 uses), written by a small renderer in `indicator/render_test.go` (integers only, `-update` refused under `-race`, the rule of plan 1 D228 mirrored in `indicator/race_test.go`). It is not under `golden/off/` and records no world state, so `golden/off/` and the pinned hash do not move (N-217). It is recorded on the pack as plan 8 leaves it; a later change of pack data moves it with the explanation of testing.md 7.2. Source: spec 23, testing.md 7.1, 7.2.
- **D578** The guards: the import-graph test (FR-IND-001, NFR-IND-004) extends `import_guard_test.go`; the signature test (FR-IND-002) and the no-float/no-`math`/no-global-state/no-goroutine test (FR-IND-008, 054) are AST scans of the non-test files of the two packages (`go/parser`; shortcut: a type alias that hides `World` is not followed, add `go/types` if one appears). The non-interference test (FR-IND-003) compares `World.Checkpoint().MarshalBinary()`, the event stream and the snapshots of two runs of one seed, one with a consumer that computes every preset after every step. Source: spec 3.1.2; testing.md 6.4.
- **D579** Mutation testing runs on `internal/measure` only, in Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4` and `gomu run . --workers 4`, one file at a time (the memory file `gomu-docker-workflow.md`); `indicator/` is covered by the golden, the worked-example tests and the identities. Source: plan-2-0 gates, `~/.claude/CLAUDE.md`, testing.md 9.

## Phases

### Phase 1: owed edits, glossary and the guards (no indicator code)

Goal: the specs say what the code will do (D562, D569, D560) and the import graph is enforced before any package exists. Files: the specs below (the supervisor applies them), `import_guard_test.go`.

| Owed edit | Where | Change |
| --- | --- | --- |
| 1 | `spec-indicators.md` 3.2.1, 5.2.1, NFR-IND-004 | "`internal/measure` imports the standard library and `internal/checked`" (D562); 5.2.1's open question 11 is answered: the one home is `internal/checked` |
| 2 | `spec-indicators.md` 3.3 (calendar row), 4.5.2 | the calendar is `World.Config().Calendar`; `RowsOf` takes `[]AgentState` (D569, D570) |
| 3 | `spec-indicators.md` 9, 20; `roadmap-v1.md` Stage 2 row IND and the Stage 3 rows EMP, STO, RSK | rows 22 to 25 and FR-IND-056 to 059 are built with EMP and RSK/STO; the roadmap rows name them (D560) |
| 4 | `doc.go` glossary | the terms of spec 19 (indicator, indicator preset, aggregator, tally, indicator window (a span of ticks), household row, price table, base window, index, Laspeyres, Paasche, Fisher, chain index, clamp, stale share, cohort share, value added (of a region), production approach, expenditure approach, income view, gross accumulation, written (goods), carriage, delivered price, freight, trade valuation, retirement basis, binding counters, reconciling item) before Phase 3 (architecture.md 3.2) |
| 5 | `doc/notes.md` | N-130 update (items 1, 3 to 6 left open, 2 read, D574, D575); N-206 update (enforced by the guard of Phase 1); new note, see Risks |
| 6 | `doc/documentation_index.md`, `doc/specs/README.md` IND row | the plan's row removed after Stage 2; the IND row says rows 22 to 25 are Stage 3 |
| 7 | none | `spec-indicators.md` 9 row 20 already gives `Unmet` as the share of household-ticks that were unfed, so plan 19's item 7 is met; `TestPriceRange_Pinned_Dispersion_Offset` asserts the unit |
| 8 | `rules/testing.md` 7.1.1, `rules/architecture.md` 2.6 (`golden/` text), `AGENTS.md` Layout (`testdata/golden/`) | a golden outside the three sets (`off/`, one directory per module, `all/`) is allowed for a harness or renderer that records no world state: `golden/indicator/`, `golden/experiment/`, `golden/stylised/`; fresh RULE-TAG on the two rule files; a second edit to 7.1.1 after plan 5 E10 (which adds the module-alone exception), each with its own fresh tags. This plan is the first to write such a golden (D577) |

Tests added (in `import_guard_test.go`, `TestImportGuard_Detects` table, `Requirement: FR-IND-001, NFR-IND-004`): `config/`, `strategy/` and `experiment/` importing `indicator` is a problem; `experiment/` importing `internal/measure` is a problem; any `internal/x` other than `measure` importing `internal/measure` is a problem; `indicator/` importing `config`, `strategy`, `experiment`, any `internal/` other than `measure`, or a third-party path is a problem; `internal/measure` importing anything but the standard library and `internal/checked` is a problem; and the real tree passes `TestImports_StdlibAndLayers` as it does today.
Exit check: `go test -short -race -run 'TestImport' .` passes; owed edits 1 to 4 and 8 applied.

### Phase 2: `internal/measure`

Goal: the arithmetic of spec 5 over primitive types, each rule with a hand-computed case from spec 18 and 5.2.2. Files: the six of Scope, plus tests.

Tests (`Requirement:` in brackets):
- `TestInt128_ArithmeticLaws` (property: add/sub/mul against `bits`-built references and identities; commutative, associative) [FR-IND-010].
- `TestRoundDiv_HalfEvenBothSigns` (rdiv(-5,2) = -2, rdiv(-3,2) = -2, rdiv(25,10) = 2, rdiv(35,10) = 4; zero divisor reports not ok, never panics) [FR-IND-012].
- `TestRoundDiv_AgreesWithChecked` (the shared domain against `checked.MulDivHalfEven` and `checked.Acc.DivHalfEven`: one rule, not two, D562).
- `TestIsqrtNearest_NoTie` (20 000 pseudo-random products below 2^62, result is the nearest integer, no tie; `146 264 715 -> 12 094`) [FR-IND-021].
- `TestGini_PairwiseDefinition` (2 000 random row sets, the exact integer identity `2 * W * S * G_num = sum_i sum_j |T_i * w_j - T_j * w_i|` of spec 5.4.1, so no float and no `math/big`), `TestGini_RowOfWeightEqualsRepeatedRows`, `TestGini_EqualAndOneHolder` (0 and `(W-1)/W`, 5 000 bp for two households), `TestGini_UndefinedAndNegative` (`W = 0` or `S = 0` not ok; a negative value is the sentinel) [FR-IND-014, FR-IND-015].
- `TestQuantile_NearestRank` (the 41-over-2 example returns 20; `Q(0)` minimum, `Q(10 000)` maximum; ties in either order), `TestShareBelow_Strict`, `TestTopBottom_SplitsTheBoundaryRow` [FR-IND-024, FR-IND-025].
- `TestToInt64_Overflow`, `TestGuardBeforeTimes10000` (a sum at `2^127 / 10 000` is refused), `TestClampBP` (at `2^31 - 1`) [FR-IND-009, FR-IND-023].
- `FuzzGini`, `FuzzRank`, `FuzzRoundDiv` (never panic; the properties above hold) [NFR-IND-003].
- `TestMeasure_NoFloatNoMath` (AST scan, D578) [FR-IND-008, NFR-IND-004].

Exit check: `go test -short -race ./internal/measure/` passes; coverage of the package at least 90%; `lizard -l go -C 10 -a 8 -L 100 -w internal/measure/*.go` clean; the first gomu pass (D579) has no surviving mutant that is not an equivalent one, each equivalent one named in the PR.

### Phase 3: foundations of `indicator/`

Goal: the types and functions that need no tally. Files: `doc.go`, `errors.go`, `bp.go`, `series.go`, `time.go`, `classes.go`, `prices.go` (the table, `PricesOfSnapshot`, `PricesOfReference`), `rows.go` (`Household`, `Stat`, `StatOp`).

Tests:
- `TestErrors_Sentinels` (each of the six sentinels wrapped with the preset's name and the field, `errors.Is` true) [spec 3.5].
- `TestQuotient_UndefinedIsNotAnError`, `TestAnnualise_ExactFromSums` (`rdiv(1 150 * 365, 2) = 209 875`) [FR-IND-012, FR-IND-053].
- `TestStats_Growth_Change` (spec 10.3.2; `OK` false for `Low <= 0` and a base at or below 0; growth clamp) [FR-IND-023].
- `TestYearWindow_PartialFirstYear`, `TestSeasonWindows_WrapsTheYear` (a wrapping season belongs to the year in which it starts; clipped runs flagged) [FR-IND-052].
- `TestClasses_UnknownAndOverlap` (`ErrUnknownClass`; a commodity in two classes is counted once by `All()`) [FR-IND-029].
- `TestPrices_ConfigMismatch`, `TestPricesOfReference_RegionOverride`, `TestPricesOfSnapshot_IsTheMarketPrice` [spec 4.3.3, 4.3.4, FR-IND-028 (snapshot part)].
- `TestStat_RowOrderDoesNotMatter` (shuffled rows, every `StatOp`) [FR-IND-010].
- `TestSignature_NoWorld` (AST scan of the exported identifiers) [FR-IND-002]; `TestIndicator_NoGlobalStateNoGoroutineNoFloat` [FR-IND-008, FR-IND-054].

Exit check: package tests pass under `-short -race`; owed edit 4 is applied.

### Phase 4: the tally and the window

Goal: `Tally`, `Window`, `Open`, `Close`, `Observe`, the pending correction and `PricesOfWindow`/`BaseOf`. Files: `tally.go`, `tally_observe.go`, `window.go`, `flows.go`, the window part of `prices.go`.

Tests:
- `TestTally_IgnoresWhatItDoesNotUse` (order and trade events, and every event while no window is open) [FR-IND-005].
- `TestTally_SizeIsAFunctionOfTheConfiguration` (the same configuration at 120 and 2 400 agents gives equal row counts) [FR-IND-004].
- `TestTally_ObserveAllocatesNothing` (`testing.AllocsPerRun`, an event of an existing row) [NFR-IND-002].
- `TestClose_WindowOrder` (missing `TickEnded`, a failed step, an event before the window, `Open.Tick >= Close.Tick`) and `TestClose_ConfigMismatch` [FR-IND-006].
- `TestTally_ConstructionBucketIsAMemoLine` (`Tick` -1 apart; `initial` at tick 0 or later is in the flows as `Endow`) [FR-IND-007].
- `TestTally_RunningHouseholdsEqualTheSnapshot` (every boundary of a 60-tick run on `one`, with births, bankruptcies, a weight change and a role change; and with a relocation once plan 3 is in) [FR-IND-011].
- `TestFlows_PendingCorrection` (a call between ticks before `Open` and one after `Close`; the corrected flows equal the ledger's) [spec 7.1.5, FR-IND-016 input].
- `TestPricesOfWindow_VolumeWeighted` (the 18.1 example: `rdiv(10*30 + 11*20, 50) = 10`; carried-forward price where no unit traded; the `ConfigHash` of the window) [FR-IND-028].
- `TestBaseOf_FreezesPricesAndQuantities` [spec 4.3.5, 10.4].
- `TestTally_RestoredWindow` and `TestTally_OpensAtWorldRestored` (a window straddling a restore with the same tally equals the uninterrupted run; a fresh tally opens at the event's snapshot) [FR-IND-055] (needs plan 2).

Exit check: tests pass; no `Observe` case names a Stage 3 event; coverage of the new files at least 90%.

### Phase 5: accounts

Goal: the production, expenditure and income-free views and their identities. Files: `accounts.go`, `value_added.go`, `sectors.go`, `trade.go`, `FlowSel`/`Value` in `flows.go`.

Tests (each on the worked fixture of spec 18; D573):
- `TestFlowSelValue_ClassAndRegion` (nil slice means all; a class turned into commodities) [spec 3.6].
- `TestGDP_ThreeViewsAgree` (18.2: alpha 1 200, beta 890, realm 2 090; `Residual` 0 for each region and, at one table, for the realm; `TradeValuation` -27 at regional prices) [FR-IND-017, FR-IND-030].
- `TestGDP_PendingShownApartFromCash` (accrual 1 200 against cash 1 300), `TestGDP_BasisIsRetirement`, `TestGDP_DiscardsAreWrittenNotConsumed` [FR-IND-030].
- `TestValueAdded_ByRecipeAndWIP` (18.3: 1 300 total, `WIP` +100 by the formula of 7.2.4; a class's value added is not defined), `TestValueAdded_ToolRecipeOverstates` (D575 item 6) [FR-IND-031].
- `TestSectors_MoneyByHolderAndNetWorth` (18.6: 5 000, 800, 200, 0; per household 50; the boundary balance by reason; the closing of the identity of 7.5.3) [FR-IND-018, FR-IND-036].
- `TestTradeBalance_GrossAndNet` (`Imp = Offer - Unsold`, `Exp = Purch`, value at a table, openness, price improvement is a transfer) [FR-IND-042].
- `TestRegionalMoneyIdentity_Terms` (relocation, contract settlement and faucets and sinks close exactly on a literal; the fiscal terms are absent until FIS) [FR-IND-019].
- `TestBinding_WarningCarriesTheCounters` (a regional comparison with a non-zero cap counter carries `Binding`) [FR-IND-051].

Exit check: tests pass; `lizard` clean on the new files.

### Phase 6: prices and markets

Goal: index numbers and the market-figure presets. Files: `index.go`, `levels.go`, `market.go`, `turnover.go`.

Tests:
- `TestPriceIndex_Food` (18.4: Laspeyres 12 105, Paasche 12 083, Fisher 12 094; 10 000 against its own base; zero denominator `OK` false; class and all-goods are separate calls) [FR-IND-020, FR-IND-021, FR-IND-032].
- `TestPriceIndex_ChainFollowsTheRule` (`C_k = rdiv(C_{k-1} * I_k, 10 000)`; a clamped link makes every later point `Clamped`) [FR-IND-022].
- `TestPriceIndex_StaleShareAndClamp` (carried-forward cells reported; index at `2^31 - 1` clamped, not an error) [FR-IND-023].
- `TestRealAndDeflator_Identity` (`|N - rdiv(R * deflator, 10 000)| <= ceil(R / 20 000)`) [FR-IND-013].
- `TestBasketLevel_ExactlyTenThousandAtReference` (18.5: 11 250, 10 800, ratio 9 600) [FR-IND-033].
- `TestDeliveredPrice_HiredAndSelfCarried` (freight 2, delivered 12; a self-carried lane `Hired` false, `Freight` 0) [FR-IND-034].
- `TestPriceGap_SpreadFromExactSums` (2 609 not 2 500; `Connected` false excludes a pair from an average) [FR-IND-035].
- `TestTurnover_VelocityAndActive` (18.6: 5 000 and 912 500), `TestTurnover_VolumeIsUnitsNotValue` (N-201) [FR-IND-037].
- `TestStockCover_UndefinedWhereNothingIsConsumed` (434 000; unsold 1 705) [FR-IND-044].
- `TestPriceRange_Pinned_Dispersion_Offset` (11 000; 1 400; 3 000; 1 111 and `Unmet`) [FR-IND-046, 047, 048, 049].

Exit check: tests pass; every preset of the phase has its `Must not hide` clause asserted as a separate check.

### Phase 7: people

Goal: household counts, welfare and inequality presets. Files: `people.go`, `rows.go` (`RowsOf`), `distribution.go`, `income.go`.

Tests:
- `TestHouseholds_AccountOfChangeCloses` (18.7: `+2`, `Residual` 0; agents and households separate; a migrant is in its origin) [FR-IND-038].
- `TestUnfed_DenominatorIsTheRunningCount` (1 220, not 1 235; `UpkeepShortfall` absent when all are fed; an agent of weight 6 unfed counts 6) [FR-IND-041].
- `TestRoleShares_ARoleAtZeroIsReported` (swing over a series) [FR-IND-045].
- `TestRealWage_AbsentWithoutAWage` [FR-IND-043].
- `TestRowsOf_ExcludesPseudoAgentsAndAbsent` (an estate is never a row; `External` dropped only when asked; an agent with no upkeep counted in `Absent`) [FR-IND-026, FR-IND-027].
- `TestWelfareDistribution_NoGini` (18.8; negative ratios allowed; `Absent`, `Cohort`; the mean differs from the aggregate ratio by the covariance, 18.9) [FR-IND-039].
- `TestInequality_CohortShare` (the same world at scale 1 and at `overlay/h200.toml` scale 4: `CohortShare` 0 against above 0; the Gini of money and `Top`/`Bottom`/`P90P10`) [FR-IND-040, FR-IND-026].
- `TestHouseholdIncome_NotAvailableInVersion1` (`OK` false) [FR-IND-050].
- `TestHouseholdIncome_BookedFromRoleRowFields` (skipped stub naming N-130 item 1; the second clause of FR-IND-050 for the day the fields exist).

Exit check: tests pass; `HouseholdIncome` returns `OK` false.

### Phase 8: run-level tests, golden, fuzz and costs

Goal: the properties that need a real world, over real packs and seeds. Files: `indicator/run_test.go`, `indicator/race_test.go`, `indicator/norace_test.go` (D576), `indicator/render_test.go`, `indicator/bench_test.go`, `testdata/golden/indicator/one.txt`, `indicator/testdata/fuzz/`.

Tests:
- `TestRun_UnitsConserved` (`dHeld + dTransit` equals the flows of 7.1.3 for every window, commodity, seed 1 to 4, packs `one` and `second`, with and without a call before `Open` and one after `Close`) [FR-IND-016].
- `TestRun_GDPIdentityHolds` (production equals expenditure at one table, every window) [FR-IND-017]; `TestRun_NetWorthIdentity` [FR-IND-018]; `TestRun_RegionalMoneyIdentity` (a relocation in the window; `RelocatedUnits` equals the `Units` of `AgentRelocated` in total) [FR-IND-019].
- `TestRun_NonInterference` (same seed and calls, one run computing every preset after every step, one with no consumer: equal checkpoint bytes, event stream and snapshots) [FR-IND-003].
- `TestRun_CostIndependentOfAgents` (allocations of the class 1 and 2 presets equal at two scales of one configuration) [NFR-IND-001].
- `TestGolden_IndicatorPresetsOnOne` (D577) [spec 23].
- `FuzzIndexFunctions`, `FuzzRowStatistics`, `FuzzAccounts` (never panic; the properties of FR-IND-010, 014, 015, 020, 024 hold for any input) [NFR-IND-003].
- `BenchmarkRowsOf` at the first consumer's size (5 000 agents) and `BenchmarkTallyObserve` (reported, not asserted; testing.md 8.2) [spec 21 question 1, NFR-IND-002].

Exit check: the tests pass under `-race` (reduced seeds as the mirrored `raceEnabled` sets) and without it (all seeds); `testdata/golden/off/` and the pinned hash did not move.

### Phase 9: exit gate

Goal: the gates of plan-2-0. Run (read-only, then the gate commands): `gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; `go mod tidy` no diff; `lizard -l go -C 10 -a 8 -L 100 -w indicator/*.go internal/measure/*.go`; coverage at least 90% for `indicator/` and `internal/measure` from the R run, and F also passes, R and F exactly as plan-2-1 D229 states them (until plan 1's Phase 2 is done, the fallback of plan-2-0: `go test -race -covermode=atomic -coverpkg=./... ./indicator/ ./internal/measure/`, stated in the PR); gomu on `internal/measure` (D579); markdownlint on the owed spec edits. No row is deleted from `notBuilt` (no configuration field).
Exit check: all of the above pass; the PR carries the Testing and Observability entries below; the supervisor has applied owed edits 1 to 6 and 8.

## Catalogue appends

| Catalogue file | Appends | Position | Hash id / tag | Spec line |
| --- | --- | --- | --- | --- |
| `config.go`, `bounds.go`, `defaults.go`, `validate.go` | none | n/a | n/a | spec 12 (no field, no default, no row) |
| `hash.go` | none | n/a | no field | spec 12 |
| `errors.go` (root) | none. The six sentinels of 3.5 are `indicator`'s own (`indicator/errors.go`), as `ErrUnknownClass` is avoided by the root (N-075 note of 2026-10-13) | n/a | n/a | spec 3.5 |
| `event.go`, `option.go`, `tick.go`, `snapshot.go`, `ledger.go` | none: no event, no option, no tick position, no snapshot field, no reason | n/a | n/a | spec 14, 15 |
| checkpoint tags | none: a tally is the consumer's value, not world state | n/a | n/a | spec 15 |
| log keys | none | n/a | n/a | spec 14.1 |
| metrics | none | n/a | n/a | spec 14.2 |
| `import_guard_test.go` | the Phase 1 rows (test code, not a catalogue file) | after the existing `indicator parser` row | n/a | FR-IND-001, NFR-IND-004 |
| `testdata/golden/indicator/` | `one.txt` | new directory beside `off/`, allowed by owed edit 8 | n/a | spec 23 |

## Cross-plan contract

Consumes:
- WorldRestored (event.go) <- plan-2-2
- World.Checkpoint, Checkpoint.MarshalBinary and RestoreWorld (checkpoint.go) <- plan-2-2
- AgentRelocated (event.go) <- plan-2-3
- World.Relocate (integration.go) <- plan-2-17
- Graph and Graph.Component (regions.go) <- plan-2-3
- RegionPrices (config.go) <- plan-2-3
- World.ReferencePrice (regions.go) <- plan-2-3
- the scale overlay h200.toml (testdata/chains/overlay/h200.toml) <- Stage 1
- MoneyFlowLimited (event.go) <- plan-2-6
- Needs plan 4 for the weight contract (SCL 2) only: nothing is imported; the weighted-household tests build `AgentState` literals with `Weight` above 1
- ReliefPaid (event.go) <- plan-2-7
- the rebalanced pack data of `one` and `warm-one` (testdata/chains/) <- plan-2-8
- PopulationChanged (event.go) <- plan-2-11
- ShipmentDispatched, ShipmentArrived and `ReasonTransitOut`, `ReasonTransitIn`, `ReasonHaulUse`, `ReasonTransitUse` (event.go) <- plan-2-12
- MigrationReviewed (event.go) <- plan-2-14
- ContractFormed, ContractSettled and ContractRound (event.go) <- plan-2-15
- OutsideTraded and the outside reasons (event.go) <- plan-2-16
- AgentState (agent.go), World.AddAgent (integration.go) <- plan-2-17
- Snapshot.Flows and FlowRow, Snapshot.Edges and Components (snapshot.go) <- plan-2-6, plan-2-3
- Snapshot.Shipments, Away and TransitFactors, ShipmentRejected (snapshot.go, event.go) <- plan-2-12
- Snapshot.Contracts, Escrow and RegionRow.MoneyContract (snapshot.go) <- plan-2-15
- Snapshot.Population, RoleRow rates and RegionRow.HouseholdCap, PopulationScaled (snapshot.go, event.go) <- plan-2-11
- Snapshot.Journeys, RegionRow.OnJourneyFrom and OnJourneyTo, MigrationStarted (snapshot.go, event.go) <- plan-2-14
- Snapshot.Outside, OutsideEnded and the outside reasons (snapshot.go, event.go) <- plan-2-16
- OwnUseChanged and RoleRow.OnOwnUse (event.go, snapshot.go) <- plan-2-10
- the race-aware scenario gate and `-update` rule of plan 1, mirrored in `indicator_test`, not imported (the functions are test-only in package `economysim_test`)
- Snapshot, RoleRow, MarketRow, BoundaryFlow, WearApplied, Calendar, Config, Commodity.Classes, Commodity.ReferencePrice (snapshot.go, event.go, config.go, calendar.go) <- Stage 1
- checked.Acc and checked.MulDivHalfEven (internal/checked) <- Stage 0

Provides:
- Tally, NewTally, Window, Open, Close and Observe (indicator/tally.go, window.go) -> plan-2-19, plan-2-20, plan-2-24
- Prices, PricesOfWindow, PricesOfSnapshot, PricesOfReference, Base and BaseOf (indicator/prices.go) -> plan-2-19, plan-2-24
- the presets and their result fields named by STY (PriceIndex, Turnover, Pinned, PriceRange, Dispersion, Offset, Unfed, Households, RoleShares, StockCover, Sectors, GDP, WelfareDistribution, Inequality, PriceGap, RealWage) (indicator/*.go) -> plan-2-19
- Classes, NewClasses, FlowSel, Household, RowsOf, Stat, Stats, Growth, Change, Point, YearWindow, SeasonWindows (indicator/) -> plan-2-19
- the mirrored `-update` flag and `raceEnabled` of the external test package (indicator/race_test.go, norace_test.go) -> plan-2-19
- RoundDiv, Quantile and IsqrtNearest (internal/measure) -> plan-2-20 (its equivalence test against `internal/reduce`, spec-experiments 11.2)
- testdata/golden/indicator/one.txt and TestGolden_IndicatorPresetsOnOne -> plan-2-24 (two architectures)
- HouseholdIncome returning OK false (indicator/income.go) -> plan-2-19 (D574)

## Testing

Testing entry (testing.md 11). Requirement IDs touched and the tests that cover each (full names in the phases):

| Requirement | Tests |
| --- | --- |
| FR-IND-001, NFR-IND-004 | `TestImportGuard_Detects` rows and `TestImports_StdlibAndLayers` (Phase 1) |
| FR-IND-002 | `TestSignature_NoWorld` |
| FR-IND-003 | `TestRun_NonInterference` |
| FR-IND-004, 005, 006, 007, 011 | `TestTally_SizeIsAFunctionOfTheConfiguration`, `TestTally_IgnoresWhatItDoesNotUse`, `TestClose_WindowOrder`, `TestClose_ConfigMismatch`, `TestTally_ConstructionBucketIsAMemoLine`, `TestTally_RunningHouseholdsEqualTheSnapshot` |
| FR-IND-008, 054 | `TestMeasure_NoFloatNoMath`, `TestIndicator_NoGlobalStateNoGoroutineNoFloat` |
| FR-IND-009, 010, 012, 023 | `TestToInt64_Overflow`, `TestGuardBeforeTimes10000`, `TestInt128_ArithmeticLaws`, `TestStat_RowOrderDoesNotMatter`, `TestRoundDiv_HalfEvenBothSigns`, `TestClampBP`, `TestPriceIndex_StaleShareAndClamp` |
| FR-IND-013, 020, 021, 022 | `TestRealAndDeflator_Identity`, `TestPriceIndex_Food`, `TestIsqrtNearest_NoTie`, `TestPriceIndex_ChainFollowsTheRule` |
| FR-IND-014, 015, 024, 025, 026, 027 | the Gini, rank and `RowsOf` tests of Phases 2 and 7, `TestInequality_CohortShare` |
| FR-IND-016, 017, 018, 019 | the worked-example tests of Phase 5 and `TestRun_UnitsConserved`, `TestRun_GDPIdentityHolds`, `TestRun_NetWorthIdentity`, `TestRun_RegionalMoneyIdentity` |
| FR-IND-028, 029 | `TestPricesOfWindow_VolumeWeighted`, `TestClasses_UnknownAndOverlap` |
| FR-IND-030 to 050 (050 first clause) | one worked-example test per preset (Phases 5 to 7), the clauses of the row's `Must not hide` as separate checks |
| FR-IND-051, 052, 053, 055 | `TestBinding_WarningCarriesTheCounters`, `TestYearWindow_PartialFirstYear`, `TestSeasonWindows_WrapsTheYear`, `TestAnnualise_ExactFromSums`, `TestTally_RestoredWindow`, `TestTally_OpensAtWorldRestored` |
| FR-IND-050, second clause (booked income, relief, cost, ratio and Gap) | deferred, plan-2-0 deferred table, owner Stage 3 (the plan that adds the RoleRow income fields) (D574); stub `TestHouseholdIncome_BookedFromRoleRowFields` (`t.Skip` naming N-130 item 1) |
| FR-IND-056 to 059 | deferred, plan-2-0 deferred table, owner Stage 3 plans of EMP, RSK and STO (D560) |
| NFR-IND-001, 002, 003 | `TestRun_CostIndependentOfAgents`, `TestTally_ObserveAllocatesNothing`, the three fuzz targets of Phase 8 and the three of Phase 2 |

- **Golden files:** `testdata/golden/off/` does not move and no module-alone golden exists (IND is core, not a module, architecture.md 2.10). One new golden, `testdata/golden/indicator/one.txt` (D577). A diff in `golden/off/` or the pinned hash is a defect of this plan.
- **Fork-at-N:** not applicable (no world state, no section). `TestTally_RestoredWindow` is the nearest test: a tally across a restore.
- **Registry and catalogue rows:** none added, none deleted from `notBuilt`; no event is added to the catalogue test.
- **Fuzz:** `FuzzGini`, `FuzzRank`, `FuzzRoundDiv`, `FuzzIndexFunctions`, `FuzzRowStatistics`, `FuzzAccounts` (corpora in each package's `testdata/fuzz/`).
- **Coverage and mutation:** at least 90% for `indicator/` and `internal/measure`; gomu on `internal/measure` (D579).
- **Bug-fix tests:** none (new code). `Guard:` tests: the import rows, `TestRoundDiv_AgreesWithChecked`.

## Observability

Observability entry (observability-and-logging.md 9): **None, because** indicators emit no event, write no log and register no instrument (spec 14). `indicator` has no `*slog.Logger` and no `WithLogger`: a warning of a result is a field of the closed set (D563), because a measure is called by the consumer's code and the consumer decides what to log (rules 2.3). A preset as an OpenTelemetry gauge is plan 22's catalogue to extend (spec 14.2). The `Tally` is an `Observer` and so receives the events of the catalogue; it adds none and must not call the world (it holds none). The test that no event class O or R is needed is `TestTally_IgnoresWhatItDoesNotUse`.

## Risks and open items

1. **The inputs of the identities come from plans that are written in parallel.** FR-IND-016 and 019 need `Snapshot` fields for transit stock (`Shipments`, plan 12), `HeldByTreasury` (POL, later), `RegionRow.MoneyContract` (plan 15, OBS 5.3.2) and `AgentRelocated.Units` (plan 3). All but one are declared `-> plan-2-18` by their plans (see Consumes); the gap is the POL treasury term (Stage 3). If a term is missing when Phase 5 starts, the identity is asserted on the terms that exist and the PR lists it. Owner: supervisor, at the cross-plan check.
2. **`ShipmentDispatched` fields.** `HaulUnits` and `Lines` (spec-observability 4.1) must exist for `DeliveredPrice` (FR-IND-034); per-line carrying units are not adopted (D575 item 5). Owner: plan 12.
3. **`RegionPrices` is plan 3's** (`Config.RegionPrices`, D265; `World.ReferencePrice`, D266). `PricesOfReference` reads the region's row, so `BasketLevel` is region-specific. Owner: plan 3.
4. **`BasketLevel` uses the world upkeep plus `ExtraUpkeep` (spec 5.3.2), not a role's `Upkeep` override** (`config.go:367`). A role with its own upkeep is outside the fixed basket by definition. Owner: IND spec; no change proposed.
5. **`HouseholdIncome` (D574).** `IncomeLast`, `ReliefLast`, `CostLast` are not added in Stage 2; this plan ships `OK` false. Owner: Stage 3 (the plan that adds the RoleRow income fields), N-130 item 1.
6. **Spec 3.3 names `World.Graph()` and `World.Calendar()`**; the second does not exist (D569). Owner: IND spec, owed edit 2.
7. **FR-IND-016 to 019 over `five`.** The identities are tested on `one` and `second`. `five` may not step before plan 3, and plan 4's Phase 5 probe result (plan-2-4 D291, Risk 4) says whether it does; plan 24 settles it and re-runs the identity tests on it. Owner: plan 24.
8. **Mutation cost.** gomu on `internal/measure` per file; if a file exceeds the Docker budget it is split, not run bare on the host. Owner: this plan.
9. **Derived rule worth a spec line.** `Window` carries no stocks (D566): the snapshot arguments are the caller's. Spec 4.2.6 should say so (add to owed edit 2).

**Note proposals (text only; the supervisor is the only editor of `doc/notes.md`):**
- N-130 update: "(plan 2-18) Read for IND. Item 1 not adopted: `HouseholdIncome` returns `OK` false; no plan owns the fields. Item 2: `WearApplied` exists and no preset reads it; depreciation is the retirement basis. Item 3: `BenchmarkRowsOf` measures a pass at 5 000 agents; no bulk read is asked. Items 4, 5, 6 not adopted. Left open."
- N-206 update: "(plan 2-18) Enforced: `import_guard_test.go` fails any package other than `indicator` and its tests that imports `indicator`, and any other than `indicator` that imports `internal/measure`. Left open for PLC question 1."
- N-208 update: "(plan 2-18) A region set is already `FlowSel.Regions`; `PriceLevelOf` and a jurisdiction selector are not adopted in version 1. Left open for PLC."
- New note (areas IND, EMP, RSK, STO): "Presets 22 to 25 of `spec-indicators.md` 9.1 (FR-IND-056 to 059) are built with EMP and RSK/STO: each of those plans adds its `Observe` case, its tally rows, its preset and its tests (`indicator/`)."
- New note (areas IND, CODE): "`internal/measure` imports `internal/checked` (plan 2-18 D562); `spec-indicators.md` 3.2.1, 5.2.1 and NFR-IND-004 say standard library only and are corrected."
