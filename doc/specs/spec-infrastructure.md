# Spec: Infrastructure: a facility as fixed capital with a condition, built and repaired by ordinary recipes, and four effects: a site that a recipe needs, a road that speeds an edge, a ground improvement that raises an endowment's capacity, and a store

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CAP-NNN` and
`NFR-CAP-NNN`, which tests cite (testing.md section 4). They are in section 20.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md),
decision 2). It is off unless `Config.Modules.Infrastructure` is present, and with it off
nothing in this spec runs: no facility exists, no recipe may name one, no position is
taken, no event is emitted, no state is saved, and the economy is the one in which
capital is only a tool held by one household (`spec-production.md` 4.4) and a road is
the configured distance. A world with the module on and no class configured is exactly
the world without it, except the configuration hash, `Modules` and `Names` (FR-CAP-001). It needs the region graph (`spec-regions.md` 3, 4, 5.1),
production (`spec-production.md` 3, 4, 5), resources (`spec-resources.md` 3, 6.3) and,
for the kinds that use them, transport (`spec-transport.md` 4, 10.2, 10.3), storage
(`spec-storage.md` 3.6.3) and settlements (`spec-settlements.md` 6). It uses hooks in
the world (`spec-world.md` 6, 8), integration (`spec-integration.md` 3, 4, 5, 6), the
agent's view (`spec-agents.md` 6), hired jobs (`spec-employment.md` 6.1), the checkpoint and
observability. It reserves seams for credit (`spec-credit.md`), risk (`spec-risk.md`), institutions
(`spec-institutions.md`), fiscal (`spec-fiscal.md`), saving and investment
(`spec-saving-and-investment.md`) and technology (`spec-technology.md`). Every change it
needs in another spec is listed in the handoff and is not made here (section 19 states
them).

Nothing has run in the library and no acceptance band exists (notes N-015, N-027).
Every number here is a **placeholder**: a default, a bound or a number of an example,
and none is called stable. The one script of section 17 is a stylised Docker Python
model of the rules of sections 3 to 6, not a claim about the library. The settings of
the worked examples (18) are invented **examples**, and the words in them (`mill`,
`miller`, `road`, `cart`, `drain`, `arable`, `barn`, `grain`, `timber`) are flagged as
such and are data, never branched on (`AGENTS.md`, N-017). The normative text says
**site**, **route**, **ground** and **store**.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

Until this module, capital is a tool: a durable that one household holds in its own
account and wears by use (`spec-production.md` 4.4). That cannot say that a mill is in a
place and works up to a rate, that a road lasts and is worn by weather, that a drain
raises what a field can hold, or that a barn is built before it is used. A fixed
capital good has a **location**, a **capacity**, a **condition** that falls and can be
restored, and an **effect** that is not a recipe's output. `spec-production.md` 6.7
deferred "a condition on a building or other fixed capital" to this spec, and
`spec-transport.md` 10.2, `spec-resources.md` 10 and `spec-storage.md` 3.6.3 each
reserved a hook for it. This spec covers:

- what a **facility class** and a **facility** are, and what condition, wear and the
  effect fraction mean (section 3);
- how a facility is **built** and **repaired**: three clauses of the recipe format, the
  refusals, who pays and what the default strategy does (section 4);
- the four **effects**, each through the seam the owner reserved (section 5);
- **time**: decay, ruin and the one new position of the tick (section 6);
- **owners over time**: removal, role change, weight, relocation and a restore
  (section 7);
- the **gate** by settlement class (section 8);
- what an agent **sees and does** (section 9);
- the position in the tick and the draws (10), the public calls and queries (11),
  parameters (12), validation (13), observability (14), the state saved (15), the
  register entries (16), what was run (17), worked examples (18), notes settled and
  needs (19), requirements (20), open questions (21) and the home in the code (22).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| Whether, when and where an agent decides to build or repair; the margin test, the saving that funds it | the agent's `Strategy`; a consumer's, the player's directive (`spec-integration.md` 4.3), or `spec-saving-and-investment.md` (SAV). The default strategy builds and repairs nothing for a role with no `Saver` block (4.5) |
| A treasury that builds a road, a toll, a levy on use, a tariff at a gate | `spec-fiscal.md` (FIS), `spec-policy.md` (PLC), with the jurisdiction of `spec-polity.md` (POL) |
| A loan to build | `spec-credit.md` (CRD): a borrowed sum is ordinary money at the purchase of the inputs |
| A facility lost to a raid, a flood or a fire; the `loss` sink | `spec-risk.md` (RSK). This spec names what RSK may read (5.5) and the call that lowers a condition (11.2) |
| A price for the use of a facility, a lease, a charter or a monopoly on one | not in version 1 (open question 2): a fee is FIS's or ORG's, a lease is a contract over a facility in the shape of `spec-merchants.md` 9 |
| Productivity growth: a facility is not a technology | `spec-technology.md` (TEC): a yield factor, not a facility |
| A sale of a facility for a price | SAV and CRD, as for a title (`spec-land.md` 6.3); this spec has a transfer by decree (11.3) |
| A road that carries a toll, a bridge with a capacity, a queue | not in version 1: TRN has no edge limit (`spec-transport.md` 6, 10.2; open question 4) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | A facility is **module state, not goods**. It is not in the ledger, has no price, no mass, no lot and no account; it is a record with an owner, a region, a class and a condition. Nothing in the ledger changes: no reason, no account kind | 3.2 |
| 2 | A class has one of **four closed kinds**: `site`, `route`, `ground`, `store`. Each is one of the seams that the owners reserved, and no code branches on a class identifier (architecture.md 9.1) | 3.1, 5 |
| 3 | **Condition** is an integer of wear units, from a full `Durability` down to 0. It falls by a fixed `Decay` each tick and, for a site, by `WearPerUse` for each execution it serves; a facility at 0 is **ruined** and removed. The effect is proportional to the condition, so a worn facility is a weaker one and there is no cliff but the last unit | 3.3, 6 |
| 4 | A facility is **built by a recipe**: a new clause `Builds` replaces the outputs, the inputs and the duration are the ordinary ones, and the builder pays by the inputs it consumes and the labour of the job. It is **repaired by a recipe** with the clause `Repairs`. No new job type, no new account, no new reason | 4 |
| 5 | A build or repair recipe **must have an input**: capital is never free, and the default strategy takes a recipe with no inputs for a source (`spec-agents.md` 7.1.4) | 4.1, 13.1 |
| 6 | The **default strategy builds and repairs nothing for a role with no `Saver` block**: such a recipe has no output value, so its net value is negative and it is never eligible by the choice rule. The decision belongs to a consumer's strategy, the player's directive and SAV (the project rule of a role with a `Saver` block requests one). A pack that wants capital at the start lists **initial facilities** in the configuration | 4.5, 3.4.2 |
| 7 | **Use is ownership.** A site is used by its owner only; sharing a mill is a trade in goods (the miller buys grain and sells flour). A route and a ground improvement are public: their effect does not ask who owns them | 3.4, 5.1 |
| 8 | An effect is a **layer multiplied into the owner's formula**, never written into the owner's slot: the transit factor is `F_slot * (10 000 + B) / 10 000`, the effective capacity is `capacity * scale * (10 000 + B) / 10^8`. A consumer's shock and a facility's gain therefore compose, and a closed road stays closed | 5.2, 5.3 |
| 9 | Gains **add** (each facility's gain is its effect fraction) and the sum is capped, so capital has a ceiling and no product of gains explodes | 5.2.2, 12.2 |
| 10 | The **one new position** is P8.3b, after regeneration and after LND's rent step P8.3a (`spec-land.md` 8.1), before the audit: decay, ruin, the retry of a store not yet installed, the rebuild of the effect tables and the ground clamp. A new facility, a repair and a wear take effect from the next P8.3b; every position of a tick sees one set of effects. A call between ticks rebuilds the tables at the call | 6, 10 |
| 11 | The module takes **no random number**, adds **no ledger reason** and declares **no money flow**: its cost is the inputs of a recipe, which `recipe_input` already carries | 10.3 |
| 12 | A facility is **tied to no one's region**: it stays where it was built, and its owner may leave it. No `region_bound` refusal is registered for a site, a route or a ground improvement (N-090, 7.5). A store is STO's `has_store` | 7.5 |
| 13 | The **owner's removal** passes a facility to the estate of its region: a route and a ground improvement go on working, a site idles, a store facility is ruined with its owner's account (`owner_removed`). A site, route or ground facility is never destroyed by the death of its owner | 7.1 |
| 14 | A **settlement class** can gate a build through a flag named by the class, tested where the build is decided (the start of the job) and at an `AddFacilities` call. It never removes a facility that exists (SET R3) | 8 |
| 15 | The silent limits are **counted**: a start cut by capacity, a start refused, a completion that created nothing and a store not yet installed are in one aggregate event, never per agent | 14.1 |
| 16 | The public face is **three calls between ticks**: add (by decree, for a scenario or a first consumer's scripted world), adjust the condition (a flood, a restoration) and transfer (a gift or a succession). Each is atomic, emits an event at the call and is saved | 11 |

## 3. The facility class and the facility

### 3.1 Classes and kinds

3.1.1. A **facility class** is data, keyed by ID like every configured entity
(architecture.md 9.1; `spec-config.md` 3.3). Its **kind** is one of four constants
(a closed enumerated set, as the shapes of `spec-money-supply.md` 4 are, and not an
identifier):

```go
// FacilityClass is a kind of facility, with the fields its kind uses.
type FacilityClass struct {
	ID         string   // matches the ID pattern of spec-config.md 3.3
	Kind       string   // "site", "route", "ground" or "store"
	Durability int64    // condition units of a new facility; 2 to MaxDurability
	Decay      int64    // units lost each tick, used or not; 0 to Durability - 1
	WearPerUse int64    // site only: units lost for each execution served; 0 to Durability
	Capacity   int64    // site only: executions per principal and tick that a whole facility serves; 1 to MaxFacilityCapacity
	BoostBP    int64    // route and ground: the gain of a whole facility, basis points; 1 to MaxBoostBP
	Modes      []string // route only: the transport modes it speeds; 1 to MaxRouteModes
	Resource   string   // ground only: the resource whose endowment it raises
	Store      string   // store only: the store class it installs (spec-storage.md 3.2)
	Needs      string   // a settlement flag the region's class must carry to build one; "" none (8)
	Label      string   // presentation only; not hashed
}
```

3.1.2. **Which fields a kind uses.** A field that the kind does not use must be its
zero value (`ErrFacilityShape`, 13.1); a field it requires must be set.

| Kind | Requires | Optional | Must be zero |
|------|----------|----------|--------------|
| `site` | `Durability`, `Capacity` | `Decay`, `WearPerUse`, `Needs` | `BoostBP`, `Modes`, `Resource`, `Store` |
| `route` | `Durability`, `BoostBP`, `Modes` | `Decay`, `Needs` | `WearPerUse`, `Capacity`, `Resource`, `Store` |
| `ground` | `Durability`, `BoostBP`, `Resource` | `Decay`, `Needs` | `WearPerUse`, `Capacity`, `Modes`, `Store` |
| `store` | `Durability`, `Store` | `Decay`, `Needs` | `WearPerUse`, `Capacity`, `BoostBP`, `Modes`, `Resource` |

3.1.3. **Why four, and not one with options.** Each kind is the one seam an owner
reserved, and a fifth would need a spec that owns it: a site is the recipe condition
production deferred (`spec-production.md` 6.7, `spec-settlements.md` 6.6.2); a route
is the transit factor of transport (`spec-transport.md` 10.2); a ground improvement is
the capacity scale of resources (`spec-resources.md` 6.3, N-108); a store is the
operation storage left to infrastructure (`spec-storage.md` 3.6.3). A kind that is
none of these (a facility that changes a yield, a price or a belief) would give
infrastructure a hold on a spec that did not ask for it. Productivity is TEC's.

3.1.4. **`FacilityClassID`** is the position of a class in the sorted order of the
effective configuration, a named integer type of the root package (architecture.md
3.3) in the names table of `WorldCreated` (14.1), as `StoreClassID` is
(`spec-storage.md` 3.6.2). A consumer persists the string ID and resolves it once from
`Config()` (`spec-integration.md` 3.4.2).

### 3.2 The facility

3.2.1. A **facility** is a record of the world's, not a good:

| Field | Meaning | Range |
|-------|---------|-------|
| `ID` | `FacilityID`, an `int64`, assigned in creation order from 1 and never reused | 1 up |
| `Class` | the `FacilityClassID` | 0 to the count of classes less 1 |
| `Region` | the `RegionID` where it stands, fixed at creation | an ordinary region (never an outside node) |
| `To` | route only: the `RegionID` of the other end of the edge it speeds, or `NoRegion` for every edge that leaves `Region` | |
| `Owner` | the `AgentID` that holds it: a household agent, an external agent, or the estate pseudo-agent of `Region` (whose ID equals the `RegionID`, `spec-world.md` 5.3) | |
| `Condition` | wear units left | 1 to `Durability` (0 is ruined and the record is removed, 6.2) |
| `Built` | the tick of creation | |
| `Installed` | store only: whether `addStore` has succeeded for it (5.4) | |

The record is plain data with no pointer (architecture.md 12.4). The facilities are
held in one slice in ascending `FacilityID`, so every iteration is in that order
(architecture.md 7.5). A removal leaves the other IDs as they are.

3.2.2. **Why not a good in the ledger.** The ledger would give a facility transfer,
inheritance by the estate, wear, audit and a checkpoint for free, and was weighed. It
was refused for three reasons. A road and an irrigation work are not for sale in a
market, and a good with no `Tradable` flag was rejected (`spec-production.md` 7).
A good has no location but its account's region, and it moves with its holder; a
facility stays where it stood. And a good's wear only falls: the ledger has no
operation that restores durability, and a repair is the point of this spec. What the
ledger gives that is lost is the inheritance, which section 7 restates in one rule.

3.2.3. **Counts.** At most `MaxFacilities` in the world and `MaxFacilitiesPerAgent`
for one owner other than an estate (12.2). The estate of a region has no per-agent
bound (it inherits what its people leave), and the world bound is the one that
applies to it.

### 3.3 Condition, wear and the effect fraction

3.3.1. **Condition** falls in two ways and rises in one:

- **decay**: `Decay` units each tick at P8.3b, whether the facility is used or not
  (weather, age); a pack with no decay sets `Decay` to 0 (the default);
- **use wear**, a site only: `WearPerUse` units for each execution the facility
  serves, at the start of the job (5.1.3);
- **repair**: units restored by a repair job at its completion (4.3.2), never above
  `Durability`; and a call (11.2) may raise or lower it by decree.

A condition is an integer and never negative; it never rises above `Durability`.

3.3.2. **The effect fraction.** The share of its effect that a facility gives is its
condition over its durability, in integers:

```
eff(f)  = floor( BoostBP * Condition / Durability )              route and ground, basis points
cap(f)  = ceil( Capacity * Condition / Durability )              site, executions of a tick
```

with the product in 128 bits. A route or ground facility gives a gain from 0 to its
`BoostBP`, so a road at half its condition gives half its gain, and a facility whose
gain has rounded to 0 still stands until it is ruined. A site gives at least one
execution a tick while its condition is above 0 (the `ceil`), because capacity is a
count of whole executions and a mill that is almost worn out is still a mill; a ruined
one gives none. Both are functions of the record alone, so a restore rebuilds them
(15.2).

3.3.3. **The unit of condition** is the author's: a wear unit has no meaning outside
the class, as a tool's durability unit has none (`spec-goods.md` 5). A class with
`Durability` 1 000 and `Decay` 1 lasts a thousand ticks unattended; one with
`Durability` 36 500 and `Decay` 1 lasts a hundred years of a 365-tick year. The year
is `TicksPerYear` (`spec-calendar.md` 3.2) and a decay is per tick; a pack that wants a
per-year rate chooses the unit so that it divides.

### 3.4 Who holds a facility

3.4.1. **The owner** is one agent. It is the principal of the build job (4.3.1), an
agent that a call gave it to (11), the holder of an initial row (3.4.2), or the estate
of the region (7.1). An agent of weight `w` owns the facility as one unit, not `w`
units: a facility is not a household's possession (as a store is, `spec-storage.md`
3.3), and its count does not scale with the weight.

3.4.2. **Initial facilities.** A pack may list facilities that exist at tick 0, as
`StartStock` lists goods:

```go
// InitialFacility is a number of facilities of a class in a region at construction.
type InitialFacility struct {
	Region string // a region of the configuration; not an outside node
	Class  string // a facility class
	Toward string // route only: the region at the other end; "" for every edge leaving Region
	Role   string // the role whose agents hold them; "" for the region's estate
	Count  int64  // 1 to MaxInitialCount
}
```

A row is keyed by (`Region`, `Class`, `Toward`, `Role`): one row per key, in canonical
order, by the mechanism of `spec-regions.md` 5.1 (a table keyed by region, with the
errors of 5.3 there). The facilities of a row are created at construction with their
full `Durability`, in the order of the rows, each taking the next `FacilityID`. A row
with a `Role` gives them **round robin to the agents of the group (region, role) in
ascending agent ID**, one each in turn, so a row of 3 over 5 agents gives the first
three agents one each, and a row of 8 over 5 gives the first three two each
(`ErrFacilityHolder` if the group has no agent, 13.1). A row with no `Role` gives them
to the region's estate: a route or ground improvement then works for everyone, and a
site idles until a call transfers it (11.3). A store row needs a `Role` (a store has an
owner, `ErrFacilityShape`).

3.4.3. **Construction order.** The facilities are created after the household agents,
the warm start and the creation draws of `spec-world.md` 5.4 to 5.7 and before the
events of 5.8, at the sub-step **8b** of step 8 (`spec-world.md` 5.1.2: after the warm start and the creation draws,
before TEC's key word 8c): they take no random number, and the stream of a run does not depend on
whether the module is on (FR-CAP-002). Each emits `FacilityChanged` (kind `added`,
cause `initial`) after the last `AgentSpawned` of construction, in ascending
`FacilityID` (14.1). A store-kind row calls `addStore` for its owner at that point
(5.4).

## 4. Building and repairing

### 4.1 The three recipe clauses

4.1.1. Three optional clauses join the recipe of `spec-production.md` 3.1. They are
data, like every field of a recipe, and name a class by its ID:

```go
// FacilityBuild is the clause of a recipe whose completion creates facilities.
type FacilityBuild struct {
	Class  string // a facility class
	Toward string // route only: the region at the other end of the edge; "" for every edge leaving the region
}

