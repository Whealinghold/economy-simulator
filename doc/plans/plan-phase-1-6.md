# Plan: Stage 1, row 6: the welfare record, the ratio and the observed wage

**Status: Proposed (2026-10-13); no phase started.** Sequences the sixth row of
Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`internal/welfare` and
`welfare.go` (the record, the ratio, the observed wage)", into five phases. It
implements [spec-welfare-and-wages.md](../specs/spec-welfare-and-wages.md) (WEL)
for the slice. It is built against `doc/rules/architecture.md` (the layout of
2.6, the `internal/` boundary of 2.9, the fixed-point and overflow rule of 3.5,
the determinism rules of section 7), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

Rule tags read for this plan: architecture=905c0e578c4e662908638dd8,
observability=748e621d4300d088eb469349, testing=b3559bc6d47470c221bc4642.
Notes tag (as read at the last revision, 2026-10-13; Phase 1 re-reads the file
and quotes the tag then): 774432c8fc96fc1e2ea7efaa.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR
or the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md)
(D3 and D5 above all). Where a decision is still owed, Phase 1 makes it in the
document that owns it, before the code that depends on it. Decision IDs D61 to
D66, D69 and D70 are this plan's (plan 1-4 has D40 to D49 and plan 1-5 D50 to
D60 and D67, so D68 is not issued and the next row starts at D71); earlier
plans' decisions are cited by file and ID.

No PRD exists for this row. The requirements source is the roadmap row and
WEL. The row is built after the ledger and goods rows
([plan-phase-1-1.md](plan-phase-1-1.md), [plan-phase-1-2.md](plan-phase-1-2.md))
and before the agent row, which holds the record in the agent.

Spec codes: WEL welfare and wages, AGT agents, WLD world, PRO production, CLR
clearing, SUB subsistence, EMP employment, MIG migration, SPN spawner, EST
estates, SKL skill barriers, OBS observability, CFG config, CHK checkpoint, VIA
viability, GDS goods, CAL calendar, STB stability, IND indicators. `FR-`/`NFR-`
IDs are the requirement IDs of those specs.

## What Stage 0 already built, and what is left

| Concern | Where it already is |
|---------|---------------------|
| `World.WelfareWindow`, `World.WindowCycles`, `World.RestBand`; `Role.Unskilled`, `TargetRatio`, `TargetSpread`, `DefaultRatio`, `InKind`, `ExpectedWorkTicks`, `MinWorkShare` and their defaults | `config.go`, `defaults.go` |
| The bounds of WEL 12.1, the `MaxWelfareWindow` bound, the `ExpectedWorkTicks` and `MinWorkShare` order rules (FR-WEL-014, 025, 028), the effective window of a role from its recipes (`effectiveWindow`, WEL 2.3.3), the overflow check of the window times the largest income, the informational `window_extended` | `bounds.go`, `validate.go` |
| `WithTargetSpread` (FR-WEL-026), applied to the effective configuration before the hash | `option.go` |
| The creation stream of D1: `siteConstruction`, tick 0, id the agent (WLD 7.1.4) | `random.go` |
| `internal/checked`: `Add`, `Sub`, `Mul`, `MulDiv`, `MulDivFloor`, `MulDivCeil`, `MulDivHalfEven` | `internal/checked` |
| The log keys `region`, `role`, `households` and the "work floor binding" row of OBS 7.3 | `internal/logkeys`, `spec-observability.md` |

What is left for this row is the behaviour WEL gives those fields: the per-agent
record and its running sums, the ratio and the derived reads, the arithmetic of
the target-income rule and the floor, the wage reads that other rows consume,
and the root glue that turns a tick's flows into a record.

## Scope

**In scope.**

- `internal/welfare` (stdlib and `internal/checked` only, primitives only):
  the ratio with padding and its earned form (2.1, 2.2, 2.1.7), the
  calibration formulas of 3.1 and the closed form of 5, the work share and the
  floor test (5, 5.3), the range of the drawn target (5.1), the wear cost of a
  tick (2.1.4), the record, the ring with running sums and its resize, the
  observed wage of one agent, the pooled wage of a set, the household-weighted
  mean ratio, and the observed premium (4.3, 11.2).
- `welfare.go` (root, unexported): the agent's history type, the scratch of a
  tick's flows and its conversion to a record at P6.3, the cost of a tick, the
  effective window of a role, the wage a region's cost anchor reads (pooled, else
  the warm-start wage), the creation of a history and of a target, the run
  counter of the floor `Warn`, and the per-(region, role) summary the snapshot
  rows are built from.
- The glossary terms these files use, in `doc.go`, before they use them
  (architecture.md 3.2): added at the start of Phase 2, the first phase whose
  code and doc comments use them.
- The spec, note and roadmap edits these files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| Holding a history in the agent, the agent's removal, split and merge steps, the view fields (ratio, target, window, working ticks), the default strategy's use of the work share and the survival rule | Roadmap Stage 1, `agent.go` and `strategy.go` row. This plan gives them the type and the functions |
| Accumulating the trade receipts and the spending by purpose into the scratch (FR-AGT-057): clearing is built before welfare and cannot call it, and `Order` carries no `Purpose` ([plan-phase-1-3.md](plan-phase-1-3.md) D32), so the world row calls `addReceipt` and `addInputSpend` from the settled trades and the purpose of each intent (plan 1-3 Phase 5: `clearMarket` appends the settled trades to a caller-owned slice and `book.filled()` gives the units per order; O5 closed). The market price a cost is valued at | Roadmap Stage 1, `world.go` row and `agent.go` row (Cross-plan contract) |
| Recording the wear units and setting the working scratch at P2.1 and P2.7 (the wear is valued at P6.3, D66): the production steps return the wear facts and the working executions and call no other row, so the world row calls `addWear(commodity, executions, wearPerUse)` and `markWorking(k, ownUse)` with them ([plan-phase-1-5.md](plan-phase-1-5.md) D51). The own-use units and the cost anchor that reads the wage (at P1.2, WLD 6.3) | Roadmap Stage 1, `world.go` row (the calls); `production.go` row (the facts and the anchor table) |
| The calls at P1.6 (relief reads the ratio), P2.5 (the floor `Warn`), P6.3 (the push), P7.1 (resize at a role change), the snapshot rows `RoleRow` and the per-agent query, the `Warn` emission | Roadmap Stage 1, `world.go` row, `estates.go` row and `spawner.go` row. This plan gives them the functions and the run state |
| The role economics check as a report (`WageBelowTarget`, WEL 7, FR-WEL-031, 038) | Roadmap Stage 1, `viability.go` row (VIA 10.3). `internal/welfare` gives it the formulas of 3.1 |
| The checkpoint encoding of the ring (WEL 13, FR-WEL-034) | Roadmap Stage 2, `checkpoint.go` row. This plan gives it an ordered read and a rebuild from records |
| In-kind pay (WEL 6, FR-WEL-029): the spec itself routes `Role.InKind` as "no mover and no position" (`spec-world.md` open question 14) | Not built; `InKind` stays a validated field. A mover is a spec decision first (note N-147) |
| The target of an agent that changes role: `clamp(new_target + (old_target - old_role_target), new_target - spread, new_target + spread)`, with no draw (SPN 4.2, FR-SPN-006). It is a formula over the role rows and the stored target, called at P7.1 | Roadmap Stage 1, `spawner.go` row. This plan gives it the range of 5.1 (`lo`, count) and keeps nothing else; a function that computes it is the spawner's, not a second welfare formula |
| A trainee's board (FR-WEL-030), the migration trigger read (`spec-migration.md`), the spawner's occupation signal (`spec-spawner.md` 6.1), the demand-side readers | `skills.go` (Stage 3), `migration.go` (Stage 2), `spawner.go` (Stage 1): each reads the functions this row gives |
| The terms of `G_t` and the record's other flows that a module owns (grants, payouts, interest, insurance, rent, tax, wages, escrow release, in-kind receipts, own-use receipts) | Each module's PR, appended (architecture.md 2.10.5). The record has the fields now (D61); the module fills them |
| Amending the last record for a flow that moves after P6.3 (WEL 2.1.9, INS 11.1.3, CRD) | The first module that needs it (CRD or INS), appended |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- `internal/welfare` is standard library and `internal/checked` only, never
  imports the root, and sees integers and slices. The root converts inline at
  the call (architecture.md 2.9; WEL 9).
- All money in the record is signed `int64` minor units, ratios are basis
  points, every product that can overflow goes through `internal/checked`
  (architecture.md 3.5; FR-WEL-010).
- The ring is per agent, hosted in the agent (`spec-agents.md` 3.1, WEL 13),
  holds the last `T` records with running sums, and the cost of a read is
  constant in `T` (NFR-WEL-001). Padding is applied at read, never stored
  (WEL 2.2).
- The window is derived, `max(WelfareWindow, WindowCycles * L)`, never
  configured per role (WEL 2.3.3). In the slice `L` is the longest `Duration`
  on the role's recipe list; the own-use list and the merchant trip join with
  their modules.
- The target of an agent is drawn once at its creation, whatever the spread
  (WEL 5.1): the first of the two creation words of WLD 5.7.1 (FR-WLD-066),
  taken from the agent's own creation stream, which is D1
  `(siteConstruction, 0, agent)` at construction, D10
  `(siteCreation, tick, new agent)` at P7.1(d) for a replacement or a birth and
  D12 at a call between ticks for an external agent (WLD 5.7.2, 7.3). The range
  `lo + draw(hi - lo + 1)` is WLD 5.7.1's. This row takes the word from the
  caller and never names a site.
- The pooled wage is unknown when no household worked outside own-use in the
  windows, or when it is not positive, and the starting wage of the warm start
  is used then; nothing is latched (WEL 4.3, `spec-checkpoint.md` 4.5 F5).
- A module that is off contributes 0 to every term, and an appended field is
  zero while off (D5 of plan 0; architecture.md 2.10.5).

