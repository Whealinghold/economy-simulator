# Plan: Stage 1, row 5: production (jobs, the yield table, `UnitCost`)

**Status: Done (2026-10-16), uncommitted on stage-1-5; Phases 1 to 8 built. Phase 8 gate 2026-10-16: root 99.2%, internal/production 96.0%, internal/ledger 99.6%, all-packages total 99.2% (-coverpkg); FuzzSite, FuzzEffective, FuzzUnitCost and FuzzExecutions 5 min each (about 42M execs each), no crasher; traceability: 58 cited IDs all in the specs.** Sequences the fifth row of
Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`internal/production` and
`production.go` (jobs, yield table, `UnitCost`)", into eight phases. It
implements [spec-production.md](../specs/spec-production.md) (PRO) for the
slice, on top of [spec-ledger.md](../specs/spec-ledger.md) section 8 (`Transform`)
and 5.4 (`WearItems`). It is built against `doc/rules/architecture.md` (the
`internal/` boundary of 2.9, the fixed-point and overflow rule of 3.5, the
determinism rules of section 7, the layout of 2.6), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR
or the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md)
(D3 and D5 above all). The few calls that are this plan's own are flagged D50
to D60, D67 and A1 to A9 for the owner's override (D61 to D66, D69 and D70 are plan 1-6's; a decision ID is a plan's own namespace, so the D67 of this plan, the job's stored factor, is not the damper D67 of `spec-stability.md` and is cited from outside as "D67 of plan-phase-1-5"); the ones that affect another
row are in the Cross-plan contract and the open items.

No PRD exists for this row. The requirements source is the roadmap row and PRO.
The siblings are [plan-phase-1-1.md](plan-phase-1-1.md) (the ledger, in
progress), [plan-phase-1-2.md](plan-phase-1-2.md) (the recipe, role and calendar
tables), [plan-phase-1-3.md](plan-phase-1-3.md) (clearing),
[plan-phase-1-4.md](plan-phase-1-4.md) (beliefs) and
[plan-phase-1-6.md](plan-phase-1-6.md) (welfare); the agent, spawner and world
rows have no plan yet.

Spec codes: PRO production, LED ledger, GDS goods, AGT agents, WEL welfare and
wages, BEL beliefs, CAL calendar, CFG config, RES resources, SKL skill
barriers, STO storage, VIA viability, WLD world, OBS observability, CHK
checkpoint, SCL scale. `FR-`/`NFR-` IDs are the requirement IDs of those specs.

## What Stage 0 and the earlier rows already give, and what is left

| Concern | Where it is |
|---------|-------------|
| `Recipe`, `Item`, `Output`, `Tool`, `YieldProfile`, `SeasonYield`, `DefaultRecipe` | `config.go`, `defaults.go` (Stage 0) |
| The bounds `MaxRecipeTicks`, `MaxYieldBP`, `MaxRequires`, `MaxRecipeItems`, `MaxQuantity`, `MaxWeight` | `bounds.go` (Stage 0) |
| Every validation rule and warning of PRO 10 (yield profile shape, unknown season, cost shares, tool durable, eligible recipe, overflow of 10.3, `unused_recipe`, `unrunnable_recipe`, `durable_not_required`, `source_unbounded`) | `validate.go` (Stage 0, rule ledger "built") |
| `RecipeExecuted`, `RecipeAbandoned`, `ProductionHalted`, `Amount`, `HaltRow`, `ReasonAgentRemoved`, `ReasonRoleChanged`, `ReasonRecipeInput`, `ReasonRecipeOutput`, `ReasonDisposal`, `ReasonWear` | `event.go` (Stage 0) |
| `internal/checked`: `Add`, `Sub`, `Mul`, `MulDiv`, `MulDivRem`, `MulDivFloor`, `MulDivCeil`, `MulDivHalfEven` with an `ok` result | Stage 0 |
| The ledger: in the tree `Move` and the lot operations, `Transform(id, inputs, outputs []Line)`, `WearItems`, `Wear`, `WearUnits`, `Expiring`, the registry with `recipe_input`, `recipe_output`, `disposal`, `wear`, boundary accounts, flow totals, and `Audit() error` (`internal/ledger/audit.go`) | [plan-phase-1-1.md](plan-phase-1-1.md) Phases 2 to 6 (in the tree, read 2026-10-13) |
| The recipe table (items, outputs and tools by typed ID, the source flag), the makers of a commodity, the role table (recipes by role), `Calendar.At` and `SeasonAt` | [plan-phase-1-2.md](plan-phase-1-2.md) Phases 2, 4 and 5 |

What is left, and is this row:

- **The rules of PRO as pure functions** (`internal/production`): the site
  factor lookup (6.2), the effective factor (6.3), the carried-remainder output
  (6.6), the rescale of a job at a weight fall (5.4), the execution count with
  its first-applicable refusal reason (4.2), and `UnitCost` with the cost
  anchor of a commodity (9.1, 9.3, 9.4).
- **`production.go`**: the resolved yield table and the yield-shock slot (6.1,
  6.8), the job type and its state (5.1), the residue layout (6.6, 15), and
  the steps that apply the rules to the ledger: the start (4.3, 4.4), the
  continuation and completion (5.2, 5.3), abandonment (5.4), the weight change
  (5.4), the job of a warm start (WLD 5.6.1 item 4), the value of the two
  events, the aggregates of the snapshot, and the table of cost anchors.

## Scope

**In scope.**

- `internal/production`: the functions above, over primitive slices and
  integers, standard library and `internal/checked` only (architecture.md 2.9).
- Two additive ledger operations in new files of `internal/ledger` (Phase 2,
  D54, D58): `Discard` (create and dispose, no room check) and `Holding` (held
  and reserved units of a commodity in an account).
- `production.go`, unexported, converting inline at the call (2.9): the
  `productionScratch` (the buffers the steps convert into; no second copy of
  the recipe table, A2), the
  `job` type, the resolved yield table with the shock slot, the residue
  layout, `start`, `advance`, `complete`, `abandon`, `reweigh`, the warm-start
  job, the cost-anchor table, and the builders of the values of
  `RecipeExecuted` and `RecipeAbandoned`.
- The `Reason` constants a slice start can be refused with (`not_on_role`,
  `zero_yield`, `no_site`, `no_tool`, `no_input`), appended to `event.go`.
- The glossary terms these files use, in `doc.go`, before they use them
  (architecture.md 3.2).
- The spec, note and roadmap edits these files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| The production phase itself: the rotation of 5.5.1 and 6.1.4, the two-pass `Work` and starts (P2.1 to P2.7), the tick's season read once at P1.1, "free agent" (not continued, not away, not teaching), `Work` and `JobRequest`, `Debug` lines of a refused start, the `Warn` "recipe output disposed", event emission, `ProductionHalted` | Roadmap Stage 1, `world.go`, `agent.go` and `strategy.go` rows. This plan gives them the steps and the values |
| The storage of a job and of residues on the agent record (AGT 3.1, 3.2), the residue draw at creation (WLD 5.7.1, FR-PRO-026), the reset of residues at a role change (AGT 3.7) | `agent.go` and `world.go` rows (D50). This plan gives `residueLen`, `offset` and the type |
| The regional wage and the published prices the cost anchor reads | `welfare.go` ([plan-phase-1-6.md](plan-phase-1-6.md), the region wage) and the market state of the `world.go` row. This plan takes them as slices |
| Calling the welfare scratch (`addWear`, `markWorking`) | The `world.go` row (D51): the steps return the facts, and welfare is built after this row |
| Which recipe an agent starts and how many (`Work`, the target-income rule, the floor) | `strategy.go` row, `welfare.go` |
| The ration of scarce endowments, `Draws`, `takeEndowment`, abundance (RES) | Roadmap Stage 2, `resources.go`. The execution count has a seam (D52) |
| `Needs`, `Builds`, `Repairs`, facility wear (CAP); rent and titles (LND); `k_room`, the pledge and `no_room` (STO); `not_permitted` (ORG); technology gain (TEC); a hired job (EMP); an own-use job (SUB); the abandon reasons `region_changed` (MIG, FR-PRO-050) and `principal_left` (SKL, EMP) | Each module's PR, appended (architecture.md 2.10.5, D3, D5). Their parameters are present in this row's functions as the neutral values 10 000 or no limit |
| `TrainingPays` (PRO 8.3, 9.4) | Roadmap Stage 3, `skills.go` (D55) |
| The public yield-shock call `SetYieldShocks` and the event `YieldShocked` | Roadmap Stage 2, `integration.go` (N-224 `deferredCore`). The slot and its arithmetic are here (D53) |
| The checkpoint encoding of the job, the residues and the shocks (PRO 15, CHK tag 6) | Roadmap Stage 2, `checkpoint.go`. The state here is plain, ordered data, sparse for shocks |
| The viability checks V1 to V4 on the two example packs (FR-PRO-046), the warm-start selection rule of VIA 8.5.2 | Roadmap Stage 1, `viability.go` row |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- `internal/production` is standard library and `internal/checked` only, never
  imports the root, knows no commodity, recipe, role or region identifier, and
  holds no map, closure or package variable (architecture.md 2.9, 7.1, 7.5,
  9.1; PRO 19).
- Every product that can overflow goes through `internal/checked`
  (architecture.md 3.5): `k * q * E`, the cost sums, the rescale. PRO 6.6 names
  the 128-bit helper.
- Rounding: truncation toward zero in the four steps of the effective factor
  (6.3, a rule constant), the output division of 6.6, and the rescale of 5.4;
  `UnitCost` and the wear cost round **up** (9.1).
- Production draws no random number (PRO 6.6.2, FR-PRO-026): the one residue
  word is the world's creation draw, and no function of this row takes a
  `*rand.Rand`.
- A job is plain data, one per agent, saved with its executor (PRO 5.1, 15).
- Yield is read when the job starts and the pending output is fixed then
  (6.4, FR-PRO-021). A job of duration `d` started in tick `s` completes in
  tick `s + d - 1` (5.2, FR-PRO-013).
- A refused start consumes nothing, and a refused ledger `Transform` leaves
  the ledger unchanged and the job unstarted (FR-PRO-010, LED 2.6).
- The master order of refusal reasons is PRO 4.2; this spec owns it, and a
  reason of a module that is off cannot occur.
- The flow totals count `recipe_input`, `recipe_output`, `disposal` and `wear`
  by commodity; pending output is in no account and is reported by the
  snapshot (LED 8.2.2, PRO 14.3).

Decided by this plan, to be written into the owning spec in Phase 1 before
code, each flagged for the owner's override in the review:

- **D50. Where the job lives.** `production.go` defines the `job` type, the
  rules over a job and the residue layout. It does not decide the storage: its
  functions take a pointer to a job and a slice of residues, so the agent row
  keeps them as the fields AGT 3.1 and 3.2 give the `Agent` (`Job`,
  `Residues`), and the checkpoint writes them in section 6 in agent order
  (CHK 4.2 tag 6). PRO 19 says "`agent.go` and `world.go`" and the roadmap
  says `production.go`; both are true of this split. If the agent row prefers
  slices of its own indexed by agent slot (architecture.md 2.10.1), the same
  functions serve (open item O1).
- **D51. This row emits nothing and calls no other row.** The steps return
  plain results (executions, the wear facts, the outputs and disposed
  amounts, the working executions of the tick), and fill caller-owned slices
  with the values of the two events. The world row emits them behind the
  "observer supplied" flag, logs the `Debug` and `Warn` lines and calls the
  welfare scratch with the facts. The reasons: row 6 is built after this row,
  an observer and a logger are the world's, and a step that emits would
  allocate or branch on the flag in the hot loop (observability-and-logging.md
  4.5). This is the pattern of plan 1-1 for `BoundaryFlow` rows.
- **D52. The count is the least of an ordered list of limits.** The execution
  count takes the weight, the strategy's maximum and a list of
  `(cap, reason)` pairs built by the caller in the master order of PRO 4.2,
  and returns the least cap and, when it is 0, the reason of the first pair
  with cap 0. A module appends its limit at its place in the order by adding
  a pair, and changes nothing here. In the slice the list is the yield pair
  (cap 0 or none, its reason fixed by D59: `zero_yield` or `no_site`),
  `no_tool` per tool and `no_input` per input.
- **D59. `no_site` and `zero_yield` are disjoint by their cause, and the
  cause is the profile, as PRO 6.3 and TEC 5.1.2 already say (Phase 1 words
  PRO 4.2 and FR-PRO-009 to match; an audit fix of 2026-10-13 replaced an
  earlier wider rule that made a profile of 0 a `no_site`).** PRO 6.3 says
  `no_site` is "a `Sited` recipe with no profile", TEC 5.1.2 says "a `Sited`
  recipe that a region has no profile for does not start (`no_site`) and a
  profile of 0 does not start (`zero_yield`)", and PRO 12.2 gives `deep_crater`
  a profile `all = 0` ("cannot run here"), which is `zero_yield`. Only PRO 4.2
  ("a `Sited` recipe with no positive factor") reads wider, and it is the line
  Phase 1 rewords. A `Sited` recipe with no profile has `E` 0 too, and the
  master order lists `zero_yield` first, so the two are made disjoint by
  definition: with `has` whether a profile row exists for (region, recipe)
  and `E` the effective factor of 6.3, the start is refused `no_site` if and
  only if the recipe is `Sited` and `has` is false (whatever the shock,
  technology, trainee and abundance terms are); it is refused `zero_yield` if
  and only if `E` is 0 and the start is not refused `no_site` (a profile of
  `all = 0`, a seasonal profile with no entry for the season, a site factor
  of 0 of a recipe that is not `Sited`, or a positive site factor with a
  shock, technology, trainee or abundance term of 0). The first applicable
  reason of 4.2 is then unambiguous and the master order is unchanged. The
  strategy sees the reason (AGT 7.3), so this is a rule and not an
  implementation choice. Nothing in TEC 5.1.2, `spec-infrastructure.md`
  4.1.3, EMP 6.1.1 item 3 or FR-EMP-023 changes: they name the two reasons and the site factor of 6.2,
  and the rule above is the one they already state.
- **D53. The yield-shock slot is here.** PRO 6.8 says production owns the slot
  and the arithmetic. The slot is a sparse list sorted by (region, recipe),
  default 10 000, with an unexported setter that validates `[0, MaxYieldBP]`.
  The public call and the event are the integration row's.
- **D54. Create-and-dispose is a sibling ledger operation, `Discard`
  (answered by coordinator 2026-10-13).** A discard output and the part of an
  output that `MaxQuantity` leaves no room for are "created and disposed in the
  same step" (PRO 5.3.1, 5.3.2, 7.2; LED 7.2.1). Completion must never fail
  for room (FR-PRO-029), and the account may already hold up to `MaxQuantity`
  of the commodity. The ledger's `Transform(id, inputs, outputs []Line)` (in
  the tree, `internal/ledger/transform.go`) refuses an output line with
  `ErrBalanceOverflow` when `held - inputs + total` exceeds the ledger's
  maximum quantity, where `held` counts every unit of the account, earmarked
  ones included, and it has no form that disposes a line it creates. So the
  amount cannot be created in the account and moved out. Plan 1-1 owns
  `transform.go`, so this row does not touch it: Phase 2 adds
  `Discard(account, commodity, quantity)` in new files of `internal/ledger`,
  which creates under `recipe_output` and sinks under `disposal` with no
  change to the account and no room check. The completion is one `Transform`
  of the outputs, each cut to the room (`maxQuantity - held`, reserved units
  counted), and then one `Discard` per discard output and per overflow part.
- **D58. A goods holding read (answered by coordinator 2026-10-13).** The
  tree's ledger exposes no per-commodity held and reserved read. Phase 2 adds
  `Holding(account, commodity) (held, reserved, err)`, zero allocation, which
  plan 1-3 also assumes. It is resolved, not assumed. Two quantities are read
  from it and they differ: the **available** units for inputs and tool items
  are `held - reserved`, as the ledger takes only unearmarked units
  (`Transform` inputs, `WearItems`); the **room** for an output against the
  maximum quantity is `maxQuantity - held`, with the reserved units counted,
  because the bound of `Transform` is on `held`. A step that used
  `held - reserved` for the room would hand `Transform` a line it refuses and
  panic.
