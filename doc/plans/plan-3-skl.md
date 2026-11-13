# Plan: Stage 3, row 1: SKL, `skills.go` and the traineeship half of the skill barriers

**Status: plan written (2026-11-13), not audited. Stage 1 of the work (requirements and plan, no code). Written on branch `s3-skl`, cut from `phase-3` (= `main` at f556398). Format model: plan-2-14.md (git history, `git show 31e3a05^:doc/plans/plan-2-14.md`).**

Sequences the roadmap-v1.md Stage 3 row "`skills.go` | SKL | PRO, WEL" into nine phases and two pull requests. It implements [spec-skill-barriers.md](../specs/spec-skill-barriers.md) (SKL): FR-SKL-001 to 028 and NFR-SKL-001, 002. The tool half (sections 2 and 3 of the spec, except the queue term of FR-SKL-005) is already in the tree (see "What is already built"); this plan builds the **traineeship half** (section 4, 8, 9) and proves both halves together. It touches the spawner (SPN, N-264, N-268, N-305), the agent's consumption (AGT 5.7), welfare (WEL 6.3), the production jobs (PRO 8), the world's removal step (AGT 3.6 step 2), the regions guard (REG 6.5.2) and the checkpoint (tag 10, N-317).

**SKL has no `Modules` pointer.** `spec-config.md` 3.7.2 says "Skill barriers are core: there is no module for them and `TrainingTicks` 0 is 'no traineeship'". So this row has **no hash field id, no `modulesOn` name and no `Config` field to add**: the four parameters (`Role.TrainingTicks` 15, `MaxTrainees` 16, `TeachingLoad` 17, `Region.CapitalReturn` 4) are built, hashed, bounded and validated since Stage 0. "On" means "some role has `TrainingTicks` above 0" (the state exists), and every pack of the tree has none, so `testdata/golden/off/` and the pinned hash cannot move. The checkpoint tag is **10**, already present as a four-count skeleton (N-317), and this plan replaces its writer and reader keeping the skeleton as the prefix. Question 1 asks the owner to confirm this reading.

Rule tags read (the three files; observability sections 5.3.1 table rows 660 to 805 were skimmed by search for `Traineeship`, not read line by line): architecture=bdb8a4b7fa1ed8fd3b2e7aec, observability=f215753306fac0bc7607c4c4, testing=66af7645b01fef4e2796c367.
Specs read: spec-skill-barriers.md (in full); spec-config.md (3.1, 3.7.2, the SKL rows, 8.2); spec-checkpoint.md (4.2 tag 10, F7); spec-spawner.md (3.2, 3.3, 4.4, 6.2, FR-SPN-009 to 011, 020, 049, 054, 058); spec-world.md (6.4 P2.2, 6.7 P7.1, C5, C10); spec-observability.md (4.1, 4.4.8, 5.3, FR-OBS-018 to 021); spec-agents.md (3.6, 5.7, 7.1.8); spec-reference-chain.md (C13, `overlay/skill.toml`, S06). roadmap-v1.md (Stage 2 rows, how each module row was delivered) and plan-2-14.md were read in full.
Notes: the index was read. The SKL-tagged entries (N-039, 071, 090, 110, 138, 149, 193, 244, 264, 268, 305, 317) were read in full and have a verdict in Risks. The ALL entries and the rest of `doc/notes.md` were **not** read in full: this plan writes no spec, so the notes tag of AGENTS.md does not apply to it. **Phase 1 edits specs and must read the whole file and quote the tag.**
Decision block: this plan numbers its decisions SK1 to SK14; the supervisor maps them to D numbers of the stage when it merges the plan.

## What is already built (the tool half, from the tree at f556398)

