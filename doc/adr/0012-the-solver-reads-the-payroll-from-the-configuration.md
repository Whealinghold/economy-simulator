# 0012. The solver reads the payroll from the configuration

- Status: Accepted (human review pending); the hypothesis it was taken on is falsified (see Outcome)
- Date: 2026-11-12
- Supersedes: none

## Context

The payroll form of the employment module (ADR 0011) moves money from the payer
roles to the `Hireable` roles. `SolveSteadyState` knows nothing of it: its prices
and wage are the duals of a programme with the goods and labour rows alone
(`spec-viability.md` 3.2, 5.2), which equalise value added per working tick
before payroll. Trial E2 of N-352 re-solved the census and the warm start of a
payroll world with that solver and got 0 of 4 seeds, with the cooks falling from 211
households to 44 to 48. The cause named at the time was that the seeded prices paid a
payer less than a worker; that is H-P below, which did not survive its test.

The primal does not change (payroll moves money, not goods), so the census, the
flows and the jobs are as before. What changes is the duals: the net wage per
working tick is equal across roles in the static balance (`spec-viability.md` 5.6).
The hypothesis this decision was taken on, **H-P** (with payroll-aware prices and
census the cooks keep at least 90% of the solver's census at tick 600), was
**falsified** (see Outcome). The decision stands for a narrower reason.

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

## Outcome (plan 2-29, 2026-11-12)

- **H-P was falsified in every trial of the ladder (T1 to T6).** The cooks sit at 30% to
  45% of the solver's census at tick 600 on every seed (T1: 78 to 81 of 210; T6: 64 to 83
  of 223), mostly gone in the first 100 ticks. The solver's census is the static
  payroll-aware balance, not the simulation's equilibrium. The measured payroll was 97.5%
  of the static receipts (FR-VIA-066), so the receipts model was not the failure.
- **What the payroll-aware solve and the module did achieve is a household count.** In the
  pack `payroll` (T6) the households stay within 10% of the tick-600 value to tick 2 200
  on seeds 101 to 104 (FR-REF-045), and on 25 of the 30 seeds 1 to 30 (failures are late step
  losses of about 19% between tick 1 600 and 2 200, with the cooks holding 57% to 74% of
  the money). With the module off the same pack collapses, so the payroll module, not the
  solver, carries the count. The mineral yield that distinguishes T6 from T5 is not
  distinguishable from T5 on seeds 1 to 10 (8 of 10 against 7 of 10).
- **Acceptance criterion 4 is not met** (non-cook households unfed 36% to 39% against a
  prior of 1%, prices about 40% of the reference, cooks four to five times their money
  share, X4 (b), (d) and (f) fail), and criterion 1 of `vision.md` 5.1 stays not met.
- **Why the decision is kept:** it makes the pack's census, prices and warm start follow
  the one configuration the world runs (the E2 mismatch cannot recur), and the module-off
  path is byte-equal. It is not kept because the static balance predicts the simulation.

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

- The receipts model (net sales less own consumption) was within 10% of the measured
  `PayrollPaid.Due` (97.5%, FR-VIA-066); the occupation review leaves the static census
  anyway, which this decision does not explain (N-356).
- The scope exclusions may be lifted later if a pack needs them.
