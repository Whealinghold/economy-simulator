# 0011. Payroll before hire: the first form of the employment module

- Status: Accepted (human review pending)
- Date: 2026-11-12
- Supersedes: none

## Context

`spec-employment.md` (EMP) specifies one form of the module: the **hire**. A
household of a `Hireable` role offers its labour, an employer posts a vacancy, a
first-fit round at P2.5a matches them, the wage is escrowed at formation and
released a tick at a time, and the hired household runs the employer's recipe
with the output going to the employer. It has 63 functional requirements, a
checkpoint section, three events, a removal step and a `region_bound` refusal,
and it needs a pack whose employers have hire lists, tools and inputs.

Every reference pack collapses (N-352, plan 2-25), and four engine levers did
not cure it. The measured mechanism is that demand is cash limited: one role
holds 49% to 90% of the money and the others earn a fraction of their upkeep.
The supervisor decision D-O5 allows one more lever, a wage, built only if the
pack-data redesign of criterion 1 of `vision.md` 5 exhausts its stop rule, and
only in a form that is default-off, deletable, and merged on its own proof, with
no claim on criterion 1.

A spike (plan 2-28, 2026-11-12; the code was reverted and is not kept) paid a
share of the receipts of the one sector that keeps them to the households that
worked, in proportion to their executions of the tick. On `one-demand` with the
base layer of the redesign trials, seeds 101 to 104: a share of 50% kept
households within 3% of the start at tick 1100 on 7 of 8 seeds (seeds 1 to 4
and 102 to 104 flat to tick 2200) and lost one seed; 25% and 37.5% collapsed two
seeds in four late; 62.5% and above collapsed every seed early. Cooks' share of
the money fell from 43% to 58% to 9% to 25%. Non-cook households unfed stayed at
33% to 43% (the bar of STY D01 is 1%), so a wage is necessary and not
sufficient.

## Decision

1. **The first form of `Modules.Employment` is the payroll form**, a subset of
   `spec-employment.md` written as its section 24. After clearing (P4.1b), each
   agent of an employer role pays `PayrollBP` basis points of that tick's
   receipts, cut at its balance, to the agents of `Hireable` roles that worked
   that tick, in proportion to their executions. No escrow, no round, no hire
   record, no state, no checkpoint section, no draw.
2. **It keeps the vocabulary and the seams of the hire form**: the module name,
   `Role.Hireable`, `Role.Employer`, the ledger reason `wage_pay`, the welfare
   terms `Y_t` and `V_t`, the pooled-wage exclusion of 8.4. The hire form
   replaces the body later without renaming; its sections (3 to 11, 14 to 17)
   are marked **later form** and stay as written.
3. **It is a departure from the written spec** and is written down first
   (architecture.md 2.10, 14.2): the spec had no profit-share wage.
4. **Default off and deletable.** `Modules.Employment` nil leaves `golden/off`,
   the pinned hash and the checkpoint format byte-equal; the module is removed
   by deleting `employment.go`, `internal/employment/` and the appended
   one-liners.
5. **No claim on criterion 1.** The module is merged, if at all, on its own
   proof (conservation, determinism, off-equality, its module-alone golden). It
   does not make any pack stable and it sets no band (N-027).

## Consequences

- The amendment is small (one section, about fourteen requirements) and the
  code is one root file and one `internal/` package of pure arithmetic.
- The pack must pin the share: the working band is narrow and one seed in eight
  collapsed in the spike.
- Cooks (or whoever is the payer) become less attractive in the spawner's
  occupation signal and their count moves; the census, the warm start and the
  `one-demand` golden are re-recorded once, with the pack overlay, not here.
- A later full hire form shares the module switch, the role fields, the reason
  and the welfare terms. It cannot reuse `PayrollBP` as `WorkerShareBP`: they
  mean different things, and the hire form appends its own fields.
- If the pack redesign succeeds without it, the module is not built (D-O5).

## Alternatives considered

- **The full hire form now.** Three to four times the cost, changes who
  produces (the output goes to the employer), needs hire lists, tools and a warm
  start with hires in flight, and was not tested by any spike.
- **A grant (MON) from the payer role to the others.** Not tied to work: the
  grant probes lost households. A wage is earned income and enters the earned
  ratio and the observed wage; a grant is `G_t`.
- **Do not build.** The recommended default until the stop rule of the pack
  redesign is exhausted; this ADR makes the form available, it does not turn it on.

## Open points

- The share's default (5 000 bp) is the spike's working value, a placeholder.
- Weighting by executions or by household weight was not compared (the plan's
  trials include it).
- Whether the payer set should be a role flag only or also a fraction of the
  payer's money.

## Related

- `spec-employment.md` section 24; `spec-world.md` 6 (P4.1b)
- plan 2-28 (stage 1 memo and plan, not merged); notes N-352, N-027
- ADR 0001, 0007, 0010
