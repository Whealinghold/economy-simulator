# Plan: Stage 1, row 12: the off-only golden, the per-position hash and the slice's pack-run tests

**Status: plan written (2026-10-17, branch stage-1-12 to be cut; audit rounds 1 to 4 applied; final audit READY with no blocking finding and its ten non-blocking items applied; J1 to J3, O1, O9 and O10 ruled by the coordinator, final and reversible).**
Sequences the twelfth and last row of Stage 1 of
[roadmap-v1.md](../roadmap-v1.md), "`testdata/golden/off/` for `one.toml` (and
`warm-one.toml`); the per-position hash helper", into nine phases. It also gives
a phase to every item of the Stage 1 "Done when" list that no earlier row owns
(the split is in the Scope table and was ruled by the coordinator on 2026-10-17).
It implements [spec-reference-chain.md](../specs/spec-reference-chain.md) (REF)
sections 7.2, 7.4 to 7.5, 8 and 9 and [testing.md](../rules/testing.md) 6.3, 6.4,
7.1 and 7.1.1 to 7.2 for the slice. It is built against `doc/rules/architecture.md`
(2.6, 2.10.6, 7.4, 10.2 to 10.5), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md` (8, 9). Where this plan and the rules or
the specs disagree, they win and this plan is corrected.

Rule tags read for this plan: architecture=905c0e578c4e662908638dd8,
observability=35827912d1e3141a76af3937, testing=b3559bc6d47470c221bc4642.
Notes tag (read 2026-10-17 for the round-2 fixes): b1d3cafecfaad39b6e7e470b. The
notes N-294 to N-300 below are **already in `doc/notes.md`** (written with this
plan); Phase 1 re-quotes the tag it finds then, verifies those seven notes against
the final text and amends them where a ruling changed (for example N-295, N-296, N-298),
and gives the file a fresh tag only if it edits it.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR,
the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md) (D3 and
D5 above all) or a ruling of the coordinator, cited where used. Where a decision
is still owed, Phase 1 makes it in the document that owns it, before the code
that depends on it. Decision IDs D170 to D189 are this plan's (D170 to D181 are
used); note IDs N-294 to N-303 are reserved for it (N-294 to N-300 are used).

No PRD exists for this row. The requirements source is the roadmap row, the
Stage 1 "Done when" list and REF. This row is **last**: it is built after the
world row (row 11, `world.go`, `tick.go`, `market.go`, `snapshot.go`) and after
the viability row (row 10), because every test below runs a world or reads the
viability rules. The only part that does not need them is Phase 1 (documents) and
the first half of Phase 2 (the plain-text parser and the renderer's unit tests on
hand-built values).

Spec codes: REF reference chain, STB stability, STY stylised facts, OBS
observability, WLD world, CFG config, VIA viability, AGT agents, BEL beliefs, PRO
production, WEL welfare, SPN spawner, EST estates, CLR clearing, LED ledger, CHK
checkpoint, INT integration, IND indicators. `FR-`/`NFR-` IDs are the
requirement IDs of those specs.

## What Stage 0 and the earlier rows already give, and what is left

| Concern | Where it already is |
|---------|---------------------|
| The packs `one` (`base`, `one`, `warm-one`), `second` (`second/second`, `second/warm-second`) and the overlays `estates`, `skill`, `a5`, `nofloor`, `notool`, `imbalance-cook`, `glut-prepared`, `h200`, `long-grow`, `overhang`, `poor`, `perturbed` and `defects/d1` to `d6`; `testdata/chains/` holds no `expected/`, `bands/`, `scripts/` or `golden/` | `testdata/chains/` (Stage 0) |
| The loader `config.LoadFS(fsys, layers...)`, the pinned hashes of `one` and `second` (`TestPacks_LoadValidateAndHash`), the six defect overlays loaded over the cold and the warm layer set with `Validate` expected nil and only `d6`'s warning asserted (`TestPacks_DefectOverlays`) | `config/integration_test.go` (Stage 0) |
| `referenceConfig()` (pack `one` built in Go), the "file and Go agree" test, the configuration-only test of the derived tables on `second` (`TestTables_ConfigurationOnlyAddition`: tables only, no run) | `reference_test.go`, `fileagree_test.go`, `config_only_test.go` |
| The test-only seam of the root package; each row appends its aliases and wrappers | `export_test.go` |
| `WithAudit`, `WithLogger`, `WithObserver`, `WithSeed`, the sealed events, `Observer`, `ObserverFunc`, the delivery loop and the "observer supplied" flag | `option.go`, `event.go`, `observer.go` (Stage 0) |
| `internal/ledger` with `Audit`, the boundary accounts and flow totals; `Discard`, `Holding` | rows 1 and 5 (`ledger.go`, `internal/ledger`) |
| Clearing with `Check` (the clearing price between the matched limits) and the market report | row 3 (`clearing.go`, `internal/clearing`) |
| Beliefs, production, welfare arithmetic and their root glue | rows 4, 5, 6 |

What is left for this row is the proof, over the real packs, of everything the
slice claims: the recorded golden files, the per-position state hash, the six
invariants over seeds, determinism with and without observation, the
configuration-only chain on `second`, the defect files, the first stability
scenarios, and the pack-level halves of the observability tests. **No production
code is written by this row**: every file it creates is a `_test.go` file, a
fixture under `testdata/` or a document (architecture.md 10.2: no back door).

## Scope

The split below is the coordinator's ruling of 2026-10-17 (Q1 (a)); row 11's
plan is told the same.

**In scope (row 12 owns, each in the phase named).**

| Item of the Stage 1 "Done when" | Phase |
|---|---|
| The golden of `one` for the length REF fixes (400 ticks, seed 1) runs and passes: `testdata/golden/off/one.txt` (REF 8.5 items 1 to 5) | 3 |
| The golden rendering of events `testdata/golden/off/events-one.txt` and its renderer, with the zero-field rule of OBS 9.5.2 | 2, 3 |
| The same seed gives the same bytes with and without a logger and an observer, split into any number of `Run` calls | 3 |
| The per-position state hash helper (testing.md 7.1.2) and its recorded file | 4 |
| The invariants of testing.md 6.3 over the seed range of REF 7.5 (`one` seeds 1 to 16, `second` seeds 1 to 4) | 5 |
| The configuration-only chain test on `testdata/chains/second/` (X4, FR-REF-031) | 6 |
| The six defect overlays load over the base layers REF names and fail the rule or give the warning REF assigns (the file tests, A2) | 6 |
| FR-REF-007: the test-only canonical TOML writer and the byte test of the `warm-*.toml` (`config/warm_bytes_test.go`; plan 1-10 D133, O10) | 6 |
| `testdata/chains/expected/one.toml` and `expected/second.toml` (config hash, solver report, warnings; J2), recorded with `-update` by `config_test` `TestExpected_Packs` and read by the same test with go-toml (FR-REF-005, 006, 039) | 6 |
| The first stability scenarios that need one region only: S01, S04, S07, S16, S19, their outcomes recorded, pass or not | 7 |
| The pack halves of the observability tests: cardinality over `one` (FR-OBS-005, 006) and the allocation limit of a full `Step` (NFR-WLD-001, NFR-OBS-002); non-interference of the whole run (FR-CFG-026 world half, FR-OBS-023) | 3 (non-interference), 8 |
| The spec, note and roadmap edits these depend on | 1 |

**Owned by another row (this row only consumes or gates on it).**

| Item | Owner |
|---|---|
| The position table test (testing.md 6.8), the registration-option halves that need a world (FR-WLD-001, 002, 003, FR-OBS-090), the run lifecycle and tick bracketing (FR-OBS-012, 013), composition (FR-OBS-024), the emission order of a scripted step (on a recording observer, not on this row's renderer), the log tests (the level, message and keys of every `Warn` and `Error` path, no `Info` in the tick, the counterpart of every `Warn`), the call-site allocation test, the catalogue test with the world emitting each declared event | Row 11 (`world.go`, `tick.go`) |
| The viability rules, `ViabilityError`, the failing-case tests of the rules (unit, on documents), the solver and its report | Row 10 |
| The registration options `WithClearing`, `WithStrategy`, `WithSpawner` and their duplicate-name and nil-extension unit tests | Rows 3, 7, 9 |
| The role-level behaviour the scenarios exercise (`Work`, the work floor, the halt, entry and exit) | Rows 7, 9, 11 |

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|---|---|
| REF 8.5 item 6, the checkpoint digest, and the golden of `five` | Stage 2: the checkpoint is a core file and not a module, so adding the digest moves `testdata/golden/off/` once, with the testing.md 7.2 explanation (N-294) |
| `five`, `famine`, `one-demand` in the invariants and determinism tables of REF 7.5 and 8.2 | Stage 2 modules (estates part, demand, regions); N-295 lists them |
| `three` in the same tables (FR-REF-028, 025: 400 ticks, seeds 1 to 4) | **Probe in Phase 5 (judgement call J1):** `three` has no module and loads and validates today, and REF 16.2 runs `two`, `three`, `five` as regions that do not trade; the row is added to the invariants table if the Stage 1 world builds and runs it, otherwise its omission is listed in the PR (FR-REF-025) and N-295 says why |
| X5 (checkpoint and resume), A7 and A9 beyond the construction audit, A1, A3, A5, A10 | Row 10 for the solver tests; Stage 2 for the checkpoint |
| `expected/<pack>.toml` of `two`, `three`, `five`, `famine`, `one-demand` | The first change that asserts them (N-298); `one` and `second` are in scope above (J2) |
| S01 leg (b) (a visitor buys): needs `AddAgent` and `Direct` of `integration.go` | Stage 2, INT (N-296); the leg is a skipped test with that note as its ID |
| The bands (values) of `bands/one.toml`, the regime classifier, the hold-out | STY (Stage 2): this row creates the file with every key unset (REF 7.1.5) |
| A CI configuration, the arm64 run of the golden, timed fuzz | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |
| The scenarios S02, S03, S05, S06, S08 to S14, S17, S18, S20 to S22 | Not in the Stage 1 list; REF 7.2 keeps them, and a later change adds them (FR-REF-035 is checked for Stage 1's list only) |

## Decisions this plan rests on

Decided by documents already:

- The golden is a canonical text rendering of the reference chain run, written
  only with `-update`, explained in the PR when it changes, with no floating
  point, map order, pointer or wall-clock time (testing.md 7.1, 7.2; REF 8.5;
  FR-REF-029).
- There are three golden sets, `off/`, `<module>/` and `all/`; the off set holds
  the reference chain with every module off, and a module's change leaves it
  unchanged (testing.md 7.1.1; architecture.md 2.10.5, 2.10.6; D5 of plan 0).
- The per-position hash is a test helper in the root package's `_test` files and
  not an exported API; on a golden failure it reports the first tick and position
  at which the state diverged (testing.md 7.1.2).
- A scenario asserts a band only when its key has a value; it always asserts its
  exact and O checks, and reports the rest (REF 7.1.5, FR-REF-022, 023).
- Seeds and lengths are the table of REF 8.2 and 7.5, test constants (FR-REF-025):
  band scenarios seeds 1 to 10, the golden seed 1 at 400 ticks, determinism seeds
  1 and 2 at 400 ticks, invariants `one` 400 ticks seeds 1 to 16, X4 `second` 1 095
  ticks seeds 1 to 4.
- Every scenario test begins with the `testing.Short()` skip and states its level
  (testing.md 10.5; FR-REF-026). A failure message names the scenario, the pack,
  the seed and the measure (FR-REF-020).
- A measure is computed from events and the snapshot only, by a recording observer
  in a `_test.go` file, without unexported state (FR-REF-027, FR-STB-012). The
  exceptions below are named and tested.
- The tests are table-driven, parallel where they share no state, use `t.Error` and
  `t.Fatal` as testing.md 3.4 says, and do not use an assertion library (testing.md
  3.6).

Decided by the coordinator (2026-10-17) and carried here:

- **D170.** The split of the "Done when" items between this row and row 11 (Scope).
- **D171.** Both golden files live in `testdata/golden/off/`: `one.txt` and
  `events-one.txt`. testing.md 7.1.1 (a rule) wins over REF 17 and 8.5, OBS 9.5.3
  and the roadmap line that name `testdata/golden/one.txt` and
  `testdata/golden/events-one.txt`; Phase 1 corrects those lines. The cold `one`
  (no warm-start document) is a reported leg of S18 only, not a golden.
- **D172.** The Stage 1 `one.txt` has REF 8.5 items 1 to 5, **including the solver
  report in the header** (consumed from row 10 as ASSUMED: `surplus_bp`, the labour
  share, `pivots` and `unpriced` of the one region) and **excluding item 6** (the
  checkpoint digest: no checkpoint exists). No line is reserved for item 6.
- **D173.** The row 10 failing-case tests of the rules are unit tests on documents;
  this row owns the d1 to d6 **file** tests, extending `TestPacks_DefectOverlays` in
  `config/`. **FINAL (ruled by coordinator, reversible):** row 10 owns the rule unit tests, this row the file tests.
- **D174.** The per-position hash helper lives in a `_test.go` file of package
  `economysim` and **replays the step by calling the unexported position methods**
  in the order of the position table; it adds no hook, field or exported symbol to
  `Step` (architecture.md 10.2; ADR 0007 open point 2: test-only until a consumer
  asks). Requirements this puts on row 11 (ASSUMED, in Consumes): every position is
  an unexported method of `World`; the non-position parts of `Step` (the guard, the
  tick advance, the `TickStarted`/`TickEnded` bracketing, the failed-world marking)
  are unexported methods that `Step` and the replay both call, and `endStep` clears the
  `stepping` flag (plan 1-11 item 4 says so after audit round 3, NB-1): the replay
  does not go through `Step`'s deferred call, and a flag left set would make the
  `Snapshot()` between replayed steps and the next `beginStep` return `ErrInStep`.
  **FINAL (ruled by coordinator, reversible): O1.**
- **D175.** S19 is built in Stage 1 with `RoleRow.MeanRatioBP` (OBS 5.3.4),
  `HeldByAgents`, `HeldByEstate` and `ProductionHalted`; its aggregate-ratio band
  (`HouseholdIncome`, IND) is reported only (`OK` is false in version 1, REF 7.2).

Decided by this plan, to be written into the owning document in Phase 1 before the
code that depends on it (each is ruled by the coordinator and reversible):

- **D176.** The renderer of REF 8.5 and OBS 9.5 is **test code in package
  `economysim_test`** (`render_test.go`), built on a recording observer and the
  public `Snapshot`; it reads typed IDs through the `Names` table of the snapshot
  and renders by reflection over exported fields for events (OBS 9.5.3: "a
  `_test.go` helper using reflection over the exported fields"). It is this row's
  helper; row 11's emission-order test uses its own recorder and does not import it.
- **D177.** The state hash of D174 is a **reflection walk** over the whole `World`
  value graph (`sha256` over a canonical byte encoding; maps by sorted key; pointers
  by first-visit index with cycle detection; `func`, `chan`, `unsafe.Pointer` and the
  fields in the skip list skipped, by name, in a list held in the test). A walk is
  state-complete by construction, so a row or module that adds a field is hashed
  without an edit here, which is what "the ledger, the agents and every module's
  state" (testing.md 7.1.2) asks. The enumerated alternative (a hand list of fields)
  would silently miss a new field and is rejected. The cost (a walk of about 600
  agents per position) is paid only in the scripted run of D178.
  **Off-invariance (testing.md 7.1.1; architecture.md 2.10.3, 2.10.5).** The
  encoding of a struct is keyed by **field name** and **omits a field at its zero
  value** (nil pointer, nil or empty slice or map, 0, false, the empty string: the
  zero rule of OBS 9.5.2), so a module field appended at its off value (a nil module
  pointer behind the `!= nil` guard) leaves every line of `one.positions.txt`
  unchanged. A position whose guarded call is not taken (its module is off) is
  **not recorded**: the file lists only the positions that ran, so a module PR does
  not add a line for itself while off. The cost: renaming a field that holds a
  non-zero value moves the file, a behaviour change to explain per testing.md 7.2.
  A unit test (Phase 4) shows that adding a zero-valued field to a hand-built struct
  leaves the hash unchanged.
- **D178.** The scripted run of the hash is **pack `one`, seed 1, ticks 0 to 29**
  (30 ticks, which covers the first stagger completions of the short recipes and the
  first tool replacements' start but not the 60-tick harvest), recorded to
  `testdata/golden/off/one.positions.txt` as one line per position,
  `t=<tick> <position> <16 hex>`: the first 8 bytes of the SHA-256. About 30 ticks
  by the core positions of the table of `spec-world.md` 6 (about 40) is about 1 200
  lines. The file is a golden (testing.md 7.2: `-update` only).
- **D179.** The invariants test (Phase 5) computes conservation **from the identities
  of `spec-ledger.md` 9.1 and FR-OBS-042 and the snapshot**, exactly, after every
  tick: (i) per commodity, the units held in accounts (the `Markets` rows'
  `HeldByAgents` + `HeldByEstate`) plus the sum of the boundary balances of that
  commodity is 0. **Pending output is not added**: the output of a job in progress is
  in no account and the identities of 9.1 hold without it (`spec-ledger.md` 8.2.2,
  FR-LED-047); there are no shipments in Stage 1. (ii) the money of the snapshot
  header plus the sum of the money boundary balances is 0. (iii) FR-OBS-042, per
  (reason, commodity) and for money: `Boundary` balance == the construction term + the sum
  of the `BoundaryFlow` events of every tick since + `Snapshot.Pending` (the signed
  totals, the sign convention of OBS 5.3.3). **The construction term**: before the
  first `Step` it is `Snapshot.Construction`; after it that field is empty (OBS
  5.2.4, 5.3.3) and the term is the sum of the `Tick` -1 `BoundaryFlow` events, so
  a recorder never adds both. `Snapshot.Pending []FlowTotal` (empty in Stage 1) is
  not `Markets[].Pending` (units of jobs in progress, which identity (i) excludes);
  the code names them apart. The production identity of `spec-ledger.md`
  8.2.2 (units fixed at job starts == created by `recipe_output` + pending + dropped
  from pending) is left to row 5's tests; this row does not repeat it. The
  negative-balance check scans the ledger's accounts
  through the internal test seam of Phase 5 (`invariants_internal_test.go`, package
  `economysim`), the one place this row reads unexported state, because the snapshot
  holds sums and not accounts; `WithAudit` also runs (it checks the same, `spec-ledger.md`
  9.1).
- **D180.** The clearing-price invariant (5) is checked from events: for each
  `TradeExecuted`, the price lies between the seller's ask limit and the buyer's bid
  limit of the same tick, read from the `OrderPlaced` events of the two agents for
  that commodity (one order per agent, commodity and side per tick: N-185); an
  estate's ask does have an `OrderPlaced` (EST 2.7.4, plan 1-8), but the one-order
  premise fails for an estate, which places one ask per held lot and whose event
  carries no lot, so a trade whose seller is an estate (agent ID below the region
  count, plan 1-8 `regionOf`) or whose side has no `OrderPlaced` (a counterparty) is
  counted and checked against the market's bounds only, and the count is reported. `Step`
  returning `ErrInvalidTrade` is also a failure (row 3's `Check`).
- **D181.** The recorded outcomes of the scenarios (Phase 7) are written by `-update`
  to `testdata/chains/expected/one.scenarios.txt`, one block per scenario and seed
  range, with the measure values and the verdict of each exact and O check; the file
  is **read by the reviewer of STY 8.4 and not compared** by the tests (a band is
  never set from the code under test without review, FR-REF-024), and a test fails
  only on an exact or O check or on a band whose key has a value.

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** Names of other rows' symbols below (`NewWorld`, `World.Step`, `World.Run`,
  `World.Snapshot`, `World.Tick`, `World.ConfigHash`, `Snapshot`, `RoleRow`,
  `MarketRow`, `Names`, `Boundary`, `Construction`, `Pending`, the position methods)
  are those of `spec-world.md` 3, `spec-observability.md` 5.3 and `spec-integration.md`
  and are **ASSUMED** until the plans of rows 10 and 11 fix them. A row that renames
  one tells this plan's author; the tests change by their call sites only.
- **A2.** `NewWorld(cfg, opts...)` with no `WithSeed` has seed 0 (architecture.md 6.2);
  every test names its seed.
- **A3.** The root package's external test package (`economysim_test`) may import
  `config/` (the existing tests do); the internal test package (`economysim`) may not
  (it would cycle). Whatever needs the packs and unexported state is two files: an
  external one that loads and runs, and an internal one with a function taking a built
  `*World`, as plan 1-6 A8 did. `checkGolden`, `updateFlag`, `loadPack`,
  `renderGolden` and `referenceConfig()` live in the external package and are
  unreachable from the internal one; so **every golden test is an external test**
  (the internal files hold only the walk, the replay and the account scan, each
  exported by a one-line wrapper in `export_test.go`), and the `-update` flag is
  defined once, in `support_test.go`.
- **A4.** No third-party package is imported by a root test (the import guard of
  `import_guard_test.go`); the small grammar of `bands/` and `scripts/` is read by
  the plain-text parser of Phase 2 (`plainTOML`), as the registry tests of testing.md
  6.7 read their tables.
- **A5.** The `-update` flag is a `flag.Bool` in a `_test.go` file (the usual Go
  golden pattern); the determinism guard of architecture.md 7.1 governs production
  code and its text search (`determinism_guard_test.go`, `guardedFiles`) excludes
  `_test.go` files. Phase 2 checks this before relying on it.
- **A6.** The run time of the whole row is **not** a few minutes. At REF 8.2's about 4
  to 6 ms a tick (placeholder cost model) the row runs about 84 000 scenario ticks,
  6 400 invariant ticks for `one` (plus 1 600 for the four `second` seeds, 8 000 in all, and 1 600 more if the `three` row is added, J1), 4 380 for X4 and about 12 000 for determinism, roughly
  110 000 ticks of `one`/`second`, and the full command of `AGENTS.md` adds `-race
  -covermode=atomic -coverpkg=./...`, which will exceed Go's default 10 minute
  package timeout for the root test binary even with parallel subtests. The tests are
  skipped with `-short`; the seeds of REF 8.2 are not reduced (FR-REF-025). What
  to do about the timeout is ruled in J3 (Open items): no rule-file edit now; the gate runs the full
  command by hand with `-timeout 90m`. Allocation
  tests are the exception to "parallel by default" (Phase 8). The measured time
  is recorded in Phase 9 against NFR-REF-002.
- **A7.** One PR per phase, in phase order, on a branch from the merge of row 11.
  Phase 1 can be reviewed earlier. The row shares `doc/notes.md`,
  `doc/documentation_index.md`, `doc/roadmap-v1.md`, `doc/specs/spec-reference-chain.md`,
  `doc/specs/spec-observability.md`, `doc/rules/testing.md` and `export_test.go` with
  the other rows: each PR re-reads the shared file just before editing it and edits
  with a targeted change.
- **A8.** Row 10 lands the viability rules **before** this row's Phase 6. Its PR must
  keep `TestPacks_DefectOverlays` green, so it flips the Stage 0 line "`Validate`
  = nil at Stage 0" for d1 to d5 to the expectation `errors.Is(err, ErrNotViable)`
  (the minimal edit); this row's Phase 6 then adds the rule check, "that rule only",
  the warm layer and the `d1` with the wild source on (O2).

## Phase 1: settle the owed spec lines and the notes

**Why first:** the paths, the contents and the Stage 1 subset of the golden and the
invariants are stated in four documents that disagree today, and a test that
chose among them would put a result-affecting choice in code (testing.md 7.1.1 is a
rule and wins). This phase changes documents only, so it starts by reading
`doc/notes.md` in full and quoting its tag, handles every open note tagged REF,
STB, OBS, CODE and ALL (below) and gives `doc/notes.md` a fresh tag
(`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **REF 17, 8.5 and FR-REF-029, 030.** Replace `testdata/golden/one.txt` and
   `five.txt` by `testdata/golden/off/one.txt` and `off/five.txt` (D171), say that
   `events-<pack>.txt` is `testdata/golden/off/events-<pack>.txt`, and add to 8.5 the
   Stage 1 form: **item 6 is added by the checkpoint change (Stage 2)** and until
   then the file ends with item 5 (D172, N-294). Keep the header's solver report.
   In the same lines correct `testdata/golden/events-all-modules.txt` (REF 886 and
   17) to `testdata/golden/all/events-all-modules.txt` (the `all/` set of testing.md
   7.1.1); `experiment-one.txt` and `stylised/<pack>.txt` stay where they are, as
   they are tied to no module set. Apply the ruling on open item O9 (closed, option (a)) to item 2
   of REF 8.5 (the mean money per household). Add to REF 7.6 a line assigning the
   run legs of A1 (warm run within `a1.price_band_bp` for 30 ticks, cold run not
   better) and A7/A8 (conservation, stagger) to the stage that owns a running
   world with the solver (plan 1-10 gives A7/A8 to the world row, plan 1-11 does not
   take them, row 10 has no world): they are listed there, not tested in Stage 1.