// FacilityRepair is the clause of a recipe whose completion restores condition.
type FacilityRepair struct {
	Class string // a facility class
	Units int64  // condition units one execution restores; 1 to the class's Durability
}

// Fields added to Recipe (spec-production.md 3.1):
//	Needs   []string        // site classes the principal must hold a usable facility of
//	Builds  *FacilityBuild  // completion creates facilities; at most one of Builds and Repairs
//	Repairs *FacilityRepair // completion restores condition to facilities the principal owns
```

In a file they are `needs = ["mill"]`, `[recipe.<id>.builds]` and
`[recipe.<id>.repairs]` (`spec-config.md` 6.2). Production keeps the format and its
validation errors, this spec owns the values and the meaning (the split of
`spec-resources.md` 7.1.3).

4.1.2. **Rules of the clauses** (each a sentinel of 13.1):

| Rule | Statement |
|------|-----------|
| B1 | A recipe has at most one of `Builds` and `Repairs`. |
| B2 | A recipe with `Builds` or `Repairs` has at least one input. Capital is never free, and a recipe with no inputs is a **source** that the default strategy takes at no cost (`spec-agents.md` 7.1.4). |
| B3 | A recipe with `Builds` or `Repairs` has **no non-discard output**: `ErrRecipeNoOutput` (`spec-production.md` 3.2.2) is waived for it, and an output it has MUST be `Discard` (waste made and disposed in the step, `spec-production.md` 7.2: rubble is that), else `ErrFacilityShape`. A valued output would give the recipe a `value_r` that the default strategy compares with its cost (`spec-agents.md` 7.1.4), and FR-CAP-018 would no longer follow from the rule. |
| B4 | `Builds` names a class of any kind; `Toward` is set only for a `route` class, and then names a region of the configuration that is not an outside node. |
| B5 | `Repairs` names a class of any kind, and `Units` is from 1 to its `Durability`. |
| B6 | `Needs` names classes of kind `site`, each at most once, at most `MaxRecipeNeeds` of them. A recipe that `Needs` a class may also build or repair (a smithy that makes its own parts). |
| B7 | A `TraineeEligible` recipe has none of the three clauses (`ErrEligibleHasTool`, extended): a trainee's output is a fraction of a full execution (`spec-production.md` 8.2) and a facility is not divisible, and a trainee's job draws on its trainer's account, not on the trainer's site. |

4.1.3. **The yield factor** of a build or repair recipe is a test and not a quantity.
A factor of 0 for the executor's region (the yield table, or `Sited` with no profile)
refuses the start with `zero_yield` or `no_site` (production's yield-profile test, `Sited`: nothing to do with a facility class of kind `site`) as for any recipe, so a pack can say
that nothing is built in a region, as data. A positive factor does not change the
number of facilities or the units restored: those are whole by definition, and a
fraction of a facility has no meaning. The shock multiplier of `spec-production.md` 6.8
acts as a factor and so refuses a build at 0 and does nothing at a positive value.

### 4.2 The start

4.2.1. A build or repair job is an ordinary job: one per agent, started only when the
agent is free, by the strategy's `JobRequest` (`spec-agents.md` 6.3) at P2.5, with its
inputs taken at once by the `recipe_input` transform at P2.7 (`spec-production.md` 4.3),
and a duration of the recipe's `Duration` ticks. Its weight `w` runs up to `w`
executions as ever. The **principal** is the agent whose account supplies the inputs
and receives the outputs (`spec-production.md` 4.3.2, 5.3.1): the executor itself in an
ordinary job, the employer in a hired one (`spec-employment.md` 6.1). A facility is
**owned, used and repaired by the principal**, as a tool is held and worn by it
(`spec-employment.md` 6.1), so a workshop built with hired labour is the employer's,
and a hired recipe that `Needs` a class is judged against the employer's facilities.
The principal is the **builder** or the **repairer**; the region of a facility is the
principal's region at the completion (the worker and the employer stand in one region,
`spec-employment.md` 5.3).

4.2.2. **What changes in the derivation of `k0` and the refusal reasons.** The world
derives `k0` at P2.5 from the state after the continuations and completions
(`spec-world.md` 6.4). It is the least of what `spec-production.md` 4.2 and
`spec-resources.md` 5.2 list and of these, taken in this order:

| # | Limit | Applies to | `k0` is at most | When it is 0 the reason is |
|---|-------|------------|-----------------|----------------------------|
| 1 | the capacity of each needed class, **per principal and tick** | a recipe with `Needs` | for each class, `cap_c` of 5.1.2 less the executions the same principal has already committed to the class in this tick (its own request and its hires, 5.1.2) | `no_facility` |
| 2 | the gate | `Builds` of a class with `Needs` (the settlement flag) | all or nothing: the flag of the region's class (8) | `not_unlocked` |
| 3 | the room for facilities | `Builds` | `MaxFacilitiesPerAgent` less the principal's count (an estate is exempt), and `MaxFacilities` less the world's, each less the builds already started in this tick, and for a store class the room of 5.4.1 | `facility_cap` |
| 4 | the target | `Builds` of `route` (an edge from the principal's region to `Toward`, or any edge leaving it when `Toward` is empty) or of `ground` (an endowment of the class's resource in the region with an effective capacity above 0) | all or nothing | `no_target` |
| 5 | the store | `Builds` of `store`: the dry run of `addStore` for the principal (5.4.1) | all or nothing | `store_refused` |
| 6 | what there is to repair | `Repairs` | the least `m` with `m * Units` at or above the units missing from the principal's facilities of the class in the region (`sum of Durability - Condition`) | `nothing_to_repair` |

**The order of the reasons**, stated once. The new reasons are constants of the root
package (`event.go`; `internal/logkeys` holds the key strings only, N-017). A start that
has several is refused with **the first of**: `zero_yield`, `no_site`, `no_tool`,
**`no_facility`**, `no_input`, `no_endowment`, `no_title` and `rent_funds` (tenure,
`spec-land.md` 4.5.1, when that module is on), then **`not_unlocked`**,
**`facility_cap`**, **`no_target`**, **`store_refused`**, **`nothing_to_repair`**, and
`no_room` (storage, `spec-storage.md` 5.3) **last**. The five build reasons are before
`no_room` and not after it because they apply to a `Builds` or `Repairs` recipe only, and
such a recipe has no valued output (B3), so it has no room to ask for: `no_room` stays
the last reason of the order that can apply to any recipe, as STO and LND say. A
refusal starts nothing and consumes nothing, is a `Debug` line and is counted in
`FacilityLimited.Refused` (14.1).

The world also derives, for a recipe with a `Builds` or `Repairs` clause, the per-recipe
field **`Standing`**: the first reason of the order above, other than `no_input` and
`no_tool`, that would refuse a start of one execution were the inputs and tools held
(limits 1 to 6 read no input; `rent_funds` is judged at the money held), or empty
(`spec-saving-and-investment.md` 5.2).

**Evaluated again at P2.7.** Limits 1 and 3 are shared by several starts of one tick
(5.1.2; two principals' builds against one world bound), so, as storage does for
`k_room` (`spec-storage.md` 5.3.1, FR-STO-011), the start at P2.7 **evaluates them again**
against the running count of the tick: the executions already started in the tick for
the same (principal, class) for limit 1, and the facilities already ordered in the tick
by builds that started earlier in the rotation for limit 3. The second evaluation can
only lower `k`; a start whose `k` falls to 0 is refused with the reason of its limit,
in the rotation order of the starts (`spec-world.md` 6.1.4) and with its inputs kept
(nothing was taken). So the excess of a tick is decided at its start, in the rotation,
and not at the completion. What the start cannot see is a build **started in an earlier
tick and not yet completed**: two builds started a tick apart can together pass a bound
that each was under, and the completion fits what it can (4.3.4).

4.2.3. **A cohort.** `k` executions of a `Builds` recipe will create `k` facilities,
each a separate record with its own `FacilityID` (3.2.1). A cohort of 10 households
that builds with `k` = 4 makes four, not ten and not one. A `Repairs` of `k`
executions restores `k * Units`, spread by 4.3.2.

### 4.3 The completion

4.3.1. **Build.** At the completion of the job (P2.1, or P2.7 for a job of `Duration`
1, which completes inside its own start, `spec-world.md` 6.4; `spec-production.md` 5.3) the
world creates `k` facilities, in ascending order of the new IDs, each with
`Class` the class of the clause, `Region` the **principal's region at the completion**
(a relocation abandons the job, so it is also the region of the start,
`spec-production.md` 5.4), `To` the clause's `Toward` resolved (`NoRegion` if empty),
`Owner` the principal, `Condition` the class's `Durability` and `Built` the tick. A
store class sets `Installed` false (5.4). The recipe's own outputs, if any, are created
by the ordinary transform. `RecipeExecuted` is emitted as for any job, with no output
rows for the clause, and then one `FacilityChanged` (kind `built`) per facility, in ID
order (14.1). The effects do not begin until the next P8.3b (6.3).

4.3.2. **Repair.** At the completion the world restores `k * Units` of condition to
the facilities **the principal owns** of the class in its region, taken **most worn
first** (least `Condition`), ties to the lower `FacilityID`. Each takes at most the
units missing from its `Durability`; what remains after all are full is lost, and
counted (`Voided`, 4.3.4). Repair acts on the principal's own facilities only: a repair
of a road that the principal does not own is not a thing the engine does (open question
5: stewardship). `FacilityChanged` (kind `repaired`) is emitted once per facility
restored, in ID order, with the condition before and after.

4.3.3. **The effect of a build or repair on the same tick.** None until P8.3b. A
facility built at P2.1 is not in the effect table that a position of the tick reads,
so a mill finished in tick `t` serves its first execution in tick `t + 1`, a road
finished in tick `t` speeds the dispatches of tick `t + 1`, and a repair restores the
record at once and the effect at P8.3b. One tick sees one set of effects (position 10
of section 2), which is why no position reads a facility built in the tick it is
built, and why no feedback runs inside a tick.

4.3.4. **A completion that can do nothing** is not a failure and is not held open
(`spec-production.md` 5.3.2 has the same rule for a quantity above `MaxQuantity`): a
build whose bound (limit 3 of 4.2.2) was reached meanwhile by a build that started in
a later tick and completed first, or by a call (11), creates the facilities that fit
and loses the rest, in the order of the completions of P2.1; a repair whose facilities were all
ruined or filled meanwhile restores nothing. The executions that took effect on
nothing are counted in `FacilityLimited.Voided` (14.1), and the inputs are not
returned: they were consumed at the start, as in any abandoned job
(`spec-production.md` 5.4.1).

### 4.4 Who pays, and what a build costs the builder

4.4.1. **The builder pays, by the inputs and by the time.** The inputs are bought like
any input, in the market, at the builder's own bid (`Purpose` `Input`,
`spec-agents.md` 7.3), and are consumed at the start into the `recipe_input` boundary.
The labour is the job: the builder has no other job for `Duration` ticks. The inputs may
be goods made by other recipes (a carpenter's beams, a quarry's stone), so a facility's
supply chain is **data** in the sense of architecture.md 9.2: nothing is specific to it.
Money is not spent by production (`spec-production.md` 4.3.3); a build has no cash cost
other than the purchases of its inputs, and a loan that funds them is CRD's.

4.4.2. **A build is booked as a cost with no return.** The welfare ratio nets the cash
spent on inputs from income (`spec-welfare-and-wages.md` 2.1) and a facility is not an
output, so a household that builds shows a falling or negative income for the length of
the job and an unchanged upkeep, and a poor ratio. The effective window of the role is
derived from the longest `Duration` of the recipes on its list (`spec-welfare-and-wages.md`
2.3.3), so a role that lists a long build recipe has a longer window and a slower
reaction, which is the existing remedy for a long job (N-032). It is not a complete
one: a household that builds can still look poor to the relief rule of
`spec-estates.md` 2.6 and to the migration rule of `spec-migration.md` 5. It is registered
(CAP-A3, 16) and answered by what the builder is: a role with a buffer of money (the
player; a rich household), not a household at the edge of its upkeep.

4.4.3. **A facility has no cost anchor.** The cost of a recipe is attributed to its
outputs by `CostShare` (`spec-production.md` 9); a build has no output, so there is no
`UnitCost` for the clause and the beliefs of the builder are not touched. The cost of
the capital enters the economy where it acts: the output of a mill is made cheaper in
time or in capacity (5.1), and no price is derived from the cost of the mill.

### 4.5 What the default strategy does

4.5.1. **It builds and repairs nothing for a role with no `Saver` block.** Its choice rule takes a recipe only if its
net value is positive or it is a source (`spec-agents.md` 7.1.4). `value_r` there sums
the **non-discard** outputs, and a build or repair recipe has none (B3: its outputs, if
any, are `Discard`), so `value_r` is 0; it has at least one input (B2), so it is no
source and its net value is `-cost_r`, never positive. FR-CAP-018 is a consequence of
B2 and B3 and needs no rule in the default strategy. For a role **with** a `Saver` block the
project rule of `spec-saving-and-investment.md` 5 requests a build or a repair (7.1.10 of
`spec-agents.md`: its request, not the choice rule of 7.1.4, which never chooses one).
This is **intended** and not a gap to be filled by a rule in the default: whether to
invest is a decision with a horizon, a discount and a return that the default has no
data for (the margin test is SAV's, `spec-saving-and-investment.md`), and a rule that
built whenever it could would be a hidden feedback (capital accumulation, CAP-A1)
that no register entry could damp.

4.5.2. **What builds, then.** Four things:

1. the **player** or another external agent with a `Directed` strategy: the directive's
   `Job` names the build recipe (`spec-integration.md` 4.3; the recipe must be on the
   role's list, `ErrNotOnRole`), and the player pays;
2. a **consumer's strategy**, which returns the build recipe from `Work` when its own
   rule says so, from the view of 9;
3. the pack's **initial facilities** (3.4.2) and the calls of 11, for the capital a
   world starts with or a scenario introduces;
4. the **default strategy of a role with a `Saver` block** (`spec-saving-and-investment.md`
   5): a project is weighed, a build or repair recipe requested, and a repair is its rule
   `RepairBelowBP` (5.8 there).

4.5.3. **What the default does with a `Needs` recipe.** It treats it like any recipe:
the view reports `MaxExecutions` as the world derived it (9.1), and a recipe whose
`MaxExecutions` is 0 for want of a facility is not eligible. A household whose role
lists only recipes that need a class it has none of does nothing and the failure ladder
sees it (`spec-spawner.md` 5): the pack gave the role a recipe that it cannot run, which
`unrunnable_recipe` and `facility_unbuildable` (13.2) warn of.

## 5. The effects

Four kinds, each through the seam its owner reserved. In every kind the effect is
computed from the **effect table**, a derived structure that the world rebuilds at P8.3b
and at each call between ticks (6.3, 11), so that every position of a tick reads the
same values.

### 5.1 A site

5.1.1. **What it is.** A recipe whose `Needs` names a site class can start only where
the principal holds a facility of the class in its own region. The facility is not an
input (it is not consumed) and not a tool (a tool is a good in the account and travels
with its holder, `spec-production.md` 4.4): it **stands in a region, has a capacity
and wears by use and by time**. `spec-production.md` 6.7 asked for exactly this.

5.1.2. **The capacity table.** For each (owner, region, class) the effect table holds
`cap_c = sum of cap(f)` over the facilities of the owner of the class in the region
(3.3.2). A job that needs the class starts at most `cap_c` executions: the number of
households of a cohort that can work on one job is limited by what the mill serves, as
`spec-scale.md` 4.2 describes for any partial start. A weight-1 agent with a capacity
of 6 is never limited by it. **The capacity is a bound per principal and tick**: the
sum of the executions that all the starts of one principal in one tick take from the
class is at most `cap_c`. A principal may start its own job and, through hires, the
jobs of workers (`spec-employment.md` 4.1.2, 6.1: the principal is the employer), and
the facility is the principal's in all of them (4.2.1), so the bound is shared by them
as the stock of a tool is (`committed` in `spec-employment.md` 5.5). The world keeps,
for the tick only, a running count `used(principal, class)`, cleared with the other
per-tick counters at P8.6 and never saved; a start at P2.7 has `k <= cap_c - used`
(4.2.2, limit 1) and adds its `k` to it; a hire at formation (P2.5a, `spec-employment.md`
5.1.1 and 5.5, with `committed_c` counting the own request's `k0` and the hires formed
before it, none of them started yet) is refused `no_facility` when `committed_c + m` is
above `cap_c` (an owed row of 19, edit 27). The same tick keeps, for each facility, the
executions already assigned to it (`served(f)`, 5.1.3), cleared at P8.6 and never saved.
It does not occupy the facility for the `Duration` of the job (5.1.4), and the next
tick's starts have the whole capacity again.

5.1.3. **Use wear.** At the start (P2.7), after `k` is final and before the inputs are
taken, each needed class is worn for the `k` executions: the executions are assigned to
the owner's facilities of the class in the region **most worn first** (least `Condition`
in the effect table, ties to the lower `FacilityID`, so every start of the tick sees one
order), each taking at most **what is left of its `cap(f)`**, that is `cap(f)` of the
table less `served(f)`, the executions that earlier starts of the same tick (of the
principal's own job and of its hires) already put on it, and each facility loses
`n_f * WearPerUse`, to a floor of 0, and adds `n_f` to `served(f)`. No facility serves
more than its `cap(f)` in a tick, and the wear does not depend on how the starts are
grouped. This is the rule of `spec-ledger.md` 5.4
for tools ("the most worn item first") applied to a facility. A facility that reaches 0
is ruined at P8.3b and not at once, so it serves the whole tick it wore out in (6.2).
Wear is at the start and is not repeated for the job's duration, as a tool's is not.

5.1.4. **What is simplified.** The facility is not held for the length of a job:
capacity bounds only the `k` of a start, and a long job does not occupy a furnace for
its `Duration`. Since the owner is one agent with one job at a time, the throughput of a
site is at most `Capacity` executions per `Duration` ticks, and a pack sizes `Capacity`
with that in mind (a mill that serves 6 executions of a 1-tick grind serves 6 a tick; one
that serves 6 executions of a 3-tick job serves 2 a tick). A model in which a job holds a
unit of capacity for its `Duration`, so that several agents queue for one site, is open
question 3 (a concurrency limit, the case that `spec-resources.md` open question 1 also
leaves).

5.1.5. **No sharing, no queue.** Only the owner starts a job that needs its facility.
Another household that wants the mill's service buys the flour: the miller is a role
that buys grain and sells flour, and the trade is the ordinary one
(`spec-clearing.md`). A facility used by many is therefore not an exclusion problem and
has no ration, no key and no draw. Open use, a fee and a lease are open question 2. A right over
the use of a facility that works without this module (a right over a recipe that requires one) is
`spec-institutions.md` 5; a lease, a store or a charter over carriage is reserved there (5.8).

### 5.2 A route

5.2.1. **What it is.** A route facility speeds the travel of **goods and travellers by
the modes it names** over an edge, by raising the transit factor `F` of
`spec-transport.md` 4.1. The seam is the one transport reserved: "a faster road raises
it" (`spec-transport.md` 10.2). It acts on the edges it affects:

- with `To` set, on the directed edge from `Region` to `To` **and** the directed edge
  from `To` to `Region` (a road is two-way; if the configuration has only one of the
  two edges, only that one is speeded, and a distance may differ between them);
- with `To` empty, on every edge that **leaves** `Region` (a port, a station: it speeds
  departures).

5.2.2. **The composition.** For a directed edge `(a, b)` that is the shortest path of its
pair (5.2.5) and a mode `m`, let `S` be the route facilities whose class lists `m` and
that affect `(a, b)` (5.2.1). Then, in
integers with 128-bit intermediates (architecture.md 3.5):

```
B(a, b, m)     = min( MaxTotalBoostBP, sum over f in S of eff(f) )
F_eff(a, b, m) = 0                                                   if F_slot(a, b, m) = 0
               = min( MaxTransitFactorBP, floor( F_slot * (10 000 + B) / 10 000 ) )   otherwise
