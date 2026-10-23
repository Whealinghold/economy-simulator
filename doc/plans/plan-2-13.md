# Plan: Stage 2, row 13: INF, `information.go` and `internal/rumour`

**Status: written 2026-10-22 against the Stage 1 tree (signed off, commit `6dd4f86`); nothing built.** Sequences the INF row of Stage 2 of [roadmap-v1.md](../roadmap-v1.md) ("`information.go` and `internal/rumour`") into eight phases, under the contract of [plan-2-0.md](plan-2-0.md). A plan decides nothing: the specs, the ADRs and `doc/rules/` decide, and where this plan and they disagree they win.

Rule tags read (all three files in full): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1 (the tags of the contract; the files show the same).

Specs implemented: [spec-information.md](../specs/spec-information.md) (INF), FR-INF-001 to 052 and NFR-INF-001 to 004. Read for it: `spec-regions.md` (REG 4.3, 4.4, 4.6, 7.3), `spec-beliefs.md` (BEL 7), `spec-world.md` (WLD 5.1, 6.5, 6.7, 7.1.4, 7.3 draws D4 and D11, C8, C17), `spec-config.md` (CFG 3.1, 4.1, 7.1.6, 8.2), `spec-checkpoint.md` (CHK 4.2 tags 16 and 22), `spec-observability.md` (OBS 4.1, 9.5), `spec-migration.md` 6.2 and `spec-merchants.md` (the two other readers), `spec-integration.md` 4.7.3, `spec-stability.md` (S46 to S50, L93 to L95).

Decision block of this plan: **D460 to D479**.