Decided by this plan, to be written into the owning spec in Phase 1 before
code (each is flagged for the owner's override in the review):

- **D61.** The record has every field of WEL 13 from the first build:
  `income`, `cost`, `relief` (`F`), `unearned` (`G`), `ownReceipt` (`K^own`),
  `working` flag, `workHH`, `ownHH`. In the slice only `income`, `cost` and the
  working fields are non-zero (relief needs the `Estates` module part, Stage 2;
  `G` and `K^own` need modules). The shape is fixed now because it is the
  checkpoint format (Stage 2) and a field added later would change it, the
  reason plan 1-1 D8 built the claims list ahead of its module.
- **D62.** Rounding and absence. The earned ratio truncates as the ratio does
  (WEL 2.1.7, FR-WEL-039: "the same padding and truncation"; 2.3.2: rounding is
  truncation) and the mean ratio is rounded toward zero (FR-WEL-036). WEL gives
  no rounding for the observed wage, the pooled wage and the premium: they
  truncate toward zero, the default of architecture.md 3.5, and Phase 1 states
  it in WEL 4.3 so that it is a rule and not a default found in code. A wage
  with no working household-tick outside own-use is absent, not 0. There are
  two set-wage reads and they differ on purpose: the **observed wage** of an
  agent or of a role in a region is present whenever the denominator is above 0
  and may be zero or negative (WEL 4.3, FR-WEL-037: the spawner's `HasWage`
  says only that someone worked); the **pooled regional wage** is known only
  when the denominator is above 0 and the quotient is above 0, and falls back
  to the starting wage. The premium is absent when the unskilled wage is not
  positive (Phase 1, deliverable 1).
- **D63.** An overflow is handled where it occurs and never panics
  (`spec-world.md` 3.3.3: "an overflow of checked arithmetic is an operation
  that cannot be applied and is handled where it occurs"; architecture.md 8.1
  and 13 forbid a panic a valid configuration and a consumer's strategy can
  reach). Validation bounds only the sales of the role's recipes
  (`validate.go`, the window times the largest income), so the receipts of a
  role with no recipes, relief and the unearned flows are not bounded by it
  and money up to `MaxMoney` times a window up to `MaxWelfareWindow` exceeds
  `int64`. Phase 1 writes the handling into WEL 2.3.2 and 12.2: `Push` clamps
  each field of the record it stores to plus or minus
  `MaxInt64 / MaxWelfareWindow`, so that no running sum over a window can
  overflow (a proof, not a hope), and returns that it clamped. The bound is
  a parameter of `Push` and `Resize`: `MaxWelfareWindow` is a root bound that
  `internal/welfare` cannot import (architecture.md 2.9, 9.6), and a ring is
  never longer than it, so a resize up stays inside the proof. **One rule for everything formed from the bounded sums.** The proof covers one
  ring's running sums, each at most `MaxInt64` in size, and nothing else: what
  is formed from several of them, or from several agents, can leave `int64` in
  a valid configuration (`MaxMoney` is 10^18 and a no-recipe role's receipts
  are not bounded by validation, so two agents near the clamp are enough).
  Every value of the tick that is formed from them is a **named value of the
  spec's formula** (a sum, a padded sum, the per-agent numerator
  `income - relief - unearned - ownReceipt`, a set's numerator and
  denominator, a weighted sum, a quotient), and each is computed with three
  helpers, the only ones in the package, built in Phase 2 on
  `internal/checked` and **unexported**: `satAdd(a, b)` (a sum; its difference form `satSub(a, b)` is over `checked.Sub`
  with the sign of the exact result, never a sum with the negated second term,
  because `-MinInt64` is not an `int64`), `satMulDiv(a, b, c)`, the 128-bit quotient
  `a * b / c` of `checked.MulDiv` whose **final value alone** saturates
  (`satMul(a, b)` is `satMulDiv(a, b, 1)`), and `satMulDivCeil(a, b, c)`, the
  same over `checked.MulDivCeil`, for the one ceiling of the wear cost (D66).
  **How the root reaches them.** It does not: the root cannot call an unexported
  symbol of another package and architecture.md 2.9 puts the arithmetic in
  `internal/`, so everything the tick forms from the scratch is a function of
  `internal/welfare` over primitive slices that `welfare.go` calls after it
  converts inline: `TickCost` (the upkeep rows times their prices times the
  weight), `WearCost` (D66) and `TickIncome` (the credits of 2.1 less the
  debits, in the slice order). The root keeps no helper, no copy and no
  `checked` call for this (Phase 2, Phase 4). A value that leaves `int64` **saturates to the
  nearest extreme**: `MaxInt64` where the exact value is above it, `MinInt64`
  where below (for a sum, the sign of the operands that overflowed; for a
  quotient, the sign of `a * b * c`, which `checked.MulDiv` leaves to the
  helper because it reports only `ok` false). A ratio or a premium is **one**
  `satMulDiv` (`num * 10 000 / den`), never a saturated product followed by a
  division: the product by 10 000 is inside the 128 bits, so the quotient is
  exact whenever it fits (a role wage of `MaxInt64 / 2` over an unskilled wage
  of 20 000 is `MaxInt64 / 2 * 10 000 / 20 000`, in range, not `MaxInt64 /
  20 000`). The terms of a sum are taken in the order of the caller's slice
  (ascending agent slot), because a saturating sum is not associative and the
  order is then part of the rule. Nothing wraps, nothing panics and nothing
  is turned into "absent" for this reason: the only absent results are those
  of D62 (a zero or non-positive denominator), applied to the saturated
  values unchanged. The same rule covers what the root forms **in the tick**
  (through `TickIncome`, `TickCost` and `WearCost`, above): `record` (the
  income and the cost of a tick), the cost of a tick (weight times quantity
  times price, summed over the upkeep rows) and the wear valuation
  `n * ceil(...)` saturate and never report `ok`; the world cannot
  refuse at P6.3 (`spec-world.md` 3.3.3), so a returned `ok` would have no
  consumer. A saturated field is above the clamp, so `Push` clamps it and
  reports it, and the `Warn` below is the one report of an overflow in the
  tick. `ok` is kept only by the functions that run outside the tick (the
  calibration formulas of 3.1 and 5, which the viability row and the tests
  call and which a validator would refuse with `ErrOverflow`). Both the clamp
  and the saturation are deterministic and change no result below their
  bounds. It is not an error of `Step`. **The clamp is not silent**
  (observability-and-logging.md 3.2, 9.1; FR-STB-013, N-045): a clamped field
  is stored at the bound, so a household whose ring holds a field at the bound
  is the evidence and is a function of saved state (N-125). Phase 1 adds the
  `Warn` "welfare record clamped" to the table of OBS 7.3 (pattern A, keys
  `region`, `role` and `households`, cells bounded by regions x roles, so
  rule 3.2.2 holds) with the counterpart `RoleRow.ClampedHouseholds` (a count
  of households, appended to the row of OBS 5.3.4: at its off value 0 it is
  omitted from the golden rendering, FR-OBS-091), and WEL 11.3 gains
  FR-WEL-042 for both. The world row emits it from the run it keeps with the
  floor run (A7). The owner may choose another handling before Phase 3 (O4);
  a handling with no counterpart is not one of them.
- **D64.** A role change resizes the ring and keeps the history (`spec-agents.md`
  3.8 and `spec-spawner.md` 4.2: "welfare history kept"; AGT 3.7 is the weight
  change table and stays unchanged): the newest `min(held, T')` records are
  kept, `T'` the new role's effective window, and the ticks missing are padded
  at read with the new role's `DefaultRatio`. WEL 2.3.3, WEL 13 ("the number of
  records held") and AGT 3.8 are silent on the length; Phase 1 says so.
- **D65.** The floor test is the literal text of FR-WEL-027: the working ticks
  in the agent's window are fewer than `MinWorkShare * T / 10 000` in integer
  division, `T` the effective window and not the records held, so a padded
  (missing) tick is not a working tick and a new agent is held to work until its
  window holds the share. `MinWorkShare` 0 turns it off.
- **D66.** Wear is valued at P6.3, not at P2.7. WEL 2.1.4 values wear "at the
  region's market price of the tool in that tick", and the tick's market clears
  at P4.1, after wear is started at P2.7. The scratch therefore holds the
  facts plan 1-5 D51 hands over as rows (commodity, `n` executions started,
  `WearPerUse`), and `record` values them at P6.3 with the same prices as
  `cost_t` (the published price of the tick, the effective reference price if
  the market has never traded). No money value is passed at P2.7. **The
  arithmetic** (PRO 4.4.2: each execution wears its own distinct item, and an
  execution is one household's, so `k` executions are `k` households and the
  weight enters only through `k`, which is at most the weight, FR-SCL-007):
  the wear cost of a row is `n * ceil(WearPerUse * price / Durability)`, one
  ceiling per execution and then the product, never `ceil(n * WearPerUse *
  price / Durability)`, which differs (WearPerUse 3, price 10, Durability 4,
  `n` 2: 2 * 8 = 16, against ceil(60 / 4) = 15). It is NOT multiplied by the
  weight again: "times the weight of the executor" in WEL 2.1.4 and FR-WEL-003
  is read as "per household execution, times the executions", and a second
  factor would count a cohort of weight 3 running 2 executions as 6. Rows
  with the same commodity and `WearPerUse` merge exactly (their `n` add);
  rows with another `WearPerUse` stay apart, because the ceilings differ.
  Phase 1 states it in WEL 2.1.4 and FR-WEL-003 (the owner may rule that the
  weight is a factor and `n` is per household; then `n` is 1 and the row
  changes, nothing else, O4).
- **D69.** What "held by the floor" is, and its denominator (AGT 7.1.3, WEL
  5.3, 11.3). An agent is held when it is free (no job in progress when `Work`
  would be called), the floor condition of D65 holds and the share the ratio
  alone gives is below 10 000: an absent ratio, or a ratio below `T_lo` (share
  already 10 000), is not held. The count is in households (the weights of
  the held agents). The `Warn` count is taken at P2.5 when `Work` is called;
  the snapshot count is the same predicate at the boundary (A7). The
  denominator of "more than half of the role's households" is every household
  of the role in the region, busy or free, so a busy agent never raises the
  share; strictly more than half warns, exactly half does not
  (`2 * held > households`). A role with no households has no run. Phase 1
  states the denominator and the tick in WEL 11.3 and FR-WEL-033.
- **D70.** The working household-ticks of a role with no recipes. An agent has at
  most one active job and does not start one in a tick in which it continued or
  completed one (FR-PRO-014; `spec-world.md` P2.5 calls `Work` only for a free
  agent, "none continued or completed in P2.1"; one job per agent, N-118), so
  in a tick the scratch is set **at most once** for an agent: by P2.1 for the
  job continued or completed, or by P2.7 for the job started, never by both,
  and never a role job and an own-use job together. `markWorking` sets once per
  tick and slot, and a second call for one slot in a tick is a violated
  programmer invariant (a panic whose message says so, architecture.md 8.1),
  not a rule to resolve by maximum or sum. For a role **with recipes** the
  working household-ticks `workHH` are the `k` of that one job and the
  own-use household-ticks `ownHH` are that `k` when the job is an own-use job
  and 0 when it is a role job, so `workHH - ownHH >= 0` follows from the one
  job and `k` at most the weight needs no cap. The part of 4.3 that is not
  already a rule is the **role with no recipes**, which has no job of its own
  to give a `k`: it works by trading, and an own-use job is a second reason to
  work. Its `workHH` is the weight when it traded or travelled, and
  `workHH = ownHH = ownUseK` when it ran an own-use job and did not trade; one
  that traded and also ran an own-use job has `workHH` = the weight and
  `ownHH = ownUseK`, so the wage denominator is `weight - ownUseK` (never
  negative, `ownUseK` at most the weight). Phase 1 states the no-recipe clause
  in WEL 4.3 and does not state a rule for two jobs in a tick, which FR-PRO-014
  excludes. The alternative of reading `workHH` of a trading no-recipe agent
  with an own-use job as `weight - ownUseK` (the trade of the households that
  also ran the own-use job not counting) is the owner's (O4).

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The signatures of `internal/welfare` are the implementer's, within WEL
  and architecture.md 2.9. Names in this plan and in the Cross-plan contract are
  indicative of the capability, not an interface promised to other rows; a row
  that needs a name checks the code of this row once it is merged.
- **A2.** `internal/welfare` holds one value type per fact, no map, closure or
  package variable (architecture.md 7.1, 7.5), and takes the effective window
  `T`, the default ratio and the current upkeep as parameters. It knows no role,
  region, commodity or agent identifier (architecture.md 9.1, 2.9).
