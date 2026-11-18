# Plan: Stage 3, row CRD: `credit.go` and `internal/credit`

**Status: written (2026-11-14), stage 1 of the work (requirements and plan, no code); audited once and revised for that audit (the audit's four blocking and eleven non-blocking items are applied, except where a line below says an owner question holds one open). Branch `s3-crd`, cut from `phase-3` (d996592, which holds the merged plans and first PRs of SKL, POL and STO). Format model: plan-3-sto.md (large, five PRs), plan-3-pol.md and plan-3-skl.md.**
Builds the credit module: the claim list of the ledger put to use. A role with a `Lend` or `Borrow` block, the request and the offer a strategy appends to its plan, the round at P3.3b that matches them at a posted rate, interest fixed at formation, the automatic settlement at P1.7a (the pot, the shield, the pro rata allocation, arrears, the optional write-off), the waterfall at a removal and the passing of a removed creditor's claims to the estate, the slot `SetRateScales`, the call `ForgiveClaims`, the six events, the snapshot rows, the checkpoint section and the neutrality of the module while it is off. It creates no money (FR-CRD-003) and draws no word of the stream (FR-CRD-062). It is off unless `Config.Modules.Credit` is present, and while it is off it leaves no trace (architecture.md 2.10.5).

Rule tags read when the plan was written: architecture=354c9304a17403fb4bd93ae8, observability=f215753306fac0bc7607c4c4, testing=66af7645b01fef4e2796c367. The architecture tag differs from the one plan-3-pol.md quotes (bdb8a4b7fa1ed8fd3b2e7aec): that file was edited after POL was planned, so the section numbers of POL's plan are re-checked, not copied. Notes tag at the end of `doc/notes.md` when this was written: c7b9e596d70af02e60d7d893. The plan edits no spec and no rule file; it names the edits Phase 1 makes, and whoever applies Phase 1 reads `doc/notes.md` in full and quotes its tag (AGENTS.md), because Phase 1 changes specs.

Specs implemented: `spec-credit.md` (CRD) all of it (FR-CRD-001 to 088, NFR-CRD-001 to 005). Specs read and, in Phase 1, edited where the owed-edits table of CRD 21.2 says so: `spec-ledger.md` (LED) 3.3, 7.2, 10 to 13 (the code of the six operations, the three reasons and the audit identities exists since Stage 1: Phase 1 verifies and does not rebuild), `spec-agents.md` (AGT) 3.6, 6.3, 6.4.1, 7, 10.3, `spec-world.md` (WLD) 6.3, 6.5, 6.11, 7.5, 8.2, 8.3, `spec-estates.md` (EST) 2.1, 2.6, 2.7.4, `spec-welfare-and-wages.md` (WEL) 2.1, `spec-observability.md` (OBS) 4.1, 5.6.1, 6.4.2, 7.3, `spec-checkpoint.md` (CHK) 4.2 tag 26, 6.2, `spec-integration.md` (INT) 4.3.2, 4.10, 5.1, 5.3, `spec-config.md` (CFG) 3.1, 4.1, 7, 8.2, `spec-reference-chain.md` (REF) 4.5, 24, scenarios S114 to S118, `spec-institutions.md` 6.6 (the rate limit, an ORG seam: FR-CRD-028).

Notes read (index, then the entries tagged CRD or ALL that bind code): N-107, N-152, N-156, N-164, N-165, N-166, N-167, N-168, N-169, N-187, N-195, N-205, N-209, N-212, N-227, N-319 (CRD; N-152 and N-156 bind no code and stay open, N-187 and N-195 appear only under Out, and item 1.9 records "left open" for each); N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217, N-315, N-335, N-340, N-345, N-351, N-353 (ALL, standing); N-243 (the Lizard limits), N-336 and N-337 (the all-on set), N-347 (a module out of row order appends its Go line and keeps its hash id), N-090 (this module registers no `region_bound` refusal), N-122 and N-124 (first-come caps and the paired comparison: the module takes no draw). `doc/notes.md` was not read in full; Phase 1 item 1.9 does that before any spec edit.

## What is already built (the tree at d996592)

| Piece | Where | State |
| --- | --- | --- |
| The claim record, the six operations (`Lend`, `Settle`, `WriteOff`, `Reassign`, `MarkOverdue`, `Obligate`), the seven totals, `ErrOpenClaim`, `ErrTooManyClaims`, the balance sheet, the audit identities of CRD 3.6, the claim block of the ledger checkpoint section (N-319) | `internal/ledger/claims.go`, `audit.go`, `checkpoint.go` | done at Stage 1 (plan 1-1), with tests citing FR-CRD-004, 005, 008 |
| The three loan reasons `LoanOut`, `LoanRepay`, `LoanInterest` as declared rows (codes 25, 26, 27) | `internal/ledger/registry.go` | declared rows: a ledger has them only when `Config.Declared` names them |
| `newLedger` with the claims limit 0 ("the credit module sets it when it is on") and `moduleReasons` with no credit branch | `ledger.go` | the two touch points the module fills |
| The reserved reason names `loan_out`, `loan_repay`, `loan_interest` for the `ErrReasonCollision` check | `moneyflow.go` line 51 | done |
| The stateless key `K(t, id)` | `internal/ordering` (`Key`) | done |
| `Welfare` scratch accessors `addUnearned` (G_t) and `addInputSpend` | `welfare.go` | done; the interest terms reuse them (Phase 5 reads which) |
| The checkpoint tag 26 row | `checkpoint_sections.go` `notBuiltSections` | listed, no writer |
| The notBuilt entries `Modules.Credit`, `Role.Credit`, the scopes `Borrow`, `Credit`, `CreditTerms`, `Lend` (registry; the Go types are `LendTerms` and `BorrowTerms`, so `scopeAlias` needs two entries, Phase 3) and the events `LoanFormed`, `LoanEnded`, `LoanPassed`, `CreditRound`, `CreditSettled`, `CreditRateScaled` (event catalogue); the positions `P1.7a` and `P3.3b` in `notBuiltPositions` | `registry_test.go`, `event_test.go`, `tick.go` | listed, to be deleted by the change that builds each |
| The sentinels `ErrModuleOff`, `ErrNotOnRole`, `ErrUnknownRole`, `ErrOutsideNode`, `ErrUnknownRegion`, `ErrDuplicateID`, `ErrMissingField`, `ErrModuleDependency` | `errors.go` | exist; `ErrCreditShape` is new |
| The bounds `MaxShockBP`, `MaxMoney`; `MaxCreditRateBP`, `MaxCreditTerm` and `MaxOpenClaims` | `bounds.go` | only the first two exist |

What no merged code names: grep for `CRD`, `credit` and `claim` in the non-test root files of d996592 finds only the reason names in `moneyflow.go` and the layout line of AGENTS.md. STO's and SKL's "credit" is a ledger credit of goods or the premium (SKL 11.4 item 3: not offered), and POL leaves nothing for this row. So the seams below are CRD's own and no other module's merged code has to change, except the named hooks.

## Scope

In:

- `credit.go` (root): the state struct `credit`, `validateCredit` and the warnings of 15.2, the construction call, the two positions, the admission and the round, formation, the settlement, the waterfall and the passing of claims, the view group and the plan checker, the slot and the call, the snapshot rows, the log lines, the checkpoint encoder and decoder, the audit helper. The Lizard limits (CCN 10, NLOC 50, 100 lines, N-243) will force continuation files; the plan names them now so architecture.md 2.6 and AGENTS.md are edited first, in the same change (Phase 1 item 1.3): `credit_round.go` (CRD 5 and 6), `credit_settle.go` (CRD 8 and 9), `credit_strategy.go` (CRD 10, the exported default rules), `credit_report.go` (events, logs, snapshot, warnings of 16 and 15.2), `credit_state.go` (the derived indexes and the checkpoint of 17). Each file's `_test.go` carries that area's `Requirement:` lines (architecture.md 2.6).
- `internal/credit` (new; standard library only; imports neither the root nor `internal/ledger` nor `internal/ordering`: the key is passed in, CRD 24): the interest with a 128-bit intermediate (`internal/checked`), the allocation by the largest remainder, the order and the matching of the round over primitive slices, the waterfall split. The row `internal/credit 90` in `scripts/ci/coverage-thresholds.txt` is added in Phase 2.
- `internal/ledger`: no new operation. Phase 3 adds only what the root needs to read: an accessor for the per-account claim index only if the audit and the derived indexes turn out to need one (read first: `AppendClaims` copies the list; the indexes are root-side, CRD 17.3, so none is expected). No setter is needed: `ledger.Config` already carries `MaxClaims` (`ledger.go:40`, validated at :178, stored at :303) and `newLedger` builds that config.
- The catalogue appends (table below), `config/mapping.go` (`[modules.credit]`, `[role.<id>.credit.lend]`, `[role.<id>.credit.borrow]`), `doc.go` glossary (21.3), `internal/logkeys` (`claim`, `claims`, `rate`, `term`, `arrears`, `principal`, `owed`, `lost`, `lenders`, `borrowers`), the overlay `testdata/chains/overlay/credit.toml`, the golden `testdata/golden/credit/`, the scenarios S114 to S118, an example under `examples/` (CRD 24).
- `INT`: `World.SetRateScales`, `World.ForgiveClaims`, `Directive.Borrow` and `Directive.Lend` (CRD 10.5), the `Credit` group of `AgentState` (CRD 21.2), `SlotState.RateScales`.

Out, with the owner:

- Everything CRD 1 lists as owned elsewhere and every open question of CRD 23: a lender that creates money, a rate that responds, collateral (N-166), credit across regions, early repayment, a traded claim, instalment claims (N-195), initial claims in a warm start (N-165), a profit test, the goods of a removed debtor (N-167). None is built; the seams are named below.
- The insurance module that calls `Obligate` (INS). This row implements the claim of origin `payout` in the settlement, the waterfall and the passing (3.1.5, 8.1.1, 11.1.1), because the ledger holds both origins in one list, and tests it with `ledger.Obligate` planted by the test. INS later appends its own events and counts.
- The rate limit of ORG (`PriceControlled`, FR-CRD-028). The admission step 5.2.3 is a guarded call that reads a rate cap through a function value; with ORG absent the cap is "none". The row builds the call and the counter field, ORG supplies the cap.
- The policy lever of kind `rate` (PLC, reserved, N-205), the tax on interest (FIS, N-187), a treasury that lends.
- The nine metrics of CRD 16.3 and the `otelobserver` instruments: see Q3. The events and the snapshot rows are complete without them, and the derivation test of Phase 8 proves the metrics are derivable (FR-CRD-077).
- The all-modules set (`testdata/golden/all/`, `testdata/checkpoints/format-1-all.*`, `allLayers`): not touched by this row (owner decision of 2026-11-13: it moves once at Stage 3 close). The `credit.toml` overlay is written so it can be layered later in the order REF 4.5 fixes.
- The format version of the checkpoint: the section is appended under a new tag 26, present only when the module is on; the format-1 fixtures do not move (N-318). Phase 1 item 1.6 confirms CHK 9.3 says the same.

## Minimal slice

The spec has one natural cut: **the claims in life** (settlement, the waterfall, the passing, the events and the checkpoint of what exists) before **the claims in birth** (the round, the plans, the default rules). A claim planted by a test through `ledger.Lend` or `ledger.Obligate` is a full claim, so PRs 2 and 3 can be proved in this order with no formation code, and the order also gives the arrears rule (a debtor in arrears is refused) a settled meaning before the round reads it. The five pull requests follow it (below); Q1 asks whether the owner prefers fewer.

## Decisions

D1. **Hash id 15 and the Go order.** `Modules.Credit` is the 15th pointer of the row (`spec-config.md` 3.1; POL's plan counts Tenure 14, Credit 15, Risk 16). Its Go line is appended after the last pointer that exists (`Storage` today), in `Modules`, in `canonical` and in `modulesOn`; its pair goes into `pairsLaterModules` between id 13 and id 19 (the ascending-id rule). The role field `Credit` takes the next free role id: `spec-config.md` 8.2.2 gives ids by struct position and "the next free id of its struct, in the pull request that adds it" (the settlements pair `Needs` and `Ceiling` is told to take its ids when that module is built), and `pairsRole` stops at 24 (`Employer`, hash.go:332), so `Role.Credit` is id 25 and PR 1 adds the row to 8.2.2 (item 1.10). The ids of the `Credit`, `CreditTerms`, `Lend` and `Borrow` fields are their positions in the declaration, read from 3.7.1 in the same item, before `TestCreditHash` is written. Blocks are written when non-nil, even when a field is 0; `Lend.RateBP` 0 is written (it is "interest-free", never defaulted). `Label`-like presentation fields: none. The rate scales are state and are not hashed (FR-CRD-012).
D2. **One position pair, no draw.** P1.7a runs after `p17Flows` and before `p18Walk`; P3.3b runs after `p33aPlanned` and before `p34Estates`. `random.go` and the stream are untouched; the number of words of a tick does not depend on the module (FR-CRD-062; test `TestCredit_NoDraws`). Every tie is `ordering.Key(t, id)` then the ID.
D3. **No per-agent state.** `agentJoined` and `agentRoleChanged` are not hooked (N-128: a claim lives in the ledger's list and a role change leaves every claim, 9.4.1; the admission reads the role at the round). A weight change touches no claim (9.3). `relocationRefusal` gets no row (N-090, 9.4.2); a test pins it.
D4. **The module section is tag 26** and holds the non-default scales (sparse, ascending region), the diagnostic run of "claims overdue" per region (sparse) and the run of "claim cap reached". The claims are saved once, in the ledger's section (17.1, FR-CRD-088). The derived indexes of 17.3 (the due ring, the overdue list, the by-account indexes) are rebuilt after `loadLedger` and the agents are in; the checks of 17.5 run in the fixed order of CHK 6.2 after the accounts and the agents.
D5. **Both origins.** The settlement, the waterfall, `Reassign` at a removal and the headroom cut treat `loan` and `payout` claims alike; `CreditSettled`, the two `Warn`s, `ForgiveClaims` and every instrument count origin `loan` only (3.1.5, N-209). `MaxClaims` counts both.
D6. **Settlement reads a snapshot.** At the entry of P1.7a the balance of each debtor to be visited is recorded and every pot is computed from that record (8.1.1.1, FR-CRD-031); the visit list is built from the due ring and the overdue list, never by a scan of the open claims (NFR-CRD-003).
D7. **Waterfall errors are invariants.** A `Settle` in the waterfall or the pass is cut to the creditor's headroom first (8.4), so a refusal is a violated invariant and uses the `must` pattern of `p17Flows` (plan 2-6 D331). That keeps `removeAgent` free of a new error branch (CCN 10 today).
D8. **Rule constants beside their code** (architecture.md 9.6): the interest formula and its half-even rounding, the allocation and its ties, the order of the checks of 5.2 and 6.4, the matching order, at most one request and one offer per plan, the refusal reasons and the four causes as exported constants of `event.go`, the positions. No default macro constant is added (N-027).
D9. **Default rules are exported functions of the view** (10.4): `DefaultBorrow(view)` and `DefaultLend(view)` in `credit_strategy.go`, called by the default strategy through one guarded call, so a wrapper strategy can call them (architecture.md 4.7).
D10. **Fixtures.** `overlay/credit.toml` is the REF 24 overlay: a borrower role on the 60-tick job role (`grow_staple`, `duration = 60` in `base.toml`) and a lender role in one region, with a warm start of a lender with money. It is layered after `employment` in the order of REF 4.5 and is used by S114 to S118 and the golden. The warm start has no claims (11.4, N-165).
D11. **Existing goldens do not move.** `testdata/golden/off/` and the pinned hash stay unchanged at every phase (N-217); a diff there is a failure of the change, not a golden to regenerate. `testdata/golden/credit/` is new, recorded once. `golden/all/` and the all-on checkpoint fixture are not touched. No existing golden is regenerated by this row.
D12. **Coverage row.** `internal/credit 90`, added in Phase 2 with the package (`coverage-check.sh` fails a package with no row; `selftest.sh` is run).
D13. **Sentinels** are root sentinels matched with `errors.Is` (N-075): new `ErrCreditShape`; reused `ErrOutOfBounds`, `ErrModuleDependency`, `ErrReasonCollision`, `ErrMissingField`, `ErrUnknownRole`, `ErrDuplicateID`, `ErrModuleOff`, `ErrUnknownRegion`, `ErrOutsideNode`, `ErrNotOnRole`, and the ledger's `ErrOpenClaim` and `ErrTooManyClaims` (mapped if the root has no public counterpart: Phase 3 greps `errors.go`).

## Requirement list and where each is verified

Every ID of FR-CRD-001 to 088 and NFR-CRD-001 to 005 appears in at least one `Requirement:` line. Test names are the working names of the phase tables; a test that exists already in `internal/ledger` is cited, not rewritten.

| IDs | Subject | Phase and PR | Main tests |
| --- | --- | --- | --- |
| 001, 012 | on/off, neutrality, hash | 3 (PR 1), 10 (PR 5) | `TestCreditHash`, `TestScenarioS114_OffIsNeutral` |
| 002, 003, 004, 005, 007, 008, 009 | the ledger side | exists (Stage 1); 1 verifies | `internal/ledger` claim tests; `TestCredit_LedgerReasonsDeclaredWhenOn` (3) |
| 006, 083 to 086 | conservation and identities | 5 (PR 2), 11 (PR 5) | `TestCredit_AuditEveryTick`, S116 |
| 010, 011 | block shape, validation | 3 (PR 1) | `TestValidateCredit_Errors`, `FuzzCreditValidate` |
| 013 to 018 | admission | 7 (PR 3) | `TestCredit_Admission_Order`, one case per row of 5.2 |
| 020 to 026 | the round and formation | 7 (PR 3) | `TestRound_*` (unit), `TestCredit_RoundWorld` |
| 027 | interest | 2 (PR 1) | `TestInterest_*`, `FuzzInterest` |
| 028 | rate cap replacement | 7 (PR 3) | `TestCredit_RateCapReplaced` with a stub cap |
| 030 to 037 | settlement, arrears, write-off | 4 and 5 (PR 2) | `TestSettle_*`, `TestCredit_SettlePass`, `FuzzAllocate` |
| 040 to 045 | removal, passing, weight, role, relocation | 5 (PR 2) | `TestCredit_Waterfall`, `TestCredit_PassToEstate`, `TestCredit_NoRegionBound` |
| 050 to 055 | view, default rules, welfare, directive | 6 (PR 3), 8 (PR 4) | `TestCredit_ViewGroup`, `TestDefaultBorrow_*`, `TestDefaultLend_*`, `TestCredit_WelfareInterest`, `TestDirect_CreditFields` |
| 060, 061 | slot and call | 8 (PR 4) | `TestSetRateScales_*`, `TestForgiveClaims_*` |
| 062 | no draw | 5 (PR 2), 7 (PR 3) | `TestCredit_NoDraws` |
| 070 to 079 | events, logs, snapshot, metrics derivable | 5, 7, 8 | `TestCredit_Events_*`, `TestCredit_NoEventsWithoutObserver`, `TestCredit_Warns`, `TestCredit_MetricsDerivable`, `TestCredit_MoneyRebuild` |
| 082 | audit | 5 (PR 2) | `TestCredit_Audit_*` |
| 087, 088 | checkpoint | 5 (PR 2), 9 (PR 4) | `TestCheckpoint_Credit`, `TestFork_Credit` |
| NFR-001, 002, 003 | allocation, cost | 5, 7, 10 | `TestCredit_AllocsPerRun`, `TestStep_AllocationsStayAtTheBaseline` row, `TestCredit_RoundCost` (operation counts, no timing) |
| NFR-004, 005 | benchmarks, fuzz | 2 | `Benchmark*` with `b.ReportAllocs()`, `FuzzInterest`, `FuzzAllocate`, `FuzzWaterfall`, `FuzzRound` |

## Phases

### Phase 1: the documents say what this plan builds (no code; PR 1)

Goal: nothing is owed that the code would then contradict.

| # | File and section | Edit or check |
| --- | --- | --- |
| 1.1 | `doc.go` glossary (architecture.md 3.2, before use) | the terms of CRD 21.3 and the type names the test `TestGlossary_DeclaredTypesNamed` reads: `Credit`, `CreditTerms`, `LendTerms`, `BorrowTerms`, `CreditRequest`, `CreditOffer`, `ClaimFilter`, `ClaimRow`, `CreditTotals`. **Name check (the POL R1 trap):** the root already has an `AgentView` and a `Claim`-free namespace; Phase 1 greps the root for `Credit`, `Lend`, `Borrow`, `Claim`, `Request` and `Offer` as type, field or method names (the plan fields `TradePlan.Offers`/`Takes` of MER sit next to `Lend`/`Borrow`) and records any clash in the PR before Phase 3 builds a type. |
| 1.2 | `architecture.md` 2.6 and 2.10.1, `AGENTS.md` Layout | add `credit_round.go`, `credit_settle.go`, `credit_strategy.go`, `credit_report.go`, `credit_state.go` as CONTINUATION files (area CRD) and `internal/credit/`; the two lists must have the same files; the rule file gets a fresh RULE-TAG in the same change. The supervisor applies this edit if several branches edit the tag (the line always conflicts, plan-3-skl.md). Also the one-line amendment of 2.10.1: its list says the step of a removal is called "as `dropAgent`", which is SKL's (`skills_queue.go:320`); CRD's call is named `dropClaims` (hook 4), so the list names both. |
| 1.3 | `observability-and-logging.md` 3.3.2, 4.3, 5.4.1 | the ten new keys (CRD 16.2), the six events, the attribute `economysim.credit_limit` and the four new values of `economysim.cause`; a fresh tag. If Q3 defers the metrics, only 3.3.2 and 4.3 are edited now and 5.3 and 5.4.1 wait with the instruments. |
| 1.4 | the rows of CRD 21.2 | for each of the 20 rows: check by text search whether it is already applied, apply the ones this row's code needs to be true (AGT 3.6 steps 2d, 2e; WLD 6.3 and 6.5 rows P1.7a and P3.3b, which `tick.go` already lists as not built; CHK 4.2 tag 26; CFG; INT 4.10), and list in the PR the ones left to the PR that builds the behaviour (OBS 7.3 with PR 2 and 3, INT with PR 4, STB with CLB). This depends on Q7: if the owner accepts it, the PR names which of the 20 rows are left to which later PR; if not, all 20 are applied in PR 1 and the PR passes the 3 000-line target, which Q1 and Q7 weigh together, and this plan does not pre-empt either. Each edit is a spec edit: the notes rule binds (item 1.9). |
| 1.5 | the five new notes of CRD 21.1 | add them to `doc/notes.md` if absent (a warm start has no claims; collateral; the goods of a removed debtor; the lender that eats its principal; plan fields), and delete from the index the notes this row applies (N-164, N-209 when PR 3 and 2 land). Fresh notes tag. |
| 1.6 | `spec-checkpoint.md` 4.2, 9.3, `checkpoint_sections.go` | confirm tag 26 "Credit" is present only with the module on, and that a world with the module off writes the bytes of format 1 (the fixtures do not move, N-318). |
| 1.7 | `spec-ledger.md` and `internal/ledger` | check that `Close` refuses an open claim last (3.2.1) and that the audit reads the identities of 3.6 and the totals; `newLedger` already takes `MaxClaims` through `ledger.Config`. A gap is listed in Risks, not fixed here. |
| 1.8 | `spec-world.md` 6.3, 6.5, 6.11 | read the exact event order against 12.1.1 and note any mismatch for PR 2 and 3. |
| 1.9 | `doc/notes.md` | read in full, quote the tag in the first message about this work, apply, reject or leave open each CRD and ALL note (AGENTS.md). |
| 1.10 | `spec-config.md` 8.2.2, 3.7.1 | add the row for the credit module (`Modules.Credit` 15, `Role.Credit` 25, the ids of `Credit`, `CreditTerms`, `Lend` and `Borrow` by declaration order) in the PR that adds the field, as 8.2.2 requires, before `TestCreditHash`. |

Exit check: 1.1 to 1.10 done and recorded in the PR; `go test -short ./...` unchanged; `npx markdownlint-cli2` clean on every changed `.md` (see Codacy hygiene).

### Phase 2: `internal/credit` (pure; PR 1)

Goal: the arithmetic and the matching of CRD 6, 7, 8.3 and 9.1 as functions over primitive slices, proved against the models of `doc/specs/models/crd/` and the worked examples of CRD 20.

Files: `internal/credit/interest.go`, `allocate.go`, `round.go`, `waterfall.go`, `doc.go`, tests, `fuzz_test.go`, `scripts/ci/coverage-thresholds.txt`. API sketch (final here): `Interest(x, rateBP, term, ticksPerYear int64) (int64, bool)`; `Allocate(pay int64, owed []int64, tie []uint64) []int64` (the largest remainder; ties by the passed key then index; no share above `owed_i`); `Order(offers, requests []Row, scaleBP int64) ...` (offers by effective rate, requests by limit descending, ties by the passed key then ID); `Match(...)` (the scan of 6.3.2, with `MinPrincipal`, returning tentative fills and the unfilled remainder, the formation callback taking the reason of a failed fill); `Waterfall(balance int64, owed []int64, tie []uint64) (pay []int64, left int64)`.

Tests (unit; `Requirement:` lines): `TestInterest_Exact` (the 128-bit product at `MaxMoney * MaxCreditRateBP * MaxCreditTerm`; half-even at the midpoint; the result within one unit of the exact quotient) FR-CRD-027. `TestInterest_Overflow` (reported, never wrapped) FR-CRD-027. `TestAllocate_SumsToPay` (property over 500 random cases with a fixed seed per case and the failing seed in the message; no share above its owed) FR-CRD-032, NFR-CRD-005. `TestAllocate_Ties` (equal fractional parts: key then ID) FR-CRD-032. `TestAllocate_ExampleB` (CRD 20.2). `TestOrder_*`, `TestMatch_RateScan`, `TestMatch_MinPrincipal` (the two cases of 6.3.2 step 3), `TestMatch_Discriminatory` (6.3.3), `TestMatch_NoLenderAtThePrice` FR-CRD-022, 023, 024. `TestWaterfall_*` (no shield; pays the claims pro rata; the rest is returned) FR-CRD-040. `FuzzInterest`, `FuzzAllocate`, `FuzzWaterfall`, `FuzzRound` (never panic; shares sum to the payment; interest within one unit; overflow reported) NFR-CRD-005. `Benchmark*` with `b.ReportAllocs()` NFR-CRD-004. A cross-check test against a second formulation (the Python models re-expressed in Go, as POL did for its tree) over 1 000 random rounds.

Exit check: `go test -short -race ./internal/credit` green; coverage at least 90%; imports `internal/checked` and the standard library only (the generic `import_guard_test.go` enforces it); the coverage row added and `selftest.sh` run; `gomu` through Docker only, `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4 --workers 4`, never on the host (the standing rule; the run on one small package is short), survivors listed in the PR; `lizard -l go -C 10 -L 100 -a 8 -w` clean on the package.

### Phase 3: configuration, bounds, validation, hash, loader, the ledger switch (PR 1)

Goal: a configuration with the module loads, validates, hashes and round-trips; the ledger declares the loan reasons and the claims limit when the module is on; with it off nothing moves.

Files: `config.go` (`Credit` with `MinPrincipal`, `ShieldTicks`, `WriteOffTicks`, `MaxClaims`; `CreditTerms`, `LendTerms`, `BorrowTerms`; `Modules.Credit *Credit` with the comment `Its hash id is 15, whatever its place in this struct`; `Role.Credit *CreditTerms` appended; `canonical` copies the blocks through two helpers, `func (m *Credit) canonical() *Credit` and `func (c *CreditTerms) canonical() *CreditTerms`, like `Storage` at `config.go:209`, because `Config.canonical` is at CCN 8, 62 NLOC and 75 lines and an inline copy would pass the limit), `bounds.go` (`MaxCreditRateBP` 100 000, `MaxCreditTerm` 3 650, `MaxOpenClaims` 1 000 000), `defaults.go` (the zero values of 4.1.2 in a comment: no default macro constant), `errors.go` (`ErrCreditShape`; the mapping of `ErrOpenClaim` and `ErrTooManyClaims` if the root has no public counterpart), `validate.go` (one call to `validateCredit` in module order, one to `creditWarnings`; the thirteen warnings of 15.2 in `credit_report.go`), `hash.go` (`pairsCredit`, the role pair, the id in `pairsLaterModules`; a continuation function if CCN needs it), `ledger.go` (`moduleReasons`, at CCN 7, gains a credit branch with the three loan reasons; `newLedger` passes `MaxClaims` in the `ledger.Config` it already builds), `moneyflow.go` (the collision check already names the three reasons: only a test), `config/mapping.go`, `registry_test.go` (`notBuilt` loses `Modules.Credit`, `Role.Credit` and the scopes `Borrow`, `Credit`, `CreditTerms` and `Lend`; `scopeAlias` gains `"Lend": "LendTerms"` and `"Borrow": "BorrowTerms"` beside the `Polity` precedent; `exemptFields` gains `Modules.Credit`), `doc.go`.

The validation order is the spec's (15.1): the bounds, then the block shape (`ErrCreditShape`), then the module dependency (`ErrModuleDependency`), then the reason collision, then the missing fields (`ErrMissingField` for `rate_bp` and `max_rate_bp` absent from a file). Errors name the path, for example `role.borrower.credit.borrow.max_debt_ticks`.

Tests: `TestValidateCredit_Errors` (one case per row of 15.1, each breaking only its rule, `errors.Is`, the path) FR-CRD-010, 011. `TestValidateCredit_Order` (two mistakes at once) FR-CRD-011. `TestCreditWarnings` (one case per warning of 15.2: the condition and the quiet case; none fails validation; each is raised once with `rule` and `path`) FR-CRD-076 (the construction-time half). `TestCreditHash` (off: the pinned hash unchanged; on: scales not hashed; the blocks in canonical order; the id after 13 and before 19; an empty non-nil `Credit` differs from nil; a change of a rate, a shield or a role block changes the hash) FR-CRD-012. `TestLoad_Credit` in `config/credit_test.go` (the overlay loaded from TOML equals the Go-built configuration, same hash; a missing `rate_bp` is `ErrMissingField`; `rate_bp = 0` loads) FR-CRD-011, 012. `TestCredit_LedgerReasonsDeclaredWhenOn` (the three reasons are declared with the module on and refused with `ErrReasonMismatch` with it off) FR-CRD-002. `FuzzCreditValidate` (root) and `FuzzLoadCredit` (`config/`, corpus under `config/testdata/fuzz/FuzzLoadCredit/`).

Exit check: `Validate`, `HashConfig`, the loader tests green; `TestHashConfig_ReferencePinned` and `testdata/golden/off/` unchanged; the position test and the event catalogue test unchanged (no position or event yet); `go vet`, `staticcheck`; this is the end of PR 1.

### Phase 4: state, derived indexes, the settlement position P1.7a (PR 2)

Goal: a claim planted by a test is settled at its due tick, with the shield, pro rata and in arrears; nothing forms a claim yet.

Files: `credit.go` (the state struct and `newCredit`), `credit_state.go` (the due ring of at most `MaxCreditTerm + 1` buckets, the overdue list, the by-account indexes, rebuilt from `AppendClaims`; the per-agent derivations `Debt`, `Owed`, `Overdue`, `NextDue`, `CreditRoom`, `Lent`, `Receivable`), `credit_settle.go` (`p17aSettle`), `world.go` (`credit *credit`), `construct.go` (`w.credit = newCredit(&w.cfg)` beside the other module constructors; `modulesOn` gets `"credit"`; the summary gains `lenders` and `borrowers`), `tick.go` (hook 1 below, with the new function `positionsFlows`, P1.7 to P1.9, between `positions` and `positionsProduce`; the comments of `positions`, `positionsProduce` and `positionsTail` that say "in source order" are edited to name it; `positionsSettle` already exists and is not reused), `tick_internal_test.go` (the two lists at lines 79 and 92 that hard-code the five function names gain `positionsFlows`, in the filter and in the hand-over exclusion), `poshash_internal_test.go` (the `replayRows` table: a row `{id: "P1.7a", run: noErr((*World).p17aSettle), on: creditOn}` after `P1.7`), `event.go` (`LoanEnded`, `CreditSettled`, the four causes `repaid`, `written_off`, `debtor_removed`, `forgiven`), `welfare.go` callers (hook 5), `event_test.go` (the two events leave `notBuiltEvents`), and, in this phase, the observer lists: `observe_events_test.go`, `observe_internal_test.go`, `otelobserver/adapter_test.go` (`ignoredEvents` while Q3 defers the instruments), `otelobserver/events_test.go` (`allEvents`) and `otelobserver/testdata/all_events.txt`, which `catalogue_otel_copy_test.go` requires to equal the `Event` types of `event.go` (the precedent is `dec3bc4` of SKL), with the `otelobserver/` gates run in this PR.

Hooks of this phase: see the table in "Hook discipline".

Tests: `TestSettle_Pot` (the shield; `pay = min(max(0, b - s*U), owed)`; the balance read at the entry for every debtor, so an agent that is both creditor and debtor has the same pot whatever the order) FR-CRD-031. `TestSettle_Allocation` (pro rata, interest first, the largest remainder; CRD 20.2) FR-CRD-032. `TestSettle_Headroom` (a payment cut at a creditor near `MaxMoney`, the cut counted) FR-CRD-033. `TestSettle_Arrears` (`MarkOverdue` at the due tick, the claim paid on each later tick, no penalty, no interest on arrears) FR-CRD-034, 036. `TestSettle_WriteOff` (with and without `WriteOffTicks`) FR-CRD-035. `TestSettle_BeforeAnyRequest` (a due claim is settled before a request of the same tick; structural: the position order) FR-CRD-037. `TestSettle_AwayDebtor` (a trip or a journey does not skip the payment) FR-CRD-045. `TestSettle_BothOrigins` (a payout claim planted with `Obligate` is settled by the same pot; `CreditSettled` counts loans only) D5. `TestCredit_EventsPhase1` (the order of 12.1.1: `LoanEnded` ascending then `CreditSettled` per region, after `MoneyFlowLimited` and `MoneyFlowEnded`; never emitted with all counts 0; `Shielded`, `Headroom` and `Arrears` fields) FR-CRD-071, 073, 074. `TestCredit_WelfareInterest` (interest paid is netted from `income_t` and kept out of the earned ratio; interest received is in `G_t` and out of the observed wage; the principal is neither) FR-CRD-054. `TestCredit_SettleCost` (the number of visited debtors and claims is the due and overdue ones, not the open claims: an operation counter) NFR-CRD-003.

Exit check: `testdata/golden/off/` and the pinned hash unchanged; the position test sees `P1.7a` in `positionsFlows`, and `positions` is measured below CCN 10 after the split; coverage not down; `lizard` clean on every new function.

### Phase 5: removal, passing, audit, checkpoint tag 26 (PR 2)

Goal: a removal pays its creditors first and passes the claims it holds to the estate; the section is saved and restored; a restore settles as the original.

Files: `credit_settle.go` (`dropClaims`: steps 2d and 2e), `fanin.go` (hook 4), `credit_state.go` (`saveCredit`, `loadCredit`, the checks of 17.5, the rebuild of the indexes), `checkpoint_sections.go` (a row for tag 26; the entry leaves `notBuiltSections`), `checkpoint_save.go` and `checkpoint_restore.go` (the one call each), `credit_report.go` (`LoanPassed`, the "claims overdue" and "claim cap reached" `Warn`s with their runs saved in the section), `event.go` (`LoanPassed`), `moneyflow.go` or `credit.go` (`auditModules` call: the identities of 3.6 are the ledger's; this adds only that every open claim names open accounts of the kinds of item 3 and that the indexes equal the list), `event_test.go` (`LoanPassed` leaves `notBuiltEvents`; `notBuiltFields` needs no entry because no existing event row gains a field), and the observer lists of Phase 4 for `LoanPassed`.

Tests: `TestCredit_Waterfall` (a removed debtor's whole balance pays its claims with no shield, then the rest goes to the estate under `estate_money`; each cause of a removal and `RemoveAgent`; a reduction of weight runs no waterfall) FR-CRD-040, 041, 043. `TestCredit_PassToEstate` (a removed creditor's claims are reassigned to the estate of its region, in ascending claim ID, no money moved; both sides in one removal: 2d then 2e) FR-CRD-042, 043. `TestCredit_NoRegionBound` (a debtor with an open claim relocates, travels, changes role, changes weight: never refused; `relocationRefusal` has no credit row) FR-CRD-044 (N-090). `TestCredit_EventsRemoval` (`LoanEnded` then `LoanPassed` then `AgentBankrupted`) FR-CRD-071, 072. `TestCredit_Audit_*` (`WithAudit` passes after every tick over planted claims; a doctored claim fails naming it; an audit changes no state) FR-CRD-082, 084 to 086. `TestCredit_Conservation` (the total of money over non-boundary accounts is unchanged by every claim operation; `M` equals `M_init` at every boundary with no other faucet) FR-CRD-003, 006. `TestCheckpoint_Credit` (a world with open claims, scales and warn runs checkpoints and restores; the section is absent with the module off; every refusal of 17.5 is `ErrCheckpointCorrupt` naming the tag and the claim; a restore with another credit configuration is `ErrCheckpointConfig`) FR-CRD-087, 088. `TestCredit_NoDraws` (the words of a tick are the same with the module on and off) FR-CRD-062. `TestCredit_Observation` (same results with and without logger and observer) FR-CRD-075. `TestCredit_AllocsPerRun` (the settlement and the waterfall allocate nothing beyond scratch) NFR-CRD-001. `TestCredit_AllocsPerRun`, `TestCredit_RoundCost` and `TestCredit_NoDraws` are not `t.Parallel()` (`testing.AllocsPerRun` panics in a parallel test, `checkpoint_test.go:755`) and never skip under `raceEnabled`; a test that must skip is named `TestCost_...` so it carries a gate-F prefix (`gateFPrefixes`, `scenario_helpers_test.go:45`).

Exit check: PR 2 gate; `golden/off/` and the pinned hash unchanged; the checkpoint fixtures of format 1 unchanged; `TestAudit_Contracts` and `TestAuditFailure_MapsAndBracketsTheFailedStep` (the existing audit tests) and the restore tests green; the otelobserver gates green with `LoanPassed` in the lists.

### Phase 6: the plan fields, the view group, the default rules (PR 3)

Goal: a strategy can append a request or an offer; the default strategy of a role with a block appends the rule of 10.2 or 10.3.

Files: `strategy.go` (`CreditRequest`, `CreditOffer`, `TradePlan.Borrow` and `TradePlan.Lend`, appended at the end of the plan; the view group is a pointer field appended to `AgentView`, nil for a role with no block), `credit_strategy.go` (`DefaultBorrow`, `DefaultLend`, `fillCreditView`, `checkCreditPlan`), `strategy_default.go` or the merchant seam (hook 6, see its row: a call through the view, not a method of `*World`), `fanin.go` (hooks 2 and 3), `agent.go` (`Directive` is PR 4).

Tests: `TestCredit_ViewGroup` (present only for an agent whose role has a block; the fields of 10.1; no other agent's claim and no lane total) FR-CRD-050. `TestDefaultBorrow_Rule` (the table of CRD 10.2 with `J`, `Late`, `H`, `need`, `ask`: a job in progress, a job to start, `Tier2` as units at the limit prices, the room cap, the `MinPrincipal` floor; the worked example A of 20.1) FR-CRD-051. `TestDefaultBorrow_NoRequest` (no job, no gap, ask below the minimum). `TestDefaultLend_Rule` (`ShareBP` of `max(0, Money - Bids - KeepTicks * U)`) FR-CRD-052. `TestDefault_NoBlockNoPlanFields` (a role with no block, a world with the module off) FR-CRD-053. `TestCredit_PlanAtMostOne` (a second request or offer is kept for the round to refuse `duplicate`).

Exit check: the strategy tests green; `testdata/golden/off/` unchanged; the per-step allocation baselines unchanged (the new fields are slices with no allocation when empty).

### Phase 7: admission, the round P3.3b, formation (PR 3)

Goal: the loans form.

Files: `credit_round.go` (`p33bRound`, `admit`, `formLoan`, the lanes), `tick.go` (hook 1b), `event.go` (`LoanFormed`, `CreditRound`, the nine refusal reason constants), `credit_report.go` (the `Debug` lines "loan formed", "request refused", "offer refused"; the `Warn` "claim cap reached" counter), `poshash_internal_test.go` (a `replayRows` row `{id: "P3.3b", run: noErr((*World).p33bRound), on: creditOn}` after `P3.3a`; `tick_internal_test.go` needs nothing more, since `positionsTrade` is already in its lists), `event_test.go` (`LoanFormed` and `CreditRound` leave `notBuiltEvents`), and the observer lists of Phase 4 for the two events, with the `otelobserver/` gates run in this PR.

Tests: `TestCredit_Admission_Order` (one case per row of 5.2 in order; each refuses whole, changes nothing, is counted, and is not an error of `Step`) FR-CRD-013 to 018. `TestCredit_ExposureNotClamped` (a request above the room is refused `exposure`, not cut) FR-CRD-015. `TestCredit_RateCapReplaced` (a stub cap replaces `MaxRateBP`, never refuses, `PriceControlled` counts) FR-CRD-028. `TestCredit_RoundOrder` (lanes ascending; offers by effective rate then key; requests by limit descending) FR-CRD-021, 022. `TestCredit_DiscriminatoryFill` (two lenders, two prices, the lower first) FR-CRD-023. `TestCredit_MinPrincipalFill` FR-CRD-024. `TestCredit_Formation` (`too_many_claims` ends the round; `overflow` drops the fill; the ledger unchanged on a drop; `Due = t + Term`) FR-CRD-025, 026. `TestCredit_RoundMovesOnlyAvailable` (an offer cannot spend what an order reserved) FR-CRD-020. `TestCredit_ScaleApplies` (the effective rate with a scale; clamped to `MaxCreditRateBP`) FR-CRD-024. `TestCredit_NoRenewal` (a due claim is paid at P1.7a before the request is admitted; a debtor in arrears is refused) FR-CRD-016, 037. `TestCredit_EventsPhase3` (`LoanFormed` in formation order then `CreditRound` per region, after `MerchantsPlanned`, before the estates' `OrderPlaced`; `CreditRound` never emitted with all counts 0; `Exposure` and `InArrears` are subsets of `RequestsRefused`) FR-CRD-070, 074. `TestCredit_MoneyRebuild` (an observer that keeps no configuration rebuilds `MoneyHouseholds` and `MoneyPool` through a loan, a payment and a waterfall from the events alone: the three readings of 16.1.3) FR-CRD-079. `TestCredit_Warns` (the two `Warn`s: pattern, keys, counterpart; none per claim) FR-CRD-076. `TestCredit_RoundCost` NFR-CRD-002 (not `t.Parallel()`, see Phase 5). `TestCredit_LoopClosed` (a borrower forms a loan, the claim falls due, the pass repays it, the claim leaves the list, the totals balance: the end-to-end of the two positions on a small hand-built world with planted money and a scripted strategy).

Exit check: PR 3 gate; `golden/off/` unchanged; the allocation baseline rows unchanged off; the otelobserver gates green.

### Phase 8: the slot, the call, INT, the snapshot (PR 4)

Goal: a consumer can set a rate scale, forgive claims, direct a request, and read the claims.

Files: `integration_slots.go`, `integration_directive.go`, `agent.go` (hooks 7 to 9: `fillModuleGroups` is in `agent.go`; `Directive.Borrow`, `Directive.Lend`), `credit.go` (`SetRateScales`, `ForgiveClaims`, `ClaimFilter`), `event.go` (`CreditRateScaled`), `snapshot.go` (`Snapshot.Claims []ClaimRow`, `Snapshot.Credit CreditTotals`, `"credit"` in `Modules`), `credit_report.go` (the rows; deep copies bounded by `MaxClaims`), `errors.go` (nothing new), the `registry_test.go` entries for `ClaimFilter`, `event_test.go` and the observer lists of Phase 4 for `CreditRateScaled`.

Tests: `TestSetRateScales_*` (the errors in the order of 13.1, the event per changed region ascending, applied from the next P3.3b to new loans only, saved, not hashed, 0 is interest-free) FR-CRD-060, 012. `TestForgiveClaims_*` (loans only, never a payout claim; the filter by region and role at the call; ascending claim ID; no money moved; `LoanEnded` per claim at the call carrying the tick of the next step; a filter that matches nothing returns 0; the identity holds with the lost totals raised) FR-CRD-061, 071. `TestDirect_CreditFields` (the directive's request replaces the fallback's; `ErrModuleOff`, `ErrOutOfBounds`, `ErrNotOnRole`) FR-CRD-055. `TestCredit_Snapshot` (`Claims` in ascending claim ID with every field of 16.4; deep copy; a restored world's snapshot equals the original's) FR-CRD-078. `TestCredit_AgentState` (the claims of one agent in `AgentState`, the group of 10.1 without the block). `TestCredit_MetricsDerivable` (each instrument of 16.3 recomputed from the events and the snapshot rows by a test-side folder, and the `outstanding` seed from a restored snapshot) FR-CRD-077.

Exit check: PR 4 gate; `otelobserver` module gates green (the lists and `testdata/all_events.txt` gained each event in its own phase, so this phase adds `CreditRateScaled` and the file now holds all six; the adapter code is untouched unless Q3 says build the instruments).

### Phase 9: the fork test and the checkpoint with open claims (PR 4)

`TestFork_Credit` through `forkAtN` with `credit.toml` on the chosen pack, at N in {0, a tick before a due, the due tick, after a `ForgiveClaims` call between ticks}: the original and two restores give the same events, snapshot, audit and checkpoint bytes. `TestCheckpoint_CreditRoundTripOpenClaims`. FR-CRD-087.

### Phase 10: overlay, golden, scenarios (PR 5)

Files: `testdata/chains/overlay/credit.toml`, `testdata/chains/expected/` entries, `scenario_s114_test.go` to `scenario_s118_test.go`, `testdata/golden/credit/{<pack>.txt,events-<pack>.txt,<pack>.positions.txt}`, `TestDeterminism_Credit` taking cases from `detCases(true)`, the new row of `TestStep_AllocationsStayAtTheBaseline` (a reviewed change, testing.md 8.2), an example under `examples/` (a player that borrows).

The module-alone golden and the reference-chain limitation. The roadmap closed Stage 2 with criterion 1 unmet: the packs `one`, `two` and `three` collapse (and `five` and `one-demand` go extinct); only `payroll` with its module keeps a population, and on the other packs every long job ends in starvation before 120 ticks (the SKL B1 finding R1). A grower with a 60-tick job is exactly the borrower of CRD-S1, so on a collapsing pack the borrowers may be removed before any loan falls due, and the waterfall, not repayment, is what a golden on `one` would record. The plan therefore does not fix the pack: Phase 10 starts with a probe, a throwaway script (not a committed test: a committed test that fails cannot be the fallback) over seeds 1 to 4 and the packs `one`, `five`, `payroll` and `two` with `credit.toml` layered, which reports whether the pack records, by tick 400, at least one loan formed, one claim repaid in full and one defaulted-or-waterfalled claim; the first pack that does is the golden's, and the choice and the probe result go in the PR. `one-demand` (the pack EMP used for its golden, `testdata/golden/employment/one-demand*`) is left out because it goes extinct like `five`, so a probe over it adds no case that `five` does not cover. If no pack does, the golden records the collapsing run on `one` (short, honest, with the audit on); its `credit/` events are expected to show `LoanEnded` with the cause `debtor_removed` (the waterfall) and possibly `written_off`, and `repaid` only if a loan falls due before the removal; the repayment path is proved by the planted-state tests of Phases 4, 5 and 7 and the scenario S114 asserted on invariants only (Q4 asks the owner to accept this fallback in advance, and this plan does not pre-empt it).

Tests: `TestScenarioS114_ColdAndWarm` and `S115` (CRD-S1: the 60-tick job role with credit on and off, cold and warm; exact checks: identities and conservation; the rest reported: unfed household-ticks, relief paid, defaults, `Unfilled`, the pulse of `CreditRound` against the job cycle) FR-CRD-083. `TestScenarioS116_Invariants` (CRD-S3: `WithAudit` over seeds 1 to `scenarioSeedCount()`, exact equality) FR-CRD-083 to 086. `TestScenarioS117_CohortBorrower` (CRD-S4: a borrower cohort whose weight falls by the ladder: debt per household, the waterfall at its removal; a claim is not split, 9.3) FR-CRD-044. `TestScenarioS118_CallsAndCheckpoint` (CRD-S5: `ForgiveClaims` and `SetRateScales` between ticks and a checkpoint with open claims restored) FR-CRD-060, 061, 087. `TestScenarioS114_OffIsNeutral` (the same pack with `credit.toml` carrying no block against the pack without it: every event, snapshot field and golden line identical except the configuration hash, `Snapshot.Modules`, the construction line and the credit warnings) FR-CRD-001, 062. `TestGolden_Credit` and `TestGolden_CreditEvents` (recorded once with `-update`; the diff against `golden/off/` read and explained in the PR) FR-CRD-001. `TestGolden_CreditOffUnmoved` (the `off/` files and the pinned hash equal the committed ones) Guard (N-217). Names carry the gate-F prefixes (`gateFPrefixes`, `scenario_helpers_test.go`), the scenario tests use `scenarioSeedCount()`, and a skip under `raceEnabled` passes `TestRaceSkips_HaveAnFPrefix`; the cost tests of Phases 5 and 7 are not parallel and do not skip.

Exit check: golden `credit/` recorded once and explained; `golden/off/` and `golden/all/` unchanged; the fork test green.

### Phase 11: the exit gate (each PR)

`gofmt -l .` empty; `go vet ./...`; `staticcheck ./...`; `go test -short -race ./...`; `go mod tidy` no diff; Lizard on every changed Go file; `npx markdownlint-cli2` on every changed `.md`; coverage at least 90% for the root and `internal/credit`; `gomu` on `internal/credit` in Docker with the caps of Phase 2 (never on the host); registry, catalogue, cardinality, glossary, position and checkpoint-section tests with this module's entries deleted from the not-built lists in the PR that builds each; the gates R and F as AGENTS.md states them (`scripts/ci/check.sh`, `full.sh`); from `otelobserver/` the same module gates. No arm64 run per PR: arm64 runs once, before Stage 3 closes, on the final `phase-3` head (owner decision of 2026-11-13).

## Hook discipline (approval by the owner needed for each)

A hook holds no branch on the module's data and no arithmetic (architecture.md 2.10.1); each is one guarded call into a method that `credit*.go` defines. Lizard figures are measured at d996592.

| # | File and function | The one call | Phase | Note |
| --- | --- | --- | --- | --- |
| 1 | `tick.go` `positions` (after `w.p17Flows()`) | `if w.credit != nil { w.p17aSettle() }` | 4 | `positions` is at CCN 10 today (measured): one more branch is 11. The established remedy is a split (plan 2-14 O7: `positionsProduce`, `positionsTrade`, `positionsSettle`, `positionsTail` exist for this): Phase 4 moves P1.7 to P1.9 into a new function `positionsFlows` in source order between `positions` and `positionsProduce`; its doc comment says so, the comments of `positionsProduce` and `positionsTail` that say "in source order" are edited, and `tick_internal_test.go` (lines 79 and 92) and `poshash_internal_test.go` (the `replayRows` table) are edited in the same phase. The straight-line shape changes, so this one needs the owner's approval (Q2). |
| 1b | `tick.go` `positionsTrade` (after `p33aPlanned`, before `p34Estates`) | `if w.credit != nil { w.p33bRound() }` | 7 | CCN 7 today: room. |
| 2 | `fanin.go` `fillTradeOnly` | `if w.credit != nil { w.fillCreditView(v, a) }` | 6 | the helper exists for trade-only fillers (CCN 3). The group is built at the start of phase 3 from the ledger list. |
| 3 | `fanin.go` `checkTradePlan` | `if w.credit != nil { w.checkCreditPlan(a, p) }` | 6 | CCN 3 to 4. If Phase 6 finds `checkMerchantPlan` collects Offers and Takes for the round at this seam, the credit call collects the request and the offer here; otherwise the collection is hook 6. |
| 4 | `fanin.go` `removeAgent`, after `dropContracts` and before `estates.receive` | `w.dropClaims(a)` (nil-receiver-safe like `dropAgent`, no return value) | 5 | `removeAgent` is at CCN 10 and 38 NLOC: a guarded call is CCN 11, so the call is unguarded and nil-safe by the precedent of `dropAgent` (`fanin.go:92`, defined at `skills_queue.go:320`). The name is `dropClaims` because `dropAgent` is SKL's and a second one would clash; item 1.2 amends the list of architecture.md 2.10.1. Steps 2d and 2e. |
| 5 | `credit_settle.go` calling `welfare.go` `addUnearned` and `addInputSpend` | none in a core file: the accessors exist | 4 | If the interest-paid term needs a new accessor (the finance cost, nets like an input), it is one line in `welfare.go` that holds no logic (architecture.md 2.10.1 names `addUnearned` as the precedent). Phase 4 reads which; the owner approves a new accessor only if needed. |
| 6 | `strategy_default.go` `DefaultStrategy.Trade` (and `TradeWith`) | `if v.Credit != nil { DefaultCredit(v, p) }`, appending the request or the offer | 6 | `Trade(v *AgentView, rng, p)` (`strategy_default.go:142`) is a value method over the view with no `*World`, and `DefaultStrategy` is a public extension point, so a `w.credit` guard cannot be written there. The guard is the nil view group (D9) and the call is the exported function of `credit_strategy.go` (`DefaultBorrow` and `DefaultLend` behind one `DefaultCredit`), the shape ADR 0010 and architecture.md 2.10.1 allow (a guarded call through the view into a function of the module file). It draws no word (FR-CRD-062) and `Trade` is at CCN 3 today, so there is room. This is a new hook shape and goes to the owner for approval as such. The merchant default is in `contract.go` (`checkMerchantPlan`, `fillMerchantView`), not here, so Phase 6 reads how that seam is reached before it fixes the PR 3 file list. |
| 7 | `integration_directive.go` `Direct` (line 23, CCN 7) | `if w.credit != nil { ...validate Borrow/Lend... }` through `w.checkCreditDirective` | 8 | CCN checked; validation lives in `credit.go`. |
| 8 | `integration_slots.go` | the slot row `SetRateScales`, the `SlotState.RateScales` field, the call `ForgiveClaims` | 8 | INT rows, new methods on `World` defined in `credit.go`; the file holds the registry row only. |
| 9 | `agent.go` `fillModuleGroups` (line 897; the type `AgentState` is at `agent.go:682` and `World.Agent` at :776) | `if w.credit != nil { w.fillCreditState(st, a) }` | 8 | `fillModuleGroups` is at CCN 9 today (17 NLOC), so one guarded call makes 10, the limit. |
| 10 | `construct.go` | `w.credit = newCredit(&w.cfg)`; `modulesOn` and the summary through `creditSummary` | 4 | the construction call (hook class (d)). |
| 11 | `ledger.go` `moduleReasons`, `newLedger` | the credit branch with three reasons; `MaxClaims` | 3 | the ledger rows of the module are built in `ledger.go` (2.10.1), no rule. |
| 12 | `checkpoint_save.go`, `checkpoint_restore.go`, `checkpoint_sections.go` | one row for tag 26; `rebuildCredit` after the agents | 5 | the section table drives both. |
| 13 | `moneyflow.go`/`credit.go` `auditModules` | the credit audit call | 5 | the audit hook. |
| 14 | `validate.go` | `validateCredit`, `creditWarnings` | 3 | the catalogue call. |

No hook in `agentJoined`, `agentRoleChanged`, `relocationRefusal`, `phase_exit.go`, `estates.go` or `clearing`: D3.

## PR split

The module touches the ledger switch, the tick, the default strategy, the removal, the welfare record, the checkpoint and the integration, so one pull request is too large to review. Five pull requests on `phase-3`, in order (target 1 500 to 3 000 lines each, tests included; the golden of PR 5 is a data file outside the count), each green on its own gates and each leaving `golden/off/` and the pinned hash unchanged:

1. **PR 1, "credit: configuration and the arithmetic" (Phases 1 to 3):** the owed edits (all 20 rows of CRD 21.2 or only those the code needs, by Q7; if all 20, PR 1 passes the target), the glossary, the layout and the rule-file edits, `internal/credit`, the configuration, validation, hash, loader, the ledger switch. About 2 500 lines. The module can be configured and validated and does nothing; this is what INS, ORG and REF need for their keys.
2. **PR 2, "credit: claims in life" (Phases 4 and 5):** state, indexes, the settlement, the waterfall, the passing, the audit, tag 26, `LoanEnded`, `LoanPassed`, `CreditSettled`. About 2 800 lines. Claims are planted by tests; nothing forms them yet. The hook of the `tick.go` split is here.
3. **PR 3, "credit: the round" (Phases 6 and 7):** the plan fields, the view group, the default rules, admission, the round, formation, `LoanFormed`, `CreditRound`, the `Warn`s. About 2 800 lines. After it the module works end to end on a small world.
4. **PR 4, "credit: the slot, the call and the snapshot" (Phases 8 and 9):** `SetRateScales`, `ForgiveClaims`, the directive fields, `AgentState`, the snapshot rows, the metrics derivation test, the fork test. About 1 800 lines. Merge into PR 5 if the diff stays under 3 000 together.
5. **PR 5, "credit: the proof" (Phase 10):** the overlay, the golden, S114 to S118, the determinism and allocation rows, the example. About 1 500 lines of tests plus data.

**Gate economics** (every PR): the full gates (`check.sh` with `R_MODE=full`, then `full.sh` for gate F) run once per PR on the final head, with `GOMAXPROCS=8`, and only one gate run at a time on the host: the root test binary reaches 20 to 40 GB and earlyoom kills it when two share the machine (three gate F runs of POL PR B were killed). Between those runs the author uses `go test -short -race` on the changed packages only. A new test must not keep per-seed snapshots for ten parallel seeds (the S113 lesson); the S114 to S118 tests keep counters, not snapshots. Merging a clean PR into `phase-3` is covered by the standing approval (merge commits, no force-push); `phase-3` into `main` needs a go-ahead.

**Codacy hygiene (every phase, before a PR opens).** The Codacy gate fails on any new issue of Info level or higher (it failed SKL A on twelve MD032 findings, and POL A on two MD060 and one Lizard CCN 15). So: every changed `.md` (this plan, specs, notes, the rule files) passes `npx markdownlint-cli2` with no finding: a blank line around every list, heading and table, MD060 table separators in the spaced form `| --- |`, no double blank line, no trailing space, a language on every fenced block. Every new or changed Go function stays within Lizard CCN 10, NLOC 50 and 100 lines, tests included (SKL B1's test functions exceeded CCN 10 and were checked against the gate): run `lizard -l go -C 10 -a 8 -L 100 -w` on the changed files and split a table-driven test into helpers before it passes. Run both on the final head, not only on the first.

## Catalogue appends and shared files (conflict forecast)

Every Stage 3 module appends to the catalogue files, so a textual conflict is certain and resolved by keeping both appends in module order. Where several branches are open the supervisor applies the edits that always conflict (the RULE-TAG lines, `AGENTS.md` Layout).

| File | What CRD appends | Also touched by | Conflict risk |
| --- | --- | --- | --- |
| `config.go` | `Modules.Credit *Credit` after `Storage`; `Credit`, `CreditTerms`, `LendTerms`, `BorrowTerms`; `Role.Credit` | every module (`Modules`), EMP, INS (`Role.Cover`), ORG (`RateLimit`) | High (same lines) |
| `hash.go` | `pairsCredit` id 15 in `pairsLaterModules`; the role pair | every module; INS (the role pair after it) | High |
| `bounds.go`, `defaults.go`, `errors.go` | three bounds; a comment; `ErrCreditShape` | every module; INS reuses `MaxCreditTerm` | Medium |
| `validate.go`, `config/mapping.go` | `validateCredit`, `creditWarnings`; the three TOML tables | every module | High |
| `event.go`, `event_test.go`, `observe_events_test.go`, `observe_internal_test.go`, `poshash_internal_test.go`, `tick_internal_test.go` | six events, nine reason constants, four cause constants; six `notBuiltEvents` entries deleted | every module; INS (`CoverEnded`, `CoverSettled` follow `LoanEnded`) | High |
| `snapshot.go` | `Snapshot.Claims`, `Snapshot.Credit`, `"credit"` in `Modules` | STO, EMP, SET, INS | Medium |
| `tick.go` | `{"P1.7a","CRD",line}` and `{"P3.3b","CRD",line}`; the `positions` split; two `notBuiltPositions` entries deleted | FIS (P1.7b, right after P1.7a), EMP, SKL, STO, SAV | High: P1.7b follows P1.7a at once; the split changes the function every later module edits |
| `fanin.go` | hooks 2, 3 and 4 | every module with per-agent state; `removeAgent` order of steps 2a to 2g (EMP, LND, INS) | High |
| `world.go`, `construct.go` | `credit *credit`; `newCredit`; `modulesOn`; the summary | every module | Medium |
| `ledger.go` | the credit branch of `moduleReasons`; `MaxClaims` | EMP, LND, FIS, INS (their reasons) | Medium: each is its own line |
| `checkpoint_save.go`, `checkpoint_restore.go`, `checkpoint_sections.go` | the tag 26 row and its calls | every module with a tag | High (the same table) |
| `strategy.go`, `strategy_default.go`, `strategy_view.go` | the plan fields, the view group, hook 6 | STO, LND, SAV (plan fields), MER (built) | Medium; the order of appended plan fields is a contract |
| `agent.go`, `integration.go`, `integration_slots.go`, `integration_directive.go` | `Directive.Borrow`, `.Lend`; `AgentState`; slot rows | EMP, LND, STO, POL, TEC, PLC | High |
| `welfare.go` | at most one accessor (hook 5) | EMP, SAV | Low |
| `doc.go`, `AGENTS.md`, `architecture.md` 2.6, the rule files | glossary; the five continuation files; the keys; fresh RULE-TAGs | every module | Medium; the tag line always conflicts |
| `internal/logkeys` | ten keys | every module | Low |
| `scripts/ci/coverage-thresholds.txt` | `internal/credit 90` | every module with a package | Low (one line) |
| `registry_test.go`, `scenario_owner_test.go`, `testdata/chains/overlay/`, `testdata/golden/` | `notBuilt` entries deleted; scenario S114 to S118 in `scenariosNotBuilt`; new overlay and golden | every module | Low in files, High in the lists |
| `otelobserver/` (`testdata/all_events.txt`, `adapter_test.go` `ignoredEvents`, `events_test.go` `allEvents`, and the adapter if Q3 builds it) | each event in the PR of its phase: three in PR 2, two in PR 3, one in PR 4 | every module | Medium |
| `testdata/golden/all/`, `testdata/checkpoints/format-1-all.*`, `all_pack_test.go` | **none** (owner decision, D11) | every module | not applicable |
| `doc/roadmap-v1.md`, `doc/notes.md` | the CRD row status; notes applied or added, a fresh notes tag | every module | Medium |

## Cross-plan contract

Consumes (identifiers as the tree has them at d996592; re-check at implementation):

- `internal/ledger` claims, totals, the reasons, `Close`'s `ErrOpenClaim`, the audit <- Stage 1 (plan 1-1).
- `ordering.Key`, `internal/checked` (`MulDiv`, `MulDivDiv`, `Acc`) <- Stage 0 and 1.
- `removeAgent`, `dropAgent`, `estates.receive`, the removal event order (spec-world.md 6.11 item 8) <- Stage 1 and plan 2-3.
- `fillAgentView`, `fillTradeOnly`, `checkTradePlan`, `normalisePlan`, the merchant plan seam (`checkMerchantPlan`, `fillMerchantView`, `p33Round`, `p33aPlanned`) <- Stage 1 and plan 2-10 (MER).
- `welfareScratch.addUnearned`, `addInputSpend` <- plan 1-6; `p16Relief`, `p17Flows` <- plan 2-6.
- `forkAtN`, `forkCase`, `checkModuleGolden`, `scenarioSeedCount`, `raceEnabled`, `gateFPrefixes` (the test helpers) <- CHK, REF.
- `Directive`, `AgentState`, `Slots()`, `RegionScale` <- INT, plan 1-9 and 2-11.

Provides:

- `Credit`, `World.SetRateScales`, `World.ForgiveClaims`, the six events, `Snapshot.Claims`, `CreditTotals` -> INS (the payout claim and `CoverSettled` follow `LoanEnded`; N-212: the cover round is after the credit round), PLC (the lever of kind `rate`, reserved, N-205), ORG (`RateLimit`, step 5.2.3 through the cap function), SAV (`CreditRequest` for a project, SAV 6.2.2), IND (interest paid and received are measurable from the events, N-130).
- `ErrCreditShape`; `DefaultBorrow` and `DefaultLend` for wrapper strategies.
- `overlay/credit.toml` -> the cover overlay (`overlay/cover.toml` needs credit or a faucet for the loan that pays a premium, REF 4.5), the later layers of the all-on set.
- The slot of the claim list for collateral later: the claim record has the seam of 3.5.3 (a field and a step), and nothing in this row blocks it.

## Dependencies on other Stage 3 modules

The roadmap row says CRD needs LED, MON, EST and WEL: all built, so CRD can be built now with no Stage 3 module before it (the first-in-order rule is not a dependency here). Soft links, none blocking: **INS** needs CRD (`Obligate` is already in the ledger; INS appends events and a cover round after P3.3b, and a premium can be paid from a loan, N-212); **ORG** reads the rate cap (the function value of 5.2.3 is `nil` until ORG exists); **PLC** (reserved lever); **SAV** (a project request, default asks for none); **EMP** (a role with an employer block and a borrow block borrows for upkeep and `Tier2` only, 11.4; no change needed); **LND** and **STO** (collateral is open question 3: the pledge of room and the title are the later candidates, N-166); **POL**, **SKL** (no link: SKL 11.4 item 3 says a premium is not a loan). Build order follows the roadmap, so INS comes after this row.

## Testing (testing.md 11)

1. Requirement to test: the table above; every FR-CRD and NFR-CRD ID appears in a `Requirement:` line.
2. Lowest level: the interest, the allocation, the order, the matching and the waterfall are unit and fuzz tests of `internal/credit` (the large majority); the ledger side is already unit-tested in `internal/ledger`; validation, hashing and the loader are root and `config/` tests; the settlement, the round, the removal, the restore and the observer rebuild need a world and are integration tests (a few ticks, so most run under `-short`); S114 to S118, the golden and the fork test are end-to-end and carry gate-F prefixes.
3. Golden files: `golden/off/` and the pinned hash do not move (a diff is a failure, N-217); `golden/credit/` is new, recorded once, with the diff against `off/` read and explained (testing.md 7.2); `golden/all/` and the all-on checkpoint fixture are not touched.
4. Fork at N: `TestFork_Credit` (Phase 9), N in {0, before a due, the due, after a call}.
5. Fuzz: `FuzzInterest`, `FuzzAllocate`, `FuzzWaterfall`, `FuzzRound` (`internal/credit`), `FuzzLoadCredit` (`config/`), `FuzzCreditValidate` (root); a crash input gets a `Fix:` comment.
6. Invariants and determinism: `TestInvariants_Packs` (`invariants_test.go`) gains the claim identities and the conservation of money with the module on; `TestDeterminism_Credit` takes its cases from `detCases(true)`.
7. Mutation: `gomu` on `internal/credit` only, through Docker with `--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS=4`, `--workers 4`; never on the host (the user's standing rule). The root files are covered by the tests above and the phase exits "break the rule, see a test fail" for the shield, the interest-first order, the waterfall rank and the `in_arrears` refusal (testing.md 9.3).
8. Gate markers in the PR description: the short race run, R, F, the coverage check, Lizard, markdownlint, the `gomu` summary; amd64 only per PR.

## Observability (observability-and-logging.md 9)

1. New failure path or rejected input: configuration errors are returned at construction; a refused request or offer is a counted reason and a `Debug` line, never an error of `Step` (5.2); a payment cut at a creditor's headroom is survived and counted (`Headroom`); a dropped fill is counted (`TooMany`, `Overflow`).
2. New state transition: a claim formed, ended, passed, a round, a settlement, a scale: six events, plain data with no map or pointer, constructed only under `w.obs.supplied` (NFR-CRD-001; `TestCredit_NoEventsWithoutObserver`).
3. New decision point: the default borrower and lender; inputs are the view's; no `Info`.
4. New loop: the pass and the round are bounded by the debtors due and the agents with a block, `Debug` guarded by `Enabled`; the two `Warn`s are per region and per world.
5. Changed event, metric or attribute: none changed. Six events and the snapshot rows are new; the nine metrics and the two attribute changes of 16.3 wait on Q3.
6. New metric attribute: none until Q3.
7. `Info` or `Error`: none in the tick; the construction summary line gains `credit` among `modules` and the counts `lenders` and `borrowers`.
8. Error logged and returned: none.
9. Tests: `TestCredit_Events_*`, `TestCredit_Warns`, `TestCredit_MoneyRebuild`, `TestCredit_NoEventsWithoutObserver`, `TestCredit_Observation`, `TestCredit_AllocsPerRun`, the cardinality test over the new IDs (claim IDs, regions), the capturing-handler test of observability-and-logging.md 8.1.

Each PR's description carries its `Testing` and `Observability` entry (AGENTS.md "Reporting a change").

## Risks and open items

- **R1. The reference chain collapses (N-352).** Packs `one`, `two` and `three` go extinct, so a golden of the module on them may record borrowers removed before any loan is repaid. Phase 10 probes for a surviving pack and falls back to invariants plus planted-state proofs (Q4).
- **R2. `tick.go` at the Lizard limit.** `positions` is at CCN 10 and `removeAgent` at CCN 10; the plan needs a split of the former and a nil-safe call in the latter (hooks 1 and 4). A third module appending to `positions` will meet the same wall; the split should be made once, in this row or the first row that lands.
- **R3. Spec text owed.** CRD 21.2 lists about 20 edited files and a handoff text the plan has not seen; Phase 1 finds which are applied. Two possible mismatches to read first: the order of the removal steps 2a to 2g between EMP, LND and CRD (CRD 9.1.1 says the supervisor numbers them), and the exact order of events in spec-world.md 6.11.
- **R4. Name clashes.** `Borrow`, `Lend`, `Credit`, `Claim`, `Request` and `Offer` are common words next to MER's `Offers` and `Takes` and the ledger's `Claim`; the glossary and a grep come first (item 1.1).
- **R5. `ledger.Claim` has no per-account index.** `AppendClaims` copies the list; the derived indexes live in the root (17.3) and the audit identities do not need one. Only if the root audit turns out to need an accessor is there a small addition to a Stage 1 package, listed in the PR.
- **R6. Welfare of an indebted household.** The interest term enters `income_t`, so a debtor works more days (CRD-A2, STB-L14) and a lender with no income eats its principal (N-168). Both are hazards the spec records and the scenarios report; no rule here damps them beyond the shield, and the plan does not tune a default (N-027).
- **R7. The shield.** `ShieldTicks` has an interior optimum that depends on the horizon (N-169); the plan sets no value and `credit.toml` uses a placeholder labelled as one.
- **R8. Run time.** S114 to S118 add ten seeds each to gate F (about 45 minutes on the 32-core dev host); the cost is measured on the first run and recorded in the PR, and the seed count drops under `-race`.
- **R9. Payout claims without INS.** The settlement code runs for origin `payout` with no producer in the tree. It is covered by planted claims; a coverage hole there would be a defect of this row, not of INS.
- **R10. Merge order.** CRD, INS and FIS all edit `tick.go` at P1.7a, P1.7b and the positions split, and `fanin.go`'s `removeAgent`. If another module's PR lands first, the split of hook 1 is rebased, not repeated.

## Questions only the owner can answer

Each has a recommendation. The questions file is `/tmp/claude-1000/dp3-crd-questions.md` (outside the repo, not durable; this list is the durable copy).

1. **Q1. Five pull requests, settlement before the round?** Recommend yes: claims in life (PR 2) before claims in birth (PR 3), with PR 4 and 5 merged if they stay under 3 000 lines.
2. **Q2. The `tick.go` split and the nil-safe `dropClaims` (hooks 1 and 4), and the new shape of hook 6 (a guarded `DefaultCredit(v, p)` through the view)?** Recommend approve all three: they follow the precedents of plan 2-14 O7 and `dropAgent`, and the alternative is a Lizard CCN 11 on two functions Codacy would fail.
3. **Q3. The nine metrics and the `otelobserver` instruments?** Recommend defer, as SKL's Q7 did: the events and the snapshot rows are complete, `TestCredit_MetricsDerivable` proves the metrics are derivable, and the instruments land in one adapter PR after Stage 3 with the catalogue rows and the attribute values.
4. **Q4. The pack of the module-alone golden and S114?** Recommend the probe of Phase 10 over `one`, `five`, `payroll`, `two`, with the fallback to the collapsing `one` run accepted in advance (the repayment path proved by planted state), and no edit of an existing pack to make credit look healthy.
5. **Q5. INT additions in this row?** Recommend yes, in PR 4: `SetRateScales`, `ForgiveClaims`, `Directive.Borrow` and `.Lend`, the `AgentState` group (CRD 21.2 says they are owed), as the spec says they are the slot and call this module adds.
6. **Q6. Implement the claim of origin `payout` now?** Recommend yes (D5): the ledger holds both origins in one list, the settlement and the waterfall must not skip one, and INS then only appends.
7. **Q7. The edits of CRD 21.2 (about 20 files)?** Recommend apply in Phase 1 only the rows this row's code needs to be true (AGT 3.6, WLD 6.3 and 6.5, CHK tag 26, CFG, the layout and the rule-file keys) and leave the rest to the PR that builds the behaviour, each with its notes check.
8. **Q8. The rate-cap seam of ORG (5.2.3)?** Recommend build the call and the counter field now with a nil cap function, so ORG only supplies the function.
9. **Q9. Typed IDs in the new `Snapshot.Claims` rows and the new events?** The new event types are free of the "never a typed ID" rule (it binds fields appended to existing rows); the new snapshot slice is empty when the module is off. Recommend typed `ClaimID` and `AgentID` fields, as `Contracts` rows have, and a Phase 1 check against architecture.md 2.10.5 and the golden renderer (the same reading POL's Q9 asked for).
10. **Q10. Defaults.** Recommend no default is set from a source (N-018, N-027): `credit.toml` carries labelled placeholders (`MinPrincipal`, `ShieldTicks` 2, `MaxClaims`, rates) and the PR says so.

## PR 1 result (Phases 1 to 3, recorded when PR 1 was built)

Rule tags read: architecture=354c9304a17403fb4bd93ae8 (replaced by a fresh tag in Phase 1), observability=f215753306fac0bc7607c4c4, testing=66af7645b01fef4e2796c367. Notes tag read: c7b9e596d70af02e60d7d893 (unchanged: PR 1 adds and removes no note).

Phase 1, with the breadth of Q7 (only what the code needs):

- 1.1 done: the terms of CRD 21.3 and the type names are in `doc.go`; the name grep found no clash (`Credit`, `Lend`, `Borrow`, `ClaimFilter`, `ClaimRow`, `CreditTotals`, `CreditRequest`, `CreditOffer` are free in the root).
- 1.2 done: the five continuation files are in `architecture.md` 2.6 and `AGENTS.md`, `architecture.md` 2.10.1 names `dropClaims` beside `dropAgent`, and the rule file has a fresh tag.
- 1.3 deferred to PR 2 and PR 3, which add the keys and events: PR 1 adds no log key and no event.
- 1.4 applied: `spec-config.md` 8.2.2 (hash ids); the other rows of CRD 21.2 are already true or belong to the PR that builds the behaviour (the P1.7a and P3.3b rows of `spec-world.md` are present; OBS, INT and STB wait for PR 2 to PR 4).
- 1.5 nothing to add: the five new notes of CRD 21.1 are N-164 to N-168.
- 1.6 to 1.8 checked: `spec-checkpoint.md` 4.2 has tag 26 for `Modules.Credit`, `Close` refuses an open claim last, the audit has the claim identities, and the event order of `spec-world.md` 6.11 matches CRD 12.1.1.
- 1.9 notes read: the index and every note tagged CRD, and the ALL notes that bind code, not the whole file (see the implementation log).
- 1.10 done: `Modules.Credit` is id 15, `Role.Credit` id 25 and the block fields by declaration order.

Deviations from the plan text:

- `internal/credit` takes the scratch of the allocation as a `Scratch` value (methods `Allocate` and `Waterfall`, a `dst` slice, an `ok` result) so that the settlement allocates nothing once it has grown (NFR-CRD-001), and adds `Interest`, `Effective`, `Pay`, `Split` and `Fit` beside the matching functions; the order is `SortOffers`, `SortRequests` and `Match` with a formation callback that answers `Formed`, `Dropped` or `Halted`.
- The validator reports paths as `roles.<id>.credit.borrow.max_debt_ticks` (the validator's own form), and the loader paths as `role.<id>...`.
- A phase of `validateCredit` that finds an error ends the run, as `validateStorage` does: bounds, then shape, then dependency.

## PR 2 result (Phases 4 and 5, recorded when PR 2 was built)

Rule tags read: architecture=b6524107bea39ff4a416ead3, observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367. `doc/notes.md` was not edited, so it was not read in full and no notes tag applies; the leads below are for whoever next changes a spec or a note.

Built: the state `credit` and the derived indexes (`credit.go`, `credit_state.go`), P1.7a and the steps 2d and 2e (`credit_settle.go`), `LoanEnded`, `LoanPassed`, `CreditSettled`, the causes and the types `ClaimID` and `Origin` (`event.go`, `value.go`), the log lines and the Warn "claims overdue" (`credit_report.go`), the audit of the indexes, tag 26 and its cross-check, the split `positionsFlows` (hook 1), `dropClaims` in `removeAgent` (hook 4) and the summary counts. Nothing forms a claim: the tests plant them through `ledger.Lend` and `ledger.Obligate` and index them.

Deviations and choices the plan left open:

- `internal/ledger` gains `Claim(id)`, the accessor R5 allows, so a pass reads one claim by ID and never copies the list. `internal/credit` gains `NextRun` and `Room`.
- The import of `internal/credit` in the root is aliased `crd`, because the root's type is `credit`.
- The Warn "claim cap reached" and its run `capRun` are saved in tag 26, but the Warn itself waits for PR 3: its trigger, a request cut for `MaxClaims`, is in the round.
- Interest paid is netted from the debtor's income through `addInputSpend`, and interest received is `addUnearned` for an agent creditor (hook 5: both accessors existed).
- `valuePrices` is called at P1.7a when `ShieldTicks` is above 0, so the upkeep cost is the one relief reads at P1.6; it is idempotent in a tick.
- `CreditSettled.Shielded` counts a debtor only when it has a claim of origin loan, so a debtor of payout claims alone adds nothing to it. For a debtor with both origins the figure is the whole debtor's, not split by origin.
- The waterfall also credits interest received to an agent creditor as unearned income.
- A claim whose due tick is a ring away from another's shares its bucket and is skipped by its `Due`: a payout claim of any term is found.

Tests: the claim index, the pot and the shield, the allocation (spec-credit.md 20.2) and its ties by `K(t, claim ID)`, the headroom cut, arrears and the write-off, both origins, the events and their order, the welfare terms, the waterfall at both balances of 20.2, the passing, the audit, the checkpoint round trip, the draw guard and the allocation test.

Test pyramid pass (stage 6): `credit_guards_internal_test.go` covers the invariant guards, the room of a role with no Borrow block and the summary counts; `credit_scenario_internal_test.go` holds two end-to-end tests over 30 ticks of the credit world with planted claims (`TestScenario_CreditClaimsInLife`: repaid, arrears, write-off, a removed debtor and a removed creditor, audit and money conserved every tick; `TestFork_CreditClaimsInLife`: a checkpoint at tick 7 finishes with the original's bytes). They stand in for the real-config scenarios S114 to S118 of PR 5, which still need the round.

Left open: `TestSettle_AwayDebtor` uses the transport module's away flag, but a journey (migration module) is not exercised; the scenario S117 and the fork tests of PR 4 and 5 cover the rest of FR-CRD-045 and 087. The counter test of NFR-CRD-003 is `TestSettle_OneBucketServesTwoTicks` (visits, not timings).

## PR 3 result (Phases 6 and 7, recorded when PR 3 was built)

Rule tags read: architecture=607ad1934df33a08dc5a9569, observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367. No rule file, spec or note was edited, so no notes tag applies.

Built: the plan fields `TradePlan.Borrow` and `Lend`, the credit group of the view (`CreditView`, hook 2), `DefaultBorrow`, `DefaultLend` and `DefaultCredit` (hook 6), the plan check `checkCreditPlan` (hook 3), admission, the round P3.3b (`p33bRound`, hook 1b), formation, `LoanFormed`, `CreditRound`, the nine refusal reasons, the Debug lines and the Warn "claim cap reached". The position P3.3b leaves `notBuiltPositions`.

Deviations and choices the plan left open:

- `DefaultBorrow(v, tier2, duration)` and `DefaultLend(v, bids)` take the figures the default `Trade` knows and the view does not hold (the limits are drawn inside `Trade`, so tier 2 at the plan's limits is read there before the money stack lowers one); `DefaultCredit(v, p, tier2, duration)` is the one guarded call.
- Hook 3 is an unguarded, nil-safe `checkCreditPlan`, so that a request with the module off is a `credit_off` Debug line (5.2 row 1) and the fan-in holds no branch.
- `internal/ledger` gains `OpenClaims()`, so the round ends at the claims limit before the ledger would refuse a loan.
- The rate cap of ORG is `credit.rateCap`, a nil function value; `creditCount.priceControlled` is its counter and no event carries it yet.
- The prices are valued once per tick for the upkeep cost (`creditPrices`).
- `TestCredit_InertOverARun` keeps the empty-module identity; the case with blocks is replaced by `TestCredit_PackRoundRuns`, which holds that the round runs with the audit on. On the pack one with the placeholder block of `creditPackDoc` no request is made (the grower asks for less than `min_principal`): the golden pack and the placeholders are Phase 10.

Left open: the `ErrBalanceOverflow` branch of formation (the borrower's balance at `MaxMoney`) has no test, since no world can fund it; the fold of `TestCredit_MoneyRebuild` covers a region-local loan, a payment to an estate and a waterfall.

## PR 4 result (Phases 8 and 9, recorded when PR 4 was built)

Rule tags read: architecture=5c0416f502ba5a59f7b2ae08, observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367. No rule file, spec or note was edited, so no notes tag applies; the spec text this PR leaves owed is listed below.

Built: the event `CreditRateScaled`, the slot `SetRateScales` (hook 8) with `SlotState.RateScales`, the call `ForgiveClaims` with `ClaimFilter`, the directive fields `Borrow` and `Lend` (rows 1 and 7 of INT 4.3.2, hooks 7), the replacement in `Directed.Trade`, the group `CreditState` of `AgentState` (hook 9), the snapshot rows `Snapshot.Claims` and `Snapshot.Credit`, and the observer lists for the sixth event (`all_events.txt`, `ignoredEvents`, `allEvents`). Every credit event is now declared and `notBuiltEvents` holds none of them.

Deviations and choices the plan left open:

- The directive's `Borrow` and `Lend` are saved in the agent record, after the directive, only when `Modules.Credit` is on (`saveDirectiveCredit`, `loadDirectiveCredit`), so that a sticky request survives a fork; the plan and the spec (CRD 17.4, CHK 4.2 tag 3) did not list them. Nothing is written with the module off, so the bytes of format 1 do not move. Owed spec edits, left for a change that reads `doc/notes.md` in full: `spec-checkpoint.md` 4.2 tag 3 and `spec-credit.md` 17.1.
- `AgentState` carries `HasCredit` and `Credit CreditState` (a value group, with `NextDue` -1 for none) rather than a pointer.
- A boundary read of the credit figures values the upkeep prices into a local slice (`boundaryUpkeep`) and never writes `w.oc.valued`, which P1.6 reads as the last tick's valuation: a call to `Agent` changes no later result (`TestCredit_AgentState`).
- The row 1 check of a directive reads `Config.Modules.Credit`, not `w.credit`, so the restore of a mailbox (which runs the same rules) uses the configuration.
- The metrics derivation test folds the events of a run of pack one with `creditLiveDoc` (the credit tables with a minimum principal of 1) and compares each instrument with the snapshot after every step, including the claims newly overdue (the rows with `Overdue` equal to the tick plus the claims removed in their due tick) and the seed of `economysim.credit.outstanding` at a restore.
- The fork test uses pack one with `creditLiveDoc`, seed 1: the first loans form at tick 58 and 59, the first are due from 66, the main wave of 202 claims at 123. `credit.toml` and the chosen golden pack are Phase 10.

Left open: the instruments of the otelobserver adapter are not built (Q3); the nine metrics are only proved derivable. `agent.go` (655 non-comment lines) and `event.go` (987) were already past 500 before this PR; the PR adds a few lines to each (two fields and one call, one event) and moves none.

## PR 5 result (Phase 10, recorded when PR 5 was built)

Rule tags read: architecture=5c0416f502ba5a59f7b2ae08, observability=9ebd6516273680595f477c95, testing=66af7645b01fef4e2796c367. No rule file, spec or note was edited, so no notes tag applies; the leads below are for whoever next changes a spec or a note (the numbers N-420 to N-423 were not used).

Built: `testdata/chains/overlay/credit.toml`, the module-alone golden `testdata/golden/credit/{one.txt,events-one.txt,one.positions.txt}` (new files, recorded once), `TestGolden_Credit`, `TestGolden_CreditEvents`, `TestPositionHash_CreditAlone`, `TestGolden_CreditOffUnmoved`, `TestDeterminism_Credit`, the row `one+credit` of `TestStep_AllocationsStayAtTheBaseline` (94, 24 above the off world's 70) and of `invariantTable`, the scenarios S114 to S118 (S114 to S118 join `stageTwoScenarios`) and the colony example's borrowing player. `golden/off/`, `golden/all/`, the pinned hash and the checkpoint fixtures are untouched.

The pack of the golden (Q4, the probe of Phase 10): a throwaway test over seeds 1 to 4, 400 ticks, the audit on, with `credit.toml` layered on the warm `one`, `two` and `five` (the payroll pack was not probed: its roles are another set). Every pack records loans formed, repaid in full and removed with a debtor (the waterfall): `one` seed 1 formed 892, repaid 181, written off 596, debtor removed 115. So the golden is pack `one` and the fallback of Q4 is not needed. No existing pack was edited.

What the golden says, read against `golden/off/` (testing.md 7.2): the configuration hash differs (the module is on), the construction warning `interest_rounds_to_zero modules/credit` is new (the placeholder rate of 600 basis points rounds the interest of the short terms to 0; it is the warning of 15.2 doing its job on a placeholder), and the markets, the roles and the region lines differ because the loans change who buys what. The events file holds the module's five events in the golden's aggregate form: 892 `LoanFormed`, 892 `LoanEnded`, 5 `LoanPassed`, 400 `CreditRound` and 71 `CreditSettled`. The position file has 24 lines a tick (the 22 of the off set plus P1.7a and P3.3b).

Scenarios (counters and one-line reports, no snapshots kept per seed, seeds 1 to `scenarioSeedCount()`, 200 ticks, the audit on):

- S114 (CRD-S1): cold and warm, credit on and off. Exact: the audit at every step, no trace with the module off, the identities of 3.6 and `Lent` equal to the sum of the `LoanFormed` principal with it on. Reported: with the cold start every household of the role asks in the first ticks (the pulse of STB-L167): 1 900 to 2 900 loans of 150 to 250 each, all but 3 repaid, interest 45 to 267 in all, and `Unfilled` in the hundreds of millions (the cause was not investigated); in the warm run the first rounds are at tick 58 to 59 and, of 890 to 1 400 loans, 67 to 81% are written off, 15 to 25% repaid and 1 to 14% end by a removal; unfed household-ticks are about 59 000 to 62 000 cold, with credit on or off, and 21 000 to 25 000 warm, where credit on lowers them by 3 to 15% on the seeds.
- S114 off-is-neutral: the module on with no block equals the module absent in every event after `WorldCreated` and every sampled snapshot, except the hash and `Modules`.
- S115 (CRD-S2): a yield shock of 5 000 at tick 60 and 15 000 at tick 120, the pool's relief on (estates module, `PoolRelease` 1 000) and off. Exact: the audit and the identities. Reported: defaults, lost principal 3.3 to 5.8 million and interest, the arrears rows at the end (0 to 61), relief 2.4 to 4.5 million with relief on; relief on lowers the lost principal by 1 to 29% on the seeds (mostly 8 to 23%) and does not remove the write-offs.
- S116 (CRD-S3): the total of money equals its start at every tick, the pair identity and rows hold at every tick, over the seeds.
- S117 (CRD-S4): with `World.Scale` 4 a borrower is a cohort of up to four households; 14 to 133 falls of weight hit a debtor with open claims per seed; every claim that stays open keeps its principal, interest and debtor (a claim is not split, 9.3), and the debt per household rises with each fall (the report line gives the first: on seed 1 an agent of 3 households falls to 2 at tick 59 with 11 395 owed, 3 798 per household before the fall and 5 697 after it).
- S118 (CRD-S5): at tick 100 `SetRateScales` and `ForgiveClaims` on the credit pack; the rows go, no money moves, lost totals rise by what was owed, the identities hold, and the original and two restores agree for 60 ticks (and from a checkpoint at tick 122 with claims open).

The cost: S114 to S118 together took about 45 s in one plain run on the dev host; the gate F cost of the whole root package was not measured here, and the seeds drop to one under `-race` through `scenarioSeedCount()`.

Deviations and choices the plan left open:

- The overlay uses `min_principal = 1` and the figures of `creditLiveDoc` (the labelled placeholders of PR 4's fork test), not the 5 000 of `creditPackDoc`: with 5 000 no request is made at all on pack one (the grower's gap is smaller).
- S115 is CRD-S2 of spec-credit.md 18.3 (the shocks and the relief), as spec-stability.md 2322 to 2326 number it; the plan's Phase 10 text puts the cold-and-warm legs under S114 and S115 and the stability table puts them under S114 alone, so S114 has the four legs.
- The example is a borrowing player added to `examples/colony` (a flag `-borrow`, `pack/credit.toml`, `borrow.go`), not a new example module: a new module would need a new line in `architecture.md` 2.6 and `AGENTS.md` and so a fresh rule tag, a line that always conflicts across the branches in flight. The guard `TestExamples_PackCopiesAgree` counts the colony's pack as 3 files, 2 copies and the example's own layer. The directive of the player is made every tick, because it replaces the role's default request only for the next step (a request of 1, below `min_principal`, while it owes).
- `TestCredit_AllocsPerRun`, `TestCredit_RoundCost` and `TestCredit_NoDraws` of PR 2 and 3 already exist; PR 5 adds no cost test.

Left open (leads, no note written):

- On pack one with the placeholders most loans (60 to 85%) are written off at the shield and the write-off tick: the default borrower asks for the gap of its job and the pack's grower income does not cover the principal over the term. This is the hazard CRD-A1 and STB-L163 record; a default rule or a pack figure that changed it is a decision for a note and the owner (N-027), not for this PR.
- The placeholder rate rounds the interest of the short terms to 0 (the warning `interest_rounds_to_zero`): the golden records the warning; a rate above 600 or a longer term would remove it.
- `TestScenarioS117` reports the debt per household before and after a fall of weight and asserts only that a claim is whole; no spec rule says what a household's debt should be after a fall, so none is asserted.
