# Spec: Viability, the steady state and the warm start

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-VIA-NNN` and
`NFR-VIA-NNN`, which tests cite (testing.md section 4). They are in section 16.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs `spec-goods.md` (the commodity fields), `spec-production.md` (recipes, jobs,
yield, tools, `UnitCost`) and `spec-welfare-and-wages.md` (upkeep, the welfare
ratio, the target-income rule). It also reads `spec-beliefs.md` (the seed-price
seam, 8.4), `spec-calendar.md`, `spec-scale.md`, `spec-estates.md`,
`spec-ledger.md`, `spec-currency.md` and `spec-config.md` (the validation
machinery, 7). `spec-agents.md` is written at the same time by another author;
section 3.3 states exactly what this spec assumes of it.

**Every number in this spec is a placeholder.** Nothing has run in Go. The numbers
in section 15 come from a prototype of the equations written in Python with exact
fractions (`doc/specs/models/via/`, run in Docker). The prototype shows the
equations are well posed on the two chains of `spec-production.md` section 12. It
is not the implementation, its chains are completed with invented recipes and
prices, and none of its results is a default or claimed to be stable (N-015,
N-027). Section 12 lists every placeholder.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

A chain can be well formed (every reference resolves, every commodity can in
principle be made) and still be impossible to live in: a seed cycle that returns
fewer seeds than it takes, a crop whose job is so long that a household's labour
cannot feed one household, a staple that rots before the next harvest. This spec
answers two questions:

1. **Can the chain sustain itself?** Each cycle of production must leave a
   surplus, and a household's labour must be able to produce its upkeep
   (roadmap, tier 1; `spec-config.md` 7.6). That is **viability**: a precise
   property of configuration data (section 4), checked partly at load and partly
   by a solver.
2. **Where does it sit when it does?** The **steady state** is the set of prices,
   flows and stocks at which the chain, with households working, keeps itself going
   with nothing changing (section 5). It is used to calibrate a chain, and to start
   a run there: the **warm start** (section 8), so that a run begins near
   equilibrium and not from beliefs that are wrong by design (`spec-beliefs.md`
   8.4; `vision.md` section 5, item 1).

It covers: the definition of viable and the layers of the check (4); the steady
state, its equations and their caveats (5); the integer arithmetic (6); where the
solver lives (7); the warm start and its interface to the world (8); regions and
trade (9); the rules added to `spec-config.md` 7 (10); parameters (11); what
viability does not prove and the scenarios handed to `spec-reference-chain.md`
(13).

Out of scope: the dynamics of the approach to the steady state, and whether the
steady state is stable (`spec-stability.md`); demand beyond upkeep (`spec-demand.md`);
transport (`spec-transport.md`); who chooses a recipe each tick (`spec-agents.md`);
the experiment harness that sweeps parameters (`spec-experiments.md`, a client of
this spec).

## 2. Positions taken

Each design pressure of the brief, with the position and the reason. Later sections
give the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What does viable mean? | Four layers, cheapest first: **V1 sourced and runnable** (every needed good can be made somewhere with a positive yield, every role can work where it lives); **V2 productive cycles** (every cycle of the material graph leaves a surplus: Hawkins-Simon); **V3 labour sufficient** (at full labour, the realm can produce `lambda >= 1` times its households' upkeep); **V4 storable** (a good needed every tick is not out of season for longer than it keeps). V1, V2, V4 and a pooled V3 are errors; per-region V3 and the rest are warnings | Each layer is a testable statement over configuration data. The error checks are *necessary* conditions and optimistic (free trade, best case) so that no pack is rejected for a flaw a trade arrangement could cure |
| 2 | What is the solver? | An exact linear programme: maximise the multiple `lambda` of the upkeep basket the realm can produce, subject to goods balance, household labour and a per-recipe start capacity. Its dual is the price vector and the wages | In a constant-returns economy with fixed-proportion upkeep, a competitive equilibrium is an optimum of this programme and its dual (section 5). One method gives viability, labour requirement, surplus capacity, prices and flows |
| 3 | Core or calibration tool? | **Core**, as an explicit pure function `SolveSteadyState`, not called by `NewWorld`. The cheap, bounded part runs inside `Validate`. The experiment harness is a client | The warm start is a version 1 done criterion a consumer must reach through the public API (architecture.md 1.3); CFG needs part of it at load; it is a pure computation over `Config` (no I/O, no goroutines) |
| 4 | Arithmetic | Exact: integers of unbounded size (`math/big`, standard library) in a fraction-free tableau, Bland's rule for the pivot. No floating point. The result is rounded to `int64` fixed point once, at the boundary, by stated rules | Determinism on every platform (architecture.md 7.4); rounding happens once, so it can be audited |
| 5 | When it does not converge | Return an error and no table. Termination is guaranteed by Bland's rule, so the only causes are a pivot cap or a cancelled context. The answer is certified in exact arithmetic or not returned | Report, never guess |
| 6 | What does the warm start hand the world? | A plain-data section of `Config`: seed prices per region and commodity, per-group starting money and stock per household, and per-group initial jobs as a share of agents in progress. The world spreads them over agents with one fixed rule, with no randomness | BEL 8.4 expects a seed table; jobs, stock and money share its source and its hash |
| 7 | Multi-region | Regions are coupled only by explicit lanes (data, supplied by the transport spec when it exists). Default is no lane: autarky. Free trade is a comparison tool and the optimistic bound for the load check | Transport is not specified; a hook is enough now |
| 8 | Where is the price level set? | By the configured reference basket: relative prices come from the solver, the level from `B_r` | `spec-currency.md` 5: the level is a calibration anchor; no inflation by design |
| 9 | What does viability not claim? | Nothing about dynamic stability, demand beyond upkeep, or that the simulation reaches the solver's point | Section 13 |

## 3. The model the solver sees

### 3.1 Household labour and activities

A **household** supplies at most one **working tick** per tick (`spec-production.md`
5.1.2: one job at a time). An execution of a recipe of `Duration` `d` occupies one
household for `d` ticks (`spec-scale.md` 4.2). So the **labour** of an execution is
`d` household-ticks, whatever the weight of the agent that runs it.

An **activity** is a recipe in a region. A recipe that is on an own-use list (`spec-subsistence.md` 3.2) and on no role's `Recipes` is **not an activity**: the steady state assumes that the markets work, and the subsistence floor is for when they do not (`spec-subsistence.md` 3.6, FR-SUB-037). A recipe that is on both lists is an activity as any other. For recipe `r` in region `rho` the solver
computes, with integer arithmetic (section 6):

- the **start capacity** `J(r, rho)`: the number of jobs one household can start in a
  year of `TicksPerYear` ticks, counted greedily from tick 0 of the year: start at
  the first tick whose yield factor is positive (`spec-production.md` 6.2), be busy
  for `d` ticks, repeat. **A start counts at every tick `t < TicksPerYear` of the open
  window, whether or not the job completes inside the year**: the steady state is
  periodic, so a job begun late in one year completes early in the next, and the annual
  flow of a stationary seasonal activity is its starts per year, the year boundary being
  arbitrary. For a recipe with no seasonal profile this is
  `TicksPerYear / d` rounded up (7 for a 60-tick job in a 365-tick year), so `J * d`
  may exceed the year by up to `d - 1`; for a crop that can be started only in spring, it
  can be 1;
- the **mean factor** `E(r, rho)`: the mean of the effective factors
  (`spec-production.md` 6.3, with shock 10 000 and trainee 10 000) at those starts,
  in basis points. It is 0, and the activity does not exist, when `J = 0`;
  For an activity with `Draws` (`spec-resources.md` 11.2) `E` is also multiplied by the abundance at the plan share of its resource, the least over its draws, a constant, so the programme stays linear;
- the **net vector** `n(r, rho)`, per execution and per commodity `c`:
  `E * q_c / 10 000` for each non-discard output, minus the input quantity, minus
  `WearPerUse / Durability` for each required tool (a tool is a flow of wear, not a
  stock). A discard output counts for nothing. The entries are rational numbers
  with denominators dividing `10 000 * Durability`; the solver keeps them exact.

The **upkeep demand** of region `rho` for commodity `c` is
`D(c, rho) = sum over groups g in rho of Households(g) * u(g, c)`, where `u(g, c)`
is the household upkeep of the group's role (3.3).

All flows are **per tick, averaged over one year**. The solver does not model the
path through the seasons; it sees the annual mean. The effect of seasons on
storage is the separate check of 4.5.

### 3.2 The reference programme

For a set of regions `P` (one region for autarky, all regions for the pooled
case), let `x(r, rho) >= 0` be the executions per tick of activity `(r, rho)`, and
`lambda >= 0` the multiple of the upkeep basket. The **reference programme** is:

```
maximise  lambda
subject to
  goods    for each commodity c:  sum over (r, rho) of x(r, rho) * n_c(r, rho)  >=  lambda * D(c, P)
  labour   for each region rho:   sum over r of d(r) * x(r, rho)               <=  phi * H(rho)
  capacity for each activity:     x(r, rho)  <=  H(rho) * J(r, rho) / TicksPerYear   (only if J * d < TicksPerYear)
  resource for each endowment (res, rho):  sum of d(res, r) * x(r, rho)  <=  Phi(res, rho)    (only with the resources module on: `spec-resources.md` 11.1)
  facility for each (role group g, facility class f) with `Needs`:  executions of the recipes that need f by g  <=  cap(g, f)   (only with the infrastructure module on: 4.2.2)
  x >= 0
```

`H(rho)` is the number of households in `rho` (the sum of the weights of its
groups). `phi` is the **available work share**: 1 (every tick) for the viability
test, and the household-weighted mean of
`ExpectedWorkTicks / TicksPerYear` for the **nominal** case (`spec-welfare-and-wages.md`
4.2). Lanes (section 9) add columns; they change no row.
With the infrastructure module on, a route or ground facility at full condition enters as the gain of its `BoostBP` on the transit factor of the lanes (9.4) and on the sustainable draw of the endowment (`spec-resources.md` 11), and the facility row above is the sum of the `cap` of the facilities a group holds (`spec-infrastructure.md` 13.4); `Decay` and repair are not in the programme (4.2.2).
The resource row is the sustainable draw of an endowment: its dual is `Rent`, the value of one more unit of draw a tick (`spec-resources.md` 11.1, 11.3). With the module off there is no row and nothing changes. With an outside (`spec-external-sector.md` 12.1) the programme gains import and export columns and one trade row (4.8); with neither the programme is the closed one above.

**Wants as final demand (round 5; `spec-demand.md`).** When the demand module is on, the
goods row gains a fixed term, `>= lambda * D(c, P) + W(c, P)`. `W(c, rho)` is the demand
for `c` from the wants at the reference: for each group, each want in its role's ladder
that has a taste row, `Households * TasteScale * RegionScale * Quantity / Period` (in
basis points of 10 000 each, kept as an exact fraction), split among the want's items in
proportion to their weights, which is the substitution share at equal price ratios
(`spec-demand.md` 4.2). `W` is **not** multiplied by `lambda`: the basket is the
subsistence part and the wants are final demand at their reference level, so `lambda*`
says how many baskets the realm can make *after* meeting the wants. With the module off
`W = 0` and nothing changes (FR-VIA-055). The fixed term is an exact fraction
(`Model.Fixed` is a slice of `Ratio`, one per goods row; an integer cannot hold it), and
the goods row is written -n x + D lambda <= -W, so a positive term gives a negative
right-hand side. The solver then runs a **first phase** (an artificial variable for each
such row, Bland's rule in both phases, artificials that end in the basis at 0 pivoted
out); the certificate of FR-VIA-014 is checked against the original programme. A first
phase that ends above 0 means the wants cannot be met at any `lambda >= 0`: the solver
returns `steady.ErrInfeasible`, and the check treats it as a case of
`wants_exceed_labour` with no `SurplusWantsBP`, the census falling back to the
upkeep-only programme (FR-VIA-057). With every term 0 no right-hand side is negative,
the first phase does not run and the result, pivots included, is that of the programme
without a term. The census and the warm-start table of a world
with wants come from this programme; the error rule `labour_short` (V3) does **not**: it
keeps the upkeep-only programme, because a want is not subsistence and a pack must not be
rejected for a luxury it cannot afford (FR-VIA-056). The diagnostic of 10.2,
`wants_exceed_labour`, says when the wants are what makes `lambda*` fall below 1.

The optimum `lambda*` is the **surplus factor**: how many times the upkeep basket
the realm can make with the labour it has. It is reported in basis points as
`SurplusBP = floor(lambda* * 10 000)`. The chain is **productive at full labour**
when `lambda* >= 1` with `phi = 1`.

The **labour share to subsist** is `LabourShareBP = ceil(10 000 * 10 000 / SurplusBP)`
at `phi = 10 000`, defined when no capacity row is tight (otherwise the labour
requirement is not simply proportional and it is reported as absent). It is the
share of their ticks that households must work to make their own upkeep.

### 3.3 What this spec assumes of other specs

Stated exactly, because `spec-agents.md` is written at the same time.

1. **Upkeep.** Each group (a role in a region) has a household upkeep `u(g)`: a list
   of (commodity, whole units per tick), owned by AGT, with the regional additions
   of `spec-currency.md` 5.3. The upkeep is the only final demand in version 1
   (DEM is later). If AGT gives upkeep per role only, `u(g)` is that list.
2. **Recipes per role.** `Role.Recipes` (`spec-production.md` 3.3) lists the recipes
   a role may run. The solver treats the activities of a region as open to any
   household of the region (free occupational choice); the role lists matter only
   for the checks that name a role and for assigning jobs in the warm start. An own-use list (`spec-subsistence.md` 3.2) is not read as a role list: a recipe only on it is no activity (3.1), and the floor is left out of the programme, which describes the market chain.
3. **Starting state.** An agent is born with money, an inventory, an optional job,
   belief centres at the seed price and a weight; its two creation draws (the target
   ratio and the residue) come **after** the warm start is applied, and the pending
   output of a warm job is fixed with a residue of 0 before them
   (`spec-world.md` 5.6, 5.7; `spec-production.md` 6.6.2, `spec-welfare-and-wages.md` 5.1). The role carries
   defaults for money and inventory (AGT's `StartMoney`, `StartStock`); the warm
   start of section 8 overrides them for a group when it names the group.
4. **Weight.** The warm start gives amounts *per household*; the world multiplies by
   the agent's weight (`spec-scale.md` 4.2).
5. **Construction.** The world is built by creating regions, then estates, then
   agents (`spec-estates.md` 2.7.6), and may create goods with the ledger reason
   `initial`. Section 8.6 lists the ledger behaviour the warm start needs.

If AGT settles any of these differently, only sections 3.3 and 8 change.

## 4. What viable means

### 4.1 The layers

| Layer | Name | Statement | Kind | Cost |
|-------|------|-----------|------|------|
| V0 | Well formed | IDs, references, bounds, the producibility fixpoint | error | `spec-config.md` 7.3, 7.5 |
| V1 | Sourced and runnable | Every commodity in an upkeep or a recipe a role runs can be produced somewhere in the realm with a positive yield; every role with recipes has one that can run where its households live | error | graph, `O(commodities * recipes)` |
| V2 | Productive cycles | Every strongly connected set of commodities in the material graph has a surplus: some mix of its recipes makes more of every commodity than it consumes | error | exact programme per set, bounded |
| V3 | Labour sufficient | The reference programme (3.2) with `phi = 1` has `lambda* >= 1` (run when V1 and V2 found nothing, 4.4) | error pooled, warning per region | the programme, size-gated |
| V4 | Storable | A commodity of the upkeep is not out of production for longer than its shelf life | error (upkeep), warning (inputs) | calendar scan |

The layers are cumulative in cost, not in logic: each is checked on its own and
each failure names its rule (section 10).

### 4.2 V1: sourced and runnable

An activity exists in region `rho` when `J(r, rho) > 0` (3.1). Commodity `c` is
**producible in `rho`** if it is sourced there (an existing activity with no inputs
outputs it, or it is marked `Raw`: 4.2.1) or
some existing activity in `rho` outputs it with every input producible in `rho`
(the fixpoint of `spec-config.md` 7.5, restricted to existing activities). For a recipe with `Draws`, "exists in `rho`" has the conjunct of `spec-resources.md` 9.2: for every draw the endowment of `rho` can supply at least one execution. It is
**producible in the realm** if it is producible in some region (4.6 for trade).
The tools a recipe requires are inputs to this fixpoint.

An own-use-only recipe (3.1) makes no commodity producible for V1: an upkeep good that only such a recipe makes is `unproducible_in_realm` in the realm, because the floor is not a source the chain can stand on (`spec-subsistence.md` 3.6, FR-SUB-037). The structural pass of `spec-config.md` 7.5 may keep counting it; V1 refines, as it does for yield and season.

V1 fails, with the rule `unproducible_in_realm`, when a commodity of any group's
upkeep, or of the inputs or tools of a recipe that exists in a region where a role
lists it, is not producible in the realm. It fails with `role_cannot_work` when a
role that lists at least one recipe has households in a region where none of its
recipes exists. A role with no recipes (a trader, a carrier) is exempt.

4.2.1. **A good marked `Raw` is sourced, in V1 as in the structural check.** Round 3
read this spec as stricter than `spec-config.md` 7.5 (a raw good needed "an activity or an
endowment"); the two now say one thing: the marker `Raw` makes a commodity sourced
everywhere, as `spec-goods.md` 6.2 says it does until the specs that supply real sources
(resources, the external sector, faucets) exist. To keep the marker honest the rule adds a
warning, `raw_without_source` (10.2): a raw commodity that no existing activity with no
inputs outputs, and that no endowment, faucet or import supplies, has nothing real behind
the marker. Marking a good `Raw` therefore does not close a cycle silently: the
configuration loads, the warning names the good, and the cycle stands on the marker alone
(`spec-reference-chain.md` 6.3). The fixture opens its cycles with real sources.

A programme whose basket goods are **all** unsourced `Raw` markers (a `Raw` good is sourced everywhere, so the pack loads with the warning `raw_without_source`, and no real activity stands behind any of its basket) has nothing for V3 or the table to describe: V3 and the table are skipped for that programme. The same holds for a programme with an empty basket (4.4, 5.3 item 1).

The real sources, now written, are three (W1 of the wave 3 integration): a recipe with no inputs and no `Draws` (labour-limited), a recipe with `Draws` on an endowment that can start once (`spec-resources.md` 9.2), and an import line with a sell side (`spec-external-sector.md` 10.1). Either of the last two satisfies `raw_without_source`; the structural pass of `spec-config.md` 7.5 keeps counting the marker and a `Sited` source with `Draws` as sourced. An import line with a sell side is a real source: `raw_without_source` is not raised for a raw commodity it supplies, and for V1 it makes the commodity producible in the gate regions (`spec-external-sector.md` 10.1.2). A raw commodity whose only sources are finite endowments draws `raw_source_exhaustible` (`spec-resources.md` 9.3), and one sourced only by an import line draws `import_only_source` (`spec-external-sector.md` 10.1.2); both are warnings and neither changes V1.

The difference from `ErrUnproducible` (`spec-config.md` 7.5) is that V1 looks at
yield, season and region: a crop that grows nowhere is producible in the sense of
7.5 and not in the sense of V1.

4.2.2. **A recipe that needs a facility** (`spec-infrastructure.md` 13.4). With the infrastructure module on, a recipe with `Needs` is **runnable in a region** only where the pack has an initial facility of the class held by that role's agents (`spec-infrastructure.md` 3.4.2); with none, the activity does not exist in that region for V1, and the warm start creates no job that needs a facility that does not exist (`ErrNotViable`, as for a missing tool). Its activity is bounded by the sum of the `cap` of those facilities per execution of a start: one row for each (role group, facility class) in 3.2. With the module off nothing changes.

### 4.3 V2: productive cycles

A recipe that needs its own product (seed grain makes grain; a tool is made with a
tool) puts a cycle in the **material graph**: commodities as nodes, an edge from each
input and each required tool of an activity to each of its non-discard outputs.
Take each strongly connected component with a cycle. It is **productive** if the
programme

```
maximise s   subject to   sum over activities in the component of x * n_c  >=  s    for each commodity c in the component,
                           sum of x = 1,  x >= 0
