# Plan: Stage 2, row 2: CHK, `internal/codec` and `checkpoint.go`

**Status: plan written (2026-10-22), not built. Needs plan 1 (the N-214 passages of the CHK and WLD specs rewritten, the plans convention, ADR 0002 accepted, the race-aware gate). Plans 3 to 24 append to what this plan builds.**

Rule tags read (the files show the tags of the contract, no difference): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1.
Notes read (entries with their updates, not the whole file): the open notes tagged CHK (N-125, N-129, N-202, N-204, N-214, N-215, N-225, N-237, N-244, N-247, N-254, N-256, N-265, N-288, N-294, N-309) and ALL (N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217). The disposition of each is in "Risks and open items".
Specs implemented: [spec-checkpoint.md](../specs/spec-checkpoint.md) (CHK) sections 3 to 8, 14, 21 and FR-CHK-001 to 038, 040 to 057, 060, 062 to 064, 070 to 072, 080 to 082 (039, 058 and 059 do not exist; FR-CHK-061 is excluded: a release rule from 1.0 with no Stage 2 owner, nothing to test before it), NFR-CHK-001 to 005 for the sections the slice saves (tags 1 to 11 and 34). Also read: spec-world.md 5, 7, 8.4, 12, ADR 0002, ADR 0008, spec-observability.md 4.4.5, and the state sections of ledger, clearing, beliefs, production, welfare, estates and spawner.
Decision block: **D240 to D259**.
Ships in two PRs without renumbering: PR A = Phases 1 to 4 (write path, nothing restores), PR B = Phases 5 to 8.

## Scope