2. **OBS 9.5.3, FR-OBS-081 and the roadmap line "testdata/golden/events-one.txt"
   (Stage 1 "Done when") and the roadmap row's citation "REF 10".** Correct the
   path of the events file to `testdata/golden/off/events-one.txt`; the roadmap row
   cites REF 8.5 and FR-REF-028 to 030, not "REF 10" (REF 10 is "Parameters" and
   holds no golden). The roadmap "Done when" also cites FR-OBS-024 for
   non-interference; OBS 9.3 and FR-OBS-023 give non-interference and FR-OBS-024 is
   composition and the panicking observer (row 11's): correct the citation to
   FR-OBS-023 and keep FR-CFG-026 world half.
3. **REF 7.5 and 8.2 (the Stage 1 subset).** State once, in a paragraph after the
   table of 7.5, that the rows of `famine`, `five` and `one-demand` need the estates
   part, regions that trade and the demand module, so the slice runs `one`, `second`
   and `three` (if Phase 5's probe shows the Stage 1 world runs it, J1) and the
   others are added with the module that makes them buildable (N-295); the table
   itself is not renumbered.
4. **REF 7.1.3 and FR-REF-021 (the Stage 1 seam).** `integration.go` is Stage 2, so
   in Stage 1 a shock of `scripts/` is applied between steps by the world row's
   unexported yield-shock setter, the function `SetYieldShocks` will call, through
   `export_test.go`; the format of `scripts/<scenario>.toml` stays this spec's. The
   visitor (`[[buyer]]`) is not buildable until `AddAgent` and `Direct` exist
   (N-296). **FINAL (coordinator): FR-REF-021 and FR-STB-014 say "only through the integration
   interface or the configuration"; the unexported setter reached from `export_test.go` is the accepted Stage 1 exception (O3). Stage 2 (INT) replaces the seam with the public `SetYieldShocks` and the S01/S04 tests switch over.**
5. **`rules/testing.md` 7.1.2 (with a fresh RULE-TAG).** Add one sentence to the
   hash's definition: it is computed by the test file replaying the step through the
   unexported position methods in the order of the position table, so the hash adds
   no hook to production code, and a guard test compares the replay's final state
   hash and golden bytes with a plain `Run` (D174). The coordinator ruled this
   (Q5); ADR 0007 open point 2 is answered in a note, not edited.
6. **REF 7.2 S01, S04, S19 (a clarification).** S04 also states in one sentence that the elasticity is reported signed (negative for a shortage) and the band is checked on its absolute value. S01 leg (b) and the visitor are
   marked "waits for INT"; S04 and S01 (a) shock through the Stage 1 seam; S19 states
   the Stage 1 measures of D175. No requirement is renumbered.
7. **`doc/documentation_index.md` and `doc/roadmap-v1.md`.** The index has the row of
   this plan (added when the plan was written); the roadmap row links it. No other
   change beyond the corrections of deliverable 2.
8. **Notes.** The dispositions below are written into `doc/notes.md` in this change.
   The notes **N-294 to N-300** (below) already exist there: this phase **verifies
   and amends** them (N-295 to the conditional `three` form, N-296 and N-298 to drop "owner confirms" and "owner-confirmable" for "ruled by the coordinator, reversible") (it adds none), re-quotes the tag it reads, and gives the file a
   fresh tag if it edits it.
9. **Sweep (acceptance check).** A text search of `doc/` finds no `golden/events-`,
   `golden/one.txt` or `golden/five.txt` outside `golden/off/` and `golden/all/`; `golden/stylised/` and `golden/experiment-one.txt` are deliberate exceptions (tied to no module set; testing.md 7.1.1 and 10.4 do not list them)
   (the places are REF 17 and 886, REF 2095 (the "golden files" row of 4.1, which writes `testdata/golden/` with `one.txt` and `five.txt`: correct it to `golden/off/`), OBS 9.5.3 and `spec-observability.md` line 2830
   where the file lives in code, FR-OBS-081, the roadmap).

**New notes (written in this change, tags in brackets):**

- **N-294** (CHK, REF, OBS, CODE): the checkpoint change of Stage 2 adds REF 8.5
  item 6 (the digest of the checkpoint at tick `T`) to `testdata/golden/off/one.txt`
  and moves it once, with the testing.md 7.2 explanation; the checkpoint is a core
  file and not a module, so architecture.md 2.10.5 does not apply to it. The file of
  Stage 1 ends with item 5 and reserves no line.
- **N-295** (REF, CODE): the packs of REF 7.5 and 8.2 the slice cannot run
  (`five`, `famine` with the estates module part, `one-demand`; `three` only if
  Phase 5's probe fails, J1; Phase 1 writes the conditional form and Phase 5 settles
  it) and the
  stage that adds each; the invariants and determinism tests are table-driven so a
  row is added, not a test rewritten.
- **N-296** (INT, REF, STB): S01 leg (b) and every `[[buyer]]` wait for `AddAgent`
  and `Direct` (`integration.go`, Stage 2); the Stage 1 shock seam is the unexported
  setter. The tests that skip carry this note as their ID (testing.md 5.7).
- **N-297** (WLD, CODE): row 11 must keep every position an unexported method of
  `World` and factor the non-position parts of `Step` into unexported methods that
  the position-hash replay calls (D174); a later module's position joins the replay
  by its row in the table. The replay is checked against a plain `Run` by a guard.
- **N-298** (REF): the files `expected/`, `bands/` and `scripts/` of REF 4.1: this
  row creates `bands/one.toml` (keys unset), `scripts/s01.toml` and `scripts/s04.toml`
  and `expected/one.scenarios.txt`; `expected/one.toml` and `expected/second.toml` (the config hash, the
  surplus, pivots, unpriced, warnings) are created and asserted by this row in Phase 6
  (J2, ruled by the coordinator, reversible; plan 1-10 creates none); other packs'
  files are created by the first change that asserts them.
- **N-299** (CFG, REF): the d1 to d5 expectation in `TestPacks_DefectOverlays` flips
  with row 10 (A8); this row adds the rule, "that rule only", the warm layer and the
  `d1` wild-source case (REF 5, "Defects" table).
- **N-300** (SCL, OBS, CODE): the first measurement of a full `Step` of `one`
  (allocations and time per tick), recorded by Phase 8 and Phase 9; it answers the
  measurement N-127 and N-222 ask for at the first slice. Raised empty in Phase 1,
  filled by Phase 8 (the note says "to be measured").

**Notes disposition** (apply, reject or leave open, with what was learned):

| Note | Disposition |
|------|-------------|
| N-016 Keep the parameter registry true (ALL, CFG, WEL) | **Leave open.** This row adds no field, default or row; the registry test and its `notBuilt` list do not change |
| N-017 Words and names (ALL) | **Apply.** The terms this row's documents and test comments use go to the `doc.go` glossary only if a test comment uses one that is not there yet: golden file, per-position hash, scripted run, replay, off-only set. These are test-vocabulary, not domain terms (architecture.md 3.2 governs the code's domain vocabulary); no `doc.go` edit is needed and the glossary guard is checked in the gate. Left open for later stages |
| N-027 No default macro constant is stable (ALL, REF, STY) | **Leave open.** The scenarios record outcomes; they claim no default as stable, and `bands/one.toml` has no value. The recorded `in_sample` seeds are 1 to 10 (REF 8.2) |
| N-030 Five regions with different climates (CAL, PRO, SCL, STY, REF) | **Leave open.** `five` is Stage 2; Phase 8 does not touch it |
| N-089 Model scripts of the demand, spawner and reference-chain specs (DEM, SPN, REF, CODE) | **Leave open.** The scripts under `/tmp/economy-boss/ref/` are not archived by this row; the golden is a Go output and not the prototype's. Learned: FR-REF-007 compares the warm tables with the Go solver, not with the scripts |
| N-092 Overlays and scenarios the wave 2 specs hand to REF (REF, STB, STY) | **Leave open.** None of those overlays is a Stage 1 scenario; Phase 7 uses only `nofloor`, `glut-prepared`, `overhang` and the pack `one` |
| N-114 The open layer of the steady state (VIA, REF) | **Leave open.** The external overlay is Stage 2 |
| N-123 A cold start is a lockstep (CFG, VIA, REF, STY) | **Leave open.** The cold `one` is a reported leg of S18, which is not in this row's list; the cold run is used in Phase 3 only as a determinism control |
| N-135 The hold-out set is a recommendation (REF, STY, CLB) | **Leave open.** This row runs seeds 1 to 10 and records `in_sample` in the report file; seeds 51 to 60 are not run |
| N-145 Profit with no outlet limits hired employment (DEM, STY, REF, SAV) | **Leave open.** No employer role in Stage 1 |
| N-146 A steady state with employers is not a zero-profit state (VIA, REF) | **Leave open.** Not this row's |
| N-168 A lender with no other income eats its principal (CRD, REF, STY) | **Leave open.** No credit in the slice |
| N-028 Can the target-income rule make a shortage worse? (WEL, STB) | **Leave open, with what this row learned.** S04 is built in this row and runs the pack with the floor on and with `overlay/nofloor` (`min_work_share` 0); its outcome is recorded (D181). N-028 closes when the owner reads that record; the note says so |
| N-039 A hard cost floor (STB, GDS, PRO, SKL) | **Leave open.** S07 is built and records the curve at `cost_floor` 10 000, 9 000, 7 000 (default) and 5 000 |
| N-042 The evidence rule may offset the price (BEL, STB) | **Leave open.** S01 records the long-run offset (the residual of 5.7% in the model); the Python cross-check is not run |
| N-043 Nothing anchors the price level if flows ignore price (AGT, PRO, SPN, BEL, STB) | **Leave open.** S16 (overhang) is built and records the share of ticks pinned in the last year |
| N-045 Dampers that act silently (STB) | **Leave open.** The test of FR-STB-013 (every damper has a counter) is row 11's log and catalogue test; this row's S04 and S16 read `ProductionHalted`, `BeliefsUpdated` and `RoleRow.HeldByFloor` |
| N-067 The production halt is not an anchor and has no counter (AGT, STB) | **Leave open.** S16 and S19 read `ProductionHalted`; S19 runs the halt on and off |
| N-086, N-087 (STB, STY, AGT) | **Leave open.** S24, S26 and S27 are not in the Stage 1 list |
| N-099 A trip is a gap in income (WEL, STB) | **Leave open.** No trips |
| N-122 First-come caps (SPN, MIG, MON, MER, STB) | **Leave open.** S73 is WLD's |
| N-124 A paired comparison is exact only while aligned (EXP, STB, CLB, WLD) | **Leave open.** The determinism test compares same-seed runs; a different seed is expected to differ (FR-REF-030). Learned: under ADR 0008 a run split in any number of `Run` calls draws the same words (FR-WLD-063) |
| N-151, N-154, N-162, N-169, N-179, N-203, N-210, N-211, N-214 (STB) | **Leave open**, none read as touched: the clauses concern modules not in the slice or edits of other owners. N-214: this row's site tests are none; the golden pins the keyed draws by their output |
| N-125 A snapshot is a function of saved state (CHK, LED, WLD, CLR, OBS) | **Leave open.** The renderer reads the snapshot only; the golden would show a field that is not a function of saved state at the Stage 2 restore. Learned: item 5 of REF 8.5 (the final state) is read from `Snapshot`, so a field added later changes the file only if it is non-zero (the zero rule) |
| N-127 Event classes at the observer (OBS, EXP, CODE) | **Leave open.** Phase 8 measures the allocation of a `Step` with the default observer; it does not measure the allocation per event the note asks for. Learned: the baseline constant is the measurement the note asks to be taken at the first slice; the number is recorded in the test and in O4 |
| N-130 What IND cannot measure exactly (OBS, IND, WEL, ...) | **Leave open.** S19 uses `MeanRatioBP`, not `HouseholdIncome` (D175) |
| N-242 What the world row calls for the production observability (PRO, WLD, OBS, CLR) | **Leave open.** The golden would show it: `RecipeRefused` and `MarketCleared.CostAnchor` appear in `events-one.txt` once row 11 emits them; a row missing there is a finding on row 11, reported in Phase 3 |
| N-243 Follow-up of the Codacy gate (CODE, PRO) | **Apply.** This row's PRs pass `lizard -l go -C 10 -a 8 -L 100 -w` on the new files (test files are excluded by `.codacy.yml`, but the fixtures are not code) and use the spaced separator row in every Markdown table; `production_report.go` is still owed to `architecture.md` 2.6 and `AGENTS.md` and is named in Phase 9 |
| N-224 Stage 0 guards and what they pin (CODE) | **Apply.** The pinned hash of pack `one`, `testdata/golden/off/` and `TestPacks_DefectOverlays` are the guards this row's PRs read and update; none is loosened. The "Handoff to Stage 1" bullet "the world-dependent observability tests" is split by D170 |
| N-217 D5 binds every module's change (ALL, CODE) | **Apply.** This row creates `testdata/golden/off/`, the files that every later module's PR must leave unchanged; the exit gate checks the pinned hash and records the files' digests in the plan |
| N-126 A new event or `Warn` adds a row (ALL) | **Leave open.** This row adds no event or `Warn`; the golden would show an event with no row only through row 11's catalogue test |
| N-103 Running several spec writers in parallel (ALL) | **Leave open.** Process note; this plan was written in parallel with five siblings and reads them only for the contract. The literal `'"'"'` search of `doc/` is run in the gate |
| N-128 A hosted per-agent field is saved once (ALL, CHK) | **Leave open.** Not this row's |
| N-163 The word "treasury" (MON, FIS, ALL) | **Reject for this row.** The word is not used |
| N-235 One combined architecture.md edit for the fan-in and admission touch points (ALL) | **Leave open.** Not this row's; the first Stage 2 module PR |
| N-198 The drawdown is O(T) per agent and tick (SCL, CODE) | **Leave open.** Phase 8's allocation test does not time anything |
| N-222 The cost of a reseed at every draw site (SCL, CODE) | **Leave open.** Learned: the per-tick time of `one` is reported in Phase 9 (the first measurement of the cost model of `spec-scale.md` 6) and N-222's question is answered there by the owner |
| N-215 The reseed rule is the identity for seed 0 and salt 0 (CHK, CODE) | **Leave open.** Seed 0 is not used by any test here; every test names seed 1 or above |
| N-216 A site constant is never a small integer (WLD, CODE) | **Leave open.** No site is added |
| N-218 The Stage 0 field set and the hash rule (CFG, CODE) | **Leave open.** The golden's header carries the configuration hash of pack `one`; `TestPacks_LoadValidateAndHash` pins it, and a change to a pack updates both with the reason |
| N-229, N-230, N-238 to N-241 (CLR, BEL, WLD) | **Leave open.** Row 11's and row 3's; the golden would show their behaviour |
| N-274 The producer of the warm documents is not in the repository (VIA, REF, CODE) | **Leave open, with what this row learned.** It bears on the byte test and writer of FR-REF-007, which this row owns (Phase 6, plan 1-10 D133); it writes no warm document |
| N-286 The step is replayable from a test file; N-297 (WLD, CODE) | **Apply (N-286) and leave open (N-297).** Phase 4 is the replay; learned: the positions inside `beginStep`/`endStep` are recorded as the pseudo-rows `begin` and `end` and a position of a module that is off records no line (D177); N-297's text says "a position with no method fails", to be read as "a position that `tick.go` lists with a call of its own" |
| N-285 The position table test is staged; N-284, N-287 (WLD, CODE) | **Leave open.** Row 11's. Learned: the replay's map reads `notBuiltPositions` and so grows with the stages |
| N-246 The strategy streams and the draw count (AGT, WLD); N-288, N-289 (CHK, WLD, OBS) | **Leave open.** Row 11's. Learned: the draw count may not be exposed, so the determinism test skips it (Phase 3) and compares `HashWorld` of the final worlds instead; the snapshot sections are those of N-289, which is what the renderer reads |
| N-255, N-257, N-258, N-254, N-244, N-245, N-247, N-248 (EST, CAP, CHK, AGT, CODE) | **Leave open.** Estate and per-agent-state notes of rows 7 and 8, not touched: the renderer reads the snapshot only |
| N-275, N-276, N-277, N-278 (VIA, CFG, DEM, CODE); N-282 (REF) | **Leave open.** Row 10's. N-282 (a recipe on several roles' lists divides executions per household) is learned as a possible source of a surprise in the first golden: a surprise there is a finding on the pack or row 10, not a reason to edit the renderer |
| N-267 An agent in a role with no wage counts as earning nothing (STB) | **Leave open.** S19 reads `MeanRatioBP`, which N-267 qualifies for such roles; the S19 report names the roles with no wage |
| The other open notes tagged REF, STB, OBS, CODE and ALL not named above (REF N-092 covered; STB N-032, N-042 covered; OBS N-159, N-242 covered; CODE N-019, N-089, N-127, N-198, N-214 to N-218, N-222, N-224, N-228, N-232, N-243; ALL N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217, N-235) | **Left open, not read as touched** except as stated. Phase 1 re-reads the index for notes raised since and adds a row for each |