```

has `s* > 0`, with `n` the net vector of 3.1 restricted to the component's
commodities. This is the Hawkins-Simon condition: there is a non-negative activity
vector whose net product is positive in every commodity of the component, equivalent
to every leading principal minor of `I - A` being positive for the component's
technology `A`. When each commodity of the component has exactly one activity the
test is the minors themselves, computed exactly. The best case over regions is
taken: an activity at its largest `E` in any region counts (the check is
necessary, not sufficient).

**Membership.** The activities of a component are every activity with a non-discard
output in the component, each with its net vector restricted to the component's
commodities. A **source activity** (one with no inputs) therefore belongs: a recipe
that makes a commodity of the cycle from nothing adds to the programme, and a chain
whose only supply of the cycle's commodity is such a source passes V2 (the cycle alone
has `s* = 0`; with the wild source it is productive).

A component that fails is `cycle_not_productive`. The commonest cause is a seed
cycle: one execution returns fewer seed units than it takes.

**The cycle gain.** The warning `cycle_cost_gain` (`spec-config.md` 7.6 item 3,
FR-CFG-044) needs the ratio `q_in / q_out` around a cycle, in exact integers, with the
**configured quantities** and no yield factor, because the ratio has no prices and the
factor depends on region and season. A component can hold exponentially many cycles, so
the cycles enumerated are the **simple cycles whose every commodity has one maker in the
component, a maker being a non-source activity (one with inputs) that outputs the
commodity**. A source activity is not counted, because a source supplies the commodity
from outside the cycle. Worked case, the seed cycle of `one`: `gather_seed` is a source,
`save_seed` is the one maker of `seed` and `grow_staple` the one maker of `staple`, so the
cycle is enumerated, and `(40 / 1 800) * (20 / 20) = 1/45`: 40 `seed` in and 1 800
`staple` out, then 20 `staple` in and 20 `seed` out, so it never warns inside the
`CostFloor` bound. The enumeration is bounded by `MaxViabilityCells` steps; a component
past the bound is the part "cycle gain" of the warning `viability_check_skipped`.

**The size gate.** Before a tableau is built its size is computed with the checked
helpers (`internal/checked`) and compared with `MaxViabilityCells`: the **rows** are the
commodities of the programme, plus its regions, plus one row per activity with
`J * d < TicksPerYear` (the capacity rows); the **columns** are the activities, plus one
for `lambda`, plus the slack of each row (Stage 1 has no lane column: the pooled form
merges the goods rows, 9.4). A product above `MaxViabilityCells`, or a solve that reaches
`MaxViabilityPivots`, is in `Validate` not an error but the warning
`viability_check_skipped`, whose message names the **part** that was not run: V2 of a
component, V3 pooled, V3 of a region, the nominal warnings, or the cycle gain. V1 and V4
are never skipped and are not programmes. In `SolveSteadyState` the same overshoot is
`ErrSolveLimit`, and its `SolveError` carries the pivots, rows and columns (5.4). The
cost is the size of the largest component; so validation time stays bounded
(NFR-CFG-001).

### 4.4 V3: labour sufficient

V3 is the reference programme with `phi = 1`. It fails when `lambda* < 1`: even with
every household working every tick, with free trade among regions, the realm cannot
make the upkeep. This is `labour_short`, an error.

**The gate.** V3 is run whenever pooled `lambda* < 1` and V1 and V2 found nothing: when V1 found `unproducible_in_realm` or V2 found `cycle_not_productive` the programme is `lambda 0` for a structural reason, which is already reported, and `labour_short` is not added to it. A programme with an empty basket is never an error in `Validate` (`no_upkeep` is the warning of `spec-config.md` 7.4.2): V3 and the programme of V2 are skipped silently for it, and V1 and V4 still run. A programme whose basket goods are all unsourced `Raw` markers is skipped the same way (4.2.1).

When the binding row of the programme at `lambda* < 1` is a resource row, the error is `resource_short` and not `labour_short` (`spec-resources.md` 11.3); the cause is reported.

It fails per region, in autarky, with the warning `region_not_self_sufficient`. A
region that is short in autarky is not a defective pack: a specialised world has
regions that import. The warning says the region depends on trade that the current
specs do not provide, so that the author knows (section 9).

The prototype shows how easily a chain fails V3 (section 15). Example A of
`spec-production.md` 12.1 as written needs 73 times the labour its households have:
a job of 120 ticks that yields 6 grain, against a household that eats a unit or
more each tick. The quantities there are placeholders, and the check is what finds
that they are not usable.

Two warnings use the nominal case (`phi` equal to the expected work share). Both are
**per region, from the region's autarky programme** (also when the pooled form is used
for V3), and a region with no programme (an empty basket, or no households) raises
neither:

- `subsistence_exceeds_expected_work`: the region's `LabourShareBP` is above the
  household-weighted mean `ExpectedWorkTicks / TicksPerYear` of its groups in basis
  points. Households would have to work more than their role expects just to
  subsist. **When the labour share is absent (3.2, FR-VIA-012: a capacity row is
  tight) the warning is not raised**: there is no quantity to compare.
- `target_above_surplus`: the household-weighted mean `TargetRatio` of the region's
  groups is above the `SurplusBP` of the region's **nominal** autarky programme
  (`phi` the expected work share). The targets cannot be met even if every spare unit
  of output goes to the households.

### 4.5 V4: storable

For each commodity `c` the **supply gap** is the longest run of ticks in the year
(cyclically) in which no existing activity in the region completes a job that
outputs `c`. It is computed from the same start windows as `J` (3.1) and the
durations. **The completions are those of the greedy start sequence of `J`** (3.1): a
start at `s` completes at `s + d - 1`, and the sequence repeats every year. A
completion tick at or after `TicksPerYear` (a job begun late in the year, which the
start rule counts) is taken modulo the year for the gap, because the steady state is
periodic: a window on days 340 to 364 with `d` 60 starts once, at 340, and completes at
tick 34 of the next year, one completion a year, gap 364. An activity whose start is open
on every tick of the year (no season at zero yield) counts as completing on every tick:
the group's households are spread over the `d` offsets (8.5.2, 8.4.3), so one of them
completes at each tick and a `d` of 2 or 12 makes no gap; the greedy sequence of one
household is read only for an activity with a closed tick. With no seasons, or an activity
that completes every tick, the gap is 0.

*Worked check.* `grow_staple` of `one` is open on days 59 to 150 (`defaults.go`) with
`d` 60: the greedy starts are 59 and 119, the completions 118 and 178, and the longest
cyclic run without one is 365 - 60 - 1 = **304**. Counting every open start tick as a
completion (completions 118 to 209) would give 365 - 92 = 273, and is not the rule. Two
plantings (spring and autumn, completions 118, 178, 302, 362) give 123.

If `c` is in the upkeep of a group in the region and its `ShelfLife` is
above 0 and below the supply gap, no stock can bridge the gap and the households
go without: `storage_gap`, an error. A commodity that no activity of the region
outputs has no supply gap there (it comes from another region, so the pooled readings
of 4.2 and 4.4 cover it) and is skipped, not given a gap of the whole year. For an input of an activity that exists all year
the same condition is the warning `storage_gap_input`.

For a non-perishable the gap is bridged by a stock, which the warm start provides
(8.4).

### 4.6 What is necessary and what is a solver result

| Check | Where | When | Result |
|-------|-------|------|--------|
| V0 | `Validate` | load | error |
| V1, V4 | `Validate` | load | error or warning; graph and calendar scan only |
| V2 | `Validate` | load | error; exact programme per component, bounded |
| V3 pooled | `Validate`, if the size gate allows | load | error |
| V3 per region, nominal warnings | `Validate`, same gate | load | warning |
| Prices, flows, census, stocks, jobs, the table | `SolveSteadyState` | on demand | a `SteadyState` and a `Report` |

The load checks are **necessary**: if they pass, a chain may still be unviable in
ways the solver or the run finds (section 13). They are **optimistic**: pooled
free trade and the best yield across regions, so that they fail only when nothing
could work. The solver adds what is not a yes or no: the numbers.

### 4.7 A finding: the closed economy and the target ratio

The solver exposes a fact about the defaults, and it is recorded here because it
bears on WEL, DEM and STB.

In a closed economy (no money faucet or sink, `spec-ledger.md` 7.3) the money of
households is conserved, and each household's money changes by its income (net of
inputs) minus its spending. If upkeep is the only thing households buy and they buy
all of it, summing over households and over a window in which holdings are
unchanged on average gives: total income equals total upkeep spending. So **the
household-weighted mean welfare ratio is 1**, whatever `TargetRatio` says, until
households buy something beyond upkeep (`spec-demand.md`), money enters from a
boundary (`spec-money-supply.md`, `spec-external-sector.md`) or a pool absorbs it.
A `TargetRatio` of 15 000 (the default, `spec-welfare-and-wages.md` 4.2) is then
unreachable by every role at once: the target-income rule never tells an agent to
rest, and every household works every tick, producing the surplus `lambda* - 1` that
nobody buys. The surplus is a glut waiting to be priced to nothing.

This is not an error of this spec; it is an identity. It is the reason for two
checks: `target_above_surplus` (the targets cannot be met physically) and the
statement in section 13 that viability says nothing about demand. The note N-054
carries it to WEL, DEM and STB. `spec-welfare-and-wages.md` 7.1 and 7.2 accept it: the
role economics check is a **report and never an error** (`WageBelowTarget`, 10.3), the
default `TargetRatio` stays 15 000 and is read as an aspiration that the whole
population cannot meet until spending beyond upkeep or a money faucet exists, and
whether the default should fall to 10 000 meanwhile is left open there. AGT accepts the
glut and bounds it by the production halt (`spec-agents.md` 11.7). The money-supply module now exists (`spec-money-supply.md`): a balanced pair of grant and levy moves money between classes and leaves the mean ratio unchanged, and an unbalanced faucet raises it only while the money supply grows (`spec-money-supply.md` 8.2, 9.1); the finding is otherwise unchanged.

**A steady state with employers is not a zero-profit state** (`spec-employment.md` 22 item 7). The programme prices every activity at cost with labour at the wage dual, as if every household worked for itself. With the employment module on, a hired household's output is shared by the worker share `phi` and the employer keeps `1 - phi` of the margin, so the prices of a pack with employers sit above the solver's by that margin. The programme is not changed for it, and the warm start has no hire in flight (8.1).

### 4.8 The open layer (EXT) and the resource rows (RES)

Two optional layers extend the programme and change no check above when their modules are off.

- **Resources.** One row per endowment (3.2) with `Phi` computed in exact rational arithmetic from the shape, the plan stock and the season mean (`spec-resources.md` 11.1), and the abundance at the plan share in `E` (3.1). A pack whose census draws more than `Phi` fails `resource_short` (4.4). The solver reports `Headroom` and `Rent` and the informational `endowment_runway` (10.3); the warning `endowment_season_peak` compares the draws of a start window with the stock (10.2). `HorizonYears` is a field of the resources module and is read from the configuration; it is not a solve option (11).
- **The open layer.** `spec-external-sector.md` 12.1 asks for import and export columns, one for each line side present, bounded above by the mean capacity per tick and priced at the quote at tick 0 with the gate's scale, and a **trade row**: imports times their quotes at most exports times their quotes, so the realm does not finance an unbounded deficit. The inputs come from a pure read function of the configuration, `Lines()`, in a deterministic order, which the solver may call without a world; the warm start applies nothing to counterparties. Writing the layer is this spec's work (a note proposed by EXT) and is not done here: until then the solver is closed and the corridor is not priced.

The unmodelled part is the same for both: the realised price in a run is the midpoint of two limits, and the stock a run settles at is decided by the loop of `spec-resources.md` 8; the programme holds the plan stock and the quotes.

## 5. The steady state

### 5.1 Definition

A **steady state** of the configuration, for a given census, is a set of flows and
prices that (a) satisfies the reference programme at `lambda = 1` with every
commodity's net production equal to upkeep demand (a commodity in excess has price
at the floor), (b) gives every active activity in a region the same value added per
working tick, the **wage**, so no household gains by changing recipe, and (c) has no
activity that would pay more than the wage. Conditions (b) and (c) are the dual
constraints of the programme (5.2).

**The flows are those of `lambda = 1`.** The programme maximises `lambda`, but a steady
state is the programme at `lambda = 1` (condition (a)): the executions per tick `x` that
the jobs (8.5.1), the census (5.2) and the lean reserve (8.4.1) use are `x* / lambda*`,
the optimum scaled down, which keeps every row satisfied. Prices and wages are duals and
are unchanged by the scaling. With alternative optima, Bland's rule picks the vector
(FR-VIA-022).

### 5.2 Unknowns and equations

| Unknown | Meaning | Where it comes from |
|---------|---------|---------------------|
| `x(r, rho)` | executions per tick | primal variable |
| `lambda` | surplus factor | primal variable |
| `p(c)` | price of commodity `c` (relative, then scaled) | dual of the goods rows |
| `w(rho)` | wage per working tick in `rho` | dual of the labour row |
| `g(r, rho)` | rent of a tight capacity row | dual of the capacity rows |

Dual feasibility, for every activity: `sum over c of p(c) * n_c(r, rho)  <=  d(r) * w(rho) + g(r, rho)`
(value added per execution cannot exceed the wage for the job's duration plus the
capacity rent), with equality where `x > 0` (complementary slackness). Normalisation:
`sum over c of p(c) * D(c) = 1` when `lambda* > 0`, so prices are in units of the
regional basket. This is the zero-profit condition with the wage endogenous: it says
what each recipe must pay per working tick, and the wage is whatever equalises that.

**The census.** Version 1 has no population dynamics (POP, SPN come later), so the
households of a group are data. The solver treats the census as given and reports,
per recipe, the labour the steady state needs and, per group, the **imbalance**: the
households in the group against the households its recipes need. It does not change the
census. The same programme with the role populations free is the **balanced census** of
the report (`Report.Census`), which is a suggestion to the author: how many households
per recipe would clear every market.
When a recipe belongs to more than one role of the region, the census is reported per
recipe only, flagged ambiguous.

**The formula.** With `x` the executions per tick of lambda 1 (5.1), the **need** of a
group `g` of a region, in households, is the larger of two figures: the labour of its
recipes, `sum over r of d(r) * x(r)`, and the start capacity of any of them,
`max over r of x(r) * TicksPerYear / J(r)` (the households that must exist for the
capacity row of 3.2 to hold; a recipe without a capacity row has no such term). The
balanced census of the region is the apportionment of its configured households
`H(rho)` over the groups in proportion to `need(g)`, by the largest-remainder method
(quotas `H(rho) * need(g) / sum of need`, floors first, the rest to the largest
remainders, ties to the earlier group in canonical order: `architecture.md` 7.6); the
imbalance of a group is its configured households minus its balanced figure. The
available work share `phi` divides every term alike and cancels in the proportion, so it
is not in the formula; a reading that divides the labour terms by `phi` and not the
capacity term does not fit the worked case below. A group's need is taken from the flows
the group runs: a recipe on the lists of several roles of the region is divided as 8.5.1
says.

*Worked case, `testdata/chains/one.toml` (600 households).* At lambda 1 the flows per tick
are: `grow_staple` 0.76705 (`1 800 x - 20 s = 1 350`, `40 x = 20 s`), `save_seed` 1.5341,
`prepare` 75, `make_tool_a` 0.76705, `make_tool_b` 9.375, `refine` 9.8864, `dig_mineral`
16.477, `cut_fuel` 49.627. The needs are: cook 75, digger 16.477, forester 49.627,
refiner 19.773 (2 ticks per `refine`), maker 30.426, and grower
`max(60 * 0.76705 + 1.5341, 0.76705 * 365 / 2) = max(47.56, 139.99) = 139.99`, since
`grow_staple` can start twice a year (59 and 119: 3.1), so growers need about three times
the labour their work alone suggests. The sum is 331.29, the factor 600 / 331.29 = 1.8111,
and the quotas are cook 135.83, digger 29.84, forester 89.88, grower 253.53, maker 55.11,
refiner 35.81. The floors sum to 596; the four largest remainders (forester .88, digger .84,
cook .83, refiner .81) take the remaining four, which gives cook 136, digger 30,
forester 90, grower 253, maker 55, refiner 36: the groups of `one.toml`, which is
therefore balanced (`spec-reference-chain.md` 4.4.1).

*The formula is pinned.* It reproduces the configured households of every group of every
region of `one`, `two`, `three`, `five` and `famine` (each is its own balanced census), and
no neighbouring reading does on `one`: the labour alone gives a grower 120 against 253, the
labour plus the capacity term 297, and the labour divided by `phi` with the capacity term
left whole 200. `second` is the one pack that is not balanced, by design (the tool sector
is five households per role: `spec-reference-chain.md` 4.4.7); its needs are 313.92
`algae_tender`, 194.34 `driller`, 0.164 `fabber`, 0.123 `smelter` and 0.011
`regolith_miner` households.

**Why it is a competitive equilibrium.** The goods and labour rows with a
fixed-proportion upkeep basket are the linear Fisher market with Leontief demand;
the programme's optimum and its dual are a competitive equilibrium, and conversely.
In the special case of one region, one primary factor (household labour), no joint
production and no capacity rent, the non-substitution theorem (Samuelson,
Georgescu-Roegen) gives prices independent of the composition of demand, so the
price vector does not depend on the census (section 14). Byproducts and
capacity rows break the theorem's conditions and the solver does not rely on it.

### 5.3 Existence and uniqueness

1. **Existence.** The programme is feasible (`x = 0`, `lambda = 0`) and bounded:
   every recipe has `d >= 1`, so the labour row bounds each `x`, and the goods rows
   then bound `lambda` when the basket is not empty. An optimum exists. If the basket
   is empty (no upkeep anywhere) the programme is unbounded and the solver returns
   `ErrNoUpkeep`; a configuration with no final demand has nothing to be viable for.
   The test is per realm, not per programme: the solver returns `ErrNoUpkeep` only when
   no group of any region has an upkeep. An autarky region whose own basket is empty
   while another region's is not is **not a programme**: it has no table row and its
   region is listed in `UnpricedWages`. A programme whose basket goods are all unsourced
   `Raw` markers (4.2.1) is treated the same way (no table row, its region in
   `UnpricedWages`, each such commodity in `UnpricedCommodities`), and `ErrNoUpkeep` is
   then returned only when no programme remains. A region with no households is not a
   programme. In `Validate` an empty basket is never an error (`no_upkeep` is a warning
   of `spec-config.md` 7.4.2): the programme is skipped silently (4.4).
2. **Uniqueness.** The optimal `lambda*` is unique. The primal `x` is not, when
   alternative optima exist (two recipes make the same good at the same value
   added). The dual `p` is unique when the primal optimum is non-degenerate. The
   solver reports `Degenerate` (a basic variable is 0), `AlternativeOptima` (a
   non-basic variable has a zero reduced cost) and, per commodity, whether the price
   came out 0.
3. **Choice among ties.** The tableau is built in canonical order (commodities by ID,
   activities by region then recipe ID) and pivots by Bland's rule, so the answer is a
   function of the configuration (FR-VIA-020), and equal on every platform.
4. **A zero price.** A commodity with dual price 0 is either in surplus (free) or
   unpriced because labour is slack in the only region that makes it (a region with
   unemployed labour has wage 0). The seed price of such a commodity is not guessed:
   it falls back to the effective reference price, and the report names it
   (`UnpricedCommodities`, section 8.2).
   **The capacity-rent case (found building the reference chain,
   `spec-reference-chain.md` 6.4).** A zero wage also arises in an ordinary viable pack:
   when a start-capacity row is tight (a crop that can start only in one season, so the
   households that can plant are the bottleneck) the labour row is slack, its dual is 0,
   the scarcity shows up as a rent on the capacity row, and every good that does not use
   the capacity prices at 0 and is raised to the floor. In the reference chain this
   happens when the staple yield of the one-region pack falls to 3 800 or below
   (`lambda*` 1.67). The table would be a poor start, with a tool priced at the floor and
   no starting wage. The rule: for a region whose labour dual at full labour is 0 and
   whose capacity rent is positive, the solver builds that region's seeds, wage and
   `lambda`-independent quantities from the **nominal** programme (the one with `phi` the
   expected work share, 3.2), where labour binds; the report lists the region under
   `NominalSeed`. If the nominal wage is also 0 the region stays unpriced as before
   (`UnpricedWages`). In the example the nominal table has a wage of 1 826 and a `tool_b` at
   12 432 (`tool_a` 10 114), against 0 and 0 at full labour, and at the unchanged yield of 4 000 the full
   labour table (wage 755) is used (FR-VIA-058).

   **The fallback is per programme.** In an autarky programme the programme is the
   region. In a pooled programme the prices are one vector, so the whole pooled table is
   rebuilt from the nominal pooled programme (`phi` the household-weighted mean over the
   regions), and every region whose full-labour wage was 0 with a tight capacity row is
   listed in `NominalSeed`; a region whose nominal wage is also 0 stays in
   `UnpricedWages`. **Which programme gives which part:** the seed prices and the wage
   come from the nominal programme, but the flows (jobs, reserve, stock inputs, census)
   come from the full-labour programme at lambda 1 (5.1), which is what the warm
   document holds: in `famine` the nominal programme gives lambda 1.11247 and wage 1 624,
   and the flows are those of lambda 96/73 = 1.3151 at full labour (`grow_staple` 2.5
   and no `save_seed`, because its capacity row is tight; `make_tool_a` 2 272 =
   2.5 * 3 / 33; reserve 393 = 383 + 10).
5. **What the equilibrium is not.** It is the frictionless equilibrium of a planner.
   Whether the simulation, with beliefs, discrete agents and delays, converges to it,
   stays near it or leaves it is a question of stability (section 13), not of
   existence.

### 5.4 Certification and failure

After the last pivot the solver checks, in exact arithmetic, that the primal is
feasible, the dual is feasible, and the two objectives are equal (strong duality).
It returns the table only if all three hold. Failures are sentinels:

| Sentinel | When |
|----------|------|
| `ErrNotViable` | the configuration fails V1 to V4 (the same error as `Validate`, FR-CFG-033) |
| `ErrNoUpkeep` | no group has an upkeep |
| `ErrSolveLimit` | the pivot or size cap was reached (`SolveError` carries the pivots, rows and columns) |
| `ErrSolveInconsistent` | the certificate failed; this cannot be caused by input and is a defect |
| the wrapped `ctx.Err()` | the context was cancelled |

No partial table is returned with any of them. Bland's rule terminates on every
input, so non-convergence is only a cap (architecture.md 8.1, 12.3).

### 5.5 Cost

The tableau has one row per commodity per pooled region set, one per region, and
one per tight capacity, and one column per activity plus lanes. With 5 regions, 30
commodities and about 45 recipes that is about 150 rows and 225 columns for the
pooled case. The prototype needed 306 pivots and 2.4 seconds with Python's
`Fraction` on this size (section 15); a Go version with `math/big` integers is
expected to be faster by a large factor, but that is not measured. The cost grows
with the number of activities times the number of constraints; a world at the
maximum bounds of `spec-config.md` 4.1 is far larger, which is why `Validate` runs
the programme only below the size gate (11).

## 6. Arithmetic and rounding

6.1. The solver computes in exact integers of unbounded size. It uses `math/big`
(standard library), a fraction-free tableau (a common denominator per row), and
no `float32`, `float64` or `math` function. Architecture.md 3.5 allows `math/big`
in `internal/steady` and only there: unbounded integers cannot overflow, so the
checked helpers are not needed *inside* the solver; the result leaves it as fixed
point at its boundary, once, and every conversion to `int64` is checked against
`int64` and the field's bound (6.4). No `math/big` type appears in anything
exported from the root package, `config/` or `strategy/`, in an event or in a
snapshot: `internal/steady` takes and returns primitive types (architecture.md 2.9).

6.2. **Net vector.** A wear flow `WearPerUse / Durability` and a factor `E / 10 000`
are rationals. The solver multiplies a row through by the least common multiple of
its denominators so every entry is an integer.

6.3. **Price scale.** The solver's prices are in basket units. The **seed price**
is the solver's price times a scale `s`, then rounded half to even
(`spec-clearing.md` 5.2) to an integer:

- autarky: `s = B_ref(rho) / sum over c of u(c) * p(c)`, where `B_ref(rho)` is the
  sum over the regional upkeep of quantity times effective reference price
  (`spec-currency.md` 5.1);
- pooled: one `s` such that the household-weighted sum of the baskets equals the
  household-weighted sum of the `B_ref`, so the realm has one numeraire.

The level is therefore the configured level, and the relative prices are the
solver's. The seed price of a commodity differs from its reference price by the
**drift**, which the report gives. A large drift means the configured reference
prices are inconsistent with the technology; the author should replace them with the
seeds (this is the calibration use).

6.4. **Bounds and floors.** A seed price is raised to the resolution floor
`ceil(10 000 / BeliefStep)` (`spec-currency.md` 3.1) and capped at `MaxPrice`. A price
raised to the floor, a price of 0, and a price outside the reference band
(`spec-beliefs.md` 3.2.1) are recorded in the report. **The table must pass `Validate`:**
a seed price that the resolution floor binds, or that is a free good (dual 0), is lifted
to the lower bound of the reference band that `Validate` enforces (8.2.2, FR-VIA-030), and
a price above the upper bound of the band is capped to it. Each lift is listed in
`Floored`, and the unclamped price is recorded in `OutsideReferenceBand`, so the record of
this section is kept while the table loads. The band rule of the validator is unchanged. A
quantity or money amount that does not fit `int64` or its bound is `ErrOverflow`.

`world.belief_step` moves the table only for a price that the floor binds, which is the one case in which it is not a check (11).

6.5. **Residual.** After rounding, the solver recomputes the value added of each
active activity at the integer prices and compares it with the wage times the
duration. The largest deviation in basis points is `Report.ResidualBP`; those above
`ResidualWarnBP` are listed. The cause is the floor on free goods (a byproduct at the
floor adds revenue the exact solution did not have) and the coarseness of the minor
unit (`spec-currency.md` 3.2). A large residual is a reason to choose a finer minor
unit, not an error.

## 7. Where the solver lives

7.1. **Core, as a pure function.** The root package exports:

```go
// SolveSteadyState solves the reference programme for cfg and returns the steady
// state and the warm-start table. It is pure: it reads cfg, draws no random
// number, writes no log and starts no goroutine. NewWorld never calls it.
func SolveSteadyState(ctx context.Context, cfg Config, opts ...SolveOption) (SteadyState, error)
```

The algorithm is in `internal/steady` over primitive slices (architecture.md 2.9).
`Validate` calls a bounded form of the same code for V1 to V4 (4.6).

**The result.** `SolveSteadyState` returns `SteadyState{Programmes []ProgrammeResult,
Table WarmStart, Report Report}`. A `ProgrammeResult` is one programme solved:
`Regions []string` (one region in autarky, all the regions of the component in the pooled
case), `SurplusBP` (the floor of `lambda* * 10 000`), `LabourShareBP` with
`HasLabourShare` (absent when a capacity row is tight: FR-VIA-012), `NominalSurplusBP`
and `Pivots`. The `Report` holds the diagnostics of 10.3, every list in a stable order
(region, then commodity or recipe ID) and no map. The exact flows are not exported: the
census and the table are the only flow-derived outputs (architecture.md 2.8). The
fields owned by a module (10.3: `Headroom`, `Rent`, `FacilityDecay`, `SurplusWantsBP`)
are appended with their module and are empty while it is off.

**Autarky is the default.** With no `WithFreeTrade`, each region with a group is solved
alone: its own programme, its own numeraire `B_ref(rho)` (6.3), and the rows of the
table for the region come from it. With `WithFreeTrade`, all the regions with a group
are one pooled programme with one scale: the free-trade set only, the complete set of
zero-cost lanes, collapsed to one goods row per commodity (the labour and capacity rows
stay per region). A region with no group has no programme and no table row. There is
no partial pooling in Stage 1 (9.4).

**The options** are functional, of the type `SolveOption` (not the world's `Option`),
and five: `WithStockCoverTicks`, `WithMoneyBufferTicks`, `WithResidualWarnBP`,
`WithMaxPivots` and `WithFreeTrade`, with the bounds of 11. A value outside its bound
returns `ErrOutOfBounds` from `SolveSteadyState` before any work; a nil option is
ignored; the last option of a kind wins.

7.2. **Why core.** (1) The warm start is part of "version 1 is done" and a consumer
must be able to produce it with the public API alone (architecture.md 1.3); a
consumer that ships a precomputed table works too, since the table is
configuration. (2) CFG needs V1 to V4 at load, and they share the programme and the
exact arithmetic. (3) The function reads `Config` and returns plain data: it is the
kind of function a pure library has. (4) It does not belong in the tick path, so its
allocation and its time are outside NFR-SCL-002 and observability-and-logging 8.2.

7.3. **Why not behind the experiment harness.** The harness (EXP) sweeps parameters
and compares runs; it needs the solver as a building block and not the other way
round. If the solver lived there, `Validate` could not reject a non-viable chain
(FR-CFG-033), and a consumer would need a second module to start a run.

7.4. **Why not inside `NewWorld`.** A table that `NewWorld` computed would depend on
solve options that are not in `Config`, so the configuration hash would not say what
the run was. The table is data in `Config.WarmStart`; `NewWorld` validates it and
applies it. A consumer who wants "solve, then run" calls both.

7.5. The solver adds nothing to the four extension interfaces (architecture.md 4.3).
Its use of `math/big` is the exception architecture.md 3.5 allows for
`internal/steady` (6.1).

## 8. The warm start

### 8.1 The section

`Config.WarmStart` is a pointer: nil means no warm start (the baseline), non-nil
means the world starts from the table. It is part of the effective configuration and
the hash (`spec-config.md` 8). It is a section like `Calendar`, not a module: it
switches no mechanism on.

```go
// WarmStart is the steady-state seed table. Plain data, sorted, no maps.
type WarmStart struct {
	Prices []SeedPrice  // sorted by region, then commodity
	Wages  []SeedWage   // sorted by region
	Groups []WarmGroup  // sorted by region, then role
	Pools  []WarmPool   // sorted by region
}

