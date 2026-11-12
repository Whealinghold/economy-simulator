# Plan 2-28: a minimal, default-off wage mechanism (payroll) for the reference chain

**Status: Proposed, stage 1 only. Not to be built unless the supervisor says so** (decision D-O5: built only if the pack-data redesign of `vision.md` 5 criterion 1 exhausts its stop rule). Lives in git history on `stage-2-done-employment`, not merged. Memo with the evidence: `DEC-employment.md` (supervisor scratchpad `s2/`). Sequences `doc/specs/spec-employment.md` (EMP), N-352, `architecture.md` 2.6 and 2.10, ADR 0007 and 0010.

## Scope

**In scope.** One optional module, `Modules.Employment`, in the "payroll" form: after clearing, an employer-role agent pays a configured share of that tick's receipts to the households of `Hireable` roles that worked that tick, in proportion to their executions. No state, no draw, no checkpoint section.

**Out of scope** (tracked in `spec-employment.md`, unchanged): the hire, the labour round, escrow, the 12 formation checks, `open_hire`, unemployment presets, wage in kind, standing employment. `credit` stays Stage 3. The pack census/recipe redesign (option A) is a separate plan; this module is necessary but not sufficient for criterion 1 (spike: unfed non-cook share still 33% to 43%).

## Assumptions this plan makes (flagged, not decided)
- A1. The module keeps the spec's names (`Modules.Employment`, `Role.Hireable`, `Role.Employer`, `wage_pay`) so the full hire can later replace the body: **decided by D-O5**; the payroll form itself is **assumption for override**, because the spec has no profit-share wage.
- A2. Share default 5 000 bp is the value that worked in the spike; the module default is a placeholder, the pack sets it.
- A3. Worker proportion is by executions of the tick (`k`), as in the spike.

## Phase 1: spec amendment, ADR, notes (docs only, PR 1)
**Why first:** a module outside the written spec is a defect (architecture.md 2.10, 14.2); the amendment fixes the shape before code.
- Read `doc/notes.md` in full, quote the notes tag, apply or reject every `EMP` and `ALL` note; add a note for the necessary-not-sufficient finding.
- Amend `spec-employment.md`: a section "payroll form" (rule, position P4.1b in `spec-world.md` 6, fields, reasons, the welfare terms `Y_t`/`V_t`, pooled wage exclusion 8.4), FRs with IDs, and mark the hire sections as "later form". Add an ADR (`doc/adr/`) choosing payroll before hire, naming the spike.
- Update `doc/roadmap-v1.md`: row `employment.go` moves from Stage 3 to Stage 2 with the note; parameter-owner table of `doc/specs/README.md`.
**Tests:** none (docs); the traceability check of testing.md 4.6 by text search.
**Observability & rollback:** n/a; revert the PR.
**Acceptance:** spec, ADR, roadmap and README table agree; every new FR has a planned test in phase 3.

## Phase 2: config, validation, hash, pure arithmetic (PR 2)
**Why second:** the default-off proof (`golden/off`, pinned hash byte-equal) must hold before behaviour exists.
- `config.go` append: `Modules.Employment`, `Role.Hireable`, `Role.Employer{PayrollBP}`; `bounds.go` (`PayrollBP` 1 to 10 000), `defaults.go`, `errors.go` (shape, dependency), `validate.go` one line, `hash.go` id appended at the end of the Go order (architecture 2.10.5); `config/` mapping keys.
- `internal/employment`: `Split(payers, workers) []Transfer` (checked arithmetic via `internal/checked`, deterministic order by slot, remainder rule stated), no root types.
- `employment.go`: state struct and `validateEmployment`; no position yet.
**Tests:** unit: table tests of `Split` (conservation `sum out = sum in`, ties, zero workers, overflow) plus a fuzz test (testing.md 7.4); config: one failing case per validation rule (3.9); parameter registry test rows; `golden/off` and pinned hash unchanged (hard gate).
**Observability & rollback:** none beyond validation errors; delete the files and one-liners to reverse.
**Acceptance:** `go test -short -race ./...` green, `golden/off` byte-equal, pinned hash unchanged.

