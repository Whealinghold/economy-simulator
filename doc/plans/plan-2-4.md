# Plan: Stage 2, row 4: SCL, `scale.go` (weights, the census, the first cost measurement)

**Status: plan written (2026-10-22), not started. Written against plan-2-0.md; nothing here is built.**
Sequences the roadmap-v1.md Stage 2 row "`scale.go` (weights, the census) | SCL" ([roadmap-v1.md](../roadmap-v1.md), Stage 2 table) into six phases. It implements [spec-scale.md](../specs/spec-scale.md) (SCL) sections 3, 4.5, 5.2, 6, 7 and 13 and the requirements FR-SCL-001 to 006, 011, 015 to 020 and NFR-SCL-001 to 004. It also makes the first measurement of a `Step` that N-300, N-222 and N-015 ask for. Where this plan and the specs or rules disagree, they win and this plan is corrected.

Rule tags read for this plan (all three files read in full, tags equal those of plan-2-0.md): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1.
Specs read: spec-scale.md (in full), spec-agents.md (3.5, 3.7, 9, FR-AGT-052), spec-world.md (5.1, 5.4, 6, NFR-WLD-001/002, FR-WLD-030 to 038), spec-checkpoint.md (4.2 tag 3, 10.2, NFR-CHK-002), spec-reference-chain.md (S10, 6.5, 7.1.5), spec-stylised-facts.md (STY-P09, FR-STY-028), spec-population.md 3.8.
Notes read, with the verdict on each (the notes file is not edited by this plan):
- N-030 (areas CAL, PRO, SCL, STY, REF): leave open. The measured world size of the first consumer is this plan's budget harness (Phase 5); the tick rate is still unknown.
- N-198 (the drawdown is `O(T)` per agent and tick): leave open. The saving module (SAV) is Stage 3; there is nothing to measure.
- N-222 (cost of a reseed at every draw site): apply the `one` and `five` half (Phase 5 measures it by a CPU profile of a `Step`); leave open for the all-on half (plan 24). Propose no close.
- N-300 (the first measurement of a `Step` of `one`): apply the `one` and `five` half (Phase 5 records the time per tick and the allocations); leave open for the all-on half (plan 24). Propose no close.
- N-015 (placeholders): apply for `MinAgents` 10, the 10% tolerance, 100 000 agent-ticks a second and the constants 2 us, 1 us and 8 (Phase 5 records the measured values; the numbers stay placeholders until the spec is edited).
- ALL: N-016 apply (no configuration field is added, so no registry row), N-017 apply (terms in Phase 1), N-027 apply (no default is claimed stable), N-103 leave open (this plan edits one file), N-126 apply (no event, no `Warn`), N-128 not applicable (no per-agent field), N-163 apply (the word "pool" only), N-217 apply (the pinned hash and `testdata/golden/off/` must not move).

Needs plans 1 and 2 (plan-2-0.md). Plan 3 is merged by then (order), so the `five` probe of Phase 5 is expected to run; if it still fails, record the error (D291) and `BenchmarkStepFive` is not built.

Decision block: D280 to D299 (D280 to D295 used).

## Scope

**Builds.**
- `scale.go` (root package), the one home of the derivation of spec-scale.md 3.2: `agentCount`, `groupWeights` (moved from `construct.go` and rebuilt on `agentCount`, same results, as spec-scale.md 13 says), `groupShape` (moved from `validate.go`), `groupPlan` and `groupPlans` (moved from `construct.go`), and the new `checkCensus` invariant (D280 to D283).
- The scale tests: the scale conformance suite (`scale_internal_test.go`, `scale_pack_test.go`), the S10 scenario (`scenario_s10_test.go`, D286 to D288), the cost benchmarks and the measurement (`scale_bench_test.go`, D289 to D293), and the declaration of the two S10 band keys in `testdata/chains/bands/one.toml`.

**Public identifiers.** None. The census is reachable today through `Snapshot` (`Agents`, `Households` per world, region and role: `snapshot.go` lines 34, 65, 98), `AgentSpawned.Weight` and the `world created` log (`agents`, `households`: `construct.go` `summary`). Spec-scale.md 13 gives the library no exported function for the census, and `AGENTS.md` allows an interface or a new exported name only where the rules say (D282).

**Leaves out, with the owner.**

| Item | Owner |
| --- | --- |
| The chance-event helper of spec-scale.md 4.6 and FR-SCL-012 (spec-scale.md 13 says "an `internal/` package") | plan 11 (POP): the carried remainder of spec-population.md 3.8 is its only user; production and tool breakage need none (spec-scale.md 12 item 3). D284, owed edit 1 |
| FR-SCL-013 (an external agent has weight 1) | plan 17 (INT) |
| FR-SCL-014 split and merge, and any weight change by population | plan 11 (births and deaths); version 1 has no split or merge (spec-scale.md 8). Reductions and entries by the spawner are Stage 1 and are only re-checked here (Phase 3) |
| The weight in the checkpoint (agent record, tag 3) | plan 2 (CHK). This plan only tests that a scaled world survives a checkpoint (Phase 4) |
| The values of `s10.tolerance_bp` and `s10.dispersion_min_bp` (the pass of FR-SCL-017) | plan 19 (STY, FR-STY-028; plan 19 D598 takes the handoff) on a rebalanced pack (plan 8) |
| The cost budget on `five` and the Stage 2 exit criterion "the cost budget of SCL holds for the first consumer's world size" | plan 24, with this plan's harness. `five` needs regions to be real (plan 3) before it can be stepped (D291) |
| `MinAgents` as a calibrated value (the least `K` at which S10 passes) | plan 21 (CLB) with plan 20 (EXP); open question 2 of the spec |
| Level of detail: split and merge in a running world | out of version 1 (spec-scale.md 8) |
| A bound on the number of agents (Risks, item 1) | owner ruling: CFG |

