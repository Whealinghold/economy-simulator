# 0012. The solver reads the payroll from the configuration

- Status: Accepted (human review pending)
- Date: 2026-11-12
- Supersedes: none

## Context

The payroll form of the employment module (ADR 0011) moves money from the payer
roles to the `Hireable` roles. `SolveSteadyState` knows nothing of it: its prices
and wage are the duals of a programme with the goods and labour rows alone
(`spec-viability.md` 3.2, 5.2), which equalise value added per working tick
before payroll. Trial E2 of N-352 re-solved the census and the warm start of a
payroll world with that solver and got 0 of 4 seeds: the seeded prices paid a
payer less than a worker and the role review moved the cooks out (211 households
to 44 to 48).

The primal does not change (payroll moves money, not goods), so the census, the
flows and the jobs are as before. What must change is the duals: the net wage per
working tick must be equal across roles (`spec-viability.md` 5.6).

## Decision

1. **The solver reads the payroll from the `Config` it is given**
   (`Modules.Employment` non-nil, `Role.Employer.PayrollBP`, `Role.Hireable`). It
   takes no new `SolveOption`.
2. **The code is a VIA continuation file and a pure `internal/` file.**
   `viability_payroll.go` builds the input from the configuration and writes the
   re-solved duals back; `internal/steady/payroll.go` is the exact arithmetic. One
   guarded call in `viability.go`'s `programme()` holds no logic. The precedent is
   the wants term: `viaModel.wantsModel` is defined in `viability.go` and called
   from `solveAt` under `Modules.Demand != nil`, and the resources module is read in
   `viability_check.go` and `viability_report.go`, so a module's term in the
   solver is a VIA-owned method behind a guard, not a method of the module file
   (architecture.md 2.10.1, ADR 0010). `employment.go` is not touched.
3. **Module-off output is byte-equal by construction**: the guarded call is not
   reached, so every table, report, warm file, golden, pinned hash and checkpoint
   fixture is unchanged. A differential test over every pack proves it.
4. **No new `Config` field, hash id, event, position, ledger reason or checkpoint
   section.** `Report` and `CensusRow` gain fields at their zero value.

## Consequences

- The pack and the solver cannot disagree about the share: both read it from the
  one configuration (E2 failed by that silent mismatch).
- FR-REF-007 stays one statement: a warm-start document is byte-identical to the
  table `SolveSteadyState` produces for the layered configuration without it.
- A programme outside the scope of `spec-viability.md` 5.6 (lanes, a nominal
  seed, a recipe shared with another role) keeps its payroll-free duals and the
  report says `unsupported`.
- Reversal: delete `viability_payroll.go`, `internal/steady/payroll.go` and the
  guarded line.

## Alternatives considered

- **A `SolveOption` carrying the share.** The pack could then be solved with a
  share that differs from the one it runs, which is the E2 failure again; and the
  option would have to be passed by every caller.
- **The logic in `employment.go`.** The arithmetic is the solver's (duals of a
  programme), not the tick's; the module file would import solver internals.
- **Leave the solver alone and tune the pack by hand.** Cannot be reproduced and
  breaks FR-REF-007.

## Open points

- Whether the receipts model (net sales less own consumption) is within 10% of the
  measured `PayrollPaid.Due` is tested by plan 2-29 Phase 4 (FR-VIA-066).
- The scope exclusions may be lifted later if a pack needs them.