**Tests:** none; documents only.

**Observability and rollback:** none, because no code changes. A wrong edit is
reverted like any document change; nothing depends on it until Phase 3.

**Acceptance:** REF, OBS, the roadmap and `testing.md` 7.1.2 carry the corrections
above (the rule file with a fresh tag); every open note tagged REF, STB, OBS, CODE
or ALL at that time has a row; `doc/notes.md` has N-294 to N-300 (verified, amended
where needed) and a fresh tag if edited; the text search of `doc/` finds no stray
shell-quoting sequence and no `golden/` path of the deliverable 9 sweep outside
`off/` and `all/`.

## Phase 2: the test support layer

**Why here:** Phases 3 to 8 share a recorder, a renderer, a pack runner and a
plain-text reader. Built once and unit-tested alone, they make a golden diff mean a
change of behaviour and not a change of the renderer. Nothing here runs a long
world; the pack runner needs row 11's `NewWorld` only for its own small test.

Deliverables (all `_test.go` files, no production code; existing tests unchanged):

- `support_test.go` (package `economysim_test`):
  - `loadPack(t, layers ...string) economysim.Config`: `config.LoadFS(os.DirFS("testdata/chains"), layers...)`, `Validate`, `t.Fatal` on error; and `loadPackBytes(t, extra []byte, layers ...string)`, which reads the layers with `os.ReadFile` and layers `extra` last with `config.Parse` (for the in-memory one-key documents of Phases 6 and 7: `LoadFS` takes one `fs.FS`, so a `MapFS` cannot be layered over `os.DirFS`). Pack constants: `packOne` (`base.toml`, `one.toml`, `warm-one.toml`), `packOneCold` (without the warm layer), `packSecond`.
  - the recording `Observer` and the capturing `slog.Handler` are **row 11's `observe_helpers_test.go`** (its D164: "which row 12 reuses"); this row defines neither, so the external test package has one of each. The names are fixed by plan 1-11 (D164): `recorder` (field `events []Event`, method `Observe`) and `captureHandler`.
  - `runPack(t, cfg, spec runSpec) result`, where `runSpec` has `Seed`, `Ticks`,
    `Drive` (`driveStep`: a `Step` loop, no `RunStarted`/`RunEnded`; or
    `driveRun(chunks ...int64)`: one `Run` call per chunk), `Observe` (attach the
    recorder), `Audit` (`WithAudit()`), `Logger` (nil or a capturing handler),
    `Snapshots` (`snapNone`, `snapSample` = the ticks of REF 8.5, `snapEvery`) and
    `Extra ...economysim.Option`. `result` holds the recorded events (empty if
    `Observe` is false), the snapshots asked for and the final snapshot. **A bare
    run** (`Observe`, `Audit` and `Logger` all off, no events) is the plain baseline
    of Phase 3 and is a value of the same helper, not a second runner. Snapshots are
    taken only between `Step`/`Run` calls (`Snapshot()` returns `ErrInStep` from an
    observer, the guards of plan 1-11 Phase 2), so `snapSample` and `snapEvery` require
    `driveStep`; a `driveRun` case gets the final snapshot only.
  - `plainTOML(path string) []tomlTable` with `tomlTable{name string; array bool;
    kv map[string]string}`, in file order: the grammar of `bands/` and `scripts/`
    only (`[table]`, `[[array]]`, `key = value` with a number, a string or a
    `{}`-free list), no third-party code (A4); a line outside the grammar is a
    `t.Fatal`. A slice and not a map keyed by name, because `[[shock]]` repeats
    (`scripts/s01.toml` has two entries, REF 7.1.3).
  - `updateFlag = flag.Bool("update", false, ...)` and `goldenPath(name)` giving
    `testdata/golden/off/<name>`; `checkGolden(t, name, got []byte)` reads the file,
    compares bytes, on `-update` writes it, and otherwise fails with a message that
    gives the first differing line number and both lines (testing.md 3.3) and says
    "regenerate with -update only after reading testing.md 7.2".
