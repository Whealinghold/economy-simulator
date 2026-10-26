# 0010. A module has named hooks outside its file

- Status: Proposed
- Date: 2026-10-25
- Supersedes: none
- Narrows: 0007 decisions 1, 2 and 3

## Context

[ADR 0007](0007-a-module-is-one-file-that-owns-its-touch-points.md) says a module
appears outside its own file only in the catalogue files (decision 1), that
`tick.go` is a straight line of one-line position calls (decision 2) and that the
world has five fan-ins, each a fixed-order sequence of guarded calls (decision 3).
`doc/rules/architecture.md` 2.10.1, 2.10.2 and 2.10.4 repeat it.

Plan 2-5 (the first module file, `demand.go`) is the first code written against
that rule, and it finds places where a module has to be called from outside its
file and outside the catalogue files, which the rule does not name:

- the default strategy, which lives in `strategy_default.go`, `strategy_work.go`
  and `strategy_view.go`, has to call the module's allocation of wants (`consider`,
  `allocate`, `emit`), and a later module (plan 10, the Work and Trade seams) has
  the same need;
- the viability check reads the module's term of the reference programme
  (`viability.go`), the phase files hold the module's aggregate events
  (`p36Aggregates` in `phase_trade.go`), the loader maps the module's keys
  (`config/mapping.go`);
- an agent is created in three places and changes role in one, and the module's
  per-agent state has to be formed in each (`createAgent`, `spawnEntrant`, the add
  of `integration.go`, and `takeRole`);
- the role table is built in `role.go` from the core terms, and the module adds
  its items to the tradable set of a role through the fifth fan-in, which the rule
  says is "called once by the role-table constructor" without saying how
  `validate.go`'s `tradableSets` reaches it.

None of this is module logic in a foreign file: each hook is one guarded call into
a method that the module file defines. But the rule as written calls it a defect
(architecture.md 2.10.2, 13), and an accepted ADR protects the rule, so the
hooks have to be admitted by a decision and not by the plan.

## Decision

1. **A module may be called from outside its file at a named hook, and a hook is
   a guarded call into a method the module file defines.** The module's logic stays
   in the module file (ADR 0007 decision 1). The hooks are these and no others:
   the call sites of the fan-ins in `fanin.go`, and the calls of the two
   agent-lifecycle fan-ins of decision 3 from `createAgent`, `spawnEntrant`, the
   integration add and `takeRole`; guarded calls from the
   `DefaultStrategy` files (`strategy_default.go`, `strategy_work.go`,
   `strategy_view.go`) into methods defined in the module file, including the Work
   and Trade seams that later modules add; the term of the reference programme in
   `viability.go`; a named `phase_*.go` call such as `p36Aggregates`; the audit
   call; the construction calls and `firstHousehold`; `offStage` extensions; and
   the module keys of the loader in `config/mapping.go`. A hook holds no branch on
   the module's data and no arithmetic; a reviewer who finds either has found a
   defect.
2. **`tick.go` stays a straight line, and the sub-step of a position is a guarded
   call inside the position's method, not a position.** A position method may call
   a module's step under the module's pointer, as `p36Aggregates` is a fixed
   sequence of guarded calls (`spec-world.md` 6). The list of positions of
   `spec-world.md` 6 is unchanged by a sub-step. The position test of testing.md
   6.8 reads the guarded form `if w.x != nil { w.pNN() }`.
3. **The fan-ins are the five of ADR 0007, with one entry point each, plus the
   agent-lifecycle calls.** `fillAgentView`, `checkTradePlan`, `removeAgent`,
   `relocationRefusal` and `tradableItems` stay as decision 3 says, and
   `tradableItems` is called from `role.go` through `tradableSets` of `validate.go`,
   the one function that `newRoleTable`, `economics` and `warnings` already share,
   so the validator and the world cannot disagree. A module that keeps per-agent
   state forms it in two **agent-lifecycle fan-ins**, `agentJoined(a, region)`
   (called from `createAgent`, `spawnEntrant` and the integration add) and
   `agentRoleChanged(a, from)` (called from `takeRole`); `removeAgent` already
   carries the step of a removal. They are not counted among the five. The audit
   call and the construction calls are hooks, not fan-ins.
4. **The catalogue files stay append-only in module order** (architecture.md
   2.10.5). A hook that is a list of calls is appended in module order and a later
   module inserts nothing.

## Consequences

- `doc/rules/architecture.md` and `AGENTS.md` gain the text under "Proposed edit"
  below, with a fresh read-receipt tag on `architecture.md`. That edit is made when
  the owner accepts this ADR, in the same change that sets the status, because an
  accepted ADR (0007) protects those two files and a rule edit alone would conflict
  with it. Nothing else in the rules changes.
