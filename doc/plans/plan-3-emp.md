# Plan: Stage 3, EMP, the hire form: `employment.go`, `internal/employment` and the presets of `indicator/`

**Status: plan written (2026-11-18), stage 1 of the Stage 3 work (requirements and plan, no code). Written on branch `s3-emp-1`, cut from the integration branch `phase-3` (= `15bcf18`, which holds the merged plans and PRs of SKL, POL, STO, TEC, SET, CRD (PR 1 to 3) and LND (PR 1 to 4)). The owner questions Q1 to Q9 are delegated to the supervisor: each carries a recommendation and the plan treats it as accepted. Format model: `plan-3-lnd.md`, `plan-3-set.md` and `plan-3-crd.md`. The spec is `spec-employment.md` sections 3 to 17 (the hire form, FR-EMP-001 to 063, NFR-EMP-001 to 005), extending the payroll form of section 24 that Stage 2 built (ADR 0011).**

Rule tags (current at `15bcf18`): architecture=5c0416f502ba5a59f7b2ae08, observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367. Read for this plan: all of `architecture.md` and `testing.md`; of `observability-and-logging.md` sections 1 to 4 (the event catalogue in full), 5.3 rows and 5.4, and 6 to 11 (the metric tables of 5.3.1 were read by grep for the employment rows only). Phase 1 requires the full read of all three before any edit, as AGENTS.md says.

Notes tag (`doc/notes.md`, tail at `15bcf18`): f85c4b2d7217a479ec7c16dc. This plan writes no spec, so the notes rule of AGENTS.md does not bind it. Read: the index, the notes tagged EMP (N-144, N-147, N-187, N-190, N-191, N-194, N-327, N-352, N-355, N-356) and the ALL notes the model plans cite (N-126, N-128, N-217, N-243, N-345, N-351, N-353). `doc/notes.md` was **not** read in full: Phase 1 (E6) does that and quotes the tag, because Phase 1 edits specs.

Specs read: `spec-employment.md` lines 1 to 1030 (sections 1 to 10), 1100 to 1360 (12 to 16 and the start of 17) and 1540 to 1923 (20 to 24). Not read in this pass: section 11 (the closed form, only FR-EMP-062 depends on it), the register entries of 17 after EMP-D2, and sections 18 and 19 (the models and the examples). Also read: ADR 0011, `spec-indicators.md` rows 22 to 24 of 9.1 and FR-IND-056 to 058, `spec-checkpoint.md` row 24, and `spec-reference-chain.md` 7.12 (S97 to S101).

## What is already built (grep of `15bcf18`)

