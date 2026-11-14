# Plan: Stage 3, row TEC: `technology.go` and `internal/technology`

**Status: plan written (2026-11-14), Stage 1 of the work (requirements and plan, no code). Written on branch `s3-tec`, cut from the integration branch `phase-3` (= `d996592`, which holds the merged plans and first PRs of SKL, POL and STO). Format model: plan-3-sto.md, plan-3-pol.md and plan-3-skl.md. The owner questions stay open. Revised after the audit of `dp3-tec-audit.md` (7 blocking, 8 non-blocking items, each checked against the tree before the edit).**

Sequences the roadmap-v1.md Stage 3 row "`technology.go`, `internal/technology` | TEC | PRO, REG, CAL" into nine phases and three pull requests. It implements [spec-technology.md](../specs/spec-technology.md) (TEC): FR-TEC-001 to 037 and NFR-TEC-001 to 005. The module is a gain in yield held per region and field (a cell), raised by four mechanisms (learning by doing, a trend, discovery, diffusion by contact), entering production as one more factor of the effective yield, with one public call (`AdjustTechnology`), two events, one position (P8.3c) and one checkpoint section (tag 33). It has no per-agent state, no draw in a tick, no ledger reason, no account and no goods (position 12 of the spec).

Rule tags read (the three files, in full): architecture=354c9304a17403fb4bd93ae8, observability=f215753306fac0bc7607c4c4, testing=66af7645b01fef4e2796c367. The architecture tag differs from the one quoted by the three earlier plans (bdb8a4b7fa1ed8fd3b2e7aec): the file was edited after they were written, and the tag above is the one in the file at `d996592`.

Specs read: spec-technology.md (in full); spec-config.md (3.1, 3.7.1, the TEC rows, 4.1, 8.2); spec-world.md (5.1.2 sub-step 8c, 5.7, 6.4 P2.1 and P2.7, 6.10 P8.3c, 7.3 row D16, 7.1); spec-checkpoint.md (4.2 tag 33, 6.8); spec-observability.md (4.1.11, 5.3 `Snapshot.Tech`, `Names.TechFields`); spec-integration.md (5 and the call table); spec-production.md (6.3, FR-PRO-022, 10.3); spec-reference-chain.md (4.5 `overlay/technology.toml`, S148 to S154). The earlier plans, the owner notes `phase-3-owner-questions.md` and `DP3-skl.md`, `DP3-pol.md`, `DP3-sto.md`, and the roadmap Stage 3 table were read.

Notes: the index of `doc/notes.md` was read. The only note tagged TEC is **N-214** (keyed random streams; its TEC leg is spec 4.4.1 and 4.4.6). Of the ALL notes, N-126, N-128, N-163, N-217, N-315, N-335, N-340, N-345, N-351 and N-353 were read (the entry text); N-016, N-017, N-027 and N-103 were not re-read (their effect on this module is in spec-technology.md 17.1, which records how each was applied). **`doc/notes.md` was not read in full and no notes tag is claimed**: this plan writes no spec. Phase 1 edits specs and must read the whole file and quote the tag (AGENTS.md).

Decision block: this plan numbers its decisions TC1 to TC14; the supervisor maps them to D numbers of the stage when it merges the plan.

## What is already built (from the tree at d996592)

