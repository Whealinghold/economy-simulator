# Spec: Configuration, defaults, bounds and validation

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CFG-NNN` and
`NFR-CFG-NNN`, which tests cite (testing.md section 4). They are in section 11.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).

**This spec was written before the specs it depends on.** The roadmap says CFG
needs every other spec, because the configuration holds all their parameters.
What is fixed here is the framework: the shape of the configuration, where
constants live, the file format, the validation machinery, the structural
rules, the hash and the registry. The *catalogue* of fields and of per-field
rules grows with each spec (section 9). It now includes the fields and rules of
production, beliefs, agents, viability, the spawner, demand, regions, money supply, population, transport, resources, information, the external sector, subsistence, migration and merchants, and of the fourteen later-stage modules (settlements, storage, employment, tenure, credit, risk, infrastructure, insurance, polity, fiscal, institutions, policy, saving and technology; sections 3.7.1, 4.1, 7.3, 7.4). A spec that is added later adds its rows in the same change as its own (9.2). Where a section depends on a spec that does not exist, it says so.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

The configuration is the data that defines a world: commodities, recipes, roles,
regions, the population, and the parameters of the core and of each module. This
spec covers:

- the shape of the configuration in Go, and how optional modules are switched on
  (section 3);
- the three kinds of constant, the bounds and their derivation (section 4);
- defaults, defined once (section 5);
- the file format, the `config/` package and how packs are layered (section 6);
- validation: the framework, the structural rules and the rules other specs
  contribute (section 7);
- the **effective configuration** and its hash, which the checkpoint stores
  (section 8);
- the parameter registry and the checks that keep it true (section 9).

It implements architecture.md 8.5 and 9.5 to 9.8. Out of scope: the content of
a pack (an RPG pack is a deliverable under `examples/`, roadmap), the fields of
commodities, recipes and roles (`spec-goods.md`, `spec-production.md`,
`spec-agents.md`), and the checkpoint format (`spec-checkpoint.md`).

## 2. Principles

2.1. The configuration is **data**. A new commodity, recipe, role, region or
chain needs no Go change (architecture.md 9.4).

2.2. The configuration types are plain structs in the root package, with no
struct tags and no embedding (architecture.md 4.7, 9.3, 12.6). A consumer can
build a world from Go alone. The file format is a convenience in `config/`,
which produces exactly those structs and nothing else.

2.3. Validation is complete and happens before a world exists. A running world
never fails because of a configuration mistake (architecture.md 8.5).

2.4. A validation failure names *where* (a path), *which rule* (a sentinel
error) and *what value*. A designer fixing a pack should not need to read the
code.

2.5. The same effective configuration always gives the same hash and the same
run. Nothing about the order of entities in a file, a slice or a map, nor the
layout of the file, may change either.

2.6. Numbers in a configuration are integers (architecture.md 3.5). A floating
point literal is an error, not a value to round.

## 3. The shape of the configuration

### 3.1 Sections

`Config` has these sections. Fields of the entity types are specified by the
spec that owns them.

| Section | Holds | Owner of the fields |
|---------|-------|---------------------|
| `World` | world-level parameters: `Scale`, `MinAgents`, the baseline `Upkeep`, the belief and agent tunables, and the like (3.7) | the owner specs, one row each in the parameter table |
| `Calendar` | the structure of time: hours in a day, days in a week, the months, the length of a tick, the seasons. One section, not an entity; Earth by default | `spec-calendar.md` |
| `Commodities` | commodity definitions | `spec-goods.md` |
| `Recipes` | recipe definitions | `spec-production.md` |
| `Yields` | yield profiles, one per (region, recipe): a basis-point factor for all seasons or by season. A section of its own, keyed by two IDs, so that a recipe names no region and a region names no recipe (`spec-production.md` 6.1); the values move to `spec-resources.md` when it exists | `spec-production.md` |
| `Roles` | role definitions and role parameters | `spec-agents.md`, `spec-welfare-and-wages.md`, `spec-skill-barriers.md` and others |
| `Regions` | region definitions and region parameters, including reference prices, the reference basket and extra upkeep | `spec-regions.md`, `spec-currency.md`, `spec-agents.md`, `spec-estates.md` |
| `Edges` | the directed routes between regions: `From`, `To`, `Distance` (an integer in the pack's distance unit); a section of its own, a list sorted by `(From, To)` | `spec-regions.md` 4.1 |
| `RegionPrices` | reference-price overrides by region and commodity, one row per key | `spec-regions.md` 5.2 |
| `Groups` | the population: households of a role in a region, with an optional scale | `spec-scale.md` |
| `WarmStart` | the steady-state seed table: seed prices, starting money, stock and jobs per group, estate pools. A pointer: nil means no warm start, and an empty non-nil `WarmStart` is a different configuration (8.2). It is a section and not a module: it switches no mechanism on | `spec-viability.md` 8.1 |
| `Modules` | one optional entry per module (3.2); `Demand` is the first with its own tables (`Wants`, `Tastes`, `RegionTastes`); `MoneySupply` (`Flows`), `Population` (schedules and `Regions`) and `Transport` (`Modes`) follow; `Resources` (`Resources`, `Endowments`, `HorizonYears`), `Information` (`InfoSpeed`, `RumourErrorBP`, `RumourMaxAge`) and `External` (`Nodes`) come with the specs of wave 3; `Subsistence` (`Recipes`, `Roles`, `EnterTicks`, `ExitTicks`, `RetryJobs`), `Migration` (nine tunables and `Regions`) and `Merchants` (`MaxOpen`) come with the specs of wave 4; and the fourteen modules of wave 5 follow, in this order and no other: `Settlements`, `Storage`, `Employment`, `Tenure`, `Credit`, `Risk`, `Infrastructure`, `Insurance`, `Polity`, `Fiscal`, `Institutions`, `Policy`, `Saving` and `Technology` (their fields are in 3.7.1); and `Estates`, the pool policy, the 25th and last (6.2.4). The position of a module in this row, counted from 1, is its field id in the hash (8.2.2): a module that is added later is appended, never inserted, and the order in which modules are built does not change it. A module that is off contributes nothing to the hash, and a field appended to an existing struct at its off value (its zero value) is written to nothing, the rule of D5 (8.1, 8.2; FR-CFG-105 pins the hash of the all-off reference configuration). At Stage 0 `Modules` is declared with no field and each module's pull request appends its pointer | each module's spec |

The entity sections are slices of structs, each with an `ID` string. Package
`config/` writes them from file tables (6.2).

### 3.2 Modules are on when configured

A module is an optional mechanism that is off unless configured (ADR 0001,
decision 2). In `Config`, `Modules` has one pointer field per module. A `nil`
pointer is off; a non-nil pointer is on, and its fields are the module's
parameters. There is no separate `Enabled` flag that could disagree with the
presence of its parameters. The zero `Modules` is a closed economy with no
faucets, taxes, credit or growth, and it is the tested baseline.

Each module lists the modules it needs, in its own spec. Validation rejects a
module that is on while a module it needs is off (7.3, `ErrModuleDependency`).

### 3.3 Identifiers

An ID is a string matching `[a-z][a-z0-9_]*`, at most 64 bytes, unique within its
kind (commodity, recipe, role, region). The restriction keeps IDs usable as file
keys, log values and metric attributes without escaping. The internal typed IDs
(`CommodityID`, `RoleID`, architecture.md 3.3) are assigned in the order of the
IDs sorted as strings, after canonicalisation (3.4), so they do not depend on how
a file or a slice was ordered. The one exception is the settlement class, whose
`SettlementClassID` is its rank by `Enter` and not the rank of its ID string (3.4,
`spec-settlements.md` 3.2). The word `pool` is **reserved** as a role ID: it names the
holder that is the pool of a region in a holding of the tenure module, so a role with
that ID is `ErrDuplicateID` (`spec-land.md` 3.3.2, 11.1).

### 3.4 Canonical order

Before a world is built or a hash is taken, every entity slice is sorted by ID
(groups by region, then role). The order in which an author wrote entities has no
effect. This follows architecture.md 7.5: nothing that influences results may
depend on an arbitrary order. A recipe's inputs and outputs, being lists of
(commodity, quantity), are sorted by commodity. Edges are sorted by `(From, To)` as region indices, region prices by region then commodity, money flows by ID, transport modes by ID, and the region-keyed tables of 5.1 of `spec-regions.md` by region. The lists and rows of the modules of wave 5 follow the same rule and are sorted as their specs say: by ID (the classes of storage, infrastructure and technology; the bodies, rights and controls; the jurisdictions; the six lists of fiscal flows; the policy actors and their levers), by the key of the row (a role, a region, a `(Region, Resource)` term, a `(Region, Resource, Holder)` holding with the pool before the roles, a `(From, To, Mode)` route row, a `(Region, Class)` store row, a `(Lever, At)` schedule step, which orders the steps of a policy actor by tick, a `Reaction` by `Lever`), and the recipes of an employer or a technology field by recipe ID.

There are two exceptions. The first is the calendar's months and seasons. Their order is their
meaning (the third month, the season after the last), so they are **ordered
lists**: they are kept as written, are written in a file as arrays and not as
tables keyed by ID, and an overriding document replaces a list as a whole (6.4).
The second is the settlement classes of `Modules.Settlements`: they are sorted by `Enter`, the households at which a class is reached, and not by ID, because the rank is the meaning (`spec-settlements.md` 3.2). A class's `Flags` are sorted by name, its `Limits` by `Name` and the region rows by region; the written order of classes has no effect, as for any entity.

### 3.5 Copy on construction

`NewWorld` takes the `Config` by value and deep-copies every slice and nested
struct. Changing the caller's `Config` afterwards cannot change a world. The
world's configuration is part of the state the world owns (ADR 0002, decision 6).

### 3.6 File and Go agree

A configuration read from a file and one built in Go with the same content are
equal, have the same hash and give the same run. A test builds the reference
chain both ways.

### 3.7 The catalogue of fields

This is the list of what each section holds today, with the spec that owns the
meaning, default and bound of each field. The parameter-owner table in
`specs/README.md` is the registry (section 9); this table is the shape. A field
absent from both is a gap (9.2). Fields marked (p) are presentation only (8.3).

The section `Config.World` has the Go type `WorldParams` and its constructor is
`DefaultWorldParams` (decision D140 of `plans/plan-phase-1-11.md`): the simulation type
is `World` (architecture.md 2.2, 12.1; `spec-world.md` 3.1) and the root package cannot
hold two. The field `Config.World`, the file table `[world]` and the field ids of 8.2.2
do not change, so no hash input moves. In the tables of this spec and of the owner
specs the scope word `World` names this section, as the word `Config` does; Go code
reads it as `WorldParams`.

| Section | Fields | Owner |
|---------|--------|-------|
| `World` | `Scale`, `MinAgents` | `spec-scale.md` |
| `World` | `BulkFactor` | `spec-goods.md` |
| `World` | `WelfareWindow` (the base window, default 30), `WindowCycles` (default 2); the effective window of a role is derived (4.2, 7.4.1), see 9.3 | `spec-welfare-and-wages.md` 2.3.1, 2.3.3 |
| `World` | `Upkeep` (the baseline list), `CoverTicks`, `InputJobs`, `OutputCoverTicks`, `FavourFloor`, `ReserveTicks`, `SpendMaxShare`, `SpendHalfTicks`, `RestBand`, `ProduceHaltRatio`, `BasketTolerance` | `spec-agents.md` 12 |
| `World` | `BeliefStep`, `BeliefAdjustProb`, `BeliefSignificance`, `BeliefStockLow`, `BeliefStockHigh`, `BeliefMaxAge`, `BeliefWidthInit`, `BeliefWidthMin`, `BeliefWidthMax`, `BeliefWidthStep`, `BeliefEvidenceShare` (default 5 000, the sixteenth), `CostFloor` (default 7 000), `CostCeiling`, `RefBandLow`, `RefBandHigh`, `PriceWindow` | `spec-beliefs.md` 11 |
| `Calendar` | `HoursPerDay`, `DaysPerWeek`, `Months`, `TicksPerDay`, `DaysPerTick`, `StartDay`, `Seasons`, `SeasonStart` | `spec-calendar.md` 3.1 |
| `Commodities[]` | `ID`, `Mass`, `Bulk`, `ShelfLife`, `Durability`, `Classes`, `Raw`, `ReferencePrice`, `Label` (p), `UnitLabel` (p) | `spec-goods.md` 8 |
| `Recipes[]` | `ID`, `Duration`, `Inputs`, `Outputs` (each with `CostShare`, `Discard`), `Requires` (each with `WearPerUse`), `Draws`, `Sited`, `TraineeEligible`, `TraineeYield`, `Label` (p) | `spec-production.md` 3.1, 11 |
| `Yields[]` | `Region`, `Recipe`, `All`, `Seasons` (a list of season and basis points) | `spec-production.md` 6.1 (the format), `spec-resources.md` 7 (the values) |
| `Roles[]` | `ID`, `Recipes`, `Upkeep`, `Trades`, `StartMoney`, `StartStock`, `Strategy` (a name, hashed: `default`, `directed`, both built in, or a registered name, 7.1.4), `Label` (p) | `spec-agents.md` 3.3 |
| `Roles[]` | `Unskilled`, `TargetRatio`, `TargetSpread`, `DefaultRatio`, `InKind`, `ExpectedWorkTicks`, `MinWorkShare` | `spec-welfare-and-wages.md` 4.2 |
| `Roles[]` | `TrainingTicks`, `MaxTrainees`, `TeachingLoad` | `spec-skill-barriers.md` |
| `Roles[]` | `MerchantFeeShare` | `spec-carriage.md` 2.2 |
| `Regions[]` | `ID`, `ReferenceBasket`, `ExtraUpkeep` | `spec-currency.md` 5, `spec-agents.md` 5.2 |
| `Regions[]` | `CapitalReturn` | `spec-skill-barriers.md` 2.5 |
| `Regions[]` | `EstateMarkdown`, `PoolRelease`, `ReliefBelow` | `spec-estates.md` 3 |
| `Modules.Estates` | no field; present = the pool policy is on (relief, and the pool as a flow target). `PoolRelease` and `ReliefBelow` above are written only when it is on (8.2.1) | `spec-estates.md` 2.6, 6.2.4 here |
| `Groups[]` | `Region`, `Role`, `Households`, `Scale` | `spec-scale.md` 3.1 |
| `WarmStart` | `Prices`, `Wages` (the starting wage per region), `Groups`, `Pools` | `spec-viability.md` 8.1 |
| `World` | `Spawner` (the registered name of the spawner, hashed; default `"default"`), `ReviewTicks`, `EntryShare`, `RoleHysteresis`, `RoleCooldown`, `EntryRatio`, `ChangePrivation`, `RemovePrivation`, `DeathStep`, `ExitShare`, `Renewal`, `EntryPoolShare` | `spec-spawner.md` 3.1.3, 11 |
| `World` | `Clearing` (the registered name of the clearing rule, hashed; default `"average"`, the built-in `AverageClearing`; an empty name is an error, 5.2) | `spec-clearing.md` 8 (decided in `spec-world.md` 4.2) |
| `Roles[]` | `Closed` (the default spawner places nobody in the role) | `spec-spawner.md` 6.2, 11 |
| `Modules.Demand` | `Wants[]` (`ID`, `Items` each with `Commodity` and `Weight`, `Quantity`, `Period`, `Rank`, `Elasticity`, `Ceiling`, `Substitution`, `Label` (p)), `Tastes[]` (`Role`, `Want`, `Scale`), `RegionTastes[]` (`Region`, `Want`, `Scale`); present = the module is on | `spec-demand.md` 3.2, 9 |
| `Regions[]` | `Label` (p) | `spec-regions.md` 3.2 |
| `Edges[]` | `From`, `To`, `Distance` | `spec-regions.md` 4.1 |
| `RegionPrices[]` | `Region`, `Commodity`, `Price` | `spec-regions.md` 5.2 |
| `World` | `DistanceLabel` (p) | `spec-regions.md` 4.1.2 |
| `Modules.MoneySupply` | `Flows[]` (`ID`, `Kind`, `Target` with `Kind`, `Region` and `Role`, and `Store` and `Jurisdiction` (owned by `spec-storage.md` and `spec-fiscal.md`, 3.7.1); `Amount`, `Level`, `RateBP`, `Exempt`, `Per`, `Start`, `Until`, `Limit`, `NotIncome`, `Label` (p)); present = the module is on | `spec-money-supply.md` 4.1, 11.1 |
| `Modules.Population` | `BirthSchedule[]` and `DeathSchedule[]` (each knot `Ratio`, `Rate`), `MinHouseholds`, `BirthStake`, `Regions[]` (`Region`, `BirthScale`, `DeathScale`, `HouseholdCap`); present = the module is on | `spec-population.md` 3.5, 8 |
| `Modules.Transport` | `Modes[]` (`ID`, `Speed`, `UnitMass`, `UnitBulk`, `Crew`, `Vehicle`, `WearPerTick`, `Feed[]` each with `Commodity` and `Quantity`, `Haul`, `HaulSpan`, `Label` (p)); present = the module is on; it needs the region graph | `spec-transport.md` 3.1, 11 |
| `Modules.Resources` | `Resources[]` (`ID`, `Shape`, `RatePerYear`, `RateBP`, `Refuge`, `Capacity`, `InitialBP`, `PlanShareBP`, `Abundance[]` each with `Share` and `Factor`, `Seasons[]` each with `Season` and `BP`, `Label` (p)), `Endowments[]` (`Region`, `Resource`, `Capacity`, `Initial`, `RegenScaleBP`), `HorizonYears`; present = the module is on; it needs production | `spec-resources.md` 3.3, 12 |
| `Modules.Information` | `InfoSpeed`, `RumourErrorBP`, `RumourMaxAge`; present = the module is on; it needs the region graph | `spec-information.md` 8 |
| `Roles[]` | `Informed` (default false; needs `Modules.Information`) | `spec-information.md` 6.1, 8 |
| `Modules.Subsistence` | `Recipes[]`, `Roles[]` (`Role`, `Recipes[]`), `EnterTicks`, `ExitTicks`, `RetryJobs`; present = the module is on | `spec-subsistence.md` 3.2, 12 |
| `Modules.Migration` | `MoveSpeed`, `MoveReviewTicks`, `MoveBelow`, `MoveGain`, `MoveWidth`, `MoveCooldown`, `MoveHorizon`, `MoveOutShare`, `MoveInShare`, `Regions[]` (`Region`, `OutShare`, `InShare`); present = the module is on; it needs `Modules.Information` | `spec-migration.md` 3.1, 11 |
| `Roles[]` | `Fixed` (default false: the households of the role are never moved by the migration module) | `spec-migration.md` 3.1, 11 |
| `Modules.Merchants` | `MaxOpen`; present = the delivery contract is on; it needs `Modules.Transport` | `spec-merchants.md` 13 |
| `Roles[]` | `Merchant` (the block: `Modes`, `LotUnits`, `FeeShareSpreadBP`, `MaxSourceAge`, `TimeCostBP`, `ConsigneeCutBP`, `Consigns`, `TakeMarginBP`, `TakeUnits`; nil = not a merchant role; needs `Modules.Transport` and `Modules.Information`). `MerchantFeeShare` keeps its row above | `spec-merchants.md` 3.2, 13 |
| `Modules.External` | `Nodes[]` (`Region`, `Lines[]` each with `Commodity`, `Base`, `PriceBySeason[]` (`Season`, `BP`), `TrendBP`, `FloorBP`, `CeilingBP`, `WalkStepBP`, `WalkRevertBP`, `WalkBoundBP`, `Sell` and `Buy` each with `MarginBP`, `Capacity`, `Window[]` (`Season`, `BP`), `TotalUnits`, `TotalValue`; `Gates[]` (`Region`, `ScaleBP`); `Label` (p)); present = the module is on; it needs the region graph | `spec-external-sector.md` 3.3, 13.1 |

The fields of the fourteen modules of wave 5, and the fields they add to `Role`, `Recipe` and `FlowTarget`, are in 3.7.1 (one row per field, with its scope, default, bound and owner section).

Not fields: the seed, the logger, the observers, `WithAudit`, `WithReseed` (a
restore-only option that gives a different continuation, `spec-checkpoint.md` 6.8, and
never a field), the extension registrations (`WithStrategy`, `WithSpawner`, `WithClearing` and `WithPolicy`, whose *names* are hashed through `Role.Strategy`, `World.Spawner`, `World.Clearing` and `PolicyActor.Policy` and whose code cannot be) and the solve options of `spec-viability.md` 11
(`StockCoverTicks`, `MoneyBufferTicks`, `ResidualWarnBP`, `MaxPivots`, `Lanes`):
they shape how a run or a table is produced and not what a world is. The yield
shocks of `spec-production.md` 6.8 are world state, not configuration. So are the other slots set between ticks: the flow scale of `spec-money-supply.md` 4.4.3, the birth and death scales of `spec-population.md` 6.6, the edge statuses of `spec-regions.md` 7 and the transit factor of `spec-transport.md` 10.3, the endowment shock scales of `spec-resources.md` 6.3, the outside price shock of `spec-external-sector.md` 6.5, the rent scales of the tenure module (`EndowmentScale{Region, Resource; BP}`, `spec-land.md` 9.1), the rate scales of the credit module (`RegionScale{Region; BP}`, `spec-credit.md` 13.1), the legal levels and the right suspensions of the institutions module (`spec-institutions.md` 7.1), the flow scale of a fiscal flow (`spec-fiscal.md` 13.1, the slot of `spec-money-supply.md` 4.4.3) and the route and store risk scales of the risk module (`spec-risk.md` 9); the public calls that set them are INT's. State, not fields, are also the own-use mode `OwnUse` and the counter `OwnUseJobs` of an agent (`spec-subsistence.md` 5.3), the journey of a migrating agent (`spec-migration.md` 14) and the contracts in flight (`spec-merchants.md` 16).

### 3.7.1 The fields of the fourteen later modules

This is the catalogue of the fourteen modules of wave 5 (3.1), in the order of the `Modules` row, and of the fields they add to the types of the earlier specs (`Role`, `Recipe`, `FlowTarget`). It has one row per field that is not presentation-only: the presentation-only `Label` of a store class, a settlement class, a jurisdiction, a facility class, a technology field, a policy actor, a body, a right, a control, a rate limit and each of the six fiscal kinds is on the allow-list of 8.3 and has no row (9.2). The **Scope** is the Go type that holds the field, as the owner spec names it; the module's own row has the scope `Modules`. `Default` and `Bound` are the owner's, and "required" means that the field has no default. Bounds in capitals are constants of 4.1. The **Owner** is the spec and the section that says what the field means; the owner table of `specs/README.md` and the registry of `spec-calibration.md` are written from this table (9.2).

| Scope | Field | Default | Bound | Owner |
|-------|-------|---------|-------|-------|
| `Modules` | `Settlements` | nil (off); needs no other module | | `spec-settlements.md` 3.1, 8 |
| `Settlements` | `Classes` | required, at least one | 1 to `MaxSettlementClasses` | `spec-settlements.md` 3.1, 8 |
| `Settlements` | `BandBP` | 1 000 | 0 to `MaxBandBP` | `spec-settlements.md` 3.4.2, 8 |
| `Settlements` | `Dwell` | 30 | 0 to `MaxWelfareWindow` | `spec-settlements.md` 3.4.3, 8 |
| `Settlements` | `Regions` | none | at most `MaxRegions` | `spec-settlements.md` 3.5, 8 |
| `SettlementClass` | `ID` | required | the ID pattern of 3.3, at most `MaxIDLength` bytes, unique | `spec-settlements.md` 3.1, 8 |
| `SettlementClass` | `Enter` | required; 0 for the first | 0 to `MaxHouseholds`, strictly increasing | `spec-settlements.md` 3.4.1, 8 |
| `SettlementClass` | `Flags` | none | each at most once; at most `MaxClassFlags` distinct in the module | `spec-settlements.md` 3.3.1, 8 |
| `SettlementClass` | `Limits` | none | each name at most once; at most `MaxClassLimits` distinct in the module | `spec-settlements.md` 3.3.2, 8 |
| `ClassLimit` | `Name` | required | the ID pattern of 3.3 | `spec-settlements.md` 3.3.2, 8 |
| `ClassLimit` | `Value` | required | 0 to `MaxQuantity` | `spec-settlements.md` 3.3.2, 8 |
| `RegionSettlement` | `Region` | required | a defined region that is not an outside node | `spec-settlements.md` 3.5, 8 |
| `RegionSettlement` | `ThresholdBP` | 10 000 | 1 to `MaxThresholdBP` | `spec-settlements.md` 3.5.2, 8 |
| `RegionSettlement` | `Floor` | none | a defined class ID | `spec-settlements.md` 3.5.3, 8 |
| `Role` | `Needs` | none | a flag some class carries; the module on | `spec-settlements.md` 6.4.1, 8 |
| `Role` | `Ceiling` | none | a limit some class carries; the module on | `spec-settlements.md` 6.4.2, 8 |
| `Modules` | `Storage` | nil (off) | | `spec-storage.md` 12.1 |
| `Storage` | `Base` | `nil`: unlimited | `Mass` and `Bulk` each 0 to `MaxRoom` | `spec-storage.md` 12.1 |
| `Storage` | `Roles` | none | at most one row per role | `spec-storage.md` 12.1 |
| `Storage` | `Classes` | none | at most `MaxStoreClasses` | `spec-storage.md` 12.1 |
| `Capacity` | `Mass` | required | 0 to `MaxRoom` | `spec-storage.md` 12.1 (the type is `spec-goods.md` 3.2) |
| `Capacity` | `Bulk` | required | 0 to `MaxRoom` | `spec-storage.md` 12.1 (the type is `spec-goods.md` 3.2) |
| `RoleRoom` | `Role` | required | an existing role | `spec-storage.md` 12.1 |
| `RoleRoom` | `Base` | `nil`: unlimited | as `Storage.Base` | `spec-storage.md` 12.1 |
| `RoleRoom` | `Stores` | none | at most `MaxStoresPerAgent`, existing classes; not on an unlimited role | `spec-storage.md` 12.1 |
| `StoreClass` | `ID` | required | the ID pattern of 3.3 | `spec-storage.md` 3.2, 12.1 |
| `StoreClass` | `Mass` | required | 0 to `MaxRoom`; not both 0 | `spec-storage.md` 3.2, 12.1 |
| `StoreClass` | `Bulk` | required | 0 to `MaxRoom`; not both 0 | `spec-storage.md` 3.2, 12.1 |
| `StoreClass` | `Needs` | empty: none | an existing settlement flag; empty unless `Modules.Settlements` is present | `spec-storage.md` 3.2.3, 12.1 (the flag is `spec-settlements.md`'s) |
| `FlowTarget` | `Store` | empty (every class) | a configured store class; meaning only for the kind `stores` | `spec-storage.md` 7.1.2, 12.3 (the type is `spec-money-supply.md` 4.1) |
| `Modules` | `Employment` | nil (off) | | `spec-employment.md` 3.1, 13 |
| `Employment` | `MaxOpen` | 1 000 | 1 to `MaxHireRecords` (the most hires open at once in a region) | `spec-employment.md` 13 |
| `Role` | `Hireable` | false | a bool; needs the module | `spec-employment.md` 3.1, 13 |
| `Role` | `Employer` | nil: the role does not hire | needs the module | `spec-employment.md` 3.2, 13 |
| `Employer` | `Recipes` | required | at least 1, known, no duplicate, on the role's `Recipes` | `spec-employment.md` 3.2, 13 |
| `Employer` | `MaxHires` | 4 | 1 to `MaxEstablishment` | `spec-employment.md` 3.2, 13 |
| `Employer` | `WorkerShareBP` | 5 000 | 1 to 10 000 | `spec-employment.md` 3.2, 5.4, 13 |
| `Employer` | `HireYieldBP` | 10 000 | 1 to `MaxYieldBP` | `spec-employment.md` 3.2, 6.1, 13 |
| `Modules` | `Tenure` | nil (off); needs `Modules.Resources` | | `spec-land.md` 3.3, 10 |
| `Tenure` | `Terms` | none (open access) | one row per (region, resource) | `spec-land.md` 3.3, 10 |
| `Tenure` | `Holdings` | none | at most `MaxHoldings`; one row per (region, resource, holder) | `spec-land.md` 3.3, 10 |
| `Tenure` | `RentStepBP` | 0 (the rent does not move) | 0 to `MaxRentStepBP` | `spec-land.md` 5.5, 10 |
| `Tenure` | `RentFloorBP` | 0 | 0 to 10 000 | `spec-land.md` 5.5, 10 |
| `Tenure` | `RentCeilingBP` | 20 000 | 10 000 to `MaxRentCeilingBP` | `spec-land.md` 5.5, 10 |
| `Tenure` | `OwnUseExempt` | false | a bool | `spec-land.md` 5.3.4, 10 |
| `Tenure` | `MaxHolderShareBP` | 10 000 (off) | 1 to 10 000 | `spec-land.md` 6.1.2, 10 |
| `TenureTerm` | `Region` | required | an existing region; not an outside node | `spec-land.md` 10 |
| `TenureTerm` | `Resource` | required | an existing resource | `spec-land.md` 10 |
| `TenureTerm` | `Rent` | 0 | 0 to `MaxPrice` | `spec-land.md` 5.2, 10 |
| `TenureTerm` | `Closed` | false | a bool | `spec-land.md` 4.3, 10 |
| `Holding` | `Region` | required | an existing region; not an outside node | `spec-land.md` 10 |
| `Holding` | `Resource` | required | an existing resource | `spec-land.md` 10 |
| `Holding` | `Holder` | required | an existing role, or `pool` | `spec-land.md` 10 |
| `Holding` | `ShareBP` | required | 1 to 10 000; the sum over an endowment at most 10 000 | `spec-land.md` 10 |
| `Modules` | `Credit` | nil (off); needs no other module | | `spec-credit.md` 4.1, 14 |
| `Credit` | `MinPrincipal` | required | 1 to `MaxMoney` | `spec-credit.md` 14 |
| `Credit` | `ShieldTicks` | 0 (no shield) | 0 to `MaxCreditTerm` | `spec-credit.md` 8.2, 14 |
| `Credit` | `WriteOffTicks` | 0 (never) | 0 to `MaxCreditTerm` | `spec-credit.md` 8.5, 14 |
| `Credit` | `MaxClaims` | required | 1 to `MaxOpenClaims` | `spec-credit.md` 6.4, 14 |
| `Role` | `Credit` | nil: the role does not deal in credit | exactly one of `Lend` and `Borrow` | `spec-credit.md` 4.2, 14 |
| `CreditTerms` | `Lend` | nil | one of the two blocks, not both | `spec-credit.md` 4.2 |
| `CreditTerms` | `Borrow` | nil | one of the two blocks, not both | `spec-credit.md` 4.2 |
| `Lend` | `RateBP` | required (0 is interest-free and is written, never defaulted) | 0 to `MaxCreditRateBP` | `spec-credit.md` 7.1, 14 |
| `Lend` | `MaxTermTicks` | required | 1 to `MaxCreditTerm` | `spec-credit.md` 10.3, 14 |
| `Lend` | `KeepTicks` | 0 | 0 to `MaxCreditTerm` | `spec-credit.md` 10.3, 14 |
| `Lend` | `ShareBP` | required | 1 to 10 000 | `spec-credit.md` 10.3, 14 |
| `Borrow` | `MaxRateBP` | required (0 accepts only interest-free credit) | 0 to `MaxCreditRateBP` | `spec-credit.md` 10.2, 14 |
| `Borrow` | `MaxDebtTicks` | required | 1 to `MaxCreditTerm` | `spec-credit.md` 5.3.3, 14 |
| `Borrow` | `SlackTicks` | 0 | 0 to `MaxCreditTerm` | `spec-credit.md` 10.2, 14 |
| `Modules` | `Risk` | nil (off) | | `spec-risk.md` 12.1 |
| `Risk` | `Routes` | none | at most `MaxRiskRows`, one per key | `spec-risk.md` 3.2.1, 12.1 |
| `Risk` | `Stores` | none | at most `MaxRiskRows`, one per key | `spec-risk.md` 3.2.2, 12.1 |
| `RouteRisk` | `From` | required | a configured edge; neither end an outside node | `spec-risk.md` 3.2.1 |
| `RouteRisk` | `To` | required | a configured edge; neither end an outside node | `spec-risk.md` 3.2.1 |
| `RouteRisk` | `Mode` | required | an existing mode | `spec-risk.md` 3.2.1 |
| `RouteRisk` | `StrikeBP` | required | 1 to 10 000 | `spec-risk.md` 3.2, 3.4 |
| `RouteRisk` | `SeverityBP` | required | 1 to 10 000 | `spec-risk.md` 3.2, 3.4 |
| `RouteRisk` | `Common` | false | a bool | `spec-risk.md` 3.2, 3.3 |
| `StoreRisk` | `Region` | required | a region that is not an outside node | `spec-risk.md` 3.2.2 |
| `StoreRisk` | `Class` | required | an existing store class | `spec-risk.md` 3.2.2 |
| `StoreRisk` | `StrikeBP` | required | 1 to 10 000 | `spec-risk.md` 3.2, 3.4 |
| `StoreRisk` | `SeverityBP` | required | 1 to 10 000 | `spec-risk.md` 3.2, 3.4 |
| `StoreRisk` | `Common` | false | a bool | `spec-risk.md` 3.2, 3.3 |
| `Modules` | `Infrastructure` | nil (off) | | `spec-infrastructure.md` 12.1 |
| `Infrastructure` | `Classes` | empty (the module is then inert) | at most `MaxFacilityClasses` | `spec-infrastructure.md` 12.1 |
| `Infrastructure` | `Initial` | empty | at most `MaxInitialRows`, one row per key | `spec-infrastructure.md` 12.1, 3.4.2 |
| `FacilityClass` | `ID` | required | the ID pattern of 3.3 | `spec-infrastructure.md` 3.1 |
| `FacilityClass` | `Kind` | required | `site`, `route`, `ground` or `store` | `spec-infrastructure.md` 3.1, 12.2 |
| `FacilityClass` | `Durability` | required | 2 to `MaxDurability` | `spec-infrastructure.md` 12.2 |
| `FacilityClass` | `Decay` | 0 | 0 to `Durability - 1` | `spec-infrastructure.md` 12.2 |
| `FacilityClass` | `WearPerUse` | 0 (site) | 0 to `Durability` | `spec-infrastructure.md` 12.2 |
| `FacilityClass` | `Capacity` | required for a site | 1 to `MaxFacilityCapacity` | `spec-infrastructure.md` 12.2 |
| `FacilityClass` | `BoostBP` | required (route, ground) | 1 to `MaxBoostBP` | `spec-infrastructure.md` 12.2 |
| `FacilityClass` | `Modes` | required (route) | 1 to `MaxRouteModes`, each a mode of the transport module | `spec-infrastructure.md` 12.2 |
| `FacilityClass` | `Resource` | required (ground) | a resource of the resources module | `spec-infrastructure.md` 12.2 |
| `FacilityClass` | `Store` | required (store) | a store class of the storage module | `spec-infrastructure.md` 12.2 |
| `FacilityClass` | `Needs` | empty: none | a settlement flag of the settlements module | `spec-infrastructure.md` 8, 12.2 |
| `InitialFacility` | `Region` | required | a region; not an outside node | `spec-infrastructure.md` 3.4.2 |
| `InitialFacility` | `Class` | required | a facility class | `spec-infrastructure.md` 3.4.2 |
| `InitialFacility` | `Toward` | empty: every edge leaving `Region` (route only) | a region; not an outside node | `spec-infrastructure.md` 3.4.2 |
| `InitialFacility` | `Role` | empty: the region's estate | a role with a group in the region | `spec-infrastructure.md` 3.4.2 |
| `InitialFacility` | `Count` | required | 1 to `MaxInitialCount` | `spec-infrastructure.md` 3.4.2 |
| `Recipe` | `Needs` | none | site classes, each at most once, at most `MaxRecipeNeeds` | `spec-infrastructure.md` 4.1.1, 4.1.2 (the format is `spec-production.md` 3.1) |
| `Recipe` | `Builds` | nil | at most one of `Builds` and `Repairs` | `spec-infrastructure.md` 4.1.1, 4.1.2 (the format is `spec-production.md` 3.1) |
| `Recipe` | `Repairs` | nil | at most one of `Builds` and `Repairs` | `spec-infrastructure.md` 4.1.1, 4.1.2 (the format is `spec-production.md` 3.1) |
| `FacilityBuild` | `Class` | required | a facility class | `spec-infrastructure.md` 4.1.1 |
| `FacilityBuild` | `Toward` | empty: every edge leaving the region (route only) | a region of the configuration; not an outside node | `spec-infrastructure.md` 4.1.1 |
| `FacilityRepair` | `Class` | required | a facility class | `spec-infrastructure.md` 4.1.1 |
| `FacilityRepair` | `Units` | required | 1 to the class's `Durability` | `spec-infrastructure.md` 4.1.1 |
| `Modules` | `Insurance` | nil (off); needs `Modules.Risk` and `Modules.Credit` | | `spec-insurance.md` 4.1, 13 |
| `Insurance` | `ValueCapBP` | required | 1 to `MaxValueCapBP` | `spec-insurance.md` 6.2, 13 |
| `Insurance` | `PayoutTicks` | 1 | 1 to `MaxCreditTerm` | `spec-insurance.md` 8.4, 13 |
| `Insurance` | `MinSum` | required | 1 to `MaxMoney` | `spec-insurance.md` 7.4, 13 |
| `Insurance` | `MaxPolicies` | required | 1 to `MaxOpenPolicies` | `spec-insurance.md` 7.4, 13 |
| `Insurance` | `MaxCoverRequests` | 4 | 1 to `MaxCoverRequestsBound` | `spec-insurance.md` 5.2, 13 |
| `Role` | `Cover` | nil: the role does not deal in cover | exactly one of `Underwrite` and `Insure` | `spec-insurance.md` 4.2, 13 |
| `CoverTerms` | `Underwrite` | nil | one of the two blocks, not both | `spec-insurance.md` 4.2 |
| `CoverTerms` | `Insure` | nil | one of the two blocks, not both | `spec-insurance.md` 4.2 |
| `Underwrite` | `LoadingBP` | required | 0 to `MaxLoadingBound` | `spec-insurance.md` 6.4, 13 |
| `Underwrite` | `StoreRateBP` | required | 0 to `MaxStoreRateBound` | `spec-insurance.md` 6.4, 13 |
| `Underwrite` | `CoverBP` | required | 1 to 10 000 | `spec-insurance.md` 4.2, 13 |
| `Underwrite` | `CapBP` | required | 1 to `MaxCapBound` | `spec-insurance.md` 4.2, 13 |
| `Underwrite` | `ShareBP` | required | 1 to 10 000 | `spec-insurance.md` 10.3, 13 |
| `Insure` | `MinLossBP` | required (0 means no route cover) | 0 to 10 000 | `spec-insurance.md` 4.2, 10.2, 13 |
| `Insure` | `RuinShareBP` | required | 0 to `MaxRuinBound` | `spec-insurance.md` 10.2, 13 |
| `Insure` | `MaxLoadingBP` | required | 0 to `MaxLoadingBound` | `spec-insurance.md` 10.2, 13 |
| `Insure` | `MinCoverBP` | required | 1 to 10 000 | `spec-insurance.md` 10.2, 13 |
| `Insure` | `StoreShareBP` | 0 (buys none) | 0 to 10 000 | `spec-insurance.md` 10.2, 13 |
| `Insure` | `MaxStoreRateBP` | required | 0 to `MaxStoreRateBound` | `spec-insurance.md` 10.2, 13 |
| `Modules` | `Polity` | nil (off); needs no other module | | `spec-polity.md` 8 |
| `Polity` | `Jurisdictions` | none | at most `MaxJurisdictions` | `spec-polity.md` 3, 8 |
| `Polity` | `Members` | none | one row per region, at most `MaxRegions` | `spec-polity.md` 3.4, 8 |
| `Jurisdiction` | `ID` | required | the ID pattern of 3.3, unique | `spec-polity.md` 3.2, 8 |
| `Jurisdiction` | `Parent` | empty: a root | a defined jurisdiction other than itself, no cycle, fewer than `MaxPolityDepth` ancestors | `spec-polity.md` 3.3, 8 |
| `Jurisdiction` | `Seat` | empty: a grouping | a region it covers that is not an outside node | `spec-polity.md` 3.5, 8 |
| `Jurisdiction` | `StartMoney` | 0 | 0 to `MaxMoney`; only with a seat | `spec-polity.md` 3.5.4, 8 |
| `Member` | `Region` | required | a defined region | `spec-polity.md` 3.4, 8 |
| `Member` | `Jurisdiction` | required | a defined jurisdiction | `spec-polity.md` 3.4, 8 |
| `Modules` | `Fiscal` | nil (off); needs `Modules.Polity` with a governed jurisdiction | | `spec-fiscal.md` 3.1, 14 |
| `Fiscal` | `Taxes` | none | all six lists together at most `MaxFiscalFlows` | `spec-fiscal.md` 3.1, 5, 14 |
| `Fiscal` | `Duties` | none | all six lists together at most `MaxFiscalFlows` | `spec-fiscal.md` 3.1, 6, 14 |
| `Fiscal` | `Tolls` | none | all six lists together at most `MaxFiscalFlows` | `spec-fiscal.md` 3.1, 7, 14 |
| `Fiscal` | `Payouts` | none | all six lists together at most `MaxFiscalFlows` | `spec-fiscal.md` 3.1, 8, 14 |
| `Fiscal` | `Purchases` | none | all six lists together at most `MaxFiscalFlows` | `spec-fiscal.md` 3.1, 9, 14 |
| `Fiscal` | `Releases` | none | all six lists together at most `MaxFiscalFlows` | `spec-fiscal.md` 3.1, 9, 14 |
| `Tax`, `Duty`, `Toll`, `Payout`, `Purchase`, `Release` | `ID` | required | the ID pattern of 3.3, unique across the six kinds; not a built-in reason or a flow of the money-supply module | `spec-fiscal.md` 3.1.1, 3.4 |
| `Tax`, `Duty`, `Toll`, `Payout`, `Purchase`, `Release` | `Authority` | required | a governed jurisdiction | `spec-fiscal.md` 3.1 |
| `Window` | `Start` | 0 | `Until` 0 or above `Start` | `spec-fiscal.md` 3.1.2, 14 |
| `Window` | `Until` | 0 (never ends) | 0 or above `Start` | `spec-fiscal.md` 3.1.2, 14 |
| `Window` | `Seasons` | empty: every season | seasons of the calendar | `spec-fiscal.md` 3.1.2, 14 |
| `FlowTarget` | `Jurisdiction` | empty | a governed jurisdiction below the authority; kind `treasury` only | `spec-fiscal.md` 3.2, 14 (the type is `spec-money-supply.md` 4.1) |
| `FlowTarget` | `Kind` | as `spec-money-supply.md` 4.1 | the new values `stores` (the storage module) and `treasury` (the fiscal module) join the kinds of the money-supply module | `spec-money-supply.md` 4.1 (the field and the type); `spec-storage.md` 7.1.2, `spec-fiscal.md` 3.2 (the values) |
| `Tax` | `Target` | required | a known region, role and jurisdiction; `Region` covered | `spec-fiscal.md` 3.2, 5.1.2, 14 |
| `Tax` | `Amount` | 0 | 0 to `MaxFlowAmount` | `spec-fiscal.md` 5.1, 14 |
| `Tax` | `RateBP` | 0 | 0 to `MaxFlowRateBP` | `spec-fiscal.md` 5.1, 14 |
| `Tax` | `Exempt` | 0 | 0 to `MaxFlowAmount` | `spec-fiscal.md` 5.1, 14 |
| `Tax`, `Payout` | `Per` | `tick` | `tick` or `year` | `spec-fiscal.md` 5.1, 8.1, 14 |
| `Duty` | `Region` | empty: every region the authority covers | a covered region | `spec-fiscal.md` 6.1, 14 |
| `Duty` | `Commodity` | empty: every commodity | a known commodity | `spec-fiscal.md` 6.1, 14 |
| `Duty` | `Payer` | required | `seller` or `buyer` | `spec-fiscal.md` 6.1, 14 |
| `Duty` | `Scope` | `any` | `domestic`, `import`, `export` or `any` | `spec-fiscal.md` 6.1.1, 14 |
| `Duty` | `RateBP` | required | 1 to `MaxDutyBP` | `spec-fiscal.md` 6.1, 14 |
| `Toll` | `On` | required | `exit` or `enter` | `spec-fiscal.md` 7.1, 14 |
| `Toll` | `Mode` | empty: every mode | a configured mode | `spec-fiscal.md` 7.1, 14 |
| `Toll` | `Amount` | required | 1 to `MaxFlowAmount` | `spec-fiscal.md` 7.1, 14 |
| `Payout` | `Target` | required | as a tax's; `Role` and `Region` covered | `spec-fiscal.md` 3.2, 8.1, 14 |
| `Payout` | `Commodity` | empty: money | a known commodity | `spec-fiscal.md` 8.1, 14 |
| `Payout` | `Amount` | 0 | 0 to `MaxFlowAmount` | `spec-fiscal.md` 8.1, 14 |
| `Payout` | `Level` | 0 | 0 to `MaxFlowAmount`; money only | `spec-fiscal.md` 8.1, 14 |
| `Payout` | `NotIncome` | false | a bool; households only | `spec-fiscal.md` 8.1, 14 |
| `Purchase`, `Release` | `Commodity` | required | a known commodity | `spec-fiscal.md` 9.1, 14 |
| `Purchase`, `Release` | `Quantity` | required | 1 to `MaxQuantity` | `spec-fiscal.md` 9.1, 14 |
| `Purchase`, `Release` | `LimitBP` | required | 1 to `MaxLimitBP` | `spec-fiscal.md` 9.1, 9.2, 14 |
| `Purchase` | `Hold` | 0 (no cap) | 0 to `MaxQuantity` | `spec-fiscal.md` 9.1, 14 |
| `Release` | `Reserve` | 0 | 0 to `MaxQuantity` | `spec-fiscal.md` 9.1, 14 |
| `Modules` | `Institutions` | nil (off) | | `spec-institutions.md` 9 |
| `Institutions` | `Bodies` | none | at most `MaxBodies` | `spec-institutions.md` 9 |
| `Institutions` | `Rights` | none | at most `MaxRights` | `spec-institutions.md` 9 |
| `Institutions` | `Controls` | none | at most `MaxControls`, shared with `RateLimits` | `spec-institutions.md` 9 |
| `Institutions` | `RateLimits` | none | at most `MaxControls`, shared with `Controls` | `spec-institutions.md` 9 |
| `Institutions` | `OwnUseBound` | false | a bool | `spec-institutions.md` 5.5.2, 9 |
| `Body` | `ID` | required | the ID pattern of 3.3, unique | `spec-institutions.md` 3.2.2, 9 |
| `Body` | `Roles` | required | at least one, defined, no (role, region) in two bodies | `spec-institutions.md` 3, 9 |
| `Body` | `Regions` | empty | defined, not an outside node; with `Jurisdictions`, at least one region in all | `spec-institutions.md` 3, 9 |
| `Body` | `Jurisdictions` | empty | defined; needs `Modules.Polity` | `spec-institutions.md` 3, 9 |
| `Body` | `Places` | 0 (no limit) | 0 to `MaxHouseholds` | `spec-institutions.md` 3, 9 |
| `Body` | `MaxTrainees` | 0 (the role's alone) | 0 to 100 | `spec-institutions.md` 3, 9 |
| `Body` | `Needs` | empty | a defined settlement flag; needs `Modules.Settlements` | `spec-institutions.md` 3, 9 |
| `Right` | `ID` | required | as a body | `spec-institutions.md` 5, 9 |
| `Right` | `Body` | empty: a prohibition | a defined body | `spec-institutions.md` 5, 9 |
| `Right` | `Activity` | required | `produce`, `sell`, `buy` or `train` | `spec-institutions.md` 5, 9 |
| `Right` | `Subject` | required | defined, of the kind the activity names | `spec-institutions.md` 5, 9 |
| `Right` | `Region` | required (exactly one of `Region` and `Jurisdiction`) | a region that is not an outside node | `spec-institutions.md` 5, 9 |
| `Right` | `Jurisdiction` | required (exactly one of `Region` and `Jurisdiction`) | a defined jurisdiction; needs `Modules.Polity` | `spec-institutions.md` 5, 9 |
| `Right` | `Who` | `principal` | `principal`, `executor` or `both`; `produce` only | `spec-institutions.md` 5.1.2, 9 |
| `Control` | `ID` | required | as a body, one namespace with `RateLimit` | `spec-institutions.md` 6.1, 9 |
| `Control` | `Commodity` | required | a defined commodity | `spec-institutions.md` 6.1, 9 |
| `Control` | `Region` | required (exactly one of `Region` and `Jurisdiction`) | as a right | `spec-institutions.md` 6.1, 9 |
| `Control` | `Jurisdiction` | required (exactly one of `Region` and `Jurisdiction`) | as a right | `spec-institutions.md` 6.1, 9 |
| `Control` | `CapBP` | 0 (none) | 0 to `MaxBasisPoints` | `spec-institutions.md` 6.1, 9 |
| `Control` | `FloorBP` | 0 (none) | 0 to `CapBP` when a cap is set | `spec-institutions.md` 6.1, 9 |
| `RateLimit` | `ID` | required | as a control | `spec-institutions.md` 6.1, 9 |
| `RateLimit` | `Region` | required (exactly one of `Region` and `Jurisdiction`) | as a right; needs `Modules.Credit` | `spec-institutions.md` 6.1, 9 |
| `RateLimit` | `Jurisdiction` | required (exactly one of `Region` and `Jurisdiction`) | as a right; needs `Modules.Credit` | `spec-institutions.md` 6.1, 9 |
| `RateLimit` | `CapBP` | required | 0 to `MaxCreditRateBP`; 0 is interest-free | `spec-institutions.md` 6.1, 9 |
| `Modules` | `Policy` | nil (off); needs `Modules.Polity` with at least one jurisdiction | | `spec-policy.md` 3.1, 10 |
| `Policy` | `Actors` | none | at most `MaxActors` | `spec-policy.md` 3.1, 10 |
| `PolicyActor` | `ID` | required | the ID pattern of 3.3, unique | `spec-policy.md` 3.1, 10 |
| `PolicyActor` | `Jurisdiction` | required | a defined jurisdiction, at most one actor each | `spec-policy.md` 3.2, 10 |
| `PolicyActor` | `Policy` | required (a registered name) | a built-in or a name registered with `WithPolicy` | `spec-policy.md` 4.5, 10 |
| `PolicyActor` | `Every` | required | 1 to `MaxPolicyEvery` | `spec-policy.md` 7.2, 10 |
| `PolicyActor` | `ReadsRegions` | false | a bool | `spec-policy.md` 5.4, 10 |
| `PolicyActor` | `Levers` | none | at most `MaxLevers` | `spec-policy.md` 3.4, 10 |
| `PolicyActor` | `Schedule` | none | at most `MaxScheduleSteps`; only with `schedule` | `spec-policy.md` 8.2, 10 |
| `PolicyActor` | `Cycle` | 0 (no cycle) | 0, or 1 to `MaxPolicyCycle`; only with `schedule` | `spec-policy.md` 8.2, 10 |
| `PolicyActor` | `Rules` | none | one per lever; only with `rule` | `spec-policy.md` 8.3, 10 |
| `Lever` | `ID` | required | the ID pattern of 3.3, unique within the actor | `spec-policy.md` 3.4 |
| `Lever` | `Kind` | required | `flow`, `edge` or `transit` (a reserved kind is `ErrOutOfBounds`) | `spec-policy.md` 3.4, 6.3 |
| `Lever` | `Flow` | by kind | a defined flow, in reach | `spec-policy.md` 3.5, 6.3 |
| `Lever` | `From` | by kind | a defined region, in reach | `spec-policy.md` 3.5, 6.3 |
| `Lever` | `To` | by kind | a defined region, in reach | `spec-policy.md` 3.5, 6.3 |
| `Lever` | `Mode` | by kind | a defined mode | `spec-policy.md` 3.5, 6.3 |
| `Lever` | `Against` | by kind | a defined jurisdiction | `spec-policy.md` 3.4.3 |
| `Lever` | `Min` | required | inside the slot's bound; `Min <= Default <= Max` | `spec-policy.md` 3.4.1 |
| `Lever` | `Max` | required | inside the slot's bound; `Min <= Default <= Max` | `spec-policy.md` 3.4.1 |
| `Lever` | `MaxStep` | 0 (no limit) | 0, or 1 to the slot's bound | `spec-policy.md` 3.4.1, 6.2.2 |
| `Lever` | `MayClose` | false | kind `edge` only | `spec-policy.md` 3.4.1, 6.2.1 |
| `ScheduleStep` | `Lever` | required | a lever of the actor | `spec-policy.md` 8.2 |
| `ScheduleStep` | `At` | required | 0 to `MaxScheduleTick`; below `Cycle` when there is one | `spec-policy.md` 8.2 |
| `ScheduleStep` | `BP` | required | in `[Min, Max]` of the lever | `spec-policy.md` 8.2 |
| `ScheduleStep` | `Closed` | false | `MayClose` only | `spec-policy.md` 8.2 |
| `Reaction` | `Lever` | required | one per lever | `spec-policy.md` 8.3 |
| `Reaction` | `Gauge` | required | `treasury`, `unfed_share`, `mean_ratio` or `price` | `spec-policy.md` 8.3.2 |
| `Reaction` | `Commodity` | required for `price` | a defined commodity; empty otherwise | `spec-policy.md` 8.3.2 |
| `Reaction` | `MaxAge` | required for `price` | 0 to `MaxPolicyEvery`; 0 otherwise | `spec-policy.md` 8.3.2 |
| `Reaction` | `Target` | required | 0 to `MaxMoney` | `spec-policy.md` 8.3.1 |
| `Reaction` | `Deadband` | 0 | 0 to `MaxMoney` | `spec-policy.md` 8.3.1, 8.3.6 |
| `Reaction` | `Span` | required | 1 to `MaxMoney` | `spec-policy.md` 8.3.1 |
| `Reaction` | `Move` | required | 0 to `MaxMoveBP` | `spec-policy.md` 8.3.1 |
| `Reaction` | `Raise` | false | a bool | `spec-policy.md` 8.3.1 |
| `Modules` | `Saving` | nil (off) | | `spec-saving-and-investment.md` 3.2, 10 |
| `Saving` | `SafetyTicks` | 2 | 0 to `MaxSafetyTicks` | `spec-saving-and-investment.md` 5.6.1, 10.2 |
| `Saving` | `HurdleSpreadBP` | 20 000 | 0 to `MaxHurdleSpreadBP` | `spec-saving-and-investment.md` 5.5.2, 10.2 |
| `Saving` | `CommitBandBP` | 2 000 | 0 to `MaxCommitBandBP` | `spec-saving-and-investment.md` 5.5.3, 10.2 |
| `Role` | `Saver` | nil: the role has no saving behaviour | present or nil | `spec-saving-and-investment.md` 3.2, 10 |
| `Saver` | `PrecautionShareBP` | 0 (no buffer) | 0 to `MaxPrecautionShareBP` | `spec-saving-and-investment.md` 4.2, 10.2 |
| `Saver` | `PrecautionCapTicks` | 30 | 0 to `MaxPrecautionCapTicks` | `spec-saving-and-investment.md` 4.2, 10.2 |
| `Saver` | `HurdleBP` | 15 000 | 10 000 to `MaxHurdleBP` | `spec-saving-and-investment.md` 5.5.1, 10.2 |
| `Saver` | `PaybackTicks` | 180 | 1 to `MaxPaybackTicks` | `spec-saving-and-investment.md` 5.3, 10.2 |
| `Saver` | `RepairBelowBP` | 5 000 | 0 to 10 000; 0 never repairs | `spec-saving-and-investment.md` 5.8, 10.2 |
| `Saver` | `Worth` | none | at most `MaxWorthRows`, each class once | `spec-saving-and-investment.md` 5.3.2, 10.2 |
| `ProjectWorth` | `Class` | required | a facility class | `spec-saving-and-investment.md` 3.2, 10.2 |
| `ProjectWorth` | `PerTick` | required | 1 to `MaxWorthPerTick` | `spec-saving-and-investment.md` 3.2, 10.2 |
| `ProjectWorth` | `Count` | required | 1 to `MaxWorthCount` | `spec-saving-and-investment.md` 3.2, 10.2 |
| `Modules` | `Technology` | nil (off) | | `spec-technology.md` 9.1 |
| `Technology` | `Fields` | empty (the module is then inert) | at most `MaxTechFields` | `spec-technology.md` 9.1 |
| `TechField` | `ID` | required | the ID pattern of 3.3 | `spec-technology.md` 3.1 |
| `TechField` | `Recipes` | required | 1 to `MaxFieldRecipes`, each a recipe, none in another field | `spec-technology.md` 3.1, 9.2 |
| `TechField` | `CapBP` | required | 1 to `MaxGainBP` | `spec-technology.md` 9.2 |
| `TechField` | `Base` | required with `DoublingBP` above 0, else 0 | 1 to `MaxTechBase` | `spec-technology.md` 9.2 |
| `TechField` | `DoublingBP` | 0 (off) | 0 to `MaxDoublingBP` | `spec-technology.md` 9.2 |
| `TechField` | `TrendBP` | 0 (off) | 0 to `MaxTechTrendBP` | `spec-technology.md` 9.2 |
| `TechField` | `DiscoveryBP` | 0 (off) | 0 to `MaxDiscoveryBP` | `spec-technology.md` 9.2 |
| `TechField` | `StepBP` | required with `DiscoveryBP` above 0, else 0 | 1 to `MaxStepBP` | `spec-technology.md` 9.2 |
| `TechField` | `CatchUpBP` | 0 (off) | 0 to `MaxCatchUpBP` | `spec-technology.md` 9.2 |
| `TechField` | `Reach` | required with `CatchUpBP` above 0, else 0 | 1 to `MaxDistance` | `spec-technology.md` 9.2 |

Where the owner spec names the Go type of a module differently from the scope of its row (`TenureConfig` in `spec-land.md` 3.3, `PolicyConfig` in `spec-policy.md` 3.1.2, which cannot be `Policy`: that name is the extension interface of architecture.md 4.3), the scope here is the module's field name in `Modules`, as the owner tables use it.

### 3.7.2 The slice at Stage 0 (decision D3 of the plan of Stage 0)

The first code declares the fields that the specs give the **core**, which is every
field that no module of `Config.Modules` owns, and nothing else: a module's pull
request appends the fields it owns, to `Modules` and to the core structs, and never
inserts one (architecture.md 2.10.5). The core includes the six presentation-only fields of the allow-list (8.3) and every key that the packs
`one` and `second` of `testdata/chains/` set: `sited`, `trainee_eligible`,
`trainee_yield`, `cost_share`, `discard`, `wear_per_use`, `unskilled`, `closed`,
`trades`, `raw`, `classes`, `shelf_life`, `durability`, `seasons`, `season_start`, the
yield, group and warm-start tables. Skill barriers (`TrainingTicks`, `MaxTrainees`,
`TeachingLoad`, `TraineeEligible`, `TraineeYield`) are core: there is no module for them
and `TrainingTicks` 0 is "no traineeship". The Go types are as follows; a field of a
named type is a count, a tick or a basis point unless the table says `Money`, `Price` or
`Quantity` (`value.go`, architecture.md 3.3). Fields are declared in the order given,
which is also the order of their ids in the hash (8.2.2). A field marked (p) is
presentation only (8.3) and has no id.

| Struct | Fields, in order | Not in the slice, and why |
|--------|------------------|---------------------------|
| `Config` | `World WorldParams`, `Calendar Calendar`, `Commodities []Commodity`, `Recipes []Recipe`, `Yields []YieldProfile`, `Roles []Role`, `Regions []Region`, `Groups []Group`, `WarmStart *WarmStart`, `Modules Modules` | `Edges []Edge` and `RegionPrices []RegionPrice` (Stage 2, REG; their ids 8 and 9 are reserved) |
| `WorldParams` | `Scale`, `MinAgents`, `BulkFactor`, `WelfareWindow`, `WindowCycles`, `Upkeep []Item`, `CoverTicks`, `InputJobs`, `OutputCoverTicks`, `FavourFloor`, `ReserveTicks`, `SpendMaxShare`, `SpendHalfTicks`, `RestBand`, `ProduceHaltRatio`, `BasketTolerance`, `BeliefStep`, `BeliefAdjustProb`, `BeliefSignificance`, `BeliefStockLow`, `BeliefStockHigh`, `BeliefMaxAge`, `BeliefWidthInit`, `BeliefWidthMin`, `BeliefWidthMax`, `BeliefWidthStep`, `BeliefEvidenceShare`, `CostFloor`, `CostCeiling`, `RefBandLow`, `RefBandHigh`, `PriceWindow`, `Spawner string`, `ReviewTicks`, `EntryShare`, `RoleHysteresis`, `RoleCooldown`, `EntryRatio`, `ChangePrivation`, `RemovePrivation`, `DeathStep`, `ExitShare`, `Renewal`, `EntryPoolShare`, `Clearing string` (45 fields with an id), `DistanceLabel string` (p) | |
| `Calendar` | `HoursPerDay`, `DaysPerWeek`, `Months []Month`, `TicksPerDay`, `DaysPerTick`, `StartDay`, `Seasons []Season`, `SeasonStart` | |
| `Month`, `Season` | `ID string`, `Days` | |
| `Commodity` | `ID string`, `Mass`, `Bulk`, `ShelfLife`, `Durability`, `Classes []string`, `Raw bool`, `ReferencePrice Price`, `Label string` (p), `UnitLabel string` (p) | |
| `Item` | `Commodity string`, `Quantity Quantity` | |
| `Output` | `Commodity string`, `Quantity Quantity`, `CostShare`, `Discard bool` | |
| `Tool` | `Commodity string`, `WearPerUse` | |
| `Recipe` | `ID string`, `Duration`, `Inputs []Item`, `Outputs []Output`, `Requires []Tool`, `Sited bool`, `TraineeEligible bool`, `TraineeYield`, `Label string` (p) | `Draws` (resources), `Needs`, `Builds`, `Repairs` (infrastructure) |
| `YieldProfile` | `Region string`, `Recipe string`, `HasAll bool`, `All`, `Seasons []SeasonYield` | |
| `SeasonYield` | `Season string`, `BP` | |
| `Role` | `ID string`, `Recipes []string`, `Upkeep []Item`, `Trades []string`, `StartMoney Money`, `StartStock []Item`, `Strategy string`, `Unskilled bool`, `TargetRatio`, `TargetSpread`, `DefaultRatio`, `InKind`, `ExpectedWorkTicks`, `MinWorkShare`, `TrainingTicks`, `MaxTrainees`, `TeachingLoad`, `Closed bool`, `Label string` (p) | `Informed` (information), `Fixed` (migration), `Merchant` and `MerchantFeeShare` (merchants), `Needs` and `Ceiling` (settlements), `Hireable` and `Employer` (employment), `Credit`, `Cover`, `Saver` |
| `Region` | `ID string`, `ReferenceBasket Money`, `ExtraUpkeep []Item`, `CapitalReturn`, `EstateMarkdown`, `Label string` (p) | `PoolRelease`, `ReliefBelow` (estates, 6.2.4) |
| `Group` | `Region string`, `Role string`, `Households`, `Scale` (0: the world's `Scale`, FR-SCL-006) | |
| `WarmStart` | `Prices []SeedPrice`, `Wages []SeedWage`, `Groups []WarmGroup`, `Pools []WarmPool` | |
| `SeedPrice` | `Region string`, `Commodity string`, `Price Price` | |
| `SeedWage` | `Region string`, `Wage Money` | |
| `WarmGroup` | `Region string`, `Role string`, `Money Money`, `Stock []Item`, `Jobs []JobSpec` | |
| `JobSpec` | `Recipe string`, `InProgressBP` | |
| `WarmPool` | `Region string`, `Money Money` | |
| `Modules` | no field | all 25 pointers, as 3.1 lists them |

Entities refer to each other by ID string; the typed IDs (`CommodityID`, `RoleID`,
`RegionID`, `RecipeID`) are assigned when a world is built (3.3). The types of
`WarmStart` are declared in `config.go` with `Config` (architecture.md 2.6: `config.go`
holds "Config and every section struct"), not in `viability.go`; `viability.go` keeps
`SolveSteadyState` and the code that reads the table (`spec-viability.md` 21).
`Item`, `Output` and `Tool` are the types of every list of (commodity, quantity)
(`spec-production.md` 3.1, 3.2); no struct embeds another (architecture.md 4.7), so
`Output` repeats the two fields of `Item`.

## 4. Constants and bounds

Architecture.md 9.6 names three kinds of constant. For this spec:

- **Tunables** are fields of `Config`. Each has a row in the parameter-owner
  table, a default (section 5), a validation bound and an owner spec.
- **Bounds** are constants in `bounds.go` in the root package. They are not
  configurable: they exist so that arithmetic cannot overflow (architecture.md
  3.5) and so that validation work is bounded.
- **Rule constants** stay beside the code they define (the half-to-even rounding
  of `spec-clearing.md` section 5.2, the rejection `reason` strings).

### 4.1 The bounds

All are provisional until the slice runs. The money and quantity bounds are
derived, not chosen: they are the largest values for which the products the
simulation computes fit in `int64` (maximum `9 223 372 036 854 775 807`, about
`9.2e18`).

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxPrice` | 10^9 minor units | with `MaxQuantity`, a trade fits: `10^9 * 10^9 = 10^18 < 9.2e18` |
| `MaxQuantity` | 10^9 units | the most of one commodity in one order, stock, recipe amount or upkeep |
| `MaxMoney` | 10^18 minor units | any balance, any single transfer, and the total money at start; equal to the largest trade, so one trade never overflows |
| `MaxWeight` | 10^6 households per agent | a weight times a per-household quantity must still be at most `MaxQuantity` (`spec-scale.md` 4.5) |
| `MaxHouseholds` | 10^9 per group | bounded by `MaxQuantity` so that total upkeep fits |
| `MaxBasisPoints` | 10^6 (a factor of 100) | ratio fields; a field with a tighter range (a probability is 0 to 10 000) says so in its owner spec |
| `MaxCommodities`, `MaxRoles`, `MaxRegions` | 1 000 each | validation and per-tick loops are bounded by the configuration |
| `MaxRecipes` | 10 000 | |
| `MaxGroups` | 100 000 | |
| `MaxYields` | `MaxRegions * MaxRecipes` = 10 000 000 | one yield profile per region and recipe; derived, so the yield table has a bounded size (FR-CFG-101); provisional, unmeasured |
| `MaxRecipeItems` | 16 per side of a recipe | |
| `MaxIDLength` | 64 bytes | |
| `MaxTicksPerYear` | 100 000 | `TicksPerYear` (derived from the calendar, `spec-calendar.md` 3.2) multiplies money and basis points in the calibration formulas; the bound keeps the checked 128-bit helper far from its limit and still allows a tick of a few minutes |
| `MaxHoursPerDay`, `MaxDaysPerWeek`, `MaxMonths`, `MaxDaysPerMonth`, `MaxDaysPerYear`, `MaxSeasons`, `MaxTicksPerDay` | 1 000, 100, 100, 10 000, 100 000, 100, 1 000 | the calendar; the reasons are in `spec-calendar.md` 7.1 |
| `MaxRecipeTicks` | 100 000 | a job's duration; equal to `MaxTicksPerYear`, so a job of a year is possible, and a duration times a count of jobs does not overflow (`spec-production.md` 10.1) |
| `MaxYieldBP` | 100 000 | a yield factor of ten times; the largest weight times a quantity times this over 10 000 must still fit `MaxQuantity` (4.2) |
| `MaxRequires`, `MaxDraws` | 4 each | tools and endowment draws of one recipe; they bound the per-execution work of production |
| `MaxUpkeepItems` | 16 | goods in one upkeep list (`spec-agents.md` 12). Each quantity is at most `MaxQuantity / MaxWeight` = 1 000, and a price at most `MaxPrice`, so the cost of a list is at most `16 * 1 000 * 10^9 = 1.6 * 10^13`, far below `MaxMoney` |
| `MaxWelfareWindow` | 3 650 | the longest effective welfare window of a role, ten Earth years of daily ticks (`spec-welfare-and-wages.md` 2.3.4); the window is the length of a per-agent record, so it bounds memory and the window-times-income product. Provisional |
| `MaxViabilityPivots` | 100 000 | the pivots the load-time viability check and a solve may use (`spec-viability.md` 11); a cap on work, since Bland's rule terminates but may be slow |
| `MaxViabilityCells` | 4 000 000 | rows times columns of the tableau `Validate` builds; above it the programmes of V2 and V3 are skipped with a warning (`spec-viability.md` 4.3). Chosen so that the exact arithmetic stays inside `NFR-CFG-001`; provisional, unmeasured |
| `MaxLanes` | `MaxRegions * MaxRegions * MaxCommodities` | the longest lane list of a solve (`spec-viability.md` 9.4, 11): one lane per ordered pair of regions and commodity. It bounds a solve option, not a `Config` field, and is kept here because bounds live in `bounds.go`. It is added with the transport module, which defines `Lane` and `WithLanes`; Stage 1 solves only the free-trade lane set and defines no `MaxLanes` (`spec-viability.md` 9.4) |
| `MaxWants` | 1 000 | want entities in the demand module (as `MaxCommodities`) |
| `MaxWantItems` | 4 | alternatives of one want; the loop over alternatives and the carry state |
| `MaxWantsPerRole` | 32 | taste rows of one role: the per-agent loop and the carry state |
| `MaxWantPeriod` | 100 000 | ticks of a want's period; equal to `MaxTicksPerYear` |
| `MaxWantRank` | 1 000 | ladder position |
| `MaxElasticityBP` | 100 000 | a slope of ten |
| `MaxCeilingBP` | 50 000 | five times the base quantity |
| `MaxSubstitution` | 4 | the integer exponent of the substitution shares; the products go through the 128-bit helper |
| `MaxTasteScale` | 100 000 | ten times (a role's or a region's multiplier) |
| `MaxSeedPrices` | `MaxRegions * MaxCommodities` | one seed price per region and commodity; derived, so a warm-start table has a bounded size (`spec-viability.md` 8.1) |
| `MaxDistance` | 10^9 | one edge of the region graph; a shortest path of at most `MaxRegions - 1` effective edges of at most `MaxDistance * MaxEdgeFactorBP / 10 000` stays below 10^13 (`spec-regions.md` 4.2) |
| `MaxEdges` | `20 * MaxRegions` | bounds the graph construction work |
| `MaxEdgeFactorBP` | 100 000 | the largest length factor of an edge status: ten times |
| `MaxRegionPrices` | `MaxRegions * MaxCommodities` | one override per region and commodity; derived, as `MaxSeedPrices` is |
| `MaxLabelBytes` | 256 | a label, for every presentation-only text field, `UnitLabel` included: one bound, in bytes, checked with `ErrOutOfBounds` (FR-GDS-021 says the same) |
| `MaxMoneyFlows` | 64 | flows of the money-supply module; the divisor of `MaxFlowTotal` (`spec-money-supply.md` 11.2) |
| `MaxFlowAmount` | `MaxMoney / MaxWeight` = 10^12 | one household's due fits `MaxMoney` |
| `MaxFlowRateBP` | 10 000 | at most the whole balance above the exemption per period |
| `MaxFlowScale` | 100 000 | ten times the configured value; the numerator reaches 10^23, inside 128 bits |
| `MaxFlowTotal` | `MaxMoney / MaxMoneyFlows` | all flows together create at most `MaxMoney` |
| `MaxKnots` | 8 | knots in one schedule of the population module (`spec-population.md` 8) and in one abundance curve of a resource (`spec-resources.md` 4.7); a knot ratio is 0 to `MaxBasisPoints` and a `Rate` 0 to 10 000, the largest probability per year, which is not a tunable |
| `MaxModes` | 64 | the loop over modes per request and per view (`spec-transport.md` 11) |
| `MaxSpeed` | 10^9 | with `MaxTransitFactorBP` the product `Speed * F` fits `int64` |
| `MaxCapacity` | 10^12 | per carrying unit; times `MaxWeight` it is 10^18 |
| `MaxFeed` | 4 | feed entries of a mode, as `MaxRequires` |
| `MaxHaulSpan` | 10^12 | distance one haul unit carries one carrying unit |
| `MaxShipmentLines` | 16 | lines of a request and the size of an event |
| `MaxDispatchPerTick` | 8 | requests an agent's plan may carry |
| `MaxTravelTicks` | `MaxTicksPerYear` | a journey of a year of ticks |
| `MaxTransitFactorBP` | 100 000 | a road ten times faster |
| `MaxResources` | 256 | resources of the module; with `MaxRegions` it bounds the dense endowment array (`spec-resources.md` 4.6) |
| `MaxEndowments` | `MaxRegions * MaxResources` | rows of the endowment table; derived |
| `MaxEndowment` | 10^15 | a capacity, a starting stock, an adjustment of an endowment; larger than `MaxQuantity` because a stock in the ground can exceed any holding |
| `MaxRegenRatePerYear` | 10^15 | the rate of a `constant` stock |
| `MaxRegenBP` | 100 000 | the annual rate of a `refill` or `logistic` stock: ten times a year |
| `MaxRegenScaleBP` | 100 000 | a row's regeneration scale |
| `MaxSeasonRegenBP` | 100 000 | a season's regeneration factor |
| `MaxShockBP` | 100 000 | a shock scale of an endowment (`spec-resources.md` 6.3) and the outside price shock (`spec-external-sector.md` 6.5): ten times |
| `MaxHorizonYears` | 1 000 | the viability horizon of a finite stock (`spec-resources.md` 11) |
| `MaxInfoSpeed` | 10^9 | as `MaxSpeed`; `D + InfoSpeed` stays far below `int64` (`spec-information.md` 5.2.3) |
| `MaxRumourErrorBP` | 5 000 | a rumoured price is never 0 or negative (`spec-information.md` 5.5.1) |
| `MaxRumourAge` | 10 000 | a tenth of `MaxTicksPerYear`; bounds the depth of the price record |
| `MaxRumourCells` | 4 000 000 | sources times `RumourMaxAge`, the entries of the price record |
| `MaxRumoursPerView` | 4 096 | the candidates of one informed agent: bounds the loop of a view and its draws |
| `MaxOutsideNodes` | 16 | outside nodes of the external module; also a divisor of `MaxLineTotal` |
| `MaxOutsideLines` | 64 | lines of one outside node (a placeholder, well below `MaxCommodities`) |
| `MaxCounterparties` | 256 | counterparties of the external module: agent IDs, accounts and boundary accounts are bounded |
| `MaxSellMarginBP` | 100 000 | an ask of at most eleven times the mid (`spec-external-sector.md` 6.6) |
| `MaxBuyMarginBP` | 9 999 | a bid is at least a basis point of the mid |
| `MinQuoteSpreadBP` | 1 | the smallest sum of the two margins of a line with both sides (`ErrCrossedQuotes`) |
| `MaxPriceFactorBP` | 100 000 | a seasonal price factor of an outside line |
| `MaxTrendBP` | 100 000 | the trend of an outside line, ten times `Base` a year |
| `MaxFactorBP` | 1 000 000 | the largest `CeilingBP` of an outside line |
| `MaxWalkStepBP` | 5 000 | the largest step of an outside price walk |
| `MaxDriftBP` | 9 999 | keeps the walk factor positive |
| `MaxGateScaleBP` | 100 000 | the scale of an outside gate |
| `MaxLineTotal` | `MaxMoney / (2 * MaxOutsideNodes * 64)` = 488 281 250 000 000 | all sides of the external module together can move at most `MaxMoney` of list value, so the money supply stays below `3 * MaxMoney` (`spec-external-sector.md` 13.2) |
| `MaxOwnUseRecipes` | 8 | entries in one own-use list; the default `Work` scans the list of a free household once a tick (`spec-subsistence.md` 13.1) |
| `MaxOwnUseTicks` | 1 000 | `EnterTicks`, `ExitTicks` and the saturation of `Secure`; the same bound as `CoverTicks` (`spec-subsistence.md` 13.1) |
| `MaxRetryJobs` | 1 000 | `RetryJobs`; the count in an `int64` agent field (`spec-subsistence.md` 13.1) |
| `MaxLotUnits` | 64 | a merchant's lot cap in carrying units; with `MaxQuantity` and `MaxPrice` a lot's product fits `int64` through the 128-bit helper (`spec-merchants.md` 13) |
| `MaxContracts` | 1 000 000 | delivery contracts in flight, so Escrow accounts (`spec-merchants.md` 13) |
| `MaxTimeCostBP` | 100 000 | ten times the upkeep for each tick away (`spec-merchants.md` 13) |
| `MaxRoom` | 10^12 per dimension | the mass room or the bulk room of a base room or of a store class; with `MaxWeight` it is 10^18, and the largest room of an account (a base room of a cohort and `MaxStoresPerAgent` stores) is below `int64` (`spec-storage.md` 12.2) |
| `MaxStoresPerAgent` | 8 | stores one agent may own (`spec-storage.md` 12.2) |
| `MaxStoreClasses` | 64 | store classes in a pack (`spec-storage.md` 12.2) |
| `MaxEstablishment` | 1 000 | hired executions in progress at once of one employer; with `MaxQuantity` and `MaxPrice` the product of a bill fits `int64` through the 128-bit helper (`spec-employment.md` 13) |
| `MaxHireRecords` | 1 000 000 | the hires open at once in a region; the Escrow accounts are at most this many (as `MaxContracts`) (`spec-employment.md` 13) |
| `MaxSettlementClasses` | 16 | size classes of the settlements module; a ranking of settlements is short, and each region holds one threshold per class (`spec-settlements.md` 8.1) |
| `MaxClassFlags` | 64 | distinct flags in the module, so that a class's flags fit one `uint64` (`spec-settlements.md` 8.1) |
| `MaxClassLimits` | 64 | distinct limits in the module (`spec-settlements.md` 8.1) |
| `MaxBandBP` | 5 000 | the band below an enter threshold: a leave threshold is at least half the enter threshold (`spec-settlements.md` 8.1) |
| `MaxThresholdBP` | 100 000 | a region's scale of the enter thresholds: ten times, as `MaxYieldBP` (`spec-settlements.md` 8.1) |
| `MaxRentStepBP` | 10 000 | the step of a posted rent each tick, basis points of the configured rate (`spec-land.md` 10) |
| `MaxRentCeilingBP` | 100 000 | the highest posted rent, ten times the configured rate (`spec-land.md` 10) |
| `MaxHoldings` | 4 096 | initial titles (rows) of the tenure module (`spec-land.md` 10) |
| `MaxTitleTransfers` | 1 024 | entries of one `TransferTitles` call (`spec-land.md` 10) |
| `MaxCreditRateBP` | 100 000 | ten times the principal a year: a rate above it is a typing error, and with `MaxCreditTerm` and `MaxMoney` the interest product fits 128 bits (`spec-credit.md` 14) |
| `MaxCreditTerm` | 3 650 | ten Earth years of daily ticks, the order of `MaxWelfareWindow`; a term above it is a different instrument (`spec-credit.md` 14). `spec-insurance.md` reuses it for `PayoutTicks` |
| `MaxOpenClaims` | 1 000 000 | the claims in the list at once; with `MaxMoney` the owed over all claims is at most the money in existence (`spec-credit.md` 14) |
| `MaxRiskRows` | 4 096 | rows in each of the two tables of the risk module (`spec-risk.md` 12.2) |
| `MaxFacilityClasses` | 64 | facility classes, authored by hand (`spec-infrastructure.md` 12.2) |
| `MaxDurability` | 1 000 000 | condition units of a new facility, a unit being the author's; a product with a capacity or a boost fits (`spec-infrastructure.md` 12.2) |
| `MaxFacilityCapacity` | 100 000 | executions a whole site facility serves in a start; at most the largest weight of a group is useful (`spec-infrastructure.md` 12.2) |
| `MaxBoostBP` | 100 000 | the gain of a whole route or ground facility: at most elevenfold (`spec-infrastructure.md` 12.2) |
| `MaxTotalBoostBP` | 200 000 | the cap of a sum of boosts (`spec-infrastructure.md` 12.2) |
| `MaxRouteModes` | 8 | transport modes a route class speeds (`spec-infrastructure.md` 12.2) |
| `MaxRecipeNeeds` | 2 | sites a recipe needs (`spec-infrastructure.md` 12.2) |
| `MaxFacilities` | 100 000 | the world's list of facilities, a scan a tick (`spec-infrastructure.md` 12.2) |
| `MaxFacilitiesPerAgent` | 64 | facilities one agent holds; the view carries them (`spec-infrastructure.md` 12.2) |
| `MaxInitialCount`, `MaxInitialRows` | 1 000, 1 024 | the count of an initial row and the rows, so that construction is bounded (`spec-infrastructure.md` 12.2) |
| `MaxValueCapBP` | 100 000 | ten times the reference price: a larger cap is a typing error, and the product with `MaxPrice` fits `int64` (`spec-insurance.md` 13) |
| `MaxLoadingBound` | 100 000 | ten times the fair premium; with `MaxMoney`, 10 000 and 10 000 the premium product fits 128 bits (`spec-insurance.md` 13) |
| `MaxStoreRateBound` | 10 000 | a premium of the whole sum insured for one tick (`spec-insurance.md` 13) |
| `MaxCapBound` | 100 000 | ten times the underwriter's money (`spec-insurance.md` 13) |
| `MaxRuinBound` | 100 000 | ten times the household's money (`spec-insurance.md` 13) |
| `MaxOpenPolicies` | 1 000 000 | the policies in the list at once (`spec-insurance.md` 13) |
| `MaxCoverRequestsBound` | 64 | the loop of a plan; with `MaxShipmentLines` the size of a round (`spec-insurance.md` 13) |
| `MaxJurisdictions` | 1 000 | jurisdictions, the value of `MaxRegions`: a jurisdiction that covers no region draws a warning (`spec-polity.md` 8) |
| `MaxPolityDepth` | 8 | levels of a chain of jurisdictions: a chain of at most 8 entries, a depth of at most 7 (`spec-polity.md` 8) |
| `MaxFiscalFlows` | 64 | fiscal flows of all six kinds together, as `MaxMoneyFlows` (`spec-fiscal.md` 11.4) |
| `MaxDutyBP` | 10 000 | the configured rate of a duty, the whole value of a trade at most at a scale of 10 000 (`spec-fiscal.md` 14) |
| `MaxLimitBP` | 100 000 | the limit of a purchase or a release as a share of the reference price: ten times (`spec-fiscal.md` 14) |
| `MaxBodies` | 256 | bodies of the institutions module (`spec-institutions.md` 9) |
| `MaxRights` | 4 096 | rights (`spec-institutions.md` 9) |
| `MaxControls` | 4 096 | price controls and rate limits together (`spec-institutions.md` 9) |
| `MaxActors` | 64 | policy actors: at most one per jurisdiction (`spec-policy.md` 10) |
| `MaxLevers` | 32 | levers of one actor (`spec-policy.md` 10) |
| `MaxScheduleSteps` | 256 | steps of one actor's schedule (`spec-policy.md` 10) |
| `MaxPolicyEvery` | 3 650 | ticks between two decisions: ten years of a 365-tick year (`spec-policy.md` 10) |
| `MaxPolicyCycle` | 3 650 | the length of a repeating schedule (`spec-policy.md` 10) |
| `MaxScheduleTick` | 1 000 000 000 | the largest `At` of a schedule step (`spec-policy.md` 10) |
| `MaxLeverKeys` | 256 | the keys that one `Against` lever expands to (`spec-policy.md` 10) |
| `MaxMoveBP` | 100 000 | the `Move` of a reaction, basis points of a lever value per `Span` (`spec-policy.md` 10) |
| `MaxSafetyTicks` | 100 | a margin of ticks of upkeep, not a plan (`spec-saving-and-investment.md` 10.2) |
| `MaxHurdleSpreadBP` | 100 000 | the spread of the hurdle by agent: the width of `MaxHurdleBP` (`spec-saving-and-investment.md` 10.2) |
| `MaxCommitBandBP` | 5 000 | the fall of the hurdle for a household that holds half its inputs: at most half the hurdle (`spec-saving-and-investment.md` 10.2) |
| `MaxPrecautionShareBP` | 20 000 | the share of the drawdown held as a buffer: up to twice the worst hole (`spec-saving-and-investment.md` 10.2) |
| `MaxPrecautionCapTicks` | 1 000 | the buffer in ticks of upkeep cost, below the largest effective window (`spec-saving-and-investment.md` 10.2) |
| `MaxHurdleBP` | 100 000 | a return of up to ten times the cost (`spec-saving-and-investment.md` 10.2) |
| `MaxPaybackTicks` | 36 500 | a hundred years of a 365-tick year (`spec-saving-and-investment.md` 10.2) |
| `MaxWorthRows` | 16 | `Worth` rows of one role (`spec-saving-and-investment.md` 10.2) |
| `MaxWorthPerTick` | `MaxMoney / MaxPaybackTicks` | so that `PerTick * T_p` stays within `MaxMoney`; derived (`spec-saving-and-investment.md` 10.2) |
| `MaxWorthCount` | 16 | facilities of a class per household up to which a `Worth` row counts (`spec-saving-and-investment.md` 10.2) |
| `MaxTechFields` | 64 | technology fields, authored by hand (`spec-technology.md` 9.2) |
| `MaxFieldRecipes` | 64 | recipes in one field (`spec-technology.md` 9.2) |
| `MaxTechBase` | 10^12 | an author's count of executions behind a field; `X + Base` fits `int64` with `MaxExperience` (`spec-technology.md` 9.2) |
| `MaxDoublingBP` | 10 000 | the gain per doubling of the cumulative work (`spec-technology.md` 9.2) |
| `MaxTechTrendBP` | 10 000 | the gain a year adds at most: the table's yield again. Not `MaxTrendBP`, which is the trend of an outside line (above, 100 000) (`spec-technology.md` 9.2) |
| `MaxDiscoveryBP` | 100 000 | ten discoveries a year (`spec-technology.md` 9.2) |
| `MaxGainBP` | 100 000 | the largest gain of a cell: a yield at most elevenfold (`spec-technology.md` 9.2) |
| `MaxStepBP` | `MaxGainBP` | the gain of one discovery is at most the cap (`spec-technology.md` 9.2) |
| `MaxCatchUpBP` | 10 000 | the share of the gap to the frontier closed in a year: at most the whole gap (`spec-technology.md` 9.2) |
| `MaxTechBP` | `10 000 + MaxGainBP` | the largest `tech` factor (`spec-technology.md` 9.2) |
| `MaxExperience` | 2^50 | the cumulative executions of a cell, unreachable (`spec-technology.md` 9.2, 4.2.4) |
| `MaxDocumentBytes` | 4 MiB | one document: a limit of the loader, checked before anything is decoded (6.3.1; ADR 0009 has the measurements; it was 16 MiB before them) |
| `MaxDocuments` | 16 | the documents of one `Parse` or `LoadFS` (6.3.1) |
| `MaxMergedBytes` | 16 MiB | the sum of the sizes of the documents of one call (6.3.1) |
| `MaxLineBytes` | 256 KiB | one line of a document: it bounds what the guard pass of 6.3.1 holds at once |
| `MaxNestingDepth` | 16 | segments of a dotted key or a table header, and nesting of arrays and inline tables in a value (6.3.1); the deepest key of a pack has 8 segments (`modules.external.node.<r>.line.<c>.sell.margin_bp`) and a value adds 2 (`window = [ { ... } ]`) |
| `MaxTableKeys` | 10 000 | the direct children of one table of a document, equal to `MaxRecipes`, the largest table a pack writes (6.3.1) |