- **D60. What `RecipeExecuted.Outputs` and `.Disposed` hold (Phase 1 writes
  it into PRO 14.1, FR-PRO-029 and FR-PRO-030, and aligns IND 7.2.4 and the
  doc comment of `event.go`).** The three readings found in the documents
  (PRO 14.1 "outputs created", `event.go` "units of the outputs that were
  discarded", IND 7.2.4 "gross of `Disposed`") are made one. `Outputs` is the
  gross quantity created by the completion for each output item: the discard
  outputs and the overflow part included, because both are created under
  `recipe_output`. `Disposed` is the part of those units that was never held:
  the whole quantity of each discard output plus, for a non-discard output,
  the part that `MaxQuantity` left no room for (and, with STO, the part a cut
  of the room left no room for). So for every commodity the units added to
  the principal's account are `Outputs - Disposed`, the flow totals under
  `recipe_output` are the sum of `Outputs` and under `disposal` the sum of
  `Disposed`, and an indicator valuing `Outputs` gross of `Disposed` (IND
  7.2.4) double counts nothing. Both slices are in output order and list a
  row only for a quantity above 0 (the golden rendering omits a zero field,
  and a row of 0 would render as a bare commodity). The `Warn` of PRO 14.2 is
  fed by `Disposed` rows of non-discard outputs only, as a discard output is
  waste by design and not an anomaly.
- **D67. The two values of `RecipeExecuted` that have no source in the job
  (Phase 1 writes it into PRO 5.1.1, 14.1 and 15 and CHK 4.2 tag 6; the lines
  are already in the specs, written by the audit fix of 2026-10-13).** The
  builder of `RecipeExecuted` fills `StartTick` and `EffectiveYield`
  (`event.go`, PRO 14.1), and the job of PRO 5.1.1 held neither. `StartTick`
  is **derived, not stored**: a job started in `s` completes in `s + d - 1`
  (5.2, FR-PRO-013), so `StartTick` is the completing tick minus the recipe's
  `Duration` plus 1. That holds after a weight fall (5.4 does not move the
  completion) and for a warm-start job (VIA 8.5, FR-VIA-035: `Remaining =
  Duration - o`, so the virtual start is `o` ticks before the first
  production tick and `StartTick` may be negative, which the event's `int64`
  allows). `EffectiveYield` is **stored**: the job gains one field, `Yield
  int64`, the effective factor `E` of 6.3 fixed at the start. It cannot be
  recomputed at completion: the season, the shock and the technology may have
  changed (6.4, FR-PRO-021), and a world restored mid-job has no start-time
  inputs (FR-PRO-043). `start` sets it from the `E` it used for the pending
  output, the warm-start constructor sets it from the `E` of the virtual
  season, `reweigh` leaves it, and the builder copies it. It is one field of
  the job record: PRO 5.1.1 and 15, CHK 4.2 tag 6 (the job's fields in order:
  `Recipe`, `Remaining`, `Executions`, `Principal`, `Yield`, then the pending
  quantities); AGT 3.1 holds the job as `*Job` defined in PRO and AGT 3.2
  points at PRO 15, so neither AGT row changes. The checkpoint encoding is
  Stage 2's (`checkpoint.go`); no code exists yet, so the field costs no
  format migration.
- **D55. `TrainingPays` waits for `skills.go`.** PRO 8.3 and 9.4 expose it as
  a pure function for the entry rule of SKL and for calibration; both are
  Stage 3. Nothing in the slice calls it, so it is not built (architecture.md
  2.1, 4.2).
- **D56. Reasons of the slice.** The constants appended to `event.go` are the
  five a slice start can be refused with. The rest of the master order and the
  abandon reasons `principal_left` and `region_changed` are appended with
  their modules, in the order of the master list, never inserted.
- **D57. A commodity that is both an input and a tool is a defensive rule.**
  PRO 4.4.3 says a tool "never appears in `Inputs`", but no rule of PRO 10.2
  enforces it (the "may appear in both lists" of 3.1.1 is `Inputs` and
  `Outputs`, seed grain, and says nothing of `Requires`). PRO 4.2 derives the
  input and tool limits separately, so a recipe that did list one commodity
  in both would let `k` exceed what the stock covers and the start would be
  refused by the ledger after the fact. The count therefore carries one
  defensive rule, written into PRO 4.2 in Phase 1 as such and citing 4.4.3:
  for a commodity found in both, the cap is the available stock over the
  input quantity `q` plus 1, `stock / (q + 1)`, attributed to `no_tool`. The
  1 is one tool item per execution (4.4.2: each execution wears a distinct
  item), not `WearPerUse`, which is a quantity of durability and not of
  items. The plan does not claim the combination is legal. Rejecting it in
  validation (a Stage 0 rule and sentinel) is the alternative and is left to
  the owner (O8); until then the rule costs one comparison and cannot change
  a result of a valid pack that follows 4.4.3.

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** Names and signatures in this plan are indicative of a capability.
  PRO fixes only `Term`, `Wear` and `UnitCost` (9.4); every other signature of
  `internal/production` and `production.go` is the implementer's, within PRO
  and architecture.md 2.9, as plan 1-1 A6 says. A row that needs a name reads
  the code of this row once merged.
- **A2.** The one owner of the recipe data this row reads is plan 1-2's
  `recipeTable` (Phase 4: `*recipeRow` with `inputs`, `outputs`, `tools`,
  `source`; the world holds it and the canonical `Config`, from which
  `Duration`, `Sited` and a tool commodity's `Durability` are read by ID).
  This row builds **no second table** of recipes. `internal/production` has
  one flat per-recipe argument struct (`Recipe`: inputs as `Term`, tools as
  `Wear`, outputs with commodity, quantity, cost share in basis points and
  the discard flag, and the duration), used only as the parameter of the
  anchor function (Phase 4) and filled **inline, at the call**, from a
  `*recipeRow` into slices of the scratch of Phase 5 (`productionScratch`:
  reusable `[]Term`, `[]Wear`, `[]ledger.Line`, the limit-pair slice and the
  `[]Recipe` of the anchor step, sized once at construction and reused, so a
  step allocates nothing). That is the scratch of architecture.md 2.9, held
  by the world: there is no converting function, file or type (the scratch
  type owns buffers and holds `*Config` and `*recipeTable`; it converts
  nothing), and no persistent copy of the table that could drift. Each of
  `start`, `maxExecutions`, `complete` and the anchor step takes the
  `*recipeRow` (or the table) and does its conversion in its own body.
- **A3.** The site lookup is a binary search over the profiles sorted by
  `(RegionID, RecipeID)`, which is the canonical order of `Config.Yields`
  (`config.go`). A dense (region, recipe) index would be up to
  `MaxYields` = 10^7 cells; the search costs `O(log n)` per start, which the
  per-agent cost of NFR-PRO-002 allows. A profile row holds its seasons as
  resolved season indexes in ascending order. The resolvers of Stage 0 cover
  commodity, recipe, role and region only, and plan 1-2 adds no season
  resolver: `Config.Yields` holds each profile's seasons sorted by season ID
  string (`config.go`, canonical form), while the calendar orders seasons by
  declaration and `Moment.Season` is that index. Phase 5 therefore builds the
  ID-to-index table from `Calendar.Seasons` once (at construction, not in the
  tick; a sorted slice of (ID, index) searched by binary search, no map is
  iterated) and re-sorts each profile's entries by index.
- **A4.** The residue layout of an agent is the role's recipes in ascending
  `RecipeID` (plan 1-2 role table), then the effective own-use list when SUB
  exists, each recipe contributing one residue per output item in output
  order, discards included (PRO 15: "one per (recipe, output item)"). The
  layout is a function of the role and the recipe table, never saved (CHK
  4.1.3). A role change resets the residues to 0 (AGT 3.7, SPN 4.2); the agent
  row calls it.
- **A5.** The pending output of a job is a slice of quantities in output
  order, one per output item including discards, held in the job and reused
  at the next start of the same agent (`append` to `[:0]`), so a steady-state
  start allocates nothing (NFR-PRO-001). The job type holds no pointer into
  other state.
- **A6.** The ledger is called through a `*ledger.Ledger` and account IDs
  passed in by the caller. The Phase 6 steps use, as they are in the tree
  (`internal/ledger/transform.go`, read 2026-10-13): `Transform(id int,
  inputs, outputs []Line) error` with `Line{Commodity int; Quantity int64}`,
  `WearItems(id, commodity int, n, units int64) error`, `Wear(id, commodity
  int, units int64) error` and `WearUnits(commodity int, bucket bool)`; and
  `Discard` and `Holding` (Phase 2 of this plan, named and fixed). The
  ledger's refusals are errors and are handled by what the step knows: an
  error of the start's input `Transform` is returned to the caller with the
  job untouched (the `Transform` is atomic: the refusals it can give, such
  as `ErrUnknownAccount` for a commodity with no `recipe_input` boundary
  account or `ErrBalanceOverflow` of a flow total, change nothing); a failure
  of any call made after a `Transform` has applied (`WearItems` of a tool the
  count showed present, a `Discard`) cannot be caused by input and is a
  violated invariant, so it panics with a message that contains the word
  "invariant" (architecture.md 8.1; the determinism guard of Stage 0 checks
  it, N-224). A returned refusal exists only for the refusals the spec makes
  into a reason (4.2).
- **A7.** The cost anchor table is dense, `region x commodity`, built into
  scratch the world holds, once per tick (PRO 9.3). The caller supplies, per
  region, the published price per commodity, the wage and the ascending list
  of recipes that "some agent of the region has on its role's list"; the
  effective factor of a recipe in a region is computed here from the table,
  the shock and the season.
- **A8.** Tests of the root file run from an internal test (package
  `economysim`) with small configuration literals, and one external test
  (package `economysim_test`) runs the same script of starts and completions
  over the packs `one` and `second`, with no code specific to either
  (architecture.md 9.4, FR-PRO-001, FR-PRO-028). The seam: an external test
  cannot call the unexported steps of `production.go`, and an internal test
  cannot import `config/` (which imports the root: an import cycle). So the
  steps are exposed to the external test by an `export_test.go` in package
  `economysim`, a `_test.go` file that is not part of the build and so adds no
  back door (architecture.md 10.2), as bare function variables; `one` is the
  Go literal `referenceConfig()` of `reference_test.go` and `second` is loaded
  through `config/`, as `fileagree_test.go` already does. Phases 6 to 8 use a real `internal/ledger` and its
  audit, not a fake (testing.md 6.1). The "ledger refuses" branch needs no
  fake: it is driven through the real ledger by not opening the `recipe_input`
  boundary account of one commodity, which `Transform` refuses with
  `ErrUnknownAccount`.
- **A9.** One PR per phase, in phase order, on a branch from `stage-1-1`.
  Phase 2 needs only the ledger as it is in the tree and may start now.
  Phases 3 and 4 need nothing of the other rows and may start while Phase 1 is
  in review; Phase 5 reads the role's recipe list and the recipe table's output
  counts, so it waits for plan 1-2 Phases 4 and 5 (or is built over ascending
  slices passed as parameters and tested with literals, and the plan 1-2
  tables are wired in Phase 7). Phase 6 needs Phases 3 to 5 and Phase 2 here,
  **plan 1-2 Phase 4** (it reads `*recipeRow` and the recipe's `Duration`, and
  a tool commodity's `Durability` from the canonical `Config`, A2; its tests
  build rows through `newRecipeTable` over a small `Config`), and the ledger's
  `Transform` and `WearItems`, which are in the tree, and, for the audit
  assertions of its invariant test, `Audit` (plan 1-1 Phase 6, in the tree
  since the audit fix of 2026-10-13); Phase 7 needs plan 1-2 Phase 4. The row shares
  `doc.go`, `event.go`, `doc/notes.md` and the roadmap with the other rows:
  each PR appends its lines and re-reads the shared file just before editing
  it.

## Phase 1: settle the owed spec lines and the notes

**Why first:** D52, D54, D57, D59 and D60 are rules the code implements, and a code
change that invented them would put a result-affecting choice where the
checkpoint and the golden files cannot see it. This phase changes specs and
notes, so it starts by reading `doc/notes.md` in full and quoting its tag, it
handles every open note tagged PRO and `ALL` (the table below), and
`doc/notes.md` gets a fresh tag (`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **PRO 19 and 5.1** (D50). The row "The job, the residues and the
   production phase" names `production.go` for the type, the rules and the
   steps, `agent.go` for the storage as AGT 3.1 gives it and `world.go` for
   the phase. Plan 1-2 Phase 1 edits other rows of the same table (struct
   names, the home of the tradable set); each edit is a targeted one after a
   re-read, and neither renames the other's rows.
2. **PRO 4.2** (D57). The defensive cap of a commodity found both in `Inputs`
   and in `Requires`, stated as defensive and with a pointer to 4.4.3 ("a tool
   never appears in `Inputs`"), which 10.2 does not enforce. FR-PRO-009 is
   reworded to say "the available stock over the input quantity plus 1 (one
   tool item per execution) for a commodity in both `Inputs` and `Requires`";
   its ID stays. The sentence must not say the combination is allowed, and
   must not cite 3.1.1, whose "both lists" are `Inputs` and `Outputs`.
3. **PRO 4.2 and 14.2** (D56). A line saying which reasons of the master
   order a slice start can return and that the rest arrive with their modules,
   in the same order. The master order, now a sentence of prose, becomes a
   table (position, reason, owner spec, "a slice start can return it") that a
   plain text parse can read, because `TestConstants_Values` reads closed
   sets from spec text by such a parse (N-224) and Phase 5's cross-read of
   PRO 4.2 has nothing to read otherwise.
4. **PRO 4.2 and 6.3** (D59). The rule that separates `zero_yield` from
   `no_site`, as D59 states it. 4.2 is reworded from "`no_site` (a `Sited`
   recipe with no positive factor)" to "`no_site` (a `Sited` recipe with no
   profile for the region)", and says `zero_yield` applies when the effective
   factor is 0 and the recipe is not refused `no_site`. 6.3 already says "a
   `Sited` recipe with no profile" and is left as it is, with the cross-
   reference to 4.2 added. FR-PRO-009 gains the sentence "the two are
   disjoint: `no_site` iff the recipe is `Sited` and has no profile for the
   region, otherwise `zero_yield` when the effective factor is 0 (a profile
   of 0, a seasonal profile with no entry for the season, or a shock,
   technology, trainee or abundance term of 0)"; its ID stays. TEC 5.1.2,
   `spec-infrastructure.md` 4.1.3, EMP 6.1.1 item 3 and FR-EMP-023 already
   say this and are **not** edited (checked 2026-10-13); PRO 12.1 and 12.2
   already give the arid region `no_site` and `deep_crater` `zero_yield`.
5. **PRO 5.3.1, 5.3.2, 7.2 and FR-PRO-012** (D54). The completion is one
   `Transform` of the outputs that fit followed by `Discard` of each discard
   output and of the part that `MaxQuantity` leaves no room for, not "one
   atomic transform step": 5.3.1 ("one ledger `Transform` creates the pending
   outputs"), 7.2 ("in one transform step") and FR-PRO-012 ("created in one
   ledger `Transform`") are reworded, FR-PRO-012 to "the pending outputs, up to
   the room, MUST be created in one ledger `Transform` ... and the rest MUST
   be disposed by the ledger's create-and-dispose operation", and FR-PRO-029
   and FR-PRO-030 say so, all IDs kept. Two other specs say what D54 shows is
   not possible and are reworded in the same change, each by a targeted edit
   after a re-read (the sentence is about the one step; nothing else of them
   changes): `spec-storage.md` 5.3.2 ("is created and moved to `disposal` in
   one `Transform` step at the completion") and 5.3.4 ("created and disposed
   of in the same step", twice, and "creates the pending output from the
   pledge"), which become "created and disposed at the completion by the
   ledger's create-and-dispose operation" and keep the pledge and the room
   rule; and the P2.1 row of `spec-world.md` 6.4 ("one `Transform` creates
   the pending outputs from `recipe_output`, the part above `MaxQuantity`
   disposed"), which becomes "one `Transform` creates the outputs that fit,
   and the create-and-dispose operation disposes the part above
   `MaxQuantity`". The ledger side (LED 7.2.1, 8.2.1 and two new requirement
   IDs) is Phase 2's, after a re-read, because the ledger workflow also edits
   `spec-ledger.md`.
6. **PRO 14.1, FR-PRO-029 and FR-PRO-030, IND 7.2.4** (D60). One sentence
   each fixing the two slices: `Outputs` is the gross quantity created per
   output item (discard outputs and the overflow part included), `Disposed` is
   the part never held (the discard outputs and the part `MaxQuantity` left no
   room for), rows above 0 only, in output order. IND 7.2.4 already values
   `Outputs` gross of `Disposed` and gains a pointer to PRO 14.1 so the three
   documents say one thing. The doc comment of `RecipeExecuted.Disposed` in
   `event.go` ("the units of the outputs that were discarded") is the one
   place that disagrees; it is Go, so it is corrected by Phase 6's PR, which
   writes the builders, and not here.
6a. **PRO 5.1.1, 14.1, 15, FR-PRO-041 and CHK 4.2 tag 6** (D67). The job gains
   `Yield` (the effective factor fixed at the start); `StartTick` is derived as
   the completing tick minus `Duration` plus 1; PRO 15 and CHK tag 6 list
   `Yield` after `Principal`. The lines and note **N-237** (PRO, CHK, AGT)
   were written by the audit fix of 2026-10-13 after a re-read of each file;
   this phase checks they are still there and that no later edit of the PRO 14.1
   row (item 6) dropped the two sentences below the table. AGT 3.1 and 3.2
   hold the job by reference and are not edited.
7. **PRO 9.4.** The surface of `internal/production` beyond `UnitCost`:
   the names are the implementer's (A1), but the list of capabilities, the
   rule that the root builds the recipe table once, and the rounding of each
   (truncate toward zero except `UnitCost` and the wear cost, up) are one
   paragraph, so that the rounding is a rule and not a default found in code.
   The sentence "the root's `UnitCost(region, commodity)` ... are root-level
   wrappers" is reworded to say that the root read is the unexported
   `unitCost(region, commodity)` (a read of the anchor table), since
   architecture.md 2.8 keeps what is not API out of the exported surface and
   the beliefs and the strategy view reach it inside the root.
8. **The roadmap.** Link this plan from Stage 1 prose, set the row to
   **in progress** when Phase 2 starts, and add the description "the shock
   slot and the cost-anchor table" to the row.
9. **Notes.** The disposition below; a new note (**N-232**, reserved for this
   plan by the cross-plan review), tagged PRO, AGT and CODE, that
   the job and residues live on the agent and what the agent row must call
   (D50); a new note (**N-233**, reserved) tagged PRO and LED (the
   `Discard` operation and `Holding`, D54 and D58). Any further gap found is a note in the same
   change. Each new note takes its ID from a re-read of the index at that time
   (the reservations above hold only if nobody has taken them) and the file
   gets a fresh tag.
10. **`doc/documentation_index.md`**: the row of this plan; corrected if the
   phases change.

### Disposition of the open notes

Every open note tagged PRO or `ALL` in the index of `doc/notes.md` (the LED, IND
and CHK notes are the second table below; read in full on 2026-10-13, tag `6cc73d3abf2fd311fb034a21` at the last re-read; the
tag moves with every edit of the file, and Phase 1 re-reads the file and
quotes the tag it finds), each applied, rejected or left open.

| Note | Tag | Disposition |
|------|-----|-------------|
| N-030 Five regions with different climates | PRO | **Left open, with what this row learned.** PRO's part (the yield profile format, `Sited`, 0 for cannot run) was applied in spec; this row builds the lookup and the check that a recipe runs in a new region by a table change alone (FR-PRO-028). The values are RES's and REF's, and the three-region scenario is REF's |
| N-039 A hard cost floor | PRO | **Left open.** The cost floor is BEL's rule and STB's scenario S07. This row makes `UnitCost` count labour (9.1), which is what moves the floor, and passes `abundance` 10 000 until RES |
| N-043 Nothing anchors the price level | PRO | **Left open.** Nothing in this row prices or halts; the halt is the strategy and world rows |
| N-090 What MIG and others must do with a relocation | PRO | **Left open.** The abandon function takes a reason; `region_changed` is appended with MIG (D56). A job of a self-carried trip cannot start (TRN) |
| N-118 One job per agent limits the subsistence floor | PRO | **Left open** for the first scenario that shows it. The job type is one slot (5.1.2) |
| N-130 What IND cannot measure exactly | PRO | **Left open.** Item (6), wear units on `RecipeExecuted`, is an ask not adopted; no event field or snapshot field is added here |
| N-137 Room is committed at the decision | PRO | **Left open** for STO. The count takes a module limit pair at the place of `no_room` (D52), and the completion never fails for room |
| N-142 A job start and a dispatch leave the upkeep room free | PRO | **Left open** for STO; the room limit is a pair the module appends |
| N-191 A right is read at a start, an order and a place | PRO | **Left open** for ORG: `not_permitted` is a pair appended after `not_on_role` and before `zero_yield`; a continuation, a completion and a holding read none |
| N-200 A gain is not a recipe | PRO | **Left open.** No unlock clause or refusal reason is added; the technology factor is a parameter of the effective factor, 10 000 in the slice |
| N-016 Keep the parameter registry true | ALL | **Applied.** This row adds no configuration field, default or bound, so the registry test and its `notBuilt` list do not change; the PR says so |
| N-017 Words and names | ALL | **Applied.** The terms of this row (job, executor, principal, residue, effective factor, site factor, yield shock, pending output, cost anchor, unit cost) are added to `doc.go` before the code uses them; setting-neutral names throughout |
| N-027 No default macro constant is stable yet | ALL | **Applied (standing).** The row sets no default and claims no stability |
| N-103 Running several spec writers in parallel | ALL | **Applied in its lesson.** Shared files are edited with targeted edits after a re-read (A9) |
| N-126 A new event or `Warn` adds a row | ALL | **Applied: nothing to add.** `RecipeExecuted`, `RecipeAbandoned` and the two production `Warn`/`Debug` lines have rows since Stage 0; this row declares no event and emits none (D51) |
| N-128 A hosted per-agent field is saved once | ALL | **Applied.** The job and residues are saved in section 6, not in the agent record (CHK 4.2 tag 6), and the plan says so for the agent row (D50) |
| N-163 The word "treasury" | ALL | **Applied: not used** |
| N-235 One combined architecture.md edit for the fan-in and admission touch points | ALL | **Left open** (owner: the first Stage 2 module PR, ruled by the coordinator). Nothing in this row adds a fan-in or an admission touch point: the tradable set is plan 1-2's and the admission table plan 1-3's, and the limit pairs of D52 are a parameter of a function, not a touch point. If the combined edit lands before Phase 1 runs, the disposition changes to "applied: nothing to add" |
| N-217 D5 binds every module's change | ALL | **Applied.** No configuration field, event field or snapshot field is appended; the module parameters are neutral values; `testdata/golden/off/` and the pinned hash do not move (the Phase 8 gate checks the hash) |

AGENTS.md asks for every open note of the code of each spec that is changed,
and this row changes six specs: PRO (Phase 1), IND 7.2.4 (Phase 1 item 6),
CHK 4.2 tag 6 (Phase 1 item 6a, D67; already written by the audit fix and only
checked here), STO 5.3.2 and 5.3.4 and WLD 6.4 (P2.1 row) (Phase 1 item 5, one
sentence each about the one-step disposal) and LED (Phase 2: 8.2, 5.2, 7.2.1,
8.2.1 and two FR-LED IDs). The table above is PRO and `ALL`; the next is LED,
IND and CHK (a note tagged several of them is in one row); the third is STO and
WLD. Phase 2 re-reads `doc/notes.md` in full before
its edit of `spec-ledger.md`, quotes the tag it finds and gives the file a
fresh tag with N-233.

| Note | Tag | Disposition |
|------|-----|-------------|
| N-021 Anything that creates or destroys money or goods | LED | **Left open, with what this row learned.** `Discard` adds no reason, no faucet, no sink and no boundary account: it moves the two boundary accounts of the built-in rows `recipe_output` and `disposal` that every world already has (LED 7.2.2). The rest of the note (a module's rows) stays owed to the modules. N-233 says so |
| N-125 A snapshot is a function of saved state | CHK, LED | **Left open (nothing to apply here).** `Discard` and `Holding` add no saved state and no snapshot field: `Discard` changes the flow totals, which are cleared at P8.6 and rebuilt after a restore (LED 9.1, 11), and `Holding` is a read. The one saved field this row adds is the job's `Yield` in tag 6 (D67, N-237), a function of saved state |
| N-130 What IND cannot measure exactly | IND, LED (and PRO, above) | **Left open.** Item (2), wear units by commodity, is unchanged: `Discard` counts no wear and `WearItems` is plan 1-1's. Item (6), wear units on `RecipeExecuted`, is not adopted. IND 7.2.4 gains only the pointer to PRO 14.1 for `Outputs` and `Disposed` (D60), so value added still reads `Outputs` gross of `Disposed` |
| N-159 Region-attributed treasury and fiscal flows need a reconciling item | IND, LED | **Left open (not this row's).** No treasury is involved. `Discard`'s two flows are attributed to the account's region like every other goods flow (FR-LED-053), which is the rule N-159 asks readers not to assume otherwise about |
| N-178 A small line's loss is a step function | IND | **Not applicable; left open** for RSK and IND. This row has no loss and no line |
| N-201 R9's volume clause is on units | IND | **Left open.** `Outputs` and `Disposed` are units of quantity; the D60 sentence changes no clause on value or volume of IND 7.2.4 beyond the pointer |
| N-206 The engine cannot call an indicator | IND | **Left open.** Nothing in `production.go` or `internal/production` imports `indicator/`; the IND edit is a cross-reference in prose |
| N-208 The gauge set and the IND presets must agree | IND | **Not applicable; left open** for PLC and IND. No gauge or preset is added |
| N-225 The ledger's reason constants are internal numbers | LED, CHK | **Left open for CHK (Stage 2).** `Discard` names `recipe_output` and `disposal` by the ledger's constants, and the root names the five production reasons of D56 by its own `Reason` constants (a refusal is a returned value and is never saved). Tag 6 saves no reason. Moved here from the CODE paragraph below |
| N-226 The creation position of the boundary accounts of the other module reasons | WLD, LED | **Left open (not this row's).** `Discard` uses only boundary accounts of the built-in group (`recipe_output` and `disposal`, created at construction in the order of LED 3.4.2); it adds no account and no module reason |
| N-129 A stateful extension is one instance per world | CHK | **Not applicable.** Production has no extension state: the job, residues and shocks are plain core state (D50, D53) |
| N-202 A module that keeps a key re-derives it at a reseed | CHK | **Not applicable.** Production keeps no key and draws nothing (FR-PRO-026) |
| N-204 A slot has two writers: the consumer and a policy actor | CHK | **Left open, with what this row learned.** The yield shock is a story shock and PLC offers no lever for it (`spec-policy.md` 6.3.3), so it has one writer; D53's setter is unexported and the public call is INT's. The sparse shock list is written in tag 6 |
| N-214 Keyed random streams (ADR 0008): the edits owed | CHK | **Left open.** The stale-marked passages of CHK (tag 1, 5.4, 6.2 step 8, 8.4, 17.2) are not touched; the tag 6 line is an addition after `Principal` and does not read them. The residue draw is the creation draw of the world's site (FR-PRO-026), not a draw site of this row |
| N-215 The reseed rule is the identity for seed 0 and salt 0 | CHK | **Not applicable.** No reseed rule is read or changed |
| N-128 A hosted per-agent field is saved once | CHK | **Applied** (as in the ALL table: the job is saved in tag 6, in agent order, not in the agent record) |
| N-237 A job stores its effective factor | CHK, PRO | **Applied in spec (D67); left open** for the agent row, the checkpoint row and the warm-start job, as above |

| Note | Tag | Disposition |
|------|-----|-------------|
| N-155 Reserved hooks of the settlement class | STO | **Left open (not this row's).** The edit to STO is one sentence about how an output is disposed; no class flag, limit or hook of 6.5 is read or added |
| N-160 Readers keyed by jurisdiction use one combining rule | STO | **Left open (not this row's).** No table keyed by jurisdiction is read or added; the module limit pair of D52 is a parameter of a function |
| N-166 Collateral is not in version 1 | STO | **Left open (not this row's).** A pledge of room is not touched; the STO sentence of 5.3.4 keeps the pledge rule and rewords only the disposal step |
| N-181 Not exposed in version 1: a household's base room, a migration journey, money | STO | **Left open (not this row's).** Risk exposure is RSK's; the STO edit changes no exposure |
| N-186 A treasury's goods have no room limit | STO | **Left open (not this row's).** The completion never fails for room (FR-PRO-029); who is limited (STO 3.3.3) is unchanged |
| N-194 Reserved, not offered: a charter over carriage, a store or a facility; a wage control; a named holder; a call that admits or expels | STO | **Left open (not this row's).** No charter or lever is added |
| N-205 A new slot is a candidate lever kind; the kinds `rate`, `rent` and `legal` are reserved | STO | **Left open (not this row's).** The job's shock slot is a story shock and not a lever (as N-204 above); the row adds no slot an authority would hold |
| N-221 STO confirms the file layout of its base room and role rows | STO | **Left open (not this row's).** The edit touches no file layout or configuration field; it is CFG's and STO's own |
| N-124 A paired comparison is exact only while aligned | WLD | **Left open (not this row's).** The row draws no random number (FR-PRO-026) and moves no draw count, so it cannot misalign a paired run |
| N-144 A busy employer cannot post vacancies | WLD | **Left open (not this row's).** The P2.1 row of 6.4 is reworded for the disposal step only; the call order of `Work` and `Trade` is unchanged |
| N-154 A count of households gates a structure: the exceptions of `spec-world.md` 9.4.1 | WLD | **Left open (not this row's).** Nothing in this row makes a rate or a gate a function of a count of households |
| N-216 A site constant is never a small integer | WLD, CODE | **Left open (not this row's).** No draw site and no site constant is added (see the CODE paragraph below) |

N-125, N-204, N-214 and N-226 also carry WLD (or CHK, LED) and are in the
second table; N-191 (STO) is in the first.

N-237 (PRO, CHK, AGT) was raised by this plan's audit fix and is applied in
the specs (D67); it stays open for the agent row (stores the field), the
checkpoint row (writes tag 6) and the warm-start job (sets it).

N-089, N-127, N-198, N-215, N-216, N-218, N-222 and N-228 (CODE) were read and
do not apply to this row: they concern the model scripts of the demand,
spawner and reference-chain specs (N-089), event classes at the observer
(N-127), the drawdown cost (N-198), the reseed rule and its cost and the
site-constant rule at draw sites (N-215, N-216, N-222), the Stage 0 field set
and hash rule (N-218) and a module's tradable items (N-228). This row draws no
random number (FR-PRO-026), adds no configuration field, no draw site, no
event class and no tradable item.

N-019, N-214 and N-224 (CODE, read for what the first change in the
area must do): the glossary, import and determinism guards run over the new
files unchanged and `TestConstants_Values` gets the new reason constants in its
table with the cross-read of PRO 4.2 (Phase 5); no draw site is added (N-214:
the residue draw is the world's creation draw); the ledger's reason constants
are internal numbers, so this row names reasons only by the ledger's constants
and `ledger.go`'s mapping (N-225, in the LED and CHK table above).

**Tests:** none; documents only.

**Observability and rollback:** none, because no code changes. A wrong edit is
reverted like any document change; nothing depends on it until Phase 5.

**Acceptance:** PRO 4.2, 5.3, 6.3, 7.2, 9.4, 14.1, 19 and FR-PRO-009, 012,
029 and 030, IND 7.2.4, `spec-storage.md` 5.3.2 and 5.3.4 and the P2.1 row of
`spec-world.md` 6.4 carry the lines above; the notes carry a fresh tag
and the new notes; the roadmap links this plan; the disposition above is
complete for every open note of PRO, IND, CHK, LED, STO, WLD and `ALL`.

## Phase 2: ledger additions (`Discard` and `Holding`)

**Why here:** the completion of a job cannot be written against the ledger as
it is. A discard output and the part of an output above `MaxQuantity` must be
"created and disposed in the same step" (PRO 5.3.1, 5.3.2, 7.2; LED 7.2.1), and
completion must never fail for room (FR-PRO-029), but a move of the created
units into an account that already holds `MaxQuantity` of the commodity is
refused by `ErrBalanceOverflow`. The steps also need to read how many units of
a commodity an account holds and how many are earmarked, and the ledger exposes
no such read. Both were questions to the owner (O2, O4 as first written) and are
**answered by the coordinator, 2026-10-13**: build them here, additively, and do
not change plan 1-1 or its Phase 4 (D54, D58).

**What the tree has (read 2026-10-13).** `internal/ledger` has `New`, `Open`,
`OpenBoundary`, `Boundary(reason, commodity)`, `Close`, `Account(id) View`
(money only), `Transfer`, `TransferReserved`, `ReserveMoney`, `ReleaseMoney`,
`Move`, `MoveReserved`, `MoveLot`, `MoveLotReserved`, `MoveStated`,
`ReserveGoods`, `ReleaseGoods`, `ReserveLot`, `ReleaseLot`, `AppendLots`,
`Tick`, `AdvanceTick`, `InConstruction`, `EndConstruction`, `ClearTotals`,
`AppendFlows`, the reason constants `RecipeInput`, `RecipeOutput`, `Disposal`
and `Wear`, and the sentinels of plan 1-1 Phases 2 and 3. Since the first
reading of this plan it also has the operations of plan 1-1 Phases 4 and 5 in
`transform.go`, `ageing.go` and `claims.go`: `Transform(id int, inputs, outputs
[]Line) error` (with `Line{Commodity int; Quantity int64}`), `WearItems(id,
commodity int, n, units int64) error`, `Wear`, `WearUnits(commodity int, bucket
bool)`, `Expiring`, `Age`, `EndTick`, and the claim operations, and the audit,
`Audit() error` in `audit.go` (plan 1-1 Phase 6). It has no read of one
account's held and reserved units of a commodity (`account.held` and
`earmarkedOf` are unexported; `Account` returns the money fields only and
`BalanceSheet` a value of goods, not units). Plan 1-3 assumes the same read
(its contract, fallback `AppendLots`), so `Holding` serves both rows.

**Start condition.** `Discard` is a sibling of `Transform` and not a change to
it, so the phase may start now, on the tree as it is (A9). It adds new files
only (`internal/ledger/discard.go` and `discard_test.go`) and edits no existing
file of the package, so it cannot collide with the workflow's edits. The audit
exists, so the tests call `Audit` (below). Before writing it, re-read `lots.go`,
`transfer.go`, `transform.go` and `ledger.go` for the unexported helpers it
uses (`at`, `open`, `slotOf`, `flows`, `boundaryAt`, `creationKey`; the
exported `Boundary`) in case the workflow renamed them.

Deliverables:

1. **The spec clauses and requirements.** `spec-ledger.md` is re-read just
   before each edit because the ledger workflow also edits it. FR-LED-039
   (about a `Transform` with an empty side) is left unchanged, since a
   requirement ID is one behaviour (testing.md 4.1, 4.4). Two new rows are
   added to the functional table of section 13, each with the next free ID
   at the time of the edit (FR-LED-065 is the last today, so 066 and 067),
   and the tests of this phase and the Testing table cite those IDs:
   - **Discard.** A clause in 8.2 after 8.2.1 and a requirement: the
     operation `Discard(account, commodity, quantity)` creates the quantity
     under `recipe_output` and sinks it under `disposal` in one step; the
     account's lots, holding and earmarks do not change; both boundary
     accounts and both flow totals move by the quantity, so the total over
     all accounts stays zero; no room check applies, since nothing is held;
     it is refused only for an invalid argument or an `int64` overflow of a
     boundary or a flow total, and then changes nothing. The clause says how
     it fits FR-LED-001 ("a balance or a quantity MUST change only through a
     transfer between two accounts with a reason"): a `Discard` is the
     composition of a `Move` of the quantity in from `recipe_output` and a
     `Move` of it out to `disposal`, collapsed because the account is never
     credited; both are transfers with a reason, both flows are attributed to
     the account's region (LED 2.1, 7.3), and the account's own balance does
     not change. FR-LED-001 is cited by the test.
   - **Holding.** A clause in 5.2 (or beside the read-only query `Expiring`
     of 5.3.3, wherever the reads of one account's goods are listed when the
     edit is made) and a requirement: `Holding(account, commodity)` returns
     the units of the commodity the account holds, earmarked ones included,
     and how many of them are earmarked; it changes nothing, an unknown
     account or commodity is `ErrUnknownAccount` or `ErrUnknownCommodity`,
     a closed account or a boundary returns 0 and 0. FR-LED-017 (a
     reservation never exceeds what is available) is a different behaviour
     and is not cited for it.
   - **Reworded, not appended.** LED 7.2.1 says a discard output is created
     and moved out "inside the one atomic completion step of the job", and
     PRO 5.3.1 and FR-PRO-012 (Phase 1) are reworded the same way: since
     the completion is a `Transform` followed by `Discard` calls, the
     sentence becomes "by the completion of the job: a `Transform` of the
     outputs that fit and a `Discard` of each part that is never held". LED
     8.2.1 ("a completion that takes nothing and creates the pending
     outputs") gains "(for the part that is held)".
2. **`Discard`.** A sibling operation, not a form of `Transform`. Reasons:
   `Transform` is plan 1-1's file and its output side checks the room on
   `held` and creates every line it accepts, so a form of it would edit
   `transform.go`; a discard has no lots and no account change, which a
   `Transform` line would have to special-case; and the completion needs it
   for one output line at a time, after the one `Transform` of the outputs
   that fit. The checks run in a fixed
   order and the first that holds is returned, all before any write
   (LED 2.6): `ErrInvalidAmount` (quantity below 1), `ErrUnknownAccount`,
   `ErrClosedAccount`, `ErrUnknownCommodity`, `ErrReasonMismatch` (a boundary
   or a kind that `recipe_output` and `disposal` do not both run to or from;
   the account must be of a kind the two rows allow, an agent in the slice),
   `ErrUnknownAccount` again if the `recipe_output` or the `disposal` boundary
   account of the commodity was never opened (`boundaryAt` is nil; it is the
   refusal `Transform` gives for a never-opened boundary, and without the
   check the implementation dereferences nil), `ErrBalanceOverflow` (the
   `recipe_output` boundary or the `disposal` boundary would leave `int64`, or
   either flow total would). The flow is
   attributed to the account's region and, in the construction phase, goes to
   the construction bucket, as every other goods flow does (FR-LED-045, 053).
3. **`Holding(account, commodity) (held, reserved int64, err error)`.** A read
   of the account's units of a commodity and how many are earmarked, so
   available is `held - reserved`, zero allocation. An unknown account is
   `ErrUnknownAccount` and an unknown commodity `ErrUnknownCommodity`; a closed
   account or a boundary holds no lots and returns 0 and 0, as `AppendLots`
   returns none for them.
4. `doc.go` of the package: no new term. No root file changes.
5. **Notes (before deliverable 1).** This phase edits `spec-ledger.md`, so it
   starts as any spec change does (`AGENTS.md`): read `doc/notes.md` in full,
   quote its tag in the first message of the PR, and for every open note
   tagged LED or `ALL` say applied, rejected or left open. The dispositions
   of N-021, N-125, N-130, N-159, N-225 and N-226 are the LED and CHK table
   of Phase 1; the phase re-reads them against the file as it is then and
   corrects a row that has gone stale (a note closed, a new one tagged LED).
   It then adds **N-233** (tagged PRO and LED; the ID is taken from a re-read
   of the index): `Discard` and `Holding` exist, what each is for, that
   `Discard` adds no reason and no account (so N-021 and N-226 are
   untouched), and that a later row that creates and sinks in one step uses
   it and does not widen `Transform`. The file gets a fresh tag in the same
   change.

**Tests:** unit, in `internal/ledger`, external package for the public
behaviour with the fixture of the package; 90% statement coverage of the
package is kept (read from `go tool cover -func`).

- `Discard`, the ordinary case with arithmetic in a comment (testing.md 5.5):
  an agent holding 7 units of a commodity, `Discard` of 5: the lots are
  `reflect.DeepEqual` to a copy taken by `AppendLots` before, the
  `recipe_output` boundary is -5 and the `disposal` boundary +5, the sum of
  all balances of the commodity is 0, and `AppendFlows` shows 5 under each of
  the two reasons in the account's region (FR-LED-001, FR-LED-037 for the
  `disposal` sink, and the new Discard requirement of deliverable 1).
- The bound: an account holding exactly `MaxQuantity` of the commodity accepts
  `Discard` of 1 and of `MaxQuantity`, its holding unchanged, where a `Move`
  of the same quantity from `recipe_output` is refused with
  `ErrBalanceOverflow`; this is the case the phase exists for (FR-LED-006,
  the new Discard requirement; FR-PRO-029 at the ledger level).
- Atomicity: one case per refusal that breaks only that rule, asserted with
  `errors.Is`, each leaving the ledger `reflect.DeepEqual` to a copy
  (FR-LED-005, 007); including `ErrUnknownAccount` for a commodity whose
  `recipe_output` boundary account was never opened and, separately, one
  whose `disposal` boundary was never opened (the ledger of the case opens
  the other), as A8 does for `Transform`'s `recipe_input`; an `int64` overflow of the `disposal` boundary balance
  reached from an internal test refuses before the first write, so the
  `recipe_output` boundary is unchanged too.
- Earmarks: units earmarked for an order stay earmarked and untouched; a
  `Discard` larger than the available stock succeeds, because nothing is taken
  (stated, not assumed).
- Construction: a `Discard` in the construction phase lands in the bucket and
  not in the tick's totals; after `EndConstruction` it lands in the totals.
- `Holding`: held and reserved after a `Move` in, a `ReserveGoods`, a marked
  move and a release; an unknown account, a closed account and a boundary
  (the new Holding requirement; a reservation made through `ReserveGoods` is
  read back as `reserved`, which is also FR-LED-017's reservation seen from
  the read).
- Audit: a scripted run that interleaves `Move`, `Discard` and the
  reservations calls `Audit` after each operation and it passes (FR-LED-035,
  036).
- Allocation tests with `testing.AllocsPerRun` (testing.md 8.2): `Discard` and
  `Holding` allocate nothing (NFR-LED-001); a benchmark with `b.ReportAllocs()`
  and `b.Loop()` for each (NFR-LED-003).

**Observability and rollback:** none, because the package logs nothing, emits
nothing and holds no global state, and nothing calls the new functions until
Phase 6 and plan 1-3. Rollback is reverting the PR: the files are new, so the
revert leaves plan 1-1's work untouched.

**Acceptance:** `gofmt -l .` prints nothing; `go vet`, `staticcheck` and
`go test -race ./...` pass with plan 1-1's own tests unchanged; the import and
determinism guards pass; `internal/ledger` is at 90% statement coverage or
more; LED 8.2 and 5.2 carry the two clauses, the two new requirement IDs are
in the table of section 13, LED 7.2.1 and 8.2.1 are reworded, and
`doc/notes.md` carries N-233, the dispositions of the LED notes and a fresh
tag.

## Phase 3: `internal/production`, the yield and output arithmetic

**Why here:** nothing else of the row can be tested without the effective
factor and the carried remainder, and they need no ledger and no configuration.

- The package `internal/production` and its doc comment.
- **Site factor** (PRO 6.2, FR-PRO-020): from `(has profile, sited, has all,
  all, seasons sorted by index, season)` to a factor in `[0, MaxYieldBP]`: the
  season entry, else `All`, else 0 when the profile has season entries, else
  10 000 for a recipe not `Sited` and 0 for one that is. The season -1 never
  matches an entry. The caller passes the bound only through a test; the
  function does not read the root's constants.
- **Effective factor** (6.3, FR-PRO-022): `site * shock / 10 000`, then
  `* tech / 10 000`, then `* trainee / 10 000`, then `* abundance / 10 000`,
  each truncated, through `internal/checked`, `ok` false on overflow.
- **Output with a residue** (6.6, FR-PRO-024): for `k`, `q`, `E` and a residue
  in `[0, 10 000)` it returns the produced units and the new residue. It
  computes `k * q * E` with the 128-bit helper and divides, adding the
  residue after the division of the product (`Q * 10 000 + R + residue`, so
  the new residue is `(R + residue) mod 10 000` and a carry of at most 1 is
  added to `Q`), which equals `(k q E + residue) / 10 000` and never needs a
  sum above `int64`. `ok` false when `produced` would exceed `int64`.
- **Rescale** (5.4, FR-PRO-019): for a weight fall to `w'`, `Executions'` is
  `min(k, w')` and each pending quantity is `floor(p * Executions' /
  Executions)`; a weight rise changes nothing.
- `doc.go`: the glossary gains job, executor, principal, residue, site factor
  and effective factor.

**Tests:** unit, in the external package `production_test`; the package is
pure.

- Site factor: one case per rule of 6.2 (a season entry; `All`; a seasonal
  profile that lists no matching season is 0; no profile for a recipe not
  `Sited` is 10 000; no profile for a `Sited` recipe is 0; the season -1 with
  `All`; a profile with only `All = 0`), each asserted for the value
  (FR-PRO-020). The two climates of PRO 12.1 (north spring 10 000 with summer
  0; south spring 12 000 and autumn 7 000) as a table whose expected values
  come from the spec (testing.md 5.5).
- Effective factor: the order and the truncation, with the arithmetic in a
  comment: `site 8 500, shock 10 000, tech 10 000, trainee 5 000, abundance
  10 000` gives `8 500 * 5 000 / 10 000 = 4 250` (FR-PRO-022, 034 in part); a
  zero in any term gives 0; a shock of 3 000 on 10 000 gives 3 000. The
  largest validated inputs (`MaxYieldBP` in all four terms) do not overflow:
  each step is at most `1e5 * 1e5 / 1e4`, the chain is 1e6, 1e7, 1e8, 1e9 and
  the result 1e9 fits `int64` (PRO 10.3 bounds it by `MaxQuantity`), so that
  case asserts the value `1_000_000_000` and `ok` true. The `ok` false case
  needs operands outside validation, reachable by a direct call:
  `site = math.MaxInt64` and `shock = 20 000` gives `2 * MaxInt64`, which does
  not fit, so `ok` is false and nothing wraps (the `MulDiv` product is held in
  128 bits and the quotient is the one that overflows). The `ok` result is a
  defence for such a call, kept because a silent wrap is a wrong output
  (architecture.md 3.5).
- Output: the worked example of PRO 6.6, one row per step with its arithmetic
  (`k=3, q=1, E=8 500`, residue 2 000 gives 2 and 7 500; then 3 and 3 000;
  then 2 and 8 500; then 3 and 4 000) (FR-PRO-024). `E = 10 000` leaves the
  residue unchanged and gives `k * q` (6.6.1). Boundaries (testing.md 3.8):
  `k = 1`, residue 9 999 with a product that is a multiple of 10 000, the
  largest `k * q * E` that fits `MaxQuantity` after 10.3, one above refused.
- Long-run property (`Guard:`, a loop over a fixed seed range with the
  failing seed in the message, testing.md 6.2): over 1 000 jobs at a constant
  `E` and a seeded start residue the total produced differs from
  `k * q * E * jobs / 10 000` by less than one unit, at weight 1 and at weight
  1 000 (FR-PRO-025).
- Rescale: a fall from 10 to 4 executions scales `[7, 12, 1]` to
  `[2, 4, 0]` (the arithmetic in a comment); a fall to the same weight and a
  rise leave the job unchanged; `Executions'` never exceeds `k` (FR-PRO-019).
- `FuzzSite` and `FuzzEffective` (testing.md 7.4, NFR-PRO-004): a byte string
  decoded into a profile and a season, asserting no panic and a site factor in
  `[0, MaxYieldBP]` for a valid profile; seed corpus with `f.Add`, corpus
  directory `internal/production/testdata/fuzz/`.
- Allocation tests with `testing.AllocsPerRun` (testing.md 8.2): every
  function allocates nothing (NFR-PRO-001 in part).

**Observability and rollback:** none, because the package logs nothing, emits
nothing and holds no global state; nothing imports it yet. Rollback is
reverting the PR.

**Acceptance:** `gofmt -l .` prints nothing; `go vet`, `staticcheck` and
`go test -race ./...` pass; the import guard and the determinism guard of
Stage 0 pass over the new package unchanged; `internal/production` is at 90%
statement coverage or more, read from `go tool cover -func=coverage.out`.

## Phase 4: `internal/production`, the count, `UnitCost` and the anchor

**Why here:** the count and the cost are the other two pure halves of PRO, and
the root steps of Phases 6 and 7 are thin wrappers over them.

- **The execution count** (4.2, D52, D57; FR-PRO-009): `weight`, an optional
  strategy maximum (0 means none), and an ordered slice of `(cap, reason)`
  pairs; the result is the least of the weight, the maximum and every cap,
  and, when it is 0, the reason of the first pair whose cap is 0. A pair with
  no limit has a cap of `math.MaxInt64`. A helper turns "available stock and a
  per-execution need" into a cap by integer division (zero need is no limit).
- **The yield pair** (D59): from `(sited, hasProfile, effective)`, whether the
  recipe is `Sited`, whether a profile row exists for (region, recipe) and the
  effective factor of 6.3, it returns the cap (0 when `effective` is 0, none
  otherwise) and whether the reason is `no_site` (`sited` and no profile) or
  `zero_yield` (every other case of `effective` 0). The package holds
  no reason constant (it does not import the root); the root maps the flag to
  `ReasonNoSite` or `ReasonZeroYield` and places the pair first in the list.
- **`UnitCost`** exactly as PRO 9.4 gives it (`Term`, `Wear`, `UnitCost`;
  FR-PRO-035, 036): `ceil((sum quantity*price + sum ceil(WearPerUse*price /
  Durability) + duration*wage + rent) * share / (outQty * yieldBP))`, 0 when
  any input or tool price is 0, the wage is 0 or less, the factor is 0, the
  share is 0 or any step overflows. No side effect, no allocation.
- **The anchor of a commodity in a region** (9.3, FR-PRO-037): over the recipes
  that make it, passed as a caller-owned `[]Recipe` (A2: a plain argument
  struct, filled by the caller inline from plan 1-2's rows, with no table
  type here) and the ascending makers of the commodity, and can run there (a
  runnable flag per recipe from the caller
  and `E` above 0) the least non-zero `UnitCost`, 0 when none; a region whose
  wage is 0 gives 0 for every commodity (FR-PRO-049). The function writes one
  region's row of a caller-owned slice indexed by commodity.
- The wear cost `ceil(WearPerUse * price / Durability)` is the same formula as
  welfare's wear cost of an execution (WEL 2.1.4, plan 1-6 Provides). Two
  internal packages do not import each other, so it is written once here and
  once there; the root test that asserts they agree over the packs' tools is
  in [plan-phase-1-6.md](plan-phase-1-6.md) Phase 4 (welfare is built after this
  row, so the test cannot be written here); until then the worked example of
  PRO 9.2 pins this copy.
- `doc.go`: the glossary gains cost anchor and unit cost.

**Tests:** unit.

- The count: the least of the weight, the maximum, the input stocks and the
  tool items, with the arithmetic in a comment; a partial `k` below the
  weight is not an error; `k = 0` returns the first applicable reason of the
  list (`no_tool` before `no_input` when both are 0; `zero_yield` before
  both); a zero maximum means none, a negative maximum is treated as none and
  does not wrap, and a maximum above the weight changes nothing (the Security
  review promises all three); a stock of 0, an input quantity larger
  than the stock, weight 1 (FR-PRO-009, 014 in part, 3.8 of testing.md).
  D57: stock 7, input quantity 2 and one tool item per execution gives a cap
  of `7 / (2 + 1) = 2`, attributed to `no_tool`; a `WearPerUse` of 5 changes
  nothing (it is durability, not items).
- The cap helper (stock and per-execution need to a cap by integer division;
  FR-PRO-009), one case each, the arithmetic in a comment: stock 7, need 2
  gives `7 / 2 = 3`; stock 6, need 2 gives 3 (an exact division loses
  nothing); stock 5, need 7 (a need larger than the stock) gives 0, the case
  that makes `k = 0` and reports the reason; stock 0, need 2 gives 0; need 0
  (zero means no limit) gives `math.MaxInt64` whatever the stock, including
  stock 0, so an input of quantity 0 or a tool of 0 items per execution is
  never the reason of a `k = 0`. In the count, a list of a zero-need pair
  (no limit) followed by a pair with cap 2, at weight 5, gives 2 with the
  second pair's reason: the pair with no limit is ignored, not read as 0.
- The yield pair, one case per combination of D59 and each breaking only its
  rule, asserted by cap and flag, the input being (`sited`, `hasProfile`,
  `effective`): `Sited`, profile, effective 8 500: no limit; `Sited`, no
  profile (effective 0): `no_site`; `Sited`, no profile and a shock of 0:
  `no_site` (the profile, not the factor, decides); `Sited`, profile `all =
  0` (effective 0, the `deep_crater` case of PRO 12.2): `zero_yield`; `Sited`,
  a seasonal profile with no entry for the season (effective 0):
  `zero_yield`; `Sited`, profile with site 10 000 and shock 0: `zero_yield`;
  not `Sited`, no profile (effective 10 000): no limit; not `Sited`, profile
  `all = 0`: `zero_yield`; not `Sited`, a seasonal profile with no entry for
  the season: `zero_yield`; not `Sited`, site 10 000 and technology, trainee
  or abundance 0: `zero_yield`. With the count: a `Sited` recipe in the arid
  region of PRO 12.1 (no profile) whose tool stock is also 0 reports
  `no_site`, not `no_tool`, and a `Sited` recipe with profile `all = 0` whose
  tool stock is also 0 reports `zero_yield`, not `no_tool` (the yield pair is
  first).
- `UnitCost`: the worked example of PRO 9.2 (`3 * 200 = 600`; wear
  `ceil(1 * 20 000 / 2 000) = 10`; labour `1 * 300 = 300`; cost 910; flour
  `ceil(910 * 10 000 / (2 * 10 000)) = 455`; at `E = 8 000`, `ceil(910 *
  10 000 / (2 * 8 000)) = 569`; the bran byproduct with share 0 is 0; a wage
  of 0 is 0) (FR-PRO-035, 036). Each 0 case breaks only its rule: an input
  price 0, a tool price 0, the wage 0 and negative, the factor 0, the share 0,
  an overflow of the input sum, of the share product (FR-PRO-036). Rounding
  up: a case where a truncating division would be one lower.
- The anchor: two recipes make one commodity, the cheaper non-zero wins; a
  recipe with an unknown cost is ignored; a commodity no runnable recipe
  makes is 0; a region with wage 0 is all 0 (FR-PRO-037, 049); a byproduct
  never sets an anchor.
- `FuzzUnitCost` and `FuzzExecutions`: no panic; `UnitCost` never negative;
  the count never exceeds the weight or any cap.
- Allocation tests at zero for the three (NFR-PRO-001 in part, NFR-PRO-003);
  benchmarks with `b.ReportAllocs()` and `b.Loop()` for the site lookup, the
  count and `UnitCost` (NFR-PRO-003). A benchmark at weight 1 and at weight
  `MaxWeight` shows the same cost (NFR-PRO-002).

**Observability and rollback:** none, for the reasons of Phase 3. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 3 pass; coverage holds at 90%.

## Phase 5: `production.go`, the yield table, the shock slot and the job

**Why here:** it holds the configuration-dependent state that Phase 6's steps
read. It needs only `Config`, its canonical form and the resolvers of Stage 0
(not the ledger), and the role and recipe tables of plan 1-2 (Phases 4 and 5)
for the recipe list of a role and its output counts.

- **The resolved yield table** (PRO 6.1, 6.2): built once from a canonical
  `Config` by `newYieldTable`, one row per profile in `(RegionID, RecipeID)`
  order with the seasons as resolved indexes (A3: the season ID to index
  table is built here from `Calendar.Seasons`, in declaration order, and each
  profile's entries, sorted by season ID string in the canonical `Config`,
  are re-sorted by index); `site(region, recipe, season)` looks the profile
  up and calls Phase 3, with the recipe's `Sited` flag, and returns the
  factor together with whether a profile row exists (the yield pair of Phase 4
  needs it: `no_site` is a `Sited` recipe with no profile, D59).
- **The shock slot** (6.8, D53, FR-PRO-023): a sparse list sorted by
  (region, recipe), default 10 000 for an absent key, an unexported setter
  that refuses a value outside `[0, MaxYieldBP]` with the existing
  `ErrOutOfBounds` (no new sentinel; tests assert it with `errors.Is`,
  testing.md 3.9) and removes a key set back to 10 000, an ordered read for
  the checkpoint.
- **The effective factor of a recipe in a region** at a season: the site
  factor, the shock, and the neutral technology, trainee and abundance terms
  (10 000 each in the slice) into Phase 3's function.
- **The production scratch** (A2, architecture.md 2.9): the unexported struct
  `productionScratch`, the one home of the buffers Phases 6 and 7 convert
  into. It holds `*Config` (canonical) and `*recipeTable` (plan 1-2) as
  read-only references, and reusable slices of `production.Term`,
  `production.Wear`, `ledger.Line` (inputs and outputs), the ordered
  limit-pair slice of Phase 4, the `[]production.Recipe` and the per-region
  anchor row of Phase 7, each sized once by `newProductionScratch(cfg,
  rt)` from the largest recipe (items, tools, outputs) and the recipe and
  commodity counts, and reused with `[:0]`. The constructor sizes buffers and
  converts no recipe; the conversion of a `*recipeRow` into `Term`, `Wear`
  and `Line` is written inline in the body of each Phase 6 and 7 step, as
  architecture.md 2.9 says, so there is no converting function, file or type
  and no persistent second copy of the recipe table. The world row holds one
  value of it (one per world: the buffers are mutable state, never shared
  between worlds) and passes it to the steps; tests build one from a small
  `Config` with `newRecipeTable` (an internal test) or through
  `export_test.go` (the external test). It is not saved and not hashed.
- **The job type** (5.1): recipe index, remaining ticks, executions, principal
  agent ID, the effective factor fixed at the start `Yield` (D67), pending
  quantities in output order (A5). There is no start-tick field: it is
  derived at completion (D67). The residue layout (A4):
  `residueLen(roleRecipes)` and `offset(roleRecipes, recipe)` over the
  ascending recipe list and the recipe table's output counts, and a function
  that sets every residue of a slice to one value (the creation draw's use,
  FR-PRO-026).
- **Admission** (4.1, 4.2, FR-PRO-008, 048): a request for a recipe not in the
  role's ascending recipe list is `not_on_role`, checked first; the check
  carries no other reason in the slice. This row builds the constant and the
  pure check; the call, with the `Debug` line, is the world's check on a
  strategy (AGT 6.5.1), in the `strategy.go` or `world.go` row, and
  [plan-phase-1-3.md](plan-phase-1-3.md) lists `not_on_role` under its
  `strategy.go` row for the same reason: the two plans agree, this one owns
  the constant and the check, the other the place that calls it.
- **The reason constants** (D56): `ReasonNotOnRole`, `ReasonZeroYield`,
  `ReasonNoSite`, `ReasonNoTool`, `ReasonNoInput`, appended to `event.go` with
  doc comments, and their rows in `TestConstants_Values` against the master
  order table that Phase 1 makes of PRO 4.2 (a plain parse of the table, as
  for the other closed sets; the parse target is that table's header and rows,
  not the prose); a `deferredReasons`-style list names the module reasons not
  built. How the test is shaped today (`event_test.go`, read 2026-10-13): it
  checks that each reason value is a backticked token of its owner spec by a
  `strings.Contains` of `` `value` `` in the whole file, and it assigns owners
  by **index range** of the `reasons` slice (the first ten clearing, the next
  two production, the rest ledger). So the new sentences with backticked
  `Outputs`, `Disposed` and `MaxQuantity` in PRO 14.1 cannot disturb it (it
  does not scan PRO for tokens), but appending the five new rows to that
  slice would put them in the ledger's range: they go in a **separate slice
  with its own owner** (`spec-production.md`), merged into `owner` after the
  loop, and the index ranges of the existing slice are not touched. The
  `deferredReasons` list that exists is clearing's (3.2); the master-order
  parse of PRO 4.2 has its own list. The root maps the yield pair's flag to `ReasonNoSite` or
  `ReasonZeroYield` (D59).
  Order and collisions (cross-plan check, 2026-10-13): appended at the end of
  the `Reason` constants of `event.go` as the PR merges, after the eight ledger
  constants of plan-phase-1-1 Phase 7 and the five belief reasons of
  plan-phase-1-4 Phase 5 if those have merged (`gate_closed`, `moved`, `held`,
  `at_floor`, `at_ceiling`); the Go names and string values of the three sets
  and of the Stage 0 constants are pairwise distinct. Re-read `event.go` and
  `TestConstants_Values` before the append; the new rows are a slice of their
  own.
- `doc.go`: the glossary gains pending output, yield shock and yield table;
  `production.go` is NOT added to the file lists of `glossary_test.go`: its
  second list (`TestGlossary_DeclaredTypesNamed`) fatals for a listed file
  that declares no exported type, and `production.go` declares unexported
  identifiers only; the `rootTypes` list would take it with no effect. Its
  terms are covered by the glossary text and the term tests of Stage 0.

**Tests:** unit, internal test (A8), with small configuration literals and one
external test over the packs.

- Lookup: the profiles of PRO 12.1 (north, south, far south) and 12.2 (lit and
  dark, permanent shadow `all = 0`) resolve to the values of 6.2 for each
  region and season; a recipe with no profile that is not `Sited` runs
  everywhere at 10 000; the arid region of 12.1 has no profile for the `Sited`
  `grow_grain` and gives 0 (FR-PRO-020, FR-PRO-028: a new region runs the
  recipe by adding a profile only, shown by a configuration that differs in
  one profile row and nothing else).
- Seasons: a calendar whose seasons are declared in an order that differs from
  their ID order (declared `winter, spring`, so the index of `winter` is 0 and
  of `spring` 1, against the ID order `spring, winter`) with a profile naming
  both: each season gives its own value, not the other's (A3; this is the
  case an index taken from the sorted profile would get wrong).
- The shock: absent is 10 000; set to 3 000 multiplies the site factor of its
  (region, recipe) only; set to `MaxYieldBP + 1` or -1 is refused with
  `ErrOutOfBounds` (`errors.Is`) and changes nothing; set back to 10 000
  removes the key; the ordered read is sorted (FR-PRO-023).
- Job and layout: the offsets of a role with two recipes of two and one
  outputs are 0, 2 and the length is 3; a role with no recipes has length 0
  (FR-PRO-002 in part); setting every residue sets all of them.
- The scratch (A2): built from a `Config` whose largest recipe has three
  inputs, two tools and two outputs, every buffer has capacity for it and
  the number of recipes, so the later steps never grow one; two scratches
  built from one `Config` share no backing array; building one reads no
  recipe's identity and holds no converted copy of the table (a text search
  of `production.go` for a `[]production.Recipe` field is the `Guard:`
  below). It allocates once, at construction.
- Admission: a recipe on the list passes, one off it is `not_on_role`, an
  empty list refuses everything (FR-PRO-008).
- `Guard:` (architecture.md 9.1, FR-PRO-001, FR-PRO-032): `production.go`
  reads no `ID` of a commodity, recipe or role and branches on no identifier,
  by a text search of the file as the Stage 0 determinism guard does; a
  byproduct (share 0, not discarded) and a co-product go through the same
  function. The same search finds no struct field or package variable of
  `production.go` of type `[]production.Recipe` other than the scratch's
  one buffer (A2: no persistent flat copy of the recipe table) and no
  function or type named for converting.
- The reason constants equal the strings of PRO 4.2 and each is declared once.

**Observability and rollback:** none, because no event is emitted and no log
line is written; the reasons are values the world row will log. Rollback is
reverting the PR; nothing outside `production.go` calls it until the world
row.

**Acceptance:** the commands of Phase 3 pass; the glossary, import and
determinism guards pass over the new file; coverage holds at 90% for the
root and for `internal/production`.

## Phase 6: `production.go`, the steps against the ledger

**Why here:** it needs Phases 3 to 5 (above all the `productionScratch` of
Phase 5), this plan's Phase 2 (`Discard`, `Holding`), **plan 1-2 Phase 4**
(the `recipeTable` and its `*recipeRow`, and `Duration` and a tool's
`Durability` from the canonical `Config`: the owner of the recipe data, A2) and
the ledger's `Transform` and `WearItems`, which are in the tree (plan 1-1
Phases 4 and 5). It is where conservation of goods through recipes is decided,
so it is the phase with the invariant test.