```

`F_slot` is the value of the slot of `spec-transport.md` 10.3 (the consumer's shock,
default 10 000, 0 closes the pair to new dispatches). The facility never writes the
slot: `Slots()` returns what the consumer set, and `Travel` returns the travel time
that uses `F_eff`. Gains **add** and are capped, so ten roads do not multiply (position
9 of section 2). The travel time is `k = max(1, ceil(D * 10 000 / (v * F_eff)))`
(`spec-transport.md` 4.1), so a gain of 5 000 on a factor of 10 000 turns 3 ticks of
60 over a speed of 20 into 2 (18.2).

5.2.3. **What it does not do.** It acts at **dispatch** and on **new** shipments only:
a shipment in flight keeps its arrival tick (`spec-transport.md` 4.3). It does not
change the distance of the edge, so the lag of a rumour (`spec-information.md` 5.2, a
function of the distance and `InfoSpeed`), a migrant's journey (`spec-migration.md`
6.4, its own speed `MoveSpeed`) and the shortest path of REG are unchanged: **a road
speeds goods, not news, and not a household's move**, in version 1. A consumer that
wants the edge itself shortened sets an edge status factor (`spec-regions.md` 7), which
is a shock and not capital. It does not carry a toll, a limit or a queue (open question
4).

5.2.4. **A pair with no mode.** A route class lists the modes it speeds (`Modes`, at
least one). A mode that no shipment uses on the edge leaves the facility without
effect, and a facility built on an edge with no configured mode warns
(`route_unused_mode`, 13.2).

5.2.5. **Which pair a layer applies to.** Transport keys its factor by the directed
**pair** (`from`, `to`, mode) and its distance `D` is REG's **shortest** distance of
the pair, which need not be an edge and, where it is one, need not be the edge's own
distance (`spec-transport.md` 4.1; `spec-regions.md` 4.4, `edge_not_shortest`). The
rule of version 1 is the smallest: **the facility layer `B` of a pair is that of its own
direct edge, and only where that edge is the shortest path of the pair**, that is, the
configuration has the directed edge (`from`, `to`) and its distance equals the pair's
`D`. A pair with no direct edge (a shipment from `a` to `c` over `a`-`b`-`c`), or whose
direct edge is longer than another path, has `B` 0 and the factor is `F_slot`. A road
on `a`-`b` therefore never speeds a multi-hop shipment, and a road on a direct edge
that is not the shortest speeds nothing. It is a **limit** and not a claim that roads
work that way: a path rule (the gain of a path composed per edge, with REG's `Path`)
is open question 12, and the cost of the simple rule is that a road network is as
good as its direct edges. 18.2 has the rows.

### 5.3 A ground improvement

5.3.1. **What it is.** A ground facility raises the **capacity** of the endowment of its
region and resource: an irrigation work, a drain, a terrace. The seam is the capacity
scale of `spec-resources.md` 6.3.1 and note N-108. It acts on one endowment, (`Region`,
the class's `Resource`).

5.3.2. **The composition.** With `B` the sum of `eff(f)` over the ground facilities
whose `Region` is the endowment's and whose class names its resource, capped at
`MaxTotalBoostBP`, the **effective capacity** of the endowment is

```
cap_eff = min( MaxEndowment, floor( Capacity * scale * (10 000 + B) / 10^8 ) )
```

with `Capacity` the configured capacity, `scale` the capacity scale of the slot (default
10 000) and one truncation, in 128 bits, bounded by `MaxEndowment` as `spec-resources.md`
3.4 bounds it (the same formula with `B` = 0; the gain can lift the quotient above the
bound that the product respects, 13.3). A facility never writes the slot: the scale a consumer set (a drought)
and the gain of the drain multiply. The product is below 2^127 with every bound (13.3).

5.3.3. **What follows.** The stock does not rise with the capacity; it regenerates by
the shape of the resource (`spec-resources.md` 4) up to the new capacity, so a drain
raises what the field can hold and not what it holds now. When the capacity falls (a
facility decays or is ruined, a consumer lowers a condition) and the stock is above it,
the stock is **clamped** and the excess is lost, exactly as RES 6.3.1 says of a scale:
it is `Lost` in the identity of `spec-resources.md` 6.4 and is reported by
`EndowmentAdjusted` with the new kind `facility` (14.1). The clamp of P8.3b comes after
`EndowmentChanged` of P8.3, so that event's `Stock`, `Capacity` and `Lost` are those of the
end of P8.3 and are not updated: the loss of the clamp is in `EndowmentAdjusted` only, and
a reader of the tick takes the later of the two for the stock and the capacity (an owed
edit to `spec-resources.md` 14, 19). A capacity of 0 in the
configuration stays 0 whatever the gain, so a build on it is refused (`no_target`, 4.2.2).

5.3.4. **Tenure.** A ground facility raises the stock, and the entitlement of every
title is a share of the stock (`spec-land.md` 3.1.3, 7.7), so the **gain accrues to the
holders of titles in proportion to their shares**, and in a commons to whoever draws.
The builder pays the cost and the holders and the drawers get the gain: this is the
free rider of CAP-A2, the same for a route. A pack in which the builder is the holder
internalises it; a pack in which it is not has the loop. LND is unchanged.

### 5.4 A store

5.4.1. **What it is.** A store facility is the **build** of a store of storage's class:
`Store` names a `StoreClass` (`spec-storage.md` 3.2). The seam is `addStore`, the
internal operation that storage left to infrastructure (`spec-storage.md` 3.6.3). At the
start of a build the dry run of `addStore` is made for the principal (limit 5 of 4.2.2:
the module is on, the role has a base room, the principal is not away or on a journey, the
class's `Needs` flag is carried, the agent is under `MaxStoresPerAgent`) and a refusal is
`store_refused`; the settlement flag of the facility class (8) and the one of the store
class are two checks and both must pass.

5.4.2. **Install and retry.** At the completion the facility exists with `Installed`
false. At the next P8.3b, in ascending ID, the world calls `addStore` for each
facility of a store kind that is not installed; on success `Installed` becomes true,
storage emits its `StoreChanged` (`Kind` `added`, `Cause` `facility`, a new cause of
`spec-storage.md` 14.1) and then `FacilityChanged` (kind `installed`) is emitted, the
store's event first. A refusal leaves it pending, counted in `FacilityLimited.Pending`,
and is retried at every P8.3b. The refusals that can arise between the start and the
install are the ones of the operation (`spec-storage.md` 3.6.2, 3.6.3): the principal
became away (`ErrAgentAway`: a store completed for an agent on a trip waits for its
arrival instead of stranding it), the role changed to one with no room, a limit was
reached (`ErrOutOfBounds`), or **the store class's own `Needs` flag fell**
(`ErrStoreLocked`, `spec-storage.md` 3.2.3). The last is storage's gate on the
creation of its store, and the owner's wording wins: the facility stays **pending**
(not ruined, and counted in `Pending`) until the flag returns or the facility decays
to 0. The class's own gate (8) was read at the start and is not read again, so 8.2
holds for it; the store class's flag is the one second read, and 8.2 says so. The
install is attempted only while the owner's account is in the facility's region (a
store is sited in the region of the account, `spec-storage.md` 3.2, 9.1), so a
facility whose owner has moved waits too, and is counted as pending. **The room does
not exist until installed**: a pending store gives nothing.

5.4.3. **Condition, ruin, and the one-to-one rule.** A store's room is all or nothing:
its condition does not scale the room, because a room that shrinks with wear would be a
cut of the room every tick (`spec-storage.md` 5.5.3). The condition falls by `Decay` and
a repair restores it as for any kind; at 0 the facility is ruined, its store is removed
by the **forced** form of `removeStore`, which does not refuse for room but cuts the room
by the rule of `spec-storage.md` 5.5.2.1 (whole lots to the estate under `estate_goods`,
until the load fits) and by 5.5.3 for the pledge (an owed edit, 19). Storage emits
`StoreChanged` (`Kind` `removed`, `Cause` `facility`) for it, and then `FacilityChanged`
(kind `ruined`) carries `GoodsListed`, the count of lots moved to the estate (14.1).

The **invariant** is that **each installed store facility is one store of its class
that the owner holds, and a store facility stands in the region where its owner's
account is**. Two rules keep it:

- storage's `RemoveStore(a, class)` is refused with a new `ErrStoreInUse`
  (`spec-storage.md` 3.6.2, an owed edit) while the agent's stores of the class would
  fall below its **installed store facilities** of that class, so a consumer ends such a
  store through `AdjustFacilities` (11.2), which ruins the facility and removes its
  store at once; a store the consumer added with `AddStore` is removed by `RemoveStore`
  as before;
- `AddFacilities` for a store class needs the owner's account in `Region`
  (`ErrFacilityRegion`, 11.1), and a build's facility takes the principal's region at the
  completion (4.3.1), which is where the account is.

A role change to one with no base room ends every store of the agent
(`spec-storage.md` 3.6.1a) and so ruins its store facilities, in ascending facility ID,
with `FacilityChanged` (kind `ruined`, cause `role_changed`). **The order** within the
change is `RecipeAbandoned`, `RoleChanged`, then for each ended store storage's
`StoreChanged` (`removed`, `role_changed`), and then, after the stores of the change,
the `FacilityChanged` (`ruined`, `role_changed`) of the facilities in ascending ID
(`spec-world.md` 6.11 item 8 gains the step, 19).

5.4.4. **Not transferable.** A store facility cannot be transferred (11.3,
`ErrFacilityNotTransferable`): its store is the owner's account's room and a transfer
would be a move of room between accounts, which storage's one-owner rule refuses
(`spec-storage.md` 3.5).

### 5.5 What no effect does

An effect changes a capacity, a speed, a capacity of the ground or a room. It does not
set a price, change a belief, change a yield or move goods or money: the ledger sees
nothing of a facility, and a measure of it is a snapshot (14.5). The things RSK, FIS and
ORG will want to read are the facilities by (region, class) and their conditions
(14.5), and the call of 11.2 that lowers a condition (a consumer's call; RSK lowers none in
version 1, 21 item 7). A facility is not an agent and has
no strategy, no account, no weight and no draw.

## 6. Time: decay, ruin and the facility step

### 6.1 Decay

6.1.1. Each facility loses `Decay` units of condition each tick, at P8.3b (6.3), to a
floor of 0, whether it was used or not. `Decay` is an integer of the class, from 0 to
`Durability - 1` (so a new facility is never ruined by its first tick). The default is
0: a class with no `Decay` is permanent unless a call lowers it or a use wears it. There
is no carried remainder: the unit of condition is the author's (3.3.3), and a class
that wants a decay of a third of a unit a tick chooses a unit three times as fine.

6.1.2. **Decay is not a function of the economy.** It reads no price, count or
aggregate (`spec-world.md` 9.4.1) and no season: a pack that wants a weather loss in a
season has no rate to set it by (open question 6). It is the rule that the decay of a
facility is a fixed property of its class, so a facility's age is its condition.

### 6.2 Ruin

6.2.1. A facility whose condition is 0 after the step of 6.3 is **ruined**: the record
is removed, its effect ends with the table rebuild of the same step, and
`FacilityChanged` (kind `ruined`) is emitted with the cause

| Cause | When |
|-------|------|
| `decay` | the condition was above 0 before the decay of P8.3b and 0 after it |
| `use` | the condition was already 0 when P8.3b began, worn out by the starts of the tick (5.1.3) |
| `consumer` | an `AdjustFacilities` call took it to 0 (11.2) |
| `role_changed` | a store facility whose owner's role has no base room (5.4.3) |
| `owner_removed` | a store facility whose owner was removed (7.1) |

6.2.2. A ruin is **permanent**: the record's ID is not reused and nothing rebuilds it.
A build creates a new facility with a new ID. What a ruin does to each kind: a site is
no longer in the capacity table (the owner's recipe that needs it is refused
`no_facility` from the next tick); a route or ground improvement leaves the effect
table, and a ground clamp may follow (5.3.3); a store is removed by the forced form of
`removeStore` (5.4.3).

6.2.3. **A ruin is not a loss of goods.** Nothing in the ledger changes: a ruined mill
leaves no lots, no `disposal`, no `loss`. The inputs that built it were consumed at the
start of the build, long before. RSK's `loss` is for goods (`spec-transport.md` 10.1).

### 6.3 The facility step (P8.3b)

6.3.1. **Where.** After the regeneration of every endowment (P8.3), after the rent
step of tenure when that module is on (P8.3a, `spec-land.md` 5.5, 8.1: it was reviewed
first and keeps the name) and before the optional audit (P8.4), so that the audit sees
the tables consistent with the state and the `TickEnded` of the tick follows it
(`spec-world.md` 6.10). The position is stable as `P8.3b` and is added to the tick by an
edit to that table (19). The two steps are independent in state: the rent step reads
the counters of the tick and writes the posted rent, and this step writes conditions,
tables and stocks, so only the name and the order of the events depend on which is
first (the order is the one of item 9 in 14.1). With the module off, or on with no
facility and no recipe clause in use, it does nothing and takes no time beyond one test.

6.3.2. **What runs, in this order.** Facilities are visited in ascending `FacilityID`,
and endowments in ascending (`RegionID`, `ResourceID`):

| # | Step | Writes | Events |
|---|------|--------|--------|
| 1 | **Decay**: each facility's condition falls by its class's `Decay`, to a floor of 0 | conditions | none |
| 2 | **Ruin**: each facility at 0 is removed (6.2), a store's room is removed by the forced `removeStore` (5.4.3), its cause is recorded | the facility list, the stores | per facility in ascending ID: for a store, storage's `StoreChanged` (`removed`, `facility`), then `FacilityChanged` (`ruined`) |
| 3 | **Install**: each store facility with `Installed` false calls `addStore` for its owner (5.4.2); a refusal leaves it pending and is counted | `Installed`, the stores | per facility in ascending ID: storage's `StoreChanged` (`added`, `facility`), then `FacilityChanged` (`installed`) |
| 4 | **Rebuild** the effect tables (6.4) from the facility list, and for each ground endowment whose effective capacity fell below its stock, clamp the stock and count the excess lost (5.3.3) | the tables, endowment stocks | `EndowmentAdjusted` (kind `facility`), ascending (`RegionID`, `ResourceID`), only for a clamp that lost units |
| 5 | **Report**: per (region, class) with a limit counted this tick, the aggregate | none | `FacilityLimited`, ascending (`RegionID`, `FacilityClassID`) |

6.3.3. **A new facility is commissioned here.** A facility built at P2.1 of the tick
is in the list when step 1 runs, so it also decays by one tick at the end of the tick
it was built in, and is in the tables from step 4: its effect begins with the next
tick (4.3.3). A repair restored its record at P2.1 and its effect enters the tables at
step 4. A facility that wore out at P2.7 and had its executions served that tick is
removed at step 2: the tick sees it whole and the next does not.

6.3.4. **No draw, no state of its own.** The step draws nothing, and its tables are
derived: a checkpoint is taken after it (a boundary), so it never has a half-done step
to save (15.2).

### 6.4 The effect table

6.4.1. The **effect table** is the derived structure that every position of a tick
reads. It holds, as dense arrays indexed by typed ID so that a lookup in the tick loop
allocates nothing (`spec-regions.md` 5.3.5):

| Table | Key | Value | Read by |
|-------|-----|-------|---------|
| site capacity | (owner, region, class) with at least one facility | `cap_c`, the sum of `cap(f)` (5.1.2) | P2.5 (`k0`), P2.7 (the wear assignment) |
| route gain | (directed edge, mode) with at least one facility | `B`, the capped sum of `eff(f)` (5.2.2) | the transit factor `F_eff` of transport, P3.2, P3.3 (dispatch) and the `Travel` query |
| ground gain | (region, resource) with at least one facility | `B`, the capped sum of `eff(f)` (5.3.2) | the effective capacity of RES, P2.3 (the snapshot of stocks), P8.3, the `Endowment` query |

6.4.2. **It is rebuilt in full, in one pass over the facilities**, at P8.3b step 4 and
at the end of each call of 11 that changed a facility. A rebuild is `O(F)` for `F`
facilities (11), with no allocation beyond the arrays sized at construction from the
configuration. The table is not saved and a restore rebuilds it (15.2): an effect is a
function of the saved records and the configuration.

6.4.3. **Why a rebuild, not an increment.** An increment (add a facility's effect when
it is built, subtract it when it decays) needs a reverse of every step to be exact and a
test per path. The rebuild is one function with one test, the saved state is the
records, and the cost is one pass per tick over the facilities, which is a count the
configuration bounds (`MaxFacilities`).

### 6.5 Between ticks

A call that changes a facility (11) acts **at the call**: it validates in full, applies,
rebuilds the table at once (6.4.2), clamps a ground stock that lost capacity and emits
its events, so the next step reads a consistent table at its first position, as the edge
status and the capacity scale do (`spec-world.md` 8.2.1). A call takes no draw and no
position.

## 7. Owners over time

### 7.1 Removal of the owner

7.1.1. When an agent is removed (`spec-agents.md` 3.6; P7.1(d) or a call between
ticks), its facilities are handled at **step 2g** of the removal, after the claims (2d,
2e) and the covers (2f) and before the money and the goods move (steps 3 and 4), in
ascending `FacilityID`:

- a **site, route or ground** facility passes to the **estate of its region**
  (`Owner` is the pseudo-agent whose ID equals the facility's `RegionID`) with
  `FacilityChanged` (kind `transferred`, cause `owner_removed`). A route and a ground
  improvement go on working: their effect does not ask who owns them. A site is
  **idle**: the estate has no strategy and runs no recipe, and the facility decays;
- a **store** facility is ruined with cause `owner_removed`: the store is part of the
  account, which is closed (its goods go to the estate at step 4 like any goods; storage's `StoreChanged` comes
  first, `spec-world.md` 6.11 item 8).

The events come **before** `AgentBankrupted` and `EstateOpened` (every event of a removal
precedes `AgentBankrupted`, `spec-world.md` 6.11 item 8, ruling C28), in ascending
`FacilityID`.

7.1.2. **Why the estate, and why idle.** Goods go to the estate and are listed there at
a markdown (`spec-estates.md` 2.7); a facility is not a good and cannot be listed, so
the estate holds it and does nothing. Making an orphaned site usable by anyone in its
region was weighed (the commons of a mill) and refused for version 1: it needs a rule of
who uses a scarce capacity first, a ration and a key, and a fee or none, which is the
open question of use (open question 2). What an estate-held site needs is a successor,
and the consumer's call of 11.3 gives it one (a gift, a succession, a sale at a price
that SAV's later call sets).

7.1.3. **A job in progress is abandoned first** (`spec-production.md` 5.4): a build in
progress creates nothing, and its inputs are lost with the abandoned job.

### 7.2 A role change

A facility stays with its owner. A site whose class no recipe of the new role needs
is idle, a route and a ground improvement go on working, and a store whose owner's new
role has no base room is ended by storage and with it the facility (5.4.3). The role's
`Recipes` list decides what the owner may build or repair (`ErrNotOnRole`,
`spec-agents.md` 6.5.1).

### 7.3 A weight change

None. A facility is one unit of its owner whatever its weight (3.4.1). A weight rise
does not add facilities and a reduction does not remove them. A job's executions are
scaled as `spec-production.md` 5.4 says, so a build job of `k` executions that falls to
`min(k, w')` creates that many.

### 7.4 A relocation of the owner

A facility **stays where it was built**; its owner may leave (`spec-regions.md` 6). A
recipe that needs a class is judged against the principal's region at the start, so an
owner that has moved cannot use its mill from the new region (`no_facility`), and a
repair acts on the facilities the principal owns **in its region** (4.3.2): the owner
away cannot repair, and the facility decays. A facility is not a reason to refuse a
relocation (7.5), and the mill left behind is the cost of leaving.

### 7.5 No `region_bound` refusal (N-090)

7.5.1. The rule of N-090 is that a module that keeps an agent's **account** in a region
while the agent is away or committed must register a refusal, or another caller can
relocate the account and break the module's invariant. This module registers none for
a site, a route or a ground improvement, with the reasons:

1. it keeps no account anywhere and no agent in any place: a facility is a record
   whose region is its own, not the owner's;
2. no invariant of the module depends on the owner's presence: a facility works
   without its owner (a road), is idle without it (a site) or is repaired only by an
   owner that is in the region at the time (4.3.2);
3. a build in progress is a job, and a relocation abandons a job in the ordinary way
   (`spec-production.md` 5.4, `region_changed`): nothing is committed that the refusal
   would protect.

7.5.2. **A store.** The store of an installed store facility is **storage's**: the
owner of an installed store is refused a relocation by `has_store`
(`spec-storage.md` 9). A store facility therefore binds its owner to the region as
any store does; this spec adds nothing to that refusal. A store facility that is not
yet installed (5.4.2) gives no store and no refusal. The rule of N-090 stands for RSK
and ORG.

### 7.6 A restore

A restore rebuilds the tables from the records (15.2) and validates the records
(15.3). The owner of a record is a live agent or the estate of its region; a store
facility's owner owns stores of its store class enough for the installed facilities.

## 8. Gating by settlement class

8.1. **The field.** A class may name a **settlement flag** in its `Needs`
(`spec-settlements.md` 6.1, 6.3 row H8): the region's class must carry the flag for a
facility of this class to be **built** there. It is the form of the hook that SET
reserved for infrastructure ("a facility: a processing site, a terminal", H8, 6.5), a
flag and not a limit.

8.2. **Where it is read.** At the two places where a facility is **created by a
decision**: the **start** of a build job (limit 2 of 4.2.2, refusal `not_unlocked`) and
the call `AddFacilities` (`ErrFacilityLocked`, 11.1). The read is `regionHas(region,
class.Needs)` of `spec-settlements.md` 6.1.1, at positions that precede P7.2a of the
tick, so it reads the class set at the last P7.2a (R1). It is **not** read at the
completion of a job already started, nor by a use, a decay, a repair or a call that
lowers or transfers: a gate bars creation at the decision, and what exists is not
removed (R3, `spec-settlements.md` 6.2). The one second read is the **store class's own flag** at the install of a store facility (5.4.2), which is storage's gate on its own creation and leaves the facility pending, not ruined.

8.3. **The seven rules of a reader** (`spec-settlements.md` 6.2), each answered:

| Rule | Here |
|------|------|
| R1 | the flag is read only through `regionHas`, at the start of a build and in `AddFacilities`, and not kept across ticks |
| R2 | no limit is used: the flag is a yes or no, and no count of facilities is a function of the class |
| R3 | a gate bars the build; no facility is removed when a region's class falls (8.2); a build already started completes |
| R4 | every refusal is counted: `FacilityLimited.Refused` and the `Debug` reason `not_unlocked` (14.1) |
| R5 | a class that names a flag while settlements are off is `ErrModuleDependency` (13.1) |
| R6 | a gate that closes what a household eats: a facility is not upkeep; a household's food comes from recipes that a pack may gate by `Needs`, and a pack that does so owns the hazard (SET-A2 is the pattern); no register entry of this spec |
| R7 | the row H8 of `spec-settlements.md` 6.3 gains the reader (19) |

8.4. **Initial facilities are not gated.** A row of 3.4.2 in a region whose class lacks
the flag is created: configuration describes the world as it starts, and a gate bars
what an agent decides. The informational warning `facility_gate_closed` says so
(13.2).

## 9. What an agent sees and does

9.1. **The view.** The `Work` view of an agent (`spec-agents.md` 6.3) gains one field,
the agent's own facilities **in its region**, in ascending `FacilityID`, at most
`MaxFacilitiesPerAgent`:

```go
// FacilityView is one facility of the agent's, as Work sees it.
type FacilityView struct {
	ID         FacilityID
	Class      FacilityClassID
	To         RegionID // NoRegion unless a route facility with a target
	Condition  int64
	Durability int64    // of its class
	Capacity   int64    // site: cap(f) of 3.3.2 (0 for other kinds)
	Installed  bool     // store only
}
```

The per-recipe `MaxExecutions` that the view already carries (`spec-production.md`
4.2; `spec-agents.md` 6.3) is the `k0` of 4.2.2, so it includes the capacity, the
caps and the repair need: **a strategy never has to recompute a limit that the world
will apply**. The same field is in `Agent(a)` of INT (11.4). The field is built from the records: a facility completed at P2.1 (P2.7 for a job of one tick)
is in it in the same tick's `Work` and `Trade` views with the `Capacity` of its record, while
its effect is in the table only from P8.3b (4.3.3, 10.4). The `Facilities` field is also in
the `Trade` view of an agent whose role has a `Saver` block
(`spec-saving-and-investment.md` 5.2); otherwise the `Trade` view is unchanged: a facility
has no market and no price.

9.2. **What the default strategy does with it.** For a role with no `Saver` block, nothing
(4.5): it reads `MaxExecutions` to know whether a `Needs` recipe is runnable, as for a tool.
For a role with a block, the project rule reads `Facilities`, `Standing` and the conditions
(`spec-saving-and-investment.md` 5).

9.3. **A consumer's strategy** may read the view to decide to build (an owner whose
mill is worn: the repair recipe's `MaxExecutions` above 0 and a condition under its
threshold), to repair, or to leave a facility to decay. It returns the build or repair
recipe in a `JobRequest` as it returns any recipe, and buys the inputs through `Trade`
intents of `Purpose` `Input`. The world checks it as it checks any request
(`spec-agents.md` 6.5), and a request that cannot start is refused with a reason of
4.2.2 and counted.

