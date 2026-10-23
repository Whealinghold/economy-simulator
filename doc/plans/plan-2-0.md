# Plan: Stage 2, overview and the contract between the plans

**Status: contract written (2026-10-22). Every plan `plan-2-1.md` to `plan-2-24.md` is written against this file and checked against it and against the other plans.**
This file sequences [roadmap-v1.md](../roadmap-v1.md) Stage 2 into 24 plans: the roadmap Stage 2 table has 24 rows (plans 1 and 8 are now rows). It fixes what the plans must agree on. A plan decides nothing: the specs, the ADRs and `doc/rules/` decide, and where a plan and they disagree they win and the plan is corrected. Stage 1 is signed off ([roadmap-v1.md](../roadmap-v1.md), commit `6dd4f86`).

Rule tags read for this contract (the tags at the time of writing): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1. Every plan quotes the tags it read; each plan, when implemented, quotes the tags in the files then. A mismatch is a finding only if the file was not edited by an earlier plan's Phase 1 (the supervisor replaces a tag once per file per change). When the supervisor applies Phase 1 spec edits it follows `AGENTS.md` "Before writing or changing a spec": it reads `doc/notes.md` in full, quotes the Notes tag, and replaces it when it edits the notes.

## The plans, in build order

A plan may consume only from a plan with a lower number, from Stage 0 or 1, or from a spec; any other consumption is a finding. The order is the roadmap's, with two additions the owner delegated ("whatever order makes sense", 2026-10-22): plan 1 (the work before any row) and plan 8 (the rebalance of the reference pack, which N-314 owes once the first money-returning modules exist). The demand pull-forward of N-314 is honoured by DEM being the first module and the rebalance coming right after the pool part of EST.

| Plan | Row | New files and packages (roadmap Stage 2) | Specs | Needs (plans) |
| --- | --- | --- | --- | --- |
| 1 | Work before any row | no module code; the stale CHK and WLD passages (N-214), the race-aware scenario tests, ADR acceptances, the plans convention, CI decision, the roadmap/vision/spec-roadmap record | CHK, WLD, ADR 0002, 0009 | none |
| 2 | CHK | `internal/codec`, `checkpoint.go`, `testdata/checkpoints/` | CHK | 1 |
| 3 | REG | `internal/graph`, `regions.go` (the public `Relocate` and `SetEdgeStatuses` are plan 17) | REG | 1, 2 |
| 4 | SCL | `scale.go` | SCL | 1, 2, 3 (the `five` probe of Phase 5 only) |
| 5 | DEM | `demand.go`, the solver's two-phase start (N-277), `tradableItems` (N-228), `one-demand` pack | DEM, VIA | 2, 3, 4 |
| 6 | MON | `moneyflow.go`, `internal/moneyflow` | MON | 2, 3 |
| 7 | EST (module part) | `estates.go` appended: relief, the pool as a flow target | EST, CFG 6.2.4, MON 4.3.2 | 2, 6 |
| 8 | Reference pack rebalance | pack data and the warm start; goldens moved once; plus the report column `CensusRow.IncomeBP` | REF, VIA, STB | 5, 6, 7 |
| 9 | RES | `resources.go`, `internal/endowment`, `internal/curve` | RES | 3, 8 |
| 10 | SUB | `subsistence.go`, `internal/ownuse` | SUB | 9 |
| 11 | POP | `population.go`, `internal/population` | POP | 7, 9 |
| 12 | TRN with CAR | `transport.go`, `internal/transit`, plus the lane set in `viability.go` and `internal/steady` (`Lane`, `WithLanes`, `MaxLanes`, N-279, N-281) | TRN, CAR | 3 |
| 13 | INF | `information.go`, `internal/rumour` | INF | 3 |
| 14 | MIG | `migration.go`, `internal/migration` | MIG | 3, 13 |
| 15 | MER | `contract.go`, `internal/merchant`, `strategy/` | MER | 12, 13 |
| 16 | EXT | `external.go`, `internal/outside` | EXT | 3, 6 |
| 17 | INT | `integration.go` | INT | 2, 3, 6, 9, 11, 12, 16 |
| 18 | IND | `indicator/`, `internal/measure` | IND | 4, 17 |
| 19 | STY | `internal/verdict`, the `stylised_long` build tag | STY | 18 |
| 20 | EXP | `experiment/`, `internal/design`, `internal/reduce` | EXP | 2, 17, 18 |
| 21 | CLB | `internal/calibration`, `testdata/calibration/` | CLB | 18, 19, 20 |
| 22 | otelobserver | `otelobserver/` (own module), `go.work` | OBS 6, rules obs 5 and 6 | 2 to 16 |
| 23 | denomination | `denomination/` | CUR 4, 7 | 1 |
| 24 | Examples and the Stage 2 exit | `examples/<first-consumer>`, `examples/<second-setting>`, the `five.toml` stable run, two architectures, the SCL budget | vision 5, architecture 1.3 | all |

