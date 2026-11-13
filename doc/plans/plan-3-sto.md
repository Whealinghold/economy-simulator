# Plan: Stage 3, STO, `storage.go` and `internal/room`

**Status: plan written (2026-11-13), not audited. Stage 1 of the Stage 3 work (requirements and plan, no code). Written on branch `s3-sto`, cut from the integration branch `phase-3` (= `main` at `f556398`). Modelled on `plan-2-10.md` (SUB), read from git history (`31e3a05^`), the closest merged Stage 2 module in shape: a module file, one `internal/` package, hooks in production and the default strategy, a module-alone golden, a fork test.**

Rule tags read (all three files in full, as `AGENTS.md` asks): architecture=bdb8a4b7fa1ed8fd3b2e7aec, observability=f215753306fac0bc7607c4c4, testing=66af7645b01fef4e2796c367.

Notes tag (`doc/notes.md`, read for this plan, which writes no spec): ffa06a0b8159c72f08471dca. Read in full: the notes tagged STO (N-140, N-155, N-160, N-166, N-181, N-186, N-191, N-194, N-205, N-221, N-244, N-327) and the ALL notes that bind a module's code (N-335, N-340, N-345, N-351, N-353). The other ALL notes (N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217, N-315) were read through the places `spec-storage.md` 19.1 cites them, not re-read in full; the audit of this plan should re-read them.

Specs and requirements implemented: [spec-storage.md](../specs/spec-storage.md) in full (FR-STO-001 to FR-STO-046, NFR-STO-001 to 004); the STO parts of [spec-config.md](../specs/spec-config.md) 3.1, 3.7.1, 8.2.2 and FR-CFG-085; [spec-checkpoint.md](../specs/spec-checkpoint.md) 4.2 tag 3 (hosted) and tag 13 (the stores-target carry), 6.2; [spec-ledger.md](../specs/spec-ledger.md) (room, `RoomLeft`, the audit identities); [spec-observability.md](../specs/spec-observability.md) 4.1 (`RoomLimited`, `StoreChanged`, `AgentSpawned.StoresLocked`), 5.3, 5.4. Also read: [spec-world.md](../specs/spec-world.md) 6 (P1.3, P2.1, P2.5, P2.7, P3.2, P3.3, P3.6a, P4.1, P7.1), [spec-clearing.md](../specs/spec-clearing.md) 3.2 and 3.3, [spec-production.md](../specs/spec-production.md) 4.2, 4.3, 5.3, [spec-transport.md](../specs/spec-transport.md) 5.4, 5.5, [spec-agents.md](../specs/spec-agents.md) 6.4.1, 7.3, 7.5, [spec-money-supply.md](../specs/spec-money-supply.md) 4.1, 4.3, 12.1, [spec-regions.md](../specs/spec-regions.md) 6.5.2, [spec-reference-chain.md](../specs/spec-reference-chain.md) 4.4 (the `overlay/storage.toml` row) and 8 (S91 to S96).

## Scope

This plan builds the storage module: an agent's account may carry a room (a pair, mass and bulk) that is a base room per household of its role plus the capacity of its stores; room is committed where a decision is made (a bid, a job start, a dispatch), so that goods are never lost for lack of room after the decision; the default strategy respects the room; the cost of a store is a money flow of the existing kind `stores`.

| In scope | Where |
| --- | --- |
| `Storage`, `Capacity`, `RoleRoom`, `StoreClass`, `Modules.Storage` (hash id **12**), `StoreClassID`, `DefaultStorage()` (none: no tunable has a non-nil default), the bounds `MaxRoom`, `MaxStoresPerAgent`, `MaxStoreClasses`, the sentinels `ErrStoreShape`, `ErrRoomShape`, `ErrNotLimited`, `ErrUnknownStoreClass`, `ErrStoreLocked`, `ErrAgentAway`, `ErrStoreInUse`, `ErrNoRoom` (root) | `config.go`, `hash.go`, `bounds.go`, `defaults.go`, `errors.go`, `validate.go`, `config/mapping.go`, `value.go` |
| The module: the resolved base room per role, the stores of each agent (module slice, N-244), `AddStore`/`RemoveStore` (unexported `addStore`/`removeStore` with the forced form), the pledge derivation, the per-position acts, the view filler, the `TradePlan` checker, the refusal `has_store`, the warnings, the snapshot rows, section 15 | `storage.go`; `storage_validate.go`, `storage_report.go`, `storage_commit.go` only if Lizard forces the split (new, D3) |
| The room of an account: `Room`, `SetRoom`, `SetPledge`, `ReserveRoom`/`ReleaseRoom`, `FreeRoom`, `RoomLeft`, the three credit forms, the load maintained by every lot operation, `ErrNoRoom`, the audit identities | `internal/ledger/` (new files) |
| The pure rules: the sufficient bound `k_room` (5.3.2), the target scaling `f` (6.2.2), the discard order (6.3.2), the excess order of a role change (5.5.2.1), the stored load (8.3.2), the room of an account | `internal/room/` (new) |
| The `no_room` row of admission and the reservation of a bid | `internal/clearing/admit.go` (the marked position), `clearing.go`, `phase_trade.go` |
| `k_room` in the view and in the start, the pledge at start, completion and abandon, the `no_room` refusal, the creation of the output from the pledge | `production.go`, `phase_production.go`, `strategy_work.go`, `strategy_view.go` |
| The dispatch check `consignee_full` (row 11a), the `has_store` row (3a), the pledge at dispatch and the credit of an arrival from the pledge with the divert | `transport_request.go`, `transport_arrival.go`, `transport_state.go` |
| The room group of the view, the scaling of the targets, the bids sized to the free room, the discard | `strategy.go`, `strategy_view.go`, `strategy_default.go`, `strategy_work.go` |
| The cut of the room: weight reduction, role change (the excess to the estate under `estate_goods`, the stores ended), the warm job's pledge at construction | `phase_exit.go`, `fanin.go`, `estates.go`, `construct.go`, `construct_warm.go` |
| The target kind `stores` (the field `FlowTarget.Store`, an amount per store, `ErrUnknownStoreClass`, the carry in tag 13) | `moneyflow.go`, `internal/moneyflow`, `config.go` |
| `AddStore`, `RemoveStore`, `AgentState` room group, the sentinels | `integration.go`, `integration_slots.go` |
| `RoomLimited`, `StoreChanged`, `AgentSpawned.StoresLocked`, the reasons `no_room`, `consignee_full`, `has_store`, `RoleRow` room fields, `StoreRow`, the log keys `store_class` and `store_classes` | `event.go`, `snapshot.go`, `internal/logkeys` |
| The checkpoint: the stores in the agent record (tag 3, hosted), the stores-target carry in tag 13, the restore checks, the rebuilt pledge | `checkpoint_save.go`, `checkpoint_restore.go`, `checkpoint_sections.go` |
| `overlay/storage.toml` (and a `three`/`two` variant for S94), `testdata/golden/storage/`, the registry, catalogue and position rows, scenarios S91 to S96, an all-on layer | `testdata/`, tests |