## Decisions

- **D280** `scale.go` holds, in this order: `agentCount`, `groupWeights`, `groupShape`, `groupPlan`, `groupPlans`, `checkCensus`. The first four are pure functions over `int64`; `groupPlans` reads a validated `*Config`. Source: spec-scale.md 13 ("Stage 2's `scale.go` takes it over unchanged"), architecture.md 2.6 (one file per concept; `scale.go` is in the layout list of `AGENTS.md`), roadmap-v1.md Stage 2 row "`scale.go` (weights, the census)".
- **D281** There is one definition of spec-scale.md 3.2. Today `construct.go:42` (`groupWeights`) and `validate.go:811` (`groupShape`) each compute `ceil(H / S)` raised to `min(K, H)`. `agentCount(households, scale, minAgents int64) int64` is that formula once; `groupWeights` builds the weights from it and `groupShape` (kept, same signature, same `ok` rule, moved) returns `n` and `ceil(H / n)` from it. `agentCount` returns 0 for `households < 1` or `scale < 1` (`groupWeights` then returns nil and `groupShape` `ok=false`, as today). The move changes no behaviour, so the pinned hash and the goldens do not move. Source: FR-SCL-002, FR-SCL-003; spec-world.md 5.4.1.
- **D282** No new exported identifier and no exported census type. The census is the `groupPlan` slice (region, role, weights) that construction already builds, plus the snapshot counts. A consumer that wants the households of a group reads `Config.Groups` (spec-land.md 6.1.1 reads `Households` the same way). Source: `AGENTS.md` hard constraints (interfaces only at the extension points), spec-scale.md 13 (no exported function), spec-observability.md 5.3.2 and 5.3.4 (the snapshot reports the counts).
- **D283** `checkCensus(plans []groupPlan, c *Config, agents, households int64)` runs at the end of the agent-creation step of `construct`, after the groups are created, and calls `invariant(...)` (the existing panic helper, `clearing.go:259`) when, for any group, the weights do not sum to `Households`, the weights differ by more than 1, a weight is below 1, a group of weight above 1 has fewer than `min(K, H)` agents, or the totals differ from the live counters (`liveCounts`). It is `O(groups)` and allocation-free. It panics rather than returns an error because a validated configuration cannot violate it (FR-WLD-036 precedent: the audit is the error path; the construction invariants are panics, `construct.go` `mustBuild`). Source: FR-SCL-003, 005, 019; spec-scale.md 7.1.
- **D284** No chance-event helper in this plan. FR-SCL-012 is served by plan 11's carried remainder with the denominator `10 000 * TicksPerYear` (spec-population.md 3.8, FR-SCL-012's "recommended method"). The test of FR-SCL-012 is plan 11's. Spec-scale.md 13's last row, which asks for an `internal/` package, is corrected by owed edit 1. Source: spec-scale.md 4.6 and 12 item 3; spec-population.md 3 (decision row 7); roadmap-v1.md Stage 2 (no SCL `internal/` package).
- **D285** The scale conformance suite runs at weights above 1 on the real packs (`one` warm, with `overlay/h200.toml`) and asserts the exact clauses only: FR-SCL-001 to 006, 015, 018, 019 and the construction half of 016. The per-module weight behaviour (upkeep, recipes, welfare, limit price, cohort invariance) is already tested at unit level by the Stage 1 rows under their own IDs (`TestConsume_Cohort`, `TestDefaultStrategyTrade_CohortInvariance`, `TestTrade_CostAndDrawsDoNotDependOnTheWeight`, `TestDefaultStrategyWork_ScalesWithTheWeight`, `TestNewAgent_Weight`, `TestExecute_RemoveAndReductionToTheWholeWeight`); Phase 2 adds `FR-SCL-007`, `008`, `009`, `010`, `014` to their `Requirement:` lines where the clause is what the test asserts, and writes no duplicate test. Source: testing.md 4.2, 4.5 (a requirement with no test is a defect; an existing test is the test), 2.1 (lowest level that can catch it).
- **D286** The S10 scenario is `scenario_s10_test.go`: pack `one` + `warm-one.toml` + `overlay/h200.toml`, `world.scale` 1, 4 and 10, `min_agents` 10 (1 200, 300 and 120 agents), seed 1, 365 ticks from tick 0, no settling period, observer recording `MarketCleared` and `BeliefsUpdated`. Exact half (runs in `-short`, construction only): the weights of each group sum to 200 (FR-SCL-003), the money and the quantity of each good at tick 0 equal at the three scales and equal to households times the per-household amount (FR-SCL-019, FR-SCL-007), `Snapshot.Households` 1 200 at every scale and `Snapshot.Agents` 1 200, 300, 120 (FR-SCL-015, 018). Run half (skipped under `-short` and when `raceEnabled`; it runs in gate F of plan-2-1 D229 and is written with `-update` only where `updateAllowed(race, updating)` allows, D228; S10 has one seed, so plan 1's seed reduction does not apply): the time-averaged price of each commodity at scales 4 and 10 against scale 1, and the dispersion of belief centres per market. Source: spec-reference-chain.md S10; spec-scale.md 7.1, 7.2.
- **D287** The run half reports and asserts only through `checkBand` (`scenario_helpers_test.go:981`) on the keys `s10.tolerance_bp` and `s10.dispersion_min_bp`, as S16 does: a key with no value asserts nothing. This plan appends the two keys, `unit = "bp"`, with no value, to `testdata/chains/bands/one.toml`. The values are STY's (STY-P09, FR-STY-028; priors 1 000 and 100, spec-stylised-facts.md 596). Source: spec-reference-chain.md 7.1.5, FR-REF-024.
- **D288** FR-SCL-017 is a SHOULD and the pack `one` collapses (N-312, N-314), so a price comparison at three scales on it measures the collapse, not the scale. This plan claims no pass of FR-SCL-017: the report is recorded in `testdata/chains/expected/one.scenarios.txt` with `-update` like the other scenarios (the D181 mechanism of Stage 1), and plan 8 (rebalance) and plan 19 re-run it. Source: plan-2-0.md "Known facts"; N-312; N-314.
- **D289** The cost benchmarks are in `scale_bench_test.go` (package `economysim_test`, like `benchOne`, `observe_bench_test.go:18`: `config/` imports the root, so an internal test cannot load the packs with `config.LoadFS`; the mean `|T_a|` is read through a small accessor in `export_test.go`). All use `for b.Loop()` and `b.ReportAllocs()` (NFR-SCL-003, testing.md 8.1). `BenchmarkStepAgents/n=600,1200,2400` multiplies every `Households` of `one` by 1, 2 and 4 at scale 1 (agents double, NFR-SCL-001). `BenchmarkStepWeights/w=1,7,40` keeps 120 agents by `Scale` and multiplies the households (same agents, other weights, FR-SCL-016 run half). Each reports `ns/agent-tick` and `agent-ticks/s` with `b.ReportMetric`. Nothing is asserted (testing.md 8.2: benchmarks report, they never fail CI). Source: NFR-SCL-001, 002, 003; spec-scale.md 6.2, 6.5.
- **D290** The cost model is fitted from two worlds of different mean tradable set: the scaled `one` (mean `|T_a|` computed in the benchmark from the role rows) and, if Phase 5's probe allows, `five` (7.7, spec-reference-chain.md 4.4.4). Two points fix `c_0` and `c_t` of `c_0 + |T_a| * c_t`; the fit and its arithmetic are written in the doc comment of the benchmark and proposed as a note update, never stated as measured beyond the machine. If only one pack can be stepped, only the per-agent-tick cost at its `|T_a|` is recorded and `c_0` and `c_t` are not separated. Source: spec-scale.md 6.1, 6.2 (the slice "must measure the real costs and replace them").
- **D291** The measured window is the same for every size: 3 warm-up steps then `b.N` steps from the warm start, as `benchOne` does (`observe_bench_test.go:15`). Because the economy collapses (N-312), the benchmark logs once, before timing, the live agents, the households and the trades of the last warm-up step, so a reader can see whether the measured tick was a working market; the numbers are labelled "collapsing pack" until plan 8 and re-measured by plan 24. `five` is stepped only if `NewWorld` accepts the warm `five` layers (`base`, `lines`, `five`, `warm-five`) and its five regions each clear their own markets; if it does not, the first consumer's size is built from `one` with households multiplied to 5 000 agents, and the real `five` run is plan 24's. Source: spec-scale.md 6.3 last row, N-030, roadmap-v1.md Stage 2 "Done when".
- **D292** N-222 is answered by a CPU profile, not by new code: `go test -run '^$' -bench 'StepAgents/n=2400' -cpuprofile ... `, then `go tool pprof -top` for the cumulative share of `(*stream).keyed`, `keyedPair` and `reseed` in a `Step`. The figure is recorded with the machine, Go version and `GOMAXPROCS`, beside the existing `BenchmarkStream_KeyedDraw` result (`random_internal_test.go:364`). The decision whether the per-agent sites (D1, D3, D5, D8, D10 of ADR 0008) need a cheaper key is not taken here: it is ADR 0008's owner's, and the plan gives them the number. Source: N-222; ADR 0008 Consequences.
- **D293** N-300 is answered in the same run: time per tick of `one` (no observer, then the recording observer through the existing `BenchmarkStep_OneNoObserver` and `BenchmarkStep_OneRecorder`), allocations per `Step` (the baseline `stepAllocBaseline = 113`, `allocation_test.go:13`, is not changed by this plan), and the time of the cardinality and allocation tests are not re-measured. Source: N-300 update of 2026-10-21; NFR-WLD-001, NFR-OBS-002.
- **D294** The agreement test `groupShape` against `groupWeights` over a grid (households 1 to 400, scale 1 to 50, floor 1 to 20, plus the bounds `1`, `MaxHouseholds` with a scale that keeps the agent count small) is the guard that the two never diverge again, and the largest weight `groupShape` returns equals `max(groupWeights(...))`. Source: FR-SCL-002, 003, 011; validate_mutation_test.go:350 (the existing single-case guard).
- **D295** The household count is tested over a run where weights change: pack `one` at `world.scale` 4 collapses and the spawner reduces weights by the failure ladder (spec-spawner.md 5), so the test first asserts that at least one `WeightChanged` was observed (else the guard is exercised by no event), and each tick `Snapshot.Households` equals the sum over `Regions` and over `Roles`, equals the sum of the weights of the live agents, and equals the count the `world created` and per-tick log lines carry (`liveCounts`). Source: FR-SCL-015, FR-SCL-014 (conservation: the audit is on with `WithAudit`), spec-scale.md 4.8.
- **D296 to D299** reserved, unused.

## Phases

### Phase 1: the documents (no code)

Goal: every spec statement the code relies on is true before Phase 2.
Files: none in the repository by this plan; the edits below are the supervisor's.

**Edits owed.**
1. `doc/specs/spec-scale.md` section 13, row "Deriving the agents": the home is `scale.go` (`agentCount`, `groupWeights`, `groupShape`, `groupPlans`, `checkCensus`), not "`construct.go` ... Stage 2's `scale.go` takes it over". Row "The chance-event helper (4.6)": the home is the carried remainder of `internal/population` (plan 11), no separate package; FR-SCL-012 is tested there.
2. `doc/specs/spec-scale.md` 6.2, 6.3, 12 item 1 and NFR-SCL-002: replaced after Phase 5 with the measured `c_0`, `c_t`, the tick of the first consumer's world and its machine. Owed after Phase 5, not before.
3. `doc/specs/spec-scale.md` FR-SCL-016: "the same work" has no operational form. Propose: at construction the agents, accounts, lots, beliefs and the checkpoint size are equal (exact, asserted, Phase 3 and 4); at run time the allocations of a `Step` and the time per agent-tick are reported by benchmark and have no tolerance until a measure shows what to expect (Risks, item 2).
4. `doc/specs/spec-config.md` 4.1 and `spec-scale.md` 4.5 / FR-SCL-011: a bound on the number of agents (Risks, item 1). Owed only if the owner rules it; no value is proposed here.
5. `doc/specs/spec-reference-chain.md` S10 and the file list: `bands/one.toml` declares `s10.tolerance_bp` and `s10.dispersion_min_bp` (unset); S10 builds on the warm layer `warm-one.toml`, which is what `packOne` is (`support_test.go:34`).
6. `doc.go` glossary, before any new code uses them (architecture.md 3.2): **census** (the configured households of the groups of a world, from which the agents are derived: `Group.Households`), **agent floor** (as `MinAgents`: the fewest agents a scaled group may have), **cohort** (an agent of weight above 1). "Weight", "scale" and "group" are already there (`doc.go:165`, `168`).
7. `doc/roadmap-v1.md` Stage 2 row `scale.go`: mark `scale.go` done with its commit at the exit gate (supervisor).

Exit check: edits 1, 5 and 6 are applied (the others are later or owner rulings); `TestGlossary_Vocabulary` and `TestGlossary_DeclaredTypesNamed` (`glossary_test.go`) still passes.

### Phase 2: move and unify (`scale.go`)

Goal: D280, D281 with no change of behaviour.
Files: `scale.go` (new), `construct.go` (`groupWeights`, `groupPlan`, `groupPlans` removed; its header comment line "Stage 2's scale.go takes it over unchanged" removed), `validate.go` (`groupShape` removed), `construct_internal_test.go` (`TestGroupWeights` moves to `scale_internal_test.go` unchanged).
Tests added or moved (all in `scale_internal_test.go`):
- `TestGroupWeights` (moved, table-driven, the spec's two worked examples 2 000 at 40 and 105 at 40 floor 10). Requirement: FR-SCL-001, FR-SCL-002, FR-SCL-003, FR-SCL-004, FR-SCL-005.
- `TestAgentCount` (table: households 0 and scale 0 give 0, `ceil` with and without remainder, floor wins, floor above households gives households, households 1, `MaxHouseholds`). Requirement: FR-SCL-002, FR-SCL-020 (a floor above the group's households is legal and silent).
- `TestGroupShape_AgreesWithGroupWeights` (D294). Requirement: FR-SCL-002, FR-SCL-003. Guard: two copies of the formula drifting apart.
- `TestGroupWeights_RejectsWhatValidationRejects` (households 0, scale 0 return nil, as `groupWeights` does today). Requirement: FR-SCL-001.
- `TestGroupWeights_DrawsNothing` (the function takes no `*stream`; the test builds two worlds of the same config with seeds 1 and 2 and compares the weight vectors). Requirement: FR-SCL-004.
- `FuzzGroupWeights` (`testdata/fuzz/FuzzGroupWeights/` corpus; inputs bounded to at most 100 000 agents so the fuzzer cannot allocate the 10^9 of Risks item 1): sum equals households, spread at most 1, count equals `agentCount`, every weight at least 1, no panic. Requirement: FR-SCL-001, 002, 003.
- `TestValidate_GroupShapeBounds` gains the rows households 0 and scale below 1 rejected by `Validate` (`validate_mutation_test.go:290`; Requirement adds FR-SCL-001). Existing, FR-SCL-020 and `TestValidate_GroupWeightIsCeilOfHouseholdsOverAgents` (line 350) must pass unchanged.
Exit check: `go test -short -race ./...` passes; `TestHashConfig_ReferencePinned` and the golden tests pass with no file changed; `git diff --stat` shows `construct.go` and `validate.go` only losing the moved code.

### Phase 3: the census invariant and the conformance suite

Goal: D283, D285, D295 and the exact clauses of the requirements.
Files: `scale.go` (`checkCensus`), `construct.go` (one call), `scale_internal_test.go`, `scale_pack_test.go`, `agent_internal_test.go`, `strategy_default_trade_internal_test.go`, `strategy_view_property_test.go`, `welfare_internal_test.go`, `strategy_default_budget_internal_test.go`, `phase_exit_exec_internal_test.go` and the other tests named in D285 (a `Requirement:` line each, no logic).
Tests added:
- `TestCheckCensus_PanicsOnABrokenPlan` (table of hand-built plans: a sum that is one short, a spread of 2, a weight 0, a group of weight above 1 under its floor, a live counter off by one; each panics through `invariant`; a good plan returns). Requirement: FR-SCL-003, FR-SCL-005, FR-SCL-019. Guard: the construction accepts a census that does not add up.
- `TestScalePack_ConstructionTotalsAreExactAtEveryScale` (`scale_pack_test.go`, integration, skipped under `-short` like its neighbours): the packs `one` (+h200) and `second` at scales 1, 4, 10, 40 and floors 1, 10: Σ weights per group equals `Households`; the money of the household accounts equals Σ households times `Role.StartMoney` (or the warm table's per-household amount times the households); each good equals households times the per-household quantity, read from the ledger's holdings; `WithAudit` passes. Requirement: FR-SCL-003, 007, 019. Arithmetic in the comment: 136 cooks at scale 4, floor 10: `n = ceil(136 / 4) = 34`, weights 4 each, 34 * 4 = 136.
- `TestScalePack_AgentSpawnedWeightsAndSnapshotCounts`: the `AgentSpawned.Weight` values of the recorder sum per (region, role) to `Households` and the snapshot reports agents and households separately at each scale (extends `TestSnapshot_CountsAndScale`, `snapshot_test.go:133`, which covers FR-SCL-015 only). Requirement: FR-SCL-018, FR-SCL-015.
- `TestScale_BankruptcyCarriesTheWeight` (`scale_internal_test.go`): a scripted bankruptcy of an agent of weight 7 emits `AgentBankrupted` with `Weight` 7. Requirement: FR-SCL-018 (second half).
- `TestScalePack_HouseholdCountIsTheSumOfWeights` (D295; 200 ticks, scale 4, `WithAudit`, seed 1). Requirement: FR-SCL-015, FR-SCL-014. Guard: a reduction that changes the live counter but not the snapshot.
- `TestScalePack_ConstructionStateDoesNotDependOnTheWeights`: two worlds with the same agent count and weights 1 and 7 (households multiplied, scale multiplied): equal numbers of agents, accounts, lots at tick 0, beliefs and welfare ring length; equal `len` of the private state a test reads through `export_test.go`. Requirement: FR-SCL-016 (construction half), NFR-SCL-004 (agents and lots, before the checkpoint half of Phase 4).
- `TestScale_FloorAboveHouseholdsIsSilent`: a group of 4 households with `MinAgents` 10 builds four agents of weight 1, `Validate` returns nil and `Warnings` holds no `group_below_floor`. Requirement: FR-SCL-020, FR-SCL-005.
- The relabelled tests of D285 gain FR-SCL-007 (upkeep: `TestConsume_Cohort`, `TestNewAgent_Weight`), FR-SCL-008 (`TestDefaultStrategyWork_ScalesWithTheWeight`, the executions line of `TestDefaultStrategyTrade_CohortInvariance`), FR-SCL-009 (`welfare_internal_test.go` `TestRecord_WearRoundingAndWeight`), FR-SCL-010 (`TestDefaultStrategyTrade_ScalesWithTheWeight`), FR-SCL-014 (`TestExecute_RemoveAndReductionToTheWholeWeight`) only after Phase 3 reads each and finds the clause asserted; a clause not asserted gets a new test in `scale_internal_test.go` (named in the exit report), not a made-up ID.
Exit check: the root coverage does not fall; `scale.go` is at 100% of its statements; the pinned hash and the goldens did not move.

### Phase 4: S10 and the checkpoint of a scaled world

Goal: D286 to D288 and NFR-SCL-004.
Files: `scenario_s10_test.go`, `testdata/chains/bands/one.toml` (two keys), `testdata/chains/expected/one.scenarios.txt` (recorded with `-update`), `scale_checkpoint_test.go`.
Tests:
- `TestScenarioS10_Exact` (runs in `-short`, construction only). Requirement: FR-SCL-003, FR-SCL-007, FR-SCL-018, FR-SCL-019, FR-REF-033.
- `TestScenarioS10_PricesAcrossScales` (skipped under `-short`, scenarios run under the existing `scenSem` semaphore; skipped when `raceEnabled`, so gate R skips it and gate F runs it): reports the per-commodity averaged price at scales 1, 4, 10, the relative gaps in basis points and the dispersion; asserts through `checkBand` on the two unset keys. Requirement: FR-SCL-017, FR-SCL-010, FR-SCL-016, FR-AGT-052. With the keys unset the test asserts nothing on prices (D288) and its report is written to the expected file.
- `TestScaleCheckpoint_SizeAndRoundTripDoNotDependOnTheWeights` (needs plan 2's `Checkpoint` and `RestoreWorld`): the same agent count at weights 1 and 40; after 20 ticks the structural counts of the two checkpoints are equal (the section counts, the number of agent records, lots, belief entries and welfare ring slots; not the byte length, because every number is a minimal-form LEB128 varint and the holdings and balances at weight 40 need longer varints, spec-checkpoint.md 5.2), and the scaled world's checkpoint restores and re-checkpoints byte-equal (FR-CHK-005). Requirement: NFR-SCL-004, NFR-CHK-002.
- The fork-at-N test of plan 2's harness is run once on the scaled `one` (scale 4) world; it is plan 2's test with this world added to its table, not a new test.
Exit check: S10's exact half passes; the report lines are recorded; `bands/one.toml` still parses (the bands loader test); `TestHashConfig_ReferencePinned` unchanged.

### Phase 5: the measurement

Goal: D289 to D293. Read-only on the library; the only files are the benchmark file and the measurement record.
Files: `scale_bench_test.go`, `testdata/chains/expected/one.scenarios.txt` unchanged by this phase.
Steps (run one at a time, on an otherwise idle machine, not under `-race` for time and under `-race -covermode=atomic -coverpkg=./...` only for the allocation count; other agents' jobs distort a time, so the record says whether the machine was idle):
1. Probe: `NewWorld` on `base`, `lines`, `five`, `warm-five` for 3 steps. Record whether it builds and steps. If yes, add `BenchmarkStepFive` (5 000 agents at scale 1, mean `|T_a|` 7.7 printed by the benchmark); if no, record the error and use the multiplied `one` of D291.
2. `go test -run '^$' -bench 'Step(Agents|Weights|Five)' -benchmem -count 6` and `benchstat` over the six; keep the median and spread.
3. The ratios of NFR-SCL-001 (time at 1 200 over 600 agents, 2 400 over 1 200) are written next to the 2.2 limit; the cost per agent-tick against `c_0 + |T_a| * c_t` of NFR-SCL-002; the 6% clearing share of spec-scale.md 6.3 is read from `BenchmarkPhase` (`observe_bench_internal_test.go:61`, phases P3 and P4).
4. The CPU profile of D292.
5. The record: a block comment at the top of `scale_bench_test.go` (machine, Go version, `GOMAXPROCS`, date, window, the live-agent log of D291, every figure) and the note update text below (Risks, item 5). No threshold is added to any test.
Tests: `BenchmarkStepAgents`, `BenchmarkStepWeights` (and `BenchmarkStepFive`). They are in `go vet` and compile under `-short`; they are not run by `go test`. A small `TestScaleBench_BuildsTheScaledWorlds` (in `scale_bench_test.go`, skipped under `-short`) builds the three sizes and the three weights once and asserts the agent and household counts, so a broken fixture fails CI instead of only the benchmark. Requirement: NFR-SCL-001, NFR-SCL-002, NFR-SCL-003, FR-SCL-016 (run half, reported).
Exit check: every figure of Phase 5 step 3 is in the record; the owed spec edit 2 is handed to the supervisor with the figures.

### Phase 6: exit gate

- Gates: scale tests that run in `-short` run in R; the S10 run half is gate F only (D286). Both R and F of plan-2-1 D229 pass before the row is done.
- `gofmt -l .` empty; `go vet ./...`; `staticcheck ./...` (no unused: every moved function has a caller, `agentCount` has two); `go mod tidy` with no diff.
- `go test -short -race ./...`; then R and F of plan-2-1 D229 (flags as defined there).
- `lizard -l go -C 10 -a 8 -L 100 -w scale.go scale_internal_test.go scale_pack_test.go scenario_s10_test.go scale_checkpoint_test.go scale_bench_test.go`.
- Coverage at least 90% for the root package, and `scale.go` at 100% of statements.
- Mutation: `gomu` on `scale.go` through the Docker container with `--cpus=4 --memory=16g --memory-swap=16g` and `-e GOMAXPROCS=4` (see `~/.claude/CLAUDE.md`), per file as the project's workflow does. There is no new `internal/` package, so there is no package-level pass. Equivalent mutants (for example the loop bound of the extra-household spread) are listed in the exit report with their reason.
- The registry and catalogue tests: no change (no configuration field, no event, no snapshot field).
- Proofs: `TestHashConfig_ReferencePinned` and `testdata/golden/off/` unchanged; `git diff --stat testdata/golden` is empty.

## Catalogue appends

| File | Append | Position, id or tag | Spec line |
| --- | --- | --- | --- |
| `config.go`, `bounds.go`, `defaults.go`, `validate.go`, `hash.go` | none. `Scale`, `MinAgents`, `Group.Households` and `Group.Scale` exist since Stage 1 (`config.go:65`, `68`, `444`, `447`; hash ids `hash.go:144`, `145`, `315`, `316`). `validate.go` loses `groupShape` (moved) | n/a | spec-config.md 3.1, 8.2 |
| `errors.go` | none | n/a | n/a |
| `event.go` | none. `AgentSpawned.Weight`, `AgentBankrupted.Weight` and `WeightChanged` exist (`event.go:608`, `626`, `681`) | n/a | spec-scale.md 9.1 |
| `option.go` | none | n/a | n/a |
| `tick.go` | none | n/a | n/a |
| `snapshot.go` | none. `Agents` and `Households` exist at world, region and role level | n/a | spec-scale.md 9.2 |
| `ledger.go` | none | n/a | n/a |
| checkpoint tags | none. The weight is saved once in the agent record, tag 3 (spec-checkpoint.md 4.2); this plan only tests it | tag 3 | spec-checkpoint.md 4.2 |
| log keys | none. `logkeys.Agents` and `logkeys.Households` exist and are on the `world created` line (`construct.go` `summary`) | n/a | spec-scale.md 9.3 |
| metrics | none. `economysim.agent.count` and `economysim.household.count` are the OBS plan's (plan 22) | n/a | spec-scale.md 9.4 |
| `testdata/chains/bands/one.toml` | `[s10.tolerance_bp]` and `[s10.dispersion_min_bp]`, `unit = "bp"`, no value (not a catalogue file) | after the `s19` keys | spec-reference-chain.md 7.1.5 |

Hash id and checkpoint tag: not applicable, because the plan adds no configuration field and no state.

## Cross-plan contract

Consumes:
- forkAtN, forkCase (checkpoint_fork_test.go) <- plan-2-2
- Checkpoint, RestoreWorld, Checkpoint.MarshalBinary (checkpoint.go) <- plan-2-2
- scenarioSeedCount, updateAllowed, gates R and F (scenario_helpers_test.go) <- plan-2-1
- runScenario, scenSem, checkBand (scenario_helpers_test.go) <- Stage 1
- `testdata/chains/overlay/trade-five.toml`, `Config.Edges` <- plan-2-3 (the Phase 5 `five` probe only; D291 fallback if absent)

Provides:
- agentCount, groupWeights, groupShape, checkCensus (scale.go) -> plan-2-11
- s10.tolerance_bp and s10.dispersion_min_bp, declared unset (testdata/chains/bands/one.toml) -> plan-2-19
- TestScenarioS10_PricesAcrossScales and its recorded report (scenario_s10_test.go) -> plan-2-8
- BenchmarkStepAgents and BenchmarkStepWeights (scale_bench_test.go) -> plan-2-24
- BenchmarkStepFive (scale_bench_test.go), present only if the Phase 5 probe passes (plan 3 is merged before this plan); otherwise a recorded note of the error and the multiplied `one` of D291 -> plan-2-24
- the first measured cost of a Step and the fitted c_0 and c_t (header record of scale_bench_test.go) -> plan-2-24


## Testing

Testing entry (testing.md 11):

| Requirement | Tests |
| --- | --- |
| FR-SCL-001 | `TestValidate_GroupShapeBounds` (rejection), `TestGroupWeights`, `TestGroupWeights_RejectsWhatValidationRejects`, `FuzzGroupWeights`; weight at least 1 in `TestCheckCensus_PanicsOnABrokenPlan` |
| FR-SCL-002 | `TestGroupWeights`, `TestAgentCount`, `TestGroupShape_AgreesWithGroupWeights`, `TestValidate_GroupWeightIsCeilOfHouseholdsOverAgents` |
| FR-SCL-003 | `TestGroupWeights`, `FuzzGroupWeights`, `TestScalePack_ConstructionTotalsAreExactAtEveryScale`, `TestScenarioS10_Exact` |
| FR-SCL-004 | `TestGroupWeights_DrawsNothing`, and the world creation draws test of Stage 1 (`FR-WLD-034`) unchanged |
| FR-SCL-005 | `TestGroupWeights`, `TestCheckCensus_PanicsOnABrokenPlan`, `TestScale_FloorAboveHouseholdsIsSilent` |
| FR-SCL-006 | `TestScale_DefaultsAndGroupOverride` (`scale_internal_test.go`: the default `Scale` is 1, `defaults.go:14`), existing `TestNewWorld_IDsAccountsAndEndowment` (`construct_pack_test.go:49`) plus a case in `TestScalePack_ConstructionTotalsAreExactAtEveryScale` with a group that gives no scale (the world's applies) and one that does |
| FR-SCL-007 | `TestConsume_Cohort`, `TestNewAgent_Weight` (relabelled), `TestScalePack_ConstructionTotalsAreExactAtEveryScale`, `TestScenarioS10_Exact` |
| FR-SCL-008, 009, 010 | the Stage 1 tests of D285, relabelled after the read in Phase 3; `TestScenarioS10_PricesAcrossScales` for 010 at run level |
| FR-SCL-011 | `TestValidate` rules at `validate_rules_test.go:751` unchanged |
| FR-SCL-012 | not this plan (plan 11) |
| FR-SCL-013 | not this plan (plan 17) |
| FR-SCL-014 | `TestExecute_RemoveAndReductionToTheWholeWeight` (reduction), the entry half (a rise by entry brings money from the pool) by `TestSpawner_EntryAndReduction` in the Stage 1 spawner tests if it asserts it, else a new `TestScale_EntryBringsMoneyFromThePool` in `scale_internal_test.go`; both read in Phase 3, `TestScalePack_HouseholdCountIsTheSumOfWeights` (conservation under `WithAudit`); split and merge are not in version 1 |
| FR-SCL-015 | `TestSnapshot_CountsAndScale`, `TestScalePack_HouseholdCountIsTheSumOfWeights` |
| FR-SCL-016 | `TestScalePack_ConstructionStateDoesNotDependOnTheWeights` (exact), `BenchmarkStepWeights` (reported) |
| FR-SCL-017 | `TestScenarioS10_PricesAcrossScales` (report; no pass claimed, D288) |
| FR-SCL-018 | `TestScalePack_AgentSpawnedWeightsAndSnapshotCounts`, `TestScale_BankruptcyCarriesTheWeight`, `TestScenarioS10_Exact` |
| FR-SCL-019 | `TestScalePack_ConstructionTotalsAreExactAtEveryScale`, `TestCheckCensus_PanicsOnABrokenPlan` |
| FR-SCL-020 | `TestValidate_GroupShapeBounds`, `TestAgentCount`, `TestScale_FloorAboveHouseholdsIsSilent` |
| NFR-SCL-001 | `BenchmarkStepAgents` (ratios reported, not asserted, testing.md 8.2) |
| NFR-SCL-002 | `BenchmarkStepAgents`, `BenchmarkStepFive` (reported) |
| NFR-SCL-003 | every benchmark of this plan calls `b.ReportAllocs()`; `TestScaleBench_BuildsTheScaledWorlds` keeps their fixtures building |
| NFR-SCL-004 | `TestScaleCheckpoint_SizeAndRoundTripDoNotDependOnTheWeights`, `TestScalePack_ConstructionStateDoesNotDependOnTheWeights` |

- **Golden files:** `testdata/golden/off/` does not move (the plan changes no behaviour); the pinned hash does not move. There is no module-alone golden and no `all` golden: SCL is core, not a module (architecture.md 2.10). A diff in either set is a defect of this plan.
- **Fork-at-N:** the scaled world is added to plan 2's fork table (Phase 4). SCL has no section and no state of its own.
- **Registry and catalogue rows:** none added and none deleted from `notBuilt`.
- **Fuzz:** `FuzzGroupWeights`. The configuration parser and `Validate` fuzzers of Stage 1 already cover `Scale` and `MinAgents` as fields.
- **Existing coverage this plan relies on:** `TestSnapshot_CountsAndScale` (scales 1 and 40), `TestTrade_CostAndDrawsDoNotDependOnTheWeight` (weights 1, 7 and `MaxWeight`), `TestDefaultStrategyTrade_CohortInvariance` (weight 7).

## Observability

Observability entry (observability-and-logging.md 9): None, because the plan adds no event, no `Warn`, no log key and no metric. It checks three things that exist: `AgentSpawned.Weight` and `AgentBankrupted.Weight` (FR-SCL-018, `TestScalePack_AgentSpawnedWeightsAndSnapshotCounts` and `TestScale_BankruptcyCarriesTheWeight`), the `agents` and `households` attributes of `world created` (FR-SCL-015, `TestScalePack_HouseholdCountIsTheSumOfWeights` reads the log through a capturing handler), and the snapshot counts. `checkCensus` logs nothing: it panics through the existing `invariant` helper, which is not a log site. The no-op path is unaffected (the allocation baseline `stepAllocBaseline = 113` must not change). The metrics `economysim.agent.count` and `economysim.household.count` are plan 22's (spec-scale.md 9.4).

## Risks and open items

1. **No bound on the number of agents (owner: CFG, spec-config.md 4.1 and bounds.go).** Weights are bounded (`MaxWeight` 10^6, `validate.go:1212`) and households per group (`MaxHouseholds` 10^9) but not agents: a group of 10^9 households at scale 1 passes `Validate` (the weight is 1) and `NewWorld` would allocate 10^9 agents. The same holds across groups and regions. SCL 4.5 and FR-SCL-011 name weight and quantity bounds only. Not decided here: the owner rules the bound (a world-level `MaxAgents`, or a work budget in the style of `MaxViabilityCells`) and this plan's `FuzzGroupWeights` stays bounded meanwhile. The plan does not add the check.
2. **FR-SCL-016 has no operational measure (owner: SCL spec).** Edit owed 3. The run half is reported, not asserted.
3. **The pack `one` collapses (N-312, N-314; plan 8).** Every run-level number of this plan (S10 prices, time per tick) describes a collapsing economy. They are labelled so and re-run by plans 8, 19 and 24.
4. **`five` may not step in Stage 1.** It has never been built as a world in a test (only solved and validated). If `NewWorld` rejects it or its regions do not clear, the first consumer's size is a multiple of `one` (D291) and plan 24 owns the real run after plan 3. The probe of Phase 5 settles which, and Phase 5 records the error text if it fails.
5. **Duplicate formula until Phase 2 lands.** `groupShape` and `groupWeights` are two copies today; Phase 2 removes the duplication, and `TestGroupShape_AgreesWithGroupWeights` keeps it removed.
6. **The measured constants are one machine.** Spec-scale.md 6.2 calls "the reference machine" without naming one. The record names its machine; the owner decides which machine is the reference before the spec's 6.2 is rewritten (edit owed 2).

**Note proposals (text only; the supervisor is the only editor of `doc/notes.md`):**
- N-300 update: "(plan 2-4, Phase 5) Measured: time per tick of `one` at 600, 1 200 and 2 400 agents, ns per agent-tick, allocations per Step, mean tradable set, and the live-agent count of the measured tick; the economy was collapsing (N-312). Time per tick is now recorded in `scale_bench_test.go`. Left open for the re-measure on the rebalanced pack (plan 8) and on `five` (plan 24)."
- N-222 update: "(plan 2-4, Phase 5) CPU profile of a Step at 2 400 agents: the keyed reseed and draw frames take X% of the step; one site call costs Y ns (`BenchmarkStream_KeyedDraw`). Recorded for ADR 0008's owner; whether a cheaper per-agent key is needed is not decided."
- New note (area SCL, CFG): "No bound on the number of agents: a group of `MaxHouseholds` households at scale 1 passes `Validate`. Rule a world-level agent bound or a work budget (spec-config.md 4.1, spec-scale.md 4.5)."
- New note (area SCL, STY): "`s10.tolerance_bp` and `s10.dispersion_min_bp` are declared unset in `bands/one.toml`; the pass of FR-SCL-017 waits for the rebalanced pack and a reviewed run (STY 8.4)."
- N-015 update: "(plan 2-4) `c_0`, `c_t` and the reference size 8 of `spec-scale.md` 6.2 have a first measurement (plan-2-4 Phase 5); they stay placeholders until the spec is edited."
