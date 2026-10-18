# Plan: Stage 1, row 9: the spawner (the ladder, the entry rule, the default)

**Status: done (2026-10-18, uncommitted on stage-1-9; the exit gate of Phase 7 passed). Plan written (2026-10-17), audited. Two plan-review passes, no blocking item left (see "Review record"). Final rulings applied (coordinator, reversible): D112 to D119, O8 to O12; O9 ruled (a), nothing waits for the owner.** Sequences the ninth row of
Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`spawner.go` (`Spawner`, the
ladder, the entry rule, the default) with `WithSpawner`, and the name check
`ErrUnknownSpawner`", into seven phases. It implements
[spec-spawner.md](../specs/spec-spawner.md) (SPN) for the slice, with the
registration rules of `spec-world.md` 4.1 to 4.3 (WLD) and the name rule of
`spec-config.md` 7.1.4 (CFG). It is built against `doc/rules/architecture.md`
(the layout of 2.6, the `internal/` boundary of 2.9, the module-file rule of
2.10, the extension rules of 4.3 to 4.5 and 7.7, the fixed-point and overflow
rule of 3.5, the tie-break rule of 7.6), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

Rule tags read for this plan: architecture=905c0e578c4e662908638dd8,
observability=35827912d1e3141a76af3937, testing=b3559bc6d47470c221bc4642.
Notes tag (as read, 2026-10-17, before this plan's edits): 61bd3ab432f88e1b1af8c3a2.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR,
a ruling of the coordinator (2026-10-17, quoted where used) or the decisions of
[plan-phase-0-foundation.md](plan-phase-0-foundation.md). Where a decision is
still owed, Phase 1 makes it in the document that owns it, before the code that
depends on it. Decision IDs D110 to D119 are this plan's (plan 1-6 has D61 to
D66, D69 and D70); earlier plans' decisions are cited by file and ID. New notes
are N-264 to N-268.

No PRD exists for this row. The requirements source is the roadmap row and SPN.
The row is built after the ledger, goods, clearing, beliefs, production and
welfare rows and, in the order of the roadmap, after the agent row (7) and the
estates row (8); it needs no `World`, because every function here is tested on
hand-built views. Row 11 (`world.go`) owns the executor of the plan this row
produces.

Spec codes: SPN spawner, AGT agents, EST estates, WLD world, WEL welfare, PRO
production, CLR clearing, SKL skill barriers, SET settlements, SUB subsistence,
POP population, MIG migration, OBS observability, CFG config, CHK checkpoint,
STB stability, REF reference chain, GDS goods, CAL calendar. `FR-`/`NFR-` IDs
are the requirement IDs of those specs.

## What Stage 0 and the earlier rows already give, and what is left

| Concern | Where it already is |
| --- | --- |
| The eleven tunables `ReviewTicks`, `EntryShare`, `RoleHysteresis`, `RoleCooldown`, `EntryRatio`, `ChangePrivation`, `RemovePrivation`, `DeathStep`, `ExitShare`, `Renewal`, `EntryPoolShare`, the name `World.Spawner` (default `"default"`, hashed as a name) and `Role.Closed`, with defaults and bounds | `config.go`, `defaults.go`, `bounds.go` (FR-SPN-037) |
| The `ladder_order` rule (`ChangePrivation` below `RemovePrivation`, `ErrOutOfBounds`), the warnings `entry_share_window` and `no_open_role`, the overflow check of a warm-start wage against `ExpectedWorkTicks` | `validate.go` (FR-SPN-038, 039 in part) |
| The events `EntryLimited`, `WeightChanged`, `RoleChanged`, `AgentSpawned`, `AgentBankrupted`, `EstateOpened`; `Cause` with `insolvent`, `starved`, `consumer`, `distress`, `review`, `renewal`; the reason `unknown_agent` | `event.go` |
| `registry[T]` (`add`, `find`), `validID`, `ErrInvalidID`, `ErrDuplicateID`, `ErrNilExtension`, the pattern `WithClearing` / `resolveClearing` / `ErrUnknownClearing` | `option.go`, `errors.go` (plan 1-3 D36, coordinator Q3: the strategy and spawner rows reuse them unchanged) |
| The draw site `siteSpawner` (D9: tick, id the region), for a consumer's `Decide` | `random.go` |
| `internal/ordering.Bucket` (the review rotation of WLD 9.1.1), `internal/checked` (`Mul`, `MulDiv`, `MulDivCeil`, `MulDivFloor`) | `internal/` |
| The observed wage of a role in a region and whether it exists (`roleSummary.Wage`, `HasWage`), the ring `Resize`, `advanceRun` and `warnRun` for bounded `Warn` runs, `welfare.TargetRange` | `welfare.go`, `internal/welfare`, `observer.go` (plan 1-6) |
| The effective upkeep of a role per region (`roleRow.upkeepIn`), `Calendar.TicksPerYear` | `role.go`, `calendar.go` (plan 1-2) |
| The job `abandon` and `reweigh` steps and the residue layout | `production.go` (plan 1-5) |

What is left for this row: the `Spawner` surface (interface, view, plan,
`DefaultSpawner`, `WithSpawner`, `ErrUnknownSpawner`), the decision logic of the
default (signals, the ladder's rung 1 and rung 3, the review, the caps,
renewal), the pure support the executor calls, the report values of
`EntryLimited` and the runs of the three `Warn`s, and the spec, note and layout
edits those depend on.

## Scope

**In scope.**

- `internal/occupation` (new; standard library, `internal/checked` and
  `internal/ordering` only; primitives only): the signals `E` and `V`
  (SPN 6.1), the hysteresis comparison and the capital test of the tool half
  (SPN 6.2, SKL 2.5), the review rotation (the privation step is
  `household.NextPrivation`, O7), the
  rung-3 test, the exit allowance and the reduction (SPN 5.3), the scaling of a
  reduction, the ranking and the caps of the role changes (SPN 6.3), the renewal
  carry (SPN 7.1), the funded number (SPN 4.3, 7.3), the placement of an entrant
  (SPN 4.3), the refusal reason of a decision (SPN 4), the target carried over a
  role change (SPN 4.2) and the predicates of the three `Warn`s.