| Out of scope | Owner |
| --- | --- |
| `StoreClass.Needs` live (the settlement flag, `ErrUnknownSettlementFlag`, `regionHas`, `store_locked_at_start` with a non-zero count) | SET (`settlements.go`). Built here as the field, the rule `needs_module_off` (`ErrModuleDependency`, always true while `Modules.Settlements` does not exist) and a nil gate; D8 |
| Who builds a store, `Cause` `facility`, `ErrStoreInUse` against installed facilities, P8.3b | CAP (`infrastructure.go`). Here: the internal `addStore`/`removeStore` with the forced form, the constant `facility` and `ErrStoreInUse` declared, the facility count passed as 0; D9 |
| `StoreLoss`, the `loss` sink, the stored-load read by risk, the `Losses` preset of `indicator/` (FR-IND-059) | RSK (N-327). This plan exposes `storedLoad` (8.3.2) and builds no preset |
| A storage-use measure of `indicator/` (N-130) | asked of IND; not a preset of the 25 and not built here (Risk 7) |
| A treasury as a limited holder, a buffer store held by a ruler, the release as a policy | FIS, PLC (N-186, N-140) |
| Renting room, a keeper, a lease; preservation (a store that slows ageing); a per-unit price of space; a regional base room | not in version 1 (spec open questions 1, 2, 3, 7) |
| The metrics `economysim.storage.limited`, `economysim.store.count`, `economysim.room.fill` and their adapter | `otelobserver/` (the row in the observability catalogue exists; the adapter change is a separate module PR after this one, D16) |
| Collateral over room (N-166), a charter over a store (N-194) | CRD, ORG |

## Decisions