- Plan 2-5 PR B2 (Phase 6, the world side of `demand.go`) does not start until
  this ADR is accepted and the edit is applied. Its PR A (the edits owed and the
  solver) and PR B1 (the pure rules, the catalogue, the loader and the ledger
  reason) do not depend on it.
- The later plans that cite these hooks (plans 9, 10, 11, 13, 14, 16 and 17, as plan
  2-5 E1 lists them) drop any "sixth fan-in" wording in favour of "agent-lifecycle
  fan-in".
- Lizard may force `demand.go` to split. The conditional row of 2.6 and of the
  `AGENTS.md` layout is `demand_plan.go` and `demand_state.go` (DEM), used only if
  the split is needed.

## Alternatives considered

- **Keep the rule as written and put the module's code in the foreign files.**
  Rejected: that is the defect the rule names, and it scatters the module.
- **A `Module` interface with hook methods.** Rejected by ADR 0007 (a registry
  that hides the order); a guarded call in a fixed sequence keeps the order in
  the source.
- **Form the module's per-agent state lazily at the first `Trade` view.** The
  fallback if this ADR is refused (plan 2-5 open item O5): a one-byte "formed" flag
  per agent in the checkpoint, and `AgentIndex` read at that tick. It changes when
  the carry is formed (FR-DEM-013) and the layout of checkpoint tag 12, and keeps
  no agent-lifecycle call.

## Proposed edit to `doc/rules/architecture.md` and `AGENTS.md`

This is plan 2-5 edit E1, one consolidated edit for the stage. It is not applied:
the owner applies it with the acceptance of this ADR. It enumerates the cases later
plans cite.

1. **2.10.1 and 2.10.2 (the touch points and where a module appears).** After the
   list of touch points add: "A module may also be called at a named hook outside
   its file ([ADR 0010](../adr/0010-a-module-has-named-hooks-outside-its-file.md)):
   a guarded call into a method the module file defines. The hooks are: (a) a
   method added to a core concept file by a module (`effectiveAt` in `production.go`,
   plan 9); (b) a new argument on a fan-in and its callers (`kind viewKind` on
   `fillAgentView`, `workAgent`, `tradeAgent`, plan 13); (c) a guarded call inside
   `entryRegion` and the spawner refusal in `internal/occupation` and `phase_exit.go`
   (plans 11, 14); (d) the construction calls and `firstHousehold` (plan 16); (e)
   the sub-step of a position is a guarded call inside the position's method, not a
   position (plan 11); (f) a field added by a module to a core view or row type
   (`RegionAgent.OwnUse` in `spawner.go`, plan 10); and the hooks of decision 1 of
   ADR 0010: the call sites of the fan-ins, the calls of the agent-lifecycle fan-ins
   from `createAgent`, `spawnEntrant`, the integration add and `takeRole`, guarded calls from the `DefaultStrategy`
   files into methods defined in the module file (the Work and Trade seams of later
   modules, plan 10, included), the term in `viability.go`, named `phase_*.go` calls
   such as `p36Aggregates`, the audit call and the `offStage` extensions." In 2.10.2
   replace "A module's logic in any other file is a defect" with "A module's logic
   in any other file is a defect; a hook of 2.10.1 holds no logic."
2. **2.10.4 (the fan-ins).** Update every "the five fan-ins" sentence (the 2.6
   comment, 2.10.4, 2.10.6, and the `AGENTS.md` Layout) to name the five and the two
   agent-lifecycle fan-ins `agentJoined` and `agentRoleChanged` (ADR 0010 decision 3).
   State that the audit call and the construction calls are hooks, not fan-ins; that
   `tradableItems` is called from `role.go` via `tradableSets`; and that the other
   plans' "sixth fan-in" wording is theirs to drop.
3. **2.6 and `AGENTS.md` Layout.** Add the conditional row: `demand_plan.go` and
   `demand_state.go` (DEM), if Lizard forces the split of `demand.go`, in the
   continuation list, with the same files in both lists (AGENTS.md says the two
   change together).
4. **Fresh RULE-TAG.** Replace the tag of `architecture.md` with a new random value
   in the same change (`od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`).

## Open points

1. Whether the hooks of decision 1 should be a test as well as a rule: a Guard test
   that reads the foreign files for calls into the module files and fails on any
   call not in the list. Decide with the second module file.
2. Plan 2-5 open item O5: if the owner refuses this ADR, the fallback above applies
   and the checkpoint layout of tag 12 changes with it.

## Related

- `doc/rules/architecture.md` 2.6, 2.10.1 to 2.10.6, 13
- [ADR 0007](0007-a-module-is-one-file-that-owns-its-touch-points.md) decisions 1, 2, 3
- `doc/plans/plan-2-5.md` D300, D301, D307, E1, E11, O5
- `spec-demand.md` 20, `spec-world.md` 6, `spec-agents.md` 3.6