- `spawner.go` (root, core file, SPN): `Spawner`, `RegionView` (with `RoleView`
  and `RegionAgent`: the name `AgentView` is plan 1-7's strategy view, so the
  spawner's per-agent group is `RegionAgent`, N-248), `Decisions`, `Decision`, `DecisionKind`, `DefaultSpawner`
  and its `Decide`, the unexported scratch, `WithSpawner`, `resolveSpawner`.
- `spawner_report.go` (root): the tally and the value of `EntryLimited`, the
  runs of the `Warn`s "role extinct", "entry unfunded" and "group below floor",
  and the per-role signals the snapshot rows read. It emits nothing.
- `ErrUnknownSpawner` in `errors.go`; the spawner reason constants in
  `event.go`; the glossary terms in `doc.go`.
- The spec, note, layout and roadmap edits these depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
| --- | --- |
| **The executor** of a plan: the world takes each decision in plan order, checks it, and does the removal (AGT 3.6, the `removeAgent` fan-in of architecture.md 2.10.4), the reduction (AGT 3.7), the role change (SPN 4.2, AGT 3.8) and the entry with funding (SPN 4.3), emits the events and the `Debug` lines, and calls this row's support functions | Roadmap Stage 1, `world.go` row 11. SPN 20 and WLD 6.9 P7.1(d) home it there; coordinator ruling 2026-10-17, Q1 (A). The calls are in the Cross-plan contract |
| The per-agent state change inside a removal, reduction and role change (`Privation`, `RoleSince`, the job, residues, history, beliefs, strategy, target) | Roadmap Stage 1, `world.go` row 11: its executor composes the primitives of rows 5, 6 and 7 (`job.abandon`, `job.reweigh`, `setResidues`, `history.Resize`, `reseedBeliefs`) and `occupation.Retarget` inline; plan 1-7 provides no combined role-change or weight-change step (coordinator ruling 2026-10-17 (d); consistency pass) |
| The pool balance, the draw of an entrant's money from the pool, the estate listing of what a leaver leaves | Roadmap Stage 1, `estates.go` row 8 (ASSUMED names below) |
| Building the view each region and tick (the per-role wage, upkeep and households, the per-agent privation), calling `Decide`, the keyed `*rand.Rand` of D9, the order of P7.1, emitting every event and `Warn`, the checkpoint of `Privation`, `RoleSince`, the renewal carry and the three runs | Roadmap Stage 1 `world.go` (row 11) and Stage 2 `checkpoint.go` |
| The population hook: `PopBirths`, `PopDeaths`, the `births` and `population` entries, the effective `Renewal` of 0 while the module is on, `renewal_with_population` | Roadmap Stage 2, `population.go` (POP). Appended with the module (D112) |
| The settlement gates: `Role.Needs`, `Role.Ceiling`, the headroom and `NoLimit` of the view, `EntryLimited.RoleCeiling` | Roadmap Stage 3, `settlements.go` (SET). Appended with the module (D112) |
| Traineeships: `Apply`, the queue, `InTraineeship`, `Surplus`, `QueueAhead`, `Trainers`, `MaxTrainees`, the queue-discounted rule of SKL 4.5, the refusals `in_traineeship` and `needs_weight_one`, the `traineeship` cause | Roadmap Stage 3, `skills.go` (SKL). Note N-264. In the slice a role with `TrainingTicks` above 0 is never entered (D111) |
| The own-use flag of the view and the `OwnUseJobs` reset | Roadmap Stage 2, `subsistence.go` (SUB), appended with the module |
| The refusal `on_journey` and the arrival that sets `RoleSince` | Roadmap Stage 2, `migration.go` (MIG), appended with the module |
| External agents: the refusal `external_agent` is built here as a pure reason; `AddAgent`, `Mortal` and the mailbox | Roadmap Stage 2, `integration.go` (INT) |
| The scenarios S06, S14, S26, S27 and the acceptance bands (the occupation cobweb, the ladder, the famine) | Roadmap Stage 1 "Done when" with rows 11 and 12 (golden); `spec-stability.md`. This row ships the unit-level worked cases of SPN 15 |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents or by the coordinator already:

| ID | Decision | Source |
| --- | --- | --- |
| D110 | **Division of the spawner between rows.** This row builds the types, `DefaultSpawner.Decide`, `internal/occupation`, the registration and the support functions of the executor; row 11 owns the executor, row 7 the per-agent state change inside it, row 8 the estate and pool calls. Each support function is listed with its signature in Provides | Coordinator ruling 2026-10-17, Q1 (A); SPN 20; WLD 2.10.4, 6.9 P7.1(d) |
| D111 | **Only the tool half of SKL in the slice.** The capital test of SKL 2.5 is built (`ToolPrice`, `CapitalReturn` in the view, the annual gain read as in SPN 6.2). `DecideApply` is **not declared** (O10, ruled (a): SKL appends it with the `Apply` emission in Stage 3); a role with `TrainingTicks` above 0 is never entered by review, rung 1 or replacement. Stage 3 `skills.go` appends the queue rule of SKL 4.5 and the `Apply` emission (N-264) | Coordinator ruling 2026-10-17, Q2 (A); SPN 6.2 ("a role with `TrainingTicks` above 0 is entered only by `Apply`") |
| D112 | **Module fields are appended with their module.** The view and the decision carry only what the slice uses; SET's headroom and `Needs`, SUB's `OwnUse`, POP's `PopBirths` and `PopDeaths`, SKL's trainee fields and MIG's `on_journey` are appended by their module, in module order. The `Cause` constants `population`, `births`, `traineeship` are appended to `event.go` after the existing six by POP and SKL; nothing is inserted or renumbered (architecture.md 2.10.5: a constant of a closed set appended to a field is omitted from the golden rendering until a world uses it). `ErrUnknownSpawner` sits beside `ErrUnknownClearing` | Coordinator ruling 2026-10-17 (the three assumptions accepted); architecture.md 2.10.5; N-017 |
| D113 | **The registration is the clearing row's, unchanged.** `WithSpawner(name, s)` is `o.spawners.add(name, s, s == nil, "default")`; `resolveSpawner(o, name, path)` returns `DefaultSpawner{}` for `"default"`, the registered extension, or a `*ValidationError` with rule `ErrUnknownSpawner`; `NewWorld` (row 11) calls it once after `Validate` succeeds. Order of the checks: `ErrInvalidID`, `ErrDuplicateID`, `ErrNilExtension` | WLD 4.1.1, 4.3; plan 1-3 D36; N-229 |
| D114 | **`internal/occupation` names no identifier.** It sees integers and slices; a decision names an agent or a role by the caller's index (`Ref`); a refusal is a small integer constant of its own that the root maps to a `Reason` (the pattern of N-225). The root converts inline at the call (architecture.md 2.9) | architecture.md 2.9; SPN 20 |

Made by this plan, to be written into the owning spec in Phase 1 before code.
**FINAL (ruled by the coordinator, reversible; reasons in O4, O8 to O12).** D112
to D119 are final: nothing waits for the owner, and the owner may still reverse
one later by a new ruling and a spec edit:

| ID | Decision |
| --- | --- |
| D115 | **Signals: rounding, absence, overflow.** `E` and `V` truncate toward zero (architecture.md 3.5 default; FR-SPN-017 states no rounding). A role has **no signal** when `HasWage` is false, when `Upkeep` or `TicksPerYear` is not above 0 (nothing to divide by, the `no_upkeep` case of a world with no upkeep), (that is the only absence besides `HasWage`). **A quotient beyond `int64` saturates to the nearest extreme** and is never silent absence: `E` is the exact truncated quotient `wage * expected * 10 000 / (upkeep * ticksPerYear)`, computed in two cases so that it saturates only when the exact value leaves `int64` (`expected * 10 000` fits: `expected` is at most `MaxTicksPerYear`). Case 1, `den, ok := checked.Mul(upkeep, ticksPerYear)` fits: `E = checked.MulDiv(wage, expected * 10 000, den)`, and an `ok` false means the exact `E` is beyond `int64`: saturate by the sign of `wage`. Case 2, `den` is beyond `int64`: then `upkeep` is above `MaxInt64 / MaxTicksPerYear` (about 9.2e13) and the numerator is at most `MaxMoney * 10^9` (1e27), so the exact `E` is below 1e14 and fits; compute it as `checked.MulDiv(wage, expected * 10 000, upkeep)` (a quotient below 1e13) followed by a truncating division by `ticksPerYear`, which equals the single quotient for a positive divisor (floor of a floor). No step of case 2 can report `ok` false for a wage within bounds; if one does (a wage beyond `MaxMoney` given by a hand-built view), saturate by the sign of `wage`. `V = checked.MulDiv(wage, 10 000, upkeep)`, saturating by the sign of `wage` on `ok` false (`spec-world.md` 3.3.3: handled where it occurs, no panic, no error, and a huge ratio is what the saturated value says). The observed wage may be 0 or negative with `HasWage` true (plan 1-6 D62): `E` and `V` are then 0 or negative and the role is not viable above `EntryRatio` 0. **The agent's own role without a signal counts as `E_o` 0** (FINAL, ruled by the coordinator, reversible: an unrated own role earns nothing, so leaving a role requires evidence of a better one, a destination with a signal; a role in which nobody worked earned nothing): a viable role with a signal then beats it, which lets a failing role be left; a destination needs a signal, so nothing moves before some role has one (a cold start has none). The hysteresis comparison `E_d * 10 000 >= E_o * (10 000 + RoleHysteresis)` is computed as `E_d >= ceil(E_o * (10 000 + RoleHysteresis) / 10 000)` by `checked.MulDivCeil` (exactly equal for integers, SPN 15.1 gives 9 041); a quotient beyond `int64` decides by its sign (the two saturated sides then compare by sign only, which is the saturated value's meaning), so `Beats` needs no 128-bit code. The capital test is the SKL 2.5 inequality exactly as stated, `diff * ut >= price * CapitalReturn` with `diff = E_d - E_o` (by `checked.Sub`; an overflow decides by the exact sign of the difference) and `ut = upkeep * ticksPerYear`, both products formed in 128 bits with `math/bits.Mul64` (stdlib; the one place this row needs 128-bit code; O12 ruled (a)). A `toolPrice` of 0 passes first, before any product is formed. `ut, ok := checked.Mul(upkeep, ticksPerYear)`; when `ut` itself is beyond `int64` the test is **denied** (the safe side; FINAL, ruled by the coordinator, reversible: conservative denial, never a wrap). `bounds.go` already makes this corner **unreachable for a validated configuration**: a list holds at most `MaxUpkeepItems` (16) items of at most `MaxQuantity / MaxWeight` (1 000) at a price of at most `MaxPrice` (1e9), so one list costs at most 1.6e13 and the effective upkeep (a role or world list plus a region's `extra_upkeep`) at most 3.2e13, below `MaxInt64 / MaxTicksPerYear` (about 9.2e13). So **no validation-bound request is made**; the denial is a defensive branch for a hand-built view, covered by a unit case with `Upkeep` 1e14 and `TicksPerYear` `MaxTicksPerYear`, and it is not a silent absence. With `diff` below 0 the left side is negative and the test fails (the right side is not below 0); otherwise each side is an unsigned 128-bit product (each factor is below 2^63) and the two compare exactly. |
| D116 | **An agent acts once per tick, and never in the tick it was created.** An agent given a rung-3 reduction or removal in this plan is not a candidate for a switch in the same plan (the plan order of SPN 10.2 puts exits first, so an agent at `RemovePrivation` with a viable role, waiting behind the entry cap, is reduced and not switched: recorded in SPN 6.4 and N-267). An agent is eligible for any action only when `RoleSince < tick` (FR-SPN-050, literally; with `RoleCooldown` 0 an agent changed in tick `t` would otherwise be reviewable in `t`) and `tick - RoleSince >= RoleCooldown`. An agent with `Unfed` 0 is given no reduction though its leaky `Privation` is at the threshold: the formula of SPN 5.3 gives a drop of 0 and a `Reduce` by 0 is `bad_households` |
| D117 | **The order of the candidates and what the counts mean.** `Admit` orders all candidates by one explicit total key (kind ascending, `Privation` descending, gap descending, `Seq`, `Seq2`; architecture.md 7.6), where kind 0 is a rung-1 switcher, kind 1 a reviewer and kind 2 a cross-role replacement batch. Rung 1 and reviewers have `Seq` the agent ID and `Seq2` 0 (SPN 6.3: rung 1 by `Privation`, then gap, then ID; reviewers by gap, then ID, with `Privation` 0 for the key). **A replacement is one batch candidate per leaver role (O9, ruled (a), FINAL):** `Households` = that role's `n_r` fallback households, `Privation` and gap 0, `Seq` the leaver's role index (unique per call) and `Seq2` 0, so batches are admitted in the role order of SPN 7.2 ("for each role in role order") and not by gap; the destination order is the open viable roles by highest `E`, ties to the lower role ID (SPN 7.2 rule 2). **Closed form:** the households of one batch are identical (weight 1, the same `E`, the same destination ordering), so admitting them one at a time is equivalent to taking, per destination in order, `max(0, min(remaining, regionCap - regionUsed, roleCap - used))` (a kind 0 or 1 candidate of weight above 1 may have overshot a cap, so a difference can be negative, and one at a time admits 0 there) and moving on to the next destination; this preserves FR-SPN-021's ordering exactly and keeps cost and scratch O(agents x roles) (NFR-SPN-001, 002), with no recorded deviation. Households left after the last destination are counted in `RegionCap` if the region cap is used up, else in `RoleCap`, as one at a time would count them. The first `Admit` call (kinds 0 and 1) and the second (kind 2) share the counts, in plan order (SPN 10.2). The gap is `E_d - E_o` of the candidate's best destination. A candidate whose best destination is full takes its next best that still qualifies (SPN 6.3). **An agent at rung 1 that is also due for a review is judged by the rung-1 rule only** (SPN 6.2: a reviewer is an agent "not already handled by rung 1"): kind 0, no band; if it has no destination it stays (SPN 6.4) and is not re-judged by the review rule in that tick. **Counting** (one count per candidate household, never more): `RegionCap` if the region cap refused it (checked first); otherwise `RoleCap` if every destination that qualified for it was full; otherwise none. `ExitShare` 0 turns the rung off and counts nothing; `EntryShare` 0 gives caps of 0 and every candidate is counted in `RegionCap`. `EntryLimited.Cooldown` is the households of agents due for a review, or at rung 1, who were held by the cooldown; `Hysteresis` is the households of reviewers whose highest-`E` candidate destination (open, with a signal, `TrainingTicks` 0, passing the capital test; no viability term, see O8) has `E_d > E_o` but fails the band of `Beats`; `ExitCap` is the households a rung-3 agent could not shed for the allowance. Both are households, not agents |
| D118 | **The renewal carry belongs to the world; `Decide` reads it, the executor advances it.** The view holds `RenewalCarry` per role; `Decide` sizes the entries by calling `occupation.Renewal` on a copy and emits them; the executor calls the same function with the households it actually removed and stores the new carry (saved in checkpoint tag 9). A refusal of a default decision is a bug (the default's decisions are functions of the view, SPN 10.1), so the two agree; a `Guard:` test of row 11 compares them (N-265). The carry state is not a field of `DefaultSpawner`, which has none (SPN 14) |
| D119 | **The counts of `Decide` reach `EntryLimited` through the plan.** `Decisions` holds, besides `List`, unexported counts (`RegionCap`, `RoleCap`, `Cooldown`, `Hysteresis`, `ExitCap`, `NoViableRole`, the last also per role for the `Debug` line "entry dropped"). Only the default fills them; the world merges them with the counts of the executor (`Changed`, `Entered`, `Removed`, `Unfunded`, `Refused`) into one `EntryLimited` per region per tick, omitted when every count is 0 (FR-SPN-035). A consumer's spawner is counted by the executor only. `RegionCap`, `RoleCap` and `ExitCap` are kept unconditionally (O(1) per candidate already computed, O11); `Hysteresis` is built only when the view's unexported `observed` flag, set once by the world when an observer is supplied, is true. The view carries an unexported `scratch` the world sets once per world; a view built by a consumer or a test without one allocates it on first use |

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The signatures of `internal/occupation` and the root names in the
  Provides table are fixed by this plan so that rows 7, 8 and 11 can be written
  against them (coordinator Q1); a later change of one is a cross-plan edit.
- **A2.** `internal/occupation` holds value types per fact, no map, closure or
  package variable (architecture.md 7.1, 7.5). Its sorts use
  `slices.SortFunc` with a total order ending in a unique key (7.6).
- **A3.** The names row 7 and row 8 give the calls of the Consumes table are
  ASSUMED and are checked against their plans once those exist; this row's code
  does not call them (the executor does), so a different name changes row 11
  only.
- **A4.** `spawner_report.go` is a second root file of the spawner, as
  `production_report.go` is of production, because Lizard counts a file of about
  500 lines of code as a warning (N-243). Phase 1 adds both to the layout.
- **A5.** The default `Decide` is split into helpers (`exits`, `candidates`,
  `switches`, `renewals`) that keep the order of SPN 10.2 and a cyclomatic
  complexity of at most 10 (N-243: `lizard -l go -C 10 -a 8 -L 100 -w`).
- **A6.** One PR per phase. Each leaves the branch passing every command of
  `AGENTS.md`.
- **A7.** The `Debug` lines "agent bankrupt", "role changed", "decision refused"
  and "entry dropped" are emitted by the executor from the executed result; this
  row's functions log nothing.

## Phase 1: settle the owed spec lines, the notes and the layout

**Why first:** D115 to D119 are rules the code implements; a code change that
invents them would put a result-affecting choice where the configuration hash
and the checkpoint cannot see it (architecture.md 9.5). The phase changes
documents only, so it starts by reading `doc/notes.md` in full and quoting its
tag, handles every open note tagged SPN and ALL and, for the specs it edits,
EST, AGT, WLD, SKL, OBS and CHK (below), and gives `doc/notes.md` a fresh tag
(`AGENTS.md`).

Deliverables, each in the document that owns it. Every edit re-reads the exact
lines first, because several rows edit the same files, and uses a small Edit.

1. **SPN 6.1, 6.2, FR-SPN-017, new FR-SPN-058** (D115). Rounding (truncation
   toward zero), the two causes of "no signal" (no wage, no upkeep or year), the saturation of a quotient, the own role without a signal
   as `E_o` 0, the equivalence of the hysteresis comparison and the `ceil` form,
   the exact two-case computation of `E`, and the capital test as the exact SKL 2.5 inequality `diff * ut >= price * CapitalReturn` (O12) (D115: it saturates only where the
   exact value leaves `int64`; the annual gain of the capital test is denied where
   `upkeep * ticksPerYear` leaves `int64`).
2. **SPN 5.3, 6.2, 6.3, 10.2, FR-SPN-013, 050, new FR-SPN-059** (D116). An agent
   with `Unfed` 0 gets no reduction; an agent reduced in the plan is not a switch
   candidate; `RoleSince < tick` for any action. SPN 6.4 and 10.2 also say that
   the plan order (SPN 10.2, FR-SPN-031) takes reductions before switches, which
   SPN 6.4 ("if none exists it stays, and the ladder goes on to rung 3") can be
   read the other way round: an agent at
   `RemovePrivation` that is reduced is not switched, even if a viable role
   exists and it is only waiting behind the entry cap; the plan lists this
   consequence for S14 and S27 in N-267.
3. **SPN 6.3, 7.2, 13.1.2, new FR-SPN-060** (D117). The order of the candidates
   (rung 1 by privation, then gap, then ID; reviewers by gap, then ID; the
   cross-role replacements of 7.2 after them, one batch candidate per leaver
   role of identical weight-1 households, in leaver-role order, not by gap, admitted
   in closed form as the one-at-a-time result, O9), the gap
   of the best destination, the meaning of `Cooldown`, `Hysteresis` and
   `ExitCap` in households, `ExitShare` 0 counting nothing, `EntryShare` 0
   counting `RegionCap`, and that a refused candidate household is counted once:
   `RegionCap` if the region cap refused it, else `RoleCap` if every qualifying
   destination was full. The same edit states that an agent at rung 1 that is
   also due is judged by the rung-1 rule only, that `DecideEnter` entries are
   appended per leaver role in role order (like-for-like entry, then its fallback
   entries), and (SPN 5.3) that an `Upkeep * w` beyond `int64` is insolvent.
4. **SPN 7.1, 14** (D118). The carry is the world's state, saved in tag 9; the
   view reads it, the executor advances it with the households removed. SPN 14
   and N-265 also list the per (region, role) "had at least one agent" bit that
   FR-SPN-042 needs for "role extinct" (`Extinct(agents, everHad, configured)`),
   saved in tag 9 with the three runs; the same bit is added to `spec-checkpoint.md`
   4.2 (the tag 9 "Spawner" row) and to FR-SPN-043, so the Stage 2 checkpoint row
   does not drop it.
5. **SPN 3.1, 3.2, 3.3, 13.1.2** (D112, D119, N-248: the Go name of the per-agent group is `RegionAgent`). The slice view and decision are
   the subset of 3.2 and 3.3 without the module-owned fields, each of which is
   marked "appended with its module"; `Decisions` carries the default's counts
   for `EntryLimited`; the unexported scratch.
6. **SPN 6.2, 6.4, FR-SPN-020, 023** (D111). The tool half is built with SKL
   2.5; a role with `TrainingTicks` above 0 is never entered by the default
   before the traineeship module exists; `DecideApply` is not declared (O10); SKL appends it.
   SPN 6.2 and FR-SPN-019 get one clarifying sentence: the reviewer's
   destination rule is as the spec has it (open, with a signal, highest `E`, the
   band, the capital test) with **no viability term** (O8, option (a), settled).
7. **SPN 3.1.3** (N-229, D113). Say "as `spec-world.md` 4.1.1" for the
   duplicate, empty and nil rules, and that `ErrUnknownSpawner` is returned by
   `NewWorld` as a `*ValidationError`.
8. **SPN 4** (the refusal reasons). State the order of the checks as the order
   of the table (`unknown_agent`, `unknown_role`, `external_agent`,
   `same_role`, `in_traineeship`, `on_journey`, `needs_weight_one`,
   `bad_households`, then `unsupported_cause` appended last: a `DecideReduce` whose cause is neither `starved` nor `insolvent` in Stage 1, which a later module extends with its cause and ledger reason together), that `unfunded` is not a refusal but a partial result, and
   which constants the slice declares (the `Refusal` values and `Reason`
   constants of `in_traineeship`, `on_journey`,
   `needs_weight_one` are appended with SKL and MIG).
9. **SPN 4.2 and 20** (the where-this-lives table). Name `internal/occupation`
   and the file split `spawner.go` / `spawner_report.go`; say that the executor
   is the world's and which pure functions it calls; add the target carried over
   a role change as a function of this row (plan 1-6, Scope row: "the target of
   an agent that changes role ... is the spawner's").
10a. **SPN 20** (ruling of the consistency pass). The row "pure rules: `E`, `V`, the caps, the apportionment of allowances, the carry, the privation update" loses "the privation update", which moves from `internal/occupation` to `internal/household` (AGT 22, plan 1-7); SPN 5.1 and FR-SPN-012 cite `household.NextPrivation` as the function that sets `Privation` at P6.1.
10. **`rules/architecture.md` 2.6 and `AGENTS.md` layout** (fresh RULE-TAG).
    Add `internal/occupation/` to the list of packages that own the core rules
    over primitive structs, and `spawner_report.go` beside `spawner.go`. Coordinate
    with the other rows of Stage 1, which edit the same lines (`internal/household`
    of AGT 22, `production_report.go` of N-243): re-read, edit only the entries of
    this row, replace the tag once.
11. **`doc/documentation_index.md` and the roadmap**: the row of this plan (done
    when the plan was written); the status changes with the phases.
12. **Notes.** The disposition table below is written into `doc/notes.md`, N-264
    to N-268 are in place (raised with this plan), and a fresh tag is given.
    Re-read the index at Phase 1 time for any note tagged SPN, ALL, EST, AGT,
    WLD, SKL, OBS or CHK raised since, and add a row.

**Notes disposition** (apply, reject or leave open, with what was learned):

| Note | Disposition |
| --- | --- |
| N-016 Keep the parameter registry true (ALL) | **Leave open.** This row adds no field and no default: Stage 0 built the eleven tunables and `Role.Closed`; the registry test and its `notBuilt` list do not change |
| N-017 Words and names (ALL) | **Apply.** The terms of this row go to the `doc.go` glossary before use (Phase 2): spawner, failure ladder, rung, privation, bankruptcy (qualified: the rung-3 removal), reduction, role change, entry rule, entry limit, entry share, entrant, replacement, renewal, like-for-like, expected ratio, viability, signal, review, hysteresis (qualified: of a role), cooldown, exit allowance, headroom is SET's. `Spawner` leaves `notYetTerms` of `glossary_test.go` |
| N-027 No default macro constant is stable (ALL) | **Leave open.** Every default of SPN stays a labelled placeholder; no test asserts one as stable; the models of SPN 16 are not re-run here |
| N-103 Running several spec writers in parallel (ALL) | **Leave open.** Process note; this plan was written beside siblings and aligned only through the coordinator |
| N-126 A new event or `Warn` adds a row (ALL) | **Leave open.** This row declares no event and no `Warn`: `EntryLimited`, `WeightChanged`, `RoleChanged` and rows 6 to 8 of OBS 7.3 exist since Stage 0 and the spec; the world emits them. The staged catalogue test is unchanged |
| N-128 A hosted per-agent field is saved once, in the agent record (ALL, CHK) | **Leave open.** `Privation` and `RoleSince` are the agent's hosted fields (SPN 14, CHK tag 3); this row holds no per-agent state. The renewal carry and the three runs are the world's (D118, N-265) |
| N-163 The word "treasury" (ALL) | **Leave open.** The words "pool" (the estate account) and "treasury" are used in the senses of the note; this row says "pool" only |
| N-217 D5 binds every module's change (ALL, CODE) | **Apply.** Nothing appends to the configuration, an event or a snapshot row: the pinned hash and `testdata/golden/off/` cannot move; the exit gate checks `TestHashConfig_ReferencePinned`. The appended `Cause` constants of POP and SKL are constants of a closed set (D112) |
| N-235 One combined architecture.md edit for the fan-in and admission touch points (ALL) | **Leave open.** The spawner adds no fan-in and no admission touch point (`spawner.go` is a core file); the edit waits for the first Stage 2 module PR |
| N-043 Nothing anchors the price level if flows ignore price (AGT, PRO, SPN, BEL, STB) | **Leave open.** Entry and exit are slow capped flows (`EntryShare`, `ExitShare`), not an anchor (SPN 16); learned: this row ships the counts (`RegionCap`, `RoleCap`, `ExitCap`) so S04 and S16 can read them |
| N-089 Model scripts of the demand, spawner and reference-chain specs (DEM, SPN, REF, CODE) | **Leave open.** Not this row's; the SPN models stay in `/tmp/economy-boss/spn/` and no number of them is re-run. The worked cases of SPN 15 are tested as arithmetic |
| N-096 The regional pool serves births, entry, relief and money flows (MON, POP, EST, SPN) | **Leave open.** The funding rule `EntryPoolShare` is the executor's; this row gives `Funded` and the advisory `Budget`. Learned: the budget is taken once after the module's deaths (C15) |
| N-118 One job per agent limits the subsistence floor (PRO, WEL, SPN) | **Leave open.** The default's rung 1 does not read `OwnUse` (SUB is Stage 2); a role change abandons a job with `role_changed` as for any job |
| N-122 First-come caps (SPN, MIG, MON, MER, STB) | **Leave open.** The caps of SPN 6.3 are first come in the order of need, counted by `RegionCap` and `RoleCap`; WLD's D87 (a keyed tie) is not adopted for them and the ID is the last key. Scenario S73 measures it |
| N-138 A role change moves excess goods to the estate (SPN, SKL) | **Leave open** for STO. The `estate_goods` rule of SPN 4.2 is the executor's and is not edited |
| N-148 The employer role's observed wage includes its profit on hands (SPN, WEL) | **Leave open** for EMP. The default reads `RoleView.Wage`, the observed wage of the role's set; the module appends its rule to the function that builds it |
| N-153 Gates by settlement class: `Role.Needs` and `Role.Ceiling` (SPN, POP, AGT) | **Leave open** for SET. Learned: the view's `Closed` is `Role.Closed` in the slice, and headroom and `NoLimit` are appended with SET (D112); no place in `Decide` reads a class |
| N-172 An idle site after its owner's death (SPN, ORG, FIS, CAP) | **Leave open.** Not this row's |
| N-192 A body's places are a second limit into the headroom (SET, SPN, POP) | **Leave open.** Not this row's |
| N-193 A completion of a traineeship is held without headroom (SKL, SPN) | **Leave open** for SKL. No traineeship exists in the slice (D111) |
| N-082 Relief and entry share the pool in time (EST) | **Leave open.** The order (relief at P1.6, entry at P7.1) is WLD's and holds; this row reads only the pool balance for the advisory budget |
| N-214 Keyed random streams: the edits owed (WLD, ... AGT, CODE) | **Leave open.** The default draws nothing (FR-SPN-029); a consumer's `Decide` gets the stream `(siteSpawner, t, region)` (D9, built); the entrants' creation words are the executor's (D10) |
| N-075 Mapping the ledger's sentinels (CFG) | **Apply for this row.** `ErrUnknownSpawner` is declared in `errors.go` and returned as a `*ValidationError`; the ledger's sentinels are not mapped here |
| N-224 Stage 0 guards and what they pin (CODE) | **Apply.** Phase 2 deletes `Spawner` from `notYetTerms`; Phase 4 extends `TestConstants_Values` with the spawner reasons and adds the contract-phrase test for the `Spawner` doc comment; the arrival of `ErrUnknownSpawner` with its interface is the "Handoff to Stage 1" row |
| N-248 The spawner's per-agent group is `RegionAgent` (SPN, AGT, CODE) | **Apply.** Phase 1 writes `RegionAgent` into SPN 3.2 (deliverable 5) and the plan uses it throughout; `AgentView` stays plan 1-7's strategy view |
| N-229 Registration names: one rule for the four options (CLR, WLD) | **Apply.** SPN 3.1.3 says "as `spec-world.md` 4.1.1" (deliverable 7); the row reuses `registry[T]` unchanged |
| N-243 Follow-up of the Codacy gate on plan 1-5 (CODE, PRO) | **Apply** to this row's files: Lizard before every push, the spaced separator in Markdown tables, the new root file in the layout (deliverable 10) |
| N-236 Three halves of FR-AGT-003, 010 and 011 are the agent row's (AGT) | **Leave open** for the agent row; this row does not multiply an upkeep by the weight except for the cause test `Money < Upkeep * w`, which uses `RoleView`/`RegionAgent.Upkeep` per household and the agent's weight |
| N-232 The job and the residues live on the agent (PRO, AGT, CODE) | **Leave open.** The role change resets the residues (the agent row's step, called by the executor); learned: the executor does the abandon and the residue reset in the one role-change step |
| N-255 What `GoodsListed` counts (EST, SPN, CAP, OBS) | **Apply for this row.** `GoodsListed` counts lots (coordinator ruling and plan 1-8 D101); the executor emits it from the lot count `acceptGoods` and `receive` return; this row names it in the Consumes table and computes none |
| N-258 What the world row calls on the estate (EST, WLD, CODE) | **Leave open.** The calls this row's executor makes (`pool`, `grant`, `acceptMoney`, `acceptGoods`, `receive`) are listed in Consumes with plan 1-8's names; N-266 lists what row 11 calls for the spawner |
| N-045, N-065, N-067 (silent dampers; closed or STB) | **Reject for this row as already settled:** the spawner's limits are `EntryLimited` since OBS; this row builds the counts (D117, D119) |
| The other open notes tagged SPN, ALL, EST, AGT, WLD, SKL, OBS and CHK | **Left open, not read as touched.** None concerns a clause this row edits (SPN 3 to 7, 10, 13, 14, 20, architecture.md 2.6); Phase 1 re-reads the index for notes raised since |

**Tests:** none; documents only.

**Observability and rollback:** none, because no code changes. A wrong edit is
reverted like any document change; nothing depends on it until Phase 2.

**Acceptance:** SPN states D111, D115 to D119, FR-SPN-058 to 060 and the
cross-references; SPN 20 names the files and the package; architecture.md 2.6 and
`AGENTS.md` list `internal/occupation/` and `spawner_report.go`, with a fresh
RULE-TAG; every open note tagged SPN or ALL has a row; `doc/notes.md` carries
the dispositions and a fresh tag; the text search of `doc/` finds no stray
shell-quoting sequence (N-103).

## Phase 2: `internal/occupation`, the arithmetic

**Why here:** Phases 3 to 5 call these functions, and the glossary terms must
exist before code and doc comments use them (architecture.md 3.2). The functions
are pure, so they are testable with no world.

Deliverables (package `internal/occupation`, `doc.go` states it holds the
arithmetic of the spawner over primitives and knows no identifier):

- The `doc.go` glossary terms of N-017 above, and the deletion of `Spawner`
  from `notYetTerms` in `glossary_test.go` (the change that builds a listed
  term does both, N-224).
- `signals.go`: `Expected` (`E`), `Viability` (`V`), `Beats` (the hysteresis
  comparison by `checked.MulDivCeil`, D115) and `CapitalOK` (the tool half of SKL 2.5, exactly `diff * ut >= price * CapitalReturn`: a `toolPrice` of 0 passes first; `ut = checked.Mul(upkeep, ticksPerYear)` beyond `int64` denies; `diff < 0` fails; otherwise both products in 128 bits by `math/bits.Mul64` and an exact compare; D115, O12 ruled (a)). `Expected` follows the two cases of D115 (one `Mul` and `MulDiv`, or the divide-by-`upkeep`-first path); the `ok` false fallback of case 2 is unreachable within bounds but is written as the one-line saturation by the sign of `wage` that D115 states (never an ignored `ok`, which would return 0 silently; architecture.md 3.5, 8.3), covered by a unit case with a hand-built wage beyond `MaxMoney`. The fuzz comparisons with `math/big` run only where the exact result fits `int64`, and for `CapitalOK` only where `E_d - E_o` fits `int64` (an overflowing difference is decided by its sign, a defensive branch with its own unit case).
- `ladder.go`: `Rung3`, `PastCooldown`, `ReviewDue` (over
  `ordering.Bucket`), `Allowance` (ceiling of `households * shareBP / 10 000`),
  `Exits` (the order of SPN 5.3: privation descending, then ID, a drop of
  `min(unfed, max(1, ceil(w * DeathStep / 10 000)))` cut to the remaining
  allowance, the cause test `Money < Upkeep * w` (a product beyond `int64` is
  insolvent: `Money` is at most `MaxMoney`), the households cut) and `Remaining`
  (`floor(x * wNew / w)` of a reduction).
- `entry.go`: `Renewal` (the carry), `Funded`, `Place` and `Retarget` (the
  target carried over a role change, with no draw).

**Reuse:** `internal/checked` for every product; `internal/ordering.Bucket` for
the rotation; no helper is copied. `Retarget` takes the range from
`welfare.TargetRange` through its caller, so no second welfare formula exists
(plan 1-6 Scope).

**Tests (unit, table-driven, base of the pyramid):** each function at the
boundaries (testing.md 3.8): `E` and `V` of SPN 15.1 (8 219 and 12 000), a zero
and a negative wage, `Upkeep` 0, a product beyond `int64`; `Beats` at the edge of
SPN 15.1 (9 040 fails, 9 041 passes) and with negative `E_o`; `CapitalOK` with
price 0, equal sides and an overflow; `ReviewDue` against SPN 15.2 (id 17, period
10, ticks 0, 10, 20); `Allowance` of SPN 15.3 (5 and 10); `Exits` of SPN 15.4
(weight 40, unfed 7, drop 4; weight 1 removed whole; `Upkeep * w` beyond `int64` gives cause insolvent), an allowance that cuts an
agent (the agent that crosses it is cut, the next waits), `Unfed` 0, equal
privation broken by ID; `Remaining` and the per-household share of FR-SPN-051;
`Renewal` of SPN 15.7 (0, 1, 0, 1); `Funded` of SPN 15.5 (1 household of 15 000);
`Place` with equal weights, a full host and none (the away host is MIG's); `Retarget` at
both clamps. Each expected value carries its arithmetic (testing.md 5.5).
**Fuzz:** `FuzzArithmetic` (no panic; `Expected` and `Viability` agree with `math/big` wherever the exact result fits (including inputs whose first quotient alone would exceed `int64`: `wage` near `MaxMoney`, `ticksPerYear` up to `MaxTicksPerYear`, `upkeep` near and above `MaxInt64 / ticksPerYear`) and saturate by the sign of `wage` only where it does not; `Beats`
agrees with `math/big`; `CapitalOK` agrees with `math/big` on the exact inequality where `upkeep * ticksPerYear` fits and denies where it does not; `Exits` never drops more than the allowance or the
weight); seeds with `f.Add`, corpus committed.

**Observability and rollback:** none; the functions log and emit nothing and
allocate nothing (an allocation test with `testing.AllocsPerRun` for `Exits`,
`Place` and `Beats` with a caller-supplied slice). Rollback is reverting the PR;
no other row uses the package yet.

**Acceptance:** coverage of `internal/occupation` at least 90% by its own tests;
the glossary guard passes with `Spawner` removed from `notYetTerms`; the SPN 15
worked cases hold; `go vet`, `staticcheck`, Lizard clean.

## Phase 3: `internal/occupation`, ranking, caps and predicates

**Why here:** the default's switches and its cross-role replacements share one
admission function, and the `Warn` predicates are needed by Phase 6.

Deliverables:

- `admit.go`: `Candidate{Kind int; Ref int (the caller's index, the last sort tiebreak); Privation, Gap, Seq, Seq2 int64; Households int64; Weight int64; Dests []Dest}` (the key fields of D117; `Seq2` is 0 in production and is a test-only tiebreak for the oracle's expanded households; `Households` is the whole count for kinds 0 and 1, which are admitted whole or not at all, and the batch size for kind 2; `Dest` is `{Role int; Qualifies bool}` in destination order), `Admission{Ref int; Admitted []DestCount}` (`DestCount{Dest int; Households int64}`: for kind 2 the admitted count per destination, for kinds 0 and 1 one entry with the candidate's households; Phase 5 deliverable 3 reads it), `Limits{RegionCap, RoleCap int64}` and `Admit`, which sorts the
  candidates by the key of D117 (a total order; unique within a call by
  construction: agent ID for kinds 0 and 1, the leaver's role index for kind 2),
  admits in that order while `regionUsed < regionCap` and the destination's
  `used < roleCap`, lets a candidate whose best destination is full take its next
  best, admits the candidate that reaches a cap (the cap is exceeded by at most
  one weight), and counts each refused candidate household once, by the rule of
  D117 (`RegionCap` first, else `RoleCap` when every qualifying destination was
  full). `used` is caller scratch, so a second call (the cross-role
  replacements, kind 2) continues the same counts. A kind-2 candidate is a
  **batch** (`Households` = `n_r`, weight 1 each) admitted in **closed form**
  (O9, ruled (a), FINAL): per destination in order it takes `max(0, min(remaining,
  regionCap - regionUsed, roleCap - used))` households (the differences can be
  negative after a weighted candidate overshot a cap), then goes to the next
  destination, and the households left are counted once each (`RegionCap` if the
  region cap is used up, else `RoleCap`). Scratch and cost are O(agents x roles),
  never O(replaced households) (NFR-SPN-001, 002; no deviation).
- `refuse.go`: `Refusal` (a closed set of small integers) and `Refuse`, which
  gives the first failing check in the order of SPN 4, or none. In the slice
  `Refusal` holds only the six slice refusals `unknown_agent`, `unknown_role`,
  `external_agent`, `same_role`, `bad_households` and `unsupported_cause` (last), with `Facts` holding only
  what those need; `in_traineeship`, `needs_weight_one` and `on_journey` are
  appended, with their `Facts` field and their `Reason`, by SKL and MIG (D112,
  N-268). So `refusalReason` is total over the slice's `Refusal`, and no dead
  branch for a module fact exists to cover. The same rule holds for the other
  `internal/occupation` structs: `Host` has no `Away` (MIG), `Candidate` and
  `Limits` have no trainee or headroom fields (SKL, SET).
- `predicates.go`: `Extinct(agents int64, everHad, configured bool) bool` (the
  condition of "role extinct", FR-SPN-042: no agent now, and the role had at least
  one agent or has a configured group; the world keeps the per (region, role)
  `everHad` bit, which Phase 1 adds to N-265 and SPN 14, and derives `configured`
  from the group list),
  `BelowFloor` (a group of weight above 1 with fewer than `MinAgents` agents,
  FR-SPN-047) and `Unfunded` (entrants due and none funded).

**Tests:** `Admit` against SPN 15.3 (a region of 1 000 households allows 5 changes;
a destination of 50 allows 1, of 400 allows 2), the reach-the-cap rule, the
next-best rule, rung 1 before reviewers, reviewers before replacements, equal gaps
broken by ID, replacement batches in leaver-role order (not by gap), a batch that spans several destinations, a batch cut by the region cap (the rest counted in `RegionCap`) and one cut by full destinations (the rest in `RoleCap`), a
candidate with three full destinations counted once under `RoleCap`, the region
cap counted before the role cap, `EntryShare` 0,
and a property over seeds (permuting the input order does not change the output,
because every candidate in the generated input has a distinct full key by
construction; no admitted count exceeds a cap by more than one weight; every
candidate household is admitted or counted exactly once). **Mandatory oracle test (O9):** a per-household reference implementation of `Admit`, written in the `_test.go` file (each kind-2 batch expanded to `n` weight-1 candidates, admitted one at a time by the plain rule above), must give the same admissions per destination, the same remaining caps and the same `RegionCap` and `RoleCap` counts as the closed-form `Admit`, over random inputs (a property over seeds, reporting the failing seed, and the fuzz target below), including cap boundaries (a cap equal to, one below and one above the demand; a cap of 0), equal gaps, the O7-style tie-break (equal keys but for the final `Seq`) and batches that span several destinations. `Refuse` one failing
case per slice reason and their order; the appended module reasons add their cases with their module.
The predicates at their edges. **Fuzz:** `FuzzAdmit` (the properties above and the oracle comparison).
`Requirement:` lines cite the FR-SPN IDs of the table below.

**Observability and rollback:** none; allocation-free with scratch (allocation
test). Rollback is reverting the PR.

**Acceptance:** `internal/occupation` at least 90%; the oracle comparison passes (and fails against a deliberately broken closed form); the property test passes
over 200 seeds, reporting the failing seed (testing.md 6.2); the fuzz target ran
60 s with no crasher and its corpus is committed.

## Phase 4: the surface in the root (types, registration, name check)

**Why here:** the executor (row 11), the view builder and the consumers need
the types and the registration, and registration is independent of `Decide`.

Deliverables:

- `spawner.go`: `Spawner` with its doc comment (the contract of architecture.md
  7.7 and the `Stateful` paragraph: no own random source, no global one, the
  `*rand.Rand` is not kept, state through `Stateful`, no sharing between worlds;
  FR-SPN-003), `RegionView`, `RoleView`, `RegionAgent`, `Decisions`, `Decision`,
  `DecisionKind` (`DecideRemove`, `DecideReduce`, `DecideSwitch`,
  `DecideEnter`), `DefaultSpawner` and the assertion
  `var _ Spawner = DefaultSpawner{}` (architecture.md 4.5). `Cause` stays as
  built (D112). The fields are those of SPN 3.2 and 3.3 without the module-owned
  ones (D112) and are listed in Provides.
- `option.go`: `options.spawners registry[Spawner]` and `WithSpawner`;
  `resolveSpawner` (D113), which returns an interface and so carries the
  architecture.md 4.4 reason comment as `resolveClearing` does (`option.go`), copying its wording.
- `errors.go`: `ErrUnknownSpawner` beside `ErrUnknownClearing`. Row 7 appends
  `ErrUnknownStrategy` to the same block: re-read before the edit and use a small
  Edit.
- `event.go`: the spawner reason constants appended, in the order listed here
  (not the SPN 4 table's), after the existing blocks: `ReasonUnknownRole`, `ReasonExternalAgent`,
  `ReasonSameRole`, `ReasonBadHouseholds`, `ReasonUnsupportedCause` (`unsupported_cause`, with its `Debug` text; the refusal appended last), `ReasonUnfunded`,
  `ReasonNoViableRole` (`ReasonUnknownAgent` exists and is reused). The values
  are `Debug` reasons and `EntryLimited` counts: none is an event field or a
  metric attribute, so `economysim.reason` (the order set) does not grow.
  `ReasonInTraineeship`, `ReasonOnJourney`, `ReasonNeedsWeightOne` are appended by
  SKL and MIG.
- `refusalReason(occupation.Refusal) Reason`, the one mapping (N-225 pattern),
  total over the six slice refusals (the module refusals are appended with
  their `Reason`). The comments of `ReasonUnfunded` and `ReasonNoViableRole` say
  they are "entry dropped" `Debug` reasons, not refusals, so `Refused` is not fed
  by them; the doc comment of `ReasonUnknownAgent` (written for an order) is
  widened to a decision of the spawner. The field comments of `EntryLimited`
  `Cooldown`, `Hysteresis` and `ExitCap` (`event.go`, now "the changes held" /
  "the removals cut") are widened to say households, as D117 defines them (a
  doc-only change, no field moves).
- `TestConstants_Values` gains a slice for the spawner reasons, each checked
  against `spec-spawner.md` as a backticked token; the contract-phrase test of
  `doc_comments_test.go` gains the `Spawner` phrases.

**Reuse:** `registry[T]`, `validID`, `ErrNilExtension`, `ErrInvalidID`,
`ErrDuplicateID` and the shape of `WithClearing` / `resolveClearing` are used
unchanged; no registry is re-invented.

**Tests:** (unit, internal and external) `WithSpawner` rejects an empty or
non-ID name (`ErrInvalidID`), `"default"` and a repeat (`ErrDuplicateID`), a nil
`Spawner` (`ErrNilExtension`), each checked with `errors.Is`, in that order;
`resolveSpawner` returns the built-in for `"default"`, a registered fake, and
`ErrUnknownSpawner` as a `*ValidationError` with the path and the value for an
empty or unregistered name; the name is in the hash through `World.Spawner`
(`TestHashConfig_ReferencePinned` does not move). A hand-written fake `Spawner`
in a `_test.go` file (architecture.md 10.3). `refusalReason` for every
slice refusal, `unsupported_cause` included (the function is total over them; a test enumerates `Refusal`'s values so a value appended without a mapping fails). The compile-time assertion is the test of FR-SPN-001. No `Example`
test: the public entry `WithSpawner` needs `NewWorld` (row 11) to show a result;
row 11 adds it (architecture.md 6.5, a SHOULD).

**Observability and rollback:** none (no state transition, no loop). Rollback is
reverting the PR; the surface has no caller until row 11.

**Acceptance:** FR-SPN-001 and the `resolveSpawner` rule of FR-SPN-040 pass; the pinned hash did not move;
`staticcheck` clean; the constants test passes with the spawner slice.

## Phase 5: `DefaultSpawner.Decide`

**Why here:** it needs the pure functions of Phases 2 and 3 and the types of
Phase 4, and it is the largest piece of logic, so it comes when everything it
calls is tested. Nothing here waits for the owner: O9 is ruled (a), the kind-2 replacements are admitted as a closed-form batch (deliverable 3).

Deliverables (`spawner.go`, helpers split per A5):

1. `exits`: collect the leavers (not external, `Rung3`, `RoleSince < tick`,
   `Unfed` above 0), call `occupation.Exits` with the region's allowance, append
   `DecideReduce` or `DecideRemove` with cause `insolvent` or `starved`, remember
   the agents acted on and the households removed per role.
2. `candidates` and `switches`: for every agent that is not external, not acted
   on, past the cooldown and either at rung 1 (every tick) or due for a review
   (`ReviewDue`), compute the signals (D115), list the destinations in rank
   order (open, signal, `TrainingTicks` 0, and beats the own role: rung 1 by
   viable at `EntryRatio` and `E_d > E_o` with no band, SPN 6.4; a review by the
   hysteresis band and **no viability term**, the literal text of SPN 6.2 and
   FR-SPN-019, see O8; the capital test and the agent's money against `ToolPrice`
   for a role with a tool), and call `Admit` (kinds 0 and 1 of D117). Append
   `DecideSwitch` with cause `distress` (rung 1) or `review`.
3. `renewals`: per role in role order, `Renewal` on the carry of the view; the
   entries go to the same role if it is open and viable (uncapped, SPN 7.2 rule
   1); otherwise the role's fallback households are one batch candidate of kind 2
   (D117: `Households` = `n_r`, ordered by leaver role index) for the open viable
   roles by highest `E` (ties to the lower role ID) through a second `Admit` that
   shares the counts and admits the batch in closed form (O9); the households of
   a batch that no viable role can take are counted in `NoViableRole`, and the
   entries appended per destination role carry the batch's admitted count. Append `DecideEnter` with cause
   `renewal`, one per destination role, **per leaver role in role order: that
   role's like-for-like entry first, then the entries of its fallback households**,
   so the executor's funding in plan order lets earlier roles draw first (SPN
   7.3); a Phase 5 test with a short budget (the test-only applier funds in plan order) pins which entries are funded.
4. The unexported counts of D119.

The plan order is exits, switches, renewal entries (SPN 10.2; the module's
deaths and births are appended by POP at their places). The scratch (candidate,
destination and admission slices) is held in `v.scratch`.

**Tests:** (unit and a small integration of the pure plan) `Decide` on hand-built
views, one per rule: rung 3 leaves a weight-1 agent removed whole, a cohort of
40 with 7 unfed loses 4 (SPN 15.4), `ExitShare` cuts and `ExitShare` 0 is off;
the privation trace of SPN 15.6 through `household.NextPrivation` (plan 1-7 Phase 2, the one implementation; this row only reads `Privation`); a review respects the
rotation, the cooldown and the hysteresis band (SPN 15.1); rung 1 picks the best
viable role without a band; an agent at rung 1 and due is judged by the rung-1 rule only, and with no destination it stays and is not re-judged by the review rule (D117); `EntryShare` caps (SPN 15.3); a role with
`TrainingTicks` above 0, a `Closed` role and a role with no wage are never
chosen (FR-SPN-041, 042, D111); a role that needs a tool is chosen only if the
money covers `ToolPrice` and the capital test passes (FR-SPN-020); like-for-like
replacement is uncapped and the fallback is capped (FR-SPN-022, 025);
`Renewal` 5 000 gives entries 0, 1, 0, 1 across calls with the carry fed back
(SPN 15.7); an external agent is never decided on (FR-SPN-010); no agent is acted
on twice, none with `RoleSince == tick` (FR-SPN-050, D116); the plan order
(FR-SPN-031), with entries sized from the view alone and no budget taken in `Decide` (the taking is the executor's, FR-SPN-026). An empty region, a role with no households and a view with no roles give an empty plan and no panic. **Properties over seeds:** a seeded random view, `Decide` twice
gives the same plan, and permuting the `Agents` slice (IDs fixed) leaves the plan
unchanged because every tie ends in `AgentID` (FR-SPN-029, 030), with a case in
which equal-gap, equal-privation candidates are ordered by ID; the roles' order is
not permuted, because it fixes the renewal entry order by design (SPN 7.2); a test-only
applier of the plan to the view (not the executor) checks that no cap is exceeded
by more than one weight and that households are conserved. **Allocation:** with
a world-supplied scratch, `Decide` allocates nothing (NFR-SPN-001, `AllocsPerRun`).
**Fuzz:** `FuzzDecide` (no panic, every `Ref` in range, every `Households` at
least 1).

**Observability and rollback:** `Decide` logs nothing and builds no event: the
decisions are data, and the lines of A7 are the executor's. The decisions that
reduce, remove, switch and enter are state transitions with events
(`AgentBankrupted`, `WeightChanged`, `RoleChanged`, `AgentSpawned`) emitted by the
executor. The silent dampers are counted (D119). A non-interference test (a run
with a logger and observer attached makes the same decisions, NFR-SPN-003) is
row 11's, because `Decide` takes neither. Rollback is reverting the PR; the
function has no caller until row 11.

**Acceptance:** the worked cases of SPN 15.1 to 15.7 pass; coverage of the root
rises or holds; the allocation test passes; Lizard reports no function above
CCN 10 or 8 parameters.

## Phase 6: `spawner_report.go`

**Why here:** the world needs the report values and the run advance before it can
emit anything, and they depend on the counts of Phase 5.

Deliverables:

- `entryTally`: the executor's counts per region (`Changed`, `Entered`,
  `Removed`, `Unfunded`, `Refused`) added to the counts of `Decisions`, and
  `limited(tick, region) (EntryLimited, bool)`, which returns the value and
  `false` when every count is 0 (FR-SPN-035). It holds no map, is sized once by
  the number of regions and allocates nothing after.
- The runs of the three `Warn`s over the shared `advanceRun`: `extinctRun`
  (pattern A, first tick and every tenth), `unfundedRun` (pattern B, ten
  consecutive ticks then every tenth) and `belowFloorRun` (pattern A), each
  returning the new run and whether to warn (FR-SPN-036). The world keeps the runs
  per region or per (region, role) and saves them in tag 9.
- `roleSignals(r *RoleView, ticksPerYear, entryRatio int64) (e, v int64, hasE, viable bool)` (`viable` is `v >= entryRatio`; built in `spawner.go` beside `Decide`, which calls it, and read by the snapshot)
  and the largest privation over a role's agents, for the snapshot row
  (`ExpectedRatioBP`, `Viable`, `HasExpected`, `MaxPrivation`, FR-SPN-046). The
  snapshot row itself is row 11's.

**Tests:** `limited` omits the value when all counts are 0 and carries each count
otherwise, with the executor and the plan counts added; the runs reset when the
condition fails, warn at the first tick for the two pattern-A runs and at the
tenth for the pattern-B run, and saturate; `roleSignals` agrees with `Decide`'s
signals on the same view (one function, called by both). Allocation test of the
tally.

**Observability and rollback:** this phase is the observability of the row:
`EntryLimited` and the three `Warn` runs are produced here and emitted by row 11,
with the counterparts `EntryLimited.Unfunded` and `RoleRow.Agents` (OBS 7.3 rows
6 to 8). No new key (`region`, `role`, `households`, `agents`, `cause`, `reason`
exist). Rollback is reverting the PR.

**Acceptance:** FR-SPN-035, 036, 042, 046, 047 covered at unit level; coverage
holds.

## Phase 7: the exit gate

- On the whole module: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`,
  `go test -short -race ./...`, the full run with
  `-covermode=atomic -coverpkg=./...`, `go mod tidy`; the per-scope coverage
  read and recorded; `lizard -l go -C 10 -a 8 -L 100 -w` on the new files.
- The pinned hash of the all-off configuration did not move
  (`TestHashConfig_ReferencePinned`; `testdata/golden/off/` does not exist until
  the golden row, N-217).
- The traceability check of testing.md 4.6 by text search: every ID in a
  `Requirement:` line exists in SPN, and every requirement the Testing table lists
  as covered in full is cited.
- The roadmap row set to **done** with its commit, the index row and the notes
  updated with what the code settled (N-264 to N-268).
- Not gating, if time allows: a mutation pass over `internal/occupation`
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
| --- | --- | --- |
| `Config.World.ReviewTicks`, `EntryShare`, `RoleHysteresis`, `RoleCooldown`, `EntryRatio`, `ChangePrivation`, `RemovePrivation`, `DeathStep`, `ExitShare`, `Renewal`, `EntryPoolShare`, `Config.World.Spawner` (the section type is renamed `WorldParams` by plan 1-11 D140; `Config.World` does not change, so this row names the fields through it), `Role.Closed`, `Role.StartMoney`, `Role.TrainingTicks`, `Region.CapitalReturn`, `Role.ExpectedWorkTicks` | Stage 0, `config.go` | Built |
| `bounds.go` bounds of the eleven tunables, `ErrOutOfBounds`, `ErrOverflow`, the `ladder_order` rule, `entry_share_window`, `no_open_role` | Stage 0, `bounds.go`, `validate.go` | Built |
| `registry[T]` (`add`, `find`), `validID`, `ErrInvalidID`, `ErrDuplicateID`, `ErrNilExtension`, `ValidationError`, the pattern of `WithClearing` and `resolveClearing` | [plan-phase-1-3.md](plan-phase-1-3.md) D36 and Phase 4 (`option.go`, `errors.go`) | Built |
| `EntryLimited`, `WeightChanged`, `RoleChanged`, `AgentSpawned`, `AgentBankrupted`, `EstateOpened`, `Cause` (six constants), `ReasonUnknownAgent`, `Money`, `RoleID`, `RegionID`, `AgentID` | Stage 0, `event.go`, `value.go` | Built |
| `internal/checked` (`Mul`, `MulDiv`, `MulDivCeil`, `MulDivFloor`), `internal/ordering.Bucket` | Stage 0 | Built |
| `advanceRun`, `warnRun` (the run of a bounded `Warn`) | [plan-phase-1-6.md](plan-phase-1-6.md) (`welfare.go`), `observer.go` | Built |
| `welfare.TargetRange(targetRatio, spread) (lo, hi)` for `Retarget`'s range, and `roleSummary.Wage`, `HasWage` to fill `RoleView.Wage`, `HasWage` | plan 1-6 (`internal/welfare`, `welfare.go`) | Built; the view is built by the world |
| `(*roleRow).upkeepIn(RegionID) []itemRow`, `Calendar.TicksPerYear()` for the view's `Upkeep` and `TicksPerYear` | [plan-phase-1-2.md](plan-phase-1-2.md) | Built |
| `siteSpawner` (D9) and the keyed `*rand.Rand` for a consumer's `Decide` | Stage 0, `random.go`; the world row keys it | Built; the call is row 11's |
| The agent type and its state: `weight`, `role`, `account`, `Privation`, `RoleSince`, `Hunger`, `Unfed`, `External`, `Mortal`; the primitives that a **role change** (reset the job and residues with `role_changed`, resize the history to the new window, carry the target by `Retarget`, set the strategy, `RoleSince`) and a **weight change** (`reweigh` of the job, the history unchanged) are made of: `job.abandon`, `job.reweigh`, `setResidues` and `residueLen` (plan 1-5), `history.Resize` (plan 1-6), `reseedBeliefs` and `newAgent` (plan 1-7) | plans 1-5, 1-6 and 1-7 | Built or planned with these names; plan 1-7 provides **no** combined step, so row 11's executor composes them inline (D110); the slot is the agent's index in the world's agent table and the region is the account's, neither is an agent field (plan 1-7 D81, FR-AGT-002) |
| `(*estates).pool(l *ledger.Ledger, r RegionID) Money` (the view's `PoolMoney`, read by the world's view builder; `Budget` is `PoolMoney * EntryPoolShare / 10 000`) | `estates.go` row 8, [plan-phase-1-8.md](plan-phase-1-8.md) Provides | Planned; names matched with plan 1-8 |
| `(*estates).grant(l, r RegionID, to int, want Money, reason ledger.Reason) (Money, error)`: an entrant's money (`start_money` or `births`), never more than the pool, called by the executor with the whole-household amount that `occupation.Funded` allows | plan 1-8 | Planned; name matched with plan 1-8; the executor, not this row, calls it |
| `(*estates).acceptMoney(l, from int, r RegionID, amount Money, reason ledger.Reason) error`: **every weight reduction** sends `x - occupation.Remaining(x, w, w')` of the agent's money through it (reasons `starvation`, `split`, `merge`; `deaths` arrives with the population module, FR-LED-055, and is not in the ledger registry yet), so the pool minimum and bound have one home; and `(*estates).acceptGoods(l, from int, r RegionID, reason ledger.Reason, only func(commodity int) bool) (lots int64, err error)`: **every role change** sends the goods outside the new role's tradable set through it, and a reduction's goods share goes through `(*estates).acceptUnits(l, from int, r RegionID, commodity int, qty int64, reason ledger.Reason) (units int64, err error)` per commodity (`qty = x - Remaining(x, w, w')`; the units feed `WeightChanged.Units`); a removal uses `(*estates).receive` | plan 1-8 (D105, D106) | Planned; names matched with plan 1-8; called by the executor (row 11) |
| `GoodsListed` counts **lots** moved to the estate (`acceptGoods` and `receive` return the count): the value of `RoleChanged.GoodsListed` and `AgentBankrupted.GoodsListed` | Coordinator ruling 2026-10-17; plan 1-8 D101 | Ruled; the executor emits it, this row only names it |
| `newTarget(s *stream, targetRatio, spread)` and `newHistory(row *roleRow)` for an entrant's creation | plan 1-6 (`welfare.go`) | Built; called by the executor |
| The executor itself: the order of P7.1(d), the checks of SPN 4 through `occupation.Refuse`, the removal fan-in, the events and `Debug` lines, `NewWorld` calling `resolveSpawner`, the keyed random source, the view builder, the merge of the counts into `EntryLimited`, the checkpoint of `Privation`, `RoleSince`, the renewal carry and the runs | `world.go` row 11; `checkpoint.go` (Stage 2) | **ASSUMED** (coordinator Q1) |
| SPN 3.1, 3.2, 3.3, 3.1.3, 4, 5.3, 6.1 to 6.4, 7.1, 10.2, 13.1.2, 14, 20 edits; FR-SPN-058 to 060; architecture.md 2.6 and `AGENTS.md` (`internal/occupation/`, `spawner_report.go`); the notes N-264 to N-268 | This row, Phase 1 | Owed here |

### Provides

| What | For | Contract |
| --- | --- | --- |
| `Spawner` (`Decide(v *RegionView, rng *rand.Rand, d *Decisions)`), `DefaultSpawner`, `var _ Spawner = DefaultSpawner{}` | `world.go` (P7.1(c)), consumers | The contract of architecture.md 7.7 in the doc comment; pure decisions, no pointer kept |
| `RegionView{Tick int64; Region RegionID; TicksPerYear int64; PoolMoney, Budget Money; ReviewTicks, EntryShare, RoleHysteresis, RoleCooldown, EntryRatio, ChangePrivation, RemovePrivation, DeathStep, ExitShare, Renewal, EntryPoolShare int64; Roles []RoleView; Agents []RegionAgent}` with an unexported scratch and an unexported `observed bool` (set once by the world when an observer is supplied; false makes `Hysteresis` 0, D119) | `world.go` (builds it each region and tick) | One instance, reused; roles in canonical order, agents in ascending ID, no pseudo-agent; module fields appended later (D112) |
| `RoleView{Role RoleID; Closed bool; Households, Agents int64; Wage Money; HasWage bool; Upkeep Money; ExpectedWorkTicks int64; StartMoney Money; ToolPrice Money; TrainingTicks, CapitalReturn, RenewalCarry int64}` | `world.go` | `Wage`, `HasWage` from `roleSummary`; `Upkeep` is one household's effective upkeep per tick at the region's prices; `ToolPrice` 0 with no tool; `RenewalCarry` the world's carry |
| `RegionAgent{Agent AgentID; Role RoleID; Weight int64; External, Mortal bool; Money Money; Unfed, Hunger, Privation, RoleSince int64; Upkeep Money}` | `world.go`, `agent.go` | Per household `Upkeep`; `Privation` set at P6.1 by `household.NextPrivation` (plan 1-7) |
| `Decisions{List []Decision}` (and unexported counts, zeroed by an unexported `reset()` that the world calls with the `List` reset before each region's `Decide`, so a consumer's spawner also starts at 0), `Decision{Kind DecisionKind; Agent AgentID; Role RoleID; Households int64; Cause Cause}`, `DecisionKind` (`DecideRemove`, `DecideReduce`, `DecideSwitch`, `DecideEnter`) | `world.go` | `DecideApply` is appended by SKL (Stage 3, O10) |
| `WithSpawner(name string, s Spawner) Option`, `resolveSpawner(o *options, name string, path []string) (Spawner, error)`, `ErrUnknownSpawner` | `world.go` (`NewWorld`), consumers | Rules of WLD 4.1.1; the name is hashed through `World.Spawner`, the code is not |
| `refusalReason(r occupation.Refusal) Reason`; the constants `ReasonUnknownRole`, `ReasonExternalAgent`, `ReasonSameRole`, `ReasonBadHouseholds`, `ReasonUnsupportedCause` (`unsupported_cause`, with its `Debug` text; the refusal appended last), `ReasonUnfunded`, `ReasonNoViableRole` | `world.go` (counting `Refused` and the `Debug` line) | Exported, appended; module reasons appended with their module |
| `entryTally` (`newEntryTally(regions int)`, `reset()`, `addPlan(region RegionID, d *Decisions)`, `addExecuted(region RegionID, changed, entered, removed, unfunded, refused int64)`) with `limited(tick int64, region RegionID) (EntryLimited, bool)` (adds the plan's counts to the executor's), `extinctRun(before int64, agents int64, everHad, configured bool) (run int64, warn bool)`, `unfundedRun(before int64, due, funded int64) (run int64, warn bool)`, `belowFloorRun(before int64, weight, agents, minAgents int64) (run int64, warn bool)` (argument lists as the predicates `Extinct`, `Unfunded`, `BelowFloor` of Phase 3 take; fixed in Phase 6 and copied here in the same PR), `roleSignals(r *RoleView, ticksPerYear, entryRatio int64) (e, v int64, hasE, viable bool)` (in `spawner.go`, called by `Decide`), `maxPrivation(agents []RegionAgent, role RoleID) int64` | `world.go` (emission), `snapshot.go` | Emit nothing; saturating runs; omitted when all counts are 0 |
| `occupation.Refuse(kind Kind, f Facts) Refusal` with the first failing check in the order of SPN 4 | the executor, for each decision against live state | Pure; `Refusal` and `Facts` hold only the slice's six refusals, module ones are appended with their module. **Field lists** (written into this row by the PR that creates each type, Phases 2 and 3, before row 11 codes against them; minimum): `Facts{AgentKnown, RoleKnown, External, SameRole, CauseSupported bool; Households, Weight int64}` (`CauseSupported` is set by the executor: false only for a `DecideReduce` whose cause is neither `starved` nor `insolvent`; it gives `unsupported_cause`. Causes of `Remove`, `Switch` and `Enter` decisions are not validated in the slice: SPN 4 is silent, and a per-kind closed set would be a spec change, left to row 11 and the spec owner); `Leaver{Ref int; Privation, Weight, Unfed, Money, Upkeep int64}`; `Exit{Ref int; Households int64; Remove, Insolvent bool}`; `Host{Ref int; Weight int64}` (hosts in ascending ID order); `Kind` the decision kind as a small integer |
| `occupation.Allowance(households, shareBP int64) (int64, bool)`; `occupation.Exits(dst []Exit, ls []Leaver, allowance, deathStepBP int64) (out []Exit, cut int64)`; `occupation.Remaining(x, w, wNew int64) (int64, bool)` | `Decide` (`Allowance`, `Exits`: caps and shares are policy, SPN 4); the executor uses `Remaining` only (the scaling of money and goods to the estate: `floor(x * w' / w)`) | SPN 5.3, 4.1; the executor takes `x - Remaining(...)` to the estate and does not re-apply the allowance |
| `occupation.Renewal(carry, removed, renewalBP int64) (n, carryOut int64, ok bool)` | the executor (advances the carry with the households actually removed) | D118 |
| `occupation.Funded(n, budget, amount int64) int64` | the executor (the funded number of an `Enter`; the unfunded are not created) | `min(n, budget/amount)`, an amount of 0 is not limited |
| `occupation.Place(hosts []Host, scale int64) (ref int, room int64, ok bool)` | the executor (the host of an entrant: smallest weight below `scale`, ties to the lower ID, an away agent is MIG's, `Host` has no `Away` in the slice; `ok` false means a new agent of weight 1) | The caller loops, taking `min(remaining, room)` households per call |
| `occupation.Retarget(target, oldRole, newRole, lo, hi int64) (int64, bool)` | the executor (`world.go`, row 11), inline in its role-change composition | `clamp(newRole + (target - oldRole), lo, hi)`, no draw (SPN 4.2) |
| `occupation.PastCooldown`, `Rung3`, `ReviewDue`, `Expected`, `Viability`, `Beats`, `CapitalOK` | `Decide`, the snapshot, tests | Pure |
| The glossary terms of N-017 in `doc.go` | all later rows | Added before use |

## Security review

The module is a library with no network, file access, credentials or
persistence. This row adds no trust boundary of its own; the one untrusted
party is a consumer's `Spawner`.

- **Who can reach it.** A consumer registers a `Spawner` and the world calls it
  with a view; the view is a copy valid for the call (SPN 3.2), so the
  extension cannot reach live state. Its output is data that the executor checks
  decision by decision against live state (SPN 4); a refused decision changes
  nothing. Panics in the extension are not recovered (architecture.md 12.5).
- **Integer overflow.** Every product goes through `internal/checked`; a
  quotient beyond `int64` saturates by sign and a capital test whose `upkeep * ticksPerYear` leaves `int64` is denied; nothing wraps or panics (D115). Validation already bounds the
  tunables (`Households * 10 000` cannot reach `int64` at `MaxHouseholds`).
- **Resource exhaustion.** A plan the extension appends is processed once, in
  order; its length is the extension's, not the library's, and a consumer that
  appends millions of decisions only costs itself the time. The library sets no
  cap on the plan (open item O2). The default's scratch is bounded by the agents
  and roles of a region; the replacement households are admitted as batches in closed form, so they add nothing proportional to their number (O9).
- **Corrupt input.** `Place` and `Exits` take caller slices and index only into
  them; a `Ref` out of range is a programmer error of the caller. The executor
  validates every `AgentID` and `RoleID` of a consumer's decision (`unknown_agent`,
  `unknown_role`).
- **A strategy gaming the ladder.** The privation and the unfed count are set by
  the world from the ledger (SPN 5.1, AGT 5.4), not by a strategy, so a strategy
  cannot hide hunger or climb the ladder on purpose.
- **Not applicable:** authentication, authorisation, injection, secrets,
  cross-tenant exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer.
  `Spawner`, `RegionView`, `Decisions`, `DefaultSpawner` and `WithSpawner` are new
  exported identifiers of the root package, each with a doc comment (architecture.md
  6.5); they are a compatibility promise from the first release (6.6), so Phase 4
  keeps the surface to what SPN 3 states.
- **Document contracts.** Phase 1 adds text to SPN, `architecture.md` 2.6 (a fresh
  RULE-TAG) and `AGENTS.md`, renumbers nothing (testing.md 4.1; FR-SPN-058 to 060
  are new IDs), and `doc/notes.md` gets a fresh tag.
- **Append-only stays true.** Module fields of the view and decision, and the
  `Cause` and `Reason` constants of POP, SKL and MIG, are appended with their
  module (D112); with the module off they add nothing, so the pinned hash and
  `testdata/golden/off/` do not move (architecture.md 2.10.5, N-217).
- **What becomes hard to change, and when.** The decision order and the counts
  (D117) enter the golden files at the end of Stage 1 and the checkpoint (the
  carry, the runs) in Stage 2. Until the first golden file is recorded they may
  be corrected in a normal PR; after it, a change is a behaviour change with the
  explanation of testing.md 7.2.
- **Rollout** is one PR per phase (A6). A phase is rolled back by reverting its
  PR; later phases depend only on earlier ones and no other row has started to use
  this one.
- **Dependency failure.** None at run time. The schedule risk is the agent and
  estate rows: this row's code does not call them (D110), so it can be built and
  tested with hand-built views if they slip (the one exception is the FR-SPN-012
  trace of Phase 5, which needs plan 1-7's `household.NextPrivation`). Only row 11's executor waits for
  them.
- **The Go toolchain.** `go.mod` says `go 1.25`. Every sort ends in a unique key
  (architecture.md 7.6) and no map is iterated, so nothing depends on a
  version-specific order.

## Testing (summary, testing.md section 11)

The pyramid for this row is all base plus a small integration of the pure plan:
unit tests in `internal/occupation` and in the root's internal tests, property
tests over seeds, fuzz targets over the arithmetic, the admission and `Decide`,
and allocation tests. There is no end-to-end test because there is no world; the
world and golden rows add them, and the scenarios S06, S14, S26 and S27 are
theirs. `Requirement:` lines cite the IDs below; no requirement ID is in a test
name.

| Phase | Covered in full | Covered in part (completed in) |
| --- | --- | --- |
| 2 | FR-SPN-051 (`Remaining`) | FR-SPN-018 (`ReviewDue` here; the eligibility half, not external and the cooldown, is `Decide`'s in Phase 5), FR-SPN-008 (`Funded` here; the pool never overdrawn is the executor's), 014 (`Exits` here; the rung in Phase 5), 016 (the allowance and the off value here, the rung in Phase 5), 017 (`E` and `V` here; the no-signal rule in Phase 5), 024 (`Renewal` here; the carry across ticks in Phase 5) |
| 3 | FR-SPN-021 (`Admit`, with the per-household oracle comparison for the closed-form kind-2 batch) | FR-SPN-004, 010 (`Refuse`, the check and its reason; the refusal that changes nothing and is counted is the executor's), 022 (the shared counts here, the uncapped like-for-like in Phase 5), 042, 047 (the predicates here, the runs in Phase 6); FR-SPN-056, 057 are SET's |
| 4 | FR-SPN-001, 003 | FR-SPN-040 (the rule here, `resolveSpawner` returning `ErrUnknownSpawner`; `NewWorld` calls it in row 11), FR-SPN-010 (the constant here; the refusal is the executor's) |
| 5 | FR-SPN-012 (the trace of SPN 15.6 through `household.NextPrivation`, plan 1-7 Phase 2: **Phase 5 requires that function merged**, and so does the PR order; if plan 1-7 slips, Phase 5 tests the rung against hand-set `Privation` and the FR-SPN-012 trace waits for the root cross-check when `internal/household` lands), FR-SPN-013, 014, 019, 020, 023, 025, 029, 030, 031 (the slice order), 041, 050, 058, 059, 060 | FR-SPN-015 (the `Decide` side; `AgentBankrupted` is the executor's), 016, 021, 022, 024 (the cases in the plan); NFR-SPN-001, 002 |
| 6 | FR-SPN-035 (the value of `EntryLimited` and its omission at zero) | FR-SPN-036, 042, 047 (the predicates in Phase 3 and the runs here; the `Warn` is emitted by the world), 046 (the signals and the largest privation here; the snapshot row is the world's) |

Built in Stage 0 and not re-tested: FR-SPN-037 (field defaults and bounds),
FR-SPN-038 (`ladder_order`), the warnings `entry_share_window` and
`no_open_role` of FR-SPN-039. Deferred with their code, with the owner:
FR-SPN-009, 011 (SKL, Stage 3: `needs_weight_one` and `in_traineeship` are appended with `Refusal`/`Facts`, N-264, N-268), FR-SPN-002, 005, 006, 007, 027, 032, 033, 034, 043, 044, 045, 052 (the executor,
the events, the checkpoint: row 11 and Stage 2), FR-SPN-026 (row 11: the executor
takes the budget once per region per tick, after the module's deaths and before
the switches and any entry; this row gives `Funded` and the advisory `Budget` of
the view, and Phase 5's plan-order test of FR-SPN-031 shows that `Decide` takes no
budget), FR-SPN-028, 053 to 055 (POP,
MIG, SUB: Stage 2), FR-SPN-048, 049 (SKL: Stage 3), FR-SPN-056, 057 (SET: Stage 3),
NFR-SPN-003 (non-interference: row 11).

"Covered in full" means at the lowest level that can verify it (testing.md 2.1).
The second half of testing.md 2.3, the same behaviour through the public API, is
the world and golden rows': S06 and S26 for the cobweb, S14 and S27 for the
ladder and the famine, S73 for the first-come cap.

Boundaries and error paths: every `ok` false, every refusal, every no-signal case
and every cap has a case that breaks only its rule. Golden files: none yet. Fuzz:
the arithmetic, `Admit` and `Decide`. Each PR states its coverage and that its
tests were run against a broken version (testing.md 9.3).

## Observability (summary, observability-and-logging.md section 9)

No event, no `Warn`, no metric and no log key is declared: `EntryLimited`,
`WeightChanged`, `RoleChanged`, `AgentSpawned`, `AgentBankrupted` and the three
`Warn`s of OBS 7.3 (rows 6 to 8) exist, and the world row emits them (1, 2). The
decisions a strategy-like extension takes are data; the operator's view of why an
agent was removed, changed role or refused is the `cause` of the event and the
`Debug` lines of the executor, which this row's reason constants feed (3). The
loops of `Decide` log nothing and build no event, and allocate nothing with the
world's scratch (4). The silent dampers (the entry cap, the exit cap, the
cooldown, the hysteresis band, the viability gate, the funding limit) are counted
in `EntryLimited` (D117, D119; N-045), and the three bounded runs are produced in
Phase 6. No metric, unit or attribute changes (5, 6); neither `Error` nor `Info`
is used (7); no error is logged and returned (8). The tests of section 8 that
apply here are the allocation tests of Phases 2, 5 and 6; the rest need a world
(9).

## Open items

- **O1.** The values of `EntryShare`, `ChangePrivation`, `RemovePrivation`,
  `ExitShare` and `EntryRatio` are guesses from the models of SPN 16 (N-015,
  N-027); scenarios S06, S14, S26 and S27 set them and need the world and the
  golden row. Not this row's to close.
- **O2.** The library sets no cap on the length of a consumer's plan. If a case
  appears, a cap is a new bound in `bounds.go` and a refusal reason, decided by
  the owner (SPN open question 19 does not list it).
- **O3.** No CI exists (O3 of plan 0). The gate of Phase 7 and the timed fuzz run
  are done by hand, on one architecture.
- **O4 (FINAL: ruled by the coordinator, reversible).** D112 to D119 are final;
  the owner may reverse one later by a new ruling and a spec edit, and no phase
  waits for one. Reason: each is spec-silent or settles an ambiguity, is
  written into the owning spec in Phase 1 and is checked by the hash and the
  goldens only after Stage 1. D115 (rounding, the absence of a signal, the own role as `E_o` 0),
  D116 (once per tick, `RoleSince < tick`, no reduction at `Unfed` 0), D117 (the
  order and the meaning of the counts), D118 (the carry's owner) and D119 (the
  counts through the plan) are spec-silent or settle an ambiguity and are made in
  Phase 1; the own role as `E_o` 0 (an unrated own role earns nothing, so leaving a role requires evidence of a better one) and the conservative denial of `CapitalOK` are among them. The own role as `E_o` 0 may herd a failing region into the first
  role that gets a wage; S06 and S26 decide (N-267).
- **O5.** The Cause constants `population`, `births` and `traineeship` are
  appended by POP and SKL; the cause of an `Enter` by a birth and of a
  switch by a completed traineeship are those modules' (D112).
- **O6.** The cost of `Decide` is proportional to the agents of a region
  (NFR-SPN-002) and unmeasured; the world row measures it against
  `spec-scale.md` 6.
- **O7 (closed, coordinator ruling 2026-10-17).** The privation step has one implementation, `household.NextPrivation` of plan 1-7 (AGT 5.4.1 and AGT 22 home it there; `consume` calls it at P6.1). This plan drops `occupation.PrivationStep`; FR-SPN-012 and the trace of SPN 15.6 are tested through plan 1-7's function. N-249 is closed.

- **O8 (FINAL: ruled by the coordinator, reversible; reviewers are not held to the `EntryRatio` gate, because the spec text gives the gate to rung 1 and entrants only and the band and the cooldown are the reviewers' dampers; closed: option (a); Phase 1 adds the clarifying sentence
  to SPN 6.2 and FR-SPN-019).** The spec is silent on whether a **reviewer**
  (SPN 6.2) is held to the viability gate. SPN 6.2 and FR-SPN-019 give the destination as the open role
  with the highest `E` that satisfies the band and the capital test; the gate
  `V >= EntryRatio` is stated for rung 1 (SPN 6.4, FR-SPN-023) and for entrants
  (D36, SPN 7.2), and SPN 11 defines `EntryRatio` as the lowest `V` of a role "an
  entrant or a distressed agent may enter". The plan follows the spec text: no
  viability term for a reviewer. Options: (a) keep it, as the spec says;
  (b) add the gate to SPN 6.2 and FR-SPN-019 (a reviewer never moves to a role
  that cannot feed a household working every tick), which also changes the
  `Hysteresis` count of D117. Recommendation: (a) until S06 or S26 show reviewers
  herding into unviable roles; the band and the cooldown are then the dampers
  (N-267).

- **O9 (FINAL: ruled by the coordinator, reversible; option (a)).** The kind-2
  replacement households are admitted as a closed-form batch. Reason: the
  replacement candidates of one leaver role are identical weight-1 households
  with the same `E` and the same destination ordering (highest `E`, lower role
  ID), so admitting `n` of them one at a time is equivalent to taking
  `min(remaining destination caps, n)` per destination in order; the closed form
  therefore preserves FR-SPN-021's ordering exactly and keeps cost and scratch
  O(agents x roles), satisfying NFR-SPN-002 with no recorded deviation. The key
  stays total (`Seq` the leaver role index). Phase 3 tests it against a
  per-household oracle in the `_test.go` file (mandatory); Phase 3's kind-2 part
  and Phase 5 are not blocked.
- **O10 (FINAL: ruled by the coordinator 2026-10-17, reversible; option (a)).** `DecideApply` is not declared in the slice: the slice has no queue and no refusal that fits it. SKL appends it with the `Apply` emission and its refusals in Stage 3 (D112, N-264). Plan 1-11 does not list `DecideApply`, so it needs no edit.
- **O12 (FINAL: ruled by the coordinator 2026-10-17, reversible; option (a)).** The capital test is the SKL 2.5 inequality exactly as stated, `diff * ut >= price * CapitalReturn`, in 128 bits (`math/bits.Mul64`), a price of 0 passing first (D115, Phase 2 `CapitalOK`, FR-SPN-058 states the exact inequality).
- **O11 (FINAL: ruled by the coordinator 2026-10-17, reversible).** `RegionCap`, `RoleCap` and `ExitCap` are kept unconditionally: each is O(1) per candidate already computed by `Admit`/`Exits` (recorded in D119). The `Hysteresis` count needs an extra comparison, so the view carries an unexported `observed bool` that the world sets once (true when an observer is supplied) and the counter is built only then (observability rule 4.5).

## Review record

Plan-review (inline, one document, no user-facing surface so dimension 8 is N/A), pass 1, 2026-10-17. Blocking: (1) a product beyond `int64` made a role silently lose its signal, a silent limit with no counter (observability 3.2; N-045): fixed by saturation and by saturation in an `E` of two cases (D115, Phase 2; as first written it was two truncating `MulDiv` steps, since replaced); (2) the Testing table called FR-SPN-004, 016, 017, 022 and 024 "covered in full" where the executor or a later phase completes them: moved to "in part". Minor, fixed: new 128-bit comparison code replaced by `checked.MulDivCeil` (reuse); a vague shuffling property replaced by a pure-view test (SPN 3.1.1) and empty-region cases. Minor, left: `RegionAgent.Hunger` and `Upkeep` duplicate spec fields the default barely reads (the spec lists them); `DecideApply` is declared and unused (spec and ruling). Pass 2 (same day, over the revised text): one blocking-class item, the Phase 6 row of the Testing table also claimed "covered in full" for the `Warn` and the snapshot row, which the world emits and builds: moved to "in part". No other finding; the verdict is ready to execute. The plan-audit loop is the coordinator's.