- **D1. Hash id and tag.** `Modules.Storage` is the 12th pointer in the row of `spec-config.md` 3.1 (`Settlements` 11, `Storage` 12, `Employment` 13: `hash.go` already writes `Employment` as id 13). Its hash id is **12** whatever the build order, and `Settlements` (11) is not built yet, so the Go line of the field is appended after the pointers that exist (architecture.md 2.10.5; `pairsLaterModules` in `hash.go` is the model). The module has **no checkpoint section of its own**: the stores are written once in the agent record (tag 3, hosted, N-128), the stores-target carry is MON's tag 13, and the format version does not change (`spec-checkpoint.md` 9.3, the paragraph on format 1). Inner ids: `Storage` `Base` 1, `Roles` 2 (`RoleRoom`: `Role` 1, `Base` 2, `Stores` 3), `Classes` 3 (`StoreClass`: `ID` 1, `Mass` 2, `Bulk` 3, `Needs` 4; `Label` has none), `Capacity` `Mass` 1, `Bulk` 2, `FlowTarget.Store` appended at the end of `FlowTarget`'s ids. Written in the explicit form; the audit checks them against `spec-config.md` 8.2.2, which today lists no inner ids for STO (Phase 1, E2).
- **D2. Per-agent state is in module slices** (N-244): `stores [][]uint8` (or a flat count per class) indexed by agent slot, grown like the other modules' slices; `agent` gets no field. A limited account's `Room` struct is the ledger's (state of the account), the resolved base room per role is a `roleRow` field (like `ownUse` in D404 of plan 2-10), filled by the role-table constructor. The pledge is not state: it is derived (FR-STO-020) and recomputed at each change by `storage.go`.
- **D3. Files.** The module file is `storage.go`. Lizard (`-C 10 -a 8 -L 100`, N-243) will likely force continuation files; if so they are `storage_validate.go` (VAL, warnings), `storage_commit.go` (bid, start, dispatch, cut acts) and `storage_report.go` (events, logs, snapshot), listed first in `architecture.md` 2.6 (fresh RULE-TAG) and `AGENTS.md` Layout, as plan 2-10 E5 did. A file that stays inside the limits is not split. `internal/room` is new and listed in 2.6 and `AGENTS.md` (the layout lists it already as the one package per module; confirm in Phase 1, E1).
- **D4. The ledger carries the invariant** (spec 4): `Room` is part of the account, the sizes are a table of primitives passed at construction (`SetSizes`), every lot operation updates the load with checked arithmetic, and an unlimited account costs one branch (NFR-STO-001). The credit forms are added as variants of the existing credit operations, not as new operations that duplicate them. `ErrNoRoom` is ledger-internal and mapped at the root (N-075).
- **D5. No new tick position, no draw, no ledger reason** (FR-STO-027, 037). The position table is unchanged except P3.6a, which `spec-world.md` 6.5 already lists and `notBuiltPositions` still names; its entry is deleted and a guarded call appended. `RoomLimited` is the only per-tick event; its counters are plain integer adds made always (the default strategy and the start sizing need them with no observer), the event is built only under `w.obs.supplied` (FR-STO-041, observability 4.5). The counters are cleared at P0.2/P8.6 like every module's (FR-STO-036).
- **D6. Seam rows that exist as marked nils** are filled, not replaced: `shipChecks` rows 3a (`has_store`) and 11a (`consignee_full`) in `transport_request.go`; the `no_room` position in `internal/clearing/admit.go` (after the funds row, as the file's comment says); the module term of `viability.go` is not touched. The `Facts` field for room is appended to `clearing.Facts` in module order (architecture.md 2.10.4); `orderFacts` resolves it under `w.sto != nil`.
- **D7. The sufficient bound of 5.3.2 and its denominator 10 000, the scaling of 6.2.2, the discard order, the excess order and the reason names are rule constants** next to their code (architecture.md 9.6). Nothing result-affecting is a literal in logic that is not a documented rule constant.
- **D8. `Needs` while SET is absent.** `StoreClass.Needs` exists as a field and is hashed; a non-empty value is `ErrModuleDependency` (`needs_module_off`) because `Modules.Settlements` does not exist in `Config`. `regionHas` is not written; the creation code calls a nil gate (`w.setGate == nil` means open), the same shape SET fills later (N-155). `StoresLocked` is declared in `AgentSpawned` and is 0 until SET. The tests for FR-STO-044 that need a settlement flag are written against a test-only gate set from an internal test and are listed as the SET plan's to finish (Risk 3).
- **D9. CAP seams.** `addStore`/`removeStore` take the cause and a `forced` flag, return the sentinels of 3.6.2, and read the count of installed store facilities from a function `w.storeFacilities(a, class)` that returns 0 until CAP exists. `ErrStoreInUse` is declared and tested only through that seam. P8.3b has no call here.
- **D10. Warm start.** The warm job's pledge is set after `construct_warm.go` item 4 and checked by `room_below_warm_job` at construction (FR-STO-045), which returns `ErrRoomShape` from `NewWorld`; a construction can never meet `ErrNoRoom`. The rule that `room_below_start_stock` is checked at `Validate` for `StartStock` and at construction for warm groups.
- **D11. Hooks are the named ones of ADR 0010 and architecture.md 2.10.1**: (a) none in a core concept file except a method added to `production.go` for `k_room`, (b) the call sites of the fan-ins, `fillAgentView` and `checkTradePlan` gain the module call, (c) `agentJoined`, `agentRoleChanged`, `dropAgent` (a store is released by a removal), the audit call, the `relocationRefusal` row `has_store` registered at construction, (d) a guarded call from `DefaultStrategy` files into methods of `storage.go`. Each is a one-line guarded call; no branch on the module's data in a core file. Any hook not on the list of 2.10.1 is a defect and the plan changes before the code does; Phase 1 E3 lists the new ones for the supervisor to approve.
- **D12. `AgentState` and the integration calls** are INT's file (`integration.go`, `integration_slots.go`) but the calls are STO's requirement (FR-STO-032, 033). They are built in this plan, in the same two files, as one guarded method each; a conflict with another Stage 3 plan that edits `integration.go` is a textual merge conflict only (Risk 4).
- **D13. Presets.** No preset of `indicator/` is built here (N-327 puts `Losses` with RSK). The tally rows for `RoomLimited` and `StoreChanged` are not needed by the 21 presets; if a preset-less tally exists it gains the two `Observe` cases only when RSK's plan lands.
- **D14. Overlay.** `testdata/chains/overlay/storage.toml` follows the row of `spec-reference-chain.md` 4.4: a base room for the households and the producers just above one tick's upkeep plus one job's load, one store class, and `[modules.storage.role.<id>]` rows. Variants (a gated class, a stores flow, a small cohort) are made in the tests by editing the loaded `Config`. A second overlay `storage-two.toml` for S94 (two shippers and one consignee) on `two` + `trade-two.toml`.
- **D15. Module-alone golden** is `testdata/golden/storage/` on pack `one` + the overlay, seed 1, 400 ticks, `WithAudit`, recorded once after the diff against `golden/off` is read. `golden/off/` and `TestHashConfig_ReferencePinned` never move (proof at every phase exit). `golden/all/` moves **once**, in the PR that appends the all-on layer (Phase 9), with the explanation testing.md 7.2 requires; if the owner prefers the all layer to wait until more Stage 3 modules land, Phase 9 is the only one dropped (question Q3).
- **D16. Adapter.** The three metrics and their attribute values are already in the catalogue of `observability-and-logging.md` 5.3 and `cardinality_test.go` has the entries; the `otelobserver` change is out of this plan's module (own `go.mod`) and is one PR after the root PR, by the same author, under that module's 80% threshold.
- **D17. Stage 3 build order.** `Needs` for STO in the roadmap says LED and GDS. The code needs more than that, and the plan lists it in Dependencies below.

## Dependencies on other Stage 3 modules

STO is built from the tree as it is on `phase-3` and needs none of the other thirteen to be correct. It leaves seams for them (D8, D9) and shares files with them (the table below). The roadmap's "Needs" column for STO (LED, GDS) is incomplete: the code also reads PRO (the job start and completion), TRN (dispatch and arrival), AGT (the view and the default strategy), CLR (admission), MON (the `stores` target), REG (the `region_bound` registry), EST/SPN/SKL (the role-change excess, rung 1 and traineeship), INT (the calls), CHK. All are merged. Roadmap row to correct in the plan's PR: `Needs` for STO reads "LED, GDS, PRO, TRN, AGT, CLR, MON, REG, EST, INT, CHK".

| Other module | Depends on STO | Seam here |
| --- | --- | --- |
| SET | `StoreClass.Needs`, `ErrStoreLocked`, `StoresLocked` | D8: nil gate, field hashed, `ErrModuleDependency` while SET is off |
| CAP | `addStore`/`removeStore`, `StoreChanged.Cause` `facility`, `ErrStoreInUse` | D9 |
| RSK | the stored load (`storedLoad`), the stores by class, `StoreRisk`, `Losses` preset | `internal/room` stored-load function; `ErrUnknownStoreClass` shared |
| FIS | a ruler's reserve stores, `RoomLimited` is not per authority | none here (N-186: a treasury is never limited) |
| EMP | none directly; shares `fanin.go`, `event.go`, `snapshot.go` | textual |
| MON | the `stores` kind is in MON's `moneyflow.go` already (`targetStores`); STO completes its shape rules and carry | Phase 6 |

## Phases

Each phase ends green on `go test -short -race ./...` and gate F on the tests it adds; `testdata/golden/off/` and the pinned hash do not move in any phase (a test run at every phase exit). The PR split below groups them.

### Phase 1: the specs are true (no code)

Goal: nothing below starts before these are applied by the supervisor (the plan itself edits no spec). Edits owed:
- **E1.** `doc.go` glossary (before any code uses the words, architecture.md 3.2): store, store class, room, base room, limited, load, pledge, reserved room, free room, upkeep room, cut of the room, stored load, buffer store, owns a store (`spec-storage.md` 19.3), and the type names `Storage`, `Capacity`, `RoleRoom`, `StoreClass`, `StoreClassID`, `StoreChanged`, `RoomLimited` (`TestGlossary_DeclaredTypesNamed`). `internal/room` is already listed in `architecture.md` 2.6 and `AGENTS.md`; no layout edit is needed unless D3 forces continuation files.
- **E2.** `spec-config.md` 8.2.2: the inner ids of D1; 3.7.1's `FlowTarget.Store` as the last id of `FlowTarget`. Check that `spec-config.md` 6.2.1 (`[modules.storage.base]`, `[modules.storage.role.<id>]`, `[modules.storage.class.<id>]`) and N-221 agree, then close N-221.
- **E3.** The hooks of D11 not already named in `architecture.md` 2.10.1 (the `Facts` field for room, the `k_room` term in `production.go`, the pledge calls at `startJob`/`finishJob`/abandon, the credit-from-pledge at the arrival) are listed in ADR 0010's table or confirmed to be (b) and (d) of 2.10.1.
- **E4.** `spec-merchants.md` 6.1 and 9.5 (19.2 owes them; the grep finds neither): a merchant whose view has stores plans no self-carried trip; `consignee_full` is a failure of the formation counted in `DispatchRefused`. `spec-indicators.md`: the storage-use measure of N-130 (ask or reject).
- **E5.** Verify the edits 19.2 lists are in the other specs (the grep finds them in LED, CLR, PRO, TRN, REG, OBS, INT, MON, CFG, VIA, SUB, SPN; not in MER, IND). Verify `spec-world.md` 6.3 P1.3, 6.4 P2.1/P2.5/P2.7, 6.5 P3.6a, 6.11, 8.3.
- **E6.** `roadmap-v1.md` STO row: `Needs` as above; status moves to "plan written" in the PR that adds this file.
- **E7.** Decide Q1 to Q5 below.

Exit check: the supervisor confirms E1 to E7; Phase 2 starts.

### Phase 2: `internal/room` and the room of the ledger (PR 1)

Goal: the pure rules and the account's invariant, with no root file touched except the catalogue test lists.
Files: `internal/room/` (`bound.go`: `KRoom`; `scale.go`: `Scale`; `discard.go`: `DiscardOrder`; `excess.go`: `ExcessOrder`; `stored.go`: `StoredLoad`; `doc.go`), `internal/ledger/room.go` (the `Room` struct, `SetRoom`, `SetPledge`, `ReserveRoom`, `ReleaseRoom`, `FreeRoom`, `RoomLeft`, `SetSizes`), and the load update and the three credit forms in the existing operations (`internal/ledger/*.go`, new files for new code where the file would pass the Lizard limit).
Tests (`Requirement:` IDs): `TestKRoom_MatchesExactLargestK` (FR-STO-012, 5.3.2, against an exact search over random recipes: soundness and the bound's tightness cases of the model), `TestKRoom_ExemptOutputMultipleOfE`, `TestKRoom_DiscardOutputTakesNone`, `TestScale_Floors` and the clamp where `Avail` is negative (FR-STO-022), `TestDiscardOrder_Tier1Only` (FR-STO-024), `TestExcessOrder_BindingDimensionFirst`, `_TiesByCommodityID`, `_WholeLotsOldestFirst`, `_ReevaluatedBeforeEachCommodity` (FR-STO-018, example 18.3 as a table case), `TestStoredLoad_Formula` (FR-STO-042); ledger: `TestRoom_CreditUnmarkedRefusedWhenNotFit`, `_FromReservedDrawsReserved`, `_FromPledgeBoundedByRoomLeft`, `_RefusedChangesNothing` (FR-STO-007), `TestRoom_LoadMaintainedByEveryOp` (FR-STO-006, a table over Move, Transform, Create, Discard, Spoil, Wear, Dispose, Estate transfer), `TestRoom_ZeroSizeTakesNoRoom`, `TestRoom_SetRoomRefusedWhenLoadExceeds`, `_SetPledgeNeverRefuses` (4.1.2), `TestRoom_UnlimitedCostsOneBranch` (an `AllocsPerRun` of 0 and a benchmark, NFR-STO-001), `TestRoom_Overflow` (a load at `MaxQuantity` times 10^6 through `internal/checked`), the worked examples of 18.1, 18.2 and 18.4 as table cases; `FuzzRoomOps` (properties: load equals the sum of lots, reserved and pledge never negative, a refused op changes nothing).
Exit check: `internal/room` and `internal/ledger` at or above 90%; `gomu` on `internal/room` only, in Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`, no surviving non-equivalent mutant; Lizard clean; the existing ledger and clearing goldens unchanged.

### Phase 3: configuration, validation, hash, loader (PR 2)

Goal: the module exists as data and is checked; it does nothing at run time.
Files: `config.go` (types, `Modules.Storage`, `FlowTarget.Store`), `value.go` (`StoreClassID`), `bounds.go` (`MaxRoom` 10^12, `MaxStoresPerAgent` 8, `MaxStoreClasses` 64), `defaults.go` (the `canonical` sort of classes by ID and rows by role; no default value), `errors.go` (the eight sentinels of Scope, after the earlier modules'), `hash.go` (`pairsStorage`, id 12, explicit form; `Label` omitted), `validate.go` (the call of `validateStorage` at the module seam in the order of the `Modules` row), `storage.go` or `storage_validate.go` (the rules of 13.1 and warnings of 13.2), `role.go` (`roleRow.room`, the resolved base room), `config/mapping.go` (`[modules.storage]`, `.base`, `.role.<id>`, `.class.<id>`, `[modules.moneysupply...target]` `store`), `internal/logkeys` (`StoreClass`, `StoreClasses`).
Tests: `TestValidate_Storage_<Rule>` one failing case per row of 13.1 asserting the sentinel with `errors.Is` (FR-STO-038, FR-CFG-085): `duplicate_id`, `unknown_role`, `store_class_shape`, `room_bound`, `unknown_store_class`, `needs_module_off` (the only form of `Needs` here, D8), `store_on_unlimited_role`, `room_below_upkeep`, `room_below_start_stock`, `stores_target_off`; `room_below_warm_job` is Phase 7. `TestWarnings_Storage` one case each for 13.2 (`no_limited_role`, `room_below_cover`, `room_below_recipe`, `store_class_unused`, `store_levy_below_upkeep`; `store_locked_at_start` waits for SET) plus the silent case on the overlay. `TestEffectiveRoom_RowReplacesBase`, `_EmptyRowMeansUnlimited`, `_NoRegionKeyedField` (reflection, FR-STO-003 and 12.1.1). `TestHash_StorageOnOff`, `_FieldIDs`, `_AfterLowerModuleIDs`, `_ListOrderIrrelevant`, `_LabelNotHashed`, `_FlowTargetStoreHashed` (FR-STO-002, FR-CFG-085). `TestLoad_StorageToml`, `FuzzLoad` corpus extended, `FuzzValidateStorage` (never panics; accepted input re-validates). Registry test: the rows of 12.4 leave `notBuilt` (`Storage`, `RoleRoom`, `StoreClass`, `Capacity`, `FlowTarget.Store`, `Modules.Storage`), both directions. `TestSentinels_*` for the eight (errors.go convention).
Exit check: off golden and pinned hash unchanged (`TestHashConfig_ReferencePinned`); registry test staged both ways; `config/` coverage at or above 90%.

### Phase 4: the account's room at run time, the calls, the region binding (PR 3)

Goal: limited accounts exist; stores can be added and removed; nothing yet commits room.
Files: `storage.go` (state `stores` slice, `newStorage` called from `construct.go` under the pointer and `modulesOn` appended after the names present, `limited(a)`, `roomOf`, `setRoom`, `agentJoined` (gives the role-row stores, with `StoresLocked` through the nil gate), `agentRoleChanged` (ends stores on a role with no base room, re-evaluates the room otherwise), `dropAgent`, `addStore`/`removeStore` with the forced form, `storedLoad`, `auditModules`), `world.go` (`sto *storage`), `fanin.go` (the module calls in `agentJoined`, `agentRoleChanged`, `removeAgent` via `dropAgent`, `relocationRefusal` row `has_store`, `AddAgent` check `ErrNoRoom`), `integration.go` and `integration_slots.go` (`AddStore`, `RemoveStore`, `AgentState` room group), `event.go` (`StoreChanged`, `AgentSpawned.StoresLocked`, the reasons), `snapshot.go` (`RoleRow` room fields, `StoreRow`), `regions_relocate.go` (a relocation carries at most the base room, FR-STO-030, 9.5), `tick.go` (no change).
Tests: `TestAddStore_<Error>` one per row of the 3.6.2 table (FR-STO-032), `TestRemoveStore_<Error>` (FR-STO-033: `ErrStoreInUse` through the seam of D9, `ErrNoRoom`, `ErrOutOfBounds`, allowed for an away agent), `TestStoreChanged_Fields_NoMap_OnlyAtCall` (FR-STO-035), `TestCreation_GivesRoleRowStores_NoEvent` (FR-STO-034), `TestRoleChange_ToUnlimitedEndsStoresInClassOrderAfterRoleChanged` (FR-STO-034, event order), `TestRoleChange_ToLimitedKeepsStores`, `TestRelocation_RefusedHasStore`, `TestRelocation_ForcedMoveCarriesBaseRoomOnly` (9.5), `TestRoom_Formula` (FR-STO-004: weight times base plus stores, a store not multiplied), `TestLimited_IffBaseRoom_PseudoAgentNever` (FR-STO-003), `TestAddAgent_NoRoom` (FR-STO-019), `TestSnapshot_RoomFields`, `TestAgentState_RoomGroup`. `TestStorage_OffLeavesNoTrace` (FR-STO-001: off golden, hash, no event, identical `Run`; and on with no role limited equals off, results only).
Exit check: `TestAllocs_*` unchanged with the module off; `TestStep_AllocationsStayAtTheBaseline` at its baseline; position golden for the off run unchanged.

### Phase 5: commitments at bids, starts and dispatches (PR 3, second half; or PR 4 if the diff passes the review size)

Goal: the invariant of 4.4 holds under trade, production and transport.
Files: `internal/clearing/admit.go` (the `no_room` row, `Facts.RoomMass/RoomBulk`, `SizeMass/SizeBulk` appended), `clearing.go` and `phase_trade.go` (reserve on admit, release at the end of P4.1, the settlement draws from reserved), `storage_commit.go` or `storage.go` (the P3.6a counters, `reserveBid`, `releaseBids`), `production.go` (`k_room` term; `no_room` last in the order), `phase_production.go` (start sizing with the upkeep room left free, pledge at start, completion from the pledge up to `RoomLeft`, create-and-dispose of the rest under `disposal`, abandon and role change drop the pledge, `MaxQuantity` overflow rule kept), `strategy_work.go`/`strategy_view.go` (`MaxExecutions` bounded by `k_room`, reason `no_room`), `transport_request.go` (rows 3a and 11a filled), `transport_state.go` and `transport_arrival.go` (the pledge at dispatch, the credit from the pledge with the divert to the estate of the destination, the pledge lowered by the whole line), `phase_exit.go` (a weight reduction: load scaled within the room, pledge recomputed, may exceed; FR-STO-017), `estates.go` (the excess move of a role change under `estate_goods`, called from the fan-in), `fanin.go` (`checkTradePlan` module call, `fillAgentView` room group), `construct_warm.go` (the warm job's pledge and `room_below_warm_job`, D10), `ledger.go` (no row; confirm none).
Tests: admission: `TestAdmit_NoRoomAfterInsufficientStock`, `_AskNeverRoomChecked`, `_PseudoAgentExempt` (FR-STO-008), `TestBid_ReservesRoomWithCash`, `TestSettlement_DrawsReservedNeverFails`, `TestReserved_ZeroAtEveryBoundary` (FR-STO-009, 010). Production: `TestStart_KIsMinK0KRoom`, `_RefusedNoRoomWhenKZero`, `_NoRoomLastInOrder` (FR-STO-011), `TestStart_UpkeepRoomLeftFree` (FR-STO-046), `TestStart_PledgeIncludesExactOutputLoad`, `_DiscardOutputTakesNone`, `TestComplete_CreditsFromPledge`, `TestComplete_AfterCutDisposesRemainderInCommodityOrder` (FR-STO-013, example 18.4), `TestAbandon_DropsPledge`. Transport: `TestShip_ConsigneeFullAfterElsewhere`, `_PledgesOnDispatch`, `TestArrival_CreditsFromPledgeInLineOrder`, `_DivertsWhatDoesNotFit`, `_LowersPledgeByWholeLine` (FR-STO-014, 015), `TestShip_HasStoreAfterTravellerAway`, `TestTrip_OwnerOfStoreRefused` (FR-STO-031). Cut: `TestWeightReduction_LoadFitsPledgeMayExceed`, `_NeverRefused` (FR-STO-017), `TestRoleChange_ExcessToEstateUnderEstateGoods` (FR-STO-018; a cargo in flight is not moved), `TestPledge_DerivedNotSaved_RecomputedAtEveryChange` (FR-STO-020). Warm: `TestWarmStart_JobPledged`, `TestConstruction_RoomBelowWarmJob` (FR-STO-045, asserts no `ErrNoRoom` reaches construction). Invariant: `TestInvariant_RoomNeverExceeded` over `invariantSeeds` with `WithAudit` (FR-STO-005, 016, 040; the `invariantTable` row `one + storage.toml`, REF 7.5); `TestCredit_NoRoomWhereCommittedPanicsOrAuditFails` (FR-STO-016, with a deliberately broken caller in an internal test).
Exit check: S91 (below) runs; `golden/off` unchanged; `TestStep_AllocationsStayAtTheBaseline` at its baseline off, and 0 allocations per limited-account operation on.

### Phase 6: the default strategy, the stores flow (PR 4)

Goal: a limited agent behaves sensibly in its room, and the cost of a store is charged.
Files: `strategy.go`, `strategy_view.go` (the room group: room, load, pledge, free, stores, sizes, FR-STO-021), `strategy_default.go` (target scaling `f`, bids sized to the free room in the order of the money stack, a draw taken for every intent whether or not the bid is placed, the tier 1 discard as a `Discard` executed as a move to `disposal`), `internal/household` (only if the pure scaling is better there; otherwise `internal/room` only), `moneyflow.go` and `internal/moneyflow` (the `stores` target: an amount per store, fixed levy and rate shapes only, `Level`/`RateBP` are `ErrFlowShape`, the carry per account, `ErrUnknownStoreClass`, `ErrModuleDependency`), `checkpoint` tag 13 (the carry shape for a stores target, section 15.1).
Tests: `TestDefault_ScalesTargetsToRoom` (FR-STO-022, example 6.2.2), `TestDefault_BidsSizedToFreeRoom_NoRejectedBid`, `TestDefault_DrawsEveryIntent` (FR-STO-023: the stream position is equal with and without a binding room), `TestDefault_DiscardsOnlyForTierOne` (FR-STO-024), `TestDefault_TierOneAlwaysFits` (FR-STO-025 as a property over random rooms satisfying `room_below_upkeep`), `TestStoresFlow_AmountPerStore`, `_ShapeErrors`, `_UnknownClass`, `_ModuleOff`, `_NoNewReason` (FR-STO-026, 027), `TestStoresFlow_CarryRestored` (tag 13), `TestLeaky_NoChargeAtExempt`.
Exit check: the existing strategy tests unchanged; `TestAllocs_DefaultTrade` at its baseline with the module off.

### Phase 7: observability (PR 4, or PR 5)

Goal: the event, the `Debug` lines, the metric inputs.
Files: `storage_report.go` or `storage.go` (the counters of D5, `p36aRoomLimited` called at P3.6a after `WantShortfall`, the `Debug` lines behind `Enabled`, the construction-summary keys), `event.go` (`RoomLimited` declared; the reasons), `tick.go` (P3.6a `{"P3.6a","STO",line}`; `notBuiltPositions` loses "P3.6a"), `event_test.go` (`RoomLimited`, `StoreChanged` leave `notBuiltEvents`), `replayRows`, `observe_internal_test.go` (`stepSites` row), `cardinality_test.go` (`store_class`, `dimension`, `room_limit`), `catalogue_otel_copy_test.go`.
Tests: `TestRoomLimited_OnlyWhenNonZero_AscendingRegionThenRole_NoMap` (FR-STO-036), `_CountersClearedAtP86`, `TestStoreChanged_OrderAfterRoleChanged`, `TestAgentSpawned_StoresLockedZeroWithoutGate`, `TestWarn_NoneInsideTick` and `TestNoInfoInsideTick` (FR-STO-041), `TestNonInterference_ObserverAndLogger` (the same results with and without), `TestAllocs_EventNoop` (construction of both events allocates nothing with the no-op observer; FR-STO-041, NFR-STO-002), `TestLogKeys_StoreClass` (every constant is a row of rules 3.3.2), `TestDebug_Lines` (behind `Enabled`), `BenchmarkAdmitWithRoom`, `BenchmarkStartSizing`, `BenchmarkTargetScaling`, `BenchmarkDiscardOrder` with `b.ReportAllocs()` (NFR-STO-003).
Exit check: catalogue test with the two events and the position out of `notBuilt`; log-key test; `golden/off` event files unchanged.

### Phase 8: checkpoint (PR 5)

Goal: the stores in the agent record, the restore checks, the rebuilt pledge.
Files: `checkpoint_save.go`/`checkpoint_sections.go` (the stores written once in the tag 3 record when the module is on: a count and (class index, count) pairs ascending), `checkpoint_restore.go` (the checks of 15.3, `checkpointCross`: class index below the count, at most `MaxStoresPerAgent`, none on an unlimited role, load within the room; the pledge rebuilt from the jobs (tag 6) and shipments (tag 17) with no room test), `storage.go` (`rebuildPledge`).
Tests: `TestCheckpoint_StoresRoundTrip` (sparse, one spelling), `TestCheckpoint_RestoreRejects_ClassIndex`, `_TooManyStores`, `_StoreOnUnlimitedRole`, `_LoadOverRoom` (all `ErrCheckpointCorrupt`, FR-STO-039), `TestCheckpoint_PledgeAboveRoomAccepted` (the cut case of 18.4 restored without a refusal), `TestFork_Storage` (through `forkAtN`/`forkCase` with `storage.toml`, N in {0, inside a job, with a cargo in flight, after a store call, after a cut}; bit-identical, events included; FR-STO-043), `FuzzDecodeAgentRecordStores` (no panic, bounded allocation, NFR-CHK-003), `TestCheckpoint_AllFixture` (a restore of `format-1-all` with the new module off still passes; no existing fixture is regenerated; a new fixture is NOT added because the all-modules-on fixture is regenerated only once, by the PR that moves `golden/all/`, Q3).
Exit check: `testdata/checkpoints/` untouched; the checkpoint tests of the other modules pass.

### Phase 9: scenarios, goldens, the all layer, exit gate (PR 5)

Files: `scenario_s91_test.go` to `scenario_s96_test.go`, `testdata/chains/overlay/storage.toml`, `storage-two.toml`, `testdata/chains/scripts/s9x.toml`, `testdata/golden/storage/{one.txt,events-one.txt,one.positions.txt}`, `testdata/chains/expected/` entries, `TestDeterminism_Storage` over `one + storage.toml` taking cases from `detCases(true)`, `all_pack_test.go` (`allLayers` appended with `overlay/storage-five.toml`, the one retarget), `testdata/golden/all/` (moves once, if Q3 says now).
- **S91 (STO-S1).** A limited staple household and a producer with a base room just above one tick's upkeep plus one job's load, the module on, over seeds with `WithAudit`: the invariant of `spec-storage.md` 4.4 every tick; `RoomLimited` counts the cuts. Requirement: FR-STO-005, 011, 016, 022, 023, 040.
- **S92 (STO-S2).** A buffer store's owner (directed or a test strategy) fills in plenty and releases in want; the goods age on their own clock (FR-STO-028).
- **S93 (STO-S3).** A producer glut: a weight-1 producer with a full room and a price at the floor stops, recovers when it sells (reported, not asserted as a band).
- **S94 (STO-S4).** Two shippers, one consignee on `two`: the second is refused `consignee_full` and retries (FR-STO-014; first come, STO-A4).
- **S95 (STO-S5).** The levy against its pair and its exemption (FR-STO-026, `store_levy_below_upkeep`).
- **S96 (STO-S6).** A cut of the room under a pledge: a weight reduction and a role change, counting how often the cut acts (spec open questions 11 and 14).
All six report-only except the invariants, as `spec-stability.md` does for a module with no acceptance band (N-015, N-027).
Gate: `gofmt -l .` empty, `go vet ./...`, `staticcheck ./...`, `go mod tidy` no diff, `go test -short -race ./...`, then gate R (`go test -race -timeout 60m ./...`) and gate F (`-covermode=set`, `-timeout 120m`) on the dev host, `scripts/ci/check.sh` and `full.sh` with `coverage-check.sh`, `lizard -l go -C 10 -a 8 -L 100 -w` on `storage*.go`, `internal/room/*.go` and every changed non-test file, coverage at or above 90% for the root and for `internal/room`, `internal/ledger`, `internal/clearing`, `internal/moneyflow`, the mutation pass of Phase 2 repeated, the quick run (testing.md 9.1) before the PR, `scripts/ci/arm64.sh` if an arm64 machine is available (Q5), the registry and catalogue tests with the STO entries out of `notBuilt`, `testdata/golden/off/` and `TestHashConfig_ReferencePinned` untouched. The exit markers go in the PR description.

## PR split

The module touches the ledger, clearing, production, transport, the default strategy, the estates, the checkpoint and the integration, so one PR is too large to review. Five PRs on `phase-3`, in order, each green on its own gates and each leaving `golden/off/` and the pinned hash unchanged:

1. **PR 1, Phase 2:** `internal/room`, `internal/ledger` room. No root change.
2. **PR 2, Phase 3:** config, validate, hash, loader, bounds, sentinels, `StoreClassID`, registry rows. The module can be configured and validated; it does nothing.
3. **PR 3, Phases 4 and 5:** the account's room, stores, calls, region binding, and the commitments at bids, starts and dispatches. The module works end to end for the default `Strategy` sized only by the reject (a bid rejected for room), without the sizing of Phase 6. S91 and S94 run here (their invariants).
4. **PR 4, Phases 6 and 7:** the default strategy, the stores flow, the event, logs and cardinality.
5. **PR 5, Phases 8 and 9:** the checkpoint, the fork test, the scenarios, the module-alone golden, the all layer.

**Minimal slice the roadmap implies.** The roadmap gives STO one row ("`storage.go`, `internal/room`; Needs LED, GDS") and no smaller slice. The smallest slice that satisfies "a module ships in one PR with its golden and a fork test" and is useful to RSK and CAP is PRs 1 to 3 plus the checkpoint of Phase 8 (limited accounts, stores, commitments, the calls, the invariant, restore). The default strategy sizing, the levy and the observability (PR 4) can follow; without them a limited agent is rejected for room and starves rather than adapts, which is correct but poor. The plan does not recommend stopping there (Q1).

## Catalogue appends

Shared files that other Stage 3 modules also append to, so merge conflicts can be predicted. Each is append-only in module order (architecture.md 2.10.5); a conflict is textual (adjacent lines) and resolved by keeping both, in module order.

| File | What STO appends | Shared with | Conflict risk |
| --- | --- | --- | --- |
| `config.go` | `Modules.Storage *Storage` after `Employment`; `Storage`, `Capacity`, `RoleRoom`, `StoreClass`; `FlowTarget.Store` | every Stage 3 module (the `Modules` struct), RSK (`StoreRisk.Class`), CAP | High (the same lines) |
| `hash.go` | `pairsStorage` id 12 in `pairsLaterModules` or a new continuation | every module | High |
| `bounds.go`, `defaults.go`, `errors.go` | three bounds; `canonical` sorts; eight sentinels | every module; `ErrUnknownStoreClass` is also RSK's | Medium |
| `validate.go`, `config/mapping.go` | the call of `validateStorage`; the TOML tables | every module | High |
| `event.go` | `StoreChanged`, `RoomLimited`, `AgentSpawned.StoresLocked`, three reasons, cause `facility` constant | EMP, RSK, CAP, SET (`AgentSpawned` fields) | High |
| `snapshot.go` | `RoleRow` room fields, `StoreRow` | EMP, SET, RSK, CAP | Medium |
| `tick.go` | `{"P3.6a","STO",line}` and the call; the entry leaves `notBuiltPositions` | SAV (P3.6b), FIS (P3.6c) share the P3.6 tail | High, and the order of the tail matters |
| `fanin.go` | calls in `agentJoined`, `agentRoleChanged`, `removeAgent`, `fillAgentView`, `checkTradePlan`, `relocationRefusal` | every module with per-agent state | High |
| `world.go`, `construct.go`, `role.go` | `sto *storage`, `modulesOn` name, `roleRow.room` | every module | Medium |
| `checkpoint_save.go`, `checkpoint_restore.go`, `checkpoint_sections.go` | the stores in the agent record; checks | every module with a hosted field | High (the same record) |
| `moneyflow.go`, `internal/moneyflow` | the `stores` kind | FIS (`purchase`/`release` targets), LND | Medium |
| `integration.go`, `integration_slots.go` | `AddStore`, `RemoveStore`, `AgentState` group | EMP, LND, CRD, POL, TEC, PLC | High |
| `phase_production.go`, `production.go`, `strategy_*.go`, `transport_*.go`, `phase_exit.go`, `estates.go` | the guarded calls of Phase 5 | LND/`no_title`, CAP (`no_facility`), EMP (hires), INS, RSK (`transport_arrival.go`) | Medium; the order of the reasons in `production.go` is a contract (`no_room` last) |
| `doc.go`, `AGENTS.md`, `architecture.md` 2.6 | glossary; the continuation files | every module | Medium; a fresh RULE-TAG on each edit of the rule file |
| `internal/logkeys` | `StoreClass`, `StoreClasses` | every module | Low |
| `internal/clearing/admit.go` | the `no_room` row | SKL/ORG (`not_permitted`), ORG (legal clamp) | Medium (adjacent marked positions) |
| `testdata/chains/overlay/`, `testdata/golden/`, `all_pack_test.go` `allLayers`, `registry_test.go` `notBuilt`, `event_test.go` `notBuiltEvents` | files and list entries | every module | Low in files, **High** in the three lists and in `golden/all/` (one binary-text file every module moves; resolved by regeneration on the integration branch, never by merging the text; Q3) |

## Testing (testing.md 11)

**1. Requirements touched and their tests.** Every ID of spec-storage section 20.

| Requirement | Tests |
| --- | --- |
| FR-STO-001 | `TestStorage_OffLeavesNoTrace` |
| FR-STO-002, 003, 004 | `TestValidate_Storage_*`, `TestEffectiveRoom_*`, `TestRoom_Formula`, `TestLimited_IffBaseRoom_PseudoAgentNever` |
| FR-STO-005, 006, 007 | `TestRoom_*` (ledger), `TestInvariant_RoomNeverExceeded` |
| FR-STO-008, 009, 010 | `TestAdmit_*`, `TestBid_Reserves*`, `TestSettlement_*`, `TestReserved_ZeroAtEveryBoundary` |
| FR-STO-011, 012, 013, 046 | `TestStart_*`, `TestComplete_*`, `TestKRoom_*` |
| FR-STO-014, 015, 031 | `TestShip_*`, `TestArrival_*`, `TestTrip_*` |
| FR-STO-016, 040 | `TestCredit_NoRoomWhereCommitted*`, the audit tests |
| FR-STO-017, 018, 019, 020 | `TestWeightReduction_*`, `TestRoleChange_*`, `TestAddAgent_NoRoom`, `TestPledge_*` |
| FR-STO-021 to 025 | `TestView_*`, Phase 6 `TestDefault_*`, `TestScale_*`, `TestDiscardOrder_*` |
| FR-STO-026, 027 | `TestStoresFlow_*` |
| FR-STO-028, 029 | S92 (ageing); `TestNoChargeableSize_Guard` (a text search that `BulkFactor` and the chargeable size are not read by storage files) |
| FR-STO-030, 032 to 035 | `TestAddStore_*`, `TestRemoveStore_*`, `TestRelocation_*`, `TestStoreChanged_*`, `TestCreation_*`, `TestRoleChange_*` |
| FR-STO-036, 037, 041 | `TestRoomLimited_*`, `TestNonInterference_*`, `TestAllocs_EventNoop`, `TestDraws_CountUnchangedWhileNothingActs` |
| FR-STO-038 | one failing case per row of 13.1/13.2 |
| FR-STO-039, 043 | `TestCheckpoint_Stores*`, `TestFork_Storage` |
| FR-STO-042 | `TestStoredLoad_Formula` |
| FR-STO-044 | field and error tests here; the gate itself is SET's (D8, Risk 3) |
| FR-STO-045 | `TestConstruction_RoomBelowWarmJob`, `TestWarmStart_JobPledged` |
| NFR-STO-001 | `TestRoom_UnlimitedCostsOneBranch`, ledger benchmarks |
| NFR-STO-002 | `TestAllocs_*` at the baseline |
| NFR-STO-003 | the four benchmarks of Phase 7 |
| NFR-STO-004 | `TestDeterminism_Storage` (two seeds, with and without observer and logger); the arm64 run is by hand (Q5) |

**2. Lowest level.** The rules of 5.3.2, 6.2.2, 6.3.2, 5.5.2.1 and 8.3.2 are unit tests of `internal/room`; the invariant is a unit and fuzz test of `internal/ledger`; the hooks are internal tests of `storage.go` and of each host file; the pack tests are S91 to S96; the end-to-end level is the module-alone golden and the fork test (a handful).
**3. New boundaries and error paths.** The eight sentinels, the bounds (`MaxRoom`, 8 stores, 64 classes), the overflow of a load at `MaxQuantity` times 10^6, the decoder of the stores record, the cut of the room under a pledge, `ErrNoRoom` at `AddAgent`, `RemoveStore` and the credit.
**4. Golden files.** `golden/off/` and the pinned hash do not move. `golden/storage/` is new. `golden/all/` moves once (Q3), with the explanation. Nothing existing is regenerated.
**5. Fork-at-N.** `TestFork_Storage` as in Phase 8, plus S96.
**6. Registry and catalogue.** Rows of 12.4 leave `notBuilt`; the two events and P3.6a leave their lists.
**7. Fuzz.** `FuzzValidateStorage`, `FuzzLoad` corpus, `FuzzRoomOps`, `FuzzDecodeAgentRecordStores`.
**8. Mutation.** `gomu` only inside Docker, `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`, on `internal/room` and on the new ledger room file; never on the host.
**9. Tests run against a broken version** (testing.md 9.3): the phase exit for each of `k_room`, the admission row and the credit-from-pledge flips a comparison and checks a test fails.

## Observability (observability-and-logging.md 9)

1. New failure path: ten validation rules returned as `ValidationError`; `ErrNoRoom` returned by `AddAgent` and `RemoveStore`; a refused bid, start and shipment survived and counted in `RoomLimited` and as `OrderRejected`/`RecipeRefused`/`ShipmentRejected` with the reasons `no_room`, `consignee_full`, `has_store`; no `Warn` inside the tick.
2. New state transition: a store added or removed (`StoreChanged`, per agent, no map), a per-region-role aggregate (`RoomLimited`, bounded by regions times roles, fixed-point, no map), `AgentSpawned.StoresLocked` (a count).
3. New decision point in a strategy: the scaling, the bid sizing and the discard of the default strategy: `Debug` lines "bid cut for room", "job refused for room" with `reason`, behind `Enabled`.
4. New loop: the room check is per order/start/dispatch; nothing logged above `Debug`; the only per-tick event is the aggregate; event construction and the counters that exist only for it are built under `w.obs.supplied` (FR-STO-041), proven by `TestAllocs_EventNoop`.
5. Changed event, metric or attribute: none changed; additions only (`RoomLimited`, `StoreChanged`, the field `StoresLocked`, three reasons, three metrics already catalogued).
6. New metric attribute: `economysim.store_class`, `dimension`, `room_limit`, all bounded by the configuration or a closed set (5.4.1); series added for the reference chain with the overlay: one class gives `store.count` 1 per region, `room.fill` 2 per (region, role), `storage.limited` 5 per (region, role).
7. `Error` or `Info`: `Info` only the existing construction summary with `store_classes` and `storage` in `modules`; no `Error`.
8. Error logged and returned: none.
9. Tests: a recording observer for both events and their order, a capturing handler for the construction `Warn`s and for the absence of any `Warn`/`Info` in a tick, non-interference, the no-op allocation test, the cardinality test.

## Risks and open items

1. **The reference chain collapses (N-352).** Every reference pack but `payroll` goes extinct and the owner closed Stage 2 on that. The module-alone golden of STO is recorded on a collapsing run (pack `one` + overlay, 400 ticks): the golden shows a short run, and a limited household that starves earlier is not a finding about STO. The scenarios S91 to S96 assert invariants and report the rest, so they do not depend on a healthy economy; but a result such as "a binding room worsens the collapse" cannot be told from the baseline. Mitigation: record the off run beside the on run in S91 and report both; use the pack `payroll` for one scenario if the supervisor allows it.
2. **Size and the Lizard limits.** The hooks in `phase_production.go`, `transport_*.go`, `fanin.go` and the default strategy add branches to files already near the limits (N-243). A continuation file may be forced (D3) and the layout rule needs a fresh RULE-TAG; budget a refactor-first commit in PR 3.
3. **Settlements and the gate.** FR-STO-044 cannot be fully tested until SET exists (D8). If SET lands first, STO must read its flag; if STO lands first, SET's plan owns `regionHas`, `ErrUnknownSettlementFlag`, `store_locked_at_start` and the non-zero `StoresLocked`. The first to land writes the nil-gate shape the other reads.
4. **Many Stage 3 plans edit the same eight files** (the Catalogue appends table). The order of merge matters for `tick.go` (the P3.6 tail: STO at 3.6a, SAV at 3.6b, FIS at 3.6c) and for the `AgentSpawned` fields. The supervisor orders the merges; a conflict resolved by keeping both lines in module order is safe.
5. **Pledge correctness under a cut.** The pledge is derived and may exceed the room after a cut (FR-STO-017, 18.4). The audit asks for less than a reader might expect (4.4 item 2); a defect in a commitment is found at the commit, not later. The fork test must cover a restore with a pledge above the room.
6. **The upkeep room and `room_below_recipe`.** A recipe may never start (the upkeep room is left free). The warning exists; the rule is placeholder-numbered (N-015, N-027); no default is called stable.
7. **Presets.** N-327 and the roadmap row put the `Losses` preset (FR-IND-059) with RSK; the spec asks IND for a storage-use measure (N-130). Neither is built here. If the owner wants the measure now, it is one preset on `RoomLimited` and the snapshot rows, in `indicator/`, with its tally row.
8. **The otelobserver module** adds three instruments in its own PR (D16); the root PR's `cardinality_test.go` and the catalogue copy test pass without it.
9. **Per-tick cost.** The `no_room` row is one comparison per bid on a limited account and one branch otherwise; the cost budget of SCL (53 ms a tick for `five`, off) is not re-measured by this plan. `cost_five_test.go` records, and `BenchmarkAdmitWithRoom` and the ledger benchmarks report the trend.

## Questions only the owner can answer

- **Q1. Scope of PR 4.** Build the default strategy's scaling, bid sizing and discard (Phase 6) in this module, or ship the invariant first (PRs 1 to 3 and the checkpoint) and the sizing after? The plan assumes all of it, five PRs.
- **Q2. `Needs` while SET is unbuilt.** Keep the field and reject any non-empty value with `ErrModuleDependency` (D8, the plan's default), or leave `Needs` out of `StoreClass` until SET adds it (a later append to the struct and the hash, no behaviour lost)?
- **Q3. The all layer and `golden/all/`.** Move `golden/all/` and the all-modules fixture now, with STO, or once at the end of Stage 3 when more modules are in? Every Stage 3 module that moves it conflicts with every other.
- **Q4. Order of the Stage 3 modules.** SET before STO (so the gate is live), or STO first as the roadmap lists? Does CAP or RSK wait for STO?
- **Q5. arm64.** No hosted job runs it. Is a manual arm64 run of the quick pattern required for this module's PRs, or only before the Stage 3 close?
- **Q6. Reference-chain health.** Is a green but short module-alone golden on the collapsing pack `one` acceptable (Risk 1), or should STO's golden and S91 use the pack `payroll`?
- **Q7. The storage-use indicator** (N-130): build one preset now, leave it to RSK's `Losses` PR, or reject it?
- **Q8. The `facility` cause and `ErrStoreInUse`.** Declare them now with the zero-facility seam (D9), or leave both to CAP's plan?