// SeedWage is the starting wage of a region: money per working household-tick,
// used as the pooled observed wage whenever that figure is unknown (8.2.5).
type SeedWage struct {
	Region string
	Wage   Money // minor units per working household-tick
}

type SeedPrice struct {
	Region, Commodity string
	Price             Price // minor units per unit
}

type WarmGroup struct {
	Region, Role string
	Money        Money     // per household, minor units
	Stock        []Item    // per household, sorted by commodity
	Jobs         []JobSpec // sorted by recipe
}

// JobSpec says what share of the group's agents start in progress on a recipe.
type JobSpec struct {
	Recipe       string
	InProgressBP int64 // basis points of the group's agents, 0 to 10 000; the file key is `in_progress` (`spec-config.md` 6.2.2)
}

type WarmPool struct {
	Region string
	Money  Money // regional estate pool at the start; default 0
}
```

In a file the four lists are written once per region (`spec-config.md` 6.2.2, which lists the
mapping): `[warm_start.region.<id>]` with `wage` (`SeedWage.Wage`) and `pool`
(`WarmPool.Money`), `[warm_start.region.<id>.price]` with one key per commodity, and
`[warm_start.region.<id>.group.<role>]` with `money`, `stock` and `jobs`, a job as
`{ recipe, in_progress }` (`in_progress` is `JobSpec.InProgressBP`). A price missing for a
commodity falls back to the reference price; a group missing keeps the role's
defaults; a region with no wage has an unknown wage whenever the pooled figure is unknown (8.2.5), and nothing is latched.

**What the table does not carry.** The table has no claim (`spec-credit.md` 11.4), no hire in flight (`spec-employment.md` 22 item 7, 4.7), no rent column (`spec-land.md` 12.3: the money seeds are not computed from a rent circuit) and no policy state: a world made from it starts with none of them, and a pack that wants an initial claim or a hire states it itself. The modules that read nothing of the table are listed in 11 ("Not read").

**Order of application.** The table states no order of its sections; the world applies them in the order of `spec-world.md` 5.6.1: the prices, the wages, then the groups (money and stock), then the jobs (after the stock, because the oldest-first moves of the job's construction step would otherwise draw on the staggered lots), then the pools. The warm start takes no random number (FR-VIA-040).

### 8.2 Prices

8.2.1. A `SeedPrice` replaces the effective reference price as the market's initial
published price (`spec-clearing.md` 5.4, price age 0) and as the starting centre of
each initial agent's belief, clamped to its bounds (`spec-beliefs.md` 8.1, 8.2,
FR-BEL-022, FR-BEL-024). It is not the reference price: the reference band of BEL
3.2.1 stays **anchored** on the reference price (it is not centred on it), and the
cost band of BEL 3.2.2 on the unit cost.

8.2.2. Validation checks each seed price against `[1, MaxPrice]`, the resolution
floor, and the reference band of BEL 3.2.1 (`ErrWarmStartPrice`, FR-VIA-030).

8.2.3. **Seeds and the cost band.** A seed price must also lie inside the cost band
of BEL 3.2.2 computed at the seed prices: `UnitCost * CostFloor` to
`UnitCost * CostCeiling`. If it does not, BEL clamps the belief centre to the band at
creation, and the market starts with a published price that no agent holds. The
solver checks this for every (region, commodity) with a known unit cost and reports
`SeedBelowCostFloor` and `SeedAboveCostCeiling`.

**The unit cost of the check is the full cost, labour included** (`spec-production.md`
9.1, 9.3): inputs, tool wear and `Duration` working ticks at the region's wage. The
solver uses its own wage (the dual of the labour row, scaled and rounded like the
prices, 6.3), which is also the starting wage of 8.2.5, so the check and the run
agree. The cost uses the integer seed prices and rounds up as production does.

**The ceiling finding is resolved (N-055).** With the cost counting inputs and wear
only, the seed was 3.7 times the unit cost for flour, 9.7 for dough, 8.4 for a loaf
and 19 for a tool, against a `CostCeiling` of 3 times, so the ceiling would have
capped four of five goods below their equilibrium price. Production now counts
labour at the observed wage (`spec-production.md` 9), and the zero-profit price
equals the full unit cost by the identity of the programme (value added equals the
wage times the duration). Re-run for this revision (section 15): the seed over the
full unit cost is between 0.991 and 1.000 for every active activity of both example
chains, at the integer prices. Against the new defaults of BEL, `CostFloor` 7 000 and
`CostCeiling` 30 000 (`spec-beliefs.md` 3.2.2), a seed at about 10 000 sits inside
the band with headroom of 30% below and 200% above. The difficulty has moved to the
floor, which would bind exactly at 10 000; BEL's reduction to 7 000 is the answer
(STB-L09, STB-L10). The solver still reports any violation, which now means a
chain whose price is not the full cost: a byproduct priced at the floor, a good
whose only activity is inactive, or a rounding effect at the resolution floor.

8.2.4. A commodity whose price is not unique (5.3) gets its reference price in the
table, and one whose price is 0 (a free good, dual 0) gets **the larger of the
resolution floor and the lower bound of the reference band** (6.4) instead of the
reference price; both are listed in `Report.UnpricedCommodities`, not guessed, and the
`Floored` and `OutsideReferenceBand` records of 6.4 are written for a lifted free good as
for any other lift.

8.2.5. **The starting wage.** `SeedWage` gives, per region, the money per working
household-tick that the cost anchor uses whenever the pooled observed wage of the
region is unknown (`spec-production.md` 9.3, which asks for it; N-060). Without it the wage is
unknown in the first ticks and no commodity has a cost band (`spec-production.md`
9.1.1). The solver sets it to the wage of the dual (the shadow price of the labour
row), scaled by the same scale as the prices (6.3) and rounded half to even; where the
labour row is slack the wage is 0, and no entry is written for the region (the wage
stays unknown and the region is listed in `Report.UnpricedWages`). The world uses it
as the pooled observed wage of `spec-welfare-and-wages.md` 4.3 whenever that figure is
unknown: no agent of the region has a working household-tick in its window, or the
pooled figure is not positive. That is a pure function of the welfare rings and the
configuration, and **nothing is latched** (`spec-checkpoint.md` 4.5 F5): an
observation replaces it while it lasts, and it is the figure again if the rings
empty. Validation checks it against `[1, MaxMoney]` (a wage is a per-tick amount, and no
bound tighter than a balance is meaningful) and that the product of the wage and the
longest `Duration` of a recipe is at most `MaxMoney` (`ErrWarmStartPrice`; the product
is the labour cost of `spec-production.md` 9.1). It is a level, not a rule: it changes
the first ticks only. Where the full-labour programme gives a region a wage of 0 with a
tight capacity row, the nominal programme supplies the wage and the prices (5.3.4).

### 8.3 Money

8.3.1. The money of a household is set so that it can buy the inputs of its job and
pay its upkeep until the first income arrives, plus a buffer:

```
Money = inputs_cost + U * (d_max + MoneyBufferTicks)
```

`inputs_cost` is the cost at seed prices of the inputs of one execution of the
group's main recipe; `U` the household's upkeep cost at seed prices; `d_max` the
longest `Duration` among the group's active recipes. An **active recipe** is one with a
positive execution in the solver's final basis scaled to lambda 1 (5.1), so a long recipe
the basis leaves unused, such as `salvage` of `second`, counts for neither `d_max` nor the
main recipe. **The main recipe** of a group is its active recipe with the **longest
`Duration`**, ties broken by the **largest inputs cost** at seed prices, then by the
lower recipe ID. Worked case, `one` (`U` = 2 * 96 + 628 + 179 = 999, `MoneyBufferTicks`
10): the grower's main recipe is `grow_staple`, and its money is 78 810 = 999 * (60 + 10)
+ 8 880, the 8 880 being `grow_staple`'s 40 `seed` at 222, not `save_seed`'s 20 at 96; the
maker's is `make_tool_b`, 22 557 = 999 * 13 + 9 570, the 9 570 being its 3 `refined` at
3 190. In `one` the longest duration and the largest inputs cost pick the same recipe in
both groups; the rule is the one that reproduces every group of the five warm documents
(the largest inputs cost alone would give the `second` fabber 52 189 from
`make_fabricator`, duration 1, where the document has 19 423 from `fab_board`, duration
3). This is the rule that ties the
warm start to N-032: a household that holds less than `U * d` cannot live through a
job. The welfare ratio itself no longer misreads a long job, because a role's window
is never shorter than twice its longest job (`spec-welfare-and-wages.md` 2.3.3; the
warning `job_longer_than_window` is replaced by the informational `window_extended`,
2.3.5, 11.4), but the cash to live through it is still the household's. The money is
capped at `MaxMoney`. A money flow adds no real demand or supply, so the warm table of quantities, jobs and relative prices stands; a balanced pair is neutral on the money supply and an unbalanced faucet has no nominal steady state (`spec-money-supply.md` 15). A class that a rate levy bounds holds at most `E + g/rho` (`spec-money-supply.md` 9.3), and a warm `Money` above that decays toward it; seeding such a hoarder class at the smaller of the two is optional.

8.3.2. The sum over all households is the amount created by the ledger reason
`initial` (`spec-ledger.md` 7.2), and that is the only money created at construction
(`spec-estates.md` 2.3). The warm start creates no other money.

8.3.3. **The estate pool.** `WarmPool.Money` is 0 by default: pools start empty, as
`spec-estates.md` 2.3 implies, so a replacement in the first ticks may start with
nothing. A table may seed a pool, created by `initial` into the estate's account.
Whether a default pool is wanted is open question 6.

### 8.4 Stock

8.4.1. The stock per household is:

- the **cover** of the upkeep: `StockCoverTicks` ticks of each upkeep commodity;
- for the group's active recipes, the inputs of one execution, so that an idle
  agent can start at once (an **active** recipe is one with a positive execution in
  the solver's final basis scaled to lambda 1, 5.1);
- each required tool, one item per execution a household can run;
- for a commodity with a **supply gap** (4.5) that the group's recipes produce, the
  **lean reserve**: the largest cumulative shortfall of consumption against
  production over the year starting at tick 0, computed from the start windows,
  durations and the upkeep rate. It is held by the groups that produce the
  commodity, in proportion to their output.

**The composition of the stock per household** (pinned against `warm-one.toml`,
`warm-second.toml` and `warm-famine.toml`; `one` is the worked example):

1. The reserve counts the **whole consumption** of the commodity in the region at the
   flows of lambda 1 (5.1): the upkeep `D(c, rho)` plus the inputs and the tool wear of
   every active recipe of the region. Upkeep alone gives 560 `staple` where the document
   has 644.
2. A commodity's stock per household is the larger of its cover and the sum of the
   one-execution inputs of the group's active recipes, **plus** the reserve and the tool
   items: a cook holds `staple` 10 = max(cover 10, `prepare` input 2); a grower holds
   `staple` 664 = 644 + max(10, `save_seed` input 20); a maker holds `refined` 5 = 2 + 3,
   the inputs summed over the group's recipes, not their maximum.
3. A recipe of duration 1 writes no job (8.5.2).
4. `InProgressBP` rounds down (8.5.1): 28.125 / 55 = 5 113.6 gives the file's 5 113.
5. No pool row is written (the documents have none).
6. A commodity that appears in no input, tool, non-discard output or upkeep of the
   programme (a discard output only, such as `waste`) has no row in the table and no
   `UnpricedCommodities` entry.

**The lean reserve, exactly.** Let `cons(c, rho)` be the whole consumption of `c` per
tick (item 1) and, for each open start tick `t` of an activity that outputs `c`,
`x1 * TicksPerYear / N_open` executions start (`x1` its flow at lambda 1, `N_open` the
open ticks of the year: the rate of 8.5.1), each yielding its non-discard output at the
factor of the start tick, credited at its completion tick (4.5). A start whose completion
tick is at or after `TicksPerYear` adds nothing: it is not wrapped to the next year, and
`N_open` still counts its tick. The asymmetry with the cyclic gap of 4.5 is deliberate:
the reserve bridges the first year from tick 0, when nothing begun before tick 0 can
complete (the jobs in progress at tick 0 are table rows, 8.5), so a start late in the
year has produced nothing by the year's end, while the gap describes the periodic steady
state. With `prod(t)` the output completing at `t`, the reserve of the realm for `c` is

```
max over t in [0, TicksPerYear) of  max(0, sum over u <= t of (cons(u) - prod(u)))
```

It is split among the groups that produce `c` in proportion to their output (largest
remainder in whole units of the total, ties by role ID), and then **per household
`ceil(the group's share / the group's households)`** (`warm-famine.toml` holds 383 + 10
where 1 350 * 118 / 417 = 382.01: the household's figure rounds up, not the total), and
cut to the shelf life (8.4.2). It exists only for a commodity with a supply gap in the
sense of 4.5, which is every commodity the region's activities output, not only upkeep and
all-year inputs (`storage_gap` and `storage_gap_input` are the findings that restrict
themselves to upkeep and inputs); so an all-year activity (`refined`, `tool_a`, `tool_b`
of `one`, `biomass` of `second`) has none. It reduces to the realm's whole use per tick
times the ticks to the first harvest, shared among the producing households, when
production is zero up to the first completion.

*Worked case, `staple` in `one`.* The first completion is at tick 118. The realm uses
1 200 `staple` a tick as upkeep, 150 as the input of `prepare` (75 executions a tick) and
30.68 as the input of `save_seed`: 1 380.68 a tick. Over 118 ticks, shared among the 253
growers, that is 1 380.68 * 118 / 253 = 643.95, which rounds up to **644** per grower; the
664 of `warm-one.toml` is these 644 plus the 20 `staple` a grower holds as the input of
one `save_seed` (item 2). In `famine` the figure is 393 = 383 + 10 (1 350 a tick, since
`save_seed` is idle at lambda 1: 1 350 * 118 / 417 = 382.01). In `two` the figures are
723 (`north`: 162 920.45 / 232 = 702.2, 703 + 20) and 789 (`south`, which plants twice, so
the first harvest is also at tick 118: 109 959.29 / 143 = 768.9, 769 + 20).

*Where the cumulative maximum is not at the first harvest.* In `centre` of `five` (spring
and autumn plantings, both at 8 000) the use is 2 314.29 `staple` a tick and the first
harvest is at tick 118, so the shortfall is 273 085.71 at tick 117. The spring harvest
(completions 118 to 209) then brings it down by 211 750.82, and the 92 ticks from 210 to
the first autumn completion (tick 302) use 212 914.29, so the shortfall is back up at
274 249.18 at tick 301, a little above the first peak. 274 249.18 / 256 growers = 1 071.3,
so a grower holds 1 072 + 20 = 1 092. The first-harvest wording gives 1 067 + 20 = 1 087,
which is what `warm-five.toml` holds. The two readings agree in every other region of
`two`, `three` and `five`, and 8.4.1 as written (the cumulative maximum) decides.

8.4.2. Cover and reserve that would last longer than a commodity's `ShelfLife`
are cut to the shelf life, and the report names the commodity.

8.4.3. **The stagger rule.** Wherever a group has a period `P` (a job's duration, a
tool's durability, a lot's shelf life) and `n` agents, agent `i` (0-based, in agent
order) is given phase `floor(i * P / n)`. The rule uses no random number, so two
runs are equal and the stream is untouched (FR-VIA-040):

- a durable in the stock has remaining durability `Durability - floor(i * Durability / n)`
  (from the full value down to 1), so the tools of a group do not wear out in the
  same tick (`spec-skill-barriers.md` 2.8; `spec-goods.md` 10);
- a lot of a perishable has `Born = -floor(i * ShelfLife / n)`, so lots do not spoil
  together (`spec-goods.md` 4.5);
- a job (8.5) has `Remaining` set by it.

8.4.4. The stock is created by the ledger reason `initial`. Total goods at
construction are the sum of these amounts and those of 8.5.

8.4.5. **Endowments (optional).** With the resources module on, the stock of an endowment is not in this table. A warm-started pack sets `InitialBP` of the resource to its `PlanShareBP`, or a later version of the table adds one row per endowment (`spec-resources.md` 11.4, open question 3 there). A cold start from a full stock is legal.

### 8.5 Jobs: the answer to the production open question 2

8.5.1. `JobSpec.InProgressBP` is the share of the group's agents that start with a
job in progress on the recipe. The solver sets it to the expected busy share at an
arbitrary tick: `x * TicksPerYear * |V| / N_open` executions in progress per tick
(for a recipe that runs all year, `|V| = d` and `N_open = TicksPerYear`, which gives
`x * d`), divided by the households of the group, clamped to 10 000 basis points,
rounded **down**. Here `x` is the flow at `lambda = 1` (5.1). `V` and `N_open` are
defined next.

**A recipe on the lists of several roles of a region** (3.3 item 2: the activities are
open to any household). The executions in progress of the recipe are divided by the sum
of the households of **all** the groups of the region that list it, and each such group
gets that same share (each household is equally likely to be the one running it). If the
shares of one group then sum above 10 000 they are scaled down in proportion, each
rounded down, and the group is listed in `Report.Census` as short. A recipe active in a
region where no group lists it gets no job and is listed in `Report.Census` too.

**A seasonal job keeps the formula as written.** For a recipe that does not run all
year the executions in progress are `x * TicksPerYear * |V| / N_open`, not `x * d`: the
formula spreads the executions over the open start ticks at the rate the steady state
has (the rate of 8.4.1), where `x * d` assumes a start at every tick. A warm document
that differs from the formula is the stale one (FR-REF-007: the documents are the
solver's output). Where the two differ (a recipe that does not run all year) the file is regenerated:
`two` south has 3 341 in the document, from `floor(10 000 * x * 60 / growers)`, against
3 110.5 from the formula; six job rows of `warm-two`, `warm-three` and `warm-five` differ
this way, and none in `one`, `second` or `famine`, where the recipe runs all year and the
two formulas coincide. Likewise the `grow_algae` row of `second` (5 163 in the document)
is the solver's to settle: the rule of 3.1 gives a share of 5 232.

8.5.2. **Spreading.** Let `V` be the **virtual start offsets** of the recipe: the
offsets `o` in `0 .. d-1` such that the yield factor at the virtual tick `-o` is
positive, where the season of tick `-o` is `Calendar.SeasonAt(-o)`
(`spec-calendar.md` 4.5.1, FR-CAL-019, FR-CAL-020): the periodic extension is the
calendar's and not an assumption made here, it accepts any `int64`, and it is
`-1` when there are no seasons. `N_open` is the number of ticks in a year at which the
factor is positive. For an agent given a job (the first `busy` agents of the group
in agent order, `busy` by the joint apportionment below), the `j`th
takes the offset `V[floor(j * |V| / busy)]`, sorted ascending. A recipe of duration 1
writes **no job**: `V` would be `{0}` and the share positive, which contradicts the
document ("the one-tick recipes have no job in progress at tick 0",
`spec-reference-chain.md` 4.4.1). Then:

- the job started `o` ticks before tick 0: `Remaining = d - o`;
- the effective factor, and so the pending outputs, are those of the season of the
  virtual start (`spec-production.md` 6.3), with the shock at its default and a
  starting residue of 0 (the creation draw of `spec-world.md` 5.7 comes after the warm
  start and sets every residue of the agent, C4; the output of the job in progress was
  fixed before it and is not recomputed);
- `Executions` equals the agent's weight, and `Principal` is the agent itself.

**The agents in progress.** For a group of `n` agents the apportionment is **one joint
largest-remainder apportionment** over the `n` agents: the quotas are
`share * n / 10 000` for each `JobSpec` of the group in canonical order, with the idle
share (`10 000` minus the shares) last; the floors are taken first, the remaining agents
go to the largest remainders, and a tie goes to the earlier entry (`architecture.md`
7.6). The busy agents are then assigned in agent order, **spec by spec**: the first
agents to the first `JobSpec`, and so on, the idle agents last. Each `JobSpec` takes the
offsets of its own `V` as above, with its own `busy`.

If `V` is empty the table is rejected (`ErrWarmStartJob`, FR-VIA-033; the rule runs at
the entry of phase 6 of `Validate`: 8.7). A recipe
that can start in a single season therefore has all its in-progress agents inside
that season's window, which is what a real calendar would give at that tick.

8.5.3. **The flow rule.** A job in progress at tick 0 was started before the run, so
its inputs were consumed before the run. The world makes the books say so:
construction grants the agent the job's inputs by `initial` and takes them at once
by `recipe_input` in one construction step, wears the tools by `WearPerUse`, and
leaves the pending output in the job. The input `Transform` consumes **oldest first**, so
the stagger vector (8.4.3) is a property of the granted stock before the job's inputs
are taken: the post-warm stock is the stagger lots plus the granted input lots (fresh,
`Born` 0) minus the oldest-first consumption of the input quantity, so its total equals
the stagger total (granted equals consumed). The goods created at completion by
`recipe_output` are then paired with the inputs, and goods are conserved from the
first tick (FR-VIA-036; the audit passes at tick 0, FR-LED-046). No new ledger reason
is needed. The flows of construction are in the **construction bucket**, not in the
flow totals of tick 0 (`spec-ledger.md` 7.4.1, FR-LED-045); the ledger's construction
phase and its operations for this are `spec-ledger.md` 5.2.4 (the job's inputs by
`initial`, taken at once by `recipe_input` with `Transform`, the tools worn with
`WearItems` 5.4.3). This answers the open question 2 of `spec-production.md` 18, and
its "recipe output without a sink" worry.

8.5.4. **What the stagger buys.** Without it, every agent of a role starts at once
and completes at once: a seasonal crop all planted in spring all completes together
(a realistic glut), and a year-round recipe all completes together (an artefact).
With the rule, completions of a year-round recipe are spread evenly over `d` ticks
(N-036, `spec-production.md` 16, STB). The test is a spread bound (section 13).

### 8.6 What the ledger and the world must do

Items 1 to 3 were requested of the ledger (N-056) and are **applied in
`spec-ledger.md`**; items 4 and 5 are the world's.

1. `initial` creates goods at construction for any non-boundary account, estate
   accounts included. *Applied in `spec-ledger.md` 5.2.4 and 7.2 (the goods form of
   `initial`).*
2. A lot may be created with a negative `Born` and an item with a remaining
   durability below `Durability` at construction (8.4.3). *Applied in 5.2.4 and
   FR-LED-043:* `Born` from `-(ShelfLife - 1)` to 0 and a remaining durability from 1
   to full, anything else refused with `ErrInvalidLot`. The stagger rule of 8.4.3
   stays inside these ranges: `-floor(i * ShelfLife / n)` is at least
   `-(ShelfLife - 1)` for `i < n`, and `Durability - floor(i * Durability / n)` is
   from 1 to `Durability`.
3. Construction flows are totalled in a bucket before tick 0, and the audit includes
   it. *Applied in `spec-ledger.md` 7.4.1 and 9.1 (FR-LED-045, FR-LED-046).*
4. A job can be created at construction with `Remaining` below its `Duration` and
   pending output fixed from the virtual start (8.5.2). The ledger side is the
   construction phase of 5.2.4; the job itself is the world's and production's.
5. The agent's creation draws (the target ratio, then the residue of production) are
   unchanged and come after the warm start is applied (`spec-world.md` 5.6 and 5.7 fix
   the order, N-009).

### 8.7 Validation of the table

`NewWorld` rejects a table with: an unknown region, commodity, role or recipe
(`ErrUnknownRegion`, ...); a price or a starting wage outside its bounds, or a wage
whose product with the longest recipe duration exceeds `MaxMoney`
(`ErrWarmStartPrice`); a second wage for one region (`ErrDuplicateID`); a
`Money` above `MaxMoney`, a stock quantity above `MaxQuantity` or a quantity times
the largest weight above `MaxQuantity` (`ErrOverflow`); a recipe not on the role's
list, shares summing above 10 000 or a share outside [0, 10 000], or an empty set
of virtual start offsets (`ErrWarmStartJob`); two entries for one key
(`ErrDuplicateID`). A warm start does not need the whole table: absent entries take
the baseline.

**Where the empty-offset rule runs.** The rule "the set of virtual start offsets of a job
is empty" (`ErrWarmStartJob`, FR-VIA-033) needs the recipe, yield and role tables, which
exist only for a canonical, validated configuration. It therefore runs at the entry of
phase 6 of `Validate` (`spec-config.md` 7.1.6), on the canonical copy of a configuration
that phases 1 to 5 found clean. The sub-rule "the shares of a group exceed 10 000" is
already built, needs no table: it runs with the structural rules, before phase 6.

With the storage module on, a group with a job in progress is also tested against the room at construction: its stock load plus the load of the job's pending output must fit the base room plus the given stores (`ErrRoomShape`, rule `room_below_warm_job`, `spec-storage.md` 13.1). It is a check: the room does not enter the programme and the table is not changed by it (11).

### 8.8 Beliefs

BEL 8.2 starts every initial agent of a group at the same centre, so the warm start
supplies prices and nothing else for beliefs. It does not set a width. Newcomers
(replacements, newborns, migrants) start from the price window or the market price
(BEL 8.3), not the table: the table is a start, not a standing anchor (N-038, SPN,
MIG, CLR parts unchanged).

## 9. Regions and trade

9.1. **Climate as yield.** A region's climate is its yield profile
(`spec-production.md` 6; N-030). The solver reads it through `J` and `E` (3.1).
Differences in yield give comparative advantage with no further input.

9.2. **Comparative advantage.** In the pooled programme, a region's labour goes to
the activities where its yield relative to the others' is highest, as in the
Ricardian model: each region specialises where its opportunity cost is lowest, and
the pooled prices lie between the regions' autarky prices (Ricardo). The solver
reports, per activity, the regions in which it is active, and for each pair of
commodities the pooled relative price against the range of the autarky relative
prices (`Report.Advantage`). A pooled ratio outside the autarky range is reported as
`ratio_outside_autarky`. It is a test oracle and not a theorem: capacity rows and
degenerate duals can produce a ratio at an endpoint or, rarely, outside it.

9.3. **The three-region check.** `spec-reference-chain.md` needs a scenario with at
least three regions whose yields differ for at least three recipes, so that no
region is best at all of them (section 13, A3). The prototype of section 15 has one:
three regions whose grain, iron and wood yields differ. In autarky the three regions
have surplus factors 2.980, 2.517 and 3.124; pooled, 3.124, with the pooled relative
price of grain to fuel (1.8) equal to the autarky maximum (1.8) of the third region
and above the first (1.5) and the second (0.949). The gain from trade is small
there because one region's farm capacity is the binding constraint; a scenario for
REF should make the yield ratios large enough to see specialisation clearly.

9.4. **Lanes (the hook).** Transport is not specified. The solver takes a list of
lanes as data:

```go
// Lane is a directed route for one commodity, as a pair of activities.
type Lane struct {
	From, To  string // regions
	Commodity string
	Labour    int64  // household-ticks of carrier labour per 10 000 units, charged in From
	LossBP    int64  // share lost in transit, basis points
}
```

Each lane becomes an activity with net vector (`-1` at `From`, `+ (10 000 - LossBP) / 10 000`
at `To`) and labour `Labour / 10 000` in `From`. Nothing else in the programme
changes. Lanes are supplied by `transport.Lanes` (`spec-transport.md` 9.4, a pure function of the configuration that a consumer passes as an option): one per hired mode, ordered pair of regions with a route and commodity, with the expected loss of `spec-risk.md` 8.3 as `LossBP` (0 with the risk module off) and labour only (the carriers' feed and vehicle wear are not in the programme). Self-carried modes give no lanes in version 1. The default is no
lanes: autarky. The option `WithFreeTrade()` adds a lane of labour 0 and loss 0
between every pair of regions for every commodity. It is a bound for comparison and
the case `Validate` checks (4.6); it is not a model of any world.

**Stage 1 realises only the free-trade lane set.** `Lane`, `Lanes` (the option that
carries a lane list) and `MaxLanes` are not published in Stage 1: they come with the
transport module, which defines `transport.Lanes` and adds the three together. The
free-trade set is the complete set of zero-cost lanes, and the programme realises it as
a **collapsed single goods row** per commodity (the labour and capacity rows stay per
region): that row is the complete free-trade set of FR-VIA-043 and FR-VIA-044, and the
lane columns, which a partial set needs, are TRN's. With no `WithFreeTrade` each region
is solved alone (autarky, 7.1). When transport adds `WithLanes`, its lane validation is
an error and not a warning (a configuration that cannot be honoured is rejected with a
sentinel).

With the infrastructure module on, a route facility at full condition raises the transit factor of the lanes of its edge by its `BoostBP` and a ground facility raises the capacity of its endowment (`spec-infrastructure.md` 13.4; `spec-resources.md` 11): a gain at full condition, a constant, so the programme stays linear. A facility with `Decay` above 0 and no repair has no steady state with it (4.2.2).

9.5. **Import-dependent regions.** A region producible in the realm and not alone
gets the warning `region_not_self_sufficient` with the list of commodities it
imports (the first consumer's regions where a crop does not grow). Under autarky the
region has `lambda* < 1`; under lanes it may not. The simulation moves goods
between regions only when the transport module is on and an agent ships (`spec-transport.md`; the merchants are MER's), so such a region may starve in a run. The
warning is the author's cue. With edges (`spec-regions.md` 4), free trade is meaningful only within a strongly connected component of the graph (`Graph.Components`, `spec-regions.md` 4.3.5, 4.5.3): a pair of regions in different components has no route, so no lane. Whether the check should use the components is VIA's decision (`spec-regions.md` open question 2). A region may also be short because of a resource: the warning carries a resource cause when the binding row of its autarky programme is an endowment (`spec-resources.md` 11.3). The outside (`spec-external-sector.md`) is not a lane: its imports are columns of the open layer (4.8).

9.6. **Prices across regions.** With lanes, a seed price differs between regions by
no more than the lane cost in value; without lanes there is no relation, and each
region has its own level (anchored by its own `B_ref`). Law-of-one-price tests are
STY's.

## 10. Rules added to `spec-config.md` 7

Each has a sentinel and a failing-case test (testing.md 3.9). A typed
`*ViabilityError` carries the `Rule`, a `Path`, the offending value and, where it
exists, the number that decides it. `errors.Is(err, ErrNotViable)` matches every
error rule.

### 10.1 Errors (`spec-config.md` 7.3 / 7.4)

| Rule | Sentinel | Rejects |
|------|----------|---------|
| `unproducible_in_realm` | `ErrNotViable` | V1: an upkeep good, an input or a tool that no region can produce with a positive yield |
| `role_cannot_work` | `ErrNotViable` | V1: a role that lists recipes, with households in a region where none of its recipes can run |
| `cycle_not_productive` | `ErrNotViable` | V2: a strongly connected component of the material graph with no surplus |
| `labour_short` | `ErrNotViable` | V3: pooled `lambda* < 1` at full labour |
| `storage_gap` | `ErrNotViable` | V4: an upkeep good out of production longer than its shelf life |
| `resource_short` | `ErrNotViable` | V3: pooled `lambda* < 1` at full labour where a tight row is a resource row (`spec-resources.md` 11.3; with the resources module on) |
| (existing) | `ErrWarmStartPrice`, `ErrWarmStartJob`, `ErrUnknown*`, `ErrOverflow`, `ErrDuplicateID` | the table of 8.7 |

**The error type and its fields.** `ViabilityError` is its own type (it does not embed
`ValidationError`, `architecture.md` 4.7) with `Rule string` (the rule name above),
`Path []string`, `Value string` (the offending value as text) and `Bound string` (the
number that decides it, as text); `Unwrap` returns `ErrNotViable`. It joins the one sorted
list of `Validate`: path, then rule name, then value and bound, cut at 100 with one
`ErrTooManyErrors` (`spec-config.md` 7.1). The rule names are plain strings. The path,
value, bound and count of each rule:

| Rule | Count | Path | Value | Bound |
|------|-------|------|-------|-------|
| `unproducible_in_realm` | one per unproducible commodity | `commodities/<id>` | the commodity ID | empty |
| `role_cannot_work` | one per role | `roles/<id>` | the first region where none of its recipes exists | empty |
| `cycle_not_productive` | one per component | `commodities/<id>` of the lowest-index commodity of the component | `s*` in basis points (floor) | `0` |
| `labour_short` | exactly one for the realm | `regions` | `SurplusBP` of the pooled programme | `10000` |
| `storage_gap` | one per (region, commodity) | `regions/<region>/commodities/<id>` | the gap in ticks | the shelf life in ticks |

### 10.2 Warnings (`spec-config.md` 7.2)

| Rule | Condition |
|------|-----------|
| `region_not_self_sufficient` | V3 per region in autarky: `lambda* < 1`, or an upkeep good not producible in the region (9.5) |
| `subsistence_exceeds_expected_work` | the labour share to subsist is above the mean expected work share (4.4) |
| `target_above_surplus` | the mean `TargetRatio` is above the nominal `SurplusBP` (4.4) |
| `storage_gap_input` | V4 for an input of an activity that exists all year |
| `viability_check_skipped` | the size gate (11) was exceeded; says which part was not run |
| `raw_without_source` | a commodity marked `Raw` that no existing activity with no inputs outputs and that no endowment, faucet or import supplies (4.2.1) |
| `endowment_season_peak` | the draws of a start window exceed what the stock holds at the opening of the window (`spec-resources.md` 11.3); an approximation, never an error |
| `rent_off_dual` | with the tenure module on, a configured rent of an endowment whose row binds (`Rent` above 0) more than `RentDualTolBP` from that dual, or above 0 where the row is slack (`spec-land.md` 5.6.2); LND owns the rule and the name, the solver supplies the dual; never an error |
| `raw_source_exhaustible` | a raw commodity whose only sources are endowments of finite stocks (`spec-resources.md` 9.3); also counted by `raw_without_source`'s union of sources (4.2.1) |
| `import_only_source` | a commodity sourced only by an import line (`spec-external-sector.md` 10.1.2) |
| `wants_exceed_labour` | the demand module is on and the programme with the wants of 3.2 has `lambda* < 1` although the upkeep-only programme has `lambda* >= 1`: the wants are more than the labour can fund at full work. Size-gated like V3; never an error |
| `own_use_beats_market` | an own-use recipe whose coverage of a covered good reaches the surplus factor of a region where it has a positive factor (`spec-subsistence.md` 4.2, 13.3): `FloorBP >= SurplusBP`, against the autarky figure of the region (10.3) and, for a region that trades, the pooled one, the smaller used. SUB owns the rule and the name; the solver supplies the figure. Size-gated like V3 (`viability_check_skipped`); never an error |
| (not a warning) | `job_longer_than_window` of `spec-production.md` 10.4 no longer occurs: a role's welfare window is never shorter than twice its longest job (`spec-welfare-and-wages.md` 2.3.3), and the informational `window_extended` (WEL 2.3.5, 11.4) replaces it. The longest job against the window is not checked here |

### 10.3 Solver report diagnostics (not validation)

`Report` lists, with a stable order: `Degenerate`, `AlternativeOptima`,
`UnpricedCommodities`, `Floored` (prices raised to the floor), `OutsideReferenceBand`,
`SeedBelowCostFloor`, `SeedAboveCostCeiling` (both against the full unit cost, 8.2.3),
`UnpricedWages` (regions with no starting wage, 8.2.5), `Drift`, `ResidualBP` and the list over
`ResidualWarnBP`, `Census` (needed against configured households per group),
`CapacityTight` (the activities whose capacity row is tight), `FacilityDecay` (each facility class with `Decay` or `WearPerUse` above 0, informational: it is not in the programme, 4.2.2), `Headroom` and `Rent` (per endowment and region: the sustainable draw over the draw at the reference programme at `lambda` 1, and the dual of its row; `spec-resources.md` 11.3), `endowment_runway` (a finite stock's years at the optimum), `Advantage` (9.2),
`CapShortened` (stock cover cut to shelf life, 8.4.2), `NominalSeed` (the regions whose
table came from the nominal programme, 5.3.4), `SurplusWantsBP` (`lambda*` of the
programme with the wants, absent when the module is off and when that programme is infeasible, 3.2), `RegionSurplusBP` (per region, the autarky `SurplusBP`, and the pooled one for a region that trades: the figures that `spec-subsistence.md` 4.2 compares with `FloorBP`; absent when a part was not run), `WageBelowTarget` (the
role-economics check of `spec-welfare-and-wages.md` 7: value added per working tick
against `TargetRatio * B_r * TicksPerYear / ExpectedWorkTicks`). It is a **report and
never an error** (WEL 7.1, N-054): with the identity of 4.7 it fails for every role in a
closed economy with upkeep only, and the default `TargetRatio` of 15 000 stays
(WEL 7.2); the report is a calibration number, and `target_above_surplus` is the only
related warning.

`Headroom` and `Rent` are the guide of `spec-land.md` 5.6: with the stock slack (`Headroom` above 1) the commons serves everyone and a rent is never charged; the warning `rent_off_dual` (10.2) compares a configured rent with the dual. The programme is not edited for tenure (`spec-land.md` 12.1).

## 11. Parameters

The configuration adds one section and no tunable. The solver has options; they
shape the table that is then data, so they are not hashed and are not `Config`
fields, but each has a default, a bound, an owner and a doc line as if they were.
They are rows of the registry of knobs of `spec-calibration.md` (6.2.8), with the scope
"Solve option", and not rows of the parameter-owner table, because the registry test of
testing.md 6.7 requires a field behind every row (`spec-config.md` 9.2, FR-CFG-047).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Config | `WarmStart` | the seed table of 8.1; nil = off | nil | per 8.7 |
| Solve option | `StockCoverTicks` | ticks of upkeep held at the start (8.4) | 5 | 0 to 1 000 |
| Solve option | `MoneyBufferTicks` | ticks of upkeep of money held beyond the job (8.3) | 10 | 0 to 10 000 |
| Solve option | `ResidualWarnBP` | residual above which the report lists an activity (6.5) | 1 000 | 0 to 100 000 |
| Solve option | `MaxPivots` | pivots allowed before `ErrSolveLimit` (5.4) | `MaxViabilityPivots` | 1 to `MaxViabilityPivots` |
| Solve option | `Lanes` | routes between regions (9.4); **not in Stage 1**, with the transport module | none | at most `MaxRegions * MaxRegions * MaxCommodities` |

`HorizonYears` (years a finite stock must last) is not a solve option: it is a field of the resources module (`spec-resources.md` 12), read from the configuration, and has its row there. The inputs of the open layer (4.8) are the pure function `Lines()` of the external module and add no solve option.

**The option names (Stage 1).** The options are functions of the type `SolveOption`,
unexported state, and five: `WithStockCoverTicks` (`StockCoverTicks`), `WithMoneyBufferTicks`
(`MoneyBufferTicks`), `WithResidualWarnBP` (`ResidualWarnBP`), `WithMaxPivots`
(`MaxPivots`) and `WithFreeTrade` (9.4, no value). A value outside the bound of the table
returns `ErrOutOfBounds` from `SolveSteadyState` before any work. The `Lanes` row is
realised by `WithLanes` with the transport module (9.4).

Bounds, provisional, in `bounds.go` (`spec-config.md` 4.1):

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxViabilityPivots` | 100 000 | a cap on the work of a solve; with Bland's rule larger instances are possible but not at load |
| `MaxViabilityCells` | 4 000 000 | rows times columns of the tableau `Validate` will build; above it V2, V3 are skipped with a warning |
| `MaxLanes` | `MaxRegions * MaxRegions * MaxCommodities` | the largest lane list; added with the transport module, not in Stage 1 |