`MaxPrice` and `MaxQuantity` are bounds, not tunables. Specs that call them
"configured" (the clearing spec does) mean that validation checks every field
against them; nothing in a `Config` can raise them.

Some products stay unbounded by these constants: a value times a number of
basis points can reach `10^18 * 10^6`. Those go through the checked
multiply-divide helper with a 128-bit intermediate (`spec-currency.md` section 7;
architecture.md 3.5). The bounds above are the ones validation can check against
a field without evaluating the whole model.

### 4.2 Derived checks

Some limits depend on two fields at once. Validation computes them and rejects
the configuration with `ErrOverflow` if one fails:

- the largest price in a region times the largest quantity of any order, stock or
  recipe amount;
- the largest weight a group can get (`spec-scale.md` 3.2) times the largest
  per-household quantity or amount of that role (`FR-SCL-011`);
- a quantity times a commodity's mass or bulk, and the sum of those over a
  container's load (`spec-goods.md` section 3);
- the **effective** welfare window of each role, `T = max(WelfareWindow,
  WindowCycles * longest Duration on the role's list)` (`spec-welfare-and-wages.md`
  2.3.3), against `MaxWelfareWindow`, and then `T` times the largest daily income
  (2.3.2). The check uses the effective window and not the base one;
- the calibration products of `spec-welfare-and-wages.md` section 3.1, a ratio in
  basis points times `B_r` times `TicksPerYear` (these use the 128-bit helper, so
  the check is that the result fits);