**Where the recipe data comes from (A2).** Every step below is a method of
`*productionScratch` (or takes it) and receives a `RecipeID`. It reads the
row with `rt.row(id)`, `cfg.Recipes[id].Duration` and, for each tool,
`cfg.Commodities[tool.commodity].Durability`, and writes the inputs as
`production.Term` (commodity, quantity, price where the step has one),
the tools as `production.Wear` (`WearPerUse`, `Durability`) and the ledger
lines as `ledger.Line` into the scratch slices, inline in its own body. It
calls no function that exists to convert and keeps no flat copy between calls.

- **`start`** (4.3, 4.4, FR-PRO-009 to 011, 013): given the executor's
  account, region, season, weight, the principal's account, the request
  (recipe and maximum), the effective factor and the stocks it reads from the
  ledger (`Holding`: the available units are `held - reserved`, D58), it
  builds the ordered limit list (the yield pair of D59 first, then D52, D57),
  takes `k`, and on `k = 0` returns the first applicable reason and changes
  nothing. Otherwise one `Transform` of `k` times each input to
  `recipe_input` (all or nothing; a recipe with no inputs, a source or a
  tool-only recipe, makes **no** start `Transform`, since one with both sides
  empty is `ErrEmptyTransform`), then `WearItems` of each tool for `k` items
  by `WearPerUse`, then the job with `Remaining` equal to the duration, the
  pending outputs fixed by Phase 3's output function over the agent's
  residues (which it updates), the executions, and `Yield` set to the
  effective factor it used (D67). A `Transform` that the
  ledger refuses is returned as that error with the job unstarted, the
  residues untouched and the ledger unchanged (FR-PRO-010; the `Transform` is
  atomic), so the start returns `(reason or error, nothing changed)`; the
  `WearItems` calls come after it has applied and are not reachable from
  input (the count showed the items present, and a commodity in both lists is
  D57), so a failure there is an invariant violation that panics (A6). It
  returns the facts the world needs: `k`, the effective factor, the wear per
  tool as (commodity, items, units each) and the inputs spent (FR-PRO-011,
  D51). Wear is applied once and never again for the job.