- **The payroll form is in the tree.** `employment.go` (266 lines: `employment` state, `validateEmployment`, `p41bPayroll`, `payRegion`, `payRow`, the Debug line, `PayrollPaid`), `internal/employment` (`Budget`, `Split`, `Payer`, `Worker`, `Transfer`), `Employment struct{}` and `Employer{PayrollBP}` in `config.go`, `Role.Hireable` (hash id 23) and `Role.Employer` (id 24), `ErrEmploymentShape`, `DefaultEmployer`, the loader keys `hireable` and `[role.<id>.employer] payroll_bp`, the ledger reason `wage_pay` (agent to agent, code 23), the welfare feeds `addEarned` and `addWagePaid`, the pooled-wage exclusion (`isPayer`), `testdata/chains/overlay/employment.toml` and `testdata/golden/employment/` (pack `one-demand`). `hash.go:503` writes `Modules.Employment` as id 13 with no inner pair.
- **Seams left for the hire form.** `notBuiltPositions` lists `P2.5a` and `P2.8b` (`tick.go:90`). `notBuiltSections` lists `{24, "Employment", "Employment"}` (`checkpoint_sections.go`). `registry_test.go:51` lists `Employment.MaxOpen`, `Employer.Recipes`, `Employer.MaxHires`, `Employer.WorkerShareBP`, `Employer.HireYieldBP` in `notBuilt`. `event_test.go:106` lists `HireFormed`, `HireEnded`, `LabourRound` in `notBuiltEvents`. N-355 owes the hire form: `MaxOpen`, the hire block with the next free ids, the pairing escrow to agent for `wage_pay`, and no reuse of `PayrollBP`.
- **The trainee mechanism the hire generalises.** `job.principal` exists (`production.go:202`); `startReq` carries `principal` and `exec`; `traineeRequest` (`skills.go:305`) and `residueOwner` (`skills.go:322`) already move inputs, tools, output and residues to a principal that is not the executor. `effectiveFor` (`production.go:173`) takes a `trainee` basis-point factor. LND (merged) carries the principal through the ration and the charge (`tenure_ration.go`), and has `tenure.leastRent(principal, cell, executions, draw)` (`tenure_ration.go:483`) and `internal/tenure.LeastRent` for formation check 8.
- **Escrow accounts exist for MER.** `ledger.KindEscrow`, reasons `EscrowIn/Out/Refund` (codes 19 to 21), the free list derived in `contract_settle.go:261` and `:389`, the account opened at `contract.go:1294`, the restore check at `checkpoint_restore.go:398` (`v.Kind == KindEscrow && w.mer != nil && w.mer.on`). The `region_bound` registry (`regions_relocate.go`, `construct.go:209`, `openContract` in `contract_settle.go:25`) is the pattern for `open_hire`.
- **What is not there.** No `LabourOffer`, `HireRequest`, `HireID`, `HireFormed`, `HireEnded`, `LabourRound`; no `wage_escrow` or `wage_refund`; no P2.5a or P2.8b; no `Employed`/`Employing` row fields; no `indicator/` preset 22 to 24 (`indicator/tally_observe.go` has no `LabourRound` case); no hire fields in `AgentView`; no `open_hire`; no `ErrEmployerWeight`. CAP, ORG and SAV are not merged, so formation checks 2, 5, 9 and 10 have no host yet.
- **Lizard state (`lizard -l go -C 8 -a 8 -L 80`, measured on `15bcf18`).** `positionsProduce` (`tick.go:154`) is at CCN 10 and `positionsTail` at 10: P2.5a and P2.8b need a refactor-first split. `fillAgentView` (`fanin.go:15`) is at 10. `removeAgent` is now 8 (refactored by STO). `limitList` 9, `start` 9, `complete` 9, `classWithoutIncome` 9, `newDemandIndex` 9, `auditModules` 10, `checkpointCross` 16. The local `lizard` run does not list `startJob`, `p27Start` or `workAgent` (a parse gap, same as LND's); LND's plan measured `startJob` at CCN 10 and LND PR 4 has added hooks since, so Phase 1 re-measures them by hand.

## Scope

This plan builds the **hire form** of the employment module: the labour offer and the vacancy as value fields of `JobRequest`, the labour round at P2.5a (admission, order, first-fit matching, the wage, formation, the escrow), the hired job (start, a wage per working tick, completion, interruptions), the hire record and its checkpoint section (tag 24), the removal step 2c, the `open_hire` refusal, the default strategy's offer and vacancies, the view group, the welfare and observed-wage rules for a hired tick, the aggregate `LabourRound` at P2.8b, the logs, and the indicator presets `Unemployment`, `WageBill` and `Employment`. It is off unless `Config.Modules.Employment` is non-nil with the hire form selected (D1); the payroll form keeps working unchanged.

| In scope | Where |
| --- | --- |
| Configuration: the form selector and `MaxOpen` on `Employment`; `Recipes`, `MaxHires`, `WorkerShareBP`, `HireYieldBP` on `Employer`; bounds `MaxEstablishment`, `MaxHireRecords`; sentinel `ErrEmployerWeight`; the nine warnings of 14.2; hash; loader | `config.go`, `bounds.go`, `defaults.go`, `errors.go`, `validate.go`, `hash.go`, `config/mapping.go`, `employment_validate.go` |
| Pure arithmetic: the wage, the bill, the order of offers and vacancies, first-fit with fit and closing, the positions bound, the closed form of 11.2 | `internal/employment/` (new files beside the payroll ones) |
| The module: the hire record and counter, the round, formation, the release, completion, interruptions, the refusal, the audit, the snapshot rows, the codec | `employment.go` (state), `employment_round.go`, `employment_hire.go`, `employment_report.go` |
| Hooks into production: the hired request, the factor `HireYieldBP` instead of `trainee`, the principal's inputs, tools and output, the cut start and its refund, wage release at P2.1 and P2.7, completion | `phase_production.go`, `production.go`, `skills.go` pattern (`traineeRequest`) |
| The strategy surface: `JobRequest.Offer` and `.Hire`, `LabourOffer`, `HireRequest`, the view group, `DefaultStrategy.Work` offer and vacancies, the planned hires in the targets (9.2) | `strategy.go`, `strategy_view.go`, `employment_strategy.go`, `strategy_default.go` |
| Welfare: `Y_t`, `V_t`, the hired working tick, the clause of WEL 4.3 not applying to a `Hireable` role, the principal's wear and floor, the pooled wage (already excludes payer roles) | `welfare.go`, `employment_hire.go` |
| Demand and subsistence: `class_without_income` for a `Hireable` role in a world with an employer; `in_own_use` refusal | `demand.go`, `subsistence.go` |
| Ledger: reasons `wage_escrow`, `wage_refund`; the second pairing of `wage_pay` (Escrow to agent); shared Escrow free list with MER | `internal/ledger/registry.go`, `ledger.go`, `event.go`, `contract_settle.go`, `contract.go`, `checkpoint_restore.go` |
| Events `HireFormed`, `HireEnded`, `LabourRound` (with `LabourRow`, `VacancyRow`), the reasons and causes, positions P2.5a and P2.8b, snapshot `Hires []HireRow` and `RoleRow.Employed`, `.Employing`, the key `vacancies` | `event.go`, `tick.go`, `snapshot.go`, `internal/logkeys` |
| Removal 2c, role change, move, weight fall; `open_hire`; integration `AgentState` hire fields | `fanin.go`, `phase_exit.go`, `regions_relocate.go`, `construct.go`, `integration.go` |
| Checkpoint section tag 24 | `checkpoint_sections.go`, `checkpoint_save.go`, `checkpoint_restore.go` |
| Presets 22 to 24 of `indicator/` (FR-IND-056 to 058): the `LabourRound` tally case, rows, the three presets, tests | `indicator/tally_observe.go`, `indicator/people.go` or a new `indicator/labour.go` |
| `testdata/chains/overlay/employment-hire.toml`, `testdata/golden/employment_hire/` (new files), scenarios S97 to S101 | `testdata/`, tests |

| Out of scope | Owner |
| --- | --- |
| Pay in kind (`wage_in_kind` stays reserved, N-147); a cohort employer; standing employment; a wage that responds to scarcity; skilled hired labour (spec open questions 1, 2, 3, 5, 9) | later notes; EMP-D1 refuses a bidding wage |
| A wage control and a legal clamp (N-190, N-194), a closed shop (`produce` right, `Who`) | ORG: formation check 2 `not_permitted` is a named slot, built when ORG lands |
| Facility checks 5, 9, 10 (`no_facility`, build reasons, `facility_cap`) and `k_cap` in the cut start | CAP: the `LabourRound` fields are declared now and stay 0 until CAP fills them (D7) |
| The outlet of an employer's profit (`employer_no_outlet` cure: a `Saver` block) | SAV, DEM; this plan builds the warning only |
| A tax on `wage_pay` (N-187) | FIS: the wage transfer is one function in `employment_hire.go`, so a hook is one call later |
| The payroll-aware viability duals for a hire world (spec open question 7) | VIA: duals stay payroll-only; a hire world's solver row is not changed (D9) |
| The metrics `economysim.hire.*`, `labour.unmatched`, `vacancy.unfilled` and their adapter | `otelobserver/` (separate PR; Q5) |
| `IncomeLast` and the other income fields of N-130 | IND, unchanged: the wage bill is measurable from `LabourRound.Paid` |
| Preset 25 `Losses` (FR-IND-059) | RSK and STO |

## Decisions

- **D1. One form per world, chosen by a field.** `Employment` gains `Hire bool` (hash id 1, written only when true) and `MaxOpen int64` (hash id 2, written only with `Hire`, default 1 000 written by `canonical` when `Hire` is set, as N-345 allows for a field the module owns). `Hire` false is the payroll form byte for byte: `golden/employment/`, `golden/off/` and the pinned hash do not move. In the hire form P4.1b does not run and the `Employer` block carries the hire fields; `PayrollBP` must be 0 there (`ErrEmploymentShape`), and in the payroll form the hire fields must be empty or 0 (`ErrEmploymentShape`). `Employer` hash ids: `PayrollBP` 1 (now written only when non-zero; every valid payroll block has it at least 1, so no valid hash changes), `Recipes` 2 (a set: sorted in `canonical`, list order irrelevant), `MaxHires` 3, `WorkerShareBP` 4, `HireYieldBP` 5. The two forms never run together: that removes the double payment (payroll pays any worker with a job in progress, a hired job included). `spec-employment.md` 24.3.2 calls the mixed role "a question for that form"; Phase 1 writes this answer into 24.3 and 13 (Q1).
- **D2. Files.** `employment.go` keeps the state struct, newEmployment and the payroll rule. Continuation files, named before code (E1): `employment_validate.go` (the validator of both forms and the nine warnings), `employment_round.go` (P2.5a: admission, order, matching, formation), `employment_hire.go` (the hire record, the release, completion, the interruptions, `open_hire`, the audit), `employment_report.go` (events, logs, the `Warn` runs, snapshot rows, the codec), `employment_strategy.go` (the default offer and vacancies, the view filler, the planned hires). Each maps to area EMP and its tests carry the `Requirement:` lines. Added to `architecture.md` 2.6 and the `AGENTS.md` Layout in one edit with a fresh RULE-TAG. A file stays unsplit if it fits Lizard (CCN 10, NLOC 50, 100 lines per function) and 500 non-comment lines.
- **D3. The pure package** `internal/employment` gains `wage.go` (`Wage`, `Bill`: half-to-even, 128-bit through `internal/checked`), `order.go` (`SortOffers`, `SortVacancies` over a total order ending in the agent ref; the key `K(t, id)` is computed by the root and passed in as `uint64`, FR-EMP-018), `match.go` (`Round`: first-fit of one lane with fit, closing and `MaxOpen`, taking the formation checks of the world as one func parameter that returns a reason and a closing kind), `positions.go` (`Positions` bound by establishment, stock, tools and cash, 4.3.3) and `circuit.go` (the closed form of 11.2, test oracle only). Stdlib and `internal/checked` and `internal/ordering` only; no draw (FR-EMP-018, FR-EMP-061).
- **D4. The hired job reuses the trainee path.** A hired request is an ask of the worker (`w.pr.asks[worker]`) whose `startReq` is built by one guarded call `w.hiredRequest(&q, a)` beside `traineeRequest`: `account` and `principal` are the employer's, `exec` the worker, `weight` bounds `k`, the effective factor uses `HireYieldBP` where `trainee` stood, and `residueOwner` returns the employer (extend its test, not its shape). Output goes to the principal at completion by the existing `complete`. The own request of a matched worker is withdrawn (`ask.set = false`, FR-EMP-012). LND's principal charge is already in `startJob`, so the rent of a hired draw is the employer's.
- **D5. Money.** Reasons `wage_escrow` (agent to Escrow) and `wage_refund` (Escrow to agent) join `internal/ledger/registry.go` with the next free codes (grep at Phase 4: 42 and 43 are free at `15bcf18`; the last code in `internal/ledger/checkpoint.go` is 41) and `ReasonWageEscrow`, `ReasonWageRefund` in `event.go`. `wage_pay` gets the second pairing Escrow to agent: Phase 4 checks whether the registry row can name two pairings or whether the payroll row needs a union set (`setAE`-style) so the payroll test of FR-EMP-072 still sees agent to agent. None is a boundary row (FR-EMP-039). A money flow named like a new reason is `ErrReasonCollision`.
- **D6. One Escrow free list.** The free list is "lowest-ID open Escrow account of the region that no open contract or hire uses". It is derived, never saved. MER's two derivations (`contract_settle.go:261`, `:389`) and the restore check (`checkpoint_restore.go:398`) gain the hire list as a second user through one helper `escrowUsed(id)`; with MER off and the hire form on the check must accept an Escrow account (today it requires `w.mer.on`). MER's audit of "exactly `price * quantity`" is for a contract's account only (FR-EMP-040, spec row for MER 9.5, 9.6.2).
- **D7. Formation checks are an ordered table with named slots.** The order of 5.5 is a rule constant in `employment_round.go`: 1 `too_many_hires` (closes the lane), 2 `not_permitted` (slot for ORG), 3 `zero_yield`/`no_site` (closes the vacancy), 4 `no_tool`, 5 `no_facility` (slot for CAP), 6 `no_input`, 7 `no_endowment`, 8 `no_title`/`rent_funds` (LND bounds, from `tenure.leastRent` and the rule of `spec-land.md` 4.5.1), 9 and 10 the CAP build reasons (slots), 11 `no_room` (STO `k_room`), 12 `employer_funds`. A slot for an unbuilt module is one line `if w.cap != nil` that does not exist yet; the table row exists, the call does not. All 22 count fields of `LabourRound` are declared now in the order of 15.1 (the event is new, so no append rule applies), and the CAP and ORG ones stay 0 until those modules fill them (Q4).
- **D8. Positions.** `P2.5a` (after P2.5) and `P2.8b` (after P2.8a) leave `notBuiltPositions` and become `line` entries in `tick.go`. `positionsProduce` is at CCN 10, so Phase 4 starts with a refactor-first commit that splits it at the P2.5 / P2.6 boundary (the position test reads the straight line in source order; it must still find them in order). P2.1 release and P2.7 release are sub-steps inside `p21Complete` and `startJob` (one guarded call each, not positions).
- **D9. No new draw, no state in the strategy.** The round takes no word of the stream; ties go by `K(t, agent id)` of `spec-world.md` 10.1.1 then the agent ID (`internal/ordering`). The module adds no row to the draw table. The default strategy keeps no state and draws nothing for an offer or a vacancy (FR-EMP-009, 061). The viability solver keeps the payroll-aware duals for the payroll form only (ADR 0012); in a hire world `SolveSteadyState` is unchanged and the report says nothing about wages (spec open question 7; a note, E6).
- **D10. Ordering of the events of a removal.** `spec-employment.md` 6.5.2 lists the worker's `HireEnded` before `TraineeshipEnded` and `ContractSettled`; `observability-and-logging.md` 4.3 and the code (`removeAgent`: trainees, job, journey, contracts, 2b, then 2c to 2g) put step 2c after `TitleTransferred`. The rule file and the code win (architecture.md 2.10.4: the order is written once): 2c is called after the tenure call and before `dropClaims`; Phase 1 corrects 6.5.2's list. The worker's own `RecipeAbandoned` (`agent_removed`) is the existing step 1 and precedes its `HireEnded` (`worker_left`); an employer's hires each emit the worker's `RecipeAbandoned` (`principal_left`) then `HireEnded` (`employer_left`), in ascending hire ID.
- **D11. The checkpoint tag is 24** (the row is in `notBuiltSections`; `spec-checkpoint.md` 4.2 already says 24). `spec-employment.md` 16 says "tag 22 and renumbering, which changes the format version": that text is stale. The format version is 1 and stays 1: sections are gated by their module, so a world with the module off writes no section (E2 corrects 16). One section, no field in the agent record (the worker's job is PRO's record, N-128). Saved: the next hire ID, the hires in flight in ascending ID (employer, worker, wage, Escrow account, formation tick, region index), the two diagnostic runs per (region, role), sparse. Not saved: the free list, committed executions, the reservation, the bid, `Employed`, `Employing`.
- **D12. Hash and registry ids taken.** `Modules.Employment` is id 13 (exists). New inner ids as D1. `Role` ids 23, 24 exist. No new `Role` field. The registry staging entries `Employment.MaxOpen`, `Employer.Recipes`, `Employer.MaxHires`, `Employer.WorkerShareBP`, `Employer.HireYieldBP` leave `notBuilt` in PR 2 (both directions); `Employment.Hire` is a new row in the parameter-owner table of `doc/specs/README.md`. `Employment.Hire`, `Employment.MaxOpen` and the four employer fields also need rows in `internal/calibration/scope_paths_test.go` and `spec-calibration.md` 6.
- **D13. Names taken (grep of the tree and the sibling branches).** Events `HireFormed`, `HireEnded`, `LabourRound`; positions `P2.5a`, `P2.8b`; checkpoint tag 24; ledger reason codes 42 and 43 (re-grep at Phase 4); hash ids D1; scenarios S97 to S101 (`spec-reference-chain.md` 7.12; no `scenario_s97` to `s101` file exists, and S105, S106 are harness tests of SET, not here); notes N-369 to N-372 (the last on `phase-3` is N-366; `s3-crd-4` holds N-367 and N-368 unmerged; the sibling worktrees `s3-cap-1`, `s3-fis-1`, `s3-org-1` and `s3-rsk-1` have none beyond N-366; the supervisor reconciles at merge); log key `vacancies` (not in the `logkeys` table yet). Overlay `overlay/employment-hire.toml` and golden directory `testdata/golden/employment_hire/` are new names (no existing file moves).
- **D14. Module-alone golden.** `testdata/golden/employment_hire/` (new files): pack `one-demand` + `overlay/employment.toml` replaced by `overlay/employment-hire.toml` (employer role with a hire list, a `Hireable` role with no recipe), seed 1, 400 ticks, `WithAudit`, recorded once after the diff against `golden/off` and `golden/employment` is read. `golden/off/`, `golden/all/`, `golden/employment/`, the pinned hash and `allLayers` are not touched by any phase (owner decision of 2026-11-13: `golden/all` moves once at Stage 3 close). `Modules.Employment` is already in `allPackNotYetOn` logic as payroll; the hire form adds nothing there.
- **D15. Reference-chain limitation.** The packs collapse (N-352). The golden and S97 to S101 assert invariants (conservation, the escrow identity, determinism, the stream equal to module off, the closed form where the pack allows) and report the rest. A probe test in Phase 8 fails if the golden records no `HireFormed`; if `one-demand` gives none the pack moves to the first of `payroll`, `five` that does (Q6).

## Dependencies on other Stage 3 modules, and the seams

EMP is built on the tree as it is. Roadmap "Needs": AGT, WEL, PRO; the code also reads LED, REG, EST, SKL (the trainee path), SUB (`in_own_use`), DEM (`class_without_income`), INT (`AgentState`), CHK, OBS and, when on, LND (check 8), STO (check 11) and TEC (`tech` in `effectiveAt`). All merged. Roadmap row to correct in Phase 1: `Needs` reads "AGT, WEL, PRO, LED, REG, EST, SKL, SUB, DEM, INT, CHK, OBS".

| Place | What is there | What this plan does |
| --- | --- | --- |
| `registry_test.go:51` | five hire fields in `notBuilt` | delete in PR 2 |
| `event_test.go:106`, `observe_internal_test.go` (`stepSites`), `cardinality_test.go`, `catalogue_otel_copy_test.go`, `otelobserver/testdata/all_events.txt` and the otelobserver lists | `HireFormed`, `HireEnded`, `LabourRound` listed not built | delete as each lands (two in PR 3, `LabourRound` in PR 5) |
| `tick.go:90` | `P2.5a`, `P2.8b` | delete, append the calls (D8) |
| `checkpoint_sections.go` | `{24, "Employment", ...}` | delete, add the row (PR 4) |
| `contract_settle.go`, `contract.go`, `checkpoint_restore.go:398` | Escrow free list and kind check assume MER | one helper (D6) |
| `skills.go` `residueOwner`, `traineeRequest` | trainee principal | add the hired case (D4) |
| `demand.go` `classWithoutIncome` (line 803, CCN 9) | no `Hireable` rule | skip the class for a `Hireable` role when an employer role exists; pure function of `Config`, no `w` (as LND's H17) |
| `welfare.go` | `addEarned`, `addWagePaid`, `markWorking`, `workRead` | the hired-tick rules (E8 checks the "bought or sold" clause location first) |
| `strategy.go:179` `JobRequest` | three fields | two value fields, zero value unchanged (FR-EMP-009) |
| `tenure_ration.go:483` `leastRent`, `storage` `k_room` | callable in the root | formation checks 8 and 11 |
| notes N-144, N-147, N-187, N-190, N-191, N-194, N-327, N-355 | open for the hire form | updated in Phase 1 (apply, reject or leave open with what was learned) |

Seams EMP leaves:

| Module | Needs from EMP | Seam here |
| --- | --- | --- |
| CAP | checks 5, 9, 10 and `k_cap` | named slots in the table of D7; the event fields exist |
| ORG | check 2 `not_permitted`; a wage control (N-190) | named slot; a hire is "formed at a price from two limits" (the clamp is ORG's) |
| SAV | the outlet of an employer's profit | the warning `employer_no_outlet`; the cure list names SAV |
| FIS | a tax on `wage_pay` | one transfer function (D5) |
| MIG, SPN | `open_hire`; a role change ends a hire | registered refusal; the agent-lifecycle fan-in hook |
| CRD | the waterfall sees the refunded escrow | step 2c before `dropClaims` (D10) |
| IND | three presets | built here (Phase 8) |

## Phases

Each phase ends green on `go test -short -race ./...` and on the tests it adds; `testdata/golden/off/`, `golden/employment/` and the pinned hash do not move in any phase (a test run at every phase exit). No existing golden is regenerated.

### Phase 1: the specs are true (no code)

Goal: nothing below starts before these are applied by the supervisor or the worker named in the PR. The plan itself edits no spec.

- **E1.** `architecture.md` 2.6 and `AGENTS.md` Layout: the five continuation files of D2, identically in both lists, with a fresh RULE-TAG. `internal/employment` already has a coverage row in `scripts/ci/coverage-thresholds.txt`; verify it. `doc/documentation_index.md`: a row for this plan. `doc.go` glossary: the terms of `spec-employment.md` 23.2 not yet there (hire, `HireID`, hire list, establishment, vacancy, positions, labour offer, reservation wage, employer's bid, worker share, labour round, lane, escrow of a hire, unmatched, unemployment, hired yield, `open_hire`) and the type names `LabourOffer`, `HireRequest`, `HireRow`, `LabourRow`, `VacancyRow`, `HireFormed`, `HireEnded`, `LabourRound` (`TestGlossary_DeclaredTypesNamed`).
- **E2.** Spec edits, one row per file of `spec-employment.md` 12: the rows that bind code before PR 3 (`spec-production.md` 4.2, 4.3.2, 6.3, 10.3; `spec-agents.md` 3.6 step 2c, 6.3, 6.4.1, 7.1; `spec-welfare-and-wages.md` 2.1, 4.3, 8.4; `spec-world.md` 6.4, 6.11; `spec-ledger.md` 3.1, 7.2; `spec-merchants.md` 9.5, 9.6.2, FR-MER-039, FR-MER-045; `spec-config.md` 3.2, 3.7, 8.2.2 with the ids of D1 and D12; `spec-checkpoint.md` 4.2). Also in `spec-employment.md`: 24.3 and 13 (the form selector, D1), 16 (tag 24, no format bump, D11), 6.5.2 (the event order, D10).
- **E3.** The hook list of "Hook discipline" is approved by the owner with measured Lizard numbers; the refactor-first commits are named (`positionsProduce`, `startJob` if at 10, `fillAgentView`, `auditModules`).
- **E4.** Read `doc/notes.md` in full and quote its tag; update N-144, N-147, N-187, N-190, N-191, N-194, N-327 and N-355 with what this plan learned; add N-369 (a hire world keeps the payroll duals out of the solver: spec open question 7), N-370 (the escrow free list is shared by MER and EMP), N-371 (the form selector and the one-form rule), N-372 (event-order of 6.5.2 vs the rule file, D10); close none without a verdict.
- **E5.** `roadmap-v1.md` row: `Needs` as above; status "plan written".
- **E6.** Decide Q1 to Q9 (the plan decides none; the recommendations are taken).
- **E7.** Verify in code: the registry can hold `wage_pay` with two pairings (D5); `residueOwner` and `traineeRequest` extend to a hired request without a new field on `startReq` beyond one flag (D4); the "bought or sold" clause of WEL 4.3 lives in one function; `fillAgentView`'s group order; `workAgent` skips an agent with `jobSet` (a matched worker's request is withdrawn); `overlay/employment.toml` loads over `one-demand`; a Hireable role with no recipe is accepted by validation (a role with empty `Recipes` and the module on).
- **E8.** Measure the host functions of the hook list with `lizard -l go -C 8 -a 8 -L 80` and by hand for the three the tool skips.

Exit check: the supervisor confirms E1 to E8; Phase 2 starts. No Go change except the glossary comment in `doc.go`.

### Phase 2: `internal/employment`, the hire arithmetic (PR 1)

Goal: every number of 4.3.3, 5.3, 5.4 and 5.5 as a function over primitives, no world.
Files: `internal/employment/{wage.go, order.go, match.go, positions.go, circuit.go}`, tests beside them.
Tests: `TestWage_ReserveToBidHalfEven_InRange` (FR-EMP-021: the wage lies in `[Reserve, Bid]`, worked cases at share 5 000, 1 and 10 000; `Reserve + (Bid - Reserve) * s / 10 000` with the half-to-even example 3 + 7 * 5 000 / 10 000 = 6.5 to 6), `_ZeroWageSkipsPair`, `_IndependentOfCounts` (FR-EMP-022); `TestBill_Product128` (the product at `MaxPrice`, `MaxRecipeTicks`, `MaxEstablishment`; FR-EMP-042); `TestOrder_OffersByReserveThenKeyThenRef`, `_VacanciesByBidDescThenKeyThenRef`, `_TotalOrder_NoTies` (FR-EMP-018, 019); `TestRound_FirstOpenVacancyAtOrAboveReserve`, `_OfferIndivisibleVacancyDivisible` (FR-EMP-020), `_FailureLowersFitToBlockLessOne`, `_LaneClosingReasonCloses`, `_VacancyClosingReasonCloses`, `_MaxOpenPerLaneNotWorld` (FR-EMP-023, 063), `_UnmatchedCountedOnce`; `TestPositions_MinOfEstablishmentStockToolsCash` (FR-EMP-015; the bounds with `k_own`, keep-the-food, `UpkeepCost`), `_ZeroWhenBidBelowOne`; `TestCircuit_ClosedForm_MatchesModelCell` (FR-EMP-062; the 16 cells of `circuit.py` as a table, expected values with their arithmetic in comments); `TestPackage_NoFloatNoRootNoDraw`. Fuzz: `FuzzWage` (never panics or overflows; wage in range; NFR-EMP-003), `FuzzRound` (every offer matched at most once, vacancy `Left` never negative, the result independent of input order after sorting), `FuzzPositions`. Benchmarks `BenchmarkRound`, `BenchmarkOrder` with `b.ReportAllocs()` (NFR-EMP-004).
Exit check: package coverage 90% or more; `gomu` on `internal/employment` only, in Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`, survivors named equivalent or killed; `lizard -w` clean; goldens and hash unchanged. The PR carries E1 to E8 (the supervisor may take them separately).

### Phase 3: configuration, validation, warnings, hash, loader (PR 2)

Goal: the hire form exists as data and is checked; it does nothing at run time.
Files: `config.go` (`Employment.Hire`, `.MaxOpen`; `Employer.Recipes`, `.MaxHires`, `.WorkerShareBP`, `.HireYieldBP`), `bounds.go` (`MaxEstablishment` 1 000, `MaxHireRecords` 1 000 000), `defaults.go` (`DefaultEmployment`, `DefaultHireEmployer`: `MaxOpen` 1 000, `MaxHires` 4, `WorkerShareBP` 5 000, `HireYieldBP` 10 000; `canonical` sorts `Recipes`), `errors.go` (`ErrEmployerWeight` after the last module's sentinels), `validate.go` (the call stays; the body moves to `employment_validate.go`), `employment_validate.go` (the shape rules of 14.1 for both forms, the cross-form rule of D1, the weight rule, the nine warnings of 14.2 through the existing `Warn` collection), `hash.go` (the pairs of id 13, the employer ids of D1, `PayrollBP` only when non-zero), `config/mapping.go` (`[modules.employment] hire = true, max_open`; `[role.<id>.employer] recipes, max_hires, worker_share_bp, hire_yield_bp`), `internal/logkeys` (`Vacancies`), `registry_test.go`, `internal/calibration/scope_paths_test.go`, `testdata/chains/overlay/employment-hire.toml`, `doc/specs/README.md` parameter rows.
Tests: `TestValidate_Employment_<Rule>`, one failing case per row of 14.1 with `errors.Is` and the path (`ErrOutOfBounds` for each field, `ErrUnknownRecipe`, `ErrDuplicateID`, `ErrEmploymentShape` for the empty list, the foreign recipe, both roles, the mixed form, `ErrEmployerWeight` for a group of scale above 1 and for more households than agents, `ErrModuleDependency`; FR-EMP-002 to 006); `TestWarnings_Employment` one case per row of 14.2 plus the silent case on the overlay (FR-EMP-059; the `employment_no_employer` and `_no_worker` are warnings in the hire form and errors in the payroll form, 24.4.2); `TestHash_EmploymentPayrollUnchanged` (the pinned encoding of a payroll world is byte equal), `_HireEncodingPinned`, `_FieldIDs`, `_ListOrderIrrelevant`, `_HireOnlyWithFlag`; `TestLoad_EmploymentHireToml`, `FuzzLoad` corpus extended, `FuzzValidateEmployment`; `TestConfigNoWageField` (FR-EMP-007: a reflection guard that no config field is a wage or multiplier). Registry test both directions.
Exit check: `golden/off`, `golden/employment` and the pinned hash unchanged; `config/` coverage at or above 90%; the staged lists lose exactly the five entries.

### Phase 4: the round and the hired job (PR 3)

Goal: with a consumer's strategy that returns an offer and a vacancy, a hire forms, the wage is escrowed, paid a tick at a time and the job completes. Interruptions come in Phase 5.
Files: `tick.go` (the refactor-first split of `positionsProduce`, then P2.5a and P2.8b entries), `strategy.go` (`LabourOffer`, `HireRequest`, the two `JobRequest` fields), `event.go` (`HireID`, the admission and formation reasons, the `Cause` constants of 15.1, `HireFormed`, `HireEnded`, `ReasonWageEscrow`, `ReasonWageRefund`), `internal/ledger/registry.go` and `ledger.go` (D5), `employment.go` (state: hires slice by ID, the next ID, scratch lanes), `employment_round.go` (admission of 5.2 in ascending agent ID, keys, lanes, order, `internal/employment.Round` with the formation callback, atomic formation of 5.5), `employment_hire.go` (the record; the release at P2.1 and P2.7; completion and the `not_started` end; the cut start refund of 6.1.3; the escrow helper of D6), `phase_production.go` (hooks H3 to H7), `production.go` (the factor, D4), `skills.go` (`residueOwner`), `contract_settle.go`, `contract.go`, `checkpoint_restore.go` (D6), `moneyflow.go` (H11), `event_test.go` and the otelobserver lists for `HireFormed` and `HireEnded` only, `observability-and-logging.md` 3.3.2 (key `vacancies`) and 4.3 (fresh RULE-TAG; the rows exist, the order text is checked).
Tests: `TestAdmission_Offer_<Reason>` and `TestAdmission_Vacancy_<Reason>` in the order of 5.2, `employment_off` Debug-only, `employer_full` cut not refused (FR-EMP-010, 011); `TestRound_RunsAfterWorkBeforeRation` (FR-EMP-017; P2.5a between P2.5 and P2.6 by the position test); `TestRound_TakesNoWord_StreamEqualsOff` (FR-EMP-018, 061; the stream with the module on equals off, with and without offers); `TestRound_WorkerAndEmployerInDifferentRegionsNeverMeet`; `TestFormation_<Check>` for checks 1, 3, 4, 6, 7, 12 in order, with the closing kind of each (FR-EMP-023, 042); `TestFormation_Atomic_BillMovesToLowestFreeEscrow_OpensWhenNone` (FR-EMP-024), `_CountsOwnTraineesAndHiresAgainstStock` (FR-EMP-025; a planted trainee), `_OwnRequestWithdrawn_UnmatchedStands` (FR-EMP-012); `TestHired_InputsToolsFromPrincipal_OutputToPrincipal_FactorHireYield` (FR-EMP-027, 028, 008; `RecipeExecuted.Agent` is the worker), `TestHired_CutStartRefundsUnstarted_NoHireEnded`, `_ZeroKEndsNotStarted_WholeEscrowRefunded` (FR-EMP-026), `TestPay_OneWagePerWorkingTick_P27ThenP21_NeverMoreThanEscrow` (FR-EMP-029), `TestCompletion_OutputLastWageRefundReleaseEnded_Completed` (FR-EMP-030; order `RecipeExecuted` then `HireEnded`, 6.3.3), `TestMoney_WagesPlusRefundsEqualBill_NoBoundaryFlow` (FR-EMP-036, 039), `TestEscrow_SharedFreeListWithContracts_AuditAcceptsHire` (FR-EMP-040; with MER on and off), `TestAudit_HireEscrowBelowWageTimesLeft_Fails` (FR-EMP-041), `TestLedger_ThreeReasonsPairings` (the `wage_pay` row accepts agent to agent and Escrow to agent, the payroll tests unchanged), `TestEvents_RebuildHouseholdMoney_HireFormedAndRelease` (FR-EMP-054 partial), `FuzzFormation` (a random scripted lane never creates or loses money). A deliberate-break pass (testing.md 9.3): flip the offer order, drop the fit rule, pay twice at the start tick, drop the escrow check; each fails a test.
Exit check: a scripted two-agent world through the public API forms, pays and completes a hire with `WithAudit`; position, catalogue and cardinality tests pass with the two events staged out; allocation baselines unchanged with the module off and a new measured row with it on (testing.md 8.2); gates of Phase 9.

### Phase 5: interruptions, the region bound, state, checkpoint (PR 4)

Goal: every row of 6.4 holds, the world can be saved and restored with hires in flight.
Files: `employment_hire.go` (the interruption table: worker removed, role change, move, weight fall, employer removed, employer role change), `fanin.go` (H1: `w.employment.dropAgent(a)` as step 2c after the tenure call; H2 the role-change call in `agentRoleChanged`/`takeRole` at the two positions of 6.5.2), `phase_exit.go` (`takeRole`, the employer's hires end before its own job, the worker's after), `regions_relocate.go` (`moveAgent`: the worker's job is abandoned `region_changed`, the hire ends `worker_moved`), `construct.go` (`w.bounds.add(openHire, w.employment.holds)`, the instance keys are the hire IDs, FR-EMP-037), `employment_report.go` (the codec, `HireRow`, `RoleRow.Employed` and `.Employing`), `snapshot.go`, `checkpoint_sections.go`, `checkpoint_save.go`, `checkpoint_restore.go` (the section of tag 24 and its checks, with the cross check called from the section's own load, not added to `checkpointCross`), `integration.go`/`integration_slots.go` (`AgentState` gains the hire as worker and as employer; the directive's `Offer`/`Hire` are admitted by the round, 4.3.2 of INT), `moneyflow.go` (the escrow in the contract sector, H11).
Tests: `TestInterrupt_WorkerRemoved_WorkerLeft`, `_WorkerRoleChanged`, `_WorkerMoved`, `_WorkerWeightFalls_NotAbandoned_RemainderRefundedAtEnd` (FR-EMP-031, 032, 033), `_EmployerRemoved_RefundToOwnAccountBeforeWaterfall_PrincipalLeft_AscendingID` (FR-EMP-034; with CRD on, the creditors' waterfall sees the refund), `_EmployerRoleChanged` (FR-EMP-035), `TestRemoval_EventOrder_TraineesThenHiresThenOwn` (FR-EMP-038, D10), `TestRelocation_PrincipalOfOpenHire_ErrRegionBound_OpenHire_KeysAreHireIDs` (FR-EMP-037), `TestWorker_KeepsEveryWagePaid` (FR-EMP-036), `TestCheckpoint_EmploymentRoundTrip` (sparse runs, one spelling; FR-EMP-044), `TestCheckpoint_RestoreRejects_<Case>` for each check (unknown employer, worker not live, escrow not an Escrow account, shared account, `ErrCheckpointCorrupt`), `FuzzDecodeEmploymentSection`, `TestFork_Employment` (through `forkAtN`/`forkCase`: after a formation, mid-job, after a removal; bit-identical; FR-EMP-043), `TestSnapshot_HiresAndRoleRows_FunctionsOfSavedState` (FR-EMP-055), `TestEmployment_OffLeavesNoTrace`.
Exit check: S99 (an employer and a worker removed mid-job) runs; the escrow identity holds at every tick over seeds with `WithAudit`; `golden/off` and `golden/employment` unchanged.

### Phase 6: the strategy, the view, welfare, demand (PR 5, first half)

Goal: the default strategy offers and posts; welfare and the observed wage treat a hired tick as 8 says.
Files: `employment_strategy.go` (the default's offer 4.3.1, reservation 4.3.2, vacancies 4.3.3; the view filler; the planned hires `H*` and the targets of 9.2), `strategy_view.go` (the view group of 9.1 and the hired yield per recipe), `strategy_default.go` (one guarded call), `fanin.go` (`fillAgentView` gains one guarded call; it is at CCN 10, so a refactor-first commit), `welfare.go` (the principal's wear cost, the hired working tick as `k` household-ticks, the principal's floor flag, the "bought or sold" clause not applying to a `Hireable` role, no window lengthening; one accessor, no logic of the module), `demand.go` (`classWithoutIncome`), `subsistence.go` (`in_own_use` admission), `employment_hire.go` (`Y_t` at the release, `V_t` at the working tick, never at formation).
Tests: `TestDefault_OfferCeilWSOverTenK_NotWhenRestOrOwnUseOrTrainee` (FR-EMP-013, 060), `TestDefault_ReservationBestAlternativeOrZero` (FR-EMP-014), `TestDefault_PostsBestRecipeBidFloorNetPerTick_PositionsMin` (FR-EMP-015), `TestDefault_PositionsNotScaledByWorkShare_OwnLabourStillRests` (FR-EMP-016), `TestDefault_NoStateNoDraw`, `TestStrategy_ZeroOfferHireUnchanged_AllStrategies` (FR-EMP-009; the existing strategies and `Directed`), `TestDirected_PlayerOffer_NotEmployer` (4.4.1), `TestWelfare_WagesInIncome_EscrowNotNetted` (FR-EMP-045), `TestWelfare_HiredTickIsWorkingTick_KHouseholdTicks_PrincipalFloorNoAdd` (FR-EMP-046), `TestWelfare_PooledWageExcludesEmployerRoles_NoState` (FR-EMP-047), `TestWelfare_HireDoesNotLengthenWindow` (FR-EMP-048), `TestWelfare_PrincipalPaysWear` (FR-EMP-049), `TestWelfare_InKindIgnored_WarningRaised` (FR-EMP-050), `TestDemand_HireableNoClassWithoutIncomeWhenEmployerExists`, `TestTargets_EmployerBuysInputsOfExpectedHires` (9.2.1), `TestView_NoHoldingsOfOthers` (a reflection guard that no view type carries another agent's hire).
Exit check: S97 (the closed form, report-only band where the pack collapses) runs; goldens unchanged.

### Phase 7: the aggregate, the logs, the seams to LND and STO (PR 5, second half)

Goal: every silent rule is counted; the cross-module formation checks are in.
Files: `tick.go` P2.8b call, `employment_report.go` (`p28bLabour`: `LabourRound` per region in ascending order with the `Workers` and `Employers` rows, the `Debug` lines, the two `Warn` runs of 15.2.2 saved in the section), `event.go` (`LabourRound`, `LabourRow`, `VacancyRow`, all counted fields of 15.1), `employment_round.go` (check 8 through `tenure.leastRent` and the rule of `spec-land.md` 4.5.1 as bounds; check 11 through the storage bound), `event_test.go`, `observe_internal_test.go`, `cardinality_test.go` (the labour refusal set joins `economysim.reason`: `employer_funds`, `no_input`, `no_tool`, `no_endowment`, `no_title`, `rent_funds`, `no_room`, `zero_yield`, `no_site`, `too_many_hires`, `no_labour`; 11 values), `catalogue_otel_copy_test.go`, `otelobserver/testdata/all_events.txt` and the otelobserver lists, `observability-and-logging.md` 3.3.2, 4.3, 5.3, 5.3.1, 5.4.1 (fresh RULE-TAG).
Tests: `TestLabourRound_OncePerRegion_OnlyWhenCounted_AscendingAfterProductionHalted_NoMap` (FR-EMP-053), `TestLabourRound_Counts_NoVacancyNotRestOwnUseTrainee` (FR-EMP-060), `TestLabourRound_WageBelowUnit_RefusalFieldsByReason`, `TestFormation_NoTitleRentFundsBounds` (FR-EMP-023 item 8, with a titled endowment), `TestFormation_NoRoomBound` (item 11), `TestEvents_RebuildMoney_FourItems` (FR-EMP-054), `TestWarn_LabourUnmatchedAndVacanciesUnfilled_TenthTickEveryTenth_PerRegionRole` (FR-EMP-056), `TestLog_OnlyDebugForRefusals_Guarded` (FR-EMP-058), `TestLog_NoInfoInTheTickAndNoError` extended with the overlay, `TestNonInterference_Employment` (observer and logger do not change results; FR-EMP-061), `TestCost_EmploymentNoopAllocs` (NFR-EMP-001; skips under race, so it takes a `gateFPrefixes` prefix, `TestRaceSkips_HaveAnFPrefix`), `TestWork_IsCounted` (NFR-EMP-002: operations proportional to admitted offers, vacancies and open hires, the pattern of `skills_queue_internal_test.go:75`), `TestCardinality_LabourReasons`, `BenchmarkRoundWorld` (NFR-EMP-004).
Exit check: catalogue, cardinality and position tests staged both ways; `golden/off` event files unchanged.

### Phase 8: the proof (PR 6)

Goal: the module is shown end to end and the three presets exist.
Files: `indicator/tally_observe.go` (the `LabourRound` case, `HireFormed`/`HireEnded` only if a preset reads them), the tally rows, `indicator/labour.go` (`Unemployment`, `WageBill`, `Employment`; one file if it stays under 500 non-comment lines), their tests with a worked example each (`worked_test.go` pattern), `indicator/doc.go` (the preset list), `testdata/chains/overlay/employment-hire.toml` is already from PR 2, `testdata/golden/employment_hire/{one-demand.txt, events-one-demand.txt, one-demand.positions.txt}`, `testdata/golden/indicator/` entries for the three presets (new files), `testdata/chains/expected/` entries, `scenario_s97_test.go` to `scenario_s101_test.go`.
Scenarios (invariants asserted, the rest reported, as `spec-reference-chain.md` 4.5 says):

- **S97 (EMP-S1).** `one-demand` + hire overlay, employers' wants off then on, seeds 1 to the scenario count, `WithAudit`: hired households reported against `h*` of 11.2; asserted: money conserved, the escrow identity, no hire without a bill, determinism. Requirement: FR-EMP-036, 039, 062.
- **S98 (EMP-S2).** A labour-bound variant (more places than offers), floor at its default and 0: the price stays within its bounds, `NoVacancy` and `Unfilled` reported. Requirement: FR-EMP-016, 046, 060.
- **S99 (EMP-S3).** An employer removed mid three-tick job and a worker removed mid another: the escrow returns to the employer's own account and reaches the estate; wages paid stay; event order. Requirement: FR-EMP-031, 034, 038.
- **S100 (EMP-S4).** A step in the yield of the employers' input: positions fall and recover, recovery time recorded (report only). Requirement: FR-EMP-015.
- **S101 (EMP-S5).** A harness test: invariants over a range of seeds with `WithAudit`; the module on takes no word, with or without offers; a checkpoint and fork mid-hire. Requirement: FR-EMP-043, 061, NFR-EMP-005.

Further tests: `TestDeterminism_Employment` over `detCases(true)` with and without observer and logger, `TestFork_EmploymentScenarios`, `TestIndicator_Unemployment_*`, `_WageBill_*`, `_Employment_*` (FR-IND-056 to 058: households not agents, `OK` false where nothing was offered or the value added is at or below 0, a role that is not hireable absent), `TestStylised_*` only if a fact reads the module, `TestGolden_EmploymentHireProbeExercisesHires` (a `gateFPrefixes` prefix since it skips under race), `TestConfigOnly_EmploymentHireNeedsOnlyConfiguration` (a setting-free pack in `testdata/chains/second/`; architecture.md 9.4).
Exit check: the golden recorded once after its diff against `golden/off` and `golden/employment` is read; `golden/all/` and the all-on checkpoint fixture untouched; new tests are in the quick-run prefixes.

### Phase 9: the exit gate (each PR)

Per PR, on the final head only (see "Gate economics"): `gofmt -l .` empty, `go vet ./...`, `staticcheck ./...`, `go mod tidy` no diff, `go test -short -race ./...`, gate R, gate F with `scripts/ci/coverage-check.sh`, in the root, `otelobserver/` and each `examples/` module; `lizard -l go -C 10 -a 8 -L 100 -w` on every changed non-test file; coverage at or above 90% for the root, `internal/employment`, `internal/ledger`, `config`, `indicator`; `gomu` per changed `internal/` file in Docker with the caps; the deliberate-break pass of testing.md 9.3; registry, catalogue, cardinality, glossary and position tests with EMP's entries deleted. Exit markers go in the PR description. arm64 is not run per PR (owner decision).

## PR split

The module touches the round, production, the strategy, the view, welfare, demand, the ledger, MER's escrow, the removal fan-in, relocation, the checkpoint, integration, the catalogue and `indicator/`, so one PR is far too large. Six PRs on `phase-3`, each green on its own gates and each leaving `golden/off/`, `golden/employment/` and the pinned hash unchanged. Estimates include tests.

1. **PR 1, Phases 1 and 2:** doc and glossary edits, the index row of this plan, the E2 spec rows, `internal/employment` hire arithmetic. No root Go change except `doc.go`. About 2 400 lines.
2. **PR 2, Phase 3:** configuration, validation, warnings, hash, loader, registry rows, calibration paths, `overlay/employment-hire.toml`. About 1 900 lines.
3. **PR 3, Phase 4:** the round and the hired job, two events, the ledger reasons, the shared Escrow free list. Refactor-first commit of `positionsProduce` first. About 3 000 lines; if over, split after formation (admission, order, matching, formation first; the release and completion after).
4. **PR 4, Phase 5:** interruptions, `open_hire`, removal 2c, state, snapshot, checkpoint tag 24, `AgentState`, fork. About 2 600 lines.
5. **PR 5, Phases 6 and 7:** the strategy, view, welfare, demand, `LabourRound`, logs, the LND and STO checks, the catalogue lists. About 3 000 lines; split at the Phase 6 and 7 boundary if over (5a, 5b).
6. **PR 6, Phase 8:** the three presets, the golden, S97 to S101, the determinism and probe tests. About 2 500 lines plus the golden.

**Minimal slice.** PRs 1 to 4 give a working hire for a consumer's strategy with a checkpoint; PR 5 gives the default strategy and the observability; PR 6 the proof. The plan does not recommend stopping before PR 6 (Q1).

## Catalogue appends (merge-conflict forecast)

Files marked * are the likely conflicts. A conflict is textual, adjacent lines, resolved by keeping both in module order.

| File | What EMP appends | Shared with | Conflict risk |
| --- | --- | --- | --- |
| `config.go`* | the fields of D1 on `Employment` and `Employer` | every module | Medium (EMP's structs exist) |
| `hash.go`* | the pairs of id 13 and the employer ids | every module | High |
| `bounds.go`, `defaults.go`, `errors.go` | two bounds, defaults, `ErrEmployerWeight` | every module; `errors.go` after CRD's | Medium |
| `validate.go`, `config/mapping.go`* | the keys of the hire form | every module | High |
| `event.go`* | `HireID`, `HireFormed`, `HireEnded`, `LabourRound`, rows, reasons, causes, `ReasonWageEscrow`, `ReasonWageRefund` | CRD, CAP, RSK, STO | High |
| `event_test.go`*, `registry_test.go` | entries deleted | every module | High (adjacent lines) |
| `snapshot.go` | `RoleRow.Employed`, `.Employing`, `HireRow`, `Hires` | CRD, CAP, RSK | Medium |
| `tick.go`* | `P2.5a`, `P2.8b`; `notBuiltPositions` loses two; `positionsProduce` split | CAP, SAV, FIS | High (the straight line of phase 2) |
| `fanin.go`* | step 2c, `fillAgentView` call | CRD, CAP, INS, ORG | High |
| `checkpoint_sections.go`, `_save.go`, `_restore.go` | the row of tag 24 | every module with a tag | High |
| `integration.go`, `integration_slots.go` | `AgentState` hire fields | CRD, CAP, PLC | Medium |
| `phase_production.go`, `production.go`, `skills.go`, `welfare.go`, `demand.go`, `subsistence.go`, `moneyflow.go` | guarded calls (Hook discipline) | LND, STO, CAP, CRD, SAV | Medium |
| `contract_settle.go`, `contract.go`, `checkpoint_restore.go` | the shared Escrow helper | MER only | Low |
| `internal/ledger/registry.go`, `ledger.go` | two reasons, one pairing | CRD, FIS | Medium |
| `indicator/tally_observe.go`, `indicator/*` | the `LabourRound` case, three presets | RSK/STO (preset 25) | Medium |
| `doc.go`, `AGENTS.md`, `architecture.md` 2.6, `observability-and-logging.md` | glossary, continuation files, key, rows, a fresh RULE-TAG per rule-file edit | every module | Medium; the tag line always conflicts, the supervisor applies these edits |
| `internal/logkeys`, `scripts/ci/coverage-thresholds.txt` | `Vacancies`; (the `internal/employment` row exists) | every module | Low |
| `testdata/chains/overlay/`, `testdata/golden/employment_hire/`, `testdata/chains/expected/` | new files | every module | Low |
| `testdata/golden/all/`, `all_pack_test.go` `allLayers`, the all-on checkpoint fixture | **none** (moves once at Stage 3 close) | every module | None here |

## Hook discipline

Each hook is one guarded call into `employment*.go` with no branch on module data and no arithmetic in the host (architecture.md 2.10.1, 2.10.2, 2.10.4). "Guard" means `if w.employment != nil && w.employment.hire`. The owner approves this list before PR 3 (E3).

| # | File and function | The call | PR |
| --- | --- | --- | --- |
| H1 | `fanin.go`, `removeAgent` after the tenure call, before `dropClaims` | `w.employment.dropAgent(a)` (step 2c) | 4 |
| H2 | `fanin.go`, `agentRoleChanged` and `phase_exit.go` `takeRole` | `w.employment.roleChanged(a, from)` at the two places of 6.5.2 (employer's hires before its job abandon, worker's after) | 4 |
| H3 | `tick.go`, the straight line | `P2.5a` after P2.5 and `P2.8b` after P2.8a (positions, not sub-steps) | 3, 5 |
| H4 | `phase_production.go`, `p21Complete` | `w.employment.release(...)` for the later working ticks and the completion hire end | 3 |
| H5 | `phase_production.go`, `startJob` | the hired request through `startReq` (H6) and the release of the start tick, the cut refund | 3 |
| H6 | `phase_production.go`, `startReq`, beside `traineeRequest` | `w.hiredRequest(&q, a)`: account, principal, factor | 3 |
| H7 | `skills.go`, `residueOwner` | the employer for a hired job | 3 |
| H8 | `regions_relocate.go`, `moveAgent` | the worker's `worker_moved` end; the registered obligation `open_hire` through `w.bounds.add` in `construct.go` | 4 |
| H9 | `fanin.go`, `fillAgentView` (CCN 10: refactor-first) | `w.employment.fillView(v, a)` | 5 |
| H10 | `strategy_default.go`, `DefaultStrategy.Work` | `employmentWork(v, &req)` (the offer and the vacancies) | 5 |
| H11 | `moneyflow.go`, `auditModules` (CCN 10: one call added inside the existing function after a refactor-first split if Lizard says so) | `w.employment.audit()`; the escrow in the contract sector | 3 |
| H12 | `welfare.go` | one accessor for the principal's flag and one for the hired working tick (beside `addEarned`/`addWagePaid`) | 5 |
| H13 | `demand.go`, `classWithoutIncome` | a read of `cfg` for a `Hireable` role with an employer role in the world (a pure function of `Config`) | 5 |
| H14 | `subsistence.go` | `in_own_use` read for the admission | 5 |
| H15 | `contract_settle.go`, `contract.go`, `checkpoint_restore.go:398` | `escrowUsed(id)` helper; the kind check accepts a hire's Escrow account | 3 |
| H16 | `checkpoint_sections.go` | the row of tag 24 | 4 |
| H17 | `integration.go` | `AgentState` hire fields through one call | 4 |
| H18 | `snapshot.go` | `w.employment.fillRows` for `RoleRow` and `Hires` | 4 |
| H19 | `production.go`, `limitList`/`start` | none: the hired request uses the existing refusals; a reasons change here is a defect | — |

H5 and H6 are the likeliest to be logic rather than a call: the host holds one call each and the arithmetic of the cut (`k = min(m, grant, live inputs, k_room, k_rent)`) stays in the existing `start`, which already takes the rent and room caps from LND and STO. H4 and H5 sit in functions at or near the Lizard limit; each has a refactor-first commit if the measurement of E8 says so.

## Testing (testing.md 11)

1. **Requirements touched and their tests.**

| Requirement | Tests |
| --- | --- |
| FR-EMP-001, 009, 061 | `TestEmployment_OffLeavesNoTrace`, `TestStrategy_ZeroOfferHireUnchanged_*`, `TestRound_TakesNoWord_StreamEqualsOff`, `TestNonInterference_Employment` |
| FR-EMP-002 to 007, 059 | `TestValidate_Employment_*`, `TestWarnings_Employment`, `TestConfigNoWageField` |
| FR-EMP-008, 024 to 030 | `TestFormation_*`, `TestHired_*`, `TestPay_*`, `TestCompletion_*` |
| FR-EMP-010 to 012, 017 to 023, 063 | `TestAdmission_*`, `TestRound_*`, `internal/employment` tests |
| FR-EMP-013 to 016 | `TestDefault_*` |
| FR-EMP-031 to 038 | `TestInterrupt_*`, `TestRemoval_EventOrder_*`, `TestRelocation_*` |
| FR-EMP-036, 039, 040, 041, 042 | `TestMoney_*`, `TestEscrow_*`, `TestAudit_*`, `TestLedger_*` |
| FR-EMP-043, 044, 055 | `TestCheckpoint_*`, `TestFork_Employment`, `TestSnapshot_*` |
| FR-EMP-045 to 050 | `TestWelfare_*`, `TestDemand_*` |
| FR-EMP-051 to 054, 056 to 058, 060 | `TestLabourRound_*`, `TestEvents_*`, `TestWarn_*`, `TestLog_*`, `TestCardinality_*` |
| FR-EMP-062 | `TestCircuit_ClosedForm_*`, S97 |
| FR-EMP-064 to 080 | the payroll tests, unchanged and rerun at every phase; `TestHash_EmploymentPayrollUnchanged` |
| FR-IND-056 to 058 | `TestIndicator_*` |
| NFR-EMP-001, 002 | `TestCost_EmploymentNoopAllocs`, `TestWork_IsCounted` |
| NFR-EMP-003 to 005 | `FuzzWage`, `FuzzRound`, `FuzzPositions`, the benchmarks, S101 |

1. **Lowest level.** The rules are unit tests of `internal/employment`; the hooks are internal tests of each host file; the pack tests are S97 to S101; the end-to-end level is the module-alone golden and the fork test.
1. **New boundaries and error paths.** The two sentinels, the bounds, the bill product at the extremes, the escrow at zero, a worker weight that falls, the decoder of the section, the principal at a removal.
1. **Golden files.** `golden/off/`, `golden/employment/`, `golden/all/`, the pinned hash and every existing fixture do not move. `golden/employment_hire/` is new and recorded once.
1. **Fork-at-N.** `TestFork_Employment` with N in {0, after a formation, mid-job, after a cut start, after a removal} and S101.
1. **Registry and catalogue.** The five fields, three events, two positions and the tag leave their lists both ways.
1. **Mutation.** `gomu` only inside Docker (`--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`) on `internal/employment` and the new ledger rows; never on the host.
1. **Broken versions** (testing.md 9.3): the offer order, the fit rule, a double release, the missing escrow check and the refund destination each flipped once per phase exit.
1. **Gate F prefixes.** Tests that skip under race (`AllocsPerRun`, the golden probe) are named with a prefix of `gateFPrefixes` (`TestCost`, `TestGolden`, `TestScenario`, `TestDeterminism`, `TestFork`, `TestCheckpoint`, `TestInvariants`): `TestRaceSkips_HaveAnFPrefix` enforces it.
1. **Fuzz.** Every parser or validator added has a `Fuzz` test and committed corpus.

## Observability (observability-and-logging.md 9)

1. **New failure path.** The validation rules return `ValidationError`. A refused offer, vacancy or formation is a counted refusal in `LabourRound` and a `Debug` line behind `Enabled`, never an error from `Step` (5.2). A transfer the escrow guarantees cannot be refused is an invariant.
2. **New state transition.** A hire forms (`HireFormed`), ends (`HireEnded`, closed `Cause`), a round is counted (`LabourRound`, per region and tick, no map).
3. **New decision point in a strategy.** The default's offer and vacancy: a `Debug` line with the reservation or bid and `reason` is not added (the counts and the events say it); a consumer's strategy logs its own.
4. **New loop.** The round is `O(O log O + V log V)` per region; event construction and the counters that exist only for it are under `w.obs.supplied`; proven by `TestCost_EmploymentNoopAllocs`. `Warn` runs are per (region, role), never per agent.
5. **Changed event, metric or attribute.** None changed; additions only. `RoleRow` gains two counts, the snapshot a slice.
6. **New metric attribute.** `economysim.reason` gains the eleven labour refusal values (bounded by the closed set). The five instruments are an adapter PR (Q5). 5.4.2: series added for the reference chain are stated in the PR (roles times regions times 11).
7. **`Error` or `Info`.** None. `Warn`: "labour unmatched" and "vacancies unfilled", first on the tenth tick of a run and every tenth after.
8. **Error logged and returned.** None.
9. **Tests.** A recording observer for the three events and their order, a capturing handler for the two `Warn`s and the absence of `Warn`/`Info` otherwise, non-interference, the no-op allocation test, the cardinality test, the money-rebuild test (FR-EMP-054).

## Codacy hygiene (every phase)

The Codacy gate fails on any new Info-or-higher issue. Before a PR is opened and again after the last push:

- Every changed `.md` passes `npx markdownlint-cli2 <file>` with no finding (PATH needs `/home/andy/.nvm/versions/node/v24.22.1/bin`): a blank line around every list, tables with `| --- |` separators, no double blank lines, no trailing spaces, one top-level heading.
- Every new Go function is within Lizard CCN 10, NLOC 50 and 100 lines (`lizard -l go -C 10 -a 8 -L 100 -w` on each changed file); test functions too. No Go file over 500 non-comment lines: `employment_round.go`, `employment_hire.go` and `employment_report.go` are split by the measured size, not up front.
- `gofmt -l .` empty, `staticcheck ./...` clean, no unused parameter, no shadowed variable, no commented-out code.
- Refactor-first commits (`positionsProduce`, `fillAgentView`, `auditModules`, and `startJob` if measured at 10) are the first commits of their PR, with their tests unchanged, so a Lizard regression is never mixed with a behaviour change.

## Gate economics

Full gates (gate R and gate F, `scripts/ci/check.sh`, `full.sh`) run **once per PR, on the final head**, with `GOMAXPROCS=8`, and one gate run at a time on the host (earlyoom killed gate F runs of POL PR B; `examples/realm` `TestServe_*` times out under `-race` when runs overlap). Between pushes only `go test -short -race` on the touched packages runs. A scenario test must not keep per-tick snapshots for every parallel seed (the S113 lesson): S97 to S101 keep counters. A gate that fails on a flake is rerun alone before it is read as a defect. `gomu` per PR only on the new `internal/employment` files and the new ledger rows.

## Risks and open items

1. **Two forms in one module (D1).** The payroll form must stay byte-equal. The hash rule "`PayrollBP` only when non-zero" and the form flag are the guard; `TestHash_EmploymentPayrollUnchanged` and `golden/employment` run in every PR.
2. **The reference chain collapses (N-352).** A hire world on `one-demand` may have no household left to hire. D15's probe fails the golden if no `HireFormed` is recorded; the scenarios assert invariants and report the rest. No claim on criterion 1.
3. **Size and Lizard.** `positionsProduce`, `fillAgentView` and `auditModules` are at CCN 10, `startJob` was at 10; refactor-first commits are budgeted; three functions are invisible to the local lizard run and are measured by hand.
4. **The shared Escrow account (D6).** A hire's account must not be taken by a contract and conversely; the audit of FR-MER-045 must not fail on a hire's account. Tests run with MER on and off.
5. **The matched worker's own request.** It is withdrawn after admission and the hired request joins the ration at P2.6; a mistake here double-starts a worker. `TestFormation_OwnRequestWithdrawn_*` and the deliberate-break pass cover it.
6. **The wage is derived, not configured, and does not clear the market (EMP-D1).** Unemployment can persist at a wage; this is reported (`LabourRound`, `Unemployment`), not corrected.
7. **The loop with the target-income rule (L14, EMP-A1).** The backward-bending supply is real and reported; no default is called stable (N-027).
8. **Merge order.** Many Stage 3 plans edit `tick.go`, `event.go`, `fanin.go` and the three lists. The supervisor orders merges; a conflict resolved by keeping both lines in module order is safe, except the three lists (`registry_test.go`, `event_test.go`, `tick.go`), which are rebuilt from the tree on the integration branch.
9. **Spec text owed.** E2 is a large table; the rows that bind code before PR 3 are in PR 1, the rest go with the PR that builds the part (Q2).
10. **Slots for unbuilt modules.** The `LabourRound` fields for CAP and ORG stay 0; a test asserts they are 0 so a later module's fill is a visible change.
11. **Cost.** The round adds work only for admitted offers and vacancies; `BenchmarkRoundWorld` and the allocation row report the trend.

## Security, rollback and compatibility

- **Untrusted input.** The only external input is what a consumer's `Strategy` or a directive returns in `Offer` and `Hire`, and the configuration. Both are checked whole before any state changes (admission 5.2, validation 14.1); every count is bounded (`MaxOpen` per region, `MaxHires` up to `MaxEstablishment`, one offer or vacancy per agent per tick, `MaxHireRecords` Escrow accounts), so a hostile strategy cannot grow state or work without bound (NFR-EMP-002). Money products go through the 128-bit helper and a refusal is a counted reason, never a wrap. A checkpoint section is decoded with bounded allocation (`FuzzDecodeEmploymentSection`) and every reference is checked as a live agent or Escrow account.
- **Rollback.** Each PR is reverted by itself: the hire form is off unless `Employment.Hire` is set, so a revert leaves every existing world, golden and hash unchanged (the guard tests run in every PR). PR 4 adds the checkpoint section: a checkpoint written with the hire form on is not readable after a revert of PR 4, which is acceptable before the first release and is stated in the PR.
- **Compatibility.** `JobRequest` and `Employer` gain fields at their zero value; the payroll form, its hash and `golden/employment` are byte-equal (D1); the format version stays 1 (D11); the `wage_pay` registry row gains a pairing and the payroll tests rerun.

## Questions only the owner can answer

Each carries a recommendation, which the supervisor has delegated and this plan treats as accepted.

- **Q1. Scope.** All six PRs, or PRs 1 to 4 first? Recommended: all six; PR 5 may split at the Phase 6 and 7 boundary and PR 3 after formation if over 3 000 lines.
- **Q2. Who applies the spec edits of E2.** Recommended: the rows that bind code before PR 3 in PR 1; the observability, checkpoint and integration rows with the PR that builds their part.
- **Q3. The hook list.** Approve H1 to H18 and the refactor-first commits. Recommended: approve; H5 and H6 are held to one call each.
- **Q4. The fields of CAP and ORG in `LabourRound` and the named slots.** Recommended: declare all 22 refusal fields now, fill only those with a built module, and test the rest at 0.
- **Q5. Metrics.** The five instruments in `otelobserver/` in a separate PR, or a note as SKL and LND did. Recommended: defer; PR 5 adds only the events to the test lists.
- **Q6. The pack of the golden.** A green but possibly short golden on the collapsing `one-demand`, with a probe that fails if no hire forms and a move to `payroll` or `five` if so. Recommended: yes.
- **Q7. Names.** Confirm `Employment.Hire` and `MaxOpen` (D1), the files of D2, the ledger codes 42 and 43 and the notes N-369 to N-372 (reconciled at merge). Recommended: yes to all.
- **Q8. The form rule.** One form per world, `Hire` selecting it, `PayrollBP` 0 in the hire form. Recommended: yes (D1); a world that wants both is a later spec.
- **Q9. The event order of a removal (D10).** The rule file and the code win over `spec-employment.md` 6.5.2. Recommended: yes; correct the spec.

## PR 2 result (Phase 3, recorded when PR 2 was built)

Rule tags read: architecture=dcf04f1ce22f6eb654b58a0e, observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367. Notes tag read: 4b534c158301600c4881b9be (the whole of `doc/notes.md` was read by a subagent I delegated the read to, 460 KB; the tag is replaced by a fresh one in this PR). Notes added: N-428 to N-431; N-355 updated; N-371 stays open until PR 3 runs the form.

Built: `Employment{Hire, MaxOpen}` and the four hire fields of `Employer` (hash ids as D1; `PayrollBP` written only when non-zero, so no payroll hash moves), `MaxEstablishment` and `MaxHireRecords` in `bounds.go`, `DefaultEmployment`, `DefaultHireEmployer` and `DefaultMaxOpen`, `ErrEmployerWeight`, the validator of both forms and the nine warnings of 14.2 in `employment_validate.go` (moved out of `employment.go`), the loader keys in `config/employment.go`, `testdata/chains/overlay/employment-hire.toml` (a new file), the five staged registry entries gone from `notBuilt`. Nothing runs yet: the form is data, and `golden/off`, `golden/employment`, `golden/all`, the pinned hash and `testdata/checkpoints` are untouched (the payroll hash bytes are pinned by `TestHash_EmploymentPayrollUnchanged`).

Deviations and choices the plan left open:

- `max_open` set with `Hire` false is `ErrEmploymentShape` (the hash would ignore it), a clause 14.1 does not carry yet.
- The loader reads `[role.<id>.employer]` over `DefaultHireEmployer` when any key of the hire form is present, else over `DefaultEmployer`: the keys say the form, so the order the tables are read in does not matter.
- The warnings that read prices use the dearest region where the employer role has a group, a yield-table entry of 1 and the non-discard outputs; a role with no group draws none (the choice of `roleWarnings`).
- The validator is in `employment_validate.go`, the loader in a new file `config/employment.go` so that `config/mapping.go` does not grow.
- The overlay makes the diggers and the foresters Hireable and keeps their recipes; no pack role has an empty list.
- The five staged entries and `Employment.Hire` and `Employment.MaxOpen` leave `notBuilt`. The README and `spec-calibration.md` rows were already in PR 1.

Left for PR 3 (found here, not built here):

- The ledger codes are 22 (`wage_escrow`) and 24 (`wage_refund`), not 42 and 43: the pinned table of `spec-ledger.md` 7.2 names both and has 42 rows, and a code of 42 or more is the code of a configured money flow (`internal/ledger/checkpoint.go`, `TestReasonCode_Pinned`).
- P4.1b must not run in the hire form (D1): `newEmployment` still marks an employer role a payer and `collect` still walks the agents. Guard it by the form flag.
- The derived overflow check of `spec-production.md` 10.3 (`yieldMax` in `validate.go`) must take `HireYieldBP` of the hire-list recipes into the largest yield factor (14.1, last paragraph).

Tests: field cases for every new row of the parameter-owner table, the one-form rule, the hire list, the weight rule, one case per warning plus a silent configuration, the pinned hire encoding and the field ids of the hash, the canonical default of `MaxOpen`, the loader (a hire document, the reject table), `FuzzValidateEmployment`, `FuzzLoadEmployment` with new seeds, and the pack test of the overlay. A break pass (the hire-list rule, the share bound, a hash id, a warning test) failed the tests it should.
