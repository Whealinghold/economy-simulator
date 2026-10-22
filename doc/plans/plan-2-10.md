# Plan: Stage 2, row 10: SUB, subsistence.go and internal/ownuse

**Status: plan written (2026-10-22), not started. Written against `plan-2-0.md`; Stage 1 is the signed-off slice (`6dd4f86`).**
Rule tags read (all three files in full, as `AGENTS.md` asks): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1 (the same as the contract's; none differs).
Specs and requirements implemented: [spec-subsistence.md](../specs/spec-subsistence.md) in full (FR-SUB-001 to FR-SUB-051, NFR-SUB-001 to 003); the SUB parts of [spec-config.md](../specs/spec-config.md) 3.1, 8.2.1, 8.2.2 and FR-CFG-077; [spec-checkpoint.md](../specs/spec-checkpoint.md) 4.2 tags 3 and 19; [spec-observability.md](../specs/spec-observability.md) 4.1 (`OwnUseChanged`), 5.3.4, 7.3 rows 14 and 15; [spec-world.md](../specs/spec-world.md) 6.4 (P2.4, P2.8). Also read: [spec-resources.md](../specs/spec-resources.md) 5 and 10, [spec-agents.md](../specs/spec-agents.md) 6.4, 7.1, [spec-welfare-and-wages.md](../specs/spec-welfare-and-wages.md) 2.1, 4.3, [spec-production.md](../specs/spec-production.md) 4.2, 9.3, [spec-viability.md](../specs/spec-viability.md) 3.1, 4.2, 10.3.
Notes read: the index; N-121, N-228, N-268, N-244 (tagged SUB), N-118, N-119 (SUB's open questions, by their areas PRO, WEL, SPN, EST), N-314 with its updates of 2026-10-21 and 2026-10-22, N-092, N-075, N-243 (cited), and N-016, N-017, N-027, N-103, N-126, N-163 (ALL: read, not touched, left open; they add no field, word or constant here), N-128 and N-217 (ALL: applied, the hosted fields are written once in tag 3 by plan 2 and this plan fills their value, D403). The plan changes no spec and no note, so it does not quote the notes tag; the proposals are in section 9.
Decision-ID block: **D400 to D419** (all used).

## Scope

This plan builds the subsistence module: an own-use recipe is a source recipe named on a list, a household takes it up in the own-use mode and leaves it on its own stock, and the default `Work` acts on the mode.

| In scope | Where |
| --- | --- |
| `SubsistenceConfig`, `RoleOwnUse`, `Modules.Subsistence` (the 8th pointer, hash id 8), `DefaultSubsistence()`, the bounds `MaxOwnUseRecipes`, `MaxOwnUseTicks`, `MaxRetryJobs`, the four sentinels | `config.go`, `hash.go`, `defaults.go`, `bounds.go`, `errors.go`, `validate.go`, `config/mapping.go` |
| The module: effective lists, covered sets, the state (`ownUse`, `ownUseJobs` by agent slot), the flag step P2.4, the view filler, the start/completion/removal/role-change hooks, the receipts, the counts, `OwnUseChanged` at P2.8, the two `Warn`s, the snapshot rows, section 19 | `subsistence.go`, and `subsistence_validate.go`, `subsistence_report.go` only if Lizard forces the split (new, D402, E5) |
| The pure rules: `Cover`, `Cost`, `Afford`, `Secure`, `Step`, `Cov`, `FloorBP`, `Pick`, `Ramp`, `RetryDue` over primitives | `internal/ownuse/` |
| The default `Work` in the mode, the own-use mark on a recipe view, the four view fields | `strategy_work.go`, `strategy.go`, `strategy_view.go`, `strategy_default.go` (behaviour-neutral seams, D407, D408) |
| `roleRow.ownUse`/`jobs`, `admit`, the residue layout, the welfare window, the tradable items of the own-use outputs | `role.go`, `production.go`, `validate.go`, `phase_production.go`, `agent.go`, `phase_exit.go` |
| `OwnUseChanged`, `RoleRow.OnOwnUse`, `RegionAgent.OwnUse`, the log key `own_use` | `event.go`, `snapshot.go`, `spawner.go`, `internal/logkeys` |
| `overlay/own-use.toml` (D416), scenarios S56 to S60, `testdata/golden/subsistence/`, the registry, catalogue and position rows | `testdata/`, tests |

| Out of scope | Owner |
| --- | --- |
| The endowment, the ration at P2.6, `Draws`, `ReasonNoEndowment`, the `Endowments` view group | plan 9 (this plan reads its seams, D413) |
| Relocation, the departure of a timed move, the journey, the trip (the `offStage` predicate's extensions) | plans 3, 14, 12 (this plan gives them the drop and the stub of the predicate) |
| The metrics `economysim.household.own_use` and `economysim.own_use.units` and their adapter | plan 22 (this plan emits the event they read) |
| `Direct`, `AgentState`, the per-agent query of `OwnUse`, `OwnUseJobs`, `Cover`, `Secure` | plan 17 (this plan gives the accessor) |
| The merchant side of N-121 (no trip while in the mode) | plan 15 |
| `OwnUseBound`, `OwnUseExempt`, `no_title`, `rent_funds`, `no_room`, the trainee, rung 1 skipping a household with a running own-use job (open question 4), a second job slot (open question 1), a provenance on lots (open question 2) | later stages; D419 |
| Home baking from the household's own grain (an own-use recipe with inputs) | not built; D401 and Risks say what it costs |

## Decisions

- **D400.** Hash id of `Modules.Subsistence` is 8 and its checkpoint tag is 19, whatever the build order (spec-config 3.1, 8.2.1; spec-checkpoint 4.2). The event position is P2.8, before P2.8a (spec-world 6.4; spec-observability 4.1). The flag step is P2.4, after the endowment snapshot P2.3 (plan 9) and before P2.5. Where build order differs from spec order, the append rule holds by explicit position: the `event.go` declaration, the `RoleRow.OnOwnUse` field and the `Modules` field are appended after what is built before them, with the hash id, tag and position taken from the spec (the `OnOwnUse` table order of spec-observability 5.3.4 is a documentation order, not a field position).
- **D401.** The source-only rule of spec-subsistence 3.4.1 is **kept**: an own-use recipe has no `Inputs` and no `Requires` (`ErrOwnUseNotSource`, FR-SUB-004). This plan does not let an own-use recipe bake from the household's own grain. Source: spec-subsistence 3.4.1, SUB-D4, open question 3; N-121 and the 2026-10-21 and 2026-10-22 updates of N-314 (they say SUB as written cannot express home baking). The decision to change it is the owner's (Risks 1); the cost is stated there. What this plan does instead: the overlay of D416 gives `prepared` a source stand-in so that `Fed` is reachable by the mode alone.
- **D402.** Types `SubsistenceConfig` and `RoleOwnUse` are in `config.go` (architecture 2.6 puts every section struct there; spec-subsistence 22 says `subsistence.go`; architecture wins). The README scope `Subsistence` resolves to `SubsistenceConfig` in the registry test as `TenureConfig` does (spec-config 3.7.1, last paragraph). New continuation files `subsistence_validate.go` (VAL) and `subsistence_report.go` (events, logs, snapshot) keep each file near 500 lines (N-243; the split is conditional on the Lizard measurement of Phase 4 and 6, and a file that stays within the limits is not split); a module's code in `strategy_work.go` and the view filler is lawful through plan 5 E1 and ADR 0010 (the hook clause for the DefaultStrategy files), and the code in `strategy_work.go` stays guarded calls into `subsistence.go`; E5 owes their entry in architecture 2.6 and `AGENTS.md`, only if the split is made.
- **D403.** Per-agent state is `ownUse []bool` and `ownUseJobs []int64` in the module, indexed by agent ID and grown like `welfareScratch.at`, never fields of `agent` (architecture 2.10.1; N-244). A removal zeroes the slot. `hashWorld` walks it by reflection, so the position hash sees it.
- **D404.** The role table gains `roleRow.ownUse` (the effective own-use list, ascending) and `roleRow.jobs` (the ascending union of `recipes` and `ownUse`; the same slice as `recipes` while the module is off), filled by the constructor of `role.go` at the marked place of the tradable items (N-228). `admit`, `residueLen`/`residueOffset` (agent.go, phase_exit.go, phase_production.go) and the scratch sizing read `jobs`; the view, the cost anchor (`w.runnable`) and the spawner read `recipes`. So an own-use recipe never sets an anchor (FR-SUB-027) and is requested only through `jobs` (FR-SUB-026). The residue layout is "recipe of the role or of its effective own-use list" (spec-checkpoint 4.2 tag 6). The non-discard outputs of the own-use list join the tradable set through `tradableItems` (plan 5).
- **D405.** The covered set is derived once per role into the module (`covered []CommodityID`, canonical order): the upkeep commodities of the role (any region) that are a non-discard output of a recipe on `ownUse`. For an agent the terms run over the covered commodities whose effective upkeep in its region is above 0 (spec-subsistence 3.3.2). A role with a non-empty list and an empty covered set draws `own_use_no_upkeep` and is never evaluated.
- **D406.** The flag step is a root loop over household agents in ascending ID that converts inline to `internal/ownuse` primitives: `Stock_c` from `ledger.Holding`, `Money` from the account balance, `Price_c` from the market report of the agent's region (the report is seeded, `newMarketTable`), `U_c = weight * u_c` checked, `Hunger` from the agent. It writes `ownUse`, `ownUseJobs` and the counts; it draws nothing and reads no other agent (FR-SUB-015, 017, 035). It skips weight 0 (estates), and an agent for which `w.offStage(a)` is true (D411).
- **D407.** The view gains flat fields `OwnUse`, `OwnUseJobs`, `Cover`, `Secure`, `ExitTicks`, `RetryJobs` and `RecipeView.OwnUse`. The two tunables are not in the spec's group (Phase 1, E1): a pure `Work` needs them. The module filler (fan-in `fillAgentView`) sets the fields and appends the marked own-use recipes after the role's, in ascending `RecipeID`, using `appendRecipe` extracted without change from `fillRecipes`; `newViewScratch` sizes from `jobs`. `Cover` and `Secure` are recomputed by the filler from the stock and prices at the moment the view is built (the pure functions of `internal/ownuse`), `OwnUse` and `OwnUseJobs` come from the state, so the Trade view built at P3 after consumption carries the current cover. Own-use recipes are not "makes" and set no anchor (FR-SUB-027): every reader of `v.Recipes` outside the mode (`plannedRecipe`, `makes` in `strategy_default.go`, `makesIn` in `strategy_view.go`) skips a recipe with `OwnUse` set; owed edit E11 states it in spec-subsistence 5.4 and FR-SUB-018. With the module off every field is zero.
- **D408.** `workReport` gains one branch. Outside the mode a recipe with `OwnUse` set is skipped before the halt, the eligibility and the decline reason, so the report and the counts are those of today (FR-SUB-048). In the mode the share is 10 000, the halt does not apply to the own-use choice, the recipe is `ownuse.Pick` over the marked recipes with `MaxExecutions >= 1`, the executions are `ownuse.Ramp`, and when `RetryJobs > 0 && OwnUseJobs >= RetryJobs` the normal loop over the unmarked recipes (the halt applies to it, spec-agents 7.1.9 item 5) runs first and a chosen role recipe wins (`ownRetried` when an own-use recipe was runnable). No runnable own-use recipe sets `ownBlocked` and the normal path runs as if out of the mode (FR-SUB-023, case a). `workReportBuf` gains `ownBlocked` and `ownRetried`; `countWork` reads them through one guarded call.
- **D409.** The counts of `OwnUseChanged` per (region, role) are integer adds made always (the two `Warn` runs need them with no observer, as `floorWarns` does): `entered`, `left` at P2.4; `blocked` (a) and `retried` at P2.5 from the report; `started` and `blocked` (b) at P2.7; `output` per commodity at completion. The event is built only under `w.obs.supplied`. A household is counted in `blocked` once (cases (a) and (b) exclude each other: (b) needs a request).
- **D410.** Welfare. `markWorking` takes `w.ownUseOf(a)` (job recipe in `roleRow.ownUse`) at P2.1 and P2.7 in place of `false`; `workingTick` and the ring already exclude own-use ticks and the receipt from the wage (plan 1-6). `finishJob` (P2.1 and the one-tick job of P2.7) appends `(slot, commodity, created)` to a module list for an own-use job; `p63Welfare` prices the list, right after `w.valuePrices()`, with `w.oc.valued` (the price of `cost_t`, FR-SUB-028) into `flows.ownReceipt` through `addOwnReceipt` (defined in `subsistence.go`). `effectiveWindow` joins the longest own-use `Duration` (spec-welfare 2.3.3).
- **D411.** Drops. Removal: a step of `removeAgent` zeroes the slot. Role change: the module's `roleChanged` is called from plan 5's fan-in `agentRoleChanged` (at `takeRole`; no edit of `execSwitch`) and resets `ownUseJobs`, keeping `ownUse` (the job is already abandoned with `role_changed`). Relocation and the departure of a timed move: one guarded call in `moveAgent` (`regions_relocate.go`, plan 3 D268, the one relocation path) zeroes both (FR-SUB-031). Weight change: nothing. Off stage: this plan creates `func (w *World) offStage(a *agent) bool` in `world.go`, returning false, and its single call site in the flag step (which skips an agent for which it is true); the call sites at `workAgent` and `tradeAgent` are plan 12's (D447), and plans 12 and 14 extend the predicate with the traveller and the journey. The trainee check waits for SKL (Stage 3).
- **D412.** Validation lives in `validateSubsistence`, called by the module seam of the validator in the order of the `Modules` row. Order: bounds (`ErrOutOfBounds`), `ErrOwnUseBand`, `ErrDuplicateID`, `ErrUnknownRole`, `ErrUnknownRecipe`, then per effective list `ErrOwnUseNotSource`, `ErrOwnUseNotSited`, `ErrOwnUseOverlap`. Warnings: `subsistence_empty`, `own_use_no_upkeep`, `own_use_exit_unreachable`, `own_use_slower_than_ladder` (reads `World.ChangePrivation`), `own_use_partial`, `own_use_trap`, `own_use_not_available` in `subsistence_validate.go`; `own_use_beats_market` is called from `viaModel.regions` with the autarky `SurplusBP` of the region (`res.SurplusBP`); the pooled figure is in the solver `Report.RegionSurplusBP` only (E3).
- **D413.** Seams read from plan 9: the refusal of a start whose grant is 0 reaches the `res.k == 0` branch of `startJob` with `ReasonNoEndowment`; this plan adds one guarded call there (`blockedStart`) and counts only that reason (E7). The own-use recipe's `MaxExecutions` and `Effective` already include the abundance in the view (spec-resources 5.5.1). The rule that a recipe with `Draws` needs `Modules.Resources` is plan 9's; this plan tests it on an own-use recipe (FR-SUB-050). Plan 9 D392 fills the `Endowments` group and `endowment_unused` for the "effective own-use list" but is built first and cannot read `roleRow.ownUse`; so this plan extends `fillEndowments` and the `endowment_unused` rule to take `roleRow.ownUse` under `w.sub != nil` (spec-subsistence 3.6, spec-resources 13.3), with `TestEndowments_OwnUseDrawsInGroup` and `TestWarnings_EndowmentUnusedCountsOwnUse`.
- **D414.** Logs. Debug (behind `Enabled`): "own use entered" (`reason` `unfed` when `Hunger` decided it, else `short_cover`), "own use left" (`cover`), "own use retry", "own use blocked" (`reason`: the production reason). Warn "role on own use" and "own use blocked" (with `reason`, the commonest production reason) at the 10th tick of a run and every 10th after, per (region, role), runs kept in the module (`runOn`, `runBlocked`) and saved in section 19. New key `own_use` in `internal/logkeys` (row exists in rules 3.3.2).
- **D415.** Snapshot: `RoleRow.OnOwnUse` (households with the flag, appended after the then-last field, set in `roleRow` by one guarded call). `RegionAgent.OwnUse` (a core-type field added by a module: E1 item (f) of plan 5, which plan 5 owes; not an unnamed `spawner.go` touch) is set in `addAgentRow` (N-268, FR-SUB-051). The per-agent query reads `(*subsistence).agentFields(slot)` (plan 17).
- **D416.** `testdata/chains/overlay/own-use.toml`: a test double, not a design. Source recipes `own_staple` (2 `staple`), `own_prepared` (1 `prepared`, a stand-in for baking, D401), `own_fuel` (1 `fuel`), duration 1, `sited`, yield rows `all = 10000` in region `north`; `[modules.subsistence]` with `recipes` the three, defaults for the rest; `[modules.subsistence.role.visitor] recipes = []` (the closed external role). `Cov` is 10 000 for each (upkeep 2, 1, 1), which Phase 3 checks against `SurplusBP` of the rebalanced pack (plan 8) and `own_use_beats_market` silent. Variants (E of 6 667 and 13 333, `RetryJobs` 0, durations 3 and 8) are made in the tests by editing the loaded `Config` (`Yields`, `Modules.Subsistence`, recipe durations), which is plain data. A second overlay `own-use-draws.toml` (S59) adds `draws` to `own_staple` and stacks on plan 9's `overlay/resources.toml`.
- **D417.** Scenarios: S56 = SUB-S1, S57 = SUB-S2, S58 = SUB-S3, S59 = SUB-S4, S60 = SUB-S5 (N-092 update of 2026-10-10). Gates are R and F of D229 of plan 1. Under `-race` S56 and S59 run the reduced size (seeds of `scenarioSeedCount`, 100 agents of weight 1 and 4 cohorts of weight 50 in S59); F runs the full size.
- **D418.** Blocked reasons are staged: FR-SUB-023 lists `not_permitted`, `no_endowment`, `no_title`, `rent_funds`, `no_room`. Only `no_endowment` exists in Stage 2. The module counts the reasons that exist; the plan that builds each later reason appends it to `blockedStart` (E7).
- **D419.** Not built, with the reason: the spawner skipping rung 1 for a household with a running own-use job (the default stays as written, spec-subsistence 8.3); a second light job slot (needs a PRO change); a provenance on lots (N-119); the retry in ticks; a raised `Keep` in the mode (open question 12). Each is a spec open question and none has a requirement.

## Phases

Each phase ends green on `go test -short -race ./...` and F of D229 on the tests it adds; `testdata/golden/off/` and the pinned hash do not move in any phase (the proof is a test run at every phase exit).

### Phase 1: the specs are true (no code)

Goal: nothing below starts before these are applied by the supervisor. Edits owed:
- **E1.** `spec-agents.md` 6.4.1 (own-use row) and 7.1.9: the view carries `ExitTicks` and `RetryJobs` beside `OwnUse`, `OwnUseJobs`, `Cover`, `Secure` (a pure `Work` needs them); the group is flat on `AgentView`.
- **E2.** `spec-subsistence.md` 5.5 item 3, FR-SUB-020 and `spec-agents.md` 7.1.9 item 3: "greatest sum, compared as exact fractions" and `Cov` truncated (4.1, FR-SUB-039) disagree. Say which: the sum of truncated `Cov`, or the exact sum of `q*E/(Duration*u_c)`. Phase 2 does not write `Pick` before this lands.
- **E3.** `spec-subsistence.md` 4.2 and 13.3, `spec-viability.md` 10.3: Validate has the autarky `SurplusBP` only; the pooled figure exists in the solver report. State that `own_use_beats_market` compares with the autarky figure at Validate, and with the smaller of the two where the report has both.
- **E4.** `spec-world.md` 6.4 rows P2.4 and P2.8: P2.4 writes the flag; the two diagnostic runs advance at P2.8, because the blocked count needs P2.5 to P2.7.
- **E5.** Only if the Lizard measurement of Phase 4 and 6 forces the split (D402): `architecture.md` 2.6 (with a fresh RULE-TAG) and `AGENTS.md` Layout list `subsistence_validate.go` and `subsistence_report.go` as continuation files of SUB, before the split is made; otherwise neither is listed and the code stays in `subsistence.go`; `spec-subsistence.md` 22: `SubsistenceConfig` and `RoleOwnUse` are in `config.go`.
- **E6.** `spec-config.md` 8.2.2: `SubsistenceConfig` `Recipes` 1, `Roles` 2, `EnterTicks` 3, `ExitTicks` 4, `RetryJobs` 5; `RoleOwnUse` `Role` 1, `Recipes` 2; `Modules.Subsistence` 8.
- **E7.** `spec-subsistence.md` 5.5 item 6 and FR-SUB-023: the reasons other than `no_endowment` belong to modules after Stage 2 (D418).
- **E8.** `spec-subsistence.md` 5.6 and FR-SUB-010: the "away" test is one predicate (`offStage`, spec-world 6.1.3), extended by TRN and MIG; the trainee test waits for SKL.
- **E9.** `doc.go` glossary (supervisor, before any code uses the words, architecture 3.2): subsistence floor, own-use recipe, own-use mode, covered set, own-use cover, afford, secure ticks, own-use retry, coverage, `FloorBP`, own-use blocked; and the type names `SubsistenceConfig`, `RoleOwnUse` and `OwnUseChanged` (TestGlossary_DeclaredTypesNamed).
- **E10.** Verify plan 3's `moveAgent` (regions_relocate.go), plan 5's validator seam, `tradableItems`, `agentRoleChanged` and `pairsModules` shape, plan 12's adoption of `offStage` and plan 2's `checkpointSections` protocol (Risks 3).
- **E11.** `spec-subsistence.md` 5.4 and FR-SUB-018, `spec-agents.md` 7.1: an own-use recipe is not a recipe the role "makes" for the output cover, the planned recipe or the cost anchor.
- **E12.** `spec-subsistence.md` 13.2: "three new sentinels" lists four.

Exit check: the supervisor confirms E1 to E12; Phase 2 starts.

### Phase 2: `internal/ownuse`

Goal: the pure rules, standard library and `internal/checked` only, no floating point, no root import (NFR-SUB-002). Files `ownuse.go` (`Good{Stock, Upkeep, Price}`, `Cover`, `Cost`, `Afford`, `Secure`, `Params{Enter, Exit, Max}`, `Step`), `coverage.go` (`Cov`, `FloorBP`, `Pick`, `Ramp`, `RetryDue`), `doc.go`. Every product is checked; an overflow returns `ok` false and the root reads it as the saturated value (a cover of `Max`).
Tests (`Requirement:` IDs): `TestCover_MinOverGoods`, `TestCost_Sum`, `TestSecure_Saturates` (FR-SUB-011); `TestAfford_ZeroWhenHungry` (FR-SUB-012); `TestStep_EnterBelowEnter`, `TestStep_LeaveAtExitCoverOnly`, `TestStep_BandAndCounterReset` (FR-SUB-013, 014, 022); `TestCov_Truncation128`, `TestFloorBP_MaxOverCovered` (FR-SUB-039); `TestPick_GreatestSumTiesToLowerID` (FR-SUB-020, per E2), `TestPick_NeverZero`; `TestRamp_Formula`, `TestRamp_CohortSteps` (FR-SUB-021, SUB-D5); `TestRetryDue` (FR-SUB-022); `TestOverflow_ReturnsNotOK` (NFR-SUB-002); worked examples of spec-subsistence 16.1 (nine ticks, `Made` column excluded) and 16.2 (ramp 27, draws 54) as table cases. `FuzzStep` (properties: the mode never leaves with cover below `Exit`; `Secure <= Max`; no panic). Guard: `TestOwnuse_ImportsOnly` (the import guard of `import_guard_test.go` gains the package).
Exit check: coverage >= 90% for the package, `gomu` on it in Docker with `--cpus=4 --memory=16g --memory-swap=16g` and `GOMAXPROCS=4`, no surviving non-equivalent mutant; Lizard clean.

### Phase 3: configuration, validation, hash, the role lists

Goal: the module exists as data and is checked; it still does nothing at run time.
Files: `config.go` (types, `Modules.Subsistence`), `bounds.go` (8, 1 000, 1 000), `defaults.go` (`DefaultSubsistence`: 2, 3, 10), `errors.go` (`ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited`, `ErrOwnUseOverlap`), `hash.go` (`pairsSubsistence`, id 8, inner ids per E6, lists in ID order; the pointer written even when empty), `config.go` `canonical` (sort the default list and the rows by role, each list by recipe), `config/mapping.go` (`[modules.subsistence]`, `[modules.subsistence.role.<id>]`), `subsistence_validate.go` (only if the split is forced, E5; else in `subsistence.go`), `role.go` (`ownUse`, `jobs`, D404), `production.go` (`admit` over `jobs`), `errors_test.go` (`TestSentinels_AreDistinctAndMatch`) and `errors_mutation_test.go` (the four sentinels), `validate.go` (`effectiveWindow` join, `unused_recipe` counts an own-use use), the `viability_check.go` call for `own_use_beats_market`, `overlay/own-use.toml`.
Tests: `TestValidate_OwnUseNotSource`, `_NotSited`, `_Overlap`, `_Band`, `_Bounds`, `_UnknownRecipe`, `_UnknownRole`, `_DuplicateRow` (FR-SUB-004 to 008, FR-CFG-077); `TestEffectiveList_RowReplacesDefault`, `_EmptyRowMeansNone`, `_SortedAndBounded` (FR-SUB-002); `TestSubsistence_NoRegionKeyedField` (reflection over the two structs) and `TestSubsistence_RegionOffersByYieldRow` (FR-SUB-009); `TestWarnings_Eight` (one case each, plus the silent case on the overlay, FR-SUB-038); `TestValidate_DrawsNeedResources` (FR-SUB-050, on plan 9's rule); `TestHash_SubsistenceOnOff`, `_SubsistenceFieldIDs`, `_SubsistenceAfterLowerModuleIDs` (pair 8 sits after the lower ids in `pairsModules`), `_ListOrderIrrelevant` (FR-CFG-077; `TestHashConfig_ReferencePinned` unchanged); `TestAdmit_OwnUseBeforeAnyOtherCheck` (FR-SUB-026); `TestWindow_JoinsOwnUse` (FR-WEL-009); `TestViability_OwnUseOnlyRecipeNotAnActivity` (FR-SUB-037); `TestLoad_SubsistenceTables` (the loader, the layers merge by key); `FuzzValidateSubsistence` (no panic, a valid input validates twice the same); the registry rows (`Subsistence.*`, `RoleOwnUse.*`, `Modules.Subsistence`, with the scope alias).
Exit check: off golden and pinned hash unchanged; registry and catalogue tests with the SUB entries deleted from `notBuilt`; `own_use_beats_market` is silent on the rebalanced pack with the overlay (otherwise the overlay quantities change, not the code).

### Phase 4: state, the flag step, the view, the default `Work`, the hooks

Goal: a household enters and leaves the mode and works the floor; every hook is a guarded one-line call into `subsistence.go`.
Files: `subsistence.go` (state, `newSubsistence` called from `construct.go` under the pointer and `modulesOn`, `flagStep` as `p24OwnUse`, `fillView`, `noteStart`, `blockedStart`, `drop`, `remove`, `roleChanged`, `ownUseOf`, `agentFields`, `tradableItems` entry), `world.go` (`sub *subsistence`), `tick.go` (P2.4 call after P2.3's; `notBuiltPositions` loses "P2.4"), `world.go` also holds `offStage`, `fanin.go` (`fillAgentView` module call, `removeAgent` step, `agentRoleChanged` call into `roleChanged`), `regions_relocate.go` (one guarded call in `moveAgent`), `fillEndowments` and the `endowment_unused` rule (D413), `phase_production.go` (`workAgent`/`countWork` call, `startJob`, `finishJob`, `p21Complete` and `startJob` `markWorking` argument), `phase_exit.go` (`addAgentRow`), `strategy.go`, `strategy_view.go` (`appendRecipe`, sizing, `makesIn`), `strategy_default.go` (`plannedRecipe`, `makes`), `strategy_work.go` (D408), `spawner.go` (`RegionAgent.OwnUse`), test seam `replayRows` (`{id: "P2.4", on: sub != nil}`), `observe_internal_test.go` (the `stepSites` row for P2.4, after P2.3, or after P2.1 if plan 9 is not merged).
Tests: `TestFlag_EnterLeaveHysteresis`, `TestFlag_IndependentOfOthers` (FR-SUB-015: changes every other agent, the regional wage, other regions), `TestFlag_CohortEqualsUnits` (FR-SUB-016), `TestFlagStep_RunsAfterCompletionsBeforeWork`, `TestFlagStep_OrderIndependent` (FR-SUB-017), `TestFlag_Eligibility` (estates, weight 0, empty covered set, no upkeep, off stage; FR-SUB-010), `TestFlag_SkipsOffStage` (a traveller, once plan 12 lands), `TestFlag_UpkeepOnlyThroughExtraUpkeepLeftOut`; `TestView_CarriesOwnUseGroup` (FR-SUB-018, off: all zero), `TestView_TradeViewCoverRecomputed`, `TestRecipes_OwnUseNotPlannedNotMadeNoAnchor` (`plannedRecipe`, `makes`, `makesIn`; FR-SUB-027); `TestSubsistence_OffLeavesNoTrace` (FR-SUB-001), `TestModulesOn_Subsistence` (`Snapshot.Modules` holds "subsistence" only when `Modules.Subsistence` is non-nil), `TestSpawner_RegionAgentCarriesOwnUse` (N-268, FR-SUB-051), `TestAllocs_Work` (Pick and Ramp, NFR-SUB-001); `TestWork_ModeShareIs10000`, `_PicksBestCoverage`, `_RampForCohort`, `_RetryChoosesRoleRecipe`, `_RetryWithNoEligibleRoleRecipeKeepsOwnUse`, `_BlockedActsAsOutOfMode`, `_HaltDoesNotApply`, `_NoRecipeRoleRequestsOwnUse` (FR-SUB-019 to 025), `TestWork_NotInModeNeverOwnUse` (FR-SUB-048), `TestWork_NoInterruption` (FR-SUB-049), `TestWork_DirectedOverrides_ExternalEvaluated` (FR-SUB-034); `TestStart_CountsOwnUseJobs` (counter +1 on own-use start, reset on any other start); `TestDrops_RemovalRoleChangeWeightChange` (FR-SUB-032), `TestDrops_Relocation` (FR-SUB-031, through plan 3's export_test `RelocateForTest` seam, moved to the public `World.Relocate` by plan 17); `TestMode_DoesNotTouchHungerPrivation` (FR-SUB-033); `TestStream_SameWithModuleOnAndOffWhenNoOneEnters` (FR-SUB-035); `TestAnchors_OwnUseNeverSets` (FR-SUB-027); `TestRation_OwnUseNoPriority` (FR-SUB-036, with plan 9); `TestPositions_*` rows for P2.4 (the table test, the swapped-call test).
Exit check: `TestStep_AllocationsStayAtTheBaseline` still at its baseline with the module off, and `TestAllocs_FlagStep`, `TestAllocs_FillView` at 0 with it on (NFR-SUB-001); position golden for the off run unchanged.

### Phase 5: receipts and the wage

Goal: the welfare record and the observed wage treat an own-use job as D410 says.
Files: `subsistence.go` (`finishJob` hook, the receipt list, `addOwnReceipt`), `phase_outcomes.go` (`p63Welfare` first line), `phase_production.go` (`markWorking` argument).
Tests: `TestWelfare_OwnReceiptIsUnitsCreatedTimesCostPrice` (FR-SUB-028, 051: after the carried remainder, the reference price when never traded), `TestWelfare_FedByFloorReadsTenThousand` (the property of spec-subsistence 7.2: a household that makes exactly its upkeep reads 10 000 at any price), `TestWage_ExcludesOwnUse` (FR-SUB-029), `TestWelfare_OwnUseTickIsWorking` (FR-SUB-030, role with and without recipes), `TestWelfare_OneTickJobFromP27` (completion in the start tick), `TestWelfare_NoReceiptWhenOff`, `TestSubsistence_NoNewReasonAndAuditHolds` (FR-SUB-003).
Exit check: the welfare golden rows of the off run unchanged.

### Phase 6: observability

Goal: the event, the two `Warn`s, the snapshot field, the Debug lines.
Files: `subsistence_report.go` (only if the split is forced, E5; else in `subsistence.go`; counts, `p28OwnUse`, runs, logs, `roleRow` call), `event.go` (`OwnUseChanged`, `isEvent`), `tick.go` (P2.8 before P2.8a, `notBuiltPositions` loses "P2.8"), `snapshot.go` (`RoleRow.OnOwnUse`), `internal/logkeys` (`OwnUse = "own_use"` in the order of rules 3.3.2, after `drift`), `event_test.go` (`OwnUseChanged` leaves `notBuiltEvents`), `replayRows` (`P2.8`), `observe_internal_test.go` (the `stepSites` row for P2.8, before P2.8a).
Tests: `TestOwnUseChanged_OnlyWhenNonZero_AscendingOrder_NoMapNoPointer` (FR-SUB-040, 041), `TestOwnUseChanged_OutputInCommodityOrder`, `TestOwnUseChanged_LevelFromLastEvent` (a change of `OnOwnUse` always carries `Entered` or `Left`), `TestWarn_RoleOnOwnUse_Ticks10And20`, `TestWarn_OwnUseBlocked_Ticks10And20`, `TestWarn_AtMostOncePerCellPerTick` (FR-SUB-042), `TestLogKey_OwnUse` (FR-SUB-043, the metrics half is plan 22), `TestNonInterference_ObserverAndLogger` (FR-SUB-045), `TestSnapshot_OnOwnUse` (FR-SUB-046), `TestDebug_Lines` (behind `Enabled`), `TestAllocs_EventNoop` (construction with the no-op observer allocates nothing, NFR-SUB-001; observability item 9), `BenchmarkFlagStep` with `b.ReportAllocs()` (NFR-SUB-003).
Exit check: the catalogue test (`OwnUseChanged` row of 4.1, the fields in the order of the row) and the log-key test pass; off event golden unchanged.

### Phase 7: checkpoint

Goal: section 19 and the agent-record fields.
Files: `subsistence.go` (`saveSubsistence`, `loadSubsistence`, and the value source of `OwnUse`/`OwnUseJobs` in the tag 3 record that plan 2 D247 already writes, zero while the module is off), `checkpoint_sections.go` (one row for tag 19 appended to `checkpointSections`, tag 19 deleted from `notBuiltSections`), `checkpoint_restore.go` (`checkpointCross` entries), `checkpoint_fork_test.go` use.
Tests: `TestCheckpoint_SubsistenceRoundTrip` (flags, counters, both runs, sparse, one spelling: a run equal to its default is corrupt), `TestFork_Subsistence` (through plan 2's `forkAtN`/`forkCase` with `overlay/own-use.toml`, N in {0, inside a job, inside a season, after a call between ticks} and in the middle of the mode; bit-identical, events included; FR-SUB-044), `TestCheckpoint_AgentRecordWritesFlagOnce` (N-128), `TestCheckpoint_OwnUseNonZeroCorruptWhenOff` (R10: a non-zero `OwnUse` or `OwnUseJobs` with the module off is corrupt), `FuzzDecodeSubsistence` (nothing panics, bounded allocation: NFR-CHK-003). A checkpoint restored with the module off while the bytes carry tag 19 fails with plan 2's error.
Exit check: no all-modules fixture exists before plan 24; `testdata/checkpoints/` is not touched here (`format-1-one.*` is plan 8's).

### Phase 8: scenarios and goldens

Files: `scenario_s56_test.go` to `scenario_s60_test.go`, `testdata/chains/overlay/own-use.toml`, `own-use-draws.toml`, `testdata/chains/scripts/s56.toml`, `testdata/golden/subsistence/{one.txt,events-one.txt,one.positions.txt}`.
- **S56 (SUB-S1).** Pack `one` with S04's staple-shortage script and a staple outage, module off and on, `E` 6 667, 10 000, 13 333, `RetryJobs` 0 and 10, `MinWorkShare` 0 and 4 000. Asserts the shape of the model of spec-subsistence 11.2 (the oracle of shape, not of numbers): unfed household-ticks fall when `Cov` >= 10 000; the price regime is reported (pinned shortage, pinned glut, interior band); `Cov` 6 667 with `RetryJobs` 0 is no better than none. Requirement: FR-SUB-019 to 022, 024.
- **S57 (SUB-S2).** `famine` with the estates overlay, module off and on: unfed households, removals and the pool's balance are lower on; the ratio of a floor-fed household is near what the floor covers, not 0; the mean ratio of the closed upkeep-only configuration of S16 is 10 000. Requirement: FR-SUB-028, 033.
- **S58 (SUB-S3).** Own-use durations 1, 3, 8 against `ChangePrivation` 3: rung 1 abandons a running own-use job only when `Duration - 1 >= ChangePrivation`, and the warning fires for exactly those. Requirement: FR-SUB-032, 038.
- **S59 (SUB-S4).** The draws overlay on a scarce `refill` commons: the ration holds, `Blocked` is non-zero when the stock is cut, cohorts and units get the same mean, the `Warn`s are bounded. Requirement: FR-SUB-023 (b), 036, 042.
- **S60 (SUB-S5).** The invariant run over `scenarioSeedCount` seeds with `WithAudit`, a checkpoint in the middle of the mode, a relocation of a household in the mode: the audit holds, no ledger reason beyond the registry, the resumed run is bit-identical, the relocated household's flag is dropped. Requirement: FR-SUB-003, 031, 044. `TestSubsistence_ConfigOnlyPack` (FR-SUB-047) runs the overlay alone with no Go. S59 also asserts no `endowment_unused` warning and an `Endowments` entry for a commons only an own-use recipe draws (D413).
Golden: `TestGolden_SubsistenceAlone` (pack `one` plus the overlay, seed 1, 400 ticks, `WithAudit`), `TestPositionHash_SubsistenceGolden`; the first run is recorded after the diff against `golden/off` is read (the diff is the module's whole effect). `TestGolden_OneOff` and the off position golden are re-run: unchanged.
Exit check: S56 to S60 recorded in `testdata/chains/expected/` by the same runner as S01 to S19.

### Phase 9: exit gate

Gates of `plan-2-0` and D229: `gofmt -l .` empty, `go vet ./...`, `staticcheck ./...`, `go mod tidy` no diff, `go test -short -race ./...`, R then F of D229, `lizard -l go -C 10 -a 8 -L 100 -w` on `subsistence*.go`, `internal/ownuse/*.go` and every changed non-test file, coverage >= 90% for the root and `internal/ownuse`, the mutation pass of Phase 2 repeated, the registry and catalogue tests with the SUB entries out of `notBuilt` ("Modules.Subsistence", "RoleOwnUse", "Subsistence", `OwnUseChanged`, positions "P2.4" and "P2.8"), `testdata/golden/off/` and `TestHashConfig_ReferencePinned` untouched, markdownlint on the files this plan adds under `doc/` (none). The PR description carries the `Testing` and `Observability` entries below.

## Catalogue appends

| File | What it appends | Position and id | Spec line |
| --- | --- | --- | --- |
| `config.go` | `Modules.Subsistence *SubsistenceConfig`; `SubsistenceConfig`, `RoleOwnUse` | after the pointers built before it; Row position 8 | spec-config 3.1; spec-subsistence 3.2 |
| `hash.go` | `pairsSubsistence`: `Recipes` 1, `Roles` 2 (`Role` 1, `Recipes` 2), `EnterTicks` 3, `ExitTicks` 4, `RetryJobs` 5 | `Modules` id **8**, in the id-explicit form of plan 5, inserted in ascending id place (after the lower ids, before any higher) | spec-config 8.2.1, 8.2.2 (E6) |
| `bounds.go` | `MaxOwnUseRecipes` 8, `MaxOwnUseTicks` 1 000, `MaxRetryJobs` 1 000 | end of the module bounds | spec-subsistence 13.1 |
| `defaults.go` | `DefaultSubsistence()`: 2, 3, 10, no list | after `DefaultRegion` and the earlier modules' | spec-subsistence 12 |
| `validate.go` | the call of `validateSubsistence`; `effectiveWindow` join; `unused_recipe` use; the call in `viability_check.go` | module seam, Row order | spec-subsistence 13; spec-production 10.4 |
| `errors.go` | `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited`, `ErrOwnUseOverlap` | after the earlier modules' sentinels | spec-subsistence 13.2; N-075 |
| `event.go` | `OwnUseChanged{Tick, Region, Role, Households, OnOwnUse, Entered, Left, Started, Blocked, Retried, Output []Amount}`; `isEvent`; log-only reasons `unfed`, `short_cover`, `cover` stay strings in `subsistence.go` (or `subsistence_report.go` if E5's split is made) | after the earlier modules' events | spec-observability 4.1; spec-subsistence 14.1 |
| `option.go` | none, because the module has no option and no extension point | none | spec-subsistence 22 |
| `tick.go` | `{"P2.4","SUB",line}` after P2.3, `{"P2.8","SUB",line}` before P2.8a; both leave `notBuiltPositions`; guarded calls `if w.sub != nil` | table order of spec-world 6.4 | spec-world 6.4 |
| `snapshot.go` | `RoleRow.OnOwnUse int64` | after the then-last field; zero while off, omitted from the golden | spec-observability 5.3.4 |
| `ledger.go` | none, because the module adds no reason and no account kind | none | FR-SUB-003 |
| checkpoint | tag **19** (the two runs, sparse); `OwnUse`, `OwnUseJobs` written once in tag 3 | tag 19; tag 3 hosted | spec-checkpoint 4.2 |
| log keys | `own_use` in `internal/logkeys` | in the order of rules 3.3.2 (after `drift`) | observability 3.3.2 |
| metrics | none here (the adapter, plan 22, feeds `economysim.household.own_use` and `economysim.own_use.units` from the event) | none | observability 5.4 |
| `construct.go` | `modulesOn` appends "subsistence" after the names already present (build order, plan 5 rule) | build order | rules 3.3.2 `modules`; spec-observability 5.3.1 |
| `fanin.go`, `world.go` | the module calls in `fillAgentView`, `removeAgent`, `agentRoleChanged`; `offStage`; `World.sub` | after the earlier modules' calls | architecture 2.10.4 |

## Cross-plan contract

Consumes:
- `Modules` struct and `pairsModules` in the explicit-id form (`config.go`, `hash.go`) <- plan-2-5
- the validator's module seam, the loader's module-table dispatch (`validate.go`, `config/mapping.go`) <- plan-2-5
- `tradableItems` fan-in with guarded module calls (`role.go`, `fanin.go`) <- plan-2-5
- the module-alone golden helper and `testdata/golden/<module>/` layout, the module construction call and the guarded `replayRows` row (`golden_test.go`, `construct.go`, `poshash_internal_test.go`) <- plan-2-5
- the rebalanced pack `one`, its warm document, `expected/` and `golden/off` (`testdata/`) <- plan-2-8 (the overlay and its bound are checked against that pack)
- the estates relief at P1.6 reading the welfare ratio (`estates.go`) <- plan-2-7 (S57 only)
- `Recipe.Draws`, `Modules.Resources`, `ReasonNoEndowment`, the ration grant reaching `startJob`, P2.3 and P2.6, the `Endowments` view group, `overlay/resources.toml` (`resources.go`, `testdata/chains/overlay/`) <- plan-2-9
- `scenarioSeedCount`, `updateAllowed`, gates R and F of D229 (`scenario_helpers_test.go`) <- plan-2-1
- `invariantSeeds` (`scenario_helpers_test.go`; the `invariantTable` row of Testing 5 and `TestDeterminism_Subsistence` call it), `detCases` (`determinism_test.go`) <- plan-2-1
- `checkpointSections`, `notBuiltSections`, `checkpointCross`, the agent-record hosted fields of D247 (`checkpoint_sections.go`, `checkpoint_restore.go`), `forkAtN`, `forkCase` (`checkpoint_fork_test.go`) <- plan-2-2
- `moveAgent`, the export_test `RelocateForTest` seam (`regions_relocate.go`, `export_test.go`) <- plan-2-3; S57 and the other users move to the public `World.Relocate` when plan 17 lands, which deletes the seam
- the fan-ins `agentJoined`, `agentRoleChanged` (`fanin.go`; this plan uses the second) <- plan-2-5
- `fillEndowments`, the `endowment_unused` rule (`resources.go`, `resources_check.go`) <- plan-2-9 (extended here, D413)
- `welfareScratch.at`, `markWorking`, `workingTick`, `valuePrices`, `Ring` own-use fields (`welfare.go`, `phase_outcomes.go`, `internal/welfare`) <- Stage 1
- Plans 2, 3, 5, 7 and 8 are named beyond Needs (9): checkpoint sections, `moveAgent`, `tradableItems` and `agentRoleChanged`, relief at P1.6, the rebalanced pack.
- `workReport`, `fillRecipes`, `newViewScratch`, `admit`, `residueLen`, `finishJob`, `startJob`, `removeAgent`, `addAgentRow`, `internal/checked`, `internal/household` (Stage 1 files) <- Stage 1

Provides:
- `Modules.Subsistence`, `SubsistenceConfig`, `RoleOwnUse`, `DefaultSubsistence` (`config.go`, `defaults.go`) -> plan-2-24
- `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited`, `ErrOwnUseOverlap` (`errors.go`) (no consumer plan)
- `AgentView.OwnUse`, `OwnUseJobs`, `Cover`, `Secure`, `ExitTicks`, `RetryJobs`, `RecipeView.OwnUse` (`strategy.go`) -> plan-2-15
- `RegionAgent.OwnUse` (`spawner.go`) (no consumer plan)
- `roleRow.ownUse`, `roleRow.jobs` (`role.go`) (no consumer plan)
- `(*World).offStage` predicate stub and its call site in the flag step (`world.go`); the call sites at `workAgent` and `tradeAgent` are plan 12's (D447) -> plan-2-12, plan-2-13, plan-2-14
- `(*subsistence).drop(slot)` (`subsistence.go`) -> plan-2-14, plan-2-17
- `(*subsistence).agentFields(slot)` (`subsistence.go`) -> plan-2-17
- `OwnUseChanged` (`event.go`) -> plan-2-18, plan-2-22
- `RoleRow.OnOwnUse` (`snapshot.go`) -> plan-2-18, plan-2-22
- `ownuse.Cov`, `ownuse.FloorBP` (`internal/ownuse`) (no consumer plan)
- position P2.4 and P2.8 (`tick.go`, `subsistence.go`) (no consumer plan)
- `overlay/own-use.toml`, `own-use-draws.toml`, `testdata/golden/subsistence/`, S56 to S60 (`testdata/`) -> plan-2-24

## Testing (testing.md 11)

**1. Requirements touched and their tests.** Every ID of spec-subsistence section 20.

| Requirement | Tests |
| --- | --- |
| FR-SUB-001 | `TestSubsistence_OffLeavesNoTrace` (off golden, hash, no event, identical Run) |
| FR-SUB-002, 009 | `TestEffectiveList_*`, `TestSubsistence_NoRegionKeyedField`, `TestSubsistence_RegionOffersByYieldRow` |
| FR-SUB-003 | `TestSubsistence_NoNewReasonAndAuditHolds`, S60 |
| FR-SUB-004 to 008, 050 | `TestValidate_OwnUse*`, `TestValidate_DrawsNeedResources` |
| FR-SUB-010, 015, 016, 017 | `TestFlag_*`, `TestFlagStep_*` |
| FR-SUB-011 to 014, 039 | Phase 2 unit tests |
| FR-SUB-018 | `TestView_CarriesOwnUseGroup` |
| FR-SUB-019 to 025, 048, 049 | `TestWork_*`, S56 |
| FR-SUB-026, 027 | `TestAdmit_OwnUseBeforeAnyOtherCheck`, `TestAnchors_OwnUseNeverSets` |
| FR-SUB-028 to 030, 051 | `TestWelfare_*`, `TestWage_ExcludesOwnUse`, S57 |
| FR-SUB-031, 032, 033 | `TestDrops_*`, `TestMode_DoesNotTouchHungerPrivation`, S58, S60 |
| FR-SUB-034, 035, 036 | `TestWork_DirectedOverrides_ExternalEvaluated`, `TestStream_*`, `TestRation_OwnUseNoPriority`, S59 |
| FR-SUB-037, 038 | `TestViability_OwnUseOnlyRecipeNotAnActivity`, `TestWarnings_Eight` |
| FR-SUB-040 to 043, 045, 046 | Phase 6 tests (FR-SUB-043 metrics half: plan 22) |
| FR-SUB-044 | `TestCheckpoint_Subsistence*`, `TestFork_Subsistence` |
| FR-SUB-047 | `TestSubsistence_ConfigOnlyPack` (the overlay alone, no Go) |
| NFR-SUB-001, 002, 003 | `TestAllocs_*`, `TestOwnuse_ImportsOnly`, `BenchmarkFlagStep` |

**2. Lowest level.** The rules are unit tests of `internal/ownuse`; the hooks are internal tests of `subsistence.go`; the pack tests are S56 to S60.
**3. New boundaries and error paths.** The four sentinels, the bounds, the overflow of `Cost` (a pack at `MaxQuantity` times `MaxPrice`), the decoder of tag 19.
**4. Golden files.** `golden/off/` and the pinned hash do not move (the proof runs at every phase exit). `golden/subsistence/` is new (module alone). `golden/all/` does not exist yet and is not touched.
**5. Fork-at-N.** `TestFork_Subsistence`, in the middle of the mode, plus S60. Add the row `one + own-use.toml` to `invariantTable` (REF 7.5) and a new `TestDeterminism_Subsistence` over `one + own-use.toml` that takes cases from `detCases(true)` by name (REF 8.2).
**6. Registry and catalogue.** Rows `Subsistence.*` (`EnterTicks`, `ExitTicks`, `RetryJobs`, `Recipes`, `Roles`), `RoleOwnUse.*`, `Modules.Subsistence`; `OwnUseChanged` and positions P2.4, P2.8 leave their `notBuilt` lists.
**7. Fuzz.** `FuzzValidateSubsistence`, `FuzzStep`, `FuzzDecodeSubsistence`.
**8. Mutation.** `gomu` on `internal/ownuse` only in Docker (`--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`).

## Observability (observability-and-logging.md 9)

1. New failure path: four configuration sentinels (returned); a household with nothing runnable is survived and counted in `Blocked`, with a bounded `Warn`.
2. New state transition: entering, leaving, a retry, a blocked tick, an output: `OwnUseChanged`, aggregate, no map, no pointer, integers.
3. New decision point in a strategy: the choice of recipe in the mode: a Debug line with the reason.
4. New loop: the flag step per agent per tick; logs behind `Enabled`; the only `Warn`s are bounded by regions times roles; event construction allocates nothing with the no-op observer (NFR-SUB-001).
5. Changed event, metric or attribute: none; one event, one key (`own_use`) and two metrics (plan 22) are added.
6. New metric attribute: none. 7. `Error` or `Info`: neither. 8. Error logged and returned: none.
9. Tests: a recording observer for the event and its order, a capturing handler for the `Warn`s at ticks 10 and 20, non-interference, the no-op allocation test.

## Risks and open items

1. **Home baking (the question N-314 puts to this plan).** An own-use recipe must be a source, so it cannot bake `prepared` from the household's own `staple` and `fuel`; this plan does not change that (D401). Cost, on pack `one` (upkeep 2 `staple`, 1 `prepared`, 1 `fuel`): the covered set can hold `staple` and `fuel` through source recipes; `prepared` only through a stand-in source (D416) that makes it from nothing, so it overstates what a household can do and breaks the link to the staple stock. Without the stand-in `Fed` (every upkeep commodity met) is unreachable by the mode, `Hunger` and `Privation` never fall, and S56 cannot assert fewer unfed households. The alternative is a **closure rule**: an own-use recipe may take inputs only from non-discard outputs of recipes on the same effective list, acyclic. It changes spec-subsistence 3.4.1, FR-SUB-004 and `ErrOwnUseNotSource`; makes `Cov` a chain figure and the bound of 4.2 a labour figure for the chain; adds `no_input` to the blocked reasons; changes the covered set and FR-SUB-037. Estimated: one more phase and about 150 lines in `internal/ownuse`, and the "floor under the market" property weakens where the input is bought. The owner decides (supervisor); the N-314 probe (e) says the closure would keep the growers fed, not the cooks and the tool chain.
2. **Reading of "exact fractions"** (E2) and **which surplus figure** (E3): both are spec points the plan does not decide.
3. **Seams of earlier plans.** `moveAgent` (plan 3), the validator seam, `tradableItems`, `pairsModules`, the golden helper (plan 5) and the checkpoint section protocol (plan 2) are taken from the plans' Provides lines; E10 verifies them. Provides to plans 12, 17, 18, 22 and 24 (`roleRow.ownUse/jobs`, P2.4/P2.8, `OwnUseChanged`, `ownuse.Cov`, the overlays) are matched by their Consumes lines except `roleRow.ownUse/jobs` and P2.4/P2.8 in plans 12 and 17; the supervisor reconciles.
4. **Staged reasons** (D418): FR-SUB-023 is fully testable only when `no_title`, `rent_funds`, `no_room` and `not_permitted` exist; S59 covers `no_endowment`.
5. **Pack dependence.** `own_use_beats_market` and the overlay's `Cov` depend on the `SurplusBP` plan 8 settles; if it falls below 10 000 for a region the overlay quantities are changed, not the rule.
6. **Cohorts above weight 1** run the approximation of spec-subsistence 5.5 item 4 (open question 7); S59 measures it.
7. **Per-tick cost** at 100 000 agents: one loop with a bounded inner loop; `BenchmarkFlagStep` records it for plan 4's budget.

**Note proposals** (text only; the supervisor is the only editor of `doc/notes.md`):
- *New note, areas SUB, REF, DEM:* "Pack `one` cannot be fed by SUB alone: upkeep includes `prepared`, which only a recipe with inputs makes. The overlay uses a source stand-in. Decide between keeping the source-only rule (and the stand-in as a test double) and a closure rule (inputs only from the same list, acyclic); cost in plan-2-10 Risks 1."
- *Update N-228:* "Applied for SUB by plan 2-10: the own-use entry of `tradableItems` (D404). Both halves are then built; propose closing at this plan's gate."
- *Update N-118:* "Plan 2-10 builds one job slot; the gap between yields is visible as `OwnUseChanged.Blocked` and in S58; left open for the first scenario that shows it."
- *Update N-119:* "Plan 2-10 builds the receipt at creation and the bound of 7.5; no provenance; left open."
- *Update N-121:* "Plan 2-10 gives a no-recipe role an own-use request in the mode and the view fields MER reads; the merchant side is plan 15."
- *Update N-268, N-244, N-128:* "Applied for SUB in plan 2-10: `RegionAgent.OwnUse` and the reset of `OwnUseJobs` are appended; the state is in module slices; the flag and counter are written once in tag 3."
- *Update N-314:* "SUB keeps the source-only rule (plan 2-10 D401); the floor covers staple and fuel by source recipes and `prepared` only by a stand-in."
- *New note, areas SUB, WLD:* "The blocked reasons of FR-SUB-023 are staged by module; each module that adds a refusal appends it to `blockedStart`."

**Open items with owners:** E2 and E3 (spec-subsistence owner, the supervisor); Risks 1 (the owner); the relocation seam (plan 3); plan 5's validator seam identifiers (its author, reconciled by the supervisor); whether the registry test's scope alias is plan 5's or this plan's (whoever builds the first `...Config` module; this plan adds it if absent).
