# Plan: Stage 3, row SET: `settlements.go` and `internal/settlement`

**Status: plan written (2026-11-14). Stage 1 of the work (requirements and plan, no code). Written on branch `s3-set`, cut from the integration branch `phase-3` (= `d996592`, which holds the merged plans and first pull requests of SKL, POL and STO). Format model: `plan-3-sto.md` (a module with a `Modules` pointer, a hash id and a checkpoint tag), `plan-3-skl.md` (the seams and hook discipline) and `plan-3-pol.md` (the PR split). The owner questions stay open.**

Sequences the roadmap-v1.md Stage 3 row "`settlements.go`, `internal/settlement` | SET | REG, SCL" into eight phases and three pull requests. It implements [spec-settlements.md](../specs/spec-settlements.md) (SET): FR-SET-001 to 033 and NFR-SET-001 to 004, with the SET parts of [spec-config.md](../specs/spec-config.md) 3.1, 3.7.1, 8.2.2 and FR-CFG-084, [spec-checkpoint.md](../specs/spec-checkpoint.md) 4.2 tag 23, [spec-world.md](../specs/spec-world.md) 6 (P7.2a) and [ADR 0006](../adr/0006-structural-rank-may-gate-entry-rights-and-places.md).

Rule tags read (the three files, in full): architecture=354c9304a17403fb4bd93ae8, observability=f215753306fac0bc7607c4c4, testing=66af7645b01fef4e2796c367.

Notes tag (`doc/notes.md`): c7b9e596d70af02e60d7d893. This plan writes no spec, so the notes rule of AGENTS.md does not bind it. It read the index and the SET-tagged notes (N-140, N-192, N-268, N-348) and the notes that name this spec's seams (N-153, N-193, N-244, N-264). It did **not** read the ALL notes (N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217, N-315, N-335, N-340, N-345, N-351, N-353) in full. **Phase 1 edits specs and must read the whole of `doc/notes.md`, quote its tag, and give each SET and ALL note a verdict.**

Specs read: spec-settlements.md in full; spec-config.md (3.1, 8.2.2, FR-CFG-084, the SET rows of 4.1 and 7); spec-checkpoint.md (tag 23); spec-world.md (P7.1, P7.2, P7.2a); spec-storage.md (3.2.3, 10.3 through plan 3-sto D8); spec-skill-barriers.md (4.5.2, through plan 3-skl SK9). Not re-read for this plan: spec-spawner.md 6.2 to 6.3 and spec-population.md 3.7.3, 4.4 (Phase 1 reads them before the readers are built; the code seams were read in the tree instead).

## What is already in the tree (from `d996592`)

- **Config surface, none of it built.** `registry_test.go` lists `Modules.Settlements`, `Settlements`, `SettlementClass`, `ClassLimit`, `RegionSettlement`, `Role.Needs` and `Role.Ceiling` in `notBuilt`. There is no `ErrUnknownSettlementClass`, `ErrUnknownSettlementFlag` or `ErrUnknownLimit` in `errors.go`, and none of `MaxSettlementClasses`, `MaxClassFlags`, `MaxClassLimits`, `MaxBandBP`, `MaxThresholdBP` in `bounds.go`.
- **The hash has gaps SET fills.** `hash.go` writes `Modules` ids 9, 10, 12, 13, 19 and 25 in `pairsLaterModules`; id 11 is SET's. `pairsRole` stops at `Employer` 24 (EMP took 23 and 24 ahead of SET), so `Role.Needs` and `Role.Ceiling` take the next free ids, **25 and 26**, as spec-config.md 8.2.2 says ("the next free ids of `Role` when that module is built"). `StoreClass.Needs` (id 4) is already hashed.
- **STO left the seam as a validation rule.** `storage_validate.go` returns `ErrModuleDependency` for any non-empty `StoreClass.Needs` (plan 3-sto D8). STO PR 3 (runtime, the gate and `StoresLocked`) has **not** merged: the tree has no `StoresLocked` and no `store_locked_at_start`. `ErrStoreLocked` exists in `errors.go`.
- **SKL left the headroom slot.** `skills.headroom func(RegionID, RoleID) (headroom int64, noLimit bool)` (`skills.go:79`) is nil; `completionHeadroom` (`skills_queue.go:26`) returns "no limit" until a module sets it (plan 3-skl SK9, N-193).
- **POP and SPN left appended-field placeholders.** `event.go:993` says the `RoleCeiling` refusal of `EntryLimited` is "appended with that module"; `event.go:1282` and `population.go:26` say the same for `PopulationChanged.BirthsCeiling`. `RoleView` has `Closed` and no `Headroom` or `NoLimit` (`spawner.go:89`; N-268 lists the order).
- **The position and the section are listed as not built.** `tick.go:88` has `"P7.2a"` in `notBuiltPositions`; `checkpoint_sections.go:74` has `{23, "Settlements", "Settlements"}` in `notBuiltSections`. `P7.2a` is not a sub-step: it is a position of its own after `P7.2` (spec 5.1), and `positionsTail` already calls `p72Migration` guarded.
- **Pieces SET reuses.** `internal/checked` (128-bit helper), `internal/ordering`, the `regionBounds` registry (SET registers none: N-090), `internal/logkeys`, the loader of `config/mapping.go`, `forkAtN` and `forkCase`, the `Warn` collection of `validate.go` (`rule` and `path`), and the `AgentSpawned` and `WorldCreated` construction events. `internal/occupation.Admit` already takes per-role caps (`Limits.RoleCap`) and counts the cut by cap.
- **Layout.** `settlements.go` and `internal/settlement/` are already in the layout of architecture.md 2.6 and AGENTS.md. Continuation files are not listed.

## Scope

**Builds.**

- `internal/settlement` (new package; stdlib and `internal/checked` only; primitives only, NFR-SET-003): the effective thresholds (`Thresholds`), the review (`Review`, with and without a floor), the dwell test, the held-change result, the target of a snapshot, the headroom of a role (`Headroom`), the dense flag and limit tables (`Table`: one `uint64` per class for flags, one for which limits exist, the values), and the construction class (`Initial`). No region, role, flag or class name.
- Root: `settlements.go` (the module-file order of architecture.md 2.10.1: state `settlements`, `newSettlements`, `validateSettlements`, `p72aSettlement`, `regionHas`, `regionLimit`, the headroom reader, the snapshot filler), and the continuation files `settlements_validate.go` (VAL: validation and warnings) and `settlements_report.go` (OBS and CHK: events, logs, snapshot rows, the codec of tag 23). The split is for the Lizard limits of N-243; Phase 1 confirms it (E1).
- Catalogue appends: the table "Catalogue and shared-file appends" below.
- The readers of the table of spec 6.3 that exist in the tree today: **H1** and **H2** (the default spawner and POP's births, through the view), **H3** (no code: a role with a `Ceiling`), **H5** (STO's `StoreClass.Needs`, the seam of plan 3-sto D8) and the **SKL headroom hook** (SK9). H4 and H6 (ORG's `Body.Needs`), H8 (CAP's `FacilityClass.Needs`) are those modules' to add; each adds a row to spec 6.3 and a reader of the class (R7), so SET leaves the two reads of 6.1 and the table.
- `testdata/chains/overlay/settlements.toml` and a probe-chosen pack, `testdata/golden/settlements/` (module alone), scenarios SET-S1 to SET-S5, band keys none.

**Public identifiers.** `SettlementClassID`, `NoSettlementClass`, `Settlements`, `SettlementClass`, `ClassLimit`, `RegionSettlement`, `Modules.Settlements`, `Role.Needs`, `Role.Ceiling`, `SettlementClassChanged`, `SettlementHeld`, the causes `CauseFounded`, `CauseGrew`, `CauseShrank` (the spelling is fixed in Phase 1 against the constants already in `event.go`), `EntryLimited.RoleCeiling`, `PopulationChanged.BirthsCeiling`, `RoleView.Headroom` and `NoLimit`, `RegionRow.Class`, `ClassSince`, `ClassTarget`, `ClassAtFloor`, the sentinels `ErrUnknownSettlementClass`, `ErrUnknownSettlementFlag`, `ErrUnknownLimit`, the bounds of spec 8.1, the log keys `settlement_class`, `from_settlement_class`, `to_settlement_class`, `target_settlement_class`, `settlement_classes`. **No** option, slot, call or ledger reason (positions 12 and 13 of the spec), and no `regionBounds` obligation (N-090).

**Leaves out, with the owner.**