- **`maxExecutions`, the read of the limit** (4.2, D52, D57, D59; WLD P2.5 and
  P2.7, AGT 7.3): a pure read, with no write to the ledger, the job or the
  residues, that builds the ordered limit list exactly as `start` does and
  returns `k0`, the executions the request could start ignoring draws, and the
  first applicable reason when `k0` is 0. `start` calls it and then acts on its
  result, so the world's `Work` view (the per-recipe `MaxExecutions` and the
  refusal reason of AGT 7.3, WLD P2.5) and the re-derivation at P2.7 cannot
  drift from the rules of D52, D57 and D59. A module that adds a limit pair
  changes this one function.
- **`advance` and `complete`** (5.2, 5.3, FR-PRO-012, 013, 014): `advance`
  lowers `Remaining` by one and reports done at 0; the world calls it once
  for a continuation and once after a start, so a duration-1 job completes in
  its start tick and a job of `d` started in `s` completes in `s + d - 1`.
  `complete` takes the maximum quantity as a parameter, and the value passed
  must equal the ledger's `Config.MaxQuantity` (`Transform` checks its own
  `l.maxQty`, so a different value would hand it a line it refuses and the
  step would panic); a parameter and not a read of `bounds.go` so that the
  test can use a small bound against a ledger built with the same one
  (`production.go` is in the root and may read `bounds.go`; it is
  `internal/production` that may not). For each non-discard
  output `i` of pending quantity `q_i`, with `held_i` the units of the
  commodity the principal's account holds (`Holding`, reserved units
  counted, as the bound of `Transform` is on `held`, D58): `room_i =
  maxQuantity - held_i`, the part held is `fit_i = min(q_i, max(0, room_i))`
  and the part sent to `Discard` is `q_i - fit_i = max(0, held_i + q_i -
  maxQuantity)`. One `Transform` with no inputs creates each `fit_i` above 0
  by `recipe_output` with `Born` and durability as the ledger sets them (LED
  5.2.3); when every `fit_i` is 0 no `Transform` is made (a `Transform` with
  both sides empty is `ErrEmptyTransform`). Then one `Discard` per discard
  output whose `q_i` is above 0 (its whole `q_i`; a discard output of quantity 0
  makes no call, as `Discard` refuses a quantity below 1) and per overflow part
  above 0 (D54). A `Transform`
  refused for room is therefore not reachable from input; a `Transform` or
  `Discard` that is refused at all is an invariant violation (the account was
  read in the same step, and the only other cause is an `int64` overflow of a
  boundary or a flow total) and panics with a message that contains
  "invariant". The call returns, per output in output order, the units
  created (`q_i`, the whole pending quantity) and the units disposed
  (`q_i` for a discard output, `q_i - fit_i` otherwise), D60 (FR-PRO-029,
  030). A recipe with no output at all makes no `Transform` (5.3.1). The job
  is cleared and its pending slice kept (A5).
