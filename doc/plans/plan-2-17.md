# Plan: Stage 2, row 17: INT, `integration.go` (external agents, `Direct`, the slot setters, the queries)

**Status: plan written (2026-10-22), not started. Written against plan-2-0.md; nothing here is built.**
Sequences the roadmap-v1.md Stage 2 row "`integration.go` (external agents, `Direct`, the slot setters, the queries) | INT (minimal: shocks and a player)" into eleven phases. It implements [spec-integration.md](../specs/spec-integration.md) (INT) sections 3 to 7, 12 and 13 for the modules that exist at this point, and the requirements FR-INT-001 to 011, 020 to 033, 040 to 054, 060 to 062, 070 and NFR-INT-001 to 005. It replaces the Stage 1 test seam of N-296 and un-skips S01 leg (b). Where this plan and the specs or rules disagree, they win and this plan is corrected.

Rule tags read for this plan (all three files read, tags equal those of plan-2-0.md): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1.
Specs read: spec-integration.md (in full), spec-world.md (3.5, 3.6, 5.3, 5.5, 6 positions P1.9, 7.1.4, 7.3, 8, 12), spec-regions.md (6, 7.3), spec-checkpoint.md (4.2 tags 3, 22, 34), spec-reference-chain.md (4.2 role `visitor`, 7.1.3, S01), spec-observability.md (4.1 rows, 5.3.5a), spec-information.md (6, 6.2).
Code read (every claim below was checked against it): `world.go`, `agent.go`, `fanin.go`, `phase_exit.go` (`spawnEntrant`, `fillSeeds`, `forget`), `phase_start.go`, `tick.go`, `strategy.go` (`Directed`), `strategy_view.go` (`fillDirective`), `option.go` (`resolveStrategy`, `isDirected`), `event.go`, `errors.go`, `snapshot.go` (`marketState`, `snapRegion`), `construct.go`, `random.go`, `export_scenarios_test.go`, `scenario_helpers_test.go`, `scenario_s01_test.go`, `testdata/chains/scripts/s01.toml`, `testdata/chains/base.toml` (role `visitor`).
Plans read for the names they provide (their Provides lines and the decisions behind them): plans 2, 3, 5, 6, 9, 10, 11, 12, 13, 14, 15, 16, with plans 18 and 19 read for what they expect of this plan. The setter forms differ per owner (D542) and are taken from those plans as written on 2026-10-22.
Notes read (index, then the entries tagged INT or ALL; the notes file is not edited by this plan):
- N-296 (INT, REF, STB): apply and close on merge. Phase 9 deletes the seam and un-skips leg (b).
- N-289 (OBS, INT): apply. `Market` returns `marketState`, the function `Snapshot` uses, so the two cannot disagree. Close on merge.
- N-290 (INT, POP, LED, CODE): the INT part is settled, not built: `refusal` in `phase_exit.go` limits only `DecideReduce` to `starved` and `insolvent`; a consumer spawner's `DecideRemove` with `CauseConsumer` already passes and `CauseConsumer` exists. No reduction cause is added (INT 3.6 offers no household removal); the POP part stays open.
- N-204 (WLD, INT, REG, TRN, MON, FIS, CHK): leave open for PLC. The public calls keep `ErrInStep`; the owners' unexported forms that this plan calls are the ones PLC calls at P0.3a with cause `policy`. Nothing here writes P0.3a.
- N-090 (PRO, SKL, INT): the INT part is applied (`Relocate`, Phase 6). Leave open for SKL (`in_traineeship`) and Stage 3.
- N-129 (EXP, INT, CHK): leave open; `forkAtN` takes `Opts` as a function, so a stateful strategy is never shared.
- N-130 (OBS, IND, ...): leave open; the bulk household read stays not offered (INT 9.4).
- N-131 (INT, PLC, ...): apply; nothing of the not-in-version-1 list is built (INT 3.6).
- N-136, N-143, N-161, N-164, N-177, N-194, N-205 (heading and first lines only): Stage 3 calls and slots (stores, credit, risk, legal, policy); leave open for their modules.
- ALL: N-016 apply (no `Config` field, no registry row), N-017 apply (Phase 1 lists the terms), N-027 apply (no default), N-103 apply (this plan edits one file), N-126 apply (one new event, `YieldShocked`, has its row), N-128 and N-244 apply (the rumours of 4.7.3 are INT's own slice indexed by agent slot, not a field of `agent`), N-163 apply (the word "pool" only), N-217 apply (`testdata/golden/off/` and the pinned hash do not move).

Ships in three PRs without renumbering: PR A = Phases 1 to 4 (record, errors, AddAgent/RemoveAgent, Direct); PR B = Phases 5 to 8 (slots, Relocate, queries, tag 22 and fork); PR C = Phases 9 to 11 (seam replacement, public legs, gate). The golden and catalogue appends of each PR are final for that PR.

Decision block: D540 to D559 (all used).

## Scope

**Builds (root package).**
- `integration.go`: `NewAgent`, `StockItem`, `AddAgent`, `RemoveAgent`, `Direct`, `Relocate`; the call frame; the slot setters `SetYieldShocks`, `SetEdgeStatuses`, `SetTransitFactors`, `SetFlowScales`, `SetBirthScales`, `SetDeathScales`, `SetCapacityScales`, `SetRegenScales`, `SetOutsideShocks` and the adjustment `AdjustEndowments`; `YieldShock`, `SlotState`, `Slots`; the position P1.9 (`p19Directives`); the rumour store and the checkpoint section of tag 22; `Calendar()`.
- Queries in the files spec-integration.md 20 names: `AgentState`, `Agent`, `Agents` (`agent.go`), `Market` (`market.go`), `RegionState`, `Region` (`snapshot.go`), `Warnings` (`world.go`).
- Catalogue: `errors.go`, `event.go` (`YieldShocked`), `tick.go` (P1.9), `checkpoint_sections.go` (tag 22), `fanin.go` (one appended call each in `fillAgentView`, `removeAgent` and `relocationRefusal` only if a rule below needs it).
- Test code: the conformance table for every setter, the fork-at-N and restore tests, the fuzz targets, the scripted call scenarios, the replacement of the seam, S01 leg (b).

**Slots that exist after this plan (nine setters and one adjustment).** A slot is listed only if its owner module exists in a plan below 17.

| Public call | Owner and plan | Needs (`Modules.`) | Event | Saved in tag |
| --- | --- | --- | --- | --- |
| `SetYieldShocks` | PRO, Stage 1 (`yieldShocks`) | none | `YieldShocked` (this plan) | 6 |
| `SetEdgeStatuses` | REG, plan 3 | none | `EdgeStatusChanged` (plan 3) | 11 |
| `SetFlowScales` | MON, plan 6 | `MoneySupply` | `MoneyFlowScaled` (plan 6) | 13 |
| `SetBirthScales`, `SetDeathScales` | POP, plan 11 | `Population` | `PopulationScaled` (plan 11) | 14 |
| `SetCapacityScales`, `SetRegenScales`, `AdjustEndowments` | RES, plan 9 | `Resources` | `EndowmentAdjusted` (plan 9) | 15 |
| `SetTransitFactors` | TRN, plan 12 | `Transport` | `TransitFactorChanged` (plan 12) | 17 |
| `SetOutsideShocks` | EXT, plan 16 | `External` | `OutsideShocked` (plan 16) | 18 |

**Slots that later modules append (Stage 3, six setters and one adjustment, making fifteen and two):** `SetRentScales` (LND, tag 25), `SetRateScales` (CRD, 26), `SetRouteRiskScales` and `SetStoreRiskScales` (RSK, 28), `SetLegalLevels` and `SetRightSuspensions` (ORG, 31), `AdjustTechnology` (TEC, 33); `SetFlowScales` gains the fiscal flows (FIS, 30). Each appends its field to `SlotState` in the order of INT 5.3, its row to the conformance table of Phase 5, its error row of 5.1.3, and its module test to `moduleOn`.

**Leaves out, with the owner.**

| Item | Owner |
| --- | --- |
| `AddStore`, `RemoveStore`, `ForgiveClaims`, `AddFacilities`, `AdjustFacilities`, `TransferFacilities`, `TransferTitles`, `DirectPolicy`, `Saving` | STO, CRD, CAP, LND, PLC, SAV (Stage 3); each appends to `integration.go`'s file list, not to a plan of this stage |
| the `Offer`, `Hire`, `Borrow`, `Lend` fields of `Directive` and the `ErrModuleOff` check for them (INT 4.3.2 row 1, 7) | EMP, CRD (Stage 3) |
| the `AgentState` groups Employment, Credit, Cover, Room, Facilities, Membership | EMP, CRD, INS, STO, CAP, ORG (Stage 3) |
| the internal forms that apply a slot and emit its event | the owner plans 3, 6, 9, 11, 12, 16 (this plan only calls them) |
| `RelocationRefusal`, `relocate`, `moveAgent`, `Graph`, `ReferencePrice`, `Endowment`, `Travel`, `Away`, `InTransit`, `RouteLoss` | plans 3, 9, 12 |
| `Checkpoint`, `RestoreWorld`, `Info` | plan 2 |
| a timer, a bundle of slots, an act-now trade, goods injection, a household removal | not provided (INT 3.6) |
| The bands of INT-S1 and INT-S2 (STY scenarios) | plan 19; this plan gives the calls, INT-S3 to S5, and S80 without bands (Phase 10) |

## Decisions

- **D540. Minimal INT.** One root file `integration.go` and no module: no `Config` field, no `Modules` pointer, no hash id, no default, no `internal/` package (INT 10, 20; architecture.md 2.9). It is a core concept file (AGENTS.md layout), so the module template of 2.10.1 applies in spirit: every touch point of INT is in this file, and outside it INT appears only in the catalogue files. Source: ADR 0003 decision 2; INT 10, 20.
- **D541. Which calls.** The calls and queries of the Scope list. The Stage 3 calls (INT 4.9, 4.10, 5.4, 6.7) and the six later slots are not built, and nothing in this plan adds a stub for them. Source: roadmap-v1.md Stage 2 and 3; INT 5.1, 5.4.
- **D542. Who checks and who applies.** INT validates the **whole slice first** (guard, module, each entry in the order of 5.1.3, duplicates), in one table-driven validator, because several owners' setters are per key and apply at once, so a bad third entry would leave two applied. Only then does it sort the keys ascending (the canonical order, so the slice order cannot change the events) and call the owner's unexported form; an error from it can only be a violated invariant (panic, INT 7.4). An entry equal to the held value is no change and no event (3.3.2). The forms, as the owner plans declare them: yield shock: `yieldShocks.set` (Stage 1), INT emits `YieldShocked`; edge status: `setEdgeStatuses(changes, cause)`, whole slice, emits itself (plan 3 D272); it returns bare sentinels with no path, so INT's validator pre-checks the entries (edge exists, factor in `[1, MaxEdgeFactorBP]`, duplicate) and builds the `*ValidationError` with its `changes[i]` path, and the owner's re-check can then only be a violated invariant; flow scale: `setFlowScale(i, bp, cause)` per flow index, INT maps the flow ID to its index (plan 6 D335), emits itself; birth and death scale: `setPopulationScale(region, kind, bp)`, state only, INT emits `PopulationScaled` (plan 11 D431); capacity, regen, adjust: `resourcesState.setCapacityScales`, `setRegenScales`, `adjustEndowments`, emit themselves (plan 9 D387); transit factor: `setTransitFactor(from, to, mode, bp, cause)`, emits itself (plan 12 D452); outside shock: `applyShocks(changes)` returns the `OutsideShocked` events, INT delivers them (plan 16 D535). Source: INT 3.3, 5 (intro), 5.1.3; the plans named.
- **D543. The call frame.** `func (w *World) enter(call string, module string, on bool) error` runs checks 1 to 3 of 3.2 (`guard()`, then `ErrModuleOff` naming the module) and wraps with the call's name (architecture.md 8.2). Every mutating call starts with it. The readers with no error result do not use it (6.1.2). The order is `ErrInStep`, `ErrWorldFailed`, `ErrModuleOff`, which is the order `guard()` already has. Source: INT 3.1.2, 3.2, 6.1.2; `world.go` `guard`.
- **D544. Argument errors.** A bad argument is a `*ValidationError` (`errors.go`) with `Path` `{"changes", "[i]", field}` rendered `changes[2].bp` by `quotePath`, `Rule` the sentinel, `Value`, and for a bound `Bound` the constant's name. Phase 2 fixes the segment spelling with one test. The slice argument is called `changes` for every setter, `n` for `AddAgent`, `d` for `Direct`. Source: INT 3.2.2; `errors.go` `ValidationError`.
- **D545. `AddAgent`, in this order, atomically.** Checks of 4.1.1 (region known and not outside, role, strategy name, bounds, commodities known and once, the fit probe of D547); then `Open(KindAgent, id, region)`, `fundInitial` (money and stock by `initial`, default lot key), `newAgent` with `external` true and `mortal` as asked, `fillSeeds` then `reseedBeliefs` (BEL 8.3, no draw), `drawCreation` on the stream `(siteAddAgent, Tick(), id)` (D546), the append to `w.agents`, `state.nextAgent++`, the counters `live.agents`, `live.households`, `live.inRole`, `state.hadAgent`, and `AgentSpawned{External: true}` with `Tick()` when an observer was supplied. It mirrors `spawnEntrant` and `builder.seedBeliefs`; the differences are the funding (`initial`, stated amount, not the pool) and the strategy. The agent-lifecycle fan-in `agentJoined(a, region)` (plan 5 D307, `fanin.go`) is called once the agent is in `w.agents` and the counters are moved, before `AgentSpawned`; it is the generic hook the modules that track agents share (plan 5 D307 provides it to 10, 11, 14 and 17; now it carries the demand carry). Source: INT 4.1; `phase_exit.go` `spawnEntrant`; `agent.go` `fundInitial`; plan-2-5 D307.
- **D546. The creation draws are keyed, not sequential.** Under ADR 0008 a draw is a function of `(seed, site, tick, id)`, so `AddAgent` takes its two words from a new site `siteAddAgent` = `0xE00949043F800843` (first eight bytes of SHA-256 of `economy-sim/site/siteAddAgent`, computed with `sha256sum` for this plan; it is more than 2^32 from every other site), keyed `(Tick(), new agent ID)`. Order of calls therefore does not change an agent's draws, and adding an agent changes no other agent's stream. INT 3.5.2, 4.1.2 item 4, 4.1.3, 9.1, 14.1, 14.2 and INT-S3 still speak of "words 14 and 15" and "call order"; they are stale (Phase 1 edits). Source: spec-world.md 7.1.3, 7.1.4, 7.3.2 (D12 "appended with its constant"), 7.3.3; `random.go`.
- **D547. No rollback, so a probe.** The ledger has no undo. Before `Open`, `AddAgent` asks a read-only probe `Ledger.FundFits(region, money, units)` (new, `internal/ledger`) whether the boundary balance, the region's flow totals and the total money stay within their bounds; false is `ErrOverflow` (INT 4.1.1, N-075 mapping of `ErrBalanceOverflow`). After the probe the funding cannot fail; a failure there is a violated invariant and panics (INT 7.4). `RemoveAgent` has the same shape: `estates.receive` can fail with a balance overflow, so a probe `estatefits` precedes the first change and returns `ErrOverflow` (a gap in INT 4.4, raised in Risks). Source: INT 4.1.1, 7.4; `fanin.go` `removeAgent`.
- **D548. Strategy index space.** `agent.strategy` indexes `ext.strategies`, which today has one entry per role (`world.go` line 237). The table grows to `len(Roles)` plus the sorted names `default`, `directed` and every registered strategy name, built once in `resolveExtensions`; `strategyIndex(name) (int, bool)` and the inverse `strategyName(a)` (plan 2's single point of change) read it. An empty `NewAgent.Strategy` means `directed`. `isDirected` is true for the dynamic type `Directed` only. The table is derived from the registrations, never saved; the checkpoint saves the name. Source: INT 4.1, 4.2.2; plan-2-2 D247 and Risks 5; `option.go`.
- **D549. `Direct` and the mailbox.** `agent.directive` stays the mailbox (saved in tag 3 by plan 2, D247). A new scratch field `agent.live *Directive` is the directive of the tick: `p19Directives` (P1.9, in `integration.go`, one line in `tick.go`) sets `a.live = a.directive` for every external directed agent and clears `a.directive` when it is not `Sticky`; `fillDirective` reads `a.live`. P1.9 is O(external agents) (spec-world.md 6 cost table): `integrationState.externals` is the ascending list of external slots, kept by `AddAgent`, `RemoveAgent` and the restore (rebuilt from the agents' `external` flag), and `p19Directives` loops over it only. The `tick.go` call and the replay row are guarded by `len(externals) > 0`, so a world with no external agent records no P1.9 line and `testdata/golden/off/one.positions.txt` does not move. `Direct` replaces the mailbox, an empty directive empties it, the checks are those of 4.3.2 rows 2 to 6 in the order "agent first, then arguments" (3.2 check 5 exception), and it emits no event and logs `Debug` "directive set". `Directive` has no `Offer`, `Hire`, `Borrow` or `Lend` yet and `Job` already is `*JobRequest` (4.3.3 is satisfied by Stage 1). Source: INT 4.3; spec-world.md 6 P1.9; `agent.go`, `strategy_view.go`.
- **D550. `RemoveAgent`.** External only (`ErrNotExternal` otherwise), mortal or not, ID found by the same lookup as `Direct`. It calls the one fan-in `removeAgent(a, CauseConsumer)`, so the event order (`RecipeAbandoned`, then the modules' events, then `AgentBankrupted`, then `EstateOpened`) is written once. It draws nothing, never refuses for an obligation, and the ID is never reused. The existing Debug line "agent bankrupt" is the log; INT 12.2's "agent removed" is a duplicate and is dropped (Phase 1 edit). Source: INT 4.4; architecture.md 2.10.4; `fanin.go`.
- **D551. `Relocate`.** The order of INT 4.5.2, composed from owners: `RelocationRefusal(a, to, "on_journey")` (plan 3), then if `w.migration.onJourney(a)` the cancel `cancelJourney(a)` (plan 14), then `abandonJob(a, region, ReasonRegionChanged)` if it has a job and `w.sub.drop(slot)` when subsistence is on (plans 14 and 10: the same two steps a departing migrant takes, D492), then `moveAgent(a, to, CauseConsumer)` (plan 3; not `relocate`, which cannot skip `on_journey`), then reseed the beliefs with `fillSeeds` and `reseedBeliefs` against `to` and set `roleSince = Tick()`. A pseudo-agent is `ErrNotRelocatable` (from the dry run), an unknown or removed agent is `ErrUnknownAgent` or `ErrAgentRemoved`, and the call changes nothing on any refusal. INT's agent lookup runs first and returns `ErrAgentRemoved` for a removed agent, before the dry run, so the dry run's `closed_account` for a removed agent is never reached. Plan 14's arrival (D491) does the same `fillSeeds`, `reseedBeliefs(a, row, nil, seeds, bp)` and `roleSince`, so the two share the Stage 1 functions and need no new helper. Source: INT 4.5; plan-2-3 D268, D269; plan-2-14 D491, D492.
- **D552. `Slots`.** One reader per slot returns the non-default list in the owner's saved order, from the same accessor the owner's checkpoint writer uses (so FR-INT-032 holds by construction). The birth and death lists differ from the configured value of the region, not from 10 000. The `SlotState` fields are in the order of INT 5.3; this plan has nine of fifteen, and a later module appends its field at the end of the struct. Source: INT 5.2.1, 5.3.
- **D553. The rumour store and tag 22.** `integrationState.views` is a slice indexed by agent slot holding, for an external agent of an `Informed` role with `Modules.Information` on, the tick of its last view and a copy of the view's rumours (at most `MaxRumoursPerView`). It is filled by one appended call in the `kind == viewTrade` branch of `fillAgentView` (plan 13 D466), after `fillRumours` (`recordRumours(v, a)`); a Work view never calls it, so it cannot clear the stored rumours, saved in tag 22 in ascending agent ID with rows only for an agent that has had a view, and reported by `AgentState` with tick -1 and an empty list when there is none. Its `present` is `Modules.Information != nil`. It affects no result (FR-INT-049). Source: INT 4.7.3, 13; CHK 4.2 tag 22; N-244.
- **D554. `AgentState` at this point.** The groups Identity, Standing, Holdings (with `Expiring` from `Ledger.Expiring`), Job, Beliefs, Markets, Upkeep, Mailbox and Rumours always; Wants (DEM on), Own use (SUB on), Away and Journey (TRN, MIG on) when their module is on. A later module appends its group; every appended field is a count, an amount, a bool or a slice, never a typed ID, so the off value contributes nothing (architecture.md 2.10.5). It is a deep copy of plain data, a function of the saved state alone, and omits `MaxExecutions`, refusal reasons and the stock after the next completions (FR-INT-033). Source: INT 4.7.2.
- **D555. `Market`, `Region`, `Agents`, `Warnings`, `Calendar`.** `Market(r, c)` returns `w.marketState(i, r, c)`; errors `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrOutsideNode`. `Region(r)` builds the one region through `snapRegion` on a one-region scratch and returns `RegionState` (OBS 5.3.5a): the core rows now, and for a module that is on the owner's rows for that region (Endowments, Shipments, Journeys) through the owner's per-region appender if it has one, else by filtering that owner's rows of a full snapshot (the cost class then rises to 3 for that module, recorded in Risks). `Agents(r)` lists live agents of the region in ascending ID with no pseudo-agent. `Warnings()` returns the construction warnings sorted by path then rule from a new `World.warnings` field set in `construct`. `Calendar()` returns `w.cfg.Calendar` as a deep copy. Source: INT 6; OBS 5.3.5a; N-289.
- **D556. Events.** One new event, `YieldShocked{Tick, Region, Recipe, BeforeBP, AfterBP}` (core gate, OBS 4.1 row), appended to `event.go` with its `isEvent`, deleted from `notBuiltEvents` and `deferredCore` in `event_test.go`. Every call's events carry `Tick()` (the tick the next step runs) and come before the next `TickStarted` because a call is between steps. Per call: `AgentSpawned`; `RemoveAgent` the fan-in's; `Relocate` `RecipeAbandoned` and `AgentRelocated`; each setter the owner's event. All are constructed only when an observer was supplied. Source: INT 12.1; OBS 4.1.
- **D557. Logs.** Debug only, guarded by `Enabled`, constant lowercase messages, keys `tick`, `agent`, `region`, `role`, `cause` from `logkeys` (no new key): "agent added", "directive set", and one per setter ("yield shocks set", "edge statuses set", "flow scales set", "birth scales set", "death scales set", "capacity scales set", "regen scales set", "endowments adjusted", "transit factors set", "outside shocks set"). No `Info`, `Warn` or `Error`. The setter lines are guarded and carry no per-key data (the event does). Source: INT 12.2; observability-and-logging.md 2.2, 3.1.3.
- **D558. "Changes nothing" is tested by bytes.** For a refused call the test takes `Checkpoint()` before and after and requires equal bytes, and counts events and stream use (no new `siteAddAgent` key). The fuzz targets `FuzzAddAgent`, `FuzzDirect` and `FuzzSetters` assert no panic and that. Source: INT 3.3.1; FR-INT-002; NFR-INT-005; plan-2-2 (`Checkpoint` is byte-stable).
- **D559. Goldens and gates.** `testdata/golden/off/` and the pinned hash do not move (no field, no event while a call is not made). There is no module-alone golden (INT is not a module). INT-S3 is a pair of runs from one call script compared state-for-state and event-for-event, and against a checkpoint-restored run; a recorded `testdata/golden/calls/` file is raised in Risks, not created. The gate is plan-2-0's, with the race-aware run of plan 1. Source: testing.md 7.1, 10; N-217.

## Phases

### Phase 1: the record (no code)

Goal: the specs and the neighbouring plans say what this plan builds. Files: none in the repository except the edits below, which the supervisor applies (plan-2-0: a plan edits one file).

Edits owed:

| File | Section | Change |
| --- | --- | --- |
| `spec-integration.md` | 3.5.2, 4.1.2 item 4, 4.1.3 last two sentences, 9.1 last paragraph, 14.1 (words 14 and 15; "changes the stream from the call onward"), 14.2 item 1, 16 INT-S3 | the creation draws are keyed `(siteAddAgent, Tick(), new agent)`; the order of calls does not change a draw; adding an agent changes no other stream (D546, ADR 0008) |
| `spec-world.md` | 7.1.4 table, 7.3 row D12, 7.3.2 | append `siteAddAgent` `0xE00949043F800843` with arity `tick`, `id` = the new agent; D12 names it |
| `spec-integration.md` | 5.1.3 row yield shock; FR-INT-040 | add `ErrOutsideNode` (an outside node has no recipes), after `ErrUnknownRegion` |
| `spec-integration.md` | 7.1, 11.2 | `ErrUnknownFlow` is declared by MON (plan 6 D335), `ErrOutsideNode` by EXT (plan 16), `ErrUnknownMode` by TRN (plan 12), `ErrUnknownResource` by RES (plan 9); INT defines the other seven |
| `spec-integration.md` | 12.2 | drop "agent removed": the existing "agent bankrupt" (`removeAgent`) is the line (D550) |
| `spec-integration.md` | 4.4.2 | the removal can overflow the estate's balance: a probe returns `ErrOverflow` before any change (D547); name the case |
| `spec-reference-chain.md` | 7.1.3 (Stage 1 seam paragraph and the sentence "a `[[buyer]]` is not buildable"), S01 ("leg (b) and the visitor wait for INT, N-296"), FR-REF-021 ("In Stage 1 a shock is applied through the unexported ...") | delete; add the buyer's limit: the ceiling of the commodity's reference band (Phase 9 item 4); `[[buyer]]` keys unchanged |
| `spec-checkpoint.md` | 4.2 tag 3 and tag 22 | the strategy-name table lists the names of external agents (D548); tag 22 `present` is `Modules.Information` and rows are sparse |
| `spec-agents.md` | 10.3 (`Directive.Job`), 14.6 | `Job *JobRequest` is built (Stage 1); `Agent(a)` returns `AgentState` |
| `doc.go` glossary | new terms, before any code uses them (architecture.md 3.2, `glossary_test.go`) | the terms agent state, forced move, market state, slot state, query class, entry (of a setter call), key (of a setter entry) (INT handoff list), and every exported type of `agent.go`, `snapshot.go` and `event.go` this plan adds, named so that `TestGlossary_DeclaredTypesNamed` passes: `YieldShocked`, `AgentState` and its group types, `SlotState`, `RegionState`, and the entry types `YieldShock`, `FlowScale`, `RegionScale`, `EndowmentScale`, `EndowmentAdjustment`, `TransitFactor`, `OutsideShock` |
| `architecture.md` 2.6 and `AGENTS.md` layout (and a fresh read-receipt tag on `architecture.md`) | `integration_slots.go`, `integration_agents.go` (INT), only if `integration.go` passes about 500 lines of code (N-243) or fails Lizard | the supervisor adds the two lines to both lists before the split is made; otherwise no new file (`integration.go` is listed) |
| `spec-integration.md` | "Edits owed to other specs" table | grep each row; apply the unapplied ones that this plan's code relies on (the rows for `spec-world.md` 3.5.1, 3.6.1, 8.3, 6.11 and `spec-regions.md` 7.3) before Phase 3 |

Reconcile (the setter forms, D542): each owner's form is different and none needs a change; three gaps are raised. Plan 9 gives no signatures for its three helpers (the entry type of `setCapacityScales` and the result of the adjustment are needed); plan 10's Provides line for `(*subsistence).drop(slot)` names plan 14 only and plan 17 uses it too; no plan declares the entry types `FlowScale`, `RegionScale`, `EndowmentScale`, `EndowmentAdjustment`, `TransitFactor` (Risks 1). A name that differs after the plans are audited is a one-line change here.
Exit: the table above is delivered to the supervisor and the reconcile list is empty or each gap has an owner.

### Phase 2: errors, the call frame, the validator

Goal: one place for checks 1 to 4 of 3.2. Files: `errors.go`, `integration.go`, `integration_internal_test.go`.

1. Append to `errors.go`: `ErrModuleOff`, `ErrAgentRemoved` (wraps `ErrUnknownAgent`, plan 3), `ErrNotExternal`, `ErrNotDirected`, `ErrNotOnRole`, `ErrNotTradable`, `ErrUnknownLine`. Not appended: `ErrUnknownAgent` and `ErrSameRegion` (plan 3), `ErrOutsideNode` (plan 16), `ErrUnknownMode` (plan 12), `ErrUnknownResource` (plan 9), `ErrUnknownFlow` (plan 6); `ErrOverflow`, `ErrUnknownStrategy`, `ErrDuplicateID`, `ErrOutOfBounds`, `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrUnknownRole` and `ErrUnknownRecipe` exist.
2. `enter`, the typed `moduleOn` check (a closure per module name over `w.cfg.Modules`), `keyedOnce` (duplicate detection by sorted copy, no map), and the `argError` constructor (D544).
3. Phase 2 adds no public call. The conformance harness `slotCase` (name, setter, module, entries, expected error per row of 5.1.3) is written here, empty, and Phases 5 appends its rows.

Tests: `TestEnter_Order` (a step running beats a failed world beats a module off; FR-INT-001, 006; the case "inside a step" is an observer that calls the setter); `TestSentinels_Distinct` (the new texts differ from each other and from the ledger's; `errors_test.go` list); `TestArgError_Path` (renders `changes[2].bp`; FR-INT-005); `TestKeyedOnce` (FR-INT-003).
Exit check: Lizard on `integration.go`, coverage of the new functions.

### Phase 3: strategy table, `AddAgent`, `RemoveAgent`

Goal: a player can be put in and taken out. Files: `integration.go`, `world.go` (`ext.strategies` growth and the name table in `resolveExtensions`), `random.go` (`siteAddAgent`), `internal/ledger/` (`FundFits`), `export_test.go`.

1. The name table and `strategyIndex`, `strategyName` (D548); `resolveStrategy` unchanged.
2. `Ledger.FundFits` and its test in `internal/ledger` (money, units, boundary and region flow bounds; no change to the ledger).
3. `AddAgent` (D545), `RemoveAgent` (D550) and the `estatefits` probe (D547).
4. `siteAddAgent` in `random.go` with its entry in the `Guard:` test of the 2^32 separation (`random_internal_test.go`, the site list).

Tests (names; `Requirement:` in the doc comment):
- `TestAddAgent_Errors` (a table with one failing case per sentinel of 4.1.1 and the order of checks; every case then compares `Checkpoint()` bytes) - FR-INT-020, 002.
- `TestAddAgent_Creates` (weight 1, `External`, `Mortal`, account in region, stated money and stock by `initial`, beliefs equal to the BEL 8.3 seed, `roleSince`, welfare record empty, one `AgentSpawned`) - FR-INT-021, 011, 009; FR-SCL-013.
- `TestAddAgent_LastID_AndKeyedDraws` (the next ID; no other agent's `target` or residue changes; the same call twice in two worlds gives equal draws; two agents added in either order get the same draws each) - FR-INT-022, 007, 008.
- `TestAddAgent_BoundaryFlow` (the construction bucket before the first step, the totals of tick `t` after; `WithAudit` passes) - FR-INT-009, 011.
- `TestStrategyName_Directed` (empty means `directed`, `default`, a registered name, an unknown one is `ErrUnknownStrategy`; a wrapper holding a `Directed` is not directed) - FR-INT-023.
- `TestExternal_Excluded` (a pack with the `visitor` role: the default spawner takes no decision on the external agent, the refusal reason is `external_agent`, with `Modules.Migration` on (plan 14 pack) the external agent is neither reviewed nor moved by the migration module, a mortal one passes a consumer spawner's `DecideRemove` with `CauseConsumer`) - FR-INT-024.
- `TestRemoveAgent_Order` (job abandoned, then `AgentBankrupted`, then `EstateOpened`; ID never reused; `ErrNotExternal` for a household; mortal or not) - FR-INT-029, 030.
- `TestRemoveAgent_EstateOverflow` (the probe: `ErrOverflow`, nothing changed) - FR-INT-029.
- `TestFundFits` (in `internal/ledger`).
- `FuzzAddAgent` - NFR-INT-005.
Exit check: gate on `internal/ledger` (`gomu` in Docker, caps) and the root.

### Phase 4: `Direct`, the mailbox and P1.9

Goal: a directive reaches the strategy and is dropped after one tick. Files: `integration.go`, `agent.go` (`live` field), `strategy_view.go` (`fillDirective` reads `live`), `tick.go`, `poshash_internal_test.go` (the P1.9 replay row), `observe_internal_test.go` (the P1.9 `stepSites` row), `view_export_test.go` (the harness `SetDirective` sets `live`).

1. `Direct` (D549), its row-by-row checks (4.3.2 rows 2 to 6) and the agent lookup shared with `RemoveAgent`.
2. `p19Directives`; `tick.go`: remove `"P1.9"` from `notBuiltPositions`, add `{"P1.9", "AGT", line}` after P1.8 in `positionList` (the table of spec-world.md 6: after the last position of phase 1 built before it, before P1.9a) and the call `w.p19Directives()` after the P1.8 call, guarded by `len(externals) > 0`. In `poshash_internal_test.go` add `{id: "P1.9", run: noErr((*World).p19Directives), on: hasExternalAgent}` to `replayRows` after the P1.8 row (the `on` guard of D177 is the same condition), or `checkRows` fails with "position P1.9 has no method in the replay map". In `observe_internal_test.go` add the `stepSites` row `{"P1.9", noError((*World).p19Directives)}` after the P1.8 row, calling it under the same `len(externals) > 0` guard as `tick.go` (a world with no external agent measures zero allocations at the site), or `TestCallSites_TheTableIsTheStraightLine` fails.

Tests:
- `TestDirect_Errors` (agent errors before argument errors; each of the seven sentinels of 4.3.2 that exist now; the mailbox unchanged on every failure) - FR-INT-025, 005, 002.
- `TestDirect_ReplaceAndEmpty` (a second call replaces, an empty directive empties) - FR-INT-026.
- `TestDirect_OneTickUnlessSticky` (a non-sticky directive acts in the next step and is gone after it, a sticky one stays until replaced; no event; a `Debug` line) - FR-INT-027, 052.
- `TestDirect_AffordableNotChecked` (a bid the agent cannot pay is accepted by the call and rejected by the step with `insufficient_funds`) - FR-INT-010.
- `TestDirect_NoJobLeavesWork` (intents without `Job`: the fallback's `Work` runs) - FR-INT-028.
- `TestPositions_P19` (the position test of `tick_internal_test.go` and `checkRows` pass with the new entry; a world with no external agent records no P1.9 line, one with an external agent records `t=N P1.9` every tick) - testing.md 6.8.
- `TestStep_AllocationsWithExternalAgent` (the baseline of `allocation_test.go` plus a constant measured in this phase) - NFR-INT-002.
- `FuzzDirect` - NFR-INT-005.

### Phase 5: the slots

Goal: every setter of the Scope table, with the conformance table. Files: `integration.go`, `event.go`, `event_test.go`, `render_test.go`, `internal/logkeys` (none), `integration_slots_test.go`.

1. `YieldShock`, `SetYieldShocks` (validate, then for each key ascending (region, recipe): `before := w.shocks.factor`, skip if equal, `w.shocks.set`, emit `YieldShocked`) and `YieldShocked`.
2. The other eight wrappers and `AdjustEndowments`: `enter`, the validator rows of 5.1.3 over the whole slice, then the owner's form of D542 per key in ascending order. `SetEdgeStatuses` is `enter`, the validator's edge rows, then `setEdgeStatuses(changes, CauseConsumer)`. `SetBirthScales` and `SetDeathScales` read the held value, skip an equal one, call `setPopulationScale` and emit `PopulationScaled`. `SetOutsideShocks` converts each entry to the owner's private type and delivers what `applyShocks` returns, in order.
3. `SlotState` and `Slots` (D552).
4. `slotCase` rows: one per setter with the failing cases of its 5.1.3 row, the equal-to-current case, the default-clears case, the duplicate, the empty slice, the module-off case (a pack with the module off).

Tests:
- `TestSlots_Conformance` (table over all rows: error sentinel, nothing changed by bytes, no event, no draw) - FR-INT-002, 003, 005, 006, 007.
- `TestSlots_EventsCanonicalOrder` (a slice in reverse key order gives events in ascending key order; `Tick()` of the next step; before `TickStarted`) - FR-INT-004, FR-OBS-017.
- `TestSlots_DefaultClears` (set then set to default: `Slots()` empty and `Checkpoint()` bytes equal a never-shocked world's) - FR-INT-048, 032.
- `TestSetYieldShocks` (a job started afterwards reads it at P2.7; a running job keeps its pending output; event fields) - FR-INT-040.
- `TestSetEdgeStatuses_Public`, `TestSetTransitFactors`, `TestSetFlowScales`, `TestSetBirthDeathScales` (default is the configured value), `TestSetCapacityRegenScales`, `TestAdjustEndowments`, `TestSetOutsideShocks` (each through the public call; the mechanism is the owner's, tested in its plan) - FR-INT-041 to 047.
- `TestSlots_RestoredEqualsOriginal` (set several, `Checkpoint`, `RestoreWorld`, `Slots()` equal) - FR-INT-032, 070.
- `TestInStep_Calls` (every public mutating call from an observer returns `ErrInStep`, state unchanged) - FR-INT-001.
- `FuzzSetters` (the nine setters and the adjustment, random entries) - NFR-INT-005.
Exit check: the catalogue test, the render test and the parameter-registry test pass with `YieldShocked` deleted from `notBuiltEvents` and `deferredCore`.

### Phase 6: `Relocate`

Goal: a forced move, including off a journey. Files: `integration.go`, `integration_relocate_test.go`.

1. `Relocate` (D551). 2. A test-only fake obligation (plan 3's registry) for the refusal cases; the real ones are plans 12, 14, 15's.

Tests:
- `TestRelocate_RefusalOrder` (unknown region, unknown agent, removed agent, estate pseudo-agent `ErrNotRelocatable`, same region, outside node, `ErrRegionBound` naming the obligation; nothing changes by bytes) - FR-INT-031, 030, 005, 002.
- `TestRelocate_Effects` (job abandoned with `region_changed`, region changed, `AgentRelocated` cause `consumer`, beliefs reseeded with no draw, `roleSince = Tick()`, money and goods unchanged, the mailbox kept) - FR-INT-031, 009.
- `TestRelocate_OnJourney` (a migrating household: journey dropped, counted in `Ended`, moved; with a second obligation open, nothing changes and the journey stays) - FR-INT-031; INT-S5. Built on the migration pack of plan 14 (`overlay/migration.toml`).
- `TestRelocate_OpenContract` (refused naming `open_contract`) - INT-S5; built on the contract pack of plan 15.
Exit check: coverage, Lizard.

### Phase 7: the queries

Goal: the readers of INT 6. Files: `agent.go`, `market.go`, `snapshot.go`, `world.go`, `construct.go` (the `warnings` field), `integration.go` (`Calendar`, `Agents`), `integration_query_test.go`.

1. `AgentState`, `Agent`, `Agents` (D554), `Market`, `RegionState`, `Region`, `Warnings`, `Calendar` (D555).

Tests:
- `TestQueries_NoDrawNoStateNoEvent` (every query on a stepped world: `Checkpoint()` bytes equal before and after, no event, no draw) - FR-INT-060.
- `TestQueries_Guards` (`ErrInStep` inside a step for those with an error result; `ErrWorldFailed` on a failed world; the readers with no error result return their value; legal in the construction phase) - FR-INT-060.
- `TestAgent_State` (each group against the ledger, the beliefs and the job of a stepped world; deep copy: changing the result changes nothing; unknown, removed and pseudo-agent IDs) - FR-INT-033, 030.
- `TestAgents_Region` (ascending, live only, no estate) - FR-INT-061.
- `TestMarket_Fields` (equals the snapshot's `MarketState` for every market; the seed price and age 0 before the first step; `ErrOutsideNode`) - FR-INT-053; N-289.
- `TestRegion_EqualsSnapshotFilter` (for every region and every module on: `Region(g)` equals the rows of `Snapshot()` with that region) - FR-INT-054; FR-OBS-044.
- `TestWarnings_SortedAndRestored` (sorted by path then rule; a restored world returns the same) - FR-INT-062.
- `TestQueryAllocations` (`testing.AllocsPerRun` on `Tick`, `Err`, `ConfigHash`: 0) - NFR-INT-001.

### Phase 8: tag 22, restore, fork-at-N

Goal: a world with calls in its history resumes bit-identically. Files: `integration.go` (`saveRumours`, `loadRumours`, `recordRumours`), `fanin.go` (one appended call), `checkpoint_sections.go` (fill the tag 22 row that plan 13 registered empty), `checkpoint_restore.go` (cross-checks), `integration_checkpoint_test.go`.

1. The tag 22 layout of CHK 4.2: per external `Informed` agent that has had a view, in ascending agent ID, the tick and the rumours in view order; sparse; a row at the default is corrupt. Cross-checks: the agent exists, is external and its role is `Informed`; each rumour's region and commodity exist; the count is at most `MaxRumoursPerView`.
2. `strategyName` writes the names of external agents; the restore resolves them through `strategyIndex` (Phase 3).

Tests:
- `TestScenarioS82_PlayerAndRestore` (INT-S4; body of the fork test below plus a mortal player on famine removed by a consumer spawner) and `TestFork_ExternalAgent` (via `forkAtN`, `Before` adds a player with money and stock, a sticky and a non-sticky directive, a yield shock and a closed edge; N in {0, 1, a tick inside a job}; M = 30; the continued run, the events and the second `Checkpoint()` equal the original) - FR-INT-070, 049; FR-CHK-050.
- `TestFork_RumoursKept` (with `Modules.Information` and an `Informed` external role: `Agent(a).Rumours` and its tick survive a restore; a later Work view at P2.5 leaves the stored rumours and tick unchanged) - FR-INT-049.
- `TestRestore_Tag22Corrupt` (unsorted agent, a role not `Informed`, a count above the cap, a default row) - FR-CHK-022 family, FR-INT-049.
- `TestFork_BeforeFirstStep` (a call made in construction, checkpoint, restore, the call's money is in the bucket) - FR-INT-011; INT 13.3.
- `TestScenarioS81_CallSequence` (INT-S3, `scenarioSeedCount()` seeds; the body of `TestDeterminism_CallSequence`: one call script run twice, in two worlds, equal state and events at every tick; two orders of calls on different keys leave equal state; a script with `AddAgent` shows `initial` in `BoundaryFlow` and passes the audit) - FR-INT-008, 009, 050; INT-S3.
- `TestScenarioS79_YieldShockCalls` (three `SetYieldShocks` calls, 3 000 for 60 and 120 ticks, exact checks only; INT-S1 bands are plan 19's) - FR-INT-040.
- `TestObservers_DoNotChangeCalls` (a recording observer and a capturing logger give the same state and snapshots as neither) - FR-INT-050, 051, 052.

### Phase 9: replace the seam, S01 leg (b)

Goal: the scenario helpers use the public calls and leg (b) runs. Files: `export_scenarios_test.go` (deleted), `scenario_helpers_test.go`, `scenario_s01_test.go`, `scenario_s04_test.go` (only if it calls the seam), `scenario_s28_test.go`, `scenario_s29_test.go`, `scenario_s48_test.go`, `scenario_s52_test.go` (each only where it calls the edge-status seam), `testdata/chains/scripts/s01b.toml` (new), `testdata/chains/expected/one.scenarios.txt`. Plan 1 has already changed these files for the race-aware seed count (`scenarioSeedCount`, `updateAllowed`); this phase edits after it and after plan 8's pack.

1. `applyScript` calls `World.SetYieldShocks` (by the ID-to-index lookups it already has) and no longer imports the seam; `SetYieldShockForTest` and its file are deleted. `TestScenarioS01` and `TestScenarioS04` must produce the same series as before (the shock is the same state).
2. `scriptFrom` accepts `[[buyer]]` beside `[[shock]]` (tick, region, money, `buy = {commodity, quantity_share_bp, ticks}`) as REF 7.1.3 writes it. A `scriptStep` has a kind. A script with a table of neither kind is still an error.
3. A new `s01b.toml` holds the leg (b) buyer: tick 400, region `north`, role `visitor`, money 2 000 000, `buy = { commodity = "fuel", quantity_share_bp = 500, ticks = 60 }`. `s01.toml` (leg (a)) is not changed: it keeps its `[[shock]]` tables only, so leg (a)'s series are unchanged (item 1). The legs run separately (REF S01).
4. The harness applies a buyer at its tick: reads the mean volume of the market in the 100 ticks before (the observer already keeps `Volume`), computes the quantity as the share of it (at least 1), calls `AddAgent`, then `Direct` with a sticky directive `[{Commodity: fuel, Side: Bid, Quantity: q, Limit: the ceiling of the commodity's reference band}]` and a `Hold` of the visitor's other tradable commodities so that the fallback adds nothing; after `ticks` steps it calls `Direct` with an empty directive. The limit is the ceiling that `marketInterval` already restates, so the published price stays inside the band (FR-STB-008); the value is the harness's, not the library's (Phase 1 asks REF to say so).
5. Leg (b) asserts what REF S01 states for both legs: exact, no published price leaves its belief interval (FR-STB-008); and the O check for demand, that the mean fuel price over the 60 ticks exceeds P0 (the direction of the shock; REF states the O sign only for leg (a), so the leg (b) sign is a prior recorded and raised in Risks, asserted from the first reviewed run). Reported: recovery time, long-run offset, unmet need, with the band keys `s01.*` the same as leg (a) (a key with no value asserts nothing). Plus the INT checks: the visitor's money enters by `initial` in the `BoundaryFlow` of tick 400 and the audit passes.
6. `one.scenarios.txt` gains the leg (b) report, written by the test's own `-update` flag (refused under `-race`, plan 1 D228); the PR gives the testing.md 7.2 explanation: new section only.

7. Plan 3's two `export_test.go` seams, `SetEdgeStatusesForTest` and `RelocateForTest`, are deleted (they would otherwise clash by name with the public methods this plan adds), each only after its last user is switched to the public call. Edge-status users: S28 and S29 (plan 3), the closure leg of S48 (`scenario_s48_test.go`, plan 13) and S52 (`scenario_s52_test.go`, plan 16; its public leg is item 3 of Phase 10) go to `SetEdgeStatuses`. `Relocate` users: `TestDrops_Relocation` (S57, plan 10) and any other test that calls the seam go to `Relocate`. Any leg skipped for want of the public call is un-skipped.

Tests: `TestScenarioS01_FuelSupplyShock` (leg b subtest no longer skipped), `TestScript_Buyer` (the loader and the quantity rule on a fixed series), `TestScript_RejectsUnknownTable`. Requirement lines: FR-STB-018, FR-STB-008, FR-REF-021, FR-REF-020, FR-AGT-053 to 055, FR-INT-025 to 027.
Exit check: S01 passes on the seeds the race-aware helper runs; the checked-in report differs only by the new section.

### Phase 10: the public legs of module scenarios

Goal: each module scenario that waited for a public call has one owner, here. Files: `scenario_s36_test.go`, `scenario_s45_test.go`, `scenario_s52_test.go`, `scenario_s66_test.go`, `scenario_s80_test.go`, `scenario_s83_test.go` (the file of the owning plan if it exists; else new), the scripts they load. Every test runs on `scenarioSeedCount()` seeds and names the failing seed.

1. S36 (MON-S4): the public `SetFlowScales` leg on the money-supply overlay (`faucet.toml`) <- plan 6.
2. S45: the public `SetYieldShocks` leg on `overlay/resources.toml` (the blight and the drought) <- plan 9 D398.
3. S52: the public `SetEdgeStatuses` leg on `overlay/external.toml` in place of plan 3's internal form <- plan 16 D539.
4. S66: the closure leg, `SetEdgeStatuses` closing the edge of a journey on `overlay/migration.toml` <- plan 14.
5. S80 (INT-S2 as the test of INT-A1, without bands): a closed road and its reopening on `three` with `trade-three.toml`; asserts the event order, the price gap opening and closing in direction, and `Slots()` empty after the reopening. The bands are plan 19's.
6. S83 (INT-S5): the forced move of a household on a journey, one test with `TestRelocate_OnJourney` as its body, run under this name on the seeds <- plan 14.

Tests: `TestScenarioS36_PublicFlowScales`, `TestScenarioS45_PublicShocks`, `TestScenarioS52_PublicClosure`, `TestScenarioS66_ClosureDuringJourney`, `TestScenarioS80_ClosedRoad`, `TestScenarioS83_ForcedMove`. Requirement lines: FR-INT-041, 042, 043, 045, 031 and the module requirements each scenario names.
Exit check: all six pass on the seeds of the race-aware helper; no report file moves except by `-update`.

### Phase 11: exit gate

Goal: the row is done. With the other agents' trees quiet:
- `gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go mod tidy` no diff.
- `go test -short -race ./...`; then gates R and F exactly as plan-2-1 D229 states them (F is the wide pattern over `./...`).
- Coverage at least 90% for the root and for `internal/ledger`.
- `lizard -l go -C 10 -a 8 -L 100 -w integration.go agent.go market.go snapshot.go internal/ledger/<file with FundFits>`. The validator is table-driven so no function exceeds CCN 10; `integration.go` splits into `integration_slots.go` and `integration_agents.go` if it passes about 500 lines of code (N-243) or fails Lizard, after the supervisor has added them to `architecture.md` 2.6 and `AGENTS.md` (Phase 1).
- `gomu` for `internal/ledger` only, in Docker with `--cpus=4 --memory=16g --memory-swap=16g` and `GOMAXPROCS=4` (see `~/.claude/CLAUDE.md`); the root is not mutation-tested (not an `internal/` package).
- `testdata/golden/off/` byte-equal to `HEAD` and `TestHashConfig_Reference` unchanged; `git diff --stat -- testdata/golden/off` empty.
- The registry and catalogue tests with `YieldShocked` deleted from `notBuilt*` (tag 22 left `notBuiltSections` in plan 13).
- Fuzz: `go test -fuzz=FuzzAddAgent -fuzztime=30s` and the same for `FuzzDirect`, `FuzzSetters`.
- The fork-at-N tests run in the matrix of plan 2 D258.

## Catalogue appends

| File | Append | Position and id | Fixed by |
| --- | --- | --- | --- |
| `config.go` | none: INT has no `Modules` field | none | INT 10 |
| `bounds.go`, `defaults.go`, `validate.go`, `hash.go` | none: every bound is the owner's (`MaxYieldBP` exists) | none | INT 5.2.6, 10 |
| `errors.go` | `ErrModuleOff`, `ErrAgentRemoved`, `ErrNotExternal`, `ErrNotDirected`, `ErrNotOnRole`, `ErrNotTradable`, `ErrUnknownLine` | after the last sentinel present, in this order | INT 11.2; architecture.md 2.10.5 |
| `event.go` | `YieldShocked{Tick, Region, Recipe, BeforeBP, AfterBP}` and its `isEvent` | after the last event present | OBS 4.1 row (core); `deferredCore` |
| `option.go` | none (the name table is in `world.go`) | none | D548 |
| `tick.go` | `{"P1.9", "AGT", line}` after P1.8 (before P1.9a); `w.p19Directives()` after the P1.8 call under `len(externals) > 0` (and the guarded P1.9 rows in `poshash_internal_test.go` `replayRows` and in `observe_internal_test.go` `stepSites`, each after the P1.8 row; the `stepSites` call keeps the guard, so the allocation test holds with no external agent); delete `"P1.9"` from `notBuiltPositions` | the table of spec-world.md 6 | spec-world.md 6; architecture.md 2.10.3 |
| `snapshot.go` | `RegionState` | with the snapshot types | OBS 5.3.5a |
| `ledger.go` / `internal/ledger` | `FundFits` (read-only) | with the ledger's probes | D547 |
| `random.go` | `siteAddAgent = 0xE00949043F800843` | after `siteOutsideWalk` (plan 16) | spec-world.md 7.1.4 (after Phase 1 edit) |
| `checkpoint_sections.go` | fill the tag 22 row that plan 13 registered empty (`present` = `Modules.Information != nil`; 22 already left `notBuiltSections`); add the cross-checks in `checkpoint_restore.go` | tag order | CHK 4.2 tag 22; plan-2-2 D243; plan-2-13 D476 |
| `fanin.go` | `recordRumours(v, a)` in the `kind == viewTrade` branch of `fillAgentView`, after `fillRumours`; nothing in `removeAgent` or `relocationRefusal` | appended | architecture.md 2.10.4 |
| log keys | none new | none | INT 12.2 |
| metrics | none new (`economysim.agent.count`, `household.count`, `household.relocations` already count these events) | none | INT 12.3 |
| checkpoint tag | 22 (no hash field id) | 22 | CHK 4.2 |

## Cross-plan contract

Consumes:

- `Checkpoint`, `RestoreWorld` (checkpoint.go) <- plan-2-2
- `checkpointSections`, `notBuiltSections` (checkpoint_sections.go), `checkpointCross` (checkpoint_restore.go) <- plan-2-2
- `codec.Writer`, `codec.Reader` (internal/codec) <- plan-2-2
- `strategyName(a)` (checkpoint_save.go) <- plan-2-2
- `Stateful`, `Directed.SaveState`, `Directed.LoadState` (strategy.go) <- plan-2-2
- `forkAtN`, `forkCase` (checkpoint_fork_test.go) <- plan-2-2
- the tag 22 row registered empty (`checkpoint_sections.go`) <- plan-2-13
- `scenarioSeedCount`, `updateAllowed` (scenario_helpers_test.go) <- plan-2-1
- `relocate`, `moveAgent`, `World.RelocationRefusal` (regions_relocate.go), `relocationRefusal(a, skip)` (fanin.go) <- plan-2-3
- `setEdgeStatuses(changes, cause)`, `EdgeStatus` (regions_status.go) <- plan-2-3
- `SetEdgeStatusesForTest` and `RelocateForTest` (export_test.go) <- plan-2-3
- `scenario_s48_test.go` (S48 closure leg) <- plan-2-13; `scenario_s52_test.go` (S52) <- plan-2-16
- `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation`, `ErrRegionBound`, `AgentRelocated`, `EdgeStatusChanged` (errors.go, event.go) <- plan-2-3
- the non-default edge-status reader (`regionState` unexported reads, regions_status.go) <- plan-2-3
- `agentJoined(a, region)` (fanin.go) <- plan-2-5
- `demandState.carry` (demand.go) <- plan-2-5
- `setFlowScale(i, bp, cause)`, `ErrUnknownFlow`, `MoneyFlowScaled`, `Modules.MoneySupply` (moneyflow.go, errors.go, event.go, config.go) <- plan-2-6
- `resourcesState.setCapacityScales`, `setRegenScales`, `adjustEndowments`, `ErrUnknownResource`, `ResourceID`, `EndowmentAdjusted` (resources_tick.go, errors.go, event.go) <- plan-2-9
- `(*subsistence).drop(slot)`, `(*subsistence).agentFields(slot)` (subsistence.go) <- plan-2-10
- `setPopulationScale(region, kind, bp)`, `PopulationScaled`, `ScaleKind`, `Modules.Population` (population.go, event.go, config.go) <- plan-2-11
- `setTransitFactor(from, to, mode, bp, cause)`, `World.Away` (transport_view.go) <- plan-2-12
- `TransitFactorChanged` (event.go) <- plan-2-12
- `ErrUnknownMode` (errors.go) <- plan-2-12
- `AgentView.Rumours`, `Rumour`, `Role.Informed`, `MaxRumoursPerView`, `viewKind` with `viewWork`, `viewTrade`, `fillRumours` (strategy.go, fanin.go, information.go, config.go, bounds.go) <- plan-2-13
- `(*migration).onJourney`, `(*migration).cancelJourney`, `ReasonRegionChanged` (migration.go, event.go) <- plan-2-14
- `ReasonOpenContract` through `relocationRefusal` (fanin.go, contract.go) <- plan-2-15
- `applyShocks`, `OutsideShocked`, `ErrOutsideNode`, `ReasonOutsideNode`, `World.isOutsideNode`, `Modules.External` (external.go, event.go, errors.go, world.go, config.go) <- plan-2-16
- `Snapshot.Endowments`, `EndowmentRow` <- plan-2-9
- `Snapshot.Shipments`, `ShipmentRow` <- plan-2-12
- `Snapshot.Journeys`, `JourneyRow` <- plan-2-14
- the rebalanced pack `one` (testdata/chains/) <- plan-2-8
- the overlays the module-on slot and query tests load: `overlay/resources.toml` <- plan-2-9, `overlay/population.toml` <- plan-2-11, `overlay/external.toml` <- plan-2-16, `overlay/trade-two.toml` <- plan-2-3, the money-supply overlay (`faucet.toml`) <- plan-2-6, `overlay/migration.toml` and the pack of S61 <- plan-2-14
- `removeAgent`, `fundInitial`, `newAgent`, `drawCreation`, `fillSeeds`, `reseedBeliefs`, `abandonJob`, `marketState`, `snapRegion`, `guard`, `Directed`, `isDirected`, `yieldShocks`, `Directive` <- Stage 1

Plans 5, 10, 13, 14 and 15 are named although the Needs column omits them: the added agent joins the demand carry, the state reads of `AgentState` and the forced move's drop, cancel and refusals belong to them; plan 13 supplies the rumour source (`AgentView.Rumours`, `fillRumours`), the tag 22 row registered empty and `viewKind` that `recordRumours` is guarded by.

Provides:

- `NewAgent`, `StockItem`, `AddAgent`, `RemoveAgent`, `Direct` (integration.go) -> plan-2-19
- `NewAgent`, `AddAgent`, `Direct` (integration.go) -> plan-2-24
- `SetYieldShocks`, `YieldShock`, `SetEdgeStatuses`, `SetTransitFactors`, `SetFlowScales`, `SetBirthScales`, `SetDeathScales`, `SetCapacityScales`, `SetRegenScales`, `AdjustEndowments`, `SetOutsideShocks` (integration.go) -> plan-2-19
- `SetYieldShocks`, `YieldShock` (integration.go) -> plan-2-20
- `SetYieldShocks`, `SetEdgeStatuses` (integration.go) -> plan-2-24
- `SlotState`, `Slots`, `Market`, `Region`, `RegionState` (integration.go, market.go, snapshot.go) -> plan-2-24
- `Relocate` (integration.go) -> plan-2-18
- `AgentState`, `Agent`, `Agents` (agent.go) -> plan-2-18
- `Agent`, `Direct`, `SetYieldShocks`, `SetFlowScales`, `SetEdgeStatuses` (integration.go, agent.go) -> plan-2-19
- `AddAgent` (integration.go) -> plan-2-18
- `AgentState`, `Agent`, `Agents` (agent.go) -> plan-2-24
- `Warnings` (world.go) -> plan-2-24
- `YieldShocked` (event.go) -> plan-2-22
- the scripted-call loader with `[[buyer]]`, the public-call `applyScript` (scenario_helpers_test.go) and `scripts/s01b.toml` -> plan-2-19
- the tag 22 section (integration.go, checkpoint_sections.go) -> none (named by the checkpoint matrix of plan-2-2 only)

`enter`, `slotCase` and the conformance table (Phases 2 and 5) are provided to the Stage 3 module plans, which append their rows; they are not Stage 2 plans, so they have no line above.

## Testing

Requirement-to-test table (testing.md 11). A requirement whose parts wait for a Stage 3 module is covered for the part that exists and the rest is named.

| Requirement | Tests |
| --- | --- |
| FR-INT-001 | `TestEnter_Order`, `TestInStep_Calls` |
| FR-INT-002 | `TestAddAgent_Errors`, `TestDirect_Errors`, `TestSlots_Conformance`, `TestRelocate_RefusalOrder`, the three fuzz targets |
| FR-INT-003 | `TestKeyedOnce`, `TestSlots_Conformance` |
| FR-INT-004, FR-OBS-017 | `TestSlots_EventsCanonicalOrder` (the `policy` half is PLC's) |
| FR-INT-005 | `TestArgError_Path`, the error tables |
| FR-INT-006 | `TestEnter_Order`, `TestSlots_Conformance` (module-off rows) |
| FR-INT-007, 008 | `TestAddAgent_LastID_AndKeyedDraws`, `TestScenarioS81_CallSequence` |
| FR-INT-009 | `TestAddAgent_Creates`, `TestAddAgent_BoundaryFlow`, `TestRelocate_Effects` |
| FR-INT-010 | `TestDirect_AffordableNotChecked` |
| FR-INT-011 | `TestAddAgent_BoundaryFlow`, `TestFork_BeforeFirstStep` |
| FR-INT-020, 021, 022, 023, 024; FR-SCL-013 | `TestAddAgent_Errors` (the `ErrNoRoom` row waits for STO), `TestAddAgent_Creates`, `TestAddAgent_LastID_AndKeyedDraws`, `TestStrategyName_Directed`, `TestExternal_Excluded` |
| FR-INT-025 to 028 | `TestDirect_Errors` (the `Borrow`, `Lend`, `Offer`, `Hire` parts wait for CRD, EMP), `TestDirect_ReplaceAndEmpty`, `TestDirect_OneTickUnlessSticky`, `TestDirect_NoJobLeavesWork` |
| FR-INT-029, 030 | `TestRemoveAgent_Order`, `TestRemoveAgent_EstateOverflow`, `TestAgent_State` |
| FR-INT-031 | `TestRelocate_RefusalOrder`, `TestRelocate_Effects`, `TestRelocate_OnJourney` |
| FR-INT-032, 048 | `TestSlots_DefaultClears`, `TestSlots_RestoredEqualsOriginal` |
| FR-INT-033 | `TestAgent_State` |
| FR-INT-040 (S79) | `TestScenarioS79_YieldShockCalls` |
| FR-INT-040 to 047 | the nine per-slot tests of Phase 5 (FR-INT-044 for birth and death; 045 capacity and regen; 046 adjust) |
| FR-INT-049 | `TestFork_RumoursKept`, `TestRestore_Tag22Corrupt` |
| FR-INT-050 to 052 | `TestObservers_DoNotChangeCalls`, `TestDirect_OneTickUnlessSticky` (Debug line) |
| FR-INT-053, 054 | `TestMarket_Fields`, `TestRegion_EqualsSnapshotFilter` |
| FR-INT-060, 061, 062 | `TestQueries_NoDrawNoStateNoEvent`, `TestQueries_Guards`, `TestAgents_Region`, `TestWarnings_SortedAndRestored` |
| FR-INT-070 | `TestFork_ExternalAgent`, `TestScenarioS82_PlayerAndRestore`, `TestSlots_RestoredEqualsOriginal` |
| NFR-INT-001 | `TestQueryAllocations` |
| NFR-INT-002 | `TestStep_AllocationsWithExternalAgent` |
| NFR-INT-003 | `import_guard_test.go` and `determinism_guard_test.go` (no goroutine, no `time.Now`, no package state) already scan every non-test file; `integration.go` is covered by them |
| NFR-INT-004 | the conformance table includes a call at the configured maximum of keys; work is bounded by the slice |
| NFR-INT-005 | `FuzzAddAgent`, `FuzzDirect`, `FuzzSetters` |

Golden files: `testdata/golden/off/` unchanged (proved by `git diff`), no module-alone golden, no all-on golden moves (no module). Fork-at-N: `TestFork_ExternalAgent` in plan 2's matrix. Registry and catalogue rows: `YieldShocked` and tag 22 leave the not-built lists. Scenarios: S01 leg (b) runs; INT-S3 (`TestScenarioS81_CallSequence`), INT-S4 (`TestScenarioS82_PlayerAndRestore`, with `TestFork_ExternalAgent`: a player, a directive and a shock survive a restore and the result equals an uninterrupted run; a non-mortal player at hunger stays, a mortal one is removed by a consumer spawner) and INT-S5 (`TestRelocate_OnJourney`, `TestRelocate_OpenContract`) are the tests named; INT-S1 and INT-S2 are plan 19's.

## Observability

Entry for the Reporting section (observability-and-logging.md 9): one new state transition has an event, `YieldShocked` (a past-tense name, plain data, fixed-point, no map, constructed only with an observer, core gate); the other events a call emits are the owners' and are listed in Decision D556. New failure paths are returned errors (the seven sentinels) and are not logged (architecture.md 8.3). Logs: Debug only, guarded by `Enabled`, no new key, no `Info`, `Warn` or `Error` (D557); `removeAgent`'s "agent bankrupt" is reused for `RemoveAgent`. No new decision point of a strategy; the one new loop (P1.9) is over the external agents only, once per tick, and emits nothing. Metrics: none new; `economysim.agent.count`, `household.count` and `household.relocations` already count these events, and a shock is state read from `Slots()`. The tests of observability-and-logging.md 8.1 are `TestObservers_DoNotChangeCalls` and the event tests of Phase 5.

## Risks and open items

| # | Item | Owner |
| --- | --- | --- |
| 1 | The setter forms differ per owner (D542). Plan 9 gives no signatures; plan 10's Provides line for `drop(slot)` names plan 14 only; no plan declares the entry types `FlowScale`, `RegionScale`, `EndowmentScale`, `EndowmentAdjustment`, `TransitFactor` (plan 6 takes an index and a value, plan 12 four scalars, plan 11 a kind), so INT declares them with `YieldShock` and `OutsideShock` (plan 16 expects that), as spec-integration.md 20 says; the owners' forms take primitives. If a plan declares one, this plan drops its copy. | supervisor, plans 9, 10 |
| 2 | Plan 11 emits nothing for a population scale and INT emits `PopulationScaled` (plan 11 declares the event and `ScaleKind`); plan 16 returns events for INT to deliver. Written as the plans say; the tests of those events' fields are INT's here. | plans 11, 16 |
| 3 | `Region(r)` cost: module rows without a per-region appender are filtered from a full snapshot (class 3, not 2). FR-INT-054 holds; the class promise of 6.1.3 does not for those modules until their plans give an appender. | plans 9, 12, 14 |
| 4 | The buyer's limit in REF 7.1.3 is unspecified; the harness uses the ceiling of the reference band. REF should say so (Phase 1). Leg (b) has no O sign in REF; the plan asserts that the price over the shock exceeds P0 only after a human has reviewed a first run (FR-REF-024), until then it reports. | REF, plan 19 |
| 5 | INT-S3 says "golden"; testing.md 7.1.1 has no directory for a call-script golden. This plan compares runs and creates no file. A `testdata/golden/calls/` needs a rule edit first. | supervisor |
| 6 | `RemoveAgent` can fail on the estate's balance bound after changing the job (D547); INT 4.4 says it never refuses. The probe returns `ErrOverflow` before any change. The spec should name the case. | INT |
| 7 | An outside node has no recipes, so a yield shock on one is meaningless; INT 5.1.3 lists no `ErrOutsideNode` for it (Phase 1 edit). | INT |
| 8 | Tag 22 is registered empty by plan 13 when `Information` is on and filled by this plan, so between them a world with `Information` on writes tag 22 with zero rows, and the checkpoint matrix of plans 13 to 16 expects 22 in its tag list (plan-2-13 D476). | plans 13 to 16 |
| 9 | Plan 13 registers tag 22 empty (zero rows) when it turns `Information` on, so the checkpoint bytes and the checkpoint digest of the module-alone goldens of INF, MIG and MER do not move when this plan fills it with rows. A world with no external agent of an `Informed` role that has had a view still writes zero rows. | plans 13, 14, 15 |
| 10 | `agent.live` is a field on the agent type; N-244 puts a module's state in the module's own slices. It is core AGT state (a `Directive` is core), not a module's; if the rule is read strictly, the field becomes `w.tk.live[slot]`. | supervisor |
| 11 | Unspecified: a `Direct` on an external agent of a role with no tradable commodity; every intent is `ErrNotTradable`, which is the right answer and needs no change. A directive for a recipe with the own-use list needs SUB's effective list (plan 10): `ErrNotOnRole` uses the role's list until then. | plan 10 |
| 12 | The matrix cost of `TestFork_ExternalAgent` under `-race` is not measured; the focused run is the gate until plan 1's gate exists. | plan 1 |

Note proposals (text only; the supervisor is the only editor of `doc/notes.md`, N-103):

- **N-new-a (areas INT, WLD, CHK).** "INT 3.5.2, 4.1.2, 4.1.3, 9.1, 14.1, 14.2 and INT-S3 predate ADR 0008: the creation draws of `AddAgent` are keyed `(siteAddAgent, Tick(), id)` and independent of call order. `siteAddAgent` = `0xE00949043F800843` is appended in WLD 7.1.4 and `random.go`."
- **N-new-b (areas INT, REF).** "S01 leg (b): the buyer's limit and the O sign of the demand leg are not in REF 7.1.3 or S01. The harness uses the ceiling of the reference band and reports the sign until reviewed."
- **N-new-c (areas INT, OBS).** "`Region(r)` is the filter of the snapshot; every module that adds rows to `RegionState` should expose a per-region appender so that the query stays class 2 (INT 6.1.3)."
- **N-new-d (areas INT, LED, AGT).** "`AddAgent` and `RemoveAgent` need a no-effect probe for the bound they can hit (`Ledger.FundFits`, `estatefits`) because the ledger has no undo; a later call that moves money between accounts at the call (`TransferTitles` with a price, CAP/LND) needs the same."
- Close on merge: N-296 and N-289; N-290 INT part settled as above.