- **A3.** The agent type of the agent row holds the history as
  `welfare.Ring` (an internal type in a field of the root's agent) and not a
  root wrapper, because a wrapper whose job is converting is forbidden
  (architecture.md 2.9). `welfare.go` holds the functions that need the
  configuration.
- **A4.** The scratch of a tick's flows is one struct per agent slot, indexed by
  the agent's slot like the module slices of architecture.md 2.10.1, owned by
  `welfare.go`, cleared at P6.3 after the push. The other rows add into it
  through the unexported methods this row declares; none writes a field of it
  directly.
- **A5.** The cost of a tick and the wear of a tick are valued at P6.3 at the
  published market price of the agent's region, and at the effective reference
  price where the market has never traded (WEL 2.1.4 for the wear cost, D66; 2.1
  says "the prices of the region the agent was in" for `cost_t`). The price
  source is the clearing row's; this row takes a price per commodity as a slice
  parameter. The region is the account's region at P6.3, which is the region of
  the tick: an arrival of a journey or a shipment (P1.x) and a relocation of
  phase 7 both fall outside P6.3 and never straddle it (WLD 6), so no tick is
  split between two regions' prices.
- **A6.** The pooled wage of a region is computed by a scan of the region's
  agents once per tick, at P1.2 with the cost anchors (WLD 6.3: P1.1 is the
  moment), from the per-agent sums
  as the previous tick's P6.3 left them (so the cost anchor "as it stood at the
  end of the previous tick", WEL 4.3). A scan is O(agents) per tick. Whether an
  incremental per-region sum is worth it is measured by the world row against
  the cost model of `spec-scale.md` 6 (open item O2); the pure function takes
  per-agent numerator and denominator slices, so either source works.
- **A7.** `HeldByFloor` is evaluated by the same pure predicate (D69) as the
  `Warn` count but at the boundary, over the agents with no job in progress on
  the record after P6.3, as OBS 5.3.4 says (they may differ, WEL 5.3); the root
  gives the snapshot a function, not a stored count.
- **A8.** As plan 1-2 A8: the root file's hand-computed cases run from an
  internal test (package `economysim`) with small configuration literals, as
  plans 0 and 1-1 do, until `NewWorld` exists. Whatever needs the packs `one`
  and `second` (the effective window against the validator, the wear agreement
  with `internal/production`) runs from one external test (package
  `economysim_test`) that loads them through `config/` and reaches the
  unexported functions through a test-only `export_test.go`: an internal test
  cannot import `config/`, which imports the root. No integration test is
  possible in this row.
- **A9.** One PR per phase, in phase order, on a branch from `stage-1-1`.
  Phases 2 and 3 may be written while Phase 1 is in review, but they implement
  spec text: Phase 2 D63 (the saturating helpers and the tick functions), D65 (the floor test) and D66 (the
  wear cost), Phase 3 D62 (the wage forms), D63 (the clamp), D64 (the resize) and
  D69 (the held predicate), so neither is merged before Phase 1 is; D63 (the
  saturating helpers) is Phase 2's as well as Phase 3's, because the ratio,
  the earned ratio, the work share, the premium and the wear cost of Phase 2
  use it and Phase 3 only reuses it; D70 (the no-recipe working household-ticks) is
  Phase 4's; Phase 4 does not start before Phase 1 is merged and needs
  Phases 2 and 3, and plan 1-2 Phase 5 (the role table and the effective
  upkeep; or test role rows, "Dependency failure" below). The agreement test
  with `internal/production`'s wear term needs plan 1-5 Phase 4. The row shares `doc.go`, `doc/notes.md` and the roadmap with
  the other rows: each PR appends its lines and re-reads the shared file just
  before editing it.
- **A10.** The weight multiplication of the cost of a tick has one owner here.
  FR-WEL-004 forms `cost_t` as the effective upkeep times the weight, and N-236
  (1) gives the agent row the multiplication of each upkeep quantity by the
  weight of FR-AGT-011. This row's cost function takes the per-household rows
  of the role's `upkeepIn(region)` and the weight and is the only place where
  the welfare cost multiplies a quantity by the weight; the agent row's product
  (what the agent must buy and eat) is its own and is not handed over, and this
  row does not read it, so no row multiplies twice. The agent row's plan
  confirms it (N-236).

## Phase 1: settle the owed spec lines and the notes

**Why first:** D62 to D66, D69 and D70 are rules the arithmetic implements; a code
change that invents them would put a result-affecting choice in a place the
configuration hash and the checkpoint cannot see (architecture.md 9.5). This
phase changes specs and notes, so it starts by reading `doc/notes.md` in full
and quoting its tag, handles every open note tagged WEL and ALL and, for the specs it edits, AGT, SPN, WLD, OBS and CHK (below), and
gives `doc/notes.md` a fresh tag (`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **WEL 4.3 and FR-WEL-018, 019, 036** (D62). State that the observed wage of
   an agent, the pooled regional wage and the premium truncate toward zero (the
   earned ratio and the mean already have their rule in 2.1.7 and FR-WEL-036;
   cite it, do not restate it); that a wage with no working household-tick
   outside own-use is absent; the order of the arithmetic (sum first, one
   truncating division); that the observed wage of a role or agent is present
   whenever that denominator is above 0, whatever the sign of the wage, while
   the pooled regional wage is known only when the quotient is also above 0;
   that **the premium is absent when the unskilled wage is not positive** (zero
   or negative, which would divide by zero or flip its sign); and that
   unskilled agents with no working household-tick outside own-use have no
   wage and are left out of the mean. **The unskilled wage** is the observed
   wage of the set of the region's agents of `Unskilled` roles: the sum of
   their numerators over the sum of their working household-ticks, one
   truncating division, so each household-tick weighs the same and an agent's
   wage is not truncated before it is weighted (the mean of FR-WEL-019,
   "weighted by households", read as the pooled form; it can differ from the
   mean of truncated agent wages by up to one unit, which moves the premium).
   The same set function serves, so there is one implementation. **The
   region's wage** has no "unknown" in a type: the function returns
   `(wage, known)` and the world row writes 0 for an unknown wage into the
   cost-anchor table, which is the form PRO 9.1.1 and FR-PRO-049 read.
2. **WEL 2.3.3, WEL 13, AGT 3.8 and SPN 4.2** (D64). Say what a role change
   does to the length of the record: the newest `min(held, T')` records are
   kept, the rest of the window is padded as in 2.2 with the new role's
   `DefaultRatio`. AGT 3.7 is the weight change table and is not edited (its
   "welfare history unchanged" row is about a weight change, which does not
   resize the ring: the window is a function of the role, not of the weight).
   The AGT 3.8 and SPN 4.2 edits are one clause each in the existing "welfare
   history kept" text (mechanical, cross-plan with the spawner row).
2b. **WEL 4.3 and FR-WEL-017, 018** (D70). Say what the working household-ticks
   of a role with no recipes are: the weight when it traded or travelled, and
   the own-use `k` when it ran an own-use job and did not trade; with a trade
   and an own-use job `workHH` is the weight and the own-use household-ticks
   the own-use `k`, so `workHH - ownHH` is not negative. For a role with
   recipes `workHH` is the `k` of its one job (FR-PRO-014) and `ownHH` is that
   `k` only when the job is an own-use job. Say nothing about two jobs in a
   tick: FR-PRO-014 and P2.5 exclude them, and the scratch is set once per
   agent per tick. Also say that the working tick of a role with recipes on a
   commercial trip has no household-ticks rule in WEL 4.3 and is left to
   `merchants.go` (Stage 2): the slice never sets `inTransit`.
3. **WEL 5.3 and FR-WEL-027** (D65). Say that the count is over the effective
   window, that a missing tick is not a working tick, and that the comparison
   is `working < MinWorkShare * T / 10 000` in integer division (truncation
   toward zero, architecture.md 3.5).
4. **WEL 5.1, FR-WEL-024 and WLD 5.7.1** (N-214, WEL part). Word 5.1 and
   FR-WEL-024 as "the first of the agent's two creation words of WLD 5.7.1
   (FR-WLD-066), from its creation stream: D1 at construction, D10 at P7.1(d),
   D12 at a call between ticks (WLD 7.3)". The text "one word of the world's
   draw" is positional language stale under ADR 0008 (N-214). The range
   `lo + draw(hi - lo + 1)` is WLD 5.7.1's and stays there; WEL 5.1 keeps its
   statement of the range in words and cites it. No site is written into WEL.
5. **WEL 9** (the where-this-lives table). Name `welfare.go` as the root home
   of the record's conversion, the scratch and the summaries, and
   `internal/welfare` as the home of the pure arithmetic and the ring, and
   correct the line that says the ring is "computed by the root from ledger
   totals" so it names the scratch.
6. **WEL 11.3, 5.3 and FR-WEL-033** (D69). Say what "held" is (free agent, floor
   condition holds, the ratio alone gives a share below 10 000), that the count
   is in households and is taken at P2.5 when `Work` is called, that the
   denominator is every household of the role in the region, that strictly
   more than half is required (exactly half does not warn), and that the run
   is kept per (region, role) as diagnostic state (13). One reading for the
   world row.
7. **WEL 2.1.4 and FR-WEL-003** (D66). Say that the wear is recorded when the
   executions start and valued at P6.3 at the prices of the tick, as `cost_t`
   is, not at the price of P2.7, and give the arithmetic once: the wear cost
   is the sum over the executions started of `ceil(WearPerUse * price /
   Durability)`, one ceiling per execution, so a row of `n` executions of a
   tool costs `n * ceil(...)`; the weight does not enter a second time (an
   execution is one household's and `k` is at most the weight); a hired
   execution is the principal's and the `n` are the principal's. Replace
   "times the weight of the executor" in FR-WEL-003 by that statement, and say
   that it equals the `wear_cost` of `spec-production.md` 9.1 per execution.
8. **WEL 2.3.2, 11.3, 12.2 and 13, OBS 5.3.4 and 7.3, `spec-checkpoint.md` 4.2
   tag 7 and FR-CHK-043, FR-WEL-042** (D63). State the handling of an overflow
   that a valid configuration can reach: the clamp of a record's fields, the
   proof that one ring's sums then fit, and **the one rule for everything
   formed from them** (D63): every named value of the ratio read (the padded sums
   and the quotient), the per-agent numerator, the set sums of numerators and
   denominators, the mean ratio, the premium, and what the root forms in the
   tick (the cost of a tick, the wear valuation, the income) is computed with
   the saturating helpers of Phase 2 (a sum, one 128-bit quotient
   whose final value alone saturates, and its ceiling form), reached from the
   root only through the exported tick functions `TickCost`, `WearCost` and
   `TickIncome`, to the nearest extreme, in the slice's
   order, never wrapping, never absent for that reason, and never reported by
   `ok` where the world cannot refuse (the clamp of `Push` and the `Warn` are
   the report); and that `ErrOverflow` of validation is
   unchanged. State the counter that makes the clamp visible: add the `Warn`
   row 26 "welfare record clamped" to the table of OBS 7.3 (pattern A; keys
   `region`, `role`, `households`; cells regions x roles; counterpart
   `RoleRow.ClampedHouseholds`; owner WEL 11.3), the field to the row of OBS
   5.3.4 (a count, appended last, omitted at 0), "20 to 25" to "20 to 26" in
   7.3.1 with the prose count of its last paragraph (lines 20 to 26 about 350
   more, 100 cells of the new row over the 100 (region, role) cells, so about
   2 050 in all, and the same figure in the next sentence), and the requirement
   FR-WEL-042 (next free ID: re-read the spec). **The checkpoint inventory.**
   The clamp run is a second diagnostic run per (region, role) and WEL 13, CHK
   and FR-CHK-043 require a spec that adds saved state to add it to the
   inventory (N-128), or a restored world would log the clamp `Warn` at other
   ticks than the original. Edit `spec-checkpoint.md` 4.2 tag 7 from "then the
   diagnostic run per (region, role) of the `Warn` of WEL 11.3, sparse" to
   "then the diagnostic runs per (region, role) of the two `Warn`s of WEL 11.3
   (the work floor run and the clamp run), sparse", and the matching bullet of
   WEL 13 (two runs per (region, role), in region then role order, one for each
   `Warn`). Re-read `spec-checkpoint.md` just before the edit: the production
   row (plan 1-5, tag 6) and the ledger row edit the same table. The Stage 2
   checkpoint row then builds the section from the inventory.
9. **WEL 2.2 and 2.1** (a clarification). The upkeep `U` of the padding at a
   read is the effective upkeep valued at the prices published at the position
   of the read (P1.6 relief, P2.5 view, P7.x), so the agent, estate and spawner
   rows agree. State beside FR-WEL-007 that the padded ratio equals
   `DefaultRatio` exactly when `n * U * d_bp` is divisible by 10 000 and is
   truncated otherwise (`pad_income` truncates: `n` * `U` 1 at 15 000 pads 1
   and reads 10 000).
10. **The roadmap and `doc/documentation_index.md`**: check, no edit. The roadmap
    (Stage 1, lines 111-114) already links this plan and says the role
    economics report is the viability row's and in-kind pay is not built; the
    index already has the row. Only the decision range of the index row
    changes (D61 to D66, D69 and D70) if it has not by then.
11. **Notes.** The disposition of every open note tagged WEL or ALL, and of the notes tagged AGT, SPN, WLD, OBS and
    CHK that the edits above touch or neighbour, is in the
    table below and is written into `doc/notes.md` in this change. Re-read the
    index at Phase 1 time for any note tagged WEL, ALL, AGT, SPN, WLD, OBS or CHK
    raised since. A
    further gap found takes **N-234** (reserved for this plan by the cross-plan
    review). Phase 1 also edits AGT 3.8, SPN 4.2, WLD 5.7.1 (a check), OBS 5.3.4
    and 7.3 and `spec-checkpoint.md` 4.2, and `AGENTS.md` asks for the open notes
    tagged with the code of each spec it changes. The table below therefore also
    holds the dispositions of the open notes tagged AGT, SPN, WLD, OBS and CHK as
    the index stood at the last revision (2026-10-13): one row for each note
    whose clause or neighbour this row edits, and one closing row for the rest,
    which are left open because their clauses are not edited. Phase 1 re-reads
    the index for notes raised since and adds a row for each.

**Notes disposition** (apply, reject or leave open, with what was learned):

| Note | Disposition |
|------|-------------|
| N-016 Keep the parameter registry true (ALL, CFG, WEL) | **Leave open.** This row adds no field and no default: every welfare field already has its owner row (Stage 0). The registry test stays Stage 0's staged test; this row's change deletes no `notBuilt` entry. Add: `WelfareWindow` has its row since 2026-10-10 |
| N-017 Words and names (ALL) | **Apply.** The terms this row's code uses go to the `doc.go` glossary before use: welfare ratio, window (qualified: the welfare window), working tick, working household-tick, own-use household-tick, observed wage, pooled regional wage, observed premium, earned ratio, work share, work floor, target ratio, record (of a tick; qualified from CLB's record), ring. Qualified spellings stay (window, record, floor). The note stays open for later stages |
| N-027 No default macro constant is stable (ALL, REF, STY) | **Leave open.** Every default of WEL stays a labelled placeholder; no test of this row asserts a default as stable and the plan states none |
| N-028 Can the target-income rule make a shortage worse? (WEL, STB) | **Leave open.** The rule is built here as arithmetic and tested at its closed form (FR-WEL-022, RestBand 0); the run that decides the floor and the pinned outcome is scenario S04 on the library, which needs the world and the agent row. Recorded: this row ships the floor with its counter and the `RestBand` form so S04 can run both ways |
| N-099 A trip is a gap in income (WEL, STB) | **Leave open.** The merchant block does not exist in the slice, so `L` is the recipes' only; the trip term joins with `merchants` (Stage 2). The window function takes `L` as a parameter so the merchant file appends its term |
| N-118 One job per agent limits the subsistence floor (tag PRO, WEL, SPN) | **Leave open.** Not this row's: SUB's mode and the own-use list do not exist in the slice. Learned: the window function must count the own-use list when SUB lands (WEL 2.3.3), a one-line append |
| N-119 The receipts of a household that feeds itself (WEL, EST) | **Leave open.** `K^own_t` is a field of the record (D61) and stays 0 in the slice; the provenance question is SUB's and EST's |
| N-130 What IND cannot measure exactly until another spec adds a field (WEL among OBS, IND, LED...) | **Leave open.** `IncomeLast`, `ReliefLast` and `CostLast` are not adopted in version 1 (`spec-observability.md` 5.3.4); this row's summary function returns the three sums as unexported values so the ask costs one snapshot field if adopted. Item 2 (wear units) is the ledger's |
| N-147 Pay in kind has no mechanism (WEL, EMP) | **Leave open.** No mover and no position; `InKind` is validated and unused. Not built (Scope table) |
| N-148 The employer role's observed wage includes its profit on hands (SPN, WEL) | **Leave open** for EMP and SPN (Stage 3). Learned: the per-role wage function takes a "pooled over roles without an employer block" selector as a parameter slice (WEL 4.3), so the module appends the rule without changing the signature |
| N-171 A build is booked as a cost with no return (WEL, MIG, CAP) | **Leave open** for CAP (Stage 3). The record has no capital-formation field, by D61 as WEL 13 has none; WEL decides whether capital formed is ever credited |
| N-183 A tax is netted from income; a MON levy is not (WEL, MON, FIS) | **Leave open** for FIS (Stage 3). WEL 2.1.9 already assigns the letters; `unearned` is the one signed field, so no field is added when FIS lands |
| N-214 Keyed random streams: the edits owed (WLD, ..., WEL, CODE) | **Apply for WEL** (deliverable 4): D1 is keyed by the agent. The other owners' edits stay open |
| N-103 Running several spec writers in parallel (ALL) | **Leave open.** Process note; this plan was written in parallel with siblings and reads them only for the contract |
| N-126 A new event or `Warn` adds a row (ALL) | **Apply.** This row adds no event and one `Warn`: "welfare record clamped" (D63) takes row 26 of OBS 7.3 with its counterpart `RoleRow.ClampedHouseholds` in OBS 5.3.4, as FR-OBS-003 and FR-OBS-051 require; no event, so the rules' catalogue (4.3) is unchanged. The "work floor binding" row exists; the world row emits both. A test reads both tables, so the row is in the same change as the code (Phase 4) |
| N-128 A hosted per-agent field is saved once, in the agent record (ALL, CHK) | **Leave open.** The ring is the agent's hosted field (WEL 13); this row's `Records` read and `FromRecords` rebuild give CHK one place to write it |
| N-163 The word "treasury" (MON, FIS, ALL) | **Reject for this row.** No use of the word; left open for its owners |
| N-235 One combined architecture.md edit for the fan-in and admission touch points (ALL) | **Leave open.** Not this row's: the `architecture.md` edit waits for the first Stage 2 module PR. Learned: welfare adds no fan-in and no admission touch point (`welfare.go` is a core concept file, not a module file; its scratch is called from other rows' positions) |
| N-217 D5 binds every module's change (ALL, CODE) | **Apply.** Nothing in this row appends to the configuration, an event or a snapshot row, so the pinned hash cannot move, and `testdata/golden/off/`, which does not exist yet, will not; the exit gate checks the pinned hash |
| N-228 A module's items in the tradable set have no touch point (AGT, DEM, SUB, CODE) | **Leave open.** AGT 3.8 is the only AGT clause this row edits and it is about the length of the record; the tradable set is not touched. Learned: welfare adds no item to the tradable set and no fan-in; its arithmetic is `internal/welfare`, not the `internal/household` of `spec-agents.md` 22 |
| N-236 Three halves of FR-AGT-003, 010 and 011 are the agent row's (AGT) | **Leave open** for the agent row. Learned: A10 here keeps one multiplication by the weight, in `TickCost`, for `cost_t`; the agent row's product (what the agent buys and eats, FR-AGT-011) is its own and is not handed over, so the agent row's plan confirms that no row multiplies twice |
| N-237 A job stores its effective factor; its start tick is derived (PRO, CHK, AGT) | **Leave open.** This row edits CHK 4.2 tag 7 (the welfare section) and not tag 6 (the job, `Yield`), which is plan 1-5's. Learned: re-read the table just before the edit, because plan 1-5 and the ledger row edit the same table |
| N-043 Nothing anchors the price level if flows ignore price (AGT, PRO, SPN, BEL, STB); N-137 Room is committed at the decision (PRO, CLR, TRN, AGT); N-142 A job start leaves the upkeep room free (PRO, TRN, AGT) | **Leave open**, none touched: D69 states what "held by the floor" is and does not edit the floor rule of AGT 7.1.3 or the view; the room rules are STO's and the price anchor is STB's. Learned for N-043: the floor with its counter (`HeldByFloor`, the bounded `Warn`) is built so scenarios S04 and S16 can run it on and off |
| N-118 One job per agent limits the subsistence floor (PRO, WEL, SPN) and N-148 The employer role's observed wage includes its profit on hands (SPN, WEL) | **Leave open** (the WEL rows above). SPN 4.2 gains one clause about the welfare history at a role change (D64) and no other change; learned: the spawner row calls `Resize` at P7.1 and the occupation signal reads the per-role observed wage of this row |
| N-138 A role change moves excess goods to the estate (SPN, SKL) | **Leave open** for STO. The SPN 4.2 edit is the welfare-history clause only and leaves the `estate_goods` rule as written; learned: the ring resize and the move of goods are independent steps of the same role change, in either order |
| N-214 Keyed random streams (WLD, ..., WEL, AGT, CODE), the WLD part | **Apply for WLD 5.7.1 only in its reading of the target**: the first of the two creation words of the agent's creation stream, which WLD 5.7.1 and FR-WLD-066 already say; Phase 1 reads 5.7.1 and edits it only if it still says "the world's draw" (the update of 2026-10-12 lists 5.7 as applied). The other owners' rows (AGT D3, D5, D10, D12 and the rest) stay open |
| N-216 A site constant is never a small integer (WLD, CODE) | **Leave open.** This row adds no site: D1 is `siteConstruction`, built, and D10 and D12 are the callers'; `internal/welfare` takes the word and names no site |
| N-154 A count of households gates a structure (WLD, STB) | **Leave open**, no new exception. The floor `Warn` of D69 compares the held households with half the role's households: a diagnostic threshold, not a rate, share, probability, period or window of a decision, so WLD 9.4.1 is not edited; the window is a function of the recipes and `WindowCycles`, never of a count |
| N-125 A snapshot is a function of saved state (CHK, LED, WLD, CLR, OBS) | **Apply** to the new field: `RoleRow.ClampedHouseholds` is derived from the saved records (the ring's at-bound count is rebuilt by `FromRecords`, never saved), and the clamp run joins the diagnostic runs of CHK 4.2 tag 7 (deliverable 8), so a restored world's snapshot and logs equal the original's (FR-OBS-043). The note stays open for the other modules |
| N-127 Event classes at the observer (OBS, EXP, CODE) | **Leave open.** This row adds no event; the one `Warn` is not an event class. Learned: nothing here changes the allocation per event the note asks to measure |
| N-130 What IND cannot measure exactly (OBS, IND, WEL, ...), the OBS side | **Leave open** (the WEL row above). OBS 5.3.4 gains `ClampedHouseholds` only and adopts none of the six asks |
| N-159 Region-attributed treasury and fiscal flows need a reconciling item (FIS, IND, LED, OBS) | **Leave open** for FIS and IND. The new snapshot field is a count of households, not a money flow, and takes no part in the regional money identity |
| N-128 A hosted per-agent field is saved once, in the agent record (ALL, CHK), the CHK side | **Leave open** (the ALL row above). The ring is WEL's own section 7 of the checkpoint (`spec-checkpoint.md` F4), not a field of the agent record, and the clamp run is a diagnostic run per (region, role), not a per-agent field: the rule of the note does not apply to either |
| N-202 A module that keeps a key re-derives it at a reseed (CHK) and N-214, the CHK part | **Leave open.** The welfare record keeps no key and draws nothing at a reseed (the target is a creation word); the stale-marked passages of CHK (tag 1, 6.2 step 8, 8.4) are not edited here |
| The other open notes tagged AGT, SPN, WLD, OBS and CHK: AGT N-067, N-087, N-101, N-111, N-139, N-144, N-153, N-156, N-164, N-188, N-196; SPN N-089, N-096, N-122, N-153, N-172, N-192, N-193; WLD N-124, N-144, N-204, N-226; OBS none left; CHK N-129, N-204, N-215, N-225 | **Left open, not read as touched.** The clauses this row edits are AGT 3.8, SPN 4.2, WLD 5.7.1 (a check), OBS 5.3.4 and 7.3 and CHK 4.2 tag 7 with FR-CHK-043, and none of these notes concerns them; the index was re-read for the codes at the last revision and Phase 1 re-reads it for notes raised since |

**Tests:** none; documents only.

**Observability and rollback:** none, because no code changes. A wrong edit is
reverted like any document change; nothing depends on it until Phase 4.

**Acceptance:** WEL states D62 to D66, D69, D70, FR-WEL-042 and the creation-stream target; AGT 3.8
and SPN 4.2 carry the role-change clause; every open note tagged WEL, ALL, AGT, SPN, WLD, OBS or CHK at
that time has a row (the closing row names the ones left open untouched); the roadmap links this plan; the notes carry the dispositions and a fresh tag; the text
search of `doc/` finds no `'"'"'` (N-103).

## Phase 2: the pure arithmetic of `internal/welfare`

**Why here:** the ring of Phase 3 and the root of Phase 4 call these functions,
and each is a closed form that can be fuzzed alone with no state.

- **First, `doc.go`**: the glossary terms of N-017 (welfare ratio, the welfare
  window, working tick, working household-tick, own-use household-tick,
  observed wage, pooled regional wage, observed premium, earned ratio, work
  share, work floor, target ratio, record of a tick, ring), before any code or
  doc comment of this row uses them (architecture.md 3.2). The glossary guard
  must pass at the end of this first commit.
- The package `internal/welfare`, with a doc comment that says it knows no
  identifier and holds no state.
- **The saturating helpers (D63), first in the package and the only ones: three.**
  `satAdd(a, b)` is `checked.Add` that returns the extreme of the operands'
  sign when `ok` is false, and `satSub(a, b)` is the same over `checked.Sub`
  (the sign of the exact result: `MaxInt64` for a non-negative `a` and a negative
  `b`, `MinInt64` for a negative `a` and a positive `b`; one helper in two
  forms); `satMulDiv(a, b, c)` is `checked.MulDiv` (the
  product in 128 bits, one truncating division) that, when `ok` is false for
  a non-zero `c`, returns `MaxInt64` where the sign of `a * b * c` is
  non-negative and `MinInt64` where it is negative, so that **only the final
  value saturates**; `satMul` is `satMulDiv(a, b, 1)`; `satMulDivCeil(a, b, c)`
  is the same over `checked.MulDivCeil` (the ceiling of the wear cost, D66, the
  one place a ceiling is read), with the same sign rule. A zero divisor is the
  caller's absent case and is never passed. All three are unexported, built on
  `internal/checked` unchanged, and every function of this package that is
  called in the tick (everything below except the calibration formulas) uses
  them and has no `ok` result: its only absent result is the zero or
  non-positive denominator of D62. There is no other overflow helper in the
  package or in Phase 3, and none in the root, which cannot call these and
  calls the exported tick functions below instead (D63: "one set of helpers,
  no function that invents its own"). Unit tests, in an internal test file of
  the package (package `welfare`, architecture.md 10.2: they need unexported
  logic; the exported functions are tested from `welfare_test`): each helper at the extremes, a quotient in
  range from a product above `int64` (`MaxInt64 * 10 000 / 20 000` is
  `MaxInt64 / 2`, not saturated), a negative overflow, a zero numerator
  with a saturating factor (0), and the sign rule for each of the eight sign
  combinations.
- **Ratio** (WEL 2.1, 2.2): `(sum_income + pad_income) * 10 000 /
  (sum_cost + pad_cost)` with `pad_income = n * U * d_bp / 10 000`,
  `pad_cost = n * U`, absent when the denominator is 0 (2.2.3), truncation
  toward zero, negative allowed (FR-WEL-001, 007, 008, 010). Signature:
  `(bp int64, present bool)` and no `ok`: the sums are `satAdd`, `pad_cost` is
  `satMul(n, U)` (a named value of 2.2, saturated as a value), `pad_income` is
  `satMulDiv(pad_cost, d_bp, 10 000)` and the ratio is one
  `satMulDiv(income_sum, 10 000, cost_sum)`, so a ratio whose true quotient is
  in range is exact whatever the size of the product by 10 000. Both pads are in:
  a ratio that pads the cost and not the income would read far below
  `DefaultRatio` for a new agent. The earned form (2.1.7, FR-WEL-039) is the same
  function over the sums less the relief.
- **Calibration formulas** (3.1, FR-WEL-012, 013): the wage for a ratio and the
  ratio for a wage, the product before the one division, `D_year` and `D_work`
  parameters and never constants. The closed form of 5 (`D_work` rounded up and
  capped at the year, FR-WEL-022). Used by the viability row and tests; the
  running simulation does not call them, so these are the functions that keep
  `ok` (false on overflow and on a zero divisor, as `internal/checked`).
- **Work share** (5, FR-WEL-021): 10 000 below `T_lo`, 0 at or above
  `T_hi = T_lo * (10 000 + RestBand) / 10 000`, the proportional value
  between, and the executions `ceil(weight * share / 10 000)`; with a guard for
  `T_hi == T_lo` (RestBand 0, where "between" is empty). It uses `satMulDiv`
  and has no `ok`; inside the bounds of `bounds.go` it cannot saturate.
- **The floor test** (5.3, D65, FR-WEL-027): working ticks held against
  `MinWorkShare * T / 10 000`.
- **Target range** (5.1): given `TargetRatio` and `TargetSpread` return `lo` and
  `hi`. The draw is the root's; this package takes none (architecture.md
  7.7), and the caller passes `lo` and `hi` to the existing
  `stream.drawRange(lo, hi)` of `random.go`, so no second range implementation
  exists.
- **Wear cost** (2.1.4, FR-WEL-003, D66): the exported `WearCost(wearPerUse,
  price, durability, executions)` is `executions * ceil(WearPerUse * price /
  Durability)`: one ceiling per execution, then the product by the
  executions. It is called only at P6.3 by `record`, so it saturates (D63) and
  has no `ok`: the ceiling is `satMulDivCeil` (its extreme is `MaxInt64`: the
  factors are non-negative), and the product is `satMul`; it reads no `ok` of
  `internal/checked` itself. A zero `Durability` is refused by validation and is a
  programmer invariant here, not a result. It takes no weight (D66).
- **The tick functions the root calls** (D63; architecture.md 2.9: the arithmetic
  is here and `welfare.go` only converts). `TickCost(quantities, prices []int64,
  weight int64) int64` is the sum over the upkeep rows, in the slice order, of
  `quantity * price * weight` (each term `satMul(satMul(quantity, price), weight)`: the factors are
  non-negative, so a product that saturates in the first step saturates in the
  second and the result is the exact saturated value; the terms are then
  `satAdd`ed); the slices are equal in length by construction (a violated
  invariant otherwise).
  `TickIncome(credits, debits []int64) int64` is the sum of the credits less
  the sum of the debits, each sum in the slice order with `satAdd` and then
  one `satSub` of the debit sum from the credit sum; the
  root assembles the slices from the scratch fields in the order of the
  formula of 2.1 (receipts, in-kind, own-use receipts, wages, relief, the
  unearned sum; then the input spend, the wear cost, the wages paid, the rent
  and the interest paid). Neither returns an `ok`.

**Tests:** unit, in the external package `welfare_test` (the exported
functions; the package is pure), except the helper tests above.

- Ratio: a table with the arithmetic of each expected value in a comment
  (testing.md 5.5): a full window; `n` missing ticks padded at `DefaultRatio`
  10 000 and at 15 000; a negative sum; a zero denominator reported absent; a
  sum that is exactly 10 000 per tick; truncation toward zero for a negative
  numerator (FR-WEL-001, 007, 008). An explicit case: with no record held
  (`n` = `T`), `U` > 0, the ratio reads `DefaultRatio` exactly when
  `pad_income = n * U * d_bp / 10 000` is exact, and the pad truncates. At
  `DefaultRatio` 10 000 it is exact for every `T` and `U` (`pad_income` is
  `n * U`, ratio 10 000). At 15 000 it is exact when `n * U` is even: `T` 30,
  `U` 2 gives `pad_income` 30 × 2 × 15 000 / 10 000 = 90 and a ratio of
  90 × 10 000 / 60 = 15 000; `T` 1, `U` 2 gives 3 and 3 × 10 000 / 2 = 15 000.
  One non-divisible case states the truncated value (FR-WEL-007, testing.md
  5.5): `T` 1, `U` 1, `DefaultRatio` 15 000 gives `pad_income` 15 000 / 10 000
  = 1 (truncated from 1.5) and a ratio of 1 * 10 000 / 1 = 10 000, not 15 000;
  Phase 1 notes the truncation in WEL 2.2 beside FR-WEL-007 (deliverable 9).
- Formulas: the rows of WEL 3.2 and 5 reproduced exactly (1 460, 2 433, 167
  days) and a round trip within the one-unit truncation WEL 3.2 states
  (FR-WEL-012, 022). A test that varying the `D_year` and `D_work` arguments
  changes each result as the formula says, with the arguments the only source
  (FR-WEL-013); the absence of a literal is not asserted by text.
- Work share: below `T_lo`, at `T_lo`, midway in the band, at `T_hi`, above;
  RestBand 0; weight 1 against weight 40 for the requested executions
  (FR-WEL-021; the free-agent condition of FR-WEL-023 is the agent row's).
- Floor: working exactly at the threshold and one below; `MinWorkShare` 0 off;
  `T` 30 at 4 000 is 12 (FR-WEL-027).
- Target range: spread 0 gives `lo` equal to `hi` at `TargetRatio`; the lowest target at
  spread 9 999; the draw's bounds in the result (FR-WEL-024, 025 in part).
- Tick functions (D63): `TickCost` for a two-row upkeep at weight 3 (rows
  2 x 5 and 1 x 7 give (10 + 7) * 3 = 51) and a row whose product leaves
  `int64` reading `MaxInt64`, the later rows then saturating the sum, in the
  slice order; `TickIncome` of credits 900 and 40 less debits 200 and 40 is
  700, a credit sum at `MaxInt64` less a debit of `MinInt64` is `MaxInt64`,
  a credit of -5 less a debit of `MinInt64` is `MaxInt64 - 4`, not
  `MaxInt64 - 5` (the `satSub` rule), and empty slices give 0 (FR-WEL-002,
  010).
- Wear cost: a hand case, a zero `Durability` a panic whose message says it
  is a violated invariant, an overflow saturating to `MaxInt64` (FR-WEL-003,
  010); and the case that tells per-execution from aggregate
  rounding: `WearPerUse` 3, price 10, `Durability` 4, `executions` 2 is
  2 × ceil(30 / 4) = 2 × 8 = 16, where `ceil(2 × 30 / 4)` = 15 would be the
  wrong answer; `executions` 1 is 8; the weight is not a parameter.
- **Fuzz** (NFR-WEL-002, testing.md 7.4 by the NFR): arbitrary `int64` inputs
  do not panic, the calibration formulas report `ok` false exactly where the
  exact `math/big` value leaves `int64`, every other function equals a
  reference in the test that computes the formula's named values and its
  quotients exactly in `math/big` and clamps each to `[MinInt64, MaxInt64]`
  **once per named value** (the sums, `pad_cost`, and the final quotient from
  the unclamped 128-bit product, never an intermediate product inside a
  quotient), so a wrapped value and a saturate-then-divide both fail it; within
  the bounds of `bounds.go` the work share lies in [0, 10 000] and the ratio of
  a window of equal records equals the ratio of one. Corpus in
  `internal/welfare/testdata/fuzz/`.
- **Benchmarks** with `b.ReportAllocs()` and `b.Loop()`: ratio, work share
  (NFR-WEL-003). They report; they gate nothing.

**Observability and rollback:** none, because the package logs nothing, emits
nothing and holds no state; nothing imports it yet. Rollback is reverting the
PR.

**Acceptance:** `gofmt -l .` prints nothing; `go vet`, `staticcheck` and
`go test -race ./...` pass; the import guard and the determinism guard of
Stage 0 pass over the new package unchanged; no function of the package called
in the tick has an `ok` result, and the three helpers (`satAdd`,
`satMulDiv`, `satMulDivCeil`) and the calibration formulas are the only places
of the package that read an `ok` of `internal/checked`; `internal/welfare` is at 90%
statement coverage or more.

## Phase 3: the record, the ring and the aggregates

**Why here:** it needs Phase 2's ratio and is what the agent row hosts. The
aggregates need the per-agent sums the ring keeps.

- **`Record`** (D61): the eight fields of WEL 13 as primitive integers and a
  bool.
- **`Ring`**: a fixed-capacity buffer of the last `T` records, oldest first,
  with running sums of `income`, `cost`, `relief`, `unearned`, `ownReceipt`,
  `workHH`, `ownHH`, the count of working ticks and **the count of records
  that hold a field at the clamp** (maintained at a push, at an eviction and at
  a resize and rebuilt by `FromRecords`, so that "this ring holds a field at
  the bound", which `ClampedHouseholds` of D63 needs for every agent every
  tick, is a constant-cost read and not a scan of up to `maxWindow` records,
  NFR-WEL-001). `Push` (drops the oldest
  when full, adjusts every sum; clamps each field of the record to plus or
  minus `MaxInt64 / maxWindow`, `maxWindow` a parameter the root passes
  (`MaxWelfareWindow`; the package cannot import it, architecture.md 2.9,
  9.6) and no ring is longer than it, so no sum can overflow, and returns
  whether it clamped, D63; it never panics),
  `Resize(T')` (D64; allocates only when growing; it returns an error and
  leaves the ring unchanged for a `T'` below 1 or above `maxWindow`, which the
  proof of D63 would not cover), `Len`, an ordered read of the records for CHK,
  and `FromRecords(records, window, maxWindow)`, which rebuilds a ring and its
  sums from an ordered slice (the restore path, WEL 13 "not saved: the running
  sums"). **`FromRecords` validates its input** (architecture.md 7.7: a
  restore validates what it is given; a checkpoint is bytes from outside, and
  `Push` is the only clamp, so a record it never stored may be beyond it) and
  **returns an error, not a clamp**: a clamp-and-report would hide a corrupt
  checkpoint as a run that clamped, and no valid run produces a record beyond
  the bound, since `Push` stored every record of a ring. It refuses, with one
  package sentinel (`ErrRecords`, which the root maps to its own
  `ErrCheckpointCorrupt` in the checkpoint row, N-075) and no ring: a `window`
  below 1 or above `maxWindow`, a `maxWindow` below 1, more records than
  `window`, and a record with any field outside plus or minus
  `MaxInt64 / maxWindow`. An empty slice is valid and gives
  an empty ring of that window with zero sums. `NewRing(window, maxWindow)`
  has the same refusals for the two numbers, so every ring in existence is
  inside the proof; nothing in the package constructs one by another path.
  `welfare.go`'s `newHistory` treats the error of `NewRing` as a violated
  invariant (the window comes from a validated role, at most
  `MaxWelfareWindow`), and the checkpoint row returns the error of
  `FromRecords` as a corrupt checkpoint.
  The ring allocates its full capacity at creation (never at a push); a role
  at `T` 3 650 costs about 233 KB per agent (the Security section), which the
  world row's cost model (O2) includes.
- **Derived reads of one ring**: the ratio and the earned ratio (Phase 2 over
  the sums, with the padding from `T`, `d_bp` and the upkeep of the position of
  the read), the observed wage of the agent as numerator and denominator
  (`income - relief - unearned - ownReceipt` over `workHH - ownHH`, FR-WEL-018)
  with absent when the denominator is not above 0, and the working-tick count
  for the floor. Every named value of a read (the sum plus the padding, the quotient
  by the cost, the four-term numerator of the wage) is formed with the
  helpers of Phase 2 and follows the one rule of D63: it saturates to the
  nearest extreme, the quotient once; nothing wraps. Phase 3 adds no helper.
- **Aggregates over a set**, taking per-agent numbers in slices so the caller
  chooses the set. Every sum of the slices (numerators, denominators, weight
  times ratio) and the premium quotient follow the one rule of D63 through the
  helpers of Phase 2: saturating to the nearest extreme, in the slice's order,
  never wrapping and never absent for that reason; the premium is one
  `satMulDiv(role_wage, 10 000, unskilled_wage)`. Two wage functions, kept apart (D62):
  - the **observed wage of a set** (a role in a region; FR-WEL-018, 037): the
    sum of numerators over the sum of denominators, present iff the
    denominator is above 0, of either sign, truncated toward zero. This is what
    the per-(region, role) summary, `Wage` and `HasWage` of the spawner and the
    premium read;
  - the **pooled regional wage** (WEL 4.3): the same sums over every agent of
    the region, known iff the denominator is above 0 **and** the quotient is
    above 0. This is what the cost anchor reads, with the warm-start fallback
    of Phase 4.
  Then: the household-weighted mean ratio over the agents whose ratio exists
  with the count of those households (FR-WEL-036); the **held-by-floor
  predicate** `heldByFloor(ratio, present, workingTicks, T, MinWorkShare, T_lo,
  RestBand)` of D69 (free-agent status is the caller's) and its count over a
  `role_wage * 10 000 / unskilled_wage`, `unskilled_wage` the observed wage of
  the set of the unskilled agents by the same set function (pooled sums, one
  division: D62), absent when the set has no unskilled agent, when the role
  has no working household-tick, **or when the unskilled wage is not
  positive** (FR-WEL-019, D62).

**Tests:** unit.

- Ring: the sums, and the count of records at the clamp, after a scripted
  sequence of pushes, evictions and resizes equal those recomputed by
  `FromRecords` from the ordered read, after every step (the
  invariant that makes "not saved" safe, FR-WEL-034 in part). Push at capacity
  drops the oldest; a ring shorter than `T` pads; a resize down keeps the
  newest records and a resize up pads (D64); a push whose field exceeds the
  clamp is stored clamped, reports that it clamped, and every sum equals the
  recomputation from the stored records (D63); a read whose quotient overflows
  saturates; nothing panics.
- Restore path and construction (FR-WEL-034 in part, architecture.md 7.7,
  testing.md 3.8, 3.9; each case breaks only its rule and asserts
  `errors.Is(err, ErrRecords)`, not a message, `C` = `MaxInt64 / maxWindow`):
  `FromRecords` of a record with `income` `C + 1` and of one with `cost`
  `-(C + 1)` is refused (one field over by one each), and one at exactly `C`
  and `-C` is accepted and its sums equal the sums recomputed from the
  slice; a slice of `T + 1` records for a window `T` is refused and one of `T`
  is accepted; a slice longer than `maxWindow` is refused whatever the window;
  an empty slice gives a ring of the window with `Len` 0, zero sums, a ratio
  that is the padded `DefaultRatio` and a first `Push` that works; a window
  of 0, a negative window and a window of `maxWindow + 1` are refused, as is a
  `maxWindow` of 0, by `FromRecords` and by `NewRing`; a refusal returns no
  ring; `Resize` to `maxWindow + 1` and
  to 0 are refused and leave the ring and its sums unchanged, `Resize` to
  exactly `maxWindow` is accepted and a push after it still stays inside the
  proof (the sums equal a recomputation from the stored records, with every
  field at `C`).
- **Overflow of what is formed from the bounded sums** (D63, FR-WEL-010; each
  case breaks only this rule and carries its arithmetic in a comment, with `C`
  = `MaxInt64 / maxWindow` the clamp and `maxWindow` the parameter):
  - the per-agent numerator: a full ring of `maxWindow` records with `income`
    `C` and `relief`, `unearned` and `ownReceipt` each `-C` has a numerator
    `income - relief - unearned - ownReceipt` of 4 × `maxWindow` × `C`, above
    `MaxInt64`: it reads `MaxInt64` with a positive denominator, present, no
    panic, and the observed wage is `MaxInt64` over that denominator;
  - the set sums: two agents each at the clamp (numerators of about
    `MaxInt64`, so their sum is about twice that) give a set numerator of
    `MaxInt64` and the observed wage of the set is `MaxInt64` over the sum of
    their denominators, present, no panic, and the pooled regional wage is
    known and positive; two agents whose denominators are each `MaxInt64`
    give a denominator of `MaxInt64` and a wage of the numerator over it;
  - the order is the rule: the numerators `MaxInt64`, 5, -10 in that order sum
    to `MaxInt64 - 10` (the 5 saturates away) and in the reverse order to
    `MaxInt64 - 5`, so a test pins the slice order and the caller's order
    (ascending slot) is the one the world row uses;
  - the mean ratio: two agents of weight 2 whose ratios are `MaxInt64` (a
    saturated quotient) give a weighted sum that saturates, a mean that is
    defined and not a panic, `RatioHouseholds` 4;
  - the premium: a role wage of `MaxInt64 / 2` over an unskilled wage of 1
    has the exact quotient above `int64` and is `MaxInt64`, not a wrapped
    value and not absent; the same role wage over an unskilled wage of 20 000
    is `MaxInt64 / 2 * 10 000 / 20 000`, which is in range and is **not**
    saturated (a saturated product divided by 20 000 would be wrong)
    (FR-WEL-019, D62);
  - the numbers a valid configuration reaches are the two-agent and the
    no-recipe-role cases, because validation bounds only the sales of a role's
    recipes (`validate.go`).
- The reads: a hand-computed agent (a working tick with `k` executions in
  progress, an own-use tick, a relief tick, a grant tick) whose observed wage,
  earned ratio and ratio equal the spec formulas, the arithmetic in a comment
  (FR-WEL-018, 039, 040). A record with no recipes' working tick (weight 3)
  against a job of `k` = 2 of weight 3 (4.3).
- Aggregates: the pooled wage over agents of different weights is per household
  and independent of the weights (FR-SCL-009 as WEL 4.3 cites it); unknown with
  no worker and when not positive; the mean ratio ignores agents whose ratio is
  absent and rounds toward zero; the premium absent without unskilled agents,
  without working ticks, with an unskilled wage of 0 and with a negative one
  (FR-WEL-019, 036, 037).
- Observed wage against pooled wage: a role whose households worked at a
  negative profit has an observed wage that is **present**, negative, with
  `HasWage` true, and a pooled regional wage that is unknown, so the cost
  anchor falls back to the warm-start wage (FR-WEL-037, D62); a role with a
  working household-tick and a wage of exactly 0 is present at 0.
- `heldByFloor` and its count: floor off (`MinWorkShare` 0); ratio below
  `T_lo` (share already 10 000, not held); ratio absent (not held); ratio at
  `T_hi` (share 0, held if the floor condition holds); working ticks one below
  and exactly at `MinWorkShare * T / 10 000` (held, not held); a count in
  households of agents of weights 1 and 40 (D69, FR-WEL-020, 033).
- **Records stay with the agent** (FR-WEL-006, in part): a `Guard:` test that the
  ring has no region or role field, so nothing in it can be reset by a move;
  the behaviour itself (a relocated agent keeps its records) is the agent and
  world rows' through the public API.
- Allocation: `Push`, the ratio, the observed wage and the work share allocate
  nothing, with `testing.AllocsPerRun` as an ordinary test (NFR-WEL-001,
  testing.md 8.2). The cost of a read being constant in the window is asserted
  structurally, not by a benchmark (a benchmark reports and never fails,
  testing.md 8.2): the ratio read takes only the running sums and `T`, so a
  test builds rings of `T` 30 and `T` 3 650 with equal sums and asserts the
  same result and that the read function has no slice parameter; a benchmark
  beside it reports the timing.
- Fuzz of the ring: a byte string decoded into pushes, resizes and rebuilds from
  records (the field values drawn from the extremes as well), asserting the
  sum invariant for every ring that exists, that a rebuild is refused exactly
  where a field, the length or the window is out of range, and that nothing
  panics (NFR-WEL-002).
- Fuzz of the set functions (the observed wage of a set, the pooled wage, the
  mean ratio, the premium): slices of arbitrary `int64`, extremes included,
  asserting that nothing panics and that each result equals a reference fold
  written in the test with `math/big`: a sum clamps to `[MinInt64, MaxInt64]`
  after every term (the saturating sum of D63, in slice order), and a
  quotient or a premium is the exact `math/big` quotient clamped once, so a
  wrapped value (an unchecked `+`) and a saturated product that is then
  divided both fail it. Corpus in
  `internal/welfare/testdata/fuzz/`.

**Observability and rollback:** none, for the reasons of Phase 2. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds at 90%; the fuzz
target runs for five minutes by hand with no crasher (a crasher is committed
with a `Fix:` comment, testing.md 7.5).

## Phase 4: `welfare.go`

**Why here:** it needs Phases 2 and 3 and the role table, the calendar and the
effective upkeep of [plan-phase-1-2.md](plan-phase-1-2.md); the world row calls
what it builds.

- **The history type** the agent holds: `welfare.Ring` (A3), plus an unexported
  `newHistory(role)` that sizes the ring by the role's effective window. There
  is no tick-zero flag: padding is applied at read (WEL 2.2).
- **The effective window of a role** (WEL 2.3.3): `max(WelfareWindow,
  WindowCycles * L)` with `L` the longest `Duration` on the role's recipe list.
  There is one implementation: this phase appends `window int64` to `roleRow`
  and one `effectiveWindow(c, lk, r)` call to `newRoleTable` (`role.go`, after
  plan 1-2 Phase 5; plan 1-2 A2a keeps the row structs append-only), which
  stores the result once per role at construction, from the `lookup` that
  `newRoleTable` already builds (plan 1-2 A6), so the validator and the world
  cannot disagree and no second walk of the configuration exists. The reader
  is `(*roleTable).row(id).window`. The own-use and
  merchant terms join `L` inside that function with their modules. A test
  asserts the stored value against hand-computed literals (below), not against
  a second implementation.
- **The scratch of a tick's flows** (A4): receipts `R`, spending on `Input`,
  **wear rows** (commodity, `n` executions started, `WearPerUse`: the facts
  plan 1-5 D51 returns, not money; D66), in-kind, relief, unearned, own-use
  receipt, the working flag, the `k` of the agent's one job of the tick and
  whether it is an own-use job (set once, by P2.1 or by P2.7, D70), and a
  `traded` flag. The wear rows are a small fixed-capacity list merged by
  (commodity, `WearPerUse`), at most the tools of one recipe (`MaxRequires`),
  since an agent starts one job in a tick. The methods the other rows call
  (`addReceipt`, `addInputSpend`, `addWear(commodity, executions, wearPerUse)`,
  `markWorking(k, ownUse)` (once per tick per slot; a second call is a
  violated invariant, D70), `markTraded`, ...) and `record(prices, upkeep)`
  that values each wear row at P6.3 as `n * ceil(WearPerUse * price /
  Durability)` (D66) and `cost_t` at the same prices (the published price of
  the tick, the effective reference price where the market has never traded),
  forms `income_t` by the formula of 2.1 and the `Record` of D61. It converts
  the scratch into slices inline and calls `welfare.WearCost`,
  `welfare.TickCost` and `welfare.TickIncome` of Phase 2: `welfare.go` holds no
  arithmetic beyond that and no `checked` call, so no `ok` exists in it (D63:
  the clamp of `Push` and the `Warn` report an overflow). The formula
  is fixed over the scratch fields D61 already has (relief, unearned, own-use
  receipt), so a module adds its flow through an unexported scratch method that
  its own file calls at its position, and `record` holds no module logic
  (architecture.md 2.10.1, 2.10.2); only a flow that needs a field D61 lacks
  appends one.
- **The working tick** (WEL 4.3, FR-WEL-017, 018): one pure function
  `workingTick(hasRecipes, k, ownUse, traded, inTransit, weight)` returning
  `(working, workHH, ownHH)`. The agent has one job in a tick or none
  (FR-PRO-014, D70), so `k` is that job's executions and `ownUse` says whether
  it is an own-use job; `ownHH` is `k` when `ownUse` and 0 otherwise. A role
  with recipes works by a job in progress: `working` true, `workHH = k`. A
  role with no recipes works if it traded or travelled, `workHH = weight`, or
  if it ran an own-use job, `workHH = ownUseK` when it did not trade (so
  `workHH = ownHH`, denominator 0); with a trade and an own-use job
  `workHH = weight` and `ownHH = ownUseK`, so `workHH - ownHH` is not
  negative. Migration travel, rest and idle are not working ticks. A role
  with recipes on a **commercial trip** is a working tick by WEL 4.3 but the
  spec gives no household-ticks for it: that clause joins with `merchants.go`
  (Stage 2), `inTransit` is a parameter the slice never sets, and a trip
  changes nothing here. `hasRecipes` is read from the role table, never from
  an identifier. The world row calls it once per agent at P6.3 with the
  scratch's `k` and `ownUse` (set once, at P2.1 or P2.7, through
  `markWorking`), `traded` (set from the settled trades through `markTraded`,
  so a no-recipe agent that bought, an `Upkeep` or `Other` bid included, is a
  working tick) and transit state, and passes the result to `record`. The `Hireable` and
  principal clauses of 4.3 join with `employment.go`.
- **The cost of a tick** (FR-WEL-004, 005): the agent's effective upkeep (plan
  1-2's role table per region) times the weight times a price per commodity,
  through `welfare.TickCost` (D63; no `ok`); an away agent is valued at its account's
  region's prices. The wear rows of the scratch are valued by the same
  price slice in `record` (D66).
- **The creation of a history and a target** (FR-WEL-007, 024): the target from
  `TargetRatio`, `TargetSpread` and the caller's bounded draw (the first of the
  agent's two creation words of WLD 5.7.1, from its creation stream: D1 at
  construction, D10 at P7.1(d), D12 at a call; the stream is the caller's,
  this function names no site), and an empty ring padded by `DefaultRatio`.
  The draw is taken whatever the spread (WEL 5.1).
- **The wage of a region** (WEL 4.3): the pooled wage from the per-agent sums
  of the region, else the warm-start wage of `Config.WarmStart.Wages`, else
  unknown; a pure function of the rings and the configuration, no state. It
  returns `(wage, known)`; "unknown" has no value of its own, and the world
  row writes 0 for it into the cost-anchor table, which is how PRO 9.1.1 and
  FR-PRO-049 read it.
- **The floor run** (FR-WEL-033, D69): per (region, role) the count of
  consecutive ticks in which `2 * held > households`, `held` from the Phase 3
  count at P2.5 and `households` every household of the role in the region,
  returning whether to warn at the tenth tick and every tenth after. It reuses
  the `warnRun` constant of `observer.go` (a rule constant, 10) and no new
  literal. The run is per (region, role), kept in the world's slices (WEL 13),
  and the world row emits the `Warn`; no logger is called here. **The clamp
  run** (D63) is the same function over the count of households whose ring
  holds a field at the clamp (read from the ring's running at-bound count, not a
  scan; `ClampedHouseholds`, pattern A: the first tick
  of a run and every tenth after, so it takes the pattern-A form of the same
  helper with `warnRun`); the world row emits "welfare record clamped" from it
  after the push at P6.3, with `region`, `role` and `households`.
- **The summaries** the snapshot rows are built from (FR-WEL-020, 036, 037,
  A7): per (region, role) the **observed wage of the set** with its flag (not
  the pooled regional wage), the premium, the window, the held-by-floor count,
  the clamped-households count (D63), the mean ratio with its households. The
  region wage of the cost anchor is
  the pooled function, with its fallback, above.
- `doc.go`: the glossary terms were added at the start of Phase 2. Add here only
  a term that first appears in this phase's code or comments.

**Tests:**

- Unit, internal test (A8), hand-computed cases from small configuration
  literals (the packs `one` and `second` need `config/` and so live in the
  external test below):
  - the effective window: a role with a 60-tick recipe at `WindowCycles` 2 gets
    120, a role with only short recipes gets `WelfareWindow`, a role with none
    gets `WelfareWindow` (FR-WEL-009);
  - `workingTick`: a role with recipes and a job of `k` = 2 executions at weight
    3 is a working tick with `workHH` 2; a role with no recipes that traded
    (by a bid of purpose `Upkeep`) at weight 3 is a working tick with `workHH`
    3 and one that did not trade is not; an own-use job of `k` = 2 is a working
    tick with `ownHH` 2 counted in `workHH`; rest, idle and migration travel are
    not (FR-WEL-017, 018);
  - the one job of a tick (D70, FR-PRO-014), weight 4: a role job of `k` 3 is a
    working tick with `workHH` 3 and `ownHH` 0; an own-use job of `k` 3 for a
    role with recipes has `workHH` 3 and `ownHH` 3 (denominator 0, no wage);
    `workHH - ownHH` is never negative; `markWorking` called twice for one
    slot in a tick panics with the invariant message and is not a case of the
    rule (FR-WEL-017, 018);
  - a role with no recipes and an own-use job (D70), weight 4, own-use `k` 2:
    that traded gives `workHH` 4 (the weight) and `ownHH` 2, so the wage
    denominator is 2; that did not trade gives `workHH` 2 and `ownHH` 2,
    denominator 0 and no wage; neither is negative (FR-WEL-017, 018);
  - wear valued at P6.3: the price of the tool is 100 at P2.7 and 120 at P6.3;
    the record carries the cost at 120 and at the reference price when the
    market has never traded (D66, FR-WEL-003);
  - wear rounding and weight (D66): a tool of `WearPerUse` 3 and `Durability`
    4 at price 10, `k` 2 executions, weight 3, adds 2 * ceil(30 / 4) = 16 to
    `cost_t`, not 15 (one ceiling over the aggregate) and not 48 (a second
    factor of the weight); two rows of the same commodity with another
    `WearPerUse` (5: ceil(50 / 4) = 13, `n` 1) add 13 and stay a separate
    row (FR-WEL-003);
  - `record` forms `income_t` for a scripted tick (receipt 900, input spend
    200, wear 40 gives 660; upkeep valued at 1 000 gives 660 * 10 000 / 1 000 = 6 600 for a
    one-tick window) with the arithmetic in a comment, and an `Upkeep`, `Tool`
    or `Other` spend is not netted (FR-WEL-002 slice terms, FR-WEL-003);
  - the cost of a tick: a two-commodity upkeep, weight 3, an away agent valued
    at its account's region (FR-WEL-004, 005); an overflow saturates to
    `MaxInt64` and has no `ok` (D63), as do `record`'s income and the wear
    valuation: a scripted tick whose receipts or whose cost leave `int64` gives
    a record field at the extreme, which `Push` then clamps and reports;
  - creation: one word is taken at spread 0 and at spread 3 000 and the next
    word of the same creation stream (the residue) is identical at the two
    (FR-WEL-024, FR-WLD-066), the stored target lies in `[lo, hi]`, a target
    below 10 000 cannot be built (FR-WEL-025 as validation already
    guarantees);
  - the region wage: pooled when known and positive, the warm-start wage when
    not, unknown with neither; the same inputs give the same answer after a
    rebuild of the rings from records (no latch, FR-WEL-034 in part);
  - the floor run: ten ticks above half warns at the tenth and the twentieth
    and not between; a tick below half resets; **exactly half does not warn**
    (`2 * held = households`); a role with no households has no run
    (FR-WEL-033, D69);
  - D63: a `record` of a tick whose receipts exceed the clamp stores the
    clamped record and reports it, and does not panic; a household whose
    ring holds a field at the clamp is counted in `ClampedHouseholds` and the
    clamp run warns at the first tick and the tenth (FR-WEL-042 in part: the
    emission is the world row's, through the public API);
- External test (package `economysim_test`, A8, `export_test.go`), the packs
  `one` and `second` through `config/`:
  - the stored effective window of every role equals
    `max(WelfareWindow, WindowCycles * L)` computed in the test from the
    recipes' `Duration` and is at most `MaxWelfareWindow` (FR-WEL-009);
  - agreement (`Guard:`, cross-plan with [plan-phase-1-5.md](plan-phase-1-5.md)
    Phase 4): `internal/welfare`'s wear cost of an execution equals
    `internal/production`'s wear term of `UnitCost` over the tools of the packs
    `one` and `second` at their seed prices, so the two copies of the formula
    cannot drift (plan 1-5 owns the other copy and cannot test it before this
    row exists). `UnitCost` does not expose its wear term alone (PRO 9.1:
    `cost = input_cost + wear_cost + labour_cost + rent_cost`, then
    `ceil(cost * CostShare / (quantity * E))`), so the test isolates it: for
    each tool it builds a recipe with no inputs, `Duration` 1, `CostShare`
    10 000, one output unit and `E` 10 000 (a site factor of 10 000 and no
    shock), a known wage `w` above 0 (an unknown wage makes the whole cost
    0, PRO 9.1.1) and the tool at its seed price, so the unit cost is
    `wear + 1 * w`; the expected wear is that value less `w`, and it is
    compared with `WearCost(WearPerUse, price, Durability, 1)`. With a
    multi-unit output the ceiling of the division would blur the term, which
    is why the case is built with one unit.
- A `Guard:` test (`determinism_guard_test.go` of Stage 0 already covers
  `time`, global rand and `init`; this one asserts `welfare.go` and the
  files of `internal/welfare` read no `.ID` selector of a role, recipe,
  commodity or region and compare no string literal, FR-WEL-013,
  architecture.md 9.1: a text scan in the manner of that guard, over the
  same `guardedFiles` scope function, to which `welfare.go` and
  `internal/welfare/*.go` are added if they are not already covered).
- Allocation (NFR-WEL-001, testing.md 4.5, 8.2): an ordinary
  `testing.AllocsPerRun` test in the internal test asserting zero allocations
  for `record(prices, upkeep)` including the wear-row valuation, `workingTick`,
  the scratch methods (`addReceipt`, `addInputSpend`, `addWear`, `markWorking`,
  `markTraded`), the cost-of-tick function, the floor run and `heldByFloor`,
  and the region-wage function over prepared rings. The scratch and the rings
  are built before the measured closure, and the wear rows stay within their
  fixed capacity, so the run measures the recording of a tick and nothing
  else.
- No integration or end-to-end test: there is no world. The requirement IDs the
  table below marks "world row" are exercised through the public API there
  (testing.md 2.3).

**Observability and rollback:** the checklist of
observability-and-logging.md section 9 is answered in the PR. No log line, no
event and no metric is added; the "work floor binding" `Warn` and
`economysim.welfare.ratio` exist in the catalogue and the "welfare record
clamped" `Warn` is added to OBS 7.3 by Phase 1 (D63); all are emitted and fed
by the world row. Rollback is reverting the PR; nothing outside `welfare.go` calls
it until the agent and world rows.

**Acceptance:** the commands of Phase 2 pass; coverage holds at 90% for the
root and `internal/welfare`; the glossary guard passes with the new terms.

## Phase 5: the exit gate

- On the whole module: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`,
  `go test -short -race ./...`, the full run with
  `-covermode=atomic -coverpkg=./...`, `go mod tidy`; the per-scope coverage
  read and recorded.
- The pinned hash of the all-off configuration did not move (D5; nothing in
  this row touches it): the test is the existing
  `TestHashConfig_ReferencePinned`. `testdata/golden/off/` does not exist yet
  (`testdata/` holds only `chains/`), so that test alone is the gate until the
  world row records the first golden file; the files are then held by the same
  rule.
- The traceability check of testing.md 4.6 by text search: every ID in a
  `Requirement:` line exists in WEL, and every requirement the table below
  lists as covered in full is cited.
- The roadmap row set to **done** with its commit, and the notes of Phase 1
  updated with what the code settled.
- Not gating, if time allows: a mutation pass over `internal/welfare`
  (testing.md 9.4, a SHOULD) with `gomu` inside Docker only,
  `--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS` and `--workers` equal
  to the CPU cap.

**Tests:** the gate itself. **Observability and rollback:** none; rollback is
reverting the PRs in reverse order, and no other row has started to use them.

**Acceptance:** every command is clean; the traceability check finds no ID
without a spec row and no "covered in full" requirement without a test; the
roadmap row reads done.

## Cross-plan contract

### Consumes

| Symbol, type, sentinel, spec edit or field | Provided by | Status |
|---|---|---|
| `Config.World.WelfareWindow`, `WindowCycles`, `RestBand`; `Role.TargetRatio`, `TargetSpread`, `DefaultRatio`, `ExpectedWorkTicks`, `MinWorkShare`, `Unskilled`; `Config.WarmStart.Wages` (`SeedWage`) | Stage 0, `config.go` ([plan-phase-0-foundation.md](plan-phase-0-foundation.md)) | Built (023cf97) |
| `MaxWelfareWindow`, the WEL bounds, `effectiveWindow` of `validate.go`, `ErrOverflow`, `ErrOutOfBounds` | Stage 0, `bounds.go`, `validate.go`, `errors.go` | Built |
| `internal/checked`: `Add`, `Mul`, `MulDiv`, `MulDivCeil` with `ok` | Stage 0, `internal/checked` | Built |
| The world's bounded draw of WLD 7.2.1 on an agent's creation stream (`siteConstruction` at tick 0 for D1 is built; `siteCreation` for D10 and D12's stream come with the spawner and integration rows, which pass the word in) | Stage 0, `random.go` (D1); `spawner.go` and `integration.go` rows (D10, D12) | D1 built; D10 and D12 **ASSUMED**: this row takes the word as a parameter |
| `WithTargetSpread` applied to the effective configuration before validation | Stage 0, `option.go` | Built |
| The role table: per role the recipe list with each recipe's `Duration`, the "has recipes" flag, the effective upkeep per region as `(commodity, quantity)` rows in ascending commodity order | [plan-phase-1-2.md](plan-phase-1-2.md) Phase 5 (`role.go`, `recipe.go`) | Provided; names fixed by the "Cross-plan contract" of plan 1-2: `(*roleTable).row(RoleID)` with `recipes` (`[]RecipeID`; "has recipes" is `len(recipes) > 0`) and `(*roleRow).upkeepIn(RegionID) []itemRow` (`itemRow{commodity, quantity}`, ascending commodity, per household); the recipe `Duration` is not in the recipe table: it stays in the configuration and is read by `RecipeID` (`cfg.Recipes[id].Duration`, plan 1-2 Phase 4) |
| `Calendar.TicksPerYear()` for the callers of the calibration formulas (`D_year`) | [plan-phase-1-2.md](plan-phase-1-2.md) A3 (`TicksPerYear() int64`) | Provided; used by tests and the viability row only |
| A published market price per (region, commodity) with the effective reference price as the fallback, to value `cost_t` and the wear cost | `marketReport` and `seedReport(price)` of [plan-phase-1-3.md](plan-phase-1-3.md) (the published price; the state is the world row's `market.go`), and `cfg.Commodities[id].ReferencePrice` for the fallback ([plan-phase-1-2.md](plan-phase-1-2.md) A10) | Provided in substance; this row takes a price slice by commodity, so any source fits |
| Accumulation of trade receipts `R_t` and of money spent on bids by `Purpose` (`Input`, `Upkeep`, `Tool`, `Other`) per agent per tick (FR-AGT-057) | `world.go` row (P4.1, from the settled trades) with the purpose from `agent.go`'s plan intents | **ASSUMED**: it calls this row's `addReceipt` and `addInputSpend`; the fixed Purpose set is AGT 6.3; the trades come from plan 1-3 Phase 5 (O5 closed) |
| The working scratch: set at P2.1 for a job continued or completed and at P2.7 for a start, the executions `k` in progress, the own-use executions and the wear **rows** (commodity, executions, `WearPerUse`: facts, not money; D66 values them at P6.3), and `traded` from the settled trades; an agent has one job in a tick (FR-PRO-014), so the scratch is set once per slot and a second `markWorking` is a violated invariant (D70) | `world.go` row, from the working executions and wear facts that [plan-phase-1-5.md](plan-phase-1-5.md) Phase 6 returns (its D51: production calls no other row; D51 hands over facts, so nothing is valued at P2.7) | Provided in substance by plan 1-5; the calls to `addWear(commodity, executions, wearPerUse)`, `markWorking(k, ownUse)`, `markTraded` and to `workingTick` at P6.3 are the world row's **ASSUMED** (WLD 6.4, 6.8 P6.3) |
| The agent type has a slot index and a weight, and holds the history field | `agent.go` row | **ASSUMED** (AGT 3.1) |
| A per-(region, role) diagnostic run slice in the world | `world.go` row | **ASSUMED** (WEL 13) |
| `roleRow.window int64`, appended to plan 1-2's `roleRow` and filled by one `effectiveWindow(c, lk, r)` call in `newRoleTable` (`role.go`) | This row, Phase 4, after [plan-phase-1-2.md](plan-phase-1-2.md) Phase 5 (A2a of plan 1-2) | Owed here |
| WEL 4.3, 2.3.3, 13, 5.3, 5.1, 2.1.4, 2.3.2, 11.3, 9 edits, `spec-checkpoint.md` 4.2 tag 7 (the clamp run) and the AGT 3.8 and SPN 4.2 clauses (D62 to D66, D69, D70, FR-WEL-042, OBS 7.3 and 5.3.4, the creation-stream target) | This row, Phase 1 | Owed here |

### Provides

| What | For | Contract |
|---|---|---|
| `internal/welfare` ratio, earned ratio and padding: a function of the window sums, `T`, the held count, `d_bp` and the current upkeep, returning `(bp, present)` | `agent.go` (the view), `estates.go` (relief reads the ratio against `ReliefBelow`), `spawner.go`, `migration.go` | Truncates toward zero; absent when the denominator is 0; constant cost in `T` |
| `welfare.Record` and `welfare.Ring`: `Push`, `Resize`, ordered read, `FromRecords`, working-tick count, the running sums | `agent.go` (holds one per agent), `checkpoint.go` (Stage 2), `spawner.go` (resize at a role change) | The eight fields of D61, oldest first; sums are derived and never saved |
| Observed wage of one agent as `(numerator, denominator, present)`; the observed wage of a set (a role in a region) as `(wage, present)`, present iff the denominator is above 0, any sign; the pooled regional wage as `(wage, known)`, known iff the denominator and the quotient are above 0 | `migration.go` (agent), `spawner.go` and the `RoleRow` (set: `Wage`, `HasWage`), `production.go` (pooled: the cost anchor, with the warm-start fallback), `viability.go` | Per working household-tick, relief, unearned and own-use excluded, truncation toward zero (D62); the set read and the pooled read are different functions |
| `welfare.go` region wage: the pooled wage else `WarmStart.Wages` else unknown, no state | `world.go` row, which passes it as a slice to the cost-anchor table of `production.go` ([plan-phase-1-5.md](plan-phase-1-5.md) Phase 7, at P1.2) | A pure function of the rings and the configuration |
| The work share and the executions requested; the floor test | `strategy.go` (default `Work`), `agent.go` (the survival rule composes with it) | FR-WEL-021, 027; the world applies the floor, a consumer's strategy cannot hide it (AGT 7.1.3) |
| Target range for creation | `world.go` (construction, WLD 5.7.1) and `spawner.go` (new agents) | `lo` and `hi - lo + 1`; the draw is the caller's |
| The wear cost of an execution | `world.go` (calls `addWear` with the wear units `production.go` returns, [plan-phase-1-5.md](plan-phase-1-5.md) D51; `record` values them at P6.3); `production.go` does not call it: `internal/production` keeps its own copy of the formula for `UnitCost`, and the root test of Phase 4 here asserts the two agree (welfare is built after production) | `executions * ceil(WearPerUse * price / Durability)` (one ceiling per execution, no second weight factor, D66); saturates to the bound on overflow and has no `ok`, because its one caller is the tick (D63) |
| The effective window of a role | `agent.go`, `spawner.go`, `world.go` (snapshot `Window`) | `max(WelfareWindow, WindowCycles * L)`; modules append their term to `L` |
| The tick scratch with `addReceipt`, `addInputSpend`, `addWear(commodity, executions, wearPerUse)`, `markWorking(k, ownUse)`, `markTraded`, `workingTick` and `record(prices, upkeep)` | `clearing.go`, `agent.go`, `world.go` (P2.1 and P2.7 with production's facts; the trades; P6.3, where the wear rows and `cost_t` are valued at the tick's published prices, D66) | Cleared at P6.3 after the push; a module adds its flow by a scratch method called from its own file, not by editing `record` |
| The floor run, the held-by-floor predicate and count, and the per-(region, role) summary (observed wage and flag, premium, window, held, clamped households, mean ratio and its households) | `world.go` (P2.5 `Warn`, the P6.3 "welfare record clamped" `Warn`, snapshot `RoleRow` with `ClampedHouseholds`, `economysim.welfare.ratio`) | The same predicate (D69) at P2.5 over free agents and at the boundary (A7); the denominator is every household of the role in the region; the world emits |
| The calibration formulas of 3.1 and the closed form of 5 | `viability.go` (`WageBelowTarget`, FR-WEL-031) | `D_year`, `D_work` arguments only |
| The glossary terms of N-017 in `doc.go` | all later rows | Added before use |

## Security review

The module is a library with no network, file access, credentials or
persistence. This row adds no trust boundary: `internal/welfare` is called only
by the root with numbers it derived from a validated configuration and from the
ledger's flows.

- **Who can reach it.** A consumer reaches a history only through the world's
  queries and the strategy's view (later rows); the strategy sees the ratio, the
  target and the window, never the ring, and a strategy cannot write a record
  (AGT 6.4). Nothing here is exported from the module.
- **Integer overflow.** Every sum and product goes through `internal/checked`.
  A function the tick calls saturates a result outside `int64` to the nearest
  extreme through the three helpers of Phase 2 and has no `ok` (D63), because the
  world cannot refuse at P6.3; `ok` false is kept only by the calibration
  formulas run outside the tick (FR-WEL-010); the push of the tick's record
  clamps and reports (D63).
  Validation bounds only the sales of a role's recipes, so a flow it does not
  bound (a no-recipe role's receipts, relief, unearned) can reach a sum: the
  record's fields are clamped at the push and a read saturates (D63), so the
  sums provably fit and nothing panics; the fuzz targets and the boundary cases
  guard the claim.
- **Corrupt input.** A restore feeds `FromRecords` bytes from outside the
  process: it refuses a record beyond the clamp, a slice longer than the
  window and a window beyond `MaxWelfareWindow` with an error (Phase 3), so a
  hostile or damaged checkpoint cannot make a running sum wrap or allocate a
  ring beyond the bound.
- **Division by zero and by a non-positive.** A zero denominator (no upkeep, no
  working household-tick, `T_hi == T_lo`, an unskilled wage of 0 or below) is an
  absent result or a branch, never a panic and never a negative premium; each
  has a test (FR-WEL-008, 019).
- **Resource exhaustion.** A ring is at most `MaxWelfareWindow` records of eight
  fields per agent, bounded by the configuration (2.3.4); a resize allocates
  only when growing, at a role change. The bound is large: a ring at `T` 3 650
  is about 233 KB (3 650 records of 64 bytes), so 5 000 such agents hold over
  1 GB. The configuration, not this row, sets `T` (the effective window of a
  role follows its longest job, 2.3.3, and validation caps it at
  `MaxWelfareWindow`); the cost is O2 and the per-agent size is a number the
  world row's cost model must include (`spec-scale.md` 6). Not an exposure to
  a consumer's strategy, which cannot change a window.
- **Manipulating the record.** A strategy could try to look poorer to receive
  relief. The record is formed by the world from the ledger's flows, not by the
  strategy (AGT 7.1.3 makes the floor "a pure function of the welfare record,
  so a consumer's strategy cannot hide it"); the same holds for every term.
- **Not applicable:** authentication, authorisation, injection, secrets,
  cross-tenant exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer.
  Nothing in this row is exported from the module; `internal/welfare` is
  internal and `welfare.go` adds unexported identifiers only.
- **Document contracts.** Phase 1 adds text to WEL, AGT 3.8, SPN 4.2, OBS 5.3.4
  and 7.3, `spec-checkpoint.md` 4.2 tag 7 (the clamp run, D63) and the notes and
  renumbers nothing (testing.md 4.1); `doc/notes.md` gets a fresh tag. The
  checkpoint row of Stage 2 reads that inventory row.
- **What becomes hard to change, and when.** The shape of the record (D61) and
  the rounding rule (D62) enter the golden files at the end of Stage 1 and the
  checkpoint in Stage 2. Until the first golden file is recorded they may be
  corrected in a normal PR; after it, a change is a behaviour change with the
  explanation of testing.md 7.2.
- **Append-only stays true.** A module adds its flow through a scratch method
  called from its own file and its summary field in its own PR; with the module off it adds nothing
  (architecture.md 2.10.5, D5), so the pinned hash and, once the world row records
  them, `testdata/golden/off/` do not move.
- **Rollout** is one PR per phase (A9). Each leaves the branch passing every
  command of `AGENTS.md`. A phase is rolled back by reverting its PR; later
  phases depend only on earlier ones and no other row of Stage 1 has started
  to use this one.
- **Dependency failure.** None: no dependency and no run-time service. The one
  schedule risk is the clearing row's price source and the role table of plan
  1-2: Phase 4 takes both as parameters, so its functions can be built and
  tested with a test price slice and test role rows if those rows slip. The
  one part outside that fallback is the `roleRow.window` field and the
  `effectiveWindow` call in `newRoleTable`: `role.go` exists only after plan
  1-2 Phase 5 and this row must not create it, so that bullet waits for that
  phase (the effective window is meanwhile a free function over a slice of
  durations, tested alone, and nothing reads the stored field).
- **The Go toolchain.** `go.mod` says `go 1.25`. Nothing here depends on a
  version-specific order: no sort and no map iteration is used.

## Testing (summary, testing.md section 11)

The pyramid for this row is all base: unit tests in `internal/welfare` and in
the root's internal test of `welfare.go`, fuzz targets over the pure functions
and the ring, and allocation tests. There is no integration or end-to-end test
because there is no world; the world and agent rows add them.

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-WEL-001, 003 (the cost formula; the valuation at P6.3 is Phase 4), 007, 008, 010 (the pure functions, the three saturating helpers and the tick functions; the ring reads and the aggregates are Phase 3), 012, 013, 021, 022, 027; NFR-WEL-002, 003 | FR-WEL-024, 025 (range here; the draw and the stored target in Phase 4; the validation was Stage 0) |
| 3 | FR-WEL-018, 019, 036, 037, 039, 040 | NFR-WEL-001 (`Push`, the ratio, the observed wage and the work share here; the recording of a tick is Phase 4); FR-WEL-006 (the ring holds no region or role here; that an agent's records survive a relocation is the agent and world rows', through the public API, testing.md 2.3); FR-WEL-010 (the saturating aggregates and ring reads here, the one rule of D63); FR-WEL-034 (the rebuild here; the checkpoint section in Stage 2, whose inventory row Phase 1 edits); FR-WEL-020 (the aggregates here, the snapshot rows in the world row) |
| 4 | FR-WEL-004, 005, 009, 017 (the working-tick function, the cases of 4.3 and the no-recipe clauses of D70; the calls are the world row's), 033 (the run and the predicate), FR-WEL-024 (creation), FR-WEL-003 (the valuation at P6.3, D66) | FR-WEL-002, 041 (the slice terms here; each module's term with the module); FR-WEL-020 (the summary function here, the rows in the world row); NFR-WEL-001 (recording a tick, the scratch and the region wage here; the ring reads and `Push` were Phase 3) |

Built in Stage 0 and not re-tested: FR-WEL-014, 015, 016 (field defaults and
bounds), FR-WEL-025 (lowest target), FR-WEL-026 (`WithTargetSpread`),
FR-WEL-028 (floor against expected work), FR-WEL-032 (the default `TargetRatio` is 15 000, `defaults.go`). Deferred with their code:
FR-WEL-011 (a property of the world: nothing clamps income; a `Guard:` test in
the world row), FR-WEL-023 (a free agent only: the agent row), FR-WEL-029 and
030 (in kind, trainee board: not built, SKL), FR-WEL-031, 038 (the
viability row), FR-WEL-042 (the "welfare record clamped" `Warn` and its counter: the world row emits it, the run function is Phase 4's), FR-WEL-034 and FR-WEL-035 (checkpoint; the non-interference run
is the world row's).

"Covered in full" means at the lowest level that can verify it (testing.md
2.1). The second half of testing.md 2.3, the same behaviour through the public
API, is the world and agent rows': the mean ratio of scenarios S19 and S24
(`spec-reference-chain.md` 9), S04 with the floor on and off, and S11 for the
derived window are the integration tests of Stage 1's "Done when" and of
`spec-stability.md`.

Boundaries and error paths: every `ok` false of a calibration formula, every saturation and every absent result has a case
that breaks only its rule. Golden files: none yet. Fuzz: the pure functions and
the ring. Each PR states its coverage and that its tests were run against a
broken version (testing.md 9.3).

## Observability (summary, observability-and-logging.md section 9)

One `Warn` added, no event, no metric: the only failure path survived rather
than returned is the clamp of a record at the push (D63), and it has a `Warn`
"welfare record clamped" (row 26 of OBS 7.3, keys `region`, `role` and
`households`, pattern A, bounded by regions x roles) with the counterpart
`RoleRow.ClampedHouseholds`, both written in Phase 1 and emitted by the world
row (1); the welfare record is a measure read from the
snapshot and WEL 11.1 adds no event, and a tick's record is per agent so an
event would be per agent per tick (OBS 4.1, 2) (2); the work-floor decision is
the strategy-side rule whose silent action is counted by `HeldByFloor` and the
bounded `Warn`, both emitted by the world row (3); the loops log nothing and
build no event (4); no event, metric, unit or attribute changes (5, 6); neither
`Error` nor `Info` is used (7); no error is logged and returned (8); the tests of
section 8 that apply here are the allocation tests of Phases 3 and 4, and the
rest need a world (9).

## Open items

- **O1.** WEL open question 2: `MinWorkShare` and `WindowCycles` are guesses;
  scenarios S04 (floor on and off) and S11 set them, and they need the world.
  Not this row's to close; N-028 stays open.
- **O2.** The cost of the pooled-wage scan (A6) is O(agents) per tick and is
  unmeasured. The world row measures it against `spec-scale.md` 6 (N-198's
  neighbour for the saving drawdown) and decides whether a per-region running
  sum is worth its state.
- **O3.** No CI exists (O3 of plan 0). The gate of Phase 5 and the timed fuzz
  run are done by hand, on one architecture.
- **O4.** D62 (the forms of the wages and the premium; it changes the reading of
  FR-WEL-019, whose mean is of agents and becomes the pooled form), D63 (the
  clamp, the one saturation rule and the `Warn`), D64 (the length of the record
  at a role change), D65 (the floor counts the window and not the records
  held), D66 (wear valued at P6.3, and its arithmetic), D69 (what is held, and
  the denominator) and D70 (the working household-ticks of a role with no
  recipes, one job per tick) are spec-silent or change a spec reading and are made in Phase 1; the
  owner may override any of them before the phase that implements it: D65 and
  D66 before Phase 2, D62, D63, D64 and D69 before Phase 3, D70 before Phase 4.
  A saturated aggregate (D63) has no counter of its own beyond the clamp's: it
  needs sums past `MaxInt64` over a set and reads as the bound itself, which a
  snapshot reader can see; whether FR-STB-013 asks for a count is the owner's
  to say.
