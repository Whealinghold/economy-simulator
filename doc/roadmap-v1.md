# Delivery order of the code

The order in which the code is built, stage by stage: which files and packages
each stage creates (their homes are `doc/rules/architecture.md` 2.6, the
module-file rule its 2.10), which specs it implements, and the tests that must
pass before the next stage starts. It is the plan; the rules are the law. Where
this file and `doc/rules/` disagree, the rules win and this file is corrected.

The staging is [ADR 0003](adr/0003-version-1-scope.md) (`vision.md` section 4):
the slice, version 1, later. The structure of the code is
[ADR 0007](adr/0007-a-module-is-one-file-that-owns-its-touch-points.md) and the
random stream [ADR 0008](adr/0008-random-draws-are-keyed-streams.md). Every
stage ends with a golden file that runs; no stage starts a module file before
the slice's golden runs (architecture.md 2.10.6).

Status values, in the order a row moves through them: **not started** (no plan),
**plan written** (a plan exists in `plans/`, not yet audited ready), **plan audited**
(the plan passed its audit with no blocking issue, ready to implement), **in
progress**, **done** (with the commit). Stages 0, 2 and 3 still use **planned** for
a row that has no plan. Update the status in the change that moves it.

## Stage 0: foundations (no economics)

Everything the slice needs that is not a rule of the economy. Nothing here
draws, trades or produces: the world does not exist yet, so a file that serves
a world later (`random.go`, the events, the options) is built and tested alone.
The implementation plan was `plan-phase-0-foundation.md` (git history, `2abbe62^`); the
decisions it rests on (the owner's, 2026-10-12) are:

- **D2.** Stage 0 ships `WithLogger`, `WithObserver`, `WithAudit`,
  `WithTargetSpread` and `WithSeed`. The registration options `WithStrategy`,
  `WithSpawner` and `WithClearing` ship in Stage 1 with their interfaces.
- **D3.** Slice fields only: a struct or an event declares the fields the specs
  give the core, that is every field no module of `Config.Modules` owns,
  including every key the packs `one` and `second` set. Each module's change
  appends its own fields at the end, never inserts (architecture.md 2.10.5).
  `Modules` is declared with no field.
- **D4.** The five viability rules of CFG 7.6, `ViabilityError` and the
  solver-dependent warm-start checks ship with `internal/steady` in Stage 1.
- **D5.** An append that is off leaves no trace: a module that is off, and an
  appended field at its off value, contribute nothing to the configuration hash
  (CFG 8.2) or to the golden rendering of an event (OBS 9.5.2), so
  `testdata/golden/off/` does not move when a module is added.

| Create | Specs | Status |
|--------|-------|--------|
| `go.mod` (`go 1.25`, no dependency until `config/`); `doc.go` (the package name `economysim`, the overview and the glossary of the terms Stage 0 uses; every later stage adds the terms it uses before it uses them: architecture.md 3.2, N-017) | architecture.md 2.2, 3.2 | done (023cf97) |
| `value.go` (`Money`, `Price`, `Quantity`, the typed IDs with `NoRegion`, `NoAgent` and `NoCommodity`: arithmetic, comparison and formatting only) | CUR, CFG 3.3 | done (023cf97) |
| `internal/checked` (the 128-bit product and ratio), `internal/ordering` (the finaliser, `Weyl32`, `Weyl2`, `Bucket`, `Key` in the `+gamma` form, the stream key with its optional fourth word), `internal/logkeys` (the key strings of the slice's log lines only) | WLD 7, 9, 10; RES 5.3.3; OBS 7 | done (023cf97) |
| `random.go`: the seed, the one PCG and `Rand`, the site constants of the slice (WLD 7.1.4), the keyed reseed, the world's draw (WLD 7.2.1) and the **pure** reseed function `seed' = finalise(seed xor finalise(salt))` (CHK 6.8.2). The option `WithReseed`, the restore step and `WorldRestored` are Stage 2 (`checkpoint.go`) | WLD 7 (as rewritten under N-214), CHK 6.8.2, ADR 0008 | done (023cf97) |
| `config.go` (`Config` and every section struct of the slice, slice fields only (D3); the `WarmStart` types live here, not in `viability.go`), `bounds.go`, `defaults.go`, `validate.go` (the framework, the structural rules of CFG 7.3, the contributed rules of 7.4.1 less the three name checks and the viability rules, which are Stage 1 (D2, D4), and the producibility fixpoint of 7.5), `hash.go` (`HashConfig`, the encoding of CFG 8.2 with its version byte and rule D5), `errors.go` (a sentinel with its first user). Sections: world, calendar, commodities, recipes, yields, roles, regions as one region, groups, warm start; `Modules` present and all nil | CFG 3, 4, 5, 7, 8, 9; GDS 8; PRO 11; AGT 12; BEL 11; SPN 11; WEL 4.2 and 12; SCL 3.1; CAL 3; VIA 8.1 | done (023cf97) |
| `event.go` (the sealed `Event`, `Side`, the events no module owns, slice fields only (D3), `Names`, `Amount`, `HaltRow`, and the `Reason`, `Cause` and `Kind` constants the slice can reach) and `observer.go` (`Observer`, `ObserverFunc`, the unexported `discardObserver` with its assertion, the delivery loop, the "observer supplied" flag, `warnRun`); `option.go` with `Option`, `WithLogger`, `WithObserver`, `WithAudit`, `WithTargetSpread` and `WithSeed` (the registration options are Stage 1) | OBS 3, 4, 8.1; WLD 4.1; WEL 5.2; `rules/observability-and-logging.md` 4 | done (023cf97) |
| `config/`: the TOML loader (the library is [ADR 0009](adr/0009-the-toml-library-is-pelletier-go-toml-v2.md), `pelletier/go-toml/v2`; the tree is mapped by hand, unknown keys are rejected by `config/` itself), the limits on untrusted input (CFG 6.3.1), pack layering | CFG 6 | done (023cf97) |

**Done when:** `FuzzLoad` and `FuzzValidate` run with seed corpora; every
validation rule of CFG 7 for the slice's sections that is not deferred (D2, D4)
has a failing-case test, and the PR that builds `validate.go` lists each rule row
as built, deferred to Stage 1 or module; "every result-affecting field moves the
hash" passes, with the allow-list of exactly the presentation-only fields of the
slice, and the pinned hash of the all-off reference configuration is committed
(FR-CFG-105); the parameter-registry test (testing.md 6.7) passes both ways in
its staged form (a `notBuilt` list, architecture.md 9.7); the keying test vectors
of WLD 7.1.5 and the reseed vectors of CHK 6.8 pass; the events of the slice
pass the shape test and the staged catalogue test (FR-OBS-001, 003, 004, 006:
`Tick` first, no pointer, map or float, the fields of 4.1 in order, every
declared event a row and every row declared or listed); the delivery loop
delivers to every observer in the order given, a nil observer is ignored and
delivery with no observer allocates nothing; the options pass from an internal
test (each option's effect, the bound and error of `WithTargetSpread`, the last
`WithSeed` wins, the default seed 0); a reseed followed by a draw allocates
nothing; the `Guard:` test of `internal/logkeys` (every constant is a row of
`rules/observability-and-logging.md` 3.3.2) and the `Guard:` text search for the
forbidden calls (architecture.md 7.1 to 7.3, 11.1, 11.2) pass; `doc.go` holds
the package name and every term Stage 0 uses; `go vet`, `staticcheck` and
`go mod tidy` are clean.

## Stage 1: the slice

One region, the reference chain, clearing, beliefs, the default strategy and
spawner, the stability checks (`vision.md` section 4, "Slice"). The order below
is the dependency order of `spec-roadmap.md` "Tier 1 dependencies".

The first row, the ledger, is built ahead of the others at the owner's request
(2026-10-13). Its implementation plan was `plan-phase-1-1.md` (git history, `2abbe62^`); it adds the claims list and the
balance sheet of LED 9.2 and 10 to the row (decision D8), so the checkpoint and the
balance sheets do not change shape when credit arrives (`vision.md` section 4). It
leaves out the room of a limited account (Stage 3, `storage.go`), `Relocate` and
the relocation totals (Stage 2, `regions.go`) and the encoding of the ledger's state
(Stage 2, `checkpoint.go`); `spec-ledger.md` 16.1 lists them.

Rows 2 to 6 were planned on 2026-10-13, one plan each (the Plan column). Notes on
what each adds beyond the table cell:

- Row 2: Stage 0 already built the structs, defaults, bounds and validation of the
  four concerns, so the row adds only the derived tables and rules (the moment,
  `SeasonAt` and the day table; the load and the chargeable size; the recipe table
  and the makers of a commodity; the role table with the tradable set and the
  effective upkeep).
- Row 3: `order.go` (`Order`, `Trade`, `LotKey`) moved here from the world row
  (decision D30), because `Clearing` cannot exist without them; `market.go` stays
  with the world row.
- Row 4: no configuration field is added (Stage 0 built all sixteen parameters and
  their validation); the root holds the exported `Belief` and `Bounds`.
- Row 5: emits no event and calls no other row; its Phase 2 adds two additive ledger
  operations (`Discard` and `Holding`) in new files of `internal/ledger`.
- Row 6: the role economics report of WEL 7 is the viability row's and in-kind pay
  (WEL 6) is not built.
- Row 6: `internal/welfare` takes the weight once, in `TickCost`, and values wear at
  P6.3 (`WearCost`, one ceiling per execution); every tick-called function
  saturates and has no `ok` (D63), `NextRun` holds the saturating run counter of the
  two `Warn`s, and the holder of the ring is the agent row. Coverage: root 99.2%,
  `internal/welfare` 98.1%.

Rows 7 to 12 were planned on 2026-10-17, one plan each (the Plan column), in
parallel and then checked against each other. Notes on what each adds beyond the
table cell:

- Row 7: adds `internal/household` (the pure rules of consumption, the targets,
  the tiers, the spend cap and the work side) and builds the unexported `agent`
  with the fields no module owns (a module's per-agent state is in the module's
  slices, N-244). The belief parameters are an exported group of both views
  (D82). The row gives the world per-agent functions (`newAgent`, `consume`,
  `fillCore`, the work report), not the loops. `Directed` has no `SaveState` and
  `LoadState` until the checkpoint row (N-247).
- Row 8: `estates.go` is a core file with no `Modules` pointer; the module half
  (relief, the pool as a money-flow target) is appended after the slice's golden
  runs. `GoodsListed` counts lots (coordinator ruling, D101). The estate has no
  snapshot row and no marker (FR-EST-010 is reworded). The checkpoint writes the
  listing rows of held lots only (N-256).
- Row 9: adds `internal/occupation` and `spawner_report.go`. It builds the
  `Spawner` surface, the default `Decide` and the pure support; the EXECUTOR of
  the plan is the world row's. Only the tool half of SKL is built, so a role with
  `TrainingTicks` above 0 is never entered (N-264).
- Row 10: adds `viability_check.go` and `viability_table.go`, the bounds
  `MaxViabilityPivots` and `MaxViabilityCells` (`MaxLanes` comes with `WithLanes` in Stage 2), and the sentinels
  `ErrNotViable`, `ErrNoUpkeep`, `ErrSolveLimit` and `ErrSolveInconsistent`. Its
  Phase 5 PR flips the d1 to d5 line of `TestPacks_DefectOverlays` (N-299), and
  the shape of `SteadyState` is what row 12's golden header reads.
- Row 11: renames the configuration section type `World` to `WorldParams`
  (Phase 2, N-284), adds `construct.go`, `construct_warm.go`, `fanin.go` and the `phase_*.go` files
  (N-287) and owns the spawner's executor and `removeAgent`. The position table
  test is staged (`notBuiltPositions`, N-285). No hook in `Step`: every position
  is an unexported method and `Step` is `beginStep`, the straight line, `endStep`
  (N-286). It owns the observability tests that need a world and no pack.
- Row 12: test code, fixtures and documents only, no production code. Besides the
  golden files (`testdata/golden/off/one.txt`, `events-one.txt`,
  `one.positions.txt`; no checkpoint digest until Stage 2, N-294) it owns the
  determinism matrix, the invariants over seeds, the configuration-only chain on
  `second`, the d1 to d6 file tests, the scenarios S01, S04, S07, S16 and S19, and
  the pack halves of the observability tests (cardinality and the allocation limit
  of a full `Step`), the FR-REF-007 byte test of the `warm-*.toml` with its TOML
  writer, and `expected/one.toml` and `expected/second.toml`.

| Create | Specs | Plan | Status |
|--------|-------|------|--------|
| `internal/ledger` (accounts, the transfer primitive, reasons, lots, reservations, `Transform`, `Wear`, ageing, flow totals, the audit, the claims list with its six operations and the balance sheet) and `ledger.go` (the conversion of the configuration, the reason names both ways and the `BoundaryFlow` rows) | LED; CRD 3.1 to 3.3, 3.6 | 1-1 (git history, `2abbe62^`), 7 phases | done (262a0d1) |
| `commodity.go`, `recipe.go`, `role.go`, `calendar.go` (the derived tables and rules of the structs Stage 0 built in `config.go`: the moment, `SeasonAt` and the day table; the load and the chargeable size; the recipe table and the makers; the role table with the tradable set and the effective upkeep) | GDS, PRO 3, AGT 3, CAL | 1-2 (git history, `2abbe62^`), 6 phases | done (10fc5fa) |
| `internal/clearing`, `order.go` (`Order`, `Trade`, `LotKey`) and `clearing.go` (`Clearing`, `AverageClearing`, admission, reservation, settlement, the market report type and its computation) with `WithClearing`, and the name check `ErrUnknownClearing` | CLR; WLD 4.1; CFG 7.4.1 | 1-3 (git history, `2abbe62^`), 6 phases | done (481d288) |
| `internal/beliefs` and `beliefs.go` | BEL | 1-4 (git history, `2abbe62^`), 6 phases | done (1dfda90) |
| `internal/production` and `production.go` (jobs, yield table, the shock slot and the cost-anchor table, `UnitCost`) | PRO | 1-5 (git history, `2abbe62^`), 8 phases | done (597479c) |
| `internal/welfare` and `welfare.go` (the record, the ratio, the observed wage) | WEL | 1-6 (git history, `2abbe62^`), 5 phases | done (cc7f74c) |
| `agent.go`, `strategy.go` (`Strategy`, `AgentView`, `TradePlan`, `JobRequest`, the default strategy, `Directed`) with `WithStrategy`, and the name check `ErrUnknownStrategy`; `Side` is already in `event.go` (Stage 0). Requirements of this row that the tables of row 2 do not cover (N-236): the weight multiplication of the effective upkeep (FR-AGT-011), the "requests no job" half of FR-AGT-010 and, with the demand module, the want items of FR-AGT-003. The row also builds `internal/household` (the pure rules), the agent's creation and consumption step, the core view filler and the work report the world calls (plan 1-7 D80, D85) | AGT | 1-7 (git history, `2abbe62^`), 8 phases | done (99f383e) |
| `estates.go`: the estate pseudo-agent and account, the listings, and the pool with its use to fund an entrant. The `Estates` module part (relief, and the pool as a money-flow target) is appended after the slice's golden runs, in Stage 2 (CFG 6.2.4, architecture.md 2.6) | EST | 1-8 (git history, `2abbe62^`), 5 phases | done (5b4d864) |
| `spawner.go` (`Spawner`, the ladder, the entry rule, the default) with `WithSpawner`, and the name check `ErrUnknownSpawner` | SPN | 1-9 (git history, `2abbe62^`), 7 phases | done (881abd5) |
| `internal/steady` (the exact solver, `math/big`) and `viability.go` (`SolveSteadyState` and `virtualStartOffsets`; the `WarmStart` types stay in `config.go`), with the five viability rules of CFG 7.6, `ViabilityError` and the solver-dependent warm-start checks in `validate.go` (D4: FR-CFG-033, 041 to 044) | VIA; CFG 7.4.1, 7.6 | 1-10 (git history, `2abbe62^`), 7 phases | done (c603a08) |
| `world.go` (`World`, `NewWorld`, `Step`, `Run`, the guards, the four fan-ins built at Stage 1; `tradableItems`, the fifth, comes with the first module PR), `tick.go` with the core positions and the position table, `market.go` (the state of each market's report and the price window; `order.go` is the clearing row's), `snapshot.go` | WLD 3, 5, 6, 8, 9; OBS 5 | 1-11 (git history, `2abbe62^`), 11 phases | done (4153fdd) |
| `testdata/golden/off/` for `one.toml` (and `warm-one.toml`); the per-position hash helper | REF 8.5, FR-REF-028 to 030, testing.md 7.1 | 1-12 (git history, `2abbe62^`), 9 phases | done (7f9e006) |

**Done when:** the golden of `one.toml` runs for the length REF fixes and
passes; the invariants of testing.md 6.3 hold over the seed range of REF 8;
the same seed gives the same bytes with and without a logger and an observer,
split into any number of `Run` calls; the position table test passes
(testing.md 6.8); the configuration-only chain test passes on
`testdata/chains/second/`; the viability rules have failing-case tests, the six
overlays of `testdata/chains/defects/` load over the base layers REF names for
each and fail the rule or give the warning REF assigns, and the registration
options reject a name given twice and a nil extension; the first stability
scenarios of STB that need one region only (S01, S04, S07, S16, S19) run and
their outcomes are recorded, pass or not. The observability tests that need a
world (`rules/observability-and-logging.md` 8, OBS 9), which Stage 0 could not
run, pass: logs (the level, message and keys of every `Warn` and `Error` path,
and no `Info` in the tick loop), non-interference (FR-CFG-026 world half,
FR-OBS-023), cardinality (FR-OBS-005 over `one`), the counterpart of every
`Warn`, the golden rendering of events (`testdata/golden/off/events-one.txt`, the
zero-field rule of OBS 9.5.2), the emission order of a scripted step, the run
lifecycle (`RunStarted` and `RunEnded`, also when cancelled or failed), the
catalogue test over the declared events with the world emitting each of them,
the option halves that needed a world (FR-WLD-003, FR-OBS-090), and the two
allocation limits of the no-op path: every emission and logging call site of
the tick loop allocates nothing, and a full `Step` stays within its baseline
(observability-and-logging.md 8.2; NFR-OBS-001, NFR-WLD-001). The slice may
move specs between stages (ADR 0003); record any move here and in `vision.md`.

**Signed off (2026-10-22), with these stated exceptions.** Every row above is done. `gofmt`, `go vet`, `staticcheck`, `go mod tidy` and `go test -short -race ./...` are clean, and the coverage run without `-race` passes. The full command with `-race` and `-coverpkg` did not finish: the root package timed out after 2 hours on an idle machine (no assertion failed), because the scenario tests S19, S07, S04 and the determinism seeds are too slow under the race detector. The scenarios run and their outcomes are recorded as the "Done when" allows, and pack `one` collapses (about 600 households fall to 20 to 100 by tick 600): the cause and the probes are N-312 and N-314, and the goldens in `testdata/golden/off/` are recorded from that run, so a later fix to the pack moves them (testing.md 7.2). `testdata/chains/expected/one.scenarios.txt` is stale.

## Stage 2: version 1

The first consumer's needs (`vision.md` section 2). Checkpoint first, because
every later file adds state to it; then the region graph, because every module
is per region; then the modules in a dependency order (the table below). The
hash field id and the checkpoint tag of a module follow the `Modules` row of
`spec-config.md` 3.1 and `spec-checkpoint.md` 4.2, not the build order
(`plans/plan-2-0.md`). The one exception to the order of the pointers is
`Modules.Estates` (the 25th pointer): it is a version 1 part that the money
flows need, so it is built with MON; its field id in the hash is 25 whatever its
build order (CFG 8.2.2).

The order of the work is [plans/plan-2-0.md](plans/plan-2-0.md): plan 1 (work
before any row), then CHK, REG, SCL, **DEM as the first module**, MON, EST, then
plan 8 (the reference-pack rebalance, N-314, after the pool part of EST), then
the other modules in the order of the table. DEM is first because money has to
return to the non-farm roles (N-314, N-277); the rebalance comes after the pool
part of EST for the same reason. Plans 1 and 8 are rows of the table.

**Owner rulings of 2026-10-23** (plan 2-1 Phase 5; the plans are written against
them). CI: yes, with an arm64 job; the workflow (`.github/workflows/ci.yml`, three
jobs: check, full, arm64) is not written by plan 2-1 and has its own row below.
The gate is two runs (`rules/testing.md` 9.1): R, `go test -race -timeout 60m ./...`
with reduced seeds, and F, the non-race atomic-coverage run with all seeds.
Answers to the questions asked at the start of Stage 2: (5) plan 8 E0: yes, the
seasonal-crop feature C04 may move from the pack to an overlay; (6) plan 24 R2(a):
the realm pack is `five` renamed, and R3: the reference machine for the SCL time is
the machine used for plan 4's record; (7) the text of FR-VIA-060, FR-REF-041 and
FR-REF-042 and the two gate figures of plan 8 D365 are approved as drafted in the
plans; (8) ADR 0010 is accepted when plan 5 drafts it; (9) the `RPG` preset of
`denomination/` is accepted as the one display-only exception to the
setting-agnostic rule of `AGENTS.md`, and the `AGENTS.md` edit itself belongs to
plan 23. Second architecture, measured by hand on 2026-10-23 (binfmt installed on
the development host): `TestGolden_OneOff` passes on emulated linux/arm64 against
the committed files (7 s wall), and `BenchmarkStep_OneNoObserver` costs 15.8 ms a
tick emulated against 1.76 ms native (about 9 times; plan 24 estimates `five` and
all-on from it).

| Create | Specs | Plan | Status |
| --- | --- | --- | --- |
| Work before any row: the stale CHK and WLD passages, the race-aware scenario tests and gate, the ADR acceptances, CI and the second architecture (no module code) | CHK, WLD, ADR 0002, 0009 | [2-1](plans/plan-2-1.md) | plan written |
| `internal/codec` and `checkpoint.go` (`Checkpoint`, `RestoreWorld`, `Stateful`, `WithReseed` over the pure reseed function of Stage 0, `WorldRestored`, `ErrOptionContext`); `testdata/checkpoints/` | CHK | [2-2](plans/plan-2-2.md) | plan written |
| `internal/graph` and `regions.go` (the graph, distances, components, overrides, the guarded relocation (unexported `relocate`, `RelocationRefusal`) and the edge status (`Graph`, `Snapshot.Edges`)) | REG | [2-3](plans/plan-2-3.md) | plan written |
| `scale.go` (weights, the census) | SCL | [2-4](plans/plan-2-4.md) | plan written |
| `demand.go` (the first module file: it proves the template of 2.10.1) | DEM | [2-5](plans/plan-2-5.md) | plan written |
| `moneyflow.go` and `internal/moneyflow` (core identities; the flows as a module) | MON | [2-6](plans/plan-2-6.md) | plan written |
| `estates.go` (the `Estates` module part, appended: relief and the pool as a money-flow target; `ReliefPaid`, `Region.PoolRelease` and `Region.ReliefBelow`) | EST 2.6; CFG 6.2.4; MON 4.3.2 | [2-7](plans/plan-2-7.md) | plan written |
| Rebalance of the reference pack once demand, money flows and relief exist (pack data, warm start, goldens moved once) | REF, VIA, STB | [2-8](plans/plan-2-8.md) | plan written |
| `resources.go` and `internal/endowment`, `internal/curve` | RES | [2-9](plans/plan-2-9.md) | plan written |
| `subsistence.go` and `internal/ownuse` | SUB | [2-10](plans/plan-2-10.md) | plan written |
| `population.go` and `internal/population` | POP | [2-11](plans/plan-2-11.md) | plan written |
| `transport.go` and `internal/transit` | TRN, CAR | [2-12](plans/plan-2-12.md) | plan written |
| `information.go` and `internal/rumour` | INF | [2-13](plans/plan-2-13.md) | plan written |
| `migration.go` and `internal/migration` | MIG | [2-14](plans/plan-2-14.md) | plan written |
| `contract.go`, `internal/merchant` and `strategy/` (`Merchant`, `Options`) | MER | [2-15](plans/plan-2-15.md) | plan written |
| `external.go` and `internal/outside` | EXT | [2-16](plans/plan-2-16.md) | plan written |
| `integration.go` (external agents, `Direct`, `Relocate`, the slot setters, the queries) | INT (minimal: shocks and a player) | [2-17](plans/plan-2-17.md) | plan written |
| `indicator/` and `internal/measure` | IND | [2-18](plans/plan-2-18.md) | plan written |
| `internal/verdict` and the `stylised_long` build tag | STY | [2-19](plans/plan-2-19.md) | plan written |
| `experiment/`, `internal/design`, `internal/reduce` | EXP | [2-20](plans/plan-2-20.md) | plan written |
| `internal/calibration` and `testdata/calibration/` | CLB | [2-21](plans/plan-2-21.md) | plan written |
| `otelobserver/` (and `go.work`, added with the second module) | OBS 6, observability-and-logging.md 5, 6 | [2-22](plans/plan-2-22.md) | plan written |
| `denomination/` (the ladder, the formatter and the RPG preset; an optional package, standard library only, imported by consumers and examples and never by the core: `spec-currency.md` 7, FR-CUR-002, 008 to 019, NFR-CUR-001; the change that adds it also adds it to architecture.md 2.6 and `AGENTS.md`, and its terms to `doc.go`) | CUR 4, 7 | [2-23](plans/plan-2-23.md) | plan written |
| `.github/workflows/ci.yml` (three jobs: check, full with the Codacy coverage upload, and arm64; owner decision of 2026-10-23, to be written in a later plan) | testing.md 9.1; architecture.md 7.4; N-019 | none yet | planned |
| `examples/<first-consumer>/` (the RPG pack, the RPG preset of `denomination/` and a complete server) and `examples/<second-setting>/` | vision.md 5 items 3 and 5; architecture.md 1.3 | [2-24](plans/plan-2-24.md) | plan written |

Each module file ships in one PR with: its `internal/` package and unit tests,
its overlay under `testdata/chains/overlay/` (the demand module's is `testdata/chains/demand.toml`, where `spec-reference-chain.md` 4.4.6 puts it), its module-alone golden under
`testdata/golden/<module>/`, the proof that `testdata/golden/off/` did not
move, its checkpoint section and the fork-at-N test with it on, its events in
the catalogue tests, and its rows in the parameter-registry test.

**Done when:** the six criteria of `vision.md` section 5 hold: the reference
chain runs to a stable state from a warm start on `five.toml` and passes the
version 1 bands of STY; the invariant, determinism, golden and
configuration-only chain tests pass on two architectures; a new chain needs
only a configuration file, shown by the second setting; a checkpoint taken
mid-run and resumed is bit-identical; the consumer example builds a complete
server from the public API; the cost budget of SCL holds for the first
consumer's world size.

## Stage 3: later

The fourteen later-stage modules, in the order of their `Modules` pointers
(`spec-config.md` 3.1), which is the order in which each needs only the ones
before it. SKL (`skills.go`) joins this stage although its spec was written
with the slice (`vision.md` section 4).

| Create | Specs | Needs |
|--------|-------|-------|
| `skills.go` | SKL | PRO, WEL |
| `settlements.go`, `internal/settlement` | SET | REG, SCL |
| `storage.go`, `internal/room` | STO | LED, GDS |
| `employment.go`, `internal/employment` | EMP | AGT, WEL, PRO |
| `tenure.go`, `internal/tenure` | LND | RES, EST, MON |
| `polity.go`, `internal/polity` | POL | REG, LED, EST |
| `credit.go`, `internal/credit` | CRD | LED, MON, EST, WEL |
| `risk.go`, `internal/risk` | RSK | TRN, STO |
| `infrastructure.go`, `internal/facility` | CAP | PRO, RES, TRN, STO, SET |
| `insurance.go`, `internal/insure` | INS | CRD, RSK |
| `fiscal.go`, `internal/fiscal` | FIS | POL, MON, CLR, TRN, EXT |
| `institutions.go`, `internal/institution` | ORG | POL, SKL, SET |
| `policy.go`, `internal/policy` (`Policy`, `WithPolicy`, `DirectPolicy`) | PLC | POL, FIS, INT |
| `saving.go`, `internal/saving` | SAV | CAP, WEL |
| `technology.go`, `internal/technology` | TEC | PRO, REG, CAL |

Each ships as a stage 2 module does. The checkpoint tag of each is appended
(`spec-checkpoint.md` 4.2); `testdata/golden/all/` moves once per module, with
the explanation testing.md 7.2 requires.

## What is deliberately not built

- A `Module` interface, a hook registry, or a package per module
  (ADR 0007, alternatives).
- A persistence or transport layer, a `main` package outside `examples/`
  (architecture.md 1.1).
- Parallel stepping: ADR 0008 removes the stream as a blocker, but
  architecture.md 12.2 still needs an owner and a lifecycle; not before a
  consumer asks with a measurement.
- The not-in-version-1 list of note N-131.
- The CI workflow file itself, until its row in the Stage 2 table is built. The
  owner decided on 2026-10-23 that CI is wanted, with an arm64 job (open item O3
  of `plan-phase-0-foundation.md`, git history, `2abbe62^`); until the workflow
  exists the commands of `AGENTS.md`, the coverage reading of testing.md 9.1 and
  the timed fuzz runs are done by hand in each change.

## Updating this file

Change a row's status in the change that starts or finishes it; add a row in
the change that adds a file or package to architecture.md 2.6; record a spec
that the slice moves between stages here, in `vision.md` section 4 and in
`spec-roadmap.md`.
