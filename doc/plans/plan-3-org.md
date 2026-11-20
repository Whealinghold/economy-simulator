# Plan: Stage 3, row ORG: `institutions.go` and `internal/institution`

**Status: plan written (2026-11-18), stage 1 of the work (requirements and plan, no code). Written on branch `s3-org-1`, cut from the integration branch `phase-3` (= `15bcf18`, which holds the merged modules SET, STO, SKL, POL, CRD, TEC, LND and the merged pull requests of MER, MIG and EMP's first part). Format model: `plan-3-set.md` (a module with a hash id, a checkpoint tag, seams to fill and a reader's side in other modules' files), `plan-3-lnd.md` and `plan-3-crd.md` (the PR split and the hook list). Reviewed once against the plan-review checklist (seven dimensions; two items fixed, none left blocking); the owner questions are DELEGATED to the supervisor: each carries a recommendation and the plan treats it as accepted.**

Sequences the roadmap-v1.md Stage 3 row "`institutions.go`, `internal/institution` | ORG | POL, SKL, SET" into eleven phases and five pull requests. It implements [spec-institutions.md](../specs/spec-institutions.md) (ORG): FR-ORG-001 to 055 and NFR-ORG-001 to 004, with the ORG parts of [spec-config.md](../specs/spec-config.md) 3.7.1, 8.2.2 and FR-CFG-094, [spec-checkpoint.md](../specs/spec-checkpoint.md) 4.2 (tag 3's agent tail and tag 31), [spec-world.md](../specs/spec-world.md) 6 (P1.9a, P7.2b), [spec-integration.md](../specs/spec-integration.md) 4.7.2 and 5.1 (the two slots and the membership group), and the rows of `spec-institutions.md` 8 owed to PRO, CLR, AGT, MER, CRD, SKL, SET, SPN, POP, SUB, INT, OBS and CHK.

Rule tags read (the three files, in full): architecture=5c0416f502ba5a59f7b2ae08, observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367.

Notes tag (`doc/notes.md`): f85c4b2d7217a479ec7c16dc. This plan writes no spec, so the notes rule of AGENTS.md does not bind it. It read the index, the first lines of the ORG-tagged notes (N-140, N-152, N-155, N-160, N-161, N-172, N-173, N-205, N-244) and of the notes that name this spec's seams (N-090, N-122, N-131, N-192, N-193, N-194, N-268). It did **not** read the ALL notes (N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217, N-315, N-335, N-340, N-345, N-351, N-353) or the bodies of the ORG notes in full. **Phase 1 edits specs and must read the whole of `doc/notes.md`, quote its tag, and give each ORG and ALL note a verdict.**

Specs read: spec-institutions.md in full; spec-settlements.md and plan 3-set (the headroom, the flag, the `roleHeadroom` slot); spec-skill-barriers.md through plan 3-skl (the places, the completion, the headroom slot); spec-clearing.md 3.2 and 3.2.1 through `internal/clearing/admit.go`; spec-merchants.md 9.3 and spec-credit.md 5.2.3 through `contract.go` and `credit_round.go`. Not re-read for this plan (Phase 1 reads them before the readers are built): spec-production.md 4.2, spec-agents.md 6.5.2, spec-spawner.md 6.2 to 6.3, spec-population.md 3.7.3 and 4.4, spec-integration.md 4.7.2 and 5, spec-checkpoint.md 4.2 and 6.2, spec-subsistence.md 14.1, spec-polity.md 6.1 and spec-world.md 8.2 and 10.2. The code seams were read in the tree.

## What is already in the tree (from `15bcf18`)

- **Config surface, none of it built.** `registry_test.go` lists `Modules.Institutions` and the scopes `Institutions`, `Body`, `Right`, `Control` and `RateLimit` in `notBuilt`. `errors.go` has `ErrInvalidID`, `ErrDuplicateID`, `ErrTooMany`, `ErrModuleDependency`, `ErrOutOfBounds`, `ErrUnknownRole` and the other reference sentinels, and no `ErrUnknownBody`, `ErrBodyShape`, `ErrBodyOverlap`, `ErrRightShape`, `ErrControlShape`, `ErrUnknownControl` or `ErrUnknownRight`. `bounds.go` has `MaxBasisPoints` (1 000 000), `MaxHouseholds` and `MaxLabelBytes` and none of `MaxBodies`, `MaxRights`, `MaxControls`.
- **The hash.** `hash.go` writes `Modules` ids 9 to 15, 19, 24 and 25 in `pairsLaterModules` and `pairsLastModules`; id **21** (the position of `Institutions` in the row of spec-config.md 3.2) is free and is ORG's. `Role` ids stop at `Needs` 26 and `Ceiling` 27 (SET); **ORG adds no `Role` field**, so it takes no `Role` id.
- **SET left the headroom and the flag.** `settlements.go` has `roleHeadroom` (`:237`), `fillRoleGates` (`:217`), `noteRoleMove` (`:269`) and the slot assignment `skills.headroom = roleHeadroom`; `regionHas` and `regionLimit` read the class. `RoleView` has `Headroom` and `NoLimit`; `EntryLimited.RoleCeiling` and `PopulationChanged.BirthsCeiling` are built. ORG adds a second source of a limit to the same headroom and **no field** (spec-institutions.md 4.5.1).
- **SKL left the places and the completion.** `skills_queue.go`: `canTrain` (`:114`), `trainersOf` (`:121`), `freest` and `fillPlaces` (`:134`, `:183`, which uses the role's `MaxTrainees`), `completeReady` (`:61`, which reads `completionHeadroom` through the slot), `fillRoleSkills` (`:319`). `internal/skill/queue.go:16` says "an institution may" lower a trainer's maximum. `skills_report.go:296` says the same. SKL's `Held` count for a held completion has to be audited in Phase 1.
- **CRD left the rate seam.** `credit.go:41` has `rateCap func(RegionID) (int64, bool)`, nil today, and `credit_round.go:139-163` replaces a request's limit by it and counts `priceControlled`, "which no event carries yet" (`credit_round.go:49`). The count is cleared with the round (`rd.reset()`), so P7.2b cannot read it: see decision D8.
- **MER left the rows.** `contract.go:977` and `:1121` say that row 5a (`not_permitted`) of an offer and of a take "have no check yet". The legal clamp of spec-institutions.md 6.7 has no seam.
- **CLR left the positions.** `internal/clearing/admit.go:69` and `:80` mark the place of `not_permitted` (after `NotTradable`) and of the legal clamp (after `headroomRow`, before `fundsRow`); `codeReasons` (`clearing.go:75`) says "the rows of the modules (not_permitted) are added with their codes". `Admit(f Facts) Code` returns a code only, so a clamp that replaces the limit needs a second result (D6).
- **PRO has the first check.** `production.go:240` `admit` returns `ReasonNotOnRole`; `phase_production.go:285` `admitJob` calls it first and `refuse`s. `ReasonNoEndowment` and `ReasonNoTitle` exist (`event.go:414`, `:418`); `ReasonNotPermitted` does not.
- **The agent and the checkpoint.** `agent.roleSince` exists (`agent.go:67`, set at creation, at a role change `phase_exit.go:557`, at an arrival `migration.go:369` and at a relocation `integration.go:394`). `agentTails` is empty (`checkpoint_sections.go:94`) and its loop is "tested by the module that appends" (`agent.go:322`). `notBuiltSections` has `{31, "Institutions", "Institutions"}`. `AgentState` (`agent.go:682`) has no membership group. `SlotState` ends with `RentScales` (`integration_slots.go:451`).
- **The positions.** `tick.go` lists `"P1.9a"` and `"P7.2b"` in `notBuiltPositions`. `P1.9` (the directives) is in `positionsFlows`.
- **Polity.** `Polity.Chain(r)` exists (`polity.go:179`) and `internal/polity` resolves the innermost mode; ORG reads it at construction only.
- **Pieces ORG reuses.** `internal/checked`, `internal/ordering` (`Key(tick, id)`, used by CRD at `credit_round.go:160`), `internal/logkeys` (which has the SET keys and **none** of the ORG keys; the rule table of observability 3.3.2 already lists them), the `Warn` collection of `validate.go`, the census of the groups that SET's warnings use (`settlements_validate.go`), `forkAtN` and `forkCase`, and the helper pattern of `agentTails`.
- **Not built, so not ORG's to read:** EMP's hire form (`Employer.Recipes`, `HireFormed` and the others are in `notBuiltEvents`), CAP, RSK, FIS and PLC. The `Who` test of a hire (spec 5.3) has no code to hook; D10.
- **Layout.** `institutions.go` and `internal/institution/` are in architecture.md 2.6 and AGENTS.md. Continuation files are not listed.

## Scope

**Builds.**

- `internal/institution` (new package; stdlib, `internal/checked` and `internal/ordering` only; primitives only): the innermost resolution of a table keyed by (subject, scope key) over a chain (`Resolve`), the legal range in minor units with checked arithmetic (`Range`, FR-ORG-028), the clamp (`Clamp`, with the count of a lowered bid and a raised ask), the membership round (`Lapse` and `Admit` over parallel slices: eligible flag, weight, `RoleSince`, member region, key), the headroom (`Headroom`) and the least of two limits (`Least`). No body, right, role, region or commodity name.
- Root: `institutions.go` (module-file order of architecture.md 2.10.1: state `institutions`, `newInstitutions`, the resolved tables, the membership test, the headroom reader) and the continuation files `institutions_validate.go` (VAL: the rules of spec 10.1 and the warnings of 10.2), `institutions_round.go` (the round at P1.9a and the counts), `institutions_rights.go` (the readers of the four activities and the `RightRefused` counts), `institutions_legal.go` (the ranges, the clamp, the slots `SetLegalLevels` and `SetRightSuspensions`, the rebuild) and `institutions_report.go` (the events of P7.2b, the snapshot filler, the logs, the codec of tag 31 and the agent tail of tag 3). The split is for the Lizard limits of N-243 and the 500-line file limit of Codacy; Phase 1 confirms it (E1).
- Catalogue appends: the table "Catalogue and shared-file appends" below.
- The readers of spec 8: PRO (`not_permitted` after `not_on_role`), AGT and CLR (`not_permitted` as an admission code, the legal clamp in `internal/clearing`), MER (rows 5a and the clamp in the contract round), CRD (the rate seam), SKL (the trainer rule, the body's `MaxTrainees`, the applicant count, the held completion), SET (the least of two limits in `roleHeadroom` and `fillRoleGates`), SPN and POP (nothing new: they read the same view), SUB (`not_permitted` is a reason of `Blocked` only with `OwnUseBound`), INT (the two slots, `SlotState` and the membership group of `AgentState`), CHK (tag 3's tail and tag 31) and OBS (five events, the names table, three snapshot fields).
- `testdata/chains/overlay/institutions.toml` and `institutions-inert.toml`, `testdata/golden/institutions/` (module alone), scenarios ORG-S1 to ORG-S5 (= S137 to S141 of spec-reference-chain.md 7.12), band keys none.

**Public identifiers.** `BodyID`, `NoBody`, `RightID`, `ControlID`, `Institutions`, `Body`, `Right`, `Control`, `RateLimit`, `LegalLevel`, `RightSuspension`, `Modules.Institutions`, the constants of the four activities and of `Who` (`ActivityProduce`, `ActivitySell`, `ActivityBuy`, `ActivityTrain`, `WhoPrincipal`, `WhoExecutor`, `WhoBoth`; the spelling is fixed in Phase 1 against the constants already in `event.go`), `ReasonNotPermitted`, the events `MembershipChanged`, `RightRefused`, `PriceControlled`, `LegalLevelSet` and `RightSuspensionSet`, the snapshot rows `BodyRow` and `LegalRow` with `Snapshot.Bodies`, `LegalLevels` and `Suspended`, `Names.Bodies`, `Rights` and `Controls`, the calls `SetLegalLevels` and `SetRightSuspensions`, `SlotState.LegalLevels` and `RightSuspensions`, `AgentState.Membership`, the sentinels `ErrUnknownBody`, `ErrBodyShape`, `ErrBodyOverlap`, `ErrRightShape`, `ErrControlShape`, `ErrUnknownControl`, `ErrUnknownRight`, the bounds `MaxBodies`, `MaxRights`, `MaxControls`. **No** option, no `region_bound` obligation (FR-ORG-009), no ledger reason (so none of the allocated codes 56 to 59 is used), no draw.

**Leaves out, with the owner.**

| Item | Owner |
| --- | --- |
| The `Who` test of a hire and a closed shop in EMP's formation (spec 5.3, FR-EMP-023) | EMP, when its hire form lands (D10); ORG leaves a note |
| A wage control; a right over carriage (`ship`), a store or a facility; a right that names a household; a call that admits or expels (spec 5.8, 6.8, 18) | reserved, with their shapes |
| A body's account, dues, a treasury (open question 1) | FIS |
| A lever of PLC of kind `legal` over `SetLegalLevels` | PLC (N-205) |
| `otelobserver` instruments for the four metrics of spec 11.6 and the two slot events; the catalogue rows are a doc change | a later change to `otelobserver/` (Question 8) |
| The preset asks `BodyShare` and `LegalPriceGap` | IND |
| Band values for ORG scenarios; stability of any default | STY and CLB; every number is a placeholder (N-015, N-027) |
| A `golden/all/` and all-modules checkpoint-fixture move | the supervisor, once at Stage 3 close |

## Decisions

Each is the spec's, a rule's or a note's. The plan decides nothing new except where a Question says so.

- **D1. Hash ids and the Go order.** `Modules.Institutions` has hash id **21**; its Go line is appended after the pointers that exist at merge time (architecture.md 2.10.5), and its pair goes in the writer between ids 19 and 24 (a short continuation of `pairsLastModules`). Inner ids, in declaration order (spec-config.md 8.2.2 gets the rows in E2): `Institutions` `Bodies` 1, `Rights` 2, `Controls` 3, `RateLimits` 4, `OwnUseBound` 5; `Body` `ID` 1, `Roles` 2, `Regions` 3, `Jurisdictions` 4, `Places` 5, `MaxTrainees` 6, `Needs` 7; `Right` `ID` 1, `Body` 2, `Activity` 3, `Subject` 4, `Region` 5, `Jurisdiction` 6, `Who` 7; `Control` `ID` 1, `Commodity` 2, `Region` 3, `Jurisdiction` 4, `CapBP` 5, `FloorBP` 6; `RateLimit` `ID` 1, `Region` 2, `Jurisdiction` 3, `CapBP` 4. `Label` is declared last in each and has no id. The slot levels and the suspensions are state and are not hashed. Whichever Stage 3 module merges second renumbers its own ids, never ORG's (merge-conflict forecast).
- **D2. Canonical order.** Bodies, rights, controls and rate limits are sorted by ID bytewise before a world is built or a hash is taken; `BodyID`, `RightID` and `ControlID` are the indexes (controls and rate limits sorted together, spec 6.1.2). The lists inside a body (`Roles`, `Regions`, `Jurisdictions`) are sorted. The sort is a method `(m *Institutions) canonical()` in `config.go`, called from `Config.canonical`, as the other modules do.
- **D3. State.** `type institutions struct` held as `w.org *institutions`, nil unless `Modules.Institutions` is present. Per-agent state is **not** a field of the agent type (architecture.md 2.10.1, N-244): `member []int32` and `memberRegion []int32` indexed by agent slot, in the module's own slices, saved in tag 3's tail (D5). Derived and rebuilt at construction and at a slot call, never saved: the resolved rights (a dense array by activity, subject and `RegionID`), the ranges in minor units by (`RegionID`, `CommodityID`), the scope regions of a body, the eligibility table by (role, region) to body, the scratch counts. Scratch is allocated once.
- **D4. Two positions, no draw.** `P1.9a` (the round) runs in phase 1 after the directives `P1.9`; `P7.2b` (the aggregate events) runs after `P7.2a`. `tick.go` gets two rows and two guarded calls and `notBuiltPositions` loses the two entries; the position test is staged both ways. Placement of `P1.9a` is inside `positionsFlows` or a new continuation of the straight line, chosen by the CCN measured in Phase 1 (the N-243 precedent that created `positionsFlows`). The round takes no word from the world's stream; the tie key is `ordering.Key(tick, agent)` (FR-ORG-017).
- **D5. The membership is one saved field, written only with the module.** Tag 3's tail is an `agentTails` entry with `present` (`Modules.Institutions != nil`), `save` and `load` method values: for every agent in ascending ID two varints, `BodyID + 1` and `RegionID + 1`, 0 for none (FR-ORG-049). The restore check of spec 12.3 runs after the agents are in; a membership of a household no longer eligible is legal. **Tag 31** is a section with `present`, `save` and `load` method values, written with the module and with zero rows when nothing differs from the configuration: the levels that differ (`ControlID`, `CapBP`, `FloorBP`), the suspended `RightID`s and the run of ticks of the `Warn` of spec 11.2, each in ascending order. `notBuiltSections` loses `{31, ...}`; `checkpoint_save.go` and `checkpoint_restore.go` are not touched. The format version does not change (tag 31 is in the table of spec-checkpoint.md 9.3). The all-modules-on fixture is **not** regenerated (supervisor, once at Stage 3 close).
- **D6. The clamp lives in `internal/clearing`, inside the admission, not before it.** A clamp that ran before `Admit` would turn an `invalid_price` order into an admitted one (spec 6.3.1 puts it after `overflow` and the settlement bounds, before `insufficient_funds`). `clearing.Facts` gets `Cap` and `Floor` (0 is none) and `NotPermitted` (false is permitted), appended in module order; a new `AdmitClamp(f Facts) (Code, price int64, lowered, raised bool)` holds the whole sequence and `Admit` becomes its wrapper that returns the code only, so every existing caller and test is unchanged (`NotPermitted` joins as the last constant of `Code`, its place in the order is in the function). The per-market range is held by the `book` (a derived pair set at construction and rewritten at a slot call), so the estate's ask and an outside counterparty's order are clamped with no change to `estates.go` or `external.go` (spec 6.3.3). The row `NotPermitted` is set only by the resolver of the facts for a household (`phase_trade.go:195`), so a pseudo-agent is exempt by construction (spec 5.5.2). The clamp count is added by one guarded call from `book.admit`.
- **D7. Reads, not positions.** Every read of a right or a range is a dense array read with no allocation: P2.5 (`admitJob`), P3.2(c) (the facts of an order), P3.3 and P3.3b (the contract and credit rounds), P3.4 and P3.5 (the clamp of the market), P7.1(a) steps (1) and (2) and P7.1(c) and (d) (the headroom). A right is read at a **start**, an **order** and a **place**; never at a continuation, a completion, a holding or a shipment (FR-ORG-018, ORG-D5). A suspended right leaves its key open and no outer row is consulted (FR-ORG-015).
- **D8. The credit seam takes the lowered limit back.** Today `rateCap(g)` returns the cap and CRD counts `priceControlled` in its own round counts, which `rd.reset()` clears before P7.2b. The seam becomes `rateCap func(g RegionID, limit int64) (replaced int64, lowered bool)` (nothing sets it today, so no result changes), ORG counts the lowering in its own scratch and reads it at P7.2b; CRD's `priceControlled` field is deleted or left as a plain count (Phase 7 decides after the audit; the CRD owner approves). Recommended (Question 9).
- **D9. The least of two limits.** A role's headroom is the least of its class limit (`Role.Ceiling`, SET) and its body's (spec 3.4.3). SET's `roleHeadroom` and `fillRoleGates` stay SET's; ORG adds `w.orgHeadroom(g, r)` in `institutions.go`, which takes SET's value when the settlements module is on and the body's when the role is in a body, and the slot becomes `skills.headroom = w.orgHeadroom` when both modules are on. `fillRoleBodies(g)` is one guarded call in `fillRoles` beside `fillRoleGates`, lowering `Headroom` and clearing `NoLimit` where a body limit applies. A role with no body is untouched; `Places` 0 gives no figure. The count of households of the body's roles is the scan of SET4 (no counter), read once per (tick, region) at the first read of the entry step and kept exact between reads by SET's `noteRoleMove` and the entry step's own admissions; ORG reuses SET's scratch and adds no hook to the lifecycle fan-ins.
- **D10. EMP's formation.** Spec 5.3 and the row "EMP" of spec 8 want a hire checked as a request (principal, executor by `Who`, a closed shop). EMP's hire form is not built (`Employer.Recipes` is in `notBuilt`; `HireFormed` is in `notBuiltEvents`). ORG reads the right at P2.5 for the principal of a `Work` request only; the executor test of a hire is added in the change that builds the hire form, by one guarded call into `institutions_rights.go`. Phase 1 adds the note and the row. Recommended (Question 5).
- **D11. Events.** Five: `MembershipChanged`, `RightRefused` and `PriceControlled` (aggregates at P7.2b in the order of spec 7.3.3), `LegalLevelSet` and `RightSuspensionSet` (at a slot call, one per changed key, carrying the tick of the next step). Plain data, no map, constructed under `w.obs.supplied` only; the counts are kept in scratch only under the same flag. `MembershipChanged.Held` is the held completions of P7.1(a) of the tick (SKL notifies ORG by one guarded call, H-13). The five rows are in the catalogue of observability 4.3 and `notBuiltEvents` loses five entries. No event gains a field; `EntryLimited.RoleCeiling` and `PopulationChanged.BirthsCeiling` already carry a body's limit.
- **D12. Documentation.** `doc.go` glossary gets the ORG terms before any code uses them (architecture.md 3.2): body, member, outsider, place, headroom (of a body), seniority, eligible, right, activity, subject, scope key, prohibition, in force, suspended, principal and executor (already PRO's), control, rate limit, legal range, cap, floor, clamp, legal price, membership round, held completion, need. `TestGlossary_DeclaredTypesNamed` reads `doc.go`, so the new type names are added in the same phase. "Floor" and "cap" stay qualified (the settlement floor, the region cap).

## Requirement list and where each is verified

"New" means built by this plan. Nothing in the tree cites a FR-ORG ID yet, so every row is new; the SKL slot, the CRD seam and the MER rows are re-checked, not owned.

| ID | Phase | Lowest-level test |
| --- | --- | --- |
| FR-ORG-001 (off or empty leaves no trace) | 4, 10 | `TestGolden_InstitutionsOffLeavesNoTrace`, `TestEmptyModule_EqualsOff` (the module present with nothing declared; seeds); ORG-S5 asserts the inert overlay (Question 4) |
| FR-ORG-002 (on iff non-nil; `ErrModuleDependency`) | 3 | `TestValidate_Institutions_JurisdictionNeedsPolity`, `_NeedsModuleOff`, `_RateLimitNeedsCredit` |
| FR-ORG-003 (sorted, `BodyID` index) | 2, 3 | `TestCanonical_BodiesSortedByID`, `TestBodyID_IsIndex` |
| FR-ORG-004 (one body per (role, region)) | 2, 3 | `TestOverlap_Detected` (unit), `TestValidate_Institutions_BodyOverlap` |
| FR-ORG-005 (eligible) | 2, 4 | `TestEligible_RoleAndRegionOnly`, `TestEligible_PseudoAgentNever`, `TestEligible_ExternalAgent` |
| FR-ORG-006 (places count households; never removed; members never above places) | 2, 4 | `TestAdmit_NeverAbovePlaces` (the model `admit.py` A1 as an exhaustive and a seeded unit test), `TestRound_NobodyRemovedForPlaces` |
| FR-ORG-007 (headroom, least of two) | 2, 9 | `TestHeadroom_LeastOfTwo`, `TestOrgHeadroom_WithAndWithoutClass` |
| FR-ORG-008 (`Needs`: bars, counts, removes none) | 4 | `TestRound_NeedsFlagMissingBarsAndCounts`, `TestRound_NeedsKeepsMembers` |
| FR-ORG-009 (no `region_bound` refusal) | 4 | `TestRegionBound_NoneRegistered` (the list of obligations is unchanged) |
| FR-ORG-010 (membership test is three comparisons) | 4 | `TestMember_StaleFieldIsNotMembership` (a role change and a move between rounds) |
| FR-ORG-011 (lapse, then admit by `RoleSince`, key, ID) | 2, 4 | `TestRound_OrderRoleSinceKeyID`, `TestLapse_RoleAndRegion` |
| FR-ORG-012 (whole cohort, skip and go on) | 2 | `TestAdmit_FirstFitSkipsCounted` |
| FR-ORG-013 (round at P1.9a only; no round at construction) | 4 | `TestPositions_P19aListed`, `TestRound_NoneAtConstruction`, `TestRound_JobInProgressRuns` |
| FR-ORG-014 (right in force: region, else first of chain, else none; resolved once) | 2, 4 | `TestResolve_InnermostChain`, `TestRights_ResolvedAtConstruction` |
| FR-ORG-015 (suspended leaves the key open) | 2, 8 | `TestResolve_SuspendedKeyOpen`, `TestSlot_SuspensionRebuildsTable` |
| FR-ORG-016 (prohibition) | 6 | `TestRight_ProhibitionBarsAllButPseudoAgents` |
| FR-ORG-017 (no draw) | 4, 10 | `TestDeterminism_InstitutionsNoDraw` (stream position equal with the module inert), `Guard:` text search for `rnd.` in `institutions*.go` |
| FR-ORG-018 (read at a start, an order and a place only) | 6 | `TestRight_NotReadAtContinuationCompletionHolding` |
| FR-ORG-019 (idempotent; order-independent) | 2, 4 | `TestRound_SecondRoundChangesNothing`, `TestAdmit_OrderOfListIrrelevant`, `FuzzAdmit` |
| FR-ORG-020 (start refused after `not_on_role`, before `k0`; `Who`) | 6 | `TestStart_NotPermittedAfterNotOnRoleBeforeK0`, `TestStart_WhoPrincipalExecutorBoth`, `TestStart_NothingConsumed` |
| FR-ORG-021 (trainee exempt from `executor`) | 6 | `TestStart_TraineeExemptFromExecutor` |
| FR-ORG-022 (own use exempt unless `OwnUseBound`) | 6 | `TestStart_OwnUseExemptUnlessBound` |
| FR-ORG-023 (pseudo-agents never refused) | 6 | `TestRight_EstateAndCounterpartyExempt` |
| FR-ORG-024 (`sell` and `buy`: reason, place, no outcome) | 6 | `TestOrder_NotPermittedAfterNotTradable`, `TestOrder_RefusedHasNoOutcome` |
| FR-ORG-025 (`train` right: trainer a member; applicant count) | 9 | `TestTrain_NonMemberHasNoPlace`, `TestTrain_TraineeshipInProgressKept`, `TestTrain_TrainersCountEligibleOnly` |
| FR-ORG-026 (`min` of role and body `MaxTrainees`) | 9 | `TestTrain_BodyMaxTraineesLeast` |
| FR-ORG-027 (held completion, queue order) | 9 | `TestCompletion_HeldWithoutHeadroom`, `TestCompletion_FirstEntryStepWithHeadroom` (the `admit.py` A5 case) |
| FR-ORG-028 (range: ref times bp, truncated, bounded, checked) | 2 | `TestRange_TruncatedBoundedChecked` (the P5 grid of `clamp.py`: `MaxPrice` times `MaxBasisPoints` fits), `FuzzRange` |
| FR-ORG-029 (`WorldCreated` unchanged; no money or agent) | 4, 5 | `TestWorldCreated_NamesTableOnly`, `TestLedger_NoOrgReason` |
| FR-ORG-030 (clamp after `overflow`, before funds, whoever places) | 2, 7 | `TestAdmitClamp_OrderOfChecks`, `TestClamp_ReservationAndOutcomeCarryReplacedLimit`, `TestClamp_EstateAndCounterpartyBound` |
| FR-ORG-031 (every trade inside the range) | 7 | `TestClamp_EveryTradeInsideRange` (the 20 000-book model of `clamp.py` as a seeded loop), the invariant row of `TestInvariants` |
| FR-ORG-032 (no quantity cut, reject, reorder or draw) | 7 | `TestClamp_NoQuantityNoRejectNoDraw`, example B of 13.2 as `ExampleClamp` |
| FR-ORG-033 (restore equals uninterrupted) | 5, 8 | `TestCheckpoint_InstitutionsRoundTrip`, `TestFork_Institutions` (ORG-S2 style, N in a hold, at a lapse, after a slot call) |
| FR-ORG-034 (credit clamp; a claim in force unchanged) | 7 | `TestRateLimit_RequestLimitReplaced`, `TestRateLimit_ClaimInForceUnchanged` (example C: 87 600 000 / 3 650 000 = 24) |
| FR-ORG-035 (contract clamp and the destination's rights) | 7 | `TestContract_TakeCappedOfferFloored`, `TestContract_DestinationRightsBind` |
| FR-ORG-036 (`SetLegalLevels`) | 8 | `TestSetLegalLevels_ErrorOrder`, `_AtomicAndRebuilds`, `_EntryEqualToConfiguredRestores` |
| FR-ORG-037 (`SetRightSuspensions`) | 8 | `TestSetRightSuspensions_ErrorOrderAtomic` |
| FR-ORG-038 (slot state saved sparse, not hashed) | 5, 8 | `TestSlots_LegalLevelsSavedNotHashed`, `TestHash_InstitutionsSlotStateExcluded` |
| FR-ORG-039 to 041 (the three aggregate events) | 5, 6, 7 | `TestEvent_MembershipChangedFieldsAndGate`, `TestEvent_RightRefusedCounts`, `TestEvent_PriceControlledCounts` |
| FR-ORG-042 (slot events at the call) | 8 | `TestEvent_LegalLevelSetCarriesNextTick`, `TestEvent_RightSuspensionSetOnePerKey` |
| FR-ORG-043 (order at P7.2b) | 5 | `TestEvents_P72bAfterSettlementBeforePhase8` |
| FR-ORG-044 (names table, snapshot rows) | 5, 8 | `TestSnapshot_BodiesLegalLevelsSuspended`, `TestNames_BodiesRightsControls` |
| FR-ORG-045 (the `Warn` by pattern B) | 5 | `TestLogs_BodyWaitingWarn` (tenth tick and every tenth), counterpart `MembershipChanged.Waiting` |
| FR-ORG-046 (construction summary) | 5 | `TestLogs_InstitutionsConstructionLine` |
| FR-ORG-047, 048 (rules and warnings) | 3 | `TestValidate_Institutions_<Rule>` one failing case per row of 10.1, `TestWarnings_Institutions` (the eleven rules of 10.2), `FuzzValidateInstitutions` |
| FR-ORG-049, 050 (agent field and tag 31; the restore check) | 5 | `TestCheckpoint_MembershipFieldPerAgent`, `TestCheckpoint_InstitutionsRefusesCorrupt` (a half-zero pair, an undefined body, a region outside the scope, IDs out of order; an ineligible member accepted), `FuzzDecodeInstitutions` |
| FR-ORG-051 (hash covers all but labels) | 3 | `TestHash_InstitutionsOnOff`, `_FieldIDs`, `_OrderIrrelevant`, `_LabelNotHashed`, `TestLoad_InstitutionsToml`, `FuzzLoad` corpus extended |
| FR-ORG-052 (no ledger reason) | 4 | `TestLedger_NoOrgReason` |
| FR-ORG-053 (no map iteration, total order) | 2, 4 | `TestDeterminism_Institutions` over `detCases(true)`, split `Run` calls, with and without observer |
| FR-ORG-054 (no aggregate read) | 6, 7 | `Guard:` text search over `institutions*.go` for price, stock, income and ratio reads; the one count read is the households against `Places` |
| FR-ORG-055 (metrics derivable) | 5 | `TestEvents_MetricsDerivable` (a recording consumer folds the events and the snapshot to the four instruments of 11.6) |
| NFR-ORG-001 (no allocation; no event without an observer) | 5, 7 | `TestAlloc_InstitutionsNoOp` (`AllocsPerRun`: right test, range read, clamp, headroom, a round), `TestStep_AllocationsStayAtTheBaseline` rows unchanged with the module off |
| NFR-ORG-002 (costs) | 4 | `TestInstitutions_WorkIsCounted` (agents visited per round), `BenchmarkP19a` (reports) |
| NFR-ORG-003, 004 (no influence; bit-identical) | 5, 10 | `TestDeterminism_InstitutionsNonInterference`, `TestDeterminism_Institutions` |

Also touched and re-checked, not owned: FR-SKL held completion (N-193) and FR-SKL-015, 019, 024; FR-CLR-016 and the reason table of CLR 3.2; FR-PRO-008 and 048; FR-MER row 5a and 9; FR-CRD-028; FR-SET-023 and 019 (ORG is a second reader of the count); FR-INT slots (FR-INT-032); FR-CHK-041 and 045; FR-WLD-040 (two positions); FR-OBS rows and N-045 (every refusal counted).

## Phases

Each phase ends green on `go test -short -race ./...` and on the tests it adds. `testdata/golden/off/`, `TestHashConfig_ReferencePinned`, `testdata/checkpoints/` digests and every existing golden stay **unchanged at every phase exit** (a test run at each exit; a failure is a defect, never a golden to regenerate). The PR split groups the phases.

### Phase 1: the owed edits, the audit and the glossary (no code) (PR A)

Goal: the specs, rules and notes are true before the code starts. Rule files that change get a fresh tag. This phase edits specs, so the worker reads **all** of `doc/notes.md`, quotes `Notes tag:` first, and gives each open note tagged ORG or ALL a verdict. Edits owed:

| # | File and section | Edit |
| --- | --- | --- |
| E1 | architecture.md 2.6, 3.3, AGENTS.md Layout | add the continuation files of D12's list (`institutions_validate.go`, `institutions_round.go`, `institutions_rights.go`, `institutions_legal.go`, `institutions_report.go`, each mapped to ORG) to both lists with the same files, and `BodyID`, `RightID`, `ControlID` among the typed IDs of 3.3; fresh RULE-TAG. **Separately:** add the row `internal/institution 90` to `scripts/ci/coverage-thresholds.txt`. A continuation file is listed only if Lizard or the 500-line limit forces it on the draft (decide by `lizard` as plan 3-sto D3) |
| E2 | spec-config.md 8.2.2 | the inner ids of D1 and hash id 21 |
| E3 | spec-config.md 3.3, 3.4 | the sort of bodies, rights, controls and rate limits by ID and of `Roles`, `Regions`, `Jurisdictions` |
| E4 | spec-institutions.md 1, 11.4, 16, 19.2, 7.3.2 | status line; section 19.2 lists the continuation files; the agent tail and tag 31 named as `agentTails` and a row of `checkpointSections` (no `checkpoint.go`); the `regionHouseholds`-style count named once (the SET scan, D9); section 16 says N-090's list gains ORG |
| E5 | spec-institutions.md 14.3 (ORG-S5), 17 (FR-ORG-001) | **a contradiction to settle (Question 4):** S141 says the inert overlay (bodies declared, `Places` 0) is identical to the world without the module except four things, but with a declared body every region of its scope emits `MembershipChanged` (the first round admits every eligible household) and `Snapshot.Bodies` has rows. The assertion becomes "identical but the module's own events, snapshot rows, names, hash, `Snapshot.Modules` and construction line" as SET-S4 reads it; FR-ORG-001's strict form is tested on the module present and empty |
| E6 | spec-clearing.md 3.2, 3.2.1; spec-agents.md 6.5.2 | the shape of D6 (`AdmitClamp`, `Facts.Cap`, `Floor`, `NotPermitted`), the place of `not_permitted` among the codes, the count of a clamp; no behaviour edit |
| E7 | spec-credit.md 5.2.3 | the seam takes the limit and returns the replaced one (D8) |
| E8 | spec-merchants.md 9.3; spec-skill-barriers.md 4.4, 4.5, 4.9, 3, 8.3; spec-settlements.md 6.3, 6.4.2, 7; spec-world.md 6, 6.11, 8.2, 9.2, 9.4.1, 10.2; spec-stability.md 5.1.3; ADR 0001 decision 7 | the rows owed by spec 8 that are not yet written (read each; edit only where the text is missing; the second reader of the count of households is named beside SET's in the four texts of SET 7.4) |
| E9 | spec-integration.md 3.4.1, 4.7.2, 5.1, 5.3, 11.2 | the two setters, the sentinels, `SlotState` fields, the membership group of `AgentState` as `{Member bool; Body BodyID}` (Question 7) |
| E10 | spec-reference-chain.md 4.5, 7.12 | confirm the two overlays and S137 to S141; the overlay names the one skilled role through data (`training_ticks` is core data, spec-config.md 3.7.2) |
| E11 | roadmap-v1.md Stage 3 row | `institutions.go`, `internal/institution`; status `plan written`, then `PR A in development`; the continuation files if E1 lands |
| E12 | notes | verdicts of Risks, below; N-140, N-155, N-160, N-161, N-205 and N-244 get an update line, none is removed before its reader exists; **new notes (numbers N-397 to N-404, the range the supervisor allocated to ORG; no other number is taken):** the EMP formation hook (D10), the second seam of CRD (D8), the S141 contradiction (E5) |
| E13 | `doc.go` glossary | D12, before any code uses a term; the type names |

Audit (read-only, recorded in the result paragraph): whether SKL keeps a `Held` count for a held completion (`completeReady`, `completionHeadroom`) and where ORG's one notification goes; the Lizard figures (`lizard -l go -C 1 -L 1`) of `positionsFlows`, `positionsTail`, `admitJob`, `book.admit`, `Admit`, `fillRoles`, `admitCreditRequests`, `admitOffers`, `admitTakes`, `takeRefusal`, `trainersOf`, `fillPlaces`, `completeReady`, `Slots`, `Agent`, `snapshot`'s region filler, `Validate`, `warnings` (85 today, above the limit; the module adds one line `institutionsWarnings` beside the others, the SET and STO precedent), `pairsLastModules`, `checkpointCross`; the lines of `contract.go`, `skills_queue.go`, `clearing.go`, `snapshot.go` and `event.go` against the 500-line limit (a file already over it needs no new code of ORG); which of the readers of spec 8 exist; whether the world builds the orders in one place (`phase_trade.go:195`) and whether `checkTradePlan` also needs the right (spec 5.2 says the world checks "while it builds the orders"); which pack carries a skilled role in a region with a group (Phase 10). The result paragraph lists, for the supervisor, the hooks of "Hook discipline" with the approval column.

Exit check: E1 to E13 applied or logged with an owner; `npx markdownlint-cli2` clean on every changed `.md`; `go test ./...` and `testdata/golden/` unchanged.

### Phase 2: `internal/institution` (pure) (PR A)

Goal: every number of spec 3 to 6 as a function over primitives, no world. Files: `internal/institution/{doc,resolve,rangeprice,clamp,admit,headroom}.go` and their tests, and the coverage row if not added in Phase 1.

- `Resolve(table []Row, chain [][]int, ...)`: the innermost mode, the row keyed by the region, else the first of its chain, else none, with a suspended row leaving the key open (FR-ORG-014, 015). The overlap check of 3.3.2 (`Overlap`) over the (role, region) sets.
- `Range(ref, capBP, floorBP, maxPrice int64) (cap, floor int64, ok bool)`: `min(MaxPrice, max(1, ref * bp / 10 000))` with `internal/checked`, truncating (FR-ORG-028).
- `Clamp(limit int64, bid bool, cap, floor int64) (admitted int64, changed bool)` (FR-ORG-030, 032).
- `Lapse(...)` and `Admit(...)` over parallel slices (eligible, weight, `RoleSince`, key, member region): lapse first, then candidates by `RoleSince`, `Key`, ID, first fit, counts of admitted, skipped, barred, waiting (FR-ORG-006, 011, 012, 019). The key is passed in; the root takes it from `internal/ordering`.
- `Headroom(places, households int64) (int64, noLimit bool)` and `Least(a, b)` (FR-ORG-007).
- Tests, table-driven, got/want, with the arithmetic of the spec in the comment: `TestResolve_*`, `TestRange_*` (the `clamp.py` P5 grid, `MaxPrice` times `MaxBasisPoints` fits), `TestClamp_*` (example B of 13.2: cap 1 200, six bids and five asks), `TestAdmit_*` (example A of 13.1: 14 households, 12 places: 12 admitted, 2 skipped; tick 4: one admitted, one waiting; the **exhaustive** small-population loop of `admit.py` A1 to A3, not a sample), `TestLapse_*`, `TestHeadroom_*`, `TestPackage_NoRootNoName`, `TestPackage_NoFloat`, `FuzzAdmit`, `FuzzRange`, `FuzzResolve`.
- Exit check: `go test -race ./internal/institution/...`; coverage at least 90%; `gomu` per new file in Docker (`--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS=4`; survivors killed or recorded equivalent); `lizard -l go -C 10 -a 8 -L 100 -w` clean.

### Phase 3: the configuration surface (PR A)

Goal: the module exists as data and is checked; at run time it does nothing. Files: `config.go` (`Institutions`, `Body`, `Right`, `Control`, `RateLimit`, `Modules.Institutions` in Go order, `canonical`), `value.go` (`BodyID`, `NoBody`, `RightID`, `ControlID`), `bounds.go` (`MaxBodies` 256, `MaxRights` 4 096, `MaxControls` 4 096, each with its check row), `defaults.go` (`DefaultInstitutions`: empty, `OwnUseBound` false), `errors.go` (the seven sentinels), `hash.go` (`pairsInstitutions` at id 21), `validate.go` (the call of `validateInstitutions` at the module seam in the order of the `Modules` row and one line `institutionsWarnings(c)` beside `storageWarnings`), `institutions_validate.go` (the rules of spec 10.1 in the order the spec gives, the eleven warnings of 10.2; `places_below_start` and `body_needs_unmet_at_start` use SET's census of the groups, `right_over_upkeep` and `cap_below_seed_cost` use the effective upkeep and the warm start), `config/mapping.go` (`[modules.institutions]`, `.body.<id>`, `.right.<id>`, `.control.<id>`, `.rate_limit.<id>`), `registry_test.go` (the rows leave `notBuilt`, both directions), `internal/logkeys` (the nine keys of observability 3.3.2: `body`, `right`, `control`, `activity`, `places`, `waiting`, `bodies`, `rights`, `controls`).

- Tests: `TestValidate_Institutions_<Rule>` one failing case per row of spec 10.1 asserting the sentinel with `errors.Is` and the path (`modules.institutions.rights[1].subject`), `TestWarnings_Institutions` (the eleven), `TestCanonical_*`, `TestHash_*` of FR-ORG-051, `TestLoad_Institutions*`, `TestSentinels_*` for the seven, `FuzzValidateInstitutions`, `FuzzLoad` corpus extended. The registry rows of the five scopes and `Modules.Institutions` leave `notBuilt`; `Modules.Institutions` joins `exemptFields` as the other pointers do.
- Exit check: `TestHashConfig_ReferencePinned` and `TestGolden_OneOff` unchanged; the registry test staged both ways; `config/` coverage at least 90%; `go test -short -race ./...`.

**First pull request (PR A) ends here with Phases 1 to 3.** It adds no behaviour (a configuration with the module on builds a world that ignores it) and lands the catalogue appends of the config files early, where Stage 3 modules collide.

### Phase 4: state, construction, the resolved tables, the round at P1.9a (PR B)

Goal: members exist, lapse and are admitted; no right or price reads them yet. **First commit (refactor-first, before the hooks):** the CCN-driven splits Phase 1 recorded (the straight line of phase 1 if `positionsFlows` is at the limit). Files: `institutions.go` (state, `newInstitutions`, the eligibility and scope tables, the resolved rights and ranges built from the configuration and the chain, `member`), `institutions_round.go` (the round: for each body in ascending `BodyID` the lapse pass over every household, then per region of the scope in ascending `RegionID` the admission; the scratch counts), `world.go` (`org *institutions`), `construct.go` (the call after the agents are created and before the first step, the `modulesOn` row `"institutions"`, the construction line keys, no round at construction), `tick.go` (D4), `internal/logkeys`.

- Tests: `TestConstruction_TablesResolved`, `TestRound_*` (example A: tick 0 and tick 4), `TestRound_JobInProgressRuns`, `TestRound_NeedsFlagMissingBarsAndCounts` (with SET on), `TestRound_NobodyRemovedForPlaces`, `TestMember_StaleFieldIsNotMembership`, `TestRound_SecondRoundChangesNothing`, `TestEligible_*`, `TestPositions_P19aListed` (`notBuiltPositions` loses `"P1.9a"`), `TestRegionBound_NoneRegistered`, `TestEmptyModule_EqualsOff` (FR-ORG-001, seeds, race-skipped with a prefix), `TestDeterminism_InstitutionsNoDraw`, `TestInstitutions_WorkIsCounted`. **Race skips (testing.md 9.1):** the multi-seed tests skip under `raceEnabled` and carry a `gateFPrefixes` prefix (`scenario_helpers_test.go:45`); the unit and one-world tests never skip.
- Exit check: `go test -short -race ./...`; the position test staged both ways; `TestStep_AllocationsStayAtTheBaseline` rows with the module off unchanged; `golden/off/` unchanged.

### Phase 5: the event at P7.2b, snapshot, names, logs, checkpoint, fork (PR B)

Goal: the module is observable and restorable. Files: `institutions_report.go` (the `MembershipChanged` event at P7.2b, in (`RegionID`, `BodyID`) order and after `SettlementClassChanged`; the `Warn` of pattern B; the snapshot filler; the agent tail and tag 31 encoder and decoder; the restore check), `event.go` (the five event types, declared in module order (only `MembershipChanged` is emitted in this phase; `RightRefused` from Phase 6, `PriceControlled` from Phase 7, the two slot events from Phase 8); `notBuiltEvents` loses the five), `snapshot.go` (`Bodies`, `LegalLevels`, `Suspended`, `Names.Bodies`, `Rights`, `Controls`), `checkpoint_sections.go` (row 31 with `present`, `save`, `load` method values; an `agentTails` entry; entry deleted from `notBuiltSections`), `observe_*` and `cardinality_test.go` rows, and in `otelobserver/` the three lists of the STO pattern (`adapter_test.go` `ignoredEvents`, `events_test.go` `allEvents`, `testdata/all_events.txt`; the adapter code is untouched). `allocation_test.go`.

- Tests: `TestEvent_MembershipChangedFieldsAndGate`, `TestEvents_P72bAfterSettlementBeforePhase8`, `TestSnapshot_*`, `TestNames_*`, `TestLogs_InstitutionsConstructionLine`, `TestLogs_BodyWaitingWarn`, `TestCheckpoint_InstitutionsRoundTrip` (a world with the module off writes **no** section and no tail: the bytes of every existing fixture are unchanged), `TestCheckpoint_MembershipFieldPerAgent`, `TestCheckpoint_InstitutionsRefusesCorrupt`, `TestFork_Institutions` (through `forkAtN` and `forkCase`: N in {0, after a first round, at a lapse, after a call}), `FuzzDecodeInstitutions` (no panic, bounded allocation), `TestAlloc_InstitutionsNoOp`, `TestDeterminism_InstitutionsNonInterference`, `TestEvents_MetricsDerivable`.
- Exit check: `testdata/checkpoints/` untouched; the catalogue, cardinality and otelobserver list tests (run in `otelobserver/`) pass with the entries deleted; `golden/off/` event files unchanged.

**Second pull request (PR B) holds Phases 4 and 5.**

### Phase 6: the rights: produce, sell, buy (PR C)

Goal: a right refuses a start and an order. Files: `institutions_rights.go` (`startRefusal`, `orderPermitted`, the `Who` and trainee and own-use rules, the counts), `phase_production.go` (H-4: one guarded call in `admitJob` after `admit`), `phase_trade.go` and `clearing.go` (H-5: the resolver of the facts sets `NotPermitted` for a household; `codeReasons` gains the row), `internal/clearing/admit.go` (`Facts.NotPermitted`, the code `NotPermitted`, its position after `NotTradable` in `AdmitClamp`, with `Admit` unchanged for callers), `contract.go` (H-7: rows 5a of an offer and a take, the destination region's `sell` and `buy` rights), `subsistence.go` (H-9: `not_permitted` is a reason of `Blocked` only with `OwnUseBound`), `event.go` (`ReasonNotPermitted`, the `RightRefused` emission at P7.2b), `internal/logkeys` (the reason value).

- Tests: the FR-ORG-016, 018, 020 to 024, 035 (the rights half), 040 rows above; `TestStart_NotPermittedAfterNotOnRoleBeforeK0`, `TestOrder_NotPermittedAfterNotTradable`, `TestRight_EstateAndCounterpartyExempt`, `TestContract_DestinationRightsBind`, `TestSubsistence_OwnUseBoundBlocks`, `TestClearing_AdmitUnchangedWithoutOrgFacts` (every existing `Admit` table case still passes), `TestReaders_StaticCheck` (a text search over the packages that read a right: only the sites of the hook list).
- Exit check: `go test -short -race ./...`; every existing PRO, CLR, MER and SUB golden unchanged (no pack has a right).

### Phase 7: the legal prices: market, contract, credit (PR C)

Goal: a control clamps the limit of the side it bounds. Files: `institutions_legal.go` (the ranges, the per-market pair, the clamp counts and the `PriceControlled` emission), `internal/clearing/admit.go` (`Facts.Cap`, `Floor`; `AdmitClamp` returns the replaced price and the two flags), `clearing.go` (H-6: `book` holds the pair; `book.admit` uses the replaced price for the reservation, the entry and the placed event, and calls the one guarded count), `contract.go` (H-7: the clamp of a take and an offer after rows 9 and 6), `credit.go` and `credit_round.go` (H-8: D8, one seam change and the call), `event.go` (`PriceControlled`), `doc.go`.

- Tests: `TestAdmitClamp_OrderOfChecks` (an `invalid_price` order is not rescued by a cap; a cap after `overflow`, before `insufficient_funds`), `TestClamp_*` (the 20 000-book model of `clamp.py` as a seeded loop: every trade inside the range, volume at least that of a rejection), `TestClamp_ReservationAndOutcomeCarryReplacedLimit`, `TestClamp_EstateAndCounterpartyBound`, `TestRateLimit_*` (example C), `TestContract_TakeCappedOfferFloored`, `TestEvent_PriceControlledCounts`, `ExampleClamp` (example B), `TestMarketCleared_ShortageSignal` (unmatched bid units at a binding cap, with the published price above the cap and `Age` counting, spec 6.5.3).
- Exit check: `TestStep_AllocationsStayAtTheBaseline` and the clearing allocation rows at baseline with the module off; the existing clearing, merchants and credit goldens unchanged.

**Third pull request (PR C) holds Phases 6 and 7.**

### Phase 8: the slots and the agent's membership group (PR D)

Goal: a consumer sets a legal level and a suspension between ticks. Files: `institutions_legal.go` (`SetLegalLevels`, `SetRightSuspensions`, the error order of spec 7.1.1, the atomic rebuild at the call, the sparse state), `integration_slots.go` (H-10: `SlotState.LegalLevels` and `RightSuspensions` appended; one guarded append in `Slots`), `agent.go` (H-11: `AgentState.Membership`, filled by one guarded call in `Agent`), `errors.go` (already in PR A), `event.go` (`LegalLevelSet`, `RightSuspensionSet`), `institutions_report.go` (tag 31 gets its content), `doc.go`.

- Tests: `TestSetLegalLevels_*`, `TestSetRightSuspensions_*` (each error of 7.1.1 in order, `ErrModuleOff` first, atomic: a refused call changes nothing), `TestSlot_SuspensionRebuildsTable`, `TestEvent_LegalLevelSetCarriesNextTick`, `TestSlots_LegalLevelsSavedNotHashed`, `TestCheckpoint_SlotStateRoundTrip` (a changed level and a suspended right survive a restore and the next round is the uninterrupted one), `TestAgentState_Membership`, `TestSlots_BetweenTicksOnly` (`ErrInStep`).
- Exit check: `Slots()` of a world with no slot set is empty; the integration goldens unchanged.

### Phase 9: SKL's gates and the shared headroom (PR D)

Goal: a body bounds entry and gates the traineeship. Files: `institutions_rights.go` (the `train` read, the trainer test, the body's `MaxTrainees`), `institutions.go` (`orgHeadroom`, `fillRoleBodies`), `skills_queue.go` (H-12: `trainersOf` and `fillPlaces` read the member test and the least maximum through two guarded calls; H-13: `completeReady` calls `w.org.noteHeld(g, r)` when it holds a completion), `skills_report.go` and `internal/skill` (the applicant's `trainers` and the expected wait count only trainers that can take a trainee; the `Debug` line with reason `not_member`), `phase_exit.go` (H-14: `fillRoleBodies` beside `fillRoleGates`), `construct.go` (`skills.headroom = w.orgHeadroom` when the module is on), `institutions_report.go` (`Held` in `MembershipChanged`, the "place not open" `Debug` line), `event.go` (the `RightRefused.Train` count).

- Tests: `TestTrain_*` (FR-ORG-025, 026), `TestCompletion_HeldWithoutHeadroom`, `TestCompletion_FirstEntryStepWithHeadroom`, `TestCompletion_HeldThenTakenBeforeEntrant` (the `admit.py` A5 ordering), `TestRight_TrainCountIsMinFA` (spec 5.6.3: `min(f, a)`, one count a place), `TestOrgHeadroom_LeastOfTwo` (both modules on, a class limit of 5 and a body's of 3), `TestRoleView_BodyHeadroom`, `TestEntryLimited_RoleCeilingIncludesBody`, `TestPopulation_BirthsCappedByBody`, `TestRegion_NoBodyRoleUntouched` (the existing SET and SKL goldens unchanged), `TestPremium_CheckDReportedNotAsserted` (spec 5.7.5).
- Exit check: the existing SKL, SET, spawner and population goldens unchanged; `TestStep_AllocationsStayAtTheBaseline` and `TestDemand_AllocationsStayAtTheBaseline` at baseline for the module off.

**Fourth pull request (PR D) holds Phases 8 and 9.**

### Phase 10: overlay, golden, scenarios (PR E)

Goal: the module is proved end to end. Files: `testdata/chains/overlay/institutions.toml` and `institutions-inert.toml` (spec-reference-chain.md 4.5), `testdata/golden/institutions/{<pack>.txt,events-<pack>.txt,<pack>.positions.txt}`, `scenario_org_test.go`, a `TestDeterminism_Institutions` case, a `TestInvariants` row.

- **Pack that completes the feature.** Recommended (Question 2): the module-alone golden on pack `five` with `overlay/institutions.toml` (spec 13.5, scenario S138): a body over one skilled role of the hub region with places at 80% of its starting households, a `produce` right and a `train` right, and a control keyed by a jurisdiction (the overlay carries the jurisdictions of `overlay/polity.toml` it needs, or says which layer to add; Phase 1 audit). A **probe test** over seeds 1 to 4 asserts the golden records at least one `MembershipChanged` with `Waiting`, one `RightRefused` and one `PriceControlled`, and fails otherwise, so the golden exercises the module; the held completion needs SKL on and is asserted on a layered world (`five` + the skill overlay), as plan 3-set layers for `grew`. If the probe finds no held completion within the window the scenario falls back to a trainee scripted between ticks and the unit test carries FR-ORG-027.
- Overlay numbers are placeholders and say so in the file header (N-015, N-027).
- Scenarios: **ORG-S1 = S137** (`one` or `three` with a cap on a staple at 80%, 100% and 120% of the solver's price, lifted at tick T, and with the trade overlay for contracts: every trade price at or below the cap, conservation, `WithAudit`; the unmatched bid units, the volume, the price after the lift and the recovery time are reported); **ORG-S2 = S138** (the body overlay over a long run: held completions bounded by the trainers' places, no negative headroom admitted; the observed premium is reported against the derived one); **ORG-S3 = S139** (a start above the places with migration on: `Waiting` falls or is bounded; nobody removed for exceeding the places); **ORG-S4 = S140** (`one` + `overlay/own-use.toml` with a prohibition of a staple's sale and `OwnUseBound` false and true: the exempt household feeds itself); **ORG-S5 = S141** (the inert overlay against `five`, seeds 1 to 10: Question 4). S5 is a harness test outside the band rule (spec 13). Every multi-seed scenario skips under `raceEnabled` and carries a `gateFPrefixes` prefix (`TestScenarioORG1_*` to `TestScenarioORG5_*`); the checkpoint and fork tests already fit one.
- `golden/institutions/` is recorded once after the diff against `golden/off` is read; it is **never regenerated** to absorb a failure. `golden/off/`, `TestHashConfig_ReferencePinned`, `testdata/checkpoints/` are untouched. `golden/all/`, `all_pack_test.go` `allLayers` and the all-modules checkpoint fixture are **not touched by this change**: `Modules.Institutions` joins `allPackNotYetOn`, and the supervisor moves the all layer once at Stage 3 close.
- Exit check: S137 to S141 run in the quick pattern (their names carry the prefixes), so arm64 covers them; `TestInvariants` with `WithAudit` over `scenarioSeedCount()` seeds.

### Phase 11: the exit gate (PR E)

Gates run by hand on the dev host (testing.md 9.1); the exit markers go in the PR description. Per the gate economics below: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`, `go test -short -race ./...`, gate R, gate F with `scripts/ci/coverage-check.sh` (every scope at its minimum, coverage not lower), `go mod tidy` no diff, in the root, in `otelobserver/` and in each `examples/` module (the public additions are types, two calls, a snapshot field, `SlotState` fields and `AgentState.Membership`; no example change is expected). A deliberate-break pass (testing.md 9.3: flip the clamp's direction, drop the lapse by region, admit in ID order, read a right at a continuation, drop the held completion, put the clamp before `invalid_price`) shows the tests fail. `gomu` per new file in Docker with the caps. Arm64: **once, before Stage 3 closes**, not per PR. Close: roadmap row `merged (PR #...)`, notes settled or updated, `spec-roadmap.md` and `documentation_index.md` updated, plan audited with plan-audit.

## Pull request split

Five pull requests to `phase-3`, in order, each green on its own gates, each leaving `golden/off/`, the pinned hash and `testdata/checkpoints/` unchanged (the minimum is 1 500 lines and the maximum 3 000, each reviewable). The module is the largest left in the stage: three reader families (PRO, CLR, MER and CRD; SKL, SET, SPN and POP; INT and CHK) are in other modules' files, so each reader group has its own PR with its own proof.

- **PR A (Phases 1 to 3): edits, `internal/institution`, the configuration surface.** No behaviour. Size estimate: 2 700 lines including tests and the doc and spec edits. It lands the catalogue appends of the config files early.
- **PR B (Phases 4 and 5): state, the round, the event, snapshot, logs, checkpoint, fork.** The module runs and is observable; no right or price reads it. Size estimate: 2 500 lines.
- **PR C (Phases 6 and 7): the rights and the legal prices.** Edits to PRO, CLR, MER, CRD, SUB files. Size estimate: 2 700 lines.
- **PR D (Phases 8 and 9): the slots, the membership group, SKL and the shared headroom.** Edits to INT, SKL, SET and spawner files. Size estimate: 2 300 lines.
- **PR E (Phases 10 and 11): overlay, golden, scenarios, the gate.** Size estimate: 1 700 lines of tests and data plus a golden of tens of thousands of lines (not counted: a recorded file). If review finds it under the minimum, fold Phase 10 into PR D and keep PR E as the gate alone.

**Minimal slice the roadmap implies.** "Needs POL, SKL, SET" only, so the smallest slice is PR A and PR B (bodies, members, the round, restorable). That slice is useful to no one yet: no right refuses and no price clamps. The plan does not recommend stopping there.

## Catalogue and shared-file appends (merge-conflict forecast)

Other Stage 3 modules (EMP's rest, CAP, RSK, INS, FIS, PLC, SAV) append to the same files. Appends are in module order and never inserted (rule 2.10.5), so a conflict is a textual one at the end of a list. Files marked * are the likely ones, and the supervisor orders the merges.

| File | ORG append | Also touched by | Note |
| --- | --- | --- | --- |
| `config.go`* | `Modules.Institutions` (Go line after the existing pointers), the five types, `canonical` call | every Stage 3 module | PR A; keep both lines in module order |
| `hash.go`* | `pairsInstitutions` id 21 in a continuation between ids 19 and 24 | every module | no `Role` id is taken |
| `bounds.go`, `defaults.go`, `errors.go` | three bounds; `DefaultInstitutions`; seven sentinels | every module; `ErrUnknownBody` and the others are ORG's alone | medium |
| `validate.go`*, `config/mapping.go`* | the call of `validateInstitutions`; the TOML tables | every module | high (the same lines) |
| `value.go` | `BodyID`, `NoBody`, `RightID`, `ControlID` | every module with a typed ID | low |
| `event.go`* | five events, `ReasonNotPermitted`, the activity and `Who` constants, `Names.Bodies`, `Rights`, `Controls` | EMP, CAP, RSK, INS, FIS, PLC, SAV | high; land PR B early |
| `event_test.go`*, `observe_internal_test.go`, `cardinality_test.go` | five `notBuiltEvents` entries deleted; rows for the five | every module | the lists are the conflict |
| `snapshot.go`* | `Bodies`, `LegalLevels`, `Suspended` | CAP, RSK, FIS | slices at the end |
| `tick.go`* | two rows, two guarded calls, two `notBuiltPositions` entries deleted | EMP, CAP, FIS, SAV, PLC (the CCN of the straight line) | the CCN helper split of plan 2-14 O7 may be needed |
| `world.go`, `construct.go`* | `org *institutions`; `modulesOn` name; the construction call; `skills.headroom` and `credit.rateCap` assignments | every module | medium |
| `phase_production.go`, `phase_trade.go`, `clearing.go`, `internal/clearing/admit.go` | the start check; the facts of an order; `book` pair, `codeReasons` row, `AdmitClamp` | EMP, LND, STO (`NoRoom`), FIS (a toll) | PR C; `Facts` grows by three fields |
| `contract.go`, `credit.go`, `credit_round.go` | rows 5a and the clamp; the seam type | CRD, MER, INS | the seam change is the CRD owner's |
| `skills_queue.go`, `skills_report.go`, `internal/skill/*` | the member test and the least maximum; `noteHeld`; the applicant count | SKL (merged), SET (`noteRoleMove`) | PR D |
| `settlements.go`, `phase_exit.go` | none in `settlements.go` (the wrap is ORG's `orgHeadroom`); one guarded call in `fillRoles` | SET, EMP, POP | one line |
| `subsistence.go` | one guarded call for `Blocked` with `OwnUseBound` | SUB | one line |
| `integration_slots.go`, `agent.go` | `SlotState` two fields and one append in `Slots`; `AgentState.Membership` and one call in `Agent` | LND (`RentScales`), CRD, RSK, FIS, PLC | the lists are the conflict |
| `checkpoint_sections.go`* | row 31, an `agentTails` entry, entry deleted from `notBuiltSections` (`checkpoint_save.go` and `checkpoint_restore.go` are not touched) | every module with a tag | the skeleton table is the conflict |
| `registry_test.go`* | the rows leave `notBuilt` | every module | the lists are the conflict |
| `internal/logkeys` | nine keys and the reason value | every module | low |
| `doc.go`* | the ORG terms and type names | every module | medium |
| `architecture.md` 2.6 and 3.3, `AGENTS.md` Layout*, rule files with a RULE-TAG* | the continuation files, three typed IDs, a fresh tag | every module that adds a file | the tag line always conflicts: the supervisor applies these edits |
| `scripts/ci/coverage-thresholds.txt` | `internal/institution 90` | every module with a new `internal/` package | one line; PR A |
| `otelobserver/adapter_test.go` (`ignoredEvents`), `events_test.go` (`allEvents`), `testdata/all_events.txt` | the five events, in the `ignoredEvents` list because the instruments are deferred (Question 8) | every module that adds an event | tests and a text file only, in the separate module; adapter code untouched |
| `testdata/chains/overlay/institutions*.toml`, `testdata/golden/institutions/` | new files | STY, CLB | low |
| `testdata/golden/all/`, `all_pack_test.go`, the all-modules checkpoint fixture | **none** | every module | moved once at Stage 3 close |

## Hook discipline

Each hook is **one guarded call into a method that `institutions*.go` defines**, or one appended field of a core type. A hook holds no branch on module data and no arithmetic (architecture.md 2.10.1, 2.10.2). Every line below needs the supervisor's approval before the PR that carries it starts; none is new in kind. Where a hook would carry logic, the logic moves into `institutions*.go` and the hook is the call.

| # | File | Function | The call or field | Kind (ADR 0010, 2.10.1) |
| --- | --- | --- | --- | --- |
| H-1 | `world.go` | `World` | field `org *institutions` | pointer in `World` (2.10.2) |
| H-2 | `construct.go`, `tick.go` | construction; `modulesOn`; `positionsFlows`/straight line | `w.org = newInstitutions(...)` guarded by the pointer; a `modulesOn` row; `p19aMembership` and `p72bInstitutions` rows and guarded calls; two `notBuiltPositions` entries deleted | construction calls (d), tick straight line (2.10.3) |
| H-3 | `construct.go` | after `newSkills`, `newSettlements`, `newCredit` | `skills.headroom = w.orgHeadroom` and `credit.rateCap = w.orgRateCap`, each guarded by the modules it needs | construction call (d); the two seams are slots built for ORG |
| H-4 | `phase_production.go` | `admitJob` | `if w.org != nil { if r := w.orgStartRefusal(a, region, req.Recipe); r != "" { w.refuse(...); return } }` after `admit` | sub-step guarded call (e); **PRO's file, approve as the reader's side of FR-PRO-048** |
| H-5 | `phase_trade.go`, `clearing.go`, `internal/clearing/admit.go` | the facts of an order; `book.admit`; `Admit` | `f.NotPermitted = w.org.orderBarred(a, o)` guarded; `book` pair `Cap`, `Floor` set by the module; `AdmitClamp` replaces the price; one count call | field of a view (f) and guarded call; **CLR's file, approve as spec-clearing.md 3.2.1** |
| H-6 | `internal/clearing/admit.go`, `clearing.go` | `codeReasons` | the row `NotPermitted` | catalogue |
| H-7 | `contract.go` | `offerChecks`, `takeRefusal`, `admitOffers`, `admitTakes` | the row 5a check and the clamp, each one guarded call | **MER's file, approve as spec-merchants.md 9.3** |
| H-8 | `credit.go`, `credit_round.go` | `rateCap`; `admitCreditRequests` | the seam takes the limit and returns the replaced one (D8) | **CRD's seam, approve as spec-credit.md 5.2.3** |
| H-9 | `subsistence.go` | the own-use start | one guarded call that lets `not_permitted` block only with `OwnUseBound` | **SUB's file** |
| H-10 | `integration_slots.go` | `SlotState`, `Slots` | two appended fields and one guarded append | field of a type (f); INT's file |
| H-11 | `agent.go` | `Agent` | `AgentState.Membership` filled by one guarded call | field of a view (f) |
| H-12 | `skills_queue.go` | `trainersOf`, `fillPlaces` | the member test and the least maximum, two guarded calls | **SKL's file, approve as spec-institutions.md 5.7** |
| H-13 | `skills_queue.go` | `completeReady` | `if w.org != nil { w.org.noteHeld(g, r) }` when a completion is held | guarded call (e); **SKL's file** |
| H-14 | `phase_exit.go` | `fillRoles` | `if w.org != nil { w.fillRoleBodies(g) }` beside `fillRoleGates` | sub-step guarded call (e) |
| H-15 | `event.go`, `snapshot.go` | the events, constants, rows, names | appended in module order | catalogue files |
| H-16 | `snapshot.go` | the region/snapshot builder | `if w.org != nil { w.fillBodies(&s) }` | guarded call in a filler |
| H-17 | `checkpoint_sections.go` | the section table, `agentTails` | row 31 and one tail entry (`present`, `save`, `load` method values); `auditModules` gets none (ORG holds no money, goods or conserved quantity) | checkpoint hook |
| H-18 | `validate.go`, `config/mapping.go`, `hash.go`, `config.go`, `defaults.go` | the module seam and `warnings` (one line beside `storageWarnings`), the loader, the writers, `Config.canonical` | one call or row each | catalogue / loader keys |

Not hooks, because they are not needed: no `agentJoined`, `agentRoleChanged`, `dropAgent`, `relocationRefusal` or `checkTradePlan` entry (membership is a status held in ORG's own slices and tested at read; a removal clears the field with the agent's slot and a role change or a move is seen at the next round, FR-ORG-009, spec 4.2.2; the one notification, `noteHeld`, is H-13), no `tradableItems` item, no `orderFacts` registry beyond H-5, no `option.go` option, no `ledger.go` row.

## Gate economics

- Full gates (gate R and gate F, `check.sh`, `full.sh`) run **once per PR, on the final head**, with `GOMAXPROCS=8`, and **one gate run at a time on the host**: earlyoom kills the 20-plus GB root test binary when two share the machine (three gate F runs of POL PR B were killed at 19, 28 and 38 GB). While a gate runs, no other session starts one. Five PRs make five full runs; the plan does not run them per phase.
- During a phase, run only the focused tests and `go test -short -race ./...` (and `-run` on a phase's own tests); do not run gate R or F per phase, per commit or per fix.
- If a gate run fails, fix, then rerun only the failing package; the full gates are rerun once more on the final head if the fix touches non-test code outside that package.
- `gomu` only inside Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`, per new file, never on the host; it does not overlap a gate run.
- `examples/realm` `TestServe_*` can time out under `-race` when several gate runs share the machine; a rerun alone settles it (not in this module's scope to harden).

## Codacy hygiene (every phase)

The Codacy gate fails on **any new Info-or-higher issue** (not on coverage, complexity or duplication). Before a commit that ends a phase, and before every push:

- Every changed `.md` (this plan, specs, rule files, notes, ADRs, the roadmap) passes `npx markdownlint-cli2 <file>` (run with `PATH=/home/andy/.nvm/versions/node/v24.22.1/bin:$PATH`): a blank line around every list, heading and fenced block (MD032, MD031, MD022), table separators written as `| --- |` (MD060), no double blank line (MD012), no trailing space, a single trailing newline, no hard tab outside a code block. `.markdownlint.json` already disables MD013, MD037 and the code-block form of MD010.
- **No Go file over 500 non-comment lines.** The module splits into six files for that reason (Scope); a file that is already over it (Phase 1 audit: `contract.go`, `clearing.go`, `event.go`, `snapshot.go`) takes only the one-line hooks, never the bulk of a reader.
- Every new or changed Go function stays within **Lizard CCN 10, NLOC 50, 100 lines** (`lizard -l go -C 10 -a 8 -L 100 -w <files>`), test functions included: split a table test into subtests rather than one long function. A hook that raises a shared function's CCN is preceded by a refactor-first commit (Phase 1 measures `admitJob`, `book.admit`, `admitOffers`, `admitTakes`, `completeReady`, `fillRoles`, the straight line of phase 1, `Slots`, `Agent` and the validator functions the module touches).
- `gofmt -l .` empty, `go vet ./...`, `staticcheck ./...` clean on the changed packages; no new unchecked error, no shadowed variable, no unused parameter or result.
- Nothing is pushed until these pass on the files the push changes.

## Cross-plan contract

Consumes (identifiers as the tree has them at `15bcf18`; re-check at implementation):

- `skills.headroom`, `completionHeadroom`, `completeReady`, `trainersOf`, `fillPlaces` <- plan 3-skl
- `roleHeadroom`, `fillRoleGates`, `noteRoleMove`, `regionHas` (the `Needs` flag of a body), `ErrUnknownSettlementFlag` <- plan 3-set
- `credit.rateCap`, `priceControlled` <- plan 3-crd; `offerChecks` row 5a and `takeRefusal` <- plan 2-16 (MER)
- `Polity.Chain`, `internal/polity` <- plan 3-pol
- `clearing.Admit`, `Facts`, `codeReasons` <- plan 1 (CLR) and plan 3-sto (`NoRoom`); `admit` of `production.go` <- plan 1-5
- `SlotState`, `Slots`, `AgentState` <- plan 2-18 (INT); `agentTails`, `checkpointSections`, `notBuiltSections` <- plans 2-2 and 3-sto
- `positionList`, `notBuiltPositions`, `forkAtN`, `forkCase`, the `Warn` collector <- plans 2-2, 2-14
- `internal/checked`, `internal/ordering` (`Key`), `internal/logkeys` <- Stage 0

Provides:

- `w.orgHeadroom(g, r)`, the member test, the resolved rights and ranges, the five events, `Snapshot.Bodies`, `LegalLevels`, `Suspended`, `AgentState.Membership` -> SPN and POP (read through the view), IND (`BodyShare`, `LegalPriceGap`), `otelobserver` (later instruments), PLC (the `legal` lever kind over `SetLegalLevels`, N-205), EMP (the hire's `Who` test, D10), CAP and LND (a right over a recipe works without them knowing)
- `overlay/institutions.toml`, `overlay/institutions-inert.toml`, `testdata/golden/institutions/`, S137 to S141 -> STY, REF

## Dependencies on other Stage 3 modules

ORG needs POL, SKL and SET (roadmap), all merged. It **fills seams** left by modules that built before it:

| Module | Relation | Seam here |
| --- | --- | --- |
| SKL | read by ORG: the places, the completion, the applicant count | H-12, H-13, one assignment (H-3) |
| SET | read by ORG: the flag of `Needs`, the class limit into the headroom | `regionHas`, `orgHeadroom` (D9) |
| POL | read by ORG: the chain of a region at construction | `Polity.Chain` |
| CRD | read by ORG: the rate cap | H-8 (D8) |
| MER | read by ORG: rows 5a and the clamp in the round | H-7 |
| PRO, CLR, AGT, SUB | read by ORG: a start, an order, a blocked own-use start | H-4 to H-6, H-9 |
| SPN, POP | read the same view; no new field | H-14 |
| EMP | its formation is checked later (D10) | none now; a note |
| LND | a right before `no_endowment`; a rent control is LND's slot | order of the checks in H-4 |
| CAP, RSK, FIS, INS, PLC, SAV, TEC, STO | none | textual conflicts only |

Build order is independent of the unbuilt modules. Only EMP's hire form and PLC care.

## Testing

Testing entry (testing.md 11) for the PRs: the requirements and tests are in the table above. The lowest level is the `internal/institution` unit test wherever the rule is pure (the exhaustive admission table and the seeded clamp books are unit tests, not models), the root integration tests of Phases 4 to 9 for what needs a world, and S137 to S141 and the module-alone golden at the top.

- **Golden files.** `golden/off/` (`TestGolden_OneOff`, `TestGolden_OneEvents`, `TestHashConfig_ReferencePinned`) and `testdata/checkpoints/` do not move in any phase. `testdata/golden/institutions/` is new, recorded once and never regenerated to absorb a failure. `golden/all/`, `allLayers` and the all-modules checkpoint fixture are not touched by any module PR; `Modules.Institutions` joins `allPackNotYetOn`.
- **Pack.** The module-alone golden is on pack `five` with `overlay/institutions.toml`, chosen by a probe that fails an empty golden (Question 2).
- **Registry and catalogue.** The five scopes and `Modules.Institutions` leave `notBuilt`; `notBuiltEvents` loses five; `notBuiltPositions` loses `P1.9a` and `P7.2b`; `notBuiltSections` loses tag 31.
- **Fuzz.** `FuzzAdmit`, `FuzzRange`, `FuzzResolve`, `FuzzValidateInstitutions`, `FuzzDecodeInstitutions`, `FuzzLoad` corpus extended.
- **Fork-at-N.** `TestFork_Institutions` (Phase 5), extended with a slot call in Phase 8.
- **Race skips.** Every multi-seed or long test that skips under `raceEnabled` starts with a `gateFPrefixes` prefix (`TestScenarioORG*`, `TestDeterminism_Institutions*`, `TestGolden_Institutions*`, `TestCheckpoint_Institutions*`, `TestFork_Institutions`, `TestEmptyModule_*`); the unit tests and `TestInstitutions_WorkIsCounted` never skip.
- **Mutation.** `gomu` only in Docker with the caps, per new file of `internal/institution` and `institutions*.go`.
- **Broken-version pass.** Phase 11 (testing.md 9.3).

## Observability

Observability entry (observability-and-logging.md 9).

- **Events:** `MembershipChanged`, `RightRefused` and `PriceControlled` (aggregates per (region, body), (region, right), (region, control) at P7.2b, bounded by the configuration), `LegalLevelSet` and `RightSuspensionSet` (one per changed key at a call). Plain data, no map, constructed only under `w.obs.supplied`, after the transition, in the order of spec 7.3.3 (rules 4.1, 4.3, 4.5). A refused order is also an `OrderRejected` with the new reason `not_permitted` (the reason joins the closed set). No existing event gains a field.
- **Logs:** `Info` only the existing construction line, which gains `bodies`, `rights`, `controls` and `institutions` in `modules`. `Warn` at construction only, one per warning of spec 10.2 with `rule` and `path`, plus the one tick `Warn` "body has waiting households" per (region, body) by pattern B, bounded by the configuration, counterpart `MembershipChanged.Waiting` (rule 3.2.2). `Debug`, guarded by `Enabled`: "start refused" and "order rejected" (existing lines, new reason), "place not open" (`agent`, `role`, `region`, `reason` = `not_member`). No `Error`.
- **Metrics:** none built (the adapter change is listed in N-405). Spec 11.6 proposes `economysim.body.members`, `body.waiting`, `right.refusals` and `control.clamped`; the rows are already in observability 5.3 and 5.3.1; the `otelobserver` instruments are deferred (Question 8), as SKL's, STO's and SET's were. The events carry bounded identifiers, so the instruments can be added without a core change. `economysim.reason` of the rejected-orders metric gains `not_permitted`, a closed-set value. No new unbounded attribute; series are bodies times regions, rights times four, controls times two.
- **Non-interference:** nothing draws; a run with and without observer and logger is identical (Phase 5 test). The clamp is counted, never silent (N-045).
- **New log keys:** the nine of spec 11.3, in `internal/logkeys`; they are already in the table of rules 3.3.2, so E1 edits the rule file only for the continuation files and the typed IDs (fresh RULE-TAG).

## Risks and open items

| # | Item | Owner |
| --- | --- | --- |
| R1 | **Reader-side edits in six other modules' files** (PRO, CLR, MER, CRD, SKL, SUB, INT). Each is one guarded call or one field and moves behaviour **only where a body, right or control is declared** (no pack of the tree has one), so no existing golden can move. A wrong ruling on the clamp position (D6) would change a result for a controlled pack only | the owners of those files; the supervisor approves the hook list |
| R2 | **`Admit`'s signature.** `AdmitClamp` is added and `Admit` wraps it, so every caller and table test stays; the cost is two entry points in `internal/clearing`. If Phase 1 prefers one, the callers (`clearing.go:325` and the tests) change in PR C only | CLR owner |
| R3 | **S141 and FR-ORG-001 disagree** (E5, Question 4). Until settled, the inert-overlay comparison is written with the module's own outputs excluded | ORG owner, the supervisor |
| R4 | **The reference chain has no price control at a stable level.** A cap below `CostFloor` of the unit cost freezes a market (spec 6.5.5, ORG-A1); the overlay's cap is chosen above it by the probe and the file header says the numbers are placeholders. S137 reports the shortage and asserts only the invariants | ORG owner, STY |
| R5 | **The held completion needs a skilled role and a binding body** in one pack. The probe may find none in the window; then the golden shows no `Held` and the layered world and the unit test carry it | ORG owner |
| R6 | **Lizard and file size on shared functions** (`admitJob`, `book.admit`, `completeReady`, the straight line, `Slots`, `snapshot`, the validator `warnings` at CCN 85). Phase 1 measures them and a refactor-first commit lands before the hook (the STO N-357 precedent). Several host files are already near 500 lines | ORG owner |
| R7 | **Concurrent Stage 3 branches** (`s3-cap-1`, `s3-emp-1` and others exist) append to the same lists; the hash id 21 and tag 31 are fixed by the spec and cannot collide, but `SlotState`, `Modules`, notes and event lists can. Whichever merges second renumbers its own appends | the supervisor |
| R8 | **A first-come place.** Admission by seniority favours incumbents (ORG-A5) and a place goes to the first held trainee in queue order; both are counted (`Skipped`, `Waiting`, `Held`) and stated. N-122's remedy applies to the tie only | ORG owner |
| R9 | **EMP's closed shop is not enforced until the hire form lands** (D10). A pack that declares `Who = executor` today gets the principal test only; `places_below_start`-style honesty: a warning `right_who_unenforced` is **not** added (spec 10.2 lists none), the note is | EMP owner |
| R10 | **The credit seam change** (D8) touches a merged module. Nothing sets the seam today, so the change is invisible to every existing run; the CRD golden is the check | CRD owner |

Note verdicts (text only; the supervisor edits `doc/notes.md`): N-140 apply the ORG leg (a right over a store is reserved) and keep the other legs. N-155 apply for ORG (a body's `Needs` and its places read the class and the count at a position before P7.2a, with their counters and the register entry) and keep for TRN. N-160 leave open; ORG reads the chain at construction and resolves a table once, innermost, with no crossing. N-161 leave open (no transfer of a region is offered). N-205 leave open until PLC (`SetLegalLevels` is integrated; PLC's `legal` lever is its own row). N-244 apply for ORG (the membership is a slice of the module, saved in tag 3's tail) and remove with the code. N-192 apply in Phase 9 (the least of two limits) and remove. N-193 apply in Phase 9 (a held completion, counted `Held`) and remove. N-268 apply for ORG (no new field: the same `Headroom`, `NoLimit`, `RoleCeiling`). N-090 keep (ORG registers none, confirmed). N-122 keep (a first-come place, counted). N-131 and N-194 keep (the reserved list). N-152, N-172 and N-173 leave open (not ORG's to settle). N-017 and N-027 standing (terms and placeholders). N-130 and N-131 leave open (`Snapshot.Bodies` is what an indicator needs).

## Questions only the owner can answer

Every question is delegated to the supervisor. The recommendation is taken as accepted unless the supervisor says otherwise.

1. **Number of pull requests: five, four or three?** Recommended: five (A: edits, pure rules, configuration; B: state, round, observability, checkpoint; C: the rights and the legal prices; D: the slots, SKL and the headroom; E: proof and gate), so each reader family is reviewed with its own tests and every PR stays between 1 500 and 3 000 lines. If PR E is under the minimum, fold Phase 10 into PR D.
2. **Which pack carries the module-alone golden?** Recommended: `five` with `overlay/institutions.toml` (spec 13.5, S138), chosen by a probe that fails an empty golden (a `MembershipChanged` with `Waiting`, a `RightRefused`, a `PriceControlled`); the held completion is asserted on `five` plus the skill overlay. A controlled staple on `one` is ORG-S1's, not the golden's.
3. **The clamp inside the admission.** Recommended: `internal/clearing.AdmitClamp` with `Admit` kept as a wrapper, the range held by the `book` (D6). The alternative (clamp before `Admit`) breaks the order of spec 6.3.1 and is rejected.
4. **S141 and FR-ORG-001.** Recommended: S141 compares the inert overlay with the world without the module, excluding the module's own events, snapshot rows, names, hash, `Snapshot.Modules` and construction line, as SET-S4 does; FR-ORG-001's strict form is tested on the module present and empty. E5 corrects the spec.
5. **EMP's `Who` test.** Recommended: ORG reads the principal only at P2.5 now and EMP adds the executor test with the hire form, by one guarded call into `institutions_rights.go`; a note records it (D10).
6. **Edits in other modules' files** (H-4 to H-9, H-12, H-13). Approve ORG editing `phase_production.go`, `phase_trade.go`, `clearing.go`, `internal/clearing`, `contract.go`, `credit.go`, `credit_round.go`, `subsistence.go`, `skills_queue.go`, `skills_report.go`, `internal/skill` and `integration_slots.go`? Recommended: yes, each a small edit reviewed with its PR, moving behaviour only where a body, right or control is declared.
7. **`AgentState.Membership` and the appended-typed-ID rule.** Rule 2.10.5 forbids an appended typed ID in an event or a snapshot row, and `NoBody` is -1. Recommended: `AgentState.Membership` is `{Member bool; Body BodyID}`, the zero value meaning no membership, so a world with the module off renders as before and `golden/off/` does not move; the spec's "BodyID or NoBody" is read as that pair (E9).
8. **Metrics.** Recommended: defer the four instruments and the two slot-event instruments to a later change to `otelobserver/`, as SKL, STO and SET did; the events and the snapshot carry the data and `TestEvents_MetricsDerivable` shows it.
9. **The credit seam.** Recommended: the seam takes the limit and returns the replaced one (D8), so ORG counts the lowering in its own scratch and the count survives the reset of the round. The alternative (read CRD's count before `rd.reset()` through a second seam) keeps the count in CRD and adds a second assignment.
10. **Where the least of two limits lives.** Recommended: ORG's `orgHeadroom` wraps SET's `roleHeadroom` (D9), so `settlements.go` is untouched and a pack with ORG and no SET still has a body limit. The alternative (a term inside SET's function) makes SET read ORG.

## PR A result (Phases 1 to 3, merged on `phase-3` as `e4459da`)

Rule tags read: architecture=5c0416f502ba5a59f7b2ae08 (replaced twice by a fresh tag, last 9fae4fd3b85f604b10bec2d6), observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367. Notes tag read: f85c4b2d7217a479ec7c16dc; the new tag is ed16ceefa88eadd3081d9735. The notes read in full were the ORG and ALL ones and those the plan names, not the whole file.

Built: the owed edits E1 to E13 (the continuation files, the glossary, the notes N-397 to N-399), `internal/institution` (`Resolve`, `Overlap`, `Range`, `Clamp`, `Tally`, `Lapse`, `Admit`, `Headroom`, `Least`), the configuration surface (the five types, hash id 21, the bounds, the seven sentinels, the validator and the eleven warnings, the loader, the nine log keys) and their tests.

Deviations and choices the plan left open:

- A sixth continuation file, `institutions_warn.go`, because the rules and the warnings together passed 500 non-comment lines; it is in `architecture.md` 2.6 and `AGENTS.md`.
- A produce right with no `Who` is canonicalised to `principal` and any other right with `principal` to none, so the two spellings of a default hash alike (FR-ORG-051).
- The body's trainee bound is `MaxBodyTrainees` in `bounds.go`; a body lists an entry once (`ErrDuplicateID`).
- `internal/institution` has 98.8% coverage; `gomu` was not run (Docker, per new file: still owed before PR E).

## PR B result (Phases 4 and 5)

Rule tags read: architecture=9fae4fd3b85f604b10bec2d6, observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367. No rule file, spec or note was edited, so no notes tag applies; the note and spec edits PR B owes are listed under "Left for the supervisor".

Built: the state and construction (`institutions.go`: the eligibility and scope tables, the membership in two slices of the module, `isMember`, the names), the round at P1.9a (`institutions_round.go`), the resolved rights (`institutions_rights.go`) and legal ranges (`institutions_legal.go`), the events, the Warn, the snapshot rows and tag 31 with the membership tail of tag 3 (`institutions_report.go`), the five event types, `Snapshot.Bodies`, `LegalLevels` and `Suspended`, `Names.Bodies`, `Rights` and `Controls`, the two positions and the section 31 row. No hash id, tag, note or event slot beyond the plan's was taken. `golden/off/`, `golden/all/`, the pinned hash and `testdata/checkpoints/` are unchanged.

Hooks used: H-1, H-2 (the two positions), H-15 to H-18; none of H-3 to H-14 (the readers are PR C and PR D).

Deviations and choices the plan left open:

- The round does the lapse pass of every body first and then the admission of each (body, region) (question 1 of the log). Spec 4.1.1 says "for each body: lapse, then admit"; the two give the same result unless a household lapses from a body and is eligible for one with a lower `BodyID`, which the spec's 4.2.1 says is a candidate in the same round. The order of the bodies would then change a result, against FR-ORG-019.
- `positionsTail` was at the Lizard CCN limit, so `positionsReview` (P7.2a and P7.2b; the `positionsClose` of P8.3a to the audit is LND's) is split from it by a pure move, and the test of the straight line knows the sixth part.
- `snapshot.go` was at 498 non-comment lines, so `Region` and `regionEndowments` moved to `regions_status.go` by a pure move.
- The resolved rights and legal ranges, the levels in force and the suspensions are built and rebuilt (`resolve`) and tag 31 saves and loads them, but nothing sets a level or a suspension until PR D and nothing reads a right or a range until PR C; their tests plant the state. `MembershipChanged.Held` is 0 until PR D.
- `TestReaders_StaticCheck` allows `institutions_round.go` to read the flag with `regionHas` (the reader of SET's hook H6, spec 3.5.2); `TestPolity_NoRegionFieldRead` lists the other ORG files among those that may name a jurisdiction.
- `Names.Bodies`, `Rights` and `Controls` are nil when none is declared, so the module present and empty leaves no trace in the names.
- The end-to-end test of PR A now compares the run without the module's own outputs (its names, rows, list of modules and hash), because the module now has outputs.
- `HashWorld` of a restored world differs from that of the uninterrupted one in the small world of `institutions_checkpoint_internal_test.go`, with the module on and with it off; the fork tests compare events, snapshots, checkpoint bytes and logs, which agree. It is not the module's, and the test compares snapshots.

Tests by layer: unit (`internal/institution`, unchanged); world (internal: the tables, the round, the need, stale fields, lapse, the resolved rights and ranges, the levels, the work count, the allocation test, the draw guard, the checkpoint of the tail and its corrupt cases, no region-bound obligation, the order of the events); integration (the events and their order, the snapshot rows, the names, the construction line, the Warn, tag 31 and its corrupt cases, `FuzzDecodeInstitutions`, the fork at N, non-interference, the empty module against the off world, determinism over the driving cases, the derivable metrics). Run: `gofmt`, `go vet`, `staticcheck`, `go test -short -race` on the root and `internal/institution`, and the named golden, hash and allocation tests; not run: gate R, gate F, `check.sh`, `full.sh` and `gomu`.

Left for the supervisor (nothing in `doc/notes.md` or a spec was edited, because that needs the whole file read and a fresh tag):

- N-244: the ORG leg is applied (the membership is a slice of the module, saved in tag 3's tail); the note is a standing rule for other modules too, so it stays open with an update line.
- Spec 4.1.1 of `spec-institutions.md`: say that every lapse runs before any admission.
- A note for PR D: `Held` and the slot setters fill what PR B built; the tests that plant state (`orgTablesToy`, the levels test) are to be replaced by the call's tests there.

## PR C result (Phases 6 and 7)

Rule tags read: architecture=645d155e21f440e7e94863d7, observability=9701798a5a03417c57f1a38a, testing=66af7645b01fef4e2796c367. No rule file, spec or note is edited, so no notes tag applies; the edits PR C owes are listed under "Left for the supervisor".

Built: the rights of `produce`, `sell` and `buy` and the legal prices. `internal/clearing` gains the code `NotPermitted` (appended after `NoRoom`, checked after `NotTradable`), `Facts.NotPermitted`, `Cap` and `Floor`, and `AdmitClamp`, which replaces the limit between the settlement bounds and the funds row; `Admit` is its wrapper. The root reads a right at a start (`admitJob` and a trainee's request in `skillsWork`, `Who` and the own-use exemption), at an order (the fact `NotPermitted` of `submit`) and at row 5a of an offer and a take of the contract round (destination region). The book holds the module and reads its range at each admission, so the estate's ask and a counterparty's order are clamped too; `OrderPlaced`, the reservation and the entry carry the replaced limit. The contract round clamps an offer's ask after the checks and a take's bid between `overflow` and `insufficient_funds` (`takeFunds`). The credit seam became `rateCap(g, limit) (replaced, lowered)` and `institutions.rateLimit` fills it. `RightRefused` and `PriceControlled` are told at P7.2b from counts kept only under an observer. `golden/off/`, `golden/all/`, the pinned hash and `testdata/checkpoints/` are unchanged; no hash id, tag, note, event or position slot beyond the plan's was taken.

Hooks used: H-4 (`admitJob`, `skillsWork`), H-5, H-6 (`submit`, `book.admit`, `codeReasons`), H-7 (rows 5a, the two clamps), H-8, H-9, and the H-3 half for `rateCap`. Not used: H-10 to H-14 (PR D).

Deviations and choices the plan left open:

- The task listed H-12 and H-13 (`skills_queue.go`) for PR C; the plan puts them in Phase 9 (PR D), and so does this PR.
- `AdmitClamp` returns `(code, price, changed)`: the side of the order says whether the change is a lowering or a raising.
- A clamp is counted only for an order that is admitted (an order refused for funds after the clamp is a rejection, counted by `OrderRejected`); likewise for a take and a request.
- A refused start counts one in `RightRefused.Produce` for each request, not for each household.
- A trainee's job is also read against a `produce` right, through one guarded call in `skillsWork` (a trainee's request does not pass `admitJob`): the trainer is tested as for `principal` whatever `Who` says, and the trainee is exempt from the executor test (spec 5.3.2). This is one more site than H-4 names.
- `ReasonNotPermitted` is column -1 of the `RecipeRefused` tally (counted by `RightRefused`, as `no_title` is counted by `RentPaid`); it blocks an own-use start in the any-other slot of `blockedBy`, so the hashed shape does not change.
- `book.admit` was split into `checkRoute` and `facts` by a pure move first (a refactor-first commit), and `takeRefusal` into `takeRefusal` and `takeFunds`, for the Lizard limit. Lizard does not parse `takeRefusal` and its neighbours in `contract.go` at all, so their CCN was counted by hand (10 and 5).
- `contract.go` (1 015 non-comment lines), `subsistence.go` (579) and `event.go` were over 500 before; they took only the lines of the hooks.

Tests by layer: unit (`internal/clearing`: the not_permitted row, the clamp rows, the order of the checks, the wrapper); world (internal: the start refusal and its order, `Who`, trainee, own use, the prohibition, no read at a completion, an order refused and its count, the estate exempt and clamped, the reservation, entry and event at the replaced limit, no rejection and no draw, `PriceControlled` counts, offer and take rows 5a and clamps, the rate limit of example C, the claim in force unchanged, a request never refused for the cap, allocation of every reader, the static check of the reader sites); integration (pack one over seeds: every trade inside the cap and inside the floor with the audit on, and the shortage signal of a binding cap). Run: `gofmt`, `go vet`, `staticcheck`, narrow `go test` of the touched tests through the slot guard, `go test -short -race ./internal/...`; not run: gate R, gate F, `check.sh`, `full.sh`, `gomu`, `golden/all`.

Left for the supervisor (nothing in `doc/notes.md` or a spec was edited, since that needs the whole file read and a fresh tag):

- A note: a clamp counts only an admitted order (spec 6.3.1 says "the count of the replaced orders").
- A note: a trainee's request is read against a `produce` right at `skillsWork`, and `spec-production.md` 4.2 ("tested as the executor is") and `spec-institutions.md` 5.3.2 read differently; this PR follows 5.3.2.
- A note: `RightRefused.Produce` counts requests, not households.
- Spec 5.2 and `spec-clearing.md` 3.2.1 may say `AdmitClamp` returns one flag.

## PR D result (Phases 8 and 9)

Rule tags read: architecture=645d155e21f440e7e94863d7, observability=9701798a5a03417c57f1a38a, testing=66af7645b01fef4e2796c367. No rule file, spec or note is edited, so no notes tag applies; the edits PR D owes are listed under "Left for the supervisor".

Built: the two slots (`SetLegalLevels`, `SetRightSuspensions`, the types `LegalLevel` and `RightSuspension`, `SlotState.LegalLevels` and `RightSuspensions`, the error order of 7.1.1, one rebuild of the resolved tables at the call, `LegalLevelSet` and `RightSuspensionSet` per changed key), the membership group (`AgentState.Membership`, the type `Membership{Member, Body}`), SKL's gates (the trainer is a member where a `train` right is in force, a member holds at most the least of its role's and its body's `MaxTrainees`, the applicant counts the eligible trainers only, `RightRefused.Train` is `min(f, a)` and the Debug line "place not open" names `not_member`), the held completion (`MembershipChanged.Held`), and the shared headroom (the least of the class limit and the body's places in the view of every role and in the slot of the completion). No hash id, tag, note, event or position slot beyond the plan's was taken; `golden/off/`, `golden/all/`, the pinned hash and `testdata/checkpoints/` are unchanged.

Hooks used: H-10 (`SlotState`, `Slots`), H-11 (`AgentState.Membership`, `fillMembership`), H-12 (`canTrain`, `trainersOf`, `freest` and `skillsRoleRows` through `trainerMax`, `fillRoleSkills`, `skillsRoleRows`, `placeRole`), H-13 (`completeReady`: `noteHeld` and `noteRoleMove`), H-14 (`fillRoles`: `fillRoleBodies`), H-3 (the slot `skills.headroom`).

Deviations and choices the plan left open:

- `canTrain` is split into `couldTrain` (SKL's own test, unchanged) and `canTrain`, which adds the one guarded call `mayTrain`; `freest` becomes a method that asks `trainerMax` for each trainer's maximum; `trainerSlot` is unchanged, because the position hash walks the world by reflection and a new field of that scratch moved the skills golden.
- The view's `MaxTrainees` is the least of the role's and the body's only where a `train` right is in force (every trainer is then a member); the snapshot's `Places` is exact for each trainer (N-464).
- The count of households of a body is ORG's own scan, once for each (tick, region), kept exact by `noteRoleMove`, so a pack with ORG and no SET still has a body limit (Question 10); `completeReady` therefore has a second one-line call beside SET's.
- `Held` is counted only under an observer and only where the body's own places are what is short (N-465). `Outcome` of `internal/institution` gains `Held`.
- A restored world had no slot of the settlements module (`wireSeams` ran only in `construct`; N-462). Fixed after the supervisor's ruling: `wireSlots` (`settlements.go`) is called by construction and by `RestoreWorld`, and `TestRestore_SettlementsSlotsAreWired` fails without it. N-462 is therefore not a note.
- The spawner's room is per role, so a body shared by several roles can be overshot in one entry step (N-463).
- Not written: `TestPremium_CheckDReportedNotAsserted` (no premium check exists in the code to report; spec 5.7.5 is a row owed to SKL) and a separate spawner-level `TestEntryLimited_RoleCeilingIncludesBody` (the spawner reads the same `Headroom` and `NoLimit`; `TestRoleView_BodyHeadroom` and `TestPopulation_BirthsCappedByBody` pin the fields it reads). `TestCompletion_HeldWithoutHeadroom` is SKL's name, so ORG's tests are `TestOrgCompletion_*`.
- The two tests of PR B that planted a level and a suspension now call the slots (N-401).

Tests by layer: unit (`internal/institution`: `Outcome.Held` is a field of the struct the round compares); world (internal: the slot calls and their rebuilds, the refusal that changes nothing, the entry equal to the configured level, the membership group, the trainer test and the body's maximum, the filling to the body's limit, the count `min(f, a)`, the least of two limits with and without each module, the held completion and its order, the wiring after a restore, the allocation of every reader, the static check of the reader sites); integration (the errors of both calls in order, `ErrModuleOff` and `ErrInStep`, the slot state in a checkpoint and not in the hash, the fork at N with a call before the boundary, a run of the traineeship under a body with the audit and with and without an observer). Run: `gofmt`, `go vet`, `staticcheck`, `lizard`, the narrow tests above; the slot-guarded run of the golden, hash, position-hash, allocation, fork, checkpoint and glossary tests is recorded below; not run: gate R, gate F, `check.sh`, `full.sh`, `gomu`, `golden/all`.

Left for the supervisor (nothing in `doc/notes.md` or a spec was edited, since that needs the whole file read and a fresh tag):

- Settle N-401 (applied) and add three notes, numbers N-463 to N-465 (N-462 was fixed in code, see above, and its number stays unused):
  - N-463 (ORG, SPN): the spawner's room is per role, so a body shared by several roles can be overshot in one entry step.
  - N-464 (ORG, SKL): the expected wait counts the body's trainee limit only where a `train` right is in force.
  - N-465 (ORG, OBS): `Held` and `Train` count each tick, `Held` only where the body's own places are short; both pressure, not households, and only under an observer.
- `spec-institutions.md` 3.4.2: the count is ORG's own scan (not SET's scratch) and a second `noteRoleMove` keeps it; 5.7.2: `n_max` of the view is exact only with a `train` right; 4.5.2: `Held` counts a body-bound hold each tick.
- `spec-integration.md` 4.7.2: the type of the group is `Membership`.

## PR E result (Phases 10 and 11)

Rule tags read: architecture=87c4da88312e78e6540e5913, observability=9b1ce26066289aa7a163811f, testing=66af7645b01fef4e2796c367. Notes tag read: a328693d6357d39f632bf880 (the header, the index, the ORG notes the change bears on and the tail; not the whole file); the new tag is in `doc/notes.md`.

Built: `overlay/institutions.toml` (a body over the cooks of the centre with 174 places of 218, a produce and a train right, a cap on the staple of `group_a` at 9 000 bp and a floor on that of `group_b` at 11 000 bp, the cook's course of 120 ticks) and `overlay/institutions-inert.toml` (the same body and produce right, places 0, two controls at level 0, **no train right**), the module-alone golden `testdata/golden/institutions/{five.txt, events-five.txt, five.positions.txt}` (new files, 25 positions per tick: the 22 of the off set, the polity's and P1.9a and P7.2b), the probe `TestGolden_InstitutionsExercisesTheModule` (seeds 1 to 4: every run has `MembershipChanged` rows with `Waiting`, `RightRefused` rows and `PriceControlled` rows), `TestGolden_InstitutionsOffLeavesNoTrace`, the scenarios `TestScenarioORG1_PriceCapLifted` to `TestScenarioORG5_InertOverlayIsNeutral`, and `TestInstitutions_TrainRightNotNeutralAtZeroPlaces`. The two viability pack loops skip the two overlays (they need polity and the region `centre`). `golden/off/`, `golden/all/`, the pinned hash, `testdata/checkpoints/` and `testdata/chains/expected/` are unchanged; the scenarios write no expected line (no band, N-015, N-027).

Deviations and choices the plan left open:

- **The inert overlay has no train right (N-540).** S141 as `spec-reference-chain.md` 4.5 wrote it (the same rights) is not neutral: a household that becomes a cook in a tick is a member only from the next round, so with a train right in force the new cook is no trainer for that tick, and the world diverges from the first completions (seed 1, tick 126: `Roles[0].PlacesTaken` 339 against 402). Without the train right the snapshots are identical at ticks 140 and 150; `spec-institutions.md` 14.3 says "no right". `spec-reference-chain.md` 4.5 now says the same. S5 compares the five with the overlay against the five without the module, both with the cook's course (core data), dropping `MembershipChanged`, `RightRefused` and `PriceControlled` and the snapshot's names, hash, module list and module rows.
- **ORG-S1 runs on pack one without the trade overlay.** The contract clamp is carried by the PR C tests. S1 caps the staple at 80, 100 and 120 percent of 96, lifts the control by `SetLegalLevels` at tick 30 and asserts every trade before the lift is at or below the cap, with the audit on; the shortage and the prices after the lift are reported by `Logf`.
- **ORG-S2 and S3 assert the places, not a premium.** No premium check exists in the code (PR D). S3 runs on the five with the trade, information and migration overlays and asserts the 44 outsiders of the first row and that no member appears but by admission; a fall of the members is allowed, because a household that leaves the world or the region between two rows is not removed for the places.
- **ORG-S4** prohibits the own-use recipe that makes the staple with `OwnUseBound` false (no start refused, the households that enter the mode are those of the world without the module) and true (starts refused, counted in `RightRefused.Produce`).
- The golden's solver report is the five's: the solver does not read a body, a right or a control.

Tests by layer: golden (the three files, the probe, the off-leaves-no-trace and the position count); scenario (S1 to S5); world (the train-right lag). Run directly (one seed or a handful of names): the overlays' load, the probe, S1 to S5, the lag test, the golden records; run through the slot guard: the short suite under `-race` and the golden, pinned-hash, position, checkpoint, registry and glossary tests (the result is in `/tmp/claude-1000/dp3-org-5-gate.log`). The notes PR C left for the supervisor are written as N-541 to N-544 (the fourth item, spec 5.2 and `spec-clearing.md` 3.2.1 on `AdmitClamp`, is N-544; no spec text is edited). Not run: gate R, gate F, `check.sh`, `full.sh`, the deliberate-break pass of Phase 11, `gomu` (owed per new file of `internal/institution` and `institutions*.go` since PR A), the arm64 run, the plan audit and the close of the roadmap row, `spec-roadmap.md` and `documentation_index.md`.

Left for the supervisor:

- The close of Phase 11 (the gates, the deliberate-break pass, `gomu`) and the edits of the close: the roadmap row, `spec-roadmap.md` and `documentation_index.md` (the overlays and the golden directory).
- The notes owed by PR C and PR D (N-463 to N-465 are in the file; the PR C notes are not) and the spec edits PR C and D listed.
- `spec-institutions.md` 14.3 and `spec-reference-chain.md` 7.12 may name the contract variant of S1, which this PR leaves to the PR C tests.