- **`abandon`** (5.4, FR-PRO-018): clears the job, moves nothing, returns the
  recipe, the executions and nothing else; the reason is the caller's.
- **`reweigh`** (5.4, FR-PRO-019): applies Phase 3's rescale to the job and
  returns the new executions.
- **The job of a warm start** (WLD 5.6.1 item 4, VIA 8.5): given the recipe,
  the executions, `Remaining` and the season of the virtual start, the pending
  output fixed with the effective factor of that season (which is also the
  job's `Yield`, D67), the shock at its
  default and **residue 0 on a scratch copy**, so the agent's own residues are
  left for the creation draw (FR-PRO-026, ruling C4). It touches no ledger:
  the grants and the construction `Transform` are the world row's, which has
  the inputs and tools from the recipe table.
- **The values of the two events** (FR-PRO-041): builders that write the
  `RecipeExecuted` and `RecipeAbandoned` struct values into a caller-owned
  slice, with `Outputs` and `Disposed` in output order and copies, never
  pointers into the job (observability-and-logging.md 4.1), holding what D60
  says: `Outputs` the gross quantity created per output item (discard and
  overflow included), `Disposed` the part never held, rows above 0 only, and
  what D67 says for the other two fields: `EffectiveYield` is the job's
  stored `Yield`, and `StartTick` is the completing tick minus the recipe's
  `Duration` plus 1, so the builder takes the tick and the duration as
  arguments and reads nothing else (a world restored mid-job, a job whose
  weight fell and a warm-start job all report the right pair). The
  same PR corrects the doc comment of `RecipeExecuted.Disposed` in `event.go`
  to say so (it says "the units of the outputs that were discarded" and is
  the one place that disagrees with PRO 14.1 and IND 7.2.4). They do not emit
  (D51). The slices of an event are copies and a recording observer keeps the
  event (observability-and-logging.md 4.1), so a builder never writes into
  scratch that the next call reuses: it takes the destination `Outputs` and
  `Disposed` slices as parameters and appends to them, and the world row
  hands each event its own pair. That allocation is the world's, made only
  when an observer was supplied (observability-and-logging.md 4.5); the
  builder allocates nothing given a destination with room, which is what the
  allocation test measures. A step that runs with no observer builds no
  event and calls no builder.
- **The aggregates of the snapshot** (14.3, FR-PRO-042): adding a job's
  pending quantities into a per-commodity slice and counting it into a
  per-recipe slice, over a region, for the world row's snapshot.
- **The working executions of the tick**: each of `start`, `advance` and
  `complete` reports the executions that are a working household-tick this
  tick (`k`, FR-PRO-015), for the welfare scratch (WEL 4.3, plan 1-6).
- `doc.go`: the glossary gains executor, principal and abandoned job if not
  already added.

**Tests:** unit, internal test (A8), with a real `internal/ledger` built from a
small configuration; integration-style but one package, so still at the unit
level (testing.md 2.1). The setup, which the steps cannot be tested without:
the test holds a small canonical `Config` literal (the recipes of PRO 9.2 and
12), builds the recipe table with `newRecipeTable(&cfg)` (plan 1-2 Phase 4),
the scratch with `newProductionScratch(&cfg, &rt)` (Phase 5) and the ledger
with the root's `newLedger(cfg)` of `ledger.go` (plan 1-1 Phase 7: it opens
the boundary accounts of the built-in group), then opens each executor and
principal account with `Ledger.Open(kind, owner, region)` and seeds stock with
`Move(Boundary(Initial, c), account, c, n, Initial)`, as
`ledger_internal_test.go` does. The external test of the two packs reaches the same four constructors
through `export_test.go` as function variables (A8); no fake and no
converter is involved.