- **The sample at tick `t`** (used by every item below) is the snapshot taken after
  `Step` `t` completes (so `Snapshot.Tick` = `t + 1`) together with the
  `MarketCleared` rows of tick `t`; `T - 1` is 399.
- `render_test.go` (package `economysim_test`), D176:
  - `renderGolden(cfgName string, seed uint64, ticks int64, res result, report
    solveReport) []byte`: REF 8.5 items 1 to 5. The header: the format version `1`;
    the effective-configuration hash (`World.ConfigHash`, 64 hex); the seed and `T`;
    the warnings (rule and path), sorted, only those that fired; the solver report
    (D172). The samples at ticks 0, 1, 2, 5, 10, 20, 50, 100, 200, 300 and `T - 1`:
    per region, per tradable commodity in `Names` order, the published price, its
    age, the volume, the unmatched bid and ask, the best bid and ask (`MarketRow.State`,
    the rows of `MarketCleared`); the units held by agents and by the estate; per
    role the agents, households and unfed households (`RoleRow`), and per region
    the mean money per household rounded down (`MoneyHouseholds / Households`,
    `RegionRow`): `RoleRow` has no money field (OBS 5.3.4), so REF 8.5 item 2's
    "mean money per household" per role cannot be read from the snapshot (FR-REF-027)
    (O9 ruled option (a); FINAL, reversible: FR-REF-027 forbids unexported state in the golden, and a `RoleRow` money field would be an OBS snapshot change owned by row 11, outside the slice); the Phase 1
    amendment of REF 8.5 item 2 follows that ruling; the pool balance and the pending output
    by commodity. Event counts: the count of each event type over the run, and for
    `OrderRejected`, `AgentBankrupted` and `RecipeAbandoned` the count by reason or
    cause; per window of 100 ticks the counts of `TradeExecuted`, `RecipeExecuted`,
    `OrderPlaced` and `UpkeepShortfall`. **These counts exclude `RunStarted` and
    `RunEnded`** (the filter is applied inside `renderGolden`, with its own unit
    test beside that of `renderEvents`), and the golden run of `one.txt` is a `Step`
    loop (`driveStep`), which emits none, so the file is the same however a caller
    later splits `Run`. Flows: the `BoundaryFlow` totals by region,
    reason and commodity, and the construction bucket (`Tick` -1). Final state: the
    agents and households by region and role, the goods and money totals and the
    jobs in progress by recipe. **The off-leaves-no-trace rule** (REF 8.5): a count,
    an amount or a flag at zero is not written, a section or row with nothing in it
    is not written, an event type or reason with a count of 0 is not listed, and a
    warning is listed only when it fired.
  - `renderEvents(res result) []byte`: OBS 9.5.2 and 9.5.3. One line per event,
    `t=<Tick> <Type> <Field>=<value> ...`, fields in declaration order by reflection
    over exported fields; a field at its zero value is omitted **except** `Tick`, the
    type name and every field of a typed-ID type or of `Side` (zero names a region,
    a commodity or an agent); integers and `Money`/`Price`/`Quantity` in decimal;
    a `[32]byte` as the first 8 bytes in lower-case hex; a typed ID as its string ID
    from the names table (`#<n>` out of range, `-` for `NoCommodity`), **except an
    `AgentID`, which is its integer** (OBS 9.5.2; its zero is never omitted, as for
    any typed ID); an empty string (`Reason`, `Cause`, `Kind`) is omitted like any
    zero field and `Side`'s zero (`Bid`) is shown; a slice as
    `[{...} {...}]`, an empty slice omitted; `Names` and `Snapshot` omitted. The file
    `events-one.txt` has the sections of 9.5.3. **Layout of the ticks (fixed here, OBS
    9.5.3 "after the events of construction")**: the events of construction come
    first as a block headed `construction`, holding **every event emitted before the first
    `TickStarted`, except `RunStarted`** (OBS 4.3.3 item 4: the `Tick` -1 `BoundaryFlow`
    bucket, `WorldCreated` and the construction `AgentSpawned` events, which carry `Tick`
    0 and so cannot be told from the first step's spawns by type and `Tick`), in emission
    order; `RunStarted` (it precedes the bucket under `driveRun`) is counted in tick 0's
    tick-line, as below; no `t=-1` line exists in
    the per-tick list and the construction events are **not** counted in tick 0's
    count; then the full rendering of ticks 0 to 4 (events grouped by `Tick`, in
    emission order); one line per tick of the run (tick 0 counts only what its own
    step emitted) `t=<n> events=<count> digest=<16 hex>`, where `RunStarted` and
    `RunEnded` are counted in the tick-line of the `Tick` they carry (the value is
    row 11's, FR-OBS-012; the file records whatever it is; the events golden is one
    `Run(ctx, 400)`, Phase 3); the count of each event type in the order
    of the table of OBS 4.1 (a type with count 0 not listed; the order is a list kept in
`render_test.go`, guarded by a test that parses that table as the catalogue test
does); the final snapshot
    rendered in the same style, a section with no rows not rendered.
- `support_internal_test.go` is not needed: the unexported seam of this row is
  `export_test.go` (Phase 5 and 7), appended, not replaced.

**Tests (unit, the base of the pyramid):** table-driven tests of the renderers on
hand-built values, run with `-short` (they need no world): each rule of OBS 9.5.2
(the zero field omitted, a zero typed ID shown, `Side` shown, a `[32]byte`, a slice
of rows, `#<n>`, `NoCommodity`), the worked example of OBS 9.5.2 (the
`AgentSpawned` line with `StoresLocked` appended: the line is byte-equal with the
field at zero and has `StoresLocked=2` with it), the reflection renderer on an event
type declared in the test, the digest of a tick (a known SHA-256 prefix), `plainTOML`
on the grammar and on a line outside it, `checkGolden` on a temp directory (equal,
different with the line numbers, absent, `-update` writes). A test that the renderer
of the golden lists a warning only when it fired. Fuzz: `FuzzPlainTOML` (never
panics, a document it accepts has the tables it declares), with a seed corpus of
the files of this row (testing.md 7.4: it parses external input, here fixtures).

**Requirement IDs:** FR-OBS-080 (rendering), FR-OBS-091 (appended field,
byte-equality), FR-REF-029 (the rendering content), FR-REF-040 (not touched:
`FuzzLoad` is `config/`'s).

**Observability and rollback:** none, because no library code changes (test code
only). Rollback is reverting the PR; no other file uses it until Phase 3.

**Acceptance:** the unit tests pass with `-short`; `go vet` and `staticcheck` clean on
the test files; the worked example of OBS 9.5.2 is a test.

## Phase 3: the golden of `one`, the events golden and determinism

**Why here:** the golden is the Stage 1 end-to-end proof (testing.md 7.1) and the
reference every later module's "off did not move" check compares with.

Deliverables:

- `golden_test.go` (external, integration/end-to-end):
  - `TestGolden_OneOff` runs pack `one` (the three layers), seed 1, 400 ticks as a
    **`Step` loop** (`driveStep`, `snapSample`: the snapshots after the steps of the
    sample ticks, which only a `Step` loop can give; no lifecycle events) with the
    recorder and `WithAudit` (the audit only checks, it changes no result,
    FR-OBS-023), renders
    `one.txt` with `renderGolden` and `checkGolden`. The solver report in the header
    comes from `economysim.SolveSteadyState(ctx, cfg)` of row 10 (ASSUMED, D172) over the **cold layer set** (`base`, `one`, loaded without `warm-one`), as FR-REF-007 solves "the layered configuration without" the warm start, so the header does not rely on the loaded `Config.WarmStart` (plan 1-10 states that the solve validates a copy with `WarmStart` nil and ignores the loaded table; VIA is silent), the
    fields the plan names being the `Report`'s `SurplusBP`, `LabourShareBP`, `Pivots`
    and `UnpricedCommodities`/`UnpricedWages`; a test-only adapter in the test file
    maps the report to the renderer's `solveReport` so a rename is one edit.
  - `TestGolden_OneEvents` renders `events-one.txt` from a **separate** run: one
    `Run(ctx, 400)` (`driveRun(400)`, seed 1, recorder on, which emits `RunStarted`
    and `RunEnded` and counts them) with the final snapshot only (the first 5 ticks
    in full, the per-tick digests, the counts, the final snapshot), and
    `checkGolden`s it. The two runs cannot be one: `one.txt` needs snapshots at the
    sample ticks and `Snapshot()` is not callable from an observer. The two runs
    must agree on their final snapshot (asserted, a cheap cross-check).
  - The two files are recorded once with `-update` after reading them (the first
    recording is the review point of testing.md 7.2); the PR states which behaviour
    the file records, that no earlier golden existed and the number of lines (about
    150 for `one.txt`, REF 8.5, and a larger events file).
- `determinism_test.go` (external, end-to-end), table-driven, REF 8.2 "Determinism":
  - `TestDeterminism_SameBytes`: pack `one`, seeds 1 and 2, 400 ticks. **What is
    compared depends on what the case can produce** (B-1 of audit round 2). The
    baseline of a seed is a **bare** `driveStep` run (no recorder, no logger, no
    audit: it yields snapshots only) and a **reference** `driveStep` run with the
    recorder (yields snapshots and events). Per case:
    (i) the final snapshot, `World.ConfigHash` and (added by Phase 4, A7) `HashWorld`
    of the final world, against the baseline, for **every** case, observed or not;
    (ii) the snapshot at every tick, for the `driveStep` cases only (the pair
    observed against bare is the FR-OBS-023 case);
    (iii) the events rendering with `RunStarted` and `RunEnded` removed (the filter of
    `renderEvents`, applied to the reference too), for the cases that attach the
    recorder, against the reference;
    (iv) the `one.txt` bytes, only among the `driveStep` cases with the recorder.
    A `driveRun` case has no per-tick snapshots and so compares (i) and (iii) only.
    Cases: (a) a second plain run; (b) `Run` split into one `Run(400)`, `Run(1)`
    then `Run(399)`, seven uneven chunks and 400 `Run(1)` calls (FR-WLD-063; each
    `Run` emits `RunStarted` and `RunEnded`, FR-OBS-012, hence the filter in
    (iii)), each with and without the recorder;
    (c) a logger (`slog.New` over a capturing handler that is `Enabled` at every level
    so the guarded sites run: this is the stricter case), a nil logger, an observer
    that records and a nil observer; (d) `WithAudit` on and off; (e) both observers
    and the logger together, with the observer's speed made irregular by an observer
    that hashes every event (no sleep: testing.md 10.2). Equal `World.ConfigHash` in
    every case (FR-WLD-020, FR-CFG-026 world half: a logger, an observer, the seed and
    `WithAudit` do not move the hash); final snapshots compared with
    `reflect.DeepEqual`. **Added by Phase 4** (it needs the replay, A7): the
    `HashWorld` assertion of (i) through the export wrapper (the skip list covers the
    observation-only fields: the "random state" half of FR-OBS-023), and
    `TestDeterminism_PositionLinesObservedEqualBare`: the per-position hash lines of
    a recorder-attached replay and a bare replay of 30 ticks are equal. This is the
    draw-count check FR-WLD-064 and FR-OBS-023 ask for where row 11 exposes no
    reader (a draw taken by observation changes the state at that site, and the
    PCG is reseeded at each draw site, so the final `HashWorld` alone would only see
    the last site of each position). If row 11 does expose a draw count, it is also
    compared. The residual gap is recorded in the Testing table.
  - `TestDeterminism_SeedsDiffer` (`driveStep`, recorder on): seed 1 and seed 2 give different `one.txt` bytes
    (FR-REF-030), and the cold `one` (no warm layer) differs from the warm one (a
    control that the comparison can see a change).
  - `TestDeterminism_TwoArchitectures` is **not** a test: the golden file is the same
    file on both architectures (REF 8.6, architecture.md 7.4). It is the manual step
    of the gate (O5) until a CI exists.
- Failure messages name the seed, the case and the first differing line
  (testing.md 6.2, 3.3).

**Tests:** end-to-end as above (skipped with `-short`: "end-to-end test, skipped
with -short"). The unit level of the same behaviour is rows 1 to 11's; this row
adds the integration proof (testing.md 2.3). Run against a deliberately broken
version (a map iteration in a position, a `time.Now` in a copy of the repo under the
scratchpad) to show the test fails (testing.md 9.3); the PR states it did.

**Requirement IDs:** FR-REF-029, FR-REF-030, FR-REF-025, FR-REF-026, FR-OBS-023,
FR-OBS-080, FR-OBS-081, FR-WLD-063, FR-WLD-064, FR-WLD-020, FR-CFG-026 (world
half), NFR-WLD-004 (one architecture here; the second is O5).

**Observability and rollback:** none new; the test attaches an observer and a
logger to prove the world is unchanged by them (rules 8.1 "Non-interference"). A
recording that is wrong is regenerated with `-update` only after the diff is
understood (testing.md 7.2); rollback is reverting the PR, which removes the two
files. The files are then the references of every later row.

**Acceptance:** the golden files exist and pass; every case of the determinism
table passes for both seeds; the PR records the line counts and the digest of each
file in the plan's "As built".

## Phase 4: the per-position state hash

**Why here:** when the golden of Phase 3 fails, the hash names the first tick and
position at which the state diverged (testing.md 7.1.2). It needs only a world and
the position table, so it follows the golden.

Deliverables:

- `poshash_internal_test.go` (package `economysim`, D174, D177), which holds **no
  pack, no `config/` import and no golden call** (A3):
  - `hashWorld(w *World) [32]byte`: the reflection walk of D177 over the whole
    `World` (an unexported-field-aware walk using `reflect.Value` reads only, never
    `Interface()` on an unexported field). The skip list is a `var skipFields =
    []string{"logger", "observers", ...}` in the test file with a comment per entry,
    reviewed against row 11's actual field names when it lands: it also covers every
    field that depends on whether an observer, a logger or the audit is attached
    (the `observed` flags, event scratch, diagnostic counters). The fields of
    production code that hold function values (a registered strategy is a struct, so
    it is walked) are skipped by kind.
  - `replayStep(w *World, record func(position string))`: calls the unexported
    `beginStep`, the unexported position methods in the order of the position table
    (the value in `tick.go` that row 11's position-table test reads, less
    `notBuiltPositions`) and `endStep(nil)` (plan 1-11 D166). The positions that
    live inside `beginStep` (P0.1, P0.2a, P0.3: the guards, the end of construction, the
    bucket events, `TickStarted`) and `endStep` (the audit result, `TickEnded`, the
    failed marking, the tick advance, the P8.6 clears) have no method of their own:
    the row-to-method map names them as two **pseudo-rows**, `begin` (P0.1..P0.3)
    and `end` (P8.5, P8.6), which call `beginStep` and `endStep(nil)` and are
    recorded under those labels. **P0.2 is a row of its own**, `p02` (an empty method
    called in the straight line, plan 1-11 D166, O8), so it is recorded after `begin`,
    not before P0.2a as the table lists it: the replay's order is `begin`, then the
    straight line of `tick.go`, then `end`, the pseudo-rows cutting the table where
    their positions sit. `p02` has a line like any position; without its row the
    guard would fail on it. After `end` the stepping flag is clear (D174), which the
    replay test asserts through `Snapshot()` returning no `ErrInStep`. **P8.4 (the audit) is a method of its own,
    `p84Audit() error`** (plan 1-11 Phase 8, D166), so it is a row of its own in the
    map, not part of `end`; the replay calls it, and passes its error to
    `endStep(err)` as `Step` does. The replay **stops and fails** the test (`t.Fatal`
    naming tick and position) if any position method returns an error (a plain
    `Step` would have returned it, D167). `t` is the tick **at `beginStep`**, so the lines of
    `end` carry the tick that has just run, not the advanced one. The guard checks
    only the positions that `tick.go` lists with a call of their own: a position with
    no method in the map, or a method with no row, is a `t.Fatal` naming it. Per D177
    a position whose module guard is not taken records nothing: each map entry of a
    module position carries its guard predicate. The scripted run of the golden has no
    logger, no observer and no audit (it needs no events). The replay guard and the
    observed-equals-bare check attach a recorder to a second replay: a recorder
    changes no state, the skip list covers its fields, and a replay that differs
    from `Step` only by its observer must hash equal line by line. The external
    tests drive a replay step by step through the exported wrapper `ReplayStep(w)`.
  - `positionHashes(w *World, ticks int) []string`: runs `replayStep` for `ticks`
    ticks and returns the lines `t=<tick> <position> <16 hex>`; exported to the
    external tests by a one-line wrapper in `export_test.go`, with `HashWorld`.
- `poshash_test.go` (external, integration), holding every golden test:
  - `TestPositionHash_Golden`: pack `one`, seed 1, ticks 0 to 29 (D178), the lines of
    `positionHashes`; `checkGolden("one.positions.txt")`.
  - `TestPositionHash_ReplayEqualsRun` (the guard of D174): a world stepped 30 ticks by
    `replayStep` and one by `Step` end with the same state hash; and a recorder-attached
    replay driven for 400 ticks through `ReplayStep`, with a snapshot taken between
    replayed steps at the sample ticks, renders the same `one.txt` bytes as the
    golden (compared through `renderGolden`, not re-recorded).
  - `TestPositionHash_NamesTheFirstDivergence`: the helper `firstDivergence(a, b []line)
    (tick int64, position string)` on two recorded sequences, one with a change
    injected at a known tick and position reports that position. The change is made by
    **mutating state inside the `record` callback** of a replay at the chosen (tick,
    position) (a perturbed config would diverge at the first line, `begin` of tick 0,
    not at a chosen position); a second case unit-tests `firstDivergence` on
    hand-made lines (a difference in the tick, in the position, a shorter sequence); and the message of a failing
    golden test calls it (the test of the golden in Phase 3 gains the call: on a
    mismatch of `one.txt`, `TestGolden_OneOff` runs the positions replay and adds
    "first divergence at tick N position P" to its message).
- The golden file `testdata/golden/off/one.positions.txt`.

**Tests:** as above; the walk has unit tests on hand-built structs (an unexported
field is hashed, a pointer cycle terminates, a map is hashed independently of its
insertion order, a skipped field does not change the hash, two equal values hash
equal and a one-bit difference changes it, and a zero-valued field added to a
hand-built struct leaves the hash unchanged: the off-invariance of D177),
table-driven, with `-short`. The replay
tests are integration (skipped with `-short`).

**Requirement IDs:** testing.md 7.1.2 (a rule; no FR). FR-WLD-004 (the order of the
positions: the replay uses the table), FR-WLD-044 (a module off leaves a position
inert: the replay records no line for it and the zero rule keeps its fields out of
the hash, D177), NFR-WLD-004.

**Observability and rollback:** none; no production code. If row 11 does not
provide the unexported methods (D174's requirements), this phase **stops** and the
coordinator is told; the fallback, a hook in production code, is forbidden by
architecture.md 10.2 and is not taken here.

**Acceptance:** the positions file exists and passes; the replay and `Run` agree;
the injected-divergence test names the injected position.

## Phase 5: the invariants over seeds

**Why here:** the invariants are the property tests of the slice (testing.md 6.3)
and the proof that conservation holds on the real pack, not a hand-built one.

Deliverables:

- `invariants_test.go` (external): `TestInvariants_Packs`, table-driven over the
  Stage 1 rows of REF 7.5: `one` 400 ticks seeds 1 to 16; `second` 400 ticks seeds
  1 to 4 (N-295 lists the rows that wait). **J1 probe:** the phase's first task loads `three`
  (`base, three, warm-three`), builds it and runs 400 ticks; if it runs, the table
  gains the row `three` 400 ticks seeds 1 to 4 (FR-REF-028; its regions do not
  trade, REF 16.2) and N-295 is amended; if not, the failure goes into N-295 and
  the PR lists the omission (FR-REF-025). Each case builds the world with
  `WithAudit` and the recorder, steps tick by tick and after **every** tick checks,
  with exact equality (no tolerance, testing.md 6.3):
  1. goods: (i) for each commodity, `sum(HeldByAgents + HeldByEstate)` + the sum of
     the boundary balances of that commodity == 0, with **no** `Pending` and no
     shipment term (D179; `spec-ledger.md` 8.2.2); (ii) FR-OBS-042 per (reason,
     commodity): `Boundary.Balance` == the construction term (D179) + the sum of the
     `BoundaryFlow` events since + `Snapshot.Pending`; also that the boundary reasons that appear are those
     the slice declares;
  2. money: the snapshot header `Money` + the sum of the money boundary balances == 0, (no escrow term in Stage 1: `MoneyContract` is not in the snapshot),
     and no money flow reason appears after construction (the packs have no faucet);
  3. no negative balance or stock: the internal seam (below) scans every ledger
     account and every holding, **skipping boundary accounts**, which are
     legitimately negative on the faucet side (`Ledger.Account(id)` returns a `View`
     with the kind);
  4. no trade has a buyer or seller that does not exist: the set of live agents is
     rebuilt from `AgentSpawned`/`AgentBankrupted`, plus the estate pseudo-agents
     (IDs `0 .. regions-1`);
  5. the clearing price between the matched limits (D180);
  6. identical seed and configuration give identical results: the run is repeated
     for seeds 1 and 2 in Phase 3 and here the first tick's snapshot is compared
     with a second world's, so a nondeterministic first tick fails at once.
  A failure reports the seed, the tick and the invariant (testing.md 6.2).
- `invariants_internal_test.go` (package `economysim`): `negativeAccounts(w *World)
  []string` reading the ledger's accounts through row 1's accessors (the audit's own
  walk); exported to the external test by `export_test.go` as a one-line wrapper
  `NegativeAccounts`. The one place this row reads unexported state, and the reason
  is that the snapshot holds sums and not accounts (D179).
- Closed-world checks that need no extra code: the same test also asserts that the
  tick advanced by one per `Step`, that `Step` returned nil and that no event is
  `OrderRejected` with `overflow`, `holding_limit` or `receipt_limit` reasons (a
  rejection by an arithmetic bound is a bug in the pack, N-235 item 3), reported if
  it occurs.

**Tests:** end-to-end; skipped with `-short`. Unit tests of the helpers: the
conservation identity on a hand-built snapshot (balanced, one unit off, and balanced
with a nonzero `Pending` to show it is excluded from identity (i)), the
event-rebuilt live set, the price-between-limits check on constructed events
(equal to the bid, equal to the ask, one above, one below). Mutation check on a copy
(testing.md 9.3): a copy of the repo with a leak (one unit created without a boundary
reason) must fail check 1, and a copy with a trade at a price one above the bid must
fail check 5.

**Requirement IDs:** FR-REF-028, FR-REF-025, FR-REF-026, FR-STB-008 (no negative
balance and the prices inside bounds at every tick of every scenario: the price
bounds half is Phase 7), FR-OBS-042 (the identity), FR-OBS-015 (the bucket at
`Tick` -1 is part of the identity), FR-LED-035, FR-LED-045, FR-LED-046 (A7's
construction audit: `Audit` passes at tick 0 and every tick), FR-CLR-019.

**Observability and rollback:** none; no production code. A failing seed is
reproducible by one `WithSeed` call (the message says so).

**Acceptance:** all 20 (pack, seed) cases pass over 400 ticks (24 if the `three` row of J1 is added); the two broken copies
fail; the run time is recorded (A6).

## Phase 6: the configuration-only chain on `second` and the defect files

**Why here:** both are pack tests that need the world (X4) or the viability rules
(A2) and no scenario machinery.

Deliverables:

- `config_chain_test.go` (external): `TestConfigOnly_SecondRunsToTheStablePredicate`
  (X4, FR-REF-031, testing.md 6.4): pack `second` (the two layers, loaded from
  `testdata/chains/second/` alone), the world built and run for 1 095 ticks, seeds 1
  to 4. The test names **no commodity, recipe or role of `second`**: it takes every
  identifier from the snapshot's `Names`, so it shows that no Go is specific to the
  pack (architecture.md 9.4). Over the last 365 ticks it evaluates the predicate:
  (a) the invariants of Phase 5 (asserted); (b) every tradable commodity with a
  producer trades in at least `x4.trade_share_bp` of the ticks; (c) the mean volume
  per active market is at least 1 unit a tick; (d) no market is pinned for more than
  `x4.pinned_share_bp` of the ticks; (e) households are not below `x4.households_bp`
  of the start; (f) the price range of each commodity in the window is under
  `x4.range_factor`. **What is asserted:** (a) and (c), because they have no band key
  (the paper's literal number, `spec-beliefs.md` 19); (b), (d), (e), (f) are band keys
  of `bands/second.toml`, unset, so they are **reported** (FR-REF-022), and asserted
  when a value is set. It also builds `second` as a Go `Config` and compares the hash
  with the file's (FR-CFG-004). No Go builder of `second` exists
  (`fileagree_test.go` compares the file with `referenceConfig()`, a Go literal of
  `one` only), so the deliverable is `secondConfig()` in `second_reference_test.go`
  (about 130 lines of TOML to transcribe, with the warm start) and its parity test
  against the loaded files. If (c) fails at the first run the failure is recorded in a
  note and the owner decides; the test is not weakened (O6).
- Extend `config/integration_test.go` `TestPacks_DefectOverlays` (D173, A8) with the
  failing assertions of REF 5 "Defects": `Validate` must fail with
  `errors.Is(err, economysim.ErrNotViable)` for d1 to d5, and, since `Validate`
  returns an `errors.Join` and `errors.As` finds only the first `*ViabilityError`,
  the test **walks `Unwrap() []error`, collects every `Rule` into a sorted set and
  compares the set**, with "that rule only" meaning a one-element set (and no
  `ErrTooManyErrors`); d2 also asserts the `Path` and `Value` of `role_cannot_work`
  (plan 1-10 D124: `Path` `["roles", "forester"]`, `Value` `north`, the first region where none of the role's recipes exists, and `Bound` empty; the `unproducible_in_realm` errors have `Path` `commodities/<id>` and `Value` the commodity ID). The rules are, per defect: d1 `cycle_not_productive`,
  d2 `unproducible_in_realm` (fuel, prepared, refined, staple, tool_a, tool_b) plus `role_cannot_work` for `forester` and no `labour_short` (final, ruled by the coordinator through plan 1-10 O12: REF 5's d2 row stands and plan 1-10 Phase 5 edits the overlay `defects/d2_upkeep_unmade.toml` so that `fuel` is not marked `raw`, because with `raw` a `Raw` good is sourced, VIA 4.2.1), d3 `role_cannot_work`, d4 `labour_short`, d5
  `storage_gap`; d3, d4 and d5 report **that rule only** (the list of errors has one
  element, FR-REF-016); d2 also reports `role_cannot_work` for `forester` (path `roles/forester`, value `north`) and d1
  `unproducible_in_realm` for seed, staple, prepared (the "Also reported" column);
  d6 loads, `Validate` is nil and the warning `cost_floor_at_full_cost` is present. Each is
  checked over the **cold and the warm layer set**, the defect last. A new case:
  `d1_seed_cycle` with the wild source on (`grow_staple` returning 30 with
  `gather_seed` at its yield of 10 000; **not** by dropping the `Yields` row, which
  leaves a sited recipe with no yield row and so keeps the source off, spec-subsistence
  3.2/3.5): built with `config.Parse` over the bytes of `base, one[, warm-one]` (read with `os.ReadFile`; `config.LoadFS` reads every layer from one `fs.FS`, and `os.DirFS` and an `fstest.MapFS` cannot be combined) plus an in-memory byte document that holds only d1's `[recipe.grow_staple]` output, or by
  setting the `gather_seed` yield back to `All = 10000`. Asserts `labour_short` (one error, path `regions`, `Value` the pooled `SurplusBP` as text, about 730 bp = `lambda*` 0.073, `Bound` `10000`, plan 1-10 D124) and that `cycle_not_productive` is absent (REF 5, 6.2). The
  Stage 0 line "`Validate` = nil at Stage 0" is gone (A8).
- `warm_bytes_test.go` (FR-REF-007, plan 1-10 D133): a test-only canonical TOML writer
  under `config/` and a byte-identity test of the warm documents of **`one` and `second`**
  (`warm-one.toml`, `second/warm-second.toml`) and of `two`, `three` and `five`.
  Plan 1-10 Phase 7 loads and solves all three in Stage 1, so the test asserts, per
  pack: `two` and `three` byte-identity of the document with the writer's rendering of
  the table `SolveSteadyState` produces for the layered configuration without it, the
  expected differences of plan 1-10 Phase 7 being already applied there (the seasonal
  job rows of `warm-two` and `warm-three` are hand-edited in 1-10 and the test
  asserts the edited bytes); `five` the same on the `warm-five.toml` that 1-10
  regenerated (the `semi_a`/`semi_d` band lift). A pack that does not load or validate
  in Stage 1 is skipped with `t.Skip` naming the reason (testing.md 5.7); the
  documents of `famine` (estates part) and `one-demand` (demand module) wait with
  their modules (N-295). The writer reproduces the house format of the existing files
  (sorted keys, inline `stock` and `jobs` arrays) so a first run does not rewrite every
  document; 1-10's throwaway generator is not committed, so a first-run difference is
  still possible and the `-update` path handles it. The table is what `SolveSteadyState`
  produces (host architecture; the two-architecture run is a hand run, O5). A first-run
  difference is handled as plan 1-10 D133 and N-274 say: the file is regenerated with the
  testing.md 7.2 explanation, through an `-update` path in `config/` (a second
  `flag.Bool` in the `config_test` binary, run per package: `go test -update ./...`
  fails in packages that do not define the flag), and the pinned hashes of
  `TestPacks_LoadValidateAndHash` / `pinnedReferenceHash` (with `referenceConfig()`, so the file-and-Go agreement test stays green; the warm-one pin does not move on plan 1-10's text, while a regenerated `warm-second` moves the `second` pin in `config/integration_test.go` and `secondConfig()` together) may move in the same PR
  with that explanation. Requirement ID: FR-REF-007 (also in the Requirement IDs
  and the Testing row of this phase).
- `bands/second.toml` (keys unset; Phase 7 creates `bands/one.toml` the same way).
- `testdata/chains/expected/one.toml` and `expected/second.toml` (J2): the config hash,
  the solver report (surplus, pivots, unpriced) and the warnings, recorded with `-update`
  and asserted against `SolveSteadyState` and `Validate`/`Warnings`. The test is
  `TestExpected_Packs` in `config/expected_test.go` (package `config_test`, with the
  `-update` flag of `warm_bytes_test.go`); it reads the file with go-toml
  (REF 8.4's `warnings = [{ rule, path }]` inline tables are not readable by the root
  `plainTOML` helper). **Layer set:** the cold set (`base, one` and `second/second`, no
  warm layer), because that is the set `SolveSteadyState` solves (FR-REF-007) and the set
  whose `cold_start_lockstep` warning FR-REF-005 puts in `expected/`; the warm set's hash
  is a second key of the file and its warning list has no `cold_start_lockstep`. A
  file recorded with `-update` from the code under test proves nothing alone, so the
  test also asserts the REF 5 and 7.3 constants independently of the file: surplus
  25 119 bp and 10 pivots for `one`, 1.966 (the figure plan 1-10 asserts) and 10 pivots for
  `second`. Requirement IDs: FR-REF-005, FR-REF-006, FR-REF-039.

**Tests:** X4 is integration (a 1 095-tick run of four seeds: skipped with `-short`);
the defect assertions are unit-level on documents (no run; they stay under `-short`
as the Stage 0 test's own skip says: it is an integration test because it reads
files, so it keeps the `testing.Short()` skip it has). A deliberately wrong rule name
in a copy must fail.

**Requirement IDs:** FR-REF-007, FR-REF-016, FR-REF-017, FR-REF-031, FR-REF-001 (the documents
load and build with no Go specific), FR-REF-005 (`Validate` of the second setting
returns no error), FR-CFG-004 (file and Go agree), FR-CFG-041 (the `*ViabilityError`),
FR-VIA-004 to FR-VIA-008, FR-VIA-011, FR-STB-017, FR-BEL-038 (A2); A4: FR-VIA-048,
FR-PRO-046, FR-CFG-014, FR-CFG-015.

**Observability and rollback:** none; no production code.

**Acceptance:** the defect table passes for all twelve (defect, layer set) cases plus
the wild-source case; X4 passes its asserted criteria for four seeds and reports the
rest.

## Phase 7: the first stability scenarios (S01, S04, S07, S16, S19)

**Why here:** the roadmap asks that the scenarios that need one region run and their
outcomes be recorded, pass or not; they sit on top of everything above.

Deliverables:

- Fixtures (data): `testdata/chains/bands/one.toml` (a table `[s01.band_bp]`,
  `[s01.recover_ticks]`, `[s01.offset_bp]`, `[s01.unmet_bp]`, `[s04.pinned_ticks]`,
  `[s04.elasticity_max_bp]`, `[s07.freeze_ticks]`, `[s07.spoiled_bp]`,
  `[s16.pinned_share_bp]`, `[s19.ratio_band_bp]`, `[s19.stock_bound_bp]`,
  `[m05.pool_share_bp]`, each with `unit` and **no `value`**, REF 7.1.5);
  `testdata/chains/scripts/s01.toml` (`[[shock]]` `tick = 400`, region `north`,
  recipe `cut_fuel`, `yield_bp = 5000`, and the clear at tick 460) and `s04.toml`
  (`grow_staple` at 5 000 from tick 424, cleared at 516). Each file starts with a
  comment naming REF and the section (testing.md 5.6).
- `scenario_helpers_test.go` (external): the measures of REF 7.1.4 over the recorder
  and snapshots: the price series per (region, commodity, tick) from `MarketCleared`;
  `P0` the mean over a window; the recovery time (first tick after the shock at which
  the ten-tick mean lies within a band of `P0` and stays); the long-run offset against
  the seed price; the unmet need share from `UpkeepShortfall`; the longest pinned
  run (the condition of the "price beliefs at bound" `Warn`, more than half of a
  market's outcomes at a bound, from `BeliefsUpdated`); the grower role's share of
  working ticks; the arc elasticity `rdiv(10 000*(q1-q0)*(p1+p0), (p1-p0)*(q1+q0))`
  in integer arithmetic with half-even rounding (STY-W05; **FINAL ruling (coordinator): the helper reports the SIGNED elasticity (negative for a shortage, which lowers quantity and raises price, per REF S04 and STY-W05) and the band check compares its ABSOLUTE VALUE against `[0, s04.elasticity_max_bp]`**; Phase 1 states this in the REF S04 wording (one sentence, an owed spec edit) and a note for STY/REF records it; computed with `internal/checked.MulDivHalfEven`, which a test at the module root may import; a zero denominator is "absent", and the function is unit-tested); `bandFor(key)` reading `bands/one.toml`
  by `plainTOML` and returning `(value, set)`; `applyScript(t, w, script)` calling the
  Stage 1 shock seam of `export_test.go` (`SetYieldShockForTest(w, region, recipe, bp)`
  over the `set` method of the world's `shocks` field (plan 1-5's `yieldShocks.set`; plan 1-11 keeps the field name) between steps (the seam of O3).
- `scenario_s01_test.go` ... `scenario_s19_test.go` (one file per scenario, end-to-end
  or integration as REF states, each with the `testing.Short()` skip and the doc
  comment of REF 8.3: scenario, pack, seeds, why):
  - **S01** (`one`, 700 ticks, seeds 1 to 10): leg (a), supply: `cut_fuel` at 5 000 bp
    from tick 400 to 459 by the script. Asserts exactly: no price leaves its belief
    bounds (FR-STB-008; **the interval** is FR-BEL-004's, read from `Config` and the
    events only, FR-REF-027: the larger of the reference-band floor and the cost floor
    times the `CostAnchor` of that tick's `MarketCleared` ("in force for the tick"),
    up to the smaller of the ceiling and `MaxPrice`; the anchor of the tick of the
    price is the one used); **and the O check** (FR-REF-023, STY-S01 "S01 sign"): the
    mean `fuel` price over ticks 400 to 459 exceeds `P0` for every seed of leg (a),
    asserted whether or not a band exists, the message naming scenario, pack, seed
    and measure. Measures and reports: the recovery time, the long-run offset
    and the unmet need; each band is asserted only if its key has a value. Leg (b),
    demand (a visitor buys): a subtest that `t.Skip`s with "waits for AddAgent and
    Direct (integration.go, Stage 2), N-296".
  - **S04** (`one` and `one` + `overlay/nofloor`, 1 400 ticks, seeds 1 to 10):
    `grow_staple` at 5 000 from 424 to 515. Asserts exactly: the price stays inside
    its belief bounds at every tick (FR-STB-008); records, for both packs, the
    longest pinned run, the grower share of working ticks and the measured demand
    elasticity (FR-STB-009), and asserts the longest pinned run is at most
    `s04.pinned_ticks` and the absolute value of the signed elasticity in `[0, s04.elasticity_max_bp]`, each only
    when its key has a value (FR-REF-022). The run without the floor is recorded beside it.
  - **S07** (`one` + `overlay/glut-prepared`, `cost_floor` 10 000, 9 000, 7 000, 5 000
    by a one-key document built in memory and layered with `config.Parse` after the pack's bytes (`loadPackBytes`, Phase 2), 400 ticks, seeds 1 to 10): measures the
    share of ticks with a trade in `prepared`, the share spoiled (the `spoilage`
    `BoundaryFlow` over the output of `RecipeExecuted`) and the longest freeze (no
    trade while both sides have unmatched orders). Asserts the bands at the default
    floor when set; records the curve of the four floors.
  - **S16** (`one` + `overlay/overhang`, 1 095 ticks, seeds 1 to 10): the share of
    ticks in the last year in which the `staple` price is at a bound; reported, and
    asserted against `s16.pinned_share_bp` when set (otherwise "reported as F5" is the
    fail text if it were set).
  - **S19** (`one` with the halt on, and with `produce_halt_ratio` 1 000 000 by an
    in-memory document layered with `config.Parse` as for S07, 1 095 ticks, seeds 1 to 10): the household-weighted mean
    welfare ratio per role from `RoleRow.MeanRatioBP` over the last 365 ticks; the
    share of money in the pool `MoneyPool / Money` (the escrow term of REF S19 is 0 in Stage 1: `MoneyContract` is module-owned, appended with EMP and MER);
    the unsold stock per produced good (`HeldByAgents + HeldByEstate`) at the end as
    a share **of the halt threshold** (REF S19 defines `s19.stock_bound_bp` that way;
    only the O check is asserted without a band); `ProductionHalted` counts. Asserts the
    pool share against `m05.pool_share_bp` and the stock bound when set; **asserts
    the O check** that, **per good g**, with the halt off the end stock of g exceeds
    the halt-on run's maximum stock of the same g, for at least one g (the measure can
    see the glut: REF S19 "with the halt off the stock of at least one good grows past
    that bound"); the bound is the halt-on run's maximum, so no band is needed. The aggregate-ratio band is
    reported only (`HouseholdIncome` `OK` false, IND is Stage 2, D175).
- The recorded outcomes: with `-update`, `expected/one.scenarios.txt` is written (D181)
  (one block per scenario: the seeds, the measures per seed and the median, the verdict
  of each exact and O check). The PR that first records it lists the outcomes in its
  description. Nothing compares the file.

**Tests:** end-to-end and integration (skipped with `-short`); the measure helpers
are unit-tested on synthetic series (a series that recovers at a known tick, a
pinned run of known length, the elasticity on two hand-computed points with the
arithmetic in a comment, `bandFor` on a set and an unset key, the script parser).
Every exact check is shown to fail on a broken copy (a price bound check on a
copy that lets a belief exceed its ceiling).

**Requirement IDs:** FR-REF-020, FR-REF-021 (Stage 1 seam, O3), FR-REF-022,
FR-REF-023 (the O checks of S01 and S19), STY-S01 (the O check of leg (a)), FR-REF-024, FR-REF-026, FR-REF-027, FR-REF-035 (S01, S04, S07, S16, S19
covered or listed), FR-REF-037 (the skipped leg carries its ID), FR-REF-039;
S01 FR-STB-018, FR-STB-008, FR-BEL-014, FR-BEL-036, FR-BEL-034, FR-CLR-019,
FR-CLR-020; S04 FR-STB-009, FR-AGT-072, FR-WEL-027, FR-WEL-021 to 023; S07
FR-STB-011, FR-BEL-038, FR-BEL-004, FR-GDS-009, FR-GDS-010; S16 FR-AGT-036,
FR-AGT-070, FR-BEL-004; S19 FR-STB-021, FR-AGT-070, FR-WEL-032, FR-WEL-031,
FR-AGT-066.

**Observability and rollback:** none; no production code. The scenario tests read the
events and the snapshot only (FR-REF-027), with the single named exception of the shock
seam, which writes the slot the way `SetYieldShocks` will.

**Acceptance:** each scenario's exact checks pass; every other measure is printed with
`t.Logf` and written to `expected/one.scenarios.txt`; the PR description lists each
outcome (pass or not) as the roadmap asks.

## Phase 8: the pack halves of the observability tests

**Why here:** these two tests run the reference chain, which row 11 cannot do without
the pack; the world-internal observability tests are row 11's (D170).

Deliverables:

- `cardinality_test.go` (external, integration): over pack `one`, seeds 1 and 2, 400
  ticks, the recorded events: every event field that becomes a metric attribute takes
  only a value present in the configuration or in its closed set (OBS 9.4.1): for
  each typed-ID field `0 <= id < len(Names.X)`; for each `Reason`, `Cause`, `Side`,
  `Kind` field a constant of its closed set. The sets are the constants declared in
  `event.go`, which are string types, so Go reflection cannot enumerate them (`Side` is `type Side int` with `iota` constants,
  which are not `BasicLit`s: the test names `Bid` and `Ask` directly and range-checks
  `0..1`): the
  test parses `event.go` with `go/parser` and walks its `GenDecl`s of `token.CONST`
  (precedent: `ledger_internal_test.go` around line 265; `sourceEvents` in
  `event_test.go` parses the event types and `declaredNames()` returns event type
  names, not constants, so neither gives the constants). `Reason` is one Go type
  holding several closed sets (order rejections, ledger reasons, refusal reasons,
  abandon reasons), so the check is **weaker by design**: the value is a member of
  *any* constant of the field's Go type (`Reason`, `Cause`, `Kind`, `Side`); a
  per-event partition by const block is not attempted, and the test says so. The list of the fields is the
  attribute column of OBS 6.2.1 for the events of the slice: commodity, role, recipe,
  side, region, reason, cause, kind, entry limit. The configuration of `five` that
  FR-REF-034 names waits for Stage 2 (N-295).
- `allocation_test.go` (external, integration; **not parallel**: `testing.AllocsPerRun`
  panics in a parallel test, so this file overrides the "parallel by default" of
  the Decisions section): a full `Step` of pack `one` with the
  default logger and observer allocates no more than a baseline constant held in the
  test (`testing.AllocsPerRun`, NFR-WLD-001, NFR-OBS-002): the world is built, stepped
  to tick 50 (past the construction bucket and the first completions), then
  `AllocsPerRun(20, func(){ w.Step() })`; measured under the flags of the gate (`-race -covermode=atomic -coverpkg=./...`, the logger and observer left at their no-op defaults and `WithAudit` off, the configuration the default `Step` runs in), because allocation counts differ under `-race`; the constant is recorded together with those flags, and the test is skipped with the reason (testing.md 5.7) when it runs under a configuration other than the one measured; it is the first measurement plus
  no slack (it is a **baseline**, "raising it is a reviewed change with a reason",
  rule 8.2), and the test message says so. The benchmark of a `Step` of `one` with `b.ReportAllocs()` and `for b.Loop()` is row 11's (its Phase 10 item 6); this row adds none.
  The per-call-site allocation test (rule 8.2, first bullet) is row 11's.
- The first measurement is recorded in the plan's "As built" and in note N-300 (the
  allocations of a `Step` of `one`, and the per-tick time, answering the question
  N-127 and N-222 ask).

**Tests:** as above; the cardinality test is shown to fail on a copy that emits a
`RecipeExecuted` with a recipe ID out of range.

**Requirement IDs:** FR-OBS-005, FR-OBS-006, FR-REF-034, NFR-WLD-001, NFR-OBS-002.
FR-OBS-090 is not cited: it is row 11's (D164); this phase asserts only the
allocation.

**Observability and rollback:** none; no production code. If the allocation of a
`Step` is above what row 11's plan expects, that is a finding on row 11 reported in
the PR, not a reason to raise the constant silently.

**Acceptance:** both tests pass; the baseline constant and the benchmark result are in
the PR description.

## Phase 9: the exit gate

- The full command is run by hand with an explicit `-timeout 90m` (J3).
- On the whole module: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`,
  `go test -short -race ./...`, the full run with `-race -covermode=atomic
  -coverpkg=./... -coverprofile=coverage.out ./...`, `go mod tidy`; the per-scope coverage
  read and recorded (this row adds no production statement, so any coverage movement
  is the tests reaching row 11's code: the thresholds of testing.md 9.1 are read per
  scope and the root's rise is reported).
- The pinned hash of the all-off reference configuration did not move
  (`TestHashConfig_ReferencePinned`, N-217), except an explained Phase 6 regeneration of a
  warm document, which moves the affected pin and its Go literal together (`pinnedReferenceHash` with `referenceConfig()` for `one`; the `second` pin with `secondConfig()` for `second`) so `fileagree_test.go` stays green; row 10's Phase 7 may regenerate a
  document before this row's writer exists (plan 1-10 D133), and that one is by hand; `testdata/golden/off/` now exists, and
  its three files (`one.txt`, `events-one.txt`, `one.positions.txt`) are recorded with their SHA-256 in the plan's "As built" so that the first
  module PR can check "off did not move" by digest.
- The traceability check of testing.md 4.6 by text search: every ID in a
  `Requirement:` line of the files of this row exists in a spec, and every ID the
  table below lists as covered in full is cited.
- A hand run of the golden on a second architecture is attempted if one is available
  (arm64); if not, O5 stays open and is told to the owner.
- `lizard -l go -C 10 -a 8 -L 100 -w` on the new files; markdownlint is not run (N-243).
- The time of the full suite and of `one` per tick are recorded against NFR-REF-002 and
  `spec-scale.md` 6 (A6).
- The roadmap row set to **done** with its commit and the Stage 1 "Done when" list
  annotated, item by item, with the test that proves it (this row's and the other
  rows'); `production_report.go` is added to `architecture.md` 2.6 and `AGENTS.md` and
  to `doc/documentation_index.md` if no earlier row has (N-243), with a fresh RULE-TAG
  on `architecture.md`.
- Not gating, if time allows: a mutation pass over the new test files is meaningless;
  instead a mutation pass over `clearing.go`'s `Check` and the ledger audit with the
  Phase 5 tests (`gomu` only in Docker, `--cpus=4 --memory=16g --memory-swap=16g`,
  `GOMAXPROCS` and `--workers` 4).

**Tests:** the gate itself. **Observability and rollback:** none; rollback is
reverting the PRs in reverse order; no other row depends on this one except through
`testdata/golden/off/`, which Stage 2 reads.

**Acceptance:** every command is clean; the traceability check finds no ID without a
spec row and no "covered in full" requirement without a test; the Stage 1 "Done when"
list has a test for each item; the roadmap row reads done.

## Cross-plan contract

### Consumes

| Symbol, type, sentinel, spec edit or field | Provided by | Status |
|---|---|---|
| `config.LoadFS`, `Config.Validate`, `Config.Warnings`, `HashConfig`, the packs and overlays of `testdata/chains/` | Stage 0 | Built (023cf97) |
| `WithAudit`, `WithLogger`, `WithObserver`, `WithSeed`, `Observer`, the sealed events, `Names` | Stage 0 (`option.go`, `event.go`, `observer.go`) | Built |
| `economysim.NewWorld(cfg, opts...) (*World, error)`, `(*World).Step() error`, `(*World).Run(ctx, ticks) (int64, error)`, `(*World).Tick()`, `(*World).ConfigHash()`, `(*World).Snapshot()` | Row 11 (`world.go`, `snapshot.go`; `spec-world.md` 3) | **ASSUMED** |
| `Snapshot` with `Tick`, `Seed`, `ConfigHash`, `Names`, `Money`, `Regions[]` (`MoneyHouseholds`, `MoneyPool` only: `MoneyContract` is not in the Stage 1 snapshot, it is module-owned and appended with EMP and MER (coordinator ruling, consistency pass), so S19 and invariant check 2 read `MoneyPool` and the household money alone), `Roles[]` (`Agents`, `Households`, `Unfed`, `MeanRatioBP`, `HasMeanRatio`, `RatioHouseholds`, `HeldByFloor`, `WageObserved`, `Window`), `Markets[]` (`State` with `Price`, `Age`, `Volume`, `UnmatchedBid/Ask`, `BestBid/BestAsk`, `HeldByAgents`, `HeldByEstate`, `Pending`), `Boundary[]`, `Construction[]`, `Pending[]`, `Jobs[]` | Row 11 (`snapshot.go`; `spec-observability.md` 5.3) | **ASSUMED**, names as the spec |
| The events the golden renders, emitted by the world behind the observer flag, including `RecipeRefused`, `MarketCleared.CostAnchor`, `RecipeExecuted.Overflow` | Row 11 (N-242) | **ASSUMED** |
| The position methods, unexported and named after their IDs (`p21Complete`, `p25Work`, ...), and the non-position methods `beginStep` (the guards P0.1, the stepping flag, the end of the construction phase, the bucket events, `TickStarted`) and `endStep(err)` (the audit result, `TickEnded`, marking failed, the tick advance, the clears of P8.6 on success and the clear of the `stepping` flag: NB-1), with `Step` = `beginStep`, the straight line of `tick.go`, `endStep` (the positions inside the two have no method of their own: the replay records them as the pseudo-rows `begin` and `end`, Phase 4); the position list value in `tick.go` (and `notBuiltPositions`) that row 11's position-table test reads | Row 11 ([plan-phase-1-11.md](plan-phase-1-11.md) D166, Provides) | Planned and read 2026-10-17; matches D174 and N-297; the exact method names are matched when row 11 is built (A1); this phase stops if it is not so (O1) |
| The world's unexported field `shocks` (a `yieldShocks`) whose built method `set(region, recipe, bp)` is what `SetYieldShocks` will call, and the field `ledger` (the `*ledger.Ledger` the internal test scans) | Row 11 ([plan-phase-1-11.md](plan-phase-1-11.md) Provides, "Fields row 12 reads"), over row 5 | Planned and read 2026-10-17; the Stage 1 seam (O3) |
| Ledger accessors to scan every account's balance and every holding (`internal/ledger` audit walk) | Row 1, through row 11's `World.ledger` field | **ASSUMED**; only `negativeAccounts` in the internal test reads them |
| `SolveSteadyState(ctx, cfg, opts...) (SteadyState, error)`; `SteadyState.Programmes[0]` (`SurplusBP`, `LabourShareBP`, `HasLabourShare`, `Pivots`) and `SteadyState.Report.UnpricedCommodities` (`[]RegionCommodity`), `UnpricedWages` (`[]string`) | Row 10 ([plan-phase-1-10.md](plan-phase-1-10.md) D128 and its Provides "pure read for the golden header") | Planned and read 2026-10-17; matches D172 |
| `ErrNotViable`, `*ViabilityError` with `Rule`, `Path []string`, `Value`, `Bound` (`Unwrap` to `ErrNotViable`; the rule names are plain strings the tests spell), the five rules `unproducible_in_realm`, `role_cannot_work`, `cycle_not_productive`, `labour_short`, `storage_gap` as errors of `Validate` | Row 10 (`validate.go`, `errors.go`) | Planned and read 2026-10-17 (plan 1-10 D124; it leaves the d1 to d6 file tests to this row); row 10 flips the d1 to d5 assertion of `TestPacks_DefectOverlays` in its own PR (A8) |
| `observe_helpers_test.go`: the recording `Observer` and the capturing `slog.Handler` (`Enabled` at every level) of the external test package | Row 11 (plan 1-11 D164, Phase 10 item 1) | Planned; **reused, not redefined** here |
| The default strategy, spawner and the production halt behaving as the scenarios need; `ProductionHalted`, `BeliefsUpdated`, `UpkeepShortfall`, `RoleRow.MeanRatioBP` emitted or filled | Rows 7, 9, 11 | **ASSUMED** |
| A rename of the **test helper** `RoleRow(t *RoleTable, id RoleID) RoleRowView` in `export_test.go` (line 76, used by `config_only_test.go:65`, `production_pack_test.go:81` and `role_pack_test.go:72`, and by any caller rows 7 to 9 add: every caller is renamed in row 11's PR) to a name that does not clash with the `type RoleRow` that row 11 declares in `snapshot.go` (both are in package `economysim` for tests, so the test build would not compile); this plan reads `Snapshot.Roles []RoleRow` | Row 11's `snapshot.go` PR (plan 1-11 told, N-M of audit round 2) | Owed by row 11; suggested name `RoleTableRow` |
| `ErrInvalidTrade` returned by `Step` on an invalid list | Row 3 / row 11 | Built in `clearing.go` (`Check`); the step's mapping is row 11's |
| `export_test.go` as the append-only seam of the root package | Rows 2 to 6 | Built; this row appends |
| The Stage 1 position table and positions golden hooks of the rules: `testing.md` 7.1.2 (one sentence added in Phase 1, D174) | This row, Phase 1 | Owed here |

### Provides

| What | For | Contract |
|---|---|---|
| `testdata/golden/off/one.txt`, `events-one.txt`, `one.positions.txt` | Every Stage 2 module PR (the "off did not move" proof, testing.md 7.1.1), the checkpoint change (adds item 6 once, N-294) | Recorded with `-update` only; a diff in a module's PR is a non-interference failure and not a golden to regenerate |
| `support_test.go` and `render_test.go` helpers: `loadPack`, `runPack`, the recorder, `checkGolden`, `renderGolden`, `renderEvents`, `plainTOML` | Stage 2 module PRs (their module-alone golden under `testdata/golden/<module>/`, testing.md 7.1.1) | Test code in package `economysim_test`; a module's golden adds a case to the same renderer, its fields obeying the zero rule |
| The position-hash replay (`poshash_internal_test.go`, golden tests in the external `poshash_test.go`) and the positions golden | Every module PR that adds a position; the first divergence reporter for any golden | A new position joins by its row in the table and by one entry (with its guard) of the row-to-method map; a position without a method fails the test. The file does not move when a module is added off (D177: zero-omitting, name-keyed walk; a guarded-off position records no line) |
| The invariants table (`TestInvariants_Packs`) | Stage 2 packs (`five`, `famine`, `one-demand`, and `three` if J1's probe failed) | A row is added per pack; check 1 to 6 are table-independent |
| The determinism table and the "same bytes with and without observation, split into any number of `Run` calls" cases | Every later module and the checkpoint row (a restored run is one more case) | FR-WLD-063 |
| `bands/one.toml`, `scripts/s01.toml`, `scripts/s04.toml`, `expected/one.scenarios.txt` and the scenario helpers | STY (sets the band values from a reviewed run, STY 8.4), EXP, CLB | Every key unset; the report file is a record and is not compared |
| The d1 to d6 file tests and the `second` X4 test | Row 10 and the module rows | Extended, never loosened |
| Notes N-294 to N-300 | Stage 2 owners | Listed in Phase 1 |

## Security review

The module is a library with no network, credentials or persistence, and this row adds
only test files and fixtures. It adds no trust boundary and no production code.

- **Who can reach it.** Only `go test` of a developer or of CI. The fixtures are read from
  `testdata/` by relative path inside the module (testing.md 10.2): no test reads or
  writes outside the module or its `t.TempDir()` except the golden files, and only on
  `-update`, which a developer passes by hand.
- **`-update` as a hazard.** A golden regenerated to make a test pass hides a regression
  (testing.md 7.2). Mitigation: the flag is explicit; `checkGolden` prints the first
  differing line and the instruction; the PR that changes a golden must explain it
  (a review rule, not a technical one); the digests of the files are recorded in the
  plan and the first module PR compares them.
- **Untrusted input.** The plain-text reader and the renderers read repository files,
  not external input. `FuzzPlainTOML` guards it against a panic. The plain parser is
  deliberately small and rejects a line outside its grammar.
- **Resource exhaustion.** The full suite is long (A6). A CI timeout is the
  maintainers'; tests are `-short`-skipped and use no sleep or network. The reflection
  walk of D177 is bounded by the size of the world, run for 30 ticks only, and detects
  cycles (a self-referencing structure would otherwise recurse for ever); its recursion
  depth is limited (a depth over 64 is a `t.Fatal`).
- **Determinism hazards in the tests themselves.** No test iterates a map into an
  output without sorting; the renderer and the walk sort keys; no test reads the clock.
- **Mutation of production state by a test.** The Stage 1 shock seam writes a slot
  through an unexported setter in `export_test.go`, which is compiled only into the
  test binary; no exported symbol is added (architecture.md 10.2).
- **Not applicable:** authentication, authorisation, injection, secrets, cross-tenant
  exposure, supply chain (no dependency is added: the import guard forbids one).

## Compatibility and rollout

- **No code contract exists to break.** This row adds no production identifier, no
  configuration field, event or snapshot field. Nothing exported changes.
- **Document contracts.** Phase 1 edits REF 17, 8.5, 7.1.3, 7.2 and 7.5, OBS 9.5.3 and
  FR-OBS-081, the roadmap (the row citation, the events path, the FR-OBS-024 citation),
  `rules/testing.md` 7.1.2 (with a fresh RULE-TAG) and the notes (a fresh tag); it
  renumbers nothing (testing.md 4.1). An ID that moves keeps its number.
- **What becomes hard to change, and when.** The content of the golden (D172) enters
  every later module PR's "off did not move" check, and Stage 2's checkpoint change
  moves it once (N-294). After the first recording a change of the file is a behaviour
  change with the explanation of testing.md 7.2; before it, the renderer may be
  corrected in a normal PR. The sample ticks and the sections are fixed by REF 8.5.
- **Append-only stays true.** A module's change appends its fields at the end of an
  event or row at its off value; the renderer omits a zero field, so the off files do
  not move (architecture.md 2.10.5; OBS 9.5.2). A module-alone golden is a new
  directory, not an edit of `off/`.
- **Rollout** is one PR per phase (A7). Each leaves the branch passing every command
  of `AGENTS.md`. A phase is rolled back by reverting its PR; later phases depend only
  on earlier ones.
- **Dependency failure.** The row depends on rows 10 and 11. If row 11 does not give the
  unexported positions (D174), Phase 4 stops and the coordinator rules (O1); Phases
  2, 3, 5, 7 and 8 do not depend on it. If row 10 slips, Phase 6's defect half waits and
  Phase 3's header omits the solver report only by a ruling (D172 makes it required).
  If row 11's `Snapshot` lacks a field the renderer needs, the renderer omits that
  item and a note is raised for row 11; the golden is then recorded without it and
  moves once when it arrives.
- **The Go toolchain.** `go.mod` says `go 1.25`. Nothing here depends on a version-specific
  order: no sort of equal elements and no map iteration reaches an output. A toolchain
  upgrade that moves a golden is a behaviour change like any other and is explained in
  its PR (testing.md 7.2).

## Testing (summary, testing.md section 11)

The pyramid for this row is, unusually, mostly its top: the row's job is the
end-to-end and integration proof of the slice, so its unit base is the helpers (the
renderers, the walk, the parsers, the measure functions), tested first and with `-short`.
Everything that runs a world is skipped with `-short` (testing.md 10.5).

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-OBS-080 (rendering rules), FR-OBS-091 (byte-equality of the appended-field example) | FR-REF-029 (the content, Phase 3) |
| 3 | FR-REF-029, FR-REF-030, FR-OBS-081, FR-OBS-023, FR-CFG-026 (world half), FR-WLD-063, FR-WLD-020 | NFR-WLD-004 (one architecture; the second is O5), FR-WLD-064 (the draw count where exposed; otherwise the per-position lines of an observed and a bare replay are compared, Phase 4: the residual gap is a draw that changes no state hash line, which is undetectable without a counter) |
| 4 | testing.md 7.1.2 | FR-WLD-004 (the replay uses the table; row 11 owns the test of the order) |
| 5 | FR-REF-028, FR-OBS-042 | FR-STB-008 (price bounds in Phase 7), FR-LED-035/045/046 (the construction audit; the audit per tick is row 11's) |
| 6 | FR-REF-007, FR-REF-016, FR-REF-017, FR-REF-001 | FR-REF-031 (X4 criteria (a) and (c) only are asserted; (b), (d), (e), (f) are reported until STY sets the band keys); FR-CFG-041 (the format is row 10's; the file half here) |
| 7 | FR-REF-020, 022, 023, 024, 026, 027 | FR-REF-021 (the Stage 1 seam; the integration calls at Stage 2); FR-REF-035 (S01, S04, S07, S16, S19 only; S02, S03, S05, S06, S08 to S14, S17, S18, S20 to S22 are neither tested nor listed in REF 7.6 yet); FR-REF-037 (the skipped S01 leg carries its ID; the other skips come with the other scenarios); FR-REF-039 (`expected/one.scenarios.txt` and, with Phase 6, `expected/one.toml` and `expected/second.toml`; `two`, `three`, `five`, `famine` have none); S01 leg (b) (Stage 2); the bands (STY). The rest of 035, 037 and 039 is tracked in N-295, which Phase 1 extends with the scenarios and the packs' `expected/` files still owed (testing.md 4.5) |
| 8 | FR-OBS-005, FR-OBS-006, FR-REF-034 (over `one`), NFR-WLD-001, NFR-OBS-002 | FR-REF-034 over `five` (Stage 2); NFR-OBS-001 (the call-site half is row 11's) |

Built before and not re-tested: NFR-WLD-005, FR-WLD-060, 067 to 069 and 091 (Stage 0: `random_internal_test.go`, `internal/ordering`; plan 1-11's "owned by other rows" line is corrected to say so); of the WLD requirements only FR-WLD-063 and 064 are this row's (Phase 3); FR-REF-005 and FR-REF-017's Go-built comparison for
`one` (Stage 0), FR-REF-040 (`FuzzLoad` seeds are `config/`'s). Deferred with their code:
FR-REF-032 and X5 (checkpoint),
FR-REF-033 and S10 (`h200` at three scales: needs the scale module, Stage 2),
FR-REF-036 (A-scenarios: row 10 for the document-level ones; the run legs of A1 and A7/A8 are listed in REF 7.6 by Phase 1, N10), FR-REF-038 (the module scenarios).

"Covered in full" means at the lowest level that can verify it (testing.md 2.1). The
second half of testing.md 2.3, the same behaviour through the public API, is what this
row supplies for rows 1 to 11: every test here goes through `NewWorld`, `Step`, `Run` and
`Snapshot`, with the two named exceptions (the position replay and the account scan),
both in internal test files.

Boundaries and error paths: the golden mismatch message, the absent golden, the replay
with a missing method, an unset and a set band key, a script line outside the grammar,
a world that fails (`Step` returns an error: the test reports the seed and stops). Golden
files: three, all new. Fuzz: `FuzzPlainTOML`. Each PR states its coverage and that its tests
were run against a broken version (testing.md 9.3), and that a `Fix:`-less change has
nothing to pin.

## Observability (summary, observability-and-logging.md section 9)

None, because no library code changes: no new failure path, state transition, decision
point or loop (1 to 4), no event, metric, unit or attribute changes (5, 6), neither `Error`
nor `Info` is used (7) and no error is logged and returned (8). The tests of section 8 that
this row owns are: non-interference (Phase 3), cardinality over `one` (Phase 8), the
allocation limit of a full `Step` (Phase 8) and the golden rendering of events (Phases 2
and 3); the others (logs, counterpart, run lifecycle, composition, emission order, the
call-site allocation, defaults) are row 11's, by D170. The tests attach a logger that is
enabled at every level so that the guarded sites run: they check that observation changes
nothing and prove no behaviour of the logs themselves.

## Open items

- **O1 (FINAL, ruled by coordinator, reversible).** Reason: rule 10.2 forbids production back doors and ADR 0007 open point 2 says test-only, so
  the hash needs no hook: the per-position hash replays
  the step through the unexported position methods and adds no hook to production code
  (D174, N-297). It puts requirements on row 11 (every position an unexported method;
  the non-position parts of `Step` factored out). If the owner prefers a test-only hook
  inside `Step`, architecture.md 10.2 and ADR 0007 open point 2 must change first.
- **O2.** The order of rows 10 and 12: row 10's PR flips the Stage 0 line of
  `TestPacks_DefectOverlays` (A8). If it does not, this row's Phase 6 does, and the
  d1 to d5 half then lands in the same PR as the rules' tests.
- **O3.** A Stage 1 exception to FR-REF-021 and FR-STB-014 ("only through the integration
  interface or the configuration"): S01 (a) and S04 apply their shock through the
  unexported setter, via `export_test.go`; S01 (b) waits for `AddAgent` and `Direct`
  (N-296). **FINAL (coordinator):** S01 and S04 are in the Stage 1 "Done when" list and the public setters belong to Stage 2 (INT), so the seam stays; Stage 2 (INT) replaces it with the public `SetYieldShocks` and the tests switch over.
- **O4.** The allocation baseline of a full `Step` of `one` (Phase 8) is a measurement,
  not a requirement; the first value is the baseline, and a row 11 whose `Step`
  allocates far more than the cost model of `spec-scale.md` 6 expects is a finding for row 11.
- **O5.** No CI exists (O3 of plan 0). The two-architecture golden check (architecture.md
  7.4, REF 8.6, NFR-WLD-004) is a hand run on arm64 if one is available; otherwise the
  golden is recorded on one architecture only and says so in the plan's "As built".
- **O6.** X4 criterion (c) (a mean volume of at least one unit per active market) is "a lead
  and not a calibration" (`spec-beliefs.md` 19). If it fails on the first real run of
  `second`, the failure is recorded in a note and the owner decides whether it is a
  finding on the pack, the beliefs row or the criterion; the test is not weakened.
- **O7.** REF 8.5's solver report in the header is required (D172) and depends on row 10
  exposing a pure read of the report; a rename there is one edit in the test adapter.
- **O9 (closed, coordinator ruling).** REF 8.5 item 2 is amended in Phase 1 to the mean
  money per household **of the region** (`MoneyHouseholds / Households`, rounded down),
  so FR-REF-027 holds and no `RoleRow` field is appended. **FINAL, ruled by coordinator, reversible:** FR-REF-027 forbids unexported state in the golden, and a `RoleRow` money field would be an OBS snapshot change owned by row 11, outside the slice; the
  alternative is option (b), a per-role money field on `RoleRow` (OBS 5.3.4, row 11).
- **O10 (closed, coordinator ruling).** This row owns FR-REF-007's byte test and TOML
  writer, as plan 1-10 D133 states (Phase 6). The two-architecture run stays a hand run
  under O5.
- **J1 (closed, coordinator ruling; reversible).** `three`: Phase 5 probes it and
  adds the row (400 ticks, seeds 1 to 4) only if the Stage 1 world runs it; otherwise
  the omission is listed in the PR (FR-REF-025) and N-295 says why.
- **J2 (closed, coordinator ruling; reversible).** Row 12 creates and asserts
  `expected/one.toml` and `expected/second.toml` (config hash, surplus, pivots,
  unpriced, warnings: FR-REF-005, 006, 039) in Phase 6. N-298 in `doc/notes.md` names
  row 12 as owner (corrected with this ruling).
- **J3 (closed, coordinator ruling; reversible).** No edit of `AGENTS.md` or
  testing.md 9.1 now (they change only with CI, O3/O5). The Phase 9 gate runs the full
  command by hand with `-timeout 90m` (about 110 000 ticks at 4 to 6 ms is 8 to 11
  minutes bare, the 90m `-timeout` being an estimate to be measured later; `-race` with `-coverpkg=./...` is roughly 5 to 8 times that). Tests
  that call `testing.AllocsPerRun` are non-parallel (Phase 8), and per-test overhead is
  cut where it costs no coverage (for example the per-tick audit and snapshot run only
  in the invariants and determinism tests, not in scenario runs). **Follow-up
  (FINAL: no rule edit until CI exists):** add the `-timeout` to the `AGENTS.md` and testing.md 9.1
  command in the change that adds CI (with a fresh RULE-TAG on testing.md).
- **O8.** The Stage 1 "Done when" mentions "the registration options reject a name given
  twice and a nil extension": rows 3, 7, 9 own the unit tests and row 11 the world half
  (FR-WLD-002); no phase here repeats them, and the Phase 9 annotation of the list names
  the test of each.