The placeholders, listed so that none is mistaken for a result: `StockCoverTicks` 5,
`MoneyBufferTicks` 10, `ResidualWarnBP` 1 000, `MaxViabilityPivots` 100 000,
`MaxViabilityCells` 4 000 000; and every price, quantity, duration and yield in
section 15, which are the prototype's inventions.

**The configuration paths the programme reads.** A table is stale when a path in
the first group of rows changes and the table is not regenerated (`spec-experiments.md`
3.3.3, EXP-A5; STB-L53). A path is written as the loader spells it
(`spec-config.md` 6.2): the table prefix of its scope and the snake case of the field.
`<id>` is an ID of the kind named. The list is of what 3 to 9 read; a field that no
row names is not read by the programme.

A `.*` at the end of a path names every field of that table, and of the tables nested in
it (`spec-calibration.md` 4.2.1 reads it so).

| Path | What the programme or a check does with it | Moves the table |
|------|--------------------------------------------|-----------------|
| `world.upkeep`, `role.<id>.upkeep`, `region.<id>.extra_upkeep` | the upkeep basket `u(g, c)` of each group (3.3 item 1) | yes |
| `region.<id>.group.<role>.households` | `H(rho)` and the demand `D(c, rho)` (3.1); `scale` is not read, since the weights do not change the households | yes |
| `recipe.<id>.duration`, `.inputs`, `.outputs` (with `cost_share` and `discard`), `.requires` (with `wear_per_use`), `.draws`, `.sited` | the net vector `n`, the labour `d` and the start capacity `J` (3.1) | yes |
| `commodity.<id>.durability`, `.shelf_life`, `.raw` | the tool wear flow (3.1), V4 and the lean reserve (4.5, 8.4), the sourcing of V1 (4.2) | yes |
| `region.<id>.yield.<recipe>` (`all`, `seasons`) | the mean factor `E` and `J` (3.1) | yes |
| `calendar.*` (`hours_per_day`, `days_per_week`, `months`, `ticks_per_day`, `days_per_tick`, `start_day`, `seasons`, `season_start`) | `TicksPerYear`, `SeasonAt` and the virtual starts of the jobs (3.1, 8.5.2) | yes |
| `role.<id>.expected_work_ticks` | the available work share `phi` of the nominal case (3.2) | yes (the nominal case only) |
| `role.<id>.recipes` | the checks that name a role and the assignment of jobs (3.3 item 2, 8.5); not the programme, which is free occupational choice | the jobs of the table |
| `modules.resources.resource.<id>.*`, `modules.resources.endowment.<region>.<resource>.*`, `modules.resources.horizon_years` | the resource row `Phi` and the abundance factor of `E` (3.1, 3.2, 4.8) | yes (module on) |
| `modules.demand.want.<id>.*`, `modules.demand.taste.<role>.<want>.*`, `modules.demand.region_taste.<region>.<want>.*` | the fixed term `W(c, P)` of the goods row (3.2) | yes (module on) |
| `modules.transport.mode.*`, `region.<id>.edge.<id>.distance` | the lanes a consumer passes (9.4); the pooled programme reads the region graph only through them | yes (when lanes are used) |
| `modules.infrastructure.*` | the facility row of each (role group, facility class) with `Needs`, the runnable test of V1 where an initial facility is held, and the gains of a route or ground facility at full condition (3.2, 4.2.2, 9.4); `Decay` and repair are not read | yes (module on) |
| `modules.external.node.*` | the import and export columns and the trade row of the open layer (4.8) | yes (module on) |
| `world.belief_step` | the resolution floor `ceil(10 000 / BeliefStep)` of a seed price (6.4); read by the programme only for a price at the floor, and by the checks of 8.2.2 to 8.2.4 | yes (a price at the floor only) |
| `commodity.<id>.reference_price`, `region.<id>.reference_price.<commodity>`, `world.cost_floor`, `world.cost_ceiling`, `world.ref_band_low`, `world.ref_band_high` | the fallback and the checks of a seed price (8.2.2 to 8.2.4); the prices are an output of the programme, not an input | no (a check) |
| `modules.storage.*` | the rule `room_below_warm_job`: a job of the table tested against the room at construction (8.7); the room is not in the programme | no (a check) |
| `modules.institutions.control.<id>.*` | the warning `cap_below_seed_cost`: a control's cap against the full unit cost at the table's prices (`spec-institutions.md` 10.2, 8.2.3); a price is an output of the programme, so no table moves | no (a check) |
| `modules.tenure.term.*` | the warning `rent_off_dual`: a configured rent against the dual `Rent` of its row (10.2, 10.3); titles and holdings are not read | no (a report) |
| `role.<id>.target_ratio`, `role.<id>.target_spread`, `world.welfare_window`, `world.window_cycles` | the reports `target_above_surplus` and `subsistence_exceeds_expected_work` (4.4, 10.2) and the longest job of the money rule (8.3.1) | no (a report) |