- the total money at start, summed with checked addition, against `MaxMoney`;
- the calendar: `DaysPerYear * TicksPerDay`, and the derived `TicksPerYear`
  against `MaxTicksPerYear` (`spec-calendar.md` 7.2);
- for every recipe (`spec-production.md` 10.3): the largest weight a group can get
  times each input quantity, and times each output quantity times the largest yield
  factor the configuration can give the recipe (the largest `Yields` entry, times
  `MaxYieldBP` as the shock bound, times the trainee factor) over 10 000, is at
  most `MaxQuantity`; the same weight times each draw quantity is at most
  `MaxQuantity`; and the sum over inputs of quantity times the largest reference
  price is at most `MaxMoney`, so `UnitCost` cannot overflow;
- for agents (`spec-agents.md` 13): the largest weight times an upkeep quantity, a
  start quantity or a job input, summed with the targets of the default strategy,
  is at most `MaxQuantity`; `StartMoney` times the largest weight is at most
  `MaxMoney`; and the cost of an upkeep list at the largest reference price is at
  most `MaxMoney`;
- for the warm start (`spec-viability.md` 8.7): each `Money` at most `MaxMoney`,
  each stock quantity at most `MaxQuantity`, and the largest weight times a stock
  quantity at most `MaxQuantity`;
- for beliefs (`spec-beliefs.md` 11): `ceil(10 000 / BeliefStep)` is at most the
  cheapest tradable commodity's effective reference price in every region (7.4);
- for wants (`spec-demand.md` 10.1): the largest weight times `Quantity` times the
  taste and region scales times `Ceiling`, over `Period`, is at most `MaxQuantity`
  (the products use the 128-bit helper), and the sum over a role's ladder of the
  largest want spending at the largest reference price is at most `MaxMoney`;
- for the spawner (`spec-spawner.md` 12): `Households * EntryShare` and
  `Households * ExitShare` at the largest weight, and `Wage * ExpectedWorkTicks`,
  within `MaxQuantity` and `MaxMoney`.
- for the region graph (`spec-regions.md` 4.2): a path sum and an effective edge length (`Base * FactorBP / 10 000`) fit `int64`, and the checked helper is still used;
- for transport (`spec-transport.md` 12.1): `MaxWeight * MaxCapacity` fits `int64` (true at the bounds); the products of a distance with a rate and a chargeable size, and of feed, wear and haul units, go through the 128-bit helper and are refused at dispatch as the matching insufficiency or `overflow`, so no further derived bound is needed;
- for the money-supply and population modules: none beyond the bounds (`spec-money-supply.md` 12.1, `spec-population.md` 9.1).
- for the resources module (`spec-resources.md` 4.6, 13.2): a capacity times the largest shock scale over 10 000 is at most `MaxEndowment`; the checks of `spec-production.md` 10.3 for draws are unchanged (`ErrOverflow`);
- for the information module (`spec-information.md` 9.2): the number of sources times `RumourMaxAge` is at most `MaxRumourCells`, and for each informed role the size of its tradable set times the number of other regions is at most `MaxRumoursPerView` (`ErrTooMany`);
- for the subsistence module: none new; the existing `ErrOverflow` check on the cost of the upkeep at the largest reference price (`spec-agents.md` 13) covers the `Cost` of `spec-subsistence.md` 5.2;
- for the migration module (`spec-migration.md` 12.2): for each role that is not `Fixed` and has recipes, `|K(role)|` times the number of other regions is at most `MaxRumoursPerView` (`ErrTooMany`); `Households * MoveOutShare` and `Households * MoveInShare` at the largest group, and those of the rows, are at most `MaxQuantity` (`ErrOverflow`); the products of the estimate use the 128-bit helper and are covered by the unit-cost check of `spec-production.md` 10.3;
- for the merchant module: none new (`spec-merchants.md` 14.1: a product of the margin arithmetic that does not fit drops a candidate at run time, a lot is at most `MaxQuantity`, and the escrow is at most the money in existence);
- for the external module: none beyond the bounds (`spec-external-sector.md` 13.2); the products of a quote and a capacity are at most `MaxPrice * MaxQuantity` = 10^18.
- for the modules of wave 5, as their specs state: tenure, a term's `Rent` times the largest draw of a recipe on its resource times the largest weight of a group in the region is at most `MaxMoney` (`spec-land.md` 11.1, `ErrOverflow`); employment, the largest yield factor of the check of `spec-production.md` 10.3 takes `HireYieldBP` into account (`spec-employment.md` 14.1, `ErrOverflow`), and a bill that does not fit fails the formation at run time and is not an error; storage, the room rules `room_below_upkeep`, `room_below_start_stock` and `room_below_warm_job` are shape errors (`ErrRoomShape`) and not overflow, and a load is computed through the 128-bit helper and refused where it would exceed the room (`spec-storage.md` 12.2, 13.1); settlements, credit, infrastructure, insurance, fiscal, saving and technology, none beyond the bounds (`spec-settlements.md` 9.1, `spec-credit.md` 15.1, `spec-infrastructure.md` 13.3, `spec-insurance.md` 14.1, `spec-fiscal.md` 11.4, `spec-saving-and-investment.md` 11.3, `spec-technology.md` 9.5: a premium, a payout or a loan that would pass `MaxMoney` is refused at formation and is a counted reason); the risk, polity, institutions and policy specs state none.

## 5. Defaults

5.1. A default is defined once, in the root package, in `defaults.go`:
`DefaultWorldParams()`, `DefaultRole()`, `DefaultRegion()` and one constructor for each
other entity or module that has defaults (architecture.md 9.8). Each returns a
fully populated value. Defaults are the stable constants of ADR 0001 (decision
5): changing one changes behaviour and is explained in its PR.

5.2. The zero value of an entity is **not** a valid default. A `Role{}` has
`TargetRatio` 0, which validation rejects (it must be at least 10 000,
`spec-welfare-and-wages.md` section 5.1). This is deliberate: a zero that
silently meant "use the default" cannot be told apart from a deliberate 0, and
some fields (such as `TargetSpread`) have a valid 0 and a non-zero default. A name
is the other case: `World.Clearing` empty fails (`ErrUnknownClearing`, 7.1.4), and
the default `"average"` comes from `DefaultWorldParams()` and not from the zero value.
The documentation of each config type says to start from its `Default` constructor.
Architecture.md 6.3 requires this to be documented.

5.3. A field with no sensible default is **required**: it has none, and the file
loader reports `ErrMissingField` if it is absent. A Go-built value that leaves it
at zero fails validation on its bound. The reference price of a commodity in a
region (`spec-clearing.md` 5.4) is an example.

5.4. The file loader (section 6) starts every entity from its `Default`
constructor and overlays the keys present. A field absent from the file takes the
default. So the file never restates a default, and a default can be read in one
place.

5.5. A default for a field with a sensible per-setting value is a *placeholder
for the engine*, not an opinion about a setting. Setting-specific values belong
in a pack (ADR 0001, decision 6), which overrides the default in a later
document (6.4).

## 6. The file format and `config/`

### 6.1 Format: TOML (decided) and library (decided)

The format is TOML, parsed by `github.com/pelletier/go-toml/v2`, a third-party
library confined to `config/` (architecture.md 5.2). The choice is recorded, with
the justification that architecture.md 5.3 requires (what it provides, why the
standard library is insufficient, maintenance status and licence) and the
experiments behind it, in [ADR 0009](../adr/0009-the-toml-library-is-pelletier-go-toml-v2.md).
In short: the standard library has no TOML parser; the library parses into a
generic tree with integers and floats told apart, has no dependency of its own, and
bounds nested arrays and inline tables itself but not dotted keys, table headers or
the number of keys of a table, which 6.3.1 bounds before it is called.

Reasons TOML was chosen:

- comments, which a designer-authored pack needs;
- integers and floats are distinct types in the syntax, so rule 2.6 is easy to
  enforce;
- no implicit typing of the YAML kind (the string `no` read as a boolean), and
  duplicate keys are an error;
- tables keyed by name fit the entities of this library (6.2).

Alternatives:

- **JSON** (standard library only, no dependency): no comments, noisy for
  hand-authoring, and a number in JSON does not say whether it is an integer. It
  would need no third-party import, which is its strength.
- **YAML**: comments, but implicit typing, anchors and a larger attack surface.
- **A Go-only configuration:** always possible and always supported (2.2); it
  just gives no data-pack file for non-programmers.
- **`github.com/BurntSushi/toml`**: measured and rejected in ADR 0009 (no bound on
  nesting; out of memory at 10 000 nested inline tables).

This spec depends on one capability of the library: "parse TOML into a tree of
`map[string]any`, `[]any`, `int64`, `float64`, `string` and `bool`, with integer and
float distinguished". `config/` decodes into that tree and builds the structs by
hand (6.2); it does not use the library's struct decoding, its strict mode or the
text of its errors (6.3.2).

### 6.2 Layout of a file

Entities are **tables keyed by ID**, not arrays, so they merge by key (6.4),
cannot be duplicated within a document, and need no ordering. Lists of
(commodity, quantity) are inline tables. Key names are the snake case of the Go
field names (`TargetRatio` becomes `target_ratio`), by a fixed rule, so the
mapping is predictable and testable. The rule treats a run of capitals as one
word: `RateBP` becomes `rate_bp`, `ID` becomes `id` and `TargetRatio` is two words,
so an acronym is never split letter by letter. A path of a key in a document is the
table prefix of its scope followed by the snake case of the field name
(`spec-experiments.md` 3.2.1, 11.3.1). The registry of knobs of `spec-calibration.md`
maps a path to its row by the same rule, with the prefix-to-scope map of
`spec-experiments.md` 11.3.1, and never by the last segment alone (N-102).

An illustration using fields specified in the specs listed in 3.7. Commodities
are elided because `spec-goods.md` fixes their fields. Names such as `worker_a`,
`grain` and `north` are placeholders:

```toml
schema = 1

[world]
scale = 40
min_agents = 10
upkeep = [ { commodity = "grain", quantity = 2 } ]   # the baseline household
belief_step = 200
welfare_window = 30

# Absent, [calendar] is Earth's. A planet sets the hours in a day, the days in a
# week, the months, the ticks in a day and the seasons: see spec-calendar.md 7.4.
# Months and seasons are arrays, because their order is data (3.4).
[calendar]
ticks_per_day = 1

[role.worker_a]
recipes = [ "grow", "mill" ]
trades = [ "salt" ]
start_money = 10000
start_stock = [ { commodity = "grain", quantity = 6 } ]
strategy = "default"
target_ratio = 15000
target_spread = 500
expected_work_ticks = 250

[recipe.grow]
duration = 120
sited = true
inputs  = [ { commodity = "seed", quantity = 1 } ]
outputs = [ { commodity = "grain", quantity = 600, cost_share = 10000 } ]

[region.north]
reference_basket = 1000
capital_return = 1000
extra_upkeep = [ { commodity = "fuel", quantity = 1 } ]
pool_release = 100

# A yield profile has a table of its own, keyed by region and recipe. A recipe
# names no region and a region names no recipe (spec-production.md 6.1).
[region.north.yield.grow]
seasons = { spring = 10000 }

[region.north.group.worker_a]
households = 2000

# The warm-start seed table (spec-viability.md 8.1). Absent means no warm start.
# A price missing here keeps the reference price; a group missing keeps the role's
# defaults.
[warm_start.region.north]
wage = 2980          # starting wage per working household-tick, used whenever the pooled figure of the region is unknown

[warm_start.region.north.price]
grain = 480

[warm_start.region.north.group.worker_a]
money = 4200
stock = [ { commodity = "grain", quantity = 6 } ]
jobs = [ { recipe = "grow", in_progress = 2300 } ]

[modules.estates]
```

A demand module is a table of its own, `[modules.demand]`, whose presence switches it on; its
wants and tastes are tables keyed by ID, so documents merge by key
(`spec-demand.md` 3.2). A role that the default spawner must not fill is written
`closed = true`:

```toml
[modules.demand]
[modules.demand.want.comfort]
items = [ { commodity = "cloth", weight = 10000 } ]
quantity = 1
period = 4
rank = 1
elasticity = 15000
[modules.demand.taste.worker_a.comfort]
scale = 10000
[modules.demand.region_taste.north.comfort]
scale = 15000

[role.visitor]
closed = true
```

The presence of `[modules.estates]` switches the module on (3.2); it is the
pool policy (relief and the pool as a flow target, `spec-estates.md` 2.6), and its
parameters are region fields (`spec-estates.md` 3) that sit in the region table.
The presence of `[warm_start]` switches nothing on: it supplies a start (3.1).
`commodity.*` is elided; see `spec-goods.md`. No setting vocabulary is in the engine
(AGENTS.md).

#### 6.2.1 One rule for module tables (N-102, decided)

Every spec used to spell its module's table its own way (the singular prefix `module` for
estates, money supply and the external sector, bare top-level tables for demand, population
and resources, and the plural prefix `modules` for the later modules). There is one
rule now, applied to every spec and every fixture together, and no other spelling is
read:

- **The table of a module is `[modules.<name>]`**, where `<name>` is the snake case
  (6.2) of the module's field in `Config.Modules`: `modules.demand`,
  `modules.money_supply`, `modules.population`, `modules.transport`,
  `modules.resources`, `modules.information`, `modules.external`,
  `modules.subsistence`, `modules.migration`, `modules.merchants`, the fourteen
  later modules in the order of 3.1, and `modules.estates`. A top-level table named
  after a module (a bare `demand`) or a table under the singular prefix `module` is
  `ErrUnknownKey`.
- **A module is on if and only if its table is in the merged tree**, even with no
  key in it (`[modules.estates]` alone is on). A sub-table such as
  `[modules.demand.want.comfort]` makes `modules.demand` present, as TOML makes the
  implicit table. A later document cannot turn a module off (6.4).
- **The entities a module owns are tables one level below its table, keyed by ID**,
  and the key is the *singular* snake case of the Go field: `Wants` is `want`,
  `Flows` is `flow`, `Modes` is `mode`, `Nodes` is `node`, `Classes` is `class`,
  `Jurisdictions` is `jurisdiction`. A row keyed by two IDs is two levels
  (`[modules.demand.taste.<role>.<want>]`,
  `[modules.tenure.term.<region>.<resource>]`). A list whose order is the data stays
  an array (3.4).
- **A block a module adds to a role or a recipe is a sub-table of that entity**, the
  singular snake case of its Go field: `[role.<id>.merchant]`,
  `[role.<id>.employer]`, `[role.<id>.credit.lend]`, `[recipe.<id>.builds]`. A flag
  or a scalar a module adds is a key of the entity (`informed`, `fixed`,
  `hireable`, `needs`, `ceiling`).
- **A table of regional rows of a module is `[modules.<name>.region.<id>]`**.
  A parameter that is a field of `Region` stays a key of `[region.<id>]` even when a
  module owns it (`pool_release`, `relief_below`; 3.7.2).

The spelling of every module's tables, in this rule's form, is in 6.2.3.

#### 6.2.2 Key names and their exceptions

A key is the snake case of the Go field name (6.2, the acronym rule) and an entity's
`ID` is the key of its table, with these exceptions, which are all of the core's at
Stage 0 (a module or a later section adds its own rows in the pull request that builds
it). A test walks the fields of every section type and fails for a field that is neither
its snake case nor a row of this table.

| In the file | In Go | Why |
|-------------|-------|-----|
| `[region.<r>.group.<role>]` with `households`, `scale` | `Groups`: `Group{Region, Role, Households, Scale}` | a group is a role in a region: both are keys of the table's path, and the section is written under its region |
| `[region.<r>.yield.<recipe>]` with `all` and `seasons = { <season> = <bp> }` | `Yields`: `YieldProfile{Region, Recipe, HasAll, All, Seasons []SeasonYield}` | keyed by region and recipe like a group; `all` present is `HasAll` (there is no key `has_all`); `seasons` is an inline table from a season ID to basis points and is a list sorted by season ID in Go |
| `[region.<r>.edge.<to>]` with `distance`; `[region.<r>.reference_price]` with `<commodity> = <price>` | `Edges`: `Edge{From, To, Distance}`; `RegionPrices`: `RegionPrice{Region, Commodity, Price}` | rows written under their region, the other end or the commodity being the key (`spec-regions.md` 4.1, 5.2; built in Stage 2) |
| `[warm_start.region.<r>]` with `wage` and `pool`; `[warm_start.region.<r>.price]` with `<commodity> = <price>`; `[warm_start.region.<r>.group.<role>]` with `money`, `stock`, `jobs` | `WarmStart{Prices, Wages, Groups, Pools}` of `SeedPrice`, `SeedWage`, `WarmGroup`, `WarmPool` | the four lists are written once per region; `wage` is `SeedWage.Wage` and `pool` is `WarmPool.Money` |
| `in_progress` in a `jobs` entry | `JobSpec.InProgressBP` | the one key that is not the snake case of its field: the packs of `testdata/chains/` were written with it, and `spec-viability.md` 8.5.1 keeps the Go name |
| `[calendar]` `months = [ { id, days } ]`, `seasons = [ { id, days } ]` | `Calendar.Months []Month{ID, Days}`, `Calendar.Seasons []Season{ID, Days}` | ordered lists: arrays of inline tables, not tables by ID (3.4); the `ID` of an element is the key `id` |
| `schema` | none | the version of the file format (6.3), not a field |

Entity lists are tables by ID named by the singular of the section: `[commodity.<id>]`
for `Commodities`, `[recipe.<id>]`, `[role.<id>]`, `[region.<id>]`. That is the rule of
6.2.1 and not an exception. A list of (commodity, quantity) is an array of inline
tables with the keys `commodity` and `quantity` (a recipe's outputs add `cost_share`
and `discard`, its tools `wear_per_use`). Module tables and later core sections follow
their owner specs and add their exceptions to this table in the PR that builds them.

A validation error's `Path` is not a file path. It is the path of the Go field in the
section's own names (`roles.worker_a.target_ratio`: the plural list name, the ID, the
snake case of the field, 7.1.1), and the loader's errors carry the file path
(`role.worker_a.target_ratio`). The two differ by the plural and by the rows above.

#### 6.2.3 The tables of each module

Each spec says its keys; this is where the tables are in the form of 6.2.1. Edges and
region prices are tables of the region (6.2.2). The estates module is `[modules.estates]`,
with no key. The demand module is `[modules.demand]` with `[modules.demand.want.<id>]`,
`[modules.demand.taste.<role>.<want>]` and `[modules.demand.region_taste.<region>.<want>]`
(`spec-demand.md` 3.2). The money-supply module is `[modules.money_supply]` with
flows as tables `[modules.money_supply.flow.<id>]` (`spec-money-supply.md` 17.6). The
population module is `[modules.population]` with `birth_schedule` and `death_schedule` as
lists of `{ ratio, rate }`, `min_households`, `birth_stake`, and regions as
`[modules.population.region.<id>]` (`spec-population.md` 8). The transport module is
`[modules.transport]` with its modes as `[modules.transport.mode.<id>]`
(`spec-transport.md` 3.1, 11). The resources module is `[modules.resources]` with
`horizon_years`, `[modules.resources.resource.<id>]` (keys `shape`, `rate_per_year`,
`rate_bp`, `refuge`, `capacity`, `initial_bp`, `plan_share_bp`, `abundance = [{ share,
factor }]`, `seasons = [{ season, bp }]`) and
`[modules.resources.endowment.<region>.<resource>]` (keys `capacity`, `initial`,
`regen_scale_bp`) (`spec-resources.md` 3.3.2, 16). The information module is
`[modules.information]` with `info_speed`, `rumour_error_bp` and `rumour_max_age`, and
`informed = true` in `[role.<id>]` (`spec-information.md` 12.1). The external module is
`[modules.external.node.<region>]` with
`[modules.external.node.<region>.line.<commodity>]` (keys `base`, `trend_bp`,
`floor_bp`, `ceiling_bp`, `walk_step_bp`, `walk_revert_bp`, `walk_bound_bp`,
`price_by_season = [{ season, bp }]`), its sides `...line.<commodity>.sell` and
`...line.<commodity>.buy` (keys `margin_bp`, `capacity`, `window = [{ season, bp }]`,
`total_units`, `total_value`) and gates `[modules.external.node.<region>.gate.<region>]`
(key `scale_bp`) (`spec-external-sector.md` 3.3.1, 18.2). The subsistence module is
`[modules.subsistence]` with `recipes`, `enter_ticks`, `exit_ticks` and `retry_jobs`, and a
role row as `[modules.subsistence.role.<id>]` with `recipes` (`spec-subsistence.md`
3.2.2, 16.1). The migration module is `[modules.migration]` with `move_speed`,
`move_review_ticks`, `move_below`, `move_gain`, `move_width`, `move_cooldown`,
`move_horizon`, `move_out_share` and `move_in_share`, a region row as
`[modules.migration.region.<id>]` with `out_share` and `in_share`, and `fixed = true` in
`[role.<id>]` (`spec-migration.md` 11, 15.3). The merchant module is
`[modules.merchants]` with `max_open`, and the merchant block is `[role.<id>.merchant]`
with `modes`, `lot_units`, `fee_share_spread_bp`, `max_source_age`, `time_cost_bp`,
`consignee_cut_bp`, `consigns`, `take_margin_bp` and `take_units`, beside the role keys
`merchant_fee_share`, `informed` and `strategy = "merchant"` (`spec-merchants.md` 19.1).
The settlements module is `[modules.settlements]` with `band_bp` and `dwell`, a class as
`[modules.settlements.class.<id>]` with `enter` and `flags`, its limits as
`[modules.settlements.class.<id>.limits]` (`<name> = <value>`), a region row as
`[modules.settlements.region.<id>]` with `threshold_bp` and `floor`, and `needs` and
`ceiling` in `[role.<id>]` (`spec-settlements.md` 3.1). The storage module is
`[modules.storage]` with a class as `[modules.storage.class.<id>]` (keys `mass`, `bulk`,
`needs`; `spec-storage.md` 7.3); by 6.2.1 its base room is the sub-table
`[modules.storage.base]` (keys `mass`, `bulk`; absent is unlimited) and a role row is
`[modules.storage.role.<id>]` with `stores` and a `[modules.storage.role.<id>.base]`
(`spec-storage.md` 12.1 states no layout for them: owed there, N-016). The employment
module is `[modules.employment]` with `max_open`, `hireable = true` in `[role.<id>]` and
the block `[role.<id>.employer]` (`recipes`, `max_hires`, `worker_share_bp`,
`hire_yield_bp`; `spec-employment.md` 13). The tenure module is `[modules.tenure]` with a
term as `[modules.tenure.term.<region>.<resource>]` and a holding as
`[modules.tenure.holding.<region>.<resource>.<holder>]`, `<holder>` being a role ID or the
word `pool` (`spec-land.md` 3.3.2). The credit module is `[modules.credit]` with the role
blocks `[role.<id>.credit.lend]` and `[role.<id>.credit.borrow]` (`spec-credit.md` 4.2).
The risk module is `[modules.risk]` with `[modules.risk.route.<from>.<to>.<mode>]` and
`[modules.risk.store.<region>.<class>]`, keys `strike_bp`, `severity_bp` and `common`
(`spec-risk.md` FR-RSK-001; a path of seven segments, within `spec-experiments.md`
3.2.1). The infrastructure module is `[modules.infrastructure]` with
`[modules.infrastructure.class.<id>]` and `[[modules.infrastructure.initial]]`, and a
recipe carries `needs`, `[recipe.<id>.builds]` and `[recipe.<id>.repairs]`
(`spec-infrastructure.md` 12.1, 4.1.1). The insurance module is `[modules.insurance]` with
`[role.<id>.cover.underwrite]` and `[role.<id>.cover.insure]` (`spec-insurance.md` 4.2).
The polity module is `[modules.polity]` with `[modules.polity.jurisdiction.<id>]`
(`parent`, `seat`, `start_money`) and `[modules.polity.member]` with `<region> =
"<jurisdiction>"` (`spec-polity.md` 3.4). The fiscal module is `[modules.fiscal]` with
`[modules.fiscal.tax.<id>]`, `.duty.<id>`, `.toll.<id>`, `.payout.<id>`, `.purchase.<id>`
and `.release.<id>` (`spec-fiscal.md` 3.1). The institutions module is
`[modules.institutions]` with `[modules.institutions.body.<id>]`, `.right.<id>`,
`.control.<id>` and `.rate_limit.<id>` (`spec-institutions.md` 9). The policy module is
`[modules.policy]` with `[modules.policy.actor.<id>]`,
`[modules.policy.actor.<id>.lever.<id>]`,
`[modules.policy.actor.<id>.schedule.<lever>.<tick>]` and
`[modules.policy.actor.<id>.rule.<lever>]` (`spec-policy.md` 3.1.2). The saving module is
`[modules.saving]` with the role block `[role.<id>.saver]` and its rows
`[[role.<id>.saver.worth]]` (`spec-saving-and-investment.md` 3.2). The technology module
is `[modules.technology]` with `[modules.technology.field.<id>]`, whose `recipes` is a
list (`spec-technology.md` 9.1).

#### 6.2.4 The estates module and the Go that holds it

The estate account, its pseudo-agent, its listings and the use of the pool to fund
an entrant exist in every world (`spec-regions.md` 3.6.2, FR-EST-004, FR-EST-014,
FR-EST-021). What `[modules.estates]` switches on is the **pool policy**: relief
(`spec-estates.md` 2.6) and the pool as a target of a money flow
(`spec-money-supply.md` 4.3.2). In Go it is `Modules.Estates *Estates`, the 25th
pointer of 3.1, appended after `Technology`, with `type Estates struct{}` (no field:
the policy's parameters are `Region.PoolRelease` and `Region.ReliefBelow`, which the
module owns, and `Region.EstateMarkdown`, which the listing rule of the core owns).
`ReliefPaid` is the module's event and is **not** an event of the slice. The two
region fields are appended to `Region` with the module, so the slice's `Region` has
`EstateMarkdown` only (3.7.2). This answers open question 8 of `spec-money-supply.md`
(the account exists without the module; the pool target needs it, `ErrModuleDependency`
stays) and the layer line of `spec-estates.md`, which says "module for the regional pool,
starting money and relief" and should say "relief".

### 6.3 What the loader rejects

A pack may come from a third party, so the loader treats every byte as untrusted
(6.3.1) and its errors as text built from untrusted input (6.3.2). It rejects:

- a document that is not TOML (`ErrSyntax`; the error carries the line and the
  column of the library's error and none of the document's text, 6.3.2);
- an **unknown key**, at any depth (`ErrUnknownKey`, with its path), including a
  module table the library does not know and a module table of a module that is not
  built;
- a floating point literal where an integer is needed (`ErrNotInteger`, rule 2.6),
  and any other value of the wrong type: a string where a number or a bool is needed,
  an integer where a string is needed, a date or a time anywhere (`ErrWrongType`);
- a missing required field (`ErrMissingField`);
- a missing or unsupported `schema` (`ErrUnsupportedSchema`). The schema number
  versions the file format and is independent of the checkpoint format version
  (ADR 0002, decision 8). It is checked on the **merged** tree (6.4): an overlay
  document, one that only changes a few keys, has no `schema` line of its own, and
  the first document of a pack has it;
- a key present as a table in one document and as a scalar in another
  (`ErrMergeConflict`, 6.4);
- an input over a limit of 6.3.1 (`ErrTooLarge`).

The eight sentinels are root sentinels in `errors.go` (14): `config/` returns them
(wrapped with the path, `%w`) and a consumer matches them with `errors.Is`. The loader
does not check cross-references or bounds. That is validation (section 7), the same
code whether the `Config` came from a file or from Go.

#### 6.3.1 Limits on untrusted input

The limits are constants of `bounds.go` (4.1). Each is checked before the work it
bounds, in this order, and a limit is `ErrTooLarge` with the name of the constant and
the measured value in its message:

| Check | Constant | Value | What it bounds |
|-------|----------|-------|----------------|
| documents passed or named | `MaxDocuments` | 16 | the count of documents of one `Parse` or `LoadFS`, checked before any is read or opened |
| one document | `MaxDocumentBytes` | 4 MiB | the size of one document |
| all documents | `MaxMergedBytes` | 16 MiB | the sum of the sizes of the documents of one call |
| one line | `MaxLineBytes` | 256 KiB | the size of one line (up to a `\n`) of a document |
| nesting | `MaxNestingDepth` | 16 | the segments of a dotted key or a table header, and the nesting of arrays and inline tables in a value, counted together from the document root |
| keys of one table | `MaxTableKeys` | 10 000 | the keys that are direct children of one table, a table header and a dotted key both counting |

The first four are byte counts and are checked on the bytes. `LoadFS` reads each file
through a reader limited to `MaxDocumentBytes + 1` bytes and to what is left of
`MaxMergedBytes`, whatever `fs.FileInfo` says of its size, because an `fs.FS` may
report any size; a file that is longer than its limit is `ErrTooLarge`. The last two are
checked by a **guard pass** that parses each document once, only to count, before the
document is decoded (ADR 0009 says why and what it costs: the library bounds nested
arrays and inline tables at 10 000 levels but not dotted keys or table headers, 2 MB of
`a.a.a. ...` decodes to 600 MB of tables, and the number of keys of one table makes its
decoding quadratic). With the six limits the decoding of a document that passes them
is bounded in memory by about 64 times its size and in time by about two seconds
per MiB (the measurements are in ADR 0009), and every recursion over the tree (the merge, the unknown-key walk, the mapping)
is at most `MaxNestingDepth` deep, so none can overflow the stack. The values are
placeholders in the sense of N-015: the reasons are in ADR 0009, and a pack that needs
more splits into documents. A merged tree has no more depth than its deepest document.

#### 6.3.2 Error text built from input

A document's keys, a Go-built `Config`'s IDs and a value that fails a rule are
input. Error text that includes them is bounded and cannot carry a control
character:

- each **path segment** is cut at the last rune boundary at or before `MaxIDLength`
  (64) bytes and written with `%q`, followed by `...` outside the quotes when it was
  cut; the path of an error is the segments joined by `.` and has at most
  `MaxNestingDepth` of them (a Go path has at most the depth of the types);
- a **value** in a message is written the same way when it is a string, and as a
  decimal when it is an integer;
- the loader **never copies the library's error text**. It reads the line and column
  of a syntax error and writes its own message; a message of the library can hold
  the whole text of a key (ADR 0009);
- the field `Path` of a `*ValidationError` holds the segments as given (the typed
  error carries the truth), and it is its `Error()` method that cuts and quotes.

The rule is the same for the loader (6.3) and for `Validate` (7.1.1), and a test feeds
each an oversized ID and an ID with control characters (FR-CFG-100).

#### 6.3.3 How many errors the loader returns

A limit (6.3.1) and a syntax error stop the load at once with one error, since there
is nothing to map. The rejections of the mapping and the merge (unknown keys, wrong
types, missing fields, merge conflicts) are collected over the whole merged tree and
returned as `Validate` returns its errors (7.1): joined with `errors.Join`, sorted by
path and then by rule, at most 100, followed by one `ErrTooManyErrors` that says how
many were left out.

### 6.4 Packs and layering

A **pack** is a set of documents (ADR 0001, decision 6). Documents are merged in
the order given, later over earlier:

- tables merge recursively by key;
- a scalar or a list in a later document replaces the earlier one, and an explicit `false` (or 0) replaces a `true` (or a non-zero value): a key that is present is never read as absent, so an overlay can turn `raw = true` of a base commodity off with `raw = false`;
- a key present as a table in one document and as a scalar in another is an error
  (`ErrMergeConflict`);
- a document cannot delete an entity or a key in version 1 (open question 3), and so
  cannot turn a module off;
- the `schema` key is merged like any scalar and checked once, on the merged tree
  (6.3), so a document that overlays a pack may omit it.

This is how the three levels of ADR 0001 decision 3 look in files: a base pack
(the chain), a pack that tweaks parameters (overrides scalars), and a pack that
adjusts modules (adds a `[modules.*]` table). Because entities are tables keyed by
ID, no merge rule for arrays of entities is needed.

### 6.5 The API

`config/` offers, in outline:

```go
// Parse merges the documents in order and returns the Config, or an error.
func Parse(docs ...[]byte) (economysim.Config, error)

// LoadFS reads the named files from fsys and calls Parse.
func LoadFS(fsys fs.FS, names ...string) (economysim.Config, error)
```

The library never opens a path itself: the consumer supplies bytes or an
`fs.FS`, which keeps it a pure library (architecture.md 1.1) and makes tests use
`testing/fstest` and fixtures under `testdata/`. `Parse` does not validate beyond
6.3; `NewWorld` does. `Parse` checks the count and the sizes of the documents before it
decodes any (6.3.1); `LoadFS` checks the count of names before it opens any and reads
each file with a bound (6.3.1). A name that cannot be read is the `fs` error wrapped
with `%w` and the name written as a path segment (6.3.2).

## 7. Validation

### 7.1 The framework

7.1.1. `Config.Validate` returns every error it finds, not the first, joined
(`errors.Join`). Each is a `*ValidationError` with a `Path` (for example
`roles.worker_a.target_ratio` or `regions.north.groups.worker_a.households`), a
`Rule` (the sentinel), the offending value and, for a bound, the bound.
`errors.Is` matches the sentinel and `errors.As` finds the typed error
(architecture.md 8.4; testing.md 3.9). The viability rules of 7.6 return a second typed error, `*ViabilityError`, with the fields `Rule` (the rule name, a plain string: `unproducible_in_realm`, `role_cannot_work`, `cycle_not_productive`, `labour_short` or `storage_gap`), `Path`, `Value` (the offending value as text) and `Bound` (the number that decides it, as text); its `Unwrap` returns `ErrNotViable`, it does not embed `ValidationError`, and its `Error()` text quotes and cuts as the `ValidationError` text does. Both types are in the one list of 7.1.2 and 7.1.3. A `Path` is the path of the Go field, in the
plural list names of the sections and the IDs of the entities as given, and is not a key
path of a file (6.2.2). The `Error()` text cuts and quotes every segment and every string
value as 6.3.2 says, so a message holds at most `MaxNestingDepth` segments of at most
`MaxIDLength` bytes and no control character; the fields of the typed error keep the
value as given.

7.1.2. Errors are sorted by path and then by rule, so the output does not depend
on map order and a test can compare it (architecture.md 7.5). A `*ViabilityError` is sorted in the same list as a `*ValidationError`, by path, then rule name, then value and then bound.

7.1.3. At most 100 errors are returned, followed by one `ErrTooManyErrors` that
says how many were left out. A pathological file cannot flood a log. The cap bounds the
report; 7.1.6 bounds the work.

7.1.4. `NewWorld` calls `Validate` and returns its error. The only other checks it
makes on configuration are the four that need the options (`spec-world.md` 3.2.2).
They run **after `Validate` succeeds**, and each returns a `*ValidationError` with
the path of the field and the rule (7.1.1): every `Role.Strategy` name is
`"default"`, `"directed"` (both built in, `spec-agents.md` 6.6, 10) or a name
registered with `WithStrategy`, else `ErrUnknownStrategy`; `World.Spawner` is the
built-in `"default"` or a name registered with `WithSpawner`, else
`ErrUnknownSpawner` (`spec-spawner.md` 3.1.3); `World.Clearing` is the built-in
`"average"` or a name registered with `WithClearing`, else `ErrUnknownClearing`
with the path `world.clearing` (`spec-clearing.md` 8; an empty configured name
is that error, and an empty name given to `WithClearing` is `ErrInvalidID`,
`spec-world.md` 4.1.1); and, with `Modules.Policy` present, every `PolicyActor.Policy` is `"hold"`, `"schedule"`, `"rule"` or `"directed"` (built in, `spec-policy.md` 4.4) or a name registered with `WithPolicy`, else `ErrUnknownPolicy` with the path `modules.policy.actors[i].policy` (`spec-policy.md` 4.5.1). The name `"merchant"` is checked like any name, and
`strategy.Options()` registers it (`spec-merchants.md` 3.1). `Validate` takes no
options, so it cannot know the registrations, and it does not see
`WithTargetSpread` either (8.1); the four rules are therefore the config rules that
live in `NewWorld` and nowhere else, and a test builds a world with an unregistered
name of each kind (FR-CFG-038). Nothing else in `NewWorld` fails on configuration
(architecture.md 8.5).

7.1.5. `Validate` is pure and bounded. Its work is limited by the bounds of 4.1
and, for the viability programmes, by `MaxViabilityPivots` and `MaxViabilityCells`
(`spec-viability.md` 4.3); it starts no goroutine and draws no random number.

7.1.6. **The order of the checks, and what `ErrTooMany` stops.** `Validate` runs in
phases, in this order, and a phase runs only if every earlier phase that bounds its
input found nothing to refuse (phase 1 always stops the rest):

1. **Counts.** The length of every top-level collection of `Config` against its bound
   (4.1), each a `len` and so constant work; then, only if all hold, the nested lists
   (a recipe's items, a role's lists), a walk bounded by the first. A count over its
   bound is `ErrTooMany` with its path. If phase 1 found any, `Validate` returns those
   errors and nothing else: no sort, no set and no fixpoint is built on a collection
   that is not known to be small.
2. **Local rules.** IDs (`ErrInvalidID`), field bounds (`ErrOutOfBounds`), shapes, each
   reading one entity.
3. **Identity and references.** Duplicates (`ErrDuplicateID`) and unknown references
   (`ErrUnknown*`), by sorting and by sets over collections that phase 1 bounded.
4. **The structural fixpoint of 7.5** and the rules that read two sections.
5. **The derived checks of 4.2** (`ErrOverflow`).
6. **Viability** (7.6), which reads a well-formed configuration and so runs only when
   phases 1 to 5 found no error, on the canonical copy of the configuration (the recipe,
   yield and role tables it needs exist only for a canonical, validated configuration), and is
   size-gated besides. The rule "the set of virtual start offsets of a job is empty" (`ErrWarmStartJob`,
   `spec-viability.md` 8.7) runs at the entry of this phase for the same reason. One unexported
   `checkViability` serves `Validate` and `Warnings`.

Phases 2 to 5 collect every error they find (7.1.1); the rules that are super-linear in
a count (3, 4, 5 and 6) are the ones `ErrTooMany` keeps from running. Without phase 1 a
Go-built `Config` of 10^9 recipes would be sorted before it was refused.

### 7.2 Warnings

A **warning** is a configuration that is legal but probably a mistake, for
example a role whose starting money cannot buy its required tool at the reference
price (`spec-skill-barriers.md` 2.3). `Config.Warnings` returns them as values
with a path and a rule; they never fail validation. `Warnings()` is also a method of
`World`, which returns what `Config.Warnings` returned at construction, so a consumer
with no logger sees them (`spec-integration.md` 6.4). `NewWorld` logs each once at
`Warn` through the injected logger, with the attributes `rule` and `path`. That
is bounded by the size of the configuration and happens once at construction
(observability-and-logging.md 3.2.2). The warnings that exist are listed with
their conditions in 7.4.2; a rule is a warning or an error, never both, and the
severity of every rule is stated in 7.4. `Warnings` appends the viability warnings (7.4.2) only when the structural phases (1 to 5 of 7.1.6) come back clean on a scratch report, and otherwise returns the others, so `Validate` and `Warnings` agree on when the check runs; both reach the one `checkViability`.

### 7.3 Structural rules

These are owned here. Each has a sentinel and a failing-case test
(testing.md 3.9).

