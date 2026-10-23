# Plan: Stage 2, row 9: RES, resources, endowments and the ration (`resources.go`, `internal/endowment`, `internal/curve`)

**Status: written (2026-10-22), not started.** Sequences the ninth plan of Stage 2 ([plan-2-0.md](plan-2-0.md)): the roadmap row "`resources.go` and `internal/endowment`, `internal/curve`" ([roadmap-v1.md](../roadmap-v1.md) Stage 2). It is a working document: the specs, the ADRs and `doc/rules/` decide, and where this plan disagrees with them they win.

Rule tags read (as the files end, 2026-10-22; they equal the contract's): architecture=403404277d5a40376fd91790, observability=e70199779a159d4cb303d273, testing=8d141d87287fe0130681bbe1. `architecture.md` 2 was read in full and the rest by the sections this row touches; `testing.md` 4, 6, 7, 9, 11 and `observability-and-logging.md` 3, 4, 9 likewise.

Implements [spec-resources.md](../specs/spec-resources.md) (RES): FR-RES-001 to FR-RES-050 and NFR-RES-001 to NFR-RES-004, in the order of the phases below. Reads [spec-production.md](../specs/spec-production.md) 3.1, 4.2, 5.5, 6.3, 6.5, 10 and [spec-regions.md](../specs/spec-regions.md) 5 (the region-keyed table). Hash id: `Modules.Resources` is field 5 (`spec-config.md` 8.2.1). Checkpoint tag 15. It also consumes plans 2 and 5 (outside its Needs column, 3 and 8) because section 15 is registered in plan 2's table and the module hooks and golden helper are plan 5's. Positions P2.3, P2.6, P8.3 (owner RES, `spec-world.md` 6.4, 6.10). **Decision block: D380 to D399.** Ships in two PRs: PR A = Phases 1 to 4 (edits, curve and endowment packages, configuration); PR B = Phases 5 to 10.

Notes: the index of `doc/notes.md` was read, and the entries of N-016, N-017, N-027, N-103, N-126, N-128, N-163, N-217 (tag ALL) and of N-030, N-092, N-109 (the notes that name RES). No open note carries the code RES (N-030, N-031, N-036, N-046, N-104, N-106, N-107 were settled for RES in `spec-resources.md` 22). N-016: apply, every field of 12 has a registry row (checked in Phase 4); N-017: apply, the glossary list in Phase 1; N-027: leave open, every number here is a placeholder; N-103: leave open, one writer, this file only; N-126: leave open, the two events are declared with their catalogue rows (Phase 6); N-128: not applicable, the module saves no hosted per-agent field; N-163: leave open, the word "treasury" is not used; N-217: apply, this row leaves no trace while off (Phase 4 and the gate); N-030: leave open for REF and STY, the RES part is the doc-only Phase 1 and the check of Phase 5; N-092: leave open, this row builds the overlay it names (Phase 9); N-109: leave open, the solver reports `Headroom` and `N_bind` inputs, `r_full` stays a pack property. `AGENTS.md` asks for the notes tag only for work that changes a spec or a note; this plan changes neither, so the Notes tag is not quoted, and Phase 1 re-reads the file before the supervisor edits any spec.

## Scope

**Builds.**

- `internal/curve` (stdlib and `internal/checked`): the one piecewise-linear evaluation of a knot list. Built here because RES is the first consumer in the order of the stage and POP (plan 11) shares it (RES 24, POP 3.5.1, FR-POP-004).
- `internal/endowment` (stdlib, `math/bits`, `internal/checked`, `internal/curve`): the regeneration of the three shapes with the carry, the effective capacity and scale, the abundance, the ration, and the two integer conditions of RES 8.3.1.
- `resources.go` (config types, defaults, state, the view group, snapshot rows, checkpoint section 15), `resources_check.go` (validation and warnings, including `no_comparative_advantage`) and `resources_tick.go` (P2.3, P2.6, P8.3, the take, the shock helpers, events, the `Warn`, the audit check). The two continuation files keep each file under the Lizard limit (N-243); Phase 1 owes `architecture.md` 2.6 and `AGENTS.md` Layout their names (D383).
- `internal/steady`: the resource rows of the programme (`sustain.go`) and the producibility predicate of RES 9.2 (Phase 8); `viability*.go` call them.
- `config/resources.go`: the loader for `[modules.resources...]` and the `draws` key of a recipe.
- `testdata/chains/overlay/resources*.toml`, `testdata/golden/resources/`, the scenarios S42 to S45 (S41 and the public half of S45 are owed, D398).
- The catalogue appends of the table below.

**Does not build.**

| Item | Owner |
| --- | --- |
| `SetCapacityScales`, `SetRegenScales`, `AdjustEndowments` (public calls and their option guards) | plan 17 (INT); this plan builds the helpers they wrap (D387) |
| The birth and death schedules, `no_fixed_factor`, the wiring of `resource_overshoot` and `resource_unstable_plan` | plan 11 (POP); this plan builds the conditions (D399) |
| Own-use recipes that draw, the `OwnUse` mode | plan 10 (SUB); its requests join this ration unchanged (FR-RES-043) |
| Tenure tiers, rent, `Held`/`Idle` in the `Warn` | LND, Stage 3; the hooks exist (RES 10.2) |
| Facility layer `B` of the capacity, P8.3b, `Kind` `facility` | CAP, Stage 3; `B` is 0 here |
| OpenTelemetry instruments of RES 14.3 | plan 22; this plan emits the events they read |
| The yield-table format and lookup | PRO, built (`production.go`); RES owns values only (D395) |
| `r_full` as a measure (RES 23.6) | VIA and OBS, open |

## Decisions

- **D380.** `internal/curve` is built in this plan, before POP. API: `type Knot struct{ X, Y int64 }`; `Eval(knots []Knot, x int64) (y int64, ok bool)` (flat outside the first and last knot, `y_i + trunc((y_{i+1}-y_i)*(x-x_i)/dx)` through `checked.MulDiv`, truncation toward zero for a falling segment, `ok` false on overflow); `Segment(knots, x) (x0, y0, x1, y1 int64, found bool)` (the segment that contains `x`, for the elasticity of 8.3.1); `Check(knots, xMax, yMax int64) (bad int)` (index of the first knot out of range or not strictly increasing, -1 if none); `NonDecreasing(knots) bool`. It knows no share, factor, ratio or rate: ranges are the caller's. Source: RES 4.7.1, 24; POP 3.5.1; FR-POP-004; the order of `plan-2-0`.
- **D381.** `internal/endowment` takes no key of its own: `ordering.Key(tick, agent)` already is the finaliser of RES 5.3.3 (`internal/ordering/ordering.go`, built in Stage 0). The caller puts it in `Request.Key`; the sketch in RES 24 that lists `endowment.Key` is corrected (Phase 1). `Regen` takes a struct, not nine parameters, for the 8-parameter Lizard limit: `Regen(p Params, in Step) (Result, bool)`. `Ration` sorts the caller's request slice in place and writes `grants` at the slot of `req.Ref`, so it allocates nothing. Source: RES 5.3.3, 24; N-243.
- **D382.** The regeneration numerator reaches 10^31 (RES 4.6), beyond `checked`, which takes and returns `int64`. `internal/endowment/wide.go` holds a local unsigned 128-bit value over `math/bits` (`mul64`, `add`, `divmod64` by `bits.Div64`, clamp of the quotient to `C - S` before narrowing). `internal/checked` is not extended: the need is one package's and shared leaves stay small. Source: RES 4.6, 24 ("`math/bits` for the 128-bit products"); architecture.md 3.5.
- **D383.** Three root files, one concept, area RES: `resources.go`, `resources_check.go`, `resources_tick.go`. `architecture.md` 2.6 and the `AGENTS.md` Layout gain the two continuation names in the same change (Phase 1, the supervisor applies it, and replaces the rule tag). Source: architecture.md 2.6 (continuation files), N-243.
- **D384.** Module state is `resourcesState` in `resources.go`, held as `World.res *resourcesState` (nil while off), dense arrays indexed `region*nResources + resource`: `stock`, `carry`, `exhaustedRun`, `capScale`, `regenScale` (saved), and per-tick `snapStock`, `snapAbundance`, `requested`, `granted`, `drawn`, `regenerated`, `lost`, `adjusted`, `denied`, `reduced` (scratch; written by P2.6, P2.7 and the between-ticks calls, read by P8.3 and P8.4, and cleared at P8.6 with the other per-tick counters of the modules, `spec-world.md` 6.10, so a call between ticks counts in the next tick). Counters are kept whether or not an observer is attached, so the position hash needs no `skipFields` entry. Source: RES 3.2.2, 6.4, 14.1; architecture.md 2.10.1; poshash D177.
- **D385.** The abundance of the snapshot is the single input of the effective yield for the tick. `yieldTable.effective` keeps its signature and stays abundance-neutral (it is called by `warmJob`, `viability.go` and the anchors); a new method `effectiveAt(sh, region, recipe, season, sited, abundance)` passes the sixth term to `production.Effective`, whose fifth argument is the abundance already. The abundance of a recipe is the least over its draws (RES 4.7.4) and is 10 000 for a recipe with no draw. The cost anchors at P1.2, the view at P2.5 and the start at P2.7 call `effectiveAt`; P1.2 reads the live stock, equal to the snapshot of P2.3 because nothing but the between-ticks slots changes a stock between P8.3 and P2.3 (RES 4.7.3), and a test asserts the equality (FR-RES-048). Source: RES 4.7.3 to 4.7.5, 5.5.1; `spec-production.md` 6.3.
- **D386.** Ration flow. P2.5 is unchanged for the strategy: `admitJob` derives `k0` ignoring draws (no cap on the `startReq`), and for a recipe with draws records one `endowment.Request{Executions: k0, Draw: d}` per draw in `resourcesState` scratch, in the order P2.5 walks the rotation; `Ref` is the agent ID, the requests are sorted by it before the ration, and the grants slice is scratch sized by the world and indexed by agent slot. P2.6 rations each endowment whose requests exceed the snapshot stock (a sum otherwise) and stores the grant per (agent, draw). P2.7 sets `startReq.drawCap` to the least over the draws of `min(grant_j, floor(live_j / d_j))` and the limit list gains one limit after the inputs with the new code `limitNoEndowment` mapped to `ReasonNoEndowment` (position 8 of the master order, after `no_input`); `start` then takes each draw by `takeEndowment`. Clipping to the live stock before `start` makes the take all-or-nothing without a second count: a stock lowered between P2.6 and P2.7 refuses with `no_endowment` and takes nothing (FR-RES-018, 022). Source: RES 5.2, 5.4.1; `spec-production.md` 4.2.1; `spec-world.md` 6.4.1.
- **D387.** The shock slots and the adjustment are built here as unexported methods of `resourcesState` (`setCapacityScales`, `setRegenScales`, `adjustEndowments`), each recomputing the effective values, clamping and counting `lost` and `adjusted`, and emitting `EndowmentAdjusted` stamped with the tick the next step runs (`spec-observability.md` 3.3.5). Plan 17 wraps them in the public calls and guards them against a call in a step. Source: RES 6.3; N-091; FR-RES-029, 030.
- **D388.** Public surface: `World.Endowment(RegionID, ResourceID) int64` (0 for an unknown pair or a module that is off; callable on a failed world, FR-WLD-110), `ErrInsufficientEndowment`, `ErrUnknownResource`, `ErrResourceShape`, `ErrCurve` (FR-RES-032), and `ResourceID` in `value.go`. `takeEndowment` is unexported. Source: RES 5.1, 13.2.
- **D389.** Regeneration is `p83Regen`, called in `positionsTail` between `p82Flows` and `p84Audit`; it regenerates every endowment in (`RegionID`, `ResourceID`) order with the season `Calendar.SeasonAt(tick)`, advances `exhaustedRun`, logs the `Warn` and emits `EndowmentChanged`; the scratch stays until P8.6 (D384). The identity of 6.4 is asserted by `auditEndowments`, called from `p84Audit` under `WithAudit` on the scratch P8.3 left, which reports endowment, tick and difference (FR-RES-025). Source: RES 4.1, 6.4, 14; `spec-world.md` 6.10.
- **D390.** Hash field ids (appended, never moved; the rows owed to `spec-config.md` 8.2.2 are in Phase 1): `Modules.Resources` 5; the module type, named `Resources` (the registry scope of `doc/specs/README.md`; `registry_test.go` accepts only built Go type names, as plan 5 D303 did for `Demand`), `Resources` 1, `Endowments` 2, `HorizonYears` 3; `Resource` `ID` 1, `Shape` 2, `RatePerYear` 3, `RateBP` 4, `Refuge` 5, `Capacity` 6, `InitialBP` 7, `PlanShareBP` 8, `Abundance` 9, `Seasons` 10 (`Label` has none); `AbundanceKnot` `Share` 1, `Factor` 2; `SeasonRegen` `Season` 1, `BP` 2; `Endowment` `Region` 1, `Resource` 2, `HasCapacity` 3, `Capacity` 4, `HasInitial` 5, `Initial` 6, `HasRegenScale` 7, `RegenScaleBP` 8; `Recipe.Draws` 9 (the next free id of `Recipe`, written only while the module is on) with `Draw` `Resource` 1, `Quantity` 2. The pointer is written even when the struct is zero. `pairsModules` writes the module pairs in ascending hash id, so pair 5 goes between the pairs of ids 3 or 4 and those of ids 6 or 8. Source: `spec-config.md` 8.2.1 to 8.2.3.
- **D391.** Validation lives in `resources_check.go`: `validateResources` (13.1, 13.2, hooked where plan 5 hooks the first module's validator), the `Draws` checks of a recipe (`ErrModuleDependency`, `ErrDuplicateID`, `MaxDraws`, `ErrUnknownResource`, the overflow of `spec-production.md` 10.3 and the capacity times the largest shock scale), the case of a draw above the capacity is the new warning `draw_impossible` (a separate rule; the `unrunnable_recipe` rule of `validate.go` is not extended), and the warnings `resource_unused`, `endowment_unused`, `abundance_not_monotone`, `regen_per_tick_high`, `resource_inert`, `raw_source_exhaustible`; `no_comparative_advantage` (D395) has its own call in `warnings(c)` because it is reported with the module off. `canonical()` sorts resources by ID, knots by share, seasons in the calendar's order, endowment rows by (region, resource), draws by resource. Source: RES 13; `spec-config.md` 3.4.
- **D392.** The view group `Endowments []EndowmentView` (`Resource`, `Stock`, `Capacity`, `AbundanceBP`) is appended to `AgentView`, for the agent's own region only, for the endowments its role's recipes and its effective own-use list draw on, sorted by resource; `fillAgentView` calls `fillEndowments` under `w.res != nil`. `RecipeView.MaxExecutions` includes `floor(A/d)` of the snapshot for each draw with reason `no_endowment` (the view only; admission ignores draws, D386). Source: RES 5.5.1; FR-RES-041; N-106.
- **D393.** Snapshot: `Snapshot.Endowments []EndowmentRow` (`Region`, `Resource`, `Stock`, `Capacity`, `AbundanceBP`, `ExhaustedRun`, `CapacityScaleBP`, `RegenScaleBP`, both 10 000 when not shocked, the names of `spec-observability.md` 5.3.6; `EndowmentAdjusted` and the checkpoint keep `CapacityBP`, `RegenBP`) in (`RegionID`, `ResourceID`) order; `Names.Resources []string` appended; `modulesOn` appends "resources" after the names already present (build order, plan 5 rule; architecture.md 2.10.5). Both are empty or zero while off, so the golden rendering does not move. Source: RES 14.4; `spec-observability.md` 3.2, 5.3.6; architecture.md 2.10.5.
- **D394.** Checkpoint section 15 holds, for each endowment with a configured capacity above 0 in (`RegionID`, `ResourceID`) order, `Stock`, `Carry`, `ExhaustedRun`, then the shock scales that are not 10 000 as (region, resource, `CapacityBP`, `RegenBP`), sparse; present when `Modules.Resources` is non-nil. A restore rebuilds everything else (the effective values, the snapshot and the grants are recomputed). Source: RES 15; `spec-checkpoint.md` 4.2.
- **D395.** The yield-table values are doc-only work (FR-RES-035): the parameter-owner rows move (Phase 1), no code changes `Yields`, and there is no climate entity. `no_comparative_advantage` (RES 7.4.1, FR-RES-036) is a core warning, reported with the module off, in `resources_check.go`, over the strongly connected components of the graph. Source: RES 7.1, 7.4.
- **D396.** The solver gets one row per (resource, region): `sum d(res, r) * x(r, rho) <= Phi(res, rho)` with `Phi` exact rational, computed in `internal/steady/sustain.go` (the only package that may use `math/big`, architecture.md 3.5) from the plan stock; the abundance at the plan share multiplies the mean factor `E` of the recipes that draw (11.2). It reports `Headroom`, `Rent`, the error `resource_short` in place of `labour_short` when the binding row is a resource, `endowment_runway`, and the warning `endowment_season_peak`. Source: RES 11; FR-RES-039, 040; `spec-viability.md` 3.2, 4.8.
- **D397.** The producibility predicate of RES 9.2 (an endowment that can start once) replaces the structural count of a `Sited` source with `Draws` in the viability existence test; `spec-config.md` 7.5 keeps counting it as sourced (FR-CFG-045). Source: RES 9.2, 9.3; FR-RES-037, 038.
- **D398.** Overlays and scenarios. Four files, because a region-keyed table naming `five`'s regions fails on `one`: `resources.toml` and `resources-binding.toml` (pack `one`, region `north`: slack 2.2 times and binding 1.1 times the draw of the reference programme, RES 18.2) and `resources-five.toml`, `resources-five-binding.toml` (the five regions, starting from the hand-derived table of 18.2 and replaced by the solver's flows). `initial_bp` equals `plan_share_bp` (RES 11.4). This plan runs S42, S43, S44 and the identity half of S45 (through the helpers of D387); S41 needs POP and is owed to plan 11, the public-call half of S45 to plan 17. Source: RES 18, 19.4; `spec-reference-chain.md` 4.5.
- **D399.** `endowment.Overshoot(Plan) bool` and `endowment.UnstablePlan(Plan) bool` implement the integer inequalities of RES 8.3.1 over a `Plan` struct of primitives (`Shape`, effective annual rate in basis points, plan share, elasticity in basis points, `r*`, `dR`, `dX`); the elasticity comes from `curve.Segment`. Plan 11 fills `r*`, `dR`, `dX` from its schedules and reports the warnings when the population module is on. Source: RES 8.3.1; FR-RES-044.

## Phases

### Phase 1: the edits owed before code

Goal: the specs are true before the code starts (the contract's rule). Files: specs and rules only; the supervisor applies them. Nothing in the repository other than this file changes in this phase. Gate statement: the resources overlays add no commodity, recipe, role or region, so `TestPacks_IncomeBalance` (plan 8 D370) does not run on them; the packs they sit on keep passing it, and a change to those packs repeats the runbook.

| File and section | Edit |
| --- | --- |
| `spec-resources.md` 24 (sketch) | `Key` is `ordering.Key`; `Regen` takes `Params` and `Step`; `Ration` sorts in place (D381) |
| `spec-resources.md` 3.3, `spec-production.md` 3.1, `spec-config.md` 3.7.2 | name the type `Draw{Resource string; Quantity Quantity}` (RES 3.1.3 speaks of "pairs"; `spec-production.md` 24 says "`Draw` is the resources module's") |
| `spec-config.md` 8.2.2 | the rows of D390 |
| `spec-resources.md` 3.3 and 24, `spec-config.md` 8.2.2 | the module type is `Resources`, not `ResourcesConfig` (the registry scope; plan 5 D303 did the same for `Demand`) |
| `spec-observability.md` 4.1.7 (RecipeRefused) | the `RecipeRefused` row gains the count `NoEndowment`, as the note in 4.2.1 says a later module appends its reason's count |
| `spec-config.md` 4.1 | `MaxKnots` 8 and `MaxDraws` 4 are in the table but not in `bounds.go`; this plan appends both (D380) |
| `spec-resources.md` 3.4 and FR-RES-005 | FR-RES-005 omits the facility term `B` of 3.4; say "with `B` 0 while no facility" |
| `spec-resources.md` 5.2 step 2 and 5.5.1 | `k0` of admission ignores draws; the view's `MaxExecutions` includes `floor(A/d)`: say both once (D386, D392) |
| `spec-resources.md` 8.5 item 2 | `no_fixed_factor` is POP's; plan 11 owns the edit of `spec-population.md` 5.2 and 9.2 |
| `doc/specs/README.md` parameter-owner table | the rows of RES 7.1.3 (values of `YieldProfile`, `Yields`, `Draws`) already read as 7.1.3 says; check they match and that the registry rows of 12 exist |
| `architecture.md` 2.6 and `AGENTS.md` Layout | add `resources_check.go`, `resources_tick.go` (D383); new rule tag |
| `architecture.md` 2.10.1, 2.10.2 | no edit here: covered by plan 5 E1 items (a) and (b). The three touches this plan relies on are `effectiveAt` in `production.go`, `fillEndowments` called from `fillAgentView`, and the `tick.go` calls of P2.3, P2.6 and P8.3; `effectiveAt` is plan 5 E1 (a); the `fillEndowments` call is a fan-in call site and the P2.3, P2.6 and P8.3 calls are `tick.go` catalogue entries, both covered by the general wording of E1 and 2.10.2, so nothing is owed by plan 5 |
| `doc.go` (before any code uses them, architecture.md 3.2) | terms: Knot (a point of a piecewise-linear curve; also names the root type `Knot` of plan 11 (`config.go`), so the glossary test passes once and the term is added once), Resource, Endowment, Stock, Capacity (of an endowment), Shape, Carry (of a regeneration), Refuge, Abundance, Abundance curve, Ration, Draw, Plan share, Sustainable draw, Finite stock; and, because `TestGlossary_DeclaredTypesNamed` needs every exported type named, the names `Resources`, `Resource`, `AbundanceKnot`, `SeasonRegen`, `Endowment`, `Draw`, `EndowmentChanged`, `EndowmentAdjusted`, `EndowmentRow`, `EndowmentView`, `ResourceID` |
| `spec-world.md` 5.6.1 item 4, `spec-viability.md` 8.5.3 (open question 13 / WLD 15) | a warm job draws no endowment and its pending output reads the neutral abundance; accept, because the warm stock is `InitialBP` (RES 11.4) |
| `spec-stability.md` | register numbers for RES-A1 to A7, D1 to D6 (the register owner assigns; the plan uses RES-S1 to S5 = S41 to S45 as RES 19.4 says) |
| `spec-observability.md` 4.1 and 7.3 | the rows of the two events and the `Warn` "endowment exhausted" exist; verify (N-126) |
| `spec-reference-chain.md` 4.5 and 7.6 | list the four resources overlay files of D398 (`resources.toml`, `resources-binding.toml`, `resources-five.toml`, `resources-five-binding.toml`) beside `overlay/resources.toml` |
| `spec-observability.md` 3.2.4 | `Names.Resources` is appended after `Names.Wants` (plan 5 O8); the field order of `Names` is build order, not the order the list now shows |

Exit: each row done or recorded as an open item in Risks; `git diff --stat` shows only spec, rule and doc files.

### Phase 2: `internal/curve` and the bounds

Goal: the shared evaluation exists with its tests. Files: `internal/curve/curve.go`, `curve_test.go`, `fuzz_test.go`; `bounds.go` (the bounds of RES 4.6 and `MaxKnots`, `MaxDraws`).

Tests: `TestEval_InteriorTruncatesTowardZero` (a rising and a falling segment, the values of RES 16.1: `9 000 + trunc(1000*3321/5000) = 9 664`), `TestEval_FlatOutsideAndAtKnots`, `TestEval_SingleKnot`, `TestEval_OverflowReported`, `TestCheck_Rejections`, `TestNonDecreasing`, `TestSegment_BoundaryBelongsToOneSegment`, `FuzzEval` (never panics; the result lies between the neighbouring `Y` values; monotone input gives monotone output). `Requirement:` FR-RES-013, FR-POP-004, NFR-RES-003. Bounds test: `TestBounds_ResourceProducts` asserts the products of RES 4.6 fit 128 bits and `D` fits int64 (FR-RES-008, FR-CFG-046).

Exit: `go test -short -race ./internal/curve`, Lizard on the file.

### Phase 3: `internal/endowment`

Goal: every rule of RES 3.4, 4, 5.3 and 8.3.1 as pure functions with exact tests. Files: `shape.go` (`Shape`, `Params`, `Step`, `Result`), `regen.go`, `wide.go`, `effective.go` (3.4), `abundance.go`, `ration.go`, `plan.go` (D399), and their tests.

Tests, with the expected values from `doc/specs/models/res/` (`regen.py`, `worked.out`, `allocate.out`; the files say how each number was made, testing.md 5.5):
- `TestRegen_NumeratorsOfTheThreeShapes` (FR-RES-008), `TestRegen_LogisticWRoundsUp` (FR-RES-009: a stock of 1 and a stock one below capacity regrow), `TestRegen_CarryIsExact` (FR-RES-010: 400 random runs against a `math/big` oracle in the test), `TestRegen_ClampBanksNothing` (FR-RES-011, including `C = 0`), `TestRegen_FiniteExtinctRefuge` (FR-RES-031), `TestRegen_ArableYearsOfWorkedOut` (FR-RES-007, 012: the spring stocks 1 700, 1 748, 1 766, 1 773 of A1), `TestRegen_MaxBoundsDoNotOverflow` (RES 4.6: numerators near 10^31, `D = 10^17`), `TestRegen_SeasonProfileComplete` (FR-RES-012).
- `TestEffective_CapacityAndScale` (RES 3.4), `TestAbundance_CurveShareAndMinimum` (FR-RES-013, 014, and `C = 0` gives 0).
- `TestRation_CoversAllWhenStockCovers`, `TestRation_ProportionalThenLeftovers` (FR-RES-017, the example of 5.3.7: grants A 3, B 2, C 2, D 2, E 1), `TestRation_NeverAboveRequestOrStock` (FR-RES-020), `TestRation_IndependentOfListingOrder` (FR-RES-020), `TestRation_CohortEqualsAgents` (FR-RES-021, `Guard:` for the fixed mix of sizes of 5.3.4, whose bias is known and bounded), `TestRation_SkipsWhatDoesNotFit`, `TestKey_IsTheSplitMix64Finaliser` (FR-RES-019, against `ordering.Key`).
- `TestOvershoot_RefillAndLogistic`, `TestUnstablePlan_Shapes` (FR-RES-044: the worked check of 8.3.1, no warning at 5 000 bp and equality at 1 600 bp).
- `FuzzRegen` (stock in [0, C], identity of one tick, never panics), `FuzzRation`, `FuzzAbundance` (NFR-RES-003); `BenchmarkRegen`, `BenchmarkRation` with `b.ReportAllocs()` and `TestRation_AllocsPerRun` at 0 (NFR-RES-001, 004).

Exit: `go test -short -race ./internal/endowment`, coverage at least 90%, Lizard (the shapes are separate helpers so no function passes complexity 10), and the mutation pass for the package (Docker, `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`, `--workers 4`, per `~/.claude/CLAUDE.md`).

### Phase 4: configuration, validation, hash, errors and the loader

Goal: the module and its draws can be written, validated, hashed and loaded; with the module off nothing moves. Files: `resources.go` (types, `DefaultResource`, `DefaultResources`, `canonical` hook, deep copy), `resources_check.go`, and appends to `config.go`, `bounds.go`, `defaults.go`, `validate.go`, `hash.go`, `errors.go`, `value.go`; `config/resources.go`, `config/mapping.go` (the `draws` key and the module dispatch plan 5 built); `registry_test.go` (delete the entries), `config/guard.go` if it lists keys.

Tests: `TestValidate_ResourcesBoundsAndShapes` (FR-RES-034, 032: each error with its sentinel and path, such as `modules.resources.resources.woodland.abundance[2].share`), `TestValidate_EndowmentRows` (FR-RES-004: `ErrUnknownRegion`, `ErrUnknownResource`, `ErrDuplicateID`, absence is no warning), `TestValidate_DrawsNeedTheModule` (FR-RES-001, FR-PRO-007), `TestValidate_DrawOverflow` (13.2 `ErrOverflow`), `TestValidate_SeasonProfile` (FR-RES-012: `ErrUnknownSeason` with no seasons), `TestWarnings_ResourceRules` (FR-RES-033: each warning fires and none fails `Validate`), `TestWarnings_NoComparativeAdvantage` (FR-RES-036: the cross ratio of 11 000 bp, the example of `spec-reference-chain.md` 4.4.4 passes), `TestHash_ResourcesFieldIds` (FR-RES-004, FR-CFG-073: also asserts the module pairs ascend by id with pair 5 between the pairs of ids 3 or 4 and 6 or 8, as `TestHashConfig_ModulePairsAscending` does for the modules already built; golden bytes for the one-region row with `Modules.Resources: &Resources{}`, `0005 0000`, and the unchanged all-off pin), `TestHash_DrawsOnlyWhileOn`, `TestCanonical_ResourceOrder` (FR-RES-003), the loader tests in `config/` (round trip of `testdata/chains/overlay/resources.toml`, unknown key, bad shape, a draw of an unknown resource), `FuzzValidateResources` (never panics, valid input is stable under `canonical`), and the registry test with the entries of the Catalogue table deleted (FR-RES-034).

Exit: `TestHashConfig_ReferencePinned` and `testdata/golden/off/` unchanged (git shows no diff there).

### Phase 5: the state, the snapshot, the ration seam and the take (P2.3, P2.6, the starts)

Goal: a recipe with `Draws` is limited by a ration and takes its draw. Files: `resources.go` (construction, the view group), `resources_tick.go` (`p23Snapshot`, `p26Ration`, `takeEndowment`, `World.Endowment`), `production.go` (`startReq.drawCap`, `limitNoEndowment`, `effectiveAt`), `phase_production.go` (the request collection in `admitJob`, the cap at `startJob`; the `k == 0` branch of `startJob`, marked excluded from coverage as unreachable, is now reachable through `limitNoEndowment`: remove the mark), `strategy_view.go` and `fanin.go` (`fillEndowments`, the `Endowments` group), `production_report.go` (anchors through `effectiveAt`; the `refusedNoEndowment` column after `refusedNoInput` in the `refusalTally` and its case in `refusalColumn`, which panics on any reason it does not count), `construct.go` (build `res`), `world.go` (`res` field), `tick.go` (positions P2.3 between `p21Complete` and `p25Work`, P2.6 between `p25Work` and `p27Start`; delete both from `notBuiltPositions`; add both to `positionList` with owner RES and `line`), `event.go` (`ReasonNoEndowment`, counted: `RecipeRefused.NoEndowment` and its `flush` mapping), `poshash_internal_test.go` (two `replayRows` with `on: w.res != nil`) and `observe_internal_test.go` (a `stepSites` row for P2.3 and one for P2.6, in position order, or `TestCallSites_TheTableIsTheStraightLine` fails), `event_test.go` (delete `no_endowment` from `deferredProduction`).

Tests: `TestSnapshot_FixedForTheTick` (FR-RES-015: `Work` results do not depend on call order), `TestStart_GrantCapsExecutions` (FR-RES-022, `no_endowment` after `no_input`), `TestRecipeRefused_CountsNoEndowment` (an observer attached, a refusal under `no_endowment` is tallied and does not panic), `TestStart_AllOrNothing` (FR-RES-018: the stock lowered between P2.6 and P2.7, nothing taken), `TestTake_FailsPastStock` (FR-RES-016), `TestEndowment_Read` (unknown pair, off module, failed world), `TestDraw_NotReturnedOnAbandon` (FR-RES-023), `TestDraw_RegionChangeKeepsDraw` (FR-RES-023, internal fixture), `TestCohortGetsProportionalShare` (FR-RES-021 through the world: 20 cohorts of weight 50 against 1 000 agents of weight 1), `TestLedgerUntouched` (FR-RES-024: the ledger audit and the reason registry unchanged), `TestView_OwnRegionEndowmentsOnly` (FR-RES-041; the view of a role with no draw has the group empty), `TestEffectiveYield_IncludesAbundance` (FR-RES-048: the anchor at P1.2, the view and the start agree), `TestNoRandomWord` (FR-RES-045: the stream of a world with the module on and off, with no recipe drawing, is identical), `TestPositions_Table` (testing.md 6.8: the three RES positions in order), `TestReplayRows_Guard`.

Exit: `go test -short -race .`; `testdata/golden/off/` unchanged; `TestPositionHash_Golden` unchanged.

### Phase 6: regeneration, shocks, events, the `Warn`, the audit, the snapshot rows

Goal: P8.3 and everything it reports. Files: `resources_tick.go` (`p83Regen`, `setCapacityScales`, `setRegenScales`, `adjustEndowments`, `auditEndowments`), `resources.go` (snapshot rows, `namesOf` and `modulesOn` appends in `construct.go`), `event.go` (the two events, `AdjustKind` constants, `isEvent`), `snapshot.go`, `tick.go` (P8.3, and the clear of the scratch at P8.6), `poshash_internal_test.go` (the `replayRows` row for P8.3, `on: w.res != nil`, after P8.2) and `observe_internal_test.go` (the `stepSites` row for P8.3, after P8.2), `internal/logkeys/logkeys.go` (`Resource`, `Stock`, `Short`; `logkeys_test.go` is not edited here, only the rule table row check is), `phase_exit.go` (`p84Audit` calls `auditEndowments`), `event_test.go` and `render_test.go` (delete the two events from the unbuilt lists), the catalogue rows.

Tests: `TestRegen_RunsAfterDrawsInOrder` (FR-RES-007), `TestEvent_EndowmentChangedFields` (FR-RES-026, 027: only when a count is non-zero, `Requested - Granted == Short`, `Drawn <= Granted`, `AgentsDenied` and `AgentsReduced`), `TestEvent_NoMapNoPointer` (FR-RES-050, a reflection check like the other events), `TestEvent_AdjustedKinds` (FR-RES-029, 030: scale, clamp and the excess in `Lost`, delta clamped to [0, C], tick stamp D387), `TestWarn_EndowmentExhausted` (FR-RES-028: tick 1 and every tenth consecutive tick, once per endowment per tick, `ExhaustedRun` resets), `TestIdentity_EveryTick` (FR-RES-025 over `one` with the slack and the binding overlays and `scenarioSeedCount()` seeds), `TestAudit_ReportsIdentityBreak` (a stock corrupted after P8.3 is reported by P8.4 with endowment, tick and difference, which needs the scratch to survive P8.3), `TestScratch_ClearedAtP86` (a shock between ticks counts in the next tick), `TestSnapshot_EndowmentRows` (FR-RES-047), `TestObservationDoesNotChangeResults` (FR-RES-046), `TestNoopObserverAllocates0` (NFR-RES-001, `testing.AllocsPerRun`), `BenchmarkP83` (NFR-RES-004), `TestCatalogue_ResourceEvents` (N-126).

Exit: `go test -short -race .`, off goldens unchanged.

### Phase 7: the checkpoint section and the fork

Goal: section 15 saves and restores. Files: `resources.go` (`saveResources`, `loadResources` over `codec.Writer` and `codec.Reader`), `checkpoint_sections.go` (one row at tag 15 in `checkpointSections`, tag 15 deleted from `notBuiltSections`), `checkpoint_restore.go` (the cross-checks of the section in `checkpointCross`: a stock above the capacity, a run above the ticks, a scale above `MaxShockBP`, a row naming an unknown endowment), `checkpoint_fork_test.go` use, `testdata/checkpoints/` (no all-modules fixture exists before plan 24, plan-2-2 D259; `testdata/checkpoints/` is not touched here).

Tests: `TestCheckpoint_ResourcesRoundTrip` (FR-RES-042: stocks, carries, runs, scales; a stock at 0 with capacity above 0 is saved and restored as 0, not as the starting stock), `TestCheckpoint_SectionAbsentWhenOff`, `TestFork_Resources` (through `forkAtN(t, forkCase)` of plan-2-2 D258 with N in {0, inside a job, inside a season, after a call between ticks}; a world forked at tick N on `resources-binding` with a stock mid-regeneration, a pending carry and a non-default scale, run on, equals an uninterrupted one in every event and in `Snapshot`; N inside a fallow season and N inside a start window), `TestFork_ShockBeforeCheckpoint`.

Exit: the fork test and `FuzzDecodeResources` (corrupt bytes give an error, never a panic).

### Phase 8: the sustainable draw in the solver

Goal: the resource rows, their reports and the predicate of 9.2. Files: `internal/steady/sustain.go` (exact `Phi` per shape), `program.go` (`Model.Resources`, the extra rows in `Build`), `activity.go` (`Activity.Draws`), `results.go` (`Outcome.ResRent`, `ResTight`), `viability.go`, `viability_table.go`, `viability_check.go`, `viability_report.go` (`Report.Headroom`, `Report.Rent`, `endowment_runway`, `resource_short`, `endowment_season_peak`, the raw predicate), `validate.go` (`raw_source_exhaustible` is in Phase 4; its sourcing test is here).

Tests: `TestSustain_PhiPerShape` (FR-RES-039: the three rows of 11.1, example B's 313 against 309 units a tick), `TestSolve_ResourceRowBinds` (FR-RES-040: `Headroom` below 1, `Rent` above 0, `resource_short` and not `labour_short`), `TestSolve_RowSlack`, `TestSolve_OffUnchanged` (the module off builds the identical programme: `Size` and `Build` outputs equal), `TestRunway_FiniteStock` (RES 16.2: 16.9 years), `TestSeasonPeak_Warning` (RES 11.3, 16.1), `TestExistence_DrawBeyondCapacity` and `TestExistence_StockCannotStart` (FR-RES-037), `TestRawSourcing_Predicate` (FR-RES-038), `FuzzSustain` (exact, never panics), `TestNoFloatingPoint` (`internal/steady/float_guard_test.go`) and `TestViabilityFiles_NoFloatingPoint` still pass with the new files.

Exit: `internal/steady` coverage at least 90%, mutation pass on the new file, `viability_*` goldens (`viability_agree_pack_test.go`) unchanged with the module off.

### Phase 9: the overlays, the golden and the scenarios

Goal: the module runs on the reference chain. Files: `testdata/chains/overlay/resources.toml`, `resources-binding.toml`, `resources-five.toml`, `resources-five-binding.toml`; `testdata/golden/resources/` (`one.txt`, `events-one.txt`, `one.positions.txt`); `golden_test.go` additions, `scenario_s42_test.go` to `scenario_s45_test.go`, `second_reference_test.go` (the two examples of RES 16).

Overlay contents: `arable` (`refill`, 10 000 bp a year, `plan_share_bp` 5 000, `initial_bp` 5 000) for `grow_staple` draw 1, `deposit` (`constant`, rate 0) for `dig_mineral` draw 3, `stand` (`logistic`, 3 000 bp a year, a refuge, `initial_bp` 6 000) for `cut_fuel` draw 14 (RES 18.1); the line gatherers stay unlimited. Capacities come from the draw a year of `SolveSteadyState` on the pack as plan 8 leaves it, the derivation in a comment at the top of each file: `arable` (`refill`) `draw / (load * rate * (1 - share))` (RES 18.2); `deposit` (`constant`, rate 0, so no division by the rate) `draw * HorizonYears * load`, the `Stock_0 / HorizonYears` term of FR-RES-039; `stand` (`logistic`) the draw that is the maximum sustainable one, `RateBP * C * sig * (1 - sig)` at the plan share, solved for `C` through `SolveSteadyState`. Only the `arable` overlay carries an `Abundance` curve; `deposit` and `stand` have none, so S43's run-down stays within 10% of `endowment_runway` (16.2: 13.8 years against 16.9 with a curve on). If a slack run does not keep `lambda* >= 1`, the overlay is wrong and not the code.

Tests: `TestGolden_OneResources` (the module alone, `-update` only by flag, testing.md 7.2), `TestPositionHash_Resources`, `TestScenarioS42_LogisticAboveAndBelowMaximum` (S42 to S45 use `scenarioSeedCount()`; the stock holds below `F_max` and collapses above it; without a refuge it stays 0 and the `Warn` is bounded; with one it recovers; the role fails by the ladder; FR-RES-009, 028, 031), `TestScenarioS43_FiniteDeposit` (run-down within `s43.runway_bp` 10% of `endowment_runway`, FR-RES-040), `TestScenarioS44_RationWithCohorts` (1 000 agents of weight 1 and 20 cohorts of weight 50 on a scarce `refill` stock; grant per household within `s44.grant_band_bp` of the mean, no agent denied more than `s44.denied_ticks`; FR-RES-017, 021), `TestScenarioS45_BlightAndDrought` (a stock set to 5% and a regeneration scale 3 000 through the D387 helpers on each shape; `refill` recovers, `logistic` with no refuge does not, `constant` as its rate allows; the identity holds in every tick; FR-RES-025, 029, 030, 031), `TestConfigOnly_SecondSettingWithResources` (FR-RES-049, architecture.md 9.4: the example B pack of RES 16.2 loads, validates and runs with no Go specific to it, invariants of testing.md 6.3 held), `TestInvariants_ResourcesOn` (goods and money conserved, FR-RES-024). The module-alone world (`one` + `resources.toml`) is added as a row to `invariantTable` (REF 7.5, 8.2), using `scenarioSeedCount()`, and a new `TestDeterminism_Resources` takes its cases from `detCases(true)` by name.

Exit: the goldens are recorded once, reviewed, and `off/` shows no diff.

### Phase 10: the exit gate

Goal: the row is done (the contract's gates). Run, in this order: `gofmt -l .` (empty), `go vet ./...`, `staticcheck ./...`, `go test -short -race ./...`, `go mod tidy` (no diff), `lizard -l go -C 10 -a 8 -L 100 -w` on every new non-test file, then markdownlint (plan-2-1 D239) on the files Phase 1 changed, then gate R and gate F of [plan-2-1.md](plan-2-1.md) D229 (until its Phase 2 is done, `go test -short -race ./...` and F alone), coverage at least 90% for the root and for `internal/curve`, `internal/endowment` and `internal/steady`, the mutation pass for `internal/curve` and `internal/endowment` and the new file of `internal/steady` (Docker with `--cpus=4 --memory=16g --memory-swap=16g -e GOMAXPROCS=4`), the registry and catalogue tests with the entries of the table below deleted from their `notBuilt` lists, and the proof that `testdata/golden/off/` and `TestHashConfig_ReferencePinned` did not move. Update `roadmap-v1.md` Stage 2 row 9 (the supervisor).

## Catalogue appends

| File | Appends | Position, id or tag | Fixed by |
| --- | --- | --- | --- |
| `config.go` | `Modules.Resources *Resources`; `Recipe.Draws []Draw`; types `Resources`, `Resource`, `AbundanceKnot`, `SeasonRegen`, `Endowment`, `Draw` | `Modules` field 5 in the hash, appended after the pointers plans 5 to 7 added in Go order (the hash id does not follow Go order); `Recipe.Draws` id 9 | `spec-config.md` 8.2.1, 8.2.2, 3.1 |
| `bounds.go` | `MaxResources`, `MaxEndowments`, `MaxEndowment`, `MaxRegenRatePerYear`, `MaxRegenBP`, `MaxRegenScaleBP`, `MaxSeasonRegenBP`, `MaxShockBP`, `MaxHorizonYears`, `MaxKnots`, `MaxDraws` | after the last bound of plans 5 to 8 | RES 4.6; `spec-config.md` 4.1 |
| `defaults.go` | `DefaultResource()`, `DefaultResources()` (`HorizonYears` 50) | after the last default | RES 12 |
| `validate.go` | one call to `validateResources`; the `Draws` checks in the recipe pass; one call to the `no_comparative_advantage` check in `warnings(c)`, outside the module guard | where plan 5 hooks the first module | RES 13; `spec-config.md` 7.1.6 |
| `hash.go` | `Modules.Resources` pair, the `Draws` pair while on, the pair functions of D390 | ids of D390 | `spec-config.md` 8.2 |
| `errors.go` | `ErrUnknownResource`, `ErrResourceShape`, `ErrCurve`, `ErrInsufficientEndowment` | after the last sentinel | RES 13.2, 5.1 |
| `event.go` | `EndowmentChanged`, `EndowmentAdjusted`; the adjust kinds; `ReasonNoEndowment`; `Names.Resources` | after the last event of earlier plans; `Names.Resources` after `Names.Wants` (build order, plan 5 O8); the reason after `ReasonNoInput` in the master order | RES 14.1; `spec-observability.md` 4.1.7, 3.2 |
| `option.go` | none | none, because the module has no option | RES 14 |
| `tick.go` | `p23Snapshot`, `p26Ration`, `p83Regen`, each guarded by `w.res != nil` | P2.3, P2.6, P8.3 in the order of `spec-world.md` 6.4 and 6.10 | `spec-world.md` 6 |
| `snapshot.go` | `Snapshot.Endowments []EndowmentRow` | after the last row type | RES 14.4 |
| `ledger.go` | none | no reason, no boundary account | RES 6.2.1, FR-RES-024 |
| checkpoint | section 15 | tag 15, written when `Modules.Resources` is non-nil | `spec-checkpoint.md` 4.2 |
| log keys | `resource`, `stock`, `short` in `internal/logkeys` | the order of `observability-and-logging.md` 3.3.2; `logkeys_test.go` is not edited, only the rule table row check is | that table |
| metrics | none in the core; the four instruments of RES 14.3 are plan 22's | none | RES 14.3 |
| `value.go` | `ResourceID` | after the last typed ID | RES 3.2.3 |
| tests | delete from `notBuilt` in `registry_test.go`: `Modules.Resources`, `Recipe.Draws`, `Resource`, `Resources`, `Endowment`, `AbundanceKnot`, `SeasonRegen`; from `event_test.go` and `render_test.go` the two events; from `deferredProduction` `no_endowment`; from `notBuiltPositions` P2.3, P2.6, P8.3 | | the staged form, testing.md 6.7, 6.8 |

## Cross-plan contract

Consumes:

- `production.Effective` with its abundance term (`internal/production/yield.go`) <- Stage 1
- `ordering.Key` (`internal/ordering/ordering.go`) <- Stage 0
- `checked.MulDiv`, `checked.MulDivRem` (`internal/checked/checked.go`) <- Stage 0
- `yieldTable`, `startReq`, `productionScratch.maxExecutions`, `production.Count` (`production.go`) <- Stage 1
- `fillAgentView`, `p25Work`, `p27Start`, `positionList`, `notBuiltPositions`, `p84Audit` (`fanin.go`, `phase_production.go`, `tick.go`, `phase_exit.go`) <- Stage 1
- the reference programme `steady.Build`, `steady.Model`, `Outcome` (`internal/steady`) <- Stage 1
- `codec.Writer`, `codec.Reader`, `checkpointSections`, `notBuiltSections` (`checkpoint_sections.go`), `checkpointCross` (`checkpoint_restore.go`), `forkAtN`, `forkCase` (`checkpoint_fork_test.go`) <- plan-2-2 (D243, D258)
- `NewGraph`, `Graph.Component`, `Graph.Components` (`regions.go`) <- plan-2-3
- the module hooks as plan 5 built them: the validator call site, `modulesOn`, the `[modules]` dispatch of `config/mapping.go`, the module-alone golden helper (`demand.go`, `config/mapping.go`, `golden_test.go`) <- plan-2-5 (its Provides names plans 6 and 7; plan 5 adds plan 9)
- `TestPacks_IncomeBalance` (`viability_pack_test.go`) and the rebalance runbook (plan 8 D370): Phase 9 runs the runbook for the resources overlays (they change draws, not the census) and the overlays are not run through the gate, which covers the packs; a re-derived capacity (Risks 3) repeats the runbook <- plan-2-8
- the rebalanced `five.toml`, `warm-five.toml`, `one.toml`, `warm-one.toml` that the overlays sit on (`testdata/chains/`) <- plan-2-8
- gate R and gate F, `-update` refused under `-race`, the race-aware seed count (`D226`, `D228`, `D229`) <- plan-2-1

Provides:

- `curve.Knot`, `curve.Eval`, `curve.Check`, `curve.NonDecreasing`, `curve.Segment` (`internal/curve`) -> plan-2-11
- `MaxKnots` (`bounds.go`) -> plan-2-11
- `endowment.Overshoot`, `endowment.UnstablePlan`, `endowment.Plan` (`internal/endowment/plan.go`) -> plan-2-11
- `testdata/chains/overlay/resources-binding.toml` and `resources-five-binding.toml` (S41 and STY-W02 run on them) -> plan-2-11
- `testdata/chains/overlay/resources.toml` and `resources-binding.toml` -> plan-2-19
- `testdata/chains/overlay/resources.toml` -> plan-2-17
- `testdata/chains/overlay/resources.toml`, `resources-five.toml` and the binding pair -> plan-2-24
- `testdata/golden/resources/` (built in Phase 9) -> plan-2-24
- `ResourceID` (`value.go`) and `Names.Resources` (`event.go`) -> plan-2-17, plan-2-22
- `World.Endowment` (`resources_tick.go`) -> plan-2-17
- `resourcesState.setCapacityScales`, `setRegenScales`, `adjustEndowments` (`resources_tick.go`) -> plan-2-17
- `Recipe.Draws`, `Draw` (`config.go`) -> plan-2-10
- `ReasonNoEndowment` and the `Endowments` group of `AgentView` (`event.go`, `strategy.go`) -> plan-2-10
- `EndowmentAdjusted` (`event.go`) -> plan-2-17
- `EndowmentChanged`, `EndowmentAdjusted` and the log keys `resource`, `stock`, `short` (`event.go`, `internal/logkeys`) -> plan-2-22
- `Report.Headroom`, `Report.Rent` (read as `SteadyState.Report.Headroom`), warning key `endowment_runway` (`viability_report.go`) -> plan-2-19
- `ErrUnknownResource` (`errors.go`) -> plan-2-17
- `Snapshot.Endowments`, `EndowmentRow` (`snapshot.go`) -> plan-2-17, plan-2-22
- `Modules.Resources` (type `Resources`), `testdata/chains/overlay/resources.toml` (`config.go`, `testdata/`) -> plan-2-10, plan-2-11
- `MaxShockBP` (`bounds.go`) -> plan-2-16 (declared here, so plan 16 does not append it)
- `EndowmentScale` and `EndowmentAdjustment` are not built here: plan 17 declares them (its Risks 1), and the helpers of D387 take primitives (`region`, `resource`, a basis-point scale or a signed delta) for it to wrap

## Testing

Entry for the PR (testing.md 11). Requirements touched and the tests that cover each:

| Requirement | Tests |
| --- | --- |
| FR-RES-001, 034 | `TestValidate_DrawsNeedTheModule`, `TestValidate_ResourcesBoundsAndShapes`, the registry test |
| FR-RES-002, 003 | `TestCanonical_ResourceOrder`, `TestLedgerUntouched`, the config-only test |
| FR-RES-004, 032 | `TestValidate_EndowmentRows`, the loader tests, `TestHash_ResourcesFieldIds` |
| FR-RES-005, 006 | `TestEffective_CapacityAndScale`, `TestEndowment_Read`, `TestIdentity_EveryTick` |
| FR-RES-007 to 012, 031 | the `TestRegen_*` family, `TestRegen_RunsAfterDrawsInOrder` |
| FR-RES-013, 014, 048 | `TestAbundance_CurveShareAndMinimum`, `TestEval_*`, `TestEffectiveYield_IncludesAbundance` |
| FR-RES-015 to 023 | `TestRation_*`, `TestKey_IsTheSplitMix64Finaliser`, the `TestStart_*`, `TestTake_*`, `TestDraw_*`, `TestCohortGetsProportionalShare` |
| FR-RES-024, 025, 045, 046 | `TestLedgerUntouched`, `TestIdentity_EveryTick`, `TestAudit_ReportsIdentityBreak`, `TestNoRandomWord`, `TestObservationDoesNotChangeResults` |
| FR-RES-026 to 028, 050 | `TestEvent_*`, `TestWarn_EndowmentExhausted` |
| FR-RES-029, 030 | `TestEvent_AdjustedKinds`, `TestScenarioS45_BlightAndDrought` |
| FR-RES-033, 036 to 038 | `TestWarnings_*`, `TestExistence_*`, `TestRawSourcing_Predicate` |
| FR-RES-035, 043 | Phase 1 (doc rows), `TestScenarioS44_RationWithCohorts` |
| FR-RES-039, 040 | `TestSustain_PhiPerShape`, `TestSolve_ResourceRowBinds`, `TestScenarioS43_FiniteDeposit` |
| FR-RES-041, 047 | `TestView_OwnRegionEndowmentsOnly`, `TestSnapshot_EndowmentRows` |
| FR-RES-042 | `TestCheckpoint_ResourcesRoundTrip`, `TestFork_Resources` |
| FR-RES-044 | `TestOvershoot_RefillAndLogistic`, `TestUnstablePlan_Shapes` (the call is plan 11's) |
| FR-RES-049 | `TestConfigOnly_SecondSettingWithResources` |
| NFR-RES-001 to 004 | `TestNoopObserverAllocates0`, `TestRation_AllocsPerRun`, the fuzz targets, the benchmarks; NFR-RES-002 by `TestRation_CoversAllWhenStockCovers` (one sum) |
| FR-POP-004 (shared) | `TestEval_*` |
| REF 7.5, 8.2 | the resources (`one` + `resources.toml`) row in `invariantTable` and a new `TestDeterminism_Resources` that takes cases from `detCases(true)` by name, with its own full seed count in the row, reduced by `invariantSeeds` of plan 1 (S42 to S45 keep `scenarioSeedCount()`) |

Golden files: `testdata/golden/off/` and the pinned hash do not move (the proof is a clean `git diff` on them at the gate and `TestGolden_OneOff`, `TestPositionHash_Golden`); `testdata/golden/resources/` is new, recorded once from the `resources.toml` overlay; `testdata/golden/all/` is not touched here (the all-modules fixture is plan 24's). Fork-at-N: `TestFork_Resources`. Fuzz: `FuzzEval`, `FuzzRegen`, `FuzzRation`, `FuzzAbundance`, `FuzzValidateResources`, `FuzzSustain`, `FuzzDecodeResources`, with `Fix:` seeds in `testdata/fuzz/`. Registry rows: every field of RES 12 (N-016). Catalogue rows: the two events, the reason, the three positions. Guards: `Guard:` on the fixed-mix bias of the ration (5.3.4) and on the `D` bound. A reduced seed count under `-race` is `scenarioSeedCount()` (plan-2-1 D226).

## Observability

Entry for the PR (observability-and-logging.md 9).

- **Events.** `EndowmentChanged` (class A, once per endowment per tick at P8.3, only when `Drawn`, `Regenerated`, `Short` or `Lost` is non-zero, no map, no pointer) and `EndowmentAdjusted` (class C, per changed endowment of a call). Both are built only when an observer is supplied; the counters they read are kept regardless (D384).
- **Warn.** "endowment exhausted" with `tick`, `region`, `resource`, `stock`, `short`, at the first tick of a run of ticks with `Short` above 0 and every tenth after, bounded by regions times resources. Without tenure data the condition is `Short > 0` (the tenure term is LND's). Configuration warnings are returned, not logged (RES 14.2).
- **Debug.** "draw rationed" (`tick`, `region`, `resource`, `quantity`, `short`) behind `Enabled`, per endowment per tick in which the ration cut. "recipe not started" with reason `no_endowment` goes through `refuse`, which this row extends with the count (Phase 5). No `Info`, no `Error`.
- **Log keys.** `resource`, `stock`, `short` (RES 14.2; `observability-and-logging.md` 3.3.2 already lists them).
- **Metrics.** None in the core; plan 22 maps the four instruments of RES 14.3 from the events, with attributes bounded by the configuration (`economysim.resource`, `economysim.region`).
- **Non-interference.** `TestObservationDoesNotChangeResults`; the no-op path allocates nothing (`TestNoopObserverAllocates0`).

## Risks and open items

1. **`resource_overshoot` and `resource_unstable_plan` have no `r*`, `dR`, `dX` until POP.** This plan builds the conditions (D399) and tests them on given numbers. Owner: plan 11 wires and reports them. Until then the registry row `resource_overshoot` has a unit test and no scenario.
2. **S41 and the public half of S45.** S41 (population on the binding overlay) is plan 11's, the public shock calls are plan 17's; both need `resources-binding.toml`, which this plan ships. Owner: plans 11 and 17.
3. **The overlays depend on plan 8's rebalanced packs.** If plan 8 changes a draw or a census after this plan sizes the capacities, the capacities are re-derived (a rerun of Phase 9, no code change). Owner: plan 8 tells plan 9.
4. **A warm job reads the neutral abundance** (`warmJob` calls `effective`), so a pack started below capacity has a first cohort of pending outputs at 10 000. Phase 1 records the acceptance; the effect is visible only with a curve and a poor `InitialBP`. Owner: WLD and VIA (WLD open item 15, VIA 8.5.3).
5. **Bias of the ration for a fixed mix of request sizes** (RES 5.3.4, open question 4) is known and bounded by the leftover; S44 measures it with a mixed mix and the result decides whether a carried remainder is owed. Owner: RES, after S44.
6. **The between-ticks `Tick` of `EndowmentAdjusted`** is the tick the next step runs (OBS 3.3.5); a shock applied before tick 0 is stamped 0. Plan 17 must guard the call against a step in progress. Owner: plan 17.
7. **128-bit helper kept local** (D382). If a second package needs a 128-bit by 64-bit product, move it to `internal/checked` then. Owner: whoever needs it.
8. **A recipe that draws a resource and is also `TraineeEligible`** takes the executor's region's endowment and the ration counts the request once (RES 5.4.4); `spec-skill-barriers.md` is silent on a trainee that cannot start for want of a grant. Owner: SKL.
9. **Registry numbers for RES-A1 to A7, D1 to D6** are the register owner's (RES 19). Owner: supervisor.

Note proposals (text only; the supervisor is the only editor of `doc/notes.md`, N-103):

- **N-new-a (areas: POP, RES).** `internal/curve` exists since plan 9; plan 11 must call `curve.Eval` for every schedule and not write a second evaluation, and must fill `endowment.Plan` (`r*`, `dR`, `dX`) from its schedules to report `resource_overshoot` and `resource_unstable_plan` when the population module is on (RES 8.3.1, FR-RES-044).
- **N-new-b (areas: INT, RES).** The unexported helpers `setCapacityScales`, `setRegenScales`, `adjustEndowments` exist since plan 9 with their events; INT wraps them and must refuse a call while a step is running and stamp the tick the next step runs.
- **N-new-c (areas: REF, STY).** `overlay/resources*.toml` exist in four files, slack and binding for `one` and `five`; S41 runs on the binding pair with the population overlay, and the capacities are derived from the solver's flow, not by hand (RES 18.2).
- **N-new-d (areas: VIA, WLD).** A job created at construction reads the neutral abundance (Risks 4); decide whether the warm start of a stock below capacity needs the abundance of `InitialBP`.
