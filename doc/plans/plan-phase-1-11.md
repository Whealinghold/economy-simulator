# Plan: Stage 1, row 11: the world (`World`, the tick, the markets, the snapshot)

**Status: done (uncommitted, stage-1-11; Phase 11 gate run 2026-10-20). See "As built" at the end.** Sequences the eleventh row of
Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`world.go` (`World`, `NewWorld`,
`Step`, `Run`, the guards, the four fan-ins), `tick.go` with the core positions
and the position table, `market.go` (the state of each market's report and the
price window; `order.go` is the clearing row's), `snapshot.go`", into eleven phases.
It implements [spec-world.md](../specs/spec-world.md) (WLD) sections 3, 5 and 6,
the parts of 8 and 9 that Scope names, and
[spec-observability.md](../specs/spec-observability.md) (OBS) section 5 for
the slice. It is built against `doc/rules/architecture.md` (the layout of 2.6,
the module-file rule of 2.10, the `internal/` boundary of 2.9, the fixed-point
rule of 3.5, determinism of section 7, concurrency of section 12),
`doc/rules/testing.md` and `doc/rules/observability-and-logging.md`. Where this
plan and the rules or the specs disagree, they win and this plan is corrected.

Rule tags read for this plan: architecture=905c0e578c4e662908638dd8,
observability=35827912d1e3141a76af3937, testing=b3559bc6d47470c221bc4642.
Notes tag read for this plan (2026-10-17, round 1 fix, before Phase 1's edits):
1fa82984f1fc7ee2b034e031 (history; Phase 1 re-reads the file and quotes the current tag).
Notes N-284 to N-289 are already in the file (see
"Disposition of the open notes"); the tag there changes with every edit of the file.
Round 3 fix (2026-10-17): decisions D193 to D196 and the changes named in them were added;
the packs live in `testdata/chains/` as `one.toml`, `warm-one.toml`, `two.toml`,
`warm-two.toml`, and `second/second.toml`, `second/warm-second.toml` (the pack `second` is
the one in a directory; the implementer reads the layer list for `LoadFS` from the tree).

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR,
the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md) (D3 and
D5 above all) or a ruling of the coordinator, and is cited. Where a decision is
still owed, Phase 1 makes it in the document that owns it, before the code that
depends on it; the ones this plan itself proposes are marked "Decided by this
plan" (ruled by the coordinator, reversible, where marked). Decision IDs D140 to D169 are this
plan's (the range is full; D190 to D192 were added in the round 1 fix, D193 to D196 in the
round 3 fix, and D158, D165, D166 and D167 were corrected in place); earlier plans' decisions are cited by file and ID.

No PRD exists for this row. The requirements source is the roadmap row, WLD 3, 5,
6, 8, 9 and OBS 5. The row is the integrating one: it is built after the ledger,
goods, clearing, beliefs, production and welfare rows
([1-1](plan-phase-1-1.md) to [1-6](plan-phase-1-6.md), all in the tree) and after
the agent, estates, spawner and viability rows (rows 7 to 10), which have no
plan at the time of writing and which this plan lists as ASSUMED in its
"Cross-plan contract". Row 12 (the golden files and the per-position hash) is
built after this row and replays this row's step from a test file.

Spec codes: WLD world, OBS observability, AGT agents, SPN spawner, EST estates,
CLR clearing, BEL beliefs, PRO production, WEL welfare, LED ledger, VIA
viability, CFG config, CHK checkpoint, INT integration, REG regions, RES
resources, STO storage, MIG migration, STB stability, REF reference chain, IND
indicators. `FR-`/`NFR-` IDs are the requirement IDs of those specs.

## What Stage 0 and the earlier rows already give, and what is left

| Concern | Where it already is |
|---------|---------------------|
| The configuration, its canonical copy, the validator, the hash, the warnings, the pack loader | `config.go`, `validate.go`, `hash.go`, `config/` (Stage 0) |
| `Option`, `options`, `WithLogger`, `WithObserver`, `WithAudit`, `WithTargetSpread`, `WithSeed`, `registry[T]` (`add`, `find`, the `validID` rule), `WithClearing`, `resolveClearing` | `option.go` (Stage 0 and row 3) |
| The observers value (`observers`, `newObservers`, `deliver`, the `supplied` flag), `warnRun`, `discardObserver` | `observer.go` (Stage 0) |
| The seed, the one PCG and `Rand` (`stream`, `newStream`, `keyed`, `keyedPair`, `word`, `draw`, `drawRange`), the seven site constants of the slice, the pure `reseed` | `random.go` (Stage 0) |
| The sealed events the slice emits and the `Reason`, `Cause`, `Kind` constants | `event.go` (Stage 0 and rows 3 to 6): `WorldCreated`, `RunStarted`, `RunEnded`, `TickStarted`, `TickEnded`, `OrderPlaced`, `OrderRejected`, `TradeExecuted`, `MarketCleared`, `BeliefsUpdated`, `RecipeExecuted`, `RecipeRefused`, `RecipeAbandoned`, `ProductionHalted`, `UpkeepShortfall`, `AgentSpawned`, `AgentBankrupted`, `EstateOpened`, `WeightChanged`, `RoleChanged`, `EntryLimited`, `BoundaryFlow`, `WearApplied` |
| The sentinels `ErrInvalidID`, `ErrDuplicateID`, `ErrOutOfBounds`, `ErrNilExtension`, `ErrUnknownClearing`, `ErrInvalidTrade` | `errors.go` |
| `internal/ledger`: accounts, transfers, lots, reservations, `Transform`, `WearItems`, `Discard`, `Holding`, `Age`, `EndTick`, flow totals (`AppendFlows`, `ClearTotals`), the construction phase (`InConstruction`, `EndConstruction`, `MoveStated`), `Audit`, `HasReservation`, claims; `ledger.go` (`newLedger`, `boundaryFlows`, `wearApplied`, the reason names both ways) | row 1 |
| `recipeTable`, `roleTable`, `dayTable`, `Calendar.At`, `SeasonAt`, `TicksPerYear` | row 2 |
| `internal/clearing`, `Order`, `Trade`, `Clearing`, `AverageClearing`, `marketReport`, `seedReport`, `book`, `clearScratch`, `receiptTally`, `clearMarket`, `registry[Clearing]` | row 3 |
| `internal/beliefs`: `Limits`, `Clamp`, `Gate`, `Update`, `Draw`, `Reseed`, `Window`, `Tally`, `NextRun`; `beliefs.go` (`beliefParams`, `beliefReason`, `beliefsEvent`) | row 4 |
| `yieldTable`, `yieldShocks`, `job`, `productionScratch` (`maxExecutions`, `start`, `complete`, `abandon`, `reweigh`, `warmJob`, `anchors`, `unitCost`, `addPending`), `refusalTally`, `recipeExecuted`, `recipeAbandoned` | row 5 |
| `internal/welfare` and `welfare.go` (`history`, `newHistory`, `newTarget`, `welfareScratch` with `addReceipt`, `addInputSpend`, `addWear`, `markWorking`, `markTraded`, `record`, `workingTick`, `readBuf` with `observedWage`, `regionWage`, `heldByFloor`, `summarise`, the runs `floorRun` and `clampRun`) | row 6 |

What is left for this row is everything that makes these a running world: the
type that holds them, the options and name checks, the construction in the order
of WLD 5, the tick in the order of WLD 6 with its one-line positions, the
per-market state the clearing and beliefs rows left to it, the failed and
in-step guards, `Run`, the logs and events at each position, the snapshot, and
the tests that need a world.

## Scope

**In scope.**

- `world.go`: `World` (the doc comment states "not safe for concurrent use",
  architecture.md 12.1), `NewWorld`, `Step`, `Run`, `Err`, `Tick`, `Config`,
  `ConfigHash`, `Audit`, the failed and in-step guards, the sentinels
  `ErrWorldFailed`, `ErrInStep` and `ErrAuditFailed`.
- `construct.go` and `construct_warm.go` (new root files, Phase 1 adds them to the
  layout): steps 5 to 10 of WLD 5.1: the derived tables, the stream, the ledger
  and its accounts in the order of 5.2, the agent ID space, the agents (the
  world opens their ledger accounts), the application of the warm start (the
  selection and the stagger, the grants, the jobs and their books, the pools:
  D191), the creation draws, the construction audit, the events and logs of 5.8.
- The executor of the spawner's plan (P7.1(d)) and the role-change and
  weight-change compositions inside it, and `removeAgent` (coordinator ruling,
  2026-10-17): Phase 7 (D192).
- WLD 8 and 9: this row builds FR-WLD-074 (the flow totals cleared at P8.6) and
  the guards of section 3. FR-WLD-070 to 073, 075 to 077 are Stage 2 or the
  modules'; FR-WLD-080, 090, 091 and NFR-WLD-005 were built in Stage 0 (plan 0
  Phase 2); FR-WLD-081 to 083 are rules of the specs that no Stage 1 code applies.
- `tick.go`: the list of positions as a value (identifier, owner, in order), the
  straight line of one-line calls, and the entry of `Step`; the unexported
  position methods in `phase_start.go`, `phase_production.go`, `phase_trade.go`,
  `phase_outcomes.go` and `phase_exit.go` (D144).
- The four fan-ins: `fillAgentView`, `checkTradePlan`, `removeAgent`,
  `relocationRefusal`, each a fixed-order composition and nothing else
  (architecture.md 2.10.4; `fanin.go`).
- `market.go`: the per-market record (the `marketReport`, the `beliefs.Window`,
  the `BoundRun`, the book), the table of markets in (`RegionID`, `CommodityID`)
  order, the P5.1 push.
- `snapshot.go`: `Snapshot` and its row types for the sections of the slice
  (OBS 5.3.1 to 5.3.5), `World.Snapshot()`.
- The logs and events of every core position the world owns (the `Warn`, `Info`
  and `Debug` lines of OBS 7.2, 7.3 rows 1 to 4, 6 to 8, 19 and 26, 7.5), the
  run state of each `Warn`.
- The tests that need a world and are this row's after the coordinator's split
  (D164): the position table test, the options halves FR-WLD-001/002/003 and
  FR-OBS-090, run lifecycle and tick bracketing, composition, emission order of a
  scripted step, the log tests (every `Warn` path, no `Info` in the tick,
  counterpart of every `Warn`), the call-site allocation test, the catalogue test
  with the world emitting each declared event, the snapshot tests of OBS 9.8.
- The glossary terms these files use, in `doc.go`, before they use them
  (architecture.md 3.2).
- The spec, rule, note and roadmap edits these files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| The agent type (unexported, only the fields no module owns), its per-agent state changes (creation helper, the consumption step of P6.1, the core view filler, the core plan checks, the production-halt predicate), `Strategy`, the default strategy, `Directed`, `WithStrategy` | Row 7 (`agent.go`, `strategy.go`). This row calls them (ASSUMED names below) |
| The estate pseudo-agent, its account and listings, the pool and its use to fund an entrant, the estates' asks and markdown | Row 8 (`estates.go`) |
| The spawner interface, `Decide`, the default rules and the pure support functions of SPN 4.1 to 4.3, `WithSpawner`, `ErrUnknownSpawner` | Row 9 (`spawner.go`). The EXECUTOR of the plan (P7.1(d)) and `removeAgent` are this row's (coordinator ruling, 2026-10-17) |
| The solver, `virtualStartOffsets`, the viability rules and the validated warm-start table (there is no reader function: the table is `Config.WarmStart` data) | Row 10 (`viability.go`, `internal/steady`). The APPLICATION of the table at construction is this row's (D191) |
| The pack runs: the golden of `one.toml`, the invariants over seeds, determinism with and without logger and observer split into `Run` calls, the configuration-only chain test, the defect-overlay file tests, S01/S04/S07/S16/S19, the events renderer and `events-one.txt`, the cardinality test, the full-`Step` allocation baseline, the per-position state hash | Row 12 (coordinator ruling 1, 2026-10-17) |
| The checkpoint, `RestoreWorld`, `WithReseed`, `WorldRestored`, `ErrOptionContext`, the saved state of the world (WLD 12) | Stage 2, `checkpoint.go`. This row keeps what is saved in one place (D145, N-288) |
| Every module: its positions, fan-in steps, snapshot rows, events | The module's PR, appended (architecture.md 2.10.5) |
| INT: `Agent`, `Agents`, `Market`, `Region`, `Slots`, `Warnings`, `Calendar`, `ReferencePrice`, `Graph`, the setters and the calls between ticks | Stage 2, `integration.go`. This row builds the guards they use |
| `RelocationRefusal` as a public read and `Relocate` | Stage 2 (`regions.go`). The fan-in `relocationRefusal` exists, with no module step (D165) |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- `World` owns all simulation state, is not safe for concurrent use and starts no
  goroutine (WLD 3.1; architecture.md 12.1, 12.2). Two worlds share nothing, so
  there is no package-level state (architecture.md 7.1; FR-WLD-011).
- `NewWorld` applies the options in order, validates in full, runs the name
  checks after `Validate` succeeds, and returns a world or an error and no world
  (WLD 3.2; FR-WLD-001, 002). The four name checks are `ErrUnknownStrategy`,
  `ErrUnknownSpawner`, `ErrUnknownClearing` and `ErrUnknownPolicy`; the first and
  last belong to rows 7 and a Stage 3 module, the third exists
  (`resolveClearing`).
- A `Step` that returns an error leaves the world failed; every started step
  emits `TickEnded`, a failed one with `Failed` true set after the world is
  marked failed; a panic emits nothing (WLD 3.5; FR-WLD-006, 012).
- A call from inside a step returns `ErrInStep` and changes nothing (WLD 3.6;
  FR-WLD-007). The flag is set at the entry of `Step` and cleared by a deferred
  function; it is not state.
- `Run` checks `ctx.Err()` before every tick including the first, never inside a
  tick, emits `RunStarted` before the first tick and `RunEnded` before it
  returns, and returns the table of WLD 3.4.2 (FR-WLD-008, 009).
- The order of the construction (WLD 5.1), of the accounts (5.2), of the agent IDs
  (5.3), of the warm start (5.6) and of the creation draws (5.7) is the golden
  output (architecture.md 7.4). Nothing takes a random number before step 8.
- The tick is the eight phases and the positions of WLD 6, each with the stable
  ID of 6.1.1; a module that is off has none of its positions and changes none
  of the others (6.1.5; FR-WLD-040, 044). `tick.go` is a straight line: each
  position is one call, in the spec's order, and holds no logic (architecture.md
  2.10.3).
- Reservations open only at P3.2, P3.4, P3.4a and P3.5 and are all settled or
  released by the end of P4.1 (6.1.6; FR-WLD-045). The flow totals are cleared at
  P8.6, after the events and the audit (6.10, C18; FR-WLD-074).
- Events are built only when an observer was supplied; the flag is recorded once
  at construction (observability-and-logging.md 4.5; OBS 8.1.1). No `Info` in the
  tick; a `Warn` in the tick is bounded by a configured entity (rules 3.2.2).
- A snapshot is a deep copy of plain data, built between ticks on a healthy
  world, a function of the saved state (OBS 5.1, 5.2; architecture.md 12.4).
- Coordinator ruling 2 (2026-10-17): no hook in the production `Step`. Every
  position is an unexported method of `World`; the non-position parts of `Step`
  (guard, tick advance, `TickStarted`, `TickEnded`, the marking of a failed
  world) are unexported methods that `Step` and a replay from a `_test.go` file
  both call, so row 12's per-position state hash can replay the step.
- Coordinator ruling 1: the split of the Stage 1 "Done when" tests (D164).

Decided by this plan, to be written into the owning document in Phase 1 before
code (D140 to D142 are final, ruled by the coordinator, reversible):