9.4. **No belief and no target change.** A facility changes no belief centre, no
`Target` and no tradable set: the inputs of a build recipe are in the executor's
tradable set because the recipe is on its role's list (`spec-agents.md` 3.4), and a
build is not an upkeep. Nothing in `spec-beliefs.md` or `spec-demand.md` changes.

## 10. Position in the tick and draws

10.1. **Positions.** Every act of this module is inside a position that exists, except
P8.3b:

| ID | What this module does there | Owner of the position | Draws | Events |
|----|----------------------------|-----------------------|-------|--------|
| P2.1, and P2.7 for a job of duration 1 | the completion of a build creates the facilities, and of a repair restores condition (4.3). A job of `Duration` 1 completes inside P2.7, after the wear and the inputs, as WLD 6.4 says of any one-tick job | PRO 5.2, 5.3 | none | `RecipeExecuted`, then `FacilityChanged` (`built`, `repaired`), in ID order (for a one-tick job in the second sub-order of `spec-world.md` 6.11 item 3, in start order) |
| P2.5 | the facility limits of 4.2.2 are part of the derivation of `k0` and of the refusals; the view carries the facilities (9.1) | AGT 6.2, 7.1; PRO 4.2 | none | none (a refusal is a `Debug` line, counted) |
| P2.7 | the limits 1 and 3 of 4.2.2 are evaluated again against the running count of the tick, and the use wear of a site is applied for the `k` that starts (5.1.3), before the inputs are taken | PRO 5.2, 5.3 | none | none |
| P7.1(d) | the removal of an owner hands its facilities to the estate (7.1) | AGT 3.6; SPN 10.2 | none | `FacilityChanged` (step 2g), before `AgentBankrupted` |
| **P8.3b** | **the facility step (6.3)**: decay, ruin, install, rebuild the tables, clamp a ground stock, report | **this spec** | none | `FacilityChanged`, `EndowmentAdjusted`, `FacilityLimited` |
| P8.6 | the per-tick counters of 14.1 are cleared with the others | WLD | none | none |

P8.3b is **new**: it follows P8.3 and LND's P8.3a and precedes P8.4, and the edit to
`spec-world.md` 6.10 and 6.11 is in the handoff. The module claims no other position (the rule of
`spec-world.md` 6.1.10: a position states its reads, writes and draws).

10.2. **Reads and writes of P8.3b.** It reads the facility list, the class table and the
endowments' stocks (after P8.3); it writes the conditions, the list, `Installed`, the
tables, the stores (through `addStore` and `removeStore`) and the endowments' stocks
(the clamp), and it reads and writes nothing else. It takes no reservation (none is
open after P4.1, `spec-world.md` 6.1.6).

10.3. **Draws.** This module takes **no** random number, at any position and at any
call, and a module that is on with no facility takes none either (`spec-world.md` 7.4.2).
The table of `spec-world.md` 7.3 gains no row. The count of draws of a tick and the
stream do not depend on whether the module is on (FR-CAP-002). A consumer's own `Work`
may draw as any strategy's does (D3).

10.4. **Order against other modules at the same tick.** A build that completes at P2.1
and a recipe that needs the class at P2.5 of the same tick: the recipe does not see the
facility (4.3.3). A road completed and a shipment dispatched at P3.2 of the same tick:
the dispatch uses the old factor. A ground facility that is ruined at P8.3b clamps the
stock before the audit and the next tick's snapshot (P2.3). A store facility's `addStore`
at P8.3b precedes the next P3.2 in which the room is first used.

## 11. Calls and queries

Three calls act on facilities between ticks, and two queries read them. They are
**calls on the world's state, not slots**: they set no value over a configured default,
so they are rows of `spec-world.md` 8.3 and of `spec-integration.md` section 5 (a new
5.4 there; section 4 is the external agents and 5 is the slots, whose conventions the calls follow), not of the slot tables. The conventions of `spec-integration.md` 3 hold:
the order of checks of 3.2, atomic, one entry per key, an event per change in the
canonical order, no draw (3.3).

### 11.1 AddFacilities

```go
// NewFacility is one facility to create between ticks.
type NewFacility struct {
	Class     FacilityClassID
	Region    RegionID
	Toward    RegionID // route only; NoRegion for every edge leaving Region
	Owner     AgentID  // a household agent, an external agent or the estate of Region
	Condition int64    // 0 means the class's full Durability; else 1 to Durability
}

// AddFacilities creates the facilities in slice order and returns their IDs.
func (w *World) AddFacilities(adds []NewFacility) ([]FacilityID, error)
```

It is the decree that gives a scenario its mill or a first consumer's world a road that
nobody built, and a way to make a test of any condition. The facilities take the next
IDs in slice order (a slice has an order, and an event is in ID order, so a consumer's
order is the creation order). The module must be on (`ErrModuleOff`).

| Error | When |
|-------|------|
| `ErrUnknownFacilityClass` | the class is not in the configuration |
| `ErrUnknownRegion`, `ErrOutsideNode` | `Region` is not a region or is an outside node |
| `ErrSameRegion`, `ErrUnknownEdge`, `ErrFacilityShape` | `Toward` equals `Region`; a route with a `Toward` where no edge joins the pair in either direction; `Toward` set for a kind that is not `route`; a store with an estate owner |
| `ErrOutOfBounds` | a condition above `Durability` or below 0 |
| `ErrUnknownAgent`, `ErrAgentRemoved`, `ErrFacilityOwner` (new) | the owner is not a live agent or the estate of `Region` (`ErrUnknownAgent`; a pseudo-agent that is not the estate of `Region` is `ErrFacilityOwner`) |
| `ErrFacilityRegion` (new) | a store class whose owner's account is not in `Region` (5.4.3: the facility stands where the account is) |
| `ErrFacilityLocked` (new) | the class's `Needs` flag is not carried by the region (8.2) |
| `ErrFacilityTarget` (new) | a ground class on a region with no endowment of its resource with a capacity above 0 |
| `ErrFacilityCap` (new) | the owner would hold more than `MaxFacilitiesPerAgent`, or the world more than `MaxFacilities` |
| a storage error | a store class: `ErrNotLimited`, `ErrAgentAway`, `ErrStoreLocked` or `ErrOutOfBounds` of `addStore` (`spec-storage.md` 3.6.2) |

A store facility created by the call is installed at once, in the call: storage emits
`StoreChanged` (`added`, cause `facility`) and then `FacilityChanged` (kind `added`,
cause `consumer`) is emitted with `Installed` true. A rebuild and a ground clamp follow
(6.5).

### 11.2 AdjustFacilities

```go
// FacilityAdjustment changes the condition of one facility by a signed delta.
type FacilityAdjustment struct {
	ID    FacilityID
	Delta int64 // signed; the result is clamped to [0, Durability]
}

// AdjustFacilities applies the deltas. A result of 0 ruins the facility.
func (w *World) AdjustFacilities(changes []FacilityAdjustment) error
```