- `internal/production/yield.go`: `Effective(site, shock, tech, trainee, abundance)` already has the `tech` term, and `yieldTable.effectiveFor` (declared at `production.go:171`, the call of `production.Effective` with the neutral `production.BP` at `:173`) passes the neutral term. `effectiveAt` (declared at `:164`, passing `production.BP` into `effectiveFor` at `:165`) and `effective` (neutral: the warm start and the solver) reach it. The five reading sites of the effective yield are `strategy_view.go:235` (the `Work` view), `phase_production.go:307` (`startReq`, the start), `production_report.go:184` and `:202` (the cost anchor of P1.2 and `bestSeasonFactor`) and `skills.go:303` (`traineeRequest`, through `effectiveFor`).
- Completion: both P2.1 (`p21Complete`) and the one-tick start of P2.7 reach one function, `finishJob` (`phase_production.go:136`, called at `:127` and `:424`); the `completion` carries `recipe` and `executions`.
- Checkpoint: `checkpoint_sections.go:78` lists `{33, "Technology", "Technology"}` in `notBuiltSections`; tag 34 (Extensions) is the last built row.
- Positions: `tick.go` lists `P8.3c` in `notBuiltPositions`; the straight line ends in `positionsTail` with `p83Regen` then `p84Audit`. The position table of spec-world.md 6.10 has the row P8.3c (owner TEC, after P8.3b).
- Registry and catalogue staging: `registry_test.go` lists `Modules.Technology` (line 45), `TechField` (68) and `Technology` (69) in `notBuilt`; `event_test.go:96` and `render_test.go:40` list `TechnologyAdvanced` and `TechnologyAdjusted` in `notBuiltEvents` and the render list. There is **no** `notBuiltFields` entry for TEC: the entries at `event_test.go:114` are all other modules' (`WorldCreated.Names` is not among them); `doc.go:445` mentions the technology term in a comment. The poshash and call-site tables (`replayRows` in `poshash_internal_test.go:505`, `stepSites` in `observe_internal_test.go`) carry one row per position and have none for P8.3c.
- Documents that already carry TEC text (verified by search): the rule files (the two events in observability-and-logging.md 4.3, the four metrics in 5.3, the keys `field`, `fields`, `gain` in 3.3.2, the attribute `economysim.field`; architecture.md 2.6 already lists `technology.go` and `internal/technology`); spec-world.md (P8.3c, draw D16 retired, 5.1.2 step 8c, 5.7); spec-production.md (the tech term in 6.3, FR-PRO-022, 10.3); spec-integration.md (the call and `ErrUnknownTechField`); spec-observability.md (`Names.TechFields`, `Snapshot.Tech`, the events); spec-config.md (the registry rows, the bounds, the hash id 24).
- `otelobserver/catalogue_want_test.go` already lists the four `economysim.technology.*` instruments; the adapter code does not feed them.
- Nothing in the merged STO, SKL or POL code names TEC. The one open question SKL left for TEC (do a trainee's executions count?) is answered by spec 3.4.3: they count in full. No seam is owed to those modules.

## Findings that shape the plan

- **The key of discovery is stale in the TEC spec.** ADR 0008 and N-214 retired draw D16: spec-world.md 5.1.2, 5.7 and 7.3 and spec-checkpoint.md 6.8 now say the technology key is derived from the effective seed and takes no word, and a reseed re-derives nothing of its own. But spec-technology.md 4.4.1, 4.4.6, 6.3, 12.1, 12.2, FR-TEC-011, FR-TEC-012, FR-TEC-027 and FR-TEC-028 still say the world draws one word at construction, saves `Key` and re-derives it at a reseed, and spec-checkpoint.md 4.2 tag 33 still lists `Key`. Saving a key derived from the seed would contradict "a reseed re-derives nothing". Phase 1 rewrites these (decision TC3, question 1).
- **The hash inner ids are not in spec-config.md 8.2.2.** Only the module id (24) is fixed. Phase 1 adds the inner ids by the 8.2.2 rule (the position in the struct declaration).
- **`spec-technology.md` 4.1 and 6.1.2 say the counters are cleared at P8.6.** The plan zeroes them where they are folded (P8.3c), which is the same behaviour and saves a hook in `endStep` (TC7).
- **`validate.go` `overflow()` needs the module's bound** (FR-TEC-020): the largest technology factor `MaxTechBP` enters the product of a recipe in a field. The closure `yieldMax` is where it goes, as one call. **`overflow()` is already far over the Lizard limits** (measured with `lizard -l go -C 10 -L 100 -T nloc=50`: `overflow` `validate.go:1498` NLOC 84, CCN 27, 120 lines; the closure is inside it at `:1531`). STO and POL did not touch it (their `validate.go` change is one `v.validateX()` call in `structural` and one `xWarnings` in `warnings`), so there is no precedent to copy. Phase 3 therefore starts with a refactor-first commit (H8).
- **Diffusion on `one` is impossible**: `one` has one region, so `diffusion_no_contact` fires and nothing diffuses. The module-alone golden that completes the feature needs at least two connected ordinary regions: pack `three`.

## Scope

**Builds.**

- `internal/technology` (new package; standard library, `internal/checked` and `internal/ordering` only; primitives only; architecture.md 2.9): `Lg12`, `Learned` (FR-TEC-006), `TrendStep` (the carried remainder), `Key` and `Hit` (the keyed hash of 4.4), `Weight` (the contact weight, FR-TEC-014), `Frontier` and `CatchUp` (over a snapshot of gains, FR-TEC-013, 015), the step composition `Step` over slices of cells with the identity of the event (FR-TEC-017) and the saturation of the experience (FR-TEC-035), and `Overflow` helpers for the products of spec 9.5. The package never imports the root.
- Root: `technology.go` (module-file order of architecture.md 2.10.1: the state struct, `validateTechnology` hand-off, `newTechnology`, `p83cTechnology`, the counter, the gain reader, the call `AdjustTechnology`, `TechFieldID` helpers), `technology_validate.go` (continuation, TEC: the rules of 10.1, the warnings of 10.2 to 10.4) and `technology_report.go` (continuation, TEC: the events, the log lines, `TechRow`, the checkpoint encoder and decoder). The continuation files are for the Lizard limits of N-243; both map to TEC and their tests carry its `Requirement:` lines (architecture.md 2.6). Question 2 asks the owner to confirm them.
- Catalogue and hook appends (the table below): `config.go`, `bounds.go`, `errors.go`, `hash.go`, `value.go`, `event.go`, `snapshot.go`, `tick.go`, `random.go` (one site constant), `config/mapping.go`, `internal/logkeys`, `scripts/ci/coverage-thresholds.txt`, `doc.go`.
- `testdata/chains/overlay/technology.toml` (learning only, REF 4.5) and `technology-four.toml` (all four mechanisms, a test double for the golden), both in PR A; `testdata/golden/technology/` (module alone), scenarios S148 to S154, the proof that `testdata/golden/off/` and `TestHashConfig_ReferencePinned` did not move.

**Public identifiers.** `Technology` (the configuration type, `Modules.Technology`), `TechField`, `TechFieldID`, `TechnologyAdjustment`, `World.AdjustTechnology`, `TechnologyAdvanced`, `TechnologyAdjusted`, `TechRow`, `Snapshot.Tech`, `Names.TechFields`, the sentinels `ErrTechShape`, `ErrTechRecipeOverlap`, `ErrUnknownTechField`, the bounds `MaxTechFields`, `MaxFieldRecipes`, `MaxTechBase`, `MaxDoublingBP`, `MaxTechTrendBP`, `MaxDiscoveryBP`, `MaxGainBP`, `MaxStepBP`, `MaxCatchUpBP`, `MaxTechBP`, `MaxExperience`. **No** option, no ledger reason, no boundary account, no `Cause` or `Reason` constant (spec 11.2), no default field (spec 9.1.2: every rate defaults to 0).

**Leaves out, with the owner.**

| Item | Owner |
| --- | --- |
| A recipe or facility class unlocked by a level, forgetting, a scale effect in discovery, a compounded law, per-agent experience, carriers of knowledge by migrants or merchants, a cap per region, technology from an outside node | later versions (spec 19, open questions 1 to 11) |
| The `Yield` preset of `indicator/` (spec 8.9, 17.2) and R9's units clause (STY) | IND, STY |
| The hired-job factor `HireYieldBP` composing with `tech` (spec-employment.md 6.1, FR-EMP-028) | EMP, when the hire form is built; the payroll form has no hire factor |
| A build or repair recipe reading a yield factor | CAP: its start path must pass the same term or read no yield (FR-CAP-011) |
| The four instruments `economysim.technology.*` in `otelobserver/` (adapter feed) | a later `otelobserver` PR, as for SKL and POL (question 6) |
| Band values for TEC-S1 to S7, calibration classes of the knobs | STY, CLB (Stage 2 criterion 1 is unmet, so none is set) |
| `testdata/golden/all/`, `format-1-all.*` checkpoint fixture, `allLayers` | not touched by any module PR; they move once at Stage 3 close (owner decision of 2026-11-13) |

## Decisions

- **TC1.** Layout: `technology.go`, `technology_validate.go`, `technology_report.go`, `internal/technology`. `Modules.Technology *Technology` is the 24th pointer of the `Modules` row: its **hash id is 24** whatever its Go position (spec-config.md 3.1 and 1689). Its Go line is appended after the last pointer present when it lands (architecture.md 2.10.5); the pair is written in `pairsLaterModules` after id 19 and before id 25 (ascending id). Reverse: inserting by row order would renumber nothing but would break the append-only rule.
- **TC2.** Hash inner ids by the 8.2.2 rule: `Technology.Fields` 1; `TechField`: `ID` 1, `Recipes` 2, `CapBP` 3, `Base` 4, `DoublingBP` 5, `TrendBP` 6, `DiscoveryBP` 7, `StepBP` 8, `CatchUpBP` 9, `Reach` 10; `Label` has none (presentation, spec 12.3). Fields sorted by ID bytewise, recipes sorted. A non-nil `Technology` is written even when empty (a module that is on with no field differs from one that is off). A spec sentence in 8.2.2 fixes the ids (E4).
- **TC3.** The key is derived, never drawn and never saved: `Key = hi word of ordering.StreamKey(effectiveSeed, siteTechnology, 0, 0)`, computed at construction and at a restore from the effective seed (`w.stream.seed`, after a reseed). `siteTechnology` is appended to `random.go` with the SHA-256 rule of that file (first 8 bytes, big-endian, of `economy-sim/site/siteTechnology` = `0xF68B3E6A63DA053C`, a record of origin only, held at least 2^32 from the other sites by the existing Guard test). The hit is the one of 4.4.2 and 4.4.3 unchanged: `K(t, c) = finalise(Key + (t * C + c + 1) * 0x9E3779B97F4A7C15)`. Consequences: FR-TEC-011 becomes "no word at construction or in a tick"; FR-TEC-012 becomes "the key is a function of the effective seed, so a reseed changes it"; tag 33 holds no `Key`. Reverse: saving the key makes a reseed branch discover on the parent's ticks, which is what 4.4.6 meant to avoid.
- **TC4.** State: `type technology struct` held as `w.tec *technology`, nil when `Modules.Technology` is nil. Module slices only (N-244, N-128): `cells []cell{experience, advance, catchUp int64}` of length `R_ord * F` in (region rank, field) order, where the rank counts ordinary regions only (an outside node has no cell, FR-TEC-036); `trendCarry []int64` per field; `counter []int64` per cell (scratch); `gain []int64` per cell (derived, never saved); `fieldOf []int32` per recipe (-1 for no field); `key uint64`. The agent type gets no field.
- **TC5.** The gain read is one allocation-free method: `(*productionScratch).techFactor(region, recipe) int64` defined in `technology.go`, returning `production.BP` when the module is off and `10 000 + gain` otherwise. `productionScratch` gets one pointer field `tec` set at construction beside `res`. The five reading sites pass it as the new `tech` argument of `effectiveAt` and `effectiveFor`; `effective` keeps passing `production.BP`, which is why the warm start, the solver and the viability check cannot see a gain (FR-TEC-034).
- **TC6.** Position P8.3c: `{"P8.3c", "TEC", line}` between P8.3 and P8.4 in `positionList`, `"P8.3c"` removed from `notBuiltPositions`, and the guarded call `if w.tec != nil { w.p83cTechnology() }` after `p83Regen` in `positionsTail`. Order inside the step is 4.6.1 (fold, trend, discovery, snapshot, diffusion, clamp and events). The step takes no word and reads no price, wage, stock or count of households (FR-TEC-033).
- **TC7.** The counter: `finishJob` calls `w.tec.count(region, c.recipe, c.executions)` under `if w.tec != nil`. Both completion positions reach `finishJob`; an abandoned job never does. The fold at P8.3c zeroes the counters it reads, so no hook in `endStep` is needed (spec 4.1 says P8.6: edited to say the fold clears them, E4).
- **TC8.** The call `AdjustTechnology` lives in `technology.go` (spec 20), enters through the existing `w.enter("AdjustTechnology", "technology", w.tec != nil)` guard like `AdjustEndowments` (`integration_slots.go:381`), and is not listed by `Slots()`. A test pins that it is absent from the slot listing.
- **TC9.** Events and rows: `TechnologyAdvanced` and `TechnologyAdjusted` appended to `event.go` in the order of OBS 4.1.11; `Names.TechFields` appended; `Snapshot.Tech []TechRow` appended (a slice, so it renders as before when off) and filled by one guarded call in `Snapshot()` (H11; `Region()` returns a `RegionState`, which carries no technology rows, so its second path needs no call: Phase 1 confirms it against spec-integration.md 6.5); `technology` joins `modulesOn`; the construction summary gains `fields`. The audit (`auditModules`, `moneyflow.go:865`) needs **no hook**: the module has no account and no ledger reason, and FR-TEC-032 is a test that the audit passes with every mechanism on. Events are built only under `w.obs.supplied`; the per-cell amounts are returned by the step only when an observer or a Debug logger needs them.
- **TC10.** Checkpoint: one section, tag 33, present when `Modules.Technology` is non-nil: per cell `experience`, `advance`, `catchUp` (three varints, count `R_ord * F` from the configuration, not written), then per field `trendCarry`. No key (TC3), no counter, no gain. The restore validates in the order of spec-checkpoint.md 6.2 and returns `ErrCheckpointCorrupt` for each failure of 12.2. The format version and `testdata/checkpoints/` fixtures do not move in this row (owner decision); the restore tests build their world from the overlay.
- **TC11.** Diffusion reads the graph's own distance, `Graph.Distance(from, to) (int64, bool)`, which already honours closed and slowed edges; the module keeps no table of weights (NFR-TEC-004). The contact weight is computed where it is used.
- **TC12.** Overlays: `technology.toml` is the learning-only field on the growers of `one` and `three` (REF 4.5), used by S148 to S154 with variants made in the tests by editing the loaded `Config`. `technology-four.toml` puts all four mechanisms on one field of `three` (a test double, not a design) and carries the golden. Each overlay file states its layer order and a reason, as `overlay/polity.toml` does.
- **TC13.** Nothing existing is regenerated. `testdata/golden/off/`, the pinned hash and `testdata/checkpoints/` must not move at any phase exit. `testdata/golden/technology/` is new and recorded once with `-update` after its diff against `off/` is read.
- **TC14.** The module has no tunable of its own and no default (spec 9.1.2), so `defaults.go` gets a one-line comment only, `canonical()` sorts fields by ID and recipes, and the registry rows leave `notBuilt` in the PR that builds each (the test fails on a stale entry).

## Requirement list and where each is verified

"Unit" is `internal/technology` or `internal/production`; "root" is the root package; "world" means a built world.

| ID | Phase | Lowest-level test |
| --- | --- | --- |
| FR-TEC-001 | 5 | `TestScenarioS148_InertEqualsOff` (harness, seeds 1 to 10 under `scenarioSeedCount()`): on with no field and on with every rate 0 equal off, except `ConfigHash`, `Modules`, `Names` |
| FR-TEC-002, 003 | 3 | `TestValidateTechnology_Errors` (one case per rule, `errors.Is`, the path in the message) |
| FR-TEC-004 | 4 | `TestNewTechnology_CellsAscendingZero`, `TestNewTechnology_OutsideNodeHasNoCell` |
| FR-TEC-005 | 2, 4 | `TestGain_CappedAndMonotone` (unit property), `TestAdvance_StoredClamped` |
| FR-TEC-006 | 2 | `TestLearned_ExactAtDoublings`, `TestLg12_Table` (vectors of `doc/specs/models/tec/law.out`) |
| FR-TEC-007 | 5 | `TestCount_FoldsExecutions` (P2.1 and P2.7, trainee, own-use, abandoned excluded, weight independent) |
| FR-TEC-008 | 2, 4 | `TestTrend_ExactOverYears` (unit, then world) |
| FR-TEC-009 | 2 | `TestHit_Threshold`, `TestHit_RateOverCellTicks` (unit, fixed keys) |
| FR-TEC-010 | 5 | `TestGain_SameAtEveryPosition` (P1.2, P2.5, P2.7 read one value; it changes only at P8.3c and a call) |
| FR-TEC-011, 012 | 4, 8 | `TestKey_DerivedNoWord` (stream fingerprint equal with module on and off), `TestKey_ReseedChangesIt` |
| FR-TEC-013 | 2 | `TestDiffusion_SnapshotAndPermutations` (six orders) |
| FR-TEC-014, 015, 016 | 2 | `TestWeight_Table`, `TestCatchUp_CarryAndNoGap`, `TestCatchUp_NeverPassesFrontier` |
| FR-TEC-017 | 2 | `TestStep_EventIdentity` (property over 100 000 random cells, fixed seed) |
| FR-TEC-018, 019 | 5 | `TestEffective_TechTermOrder` (unit), `TestStart_OutputFixedAtStart` (world) |
| FR-TEC-020 | 3 | `TestValidate_OverflowReadsMaxTechBP` |
| FR-TEC-021, 022, 023 | 6 | `TestAdjustTechnology_*` (atomic, order of errors, event fields, not in `Slots()`) |
| FR-TEC-024 | 6 | `TestTechnologyAdvanced_PerCellOnlyWhenAccrued` |
| FR-TEC-025 | 6 | `TestNames_TechFields`, the cardinality test gains the field range |
| FR-TEC-026 | 6, 8 | `TestSnapshot_TechRows`, `TestSnapshot_RestoredEqualsOriginal` |
| FR-TEC-027, 028 | 8 | `TestCheckpoint_TechnologyRoundTrip`, `TestCheckpoint_TechnologyRefusesCorrupt`, `TestFork_Technology` |
| FR-TEC-029, 030, 031 | 3 | `TestWarnings_Technology` (one case per row of 10.2 to 10.4, with the RES and POP fixtures) |
| FR-TEC-032 | 5, 9 | `TestAudit_TechnologyCreatesNothing` (`WithAudit`, every mechanism on) |
| FR-TEC-033 | 5 | `TestStep_PureFunctionOfInputs` |
| FR-TEC-034 | 5 | `TestTechnology_TickZeroUnmoved` (the yield table, the warm start and the viability report equal with the module on) |
| FR-TEC-035 | 2 | `TestExperience_SaturatesNoWrap` |
| FR-TEC-036 | 4, 5 | `TestContact_ClosedEdgeStopsAndResumes` (`SetEdgeStatuses`) |
| FR-TEC-037 | 6 | `TestLogs_TechnologyDebugOnly` (no `Info` or `Warn` in the tick, `Enabled` guards) |
| NFR-TEC-001 | 8, 9 | `TestDeterminism_Technology` (`detCases(true)`, split `Run` calls, with and without observer); the arm64 run is by hand at Stage 3 close |
| NFR-TEC-002 | 2, 5 | `TestDiffusion_SnapshotAndPermutations`, `TestStep_CellOrderPermuted` |
| NFR-TEC-003 | 5 | `TestStable_TechnologyAllocNoOp` (`AllocsPerRun`), `BenchmarkP83c` |
| NFR-TEC-004 | 5 | `TestStep_OperationCounts` (counts, no timing), `Guard:` text search that the module keeps no table sized by regions squared |
| NFR-TEC-005 | 2, 3 | `TestProducts_AtBounds` (spec 9.5, each product at its bound inside `int64` or the checked helper) |

Also touched and re-checked, not owned: FR-PRO-022, FR-PRO-009, FR-SKL-014 (a trainee's `TraineeYield` multiplies the technique's output), FR-CFG-097, FR-OBS-003, FR-OBS-091, FR-WLD-076.

## Phases

### Phase 1: the owed edits, the audit and the glossary (no code)

Goal: the specs, rules and notes are true before the code starts. A rule file edit gets a fresh RULE-TAG. The shared edits are applied by the supervisor.

| # | File and section | Edit |
| --- | --- | --- |
| E1 | architecture.md 2.6, AGENTS.md Layout | add `technology_validate.go` and `technology_report.go` to the continuation files (TEC), identically in both lists; fresh RULE-TAG. `technology.go` and `internal/technology/` are already listed. Only the names Lizard forces stay: a name unused at the close of PR C is removed in that PR |
| E2 | `doc.go` glossary (architecture.md 3.2) | the terms of spec 17.3 (technology, technology field, cell, experience, learned gain, advance, gain, technology factor, doubling, base, trend, discovery, key, contact, contact weight, reach, frontier, gap, catch-up, cap, practises, premium) and the type names `Technology`, `TechField`, `TechFieldID`, `TechnologyAdjustment`, `TechRow`, `TechnologyAdvanced`, `TechnologyAdjusted` (`TestGlossary_DeclaredTypesNamed`) |
| E3 | spec-technology.md 4.4.1, 4.4.6, 6.3, 6.3.4, 12.1, 12.2, FR-TEC-011, 012, 027, 028 | TC3: the key is derived from the effective seed, takes no word and is not saved; D16 is retired; a reseed changes it by changing the seed |
| E4 | spec-technology.md 4.1, 6.1.2 | the fold at P8.3c clears the counters it reads (TC7) |
| E5 | spec-checkpoint.md 4.2 tag 33 | the row loses `Key` |
| E6 | spec-config.md 8.2.2 | the inner ids of TC2 |
| E7 | spec-world.md 6.4 (P2.1, P2.7) | verify the clause "the executions of a completed job of a recipe in a field are added to the counter of its cell"; add it if missing |
| E8 | roadmap-v1.md Stage 3 table | the TEC row says "plan 3-tec" and its status |
| E9 | notes | N-214: remove the TEC leg once E3 is applied; N-202 settled by TC3; add the four new notes of spec 17.1 (a gain is not a recipe; R9's volume clause is on units; a module that keeps a key derives it from the seed; the stability check is made with the module off); the supervisor numbers them |
| E10 | rule files | verify by text search that the keys `field`, `fields`, `gain`, the two events, the four metrics and the attribute `economysim.field` are present (they are); no edit expected |
| E11 | spec-reference-chain.md 4.5 and 7 | S148 to S154 are placed on `three` with `overlay/technology.toml` and `overlay/technology-four.toml` (TC12, question 5); S150 uses `overlay/trade-three.toml` (the file the tree has), not the `overlay/trade.toml` the table names, which does not exist; the layering order of 4.5 gains `technology-four`. If the owner rejects question 5, this row reverts to `one` |

Audit (read-only, recorded in the result paragraph): for each "already built" bullet above, name the test that guards it; name every test that calls `effectiveAt` or `effectiveFor` (they gain an argument); check that `Graph.Distance` honours a closed edge; check the `Slots()` listing test and the integration call tables for a place where `AdjustTechnology` must be named; confirm the checkpoint section order with tag 33 after the last present row and before tag 34; check that `RegionState` (`snapshot.go:400`, returned by `Region`) is meant to carry no technology rows (spec-integration.md 6.5); confirm what `TestGlossary_DeclaredTypesNamed` (`glossary_test.go:124`) does: it checks one direction only (every exported type declared in `value.go`, `config.go`, `event.go`, `snapshot.go` and the other listed files is named in `doc.go`; a glossary name with no declaration is not an error), so E2 can land in PR A with the names of types that only PR B declares (`TechnologyAdjustment`, `TechRow`, the two events); if the check proves two-way when it is read in Phase 1, E2 splits and the PR B names move to PR B; name the test that guards each corrected name (Phases 4 and 7: `TestSiteConstants_Guard`, `TestSections_AgreeWithSpec`, `TestSave_Sections`); run Lizard on the functions the hooks touch and put the figures in the plan (measured at `d996592`: `overflow` NLOC 84, CCN 27, 120 lines; `positionsTail` NLOC 30, CCN 8; `pairsModules` CCN 9; `pairsLaterModules` CCN 7; `namesOf` CCN 6; `finishJob` CCN 4; `modulesOn` CCN 3).

Exit check: E1 to E11 applied or logged with an owner; `go test ./...` unchanged; `testdata/golden/` unchanged; `npx markdownlint-cli2` clean on every changed `.md` (the Codacy hygiene rules below).

### Phase 2: `internal/technology` (pure rules)

Goal: every number of spec 4 as a function over primitives, proved against the worked examples and the model outputs. Files: `internal/technology/{doc,law,trend,hash,contact,step,products}.go`, tests, the row `internal/technology 90` in `scripts/ci/coverage-thresholds.txt` (the checker fails a package that is not listed; this row lands in PR A).

- `Lg12(v) int64` and `Learned(doublingBP, base, x) (int64, bool)` (4.2.1, exact at a doubling, checked); `TrendStep(carry, trendBP, ticksPerYear) (add, carry2)`; `Key(seed, site uint64) uint64` and `Hit(key uint64, tick, cell, cells, discoveryBP, ticksPerYear int64) bool` (the 128-bit product by `bits.Mul64`); `Weight(reach, d int64, ok bool) int64` (4.5.1); `Frontier` and `CatchUp(carry, gap, catchUpBP, ticksPerYear) (add, carry2 int64)` (4.5.2 to 4.5.4); `Step(cells []Cell, counters []int64, fields []Field, dist DistanceFunc, ...) Result` composing 4.6.1 and returning the identity components; a distance is passed as a function so the package imports nothing of the graph.
- Tests (table-driven, got/want, the arithmetic of the spec in the comment, `Requirement:` lines as in the table above): the vectors of `doc/specs/models/tec/law.out` (exact at 21 doublings; within 0.086 between), `doc/specs/models/tec/worked.out` (example A of 16.1 over five years, one step by hand), the 100 000 random cells of the identity, six permutations of the diffusion order, a closed edge as `ok` false, the keyed hash over two keys and two cells, overflow at every bound of 9.5. `FuzzStep` (never panics, the identity holds, `Clamped` is not negative, `After` is not below `Before`), `FuzzLearned`.
- Exit check: `go test -race ./internal/technology/...`; coverage at least 90%; the package imports `internal/checked`, `internal/ordering` and the standard library only (the generic import guard enforces it); `gomu` in Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4 --workers 4`, per file, survivors killed or recorded as equivalent; Lizard clean on the package.

### Phase 3: configuration, bounds, validation, hash, loader (PR A)

Goal: a configuration with the module loads, validates, hashes and round-trips; the module off hashes as before; nothing runs. Files: `config.go` (`Technology`, `TechField`, the pointer `Modules.Technology` with the comment "Its hash id is 24, whatever its place in this struct"; `canonical`), `value.go` (`TechFieldID`), `bounds.go` (the eleven bounds: the ten of spec 9.2, `MaxTechFields` to `MaxTechBP`, and `MaxExperience`), `defaults.go` (a comment), `errors.go` (`ErrTechShape`, `ErrTechRecipeOverlap`, `ErrUnknownTechField`), `hash.go` (`pairsTechnology`, id 24), `validate.go` (the call `v.validateTechnology()` after the last module's, the refactor-first commit that hoists `yieldMax` out of `overflow()` and the one call in it, see H8), `technology_validate.go` (10.1 errors, 10.2 warnings, 10.3 and 10.4 inequalities with the checked 128-bit helper), `config/mapping.go` (`[modules.technology]` and `[modules.technology.field.<id>]` with `recipes = [...]`), `internal/logkeys` (`Field`, `Fields`, `Gain`), `registry_test.go` (see below), `doc.go`, and **the two overlays**, `testdata/chains/overlay/technology.toml` and `technology-four.toml` (moved here from Phase 8, so that `TestLoad_Technology`, the `FuzzLoadTechnology` seed corpus under `config/testdata/fuzz/FuzzLoadTechnology/` and every PR B world test have a text to load; the golden stays in PR C). Each overlay states its layer order and a reason (TC12). The PR B tests build their worlds by loading these overlays and editing the `Config` in the test where a variant is needed; no Go-built duplicate is kept.

- `registry_test.go` edits, named: delete `Modules.Technology` (line 45), `TechField` (68) and `Technology` (69) from `notBuilt`, and add `"Modules.Technology"` to `exemptFields` (a switch pointer, as `Modules.Polity` and `Modules.Storage` are). There is no `notBuiltFields` entry to delete. Phase 1 checks whether an overlay census test (`TestPacks_*`, `TestFixture_*`) requires an `expected/` row or a loads-test per overlay, and if so adds the loads-test to this phase.

- Validation order is the spec's (10.1): shape (`ErrTechShape`: empty `Recipes`, `CapBP` 0, a field of an off mechanism non-zero, a field an on mechanism needs equal to 0), duplicates and counts (`ErrDuplicateID`, `ErrTooMany`), bounds (`ErrOutOfBounds`, a `Label` above `MaxLabelBytes` included), recipe references (`ErrUnknownRecipe`), overlap (`ErrTechRecipeOverlap`), overflow (`ErrOverflow`, FR-TEC-020).
- Warnings (each a `Warn` at construction with `rule` and `path`, never fails validation): `field_inert`, `field_unused` (no recipe of the field is on any role's list or own-use list), `diffusion_no_contact` (checked at construction only), `technology_past_maximum_yield` (reuses the abundance curve of `internal/endowment` and `resourcePlan`) and `trend_outpaces_schedule` (reuses `Population.replacementOf`, which returns `r*`, `dR`, `dX`).
- Tests: `TestValidateTechnology_Errors` (FR-TEC-002, 003), `TestValidate_OverflowReadsMaxTechBP` (FR-TEC-020), `TestWarnings_Technology` (FR-TEC-029 to 031, including the worked examples 16.3: `CapBP` 1 999 silent and 2 000 firing, trend 200 silent and 201 firing), `TestTechnologyHash` (off: the pinned hash unchanged; on: `Label` not hashed; fields in canonical order whatever the written order; id 24 written after 19 and before 25; an empty non-nil `Technology` differs from nil; a change of any field changes the hash), `TestLoad_Technology` in `config/` (the overlay loaded from TOML equals the Go-built configuration), `TestParameterRegistry` (staged), `FuzzLoadTechnology` (never panics, valid input round-trips; any crash input committed under `config/testdata/fuzz/` with a `Fix:` comment), `FuzzValidateTechnology`, `TestProducts_AtBounds` for the validator products.
- First commit of the phase (refactor-first, no behaviour change): hoist the `yieldMax` closure of `overflow()` into a method of `validator` in `validate.go`, `golden/off/` and every test unchanged (H8). Lizard on `validate.go` before and after: the hoisted method is within the limits and `overflow` falls, but it stays over them (an existing finding, recorded in the PR; splitting the rest of `overflow` is out of scope).
- Exit check: `testdata/golden/off/`, the pinned hash and the position and event catalogue tests unchanged; `go vet`, `staticcheck`; `config/` coverage not down.

### Phase 4: state, cells, key and the gain reader (PR B)

Goal: a world with the module on holds its cells, reads a gain and does nothing else. Files: `technology.go` (`technology` state, `newTechnology`, `techFactor`, `count`, the key derivation), `world.go` (`tec *technology`), `construct.go` (`w.tec = newTechnology(...)`, `w.prod.tec`, `namesOf`, `modulesOn`, the summary), `event.go` (`Names.TechFields`), `snapshot.go` (`Snapshot.Modules` through `modulesOn` only), `random.go` (`siteTechnology`), tests.

- Tests: `TestNewTechnology_CellsAscendingZero`, `TestNewTechnology_OutsideNodeHasNoCell`, `TestKey_DerivedNoWord` (the stream fingerprint of a world with a discovery field equals the one without it: no word), `TestSiteConstants_Guard` (`random_internal_test.go:268`, the existing Guard gains the new constant), `TestNames_TechFields`, `TestTechFactor_NeutralWhenOff`, `TestConstruction_Summary` (the key `fields`), `TestGain_PositionsReadOneValue` (a planted gain is read identically by the anchor, the view and the start).
- Exit check: with the module on and no field the world equals off except the three carriers (`ConfigHash`, `Modules`, `Names`); `off/` unchanged.

### Phase 5: the step, the counter and the production term (PR B)

Goal: gains rise and production sees them. Files: `technology.go` (`p83cTechnology`), `technology_report.go` (the amounts and the Debug line), `tick.go` (TC6), `phase_production.go` (the counter hook, TC7), `production.go`, `strategy_view.go`, `production_report.go`, `skills.go` (the five reading sites, TC5), `event_test.go` and `render_test.go` staging, and the two position tables that every new position must update in the same change: `poshash_internal_test.go` (`replayRows`, next to `{id: "P8.3", ... on: resourcesOn}` at line 505: add `{id: "P8.3c", run: noErr((*World).p83cTechnology), on: techOn}` with `func techOn(w *World) bool { return w.tec != nil }` beside `resourcesOn`; without the guard the off position goldens would gain a P8.3c line; `TestReplayRows_Guard` fails with "position P8.3c of tick.go has no method in the replay map" until it lands) and `observe_internal_test.go` (a `{"P8.3c", ...}` row in `stepSites` at line 241, in the order of `positionList`, which `TestCallSites_TheTableIsTheStraightLine` at line 404 keeps complete; the row is guarded so that the site is measured with the module on).

- Tests: `TestCount_FoldsExecutions` (both positions, trainee, own-use, abandoned excluded, a weight reduction mid-job counts `min(k, w)`), `TestEffective_TechTermOrder` (E = site x shock x tech x trainee x abundance, truncating at each division, 10 000 off, 0 when the site factor is 0), `TestStart_OutputFixedAtStart` (a gain set at P8.3c acts on the next tick's starts; a running job keeps its output), `TestGain_SameAtEveryPosition`, `TestStep_PureFunctionOfInputs`, `TestStep_CellOrderPermuted`, `TestContact_ClosedEdgeStopsAndResumes`, `TestTechnology_TickZeroUnmoved`, `TestAudit_TechnologyCreatesNothing`, `TestPositions_P83cListed`, `TestStable_TechnologyAllocNoOp` (the step and the counter allocate nothing with the default logger and observer), `TestStep_OperationCounts`, `BenchmarkP83c` with `b.ReportAllocs()`, `TestScenarioS148_InertEqualsOff` (seeds 1 to 10 under `scenarioSeedCount()`; keep only counters and hashes per seed, never one snapshot per seed: the POL S113 run was killed by earlyoom at 38 GB for that reason).
- Test names and the gate F prefix rule (`TestRaceSkips_HaveAnFPrefix`, `scenario_helpers_test.go:1634`): a root test that skips under `raceEnabled` must start with a `gateFPrefixes` prefix (`scenario_helpers_test.go:45`), or it runs in neither gate. `TestStable_TechnologyAllocNoOp` (`AllocsPerRun` skips under race) takes the `TestStable` prefix for that reason. `TestAudit_TechnologyCreatesNothing`, `TestStep_OperationCounts`, `TestStart_OutputFixedAtStart`, `TestGain_SameAtEveryPosition` and `TestContact_ClosedEdgeStopsAndResumes` do not skip under race (they skip under `-short` only, which the rule does not govern), so they keep their names; a test that must skip under race gets a prefix instead. `BenchmarkP83c` is exempt.
- Exit check: `TestStep_AllocationsStayAtTheBaseline` at its baseline with the module off (the existing rows); `off/` unchanged, and `testdata/golden/off/*.positions.txt` byte-identical (the `replayRows` guard keeps P8.3c out of the off lines, TC13); the prefix rule above holds for every PR B test (`go test -run TestRaceSkips_HaveAnFPrefix`); the effective-yield tests that call `effectiveAt` updated for the new argument only.

### Phase 6: the call, the events, the snapshot, the logs (PR B)

Goal: everything a consumer and an observer see. Files: `technology.go` (`AdjustTechnology`), `technology_report.go` (`TechnologyAdvanced`, `TechnologyAdjusted`, `TechRow`, `Snapshot.Tech`, the Debug lines), `event.go`, `snapshot.go`, `observe_events_test.go` (`sliceNotEmitted` gains the two names if the module-alone slice does not emit them; the rest is checked in Phase 1), and in `otelobserver/`: `events_test.go` (the zero-value list near lines 55 to 68 gains `economysim.TechnologyAdvanced{}` and `TechnologyAdjusted{}`), `adapter_test.go` (the two names go in `ignoredEvents`, line 277, **not** in `handledEvents`, because the feed is deferred, question 6) and `testdata/all_events.txt` (55 lines today, two appended in the order of the event list). The SKL change touched exactly these three (commit dec3bc4); `catalogue_copy.md` is checked in Phase 1 and edited only if it lists events. `catalogue_want_test.go` already lists the four `economysim.technology.*` instruments; whether the exhaustiveness test accepts instruments with no feed is read in Phase 1 (the POL and SKL instruments were in the same state); if it does not, the instruments join its deferred list.

- Tests: `TestAdjustTechnology_AtomicAndErrors` (the order of FR-TEC-022: `ErrInStep`, `ErrWorldFailed`, `ErrModuleOff`, then per entry `ErrUnknownRegion`, `ErrOutsideNode`, `ErrUnknownTechField`, `ErrOutOfBounds`, `ErrDuplicateID`; no state change on an error), `TestAdjustTechnology_ClampAndLost` (example 16.2: -500 on 900 gives 400; on 300 gives 0 with `LostBP` 200), `TestAdjustTechnology_EventOrderAndTick` (the tick of the next step, ascending (`RegionID`, field)), `TestAdjustTechnology_NotInSlots`, `TestTechnologyAdvanced_PerCellOnlyWhenAccrued`, `TestTechnologyAdvanced_Fields` (no map, the identity, every count present even when zero), `TestEvents_OrderAfterP83b`, `TestSnapshot_TechRows` (a function of the saved state, deep copy), `TestLogs_TechnologyDebugOnly` (no `Info` or `Warn` in the tick, `Enabled` guards, constant messages), `TestCardinality_TechField` (the field attribute is a configured field), `TestNonInterference_Technology`, the catalogue tests with the two events out of `notBuiltEvents`.
- Exit check: PR B gate (below).

### Phase 7: checkpoint tag 33 (PR C)

Goal: the module's state survives a checkpoint, and a restored world continues exactly. Files: `technology_report.go` (`saveTechnology`, `loadTechnology`, the cross-checks), `checkpoint_sections.go` (the row for tag 33, `notBuiltSections` loses it), tests.

- Restore order: the gain array is re-derived at load from `Advance` and the cap (it is never saved), and the key is derived after the reseed has set `w.stream.seed` (TC3), because FR-TEC-012 turns into "a reseed changes the key"; a test pins both (`TestCheckpoint_TechnologyGainRederived`, `TestFork_Technology` with `WithReseed`).
- Tests: `TestCheckpoint_TechnologyRoundTrip` (bytes equal after restore; sparse and dense worlds), `TestCheckpoint_TechnologyRefusesCorrupt` (each refusal of 12.2: cell count against the configuration, `Experience` above `MaxExperience`, `Advance` outside `[0, CapBP]`, `CatchUp` outside `[0, 10 000 * TicksPerYear)`, `TrendCarry` outside `[0, TicksPerYear)`, trailing bytes), `TestSections_AgreeWithSpec` (`checkpoint_internal_test.go:63`) and `TestSave_Sections` (`:190`), with tag 33 built, `FuzzDecodeTechnology` (no panic, bounded allocation), `TestFork_Technology` (through `forkAtN` and `forkCase` with `technology-four.toml` on `three`, N in {0, a tick before a catch-up add, a tick after a discovery, after an `AdjustTechnology` call}: events, snapshot, audit and checkpoint bytes equal; a branch restored with `WithReseed` discovers on other ticks and equals an uninterrupted reseeded run, FR-TEC-012).
- Exit check: `testdata/checkpoints/` untouched (the all-modules fixture does not gain this module; owner decision); the checkpoint tests of the other modules pass.

### Phase 8: overlays, golden, scenarios (PR C)

Goal: the module is proved end to end. **The pack that completes the feature is `three`** (open: question 5; spec-reference-chain.md says `one`, E11) (spec-reference-chain.md 4.5; three connected ordinary regions, so learning, trend, discovery and diffusion all act). Packs `one` and `two` have one and two regions and collapse early: `one` cannot diffuse (`diffusion_no_contact`) and serves the learning-only scenarios. Criterion 1 does not bite here as it does for SKL: experience accrues from the first ticks and the gain never falls without a call, so a collapsing run still records a rising gain. The overlay sets `Base` small enough that the gain moves in the recorded length, and a probe test fails if the golden would record no `TechnologyAdvanced`, no discovery or no catch-up add (seeds 1 to 4; fall back to `five` if `three` records none).

- Files: `testdata/golden/technology/` (the two overlays landed in PR A, Phase 3) (counts, events, positions, the per-position state hash including P8.3c), `scenario_s148_test.go` to `scenario_s154_test.go`, `testdata/chains/expected/<pack>.scenarios.txt` rows (created from an existing one if the pack has none), `testdata/chains/bands/` keys declared with a unit and no value, `TestDeterminism_Technology`, `TestInvariants` row, `TestConfigOnly_TechnologyNeedsOnlyConfiguration` (architecture.md 9.4: a field by configuration alone, on `testdata/chains/second/`). S150 layers `overlay/trade-three.toml`.
- Scenarios (spec 14.4, REF 7): S148 (TEC-S1) and S153 (TEC-S6) are harness tests asserted over seeds; S149 (learning with POP and RES, report), S150 (diffusion on `three` with `trade-three.toml`, a closed edge stops it: the lagging cell stays below `w` times the leader), S151 (logistic stock, a cap below and above the threshold of 10.3), S152 (staple with inelastic demand, report) and S154 (the call: a negative advance lowers the gain at once, `LostBP` when the bound cuts) report. Outcomes are recorded in `expected/*.scenarios.txt` whether they pass or not.
- Golden: recorded once with `-update` after reading its diff against `off/`; the PR explains it (testing.md 7.2). `golden/all/` is not touched. A check proves `golden/off/` and `TestHashConfig_ReferencePinned` unchanged.
- Exit check: the overlay runs for the length REF fixes with `WithAudit`; every new long test is in the quick pattern (`TestScenario`, `TestDeterminism`, `TestGolden`, `TestInvariants`, `TestFork`, `TestStylised` prefixes) so arm64 can run them, and `TestRaceSkips_HaveAnFPrefix` passes; the S-tests keep memory per seed constant.

### Phase 9: the exit gate

Gates run by hand on the dev host (testing.md 9.1) and recorded in the PR description: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`, `go test -short -race ./...`, gate R, gate F with `scripts/ci/coverage-check.sh` (every scope at its minimum, coverage not lower), `go mod tidy` no diff, in the root and in `otelobserver/` and each `examples/` module (no example uses technology; no change expected). `gomu` per new file of `internal/technology` in Docker with the caps. A deliberate-break pass (testing.md 9.3: flip the sign of the learned gain, drop the stored clamp of `A`, apply catch-up before the snapshot, put `tech` after the trainee term) shows the tests fail. Close: roadmap row to `merged (PR #...)`, notes updated, `spec-roadmap.md` and `documentation_index.md` updated, the plan audited with plan-audit. arm64 is not run per PR: once, before Stage 3 closes (owner decision).

## Pull request split

Three pull requests on `phase-3`, in order, each leaving `off/`, the pinned hash and `testdata/checkpoints/` untouched:

- **PR A (Phases 1 to 3): edits, `internal/technology`, configuration, bounds, validation, hash, loader.** The module can be configured and validated and does nothing at run time. It lands the catalogue appends of the configuration side early (where Stage 3 modules collide) and the coverage row. Size estimate: 3 000 lines including tests (the two overlays and the position-table edits are in A or B now).
- **PR B (Phases 4 to 6): state, key, step, counter, production term, call, events, snapshot, logs.** The module works for a world that is never checkpointed. Size estimate: 3 200 lines including tests (the two position tables, the otelobserver lists).
- **PR C (Phases 7 to 9): checkpoint tag 33, fork, overlays, golden, scenarios, exit gate.** Size estimate: 2 000 lines plus one golden of some tens of thousands of lines (the migration golden was 22 000).

**Minimal slice the roadmap implies.** The row needs PRO, REG and CAL only (no SET, ORG, EMP or STO), so no other Stage 3 module blocks it. The smallest slice that is safe is A, B and Phase 7 (a world with the module on cannot be restored without tag 33); the scenarios, the golden and the exit gate follow in the same PR C, because a module ships with its golden.

## Hooks in core files (hook discipline, owner approval needed)

Each hook is ONE guarded call into a method defined in `technology*.go`, with no branch on module data and no arithmetic (architecture.md 2.10.1, 2.10.2, ADR 0010). Any hook not on this list is a defect and the plan changes before the code does.

| # | File and function | The call | Kind |
| --- | --- | --- | --- |
| H1 | `tick.go`, `positionsTail` | `if w.tec != nil { w.p83cTechnology() }` after `w.p83Regen()`; `positionList` gains `{"P8.3c", "TEC", line}`; `notBuiltPositions` loses `"P8.3c"`. The test tables `replayRows` and `stepSites` gain the row in the same change (Phase 5) | catalogue (2.10.3) |
| H2 | `phase_production.go`, `finishJob` | `if w.tec != nil { w.tec.count(region, c.recipe, c.executions) }` | named `phase_*.go` call |
| H3 | `production.go`, `effectiveAt` and `effectiveFor` | a new `tech int64` argument; `effective` passes `production.BP` | (a) a method of a core concept file |
| H4 | `strategy_view.go:235`, `phase_production.go:307`, `production_report.go:184` and `:202`, `skills.go:303` | `c.prod.techFactor(region, id)` (or `w.prod.techFactor`) as the argument, a method defined in `technology.go` that returns `production.BP` when the module is off | (b) the callers of the fan-in term |
| H5 | `construct.go`, the construction sequence | `w.tec = newTechnology(&w.cfg, ...)`, `w.prod.tec = w.tec`, `n.TechFields = techNames(c)`, the `{"technology", m.Technology != nil}` entry of `modulesOn`, the summary key `fields` | (d) construction calls |
| H6 | `world.go` | one pointer field `tec *technology`, one pointer field `tec` on `productionScratch` | catalogue |
| H7 | `validate.go`, `validate` | `v.validateTechnology()` after the last module's validator | catalogue |
| H8 | `validate.go`, the `yieldMax` closure of `overflow()` | `return v.withTech(rc.ID, top*MaxYieldBP/wholeBP)`, a method in `technology_validate.go` that multiplies by `MaxTechBP / 10 000` for a recipe in a field. `overflow()` is already at NLOC 84, CCN 27, 120 lines, so the call is preceded by a refactor-first commit (Phase 3) that hoists `yieldMax` into a `validator` method; the call lands in the hoisted method, which adds no branch to `overflow()`. Extends question 7 | (a) |
| H9 | `checkpoint_sections.go` | one row `{tag: 33, ...}` before tag 34; `notBuiltSections` loses tag 33 | catalogue (2.10.5) |
| H10 | `event.go`, `snapshot.go` (the `Tech` field only), `errors.go`, `bounds.go`, `random.go`, `hash.go`, `config.go`, `value.go`, `config/mapping.go`, `internal/logkeys` | appends only, in module order | catalogue |
| H11 | `snapshot.go`, `Snapshot` | `w.snapshotTechnology(&s)` after `w.snapshotContracts(&s)` (`snapshot.go:370`), a method in `technology_report.go` that returns at once when `w.tec` is nil; `Region()` (the second path, `:435`) gets no call because `RegionState` carries no technology rows | (c) the snapshot fan-in call, as `snapshotMigration` and `snapshotContracts` |

No audit hook (`auditModules` stays as it is: the module has no account). No hook in `fanin.go` (no per-agent state, no removal step, no relocation refusal, no filler, no checker), none in `endStep` (TC7), none in `agent.go`, `welfare.go`, `strategy_default.go`, `estates.go`, `phase_exit.go`, `phase_trade.go` or `internal/clearing`.

## Shared files other Stage 3 modules also touch (merge conflict forecast)

Appends are in module order and never inserted, so a conflict is textual (adjacent lines) and resolved by keeping both. The starred files are the likely ones.

| File | TEC append | Also touched by | Note |
| --- | --- | --- | --- |
| `config.go`* | `Modules.Technology`, `Technology`, `TechField` | every Stage 3 module | the `Modules` struct end |
| `hash.go`* | `pairsTechnology`, id 24 in `pairsLaterModules` | every module | between ids 19 and 25 |
| `event.go`* | two events, `Names.TechFields` | every module with events | appended at the end of each list |
| `event_test.go`*, `render_test.go` | two names deleted from `notBuiltEvents` and the render list | all | |
| `snapshot.go`* | `Snapshot.Tech`, one `snapshotTechnology` call (H11) | SET, STO, EMP, SAV, ORG, CAP | after the earlier modules' fields |
| `tick.go`* | the P8.3c entry, the call, `notBuiltPositions` entry deleted | LND (P8.3a), CAP (P8.3b), RSK (P8.1a), INS (P8.1b), SAV | the same tail of `positionList`; `positionsTail` may need the CCN split of plan 2-14 O7 once several append |
| `construct.go`* | `newTechnology`, `namesOf`, `modulesOn`, summary | every module | |
| `world.go` | `tec` fields | every module | |
| `validate.go`, `errors.go`, `bounds.go`, `defaults.go` | one call, the hoist of `yieldMax` and one call in it, three sentinels, eleven bounds, a comment | every module | `overflow()` is shared: whoever touches it next meets the hoist |
| `checkpoint_sections.go`* | the row for tag 33 | every module with a tag | tag 33 is the last built row before 34 |
| `production.go`, `phase_production.go`, `strategy_view.go`, `production_report.go`, `skills.go` | the `tech` argument and the counter call | STO (start pledge), CAP (no_facility), EMP (hired request), SKL (already merged) | `phase_production.go` and `production.go` are the files others edit most; land PR B early |
| `random.go` | `siteTechnology` | RSK (D13 to D15), PLC (D17), EMP | each appends its own constant |
| `config/mapping.go`, `internal/logkeys` | the tables and three keys | every module | |
| `scripts/ci/coverage-thresholds.txt` | `internal/technology 90` | every module with a new `internal/` package | one line; PR A |
| `doc.go`*, architecture.md 2.6, AGENTS.md Layout*, rule files with a RULE-TAG | the glossary, two continuation files, a fresh tag | every module that adds a file | the tag line always conflicts: the supervisor applies these edits |
| `registry_test.go` | three `notBuilt` entries deleted, `Modules.Technology` added to `exemptFields` | every module | |
| `poshash_internal_test.go`, `observe_internal_test.go` | one `P8.3c` row each (`replayRows`, `stepSites`) | every module that adds a position | named in plan-3-sto.md and plan-3-skl.md as well; append-only rows |
| `otelobserver/events_test.go`, `adapter_test.go` (`ignoredEvents`), `testdata/all_events.txt` | the two event names | every module with events | the adapter code is not touched; the metrics are deferred |
| `testdata/chains/overlay/` (PR A), `expected/`, `bands/`, `testdata/golden/technology/` (PR C) | new files | STY, CLB | |
| `testdata/golden/all/`, `testdata/checkpoints/`, `all_pack_test.go` | **none** | | the all layer moves once at Stage 3 close |

## Cross-plan contract

Consumes (identifiers as the tree has them; re-check at implementation):

- `production.Effective`, `yieldTable.effectiveAt`, `effectiveFor`, `productionScratch.abundance` (the pattern of the `res` pointer) <- Stage 1, plan 2-10.
- `finishJob`, `completion.executions` <- Stage 1 (plan 1-5).
- `Graph.Distance`, the effective graph and edge statuses <- REG.
- `checkpointSections`, `forkAtN`, `forkCase`, the section table test <- plan 2-2.
- `ordering.Finalise`, `ordering.StreamKey`, the site table of `random.go` <- ADR 0008, plan 2-1.
- `Population.replacementOf`, `resourcePlan`, `endowment.Abundance` for the warnings <- POP, RES.
- `w.enter(name, module, on)` and the call error order <- INT.
- The overlay and golden helpers of POL, STO and SKL (`checkModuleGolden`, `renderGolden`, `scenarioSeedCount`, `gateFPrefixes`) <- plans 3-pol, 3-sto, 3-skl.

Provides:

- `techFactor` and the cells -> CAP (its start path), EMP (the hired-job factor composes with `tech`), SAV (`net_r` reads `E` through the view; no code), RSK and others (no dependency).
- `Snapshot.Tech`, `TechnologyAdvanced`, `TechnologyAdjusted`, `Names.TechFields` -> IND (a `Yield` preset), `otelobserver` (later), consumers.
- `AdjustTechnology` -> consumers, STY and EXP scenarios; a policy actor has no lever on a gain (spec-policy.md 6.3.3).
- `overlay/technology.toml`, `technology-four.toml`, `testdata/golden/technology/` -> STY (TEC-S1 to S7), CLB (the knob rows).
- The rule "a module that keeps a key derives it from the effective seed" -> any later module with a keyed hash.

## Dependencies on other Stage 3 modules

TEC needs PRO, REG and CAL only (roadmap), all merged; it needs no other Stage 3 module and can merge before or after any of them. Soft links, none blocking: **EMP** (the hire form's `HireYieldBP` composes with `tech` when EMP builds it), **CAP** (a build or repair recipe in a field counts executions and reads no yield, FR-CAP-011; CAP's start path must pass the same term), **SKL** (a trainee's executions count in full; the `traineeRequest` site is in `skills.go` and is a hook here), **STO** (its start pledge edits the same lines of `phase_production.go`), **POL** (a jurisdiction has no lever on a gain), **SAV** and **LND** (read the effect through `E`, no rule), **POP** and **RES** (read only by the warnings of 10.3 and 10.4 when those modules are on). Merge order does not matter for correctness; the shared-file table says where conflicts fall.

## Implementation notes, PR A (Phases 1 to 3)

- `internal/technology` has no `products.go`: the products of spec 9.5 use `internal/checked` or 128-bit `math/bits` at the call, and `TestProducts_AtBounds` (root) and `TestLearned_ErrorsAndBounds` check them at the bounds. `Weight` and `Frontier` have no overflow to report (the 128-bit product is divided by a value above the weight), so they return no `ok`.
- `Step` takes one `Input` (cells in region-major order, counters, trend carries, the fields as primitives, a `DistanceFunc` by region index and the key), writes its new state only on success and returns one `Result` per cell, which carries the identity of 4.6.3.
- The key is `technology.Key(seed, site)`; the site constant `siteTechnology` is appended to `random.go` in PR B (Phase 4), as the plan has it.
- A recipe named twice in one field is `ErrDuplicateID` at that list entry (the spec is silent; the hash would otherwise tell two spellings of one behaviour apart).
- The hoist of `yieldMax` leaves `overflow` at CCN 27 and NLOC 84 (111 lines, down from 120): an existing finding, as the plan says.
- `Modules.Technology` is listed in `allPackNotYetOn` (all_pack_test.go), as Storage is: the all-on layer moves once at Stage 3 close.

## Codacy hygiene (every phase)

The Codacy gate fails on any new issue of level Info or higher (`phase-3-owner-questions.md`: PR #62 failed on 12 MD032 findings, PR #63 on two MD060 findings and one Lizard CCN 15). So, before a commit that opens or updates a PR, in every phase:

- **Markdown.** Run `npx markdownlint-cli2` on every changed `.md` (this plan, spec edits, notes, `doc/documentation_index.md`, the roadmap, a rule file) and fix every finding: a blank line around every list, heading, table and fenced block (MD032, MD022, MD031, MD058); table separator rows in the compact form `| --- | --- |` with the same number of columns as the header (MD060 and MD055/MD056); no double blank lines (MD012); no trailing spaces (MD009). `.markdownlint.json` already turns off MD013, MD037 and the code-block form of MD010.
- **Go.** Every new or changed Go function stays within Lizard CCN 10, NLOC 50 and 100 lines (`lizard -l go -C 10 -L 100 -T nloc=50 -w <changed non-test files>`). `.codacy.yml` excludes `*_test.go` from Lizard, but the SKL review split its over-limit tests anyway, so keep tests within the limits when it is cheap. Touching a function already at the limit needs a refactor-first commit: `positionsTail` (one more `if`), `overflow()` in `validate.go` and `construct.go`'s construction sequence are the likely ones. Measured at `d996592` (`lizard -l go -C 10 -L 100 -T nloc=50`): `overflow` NLOC 84, CCN 27, 120 lines (**over** all three limits; `validate.go` has 15 functions over them already, `warnings` CCN 85, so the file is an accepted finding); `positionsTail` NLOC 30, CCN 8 (one more `if` gives 9, within the limit, but other modules append to it, R3); `pairsModules` CCN 9; `pairsLaterModules` CCN 7; `namesOf` CCN 6; `finishJob` CCN 4; `modulesOn` CCN 3; `structural` small. So `overflow` takes the refactor-first commit of H8 and `pairsModules` is the one to watch in `hash.go` (the plan adds the pair to `pairsLaterModules`, TC1).
- `gofmt -l .` empty, `go vet ./...` and `staticcheck ./...` clean before each push.

## Gate economics

- Full gates (`scripts/ci/check.sh` with `R_MODE=full`, `full.sh` for gate F) run **once per PR, on the final head**, not per commit. Between pushes run `go test -short -race ./...` and the tests of the package being edited.
- Run gates with `GOMAXPROCS=8`, and **one gate run at a time on the host**: the root test binary under coverage grows to 20 GB and more (POL PR B lost three gate F runs to earlyoom at 19, 28 and 38 GB while other sessions shared the host). Check that no other gate is running before starting one.
- A scenario or fork test keeps counters, hashes and one snapshot at a time, never a snapshot per seed (the cause of the 38 GB run), and uses `scenarioSeedCount()`.
- `examples/realm` `TestServe_*` can time out under `-race` when several runs share the host; run it alone if it fails under load before concluding anything.
- `gomu` only in Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4` and per file; never on the host.

## Testing

Testing entry (testing.md 11) for the PRs: the requirement table above names the lowest-level test of each ID; the rules of spec 4 are unit tests of `internal/technology` (the large majority), validation, hash and loader are root and `config/` tests, the step, the counter, the production term, the events and the restore need a world and are integration tests (skipped under `-short` where they run more than a few ticks), S148 to S154, the module-alone golden and the fork test are the end-to-end handful. Golden files: `off/`, the pinned hash and `testdata/checkpoints/` unchanged (a diff there is a failure, not a golden to regenerate); `testdata/golden/technology/` is new, recorded once, never regenerated to absorb a failure; `golden/all/` untouched. Registry and catalogue: `Modules.Technology`, `Technology`, `TechField` leave `notBuilt`; the two events leave `notBuiltEvents`; `P8.3c` leaves `notBuiltPositions`; tag 33 leaves `notBuiltSections`. Fuzz: `FuzzStep`, `FuzzLearned`, `FuzzLoadTechnology`, `FuzzValidateTechnology`, `FuzzDecodeTechnology`. Each PR description records the gate exit markers and the deliberate-break pass, and says that a `Fix:` test, if any, was seen to fail first.

## Observability

Observability entry (observability-and-logging.md 9).

1. **New failure path:** configuration errors are returned at construction (`ErrTechShape`, `ErrTechRecipeOverlap`, the reused sentinels); `AdjustTechnology` returns its errors and logs none. Five conditions are survived and warned at construction only (`field_inert`, `field_unused`, `diffusion_no_contact`, `technology_past_maximum_yield`, `trend_outpaces_schedule`), each a `Warn` through the existing configuration-warning line with `rule` and `path`, bounded by the configuration. No `Warn` and no `Info` in the tick (FR-TEC-037).
2. **New state transition:** a cell's gain changes at P8.3c (`TechnologyAdvanced`, per cell in which anything accrued, class A, bounded by regions times fields) and at a call (`TechnologyAdjusted`, per changed key, class C). No map, no pointer, fixed-point integers; the identity of the amounts is a test.
3. **New decision point:** none in a strategy or the clearing; the effective yield a strategy reads includes the gain. A `Debug` line "technology advanced" per cell and "technology adjusted" per changed key, behind `Enabled`, with `region`, `field`, `gain`.
4. **New loop:** P8.3c is a pass over regions times fields (plus the distance reads of diffusion); events and the counters that exist only to build them are under `w.obs.supplied`; `TestStable_TechnologyAllocNoOp` proves the no-op path allocates nothing.
5. **Changed event, metric or attribute:** none changed; additions only. The four `economysim.technology.*` metrics and the attribute `economysim.field` are in the rules' catalogue already; the adapter feed is deferred (question 6), so no metric attribute is added to a running series here (rule 5.4.2: none).
6. **New log keys:** `field`, `fields`, `gain` exist in rule 3.3.2; the constants are added to `internal/logkeys`.
7. **`Error` or `Info`:** `Info` only in the existing construction summary (the key `fields` and `technology` in `modules`); no `Error`.
8. **Counterparts:** the cap is counted by `ClampedBP` in the event and the snapshot rows give the gains, so every bound that acts is visible; no `Warn` needs a counterpart.
9. **Non-interference:** the module draws nothing; a run with and without observer and logger is identical (`TestNonInterference_Technology`).

## Risks and open items

| # | Item | Owner |
| --- | --- | --- |
| R1 | The spec text on the key is stale (TC3). If the owner prefers a saved key, E3, E5 and TC10 change and a reseed needs the re-derivation rule of 4.4.6 | TEC owner (question 1) |
| R2 | The five yield-reading sites change signature together (H3, H4), and the new position changes two test tables with it (`replayRows`, `stepSites`, Phase 5). One site missed means a gain the anchor sees and the start does not, or the reverse; `TestGain_SameAtEveryPosition` and a text search for `effectiveAt(` and `effectiveFor(` callers guard it | TEC owner |
| R3 | `overflow()` is far over the Lizard limits (CCN 27; H8 hoists `yieldMax` first); `positionsTail` is at CCN 8 and several Stage 3 modules append to the tail (P8.1a, P8.1b, P8.3a, P8.3b). A refactor-first commit of the tail may be needed, and the order of merges decides who pays | tick.go owner |
| R4 | Every reference pack collapses (N-352). Experience accrues from the first ticks and the gain never falls without a call, so the golden is not blind, but a claim such as "growth raises the population" cannot be shown on `one`, `two` or `three`. S149 and S151 report and assert only invariants | STY owner |
| R5 | The cell index `c` of the discovery hash follows the canonical order of cells that exist (outside nodes have none); renaming regions so their order changes moves the hits. The spec states it (4.4.2); a test pins the order | TEC owner |
| R6 | `diffusion_no_contact` is checked at construction only; a closed edge later can silence diffusion with no warning. The spec accepts it | TEC owner |
| R7 | `technology_past_maximum_yield` and `trend_outpaces_schedule` read RES and POP internals (`resourcePlan`, `replacementOf`); a change to either breaks the warnings silently. The tests use the worked examples 16.3 | TEC owner |
| R8 | The golden of `three` is large and a new file; gate F time grows by the scenarios S148 to S154 (ten seeds each of the harness ones). Measure on the first run and record it in the PR | TEC owner |
| R9 | A restored world with the module on needs tag 33: until PR C lands, a world built on PR B cannot be checkpointed. PR B must not be released alone (it merges to `phase-3`, not `main`) | supervisor |
| R10 | The all-modules set, `allLayers` and the format-1 fixtures are untouched, so the all-on reference never exercises TEC until Stage 3 close; the module-alone golden and the fork test carry the proof until then | supervisor |

## Questions only the owner can answer

Each has a recommendation; the plan decides none of them. The questions file is `/tmp/claude-1000/dp3-tec-questions.md` (outside the repo, not durable; this list is the durable copy). Audit items that depend on an answer say so in place and do not pre-empt it: E3, E5 and TC10 on question 1; E1 on question 2; the PR sizes and the overlay move on question 3; E6 on question 4; E11 and Phase 8's pack on question 5; the otelobserver lists on question 6; the H8 hoist and H11 on question 7.

1. **The key of discovery:** derived from the effective seed and not saved (TC3; spec-world.md and spec-checkpoint.md 6.8 already say so, spec-technology.md and the tag 33 row do not), or saved in the section? Recommended: derived and not saved; E3 and E5 rewrite the stale text.
2. **Continuation files:** list `technology_validate.go` and `technology_report.go` in architecture.md 2.6 and AGENTS.md now (a fresh RULE-TAG), or only `technology.go`? Recommended: list both now; a name unused at the close of PR C is removed there.
3. **PR split:** three pull requests (A: edits, pure rules, configuration; B: behaviour; C: checkpoint and proof), two, or one? Recommended: three, so the configuration appends land early.
4. **Hash inner ids** (TC2: `Fields` 1; `TechField` `ID` 1 to `Reach` 10, `Label` none) added to spec-config.md 8.2.2 as a spec sentence: confirm? Recommended: yes.
5. **Pack of the module-alone golden:** `three` with a second overlay `technology-four.toml` (all four mechanisms; `one` has one region and cannot diffuse), instead of `one` with the learning-only overlay of REF 4.5? Recommended: `three` with the second overlay, fall back to `five` if the probe records no discovery or catch-up.
6. **Metrics:** the four `economysim.technology.*` instruments are in the rules' catalogue; build the adapter feeds in this module, or defer them to the later `otelobserver` PR as for SKL and POL? Recommended: defer; only the event names go into the adapter's test lists now.
7. **Hooks H1 to H11:** approve, in particular the new `tech` argument of `effectiveAt` and `effectiveFor` (five callers, one in SKL's `skills.go`), the `snapshotTechnology` call (H11) and the one call in `yieldMax` of `validate.go`, which the audit showed needs a refactor-first commit (a hoist of `yieldMax` out of `overflow()`, itself far over the Lizard limits) or else acceptance as an existing finding. Recommended: approve as listed, with the hoist.
8. **Counters cleared at P8.3c instead of P8.6** (TC7, no `endStep` hook; same behaviour): accept the spec wording change in 4.1 and 6.1.2? Recommended: yes.
9. **Scenario scope:** S148 (inert equals off) and S153 (checkpoint and reseed) asserted over seeds, S149 to S152 and S154 report-only, as spec 14.4 says; band values stay undeclared. Recommended: as the spec says.
10. **Notes:** accept that this plan read the index, N-214 and the ALL notes that bind code but not `doc/notes.md` in full, and that Phase 1 reads it in full and quotes the tag before any spec edit? Recommended: yes, as for SKL and POL.