- **D140.** The configuration section type `World` is renamed `WorldParams`
  (and `DefaultWorld` `DefaultWorldParams`); `Config.World`, the TOML table
  `[world]` and the hash field ids do not change; the scope cells "World" of the parameter-owner table become `WorldParams` (in the Phase 2 commit, with the type: round 4 NB-2). Reason: `config.go:62` declares `type World struct` for the
  world-level parameters, and architecture.md 2.2, 3.2, 6.1 and 12.1 and WLD 3.1
  require the simulation type to be `World`; the two cannot share the root
  package (doc.go's glossary entry already names both senses). The rename is
  mechanical (about 160 uses, `gopls` rename or `gofmt -r`), touches `config/`
  and the tests, and the registry test and `notBuilt` entries follow the new scope name (testing.md 6.7 reads a Go type name; the README cell is that name). The
  alternative of renaming the simulation type contradicts the rules and about 20
  spec mentions. Accepted by the coordinator (2026-10-17): architecture.md 2.2 and 2.10.2 name the root type `World` as the simulation, the rules beat `spec-config.md` 3.7, the module is unreleased (architecture.md 6.6 not yet binding) and no hash input changes. FINAL (ruled by coordinator, reversible: a rename back is mechanical while the module is unreleased). Phase 1 writes it into
  `spec-config.md` 3.7 and the glossary; Phase 2 does the rename.
- **D141.** The position table test is staged (testing.md 6.8 gains the form of
  6.7 and architecture.md 9.7): `tick.go` holds the list of every position of
  WLD 6 that has a call, and one explicit list `notBuiltPositions` names the
  spec positions that have no call yet; the test compares identifier, owner and
  order with the table of `spec-world.md` 6 by a plain text parse, fails when a
  listed position has a call (a stale entry) and when a spec position is neither
  called nor listed (a typo). Each module's change deletes its entries and
  appends its call. The test also reads `tick.go`'s straight-line function with
  the standard library's `go/parser` and checks that it calls the listed
  positions of `where` `line` in the listed order and nothing else, so "a call with no row" fails
  too (testing.md 6.8). Each list entry has a third field, `where`: `line` (a call
  in the straight line) or `bracket` (run inside `beginStep` or `endStep`: P0.1,
  P0.2a, P0.3, P8.5, P8.6, D166). The order comparison with the spec table covers all
  entries; the parse expects a `line` entry's call in the straight line and no call
  for a `bracket` entry. What "owner" compares, and what P0.2 ("nothing is
  cleared at entry": no call) is, are ruled in O8 (first area code, "this spec" read as WLD; P0.2 is an empty position method, a `line` entry). Reason: about 70 positions belong to modules built later;
  a test that demanded a call for each would force module scaffolding into the
  slice (architecture.md 2.10.6). Phase 1 edits testing.md 6.8 and architecture.md
  2.10.3 with fresh tags. Accepted by the coordinator (2026-10-17), as the staged form of architecture.md 9.7 and testing.md 6.7.
- **D142.** The order of the implementation is rows 7 to 10 before this row, with
  no test doubles for them (question 4). The only fakes are the sanctioned ones at
  the extension points (a faulty `Clearing`, a recording `Observer`, a
  capturing `slog.Handler`, a consumer-style `Strategy` and `Spawner` where a test
  needs a behaviour the defaults do not give).
- **D143.** The failed-world state is one error field `failed error` set before
  `TickEnded` is emitted and a boolean `stepping` set at the entry of `Step`.
  Both are unexported, are not state, and are not saved (WLD 3.6.2, 12.1). A
  panic that leaves a step marks the world failed through a deferred function
  that does not call `recover` and emits nothing (WLD 3.5.4).
- **D144.** The files of the row, named for what they hold (architecture.md
  2.4), added to the layout of architecture.md 2.6 and `AGENTS.md` in Phase 1
  (the lesson of N-243: a new root file is listed when it is made): `world.go`
  (type, `NewWorld` steps 1 to 4, readers, `Step`, `Run`, guards), `construct.go`
  (steps 5 to 10), `construct_warm.go` (the application of the warm start, D191),
  `tick.go` (the list and the straight line), `phase_start.go`
  (P0, P1), `phase_production.go` (P2), `phase_trade.go` (P3, P4),
  `phase_outcomes.go` (P5, P6), `phase_exit.go` (P7, P8), `fanin.go` (the four
  fan-ins), `market.go`, `snapshot.go`. The split keeps each file under the
  Lizard limits (about 500 lines, cyclomatic complexity 10, 8 parameters:
  N-243). The module file rule (architecture.md 2.10.1) is for modules; these
  are core concept files.
- **D145.** What the world saves is kept in one struct, `worldState`: the
  effective seed, the next agent ID, the construction flag, the diagnostic
  runs of the bounded `Warn`s the world keeps (D156), and the spawner's state
  per (region, role): the renewal carry and the "had an agent" bit (plan 1-9
  D118, N-265). The tick counter is the
  ledger's (`Ledger.Tick`, `AdvanceTick`). Everything else on `World` is derived
  or scratch and is marked so in its comment. This is only a place for
  `checkpoint.go` (Stage 2) to read; the row writes no encoding.
- **D146.** The slice's snapshot holds the sections the slice has state for, no
  more (D3 of plan 0: slice fields only; a module appends its own rows at the
  end): the header (OBS 5.3.1), `Regions` (5.3.2, the fields of the slice),
  `Boundary`, `Pending`, `Construction` (5.3.3), `Roles` (5.3.4, the welfare,
  agent, spawner fields of the slice), `Markets` and `Jobs` (5.3.5). The module
  slices (`Flows`, `Edges`, `Shipments`, ...) are not declared. `Modules` is the
  empty slice. **`RegionRow.MoneyContract` (and `MoneyTreasury`) is not in the Stage 1 snapshot** (coordinator ruling 2026-10-17): it is sourced from the escrow of EMP and MER (OBS 5.3.2), module-owned, and D3 of plan 0 says slice fields only; it is appended with its module and omitted from the golden at zero. In Stage 1 the conservation identity of FR-OBS-041 reads `MoneyHouseholds` and `MoneyPool` only (escrow is 0). This is the decision of Phase 8.
- **D147.** `MarketState` (OBS 5.3.5, INT 6.3) is declared here, in
  `snapshot.go`, because the snapshot needs it now and INT's query `Market`
  (Stage 2) returns the same type, "so the snapshot and the query cannot
  disagree". Its fields are the `marketReport` fields plus the window's
  `WindowLen`, `WindowMean`, `WindowLow`, `WindowHigh`, under INT's names.
- **D148.** The readers of the slice are `Tick`, `Err`, `Config`, `ConfigHash`,
  `Snapshot` and `Audit`. `Calendar`, `ReferencePrice`, `Warnings`, `Graph`,
  `Travel`, `Polity`, `TreasuryAgent` and the INT queries arrive with their
  owners (INT 6). FR-WLD-110 is tested for the readers that exist; the rest is
  appended with INT.
- **D149.** `Audit() error` (WLD 3.7.1, LED 12.4) is public, returns
  `ErrInStep` or `ErrWorldFailed` under the guards, and otherwise the ledger's
  `Audit` with the module checks (none in the slice). `WithAudit` runs it at
  construction (step 9) and at P8.4.
- **D150.** An audit failure at P8.4 sets `Failed` for P8.5, the world is marked
  failed before `TickEnded`, and `Step` returns an error wrapping
  `ErrAuditFailed` and naming the tick (WLD 3.3.3, 3.5.6, C13). The mapping is the
  pure function of D190.
- **D151.** A failure at P4.1 is `clearMarket`'s `*tradeListError`
  (`errors.Is(err, ErrInvalidTrade)`); the step wraps it with `%w` and the tick,
  marks the world failed, emits `TickEnded` with `Failed` and returns. The
  markets after it are not cleared (WLD 6.6.1; FR-WLD-048).
- **D152.** `Run` logs `Info` "run started" (`ticks`) and "run ended" (`ticks`
  completed, `reason` one of `completed`, `cancelled`, `failed`) whenever `Run`
  emits `RunStarted` and `RunEnded`, with or without an observer (the logger is
  independent of the observer flag, OBS 7.2); a `Run` that ends in a panic writes
  and emits neither end; the error is returned, not logged
  (WLD 16.3; architecture.md 8.3). A negative `ticks` is `ErrOutOfBounds`
  naming `ticks`, with no event and no line.
- **D153.** The rotation: at the start of P2, per region, the agents whose
  account is in the region, in ascending agent ID, are listed once into a
  scratch slice (`n_r` is its length, away agents included); the calls of `Work`
  and the starts walk it from index `t mod n_r`, wrapping (WLD 6.1.4). The list
  is built at the start of P2 after the completions of P2.1 do not change it
  (completions do not add or remove agents).
- **D154.** One scratch per world, allocated at construction and reused:
  `clearScratch` (row 3, N-240), one `book` per market (kept until the outcomes
  of P5.4 are delivered), `receiptTally`, `productionScratch`, `refusalTally`,
  `welfareScratch`, `readBuf`, the rotation lists, the event slices. The world
  never shrinks them. With the default logger and observer no position
  allocates (rules 8.2; D164).
- **D155.** The world's streams. Before each call to an extension or a draw site
  the world reseeds the one PCG with the key of the site (WLD 7.3.3). The slice's
  sites: D1 `siteConstruction` (tick 0, the agent) at the creation draws, D7
  `siteClearing` (the tick, region, commodity) at each `Clear`, D8 `siteOutcome`
  (the tick, the agent, the commodity; one stream per order) at each outcome, D10
  `siteCreation` (the tick, the new agent) at a creation in phase 7, D5 `siteTrade`
  (the tick, the agent) before every `Trade` call, because the default takes one word
  per intent it considers (plan 1-7 D93, N-246; WLD 7.3 D5), and, only for a
  consumer's extension, D3 `siteWork` and D9 `siteSpawner`. A call to the default
  strategy or spawner at a site where the default takes no word (`Work`, `Decide`)
  needs no reseed and the world skips it (WLD 7.3.3). The world takes no
  other word (FR-WLD-062).
- **D156.** The bounded `Warn`s of the slice that the world emits, and the run
  each keeps: row 2 "households unfed" (B, per region and role, run in
  `worldState`), row 3 "price beliefs at bound" (B, per market: `BoundRun`, which
  `beliefs.NextRun` advances for every market every tick, also a tick with no
  outcome: N-230), row 4 "work floor binding" (B, `floorRun`), row 6 "role
  extinct" (A, spawner row's run), row 7 "entry unfunded" (B), row 8 "group below
  floor" (A), row 19 "recipe output disposed" (no run, one per recipe per tick,
  fed from `RecipeExecuted.Overflow`: N-242 point 3), row 26 "welfare record
  clamped" (A, `clampRun`). Each run is advanced at the position that owns its
  condition and is a diagnostic: it affects no result. The runs live in
  `worldState` (D145) so a resumed run logs as the original did.
- **D157.** Cost anchors (P1.2) read the published price left by P4.1 of the
  previous tick (the seed price at tick 0) and the pooled regional wage
  (`readBuf.regionWage`, falling back to the warm-start wage), and call
  `productionScratch.anchors`; the anchor in force for a market is set on the
  book before its clearing (`MarketCleared.CostAnchor`, N-242 point 2). The
  table is derived and never saved.
- **D158.** The welfare calls (WEL; plan 1-6 "Out of scope"): the world calls
  `markWorking` and `addWear` with the facts the production steps return at P2.1
  and P2.7 (`markWorking` once per agent per tick: it panics on a second call, so
  a job started at P2.7, a one-tick job included, is marked at its start and a
  continued or completed job at P2.1, never both), `markTraded` **after each
  market clears at P4.1, for the buyer and the seller of each settled `Trade`**
  (`welfare.go:180`: "from the settled trades"; WEL 4.3, FR-WEL-017: a role with no
  recipes works if it bought or sold, and an order that never fills is not a
  working tick), `addReceipt` and `addInputSpend` from the settled trades and the
  purpose of each intent (plan 1-3: `Order` has no purpose, so the world keeps the
  facts of each admitted intent for the tick (D194: a `Trade` carries only `Buyer` and
  `Seller` agent IDs, not an admission index), and at P6.3 `record`, `Push`,
  the clamp run and the reset of the scratch, in ascending agent ID. All of
  these skip a pseudo-agent (an estate seller has no welfare slot).
  **The scratch is also cleared in full at P8.6** (a `reset(slot)` over every slot, in
  ascending order, allocation-free) from Phase 4, where the feeds are first called:
  `reset(slot)` is the only clear (`welfare.go:191`) and `markWorking` panics on a
  second call for a slot, so without it a job continued at P2.1 in the second tick
  would panic and the receipts would accumulate until Phase 6. After Phase 6 the
  P6.3 reset and the P8.6 clear are both kept; the second finds zeros.
- **D159.** Events are built through one helper per event kind taking the
  `observers` value, and every construction site is guarded by
  `w.obs.supplied` (observability-and-logging.md 4.5). A counter that exists only
  to build an event (the refusal tally's flush rows, the aggregate counters) is
  filled under the same flag, **except** a count that a `Warn` also reads (row 19 reads `completion.overflow`,
  row 2 the unfed counts of P6.1, row 3 the `Tally`): those exist whether or not an observer is
  supplied, and the `Warn`s read them, so a logger without an observer still gets its `Warn`s
  (OBS 7.1, FR-OBS-070/071; round 4 NB-8).
- **D160.** The construction bucket's `BoundaryFlow` events (`Tick` -1) and the
  bucket's `WearApplied` rows are emitted at P0.2a at the entry of the first
  step of a world that is still in its construction phase, before `TickStarted`;
  P8.2 emits the tick's `BoundaryFlow` rows and then the `WearApplied` rows
  (WLD 6.2, 6.10; OBS 4.4.1.2). The world ends the construction phase when the
  first `Step` begins (`Ledger.EndConstruction`, FR-WLD-035).
- **D161.** The tick advances at P8.6 with `Ledger.EndTick()` (`AdvanceTick`, then
  `ClearTotals`: one call, so the order cannot drift) after `TickEnded` and the
  audit, then the per-tick counters of the rows and the welfare scratch (D158) are
  cleared; a failed step does not advance the tick and
  clears nothing (`Tick()` is the tick the failed step was running).
- **D162.** The sentinels added at the root, each with its first user (plan 0,
  Phase 3): `ErrWorldFailed` (every call under a failed world), `ErrInStep`
  (every call under a running step), `ErrAuditFailed` (construction and P8.4).
  `ErrOptionContext` is `WithReseed`'s, Stage 2. No ledger sentinel is mapped to
  a public one (N-075; plan 1-3 D33, D34): a ledger refusal at the world's calls
  is a violated invariant and panics with a message that says so
  (architecture.md 8.1) or, with `WithAudit`, is found by the audit.
- **D163.** The `Debug` line "tick ended" (`tick`, `agents`, `households`) is
  emitted once per step guarded by `logger.Enabled`, and reads the counts from
  the world's counters, not by a scan (WLD 16.3). The construction lines are
  "configuration warning" (once each, `rule`, `path`), "warm start applied" and
  "world created" (the keys of WLD 5.8.1 for the modules that exist in the slice).
- **D164.** Tests of this row (coordinator ruling 1): the position table test;
  FR-WLD-001/002/003 and FR-OBS-090 option halves; run lifecycle and tick
  bracketing; composition of observers; emission order of a scripted step with a
  recording observer (not row 12's renderer); the log tests (every `Warn` path,
  no `Info` in the tick, the counterpart of every `Warn`, keys, constant
  messages, no `Error`); the call-site allocation test; the catalogue test with
  the world emitting each declared event; the snapshot tests of OBS 9.8. Row 12
  owns the pack runs, the cardinality test, the full-`Step` allocation baseline,
  determinism and the events renderer. The hand-written recorder and capturing
  handler are in `observe_helpers_test.go` of the external test package, which
  row 12 reuses.
- **D165.** The fan-ins (architecture.md 2.10.4) in the slice. `fillAgentView`
  (built in **Phase 4**, because P2.5 calls `Work` with a view) is the composition
  of row 7's `fillCore`, which computes `maxExecutions`, the welfare fields, the beliefs'
  bounds and the belief group from `viewCtx`, and the module fillers (none in the slice;
  plan 1-7 Phase 7 item 2). `viewCtx` (the per-region market reports, windows, anchors,
  published prices and the season, from the market records of Phase 3) is assembled by
  `(*World).refreshViewCtx` in `market.go`, called once at the start of P2.5 (the state
  after P2.1) and once at the start of P3.2 (the state after P2.7), never cached across
  positions; it is not part of the fan-in, so the fan-in stays a composition of calls.
  `checkTradePlan` (Phase 5) composes the world's checks of the plan (`normalise`,
  `orderOf`, the discard checks and the tradable set; the `JobRequest` check is P2.5's `admit` and
  affordability is admission's: round 4 NB-11).
  `removeAgent` composes, in the order of `spec-agents.md` 3.6, the production
  `abandon`, the removal steps composed here from the primitives (account closure, money and goods
  to the estate row's account; row 7 builds no removal core), the events in the order of WLD 6.11 item 8; its
  module steps 2b to 2g are empty in the slice. `relocationRefusal` composes the
  refusals of the modules that are on and in the slice returns no refusal; it
  exists so that Stage 2's `Relocate` and MIG's guard have the one entry point.
  A second removal path, or module logic inside a fan-in, is a defect.
- **D166.** The non-position methods of `Step` (the replay seam of coordinator
  ruling 2; FINAL, ruled by coordinator, reversible: the seam is unexported methods that row 12's per-position hash calls in the same order, and no hook, callback or build tag in production code) are `beginStep` (the guards P0.1, the stepping flag, the end of the
  construction phase, the bucket events P0.2a, `TickStarted`), `endStep(err)`
  (the audit result, `TickEnded`, marking failed, the tick advance and the clears
  of P8.6 only on success) and the position methods named after their IDs
  (`p21Complete`, `p25Work`, ...). `Step` is `beginStep`, the call of the
  straight line in `tick.go`, `endStep`, with the guard and the deferred function of
  D167. `endStep(err)` clears the `stepping` flag as its last act on its normal return
  (the replay of row 12 has no `Step` around it). The straight line starts with the
  empty `p02()` (O8); because `beginStep` runs P0.1, P0.2a and P0.3 first, `p02()`
  executes after them, which nothing can observe (it is empty and takes no state);
  the list value is in spec order (O9, ruled (a): it stays in the straight line). A test file replays the step by calling
  the same unexported methods in the same order. No production code reads a
  hook, a callback or a build tag. Which spec positions are the bracket and which
  are methods of their own: P0.1, P0.2a and P0.3 run inside `beginStep` and P8.5
  and P8.6 inside `endStep`; every other position, P8.4 (the audit) included, is a
  method named after its ID (`p84Audit() error`), and the error of P8.4 is the
  argument of `endStep(err)`, which is how the audit's failure reaches `TickEnded`
  (D190). Row 12's row-to-method map points P0.1, P0.2a and P0.3 at `beginStep` and
  P8.5 and P8.6 at `endStep`; P0.2 is an empty method `p02()` (O8).
- **D167.** `Step` returns at the first error of a position; the positions after
  it do not run, and `endStep(err)` is still called with that error. The shape:
  a local `finished := false` and a deferred function installed first; the body
  `if err := beginStep(); err != nil { finished = true; return err }` (a guard
  refusal, `ErrInStep` or `ErrWorldFailed`, returns at once with the flag and the
  failure cause untouched), the straight line, `err = endStep(err)`, then
  `finished = true`. The deferred function does nothing when `finished` is true: a
  refused inner call (it never clears the outer step's flag, and a refused call on a
  failed world never replaces the real cause: FR-WLD-006, round 3 NB1) and the
  normal and the error return (`endStep` has cleared the flag). Otherwise a panic is
  leaving the step
  (in `beginStep` after the guard, a position, an observer, or `endStep` itself): the function
  marks the world failed with an error naming the tick and that the step panicked
  (no `recover`, so the value is not read), emits nothing and clears `stepping`
  (D143; WLD 3.5.4). So the flag is clear on every path, and a consumer that
  recovers gets `ErrWorldFailed`, not `ErrInStep`.
- **D168.** `Run` wraps a step's error with `%w` and the number of ticks
  completed, and wraps a cancellation as `fmt.Errorf("run cancelled after %d
  ticks: %w", completed, ctx.Err())` (WLD 3.4.2).
- **D169.** The names table of `WorldCreated` is built once at construction from
  the canonical configuration (`Names`; the string data may be shared, but every
  event and every snapshot owns its slices, FR-OBS-002 and Phase 9's deep-copy test),
  and
  `AgentSpawned` is emitted for each household agent in ascending ID after it
  (WLD 5.8.2).
- **D193.** (Round 3 fix.) The agents and weights of a group are derived by the pure
  function `groupWeights` of Phase 3 deliverable 2a (`spec-scale.md` 3.2, 11: inline in
  construction, a small pure function with a table test; Stage 2's `scale.go` takes it
  over), so the stagger, the jobs' selection and the ID tests have one source.
- **D194.** (Round 3 fix; widened in round 4, NB-1.) How the facts of an intent reach the
  settled `Trade` and the outcome. `Order` and `bookEntry` carry no `Purpose`, `Target`,
  `Reserve` or `UnitCost` (spec-agents.md 6.4: they are on the `Intent`), yet P4.1 needs
  the buyer's purpose and P5.4 needs `Target`, `UnitCost` and the reserve that
  `outcomeStock(held, reserve)` takes (FR-AGT-049) and the unit cost that
  `beliefs.Limits` takes. Each market's scratch therefore holds `intents`, a slice of
  `{purpose, target, reserve, unitCost}` **parallel to `book.entries`** (same index: a
  successful `book.admit` appends one entry, and the world appends one record at that
  index straight after, for every admitted order of a household agent, bid or ask; an
  estate's ask has a zero record and no outcome), cleared with `book.reset`. Because
  `book.duplicate` allows an agent at most one bid per commodity per market, the entry of
  a (market, buyer agent) pair is unique; `slotOf`, a small slot table indexed by the
  dense `AgentID` (resized once at construction, grown with the agent table), maps the
  buyer to its entry index, is written at the same moment, read at P4.1 for the `Buyer`
  of every trade the `Clearing` returns (a custom `Clearing` may return several trades
  for one order: each finds the same entry), and its touched slots are listed when
  written and cleared at the next `book.reset`, so the reset is proportional to the
  orders and nothing allocates after warm-up. P5.4 walks the entries in admission order
  and reads `Target`, `Reserve`, `UnitCost` (and `Purpose`) from the record of the
  entry it is on, so Phase 6 adds no retrofit to Phase 5's admission code. No map, no
  scan of the book's entries. The seller side needs no purpose (it is a receipt).
- **D195.** (Round 3 fix.) The fan-ins are compositions (architecture.md 2.10.4) and have no
  list to append a test fake to, and D166 forbids a production hook, so their composition
  is tested by shape, not by a fake: (a) a `go/parser` test, as for `tick.go` (D141),
  reads the body of each of `fillAgentView`, `checkTradePlan`, `removeAgent` and
  `relocationRefusal` and checks that it consists only of calls, `if` guards (`if w.obs.supplied`, `if` the agent has a job) and
  composite literals (event values), with no `for`/`range` and no arithmetic on state (round 4 NB-11), and that the calls are, in order, the expected list; a module PR
  edits the expected list in the same change; (b) the order of effects is observed from
  outside: `removeAgent` through the event order and the ledger flow order of an actual
  removal (Phase 7), `fillAgentView` through the view it returns. What is not testable
  without a seam: that a module step appended later is called in its slot; that is the
  module PR's own test.
- **D196.** (Round 3 fix.) The refusals of the spawner's plan (Phase 7). The executor checks
  each decision with `occupation.Refuse` against live state; a refused decision changes
  nothing, adds one to the region's `entryTally` `Refused` (and nothing else) and writes the
  `Debug` line "decision refused" (`agent`, `reason`, from `refusalReason`) behind
  `Enabled`; the plan continues with the next decision (FR-SPN-004).
- **D190.** (Round 1 fix.) The failure mappings are pure functions, tested alone,
  because the real `Ledger.Audit` cannot be made to fail from the root:
  `ClearTotals` re-anchors the boundary bases to the balances, every ledger
  operation conserves, and D166 forbids a seam. `auditFailure(tick int64, err
  error) error` maps the ledger's result at P8.4 (nil to nil; otherwise an error
  wrapping `ErrAuditFailed` and the `*ledger.AuditError`, naming the tick) and
  `finishConstruction(auditErr error) error` maps it at step 9. The bracketing
  (`Failed` true, the world failed before `TickEnded`, the tick not advanced, the
  totals not cleared) is tested by an internal test that calls `beginStep()` and
  `endStep(auditFailure(t, synthetic))`. That `p84Audit` calls `Ledger.Audit` is not
  testable without a seam and is said so in Phase 8. The construction draw count is
  likewise a state assertion on the PCG (Phase 3), not a counting wrapper.
- **D191.** (Round 1 fix.) The application of the warm start is this row's
  (plan 1-10 Provides gives only `virtualStartOffsets` and the validated table). It
  is built in `construct_warm.go` in the sub-order of WLD 5.6.1 and VIA 8.4.3, 8.5.2,
  8.5.3, as the deliverables 3a to 3h of Phase 3: the helpers (`apportion`,
  `phase`), the prices, the wages, the money and stock with the stagger, the jobs'
  selection and virtual start, the jobs' books (one construction `Transform`, tool
  wear), the pools, and the line and guards. It takes no random number and never
  calls the solver.
- **D192.** (Round 1 fix.) The executor of the spawner's plan has its own phase and
  its own tests (Phase 7): it composes, in plan order, the primitives of rows 5, 6,
  7, 8 and 9, and the world holds the renewal carry per (region, role) that it
  advances (D145, N-265).

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The names of the agent, estates, spawner and viability rows are not yet
  fixed; every symbol this plan uses from them is in "Consumes" as ASSUMED with
  the owning row. Before Phase 3 starts, the implementer matches each ASSUMED
  name against the merged code of rows 7 to 10 and corrects this plan's table,
  not the other rows (plan 1-3 A-style "names are indicative of the capability").
- **A2.** The per-agent record of the slice is the agent row's; the world holds
  an agent table indexed by agent slot (`AgentID` is dense in creation order,
  pseudo-agents first, WLD 5.3) and per-slot scratch (welfare scratch, the
  rotation). If the agent row stores its fields in slices indexed by slot
  (architecture.md 2.10.1) the same calls serve.
- **A3.** One region is the slice, but every loop is written over regions, so a
  second region needs no change (WLD 6.1.2: ascending `RegionID`).
- **A4.** A pack with no group is a valid configuration for the shell tests of
  Phase 2 only if `Validate` accepts it. If it does not, Phase 2's `Run` tests
  use the smallest valid pack (`testdata/chains/one.toml`) with its agents.
- **A5.** The `Debug` lines of the other rows (CLR "order rejected", "market
  cleared"; BEL "belief updated"; AGT; PRO "recipe not started") are emitted by
  the world at the position that has the facts, behind `Enabled`; each row's
  plan gave the line and its keys, and the Phase 10 table cites them. A `Debug`
  line is not required by a test of this row except the three that the spec
  names (OBS 7.5).
- **A6.** The snapshot is built with one pass per scan class of OBS 5.5 and
  allocates only its own rows. Its allocation count is bounded in Phase 9 but
  its absolute cost is not measured (N-015: counts, not timings).
- **A7.** Phase 2 builds the derived tables (including `dayTable`) and the ledger with
  `newLedger` (the built-in boundary accounts only), so that `Tick()`, `Step` and `Run`
  work; Phase 3 adds the estate accounts (order 4 of WLD 5.2) and the agent accounts
  (order 6). No agent exists before Phase 3.
- **A7b.** `Step` on a world with no agents is legal (a configuration whose groups
  have 0 households fails `Validate`; a pack with one region and the pool alone
  is the shell test). If not, the shell tests use `one.toml`.
- **A8.** The `Warn` volume of the slice is far below the bound of OBS 7.3.1; the
  log tests use the cells of `one.toml`.
- **A10.** `NewWorld` reads the viability work once if row 10 allows it (N-276); see Consumes.
- **A9.** Rule edits (architecture.md 2.6 and 2.10.3, testing.md 6.8) and the
  spec edits are made by the Phase 1 implementer, each with a fresh tag. If other
  rows' Phase 1 edit the same rule file, the edits are small `Edit` calls after a
  re-read (N-103, the lesson of plan 1-3 and 1-4).

## Phase 1: settle the owed spec lines, the rules and the notes

**Why first:** the row cannot start code until the type name collides no more,
the position table has a test form, the files are in the layout, and the places
where the specs disagree with the built code are corrected. This phase changes
documents only, starts by reading `doc/notes.md` in full and quoting its tag,
handles the open notes tagged WLD, OBS and `ALL` (the table below), and gives
`doc/notes.md` a fresh tag (`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **`spec-config.md` 3.7 and 9.2** (D140): the section type is `WorldParams`, `Config.World` and `[world]` are
   unchanged; Phase 1 corrects the lines `Config.World` / `World World` of `spec-config.md` 3.7 and 9.2. The scope cells "World" of the parameter-owner table of `specs/README.md` and the `DefaultWorld` / `WorldParams` mentions of the `doc.go` glossary are **not** edited here: `registry_test.go` reads each README scope cell as a Go type name (`typesOf` names a section by `t.Name()`) and fails "scope WorldParams is neither built nor listed in notBuilt" while the type does not exist, so they change in the Phase 2 commit with the rename (round 4 NB-2). The owner specs' own tables keep the scope word "World", and `TestParameterRegistry_SpecDefaultsEqualConstructors` keeps its key on it.
2. **`testing.md` 6.8 and `architecture.md` 2.10.3** (D141): the staged form of
   the position table test with the `notBuiltPositions` list, with fresh
   RULE-TAGs in both. Roadmap Stage 1 "Done when" already names the test.
3. **`architecture.md` 2.6 and `AGENTS.md`** (D144): the files `construct.go`,
   `construct_warm.go`, `fanin.go`, `phase_start.go`, `phase_production.go`, `phase_trade.go`,
   `phase_outcomes.go`, `phase_exit.go`, and (the omission of N-243) the file
   `production_report.go`; a fresh RULE-TAG. Rows 7, 9 and 10 edit the same lines
   (`internal/household`, `internal/occupation` with `spawner_report.go`, and
   `viability_check.go` with `viability_table.go`) by small edits: re-read the lines
   first, add only the entries that are missing (an entry a sibling has added is not
   added twice), and replace the tag once per edit. N-235's combined edit (the fifth
   fan-in, the admission touch point) stays owed to the first Stage 2 module and
   is not made here.
4. **`spec-world.md` 3.2.2, 3.7.1, 16.1, 23** : (a) `ErrWorldFailed`, `ErrInStep`
   and `ErrAuditFailed` are root sentinels with the first users of D162; (b) the
   readers of the slice (D148) and the statement that `Warnings`, `Calendar` and
   the others are INT's; (c) the names `Err`, `Tick`, `Config`, `ConfigHash`
   are confirmed (they were provisional); (d) the file list of 23 follows D144.
5. **`spec-world.md` 6.2 to 6.10, the P5.4 and P6.3 rows**: the calls the earlier
   plans assumed the world makes are written as one list (D158, D157, N-242): the
   welfare scratch calls, the anchor set on the book before clearing, the
   `refusalTally` flush at P2.8a, the `Warn` runs advanced for every cell every
   tick (N-230). The rows already carry them as prose; the edit adds one sentence to 6.1: "the owner of each count calls it at its position".
6. **`spec-observability.md` 5.3.1 and 16.5**: `Seed` is the effective seed (the
   text says "for information"; WLD 12.3 and ADR 0008 make it the seed read
   back); `WorldRestored` is not in the slice.
7. **`doc.go` glossary** (architecture.md 3.2, N-017), before the code uses them.
   `doc.go` already has entries "World", "Rotation", "Bucket", "Names table",
   "Construction bucket", "Construction phase", "Market", "Tick" and "Executor": these
   are edited, never added a second time (`glossary_test.go` fails a term that appears
   twice). "World" gains `WorldParams` as the section (in Phase 2, with the rename); "Executor" is the job's
   (production), so the spawner plan's executor is named in the "Step" entry or by
   another word, not as a second "Executor". New entries: Step, Tick boundary,
   Phase, Position, Entry (of a tick), Failed world, In-step guard, Market record,
   Snapshot row, Fan-in. **"Run" and "Snapshot" are not added in Phase 1** (round 4 NB-3):
   both are in `glossary_test.go`'s `notYetTerms` and an entry present while the term is
   still listed fails "stale entry"; the entry is added in the PR that removes the term
   (Phase 2 for `Run`, Phase 9 for `Snapshot`). If `world.go` or `snapshot.go` joins the
   file list of `TestGlossary_DeclaredTypesNamed`, `doc.go` must name each exported type
   as an identifier: the "Snapshot row" entry names `RegionRow`, `RoleRow`, `MarketRow`,
   `JobRow`, `MarketState` and every other exported type of Phase 9 item 1, and the
   "World" entry names `World`.
8. **`doc/notes.md`**: the dispositions below, the index, a fresh tag. The notes
   N-284 to N-289 already exist (written with this plan) and so does N-290 (O10, the
   reduction cause: areas INT, POP, LED, CODE; no disposition is owed, it carries no WLD,
   OBS or ALL tag); Phase 1 verifies all seven and
   corrects N-287 (the file `construct_warm.go`) and N-288 (the renewal carry and
   the "had an agent" bit are saved state and not a diagnostic run, D145) in place,
   and adds none twice.
9. **`doc/roadmap-v1.md`**: verify that the row 11 cell links this plan, counts its
   eleven phases and states its notes (the section under the table: the type rename,
   the staged position table, the replay seam), and correct it if not.
10. **`doc/documentation_index.md`**: verify the one row for this plan (eleven
    phases, the new files) and correct it if not.
11. **`spec-viability.md` 8.5.2 and `spec-scale.md` 7** (O7): the apportionment of
    the agents in progress states its tie-break and how the `JobSpec`s of one group
    share the agents, as ruled (O7, closed), before Phase 3's deliverable 3a starts.
12. **`testing.md` 6.8 and `spec-world.md` 6.1** (O8): what the position test
    compares as "owner" (the first area code; "this spec" reads as WLD), what P0.2 is,
    and the `where` field of a list entry (`line` or `bracket`), as ruled (O8, closed),
    with the fresh tag of deliverable 2.
13. **`spec-viability.md` 8.5.3 and `spec-world.md` 5.6.1** (O11): one sentence in each: the
    job's input `Transform` consumes oldest first, so the stagger vector is a property of the
    granted stock before the job's inputs are taken; the post-warm stock is the stagger lots plus
    the granted input lots (fresh, Born 0) minus the oldest-first consumption of the input quantity,
    so its total equals the stagger total (granted equals consumed).

No code in this phase.

**Tests:** none beyond the text checks of N-103: every `spec-x.md N.N` pointer
this phase writes resolves; the literal `'"'"'` search of `doc/` is clean; the
notes index equals the areas of the open notes; the read-receipt tags changed
only for the files edited.

**Observability:** none, because documents only. The table of OBS 7.3 is not
changed by this row (the `Warn`s of the slice are rows 2, 3, 4, 6, 7, 8, 19, 26
and exist).

**Requirements:** FR-WLD-002 (the registration halves are the existing rule),
FR-WLD-040 (positions), FR-WLD-110, FR-VIA-034 (the rule stated, O7), N-016, N-126 (no new event or `Warn` row).

**Acceptance:** the documents above carry the lines; `go build ./...` is
unchanged; the notes index matches.

### Disposition of the open notes

Notes tagged WLD, OBS or `ALL` (and the others that name the world row), read at
Phase 1. "Apply" means this row's change does what the note asks; "leave open"
means it is not this row's to settle, with what was learned.

| Note | Disposition |
|------|-------------|
| N-016 Keep the parameter registry true | Leave open. The row adds no configuration field. The rename of D140 changes the scope cells of `specs/README.md` (they are read as Go type names), in the Phase 2 commit with the rename and not in Phase 1 (round 4 NB-2) |
| N-017 Words and names | Apply: the terms of Phase 1 deliverable 7 go to `doc.go` before use. Standing; leave open |
| N-027 No default macro constant is stable | Leave open. The row sets no default |
| N-103 Running several spec writers in parallel | Leave open. Process: the shared files are edited by targeted edits after a re-read; the `'"'"'` search is run after Phase 1 |
| N-126 A new event or `Warn` adds a row | Leave open. The row adds no event and no `Warn` row (all are rows already); the catalogue test is staged and this row's Phase 10 moves the world-emitted events out of `notBuiltEvents` only if they were listed |
| N-128 A hosted per-agent field is saved once | Leave open. The world holds no per-agent field; the ring, the job and the residues are the agent row's hosted fields |
| N-163 The word "treasury" | Leave open. The row says "pool" for the estate account and does not use "treasury" |
| N-217 D5 binds every module's change | Applied: the row appends no configuration, event or snapshot field to anything that exists; the snapshot is new and declares slice fields only; the pinned hash and `testdata/golden/off/` are not moved (the rename of D140 changes no hash input). Standing; left open |
| N-235 One combined architecture.md edit | Leave open (owner: the first Stage 2 module). The row adds the four fan-ins of 2.10.4 as they are; the fifth and the admission touch point are not needed in the slice |
| N-124 A paired comparison is exact only while aligned | Leave open for EXP, STB, CLB. The draws are keyed (ADR 0008); the world adds none (FR-WLD-062) |
| N-125 A snapshot is a function of saved state | Apply for the slice: every field of D146 is derived from the saved agent, ledger, market and welfare state; the market record's `marketReport` and window are saved state (CLR 16, BEL 7) and the world holds them in one place (Phase 3). Leave open for CHK |
| N-144 A busy employer cannot post vacancies | Leave open (EMP, Stage 3). `Work` is called only for a free agent at P2.5, as written |
| N-154 A count of households gates a structure | Read; the world reads no count into a rate (D156 diagnostics compare held with half, a threshold of a `Warn`, not a decision). Leave open |
| N-204 A slot has two writers | Leave open. The slots are Stage 2 and P0.3a is a module position; the row's `beginStep` leaves the place |
| N-214 Keyed random streams (the edits owed) | Apply for the world's sites (D155): D1, D7, D8, D10 are called with the keys of WLD 7.3; the draws of D2, D4, D6, D11 to D15, D17 are modules'. Leave open for the others |
| N-216 A site constant is never a small integer | Read; the row adds no site |
| N-226 The creation position of the boundary accounts of the other module reasons | Leave open. The slice has none; the world creates the accounts in the order of 5.2 rows 1, 4 and 6 and leaves 2, 3, 3a, 5, 5a to the modules |
| N-229 Registration names | Apply for the options: `NewWorld`'s name checks call `resolveClearing` and the strategy and spawner rows' `resolve*` in that order and report a `*ValidationError` with the path of the field; `registry[T]` is reused unchanged (Phase 2) |
| N-230 The run of "price beliefs at bound" is advanced for every market every tick | Apply (D156, Phase 6): `beliefs.NextRun` is called for every market every tick and `BeliefsUpdated` is emitted before the `Warn`. Whether the double count of a pinned market matters stays open for S01/S04 (row 12) |
| N-239 `AverageClearing` can produce a self-trade; the check fails the step | Apply: the step fails with `ErrInvalidTrade` and the world is failed (D151); the test uses a faulty `Clearing` fake (Phase 5, Phase 10) |
| N-240 The book holds its entries and filled units | Apply: the world owns one `clearScratch` and one `book` per market and keeps each `fill` until the outcomes are delivered (D154, Phase 6) |
| N-242 What the world row calls for the production observability | Apply (D157, D158, Phase 4 and 5): the `refusalTally` per world, `add` at a refused free household's request, `flush` at P2.8a; `book.anchor` set before each clearing; `RecipeExecuted` built with `recipeExecuted(outs, disp, over, ...)` and the "recipe output disposed" `Warn` fed from `Overflow`; the allocation test of the production phase at zero with the tally |
| N-127 Event classes at the observer | Leave open. The row measures nothing; `NFR-OBS-002`'s baseline is row 12's |
| N-130 What IND cannot measure exactly | Read; the row emits `WearApplied` at P8.2 (item 2 of the note) and adopts none of the asks |
| N-159 Region-attributed flows | Leave open (FIS/IND); the world reports `Pending` and `Construction` as the ledger gives them |
| N-019, N-198, N-222 Housekeeping, drawdown cost, reseed cost | Leave open. Phase 10 records the allocation count of a site call; nothing is measured in time |
| N-224 Stage 0 guards | Apply: `guardedFiles` is a directory walk and needs no edit (round 4 NB-3); `deferredCore` loses nothing (the events of the slice are built), `notYetTerms` loses `Run`, `Snapshot` |
| N-232 The job and the residues live on the agent | Leave open for row 7 (stores the fields); the world calls the steps at the positions of 6.4 |
| N-234 The wear cost takes no second factor of the weight | Apply as built: the world hands `addWear` the executions started (D158) |
| N-236 Three halves of FR-AGT-003, 010 and 011 | Leave open for row 7 |
| N-237 A job stores its effective factor | Leave open for rows 7 and Stage 2 |
| N-243 Follow-up of the Codacy gate | Apply: each file of the row passes `lizard -l go -C 10 -a 8 -L 100 -w`; the tables of Phase 1 use the spaced separator row; D144 lists the new files |
| N-244 A module's per-agent state is in the module's slices | Read. The world holds no per-agent field of a module; row 7 builds the unexported agent with the fields no module owns, and the checkpoint writes the rest (Stage 2). Leave open |
| N-245 A tunable a module's default rule reads joins the view's Tunables group | Apply for the slice: `fillCore` (row 7, called by `fillAgentView`) fills the belief-parameter group of `AgentView` from `beliefParams`, converted inline, and the world builds `viewCtx` (`refreshViewCtx`, Phase 4); the world fills the belief group of `OutcomeView` itself in P5.4 (plan 1-7: `fillCore` fills `AgentView` only; Phase 7); a module appends its tunable to the same group. Leave open |
| N-246 What the world row must do for the strategy streams and the draw count | Apply (D155): the stream is keyed before each call to an extension, `siteTrade` before every `Trade` (the default takes one word per intent it considers, in canonical commodity order), `siteWork` only for a consumer's `Work` (the default takes none, so the reseed is skipped); for `Revise` the outcome's word `Draw` is taken first from `(siteOutcome, tick, agent, commodity)` and the same `*rand.Rand` is passed on; the creation draws are `drawCreation` (Phases 3, 5, 6, 7). Leave open for row 7 |
| N-255 What `GoodsListed` counts | Apply as ruled (lots): `removeAgent` and the executor count lots through the one function every caller shares; units travel in `WeightChanged.Units` (Phase 7). Closes with plan 1-8's Phase 1 |
| N-257 FR-EST-010 asks for a snapshot marker that OBS 5.1.2 does not have | Apply as ruled: the estate has no per-agent snapshot row, so no marker is built; its money is `Regions[].MoneyPool` (Phase 9). Closes with plan 1-8's Phase 1 |
| N-258 What the world row calls on the estate | Apply: the estate is reached only through `estates.go`, at the positions of the note (Phases 3, 5, 6, 7); a text-search guard in Phase 7 fails a second ledger call on the estate account for money |
| N-265 The renewal carry and the three runs are the world's state | Apply (D145, D192): `worldState` holds the carry per (region, role), the "had an agent" bit and the three runs; the executor advances the carry with `occupation.Renewal`; the Guard test the note asks for is in Phase 7. Leave open for CHK (tag 9) |
| N-266 What the world row calls for the spawner | Apply (Phase 7): the five items of the note are Phase 7's deliverables 2 to 4 and the `resolveSpawner` call of Phase 2; `GoodsListed` counts lots |
| N-276 `Validate`, `Warnings` and `NewWorld` each reach the programmes | Apply (A10): `NewWorld` calls `Validate` once and reads the warnings from the same outcome if row 10 exposes `checkViability`; otherwise it accepts the double work, which row 10's Phase 7 measures. Leave open for row 10 |
| N-280 One function gives the virtual start offsets | Apply: piece 3e calls `virtualStartOffsets`, so the rule of `validate.go` and the construction cannot disagree (FR-VIA-053); the stagger and the application are this row's. Leave open |
| N-294 The checkpoint change adds the digest to the off-only golden once | Read. No checkpoint item in Stage 1; the world adds none. Leave open |
| N-295, N-296 The packs the slice cannot run; S01 leg (b) | Read and applied: Stage 1 runs `one` and `second` (with `two` and `warm-two` for the interleave test only); no `famine`, `five` or `three` in this row's acceptance or scenarios (Phase 3, Phase 10 item 2); the shock seam is row 12's. Leave open |
| N-297 The per-position hash replays the step through unexported methods | Apply (D166): every position is an unexported method, `beginStep` and `endStep(err)` are the non-position parts, no hook. Leave open for row 12 |
| N-300 The first measurement of a `Step` of `one` | Leave open for row 12. This row's allocation test asserts the call sites only and its benchmark reports (Phase 10) |

New notes of this plan (N-284 to N-290, written in Phase 1): N-284 the type
`World` of the section against the simulation type (WLD, CFG, CODE); N-285 the
staged position table (WLD, CODE); N-286 the replay seam of the step (WLD, CODE);
N-287 the files of the world row (CODE); N-288 what the world saves and the
diagnostic runs (CHK, WLD, OBS); N-289 the slice's snapshot and `MarketState`
(OBS, INT); N-290 the cause set of a reduction (INT, POP, LED, CODE; O10).

## Phase 2: the rename, the sentinels and the shell of `World`

**Why here:** everything later hangs on the `World` type, `NewWorld`'s first
steps and the guards, and none of it needs an agent. This phase is buildable and
testable with the structures that exist.

Deliverables:

1. **The rename** (D140): `World` (section) becomes `WorldParams` and
   `DefaultWorld` becomes `DefaultWorldParams` in `config.go`, `defaults.go`,
   `hash.go` (`pairsWorld`), `bounds.go` (`worldRanges`), `validate.go`
   (`resolutionFloor`), `beliefs.go` (`beliefParams`), `config/`, the tests and the
   registry test, **and in the same commit the scope cells "World" of the parameter-owner table of `specs/README.md` (they become `WorldParams`) and the `doc.go` mentions** (round 4 NB-2: the cell is the Go type name, there is no scope map, so the README and the type change together and `TestParameterRegistry_Config` is never red between phases). `go build ./...` and the whole suite pass unchanged
   in behaviour; the pinned hash does not move (FR-CFG-105).
2. **`errors.go`**: `ErrWorldFailed`, `ErrInStep`, `ErrAuditFailed` (D162).
3. **`world.go`**: the `World` struct (D154, D143, D145 fields; the ledger, the
   canonical `Config`, the derived tables `recipeTable`, `roleTable`, `dayTable`,
   `yieldTable`, `yieldShocks`, the `options` value, the stream), the doc comment
   with "not safe for concurrent use", `NewWorld` steps 1 to 6 of WLD 5.1 (apply
   options in order, copy and canonicalise the configuration with
   `WithTargetSpread`, `Validate`, the name checks after `Validate` succeeds with
   the path of the field (`resolveClearing`, `resolveStrategy` for every role,
   `resolveSpawner`: the world half of FR-CFG-038), the hash, the derived tables
   including `dayTable`, the seed and the stream, and the ledger by `newLedger` with
   the built-in boundary accounts only: A7), returning `(nil, err)` on any failure
   (FR-WLD-001). `Err`, `Tick`, `Config`
   (a deep copy), `ConfigHash`.
4. **`Step` and `Run`** over the non-position methods of D166 with an empty
   straight line (only `p02()`): `beginStep` (guards `ErrInStep`, `ErrWorldFailed`, the
   stepping flag set, `TickStarted`), `endStep` (`TickEnded`, failed marking,
   tick advance, **the stepping flag cleared as its last act**: row 12's replay calls only
   `beginStep` and `endStep`, so the clear is in `endStep`; plan 1-12 audit round 3,
   NB-1), `Step` with the deferred function of D167 that clears the flag on the panic
   path (D143), and `Run` (D152, D168, the table of WLD 3.4.2). `Step` sets its local
   `finished` to true when `beginStep` returns an error, so that function does nothing
   for a refused call: a refused inner call (`ErrInStep`) never clears the outer step's
   flag and a refused call on a failed world (`ErrWorldFailed`) never overwrites the
   real cause (D167, round 3 NB1).
   `func (w *World) Audit() error` is built here too (D149, round 3 NB11): `ErrInStep`
   or `ErrWorldFailed` under the guards, else the ledger's `Audit` (no module check in
   the slice).
5. **`tick.go`**: the `position` struct (identifier, owner, `where`: D141), the list value, the
   one entry `func (w *World) positions()` that is the straight line (in Phase 2 only the
   call of the empty `p02()`), and
   the unexported `notBuiltPositions` list holding every position of WLD 6; the
   position table test (D141) reading `spec-world.md` 6 by a plain text parse and
   the straight-line function by `go/parser` (the owner is compared as the first area code of the Owner cell, "this spec" read as WLD; P0.2 is an empty `line` method with a one-line comment and the parse does not special-case it; the `bracket` entries are P0.1, P0.2a, P0.3, P8.5, P8.6: O8, D141). Staging of the `bracket` entries (round 4 NB-4): Phase 2 deletes P0.1, P0.3, P8.5 and P8.6 from `notBuiltPositions` with the shell; Phase 4 deletes P0.2a. Because the parse expects no call in the straight line for a `bracket` entry, a stale entry cannot be seen there, so the test also fails when a `bracket` entry is listed `notBuilt` other than P0.2a before Phase 4 (and any, after). This NB-4 test rule is FINAL (ruled by coordinator, reversible: it only tightens a test).
6. **`doc.go`**: the terms of Phase 1, and the entry "Run" added in this PR with `glossary_test.go` losing `Run` from `notYetTerms` (round 4 NB-3).
7. **`observe_helpers_test.go`** (external test package; round 3 NB8): the recording
   `Observer` (`recorder`, field `events []Event`, method `Observe`) and the capturing
   `slog.Handler` (`captureHandler`; level, message, attributes; `Enabled` true for every
   level) are built here, at their first user, so the event and log tests of Phases 2 to
   9 and row 12 reuse one file; Phase 10 extends it and does not recreate it.

**Tests** (external `_test` package; the shell tests use `one.toml` with its
agents if Phase 2 has none, A4):

- Unit: option halves, table-driven: options applied in order, the last
  `WithSeed` wins, a nil logger or observer is the default and not an error,
  `WithObserver` additive and ordered, a `WithTargetSpread` out of bounds fails,
  an unregistered clearing name is `ErrUnknownClearing` with a `*ValidationError`
  path, a registered name given twice is `ErrDuplicateID`, an invalid name is
  `ErrInvalidID`, a nil clearing is `ErrNilExtension`, a role naming an unknown
  strategy is `ErrUnknownStrategy` at `roles.<id>.strategy` and an unknown
  `world.spawner` is `ErrUnknownSpawner` (the world half of FR-CFG-038; FR-WLD-001,
  002, 003, 020, 021; FR-OBS-090).
- Unit: the hash and the options (FR-WLD-020, 021): `WithSeed`, `WithLogger`,
  `WithObserver` and `WithAudit` leave `ConfigHash()` equal to `HashConfig` of the
  same configuration, `WithTargetSpread` changes it; every option that can change a
  result is a datum of the effective configuration or a registered name (a table
  over the options of `option.go`).
- Unit: the guards. An observer that calls `Step` or `Run` from inside a step
  gets `ErrInStep` and changes nothing; a failed world returns `ErrWorldFailed`
  wrapping the cause from `Step`, `Run` and `Audit`; `Tick`, `Err`, `Config`,
  `ConfigHash` work on both (FR-WLD-006, 007, 110 for the readers that exist); a
  refused inner call does not clear the outer step's flag, so a second inner call
  in the same step is refused too; two worlds stepped on two goroutines under `-race`
  share nothing (FR-WLD-011).
- Unit: `Run` table of WLD 3.4.2: completes, cancelled before tick k (a context
  cancelled by an observer on the k-th `TickEnded`), `ticks` 0, `ticks` negative
  is `ErrOutOfBounds` with no event, a failed or in-step world emits neither
  event (FR-WLD-008, 009, 012).
- Unit: a panicking observer leaves the world failed, the panic propagates, no
  `TickEnded` is emitted, and a `Run` that ends in a panic emits no `RunEnded`
  (FR-WLD-006, 012; FR-OBS-013, 024 half). A logger with no observer still receives
  "run started" and "run ended" (D152).
- Unit (internal), the bracketing: `beginStep()` then `endStep(err)` with a synthetic
  error marks the world failed before `TickEnded`, emits `TickEnded` with `Failed`
  true, leaves `Tick()` unchanged and clears no totals; `endStep(nil)` advances the
  tick (D190, D161).
- Guard: a text check over the new files finds no `range` over a map (FR-WLD-041:
  the position test cannot show it).
- Unit: the position table test (D141) passes with every position listed `notBuilt`
  except P0.1, P0.2, P0.3, P8.5 and P8.6 (the shell's `beginStep`, `p02()` and `endStep`;
  P0.2a stays listed until Phase 4), fails when the list is edited to remove a real position (a
  mutation on a copy).
- Unit: a refused call on a failed world keeps the real cause (round 3 NB1): after a
  failing step (a `endStep` with a mapped error, or an observer panic recovered by the
  test), a second `Step` returns `ErrWorldFailed` wrapping the same cause, `Err()` still
  returns the original error (same value, not a new "panicked" error), and a third call
  does the same.
- Unit (internal), the flag on every path: after a panic in an observer (the only
  place a panic can come from in this phase: the straight line holds the empty `p02()`
  and no other position exists yet; a panic in a position is tested in Phase 4, once one
  can fail) and a `recover` by the test, `Step` and `Snapshot`/`Audit` return
  `ErrWorldFailed` (not `ErrInStep`; the `Snapshot` call is added to this test in Phase 9); after a replay-style `beginStep()` then `endStep(nil)` (no `Step`),
  the flag is clear and `Audit` works (FR-WLD-006, 007).
- Fuzz: none (no parser).

**Observability:** `Info` "run started" and "run ended" (D152); the names and
keys are constants of `internal/logkeys` (`ticks`, `reason`); tested with the
capturing handler of deliverable 7. No `Info` in the
tick (the shell has none).

**Requirements:** FR-WLD-001, 002, 003, 006, 007, 008, 009, 011, 012, 020, 021,
041 (the map guard), 110; FR-CFG-038 (the world half); FR-OBS-012, 013, 024, 090;
NFR-WLD-003.

**Acceptance:** `gofmt -l .`, `go vet ./...`, `staticcheck ./...`,
`go test -short -race ./...` pass; the pinned hash and every existing test pass
after the rename; `Run` on the shell world matches the table; the position test
passes with the list.

## Phase 3: construction, the warm start and the market record

**Why here:** steps 5 to 10 of WLD 5.1 produce the world every position reads, and
`market.go` is the state the clearing and beliefs rows left to this row. They need
rows 7 to 10 (A1, D142), and nothing of the tick. The warm start is the largest
piece of the construction: its table is `Config.WarmStart` data (plain slices) and
the solver is never called (FR-VIA-021). Row 10 gives this row `virtualStartOffsets`
and the validated table and nothing else (plan 1-10 Provides), so the selection,
the stagger, the grants and the jobs are this row's and are broken into the
deliverables 3a to 3h (D191); one commit each.

Deliverables:

1. **`market.go`**: the record of one market (`region`, `commodity`, the
   `marketReport` from `seedReport(seedPrice)`, the `beliefs.Window` sized by
   `PriceWindow` and `MaxPrice`, the `BoundRun` count (an `int64`, `beliefs.Run.Count`; `Run` itself is the per-tick result, `internal/beliefs/tally.go:70`) of the "at bound" `Warn`, the
   `book`), the table of markets in (`RegionID`, `CommodityID`) order for every
   region that is not an outside node (6.1.11), the lookup by pair, the push at
   P5.1 (only a market in which units traded), and the seed price (warm start
   `Prices`, else the effective reference price). The exported `MarketState` and
   its fill are Phase 9 (D147, N-289), not this phase.
2. **`construct.go`** (steps 5 to 10, without the warm start). The ledger exists
   since Phase 2 (`newLedger` creates the built-in boundary accounts, row 1) and so
   do the derived tables including `dayTable`; this phase adds, in the order of
   WLD 5.2: the estate accounts (row 8 `openEstates`, order 4), the strategy table
   (`resolveStrategy` for every role into an index that is passed to `newAgent`,
   row 7), the agent ID space of 5.3 (estates `0..n-1` with the ID equal to the
   `RegionID`, then the households in (region, role, index) order) and the agents.
   Row 7's `newAgent` has no ledger access, so **the world opens each agent's
   ledger account with `Ledger.Open(KindAgent, owner, region)`** (and closes it with
   `Ledger.Close` in `removeAgent`, Phase 7): FR-SPN-044 depends on it. The agent
   gets its weight per group, its money and stock by `fundInitial` (reason `initial`, unstated lot keys) for a group with no warm entry, and for a warm entry the stock by `Ledger.MoveStated` (3d; never `fundInitial`, plan 1-7 Provides: unstated keys only; the money half may reuse `fundInitial`), the seed price for its beliefs (3b) and no draw. The creation
   draws are `drawCreation(s, a, targetRatio, spread)` of row 7 (the two words in
   order, the target ratio through `newTarget` and the residues through
   `setResidues` inside it; the same function serves P7.1(d), Phase 7;
   FR-WLD-034, 066). The agents and weights of each group come from the pure function
   `groupWeights` (deliverable 2a, round 3 NB4). The per-region slices that `fillCore`
   reads (`viewCtx`: the reports, the published prices, the anchors, the season; filled by
   Phase 4's `refreshViewCtx` at each use) and the
   `runnable [][]RecipeID` per region that `productionScratch.anchors` needs
   (`production_report.go:134`: the recipes some agent has on its role's list, built
   at P1.2 from the counts of agents per (region, role), not by a scan of agents)
   are allocated here. `WithAudit` runs `Ledger.Audit` at step 9 and maps a failure
   by `finishConstruction(auditErr)` (D190; FR-WLD-036). The warnings go to `Warn`,
   "warm start applied" and "world created" to `Info` (D163), then `WorldCreated`
   and the `AgentSpawned` events in ascending ID (D169; FR-WLD-037) only when an
   observer was supplied. The construction phase is open (FR-WLD-035). The scratch
   of D154 is allocated here.
2a. **`groupWeights(households, scale, minAgents int64) []int64`** (in `construct.go`;
   round 3 NB4, D193): the pure derivation of the agents and weights of a group
   (`spec-scale.md` 3.2, which says it is inline in world construction, a small pure function
   with a table-driven test): `a = max(ceil(H / S), min(K, H))` agents, weights summing to
   `H` and differing by at most 1, the first `H mod a` agents taking the extra household,
   no random number (FR-SCL-001 to 005). Stage 2's `scale.go` ("weights, the census") takes
   it over unchanged. The stagger (3d), the jobs' selection (3e) and the account and ID
   tests of this phase depend on it. Table test: `H` 1 and `S` 1; `H` 10, `S` 4 (3 agents
   of 4, 3, 3); `H` below `K` (agents raised to `H`); `H` 40, `S` 40; `H` 7, `K` 3, `S` 100;
   the sum and the spread asserted for every row.
3. **`construct_warm.go`**, the application of the warm start in the order of WLD
   5.6.1 (D191). Row 10 is not asked for a reader: the table is read from
   `Config.WarmStart`. Each piece takes no random number (FR-VIA-040).
   - **3a. The pure helpers** (unit tested alone): `apportion(shares []int64,
     n int64) []int64`, the largest-remainder apportionment of the shares of a group
     over its `n` agents (FR-VIA-034, VIA 8.5.2; the tie-break and the sharing of
     the agents between the `JobSpec`s of one group are not settled by the spec:
     ruling O7: one joint largest-remainder apportionment, ties to the earlier entry, idle share last; written into VIA 8.5.2 in Phase 1), and `phase(i, n, p int64) int64 = floor(i * p / n)` through
     `checked.MulDivFloor` (FR-VIA-037, VIA 8.4.3), overflow checked
     (architecture.md 3.5).
   - **3b. Prices** (WLD 5.6.1 item 1, FR-VIA-031, 042): a `SeedPrice` replaces the
     effective reference price as the market's initial published price
     (`seedReport`) and as the centre of each initial agent's belief, clamped to the
     agent's bounds by `beliefs.Clamp` (row 7's `beliefSeed`/`reseedBeliefs`); a
     market with no row takes the effective reference price. The region and
     commodity strings are mapped to IDs once.
   - **3c. Wages** (item 2, FR-VIA-052): the world keeps `seedWages(cfg)`
     (`welfare.go:317`) and hands it to `readBuf.regionWage(rings, seed)` at every
     read of the cost anchor (P1.2); the choice is not latched (the figure is a
     function of the rings and the configuration), and a region with no `SeedWage`
     has an unknown wage while the pooled figure is unknown.
   - **3d. Groups: money and stock** (item 3, FR-VIA-032, 037, 038, 042): a group
     with an entry is created with `Money * weight` and `Stock * weight` by one
     `initial` in place of `StartMoney` and `StartStock`; a group with no entry
     takes the defaults. For the stock of agent `i` of the group's `n` agents (0-based,
     agent order) a perishable lot has `Born = -phase(i, n, ShelfLife)` and a durable
     item has remaining durability `Durability - phase(i, n, Durability)`, created
     through the ledger's construction-phase operation for a stated key
     (`Ledger.MoveStated`, `spec-ledger.md` 5.2.4; both stay inside the ranges of
     VIA 8.6 item 2). The cut of cover to the shelf life (FR-VIA-039) is in the
     solver's table (plan 1-10 Phase 6; Phase 4 computes the reserve) and is not applied again here.
   - **3e. Jobs: selection and virtual start** (item 4, FR-VIA-033 to 035, 053):
     for each `JobSpec` of a group, `busy` agents by `apportion` (3a), the first
     `busy` agents in agent order (O7), the `j`th takes the offset
     `V[floor(j * |V| / busy)]` of `virtualStartOffsets(cfg, calendar, yieldTable,
     region, recipe)` (row 10, sorted ascending; an empty set cannot reach the world
     because `Validate` returned `ErrWarmStartJob`, so the construction meets it
     only as a violated invariant). The job has `Remaining = Duration - o`, the
     effective factor of the season `Calendar.SeasonAt(-o)` (no other rule), the
     shock at its default, `Executions` equal to the agent's weight and the agent
     itself as principal, made by `productionScratch.warmJob(j, yieldTable, region,
     season, recipe, principal, executions, remaining)` with a residue of 0
     (WLD 5.6.2); no endowment is drawn.
   - **3f. Jobs: the books** (item 4, FR-VIA-036, VIA 8.5.3): the inputs of each job
     are granted by `initial` as separate lots at the default key and taken at once
     by `recipe_input` in one construction `Ledger.Transform`, the tools are worn
     by `Ledger.WearItems`, all inside the construction phase so the moves are in
     the bucket and the audit passes at tick 0 (FR-LED-046). **Ruled (O11, coordinator,
     option (a), the literal spec):** the input `Transform` consumes oldest first (GDS: oldest
     used first), so the stagger vector of 3d is a property of the granted stock **before** the
     job's inputs are taken; the tests assert the vector on the granted stock (before the
     `Transform`) and the post-warm stock as the stagger lots **plus the granted input lots** (fresh,
     Born 0) **minus the oldest-first FIFO consumption** of the input quantity; the total therefore
     equals the stagger total, granted equals consumed. Hand arithmetic for one perishable input
     `c`, `n = 4`, `ShelfLife = 4`, agent `i = 2`, weight 1, `Stock` 3, job input quantity 2:
     the stagger lot is `Born = -floor(2*4/4) = -2`, quantity 3; the grant adds `Born 0`, quantity 2;
     the vector before the `Transform` is `[(-2,3), (0,2)]` (total 5); the oldest-first take of 2
     leaves `[(-2,1), (0,2)]` (total 3 = the stagger total). 3d and 3f are separate unexported
     functions, so an internal test runs 3d to 3e, grants the inputs, stops before the `Transform`
     of 3f and asserts `[(-2,3), (0,2)]`; the black-box test through `NewWorld` asserts only the
     result `[(-2,1), (0,2)]`, because the pre-`Transform` vector is not observable there.
   - **3g. Pools** (item 5, FR-EST-015): `WarmStart.Pools` create each region's pool
     by `initial` into its estate account (`seedPool`, row 8; `WarmPool.Region` is
     a `string`, mapped to a `RegionID` first), 0 when absent. An initial agent's
     money is by `initial` and not from the pool.
   - **3h. The line and the guards** (FR-VIA-021, 040, 046, 047): "warm start
     applied" once at `Info` with the counts of price rows, wage rows, groups, jobs and
     regions (OBS 7.2) and no other warm-start line; `WorldCreated.WarmStart` true. A
     text-search guard test: `SolveSteadyState` is called by no file of `NewWorld`'s
     path, and `WarmStart` is read only by `construct_warm.go` and the Stage 0 and
     row 10 files (so no rule after tick 0 depends on the table, FR-VIA-047).
4. **`worldState`** (D145) and the counters `agents`, `households` read by the
   `Debug` line.
5. **Fan-in stubs are not made here**: `fanin.go` gains `fillAgentView` in Phase 4,
   `checkTradePlan` in Phase 5, `removeAgent` and `relocationRefusal` in Phase 7.

**Tests:**

- Integration, over `one.toml`, `warm-one.toml` and `second.toml`: the account IDs
  equal the worked count of `spec-ledger.md` (64 boundary accounts for the pack
  `one`, the estate account at 64, then the agents), the agent IDs are the estate
  first and the households in order (FR-AGT-001, first half and order), each
  agent has its weight, money and stock (`initial` once, a warm group replaces the
  defaults), and `Audit()` passes (FR-WLD-030, 031, 032, 033, 034, 035, 038).
- Fuzz (NFR-VIA-004, the stagger and apportionment half that plan 1-10 gives to this
  row; testing.md 7.4; round 3 coordinator item), internal test file
  `construct_warm_fuzz_test.go`, corpus in `testdata/fuzz/` of the root package. Requirement
  IDs: NFR-VIA-004, FR-VIA-034, FR-VIA-037. Two targets over inputs decoded from bytes
  and clamped to the valid ranges of the helpers (a failing input is committed with a
  `Fix:` comment, testing.md 7.5):
  - `FuzzApportion(shares, n)`: never panics; every result is a valid vector (length
    equal to the number of shares, no entry negative); the busy count of each entry is
    within `[floor(quota), floor(quota) + 1]` and **the sum never exceeds `n`**, so the
    total busy count of a group never exceeds its agents; the result is the same on a
    second call and **independent of iteration order** (the shares are read in the given
    canonical order only: the same slice passed through a copy gives an identical vector,
    and a version of the helper over a shuffled copy of the entries, with ties broken by
    the original index, gives the same vector permuted back: ties go to the earlier
    entry, O7); an unreachable overflow is an error and not a wrap (architecture.md 3.5).
  - `FuzzStagger(i, n, p, offsets)`: `phase(i, n, p)` never panics, lies in `[0, p)`, is
    non-decreasing in `i`, is `0` at `i = 0`, and equals `floor(i * p / n)` computed with
    `math/big` as the oracle (the test file may import it, as plan 1-10 notes);
    evaluating the phases of a group's agents in a shuffled index order gives the same
    value per agent (it depends only on `(i, n, p)`); and for a sorted offset set `V` the
    `j`th busy agent of `busy` takes `V[floor(j * |V| / busy)]`, an index inside `V`, so
    every chosen offset is a member of `V` (the set `virtualStartOffsets` returned) and
    the offsets are non-decreasing in `j`; two calls give identical results (no random
    number, FR-VIA-040). The offsets function itself is fuzzed in plan 1-10
    (`FuzzVirtualStartOffsets`); this target covers the world's use of its result.
- Unit (3a), table-driven: `apportion` over largest-remainder cases with `busy`
  0, 1 and `n` (for example 20 agents and a share of 5 113 bp: quota 10.226, so 10;
  and a share of 418 bp: quota 0.836, so 0 or 1 by the joint rule of O7), the
  sum never above `n`; `phase` vectors: a perishable of `ShelfLife` 5 over `n` = 4
  gives `Born` 0, -1, -2, -3 (`floor(5 i / 4)` = 0, 1, 2, 3) and a durable of
  `Durability` 10 gives remaining 10, 8, 5, 3 (`floor(10 i / 4)` = 0, 2, 5, 7);
  the offsets rule: `V` = [0, 1, 2, 3] and `busy` 2 give the offsets 0 and 2
  (`floor(j * 4 / 2)`).
- Integration (3b to 3g) over `warm-one.toml` and `warm-second.toml`: a seed price
  is the published price and the belief centre (clamped when it lies outside an
  agent's bounds); the sum of money created by `initial` is the sum over warm
  groups of `Money * weight` plus `StartMoney * weight` of the groups with no
  entry and no other money exists (FR-VIA-032); a group with no entry takes the
  defaults (FR-VIA-042); the stock of a group follows the stagger vectors above (asserted
  on the granted stock before the job's `Transform` by the internal test that stops between 3d and 3f, and the post-warm stock of a job agent as the
  stagger lots plus the granted input lots minus the oldest-first FIFO consumption of the inputs, total
  equal to the stagger total, O11; 3f wears the tools with
  `WearItems`, which wears the most worn item first, so the durability vector of a tool is asserted
  after that wear in the same way); a
  warm job has `Remaining = Duration - o` with the offset the rule gives, the
  season factor of `SeasonAt(-o)`, `Executions` the weight, the agent as principal
  and its pending output with residue 0; the construction `Transform` conserves
  goods: the bucket holds the `initial` and `recipe_input` rows, the flow totals of
  tick 0 are empty, `Audit()` passes at tick 0 (FR-VIA-036, construction half); the pool is
  `WarmPool.Money` by `initial` (FR-EST-015). The halves that need the tick are
  Phase 4's and Phase 6's: the run to the first completion (Phase 4) and the
  `SeedWage` choice (Phase 4 and 6), because `Step` has an empty straight line in
  this phase.
- Integration: construction takes no word outside D1, checked structurally (D190;
  see D190): the stream's PCG state after `NewWorld` (an internal test,
  `MarshalBinary` of the world's `stream.pcg`) equals a fresh PCG keyed
  `(siteConstruction, 0, lastAgentID)` advanced by the two words of that agent's
  creation draw, with and without a warm start and with and without an observer;
  the two words of the first agent equal the vectors of WLD 7.1.5 (the test
  computes them with `internal/ordering`); for every agent the target ratio and the
  residue equal those of a world built without the warm start (FR-VIA-040). That no
  other draw is taken is a property of the code, not of one run: it is guarded by
  a text-search test that `siteConstruction` is keyed only in `drawCreation`'s
  callers, and by row 12's pack-level hash. There is no counting wrapper, because
  `stream` is a concrete type and a seam would be production code (D166; FR-WLD-062,
  064).
- Integration: events and logs at construction: `WorldCreated` then the
  `AgentSpawned` events in ascending ID with `Tick` 0, no `BoundaryFlow`, the
  `Info` line keys and counts equal the configuration's (including the one
  "warm start applied" line with its counts, FR-VIA-046), one `Warn` per
  configuration warning (FR-WLD-037; FR-OBS-010, 076).
- Unit: `market.go`: the table order, the push only on a trade, the seed price
  rule, a restored window (`Values`, `Restore`) equals the original.
- Failure: the construction audit's mapping is a pure function and is tested
  directly (D190): `finishConstruction` over a synthetic `*ledger.AuditError`
  returns an error wrapping `ErrAuditFailed` and `NewWorld` returns no world. A
  hand-made inconsistent `Config` cannot make the real audit fail, because every
  ledger operation is atomic and conserving and `Validate` rejects the rest; the
  failing branch of `Ledger.Audit` itself is covered by
  `internal/ledger/audit_internal_test.go`, and this row does not claim to test it
  (FR-WLD-036, mapping half).

**Observability:** the construction lines and events above; none is per agent
except `AgentSpawned`, which is a state transition of one agent (rules 4.3).
The construction counts are not computed by a scan at the `Debug` line (D163).

**Requirements:** FR-WLD-030 to 038, 062, 064, 066, 076 (the Stage 1 half: pseudo-agents, then households, the warm start, the creation draws and no other construction word; the later sub-steps are modules'; tested by the ID, account and PCG-state tests above); FR-LED-052; FR-AGT-001 (first
half); FR-SCL-001 to 005 (`groupWeights`); FR-EST-015; FR-VIA-021, 031 to 038, 040, 042, 046, 047, 052, 053 (the world
side; the offsets function and the cut of FR-VIA-039 are row 10's); NFR-VIA-004 (the
stagger and apportionment fuzz; the offsets and solver fuzz are row 10's); FR-OBS-010,
076; NFR-WLD-003.

**Acceptance:** a world builds from every pack of `testdata/chains/` that the
slice runs (`one`, `warm-one`, `second`, `warm-second`, and `two` and `warm-two`,
which are built and audited here and run in Phase 7's interleave test; `three`,
`five`, `famine` and `one-demand` are Stage 2: N-295, plan 1-12 Out of scope) and
audits clean; the account and ID tests pass; the events and logs match; the shell
tests of Phase 2 still pass.

## Phase 4: `tick.go`, the entry and phases 1 and 2

**Why here:** the straight line, its entry and the first positions are the
foundation of every later position; production is the first row's logic that
needs the agents.

Deliverables (each position an unexported method, one line in `tick.go`, D166;
the loops are in `phase_start.go` and `phase_production.go`):

1. **`tick.go`**: the list gains the built positions (`line` entries call their
   method from the straight line; P0.1, P0.2a and P0.3 are `bracket` entries run inside
   `beginStep`, D141, D166) as they are built; `notBuiltPositions`
   loses the entries built in this phase.
2. **P0.1 to P0.3** (P0.1 and P0.3 exist since the Phase 2 shell and are already out of `notBuiltPositions`; this phase builds P0.2a and deletes its entry): the guards, the end of the construction phase, P0.2a (the
   construction bucket's `BoundaryFlow` and `WearApplied` rows with `Tick` -1,
   once, D160), `TickStarted`.
3. **P1.1** the moment (`Calendar.At` or the day table, once, derived) and **P1.2**
   the cost anchors (D157): `productionScratch.anchors` over the previous tick's
   published prices (the seed price at tick 0) and the wages of
   `readBuf.regionWage` with the warm-start `SeedWage` fallback, per region, with
   `runnable [][]RecipeID` built from the counts of agents per (region, role). The
   anchors are computed once per tick, before the first `Work` call, and a job
   request is checked against the role's list first (`not_on_role`) before any
   other check (FR-PRO-048).
3a. **The view builder** (round 3 B1, D165): P2.5 calls `Work` with a view, so the
   builder is built here and not in Phase 5. (i) `refreshViewCtx` in `market.go` assembles
   `viewCtx` from what exists after Phase 3 and the earlier positions of this phase (the
   per-region slices of the market reports and windows from the market records, the
   published prices left by P4.1 of the previous tick or the seed price at tick 0, the cost
   anchors of P1.2, the season of P1.1; the tables the world already holds), called at the
   start of P2.5. (ii) `fanin.go: fillAgentView(v, a)` calls row 7's
   `(*viewScratch).fillCore` (which fills the identity, money, state, upkeep, job,
   per-commodity, per-recipe, tunables and mailbox groups, `MaxExecutions`, the welfare
   reads, `Bounds` and the belief group) and then the module fillers (none in the slice),
   and nothing else. It is the **same builder** at every use: P3.2 (the state after P2.7,
   Phase 5) calls `refreshViewCtx` again and does not share a cached context.
   `checkTradePlan` is the part that stays in Phase 5.
4. **P2.1** continuations and completions in the rotation order (D153):
   `job.advance`, `complete` (the ledger's `Transform` and `Discard`, the credit
   bounded by `MaxQuantity`), `RecipeExecuted` through `recipeExecuted` behind the
   flag, `markWorking` for a job continued or completed in this tick (called once
   per agent per tick: it panics on a second call, so a job started at P2.7 is
   marked at its start only, D158), the "recipe output disposed" `Warn` from
   `Overflow` (D156). **P2.5** `Work` for every free agent in the rotation order, two-pass
   (6.4.1): the view of deliverable 3a, then **the work report (the production halt report and the floor
   count) computed from the view as filled, before the call to `Work`** (plan 1-7 `workReport`, N-246,
   FR-OBS-016: never from a view the strategy has had in hand), then the agent row's `Work`, the world's derivation of `k0` by
   `maxExecutions`, the refusals counted in the `refusalTally`, the floor `Warn` run (`floorRun`) from `readBuf.heldByFloor`.
   **P2.7** the starts in the rotation order: `productionScratch.start`, the wear
   facts to `addWear`, `markWorking` (at the start, including a one-tick job, which
   then completes as in P2.1 without a second mark).
   **P2.8a** `ProductionHalted` and `RecipeRefused` through `refusalTally.flush`.
   Skipped agents (away, on a journey, teaching) are skipped by the predicates the
   module rows will fill; in the slice none is.
5. The rotation lists (D153) and their scratch.
6. **The welfare scratch clear at P8.6** (D158): `endStep`'s P8.6 clears gain the
   `reset(slot)` over every slot of the welfare scratch, built here with the first
   `markWorking` and `addWear` calls, so the multi-tick tests of this phase and of
   Phase 5 do not meet a stale flag. P8.6 is `Ledger.EndTick()` (the advance, then
   `ClearTotals`: `internal/ledger/ageing.go`), then the counters of the rows and
   this clear; use `EndTick` so the order cannot drift (D161).

**Tests:**

- Unit: the rotation: for `n_r` agents and ticks 0..2 `n_r` the start index is
  `t mod n_r`, the walk wraps, the list is built once per phase (a loop over `t`
  against a hand table, FR-WLD-043).
- Integration over `one.toml`: after N ticks the goods-conservation invariant
  with recipes (inputs consumed, outputs created: `Audit()` plus the flow totals
  equal the recipes' arithmetic for a scripted two-agent world built in Go,
  testing.md 6.3 first bullet); `RecipeExecuted` fields equal the arithmetic of a
  worked example (PRO) in a comment (testing.md 5.5); a job of duration 3
  completes in `s + 2` and a one-tick job in its start tick; a pending output
  survives a tick; `Work` is called for every free agent before any start (a
  recording `Strategy` fake wrapping the default asserts the call order against
  the starts, FR-WLD-046).
- Integration (round 3 B1): the view `Work` receives at P2.5 is built from the state after
  P2.1 (a recording `Strategy` fake wrapping the default stores the view: after a
  completion at P2.1 the output stock is in the view; the published price of a market is
  the seed price at tick 0 and the previous tick's P4.1 price after it, the `UnitCost` is
  the anchor of P1.2 for a commodity some local recipe makes and 0 otherwise, and a
  strategy that writes to its view changes no state and no later view: FR-AGT-024). The
  anchor in the view is the one computed at P1.2, passed once before the first `Work`
  (FR-AGT-073).
- Unit (FR-AGT-024, FR-OBS-016; plan 1-7 carry): a consumer `Strategy` fake whose `Work`
  writes to the view it receives (zeroes `Stock`, clears the halted flags) leaves the
  `ProductionHalted` rows and the floor count equal to those of the view as filled.
- Integration: the production halt and refusal counts equal a hand count for a
  scripted no-input world; `RecipeRefused` only for a tick with a refusal.
- Integration over `warm-one.toml` (the halves of Phase 3's warm start that need the
  tick): run to the first completion of a warm job, the `recipe_output` goods pair
  with the `recipe_input` goods of the construction `Transform` (FR-VIA-036); the cost
  anchor of P1.2 reads `SeedWage` while the region has no working household-tick
  (tick 0; the return to the pooled wage and back is Phase 6's, FR-VIA-052).
- Unit: a 5-tick run of a scripted world with a job of duration 3 does not panic at
  `markWorking` in tick 2, and the scratch of every slot is zero after each step
  (the P8.6 clear; D158).
- Unit: `markWorking` is called once per agent per tick on every path (a job
  continued, completed, started, a one-tick job, a job started and completed in
  one tick), checked by a welfare scratch that panics on the second call; the
  working tick of a role with recipes follows FR-WEL-017 (started, continued or
  completed). The wear cost of the executions started is recorded at the start
  (FR-WEL-003, first half; its valuation at P6.3 is Phase 6).
- Integration: the order and the rotation (FR-PRO-016, 017): continuations and
  completions of all agents come before any start, a start walk begins at
  `t mod n_r`, and a request for a recipe not on the role's list is `not_on_role`
  before any other check, the cost anchors computed once before the first `Work`
  (FR-PRO-048).
- Integration: the construction bucket's `BoundaryFlow` rows (`Tick` -1) come
  once, before the first `TickStarted`, and equal `Ledger.AppendFlows(true)`
  (FR-OBS-015).
- Unit (round 3 NB12, FR-WLD-006, 012): a panic inside a position (a consumer `Strategy`
  whose `Work` panics, recovered by the test) leaves the world failed, emits no
  `TickEnded` and clears the `stepping` flag, so the next call returns `ErrWorldFailed`.
- Integration (FR-OBS-018): over a two-region Go-built `Config` (or `two.toml`), the
  `Region` of every `RecipeExecuted` and `RecipeAbandoned` equals the region of the
  agent's ledger account at the event.
- Unit: the position table test with the built positions removed from the
  `notBuilt` list; a mutation that swaps two positions in `tick.go` fails it.

**Observability:** events `RecipeExecuted`, `RecipeRefused`, `ProductionHalted`,
`RecipeAbandoned` (only where a job ends), `BoundaryFlow` Tick -1, `TickStarted`;
`Warn` rows 4 and 19; `Debug` "recipe not started" behind `Enabled`. Each
event is built only under the flag; the allocation of the production phase with
the tally is checked at zero with the default observer (N-242 point 4;
NFR-OBS-001).

**Requirements:** FR-WLD-004, 040, 041, 042, 043, 044, 046, 058 (P1.2 half);
FR-AGT-024, 072 (the floor count and its `Warn` run, tested with the P2.5 floor case of
the Phase 10 scenario row 4 and a unit case on `heldByFloor`), 073 (the view and the anchor passed once); FR-OBS-018 (the region on
`RecipeExecuted` and `RecipeAbandoned` equals the region of the agent's account: asserted
in the events test above); FR-PRO-042 (the `JobRow` aggregates are Phase 9's; the pending
output is booked here);
FR-PRO-014, 015, 016, 017, 026, 029, 030, 041, 048, 052, 053; FR-OBS-014 (the order
of rows), 015, 016, 095, 096 (`Overflow` and the row 19 `Warn`); FR-WEL-003 (first half), 017 (the role-with-recipes half),
033 (floor `Warn`).

**Acceptance:** a world steps through phases 1 and 2 (the later positions are in
the `notBuilt` list); `Work` is called with a view built by `fillAgentView`, so a job
starts, completes and books its `RecipeExecuted`; the production invariant holds; the
position test passes.

## Phase 5: phases 3 and 4, the trade, the clearing and the fan-ins of the plan

**Why here:** orders and clearing are the core of a tick; the view and plan
fan-ins exist before any module.

Deliverables:

1. **`fanin.go`**: `checkTradePlan` (the world's checks of the plan, ASSUMED row 7). The
   view builder `fillAgentView` was built in Phase 4 (round 3 B1, D165) and is reused
   here unchanged, with `refreshViewCtx` called at the start of P3.2 (the state after P2.7).
2. **P3.2** for each agent in ascending ID: build the view with `fillAgentView`, call `Trade`, check
   the plan, execute it (discards, then the orders in plan order, each admitted
   with its reservation by `book.admit` or rejected with a `reason`), the
   `receiptTally` cleared and each book reset (`book.reset(tick)`, which sets the
   tick of every event the book emits) at the start of P3.2 (plan 1-3 D34), the
   world calling `b.emitPlaced(obs, o)` after an admit and `b.emitRejected(obs, log,
   o, reason)` after a rejection (`clearing.go:418-451`: the book builds the events
   and the `Debug` line "order rejected"; the world builds no second copy), the
   facts of each admitted intent (purpose, `Target`, `Reserve`, `UnitCost`) kept in the
   book's `intents` record at the entry's index, and the buyer's entry found through
   `slotOf` for P4.1 (D194, D158), the draws D5 of the default's one word per intent and of a consumer's own
   (D155). `markTraded` is NOT called here: an order that is placed and never
   fills is not a working tick (D158).
3. **P3.4** the estates' asks, per region ascending, one ask per lot held
   (ASSUMED row 8), through the same `book.admit`.
4. **P4.1** for each market in ascending (`RegionID`, `CommodityID`):
   `book.anchor` from `unitCost(region, commodity)` (D157), the reseed of the
   stream of D7 and `clearMarket`, which itself emits `TradeExecuted`, `MarketCleared`
   and the `Debug` line "market cleared" (`clearing.go:574-647`; the world builds
   none of them again), the `receiptTally`, `addReceipt` and `addInputSpend` from the
   settled trades, and **`markTraded` for the buyer and the seller of each settled
   `Trade`** (pseudo-agents skipped: an estate seller has no welfare slot; D158), the
   market record's report and age updated; an invalid list fails the step (D151,
   FR-WLD-048). A market with no admitted order is still cleared for the report
   (`MarketCleared` is emitted also with no trade, 6.6).
5. The input order of a market: agents' orders, the estates' asks, in that
   order (6.5.1; FR-WLD-047).

**Tests:**

- Integration: a scripted two-agent market (one bid, one ask crossing) on a Go
  `Config`: the trade price is the half-even midpoint, balances move by the
  trade, `TradeExecuted` and `MarketCleared` fields equal the arithmetic in a
  comment, and the order of events is `OrderPlaced` x2, `TradeExecuted`,
  `MarketCleared` (FR-CLR-001, 004; FR-WLD-047, 048).
- Integration: no reservation is open after P4.1 (`Ledger.HasReservation()` false
  at every tick boundary of a 50-tick run over a seed range; FR-WLD-045), a
  failing seed is named in the message (testing.md 6.2). An internal test checks it
  also directly after the P4.1 method returns and before P5.1 by calling the
  unexported position methods, because a reservation released at P6 would pass a
  check made only at tick boundaries (FR-WLD-045).
- Integration (round 3 NB10; FR-AGT-057, 079): over a scripted two-agent market, the
  per-tick totals of receipts and of spend by purpose that the welfare scratch holds equal
  the ledger's `trade` flows for the tick (sum of both sides), and a `TradePlan` is executed
  in the order discards, then intents (an event-order assertion; the shipment half and the
  away/journey halves are modules').
- Integration (D194): the buyer's purpose reaches `addReceipt`/`addInputSpend` for every
  trade, including a custom `Clearing` fake that returns two trades for one bid (both find
  the bid's slot), and the slots are clear at the next tick's first admission (a bid with
  a different purpose in tick 2 is not charged with tick 1's). The record of an admitted
  intent keeps its `Target`, `Reserve` and `UnitCost` untouched from P3.2 to the end of
  P4.1 (asserted for a bid and an ask of a household agent; Phase 6 reads them at P5.4).
- Integration, `Guard:` (FR-WEL-017, WEL 4.3): in a role with no recipes (a Go-built
  `Config` with a populated recipe-less role: every role of the packs `one` and `second`
  lists recipes, and `visitor` of `base.toml` is closed with no group, so no agent exists
  there; FR-VIA-005 exempts a role that lists none), an agent whose order never fills has no working tick, and an agent whose
  order fills has one; `markTraded` marks the buyer and the seller of a settled
  trade and no pseudo-agent.
- Integration: `MarketCleared.CostAnchor` equals the anchor in force for the
  market, `unitCost(region, commodity)` (FR-OBS-097); the sale of an estate's ask
  is a `trade` transfer into the estate account (FR-EST-013); a recording
  `Strategy` fake wrapping the default is never passed an estate ID (FR-AGT-032,
  FR-EST-011).
- Failure: a faulty `Clearing` fake (a hand-written type at the extension point)
  returns a list with a self-trade, a price outside the limits and an unknown
  buyer; each time `Step` returns an error with `errors.Is(err, ErrInvalidTrade)`
  naming the tick, region and commodity, `TickEnded` has `Failed` true, the world
  is failed, the markets after it are not cleared, the reservations of the failed
  market are released (FR-WLD-005, 006, 012, 048; N-239).
- Integration: the keys of a market come from its own stream: clearing markets in
  a different order (an internal test calling the markets in reverse) gives the
  same trades (FR-WLD-069 half; C22).
- Unit, `Guard:` (D195, round 3 NB2): `fillAgentView` and `checkTradePlan` are
  compositions, tested without a seam (there is no list to append a fake to and D166
  forbids a hook): a `go/parser` test reads each body and checks that it is only the
  expected calls in the expected order; what it cannot show (a later module step called in
  its slot) is the module PR's own test (architecture.md 2.10.4).

**Observability:** events `OrderPlaced`, `OrderRejected`, `TradeExecuted`,
`MarketCleared`, all built by the clearing row's `emitPlaced`, `emitRejected` and
`clearMarket` (which also write the `Debug` lines "order rejected" and "market
cleared" behind `Enabled`): this row passes them the observers, the logger, the
reset tick and the anchor and writes none of these lines itself, so there is no
double line. No `Warn`. Events only under the flag.

**Requirements:** FR-WLD-042 (the pseudo-agent half: the recording `Strategy` fake never sees an estate; the away, journey and teaching halves are modules'), 045, 047, 048, 062, 069 (the half of keyed markets, tested by the reverse-order case; the rest is Stage 0 and row 12); FR-CLR-001, 004, 023, 031, 033,
034; FR-WEL-017 (the role-without-recipes half); FR-AGT-032, 057, 079 (the discards-then-intents half); FR-EST-011, 013;
FR-OBS-022 (the order of a market's events, and the order half), 097; FR-LED-020 (no reservation at a tick boundary: the no-reservation window test); NFR-WLD-001 (the
call-site half is Phase 10).

**Acceptance:** a world steps through phase 4; the no-reservation window holds;
the failure path is tested.

## Phase 6: phases 5 and 6, the outcomes, consumption and the welfare push

**Why here:** the outcomes close the price loop and the consumption closes the
tick's flows; both read the reports of Phase 5.

Deliverables:

1. **P5.1** the price window push for a market in which units traded (the
   `beliefs.Window` of the market record) and **P5.3** the estate markdown
   (ASSUMED row 8).
2. **P5.4** the outcomes: for each admitted order of a household agent in
   ascending `RegionID`, `CommodityID` and admission order, the world reseeds
   D8's stream, takes one `uint64` (the outcome's `Draw`), builds the `OutcomeView`
   (round 3 NB5: plan 1-7's `fillCore` fills `AgentView` only, and the world fills the
   outcome view itself: the market `Report` and the filled units from `book.filled()`,
   `Stock` by `outcomeStock(held, reserve)`, `Target`, `UnitCost` and the reserve read
   from the entry's `intents` record (D194), the bounds (`beliefs.Limits` with that unit cost), the belief
   group copied from `beliefParams` (N-245) and the market's window), calls the agent
   row's `Revise`, applies
   the returned belief inside its bounds (`beliefs.Clamp`), builds the
   `beliefs.Change` and adds it to the market's `Tally` (FR-WLD-049; FR-BEL-008,
   026). An estate's ask has no outcome and takes no word (C21). After the last
   outcome of a market with one, `BeliefsUpdated` (`beliefsEvent`) is emitted
   under the flag and then the "price beliefs at bound" `Warn` from the run
   advanced by `beliefs.NextRun` for every market every tick (D156, N-230). The
   outcomes are delivered once, after every market of the tick has cleared, in
   order of region, commodity and admission, to the strategy's `Revise`, however many
   trades filled the order; a rejected order and an agent with no order give none
   (FR-CLR-025, FR-BEL-007). The belief update writes no `Info`, and each `Debug`
   line is guarded by `Enabled` (FR-BEL-032).
3. **P6.1** upkeep for each agent and upkeep commodity (ASSUMED row 7: the
   consumption step moves `min(held, U)` to `consumption`, sets `Fed`, `Unfed`,
   `Hunger`, `Privation`; the world calls `household.NextPrivation` of plan 1-7 through the consumption step, and there is no spawner-side privation step: plan 1-9 drops `occupation.PrivationStep`), **P6.3** the welfare record of each agent in ascending
   ID: `record`, `Push`, the clamp run and the `Warn` "welfare record clamped",
   `reset` (D158; the wear cost recorded at the start is valued here at the region's
   market price of the tool in the tick, the effective reference price if it never
   traded: FR-WEL-003; "never traded" is `market.window.Len() == 0`, because the window
   is pushed on every tick with a trade and never shrinks, whereas a seeded market has
   price age 0 and `marketReport` has no such flag; a test seeds a warm-start price
   that differs from the reference price); **P6.4** `UpkeepShortfall` per (region, role) where a household
   went unfed, and the `Warn` "households unfed" (D156).

**Tests:**

- Integration: a scripted market with one bid that fails and one ask that
  trades: the outcomes are delivered in the order of FR-WLD-049, one word per
  admitted order of a household and none for an estate's ask; the words equal
  the stream vectors of `internal/ordering` for the keys (a test computes
  them); `BeliefsUpdated` counts equal the arithmetic (FR-BEL-008, 026).
- Integration: the window is pushed only for a market that traded (C19); the
  published price at tick 0 is the seed price; the price age counts up on a
  no-trade market, and a market that has never traded has an empty window
  (FR-WLD-058; FR-CLR-032; FR-BEL-039).
- `Guard:` (FR-WEL-011): nothing clamps an agent's income to keep its ratio above 1.
  A scripted agent whose ratio stays far below 1 (and one far above) has a record
  income equal to the sum of its receipts and its money moved only by the
  trades and upkeep of the tick; the world tops up nothing. If a future change
  clamped income, this test is the one that would fail.
- Integration: upkeep and the welfare push over a scripted agent: income,
  cost and the working flags equal the arithmetic of a worked example (WEL); a
  household with too little money goes unfed and `UpkeepShortfall` is emitted
  once per (region, role) and not for a tick with none (FR-WLD-050, FR-OBS-009); its
  `Privation` rises by 1 each unfed tick and falls by 1 (to 0) each fed tick (FR-AGT-075).
- Integration over `warm-one.toml` (FR-VIA-052, not latched; needs the welfare
  pushes of P6.3): the cost anchor uses `SeedWage` while the pooled wage is unknown,
  the pooled wage once the rings give one, and `SeedWage` again when it becomes
  unknown (a scripted ring state through the unexported scratch).
- Log test (with the handler of Phase 2, the scenarios built in Phase 10): "households
  unfed", "price beliefs at bound", "welfare record clamped" at the tick their
  pattern says and again at the next multiple of 10.

**Observability:** events `BeliefsUpdated`, `UpkeepShortfall`; `Warn` rows 2, 3,
26; `Debug` "belief updated". The aggregates are bounded by markets or by
(region, role) (rules 3.2.2).

**Requirements:** FR-WLD-049, 050, 058; FR-BEL-007, 008, 026, 032 (the code half; the
log half is Phase 10), 036, 039; FR-CLR-025; FR-WEL-003 (valuation half), 011, 042;
FR-AGT-023, 075 (`Privation` set at P6.1 by `household.NextPrivation`, after `Hunger`; same test as FR-WLD-050); FR-OBS-009, 094; NFR-WLD-001. NFR-BEL-001 (the belief path allocates
nothing) is a row of Phase 10's call-site table.

**Acceptance:** a world steps through phase 6; the price loop moves a belief on a
scripted market; the runs of the three `Warn`s are advanced every tick.

## Phase 7: phase 7, the removal fan-in and the executor of the spawner's plan

**Why here:** entry and exit change the agent set, so they come after everything
that reads it. The executor is the largest piece of logic of the row and the only
one that composes the primitives of five rows by hand (coordinator ruling,
2026-10-17: the executor, `removeAgent` and the role-change and weight-change
composition are this row's; plan 1-7 builds no combined step, plan 1-9 builds the
plan and the pure support), so it has its own phase and its own tests (D192).

Deliverables:

1. **`fanin.go`**: `removeAgent` (D165), in the order of `spec-agents.md` 3.6: the
   production `abandon` with the reason, the agent row's per-agent state, the
   estate row's `receive` at removal steps 3 and 4 (money and goods to the estate
   account of the region the agent's account is in, FR-EST-002; `GoodsListed`
   counts LOTS, coordinator ruling, N-255; `EstateOpened.Balance` reads `pool`),
   then `Ledger.Close` of the agent's account (the world opened it in Phase 3), the
   events in the order of WLD 6.11 item 8 (`RecipeAbandoned`, then `AgentBankrupted`,
   then `EstateOpened`), the counters. Its module steps 2b to 2g are empty in the
   slice. `relocationRefusal` (the empty composition, D165). There is one removal
   path (architecture.md 2.10.4).
2. **The view builder** for P7.1: the `RegionView` of each region (roles in
   canonical order, agents in ascending ID, no pseudo-agent), every field of plan 1-9
   Provides (names matched at A1): `Tick`, `Region`, `TicksPerYear`, `PoolMoney` from
   `estates.pool`, `Budget`, the tunables copied from `WorldParams` (`ReviewTicks`,
   `EntryShare`, `RoleHysteresis`, `RoleCooldown`, `EntryRatio`, `ChangePrivation`,
   `RemovePrivation`, `DeathStep`, `ExitShare`, `Renewal`, `EntryPoolShare`); per
   `RoleView`: `Role`, `Closed`, `Households`, `Agents`, `Wage` and `HasWage` from the
   welfare summary `roleSummary` (the region's unskilled wage is computed first, for
   `readBuf.summarise`), `Upkeep` per household at the published prices,
   `ExpectedWorkTicks`, `StartMoney`, `ToolPrice`, `TrainingTicks`, `CapitalReturn`,
   `RenewalCarry` from `worldState` (item 4); per `RegionAgent`: `Agent`, `Role`,
   `Weight`, `External`, `Mortal`, `Money`, `Unfed`, `Hunger`, `Privation`,
   `RoleSince`, `Upkeep`; and `Decisions.reset()` before each region's `Decide`; with the `Privation` of each agent already set at P6.1 by
   `household.NextPrivation` (plan 1-7: the single privation step; plan 1-9 builds
   none), the one scratch of the world, the `RegionView`'s unexported `observed` flag (true when an observer is supplied, set once; without it `EntryLimited.Hysteresis` is always 0: plan 1-9 D119), and a `Decide` call with the keyed stream
   `(siteSpawner, t, region)` only for a consumer's spawner (the default takes no
   word, so the world skips the reseed: D155).
3. **The executor** (`phase_exit.go`): applies `Decisions` in plan order, checking
   each against live state with `occupation.Refuse` (a refusal of a default
   decision is a bug and fails the Guard of item 4), using row 9's pure support
   (`Remaining`, `Renewal`, `Funded`, `Place`, `Retarget`; `Allowance` and `Exits` are
   the default `Decide`'s and the executor does not call them: round 3 NB6),
   and the estate calls of row 8.
   - **Refusals** (FR-SPN-004, 010, D196; round 3 B2): a decision that fails `Refuse`
     (`unknown_agent`, `unknown_role` and an agent in another region, `external_agent` for
     an external agent that is not mortal, `same_role`, `bad_households`, `unsupported_cause`; the
     module-owned reasons are appended with their modules) is refused: nothing is changed
     (no ledger call, no event but the count), it is added to `Refused` through
     `entryTally` for the region and the `Debug` line "decision refused" (`agent`,
     `reason` from `refusalReason`) is written behind `Enabled`; the executor goes on to
     the next decision of the plan. A refused `Enter` that is only partly funded is **not** a refusal:
     the funded part proceeds and the rest is counted `Unfunded` (FR-SPN-008).
   By kind:
   - **Remove**: `removeAgent` with the cause.
   - **Reduce** (a weight `w` to `w'`, SPN 4.1, AGT 3.7): the money kept is
     `occupation.Remaining(x, w, w') = floor(x * w' / w)` and the difference goes
     to the estate through `acceptMoney` with the ledger reason `starvation` for the
     causes `starved` and `insolvent` (SPN 4.1; `deaths` is POP's cause `population`
     and is not in the ledger registry: Stage 3; any other cause is
     **refused** (O10, ruled by the coordinator: a reduction's ledger reason must name what it is, and a
     module that introduces a new cause appends its cause constant and its ledger reason together in its
     own change, architecture.md 2.10.5, N-290; no reason is guessed now): the executor sets
     `Facts.CauseSupported` (plan 1-9) from the cause before `occupation.Refuse`, so the decision is refused before any ledger call through the refusal path of D196 (counted into
     `EntryLimited.Refused`, `Debug` line "decision refused", reason `ReasonUnsupportedCause` = row 9's exported `unsupported_cause`, from `refusalReason`)); each commodity likewise through `acceptUnits` with the same reason;
     then `job.reweigh`, in the steps and order of AGT 3.7; `WeightChanged` with
     `Money` (signed, positive to the agent) and `Units` (FR-OBS-019). **A reduction
     by the whole weight (`w' = 0`) is a removal** (SPN 4.1; FR-SPN-015): the executor
     takes the `Remove` path with `removeAgent`, so the events are `AgentBankrupted`
     (with the decision's cause as `Decide` gave it, plan 1-9 Phase 5 deliverable 1 and `Exits` (`Remove`, `Insolvent`): the executor never
     recomputes the cause; round 4 NB-6) and not `WeightChanged`. A `DecideReduce` with an
     unsupported cause (for example `consumer`) and `w' = 0` is refused (`unsupported_cause`)
     **before** the Remove path is reached, while a `DecideRemove` with that cause is valid; the
     executor sets `CauseSupported` true for every other decision kind. Dependency: the SPN 4 text of
     `unsupported_cause` is written by plan 1-9 Phase 1 deliverable 8 (`spec-spawner.md` does not have it
     yet); Phase 7 does not start before that PR.
   - **Switch** (a role change, SPN 4.2, AGT 3.8): the account and the money are
     kept; the goods outside the new role's tradable set go to the estate through
     `acceptGoods` (`estate_goods`, whole lots, counted in lots); the job is
     abandoned with `role_changed` (`job.abandon`), which emits `RecipeAbandoned`
     (cause `role_changed`) before `RoleChanged` (WLD 6.11 item 8; a weight reduction
     emits no abandon); the history is resized
     (`history.Resize`); the residues are reset (`setResidues`); the target is
     carried by `occupation.Retarget` (no draw); the beliefs for the commodities
     both roles trade are kept and the others reseeded (`reseedBeliefs`); the
     strategy table index is switched; `RoleSince` is the current tick;
     `RoleChanged` with `GoodsListed` (FR-SPN-006, 033).
   - **Enter** (SPN 4.3): `occupation.Place` picks the role's agent of smallest
     weight below the group's `Scale`, else a new agent of weight 1; each household
     brings `Role.StartMoney` from the pool through `grant` with the reason `births`
     for households added to an existing agent and `start_money` for a new agent (SPN
     4.3; both run estate to agent in `internal/ledger/registry.go`; the tests check
     the reason of each); the budget is taken once
     per region per tick after the deaths and before the switches:
     `PoolMoney * EntryPoolShare / 10 000` (FR-SPN-026, ruling C15), and the executor
     counts the unfunded households; a new agent gets its ledger account
     (`Ledger.Open`), `newAgent`, the two creation draws of D10 (`siteCreation`, the
     tick, the new agent, through `drawCreation`), `Hunger` and `Privation` 0, beliefs
     from the price window mean or the market price and not the warm-start table,
     `RoleSince` the current tick (FR-SPN-027), and `AgentSpawned` with
     `StartMoney` (the event has no `StoresLocked` yet: the storage module appends it, `event.go`; FR-AGT-007; FR-SPN-007).
     A new agent of weight `w` receives `min(w * Role.StartMoney, pool balance)` from the
     pool and no goods (FR-EST-014; at the weight 1 of an entrant, the funded number of
     households of SPN 4.3). An entry into an
     existing agent opens no account (FR-SPN-044), raises its weight by the funded
     households and emits `WeightChanged` (`From`, `To`, the cause of the decision,
     `Money` positive to the agent, `Units` 0; SPN 13.1 lists "an entry into an existing
     agent" as a `WeightChanged` case), after the transfer. The pool is never overdrawn
     (FR-SPN-008, FR-EST-022): the funded number is `Funded(n, budget, StartMoney)`
     = `min(n, floor(budget / StartMoney))`, the unfunded households are neither created
     nor deferred and are counted in `Unfunded`; the budget is also capped by the pool's
     balance at each grant.
   - The counts of the plan and of the executor merge into one `EntryLimited` per
     region per tick (`entryTally.limited`), with the `Debug` lines and the three
     bounded `Warn`s "role extinct", "group below floor" and "entry unfunded"
     (D156). `Decide` is never called for an outside node (FR-SPN-002). The
     counters of agents and households are updated in the same place.
4. **The renewal carry and the runs** (D145, N-265): `worldState` holds, per
   (region, role), the renewal carry that `RoleView.RenewalCarry` reads and that the
   executor advances by calling `occupation.Renewal` with the households it actually
   removed, the per (region, role) "had at least one agent" bit (`Extinct(agents,
   everHad, configured)`, FR-SPN-042) and the three runs of `extinctRun`,
   `unfundedRun`, `belowFloorRun`. The Guard test of N-265 is below.
5. A text-search guard for the one removal path (a second call site of the ledger's
   close, or a second composition of the removal events, fails it).

**Tests:**

- **Executor, table-driven, integration** over a consumer `Spawner` fake (D142; a
  hand-written type in a `_test.go` file) that scripts one decision of each kind
  (`DecideReduce`, `DecideSwitch`, `DecideEnter`, `DecideRemove`) on a hand-built Go
  `Config` of two roles and one region, one case per kind and cause. Each case
  asserts, with the arithmetic in a comment (testing.md 5.5): the money and goods
  of the agent, the estate and the pool by exact equality; the ledger reason of each
  money move (`starvation` for the causes `starved` and `insolvent`; `births` for an
  entry into an existing agent, `start_money` for a new agent; a reduction of any other cause
  is a refusal: O10); the events and their
  order; `GoodsListed` in lots; that no ledger account is opened or closed by a
  reduce, a switch or an enter into an existing agent (the account count before and
  after, FR-SPN-044); the abandon reason of the job (`role_changed` for a switch,
  `agent_removed` for a removal); the length of the history; the residues reset;
  the belief reseed; the strategy table index; `RoleSince`; and conservation by
  `Audit()` and by the flow totals (FR-SPN-045). A worked reduction: an agent of
  weight 4 with 10 money and 6 units of one good, reduced to 3, keeps `floor(10 *
  3 / 4)` = 7 money and `floor(6 * 3 / 4)` = 4 units, so 3 money and 2 units go to
  the estate, and `WeightChanged` has `From` 4, `To` 3, `Money` -3, `Units` 2
  (the signs and the cause constants are re-derived from SPN 4.1 by the
  implementer). Cases: reduce with a job in progress (the job's executions
  re-weighed); switch with goods outside the new set and a job to abandon;
  enter into an existing agent below `Scale` and into a new agent; enter with the
  pool short of the budget (unfunded households counted); remove.
- **Refusals** (FR-SPN-004, 010; round 3 B2), table-driven over the same consumer
  `Spawner` fake: one case per reason the slice builds, each scripting one bad decision
  between two good ones: an unknown agent, an agent of another region (`unknown_agent`),
  an unknown role (`unknown_role`), a switch to the agent's own role (`same_role`), a
  `Reduce` by 0 and by more than the weight, an `Enter` of 0 households
  (`bad_households`), and a decision on an external agent that is not mortal
  (`external_agent`; a mortal external agent is accepted), and a `Reduce` whose cause is
  neither `starved` nor `insolvent` (e.g. `consumer`; O10: refused, `unsupported_cause`, no ledger call). Each case asserts that the
  refused decision changes nothing (ledger balances, account count, agent table, the
  `Audit` and the flow totals equal before and after the decision), that `EntryLimited.Refused`
  counts it (and the other counts do not move), that the `Debug` line "decision refused"
  has the `agent` and the `reason` of `refusalReason` (capturing handler), that the next
  decisions of the plan are executed, and that a tick whose only action was a refusal
  still emits `EntryLimited` (FR-SPN-035) while a tick with nothing emits none.
- **Pool overdraw** (FR-SPN-008, FR-EST-014, FR-EST-022): an `Enter` of `n` households
  with the pool short (a pool of 2.5 `StartMoney`: the funded number is 2, the third is
  `Unfunded`, the pool ends at the 0.5 remainder, never negative), the same with the pool
  at 0, and two `Enter`s in one plan sharing one budget (the second is limited by what
  the first left); the unfunded households are not created and not deferred (the next
  tick's plan does not see them).
- **Reduction to the whole weight**: a `Reduce` of an agent of weight 3 by 3 is a removal
  (the events are `RecipeAbandoned` if a job, `AgentBankrupted`, `EstateOpened`, no
  `WeightChanged`; the account closed; the cause is the one the scripted decision carries,
  asserted once with `starved` and once with `insolvent` (the world half of FR-SPN-015 is only that a
  reduction to 0 emits `AgentBankrupted` and no `WeightChanged`)). **An entry into an existing agent** emits
  `WeightChanged` with `To - From` equal to the funded households, `Money` positive and
  the ledger reason `births` (the counts test below rebuilds the households from it).
- **The budget order**: the entry budget is taken after the deaths (a scripted
  pool: the births of the same tick are funded by the estates of the dead,
  FR-SPN-026, FR-WLD-056) and the budget is taken once.
- **The Guard of N-265** (internal test, `Guard:`): over a seed range with the
  default spawner and a configuration with a positive `Renewal`, the renewal
  entries `Decide` plans for each (region, role, tick) equal the entries
  `occupation.Renewal` gives from the households the executor actually removed and
  the carry before the tick; a refusal of a default decision fails it.
- **Counts**: the agent count rebuilt from `AgentSpawned` and `AgentBankrupted`,
  and the household count also from `WeightChanged` (`To` less `From`), equal the
  world's counters at every tick of the scenarios above (FR-SPN-034; FR-OBS-040
  is Phase 9).
- **The view**: a table test of the built `RegionView` of a scripted region (two
  roles, three agents, one pool balance, a known and an unknown wage) compares every
  field of the list in item 2 with its source by exact equality, with the arithmetic
  in a comment, and a consumer `Spawner` fake checks that `Decisions` is empty at
  each region's `Decide` (plan 1-9 Provides; NB-5 of the round 2 audit).
- **Removal**: a starving scripted agent is removed in phase 7: the events come in
  the order of WLD 6.11 item 8 (`RecipeAbandoned` for its job, then
  `AgentBankrupted`, `EstateOpened`), its money and goods are in the estate of the
  region its account was in (a two-region case, FR-EST-002), the estate emits
  `EstateOpened` with the pool policy on or off (FR-EST-021), the world's agent
  counter falls by one (the snapshot's count is asserted in Phase 9), and a replacement is created with D10's two
  draws; IDs are never reused (FR-AGT-001, second half; FR-WLD-051, 056, 059;
  FR-OBS-020).
- **Regions interleave**: in a two-region world (`two.toml`, or a Go-built
  two-region `Config` if `two` or `warm-two` does not validate in Stage 1), the
  entrants of region `r` are created before region `r + 1`'s `Decide`: a consumer `Spawner` fake records,
  at each region's `Decide`, the agent count and the `AgentSpawned` events already emitted (the keyed
  streams move no word with the order of creation, spec-world.md 6.9.1, so the order is observed
  from the fake and not from the words; round 4 NB-12).
- `removeAgent` and `relocationRefusal` are compositions (architecture.md 2.10.4), tested
  without a seam (D195, round 3 NB2): the `go/parser` shape test of Phase 5 covers the
  bodies; the order of `removeAgent`'s effects is observed from outside, by a removal of
  an agent with a job, goods and money whose `RecipeAbandoned`, `AgentBankrupted`,
  `EstateOpened` come in the order of WLD 6.11 item 8 and whose ledger moves come in the
  order of `spec-agents.md` 3.6 (goods and money to the estate before the account is
  closed); one removal path (item 5).
- The seeds of the removal tests are named in failure messages (testing.md 6.2).

**Observability:** events `RecipeAbandoned` (a switch or a removal with a job),
`AgentBankrupted`, `EstateOpened`, `WeightChanged`, `RoleChanged`, `EntryLimited`,
`AgentSpawned`; `Warn` rows 6, 7, 8; `Debug` "agent
bankrupt", "role changed", "decision refused", "entry dropped" (SPN 13.3). A removal's events are per agent, a state transition
(rules 4.3). Events only under the flag.

**Requirements:** FR-WLD-051, 053, 056, 059; FR-AGT-001 (second half), 007, 008,
077; FR-SPN-002, 004, 005, 006, 007, 008, 010, 015 (the world half: a reduction to the
whole weight is a removal, with the cause), 026, 027, 032, 033, 034, 035 (the omission at
zero and the emission on a refusal), 042, 044, 045, 046 (the snapshot half is Phase 9);
FR-EST-002, 014, 021, 022; FR-OBS-014, 019, 020; FR-LED-031, 035, 036.

**Acceptance:** the executor table passes for every kind; the Guard of N-265 passes;
the world steps through phase 7 with the default spawner; the position test passes.

## Phase 8: phase 8, the end of the tick, the audit and the advance

**Why here:** the end of the tick closes the books, after everything that moves
goods and money.

Deliverables:

1. **P8.1** ageing and spoilage (`Ledger.Age`), **P8.2** `BoundaryFlow` rows for
   the reasons of kind faucet or sink and `WearApplied` rows (`boundaryFlows`,
   `wearApplied`, D160), **P8.4** the audit as the method `p84Audit() error`, which
   runs `Ledger.Audit` under `WithAudit` (D149) and maps a failure by the pure
   function `auditFailure(tick, err)` (D190, D150), **P8.5** `TickEnded`
   (`endStep(err)` with the audit's result; P8.5 and P8.6 are out of `notBuiltPositions` since Phase 2), **P8.6** `Ledger.EndTick()` (`AdvanceTick`,
   `ClearTotals`) and the clears of the per-tick counters and of the welfare scratch
   (the last built in Phase 4), only on success (D161).
2. The `Debug` "tick ended" line (D163).

**Tests:**

- Integration: `Audit()` passes every tick over a seed range with `WithAudit`; the
  flow totals are cleared only at P8.6: a call made between two ticks (an `Audit`, a
  read) changes none; a faucet move made between ticks by an internal test with the
  world's own ledger (`w.ledger.Transfer(...)`; there is no hook, D166) falls in the
  next tick's `BoundaryFlow` (FR-WLD-074).
- Failure, **the audit mapping** (D190). The real `Ledger.Audit` cannot be made to
  fail from the root: `ClearTotals` re-anchors the boundary bases to the balances
  (`internal/ledger/ledger.go:371`), so the identity stays true, and every ledger
  operation conserves. So the root tests the mapping and the bracketing, and says
  so: (a) `auditFailure(t, syn)` over a synthetic `&ledger.AuditError{Identity:
  ledger.IdentityConservation, Account: ledger.None, Commodity: ledger.None, Claim:
  -1, Detail: "synthetic"}` wraps `ErrAuditFailed` and the `*ledger.AuditError`
  (`errors.Is`/`errors.As`) and names tick `t`; nil maps to nil; (b) an internal test
  calls `beginStep()`, then `endStep(auditFailure(t, syn))` and asserts: the
  `TickEnded` event has `Failed` true, the world is already marked failed when the
  event is delivered (an observer that reads the unexported field), `Tick()` is
  unchanged (the tick is not advanced), the flow totals are not cleared
  (`AppendFlows` equal before and after), `Step`'s caller sees the error, and the
  next call returns `ErrWorldFailed` wrapping it (FR-WLD-005, 012, 052). What this
  row does not test, and why: that `p84Audit` calls `Ledger.Audit` and is gated by
  `WithAudit`, which is one line and cannot be shown to fail without a seam in
  production code (D166); the failing branch of `Ledger.Audit` is covered by
  `internal/ledger/audit_internal_test.go`.
- WearApplied: the rows are emitted after the `BoundaryFlow` rows, in ascending
  commodity order, never for a tick with no wear, and the construction bucket's
  rows with `Tick` -1 (FR-OBS-092).

**Observability:** events `BoundaryFlow`, `WearApplied`, `TickEnded`; `Debug` "tick
ended". The error of a failed audit is returned, not logged (architecture.md 8.3).

**Requirements:** FR-WLD-005, 012, 052, 074; FR-OBS-014, 092; FR-LED-031, 035, 036
(the audit identities are the ledger row's; this phase tests the world's use; the comparison of a run with and without `WithAudit` is row 12's matrix).

**Acceptance:** the world steps all eight phases; `notBuiltPositions` lists only
module positions; the position test passes.

## Phase 9: `snapshot.go`

**Why here:** the snapshot reads the whole state; its tests (OBS 9.8) need the
tick to run.

Deliverables:

1. **`snapshot.go`**: `Snapshot` and the row types of D146 with the fields of OBS
   5.3.1 to 5.3.5 for the slice, all plain data, no map, no pointer, no embedded
   type, ordered by the canonical keys (5.1.4); a field at its off value is zero;
   appended fields at the end:
   - the header: `Tick`, `Seed` (the effective seed), `ConfigHash`, `Moment`,
     `Names`, `Modules` (empty), `Agents`, `Households`, `Money`;
   - `RegionRow`: `Region`, `Outside`, `Agents`, `Households`, `MoneyHouseholds`,
     `MoneyPool`, `EntryBudget` (and no `MoneyContract` or `MoneyTreasury`: D146);
   - `RoleRow` (**compile clash:** `export_test.go` line 76 already declares a test
     helper `func RoleRow(t *RoleTable, id RoleID) RoleRowView`, used by
     `config_only_test.go`, `production_pack_test.go` and `role_pack_test.go`; this PR
     renames that helper, for example to `RoleTableRow`, and every caller in those three
     files (and any that rows 7 to 9 add); plan 1-12 audit rounds 2 and 3, N-M, NB-5): `Region`, `Role`, `Agents`, `Households`, `Unfed`, `UnfedAgents`,
     `MaxHunger`, `MaxPrivation`, `HeldByFloor`, `MeanRatioBP`, `HasMeanRatio`,
     `RatioHouseholds`, `WageObserved`, `HasWage`, `PremiumBP`, `HasPremium`,
     `Window`, `ExpectedRatioBP`, `Viable`, `HasExpected` (from row 9's
     `roleSignals`), `ClampedHouseholds`. Row 12 reads `Window`, `MeanRatioBP`,
     `HasMeanRatio`, `RatioHouseholds`, `WageObserved`, `HasWage`, `UnfedAgents`,
     `MaxHunger`, `MaxPrivation`; the counterpart test of FR-OBS-051 needs `Unfed`,
     `HeldByFloor`, `ClampedHouseholds`, `Agents`, `Households`, `EntryBudget`.
     `roleSummary` (`welfare.go:390`) already supplies the welfare ones;
   - `Boundary` rows are balances, read with `Ledger.Boundary(r, commodity)` and
     `Ledger.Account(id)` (not `AppendFlows`); `Pending` is `AppendFlows(false)`;
     `Construction` is `AppendFlows(true)` and only while `InConstruction()`;
   - `MarketRow` with the exported `MarketState` (D147) and its fill (`Price`,
     `Age`, `Volume`, `UnmatchedBid`, `UnmatchedAsk`, `BestBid`, `BestAsk`,
     `HasBid`, `HasAsk`, `WindowLen`, `WindowMean`, `WindowLow`, `WindowHigh`),
     `HeldByAgents`, `HeldByEstate`, `Lots`, `Pending`, and the belief aggregates;
   - `JobRow`.
2. **`World.Snapshot() (Snapshot, error)`**: `ErrInStep`, `ErrWorldFailed`, no
   draw, no state change, deep copy; the scans are the classes of OBS 5.5 using
   `readBuf.summarise(set, p, upkeep, prices, unskilled, hasUnskilled)` (the region's
   unskilled wage is computed first, `welfare.go:406`), `Ledger.Holding`, `AppendLots`, `addPending` and the
   beliefs' aggregates. The pool balance reported per region equals the estate
   account's money (FR-EST-013).
3. `glossary_test.go` loses `Snapshot`.

**Tests** (OBS 9.8; FR-OBS-030 to 033, 040 to 042, 045 half; the restore and
region-state tests wait for their owners):

- Unit: deep copy: mutating a snapshot changes nothing in the world or the next
  snapshot; a snapshot taken, then the world stepped, is unchanged; marshalling on
  another goroutine under `-race` has no race (FR-OBS-030, FR-WLD-010).
- Unit: `Snapshot` returns `ErrInStep` from an observer and `ErrWorldFailed` on a
  failed world (FR-OBS-031).
- Integration: agents and households are different fields; estates are in
  neither; a cohort of weight 40 counts 1 and 40; the sum over `Roles` equals the
  header (FR-OBS-032).
- Integration: the number of rows is equal at scale 1 and at scale 40 for one
  configuration (FR-OBS-033).
- Integration, over a seed range and the packs: the agents and households by role
  and region rebuilt from `AgentSpawned`, `AgentBankrupted`, `WeightChanged` and
  `RoleChanged` equal the snapshot at every boundary (FR-OBS-040); the money of
  the households and the pool rebuilt from the events plus `Construction` plus
  `Pending` equals `MoneyHouseholds` and `MoneyPool` (FR-OBS-041; in Stage 1
  escrow is 0, so the identity reads those two only, D146; money enters or leaves
  only by a named faucet or sink, FR-EST-016); the boundary balance of each reason
  equals `Construction` plus the sum of the `BoundaryFlow` rows plus `Pending`
  (FR-OBS-042); the `MarketRow.State` equals the last `MarketCleared`
  (FR-OBS-045 half); `RoleRow.HeldByFloor` is evaluated at the boundary over the
  agents with no job in progress (FR-WEL-020).
- Allocation: the allocation count of `Snapshot` at scale 1 and at scale 40 is
  equal (NFR-OBS-004 core half).
- Integration (round 3 NB10): the `JobRow` pending output by region and commodity and the
  number of agents with a job by recipe equal a hand count of a scripted world with jobs
  in progress (FR-PRO-042); `RoleRow.ExpectedRatioBP`, `Viable`, `HasExpected` and the
  largest privation (`MaxPrivation`) equal row 9's `roleSignals` for a scripted role, and
  the counts of `EntryLimited` are the event and not a snapshot field (FR-SPN-046).

**Observability:** none beyond the above, because the snapshot is a read that
draws nothing; its fields are the only record of the standing state (rules 2).

**Requirements:** FR-OBS-030, 031, 032, 033, 093 (`Markets[].Lots`), 040, 041, 042, 043 (its derivability
half only: each field is built from saved state and its comment names the source;
asserted by the construction of each field from saved state; the restore equality
is Stage 2), 045; FR-WLD-010; FR-WEL-020; FR-PRO-042; FR-SPN-046 (the snapshot half);
FR-EST-013, 016; NFR-OBS-004.

**Acceptance:** the snapshot tests pass; every field is documented with its
source; the snapshot of a failed world is refused.

## Phase 10: the observability tests that need a world

**Why here:** with all positions built the logs and events are testable end to
end; the split is the coordinator's (D164).

Deliverables (test files only, plus any unexported seam they need that is not a
hook in production code, D166):

1. **`observe_helpers_test.go`**: built in Phase 2 (deliverable 7, round 3 NB8); this
   phase extends it with what the scenarios need and does not recreate it. The recording
   `Observer` (appends each event by value) and the capturing `slog.Handler` (level,
   message, attributes, `Enabled` true for every level) are shared with row 12, which
   defines neither (plan 1-12 D176): the types are `recorder` and `captureHandler`. Row 12
   builds its renderer on `recorder`. The internal allocation test (item 5) needs neither.
2. **The scenario configs** (the packs `famine` and `five` are Stage 2: N-295):
   `one` run for a few ticks; `one` with a `Config` edited in Go so that households
   go unfed and starve (a zero pool, a tiny `StartMoney`, no stock; the
   implementer checks it still validates); a hand-built Go `Config` of two roles
   with a consumer `Spawner` fake scripting a reduction (cause `starved`; a reduction of another cause is the refusal case, O10), a switch, an entry and a
   removal; a consumer `Strategy` fake that places an invalid order (an
   `OrderRejected`); a consumer `Spawner` fake that removes every agent of a role
   with no replacement (the "role extinct" `Warn`); a
   faulty `Clearing` fake (a failed step); a scripted removal of an agent with a
   job in progress (`RecipeAbandoned`); and one scenario for each `Warn` row
   not produced by the above, with Go-built `Config`s (the implementer checks each
   validates): row 3 "price beliefs at bound" (a market whose beliefs are pinned at
   the bound by a narrow configured band), row 4 "work floor binding" (more than half
   of a role's households held by the floor), row 8 "group below floor" (a group of
   weight above 1 whose consumer-spawner removals leave fewer than `MinAgents`
   agents), row 19 "recipe output disposed" (a recipe whose output at the group's
   weight passes `MaxQuantity`), and row 26 "welfare record clamped", which a run
   cannot reach, by an internal test that sets the ring state through the unexported
   agent table and calls the P6.3 method.
3. The tests of OBS 9.1 to 9.3 and 9.7 that are this row's:
   - catalogue: every event declared in the root package is a row of
     `spec-observability.md` 4.1 or named by `notBuiltEvents`; the world emits each
     declared event of the slice at least once over the scenarios of item 2, and the
     staged lists fail on a stale entry (FR-OBS-003). The events and the scenario
     that emits each: `WorldCreated`, `AgentSpawned`, `RunStarted`, `RunEnded`,
     `TickStarted`, `TickEnded`, `OrderPlaced`, `TradeExecuted`, `MarketCleared`,
     `BeliefsUpdated`, `RecipeExecuted`, `BoundaryFlow`, `WearApplied` (the first
     scenario); `UpkeepShortfall` and `AgentBankrupted`/`EstateOpened` (starving);
     `RecipeRefused`, `ProductionHalted` (a scripted no-input world, Phase 4);
     `RecipeAbandoned` (removal with a job); `WeightChanged`, `RoleChanged`,
     `EntryLimited` (the consumer spawner); `OrderRejected` (the consumer strategy);
     `TickEnded` with `Failed` (the faulty clearing);
   - emission order of a scripted step against WLD 6.11 (a recording observer,
     the sequence of types; FR-OBS-022, FR-WLD-053);
   - run lifecycle (cancelled, failed, negative, in-step, failed world) and tick
     bracketing (a step that fails at P4.1 with the faulty clearing, or through
     `endStep` with a mapped audit error from an internal test, has `Failed` true; a
     guard at P0.1 emits neither; a panic emits no `TickEnded`) (FR-OBS-012, 013);
   - composition: several `WithObserver` options each receive every event in
     order; a type-switching observer sees exactly its subset (FR-OBS-024);
   - defaults: a world with no logger and no observer builds, steps and runs,
     writes nothing to stdout or stderr (the test replaces `os.Stdout` and
     `os.Stderr` with pipes), and a nil logger or observer behaves the same
     (FR-OBS-090);
   - pseudo-agents: no `AgentSpawned`, `AgentBankrupted`, `WeightChanged` or `RoleChanged` has an estate as its subject and no count includes one (FR-OBS-007); a recording `Strategy` fake wrapping the default asserts that `Work`, `Trade` and `Revise` are never called for an estate ID and that no estate ID is a subject of an agent event (FR-EST-011, FR-AGT-032); the events of an estate's orders carry its agent ID (FR-EST-012);
   - module off and silent aggregates: an aggregate with "never when all are
     zero" is absent in a quiet tick (FR-OBS-008, 009);
   - non-interference of beliefs: a run with a logger and an observer gives the
     same snapshot and, in an internal test, the same belief of every agent as a
     run without (FR-BEL-033; the pack-level proof is row 12's).
4. The log tests of OBS 9.2: each `Warn` row of the slice (2, 3, 4, 6, 7, 8, 19,
   26 and the configuration warning, 1) is produced by a scenario of item 2 at the tick its
   pattern says and again at the next multiple of 10 (FR-OBS-071); no cell emits
   more than `ceil(N / warnRun) + 1` lines in a run of N ticks (FR-OBS-072); the
   counterpart of every row exists and carries the number the line reports, read
   by a plain text parse of OBS 7.3 (FR-OBS-051); no record at `Info` or above
   between `TickStarted` and `TickEnded` other than a row of 7.3 (FR-OBS-070);
   no `Error` record (FR-OBS-074); every key is a key of the rules' table 3.3.2
   (FR-OBS-073); constant messages (FR-OBS-075); "world created" has the keys of
   OBS 7.2 (FR-OBS-076); the belief update writes no `Info` and every `Debug` line
   is guarded by `Enabled` (FR-BEL-032). Each `Warn` scenario also runs in a logger-only
   variant (a capturing handler and no observer) and must produce the same `Warn` lines
   (round 4 NB-8). The bounded `Warn`s of rows 6 to 8 are the cover of FR-SPN-036 and 047.
5. The call-site allocation test (rules 8.2 (a)): with the default logger and
   observer, `testing.AllocsPerRun` over each event-emission and logging call
   site the tick loop reaches, including the executor's (phase 7: NFR-SPN-001) and those of the belief path (the update, the
   draw, the window push: NFR-BEL-001) and the production phase with the tally
   (a table of the sites, each exercised through the unexported position method in
   an internal test, reached by the scenarios above) reports 0 (NFR-OBS-001
   call-site half; NFR-WLD-001 is row 12's full-`Step` baseline). It uses the
   default observer and logger, so it needs no recorder and no helper file of the
   external package.
6. Benchmarks with `b.ReportAllocs()` and `for b.Loop()` of a `Step` of `one`
   with no observer and with the recorder, and one per phase of the tick (the eight
   phases, through the unexported position methods of an internal benchmark file, so
   NFR-WLD-002 can be read per phase; no threshold is asserted: N-015) (testing.md 8.1;
   NFR-WLD-002, a SHOULD).
7. A runnable `Example` of `NewWorld`, `Step`, `Run` and `Snapshot` in an external
   test file, with an `// Output:` comment, and the `WithSpawner` example that
   plan 1-9 hands to this row and the `Clearing` example plan 1-3 hands to it
   (architecture.md 6.5, 1.3).

**Tests:** the above are the deliverables; each carries a `Requirement:` line
(testing.md 4.2), a worked comment for every expected value (5.5), and names the
seed of a failing run (6.2).

**Observability:** this phase is the test of the observability of Phases 2 to 9;
it adds no log or event.

**Requirements:** FR-OBS-003, 007, 008, 009, 012, 013, 022, 024, 051, 070 to 076,
090; FR-WLD-003, 053, 054, 064 (the logger/observer half is row 12's, this phase
tests that a logger or an observer changes no event stream between two observed
runs); FR-BEL-032, 033; FR-EST-011, 012; FR-AGT-032; NFR-OBS-001; NFR-WLD-002 (the
benchmarks); NFR-BEL-001; FR-SPN-036, 047 (the bounded `Warn`s, item 4); NFR-SPN-001 (the executor's call sites, item 5);
NFR-SPN-003 (non-interference of decisions: row 12's pack matrix).

**Acceptance:** all the tests above pass under `-race`; mutating a copy of the
production code (flip an `Enabled` guard, move an event before its transition,
drop a `Warn` run advance) fails at least one of them (testing.md 9.3).

## Phase 11: the exit gate and the traceability

**Why last:** it proves the row as a whole and hands row 12 a stable surface.

Deliverables:

1. The traceability table of this plan's Testing section filled with the test
   names (testing.md 4.5), and the requirement IDs with no test listed as owned
   by another row (N-236 style).
2. Coverage by scope (`go test -race -covermode=atomic -coverpkg=./...`): root
   90% or more, every package under `internal/` 90% or more, `config/` 90% or
   more; a function excluded from coverage carries a comment (testing.md 9.5).
3. The checks of AGENTS.md: `gofmt -l .` clean, `go vet`, `staticcheck`,
   `go test -short -race ./...`, `go mod tidy` no diff, `lizard -l go -C 10 -a 8
   -L 100 -w` on the new files (N-243); the pinned all-off hash
   (`TestHashConfig_ReferencePinned`) unmoved (N-217).
4. The guard tests of Stage 0 updated, not loosened (N-224): `guardedFiles`
   needs no edit (it walks the directories: round 4 NB-3); `glossary_test.go` terms; `import_guard_test.go` passes;
   the panic guard (`invariant` messages); the determinism guard (no `time.Now`,
   no `init`, no package-level mutable state, and no `range` over a map in the
   files of this row: FR-WLD-041).
5. A one-page hand-over for row 12 in this plan's "As built" section at the end of
   the row: the names of the unexported methods of D166, the packs that run, the
   events the world emits, and the test helpers.
6. The roadmap row's status and the index row set to done with the commit.

**Tests:** the whole suite; a mutation pass on a copy of `world.go` and `tick.go`
(swap two positions; skip the `Failed` marking; clear the totals at the entry of a
step) fails a test for each (testing.md 9.3).

**Observability:** none.

**Requirements:** all of the above.

**Acceptance:** the exit commands pass; coverage is recorded; the notes and
roadmap are updated.

## Cross-plan contract

### Consumes

| Symbol, type, sentinel, spec edit or field | Provided by | Status |
|---|---|---|
| `Config` and its sections, `defaults.go`, `bounds.go`, `validate.go`, `hash.go` (`HashConfig`), `Config.canonical`, the warnings of `Validate` | Stage 0 | Built (023cf97) |
| `Option`, `options`, `registry[T]`, `resolveClearing`, `WithClearing`, `WithSeed`, `WithLogger`, `WithObserver`, `WithAudit`, `WithTargetSpread` | Stage 0, row 3 (`option.go`) | Built |
| `observers` (`supplied`, `deliver`), `warnRun`, `discardObserver` | Stage 0 (`observer.go`) | Built |
| `stream`, `newStream`, `keyed`, `keyedPair`, `word`, `draw`, `drawRange`, the site constants, `reseed` | Stage 0 (`random.go`) | Built |
| The events and reason constants of the slice | Stage 0, rows 3 to 6 (`event.go`) | Built |
| `ErrInvalidID`, `ErrDuplicateID`, `ErrOutOfBounds`, `ErrNilExtension`, `ErrUnknownClearing`, `ErrInvalidTrade`, `ErrOverflow` | Stage 0, row 3 | Built |
| `internal/ledger`: `Ledger` accounts and operations (`Open` and `Close` of the agent accounts, which the world calls; `MoveStated` for the stated lot keys of the stagger; `Boundary` and `Account` for the snapshot's boundary rows), `Audit` and `*AuditError`, `HasReservation`, `Age`, `EndTick`, `AdvanceTick`, `ClearTotals`, `AppendFlows`, `EndConstruction`, `InConstruction`, `Holding`, `Discard`, `Transform`, `WearItems`, `AppendLots`; `ledger.go` (`newLedger`, `boundaryFlows`, `wearApplied`) | row 1, row 5 | Built (read 2026-10-17) |
| `recipeTable`, `roleTable`, `dayTable`, `Calendar` | row 2 | Built |
| `Order`, `Trade`, `Clearing`, `AverageClearing`, `marketReport`, `seedReport`, `book`, `newBook`, `clearScratch`, `receiptTally`, `clearMarket`, `tradeListError` | row 3 | Built |
| `beliefs.Window`, `Tally`, `NextRun`, `Limits`, `Clamp`, `Gate`, `Draw`, `Reseed`; `Belief`, `Bounds`, `beliefParams`, `beliefReason`, `beliefsEvent` | row 4 | Built |
| `productionScratch` (`maxExecutions`, `start`, `complete`, `anchors`, `unitCost`, `addPending`), `job`, `yieldTable`, `yieldShocks`, `refusalTally` (`add`, `flush`), `recipeExecuted`, `recipeAbandoned`, `residueLen`, `setResidues` | row 5 | Built |
| `welfareScratch` (`addReceipt`, `addInputSpend`, `addWear`, `markWorking`, `markTraded`, `record`, `reset`), `history`, `newHistory`, `newTarget`, `readBuf` (`observedWage`, `regionWage`, `heldByFloor`, `summarise`), `floorRun`, `clampRun`, `seedWages`, `roleSummary` | row 6 | Built |
| The unexported `agent` type (`hasJob`), `newAgent(p agentInit, row, rt)` (`agentInit{id, account, role, weight, strategy, external, mortal, tick}`, plan 1-7 N-243), `drawCreation(s, a, targetRatio, spread)`, `defaultEndowment` and `fundInitial`, `reseedBeliefs` and `beliefSeed`, `upkeepTotals`, `consume(l, a, from, per, sc)` with `consumeScratch` (P6.1), `newViewScratch`, `(*viewScratch).fillCore` and `viewCtx` (the core of `fillAgentView`), `workReport` (P2.5 counts, P2.8a `ProductionHalted` rows, the floor count), `normalise`, `orderOf` and `outcomeStock` (the plan checks of `checkTradePlan` and the outcome loop); the exported `Strategy`, `AgentView`, `OutcomeView` (with the belief-parameter group), `TradePlan`, `Intent`, `Discard`, `Purpose`, `JobRequest`, `DefaultStrategy`, `Directed`, `Directive`; `WithStrategy`, `ErrUnknownStrategy`, `resolveStrategy(o, name, path)`, `isDirected`; the log-only `ReasonRest`, `ReasonHalted`, `ReasonNoRecipe` | Row 7 ([plan-phase-1-7.md](plan-phase-1-7.md) Provides) | Planned; names matched 2026-10-17. Row 7 builds **no** removal core and **no** combined role-change or weight-change step: the executor and `removeAgent` compose the primitives (D165) |
| `estates` with `openEstates(l, regions)`, `agentID`, `regionOf`, `account`, `seedPool`, `pool`, `grant`, `acceptMoney`, `acceptUnits`, `receive`, `acceptGoods`, `asks`, `facts`, `admitted`, `markdown`, `rows`, `restore` (all unexported, `estates.go`) | Row 8, [plan-phase-1-8.md](plan-phase-1-8.md) "Provides" | Planned; read 2026-10-17 and re-checked after the coordinator's note: `openEstates` after the built-in boundary accounts and before every other account; `seedPool` during construction only; `grant` takes the budget the executor computes (whole households) and the executor counts the unfunded households; weight reductions use `acceptMoney` and, for goods, `acceptUnits` (per commodity), a role change `acceptGoods` (plan 1-8 round-1 fix). This row calls `openEstates` at WLD 5.2 order 4, `seedPool` at 5.6 item 5, `asks`/`facts`/`admitted` at P3.4, `markdown` at P5.3, `receive` in `removeAgent`, `grant`, `acceptMoney` and `acceptUnits` in the executor; it owns the `receiptTally` and `clearScratch` row 8 assumes |
| `Spawner`, `DefaultSpawner`, `RegionView` (with `RoleView` and `RegionAgent`, built by this row each region and tick), `Decisions`, `Decision`, `DecisionKind`; `WithSpawner`, `ErrUnknownSpawner`, `resolveSpawner(o, name, path)`, `refusalReason`; `occupation.Refuse`, `Remaining`, `Renewal`, `Funded`, `Place`, `Retarget` (`Allowance` and `Exits` are the default `Decide`'s: caps and shares are policy, SPN 4); `entryTally` (`limited`), `extinctRun`, `unfundedRun`, `belowFloorRun`, `roleSignals` | Row 9 ([plan-phase-1-9.md](plan-phase-1-9.md) Provides) | Planned; names matched 2026-10-17. The executor is this row's |
| The view carries an exported plain-data group of belief params, filled by this row from `beliefParams` (plan 1-4) in `fillAgentView`; the SKL queue is not in the slice (tool half of the capital test only) | Row 7, coordinator rulings (c), (d) | Ruled |
| The warm-start table: `Config.WarmStart` (`Prices`, `Wages`, `Groups` with `Stock` and `Jobs`, `Pools`), plain slices validated by `Validate`. There is no reader function and none is needed or assumed; the application at construction is this row's (D191, Phase 3) | Stage 0 (`config.go`) and row 10 (the validation) | Built (the types); the application is this row's |
| Test helpers `observe_helpers_test.go` reused by row 12 | This row | Provided |
| `virtualStartOffsets(cfg, calendar, yieldTable, region, recipe)` (row 10) and `warmJob(j, yieldTable, region, season, recipe, principal, executions, remaining)` (`production.go`, row 5, built): the offsets of VIA 8.5.2 and the pending output of a job made from a virtual start (an empty offset set is `ErrWarmStartJob`, FR-VIA-053). Row 10 gives nothing else for the warm start: the apportionment, the stagger, the grants, the construction `Transform` and the pools are this row's pieces 3a to 3g (D191), so no further symbol is assumed from row 10 | Row 10, [plan-phase-1-10.md](plan-phase-1-10.md) Provides (N-280), row 5 | Planned; this row owns the stagger and the application order of 5.6.1. `NewWorld` never calls `SolveSteadyState` (FR-VIA-021; a text-search guard in Phase 3) |
| `checkViability(c *Config, lk lookup) viabilityOutcome` (unexported) and the phase 6 of `Validate`; `ErrNotViable`, `*ViabilityError` | Row 10 (N-276) | Planned. Decision A10: `NewWorld` calls `Validate` once and reads the warnings from the same outcome if row 10 exposes it; otherwise it calls `Validate` and `Warnings` and accepts the double cost that row 10's Phase 7 measures |
| The Stage 1 yield-shock setter: `yieldShocks.set` on the world's `yieldShocks` field (field name `shocks`), unexported | Row 5, read by row 12's `export_test.go` (`SetYieldShockForTest`, plan 1-12 Phase 7 and O3) | Built; the world keeps the field under that name so the test file reaches it |
| `SolveSteadyState`, `SteadyState.Programmes`, `Report` | Row 10 | Not consumed: the golden header is row 12's; the world never solves |

### Provides

| What | For | Contract |
|---|---|---|
| `World`, `NewWorld`, `Step`, `Run`, `Err`, `Tick`, `Config`, `ConfigHash`, `Audit`, `Snapshot` | consumers, row 12, Stage 2 | WLD 3, OBS 5 for the slice; not safe for concurrent use |
| `ErrWorldFailed`, `ErrInStep`, `ErrAuditFailed` | consumers, INT, CHK | root sentinels, matched with `errors.Is` |
| The unexported non-position methods `beginStep`, `endStep` and the position methods named by ID | row 12 (the per-position state hash replays the step from a `_test.go`), Stage 2 modules (append a guarded call) | no hook in production code (coordinator ruling 2); the order is the order of `tick.go` |
| `tick.go`'s position list (identifier, owner, `where`: `line` or `bracket`) and `notBuiltPositions` | the position table test, every module PR, row 12's row-to-method map (the `bracket` entries are its pseudo-rows `begin` and `end`) | staged form (D141); a module deletes its entries and appends its call |
| `market.go`: the per-market record and `MarketState` | INT's `Market` query (Stage 2), CHK (the report and window as saved state), row 12 | one record per market; the report and window are saved state |
| `fanin.go`: `fillAgentView`, `checkTradePlan`, `removeAgent`, `relocationRefusal` | modules (append steps), MIG and INT (the one entry point of the guard) | compositions only |
| `snapshot.go`: `Snapshot` and rows of the slice | IND, STY, EXP, `otelobserver`, row 12 (`RoleRow.MeanRatioBP`, `HeldByAgents`, `HeldByEstate` for S19) | plain data, canonical order, appended fields at the end |
| No draw-count reader | row 12 (FR-WLD-064 draw-count assertion) | The world exposes none; row 12 skips that assertion with a note (plan 1-12 Phase 3). Position methods may have any unexported name: row 12 keeps a row-to-method map in its test and fails on a position without a method or a method without a row |
| `worldState` | `checkpoint.go` (Stage 2) | the seed, the next agent ID, the construction flag, the diagnostic runs of the bounded `Warn`s, and the spawner's state per (region, role): the renewal carry and the "had an agent" bit (D145, N-265) |
| Fields row 12 reads: `World.ledger` (the `*ledger.Ledger`, scanned by the internal test of plan 1-12 Phase 5) and `World.shocks` (the `yieldShocks`, whose `set` is the Stage 1 shock seam) | row 12's `export_test.go` and internal tests | unexported field names kept as written; a rename is a cross-plan edit |
| The glossary terms of Phase 1 | all later rows | in `doc.go` before use |
| No configuration field, default or bound, no event or event field | | the pinned hash and `testdata/golden/off/` do not move (N-217) |

## Security review

The row has no input from outside the process other than the configuration (already
bounded and validated), the options and the consumer's extensions. Worst cases:

- **A consumer's `Clearing`, `Strategy` or `Spawner` is hostile or faulty.** The
  world never trusts a returned list: `clearMarket` validates the whole trade
  list (`ErrInvalidTrade`), the plan is checked by `checkTradePlan`, and a
  returned plan from the spawner is validated by its owner. A faulty extension can
  fail the step and so deny service to its own world only; worlds share nothing.
- **A panicking observer or extension.** The world marks itself failed and the
  panic propagates; no `recover` hides it (WLD 3.5.4). A consumer that recovers
  gets `ErrWorldFailed`, not a world in an unknown state.
- **Re-entrancy.** An observer that calls the world gets `ErrInStep`; the flag is
  one boolean, so it cannot be bypassed by a goroutine because the world is not
  safe for concurrent use and says so (a data race there is the consumer's; the
  library starts no goroutine, architecture.md 12.2).
- **Resource exhaustion.** The size of a snapshot is bounded by the configuration
  (OBS 5.6), the events of a step by the orders and trades, the logs by the cells
  of the configuration; the scratch is sized once and the validator's bounds cap
  every count (architecture.md 8.5). An overflow is handled where it occurs and
  never panics on a valid configuration (D162).
- **Information disclosure.** Logs carry identifiers and counts, never a whole
  configuration or an order book (rules 3.3.5); no secret reaches the library.
- **Determinism as integrity.** Logging and observation draw no random word and
  change no state (rules 1.4); the non-interference tests prove it (row 12 runs
  the pack-level proof).

## Compatibility and rollout

The library has no release yet (architecture.md 6.6 applies from the first
release). The one contract this row changes is the Stage 0 type name `World`
(D140): every consumer of `economysim.World{}` and `DefaultWorld()` is in the
module (`config/`, the tests); no external caller exists. The rename is one
commit with no behaviour change, the full suite and the pinned hash as its proof,
and it precedes the `World` simulation type so that no step of the work has two
types of one name. Everything else is new and unexported or appended: the new
sentinels and the `Snapshot` types are additive; `Config.World`, `[world]` and the
hash ids do not change. Rollback is a revert of the phase's commit; Phases 2 to 8
each leave the module building and the suite green, so a phase can be dropped
without the others. A dependency failure (a row 7 to 10 name that differs from A1)
stops the phase at its start, not mid-way: Phase 2 for `resolveStrategy` and
`resolveSpawner` (rows 7 and 9), Phase 3 for the rest.

## Testing (summary, testing.md section 11)

1. **Requirements touched, with their tests**:

| Requirement | Test (phase) |
| --- | --- |
| FR-WLD-001, 002, 003, 020, 021; FR-CFG-038 (world half) | option and name-check tables, including `ErrUnknownStrategy` and `ErrUnknownSpawner` (2) |
| FR-WLD-004, 040, 044 | position table test and a scripted step (2, 4, 8) |
| FR-WLD-041 | the no-`range`-over-a-map text guard (2, 11) |
| FR-WLD-005, 006, 007, 012, 110 | guard, failed-world, panic tests; faulty clearing; the audit mapping and the bracketing through `endStep` (2, 5, 8) |
| FR-WLD-008, 009 | `Run` table (2) |
| FR-WLD-010, 011 | deep copy (9); two worlds on two goroutines under `-race` (2) |
| FR-WLD-030 to 038 | construction tests (3) |
| FR-WLD-042 to 047 | rotation, two-pass `Work`, phase 3 order; the strategy never sees an estate, the pseudo-agent half of 042 (4, 5, 10) |
| FR-WLD-045, 048 | no-reservation window, also after P4.1; invalid list (5) |
| FR-WLD-049, 050, 058 | outcomes, window push, upkeep (6) |
| FR-WLD-051, 053, 056, 059 | phase 7 executor and removal tests, order of events (7, 10) |
| FR-WLD-052, 074 | the audit mapping, the totals cleared at P8.6 (8) |
| FR-WLD-062, 064, 066, 076 | the PCG state after construction, the construction order of the Stage 1 sub-steps, the observer does not change the stream (3, 10) |
| FR-WLD-054 | no event is built without an observer: the call-site allocation test, and the order test that an event follows its transition (10) |
| FR-WLD-069 (half) | the markets cleared in reverse order give the same trades (5) |
| FR-VIA-021, 040, 046, 047 | the text-search guards and the line of the warm start (3) |
| FR-VIA-031 to 038, 042, 052, 053 (world side) | the warm-start application tests: helpers, prices, wages, groups and stagger, jobs and their books, pools (3) |
| FR-PRO-016, 017, 048 | order of production, the rotation, the request check (4) |
| FR-WEL-003, 017, 011, 020 | wear cost start and valuation (4, 6); the working tick, markTraded after the trades (4, 5); the income Guard (6); the snapshot fields (9) |
| FR-BEL-007, 032, 033, 039; NFR-BEL-001; FR-CLR-025 | outcomes once per admitted order, no `Info`, non-interference, the seed price, the belief path allocation (6, 10) |
| FR-SPN-002, 005, 006, 007, 026, 027, 032, 033, 034, 042, 044, 045; FR-OBS-019; FR-AGT-001, 007, 008, 077, 032 | the executor table, the budget order, the counts, the N-265 Guard (7); the strategy never sees an estate (5, 10) |
| FR-SPN-004, 008, 010, 015 (world half), 035; FR-EST-014, 022 | the refusal table (one case per `Refuse` reason: nothing changed, counted in `Refused`, the `Debug` line), the pool-overdraw cases, the reduction to the whole weight, the entry into an existing agent emitting `WeightChanged` (7) |
| FR-SPN-046 (snapshot half); FR-PRO-042 | the `RoleRow` signals and the `JobRow` aggregates (9) |
| FR-SCL-001 to 005 | the `groupWeights` table test (3) |
| NFR-VIA-004 (stagger and apportionment half), FR-VIA-034, 037 | `FuzzApportion` and `FuzzStagger`, plus the table cases of 3a (3) |
| FR-AGT-024, 072, 073 | the view of P2.5 built after P2.1, the anchor passed once, the floor count (4) |
| FR-AGT-057, 079 (discards-then-intents half) | the receipts and spend totals equal the ledger's `trade` flows; the plan order (5) |
| FR-AGT-075 | `Privation` rises and falls at P6.1 (6) |
| FR-OBS-018 | the region on `RecipeExecuted` and `RecipeAbandoned` (4) |
| FR-EST-002, 011, 012, 013, 015, 016, 021 | removal region, no estate subject, order subject, proceeds and pool, initial money, conservation, `EstateOpened` (7, 10, 5, 3, 9) |
| FR-OBS-003, 007 to 013, 022, 024, 030 to 033, 040 to 042, 045, 051, 070 to 076, 090, 092, 097 | Phases 8, 9 and 10 (92 in 8, 97 in 5) |
| NFR-OBS-001 (call sites) | allocation test (10) |
| Owned by other rows | FR-WLD-060, 061, 063, 065, 067 to 069, 091, 100, 101 (Stage 0 for 060, 067 to 069, 091 and NFR-WLD-005; row 12 for 063, 064; CHK and modules for the rest; plan 1-12 audit round 3, NB-11); NFR-WLD-001, 004 (row 12); FR-VIA-039, the offsets function (row 10) |


**Traceability as built (Phase 11, 2026-10-20).** Each row of the table above, with the tests that carry its IDs in a `Requirement:` line (testing.md 4.5). The list is every test of the row's own test files that names at least one ID of the row, so a test can appear under several rows. An ID of a row with no `Requirement:` line in the row's own files is named after the list.

- **FR-WLD-001, 002, 003, 020, 021; FR-CFG-038 (world half)**: `TestDefaults_ANilLoggerAndObserverWriteNothing`, `TestNewObservers_SuppliedFlag`, `TestNewWorld_HashAndOptions`, `TestNewWorld_ObserversAreAdditiveAndOrdered`, `TestNewWorld_OptionsAndNames`, `TestNewWorld_SeedLastWins`, `TestObservers_CompositionOverAScenario`, `TestObservers_DeliverInOrder`, `TestWorld_ConfigIsACopy`.
- **FR-WLD-004, 040, 044**: `TestFanIns_AreCompositionsOfTheExpectedCalls`, `TestFanIns_TheShapeCheckCanFail`, `TestP11Moment_IsTheMomentOfTheTick`, `TestP81Age_SpoilageIsASinkInTheTickItHappens`, `TestPositions_SwappedCallsInTickGoFail`, `TestPositions_TableMatchesTheSpec`, `TestPositions_TheCheckCanFail`, `TestRelocationRefusal_HasNoRefusalInTheSlice`, `TestStep_JobOfDurationThreeCompletesInSPlusTwo`, `TestStep_OneTickJobStartsAndCompletesInItsStartTick`, `TestStep_WarmJobsCompleteWithTheirConstructionBooks`, `TestTickEnded_DebugLineCarriesTheLiveCounts`, `TestTickInspecting_ReplayEqualsStep`.
- **FR-WLD-041**: `TestWorldFiles_NoRangeOverAMap`.
- **FR-WLD-005, 006, 007, 012, 110**: `TestAuditFailure`, `TestAuditFailure_MapsAndBracketsTheFailedStep`, `TestEndStep_FailureBracketsTheStep`, `TestRun_FailedStepEndsTheRun`, `TestRun_FailedStepEndsTheRunAndAFailedWorldEmitsNothing`, `TestStep_AnInvalidTradeListFailsTheStep`, `TestStep_EveryStartedStepEndsAndAPanicEmitsNoTickEnded`, `TestStep_FlagOnEveryPath`, `TestStep_RefusedCallKeepsTheCause`, `TestWork_PanicLeavesTheWorldFailed`, `TestWorld_AuditOnAHealthyWorld`, `TestWorld_ConfigIsACopy`, `TestWorld_GuardsInStep`, `TestWorld_PanicFailsTheWorld`, `TestWorld_Run`, `TestWorld_RunEndingInAPanic`.
- **FR-WLD-008, 009**: `TestRun_FailedStepEndsTheRun`, `TestWorld_Run`.
- **FR-WLD-010, 011**: `TestSnapshot_BoundaryPendingConstruction`, `TestSnapshot_ChangesNothing`, `TestSnapshot_DeepCopyAndNoRace`, `TestWorld_TwoWorldsShareNothing`.
- **FR-WLD-030 to 038**: `TestBeginStep_ConstructionBucketIsEmittedOnceBeforeTheFirstTickStarted`, `TestFinishConstruction`, `TestNewWorld_ConstructionTakesOnlyTheCreationWords`, `TestNewWorld_EventsAndLogsAtConstruction`, `TestNewWorld_IDsAccountsAndEndowment`, `TestNewWorld_StateAndScratch`, `TestNewWorld_WarmStartEventsAndLogs`, `TestNewWorld_WarmStartPricesMoneyAndPools`, `TestNewWorld_WarmStartStaggerJobsAndBooks`, `TestWarmStart_StaggerBeforeTheOldestFirstTake`.
- **FR-WLD-042 to 047**: `TestP53_AnUnsoldEstateLotIsMarkedDown`, `TestPositions_NoReservationIsOpenAfterP41`, `TestPositions_TableMatchesTheSpec`, `TestRotationAt`, `TestStep_APlanIsExecutedDiscardsThenOrders`, `TestStep_AnEstateSellsThroughTheMarketAndHasNoWelfareSlot`, `TestStep_AnOrderTheBookCannotAdmitIsRejectedWithItsReason`, `TestStep_NoReservationIsOpenAtAnyTickBoundary`, `TestStep_OneTickJobStartsAndCompletesInItsStartTick`, `TestStep_RegionOfEventsIsTheRegionOfTheAccount`, `TestStep_RegionOfRecipeExecutedOverTwoRegions`, `TestStep_ScriptedMarketClearsAtTheHalfEvenMidpoint`, `TestStep_TheRecordOfAnIntentIsParallelToTheBook`, `TestStep_WorkIsCalledForEveryFreeAgentBeforeAnyStart`, `TestWork_RefusalsAndTheHaltAreCountedFromTheViewAsFilled`, `TestWork_ViewIsBuiltFromTheStateAfterP21`.
- **FR-WLD-045, 048**: `TestPositions_NoReservationIsOpenAfterP41`, `TestStep_AnInvalidTradeListFailsTheStep`, `TestStep_NoReservationIsOpenAtAnyTickBoundary`, `TestStep_ScriptedMarketClearsAtTheHalfEvenMidpoint`.
- **FR-WLD-049, 050, 058**: `TestP12Anchors_WageAndPrices`, `TestP51_WindowIsPushedOnlyForAMarketThatTraded`, `TestP54_AMarketWithNoOutcomeEmitsNothingAndKeepsItsRun`, `TestP54_OutcomesAreDeliveredOnceInTheOrderOfTheSpec`, `TestP61_UpkeepUnfedHungerPrivationAndTheShortfall`, `TestP63_WelfareRecordOfAScriptedTick`, `TestStep_ABuyerFindsItsOwnEntryWhenAnotherRecordSitsFirst`, `TestStep_FirstTickCostAnchorsReadTheStartingWage`, `TestStep_ReceiptsAndInputSpendEqualTheTradeFlows`, `TestStep_TheBuyersPurposeReachesEveryTradeOfItsBid`.
- **FR-WLD-051, 053, 056, 059**: `TestExecute_ARemovalInPhase7ReplacesTheAgentWithNewIDs`, `TestExecute_CountsHoldAtEveryTickOfAScriptedRun`, `TestExecute_RegionsInterleave`, `TestExecute_RemoveAndReductionToTheWholeWeight`, `TestExecute_TheBudgetIsTakenOnceAfterTheDeaths`, `TestExecute_TheWorldStepsThroughPhase7WithTheDefaultSpawner`, `TestRemoval_HasOneCallSiteOfTheCloseAndOneCompositionOfItsEvents`, `TestRemoveAgent_MovesEverythingToTheEstateOfItsRegionAndEmitsInOrder`, `TestStep_EventsComeInTheOrderOfTheTick`.
- **FR-WLD-052, 074**: `TestAuditFailure_MapsAndBracketsTheFailedStep`, `TestPhase8_AuditPassesEveryTickAndTotalsClearOnlyAtP86`.
- **FR-WLD-062, 064, 066, 076**: `TestConstruction_TextGuards`, `TestDecideRegion_EmptyPlanAndKeyedStream`, `TestNewWorld_ConstructionTakesOnlyTheCreationWords`, `TestNewWorld_WarmStartTakesNoRandomWord`, `TestObservers_ALoggerAndAnObserverChangeNothing`, `TestObservers_BeliefsAndSnapshotsAreTheSameWithAndWithoutALoggerAndAnObserver`. No test of this row: FR-WLD-076.
- **FR-WLD-054**: `TestStep_EventsComeInTheOrderOfTheTick`, `TestStep_SnapshotAgreesWithTheEventsAfterEachStep`.
- **FR-WLD-069 (half)**: `TestStep_AMarketIsClearedFromItsOwnStream`.
- **FR-VIA-021, 040, 046, 047**: `FuzzStagger`, `TestConstruction_TextGuards`, `TestNewWorld_WarmStartEventsAndLogs`, `TestNewWorld_WarmStartTakesNoRandomWord`, `TestWarmStart_TextGuards`.
- **FR-VIA-031 to 038, 042, 052, 053 (world side)**: `FuzzApportion`, `FuzzStagger`, `TestApportion`, `TestNewWorld_SeedPriceIsTheWarmEntryElseTheReference`, `TestNewWorld_WarmStartPricesMoneyAndPools`, `TestNewWorld_WarmStartStaggerJobsAndBooks`, `TestP12Anchors_WageAndPrices`, `TestP12Anchors_WageFallsBackToTheSeedWageWhenThePooledWageIsUnknown`, `TestPhase`, `TestPickOffset`, `TestStaggerKey`, `TestStep_FirstTickCostAnchorsReadTheStartingWage`, `TestStep_WarmJobsCompleteWithTheirConstructionBooks`, `TestWarmStart_JobWithoutItsToolIsRefused`, `TestWarmStart_PoolIsSeededByInitial`, `TestWarmStart_StaggerBeforeTheOldestFirstTake`.
- **FR-PRO-016, 017, 048**: `TestP12Anchors_WageFallsBackToTheSeedWageWhenThePooledWageIsUnknown`, `TestRotationAt`, `TestStep_OneTickJobStartsAndCompletesInItsStartTick`, `TestStep_WorkIsCalledForEveryFreeAgentBeforeAnyStart`, `TestWork_RefusalsAndTheHaltAreCountedFromTheViewAsFilled`, `TestWork_ViewIsBuiltFromTheStateAfterP21`.
- **FR-WEL-003, 017, 011, 020**: `TestMarkWorking_OncePerAgentPerTickOnEveryPath`, `TestP63_ANeverTradedMarketIsValuedAtTheReferencePrice`, `TestP63_NothingClampsIncomeToKeepTheRatioAboveOne`, `TestP63_WearIsValuedAtThePriceOfTheTick`, `TestP63_WelfareRecordOfAScriptedTick`, `TestSnapshot_JobsPendingAndSignals`, `TestStep_ABuyerFindsItsOwnEntryWhenAnotherRecordSitsFirst`, `TestStep_ARoleWithNoRecipesWorksByTrading`, `TestStep_AnEstateSellsThroughTheMarketAndHasNoWelfareSlot`, `TestStep_ReceiptsAndInputSpendEqualTheTradeFlows`, `TestStep_TheBuyersPurposeReachesEveryTradeOfItsBid`, `TestStep_WearFactsAreRecordedAtTheStart`.
- **FR-BEL-007, 032, 033, 039; NFR-BEL-001; FR-CLR-025**: `TestCallSites_AllocateNothingWithTheDefaultLoggerAndObserver`, `TestLog_TheBeliefUpdateWritesNoInfoAndDebugOnly`, `TestMarketTable_OrderAndSeed`, `TestObservers_ALoggerAndAnObserverChangeNothing`, `TestObservers_BeliefsAndSnapshotsAreTheSameWithAndWithoutALoggerAndAnObserver`, `TestP51_WindowIsPushedOnlyForAMarketThatTraded`, `TestP54_AMarketWithNoOutcomeEmitsNothingAndKeepsItsRun`, `TestP54_DebugLinesCarryTheirKeys`, `TestP54_OutcomesAreDeliveredOnceInTheOrderOfTheSpec`.
- **FR-SPN-002, 005, 006, 007, 026, 027, 032, 033, 034, 042, 044, 045; FR-OBS-019; FR-AGT-001, 007, 008, 077, 032**: `TestBuildRegionView_EveryFieldEqualsItsSource`, `TestBuildRegionView_ObservedFlagAndRemovedAgents`, `TestDecideRegion_EmptyPlanAndKeyedStream`, `TestExecute_ARemovalInPhase7ReplacesTheAgentWithNewIDs`, `TestExecute_CountsHoldAtEveryTickOfAScriptedRun`, `TestExecute_EnterIntoAnExistingAgentAndIntoANewAgent`, `TestExecute_EntryNeverOverdrawsThePool`, `TestExecute_ExtinctAndBelowFloorWarns`, `TestExecute_ReduceMovesTheDifferenceToTheEstate`, `TestExecute_RefusalsChangeNothingAndAreCounted`, `TestExecute_RegionsInterleave`, `TestExecute_RemoveAndReductionToTheWholeWeight`, `TestExecute_SwitchKeepsTheAccountAndMovesTheGoodsOutsideTheNewSet`, `TestExecute_TheBudgetIsTakenOnceAfterTheDeaths`, `TestExecute_TheWorldStepsThroughPhase7WithTheDefaultSpawner`, `TestExecute_WorkedReductionFloorsWhatTheAgentKeeps`, `TestGuardN265_PlannedRenewalEqualsTheRuleOnTheHouseholdsRemoved`, `TestNewWorld_IDsAccountsAndEndowment`, `TestOnlyOnce_SeesASecondPathAndAMissingOne`, `TestPseudoAgents_AreNeverTheSubjectOfAnAgentEvent`, `TestRemoval_HasOneCallSiteOfTheCloseAndOneCompositionOfItsEvents`, `TestRemoveAgent_MovesEverythingToTheEstateOfItsRegionAndEmitsInOrder`, `TestStep_AnEstateSellsThroughTheMarketAndHasNoWelfareSlot`, `TestStep_EventsComeInTheOrderOfTheTick`.
- **FR-SPN-004, 008, 010, 015 (world half), 035; FR-EST-014, 022**: `TestBuildRegionView_ObservedFlagAndRemovedAgents`, `TestExecute_ARefusedDecisionDoesNotStopThePlan`, `TestExecute_ATickWithNothingEmitsNoEntryLimited`, `TestExecute_DecisionKindsAreTheKindsOfTheRefusalRules`, `TestExecute_EnterIntoAnExistingAgentAndIntoANewAgent`, `TestExecute_EntryNeverOverdrawsThePool`, `TestExecute_ReduceMovesTheDifferenceToTheEstate`, `TestExecute_RefusalsChangeNothingAndAreCounted`, `TestExecute_RemoveAndReductionToTheWholeWeight`, `TestExecute_TheBudgetIsTakenOnceAfterTheDeaths`, `TestExecute_UnfundedHouseholdsAreNotDeferred`, `TestGuardN265_PlannedRenewalEqualsTheRuleOnTheHouseholdsRemoved`.
- **FR-SPN-046 (snapshot half); FR-PRO-042**: `TestSnapshot_JobsPendingAndSignals`.
- **FR-SCL-001 to 005**: `TestGroupWeights`, `TestNewWorld_IDsAccountsAndEndowment`.
- **NFR-VIA-004 (stagger and apportionment half), FR-VIA-034, 037**: `FuzzApportion`, `FuzzStagger`, `TestApportion`, `TestNewWorld_WarmStartStaggerJobsAndBooks`, `TestPhase`, `TestPickOffset`, `TestStaggerKey`, `TestWarmStart_StaggerBeforeTheOldestFirstTake`.
- **FR-AGT-024, 072, 073**: `TestWork_FloorCountAndWarnRun`, `TestWork_RefusalsAndTheHaltAreCountedFromTheViewAsFilled`, `TestWork_ViewIsBuiltFromTheStateAfterP21`.
- **FR-AGT-057, 079 (discards-then-intents half)**: `TestStep_ABuyerFindsItsOwnEntryWhenAnotherRecordSitsFirst`, `TestStep_ADiscardTheAgentCannotMakeIsRefusedWhole`, `TestStep_APlanIsExecutedDiscardsThenOrders`, `TestStep_ReceiptsAndInputSpendEqualTheTradeFlows`, `TestStep_TheBuyersPurposeReachesEveryTradeOfItsBid`. No test of this row: FR-AGT-079.
- **FR-AGT-075**: `TestP61_PrivationFallsOneForEachFedTick`, `TestP61_UpkeepUnfedHungerPrivationAndTheShortfall`.
- **FR-OBS-018**: `TestStep_RegionOfEventsIsTheRegionOfTheAccount`, `TestStep_RegionOfRecipeExecutedOverTwoRegions`.
- **FR-EST-002, 011, 012, 013, 015, 016, 021**: `TestExecute_ARemovalInPhase7ReplacesTheAgentWithNewIDs`, `TestExecute_RemoveAndReductionToTheWholeWeight`, `TestNewWorld_WarmStartPricesMoneyAndPools`, `TestPseudoAgents_AreNeverTheSubjectOfAnAgentEvent`, `TestRemoveAgent_MovesEverythingToTheEstateOfItsRegionAndEmitsInOrder`, `TestSnapshot_PoolIsTheEstateAccount`, `TestSnapshot_RebuiltFromEvents`, `TestStep_AnEstateSellsThroughTheMarketAndHasNoWelfareSlot`, `TestStep_EventsComeInTheOrderOfTheTick`, `TestWarmStart_PoolIsSeededByInitial`.
- **FR-OBS-003, 007 to 013, 022, 024, 030 to 033, 040 to 042, 045, 051, 070 to 076, 090, 092, 097**: `TestAggregates_AreSilentWhenAllTheirCountsAreZero`, `TestBoundWarn_FiresAtTheTenthPinnedTickWithTheBoundAndTheCounts`, `TestDefaults_ANilLoggerAndObserverWriteNothing`, `TestExecute_UnfundedWarnIsBounded`, `TestLog_KeysAreTheKeysOfTheRulesTable`, `TestLog_MessagesAreConstantAndInTheCatalogue`, `TestLog_NoInfoInTheTickAndNoError`, `TestLog_WorldCreatedHasTheKeysOfTheSpecWithTheScenariosValues`, `TestNewObservers_SuppliedFlag`, `TestNewWorld_EventsAndLogsAtConstruction`, `TestNewWorld_ObserversAreAdditiveAndOrdered`, `TestNewWorld_WarmStartEventsAndLogs`, `TestObservers_CompositionOverAScenario`, `TestObservers_DeliverInOrder`, `TestObservers_PanicPropagates`, `TestObservers_ReceiveTheEvent`, `TestP54_OutcomesAreDeliveredOnceInTheOrderOfTheSpec`, `TestP61_UpkeepUnfedHungerPrivationAndTheShortfall`, `TestP63_WelfareRecordClampedWarnsOnTheFirstTickAndEveryTenth`, `TestP64_HouseholdsUnfedWarnsAtTheTenthTickAndEveryTenthAfter`, `TestP82Flows_WearRowsFollowTheBoundaryRowsInCommodityOrder`, `TestPseudoAgents_AreNeverTheSubjectOfAnAgentEvent`, `TestRun_FailedStepEndsTheRunAndAFailedWorldEmitsNothing`, `TestScenarios_EmitEveryDeclaredEventOfTheSlice`, `TestSnapshot_BoundaryPendingConstruction`, `TestSnapshot_ChangesNothing`, `TestSnapshot_CountsAndScale`, `TestSnapshot_DeepCopyAndNoRace`, `TestSnapshot_Guards`, `TestSnapshot_PoolIsTheEstateAccount`, `TestSnapshot_RebuiltFromEvents`, `TestStep_EventsComeInTheOrderOfTheTick`, `TestStep_EveryStartedStepEndsAndAPanicEmitsNoTickEnded`, `TestStep_MarketClearedCarriesTheCostAnchorInForce`, `TestStep_ScriptedMarketClearsAtTheHalfEvenMidpoint`, `TestStep_SnapshotAgreesWithTheEventsAfterEachStep`, `TestWarnRun_IsTen`, `TestWarn_EachRowAtItsPatternAndAgainAtTheNextMultipleOfTen`, `TestWarn_EveryRowHasItsCounterpartCarryingTheNumberTheLineReports`, `TestWarn_NoCellWritesMoreThanOnePerRunPlusOne`, `TestWarn_WelfareRecordClampedThroughTheMethodOfP63`, `TestWarn_WorkFloorBindingOverARun`, `TestWorld_PanicFailsTheWorld`, `TestWorld_RunEndingInAPanic`, `TestWorld_RunLogs`. No test of this row: FR-OBS-011.
- **NFR-OBS-001 (call sites)**: `TestCallSites_AllocateNothingWithTheDefaultLoggerAndObserver`, `TestCallSites_TheTableIsTheStraightLine`, `TestObservers_DeliverAllocs`, `TestStep_StepAllocatesNothingWithTheDefaultObserver`.

The requirement IDs of the plan's list with no `Requirement:` line in this row's own files, and why: FR-AGT-079 (the discards-then-intents half) comes with the transport module, so only the Stage 1 half is covered (the other half is tested by `strategy_default_trade_internal_test.go`, row 7); FR-OBS-011 (the restore) is Stage 2 (`checkpoint.go`); FR-EST-021, FR-SPN-032 and FR-VIA-021, 047 were named in doc comments only and now carry a `Requirement:` line (`TestStep_EventsComeInTheOrderOfTheTick`, `TestConstruction_TextGuards`, `TestWarmStart_TextGuards`).

2. **Lowest level**: unit for the rotation, the position list and the options;
   integration for the tick (a Go `Config` of one or two agents with hand-computed
   arithmetic in comments); no end-to-end test in this row (row 12 owns them).
3. **New boundaries and error paths**: every sentinel of D162 has an
   `errors.Is` test; the faulty clearing, the audit failure, the panicking
   observer, the cancelled run, the negative `ticks`; a pure mapping and a bracketing
   test stand for the audit failure that cannot be injected (D190).
4. **Bug fixes**: none; a `Fix:` test is written first if one is found.
5. **Guards**: the staged position table, the one removal path, the
   fan-ins-are-compositions `go/parser` shape test (D195: no test-only list, since D166
   forbids a hook), the no-hook text search.
6. **Golden**: none changes (row 12 records the first).
7. **Fuzz**: `FuzzApportion` and `FuzzStagger` in `construct_warm_fuzz_test.go` (Phase 3;
   NFR-VIA-004, FR-VIA-034, FR-VIA-037): the stagger and apportionment half of NFR-VIA-004
   that plan 1-10 gives to this row (busy count never above `n`, result independent of
   iteration order, offsets are members of `virtualStartOffsets`' set, determinism). No
   other fuzz: no parser or validator is added.
8. **Coverage**: Phase 11 records it; no decrease allowed.
9. **Mutation**: Phase 11's hand mutations on a copy (testing.md 9.3); `gomu` only
   in Docker with `--cpus` and `--memory` (the global rule).

## Observability (summary, observability-and-logging.md section 9)

1. **Failure paths**: all are returned errors (`ErrInvalidTrade`,
   `ErrAuditFailed`, `ErrWorldFailed`, `ErrInStep`, `ErrOutOfBounds`); none is
   logged where returned. The survived conditions are the `Warn`s of D156, each
   with a counterpart.
2. **State transitions**: each has its event already (a row of OBS 4.1); the row
   emits them behind the observer flag. The names table of `WorldCreated` has no
   map.
3. **Decision points**: `Debug` lines behind `Enabled` (A5).
4. **Loops**: no `Info` in the tick; every `Warn` is per cell of the configuration
   (D156); the call-site allocation test proves the no-op path allocates nothing.
5. **Changed event, metric or attribute**: none.
6. **New metric attribute**: none.
7. **`Error`/`Info` use**: no `Error`; `Info` only at construction and in `Run`.
8. **Log and return the same error**: never (architecture.md 8.3).
9. **Tests**: Phase 10 and the snapshot tests of Phase 9.

## Open items

- **O1.** The coordinator accepted D140 (the rename of the section type), D141 (the staged position table) and D142 (rows 7 to 10 first, no fakes: testing.md 6.1, architecture.md 10.3) on 2026-10-17. D140 is FINAL (ruled by coordinator, reversible: architecture.md 2.2 names `World` as the simulation type, the module is unreleased, no hash input changes).
- **O2.** The names in "Consumes" marked ASSUMED are checked against the merged
  code of rows 7 to 10 before Phase 3 (A1).
- **O3.** CI does not exist (open item O3 of plan 0).
- **O4.** Closed by D190: the real `Ledger.Audit` failure cannot be injected from the root (`ClearTotals` re-anchors the bases, so clearing the totals leaves the identity true), so Phase 8 tests the pure mapping and the bracketing and says what it does not test. No seam is added to production code (coordinator ruling 2).
- **O5.** The alignment of the unexported method names with row 12's replay
  (D166) is confirmed with the row 12 plan before Phase 4 starts.
- **O6.** Closed by the coordinator's ruling and row 8's plan: the estate calls are row 8's, the executor of the spawner plan and `removeAgent` are this row's.
- **O7.** CLOSED by coordinator ruling (A), final, reversible: one joint largest-remainder apportionment over the group's n agents (quotas share*n/10 000 per JobSpec in canonical order, idle share last), floors first, remaining agents to the largest remainders, ties to the earlier entry (architecture.md 7.6), busy agents assigned in agent order spec by spec. Phase 1 deliverable 11 writes it into VIA 8.5.2 / FR-VIA-034. Piece 3a is not blocked.
- **O8.** CLOSED, FINAL (ruled by coordinator, reversible: the empty `p02()` and the `line`/`bracket` kinds are test and shell shape, not behaviour): (1) the position test compares the first area code of the Owner cell; (2) P0.2 ("nothing is cleared at entry") is an empty position method `p02()` with a one-line comment, called in `tick.go`; the parse does not special-case it and no permanent `notBuiltPositions` entry exists for it (testing.md 6.8: a spec position with no call fails); (3) and (4) were added in the round 2 fix, as consequences the owner may reverse: (3) the owner cells of P0.1 and P8.6 read "this spec": the test reads that as WLD; (4) each list entry has `where` (`line` or `bracket`, D141) so that the positions run inside `beginStep` and `endStep` are compared for identifier, owner and order but have no call to find in the straight line. Phase 1 deliverable 12 writes this into testing.md 6.8 and `spec-world.md` 6.1.
- **O9.** CLOSED, FINAL (ruled by coordinator, reversible: revisit only if a module ever needs a P0.2 effect): option (a), `p02()` stays in the straight line of `tick.go` and its comment says so; the plan as written. Original statement (round 2 fix): O8 (2) keeps `p02()` in the straight line of `tick.go`, so it executes after `beginStep` has run P0.1, P0.2a and P0.3, while the list is in spec order. It is empty and takes no state, so nothing observes the difference. Options: (a) keep as ruled and say so in the comment of `p02()` (the plan as written); (b) call `p02()` from `beginStep` between the guard and P0.2a and make P0.2 a `bracket` entry (the executed order is then the spec order, but the O8 wording "called in `tick.go`" changes and the row 12 replay map gains a bracket position). Recommendation: (a), because the method is a no-op and the ruling stands; revisit only if a module ever needs a P0.2 effect.
- **O10.** CLOSED by coordinator ruling (final, reversible: the cause set is not a public contract before INT and POP exist). In Stage 1 a `DecideReduce` is valid for the causes the default spawner emits (`starved`, `insolvent`); any other cause is refused through the existing refusal path (Phase 7, the refusal table of Phase 7, Phase 10), because a reduction's ledger reason must name what it is. A module that introduces a new cause (consumer-driven removal through INT, population) appends its cause constant and its ledger reason together in its own change (architecture.md 2.10.5); recorded as N-290.
- **O11.** CLOSED by coordinator ruling (final, reversible: option (b) stays open if a later spec wants the stock untouched). Option (a), the literal spec: the input `Transform` consumes oldest first, so the stagger vector is a property of the granted stock before the job's inputs are taken (Phase 3 3f, Phase 1 deliverable 13).

## As built

Written at the exit gate (Phase 11, 2026-10-20). The code is in the working tree on
`stage-1-11`, uncommitted: there is no commit hash yet.

### Exit gate

| Check | Result |
| --- | --- |
| `gofmt -l .` | prints nothing |
| `go vet ./...`, `staticcheck ./...` | clean |
| `go test -short -race ./...` | passes |
| `go test -race -covermode=atomic -coverpkg=./... ./...` | passes (about 160 s for the root package) |
| `go mod tidy` | no diff |
| `TestHashConfig_ReferencePinned` | passes, the pinned all-off hash did not move (N-217) |
| `lizard -l go -C 10 -a 8 -L 100 -w` on the production files of the row | clean after one split (below). The other warnings in the tree are in `validate.go` and `config/` (Stage 0, unchanged by this row but for a two-line edit) |

Coverage by scope, from the merged `coverage.out` of `-coverpkg=./...` (statements
covered over statements): root 98.6% (5384/5460); `config/` 99.8%; `internal/beliefs`
99.4%; `internal/checked` 100%; `internal/clearing` 98.9%; `internal/household`
99.3%; `internal/ledger` 99.8%; `internal/logkeys` has no statements; `internal/occupation` 100%;
`internal/ordering` 100%; `internal/production` 99.2%; `internal/steady` 99.9%;
`internal/welfare` 100%. Every scope is at 90% or more, so no test was added for
coverage. (`go test` prints a per-package figure of the whole `-coverpkg` set, for example 91.7% for
the root run; the figures above are per scope, as testing.md 9.1 reads them.)

Mutation pass (testing.md 9.3), by hand on a copy of the tree in the scratchpad, each
run with `go test -short -race .`:

| Mutation | Tests that failed |
| --- | --- |
| two positions swapped in `tick.go` (`p12Anchors` and `p21Complete`) | `TestPositions_TableMatchesTheSpec`, `TestPositions_SwappedCallsInTickGoFail`, `TestPositions_TheCheckCanFail` |
| `endStep` no longer marks the world failed | `TestStep_RefusedCallKeepsTheCause`, `TestRun_FailedStepEndsTheRunAndAFailedWorldEmitsNothing`, `TestStep_AnInvalidTradeListFailsTheStep`, `TestAuditFailure_MapsAndBracketsTheFailedStep`, `TestEndStep_FailureBracketsTheStep` |
| flow totals cleared (`ClearTotals`) at the entry of a step | `TestP82Flows_WearRowsFollowTheBoundaryRowsInCommodityOrder`, `TestPhase8_AuditPassesEveryTickAndTotalsClearOnlyAtP86`, `TestEndStep_FailureBracketsTheStep` |

`gomu` was not run (it needs the Docker CPU and memory caps of the global rule); the
Phase 10 mutations (a flipped `Enabled` guard and so on) belong to that phase's acceptance
and were not repeated here.

Two edits made at the gate: `positions()` in `tick.go` had a cyclomatic complexity of 11
(ten `if err` positions), so its second half is `positionsTail()`, reached by the last
line of `positions()`; `straightLineCalls` in `tick_internal_test.go` reads both
functions as one straight line and ignores only the hand-over call, so the test
is no looser (a call in either function that is not a position still fails it). And
`Requirement:` lines were added to four tests (above).

### Hand-over for row 12

**The step, in the order `Step` runs it** (D166; unexported, no hook in production code).
`beginStep` runs P0.1 (the guards and the in-step flag), P0.2a (`p02aBucket`: ends the
construction phase and delivers its bucket, `Tick` -1, once) and P0.3 (`TickStarted`).
`positions` and `positionsTail` call, in this order: `p02` (empty), `p11Moment` (P1.1),
`p12Anchors` (P1.2), `p21Complete` (P2.1), `p25Work` (P2.5), `p27Start` (P2.7),
`p28aHalted` (P2.8a), `p32Trade` (P3.2), `p34Estates` (P3.4), `p41Clear` (P4.1), then
`p51Window` (P5.1), `p53Markdown` (P5.3), `p54Outcomes` (P5.4), `p61Upkeep` (P6.1),
`p63Welfare` (P6.3), `p64Shortfall` (P6.4), `p71Entry` (P7.1), `p81Age` (P8.1), `p82Flows`
(P8.2) and `p84Audit` (P8.4). `endStep(err)` is P8.5 (a non-nil error fails the world before
`TickEnded`, with `Failed` true) and P8.6 (on success `Ledger.EndTick` and the welfare reset). The
replay in row 12 calls `beginStep`, the `line` positions of `positionList` in order and `endStep`;
`positionList` and `notBuiltPositions` in `tick.go` carry the identifier, the owner and
`where` (`line` or `bracket`; the brackets are P0.1, P0.2a, P0.3, P8.5 and P8.6). The four
fan-ins are in `fanin.go`: `fillAgentView`, `checkTradePlan`, `removeAgent` (the one
removal path; the executor's removal calls it) and `relocationRefusal` (no refusal in
the slice). The executor of the spawner's plan is in `phase_exit.go` (`entryRegion`,
`execute`, `execRemove`, `execReduce`, `execSwitch`, `execEnter`).

**Fields row 12 reads.** `World.ledger` (a `*ledger.Ledger`), `World.shocks` (the
`yieldShocks`), `World.markets` (the per-market records of `market.go`), `World.live`
(agent and household counts), `World.estates`, `World.seedWage`. There is no draw-count
reader: row 12 skips the draw-count assertion of FR-WLD-064 with a note.

**Packs that run in this row's tests.** `one.toml` and `warm-one.toml`, `two.toml` and
`warm-two.toml`, and `second/second.toml` with `second/warm-second.toml` (the construction
and warm-start tests, `construct_pack_test.go`, `construct_warm_pack_test.go`), plus Go
configurations built in the tests for the scripted steps (`observe_scenarios_test.go`:
the scenarios `starving` and the like, seed `obsSeed`). `five.toml`, `three.toml`, the
famine and demand packs and the defect overlays are row 12's.

**Events the world emits** (behind `obs.supplied`): `WorldCreated`, `AgentSpawned`,
`RunStarted`, `RunEnded`, `TickStarted`, `TickEnded`, `BoundaryFlow` and `WearApplied` (the
construction bucket at `Tick` -1 and each tick's), `ProductionHalted`, `RecipeExecuted`,
`RecipeRefused`, `RecipeAbandoned`, `OrderPlaced`, `OrderRejected`, `TradeExecuted`,
`MarketCleared`, `BeliefsUpdated`, `UpkeepShortfall`, `AgentBankrupted`, `EstateOpened`,
`WeightChanged`, `RoleChanged`, `EntryLimited`. No event or event field was added.

**Test helpers row 12 may reuse.** `observe_helpers_test.go` (the recording `Observer`, the
capturing `slog.Handler` and the level-counting handler); `construct_export_test.go`
(`ConstructionRows`, `ConstructionAccounts`, `MarketRecords`, `WorldCounts`, `AgentLots`,
`AgentJob`, `PoolMoney`, `StartOffsets`, `SeedWages`, `ConstructionFlows`, `PCGState`);
`tick_export_test.go` (`UnitCostOf`, `HasReservation`); `snapshot_export_test.go`
(`RoleSignalsOf`); `observe_scenarios_test.go` (`scenarios`, `play`, `who`).

### Deviations from the plan, and what is left

- `positions()` is split into two functions for the Lizard limit, and its test reads both (above).
  The plan said "the straight-line function"; this is the smallest change that keeps the
  position test strict and the file within N-243.
- Coverage needed no new tests. The mutation pass is the three hand mutations of Phase 11,
  not `gomu`.
- FR-AGT-079 (transport half) and FR-OBS-011 (restore) have no test in this row; both are owned by later stages.
- The roadmap row and the documentation index row say "uncommitted": the commit hash is added
  by whoever commits.