The "Needs" column is the minimum a plan may assume. Consumption from any lower plan through a Provides line is allowed; the column lists the minimum. A Needs entry that the plan does not consume is a build-order constraint only (plans 21 and 22), and plan 4's `BenchmarkStepFive` probe additionally needs plan 3 merged (plan 4 D291).

## What every plan must contain

One file, `doc/plans/plan-2-X.md`, at most about 450 lines (plans 1, 8 and 24 may be longer), in this order, with these headings:

1. **Header.** `# Plan: Stage 2, row X: <title>`, a **Status** line, the rule tags the author read (all three rule files are read in full, as `AGENTS.md` requires), the specs and requirement IDs implemented, and the decision-ID block of this plan.
2. **Scope.** What this plan builds (files, packages, public identifiers) and what it leaves out, with the owning plan or spec for each exclusion.
3. **Decisions.** Numbered `D<n>`, each with its source (a spec section, a rule number, an ADR, a note, or the owner's delegation of 2026-10-22). Plan X owns the block `D(200+20*X)` to `D(219+20*X)` (plan 1 owns D220 to D239, plan 2 D240 to D259, and so on). Never reuse a number of another block.
4. **Phases.** Each phase has a goal, the files it touches, the tests it adds (named, with the `Requirement:` IDs each serves), and an exit check. Phase 1 of a module plan lists **every edit owed to a spec, a rule or a note** (file and section), because the code must not start before its specs are true. The last phase is the exit gate (below).
5. **Catalogue appends.** A table with one row per catalogue file the plan touches (`config.go`, `bounds.go`, `defaults.go`, `validate.go`, `hash.go`, `errors.go`, `event.go`, `option.go`, `tick.go`, `snapshot.go`, `ledger.go`, checkpoint tags, log keys, metrics): what it appends, in what position, its hash field id or checkpoint tag, and the spec line that fixes the position.
6. **Cross-plan contract.** `Consumes` and `Provides` as bullet lists, one identifier per line, in the form `- <identifier> (<file>) <- plan-2-N` and `- <identifier> (<file>) -> plan-2-N`. A reviewer resolves every line against the other plans.
7. **Testing** (testing.md 11): the requirement-to-test table, the golden files (off unchanged, module alone, all-on when it moves), the fork-at-N test, the registry and catalogue rows, the fuzz targets.
8. **Observability** (observability-and-logging.md 9): events, `Warn`s, log keys, metrics, and "None, because ..." where it applies.
9. **Risks and open items.** Anything the plan could not settle, who owns it, and **note proposals** (text only: a plan never edits `doc/notes.md`; the supervisor is its only editor, N-103).

## Rules every plan follows

- **A plan edits one file, its own.** It does not edit a spec, a rule, a note, the roadmap, `AGENTS.md`, `doc.go`, `documentation_index.md` or another plan. Edits it needs there are listed in its Phase 1; the supervisor applies the shared ones. The status of the plan's `roadmap-v1.md` Stage 2 row moves to done, with the commit, at the plan's exit gate (`roadmap-v1.md` line 20); the supervisor applies it, so a plan need not repeat it in its Phase 1 table.
- **The module template is `architecture.md` 2.10.** One root file per module that owns every touch point; the catalogue files are append-only; `tick.go` is a straight line; no module file is created without its overlay, its module-alone golden, the proof that `testdata/golden/off/` did not move, its checkpoint section and fork-at-N test, its events in the catalogue tests and its rows in the registry test (`roadmap-v1.md` Stage 2).
- **Hash field ids, checkpoint tags and event positions are fixed by the specs, not by build order.** `spec-config.md` 3.1 and 8.2.2 fix the hash id of each module (counted from 1 in the `Modules` row, `Estates` the 25th); `spec-checkpoint.md` 4.2 fixes the tags; `spec-observability.md` 4.1 and `rules/observability-and-logging.md` 4.3 fix the event order. Where the build order differs from that order the plan says how the append rule (`architecture.md` 2.10.5) still holds.
- **A change that adds a module leaves no trace while it is off** (`architecture.md` 2.10.5, N-217): `testdata/golden/off/` and the pinned hash do not move. A plan that must move them says so and gives the testing.md 7.2 explanation. Only two moves are allowed: plan 2 moves `golden/off/one.txt` once (the digest line, no pin); plan 8 moves the golden sets of the packs it changes and the pins in `hash_test.go` and `config/integration_test.go`. No other plan moves either. Plan 8 also re-records `testdata/checkpoints/format-1-one.*` and, with the golden sets, the guard test of plan 2 that pins the previous `one.txt` (`TestGolden_OneOffMovedOnlyByItem6`, D371); plan 24 adds new files (`off/five*`, `golden/all/`, `format-1-all.*`) and moves none.
- **Gates for every row's last phase:** `gofmt -l .` empty, `go vet ./...`, `staticcheck ./...`, `go test -short -race ./...`, `go mod tidy` with no diff, `lizard -l go -C 10 -a 8 -L 100 -w` on the new files, markdownlint with `.markdownlint.json` on the files the row writes (plan-2-1 D238), coverage at least 90% for the root and for each new package, a mutation pass on the new `internal/` package (`gomu` only in Docker with `--cpus` and `--memory`, see `~/.claude/CLAUDE.md`), the registry and catalogue tests with the row's entries deleted from `notBuilt`. The full-run gate is gate R and gate F of `plan-2-1` D229, both passing.
- **Notes.** Each plan reads the notes index and every open note tagged with its spec's area code and `ALL` (the notes are not read in full: a plan changes no spec and no note), and says in its header which notes it read. For each it says apply, reject (with the reason) or leave open (with what it learned), as `AGENTS.md` asks of spec work.
- **Names are setting-agnostic** (`AGENTS.md` hard constraints). New glossary terms are listed for `doc.go` before any code uses them (architecture.md 3.2). Phase 1 also lists, for `doc.go`, every new exported type, event, row type and config struct of `value.go`, `config.go`, `event.go`, `snapshot.go`, `strategy.go`, `agent.go` and `world.go`, by exact name, because `TestGlossary_DeclaredTypesNamed` fails on an unnamed one.
- **Stage 0 guards.** Each module plan reads N-224 and updates the guards it names (glossary `notYetTerms` and `doc.go` terms, `deferredCore` and `notBuiltEvents`, the `validate_counts_test` and `errors_test` rows, `import_guard` and `guardedFiles` for a new directory) in the same change, and reads N-245 when a default rule reads a tunable.
- **No plan invents a requirement.** A behaviour with no requirement is a gap raised in the plan, not a test with a made-up ID (testing.md 4.4). A plan may list a proposed requirement text (and a proposed gate figure) in its Phase 1, marked "(proposed)". The supervisor applies it to the spec only after the owner approves it, as an owner action asked in plan 1 Phase 5; until then tests cite it as `(proposed)` and no plan relies on it as defined.

## Stage 2 exit, and which plan owns each part

`roadmap-v1.md` Stage 2 "Done when":

| Criterion | Owner |
| --- | --- |
| `five.toml` runs to a stable state from a warm start and passes the version 1 STY bands | plans 8, 19, 24 |
| the invariant, determinism, golden and configuration-only chain tests pass on two architectures | plan 24 (with plan 1 for the gate) |
| a new chain needs only a configuration file, shown by the second setting | plan 24 |
| a checkpoint taken mid-run and resumed is bit-identical | plan 2, and every module plan's fork-at-N test |
| the consumer example builds a complete server from the public API | plan 24 |
| the cost budget of SCL holds for the first consumer's world size | plans 4, 24 |

## Known facts the plans start from

- Stage 1 is signed off with these exceptions: the full `-race` run with `-coverpkg` does not finish in 2 hours (the scenario tests S04, S07, S19 and the determinism seeds), and pack `one` collapses (N-312, N-314). The goldens in `testdata/golden/off/` are recorded from that run.
- The demand module has a spec and a pack (`testdata/chains/demand.toml`, `one-demand.toml`, `warm-one-demand.toml`) and no code. The solver rejects a non-zero fixed term in a goods row until DEM adds the first phase (N-277).
- `doc/plans/` held the Stage 1 plans and was deleted after the stage; these plans are working documents of the same kind.
- Pack `one`'s census is the solver's capacity-balanced one. At the reference prices a grower earns about 45% of its upkeep and the other roles about 138% (N-314). Plan 8 owns that.

## Requirements deferred past Stage 2

No Stage 2 plan builds these. A plan cites such an ID only as "see plan-2-0 deferred table".

| ID | Spec | Reason | Owner |
| --- | --- | --- | --- |
| FR-IND-050 (second clause) | IND | the income fields (`IncomeLast`) are not in the snapshot until their module exists | the Stage 3 plan that adds `IncomeLast` |
| FR-IND-056 to 059 | IND | presets for rows 22 to 25 (unemployment, wage bill, employment, losses) read the employment and risk modules | Stage 3 plans of EMP, RSK and STO |
| FR-STY-058 (STY-W07) | STY | the unemployment share and the employers' margin need the employment module | Stage 3 EMP plan |
| FR-MER-064 | MER | caps and floors need the institutions module | Stage 3 institutions plan |
| FR-CHK-061 | CHK | reading every earlier format version binds only from 1.0 | the 1.0 release |
| FR-OBS-021 | OBS | traineeship event order needs the skills module | Stage 3 SKL plan |
| FR-CAR-005 to 008 | CAR | superseded; see FR-TRN-009, 010, 012 and the cost anchor (FR-PRO-035 to 037) | none (closed) |
| Open layer of the steady state (import and export columns, trade row) | VIA 4.8, EXT 12.1, N-114 | spec-viability 4.8 says it is not written; no requirement ID | a later VIA change, or an owner ruling that adds it to plan 16 |
| FR-MON-038 (stores and treasury targets), FR-MON-039 (`store_levy_below_upkeep` warning only) | MON | the stores target needs the storage module and the treasury target the fiscal and polity modules; plan 6 rejects both with `ErrModuleDependency` (D320) and raises the other six warnings | Stage 3 STO plan (stores, the warning) and FIS and POL plans (treasury) |

## Owner decisions the plans depend on

Collected from the plans by a synthesis on 2026-10-22 (before the last two fix passes, so a wording may have moved; the plans are the source of truth). **None of these is decided.** A plan may rely on the recommended answer if it marks the dependency; it must not assume a spec or rule has already changed. "Blocks" says what waits on the answer.

| ID | Question | Plans | Recommendation | Blocks |
| --- | --- | --- | --- | --- |
| OD-01 | CI and arm64: add a CI workflow (N-019) and provide an arm64 runner or a working binfmt/QEMU emulation (not installed on this host)? | 1, 20, 24 | Yes to both. Add the workflow of plan 1 Phase 5 with an arm64 job (hosted arm64 runner if available, else QEMU nightly). Without it NFR-WLD-004 and Stage 2 exit criterion 2 cannot be shown. | plan 24 exit (two architectures), plan 20 golden two-arch run, plan 18 TestGolden_IndicatorPresetsOnOne two-arch |
| OD-02 | Agent-lifecycle fan-ins (agentJoined, agentRoleChanged, drop) and the new ADR 0010 narrowing ADR 0007 decisions 1 to 3 (plan 5 E1, E11, O5). Accept, or take plan 5's fallback (a formed flag per agent in tag 12)? | 5, 10, 11, 14, 17 | Accept ADR 0010 and the fan-ins now. Plans 10, 11, 14 and 17 are written against them and the fallback changes tag 12 and the moment a carry is formed (FR-DEM-013). Accept before plan 5 Phase 6. | plan 5 Phase 6; plans 10, 11, 14, 17 consume agentJoined/agentRoleChanged |
| OD-03 | CFG 8.2.1: is a field a module adds to a core struct a pair only when non-zero (first bullet, FR-CFG-104) or whatever its value when the module is on (last bullet)? May such a field default to a non-zero value (PoolRelease 100, ReliefBelow 10 000)? | 7, 13, 14, 15 | Keep the first bullet (non-zero only), edit the last bullet, allow non-zero module defaults. Row 4 of 8.2.4 and every pinned digest stay. Plans 13 (Role.Informed 19), 14 (Role.Fixed 20) and 15 (Role.MerchantFeeShare 21, Role.Merchant 22) hash on the same rule. | plan 7 Phase 1 item 3 and its pinned row; hash vectors of plans 13 to 15 |
| OD-04 | Count over a bound: ErrTooMany (CFG 7.1.6) or ErrOutOfBounds (DEM 10.1, MON 12.1)? | 5, 6 | ErrTooMany for every count checked in counts(), as both plans default; edit DEM 10.1, MON 12.1 and FR-MON-034. | validate_counts_test rows of plans 5 and 6 |
| OD-05 | DEM Want_c: planned units (plan 5 default) or funded units (DEM 4.6 text)? And the wants programme infeasible at lambda = 0 (VIA silent): warning wants_exceed_labour with fallback to the upkeep-only census? | 5, 8 | Planned units in Keep/Target, funded units are what tier 4 buys; infeasible case becomes the warning with SurplusWantsBP absent. Edit DEM 4.6, AGT 7.3.5, VIA 10.3. | plan 5 Phases 5 to 8; plan 8 re-solve of one-demand |
| OD-06 | Plan 8 rebalance: allow the seasonal-crop feature C04 to move from the pack to an overlay (E0)? If the data rungs fail, which of E1 (stability claim rides on one-demand; amend FR-REF-018, C09, VIA 4.7 and the Stage 2 exit wording), E2 (pool relief in the pack) or E3 (carry exit criterion 1) applies? Confirm the screening gate (lowest IncomeBP 8 000, highest at most 1.5 times the lowest) and make it FR-REF-041. | 8, 9, 10, 11, 16, 19, 24 | E0 yes, asked in Phase 1 before any trial. On a stop choose E1. Confirm the gate figures. Plans 9 onward start on the unchanged packs if plan 8 stops without acceptance. | plan 8 Phase 2 onward; plan 19 STY-G01 on one; plan 24 exit claim; plans 9 to 17 overlays (capacities re-derived after acceptance) |
| OD-07 | Number of agents has no bound (a group of MaxHouseholds at scale 1 passes Validate). Add a world-level MaxAgents or a work budget? | 4, 24 | Add MaxAgents (derived bound on the sum of agentCount, ErrTooMany) in spec-config 4.1 and spec-scale 4.5; plan 4 then adds the check. Until ruled plan 4 adds none. | nothing hard (plan 4 FuzzGroupWeights stays bounded) |
| OD-08 | Which machine is the reference machine of spec-scale 6.2, and what is the first consumer's tick rate (vision 6)? | 4, 24 | Name the CI/dev machine used by plan 4's record as the reference until the consumer states a rate; report, do not assert, against the 53 ms placeholder. | plan 4 edit 2 (rewrite of 6.2); plan 24 Phase 6 verdict |
| OD-09 | Subsistence home baking: keep the source-only own-use rule (a stand-in source for prepared in the overlay) or adopt the closure rule (inputs only from non-discard outputs of the same list, acyclic)? | 10, 19, 24 | Keep source-only for Stage 2 (plan 10 D401), record the stand-in as a test double, and raise the closure rule as a note; it costs a phase and weakens the floor-under-the-market property. | plan 10 S56 (fewer unfed households), overlay own-use.toml |
| OD-10 | Initial agents' RoleSince (warm start) is the creation tick 0, so no household is settled before MoveCooldown/T_a (60 ticks) and the first migration is delayed in every pack. Allow a negative RoleSince at construction? | 14 | Yes: set RoleSince = -MoveCooldown for warm-start agents (spec-migration 5.2.2 allows negative); edit SPN 14 and WLD 5. | plan 14 Phase 2 and S61 to S66 timing |
| OD-11 | Rounding of a negative net_r and of a source recipe with tool wear (MIG 6.3): floor (spec) or truncation toward zero (architecture 3.5)? | 14 | Truncate toward zero per architecture 3.5 and edit MIG 6.3; the MIG owner confirms before Phase 2. | plan 14 Phase 2 |
| OD-12 | EXT side-state key (R1): one capacity/window/cap per side of a line (EXT 3.3) against one counterparty per (node, gate, side) (4.2.1). Key the state per gate and side and restate MaxLineTotal? | 16, 17, 24 | Key by (node, line, gate, side), restate MaxLineTotal as MaxMoney / (2 * 16 * 64 * gates), edit EXT 7, 13.2, 16, CHK tag 18 and OBS 5.3.9. Decide before plan 16 Phase 4. | plan 16 Phase 4 onward; tag 18 layout and golden/external |
| OD-13 | Is Lines() (FR-EXT-047) public, and under what name (INT/VIA owner)? | 16, 17 | Keep it unexported in Stage 2 (as plan 16 does); export later with the open layer of VIA. | nothing in Stage 2 |
| OD-14 | Experiment document cap: config.Parse takes 16 documents and EXP makes one per change, so MaxFactors 16 and CLB's screen of every knob cannot run. Raise MaxDocuments, let a level share one document, or lower MaxFactors? Also confirm the always-on PairingGain reading of ADR 0008 decision 6 (Spec.Pair removed). | 20, 21 | Raise MaxDocuments to 64 (CFG 4.1) and keep MaxFactors 16; confirm Spec.Pair removal now (removing it later breaks the API). Plan 21 stage split uses the lower cap until ruled. | plan 20 Validate bound; plan 21 stage C4a split |
| OD-15 | internal/checked import for platform packages: plan 18 (internal/measure), plan 20 (experiment) and plan 21 (internal/calibration) import it and edit their specs; plan 19 keeps internal/verdict stdlib only (O4, D587). | 18, 19, 20, 21 | One ruling for all four: internal platform packages may import internal/checked (a leaf; architecture 3.5 wins over NFR-IND-004, NFR-STY-004, NFR-CLB-003). Plan 19 then uses checked and drops the constants-only range check. | plan 19 Phase 2 (D587); consistency of the three spec edits |
| OD-16 | Who reviews a first setting run (STY 8.4.5 says a human), and where does the nightly / tier L schedule live? | 19, 24 | The repository owner reviews; an agent may prepare the proposal and, on the owner's explicit instruction, edit value lines in the reviewed commit. Put the schedule in the CI workflow of plan 1 Phase 5. | plan 19 Phase 9; plan 24 Phase 6a hold-out |
| OD-17 | Plan 24 content choices: realm pack is five renamed (D682) and colony is the fixture setting (D687) rather than newly authored packs; wording of the exit record ('passes the X4 predicate and the version 1 bands in the recorded run', not 'stable'). | 24, 8, 19 | Accept both and the wording (N-027, STY 7.3.1). A newly authored RPG pack is a later REF change. | plan 24 Phase 3 |
| OD-18 | Flow-ID cardinality for economysim.reason: add Names.Flows []string or state the cap (MaxMoneyFlows and the adapter intern cap) as the rule? | 6, 22 | State the cap (plan 6 edit 11, no Names.Flows): fewer API additions. Edit OBS 6.5.1 and 9.4.1. | plan 22 cardinality test (Phase 9) |
| OD-19 | Gauge whose row disappears (stale last point) in the OTel adapter: record 0 or accept the stale point? Also confirm OpenTelemetry v1.38.0 is a real release older than two weeks, and whether the workspace changes the root gates (GOWORK=off). | 22 | Accept the stale point and document it in OBS 6.2.2; plan 22 Phase 1 confirms the version and measures go.work effect; add GOWORK=off for the core gates in AGENTS Commands if needed. | plan 22 Phases 1 and 8 |
| OD-20 | denomination/: (a) the RPG preset vocabulary (cp, sp, gp) allowed outside examples/ as an exception in AGENTS.md; (b) Format takes int64 not Money; (c) step 5 at p > 0 prints 0.00 cp; (d) indicator/ and experiment/ may import denomination/; (e) its two bounds stay beside the code, not in bounds.go. | 23, 24 | Accept (a) to (e) as plan 23 recommends; (a) is the only AGENTS.md hard-constraint change, so confirm it explicitly. | plan 23 Phase 1; examples/realm formatting |
| OD-21 | Plan 1 timing and CHK/WLD ruling: confirm D225 (WLD 12.3 and ADR 0008 resolve CHK 5.4.1), the reduced-seed start value raceSeeds = 2, and accept ADR 0002 and ADR 0009. | 1, 2 | Confirm D225, accept ADR 0002 and ADR 0009 first (before plan 2 Phase 4), start raceSeeds at 2 and let Phase 2 step 1 measure. | plan 2 Phase 4; plan 1 gate R/F |
| OD-22 | Plan 15 spec choices: place of Outbound (TRN 9.1 on every view vs MER 4.1 in the merchant group); adopt or reject the welfare window of MER 12.3 (N-099); whether SolveSteadyState accepts a no-recipe merchant role in a pack. | 15, 12, 8 | Outbound on every view (TRN); adopt the window (ten-line change, otherwise long trips read as starvation); if the solver rejects a no-recipe role apply the overlay after the warm start. | plan 15 Phases 2 to 5 |
| OD-23 | VIA answers for plan 12 Phase 8 (lane programme: per-region prices from one programme or one per component, sentinels) and the AGT/INT row-type names and query signatures (E1, E2). | 12, 17, 24 | One programme per component, sentinels as plan 12 E5; adopt plan 12's row names. If VIA defers, split Phase 8 out and leave N-279/N-281 open. | plan 12 Phase 8 (TransportLanes, WithLanes); plan 24 lanes |
| OD-24 | Plan 17: agent.live as an agent field (core AGT state) rather than w.tk.live[slot]; RemoveAgent can fail on the estate balance bound (INT 4.4 says never); a call-script golden directory. | 17 | Keep agent.live (a Directive is core). Name the ErrOverflow probe case in INT 4.4. No testdata/golden/calls/: compare runs. | plan 17 Phases 3 and 9 |
| OD-25 | POP: do unfunded households of births count in EntryLimited.Unfunded and the 'entry unfunded' Warn (SPN owner)? Margin of no_fixed_factor? | 11, 19 | Yes, they count (plan 11 D429); first-level test for no_fixed_factor until VIA gives the margin. | plan 11 entryRegion; plan 19 consumes BirthsUnfunded |
| OD-26 | Where do cross-cutting guarded tick.go calls go once positions/positionsTail hit the Lizard CCN limit (found in this review, plan 14 O7)? | 5, 6, 7, 9, 10, 12, 13, 14, 15, 16, 17 | Plan 5 splits the straight line into one short function per phase before the first guarded call and updates the position test to read them in source order; later plans append into their phase function. | plan 5 Phase 6 and every later tick.go touch |