Not read: the spawner, the migration, information, merchant and subsistence modules,
the money flows (a balanced pair adds no real demand or supply, 8.3.1), the population
module, the settlements module (the table does not read a class, `spec-settlements.md` 6.4.4), the employment module (no hire in flight, 8.1), the credit module (no claim, 8.1), the risk module (a lane's loss reaches the programme through `Lanes`, 9.4, a derived input and not a path of the table), the insurance module, the polity module, the fiscal module (the programme counts none of its flows: `spec-fiscal.md` 21.2), the policy and saving modules, the technology module (a gain is 0 at tick 0, so no parameter of `modules.technology.*` moves the table or needs a regenerated one: `spec-technology.md` 3.3.2), the seed, the options and every presentation-only field. The consumer
(`spec-calibration.md` 4.2.1 flags a knob that moves the table with its own mark) adds
its flags to this list; none is decided here.

## 12. Placeholders and sources of the numbers

No default is set from a source. The prior art in section 14 supports the *method*.
Every quantity above is structural or invented. The prototype chains are labelled
setting-specific content in section 15.

## 13. What viability does not prove, and the scenarios handed to REF

### 13.1 What it does not prove

1. **Dynamic stability.** A viable steady state can be unstable: the target-income
   loop (N-028), the cobweb, a cost-push ratchet, herding. That is STB's register and
   STY's bands. Passing V1 to V4 is a necessary condition (`spec-config.md` 7.6).
2. **That the run reaches the point.** Beliefs, delays, the discrete agents and the
   spawner can leave the neighbourhood of the solver's prices.
3. **Demand beyond upkeep.** Section 4.7: the surplus `lambda* - 1` is capacity that
   nothing buys until DEM, MON or EXT exists.
4. **That the census is balanced.** The solver reports imbalance; it does not fix it.
5. **That the reference prices are right.** The drift tells you; it does not change
   them.
6. **Skill barriers.** Whether training pays and whether the premium is in its band
   are `spec-skill-barriers.md` 3 and 8.4 of PRO; the solver supplies the wages those
   checks need.
7. **Seasonal paths.** The programme is an annual mean; V4 and the lean reserve are
   the only seasonal checks. A seasonal storage model is open question 2.
8. **Anything about the first N ticks' transient** other than the stagger.

### 13.2 Acceptance scenarios for `spec-reference-chain.md`

REF assigned them to packs in `spec-reference-chain.md` 7.3: A1 `one`, A2 `defects/`, A3
`three`, A4 `second`, A5 `two` with `overlay/a5`, A6 no pack, A7 `one`, `five`, `second`,
A8 is S18, A9 every pack, A10 `five`.

| ID | Scenario | Asserts |
|----|----------|---------|
| A1 | One region, the reference chain | `lambda* >= 1`; every active activity at the wage within `ResidualWarnBP`; the table loads; a warm run keeps each price within `a1.price_band_bp` (STY-H01, prior 1 500 basis points, a placeholder) of its seed for the first 30 ticks (`T` is the welfare window, STY-H01; `spec-stylised-facts.md` 5, FR-STY-053) and a cold run does not do better |
| A2 | A set of non-viable variants of A1, one defect each | each is rejected by its own rule: a seed cycle with a yield below the seed in; an upkeep good no region makes; a role with no runnable recipe; a job so long that `lambda* < 1` (the as-written example A of PRO 12.1); a staple whose shelf life is below its supply gap |
| A3 | Three regions with different yields for at least three recipes | each region's autarky `lambda*` differs; the pooled solution has an activity active in a strict subset of regions; pooled relative prices lie in the autarky range or are reported |
| A4 | The space colony (a second setting) | the same checks pass with no Go specific to it (architecture.md 9.4). Run by REF as the configuration-only chain test X4 on the pack `second` (`spec-reference-chain.md` 7.3, 7.4) |
| A5 | A region that cannot make a staple | autarky fails, pooled passes; the warning names the import |
| A6 | A configuration with alternative optima and one with a slack region | the report flags `AlternativeOptima`, `Degenerate` and `UnpricedCommodities`; the table is still valid. **No pack:** a unit test of the solver on a hand-made programme; the reference chain cannot be slack (every unsited recipe can run anywhere, so a region's labour always has a use), `spec-reference-chain.md` 7.3 |
| A7 | Warm start conservation | the audit passes at tick 0 and every tick after; construction money equals the sum of `Money`; goods equal stocks plus the inputs of initial jobs |
| A8 | Stagger | over the first `d` ticks the completions per tick of a year-round recipe are within a stated spread of their mean (placeholder: no tick above twice the mean) |
| A9 | Determinism | two runs of the solver, on two architectures, give the same table bytes and hash |
| A10 | Cost | the solver on 5 regions, 30 commodities and 45 recipes finishes below a stated time (placeholder, measured in the slice) |

## 14. Prior art

"Read in full" means the page text was returned whole; the fetch tool returns a
summary of a page, so only those marked were checked against a stated claim.

| Source | Status | Taken | Not taken |
|--------|--------|-------|-----------|
| Hawkins-Simon condition, Wikipedia (`en.wikipedia.org/wiki/Hawkins-Simon_condition`) | opened through a summarising fetch; the statement quoted in 4.3 was returned. Its summary says the page does not discuss the spectral radius | The condition: all leading principal minors of `I - A` positive, equivalent to a non-negative `x` with `(I - A) x > 0` | The spectral-radius statement, which came from the Leontief page |
| Input-output model, Wikipedia (`en.wikipedia.org/wiki/Input-output_model`) | opened through a summarising fetch | `x = (I - A)^-1 y`; the productivity condition (largest eigenvalue of `A` below 1) | The closed-model-with-households text did not describe what is needed; the dynamic extension |
| Computable general equilibrium, Wikipedia | opened through a summarising fetch | Calibration to a benchmark equilibrium: choose parameters so the model reproduces the base data as an equilibrium (the warm start does the same for a chain) | Solution software (GAMS, GEMPACK) |
| Bland's rule, Wikipedia | opened through a summarising fetch | The rule never cycles; used for determinism and termination | Its slowness is accepted; sizes here are small |
| Comparative advantage, Wikipedia | opened through a summarising fetch | Specialisation on lowest opportunity cost; terms of trade between the autarky ratios; Ricardo's two-country example | Its trade-theory extensions |
| Non-substitution theorem (Samuelson 1951, Georgescu-Roegen 1951) | **search snippet only** (a search result text; the Wikipedia URL returned 404) | The conditions (one primary factor, no joint production) and the conclusion (price-to-wage ratios independent of demand composition) | Anything numerical |
| Factory solver mod and FactorioCalc (Factorio) | **search snippets only** | The idea that recipe chains with loops and byproducts are solved by linear programming or a custom simplex over machine counts | Interior-point methods; floating point |
| Sraffa (1960) *Production of Commodities by Means of Commodities*; von Neumann (1937); Leontief (1941) | **not opened**, named from general knowledge of the subject | Nothing is set from them. The idea of a subsistence economy with a surplus matches 4.1 | |
| Allen (2013) welfare ratio | read directly earlier (`evidence.md`) | The ratio in 4.7 and 8.3 is the WEL ratio | |

New evidence rows are in the handoff. No default in this spec is set from any source
(N-018).

## 15. The prototype

`proto3.py`, `round.py` and `ricardo.py` in `doc/specs/models/via/`, run with
`docker run --rm -v "$PWD/doc/specs/models/via":/w -w /w python:3.12-slim python <file>`. They
implement the programme of 3.2 with Python `Fraction`, Bland's rule and the
greedy start count of 3.1 on the Earth calendar (365 ticks, seasons from day 59),
so they check that the equations are well posed. They are not the Go design and
they use invented recipes. Setting-specific content, placeholder numbers.

**Chains.** Example A (`spec-production.md` 12.1) is completed with `save_seed`
(1 grain gives 1 seed), `cut_wood` (a source), `mine_iron` (a source) and the three
tool recipes (iron in, tool out; durability 2 000), upkeep 2 loaves and 1 fuel per
household per tick, 1 000 households. Example B (12.2) is completed with `melt`,
`dig_regolith`, `refine_wafer`, `refine_solder`, `make_drill`, `make_fabricator`;
upkeep 2 biomass and 1 water; algae in season `lit`, treated as spring and summer.

| Run | `lambda*` | Reading |
|-----|----------:|---------|
| A as written (6 grain per 120-tick job, 1 seed in) | 0.0137 (1/73) | needs 73 times the labour: **not viable** |
| A with the farm job rescaled (900 grain, 100 straw, 60 seed) | 0.296 | still not viable: the processing steps are too slow for 2 loaves a tick |
| A with outputs also rescaled (flour 12, dough 12, loaf 18 per execution, fuel 12, iron 18, grain 1 500) | 2.980 | viable; labour share to subsist 33.6%; 11 pivots |
| B as written (ice 5, biomass 5 per job) | 0.082 | not viable |
| B rescaled (ice 30, water 6, biomass 120, regolith 30) | 1.966 | viable; labour share 50.9% |
| B, two regions, free trade (crater rim 600 households, deep crater 400) | 1.911 | viable; deep crater alone cannot make biomass (`lambda* = 0`): import-dependent. Deep crater's labour is slack, its wage is 0, and every price but biomass comes out 0: the degenerate case of 5.3.4 |
| Three regions, autarky | 2.980, 2.517, 3.124 | each viable alone |
| Three regions, free trade | 3.124 (24 pivots) | the best region sets the bound; north and south bake, the third grows and mills |

Three regions: grain to fuel price ratio is 1.5, 0.949 and 1.8 in autarky and 1.8
pooled; iron to grain 0.444, 0.843, 0.296 and 0.296 pooled. The pooled ratios lie in
the autarky range at its endpoints (9.2, 9.3).

**Rounding.** Chain A rescaled, one region, scaled to `B_ref = 1 000`, floor 50
(`BeliefStep` 200), half to even: seed 3 353, grain 373, flour 342, dough 277, loaf
376, fuel 248, iron 166, hoe 3 146, millstone 3 312, oven 3 312, the byproducts straw
and bran at the floor 50. The exact wage is 2 980.47 per working tick (households work
33.6% of ticks, so 1 000 a tick). Value added per execution against the wage times the
duration is within 0.25% for every active activity except `mill_flour` (+10.16%,
the bran at the floor adds revenue) and `grow_grain` (+1.58%). These digits were
recomputed with `round.py` for the start rule of 3.1 (a start counts at every tick of the
open window): they stand, and `bake_loaf` has `J` 183 = ceil(365 / 2) there.

**Seed prices against unit cost.** Two versions, because the unit cost changed
(`spec-production.md` 9, N-055).

- *Inputs and wear only* (the first draft of this spec): grain 2.78, flour 3.66, dough
  9.7, loaf 8.4, hoe 19 times the unit cost. The default `CostCeiling` of 3 times lay
  below four of the five. Kept as the record of the finding.
- *Full cost, labour at the solver's wage* (the rule now in force), re-run for this
  revision by `sweep.py` at the integer seed prices, rounded up as production does.
  Chain A, one region, wage 2 980 per working tick: grain 373 over 373, flour 342 over
  342, dough 277 over 277, loaf 376 over 376, hoe 3 146 over 3 146, millstone and oven
  3 312 over 3 312, iron 166 over 166, seed 3 353 over 3 353: ratio 1.000 each; fuel
  248 over 249, 0.996. Chain B, one region, wage 1 966: biomass 327 over 327, board
  12 364 over 12 363, drill 14 330 over 14 330, fabricator 41 155 over 41 155, regolith
  66 over 66, wafer 2 163 over 2 164, solder 2 097 over 2 098, water 346 over 347, ice
  112 over 113: ratios 0.991 to 1.000. So the seed sits at the unit cost, inside the
  band of `CostFloor` 7 000 to `CostCeiling` 30 000, and the finding is resolved.
  The differences from 1.000 are the rounding of prices to whole minor units.

**The examples of `spec-production.md` 12.** The quantities there are the values of
this section, and the recipes were compared line by line with the prototype's. Re-run
for this revision (`sweep.py`): chain A, one region, `lambda*` 2.980, chain B, one
region, 1.966, so both still pass V3 at full labour. The three-region bread values
(2.980, 2.517, 3.124, pooled 3.124) come from the earlier run of the same prototype
code with the same quantities; they were not re-run, because nothing they depend on
changed (note that the wood yields by region in that run are not in
`spec-production.md` 12.1, which lists only grain and iron). FR-VIA-048 still holds.

**Cost.** A synthetic realm, 5 regions, 30 commodities, 45 recipes, 1 000 households per
region, pooled: 306 pivots, 2.4 seconds in Python with fractions. This says the
tableau is small, not how fast Go is. The reference chain `five` of
`spec-reference-chain.md` 4.4.4 (30 commodities, 45 recipes, 5 regions, 5 000
households) needed 202 pivots and 1.7 seconds.

**Round 5: V1, V2 and V4 in the prototype.** The prototype above ran only the programme of
V3. The prototype of `spec-reference-chain.md` 14 (`/tmp/economy-boss/ref/lp.py`, Docker)
adds the producibility fixpoint of V1 on existing activities, the components and exact
programme of V2 and the supply-gap scan of V4, and the two example chains of
`spec-production.md` 12 were run through all of them. Results: example A as written
(unrescaled quantities aside, the rescaled chain of this section) fails V1 (`seed`,
`grain`, `flour`, `dough`, `loaf` unproducible: the seed cycle has no source) and has a seed
cycle surplus `s*` of 0.92; with a wild-seed source recipe V1 passes, `s*` is 1.92 and
`lambda*` is 2.981. Example B as written fails V1 (`fabricator`, `board`, `drill`, `ice`,
`water`, `biomass`: the tool cycle has no source) with `s*` 0.20 for board and fabricator;
with a `salvage` recipe V1 passes, `s*` is 0.5 and `lambda*` stays 1.966. So V2 alone did not
see the defect (a cycle can be productive and have no way in), and V1 is what rejects it
(`spec-production.md` 10.6). V4 passes for both. The programme with wants (3.2) was run
for the demand pack of the reference chain: `lambda*` 2.040 against 2.512 without the
wants. The nominal fallback of 5.3.4 was run on the one-region pack with the staple yield
at 4 000, 3 800, 3 000 and 2 500: at full labour the wage is 755, 0, 0 and 0; nominal it
is 1 868, 1 826, 1 624 and 755.

The `s*` figures of this round (0.92, 1.92, 0.50, 0.20) come from the prototype of
`spec-reference-chain.md` 14, which is not archived (N-274); they are the record of that
run and are not recomputed here. A test of V2 asserts only what can be derived by hand:
a one-recipe seed cycle that takes 1 `seed` and returns 1 has `s* = 0` (not productive),
and one that returns 2 has `s* = 1 = 2 - 1` (productive); with a source recipe in the
component the programme of 4.3 takes it as an activity.

## 16. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-VIA-001 | The labour of an execution MUST be the recipe's `Duration` in household-ticks, and the net vector of an activity MUST be `E * q / 10 000` per non-discard output minus the input quantities minus `WearPerUse / Durability` per required tool, discard outputs counting for nothing. |
| FR-VIA-002 | The start capacity `J` of an activity MUST be the number of jobs one household can start in a year counted greedily from the first tick with a positive yield factor, a start counting at every tick of the open window whether or not the job completes inside the year, and an activity with `J = 0` MUST NOT exist. |
| FR-VIA-003 | The reference programme MUST maximise `lambda` subject to the goods, labour and capacity rows of 3.2, and `SurplusBP` MUST be `floor(lambda* * 10 000)`. |
| FR-VIA-004 | A configuration MUST be rejected with `ErrNotViable` and the rule `unproducible_in_realm` when an upkeep good, or an input or tool of a recipe that exists in a region where a role lists it, cannot be produced in any region with a positive yield. |
| FR-VIA-005 | A configuration MUST be rejected with the rule `role_cannot_work` when a role that lists recipes has households in a region where none of its recipes has a positive yield; a role that lists none MUST be exempt. |
| FR-VIA-006 | A configuration MUST be rejected with the rule `cycle_not_productive` when a strongly connected component of the material graph has no non-negative activity vector whose net product is positive in every commodity of the component, computed in exact arithmetic with the best yield across regions. |
| FR-VIA-007 | A configuration MUST be rejected with the rule `labour_short` when the pooled free-trade programme has `lambda* < 1` at `phi = 10 000` and V1 and V2 found nothing; V3 MUST be skipped silently for a programme with an empty basket or whose basket goods are all unsourced `Raw` markers. |
| FR-VIA-008 | A configuration MUST be rejected with the rule `storage_gap` when an upkeep good has a shelf life above 0 and below its longest supply gap in the region, and MUST warn with `storage_gap_input` for an input of an all-year activity. |
| FR-VIA-009 | `Validate` MUST warn, and MUST NOT fail, with `region_not_self_sufficient`, `subsistence_exceeds_expected_work`, `target_above_surplus` and `viability_check_skipped` under the conditions of 10.2. |
| FR-VIA-010 | `Validate` MUST skip the programmes of V2 and V3 and warn with `viability_check_skipped`, naming the part skipped, when the tableau would exceed `MaxViabilityCells` rows times columns (rows and columns as in 4.3, computed with the checked helpers) or the pivots would exceed `MaxViabilityPivots`, and MUST NOT skip V1 or V4; `SolveSteadyState` MUST return `ErrSolveLimit` carrying the pivots, rows and columns for the same overshoot. |
| FR-VIA-011 | Every error of V1 to V4 MUST match `errors.Is(err, ErrNotViable)` and be a `*ViabilityError` with a `Rule`, a `Path` and the offending value. |
| FR-VIA-012 | The labour share to subsist MUST be `ceil(10 000 * 10 000 / SurplusBP)` at full labour when no capacity row is tight, and absent otherwise. |
| FR-VIA-013 | The dual of the reference programme MUST give prices and wages that satisfy dual feasibility for every activity and equality for every active activity, checked in exact arithmetic. |
| FR-VIA-014 | The solver MUST return a result only after checking primal feasibility, dual feasibility and equality of the objectives in exact arithmetic, and otherwise MUST return `ErrSolveInconsistent`. |
| FR-VIA-015 | The solver MUST use no floating-point type or function, and MUST use exact integers of unbounded size internally. |
| FR-VIA-016 | The solver MUST use Bland's rule, and MUST terminate on every input. |
| FR-VIA-017 | The solver MUST return `ErrSolveLimit` and no table when `MaxPivots` is reached, and the wrapped `ctx.Err()` and no table when the context is cancelled, checking the context between pivots. |
| FR-VIA-018 | The solver MUST return `ErrNoUpkeep` when no group has an upkeep. |
| FR-VIA-019 | The solver MUST report `Degenerate`, `AlternativeOptima` and the commodities whose price is 0, in a stable order. |
| FR-VIA-020 | `SolveSteadyState` MUST be a pure function of its arguments: it MUST draw no random number, MUST write no log, MUST start no goroutine, and MUST give the same result on every run and platform. |
| FR-VIA-021 | `NewWorld` MUST NOT call `SolveSteadyState`. |
| FR-VIA-022 | The tableau MUST be built in canonical order, commodities by ID and activities by region then recipe ID, so that the result does not depend on the order of the configuration. |
| FR-VIA-023 | The seed price MUST be the solver's price times the scale of 6.3, rounded half to even, raised to the resolution floor and capped at `MaxPrice`, and a price floored by the resolution floor or a free good MUST be lifted to the lower bound of the reference band (and one above its upper bound capped to it) so that the table passes `Validate`; each such change MUST be listed in `Floored` and the unclamped price recorded in `OutsideReferenceBand`. |
| FR-VIA-024 | The price scale MUST make the regional basket (autarky) or the household-weighted baskets (pooled) equal the configured reference basket values. |
| FR-VIA-025 | The report MUST give, per commodity, the drift of the seed from the effective reference price, and the largest residual in basis points of the value added of an active activity at the integer prices against the wage times the duration. |
| FR-VIA-026 | A commodity with a non-unique price MUST have its reference price in the table, and one with a price of 0 MUST have the larger of the resolution floor and the lower bound of the reference band; both MUST be listed in `UnpricedCommodities`. |
| FR-VIA-027 | The report MUST list every commodity whose seed price is below `UnitCost * CostFloor` or above `UnitCost * CostCeiling` at the seed prices, where `UnitCost` is the full unit cost of `spec-production.md` 9.1 (inputs, tool wear and `Duration` working ticks at the solver's wage, rounded up, computed from the integer seed prices). (changed: it was the cost of inputs and wear) |
| FR-VIA-028 | The solver MUST report, for each group, its need in households (the larger of the labour of its recipes and the start-capacity term of 5.2, at the flows of lambda 1, with no available work share in either term), the balanced census (the largest-remainder apportionment of the region's configured households in proportion to need, ties to the earlier group in canonical order) and the imbalance (configured minus balanced), and per recipe where a recipe is on the lists of several roles of a region; and MUST NOT change the census. |
| FR-VIA-029 | `Config.WarmStart` MUST be nil by default, MUST be part of the effective configuration and the hash, and MUST be plain data with sorted slices and no map. |
| FR-VIA-030 | A seed price MUST be within `[1, MaxPrice]`, at or above the resolution floor and within the reference band, or the configuration MUST be rejected with `ErrWarmStartPrice`. |
| FR-VIA-031 | A seed price MUST replace the effective reference price as the market's initial published price and as the initial centre of each initial agent's belief, clamped to the bounds. |
| FR-VIA-032 | A group's `Money` MUST be per household, MUST be multiplied by the agent's weight, and the sum over the world MUST be the money created by `initial` at construction, with no other money created. |
| FR-VIA-033 | A `JobSpec` MUST be rejected with `ErrWarmStartJob` when the recipe is not on the role's list, the share is outside [0, 10 000], the shares of a group exceed 10 000, or the set of virtual start offsets is empty. |
| FR-VIA-034 | The agents of a group in progress MUST be the first `busy` agents in agent order, `busy` coming from one joint largest-remainder apportionment over the group's agents (quotas `share * n / 10 000` per `JobSpec` in canonical order, the idle share last, floors first, ties to the earlier entry) with the busy agents assigned spec by spec, and the `j`th MUST take the offset `V[floor(j * |V| / busy)]` of the sorted virtual start offsets. |
| FR-VIA-035 | A job created from a `JobSpec` MUST have `Remaining = Duration - o`, the effective factor of the season of the virtual start tick, `Executions` equal to the agent's weight, and the agent itself as principal. |
| FR-VIA-036 | Construction MUST grant the inputs of each initial job by `initial` and take them by `recipe_input` in the same step and wear its tools, and goods MUST be conserved by audit from tick 0. |
| FR-VIA-037 | The stagger MUST give agent `i` of `n` the phase `floor(i * P / n)` for a job duration, a tool durability and a perishable shelf life, using no random number. |
| FR-VIA-038 | A durable in a group's warm stock MUST have remaining durability `Durability - floor(i * Durability / n)` for agent `i`, and a perishable lot MUST have `Born = -floor(i * ShelfLife / n)`. |
| FR-VIA-039 | Stock cover or reserve that exceeds a commodity's shelf life MUST be cut to it and listed in the report. |
| FR-VIA-040 | Applying the warm start MUST draw no random number, and MUST leave the random stream of agent creation as it would be without a warm start. |
| FR-VIA-041 | `NewWorld` MUST reject a warm-start table with an unknown reference, a duplicate key, or an amount that overflows its bound, using the sentinel for that kind. |
| FR-VIA-042 | A group absent from the table MUST take the role's defaults, and a price absent MUST take the reference price. |
| FR-VIA-043 | With lanes, each lane MUST be an activity with the net vector and labour of 9.4, and a configuration with no lanes MUST be solved in autarky. |
| FR-VIA-044 | `WithFreeTrade` MUST add a lane of labour 0 and loss 0 for every ordered pair of regions and commodity, and `Validate` MUST use that case for the pooled check. |
| FR-VIA-045 | The report MUST list, for the pooled solution, the regions in which each activity is active, and the pooled relative price of each reported pair against the range of the autarky ratios, flagging `ratio_outside_autarky`. |
| FR-VIA-046 | `NewWorld` MUST log `warm start applied` once at `Info` with the counts of price rows, wage rows, groups and jobs, and MUST log no other line for the warm start. |
| FR-VIA-047 | The warm start MUST NOT change any rule of the simulation after tick 0, and a world built with the table and one built without MUST differ only in the initial state. |
| FR-VIA-048 | The two example packs of `spec-production.md` 12, completed and rescaled, MUST pass V1 to V4, solve, produce a table that loads and run, with no Go code specific to either. |
| FR-VIA-049 | `WarmStart.Wages` MUST be a list of at most one `SeedWage` per region, sorted by region, each a wage in minor units per working household-tick, part of the effective configuration and the hash. |
| FR-VIA-050 | The solver MUST set the `SeedWage` of a region to the shadow price of its labour row scaled by the price scale of 6.3 and rounded half to even, MUST write no entry where that wage is 0, and MUST list such regions in `Report.UnpricedWages`. |
| FR-VIA-051 | A `SeedWage` MUST be rejected with `ErrWarmStartPrice` outside `[1, MaxMoney]` or when its product with the longest recipe `Duration` exceeds `MaxMoney`, and two wages for one region with `ErrDuplicateID`. |
| FR-VIA-052 | The world MUST use a region's `SeedWage` as the pooled observed wage of the cost anchor whenever the pooled figure is unknown (no agent of the region has a working household-tick in its window, or the figure is not positive) and MUST NOT latch the choice, so the figure is a function of the welfare rings and the configuration alone; a region with no `SeedWage` MUST have an unknown wage while the pooled figure is unknown. |
| FR-VIA-053 | The season of a virtual start tick `-o` MUST be `Calendar.SeasonAt(-o)`, and the solver and the world MUST NOT compute it by any other rule. |
| FR-VIA-054 | The seed-band check of FR-VIA-027 MUST use the solver's own wage, so that the check and the starting wage agree, and for the rescaled example chains the seed over the full unit cost MUST be within 1% of 1 at the integer prices (the prototype gives 0.991 to 1.000). |
| FR-VIA-055 | With the demand module off the goods rows of the reference programme MUST be unchanged, and with it on they MUST carry the fixed term `W(c, P)` of 3.2, not multiplied by `lambda`, computed as an exact fraction from the groups' households, tastes, region tastes, quantities, periods and item weights. |
| FR-VIA-056 | The error rule `labour_short` MUST use the programme without the wants, and the programme with the wants MUST be used for the census, the seed table and `SurplusWantsBP`. |
| FR-VIA-057 | `Validate` MUST warn, and MUST NOT fail, with `wants_exceed_labour` when the demand module is on and the programme with the wants has `lambda* < 1` while the upkeep-only programme has `lambda* >= 1`, or when the programme with the wants is infeasible (3.2) while the upkeep-only one has `lambda* >= 1`, and with `raw_without_source` for a commodity marked `Raw` that nothing real supplies. |
| FR-VIA-058 | For a region whose labour dual at full labour is 0 while a capacity row has a positive dual, the solver MUST build its seeds and wage from the nominal programme, MUST list the region in `NominalSeed`, and MUST leave a region whose nominal wage is also 0 unpriced (`UnpricedWages`). |
| FR-VIA-059 | A commodity marked `Raw` MUST count as sourced in V1 as in the structural check of `spec-config.md` 7.5, so the two checks MUST agree on every configuration. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-VIA-001 | The solver MUST allocate only outside the tick path, and MUST NOT be counted in the allocation limits of observability-and-logging.md 8.2. |
| NFR-VIA-002 | The V1 and V4 checks SHOULD run in time proportional to commodities times recipes, and the whole of `Validate` including V2 and V3 SHOULD finish in one second at the maximum sizes, by the size gate. Provisional. |
| NFR-VIA-003 | The solver on 5 regions, 30 commodities and about 45 recipes SHOULD have a benchmark with `b.ReportAllocs()`. |
| NFR-VIA-004 | The solver, the stagger and the table validators MUST have fuzz tests (testing.md 7.4) asserting that they never panic and that every result is an error or a valid table. |

## 17. Observability

17.1. **No event in the tick path.** The warm start is applied at construction. The
initial `AgentSpawned` events carry the starting money of each agent
(`spec-estates.md` 2.5); no other event is added. `WorldCreated` carries a boolean
`WarmStart`, which observability-and-logging.md 4.3 lists.

17.2. **Logs.** One `Info` line at construction, "warm start applied", with
`regions` (a count), `prices`, `wages`, `groups` and `jobs` counts (FR-VIA-046); the last four are in the table of observability-and-logging.md 3.3.2, and `regions` is the count key also used by `spec-config.md` 10.1. The new warnings use the
mechanism of `spec-config.md` 7.2 (one `Warn` "configuration warning" per warning
with `rule` and `path`), bounded by the size of the configuration. The solver is a
pure function and logs nothing; it returns its diagnostics in the `Report`.

17.3. **Errors returned, never logged** (architecture.md 8.3).

17.4. **No new metric.** A consumer can build a gauge of the seed from the first
`MarketCleared` events.

## 18. State contributed to a checkpoint

The warm start adds no state. `Config.WarmStart` is configuration: it is in the
hash (FR-VIA-029), and a restore with a different table fails the hash check. The
effect of the warm start is ordinary state after tick 0 (inventories, jobs,
market prices, beliefs) which the other specs save. The solver's output is not
saved, and the solver has no state.

## 19. Entries for the stability register

For `spec-stability.md` (a sign, a delay, a damper and a test are given when that
spec is written):

- **Lockstep of jobs.** Agents that start together complete together. Damper: the
  stagger of 8.4.3 and 8.5. Test: A8.
- **The closed-economy identity and the target ratio** (4.7). With upkeep as the only
  demand the mean welfare ratio is 1, targets above 1 never rest, and the surplus is
  produced anyway. Sign: positive (more work, more supply, lower price, lower
  income, more work). Damper: demand beyond upkeep (DEM), a faucet, or price floors.
  Test: a run with the default `TargetRatio` and no DEM asserts the mean ratio near 1
  and a bounded glut; this is the same loop as N-028 seen from the other side.
- **A glut at `lambda* > 1`.** The surplus capacity has no buyer; prices of the
  surplus goods fall toward the floor, and producers leave. Same damper.
- **A warm start is not a stable start.** Starting at the solver's point removes the
  transient of bad beliefs, not the dynamic instability. Test: A1 against a cold start.
- **Census imbalance.** A configured census that differs from the balanced one starts
  a transient of role changes (SPN). Test: the same chain at the balanced census and
  at 20% off, asserting the occupation shares return.
- **The cost ceiling against the equilibrium price** (8.2.3): resolved by counting
  labour in the unit cost (`spec-production.md` 9; STB-L52 resolved). What remains is
  the floor: the seed equals the full unit cost, so a `CostFloor` of 10 000 binds at
  the steady state (STB-L09, STB-L10), and BEL lowered it to 7 000. Test: the seed is
  inside the band for the reference chain, and a wage 25% above the solver's still
  leaves the floor below the seed.
- **Rounding residual.** A free good at the floor changes the wage balance of its
  producer (+10% on `mill_flour`). Test: the residual stays under `ResidualWarnBP`
  for the reference chain.
- **Seasonal storage.** A gap longer than the shelf life makes a famine that no
  price can cure (V4). Test: A2.

## 20. Open questions

1. **Prices when there is demand beyond upkeep. Answered in round 5.** The programme is
   kept as the subsistence part and the wants of `spec-demand.md` enter as a fixed final
   demand at their reference level (3.2); the prices are then the prices at that demand,
   not an equilibrium of the price-responsive schedule. A convex programme with the
   schedules would replace the Leontief rows and is not needed for a warm start, which
   is a start and not an anchor. *Next step:* compare, in S23 to S25 of
   `spec-demand.md` 13, the run's prices with the programme's.
2. **Annual mean against a seasonal path.** The programme ignores timing apart from
   `J` and the lean reserve. A time-expanded programme (one period per season) would
   make storage endogenous and cost a factor of the number of seasons in size.
3. **The census.** The solver reports and does not change it. Whether the author
   should be offered a rewrite (a `BalancedCensus` that produces a `Config`) is for
   CLB.
4. **More than one recipe per role.** The solver lets households pick any recipe of
   the region; AGT's rule picks by expected net value. If the two disagree the warm
   start's jobs are not what the strategy would start. Check once AGT exists.
5. **Lanes.** Answered for transport: `transport.Lanes` supplies them (`spec-transport.md` 9.4; labour only, with the expected loss of `spec-risk.md` 8.3: 0 with the risk module off). Who uses them for a merchant's decision belongs to MER.
6. **A default pool.** Whether the warm start seeds the estate pool by default
   (8.3.3) so the first replacements have money.
7. **Exact arithmetic in Go.** `math/big` is standard library and exact; whether its
   speed is enough for load-time checks at the size gate is not measured. The
   permission question is answered: architecture.md 3.5 allows `math/big` in
   `internal/steady` (6.1, N-058). The wording difference is closed: architecture.md 3.5
   now says the result converts to fixed point "by the rounding rule stated for that field"
   (prices and wages half to even, a surplus down, a labour share up), which is what 3 and
   6.3 say.
8. **Joint production.** The programme counts a byproduct's value as revenue; the
   belief rule allocates cost by `CostShare`. They agree when prices are the solver's;
   they may not at other prices.
9. **Imports and a closed region.** Whether an author can declare a region closed
   (no lanes ever) so that autarky failure is an error and not a warning.
10. **The labour of the first job.** A household that is a carrier or a trader has
    labour that is not in the reference programme; the lanes charge it, the trader
    role is not modelled.
11. **Floor for free goods.** A byproduct at the resolution floor adds revenue the
    exact solution lacks (6.5). Whether it should be subtracted before the
    comparison is a calibration question.
12. **Skill premium and the common wage.** The starting wage and the cost of 9.1 use
    one wage per region (`spec-production.md` 9.3, open question 10 there). A skilled
    recipe is costed at the common wage; whether the solver should report a wage per
    recipe class is for SKL.
13. **A warm job draws no endowment.** `spec-world.md` 5.6.1 item 4 says that no
    endowment is drawn for a job created at construction (the virtual start precedes
    the run) and asks VIA to confirm, because 8.5.3 grants the job's inputs and says
    nothing of the resources a recipe with `Draws` takes. *Recommendation:* accept. The
    stock of an endowment at tick 0 is its `InitialBP` (8.4.5), which already states
    what is left, so a draw made for the warm job would count the same units twice.
    *Next step:* VIA confirms, and 8.5.3 gets one sentence.

## 21. Where this lives in the code

| Concern | Home |
|---------|------|
| `WarmStart`, `SeedPrice`, `SeedWage`, `WarmGroup`, `JobSpec`, `WarmPool` | root package, `config.go`, with `Config` (architecture.md 2.6: `config.go` holds "Config and every section struct"; an earlier draft said `viability.go`) |
| `SteadyState`, `ProgrammeResult`, `Report`, `SolveOption` and the five `With*` options | root package, `viability.go`. Stage 1 publishes no `Lane`, `Lanes` or `MaxLanes`: they come with the transport module (9.4) |
| `SolveSteadyState` | root package, `viability.go`, converting inline to `internal/steady` |
| The checks of V1 to V4 | one unexported `checkViability`, in `viability_check.go`; `Validate` and `Warnings` both call it (`spec-config.md` 7.1.6), and it is not exported (architecture.md 2.8) |
| The table from the solved programme | root package, `viability_table.go` |
| The file split | the concept is split in three files, `viability.go` (the exported surface), `viability_check.go` and `viability_table.go`, for the size limits of N-243; a continuation file maps to this area code (`architecture.md` 2.6) |
| The tableau, Bland's rule, the certificate, the greedy start count, the component search, the programme per component | `internal/steady`: stdlib only (`math/big`), primitive slices (architecture.md 2.9) |
| The stagger and the application of the table | root package, `world.go`, `agent.go` at construction |
| The bounds of section 11 | root package, `bounds.go` |
| Sentinels (`ErrNotViable`, `ErrNoUpkeep`, `ErrSolveLimit`, `ErrSolveInconsistent`, `ErrWarmStartPrice`, `ErrWarmStartJob`) and `ViabilityError` | root package, `errors.go` |
| The rules of section 10 | root package, `validate.go` |
| The file mapping of the section | `config/` |
| The reason constants (exported from the root package, `event.go`) and their key strings | the root package for the constants; `internal/logkeys` holds the key strings only |
