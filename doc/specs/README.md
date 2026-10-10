# Economy specs

Design specs for the economy simulation, one area per file. They explain
derivations and calibration, and each one carries the requirements for its area
in a Requirements section. There is no single product requirements document:
testing.md section 4 makes these specs the home of the requirements. The specs
are not normative over `doc/rules/`: where a spec and a rule disagree, the rule
wins and the spec is corrected.

## Index

| Spec | Topic |
|------|-------|
| [spec-currency.md](spec-currency.md) | (core) The minor unit, an optional denomination formatter with an RPG preset, the per-region reference basket, and the principles shared by every spec |
| [spec-goods.md](spec-goods.md) | (core) The attributes of a commodity as data: the unit and how to choose it, mass and bulk with loads and the chargeable size that prices transport and storage, spoilage by shelf life, wear by durability, classes for measurement, raw goods and the reference price; and what is not modelled (quality grades, labour, money as a good) |
| [spec-ledger.md](spec-ledger.md) | (core) The ledger: accounts (agent, pool, estate, escrow, transit and signed boundary accounts), the single transfer primitive with a reason from a registry, money, goods as lots (oldest first, ageing, wear), reservations, flow totals, production and consumption as boundary flows, conservation and the sectoral identities, balance sheets, claims, and the state saved in a checkpoint |
| [spec-production.md](spec-production.md) | (core) Recipes as data (inputs, outputs with cost shares and discard, duration, required tools, endowment draws), execution count and the all-or-nothing start, multi-tick jobs (one per agent; timing, interruption, weight change), yield by region and season through a table with a seeded carried remainder, byproducts, waste and the `disposal` sink, trainee-eligible recipes, the production cost (`UnitCost`) exposed to beliefs, validation, and two worked settings |
| [spec-clearing.md](spec-clearing.md) | (core) The per-region double auction: order admission and reservation, matching with a crossing check and a deterministic tie-break, midpoint pricing with half-to-even rounding, the published market price in thin markets, settlement, and the `Clearing` extension contract |
| [spec-beliefs.md](spec-beliefs.md) | (core) Price beliefs: a centre and half-width per agent and commodity that cannot invert, bounds from the reference price and the unit cost, an asynchronous, once-per-order, bounded-step update driven by price age, the best unmatched quotes and the stock ratio, drawing a limit price, a per-market price window, seeding and a warm-start hook, cohorts, the relation to `Strategy`, and the `FR-BEL`/`NFR-BEL` requirements |
| [spec-agents.md](spec-agents.md) | (core) Agents as households: the agent's state and where each piece lives, roles, the derived tradable set, what an agent is born with and what removal does, upkeep as a boundary flow with fed, unfed and hunger as the signals for population and spawner, the `Strategy` extension point (three methods `Work`, `Trade`, `Revise`, plain-data views, what the world checks, registration by name), the default strategy (work share with a rest band and a survival rule, recipe choice by net value, targets and cover, a three-tier budget and a saving rule, order building, belief revision), no inventory capacity, cohorts, the player as a directed external agent, a staple's demand elasticity, parameters, validation, observability, the checkpoint state, entries for the stability register, and the `FR-AGT`/`NFR-AGT` requirements |
| [spec-spawner.md](spec-spawner.md) | (core) The spawner: the extension interface (`Decide`, a plan of remove, reduce, switch, apply and enter), bankruptcy by privation, the failure ladder (change role, relief, then reduction and removal), replacement and the entry rule with damped occupation change, pool funding of entrants, the hook for population, external agents, events and the `FR-SPN`/`NFR-SPN` requirements |
| [spec-population.md](spec-population.md) | (module) Births and deaths as the formation and dissolution of households, driven by the household's own welfare ratio through two schedules, region scales, the hooks `PopBirths` and `PopDeaths` of the spawner, the stake that funds a birth, `Renewal` ignored while the module is on, the Malthusian fixed point and what it needs from RES, collapse and recovery, entries for the register and the `FR-POP`/`NFR-POP` requirements |
| [spec-demand.md](spec-demand.md) | (module) Wants beyond upkeep as data (alternatives, quantity per period, rank, elasticity with a ceiling, integer substitution), roles as classes with taste rows, the price schedule with a choke price, whole units from a carried remainder, the ladder funded by the saving rule's cap, consumption of wants, the closed-economy ratio above 1, what the money specs must supply, entries for the register, and the `FR-DEM`/`NFR-DEM` requirements |
| [spec-stability.md](spec-stability.md) | (core) The feedback-loop register: the entry format, 86 loops with sign, delay, damper, condition and test, the catalogue of 52 dampers and the rule for adding one, three numeric analyses (the target-income labour loop, the belief/price/inventory loop, the production and occupation cobwebs), the failure regimes, the scenarios the acceptance bands must cover, the checklist for spec authors, the loops no spec yet dampens, and the `FR-STB`/`NFR-STB` requirements |
| [spec-viability.md](spec-viability.md) | (core) Viability and the steady state: four layers of the check (sourced and runnable, productive cycles by the Hawkins-Simon condition, labour sufficient by an exact linear programme, storable), the programme and its dual as prices, wages and flows, exact integer arithmetic, why the solver is a pure core function and not part of NewWorld, the warm-start table (seed prices, money, stock, staggered jobs), regions and lanes, the rules added to config validation, what viability does not prove, and the FR-VIA/NFR-VIA requirements |
| [spec-reference-chain.md](spec-reference-chain.md) | (fixture) The standard test fixture: a base chain as TOML, seven product lines to the first consumer's size, variants for one, two, three and five regions, a starvation variant, a demand overlay, a second setting and six defect packs, layered by key; solver-generated warm starts; the scenarios S01 to S27 and A1 to A10 assigned to packs with measures and the form of their pass; the invariants over seeds, the golden rendering, the configuration-only chain test; the `FR-REF` requirements |
| [spec-scale.md](spec-scale.md) | (core) Population scale: an agent's weight (the households it stands for), deriving agents from a census at a chosen scale with a floor, what other specs must do with a weight, the cost model and a provisional budget |
| [spec-config.md](spec-config.md) | (core) The configuration framework: its shape and how modules are switched on, bounds with their derivation, defaults defined once, the proposed TOML file format and pack layering, validation (framework, structural rules, rules from other specs), the configuration hash, and the parameter registry with the gaps found in it. Written ahead of the specs it depends on; its field catalogue grows |
| [spec-welfare-and-wages.md](spec-welfare-and-wages.md) | (module) The welfare ratio, wage formulas, observed wages and premiums, the target-income rule, in-kind pay |
| [spec-skill-barriers.md](spec-skill-barriers.md) | (module) Required tools, how the skill premium is checked, traineeship training |
| [spec-estates.md](spec-estates.md) | (core, module) What a dead agent leaves, the regional money pool, starting money and relief |
| [spec-regions.md](spec-regions.md) | (core) The region as one settlement and one market, its fields and who owns them, directed edges with an integer distance, deterministic queries with a total-order tie-break, components and islands, one override mechanism and the reference-price override, the operation that moves an account between regions and its hook for goods in transit, the edge status for shocks, entries for the register and the `FR-REG`/`NFR-REG` requirements |
| [spec-money-supply.md](spec-money-supply.md) | (core, module) Where money enters and leaves as named flows in data: grants and levies on a class of households or a pool, four non-reactive shapes, a window, a cumulative limit and a runtime scale seam; declared reasons; the money supply and its identities; the closed economy and its derived liquidity floor; the primitives reserved for the external sector; entries for the register; the `FR-MON`/`NFR-MON` requirements |
| [spec-carriage.md](spec-carriage.md) | (module) Carrier pay by merchants: the fee is the market price of haul units, an ordinary commodity (the seed of the contract spec) |
| [spec-transport.md](spec-transport.md) | (module) Transport modes as data (speed, a capacity pair per carrying unit, self-carried or hired), travel time from a REG distance, the shipment (request, 23 ordered refusal checks, dispatch, transit in a reused Transit account, arrival, diverted cargo), goods in transit in the ledger, audit and checkpoint, the decision on the carriage unit (N-076), what transport exposes to strategies and information, hooks for risk and infrastructure, entries for the register, a three-region worked example and the `FR-TRN`/`NFR-TRN` requirements |
| [evidence.md](evidence.md) | The sources behind the numbers, and which facts are unverified |
| [bazaarbot-reference.md](bazaarbot-reference.md) | What the bazaarBot source does, its defects, and where each piece goes in the port (a reference, with no requirements and no area code) |