- Config: `Role.TrainingTicks`, `MaxTrainees`, `TeachingLoad`, `Region.CapitalReturn`, `Recipe.TraineeEligible`, `TraineeYield`, `Recipe.Requires` with the bounds, defaults, hash ids and registry rows (`config.go`, `bounds.go`, `defaults.go`, `hash.go`); the rule `traineeship_needs_weight_one` (`validate.go:1266`, FR-CFG-054), `ErrEligibleHasTool`, the warning `tool_unaffordable` (FR-SKL-004, tested in `validate_rules_test.go`).
- Production: tools held, not consumed, wear by the durability counter, the trainee term of the effective yield (`internal/production/yield.go`, `Effective(site, shock, tech, trainee, abundance)`), jobs that carry an executor and a `principal` (`production.go`), the abandon reason `principal_left`.
- Spawner: the tool half of the entry rule, `internal/occupation.CapitalOK` (FR-SPN-020, 058); a role with `TrainingTicks` above 0 is **never entered** (spawner.go:340). `DecideApply`, the trainee view fields, the refusals `in_traineeship` and `needs_weight_one` and the cause `traineeship` are **not declared** (N-264, N-268; `refuse.go` and `event.go` say so in comments).
- Agent: `consume(a, from, ...)` already takes the account to eat from (`agent.go`, the trainer's for a trainee, FR-AGT-068); nothing calls it with another account.
- Welfare: `PremiumBP`, `HasPremium` on `RoleRow` (`snapshot.go:209`) and the observed wage and premium of the role (`welfare.go:444`).
- Estates: the estate's ask of a durable lot and its markdown (FR-EST-008, 009; FR-SKL-008, 009 ride on them).
- Checkpoint: tag 10 `Skills` with `saveSkills` and `loadSkills` as zero-row skeletons (`checkpoint_save.go:27`, `checkpoint_restore.go:325`); the registry of `region_bound` obligations (`regionBounds.add`, `construct.go:143` to 149); the fork helpers `forkAtN`, `forkCase`.
- Fixtures: `testdata/chains/overlay/skill.toml` (120 ticks, 3 trainees, 1 000 load on `cook` and `crafter_a`), the eligible recipes of C13 in `lines.toml` and `five.toml`, `bands/` files. `expected/*.scenarios.txt` has no S06 row.
- Positions: `P2.2` is in `notBuiltPositions` (`tick.go:84`); P7.1(a) is a sub-step of `p71Entry` (`phase_exit.go:249`), not a position, so `tick.go` gains one entry only.

## Scope

**Builds.**
- `internal/skill` (new package; stdlib, `internal/checked` and `internal/ordering` only; primitives only): the trainer's gain, the teaching accumulator step, the queue order and the requeue order, the open places of a region and role, the ticks-served advance and the completion test (complete, held, wait), `C_train`, the break-even premium of check D and its tolerance, and the run counter of the `Warn`. Question 2 asks whether the package is wanted; the alternative is `internal/occupation` for all of it.
- `internal/occupation` appends: `ApplyOK` (the queue-discounted inequality of FR-SKL-005, with `C_train` and the expected wait `q_w` of FR-SKL-019, both sides as exact 128-bit products and the bracket floored at 0), the refusals `InTraineeship` and `NeedsWeightOne` (with `Facts` fields), and the default `Decide`'s `Apply` emission for a weight-1 agent (N-264, N-305).
- Root: `skills.go` (the module-file order of architecture.md 2.10.1: state, `newSkills`, the `region_bound` obligation `in_traineeship`, the step of a removal, `p22Teaching`, the view fillers, the spawner's `Apply` executor and its checks), `skills_queue.go` (continuation, SKL: P7.1(a) completions, placement and ticks served; applications and the queue; the trainer's gain test) and `skills_report.go` (continuation, SKL: events, logs, snapshot rows, the checkpoint encoder and decoder, the `Warn` counter). The split is for the Lizard limits of N-243; both continuation files map to SKL and their tests carry its `Requirement:` lines (architecture.md 2.6; the two names are added to 2.6 and to the AGENTS.md layout in Phase 1).
- Appends to the catalogue and hook files (the table below); `testdata/golden/skills/` (module alone); the proof that `testdata/golden/off/` and `TestHashConfig_ReferencePinned` did not move; S06 and its four combinations; band keys STY-W04 and STY-D03 for the premium reports.

**Public identifiers.** `TraineeshipStarted`, `TraineeshipCompleted`, `TraineeshipEnded` (event.go, in the order of OBS 4.1); `DecideApply`; `CauseTraineeship`; the reasons `ReasonInTraineeship`, `ReasonNeedsWeightOne` and the `TraineeshipEnded` reasons `trainer_left`, `agent_left`, `role_removed` (names fixed in Phase 1); `RegionAgent.InTraineeship` and `Surplus`, `RoleView.QueueAhead`, `Trainers`, `MaxTrainees` (check which of `TrainingTicks` and `CapitalReturn` the view already has); `RoleRow.Places`, `PlacesTaken`, `Queue`, `Trainers`, `Held`. **No** `Config` field, `Modules` field, option, sentinel (FR-SKL-012 and 026 reuse `ErrOutOfBounds`), log key (the standard `agent`, `role`, `region`, `reason` serve), ledger reason or boundary account (the board is the existing `consumption` reason; a tool's wear is the existing `wear`).

**Leaves out, with the owner.**

| Item | Owner |
| --- | --- |
| The `train` right, a body's `MaxTrainees`, the members-only place, `not_member`, the held completion's **headroom** (class limit and body) | ORG and SET. SKL ships the hold logic with an unlimited-headroom hook, tested by unit test only until SET exists (Question 4) |
| A cohort as trainer or trainee, a cash premium, credit for it, voluntary early termination, journeyman years (spec 7, 11.4) | later versions |
| A per-class wage in the solver (N-071) and skilled hired labour (N-149) | VIA, EMP |
| The estate's anchor change to the price-window mean (11.4 item 4) | EST owner |
| Metrics: the rules' catalogue 5.3 has no traineeship instrument, and `otelobserver` is not touched | a later change to `otelobserver/` (Question 7) |
| Band **values** for STY-W04 and STY-D03 and the tuned tolerance of check D | STY and CLB; Stage 2 criterion 1 is unmet, so no value is set (Question 8) |
| A public API to apply or end a traineeship for an external agent beyond the consumer spawner's `Apply` (spec-integration 4.6) | INT, already merged; only re-checked |

## Decisions

Each is the spec's, a rule's or a note's; the plan decides nothing new except where Question 1 to 8 say so.

- **SK1.** Layout: `skills.go`, `skills_queue.go`, `skills_report.go`, `internal/skill`, additions to `internal/occupation`. No `Modules.Skills`, no hash id, no `modulesOn` name, no `Config` change (spec-config.md 3.7.2). Reverse: a pointer would move the hash ids and the pinned hash, which rule 2.10.5 forbids for an append that is off.
- **SK2.** State (SKL 9, N-244, architecture.md 2.10.1): `type skills struct` held as `w.skills *skills`, **nil unless some role has `TrainingTicks` above 0**, built in `construct.go` and guarded like a module's pointer. Per-agent state in the module's slices indexed by agent slot: `trainee []traineeRec{trainer AgentID, role RoleID, served, applied int64, live bool}`; `trainer []trainerRec{acc int64, trainees []AgentID}` with the trainees in ascending agent ID; `queue [][]applicant` per (region, role) cell (`applicant{agent, applied, served}`) in queue order; `warnRun []int64` per cell. Derived, never saved: the places, the throttle, the expected wait, the entry decision, the `teaching` flag of the tick (SKL 9 "Not saved"). The agent type gets no field (`TestAgent_HoldsNoMoneyQuantityOrRegion` keeps passing).
- **SK3.** Positions: `P2.2` leaves `notBuiltPositions` and `{"P2.2", "SKL", line}` is appended between P2.1 and P2.3 with the guarded call `if w.skills != nil { w.p22Teaching() }`. P7.1(a) is a guarded call at the head of `p71Entry`'s per-region step (rule 2.10.3: a sub-step is no position). The trainee's board is eaten at the trainee's own position of P6.1 (C10). The tick order of completion, placement, advance is SKL 4.2.1.
- **SK4.** Hooks (ADR 0010, architecture.md 2.10.1), each a guarded call with no logic: `agentRoleChanged`/`takeRole` (a trainer's switch is refused before it); `dropAgent` in `removeAgent` (step 2: a trainee's traineeship ends; a trainer's trainees return and requeue); the `in_traineeship` obligation (`w.bounds.add`, `construct.go`); the filler of `RegionAgent` and `RoleView` (`fillAgentView` is not the right fan-in: the spawner view is built in `spawner.go`, so the filler is a guarded call there); the `Work` seam for a trainee (`strategy_work.go`, `phase_production.go`: the trainer's eligible recipes, `principal` = the trainer); `consume` called with the trainer's account.
- **SK5.** Randomness: none. No site constant is added to `random.go`; FR-SKL-028 is a `Guard:` text search plus the determinism test.
- **SK6.** Events (SKL 4.7, 8.1; OBS 4.1, 4.4.8): three, per entity (a start, a completion, an end), identifiers and a reason only, constructed under `w.obs.supplied`. Order in a region's entry step: for a completion `RecipeAbandoned` (`role_changed`) if any, `TraineeshipCompleted`, the `RoleChanged` it causes (cause `traineeship`); then each place filled `TraineeshipStarted`; all before the population events and the plan. Order in a removal: per trainee in ascending ID `RecipeAbandoned` (`principal_left`) then `TraineeshipEnded` (`trainer_left`), then the trainer's own events.
- **SK7.** Reports: `RoleRow` gains `Places`, `PlacesTaken`, `Queue`, `Trainers`, `Held` appended at the end (counts: the off value is 0, so the golden rendering of `off/` is unchanged, rule 2.10.5). One `Warn` "no places open" per (region, role), bounded by the configuration, with a counterpart in the snapshot (`Queue` and `Places`), as observability-and-logging.md 8.1 (Counterpart) requires. `Debug`: "traineeship refused" (`weight`, `no_trainer`, `gain_negative`, `queue_too_long`) and nothing above `Warn`.
- **SK8.** Checkpoint: tag 10 keeps its four leading counts (trainees, trainers, queued rows, warn-run rows) so a world with no row writes the same bytes and the digests do not move; the rows follow in the order of SKL 9 (trainees, then trainers, then cells). The decoder refuses a trainee without its trainer, a trainer's edge missing on the other side (F7, FR-CHK-023), a role or region out of range, a row not above the previous, a served count above `TrainingTicks`, an accumulator outside `[0, 9 999]`, a queued agent that is a trainee.
- **SK9.** Completion uses `internal/skill.Complete(served, ticks, headroom, weight)`; until SET, the root passes "no limit". The held path (a trainee at `TrainingTicks` that stays, keeps working at `TraineeYield`, holds a place, counts in the trainer's load, completes at the first entry step with headroom) is a unit test of `internal/skill` and a root test with a planted state.
- **SK10.** The rumoured premium of SKL 2.5 is not built: the entry rule uses the region's own observed values (SKL 2.5 text; N-110). The estimate of `p_d * D_work` is the spawner's (SPN 6.2).
- **SK11.** The board is consumed from the trainer's account at the trainee's own position (C10), with the reason `consumption`; the trainee's receipt is in kind (FR-AGT-057, FR-WEL-030) and the same value is netted from the trainer's income; the trainee's welfare ratio is 10 000 while boarded (SKL 4.2).
- **SK12.** A trainee runs only trainee-eligible recipes of the trainer's role (FR-PRO-008, 033, 034): inputs from the trainer's account, outputs into it, yield `TraineeYield`, inputs not reduced; it is the executor, the trainer the principal; a trainee's job counts toward the trainer's income and not toward its own role's.
- **SK13.** Fixtures: `overlay/skill.toml` is used as it is (a fixture edit, with a reason, only if Phase 1 finds it does not load over the pack chosen in Phase 8). The module-alone golden is `testdata/golden/skills/` (events, positions, state; the form plan 2-5 fixed). The pack is chosen in Phase 8 by a probe (Question 5).
- **SK14.** Documentation: `doc.go` glossary gets the new terms before any code uses them (architecture.md 3.2): Trainer, Application (to a traineeship), Queue (of applicants), Place, Teaching tick, Teaching load, Expected wait, Board, Held (completion). "Trainee" and "Observed premium" exist. `TestGlossary_DeclaredTypesNamed` reads `doc.go`, so the names `TraineeshipStarted`, `TraineeshipCompleted`, `TraineeshipEnded` are added in the same phase.

## Requirement list and where each is verified

"Built" means the behaviour is in the tree; the plan adds a `Requirement:` line to an existing test that verifies it, or a thin test in Phase 8. "New" is built here.

| ID | State | Phase | Lowest-level test |
| --- | --- | --- | --- |
| FR-SKL-001, 002, 003 | built (PRO, LED, CFG) | 1 audit, 8 | existing PRO and LED tests get the line; `TestConfig_NoToolPriceField` (`Guard:`) |
| FR-SKL-004 | built, tested | 1 | `validate_rules_test.go` |
| FR-SKL-005 | tool half built; queue term new | 2, 4 | `TestApplyOK_QueueDiscountAndFloor`, `TestDecide_AppliesWhenReturnClears` |
| FR-SKL-006 | built | 1 | default and hash tests |
| FR-SKL-007 | built (spec-agents 7.3.5), verify | 1, 8 | `TestStrategy_ToolHolderDoesNotResell` |
| FR-SKL-008, 009 | built (EST) | 1, 8 | `TestScenario_ToolHolderDiesEstateSells` (end to end) |
| FR-SKL-010 | new | 4 | `TestApply_OnlyWayIntoTrainingRole`, `TestExecutor_RefusesSwitchIntoTrainingRole` |
| FR-SKL-011 | new | 5 | `TestTrainee_KeepsRoleAndRunsNoOwnRecipe`, `TestCompletion_KeepsMoneyAndGoods` |
| FR-SKL-012 | validation built; apply half new | 4 | `TestApply_RefusedNeedsWeightOne` |
| FR-SKL-013 | new | 5 | `TestBoard_ConsumedFromTrainerAccount` |
| FR-SKL-014 | new | 5 | `TestTraineeWork_EligibleOnlyExecutorPrincipal` |
| FR-SKL-015 | new | 5 | `TestAccumulate_*` (unit), `TestTeaching_BarsStartNotJob`, `TestEntryStep_OrderCompleteFillAdvance`, `TestCompletion_HeldWithoutHeadroom` |
| FR-SKL-016 | new | 2, 4 | `TestGain_*` (unit), `TestPlaceOpening_NegativeGainKeepsStarted` |
| FR-SKL-017 | new | 2, 4 | `TestPlaces_MaxTraineesAndQueueOrder` |
| FR-SKL-018 | new | 2 | `TestCTrain_FlooredAtZeroAndChecked` |
| FR-SKL-019 | new | 2 | `TestExpectedWait_CeilingAndNoTrainer` |
| FR-SKL-020 | new | 6 | `TestTrainerRemoved_TraineesRequeue` |
| FR-SKL-021 | new | 7 | `TestEvents_TraineeshipFields`, the order test of OBS 4.4.8 |
| FR-SKL-022 | new | 7 | `TestWarn_NoPlacesOpenTenthTick` |
| FR-SKL-023, 024, 025 | new | 8 | `TestSkillPremium_CheckC`, `TestSkillPremium_CheckA_ReportOnly`, `TestSkillPremium_CheckD` (see Question 8) |
| FR-SKL-026 | built | 1, 8 | bounds and registry tests get the line |
| FR-SKL-027 | new | 7 | `TestCheckpoint_SkillsRoundTrip`, `TestFork_Skills` |
| FR-SKL-028 | new | 2, 8 | `TestPackage_NoFloatNoRoot`, `Guard:` text search for a draw in the SKL files |
| NFR-SKL-001 | new | 7 | `TestAlloc_SkillsNoOp` (`testing.AllocsPerRun`) |
| NFR-SKL-002 | new | 7 | `TestSkills_WorkIsCounted` (enforcement), `BenchmarkTeachingAndQueue` (reports only) |

Also touched and re-checked, not owned: FR-SPN-009, 011, 013, 018, 020, 049, 054, 058; FR-AGT-057, 068; FR-WEL-030; FR-PRO-008, 018, 033, 034; FR-REG-048, 060; FR-OBS-018, 020, 021, 003; FR-CHK-023; FR-CFG-054.

## Phases

### Phase 1: the owed edits, the audit and the glossary (no code)

Goal: the specs, rules and notes are true before the code starts. Rule files get fresh tags. Edits owed (the supervisor applies the shared ones):

| # | File and section | Edit |
| --- | --- | --- |
| E1 | architecture.md 2.6, AGENTS.md Layout | add `skills_queue.go` and `skills_report.go` to the continuation files (SKL) and `skill/` to the `internal/` list (one package per module whose rules need arithmetic); fresh RULE-TAG. If Question 2 is "fold into occupation", only the continuation files |
| E2 | roadmap-v1.md Stage 3 table | the row for `skills.go` says `internal/skill` and "no `Modules` pointer, no hash id; tag 10"; status `plan written` |
| E3 | spec-skill-barriers.md 1, 12 | the "Layer: module" line says the barriers are core data (spec-config 3.7.2) |
| E4 | spec-skill-barriers.md 4.2, 4.6 | rulings for the gaps in Risks R3 to R7 (accumulator when the loads sum above 10 000, initial accumulator, requeue order, `role_removed`, the refusal of a consumer `Switch` or `Enter` into a training role) |
| E5 | spec-spawner.md 6.2, 6.3.0, 3.3 | N-305: re-read both; the training rule and the counts of 6.3.0 now hold with a queue; state whether an `Apply` counts toward the role cap and the outflow cap |
| E6 | spec-observability.md 4.1 | the field list of the three events and the closed set of `TraineeshipEnded` reasons; the snapshot columns |
| E7 | spec-world.md 6.4 | P2.2 and the sub-step P7.1(a) point at this plan's functions; no table change (the position exists) |
| E8 | notes | the verdicts of Risks; N-317 text is settled by Phase 3 |
| E9 | `doc.go` glossary | SK14, before any code uses a term |

Audit (read-only, recorded in the result paragraph): for each "built" row of the table above, name the test that verifies it. Where none exists, Phase 8 writes it. Check that `overlay/skill.toml` loads over `one` (the tests use it that way, but `crafter_a` is not a role of `one`), that the groups of `cook` have scale 1, and which packs give a skilled role at all.
Exit check: E1 to E9 applied or logged with an owner; `go test ./...` unchanged; `testdata/golden/` unchanged.

### Phase 2: `internal/skill` and the rule in `internal/occupation` (pure)

Goal: every number of SKL 4 as a function over primitives, no world. Files: `internal/skill/{doc,gain,teach,queue,complete,premium}.go`, `internal/occupation/{apply,refuse}.go` and tests.
- `Gain(output, upkeep, load, wTrainer int64) (g int64, ok bool)` (FR-SKL-016, checked); `Teach(acc, sumLoads int64) (acc2 int64, teaching bool)` (FR-SKL-015; one subtraction of 10 000 per tick per the ruling E4); `Open(trainers []Trainer{Held, Max}) int64`; `Less(a, b Applicant)` and `Requeue(a, b)` (FR-SKL-017, 020); `Wait(ahead, ticks, nmax int64) (q int64, ok bool)` (ceiling, no trainer gives not ok; FR-SKL-019); `CTrain(ticks, surplus int64) (int64, bool)` (FR-SKL-018); `Complete(served, ticks, headroom, weight int64, noLimit bool) Outcome` with `Done`, `Held`, `Serving`; `NextRun(run int64, blocked bool) (run2 int64, warn bool)` (FR-SKL-022: the tenth tick and every tenth after); `BreakEven(price, ctrain, capReturn, dwork, wUnskilled int64) (bp int64, ok bool)` and `Tolerance(excess, cvx2 int64) int64` (FR-SKL-024).
- `occupation.ApplyOK(p ApplyParams) (ok bool)` (FR-SKL-005): `p_d * D_work * (10 000 - CapitalReturn * q_w / TicksPerYear) >= (P + C_train) * CapitalReturn`, bracket floored at 0, no application when it is 0, both sides as exact 128-bit products (`internal/checked`); the refusals `InTraineeship` and `NeedsWeightOne` appended after the existing values with their `Facts` fields, each checked in the switch where `OnJourney` is.
- Tests (table-driven, got/want, with the arithmetic of the spec in the comment): `TestGain_*` (4.3 sign boundary, zero load, overflow), `TestTeach_*` (loads 1 000 x 3 over ten ticks gives three teaching ticks; accumulator never outside `[0, 9 999]`; the sum above 10 000), `TestOpen_*`, `TestLess_ApplicationTickThenID`, `TestRequeue_*`, `TestWait_*` (example of 4.9: 10 trainers, 3 places, 30 places, 60 ticks), `TestCTrain_*` (4.8: 2 555 x 500 = 1 277 500), `TestComplete_*` (held, done, serving, exact boundary), `TestNextRun_*`, `TestBreakEven_WorkedExample` (4.8: 12 733), `TestTolerance_*` (2.8, 3: 25% of the excess, 200 floor, twice the CV), `TestApplyOK_*` (FR-SKL-005 both forms, floor at 0, equal sides), `TestRefusals_OrderOfTests`, `TestPackage_NoFloatNoRoot` (FR-SKL-028), `FuzzApplyOK`, `FuzzTeach`.
- Exit check: `go test -race ./internal/skill/... ./internal/occupation/...`; coverage of each at least 90%; `gomu` in Docker (`--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS=4`, per file; survivors killed or recorded equivalent); `lizard -l go -C 10 -a 8 -L 100 -w` clean.

### Phase 3: the surface: state, construction, guard, types, checkpoint tag 10

Goal: a traineeship can exist (planted through an internal seam), block a relocation, be saved and restored, and the world does nothing else new. **First pull request ends here with Phases 1 and 2.** Files: `skills.go`, `skills_report.go` (the codec only), `world.go` (`skills *skills`), `construct.go` (`newSkills`, `bounds.add(inTraineeship, w.skills.holds)`), `fanin.go` (`dropAgent` call), `tick.go` (the P2.2 entry, an empty `p22Teaching`; the CCN note O7 of plan 2-14 applies), `event.go` (three events, `DecideApply`, `CauseTraineeship`, reasons), `spawner.go` (types only, appended), `snapshot.go` (`RoleRow` columns), `checkpoint_sections.go` (the F7 cross-check and the owner of tag 10), `poshash_internal_test.go`, `observe_internal_test.go`, `event_test.go`.
- `holds(a, skip)` is true for a trainer with trainees and for a trainee; the textual key is `in_traineeship:<agent>`; the obligation is registered only when `w.skills != nil` (FR-REG-060: "exactly those of the modules that are on").
- Tests: `TestRefusal_InTraineeshipBlocksRelocate` (FR-SKL guard of 4.6.1, FR-REG-060, FR-MIG-031's skip), `TestCheckpoint_SkillsRoundTrip` (a planted trainee, trainer, two queues, a warn run: bytes equal after restore; a world with none writes the skeleton bytes), `TestCheckpoint_SkillsRefusesCorrupt` (each refusal of SK8; F7), `TestWorld_SkillsNilWithoutTrainingRole`, `TestPositions_P22Listed` (the position table test), event catalogue and `TestConstants_Values` rows, `notBuiltEvents` loses three entries.
- Exit check: `gofmt`, `go vet`, `staticcheck`, `go test -short -race ./...`; `testdata/golden/off/`, `testdata/checkpoints/` digests and the pinned hash **unchanged**.

### Phase 4: the entry side: view, `Apply`, queue, places

Goal: the default spawner applies and a trainer takes a trainee. Files: `skills_queue.go`, `spawner.go` (`fillView` calls, the `Apply` executor in `phase_exit.go`), `internal/occupation`.
- The filler sets `QueueAhead`, `Trainers`, `MaxTrainees` per role and `Surplus` and `InTraineeship` per agent (the effective figures; the institutions amendments are ORG's). The default `Decide` emits `Apply` for a weight-1, non-trainee, non-external agent when `ApplyOK` holds; the executor refuses with `in_traineeship`, `needs_weight_one`, `on_journey` and the existing refusals, and rejects a `Switch` or `Enter` into a role with `TrainingTicks` above 0 (ruling E4). P7.1(a) step 2 fills places in queue order where a trainer has fewer than `MaxTrainees` and the gain test passes (4.3.1: it decides a new place only); the trainee is recorded, the trainer's list updated.
- Tests: `TestApply_QueueOrderApplicationTickThenID` (FR-SKL-017), `TestApply_RefusedNeedsWeightOne` (FR-SKL-012, FR-SPN-009), `TestApply_RefusedInTraineeship` (FR-SPN-011), `TestApply_OnlyWayIntoTrainingRole` (FR-SKL-010, FR-SPN-020), `TestPlaceOpening_NegativeGainKeepsStarted` (FR-SKL-016), `TestPlaces_NeverAboveMaxTrainees`, `TestDecide_AppliesWhenReturnClears` and the three negative cases (queue too long, no trainer, gain 0 bracket) (FR-SKL-005, 019), `TestDecide_ExternalNeverApplies` (SPN 5.2.3), `TestView_SkillFields` (SPN 3.2.2).
- Exit check: S06 smoke on the overlay for 200 ticks with `WithAudit`; every identity holds; `off/` unchanged.

### Phase 5: the living traineeship: board, work, teaching, completion

Goal: a trainee is boarded, works as the trainer's executor, the trainer teaches, and the course completes or is held. Files: `skills.go` (`p22Teaching`), `skills_queue.go` (the completion and the advance), `agent.go` (the `consume` call), `welfare.go` (the in-kind receipt and the trainer's netting, the trainee excluded from the target role), `strategy_work.go`, `phase_production.go` (the free-agent test gains "not teaching"; the trainee request), `phase_exit.go` (`takeRole` with `CauseTraineeship`).
- Tests: `TestBoard_ConsumedFromTrainerAccount` (FR-SKL-013, FR-AGT-068: the reason `consumption`, the trainer's use includes it), `TestBoard_WelfareRatioTenThousand` and `TestBoard_NettedFromTrainerIncome` (FR-WEL-030), `TestTraineeWork_EligibleOnlyExecutorPrincipal` (FR-SKL-014, FR-PRO-008, 033, 034: inputs from the trainer, outputs into it, yield `TraineeYield`, inputs not reduced), `TestTrainee_KeepsRoleAndRunsNoOwnRecipe` (FR-SKL-011), `TestTeaching_BarsStartNotJob` (FR-SKL-015: a job in progress continues, P2.1 first), `TestEntryStep_OrderCompleteFillAdvance` (SKL 4.2.1: a place filled at `s` serves ticks `s + 1` to `s + TrainingTicks` and completes at `s + TrainingTicks`), `TestCompletion_KeepsMoneyAndGoods` (FR-SKL-011; goods outside the new role's tradable set to the estate), `TestCompletion_HeldWithoutHeadroom` (planted limit; SK9), `TestCompletion_AbandonsJobRoleChanged` (the order of OBS 4.4.8), `TestObservedPremium_ExcludesTrainees` (WEL 4.3).
- Exit check: `TestInvariants` with the overlay (goods and money conservation, no negative balance) over `scenarioSeedCount()` seeds; `off/` unchanged.

### Phase 6: interruptions, removal, relocation, weight

Goal: every way a traineeship ends or is refused. Files: `skills.go` (`dropAgent` step), `fanin.go`, `phase_exit.go`.
- Tests: `TestTrainerRemoved_TraineesRequeue` (FR-SKL-020: ticks served kept, role kept, queue order, events in ascending trainee ID, `RecipeAbandoned` then `TraineeshipEnded`), `TestTraineeRemoved_EndsAndGoesToEstate` (AGT 3.6 step 2; `agent_left`), `TestTrainerSwitchRefused` (`in_traineeship`), `TestRemovalNeverRefusedForTraineeship` (FR-SPN-011 second half), `TestRelocate_RefusedWhileTraineeship` and `TestRelocate_AllowedAfterEnd` (FR-REG-060, FR-REG-048: state unchanged by a relocation), `TestMigration_SkipsTrainee` (FR-MIG, already built: re-check), `TestExternalTrainee_Directed` (SPN 5.2.3, INT 4.6: a consumer `Apply`; exempt from the ladder).
- Exit check: `TestCheckpoint_*`/`TestFork_*` suites still pass; `off/` unchanged.

### Phase 7: events, logs, snapshot, the allocation limits and the fork test

Goal: observability and the checkpoint proof. **Second pull request holds Phases 4 to 9** (or splits at this phase if reviewers want, Question 3). Files: `skills_report.go`, `snapshot.go`, `observe_events_test.go`, `allocation_test.go`.
- Tests: `TestEvents_TraineeshipFields` (FR-SKL-021, FR-OBS-018: identifiers and reason only, no map, region of the agent, after the transition), `TestEvents_OrderInEntryStepAndRemoval` (FR-OBS-020, 021), `TestWarn_NoPlacesOpenTenthTick` (FR-SKL-022: per region and role, never per agent, the tenth tick and every tenth after, counterpart in the snapshot), `TestLogs_SkillsDebugOnly` (no `Info` in the tick, constant messages, `Enabled` guards), `TestSnapshot_SkillRows` (SKL 8.2: places, taken, queue, trainers, held), `TestCardinality_*` (reason sets closed), `TestAlloc_SkillsNoOp` (NFR-SKL-001: entry decision, accumulator and queue operations allocate nothing beyond world scratch with the default logger and observer), `TestSkills_WorkIsCounted` (NFR-SKL-002: operations proportional to applicants and trainers, not weights), `TestNonInterference_Skills` (FR-OBS-023), `TestFork_Skills` (FR-SKL-027, FR-CHK-050: fork at several ticks during a course, a queue, a held completion; same traineeships, queues and teaching ticks as an uninterrupted run), `BenchmarkTeachingAndQueue`.
- Exit check: the staged catalogue and cardinality tests pass with the entries deleted; `TestEvents_CatalogueStaged`, `TestConstants_Values` rows for the new reasons and cause.

### Phase 8: overlay, golden, scenarios, premium checks

Goal: the module is proved end to end, tool and training together.
- Pack: chosen by a probe over seeds 1 to 4 (Question 5): the first of `one`, `five`, `payroll`, `two` with `overlay/skill.toml` (and, for S06, `overlay/imbalance-cook.toml`) in which at least one traineeship completes by tick 600; the probe is a test that fails if the golden would record no completion, so the golden exercises the module. `overlay/skill.toml` edits, if any, carry a reason (testing.md 7.2).
- Golden: `testdata/golden/skills/` (counts, events, positions, the per-position state hash with P2.2 and the entry step) and the proof that `testdata/golden/off/` and `TestHashConfig_ReferencePinned` are unchanged. `testdata/golden/all/` moves only if the all-on pack takes the overlay; if it does, the PR explains it (testing.md 7.2), if not it is untouched.
- Scenarios: S06 and its four combinations (`overlay/skill` with and without `overlay/notool`, the baseline of FR-SKL-023), `five` + skill for the second eligible recipe (C13: `prepare_basic` pays, `finish_a_basic` does not, so no places open for it and the `Warn` fires), the throttle visible at `TrainingTicks > n_max / a` (4.9), the tool-holder death scenario (FR-SKL-008, 009, 007). Outcomes are recorded in `expected/<pack>.scenarios.txt`, pass or not (the Stage 1 "Done when" form), because criterion 1 is unmet.
- Premium checks (FR-SKL-023 to 025; Question 8): check C asserted where the pack survives to the measure (premium above 10 000 with a tool and rising with its price, within the tolerance of 10 000 without); check A reported by the test log, never fails; check D asserted only on a purpose-built small fixture whose economy is stable (the tolerance `max(25% of the excess, 200 bp, 2 CV)` read from `testdata/chains/bands/`), reported on the reference packs. Band keys `s06.*` for STY-W04 and STY-D03 are declared with a unit and no value, as plan 2-14 did for S61 to S66.
- Verification tests for the built halves: `TestStrategy_ToolHolderDoesNotResell`, `TestScenario_ToolHolderDiesEstateSells`, `TestConfig_NoToolPriceField`, `TestConfigOnly_SkillNeedsOnlyConfiguration` (architecture.md 9.4: a training role by configuration alone, `testdata/chains/second/`), `TestDeterminism_Skills` (the cases of `detCases(true)`, split `Run` calls, with and without observer), `TestInvariants` row for the skills world.
- Exit check: the module runs the overlay for the length REF fixes with `WithAudit`; the new tests are in the quick pattern (`TestScenario`, `TestDeterminism`, `TestGolden`, `TestInvariants`, `TestFork`, `TestStylised` prefixes), so arm64 runs them.

### Phase 9: the exit gate

Gates run by hand on the dev host (testing.md 9.1) and recorded in the PR description: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`, `go test -short -race ./...`, gate R, gate F with `scripts/ci/coverage-check.sh` (every scope at its minimum, coverage not lower), `go mod tidy` no diff, in the root and in `otelobserver/` and each `examples/` module (the examples build against the public API; `examples/realm` has no training role, so no change is expected). `gomu` per new file in Docker with the caps. A deliberate-break pass (testing.md 9.3: flip the gain sign, drop the `+ C_train`, drop the requeue, swap two phases of the entry step) shows the tests fail. Close: roadmap row set to `merged (PR #...)`, notes N-264, N-305, N-317, N-193 (as far as built), N-090 and N-110 settled or updated, `spec-roadmap.md` and `documentation_index.md` updated, plan audited with plan-audit.

## Pull request split

Two pull requests, in order, each leaving `off/` and the pinned hash untouched:
- **PR A (Phases 1 to 3): edits, `internal/skill`, `internal/occupation` additions, state, guard, types, checkpoint tag 10.** It adds no behaviour: the default spawner emits no `Apply` yet, so no world changes. It lands the catalogue appends early, which is where Stage 3 modules collide (see below). Size estimate: 2 500 lines including tests.
- **PR B (Phases 4 to 9): the behaviour, the reports, the proof.** Size estimate: 5 000 lines of code and tests and one golden of tens of thousands of lines (migration's was 22 000). If review wants it, split after Phase 6 (the traineeship works; the reports and the proof follow). A single PR for the whole row, as plan 2-14 did, is the alternative (Question 3).

**Minimal slice the roadmap implies.** The row "Needs PRO, WEL" only, so the smallest slice needs no SET, ORG, EMP or STO: Phases 1 to 6 with the events of Phase 7 (apply, queue, place, board, trainee work, teaching tick, completion, trainer and trainee removal, the `in_traineeship` guard, tag 10, three events). It leaves out the snapshot columns, the `Warn`, the premium checks A and D, the held path with real headroom and `golden/all` movement. That slice makes a training role enterable, which is all the tool-only slice of Stage 1 lacked (N-264).

## Catalogue and shared-file appends (merge-conflict forecast)

Other Stage 3 modules append to the same files. Appends are in module order and never inserted (rule 2.10.5), so a conflict is a textual one at the end of a list; the files marked * are the likely ones.

| File | SKL append | Also touched by | Note |
| --- | --- | --- | --- |
| `event.go`* | three events, `DecideApply`, `CauseTraineeship`, `ReasonInTraineeship`, `ReasonNeedsWeightOne`, `TraineeshipEnded` reasons | every Stage 3 module (events); SET, STO, EMP, ORG (reasons) | land PR A first |
| `event_test.go`* | three `notBuiltEvents` entries deleted, `TestConstants_Values` rows | all | |
| `snapshot.go`* | five `RoleRow` columns at the end | SET, STO, EMP, SAV, ORG | appended after the columns of earlier modules |
| `tick.go`* | `{"P2.2","SKL",line}`, one guarded call, `notBuiltPositions` entry deleted | EMP (P2.5a, P2.8b), CAP, TEC, STO, SAV, RSK | the CCN helper split of plan 2-14 O7 may be needed again |
| `spawner.go`, `internal/occupation/*` | `DecideApply`, view fields, refusals, `ApplyOK` | SET (`Headroom`, `NoLimit`), POP (built), ORG | N-268 lists the order |
| `fanin.go` | one `dropAgent` call | EMP, LND, CRD, INS, CAP, STO, ORG | the order of steps 2a to 2g |
| `construct.go`* | `newSkills` and `bounds.add` | every module | |
| `world.go` | `skills *skills` | every module | |
| `phase_exit.go` | the P7.1(a) call, `takeRole` cause | SET (P7.2a is elsewhere), ORG (membership), POP | |
| `phase_production.go`, `strategy_work.go` | the "not teaching" test and the trainee request | EMP (hired request), SUB (built), CAP | |
| `agent.go`, `welfare.go` | the trainer's account in `consume`; the in-kind receipt | EMP, SAV | |
| `checkpoint_save.go`, `checkpoint_restore.go`, `checkpoint_sections.go` | tag 10 body, F7 cross-check | all modules with a tag | the skeleton prefix keeps the bytes (SK8) |
| `doc.go`* | the terms of SK14 | all | |
| `architecture.md` 2.6, `AGENTS.md` Layout*, rule files with a RULE-TAG* | continuation files, `internal/skill`, a fresh tag | all modules that add a file | the tag line always conflicts: the supervisor applies these edits (plan-2-14 E7) |
| `registry_test.go` | none (no field is new) | | rows exist and are not in `notBuilt` |
| `hash.go`, `config.go`, `bounds.go`, `defaults.go`, `errors.go`, `option.go`, `random.go`, `ledger.go`, `internal/logkeys` | **none** | | SKL is the one Stage 3 module with no config surface |
| `testdata/chains/overlay/skill.toml`, `bands/*.toml`, `expected/*.scenarios.txt`, `testdata/golden/skills/` | new golden; band keys declared | STY, CLB | |

## Cross-plan contract

Consumes (identifiers as the tree has them at f556398; re-check at implementation):
- `internal/occupation` (`CapitalOK`, `Facts`, `Refusal` values, the `Decide` entry), `DecisionKind`, `Decision`, `RegionAgent`, `RoleView` (`spawner.go`) <- Stage 1 and plan 2-14
- `regionBounds.add(name, held)`, `relocationRefusal`, `RegionBoundError` <- plan 2-3
- `removeAgent`, `dropAgent`, `agentRoleChanged`, `takeRole`, `moveAgent` <- Stage 1 and 2-3
- `consume(a, from, ...)` (`agent.go`) <- Stage 1 (plan 1-7)
- the job fields `principal`, `startReq`, `abandonJob` and the reasons `principal_left`, `role_changed` <- Stage 1 (plan 1-5)
- `internal/production.Effective(..., trainee, ...)` <- Stage 1
- `PremiumBP` and the role summary (`welfare.go`) <- Stage 1 (plan 1-6)
- `checkpointSections`, the tag-10 skeleton, `forkAtN`, `forkCase` <- plan 2-2
- the estate's listing and markdown (FR-EST-008, 009) <- Stage 1 (plan 1-8)
- `testdata/chains/overlay/skill.toml`, `lines.toml` and `five.toml` eligible recipes, the module-alone golden form <- plans 2-5 and 2-8

Provides:
- `skills` state, `p22Teaching`, the `in_traineeship` obligation, `(*skills).isTrainee(a)`, `(*skills).trainerOf(a)` -> EMP (`Hireable` excludes a trainee, AGT 7.x), MIG (re-check), INT
- `placesOf(region, role)` and `trainersOf(region, role)` as unexported hooks with the institutions amendment slot -> ORG (`train` right, body `MaxTrainees`, `not_member`)
- `completionHeadroom(region, role)` hook returning "no limit" -> SET (`Headroom`), ORG
- `DecideApply`, `CauseTraineeship`, the view fields -> SET, ORG, consumers' spawners
- `TraineeshipStarted`, `TraineeshipCompleted`, `TraineeshipEnded`, `RoleRow` columns -> IND (a possible traineeship preset), otelobserver (later)
- `overlay/skill.toml` and the S06 combinations, `testdata/golden/skills/` -> STY (STY-W04, STY-D03), plan 24-style five-sized rows

## Dependencies on other Stage 3 modules

SKL needs only PRO and WEL (roadmap). Modules that need SKL: **ORG** (`spec-institutions` 5.7 amends places, trainers, `MaxTrainees`; ORG is listed "POL, SKL, SET"). Soft links, none blocking: **SET** (the headroom of the held completion, `Role.Needs`), **EMP** (`Hireable` is "not a trainee"; N-149), **STO** (a completion's role change moves goods above the new role's room, N-138), **TEC** (check whether a trainee's executions count as experience: not stated in `spec-technology.md`, a note to add), **CAP** (builds and repairs are not trainee-eligible, FR-PRO-033), **INT** (consumer `Apply` for an external agent, already merged), **CRD** (no premium loan in version 1). Build order is therefore independent: SKL can merge before or after SET, EMP and STO; only ORG must follow it.

## Testing

Testing entry (testing.md 11) for the PRs: the requirements and tests are in the table above; the lowest level is the `internal/skill` and `internal/occupation` unit test wherever the rule is pure, the root integration tests of Phases 3 to 7 for what needs a world, and the S06 scenarios and the module-alone golden at the top. Golden files: `off/` unchanged (`TestGolden_OneOff`, `TestGolden_OneEvents`, `TestHashConfig_ReferencePinned`), `testdata/checkpoints/` unchanged; the new `testdata/golden/skills/` is recorded once and never regenerated to absorb a failure; `all/` moves only if the all-on pack takes the overlay, with the testing.md 7.2 explanation. Registry and catalogue: `notBuiltEvents` loses three entries and `notBuiltPositions` loses `P2.2`; no `notBuilt` entry changes. Fuzz: `FuzzApplyOK`, `FuzzTeach`; the loader and validators are not changed, so no new corpus. The PR descriptions record the exit markers of the gates and the deliberate-break pass.

## Observability

Observability entry (observability-and-logging.md 9).
- **Events:** `TraineeshipStarted`, `TraineeshipCompleted`, `TraineeshipEnded`, per entity transition, plain data, no map, constructed only under `w.obs.supplied`, after the transition, in the order of SK6 (rules 4.1, 4.3, 4.5).
- **Warn:** one, "no places open", per (region, role), at the tenth consecutive tick with applicants and no place because the gain is negative and every tenth after; bounded by the configuration (rule 3.2.2); counterpart in the snapshot (`Queue`, `Places`); keys `region`, `role`, `reason`. No `Info`, no `Error`.
- **Debug:** "traineeship refused" (`agent`, `role`, `reason` in `weight`, `no_trainer`, `gain_negative`, `queue_too_long`), guarded by `Enabled`; `not_member` is ORG's.
- **Metrics:** none defined in the core. The rules' catalogue 5.3 has no traineeship instrument; the events carry bounded identifiers (role, region, reason) so instruments can be added later without a core change (Question 7). No new metric attribute, so rule 5.4.2 adds no series.
- **Non-interference:** nothing draws; a run with and without observer and logger is identical (Phase 7 test).

## Risks and open items

| # | Item | Owner |
| --- | --- | --- |
| R1 | Pack `one` and every reference pack but `payroll` go extinct (Stage 2 closing record). A module-alone golden on a collapsing pack may record no traineeship. Phase 8 makes the golden fail to record if no course completes and picks the pack by probe | SKL owner |
| R2 | The queue-discounted rule changes 6.2 and 6.3.0 of the spawner (N-305): whether an `Apply` counts toward the role cap and the outflow cap, and how a sleeping applicant interacts with the review rotation, are spec gaps. Phase 1 E5 rules; a wrong ruling would move spawner behaviour, though only where a training role exists | SPN, SKL owners |
| R3 | SKL 4.2 says the accumulator "reaches 10 000, subtracts 10 000": with three trainees at 10 000 each the sum is 30 000 a tick. Proposed: one subtraction a tick, accumulator kept below 10 000 only when the sum is below it; rule it in E4 | SKL owner |
| R4 | The initial accumulator and whether it survives a change of trainees: flagged "not stated" by the spec; proposed 0 at the trainer's first trainee, kept while any trainee remains and reset to 0 when the last leaves | SKL owner |
| R5 | FR-SKL-020 requeues trainees "ordered by ticks served and then application tick": direction not stated. Proposed ticks served descending, then application tick ascending | SKL owner |
| R6 | `TraineeshipEnded` reason `role_removed` (8.1) has no stated cause: roles are data and are not removed at runtime. Proposed: declared, unreachable until a later module can remove a role, and listed in the constants test | SKL owner |
| R7 | Which refusal a consumer's `Switch` or `Enter` into a training role gets is not in SPN 4; FR-SPN-020 only says it "MUST be entered only by `Apply`". Proposed: a new refusal reason appended after the others (working name `needs_traineeship`, named in E4); reusing `closed` would mislead. Needs a ruling | SPN owner |
| R8 | A trainee's executions and the trainer's recipe profit: whether a trainee's output enters the trainer's observed wage is WEL's (FR-WEL-030 nets the board). Phase 5 test pins the behaviour of WEL 6.3 as written | WEL owner |
| R9 | `positions` and `positionsProduce` may be at the CCN limit (plan 2-14 O7); one call is added | tick.go owner |
| R10 | Premium check D needs a stable economy to measure the coefficient of variation; none of the reference packs is stable. See Question 8 | STY owner |
| R11 | The per-agent `Surplus` (`S_u`, observed income per tick above upkeep) needs the welfare window of the agent; for a new household with no window it is 0 and `C_train` is 0, so the first applicants see a cheap course. Test and note | SKL, WEL owners |
| R12 | Interaction with MIG: a trainee on a journey is impossible (the guard) and a migrant cannot apply (`on_journey`); the reverse order, an agent that applies and is then pushed, is refused by `in_traineeship` at the `Switch`. Tested in Phase 6 | MIG, SKL owners |

Note verdicts (text only; the supervisor edits `doc/notes.md`): N-039 leave open (BEL's rule; the wording of SKL 2.4 is already rewritten). N-071 leave open (VIA; the premium uses the observed wage of each role, so no per-class wage is needed here). N-090 apply for SKL (the obligation is registered in Phase 3) and keep the other legs. N-110 apply for SKL (SK10: no rumoured premium), then remove the SKL leg. N-138 leave open (STO). N-149 leave open (EMP). N-193 leave open until SET supplies headroom (the held path is built and unit-tested). N-244 standing rule, kept. N-264 apply (Phases 2 to 4), remove at the gate. N-268 apply for SKL, keep for SET. N-305 apply (Phase 1 and 4), remove. N-317 apply for tag 10 (Phase 3), remove the tag-10 leg.

## Questions only the owner can answer

1. **Is SKL core, with no `Modules` pointer, no hash id and "on" meaning some role has `TrainingTicks` above 0?** That is what `spec-config.md` 3.7.2 and architecture.md 2.6 say; the roadmap's instruction to take a hash id and tag from the `Modules` row would otherwise need a new pointer, which moves the hash ids. Recommended: yes.
2. **A new `internal/skill` package (needs a line in architecture.md 2.6 and AGENTS.md, a fresh RULE-TAG, and two continuation files), or everything in `internal/occupation`?** Recommended: the new package, as rule 2.10.1 puts a module's arithmetic under `internal/<module>`.
3. **One PR for the row (as MIG) or two (A: edits, pure rules, state; B: behaviour and proof) or three?** Recommended: two, so the catalogue appends land early and the Stage 3 modules built in parallel conflict less.
4. **Is it acceptable that the held completion has no real headroom until SET exists** (the hook returns "no limit", the held path is unit-tested only), or should SET be built first?
5. **Which pack carries the module-alone golden:** the spec's `one` (which collapses), `five`, or `payroll` (the only pack that keeps households, with the payroll module on)? Recommended: whichever of `one`, `five`, `payroll` passes the Phase 8 probe first, in that order.
6. **Rulings on the spec gaps R3 to R7** (accumulator with several trainees, initial accumulator, requeue direction, `role_removed`, the refusal of a `Switch` into a training role): accept the proposals, or decide otherwise?
7. **Metrics:** the rules' catalogue has no traineeship instrument; add the rows and the `otelobserver` instruments in this change, or defer to a note? Recommended: defer.
8. **Premium checks:** with Stage 2 criterion 1 unmet, assert only check C on the packs, report A, and assert D on a purpose-built small stable fixture? Or leave C and D as reports until the reference chain is stable? Recommended: assert C and D on the fixture, report on the packs.
