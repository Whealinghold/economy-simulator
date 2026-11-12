# Plan 2-29: a payroll-aware steady state and a sub-chain sized by final demand (PR B of stage-2-chain)

**Status: Proposed, stage 1 only (the plan lives in git history on `stage-2-chain-b`, not for merge).** Sequences the supervisor task `stage-2-chain-payroll-chain.md` (option (a) of N-352, approved by the owner on 2026-11-12), `doc/specs/spec-viability.md` 5 to 8 and 10.3 (VIA), `doc/specs/spec-reference-chain.md` (REF, FR-REF-007, 041, 042), `doc/specs/spec-employment.md` 24 (the payroll form, FR-EMP-064 to 080, ADR 0011) and `doc/specs/spec-stability.md` 8.4 (STY). Built against `architecture.md` 2.9, 2.10, 3.5, 9 and `testing.md` 6, 7.1.1, 11. Number: 2-29 follows plan 2-28 (payroll module, PR A).

Rule tags: architecture=c4fe78572c8c450e187ee13c, observability=e7cfbd33b1620a09dc78dff7, testing=89c3de8888cdbfc9684e8240. Notes tag: f5fc7249054887b83ff9d192 (N-352 read in full and the tail of `doc/notes.md`; the whole-file read with apply/reject of every `VIA`, `REF`, `STY`, `EMP`, `DEM` and `ALL` note is Phase 1's first deliverable, because only Phase 1 changes a spec).

**Precondition.** PR A (`origin/stage-2-done-employment`: `Modules.Employment`, `Role.Hireable`, `Role.Employer.PayrollBP`, ledger reason `wage_pay`, event `PayrollPaid`, position P4.1b, overlay `testdata/chains/overlay/employment.toml`, ADR 0011) is merged into `main`. PR B starts only then (stacking is not allowed). Phase 0 re-reads the merged names; if PR A changed one, this plan's names follow it.

## Evidence this plan stands on (all in N-352 and the supervisor scratchpad `s2/`)

| Finding | Source | Consequence for this plan |
|---|---|---|
| E2: census and warm start re-solved by `SolveSteadyState` under payroll gave 0 of 4 seeds; cooks fell from 211 to 44 to 48 households and held 67% to 83% of the money | `impl-employment-c.md` | The solver gives the wrong answer when the module is on. Phase 2 and 3. |
| The solver's static programme has no payroll: flows `x` and the duals `p`, `w` come from the goods and labour rows alone (`spec-viability.md` 3.2, 5.2) | `viability.go`, `viability_report.go` `income` | Hypothesis H-P below: E2 failed on the prices and wage it seeded, not on the census |
| The five discretionary roles (digger, refiner, maker, comfort maker, treat maker; 204 households in E1) earn about 35 000 a tick from outside their closed chain and spend about 85 000; at the observed 30% to 40% price level their final demand feeds about 100 households | flows analysis, `-screen.flows` | Phase 5 sizes them by final demand |
| E4 (discretionary share cut to 12% by hand, census solved without payroll): every role extinct; `tool_b` price 2.8 to 16 times the reference; 96% of the maker's sales are `tool_b` to cooks. A use of `tool_b` costs price/durability and gives 4 more `prepared` than `prepare_basic`, so it pays only while 4 p(prepared) is at least price/durability: a knife-edge on one price | `DEC-redesign.md` 1.1, `base.toml` (`prepare` 8 outputs, `prepare_basic` 4, `tool_b` durability 8) | Phase 5 recipe change and its static margin gate |
| E5 (wage only to the five discretionary roles): their unfed share did not move (56% to 70%) | `impl-employment-c.md` | Cash alone does not feed them. Credit is not indicated. |
| Payroll alone stops the decline (561 to 580 of 600 households against 466 to 533) but leaves non-cook unfed at 33% to 62% | spike, E1 | Necessary, not sufficient (spec-employment.md 24.7.1) |

## Scope

**In scope.**
1. A payroll-aware `SolveSteadyState`: with `Modules.Employment` on, the solver's prices, wage, `IncomeBP` and warm-start table include the wage flow of the payroll form. With the module off its output is byte-equal to today's.
2. The five discretionary roles sized by final demand, as pack data (a new pack, no code), with a static check that the size is what the solver says.
3. A recipe/data change so `tool_b` demand does not collapse when the sub-chain is thin, with a static margin gate.
4. A trial ladder of at most 6 trials on the harness `scenario_screen_test.go`, the claim of acceptance criterion 2, and the report if it fails.

**Out of scope** (with where it goes): credit (not indicated, N-352; stays Stage 3); the full hire form of `spec-employment.md` 3 to 11 (option (b), 3 to 4 times the cost, a later decision); the four parked engine levers of `origin/stage-2-done-stability` (unmerged); setting or changing any band (`testdata/chains/bands/*`), `x4.*`, `x3.ms_per_tick`, or the bar of `vision.md` 5.1 (acceptance criterion 3: a separate decision the supervisor reports to the owner); PR D of plan 24 (`TestScenarioStable_FiveWarm`, `expected/five.scenarios.txt`; blocked until the claim holds and the owner agrees); porting the result to `five` and `all`, and rewording `vision.md` 5.1 or FR-REF-042 (D6); a new `Config` field, event, metric, position, ledger reason or checkpoint section (none is needed, see D2).

## Decisions and assumptions

**Decided** (from a document, or from the task):
- D1. Payroll is the form of `employment` for Stage 2 (D-O6, ADR 0011). This plan builds on it and does not alter the module.
- D2. The solver reads the payroll from the `Config` it is given (`Modules.Employment` present, `Role.Employer.PayrollBP`, `Role.Hireable`). It takes no new `SolveOption`. Why: the pack and the solver then cannot disagree (the E2 failure was a silent mismatch), FR-REF-007 ("each warm-start document is byte-identical to the table `SolveSteadyState` produces for the layered configuration without it") stays one statement, and the module-off path is unchanged by construction. Reversal: delete `viability_payroll.go` and `internal/steady/payroll.go` and the guarded call (Phase 3 lists it).
- D3. No existing pack changes (`one`, `one-demand`, `five`, `all`, and also `two`, `three`, `famine`, `second`): none turns the module on, so every solver output, warm file, `expected/` file, golden file, pinned hash and checkpoint fixture is byte-equal. The claim runs on a new pack (D4).
- D4. The new pack is `payroll`: the layers `base`, `demand`, `one-demand`, `overlay/employment`, `overlay/chain-payroll`, then its own warm start `warm-payroll.toml` (solver output). It is layered over `one-demand` because the trials, the evidence and the harness use that region document. Existing files are not edited.
- D5. Stop rule and bar from the task: at most 6 trials, three consecutive non-improving trials stop it, the hard stop is a result, the bar is not lowered, no band is set. T1 is the baseline: it counts as the first best, and "non-improving" is counted from T2.
- D6. (Supervisor decision Q1.) If the claim holds on the new pack `payroll`, PR B **reports it and stops there**. It does not port `five` or `all`, and it does not reword `vision.md` 5.1 or FR-REF-042. Porting them and rewording the criterion is a separate owner decision after the claim, taken on the report. `vision.md` 5.1 criterion 1 stays NOT MET in every file PR B touches (N-352 ruling); PR B says "the claim holds on `payroll`", never "criterion 1 met".
- D7. (Supervisor decision Q2.) If the claim fails, PR B merges **only a docs record**: the N-352 update and the roadmap sentence. The solver, the pack, the spec/ADR/architecture changes of Phase 1 and this plan file stay in the branch history of `stage-2-chain-b` and are not merged (a spec that names a solver that was not merged would be a defect, architecture.md 14.2). The record is committed on a branch cut from `main` that holds those two files only, not on top of the Phase 1 to 5 commits.
- D8. (Supervisor decision Q3.) A pass on seeds 101 to 104 with a failure on any of seeds 1 to 10 is **partial**, not criterion 1 met and not a clean claim. The outcome vocabulary and what each one merges is the matrix of Phase 6.

**Assumptions for override** (a sequencing call this plan makes, so it is easy to challenge):
- A1. "Receipts" in the static model are the money a payer receives from trades of its output, net of what it consumes itself (a cook eats its own `prepared` and buys none, `DEC-redesign.md` 1.1), matching `welfare.receiptsOf` (spec-employment.md 24.1.3). Phase 4 tests this against the measured `PayrollPaid.Due`; if it is off by more than 10% the model is corrected before Phase 5, and that is recorded.
- A2. A worker's `k` is 1 per working tick in the packs (weight-1 agents), so wages are in proportion to working household-ticks; the solver takes `k` as 1 and a pack with weight above 1 is rejected by a new Phase 3 diagnostic, not silently mis-solved. (Checked in Phase 0 against `welfare.workRead`.)
- A3. The claim needs seeds 101 to 104 and then seeds 1 to 10 (D-O6 style). The task's criterion 2 names 101 to 104 and says "also run 1 to 10"; a failure on 1 to 10 is "partial" (D8).
- A4. If the claim fails, nothing of Phases 1 to 5 is merged (D7): the precedent is D-O6 (2), "default-off, no consumer, no claim".

## Hypothesis H-P (what the solver must know) and how E2 failed

Payroll moves money, not goods: the executions `x*` at lambda 1 and so the labour of each group and the balanced census (apportioned by need, `spec-viability.md` 5.2) are the same with and without it. What it changes is the **net wage per working tick** that each role is paid, and with free role choice (the spawner reads the observed wage including pay, spec-employment.md 24.2.2) the steady state is where that net wage is equal across roles. The programme's duals `p`, `w` equalise value added per working tick **before** payroll. So at the seeded prices, with a 50% payroll, a cook earns less than a forester and the role review moves cooks out (E2: 211 to 44 to 48), which is the E2 observation. The payroll-aware solution keeps the primal flows and re-solves the duals:

```
payer activity r    (value added v_r(p) = p . n_r):   v_r(p) - beta_g * S_g(p) / L_g   = d_r * w
worker activity r   (group hireable):                  v_r(p) + sigma(p) * d_r           = d_r * w
other activity r:                                      v_r(p)                            = d_r * w
sigma(p) = (sum over payers g of beta_g * S_g(p)) / (sum over hireable h of labour l_h)
```

`beta_g` is `PayrollBP/10 000`, `S_g(p)` the payer group's trade receipts per tick (linear in `p` at the fixed flows, A1), `L_g` its labour, `l_h` the labour of a hireable group (A2). The system is linear in `(p, w)` for the active set of the primal optimum, with the same normalisation `sum p . D = 1`; it is solved in exact rationals (`internal/steady/rational.go`, `math/big` allowed there by architecture.md 3.5), rounded half to even at the boundary like prices and wages (spec-viability.md 6.3). Inactive activities must not pay more than the net wage (condition (c) of 5.1); a violation, or no non-negative solution (a share too high for the chain), is a report diagnostic (`payroll_infeasible`, in the class of 10.3), never a validation error and never a changed census. Outputs that change with the module on: the seed prices (and so the reference prices a pack writes from them), the starting wage, `IncomeBP` (now net of payroll, so FR-REF-041 gates the net income), and the warm-start money and stock that read the seed prices. The census rows do not change. **This is a hypothesis**: Phase 4 is its falsifier and is trial T1 of the ladder.

## Which packs change, and what the owner must agree

| Pack | Changes in PR B? | Why | Owner agreement needed |
|---|---|---|---|
| `one`, `one-demand`, `five`, `all`, `two`, `three`, `famine`, `second` (files, warm starts, `expected/`, `bands/`, goldens, checkpoint fixtures, pinned hash, `examples/realm` pack) | **No, byte-equal** | None turns on `Modules.Employment`; the guarded call is not reached; proved by the differential test of Phase 3 | None |
| `payroll` (new: `overlay/chain-payroll.toml`, `warm-payroll.toml`, its `expected/payroll.toml`, `bands/payroll.toml` created empty of values) | **New** | The claim pack | None for PR B; whether `five`/`all` follow is the separate decision of D6 |
| `five`, `all` re-derived on payroll (new census, prices, warm start, `expected/`, the `all` golden set, the pinned hash of any pack that turns the module on, checkpoint fixtures, the RPG pack of `examples/realm`, PR D) | **Not in PR B, on a pass or a fail** (D6) | Criterion 1 of `vision.md` 5.1 names `five.toml`; D-B of the earlier ruling named `all` | A later owner decision, taken on PR B's report: whether to judge criterion 1 on `payroll` (reword `vision.md` 5.1 and FR-REF-042) or to port `five` and `all` (their goldens, the pinned hash of anything with the module on, `examples/realm` and the `expected/` files move once) |

## Questions answered by the supervisor (were open in the first draft)

- Q1 (which pack is judged if the claim holds): answered, D6. Report only; no port, no reword.
- Q2 (claim fails): answered, D7. Docs record only.
- Q3 (pass on 101 to 104, failure on 1 to 10): answered, D8. Partial.
- Task open question "does any existing pack change under the new solver?": no, D3, proved by the Phase 3 differential.

## Phase 0 (no commit): confirm the preconditions
**Why first:** every later name depends on the merged shape of PR A.
- Read the merged `employment.go`, `internal/employment`, ADR 0011, `spec-employment.md` 24 and `testdata/chains/overlay/employment.toml`; confirm the names above, A1 and A2 (`welfare.receiptsOf`, `welfare.workRead`), and that the harness flags of `scenario_screen_test.go` (`-screen.pack`, `-screen.layers`, `-screen.seeds`, `-screen.ticks`, `-screen.at`, `-screen.census`, `-screen.flows`, `-screen.x4`) are on `main` (they are: PR #57).
- A2 is unverified in this plan: if `workRead` gives `k` above 1 for a weight-1 agent (a batch job), Phase 2's wage share is by `k` times working ticks and the model input gains a per-group mean `k`; Phase 1 states whichever holds, with the line number read.
- **Placement check (architecture.md 2.10.1, 2.10.2, ADR 0010).** The payroll arithmetic is employment-module vocabulary placed in a VIA file. Find where the Demand term of the programme lives (`viaModel.wantsModel`, called from `viability.go` `solveAt` under `Modules.Demand != nil`) and whether `viability_check.go` and `viability_report.go` already read `Modules.Resources` and `Modules.Subsistence` in VIA files; record the file and line. If the precedent is a VIA-owned method, `viability_payroll.go` stands and ADR 0012 cites it as the same kind of hook (a guarded call into a VIA method, no module state). If the precedent is a method defined in the module file, the wiring method moves to `employment.go` (a named hook of 2.10.1, the term in `viability.go`), `employment.go` is then touched, and D1's "does not alter the module" becomes "adds one method". Phase 1 states which, in `architecture.md` 2.6 and the ADR, before any code.
- Confirm that no existing test renders `Report` or `CensusRow` in a way that a new zero-valued field would change (grep `CensusRow` in `*_test.go`; the Phase 3 differential covers the rest).
- Confirm the unit costs: the harness takes about 5.5 s a seed for 2 200 ticks at `GOMAXPROCS=4` (impl-redesign-a), so a four-seed trial is minutes.
**Acceptance:** a one-paragraph note in the Phase 1 commit message listing any renamed symbol. If PR A is not merged, stop.

## Phase 1: spec, ADR, notes, roadmap (docs only, one commit)
**Why first:** a solver rule outside the written spec is a defect (architecture.md 14.2, 2.10); the spec fixes the equations before code, and the FR IDs are what the later tests cite. These commits merge only if the claim passes or is partial (D7); on a fail they stay in branch history.
- Quote the notes tag; read `doc/notes.md` in full (the N-352 "Next owner decision" text, candidate (a), is what this plan executes); for every open note tagged `VIA`, `REF`, `STY`, `EMP`, `DEM` and `ALL` apply, reject (with the reason) or leave open (with what was learned). Add a note for H-P if it survives or is falsified (Phase 4 updates it). Replace the notes tag. N-352 gets a pointer to this work in the same change.
- `spec-viability.md`: a section "5.6 The payroll-aware steady state" with the equations above, A1/A2, the diagnostic `payroll_infeasible` (10.3), the statement that the census and the primal flows are unchanged, and the module-off identity; requirements FR-VIA-064 to 069 (next free ID is 064) and NFR-VIA-005; section 21 (where this lives) names `viability_payroll.go` and `internal/steady/payroll.go`.
- `spec-reference-chain.md`: the variant `payroll` (layers, why `one-demand`), FR-REF-043 (the pack's warm start is byte-identical to the payroll-aware solver, as FR-REF-007), FR-REF-044 (the static tool margin gate of Phase 5), FR-REF-045 (the claim on `payroll` of Phase 6, status "unmet" until it holds; it is about the pack `payroll`, not about criterion 1, D6), and a worked case for the toy model. FR-REF-042 and `vision.md` 5.1 are not edited (D6).
- `doc/adr/0012-the-solver-reads-the-payroll-from-the-configuration.md`: D2 with the alternative (a `SolveOption`) and the E2 reason.
- `architecture.md` 2.6 and `AGENTS.md` Layout: `viability_payroll.go` (VIA) added to the continuation files in both lists and `internal/steady/payroll.go` described, same change, fresh RULE-TAG for `architecture.md` (14.2). `doc/documentation_index.md` rows. `doc/roadmap-v1.md` "Stage 2 status": a sentence that PR B is in progress, no status change; no sentence in any file says criterion 1 is met (D6). `doc/specs/README.md` parameter table: no row (D2: no new parameter).
**Tests:** none (docs). The traceability search of testing.md 4.6 (every new FR ID has a planned test in Phases 2 to 6) is run by hand and the table is in the commit.
**Observability & rollback:** none; revert the commit.
**Acceptance:** spec, ADR, architecture.md, AGENTS.md, index and roadmap agree; every new FR names its Phase; notes tag replaced.

## Phase 2: the pure arithmetic, `internal/steady/payroll.go`
**Why second:** it has no root types, so it is testable alone and can be wrong without touching a pack (architecture.md 2.9).
- `steady.PayrollSolve(model, outcome, payrollInput) (PayrollResult, error)` over primitive structs: the activities with their group index, labour, net vector and receipts coefficient, the per-group `beta`, the hireable flags; returns the re-solved `(p, w)` as exact fractions, per-group `out_g`, `in_g`, net income, and a typed reason when infeasible. Reuses `Rational`/`Frac`, `Apportion`, the tableau helpers and `checked` (extends `internal/steady`, no new package; 3.5 allows `math/big` there).
- No float, no map iteration that affects results, deterministic order by group and activity index (architecture.md 7.5, 7.6).
**Tests (unit, the base of the pyramid; FR-VIA-064 to 067):**
- Toy model with a closed form worked in the test comment (architecture/testing 5.5): two payer-and-worker activities, `beta` 0, 5 000 and 10 000; `beta` 0 reproduces the payroll-free duals exactly (the identity, FR-VIA-068).
- Conservation: sum over groups of `out_g` equals sum of `in_g` exactly (integers/fractions, no tolerance, testing.md 6.3).
- Infeasible share returns the typed reason; a payer with no worker and a worker with no payer return the identity or the reason, as the table of Phase 1 says.
- Property test over random small models (fixed seeds, failing seed printed): the result satisfies the net-wage equations exactly; and a fuzz test (testing.md 7.4) that the function never panics and never overflows silently.
- Mutation run of the file under `gomu` inside Docker with `--cpus` and `--memory` caps (global CLAUDE.md; `testing.md` 9.4), at most a quarter of the host.
**Observability & rollback:** pure function, no logger, no event (observability-and-logging.md 9: "None, because it is arithmetic with no state transition"); delete the file to reverse.
**Acceptance:** `go test -short -race ./internal/steady/...` green; coverage of the file at least 90%; no root file touched.

## Phase 3: wiring into `SolveSteadyState`, default off by construction
**Why third:** needs the arithmetic, and the off proof must hold before any pack uses it.
- `viability_payroll.go` (VIA continuation file): `func (s *viaSolve) payroll(...)`, built from the config (`Modules.Employment`, roles' `Employer` and `Hireable`) and the solved programme; converts to `steady.PayrollInput` inline at the call (architecture.md 2.9) and writes the re-solved prices and wage back before `rc.prices()`, `rc.wages()` and the warm tables read them. One guarded call in `viability.go` (`if s.m.c.Modules.Employment != nil`), no logic in the call (2.10.1). Report rows: appended fields on `CensusRow` (net `IncomeBP` stays in `IncomeBP`; `PayrollOutBP` and `PayrollInBP` new, zero when off and omitted by the renderer at zero, as architecture.md 2.10.5 requires for any appended field) and a `Report.PayrollInfeasible` diagnostic. A weight above 1 on a payer or hireable role (A2) is a diagnostic.
- Reuse: `viaModel`, `regionCtx`, `Report`, the `incomeExceptions` path of FR-REF-041, `steady.Apportion` untouched.
**Tests:**
- **Off proof (hard gate):** a differential test over every pack in `testdata/chains` (`one`, `one-demand`, `two`, `three`, `five`, `famine`, `second`, `all`): the solver report and warm table rendered with the new code equal the existing `expected/` and `warm-*.toml` bytes. `golden/off/`, the pinned hash (`pinnedReferenceHash`), `golden/all/` and the checkpoint fixtures are not regenerated; if any moves the change is wrong (testing.md 7.1.1, 12).
- **On:** the `payroll` identity at `PayrollBP` 1 (a payroll of one basis point changes prices by less than a stated bound, a property); a pack-level test on a two-role config built in Go in the test (config only, architecture.md 9.4); FR-VIA-064 to 069 each cited by a `Requirement:` line.
- No new `Config` field, so the parameter registry test and `hash.go` are untouched; a guard test asserts that `HashConfig` of `payroll` differs from `one-demand` only through the module and role fields of PR A.
**Silent-failure guard:** a `payroll_infeasible` or weight diagnostic must be loud where it is consumed: the scratch emitter and `TestPacks_Payroll` fail on any payroll diagnostic, and the harness prints it, so an infeasible share cannot produce a pack quietly (observability-and-logging.md 7).
**Observability & rollback:** none beyond Report diagnostics; the `Report` is not an event. Rollback: delete the two files and the guarded line.
**Acceptance:** short race green; the differential test green with zero regenerated files; gate F deferred to Phase 7.

## Phase 4: the falsifier of H-P (trial T1 of the ladder)
**Why now:** cheap (minutes), decides whether the model is right before any pack work. It is counted as T1.
- Layers (scratch, outside the repo, a path starting with `/` is allowed by `-screen.layers`): `base, demand, one-demand, overlay/employment`, the E1 data knobs (yield in all seasons, `remove_privation` 40, `change_privation` 12) and `tool_b` durability 24, with the census, prices and warm start from the Phase 3 solver (a scratch emitter test as in `impl-redesign-a`, never committed). `payroll_bp` 5 000.
- Run seeds 101 to 104, 2 200 ticks, `-screen.x4 -screen.flows -screen.census`, `GOMAXPROCS=4`, two runs at a time.
- Read, before looking at anything else (written first, in the commit message of the record): (i) cooks at tick 300 and 600 against their census (E2: 211 to about 100 by tick 300); (ii) prepared/staple price ratio against the solver's; (iii) the measured `PayrollPaid.Due` per tick against the static `S_g` (A1) within 10%; (iv) households at tick 600 against the start.
- **H-P is supported** if (i) cooks stay at least 90% of the census at tick 600 on at least 3 of 4 seeds, and (iii) holds. **Falsified** if cooks still leave (E2 pattern, fewer than 3 of 4 seeds keep 90% of the census at tick 600): the static model is wrong and it is the premise of the whole option. The ladder stops at T1 (one trial used), Phases 5 and 6 are not run, and the record (Phase 6, "claim fails" branch) says H-P is falsified and what the measured cook trajectory was. H-P is judged on the cooks and the receipts only, not on the claim: if H-P is supported the ladder goes on to T2 whatever T1 scored on the claim.
**Tests:** an integration test, report-only first (the measured/static comparison of (iii), `TestScenarioChain_PayrollReceiptsMatchStatic`, with `Requirement: FR-VIA-066`), skipped under `-short`, named with a gate F prefix; it asserts exact facts only (conservation, the module-off identity) and prints the 10% reading.
**Observability & rollback:** test code only; revert the commit.
**Acceptance:** the T1 row of the ladder table filled; H-P supported or falsified in writing.

## Phase 5: pack data: the sub-chain sized by final demand, the tool recipe (data only)
**Why fifth:** data depends on a solver that is known to be right (Phase 4).
- `testdata/chains/overlay/chain-payroll.toml` and `testdata/chains/warm-payroll.toml` (solver output, FR-REF-007, never hand-edited), `expected/payroll.toml`, and `bands/payroll.toml` with no value lines (so no band is set; `TestPacks_BandFilesLoad` must still load it); layer order as D4; the pack is at most 16 documents (`MaxDocuments`, N-352).
- **Sub-chain by final demand.** The want quantities and periods and the tool wear are the only inputs that fix the discretionary census (the solver already sizes the five roles from them: T1 of the redesign gave 121 households with durability 24; wants at half frequency and durability 32 gave 85). The target is about 100 households (about 17% of 600), and the census is whatever the payroll-aware solver balances at that demand: the layer changes **want data and durability, never a household count by hand** (E4 cut the count by hand and died).
- **Tool recipe change (E4).** `tool_b` demand must not depend on `prepared` being above 85% of its reference. Static margin: break-even price of `prepared` for `prepare` against `prepare_basic` is `(tool price / durability) / (yield(prepare) - yield(prepare_basic))`. Gate FR-REF-044 (pack test, config only, no branch in library code): at the solver's tool price the break-even is at most 50% of the reference `prepared` price, and at three times that tool price (the observed E4 spike) at most 100%. Variables the layer may change: `tool_b` durability (8 in `base.toml`; the trials used 24 and 32) and the `prepare_basic` yield (4); the numbers are placeholders (N-015, N-027) and the gate computes them.
**Tests:** `TestPacks_Payroll` (the layers load and validate, the census is the solver's balanced census within the stated bounds, `IncomeBP` of every group at least 8 000 and the spread at most 1.5 as FR-REF-041, FR-REF-043 byte-equality of the warm file on two architectures via the quick run), the static margin gate (FR-REF-044), the configuration-only chain check (testing.md 6.4; no Go specific to the pack). Fuzz corpus gets the new documents (FR-REF-040).
**Pack enumeration check:** grep every test that ranges over `testdata/chains` packs, `expected/` or `bands/` (e.g. `TestPacks_BandFilesLoad`, the band-reading scenario tests) and state, with the file and line read, that a `payroll` pack with no band values neither fails them nor starts a stability claim (criterion 3: no band is set by PR B).
**Observability & rollback:** none; remove the new files.
**Acceptance:** the pack tests green; no existing file changed (`git diff --stat` shows additions only under `testdata/` and tests).

## Phase 6: the ladder and the claim (harness, at most 6 trials)
**Why last:** the claim is evidence about Phases 3 to 5, not a feature.
- **Harness and flags.** `go test -run TestScreen_ -v -screen.pack=<pack or layer list> -screen.layers=<scratch layers, absolute paths> -screen.seeds=101-104 -screen.ticks=2200 -screen.at=100,300,600,1100,1600,2200 -screen.x4 -screen.flows -screen.census`, `GOMAXPROCS=4`, two runs at a time. The harness is already merged (PR #57); a scratch emitter test prints the solver's census and warm start for each layer set.
- **Score of a trial** (decides "improving"): the number of seeds of 101 to 104 that pass the claim; then the least ratio of households at tick 2 200 to tick 600; then the worst role's unfed share of household-ticks (windows 6 and later). A trial improves on the previous best if it raises the first figure, or ties it and raises the second.
- **The claim (acceptance criterion 2), per seed:** households at tick 1 100 and at tick 2 200 each within 10% of the tick-600 value; and no role of the eight with zero households at any tick (without `role_keep` or any floor). It holds when all four seeds of 101 to 104 pass; then seeds 1 to 10 are run with the same two conditions and every one must pass, else the result is partial (A3, D8). Seeds 1 to 10 are a confirmation, not a trial.
- **Also reported, not a gate (criterion 4):** the unfed share of non-cook household-ticks against the spec prior of 1% (STY D01), as a figure per seed; the cooks' share of the money against their share of the households (reading at most 1.5 times); prices against the reference (70% to 130%); X4 (a) to (f); the R3/R6 verdicts stay Unknown because `bands/regimes.toml` has no thresholds. A claim that passes the count and leaves 30% to 40% (or more) of non-cook households unfed is reported as **partial, not criterion 1 met**; between 1% and 30% the figure is reported with the prior beside it and the label is the supervisor's, not this plan's. PR B never writes "criterion 1 met" (D6): that wording is the owner's, after the report. Criterion 3: no band, regime threshold, `x4.*` or `x3.ms_per_tick` value is set by PR B.

**Outcome matrix (what the record says and what merges):**

| Outcome | Condition | Label | Merges |
|---|---|---|---|
| Fail | no trial passes the claim on 101 to 104 within the stop rule | claim fails | docs record only (D7) |
| Partial | passes on all of 101 to 104, and fails on one or more of seeds 1 to 10 (D8) | claim partial | code, pack, spec/ADR, `TestScenarioChain_Claim` over 101 to 104, plus the N-352 and roadmap record naming the failing seeds of 1 to 10; no skipped test; flagged to the supervisor, who may instead choose the docs-only record |
| Claim holds | passes on 101 to 104 and on all of 1 to 10 | claim holds on `payroll`; criterion 4 and criterion 1 labelled separately | same as partial, with the figures; criterion 1 stays NOT MET until the owner decides (D6) |

| Trial | Setting (each re-solved by the Phase 3 solver) | Tests |
|---|---|---|
| T1 | E1 base + payroll 5 000 bp, census/prices/warm from the payroll-aware solver (Phase 4) | H-P |
| T2 | T1 + sub-chain sized by final demand (want data and durability so the solver's discretionary census is about 100) | final demand |
| T3 | T2 + tool recipe change that passes the margin gate (durability) | E4 knife-edge |
| T4 | T3 with the other margin variable (`prepare_basic` yield) in place of the durability change | which lever carries it |
| T5 | best of T1 to T4 with `payroll_bp` 4 000 or 5 500 (E3 showed a narrow band) | the share |
| T6 | one variant chosen from the flows output of T5 (spend on `-screen.flows`: which role still has NoInput refusals) | the last residue |

- **Stop rule.** Stop at the first trial that passes the claim on 101 to 104; stop when three consecutive trials do not improve on the best; hard stop after T6. A stop without a pass is a result, not a failure: nothing is retried, no band or bar is lowered, no `role_keep`-type floor counts.
- **If the claim holds or is partial:** add `TestScenarioChain_Claim` (prefix `TestScenario`, a gate F test, seeds 101 to 104, FR-REF-045 turned from unmet to met in the spec for the pack `payroll` only, with the figures); run seeds 1 to 10 and record them; write the claim and the criterion 4 readings in N-352 and the roadmap truthfully (partial if criterion 4 or seeds 1 to 10 say so); no band is set and no criterion is reworded (the supervisor reports to the owner, criteria 1 and 3).
- **If the claim fails:** the record is a docs-only change cut from `main` (D7: N-352 extended with the rows run, the measured figures and the flows readings; the roadmap sentence) and the branch history keeps Phases 1 to 5, not merged; the report says which of H-P, the sub-chain sizing and the tool recipe were supported or falsified. No claim test is added (a failing test must not be merged; a skipped one needs an issue ID, testing.md 5.7).
**Tests:** the ladder is evidence, not a gate; the claim test is the only addition and only on a pass.
**Observability & rollback:** none; scratch layers live outside the repository.
**Acceptance:** the ladder table in the PR (or the record) is complete for every trial run; the stop reason is named.

## Phase 7: gates and the PR (on a pass or partial, or for the docs record)
- Docs record (claim failed, D7, on the branch cut from `main`): `gofmt -l .`, `go vet ./...`, `staticcheck ./...`, `go test -short -race ./...`, `go mod tidy` (no diff).
- Code (claim holds or partial): the same, plus `go test -race -timeout 60m ./...` (gate R), and **once**, after the last code change, gate F `go test -covermode=set -coverpkg=./... -coverprofile=coverage.out -timeout 120m ./...` and `scripts/ci/coverage-check.sh` (about 45 min on the dev host; one F at a time under the supervisor's lock); coverage not lowered (9.2). Merge `origin/main` first; notes renumbered if a number was taken.
- The PR description carries a `Testing` entry (requirement IDs FR-VIA-064 to 069, NFR-VIA-005, FR-REF-043 to 045 and the test for each, the golden/hash statement "no file moved", the broken-version check of testing.md 9.3, the mutation result) and an `Observability` entry ("None, because the solver is a pure function: no event, metric, log or state transition; its diagnostics are Report fields").
**Acceptance:** all gates exit 0 on the dev host; `golden/off`, the pinned hash and every existing pack file byte-equal.

## Test pyramid (testing.md 2)

| Level | What | Where |
|---|---|---|
| Unit (most) | the net-wage equations on toy models with closed forms, conservation, infeasibility, identity at `beta` 0, fuzz; the margin-gate arithmetic | `internal/steady/payroll_test.go`, `viability_payroll_internal_test.go` |
| Integration | the off differential over all packs, the on case on a Go-built two-role config, `TestPacks_Payroll`, the receipts comparison (Phase 4), the config-only check | root `_test` files, `testdata/chains/` |
| End to end (a handful) | the claim `TestScenarioChain_Claim` (only on a pass or partial); no new golden unless the claim holds and a module-alone golden for the pack is asked for (testing.md 7.1.1 exception for a pack with its own census, as `one-demand`) | `scenario_*_test.go` |

## Security review
No new input or trust boundary: the solver reads a configuration that `Validate` already bounded (PR A: `PayrollBP` 1 to 10 000), computes in exact integers (`math/big` only inside `internal/steady`, rounded and range-checked once at the boundary, architecture.md 3.5), draws no random number, writes nothing but its return value, and starts no goroutine. Worst case: a hostile configuration with a huge model drives the pivot count; the existing `WithMaxPivots` bound (`MaxViabilityPivots`) and `ErrSolveLimit` apply, and the fuzz test of Phase 2 exercises it. The scratch layers of the ladder are read from absolute paths by a test harness that is skipped unless a flag is given. No secret, PII or network.

## Compatibility & rollout
Additive and off by construction: no `Config` field, hash id, event, position, ledger reason, checkpoint section or format version is added, so `golden/off`, the pinned hash and all existing pack files must be byte-equal and the differential test fails if not. `Report` and `CensusRow` gain fields at zero value omitted by the renderer (architecture.md 2.10.5); the public `Report` change is additive (6.6). The new pack is new files. Rollout order: docs, arithmetic, wiring, falsifier, pack, ladder; each phase leaves the tree green and is revertable alone. Risks: (1) H-P may be false (Phase 4 says so cheaply); (2) the working band of the payroll share is narrow and bimodal in the seed (spike, E3): T5 explores it once; (3) the static model of receipts (A1) may be off; (4) deflation is untouched by anything here, so the price-level reading may stay at 40% of reference, which the report must say and which criterion 4 treats as partial; (5) `five` and `all` are not fixed by PR B even on a pass, and `vision.md` 5.1 is not reworded (D6).