| Item | Owner |
| --- | --- |
| `Body.Needs`, a body's places as a second limit into the headroom, the min of the class limit and the body's | ORG (N-192); SET ships the class-limit half and a slot the body's half joins (SET-3) |
| `FacilityClass.Needs`, `Mode.Needs` | CAP, TRN (reserved) |
| A defence limit (H7), a recipe condition (H9), a market per commodity (H10), a class-keyed region field | declined by the spec (6.6) |
| A world-enforced gate (`Needs` and `Ceiling` for an external agent or a consumer's spawner) | spec open question 2, later |
| A class in any strategy view (position 8) | never in the core (FR-SET-018) |
| The two metrics of spec 10.4 and their `otelobserver` instruments; the catalogue rows are a doc change | a later change to `otelobserver/` (Question 6) |
| Band values for SET scenarios; stability of any default | STY and CLB; every default stays a placeholder (N-015, N-027) |
| The viability check reading the class (open question 5, SET-A3) | VIA, if a pack hits it |
| A `golden/all/` and all-modules checkpoint-fixture move | the supervisor, once at Stage 3 close (owner decision, `phase-3-owner-questions.md`) |

## Decisions

Each is the spec's, a rule's or a note's. The plan decides nothing new except where a Question says so.

- **SET1. Hash ids and the Go order.** `Modules.Settlements` has hash id **11** (position in the row of spec-config.md 3.1). Its Go line is appended after the pointers that exist (after `Polity`, after `Storage` and `Employment` if they stand there; architecture.md 2.10.5; `pairsLaterModules` is the model). Its pair goes between ids 10 and 12 in the writer. Inner ids, in declaration order: `Settlements` `Classes` 1, `BandBP` 2, `Dwell` 3, `Regions` 4; `SettlementClass` `ID` 1, `Enter` 2, `Flags` 3, `Limits` 4 (`Label` has none); `ClassLimit` `Name` 1, `Value` 2; `RegionSettlement` `Region` 1, `ThresholdBP` 2, `Floor` 3. `Role.Needs` **25** and `Role.Ceiling` **26**, each written only when non-empty (the off value writes nothing, so `golden/off/` and the pinned hash do not move). E2 records these in spec-config.md 8.2.2, which today lists no inner ids for SET.
- **SET2. The canonical order is the second exception to "by ID".** Classes are sorted by `Enter` before a world is built or a hash is taken; flags by name, limits by `Name`, region rows by `RegionID` (spec 3.2). The hash writes the **sorted** form, so two configurations of equal content hash alike (FR-SET-030). `SettlementClassID` is the index in that order. The `canonical` function in `defaults.go` carries the sort; E3 adds the sentence to spec-config.md 3.4 and 3.3 (handoff row 14 of the spec).
- **SET3. State.** `type settlements struct` held as `w.set *settlements`, nil unless `Modules.Settlements` is present (a module pointer, unlike SKL). Per region that is not an outside node: `class []int32` and `since []int64`. The effective thresholds (`E`, `L` per region and class), the flag and limit tables and the household count are **derived**, built at construction and rebuilt by a restore, never saved (spec 11). The agent type gets no field. Scratch for the review is allocated once.
- **SET4. The household count is a scan at P7.2a, not a maintained counter.** Spec 4.1.3 allows both. A counter needs a hook in `agentJoined`, `dropAgent`, `relocate` and every weight change, four more calls in the busiest shared files for a per-tick cost the scan already meets (NFR-SET-001: O(A) once per tick). The scan uses the function `RegionRow.Households` uses (Phase 4 names it; if the snapshot computes it inline, a refactor-first commit extracts one function both call, so FR-SET-009 is one definition). Recommended; Question 3.
- **SET5. One position, one value.** `P7.2a` runs after `P7.2`, in `positionsTail`: `if w.set != nil { w.p72aSettlement() }`, one line beside `p72Migration`; `{"P7.2a", "SET", line}` is appended in the table order between `P7.2` and `P8.1`, and `"P7.2a"` leaves `notBuiltPositions`. The class is written only there (FR-SET-014). No other write exists, so a reader sees one value from P0.1 to P7.2 (5.2.1). The call returns no error: the review draws nothing and moves nothing (4.7).
- **SET6. Hooks are the named ones of ADR 0010 and architecture.md 2.10.1** and each is one guarded call or one appended field, with no branch on module data and no arithmetic in a core file. The list, with the file, function and call, is the section "Hook discipline" below. The reader's side (what the spawner and POP do with a headroom) is the reader's own rule over a view field, as N-268 and spec 6.4 say; it is not a module hook and it is listed separately with an owner approval flag.
- **SET7. Randomness and the ledger.** None. No site constant is added to `random.go`; no ledger reason, no boundary account, no money or goods move (FR-SET-016). A `Guard:` text search plus the determinism test show it.
- **SET8. Events.** Two, per spec 10.1: `SettlementClassChanged` (per region, per change) and `SettlementHeld` (per region per tick of a hold). Plain data, no map, `Households int64`, closed-set cause, constructed under `w.obs.supplied` only. Order at construction: after the `AgentSpawned` events, one `founded` per region in ascending `RegionID` (5.4.1). Order in a step: at P7.2a after the last `MigrationReviewed`, region ascending, never both for one region in one tick (5.4.2). Neither exists with the module off.
- **SET9. Reports.** `RegionRow` appends `Class`, `ClassSince`, `ClassTarget` and `ClassAtFloor` at the end (spec 10.5). Their off value is 0 or false, but `Class` and `ClassTarget` are `SettlementClassID` values and rule 2.10.5 forbids an appended typed ID because zero is a valid ID. Proposed reading: keep the spec's types and have the golden writer omit the four fields while the module is off, so `golden/off/` does not move (E5, R8, Question 4). The two metrics are catalogue rows only (Question 6).
- **SET10. Checkpoint tag 23.** A dense section written when `Modules.Settlements` is set and absent otherwise: per non-outside region in ascending `RegionID`, `Class` then `Since` as unsigned varints (spec 11). The decoder rejects a class not below the class count, below the region's `Floor`, or a `Since` above the checkpoint tick less 1 (0 at tick 0), and a section present with the module off or absent with it on. The format version does not change: the tag is already in the table (`spec-checkpoint.md` 9.3), `notBuiltSections` loses its entry. The one all-modules-on fixture is **not** regenerated by this change (owner decision, Q3 of plan 3-sto).
- **SET11. Seams with STO and SKL are filled, not replaced.** (a) `StoreClass.Needs`: the rule `needs_module_off` in `storage_validate.go` becomes "a flag some class carries, the module on" (`ErrModuleDependency` while off, `ErrUnknownSettlementFlag` when no class carries it); the runtime gate of STO PR 3 is filled by `regionHas`; `store_locked_at_start` and a non-zero `AgentSpawned.StoresLocked` are completed here **if STO PR 3 has merged**, otherwise SET leaves the one-line seam and says so (Risk R1). (b) `skills.headroom` is set at construction to `w.roleHeadroom`, the class-limit half of spec-institutions.md 4.5.2; ORG later wraps it with the body's half.
- **SET12. Documentation.** `doc.go` glossary gets the SET terms before any code uses them (architecture.md 3.2): settlement class, enter threshold, leave threshold, band, dwell, settlement review, household count (of a region), flag, limit, gate, class floor, held change, headroom (of a role's ceiling), ceiling (of a role), `SettlementClassID`, `NoSettlementClass`. `TestGlossary_DeclaredTypesNamed` reads `doc.go`, so the new type names (`Settlements`, `SettlementClass`, `ClassLimit`, `RegionSettlement`, `SettlementClassChanged`, `SettlementHeld`) are added in the same phase. "Floor", "class" and "limit" stay qualified (spec 10.3 and the README list).

## Requirement list and where each is verified

"New" means built by this plan. Nothing in the tree cites a FR-SET ID yet, so every row is new; the fields `StoreClass.Needs` and the SKL slot are re-checked, not owned.

| ID | Phase | Lowest-level test |
| --- | --- | --- |
| FR-SET-001 (off leaves no trace; `Needs` or `Ceiling` with the module off is `ErrModuleDependency`) | 3, 4 | `TestSettlements_OffLeavesNoTrace` (off golden, hash, no event, no section), `TestValidate_Settlements_NeedsModuleOff` |
| FR-SET-002 (class count, sort by `Enter`, index is the ID, `first_class_open`, `enter_order`, `no_class`) | 2, 3 | `TestCanonical_ClassesSortedByEnterNotID` (the `city` before `hamlet` case of spec 3.2.1), `TestValidate_Settlements_FirstClassOpen`, `_EnterOrder`, `_NoClass` |
| FR-SET-003 (bounds on `Enter`, `BandBP`, `Dwell`, `ThresholdBP`, a limit value) | 3 | `TestValidate_Settlements_Bounds` (one failing case per bound, path in the message) |
| FR-SET-004 (IDs, duplicates, `MaxClassFlags`, `MaxClassLimits`) | 3 | `TestValidate_Settlements_InvalidID`, `_Duplicate`, `_TooMany` |
| FR-SET-005 (flags and limits as unions; absent limit is no limit, a 0 is a closed gate) | 2, 4 | `TestTable_NoLimitVersusZero`, `TestRegionLimit_ReadsClassAsItStands` |
| FR-SET-006 (region rows: `ErrUnknownRegion`, `ErrOutsideNode`, `ErrDuplicateID`, sorted, absent row silent) | 3 | `TestValidate_Settlements_RegionRows`, `TestWarnings_AbsentRowSilent` |
| FR-SET-007 (`Floor` unknown is `ErrUnknownSettlementClass`; class never below floor) | 2, 3, 4 | `TestValidate_Settlements_FloorUnknown`, `TestReview_NeverBelowFloor` (unit and the model's count range 0 to 3 000 of spec 14.1) |
| FR-SET-008 (`E_k = ceil`, `L_k = floor`, `L_k <= E_k - 1` with a band) | 2 | `TestThresholds_CeilAndFloor` (the worked tables of 12.1 and 12.2), `TestThresholds_BandNeverEmpty` (4.2.2 cases), `FuzzThresholds` |
| FR-SET-009 (household count is the sum of weights, pseudo-agents out, away and on-journey in origin; equals `RegionRow.Households`) | 4 | `TestHouseholdCount_EqualsSnapshotRow`, `TestHouseholdCount_JourneyAndAwayInOrigin`, `TestHouseholdCount_PseudoAgentsExcluded` |
| FR-SET-010 (the review's two directions, exclusive; inside its band after a review) | 2 | `TestReview_UpThenDownExclusive`, `TestReview_MovesMoreThanOneClass`, `TestReview_InsideBandUnlessFloor` (an exhaustive table, not a sample), `FuzzReview` (NFR-SET-004) |
| FR-SET-011 (a held change leaves state, emits `SettlementHeld`, is not remembered) | 2, 4 | `TestDwell_HoldsAndForgets` (plague that recovers, example A of 12.1), `TestP72a_HeldEmitsHeldNotChanged` |
| FR-SET-012 (`Since` from the last change, 0 at construction) | 2, 4 | `TestDwell_RunsFromLastChange`, `TestConstruction_SinceZero` |
| FR-SET-013 (construction class, no band, `founded` events after `AgentSpawned`) | 4 | `TestConstruction_ClassIsHighestReached`, `TestConstruction_FoundedEventsOrder` (the five regions of 12.1) |
| FR-SET-014 (one review per tick at P7.2a, one value from P0.1 to P7.2) | 4 | `TestPositions_P72aListed`, `TestClass_ConstantWithinTick` (a recording reader at P1, P3, P7.1) |
| FR-SET-015 (one class, no gate: every result equal to the world without the module) | 4, 7 | `TestInert_OneClassEqualsOff` (SET-S4 over seeds: snapshot fields other than the module's, events other than the module's, final PCG state) |
| FR-SET-016 (no draw, no money or goods, no reason) | 4 | `TestSettlements_NoDrawNoLedger` (stream position and ledger identical on and off for the one-class pack), `Guard:` text search |
| FR-SET-017 (no call or slot sets a class; a call between ticks changes the count only) | 4 | `TestBetweenTicks_ClassUnchanged` (a thousand external agents, same class until a step), `Guard:` over `option.go` and `integration*.go` |
| FR-SET-018 (no `Work`, `Trade`, `Revise` view or outcome carries a class, flag or limit; the spawner's view carries openness and headroom only) | 6 | `TestViews_CarryNoClass` (reflection over the view types), `TestSpawnerView_HeadroomNotClass` |
| FR-SET-019 (readers are exactly the adopted rows of 6.3) | 6 | `TestReaders_StaticCheck` (a text search over the packages that call `regionHas` and `regionLimit`) |
| FR-SET-020 (dense tables resolved at construction; a read allocates nothing) | 2, 4 | `TestRegionHas_ZeroAllocs`, `TestRegionLimit_ZeroAllocs` (`testing.AllocsPerRun`), `BenchmarkRegionHas` |
| FR-SET-021 (`ErrUnknownSettlementFlag`, `ErrUnknownLimit`, `ErrModuleDependency`) | 3 | `TestValidate_Settlements_UnknownFlag`, `_UnknownLimit`, `_NeedsModuleOff`, `_StoreClassNeeds` (STO's field) |
| FR-SET-022 (a gated role is closed as `Role.Closed` is; households stay) | 6 | `TestGate_NeedsClosesRoleInRegion` (no entrant, no switcher, no like-for-like, no birth, no carry advance), `TestGate_HouseholdsStay` |
| FR-SET-023 (headroom `max(0, limit - households)`; admission and births capped and counted; none removed) | 6 | `TestCeiling_HeadroomAndRefusalCounted` (the 9 + 4 entrants of 12.1: three admitted, one counted in `RoleCeiling`), `TestCeiling_BirthsCappedAndCounted`, `TestCeiling_NobodyRemoved` |
| FR-SET-024 (`SettlementClassChanged` once per change, cause grew, shrank, founded, no map) | 4 | `TestEvent_ClassChangedFields_NoMap`, `TestEvent_CauseByDirection` |
| FR-SET-025 (`SettlementHeld` once per region per tick of a hold; never in a tick of a change, never when nothing is due) | 4 | `TestEvent_HeldTicksLeftCountsDown` (25 down to 1 as in 12.1), `TestEvent_NeverBothForOneRegion` |
| FR-SET-026 (`RegionRow` fields: saved and derived; restore equals original) | 5 | `TestSnapshot_ClassFields`, `TestSnapshot_ClassAtFloor` (the `north` row of 12.1), `TestSnapshot_ClassTargetIgnoresDwell` |
| FR-SET-027 (construction line, `Warn` once each at construction, `Debug` only in the tick, guarded) | 5 | `TestLogs_SettlementsConstructionLine`, `TestLogs_SettlementsDebugOnly`, the existing "no Info or Warn in the tick" test extended with the overlay |
| FR-SET-028 (the warnings of 9.2: raised as described, never fail validation, never in a tick) | 3 | `TestWarnings_Settlements` (one case each of the seven, plus the silent case on the overlay) |
| FR-SET-029 (tag 23: class and `Since` only; decoder rejects; next review as the uninterrupted world) | 5 | `TestCheckpoint_SettlementsRoundTrip`, `TestCheckpoint_SettlementsRefusesCorrupt`, `TestFork_Settlements` (SET-S5: a fork taken inside a hold) |
| FR-SET-030 (hash covers the sorted content, no `Label`; file equals Go) | 3 | `TestHash_SettlementsOnOff`, `_FieldIDs`, `_ClassOrderIrrelevant`, `_LabelNotHashed`, `TestLoad_SettlementsToml`, `TestLoad_SettlementsUnknownKey` (`ErrUnknownKey`), `FuzzLoad` corpus extended |
| FR-SET-031 (the two metrics derivable from the events; construction events give the gauge its first value) | 5 | `TestEvents_MetricsDerivable` (a recording consumer folds the events to the gauge and counter of 10.4); the instruments are Question 6 |
| FR-SET-032 (checked arithmetic, the largest case does not overflow) | 2 | `TestThresholds_LargestCaseNoOverflow` (`MaxHouseholds * MaxThresholdBP`) |
| FR-SET-033 (deterministic classes, events and order across runs and platforms; no map iteration) | 4, 7 | `TestDeterminism_Settlements` over `detCases(true)`, split `Run` calls, with and without observer; the arm64 quick run once at Stage 3 close |
| NFR-SET-001 (O(R) with a counter, O(A) with a scan, nothing that grows with households) | 4 | `TestSettlements_WorkIsCounted` (agents visited per tick equals live agents, enforced), `BenchmarkP72a` (reports) |
| NFR-SET-002 (no allocation beyond the module's tables; no event without an observer) | 5 | `TestAlloc_SettlementsNoOp` (`AllocsPerRun`, default logger and observer), `TestStep_AllocationsStayAtTheBaseline` rows unchanged with the module off |
| NFR-SET-003 (rules in `internal/settlement`, stdlib only, no name) | 2 | `TestPackage_NoRootNoName` (imports and identifiers), `TestPackage_NoFloat` |
| NFR-SET-004 (validator and review fuzzed: never panics; result in `[floor, K)`, inside its band) | 2, 3 | `FuzzReview`, `FuzzThresholds`, `FuzzValidateSettlements` |

Also touched and re-checked, not owned: FR-STO-044 (the gate itself is SET's), FR-SKL held completion (N-193), FR-SPN-041 and FR-POP-007 (the `Needs` half; N-348), FR-REG-060 (SET registers no obligation), FR-OBS-018 to 021 and 076, FR-CHK-023, FR-CFG-054 and 084, FR-WLD-040.

## Phases

Each phase ends green on `go test -short -race ./...` and on the tests it adds. `testdata/golden/off/`, `TestHashConfig_ReferencePinned`, `testdata/checkpoints/` digests and every existing golden stay **unchanged at every phase exit** (a test run at each exit; a failure is a defect, never a golden to regenerate). The PR split groups the phases.

### Phase 1: the owed edits, the audit and the glossary (no code) (PR A)

Goal: the specs, rules and notes are true before the code starts. Rule files that change get a fresh tag. This phase edits specs, so the worker reads **all** of `doc/notes.md` and quotes `Notes tag:` first, and gives each open note tagged SET or ALL a verdict. Edits owed:

| # | File and section | Edit |
| --- | --- | --- |
| E1 | architecture.md 2.6, AGENTS.md Layout | add the continuation files `settlements_validate.go` and `settlements_report.go` (SET) to both lists with the same files; add the row `internal/settlement 90` to `scripts/ci/coverage-thresholds.txt` (the checker fails a package that is not listed); fresh RULE-TAG. If Lizard does not force the split at Phase 3 and 4 the files are not created and the edit is not made (decide by `lizard` on the draft, as plan 3-sto D3) |
| E2 | spec-config.md 8.2.2 | the inner ids of SET1 (`Settlements` 1 to 4, `SettlementClass` 1 to 4, `ClassLimit` 1 and 2, `RegionSettlement` 1 to 3) and `Role.Needs` 25, `Role.Ceiling` 26 |
| E3 | spec-config.md 3.3, 3.4 | the sort of classes by `Enter`, of flags by name and of limits by `Name` as the second exception (handoff row 14 of spec-settlements.md); the sentence in 3.7.1 if absent |
| E4 | spec-settlements.md 1, 17, 18 | status line; section 15 says "no open note carries the code SET" but the index has N-140, N-192, N-268, N-348: correct it; section 18 adds the continuation files; the `regionRow` helper named in SET4 |
| E5 | spec-observability.md 3.2.4, 4.1, 5.3.2 | confirm the names table gains the class IDs (`Names.SettlementClasses`) and that a module row field absent for want of the module is **omitted** from the golden rendering, not written as 0 (SET9, R8): the rule 2.10.5 says an appended field is never a typed ID, and `Class` is a `SettlementClassID`. If the spec says otherwise, the ruling is the owner's (Question 4) |
| E6 | spec-spawner.md 6.2, 6.3.0, 3.2; spec-population.md 3.7.3, 4.4, 10.1 | read both for the reader's side: where the headroom enters the plan (the count of the role's households at the start of the region's entry step, each admission reduces it), the order of the refusals, `RoleCeiling` and `BirthsCeiling` as appended fields; no edit unless a gap appears. Record the ruling for a candidate whose weight is above the headroom (spec 6.4.2: "only while the weight of the agent fits the headroom") |
| E7 | spec-world.md 6.1, 6.4, 9.4.1; ADR 0001, ADR 0006; spec-stability.md 5.1.3 | verify the six texts of spec 7.4 are already written (ADR 0006 exists); record, no edit if true |
| E8 | roadmap-v1.md Stage 3 row | `settlements.go`, `internal/settlement`; `Needs` stays "REG, SCL"; status `plan written`, then `PR A in development`. Add the continuation files if E1 lands |
| E9 | notes | verdicts of Risks, below; N-153, N-192, N-193, N-268 and N-348 get an update line, none is removed before its reader exists |
| E10 | `doc.go` glossary | SET12, before any code uses a term; the six type names |
| E11 | spec-reference-chain.md 4.5 | `overlay/settlements.toml` row and scenarios SET-S1 to SET-S5 (spec 12.4) if not yet in the table |

Audit (read-only, recorded in the result paragraph): which of the readers of 6.3 exist in the tree (H1, H2, H5, the SKL slot) and at which function; whether STO PR 3 has merged (R1); the Lizard figures of `fillRoles`, `positionsTail`, `Decide`'s admission function, `births`, `checkpointCross`, `announce`; which pack carries a role that is gated and a group in each region (Phase 7). The result paragraph lists, for the supervisor, the hooks of "Hook discipline" with the owner approval column.

Exit check: E1 to E11 applied or logged with an owner; `npx markdownlint-cli2` clean on every changed `.md`; `go test ./...` and `testdata/golden/` unchanged.

### Phase 2: `internal/settlement` (pure) (PR A)

Goal: every number of SET 3 and 4 as a function over primitives, no world. Files: `internal/settlement/{doc,thresholds,review,table,headroom}.go` and their tests, plus the `scripts/ci/coverage-thresholds.txt` row if not already added in Phase 1.

- `Thresholds(enter []int64, scaleBP, bandBP int64) (e, l []int64)` (FR-SET-008, 032: `E_k = ceil(Enter_k * s / 10 000)`, `L_k = floor(E_k * (10 000 - band) / 10 000)`, checked).
- `Review(c, floor int, h int64, e, l []int64) int` (the up loop, then the down loop only if nothing moved, never below the floor; FR-SET-010) and `Decide(c, c2 int, since, t, dwell int64) Outcome` with `Same`, `Changed`, `Held` and the ticks left (FR-SET-011, 012).
- `Initial(h int64, e []int64, floor int) int` (the construction class, no band; FR-SET-013) and `Target(...)` (the snapshot's `ClassTarget` and `ClassAtFloor`, FR-SET-026).
- `Table` (dense flags and limits: one `uint64` per class for the flags, one for which limits exist, the values) with `Has(class, flag int) bool` and `Limit(class, limit int) (int64, bool)` (FR-SET-005, 020). `Headroom(limit, households int64) int64` (`max(0, limit - households)`; FR-SET-023).
- Tests, table-driven, got/want, with the arithmetic of the spec in the comment: `TestThresholds_*` (the tables of 12.1 and 12.2, the band of one household for `Enter` 5 and 10 at 1 000 bp, `Enter` 3 at 5 000 bp, equal thresholds collapsing), `TestReview_*` (an **exhaustive** loop over counts 0 to 3 000, bands 0 and 1 000, floors none and 0, 1, 2, from every class: the model of spec 14.1 re-run as a Go test, so the invariant "inside its band unless at the floor" is checked, not sampled), `TestDecide_*` (dwell from the last change, a held change not remembered), `TestInitial_*`, `TestTarget_*`, `TestTable_*`, `TestHeadroom_*`, `TestThresholds_LargestCaseNoOverflow`, `TestPackage_NoRootNoName`, `TestPackage_NoFloat`, `FuzzThresholds`, `FuzzReview`.
- Exit check: `go test -race ./internal/settlement/...`; coverage at least 90%; `gomu` per new file in Docker (`--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS=4`; survivors killed or recorded equivalent); `lizard -l go -C 10 -a 8 -L 100 -w` clean.

### Phase 3: the configuration surface (PR A)

Goal: the module exists as data and is checked; at run time it does nothing. Files: `config.go` (`Settlements`, `SettlementClass`, `ClassLimit`, `RegionSettlement`, `Modules.Settlements` appended in Go order, `Role.Needs`, `Role.Ceiling`), `value.go` (`SettlementClassID`, `NoSettlementClass`), `bounds.go` (`MaxSettlementClasses` 16, `MaxClassFlags` 64, `MaxClassLimits` 64, `MaxBandBP` 5 000, `MaxThresholdBP` 100 000, each with the check row), `defaults.go` (`DefaultSettlements` with `BandBP` 1 000 and `Dwell` 30; the `canonical` sort of SET2), `errors.go` (the three sentinels, after the earlier modules'), `hash.go` (`pairsSettlements` in `pairsLaterModules` at id 11, `Role.Needs` 25 and `Role.Ceiling` 26 in `pairsRole`, written only when non-empty), `validate.go` (the call of `validateSettlements` at the module seam in the order of the `Modules` row; the role rules `needs_unknown` and `ceiling_unknown` run with the roles), `settlements_validate.go` (the rules of spec 9.1 and the seven warnings of 9.2: the warning for a class at construction uses the census of the groups, spec 4.1.1), `config/mapping.go` (`[modules.settlements]`, `.class.<id>`, `.class.<id>.limits`, `.region.<id>`; `needs` and `ceiling` on a role), `storage_validate.go` (SET11 (a): `needs_module_off` becomes the real rule), `registry_test.go` (the rows leave `notBuilt`, both directions), `internal/logkeys` (the five keys).

- Tests: `TestValidate_Settlements_<Rule>` one failing case per row of spec 9.1 asserting the sentinel with `errors.Is` and the path (`modules.settlements.classes[2].enter`), `TestWarnings_Settlements` (the seven), `TestCanonical_*`, `TestHash_*` of FR-SET-030, `TestLoad_Settlements*`, `TestSentinels_*` for the three, `TestValidate_StoreClassNeeds` (STO's field: unknown flag, flag carried, module off), `FuzzValidateSettlements`, `FuzzLoad` corpus extended. The registry rows of 8 (the five scopes, the two fields) leave `notBuilt`.
- Exit check: `TestHashConfig_ReferencePinned` and `TestGolden_OneOff` unchanged; the registry test staged both ways; `config/` coverage at least 90%; `go test -short -race ./...`.

**First pull request (PR A) ends here with Phases 1 to 3.** It adds no behaviour (a configuration with the module on builds a world that ignores it), lands the catalogue appends of the config files early, and is where Stage 3 modules collide.

### Phase 4: state, construction, the review at P7.2a, the events (PR B)

Goal: classes exist, change and are reported; no reader reads them yet. Files: `settlements.go` (state `settlements`, `newSettlements`, `regionHas`, `regionLimit`, `p72aSettlement`, the household scan), `settlements_report.go` (the events and `Debug` lines), `world.go` (`set *settlements`), `construct.go` (the call after the agents are created and before the first step; `modulesOn` gains `"settlements"` appended in Go order; the `founded` events after the `AgentSpawned` events; the construction line keys), `tick.go` (SET5), `event.go` (the two events, the causes, appended in module order after the existing events; `notBuiltEvents` loses two entries), `event_test.go`, `observe_internal_test.go`, `poshash_internal_test.go`.

- Tests: `TestConstruction_*` (the five regions of 12.1, floor `north`, scale `west`), `TestP72a_*` (change, hold, both-never), `TestDwell_FirstReleaseAtTickDwell` (4.4.3: regions founded together are held together), `TestPositions_P72aListed` (`notBuiltPositions` loses `"P7.2a"`), `TestHouseholdCount_*`, `TestBetweenTicks_ClassUnchanged`, `TestClass_ConstantWithinTick`, `TestEvent_*`, `TestInert_OneClassEqualsOff` (FR-SET-015, SET-S4 on a few seeds; Phase 7 runs the full range), `TestSettlements_NoDrawNoLedger`, `TestSettlements_OffLeavesNoTrace`, `TestSettlements_WorkIsCounted`.
- Exit check: `go test -short -race ./...`; the position test and the event catalogue staged both ways; `TestStep_AllocationsStayAtTheBaseline` rows with the module off unchanged; `golden/off/` unchanged.

### Phase 5: snapshot, logs, checkpoint tag 23, fork, allocation (PR B)

Goal: the module is observable and restorable. Files: `snapshot.go` (the four `RegionRow` fields at the end and the filler call), `settlements_report.go` (the filler, the encoder and decoder of tag 23), `checkpoint_sections.go` (row 23 added, entry deleted from `notBuiltSections`), `checkpoint_save.go` and `checkpoint_restore.go` (one call each from the section table, as SKL did), `internal/logkeys`, `cardinality_test.go` and `observe_events_test.go` (`SettlementClassChanged` and `SettlementHeld` rows), `otelobserver/catalogue_want_test.go`, `otelobserver/stage2_test.go` and `otelobserver/testdata/all_events.txt` (the two events are added to the lists; the adapter code is untouched, as SKL did), `allocation_test.go`.

- Tests: `TestSnapshot_ClassFields`, `_ClassAtFloor`, `_ClassTargetIgnoresDwell`, `TestLogs_Settlements*` (the construction line carries `settlement_classes` and `settlements` in `modules`; one `Warn` per warning; `Debug` only inside the tick and guarded by `Enabled`; constant messages), `TestCheckpoint_SettlementsRoundTrip` (a world with the module off writes **no** section and the bytes of every existing fixture are unchanged), `TestCheckpoint_SettlementsRefusesCorrupt` (class out of range, below the floor, `Since` above tick less 1, a section with the module off, a section missing), `TestFork_Settlements` (through `forkAtN` and `forkCase`: N in {0, in a hold, at the tick of a change, after a between-ticks call that moved the count past a threshold}; bit-identical, events included; FR-SET-029, SET-S5), `FuzzDecodeSettlements` (no panic, bounded allocation), `TestAlloc_SettlementsNoOp`, `TestNonInterference_Settlements`, `TestEvents_MetricsDerivable`.
- Exit check: `testdata/checkpoints/` untouched (the all-modules-on fixture is not regenerated; a module's restore test builds its world from its overlay in the test, N-257); the catalogue, cardinality and otelobserver list tests pass with the entries deleted; `golden/off/` event files unchanged.

**Second pull request (PR B) holds Phases 4 and 5.**

### Phase 6: the readers: H1, H2, the STO gate, the SKL headroom (PR C)

Goal: a class gates a role, bounds a role's households and fills the two seams. Files: `settlements.go` (`fillRoleGates`, `roleHeadroom`, the construction assignments), `phase_exit.go` (one guarded call in `fillRoles`), `spawner.go` (the appended view fields `Headroom` and `NoLimit`; the default decider's use), `internal/occupation` (a second per-role cap in `Limits` and the count of the cut), `spawner_report.go` and `event.go` (`EntryLimited.RoleCeiling`, `PopulationChanged.BirthsCeiling`), `population.go` (the births cap by headroom and the counter), `construct.go` (`w.skills.headroom = w.roleHeadroom`; STO's gate if PR 3 merged), `storage.go` (only if PR 3 merged: the gate and `store_locked_at_start`), `internal/logkeys`.

- Reader of H1: `fillRoles` sets `Closed` to `cfg.Closed || not regionHas(g, needs)` after the literal, through `fillRoleGates(g)`; the same view feeds the spawner (no entrant, no switcher, no like-for-like replacement, `NoViableRole`) and POP (`v.Roles[ra.Role].Closed` stops the birth carry from advancing, `population.go:489`), so one hook serves both. `populationCross` keeps asserting a zero carry for `cfg.Closed` only; a gated role's carry is **not** reset when it closes (spec 6.4.1, "does not advance"), so the cross-check is not widened (R5).
- Reader of H2: `fillRoleGates` sets `Headroom = max(0, limit - Households)` and `NoLimit`; the default decider takes the least of this and its own `roleCap` per role, counts the cut in `RoleCeiling`, and POP's `births` takes it into `population.Allow` and counts `BirthsCeiling`. A candidate whose weight does not fit the headroom is refused whole (E6 settles the ruling).
- SKL: `roleHeadroom(g, r)` returns the class half; ORG wraps it later.
- Tests: `TestGate_NeedsClosesRoleInRegion`, `_HouseholdsStay`, `_ClosedByEitherField` (`Closed` or `Needs`), `TestCeiling_*` of FR-SET-023 (the worked 9 + 4 case), `TestCeiling_FirstComeIsPlanOrder` (SET-A5: entrants in need, then agent ID), `TestRoleView_HeadroomOnlyWithModule`, `TestViews_CarryNoClass`, `TestReaders_StaticCheck`, `TestSkillHeadroom_HeldCompletionWithClassLimit` (a trainee waits at `TrainingTicks` while the target role's limit is 0 and completes the first entry step after the limit rises: the held path of plan 3-skl SK9 now with a real headroom), `TestStoreClassNeeds_GatesStore` (only if STO PR 3 merged; else a test-only gate in an internal test and a line in the result paragraph), `TestPopulation_BirthsCeilingCounted` (N-348: the `Needs` half of FR-POP-007 and `BirthsCeiling`), `TestEntryLimited_RoleCeilingField`.
- Exit check: `go test -short -race ./...`; `TestStep_AllocationsStayAtTheBaseline` and `TestDemand_AllocationsStayAtTheBaseline` at baseline for the module off; the existing spawner, population and skills goldens unchanged.

### Phase 7: overlay, golden, scenarios (PR C)

Goal: the module is proved end to end. Files: `testdata/chains/overlay/settlements.toml`, `testdata/chains/scripts/` rows if a scenario needs a script, `testdata/golden/settlements/{<pack>.txt,events-<pack>.txt,<pack>.positions.txt}`, `testdata/chains/expected/<pack>.scenarios.txt` rows, `scenario_set*_test.go`, a `TestDeterminism_Settlements` case, a `TestInvariants` row.

- **Pack that completes the feature.** Criterion 1 of Stage 2 is unmet, so packs `one`, `two` and `three` collapse; a collapse makes the class sequence non-increasing, which exercises `shrank`, `SettlementHeld` and the floor, but never `grew`. Recommended (Question 2): the module-alone golden on pack `five` (five regions as in spec 12.1; it keeps 182 of 5 000 households at tick 300 per the SKL findings, so a threshold set between the start and the end count gives a `shrank` and a hold in the window) with `overlay/settlements.toml`. A **probe test** over seeds 1 to 4 asserts the golden records at least one `shrank` and one `SettlementHeld`, and fails otherwise, so the golden exercises the module. `grew` and the staircase (SET-S1) are asserted on a layered world: `five` + `overlay/population.toml` (births, so households can rise). This is the one scenario that layers a second module, as the spec's SET-S1 says, and it is not the module-alone golden.
- Overlay: the classes of spec 12.1 re-based to the households of `five`, a gated role (an existing role of the pack, not a new `merchant` role) with a `needs` and a `ceiling`, the `west` scale row and the `north` floor row. The numbers are placeholders and say so in the file header (N-015, N-027).
- Scenarios: **SET-S1** (growing region, staircase at most `K - 1`, no two changes less than `Dwell` apart, the gate opens in the tick after the change), **SET-S2** (noisy count held on a threshold by external agents added and removed between ticks; default band and dwell never change the class, `BandBP` 0 and `Dwell` 0 do; `SettlementHeld` non-zero when the dwell binds), **SET-S3** (`famine` with a gated role the upkeep needs, with and without a floor; `needs_unmet_at_start` raised), **SET-S4** (one class, no gate, against the module off, over a range of seeds), **SET-S5** (a checkpoint taken during a held change, restored and run on; `TestFork_Settlements` of Phase 5 is the test). S4 and S5 are harness tests outside the band rule (spec 13). S1 to S3 report and assert only the invariants the spec lists; outcomes go in `expected/<pack>.scenarios.txt`, pass or not. If the probe chooses a pack with no `.scenarios.txt` today (`five` has `expected/five.toml` only) Phase 7 creates one in the form of an existing file.
- `golden/settlements/` is recorded once after the diff against `golden/off` is read; it is **never regenerated** to absorb a failure. `golden/off/`, `TestHashConfig_ReferencePinned`, `testdata/checkpoints/` are untouched. `golden/all/`, `all_pack_test.go` `allLayers` and the all-modules checkpoint fixture are **not touched by this change**: `Modules.Settlements` joins `allPackNotYetOn` (as `Modules.Storage` did), and the supervisor moves the all layer once at Stage 3 close.
- Exit check: S1 to S5 run in the quick pattern (`TestScenario`, `TestDeterminism`, `TestGolden`, `TestInvariants`, `TestFork`, `TestStylised` prefixes) so arm64 covers them; `TestInvariants` with `WithAudit` over `scenarioSeedCount()` seeds.

### Phase 8: the exit gate (PR C)

Gates run by hand on the dev host (testing.md 9.1) and the exit markers go in the PR description. Per the gate economics below: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`, `go test -short -race ./...`, gate R, gate F with `scripts/ci/coverage-check.sh` (every scope at its minimum, coverage not lower), `go mod tidy` no diff, in the root and in `otelobserver/` and each `examples/` module (the examples build against the public API; a new `RegionRow` field and `Role` fields are the only public additions, so no example change is expected). A deliberate-break pass (testing.md 9.3: flip `>=` to `>` in the up loop, drop the floor, swap the two directions, drop the dwell test, make the held change remembered) shows the tests fail. `gomu` per new file in Docker with the caps. Arm64: **once, before Stage 3 closes**, not per PR (owner decision). Close: roadmap row `merged (PR #...)`, notes N-153, N-192, N-193, N-268, N-348 and N-140 (the SET leg) settled or updated, `spec-roadmap.md` and `documentation_index.md` updated, plan audited with plan-audit.

## Pull request split

Three pull requests to `phase-3`, in order, each green on its own gates, each leaving `golden/off/`, the pinned hash and `testdata/checkpoints/` unchanged (the minimum is 1 500 lines and the maximum 3 000, each reviewable):

- **PR A (Phases 1 to 3): edits, `internal/settlement`, the configuration surface.** No behaviour. Size estimate: 2 600 lines including tests and the doc and spec edits. It lands the catalogue appends of the config files (`config.go`, `hash.go`, `bounds.go`, `defaults.go`, `errors.go`, `validate.go`, `config/mapping.go`, `registry_test.go`) early, which is where Stage 3 modules collide.
- **PR B (Phases 4 and 5): state, the review, events, snapshot, logs, checkpoint, fork.** The module runs and is observable; no role reads the class. Size estimate: 2 200 lines.
- **PR C (Phases 6 to 8): the readers, the seams, the overlay, the golden, the scenarios, the gate.** Size estimate: 2 800 lines of code and tests plus a golden of tens of thousands of lines (not counted: a recorded file). If review wants it, split after Phase 6 (the readers) and keep the golden and scenarios in a fourth, small PR.

**Minimal slice the roadmap implies.** "Needs REG, SCL" only, so the smallest slice is PR A and PR B (a class per region, reviewed and restorable, with no reader). That slice is useful to no one yet: STO's gate and SKL's headroom stay "no limit". The plan does not recommend stopping there.

## Catalogue and shared-file appends (merge-conflict forecast)

Other Stage 3 modules append to the same files. Appends are in module order and never inserted (rule 2.10.5), so a conflict is a textual one at the end of a list. Files marked * are the likely ones, and the supervisor orders the merges.

| File | SET append | Also touched by | Note |
| --- | --- | --- | --- |
| `config.go`* | `Modules.Settlements` (Go line after the existing pointers), the four types, `Role.Needs`, `Role.Ceiling` | every Stage 3 module (the `Modules` struct); EMP also appends to `Role` | PR A; keep both lines in module order |
| `hash.go`* | `pairsSettlements` id 11 in `pairsLaterModules`; `Role` 25 and 26 in `pairsRole` | every module (the `Modules` writer); EMP (`Role` 23 and 24) | the pair goes between ids 10 and 12 |
| `bounds.go`, `defaults.go`, `errors.go` | five bounds; `canonical` sort and `DefaultSettlements`; three sentinels | every module; `ErrUnknownSettlementFlag` is reused by STO, ORG and SPN, `ErrUnknownSettlementClass` by ORG | medium |
| `validate.go`*, `config/mapping.go`* | the call of `validateSettlements`; the TOML tables, `needs` and `ceiling` on a role | every module | high (the same lines) |
| `value.go` | `SettlementClassID`, `NoSettlementClass` | every module with a typed ID | low |
| `event.go`* | two events, three causes, `EntryLimited.RoleCeiling`, `PopulationChanged.BirthsCeiling`, `RoleView.Headroom`, `NoLimit` (in `spawner.go`), `Names` class IDs | EMP, STO, CAP, ORG, CRD (events and `AgentSpawned` fields) | high; land PR B early |
| `event_test.go`*, `observe_internal_test.go`, `cardinality_test.go` | two `notBuiltEvents` entries deleted; rows for the two events | every module | the lists are the conflict |
| `snapshot.go`* | four `RegionRow` fields at the end | STO, CAP, RSK, LND, FIS | appended after the columns of earlier modules |
| `tick.go`* | `{"P7.2a","SET",line}`, one guarded call in `positionsTail`, `notBuiltPositions` entry deleted | EMP, CAP, TEC, SAV, CRD (the Lizard CCN of `positionsTail` is 8 today) | the CCN helper split of plan 2-14 O7 may be needed |
| `world.go`, `construct.go`* | `set *settlements`; `modulesOn` name; the construction call; the two assignments to `skills.headroom` and STO's gate | every module | medium |
| `phase_exit.go` | one guarded call in `fillRoles` | EMP, ORG (membership), POP | one line |
| `spawner.go`, `internal/occupation/*`, `spawner_report.go` | `Headroom`, `NoLimit`, the second per-role cap, the `RoleCeiling` count | ORG (`not_member`, a body's places), SKL (done), POP | N-268 lists the order |
| `population.go` | the births cap by headroom, `BirthsCeiling` counter | none other, POP is built | owner of POP approves (Hook discipline) |
| `storage.go`, `storage_validate.go` | the gate and `store_locked_at_start` (if PR 3 merged), the real `needs_module_off` rule | STO PR 3 to 5 | **depends on STO's merge order** (R1) |
| `skills.go`, `skills_queue.go` | none in code: the assignment is in `construct.go` | SKL (merged) | the slot exists |
| `checkpoint_sections.go`*, `checkpoint_save.go`, `checkpoint_restore.go` | row 23, entry deleted from `notBuiltSections`, one call each | every module with a tag | the skeleton table is the conflict |
| `registry_test.go`* | the seven rows leave `notBuilt` | every module | the lists are the conflict |
| `internal/logkeys` | five keys | every module | low |
| `doc.go`* | the SET terms and six type names | every module | medium |
| `architecture.md` 2.6, `AGENTS.md` Layout*, rule files with a RULE-TAG* | the two continuation files, a fresh tag | every module that adds a file | the tag line always conflicts: the supervisor applies these edits |
| `scripts/ci/coverage-thresholds.txt` | `internal/settlement 90` | every module with a new `internal/` package | one line; PR A |
| `otelobserver/catalogue_want_test.go`, `otelobserver/stage2_test.go`, `otelobserver/testdata/all_events.txt` | the two events | every module that adds an event | tests and a text file only; adapter code untouched |
| `testdata/chains/overlay/settlements.toml`, `testdata/golden/settlements/`, `testdata/chains/expected/<pack>.scenarios.txt` | new files and rows | STY, CLB | low |
| `testdata/golden/all/`, `all_pack_test.go`, the all-modules checkpoint fixture | **none** (owner decision) | every module | moved once at Stage 3 close |

## Hook discipline

Each hook is **one guarded call into a method that `settlements*.go` defines**, or one appended field of a core type. A hook holds no branch on module data and no arithmetic (architecture.md 2.10.1, 2.10.2). Every line below needs the owner's (supervisor's) approval before PR B and PR C start; none is new in kind. Where a hook would carry logic, the logic moves into `settlements*.go` and the hook is the call.

| # | File | Function | The call or field | Kind (ADR 0010, 2.10.1) |
| --- | --- | --- | --- | --- |
| H-1 | `world.go` | the `World` struct | field `set *settlements` | pointer in `World` (2.10.2) |
| H-2 | `construct.go` | the construction sequence, after the agents and before the first step | `w.set = newSettlements(&w.cfg, ...)`, guarded by `w.cfg.Modules.Settlements != nil`; the `founded` events are emitted from `announce` by one call `w.set.announce()` after the `AgentSpawned` events | construction calls (d) |
| H-3 | `construct.go` | `modulesOn` | one `{"settlements", m.Settlements != nil}` row appended in Go order | catalogue |
| H-4 | `tick.go` | `positionList`, `positionsTail` | the row `{"P7.2a", "SET", line}`; `if w.set != nil { w.p72aSettlement() }` after `p72Migration`; `"P7.2a"` deleted from `notBuiltPositions` | tick straight line (2.10.3) |
| H-5 | `phase_exit.go` | `fillRoles` | `if w.set != nil { w.fillRoleGates(g) }` beside the existing `if w.skills != nil { w.fillRoleSkills(g) }`; sets `Closed`, `Headroom`, `NoLimit` for the region | sub-step guarded call (e) and field of a view (f) |
| H-6 | `spawner.go`, `internal/occupation` | `RoleView`; the default decider's admission; `Limits` | appended fields `Headroom`, `NoLimit`; the decider takes the least of its cap and the headroom; `RoleCeiling` counted | field of a view (f) and the default rule; **a rule change in `internal/occupation`, not a module call: the supervisor must approve it as SPN's** (N-268) |
| H-7 | `population.go` | `births` | the headroom enters `population.Allow` and `birthsCeiling` is counted | **an edit of another module's file (POP)**, owed by the spec (6.4.2, N-348); approve as POP's reader side |
| H-8 | `event.go`, `snapshot.go` | `EntryLimited`, `PopulationChanged`, `RegionRow`, `Names`, the events | appended fields and types in module order | catalogue files |
| H-9 | `snapshot.go` | the `RegionRow` builder | `if w.set != nil { w.fillRegionClass(&row, g) }` | guarded call in a filler |
| H-10 | `construct.go` | after `newSkills` | `if w.skills != nil && w.set != nil { w.skills.headroom = w.set.roleHeadroom }`: one assignment; SKL's slot | construction call (d) |
| H-11 | `storage.go`, `storage_validate.go` | STO's gate and `needs_module_off` | `w.sto.gate = w.set.regionHas` (if STO PR 3 has merged) and the validator rule | STO's seam (plan 3-sto D8); depends on merge order (R1) |
| H-12 | `validate.go`, `config/mapping.go`, `hash.go`, `defaults.go` | the module seam, the loader, the writers, `canonical` | one call or row each | catalogue / loader keys |
| H-13 | `checkpoint_sections.go`, `checkpoint_save.go`, `checkpoint_restore.go` | the section table | row 23 and one call each; `auditModules` gets none (SET holds no money, goods or conserved quantity) | checkpoint hook |

Not hooks, because they are not needed: no `agentJoined`, `agentRoleChanged`, `dropAgent`, `relocationRefusal` or `checkTradePlan` entry (SET keeps no per-agent state and registers no `region_bound` obligation: N-090, spec 15; the count is a scan: SET4), no `tradableItems` item, no `orderFacts` fact, no `option.go` option, no `integration*.go` call, no `ledger.go` row.

## Gate economics

- Full gates (gate R and gate F, `check.sh`, `full.sh`) run **once per PR, on the final head**, with `GOMAXPROCS=8`, and **one gate run at a time on the host**: earlyoom kills the 20-plus GB root test binary when two share the machine (three gate F runs of POL PR B were killed at 19, 28 and 38 GB). While a gate runs, no other session starts one.
- During a phase, run only the focused tests and `go test -short -race ./...` (and `-run` on a phase's own tests); do not run gate R or F per phase, per commit or per fix.
- If a gate run fails, fix, then rerun only the failing package; the full gates are rerun once more on the final head if the fix touches non-test code outside that package.
- `gomu` only inside Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`, per new file, never on the host; it does not overlap a gate run.
- `examples/realm` `TestServe_*` can time out under `-race` when several gate runs share the machine; a rerun alone settles it (not in this module's scope to harden).

## Codacy hygiene (every phase)

The Codacy gate fails on **any new Info-or-higher issue** (not on coverage, complexity or duplication). Before a commit that ends a phase, and before every push:

- Every changed `.md` (this plan, specs, rule files, notes, ADRs, the roadmap) passes `npx markdownlint-cli2 <file>`: a blank line around every list and heading and fenced block (MD032, MD031, MD022), table separators written as `| --- |` (MD060: the separator row's pipes must be spaced the same way as the header), no double blank line (MD012), no trailing space, a single trailing newline, no hard tab outside a code block. `.markdownlint.json` already disables MD013 (line length), MD037 and the code-block form of MD010.
- Every new or changed Go function stays within **Lizard CCN 10, NLOC 50, 100 lines** (`lizard -l go -C 10 -a 8 -L 100 -w <files>`), test functions included: split a table test into subtests rather than one long function (SKL's PR needed this). A hook that raises a shared function's CCN is preceded by a refactor-first commit (the CCN of `fillRoles`, `positionsTail`, `births`, the admission function, `checkpointCross` and `announce` is measured in Phase 1).
- `gofmt -l .` empty, `go vet ./...`, `staticcheck ./...` clean on the changed packages; no new unchecked error, no shadowed variable, no unused parameter or result.
- Nothing is pushed until these pass on the files the push changes.

## Cross-plan contract

Consumes (identifiers as the tree has them at `d996592`; re-check at implementation):

- `skills.headroom`, `completionHeadroom` <- plan 3-skl SK9
- the `StoreClass.Needs` field, the hash id 4, `ErrStoreLocked`, the rule `needs_module_off` <- plan 3-sto PR 2; the gate and `StoresLocked` <- plan 3-sto PR 3 (not merged)
- `RoleView`, `occupation.Limits.RoleCap`, `Admit` <- Stage 1 and plan 2-14; `EntryLimited`, `PopulationChanged` <- plans 1-9, 2-11
- `regionBounds`, `relocationRefusal` <- plan 2-3 (SET registers none)
- `positionList`, `notBuiltPositions`, `checkpointSections`, `notBuiltSections`, `forkAtN`, `forkCase`, the `Warn` collector of `validate.go` <- plans 2-2, 2-14
- `internal/checked`, `internal/ordering`, `internal/logkeys` <- Stage 0

Provides:

- `regionHas(r, flag)`, `regionLimit(r, limit)`, `roleHeadroom(g, r)`, the class as `RegionRow.Class` and `ClassTarget` -> STO (`Needs`), ORG (`Body.Needs`, a body's places as a second limit: N-192), CAP (`FacilityClass.Needs`), SKL (the held completion), POP and SPN (H1, H2), IND (a settlement-class preset or tally, N-130, N-131)
- `SettlementClassChanged`, `SettlementHeld`, the causes, `EntryLimited.RoleCeiling`, `PopulationChanged.BirthsCeiling` -> `otelobserver` (later instruments), IND
- `overlay/settlements.toml`, `testdata/golden/settlements/`, SET-S1 to S5 -> STY, REF

## Dependencies on other Stage 3 modules

SET needs REG and SCL only (roadmap), both merged. It **fills seams** left by modules that built before it and is **filled** by modules that follow:

| Module | Relation | Seam here |
| --- | --- | --- |
| STO | reads SET (`StoreClass.Needs`, `ErrStoreLocked`, `StoresLocked`) | PR A flips the validator rule; the runtime gate is `regionHas` (SET11 a, R1) |
| SKL | reads SET (the held completion's headroom) | one assignment (H-10) |
| POP, SPN | read SET (`Closed` by `Needs`, `Headroom`, `BirthsCeiling`, `RoleCeiling`) | H-5 to H-7 |
| ORG | reads SET (`Body.Needs`, a body's places into the same headroom) | `roleHeadroom` is the slot ORG wraps; ORG is listed "POL, SKL, SET" |
| CAP | reads SET (`FacilityClass.Needs`) | none here; CAP adds its row to spec 6.3 (R7) |
| RSK | declined (spec 6.3 H7) | none |
| EMP, LND, CRD, INS, FIS, PLC, SAV, TEC, POL | none | textual conflicts only |

Build order is independent of EMP, LND, POL and the rest. Only **STO PR 3** and **ORG** care: if STO PR 3 lands first, SET completes the gate (H-11); if SET lands first, STO PR 3 reads `regionHas` and the one-line seam is STO's.

## Testing

Testing entry (testing.md 11) for the PRs: the requirements and tests are in the table above. The lowest level is the `internal/settlement` unit test wherever the rule is pure (the exhaustive review table is a unit test, not a model), the root integration tests of Phases 3 to 6 for what needs a world, and SET-S1 to S5 and the module-alone golden at the top.

- **Golden files.** `golden/off/` (`TestGolden_OneOff`, `TestGolden_OneEvents`, `TestHashConfig_ReferencePinned`) and `testdata/checkpoints/` do not move in any phase. `testdata/golden/settlements/` is new, recorded once and never regenerated to absorb a failure. `golden/all/`, `allLayers` and the all-modules checkpoint fixture are not touched by any module PR; `Modules.Settlements` joins `allPackNotYetOn`.
- **Pack.** The module-alone golden is on a pack chosen by a probe that fails if no `shrank` and no hold is recorded; `grew` is asserted on `five` + `overlay/population.toml` (Question 2). Packs one, two and three collapse (criterion 1 of Stage 2); a collapse-only golden proves only the downward half.
- **Registry and catalogue.** The seven rows leave `notBuilt`; `notBuiltEvents` loses two; `notBuiltPositions` loses `P7.2a`; `notBuiltSections` loses tag 23.
- **Fuzz.** `FuzzThresholds`, `FuzzReview`, `FuzzValidateSettlements`, `FuzzDecodeSettlements`, `FuzzLoad` corpus extended.
- **Fork-at-N.** `TestFork_Settlements` (Phase 5).
- **Mutation.** `gomu` only in Docker with the caps, per new file of `internal/settlement` and `settlements.go`.
- **Broken-version pass.** Phase 8 (testing.md 9.3).

## Observability

Observability entry (observability-and-logging.md 9).

- **Events:** `SettlementClassChanged` (per region, per change, plus one `founded` per region at construction) and `SettlementHeld` (an aggregate per region per tick of a hold), plain data, no map, constructed only under `w.obs.supplied`, after the transition, in the order of SET8 (rules 4.1, 4.3, 4.5). `EntryLimited.RoleCeiling` and `PopulationChanged.BirthsCeiling` are appended counts, emitted only when non-zero, as the events already are.
- **Logs:** `Info` only the existing construction line, which gains `settlement_classes` and `settlements` in `modules`. `Warn` at construction only, one per warning of spec 9.2 with `rule` and `path`, bounded by the configuration; **none inside the tick**; the counterparts are the events and the snapshot (rule 8.1: `SettlementHeld` and `ClassTarget` say what the dwell held). `Debug`: "settlement class changed" and "settlement class held", guarded by `Enabled`. No `Error`.
- **Metrics:** none built. Spec 10.4 proposes `economysim.settlement.class` (gauge) and `economysim.settlement.changes` (counter), both attributed by region; the catalogue rows and the `otelobserver` instruments are deferred (Question 6), as SKL's and STO's were. The events carry bounded identifiers, so the instruments can be added without a core change. No new metric attribute in this change; cardinality is bounded by the regions.
- **Non-interference:** nothing draws; a run with and without observer and logger is identical (Phase 5 test).
- **New log keys:** the five of spec 10.3, in `internal/logkeys` and rules 3.3.2 (E1 edits the rule file only if the keys are not listed there; fresh RULE-TAG).

## Risks and open items

| # | Item | Owner |
| --- | --- | --- |
| R1 | **STO PR 3 has not merged.** The runtime gate, `StoresLocked` and `store_locked_at_start` live there, so the order of the two decides who writes the nil-gate shape (plan 3-sto Risk 3). SET writes the reader (`regionHas`) either way; the one-line seam is whichever module lands second. Phase 1 reads the tree and records which case holds | STO, SET owners; the supervisor orders the merges |
| R2 | **The reference chain collapses.** Packs one, two and three go extinct, `five` keeps 182 of 5 000 households at tick 300. A collapse-only golden records `shrank` and holds but never `grew`, and SET-A1 (the staircase) needs growth. Mitigation: a probe that fails an empty golden, and SET-S1 on `five` + `population.toml`; a result such as "a gate helps a region recover" cannot be told from the baseline and is reported, not asserted | SET owner, STY |
| R3 | **The first release of regions founded together** (spec 4.4.3, open question 4): every region is held from tick 0 to tick `Dwell` and released together. The golden records it; a bucketed start is not built (five regions need none) | SET owner |
| R4 | **A rule change in the default spawner and in POP.** H-6 and H-7 edit `internal/occupation`, `spawner.go` and `population.go`, which belong to SPN and POP. Each is small, but it moves behaviour **only where a role has `Needs` or `Ceiling`** (every pack of the tree has none), so the existing goldens cannot move. A wrong ruling on how a candidate larger than the headroom is admitted (E6) would change a result for a gated pack only | SPN, POP owners |
| R5 | **`populationCross` and the birth carry.** A role that a class gate closes keeps its carry (the carry "does not advance", spec 6.4.1). The restore cross-check asserts a zero carry only for `Role.Closed`; it must not be widened to the gate, or a valid world is refused. Tested in Phase 6 | POP owner |
| R6 | **Lizard on shared functions.** `fillRoles` (one guarded call added), `positionsTail` (CCN 8, plus 1), `births`, the admission function and `announce` may sit at the limit; Phase 1 measures them and a refactor-first commit lands before the hook (the STO N-357 precedent) | SET owner |
| R7 | **`internal/settlement` coverage and the exhaustive table.** The count range 0 to 3 000 times the floors times the bands is about 50 000 reviews; the test stays under a second. If it does not, the table runs a stride and the full range sits behind `-short` skipped | SET owner |
| R8 | **An appended typed ID.** Rule 2.10.5 forbids a typed ID as an appended field of an event or snapshot row (zero is a valid ID), but `RegionRow.Class` is a `SettlementClassID` and spec 10.5 says "zero with the module off". Proposed reading: the golden writer omits the four fields when the module is off, so `golden/off/` does not move, and `Modules` says which are on. E5 and Question 4 ask for the ruling before Phase 5 | SET, OBS owners, the owner |
| R9 | **The warnings that need a census at validation.** `needs_unmet_at_start`, `ceiling_below_start` and `class_unreachable` need the class at construction, so `Validate` computes it from the groups (spec 4.1.1). A group of weight above 1 or a scale makes the census equal the sum of `Households`; a test pins it against the real construction class | SET owner |
| R10 | **A gated role the upkeep needs** (SET-A2, the absorbing class, and SET-A3). The module does not stop it; the warning and the floor are the damper. The default `Dwell` 30 equals the welfare window, `RoleCooldown` and `MoveReviewTicks` (spec 13): a place where cycles form, and SET-S1 is the test | STY, the pack author |
| R11 | **First-come ceiling** (SET-A5): when a ceiling binds, entrants are served in the plan's order (need, then agent ID). D87 is not adopted here; the counter is `RoleCeiling` | SPN owner |

Note verdicts (text only; the supervisor edits `doc/notes.md`): N-140 apply the SET leg (the gate of a store class) in Phase 6 and keep the other legs. N-153 leave open until Phase 6, then remove (SPN and POP readers built). N-192 leave open, with this added: SET ships the class half of the headroom and the slot; ORG supplies the body's. N-193 apply in Phase 6 (a real headroom for the held completion) and remove the "waits for SET" line. N-268 apply for SET (the fields `Headroom`, `NoLimit`, `EntryLimited.RoleCeiling`, folded `Needs`), then remove. N-348 apply in Phase 6, remove. N-090 keep (SET registers none, confirmed). N-017 and N-027 standing (terms and placeholders, spec 15). N-130 and N-131 leave open (`Class` and `ClassTarget` are the fields an indicator needs).

## Questions only the owner can answer

1. **STO PR 3 and SET: which merges first?** STO PR 3 holds the runtime gate that SET fills. Recommended: merge SET PR A and PR B first (they touch no STO runtime file), then STO PR 3, which then reads `regionHas`; SET PR C completes `store_locked_at_start` in its H-11 row if STO PR 3 is already in, and leaves one line for STO otherwise. Either order works; the plan only needs to be told.
2. **Which pack carries the module-alone golden?** Recommended: pack `five` with `overlay/settlements.toml`, chosen by a probe that fails an empty golden, and `grew` asserted on `five` + `population.toml` (SET-S1). The spec's own example (the five regions) is `five`; `one` collapses and has one region.
3. **Household count: a scan at P7.2a or a maintained counter?** Recommended: the scan (SET4). A counter adds four hooks to the lifecycle fan-ins for a cost the scan already meets.
4. **An appended `SettlementClassID` in `RegionRow`** (rule 2.10.5 forbids a typed ID as an appended field; spec 10.5 wants `Class`, `ClassSince`, `ClassTarget`, `ClassAtFloor`). Recommended: keep the types the spec says and have the golden writer omit all four fields when the module is off, so `golden/off/` does not move; the alternative is plain `int64` fields, which weakens the type for a consumer.
5. **Rule changes in SPN's and POP's files (H-6, H-7).** Approve SET editing `internal/occupation`, `spawner.go` and `population.go` (the reader side, owed by the spec and by N-153, N-268, N-348)? Recommended: yes, as the spec says it, each a small edit reviewed with PR C, moving behaviour only where a role has `Needs` or `Ceiling`.
6. **Metrics:** the two instruments of spec 10.4 and the `otelobserver` adapter. Recommended: defer to a later change to `otelobserver/`, as SKL and STO did; the events carry the data and `TestEvents_MetricsDerivable` shows it.
7. **A candidate larger than the headroom** (spec 6.4.2, "only while the weight of the agent fits"). Recommended: refuse the whole candidate and count its households in `RoleCeiling`, rather than admit a part of an agent; the spawner's `Admit` already takes whole candidates against a cap, and the weights of the first consumer are 1.
8. **Number of pull requests: three or two?** Recommended: three (A: edits, pure rules, configuration; B: state, review, events, checkpoint; C: readers, golden, scenarios, gate), so the catalogue appends land early and the readers are reviewed with their proof.