- Start: a recipe with two inputs and a tool at weight 5 whose stock covers 3
  executions starts `k = 3`; the ledger shows exactly `3 * q` of each input
  moved to `recipe_input`, the tool items worn most-worn-first by `WearPerUse`
  on three distinct items, none worn twice (FR-PRO-010, 011); a tool at
  durability equal to `WearPerUse` is destroyed to `wear` (FR-PRO-011); a
  start with no input returns `no_input` and the ledger's observable state
  (every `Account` view, `AppendLots` of each account, `AppendFlows` and the
  boundary balances; the ledger has no clone) equals what was read before the
  call (FR-PRO-009). Reserved units: a stock of 5 units of an input of which
  2 are earmarked by a reservation gives a cap of `(5 - 2) / q`, not
  `5 / q`, and a tool stock of which 1 item is earmarked is one item fewer
  (D58; the available units are `held - reserved`), so `k` is lower and the
  earmarked units are untouched. Reasons: a `Sited` recipe in a region with no
  profile returns `no_site`, one whose profile gives the season no entry
  returns `zero_yield`, as does a profile of `all = 0` and a recipe with a
  shock of 0, each with the ledger unchanged (D59). The ledger refuses the input `Transform`:
  the ledger is built with the `recipe_input` boundary account opened for
  every commodity but one input, so `Transform` returns `ErrUnknownAccount`
  (`errors.Is`); `start` returns that error, no job is set, the residues are
  as they were and the ledger's observable state equals what was read before
  (FR-PRO-010). The test needs no fake and no ledger filled to its bound: an
  inputs-only `Transform` cannot reach `ErrBalanceOverflow` by ordinary
  operations. A failed `WearItems` after an applied `Transform` is not
  reachable through the ledger once the count has run, so it is not driven by
  a contrived call (and there is no coverage reason to: the guard is one
  `panic` line, marked in a comment as excluded and why, testing.md 9.5); the
  Stage 0 determinism guard fails a `panic(` whose message lacks the word
  "invariant" (N-224), which is what pins the message.
- Timing: a duration-1 recipe started and advanced in one tick completes in
  it; duration 3 started in tick 10 completes in tick 12 and produces nothing
  in 10 and 11; the job is never started in a tick that continued one (the
  precondition panics with a message that contains "invariant", asserted
  with `recover`) (FR-PRO-013, 014). Season at completion (FR-PRO-021): a job
  started in a season whose site factor is positive (spring 10 000) and
  completed in one whose site factor is 0 (summer, per the north profile of
  PRO 12.1) produces the quantity fixed at the start, and a start made in the
  summer is refused `zero_yield` (the north profile exists and lists no
  summer entry), and `no_site` only in the arid region, which has no profile
  (D59).
- Completion: the outputs have `Born` equal to the completing tick for a
  spoiling commodity and full durability for a durable (FR-PRO-012); the
  fractional yield of PRO 6.6 over four jobs produces 2, 3, 2, 3 units
  (FR-PRO-024); a discard output never appears in any account afterwards and
  both `recipe_output` and `disposal` show in the flow totals by commodity
  (FR-PRO-030, 031); an output that would push the holding above
  `MaxQuantity` is created in part and the rest disposed, the job completes
  and `Disposed` reports the amount (FR-PRO-029); a source recipe (no inputs)
  starts with no `Transform` and creates from nothing (FR-PRO-003); a
  tool-only recipe (no inputs, one tool) wears the tool and makes no
  `Transform` at its start; a recipe with no output makes no transform at its
  completion; a discard output whose quantity is 0 (a residue that gives no
  unit) makes no `Discard` call.
- The values of the two slices (D60, FR-PRO-029, 030, 041), one case each with
  the arithmetic in a comment, asserted on the event value and on the
  ledger's flow totals: (a) a recipe with a grain output of 6 and a slag
  discard output of 2 and room for all: `Outputs` is `[grain 6, slag 2]`,
  `Disposed` is `[slag 2]`, the account gains 6, `recipe_output` shows 8 and
  `disposal` 2; (b) the account holds `MaxQuantity - 4` of the grain, same
  job: `fit = 4`, `Outputs` `[grain 6, slag 2]`, `Disposed` `[grain 2, slag
  2]`, the holding is `MaxQuantity`; (c) the account holds `MaxQuantity`:
  `fit = 0`, no `Transform` is made for the grain, `Disposed` is `[grain 6,
  slag 2]`, the holding is unchanged; (d) reserved units count against the
  room: the account holds `MaxQuantity - 2` units of the grain and all of
  them are earmarked, so the available units are 0 and the room is 2: `fit =
  2`, `Disposed` `[grain 4, slag 2]`, the completion does not fail (a room taken
  as `maxQuantity - (held - reserved)` would offer 6 to `Transform`, which
  refuses on `held`, and the step would panic); (e) a discard output only:
  no `Transform`, `Outputs` and `Disposed` both `[slag 2]`; (f) a zero
  quantity (a factor below 10 000 and a residue that gives 0 units) is no
  row in either slice.