Reading order: currency, goods, ledger, production, clearing, beliefs, agents, spawner, population, demand, scale, config, welfare and wages, skill barriers, estates, regions, money supply, carriage, transport, viability, stability, reference chain.
The layer in brackets (core, module, extension) is defined in [../roadmap.md](../roadmap.md) and
[ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md). A spec says which of its
parts are core and which are optional.
The plan for the specs not yet written is in [../roadmap.md](../roadmap.md).

## Requirement IDs and area codes

A requirement has an ID `FR-<AREA>-NNN` (functional) or `NFR-<AREA>-NNN`
(non-functional), where `<AREA>` is the three-letter code of the spec that
defines it (testing.md section 4.1). Tests cite IDs in a `Requirement:` line.
Rules for the IDs:

- An ID is stable and unique across the module. A code is never reused.
- A requirement that moves to another spec keeps its ID and its original code.
- A requirement is a single testable statement using MUST, SHOULD or MAY as in
  RFC 2119, and it lives in the Requirements section of its spec.
- A spec with no Requirements section yet says so in its preamble. Until it is
  written, no test may cite an ID from it.

| Code | Spec | Status |
|------|------|--------|
| `CUR` | `spec-currency.md` | exists; Requirements section written (FR-CUR-001 to 023, NFR-CUR-001); all numbers are placeholders |
| `WEL` | `spec-welfare-and-wages.md` | exists; Requirements section written (FR-WEL-001 to 038, NFR-WEL-001 to 003); all numbers are placeholders |
| `SKL` | `spec-skill-barriers.md` | exists; Requirements section written (FR-SKL-001 to 028, NFR-SKL-001 to 002); all numbers are placeholders |
| `EST` | `spec-estates.md` | exists; Requirements section written (FR-EST-001 to 024, NFR-EST-001); bounds and defaults are placeholders |
| `CAR` | `spec-carriage.md` | exists; Requirements section written (FR-CAR-001 to 013; 001 and 002 moved to the future contract spec, 005 to 007 superseded by `spec-transport.md`); depends on the unwritten contract and merchant specs |
| `CLR` | `spec-clearing.md` | exists; Requirements section written (FR-CLR-001 to 033, NFR-CLR-001 to 003) |
| `BEL` | `spec-beliefs.md` | exists; Requirements section written (FR-BEL-001 to 039, NFR-BEL-001 to 003); all defaults are placeholders |
| `AGT` | `spec-agents.md` | exists; Requirements section written (FR-AGT-001 to 077, NFR-AGT-001 to 004); all defaults are placeholders |
| `LED` | `spec-ledger.md` | exists; Requirements section written (FR-LED-001 to 051, NFR-LED-001 to 004) |
| `PRO` | `spec-production.md` | exists; Requirements section written (FR-PRO-001 to 049, NFR-PRO-001 to 004); all numbers are placeholders |
| `SPN` | `spec-spawner.md` | exists; Requirements section written (FR-SPN-001 to 052, NFR-SPN-001 to 003); all numbers are placeholders |
| `WLD` | `spec-world.md` | planned: tick phases, scheduler, determinism, options |
| `REG` | `spec-regions.md` | exists; Requirements section written (FR-REG-001 to 057, NFR-REG-001 to 004); core; the region, the directed graph, overrides, relocation; bounds are placeholders |
| `CFG` | `spec-config.md` | exists; Requirements section written (FR-CFG-001 to 066, NFR-CFG-001 to 003); the field catalogue is section 3.7 |
| `OBS` | `spec-observability.md` | planned: events, snapshot, metrics |
| `GDS` | `spec-goods.md` | exists; Requirements section written (FR-GDS-001 to 023, NFR-GDS-001) |
| `REF` | `spec-reference-chain.md` | exists; Requirements section written (FR-REF-001 to 040, NFR-REF-001 to 002); fixture; bands unset, all numbers are placeholders |
| `RES` | `spec-resources.md` | planned: regional endowments, depletion, regeneration |
| `CAL` | `spec-calendar.md` | exists; Requirements section written (FR-CAL-001 to 021, NFR-CAL-001 to 003); the structure of time, derived `TicksPerYear`, optional seasons |
| `POP` | `spec-population.md` | exists; Requirements section written (FR-POP-001 to 040, NFR-POP-001 to 004); module; all numbers are placeholders |
| `MIG` | `spec-migration.md` | planned: migration triggers and destinations |
| `INF` | `spec-information.md` | planned: local prices and delayed rumours |
| `TRN` | `spec-transport.md` | exists; Requirements section written (FR-TRN-001 to 046, NFR-TRN-001 to 004); module; all numbers are placeholders |
| `STO` | `spec-storage.md` | planned: warehousing and storage cost |
| `MER` | `spec-merchants.md` | planned: merchant decisions and delivery contracts |
| `INT` | `spec-integration.md` | planned: queries, player agents, external shocks |
| `MON` | `spec-money-supply.md` | exists; Requirements section written (FR-MON-001 to 046, NFR-MON-001 to 003); core for the identities, module for the flows; all numbers are placeholders |
| `FIS` | `spec-fiscal.md` | planned: taxes, tolls, treasury, granaries |
| `SCL` | `spec-scale.md` | exists; Requirements section written (FR-SCL-001 to 019, NFR-SCL-001 to 004); cost figures provisional |
| `DEM` | `spec-demand.md` | exists; Requirements section written (FR-DEM-001 to 043, NFR-DEM-001 to 004); module; all numbers are placeholders |
| `RSK` | `spec-risk.md` | planned: loss, conflict, banditry, disasters |
| `LND` | `spec-land.md` | planned: land tenure and rent |
| `EMP` | `spec-employment.md` | planned: hired labour |
| `CHK` | `spec-checkpoint.md` | planned: save and restore |
| `IND` | `spec-indicators.md` | planned: GDP, price index, inequality |
| `SET` | `spec-settlements.md` | planned: settlement size classes and what they unlock |
| `POL` | `spec-polity.md` | planned: realms, provinces, jurisdictions |
| `ORG` | `spec-institutions.md` | planned: associations, charters, price controls |
| `CAP` | `spec-infrastructure.md` | planned: mills, ships, roads, granaries |
| `CRD` | `spec-credit.md` | planned extension: loans and interest |
| `INS` | `spec-insurance.md` | planned extension: insurance |
| `EXT` | `spec-external-sector.md` | planned: the outside world and money flows across the boundary |
| `SAV` | `spec-saving-and-investment.md` | planned: saving, hoarding, investment decisions |
| `TEC` | `spec-technology.md` | planned: productivity growth |
| `PLC` | `spec-policy.md` | planned: policy actors and rules |
| `STY` | `spec-stylised-facts.md` | planned: testable macro properties and failure regimes |
| `CLB` | `spec-calibration.md` | planned: tuning knobs and calibration |
| `EXP` | `spec-experiments.md` | planned: experiment harness (core) |
| `STB` | `spec-stability.md` | exists; Requirements section written (FR-STB-001 to 031, NFR-STB-001 to 002); the register has 86 loops, 52 dampers and 39 scenarios (S01 to S40, S15 unused); all numbers are placeholders |
| `VIA` | `spec-viability.md` | exists; Requirements section written (FR-VIA-001 to 059, NFR-VIA-001 to 004); all numbers are placeholders |
| `SUB` | `spec-subsistence.md` | planned: household production as a floor |