A flood that wrecks a bridge, a repair by decree, a scenario at a given condition. A
negative delta is a damage and a positive one a restoration; the result is clamped. A
result of 0 ruins the facility at once (cause `consumer`), with the table rebuild, the
ground clamp and the forced `removeStore` of a store (5.4.3: storage's `StoreChanged` `removed`, cause `facility`, comes first, and `FacilityChanged` `ruined` carries `GoodsListed`). It is also how a consumer ends a store facility, since `RemoveStore` is refused `ErrStoreInUse` for one (5.4.3). Errors: `ErrModuleOff`,
`ErrUnknownFacility` (new), `ErrOutOfBounds` (a delta above the bound
`MaxDurability` in magnitude), `ErrDuplicateID`. An entry whose result equals the
current condition is accepted and is not a change. It is the only call that RSK will
need for a loss of capital (open question 7), and it creates and destroys nothing in
the ledger.

### 11.3 TransferFacilities

```go
// FacilityTransfer gives one facility to another owner.
type FacilityTransfer struct {
	ID FacilityID
	To AgentID // a live household or external agent, or the estate of the facility's Region
}

// TransferFacilities changes the owners. It moves no money: a price is SAV's.
func (w *World) TransferFacilities(moves []FacilityTransfer) error
```

A gift, a succession, an abandonment (to the estate) or a first consumer's scripted
transfer of a mill to a new miller. No money moves, no reason is used and no event but
`FacilityChanged` (kind `transferred`, cause `consumer`). Errors: `ErrModuleOff`,
`ErrUnknownFacility`, `ErrUnknownAgent`, `ErrAgentRemoved`, `ErrFacilityOwner` (the
new owner is a pseudo-agent that is not the estate of the facility's region),
`ErrFacilityNotTransferable` (new: a store facility, 5.4.4), `ErrFacilityCap` (the new
owner at `MaxFacilitiesPerAgent`), `ErrDuplicateID`. The new owner may be in any region:
a site it holds in another region is idle until it is there, which is the user's affair.

### 11.4 Queries

| Query | Returns | Class (`spec-integration.md` 6.1.3) |
|-------|---------|-------------------------------------|
| `Facility(id FacilityID) (FacilityState, error)` | the record, its class, region, target, owner, condition, built tick, installed, and the derived `Effect` (`eff` or `cap`); a search of the sorted list, `O(log F)` | 2 |
| `Facilities(r RegionID) ([]FacilityState, error)` | the facilities that stand in region `r`, ascending ID | 2 |

`FacilityState` is plain data. Both are readers of live state and follow the rules of
`spec-integration.md` 6.1 (a query with an error result returns `ErrInStep` from inside a
step and `ErrWorldFailed` on a failed world; no draw, no event, a deep copy). `Agent(a)`
gains `Facilities`, the agent's facilities in all regions, in ascending ID, at most
`MaxFacilitiesPerAgent` (`spec-integration.md` 4.7.2 gains a group, 19). The effective
travel time, `Travel`, and the effective capacity, `Endowment`, are the existing reads
and already include the facility layer: **a consumer never recomputes a gain**.

## 12. Parameters

### 12.1 The module and its types

12.1.1. The module is present when `Config.Modules.Infrastructure` is non-nil
(`spec-config.md` 3.2: no separate flag):

```go
// Infrastructure is the module of fixed capital.
type Infrastructure struct {
	Classes []FacilityClass   // at most MaxFacilityClasses, keyed by ID, sorted by ID
	Initial []InitialFacility // at most MaxInitialRows, one row per key (3.4.2)
}
```

In a file it is `[modules.infrastructure]`, `[modules.infrastructure.class.<id>]` and
`[[modules.infrastructure.initial]]`, the form of `[modules.merchants]` and
`[modules.storage]` (N-102). `FacilityClass` is the type of 3.1.1 and
`InitialFacility` of 3.4.2. The clauses of a recipe (4.1.1) are fields of `Recipe` and
are valid only with the module on (`ErrModuleDependency`).

12.1.2. **The module has no tunable of its own and no default class.** An empty
`Classes` is legal: the module is then on and inert (FR-CAP-001). Every number is a
field of a class, which a pack authors, and every class field that is optional has the
zero value as its default, which turns the mechanism off (no decay, no use wear).

### 12.2 The fields and the bounds

Every field is a tunable of the pack and has an owner, a bound and a doc line
(architecture.md 9.5, 9.7). The defaults and bounds are placeholders (N-015, N-027).

| Field | Scope | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| `Kind` | class | the kind of effect (3.1) | **required** | `site`, `route`, `ground`, `store` |
| `Durability` | class | condition units of a new facility | **required** | 2 to `MaxDurability` |
| `Decay` | class | units lost a tick, used or not | 0 | 0 to `Durability - 1` |
| `WearPerUse` | class (site) | units lost per execution served | 0 | 0 to `Durability` |
| `Capacity` | class (site) | executions a whole facility serves in a start | **required** for a site | 1 to `MaxFacilityCapacity` |
| `BoostBP` | class (route, ground) | the gain of a whole facility, basis points | **required** | 1 to `MaxBoostBP` |
| `Modes` | class (route) | the modes it speeds | **required** | 1 to `MaxRouteModes`, each a mode of transport |
| `Resource` | class (ground) | the resource whose endowment it raises | **required** | a resource of the resources module |
| `Store` | class (store) | the store class it installs | **required** | a store class of the storage module |
| `Needs` | class | the settlement flag a region's class must carry to build one | "" | a flag of the settlements module |
| `Needs`, `Builds`, `Repairs` | recipe | the clauses of 4.1 | none | 4.1.2 |
| `Region`, `Class`, `Toward`, `Role`, `Count` | initial row | 3.4.2 | **required** except `Toward`, `Role` | `Count` 1 to `MaxInitialCount` |

The bounds are in `bounds.go` (architecture.md 9.6), chosen so that every product of
this spec fits in 128 bits (13.3):

| Bound | Placeholder | Why |
|-------|-------------|-----|
| `MaxFacilityClasses` | 64 | classes are authored by hand |
| `MaxDurability` | 1 000 000 | a unit is the author's (3.3.3); a product with a capacity or a boost fits |
| `MaxFacilityCapacity` | 100 000 | at most the largest weight of a group (`spec-scale.md` 3.2) is useful |
| `MaxBoostBP` | 100 000 | a facility can at most raise a factor elevenfold (`(10 000 + 100 000) / 10 000`); `MaxTransitFactorBP` caps a transit factor at tenfold in any case |
| `MaxTotalBoostBP` | 200 000 | the cap of a sum; `MaxTransitFactorBP` caps a transit factor at tenfold in any case |
| `MaxRouteModes` | 8 | modes are few |
| `MaxRecipeNeeds` | 2 | a recipe needs at most two sites |
| `MaxFacilities` | 100 000 | the world's list, a scan a tick (6.4.2) |
| `MaxFacilitiesPerAgent` | 64 | a household does not own a city; the view carries them (9.1) |
| `MaxInitialCount`, `MaxInitialRows` | 1 000, 1 024 | construction is bounded |

### 12.3 Rule constants

Beside their code, not parameters (architecture.md 9.6): the four kinds; the effect
fraction (3.3.2) with its roundings (`floor` for a gain, `ceil` for a capacity); the
composition by a capped sum (5.2.2, 5.3.2); the **most worn first** assignment of use
wear and of repair, with ties to the lower ID (4.3.2, 5.1.3); the order of the facility
step (6.3.2); the commissioning at P8.3b (4.3.3); the inheritance by the estate
(7.1); the reasons and causes of 14.2; and the position P8.3b.

### 12.4 Rows for the parameter-owner table

For `specs/README.md`, every field a row with its default and a validation test
(architecture.md 9.7, testing.md 6.7). All are owned by this spec (`spec-infrastructure.md`
section) except the clause fields on the recipe, whose **format** is `spec-production.md`
3.1 and whose **values** are here, as for the yield table (`spec-resources.md` 7.1.3):

| Parameter | Scope | Owner |
|-----------|-------|-------|
| `Infrastructure` | Modules (module; nil = off) | `spec-infrastructure.md` section 12.1 |
| `Classes` | Infrastructure | section 12.1 |
| `Initial` | Infrastructure | section 12.1, 3.4.2 |
| `ID` | FacilityClass | section 3.1 |
| `Kind`, `Durability`, `Decay`, `WearPerUse`, `Capacity`, `BoostBP`, `Modes`, `Resource`, `Store`, `Needs` | FacilityClass | section 3.1, 12.2 |
| `Region`, `Class`, `Toward`, `Role`, `Count` | InitialFacility | section 3.4.2 |
| `Needs`, `Builds`, `Repairs` | Recipe | values: section 4.1; format: `spec-production.md` section 3.1 |
| `Class`, `Toward` | FacilityBuild | section 4.1 |
| `Class`, `Units` | FacilityRepair | section 4.1 |

The clause fields and the class fields are **classified in the registry of knobs**
of `spec-calibration.md` 6 in the same change (`spec-stability.md` 9.10): every one is
a **structural choice** or a **damper strength** with no search range (the range is
unknown, `spec-calibration.md` 4.3): `Decay` and `WearPerUse` are damper strengths of CAP-A1 (where decay
brings the accumulation down) and the **driver** of CAP-A5 (where they are not a damper),
`BoostBP` and `Capacity` are structural, and the rest are
structural or presentation. The class of each is in the handoff.

## 13. Validation

### 13.1 Errors

Each has a sentinel and a failing-case test (testing.md 3.9). A path names the field,
for example `modules.infrastructure.class.mill.capacity`. The sentinels are root
sentinels mapped as N-075 asks.

| Sentinel | Rejects |
|----------|---------|
| `ErrFacilityShape` | a class field set that its kind does not use or a required one missing (3.1.2); `Toward` on a clause of a class that is not `route` (B4); both `Builds` and `Repairs` on a recipe (B1); a class in `Needs` that is not a site or that appears twice (B6); an output of a `Builds` or `Repairs` recipe that is not `Discard` (B3); a store row with no `Role` (3.4.2) |
| `ErrOutOfBounds` | a field outside its bound (12.2); more than `MaxRecipeNeeds` needs |
| `ErrDuplicateID` | a class ID twice; two initial rows with one key; a class twice in a recipe's `Needs` |
| `ErrUnknownFacilityClass` | a recipe clause or an initial row that names a class not defined |
| `ErrUnknownMode`, `ErrUnknownResource`, `ErrUnknownStoreClass` | a route, ground or store class that names a mode, resource or store class not defined |
| `ErrUnknownRegion`, `ErrUnknownRole` | an initial row's `Region`, `Toward` or `Role` |
| `ErrSameRegion` | an initial row whose `Toward` equals its `Region` (a clause cannot know its region: a build whose `Toward` is the principal's region is refused `no_target`) |
| `ErrOutsideNode` | an initial row (or a `Toward`) that names an outside node (it has no agents and no market, `spec-regions.md` 3.5) |
| `ErrUnknownEdge` | a route row or clause with a `Toward` where no edge joins the pair in either direction |
| `ErrFacilityNoCost` (new) | a recipe with `Builds` or `Repairs` and no input (B2) |
| `ErrFacilityHolder` (new) | an initial row with a `Role` whose group has no agent in the region, or a store row whose role has no base room (`spec-storage.md` 3.3.1) |
| `ErrFacilityCount` (new) | the initial rows add up to more than `MaxFacilities`, or the round robin would give an agent more than `MaxFacilitiesPerAgent` |
| `ErrModuleDependency` | a route class with the transport module off; a ground class with resources off; a store class with storage off; a `Needs` flag with settlements off (R5); a clause, `Needs`, `Builds` or `Repairs` with this module off |
| `ErrEligibleHasTool` (extended) | a `TraineeEligible` recipe with a clause (B7) |
| `ErrRecipeNoOutput` (waived) | not raised for a recipe with `Builds` or `Repairs` (B3) |
| `ErrOverflow` | a derived product above `int64` or 128 bits (13.3) |

### 13.2 Warnings

A warning never fails validation (`spec-config.md` 7.2). Each has a rule name and a path.

| Warning | Condition |
|---------|-----------|
| `facility_unused` | a `site` class that no recipe needs (informational) |
| `facility_unbuildable` | a class with no initial row and no recipe with `Builds` of it on any role's list: nothing can create it (informational for a class that is never needed) |
| `unrunnable_recipe` (extended) | a recipe `Needs` a class that is `facility_unbuildable` and has no initial facility in a region where the recipe is otherwise runnable |
| `decay_without_repair` | a class with `Decay` or `WearPerUse` above 0 and no recipe with `Repairs` of it on a role's list: it can only be ruined (informational) |
| `capacity_never_binds` | a site class whose `Capacity` is at least the largest weight of a group whose role lists a recipe that needs it: the capacity never cuts a start (informational) |
| `route_unused_mode` | none of a route class's modes is listed by any role's merchant block (`spec-merchants.md` 3.2) and none is a hired mode with a haul commodity: the road is speeded for nobody (informational) |
| `facility_gate_closed` | an initial row in a region whose class lacks the class's `Needs` flag at construction (8.4) |

### 13.3 What is not validated here

- **Overflow.** With the bounds of 12.2: `eff` is `BoostBP * Condition` below 10^5 *
  10^6 = 10^11; `cap` is `Capacity * Condition` below 10^5 * 10^6; the transit factor is
  `F_slot * (10 000 + B)` below `MaxTransitFactorBP * 210 000` = 2.1 * 10^10; the ground
  capacity is `Capacity * scale * (10 000 + B)` below `MaxEndowment` (10^15) * `MaxShockBP`
  (10^5) * 210 000 = 2.1 * 10^25, which fits 128 bits (1.7 * 10^38). The effective capacity is then bounded by
  `MaxEndowment` (5.3.2). The 128-bit helper
  (`spec-currency.md` 7) is used for the last two and for the rest the `int64` checked
  product is enough. `ErrOverflow` is for a bound that a later edit raises.
- **That a build is worth it.** No rule judges the economics of a recipe: a build that
  never pays is a pack's choice, and the viability check does not read it (13.4).
- **Whether a recipe's `Needs` and a role's `Recipes` agree with a region's facilities**
  beyond the two warnings above: a role may list a recipe that needs a class that only
  some regions have.

### 13.4 Viability

The steady state of `spec-viability.md` is a function of recipes, yields, endowments
and the upkeep. A recipe with `Needs` is **runnable in a region** only where the pack
has an initial facility of the class held by that role's agents (3.4.2), and its
activity is bounded by the sum of their `cap` per execution of a start: a row of the
programme for each (role group, class). A route or ground facility at full condition
enters the programme as the gain of its `BoostBP` on the transit factor of the lanes and
on the capacity of the endowment (`spec-resources.md` 11, the sustainable draw). **Decay
and repair are not in the programme in version 1**: a class with `Decay` above 0 and no
repair has no steady state with its facility, and a pack that wants one writes the
repair recipe and its loop into the chain. The solver reports `FacilityDecay`
(informational) for each class with `Decay` or `WearPerUse` above 0. The warm start
creates no job that needs a facility that does not exist (`ErrNotViable` for a `Needs`
recipe with none, as for a missing tool). The edit is in the handoff (19).

## 14. Observability

### 14.1 Events

Two events, one new kind of an existing event, and gated by the module (an event of a
module that is off is not emitted, `spec-observability.md` 4.1). Neither event has a
map or a pointer, and both are built only when an observer was supplied
(observability-and-logging.md 4.5). Section 19 gives the rows for
`spec-observability.md` 4.1 and 4.3 (N-126).

| Event | Position | Class | Fields (besides `Tick`) |
|-------|----------|-------|-------------------------|
| `FacilityChanged` | construction (after the last `AgentSpawned`), P2.1, P2.7 (a job of duration 1), P7.1(d), the role change of `spec-world.md` 6.11 item 8 (`ruined`, `role_changed`), P8.3b, and the calls of 11 | E (per transition) | `Facility FacilityID`, `Class FacilityClassID`, `Region`, `To` (`NoRegion` unless a route with a target), `Owner AgentID` (after), `Kind` (`added`, `built`, `repaired`, `installed`, `transferred`, `ruined`), `Cause` (the causes below), `Before` and `After` (the condition; `Before` is 0 for a creation and `After` is 0 for a ruin), `Effect` (`eff` or `cap` after, 0 for a store), `GoodsListed` (lots moved to the estate by the forced removal of a store at a ruin, `estate_goods`; 0 otherwise) |
| `FacilityLimited` | P8.3b step 5, per region ascending then class; never when `Cut`, `Refused`, `Voided` and `Pending` are all 0 | A (aggregate) | `Region`, `Class`, `Served` (executions started that used a facility of the class), `Cut` (executions not started for capacity though inputs and weight allowed them), `Refused` (starts refused whole, with a reason of 4.2.2), `Voided` (executions of a build or repair that did nothing at the completion, 4.3.4), `Pending` (store facilities waiting for `addStore`) |
| `EndowmentAdjusted` (a new `Kind`) | P8.3b step 4 and the calls of 11 | C at a call; at P8.3b one per clamped endowment at most, bounded by regions times resources (an owed row to `spec-observability.md` 3.3.9) | `Kind` `facility`; the fields of `spec-resources.md` 14: the scales after, stock before and after, units lost |
| `WorldCreated` (the names table gains) | construction | | the `FacilityClassID`s with their strings (`spec-observability.md` 3.2.4) |

The **causes** of `FacilityChanged` are constants of the root package (`event.go`):
`recipe` (a `built` or `repaired` at a completion), `initial` (3.4.3), `consumer` (a
call), `decay`, `use` (6.2.1), `role_changed`, `owner_removed` (7.1, 5.4.3).
`FacilityLimited` is **bounded by the configuration** (regions times classes), not by
agents or executions: it is an aggregate of the kind of `WantShortfall`.
`FacilityChanged` is per facility per transition, like `AgentRelocated`, and is rare:
builds, ruins and calls, never per tick per facility (a decay that changes no
transition is not an event, and `Snapshot` carries the condition).

**The order of the events of P8.3b** (for `spec-world.md` 6.11 item 9), after `BoundaryFlow`,
`EndowmentChanged` and LND's `RentPaid` (P8.3a): per ruined facility in ascending ID,
storage's `StoreChanged` (`removed`, cause `facility`) for a store and then
`FacilityChanged` (`ruined`); then per installed store facility in ascending ID,
`StoreChanged` (`added`, `facility`) and then `FacilityChanged` (`installed`); then
`EndowmentAdjusted` (`facility`) in ascending (`RegionID`, `ResourceID`); then
`FacilityLimited`; then `TickEnded`. The state results of the two steps P8.3a and P8.3b are
independent, so a golden stream differs only by this order, which is written once here.

**What `FacilityLimited` counts and where.** `Cut` and `Refused` are counted at P2.5 and, for a start that the second evaluation of 4.2.2 lowers or refuses, at P2.7,
`Served` at P2.7, `Voided` at P2.1 (a completion), `Pending` at P8.3b step 3; all are
per-tick counters cleared at P8.6 (`spec-world.md` P0.2). They are kept only under the
flag that an observer was supplied (observability-and-logging.md 4.5): a counter that
exists only to build an event is not built without one.

### 14.2 Reasons

Constants of the root package (`event.go`), with their key strings in
`internal/logkeys` only (`spec-ledger.md` 16; N-017):

| Constant (value) | Where it is a `reason` |
|------------------|------------------------|
| `no_facility` | a job refusal (`spec-production.md` 4.2, after `no_tool` and before `no_input`) |
| `not_unlocked`, `facility_cap`, `no_target`, `store_refused`, `nothing_to_repair` | a job refusal for a build or repair (4.2.2, before `no_room`, after `rent_funds`) |

The kinds and causes of 14.1 are closed sets of constants too. No `OrderRejected`
reason and no ledger reason is added.

### 14.3 Logs

Through the injected logger, constant lowercase messages, the keys of
observability-and-logging.md 3.3.2 (section 19: `facility`, an ID, `facility_class`, a
class ID, `facilities` and `facility_classes`, counts in the construction summary, and
`condition`, are the new keys):

| Level | Message | Attributes | When |
|-------|---------|------------|------|
| `Info` | "world created" gains `facility_classes`, `facilities` and the module name `infrastructure` in `modules` | | once, at construction |
| `Warn` | the warnings of 13.2 | `rule`, `path` | once each at construction |
| `Debug` | "facility built", "facility repaired", "facility ruined", "facility installed" (constant messages) | `tick`, `facility`, `facility_class`, `agent`, `condition` | per transition, guarded by `Enabled` |
| `Debug` | "facility start refused" | `tick`, `agent`, `recipe`, `reason` | per refusal, guarded |

No `Info` and no `Warn` is emitted inside the tick: `FacilityLimited` is the counter
of the limits (observability-and-logging.md 3.2.2; N-045, FR-STB-013), so no `Warn` is
needed and none is added (N-126). The `Warn` of the module is only at construction.

### 14.4 Metrics

Derived from the events and the snapshot (observability-and-logging.md 5.3.1), integers,
bounded attributes (5.4). Section 19 gives the rows.

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.facility.count` | UpDownCounter | `{facility}` | `economysim.facility_class`, `economysim.region` | `FacilityChanged` (+1 for `added` and `built`, -1 for `ruined`; `repaired`, `installed` and `transferred` change no count) |
| `economysim.facility.limited` | Counter | `{execution}` | `economysim.facility_class`, `economysim.region`, `economysim.facility_limit` | `FacilityLimited` (`Cut` as `cut`, `Voided` as `voided`) |
| `economysim.facility.refused` | Counter | `{start}` | `economysim.facility_class`, `economysim.region` | `FacilityLimited` (`Refused`: starts refused whole, one unit per start) |
| `economysim.facility.pending` | Gauge | `{facility}` | `economysim.facility_class`, `economysim.region` | the snapshot's facility rows, `Pending` (store facilities not installed) |
| `economysim.facility.condition` | Gauge | `{basis_point}` | `economysim.facility_class`, `economysim.region` | the snapshot's facility rows, `Condition * 10 000 / Durability` summed over the row |

New attribute values are bounded by the configuration: `economysim.facility_class`
(the classes), `economysim.region` (existing) and `economysim.facility_limit` (the two
constants `cut` and `voided`). The gauges are fed by a snapshot and are absent, not zero, until one is passed
(observability-and-logging.md 5.3.1).

### 14.5 Snapshot

OBS owns the names (`spec-observability.md` 5.3); this spec asks for these, all plain
data and all functions of saved state (N-125):

- a `FacilityRow` per (region, class) with a non-zero count: `Facilities` (a count of
  facilities, never of agents or households), `ConditionBP` (the sum of
  `Condition * 10 000 / Durability` over the facilities, so the mean is it over the
  count), `Effect` (the sum of `eff` for a route or ground class, of `cap` for a site, 0
  for a store) and `Pending` (store facilities not installed);
- the header of the snapshot gains nothing: the effect tables are derived, and the
  effective capacity of an endowment is already in its row.

The individual facilities are read by the queries of 11.4, which are the only
per-facility reads; a snapshot with a row per facility would grow with `MaxFacilities`,
and the rows of this spec are bounded by regions times classes.

### 14.6 Cost with no observer

With the default observer and logger, no event is built and no log call allocates (the
flag of observability-and-logging.md 4.5). The module adds, at P2.5, a table read per
needed class and, at P2.7, a wear assignment over the owner's facilities, which is
`O(facilities of the owner in the region)` and at most `MaxFacilitiesPerAgent`. At
P8.3b it adds one pass over the facilities. No allocation per step beyond the arrays
sized at construction (NFR-CAP-002).

## 15. State contributed to a checkpoint

15.1. **Saved: the facilities**, in a section of their own (a new tag in the dependency
order of `spec-checkpoint.md` 4.1; the facility is not a field of one agent, N-128, and
exists apart from its owner, so the agent record is not its home). A header and a list:

| Field | Meaning |
|-------|---------|
| next facility ID | `int64`, the ID the next creation takes: an ID is never reused, and the highest live ID is not enough if the highest was ruined |
| count `n` | the facilities, at most `MaxFacilities` |
| per facility, ascending ID | `ID`, class index (the position in the configuration's sorted classes), `Region`, `To` (`NoRegion` as the integer the codec writes for it), `Owner` (an agent ID), `Condition`, `Built` (tick), `Installed` (a flag, store only) |

All are varints and one spelling (`spec-checkpoint.md` 5.2). The shape depends on
whether the module is on, which the configuration hash fixes (`spec-checkpoint.md`
4.2.1).

15.2. **Derived and rebuilt, never saved** (`spec-checkpoint.md` 4.3): the effect tables
(6.4), `eff` and `cap`, the per-owner counts, the per-tick counters of 14.1 and `used(principal, class)` and `served(f)` (cleared at
P8.6, so a boundary has none), and the order of use wear. Nothing a position half-did is
saved: a checkpoint is taken after P8.3b and before the next P0.

15.3. **Restore validation** (`spec-checkpoint.md` 6.2 gains these, section 19), each a
failure with `ErrCheckpointCorrupt`: the IDs are strictly ascending and below the next
ID; a class index is below the count of classes; `Condition` is from 1 to the class's
`Durability`; `Region` is an ordinary region and `To` is `NoRegion` unless the class is a
`route`, in which case it is `NoRegion` or a region joined to `Region` by an edge in
either direction; `Owner` is a live agent or the estate of `Region` (the estate's ID
equals the `RegionID`); the count is at most `MaxFacilities` and per non-estate owner
at most `MaxFacilitiesPerAgent`; `Installed` is true only for a store class, and for
each (owner, store class) the installed facilities are at most the stores of that class
the owner holds (`spec-storage.md` 15.1; `RemoveStore` keeps it so, 5.4.3), and the owner's
account of an installed facility is in the facility's `Region` (5.4.3); a store
facility's owner is not an estate.

15.4. **Format version.** The checkpoint gains a section with the module on, so the
format version changes with the first release that has the module
(`spec-checkpoint.md` 9.3), and a fixture of it joins `testdata/checkpoints/`.

15.5. **Not state:** the effect tables, the per-tick counters, the events, the
warnings, and the recipes' `Needs`, `Builds` and `Repairs` (configuration, hashed). The
jobs in progress are the agent's (`spec-checkpoint.md` 4.2 tag 6), and a build job's
clause is read from the recipe at completion.

## 16. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. The checklist of
`spec-stability.md` 9 was followed: 9.1 each rule of sections 4 to 7 was read for the
chain it creates; 9.6 the delays were checked against the neighbours and each entry
names the ones that match: a build job's `Duration` and the effective welfare window
`T = max(WelfareWindow, WindowCycles * longest Duration)` (`spec-welfare-and-wages.md`
2.3.3), the settling rule and the review period of migration (`spec-migration.md` 5),
the transit time `k` of a shipment (`spec-transport.md` 4), the regeneration of an
endowment (`spec-resources.md` 4) and the failure ladder's privation count
(`spec-spawner.md` 5).

**CAP-A1 Capital accumulation and the overbuilding cobweb.** *Kind:* loop (a cobweb).
*Loop:* a high price of what a facility makes (+) the builder's margin (+) builds started
(+) capacity, `Duration` and one tick later (+) output (-) the price; and the facility
decays (-) capacity. One negative link, so balancing (B), with a delay that makes a
cycle and a one-way ratchet (a facility is not unbuilt, only worn): builders respond to
a gap that has closed by the time their capacity exists. *Delay:* the build's `Duration`
plus the commissioning tick (4.3.3). *Owner:* CAP 4 for the mechanism; **the decision
is the strategy's, the player's directive or SAV's rule (SAV-A3)**, and the default strategy of
a role with no `Saver` block takes none (4.5.1), so the loop exists only with a consumer's
rule or a role with a `Saver` block. *Damper:* CAP-D2 (the decay brings capital down by
itself), CAP-D3 (the capped sum of gains), the belief rule's stock term that lowers an
asking price as stock rises (L06) and the production halt (D24). *Undamped:* a
strategy that builds on a price signal with a gain and a delay. *Condition:* in the
model of 17 (an invented rule) a builder's gain `g` times the delay `L` is bounded: at
`L` = 10 a gain of 1.0 oscillates by 21.6 facilities and a gain of 0.5 by 0.2, at `L` = 2
none of the five gains does. It is a model of an invented rule, not of any rule in the
library. *Neighbours (9.6):* `Duration` of a build against the effective window `T` (a
match by construction, so the welfare record spans the build, 4.4.2) and the cobweb of
occupations at about two periods (L36). *Test:* CAP-S2. *Status:* model.

**CAP-A2 A free rider on a public effect.** *Kind:* hazard (a public good). *Loop:* the
cost of a road or a drain falls on its builder (-) the builder's return is its own share
of a gain that everyone has (the merchants who ship over it, the holders and drawers of
the ground) (-) builds fall (-) the gain decays (-) the gain, so a public good is
under-provided by private builders. *Delay:* `Durability / Decay`. *Owner:* CAP 5.2,
5.3. *Damper:* none in this spec, by design: **a treasury that builds and a toll belong
to FIS and PLC**, and a pack that wants roads in the meantime internalises the gain
(the builder is the merchant house that ships over the road) or provides them by
`AddFacilities`. *Undamped:* a pack in which the builder is not the user. *Condition:*
the builder's share of the gain times the gain exceeds the cost of the build over its
life. *Test:* CAP-S3. *Status:* hand.

**CAP-A3 A build is booked as a cost with no return, and the ladder acts on it.** *Kind:*
loop (reinforcing). *Loop:* a build job (+) the cash spent on inputs, netted from income
(`spec-welfare-and-wages.md` 2.1) (-) the builder's welfare ratio (-) the relief,
the failure ladder's role change or the migration review (+) the builder's job
abandoned (`role_changed`, `region_changed`) (+) the inputs lost (-) capital formed
(-) the future ratio. *Delay:* `Duration`, and the window `T` of the role. *Owner:* CAP
4.4.2 with WEL and MIG. *Damper:* D29, the derived window (a role that lists a long build
recipe has a window longer than it), the settling rule of migration (`spec-migration.md`
5.2, at least the window), the privation count's leak (`spec-spawner.md` 5.1). *Undamped:*
a household at the edge of its upkeep that builds. *Condition:* the builder holds money
for the inputs and `Duration` ticks of upkeep, and a window above the job (D29).
*Neighbours (9.6):* `Duration` against `T` and against `MoveCooldown`, a match by
design. *Test:* CAP-S4. *Status:* hand.

**CAP-A4 A ground improvement is absorbed by the population.** *Kind:* loop, a case of
the Malthusian loop (`spec-population.md` 5; `spec-resources.md` 8). *Loop:* a drain
raises the capacity (+) the stock the field can hold (+) the draw per household (+) the
welfare ratio (+) births (POP) (+) households (-) the draw per household. Balancing, and
the end state is a larger population at the replacement ratio, **not a higher standard
of living**. *Delay:* the regeneration of the stock and the birth schedule's response.
*Owner:* CAP 5.3 with POP and RES. *Damper:* none to add: the fixed point of POP is the
end state, and `HouseholdCap` bounds it. *Condition:* with the population module on the
long-run ratio is `r*` whatever the capacity; with it off the gain persists. *Neighbours
(9.6):* the regeneration rate against the birth schedule's window (a match to check in
the run). *Test:* CAP-S5. *Status:* hand; a merge into the Malthusian entries (L75 to
L83) with this as a case.

**CAP-A5 Capital decays silently without repair.** *Kind:* loop (reinforcing, a poverty
trap of capital). *Loop:* condition falls by decay and wear (-) the effect (-) output
(-) the owner's income (-) the means to buy the repair's inputs (-) condition. *Delay:*
`Durability / (Decay + wear)`. *Owner:* CAP 3.3, 6.1. *Damper:* **none**. Decay and
wear are the loop's **driver** (they lower the condition), not something that opposes
it; CAP-D1 makes the fall gradual and visible and CAP-D2 states the driver and the
permanent ruin, and neither damps. The only counter is a repair, which is a consumer's
strategy's or SAV's decision (4.5.1): a repair is SAV's rule for a role with a block
(`RepairBelowBP`, `spec-saving-and-investment.md` 5.8) and otherwise the default repairs
nothing, so a pack with `Decay` above 0 and no consumer's strategy and no `Saver` block
has a capital that runs down, and the warning `decay_without_repair` says so. *Undamped:* by design. *Condition:* none is
claimed for the loop. The model of 17 (part 4) is a **threshold policy on one mill**
with no cash, no input and no income, so it does not contain the reinforcing link
through income and is not evidence for the trap; it shows only the executions that an
invented threshold gives against the time a repair costs: a mill never repaired gives
421 executions in a thousand ticks, repaired below 100 gives 3 070, below 300 3 946,
below 600 4 763 and below 900 4 368 (too early: the job time is lost). *Test:* CAP-S2.
*Status:* hand.

**CAP-A6 Capital idles when its owner dies.** *Kind:* hazard. *Loop:* an owner is
removed (+) its site passes to the estate (+) the site idles (-) the role's output
(-) the region's supply of what it made; a ruin follows at the site's decay. *Delay:*
the ladder. *Owner:* CAP 7.1. *Damper:* none in the engine: a consumer's
`TransferFacilities` (11.3) gives it a successor. *Undamped:* by design (open question
2). *Test:* CAP-S6. *Status:* hand.

**CAP-A7 A road changes the delay of a running loop.** *Kind:* hazard. *Loop:* a route
facility completes or decays (+) the transit factor of new dispatches changes (+) the
travel time `k` of a shipment changes by a tick (+) the pipeline holds two vintages
(+) arrivals bunch; and `k` is the delay of the shipper's loop (L84, L85: the stable
gain bound `2 sin(pi / (2 (2k + 1)))` rises as `k` falls). *Owner:* CAP 5.2 with TRN
15 and MER 7. *Damper:* L85's count of `Inbound`, D39's cap on the dispatch, and the
smooth decay (CAP-D1), by which `k` changes by at most a tick at a time. *Undamped:*
none beyond TRN's. *Test:* CAP-S3. *Status:* hand.

**CAP-A8 A bound that binds is first-come.** *Kind:* bias. *Loop:* none: when
`MaxFacilities` or `MaxFacilitiesPerAgent` is reached the builds that start in one tick
are refused `facility_cap` at P2.7 in the order of the starts, which is the rotation of
`spec-world.md` 6.1.4 (4.2.2: the bound is evaluated again against the running count of
the tick), so no ID is systematically first; a completion that no longer fits is voided
in the order of P2.1 (4.3.4). *Owner:* CAP 4.2.2 with
`spec-world.md` 10. *Damper:* the rotation. *Counted:* `FacilityLimited.Refused`.
*Test:* none beyond the unit test of the bound. *Status:* hand.

### 16.1 Dampers this spec adds or specifies

| ID | Damper | Where | Status |
|----|--------|-------|--------|
| CAP-D1 | The effect is proportional to the condition: a worn facility is a weaker one, with no cliff but the last unit; a road's `k` and a mill's capacity step down one unit at a time | 3.3.2 | specified (rule) |
| CAP-D2 | Decay and ruin, **with two roles**: against CAP-A1 (accumulation) it is a damper, because capital that is not repaired runs down by itself and the stock cannot ratchet up; against CAP-A5 it is the **driver**, not a damper, and a ruin is permanent and counted. Cost: capital is lost for want of attention | 6.1, 6.2 | specified (data; `Decay` 0 is the off value) |
| CAP-D3 | The capped sum of gains (`MaxTotalBoostBP`): no number of facilities multiplies an effect past a bound. Cost: the last facility adds nothing | 5.2.2, 5.3.2 | specified (a bound) |
| CAP-D4 | The commissioning at P8.3b: no effect inside the tick that builds it, so no loop of zero delay and one set of effects per tick | 4.3.3, 6.3.3 | specified (rule) |
| CAP-D5 | The world and per-agent bounds on facilities, counted. Cost: a build is refused at the bound | 4.2.2, 12.2 | specified (a bound) |

### 16.2 Scenarios proposed

| ID | Scenario | Measure and pass |
|----|----------|------------------|
| CAP-S1 | The module on with no class and no initial row, over a range of seeds, against the same pack with it off | the state, the snapshot rows and fields other than `ConfigHash`, `Modules` and `Names`, the events (other than `ConfigHash` and `Names` of `WorldCreated`) and the final PCG state are equal; no draw differs (FR-CAP-001, 002). **Harness test** |
| CAP-S2 | One site and a repairing owner (a consumer's strategy) over a long run with `Decay` and `WearPerUse`, and the same without repair | the condition of the repaired mill stays above the threshold, the unrepaired one is ruined and `FacilityChanged` (`ruined`, cause `decay`) or (`use`) is emitted once; the executions with repair are above those without (the model's order) |
| CAP-S3 | A road between two regions of `three` with the trade overlay, built by a merchant role, against the same pack with the road by `AddFacilities` and with none | the price gap band between the two regions narrows with the road; the builder's return is below the gain of the whole (the free rider); `k` falls by the model's amount; no shipment in flight changes its arrival |
| CAP-S4 | A build by a thin household and by a household with a buffer, with relief and migration on | the thin one is flagged by the ratio and may be moved or relieved; the one with a buffer finishes; the abandoned job loses its inputs (counted by `RecipeAbandoned`) |
| CAP-S5 | A drain on the arable endowment of `one` with the population module on and off | with POP off the capacity and the draw per household rise and stay; with POP on the households rise to a larger fixed point and the ratio returns to `r*` |
| CAP-S6 | The removal of a site's owner, then a transfer | the site passes to the estate and idles; a transfer restores its use; no good or money is created |
| CAP-S7 | A checkpoint taken mid-build and with a store facility pending, restored and run | the continuation equals the uninterrupted run (FR-CAP-041). **Harness test** |

## 17. What was run

One script, `models/cap/worked.py` (README beside it), Docker Python 3.12 with
`PYTHONDONTWRITEBYTECODE=1`, standard library only, the output captured by the host
shell in `worked.out`. It is a model of the rules, not of the library, with
placeholders (N-015, N-027).

| Part | What it shows | What it does not show |
|------|---------------|-----------------------|
| 1 | the transit factor and the travel time with route facilities (`BoostBP` 6 000): 3 ticks with none, 2 with a road at 2 000, 2 with one built (1 999 in the table), 3 at half its condition, 2 with two, 4 with a damaged edge (F_slot 5 000), a closed pair stays blocked; `k` is 2 down to condition 1 667 | traffic, a toll, a queue |
| 1b | which pair a layer applies to (5.2.5): a direct shortest edge gets the gain; a pair with no direct edge, or whose edge is not the shortest path, gets none | a path rule (open question 12) |
| 2 | a ground facility as its condition falls: the effective capacity falls from 1 560 to 1 200 over 500 ticks and a stock kept at capacity loses 360 units in all (72 per 100 ticks) | the regeneration of the stock, or any population |
| 3 | the most-worn-first assignment over 5 000 random lists and two starts of one principal in a tick: every execution is placed, no condition is negative, no facility serves above its `cap(f)` over the tick (the remaining `cap(f)` is tracked), and splitting the starts changes neither the assignment nor the wear | that the rule is fair between owners (there is one owner) |
| 3c | two mills of `cap(f)` 3 (`cap_c` 6), an own start of 3 and a hire of 3: each mill serves 3 and loses `3 * 2` (894 and 944 from 900 and 950); the second start does not pile on the most worn mill | the ordering of starts across principals (there is one principal) |
| 3b | the capacity per principal and tick (5.1.2): requests 4, 5 and 3 against a capacity of 6 take 4, 2 and 0 | that EMP's formation check agrees (an owed row) |
| 4 | a mill of a cohort of 10: never repaired it is ruined at tick 158 with 421 executions; repaired below 100, 300, 600 and 900 it gives 3 070, 3 946, 4 763 and 4 368 executions in 1 000 ticks, the best near the middle because a repair job costs time | that any threshold is right (a strategy's choice), or the income link of CAP-A5: the mill has no cash, no input and no income |
| 5 | a builder with an invented rule against decaying capital: 0.1 facility of swing at a gain of 1.0 and a delay of 5, 21.6 at a delay of 10 with that gain, 12.8 at a delay of 5 with a gain of 2.0 and none at a delay of 2 | that any rule of the library has this gain; the default builds nothing for a role with no `Saver` block |

## 18. Worked examples

The settings are invented **examples**; the words in them are data.

### 18.1 Example A: a mill in a village (a pre-industrial setting; unit: one grind)

A site class `mill`: `Durability` 1 000, `Decay` 1, `WearPerUse` 2, `Capacity` 6. A role
`miller` of one cohort of weight 10 lists the recipe `grind` (`Needs = ["mill"]`,
one input `grain`, one output `flour`, `Duration` 1) and the recipe `repair_mill`
(`Repairs {Class: "mill", Units: 100}`, one input `timber` times 2, `Duration` 3).

- **Tick 0**, one mill at 1 000 held by the cohort. `cap = ceil(6 * 1 000 / 1 000)` = 6;
  `k0 = min(10, 6)` = 6 (a partial start: six households grind, four cannot, as
  `spec-scale.md` 4.2). At P2.7 the six executions are assigned to the one mill, which
  loses `6 * 2` = 12, to 988; at P8.3b it decays by 1, to 987.
- **Tick 1**: `cap = ceil(6 * 987 / 1 000)` = `ceil(5.922)` = 6; the mill falls to 975,
  then 974.
- **Tick 13**: the condition at its start is 831, under 833, and `cap` becomes 5: the cohort's output
  falls by a sixth, with no cliff (CAP-D1). A mill never repaired is ruined at tick 158,
  after 421 executions in all (17, part 4). The miller's recipe is then refused
  `no_facility` and the cohort does nothing until a call or a build gives it another.
- **Repaired below 600**, the cohort starts `repair_mill` when the condition passes
  under 600: `k0 = min(10, w, ceil(missing / 100))`, so 5 executions for a missing of
  about 410, which restore 500 (capped at the `Durability`) at the completion
  in the tick `s + 2` of a start in tick `s` (a job of `Duration` 3 occupies `s` to
  `s + 2`, `spec-production.md` 5.2), at a cost of 10 `timber` (2 each). It gives 4 763 executions in 1 000 ticks
  (17, part 4; the model's policy is one invented threshold).

### 18.2 Example B: a road between two regions (a pre-industrial setting; unit: one league)

A route class `road`: `Durability` 2 000, `Decay` 1, `BoostBP` 6 000, `Modes = ["cart"]`.
The edge `alpha` to `beta` is 60 long and the mode `cart` has a speed of 20 at a
factor of 10 000, so `k = ceil(60 * 10 000 / (20 * 10 000))` = 3 (`models/cap/worked.out`
part 1).

| State | `B` | `F_eff` | `k` |
|-------|-----|---------|-----|
| no road | 0 | 10 000 | 3 |
| one road at 2 000 (made by `AddFacilities`) | 6 000 | 16 000 | 2 |
| one road **built** by a job (it has decayed once at P8.3b of its tick: 1 999 in the table) | 5 997 | 15 997 | 2 |
| the road at half its condition (1 000) | 3 000 | 13 000 | 3 |
| two roads at 2 000 | 12 000 | 22 000 | 2 (1.36 rounds up) |
| one road at 2 000 and a consumer's shock of 5 000 on the pair (a damaged edge) | 6 000 | 8 000 | 4 |
| one road at 2 000 and a shock of 0 (closed) | 6 000 | 0 | blocked |

The road is built by a role that lists `build_road` (`Builds {Class: "road", Toward:
"beta"}`, inputs `stone` and `timber`) and stands in `alpha`. It speeds both
`alpha` to `beta` and `beta` to `alpha`. A built road has `k` 2 until its condition falls
to 1 667, which is 332 ticks after its first table, and `k` 3 after that, so a road
that was exactly at full condition is not the case of a built one (it decays at the
P8.3b of the tick it is built, 6.3.3). A repair recipe, if the owner has one, restores it.
Shipments in flight when the road completes keep their arrival tick (5.2.3). The lag of
a rumour from `beta` to `alpha`, a function of the distance, is unchanged (5.2.3).

**Which pairs the road acts on** (5.2.5; part 1b of the model), the same road on every
direct edge, speed 20:

| Pair | Direct edge | Shortest `D` | Layer applies | `B` | `k` |
|------|-------------|--------------|---------------|-----|-----|
| `alpha` to `beta` | 60 | 60 | yes (its own edge is the shortest path) | 6 000 | 2 |
| `alpha` to `gamma`, over `alpha`-`beta`-`gamma` | none | 120 | no (no direct edge) | 0 | 6 |
| `alpha` to `delta`, a direct edge of 110 and a path of 100 through `beta` | 110 | 100 | no (the edge is not the shortest path) | 0 | 5 |

The second and third rows are the **limit** of version 1 (open question 12): a road on
`alpha`-`beta` does not speed a shipment to `gamma`, and a direct road of 110 where a
path of 100 exists speeds nothing.

### 18.3 Example C: a drain on a field (an agricultural setting; unit: one measure)

A ground class `drain`: `Durability` 500, `Decay` 1, `BoostBP` 3 000, `Resource =
"arable"`. The endowment of `alpha` has a configured capacity of 1 200 and the consumer
has not shocked it (scale 10 000).

- A drain made by `AddFacilities` at its full 500: `B = 3 000`, `cap_eff = floor(1 200 *
  10 000 * 13 000 / 10^8)` = 1 560. A drain **built** by a job has decayed once at its
  first P8.3b (6.3.3): condition 499, `B = 2 994`, `cap_eff` = 1 559.
- After `n` steps of P8.3b the condition is `500 - n`. At condition 400 (`n` = 100):
  `B = 2 400`, `cap_eff` = 1 488; at 300, 1 416; at 200, 1 344; at 100, 1 272; at 1,
  `eff` is 6 bp and the capacity has rounded to 1 200 (`floor(1 200.72)`).
- A stock that sat at the capacity loses the difference each time the capacity falls:
  `EndowmentAdjusted` (kind `facility`, units lost) is emitted at P8.3b of each tick
  in which a unit was lost (RES counts it as `Lost` in the identity of 6.4); the
  total over the life of a drain made by `AddFacilities` is 360 (359 for a built one,
  which never held 1 560). At the tick the drain is ruined it is removed and the
  capacity is the configured 1 200 again.
- A consumer's drought, `SetCapacityScales` 6 000, multiplies: with the new drain
  `floor(1 200 * 6 000 * 13 000 / 10^8)` = 936, with no drain 720.

### 18.4 Example D: a barn (a pre-industrial setting; unit: one measure)

A store class `barn` of storage (`spec-storage.md` 18.1) and a facility class
`barn_works`: kind `store`, `Store = "barn"`, `Durability` 2 000, `Decay` 1. The
role `steward` has a base room and lists `build_barn` (`Builds {Class: "barn_works"}`,
inputs `timber` and `stone`, `Duration` 30).

- The build starts at tick 10. A job of `Duration` 30 started in tick 10 occupies ticks
  10 to 39 and completes at P2.1 of tick 39 (`spec-production.md` 5.2). At the
  completion the facility exists with `Installed` false (`Built` 39) and the steward has
  no new room.
- At P8.3b of tick 39 the facility decays once (to 1 999) and the world calls
  `addStore`: the steward is at home, so storage's `StoreChanged` (`added`, cause
  `facility`) and then `FacilityChanged` (`installed`) are emitted; the room exists from
  tick 40.
- If the steward had been **away on a trip** at tick 39, `addStore` is refused
  (`ErrAgentAway`), the facility stays pending and `FacilityLimited.Pending` is 1 at
  every tick until the trip ends and the retry succeeds (5.4.2). The traveller is not
  stranded by a region-bound refusal that a store would otherwise cause at its arrival.
- The condition is `2 000 - (n + 1)` after the step of tick `39 + n`, so it reaches 0
  at P8.3b of tick 2 038: the facility is ruined, storage's `StoreChanged` (`removed`,
  cause `facility`) comes first, the load above the base room moves to the estate by
  `estate_goods` (`spec-storage.md` 5.5.2.1), and `FacilityChanged` (`ruined`, cause
  `decay`, `GoodsListed` the count of lots moved) is emitted.

## 19. Notes settled, and what this spec needs from the other specs

### 19.1 Notes

| Note | Status here | Where |
|------|-------------|-------|
| N-108 (LND, CAP, EXT) | **Applied for CAP**: a facility that raises an endowment's capacity is a ground facility; it is a layer multiplied into the effective capacity and never writes the capacity-scale slot of `spec-resources.md` 6.3; the clamp is `Lost` in the identity. LND and EXT stay (LND is answered, `spec-land.md` 7.7; EXT's part is the import predicate) | 5.3 |
| N-046 (EMP, SAV, CRD, STO, RSK, FIS, PLC, LND) | CAP is not in the note's list of owed specs; its entries are CAP-A1 to CAP-A8, the dampers CAP-D1 to D5 and the scenarios CAP-S1 to S7 in section 16, owed to the register | 16 |
| N-090 | **Applied**: a site, a route and a ground improvement register no `region_bound` refusal, with the reasons; a store facility is bound by STO's `has_store` | 7.5 |
| N-104 | **Applied**: the initial rows follow the one mechanism (one row per key, canonical order, the errors of `spec-regions.md` 5.3); a row is added to the list of 5.1.3 there | 3.4.2 |
| N-122 | **Respected**: the only binding cap is the bound on facilities, refused in the rotation of the starts; no place is added to `spec-world.md` 10.2 (CAP-A8) | 4.2.2 |
| N-016 | Applied: the rows of 12.4 for the parameter-owner table; no unnamed parameter was found | 12.4 |
| N-017 | Applied: the terms of 19.3, the setting words only in the flagged examples | 19.3 |
| N-021, N-107 | Applied: no faucet, no sink, no ledger reason, no money flow; a build's cost is the inputs, consumed by `recipe_input` | 10.3 |
| N-027, N-015 | Respected: every number is a placeholder, none is called stable | all |
| N-045 | Applied: every silent limit (a start cut by capacity, a refusal, a void completion, a pending store) is counted in `FacilityLimited` | 14.1 |
| N-125 | Applied: the rows of 14.5 and `Facility`, `Facilities` are functions of the saved records | 14.5, 15 |
| N-126 | Applied: the events of 14.1 and the metrics of 14.4 are rows owed to OBS; no `Warn` is emitted inside the tick | 14 |
| N-128 | Applied: a facility is not a per-agent field, so it has its own section | 15.1 |
| N-103 | Read: nothing for this spec to apply | |
| N-011, N-120 | Not tagged for CAP; STO applied them. A build of a store facility does not use the chargeable size | 5.4 |

New notes to add (area codes in the handoff): a route's effect on goods and not news
(N-INF, MIG, TRN); a build's cost booked as a loss (WEL, MIG); the idle site of a dead
owner (SPN, ORG, FIS).

### 19.2 What this spec needs from the other specs

None of these edits is made here (`writer-common.md` 1); the handoff has one table with
the file, the section and the exact change. In short:

| Spec | Why |
|------|-----|
| `spec-production.md` | the clauses `Needs`, `Builds`, `Repairs` in 3.1; `ErrRecipeNoOutput` waived (3.2.2, 10.2); the capacity and repair limits in the derivation of `k0` (4.2) and the reasons in order; the wear of a site beside the tool wear (4.4); the completion (5.3); `ErrEligibleHasTool` extended; the deferral of 6.7 answered |
| `spec-world.md` | the position P8.3b (6.10), the events of 6.11, the calls in 8.3, the table of draws unchanged (a note), the construction events (5.8.2) |
| `spec-transport.md` | the facility layer in the transit factor (4.1, 10.2, 10.3, 9.2 `Travel`), applied to a pair that is its own shortest direct edge (CAP 5.2.5) |
| `spec-resources.md` | the layer in the effective capacity (3.2.1, 3.4, 6.3.1), the clamp and its identity (6.4: the loss of a P8.3b clamp is in `EndowmentAdjusted` only), the new kind of `EndowmentAdjusted` (14), `MaxEndowment` on the effective capacity (3.4), the sustainable draw at tick 0 (11) |
| `spec-employment.md` | the formation checks of 5.5 gain `no_facility` (the capacity bound `committed_c + m`) and the build reasons, and 6.1.2/6.1.3 the facility (edit 27) |
| `spec-land.md` | the order of 4.5.1 and FR-LND-014 name CAP's five build reasons before `no_room`; the position P8.3a stays LND's and CAP's step is P8.3b after it (6.3.1) |
| `spec-storage.md` | the forced form of `removeStore` for a ruin (3.6.3, 5.5.2), `ErrStoreInUse` for `RemoveStore` (3.6.2), the cause `facility` of `StoreChanged` (14.1), `no_room` last with CAP's reasons before it (5.3, FR-STO-011), the store facility in the module table (10.2) |
| `spec-settlements.md` | the reader in the table of 6.3 (row H8) and 6.5 |
| `spec-agents.md` | the view's `Facilities` (6.3), the removal step (3.6), the note in 7.1.4 |
| `spec-integration.md` | the calls of 11 (a new 5.4), the queries (6.2), `Agent(a)` (4.7.2), the sentinels (7.1), the module-off rows |
| `spec-observability.md` | the events and kinds (4.1, 4.4), the rows (5.3), the metrics (6), the keys (7), the names table (3.2.4), the warnings |
| `spec-checkpoint.md` | the section (4.2), its validation (6.2), the format version (9.3) |
| `spec-config.md` | the module (3.2), fields (3.7), bounds (4.1), errors (7.3), warnings (7.4), the hash (8.1) |
| `spec-viability.md` | the capacity row and the runnability of a `Needs` recipe (13.4), `FacilityDecay` |
| `spec-calibration.md`, `spec-stability.md`, `spec-regions.md` | the registry rows (6); the register entries and scenarios (16); the row in the list of 5.1.3 |
| `rules/observability-and-logging.md` | keys (3.3.2), events (4.3), metrics (5.3, 5.4.1); a fresh read-receipt tag |

### 19.3 Terms (N-017)

**Facility** (a record of fixed capital with an owner, a region, a class and a
condition; not a good), **facility class** (data: a kind and its fields), **kind** (one of
`site`, `route`, `ground`, `store`), **site** (a facility a recipe needs, with a
capacity), **route** (a facility that speeds an edge), **ground improvement** (a
facility that raises an endowment's capacity), **store facility** (a facility that
installs a store), **condition** (wear units left), **durability** (of a facility class; a
different thing from a commodity's), **decay** (the loss per tick), **use wear**, **effect
fraction**, **gain** (the capped sum of effect fractions, `B`), **ruin**, **repair**,
**build** (a job whose completion creates facilities), **builder**, **repairer**,
**commissioning** (the entry of a new facility into the effect table at P8.3b), **effect
table**, **install** (of a store facility), **pending** (a store facility not installed).
The words `site`, `route` and `ground` are kinds of a class and not entities of the
world, and they are **not** the other specs' words of the same spelling: a `route` is not
REG's route (an edge, N-017 of REG) but a facility that speeds one, and a `site` is not
PRO's site factor of a yield profile (`Sited`, `no_site`) but a facility a recipe `Needs`,
with the refusal `no_facility`. The kind strings are data and are never branched on, so
the examples keep them. `Needs` is one field name on two types, a recipe's list of site
classes (4.1) and a class's settlement flag (3.1.1), and the text says which. `Durability` and `Capacity` repeat names of goods and of resources and are
qualified in the text as the class's. They go to the glossary of `doc.go` before code
uses them.

## 20. Requirements

IDs are stable; a requirement that moves keeps its ID (testing.md 4.1). All numbers are
placeholders (N-015, N-027).

### Functional

| ID | Requirement |
|----|-------------|
| FR-CAP-001 | With the module off, or on with no class and no initial row, the world MUST give the same state, the same events, the same snapshot rows and fields other than `ConfigHash`, `Modules` and `Names` (and `WorldCreated`'s `ConfigHash` and `Names`), and the same final state of the random source, as the world without it. A clause of a recipe with the module off MUST be `ErrModuleDependency`. |
| FR-CAP-002 | The module MUST take no random number at any position or call, and the number and order of draws of a tick MUST NOT depend on whether it is on. |
| FR-CAP-003 | A class MUST have one of the kinds `site`, `route`, `ground` or `store`; a field that the kind does not use MUST be its zero value and a field it requires MUST be set, else `ErrFacilityShape` (3.1.2). |
| FR-CAP-004 | Every field of a class and of an initial row MUST be within its bound (12.2), with `Decay` below `Durability`, else `ErrOutOfBounds`; the references MUST resolve (`ErrUnknownFacilityClass`, `ErrUnknownMode`, `ErrUnknownResource`, `ErrUnknownStoreClass`, `ErrUnknownRegion`, `ErrUnknownRole`, `ErrUnknownEdge`). |
| FR-CAP-005 | A facility MUST be a record with an ID assigned in creation order from 1 and never reused, held in ascending ID, and MUST NOT have a ledger account, lot, reason or price. |
| FR-CAP-006 | The effect of a route or ground facility MUST be `floor(BoostBP * Condition / Durability)` and the capacity of a site `ceil(Capacity * Condition / Durability)`, in 128 bits, and 0 at a condition of 0 (3.3.2). |
| FR-CAP-007 | At P8.3b every facility's condition MUST fall by its class's `Decay`, to a floor of 0, whether used or not. |
| FR-CAP-008 | A facility at a condition of 0 after the step of 6.3 MUST be removed, its ID MUST NOT be reused, and `FacilityChanged` (kind `ruined`) MUST be emitted with the cause of 6.2.1. |
| FR-CAP-009 | The facilities of an initial row MUST be created at construction with full `Durability`, in row order, round robin to the agents of the group (region, role) in ascending agent ID, or to the region's estate with no role, without a draw and with `FacilityChanged` (kind `added`, cause `initial`) after the last `AgentSpawned` in ascending ID. |
| FR-CAP-010 | A recipe MUST have at most one of `Builds` and `Repairs`; a recipe with either MUST have at least one input (`ErrFacilityNoCost`) and MUST have no non-discard output (an output it has is `Discard`, else `ErrFacilityShape`); `Toward` MUST be set only for a route class; `Needs` MUST name site classes, each once, at most `MaxRecipeNeeds`; a `TraineeEligible` recipe MUST have none of the three (`ErrEligibleHasTool`). |
| FR-CAP-011 | The yield factor of a build or repair recipe MUST act only as a test: a factor of 0 refuses the start, and a positive factor MUST NOT change the number of facilities created or the units restored. |
| FR-CAP-012 | The executions a start may make MUST be limited, for a recipe with `Needs`, by the sum of `cap(f)` of the principal's facilities of each needed class in its region, less the executions the same principal has already started or committed to the class in the tick (a bound per principal and tick); for `Builds` by the bounds on facilities; for `Repairs` by the units missing; and the limits 1 and 3 MUST be evaluated again at P2.7 against the running count of the tick, so that the excess is refused in the rotation order of the starts with the inputs kept; a start at 0 MUST be refused with the first applicable reason in the order of 4.2.2, in which the build reasons are before `no_room`. |
| FR-CAP-013 | A refused start MUST consume nothing, MUST be a `Debug` line and MUST be counted in `FacilityLimited.Refused`; an execution cut by capacity MUST be counted in `Cut`. |
| FR-CAP-014 | The completion of a build MUST create `k` facilities in the principal's region at the completion, with the principal as owner and full `Durability`, in ascending new IDs, and emit `FacilityChanged` (kind `built`) after `RecipeExecuted` (at P2.1, or at P2.7 for a job of `Duration` 1). |
| FR-CAP-015 | The completion of a repair MUST restore `k * Units` to the principal's own facilities of the class in its region, most worn first with ties to the lower ID, each to at most its `Durability`, and count the remainder as `Voided`. |
| FR-CAP-016 | No position of a tick MUST read the effect of a facility built, or the restoration of a repair, in that tick: effects MUST enter the tables at P8.3b and be read from the next tick. |
| FR-CAP-017 | A completion that can create or restore nothing MUST NOT fail or hold the job open, MUST count the executions in `Voided` and MUST NOT return the inputs. |
| FR-CAP-018 | The default strategy of a role with no `Saver` block MUST NOT request a build or repair recipe: in a world whose roles all use the default strategy and none has a `Saver` block, no facility is built and no condition is restored; a role with a `Saver` block requests one by the project rule of `spec-saving-and-investment.md` 5. |
| FR-CAP-019 | At the start of a job that needs a class, after `k` is final and before the inputs are taken, the `k` executions MUST be assigned to the owner's facilities of the class in the region most worn first, each wearing by `WearPerUse` per execution, to a floor of 0, with `k` at most `cap_c` less what the principal's other starts of the tick took, each facility taking in all the starts of one tick at most its `cap(f)` (the remaining `cap(f)` is the table's less the executions already assigned to it in the tick, and the order is that of the table); the wear MUST NOT be repeated for the job's duration. |
| FR-CAP-020 | A site that wears to 0 in a tick MUST serve that tick and MUST be ruined at P8.3b. |
| FR-CAP-021 | The effective transit factor MUST be `0` when the slot's is 0 and otherwise `min(MaxTransitFactorBP, floor(F_slot * (10 000 + B) / 10 000))` with `B` the capped sum of the effect fractions of the route facilities that affect the edge and name the mode, for a directed pair whose own direct edge is its shortest path, and 0 for any other pair (5.2.5); the facility MUST NOT write the slot, and `Slots()` MUST return the consumer's values. |
| FR-CAP-022 | A route facility with a target MUST affect both directed edges of the pair; one with none MUST affect every edge that leaves its region; it MUST act on new dispatches only. |
| FR-CAP-023 | A route facility MUST NOT change an edge's distance, a rumour's lag, a migration journey or a shortest path. |
| FR-CAP-024 | The effective capacity of an endowment MUST be `min(MaxEndowment, floor(Capacity * scale * (10 000 + B) / 10^8))` with `B` the capped sum of the effect fractions of the ground facilities of its region and resource, in 128 bits with one truncation and not above `MaxEndowment`, and a facility MUST NOT write the capacity-scale slot. |
| FR-CAP-025 | When the effective capacity falls below the stock the stock MUST be clamped and the excess counted as `Lost` in the endowment identity, with `EndowmentAdjusted` (kind `facility`) emitted only when units were lost. |
| FR-CAP-026 | A store facility MUST be installed by `addStore` at the next P8.3b (or at the call that created it); a refusal (including a fallen `Needs` flag of the store class) MUST leave it pending, counted in `Pending` and retried at every P8.3b; a pending facility MUST give no room; a ruin MUST remove the store by the forced form. |
| FR-CAP-027 | A store facility MUST NOT be transferable; it MUST be ruined (cause `role_changed`) when its owner's role has no base room and (cause `owner_removed`) when its owner is removed. |
| FR-CAP-028 | P8.3b MUST run, in this order and in ascending ID, the decay, the ruin, the install, the rebuild of the effect tables with the ground clamp, and the report, after P8.3 and before the audit. |
| FR-CAP-029 | The effect tables MUST be derived from the facility records and the configuration alone, rebuilt in full at P8.3b and at each call that changed a facility, and not saved. |
| FR-CAP-030 | On the removal of an owner a site, route or ground facility MUST pass to the estate of its region with `FacilityChanged` (kind `transferred`, cause `owner_removed`) at step 2g of the removal, before `AgentBankrupted` and `EstateOpened`, in ascending ID. |
| FR-CAP-031 | A facility MUST NOT scale with its owner's weight, MUST stay in place when its owner relocates or changes role, and the module MUST register no `region_bound` refusal for a site, route or ground facility. |
| FR-CAP-032 | The class's `Needs` flag MUST be tested only at the start of a build and at `AddFacilities`, MUST NOT remove or disable a facility, and MUST NOT gate an initial row (warning `facility_gate_closed`). |
| FR-CAP-033 | The `Work` view MUST carry the agent's own facilities in its region in ascending ID, at most `MaxFacilitiesPerAgent`, and the per-recipe `MaxExecutions` MUST be the `k0` of 4.2.2. |
| FR-CAP-034 | `AddFacilities` MUST validate every entry before applying any, create the facilities in slice order with the next IDs, return them, rebuild the tables and clamp a ground stock at the call, and emit `FacilityChanged` (kind `added`, cause `consumer`) per facility. |
| FR-CAP-035 | `AdjustFacilities` MUST clamp the result to `[0, Durability]`, MUST ruin a facility at a result of 0 at once, and MUST NOT emit an event for an entry that changes nothing. |
| FR-CAP-036 | `TransferFacilities` MUST move no money, MUST refuse a store facility (`ErrFacilityNotTransferable`) and a new owner at `MaxFacilitiesPerAgent`, and MUST emit `FacilityChanged` (kind `transferred`, cause `consumer`). |
| FR-CAP-037 | `Facility` and `Facilities` MUST return deep copies, in ascending ID, take no draw and emit no event. |
| FR-CAP-038 | `FacilityLimited` MUST be emitted at P8.3b once per (region, class) with a non-zero `Cut`, `Refused`, `Voided` or `Pending`, and never when all are 0; no event of the module MAY contain a map or a pointer. |
| FR-CAP-039 | With a logger and an observer attached the run MUST be identical to one without; no event MAY be built without an observer. |
| FR-CAP-040 | The module MUST emit no `Info` or `Warn` inside the tick; its warnings are at construction. |
| FR-CAP-041 | A checkpoint MUST save the next facility ID and every facility, and a restore MUST rebuild the tables and reject a record that breaks 15.3 with `ErrCheckpointCorrupt`; a restored run MUST equal the uninterrupted one. |
| FR-CAP-042 | The module MUST create and destroy no money or goods: a build's inputs are consumed by `recipe_input`, and the identities of the ledger MUST hold across a build, a ruin and a ground clamp. |
| FR-CAP-043 | Per class and region the count of facilities MUST equal the initial count plus those added and built less those ruined, which a test checks over a range of seeds. |
| FR-CAP-044 | The module MUST add no ledger reason, no money flow and no position other than P8.3b. |
| FR-CAP-045 | A world with a new facility chain (a site, a build recipe and a repair recipe) MUST be buildable from a configuration file alone (architecture.md 9.4). |
| FR-CAP-046 | Every product of this spec MUST be computed with checked arithmetic that fits within the bounds of 12.2; an overflow MUST NOT wrap (architecture.md 3.5). |
| FR-CAP-047 | An install, a ruin or a role change of a store facility MUST emit storage's `StoreChanged` (cause `facility`, or `role_changed`) before the `FacilityChanged` of the same facility, in ascending facility ID; `FacilityChanged` of a ruin of a store MUST carry `GoodsListed`; `RemoveStore` MUST be refused `ErrStoreInUse` while the agent's stores of the class would fall below its installed store facilities; `AddFacilities` of a store class MUST be refused `ErrFacilityRegion` unless the owner's account is in `Region`. |
| FR-CAP-048 | The events of P8.3b MUST follow `BoundaryFlow`, `EndowmentChanged` and, when tenure is on, `RentPaid` of P8.3a (`spec-land.md` 8.1), in the order of 14.1, and P8.3b MUST be the position after P8.3a. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-CAP-001 | Every iteration MUST be over a sorted slice or an array indexed by a typed ID, never over a map (architecture.md 7.5). |
| NFR-CAP-002 | With the default observer and logger, P2.5, P2.7 and P8.3b MUST allocate nothing per call in steady state (`testing.AllocsPerRun`, observability-and-logging.md 8.2). |
| NFR-CAP-003 | The effect fraction, the composition, the assignment of wear and the distribution of a repair SHOULD have fuzz tests asserting no panic, no overflow within the bounds and the invariants of 17 part 3. |
| NFR-CAP-004 | The facility step SHOULD be `O(F)` for `F` facilities and have a benchmark that calls `b.ReportAllocs()` at `MaxFacilities`. |
| NFR-CAP-005 | The pure rules SHOULD live in `internal/facility` over primitive slices and be tested without the world (architecture.md 2.9). |

## 21. Open questions

1. **Does a facility need a title?** Where LND is on, a ground facility's gain accrues
   to holders and a builder needs no title (5.3.4). Whether building on a title should
   need the holder's consent, or the builder to hold a share, is LND's with ORG; the
   engine lets anyone build in a region they stand in.
2. **Use by others, a fee and a lease.** A site is used by its owner only (5.1.5), so
   an orphaned site idles (7.1.2). The shape to start from is the delivery contract of
   `spec-merchants.md` 9 over a facility's capacity: an offer of capacity by an owner,
   a take by a user, a first-fit round (ADR 0005), a fee escrowed and the capacity
   pledged. An open-use pool would need the ration of `spec-resources.md` 5.3 with
   `d_i` 1 and the stateless key (`spec-world.md` 10.1.1). Not written here because a
   first consumer has no keeper of a mill that lends it and a library that guesses a
   contract form has to keep it.
3. **Capacity as concurrency.** A job could hold a unit of capacity for its `Duration`
   so that several agents queue for one site (5.1.4). It would need a pledge in the
   checkpoint and a rule at the start, as storage's room has.
4. **An edge limit, traffic wear and a toll.** TRN has no limit on an edge
   (`spec-transport.md` 6, 10.2). A bridge with a capacity (`edge_full`), a wear of a
   route by the shipments over it (so that a road is a commons) and a toll (FIS) are
   three seams, each needing a hook at dispatch. **A border toll** per carrying unit is FIS's (`spec-fiscal.md` 7) and is not this item; a toll on a facility's use stays open. Traffic wear is the natural one:
   `WearPerUse` per shipment on the edge at P3.2 and P3.3, most worn first, as the site's.
5. **Stewardship.** Repair acts on the principal's own facilities (4.3.2). Whether an
   agent may repair another's or an orphan (a charity, a levy of the commons) is
   ORG's or FIS's.
6. **Seasonal decay.** Decay is per tick and flat (6.1). A weather loss by season
   would be a table keyed by season, as the yield and the regeneration are
   (`spec-calendar.md` 5.3), and a profile on the class.
7. **Loss by conflict.** RSK lowers none in version 1 (a raid on a bridge): the call of
   11.2 stays the consumer's; the reserved shape is `spec-risk.md` open question 13 (a table
   `Facilities` keyed by region and class, a draw per facility at P8.3b, the internal form of
   `AdjustFacilities` with a cause `risk`).
8. **More effects.** A ground facility that raises the regeneration scale and not the
   capacity, a facility that lowers the input of a recipe (a furnace): a gain on a recipe
   is TEC's or a fifth kind with an owner.
9. **A default repair rule.** The default builds and repairs nothing (4.5.1). A second
   registered strategy, as MER's "merchant", that repairs below a threshold of the
   condition would let a pack with decay and no consumer strategy keep its capital; the
   threshold would be a tunable of that strategy's block. Decided by the first
   consumer's world: it has a player.
10. **The `ceil` of a capacity and the `floor` of a gain** (3.3.2): a single rule would
    be simpler. The cost of the `ceil` is that a nearly worn mill serves a whole
    execution; of the `floor`, that a nearly worn road gives nothing.
11. **A role ceiling on facilities.** SET offers limits for entries; a cap on the facilities
    of a class in a region by the settlement's size would be such a limit (SET 6.5). It is
    not offered: the world bound and the gate are the only limits.
12. **A path rule for a route.** A layer applies in version 1 only to a pair that is its own
    shortest direct edge (5.2.5). Whether the gain of a multi-hop pair is the composition
    over the edges of REG's path (harmonic, or the least gain on the path), and what a road
    on a direct edge that is not the shortest does, wants a run with a road network and the
    transport module's own path cache; the cost of the simple rule is a road network that
    is only as good as its direct edges.

## 22. Where this lives in the code

| Concern | Home |
|---------|------|
| `FacilityClass`, `InitialFacility`, `Infrastructure`, `FacilityBuild`, `FacilityRepair`, `FacilityID`, `FacilityClassID`, `FacilityState`, `FacilityView`, `NewFacility`, `FacilityAdjustment`, `FacilityTransfer`, the kinds, causes and reasons | root package, `infrastructure.go`; the events in `event.go` |
| The bounds of 12.2 | root package, `bounds.go` |
| Validation, the sentinels and the warnings of 13 | root package, `validate.go`, `errors.go` |
| The effect fraction (3.3.2), the composition (5.2.2, 5.3.2), the capacity table, the most-worn-first assignment (5.1.3), the repair distribution (4.3.2), the round robin of the initial rows: pure functions | `internal/facility`, over primitive slices (architecture.md 2.9), standard library only |
| The facility list, the effect tables, the facility step (P8.3b), the calls and queries of 11 | root package, `infrastructure.go` and `world.go`, calling `internal/facility`, `internal/ledger` (none: no ledger operation), and storage's `addStore` and `removeStore` inline |
| The checkpoint section | `checkpoint.go` with `internal/codec` |
| The overlay of the fixture | `testdata/chains/overlay/infrastructure.toml` (REF builds it: `spec-reference-chain.md` 7) |
| The model | `doc/specs/models/cap/` |