- `StartTick` and `EffectiveYield` (D67, FR-PRO-021, 041): a job of duration
  3 started in tick 10 in spring (factor 10 000) and completed in tick 12 in
  summer, after the shock was set to 3 000 in tick 11, reports `StartTick`
  10 (`12 - 3 + 1`) and `EffectiveYield` 10 000 (the factor of its start,
  not the 3 000 or the summer's 0 of completion); a duration-1 job reports
  `StartTick` equal to the tick; a job whose weight fell in between still
  reports `StartTick` 10; a warm-start job with `Remaining` 2 of duration 3
  and advanced once in tick 0 and once in tick 1 completes in tick 1 and
  reports `StartTick` -1 (`1 - 3 + 1`) and the factor of the virtual season; the job record
  copied (as a restore does) before completion reports the same pair.
- The read: `maxExecutions` returns the `k0` and the reason that a `start` of
  the same request then returns, over every case of the start tests above, and
  leaves the ledger's observable state, the job and the residues as they were
  (FR-PRO-009); it allocates nothing.
- The working executions of the tick (FR-PRO-015), one case each: `start` of
  a recipe at weight 5 whose stock covers 3 executions returns 3; a `start`
  refused with `no_input`, `no_tool`, `no_site` or `zero_yield` returns 0
  (and no job exists, so nothing is working); `advance` of that job in the
  next tick returns 3, and `complete` in the completing tick returns 3; after
  a weight fall to 2 (`reweigh` returns 2) the next `advance` and the
  `complete` return 2, the value the job holds and not the 3 of the start;
  a duration-1 job returns 3 from `start`, 3 from the `advance` of the same
  tick and 3 from `complete`. The value is the executions, never the
  weight or the number of calls.
- The aggregates of the snapshot (FR-PRO-042), over one region with four
  agents and one in another region, the arithmetic in a comment: recipe `r0`
  has outputs (`c0`, `c1`) and `r1` the output `c0`. Agent A holds a job of
  `r0` with pending `[6, 2]`, B a job of `r0` with `[4, 1]`, C a job of `r1`
  with `[5]`, D a job that was abandoned (no job, no pending), and E a job of
  `r1` with pending `[8]` at weight 4 whose weight fell to 2 (`reweigh`:
  `8 * 2 / 4 = 4`, exact, so no rounding rule is needed). The pending sum for
  the region is `c0 = 6 + 4 + 5 + 4 = 19` and `c1 = 2 + 1 = 3`; the job count
  is `r0 = 2` and `r1 = 2` (D is not counted); a job of the other region is
  added only by a second call into that region's own slices. Adding is the
  contract: a second call into the same slices doubles the figures (the
  caller zeroes), and a job completed between two calls leaves the figures of
  the remaining jobs. The call allocates nothing into slices that have room
  (NFR-PRO-001), asserted with `testing.AllocsPerRun`.
- **Guard:** plain ordered state (FR-PRO-043, the half that is here). By
  `reflect` over the job type: no field is a pointer, map, function,
  interface or channel, and the only slice field is the pending quantities
  (of `int64`), which the steps reuse (A5); and the ordered read of the shock
  slot after a scripted set of keys in a scrambled order, with one set back
  to 10 000, equals the keys sorted by (region, recipe) with the removed key
  absent, twice over the same sets (equal bytes for equal state, the form
  the checkpoint of Stage 2 needs). What it protects: a map or a pointer in
  the job, or a shock list read in map order, would make a restored world
  complete on other ticks or give two checkpoints of one state different
  bytes. The restore of a world mid-job is Stage 2's, with the checkpoint.
- Abandon and reweigh: an abandoned job moves nothing and creates no output;
  the inputs stay in `recipe_input`; a weight fall scales the pending output
  as PRO 5.4 and the later completion creates the scaled amount; a rise
  changes nothing (FR-PRO-018, 019).
- The warm-start job: pending output equals the floor of `k * q * E / 10 000`
  with the season of the virtual start and residue 0, and the agent's residues
  are not read or written (FR-PRO-026 in part).
- Invariant (testing.md 6.3, FR-PRO-044, 045; LED 8.2.2): a scripted run of
  several thousand operations (starts, advances, completions, abandons,
  weight falls) drawn from a test-local `math/rand/v2` source with a fixed
  seed, over a range of seeds, the failing seed in the message. After every
  operation the ledger's audit passes, goods are conserved except by
  `recipe_input`, `recipe_output`, `disposal` and `wear`, and for every
  commodity the units fixed at starts equal the units created plus the units
  pending plus the units dropped by abandons and weight falls.
- Two settings (A8, architecture.md 9.4, FR-PRO-001, 028): the same script
  runs over `one` and `second` with no code specific to either.
- Allocation tests (testing.md 8.2): a steady-state `start` after the first,
  an `advance`, a `complete` without overflow and the event builders into
  destination slices that already have room allocate nothing (NFR-PRO-001,
  NFR-PRO-003 in part). What the test measures is the builder; the per-event
  copies of the emission are the world row's and exist only with an observer
  (observability-and-logging.md 4.5).

**Observability and rollback:** the checklist of
observability-and-logging.md section 9 is answered in the PR. No log line, no
event and no metric is added (D51): the disposal `Warn` and the refused-start
`Debug` line have their rows already and are written by the world row, which
has the logger, the tick and the bound of one per recipe per tick. The
counterpart of the `Warn` is the `Disposed` rows of the non-discard outputs of
`RecipeExecuted`, which the builders fill (D60). Rollback is reverting the PR; nothing outside
`production.go` calls it until the world row.

**Acceptance:** the commands of Phase 3 pass; the invariant test passes over
its seed range; the allocation tests pass at zero; coverage holds at 90%.

## Phase 7: `production.go`, the cost anchors and the root `UnitCost`

**Why here:** it needs Phase 4's anchor function, Phase 5's effective factor
and plan 1-2's recipe table and makers index; it is the last file-level piece
the world calls at P1.2.

- **The recipe data for the anchor** (A2): there is no table to build. The
  anchor step is a method of the Phase 5 `productionScratch`; at each call,
  for each recipe it needs, it fills the scratch `[]production.Recipe`
  entry from plan 1-2's `rt.row(id)` (inputs as `Term` with the region's
  published price, tools as `Wear` with `WearPerUse` and the tool
  commodity's `Durability` read from `cfg.Commodities`, outputs with share
  and discard flag, `cfg.Recipes[id].Duration`) **inline in the loop**, into
  slices that already have room, and calls Phase 4's anchor function. The
  per-commodity list of makers in ascending `RecipeID` is plan 1-2's
  `makers`, read in place. Architecture.md 2.9 allows exactly this (inline
  conversion at the call into scratch the world holds) and forbids a type,
  file or function whose job is converting, so there is no
  `newProductionRecipes`, no persisted flat table and no cache to
  invalidate: the cost is one pass over the recipes' rows a tick, which the
  benchmark below measures against the `five` size. The PR says why the
  `Recipe` argument struct exists (the package cannot import the root, 2.9,
  so it needs its own flat shape) and that nothing of it is saved or kept
  between calls.
- **The anchor table** (9.3, FR-PRO-037, 047 to 049, A7): `anchors(dst,
  prices, wages, runnable, season)`, on the scratch, fills, per region
  ascending and commodity, the cost anchor from the previous tick's published
  prices (the seed price at tick 0), the region's wage and the recipes some
  agent of the region has on its role's list, computing each recipe's
  effective factor for that region and season through Phase 5. It writes
  scratch the world holds, allocates nothing, and is derived, not state.
- **The root `UnitCost(region, commodity)`** (PRO 9.4): the single value
  for a (region, commodity), as a read of the anchor table; the only
  consumers are the belief rule through the strategy view and tests.
- **No warm-start wage logic**: the substitution of the starting wage when
  the pooled figure is unknown is `welfare.go`'s region wage (plan 1-6); this
  function takes the wage it is given, and 0 gives 0 (FR-PRO-049).
- `doc.go`: no new term.

**Tests:** unit, internal test.

- A two-region configuration with the flour recipe of PRO 9.2 in both, wage 300
  in one and 0 in the other, a poor region at `E = 8 000` (a profile), and a
  shock of 5 000 in one: the anchors are 455, 569 and 0 where the arithmetic in
  a comment gives them (FR-PRO-035, 037, 049); the shocked region's anchor is
  `ceil(910 * 10 000 / (2 * 5 000)) = 910`.
- A recipe that no agent of the region has on its role list does not set an
  anchor (the `runnable` flag); a recipe whose factor is 0 there is ignored;
  a commodity made by two recipes takes the lesser (FR-PRO-037).
- Determinism: the table built twice from the same inputs is equal, and does
  not depend on the order the runnable list is given in, by a shuffled input
  (architecture.md 7.5).
- Both packs (A8: the external test, through `export_test.go`): the anchors of
  `one` and `second` at their seed prices and a fixed wage are computed with
  no code specific to either.
- Allocation test at zero (NFR-PRO-001); a benchmark of the full table at the
  size of the first consumer's world (`five`: 5 regions, 45 recipes, 30
  commodities), reporting `ReportAllocs` (NFR-PRO-003).

**Observability and rollback:** none, for the reasons of Phase 6: the table is
derived and no decision an operator needs explained is taken here. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 3 pass; coverage holds at 90%.

## Phase 8: the exit gate and the traceability

**Why last:** it checks the row as a whole and closes the documents.

- The exit gate on the whole module: `gofmt -l .`, `go vet ./...`,
  `staticcheck ./...`, `go test -short -race ./...`, the full run with
  `-covermode=atomic -coverpkg=./...`, `go mod tidy`; the per-scope coverage
  read from `go tool cover -func=coverage.out` and recorded, with the root at
  or above its Stage 0 figure and `internal/production` at 90% or more.
- Each fuzz target runs five minutes by hand with no crasher; a crasher is
  committed with a `Fix:` comment (testing.md 7.5).
- The traceability check of testing.md 4.6 by text search: every ID in a
  `Requirement:` line of the new tests exists in PRO, and every requirement the
  table below lists as covered in full is cited.
- The proof of D5: the pinned hash of the all-off reference configuration
  (`TestHashConfig_ReferencePinned`) is unchanged; `testdata/golden/off/` is
  checked once the world row records it.
- The roadmap row set to **done** with its commit; `documentation_index.md`
  corrected; N-030, N-137 and N-090 updated with what this row applied and what
  stays open.

Not gating, worth doing if time allows: a mutation pass over
`internal/production` and `production.go` (testing.md 9.4, a SHOULD), with
`gomu` inside Docker only, `--cpus=4 --memory=16g --memory-swap=16g`,
`GOMAXPROCS` and `--workers` equal to the CPU cap.

**Tests:** the tests of Phases 3 to 7 run under the full gate. No integration
or end-to-end test: there is no world, so the half of testing.md 2.3 that
exercises a requirement through the public API is the world row's, and the
table below says so.

**Observability and rollback:** the checklist of section 9 is answered in each
PR and in the summary below. Rollback is reverting the PR; nothing outside
the two places calls them until the world row.

**Acceptance:** every command of the gate is clean; the traceability check
finds no ID without a spec row and no "covered in full" requirement without a
test; the roadmap row reads done.

## Cross-plan contract

### Consumes