A planned code is reserved. If a planned spec is merged with another or dropped,
the code stays reserved and unused.

## Shared conventions

- Before writing or changing a spec, read [../notes.md](../notes.md) in full,
  quote its read-receipt tag, and settle or update the notes for the spec's area
  code and for `ALL` (`AGENTS.md`).
- The shared principles are in `spec-currency.md` section 2: one unit of
  account, integer arithmetic with a stated rounding rule, a tick as the unit of
  time and a year of `TicksPerYear` ticks,
  data not code, and wages that are measured not configured.
- Terms in **bold** on first use in a spec are new domain terms. They must be
  added to the glossary in `doc.go` before any code uses them
  (architecture.md 3.2).
- A section reference in a spec is to a section of the same file unless it names
  another file.
- Parameters have exactly one owner, listed below. A spec that uses another
  spec's parameter links to it and does not restate its default.

## Parameter owners

| Parameter | Scope | Owner |
|-----------|-------|-------|
| `ReferenceBasket` (`B_r`) | Region | `spec-currency.md` section 5 |
| `Unskilled` | Role | `spec-welfare-and-wages.md` section 4.2 |
| `TargetRatio` | Role | `spec-welfare-and-wages.md` section 5 |
| `TargetSpread` | Role | `spec-welfare-and-wages.md` section 5.1 |
| `WithTargetSpread` option | World | `spec-welfare-and-wages.md` section 5.2 |
| `DefaultRatio` | Role | `spec-welfare-and-wages.md` section 2.2 |
| `InKind` | Role | `spec-welfare-and-wages.md` section 6 |
| `CapitalReturn` | Region | `spec-skill-barriers.md` section 2.5 |
| `TrainingTicks` | Role | `spec-skill-barriers.md` section 4.1 |
| `MaxTrainees` | Role | `spec-skill-barriers.md` section 4.4 |
| `TeachingLoad` | Role | `spec-skill-barriers.md` section 4.2 |
| `EstateMarkdown` | Region | `spec-estates.md` section 3 |
| `PoolRelease` | Region | `spec-estates.md` section 2.6 |
| `ReliefBelow` | Region | `spec-estates.md` section 2.6 |
| `MerchantFeeShare` | Role | `spec-carriage.md` section 2.2 |
| `Mass` | Commodity | `spec-goods.md` section 8 |
| `Bulk` | Commodity | `spec-goods.md` section 8 |
| `BulkFactor` | World | `spec-goods.md` section 8 |
| `ShelfLife` | Commodity | `spec-goods.md` section 8 |
| `Durability` | Commodity | `spec-goods.md` section 8 |
| `Classes` | Commodity | `spec-goods.md` section 8 |
| `Raw` | Commodity | `spec-goods.md` section 8 |
| `ReferencePrice` | Commodity (region override: `RegionPrices`, `spec-regions.md` section 5.2) | `spec-goods.md` section 8 |
| `HoursPerDay` | Calendar | `spec-calendar.md` section 3.1 |
| `DaysPerWeek` | Calendar | `spec-calendar.md` section 3.1 |
| `Months` | Calendar | `spec-calendar.md` section 3.1 |
| `TicksPerDay` | Calendar | `spec-calendar.md` section 3.1 |
| `DaysPerTick` | Calendar | `spec-calendar.md` section 3.1 |
| `StartDay` | Calendar | `spec-calendar.md` section 3.1 |
| `Seasons` | Calendar | `spec-calendar.md` section 3.1 |
| `SeasonStart` | Calendar | `spec-calendar.md` section 3.1 |
| `ExpectedWorkTicks` | Role | `spec-welfare-and-wages.md` section 4.2 |
| `Households` | Group (role in a region) | `spec-scale.md` section 3.1 |
| `Scale` | Group; world default 1 | `spec-scale.md` section 3.1 |
| `MinAgents` (`K`) | World | `spec-scale.md` section 5.2 |
| `Duration` | Recipe | `spec-production.md` section 5 |
| `Sited` | Recipe | `spec-production.md` section 6.2 |
| `TraineeEligible` | Recipe | `spec-production.md` section 8 |
| `TraineeYield` | Recipe | `spec-production.md` section 8.2 |
| `WearPerUse` | Recipe tool | `spec-production.md` section 4.4 |
| `CostShare` | Recipe output | `spec-production.md` section 3.2 |
| `Discard` | Recipe output | `spec-production.md` section 7.2 |
| `Draws` | Recipe | `spec-production.md` section 6.5 (resource values owned by RES) |
| `YieldProfile` | Region and recipe | `spec-production.md` section 6.1 (values move to `spec-resources.md` when it exists) |
| `BeliefStep` | World | `spec-beliefs.md` section 11 |
| `BeliefAdjustProb` | World | `spec-beliefs.md` section 11 |
| `BeliefSignificance` | World | `spec-beliefs.md` section 11 |
| `BeliefStockLow` | World | `spec-beliefs.md` section 11 |
| `BeliefStockHigh` | World | `spec-beliefs.md` section 11 |
| `BeliefMaxAge` | World | `spec-beliefs.md` section 11 |
| `BeliefWidthInit` | World | `spec-beliefs.md` section 11 |
| `BeliefWidthMin` | World | `spec-beliefs.md` section 11 |
| `BeliefWidthMax` | World | `spec-beliefs.md` section 11 |
| `BeliefWidthStep` | World | `spec-beliefs.md` section 11 |
| `CostFloor` | World | `spec-beliefs.md` section 11 |
| `CostCeiling` | World | `spec-beliefs.md` section 11 |
| `RefBandLow` | World | `spec-beliefs.md` section 11 |
| `RefBandHigh` | World | `spec-beliefs.md` section 11 |
| `PriceWindow` | World | `spec-beliefs.md` section 11 |
| `Upkeep` | World (also Role, replacing the world's) | `spec-agents.md` section 5.2, 12 |
| `ExtraUpkeep` | Region | `spec-agents.md` section 5.2, 12 |
| `CoverTicks` | World | `spec-agents.md` section 12 |
| `InputJobs` | World | `spec-agents.md` section 12 |
| `OutputCoverTicks` | World | `spec-agents.md` section 12 |
| `FavourFloor` | World | `spec-agents.md` section 12 |
| `ReserveTicks` | World | `spec-agents.md` section 12 |
| `SpendMaxShare` | World | `spec-agents.md` section 12 |
| `SpendHalfTicks` | World | `spec-agents.md` section 12 |
| `RestBand` | World | `spec-agents.md` section 12 |
| `ProduceHaltRatio` | World | `spec-agents.md` section 12 |
| `BasketTolerance` | World | `spec-agents.md` section 12 (closes the unnamed tolerance of `spec-currency.md` 5.3) |
| `Recipes` | Role | `spec-agents.md` section 3.3 |
| `Trades` | Role | `spec-agents.md` section 3.3 |
| `StartMoney` | Role | `spec-agents.md` section 3.3 |
| `StartStock` | Role | `spec-agents.md` section 3.3 |
| `Strategy` | Role (a registered name) | `spec-agents.md` section 6.6 |
| `WarmStart` | Config section (nil = off) | `spec-viability.md` section 8.1 |
| `WelfareWindow` | World | `spec-welfare-and-wages.md` section 2.3.1 |
| `WindowCycles` | World | `spec-welfare-and-wages.md` section 2.3.3 |
| `MinWorkShare` | Role | `spec-welfare-and-wages.md` section 5.3 |
| `BeliefEvidenceShare` | World | `spec-beliefs.md` section 11 |
| `ID` | Commodity | `spec-goods.md` section 8 |
| `ID` | Recipe | `spec-production.md` section 3.1, 11 |
| `Inputs` | Recipe | `spec-production.md` section 3.1, 11 |
| `Outputs` | Recipe | `spec-production.md` section 3.1, 11 |
| `Requires` | Recipe | `spec-production.md` section 3.1, 11 |
| `Region` | Yield | `spec-production.md` section 6.1 |
| `Recipe` | Yield | `spec-production.md` section 6.1 |
| `All` | Yield | `spec-production.md` section 6.1 |
| `ID` | Role | `spec-agents.md` section 3.3 |
| `ID` | Region | `spec-currency.md` section 5 |
| `Role` | Group | `spec-scale.md` section 3.1 |
| `Prices` | WarmStart | `spec-viability.md` section 8.1 |
| `Wages` | WarmStart | `spec-viability.md` section 8.1 |
| `Groups` | WarmStart | `spec-viability.md` section 8.1 |
| `Pools` | WarmStart | `spec-viability.md` section 8.1 |
| `Region` | Group | `spec-scale.md` section 3.1 |
| `Seasons` | Yield | `spec-production.md` section 6.1 |
| `ReviewTicks` | World | `spec-spawner.md` section 11 |
| `EntryShare` | World | `spec-spawner.md` section 11 |
| `RoleHysteresis` | World | `spec-spawner.md` section 11 |
| `RoleCooldown` | World | `spec-spawner.md` section 11 |
| `EntryRatio` | World | `spec-spawner.md` section 11 |
| `ChangePrivation` | World | `spec-spawner.md` section 11 |
| `RemovePrivation` | World | `spec-spawner.md` section 11 |
| `DeathStep` | World | `spec-spawner.md` section 11 |
| `ExitShare` | World | `spec-spawner.md` section 11 |
| `Renewal` | World | `spec-spawner.md` section 11 |
| `EntryPoolShare` | World | `spec-spawner.md` section 11 |
| `Spawner` | World | `spec-spawner.md` section 3.1.3 |
| `Closed` | Role | `spec-spawner.md` section 11 |
| `Wants` | Demand | `spec-demand.md` section 9 |
| `Items` | Want | `spec-demand.md` section 9 |
| `Quantity` | Want | `spec-demand.md` section 9 |
| `Period` | Want | `spec-demand.md` section 9 |
| `Rank` | Want | `spec-demand.md` section 9 |
| `Elasticity` | Want | `spec-demand.md` section 9 |
| `Ceiling` | Want | `spec-demand.md` section 9 |
| `Substitution` | Want | `spec-demand.md` section 9 |
| `Scale` | Taste | `spec-demand.md` section 9 |
| `Commodity` | Want item | `spec-demand.md` section 9 |
| `Weight` | Want item | `spec-demand.md` section 9 |
| `Want` | Taste | `spec-demand.md` section 9 |
| `Edges` | Config section (optional) | `spec-regions.md` section 4.1 |
| `From` | Edge | `spec-regions.md` section 4.1 |
| `To` | Edge | `spec-regions.md` section 4.1 |
| `Distance` | Edge | `spec-regions.md` section 4.1 |
| `RegionPrices` | Config section (optional) | `spec-regions.md` section 5.2 |
| `Region` | RegionPrice | `spec-regions.md` section 5.2 |
| `Commodity` | RegionPrice | `spec-regions.md` section 5.2 |
| `Price` | RegionPrice | `spec-regions.md` section 5.2 |
| `Flows` | Modules.MoneySupply (module; nil = off) | `spec-money-supply.md` section 4.1 |
| `ID` | MoneyFlow (also the ledger reason) | `spec-money-supply.md` sections 4.5, 11.1 |
| `Kind` | MoneyFlow | `spec-money-supply.md` sections 4.1, 11.1 |
| `Target` | MoneyFlow | `spec-money-supply.md` sections 4.3, 11.1 |
| `Amount` | MoneyFlow | `spec-money-supply.md` sections 4.2, 11.1 |
| `Level` | MoneyFlow | `spec-money-supply.md` sections 4.2, 11.1 |
| `RateBP` | MoneyFlow | `spec-money-supply.md` sections 4.2, 11.1 |
| `Exempt` | MoneyFlow | `spec-money-supply.md` sections 4.2, 11.1 |
| `Per` | MoneyFlow | `spec-money-supply.md` sections 4.2, 11.1 |
| `Start` | MoneyFlow | `spec-money-supply.md` section 4.4.1 |
| `Until` | MoneyFlow | `spec-money-supply.md` section 4.4.1 |
| `Limit` | MoneyFlow | `spec-money-supply.md` section 4.4.2 |
| `NotIncome` | MoneyFlow | `spec-money-supply.md` section 7.2 |
| `Population` | Modules (module; nil = off) | `spec-population.md` section 8 |
| `BirthSchedule` | Population | `spec-population.md` section 8 |
| `DeathSchedule` | Population | `spec-population.md` section 8 |
| `Ratio` | Knot | `spec-population.md` section 8 |
| `Rate` | Knot | `spec-population.md` section 8 |
| `MinHouseholds` | Population | `spec-population.md` section 8 |
| `BirthStake` | Population | `spec-population.md` section 8 |
| `Region` | RegionPopulation | `spec-population.md` section 8 |
| `BirthScale` | RegionPopulation | `spec-population.md` section 8 |
| `DeathScale` | RegionPopulation | `spec-population.md` section 8 |
| `HouseholdCap` | RegionPopulation | `spec-population.md` section 8 |
| `Transport` | Modules (module; nil = off) | `spec-transport.md` section 3.1 |
| `Modes` | Transport | `spec-transport.md` section 3.1 |
| `ID` | Mode | `spec-transport.md` section 3.1 |
| `Speed` | Mode | `spec-transport.md` section 3.1 |
| `UnitMass` | Mode | `spec-transport.md` section 3.1 |
| `UnitBulk` | Mode | `spec-transport.md` section 3.1 |
| `Crew` | Mode | `spec-transport.md` section 3.1 |
| `Vehicle` | Mode | `spec-transport.md` section 3.1 |
| `WearPerTick` | Mode | `spec-transport.md` section 3.1 |
| `Feed` | Mode | `spec-transport.md` section 3.1 |
| `Haul` | Mode | `spec-transport.md` section 3.1 |
| `HaulSpan` | Mode | `spec-transport.md` section 3.1 |

## New domain terms to add to the glossary

Minor unit, denomination, ladder, formatter, precision, household, reference basket, welfare ratio, padded
history, observed wage, observed premium, working tick, commercial trip,
target-income rule, tool, durability, estate, pool, relief, carriage market,
trainee, trainer, traineeship, trainee-eligible recipe, account, boundary, faucet, sink, pack, layer, policy, feedback loop, damper, viability, steady state, subsistence floor, bid, ask, limit price, admission, reservation, crossing, tie-break key, market price, price age, reference price, weight, scale, group, effective configuration, configuration hash, pack, presentation-only field, warning, sourced, producible, year, commodity, unit, mass, bulk, load, container, capacity, binding dimension, chargeable size, lot, born tick, transfer, reason (of a transfer), claim, balance sheet, sector, flow total, pseudo-agent, earmark, named lot, audit, shelf life, age, spoil, durability (listed), durable, tool (listed), class, raw, calendar, calendar day, absolute day, moment, season, season cycle, ordered list, job, execution, executor, principal, duration, source (recipe with no inputs), co-product, byproduct, discard (output), disposal, yield factor, effective factor, site (sited recipe), yield profile, yield shock, residue, draw (endowment draw), pending output, work in progress, trainee yield, unit cost, cost anchor, cost share, price belief, centre (of a belief), width (of a belief), belief step, effective range, outcome (of an order), adjustment gate, stock ratio, target stock, seed price, price window, reference band, cost band, re-clamp, upkeep, effective upkeep, fed, unfed, hunger, tradable set, tradable (commodity), role, strategy, intent, trade plan, job request, purpose (of a bid), agent view, outcome view, work share, rest band, survival rule, planned recipe, reserve (units), keep, gap, surplus, target (stock), use target, output target, cover period, tier (of a bid), reserve money, spend cap, favourability, external agent, directive, directed strategy, mailbox, production stop, household labour, activity, start capacity, mean factor, net vector, reference programme, surplus factor, labour share to subsist, available work share, warm start, seed table, virtual start offset, stagger (rule), supply gap, lean reserve, balanced census, lane, residual, drift, register entry, failure regime, pinned market, stability scenario, evidence offset, effective window, working household-tick, wear cost, work floor, evidence share, full cost, regional wage, anchor (reference), starting wage, full unit cost, construction phase, construction bucket, periodic season, stated lot key, held to work (by the floor), spawner, privation, ladder (failure ladder), bankruptcy, entry share, entry limit, review (occupation review), cooldown, viability (of a role), expected ratio, renewal, entrant, mortal (external agent), role change, want, rank, taste, schedule, choke price, ceiling (of a schedule), substitution share, want carry, discretionary budget, nominal seed, final demand (wants), variant, overlay, product line, balanced census, stable-state predicate, bands document, shock script, cold variant, report-only (scenario), region, region graph, edge, route (an edge), distance, `RegionID`, `EdgeID`, `NoRegion`, hop, shortest path, next hop, component (strongly connected), island, edge status, effective length, factor (of an edge), relocation, relocation totals, region override, effective reference price, `Graph`, money supply, money flow, grant, levy, top-up, class (of accounts), flow carry, flow scale, declared reason, holding charge, liquidity floor, spending threshold, closed economy, balanced pair, birth (of a household), death (of a household), schedule (birth, death), knot, replacement ratio, exposed household, natural increase, birth stake, household cap, death carry, birth carry, fixed factor, gain (of the population loop), money-limited population, mode, self-carried mode, hired mode, carrying unit, crew (of a mode), haul unit, haul span, traveller, shipper, consignee, shipment, dispatch, arrival, transit account, transit factor, away (agent), inbound, diverted (cargo).

## Decisions taken

| Question | Decision | Where |
|----------|----------|-------|
| Where an estate's money goes | A regional pool that funds the starting money of new agents | `spec-estates.md` section 2 |
| What an idle pool does | Releases a configurable fraction per tick as relief to agents below subsistence | `spec-estates.md` section 2.6 |
| What happens to a tool at death | Listed by the estate at a durability-discounted ask | `spec-skill-barriers.md` section 2.7 |
| Whether training is modelled | Yes: a traineeship (apprenticeship in a medieval setting) under a trainer, a second barrier beside the tool | `spec-skill-barriers.md` section 4 |
| Who funds training | The trainer, repaid by the trainee's work. No fee and no credit | `spec-skill-barriers.md` section 4.2 |
| What counts as a working tick | Recipe or commercial travel for roles with recipes; trades or commercial travel for roles without | `spec-welfare-and-wages.md` section 4.3 |
| How the premium is checked | Ordinal properties required, historical band reported, derived equilibrium once the entry rule is built | `spec-skill-barriers.md` section 3 |
| How entrants decide to enter | Return on tool plus training cost must beat `CapitalReturn` | `spec-skill-barriers.md` sections 2.5 and 4.5 |
| How merchants set the carriage fee | The market price of haul units, an ordinary commodity made by carriers' recipes and traded like any good (N-076); no separate carriage market | `spec-carriage.md` section 2.2, `spec-transport.md` section 8 |
| Tool prices | Emergent from recipe cost; never configured | `spec-skill-barriers.md` section 2.4 |
| Tool durability and price ranges | The placeholders are accepted | `spec-skill-barriers.md` section 2.9 |
| Estate listing price with no recent trades | The region's effective reference price, times remaining over full durability for a durable, less the per-tick markdown | `spec-estates.md` section 2.7.3, `spec-skill-barriers.md` section 2.7 |
| `TargetSpread` default | 0 until experiments say otherwise | `spec-welfare-and-wages.md` section 5.2 |
| Does a bid below an ask trade? | No: only crossing pairs trade (bazaarBot defect 1) | `spec-clearing.md` sections 4.3 and 10 |
| Can an agent bid beyond its cash or ask beyond its stock? | No: every order is backed by a reservation at admission and rejected whole otherwise (bazaarBot defect 2) | `spec-clearing.md` section 3 |
| How are ties and rounding settled? | A seeded per-order key then input position; midpoint and volume-weighted mean round half to even | `spec-clearing.md` sections 4.2 and 5.2 |
| How is a large population held at an affordable cost? | One mechanism: an agent has an integer weight, the households it stands for; the configuration states households per group and the agents are derived at a scale, with a floor of `K` agents per group | `spec-scale.md` sections 2, 3 and 5 |
| Are regions with no agents part of scale? | No: they are the outside world, `spec-external-sector.md`. Changing scale in a running world is left out of version 1 | `spec-scale.md` section 8 |
| Where do tunables, bounds and rule constants live? | Tunables in `Config` with a `Default` constructor each; bounds in `bounds.go`, derived from `int64` limits; rule constants beside their code | `spec-config.md` sections 4 and 5, architecture.md 9.5 to 9.8 |
| How is a module switched on? | By the presence of its entry in `Config.Modules`; no separate flag | `spec-config.md` section 3.2 |
| How are entities written in a file? | As tables keyed by ID, so documents merge by key and file order never matters; the format is TOML (decided); the parsing library is not | `spec-config.md` section 6 |
| What makes a good divisible, and how big is a unit? | The author's choice of unit; quantities are whole units. Validation requires upkeep of at least one unit and a price at or above the resolution floor | `spec-goods.md` section 2 |
| How are money and goods created, destroyed and moved? | Only by a transfer between two accounts with a reason from a closed registry; boundary accounts are signed, so the total over all accounts is always zero and every faucet and sink is a named flow. Recipes, consumption, spoilage and wear are boundary flows | `spec-ledger.md` sections 2, 7 and 8 |
| How does the estate sell goods? | The estate is a per-region pseudo-agent with one ledger account holding both the pool money and the unsold goods; it places asks only, one per lot it holds, priced by remaining durability with a per-tick markdown. Sale proceeds are therefore the pool | `spec-estates.md` section 2.7, `spec-clearing.md` section 3.1.2, `spec-ledger.md` section 6.7 |
| Can a consumer audit the ledger? | Yes: `World.Audit()` between ticks, and the option `WithAudit()` which audits every step and makes `Step` return `ErrAuditFailed`. It reads only, changes no result and is not hashed | `spec-ledger.md` section 12.4 |
| How are goods held? | As lots by commodity, born tick and remaining durability, merged into a canonical form; taken oldest first, then most worn first; age and wear travel with the goods | `spec-ledger.md` section 5 |
| What backs an order, and what holds money for a contract? | Order: a reservation counter, which exists only inside a tick. Contract: a transfer into an escrow account, not a reservation | `spec-ledger.md` sections 6.1 to 6.6 |
| Are mass and bulk one dimension or two? | Two: `Mass` and `Bulk`. A load is the pair of sums and fits a container only if both are within capacity. For pricing, transport and storage use one chargeable size per good, `max(mass, bulk * BulkFactor / 10 000)`, so they remain per-unit markets | `spec-goods.md` section 3 |
| Do goods spoil gradually or at a fixed age? | At a fixed age: a lot spoils whole at the end of the tick its age reaches the shelf life, as a named sink; oldest sold first | `spec-goods.md` section 4 |
| Which spec owns the wear fields? | `Durability` (capacity) on the commodity belongs to goods; `Requires` and `wear_per_use` (rate) belong to production | `spec-goods.md` section 5.3 |
| Are there quality grades? | No: a grade is a separate commodity, so each commodity clears at one price | `spec-goods.md` section 7.1 |
| Who owns the reference price? | The commodity owns a required default; a region may override it | `spec-goods.md` section 6.3 |
| Are the length of a year and the working time of a role fixed? | No: the year is derived from a `Calendar` (default Earth, 365 ticks) and `ExpectedWorkTicks` (Role, default 250) is a parameter. A "day" in the specs means a tick; a calendar day is a different unit. `ExpectedWorkTicks` is an expectation for calibration and the entry rule; days actually worked still emerge | `spec-calendar.md` section 3.2, `spec-welfare-and-wages.md` sections 3.3 and 4.2 |
| What does the configuration hash cover? | The effective configuration: defaults applied, entities in ID order, result-affecting options included; not the seed, logger, observers or presentation-only fields | `spec-config.md` section 8 |
| Price in a market where nothing trades | The previous price carried forward, starting from the reference price, with a price age | `spec-clearing.md` section 5.4 |
| How is time configured? | A `Calendar` section, Earth by default: hours in a day, days in a week, months of any lengths, ticks in a day or days in a tick (one of the two), a start day, and an optional cycle of seasons that partitions the year. `TicksPerYear` is derived. Months and seasons are ordered lists, not tables keyed by ID. One season cycle for the world; regions differ by a table of yields by season | `spec-calendar.md` sections 3 to 5, `spec-config.md` section 3.4 |
| Where does unfinished multi-tick work live? | In the agent as one job (one per agent): inputs are consumed at the start, outputs created at completion, a duration-d job started in tick s completes in tick s+d-1. Nothing half-made is in the ledger | `spec-production.md` section 5 |
| How does a region or season change a recipe's output? | A yield-factor table by region, recipe and season, read at job start; 0 = cannot run; `Sited` recipes default to 0 where no profile exists. No second recipe per region | `spec-production.md` section 6 |
| How is a fractional yield resolved? | A deterministic carried remainder per agent and output, at every weight, started at one seeded draw per agent. No chance events in production | `spec-production.md` section 6.6 |
| Is there a `Tradable` flag? | No. A waste output can be marked `Discard` (created and disposed in one step); a `disposal` sink is needed in the ledger registry (note N-033) | `spec-production.md` section 7 |
| Can a trainee-eligible recipe need a tool? | No; and a trainee's output is `TraineeYield` (placeholder 5 000 bp) of a full execution | `spec-production.md` section 8 |
| What is a price belief? | A centre and a half-width in basis points, not a low and a high, so the range is derived and cannot invert | `spec-beliefs.md` sections 3.1, 3.3 |
| When do beliefs update? | Once per admitted order after its market clears, with probability `BeliefAdjustProb`, by at most `2 * BeliefStep` of the centre | `spec-beliefs.md` section 5 |
| What is a belief anchored to? | A reference band and, when a unit cost is known, a cost band; the update is driven by price age, unmatched quotes and the stock ratio | `spec-beliefs.md` sections 3.2, 5.4, 5.5 |
| How is the bazaarBot lookback window (defect 7) fixed? | A per-market window of traded prices, no per-agent trade history | `spec-beliefs.md` section 7 |
| Is the belief update rule a fifth extension interface? | No: architecture.md 4.3 already gives belief updates to `Strategy`; the world clamps whatever a strategy returns | `spec-beliefs.md` section 10 |
| Is the upkeep of a household consumed independently by commodity? | Yes, each good up to what is held; `Fed` is the least over goods of consumed over per-household need, `Unfed` is the rest | `spec-agents.md` sections 5.3, 5.4 |
| What is starvation in AGT? | A signal only (`Unfed`, `Hunger`); losses of weight and removal are POP and SPN | `spec-agents.md` section 5.5 |
| How many methods does `Strategy` have? | Three: `Work` (before production), `Trade` (after production), `Revise` (per order outcome) | `spec-agents.md` section 6.2 |
| Is there an inventory capacity in version 1? | No; a discard is a strategy decision executed through the `disposal` sink | `spec-agents.md` section 8 |
| What is the tradable set? | Derived: upkeep, recipe inputs, tools and non-discard outputs, plus `Role.Trades` | `spec-agents.md` section 3.4 |
| How does the household save when poor? | A three-tier budget: upkeep, job inputs, then buffers only above `ReserveTicks` of upkeep cost under a concave, bounded cap | `spec-agents.md` section 7.4 |
| What is a staple's demand elasticity? | Between 0 and 1, never above 1; not made elastic by invention. The stability condition for the labour loop therefore fails for a staple (see `spec-stability.md` 6.1) | `spec-agents.md` section 11.6 |
| How does a player act inside the synchronous `Step`? | Through a directive left in a mailbox between steps, read by the `Directed` strategy, or through a non-blocking consumer strategy; no immediate action | `spec-agents.md` section 10 |
| Does a closed economy get spending beyond upkeep? | No; the glut is accepted and bounded by the production halt. DEM, MON or EXT must supply it | `spec-agents.md` section 11.7 |
| What does the viability check consist of? | Four layers: sourced and runnable, productive cycles (Hawkins-Simon), labour sufficient (an exact linear programme), storable | `spec-viability.md` section 4 |
| Where does the steady-state solver live? | In the core as an exported pure function, not in `NewWorld` and not in the experiment harness; exact integer arithmetic | `spec-viability.md` sections 6, 7 |
| What is the warm start? | A `Config.WarmStart` section (nil = off): seed prices, starting money and stock, and staggered initial jobs | `spec-viability.md` section 8 |
| How is a feedback loop recorded? | As a register entry with sign, delay, damper, condition and test; a spec that adds a loop and no entry is not done | `spec-stability.md` sections 3, 9 |
| How is the welfare window chosen for a long job? | Derived per role: `T = max(WelfareWindow, WindowCycles * longest Duration)`; income attribution to a job was rejected | `spec-welfare-and-wages.md` section 2.3.3 |
| What is the floor on days worked? | `MinWorkShare`, a share of the window, default 4 000 bp (a placeholder); it replaces the price ceiling by a price set by the floor and does not remove the instability | `spec-welfare-and-wages.md` section 5.3 |
| What is netted from income? | Inputs bought (cash) and tool wear (accrual); not upkeep, not tool purchase | `spec-welfare-and-wages.md` section 2.1 |
| Does the default `TargetRatio` change in a closed economy? | No: it stays 15 000. In a world whose only demand is upkeep it cannot be met on average (the mean ratio is 1), and the glut is bounded by the production halt. Revisit when DEM, MON or EXT exist | `spec-welfare-and-wages.md` section 7.2, `spec-viability.md` section 4.7 |
| Does a queue count in the traineeship application rule? | Yes, as a delay discount | `spec-skill-barriers.md` section 4.5 |
| Does the cost anchor include labour? | Yes, at the pooled observed wage of the previous tick; an unknown wage makes the unit cost unknown | `spec-production.md` section 9 |
| What is the cost floor and ceiling? | 70% and 300% of the full cost (placeholders); a 100% floor freezes markets with perishables | `spec-beliefs.md` section 3.2.2 |
| How is price evidence compared? | To the order's limit, guarded, at a share of the step; the offset below the clearing price falls from about 11% to about 6% in a stylised model and is not removed | `spec-beliefs.md` section 5.4 |
| Who keeps the price window? | The market, pushed once per tick after clearing and before outcomes; `Clearing` never sees it | `spec-clearing.md` section 5.4 |
| What is the price before the first trade? | The seed price: the effective reference price, or the warm-start entry | `spec-clearing.md` section 5.4 |
| Where does the estate's ask anchor? | On the region's effective reference price (times remaining over full durability for a durable); a market-anchored rule is an experiment for later | `spec-estates.md` section 2.7.3, `spec-skill-barriers.md` section 2.7 |
| Is a strategy name checked by `Validate`? | No, by `NewWorld`: `Validate` takes no options | `spec-config.md` section 7.1.4 |
| Is `ExpectedWorkTicks` scaled when the year is short? | No, it is an error | `spec-config.md` section 13.5 |
| Is there a runtime stability controller? | No; stability comes from local rules, and the register records the loops and dampers | `spec-stability.md` sections 2.6, 5.1.3 |
| What anchors a staple's price? | The belief bounds, the production halt, buffer buying, recipe choice and exit; the halt alone does not anchor it | `spec-stability.md` section 6.2 run 1b |
| What is bankruptcy? | A leaky privation count read by the failure ladder, not money at 0 and not the one-tick `Hunger` | `spec-spawner.md` section 5.1 |
| In what order does the failure ladder act? | Rung 0 the survival rule, rung 1 role change, rung 2 relief (unconditional), rung 3 weight reduction by the unfed share and then removal, capped by `ExitShare` | `spec-spawner.md` section 5 |
| What damps occupation change? | The entry share (the damper that works in the model), with rotation of reviews, hysteresis and a cooldown | `spec-spawner.md` sections 6, 16.1 |
| Which role replaces a leaver? | Like for like if viable, else the best viable role, else none; not the most profitable | `spec-spawner.md` section 7.2 |
| Is `AgentBankrupted` the only removal event? | Yes, with `Cause` `insolvent`, `starved`, `consumer` or `population`; `WeightChanged` and `RoleChanged` are not removals | `spec-spawner.md` section 13.1.1 |
| Who owns births and deaths? | POP; the spawner has a hook (`PopDelta`) and does not write the demographic rates | `spec-spawner.md` section 8 |
| How do `Hunger` and `Privation` relate? | Both computed by the world from `Unfed`; `Hunger` short for the strategy, `Privation` leaky for the ladder; `Hunger` never exceeds `Privation` | `spec-agents.md` section 5.4.1 |
| What is a class in the demand module? | A role; its tastes are rows of `Tastes` | `spec-demand.md` section 3.3 |
| How does demand grow with income and fall with price? | Through the concave spend cap and a ladder of wants, and a linear schedule of the price ratio with a choke price and a ceiling, all integer | `spec-demand.md` sections 4, 5 |
| Can a closed economy exceed a mean welfare ratio of 1? | Yes with wants: `1 + Wref/U` expected, `1 + Wmax/U` at most; with upkeep only it is exactly 1 | `spec-demand.md` section 7 |
| How is a `Raw` good with no source handled? | It counts as sourced in the viability check, with the warning `raw_without_source` | `spec-viability.md` section 4.2.1 |
| What does the warm start do when the solver's wage is 0? | It builds seeds and the wage from the nominal programme and reports `NominalSeed` | `spec-viability.md` section 5.3.4 |
| Is the fixture code or data? | Data only: TOML packs layered by key; warm tables are solver output, never hand-written | `spec-reference-chain.md` section 2 |
| `math/big` | Allowed in `internal/steady` only (architecture.md 3.5) | `spec-viability.md` section 6 |
| What is the version 1 scope? | The staging of `vision.md` section 4; FIS and PLC later; INT only as shocks and a player agent | [ADR 0003](../adr/0003-version-1-scope.md) |
| Is a region one settlement or several? | One settlement per region, one market; a town and its hamlets are regions joined by short edges | `spec-regions.md` sections 3.1, 3.4 (ADR 0003 decision 4) |
| What is a distance? | A directed edge's integer length in a unit the pack chooses; not a duration; asymmetric and one-way edges allowed; zero and self edges rejected | `spec-regions.md` section 4.1 |
| Which path between two regions? | The shortest by total distance; ties by the smallest region index at each step (the lexicographically smallest shortest path) | `spec-regions.md` section 4.3 |
| Must the region graph be connected? | No; a closed region is valid and warnings say so | `spec-regions.md` section 4.5 |
| How does a region differ from the world's defaults? | One mechanism: a row keyed by region and subject in the subject owner's table, combined by replace, add or scale | `spec-regions.md` section 5 |
| Can an account change region? | Yes, an agent account, by one operation between ticks with no reservation open; it is not a transfer, has no reason, and is reported by relocation totals and an event. A hook lets a spec refuse it while an obligation tied to the region is open | `spec-regions.md` section 6 |
| Can the graph change in a run? | Only an edge status (closed, or a length factor) set between ticks | `spec-regions.md` section 7 |
| How does money enter and leave? | Only through named flows in data (a grant is a faucet on a class of households or a pool, a levy a sink), whose reason is the flow ID; a mine is not a recipe output and a mint is not a conversion (EXT's) | `spec-money-supply.md` sections 4, 8 |
| Is a faucet ever proportional to a balance? | No, only sinks are | `spec-money-supply.md` section 4.2 |
| What is a birth, a death? | The formation and dissolution of a household; no age structure | `spec-population.md` sections 3.2, 3.3 |
| Who is exposed to the population schedules? | Fed households only; unfed ones are the failure ladder's | `spec-population.md` sections 3.7, 4.5 |
| Is `Renewal` kept with population? | No, it is ignored while the module is on | `spec-population.md` section 4.6 |
| How are births funded? | From the pool, at a stake of `BirthStake` of `StartMoney`; a closed population is capped at the pool's money over the stake | `spec-population.md` sections 4.3, 5.5 |
| What is the Malthusian fixed point? | A replacement ratio where births equal deaths, stable when the gain times the delay is small; it needs a fixed factor that only RES can provide | `spec-population.md` section 5 |
| What is the carriage unit? | A produced good: the haul unit of a hired mode, made by a recipe and traded in an ordinary market; no change to clearing (N-076) | `spec-transport.md` section 8 |
| Can goods move without a traveller? | Yes: a hired shipment redeems haul units and the goods travel alone; a self-carried traveller carries everything its account holds | `spec-transport.md` sections 3.2, 5.1 |
| How long does a journey take? | `max(1, ceil(distance * 10 000 / (Speed * F)))` ticks, fixed at dispatch; there are no queues | `spec-transport.md` sections 4, 6 |

## Still open

1. Which recipes in the reference chain are trainee-eligible, and how large is their output? Production supplies the lever (`TraineeEligible`, `TraineeYield`, `TrainingPays`, `spec-production.md` section 8.3). The chain's actual flags and yields are calibration for `spec-reference-chain.md` and `spec-viability.md`; the reference chain must contain one eligible recipe above and one below break-even.
2. (Answered 2026-10-10.) Time in the traineeship queue counts in the application rule as a delay discount: `spec-skill-barriers.md` section 4.5.

3. (Answered 2026-10-10.) The shortest traineeship that shows the throttle is derived in `spec-skill-barriers.md` section 4.9.

4. Does relief (`spec-estates.md` section 2.6) suppress migration too much, since it raises
   the ratio that triggers it? Check in the experiments.
5. How does the price of hired carriage behave with a backhaul? The cost of a haul unit
   includes the whole round trip of the producing recipe, which double counts when cargo
   is found for the return (`spec-carriage.md` 5.1). Restated as open in
   `spec-transport.md` 20.2; it belongs to the contract spec and MER.
6. How does a merchant estimate `expected_margin_per_haul_unit` for the
   `MerchantFeeShare` ceiling? It uses its rumoured remote prices, but the
   exact rule belongs to the merchant strategy spec (MER; `spec-carriage.md` 5.2).
7. (Answered 2026-10-10.) The tolerance rule is `max(25% of the excess, 200 bp)`, raised to twice the observed spread: `spec-skill-barriers.md` section 3.

8. (Answered 2026-10-10.) The Justman and Beek (2015) paper was read; the two elasticities are reconciled in `evidence.md` (the 0.4 snippet is the rounded ordinary-least-squares wage value; the instrumented wage elasticity is about 1.0). The Statute of Artificers limit of three apprentices stays unverified.