| Sentinel | Rejects | Source |
|----------|---------|--------|
| `ErrInvalidID` | an ID that does not match 3.3 | here |
| `ErrDuplicateID` | two entities of one kind with the same ID (in a Go-built `Config`; a file cannot) | architecture.md 8.5 |
| `ErrUnknownCommodity`, `ErrUnknownRole`, `ErrUnknownRecipe`, `ErrUnknownRegion` | a reference to an ID that is not defined | architecture.md 8.5 |
| `ErrRecipeNoOutput` | a recipe with no output | architecture.md 8.5 |
| `ErrOrphanCommodity` | a commodity that no recipe makes or uses, that no upkeep, tool or boundary mentions | architecture.md 8.5 |
| `ErrUnproducible` | a commodity that something consumes but that cannot be produced from a raw source (7.5); the error names one cycle that has no way in when there is one (7.5) | architecture.md 8.5 |
| `ErrOutOfBounds` | a value outside the bound of its field or of section 4.1 | architecture.md 3.5, 8.5 |
| `ErrOverflow` | a derived product of 4.2 that does not fit | architecture.md 3.5 |
| `ErrTooMany` | more entities of a kind than its bound; checked first, and it stops the other rules (7.1.6) | here |
| `ErrModuleDependency` | a module on while a module it needs is off (declared at Stage 0 with no returner yet: the first rule that returns it is a module's own, such as a recipe with `Draws` while resources is off) | here |
| `ErrNoRegions` | a configuration with no region | `spec-regions.md` 9.1 |
| `ErrSelfEdge` | an edge whose `From` equals its `To` | `spec-regions.md` 9.1 |
| `ErrUnknownResource` | a draw of a recipe or an endowment row that names a resource that is not defined | `spec-resources.md` 13.2 |
| `ErrResourceShape` | a resource shape that is not one of the three, a rate field the shape does not use set above 0, a refuge off a logistic stock or above a capacity, an initial stock above its capacity | `spec-resources.md` 13.2 |
| `ErrCurve` | abundance knots not strictly increasing, or a share or factor out of range | `spec-resources.md` 13.2 |
| `ErrOutsideNodeInhabited` | a group, warm-start row, population region row, taste row or money-flow target naming an outside node | `spec-external-sector.md` 14.1 |
| `ErrCrossedQuotes` | an outside line with both sides whose margins sum to less than `MinQuoteSpreadBP` | `spec-external-sector.md` 14.1 |
| `ErrNotAGate` | an outside gate row naming a region with no direct edge to or from the node | `spec-external-sector.md` 14.1 |
| `ErrOutsideNode` | a relocation into an outside node (run time, not validation) | `spec-regions.md` 6.1.3 |

The root maps the following sentinels of the new specs to public errors (N-075): `ErrNoRegions`, `ErrSelfEdge`, `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation`, `ErrRegionBound` (REG; the last three are ledger-level if `relocate` lives in `internal/ledger`); `ErrFlowShape`, `ErrReasonCollision` (MON); `ErrResourceShape`, `ErrCurve`, `ErrUnknownResource` (RES); `ErrOutsideNodeInhabited`, `ErrCrossedQuotes`, `ErrNotAGate`, `ErrOutsideNode` (EXT); INF adds none; `ErrVehicleNotDurable`, `ErrModeShape`, `ErrUnknownMode` (TRN: `ErrUnknownMode` is a root sentinel with the transport module alone, because `SetTransitFactors` returns it, `spec-integration.md` 5.1.3, and the merchant module reuses it); `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited`, `ErrOwnUseOverlap` (SUB); MIG adds none; `ErrMerchantShape`, `ErrMerchantWeight` (MER); `ErrUnknownClearing`, `ErrNilExtension`, `ErrWorldFailed`, `ErrInStep` (WLD, `spec-world.md` 15.1); `ErrCheckpointVersion`, `ErrCheckpointCorrupt`, `ErrCheckpointConfig`, `ErrCheckpointExtension`, `ErrOptionContext` (CHK, `spec-checkpoint.md` 13.1); `ErrModuleOff`, `ErrUnknownFlow`, `ErrUnknownLine`, `ErrAgentRemoved`, `ErrNotExternal`, `ErrNotDirected`, `ErrNotOnRole`, `ErrNotTradable` (INT, `spec-integration.md` 11.2; `ErrUnknownAgent` is REG's, above). The sentinels of the fourteen modules of wave 5 are mapped as well (N-075): `ErrUnknownSettlementClass`, `ErrUnknownSettlementFlag`, `ErrUnknownLimit` (SET; the flag name keeps it apart from the live `ErrUnknownFlow`); `ErrNoRoom` (the ledger's, mapped), `ErrUnknownStoreClass`, `ErrStoreShape`, `ErrRoomShape`, `ErrNotLimited`, `ErrAgentAway`, `ErrStoreLocked`, `ErrStoreInUse` (STO); `ErrEmploymentShape`, `ErrEmployerWeight` (EMP); `ErrTitleShares`, `ErrTitleCap`, `ErrNoTitle`, `ErrNotAHolder` (LND); `ErrCreditShape`, `ErrTooManyClaims` and the ledger's `ErrOpenClaim`, `ErrUnknownClaim`, `ErrClaimShape` (CRD); `ErrRiskShape`, `ErrUnknownRiskRow` (RSK); `ErrFacilityShape`, `ErrFacilityNoCost`, `ErrFacilityHolder`, `ErrFacilityCount`, `ErrUnknownFacilityClass`, `ErrFacilityOwner`, `ErrFacilityLocked`, `ErrFacilityTarget`, `ErrFacilityCap`, `ErrUnknownFacility`, `ErrFacilityNotTransferable`, `ErrFacilityRegion` (CAP); `ErrCoverShape` (INS); `ErrUnknownJurisdiction`, `ErrJurisdictionCycle`, `ErrSeatNotCovered`, `ErrSeatOutsideNode`, `ErrStartWithoutSeat` (POL); `ErrNotGoverned`, `ErrNotCovered`, `ErrFiscalShape` (FIS); `ErrUnknownBody`, `ErrBodyShape`, `ErrBodyOverlap`, `ErrRightShape`, `ErrControlShape`, `ErrUnknownControl`, `ErrUnknownRight` (ORG); `ErrPolicyShape`, `ErrLeverNotInReach`, `ErrLeverShared`, `ErrUnknownPolicy`, `ErrUnknownActor`, `ErrUnknownLever` (PLC); `ErrNotSaver` (SAV); `ErrTechShape`, `ErrTechRecipeOverlap`, `ErrUnknownTechField` (TEC). A sentinel that the owner spec reuses from another keeps its owner (7.4.1 names it). `ErrTravellerAway` does not exist: a refusal of a region-bound obligation is `ErrRegionBound` naming `open_shipment`.

A role with no group in any region is not a fault and draws no warning: it is how a
role for external agents is written (`visitor` in `spec-reference-chain.md` 4.2, with
`Closed`), and a role that a region does not use is legal.

### 7.4 Rules contributed by other specs

Each spec that adds parameters adds the rules for them to its own Requirements
section, with a sentinel, and the framework runs them. Every row gives its
severity: an **error** fails `Validate` (or `NewWorld`, where stated); a
**warning** never does (7.2). Known today:

#### 7.4.1 Errors

| Rule | Sentinel | Spec | Where |
|------|----------|------|-------|
| the cheapest tradable commodity's effective reference price in every region is at least the resolution floor `ceil(10 000 / BeliefStep)`; "tradable" is defined by `spec-agents.md` 3.4 (a commodity in the tradable set of at least one role); the effective reference price is the region's override or the commodity's (`spec-regions.md` 5.2) | `ErrOutOfBounds` | `spec-currency.md` 3.1, `spec-beliefs.md` FR-BEL-029 | `Validate` |
| a region's `ReferenceBasket` differs from the cost of the baseline `World.Upkeep` plus the region's `ExtraUpkeep` at its effective reference prices by more than `BasketTolerance` basis points of `ReferenceBasket` | `ErrBasketMismatch` | `spec-currency.md` 5.3, `spec-agents.md` 5.6.1 | `Validate` |
| the lowest possible per-agent target ratio is at least 10 000 | `ErrOutOfBounds` | `spec-welfare-and-wages.md` 5.1 | `Validate` |
| the derived `TicksPerYear` is between 1 and `MaxTicksPerYear`, and each role's `ExpectedWorkTicks` is between 1 and `TicksPerYear`. A role left at the default 250 fails if the calendar gives fewer than 250 ticks a year; this is decided and deliberate (5.2, 13.5); the error carries `TicksPerYear` as the bound, so the author sees the number to stay under | `ErrOutOfBounds` | `spec-calendar.md` 3.2, `spec-welfare-and-wages.md` 4.2 | `Validate` |
| the calendar's bounds, tick shape and seasons | `ErrOutOfBounds`, `ErrCalendarTick`, `ErrCalendarSeasons` | `spec-calendar.md` 7 | `Validate` |
| an effective welfare window above `MaxWelfareWindow`; a `WindowCycles` outside 1 to 100; a `MinWorkShare` outside 0 to 10 000 or above `ExpectedWorkTicks * 10 000 / TicksPerYear` | `ErrOutOfBounds` | `spec-welfare-and-wages.md` 2.3.4, 5.3, 12, FR-WEL-009, FR-WEL-028 | `Validate` |
| rule `traineeship_needs_weight_one`: a role with `TrainingTicks` above 0 that has a group with a scale above 1 (an agent of weight above 1) | `ErrOutOfBounds` | `spec-skill-barriers.md` 4.1.1 | `Validate` |
| `MinAgents` outside 1 to `MaxHouseholds` (a value above a group's households is legal: every household is its own agent) | `ErrOutOfBounds` | `spec-scale.md` 5.2, FR-SCL-020 | `Validate` |
| a welfare window or income that would overflow (the effective window, 4.2); the largest weight times the largest quantity or amount above `MaxQuantity`; money products that overflow | `ErrOverflow` | `spec-welfare-and-wages.md` 2.3.2, `spec-scale.md` 4.5, FR-SCL-011 | `Validate`, by 4.2 |
| a price or quantity of an order limit outside `[1, MaxPrice]` and `[1, MaxQuantity]` | `ErrOutOfBounds` | `spec-clearing.md` 3.2 | the bounds are 4.1 |
| a `Duration`, quantity, `WearPerUse`, yield, `TraineeYield`, or count of items, tools or draws outside its bound | `ErrOutOfBounds` | `spec-production.md` 10.2 | `Validate` |
| a commodity twice in one list of a recipe, or two yield profiles for one region and recipe | `ErrDuplicateID` | `spec-production.md` 10.2 | `Validate` |
| non-discard `CostShare`s that do not add to 10 000, a discard output with a share above 0, or a missing share with several non-discard outputs | `ErrCostShares` | `spec-production.md` 10.2, FR-PRO-004 | `Validate` |
| a `Requires` entry for a commodity with `Durability` 0 | `ErrToolNotDurable` | `spec-production.md` 10.2, FR-PRO-006 | `Validate` |
| a `TraineeEligible` recipe with `Requires` | `ErrEligibleHasTool` | `spec-production.md` 10.2, FR-PRO-033 | `Validate` |
| a yield entry for a season the calendar lacks, including any season entry when the calendar has no seasons | `ErrUnknownSeason` | `spec-production.md` 10.2, FR-PRO-038 | `Validate` |
| a recipe with `Draws` while the resources module is off | `ErrModuleDependency` | `spec-production.md` 10.2, FR-PRO-007 | `Validate` |
| a yield profile naming an unknown region or recipe; an upkeep, `StartStock`, `Trades`, `Recipes` or warm-start reference to an unknown commodity, recipe, role or region | `ErrUnknownRegion`, `ErrUnknownRecipe`, `ErrUnknownCommodity`, `ErrUnknownRole` | `spec-production.md` 10.2, `spec-agents.md` 13, `spec-viability.md` 8.7 | `Validate` |
| an upkeep or start quantity below 1, more than `MaxUpkeepItems` upkeep items, or an agent tunable outside its bound | `ErrOutOfBounds` | `spec-agents.md` 12, 13, FR-GDS-003 | `Validate` |
| a role naming a strategy that is neither `default`, `directed` nor registered | `ErrUnknownStrategy` | `spec-agents.md` 6.6 | **`NewWorld` only** (7.1.4) |
| a spawner name that is not registered | `ErrUnknownSpawner` | `spec-spawner.md` 3.1.3, 12 | **`NewWorld` only**, as the strategy |
| `World.Clearing` empty or naming no registered clearing (`average` is built in) | `ErrUnknownClearing` | `spec-clearing.md` 8, `spec-world.md` 3.2.2 | **`NewWorld` only**, as the strategy |
| a spawner parameter outside its bound; `ChangePrivation` not below `RemovePrivation` (rule `ladder_order`) | `ErrOutOfBounds` | `spec-spawner.md` 11, 12, FR-SPN-037, FR-SPN-038 | `Validate` |
| a want field outside its bound or a `Quantity` below 1, a `Ceiling` below 10 000, more than `MaxWantsPerRole` taste rows of a role | `ErrOutOfBounds` | `spec-demand.md` 9, 10.1 | `Validate` |
| a want ID, a (role, want) pair or a (region, want) pair twice, or a commodity twice in one want | `ErrDuplicateID` | `spec-demand.md` 10.1 | `Validate` |
| a want item, taste or region taste naming something undefined | `ErrUnknownCommodity`, `ErrUnknownRole`, `ErrUnknownRegion`, `ErrUnknownWant` | `spec-demand.md` 10.1 | `Validate` |
| a want with no item or with all weights 0 | `ErrWantItems` | `spec-demand.md` 10.1 | `Validate` |
| a want item with `Durability` above 0 | `ErrWantDurable` | `spec-demand.md` 3.2.1, 10.1 | `Validate` |
| the wants products of 4.2 that do not fit | `ErrOverflow` | `spec-demand.md` 10.1 | `Validate`, by 4.2 |
| a belief parameter outside its bound; `BeliefWidthMin` above `BeliefWidthInit` or `BeliefWidthInit` above `BeliefWidthMax`; `BeliefWidthMax` of 10 000 or more; `BeliefStockLow` not below `BeliefStockHigh`; `CostCeiling` below `CostFloor`; `RefBandLow` above 10 000; `RefBandHigh` below 10 000; `BeliefAdjustProb` of 0; `BeliefEvidenceShare` outside 1 to 10 000 | `ErrOutOfBounds` | `spec-beliefs.md` 11, FR-BEL-027, FR-BEL-028, FR-BEL-037 | `Validate` |
| a seed price outside `[1, MaxPrice]`, below the resolution floor or outside the reference band; a starting wage outside `[1, MaxMoney]` or whose product with the longest `Duration` of a recipe exceeds `MaxMoney` | `ErrWarmStartPrice` | `spec-viability.md` 8.2.2, 8.2.5, FR-VIA-030, FR-VIA-051 | `Validate` |
| a second starting wage for one region | `ErrDuplicateID` | `spec-viability.md` 8.7, FR-VIA-051 | `Validate` |
| a `JobSpec` whose recipe is not on the role's list, whose share is outside [0, 10 000], whose group shares exceed 10 000, or whose set of virtual start offsets is empty (checked at the entry of phase 6 of 7.1.6, on the canonical copy) | `ErrWarmStartJob` | `spec-viability.md` 8.7, FR-VIA-033 | `Validate` |
| a warm-start amount that overflows its bound, a duplicate key, or an unknown reference | `ErrOverflow`, `ErrDuplicateID`, `ErrUnknown*` | `spec-viability.md` 8.7, FR-VIA-041 | `Validate` |
| **viability**, each with the `*ViabilityError` rule named in the next four rows: `errors.Is(err, ErrNotViable)` matches every one (7.6) | `ErrNotViable` | `spec-viability.md` 4, 10.1 | `Validate` |
| rule `unproducible_in_realm`: an upkeep good, or an input or tool of a recipe that exists in a region where a role lists it, that no region can produce with a positive yield | `ErrNotViable` | FR-VIA-004 | `Validate` |
| rule `role_cannot_work`: a role that lists recipes, with households in a region where none of its recipes has a positive yield | `ErrNotViable` | FR-VIA-005 | `Validate` |
| rule `cycle_not_productive`: a strongly connected component of the material graph with no surplus (the Hawkins-Simon test, with the best yield across regions) | `ErrNotViable` | FR-VIA-006 | `Validate`, size-gated |
| rule `labour_short`: the pooled free-trade programme has a surplus factor below 1 at full labour; rule `storage_gap`: an upkeep good with a shelf life below its longest supply gap | `ErrNotViable` | FR-VIA-007, FR-VIA-008 | `Validate`; `labour_short` size-gated |
| the region graph: no region (`ErrNoRegions`), a self edge (`ErrSelfEdge`), an unknown region or commodity in an edge or a price override, a duplicate edge or override, a `Distance` outside `[1, MaxDistance]`, a price outside `[1, MaxPrice]`, more than `MaxEdges` edges | `ErrNoRegions`, `ErrSelfEdge`, `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrDuplicateID`, `ErrOutOfBounds`, `ErrTooMany` | `spec-regions.md` 9.1, FR-REG-001 to 013, 033 | `Validate` |
| a money flow: a field outside its bound or an unknown `Kind` or `Per`; a wrong field combination; a flow ID equal to a built-in reason or a duplicate; an unknown role or region; a pool target while the estates module is off | `ErrOutOfBounds`, `ErrFlowShape`, `ErrReasonCollision`, `ErrDuplicateID`, `ErrUnknownRole`, `ErrUnknownRegion`, `ErrModuleDependency` | `spec-money-supply.md` 12.1, FR-MON-034 to 038 | `Validate` |
| a population schedule with no knot or more than `MaxKnots`, knot ratios not strictly increasing or out of bound, a `Rate` outside 0 to 10 000, `MinHouseholds`, `BirthStake`, `BirthScale`, `DeathScale` or `HouseholdCap` out of bound; an unknown or duplicate region row (the rules of `spec-regions.md` 5.3.1, 5.3.2, not restated) | `ErrOutOfBounds`, `ErrUnknownRegion`, `ErrDuplicateID` | `spec-population.md` 9.1, FR-POP-025 to 027 | `Validate` |
| a transport mode: a field outside its bound, `UnitMass` and `UnitBulk` both 0, a duplicate mode or feed commodity, an unknown commodity, a `Vehicle` without `Durability`, a wrong field combination for a hired or a self-carried mode, the module on while the region graph is off, a product that overflows | `ErrOutOfBounds`, `ErrDuplicateID`, `ErrUnknownCommodity`, `ErrVehicleNotDurable`, `ErrModeShape`, `ErrModuleDependency`, `ErrOverflow` | `spec-transport.md` 12.1 | `Validate` |
| the resources module: a field outside its bound, an unknown resource or region in a row, a duplicate resource or row or a resource twice in the draws of a recipe, a wrong shape or rate combination, a bad abundance curve, an unknown season, a product that overflows | `ErrOutOfBounds`, `ErrUnknownResource`, `ErrUnknownRegion`, `ErrDuplicateID`, `ErrResourceShape`, `ErrCurve`, `ErrUnknownSeason`, `ErrOverflow` | `spec-resources.md` 13.2 | `Validate` |
| the information module: `InfoSpeed`, `RumourErrorBP` or `RumourMaxAge` outside its bound; sources times `RumourMaxAge` above `MaxRumourCells`, or an informed role with too many candidates; `Role.Informed` while the module is off | `ErrOutOfBounds`, `ErrTooMany`, `ErrModuleDependency` | `spec-information.md` 9.1, 9.2 | `Validate` |
| the subsistence module: `EnterTicks`, `ExitTicks` or `RetryJobs` outside its bound, a list longer than `MaxOwnUseRecipes`, `ExitTicks` not above `EnterTicks`, an own-use recipe that is unknown, twice in a list, with `Inputs` or `Requires`, not `Sited`, or also on the same role's `Recipes`, a row for an unknown or twice-named role, a recipe with `Draws` while the resources module is off | `ErrOutOfBounds`, `ErrOwnUseBand`, `ErrUnknownRecipe`, `ErrUnknownRole`, `ErrDuplicateID`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited`, `ErrOwnUseOverlap`, `ErrModuleDependency` | `spec-subsistence.md` 13.2 | `Validate` |
| the migration module: a field outside its bound (a missing `MoveSpeed` included), a row share above 10 000, `Modules.Migration` while `Modules.Information` is nil, a row for an unknown or twice-named region, more rows than `MaxRegions`, a mobile role with too many candidates, a product that overflows | `ErrOutOfBounds`, `ErrModuleDependency`, `ErrUnknownRegion`, `ErrDuplicateID`, `ErrTooMany`, `ErrOverflow` | `spec-migration.md` 12.1, 12.2 | `Validate` |
| the merchant module: a field outside its bound, `FeeShareSpreadBP` above `MerchantFeeShare` or with their sum above 10 000, an unknown mode or a mode twice in `Modes`, a merchant block on a role with recipes, `Strategy` `"merchant"` with no block, a block with `Informed` false, a trade input not in `Trades`, an empty cargo set, a self-carried mode of `Crew` above 1, `Consigns` with no `TakeUnits`, a merchant role with a group of scale above 1, a block while transport or information is off, the module while transport is off, a hired mode or `Consigns` while `Modules.Merchants` is absent | `ErrOutOfBounds`, `ErrUnknownMode` (a root sentinel of the transport module, reused here), `ErrDuplicateID`, `ErrMerchantShape`, `ErrMerchantWeight`, `ErrModuleDependency` | `spec-merchants.md` 14.1 | `Validate` |
| the external module: a field outside its bound, a side set on an absent side, an unknown region, commodity or season, a duplicate row, an outside node that has agents, crossed quotes, a gate that is not a gate, a flow ID equal to one of the six reasons, the module on while the region graph is off | `ErrOutOfBounds`, `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrUnknownSeason`, `ErrDuplicateID`, `ErrOutsideNodeInhabited`, `ErrCrossedQuotes`, `ErrNotAGate`, `ErrReasonCollision`, `ErrModuleDependency` | `spec-external-sector.md` 14.1 | `Validate` |
| the settlements module: an `Enter` outside `[0, MaxHouseholds]`; the first class with an `Enter` other than 0 (rule `first_class_open`); two classes with one `Enter` or `Enter` not strictly increasing in the sorted order (rule `enter_order`); `BandBP`, `Dwell` or `ThresholdBP` outside its bound; a limit `Value` outside `[0, MaxQuantity]`; a class ID, a flag or a limit name that is not an ID; a class, a flag or a limit twice, two rows for one region; more than `MaxSettlementClasses` classes, `MaxClassFlags` distinct flags, `MaxClassLimits` distinct limits or `MaxRegions` rows, or no class (rule `no_class`); a row for an undefined region or an outside node; a `Floor` that names no class; a `Role.Needs` that no class carries; a `Role.Ceiling` that no class carries; a `Needs` or `Ceiling` while the module is off | `ErrOutOfBounds`, `ErrInvalidID`, `ErrDuplicateID`, `ErrTooMany`, `ErrUnknownRegion`, `ErrOutsideNode`, `ErrUnknownSettlementClass`, `ErrUnknownSettlementFlag`, `ErrUnknownLimit`, `ErrModuleDependency` | `spec-settlements.md` 9.1 | `Validate`; the rules that read a role run with the roles |
| the storage module: a class ID or a role row twice (rule `duplicate_id`); a row for an unknown role (`unknown_role`); a class with both capacities 0 or a capacity outside `0 .. MaxRoom` (`store_class_shape`); a `Base` outside `0 .. MaxRoom`, more than `MaxStoreClasses` classes or more than `MaxStoresPerAgent` stores in a row (`room_bound`); a row's `Stores`, or a stores target of a flow, that names a class not configured (`unknown_store_class`); a `Needs` that no settlement class carries (`needs_flag`) or a `Needs` while `Modules.Settlements` is absent (`needs_module_off`); stores listed for a role whose resolved base room is `nil` (`store_on_unlimited_role`); a limited role whose base room per household is below one tick of its upkeep (`room_below_upkeep`), whose `StartStock` or warm-start stock has a load above the room and the given stores (`room_below_start_stock`), or whose warm-start job does not fit (`room_below_warm_job`); a flow that targets stores while the module is off (`stores_target_off`) | `ErrDuplicateID`, `ErrUnknownRole`, `ErrStoreShape`, `ErrOutOfBounds`, `ErrUnknownStoreClass`, `ErrUnknownSettlementFlag`, `ErrModuleDependency`, `ErrNotLimited`, `ErrRoomShape` | `spec-storage.md` 13.1 | `Validate`; `room_below_warm_job` at construction, once the warm start is applied (`NewWorld` returns it) |
| the employment module: a field outside its bound; a hire-list entry that is not a recipe; a recipe twice in a hire list; an employer block on a role with an empty `Recipes`, an empty hire list, a hire-list recipe not on `Role.Recipes`, a role both `Hireable` and an employer; an employer role with a group of scale above 1; `Hireable` or an employer block while the module is off | `ErrOutOfBounds`, `ErrUnknownRecipe`, `ErrDuplicateID`, `ErrEmploymentShape`, `ErrEmployerWeight`, `ErrModuleDependency` | `spec-employment.md` 14.1 | `Validate` |
| the tenure module: the module while `Modules.Resources` is nil; a term or a holding naming an unknown region, resource or role, or an outside node; two terms for one (region, resource), two holdings for one (region, resource, holder), a role whose ID is the word `pool`; a field outside its bound, more than `MaxHoldings` holdings; a `ShareBP` outside 1 to 10 000, holdings whose total on one endowment exceeds 10 000; a holding whose largest agent would hold more than `MaxHolderShareBP`; a money flow whose ID is `rent` or `title_sale`; a term's rent product that overflows | `ErrModuleDependency`, `ErrUnknownRegion`, `ErrUnknownResource`, `ErrUnknownRole`, `ErrOutsideNode`, `ErrDuplicateID`, `ErrOutOfBounds`, `ErrTitleShares`, `ErrTitleCap`, `ErrReasonCollision`, `ErrOverflow` | `spec-land.md` 11.1 | `Validate` |
| the credit module: a field outside its bound, `MinPrincipal` or `MaxClaims` below 1; a `CreditTerms` with both `Lend` and `Borrow` or with neither; a role with a credit block while the module is off; a money flow whose ID is `loan_out`, `loan_repay` or `loan_interest`; a `rate_bp` of a `Lend` block or a `max_rate_bp` of a `Borrow` block absent from a file (the generic rule of FR-CFG-008: a rate of 0 is legal and is written, so an omission is never a silent interest-free loan); an unknown role or a duplicate role row | `ErrOutOfBounds`, `ErrCreditShape`, `ErrModuleDependency`, `ErrReasonCollision`, `ErrMissingField`, `ErrUnknownRole`, `ErrDuplicateID` | `spec-credit.md` 15.1 | `Validate` |
| the risk module: a route row with `Modules.Transport` absent or a store row with `Modules.Storage` absent (rule `risk_module_dependency`); a `From`, `To` or `Region` naming no region (`unknown_region`); a pair that is not a configured edge (`unknown_edge`); a `Mode` that is not configured (`unknown_mode`); a store `Class` that is not configured (`unknown_store_class`); a route row's edge with an outside node at an end, or a store row's region that is an outside node (`risk_outside_node`); a `StrikeBP` or `SeverityBP` outside `1 .. 10 000` (`risk_row_shape`); a key twice (`duplicate_id`); more than `MaxRiskRows` rows in a table (`risk_bound`) | `ErrModuleDependency`, `ErrUnknownRegion`, `ErrUnknownEdge`, `ErrUnknownMode`, `ErrUnknownStoreClass`, `ErrOutsideNode`, `ErrRiskShape`, `ErrDuplicateID`, `ErrOutOfBounds` | `spec-risk.md` 13.1 | `Validate` |
| the infrastructure module: a class field set that its kind does not use or a required one missing, `Toward` on a clause of a class that is not `route`, both `Builds` and `Repairs` on a recipe, a class in `Needs` that is not a site or appears twice, an output of a build or repair recipe that is not `Discard`, a store row with no `Role`; a field outside its bound, more than `MaxRecipeNeeds` needs; a class twice, two initial rows with one key; a clause or an initial row naming an undefined class; a route, ground or store class naming an undefined mode, resource or store class; an initial row's `Region`, `Toward` or `Role` unknown; `Toward` equal to `Region`; an outside node in an initial row; a route row or clause with no edge; a recipe with `Builds` or `Repairs` and no input; an initial row whose role has no agent in the region, or a store row whose role has no base room; initial rows above `MaxFacilities` or an agent above `MaxFacilitiesPerAgent`; a route class with transport off, a ground class with resources off, a store class with storage off, a `Needs` flag with settlements off, a clause with the module off; a `TraineeEligible` recipe with a clause; a product that overflows | `ErrFacilityShape`, `ErrOutOfBounds`, `ErrDuplicateID`, `ErrUnknownFacilityClass`, `ErrUnknownMode`, `ErrUnknownResource`, `ErrUnknownStoreClass`, `ErrUnknownRegion`, `ErrUnknownRole`, `ErrSameRegion`, `ErrOutsideNode`, `ErrUnknownEdge`, `ErrFacilityNoCost`, `ErrFacilityHolder`, `ErrFacilityCount`, `ErrModuleDependency`, `ErrEligibleHasTool`, `ErrOverflow` | `spec-infrastructure.md` 13.1 | `Validate`; `ErrRecipeNoOutput` is waived for a recipe with `Builds` or `Repairs` |
| the insurance module: a field outside its bound, `ValueCapBP`, `MinSum` or `MaxPolicies` below 1; a `CoverTerms` with both blocks or neither; the module while `Modules.Risk` or `Modules.Credit` is nil, a role with a cover block while the module is off, an `Insure` block with `MinLossBP` above 0 and no transport or with `StoreShareBP` above 0 and no storage; a money flow whose ID is `premium` or `payout`; a required field whose bound admits 0 (`loading_bp`, `store_rate_bp`, `min_loss_bp`, `ruin_share_bp`, `max_loading_bp`, `max_store_rate_bp`) absent from a file (the generic rule of FR-CFG-008); an unknown role or a duplicate role row | `ErrOutOfBounds`, `ErrCoverShape`, `ErrModuleDependency`, `ErrReasonCollision`, `ErrMissingField`, `ErrUnknownRole`, `ErrDuplicateID` | `spec-insurance.md` 14.1 | `Validate` |
| the polity module: a jurisdiction ID that is not an ID or is twice, two membership rows for one region; more than `MaxJurisdictions`; a `Parent` or a member row naming an undefined jurisdiction; a member row or a `Seat` naming an undefined region; a parent chain that returns to the jurisdiction; `MaxPolityDepth` or more ancestors, a `StartMoney` outside `[0, MaxMoney]`; a `Seat` that is not a direct member of the jurisdiction or of one of its descendants; a `Seat` that is an outside node; a `StartMoney` above 0 with no seat | `ErrInvalidID`, `ErrDuplicateID`, `ErrTooMany`, `ErrUnknownJurisdiction`, `ErrUnknownRegion`, `ErrJurisdictionCycle`, `ErrOutOfBounds`, `ErrSeatNotCovered`, `ErrSeatOutsideNode`, `ErrStartWithoutSeat` | `spec-polity.md` 9.1 | `Validate`; the rules run in the order of 9.1 there |
| the fiscal module: `Modules.Fiscal` while `Modules.Polity` is nil or has no governed jurisdiction, a pool target while the estates module is off, `import` or `export` scope while the external module is off, a toll while transport is off; more than `MaxFiscalFlows` flows; an ID that is not an ID or is repeated across the six kinds; an ID equal to a built-in reason, a flow of the money-supply module or a reason a later spec declares; an undefined `Authority` or target jurisdiction; an `Authority` or target treasury with no seat; a region the authority does not cover, a target treasury that is not a proper descendant; an undefined role, region, commodity, mode or season; a field outside its bound, a `Per`, `Payer`, `Scope`, `On` or target `Kind` that is not one of its values, `Until` not above `Start`; a wrong field combination of a tax, a payout, a duty, a purchase or a release | `ErrModuleDependency`, `ErrTooMany`, `ErrInvalidID`, `ErrDuplicateID`, `ErrReasonCollision`, `ErrUnknownJurisdiction`, `ErrNotGoverned`, `ErrNotCovered`, `ErrUnknownRole`, `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrUnknownMode`, `ErrUnknownSeason`, `ErrOutOfBounds`, `ErrFiscalShape` | `spec-fiscal.md` 15.1 | `Validate`, after the polity's rules |
| the institutions module: an ID that is not an ID or is twice, two rights with one (activity, subject, scope key), two controls with one (commodity, scope key), two rate limits with one scope key; more than `MaxBodies`, `MaxRights` or `MaxControls`; a `Jurisdictions` entry or a right, control or rate limit keyed by a jurisdiction while `Modules.Polity` is absent, a `Needs` while `Modules.Settlements` is absent, a rate limit while `Modules.Credit` is absent; an undefined role, region, jurisdiction, recipe, commodity or flag; a right's `Body` that names no body; an outside node in a scope; a body with no role or no region, a (role, region) in the scope of two bodies; a right with an activity outside the four, a subject of the wrong kind, both or neither of `Region` and `Jurisdiction`, a `Who` other than `principal` on an activity other than `produce`; a control with both or neither of `Region` and `Jurisdiction`, a `FloorBP` above a `CapBP`, a `FloorBP` on a rate limit; a `Places`, `MaxTrainees` or level outside its bound | `ErrInvalidID`, `ErrDuplicateID`, `ErrTooMany`, `ErrModuleDependency`, `ErrUnknownRole`, `ErrUnknownRegion`, `ErrUnknownJurisdiction`, `ErrUnknownRecipe`, `ErrUnknownCommodity`, `ErrUnknownSettlementFlag`, `ErrUnknownBody`, `ErrOutsideNode`, `ErrBodyShape`, `ErrBodyOverlap`, `ErrRightShape`, `ErrControlShape`, `ErrOutOfBounds` | `spec-institutions.md` 10.1 | `Validate`; the rules run in the order of 10.1 there |
| the policy module: `Modules.Policy` with `Modules.Polity` nil or with no jurisdiction, a `flow` lever with neither money supply nor fiscal on, a `transit` lever with transport off; more than `MaxActors`, `MaxLevers`, `MaxScheduleSteps` or `MaxLeverKeys`; an actor, lever, step or reaction repeated, two levers of one actor naming one key; an undefined `Jurisdiction` or `Against`; two actors on one jurisdiction (rule `one_actor_per_jurisdiction`); a `Kind` that is not offered, a field outside its bound, `Min > Max`, a default outside `[Min, Max]`; `Schedule`, `Cycle` or `Rules` with the wrong policy (`policy_data`); the key fields of a kind wrong (`lever_key`); an undefined flow, edge, region or mode; a `transit` lever with equal or outside-node ends (`transit_pair`); a key outside the reach of 3.5 there; an `Against` that names the actor's own jurisdiction, an ancestor or a descendant (`against_nested`); a schedule step or a reaction that is wrong (`schedule_lever`, `schedule_value`, `schedule_step`, `rule_lever`, `rule_gauge`, `rule_reads`); a `treasury` gauge on a jurisdiction with no seat; a key held by the levers of two actors; a `Policy` name that is neither built in nor registered (`NewWorld` only, 7.1.4) | `ErrModuleDependency`, `ErrTooMany`, `ErrDuplicateID`, `ErrUnknownJurisdiction`, `ErrPolicyShape`, `ErrOutOfBounds`, `ErrUnknownFlow`, `ErrUnknownEdge`, `ErrUnknownRegion`, `ErrUnknownMode`, `ErrLeverNotInReach`, `ErrNotGoverned`, `ErrLeverShared`, `ErrUnknownPolicy` | `spec-policy.md` 11.1 | `Validate`, in the order of the table there; `ErrUnknownPolicy` in `NewWorld` only |
| the saving module: a field outside its bound (`Count` of a `Worth` row included), a `RepairBelowBP` above 10 000; a `Saver` block while `Modules.Saving` is absent, a `Worth` row while `Modules.Infrastructure` is absent; a `Worth` row that names an undefined class; a class twice in one role's `Worth`. The query `Saving(agent)` for an agent whose role has no block is `ErrNotSaver` at run time | `ErrOutOfBounds`, `ErrModuleDependency`, `ErrUnknownFacilityClass`, `ErrDuplicateID`, `ErrNotSaver` (a call) | `spec-saving-and-investment.md` 11.1 | `Validate` |
| the technology module: a field of a mechanism that is off that is non-zero, a field that an on mechanism requires that is 0, or an empty `Recipes` (3.1.2 there); a recipe in two fields; a recipe of a field that is not configured; two fields with one ID; a field outside its bound (a `Label` above `MaxLabelBytes` included); more than `MaxTechFields` fields or `MaxFieldRecipes` recipes in one. A `TechFieldID` outside `[0, F)` in a call is `ErrUnknownTechField` | `ErrTechShape`, `ErrTechRecipeOverlap`, `ErrUnknownRecipe`, `ErrDuplicateID`, `ErrOutOfBounds`, `ErrTooMany`, `ErrUnknownTechField` (a call) | `spec-technology.md` 10.1 | `Validate` |

The structural rules of 7.3 and the producibility rule of 7.5 are errors. A rule
that appears in a spec with no row here is not enforced until it has one (FR-CFG-028
and the registry test).

#### 7.4.2 Warnings

| Warning | Condition | Spec |
|---------|-----------|------|
| `cold_start_lockstep` | **informational**: a role with a recipe of `Duration` above 1 and more than one agent in a group, with no warm start or with a warm start that gives the role no job row, so the first jobs of the group start and complete together (STB-L22, WLD-A3). A count, not a number to calibrate; it never fails validation | `spec-world.md` 15.2 |
| `no_upkeep` | no upkeep at the world, role or region level, so no household can starve | `spec-agents.md` 5.2, 13 |
| `commodity_untraded` | a commodity in the tradable set of no role, that is not only a discard output | `spec-agents.md` 13 |
| `upkeep_unsourced` | an upkeep commodity that is not raw, that no recipe makes and that no `StartStock` supplies | `spec-agents.md` 13 |
| `start_stock_below_target` | a role's `StartStock` of an upkeep or input commodity is below one tick of its target | `spec-agents.md` 13 |
| `reserve_exceeds_money` | a role's `StartMoney` is below `ReserveTicks` times the cost of its effective upkeep at the region's reference prices | `spec-agents.md` 13 |
| `strategy_default_unused_trades` | a role has a `Trades` entry but uses the default strategy, which sells it at once | `spec-agents.md` 13 |
| `unused_recipe`, `unrunnable_recipe`, `durable_not_required`, `source_unbounded` | as `spec-production.md` 10.4 and FR-GDS-023 | `spec-production.md` 10.4 |
| `window_extended` | **informational**, reported once per role: the role's effective window is longer than the world's `WelfareWindow` because of its longest job; names the role, the window and the recipe. It replaces `job_longer_than_window`, which cannot occur because the window always exceeds the longest job (`spec-welfare-and-wages.md` 2.3.3, 2.3.5, 11.4). It never fails validation | `spec-welfare-and-wages.md` 11.4 |
| `region_not_self_sufficient` | in autarky a region's surplus factor is below 1, or an upkeep good is not producible in the region | `spec-viability.md` 10.2 |
| `subsistence_exceeds_expected_work`, `target_above_surplus` | the labour share to subsist is above the mean expected work share; the mean target ratio is above the nominal surplus factor | `spec-viability.md` 4.4, 10.2 |
| `storage_gap_input` | an input of an all-year activity has a shelf life below its supply gap | `spec-viability.md` 4.5 |
| `viability_check_skipped` | the size gate of 7.6 was exceeded; names the part not run | `spec-viability.md` 4.3, 10.2 |
| `cycle_cost_gain` | a cycle of recipes whose cost-loop gain `g = CostFloor / 10 000` times the cycle's full-cost share is 1 or more although the material cycle is productive (restated in 7.6 item 3 for the labour-inclusive unit cost) | `spec-stability.md` FR-STB-017, STB-L10 |
| `cost_floor_at_full_cost` | `CostFloor` is 10 000 or more: the floor equals the full unit cost at the steady state, and the wage loop has gain 1 (STB-L10, L52) | `spec-beliefs.md` FR-BEL-038 |
| `inexact_hour`, `season_shorter_than_tick` | `TicksPerDay` does not divide `HoursPerDay`; `DaysPerTick` above 1 and a season shorter than a tick (rule names proposed here; the calendar spec states the conditions without names) | `spec-calendar.md` 7.3 |
| `tool_unaffordable` | a role's starting money cannot buy its required tool at the reference price (rule name proposed here) | `spec-skill-barriers.md` 2.3 |
| `entry_share_window` | `EntryShare` times the longest effective welfare window exceeds 2 500 (a placeholder condition) | `spec-spawner.md` 12 |
| `renewal_with_population` | `Renewal` above 0 and the population module on | `spec-spawner.md` 12 |
| `no_open_role` | every role is `Closed` | `spec-spawner.md` 12 |
| `group_below_floor` | **at run time**, not in `Validate`: a group of weight above 1 whose agent count has fallen below `MinAgents` (a bounded `Warn`, `spec-spawner.md` 13.3); listed so the rule has one name | `spec-spawner.md` 12, FR-SPN-047 |
| `want_unused`, `want_unsourced`, `want_inelastic`, `wants_exceed_spend_cap`, `target_above_demand`, `class_without_income`, `region_taste_without_taste` | the conditions of `spec-demand.md` 10.2 | `spec-demand.md` 10.2 |
| `raw_without_source` | a commodity marked `Raw` that no existing activity with no inputs outputs and that no endowment, faucet or import supplies | `spec-viability.md` 4.2.1, 10.2 |
| `wants_exceed_labour` | the demand module is on and the programme with the wants has `lambda* < 1` while the upkeep-only one has `lambda* >= 1` (size-gated) | `spec-viability.md` 3.2, 10.2 |
| `region_isolated`, `graph_not_strongly_connected`, `edge_not_shortest`, `region_without_agents`, `override_equals_default` | the conditions of `spec-regions.md` 9.2 | `spec-regions.md` 9.2 |
| `no_flows`, `faucet_without_sink`, `pair_unbalanced`, `levy_below_upkeep`, `class_empty`, `grant_to_pool_no_outlet` | the conditions of `spec-money-supply.md` 12.2 | `spec-money-supply.md` 12.2 |
| `no_replacement_ratio`, `schedule_direction`, `replacement_above_closed_ceiling`, `population_gain`, `no_fixed_factor`, `population_without_upkeep` | the conditions of `spec-population.md` 9.2; `renewal_with_population` (above) is raised only for a non-default `Renewal` above 0 with the module on. **Informational**, never failing: `replacement_ratio` and `renewal_ignored` (the module on and `Renewal` at its default) | `spec-population.md` 9.2, 4.6.1 |
| `transport_without_routes`, `haul_unsourced`, `haul_never_spoils`, `haul_durable`, `vehicle_wears_in_one_trip`, `trip_longer_than_window`, `route_too_slow_for_shelf_life` | the conditions of `spec-transport.md` 12.2 | `spec-transport.md` 12.2 |
| `resource_unused`, `endowment_unused`, `draw_impossible`, `abundance_not_monotone`, `regen_per_tick_high`, `resource_inert`, `raw_source_exhaustible`, `resource_overshoot`, `resource_unstable_plan`, `endowment_season_peak`, `no_comparative_advantage` | the conditions of `spec-resources.md` 13.3; `resource_short` (an error) and the informational `endowment_runway` are reported by the solver (`spec-viability.md`) | `spec-resources.md` 13.3 |
| `subsistence_empty`, `own_use_no_upkeep`, `own_use_beats_market` (needs the solver, size-gated), `own_use_partial` (informational), `own_use_trap`, `own_use_exit_unreachable`, `own_use_slower_than_ladder`, `own_use_not_available` | the conditions of `spec-subsistence.md` 13.3 | `spec-subsistence.md` 13.3 |
| `migration_without_routes`, `migration_no_mobile_role`, `migration_cohort`, `migration_provision`, `migration_far`, `migration_flow_window`, `migration_uncapped` | the conditions of `spec-migration.md` 12.3 | `spec-migration.md` 12.3 |
| `merchant_spread_narrow`, `merchant_fee_share_high`, `merchant_fee_share_low`, `merchant_no_consignee`, `lane_without_consignee`, `cargo_never_delivered`, `consigns_without_cargo`, `merchants_one_agent` | the conditions of `spec-merchants.md` 14.2 | `spec-merchants.md` 14.2 |
| `information_without_readers`, `information_without_routes`, `rumour_horizon_short`, `informed_without_sources` | the conditions of `spec-information.md` 9.3 | `spec-information.md` 9.3 |
| `node_without_gate`, `gate_edge_wrong_direction`, `line_without_side`, `window_all_closed`, `quote_outside_reference_band`, `walk_faster_than_beliefs`, `outside_faucet_without_sink`, `outside_drain_without_source`, `outside_arbitrage`, `import_only_source`, `line_untraded` | the conditions of `spec-external-sector.md` 14.2 | `spec-external-sector.md` 14.2 |
| `needs_unmet_at_start`, `ceiling_below_start`, `dwell_below_window`, `band_one_household`, `threshold_collapsed`, `class_unreachable`, `flag_unused`, `limit_unused` | the conditions of `spec-settlements.md` 9.2; `flag_unused` and `limit_unused` are informational | `spec-settlements.md` 9.2 |
| `no_limited_role`, `room_below_cover`, `room_below_recipe`, `store_class_unused`, `store_locked_at_start`, `store_levy_below_upkeep` | the conditions of `spec-storage.md` 13.2 | `spec-storage.md` 13.2 |
| `employment_no_employer`, `employment_no_worker`, `no_employer_in_region`, `employer_no_outlet`, `hire_never_profitable`, `wage_below_upkeep`, `payroll_exceeds_money`, `inkind_wage_ignored`, `worker_share_full` | the conditions of `spec-employment.md` 14.2; `wage_below_upkeep` is informational | `spec-employment.md` 14.2 |
| `no_titles`, `rent_without_titles`, `titles_without_rent`, `holding_unused`, `role_locked_out`, `entitlement_below_unit`, `rent_above_output` (informational), `rent_hoarded`, `rent_step_inert`, `rent_off_dual` | the conditions of `spec-land.md` 11.2; `rent_off_dual` is reported by the solver (`spec-land.md` 5.6.2, 12) | `spec-land.md` 11.2 |
| `no_credit_roles`, `credit_no_lender`, `credit_no_borrower`, `rate_above_limits`, `term_exceeds_lenders`, `cap_below_horizon`, `lender_without_cash`, `lender_without_income`, `lender_keep_below_term`, `no_shield`, `credit_no_upkeep`, `credit_free`, `interest_rounds_to_zero` | the conditions of `spec-credit.md` 15.2; informational, none fails validation | `spec-credit.md` 15.2 |
| `no_risk_rows`, `risk_edge_off_path`, `risk_store_class_unowned`, `risk_certain` | the conditions of `spec-risk.md` 13.2 | `spec-risk.md` 13.2 |
| `facility_unused`, `facility_unbuildable`, `decay_without_repair`, `capacity_never_binds`, `route_unused_mode`, `facility_gate_closed`, and `unrunnable_recipe` (above, extended) | the conditions of `spec-infrastructure.md` 13.2; `unrunnable_recipe` also fires for a recipe that `Needs` a class that is `facility_unbuildable` and has no initial facility in a region where the recipe is otherwise runnable | `spec-infrastructure.md` 13.2 |
| `no_cover_roles`, `cover_no_underwriter`, `cover_no_insured`, `cover_without_risk`, `loading_above_limits`, `cover_below_minimum`, `underwriter_without_income`, `underwriter_cap_above_money`, `claims_below_policies`, `no_shield_underwriter`, `value_cap_unbound` | the conditions of `spec-insurance.md` 14.2 | `spec-insurance.md` 14.2 |
| `jurisdiction_empty`, `region_unassigned` | the conditions of `spec-polity.md` 9.2 | `spec-polity.md` 9.2 |
| `no_fiscal_flows`, `treasury_unfunded`, `treasury_idle`, `tax_below_upkeep`, `payout_pool_no_outlet`, `purchase_unbounded`, `release_unsourced`, `duty_stack_above_whole`, `toll_no_border`, `fiscal_class_empty` | the conditions of `spec-fiscal.md` 15.2. The names `no_fiscal_flows` and `fiscal_class_empty` are the module's own: `no_flows` and `class_empty` are the rows of `spec-money-supply.md` 12.2 above and keep their meaning | `spec-fiscal.md` 15.2 |
| `body_empty`, `places_below_start`, `right_unreachable`, `right_over_upkeep`, `train_right_no_training`, `cap_below_band`, `floor_above_band`, `cap_below_seed_cost`, `rate_limit_below_floor`, `control_inert`, `body_needs_unmet_at_start` | the conditions of `spec-institutions.md` 10.2; `control_inert` and `body_needs_unmet_at_start` are informational | `spec-institutions.md` 10.2 |
| `no_actors`, `actor_no_levers`, `lever_fixed`, `lever_no_keys`, `reads_unused`, `rule_undamped`, `rule_inert`, `cadence_long`, `schedule_dense` | the conditions of `spec-policy.md` 11.2 | `spec-policy.md` 11.2 |
| `saver_no_project`, `worth_unused`, `worth_on_site`, `project_below_hurdle`, `safety_below_slack`, `saving_target_exceeds_money`, `repair_never`, `spread_below_margin` | the conditions of `spec-saving-and-investment.md` 11.2; informational, none fails validation | `spec-saving-and-investment.md` 11.2 |
| `field_inert`, `field_unused`, `diffusion_no_contact`, `technology_past_maximum_yield`, `trend_outpaces_schedule`; the informational diagnostic `technology_stock_share` | the conditions of `spec-technology.md` 10.2, 10.3.4; `unrunnable_recipe` is unchanged | `spec-technology.md` 10.2 |

### 7.5 Producibility

The rule "a supply chain with a cycle that has no raw-material source" (architecture.md
8.5) and the rule "no commodity is consumed that cannot be made" are one
fixpoint:

1. A commodity is **sourced** if a boundary faucet, an endowment or an external
   import can supply it, or if some recipe with no inputs outputs it. These come
   from the specs for money supply, resources and the external sector; until they
   exist, a recipe with no inputs, or a commodity marked `Raw`
   (`spec-goods.md` 6.2), is the source. A `Sited` recipe with no inputs and with
   `Draws` counts as a source (`spec-production.md` 10.5): its endowment is the
   boundary, and the resources module must be on (`ErrModuleDependency`). Whether the endowment can start at least once is the test of `spec-resources.md` 9.2, used by V1; an import line with a sell side, a gate and a window not closed in every season is also a source (`spec-external-sector.md` 10.1). Either source satisfies `raw_without_source` (`spec-viability.md` 4.2.1).
2. Repeat: a commodity is **producible** if it is sourced, or if some recipe that
   outputs it has every input producible.
3. When no more commodities are added, every commodity that some recipe, upkeep
   or tool *consumes* must be producible. A commodity that is not is reported
   with `ErrUnproducible`, naming one recipe whose inputs are not all producible.

A cycle (a recipe chain that feeds itself) with no source stays unproducible
because none of it is ever added, so the cycle is rejected by the same rule.
**The error names the cycle.** When a commodity is unproducible, the validator looks for
a strongly connected set of unproducible commodities that every recipe making one of them
reads from (a required tool counts as an input), and reports one such cycle, the
commodities and the recipes in ID order, in the `Path` and the message: for example
`cycle with no source: tool_b <- make_tool_b <- refined <- refine <- mineral <-
dig_mineral <- tool_a`. The commonest authoring mistake is a source recipe that needs a
tool made from its own product (`spec-production.md` 10.6); naming the cycle shows it. A
commodity that is unproducible but on no cycle is reported with the recipe whose input is
missing, as before. The marker `Raw` makes a commodity sourced here and in V1
(`spec-viability.md` 4.2.1); the warning `raw_without_source` says when nothing real
stands behind it. The
algorithm is deterministic (it iterates commodities in ID order) and runs in
`O(commodities * recipes)`.

An own-use recipe (`spec-subsistence.md` 3.6) may still count as a source in this structural pass, which looks only at the graph; V1 does not count it: an upkeep good made only by an own-use recipe is `unproducible_in_realm` (7.6).

This fixpoint looks only at the recipes' graph. It is blind to yield, region and
season: a crop that grows in no region is producible here and not in the sense of
7.6, which uses the yield tables. Both run; this one is the cheap structural pass.

### 7.6 Viability

The stability duty of the roadmap: validation rejects a chain that fails the
viability check of `spec-viability.md`. The check is specified there (sections 4
and 10); this section says how the framework runs it.

1. **Where.** `Validate` runs V1 (sourced and runnable) and V4 (storable) always,
   and V2 (productive cycles) and V3 (labour sufficient) when the size gate allows:
   the tableau would have at most `MaxViabilityCells` entries and the pivots would
   not pass `MaxViabilityPivots`. The size is computed before a tableau is built, with the checked arithmetic of `internal/checked`: `rows` is the commodities of the programme plus its regions plus one row per activity whose start capacity `J` times its duration `d` is below `TicksPerYear` (the capacity rows), `cols` is the activities plus one for the surplus factor plus the slack of each row, and `rows * cols` is compared with `MaxViabilityCells`. Above the gate, or on reaching `MaxViabilityPivots`, the programmes are skipped and the
   warning `viability_check_skipped` names the part not run (V2 of a component, V3 pooled, V3 per region, the nominal warnings or the cycle gain), so the validation time
   stays bounded (NFR-CFG-001). The check is **necessary and optimistic**: pooled
   free trade, the best yield across regions.
2. **The error.** A failure is `ErrNotViable` carried by a `*ViabilityError` with a
   `Rule` (`unproducible_in_realm`, `role_cannot_work`, `cycle_not_productive`,
   `labour_short`, `storage_gap`), a `Path`, the offending value and the number that
   decides it (the fields are those of 7.1.1). `errors.Is(err, ErrNotViable)` matches each (FR-CFG-033,
   FR-VIA-011), and `errors.As` finds the `*ViabilityError`; the framework's ordering (7.1.2) and its
   cut at 100 errors (7.1.3) apply to it. Phase 6 runs only on a configuration the earlier phases passed (7.1.6).
3. **The cycle gain** (`spec-stability.md` FR-STB-017, STB-L10, STB-L52; restated
   after labour entered the unit cost, `spec-production.md` 9, N-066). The unit cost
   is now the full cost: inputs, wear and labour. A rise in a price passes round a
   recipe cycle through the part of the full cost that the cycle carries, and a
   rise in the wage through the labour part, so there are two gains.
   - *The material (input) loop.* For a cycle, the **full-cost share** `s` is the
     share of the full unit cost of the cycle's product that is the value of the
     cycle's own inputs. At the zero-profit prices of the solver (value added equals
     the wage times the duration) `s = q_in / q_out`, the quantity ratio around the
     cycle (a product over the recipes of a longer cycle), so the old formula is
     the special case in which prices are the solver's. The gain is
     `g = CostFloor / 10 000 * s`, to be below 1. Validation computes `s` as
     `q_in / q_out` (it has no prices) in exact integer arithmetic. A cycle with
     `q_in / q_out` of 1 or more fails V2 (an error, `cycle_not_productive`). A
     productive cycle with `g` of 1 or more at the configured `CostFloor` (possible
     only when `CostFloor` exceeds 10 000 times `q_out / q_in`) is the warning
     `cycle_cost_gain`.
   - *The wage loop.* Price, income, wage, full cost, floor, price has gain
     `CostFloor / 10 000` times the labour share of the full cost where the floor
     binds, at most `CostFloor / 10 000`. It is below 1 for any `CostFloor` under
     10 000 and reaches 1 only at `CostFloor` of 10 000 or more with a labour share of
     1. It is flagged by its own warning `cost_floor_at_full_cost` (FR-BEL-038),
     not by `cycle_cost_gain`, because it needs no cycle of recipes: it exists in a
     chain with no cycle at all.
   - *Which cycles.* A component can hold many cycles, so the ratio is taken over the simple
   cycles whose every commodity has one maker in the component, a maker being a non-source
   activity (one with inputs) that outputs the commodity; a source activity supplies the
   commodity from outside the cycle and is not counted. `q_in` and `q_out` are the configured quantities, with no yield factor. The enumeration runs for at most
   `MaxViabilityCells` steps; a component past that is the "cycle gain" part of
   `viability_check_skipped`. Worked case: the seed cycle of the `one` pack, 40 `seed` in and 1 800 `staple` out, then 20
   `staple` in and 20 `seed` out, has `q_in / q_out = (40 / 1 800) * (20 / 20) = 1/45`, so it never
   warns inside the bound of `CostFloor`.
   The two tests use exact integer arithmetic. **FR-STB-017 follows** (owed to
   `spec-stability.md`): its formula should read `g = CostFloor / 10 000 * s` with
   `s` the full-cost share of the cycle's own inputs, `s = q_in / q_out` at the
   solver's prices, plus the wage-loop condition `CostFloor` below 10 000.
4. **What it does not do.** A chain that passes every rule may still be unviable in
   ways only a run shows (`spec-viability.md` 13.1). Passing validation is a
   **necessary** condition for a stable economy, not a sufficient one; stability is
   shown by the acceptance bands in `spec-stylised-facts.md`.

## 8. The effective configuration and its hash

8.1. The **effective configuration** is the `Config` after defaults and
canonicalisation (3.4), with every option that affects results applied. An option
that affects results is part of the effective configuration, not outside it:
`WithTargetSpread` (`spec-welfare-and-wages.md` 5.2) overrides the roles' spreads
and so changes every run. A stand-alone `Config.Validate` and the pure hash function of 8.5 see a `Config` as given and not the options, so they do not see `WithTargetSpread`, and `World.ConfigHash` differs from the hash of the same `Config` whenever that option was given. The seed, the logger, the observers and `WithAudit`
(`spec-ledger.md` 12.4) do not affect what the world *is* and are excluded; the seed travels in the checkpoint
instead (a checkpoint holds the seed and no stream state, ADR 0008). `WithReseed` is excluded too: it is a restore-time option that changes the continuation and not what the world is (`spec-checkpoint.md` 6.8). The cases settled here:

- **Strategy.** The *names* in `Role.Strategy` (`default`, `directed` or a registered name) are hashed; the code behind a name
  is not and cannot be (`spec-agents.md` 6.6). The registrations made with
  `WithStrategy` are options and are not hashed beyond the names the roles use.
- **The spawner, the clearing and the demand module.** The *name* in `World.Spawner` and the *name* in `World.Clearing` are hashed, not the
  code behind them (`spec-spawner.md` 3.1.3, `spec-clearing.md` 8, as for strategies; a world with a custom rule does not hash like one with the default). `Modules.Demand`, when present, is hashed with its wants, tastes and region tastes in
  canonical order (wants by ID, items by commodity, tastes by role then want, region
  tastes by region then want), every field except `Label`; `Role.Closed` and the
  spawner's parameters are hashed as any field.
- **The warm start.** `Config.WarmStart` is part of the configuration: when it is non-nil its prices, wages, groups and pools are hashed (`spec-viability.md` 8.1), and a non-nil `WarmStart` with no row is not the same configuration as a nil one (8.2.1). The solve
  options that produced it (`StockCoverTicks` and the rest, `spec-viability.md`
  11) are not fields and not hashed; only the table is.
- **Regions, edges and the modules.** `Edges` and `RegionPrices` are hashed in canonical order, every field, and encode to nothing when empty; `Region.Label` and `World.DistanceLabel` are not hashed (`spec-regions.md` 4.1.2). A module is hashed **only when it is on** (its pointer is non-nil), with every field it owns but the presentation-only ones (8.3), and a field that a module adds to a core struct (`Role.Informed`, `Region.PoolRelease`, a recipe's `Builds`) is hashed only when its module is on: an off module leaves no trace (8.2.3). A module's lists are in the canonical order of 3.4 and its rows by the key the owner spec names: `Modules.MoneySupply` with its flows by ID; `Modules.Population` with its schedules in knot order and its region rows by region; `Modules.Transport` with its modes by ID; `Modules.Resources` with its resources by ID (knots by share, seasons in the calendar's order), its endowment rows by region then resource, and `HorizonYears`; `Modules.Information` with its three fields; `Modules.External` with its nodes by region, lines by commodity, gates by region and season factors by season; `Modules.Subsistence` with its default list and its role rows by role (each list by recipe ID) and its three tunables; `Modules.Migration` with its nine fields and its `Regions` rows by region; `Modules.Merchants` with `MaxOpen`, and the `Role.Merchant` block (modes by ID); `Role.Fixed` as any role field when migration is on. The per-agent state (`OwnUse`, `OwnUseJobs`, journeys), the contracts, the shock scales, the flow scale, the transit factor, the birth and death scales, the drift, the carries and the cumulative totals are state and not hashed.
- **The modules of wave 5 and `Estates`.** Each is hashed when present, every field but the presentation-only ones (8.3), in canonical order (3.4): `Modules.Settlements` with its classes by `Enter` (each class's `Flags` by name and `Limits` by `Name`), `BandBP`, `Dwell` and its region rows by region, and `Role.Needs` and `Role.Ceiling`; `Modules.Storage` with `Base`, its role rows by role and its classes by ID, and `FlowTarget.Store`; `Modules.Employment` with `MaxOpen`, and `Role.Hireable` and the `Role.Employer` block (the hire list by recipe ID); `Modules.Tenure` with its terms by (region, resource), its holdings by (region, resource, holder), the pool before the roles, and its five tunables; `Modules.Credit` with its four tunables, and the `Role.Credit` block; `Modules.Risk` with its route rows by (`From`, `To`, `Mode`) and its store rows by (`Region`, `Class`); `Modules.Infrastructure` with its classes by ID and its initial rows by key, and the `Needs`, `Builds` and `Repairs` of a recipe; `Modules.Insurance` with its five tunables, and the `Role.Cover` block; `Modules.Polity` with its jurisdictions by ID and then its members by region; `Modules.Fiscal` with its six lists each by ID, every field but `Label`, and the seasons of a window in the calendar's order; `Modules.Institutions` with its bodies, rights, controls and rate limits by ID; `Modules.Policy` with every field of every actor by ID, the policy by **name** only (a registered implementation and the mailbox are not hashed); `Modules.Saving` with its three tunables, and the `Role.Saver` block; `Modules.Technology` with its fields by ID; `Modules.Estates` with no field, so that its presence is the whole of its contribution, and `Region.PoolRelease` and `Region.ReliefBelow`. The state of each module (the rent scales, the rate scales, the legal levels and right suspensions, the risk scales, the hire records, the claims, the titles, the facilities, the policies, the experience and the advance of a cell, a store of an agent) is state and is not hashed.
- **Yields.** `Yields` is hashed in canonical order (region, then recipe, and
  each profile's season entries sorted by season ID, as `spec-production.md` 6.1
  has it). The yield shocks of
  `spec-production.md` 6.8 are state, not configuration, and are saved in the
  checkpoint.

8.2. The **configuration hash** is the SHA-256 of a canonical binary encoding of
the effective configuration. The checkpoint stores it and a restore with a
different hash fails with `ErrCheckpointConfig` (ADR 0002, decision 4; `spec-checkpoint.md` 13.1). The encoding
is written by hand, with a version byte, and uses no reflection and no map, so equal
configurations give equal bytes on every platform. Its version is written in the header of a checkpoint (`spec-checkpoint.md` 5.1), a restore hashes the effective configuration with that version, and from 1.0 an implementation keeps the encoder of every version it has written (`spec-checkpoint.md` 9.2.1 item 4, FR-CHK-063).

**The rule of D5: an append that is off leaves no trace.** Nothing in the encoding
depends on how many fields a struct has or how many modules exist. A field that is at
its zero value is not written, and a module that is off is not written. So a pull
request that adds a field or a module, and whose default configuration leaves it at
zero or off, does not change the hash of any configuration that existed before it, and
`testdata/golden/off/` stays as it was (testing.md 7.1.1) by construction. This
replaces the presence byte per module of the first draft of this section, which moved
every hash whenever a module was added.

8.2.1. **The encoding, version 1.** The bytes are the version byte `0x01`, then the
encoding of `Config` as a struct. The SHA-256 is taken over exactly those bytes.

- **A struct** is a sequence of *pairs* and then a terminator. A pair is a field id
  (2 bytes, big-endian, from 1) and the value; pairs are in ascending id order; the
  terminator is the two bytes `0x00 0x00`. A field is a pair only if its value is not
  zero, and a presentation-only field (8.3) is never a pair.
- **Zero** is: the integer 0, `false`, the empty string, a nil or empty slice, a
  nil pointer, and a struct all of whose fields are zero. It is one rule for every
  field of every type, so no field has an "off value" of its own.
- **Values.** An integer or a named integer type (`Money`, `Price`, `Quantity`) is 8
  bytes, two's complement, big-endian. A `bool` is the byte `0x01` (false is never
  written). A string is its length as 8 bytes and then its bytes. A slice is its count
  as 8 bytes and then its elements in the order the field's rule gives (3.4; an ordered
  list as written), each encoded as a value of its element type: an integer, a string
  or a struct. A struct, or a pointer to a struct that is not nil, is its pairs and
  terminator.
- **A non-nil pointer is always written**, even when its struct is zero: the pair is
  the id, the empty pairs and the terminator. So `WarmStart: &WarmStart{}` and a nil
  `WarmStart` are two configurations with two hashes, and a module that is on and has
  no field (`Estates`) is the pair that names it.
- **`Modules`** is a struct like any other whose fields are the 25 pointers. The field
  id of a module is its position in the `Modules` row of 3.1, counted from 1 (`Demand` 1,
  `MoneySupply` 2, `Population` 3, `Transport` 4, `Resources` 5, `Information` 6,
  `External` 7, `Subsistence` 8, `Migration` 9, `Merchants` 10, `Settlements` 11 to
  `Technology` 24, `Estates` 25). A module that is off is a nil pointer and is not a
  pair. The ids do not depend on the order in which the modules are built.
- **A field a module adds to a core struct** (`Role.Informed`, `Region.PoolRelease`,
  a recipe's `Builds`) is a pair only when its module is on, whatever its value. With
  the module off it is not written and not validated for its value.

8.2.2. **Field ids.** A field's id is its position in the struct's declaration, counted
from 1 and not counting presentation-only fields, as 3.7.2 lists the fields. A field a
later module adds takes the next free id of its struct, in the pull request that adds it,
and that pull request adds a row to this paragraph; an id is never reused or moved. So
at Stage 0: `Config` has `World` 1, `Calendar` 2, `Commodities` 3, `Recipes` 4, `Yields`
5, `Roles` 6, `Regions` 7, `Edges` 8, `RegionPrices` 9, `Groups` 10, `WarmStart` 11 and
`Modules` 12. `WorldParams` (the type of `Config.World`) has its 45 fields as 3.7.2 lists them, `Scale` 1 to `Clearing` 45.
`Calendar` has `HoursPerDay` 1 to `SeasonStart` 8; `Month` and `Season` have `ID` 1,
`Days` 2. `Commodity` has `ID` 1, `Mass` 2, `Bulk` 3, `ShelfLife` 4, `Durability` 5,
`Classes` 6, `Raw` 7, `ReferencePrice` 8. `Item` has `Commodity` 1, `Quantity` 2;
`Output` has `Commodity` 1, `Quantity` 2, `CostShare` 3, `Discard` 4; `Tool` has
`Commodity` 1, `WearPerUse` 2. `Recipe` has `ID` 1, `Duration` 2, `Inputs` 3,
`Outputs` 4, `Requires` 5, `Sited` 6, `TraineeEligible` 7, `TraineeYield` 8.
`YieldProfile` has `Region` 1, `Recipe` 2, `HasAll` 3, `All` 4, `Seasons` 5;
`SeasonYield` has `Season` 1, `BP` 2. `Role` has `ID` 1, `Recipes` 2, `Upkeep` 3,
`Trades` 4, `StartMoney` 5, `StartStock` 6, `Strategy` 7, `Unskilled` 8,
`TargetRatio` 9, `TargetSpread` 10, `DefaultRatio` 11, `InKind` 12,
`ExpectedWorkTicks` 13, `MinWorkShare` 14, `TrainingTicks` 15, `MaxTrainees` 16,
`TeachingLoad` 17, `Closed` 18. `Region` has `ID` 1, `ReferenceBasket` 2, `ExtraUpkeep`
3, `CapitalReturn` 4, `EstateMarkdown` 5. `Group` has `Region` 1, `Role` 2,
`Households` 3, `Scale` 4. `WarmStart` has `Prices` 1, `Wages` 2, `Groups` 3, `Pools` 4;
`SeedPrice` has `Region` 1, `Commodity` 2, `Price` 3; `SeedWage` has `Region` 1,
`Wage` 2; `WarmGroup` has `Region` 1, `Role` 2, `Money` 3, `Stock` 4, `Jobs` 5;
`JobSpec` has `Recipe` 1, `InProgressBP` 2; `WarmPool` has `Region` 1, `Money` 2. A
struct has at most 65 534 fields.

8.2.3. **What an append must do.** A pull request that adds a module or a field obeys
three rules, and the pinned hash of 8.2.5 fails if it does not: (1) a module's pointer
takes its position in the `Modules` row and nothing else moves; (2) a field added to a
core struct takes the next id and its **default value is its zero value**, because a
default that is not zero would be written by every `DefaultX()` and so would move every
existing hash: a field whose natural "off" is not zero (a cap of 10 000 that means "no
cap") is declared so that zero is off, or belongs to a module; (3) a field a module adds
to a core struct is written only when the module is on (8.2.1). A change to the encoding
itself (a type, the terminator, an id) is a new version byte and a behaviour change
(architecture.md 6.6).

8.2.4. **Worked examples.** The bytes are in hex and the digests are the SHA-256 of
the whole byte string, written by a reference encoder in Go and again in Python, which
agree on every row. A count or a length is 8 bytes, big-endian.

| Configuration | Bytes | SHA-256 |
|---------------|-------|---------|
| `Config{}`: everything zero, every module off | `01` `0000` (the version, the terminator) | `fb50dc0717ff266cf9baf82b1ce7a1c2ef6d9247859680b11a19fb7077f5f222` |
| one region and nothing else, `Regions: [{ID: "north"}]` | `01` `0007` `0000000000000001` `0001` `0000000000000005` `6e6f727468` `0000` `0000` (version; `Regions` and its count; `ID` and its length and bytes; the region's terminator; the configuration's terminator) | `ed652dfd86904c18fd76feb17175e7efeef6d4afe94bec1e3b4f5a2e2409f013` |
| the same with `WarmStart: &WarmStart{}` | the row above with `000b` `0000` inserted before the last `0000` | `a2f375f381fced5c69544bf3dfb14446d513d6e0eaceb9574964bbfe8ea8a812` |
| the same with `Modules.Estates: &Estates{}`, a module on | the one-region row with `000c` `0019` `0000` `0000` inserted before the last `0000` (`Modules`; `Estates`, id 25, and its terminator; the terminator of `Modules`) | `0848fee29c503bda99a17acb2c44c42fbba8b730eadc2079e3d8123b2d3317ad` |
| the one-region row with a field appended later (`Region.PoolRelease`) at its off value 0 | the bytes of the one-region row, unchanged | `ed652dfd86904c18fd76feb17175e7efeef6d4afe94bec1e3b4f5a2e2409f013` |

The first row is the hash of an all-off configuration with nothing in it, the fourth
shows a module costing exactly the bytes that name it, and the last shows an append
costing nothing. A fuller configuration of one commodity `grain` (`Mass` 1, `Bulk` 1,
`ReferencePrice` 100), one recipe `grow` (`Duration` 1, one output of 1 `grain` with
`CostShare` 10 000), one role `farmer` (`Recipes` [`grow`], `StartMoney` 10 000,
`Strategy` `default`, `TargetRatio` 15 000, `DefaultRatio` 10 000,
`ExpectedWorkTicks` 250, `MinWorkShare` 4 000), one region `north`
(`ReferenceBasket` 100) and one group of 10 households of `farmer` in `north`, with the
world section left at zero (a configuration the validator would refuse, which still
has a hash, 8.5), encodes to 350 bytes:

```
010003000000000000000100010000000000000005677261696e000200000000
0000000100030000000000000001000800000000000000640000000400000000
000000010001000000000000000467726f770002000000000000000100040000
00000000000100010000000000000005677261696e0002000000000000000100
0300000000000027100000000000060000000000000001000100000000000000
066661726d657200020000000000000001000000000000000467726f77000500
000000000027100007000000000000000764656661756c740009000000000000
3a98000b0000000000002710000d00000000000000fa000e0000000000000fa0
000000070000000000000001000100000000000000056e6f7274680002000000
00000000640000000a0000000000000001000100000000000000056e6f727468
000200000000000000066661726d65720003000000000000000a00000000
```

and its SHA-256 is `87a8a42335e50ea33777b9290b1adc14c80a8b8a1451e31b72dd664f72c2f0f5`.

8.2.5. **The hash of the reference configuration is pinned.** The test of the
reference configuration (the pack `one` of `spec-reference-chain.md`, built in Go,
3.6; no module is on) compares its hash with a constant in the test file. A pull
request that moves the constant has broken 8.2.3 or changed the encoding on purpose, and
says which (testing.md 7.2). It is the earliest signal that an append was not off, before
any golden file exists. A new pack's hash is pinned the same way when the pack is.

8.3. Some fields are **presentation only**: a human-readable label, a display
ladder (`spec-currency.md` 2.2). They do not change any result, so they are not
hashed. A field is presentation only if and only if its doc comment says so. A
test changes every field of `Config` in turn and checks that the hash changes,
except for the fields on a short allow-list of presentation-only fields. The test
runs on a configuration with every module that is built switched on, because a
field of a module that is off is not hashed (8.2.1); at Stage 0 no module is built.
A new field that the encoder forgets therefore fails a test. The allow-list is
checked against the doc comments: a field whose comment says presentation only and is not
on the list fails, and so does a listed field whose comment does not say so. The allow-list is `Region.Label`, `World.DistanceLabel`, `Mode.Label`, `MoneyFlow.Label`, `Resource.Label`, `OutsideNode.Label` and the labels of the earlier specs, and, for the modules of wave 5, `StoreClass.Label`, `SettlementClass.Label`, `Jurisdiction.Label`, `FacilityClass.Label`, `TechField.Label`, `PolicyActor.Label`, the `Label` of a body, of a right, of a control and of a rate limit, and the `Label` of each of the six kinds of fiscal flow.

8.4. The file's layout (comments, whitespace, key order, and the split into
documents) does not affect the hash (2.5).

8.5. **A pure hash function of a `Config`.** The root package exports one, taking a
`Config` and returning the 32 bytes of 8.2 as `[32]byte`. Its name is **`HashConfig`**
(adopted; architecture.md 2.6 lists it beside the validator, and `World.ConfigHash` and
`WorldCreated.ConfigHash` are the fields that carry the same value). It hashes the configuration **as given**: it applies no
option, so it differs from `World.ConfigHash` when `WithTargetSpread` was given (8.1),
and it validates nothing (an invalid `Config` has a hash, which `NewWorld` then
refuses). It lets a harness tell two documents apart, or find a change that leaves
the hash equal (`spec-experiments.md` 14.3, 14.4), without building a world.

## 9. The parameter registry

### 9.1 What it is

The registry is the parameter-owner table in `specs/README.md`. Each row names a
parameter, its scope and the one spec that owns it. The owner's spec has the
parameter table with the field's meaning and default. Architecture.md 9.7 and
testing.md 6.7 require that table and the configuration types to agree in both
directions.

### 9.2 The checks

When the configuration types exist, a test (testing.md 6.7) checks that:

1. every row of the owner table is a field of the `Config` type of its scope;
2. every field of `Config` is a row (presentation-only fields are exempt and
   say so);
3. the default of the field, from its `Default` constructor, passes validation;
4. the default stated in the owner's parameter table equals the default in the
   constructor. The tables use the form `| Scope | Field | Meaning | Default |`,
   so a plain text parse reads them (only the number is compared, after removing
   spaces). This keeps a documented default from drifting from the code;
5. the validation tests have a failing case for the field.

Checks 1 and 2 are the two directions the brief of architecture.md 9.7 asks for and
are MUST; check 3 is MUST; check 4 stays a SHOULD (FR-CFG-030) and check 5 is MUST
(FR-CFG-031). If check 4 proves brittle it is relaxed to a review item (13.4).

**How a row is read.** The scope cell of a row is the name of a Go type, the words before
an optional parenthesis (`World (option; hashed, spec-config.md 8.1)` is `World`), and a row
names one field of one type. A parameter that two types carry has two rows: `Scale` of
`World` and `Scale` of `Group`, `Upkeep` of `World` and of `Role`. A section of `Config`
has a row with the scope `Config` (`Yields` as `Edges` and `WarmStart`; there is no row
`YieldProfile` for "a region and a recipe"). An element type has a row for each field it
has (`Month`, `Season`, `SeasonYield`, `Item`, `Output`, `Tool`, `SeedPrice`, `SeedWage`,
`WarmGroup`, `JobSpec`, `WarmPool`, and the yield rows with the scope `YieldProfile`).
The section of `Config.World` is the Go type `WorldParams` (3.7, D140): the scope of its rows
is `WorldParams`, in the README table and in the `notBuilt` list, from the change that renames
the type (the type `World` is the simulation, and the test reads a scope as a type name, so a
cell that names a type that does not exist fails it); the owner specs' own tables keep the
word `World`, and the test of their defaults (`spec-config.md` 9.2, check 4) keys on it.

**Staged form** (decision A6 of the plan of Stage 0; architecture.md 9.7 and testing.md 6.7
state the same rule). The specs describe all 25 modules and the code is built in stages,
so checks 1, 3 and 5 run over the **built** rows only. The test holds one explicit list,
`notBuilt`, of the scopes (a Go type name) and of the fields of a built scope
(`Scope.Field`) that the code does not have yet, and it skips the rows named by it. It
fails when a listed scope or field exists in the code, so a stale entry cannot hide a
gap, and when a row is neither built nor listed, so a typo cannot hide one. Check 2 is not
staged: a field the code has with no row fails at once. The pull request that builds a
scope or a field deletes its entries in the same change, and the list is empty when the
last module is built, so the staged form leaves no trace.

The owner table and the fields of 3.7 have a **second reader**: the registry test of
`spec-calibration.md` 11.1 (FR-CLB-001 to 003), which checks that every row of the
owner table and every field of 3.7 and 3.7.1 has a row in CLB's registry of knobs, and the
reverse, and reads the solve options (6.2.8) and the presentation-only fields (6.3) as rows of
its own. It adds no requirement to this spec: the checks above are
unchanged, and a new field needs its registry row in the same change as its row in
the owner table.

**What is a row.** A row is a field of `Config`: a World, Role, Region, Group,
Commodity, Recipe, YieldProfile or Calendar field, a field of a module's tables (scopes Demand,
Want, Taste and RegionTaste for `Modules.Demand`, `spec-demand.md` 9; and the scopes of 3.7.1 for the fourteen modules of wave 5, written as Go type names: `Settlements`, `SettlementClass`, `ClassLimit`, `RegionSettlement`, `Storage`, `RoleRoom`, `StoreClass`, `Capacity`, `FlowTarget`, `Employment`, `Employer`, `Tenure`, `TenureTerm`, `Holding`, `Credit`, `CreditTerms`, `Lend`, `Borrow`, `Risk`, `RouteRisk`, `StoreRisk`, `Infrastructure`, `FacilityClass`, `InitialFacility`, `FacilityBuild`, `FacilityRepair`, `Insurance`, `CoverTerms`, `Underwrite`, `Insure`, `Polity`, `Jurisdiction`, `Member`, `Fiscal`, `Tax`, `Duty`, `Toll`, `Payout`, `Purchase`, `Release`, `Window`, `Institutions`, `Body`, `Right`, `Control`, `RateLimit`, `Policy`, `PolicyActor`, `Lever`, `ScheduleStep`, `Reaction`, `Saving`, `Saver`, `ProjectWorth`, `Technology` and `TechField`, and the fields they add to `Role` and `Recipe`), or a bound-checked
section such as `WarmStart`. The `Label` of a store class, a settlement class, a jurisdiction, a facility class, a technology field, a policy actor, a body, a right, a control, a rate limit and a fiscal flow is presentation only and has no row (8.3). Two kinds of thing are NOT rows and the registry test MUST NOT look
for a field behind them:

- the **solve options** of `spec-viability.md` 11 (`StockCoverTicks`,
  `MoneyBufferTicks`, `ResidualWarnBP`, `MaxPivots`, `Lanes`). They shape how a
  table is produced and are not part of what a world is. They have defaults, bounds
  and an owner in their own spec and are not in the parameter-owner table, because
  the test of testing.md 6.7 requires a field behind every row (FR-CFG-047). They
  are rows of **CLB's registry of knobs** (`spec-calibration.md` 6.2.8), which is a
  different thing from the parameter-owner table: it classifies each row of that
  table and the solve options, and holds no default or bound (`spec-calibration.md`
  4.1, 4.5);
- the **bounds** of 4.1, which are constants in `bounds.go`, and the **options** of
  architecture.md 6.2 and `spec-world.md` 4.1 (`WithSeed`, `WithLogger`,
  `WithObserver`, `WithAudit`, `WithStrategy`, `WithSpawner`, `WithClearing`,
  `WithPolicy`, `WithReseed`). `WithPolicy` registers the code behind a policy name; the name is hashed through the row `PolicyActor.Policy`. One option is the exception: `WithTargetSpread` changes results and
  is part of the effective configuration (8.1), so it **keeps its row** in the owner
  table, with a Scope cell that reads `World (option; hashed, spec-config.md 8.1)`.
  The registry test reads that cell and checks the row against the option, not
  against a `Config` field (checks 1 and 3 do not apply to it), and the hash test of
  8.3 covers it through the effective configuration. An option that is a row is the
  only one that changes results; an option that changes none has no row.

The registry must contain one row for each field of 3.7 and 3.7.1, including the sixteen belief
parameters, the eleven agent tunables, the recipe, yield and role fields, and
`WarmStart`.

### 9.3 Gaps found in the current catalogue

Reading the existing specs against rules 9.5 to 9.7 showed these. They are fixed
in the specs that own them, not here, except where this table says the framework
decided.

| Gap | Where | State |
|-----|-------|-------|
| The welfare window `T`, default 30 ticks, is a configuration value (`spec-welfare-and-wages.md` 2.3.1) but has no row in the owner table | WEL | **Resolved on the spec side.** WEL adopted `World.WelfareWindow`, default 30, bound 1 to 1 000 (2.3.1); the effective window of a role is `max(WelfareWindow, WindowCycles * longest Duration)`, at most `MaxWelfareWindow` (4.1, 4.2). The owner-table rows (`WelfareWindow`, `WindowCycles`, `MinWorkShare`) are still the README's to add |
| The tolerance for `ReferenceBasket` against the sum of reference prices is "configured" (`spec-currency.md` 5.3) but unnamed and has no default | CUR | **Resolved.** It is `World.BasketTolerance`, default 1 000 basis points of `ReferenceBasket`, owner `spec-agents.md` 5.6.1 and 12; the rule is in 7.4.1 with `ErrBasketMismatch` |
| `WithTargetSpread` changes results but is an option, so a restore with a different option would diverge silently | WEL | **Resolved.** It is in the effective configuration and the hash (8.1), and the owner table keeps its row with the Scope cell `World (option; hashed, spec-config.md 8.1)` (9.2), which the README adds |
| `World.Clearing`, the registered name of the clearing rule, is in no field catalogue: 3.7 listed the spawner's name only and `spec-clearing.md` 8 named no field | CLR, WLD | **Resolved.** `World.Clearing` (a registered name, default `"average"`, hashed) is the field for the name, as `World.Spawner` is for the spawner (3.7, 7.1.4, FR-CFG-081); the owner is `spec-clearing.md` 8, and the owner table gets its row (`World`, `Clearing`, `spec-clearing.md` section 8) in the same change, or FR-CFG-028 fails |
| `spec-viability.md` 11 said the solve options "are rows in the parameter-owner table" while 9.2 and FR-CFG-047 say they are not, because the registry test requires a field behind every row | VIA, CLB | **Resolved, CFG wins** (`spec-calibration.md` X1). The solve options are rows of CLB's registry of knobs (`spec-calibration.md` 6.2.8), not of the parameter-owner table; `spec-viability.md` 11 says so |
| ~~Formulas used 365 days a year and 250 working days as constants, while ADR 0001 point 8 says a tick has a configurable length and `spec-currency.md` 2.4 said a tick is a day~~ | CUR, WEL, SKL | **Resolved.** The year is derived from the `Calendar` (`TicksPerYear`, `spec-calendar.md` 3.2, 365 at the Earth default) and `ExpectedWorkTicks` (Role, default 250, `spec-welfare-and-wages.md` 4.2) is a parameter; the sibling specs refer to them |
| ~~The reference price of a commodity in a region has no owner~~ | GDS, REG | **Resolved.** The commodity's `ReferencePrice` is required and owned by `spec-goods.md` 6.3; a region's override is `spec-regions.md`'s |
| `MaxPrice` and `MaxQuantity` are referenced by the clearing spec as "configured" | CLR | **Resolved here, wording owed there.** They are bounds, not tunables, and 4.1 says so; `spec-clearing.md` 3.2 and its parameter lines should say "bounded by" and not "configured" |
| A role's default `ExpectedWorkTicks` of 250 is invalid when the calendar gives fewer than 250 ticks a year | WEL, CAL | **Decided: keep the error** (13.5). The default is not scaled silently; the error names `TicksPerYear` |
| `spec-spawner.md` adds eleven World fields and `Role.Closed`, and `spec-demand.md` adds the tables of `Modules.Demand`; the spawner's *name* has no field in either | SPN, DEM | **Resolved here, a choice to confirm.** `World.Spawner` (a registered name, default `"default"`, hashed) is the field for the name, as `Role.Strategy` is for a strategy (FR-CFG-063); it needs a row, owner `spec-spawner.md` 3.1.3. The other fields are rows already proposed by their owners (FR-CFG-064) |
| The estates example in 6.2 put a region parameter (`pool_release`) in a module table | CFG | **Fixed** in 6.2: estate parameters are region fields (`spec-estates.md` 3) and `[modules.estates]` is the switch |
| The owner table could not be read one row to one field: `Scale` (group, "world default 1"), `Upkeep` (world, "also Role"), `YieldProfile` (a region and a recipe), the scopes `Yield`, `Recipe tool`, `Recipe output` and `Want item`, and no row for the elements of the calendar, the yield seasons, the lists of items and the warm-start tables | CFG, README | **Fixed** (9.2, "How a row is read"): two rows each for `Scale` and `Upkeep`, `Yields` a section row, scopes spelled as Go type names (`YieldProfile`, `Tool`, `Output`, `WantItem`), and rows for `Month`, `Season`, `SeasonYield`, `Item`, `Output`, `Tool`, `SeedPrice`, `SeedWage`, `WarmGroup`, `JobSpec` and `WarmPool` |
| `[modules.estates]` mapped to no field of `Modules` (3.1 listed 24 pointers and none was estates) | CFG, EST, OBS | **Fixed** in 6.2.4: `Modules.Estates`, the 25th pointer, empty struct, the pool policy; `ReliefPaid` is not a slice event |

## 10. Observability

10.1. `NewWorld` logs `world created` at `Info` once, with a configuration
summary: the seed, the counts of commodities, recipes, roles, regions, agents and
households, which modules are on, whether a warm start is present, and the first 8
bytes of the hash as `config_hash` (observability-and-logging.md 3.2, 3.3.5). The
attribute keys are `seed`, `commodities`, `recipes`, `roles`, `regions`, `agents`,
`households`, `modules`, `warm_start` and `config_hash`, and, for the modules that are on, the per-module counts (`prices`, `wages`, `groups`, `jobs`, `edges`, `components`, `nodes`, `counterparties`, `modes`, `merchants`) that `spec-world.md` 5.8.1 fixes. All are keys of the table of observability-and-logging.md
3.3.2. It does not log the
configuration.

10.2. Each warning (7.2) is a `Warn` line "configuration warning" with `rule` and
`path`. There is no other log line in this spec. A validation error is returned,
never logged (architecture.md 8.3).

10.3. `WorldCreated` carries the seed (`Seed`), the whole hash (`ConfigHash`, a fixed-size array), a boolean `WarmStart` that says whether `Config.WarmStart` was present
(`spec-viability.md` 17.1) and the **names table** (`Names`, the strings of the typed IDs, `spec-observability.md` 3.2.4, 4.4); observability-and-logging.md 4.3 lists them. The event holds no map: each kind of the table is a slice of strings indexed by the ID type of its kind. The summary of 10.1 also says whether a warm
start was applied; the warm start's own `Info` line is `spec-viability.md` 17.2's.

10.4. The keys `path`, `rule` and `config_hash` are in the table of
observability-and-logging.md 3.3.2, and so are the summary keys of 10.1 that were
added with it: `commodities`, `recipes`, `roles` and `regions` (counts), `modules` (the
names of the modules that are on) and `warm_start` (a boolean).

## 11. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CFG-001 | The configuration types MUST be plain structs in the root package, with no struct tags and no embedded types, and a world MUST be buildable from Go alone. |
| FR-CFG-002 | A module MUST run if and only if its entry in `Modules` is non-nil, and a `Config` with no modules MUST give a closed economy with no faucets, taxes, credit or growth. |
| FR-CFG-003 | `NewWorld` MUST deep-copy its `Config`, so that changing the caller's value afterwards does not change the world. |
| FR-CFG-004 | A `Config` loaded from a file and one built in Go with the same content MUST have the same hash and MUST give identical runs. |
| FR-CFG-005 | Entities MUST be canonicalised by ID before a world is built or a hash taken, so their order in a file or slice MUST NOT change the hash, the typed IDs or the results. |
| FR-CFG-006 | An ID that does not match `[a-z][a-z0-9_]*` or exceeds 64 bytes MUST be rejected with `ErrInvalidID`, and two entities of one kind with the same ID MUST be rejected with `ErrDuplicateID`. |
| FR-CFG-007 | The defaults MUST be defined once, in `Default` constructors in the root package, and the file loader MUST start each entity from its constructor so that an absent key takes the default. |
| FR-CFG-008 | A required field that is absent from a file MUST be rejected with `ErrMissingField`, and a zero-value entity MUST fail validation rather than be treated as the default. |
| FR-CFG-009 | A value outside the bound of its field, or outside a constant of section 4.1, MUST be rejected with `ErrOutOfBounds` carrying its path, value and bound. |
| FR-CFG-010 | A derived product of section 4.2 that does not fit in `int64` MUST be rejected with `ErrOverflow`. |
| FR-CFG-011 | `Validate` MUST return every error found, up to 100, each with a path and a sentinel matching `errors.Is`, followed by `ErrTooManyErrors` when more were left out. |
| FR-CFG-012 | The errors returned by `Validate` MUST be in order of path and then rule, identical on every run. |
| FR-CFG-013 | A reference to an undefined commodity, role, recipe or region MUST be rejected with the sentinel for that kind. |
| FR-CFG-014 | A recipe with no output MUST be rejected with `ErrRecipeNoOutput`, and a commodity that nothing makes, uses or sources MUST be rejected with `ErrOrphanCommodity`. |
| FR-CFG-015 | A commodity that is consumed but cannot be produced from a source, including one only inside a cycle with no source, MUST be rejected with `ErrUnproducible`. |
| FR-CFG-016 | A module that is on while a module it needs is off MUST be rejected with `ErrModuleDependency`. |
| FR-CFG-017 | More entities of a kind than its bound MUST be rejected with `ErrTooMany`. |
| FR-CFG-018 | A warning MUST NOT fail validation, MUST be returned by `Warnings` with a path and a rule, and MUST be logged once at `Warn` by `NewWorld`. |
| FR-CFG-019 | The loader MUST reject an unknown key at any depth with `ErrUnknownKey` and its path. |
| FR-CFG-020 | The loader MUST reject a floating point literal where an integer is needed with `ErrNotInteger`, and any other value of the wrong type (a string where a number or a bool is needed, an integer where a string is needed, a date or a time) with `ErrWrongType`, each naming the path. |
| FR-CFG-021 | The loader MUST reject a missing or unsupported `schema` in the merged tree with `ErrUnsupportedSchema`, and a document that overlays another MUST NOT need a `schema` line of its own. |
| FR-CFG-022 | The loader MUST reject a document longer than `MaxDocumentBytes` with `ErrTooLarge` before allocating in proportion to its size, and MUST reject with `ErrTooLarge`, before decoding any document, more than `MaxDocuments` documents or a sum above `MaxMergedBytes`, and, before decoding the document, a line above `MaxLineBytes`, a key or a value nested deeper than `MaxNestingDepth` and a table with more than `MaxTableKeys` keys (6.3.1). `LoadFS` MUST read each file through a bound whatever `fs.FileInfo` says. |
| FR-CFG-023 | Documents MUST be merged in order, tables recursively by key and scalars and lists by replacement, and a table-versus-scalar conflict MUST be rejected with `ErrMergeConflict`. |
| FR-CFG-024 | The loader MUST NOT panic on any input. |
| FR-CFG-025 | The configuration hash MUST be the SHA-256 of the canonical encoding of 8.2.1 of the effective configuration, and equal effective configurations MUST give equal hashes on every platform. |
| FR-CFG-026 | Changing any field of the configuration that is not on the presentation-only allow-list MUST change the hash, provided the module that owns it is on (8.2.1), and changing a presentation-only field, the layout of a file, the seed, the logger or an observer MUST NOT. |
| FR-CFG-027 | An option that affects results MUST be applied to the effective configuration before it is hashed. |
| FR-CFG-028 | Every row of the parameter-owner table MUST correspond to a field of the configuration, and every non-presentation field MUST have a row. In the staged form of 9.2 the rows of a scope or a field that the code does not yet have MUST be skipped only by an explicit list in the test, and a listed scope or field that exists in the code, or a row that is neither built nor listed, MUST fail the test. |
| FR-CFG-029 | The default of every field MUST pass validation. |
| FR-CFG-030 | The default stated in an owner's parameter table SHOULD equal the default in the `Default` constructor. |
| FR-CFG-031 | The validation tests MUST include a failing case for each parameter. |
| FR-CFG-032 | `WorldCreated` MUST carry the configuration hash, and `NewWorld` MUST log `world created` once at `Info` with the summary of section 10.1 and no configuration contents. |
| FR-CFG-033 | A chain that fails the viability check MUST be rejected with `ErrNotViable` (satisfied by FR-VIA-004 to 008 and 011; 7.6). |
| FR-CFG-034 | The calendar's months and seasons MUST be ordered lists that keep their written order through loading, canonicalisation and the hash, and an overriding document MUST replace such a list as a whole. |
| FR-CFG-035 | `Config` MUST have a `Yields` section of profiles keyed by region and recipe, sorted by region then recipe, and the production rules of `spec-production.md` 10.2 MUST be run on it and on the recipes with the sentinels `ErrCostShares`, `ErrToolNotDurable`, `ErrEligibleHasTool`, `ErrUnknownSeason`, `ErrModuleDependency`, `ErrDuplicateID` and `ErrOutOfBounds`. |
| FR-CFG-036 | The derived checks of 4.2 for recipes, agents, the warm start and beliefs MUST be run, and a product that does not fit MUST be rejected with `ErrOverflow` carrying the path. |
| FR-CFG-037 | `Config.WarmStart` MUST be nil by default, a non-nil value MUST NOT switch any mechanism on, and it MUST be validated with `ErrWarmStartPrice`, `ErrWarmStartJob` and the reference, duplicate and overflow sentinels of 7.4.1. |
| FR-CFG-038 | A `Role.Strategy` name that is none of `"default"`, `"directed"` and a name registered with `WithStrategy`, a `World.Spawner` name that is not `"default"` or registered with `WithSpawner`, and a `World.Clearing` name that is not `"average"` or registered with `WithClearing`, and a `PolicyActor.Policy` name that is none of `"hold"`, `"schedule"`, `"rule"`, `"directed"` and a name registered with `WithPolicy` MUST each be rejected by `NewWorld`, after `Validate` has succeeded, with `ErrUnknownStrategy`, `ErrUnknownSpawner`, `ErrUnknownClearing` and `ErrUnknownPolicy` as a `*ValidationError` with the path and the rule, `Validate` MUST NOT check them, and the four names MUST be in the effective configuration and the hash. |
| FR-CFG-039 | The sixteen belief parameters (the fifteen of the first draft and `BeliefEvidenceShare`) MUST each have a default that passes validation and a bound, and the rules of FR-BEL-027, FR-BEL-028 and FR-BEL-037 MUST be run with `ErrOutOfBounds`. (changed: fifteen became sixteen) |
| FR-CFG-040 | A region whose `ReferenceBasket` differs from the cost of the baseline upkeep plus its `ExtraUpkeep` at its effective reference prices by more than `BasketTolerance` basis points MUST be rejected with `ErrBasketMismatch`. |
| FR-CFG-041 | Every viability error MUST be a `*ViabilityError` with a `Rule`, a `Path` and the offending value, matching `errors.Is(err, ErrNotViable)`, and the rules `unproducible_in_realm`, `role_cannot_work`, `cycle_not_productive`, `labour_short` and `storage_gap` MUST be errors. |
| FR-CFG-042 | The viability warnings `region_not_self_sufficient`, `subsistence_exceeds_expected_work`, `target_above_surplus`, `storage_gap_input`, `viability_check_skipped` and `cycle_cost_gain` MUST be warnings and MUST NOT fail validation. |
| FR-CFG-043 | `Validate` MUST skip the programmes of V2 and V3 and warn with `viability_check_skipped` when the tableau would exceed `MaxViabilityCells` or the pivots `MaxViabilityPivots`, MUST NOT skip V1 or V4, and MUST run the programmes below the gate. |
| FR-CFG-044 | A recipe cycle whose `q_in / q_out` is 1 or more MUST be rejected (`cycle_not_productive`), and a productive cycle whose cost-loop gain `CostFloor / 10 000 * q_in / q_out` (the full-cost share of its own inputs at the solver's prices, 7.6 item 3) is 1 or more MUST produce the warning `cycle_cost_gain`, computed in exact integer arithmetic. (changed: restated for the labour-inclusive unit cost, N-066) |
| FR-CFG-045 | A `Sited` recipe with no inputs and with `Draws` MUST count as sourced in the producibility fixpoint, while the resources module is on. |
| FR-CFG-046 | The bounds `MaxRecipeTicks`, `MaxYieldBP`, `MaxRequires`, `MaxDraws`, `MaxUpkeepItems`, `MaxViabilityPivots`, `MaxViabilityCells`, `MaxLanes` and `MaxSeedPrices` MUST be constants in `bounds.go` with the values and derivations of 4.1. |
| FR-CFG-047 | The solve options of `spec-viability.md` 11 MUST NOT be `Config` fields, MUST NOT be hashed, and the registry test MUST NOT require a field behind them. |
| FR-CFG-048 | A role whose `ExpectedWorkTicks` exceeds the derived `TicksPerYear` MUST be rejected with `ErrOutOfBounds` whose bound is `TicksPerYear`, and no default MUST be scaled silently. |
| FR-CFG-049 | `World.WelfareWindow` and `World.BasketTolerance` MUST be configuration fields with defaults 30 and 1 000, a bound each, and rows in the parameter-owner table. |
| FR-CFG-050 | `World.WindowCycles` MUST be a configuration field with default 2 and bound 1 to 100, and `Role.MinWorkShare` one with default 4 000 and bound 0 to 10 000, each with a row in the parameter-owner table and a failing-case validation test. |
| FR-CFG-051 | A role whose `MinWorkShare * TicksPerYear / 10 000` exceeds its `ExpectedWorkTicks` MUST be rejected with `ErrOutOfBounds` carrying that bound. |
| FR-CFG-052 | The effective welfare window of a role, `max(WelfareWindow, WindowCycles * longest Duration on its list)`, MUST be computed by validation, MUST be at most `MaxWelfareWindow` (3 650, provisional) or the configuration rejected with `ErrOutOfBounds`, and MUST be the window used by the overflow check of window times the largest daily income. |
| FR-CFG-053 | Validation MUST NOT emit `job_longer_than_window`; it MUST report the informational `window_extended` once per role whose effective window exceeds `WelfareWindow`, and that diagnostic MUST NOT fail validation. |
| FR-CFG-054 | A role with `TrainingTicks` above 0 that has a group with a scale above 1 MUST be rejected with `ErrOutOfBounds` and the rule `traineeship_needs_weight_one`. |
| FR-CFG-055 | A `WarmStart` starting wage MUST be rejected with `ErrWarmStartPrice` when it is outside `[1, MaxMoney]` or when its product with the longest recipe `Duration` exceeds `MaxMoney`, two wages for one region MUST be rejected with `ErrDuplicateID`, and `WarmStart.Wages` MUST be part of the hash. |
| FR-CFG-056 | Validation MUST warn, and MUST NOT fail, with `cost_floor_at_full_cost` when `CostFloor` is 10 000 or more. |
| FR-CFG-057 | `World.BeliefEvidenceShare` MUST be a configuration field with default 5 000 and bound 1 to 10 000, `World.CostFloor` MUST default to 7 000, and each MUST have a row in the parameter-owner table and a failing-case validation test. |
| FR-CFG-058 | The error for an unproducible commodity MUST name one cycle with no source, the commodities and recipes in ID order, whenever the unproducible set contains a cycle, and a required tool MUST count as an input of that cycle. |
| FR-CFG-059 | A commodity marked `Raw` MUST count as sourced in the producibility check and in V1, and `Validate` MUST warn, and MUST NOT fail, with `raw_without_source` when nothing real supplies it. |
| FR-CFG-060 | `Config.Modules.Demand` MUST be on if and only if non-nil, MUST be written as the table `[modules.demand]` with wants, tastes and region tastes keyed by ID, and MUST be part of the effective configuration and the hash, every field but `Label`, in canonical order. |
| FR-CFG-061 | The demand module's bounds `MaxWants`, `MaxWantItems`, `MaxWantsPerRole`, `MaxWantPeriod`, `MaxWantRank`, `MaxElasticityBP`, `MaxCeilingBP`, `MaxSubstitution` and `MaxTasteScale` MUST be constants of `bounds.go`, and a field outside its bound MUST be rejected with `ErrOutOfBounds`. |
| FR-CFG-062 | Validation MUST reject, with the sentinel for the kind, an unknown commodity, role, region or want in a want item, taste or region taste, a want with no item or all weights 0 (`ErrWantItems`), a durable want item (`ErrWantDurable`), a duplicate want, taste or region taste (`ErrDuplicateID`), and wants products that overflow (`ErrOverflow`). |
| FR-CFG-063 | `World.Spawner` MUST be a registered name defaulting to `"default"`, checked by `NewWorld` and not by `Validate` (`ErrUnknownSpawner`), and the name MUST be hashed. |
| FR-CFG-064 | The eleven spawner parameters and `Role.Closed` MUST be configuration fields with the defaults and bounds of `spec-spawner.md` 11, each with a row in the parameter-owner table and a failing-case test, and `ChangePrivation` not below `RemovePrivation` MUST be rejected with `ErrOutOfBounds` (rule `ladder_order`). |
| FR-CFG-065 | Validation MUST warn, and MUST NOT fail, with `entry_share_window`, `renewal_with_population`, `no_open_role`, the warnings of `spec-demand.md` 10.2 and `wants_exceed_labour`; `group_below_floor` MUST be a bounded run-time `Warn` and not a validation result. |
| FR-CFG-066 | A role with no group in any region MUST be valid and MUST draw no warning. |
| FR-CFG-067 | `Edges` and `RegionPrices` MUST be sorted (edges by `(From, To)` as region index, prices by region then commodity), MUST be hashed with every field, and the graph rules of `spec-regions.md` 9.1 MUST run in `Validate`; `ErrNoRegions` and `ErrSelfEdge` MUST be matched with `errors.Is`. |
| FR-CFG-068 | `Config.Modules.MoneySupply` MUST be on if and only if non-nil, MUST be written as `[modules.money_supply]`, MUST be part of the effective configuration and the hash with every field but `Label`, and its bounds MUST be constants of `bounds.go`. |
| FR-CFG-069 | `Config.Modules.Population` MUST be on if and only if non-nil, MUST be written as the table `[modules.population]`, MUST be part of the hash with its schedules in knot order and its region rows in region order, and a schedule outside the bound `MaxKnots` MUST be rejected with `ErrOutOfBounds`. |
| FR-CFG-070 | `Config.Modules.Transport` MUST be on if and only if non-nil, MUST be rejected with `ErrModuleDependency` while the region graph is off, MUST be part of the hash with its modes in ID order and every field but `Label`, and its modes MUST be validated by the rules of `spec-transport.md` 12.1. |
| FR-CFG-071 | `Region.Label`, `World.DistanceLabel`, `Mode.Label` and `MoneyFlow.Label` MUST be presentation only: changing them MUST NOT change the hash, and they MUST be on the allow-list of the hash test. |
| FR-CFG-072 | Validation MUST warn, and MUST NOT fail, with the warnings of `spec-regions.md` 9.2, `spec-money-supply.md` 12.2, `spec-population.md` 9.2 and `spec-transport.md` 12.2; the informational diagnostics `replacement_ratio` and `renewal_ignored` MUST NOT fail validation. |
| FR-CFG-073 | `Config.Modules.Resources` MUST be on if and only if non-nil, MUST be written as the table `[modules.resources]`, MUST be part of the hash with resources in ID order and endowment rows in (region, resource) order, and a draw of an undefined resource MUST be rejected with `ErrUnknownResource`, a bad shape or rate combination with `ErrResourceShape` and a bad abundance curve with `ErrCurve`. |
| FR-CFG-074 | `Config.Modules.Information` MUST be on if and only if non-nil, MUST be written as the table `[modules.information]`, MUST be part of the hash, and `Role.Informed` MUST be rejected with `ErrModuleDependency` while it is nil; the derived checks of `spec-information.md` 9.2 MUST be rejected with `ErrTooMany`. |
| FR-CFG-075 | `Config.Modules.External` MUST be on if and only if non-nil, MUST be written as `[modules.external]`, MUST be rejected with `ErrModuleDependency` while the region graph is off, MUST be part of the hash with nodes, lines, gates and season factors in canonical order and every field but `Label`, and MUST NOT hash its runtime state; its rows MUST be validated by the rules of `spec-external-sector.md` 14.1. |
| FR-CFG-076 | Validation MUST warn, and MUST NOT fail, with the warnings of `spec-resources.md` 13.3, `spec-information.md` 9.3 and `spec-external-sector.md` 14.2. |
| FR-CFG-077 | `Config.Modules.Subsistence` MUST be on if and only if non-nil, MUST be written as the table `[modules.subsistence]` with role rows `[modules.subsistence.role.<id>]`, MUST be part of the hash with its lists and rows in canonical order and its three tunables, its bounds `MaxOwnUseRecipes`, `MaxOwnUseTicks` and `MaxRetryJobs` MUST be constants of `bounds.go`, and the rules of `spec-subsistence.md` 13.2 MUST be rejected with `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited` and `ErrOwnUseOverlap`. |
| FR-CFG-078 | `Config.Modules.Migration` MUST be on if and only if non-nil, MUST be written as `[modules.migration]`, MUST be part of the hash with its `Regions` rows in region order, MUST be rejected with `ErrModuleDependency` while `Modules.Information` is nil, and `Role.Fixed` MUST default to false and be hashed as any role field (`spec-migration.md` 11, 12). |
| FR-CFG-079 | `Config.Modules.Merchants` MUST be on if and only if non-nil, MUST be written as `[modules.merchants]` with the merchant block `[role.<id>.merchant]`, MUST be part of the hash with the block of each role, its bounds `MaxLotUnits`, `MaxContracts` and `MaxTimeCostBP` MUST be constants of `bounds.go`, and the rules of `spec-merchants.md` 14.1 MUST be rejected with `ErrUnknownMode` (a root sentinel with the transport module alone, 7.3), `ErrMerchantShape`, `ErrMerchantWeight` and the sentinels named there. |
| FR-CFG-080 | Validation MUST warn, and MUST NOT fail, with the warnings of `spec-subsistence.md` 13.3, `spec-migration.md` 12.3 and `spec-merchants.md` 14.2; `own_use_beats_market` is size-gated and `own_use_partial` informational. |
| FR-CFG-081 | `World.Clearing` MUST be a configuration field holding a registered name, defaulting to `"average"`, with a row in the parameter-owner table (owner `spec-clearing.md` 8), MUST be hashed as a name, and an empty value MUST be rejected with `ErrUnknownClearing` by `NewWorld`. |
| FR-CFG-082 | A configuration with a role of a recipe of `Duration` above 1, more than one agent in a group, and no warm start or a warm start with no job row for the role, MUST produce the informational diagnostic `cold_start_lockstep` once at construction, and it MUST NOT fail validation. |
| FR-CFG-083 | `Validate` MUST NOT see `WithTargetSpread`, and the pure hash function of 8.5 MUST hash a `Config` as given, so that it differs from `World.ConfigHash` when that option was given. |
| FR-CFG-084 | `Config.Modules.Settlements` MUST be on if and only if non-nil and MUST need no other module, MUST be written as `[modules.settlements]` with classes, limits and region rows as tables keyed by ID, MUST be part of the hash with its classes sorted by `Enter` and not by ID, a class's `Flags` by name and its `Limits` by `Name`, every field but `Label`, MUST have its bounds `MaxSettlementClasses`, `MaxClassFlags`, `MaxClassLimits`, `MaxBandBP` and `MaxThresholdBP` as constants of `bounds.go`, MUST reject a `Role.Needs` or `Role.Ceiling` with `ErrModuleDependency` while it is nil and with `ErrUnknownSettlementFlag` or `ErrUnknownLimit` when no class carries it, and its warnings (`spec-settlements.md` 9.2) MUST NOT fail validation. |
| FR-CFG-085 | `Config.Modules.Storage` MUST be on if and only if non-nil, MUST be written as `[modules.storage]` with classes as `[modules.storage.class.<id>]`, MUST be part of the hash with its role rows by role and its classes by ID, every field but `StoreClass.Label`, MUST have its bounds `MaxRoom`, `MaxStoresPerAgent` and `MaxStoreClasses` as constants of `bounds.go`, MUST reject the shapes of `spec-storage.md` 13.1 with `ErrStoreShape`, `ErrRoomShape`, `ErrNotLimited`, `ErrUnknownStoreClass` and `ErrModuleDependency`, MUST check `room_below_warm_job` at construction and not in `Validate`, and its warnings (`spec-storage.md` 13.2) MUST NOT fail validation. |
| FR-CFG-086 | `Config.Modules.Employment` MUST be on if and only if non-nil, MUST be written as `[modules.employment]` with `hireable = true` and the block `[role.<id>.employer]` on a role, MUST be part of the hash when on, with `MaxOpen`, `Role.Hireable` and the `Role.Employer` block with its hire list by recipe ID, MUST NOT hash the hire records or the next hire ID, MUST have its bounds `MaxEstablishment` and `MaxHireRecords` as constants of `bounds.go`, MUST reject a `Hireable` role or an employer block while the module is off with `ErrModuleDependency`, a role that is both with `ErrEmploymentShape` and an employer group of scale above 1 with `ErrEmployerWeight`, and its warnings (`spec-employment.md` 14.2) MUST NOT fail validation. |
| FR-CFG-087 | `Config.Modules.Tenure` MUST be on if and only if non-nil, MUST be rejected with `ErrModuleDependency` while `Modules.Resources` is nil, MUST be written as `[modules.tenure]` with `[modules.tenure.term.<region>.<resource>]` and `[modules.tenure.holding.<region>.<resource>.<holder>]`, MUST be part of the hash with its terms and holdings in canonical order, MUST reserve the word `pool` as a role ID (`ErrDuplicateID`), MUST have its bounds `MaxRentStepBP`, `MaxRentCeilingBP`, `MaxHoldings` and `MaxTitleTransfers` as constants of `bounds.go`, MUST reject a title total above 10 000 with `ErrTitleShares` and a holder above `MaxHolderShareBP` with `ErrTitleCap`, and its warnings (`spec-land.md` 11.2) MUST NOT fail validation. |
| FR-CFG-088 | `Config.Modules.Credit` MUST be on if and only if non-nil and MUST need no other module, MUST be written as `[modules.credit]` with the role blocks `[role.<id>.credit.lend]` and `[role.<id>.credit.borrow]`, MUST be part of the hash with the module and every block and MUST NOT hash the rate scales, MUST have its bounds `MaxCreditRateBP`, `MaxCreditTerm` and `MaxOpenClaims` as constants of `bounds.go`, MUST reject a block with both or neither of `Lend` and `Borrow` with `ErrCreditShape`, a block while the module is off with `ErrModuleDependency` and an absent `rate_bp` or `max_rate_bp` with `ErrMissingField`, and its warnings (`spec-credit.md` 15.2) MUST NOT fail validation. |
| FR-CFG-089 | `Config.Modules.Risk` MUST be on if and only if non-nil, MUST be written as `[modules.risk]` with `[modules.risk.route.<from>.<to>.<mode>]` and `[modules.risk.store.<region>.<class>]`, MUST be part of the hash with every field, its route rows by (`From`, `To`, `Mode`) and its store rows by (`Region`, `Class`), MUST have its bound `MaxRiskRows` as a constant of `bounds.go`, MUST reject a probability or a severity outside 1 to 10 000 with `ErrRiskShape`, a route row while transport is off or a store row while storage is off with `ErrModuleDependency`, and its warnings (`spec-risk.md` 13.2) MUST NOT fail validation. |
| FR-CFG-090 | `Config.Modules.Infrastructure` MUST be on if and only if non-nil, MUST be written as `[modules.infrastructure]` with `[modules.infrastructure.class.<id>]` and `[[modules.infrastructure.initial]]`, MUST be part of the hash with its classes by ID and its initial rows by key, every field but `FacilityClass.Label`, MUST have its eleven bounds, `MaxFacilityClasses` to `MaxInitialRows`, as constants of `bounds.go`, MUST reject a recipe clause while the module is off with `ErrModuleDependency`, a class field set that its kind does not use with `ErrFacilityShape`, and the cases of `spec-infrastructure.md` 13.1 with `ErrFacilityNoCost`, `ErrFacilityHolder`, `ErrFacilityCount` and `ErrUnknownFacilityClass`, and its warnings (`spec-infrastructure.md` 13.2) MUST NOT fail validation. |
| FR-CFG-091 | `Config.Modules.Insurance` MUST be on if and only if non-nil, MUST be rejected with `ErrModuleDependency` while `Modules.Risk` or `Modules.Credit` is nil, MUST be written as `[modules.insurance]` with the role blocks `[role.<id>.cover.underwrite]` and `[role.<id>.cover.insure]`, MUST be part of the hash with the module and every block, MUST have its seven bounds, `MaxValueCapBP` to `MaxCoverRequestsBound`, as constants of `bounds.go`, MUST reject a block with both or neither of `Underwrite` and `Insure` with `ErrCoverShape` and an absent required field whose bound admits 0 with `ErrMissingField`, and its warnings (`spec-insurance.md` 14.2) MUST NOT fail validation. |
| FR-CFG-092 | `Config.Modules.Polity` MUST be on if and only if non-nil and MUST need no other module, MUST be written as `[modules.polity]` with `[modules.polity.jurisdiction.<id>]` and `[modules.polity.member]`, MUST be part of the hash with its jurisdictions by ID and its members by region, every field but `Jurisdiction.Label`, MUST have its bounds `MaxJurisdictions` and `MaxPolityDepth` as constants of `bounds.go`, MUST reject a cycle with `ErrJurisdictionCycle`, a seat the jurisdiction does not cover with `ErrSeatNotCovered`, an outside-node seat with `ErrSeatOutsideNode` and a `StartMoney` without a seat with `ErrStartWithoutSeat`, and its warnings (`spec-polity.md` 9.2) MUST NOT fail validation. |
| FR-CFG-093 | `Config.Modules.Fiscal` MUST be on if and only if non-nil, MUST be rejected with `ErrModuleDependency` while `Modules.Polity` is nil or has no governed jurisdiction, MUST be written as `[modules.fiscal.tax.<id>]`, `.duty.<id>`, `.toll.<id>`, `.payout.<id>`, `.purchase.<id>` and `.release.<id>`, MUST be part of the hash with its six lists each by ID, every field but `Label`, the seasons in the calendar's order, and MUST NOT hash the flow scale, MUST have its bounds `MaxFiscalFlows`, `MaxDutyBP` and `MaxLimitBP` as constants of `bounds.go`, MUST reject an authority with no seat with `ErrNotGoverned`, a region or a target the authority does not cover with `ErrNotCovered` and a wrong field combination with `ErrFiscalShape`, and its warnings (`spec-fiscal.md` 15.2, `no_fiscal_flows` and `fiscal_class_empty` among them) MUST NOT fail validation. |
| FR-CFG-094 | `Config.Modules.Institutions` MUST be on if and only if non-nil, MUST be written as `[modules.institutions]` with `[modules.institutions.body.<id>]`, `.right.<id>`, `.control.<id>` and `.rate_limit.<id>`, MUST be part of the hash in canonical order, every field but `Label`, and MUST NOT hash the legal levels or the right suspensions, MUST have its bounds `MaxBodies`, `MaxRights` and `MaxControls` as constants of `bounds.go`, MUST reject an unknown body with `ErrUnknownBody`, the shapes of `spec-institutions.md` 10.1 with `ErrBodyShape`, `ErrBodyOverlap`, `ErrRightShape` and `ErrControlShape` and a keyed jurisdiction, flag or rate limit while its module is off with `ErrModuleDependency`, and its warnings (`spec-institutions.md` 10.2) MUST NOT fail validation. |
| FR-CFG-095 | `Config.Modules.Policy` MUST be on if and only if non-nil, MUST be rejected with `ErrModuleDependency` while `Modules.Polity` is nil or has no jurisdiction, MUST be written as `[modules.policy]` with `[modules.policy.actor.<id>]` and its lever, schedule and rule tables, MUST be part of the hash with the policy of an actor by name only and MUST NOT hash a registered implementation or the mailbox, MUST have its eight bounds, `MaxActors` to `MaxMoveBP`, as constants of `bounds.go`, MUST reject the shapes of `spec-policy.md` 11.1 with `ErrPolicyShape`, `ErrLeverNotInReach` and `ErrLeverShared`, and an unregistered name with `ErrUnknownPolicy` in `NewWorld` (7.1.4), and its warnings (`spec-policy.md` 11.2) MUST NOT fail validation. |
| FR-CFG-096 | `Config.Modules.Saving` MUST be on if and only if non-nil, MUST be written as `[modules.saving]` with the role block `[role.<id>.saver]` and its rows `[[role.<id>.saver.worth]]`, MUST be part of the hash with the module and every block, MUST have its ten bounds, `MaxSafetyTicks` to `MaxWorthCount`, as constants of `bounds.go`, MUST reject a `Saver` block while the module is off and a `Worth` row while `Modules.Infrastructure` is nil with `ErrModuleDependency`, and its warnings (`spec-saving-and-investment.md` 11.2) MUST NOT fail validation. |
| FR-CFG-097 | `Config.Modules.Technology` MUST be on if and only if non-nil, MUST be written as `[modules.technology]` with `[modules.technology.field.<id>]`, MUST be part of the hash with its fields by ID, every field but `Label`, and MUST NOT hash the experience or the advance of a cell, MUST have its bounds `MaxTechFields`, `MaxFieldRecipes`, `MaxTechBase`, `MaxDoublingBP`, `MaxTechTrendBP`, `MaxDiscoveryBP`, `MaxGainBP`, `MaxStepBP`, `MaxCatchUpBP`, `MaxTechBP` and `MaxExperience` as constants of `bounds.go`, MUST reject a field whose mechanism is off but set or on but unset with `ErrTechShape` and a recipe in two fields with `ErrTechRecipeOverlap`, and its warnings (`spec-technology.md` 10.2) MUST NOT fail validation. |
| FR-CFG-098 | A module table MUST be written `[modules.<name>]` with `<name>` the snake case of its field of `Modules`, the module MUST be on if and only if the table is present in the merged tree (even with no key), and a table named after a module at the top level, or under the singular prefix `module`, MUST be rejected with `ErrUnknownKey` (6.2.1). |
| FR-CFG-099 | `Config.Modules.Estates` MUST be on if and only if non-nil, MUST be written `[modules.estates]` with no key, MUST switch on relief (`spec-estates.md` 2.6) and the pool as a flow target and nothing else, and `ReliefPaid` MUST be emitted only when it is on; `Region.PoolRelease` and `Region.ReliefBelow` MUST be hashed only when it is on (6.2.4, 8.2.1). |
| FR-CFG-100 | A path segment or a string value in the text of a loader error or of a `*ValidationError` MUST be cut at `MaxIDLength` bytes on a rune boundary and written with `%q`, so that the text holds no control character and at most `MaxNestingDepth` segments, and the loader MUST NOT copy the text of the library's errors (6.3.2). |
| FR-CFG-101 | `Validate` MUST check the count bounds of every collection before any other rule, and a configuration with any count over its bound MUST return the `ErrTooMany` errors and no other error and build no sorted or hashed structure (7.1.6). |
| FR-CFG-102 | The loader MUST return a limit error (6.3.1) or a syntax error alone, and MUST return the rejections of the mapping and the merge joined, sorted by path and then rule, at most 100, followed by `ErrTooManyErrors` (6.3.3). |
| FR-CFG-103 | The loader's keys MUST be the snake case of the Go field names except for the exceptions of 6.2.2, and a test MUST walk the fields of every section type and fail for a field that is neither. |
| FR-CFG-104 | The encoding of 8.2.1 MUST write a field only when its value is not zero, MUST write a module only when it is on, MUST write a non-nil pointer even when its struct is zero, and MUST give each field the id of 8.2.2, so that adding a module, or a field at its zero value, does not change the hash of any configuration that existed before it. |
| FR-CFG-105 | The hash of the reference configuration MUST be pinned by a constant in a test (8.2.5), and the examples of 8.2.4 MUST be tests of the encoder. |
| FR-CFG-106 | `HashConfig` MUST be the exported name of the pure hash function of 8.5 and MUST return a `[32]byte`. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-CFG-001 | Validation of a configuration at the maximum sizes of section 4.1 SHOULD complete in one second on the reference machine. Provisional; the slice measures it. |
| NFR-CFG-002 | The loader and `Validate` MUST have fuzz tests (testing.md 7.4) with seed inputs, asserting that they never panic and that every result is either an error or a valid `Config`, and fuzz crash inputs MUST be committed (testing.md 7.5). |
| NFR-CFG-003 | `Validate` MUST be deterministic and MUST start no goroutine; its viability work MUST be bounded by `MaxViabilityPivots` and `MaxViabilityCells` so that NFR-CFG-001 holds with the programmes on or skipped. |

## 12. Stability register entries

Merged into `spec-stability.md` section 4 (STB-L42, STB-L43), where they are
indexed with their dampers and tests. The text below stays as the source:
The cold-start warning `cold_start_lockstep` (7.4.2) is the registered remainder of
STB-L22 (WLD-A3).


For `spec-stability.md`:

- **A stable default can be undone by a pack.** A pack may override any
  parameter, including ones whose defaults keep the economy inside the
  acceptance bands. Validation cannot detect this; each shipped pack is tested
  against the bands of `spec-stylised-facts.md` (ADR 0001, consequences).
- **A reachable parameter space is not a stable one.** Every value inside its
  bounds is accepted. Bounds prevent overflow, not instability. The experiment
  harness (`spec-experiments.md`) and the viability check (7.6) are the guards.

## 13. Open questions

1. **The TOML library. Decided (2026-10-12): `github.com/pelletier/go-toml/v2`**, with the
   guard pass of 6.3.1, in [ADR 0009](../adr/0009-the-toml-library-is-pelletier-go-toml-v2.md)
   (status Proposed until the first code that imports it). The experiments are in the ADR.
2. **Strict decoding in a chosen library. Decided: `config/` checks the tree itself.**
   The library is used for syntax and for the generic tree only; its struct decoding,
   its strict mode and its error text are not used (6.1, 6.3.2).

Questions 3 to 8 and 10 were decided on 2026-10-10 and question 11 on 2026-10-12; the
answers are kept so the reasons are not lost.

3. **Deleting in a later document. Decided: not in version 1.** A pack cannot
   remove an entity or reset a key to its default. No pack of the slice needs it:
   the first consumer's data pack extends a base chain, and a pack that wants fewer
   entities starts from a smaller base. TOML has no null, and inventing a marker
   would add a merge rule to every consumer of the format. Revisit when a real pack
   needs it, with the merge rules of 6.4.
4. **How strict the registry test is. Decided: as written in 9.2.** Checks 1, 2, 3
   and 5 MUST pass; check 4 (the documented default equals the coded default) is a
   SHOULD (FR-CFG-030), enforced by the parse of the `| Scope | Field | Meaning |
   Default |` tables, which every owner spec now uses. If a spec's table shape
   defeats the parse, that spec's table is fixed; check 4 is relaxed to a review
   item only if the fix is not possible.
5. **Dependent defaults. Decided: keep the error.** `ExpectedWorkTicks` (250) must
   not exceed `TicksPerYear`, so a calendar with a year under 250 ticks makes the
   default invalid until the role sets its own. Silently scaling a default would
   change results without a trace in the pack, and the expectation in whole days is
   the unit the sources and worked examples use. The cost is one line per role in a
   pack with a short year, and the error says which number to stay under
   (FR-CFG-048). A share of the year in basis points was rejected: it would hide the
   same choice behind a conversion.
6. **Modules as pointers. Decided: pointers, with a deep copy.** Pointer fields are
   the simplest "on if present"; the risk that a consumer shares a pointer between
   two worlds is removed by the deep copy of 3.5 (FR-CFG-003). A value plus a flag
   was rejected: it brings back the disagreement 3.2 exists to avoid. `WarmStart`
   follows the same rule.
7. **Where a pack's version lives. Decided: not in version 1.** The `schema` number
   versions the file format (6.3); the configuration hash catches a pack that
   differs from a checkpoint's. A pack that declares the engine version it needs is
   a feature of the data-pack deliverable (roadmap, "Deliverables that are not
   specs"), not of the framework.
8. **Where the name rules live. Decided: `NewWorld`.** `Validate` takes no
   options and cannot see `WithStrategy`, `WithSpawner`, `WithClearing` or `WithPolicy` (7.1.4);
   there are four name checks, strategy, spawner, clearing and policy, run after `Validate`
   succeeds. A `Validate` that took the registrations as an argument was rejected:
   it would put an option in a method that is meant to be pure data.
9. **The size gate of the viability check.** `MaxViabilityCells` and
   `MaxViabilityPivots` are guesses and the speed of exact arithmetic in Go is not
   measured (`spec-viability.md` open question 7). Measure in the slice and set
   them so that NFR-CFG-001 holds. Open until then.
10. **Warning rule names that other specs leave unnamed.** `inexact_hour`,
    `season_shorter_than_tick` and `tool_unaffordable` are named here (7.4.2) because
    the registry and the tests need a name. The calendar and skill-barriers specs
    should adopt them. Open until they do.
11. **Unify the module table names before the loader is written. Decided (2026-10-12):
    `[modules.<name>]` for every module (6.2.1).** The first draft recorded each spec's own
    spelling (the singular prefix `module`, bare top-level tables and the plural prefix
    `modules` for the later modules). The one
    rule is the form the later modules already used, and the specs and the fixtures of
    `testdata/chains/` were respelled in the same change: a text search for the old forms
    finds none. The answer for estates is 6.2.4: it is the 25th module and its table is
    `[modules.estates]`, with no key. N-102 is closed.

## 14. Where this lives in the code

| Concern | Home |
|---------|------|
| `Config` and its section types | root package, `config.go` |
| `Default` constructors | root package, `defaults.go` |
| Bounds (4.1) | root package, `bounds.go` |
| `Validate`, `Warnings`, `ValidationError` and the sentinels | root package, `validate.go`, `errors.go` |
| The producibility fixpoint (7.5) | root package, `validate.go`; a pure function over primitive slices that may live in `internal/` when it grows |
| The viability rules of 7.6 and the sentinels `ErrNotViable`, `ErrWarmStartPrice`, `ErrWarmStartJob` and `ViabilityError` | root package, `validate.go`, `errors.go`; the programmes and checks in `internal/steady` (`spec-viability.md` 21) |
| The four name checks (7.1.4: strategy, spawner, clearing, policy) | root package, `world.go` (they need the options); `ErrUnknownStrategy`, `ErrUnknownSpawner`, `ErrUnknownClearing` and `ErrUnknownPolicy` in `errors.go` |
| `WarmStart` and its types (`SeedPrice`, `SeedWage`, `WarmGroup`, `JobSpec`, `WarmPool`) | root package, `config.go` with `Config` (architecture.md 2.6: `config.go` holds "Config and every section struct"; `spec-viability.md` 21 said `viability.go` and is corrected); `viability.go` holds `SolveSteadyState` and the code that reads the table |
| The loader sentinels `ErrSyntax`, `ErrUnknownKey`, `ErrNotInteger`, `ErrWrongType`, `ErrMissingField`, `ErrUnsupportedSchema`, `ErrMergeConflict`, `ErrTooLarge` (6.3) and `ErrTooManyErrors` | root package, `errors.go` (architecture.md 2.6: "every public sentinel"); `config/` imports the root and returns them, wrapped |
| The loader limits (6.3.1) | root package, `bounds.go` as `MaxDocuments`, `MaxDocumentBytes`, `MaxMergedBytes`, `MaxLineBytes`, `MaxNestingDepth`, `MaxTableKeys`; `config/` reads them from the root, and the guard pass and the merge live in `config/` |
| `HashConfig` (8.5) | root package, `hash.go` |
| `Modules.Estates` and `Estates` (6.2.4) | root package, `config.go` (the type), `estates.go` (the policy, with the core estate) |
| Canonical order and the hash encoding | root package, `hash.go` |
| `Parse`, `LoadFS`, the merge and the key mapping | package `config/` (the only package with a third-party import) |
| The registry test | the root package's external test package, reading `doc/specs/README.md` by relative path (testing.md 10.2 allows reading inside the module) |

The PR that adds the first of these files also updates the layout in
architecture.md 2.6 and `AGENTS.md`, and the new terms in `doc.go`
(architecture.md 3.2, 14.2).