## Phase 3: the payroll step, ledger, welfare, event (PR 3, may merge with PR 2)
**Why third:** needs phase 2 config.
- `employment.go`: `p41bPayroll` (reads receipts and `workRead` of the welfare scratch, pays through `internal/employment.Split`); `tick.go`: one guarded line and `positionList` entry; `spec-world.md` 6 row.
- Reuse: `welfare.go` (`addReceipt`, `workRead`; add `addEarned`/`addWagePaid` beside `addUnearned`), `internal/ledger` registry row `wage_pay` (internal, agent to agent, no boundary), `moveFlow`-style transfer error handling (`invariant`), pooled-wage exclusion hook (spec 8.4).
- `event.go`: `PayrollPaid` (aggregate per region and employer role, count/amount fields only); `doc.go` glossary terms (payroll, employer, hireable); `logkeys` key only if a log is added.
**Tests:** unit for the position with a hand-built two-role world (conservation exact, payer cannot go below balance); integration: reference chain with the module on, invariants of testing.md 6.3 over seeds with `WithAudit`; event golden; non-interference with logger/observer; allocation test of the no-op path (observability 8.2); `TestPositions` table; module-alone golden `golden/employment/` (pack `one-demand` exception of testing.md 7.1.1); every FR cited by a `Requirement:` line.
**Observability & rollback:** event `PayrollPaid` and metric `economysim.payroll.paid` (attributes bounded by roles/regions); `Debug` line per refusal guarded by `Enabled`; observability checklist (section 9) answered in the PR. Rollback: nil `Modules.Employment`; removal of the module restores `golden/off` by construction.
**Acceptance:** all gates (`gofmt`, `vet`, `staticcheck`, short race, gate R, gate F) green on the dev host; coverage thresholds not lowered; `golden/off` unchanged.

## Phase 4: pack overlay and screen (PR 4, or in PR 3)
**Why last:** tuning is data and depends on the module.
- `testdata/chains/overlay/employment.toml` for `one-demand` (cook payer, others hireable, share 5 000 bp), census/warm start re-solved with `SolveSteadyState` if the census moves, `expected/` and checkpoint fixtures re-recorded once (golden explained per testing.md 7.2).
- Calibration with the harness `scenario_screen_test.go`: at most 6 trials (share 4 000 to 5 500, payer set, weight versus executions), seeds 101 to 104 then 1 to 10, 2 200 ticks, GOMAXPROCS=4, two runs at a time.
- Report labour share as wage bill over value added (target reading 55% to 70%, OECD range, a reading not a gate).
**Tests:** the screen as an integration reading; a scenario test `TestScenario_Employment*` asserting no extinct role and households within 10% of tick 600 (no band set, N-027).
**Observability & rollback:** n/a beyond phase 3; revert the overlay.
**Acceptance:** stop rule below.

## Stop rule
Build only if option A exhausted its stop rule. Stop and revert if after phase 4's six trials fewer than 6 of 8 seeds (1 to 4, 101 to 104) keep households within 10% of tick 600 to 2200, or no combination with the best of option A brings non-cook unfed below 20%; record N-352 as data and wage routes exhausted.

## Security review
No new input or trust boundary: configuration is validated (bounds, shape, dependency), payroll arithmetic is checked (128-bit), no network, no persistence, no randomness. Worst case is a mis-set share draining payers: bounded by the payer's free balance and conservation tests.

## Compatibility & rollout
Additive and default-off: `Modules.Employment` nil leaves `golden/off`, the pinned hash and checkpoint format unchanged (no state saved). Hash id appended, nothing renumbered. A `one-demand` golden, `expected/` and checkpoint fixture change once, in phase 4, explained in the PR. Risks: spec departure (phase 1), narrow working band and one seed in eight collapsing (spike), deflation untouched.