Notes read (index, then each open note tagged INF or ALL, with updates; the file was not read in full and no note or spec is edited, so no Notes tag is quoted, `AGENTS.md`): INF: N-110 leave open (answered by MIG: prices only), N-112 **apply** to the code (D463: no `Informs`, no outside source; the edit to the spec is owed, Phase 1), N-156 and N-157 leave open (not INF's to settle: a rumoured rent or title is a second kind of rumour), N-170 leave open (a route does not change a lag; nothing to read), N-214 **apply** (D465: the site `siteInform` is appended here). ALL: N-016 apply (rows exist for the four fields; the registry entries are deleted, Phase 3), N-017 apply (setting-neutral names, the glossary, Phase 1), N-027 apply (every default is a placeholder and says so), N-103 leave open (a plan edits one file), N-126 apply (the event row and the catalogue entries, Phases 5 and 8), N-128 leave open (INF has no per-agent state; its record is its own section 16), N-163 leave open (the word "treasury" is not used), N-217 apply (the module leaves no trace while off, D460, D461). N-111 (a view takes draws) is read too: the filler of D466 keeps its order.

## Scope

**Builds.**

- `internal/rumour` (stdlib and `internal/checked` only, knows no region, commodity or role, architecture.md 2.9, NFR-INF-003): `Lag` (checked `1 + ceil(distance / speed)`), `Decide` (the four cases of INF 5.3.1 over primitives), `Blur` (half-even, clamped, checked), and the `Record` (a ring of `Obs{Price, Mean, Age}` for `S` sources and depth `D`, in one slice).
- `information.go` (the module file, architecture.md 2.10.1, in this order): the config types `Information` and `DefaultInformation` live in `config.go` and `defaults.go` (catalogue); the file holds the state struct, `validateInformation` and the warnings, `newInformation` (sources), the P5.2 method, the P3.6 method, the filler of the trade view, the shared function `rumoursFor`, `Rumour`, `AgentView.Rumour`, and the checkpoint section 16 writer and reader.
- Catalogue appends (table below): `Modules.Information` and `Role.Informed`, five bounds, `RumoursServed`, three reason constants, the hash fields, the `Warning` count, the site constant, one position of `tick.go` (P5.2) and the INF emitter inside plan 5's P3.6, three log keys (`rumours`, `lag`, `pairs`).
- `config/mapping.go`: `[modules.information]` and `informed` on a role.
- The overlays `testdata/chains/overlay/information-two.toml`, `information-three.toml` and `information-five.toml` (module table and `informed` only, no edge tables), the golden `testdata/golden/information/`, the scenarios S47 to S50 (S46 and half of S47 wait for MER).

**Leaves out**, with the owner:

| Item | Owner |
|---|---|
| The migration pass that calls `rumoursFor` at P7.2, its draws (D11, its own site) and its counts | plan 14 (MIG); this plan builds and tests the function (FR-INF-051 half) |
| The merchant strategy, the shipper and S46, S67, S69 | plan 15 (MER) |
| The rumours kept for an external agent | plan 17 (INT 4.7.3); tag 22 is registered here, empty; plan 17 owns the rows and the cross-checks; this plan promises that `v.Rumours` is complete before `Trade` (D466) |
| `economysim.information.rumours` and the attribute `economysim.outcome` | plan 22 (`otelobserver/`); this plan gives the event (section Cross-plan contract) |
| The away and on-journey skip of a view | plan 10 (`offStage` stub), 12, 14 through `offStage` at `tradeAgent`; this plan relies on it |
| The public `SetEdgeStatuses` and the edge-status shock | plan 17; S48 uses the unexported `setEdgeStatuses` of plan 3 |
| A rumour of an outside node, a relay, a smoothed field, a blackout shock, a warm-started record | closed or open in INF 17; not built (N-112, FR-EXT-033) |
| `testdata/golden/all/` | does not exist yet; nothing to move |

## Decisions

- **D460. One file, one pointer, one package.** `information.go` and `internal/rumour` as architecture.md 2.6 lists. The world holds `info *information` (nil = off); every call is guarded by it. The module is hash field 6 (CFG 8.2.1); with it nil nothing is allocated, hashed, drawn or emitted (FR-INF-001, FR-INF-047, N-217, architecture.md 2.10.5). Source: ADR 0007, INF 18.
- **D461. Hash.** `Modules.Information` is written as pair 6 of `Modules` by `pairsModules` (explicit ids, the code writing the pairs in ascending id place, whatever the declaration order); its fields are `InfoSpeed` 1, `RumourErrorBP` 2, `RumourMaxAge` 3 in declaration order. `Role.Informed` takes id 19, the first free id after `Closed` 18 (plan 12 adds no `Role` field; `MerchantFeeShare` is appended with the merchants module, so Fixed is 20 (plan 14), MerchantFeeShare 21 and Merchant 22 (plan 15); the implementer checks `hash.go` and writes the id in CFG 8.2.2, Phase 1) and is written only when `Modules.Information` is non-nil, and as every bool only when true. `pairsRole` gains the `Modules` value as a parameter if no earlier plan added it. Source: CFG 8.2.2, 8.2.3, FR-INF-044.
- **D462. No lag table.** INF 7.2.1 keeps an `R * R` table rebuilt after each status change. This plan derives `Lag` at use from REG's distance (one table read and one checked division per candidate, about 1 300 a tick for the first consumer's world), so nothing is rebuilt, nothing needs a hook in `regions.go` (a module's logic outside its file is a defect, architecture.md 2.10.2) and a stale table cannot exist. Observable behaviour is the same: the next phase 3 uses the new lags, a built rumour does not change (FR-INF-026). Source: architecture.md 2.10.2; the wording of INF 7.2.1, 14 and FR-INF-026, 030 is owed (Phase 1).
- **D463. Sources.** `(region, commodity)` is a source iff the commodity is in the tradable set (`roleRow.tradable`, which includes the items of the modules before it) of some role that has a `Group` in the region, in ascending (`RegionID`, `CommodityID`) order: "a market is active when some agent of the region can trade it" (REG 3.1.1). The set is derived at construction and fixed (INF 4.1.1). No `Informs` function and no declared pair: EXT declares none (FR-EXT-033, N-112), and an interface with no caller is not built (architecture.md 4.2). A gate region is a source like any grouped region (MER 11.3). FR-INF-003 and the declared half of FR-INF-002 are reserved; the wording is owed (Phase 1).
- **D464. The record.** One `[]Obs` of `S * D` cells, cell `source * D + tick % D`, and one `held` count (every source is written every tick, so all hold the same number; the writer keeps one, the checkpoint stays per source as CHK 4.2 tag 16 says and the reader rejects unequal counts as corrupt, CHK 5). P5.2 writes tick `t` after P5.1 and before P5.3: price and age from `markets.reports[i]`, mean from `markets.windows[i].Mean()` (0 when empty), in ascending source order, no event, no allocation. Source: INF 4.2 to 4.4, WLD P5.2.
- **D465. The draw site.** `siteInform uint64 = 0x7B15D7AD592D7D8F`, the first eight bytes big-endian of SHA-256 of `economy-sim/site/siteInform` (the rule of WLD 7.1.4; the same command gives the stated value of `siteWork`, checked), keyed `(siteInform, tick, agent)`. It differs by more than 2^32 from the seven sites of `random.go` (checked); the Guard test over the site table gains it. The draw is `w.rnd.drawRange(-E, E)`, which is `lo + draw(2E + 1)` of WLD 7.2.1 (FR-INF-014). The filler keys the stream once per informed view and only when `E > 0`; `tradeAgent` keys `siteTrade` afterwards, so the two cannot interleave. Source: WLD 7.1.4, 7.3 row D4, ADR 0008, N-214.
- **D466. The filler and the view kind.** `fillAgentView(v, a)` has no argument that says which view it builds, and `Work` must carry no rumours (FR-INF-021). This plan adds `kind viewKind` (`viewWork`, `viewTrade`) to `fillAgentView` and to its callers (`workAgent`, `tradeAgent` and `phase_production_internal_test.go`); plan 12 adds no such argument. After the core call the fan-in runs `if w.info != nil && kind == viewTrade { w.fillRumours(v, a) }`. `fillRumours` skips a role that is not informed, builds `v.Rumours` into the world's scratch and logs the Debug line. It runs only for an agent that has a view, so a removed agent, the estates and an agent that is away get none (FR-INF-022). Source: INF 6.2, 6.3; architecture.md 2.10.4.
- **D467. One function, two readers.** `rumoursFor(dst []Rumour, q rumourQuery, tally *rumourTally) []Rumour` with `rumourQuery{reader RegionID, regions []RegionID, commodities []CommodityID, asOf int64}`: for each region of `regions` (nil = every region but the reader, ascending) and each commodity (ascending) that is a source, one candidate: draw first (when `E > 0`), then decide (5.3.1), then blur. It does **not** key the stream: the caller keys it (the view with `siteInform`, MIG with its own site), which is what keeps the draws of phases 3 and 7 apart. A view passes `asOf = t`, MIG passes `t + 1` (C17). `tally` may be nil (MIG). Source: INF 5.3 to 5.5, 6.1.3, FR-INF-051.
- **D468. The arithmetic.** `internal/rumour.Lag(distance, speed) (int64, bool)`: `false` for `distance < 1`, `speed < 1` or overflow, else `1 + (distance + speed - 1) / speed` with `checked.Add`; at least 2. `Decide(hasPath bool, lag, maxAge, asOf int64) Outcome` returns the first case that applies: `NoPath`, `TooOld` (`lag > maxAge`), `NotYet` (`lag > asOf`), `Offered`. `Blur(price, e, maxPrice) (int64, bool)` is `checked.MulDivHalfEven(price, 10000+e, 10000)` clamped to `[1, maxPrice]`; `Mean` is blurred only when above 0. An overflow is a violated invariant after validation (the bounds make it impossible) and panics with "invariant". Source: INF 5.2, 5.3.1, 5.5.1; architecture.md 3.5, 8.1.
- **D469. Candidates and order.** For a view the candidates are `(r, c)` with `r != reader`, `c` in the agent's tradable set and `(r, c)` a source, ascending `r` then `c`; one draw each, offered or not, when `E > 0` (5.3.2); the count of draws of a tick is the sum over the informed views (FR-INF-014). A view carries at most one rumour per pair, already sorted (5.6).
- **D470. The view group.** `Rumour` (six fields, FR-INF-019) is declared in `information.go`. `AgentView` gains `Rumours []Rumour` (before the unexported `scratch`), a slice of a scratch sized at construction to the largest informed role's bound (at most `MaxRumoursPerView`), capacity capped, valid for the call only. `func (v *AgentView) Rumour(region RegionID, c CommodityID) (Rumour, bool)` is a binary search written as a loop (no closure, no allocation, FR-INF-024). Source: INF 6.2, 6.7.
- **D471. The event and its position.** `RumoursServed{Tick, Region, Readers, Offered, NoPath, TooOld, NotYet, MaxLag}` (counts and `int64`s, a new event, so its `Region` may be a typed ID, observability 4.2). The per-region tally is scratch, reset by the P3.6 emitter after it emits, filled by `fillRumours` only when `w.obs.supplied`, and emitted at P3.6 per region ascending when `Readers > 0`, before `WantShortfall`. Source: INF 10.1, WLD P3.6, FR-INF-040 to 042.
- **D472. Outcome constants.** `ReasonNoPath`, `ReasonTooOld`, `ReasonNotYet` of type `Reason` (`no_path`, `too_old`, `not_yet`) are appended to `event.go`; they name the closed set of the metric attribute with the literal `offered`, which the adapter spells. Source: INF 8 ("Rule constants"), observability 5.4.1.
- **D473. Validation.** `validateInformation(v)` is called once in `structural()` after `economics()` and before `overflow()`: field bounds first (`ErrOutOfBounds`, a missing `InfoSpeed` included, path `modules.information.info_speed`), `Role.Informed` with the module nil (`ErrModuleDependency`), then, only if the bounds held, the derived checks (`ErrTooMany`: `S * RumourMaxAge` over `MaxRumourCells`; an informed role with `|tradable| * (R - 1)` over `MaxRumoursPerView`). No new sentinel. Source: INF 9.1, 9.2; CFG 7.1.6, FR-CFG-074.
- **D474. Warnings.** `information_without_readers`, `information_without_routes` and `informed_without_sources` are added in `warnings()` (they read the configuration alone). `rumour_horizon_short` needs shortest distances, so it is appended in `Warnings()` after the structural phases come back clean, as the viability warnings are (CFG 7.2). It carries `pairs` as a new `Count int64` on `Warning` (appended; zero for a warning that counts nothing), logged under the new key `pairs`; the edits are owed in Phase 1 items 5 and 7. Source: INF 9.3, FR-INF-035 to 039.
- **D475. Logs.** The construction summary's `modules` list gains `information`; each warning is the existing "configuration warning" Warn line (rule, path). One Debug line "rumours served" per informed view (`agent`, `region`, `rumours`, `lag`), behind `Enabled`; no Info and no Warn in the tick. The keys `rumours` and `lag` are already rows of observability 3.3.2; `internal/logkeys` lacks both and gets them. Source: INF 10.2, 10.3.
- **D476. Checkpoint.** `information.go` writes and reads section 16 exactly as CHK 4.2 and INF 14 say: per source ascending, the count held (0 to `D`) and the entries oldest first (`Price`, `Mean`, `Age`). A restore at tick `T` places entry `i` of `n` at tick `T - n + i`; a count above `D`, unequal counts, a `Price` or `Mean` outside `[0, MaxPrice]` (`Price` at least 1) or a negative `Age` are corrupt. The write position, the sources and the rumours are not saved (FR-INF-029, 030). Tag 22 is registered here with an empty (zero-row) `saveRumours`/`loadRumours` pair in `information.go` so the checkpoint bytes, and the digest of the module-alone golden, do not move when plan 17 fills it (plan-2-0, testing.md 7.1.1): its row in `checkpointSections` has `present` = `Modules.Information != nil`, and 22 leaves `notBuiltSections`. Plan 17 owns the rows and the cross-checks.
- **D477. File shape.** `[modules.information]` read over `DefaultInformation()` (`RumourErrorBP` 500, `RumourMaxAge` 60, `InfoSpeed` 0 so a missing key is `ErrOutOfBounds`), keys `info_speed`, `rumour_error_bp`, `rumour_max_age`; `informed` is a flag of `[role.<id>]`. An unknown key is `ErrUnknownKey`, a non-integer `ErrNotInteger`; a file and the same Go value hash alike (FR-INF-045). A Go caller starts from `DefaultInformation()` too.
- **D478. Scenarios.** S47 (first assertion), S48, S49 and S50 run in this plan with a recording strategy; S46 and the second assertion of S47 wait for MER (plan 15, and TRN's dispatch cap from plan 12). The closure leg of S48 uses plan 3's unexported `setEdgeStatuses(changes, cause)` from an internal test. Packs: `two` layered with plan 3's `overlay/trade-two.toml` (the edges) and then `overlay/information-two.toml`. The overlays are split by variant as plan 3 D278 does, so plans 14, 15 and 24 layer the one of their pack after `trade-<variant>.toml`.
- **D479. Fuzz, mutation, benchmark.** `FuzzLag` and `FuzzBlur` in `internal/rumour` (NFR-INF-004); `gomu` on `internal/rumour` in Docker (`--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS=4`, `--workers 4`); `BenchmarkRumours_ReferenceView` reports the 1 280-candidate tick (NFR-INF-002, unmeasured, a report and not a gate).

## Phases

### Phase 1: the edits owed before any code

**Goal.** The specs, rules and notes say what the code does (D462, D463, D465, D474). No code. **Files.** The documents below; the supervisor applies the shared ones (plan-2-0).

| # | File and section | Edit |
|---|---|---|
| 1 | `spec-information.md` 4.1.1 | state "active" as D463 (tradable set of a role with a group in the region), citing REG 3.1.1 |
| 2 | `spec-information.md` 4.5, 4.5.2, 9.1 (last row), FR-INF-002, FR-INF-003, 17.1 | the declared-source half is reserved: EXT declares none (FR-EXT-033, N-112); FR-INF-003 reads "reserved, no declarer exists" and its test is owed to EXT if N-112 is reversed |
| 3 | `spec-information.md` 7.2.1, 14 (Not saved), FR-INF-026, FR-INF-030 | no lag table: the lag is derived from REG's distance at use (D462) |
| 4 | `spec-information.md` 13 (INF-S4 row) and 14 (last paragraph) | "FR-INF-028" is FR-INF-029 (the restore); FR-INF-028 is the trip |
| 5 | `spec-information.md` 10.3 and `rules/observability-and-logging.md` 3.3.2 | the key `pairs` (count of ordered region pairs), fresh rule tag; `Warning.Count` (D474) |
| 6 | `spec-world.md` 7.1.4 (site table), 7.3 (row D4: stream and the list of "no constant yet") | add `siteInform`, arity `tick`, `id` = the agent (D465); `random.go` comment follows in Phase 4 |
| 7 | `spec-config.md` 8.2.2 and 7.2 | ids of `Information` (1 to 3) and of `Role.Informed`; `Warnings` appends `rumour_horizon_short` after the structural phases; the `Warning.Count` field (D474) |
| 8 | `doc.go` glossary (Phase 3 PR, architecture.md 3.2) | Rumour, Rumour source (not "Source", which is PRO's recipe term), Informed role, Vintage, Lag, Information speed, Price record, and the identifiers `Information`, `RumoursServed` (the glossary test reads them) |
| 9 | `doc/notes.md` | text proposals under Risks (supervisor applies) |
| 10 | `architecture.md` 2.6; `AGENTS.md` Layout | conditional on a Lizard failure (N-243): the same layout row for `information.go` as plans 12, 14 and 15 give theirs (`information_view.go`: filler, `Rumour`, `AgentView.Rumour`; `information_state.go`: state, P5.2, section 16; the supervisor confirms the exact names before the split, no glob), added before any split; fresh RULE-TAG on `architecture.md`; the same words in `AGENTS.md` Layout |
| 11 | `spec-reference-chain.md` 4.5 and 7.6 | list `overlay/information-two.toml`, `information-three.toml`, `information-five.toml` (module table and `informed` only), layered after `trade-<variant>.toml` |

**Exit check.** Items 1 to 9 and 11 (and 10 if a split is made) are applied or listed as refused by the supervisor; the references of this plan resolve.

### Phase 2: `internal/rumour`

**Goal.** The pure rules (D468, D464). **Files.** `internal/rumour/lag.go`, `blur.go`, `decide.go`, `record.go` and tests, `internal/rumour/testdata/fuzz/`. No dependency on a plan.

Tests (all table-driven, in `package rumour_test`): `TestLag_FloorAndCeiling` (distance 1 with any speed gives 2; 40 and 55 gives 3; the 12.1 table), `TestLag_Overflow` and `TestLag_BadInput` (FR-INF-008); `TestDecide_FirstCaseWins` (a pair with no path and a lag above the horizon is `NoPath`; `TooOld` before `NotYet`) (FR-INF-010 to 013); `TestBlur_BoundsAndRounding` (370 at 500 gives 352 and 388 by half-even, the clamp at 1, `|d| <= ceil(p*E/10000)`) (FR-INF-015); `TestRecord_RingKeepsLastD` , `TestRecord_EmptyAtStart`, `TestRecord_VintageAt` (the oldest vintage reads at lag `D`; an overwritten cell is refused), `TestRecord_RestoreOrder` (FR-INF-007, FR-INF-017); `TestRecord_ReadWriteAllocate` (zero allocations, NFR-INF-001); `FuzzLag`, `FuzzBlur` (never panic, lag at least 2 for distance at least 1, result within `[1, MaxPrice]` and the bound, overflow reported, NFR-INF-004), with their seed corpus.

**Exit.** `go test -race ./internal/rumour`, coverage at least 90%, the fuzz targets run 1 minute each without a crasher.

### Phase 3: configuration, validation, hash, file

**Goal.** The module can be configured, validated, hashed and read from a file, and does nothing else. **Needs** plan 5 (the `Modules` pointer pattern and the guarded form). **Files.** `config.go`, `bounds.go`, `defaults.go`, `validate.go`, `hash.go`, `information.go` (validation and warnings only), `config/mapping.go`, `doc.go`, `registry_test.go`.

Work: `Information`, `Modules.Information` at its place in the order of the `Modules` row, `Role.Informed`; the five bounds with the values of INF 8; `DefaultInformation()`; `validateInformation` (D473) and the three warnings of D474; `pairsInformation`, id 6 and `Role.Informed` (D461); the loader (D477); delete `Modules.Information`, `Information`, `Role.Informed` from `notBuilt` in `registry_test.go`; add the glossary terms. `Warning` gains `Count int64` and the construction line logs it as `pairs` (D474).

Tests: `TestValidate_Information` (a case that breaks one rule each: `InfoSpeed` 0 and `MaxInfoSpeed + 1`, `RumourErrorBP` 5001, `RumourMaxAge` 1 and 10001, `Informed` with the module nil, `S * D` over the cells, an informed role over 4 096 candidates; `errors.Is`, the path) (FR-INF-001, 031, 033, 034, FR-CFG-074); `TestDefaults_Information` (FR-INF-032); `TestBounds_Information` (FR-INF-048); `TestWarnings_Information` (the four, one case each, volume once, FR-INF-035 to 039, FR-CFG-076); `TestHashConfig_Information` (the module pairs are written in ascending hash id, with pair 6 after the earlier modules' pairs; each of the four fields changes the hash; module off with `Informed` true hashes as off; the pin `TestHashConfig_ReferencePinned` does not move) (FR-INF-044, FR-CFG-105); `TestLoad_Information` and `TestLoad_InformationErrors` in `config/` (FR-INF-045); `TestParameterRegistry_Config` passes with the three entries deleted (FR-CFG-028).

**Exit.** The pinned hash and `testdata/golden/off/` are byte-identical; `go test -short -race ./...` passes.

### Phase 4: the record, the filler, the function

**Goal.** Rumours exist, with draws, in the trade view and for the migration pass. **Needs** plan 3 (the distance read, `Config.Edges`), plan 5 (the p36 and tick form). **Files.** `information.go`, `strategy.go` (the field), `fanin.go`, `phase_trade.go`, `phase_production.go` and `phase_production_internal_test.go` (the `kind` argument), `tick.go`, `poshash_internal_test.go`, `observe_internal_test.go`, `random.go`, `world.go` and `construct.go` (the pointer and `newInformation`), `internal/logkeys`.

Work: D463 to D470; `p52Record` as the P5.2 line in `positionsTail` between `p51Window` and `p53Markdown`, guarded by `w.info != nil`, its entry moved from `notBuiltPositions` to `positionList` with owner `INF`; the constructor in `newShell` after the tables it reads; `siteInform` and its Guard-test entry; the replay list of the per-position hash (`poshash_internal_test.go`) gains P5.2, and `observe_internal_test.go` gains its `stepSites` row; `modulesOn` gains `information`.

Tests (internal, `information_internal_test.go`, and external `information_test.go`; worlds built in Go, small, with edges): `TestSources_ActiveMarkets` (FR-INF-002); `TestRecord_AppendEveryTick` (every source, a tick without trade included, FR-INF-004); `TestObservation_Fields` (FR-INF-005); `TestRecord_OnlyP52Writes` (Guard, `go/parser`: the only call to the record's push is in `p52Record`, and no exported identifier has the type, FR-INF-006); `TestRecord_DepthAndNoSeed` (FR-INF-007); `TestLag_FromDistance` and `TestLag_Asymmetric` (the 12.1 tables, FR-INF-008, 009); `TestRumours_Cases` (a cut graph gives `NoPath`, a short horizon `TooOld`, tick 0 `NotYet`, FR-INF-010 to 013); `TestRumour_Allocations` (record write, view build, `Rumour()` allocate nothing, NFR-INF-001); `TestRumoursFor_NoViewNoDraw` (`Informed` false: no view and no draw; `E = 0`: no draw; a draw only in the caller's keyed stream, FR-INF-051); `TestRumours_DrawOrder` (replays `(siteInform, tick, agent)` and `drawRange`, one draw per candidate offered or not, in ascending source order, FR-INF-014, 015); `TestRumour_LagAgeVintage` (FR-INF-016, 017, 050); `TestRumour_FloorAtMaxSpeed` and `TestRumoursFor_AsOfNextTick` (FR-INF-018); `TestRumour_Shape` (reflection, six fields, FR-INF-019); `TestView_OrderAndOwnRegion` (FR-INF-020); `TestView_TradeViewOfInformedOnly` (Work, `OutcomeView`, other roles, no draw, FR-INF-021); `TestView_NoViewNoRumour` (estate and removed agent, FR-INF-022); `TestRumour_CohortNotMultiplied` (FR-INF-023); `TestAgentView_Rumour` (found, not found, zero allocations, FR-INF-024); `TestRumours_StatusChangeStepsVintage` (the closure of S48 as a unit, FR-INF-026); `TestRumours_RelocationChangesReader` (FR-INF-027); `TestRecord_EntryIsMarketPublication` (the record equals the market's publication every tick, a shipment in the run, FR-INF-028); `TestInformation_IgnoringStrategyChangesNothing` (module on against off with the default strategy: equal final snapshots and equal events but the hash, FR-INF-025); `TestDraws_NoneWithoutErrorOrReader` (E = 0, or no informed role: the PCG state after every tick equals the off world's, FR-INF-047); `TestInformation_Conservation` (`WithAudit`, flow totals equal off, FR-INF-049).

**Exit.** Those tests pass; `testdata/golden/off/` unchanged; `TestPositions_TableMatchesTheSpec` passes with P5.2 built; `tick_internal_test` shows no stale entry.

### Phase 5: the event, the aggregate, the logs

**Goal.** The three silent limits are counted (FR-STB-013). **Files.** `event.go`, `information.go`, `tick.go` (P3.6), `poshash_internal_test.go` (the P3.6 replay row), `construct.go` (no change), `event_test.go`, `render_test.go`, `observe_*_test.go`.

Work: D471, D472, D475. `RumoursServed` appended to `event.go`; "RumoursServed" deleted from `notBuiltEvents` in `event_test.go` (its place in `obsEventOrder` is already right); the P3.6 method calls this plan's emitter before DEM's `WantShortfall` (O2), plan 5 (D312) owns the `P3.6` entry of `positionList` and `p36Aggregates`; this plan only inserts its emitter ahead of DEM's call and widens the guard of the P3.6 replay row (created by plan 5, guarded by `w.demand != nil`) in `poshash_internal_test.go` to `w.demand != nil || w.info != nil`, so a world with only information records P3.6.

Tests: `TestRumoursServed_PerRegionAscending` (once per reader region, never with no informed view, after the last plan and before `WantShortfall`, FR-INF-040); `TestRumoursServed_CountsAddUp` (`Offered + NoPath + TooOld + NotYet` equals the candidates of the views, `MaxLag`, FR-INF-041); `TestRumoursServed_ExcludesMigrationCalls` (FR-INF-052, through `rumoursFor` with a nil tally); `TestRumoursServed_NoMapNoAlloc` and `TestInformation_NonInterference` (a logger and observer change nothing, the no-op path allocates nothing, FR-INF-042, NFR-INF-001, observability 8.2); `TestLogs_RumoursServed` (Debug only, keys `agent`, `region`, `rumours`, `lag`, behind `Enabled`, no Info in the tick, FR-INF-043, FR-INF-039); the cardinality test over the pack sees only values of the closed set.

**Exit.** The catalogue test passes with the entry deleted; the allocation test of a full `Step` stays within its baseline.

### Phase 6: the checkpoint section

**Goal.** Tags 16 and 22 (22 empty), and a fork at tick N that is bit-identical. **Needs** plan 2. **Files.** `information.go` (writer and reader of 16; the empty pair of 22), `checkpoint_sections.go` (rows 16 and 22), `checkpoint_information_test.go`. Both rows are in place before the golden of Phase 7 is recorded.

Tests: `TestCheckpoint_InformationRoundTrip` (the section bytes of a run equal the sources times counts and entries, no derived state, FR-INF-029, 030); `TestCheckpoint_InformationCorrupt` (count above `D`, unequal counts, price 0, negative age: refused, D476); `TestFork_Information` (E = 500, run 200 ticks, checkpoint at 80 and at plan 2's D258 set of N (0, inside a job, inside a season, after a call between ticks; the call case waits for plan 17), restore, run on: the snapshots, the events and the per-position hash equal the uninterrupted run, the draws taken equal the candidates of the informed views; S49, FR-INF-029, 046); `TestFork_InformationReseed` (`WithReseed` changes the errors and nothing else).

**Exit.** The fork test passes with the module alone on and the section is in the expected list of tags of a world that has it (and absent otherwise, CHK 4.2.1).

### Phase 7: overlay, golden, scenarios

**Goal.** The module-alone golden and the runnable scenarios. **Files.** `testdata/chains/overlay/information-two.toml`, `information-three.toml`, `information-five.toml` (each: module on, `info_speed`, `rumour_error_bp = 500`, `informed = true` on the roles of that pack that trade in more than one region; no edge tables, the edges come from `trade-<variant>.toml` layered before), `testdata/golden/information/` (rendering of `two` + `trade-two.toml` + `information-two.toml`: the pack digest, the event digest including `RumoursServed`, the positions), `golden_test.go`, `scenario_s47_test.go` to `scenario_s50_test.go`, `information_helpers_test.go` (the recording strategy, a hand-written fake registered with `WithStrategy`).

Tests: `TestGolden_Information` (module alone, seed 1, the length REF fixes for the golden, two architectures in CI, `-update` only after the diff is understood, FR-INF-046); `TestGolden_OffUnchangedByInformation` (Guard: the three files of `testdata/golden/off/` and the pinned hash are equal to the Phase 3 values, testing.md 7.1.1). The `two` + trade + information world (the golden world) is added to the invariants matrix (REF 7.5) and the determinism matrix (REF 8.2). Scenarios (end to end, skipped with `-short`): **S47** first assertion (a thin source: the rumour carries the source's `SourceAge`, `Lag` unchanged; INF-A3); **S48** (a three-region chain; closure for N ticks then reopening: the rumours of the cut region are absent and `NoPath` is counted, the vintage steps by the change of lag, the rumours return with the old lags, `EdgeStatusChanged` agrees with `RumoursServed`; INF-A4, FR-INF-010, 026); **S49** (E = 500 and E = 0, checkpoint at 80 resumed; with E = 0 the stream equals the module-off stream; FR-INF-014, 029, 047); **S50** (a seasonal swing at the source, lags of an eighth, a quarter and a half of the period: the phase shift of the rumoured gap; a measurement of the signal, INF-A5). Their seed counts follow `scenarioSeedCount()` (plan-2-1). Their outcomes are recorded in `testdata/chains/expected/` as Stage 1 did.

**Exit.** The golden is recorded once with the explanation of testing.md 7.2; `off/` is unchanged.

### Phase 8: the gate

`gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; `go mod tidy` no diff; `lizard -l go -C 10 -a 8 -L 100 -w` on `information.go` and `internal/rumour`; coverage at least 90% for the root and for `internal/rumour` (R and F of plan-2-1 D229); `gomu` on `internal/rumour` inside Docker (D479), survivors read and explained; `FuzzLag`, `FuzzBlur` 5 minutes each; the registry and catalogue tests with the INF entries deleted (`Modules.Information`, `Information`, `Role.Informed`, `RumoursServed`, position P5.2). The PR carries the `Testing` and `Observability` entries (below).

## Catalogue appends

| File | Appends | Position, id or tag | Spec line |
|---|---|---|---|
| `config.go` | `Modules.Information *Information`; `Information{InfoSpeed, RumourErrorBP, RumourMaxAge}`; `Role.Informed bool` | declared last in Go at the time (appended, after `Transport`); hash id 6 is the row position, written by explicit id in `pairsModules`; `Role` field after the last id | CFG 3.1, 8.2.2; INF 8 |
| `bounds.go` | `MaxInfoSpeed` 10^9, `MaxRumourErrorBP` 5 000, `MaxRumourAge` 10 000, `MaxRumourCells` 4 000 000, `MaxRumoursPerView` 4 096 | a new group at the end | INF 8; FR-INF-048 |
| `defaults.go` | `DefaultInformation()` (500, 60, speed 0); `Informed` stays false | end | INF 8; FR-INF-032 |
| `validate.go` | one call `validateInformation` in `structural()`; the warnings in `warnings()`; `rumour_horizon_short` in `Warnings()` | after `economics()`; after the viability block | CFG 7.1.6, 9.2; INF 9 |
| `hash.go` | `pairsModules`: pair **6** and `pairsInformation` (1, 2, 3); `pairsRole`: `Informed` gated by the module | explicit ids, the code in ascending id place, never by declaration order; `TestHashConfig_Information` checks the pair order | CFG 8.2.1 to 8.2.3; FR-INF-044 |
| `errors.go` | none: `ErrOutOfBounds`, `ErrTooMany`, `ErrModuleDependency` exist | | INF 9.1 |
| `event.go` | `RumoursServed`; `ReasonNoPath`, `ReasonTooOld`, `ReasonNotYet` | appended to the event list of `event.go`; its row in `obsEventOrder` is already between `EndowmentAdjusted` and `OutsideTraded` | OBS 4.1; INF 10.1 |
| `option.go` | none (the module has no option) | | |
| `tick.go` | `P5.2` into `positionList` (owner INF, `line`), its call between `p51Window` and `p53Markdown`; the INF emitter inserted first in plan 5's `p36Aggregates` (the `P3.6` entry is plan 5's); the P3.6 replay row in `poshash_internal_test.go` widened to `w.demand != nil || w.info != nil`; the P5.2 `replayRows` row (`poshash_internal_test.go`, guarded `w.info != nil`) and the P5.2 `stepSites` row (`observe_internal_test.go`, between P5.1 and P5.3, kept complete by `TestCallSites_TheTableIsTheStraightLine`) | the order of the table of WLD 6 | WLD 6.5, 6.7; testing.md 6.8 |
| `snapshot.go`, `ledger.go` | none (no snapshot field, no reason, no account) | | INF 10.5, position 15 |
| checkpoint tags | **16** Information and **22** (empty, plan 17 fills it), both present with `Modules.Information`; both deleted from `notBuiltSections` | after 15; after 21 | CHK 4.2; INF 14 |
| `random.go` | `siteInform` and the row of the sites | after `siteCreation`, as the first appended site (build order 13; plans 14 and 16 append after it) | WLD 7.1.4 |
| `strategy.go`, `fanin.go` | `AgentView.Rumours`; `kind viewKind` and the guarded call | before `scratch`; after the core call | INF 6.2; AGT 6.4.1 |
| `internal/logkeys` | `Rumours = "rumours"`, `Lag = "lag"` (and `Pairs`, D474) | in the order of observability 3.3.2; the new `pairs` key needs its 3.3.2 row, which Phase 1 edit 5 already owes | INF 10.3 |
| metrics | `economysim.information.rumours`, Counter `{rumour}`, attribute `economysim.outcome` (`offered`, `no_path`, `too_old`, `not_yet`) | recorded by plan 22 from `RumoursServed` | observability 5.3, 5.3.1, 5.4.1 |
| tests' staged lists | delete `RumoursServed` from `notBuiltEvents`; `Modules.Information`, `Information`, `Role.Informed` from `notBuilt`; `P5.2` from `notBuiltPositions` | | architecture.md 9.7, 2.10.3 |

## Cross-plan contract

Consumes (names are as the specs give them; the supervisor resolves each against the finished plan, O5). Plans 2, 5, 10 and 12 are named beyond Needs (3): section 16, the P3.6 form, offStage, dispatch.

- `Modules` pointer pattern and the `pairsModules` form (`config.go`, `hash.go`) <- plan-2-5
- the guarded `if w.x != nil` call and the accepted form of the position test (`tick.go`) <- plan-2-5
- the P3.6 method that emits `WantShortfall` (`phase_trade.go`) <- plan-2-5
- the `regionState` reads of the shortest open distance and next hop (`distance`, `next`, `regions.go`) <- plan-2-3
- `Config.Edges` and the file syntax `[region.<from>.edge.<to>]` (`config.go`, `config/mapping.go`) <- plan-2-3
- the all-pairs distances of the configured graph, for the warning (`internal/graph`) <- plan-2-3
- `overlay/trade-two.toml` (`testdata/chains/overlay/`), the edges under `information-two.toml` (the trade overlay of pack two) <- plan-2-3
- the rebalanced `two.toml` and `warm-two.toml` (`testdata/chains/`), the base of the golden and of S47 to S50 <- plan-2-8
- the unexported edge-status setter with a cause, used by S48 (`regions_status.go`) <- plan-2-3
- the unexported `relocate(a, to, cause)` and `moveAgent`, for FR-INF-027 (`regions_relocate.go`) <- plan-2-3
- `checkpointSections` / `notBuiltSections` (tag 16 row) and `checkpointCross` <- plan-2-2
- the codec primitives (`internal/codec`) <- plan-2-2
- `Checkpoint` and `RestoreWorld`, and the fork-at-N helper `forkAtN` with `forkCase` (`checkpoint_fork_test.go`), called by `TestFork_Information` <- plan-2-2
- `(*World).offStage` stub <- plan-2-10; its call sites at `workAgent` and `tradeAgent` and the transport half <- plan-2-12
- `dispatch`, to put a shipment in the run of `TestRecord_EntryIsMarketPublication` (`transport_request.go`) <- plan-2-12
- `fillAgentView` (`fanin.go`) <- Stage 1, row 11
- `tradeAgent` (`phase_trade.go`) and `workAgent` (`phase_production.go`) <- Stage 1, row 11
- `viewScratch` (`strategy_view.go`) and `AgentView` (`strategy.go`) <- Stage 1, row 7
- `roleRow.tradable` (`role.go`) <- Stage 1, row 2
- `marketTable.reports` and `marketTable.windows` (`market.go`), `beliefs.Window.Mean` (`internal/beliefs`) <- Stage 1, rows 3 and 4
- `stream.keyed` and `stream.drawRange` (`random.go`) <- Stage 0
- `Config.Warnings` and `warnings` (`validate.go`) <- Stage 0
- `WithStrategy` (`option.go`) and `World.obs.deliver` (`observer.go`) <- Stage 1
- `invariant` (`clearing.go`) <- Stage 1

Provides:

- `Rumour` (`information.go`) -> plan-2-15
- `Rumour` (`information.go`) -> plan-2-17
- `AgentView.Rumours` (`strategy.go`) -> plan-2-15
- `AgentView.Rumours` (`strategy.go`) -> plan-2-17
- `AgentView.Rumour` (`information.go`) -> plan-2-15
- `rumoursFor` (`information.go`) -> plan-2-14
- `rumourQuery` (`information.go`) -> plan-2-14
- `rumourTally` (`information.go`) -> plan-2-14
- `Information` (`config.go`) -> plan-2-14
- `Information` (`config.go`) -> plan-2-15
- `DefaultInformation` (`defaults.go`), used by this plan's tests and by Go callers
- `Modules.Information` (`config.go`) -> plan-2-14
- `Modules.Information` (`config.go`) -> plan-2-15
- `Role.Informed` (`config.go`) -> plan-2-15
- `viewKind` with `viewWork` and `viewTrade` (`fanin.go`, D466), used inside this plan and by `phase_production_internal_test.go`
- `MaxRumoursPerView` and `MaxRumourAge` (`bounds.go`) -> plan-2-14, plan-2-15, plan-2-17
- `fillRumours` (`information.go`, `fanin.go`) -> plan-2-17
- `viewKind` with `viewWork` and `viewTrade` (`fanin.go`) -> plan-2-17
- `Role.Informed` and `Modules.Information` (`config.go`) -> plan-2-17
- `RumourErrorBP`, a field of `Information` (`config.go`) -> plan-2-14
- the log keys `lag` and `pairs` (`internal/logkeys`) and `Warning.Count` (`validate.go`) -> plan-2-14
- tag 22 row (`checkpoint_sections.go`) -> plan-2-17
- the S48 scenario's edge-status call (`scenario_s48_test.go`) -> plan-2-17
- `RumoursServed` (`event.go`) -> plan-2-15
- `RumoursServed` (`event.go`) -> plan-2-22
- `ReasonNoPath`, `ReasonTooOld`, `ReasonNotYet` (`event.go`) -> plan-2-22
- `overlay/information-two.toml`, `information-three.toml`, `information-five.toml` (`testdata/chains/overlay/`) -> plan-2-14, plan-2-15, plan-2-16, plan-2-24
- `recordingStrategy` (`information_helpers_test.go`, test only; used by S47 to S50 of this plan)
- `testdata/golden/information/` (module-alone golden, pack two + trade-two + information-two) -> plan-2-24

`siteInform` (`random.go`) is used by no other plan; MIG appends its own site for D11.

## Testing

Requirement to test (each test carries its `Requirement:` line; a requirement with no test is a defect, testing.md 4.5):

| Requirements | Tests |
|---|---|
| FR-INF-001, 031 to 034, FR-CFG-074 | `TestValidate_Information` |
| FR-INF-002, 007 | `TestSources_ActiveMarkets`, `TestRecord_DepthAndNoSeed`, `TestRecord_RingKeepsLastD` |
| FR-INF-003 | none: reserved (Phase 1 item 2); the requirement is edited, not left untested |
| FR-INF-004 to 006 | `TestRecord_AppendEveryTick`, `TestObservation_Fields`, `TestRecord_OnlyP52Writes` |
| FR-INF-008 to 013 | `TestLag_*`, `TestDecide_FirstCaseWins`, `TestRumours_Cases` |
| FR-INF-014, 015, 047 | `TestRumours_DrawOrder`, `TestBlur_BoundsAndRounding`, `TestDraws_NoneWithoutErrorOrReader` |
| FR-INF-016 to 020, 023, 024, 050 | `TestRumour_LagAgeVintage`, `TestRumour_FloorAtMaxSpeed`, `TestRumour_Shape`, `TestView_OrderAndOwnRegion`, `TestRumour_CohortNotMultiplied`, `TestAgentView_Rumour` |
| FR-INF-021, 022 | `TestView_TradeViewOfInformedOnly`, `TestView_NoViewNoRumour` (estate, removed); the away and journey skips are tested in plans 12 and 14 |
| FR-INF-025, 049 | `TestInformation_IgnoringStrategyChangesNothing`, `TestInformation_Conservation` |
| FR-INF-026, 027, 028 | `TestRumours_StatusChangeStepsVintage` (S48), `TestRumours_RelocationChangesReader`, `TestRecord_EntryIsMarketPublication` (the record equals the market's publication every tick, with a shipment in the run) |
| FR-INF-029, 030, 046 | `TestCheckpoint_InformationRoundTrip`, `TestFork_Information`, `TestGolden_Information` |
| FR-INF-032, 044, 045, 048 | `TestDefaults_Information`, `TestHashConfig_Information`, `TestLoad_Information*`, `TestBounds_Information` |
| FR-INF-035 to 039 | `TestWarnings_Information` and a capturing handler (once, `Warn`, rule and path, nothing in the tick) |
| FR-INF-040 to 043, 052 | the Phase 5 tests |
| FR-INF-051 | `TestRumoursFor_AsOfNextTick` (the as-of tick) and `TestRumoursFor_NoViewNoDraw`; the end-to-end half is plan 14's |
| NFR-INF-001 | `TestRumour_Allocations` (record, view build, `Rumour()`) |
| NFR-INF-002 | `BenchmarkRumours_ReferenceView` (reported, not asserted) |
| NFR-INF-003 | the existing import guard over `internal/rumour` and a review of its signatures: primitives only |
| NFR-INF-004 | `FuzzLag`, `FuzzBlur` |

Golden files: `testdata/golden/off/` and the pinned hash do not move (proved by `TestGolden_OffUnchangedByInformation`); the module alone is `testdata/golden/information/` (pack `two` with `trade-two.toml` and `information-two.toml`, new); there is no `all/` set to move. Fork-at-N: `TestFork_Information`. Registry and catalogue rows: Phase 8. Per-position hash: P5.2 is added to the replay (testing.md 7.1.2). Fuzz: `internal/rumour/testdata/fuzz/`, committed with `Fix:` comments if a crasher appears.

**Scenarios of the register.** S47 (first assertion), S48, S49, S50 run here. **Waiting for MER (plan 15), with TRN's dispatch cap (plan 12):** S46 whole (a shipper that ships on a rumoured gap, the cap on and off, lags of 1, 2 and 4: INF-A1, INF-A2, L84, L93) and the second assertion of S47 (a merchant that ignores `SourceAge` ships, one that reads it does not: INF-A3, L94). S67 and S69 are MER's own versions. S49 is also the fork of this module.

## Observability

Observability-and-logging.md 9, answered:

1. **Failure path.** None at run time: the arithmetic cannot overflow after validation (an invariant panic, never an input). Every configuration error is returned by `Validate`.
2. **State transition.** The record is written with no event (INF 4.3.1); a rumour is not a transition. The three silent limits are counted by the aggregate `RumoursServed` (D471): once per reader region, a count struct, no map, no pointer, constructed only when an observer was supplied.
3. **Decision point.** Which candidate got no rumour and why is the Debug line and the three counts.
4. **Loops.** The candidate loop is per informed agent: no Info, no Warn, one Debug line per view behind `Enabled`; event construction allocates nothing under the no-op observer (allocation test).
5. **Changed event, metric or attribute.** A new event and, in plan 22, a new instrument; nothing existing changes.
6. **New metric attribute.** `economysim.outcome`, four values, so four series whatever the configuration (observability 5.4.1; plan 22 adds it to the list).
7. **Levels.** Debug only in the tick; `Warn` only at construction (the four warnings), bounded by the configuration.
8. **Logged and returned.** Nothing.
9. **Tests.** Phase 5, and the non-interference and cardinality tests (observability 8.1).

Log keys: `rumours`, `lag` (rows of 3.3.2, added to `internal/logkeys`), `pairs` (new, D474). Events: `RumoursServed`. Metrics: none defined here. The construction warnings have `Config.Warnings()` as their counterpart (observability 8.1); there is no run-time `Warn`.

## Risks and open items

- **O2. P3.6 holds two modules (owner: supervisor, plans 5 and 13).** WLD fixes `RumoursServed` before `WantShortfall` and gives the position the owner INF. Plan 5 builds the position first with `WantShortfall`; this plan inserts its emitter before DEM's call in the P3.6 method (not a catalogue list, so no append rule is broken). Plan 5 must write the method as a fixed-order sequence of guarded calls and set the position's owner to INF.
- **O5. Names from plans not yet written.** Plan 2's section hooks, plan 3's distance read and edge syntax, plan 5's module pattern and plan 10's offStage stub (extended by 12 and 14) are named from the specs. The supervisor checks each Consumes line against the finished plans; a change of a name changes this plan's text, not its design.
- **O6. The all-modules-on fixtures (owner: plan 24, D683).** `format-1-all.*` and `golden/all/` are created there; plan 2 owns only the slice fixture `format-1-one.*`. Nothing is regenerated by this plan, which proves its section by its own fork test.
- **O7. The word "source".** INF uses "source" for a (region, commodity) market, and the glossary's "Source" is PRO's recipe term. The code says `rumourSource`; the spec text is owed a clarification (Phase 1 item 8).
- **O8. Cost.** NFR-INF-002 is a placeholder; if the benchmark of 1 280 candidates is not under 0.5 ms the loop is rewritten, not the rules.

**Note proposals (text only; the supervisor applies).**

- *N-111 update (plan 2-13):* the filler of INF keys `(siteInform, tick, agent)` once per informed view and only with `RumourErrorBP` above 0; `rumoursFor` never keys the stream, its caller does; MIG keys its own site. Left open for MIG.
- *N-112 update (plan 2-13):* the code builds no `Informs`; sources are the active markets of grouped regions; FR-INF-003 and the declared half of FR-INF-002 are marked reserved. Left open for EXT.
- *N-214 update (plan 2-13):* `siteInform = 0x7B15D7AD592D7D8F` is appended (WLD 7.1.4, row D4); D11 (MIG) keeps no constant until plan 14.
- *New note (CFG, OBS):* a warning that counts (`rumour_horizon_short`, `pairs`) needs a count on `Warning` and a log key (D474).