**Builds (all in the one change; the root is the library's `economysim` package):**

| Item | Home |
| --- | --- |
| The canonical primitives U, I, B, raw, bytes, count, index, with R1 to R13 checks and a sticky error | `internal/codec` (new, stdlib only, imports nothing of the module) |
| The ledger's section: write and read of accounts, lots, boundary balances, claims, pending totals, the construction bucket | `internal/ledger/checkpoint.go` (new file of an existing package, over `internal/codec`) |
| `Checkpoint`, `CheckpointInfo`, `(*World).Checkpoint`, `RestoreWorld`, `Stateful`, the frame, the digest, the section table, the validation order | `checkpoint.go`, continued in `checkpoint_save.go`, `checkpoint_restore.go`, `checkpoint_sections.go` (CHK continuation files, D243) |
| `WithReseed`, `ErrOptionContext` and the context rule of the options | `option.go`, `errors.go` |
| `ErrCheckpointVersion`, `ErrCheckpointCorrupt`, `ErrCheckpointConfig`, `ErrCheckpointExtension` | `errors.go` |
| `WorldRestored` | `event.go` |
| The log keys `format` and `bytes` | `internal/logkeys` |
| The section encoders and decoders of what the slice saves: world (tag 1), ledger (2), agents (3), markets (4), beliefs (5), jobs (6), welfare (7), estate listings (8), spawner (9), skills and regions skeletons (10, 11), extensions (34) | method pairs in `world.go`, `ledger.go`, `agent.go`, `market.go`, `beliefs.go`, `production.go`, `welfare.go`, `estates.go`, `spawner.go` (architecture.md 2.10.1, spec 21) |
| `Directed.SaveState` and `LoadState` (N-247) and the assertion `var _ Stateful = Directed{}` | `strategy.go` |
| Item 6 (the digest) in `testdata/golden/off/one.txt`, moved once (N-294) | `testdata/golden/off/`, `golden_test.go` |
| `testdata/checkpoints/` with the format 1 fixture of pack `one` | new directory |
| The fork-at-N helper every later module plan reuses | `checkpoint_fork_test.go` |

**Leaves out, with the owner:** the sections 12 to 33 (the module that owns each appends its row, D243); the region graph and tag 11's rows (plan 3); SKL's tag 10 rows (Stage 3); the built-in policy `directed`, the policy registry and extension kind 3 (PLC, Stage 3); external agents and their strategy names, `AddAgent`, the shock slots and the section 22 rumours (plan 17); `five`'s golden digest (plan 8 and 24: `five` needs regions); the all-modules fixture of 9.2.1 item 3 (plan 24, D259); the OTel adapter's use of `WorldRestored` (plan 22); EXP's branching harness (plan 20); `Stream` (dropped, D241); the rewrite of the stale passages of the CHK spec (plan 1); streaming, compaction, an in-memory clone and an HMAC (spec 20 questions 2, 7, 9, 10).

## Decisions

Each decision has its source. A decision here only chooses what the spec leaves to the code; where the spec is silent or stale the item is an owed edit in Phase 1 and a line in "Risks".

- **D240. Format 1, one frame.** `checkpointFormat = 1` and the magic `45 53 43 4B` are rule constants in `checkpoint.go`. The frame is magic, format U, hash version U (the value of `hashVersion`, hash.go), hash 32, tick U, seed U (the effective seed), the sections, the trailer SHA-256 of every byte before it (CHK 5.1, 5.4; ADR 0008 for the seed). Sections 23 to 33 are written into format 1 and the version is not bumped by a module (CHK 9.3, last paragraph); the version is bumped only by a change that alters a section already written by a released library. Source: CHK 5.1, 9.1, 9.3.
- **D241. The value and its surface.** `type Checkpoint struct{ b []byte }` (unexported field, immutable); `UnmarshalBinary` copies and checks the frame only (magic, then format with `ErrCheckpointVersion` before the digest, then digest, then header ranges: tick at least 0); `MarshalBinary` and `AppendBinary` copy. `Info()` has no error result. **`Stream` is not built** (CHK 3.1.2 and FR-CHK-006 are stale under ADR 0008; plan 1 rewrites them) and `Info().Seed` is the effective seed. A zero `Checkpoint` is invalid: `RestoreWorld` of it returns `ErrCheckpointCorrupt`. Source: CHK 3.1, 3.3, 6.2 step 1; N-214.
- **D242. `internal/codec`.** `Writer` (append only; `U`, `I`, `B`, `Raw`, `Bytes`, `Count`) and `Reader` (`U`, `I`, `B`, `Raw(n)`, `Bytes`, `Count(minPerElement)`, `Index(n)`, `Strict(prev)`, `Remaining`, `Done`). The reader keeps the first error and returns zero values after it, so a decoder reads straight down and checks `Err()` at the end of a record; the error is `codec.ErrMalformed` wrapped with the byte offset. `Count` enforces R8 (a count is at most the bytes remaining divided by the least bytes an element takes); `Bytes` enforces R5; `U` rejects a non-minimal, an overflowing and a truncated varint (R1), `B` any byte but 0 and 1 (R3), `Index` a value at or above its bound (R6), `Strict` an ID not above the previous (R9). `I` is zigzag (R2). The root maps `ErrMalformed` to `ErrCheckpointCorrupt` with both in the chain (`%w: %w`), adding the tag. No `encoding/binary.Uvarint` for decoding (CHK 5.2.1). Source: CHK 5.2, 13.3, N-075 update.
- **D243. The section table and how a module appends its section.** `checkpointSections` in `checkpoint_sections.go` is a fixed array of `section{tag, present func(*Config) bool, save func(*World, *codec.Writer), load func(*World, *codec.Reader) error}` in tag order; `present` nil means always. `notBuiltSections` names the tags 12 to 33 that have no row yet. The protocol for a module (architecture.md 2.10.1 and 2.10.5): (1) write `saveX` and `loadX` in the module's own root file over `internal/codec`, with the layout of CHK 4.2 for its tag and the sparse and one-spelling rules of 4.1.4 and R10; (2) append one row to `checkpointSections` at its tag position (rows already in the table keep their place; nothing is renumbered); (3) delete its tag from `notBuiltSections`; (4) add its cross-checks to `checkpointCross` in `checkpoint_restore.go` in the step 7 order; (5) add its fork-at-N test (D258), which also covers the golden digest of its module-alone run; (6) add its restore-failure cases (corrupt count, unsorted key, default row). A module that is off has no row in the expected list, so its tag is absent from the frame and an unexpected tag is corrupt (CHK 5.3.1). A test parses the table of CHK 4.2 by plain text and compares tags, names and the "Present when" cells with `checkpointSections` and `notBuiltSections` (the staged form of testing.md 6.8). The continuation files are named in the owed edit to architecture.md 2.6 and AGENTS.md (Phase 1) because Lizard (N-243) forbids one file. Source: architecture.md 2.10.5, CHK 4.2.1, 5.3.
- **D244. Three sections whose owner has no code yet.** Tag 1 is the next agent ID U and the construction flag B only (CHK 4.2 under ADR 0008: no PCG state; the tick and the seed are in the header). The construction bucket and the pending totals are the ledger's and are in tag 2 (D245), as CHK 4.2 and 4.4 say. Tag 10 (SKL) and tag 11 (REG) are always present (CHK 4.2.1) and are written as their **zero-row skeletons**: tag 10 is four counts of 0 (trainees, trainers, queued (region, role) rows, warn-run rows), in the order `spec-skill-barriers.md` 9 lists them; tag 11 is one count of 0 (the edges whose status is not the default). The owner replaces the writer and reader with the real one and the skeleton is a prefix of the real layout, so a world with no rows has the same bytes before and after (the digest does not move when plan 3 or the SKL change lands). The layout of tag 10 is not fixed in CHK 4.2 beyond its list; Owed edit 3 writes the order of the four counts there and is a precondition of Phase 4. Source: CHK 4.2.1, 4.2 tags 10 and 11.
- **D245. The ledger section is written by `internal/ledger`.** `Ledger.WriteState(*codec.Writer)` and `Ledger.ReadState(*codec.Reader) error` on a ledger that `newLedger` built (its boundary accounts already open, so the boundary list is a count the configuration implies). Layout: the non-boundary accounts in ID order (kind, closed flag, and when open: region, owner reference, money, lots as commodity, quantity, `Born` signed, durability, in canonical order), the boundary balances (signed) in account order, next claim ID and the open claims (always; zero claims with the credit module off), the pending totals (flow totals, wear units, as sparse rows; a reason is its pinned code, D246) and **the pending relocation list, always written, count 0 in this plan** (rows `(from, to, thing, total)` sorted by `(from, to, thing)`, money first; plan 3 fills it through `Ledger.AppendRelocations` and `RestoreRelocations` with no change to the bytes of a world that has no row), and, only when the ledger is in construction, the construction bucket in the same layout. The ledger saves the owner reference as an unsigned number it does not interpret; the root checks it per kind (CHK 4.2.2). `ReadState` checks R13 (balance in [0, MaxMoney], quantity in [1, MaxQuantity], reserved 0, lots in canonical order without repeats), rebuilds the expiry hints (`RebuildHints`), sets `cleared` and the per-boundary baseline as the restored balance less the pending totals (spec-ledger.md 9.1), and the tick and construction flag from arguments the root passes. It returns `codec.ErrMalformed` wrapped, never a panic. The ledger section is the only user of the claim layout of CHK 4.2 until the credit module (Stage 3). Source: CHK 4.2 tag 2, 4.4, 6.2 step 10, FR-CHK-044, 046, 047; spec-ledger.md 9.1, 11; N-125.
- **D246. A reason is saved by a pinned code, never by its Go number.** The code is a function `reasonCode`/`reasonFromCode` in `internal/ledger` over the registry table of spec-ledger.md 7.2 (the order of the `BoundaryFlow` rows); a `Guard:` test pins the table `code -> name` (trade 0, initial 1, and so on) so a renumbering of the Go constants fails it. The pinned table is the whole of the table of spec-ledger.md 7.2 in its order, including the rows no module has built yet; `pinnedRows` is the count of that table, fixed by this plan and never changed by a module plan, and a declared reason's code is its row. `TestReasonCode_Pinned` holds the full `code -> name` list. `pinnedRows` is not `NumReasons`. The code of a configured flow reason (a fiscal flow too) is `pinnedRows + flowIndex`, so adding a Reason constant never changes saved bytes; `ReadState` rejects a flow code at or above `pinnedRows + Flows` as corrupt (`codec.ErrMalformed`). Thing is Money or a commodity index (R6). Source: N-225 (the spec is silent on the spelling: owed edit to CHK 4.2 tag 2).
- **D247. The agent section (tag 3) and the hosted fields.** Written for the live agents in ascending ID (a removed agent is a zero slot, `forget` in phase_exit.go, and writes nothing; the gap is the next-ID in tag 1). The table of strategy names first (sorted, written once). Record, in the order of CHK 4.2 tag 3: ID (strictly ascending, at least the region count, below the next ID), account (an index checked against the open Agent accounts), role (R6), weight at least 1, flag word (bit 1 External, bit 2 Mortal; both 0 in the slice), strategy name index, `Unfed`, `Hunger`, `Privation`, `RoleSince` (signed), `DeathCarry` (signed; the agent's slice value), `OwnUse` and `OwnUseJobs` (written as 0), the journey (optional flag, written as absent), the directive (optional, with the optional job request, intents, discards, holds, `Sticky`). **The hosted fields are written always, as R10 says**, so the bytes of tag 3 do not change when SUB, POP or MIG land (N-217): the reader rejects a non-zero `OwnUse` or `OwnUseJobs` and a present journey while the module is off (R10). The modules' plans replace the value source and the module-on check in the same record; they do not add or move a field. The conditional per-agent blocks after the agent list (stores, membership) are `agentTails`, a fixed slice of `{present func(*Config) bool, save, load}` in checkpoint_sections.go, empty in this plan; plans 10, 11 and 14 append to it or to the record. Then the sparse run of consecutive unfed ticks per (region, role) (`worldState.unfedRun`). In the slice an agent's strategy index equals its role, so the name is `strategyName(a)` (checkpoint_save.go, `Roles[role].Strategy`), the one place plan 17 changes. Source: CHK 4.2 tag 3, 4.2.4, 4.5 F3 and F8, R10, FR-CHK-022, 041; N-128, N-217, N-244, N-247.
- **D248. Markets, beliefs, jobs, welfare.** Tag 4: per region then commodity, the report (price, age, volume, unmatched bid, unmatched ask, best bid present and price, best ask present and price), the window as a count and the prices oldest first (`Window.Values`, restored by `Window.Restore`), and `BoundRun`; dense. Tag 5: per live agent in ID order and per commodity of its role's tradable set, `Centre` and `Width`; no count (implied). Tag 6: per live agent: has job B, then `Recipe` (R6), `Remaining`, `Executions`, `Principal` (a live agent ID), `Yield`, the pending quantity of each output in output order; its residues (a count implied by `residueLen`); then the yield shocks that are not 10 000 as (region, recipe, factor), sparse, from `yieldShocks.ordered`. Tag 7: per live agent: records held (at most the role's effective window), each record oldest first as `Income`, `Cost`, `Relief`, `Unearned`, `OwnReceipt`, `OwnHH`, `Working`, `WorkHH`, then the target ratio; then the sparse runs `floorRun` and `clampRun` per (region, role). Restore uses `welfare.FromRecords` (refuses a field beyond the clamp) and checks every other value against the bound its writer's rule gives: price in [1, `MaxPrice`], quantities in [0, `MaxQuantity`], centre in [1, `MaxPrice`], width in [0, 10 000], yield in [0, `MaxYieldBP`], remaining and executions at least 1 for a job that exists. Source: CHK 4.2 tags 4 to 7, R13; FR-CHK-042, FR-BEL-035, FR-PRO-043, FR-WEL-034; N-237 (`Yield` after `Principal`).
- **D249. Estate listings (tag 8).** For each region in ascending `RegionID`: the rows of `estates.rows(ledger, r, dst)` (held lots only, filter-only), each as commodity, born (signed), durability, price. Restore passes them to `estates.restore(ledger, r, rows)` after the ledger section is loaded and wraps `errListingRows` in `ErrCheckpointCorrupt` with the tag. The `asked` mark is scratch and not saved. Source: CHK 4.2 tag 8, FR-CHK-023, FR-EST-024; N-256, N-309 (2).
- **D250. Spawner and runs (tag 9).** The renewal carry per (region, role) sparse (default 0), the `hadAgent` bit per (region, role) as the set bits only, the unfunded run per region, the empty-tick run (`extinctRun`) and the group-below-floor run per (region, role), all from `worldState`; sparse rows are (region, role, value) in ascending key order and a row at the default is corrupt (R10). A cross-check: a role with a live agent in a region has its `hadAgent` bit set (owed to CHK 6.2 step 7, D253; `hadAgent` is the field of `worldState`, the spec's "everHad"). Source: CHK 4.2 tag 9, FR-SPN-043; N-265, N-288.
- **D251. Options and the context rule.** `options` gains `seedGiven`, `reseed`, `reseedGiven`. `WithSeed` sets `seedGiven`; `WithReseed(salt)` sets the two reseed fields (last wins). `applyOptions` is unchanged in its order; `NewWorld` returns `ErrOptionContext` for `reseedGiven` and `RestoreWorld` for `seedGiven`, both right after `applyOptions` (CHK 6.2 step 2: an option error comes before the configuration is looked at). The restored seed is `reseed(header seed, salt)` when given (random.go, built in Stage 0), else the header seed; restore **sets `o.seed` to it before `buildShell`**, so `newStream(o.seed)`, `w.rnd` and `w.state.seed` get the effective seed through the one path construction uses, and no stream state is installed. `WithTargetSpread` is accepted and enters the hash as at construction. Source: CHK 6.4, 6.8; ADR 0008 decision 5; N-215.
- **D252. Restore builds a new world through the same shell.** `newShell(cfg, opts)` (which today calls `applyOptions` first) is split into `newShell(cfg, opts) = applyOptions` then `buildShell(cfg Config, o options) (*World, error)` (world.go); `NewWorld` and `RestoreWorld` each call `applyOptions`, make their context check, then (restore: after setting `o.seed`) `buildShell`, so construction and restore run the same validation, name checks, hash and derived tables. Restore then: checks the hash against the header's with `hashVersion` (an unknown header hash version is `ErrCheckpointCorrupt`, Risks item 4); checks the extensions (step 4); calls `allocate()`; loads the sections in the expected order; attaches the estates from the ledger's Estate accounts (`attachEstates`, one per region, in region order, the first non-boundary accounts); builds the scratch that `builder.finish` builds (`finishScratch`, extracted); runs the cross-checks; sets the tick, the next agent ID, the construction flag and the live counters from the loaded agents; reseeds; audits; calls `LoadState`; reports. It writes nothing of the consumer's until step 11, and a failure drops the half-built world. The agent table is allocated only after the ledger is read and only if `nextAgent - regions` is at most the ledger's account count (every agent ever created has one account, open or closed), so no allocation depends on a number the input chose (CHK 6.6.1). Source: CHK 6.1, 6.2, FR-CHK-016, 017, 018.
- **D253. Cross-checks of step 7 that the slice has.** Every open Agent account is the account of exactly one agent and of the agent that names it, and that agent's region is the account's; the Estate account of region r has owner r and the estates are the first non-boundary accounts in region order (FR-CHK-049); every `Principal` of a job is a live agent; every belief, record, job and run names an existing region, role, recipe or commodity; `hadAgent` is set for every (region, role) with a live agent (D250); the estate listing rows (D249). The checks of later modules are appended by their plans (D243 item 4). Source: CHK 6.2 step 7, FR-CHK-023, 049.
- **D254. The restore audit always runs.** `ledger.Audit` and every check of the modules that are on, whether or not `WithAudit` was given; a failure is `ErrCheckpointCorrupt` wrapping `ErrAuditFailed` (the existing `auditFailure` mapping). For a world restored in construction the saved bucket is the identity's construction term (D245). Source: CHK 6.2.2, FR-CHK-024; spec-world.md 12.2.
- **D255. `Stateful`, `Directed` and the extensions section (tag 34).** `Stateful` is in `checkpoint.go` (two methods, doc comment states the contract of architecture.md 7.7). `Directed` implements it by value receivers: `SaveState` returns the fallback's when the fallback is `Stateful`, else `(0, nil, nil)`; `LoadState` hands the bytes to the fallback or requires them empty (N-247, CHK 7.5). The built-in strategy `directed` (`Directed{Fallback: DefaultStrategy{}}`, option.go) is `Stateful`, so **every world has the entry (kind 0, "directed", version 0, no bytes)**, as FR-CHK-037 says. The world collects the `Stateful` set from the registries (`options.clearings`, `strategies`, `spawners`, plus the built-in) in (kind, name) order, kind 0 strategy, 1 spawner, 2 clearing; kind 3 (policy) has no registry yet and no entry. `Checkpoint` calls `SaveState` once per entry and copies the bytes; a `SaveState` error is `ErrCheckpointExtension` wrapping it. Restore step 4 compares the registered set with the entries by (kind, name) and that each strategy name of tag 3 is registered (joined with `ErrUnknownStrategy`); step 11 calls `LoadState` last. Source: CHK 7, FR-CHK-021 to 038.
- **D256. Events and logs.** `WorldRestored{Tick, ConfigHash, Format uint32, Reseeded bool, Salt uint64, Names, Snapshot}` is delivered first, only when an observer was supplied, with the snapshot built by `Snapshot()` of the restored world at the boundary; a failed restore delivers nothing. Logs: `Warn` per configuration warning, `Info` "world restored" with `tick`, `seed`, `config_hash`, `agents`, `households`, `format`, `bytes`, `Debug` "checkpoint taken" with `tick`, `bytes`, each guarded by `Enabled`. `logkeys.Format` and `logkeys.Bytes` are added (the rules table already lists them). `Checkpoint` emits no event. Source: CHK 6.7, 14; OBS 4.4.5; observability-and-logging.md 3.3.2, 4.3.
- **D257. Item 6 of the golden.** `testdata/golden/off/one.txt` gains one last line `checkpoint_digest <64 hex>`: the trailer of the checkpoint of the golden run at tick 400, seed 1, taken after the run with `WithAudit` and the recorder as today. `events-one.txt` and `one.positions.txt` do not move (a checkpoint emits no event and adds no position). The line is emitted by the shared renderer in `golden_test.go` for every pack and every set (off, module-alone, all) from this plan on; `off/one.txt` is the only existing file that moves, and later module goldens are recorded with the line already present. The move is made once, with `-update`, and the PR gives the testing.md 7.2 explanation: item 6 is new (REF 8.5 item 6, N-294), nothing else in the file changes, and the rest of the file is byte-identical, which the test proves by comparing the old file with the new file minus its last line. The golden header keeps `format 1`. Source: REF 8.5, CHK 8.3, N-294.
- **D258. The fork-at-N pattern.** `forkAtN(t, forkCase)` in `checkpoint_fork_test.go`: `forkCase{Name, Cfg, Opts func() []economysim.Option, Seed, N, M, Before func(*World), Reseed *uint64}`. It builds world A with `Opts()` and a recording observer, applies `Before`, runs N ticks, takes `c := A.Checkpoint()` twice (the two must be byte-equal), restores B with a fresh `Opts()` (so a stateful extension is never shared), and for M further ticks compares after each `Step` the events since the boundary (B's `WorldRestored` skipped and checked on its own), `Snapshot()`, `Tick()`, `Audit()` and `Checkpoint()` bytes of A and B, then restores a third world from B's checkpoint (restore of a restore). It prints the seed, N and the tick of the first difference. Each module plan names its test `TestFork_<Module>` and calls `forkAtN` (no hand-rolled copy) with its overlay for N in {0, inside a job, inside a season, after a call between ticks} (CHK 8.2.1). Source: CHK 8.1, 8.2, FR-CHK-050, 054, 055; roadmap-v1.md Stage 2.
- **D259. Fixtures, fuzz and platforms.** `testdata/checkpoints/format-1-one.bin` is the checkpoint of pack `warm-one`, seed 1, tick 50 (the golden digest of D257 is taken on the cold-start `one`, the fixture on `warm-one`: two different worlds), with the sidecar `format-1-one.txt` holding `tick`, `config_hash`, `digest` and the digest of the world 20 ticks later; `TestFixture_Format1` restores it and continues. It is written only with `-update` (testing.md 7.2). It is the slice fixture; the every-module-on fixture of CHK 9.2.1 item 3 is plan 24's (architecture.md 2.6). Fuzz targets: `FuzzUnmarshalCheckpoint`, `FuzzRestoreWorld` (a small configuration: pack `one` trimmed in the test) and `FuzzCodec` (names do not collide with the existing `FuzzRestore` of estates); every crash is committed with a `Fix:` comment. The digest is compared on amd64 and arm64 by the golden file (architecture.md 7.4); the repository has no CI file, so the two-architecture run is a manual gate (command and fallback in Phase 7). The fixture is re-recorded once by plan 8 when it rebalances `warm-one`; the format version does not change. Source: CHK 9.2.1, FR-CHK-056, 062, NFR-CHK-003; testing.md 7.4, 7.5.

## Phases

### Phase 1. Documents first (no code)

Goal: every statement the code relies on is true. This plan edits no document; the supervisor applies these. Plan 1 owns the N-214 passages; this plan verifies them before Phase 4.

Edits owed (file, section):

1. `doc/specs/spec-checkpoint.md` 3.1/3.1.2 (`Stream` and its example removed), 4.2 tag 1 (next agent ID and construction flag only), 5.1 and 5.4.1 (the header seed is the effective seed and is read back), 6.2 step 8, 6.4 (`WithSeed` row), 6.6.3, 8.4, FR-CHK-006, 025, 051, 17.2 and the size table of 10.2: **plan 1**; verify here. Plan 2-1 D223 and its line 61 read "tag 1 is the next agent ID and the construction flag; the bucket and the pending totals are in tag 2" (D244, D245).
2. `spec-checkpoint.md` 4.2 tag 2: pending totals are written as sparse rows (region, reason code, thing, total) with the reason as the pinned code of D246, wear units as (commodity, units); the claims list is introduced by the next claim ID and a count, both always written; the pending relocation totals are always-present rows `(from, to, thing, total)` sorted by `(from, to, thing)`, count 0 until plan 3 (plan 3 E6 names the content). (N-225, D245, D246.)
3. `spec-checkpoint.md` 4.2 tag 10 and tag 11: the zero-row skeletons of D244 (four counts, one count) and the order of the four counts of tag 10. 4.2.5: dense/sparse for tag 10 stays as listed. This edit is a precondition of Phase 4; plan 2-1 does not own it. Also 4.2 tag 3: no change (D247 follows it).
4. `spec-checkpoint.md` 6.2 step 1: an unknown hash version in the header is `ErrCheckpointCorrupt` (header range). Step 7: add "`hadAgent` is set for every (region, role) with a live agent".
5. `spec-checkpoint.md` 4.2 tag 34 and 7.5: reconcile "`directed`, which `spec-integration.md` 4.2.2 builds in" with the built-in **strategy** `directed` (a `Stateful` `Directed`), which already exists in the slice (option.go); 7.5 says the built-in policy `directed` is not `Stateful` and the built-in strategy is. State both.
6. `spec-reference-chain.md` 8.5 item 6: the line `checkpoint_digest <hex>` and that `one.txt` moves once (D257).
7. `doc.go` glossary (architecture.md 3.2, before any code uses them): checkpoint, restore, format version, section, frame, header, trailer, digest (of a checkpoint), canonical encoding, one spelling, sparse row, pending totals, construction flag, extension state, `Stateful`, entry (of an extension), salt, branch (of a checkpoint), fixture (of a format). The existing `Reseed` entry in `doc.go` already has this meaning and is kept (extend it only if the CHK wording differs); the term is not added twice (`TestGlossary_Vocabulary` fails on a duplicate). Fix the entry "Effective seed" if plan 1 rewords it. Also, for `TestGlossary_DeclaredTypesNamed`, the name `WorldRestored` (`CheckpointInfo` is in checkpoint.go, which the test does not read).
8. `doc/rules/architecture.md` 2.6 and `AGENTS.md` Layout: add the CHK continuation files `checkpoint_save.go checkpoint_restore.go checkpoint_sections.go (CHK)`; in the same edit also the `checkpoints/` line of 2.6 and of `AGENTS.md` Layout: "one fixture of pack one and one all-modules-on fixture per format version"; fresh read-receipt tag on architecture.md. (4.3, 7.7 and 6.2 already carry the Stateful and reseed text.)
9. `doc/notes.md` (supervisor only): proposals in "Risks and open items".
10. `doc/roadmap-v1.md` Stage 2 CHK row: status after the merge.

Exit check: a reader of the specs finds no statement that D240 to D259 contradict.

### Phase 2. `internal/codec`

Files: `internal/codec/codec.go`, `reader.go`, `doc.go`, tests, `testdata/fuzz/FuzzCodec/`.
Tests (unit, table-driven):
- `TestUvarint_Vectors` (`00`, `80 01`, `ac 02`, the ten-byte maximum) - FR-CHK-010.
- `TestUvarint_RejectsNonMinimalOverflowAndTruncated` (`80 00`, `81 80 00`, eleven bytes, a tenth byte above 1, cut input) - FR-CHK-010.
- `TestZigzag_RoundTripAndVectors` (`I(0)=00`, `I(-1)=01`, `I(1)=02`, `I(10000)=a0 9c 01`, `MinInt64`, `MaxInt64`) - FR-CHK-011.
- `TestBool_RejectsOtherBytes`, `TestIndex_BoundAndStrictOrder` - FR-CHK-012, 014.
- `TestCount_AtMostBytesRemaining`, `TestBytes_LengthAtMostRemaining`, `TestReader_StickyErrorAndOffset`, `TestReader_DoneRejectsTrailingBytes` - FR-CHK-013, 014.
- `TestWriterReader_RoundTrip` over 1000 random values of a fixed seed (the test's own source, outside the library) - FR-CHK-015.
- `FuzzCodec`: no panic, an accepted sequence of reads re-encodes to the same bytes (the one-spelling property) - NFR-CHK-003.
- `TestAlloc_CountBoundsTheSlice`: a crafted count of 2^40 allocates nothing - FR-CHK-016.
- `TestImports_StdlibOnly` (the package imports `errors`, `fmt`, `math/bits` only) - NFR-CHK-005.
Exit: coverage at least 90% for `internal/codec`; `gomu` inside Docker (Phase 8) on the package with no surviving non-equivalent mutant in `reader.go`.

### Phase 3. The ledger's section

Files: `internal/ledger/checkpoint.go`, `internal/ledger/checkpoint_test.go`, a seam in `ledger.go` (root) only if the root needs a function the ledger does not export.
Tests:
- `TestState_RoundTripAfterScript`: drive the existing script generators of the ledger tests over 20 seeds, write, read into a fresh ledger of the same configuration, compare `fingerprint` and the audit, write again and compare bytes - FR-CHK-004, 005, 046, 047.
- `TestState_ClosedAccountIsKindAndFlag`: a closed slot is two bytes and keeps its ID - FR-CHK-046.
- `TestState_PendingTotalsAndBaseline`: a call after `ClearTotals` outside construction is saved as pending and the restored audit passes with the baseline less the pending totals; `EndTick` clears it - FR-CHK-024, 044; spec-ledger.md 9.1.
- `TestState_PendingRelocationListAlwaysPresent`: the list is written with count 0, a non-zero count is accepted only as sorted valid rows, and the bytes of a world with no row are the same before and after plan 3 - FR-CHK-044.
- `TestState_ConstructionBucketOnlyInConstruction`: saved while `InConstruction`, absent after `EndConstruction` - FR-CHK-026, 044.
- `TestState_Corrupt` (table): balance above `MaxMoney`, a zero-quantity lot, lots out of canonical order, a repeated key, reserved above 0, a closed account with a balance, an unknown kind, a reason code out of range, trailing bytes - FR-CHK-012, 013, 014, R13.
- `TestReasonCode_Pinned` (Guard) - D246.
- `TestRestore_RebuiltHints` reuses `TestRestore_RebuiltHintsContinueExactly` through the new path - FR-LED-030.
Exit: the ledger package at least 90% coverage; `gomu` on `checkpoint.go` in Docker.

### Phase 4. The root: sentinels, options, frame, `Stateful`, the table, the write path

Precondition: owed edits 2 and 3 are applied (tag 2 and tag 10 layouts).
Files: `checkpoint.go`, `checkpoint_save.go`, `checkpoint_sections.go`, `errors.go`, `option.go`, `event.go`, `strategy.go`, `world.go`, the concept files' `saveX`, `internal/logkeys`.
Tests:
- `TestCheckpoint_FrameLayout`: magic, format 1, hash version, hash, tick, seed, trailer equals SHA-256 of the rest, for a world at ticks 0 and 5 - FR-CHK-009.
- `TestCheckpoint_TwiceEqualAndNoSideEffect`: two checkpoints of one boundary are byte-equal, the state hash, the events and the log of a run with and without a `Checkpoint` call are equal, no event, `Info` or `Warn` - FR-CHK-001, 004, 038, 081; Guard for the random stream.
- `TestCheckpoint_Refusals`: `ErrWorldFailed` (a failed world, matched with `errors.Is`), `ErrInStep` (from an observer of `TickStarted`), a `SaveState` error wrapped in `ErrCheckpointExtension`, success before the first step - FR-CHK-002.
- `TestCheckpoint_NoConfigOrExtensionCodeInside` (the bytes contain neither the configuration's strings nor a registered name other than the entries) - FR-CHK-007.
- `TestSections_AgreeWithSpec` (D243) - FR-CHK-008, 040.
- `TestInfo_ReportsFrame` - FR-CHK-006.
- `TestCheckpoint_ImmutableCopies` (`MarshalBinary` and `UnmarshalBinary` copy; mutate the result) - FR-CHK-003.
- `TestOptions_Context` (table: `WithSeed` on a restore, `WithReseed` on `NewWorld`, `WithReseed(0)` accepted, `WithTargetSpread` accepted) - FR-CHK-020.
- `TestDirected_StatefulDelegates` (fallback Stateful, fallback plain, `LoadState` of non-empty bytes into a plain fallback refused) and `TestStateful_BuiltInDirectedEntry` - FR-CHK-035, 037; N-247.
- `TestExtensions_EntriesSortedAndCopied` (a registered stateful strategy, spawner and clearing; the entry order is kind then name; `SaveState` is called once; the returned slice is copied) - FR-CHK-031, 032, 033.
- `TestSave_Sections` (one per section, over a fixed pack-one world at tick 20): the layout of D244 to D250 against a hand-computed prefix, with the arithmetic in the comment (testing.md 5.5) - FR-CHK-040, 041, 042, 043, 044.
- `TestSave_SparseRowsSkipDefaults` (a run at zero is not a row) - FR-CHK-014.
- `TestSave_HostedFieldsAlwaysWritten`: the bytes of tag 3 hold `DeathCarry`, `OwnUse`, `OwnUseJobs` and the journey flag with every module off, and a restore refuses a non-zero `OwnUse` or a present journey - R10; N-217.
- `TestSave_SizeIndependentOfWeights`: two worlds of one configuration with different household weights have checkpoints of equal length - NFR-CHK-002.
- `TestCheckpoint_StatefulContractFlags`: `Stateful` is optional (a plain strategy gives no entry), is never required of `Observer`, and the version read is passed to `LoadState` unchanged - FR-CHK-030, 034.
- `TestKeys_AreRowsOfTheTable` (rewritten, `internal/logkeys/logkeys_test.go`): no equality of counts; see the Log keys row of the Catalogue appends - observability-and-logging.md 3.3.2.
- `TestSave_AllocationOutsideTheStep`: `Checkpoint` adds nothing to the allocation baseline of `Step` - NFR-CHK-001.
Exit: gates of Phase 8 for the touched files; the write path is complete but nothing restores yet.

### Phase 5. Restore

Files: `checkpoint_restore.go`, the concept files' `loadX`, `world.go` (the `buildShell` split), `construct.go` (`finishScratch` extracted, behaviour unchanged), `estates.go` (`attachEstates`).
Tests:
- `TestRestore_Sentinels` (table, each `errors.Is`): too short, wrong magic, bit flip (digest), a newer format (`ErrCheckpointVersion` with both versions in the message, and before the digest is looked at), another configuration (`ErrCheckpointConfig`, both hashes), an invalid configuration (the `*ValidationError` of `NewWorld`), an unregistered strategy and a missing or surplus `Stateful` entry (`ErrCheckpointExtension`, `ErrUnknownStrategy`), a `LoadState` error - FR-CHK-006, 018, 019, 021, 022, 060.
- `TestRestore_FailureReturnsNilAndEmitsNothing` (an observer and a logger; nothing delivered, no `Info`) - FR-CHK-017, 080.
- `TestRestore_Corrupt` (table, one case per rule R1 to R13 on a real checkpoint re-digested after the edit): a non-minimal varint in a section, a count above the bytes left, an unsorted agent ID, a repeated one, a default sparse row, a bad enum, a quantity above `MaxQuantity`, a missing tag, a surplus tag, a section that ends early and one that runs over, a dangling account owner, an agent account shared by two agents, `hadAgent` unset for a live role, a listing row for a lot the estate does not hold - FR-CHK-008, 012, 013, 014, 023, 049.
- `TestRestore_AuditAlwaysRuns`: a checkpoint whose ledger no longer conserves money (edited and re-digested) is `ErrCheckpointCorrupt` wrapping `ErrAuditFailed` without `WithAudit` - FR-CHK-024.
- `TestRestore_ExtensionOrderAndLoadLast`: `LoadState` is called last, once per entry, in (kind, name) order, and only after every other step succeeded (a failing cross-check leaves the instance untouched) - FR-CHK-029, 034.
- `TestRestore_StatefulHiddenState` (a stateful extension continues identically; a deliberately hidden one diverges and is not detected) - FR-CHK-036; CHK-S1.
- `TestRestore_Reseed`: the 16 vectors of CHK 17.5 through the option, `Reseeded` and `Salt` reported, seed 0 with salt 0 reported as reseeded, two salts differ, one salt twice is identical, a reseeded world's checkpoint carries `seed'` and restores without `WithReseed` to the same streams - FR-CHK-070, 071, 072; CHK-S2.
- `TestRestore_InConstruction`: a world checkpointed before its first `Step` is restored in construction with its bucket and emits the bucket as `BoundaryFlow` with `Tick` -1 at its first step; one checkpointed after the first step is not and emits none - FR-CHK-026.
- `TestRestore_ConfigAndHash` (`Config()` equals the effective configuration and `ConfigHash()` the header's) - FR-CHK-082.
- `TestRestore_SharesNothing` (restore twice, step one, the other unchanged) - FR-CHK-003, 025.
- `TestRestore_TwoReadersAfterFailure`: after a step-11 failure a new restore with new instances succeeds - FR-CHK-017.
Exit: restore of every checkpoint of Phase 4's worlds is byte-equal on re-save (FR-CHK-005) before Phase 6 begins.

### Phase 6. Bit-identity, fork-at-N, canonical fixed point

Files: `checkpoint_fork_test.go`, `checkpoint_test.go` (external), `checkpoint_internal_test.go`.
Tests (end-to-end, `testing.Short()` skip; pack `one`, `warm-one`, seeds 1 to 4 for the matrix and 1 to 16 for the fixed point):
- `TestFork_OneAtN`: N in {0, 1, a tick inside a job, a tick across a season change, 60}, M = 30, through `forkAtN`; the continued `EffectiveYield` of a job restored across a season change equals the original's (N-237) - FR-CHK-050, 052, 053, 026; CHK-S1.
- `TestFork_WithObserversAndLogger`: the restored world with a new observer sees `WorldRestored` first (it precedes every other event; no `AgentSpawned` or `BoundaryFlow` is replayed) and the same later events as the original; a recording `slog` handler on both worlds sees equal `Warn` and `Info` records (message, attributes, order) for the M ticks after the boundary - FR-CHK-011 (as FR-OBS-011), 027, 028, 043.
- `TestFork_SplitRunsWithRestoreInEachGap` (a 40-tick run as `Run(5)` x 8, restoring each time) - FR-CHK-054, FR-WLD-063.
- `TestFork_RestoreOfRestore` - FR-CHK-055.
- `TestFork_DirectiveInAgent`: a directive set through the test seam before the checkpoint (sticky and not) survives; the strategy name table is checked - FR-CHK-041; FR-AGT-055.
- `TestFork_RemovedAgentsLeaveGaps`: a run in which a role loses households (an existing overlay of `testdata/chains/overlay/` that drives exits, chosen in the phase by running them, else a test-built configuration with a low `StartMoney`) restores with the same next ID and zero slots; the agent table, `live` counters and snapshot equal - FR-CHK-046; FR-SPN-052.
- `TestFork_X5` (200 + 200 ticks on `one`, byte-equal at every boundary, REF X5) - FR-CHK-057 for X5; the legs of S60 (own-use, plan 10) and S72 (merchants, plan 15) are owned by those plans through `forkAtN`; skipped when `raceEnabled` and run by gate F (plan 2-1 D229, whose `-run` pattern is widened to include `TestFork`), or reduced to the first fork point under `-race`, because it is slow under `-race`.
- `TestCheckpoint_FixedPoint` (every checkpoint taken in the matrix restores and re-saves to the same bytes) - FR-CHK-005.
- `FuzzUnmarshalCheckpoint`, `FuzzRestoreWorld`: no panic, bounded allocation (`testing.AllocsPerRun` on the crafted-count seeds), every accepted input is a fixed point of restore-then-save - FR-CHK-016, NFR-CHK-003.
- `TestFields_EveryLiveFieldSavedOrDerived` (FR-CHK-045): by `go/parser` over `world.go`, `agent.go`, `market.go`, `production.go`, `welfare.go`, `spawner.go` and `estates.go`, every field of `World`, `agent`, `job`, `worldState`, `marketTable`, `estates` is listed as saved (with its tag) or as derived/scratch (with CHK 4.3's row) in a table in the test; an unnamed field fails. The staged form: later plans add their fields.
- `TestSnapshot_FunctionOfSavedState` (snapshot of a restored world equals the original's at the same tick on 8 seeds) - FR-CHK-053; FR-OBS-043.
- `BenchmarkCheckpoint`, `BenchmarkRestoreWorld` with `b.ReportAllocs()`, reporting bytes, allocations and the closed-slot share - NFR-CHK-004.
Exit: the matrix passes under `-race` (focused run, see gates).

### Phase 7. Golden item 6, the fixture, the digest across architectures

Files: `golden_test.go`, `support_test.go` (`runPack`, line 144, returns the final checkpoint trailer when asked), `testdata/golden/off/one.txt`, `testdata/checkpoints/format-1-one.bin` and `format-1-one.txt`, `checkpoint_fixture_test.go`.
Tests:
- `TestGolden_OneOff` extended: the rendering ends with `checkpoint_digest`; a one-time guard test `TestGolden_OneOffMovedOnlyByItem6` (plan 8 re-pins or retires it when it moves `one.txt`) compares the sha256 of the previous file (embedded in the test) with the sha256 of the new file with its final line removed, the line and its newline both (the remainder ends with the newline of the line before) - FR-CHK-064, FR-REF-029.
- `TestGolden_EventsAndPositionsUnchanged` (their files byte-identical to the committed ones) - testing.md 7.1.1, N-217.
- `TestFixture_Format1`: restore the fixture with `warm-one` seed 1, check its sidecar, continue 20 ticks and compare the digest; `-update` rewrites both with the testing.md 7.2 note - FR-CHK-062, 060.
- `TestDigest_TwoArchitectures`: the digest of the golden run is compared by the file on amd64 and arm64; the arm64 run is manual (D259): the method of plan 2-1 Phase 5: `GOARCH=arm64 go test -c -o <scratch>/ec.arm64.test .`, then `docker run --rm --platform linux/arm64 --cpus=4 --memory=16g --memory-swap=16g -v "$PWD":/w -v "<scratch>":/s -w /w golang:1.25 /s/ec.arm64.test -test.run 'TestGolden_OneOff|TestFixture_Format1'` without `-race`; it needs the binfmt handler (plan 2-1 Phase 5 owner action 4); run once when plan 2-1 Phase 5 has made arm64 available; if it is not available at merge the PR records the item as open and carries it to plan 24, it is not dropped - FR-CHK-015, 056.
- `TestCheckpoint_NoFloatsNoMapsNoTime` (`go/parser` over `checkpoint*.go` and `internal/codec`: no `float`, `map`, `time.`, `rand.`, goroutine) - FR-CHK-015, NFR-CHK-005.
Exit: the PR carries the testing.md 7.2 explanation; the digest is the first move of `testdata/golden/off/` since Stage 1.

### Phase 8. Exit gate

`gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; `go mod tidy` with no diff; `lizard -l go -C 10 -a 8 -L 100 -w` on `checkpoint*.go`, `internal/codec`, `internal/ledger/checkpoint.go` and the `saveX`/`loadX` functions (split a decoder by record kind if it fails, not by hiding logic); coverage at least 90% for the root and `internal/codec`, and `internal/ledger` not lower; `gomu` only in Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4` on `internal/codec` (per file, `reader.go` then `codec.go`) and on `internal/ledger/checkpoint.go`; the registry and catalogue tests with `WorldRestored` removed from `deferredCore` and `notBuiltEvents`, and the sentinel and option rows added; the two-architecture digest run (Phase 7), then gate R and gate F of plan-2-1 D229. Until plan 1 makes the full `go test -race -covermode=atomic -coverpkg=./... ./...` finish, this plan's gate is the short run plus `go test -race -run 'Checkpoint|Restore|Fork|Fixture|Golden' .` and the coverage run without `-race`, stated in the PR.

## Catalogue appends

| File | Appends | Position and id | Fixed by |
| --- | --- | --- | --- |
| `config.go`, `bounds.go`, `defaults.go`, `validate.go` | nothing (CHK adds no field, default or bound) | none | CHK 12 |
| `hash.go` | none (the header uses `hashVersion`; `HashConfig` unchanged) | none | CHK 5.1, FR-CHK-063 |
| `errors.go` | `ErrCheckpointVersion`, `ErrCheckpointCorrupt`, `ErrCheckpointConfig`, `ErrCheckpointExtension`, `ErrOptionContext` | after `ErrSolveInconsistent`, in this order | CHK 13.1, 6.3; N-075 |
| `event.go` | `WorldRestored` (class W, core) with `isEvent`; entries deleted from `deferredCore` and `notBuiltEvents` in `event_test.go` | after the existing set | OBS 4.1 row, 4.4.5 |
| `option.go` | `WithReseed(salt uint64)`, fields `seedGiven`, `reseed`, `reseedGiven` | after `WithSeed` | CHK 6.4, 6.8; architecture.md 6.2 |
| `tick.go` | nothing: `Checkpoint` and `RestoreWorld` are between-ticks calls and add no position | none | CHK 3.2; `positionList` unchanged |
| `snapshot.go` | nothing | none | CHK 14.4 |
| `ledger.go` (root) | nothing; the reason code is the ledger's (D246) | none | N-225 |
| Checkpoint tags | rows for tags 1 to 11 and 34 in `checkpointSections`; tags 12 to 33 in `notBuiltSections` | tag order | CHK 4.2, 4.2.1 |
| Log keys | `format` (`slog.Int64(int64(f))`), `bytes` (`int64`) as `logkeys.Format`, `logkeys.Bytes` | after the last key; `TestKeys_AreRowsOfTheTable` is rewritten here once (Phase 4): every constant is a row of the 3.3.2 table, unique, snake_case, no imports; the 35 Stage 1 keys must each be present (subset check); no total count. Plans 3 to 16 add a constant and a table row and edit no test | observability-and-logging.md 3.3.2 |
| Metrics | none | none | CHK 14.3 |
| Hash field ids | none | none | CHK 12 |

## Cross-plan contract

Consumes:
- spec-checkpoint.md passages of N-214 rewritten (doc/specs/spec-checkpoint.md) <- plan-2-1
- the plans convention and the race-aware gate command (doc/plans/plan-2-1.md) <- plan-2-1
- ADR 0002 accepted (doc/adr/0002-persistence-by-checkpoint.md) <- plan-2-1
- reseed (random.go) <- Stage 0
- ordering.Finalise (internal/ordering) <- Stage 0
- HashConfig, hashVersion (hash.go) <- Stage 0
- newShell, applyOptions, resolveExtensions, worldState, World.guard, auditFailure (world.go) <- Stage 1
- builder.finish, allocate, namesOf, summary (construct.go) <- Stage 1
- estates.rows, estates.restore, errListingRows (estates.go) <- Stage 1
- welfare.FromRecords, Ring.AppendRecords (internal/welfare) <- Stage 1
- beliefs.Window.Values, Window.Restore (internal/beliefs) <- Stage 1
- Ledger.RebuildHints, Ledger.Audit, Ledger.AppendFlows (internal/ledger) <- Stage 1
- yieldShocks.ordered (production.go) <- Stage 1
- Directed, DefaultStrategy, the built-in "directed" name (strategy.go, option.go) <- Stage 1
- runPack (support_test.go), loadValidPack, packOneCommodities, renderGolden, checkGolden (golden and scenario test helpers) <- Stage 1
- logkeys table in observability-and-logging.md 3.3.2 (doc/rules) <- Stage 1

Provides:
- Checkpoint, CheckpointInfo, (*World).Checkpoint, RestoreWorld (checkpoint.go) -> plans 2-3, 2-4, 2-6, 2-7, 2-13, 2-16, 2-17, 2-18, 2-19, 2-20, 2-22, 2-24
- Checkpoint, RestoreWorld (checkpoint.go) -> plan-2-19
- Stateful (checkpoint.go) -> plan-2-17, plan-2-20
- Directed.SaveState, Directed.LoadState (strategy.go) -> plan-2-17
- WithReseed (option.go), ErrOptionContext, ErrCheckpointVersion, ErrCheckpointCorrupt, ErrCheckpointConfig, ErrCheckpointExtension (errors.go) -> plan-2-20
- WorldRestored (event.go) -> plan-2-18, plan-2-22
- codec.Writer, codec.Reader (internal/codec) -> plans 2-3, 2-5, 2-6, 2-9, 2-11 to 2-17
- checkpointSections, notBuiltSections (checkpoint_sections.go) -> plans 2-3, 2-5, 2-6, 2-9 to 2-17
- checkpointCross (checkpoint_restore.go) -> plans 2-3, 2-5, 2-6, 2-9, 2-10, 2-11, 2-12, 2-13, 2-14, 2-15, 2-16, 2-17
- the agent-record hosted fields and `agentTails` (D247: checkpoint_sections.go) -> plan-2-10, plan-2-11, plan-2-14
- strategyName(a) (checkpoint_save.go) -> plan-2-17
- Ledger.WriteState, Ledger.ReadState, with the always-present pending relocation list (count 0, D245) (internal/ledger/checkpoint.go) -> plan-2-3, plan-2-6
- forkAtN, forkCase (checkpoint_fork_test.go) -> plans 2-3 to 2-7, 2-9 to 2-17, 2-20, 2-24
- checkpoint_digest line of testdata/golden/off/one.txt -> plan-2-8, plan-2-24
- testdata/checkpoints/format-1-one.bin -> plan-2-8 (re-recorded with the rebalance), plan-2-24

## Testing (testing.md 11)

| Requirement | Tests (phase) |
| --- | --- |
| FR-CHK-001, 002, 003, 004 | `TestCheckpoint_TwiceEqualAndNoSideEffect`, `_Refusals`, `_ImmutableCopies` (4) |
| FR-CHK-005 | `TestCheckpoint_FixedPoint`, the fuzz targets (6) |
| FR-CHK-030, 034 | `TestCheckpoint_StatefulContractFlags` (4) |
| FR-CHK-043 | the log-equality assertion of `TestFork_WithObserversAndLogger` (6) |
| S77, S78 (CHK) | `TestRestore_StatefulHiddenState`, `TestFork_OneAtN`, `TestCheckpoint_FixedPoint`: REF 7.11 "not built by REF"; they carry no `TestScenarioS<nn>_` name, so plan 24's scan excludes S77 and S78 |
| FR-CHK-057 | `TestFork_X5` (6); S60 and S72 are plans 10 and 15 |
| FR-CHK-061 | excluded: a release rule from 1.0; owner in the plan-2-0 deferred table, nothing to test before it |
| FR-CHK-006, 009, 060 | `TestInfo_ReportsFrame`, `_FrameLayout`, `TestRestore_Sentinels` (4, 5) |
| FR-CHK-007, 008, 040 | `TestCheckpoint_NoConfigOrExtensionCodeInside`, `TestSections_AgreeWithSpec` (4) |
| FR-CHK-010 to 015 | the codec tests of Phase 2, `TestRestore_Corrupt`, `TestCheckpoint_NoFloatsNoMapsNoTime` (2, 5, 7) |
| FR-CHK-016, 017, 018, 019, 080 | fuzz, `TestAlloc_CountBoundsTheSlice`, `TestRestore_FailureReturnsNilAndEmitsNothing`, `_Sentinels` (2, 5, 6) |
| FR-CHK-020, 021, 022, 029 to 038 | `TestOptions_Context`, `TestDirected_*`, `TestStateful_*`, `TestExtensions_*`, `TestRestore_ExtensionOrderAndLoadLast`, `_StatefulHiddenState` (4, 5) |
| FR-CHK-023, 024, 049 | `TestRestore_Corrupt`, `_AuditAlwaysRuns`, `TestState_Corrupt` (3, 5) |
| FR-CHK-025, 026, 082 | `TestRestore_SharesNothing`, `_InConstruction`, `_ConfigAndHash` (5) |
| FR-CHK-027, 028, 081 | `TestFork_WithObserversAndLogger`, the log assertions of Phase 4 (4, 6) |
| FR-CHK-041, 042, 044 to 048 | `TestSave_*`, `TestState_*`, `TestFields_EveryLiveFieldSavedOrDerived` (3, 4, 6) |
| FR-CHK-050 to 055 | `TestFork_*`, `TestSnapshot_FunctionOfSavedState` (6) |
| FR-CHK-056, 062, 064 | `TestDigest_TwoArchitectures`, `TestFixture_Format1`, the golden tests (7) |
| FR-CHK-063 | hash version in the header: `TestCheckpoint_FrameLayout` (4) |
| FR-CHK-070 to 072 | `TestRestore_Reseed` (5) |
| NFR-CHK-001 to 005 | allocation test, `TestSave_SizeIndependentOfWeights` (NFR-CHK-002; the 5 MB figure is reported by the benchmark, a placeholder in the spec), fuzz, benchmark, `TestImports_StdlibOnly`, the no-float test |
| FR-LED-030, FR-WLD-063, FR-OBS-043, FR-AGT-055, FR-BEL-035, FR-PRO-043, FR-WEL-034, FR-SPN-043, 052, FR-EST-024, FR-CLR-034 | the section tests and `TestFork_*` (4, 6) |

- **Goldens.** Off: `one.txt` gains item 6 once (D257), `events-one.txt` and `one.positions.txt` do not move. Module-alone and all-on: none (no module). The hash `TestHashConfig_ReferencePinned` does not move.
- **Fork-at-N.** D258 is the pattern; this plan is its first user (pack `one`). It is the test every later module repeats with its overlay.
- **Registry and catalogue rows.** `ErrOptionContext`, the four sentinels and `WithReseed` get their failing-case tests (testing.md 3.9); `WorldRestored` leaves `deferredCore` and `notBuiltEvents`; the parameter registry has no row (no field).
- **Fuzz.** `FuzzCodec`, `FuzzUnmarshalCheckpoint`, `FuzzRestoreWorld`; corpora under `testdata/fuzz/` of the owning package, committed with `Fix:` comments.
- **Mutation.** `internal/codec` and `internal/ledger/checkpoint.go` in Docker (gomu per file); the equivalent mutants are listed in the PR.
- **Broken-copy checks (testing.md 9.3).** Run the section tests against a copy that drops one field per section, one that writes `BoundRun` as sparse, one that skips the audit, one that calls `LoadState` before the cross-checks; each must fail.
- **Cost.** The e2e tests are slow under `-race` (Stage 1 sign-off exception); they are guarded by `testing.Short()` and the focused run of Phase 8 names them.

## Observability (observability-and-logging.md 9)

1. New failure paths: five sentinels, all returned and none logged (architecture.md 8.3); a failed restore emits nothing. No failure is survived, so no new `Warn`.
2. New state transition: a restore. It has the event `WorldRestored` (class W, once per call) with fixed-point and closed-set fields only (the names table and the snapshot are slices, no maps). `Checkpoint` is a read: no event, one `Debug` line.
3. No new decision point in a strategy, clearing rule or spawner.
4. No new loop in the tick. The restore and the checkpoint are outside it; the only logs in them are the configuration `Warn`s (once each, as at construction) and the two lines above, each guarded by `Enabled`. With the default logger and observer the restore builds no event and no snapshot (NFR-CHK-001, tested by `testing.AllocsPerRun`).
5. No changed event, metric or attribute. `WorldCreated` is untouched.
6. No metric and no attribute.
7. Levels: `Info` for "world restored" (a once-per-call lifecycle line like "world created"), `Debug` for "checkpoint taken", `Warn` for the configuration warnings only.
8. No error is both logged and returned.
9. Tests: `WorldRestored` first and nothing on failure (`TestRestore_FailureReturnsNilAndEmitsNothing`, `TestFork_WithObserversAndLogger`), the log keys against the rules table (the logkeys test), the no-op path allocations (Phase 4), the snapshot of `WorldRestored` equal to the original's (Phase 6).

## Risks and open items

| # | Item | Owner |
| --- | --- | --- |
| 1 | The N-214 passages of CHK (tag 1, 5.1, 5.4.1, 6.2 step 8, 6.4, 8.4, FR-CHK-006, 025, 051, 17.2, 10.2) and ADR 0002 must be corrected before Phase 4; this plan builds without `Stream` and with the header seed read back. | plan 1 |
| 2 | Tags 10 and 11 are written as zero-row skeletons (D244). The spec says only "empty"; SKL and REG must keep the skeleton as the prefix of their layout. Owed edit 3. | supervisor, CHK |
| 3 | The spelling of a reason in the pending totals (N-225) is chosen as a pinned code (D246); the spec is silent. Owed edit 2. | supervisor, LED/CHK |
| 4 | An unknown hash version in a header: the spec says ranges are corrupt; a future encoder is then a format change. Chosen: `ErrCheckpointCorrupt` (owed edit 4). If the owner prefers `ErrCheckpointVersion`, one line changes. | supervisor, CHK |
| 5 | The strategy table of tag 3 assumes `agent.strategy == role` in the slice; plan 17 adds external agents and the registered names. The writer takes the name through one function (`strategyName(a)`) so INT changes one place. | plan 17 |
| 6 | `testdata/checkpoints/` is meant to hold an every-module-on fixture per format (architecture.md 2.6, CHK 9.2.1); no module exists yet, so this plan commits a slice fixture on `one`. | plan 24 |
| 7 | The digest depends on pack `one`'s state, which collapses (N-312, N-314). Plan 8 rebalances the pack and must move `one.txt` (item 6 and the rest) with its own testing.md 7.2 explanation, re-pin or retire `TestGolden_OneOffMovedOnlyByItem6`, and regenerate `format-1-one.bin` and `.txt` in its Phase 6 (it owns that refresh). `five`'s digest (CHK 8.3) waits for regions. | plan 8, plan 24 |
| 8 | The golden line label `checkpoint_digest` is mine (REF 8.5 gives no label). Owed edit 6. | supervisor, REF |
| 9 | The built-in `directed` is both a strategy name and a policy in the specs (CHK 4.2 tag 34, 7.5, INT 4.2.2). The slice has the strategy only. Owed edit 5. | supervisor, CHK/INT |
| 10 | EXP 7.3 and FR-EXP-062 still describe a stream fingerprint read from `Stream`; `Stream` is not built (D241). | plan 20 |
| 11 | The slice has no `Policy` registry, so extension kind 3 has no entry; PLC adds it with its registry (kind 3 stays reserved). | PLC (Stage 3) |
| 12 | Under `-race` the matrix is slow; the focused run is the gate . If the matrix does not fit, N is reduced to {0, 1, in-job, season change} and 3 seeds; the PR says so. | this plan, plan 1 |
| 13 | New file names `checkpoint_save.go`, `checkpoint_restore.go`, `checkpoint_sections.go` are not in architecture.md 2.6 yet (Phase 1 item 8). The change must not land before they are. | supervisor |
| 15 | Plan 2-1 D223 already places the bucket and the pending totals in tag 2 (CHK 4.2, 4.4), as D244 and D245 do; no conflict remains. | plan 1 |
| 14 | A decoder that reads a number the input chose and allocates by it is the main memory-safety risk; D252 bounds the agent table by the ledger's account count and R8 bounds every count. The fuzz seeds include a crafted next ID and a crafted count. | this plan |

**Dispositions of the notes read** (as AGENTS.md asks; the supervisor edits `doc/notes.md`):
- **Apply when the change merges:** N-225 (D246), N-237 (`Yield` after `Principal` in D248 and the mid-job season test), N-247 (D255, `TestDirected_*`), N-256 and N-309 part 2 (D249), N-265 (tag 9 and the carry; the `Guard:` test of the carry is row 11's, already built), N-288 (every run placed: `BoundRun` tag 4, unfed run tag 3, floor and clamp runs tag 7, the rest tag 9), N-294 (D257).
- **Apply, then close:** N-215. Proposal: accept the identity of seed 0 with salt 0 (CHK 6.8.2 documents it; the vector test and `TestRestore_Reseed` pin it, and the golden does not use a reseed).
- **Leave open:** N-125 (closes when `TestSnapshot_FunctionOfSavedState` passes; add: the snapshot header's seed is the effective seed); N-128 and N-244 (standing rules; this plan writes the hosted fields only in the agent record and adds none); N-129 (standing; `forkAtN` takes `Opts` as a function for that reason); N-202 (closes when TEC 4.4.1 and 4.4.6 are rewritten; no module keeps a key in the slice); N-204 (the slot setters are INT's); N-214 (CHK passages are plan 1's; EXP, CLB and STB passages remain); N-254 (a scenario, not this plan); N-217 and the other ALL notes (N-016, N-017, N-027, N-103, N-126, N-163): this plan adds no field, event field, word of setting vocabulary or default macro constant; `WorldRestored` adds its row to the OBS table already.

**Note proposals (text only):**
1. *N-new (CHK, CODE): the section protocol.* "A module's checkpoint section is appended to `checkpointSections` at its tag with its `saveX`/`loadX` in its own file, its tag deleted from `notBuiltSections`, its cross-checks appended to `checkpointCross`, and its fork-at-N test; nothing is renumbered. Raised by plan 2-2 D243."
2. *N-new (CHK, SKL, REG): tags 10 and 11 are written as zero-row skeletons by the checkpoint change; the owner keeps the skeleton as the prefix of its layout so a world with no rows keeps its bytes and the digest. Raised by plan 2-2 D244.*
3. *N-new (CHK, CLB, REF): the golden digest moves with every rebalance of a pack; each such change gives the testing.md 7.2 explanation for item 6 too. Raised by plan 2-2 D257.*
4. *Update to N-215:* accepted as stated above; close with the merge.
5. *Update to N-256, N-294, N-247, N-237, N-225, N-288:* "built in `checkpoint*.go` (plan 2-2); close" (N-294 only after plan 8 re-records the digest line, plan 8 D371). *Update to N-309:* "item (2) built in `checkpoint*.go` (plan 2-2); the note stays open for plan 7 (items 1a, 3), plan 12 (FR-EST-003) and the golden item (5)."