| Symbol, type, sentinel, spec edit or field | Provided by | Status |
|---|---|---|
| `Config.Recipes`, `Recipe` (`Duration`, `Inputs`, `Outputs`, `Requires`, `Sited`, `TraineeEligible`, `TraineeYield`), `Config.Yields`, `YieldProfile`, `SeasonYield`, `Config.Regions`, `Role.Recipes`, `Config.Calendar`, `Commodity.ShelfLife`, `Durability` | Stage 0, `config.go` ([plan-phase-0-foundation.md](plan-phase-0-foundation.md)) | Built (023cf97) |
| `Config.canonical`, `recipeID`, `regionID`, `commodityID` (typed ID = index in the canonical slice) | Stage 0, `config.go` | Built |
| `MaxQuantity`, `MaxMoney`, `MaxWeight`, `MaxYieldBP`, `MaxRecipeTicks`, `MaxRequires`, `MaxRecipeItems` | Stage 0, `bounds.go` | Built |
| `internal/checked`: `Add`, `Sub`, `Mul`, `MulDiv`, `MulDivRem`, `MulDivFloor`, `MulDivCeil` with `ok` | Stage 0 | Built |
| `RecipeExecuted`, `RecipeAbandoned`, `Amount`, `ReasonAgentRemoved`, `ReasonRoleChanged`, `ReasonRecipeInput`, `ReasonRecipeOutput`, `ReasonDisposal`, `ReasonWear`, `RegionID`, `RecipeID`, `CommodityID`, `AgentID`, `Quantity` | Stage 0, `event.go`, `value.go` | Built |
| `ErrOverflow`, `ErrOutOfBounds`, `ErrUnknownSeason` and the rest of the validation sentinels | Stage 0, `errors.go` | Built; this row adds no sentinel |
| `ledger.Ledger.Transform(account, inputs, outputs)`, all or nothing, an empty side allowed, `ErrEmptyTransform` | [plan-phase-1-1.md](plan-phase-1-1.md) Phase 4 (LED 8.1, 8.2.1) | **In the tree** since the cross-plan review of 2026-10-13: `Transform(id int, inputs, outputs []Line) error` with `Line{Commodity int; Quantity int64}`; Phase 6 here no longer waits for it |
| `ledger.Ledger.WearItems(account, commodity, n, units)` (`n` most worn distinct items, destruction to `wear`, wear units counted) | plan 1-1 Phase 4 (LED 5.4.3) | **In the tree** (2026-10-13): `WearItems(id, commodity int, n, units int64) error`, `Wear(id, commodity int, units int64) error` and the counter `WearUnits(commodity int, bucket bool) (int64, error)`. It returns no wear units: the facts `start` returns (commodity, items, units each) come from the recipe's `WearPerUse`, not from the ledger |
| `Ledger.Holding(account, commodity) (held, reserved, err)`, zero allocation: the units of a commodity in an account and how many are earmarked (the room against `MaxQuantity`, the tool and input stocks) | **This row, Phase 2** (D58; plan 1-3 assumes the same read, fallback `AppendLots`) | **Resolved**: the tree has no such read (`account.held` is unexported); added additively in new files |
| `Ledger.Move(from, to, commodity, qty, reason)` of an agent's goods to the `disposal` boundary, `Boundary(reason, commodity)`, the reason constants `RecipeInput`, `RecipeOutput`, `Disposal`, `Wear`, a boundary account per commodity, `AppendFlows`, `Tick`, `ClearTotals`, `AppendLots` | plan 1-1 Phases 2 and 3 | **In the tree** (read 2026-10-13: `internal/ledger/lots.go`, `ledger.go`, `registry.go`); names as in the code |
| `Ledger.Discard(account, commodity, quantity)`: create under `recipe_output` and sink under `disposal` with no net change in the account and no room check | **This row, Phase 2** (D54, answered by coordinator 2026-10-13); LED 8.2 gains a clause and the table of section 13 a new requirement ID for it (FR-LED-039 is unchanged) | **Resolved**: a sibling operation in new files, so plan 1-1 and its Phase 4 are untouched |
| The ledger's `Audit() error` (for the invariant test) and the construction-phase operations (`MoveStated`, in the tree, for the warm-start job's inputs) | plan 1-1 Phases 6 and 3 | **In the tree** (`internal/ledger/audit.go`, read 2026-10-13); the warm-start steps are the world row's |
| The recipe table (inputs, outputs, tools by typed ID, `Discard`, `CostShare`, source flag) and the makers of a commodity in ascending `RecipeID`: the **one owner** of the recipe data; this row keeps no second table (A2) and reads the rows inline into the Phase 5 scratch at Phases 6 and 7 (a tool's `Durability` and `Duration` come from the canonical `Config`, by ID) | [plan-phase-1-2.md](plan-phase-1-2.md) Phase 4 (`newRecipeTable`, A2); **Phase 6 here waits for it** (A9), as Phases 5 and 7 do | Provided; names fixed by the "Cross-plan contract" of plan 1-2: `newRecipeTable(c *Config) recipeTable`, `(*recipeTable).row(RecipeID) *recipeRow` with `inputs`, `outputs` (`costShare`, `discard`), `tools` (`wearPerUse`) and `source`, and `(*recipeTable).makers(CommodityID) []RecipeID`; `Duration`, `Sited` stay in the configuration and are read by ID (its Phase 4) |
| The role table: per role the ascending `RecipeID` list | plan 1-2 Phase 5 (`newRoleTable(c *Config) roleTable`) | Provided; names fixed by the "Cross-plan contract" of plan 1-2: `(*roleTable).row(RoleID).recipes` (`[]RecipeID`, ascending) |
| `Calendar.SeasonAt(tick) int` (any `int64`, -1 with no seasons) and `Calendar.At(tick) (Moment, error)` with `Moment.Season` | plan 1-2 Phase 2, A3 | Provided (signatures fixed by plan 1-2 A3) |
| The pooled regional wage per region as `int64`, 0 when unknown, with the warm-start wage substituted | [plan-phase-1-6.md](plan-phase-1-6.md) ("region wage", Provides) | Provided in substance by plan 1-6 (built after this row); this row takes a slice parameter, so any source fits |
| The published price per (region, commodity) of the previous clearing, the seed price at tick 0 | `marketReport` and `seedReport(price)` of [plan-phase-1-3.md](plan-phase-1-3.md) Provides; the state is the world row's `market.go` | Provided in substance by plan 1-3; this row takes a price slice per region, so any source fits |
| The agent slot, weight, role, account and region of an executor; the "free agent" test; the rotation; the season read once at P1.1 | `agent.go`, `world.go` rows | **ASSUMED** (WLD 6.4): passed in as arguments |
| WEL scratch methods (`addWear`, `markWorking`) | plan 1-6 Phase 4 | Not called here (D51); the world row calls them with the facts of Provides |
| PRO 19, 4.2, 9.4, 5.3 edits (D50, D57, D56, D54) | This row, Phase 1; plan 1-2 Phase 1 edits other rows of the PRO 19 table | Owed here; coordinate with plan 1-2 |
| LED clauses and two new FR-LED IDs for `Discard` and `Holding`; LED 7.2.1 and 8.2.1 reworded (the completion is a `Transform` and `Discard` calls) | This row, Phase 2 | Owed here; targeted edits after a re-read, with the IDs taken at that time, because the ledger workflow also edits `spec-ledger.md` |

### Provides

| What | For | Contract |
|---|---|---|
| `production.Term`, `production.Wear`, `production.UnitCost` with the signature of PRO 9.4 | `production.go` Phase 7, tests of `beliefs.go` (they take `UnitCost` as a plain `int64`) | Fixed by the spec; rounds up; 0 means unknown; no side effect, no allocation |
| `internal/production`: site factor, effective factor, output with residue, rescale, the count over an ordered list of `(cap, reason)`, the cost anchor of a commodity | `production.go`; later modules adding a limit or a factor term | Pure over primitives; truncation toward zero except cost (up); `ok` false on overflow; a module appends a pair or a neutral parameter and changes no signature |
| The unexported `job` type: recipe index, remaining, executions, principal agent ID, the effective factor `Yield` fixed at the start (D67), pending quantities in output order | `agent.go` (holds one per agent as AGT 3.1 says, O1), `checkpoint.go` (CHK tag 6, `Yield` after `Principal`), the snapshot | Plain data, no pointer to other state; `Remaining` at least 1 while active; no start-tick field (derived at completion) |
| `productionScratch` and `newProductionScratch(cfg *Config, rt *recipeTable)` (Phase 5): the buffers of the steps and of the anchor, sized once, one per world, not saved | `world.go` (holds one value and passes it to `start`, `maxExecutions`, `complete` and the anchor step) | Read-only references to the canonical `Config` and the recipe table; mutable buffers, so never shared between worlds; the conversions are inline in the steps (architecture.md 2.9) |
| `residueLen` and `offset` of a role's recipe list, and "set every residue" | `agent.go`, `world.go` (creation draw, role change reset to 0) | Layout in ascending recipe, then output order; a function of the role and the recipe table, never saved |
| `start`, `advance`, `complete`, `abandon`, `reweigh` over a `*job`, a residue slice and a ledger | `world.go` (P2.1, P2.7, P7.1, removal fan-in), `spawner.go` (role change, weight change) | Each atomic against the ledger; `start` returns `k`, the factor and the wear facts, or a reason with nothing changed; the caller owns the ordering of 5.5 |
| `maxExecutions`: the read of `k0` and the first reason for a request, with no write | `world.go` (P2.5, P2.7), `strategy.go` (the view's per-recipe `MaxExecutions` and reason, AGT 7.3) | Pure; the same function `start` calls, so the view cannot drift from `start` |
| The warm-start job constructor (recipe, executions, remaining, season of the virtual start) | `viability.go`, `world.go` (WLD 5.6.1 item 4) | Pending output with the virtual season's factor, shock default, residue 0 on a scratch copy; touches no ledger |
| The values of `RecipeExecuted` and `RecipeAbandoned` written into a caller-owned slice | `world.go` (emission behind the observer flag) | No pointers into the job; `Outputs` and `Disposed` in output order; `StartTick` = completing tick - `Duration` + 1, `EffectiveYield` = the job's `Yield` (D67) |
| The working executions of each step, the wear facts (commodity, items, wear units each) and the inputs spent | `welfare.go` scratch via the world row (`markWorking`, `addWear`, `addInputSpend` is clearing's) | `k` household-ticks per tick of a job (FR-PRO-015) |
| The cost-anchor table per (region, commodity) and the root `UnitCost(region, commodity)` read | `strategy.go` (the view's `UnitCost`, AGT 7.3.6), the callers of `beliefs.Limits` in the agent and world rows (the belief bounds; `beliefs.go` holds no call of the rule) | Computed once per tick at P1.2 from the previous tick's prices and wage; 0 is unknown; derived, never saved |
| The yield-shock slot: ordered read, unexported setter with bound | `integration.go` (`SetYieldShocks`, `YieldShocked`), `checkpoint.go` (section 6, sparse) | Default 10 000, `[0, MaxYieldBP]`, sorted by (region, recipe) |
| The snapshot aggregates: pending output per (region, commodity) and agents with a job per recipe | `snapshot.go` (FR-PRO-042, `economysim.production.pending`) | Added over the agents of a region by the caller |
| The reason constants `not_on_role`, `zero_yield`, `no_site`, `no_tool`, `no_input` in `event.go` | `world.go`, `strategy.go`, the observability tests | Exported root constants; the module reasons are appended later, in the master order |
| `Ledger.Discard(account, commodity, quantity)` and `Ledger.Holding(account, commodity) (held, reserved, err)` in `internal/ledger/discard.go` | this row's Phases 6 and 7; plan 1-3 (`Holding` in place of its `AppendLots` fallback); later rows that need a held quantity | Additive; `Discard` is atomic, never refused for room, and moves both boundary accounts and both flow totals; `Holding` allocates nothing, closed account and boundary give 0 and 0 |
| The glossary terms of N-017 in `doc.go` | all later rows | Added before use |

## Security review

The module is a library with no network, file access, credentials or
persistence. This row adds no trust boundary: `internal/production` and
`production.go` are called only by the root with numbers derived from a
validated configuration and from the ledger. A consumer reaches production
through a `Strategy`'s `JobRequest` (a recipe ID and a maximum) and, later, the
yield-shock call.

- **Who can reach it, and with what input.** A strategy can name any recipe
  and any maximum, including a negative or enormous one; the world refuses a
  recipe off the role's list first (`not_on_role`) and the count takes the
  least of the weight and the caps, so a maximum above the weight changes
  nothing, 0 means no maximum (AGT 6.3), and a negative maximum is treated as
  0 by the count function rather than wrapping. Tests include all three. A strategy
  that tries to start a job in a tick it continued one is a programmer error
  of the world's, not of a consumer: the world decides "free".
- **Integer overflow.** `k * q * E`, the cost sums, the share product and the
  rescale go through `internal/checked`; validation (PRO 10.3) bounds them, and
  the step still reports `ok` false rather than wrapping (FR-PRO-036, 039). A
  wrap would be a silent wrong output; the boundary cases and the fuzz targets
  guard it.
- **Partial application.** A start that took inputs and then failed to wear a
  tool would destroy goods. The order is the count (read only), one atomic
  `Transform`, then the wear of items the count has already shown present; a
  failure of the wear after a successful `Transform` is an invariant
  violation and panics with a message that says so (architecture.md 8.1),
  never a half start. The invariant test and the ledger comparison after each
  refusal guard it.
- **Panics and division by zero.** A zero output quantity, a zero yield or a
  zero duration cannot reach the functions after validation; each division
  has a guard that returns "unknown" (0) or `ok` false, and each has a test.
  Untrusted input does not reach this row directly.
- **Resource exhaustion.** A step is bounded by the recipe's items
  (`MaxRecipeItems`, `MaxRequires`) and not by the weight (NFR-PRO-002); the
  anchor table is `regions x commodities` words, bounded by `MaxRegions` times
  `MaxCommodities` (10^6) and built once per world; the yield table is at most
  `MaxYields` rows, which the count bounds of Stage 0 check before it is built.
- **Manipulating the cost anchor.** The anchor reads the previous tick's
  pooled wage and published prices, so a single agent's order cannot move this
  tick's anchor (PRO 9.3), and a strategy cannot write it.
- **Not applicable:** authentication, authorisation, injection, secrets,
  cross-tenant exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer.
  Nothing in this row is exported from the module except five `Reason`
  constants appended to `event.go` (a compatibility promise of architecture.md
  2.8, 6.6; appended, never inserted, 2.10.5). `internal/production` is
  internal and `production.go` adds unexported identifiers only.
- **Document contracts.** Phase 1 completes FR-PRO-009, 012, 029 and 030 and
  adds sentences to PRO 14.1 and IND 7.2.4; D67 adds the job's `Yield` to PRO
  5.1.1 and 15 and CHK 4.2 tag 6 and a clause to FR-PRO-041 (no ID changes); Phase 2 adds two FR-LED IDs and
  rewords LED 7.2.1 and 8.2.1. FR-LED-039 and every existing ID stay as they
  are and nothing is renumbered (testing.md 4.1). `doc/notes.md` gets a fresh
  tag.
- **What becomes hard to change, and when.** The residue layout, the order of
  the output rows of a job and the rounding rules enter the golden files at the
  end of Stage 1 and the checkpoint in Stage 2 (CHK tag 6). Until the first
  golden file is recorded they may be corrected in a normal PR; after it, a
  change is a behaviour change with the explanation of testing.md 7.2.
- **Append-only stays true.** A module adds a limit pair, a factor term, a
  reason and its abandon reason in its own PR, with the module off adding
  nothing (D5), so `testdata/golden/off/` does not move.
- **Rollout** is one PR per phase (A9). Each leaves the branch passing every
  command of `AGENTS.md`. A phase is rolled back by reverting its PR; later
  phases depend only on earlier ones, and no other row calls this one yet.
- **Dependency failure.** None at run time. `Transform` and `WearItems` are in
  the tree, as is `Audit`, so no schedule risk is left on plan 1-1; Phase 2 (the
  ledger additions) needs only what is in the tree and Phases 3 to 5 are
  independent of the ledger, so they can land first, and Phase 7 takes the
  wage and prices as parameters so it is not blocked by rows 3 and 6.
- **The Go toolchain.** `go.mod` says `go 1.25`. No map is iterated
  and the tick sorts nothing. The only sorts are at construction (A3: each
  profile's season entries by season index, and the (ID, index) table of the
  seasons), on keys that are unique within their list, so the order of equal
  elements cannot arise (architecture.md 7.6); every other table is built in
  index order (7.5).

## Testing (summary, testing.md section 11)

The pyramid for this row is all base: table-driven unit tests in
`internal/production` and in the root's internal test of `production.go`, fuzz
targets over the pure functions, one invariant test over the steps against a
real ledger, allocation tests and benchmarks. There is no integration or
end-to-end test because there is no world.

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | the two new FR-LED requirements of Phase 2 deliverable 1 (`Discard`, today 066; `Holding`, today 067; IDs taken at the edit); FR-LED-006, 007, 037 (its refusals and atomicity); NFR-LED-001 (the read and the operation allocate nothing). FR-LED-039 is not cited: it is about a `Transform` with an empty side and is unchanged | FR-PRO-029 and 030 (the ledger half here, the completion in Phase 6); FR-LED-035, 036 (the audit after each operation of the scripted run) |
| 3 | FR-PRO-020, 022, 024, 025 | FR-PRO-019 (the rescale here, the job in Phase 6); FR-PRO-034 (the factor here, the trainee in SKL); NFR-PRO-004 (the lookup fuzz here; the validator fuzz is Stage 0's `FuzzValidate`, which this row does not re-run as new) |
| 4 | FR-PRO-035, 036, 037, 049; NFR-PRO-002, 003 | FR-PRO-009 (the count and the yield pair of D59 here, the steps in Phase 6; module limits with their modules); FR-PRO-047 (the wage is a parameter, the source is plan 1-6) |
| 5 | FR-PRO-008, 023, 028, 032 | FR-PRO-001 (the identifier guard of `production.go` here; the plain-struct and no-embedded-field rule of `Recipe` is plan 1-2's struct guard); FR-PRO-002 (the layout; validation was Stage 0); FR-PRO-048 (the check, the order of the phase is the world row) |
| 6 | FR-PRO-010, 011, 012, 013, 014, 018, 019, 021, 029, 030, 031, 044, 045; NFR-PRO-001 (the steps) | FR-PRO-015 (the working executions returned by each step, tested here; the scratch is welfare and the world row); FR-PRO-026 (the warm-start residue 0 here, the draw in the world row); FR-PRO-041 (the values, including `StartTick` and `EffectiveYield` of D67, here; the emission in the world row); FR-PRO-042 (the aggregates here, with their own test; the snapshot in the world row); FR-PRO-043 (plain ordered state here, a Guard test of the job type and the shock read; the checkpoint and the restore mid-job in Stage 2) |
| 7 | none new | FR-PRO-037 (the table over the packs), FR-PRO-035 (the root read) |
| 8 | none | The traceability check |

"Covered in full" means at the lowest level that can verify it (testing.md
2.1). The second half of testing.md 2.3, the same behaviour through the public
API at the integration level, is the world row's for every consumer-visible
requirement above: the invariants of testing.md 6.3 over the reference chain
and a range of seeds (FR-PRO-044, 045) and `RecipeExecuted` and
`RecipeAbandoned` with `Disposed` (FR-PRO-041) are in Stage 1's "Done when".

Built in Stage 0 and not re-tested here: FR-PRO-004 to 007 (recipe shape and
module dependency), FR-PRO-038, 039, 040 (validation and warnings), FR-PRO-003
and FR-PRO-005 beyond a start of a source recipe. Deferred with their code:
FR-PRO-016, 017, 048 (the order and the rotation: `world.go`); FR-PRO-027
(`takeEndowment`: `resources.go`); FR-PRO-033, 034 (the trainee: `skills.go`);
FR-PRO-046 (the two packs through V1 to V4: `viability.go`); FR-PRO-047 (the
wage: `welfare.go`); FR-PRO-050 (`migration.go`) and FR-PRO-051
(`subsistence.go`).

Boundaries and error paths: every reason of the slice and every `ok` false
has a case that breaks only its rule and is asserted by value or with
`errors.Is` (testing.md 3.9). Golden files: none exists yet. Fuzz: `FuzzSite`,
`FuzzEffective`, `FuzzUnitCost`, `FuzzExecutions`. Each PR states its coverage
and that its tests were run against a broken version (testing.md 9.3).

## Observability (summary, observability-and-logging.md section 9)

None added, because: no failure path is survived rather than returned: a
refused start is a returned reason and the world row logs it at `Debug` (1);
the transitions of a job (completion, abandon) have events since Stage 0 and
this row builds their values without emitting (2, D51); the count and the
cost make no decision an operator needs explained beyond the reason they
return (3); no loop runs in a tick in this row's own code, and the step and
builder allocations are tested at zero so the no-op path stays free (4, 8.2);
no event, metric, unit or attribute changes (5, 6); neither `Error` nor `Info`
is used, and the one `Warn` of PRO 14.2 is bounded one per recipe per tick by
the world row (7); no error is logged and returned (8); the tests of section 8
that apply here are the allocation tests, and the rest (logs, golden
rendering, emission order, non-interference) need a world (9).

## Open items

- **O1.** Owed to the agent row: where the job and residues are stored. This
  plan assumes the `Agent` fields of AGT 3.1 (D50). If the agent row follows
  architecture.md 2.10.1 and keeps them in slices indexed by agent slot, the
  functions here are unchanged and PRO 19 and AGT 3.2 are edited to say so.
- **O2.** Closed (coordinator, 2026-10-13): create-and-dispose is the sibling
  operation `Discard`, built in Phase 2 of this plan in new files of
  `internal/ledger` (D54). Plan 1-1 and its Phase 4 are untouched. Left for the
  plan 1-1 owner, optionally: a cross-reference to `Discard` in the doc comment
  of `Transform` when Phase 4 lands.
- **O3.** Owed to the owner (D55): `TrainingPays` is left to `skills.go`.
- **O4.** Closed (coordinator, 2026-10-13): the goods read is `Holding`,
  built in Phase 2 (D58), and plan 1-3 may use it in place of its
  `AppendLots` fallback.
- **O7.** Mismatches with sibling plans, adopted from them: (a) plan 1-6
  says the cost anchors are computed at P1.1 and "provides the wear cost of an
  execution to `production.go`"; WLD 6.3 puts the moment at P1.1 and the anchors
  at P1.2, and this row does not call welfare (D51): the steps return the wear
  facts and the world row passes them to welfare's scratch. Corrected in plan 1-6 by the
  cross-plan consistency review (2026-10-13). (b) Plan 1-3 puts
  `order.go` and the market report type in `clearing.go` and leaves `market.go`
  to the world row; this row assumes nothing of them beyond a price slice. (c)
  Plan 1-2 fixes `SeasonAt(tick int64) int` and `At(tick) (Moment, error)`;
  this row uses `SeasonAt` for the warm-start job and takes the tick's season
  as an argument for a start, so it needs neither the error path nor `Moment`.
- **O5.** Whether the cost-anchor scan "some agent of the region has the
  recipe" is cheaper as a per-region count kept by the spawner and the world
  than a scan of agents per tick is measured by the world row against SCL 6
  (as plan 1-6 O2 does for the wage); this row takes the flag list as input,
  so either source works.
- **O6.** No CI exists (O3 of plan 0). The gate of Phase 8 and the timed fuzz
  runs are done by hand, on one architecture.
- **O8.** Owed to the owner (D57): whether a recipe that lists one commodity
  in both `Inputs` and `Requires` is rejected in validation (a Stage 0 rule
  and sentinel citing PRO 4.4.3, "a tool never appears in `Inputs`"), which
  would delete the defensive cap of D57. Not decided here; the cap is correct
  either way.
