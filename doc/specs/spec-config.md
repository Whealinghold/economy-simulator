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
production, beliefs, agents, viability, the spawner, demand, regions, money supply, population, transport, resources, information and the external sector (sections 3.7, 4.1, 7.4); the specs
still unwritten (migration, merchants and the rest) add theirs when they
are. Where a section depends on a spec that does not exist, it says so.

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
| `WarmStart` | the steady-state seed table: seed prices, starting money, stock and jobs per group, estate pools. A pointer: nil means no warm start. It is a section and not a module: it switches no mechanism on | `spec-viability.md` 8.1 |
| `Modules` | one optional entry per module (3.2); `Demand` is the first with its own tables (`Wants`, `Tastes`, `RegionTastes`); `MoneySupply` (`Flows`), `Population` (schedules and `Regions`) and `Transport` (`Modes`) follow; `Resources` (`Resources`, `Endowments`, `HorizonYears`), `Information` (`InfoSpeed`, `RumourErrorBP`, `RumourMaxAge`) and `External` (`Nodes`) come with the specs of wave 3; `Subsistence` (`Recipes`, `Roles`, `EnterTicks`, `ExitTicks`, `RetryJobs`), `Migration` (nine tunables and `Regions`) and `Merchants` (`MaxOpen`) come with the specs of wave 4 | each module's spec |

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
a file or a slice was ordered.

### 3.4 Canonical order

Before a world is built or a hash is taken, every entity slice is sorted by ID
(groups by region, then role). The order in which an author wrote entities has no
effect. This follows architecture.md 7.5: nothing that influences results may
depend on an arbitrary order. A recipe's inputs and outputs, being lists of
(commodity, quantity), are sorted by commodity. Edges are sorted by `(From, To)` as region indices, region prices by region then commodity, money flows by ID, transport modes by ID, and the region-keyed tables of 5.1 of `spec-regions.md` by region.

The one exception is the calendar's months and seasons. Their order is their
meaning (the third month, the season after the last), so they are **ordered
lists**: they are kept as written, are written in a file as arrays and not as
tables keyed by ID, and an overriding document replaces a list as a whole (6.4).

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
| `Modules.MoneySupply` | `Flows[]` (`ID`, `Kind`, `Target` with `Kind`, `Region` and `Role`; `Amount`, `Level`, `RateBP`, `Exempt`, `Per`, `Start`, `Until`, `Limit`, `NotIncome`, `Label` (p)); present = the module is on | `spec-money-supply.md` 4.1, 11.1 |
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

Not fields: the seed, the logger, the observers, `WithAudit`, `WithReseed` (a
restore-only option that gives a different continuation, `spec-checkpoint.md` 6.8, and
never a field), the extension registrations (`WithStrategy`, `WithSpawner` and
`WithClearing`, whose *names* are hashed through `Role.Strategy`, `World.Spawner` and
`World.Clearing` and whose code cannot be) and the solve options of `spec-viability.md` 11
(`StockCoverTicks`, `MoneyBufferTicks`, `ResidualWarnBP`, `MaxPivots`, `Lanes`):
they shape how a run or a table is produced and not what a world is. The yield
shocks of `spec-production.md` 6.8 are world state, not configuration. So are the other slots set between ticks: the flow scale of `spec-money-supply.md` 4.4.3, the birth and death scales of `spec-population.md` 6.6, the edge statuses of `spec-regions.md` 7 and the transit factor of `spec-transport.md` 10.3, the endowment shock scales of `spec-resources.md` 6.3 and the outside price shock of `spec-external-sector.md` 6.5; the public calls that set them are INT's. State, not fields, are also the own-use mode `OwnUse` and the counter `OwnUseJobs` of an agent (`spec-subsistence.md` 5.3), the journey of a migrating agent (`spec-migration.md` 14) and the contracts in flight (`spec-merchants.md` 16).

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
| `MaxLanes` | `MaxRegions * MaxRegions * MaxCommodities` | the longest lane list of a solve (`spec-viability.md` 9.4, 11): one lane per ordered pair of regions and commodity. It bounds a solve option, not a `Config` field, and is kept here because bounds live in `bounds.go` |
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
| `MaxLabelBytes` | 256 | a label, for the presentation-only fields |
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
| `MaxDocumentBytes` | 16 MiB | a parser limit (6.5) |

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

## 5. Defaults

5.1. A default is defined once, in the root package, in `defaults.go`:
`DefaultWorld()`, `DefaultRole()`, `DefaultRegion()` and one constructor for each
other entity or module that has defaults (architecture.md 9.8). Each returns a
fully populated value. Defaults are the stable constants of ADR 0001 (decision
5): changing one changes behaviour and is explained in its PR.

