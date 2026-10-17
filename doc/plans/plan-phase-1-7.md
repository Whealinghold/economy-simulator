# Plan: Stage 1, row 7: the agent, the strategy and the default strategy

**Status: plan written (2026-10-17), audit round 5 fixes applied (2026-10-17); coordinator rulings applied (O14 first-buyer rule final, Phase 6b folded into Phase 6, eight phases).** Sequences the seventh row of
Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`agent.go`, `strategy.go`
(`Strategy`, `AgentView`, `TradePlan`, `JobRequest`, the default strategy,
`Directed`) with `WithStrategy`, and the name check `ErrUnknownStrategy`", into
eight phases (1 to 8; the tier combination of `Trade` is ruled, O14). It implements [spec-agents.md](../specs/spec-agents.md) (AGT) for
the slice. It is built against `doc/rules/architecture.md` (the layout of 2.6,
the module-file rule of 2.10, the `internal/` boundary of 2.9, the fixed-point
and overflow rule of 3.5, the extension rules of 4 and 7.7),
`doc/rules/testing.md` and `doc/rules/observability-and-logging.md`. Where this
plan and the rules or the specs disagree, they win and this plan is corrected.

Rule tags read for this plan: architecture=905c0e578c4e662908638dd8,
observability=35827912d1e3141a76af3937, testing=b3559bc6d47470c221bc4642.
Notes tag (as read at the start of this plan, 2026-10-17, before any edit of this
row): 61bd3ab432f88e1b1af8c3a2; the file has been edited since (N-244 to N-248
are written, the consistency pass), so the tag is stale by design: Phase 1
re-reads the file in full and quotes the tag it finds. The notes file was read
through its header, its full index and the full text of every open note tagged
AGT or ALL; Phase 1 reads it in chunks again before it edits it (note N-103,
point 5 of the update of 2026-10-13: the tag at the end is not proof of a full
read).

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR,
the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md) (D3
and D5 above all), a ruling of the coordinator of this wave (marked
"coordinator"; reversible), or is flagged for the owner's review. Where a decision is still
owed, Phase 1 makes it in the document that owns it, before the code that
depends on it. Decision IDs D80 to D98 and the rulings JC1 to JC4 are this plan's (plan 1-6 has D61 to
D66, D69 and D70; the next row, 1-8, starts at D100); earlier plans' decisions are
cited by file and ID. Note IDs N-244 to N-247 are this plan's and are already written in
`doc/notes.md` (N-248 is the consistency pass's).

No PRD exists for this row. The requirements source is the roadmap row, the
note N-236 and AGT. The row is built after the ledger, goods, clearing, beliefs,
production and welfare rows ([plan-phase-1-1.md](plan-phase-1-1.md) to
[plan-phase-1-6.md](plan-phase-1-6.md)) and before the estates, spawner,
viability and world rows, which call it.

Spec codes: AGT agents, WLD world, PRO production, CLR clearing, BEL beliefs, WEL
welfare and wages, SPN spawner, EST estates, INT integration, CHK checkpoint, OBS
observability, CFG config, VIA viability, GDS goods, CAL calendar, STB stability,
SUB subsistence, DEM demand, SKL skill barriers, STO storage, TRN transport, MER
merchants. `FR-`/`NFR-` IDs are the requirement IDs of those specs.

## What Stage 0 and the earlier rows already give, and what is left

| Concern | Where it already is |
|---------|---------------------|
| The 12 AGT tunables (`CoverTicks`, `InputJobs`, `OutputCoverTicks`, `FavourFloor`, `ReserveTicks`, `SpendMaxShare`, `SpendHalfTicks`, `RestBand`, `ProduceHaltRatio`, `BasketTolerance`, `Upkeep`, `StartMoney`), `Role.Recipes`, `Upkeep`, `Trades`, `StartMoney`, `StartStock`, `Strategy`, their defaults, bounds, validation, the warnings `no_upkeep`, `commodity_untraded`, `upkeep_unsourced`, `start_stock_below_target`, `reserve_exceeds_money`, `strategy_default_unused_trades`, `ErrBasketMismatch`, `ErrOverflow` of the largest weight times an upkeep | `config.go`, `defaults.go`, `bounds.go`, `validate.go`, `errors.go` (Stage 0); the registry test is staged and has no AGT entry left |
| `Side` (`Bid`, `Ask`, `String`), `AgentSpawned`, `AgentBankrupted`, `UpkeepShortfall`, `ProductionHalted` (`HaltRow`), the reasons `not_on_role`, `zero_yield`, `no_site`, `no_tool`, `no_input`, `not_tradable`, `duplicate_order`, `consumption`, `disposal`, `initial`, `start_money` | `event.go` (Stage 0) |
| The sites `siteWork` (D3), `siteTrade` (D5), `siteOutcome` (D8), `siteCreation` (D10) and the stream `keyed`/`keyedPair`/`word`/`draw`/`drawRange` | `random.go` (Stage 0). This row adds no site |
| `registry[T]` (`add`, `find`), `validID`, `ErrInvalidID`, `ErrDuplicateID`, `ErrNilExtension`, `WithClearing`, `resolveClearing` returning a `*ValidationError` with the `ErrUnknown*` rule | `option.go`, `errors.go` (Stage 0 and [plan-phase-1-3.md](plan-phase-1-3.md) D36) |
| The role table (`roleRow`: `recipes`, `trades`, `upkeep`, `tradable`, `member`, `regionUpkeep`, `window`; `upkeepIn(region)`, `isTradable`, `tradableSomewhere`) and the recipe table (`recipeRow`, `makers`) | `role.go`, `recipe.go` ([plan-phase-1-2.md](plan-phase-1-2.md), done) |
| `Order`, `Trade`, `LotKey`, `Clearing`, `AverageClearing`, `marketReport`, `orderFacts` (with `Tradable`), the admission of `not_tradable` and `duplicate_order`, the book | `order.go`, `clearing.go` ([plan-phase-1-3.md](plan-phase-1-3.md), done) |
| `Belief`, `Bounds`, `beliefParams`, `beliefs.Limits`, `Clamp`, `Gate`, `Update`, `Draw`, `Reseed` (with `Seed`), `Window`, `Position` | `beliefs.go`, `internal/beliefs` ([plan-phase-1-4.md](plan-phase-1-4.md), done) |
| `job` (`recipe`, `remaining`, `executions`, `principal`, `yield`, `pending`), the residue layout (`residueLen`, `setResidues`), `admit` (the `not_on_role` check against the role's list, so a role with no recipes refuses every request), `productionScratch.maxExecutions`, `start`, the `yieldTable.effective` read, the cost-anchor table, `production.WearCost` | `production.go`, `internal/production` ([plan-phase-1-5.md](plan-phase-1-5.md), done) |
| `history` (`welfare.Ring`), `newHistory(row)`, `newTarget(stream, targetRatio, spread)`, `paramsOf`, `welfare.WorkShare`, `Executions`, `FloorBinds`, `HeldByFloor`, `TickCost` (the one place the cost of a tick multiplies by the weight), `Ring.Ratio`, `WorkingTicks` | `welfare.go`, `internal/welfare` ([plan-phase-1-6.md](plan-phase-1-6.md), done) |
| `internal/ledger` operations (`Open`, `Move`, `Transfer`, `Holding`, `Expiring`, `Boundary`, `Audit`) | [plan-phase-1-1.md](plan-phase-1-1.md), [plan-phase-1-5.md](plan-phase-1-5.md) Phase 2 |

What is left for this row is the agent and what is done to it, and the strategy:

- the **agent type** with the fields no module owns, its creation, its upkeep in
  total quantities, its consumption step and its hunger signals;
- the **`Strategy` extension point**: the interface, the plain-data views and
  plan, `DefaultStrategy`, `Directed`, `WithStrategy` and the name check;
- the **pure rules** of the default strategy in `internal/household`;
- the **root glue** the world calls: the core of the view filler, the intent to
  order conversion, the one report of the work decision.

## Scope

**In scope.**

- `internal/household` (stdlib and `internal/checked` only, scalar functions over
  primitives): `Fed`, the hunger and privation steps (5.4, 5.4.1), the upkeep
  cost and the reserve money (7.4.2), the targets (7.3), the tiers (7.3.4), the
  spend cap (7.4.4), the buy share (7.5.2), the affordable quantity (7.5.3,
  7.5.4), the output target and the halt predicate (7.1.5), the recipe value
  accumulator and the comparison of two recipes (7.1.4), the keep-the-food cap
  (7.1.6). `NFR-AGT-004` names the package.
- `agent.go` (root, unexported type): the `agent` type, `newAgent`, the
  creation draws in order, the funding of an initial agent, the effective upkeep
  in total quantities (FR-AGT-011), the consumption step of one agent
  (FR-AGT-013, 014, 015, 075), the belief reseed helper, the mailbox type
  `Directive`.
- The strategy files (root; the Lizard limits of N-243 are 500 lines a file, cyclomatic
  complexity 10 and 8 parameters, so the row is split from the start; the file names are
  indicative, A1): `strategy.go` (`Strategy`, `AgentView` and its groups, `OutcomeView`,
  `TradePlan`, `Intent`, `Discard`, `Purpose`, `JobRequest`, `Directed`),
  `strategy_default.go` (`DefaultStrategy`: `Work`, `Trade`, `Revise` and the small
  helpers `Trade` is split into: the needs, the tiers, the money allocation, the
  intents), `strategy_work.go` (the work report: the share, the floor, the halt and
  the rest as one pure function), `strategy_view.go` (`viewScratch`, the core view
  filler, the intent to order conversion), and the strategy resolution in
  `option.go`.
- `option.go`, `errors.go`: `WithStrategy`, `resolveStrategy`, `ErrUnknownStrategy`.
- `event.go`: the log-only reasons `rest`, `halted` and `no_recipe` (D87).
- The glossary terms these files use, in `doc.go`, before they use them
  (architecture.md 3.2): added at the start of Phase 2.
- The spec, note, rule-layout and roadmap edits these files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| The loops over agents, the rotation order, the call of `Work` and `Trade` at P2.5 and P3.2, the execution of discards and orders against the ledger, the building of `orderFacts`, the strategy table, `UpkeepShortfall`, `ProductionHalted`, `AgentSpawned` and the "households unfed" run and `Warn`, the Debug lines, the outcome loop of P5.4 and the `OutcomeView` filling | Roadmap Stage 1, `world.go` row (row 11). This row gives it the per-agent functions of the Cross-plan contract |
| The removal of an agent (the single fan-in, AGT 3.6 steps 1 to 6), the weight change, the role change, the split and merge. This row adds **no** `changeRole` or `reweighAgent`: its "per-agent functions" are create, consume, view and work report only, and row 11's executor composes the role-change and weight-change primitives from the other rows' pieces (coordinator ruling) | `world.go` (the fan-in and the spawner's executor, which composes the primitives inline: row 11) and `estates.go` (the estate account: row 8); the spawner row (9) gives only pure support. This row gives the primitives (`job.abandon` use, `reseedBeliefs`) and builds no combined role-change or weight-change step |
| A later agent funded from the pool (`start_money`, FR-AGT-007) | `estates.go` and `world.go` rows. `newAgent` takes no amount; `fundInitial` is for 3.5.1 only |
| The ID space and the pseudo-agents (FR-AGT-001's second half), FR-AGT-032 | `world.go` row (WLD 5.3), `estates.go` row |
| The module hosts of AGT 3.1 (`DeathCarry`, `OwnUse`, `OwnUseJobs`, `Journey`, `Member`, `Stores`, `Trainee`) and the view groups, plan fields and request fields of the modules (shipments, offers, takes, notes, credit, cover, labour, room, saving, facilities, endowments, rumours, transport, own-use, merchant, wants) | Each module's PR, appended (architecture.md 2.10.5); the per-agent state in the module's slices (D81, N-244). FR-AGT-051 (the storage half), 076, 078 to 085 are theirs |
| `Direct`, `AddAgent`, `AgentState`, the directive's timing (read in phase 1, discarded after the tick unless `Sticky`), `ErrNotDirected` | Roadmap Stage 2, `integration.go` row (INT). This row builds the `Directive` type and `Directed` |
| `Directed`'s `SaveState` and `LoadState`, the checkpoint section of the agent record (AGT 15), the directive in the checkpoint, the strategy-name table | Roadmap Stage 2, `checkpoint.go` row (`Stateful` does not exist yet; N-247) |
| The check that a run has no `insufficient_funds`, `insufficient_stock`, `overflow`, `holding_limit` or `receipt_limit` rejection from the default over the seeds of REF 8 (FR-AGT-044, the world half), the scenarios S04, S16 (N-043, N-067, N-087) | Roadmap Stage 1 exit (`world.go`, golden rows). This row proves the property on a many-state fixture against the real admission (Phase 7) |
| The weight-1 and cohort invariance of a whole run (FR-AGT-052 at world level) | The golden/invariant rows; this row proves it on the pure rules and the plan |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- `internal/household` is standard library and `internal/checked` only, never
  imports the root, and sees integers (architecture.md 2.9; AGT 22, which homes
  the pure rules there; `spec-viability.md` needs them without a world, AGT 19).
- The default strategy has no fields and no state (AGT 7, FR-AGT-031). Its
  tunables arrive in the view, so a consumer can call it with a view of its own
  making.
- A strategy reads its view and the world's `*rand.Rand` and returns plain data;
  it keeps neither (architecture.md 7.7; AGT 6.1). `Work` takes no word from the
  default; `Trade` takes one word per intent it considers, in canonical commodity
  order, from the stream `(siteTrade, tick, agent)`; `Revise` takes none, the
  outcome's word being the world's (WLD 7.3, D3, D5, D8; BEL 4.5).
- All money is `int64` minor units, ratios are basis points, every product that
  can overflow goes through `internal/checked` (architecture.md 3.5). A view the
  world builds from a validated configuration cannot overflow (`ErrOverflow`
  bounds the largest weight times an upkeep, a start quantity or a job input
  with the targets, FR-AGT-059).
- The registration options reuse `registry[T]`, `validID`, `ErrInvalidID`,
  `ErrDuplicateID`, `ErrNilExtension` and the `*ValidationError` of
  `resolveClearing` (WLD 4.1.1; [plan-phase-1-3.md](plan-phase-1-3.md) D36;
  coordinator: plan 1-9 does the same for the spawner, both by small edits).
- A module that is off contributes nothing and an appended field is zero while
  off (D5 of plan 0); this row appends nothing to the configuration, an event
  field or a snapshot field.
- Rules beat specs: architecture.md 2.10.1 puts a module's per-agent state in the
  module's slices, never as fields of the agent type, so the AGT 3.1 sketch of an
  agent that hosts module fields is read as a sketch (the sketch itself says the
  exported form is a matter for INT and the snapshot).

Decided by this plan, to be written into the owning spec in Phase 1 before code
(each ruled by the coordinator of this wave and reversible by the owner, with
the reason in the item):

- **D80.** `internal/household` is created, and Phase 1 adds it to the layout of
  architecture.md 2.6 and of `AGENTS.md` (N-228 left the choice to the agent
  row, AGT 22 homes the rules there, AGT 19 and NFR-AGT-004 need it; architecture.md
  2.1 asks for a purpose and a reason to exist independently, and both hold:
  the viability solver reads the targets without a world). Owner-confirmable;
  coordinator accepted. Its API is **scalar** (D89).
- **D81.** The agent type is the unexported `agent`, with only the fields no
  module owns: `id`, `account`, `role`, `weight`, `external`, `mortal`,
  `strategy` (an index), `beliefs`, `job`, `residues`, `history`, `target` (the
  target ratio drawn at creation, WEL 13), `unfed`, `hunger`, `privation`,
  `roleSince` and the `directive` mailbox. `DeathCarry`, `OwnUse`,
  `OwnUseJobs`, `Journey`, `Member`, `Stores` and `Trainee` are the slices of
  their modules, indexed by agent slot (architecture.md 2.10.1); the checkpoint
  still writes them in the agent record (N-128, N-244). The region is not a field
  (FR-AGT-002): it is the account's, read with `ledger.Account(id).Region`.
  Ruled by coordinator, reversible.
- **D82.** Both views carry an exported plain-data group of **only the belief
  parameters `beliefs.Params` reads** (coordinator, option (a)): the fifteen
  fields of BEL 11 that `beliefs.Params` holds (`Step`, `AdjustProb`,
  `Significance`, `EvidenceShare`, `StockLow`, `StockHigh`, `MaxAge`,
  `WidthInit`, `WidthMin`, `WidthMax`, `WidthStep`, `CostFloor`, `CostCeiling`,
  `RefBandLow`, `RefBandHigh`), not `MaxPrice` (a root bound that the inline
  conversion fills) and not `PriceWindow` (the capacity of a market's window).
  The world fills the group from `beliefParams(World)` of plan 1-4 (this row lands
  before plan 1-11 renames the section type `World` to `WorldParams`, D140 there:
  the code and tests of this row reach the section only through `beliefParams`
  and a variable, never the type name, so the rename is mechanical and touches no
  logic here), and the
  default converts it inline into `beliefs.Params` at the call (architecture.md
  2.9). Plan 1-4 exports only `Belief` and `Bounds` at the root, so nothing
  exported there is duplicated; a test asserts the group converted back equals
  `beliefParams` field by field, so the two cannot drift. **Why (b) and (c)
  fail.** (b), the world calling `beliefs.Update` itself with `Revise` a
  pass-through, breaks FR-AGT-050 ("`DefaultStrategy.Revise` MUST return the
  result of `beliefs.Update`") and the wrapper of BEL 10.5 (a consumer's
  strategy delegates to `DefaultStrategy.Revise`, which must then work). (c),
  fields on `DefaultStrategy`, breaks AGT 7 (a zero-value struct with no
  fields, FR-AGT-031) and would give the default state the checkpoint would have
  to save. The group is a view of tunables like the section 12 group.
- **D83.** `Directed` and its `Fallback` are in `strategy.go`; the `Directive`
  mailbox type is in `agent.go` (it is agent state); AGT 22's `directive.go` is
  dropped (it is not in the layout of architecture.md 2.6 and the roadmap row
  names `strategy.go`). Coordinator accepted.
- **D84.** "The welfare module" does not exist: welfare is core and has no
  switch (`Modules` has no such pointer). AGT 7.1.2 and FR-AGT-033 are reworded to
  "the ratio is absent" (no upkeep, so no ratio), the case 7.1.2 already treats:
  the share is 10 000 and the agent works whenever a recipe is eligible.
  FR-AGT-033 keeps its ID; FR-AGT-034 is reworded from "with the welfare module
  on" to "when the ratio is present". Coordinator accepted.
- **D85.** The split with the world row (coordinator accepted, (i) to (v)): this
  row provides the pure or per-agent functions listed in the Cross-plan
  contract, and the world keeps the loops, the rotation, the execution against
  the ledger and the emission. The roadmap row 7 cell is widened to say so in
  Phase 1.
- **D86.** The intents of the default are returned **in canonical commodity
  order**, one per commodity, whatever the tier: the draws are taken in that
  order (WLD 7.3 D5: "intents in canonical commodity order"), one order per
  commodity combines the tiers (7.3.3), and a combined intent has no single tier
  to sort by. AGT 7.5.1's sentence that the intents are "returned in the order of
  the tiers' priority, upkeep first" is read as "the money is allocated tier by
  tier" (7.4.3), which is what its own next clause says ("the budget is computed
  here, the order matters only for the world's check"). Phase 1 rewords 7.5.1.
  The submission order of the world is then canonical commodity order; a test
  shows no admission rejection depends on it (Phase 7). Ruled by coordinator, reversible (O1): determinism (architecture.md 7.5, 7.6).
- **D87.** The strategy is pure and logs nothing, so the world logs the Debug
  lines of AGT 14.3 and this row gives it their facts. The reason `over_budget`
  of the line "intent dropped" has no source (a default intent that cannot be
  afforded is never formed, so nothing exists to drop, and a consumer's strategy
  cannot say it), so Phase 1 removes it from 14.3 and 22, and no constant is
  declared. The constants this row declares in `event.go` are the log-only
  `ReasonRest`, `ReasonHalted` and `ReasonNoRecipe` (AGT 14.3, 14.4, 22; closed
  set of the line "work declined" together with the existing refusal reasons);
  they are values of the `reason` log key, not of the metric attribute
  `economysim.reason` (observability-and-logging.md 5.4.1 lists the metric sets).
  Ruled by coordinator, reversible (O2): nothing can produce it.
- **D88.** One pure **work report** serves the default `Work` and the world's
  counts: from an `AgentView` it returns the work share, whether the floor
  `MinWorkShare` changed the share, whether the agent rests, the recipes the halt
  skips (ascending `RecipeID`) and the reason a request would be declined with.
  `Work` reads it; the world's `ProductionHalted` rows, its floor count and its
  "work declined" Debug line read the same function, so a consumer's strategy
  cannot hide the halt or the floor (AGT 7.1.3, 7.1.5, FR-AGT-072) and the
  default and the count cannot drift. `JobRequest` and `Work` keep their shape.
- **D89.** `internal/household` is **scalar** where the tick calls it: a function
  takes the numbers of one commodity or one recipe, and a small accumulator
  takes the terms of a recipe one at a time, so the root loops over the view's
  slices and casts each number inline, with no slice conversion and no
  allocation in the tick (NFR-AGT-001; architecture.md 2.9 "conversion
  allocates"). Only `Fed` takes slices, once per agent, over the upkeep rows.
- **D90.** The wear of a tool in the value of a recipe (7.1.4) is
  `production.WearCost` of plan 1-5 (`ceil(WearPerUse * price / Durability)`),
  called by the root and passed to `internal/household` as a number: the formula
  has one implementation. `internal/household` does not import
  `internal/production`.
- **D91.** The view filler keeps its scratch in a `viewScratch` sized once at
  construction from the largest role (tradable set, recipes, upkeep) and owned
  by the world. A view a consumer builds by hand has no scratch: the default
  strategy needs none for `Work` (it reads the work report) and for `Trade` it
  falls back to allocating its small working arrays when the view carries none,
  which is allowed outside the world's hot path (NFR-AGT-001 binds the world's
  views). The default never writes to a view (AGT 6.4.1). The scratch is an unexported
  field of `AgentView` set by `viewScratch` (nil in a hand-built view), written into
  the AGT 6.4.1 shape in Phase 1 deliverable 2, since `Work(v, rng)` and
  `Trade(v, rng, p)` have no scratch parameter.
- **D92.** The name check follows `resolveClearing`: `resolveStrategy(o *options,
  name string, path []string) (Strategy, error)`; `"default"` is
  `DefaultStrategy{}`, `"directed"` is `Directed{Fallback: DefaultStrategy{}}`,
  any other is a registered one, else a `*ValidationError{Path, Rule:
  ErrUnknownStrategy, Value: name}`. `resolveStrategy` is built in Phase 6, with
  `DefaultStrategy` (it names the type; Phase 5 builds `WithStrategy`, the registry
  field, `ErrUnknownStrategy` and `isDirected`, which need no `DefaultStrategy`, and
  nothing calls `resolveStrategy` before the world row). `WithStrategy` registers through
  `o.strategies.add(name, s, s == nil, "default", "directed")`: an invalid name
  is `ErrInvalidID`, a name given twice or built in is `ErrDuplicateID`, a nil
  `s` is `ErrNilExtension`, in that order (WLD 4.1.1). A typed nil inside a
  non-nil interface is the consumer's and is not detected (the same as
  `WithClearing`). `isDirected(Strategy) bool` is a type assertion to `Directed`
  (a wrapper holding one in a field is not directed, AGT 6.6).
- **D93.** The draws of `Trade`: one `rng.Uint64()` per intent considered, taken
  in the first pass over the commodities in canonical order, before any
  allocation of money, so the count is a function of the view and not of the
  outcome of the allocation (AGT 7.5.3). An intent is "considered" when the
  commodity has a gap (a bid candidate) or a surplus or expiring units (an ask
  candidate); a commodity with neither takes none. **Pinned in Phase 1
  (deliverable 10d, AGT 7.3.5 and 7.5.3).** The candidate is decided by the view
  alone, never by affordability: a *bid candidate* is `Gap > 0`; an *ask
  candidate* is `Gap == 0` and (`Surplus > 0` or `Expiring - Reserve > 0`),
  excluding a tool the role does not make (not a candidate, no word). "No bid is placed" of 7.3.5
  means "not a bid candidate" (`Gap == 0`), so a commodity takes exactly one
  word whatever the money: a bid candidate whose limit cannot buy one unit spends
  its word (7.5.3: "the draw was spent") and forms no ask. The ask quantity is
  `max(Surplus, max(0, Expiring - Reserve))` (the oldest lots go to the upkeep
  first, LED 5.2.1; the expiring units are part of the stock, so the two are not
  added). The stream is keyed by the
  world before the call (`siteTrade`, tick, agent) and read through the
  `*rand.Rand`, whose `Uint64` is one source word (the world's own draw is
  `Mul64` of such a word, WLD 7.2.1). N-246 records what the world must do.
- **D94.** The consumption step takes the **account to eat from** as a parameter
  (the agent's own, or its trainer's for a trainee, AGT 5.7, SKL 4.2 at a later
  stage), and leaves the shortfall per upkeep row in a scratch the world reads
  for its Debug line "upkeep unmet". It sets `unfed`, `hunger` and `privation`
  in the agent. The ledger moves are `Move` to the `consumption` boundary, which
  takes the oldest lots first (LED 5.2.1), one per upkeep commodity in canonical
  order.
- **D95.** The two creation draws are taken in one function, in the order of
  AGT 3.5.2 and WLD 5.7.1: `drawCreation(s *stream, a *agent, tr, spread int64)`
  takes the target (plan 1-6 `newTarget`, which takes the first word whatever
  the spread) and then the residue word `s.draw(residueBound)` (`residueBound = 10_000`, a named rule constant with a doc comment citing FR-PRO-026, not an inline literal) copied to every residue
  (`setResidues`, FR-PRO-026), and takes nothing else. The caller keys `s` to the
  agent's creation stream (D1 at construction, D10 at P7.1(d), D12 at a call);
  the order is fixed in one place.
- **D96.** A view the world builds cannot overflow, so a violated bound in the
  world's path is an invariant panic with a message that says so (architecture.md
  8.1; `newRoleTable` does the same for the merged upkeep). A view a consumer
  builds can hold anything, so `DefaultStrategy` treats a failed checked
  operation as "nothing to order for this commodity" and never panics on view
  data (architecture.md 8.1, 13); the household functions return `ok` and the
  strategy reads it. Phase 3 fuzzes this.
- **D97.** The `Directive` copy. The view carries the pending directive **by
  value** with its slices copied into the view's scratch (`HasDirective bool` and
  `Directive Directive`), because AGT 6.4.2 says nothing in a view points into
  live state. A directive is rare (an external agent only), so the copy is not a
  hot path, and `Directed` reads only the view. `Directive.Job` is a `*JobRequest`:
  the copy copies the pointee into the view's scratch and repoints `Job`, so a
  strategy that writes the view (the `Guard:` test of FR-AGT-024) cannot change the
  agent's directive.

- **D98 (FINAL, O14 ruled by the coordinator).** One order has one limit, drawn once
  per intent; in the cross-tier combination (`combineTiers`) **only the first tier to
  buy may have the order's limit lowered by `Afford`**, and a later (lower-priority)
  tier that cannot pay one unit at the order's current limit is **dropped, never
  lowered**. Every tier buys at the order's limit; the first tier to buy is charged
  at the limit it bought at; the lowering leaves 0 money, so no later tier buys.
  `limit * qty` is then at most the sum charged, so the admission cannot refuse the
  order for funds (FR-AGT-044). The cap bounds the tier-3 quantity only and never
  lowers the limit. Reason: standard consumer theory under a budget constraint is
  lexicographic (priority) spending on necessities first (Engel-style: upkeep and
  job inputs before discretionary tiers); a lower-priority want must not drag down
  the limit price of a necessity and so reduce the household's ability to buy it;
  the reservation-price rule is the per-unit willingness to pay, and a later tier
  that cannot afford it is simply not bought. Rejected: the former D98 (the order's
  limit is the least `limitOut` over the tiers that bought), which in the
  money 250, limit 105, `T1` = 3 case would lower the order to limit 40. Ruled by
  coordinator, reversible (golden-affecting, spec-silent, so the sentence is written
  into AGT 7.5.3/7.5.4 in Phase 1, deliverable 10e).

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The names of unexported functions in this plan and in the Cross-plan
  contract are indicative of the capability and of the arguments the caller
  must hold; a row that needs one checks the code of this row once it is
  merged. The exported names (`Strategy`, `AgentView`, `OutcomeView`,
  `TradePlan`, `Intent`, `Discard`, `Purpose` and its constants, `JobRequest`,
  `DefaultStrategy`, `Directed`, `Directive`, `WithStrategy`, `ErrUnknownStrategy`)
  are the spec's and are the part to get right before a release (architecture.md
  6.6). The Go shape of the core groups of the views is written into AGT 6.4.1 in
  Phase 1 (the spec has a table of groups and no Go), so it is a document the
  review can challenge before code.
- **A2.** The core view groups are Identity, Money, State, Upkeep, Job, the
  per-commodity group, the per-recipe group, Tunables (section 12, D82 and the role's
  `MinWorkShare`) and
  Mailbox. Every other group of AGT 6.4.1 is a module's and is appended with its
  module (D3 of plan 0; architecture.md 2.10.5). `Moment` is the calendar type of
  plan 1-2.
- **A3.** The per-commodity group carries, for each commodity of the tradable set
  in canonical order: `Commodity`, `Stock`, `Expiring`, `Belief`, `Bounds`, `Ref`
  (the effective reference price), `Tool` (the commodity is a required tool of
  some recipe of the configuration, derived once in the role and recipe table;
  AGT 7.3.5 offers a tool only by a role that makes it and no other group of the
  view carries the fact; derived from the whole configuration), `UnitCost` (0 unknown; the filler zeroes it for a
  commodity the role has no recipe making, FR-AGT-048), the market report
  (`Price`, `Age`, `BestBid`, `BestAsk`, `HasBestBid`, `HasBestAsk`) and the
  price window (`Mean`, `Low`, `High`, `Position`, `HasWindow`). The per-recipe
  group carries, for each recipe of the role in ascending `RecipeID`: `Recipe`,
  `Duration`, `Inputs`, `Outputs` (with `CostShare` and `Discard`), `Tools` (with
  `WearPerUse` and `Durability`), `Effective` (the yield `E`), `MaxExecutions`
  and `Reason`. `Expiring` is `ledger.Expiring` read at the fill (AGT 7.3.5).
- **A4.** One PR per phase, in phase order, on a branch from `stage-1-6`. No phase
  is gated on the owner: the D98 cross-tier combination of `Trade` is ruled and
  Phase 6 ships it as final. Phase 7 follows Phase 6; Phase 8
  (the exit gate) follows Phase 7. Phase 2 and 3 may be written while Phase 1 is in review, but they implement spec text
  (D84, D86, D88, D93, D98), so neither is merged before Phase 1 is. Phase 4 needs plan
  1-2's role table (done), plan 1-5's `job` and plan 1-6's `history` (done).
  Phase 7 needs plan 1-3's `book.admit`, plan 1-5's `maxExecutions` (done) and Phase 6's
  `DefaultStrategy`.
  The row shares `doc.go`, `event.go`, `errors.go`, `option.go`, `doc/notes.md`,
  the roadmap, `architecture.md` and `AGENTS.md` with the other rows: each PR
  appends its lines by a small edit and re-reads the shared file just before
  editing it.
- **A5.** As plan 1-2 A8: the hand-computed cases run from an internal test
  (package `economysim`) with small configuration literals until `NewWorld`
  exists. A test that needs the packs `one` and `second` (the real role tables,
  the real admission) runs from one external test (package `economysim_test`)
  that loads them through `config/` and reaches the unexported functions through
  a test-only `export_test.go`: an internal test cannot import `config/`. No
  integration test through `NewWorld` is possible in this row.
- **A6.** A hunger or privation step, a fed count and a consumption step are the
  agent's (this row); the aggregate `UpkeepShortfall`, which sums them per
  (region, role), is the world's. The step is a function of the agent and the
  ledger and reads no random number.
- **A7.** `viewCtx`, the read-only references the world assembles for the core
  filler, is a struct of pointers to tables the world already holds (the
  canonical `Config`, `roleTable`, `recipeTable`, the ledger, `productionScratch`,
  `yieldTable`, `yieldShocks`) and of per-region slices indexed by commodity (the
  market reports, the windows, the cost anchors, the published prices). Its exact
  fields are the implementer's within that need; the world row builds the
  per-region slices from its market state (`market.go`, row 11).
- **A8.** `recipeRow` carries per tool only `wearPerUse`; the tool's `Durability`
  is read from `cfg.Commodities[id].Durability` (plan 1-2 reads configuration by
  ID, no wrapper), so the filler copies both into the view.
- **A9.** The reference price a view and a seed use is
  `cfg.Commodities[id].ReferencePrice`; the per-region override is Stage 2
  (plan 1-2 A10).

## Phase 1: settle the owed spec lines and the notes

**Why first:** D80 to D84, D86, D87, D88, D93, D98 and D82's group are rules and a layout the
code implements; a code change that invents them would put a result-affecting
choice in a place the configuration hash and the checkpoint cannot see
(architecture.md 9.5) and would change a public type after it is used. This phase
changes specs, rules and notes, so it starts by reading `doc/notes.md` in chunks
and quoting its tag, handles every open note tagged AGT and ALL and, for the
specs it edits (AGT, BEL and, only if the check of deliverable 3 finds an edit owed,
INT), the notes of those codes that it touches, and gives `doc/notes.md` a fresh tag (`AGENTS.md`). It also edits
`architecture.md` and `AGENTS.md`, which gives `architecture.md` a fresh
RULE-TAG.

Deliverables, each in the document that owns it:

1. **AGT 6.4.1 and 6.4.4, with a pointer in `spec-beliefs.md` 10.1** (D82). Add the
   group of belief parameters to the `AgentView` Tunables row and to the
   `OutcomeView` table (the fifteen fields that `beliefs.Params` holds, named as
   BEL 11 names them, plain data, filled by the world from the beliefs
   parameters, converted inline at the call), say in one sentence why the
   alternatives fail (FR-AGT-050, AGT 7), and in `spec-beliefs.md` 10.1 (the
   paragraph that lists what the `OutcomeView` carries) add "and the belief
   parameters". `spec-beliefs.md` 10.5 says the default function is exported only
   through the strategy; it says nothing that conflicts and is not edited.
2. **AGT 6.4.1** (A1, A2, A3). Write the Go shape of the core groups of the views
   under the table: the exported type names, the fields of A3 and the scratch
   rule (valid for the call only; the slices are scratch copies), the
   unexported scratch field (D91) and `MinWorkShare` in the State/Tunables shape (the
   work report needs it for `welfare.FloorBinds`, FR-AGT-071; it is a Role field owned
   by WEL, read here and not in section 12). The module
   groups stay in the table and are marked "appended with the module".
3. **AGT 3.1, 3.2 and 22** (D81, D83). Mark the struct of 3.1 as a sketch of
   the agent's state; say that the type is unexported, that the fields the
   sketch hosts for a module (`DeathCarry`, `OwnUse`, `OwnUseJobs`, `Journey`,
   `Member`, `Stores`, `Trainee`) are kept in the module's slices by agent slot
   (architecture.md 2.10.1), and that the checkpoint still writes them in the
   agent record (3.2's table is unchanged in its "saved by" column). In 22, drop
   `directive.go`, home `Directive` in `agent.go` and `Directed` in `strategy.go`,
   move `WorkShare` from the `internal/household` sketch to `internal/welfare` (built,
   share.go),
   name the strategy resolution in `option.go`, the strategy table in `world.go`
   and the per-agent functions of this row in `agent.go`. `spec-integration.md`:
   check 4.7.2 and the home table (line ~1822); edit only if it names the
   struct `Agent` as the exported form (the line names the query methods
   `Agent` and `Agents`, not the struct, so a check and no edit is expected).
4. **AGT 7.1.2, 7.1.3 and FR-AGT-033, 034** (D84). Replace "if the welfare module
   is off" by "if the ratio is absent (the agent has no upkeep)" and "with the
   welfare module on" by "when the ratio is present". FR-AGT-033 keeps its ID.
   The share of 10 000 when the ratio is absent is already the last sentence of
   7.1.2.
5. **AGT 7.5.1** (D86). Reword the clause on the order of the intents: canonical
   commodity order, one per commodity; the money is allocated tier by tier.
6. **AGT 14.3, 14.4 and 22** (D87). Remove `over_budget` from the line "intent
   dropped" and from the list of reason constants; say that the strategy is pure
   and the world logs; list `rest`, `halted` and `no_recipe` as the constants
   this row adds to `event.go`; say that `rest`, `halted`, `no_recipe` and the
   refusals of the work report are the values of "work declined".
7. **AGT 7.1.5 and 7.1.3** (D88). One sentence each: the world's counts and the
   default `Work` both read the work report (the pure function of the view);
   `JobRequest` is unchanged.
8. **AGT 20, FR-AGT-010** (N-236 (2)). State that the world's refusal of a
   request from a role with no recipes is the `not_on_role` of the production
   admission (FR-PRO-048, [plan-phase-1-5.md](plan-phase-1-5.md) `admit`), which
   a role with an empty list refuses for every recipe; the default requests none.
9. **`architecture.md` 2.6 and `AGENTS.md` layout** (D80; N-243). Add the root files this row
   adds (`agent.go`, `strategy.go`, `strategy_default.go`, `strategy_work.go`,
   `strategy_view.go`) and the `production_report.go` that N-243 owes (plan 1-5), if the
   layout block does not list them yet, and add `household/` to the
   line of the core-rule packages (`ledger/ clearing/ beliefs/ production/
   welfare/ steady/ graph/ curve/` and `household/`), with its purpose. Replace
   the RULE-TAG of `architecture.md` with a fresh random value
   (`od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`). Do it by one small Edit after
   re-reading the lines: N-235 holds the combined fan-in edit of 2.10.4 for the
   first Stage 2 module PR, and this edit is not that one (it touches 2.6 only).
10. **Roadmap and `doc/documentation_index.md`** (D85). Row 7 of Stage 1 and the
    index row of this plan already carry the text (the coordinator's pass wrote
    them). **Verify, edit only if wrong**: the Plan column "[1-7](plans/plan-phase-1-7.md),
    8 phases", the Status "plan audited", the Create cell naming what the row adds
    beyond the original text (`internal/household`, the agent's creation and
    consumption step, the core view filler and the work report, "the requirements
    N-236 names: FR-AGT-010, 011 and the want items of FR-AGT-003 with the demand
    module") and the index row. Any edit is a small Edit after a re-read;
    siblings edit the same files.
10a. **AGT 7.1.6, 7.1.7 and 7.1.5** (B1, N6). Pin the case of a food cap of 0 for
    the chosen recipe: the request is never `Executions` 0 (0 means "the most the
    agent can run", AGT 6.3, and `production.Count` reads 0 as no maximum, so a 0
    would start the whole job and eat the food, which FR-AGT-037 forbids). The
    ruling (coordinator): a recipe whose food cap is 0 is **ineligible** for the
    request and the next best eligible recipe is tried; `NoJob` with `no_recipe`
    only if none is left. Phase 1 rewords 7.1.6/7.1.7 so "at least 1 if `s > 0`
    and a recipe is eligible" counts only recipes whose cap is at least 1. In 7.1.5 state that an
    `OutputTarget` of 0 never halts a recipe (the literal inequality
    `Stock * 10 000 >= Ratio * 0` is always true, and `OutputCoverTicks` may be 0,
    `bounds.go`), the reading the code implements.
10b. **AGT 7.3 and the AGT 22 sketch** (B3, B5). The tool term of `Job_c`
    (`+ k*` for a tool of `r*`) is in 7.3 already; the plan's code and tests
    implement it (Phase 2, 6). `OutTgt_c = ceil(Make_c * OutputCoverTicks)` with
    `Make_c` the unrounded fraction `k* * q_o * E / (10 000 * Duration)`: **one
    ceiling over the whole product**, the same number as `OutputTarget` of 7.1.5
    (`ceil(w * q_o * E * OutputCoverTicks / (10 000 * Duration))` with `w = k*`),
    so the target and the halt cannot disagree for one commodity. Correct the AGT
    22 sketch's `Make int64 // truncated up` to say that `Need` carries the output
    target already formed, not a rounded `Make`.
10c. **AGT 9.1 and FR-AGT-046, 052** (B2, r2 B1, r3 B2). State invariance only
    where it is exact, each claim with its condition. A cohort of weight `w` holds
    `w` times the stock and money of a weight-1 household of the same role, with the
    same beliefs and the same drawn words. Then:
    - **Exactly `w` times** (the quantities are linear: sums, `max` and `min` of
      homogeneous terms): the wants `T1`, `T2`, `T3` before the buy share and the
      cap, `Gap`, `Surplus`, the asks (`Surplus`, `Expiring - Reserve`), the
      reserves, `Keep` and `UseTgt`, and a `Target` that comes from `UseTgt`.
    - **Closed form, not `w` times:** a `Target` that comes from `OutTgt` is
      `ceil(w * X)` with `X = q_o * E * OutputCoverTicks / (10 000 * Duration)`
      (one ceiling over the whole product, 10b), so `w * X <= OutTgt <= w *
      ceil(X)`, and it equals `w` times the weight-1 value only where `X` is an
      integer (a recipe of `Duration` 3 and output 1 with `OutputCoverTicks` 1
      gives 1 at `w` 1 and 3 at `w` 7, against `7 * 1`; the packs' `make_tool_a`,
      `make_tool_b` and `fab_board` are such recipes). The halt `w * s * 10 000 >=
      Ratio * ceil(w * X)` is not scale-invariant at the threshold, so the planned
      recipe `r*`, and with it `Job_c`, `Keep` and the asks, is the same at both
      weights only away from the halt threshold.
    - **Limits** are equal (the draw does not see the weight; a limit never scales)
      only while the first tier to buy is not lowered by `Afford`: a first buyer that cannot pay one unit at
      the limit lowers it to its money (a later tier is dropped, never lowered), and that can happen at weight 1 and not at
      weight 7 (money 50, limit 105: one unit at limit 50; money 350: three units at
      105).
    - **Allocated quantities** of tier 1 and tier 2 are `w` times the weight-1
      quantity only when both households fund the tier in full at the drawn limit
      (ample money, no lowering). When money binds the allocated quantity is
      `floor(w * m / L)`, which is not `w * floor(m / L)` (money 250, limit 105,
      wants 3: 2 units at weight 1; with `w` 7 and wants 21, 16 units against 14).
    - **Tier 3, the cap, the executions and `Fed`** are floors: `floor(w*X*b/10 000)`
      is not `w * floor(X*b/10 000)` (AGT 16.2 at `w = 7`: tier 3 of the staple is
      33 against 7 * 4 = 28; the cap is `floor(e*H/(e+H))` of totals, 21 740 against
      7 * 3 105 = 21 735), the executions are `ceil(w*s/10 000)`
      (`welfare.Executions`: weight 7, share 6 000 gives 5, below `7 * 1`), and `Fed
      = floor(w*h/u)` of holdings (identical whole holdings of 2 against an upkeep
      of 3: weight 1 gives `Fed` 0, weight 40 gives 26). The tier-3 quantity is
      bounded by the want rule `floor(w*T3*b/10 000)` and, **for the want-bound
      case only**, lies in `w * q1 <= q <= w * q1 + w - 1`; the spend cap, a
      cap-bound allocation and the executions are compared with their closed form
      on the totals (no `w * q1` band: with `SpendMaxShare` other than 5 000 the
      band fails, for example `e1 = 2, H1 = 3, SpendMaxShare = 9 000, w = 7` gives
      cap 0 and cap 7); the `Fed` fraction is equal only when `w*h/u` is exact,
      otherwise it is `floor(w*h/u)/w`.
    Reword 9.1 and FR-AGT-052 to these conditions, and say once that a test of
    weight invariance builds its states inside them (ample money, an output target
    whose `X` is an integer or asserted in the `ceil(w * X)` form, a stock away from
    the halt threshold).
10d. **AGT 6.4.1 and 7.3.5** (B4, D93). Add `Tool` to the per-commodity group of
    the table and say it is derived from the configuration. State the candidate
    rules of D93 (bid candidate `Gap > 0`; ask candidate `Gap == 0` and
    `Surplus > 0` or `Expiring - Reserve > 0`; "no bid placed" means `Gap == 0`;
    the ask is `max(Surplus, max(0, Expiring - Reserve))`; one word per candidate;
    a tool the role does not make is not a candidate and takes no word; `Tool`
    is derived from the whole configuration; this fixes the D5 stream layout
    before any golden exists).
10e. **AGT 7.5.3 and 7.5.4** (B7, D98 ruled). One order has one limit. State the rule:
    the limit `L_c` is drawn once; each tier buys at the order's limit; **only the
    first tier to buy may have the limit lowered**: a first buyer whose money
    cannot buy one unit at `L_c` lowers the limit to the money it has (`Afford`)
    and buys one unit; a later tier that cannot pay one unit at the order's current
    limit is dropped, never lowered. Every tier is charged at the limit it bought
    at. This is safe: the order reserves `limit * qty`, which is at most the sum
    of what the tiers were charged, so the admission cannot refuse it for funds
    (FR-AGT-044). After a lowering the money left is 0, so at most one tier of one
    commodity is ever lowered. The spend cap bounds the tier-3 quantity only and
    never lowers a limit (7.5.3 lowers a limit only to the money left); state this
    once in 7.4.4/7.5.4. Reason (one clause): spending is by priority, necessities
    first, and a lower-priority want must not lower the limit price of a necessity.

10f. **AGT 6.5.2** (O13). One sentence: a negative `Target`, `Reserve` or `UnitCost`
    from a strategy is treated as 0, as one above `MaxQuantity` is.
11. **Notes.** N-244 to N-247 **are already written** in `doc/notes.md` (with
    N-248, the consistency pass's): check their text against D81, D82, D93 and
    this phase and update them, do not add them again (N-246 changes with D93's
    candidate rules and the non-maker tool rule). The disposition of every open note tagged AGT or ALL
    is in the table below and is written into `doc/notes.md` in this change.
    Re-read the index at Phase 1 time for any note tagged AGT or ALL raised since.
    The notes of this plan are N-244 to N-247 (the range of this plan): N-244 (module per-agent
    state is in the module's slices, not in the agent type; the checkpoint still
    writes it in the agent record; areas AGT, CHK, SUB, MIG, POP, ORG, STO, SKL,
    CODE), N-245 (a tunable a module's default rule reads joins the view's
    Tunables group, as the belief parameters did; areas AGT, BEL, CODE), N-246
    (what the world row must do for the strategy streams and the draw count;
    areas AGT, WLD) and N-247 (`Directed.SaveState`/`LoadState`, the directive in
    the checkpoint and the strategy-name table wait for the checkpoint row; areas
    AGT, CHK, CODE).

**Notes disposition** (apply, reject or leave open, with what was learned):

| Note | Disposition |
|------|-------------|
| N-016 Keep the parameter registry true (ALL, CFG, WEL) | **Leave open.** This row adds no field and no default: every AGT field has its owner row since Stage 0 and no `notBuilt` entry is left for it. Learned: the belief-parameter group of D82 is a view, not a `Config` field, and adds no row |
| N-017 Words and names (ALL) | **Apply.** The terms this row's code uses go to the `doc.go` glossary before use (Phase 2): strategy, agent view, outcome view, trade plan, intent, purpose, job request, directive (mailbox; qualified: not PLC's), directed strategy, fed, unfed, hunger, privation, target (qualified: the inventory target; not the welfare target ratio), keep, reserve (qualified: the units for this tick's upkeep, and the reserve money), cover (qualified: ticks of use held; not INS's cover), tier, budget stack, spend cap, favourability, planned recipe, production halt, work share (exists from 1-6), work report. `Strategy` leaves `notYetTerms` of `glossary_test.go` in Phase 5. (`effective upkeep` is already in `doc.go`, line ~227; a second entry fails `TestGlossary_Vocabulary`, so check every term against `doc.go` before adding it, `agent`, `upkeep` and `job` included). The note stays open |
| N-027 No default macro constant is stable (ALL, REF, STY) | **Leave open.** Every default of AGT stays a labelled placeholder; no test of this row asserts a default as stable, and the worked examples of AGT 16 are used as hand-computed tests, not as results |
| N-103 Running several spec writers in parallel (ALL) | **Leave open.** Process note: this plan was written in parallel with five siblings and reads them only for the contract. Applied in the lesson: shared files by small edits after a re-read, and this file read in chunks |
| N-126 A new event or `Warn` adds a row (ALL) | **Apply (nothing owed).** This row declares no event and no `Warn`: `AgentSpawned`, `UpkeepShortfall` and `ProductionHalted` exist since Stage 0 and are emitted by the world row; the "households unfed" `Warn` is the world's. Three log-only reason constants are added (D87), which are not events |
| N-128 A hosted per-agent field is saved once, in the agent record (ALL, CHK) | **Apply in its lesson, leave open.** D81: the core fields are in the agent; the module fields are in module slices and the checkpoint writes them in the agent record all the same (N-244). Learned: the ring (`history`) and the job are the agent's hosted fields; their sections are 1-5's and 1-6's |
| N-163 The word "treasury" (MON, FIS, ALL) | **Reject for this row.** The word is not used; left open for its owners |
| N-217 D5 binds every module's change (ALL, CODE) | **Apply.** Nothing in this row appends to the configuration, an event or a snapshot field, so the pinned hash cannot move and `testdata/golden/off/` (not yet recorded) will not; the exit gate checks the pinned hash. `Strategy` is already a hashed name through `Role.Strategy` |
| N-235 One combined architecture.md edit for the fan-in and admission touch points (ALL) | **Leave open.** This row adds no fan-in and no admission touch point: the core filler and the intent conversion are called by the world's fan-ins (`fillAgentView`, `checkTradePlan`), and admission of `not_tradable` and `duplicate_order` is plan 1-3's. Learned: the edit of 2.6 (D80) is separate from the one N-235 holds |
| N-042 The evidence rule may offset the price (BEL, STB) | **Leave open.** Phase 1 edits `spec-beliefs.md` 10.1 only for the belief-parameter group; the evidence offset and scenario S01 (BEL 5.4) are not this row's. `Revise` calls `beliefs.Update` unchanged |
| N-230 The run of the "price beliefs at bound" `Warn` is advanced for every market every tick (BEL, WLD) | **Leave open.** The run is the world's (row 11 calls `BoundRun` for every market); `Revise` returns the belief and discards `Result` (D82) |
| N-043 Nothing anchors the price level if flows ignore price (AGT, PRO, SPN, BEL, STB) | **Leave open.** The flows are built here as rules (buffers, the halt, the recipe choice by margin); the scenarios S04 and S16 decide. Learned: the work report (D88) lets S04 and S16 run the floor and the halt with their counts |
| N-067 The production halt is not an anchor and has no counter (AGT, STB) | **Leave open for S16.** The count is the world's and is computed by the work report (D88); the threshold is a placeholder |
| N-087 Wants against the production halt (AGT, STB) | **Leave open.** No want exists in the slice; S24 is a Stage 2 run |
| N-101 Does the default Target count Inbound? (AGT) | **Leave open.** The view has no `Inbound` in the slice (transport is Stage 2); the default ignores it until decided (AGT 21 question 20) |
| N-111 A view takes draws (AGT) | **Leave open.** No informed role in the slice. Learned: the view of an informed agent is built before its `Trade`; D93 keeps the count of the default a function of the view |
| N-137 Room is committed at the decision (PRO, CLR, TRN, AGT) | **Leave open** for STO. The view has no room group in the slice |
| N-139 The default strategy discards, for upkeep only, when the module is on (AGT) | **Leave open** for STO. The default never discards in the slice (FR-AGT-051, first half; tested) |
| N-142 A job start and a dispatch leave the upkeep room free (PRO, TRN, AGT) | **Leave open** for STO |
| N-144 A busy employer cannot post vacancies (AGT, WLD, EMP) | **Leave open** for EMP. `JobRequest` has no `Offer` or `Hire` in the slice |
| N-153 Gates by settlement class: `Role.Needs` and `Role.Ceiling` (SPN, POP, AGT) | **Leave open** for SPN and POP. No view carries a class (AGT 6.4.1) |
| N-156 A title or a facility bought for a price (SAV, CRD, AGT, LND, CAP, INF) | **Leave open.** No `TitleBids` field in the slice |
| N-164 Credit requests and offers are plan fields (AGT, CRD, INT) | **Leave open** for CRD. `TradePlan` has `Intents` and `Discards` only; the plan type is built so that a field is appended (architecture.md 2.10.5) |
| N-188 Incidence is statutory; a duty is not in any limit price (AGT, FIS) | **Leave open.** No view field for a duty |
| N-196 A `Facilities` group in the `Trade` view (AGT, CAP) | **Leave open** for CAP and SAV |
| N-214 Keyed random streams (WLD, ..., AGT, CODE) | **Apply for AGT (the D3, D5, D10 and D12 rows) and leave open for the rest.** The keys are built (`siteWork`, `siteTrade`, `siteCreation`); this row names the draws: `Work` takes none, `Trade` one per intent considered (D93), the creation draws are the two of D95; N-246 says what the world keys. D12's site (the external agent's call) is INT's |
| N-228 A module's items in the tradable set have no touch point (AGT, DEM, SUB, CODE) | **Apply in part, leave open.** D80 settles the second point (the agent row needs `internal/household` and adds it to 2.6). The first point (the fifth fan-in for the tradable set) stays for N-235's edit; the want items of FR-AGT-003 join the role table with the demand module |
| N-232 The job and the residues live on the agent; what the agent row must call (PRO, AGT, CODE) | **Apply for point (2) only.** The residue word at creation is this row's (`drawCreation`, D95, which calls the existing `setResidues` and `residueLen`); point (3), the reset at a role change, is the executor's (row 11, which composes `reseedBeliefs`, `setResidues`, `history.Resize` and `job.abandon` inline with the existing `residueLen` and `setResidues`; this row builds no role-change helper); (1), (4) and (5) are the world row's, listed in the Cross-plan contract; the agent keeps the `job` by value and writes `Yield` with it (N-237) |
| N-236 Three halves of FR-AGT-003, 010 and 011 are the agent row's (AGT) | **Apply for (1) and (2); leave (3) open.** (1) FR-AGT-011: `upkeepTotals` multiplies each quantity by the weight, checked (Phase 4); the welfare cost multiplies by the weight in `welfare.TickCost` and nowhere else, and this row never calls it with a total (plan 1-6 A10: no row multiplies twice). (2) FR-AGT-010: the default requests none (Phase 6) and the world's refusal is the production `admit` (deliverable 8). (3) The want items are the demand module's PR |
| N-237 A job stores its effective factor; its start tick is derived (PRO, CHK, AGT) | **Apply for the agent row's part.** The agent holds `job` by value, `yield` included; the world row stores it at P2.7; the warm-start job and the checkpoint are others'. Left open for them |
| N-239 `AverageClearing` can produce a self-trade (CLR, AGT, WLD) | **Apply.** The default places at most one order per commodity (7.3.3: a bid if there is a gap, else an ask), so it cannot trade with itself; a test in Phase 6 asserts one order per commodity for every state of a table and a fuzz asserts it. Left open for the pseudo-agents and for the modules |
| N-244 to N-247 (AGT, this plan's) | **Written already** (see deliverable 11): update the text against D81, D82, D93 and the candidate rules; do not add again. They stay open until Phase 8 |
| N-248 The spawner's per-agent group is `RegionAgent` (SPN, AGT, CODE) | **Apply for this row.** `AgentView` is this row's exported type; before declaring any exported name Phase 5 checks it against the root's names and against plan 1-9's `RegionAgent` |
| N-249 The privation step is planned twice (AGT, SPN, CODE) | **Closed** (coordinator ruling): `household.NextPrivation` is the single implementation (Provides). Recorded here; no action |
| N-266 What the world row calls for the spawner (SPN, WLD, AGT, EST) | **Leave open.** Row 11 calls `consume` and `household.NextPrivation` through it; this row builds no combined step. Learned: `consume` sets `privation` and the spawner's view reads it |
| N-280 One function gives the virtual start offsets (VIA, WLD, AGT) | **Leave open.** Not this row's: the world's job construction calls it |

## Phase 2: `internal/household`, consumption, targets, tiers and the cap

**Why now:** the arithmetic is pure and the rest of the row (the agent's
consumption step, the default `Trade`) is a thin caller of it. It does not
depend on any other root file, so it is built and fuzzed first, against the
worked examples of AGT 16 and the requirement IDs. `NFR-AGT-004` names it.

Deliverables:

1. **`doc.go` glossary** (N-017): the terms of the disposition table above that
   this phase's code and doc comments use, added before use; the others when
   their phase starts. **Not `strategy`** (nor any term that is a `notYetTerms`
   entry of `glossary_test.go`, today `Strategy`): `TestGlossary_Vocabulary` fails
   when a glossary term is still in `notYetTerms`, so "strategy" is added in
   Phase 5 in the same change that removes `Strategy` from `notYetTerms`. Phase 5
   also adds `agent.go` and `strategy.go` to the file list of
   `TestGlossary_DeclaredTypesNamed` (and `strategy.go` to the `rootTypes` list of
   `TestGlossary_Vocabulary`), or the new exported types are unchecked.
2. **`internal/household`** (D80, D89): standard library and `internal/checked`,
   no map, closure, float or package variable (architecture.md 7.1, 7.5), no
   role, region, commodity or agent identifier (9.1), exported identifiers with
   doc comments (the package is internal, but the style is the module's).
   - `Params`: the eight numbers the rules read (`CoverTicks`, `InputJobs`,
     `OutputCoverTicks`, `FavourFloor`, `ReserveTicks`, `SpendMaxShare`,
     `SpendHalfTicks`, `ProduceHaltRatio`), plain `int64`.
   - `Fed(consumed, perHousehold []int64, weight int64) int64`: the least over the
     upkeep rows of `floor(consumed_c / u_c)`, capped at the weight; 0 rows give the
     weight (FR-AGT-014); a per-household quantity of 0 is an invariant
     (validation forbids it); no rounding up in favour of the household
     (5.4, example 16.3).
   - `NextHunger(before, unfed int64) int64` and `NextPrivation(before, unfed int64)
     int64` (5.4.1: `Hunger + 1` or 0; `Privation + 1` or `max(0, Privation - 1)`);
     they saturate rather than wrap, with the counters of plan 1-4/1-6 in mind
     (a run counter cannot reach `MaxInt64` in a run, so the saturation is a
     guard and not a rule; stated in the doc comment).
   - `Cost`, an accumulator with `Add(total, centre int64)` (adds `U_c * Centre_c` with
     `checked.Mul` and `checked.Add`, sticky `ok`) and `Value() (int64, bool)` (the
     `UpkeepCost` of 7.4.2); `ReserveMoney(upkeepCost, reserveTicks int64) (int64,
     bool)`.
   - `Need` and `Targets`, `ComputeTargets(n Need, p Params) (Targets, bool)`: for one
     commodity, from the totals `Upkeep` (`U_c`), `Job` (`Job_c`, already 0 unless
     `StartsIn` is 1), `OutTgt` (the output target, already formed by the caller
     with `OutputTarget` below, **one ceiling over the whole product** `ceil(k* *
     q_o * E * OutputCoverTicks / (10 000 * Duration))`, the same number the halt
     of 7.1.5 uses; `Need` carries no rounded `Make`, B5), `Want` (0 in the
     slice): `Reserve = U`, `UseTgt = U * CoverTicks + Job`, `Target = UseTgt +
     Want` if either is above 0 else `OutTgt`, `Keep = Reserve + UseTgt + Want`
     (7.3). `Job` is formed by the caller from the recipe, which the function does
     not know: `Job_c = InputJobs * k* * q_c` for an input of `r*` **plus `k*` for
     a tool of `r*`** (7.3; a commodity that is both an input and a tool takes both
     terms), and 0 unless `StartsIn` is 1.
   - `Tiers(keep, stock, reserve, jobwant int64) (gap, surplus, t1, t2, t3 int64)`:
     `Gap = max(0, Keep - Stock)`, `Surplus = max(0, Stock - Keep)` (an ask only if
     no bid), `T1 = max(0, Reserve - Stock)`, `T2 = min(Gap - T1, max(0, Jobwant -
     max(0, Stock - Reserve)))`, `T3 = Gap - T1 - T2` (7.3.3, 7.3.4); the three
     add to the gap by construction (FR-AGT-040, example 16.2).
   - `SpendCap(e, upkeepCost int64, p Params) (int64, bool)`: `e * H / (e + H)` in
     128 bits with `H = SpendHalfTicks * upkeepCost`, times `SpendMaxShare /
     10 000`; `upkeepCost` 0 gives `SpendMaxShare * e / 10 000`; 0 at `e` 0, never
     decreasing, never above `e` (7.4.4, FR-AGT-042).
   - `Excess(avail, savingTarget int64) int64` (`max(0, avail - savingTarget)`).
   - `BuyShare(pos int64, p Params) int64`: `FavourFloor + (10 000 - FavourFloor)
     * (10 000 - pos) / 10 000`, with `pos` the position of 7.5.2 (`EmptyPos`,
     5 000 for an empty window, the caller passes it) (7.5.2, FR-AGT-043); applied
     by the caller to tier 3 only. The two literals of the package, `EmptyPos =
     5 000` and `Basis = 10 000`, are named constants with doc comments citing
     AGT 7.5.2 and architecture.md 9.5/9.6 (a result-affecting literal is a
     named rule constant, not an inline number).
   - `Afford(avail, limit, want int64) (limitOut, qty int64)`: the quantity **one
     tier** buys at a limit with the money left, the limit lowered to the money
     when it exceeds it for one unit, 0 when not even one unit can be bought
     (7.5.3 Affordability, 7.5.4); a pure function of its arguments; the draw is
     the caller's. **The caller combines the tiers** (`combineTiers`, Phase 6,
     D98 first-buyer rule): one order has one limit; the tiers run at the order's
     current limit; only the first tier to buy may have the limit lowered (by
     `Afford`, to the money it has, leaving 0 money); a later tier that cannot pay
     one unit at the order's current limit is dropped, never lowered. Each tier
     is charged at the limit it bought at; the cap bounds the tier-3 quantity before
     `Afford` and never lowers the limit. `Afford` itself is per tier, knows
     nothing of the order and only lowers the limit (qty 0 only with no money or
     no want); the drop of a later tier belongs in `combineTiers`.
3. **Benchmarks** (NFR-AGT-003) with `b.ReportAllocs()` for `ComputeTargets`,
   `Tiers`, `SpendCap`, `Fed`; **a fuzz test** (`FuzzTiersAndCap`, NFR-AGT-004)
   with `f.Add` seeds from the worked examples: no input within the bounds
   panics or overflows, the three tiers add to the gap, the cap is 0 at 0
   excess, monotone in the excess and never above it, `Afford` never spends more
   than `avail`. Corpora committed in `internal/household/testdata/fuzz/`.

**Tests (unit; the whole phase is the base of the pyramid):** table-driven, one
function per rule, parallel, with the arithmetic of the expected value in a
comment (testing.md 5.5): the worked examples of AGT 16.1 (keep
12, 4, 2 for `staple`, `fuel`, `ore`, from use targets 9, 3, 2; item: output target 1,
keep 0; tiers 0/0/8, 1/0/3, 0/2/0), 16.2 (`e = 4 505`,
`H = 10 000`, `4 505 * 10 000 / 14 505 = 3 105`, `cap = 1 552`, `buyshare = 6 000`,
the allocation 105 then 1 240 with 207 left) and 16.3 (`Fed = min(33, 40) = 33`,
`Unfed = 7`); the boundaries (stock above keep, stock equal to keep, `Reserve` above
stock, `Job` above the left stock, `e` 0, `upkeepCost` 0, the largest values of
the bounds); `ok` false at overflow; a property test that
`Tiers` adds to the gap over a grid; a property test that `Tiers` and `ComputeTargets` are `w` times
their weight-1 values for the totals, given an already-formed `OutTgt` (FR-AGT-052,
FR-AGT-046: the rules compare totals to totals); the tier-1 quantity
is the same at every price (FR-AGT-069: `Tiers` takes no price). The property
holds for `Tiers` (the wants, `Gap`, `Surplus`) and for `ComputeTargets` because both are
linear in their inputs; it is **not** asserted for `OutputTarget` (the form
`ceil(w * X)`, tested separately against that closed form), for the floors of `BuyShare`
and `SpendCap`, for allocated quantities when money binds, or for `Afford` (deliverable
10c of Phase 1: the conditions are written there; tier 3 and the cap are compared with
the closed form on the totals, with the band `w * q1 <= q <= w * q1 + w - 1` for the
want-bound tier 3 only). Hand cases
added: a recipe with a tool (`Job = InputJobs * k* * q + k*` for a commodity that
is both input and tool, `k*` alone for a tool only, 16-series style hand
computation); `OutputCoverTicks` 10 with a fractional `Make` (`k* * q_o * E /
(10 000 * Duration) = 0.5` gives `OutTgt = 5`, not `ceil(0.5) * 10 = 10`: one
ceiling).

- `NextHunger` and `NextPrivation` (FR-AGT-075): a table of fed and unfed sequences from 0, saturation at the type's maximum, the fed step from 0 staying at 0, and the invariant `Hunger <= Privation` over a generated sequence of fed/unfed ticks; both allocate nothing.

**FR-/NFR- IDs:** FR-AGT-014, 038, 040, 041 (the budget half: `Afford` and the
cap), 042, 043, 046, 052, 069, 075 (the hunger/privation functions);
NFR-AGT-001 (no allocation: `testing.AllocsPerRun` over every function),
NFR-AGT-003, NFR-AGT-004.

**Observability:** none, because the package is pure arithmetic: no log, no
event, no random number. The allocation test is the non-interference evidence.

**Acceptance:** the examples of 16.1 to 16.3 reproduce; the fuzz ran 60 s with no
crasher; the package's coverage is at least 90% on its own tests; `go vet`,
`staticcheck` and `gofmt -l .` are clean; `testdata/golden/` and the pinned hash
are untouched.

## Phase 3: `internal/household`, the work-side rules

**Why now:** the recipe choice, the halt and the keep-the-food cap are the rest
of the pure rules; they need no root type and are the largest source of
branches, so they are built and fuzzed before the strategy that calls them.

Deliverables:

1. **The recipe value accumulator** (7.1.4, D89, D90): `type Net struct` with
   methods `Output(q, effective, centre int64)` (adds `floor(q * E * centre /
   10 000)` of a non-discard output, 128-bit), `Input(q, centre int64)` (adds
   `q * centre` to the cost), `Cost(amount int64)` (adds an amount already
   formed, the wear of a tool from `production.WearCost`, and the rent of the
   tenure module later) and `Value() (net int64, ok bool)`; sticky `ok` false at
   the first overflow. No slice, no allocation.
2. **`Better(netA, durA, netB, durB int64) bool`** (7.1.4): `netA / durA > netB /
   durB` by 128-bit cross-multiplication with the sign handled, a tie false (the
   caller keeps the lower `RecipeID` by iterating ascending and replacing only on
   a strict improvement); `Eligible(net int64, source bool) bool` (`net > 0` or a
   source, FR-AGT-035).
3. **`OutputTarget(weight, q, effective, outputCoverTicks, duration int64) (int64,
   bool)`** and **`Halted(stock, outputTarget, haltRatio int64) bool`** (7.1.5):
   `ceil(w * q * E * OutputCoverTicks / (10 000 * Duration))` and `Stock * 10 000
   >= ProduceHaltRatio * OutputTarget` in 128 bits; an `OutputTarget` of 0 never
   halts (a stock of 0 against a target of 0 would otherwise halt: stated in the
   doc comment and tested, and written into AGT 7.1.5 in Phase 1 deliverable 10a
   because the literal inequality would always halt; the rule is FR-AGT-036 "at or above `ProduceHaltRatio`
   of its output target" and a recipe whose output target is 0 has no target);
   the caller applies it to outputs with a positive `CostShare` only (a
   byproduct never halts, FR-AGT-036). The doc comment also says the result can exceed `MaxQuantity` for a valid configuration (validation bounds the keep, not the output target); the intent's `normalise` then cuts the `Target` to 0 (Phase 7).
4. **`KeepFoodCap(stock, upkeepTotal, q int64) int64`** (7.1.6, FR-AGT-037):
   `floor((stock - U) / q)` and 0 if `stock <= U`.
5. **`PlannedExecutions`: `Request(executions, kcap int64) (n int64, ok bool)`**:
   `min(executions, kcap)`, the caller passing `welfare.Executions(weight, share)` (one
   implementation of `ceil(w*s/10 000)`, welfare/share.go; `household` imports only
   `internal/checked`). This function keeps the "at least 1 if the share is above
   0 and a recipe is eligible" rule in one place, read in 7.1.7: a request above
   what can run is cut. **It never returns a request of 0**: `JobRequest.Executions`
   0 means "the most the agent can run" (AGT 6.3) and `production.Count` reads 0 as
   no maximum, so a 0 from a food cap of 0 would start the whole job and eat the
   food (FR-AGT-037, B1). When `kcap <= 0` or `executions <= 0` it returns `ok`
   false, and the caller does not form a request for that recipe. What the caller
   does then is ruled: a recipe with `ok` false is ineligible, the next best
   eligible recipe is tried, and `NoJob` with `no_recipe` only if none is left
   (Phase 1 deliverable 10a rewords 7.1.6/7.1.7 to match).
6. **Fuzz** `FuzzWorkRules` (NFR-AGT-004): no panic or overflow within the bounds,
   `Better` is a strict weak order on the grid (antisymmetric, transitive on
   samples), `Halted` is monotone in the stock.

**Tests (unit):** the worked case of 16.1 (`net = 1 * 10 000 * 900 / 10 000 - 2 *
200 = 500`); `Better` on fractions with equal ratios (tie false), a negative net,
a duration of 1 and the largest values (128-bit); `Eligible` for a source with a
negative net (eligible) and a recipe with inputs and a negative net (not); the
halt at the exact boundary (`Stock * 10 000 == Ratio * Target` halts), a byproduct
(the caller's gate, tested at the root in Phase 6); `OutputTarget` rounding up and
the overflow `ok`; `KeepFoodCap` at `stock == U` (0) and `stock == U + q` (1); `Request` with
`kcap` 0 and with `executions` 0 gives `ok` false, never `(0, true)`
(FR-AGT-037); `OutputTarget` of 0 never halts, tested with `OutputCoverTicks` 0
here and at the root in Phase 6; the property that the work-side rules ignore the
weight except through the totals.

**FR-/NFR- IDs:** FR-AGT-035, 036, 037, 070 (the halt bounds the unsold stock: the
function half; the run is the golden's), 034 (the executions composition),
NFR-AGT-001, 003, 004.

**Observability:** none (pure arithmetic).

**Acceptance:** as Phase 2; the whole package at least 90% covered by its own
tests; the 60 s fuzz clean; a `Guard:` test of
`internal/household` parses its non-test files and fails on any import beyond the
standard library and `internal/checked` (the import guard test bans only third-party
paths and the packages above `internal/`, so it does not enforce this).

## Phase 4: `agent.go`, the agent and its consumption

**Why now:** the strategy and the world row both need the agent's state, its
creation, its upkeep in totals and the consumption step; this phase is also N-236
(1) and the consumption half of AGT 5. It depends on `role.go`, `production.go`
and `welfare.go` (done) and on `internal/household` (Phase 2).

Deliverables:

1. **The `agent` type** (D81) with the fields listed there, doc comment per field
   with the spec reference, plain data, no map, no pointer to other state (the
   `directive` mailbox field is added in Phase 5 with its type, item 2), the
   slices owned. `hasJob()` reads `job.executions
   > 0 && job.remaining > 0`. The region is read from the ledger, not stored
   (FR-AGT-002); a **Guard** test lists the struct's fields with `reflect` and
   fails on any field of a money type, a quantity type or a ledger-balance name,
   and on a `Region` field (the tests of FR-AGT-002 say "a test that sums the
   agent's fields cannot find money").
2. **`Directive`** is not in this phase. Its field types (`Intent`, `Discard`,
   `JobRequest`) are declared in Phase 5, so the `Directive` type (D83: AGT 10.3,
   in `agent.go`, in the slice shape `Job *JobRequest`, `Intents []Intent`,
   `Discards []Discard`, `Hold []CommodityID`, `Sticky bool`; the credit fields
   `Borrow` and `Lend` are appended with the credit module) and the agent's
   `directive *Directive` field are added by Phase 5's PR, in `agent.go`. Phase 4's
   `agent` has every other field of D81.
3. **`newAgent(p agentInit, row *roleRow, rt *recipeTable) agent`** (N-243: 3
   parameters, not 10; `agentInit` is a small plain struct `{id AgentID; account int;
   role RoleID; weight int64; strategy int; external, mortal bool; tick int64}`):
   builds the struct from a role row and `p`, with `history` from `newHistory(row)`,
   `residues` sized by `residueLen(row.recipes, rt)` and set to 0, `job` cleared,
   `unfed`, `hunger` and `privation` 0, `roleSince` the creation tick (SPN
   5.2.2a), `target` 0 until the draw. It takes no amount and draws nothing (it
   is called for every way of creating an agent, FR-AGT-005).
4. **`reseedBeliefs`** (BEL 8.3, plan 1-4 Provides "agent row ... one helper"):
   `reseedBeliefs(a *agent, row, oldRow *roleRow, src []beliefSeed, p
   beliefs.Params)` builds the `beliefs.Seed` list inline for the commodities of
   `row.tradable` (reference price and unit cost from the caller's `beliefSeed`,
   the window mean or the market price, `Kept` and the stored belief for a
   commodity in both `oldRow` and `row`, nil `oldRow` at a creation) and appends the
   beliefs from `beliefs.Reseed` in canonical order, reusing the agent's slice. It
   takes no random word. Used by creation, by the spawner's role change and by
   migration and transport arrivals later (the others call it with their
   `beliefSeed`s).
5. **`drawCreation`** (D95): the two creation draws in order, nothing else
   (FR-AGT-006).
6. **`defaultEndowment(cfg *Config, r *Role, weight int64) (Money, []itemRow, bool)`**
   (`Role.StartStock` holds commodity strings and `roleRow` carries no start
   stock, so the mapping to `itemRow` needs `cfg.commodityID`; A1 allows the
   argument, and row 11 calls this signature) and
   **`fundInitial(l *ledger.Ledger, account int, money Money, stock []itemRow) error`**
   (AGT 3.5.1): the money `w * StartMoney` and the goods `w * quantity` of each
   `StartStock` item by checked multiplication (`ok` false is an invariant panic in
   the world's path: validation bounds it), and the grants by the `initial`
   reason through the ledger (`Transfer` from the money boundary, `Move` from the
   goods boundary; the construction phase stamps the born tick as LED 5.2.3
   says). The ledger refuses an amount below 1 (`ErrInvalidAmount`,
   internal/ledger/transfer.go and lots.go), so `fundInitial` skips a zero money
   (`StartMoney` may be 0) or a zero quantity. `fundInitial` is the uniform endowment with unstated lot keys
   (money and goods by `initial`); a warm-start group's stock with stated keys
   (WLD 5.6.1 item 3) is created by row 11 through `Ledger.MoveStated`, and its money
   may reuse the money half of `fundInitial`; a later agent is funded by the estates/world rows, not
   here (FR-AGT-007).
7. **`upkeepTotals(dst []itemRow, per []itemRow, weight int64) ([]itemRow, bool)`**
   (FR-AGT-011, N-236 (1)): each per-household quantity of `roleRow.upkeepIn(region)`
   times the weight, `checked.Mul`, the commodity order kept (it is canonical
   already), appended to `dst`; `ok` false is an invariant panic in the world's
   path (D96). It is the one place the agent's product multiplies by the weight;
   `welfare.TickCost` multiplies on its own for the cost, and no caller passes it
   a total (plan 1-6 A10).
8. **`consume`** (D94, FR-AGT-013, 014, 015, 075, 067, 068's account parameter):
   `consume(l *ledger.Ledger, a *agent, from int, per []itemRow, sc *consumeScratch)
   error`: for each upkeep row in canonical order, `consumed = min(held, U_c)`
   where `held` is `Holding(from, c)` (no reservation exists at P6.1, 6.1.6) and `U_c`
   is the total from `upkeepTotals` (`consume` calls it into the scratch's buffer;
   it does not multiply by the weight itself, so FR-AGT-011 has one
   multiplication), a `Move` to the `consumption` boundary account of the commodity
   with the reason `consumption`; the ledger takes the oldest lots first; then
   `a.unfed = w - household.Fed(consumed, per-household, w)`, `a.hunger =
   NextHunger`, `a.privation = NextPrivation` (in that order, `Hunger <=
   Privation` follows, FR-AGT-075). The shortfall `U_c - consumed_c` per upkeep row
   is left in `sc` for the world's "upkeep unmet" Debug line. An agent with no
   upkeep has `unfed` 0. A ledger error is returned (the world fails the step; a
   `Move` of at most the holding cannot refuse, so a refusal is an invariant).
   The consumption never takes more than is held and a household that cannot eat
   does not stop another. A `consumed` of 0 (the agent holds nothing) skips the
   `Move`, since the ledger refuses a quantity below 1; a `Move` of at least 1 and at
   most the holding cannot refuse.
9. **`removalJob`** is not built: the removal fan-in is the world row's; `job.abandon`
   (plan 1-5) is what it calls. This phase adds nothing for it.

**Tests (unit and integration, internal package `economysim`, A5):**

- *Guard*: the agent's field list (above); a test that a role with an empty `Recipes` list builds an agent with a zero-length `residues` and a history of the role's window (FR-AGT-010's validity half).
- `newAgent` + `drawCreation` against the vector of WLD 5.7.1: for seed 42 and agent
  IDs 1 and 2 on `(siteConstruction, 0, agent)` the target is the first word's
  range value and the residue the second word's, equal to a stream replay in the
  test (the test replays the stream itself and compares; the vectors of Stage 0
  pin the stream); the draw count is two whatever the spread (0 or 2 000) and the
  configuration; the residues are all equal and in `[0, 10 000)`.
- `fundInitial` with `StartMoney` 0 and a zero-quantity item moves nothing and returns no error; `fundInitial` + `Audit`: after creating agents of weights 1 and 7 (`StartMoney`
  10 000, `StartStock` of two items) the account holds `w * StartMoney` money and
  `w * q` of each item, the construction bucket holds the `initial` faucet, the
  ledger audit passes (FR-AGT-005, 066).
- `upkeepTotals`: weight 1, 3 and `MaxWeight`; a region with extra upkeep merges
  as `upkeepIn` says; the product is the weight times the per-household quantity;
  the overflow case through a hand-made row (`ok` false).
- `consume`, the worked example 16.3 (weight 40, staple 100 against 120, fuel 40
  against 40: `Fed = 33`, `Unfed = 7`, `hunger` 1, `privation` 1); a fed tick resets
  `hunger` to 0 and lowers `privation` by 1 to a floor of 0; an agent with no
  upkeep; an agent holding nothing (everything short, no error); a held amount above the
  need; oldest-first (two lots of different born ticks: the older goes first);
  the flow totals show the units under `consumption` and nowhere else
  (FR-AGT-067); `Audit` passes; a trainee-style call with `from` another
  account takes the units from that account (FR-AGT-068's mechanism).
- `newAgent` with a weight below 1 panics with "invariant" (D96; FR-AGT-001's "weight of at least 1"); weight 1 and `MaxWeight` build.
- `consume` allocates nothing with a warmed scratch (`testing.AllocsPerRun`, NFR-AGT-001).
- `reseedBeliefs`: creation (one belief per commodity of the tradable set, canonical
  order, the seed price with width `BeliefWidthInit`, FR-AGT-004, 005), a role
  change keeping a belief of a commodity in both sets and reseeding the others,
  and the slice reused without growth (allocation test).
- A property test over seeds: conservation (goods and money) over random
  consumption on a small fixture, with the seed in the failure message
  (testing.md 6.2).

**FR-/NFR- IDs:** FR-AGT-002 (Guard), 004, 005, 006, 010 (validity), 011, 013, 014,
015, 066 (this row's part), 067, 068 (the account parameter), 075; NFR-AGT-001 (the
consumption step allocates nothing with a warmed scratch).

**Observability:** the step logs and emits nothing; its facts (`unfed`, `hunger`, the
shortfall per row) are what the world's `UpkeepShortfall`, "households unfed"
`Warn` and "upkeep unmet" Debug line read. Nothing here draws a random number
beyond the two creation words, which are taken whether or not an observer is
attached (FR-AGT-063).

**Acceptance:** the examples reproduce; the conservation property holds over 50
seeds; `agent.go` is at least 90% covered; the pinned hash and the registry test
pass unchanged.

## Phase 5: `strategy.go`, the types, `WithStrategy` and `Directed`

**Why now:** the extension point is the public API of the row; its types are the
promise (architecture.md 6.6), so they are fixed and documented before the default
strategy is written against them. Registration reuses plan 1-3's pattern.

Deliverables:

1. **The types** (AGT 6.3, 6.4, A2, A3, D82, D97): `Strategy` with `Work`, `Trade`,
   `Revise`; `Purpose` and `PurposeOther`, `PurposeUpkeep`, `PurposeInput`,
   `PurposeTool`; `Intent`; `Discard`; `TradePlan` with `Intents` and `Discards`
   (the other fields are appended with their modules); `JobRequest` with `NoJob`,
   `Recipe`, `Executions`; `Directive` (agent.go, AGT 10.3, in the slice shape of
   Phase 4 item 2) and the agent's `directive *Directive` field, with a doc comment
   and its glossary entry (N-017 list), so the pointer Phase 4 leaves has a target; `AgentView` and its groups and `OutcomeView`; the
   belief-parameter group (D82; proposed name `BeliefParams`, fixed by Phase 1's
   AGT 6.4.1 shape; the type name differs from plan 1-4's unexported function
   `beliefParams` only by its case, so Phase 1 may rename it if the review finds
   that confusing); `Belief` and `Bounds` are plan 1-4's. Every exported
   identifier has a doc comment beginning with its name; `Strategy`'s comment states
   the contract of architecture.md 7.7 (the world's random source is a parameter;
   do not create or keep one) and the `Stateful` contract (an implementation that
   keeps state must also implement the optional pair of the checkpoint, which
   exists from Stage 2; save every piece of state, equal bytes for equal state,
   `LoadState` replaces the whole state and validates, an instance is not shared
   between worlds). **`DefaultStrategy` is not declared in this phase**: the type,
   its three methods and `var _ Strategy = DefaultStrategy{}` (architecture.md 4.5)
   arrive together in Phase 6, so this PR compiles without a stub and ships no
   placeholder behaviour. Nothing in this phase names `DefaultStrategy`
   (`Directed` is tested with a fake fallback; the reserved names `"default"` and
   `"directed"` are strings).
2. **`WithStrategy`, `ErrUnknownStrategy`, `isDirected`** (D92): `option.go` gains
   `strategies registry[Strategy]` in `options`, `WithStrategy` (reserving the
   built-in names `"default"` and `"directed"` as strings) and `isDirected`;
   `resolveStrategy` is **not** built here: it returns `DefaultStrategy{}` and
   `Directed{Fallback: DefaultStrategy{}}` for the built-in names and so needs the
   type, so it is deliverable 0 of Phase 6 (D92). `errors.go` gains `ErrUnknownStrategy`
   with its doc comment, appended after `ErrUnknownClearing` (the sentinel
   already named by CFG 7.1.4 and FR-CFG-038). Each by a small Edit after a
   re-read.
3. **`Directed`** (AGT 10.3, 10.4, FR-AGT-054, D97): `Directed{Fallback Strategy}`
   with `Work`, `Trade`, `Revise`: `Work` returns the directive's job if `Job != nil`
   (a non-nil request with `NoJob` requests nothing) else the fallback's (a nil
   `Fallback` requests nothing without a directive); `Trade` runs the fallback
   then removes the intents of the commodities in `Hold`, replaces an intent
   by the directive's for the same commodity (an intent for a commodity the
   fallback did not plan is added in canonical commodity order). A commodity in both `Hold` and the
   directive's intents keeps the directive's intent: `Hold` removes only the
   fallback's intent (ruled; AGT 10.4 item 3 gets the sentence in Phase 1, with one
   table row in the `Directed` tests); it appends the
   directive's discards, with no allocation beyond the plan's slices; `Revise` is the
   fallback's (the belief unchanged with a nil `Fallback`). The directive is read
   from the view (`HasDirective`). `Directed`'s `SaveState`/`LoadState` are not
   built (N-247).
4. **`event.go`** (D87): `ReasonRest`, `ReasonHalted`, `ReasonNoRecipe`, appended
   with doc comments saying they are log-only values of "work declined".
5. **`doc.go`**: the glossary terms of this phase, **including `strategy`**, added
   in the same change that removes `Strategy` from `notYetTerms` of
   `glossary_test.go` (the type now exists and the glossary has the term; either
   half alone fails `TestGlossary_Vocabulary`); `agent.go` and `strategy.go` join
   the file lists of `TestGlossary_DeclaredTypesNamed` and the `rootTypes` list of
   `TestGlossary_Vocabulary`. The guard checks only the files it lists
   (glossary_test.go), so every file of this row that declares an exported type
   must be listed: `strategy_view.go` too if it declares an exported view-group
   type, and `strategy_default.go` (`DefaultStrategy`) in Phase 6, which names
   `DefaultStrategy` in `doc.go` in the same change. Before declaring any exported name, check it against
   the root's names and plan 1-9's `RegionAgent` (N-248).

**Tests:**

- *Registration* (testing.md 3.9, WLD 4.1.1 order): an invalid name
  (`""`, `"A"`, `"1x"`, a name longer than `MaxIDLength`) is `ErrInvalidID`; `"default"`
  and `"directed"` and a name given twice are `ErrDuplicateID`; a nil strategy is
  `ErrNilExtension`; the three checks fire in that order for a name that breaks
  two; a registered strategy is found in `options.strategies` under its name;
  `isDirected` is true for `Directed{}` and false for a wrapper holding a
  `Directed` in a field. These are the Stage 1 done-when item "the registration
  options reject a name given twice and a nil extension". The resolution tests
  (`resolveStrategy`: registered names, `ErrUnknownStrategy`, the built-ins) are
  Phase 6's, with the function.
- *`Directed`*, a recording fallback (a hand-written fake in a `_test.go` file,
  architecture.md 10.3): the directive's job wins; a nil job falls back; `NoJob`
  requests nothing; an intent replaces the fallback's for the same commodity, a
  commodity in `Hold` is removed, discards are appended; a nil fallback with no
  directive returns the empty plan and the unchanged belief; the fallback's
  `Revise` is called for a non-nil fallback.
- *Contract guards*: a text-level `Guard:` test (the style of
  `determinism_guard_test.go`) that `strategy.go` and `agent.go` import no clock,
  no global `rand`, no `crypto/rand`; `strategy.go` has no goroutine.
- The doc-comment test (`doc_comments_test.go`) checks only listed phrases, so
  `TestDocComments_Contracts` gains a `Strategy` entry (architecture.md 7.7): the
  phrases for the world's random source (do not create or keep one) and the
  `Stateful` contract.

**FR-/NFR- IDs:** FR-AGT-019 (the three methods: a compile-time and a reflect
check of the method set on `Directed`; `DefaultStrategy`'s assertion is 6), 030
(the registration half and the reserved names; the resolution half is 6; the names
are hashed already through `Role.Strategy`: the Stage 0 hash test is re-run), 054,
056 (documented: a strategy never blocks), FR-WLD-002/003's option half for the
strategy (the `NewWorld` half is row 11); FR-CFG-038 is completed by 6; NFR-AGT-001 for `Directed.Trade` on a warmed plan.

**Observability:** the registration options and the resolution log nothing and emit
nothing; a refusal is a returned error (observability-and-logging.md 2.3). The
three reasons are log-only constants, not events, so the event catalogue is not
changed; but `TestConstants_Values` (event_test.go) pins every declared `Reason`
string with its owning spec by index range, so the change adds an `agentReasons`
slice (`rest`, `halted`, `no_recipe`, owner `doc/specs/spec-agents.md`) with its own
owner entry, as plan 1-9 does for its reasons, and the existing guard that every
`Reason` constant is documented passes.

**Acceptance:** the registration table passes; `Directed` passes against the
fake; `Strategy` is out of `notYetTerms`; `TestConstants_Values` pins the three new
reasons; the PR compiles with no `DefaultStrategy`; `go vet` and `staticcheck` clean.

## Phase 6: `DefaultStrategy` (`Work`, `Revise`, `Trade` with the ruled first-buyer tier combination)

**Why now:** the rules are tested (Phases 2, 3), the types exist (Phase 5); this
phase composes them over a view. It is tested with views built by hand, as a
consumer would build one (D91).

**The tier combination is ruled (D98, O14 closed).** The cross-tier combination is
one small function, `combineTiers`, and it is the final first-buyer rule, not a
provisional one: every tier buys at the order's limit; `Afford` lowers the limit only
for the first tier of the order to buy a unit (the lowering leaves 0 money, so no
later tier buys); a later tier that cannot buy one unit at the order's limit is
**dropped, never lowered**. This is safe (`limit * qty` is at most the sum charged,
FR-AGT-044) and takes the same draws (D93). The spec sentences (AGT 7.5.3/7.5.4) are
Phase 1 deliverable 10e. No golden exists before row 11. Ruled by coordinator,
reversible.

Deliverables:

0. **`DefaultStrategy` and `resolveStrategy`** (D92, AGT 7): `strategy_default.go`
   declares the empty struct, its three methods (below) and `var _ Strategy =
   DefaultStrategy{}` (architecture.md 4.5) in this one PR, so the type is never
   present without its methods. `option.go` gains `resolveStrategy(o *options,
   name string, path []string) (Strategy, error)` (`"default"` is
   `DefaultStrategy{}`, `"directed"` is `Directed{Fallback: DefaultStrategy{}}`, a
   registered name its value, else a `*ValidationError` with `ErrUnknownStrategy`),
   by a small Edit after a re-read; `doc.go` names `DefaultStrategy`, and
   `strategy_default.go` (and `strategy_view.go` if it declares an exported type)
   joins the `TestGlossary_DeclaredTypesNamed` file list.
1. **The work report** (D88): an unexported function over `*AgentView` returning
   the share (`welfare.WorkShare` composed with the survival rule and the
   floor, 7.1.2, 7.1.3: `Hunger > 0`, `Money < ReserveMoney`, or `welfare.FloorBinds`
   gives 10 000; a ratio absent gives 10 000), whether the floor changed the share
   (the floor condition holds and the share the ratio alone gives is below
   10 000: the same predicate as plan 1-6's `HeldByFloor`, D69 there), whether the
   agent rests (the share is 0), the recipes skipped by the halt (ascending
   `RecipeID`, outputs with a positive `CostShare` only, via `household.Halted`
   and `OutputTarget`), and the reason a request would be declined with
   (`ReasonRest`, `ReasonHalted`, `ReasonNoRecipe`, or the refusal reason of the
   view's `Reason`). The report performs the whole selection (share, halt,
   eligibility including a food cap of at least 1, `Better`) and returns the chosen
   recipe and its `kcap`, so `Work` and the counts cannot differ (JC1). The reason
   has this precedence: `rest` (share 0), then `halted` (a recipe was skipped by the
   halt and none is chosen, AGT 7.1.5), then the production `Reason` of the first
   recipe with `MaxExecutions` 0, then `no_recipe`. It writes into a caller-owned struct
   whose slice is reused; it allocates nothing with a scratch.
2. **`DefaultStrategy.Work`** (7.1): no recipes (an empty list and no own-use
   list) requests `NoJob` (FR-AGT-010); the share from the report; the eligible
   recipes with `MaxExecutions >= 1` not halted, valued by `household.Net` over the
   view's beliefs (outputs at `Centre`, inputs at `Centre`, tools by
   `production.WearCost`), compared with `household.Better` ascending by `RecipeID`
   (a strict improvement replaces, so ties keep the lower ID); a source stays
   eligible at a non-positive net; the keep-the-food cap of 7.1.6 per input that is
   also upkeep; `Executions = household.Request(welfare.Executions(weight, share), kcap)`, at least 1
   when the share is above 0 and a recipe is eligible; `NoJob` when none is. A
   recipe whose food cap is 0 gives `Request` `ok` false and **never** a request
   with `Executions` 0 (that would mean "the most it can run", AGT 6.3, B1): the
   recipe is ineligible and the next best eligible one is tried, `NoJob` with the
   reason `no_recipe` (fixed in the work report) only if none is left (ruled;
   written into 7.1.6/7.1.7 by Phase 1 before this code). A request above what can run is cut and not refused. It takes **no**
   random word (D3: the default takes none).
3. **`DefaultStrategy.Trade`** (7.2 to 7.5), in three passes over the commodities
   in canonical order (D93, D96), each pass a small function so no function exceeds
   cyclomatic complexity 10 (N-243: the planned recipe, the needs and targets, the
   tiers and draws, the money allocation, the intents): *(a)* the planned recipe `r*` (7.2: the best
   eligible recipe with `Effective > 0` (7.2 plans only recipes with positive yield; a
   zero-yield source is `Eligible` and must not become `r*`) ignoring stock, not halted, may be none; `k* = weight`), the
   per-commodity needs (`Reserve`, `Job` with **both terms**, the inputs
   `InputJobs * k* * q` and the tools `k*`, when `StartsIn` is 1, and the output
   target by `household.OutputTarget`), the targets
   by `household.ComputeTargets`, the gap or surplus by `household.Tiers`, and for
   each intent considered (the candidate rules of D93) **one `rng.Uint64()`** and
   the limit by `beliefs.Draw` (converted inline from the view's belief, bounds
   and belief-parameter group). The top of the range for a bid with `Hunger > 0`
   and `T1 > 0` is `High` of the range: `beliefs.Draw` with the word
   `math.MaxUint64` returns exactly `High` (the high word of `u * n` is `n - 1`),
   so no new beliefs API is needed and `household` does not re-derive
   `min(Ceiling, Centre + half)` (beliefs.go, `Draw`); the real draw is still
   taken. *(b)* the money stack: tier 1 over all commodities, tier 2, then
   the cap by `household.SpendCap` over the excess of the money left above the
   reserve money, then tier 3 scaled by `household.BuyShare` and bounded by the
   cap (a quantity bound: the cap never lowers the limit), each tier at the
   limit in force, with `household.Afford` lowering the limit (only for the first
   tier to buy; `Afford` never drops a tier: `combineTiers` drops a later tier whose
   `limitOut` is below the order's current limit). **One order has one
   limit (D98, the combination in `combineTiers`):** this phase builds the
   ruled rule of the header (first-buyer lowering, a later tier is dropped,
   never lowered). Each tier is
   charged at the limit it bought at and `limit * qty` is at most the sum charged. *(c)* the intents, in
   canonical commodity order (D86): a bid for the sum of the tiers or, when
   `Gap == 0` ("no bid is placed" is a property of the view, D93), an ask for
   `max(Surplus, max(0, Expiring - Reserve))` (7.3.5, `Expiring` of the view; the
   units the upkeep eats first, oldest lots first, are not offered), an ask for a
   tool only by a role that makes it (the view's `Tool` flag of the commodity and
   a non-discard output of one of the role's recipes, read from the view's per-recipe `Outputs` with their `Discard` flag; a tool the role does not make is not a
   candidate and takes no word),
   each carrying `Target`, `Reserve`, `UnitCost` (the view's, passed
   through unchanged, FR-AGT-073) and the `Purpose` of 7.3.6 (`PurposeInput` if
   `Job_c > 0` and `c` is an input of `r*`, `PurposeTool` if it is a tool of
   `r*`, else `PurposeUpkeep` if upkeep, else `PurposeOther`; both input and
   upkeep is `PurposeInput`). No discard, no
   shipment (FR-AGT-051 first half, 079). At most one order per commodity
   (FR-AGT-039). Quantities are the weight times the per-household amounts and
   limits never scale with the weight (FR-AGT-046).
4. **`DefaultStrategy.Revise`** (7.6, FR-AGT-050, D82): `beliefs.Update(b, bd, o, q,
   stock, target, p, u)` with the belief parameters of the view converted
   inline, the quote from the market report, `o.Stock`, `o.Target`, `o.UnitCost` and
   `o.Draw`; returns the new belief and discards `Result` (the world computes the
   gate itself, plan 1-4 A13).

**Tests:**

- *Unit, hand-built views, table-driven* (the worked examples of AGT 16): 16.1 (a
  poor household places the two bids `fuel` 1 at 105 and `ore` 2 at 195 and one ask
  of `item` 1, buys no buffer: `cap = 0`, `e = 0`; the draws 105, 195, 310 are set by
  a fake `rng` source, a seeded `rand.PCG`, with the words chosen so that
  `beliefs.Draw` returns them; the arithmetic of the expected values is in a
  comment), 16.2 (a richer household: `cap = 1 552`, 1 fuel and 4 staple of buffer),
  16.4 (the work share 6 000 of a cohort of 40 requests 24 executions).
- *Work*: a role with no recipes requests `NoJob`; a source recipe at a negative
  net is chosen, a recipe with inputs at a negative net is not; two recipes with
  equal value per tick tie to the lower ID; the halt skips a recipe whose co-product
  stock is at the ratio, a byproduct (`CostShare` 0) never halts (FR-AGT-036); the
  keep-the-food cap shortens `Executions` (FR-AGT-037); a table case with an
  input that is also upkeep and `Stock == U` (cap 0) never gives a request with
  `Executions` 0: the outcome is the ruled one (the recipe is ineligible, the
  next best is tried, else `NoJob` with `no_recipe`), and a Phase 7 case shows the request does not start a
  job (`maxExecutions`/`Count` with the strategy's maximum); the survival rule
  (`Hunger > 0`, money below the reserve) and the floor give 10 000 whatever the
  ratio says; the ratio absent gives 10 000 and works whenever a recipe is
  eligible (FR-AGT-033); a share of 0 requests `NoJob` with the reason `rest`;
  FR-AGT-070's shape: with every role's `TargetRatio` above 1 and an output stock
  at the ratio the recipe is still skipped; the work report's counts agree with
  `Work` for every row of the table (the same function).
- *Trade, the combination (D98, ruled)*: money 50, limit 105, `T1` = 3: tier 1 is the
  first buyer, cannot pay one unit at 105, is lowered to its money and buys 1 unit at
  50, leaves 0, and no later tier buys. Money 250, limit 105, `T1` = 3 for a commodity
  that is both upkeep and an input: tier 1 is the first buyer and buys
  `floor(250 / 105)` = 2 units at 105, charged 2 * 105 = 210, leaving 250 - 210 = 40;
  tier 2 (input) cannot pay one unit at the order's limit (40 < 105), so it is
  dropped, never lowered: **the ruled expected value is an order of 2 units at limit
  105, money left 40, reservation 2 * 105 = 210 <= 250**. (The rejected least-limit
  rule would have lowered the order to limit 40 for 3 units.)
- *Trade*: a zero-yield source recipe is never `r*` (no tool or input is bought for a
  job that cannot run); one order per commodity over a grid of states (FR-AGT-039, N-239); the
  three tiers add to the gap (the plan's bid equals the sum of the allocated
  tiers); tiers 1 and 2 are bought before tier 3 and the sum of limit times
  quantity is at most the money (FR-AGT-041); the cap and the buy share apply to tier 3
  only (FR-AGT-042, 043); an urgent bid uses the top of the range and still takes
  its draw (FR-AGT-045, a draw-count test on a recording `rng`); a commodity with
  nothing to do takes no draw; the draw count equals the number of intents
  considered whatever the money (D93), with one draw-count row for each case of
  the candidate rules: a bid candidate that cannot afford one unit (word spent, no
  order, no ask), `Gap == 0` with `Surplus > 0`, `Gap == 0` with `Expiring -
  Reserve > 0` and no surplus (ask for `Expiring - Reserve`), `Expiring > 0` but
  `Expiring - Reserve <= 0` and no surplus (no candidate, no word), and a tool the
  role does not make (not a candidate, no word); an ask is the whole surplus, expiring
  units are offered when no bid is placed, a tool is asked only by a maker
  (FR-AGT-047); a recipe with a tool: `Job` of the tool is `k*`, tier 2, `Keep`,
  and `PurposeTool`, with a commodity that is both input and tool and a commodity
  that is input and upkeep (`PurposeInput`), by hand computation (FR-AGT-038,
  048); the reservation `limit * qty` never exceeds the charge for the ruled
  combination, over a grid that includes a first-buyer lowering and a dropped later
  tier; a `StartsIn` row: the same state with `StartsIn` 1 and 3 gives `Job` non-zero
  and 0 (tier 2 and `Keep` change, and `Purpose` is not `PurposeInput` for an input
  of `r*` when `Job_c` is 0; FR-AGT-038, second clause); every intent carries `Target`, `Reserve`, `UnitCost` and the
  `Purpose` of 7.3.6 (an input that is also upkeep is `PurposeInput`, FR-AGT-048);
  the default never discards and never appends a shipment.
- *Scale and determinism* (property tests over seeds, the seed in the message,
  testing.md 6.2). The claim is conditional (Phase 1 deliverable 10c), so the
  generator builds states **inside** the conditions and the test asserts they hold:
  ample money (every tier funded in full at the drawn limit at both weights; the
  test asserts that no `Afford` lowering occurred and that money did not bind), a
  stock away from the halt threshold (the planned recipe `r*` is asserted equal at
  both weights), and output targets either from a recipe whose `X = q_o * E *
  OutputCoverTicks / (10 000 * Duration)` is an integer or compared in the `ceil(w *
  X)` form. Then a weight-`w` view equals the weight-1 view with the same words: equal
  limits; `w` times **exactly** the wants `T1`, `T2`, `T3`, `Surplus`, the asks, the
  reserves, `Keep`, a `Target` from `UseTgt` and the allocated quantities of tier 1
  and tier 2; a `Target` from `OutTgt` equal to `ceil(w * X)`; tier 3, the cap and the
  executions against the closed form on the totals, and only the want-bound tier 3
  against the band `w * q1 <= q <= w * q1 + w - 1`; no `w * q1` band for the
  executions, the cap or a cap-bound state, because
  `floor(w * X * b / 10 000)` is not `w * floor(X * b / 10 000)`. **The packs'
  duration-3, output-1 recipes (base `make_tool_a`, `make_tool_b`, second `fab_board`)
  are not exact-`w` fixtures**: they appear only in the `ceil(w * X)` form. A separate
  **divergence table** pins what is not invariant, so the claim is not re-added: the
  output target 1 at `w` 1 against 3 at `w` 7 (duration 3, output 1, `OutputCoverTicks`
  1); money 50, limit 105 lowered at weight 1 and not at weight 7 (money 350: three units
  at 105); money 250, limit 105, wants 3 at weight 1 (2 units) against money 1 750, wants
  21 at weight 7 (16 units, not 14); a stock at the halt threshold where `r*` differs
  between the weights. The "same words" comparison fixes the
  sequence of draws per commodity (the weight-1 and weight-`w` views have the same
  candidates); identical views and
  identical words give identical plans (FR-AGT-020, 031); `DefaultStrategy` is a
  zero-size type (`unsafe.Sizeof == 0` through `reflect`: FR-AGT-031 "keeps no
  state") and a second call with the same arguments returns the same value.
- *Revise*: the update equals a direct call of `beliefs.Update` for a grid of
  outcomes (FR-AGT-050); a consumer-style wrapper in a named field delegating to
  `DefaultStrategy.Revise` gives the same result (BEL 10.5).
- *Robustness*: a view with absurd numbers (a weight and an upkeep whose product
  overflows, a limit at `MaxPrice` with a quantity at `MaxQuantity`) returns a
  plan without a panic and without an order for the commodity (D96); `FuzzTrade`
  asserts no panic, at most one intent per commodity, in canonical order, every
  quantity at least 1 and every limit in `[1, MaxPrice]`, the money bound.
- *Allocation* (testing.md 8.2): `testing.AllocsPerRun` of `Work`, `Trade` and
  `Revise` with a warmed plan and a view that carries scratch is 0 (NFR-AGT-001);
  benchmarks for the targets, the allocation, the work share and the recipe choice
  (NFR-AGT-003).

**FR-/NFR- IDs:** FR-AGT-010, 019 (`DefaultStrategy`'s assertion), 020, 030 (the
resolution half), 031, 033, 034, 035, 036, 037, 038, 039, 040, 041 (the budget
half, with the ruled combination), 042, 043, 045, 046, 047, 048, 050, 051 (the
first half), 052, 069, 070, 073; FR-CFG-038 (the strategy part; the world half is row
11); NFR-AGT-001, 003. FR-AGT-044 is Phase 7's.

**Observability:** the strategy logs nothing and emits nothing (D87). The work
report supplies the world's Debug line "work declined" and its counts; the Debug
lines and `ProductionHalted` are row 11's. The draws are taken whether or not an
observer is attached (FR-AGT-063: the strategy cannot see an observer). Rule 9.3 of
observability-and-logging.md (a decision point of a strategy needs a `Debug` with
its inputs and `reason`): answered by the work report's reason and the plan itself,
which the world's Debug lines print; the strategy is the pure half.

**Acceptance:** the examples reproduce; the property tests pass over 100 seeds; the
fuzz ran 60 s with no crasher, corpora committed; the allocation tests pass;
`strategy.go` covered at least 90% by the end of Phase 7. The resolution table
(`resolveStrategy`: registered names resolve to the registered value, an unknown name is
a `*ValidationError` with `errors.Is(err, ErrUnknownStrategy)`, the path and the name by
`errors.Is`/`errors.As`, `"default"` is `DefaultStrategy{}`, `"directed"` a `Directed`
whose fallback is `DefaultStrategy{}`, and `isDirected` is true for the resolved
built-in) passes.

## Phase 7: the core view filler, the conversion and the checks against the real tables

**Why now:** the strategy works on a view; the world must be able to build the
real one without code of its own, and the plan must reach the real admission.
This phase makes the row's output usable by row 11 and proves FR-AGT-044 against
the real clearing admission.

Deliverables:

1. **`viewScratch` and `newViewScratch(cfg *Config, rt *roleTable, rc *recipeTable)`**
   (D91): the backing slices of one `AgentView`, sized once from the largest role,
   and the plan (`TradePlan` with its slices) and the outcome view; not saved, not
   shared between worlds.
2. **`fillCore(v *AgentView, a *agent, c *viewCtx) error`** (D85 (iii), A7, A3,
   FR-AGT-024, 071, 073): fills the Identity, Money, State, Upkeep, Job,
   per-commodity, per-recipe, Tunables (section 12 and D82) and Mailbox groups
   from the agent, the ledger (`Account` for the region and the available money,
   `Holding` for the stock, `Expiring`), the tables, the market reports and
   windows, the anchors, the seasons and the production reads (`yieldTable.effective`
   for `Effective`, `productionScratch.maxExecutions` for `MaxExecutions` and its
   reason), and the welfare reads (`history.Ratio(TickCost(...))`, `WorkingTicks`, the
   role's effective window, the target ratio). `UnitCost` is the anchor of the region
   only for a commodity the role makes, i.e. a non-discard output of one of the role's recipes (derived from `makers(c)` intersected with the role's ascending recipe list; `roleRow` holds no such set; a discard output such as the base pack's `waste` is not "makes") and
   **0 otherwise** and while the region has no observed wage (FR-AGT-048, 7.3.6; the anchor
   table is nonzero for any commodity a local recipe makes, so the check is the filler's);
   `Tool` comes from the role and recipe table (item 2a). `fillCore` also
   computes `MaxExecutions`, the welfare reads, `Bounds` (`beliefs.Limits(Ref,
   UnitCost, p)` with the same zeroed `UnitCost` the view carries, BEL 3.2.4) and the
   belief-parameter group from `viewCtx`; `StartsIn` is 1 without a job, else
   `job.remaining + 1` (AGT 6.4.3; `job.remaining` semantics of production.go), so
   `Trade` forms `Job_c` only when the agent can start a job next tick (FR-AGT-038);
   row 11's `fillAgentView` builds `viewCtx` (the
   per-region slices of A7) and adds module groups only. Copies, never pointers into live
   state: the slices are scratch copies, a strategy that writes to a view changes no
   state and no later view (FR-AGT-024). The cost of the fill is proportional to the
   tradable set plus the recipes and does not depend on the weight (NFR-AGT-002).
   Every fan-in rule holds: this function is core, called first by the world's
   `fillAgentView`, which then calls the module fillers (none in the slice).
2a. **The `Tool` flag** (A3, JC2): `recipeRow` and `roleRow` hold no tool flag, so
   `newRecipeTable` (recipe.go, plan 1-2) gets one appended per-commodity field,
   true when the commodity is a required tool of any recipe of the configuration; a
   unit test shows a tool of another role's recipe is flagged for a non-maker, and a
   commodity that is no tool is not; a tool whose only appearance in the role's recipes
   is a discard output is not "made" (base `waste`) in both the filler and `Trade`.
3. **`orderOf(a *agent, in Intent) Order`** and **`normalise(in *Intent)`** (D85
   (iv), AGT 6.5.2): the intent's `Limit` becomes the order's `Price`, `Quantity`
   the order's; a `Target` or `Reserve` above `MaxQuantity` is treated as 0 (a negative `Target`, `Reserve` or `UnitCost` is treated as 0 too, O13 closed) and a
   `Purpose` outside the enumeration as `PurposeOther`, returning that it changed
   (the world logs a Debug line); the `Tradable` flag of `orderFacts` is `row.isTradable(c)`
   (the world reads it; this row tests the pair). `not_tradable` and
   `duplicate_order` are the admission of plan 1-3 and are not re-implemented
   here (FR-AGT-025, 026); this row proves the pair of calls.
4. **`workReport` exposure for the world** (D88): the exported-to-the-package
   function the world calls for the counts, with the scratch type it fills.
5. **`outcomeStock(held, reserve int64) int64`**: `max(0, held - reserve)`, the
   `Stock` of `OutcomeView` (FR-AGT-049); the world fills the rest of the
   outcome view from the book and the window.
6. **The reasons and the glossary** as needed by the above.

**Tests:**

- *Filler against the real tables* (external test `economysim_test` with
  `export_test.go`, loading `testdata/chains/one.toml` and `second/second.toml`
  through `config/`, A5): for every role and a range of states (held stock, money,
  hunger, a job in progress, a published price, an empty window) the filled view's
  fields equal an independent computation in the test of the same quantities
  (the stock from the ledger, the effective upkeep from `upkeepIn`, the ratio from
  the ring, `MaxExecutions` from a direct `maxExecutions`); the tradable set in
  the view is the role's; `UnitCost` is 0 for a commodity the role cannot make even
  when another role's recipe gives the region an anchor for it, and `Tool` is true
  exactly for the commodities that are a required tool of some recipe; the belief parameters converted back equal
  `beliefParams(World)` field by field (D82); a strategy that writes every field
  of the view leaves the world's tables and the next fill unchanged (FR-AGT-024,
  a `Guard:` test); the fill allocates nothing with a warmed scratch.
- *FR-AGT-044, the property against the real admission* (integration, `testing.Short`
  skip): over 200 seeds, roles of `one` and `second`, states drawn by a seeded
  generator (money from 0 to many ticks of upkeep, stocks from 0 to above the
  target, prices anywhere in the belief range, hunger 0 and above), build a view,
  run `DefaultStrategy.Trade`, convert the intents with `orderOf`, and admit them
  in plan order into a real `book` over a real ledger with the facts the world
  would build (`isTradable`): none is rejected for `insufficient_funds`,
  `insufficient_stock`, `duplicate_order`, `not_tradable`, `overflow`,
  `holding_limit` or `receipt_limit`; the failing seed is in the message. It runs
  against Phase 6's ruled combination. The
  money reserved by the admission never exceeds the balance. This is the
  evidence for D86: the order of submission (canonical commodity order) rejects
  nothing.
- *Work through production*: for every role of the packs and a range of states the
  default's `JobRequest` is accepted by `productionScratch.maxExecutions`/`start`
  (it is never `not_on_role` for a role with a list, and a role with no recipes
  is refused `not_on_role` by `admit` for any request, FR-AGT-010's second half,
  FR-AGT-028, 074 for the production half); a request above `k0` is cut, not
  refused.
- *Cohort invariance on the real tables* (FR-AGT-052), inside the conditions of Phase 1
  deliverable 10c: states with ample money (no tier lowered or money-bound at either
  weight, asserted), a stock away from the halt threshold (the same `r*`, asserted),
  and, for the targets, a recipe whose `X = q_o * E * OutputCoverTicks / (10 000 *
  Duration)` is an integer for exact `w` times. The same state at weight 1 and at
  weight 7 then gives the same limits and 7 times the wants, the asks, the reserves,
  `Keep`, a `Target` from `UseTgt` and the allocated quantities of tiers 1 and 2;
  a `Target` from `OutTgt` equals `ceil(7 * X)` (the packs' duration-3, output-1 recipes,
  base `make_tool_a`, `make_tool_b` and second `fab_board`, are checked only in this
  form, never as exact 7 times); the tier-3 quantity (want-bound case only) lies
  within the band of one floor, and the cap, the executions and a cap-bound allocation
  are compared with their closed form on the totals. A state where money binds,
  where `Afford` lowers a limit at one weight only, or where `r*` flips at the halt
  threshold is the divergence row of Phase 6, not an invariance case.
- *`normalise` and `orderOf`* (AGT 6.5.2, O13): `Target` or `Reserve` above `MaxQuantity` and negative -> 0, negative `UnitCost` -> 0, `Purpose` out of range -> `PurposeOther`, each with the `changed` result (and false for a clean intent); `Limit` -> `Price` and `Quantity` -> `Quantity` in `orderOf`. A `Target` from `OutTgt` above `MaxQuantity` (a valid configuration with `output_cover_ticks` near its bound, FR-AGT-059 bounds the use targets only) is cut to 0 here, and `OutputTarget`'s doc comment says so.
- *The pair through the real admission* (FR-AGT-025, 026): a consumer-style plan with an intent outside the role's tradable set and two intents for one commodity, converted by `orderOf` and passed to the real `book.admit`, is refused `not_tradable` and `duplicate_order`.
- *The work report* (D88): a strategy that writes every field of the view does not change the `workReport` computed from the view before the call.
- *`StartsIn` in the filler* (FR-AGT-038): a job in progress gives `StartsIn =
  remaining + 1`, no job gives 1 (hand-set jobs of the packs' recipes).
- *`outcomeStock`* (FR-AGT-049): `max(0, held - reserve)` at held below, equal to and above the reserve.
- *`floorChanged`* (FR-AGT-072): the work report's flag equals `welfare.HeldByFloor` over a grid of working ticks, windows and `MinWorkShare` (0 turns the floor off).
- *Benchmark* (NFR-AGT-002): `fillCore` and `Trade` across weights 1, 7 and `MaxWeight` (`b.ReportAllocs()`), the cost not depending on the weight.
- *The reasons*: the `Reason` constants are documented and appear once.

**FR-/NFR- IDs:** FR-AGT-024, 025, 026 (through the admission), 028 (the production
half), 044, 046, 049, 052, 071, 072 (the work report is the pure function; the
count and the `Warn` are row 11's), 073, 074; NFR-AGT-001, 002.

**Observability:** none emitted. `Debug` lines are the world's. The filler builds no
event and calls no observer; it reads the "observer supplied" flag nowhere (the
view does not depend on observation, FR-AGT-063).

**Acceptance:** the property passes over 200 seeds; the fill and `Trade` allocate
nothing with a warmed scratch; `agent.go` and `strategy.go` are at least 90%
covered by the end of this phase.

## Phase 8: the exit gate and the traceability

**Why last:** the row is done when every requirement it owns has a test that fails
when the behaviour is broken and every shared file is consistent. It requires Phase 7
merged.

Deliverables:

1. **The requirement table** (below) is checked: every FR-AGT/NFR-AGT ID assigned to
   this row has a test with a `Requirement:` line (testing.md 4.2) and the table of
   this plan is updated with the test names. A text search over `doc/specs/spec-agents.md`
   and the tests lists no ID of this row without a test.
1a. **Identifier scan** (FR-AGT-065): a `Guard:` test parses `agent.go`, the strategy
   files and `internal/household` and fails on a string literal equal to a commodity,
   recipe or role identifier of `testdata/chains/` (no branch on an identifier).
2. **Mutation check** (testing.md 9.3, 9.4): each agent runs the new tests against a
   deliberately broken copy under the scratchpad (flip the tier order, drop the
   `Hunger > 0` urgency, take two draws for an intent, multiply by the weight twice,
   return the fallback's job when the directive has one, sort the intents by tier, lower a later tier's limit instead of dropping it or take the least limit over the tiers: the Phase 6 money-50 and money-250 tests must fail);
   `gomu` only inside Docker with a CPU and memory cap, if used at all
   (CLAUDE.md global rule), never on the host.
3. **The gates by hand** (O3): `gofmt -l .` prints nothing; `go vet ./...`;
   `staticcheck ./...`; `go test -short -race ./...`; the coverage command of
   testing.md 9.1 (root at least 90%, `internal/household` at least 90%, no package
   below its previous figure); `go mod tidy` produces no diff; Lizard (limits of N-243: cyclomatic complexity 10, 8 parameters, about 500 lines a file) over the new files with no new issue; the pinned hash test
   (`TestHashConfig_ReferencePinned`) passes; the glossary test, the import guard,
   the determinism guard, the doc-comment test, the registry test and the event
   catalogue test pass unchanged; `testdata/golden/` is untouched.
4. **Fuzz runs**: `FuzzTiersAndCap`, `FuzzWorkRules` and `FuzzTrade` 5 minutes each if
   time allows, otherwise 60 s each recorded as such; corpora committed.
5. **Documents**: the roadmap row 7 to "done" with the commit and the coverage; the
   documentation index row; the notes updated (the disposition table's outcome
   lines, the new notes' status); the plan's status line.

**FR-/NFR- IDs:** the table below.

**Observability:** the gate checks that no file of this row writes to stdout/stderr,
calls `slog.Default()` or reads a global provider (the Stage 0 text guards cover
the root and `internal/`).

**Acceptance:** the table is green; the gates pass; the roadmap and the index are
consistent.

### Requirements of AGT and who owns each (the traceability the audit checks)

Row 7 = this row. "World" = row 11. Module = the owner row of Stage 2/3.

| ID | Owner and phase |
|----|-----------------|
| FR-AGT-001 | World (ID space, WLD 5.3); this row only a weight of at least 1 in `newAgent` (Phase 4 test) |
| FR-AGT-002 | Row 7, Phase 4 (Guard) |
| FR-AGT-003 | Plan 1-2 (core terms); want items: the demand module (N-236 (3)) |
| FR-AGT-004, 005 | Row 7, Phase 4 |
| FR-AGT-006 | Row 7, Phase 4 (`drawCreation`); the world calls it with the keyed stream |
| FR-AGT-007 | Estates and world rows |
| FR-AGT-008, 009 | World, estates and spawner rows (removal, weight change) |
| FR-AGT-010 | Row 7, Phases 4 and 6; the world's refusal is plan 1-5's `admit` (Phase 7 test) |
| FR-AGT-011 | Row 7, Phase 4 (`upkeepTotals`) |
| FR-AGT-012 | Stage 0 (validation) |
| FR-AGT-013, 014, 015, 075 | Row 7, Phases 2 and 4 |
| FR-AGT-016, 017 | World (aggregate event and `Warn`) |
| FR-AGT-018 | Stage 0 (validation) |
| FR-AGT-019 | Row 7, Phase 5 (`Directed`, the method set) and Phase 6 (`DefaultStrategy`) |
| FR-AGT-020 | Row 7, Phase 6 (pure half); the keyed stream is the world's |
| FR-AGT-021, 022, 023 | World (calls and order); `Revise` itself is Phase 6 |
| FR-AGT-024 | Row 7, Phase 7 |
| FR-AGT-025, 026 | Plan 1-3 (admission); the pair tested in Phase 7 |
| FR-AGT-027 | World (discards against the ledger) |
| FR-AGT-028 | Plan 1-5 (`admit`); tested through the default in Phase 7 |
| FR-AGT-029 | World (`beliefs.Clamp` after `Revise`, plan 1-4) |
| FR-AGT-030 | Row 7, Phase 5 (registration, reserved names) and Phase 6 (`resolveStrategy`, the check); `NewWorld` calls it (world) |
| FR-AGT-031, 033, 034, 035, 036, 037 | Row 7, Phases 2, 3, 6 |
| FR-AGT-038 | Row 7, Phases 2, 6 (`Job` zero unless `StartsIn` is 1: the `StartsIn` row) and 7 (the filler forms `StartsIn`) |
| FR-AGT-032 | World and estates |
| FR-AGT-039 to 043, 045 to 050, 052 | Row 7, Phases 2 and 6 (the tier combination of 041 is final in 6; 052 only inside the conditions of Phase 1 10c) |
| FR-AGT-044 | Row 7, Phase 7 (against the real admission); the seed range of REF 8 is the world's golden |
| FR-AGT-051 | First half (no discard with storage off): Phase 6; second half: STO |
| FR-AGT-053, 055, 056 | INT (Stage 2); `Directed` Phase 5 |
| FR-AGT-054 | Row 7, Phase 5 |
| FR-AGT-057 | Plan 1-6 scratch and the world row (the totals); `Purpose` is Phase 6 |
| FR-AGT-058, 059, 060 | Stage 0 (validation) |
| FR-AGT-061, 062, 063 | World (the event, the Debug guard, non-interference); the pure half Phases 4, 6 |
| FR-AGT-064 | Checkpoint row (Stage 2) |
| FR-AGT-065 | Row 7, Phase 8 (a `Guard:` scan, deliverable 1a); no Stage 0 guard covers it |
| FR-AGT-066 | Row 7 for the consumption step (Phase 4), the world for the whole tick |
| FR-AGT-067 | Row 7, Phase 4 |
| FR-AGT-068 | SKL (Stage 3); the account parameter is Phase 4 |
| FR-AGT-069, 070 | Row 7, Phases 2, 3, 6 |
| FR-AGT-071, 073, 074 | Row 7, Phase 7 |
| FR-AGT-072 | Plan 1-6 (the predicate), row 7 Phase 6/7 (the report), the world (count, `Warn`) |
| FR-AGT-076 to 085 | Modules (DEM, DeathCarry, TRN, SUB, MER, ORG, STO) and the world (FR-AGT-077) |
| NFR-AGT-001 | Row 7, Phases 2 to 7 incl. 6 (allocation tests); the world's step baseline is row 11 |
| NFR-AGT-002 | Row 7, Phase 7 (the benchmark across weights, listed in its tests) |
| NFR-AGT-003 | Row 7, Phases 2, 3, 6 |
| NFR-AGT-004 | Row 7, Phases 2, 3 (fuzz) |

## Cross-plan contract

The coordinator uses these two tables to check consistency across the plans of
Stage 1. A "Consumes" row names the plan or phase that provides the symbol, or
says ASSUMED where no plan fixes it. The signatures in Provides are what the world
row (row 11) calls; they are indicative of the arguments (A1) and fixed in
capability.

### Consumes

| Symbol, type, sentinel, spec edit or field | Provided by | Status |
|---|---|---|
| The AGT tunables and role fields (`World.CoverTicks` ... `ProduceHaltRatio`, `RestBand`, `Role.Recipes`, `Upkeep`, `Trades`, `StartMoney`, `StartStock`, `Strategy`), their bounds and validation | Stage 0, `config.go`, `bounds.go`, `validate.go` ([plan-phase-0-foundation.md](plan-phase-0-foundation.md)) | Built (023cf97) |
| `Side`, `Bid`, `Ask`, `Reason` and the production and clearing reasons, `Cause`, `AgentSpawned`, `UpkeepShortfall`, `ProductionHalted` | Stage 0, `event.go` | Built |
| `siteWork`, `siteTrade`, `siteOutcome`, `siteCreation`, `stream.keyed`, `keyedPair`, `draw`, `drawRange`, `word` | Stage 0, `random.go` | Built |
| `registry[T]` (`add`, `find`), `validID`, `ErrInvalidID`, `ErrDuplicateID`, `ErrNilExtension`, `*ValidationError`, `ErrUnknownClearing`'s pattern (`resolveClearing`) | Stage 0 and [plan-phase-1-3.md](plan-phase-1-3.md) D36 | Built; the strategy and spawner rows reuse it unchanged |
| `internal/checked`: `Add`, `Sub`, `Mul`, `MulDiv`, `MulDivCeil`, `MulDivFloor` with `ok` | Stage 0 | Built |
| The role table (`newRoleTable`, `(*roleTable).row`, `roleRow.recipes`, `upkeep`, `tradable`, `member`, `window`, `upkeepIn(RegionID)`, `isTradable`), the recipe table (`recipeRow`, `makers`), `itemRow{commodity, quantity}`, `cfg.Recipes[id].Duration`, `cfg.Commodities[id].ReferencePrice`/`Durability` read by ID, `Moment`, `Calendar.SeasonAt` | [plan-phase-1-2.md](plan-phase-1-2.md) Phases 2 to 5 and its "Cross-plan contract" | Built |
| `Order`, `Trade`, `LotKey`, `marketReport` (`Price`, `Age`, `BestBid`, `BestAsk`, `HasBestBid`, `HasBestAsk`), `orderFacts.Tradable`, `book.admit`, `Clearing` | [plan-phase-1-3.md](plan-phase-1-3.md) | Built |
| `Belief`, `Bounds`, `beliefParams`, `beliefs.Limits`, `Draw`, `Update`, `Reseed` and `Seed`, `Window` (`Mean`, `Low`, `High`, `Position`), `Quote`, `Outcome` | [plan-phase-1-4.md](plan-phase-1-4.md) and its Provides | Built; the belief store per agent is two `int64` per commodity of the tradable set, canonical order, held here (D81) |
| `job`, `residueLen`, `setResidues`, `admit`, `productionScratch`, `newProductionScratch`, `maxExecutions`, `start`, `yieldTable.effective`, `yieldShocks`, `job.abandon`, `reweigh`, `warmJob`, the cost-anchor table and its root read `(*productionScratch).unitCost(region, commodity)` (`production_report.go`), `production.WearCost` | [plan-phase-1-5.md](plan-phase-1-5.md) and its Provides | Built; "the agent holds one `job` by value as AGT 3.1 says" is plan 1-5's O1, closed here (D81) |
| `history` (`welfare.Ring`), `newHistory`, `newTarget`, `welfare.WorkShare`, `Executions`, `FloorBinds`, `HeldByFloor`, `TickCost`, `Ring.Ratio`, `Ring.WorkingTicks`, `paramsOf`, `welfareParams` | [plan-phase-1-6.md](plan-phase-1-6.md) and its Provides | Built; "the agent type has a slot index and a weight and holds the history field" is its ASSUMED, closed here (D81): the agent holds the weight and the `history`; the slot is **not a field**, it is the agent's index in the world's agent table (`AgentID` is dense in creation order, plan 1-11 A2), and the region is read from the account (FR-AGT-002), not stored |
| `ledger.Ledger`: `Open`, `Account` (region, balance, reserved), `Holding`, `Expiring`, `Move`, `Transfer`, `Boundary`, `Audit`, `InConstruction` | [plan-phase-1-1.md](plan-phase-1-1.md), [plan-phase-1-5.md](plan-phase-1-5.md) Phase 2 | Built |
| The ledger reason constants `consumption`, `initial` and their mapping to the root `Reason` | `ledger.go` ([plan-phase-1-1.md](plan-phase-1-1.md)) | Built |
| The market state per (region, commodity) (the report, the window), the published prices, the cost-anchor slice and the season at P1.1 as per-region slices, assembled into `viewCtx` | `world.go`/`market.go` (row 11) | **ASSUMED**: the world row builds the slices from its market state (A7); this row names the need |
| `NewWorld` calls `resolveStrategy` for every role (and later for an external agent), builds the strategy table and keys the streams before each call (`siteWork` for `Work`, `siteTrade` for `Trade`, `siteOutcome` with the commodity for `Revise`), takes the outcome's word | `world.go` (row 11) | **ASSUMED** (N-246) |
| `(*estates).acceptMoney(l *ledger.Ledger, from int, r RegionID, amount Money, reason ledger.Reason) error` (agent to estate, no transfer for `amount < 1`) and `(*estates).acceptGoods(l *ledger.Ledger, from int, r RegionID, reason ledger.Reason, only func(commodity int) bool) (lots int64, err error)` (the lots of the selected commodities, nil for all; the count is of lots, the coordinator's ruling for `GoodsListed`) | [plan-phase-1-8.md](plan-phase-1-8.md) (D105, D106, D101) | **ASSUMED**, not called by this row: a weight reduction (`split`, `merge`, `starvation`; `deaths` arrives with the population module and is not in the ledger registry yet) sends its money through `acceptMoney` and its goods through `acceptUnits` (`(*estates).acceptUnits(l, from, r, commodity int, qty int64, reason) (units int64, err error)`, per commodity, plan 1-8 D106), and a role change goes through `acceptGoods`, the one home of the pool minimum and bound, so the callers (the world executor, row 11, and the spawner, row 9) use them and this row's `agent.go` adds no second path. `agent.go` holds no transfer to an estate |
| The estate account opened at a removal, the pool funding a later agent (`grant`) | `estates.go` (row 8) | **ASSUMED**, not consumed by this row |
| The executor's role change and weight change, which call `reseedBeliefs`, `setResidues`, `history.Resize`, `job.abandon` and `job.reweigh` | `world.go` (row 11, the executor of the spawner's plan; plan 1-9 D110) | **ASSUMED**, not consumed by this row |
| `Stateful` and the checkpoint section | `checkpoint.go` (Stage 2) | Not consumed; `Directed` gets its methods then (N-247) |
| AGT 6.4.1, 6.4.4, 3.1, 3.2, 7.1.2, 7.1.5, 7.5.1, 14.3, 14.4, 20, 22 edits; `spec-beliefs.md` 10.1; `architecture.md` 2.6; `AGENTS.md` layout; roadmap row 7 | This row, Phase 1 | Owed here |

### Provides

For the world row (row 11) to call, with the arguments each needs. All unexported in
package `economysim` unless noted.

| What | For | Contract |
|---|---|---|
| `type agent struct{...}` (D81), `(*agent).hasJob() bool`, `type Directive struct{...}` (exported, D83) | world, estates, spawner, integration | Plain data; the core fields only; the module fields are in module slices (N-244) |
| `newAgent(p agentInit, row *roleRow, rt *recipeTable) agent` (`agentInit{id, account, role, weight, strategy, external, mortal, tick}`; plan 1-11 uses the same signature) | world (construction, entrants, births, `AddAgent`), spawner | No ledger access, no draw, no amount; the caller funds and seeds |
| `drawCreation(s *stream, a *agent, targetRatio, spread int64)` | world (construction after the warm start, P7.1(d), `AddAgent`) | Takes exactly the two words of AGT 3.5.2 in order from the stream the caller has keyed to the agent's creation stream |
| `defaultEndowment(cfg *Config, r *Role, weight int64) (Money, []itemRow, bool)`; `fundInitial(l *ledger.Ledger, account int, money Money, stock []itemRow) error` | world (construction; a warm-start group's stock with stated lot keys is row 11's `MoveStated`, its money may reuse this) | Reason `initial`; checked; unstated keys only; not for a later agent |
| `reseedBeliefs(a *agent, row, oldRow *roleRow, src []beliefSeed, p beliefs.Params)` and `type beliefSeed` | world, spawner (role change), migration, transport, integration | Canonical order, no random word; `oldRow` nil at a creation; the one helper of BEL 8.3 |
| `upkeepTotals(dst []itemRow, per []itemRow, weight int64) ([]itemRow, bool)` | world (consumption, the view), spawner (the unfed share) | The only multiplication of the agent's upkeep by the weight (FR-AGT-011) |
| `consume(l *ledger.Ledger, a *agent, from int, per []itemRow, sc *consumeScratch) error` and `consumeScratch` (the shortfall per upkeep row) | world (P6.1, the trainee's `from` later) | Sets `unfed`, `hunger`, `privation`; moves to the `consumption` boundary; never takes more than is held |
| `newViewScratch(cfg *Config, rt *roleTable, rc *recipeTable) *viewScratch`, `(*viewScratch).fillCore(v *AgentView, a *agent, c *viewCtx) error`, `type viewCtx` | world (`fillAgentView` calls it first) | Copies; allocation-free when warmed; `fillCore` computes `MaxExecutions`, the welfare reads, `Bounds` and the belief group from `viewCtx`; row 11 builds `viewCtx` and adds module groups only |
| `workReport(v *AgentView, dst *workReportBuf)` with the share, `floorChanged`, `rest`, the skipped recipes ascending and the decline reason | world (P2.5 Debug line, P2.8a `ProductionHalted` rows, the floor `Warn` and the snapshot count with plan 1-6's predicate) | Pure; the same function `Work` reads (D88); the world computes the report from the view **as filled, before it is passed to `Work`** (or from a refill), never from a view the strategy has had, so a writing strategy cannot change the counts (to be carried into plan 1-11's call position and N-246) |
| `normalise(in *Intent) (changed bool)`, `orderOf(a *agent, in Intent) Order`, `outcomeStock(held, reserve int64) int64` | world (`checkTradePlan`, the outcome loop of P5.4) | The intent to order conversion and the clamps of AGT 6.5.2; `Tradable` for `orderFacts` is `row.isTradable(c)` |
| `Strategy`, `AgentView`, `OutcomeView`, `TradePlan`, `Intent`, `Discard`, `Purpose` (and constants), `JobRequest`, `DefaultStrategy`, `Directed`, `Directive` | consumers, `strategy/` (Merchant, Stage 2), integration | Exported; the shapes are the promise (A1) |
| `WithStrategy(name string, s Strategy) Option`, `ErrUnknownStrategy`, `resolveStrategy(o *options, name string, path []string) (Strategy, error)`, `isDirected(Strategy) bool`, `options.strategies` | world (`NewWorld`'s name check, WLD 3.2.2), integration (`AddAgent`, `Direct`) | The pattern of `resolveClearing`; the built-in names are `default` and `directed` |
| `ReasonRest`, `ReasonHalted`, `ReasonNoRecipe` | world (the Debug line "work declined"), tests | Log-only |
| `household.NextPrivation(before, unfed int64) int64` (the **single implementation** of the privation step, AGT 5.4.1; coordinator ruling of the consistency pass) | row 11 (through `consume`, P6.1), [plan-phase-1-9.md](plan-phase-1-9.md) (cites it for FR-SPN-012 and drops its own `occupation.PrivationStep`; its SPN 20 correction is theirs) | `Privation + 1` if `unfed > 0`, else `max(0, Privation - 1)`; saturating; pure; no other row duplicates it |
| `internal/household`: `Fed`, `NextHunger`, `NextPrivation`, `ComputeTargets`, `Tiers`, `SpendCap`, `BuyShare`, `Afford`, `Net`, `Better`, `Eligible`, `OutputTarget`, `Halted`, `KeepFoodCap`, `Request` (scalar) | the strategy; available to `viability.go` (AGT 19), which plan 1-10 does not consume (VIA 8.4.1 has its own stock rule) | Pure, primitives, `ok` on overflow; no state |
| The glossary terms of N-017 in `doc.go` | all later rows | Added before use |
| `internal/household` in architecture.md 2.6 and `AGENTS.md` | all | Phase 1 |
| No configuration field, default, bound, event or snapshot field; one sentinel (`ErrUnknownStrategy`) and three log-only reasons | - | `testdata/golden/off/` and the pinned hash do not move |

## Security review

The module is a library with no network, file access, credentials or persistence.
This row adds one trust boundary that matters: a consumer's `Strategy`
implementation, which is untrusted code that runs inside the step and returns data
the world must not believe, and a consumer's directive, which is data.

- **A faulty or hostile `Strategy`.** It may return intents for a commodity outside the
  tradable set, two for one commodity, a quantity beyond its money, a `Target`
  above `MaxQuantity`, a `Purpose` out of range, discards of more than it holds, a
  recipe off the list or a huge plan. None reaches the ledger unchecked: the
  admission of plan 1-3 rejects with a reason (`not_tradable`, `duplicate_order`,
  `insufficient_funds`, ...), `normalise` clamps the belief inputs (D85 (iv)), the
  discard is refused whole by the world, a recipe off the list is `not_on_role`.
  A plan can be long: the cost is its length (the world's loop) and the duplicate
  scan of the book is linear (N-241), a consumer's own cost. It cannot create or
  destroy money or goods, only fail an order.
- **The random source.** The `rng` is the world's; the contract forbids keeping it or
  creating another (7.7). The library cannot detect a violation; the doc comment of
  `Strategy` and the determinism guard say so. The default takes exactly one word per
  intent considered and none for `Work` and `Revise`.
- **State.** The default keeps none (FR-AGT-031); a consumer's that keeps state must be
  `Stateful` (Stage 2); the library cannot detect one that does not. A
  `Directed.Fallback` that holds state is saved through `Directed` at Stage 2.
- **Integer overflow.** Every product that can overflow is checked. The world's views
  cannot overflow (validation bounds, FR-AGT-059); a view a consumer builds can, so
  the default treats a failed check as "no order" and never panics on view data
  (D96, fuzzed). A panic remains only for a violated programmer invariant of the
  world's path, with a message that says so (architecture.md 8.1).
- **The directive.** It is consumer data left between steps; `Directed` applies it
  through the same admission, so it cannot overspend or trade outside the tradable
  set (AGT 10.4 item 5). It is copied into the view, so a later write to the
  mailbox cannot change a call in progress.
- **Panics in a strategy** are not recovered (AGT 6.1.5) and propagate to the caller
  of `Step`, as an observer's do.
- **Resource exhaustion.** The view and the plan are scratch sized once; the fill is
  bounded by the tradable set and the recipes of one role; nothing grows with the weight.
- **Not applicable:** authentication, authorisation, injection, secrets, cross-tenant
  exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer. This row
  adds exported identifiers: `Strategy`, `AgentView` and its groups, `OutcomeView`,
  `TradePlan`, `Intent`, `Discard`, `Purpose` and four constants, `JobRequest`,
  `DefaultStrategy`, `Directed`, `Directive`, `WithStrategy`, `ErrUnknownStrategy`
  and three `Reason` constants. Each is appended, not inserted (architecture.md
  2.10.5), and a change to one after the first release is breaking (6.6), so the shapes
  of A1 to A3 are the thing to get right now and are written into the spec in Phase 1
  for the review. A module appends fields to `AgentView`, `TradePlan` and `JobRequest`
  at the end.
- **Document contracts.** Phase 1 edits (7.5.3 and 7.5.4 included, deliverable 10e) AGT 3.1, 3.2, 6.4.1, 6.4.4, 7.1.2, 7.1.3, 7.1.5, 7.5.1,
  14.3, 14.4, 20 (FR-AGT-010, 033, 034) and 22, `spec-beliefs.md` 10.1, `architecture.md`
  2.6, `AGENTS.md`, the roadmap row 7 and the index; it renumbers no requirement
  (testing.md 4.1) and changes the text of FR-AGT-033 and 034 only. `doc/notes.md`
  gets the dispositions and N-244 to N-247 and a fresh tag; `architecture.md` gets a
  fresh RULE-TAG.
- **Configuration and the hash.** No field is added, so the configuration hash and the
  checkpoint format do not change; the strategy names are hashed already through
  `Role.Strategy`.
- **Dependency failure during rollout.** The order is the dependency order: Phase 1
  (documents), Phases 2 and 3 (a leaf package), Phase 4 (needs the role, job and
  welfare rows, done), Phase 5 (types and options, no `DefaultStrategy`), Phase 6 (the
  rules over the views, `resolveStrategy`), Phase 7 (needs the clearing admission and
  production reads, done, and 6). If a sibling row
  changes a symbol this row calls (`marketReport`, `book.admit`, `maxExecutions`), the
  compile error names it; the Cross-plan contract is the list to check. If row 11
  (world) is not ready, nothing here is blocked: every phase tests with hand-built
  views, a real ledger and the real admission (A5).
- **Rollback.** Each phase is one PR and adds files or appends lines; a revert removes
  it. Phase 1's rule-file edits (`architecture.md` 2.6) are one line and revert alone.

## Testing (summary, testing.md section 11)

1. **Requirements touched and their tests.** The table "Requirements of AGT and who
   owns each" lists every FR-AGT and NFR-AGT ID with its owner and phase; the tests carry
   `Requirement:` lines (testing.md 4.2). This row's own: FR-AGT-002, 004, 005, 006, 010,
   011, 013, 014, 015, 019, 020, 024, 025 and 026 (the pair test, Phase 7), 028 (the production half), 030, 031, 033 to 043, 044,
   045 to 052, 054, 066 (consumption), 067, 069, 070, 071, 072 (partial: the report, Phase 6/7), 073, 074, 075 and NFR-AGT-001
   to 004.
2. **Lowest level.** The arithmetic is unit-tested in `internal/household` (Phases 2, 3);
   the strategy over hand-built views (Phase 6); the agent over a real ledger (Phase 4);
   the filler and the admission pair at the integration level (Phase 7, A5). No end-to-end
   test is possible without the world row: the golden and the invariants over the REF seed
   range are row 11's and 12's.
3. **New boundaries and error paths.** The registration errors (`ErrInvalidID`,
   `ErrDuplicateID`, `ErrNilExtension`) and `ErrUnknownStrategy` with `errors.Is`; overflow
   of the household functions (`ok`); the view of absurd numbers; the consumption of an
   agent holding nothing; the fallback nil of `Directed`; each asserted by sentinel, never
   by message (testing.md 3.9).
4. **Bug fix?** No. If an audit finds one, a `Fix:` test is written first (5.3).
5. **Invariants and edges (`Guard:`).** The agent has no balance field (FR-AGT-002); the
   strategy has no field (FR-AGT-031); one order per commodity (N-239); the belief group
   equals `beliefParams` field by field (D82); the draw count equals the intents considered
   (D93); the default's plan rejects nothing at the real admission (FR-AGT-044);
   `strategy.go` and `agent.go` use no clock, global `rand` or goroutine.
6. **Golden file.** None changes: `testdata/golden/` does not exist yet and the pinned
   hash is untouched (N-217).
7. **Parsing or validation.** No new parsing; the validation is Stage 0's. Fuzz tests are
   added for the pure rules (`FuzzTiersAndCap`, `FuzzWorkRules`, `FuzzTrade`, NFR-AGT-004).
8. **Coverage.** Root and `internal/household` at least 90% (testing.md 9.1); the figures
   go in the status line at the end. No scope falls.
9. **Broken-code check.** Phase 8 item 2.

## Observability (summary, observability-and-logging.md section 9)

1. **New failure path or rejected input?** Returned, not logged: `ErrInvalidID`,
   `ErrDuplicateID`, `ErrNilExtension`, `ErrUnknownStrategy` (a `*ValidationError`) and the
   `error` of `consume`. A strategy's bad intent is an `OrderRejected` event of plan 1-3's
   admission; a bad discard is the world's Debug line.
2. **New state transition?** The agent's `unfed`, `hunger` and `privation` change at P6.1;
   their aggregate is `UpkeepShortfall` (exists, emitted by the world). No new event: none is
   needed and none is added (observability-and-logging.md 4.3).
3. **New decision point in a strategy?** Yes, the work decision and the trade plan. The
   strategy is pure and cannot log; the world logs "work declined" with the `reason` of the
   work report (`rest`, `halted`, `no_recipe` or the production refusal) and the Debug lines of
   AGT 14.3, guarded by `Enabled`, and the plan itself appears as `OrderPlaced` events.
   `over_budget` is removed because nothing can produce it (D87).
4. **New loop?** The tick loops are the world's; the strategy and the filler loop over a role's
   tradable set and recipes with no logging and no allocation (NFR-AGT-001, tested).
5. **Changed event, metric name, unit or attribute?** None.
6. **New metric attribute?** None. The three reasons are log-only and not values of
   `economysim.reason`.
7. **`Error` or `Info`?** Neither is used.
8. **Logging an error also returned?** No log exists in this row.
9. **Tests.** The allocation limits and the non-interference of the no-op path are tested here
   for the pure parts (`AllocsPerRun`); the logs, events, cardinality and golden rendering that
   need a world are row 11's and 12's. The draws are taken whether or not an observer exists
   because the strategy cannot see one.

## Open items

- **O1 (closed, ruled by coordinator, reversible: D86).** The intents are in canonical commodity order and AGT 7.5.1's "tiers'
  priority" sentence is reworded. Reason: determinism (architecture.md 7.5, 7.6) and the
  draw order D5 and one order per commodity (7.3.3) fix it.
- **O2 (closed, ruled by coordinator, reversible: D87).** `over_budget` is removed from AGT 14.3 and 22
  because nothing can produce it.

- **O3.** CI: still no row and no owner (open item O3 of plan 0); the commands of `AGENTS.md`,
  the coverage reading and the timed fuzz runs are done by hand in each change.
- **O4 (closed, ruled by coordinator, reversible).** D80, D81, D83, D84 are final. D80
  adds a package to the layout of the rules; D81 reads architecture.md 2.10.1 over the AGT 3.1
  sketch.
- **O5.** Sibling assumptions: the world row (11) must build the per-region slices of `viewCtx`
  from its market state, key the streams before each call, call the functions of the Provides
  table at the positions of WLD 6 and take the outcome's word (N-246) and, in the spawner's executor, call
  `reseedBeliefs`, `setResidues`, `history.Resize`, `job.abandon` and `job.reweigh` in its
  role-change and weight-change steps; the estates row (8) opens the estate at a removal. None of them blocks a phase of this row.
- **O6.** `Directed.SaveState` and `LoadState`, the directive in the checkpoint and the
  strategy-name table (N-247), the demand module's want items (FR-AGT-003, 076) and every module
  group of the views are appended by the PRs that own them (architecture.md 2.10.5).
- **O7.** AGT 21 questions that stay open and are not decided here: 17 (cover against shelf
  life), 19 (a switching margin), 20 (does `Target` count `Inbound`; the default ignores it).
- **O8 (closed, coordinator ruling of the consistency pass).** `household.NextPrivation` (Phase 2)
  is the single implementation of the privation step (AGT 5.4.1, SPN 5.1); plan 1-9 dropped its
  `occupation.PrivationStep` (N-249 is closed). No cross-check test is owed.

- **O13 (closed, ruled by coordinator, reversible).** A negative `Target`,
  `Reserve` or `UnitCost` from a consumer's strategy is treated as 0, symmetrical to the
  clamp above `MaxQuantity`; Phase 1 deliverable 10f writes it into AGT 6.5.2 and Phase 7
  item 3 (`normalise`) implements and tests it.
- **O14 and O15 (closed, ruled by coordinator, reversible).** The cross-tier
  combination is the first-buyer rule of D98 (see there for the reason); the
  least-`limitOut` alternative is rejected. There is no provisional rule and no
  owner gate: Phase 6 implements the rule as final, and no phase waits.

- **O9 (JC1, closed by coordinator ruling).** (a): a recipe whose keep-the-food cap
  is 0 is ineligible and the next best is tried; `NoJob` with `no_recipe` only if
  none is left. Reason: AGT 7.1.4 already chooses among recipes with
  `MaxExecutions >= 1`, and FR-AGT-037 forbids the request that would eat upkeep
  food. Phase 1 (deliverable 10a) rewords 7.1.6/7.1.7 so "at least 1" no longer
  contradicts FR-AGT-037.
- **O10 (JC2, closed by coordinator ruling).** (i) `Tool` is derived from the whole
  configuration; (ii) a tool the role does not make is not a candidate and takes no
  word (AGT 7.3.5 "placed only by a role that makes it"). This fixes the D5 stream
  layout before any golden exists.
- **O11 (JC3, closed by coordinator ruling).** (a): the cap bounds the tier-3
  quantity only; 7.5.3 lowers a limit only to the money left. Phase 1 states it
  once in AGT 7.4.4/7.5.4.
- **O12 (JC4, closed by coordinator ruling).** (a): `Hold` removes only the
  fallback's intent, so the directive's own intent for a held commodity survives
  (FR-AGT-054: "remove from the fallback's plan the commodities in Hold"). Phase 1
  adds the sentence to AGT 10.4 item 3.

## Decisions table

| ID | Decision | Basis | Confirm |
|----|----------|-------|---------|
| D80 | Create `internal/household`; add it to architecture.md 2.6 and `AGENTS.md` | AGT 22, 19; N-228; NFR-AGT-004 | Ruled by coordinator, reversible |
| D81 | Unexported `agent`, core fields only; module state in module slices | architecture.md 2.10.1; AGT 3.1 sketch | Ruled by coordinator, reversible |
| D82 | The belief parameters (fifteen fields) as an exported group of both views | Coordinator ruling (a); FR-AGT-050; AGT 7 | Coordinator ruled |
| D83 | `Directed`/`Directive` in `strategy.go`/`agent.go`; no `directive.go` | architecture.md 2.6; roadmap row | Ruled by coordinator, reversible |
| D84 | "Welfare module off" becomes "the ratio is absent" | no `Modules` pointer for welfare; AGT 7.1.2 last sentence | Ruled by coordinator, reversible |
| D85 | The split with the world row, (i) to (v) | AGT 22; N-236 | Coordinator accepted |
| D86 | Intents in canonical commodity order | WLD 7.3 D5; AGT 7.3.3 | Ruled by coordinator, reversible: determinism (architecture.md 7.5, 7.6) |
| D87 | No `over_budget`; log-only `rest`, `halted`, `no_recipe` | AGT 14.3; strategy purity | Ruled by coordinator, reversible: nothing can produce it |
| D88 | One work report for `Work` and the world's counts | AGT 7.1.3, 7.1.5; OBS 4.4.6 | Ruled by coordinator, reversible: one source of truth for the count |
| D89 | Scalar `internal/household` API | architecture.md 2.9; NFR-AGT-001 | Implementer-level |
| D90 | Tool wear in the recipe value is `production.WearCost` | one formula (PRO 9.4, WEL 2.1.4) | Implementer-level |
| D91 | `viewScratch` sized once; hand-built views allocate | AGT 6.4.1; NFR-AGT-001 | Implementer-level |
| D92 | `resolveStrategy` like `resolveClearing`; the registration order of WLD 4.1.1 | WLD 4.1.1; plan 1-3 D36 | From the specs |
| D93 | One `Uint64` per intent considered, in the first pass | AGT 7.5.3; WLD 7.3 D5 | From the specs |
| D94 | `consume` takes the account to eat from | AGT 5.7; SKL 4.2 | Implementer-level |
| D95 | The two creation draws in one function | AGT 3.5.2; WLD 5.7.1 | From the specs |
| D96 | World views panic on a violated invariant; consumer views never | architecture.md 8.1, 13 | Ruled by coordinator, reversible: invariant panic (architecture.md 8.1) |
| D97 | The directive is copied into the view by value | AGT 6.4.2 | Implementer-level |
| D98 | One order has one limit: only the first tier to buy may have it lowered by `Afford`; a later tier that cannot pay one unit at the order's limit is dropped, never lowered (the former least-`limitOut` rule is rejected) | AGT 7.4.1, 7.5.3, 7.5.4; FR-AGT-044 | Ruled by coordinator, reversible; O14 and O15 closed |
| JC1 | JC1: a recipe with food cap 0 is ineligible, next best tried, else `NoJob`/`no_recipe` | AGT 7.1.4; FR-AGT-037 | Ruled by coordinator, reversible |
| JC2 | JC2: `Tool` from the whole configuration; a non-maker's tool is no candidate and takes no word (fixes the D5 stream layout before any golden) | AGT 7.3.5 | Ruled by coordinator, reversible |
| JC3 | JC3: the cap bounds the tier-3 quantity only | AGT 7.5.3, 7.4.4 | Ruled by coordinator, reversible |
| JC4 | JC4: `Hold` removes only the fallback's intent | FR-AGT-054 | Ruled by coordinator, reversible |