5.2. The zero value of an entity is **not** a valid default. A `Role{}` has
`TargetRatio` 0, which validation rejects (it must be at least 10 000,
`spec-welfare-and-wages.md` section 5.1). This is deliberate: a zero that
silently meant "use the default" cannot be told apart from a deliberate 0, and
some fields (such as `TargetSpread`) have a valid 0 and a non-zero default. A name
is the other case: `World.Clearing` empty fails (`ErrUnknownClearing`, 7.1.4), and
the default `"average"` comes from `DefaultWorld()` and not from the zero value.
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

### 6.1 Format: TOML (decided)

The format is TOML, parsed by a third-party library confined to `config/`
(architecture.md 5.2). **The format is decided; the library is not.** Choosing the
library needs the justification that architecture.md 5.3 requires (what it
provides, why the standard library is insufficient, maintenance status and
licence). The choice is recorded as an ADR with the code that first imports it.

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

Facts about particular TOML libraries (strict decoding, unknown-key reporting)
are to be checked against their documentation when one is chosen. This spec
depends on none of them: `config/` decodes into a generic tree and builds the
structs by hand (6.2), so only "parse TOML into a tree with integer and float
distinguished" is needed of the library.

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

[module.estates]
```

A demand module is a table of its own, `[demand]`, whose presence switches it on; its
wants and tastes are tables keyed by ID, so documents merge by key
(`spec-demand.md` 3.2). A role that the default spawner must not fill is written
`closed = true`:

```toml
[demand]
[demand.want.comfort]
items = [ { commodity = "cloth", weight = 10000 } ]
quantity = 1
period = 4
rank = 1
elasticity = 15000
[demand.taste.worker_a.comfort]
scale = 10000
[demand.region_taste.north.comfort]
scale = 15000

[role.visitor]
closed = true
```

The presence of `[module.estates]` switches the module on (3.2); its parameters
are region fields (`spec-estates.md` 3) and sit in the region table. The presence
of `[warm_start]` switches nothing on: it supplies a start (3.1). `commodity.*` is
elided; see `spec-goods.md`. No setting vocabulary is in the engine (AGENTS.md).

The other specs write their tables as follows, and this spec records each as its spec says. Edges and region prices are tables of the region, keyed by the other end or the commodity, with integer values (a float is `ErrNotInteger`): `[region.north.edge.south]` with `distance = 100`, and `[region.north.reference_price]` with `grain = 480` (`spec-regions.md` 4.1, 5.2). The money-supply module is `[module.money_supply]` with flows as tables `[module.money_supply.flow.<id>]` (`spec-money-supply.md` 17.6), like `[module.estates]`. The population module is the table `[population]`, schedules as lists of `{ ratio, rate }`, regions as tables keyed by region ID, like `[demand]` (`spec-population.md` 8). `spec-transport.md` states no file layout for its modes; a table of its own keyed by mode ID is assumed. The resources module is the table `[resources]` with `[resources.resource.<id>]` (keys `shape`, `rate_per_year`, `rate_bp`, `refuge`, `capacity`, `initial_bp`, `plan_share_bp`, `abundance = [{ share, factor }]`, `seasons = [{ season, bp }]`) and `[resources.endowment.<region>.<resource>]` (keys `capacity`, `initial`, `regen_scale_bp`) (`spec-resources.md` 3.3.2, 16). The information module is `[modules.information]` with `info_speed`, `rumour_error_bp` and `rumour_max_age`, and `informed = true` in `[role.<id>]` (`spec-information.md` 12.1). The external module is `[module.external.node.<region>]` with `[module.external.node.<region>.line.<commodity>]` (keys `base`, `trend_bp`, `floor_bp`, `ceiling_bp`, `walk_step_bp`, `walk_revert_bp`, `walk_bound_bp`, `price_by_season = [{ season, bp }]`), its sides `...line.<commodity>.sell` and `...line.<commodity>.buy` (keys `margin_bp`, `capacity`, `window = [{ season, bp }]`, `total_units`, `total_value`) and gates `[module.external.node.<region>.gate.<region>]` (key `scale_bp`) (`spec-external-sector.md` 3.3.1, 18.2). The subsistence module is `[modules.subsistence]` with `recipes`, `enter_ticks`, `exit_ticks` and `retry_jobs`, and a role row as `[modules.subsistence.role.<id>]` with `recipes` (`spec-subsistence.md` 3.2.2, 16.1). The migration module is `[modules.migration]` with `move_speed`, `move_review_ticks`, `move_below`, `move_gain`, `move_width`, `move_cooldown`, `move_horizon`, `move_out_share` and `move_in_share`, a region row as `[modules.migration.region.<id>]` with `out_share` and `in_share`, and `fixed = true` in `[role.<id>]` (`spec-migration.md` 11, 15.3). The merchant module is `[modules.merchants]` with `max_open`, and the merchant block is `[role.<id>.merchant]` with `modes`, `lot_units`, `fee_share_spread_bp`, `max_source_age`, `time_cost_bp`, `consignee_cut_bp`, `consigns`, `take_margin_bp` and `take_units`, beside the role keys `merchant_fee_share`, `informed` and `strategy = "merchant"` (`spec-merchants.md` 19.1). The names differ (`[module.estates]`, `[demand]`, `[module.money_supply]`, `[population]`, `[resources]`, `[modules.information]`, `[module.external]`); they are not unified here (13.11).
### 6.3 What the loader rejects

- an **unknown key**, at any depth (`ErrUnknownKey`, with its path);
- a value of the wrong type, including a floating point literal where an integer
  is needed (`ErrNotInteger`) and a string where a number is needed;
- a missing required field (`ErrMissingField`);
- a missing or unsupported `schema` (`ErrUnsupportedSchema`). The schema number
  versions the file format and is independent of the checkpoint format version
  (ADR 0002, decision 8). It is checked on the **merged** tree (6.4): an overlay
  document, one that only changes a few keys, has no `schema` line of its own, and
  the first document of a pack has it;
- a document longer than `MaxDocumentBytes` (`ErrTooLarge`), checked before
  decoding allocates in proportion to the input.

The loader does not check cross-references or bounds. That is validation (section
7), the same code whether the `Config` came from a file or from Go.

### 6.4 Packs and layering

A **pack** is a set of documents (ADR 0001, decision 6). Documents are merged in
the order given, later over earlier:

- tables merge recursively by key;
- a scalar or a list in a later document replaces the earlier one;
- a key present as a table in one document and as a scalar in another is an error
  (`ErrMergeConflict`);
- a document cannot delete an entity or a key in version 1 (open question 3);
- the `schema` key is merged like any scalar and checked once, on the merged tree
  (6.3), so a document that overlays a pack may omit it.

This is how the three levels of ADR 0001 decision 3 look in files: a base pack
(the chain), a pack that tweaks parameters (overrides scalars), and a pack that
adjusts modules (adds a `[module.*]` table). Because entities are tables keyed by
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
6.3; `NewWorld` does.

## 7. Validation

### 7.1 The framework

7.1.1. `Config.Validate` returns every error it finds, not the first, joined
(`errors.Join`). Each is a `*ValidationError` with a `Path` (for example
`roles.worker_a.target_ratio` or `regions.north.groups.worker_a.households`), a
`Rule` (the sentinel), the offending value and, for a bound, the bound.
`errors.Is` matches the sentinel and `errors.As` finds the typed error
(architecture.md 8.4; testing.md 3.9).

7.1.2. Errors are sorted by path and then by rule, so the output does not depend
on map order and a test can compare it (architecture.md 7.5).

7.1.3. At most 100 errors are returned, followed by one `ErrTooManyErrors` that
says how many were left out. A pathological file cannot flood a log.

7.1.4. `NewWorld` calls `Validate` and returns its error. The only other checks it
makes on configuration are the three that need the options (`spec-world.md` 3.2.2).
They run **after `Validate` succeeds**, and each returns a `*ValidationError` with
the path of the field and the rule (7.1.1): every `Role.Strategy` name is
`"default"`, `"directed"` (both built in, `spec-agents.md` 6.6, 10) or a name
registered with `WithStrategy`, else `ErrUnknownStrategy`; `World.Spawner` is the
built-in `"default"` or a name registered with `WithSpawner`, else
`ErrUnknownSpawner` (`spec-spawner.md` 3.1.3); `World.Clearing` is the built-in
`"average"` or a name registered with `WithClearing`, else `ErrUnknownClearing`
(`spec-clearing.md` 8). The name `"merchant"` is checked like any name, and
`strategy.Options()` registers it (`spec-merchants.md` 3.1). `Validate` takes no
options, so it cannot know the registrations, and it does not see
`WithTargetSpread` either (8.1); the three rules are therefore the config rules that
live in `NewWorld` and nowhere else, and a test builds a world with an unregistered
name of each kind (FR-CFG-038). Nothing else in `NewWorld` fails on configuration
(architecture.md 8.5).

7.1.5. `Validate` is pure and bounded. Its work is limited by the bounds of 4.1
and, for the viability programmes, by `MaxViabilityPivots` and `MaxViabilityCells`
(`spec-viability.md` 4.3); it starts no goroutine and draws no random number.

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
severity of every rule is stated in 7.4.

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
| `ErrTooMany` | more entities of a kind than its bound | here |
| `ErrModuleDependency` | a module on while a module it needs is off | here |
| `ErrNoRegions` | a configuration with no region | `spec-regions.md` 9.1 |
| `ErrSelfEdge` | an edge whose `From` equals its `To` | `spec-regions.md` 9.1 |
| `ErrUnknownResource` | a draw of a recipe or an endowment row that names a resource that is not defined | `spec-resources.md` 13.2 |
| `ErrResourceShape` | a resource shape that is not one of the three, a rate field the shape does not use set above 0, a refuge off a logistic stock or above a capacity, an initial stock above its capacity | `spec-resources.md` 13.2 |
| `ErrCurve` | abundance knots not strictly increasing, or a share or factor out of range | `spec-resources.md` 13.2 |
| `ErrOutsideNodeInhabited` | a group, warm-start row, population region row, taste row or money-flow target naming an outside node | `spec-external-sector.md` 14.1 |
| `ErrCrossedQuotes` | an outside line with both sides whose margins sum to less than `MinQuoteSpreadBP` | `spec-external-sector.md` 14.1 |
| `ErrNotAGate` | an outside gate row naming a region with no direct edge to or from the node | `spec-external-sector.md` 14.1 |
| `ErrOutsideNode` | a relocation into an outside node (run time, not validation) | `spec-regions.md` 6.1.3 |

The root maps the following sentinels of the new specs to public errors (N-075): `ErrNoRegions`, `ErrSelfEdge`, `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation`, `ErrRegionBound` (REG; the last three are ledger-level if `relocate` lives in `internal/ledger`); `ErrFlowShape`, `ErrReasonCollision` (MON); `ErrResourceShape`, `ErrCurve`, `ErrUnknownResource` (RES); `ErrOutsideNodeInhabited`, `ErrCrossedQuotes`, `ErrNotAGate`, `ErrOutsideNode` (EXT); INF adds none; `ErrVehicleNotDurable`, `ErrModeShape`, `ErrUnknownMode` (TRN: `ErrUnknownMode` is a root sentinel with the transport module alone, because `SetTransitFactors` returns it, `spec-integration.md` 5.1.3, and the merchant module reuses it); `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited`, `ErrOwnUseOverlap` (SUB); MIG adds none; `ErrMerchantShape`, `ErrMerchantWeight` (MER); `ErrUnknownClearing`, `ErrNilExtension`, `ErrWorldFailed`, `ErrInStep` (WLD, `spec-world.md` 15.1); `ErrCheckpointVersion`, `ErrCheckpointCorrupt`, `ErrCheckpointConfig`, `ErrCheckpointExtension`, `ErrOptionContext` (CHK, `spec-checkpoint.md` 13.1); `ErrModuleOff`, `ErrUnknownFlow`, `ErrUnknownLine`, `ErrAgentRemoved`, `ErrNotExternal`, `ErrNotDirected`, `ErrNotOnRole`, `ErrNotTradable` (INT, `spec-integration.md` 11.2; `ErrUnknownAgent` is REG's, above). `ErrTravellerAway` does not exist: a refusal of a region-bound obligation is `ErrRegionBound` naming `open_shipment`.

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
| a `JobSpec` whose recipe is not on the role's list, whose share is outside [0, 10 000], whose group shares exceed 10 000, or whose set of virtual start offsets is empty | `ErrWarmStartJob` | `spec-viability.md` 8.7, FR-VIA-033 | `Validate` |
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
   not pass `MaxViabilityPivots`. Above the gate the programmes are skipped and the
   warning `viability_check_skipped` names what was not run, so the validation time
   stays bounded (NFR-CFG-001). The check is **necessary and optimistic**: pooled
   free trade, the best yield across regions.
2. **The error.** A failure is `ErrNotViable` carried by a `*ViabilityError` with a
   `Rule` (`unproducible_in_realm`, `role_cannot_work`, `cycle_not_productive`,
   `labour_short`, `storage_gap`), a `Path`, the offending value and the number that
   decides it. `errors.Is(err, ErrNotViable)` matches each (FR-CFG-033,
   FR-VIA-011); the framework's ordering (7.1.2) applies.
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
(`spec-ledger.md` 12.4) do not affect what the world *is* and are excluded; the random state travels in the checkpoint
instead. `WithReseed` is excluded too: it is a restore-time option that changes the continuation and not what the world is (`spec-checkpoint.md` 6.8). Three further cases are settled here:

- **Strategy.** The *names* in `Role.Strategy` (`default`, `directed` or a registered name) are hashed; the code behind a name
  is not and cannot be (`spec-agents.md` 6.6). The registrations made with
  `WithStrategy` are options and are not hashed beyond the names the roles use.
- **The spawner, the clearing and the demand module.** The *name* in `World.Spawner` and the *name* in `World.Clearing` are hashed, not the
  code behind them (`spec-spawner.md` 3.1.3, `spec-clearing.md` 8, as for strategies; a world with a custom rule does not hash like one with the default). `Modules.Demand` is
  hashed with a presence byte and, when present, its wants, tastes and region tastes in
  canonical order (wants by ID, items by commodity, tastes by role then want, region
  tastes by region then want), every field except `Label`; `Role.Closed` and the
  spawner's parameters are hashed as any field.
- **The warm start.** `Config.WarmStart` is part of the configuration: its presence
  byte, its prices, wages, groups and pools are hashed (`spec-viability.md` 8.1). The solve
  options that produced it (`StockCoverTicks` and the rest, `spec-viability.md`
  11) are not fields and not hashed; only the table is.
- **Regions, edges and the new modules.** `Edges` and `RegionPrices` are hashed in canonical order, every field; `Region.Label` and `World.DistanceLabel` are not (`spec-regions.md` 4.1.2). `Modules.MoneySupply` is hashed with a presence byte and, when present, its flows in ID order, every field but `Label`; the flow scale is state, not configuration. `Modules.Population` is hashed with a presence byte, its schedules in knot order and its region rows by region; the effective scales a shock sets are state. `Modules.Transport` is hashed with a presence byte and its modes in ID order, every field but `Label`; the transit factor is state.
- **The resources, information and external modules.** Each is hashed with a presence byte and, when present, every field but `Label`: `Modules.Resources` with its resources by ID (knots by share, seasons in the calendar's order) and its endowment rows by region then resource, and `HorizonYears`; `Modules.Information` with its three fields, and `Role.Informed` as any role field; `Modules.External` with its nodes by region, lines by commodity, gates by region and season factors by season. The shock scales, drift, carries and cumulative totals are state and not hashed.
- **The subsistence, migration and merchant modules.** Each is hashed with a presence byte and, when present, every field: `Modules.Subsistence` with its default list and its role rows by role (each list by recipe ID) and its three tunables; `Modules.Migration` with its nine fields and its `Regions` rows by region; `Modules.Merchants` with `MaxOpen`; `Role.Fixed` and the `Role.Merchant` block (modes by ID) as any role field. The per-agent state (`OwnUse`, `OwnUseJobs`, journeys) and the contracts are state and not hashed.
- **Yields.** `Yields` is hashed in canonical order (region, then recipe, and
  each profile's season entries sorted by season ID, as `spec-production.md` 6.1
  has it). The yield shocks of
  `spec-production.md` 6.8 are state, not configuration, and are saved in the
  checkpoint.

8.2. The **configuration hash** is the SHA-256 of a canonical binary encoding of
the effective configuration. The checkpoint stores it and a restore with a
different hash fails with `ErrCheckpointConfig` (ADR 0002, decision 4; `spec-checkpoint.md` 13.1). The encoding
is written by hand in field order, with a version byte, integers as 8 bytes in
big-endian order, strings and slices with a length prefix, entity slices in
canonical order, the calendar's months and seasons in the order written, and a
presence byte for each module and for `WarmStart`. It uses no reflection and
no map, so equal configurations give equal bytes on every platform. The version of this encoding is written in the header of a checkpoint (`spec-checkpoint.md` 5.1), a restore hashes the effective configuration with that version, and from 1.0 an implementation keeps the encoder of every version it has written (`spec-checkpoint.md` 9.2.1 item 4, FR-CHK-063).

8.3. Some fields are **presentation only**: a human-readable label, a display
ladder (`spec-currency.md` 2.2). They do not change any result, so they are not
hashed. A field is presentation only if and only if its doc comment says so. A
test changes every field of `Config` in turn and checks that the hash changes,
except for the fields on a short allow-list of presentation-only fields. A new
field that the encoder forgets therefore fails a test. The allow-list is `Region.Label`, `World.DistanceLabel`, `Mode.Label`, `MoneyFlow.Label`, `Resource.Label`, `OutsideNode.Label` and the labels of the earlier specs.

8.4. The file's layout (comments, whitespace, key order, and the split into
documents) does not affect the hash (2.5).

8.5. **A pure hash function of a `Config`.** The root package exports one, taking a
`Config` and returning the 32 bytes of 8.2 (the name is chosen with the code; this
spec proposes `HashConfig`). It hashes the configuration **as given**: it applies no
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

The owner table and the fields of 3.7 have a **second reader**: the registry test of
`spec-calibration.md` 11.1 (FR-CLB-001 to 003), which checks that every row of the
owner table and every field of 3.7 has a row in CLB's registry of knobs, and the
reverse, and reads the solve options (6.2.8) and the presentation-only fields (6.3) as rows of
its own. It adds no requirement to this spec: the checks above are
unchanged, and a new field needs its registry row in the same change as its row in
the owner table.

**What is a row.** A row is a field of `Config`: a World, Role, Region, Group,
Commodity, Recipe, Yield or Calendar field, a field of a module's tables (scopes Demand,
Want, Taste and RegionTaste for `Modules.Demand`, `spec-demand.md` 9), or a bound-checked
section such as `WarmStart`. Two kinds of thing are NOT rows and the registry test MUST NOT look
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
  `WithReseed`). One option is the exception: `WithTargetSpread` changes results and
  is part of the effective configuration (8.1), so it **keeps its row** in the owner
  table, with a Scope cell that reads `World (option; hashed, spec-config.md 8.1)`.
  The registry test reads that cell and checks the row against the option, not
  against a `Config` field (checks 1 and 3 do not apply to it), and the hash test of
  8.3 covers it through the effective configuration. An option that is a row is the
  only one that changes results; an option that changes none has no row.

The registry must contain one row for each field of 3.7, including the sixteen belief
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
| The estates example in 6.2 put a region parameter (`pool_release`) in a module table | CFG | **Fixed** in 6.2: estate parameters are region fields (`spec-estates.md` 3) and `[module.estates]` is the switch |

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
| FR-CFG-020 | The loader MUST reject a non-integer number, or a string where a number is needed, with `ErrNotInteger` or a type error naming the path. |
| FR-CFG-021 | The loader MUST reject a missing or unsupported `schema` in the merged tree with `ErrUnsupportedSchema`, and a document that overlays another MUST NOT need a `schema` line of its own. |
| FR-CFG-022 | The loader MUST reject a document longer than `MaxDocumentBytes` with `ErrTooLarge` before allocating in proportion to its size. |
| FR-CFG-023 | Documents MUST be merged in order, tables recursively by key and scalars and lists by replacement, and a table-versus-scalar conflict MUST be rejected with `ErrMergeConflict`. |
| FR-CFG-024 | The loader MUST NOT panic on any input. |
| FR-CFG-025 | The configuration hash MUST be the SHA-256 of the canonical encoding of the effective configuration, and equal effective configurations MUST give equal hashes on every platform. |
| FR-CFG-026 | Changing any field of the configuration that is not on the presentation-only allow-list MUST change the hash, and changing a presentation-only field, the layout of a file, the seed, the logger or an observer MUST NOT. |
| FR-CFG-027 | An option that affects results MUST be applied to the effective configuration before it is hashed. |
| FR-CFG-028 | Every row of the parameter-owner table MUST correspond to a field of the configuration, and every non-presentation field MUST have a row. |
| FR-CFG-029 | The default of every field MUST pass validation. |
| FR-CFG-030 | The default stated in an owner's parameter table SHOULD equal the default in the `Default` constructor. |
| FR-CFG-031 | The validation tests MUST include a failing case for each parameter. |
| FR-CFG-032 | `WorldCreated` MUST carry the configuration hash, and `NewWorld` MUST log `world created` once at `Info` with the summary of section 10.1 and no configuration contents. |
| FR-CFG-033 | A chain that fails the viability check MUST be rejected with `ErrNotViable` (satisfied by FR-VIA-004 to 008 and 011; 7.6). |
| FR-CFG-034 | The calendar's months and seasons MUST be ordered lists that keep their written order through loading, canonicalisation and the hash, and an overriding document MUST replace such a list as a whole. |
| FR-CFG-035 | `Config` MUST have a `Yields` section of profiles keyed by region and recipe, sorted by region then recipe, and the production rules of `spec-production.md` 10.2 MUST be run on it and on the recipes with the sentinels `ErrCostShares`, `ErrToolNotDurable`, `ErrEligibleHasTool`, `ErrUnknownSeason`, `ErrModuleDependency`, `ErrDuplicateID` and `ErrOutOfBounds`. |
| FR-CFG-036 | The derived checks of 4.2 for recipes, agents, the warm start and beliefs MUST be run, and a product that does not fit MUST be rejected with `ErrOverflow` carrying the path. |
| FR-CFG-037 | `Config.WarmStart` MUST be nil by default, a non-nil value MUST NOT switch any mechanism on, and it MUST be validated with `ErrWarmStartPrice`, `ErrWarmStartJob` and the reference, duplicate and overflow sentinels of 7.4.1. |
| FR-CFG-038 | A `Role.Strategy` name that is none of `"default"`, `"directed"` and a name registered with `WithStrategy`, a `World.Spawner` name that is not `"default"` or registered with `WithSpawner`, and a `World.Clearing` name that is not `"average"` or registered with `WithClearing` MUST each be rejected by `NewWorld`, after `Validate` has succeeded, with `ErrUnknownStrategy`, `ErrUnknownSpawner` and `ErrUnknownClearing` as a `*ValidationError` with the path and the rule, `Validate` MUST NOT check them, and the three names MUST be in the effective configuration and the hash. |
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
| FR-CFG-060 | `Config.Modules.Demand` MUST be on if and only if non-nil, MUST be written as the table `[demand]` with wants, tastes and region tastes keyed by ID, and MUST be part of the effective configuration and the hash, every field but `Label`, in canonical order. |
| FR-CFG-061 | The demand module's bounds `MaxWants`, `MaxWantItems`, `MaxWantsPerRole`, `MaxWantPeriod`, `MaxWantRank`, `MaxElasticityBP`, `MaxCeilingBP`, `MaxSubstitution` and `MaxTasteScale` MUST be constants of `bounds.go`, and a field outside its bound MUST be rejected with `ErrOutOfBounds`. |
| FR-CFG-062 | Validation MUST reject, with the sentinel for the kind, an unknown commodity, role, region or want in a want item, taste or region taste, a want with no item or all weights 0 (`ErrWantItems`), a durable want item (`ErrWantDurable`), a duplicate want, taste or region taste (`ErrDuplicateID`), and wants products that overflow (`ErrOverflow`). |
| FR-CFG-063 | `World.Spawner` MUST be a registered name defaulting to `"default"`, checked by `NewWorld` and not by `Validate` (`ErrUnknownSpawner`), and the name MUST be hashed. |
| FR-CFG-064 | The eleven spawner parameters and `Role.Closed` MUST be configuration fields with the defaults and bounds of `spec-spawner.md` 11, each with a row in the parameter-owner table and a failing-case test, and `ChangePrivation` not below `RemovePrivation` MUST be rejected with `ErrOutOfBounds` (rule `ladder_order`). |
| FR-CFG-065 | Validation MUST warn, and MUST NOT fail, with `entry_share_window`, `renewal_with_population`, `no_open_role`, the warnings of `spec-demand.md` 10.2 and `wants_exceed_labour`; `group_below_floor` MUST be a bounded run-time `Warn` and not a validation result. |
| FR-CFG-066 | A role with no group in any region MUST be valid and MUST draw no warning. |
| FR-CFG-067 | `Edges` and `RegionPrices` MUST be sorted (edges by `(From, To)` as region index, prices by region then commodity), MUST be hashed with every field, and the graph rules of `spec-regions.md` 9.1 MUST run in `Validate`; `ErrNoRegions` and `ErrSelfEdge` MUST be matched with `errors.Is`. |
| FR-CFG-068 | `Config.Modules.MoneySupply` MUST be on if and only if non-nil, MUST be written as `[module.money_supply]`, MUST be part of the effective configuration and the hash with every field but `Label`, and its bounds MUST be constants of `bounds.go`. |
| FR-CFG-069 | `Config.Modules.Population` MUST be on if and only if non-nil, MUST be written as the table `[population]`, MUST be part of the hash with its schedules in knot order and its region rows in region order, and a schedule outside the bound `MaxKnots` MUST be rejected with `ErrOutOfBounds`. |
| FR-CFG-070 | `Config.Modules.Transport` MUST be on if and only if non-nil, MUST be rejected with `ErrModuleDependency` while the region graph is off, MUST be part of the hash with its modes in ID order and every field but `Label`, and its modes MUST be validated by the rules of `spec-transport.md` 12.1. |
| FR-CFG-071 | `Region.Label`, `World.DistanceLabel`, `Mode.Label` and `MoneyFlow.Label` MUST be presentation only: changing them MUST NOT change the hash, and they MUST be on the allow-list of the hash test. |
| FR-CFG-072 | Validation MUST warn, and MUST NOT fail, with the warnings of `spec-regions.md` 9.2, `spec-money-supply.md` 12.2, `spec-population.md` 9.2 and `spec-transport.md` 12.2; the informational diagnostics `replacement_ratio` and `renewal_ignored` MUST NOT fail validation. |
| FR-CFG-073 | `Config.Modules.Resources` MUST be on if and only if non-nil, MUST be written as the table `[resources]`, MUST be part of the hash with resources in ID order and endowment rows in (region, resource) order, and a draw of an undefined resource MUST be rejected with `ErrUnknownResource`, a bad shape or rate combination with `ErrResourceShape` and a bad abundance curve with `ErrCurve`. |
| FR-CFG-074 | `Config.Modules.Information` MUST be on if and only if non-nil, MUST be written as the table `[modules.information]`, MUST be part of the hash, and `Role.Informed` MUST be rejected with `ErrModuleDependency` while it is nil; the derived checks of `spec-information.md` 9.2 MUST be rejected with `ErrTooMany`. |
| FR-CFG-075 | `Config.Modules.External` MUST be on if and only if non-nil, MUST be written as `[module.external]`, MUST be rejected with `ErrModuleDependency` while the region graph is off, MUST be part of the hash with nodes, lines, gates and season factors in canonical order and every field but `Label`, and MUST NOT hash its runtime state; its rows MUST be validated by the rules of `spec-external-sector.md` 14.1. |
| FR-CFG-076 | Validation MUST warn, and MUST NOT fail, with the warnings of `spec-resources.md` 13.3, `spec-information.md` 9.3 and `spec-external-sector.md` 14.2. |
| FR-CFG-077 | `Config.Modules.Subsistence` MUST be on if and only if non-nil, MUST be written as the table `[modules.subsistence]` with role rows `[modules.subsistence.role.<id>]`, MUST be part of the hash with its lists and rows in canonical order and its three tunables, its bounds `MaxOwnUseRecipes`, `MaxOwnUseTicks` and `MaxRetryJobs` MUST be constants of `bounds.go`, and the rules of `spec-subsistence.md` 13.2 MUST be rejected with `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited` and `ErrOwnUseOverlap`. |
| FR-CFG-078 | `Config.Modules.Migration` MUST be on if and only if non-nil, MUST be written as `[modules.migration]`, MUST be part of the hash with its `Regions` rows in region order, MUST be rejected with `ErrModuleDependency` while `Modules.Information` is nil, and `Role.Fixed` MUST default to false and be hashed as any role field (`spec-migration.md` 11, 12). |
| FR-CFG-079 | `Config.Modules.Merchants` MUST be on if and only if non-nil, MUST be written as `[modules.merchants]` with the merchant block `[role.<id>.merchant]`, MUST be part of the hash with the block of each role, its bounds `MaxLotUnits`, `MaxContracts` and `MaxTimeCostBP` MUST be constants of `bounds.go`, and the rules of `spec-merchants.md` 14.1 MUST be rejected with `ErrUnknownMode` (a root sentinel with the transport module alone, 7.3), `ErrMerchantShape`, `ErrMerchantWeight` and the sentinels named there. |
| FR-CFG-080 | Validation MUST warn, and MUST NOT fail, with the warnings of `spec-subsistence.md` 13.3, `spec-migration.md` 12.3 and `spec-merchants.md` 14.2; `own_use_beats_market` is size-gated and `own_use_partial` informational. |
| FR-CFG-081 | `World.Clearing` MUST be a configuration field holding a registered name, defaulting to `"average"`, with a row in the parameter-owner table (owner `spec-clearing.md` 8), MUST be hashed as a name, and an empty value MUST be rejected with `ErrUnknownClearing` by `NewWorld`. |
| FR-CFG-082 | A configuration with a role of a recipe of `Duration` above 1, more than one agent in a group, and no warm start or a warm start with no job row for the role, MUST produce the informational diagnostic `cold_start_lockstep` once at construction, and it MUST NOT fail validation. |
| FR-CFG-083 | `Validate` MUST NOT see `WithTargetSpread`, and the pure hash function of 8.5 MUST hash a `Config` as given, so that it differs from `World.ConfigHash` when that option was given. |

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

1. **The TOML library.** The format is decided (6.1). Choose the library under
   architecture.md 5.3 and record an ADR with the first code that imports it.
2. **Strict decoding in a chosen library.** Whether it reports unknown keys with
   their paths, or whether `config/` must check the tree itself (this spec assumes
   the second, which is library-independent).

Questions 1 and 2 are code-time decisions and stay open. The rest are decided
(2026-10-10); the answers are kept so the reasons are not lost.

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
   options and cannot see `WithStrategy`, `WithSpawner` or `WithClearing` (7.1.4);
   there are three name checks, strategy, spawner and clearing, run after `Validate`
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
11. **Unify the module table names before the loader is written.** The file layout of the modules differs from spec to spec: `[module.estates]`, `[demand]`, `[module.money_supply]`, `[population]`, and a table of its own for transport (6.2). The framework records each as its spec says and does not unify them now. *Next step:* choose one rule (for example `[module.<name>]` for every module) in the slice, before `config/` is written, and amend the specs and the reference packs together.

## 14. Where this lives in the code

| Concern | Home |
|---------|------|
| `Config` and its section types | root package, `config.go` |
| `Default` constructors | root package, `defaults.go` |
| Bounds (4.1) | root package, `bounds.go` |
| `Validate`, `Warnings`, `ValidationError` and the sentinels | root package, `validate.go`, `errors.go` |
| The producibility fixpoint (7.5) | root package, `validate.go`; a pure function over primitive slices that may live in `internal/` when it grows |
| The viability rules of 7.6 and the sentinels `ErrNotViable`, `ErrWarmStartPrice`, `ErrWarmStartJob` and `ViabilityError` | root package, `validate.go`, `errors.go`; the programmes and checks in `internal/steady` (`spec-viability.md` 21) |
| The three name checks (7.1.4: strategy, spawner, clearing) | root package, `world.go` (they need the options); `ErrUnknownStrategy`, `ErrUnknownSpawner` and `ErrUnknownClearing` in `errors.go` |
| `WarmStart` and its types | root package, `viability.go` (`spec-viability.md` 21), a field of `Config` in `config.go` |
| Canonical order and the hash encoding | root package, `hash.go` |
| `Parse`, `LoadFS`, the merge and the key mapping | package `config/` (the only package with a third-party import) |
| The registry test | the root package's external test package, reading `doc/specs/README.md` by relative path (testing.md 10.2 allows reading inside the module) |

The PR that adds the first of these files also updates the layout in
architecture.md 2.6 and `AGENTS.md`, and the new terms in `doc.go`
(architecture.md 3.2, 14.2).
