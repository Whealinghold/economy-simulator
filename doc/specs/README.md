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
| [spec-stability.md](spec-stability.md) | (core) The feedback-loop register: the entry format, 202 loops with sign, delay, damper, condition and test, the catalogue of 169 dampers and the rule for adding one, three numeric analyses (the target-income labour loop, the belief/price/inventory loop, the production and occupation cobwebs), the failure regimes, the scenarios the acceptance bands must cover, the checklist for spec authors, the loops no spec yet dampens, and the `FR-STB`/`NFR-STB` requirements |
| [spec-viability.md](spec-viability.md) | (core) Viability and the steady state: four layers of the check (sourced and runnable, productive cycles by the Hawkins-Simon condition, labour sufficient by an exact linear programme, storable), the programme and its dual as prices, wages and flows, exact integer arithmetic, why the solver is a pure core function and not part of NewWorld, the warm-start table (seed prices, money, stock, staggered jobs), regions and lanes, the rules added to config validation, what viability does not prove, and the FR-VIA/NFR-VIA requirements |
| [spec-reference-chain.md](spec-reference-chain.md) | (fixture) The standard test fixture: a base chain as TOML, seven product lines to the first consumer's size, variants for one, two, three and five regions, a starvation variant, a demand overlay, a second setting and six defect packs, layered by key; solver-generated warm starts; the scenarios S01 to S83 (the harness tests S77 and S78 are not built by REF) and A1 to A10 assigned to packs, or to the spec each waits for, with measures and the form of their pass; the invariants over seeds, the golden rendering, the configuration-only chain test; the `FR-REF` requirements |
| [spec-scale.md](spec-scale.md) | (core) Population scale: an agent's weight (the households it stands for), deriving agents from a census at a chosen scale with a floor, what other specs must do with a weight, the cost model and a provisional budget |
| [spec-config.md](spec-config.md) | (core) The configuration framework: its shape and how modules are switched on, bounds with their derivation, defaults defined once, the proposed TOML file format and pack layering, validation (framework, structural rules, rules from other specs), the configuration hash, and the parameter registry with the gaps found in it. Written ahead of the specs it depends on; its field catalogue grows |
| [spec-welfare-and-wages.md](spec-welfare-and-wages.md) | (module) The welfare ratio, wage formulas, observed wages and premiums, the target-income rule, in-kind pay |
| [spec-skill-barriers.md](spec-skill-barriers.md) | (module) Required tools, how the skill premium is checked, traineeship training |
| [spec-estates.md](spec-estates.md) | (core, module) What a dead agent leaves, the regional money pool, starting money and relief |
| [spec-regions.md](spec-regions.md) | (core) The region as one settlement and one market, its fields and who owns them, directed edges with an integer distance, deterministic queries with a total-order tie-break, components and islands, one override mechanism and the reference-price override, the operation that moves an account between regions and its hook for goods in transit, the edge status for shocks, entries for the register and the `FR-REG`/`NFR-REG` requirements |
| [spec-money-supply.md](spec-money-supply.md) | (core, module) Where money enters and leaves as named flows in data: grants and levies on a class of households or a pool, four non-reactive shapes, a window, a cumulative limit and a runtime scale seam; declared reasons; the money supply and its identities; the closed economy and its derived liquidity floor; the primitives reserved for the external sector; entries for the register; the `FR-MON`/`NFR-MON` requirements |
| [spec-carriage.md](spec-carriage.md) | (module) Carrier pay by merchants: the fee is the market price of haul units, an ordinary commodity (the seed of the delivery contract of `spec-merchants.md`) |
| [spec-transport.md](spec-transport.md) | (module) Transport modes as data (speed, a capacity pair per carrying unit, self-carried or hired), travel time from a REG distance, the shipment (request, 23 ordered refusal checks, dispatch, transit in a reused Transit account, arrival, diverted cargo), goods in transit in the ledger, audit and checkpoint, the decision on the carriage unit (N-076), what transport exposes to strategies and information, hooks for risk and infrastructure, entries for the register, a three-region worked example and the `FR-TRN`/`NFR-TRN` requirements |
| [spec-resources.md](spec-resources.md) | (module; the yield-table values are core data) Resources as data, not commodities: endowments per region with a stock, a capacity and three regeneration shapes (constant, refill, logistic) with a carried remainder and an optional refuge, an abundance curve, the seam to production (`Endowment` read and an internal all-or-nothing take), the proportional ration of a scarce stock, permanent and recoverable depletion, shock slots, the owner of the yield-table values, the fixed factor of the population loop with its stability condition, the sourcing of raw goods, open access and the hook for tenure, the sustainable draw for the solver, entries for the register, the models run and the `FR-RES`/`NFR-RES` requirements |
| [spec-information.md](spec-information.md) | (module) What an agent knows of other regions: only its own market report for free, and delayed, blurred rumours of remote prices for roles flagged `Informed`; the price record per source market (a ring of published prices), the rumour (region, commodity, price, mean, lag, source age), the lag from a REG distance and one speed with a floor of 2, an independent bounded error from the world's source, the cases with no rumour, the effect of a closed road, entries for the register, the models run and the `FR-INF`/`NFR-INF` requirements |
| [spec-external-sector.md](spec-external-sector.md) | (module) The outside world: an outside node is an agentless region marked in the configuration, with lines (a price path, up to two sides, a capacity, windows and caps); it trades through pass-through counterparties, pseudo-agents with an Outside account topped up each tick and swept to zero, so clearing is unchanged; six built-in ledger reasons, the identities and the balance of trade, quotes that are limits and not pegs, what an agent sees of the outside, the sourcing of raw goods by imports, the open layer of the steady state, entries for the register, the models run and the `FR-EXT`/`NFR-EXT` requirements |
| [spec-subsistence.md](spec-subsistence.md) | (module) Household production for own use as the floor under market failure: own-use recipes as ordinary source recipes named on a list (a module default and per-role rows), the bound that keeps the market preferred, the mode a household enters and leaves on its own stock, money, hunger and own-market price, the default `Work`, the ledger form (an ordinary recipe execution), the receipts in the welfare ratio and out of the observed wage, the failure ladder, the commons, seasons, the effect on markets with a model, entries for the register, the models run and the `FR-SUB`/`NFR-SUB` requirements |
| [spec-migration.md](spec-migration.md) | (module) Whether, when and where a household moves: a whole agent, a rule on the household's own earned ratio and measured wage and on rumoured prices only (never a regional aggregate), a staggered review of a settled household, an estimate of the income its role would earn at each destination from rumoured prices, the table's yield and the upkeep there, discounted by the journey, a margin with a width, a timed journey in the shape of a trip, caps on outflow and on the households on the way, provisioning from the household's own stock, no money or goods created, reseeded beliefs, seven dampers, the models run, entries for the register and the `FR-MIG`/`NFR-MIG` requirements |
| [spec-merchants.md](spec-merchants.md) | (module) The merchant: a second Strategy (registered as "merchant", wrapping the default) whose planned trip replaces the planned recipe; margin estimation from a cautiously read rumour and the local report, with SourceAge, spoilage and the cost of carriage in money or time; one fee-share rule for every mode and the exact expected_margin_per_haul_unit; the lot, the lane lock, the lot cap and a fixed spread of the fee share by agent ID; the consignee and the delivery contract (offers and takes matched in a round, escrow, settlement on arrival, failure and default; FR-CAR-001 and FR-CAR-002 restated); backhaul; the gate of the outside; entries for the register, models run, and the `FR-MER`/`NFR-MER` requirements |
| [spec-world.md](spec-world.md) | (core) The world: the eight-phase tick with every position, its owner, reads, writes, draws and events; the failed world and the in-step guard; options and what the hash covers (and the `Clearing` name); construction order and the creation draws; the keyed random streams (one seed, the key of a stream, the site constants, the world's draw, the table of every draw with its stream); between ticks (slots, calls, the checkpoint position); decisions at different rates (rotation by bucket, carried remainder, window, gate); fairness and keyed ties; the rulings on positions two specs claimed; entries for the register and the `FR-WLD`/`NFR-WLD` requirements |
| [spec-observability.md](spec-observability.md) | (core) What the world tells the outside: the event catalogue (owner, position, volume class, gate), the snapshot, the metric catalogue and the adapter that derives it, every `Warn` and `Info` path with its keys and bound, the cost of observing, the golden rendering, and the `FR-OBS`/`NFR-OBS` requirements |
| [spec-integration.md](spec-integration.md) | (core) What a game needs: external agents (`AddAgent`, `Direct` and the mailbox, `RemoveAgent`, `Relocate`), the table of nine slot setters and one adjustment with position and saved state, the queries and their cost classes, the sentinels and order of checks, the shape of a consumer's loop, and the `FR-INT`/`NFR-INT` requirements |
| [spec-checkpoint.md](spec-checkpoint.md) | (core) The checkpoint: an opaque immutable value of canonical bytes taken between ticks on a healthy world; the inventory of state by owning spec; one varint encoding with a SHA-256 trailer; restore as an all-or-nothing constructor with a fixed validation order; `WithReseed` for a branch; the optional `Stateful` extension state; bit-identity; compatibility before and after 1.0; cost; the `FR-CHK`/`NFR-CHK` requirements |
| [spec-indicators.md](spec-indicators.md) | (core) Derived measures of a run as pure functions of snapshots and events: generic aggregators and presets, index numbers in basis points (Laspeyres, Paasche, Fisher, chain), the Gini and quantiles over weighted households, the production, expenditure and income accounts and their identity, regions and the realm, trade, `indicator/` and its import rule, and the `FR-IND`/`NFR-IND` requirements |
| [spec-stylised-facts.md](spec-stylised-facts.md) | (core) The testable macro properties of a healthy run: the form of a fact (class, band, prior, confidence, source), the facts in seven groups, nine failure regimes as detectors, the three stability checks and what "stable" may mean, acceptance versus report and the seed rules, how a band is set without tuning, the four test tiers, what STY hands to EXP and CLB, and the `FR-STY`/`NFR-STY` requirements |
| [spec-experiments.md](spec-experiments.md) | (core) The sweep harness: an experiment as plain data over a base pack by field path through the loader's own layering, three designs (grid, one at a time, a counter-based Latin hypercube), consecutive-integer replica seeds, a sequential runner with a contract for a consumer's own parallel one, branching from one burn-in, aligned and paired runs with a measured gain, integer reductions, a comparison and a sensitivity, failure as a result, the cost, `FR-EXP`/`NFR-EXP` |
| [spec-calibration.md](spec-calibration.md) | (core) The registry of tuning knobs: one row for each tunable and solve option, classified as a knob, a damper strength, a structural choice or presentation, with its owner, its kind of range (a validation bound is never a search range; nearly every search range is unknown), its grid and neighbours, its expected effects with sign and source, its loops and the pairs that must move together; the procedure that calibrates a setting against the stylised facts (search blocks, screen, explore, choose, confirm; stages C0 to C9; stop rules), the record, the packs, sensitivity and identifiability, and the `FR-CLB`/`NFR-CLB` requirements |
| [spec-storage.md](spec-storage.md) | (module, later stage; draft) Warehousing: what a store is, its capacity pair, storage cost as a named money flow, spoilage and wear in storage, hooks for risk and settlements |
| [spec-employment.md](spec-employment.md) | (module, later stage; draft) Hired labour: the hire as a contract matched in a round, wages, the target-income rule for earners and employers, unemployment as a measure |
| [spec-settlements.md](spec-settlements.md) | (module, later stage) Settlement size classes derived from households with a hysteresis band, and the flags each class sets for other modules |
| [spec-land.md](spec-land.md) | (module, later stage; draft) Tenure over the endowments of resources: titles as shares of an endowment's stock, the entitlement before the ration, rent as a transfer valued at the dual and tenancy by let units, with the open access of RES as the default |
| [spec-polity.md](spec-polity.md) | (module, later stage; draft) Jurisdictions over regions as one tree with membership, the chain, covers, crossing and between as deterministic queries, the treasury account and pseudo-agent of a governed jurisdiction, and one combining rule for the rows later specs key by jurisdiction; no law and no inheritance of a region field |
| [spec-credit.md](spec-credit.md) | (module, later stage; draft) Credit: a claim formed by a loan at a posted rate matched in a first-fit round, interest as a transfer and never a faucet, settlement at the due tick with a shield, arrears and the waterfall at a removal, the default borrower and lender, and no collateral in version 1 |
| [spec-infrastructure.md](spec-infrastructure.md) | (module, later stage; draft) Facilities as fixed capital with a condition, built and repaired by ordinary recipes, and four effects: a site that a recipe needs, a route that speeds an edge, a ground improvement that raises an endowment's capacity, and a store |
| [spec-risk.md](spec-risk.md) | (module, later stage; draft) Loss of cargo in transit and of stored goods as rows of data, strikes drawn from the world's source, the `loss` sink, the embargo as an edge status, and the seams for insurance |
| [spec-fiscal.md](spec-fiscal.md) | (module, later stage; draft) Taxes, duties, tolls, the treasury's payouts and its standing orders: six typed lists over a governed jurisdiction's treasury, internal reasons only, a pro rata cut, a flow scale as the one lever |
| [spec-institutions.md](spec-institutions.md) | (module, later stage; draft) Bodies that select households with places, exclusive rights over an activity in a scope, and legal prices as a clamp in admission; no law table, no account and no enforcement agent |
| [spec-saving-and-investment.md](spec-saving-and-investment.md) | (module, later stage; draft) A savings target above the reserve, a precaution buffer taken from the household's own cash cycle, a project test for a build or a repair, and the hoard as a measure |
| [spec-technology.md](spec-technology.md) | (module, later stage; draft) A gain in yield held per region and field, raised by doing, by a trend, by discovery and by contact, and a bound on what it can do to the population loop |
| [spec-policy.md](spec-policy.md) | (module, later stage; draft) Policy actors: an interface of one method (`Decide`) and not `Strategy`; a policy actor is a row of configuration attached to a jurisdiction with a cadence and a list of levers (a named, bounded, rate-limited handle on a flow scale, an edge status or a transit factor) that is both its mandate and the only thing it can move; what it reads (the boundary state of the regions it covers and its treasury), when it acts (a position at the entry of the tick, a rotation), four built-ins (hold, a fixed schedule, a reactive rule on four gauges, a player's mailbox), the call a player uses, the loops and the models run |
| [spec-insurance.md](spec-insurance.md) | (module, later stage; draft) Single-event cover of a cargo and of stored goods against the losses of the risk module, priced from the expected loss, paid from an underwriter's money with the unpaid part a credit claim |
| [evidence.md](evidence.md) | The sources behind the numbers, and which facts are unverified |
| [bazaarbot-reference.md](bazaarbot-reference.md) | What the bazaarBot source does, its defects, and where each piece goes in the port (a reference, with no requirements and no area code) |

Reading order: currency, goods, ledger, production, clearing, beliefs, agents, spawner, population, resources, subsistence, demand, scale, config, welfare and wages, skill barriers, estates, regions, money supply, carriage, transport, information, migration, external sector, merchants, viability, stability, reference chain, world, observability, integration, checkpoint, indicators, stylised facts, experiments, calibration, then the later-stage modules (settlements, storage, employment, land, credit, risk, infrastructure, insurance, polity, fiscal, institutions, policy, saving and investment, technology). The later-stage order is the order in which each module needs only the ones before it, and it is the order of the modules appended to `Modules` (`spec-config.md` 3.1).
The layer in brackets (core, module, extension) is defined in [../spec-roadmap.md](../spec-roadmap.md) and
[ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md). A spec says which of its
parts are core and which are optional.
All the specs of the roadmap are written; the fourteen later-stage specs are drafts. The tier tables of [../spec-roadmap.md](../spec-roadmap.md) stay the home of the order of work, the area codes and what each spec needs.

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
| `WEL` | `spec-welfare-and-wages.md` | exists; Requirements section written (FR-WEL-001 to 041, NFR-WEL-001 to 003); all numbers are placeholders |
| `SKL` | `spec-skill-barriers.md` | exists; Requirements section written (FR-SKL-001 to 028, NFR-SKL-001 to 002); all numbers are placeholders |
| `EST` | `spec-estates.md` | exists; Requirements section written (FR-EST-001 to 024, NFR-EST-001); bounds and defaults are placeholders |
| `CAR` | `spec-carriage.md` | exists; Requirements section written (FR-CAR-001 to 013; 001 and 002 moved to `spec-merchants.md` 9.9, 005 to 007 superseded by `spec-transport.md`); the contract and the merchant are in `spec-merchants.md` |
| `CLR` | `spec-clearing.md` | exists; Requirements section written (FR-CLR-001 to 035, NFR-CLR-001 to 003) |
| `BEL` | `spec-beliefs.md` | exists; Requirements section written (FR-BEL-001 to 039, NFR-BEL-001 to 003); all defaults are placeholders |
| `AGT` | `spec-agents.md` | exists; Requirements section written (FR-AGT-001 to 085, NFR-AGT-001 to 004); all defaults are placeholders |
| `LED` | `spec-ledger.md` | exists; Requirements section written (FR-LED-001 to 065, NFR-LED-001 to 004) |
| `PRO` | `spec-production.md` | exists; Requirements section written (FR-PRO-001 to 051, NFR-PRO-001 to 004); all numbers are placeholders |
| `SPN` | `spec-spawner.md` | exists; Requirements section written (FR-SPN-001 to 060, NFR-SPN-001 to 003); all numbers are placeholders |
| `WLD` | `spec-world.md` | exists; Requirements section written (FR-WLD-001 to 110, NFR-WLD-001 to 005); core; adds only the field `WorldParams.Clearing` (owned by `spec-clearing.md` 8); all numbers are placeholders |
| `REG` | `spec-regions.md` | exists; Requirements section written (FR-REG-001 to 060, NFR-REG-001 to 004); core; the region, the directed graph, overrides, relocation; bounds are placeholders |
| `CFG` | `spec-config.md` | exists; Requirements section written (FR-CFG-001 to 097, NFR-CFG-001 to 003); the field catalogue is section 3.7 (3.7.1 for the fourteen later-stage modules) |
| `OBS` | `spec-observability.md` | exists; Requirements section written (FR-OBS-001 to 091, NFR-OBS-001 to 005); core; no `Config` field (`warnRun` is a rule constant); all numbers are placeholders |
| `GDS` | `spec-goods.md` | exists; Requirements section written (FR-GDS-001 to 023, NFR-GDS-001) |
| `REF` | `spec-reference-chain.md` | exists; Requirements section written (FR-REF-001 to 040, NFR-REF-001 to 002); fixture; bands unset, all numbers are placeholders |
| `RES` | `spec-resources.md` | exists; Requirements section written (FR-RES-001 to 050, NFR-RES-001 to 004); module; the yield-table values are core data; all numbers are placeholders |
| `CAL` | `spec-calendar.md` | exists; Requirements section written (FR-CAL-001 to 021, NFR-CAL-001 to 003); the structure of time, derived `TicksPerYear`, optional seasons |
| `POP` | `spec-population.md` | exists; Requirements section written (FR-POP-001 to 041, NFR-POP-001 to 004); module; all numbers are placeholders |
| `MIG` | `spec-migration.md` | exists; Requirements section written (FR-MIG-001 to 048, NFR-MIG-001 to 004); module; all numbers are placeholders |
| `INF` | `spec-information.md` | exists; Requirements section written (FR-INF-001 to 052, NFR-INF-001 to 004); module; all numbers are placeholders |
| `TRN` | `spec-transport.md` | exists; Requirements section written (FR-TRN-001 to 050, NFR-TRN-001 to 004); module; all numbers are placeholders |
| `STO` | `spec-storage.md` | exists (draft); Requirements section written (FR-STO-001 to 046, NFR-STO-001 to 004); module; reviewed (FAIL on R1 and R2, fixed by a second revision, not re-verified); all numbers are placeholders |
| `MER` | `spec-merchants.md` | exists; Requirements section written (FR-MER-001 to 064, NFR-MER-001 to 005; FR-CAR-001 and 002 restated); module; all numbers are placeholders |
| `INT` | `spec-integration.md` | exists; Requirements section written (FR-INT-001 to 070, NFR-INT-001 to 005); core, and minimal: the player and other external agents, the world slots and the queries; no `Config` field |
| `MON` | `spec-money-supply.md` | exists; Requirements section written (FR-MON-001 to 046, NFR-MON-001 to 003); core for the identities, module for the flows; all numbers are placeholders |
| `FIS` | `spec-fiscal.md` | exists (draft); Requirements section written (FR-FIS-001 to 013, 020 to 026, 030 to 056 and 060 to 063, in groups; NFR-FIS-001 to 005); module; reviewed and re-verified (PASS, minors R1, R2); all numbers are placeholders |
| `SCL` | `spec-scale.md` | exists; Requirements section written (FR-SCL-001 to 020, NFR-SCL-001 to 004); cost figures provisional |
| `DEM` | `spec-demand.md` | exists; Requirements section written (FR-DEM-001 to 043, NFR-DEM-001 to 004); module; all numbers are placeholders |
| `RSK` | `spec-risk.md` | exists (draft); Requirements section written (FR-RSK-001 to 040, NFR-RSK-001 to 004); module; reviewed and re-verified (PASS, minors R1, R2); all numbers are placeholders |
| `LND` | `spec-land.md` | exists (draft); Requirements section written (FR-LND-001 to 046, NFR-LND-001 to 004); module; reviewed and re-verified (PASS, minors R1 to R6); all numbers are placeholders |
| `EMP` | `spec-employment.md` | exists (draft); Requirements section written (FR-EMP-001 to 063, NFR-EMP-001 to 005); module; reviewed once, revisions pending; all numbers are placeholders |
| `CHK` | `spec-checkpoint.md` | exists; Requirements section written (FR-CHK-001 to 082, NFR-CHK-001 to 005); core; no `Config` field; sizes and costs are counts of placeholders |
| `IND` | `spec-indicators.md` | exists; Requirements section written (FR-IND-001 to 059, NFR-IND-001 to 004); core; the `indicator/` package; no `Config` field; all numbers are placeholders |
| `SET` | `spec-settlements.md` | exists; Requirements section written (FR-SET-001 to 033, NFR-SET-001 to 004); module; reviewed and re-verified (three minors left in the review file); all numbers are placeholders |
| `POL` | `spec-polity.md` | exists (draft); Requirements section written (FR-POL-001 to 035, NFR-POL-001 to 004); module; reviewed and re-verified (PASS, minors R1, R2); the tree and the membership are configuration; all numbers are placeholders |
| `ORG` | `spec-institutions.md` | exists (draft); Requirements section written (FR-ORG-001 to 055, NFR-ORG-001 to 004); module; reviewed and re-verified (PASS, minors R1 to R5); all numbers are placeholders |
| `CAP` | `spec-infrastructure.md` | exists (draft); Requirements section written (FR-CAP-001 to 048, NFR-CAP-001 to 005); module; reviewed (FAIL on R1 major, not re-verified); all numbers are placeholders |
| `CRD` | `spec-credit.md` | exists (draft); Requirements section written (FR-CRD-001 to 018, 020 to 027, 030 to 037, 040 to 045, 050 to 055, 060 to 062, 070 to 079 and 082 to 088, in groups; NFR-CRD-001 to 005); module; reviewed (FAIL on N1, fix owed, not re-verified); all numbers are placeholders |
| `INS` | `spec-insurance.md` | exists (draft); Requirements section written (FR-INS-001 to 008, 010 to 016, 020 to 025, 030 to 036, 040 to 046, 050 to 053, 060 to 064, 070 to 082 and 087, in groups; NFR-INS-001 to 005); module; reviewed (FAIL on R1, fixed by a third pass and accepted by grep, not re-verified by a reviewer); all numbers are placeholders |
| `EXT` | `spec-external-sector.md` | exists; Requirements section written (FR-EXT-001 to 050, NFR-EXT-001 to 004); module; all numbers are placeholders |
| `SAV` | `spec-saving-and-investment.md` | exists (draft); Requirements section written (FR-SAV-001 to 047, NFR-SAV-001 to 005); module; reviewed (FAIL on R1 major, not re-verified); all numbers are placeholders |
| `TEC` | `spec-technology.md` | exists (draft); Requirements section written (FR-TEC-001 to 037, NFR-TEC-001 to 005); module; reviewed and re-verified (PASS, minors N1 to N4); all numbers are placeholders |
| `PLC` | `spec-policy.md` | exists (draft); Requirements section written (FR-PLC-001 to 013, 020 to 025, 030 to 035, 040 to 047, 050 to 055, 060 to 064, 070 to 074, 080 to 086 and 090 to 092, in groups; NFR-PLC-001 to 005); module, later stage; the fifth extension interface `Policy`; adds the position P0.3a and the call `DirectPolicy`; reviewed and re-verified (PASS, minor R1); all numbers are placeholders |
| `STY` | `spec-stylised-facts.md` | exists; Requirements section written (FR-STY-001 to 098, NFR-STY-001 to 004); core; every band is a prior and no default is called stable |
| `CLB` | `spec-calibration.md` | exists; Requirements section written (FR-CLB-001 to 091 and 100 to 109, NFR-CLB-001 to 005); core; no `Config` field; all procedure constants are placeholders |
| `EXP` | `spec-experiments.md` | exists; Requirements section written (FR-EXP-001 to 072, NFR-EXP-001 to 006); core; no `Config` field; all bounds are placeholders |
| `STB` | `spec-stability.md` | exists; Requirements section written (FR-STB-001 to 031, NFR-STB-001 to 002); the register has 202 loops, 169 dampers and 165 scenarios (S01 to S165, S15 unused); all numbers are placeholders |
| `VIA` | `spec-viability.md` | exists; Requirements section written (FR-VIA-001 to 059, NFR-VIA-001 to 004); all numbers are placeholders |
| `SUB` | `spec-subsistence.md` | exists; Requirements section written (FR-SUB-001 to 051, NFR-SUB-001 to 003); module; all numbers are placeholders |

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

The class, the kind of range, the expected effects and the interactions of each row are in `spec-calibration.md` section 6, found by (scope token, parameter).

A Go field that only spells in Go what the file says by a key or by absence has no row (`spec-config.md` 9.2): the `Has` flag of a row (`TenureTerm.HasRent` and `HasClosed`, `YieldProfile.HasAll`, as `HasCapacity` of an endowment row) says that the row's value applies, which the file says by writing the key, and `HolderRef.Kind` and `Role` are the Go form of the `<holder>` key of a holding (a role ID or the word `pool`, `spec-land.md` 3.3.2), which is the row `Holder`.

| Parameter | Scope | Owner |
|-----------|-------|-------|
| `ReferenceBasket` (`B_r`) | Region | `spec-currency.md` section 5 |
| `Unskilled` | Role | `spec-welfare-and-wages.md` section 4.2 |
| `TargetRatio` | Role | `spec-welfare-and-wages.md` section 5 |
| `TargetSpread` | Role | `spec-welfare-and-wages.md` section 5.1 |
| `WithTargetSpread` option | World (option; hashed, `spec-config.md` 8.1) | `spec-welfare-and-wages.md` section 5.2 |
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
| `BulkFactor` | WorldParams | `spec-goods.md` section 8 |
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
| `Scale` | WorldParams (the default of a group that gives none; default 1) | `spec-scale.md` section 3.1 |
| `Scale` | Group (0: the world's `Scale`) | `spec-scale.md` section 3.1 |
| `MinAgents` (`K`) | WorldParams | `spec-scale.md` section 5.2 |
| `Duration` | Recipe | `spec-production.md` section 5 |
| `Sited` | Recipe | `spec-production.md` section 6.2 |
| `TraineeEligible` | Recipe | `spec-production.md` section 8 |
| `TraineeYield` | Recipe | `spec-production.md` section 8.2 |
| `WearPerUse` | Tool | `spec-production.md` section 4.4 |
| `CostShare` | Output | `spec-production.md` section 3.2 |
| `Discard` | Output | `spec-production.md` section 7.2 |
| `Draws` | Recipe | the field: `spec-production.md` section 3.1; the resources it names: `spec-resources.md` section 3 |
| `Yields` | Config (a section; the profiles of a region and a recipe) | values and authoring: `spec-resources.md` section 7; format and lookup: `spec-production.md` section 6.1 |
| `BeliefStep` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefAdjustProb` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefSignificance` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefStockLow` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefStockHigh` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefMaxAge` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefWidthInit` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefWidthMin` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefWidthMax` | WorldParams | `spec-beliefs.md` section 11 |
| `BeliefWidthStep` | WorldParams | `spec-beliefs.md` section 11 |
| `CostFloor` | WorldParams | `spec-beliefs.md` section 11 |
| `CostCeiling` | WorldParams | `spec-beliefs.md` section 11 |
| `RefBandLow` | WorldParams | `spec-beliefs.md` section 11 |
| `RefBandHigh` | WorldParams | `spec-beliefs.md` section 11 |
| `PriceWindow` | WorldParams | `spec-beliefs.md` section 11 |
| `Upkeep` | WorldParams | `spec-agents.md` section 5.2, 12 |
| `Upkeep` | Role (replacing the world's) | `spec-agents.md` section 5.2, 12 |
| `ExtraUpkeep` | Region | `spec-agents.md` section 5.2, 12 |
| `CoverTicks` | WorldParams | `spec-agents.md` section 12 |
| `InputJobs` | WorldParams | `spec-agents.md` section 12 |
| `OutputCoverTicks` | WorldParams | `spec-agents.md` section 12 |
| `FavourFloor` | WorldParams | `spec-agents.md` section 12 |
| `ReserveTicks` | WorldParams | `spec-agents.md` section 12 |
| `SpendMaxShare` | WorldParams | `spec-agents.md` section 12 |
| `SpendHalfTicks` | WorldParams | `spec-agents.md` section 12 |
| `RestBand` | WorldParams | `spec-agents.md` section 12 |
| `ProduceHaltRatio` | WorldParams | `spec-agents.md` section 12 |
| `BasketTolerance` | WorldParams | `spec-agents.md` section 12 (closes the unnamed tolerance of `spec-currency.md` 5.3) |
| `Recipes` | Role | `spec-agents.md` section 3.3 |
| `Trades` | Role | `spec-agents.md` section 3.3 |
| `StartMoney` | Role | `spec-agents.md` section 3.3 |
| `StartStock` | Role | `spec-agents.md` section 3.3 |
| `Strategy` | Role (a registered name) | `spec-agents.md` section 6.6 |
| `WarmStart` | Config (a section; nil = off) | `spec-viability.md` section 8.1 |
| `WelfareWindow` | WorldParams | `spec-welfare-and-wages.md` section 2.3.1 |
| `WindowCycles` | WorldParams | `spec-welfare-and-wages.md` section 2.3.3 |
| `MinWorkShare` | Role | `spec-welfare-and-wages.md` section 5.3 |
| `BeliefEvidenceShare` | WorldParams | `spec-beliefs.md` section 11 |
| `ID` | Commodity | `spec-goods.md` section 8 |
| `ID` | Recipe | `spec-production.md` section 3.1, 11 |
| `Inputs` | Recipe | `spec-production.md` section 3.1, 11 |
| `Outputs` | Recipe | `spec-production.md` section 3.1, 11 |
| `Requires` | Recipe | `spec-production.md` section 3.1, 11 |
| `Region` | YieldProfile | `spec-resources.md` section 7 (format: `spec-production.md` section 6.1) |
| `Recipe` | YieldProfile | `spec-resources.md` section 7 (format: `spec-production.md` section 6.1) |
| `All` | YieldProfile | `spec-resources.md` section 7 (format: `spec-production.md` section 6.1) |
| `ID` | Role | `spec-agents.md` section 3.3 |
| `ID` | Region | `spec-currency.md` section 5 |
| `Role` | Group | `spec-scale.md` section 3.1 |
| `Prices` | WarmStart | `spec-viability.md` section 8.1 |
| `Wages` | WarmStart | `spec-viability.md` section 8.1 |
| `Groups` | WarmStart | `spec-viability.md` section 8.1 |
| `Pools` | WarmStart | `spec-viability.md` section 8.1 |
| `ID` | Month | `spec-calendar.md` section 3.1 |
| `Days` | Month | `spec-calendar.md` section 3.1 |
| `ID` | Season | `spec-calendar.md` section 3.1 |
| `Days` | Season | `spec-calendar.md` section 3.1 |
| `Season` | SeasonYield | `spec-production.md` section 6.1 (values: `spec-resources.md` section 7) |
| `BP` | SeasonYield | `spec-production.md` section 6.1 (values: `spec-resources.md` section 7) |
| `Commodity` | Item | `spec-production.md` section 3.1 (the type of every list of (commodity, quantity): an upkeep, a start stock, a recipe's inputs, a warm group's stock) |
| `Quantity` | Item | `spec-production.md` section 3.1 |
| `Commodity` | Output | `spec-production.md` section 3.2 |
| `Quantity` | Output | `spec-production.md` section 3.2 |
| `Commodity` | Tool | `spec-production.md` section 4.4 |
| `Region` | SeedPrice | `spec-viability.md` section 8.1 |
| `Commodity` | SeedPrice | `spec-viability.md` section 8.1 |
| `Price` | SeedPrice | `spec-viability.md` section 8.1 |
| `Region` | SeedWage | `spec-viability.md` section 8.2.5 |
| `Wage` | SeedWage | `spec-viability.md` section 8.2.5 |
| `Region` | WarmGroup | `spec-viability.md` section 8.1 |
| `Role` | WarmGroup | `spec-viability.md` section 8.1 |
| `Money` | WarmGroup | `spec-viability.md` section 8.3 |
| `Stock` | WarmGroup | `spec-viability.md` section 8.3 |
| `Jobs` | WarmGroup | `spec-viability.md` section 8.5 |
| `Recipe` | JobSpec | `spec-viability.md` section 8.5 |
| `InProgressBP` | JobSpec | `spec-viability.md` section 8.5.1 (the file key is `in_progress`, `spec-config.md` 6.2.2) |
| `Region` | WarmPool | `spec-viability.md` section 8.3.3 |
| `Money` | WarmPool | `spec-viability.md` section 8.3.3 |
| `Region` | Group | `spec-scale.md` section 3.1 |
| `Seasons` | YieldProfile | `spec-resources.md` section 7 (format: `spec-production.md` section 6.1) |
| `ReviewTicks` | WorldParams | `spec-spawner.md` section 11 |
| `EntryShare` | WorldParams | `spec-spawner.md` section 11 |
| `RoleHysteresis` | WorldParams | `spec-spawner.md` section 11 |
| `RoleCooldown` | WorldParams | `spec-spawner.md` section 11 |
| `EntryRatio` | WorldParams | `spec-spawner.md` section 11 |
| `ChangePrivation` | WorldParams | `spec-spawner.md` section 11 |
| `RemovePrivation` | WorldParams | `spec-spawner.md` section 11 |
| `DeathStep` | WorldParams | `spec-spawner.md` section 11 |
| `ExitShare` | WorldParams | `spec-spawner.md` section 11 |
| `Renewal` | WorldParams | `spec-spawner.md` section 11 |
| `EntryPoolShare` | WorldParams | `spec-spawner.md` section 11 |
| `Spawner` | WorldParams | `spec-spawner.md` section 3.1.3 |
| `Clearing` | WorldParams | `spec-clearing.md` section 8 |
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
| `Commodity` | WantItem | `spec-demand.md` section 9 |
| `Weight` | WantItem | `spec-demand.md` section 9 |
| `Want` | Taste | `spec-demand.md` section 9 |
| `Role` | Taste | `spec-demand.md` section 9 |
| `Region` | RegionTaste | `spec-demand.md` section 9 |
| `Want` | RegionTaste | `spec-demand.md` section 9 |
| `Scale` | RegionTaste | `spec-demand.md` section 9 |
| `Edges` | Config (a section; optional) | `spec-regions.md` section 4.1 |
| `From` | Edge | `spec-regions.md` section 4.1 |
| `To` | Edge | `spec-regions.md` section 4.1 |
| `Distance` | Edge | `spec-regions.md` section 4.1 |
| `RegionPrices` | Config (a section; optional) | `spec-regions.md` section 5.2 |
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
| `Kind` | FlowTarget | `spec-money-supply.md` section 4.1 (kinds `stores` and `treasury`: sections 4.3.4 and 4.3.5) |
| `Region` | FlowTarget | `spec-money-supply.md` section 4.1 |
| `Role` | FlowTarget | `spec-money-supply.md` section 4.1 (`spec-fiscal.md` section 3.2 reads an empty `Role` of a tax or a payout as every role) |
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
| `Resources` | Modules (module; nil = off) | `spec-resources.md` section 3.3 |
| `Resources` | Resources (the list of resource definitions) | `spec-resources.md` section 3.3 |
| `Endowments` | Resources (the region-keyed table) | `spec-resources.md` section 3.3 |
| `HorizonYears` | Resources | `spec-resources.md` section 12 |
| `ID` | Resource | `spec-resources.md` section 12 |
| `Shape` | Resource | `spec-resources.md` section 12 |
| `RatePerYear` | Resource | `spec-resources.md` section 12 |
| `RateBP` | Resource | `spec-resources.md` section 12 |
| `Refuge` | Resource | `spec-resources.md` section 12 |
| `Capacity` | Resource | `spec-resources.md` section 12 |
| `InitialBP` | Resource | `spec-resources.md` section 12 |
| `PlanShareBP` | Resource | `spec-resources.md` section 12 |
| `Abundance` | Resource | `spec-resources.md` section 4.7, 12 |
| `Share` | AbundanceKnot | `spec-resources.md` section 4.7, 12 |
| `Factor` | AbundanceKnot | `spec-resources.md` section 4.7, 12 |
| `Seasons` | Resource (regeneration by season) | `spec-resources.md` section 4.5, 12 |
| `Season` | SeasonRegen | `spec-resources.md` section 4.5 |
| `BP` | SeasonRegen | `spec-resources.md` section 4.5 |
| `Region` | Endowment | `spec-resources.md` section 3.3 |
| `Resource` | Endowment | `spec-resources.md` section 3.3 |
| `Capacity` | Endowment (replace) | `spec-resources.md` section 3.3.1, 12 |
| `Initial` | Endowment (replace) | `spec-resources.md` section 3.3.1, 12 |
| `RegenScaleBP` | Endowment (scale) | `spec-resources.md` section 3.3.1, 12 |
| `Information` | Modules (module; nil = off) | `spec-information.md` section 8 |
| `InfoSpeed` | Information | `spec-information.md` section 8 |
| `RumourErrorBP` | Information | `spec-information.md` section 8 |
| `RumourMaxAge` | Information | `spec-information.md` section 8 |
| `Informed` | Role | `spec-information.md` section 6.1 |
| `External` | Modules (module; nil = off) | `spec-external-sector.md` section 3.1 |
| `Nodes` | External | `spec-external-sector.md` section 3.1 |
| `Region` | OutsideNode | `spec-external-sector.md` section 3.1 |
| `Lines` | OutsideNode | `spec-external-sector.md` section 3.3 |
| `Gates` | OutsideNode | `spec-external-sector.md` section 3.2.4 |
| `Region` | OutsideGate | `spec-external-sector.md` section 3.2.4 |
| `ScaleBP` | OutsideGate | `spec-external-sector.md` sections 3.2.4, 6.6 |
| `Commodity` | OutsideLine | `spec-external-sector.md` section 3.3 |
| `Base` | OutsideLine | `spec-external-sector.md` section 6.1 |
| `PriceBySeason` | OutsideLine | `spec-external-sector.md` section 6.2 |
| `TrendBP` | OutsideLine | `spec-external-sector.md` section 6.3 |
| `FloorBP` | OutsideLine | `spec-external-sector.md` section 6.3 |
| `CeilingBP` | OutsideLine | `spec-external-sector.md` section 6.3 |
| `WalkStepBP` | OutsideLine | `spec-external-sector.md` section 6.4 |
| `WalkRevertBP` | OutsideLine | `spec-external-sector.md` section 6.4 |
| `WalkBoundBP` | OutsideLine | `spec-external-sector.md` section 6.4 |
| `Sell` | OutsideLine | `spec-external-sector.md` section 3.3 |
| `Buy` | OutsideLine | `spec-external-sector.md` section 3.3 |
| `MarginBP` | OutsideSide | `spec-external-sector.md` section 6.6 |
| `Capacity` | OutsideSide | `spec-external-sector.md` section 7.1 |
| `Window` | OutsideSide | `spec-external-sector.md` section 7.2 |
| `TotalUnits` | OutsideSide | `spec-external-sector.md` section 7.3 |
| `TotalValue` | OutsideSide | `spec-external-sector.md` sections 7.3, 7.4 |
| `Season` | SeasonFactor | `spec-external-sector.md` sections 6.2, 7.2 |
| `BP` | SeasonFactor | `spec-external-sector.md` sections 6.2, 7.2 |
| `Subsistence` | Modules (module; nil = off) | `spec-subsistence.md` section 3.2 |
| `Recipes` | Subsistence (the default own-use recipes) | `spec-subsistence.md` sections 3.2, 12 |
| `Roles` | Subsistence (the role-keyed rows, replace) | `spec-subsistence.md` sections 3.2.1, 12 |
| `Role` | RoleOwnUse | `spec-subsistence.md` sections 3.2, 12 |
| `Recipes` | RoleOwnUse | `spec-subsistence.md` sections 3.2, 12 |
| `EnterTicks` | Subsistence | `spec-subsistence.md` sections 5.3, 12 |
| `ExitTicks` | Subsistence | `spec-subsistence.md` sections 5.3, 12 |
| `RetryJobs` | Subsistence | `spec-subsistence.md` sections 5.5, 12 |
| `Migration` | Modules (module; nil = off) | `spec-migration.md` sections 3.1, 11 |
| `MoveSpeed` | Migration | `spec-migration.md` sections 6.4, 11 |
| `MoveReviewTicks` | Migration | `spec-migration.md` sections 5.1, 11 |
| `MoveBelow` | Migration | `spec-migration.md` sections 5.3, 11 |
| `MoveGain` | Migration | `spec-migration.md` sections 6.5, 11 |
| `MoveWidth` | Migration | `spec-migration.md` sections 6.5, 11 |
| `MoveCooldown` | Migration | `spec-migration.md` sections 5.2, 11 |
| `MoveHorizon` | Migration | `spec-migration.md` sections 6.4, 11 |
| `MoveOutShare` | Migration | `spec-migration.md` sections 7.3, 11 |
| `MoveInShare` | Migration | `spec-migration.md` sections 7.5, 11 |
| `Regions` | Migration (the region-keyed rows) | `spec-migration.md` section 11.2 |
| `Region` | RegionMove | `spec-migration.md` section 11.2 |
| `OutShare` | RegionMove (replace) | `spec-migration.md` section 11.2 |
| `InShare` | RegionMove (replace) | `spec-migration.md` section 11.2 |
| `Fixed` | Role | `spec-migration.md` sections 3.1, 11 |
| `Merchants` | Modules (module; nil = off) | `spec-merchants.md` section 13 |
| `MaxOpen` | Merchants | `spec-merchants.md` section 13 |
| `Merchant` | Role (the merchant block; nil = not a merchant role) | `spec-merchants.md` section 3.2 |
| `Modes` | Merchant | `spec-merchants.md` section 3.2 |
| `LotUnits` | Merchant | `spec-merchants.md` section 3.2 |
| `FeeShareSpreadBP` | Merchant | `spec-merchants.md` section 3.2 |
| `MaxSourceAge` | Merchant | `spec-merchants.md` section 3.2 |
| `TimeCostBP` | Merchant | `spec-merchants.md` section 3.2 |
| `ConsigneeCutBP` | Merchant | `spec-merchants.md` section 3.2 |
| `Consigns` | Merchant | `spec-merchants.md` section 3.2 |
| `TakeMarginBP` | Merchant | `spec-merchants.md` section 3.2 |
| `TakeUnits` | Merchant | `spec-merchants.md` section 3.2 |
| `Storage` | Modules (module; nil = off) | `spec-storage.md` section 12.1 |
| `Base` | Storage | `spec-storage.md` section 12.1 |
| `Roles` | Storage | `spec-storage.md` section 12.1 |
| `Classes` | Storage | `spec-storage.md` section 12.1 |
| `Role` | RoleRoom | `spec-storage.md` section 12.1 |
| `Base` | RoleRoom | `spec-storage.md` section 12.1 |
| `Stores` | RoleRoom | `spec-storage.md` section 12.1 |
| `ID` | StoreClass | `spec-storage.md` sections 3.2, 12.1 |
| `Mass` | StoreClass | `spec-storage.md` sections 3.2, 12.1 |
| `Bulk` | StoreClass | `spec-storage.md` sections 3.2, 12.1 |
| `Needs` | StoreClass | `spec-storage.md` sections 3.2.3, 12.1 (the flag is `spec-settlements.md`'s) |
| `Mass` | Capacity | `spec-storage.md` section 12.1 (the type is `spec-goods.md` 3.2) |
| `Bulk` | Capacity | `spec-storage.md` section 12.1 (the type is `spec-goods.md` 3.2) |
| `Store` | FlowTarget | `spec-storage.md` sections 7.1.2, 12.3 (the type is `spec-money-supply.md` 4.1) |
| `Employment` | Modules (module; nil = off) | `spec-employment.md` sections 3.1, 13 |
| `MaxOpen` | Employment (per region) | `spec-employment.md` section 13 |
| `Hireable` | Role | `spec-employment.md` sections 3.1, 13 |
| `Employer` | Role (the employer block; nil = not an employer) | `spec-employment.md` sections 3.2, 13 |
| `Recipes` | Employer (the hire list) | `spec-employment.md` sections 3.2, 13 |
| `MaxHires` | Employer | `spec-employment.md` sections 3.2, 13 |
| `WorkerShareBP` | Employer | `spec-employment.md` sections 3.2, 5.4, 13 |
| `HireYieldBP` | Employer | `spec-employment.md` sections 3.2, 6.1, 13 |
| `Settlements` | Modules (module; nil = off) | `spec-settlements.md` sections 3.1, 8 |
| `Classes` | Settlements (the size classes, sorted by `Enter`) | `spec-settlements.md` sections 3.1, 8 |
| `BandBP` | Settlements | `spec-settlements.md` sections 3.4.2, 8 |
| `Dwell` | Settlements | `spec-settlements.md` sections 3.4.3, 8 |
| `Regions` | Settlements (the region-keyed rows) | `spec-settlements.md` sections 3.5, 8 |
| `ID` | SettlementClass | `spec-settlements.md` sections 3.1, 8 |
| `Enter` | SettlementClass | `spec-settlements.md` sections 3.4.1, 8 |
| `Flags` | SettlementClass | `spec-settlements.md` sections 3.3.1, 8 |
| `Limits` | SettlementClass | `spec-settlements.md` sections 3.3.2, 8 |
| `Name` | ClassLimit | `spec-settlements.md` sections 3.3.2, 8 |
| `Value` | ClassLimit | `spec-settlements.md` sections 3.3.2, 8 |
| `Region` | RegionSettlement | `spec-settlements.md` sections 3.5, 8 |
| `ThresholdBP` | RegionSettlement (scale) | `spec-settlements.md` sections 3.5.2, 8 |
| `Floor` | RegionSettlement (replace) | `spec-settlements.md` sections 3.5.3, 8 |
| `Needs` | Role (a settlement flag) | `spec-settlements.md` sections 6.4.1, 8 |
| `Ceiling` | Role | `spec-settlements.md` sections 6.4.2, 8 |
| `Tenure` | Modules (module; nil = off) | `spec-land.md` sections 3.3, 10 |
| `RentStepBP` | Tenure | `spec-land.md` sections 5.5, 10 |
| `RentFloorBP` | Tenure | `spec-land.md` sections 5.5, 10 |
| `RentCeilingBP` | Tenure | `spec-land.md` sections 5.5, 10 |
| `OwnUseExempt` | Tenure | `spec-land.md` sections 5.3.4, 10 |
| `MaxHolderShareBP` | Tenure | `spec-land.md` sections 6.1.2, 10 |
| `Terms` | Tenure (the region-keyed table) | `spec-land.md` sections 3.3, 10 |
| `Holdings` | Tenure | `spec-land.md` sections 3.3, 10 |
| `Region` | TenureTerm | `spec-land.md` section 10 |
| `Resource` | TenureTerm | `spec-land.md` section 10 |
| `Rent` | TenureTerm | `spec-land.md` sections 5.2, 10 |
| `Closed` | TenureTerm | `spec-land.md` sections 4.3, 10 |
| `Region` | Holding | `spec-land.md` section 10 |
| `Resource` | Holding | `spec-land.md` section 10 |
| `Holder` | Holding | `spec-land.md` section 10 |
| `ShareBP` | Holding | `spec-land.md` section 10 |
| `Polity` | Modules (module; nil = off) | `spec-polity.md` section 8 |
| `Jurisdictions` | Polity | `spec-polity.md` sections 3, 8 |
| `Members` | Polity (the region-keyed rows, replace) | `spec-polity.md` sections 3.4, 8 |
| `ID` | Jurisdiction | `spec-polity.md` sections 3.2, 8 |
| `Parent` | Jurisdiction | `spec-polity.md` sections 3.3, 8 |
| `Seat` | Jurisdiction | `spec-polity.md` sections 3.5, 8 |
| `StartMoney` | Jurisdiction | `spec-polity.md` sections 3.5.4, 8 |
| `Region` | Member | `spec-polity.md` sections 3.4, 8 |
| `Jurisdiction` | Member | `spec-polity.md` sections 3.4, 8 |
| `Credit` | Modules (module; nil = off) | `spec-credit.md` sections 4.1, 14 |
| `MinPrincipal` | Credit | `spec-credit.md` section 14 |
| `ShieldTicks` | Credit | `spec-credit.md` sections 8.2, 14 |
| `WriteOffTicks` | Credit | `spec-credit.md` sections 8.5, 14 |
| `MaxClaims` | Credit | `spec-credit.md` sections 6.4, 14 |
| `Credit` | Role (the credit block; nil = not a credit role) | `spec-credit.md` sections 4.2, 14 |
| `Lend` | CreditTerms | `spec-credit.md` section 4.2 |
| `Borrow` | CreditTerms | `spec-credit.md` section 4.2 |
| `RateBP` | Lend | `spec-credit.md` sections 7.1, 14 |
| `MaxTermTicks` | Lend | `spec-credit.md` sections 10.3, 14 |
| `KeepTicks` | Lend | `spec-credit.md` sections 10.3, 14 |
| `ShareBP` | Lend | `spec-credit.md` sections 10.3, 14 |
| `MaxRateBP` | Borrow | `spec-credit.md` sections 10.2, 14 |
| `MaxDebtTicks` | Borrow | `spec-credit.md` sections 5.3.3, 14 |
| `SlackTicks` | Borrow | `spec-credit.md` sections 10.2, 14 |
| `Infrastructure` | Modules (module; nil = off) | `spec-infrastructure.md` section 12.1 |
| `Classes` | Infrastructure | `spec-infrastructure.md` section 12.1 |
| `Initial` | Infrastructure | `spec-infrastructure.md` sections 12.1, 3.4.2 |
| `ID` | FacilityClass | `spec-infrastructure.md` section 3.1 |
| `Kind` | FacilityClass | `spec-infrastructure.md` sections 3.1, 12.2 |
| `Durability` | FacilityClass (of the class) | `spec-infrastructure.md` section 12.2 |
| `Decay` | FacilityClass | `spec-infrastructure.md` section 12.2 |
| `WearPerUse` | FacilityClass | `spec-infrastructure.md` section 12.2 |
| `Capacity` | FacilityClass (of the class) | `spec-infrastructure.md` section 12.2 |
| `BoostBP` | FacilityClass | `spec-infrastructure.md` section 12.2 |
| `Modes` | FacilityClass | `spec-infrastructure.md` section 12.2 |
| `Resource` | FacilityClass | `spec-infrastructure.md` section 12.2 |
| `Store` | FacilityClass | `spec-infrastructure.md` section 12.2 |
| `Needs` | FacilityClass (a settlement flag) | `spec-infrastructure.md` sections 8, 12.2 |
| `Region` | InitialFacility | `spec-infrastructure.md` section 3.4.2 |
| `Class` | InitialFacility | `spec-infrastructure.md` section 3.4.2 |
| `Toward` | InitialFacility | `spec-infrastructure.md` section 3.4.2 |
| `Role` | InitialFacility | `spec-infrastructure.md` section 3.4.2 |
| `Count` | InitialFacility | `spec-infrastructure.md` section 3.4.2 |
| `Needs` | Recipe (a list of facility classes) | `spec-infrastructure.md` sections 4.1.1, 4.1.2 (the format is `spec-production.md` 3.1) |
| `Builds` | Recipe | `spec-infrastructure.md` sections 4.1.1, 4.1.2 (the format is `spec-production.md` 3.1) |
| `Repairs` | Recipe | `spec-infrastructure.md` sections 4.1.1, 4.1.2 (the format is `spec-production.md` 3.1) |
| `Class` | FacilityBuild | `spec-infrastructure.md` section 4.1.1 |
| `Toward` | FacilityBuild | `spec-infrastructure.md` section 4.1.1 |
| `Class` | FacilityRepair | `spec-infrastructure.md` section 4.1.1 |
| `Units` | FacilityRepair | `spec-infrastructure.md` section 4.1.1 |
| `Risk` | Modules (module; nil = off) | `spec-risk.md` section 12.1 |
| `Routes` | Risk | `spec-risk.md` sections 3.2.1, 12.1 |
| `Stores` | Risk | `spec-risk.md` sections 3.2.2, 12.1 |
| `From` | RouteRisk | `spec-risk.md` section 3.2.1 |
| `To` | RouteRisk | `spec-risk.md` section 3.2.1 |
| `Mode` | RouteRisk | `spec-risk.md` section 3.2.1 |
| `Region` | StoreRisk | `spec-risk.md` section 3.2.2 |
| `Class` | StoreRisk | `spec-risk.md` section 3.2.2 |
| `StrikeBP` | RouteRisk | `spec-risk.md` sections 3.2, 3.4 |
| `StrikeBP` | StoreRisk | `spec-risk.md` sections 3.2, 3.4 |
| `SeverityBP` | RouteRisk | `spec-risk.md` sections 3.2, 3.4 |
| `SeverityBP` | StoreRisk | `spec-risk.md` sections 3.2, 3.4 |
| `Common` | RouteRisk | `spec-risk.md` sections 3.2, 3.3 |
| `Common` | StoreRisk | `spec-risk.md` sections 3.2, 3.3 |
| `Fiscal` | Modules (module; nil = off) | `spec-fiscal.md` sections 3.1, 14 |
| `Taxes` | Fiscal | `spec-fiscal.md` sections 3.1, 5, 14 |
| `Duties` | Fiscal | `spec-fiscal.md` sections 3.1, 6, 14 |
| `Tolls` | Fiscal | `spec-fiscal.md` sections 3.1, 7, 14 |
| `Payouts` | Fiscal | `spec-fiscal.md` sections 3.1, 8, 14 |
| `Purchases` | Fiscal | `spec-fiscal.md` sections 3.1, 9, 14 |
| `Releases` | Fiscal | `spec-fiscal.md` sections 3.1, 9, 14 |
| `ID` | Tax (also the ledger reason) | `spec-fiscal.md` sections 3.1.1, 3.4 |
| `ID` | Duty (also the ledger reason) | `spec-fiscal.md` sections 3.1.1, 3.4 |
| `ID` | Toll (also the ledger reason) | `spec-fiscal.md` sections 3.1.1, 3.4 |
| `ID` | Payout (also the ledger reason) | `spec-fiscal.md` sections 3.1.1, 3.4 |
| `ID` | Purchase | `spec-fiscal.md` sections 3.1.1, 3.4 |
| `ID` | Release | `spec-fiscal.md` sections 3.1.1, 3.4 |
| `Authority` | Tax | `spec-fiscal.md` section 3.1 |
| `Authority` | Duty | `spec-fiscal.md` section 3.1 |
| `Authority` | Toll | `spec-fiscal.md` section 3.1 |
| `Authority` | Payout | `spec-fiscal.md` section 3.1 |
| `Authority` | Purchase | `spec-fiscal.md` section 3.1 |
| `Authority` | Release | `spec-fiscal.md` section 3.1 |
| `Window` | Tax | `spec-fiscal.md` sections 3.1.2, 5.1 |
| `Window` | Duty | `spec-fiscal.md` sections 3.1.2, 6.1 |
| `Window` | Toll | `spec-fiscal.md` sections 3.1.2, 7.1 |
| `Window` | Payout | `spec-fiscal.md` sections 3.1.2, 8.1 |
| `Window` | Purchase | `spec-fiscal.md` sections 3.1.2, 9.1 |
| `Window` | Release | `spec-fiscal.md` sections 3.1.2, 9.1 |
| `Start` | Window (every kind) | `spec-fiscal.md` sections 3.1.2, 14 |
| `Until` | Window | `spec-fiscal.md` sections 3.1.2, 14 |
| `Seasons` | Window | `spec-fiscal.md` sections 3.1.2, 14 |
| `Target` | Tax (a `FlowTarget`) | `spec-fiscal.md` sections 3.2, 5.1.2, 14 |
| `Amount` | Tax | `spec-fiscal.md` sections 5.1, 14 |
| `RateBP` | Tax | `spec-fiscal.md` sections 5.1, 14 |
| `Exempt` | Tax | `spec-fiscal.md` sections 5.1, 14 |
| `Per` | Tax | `spec-fiscal.md` sections 5.1, 8.1, 14 |
| `Region` | Duty | `spec-fiscal.md` sections 6.1, 14 |
| `Commodity` | Duty | `spec-fiscal.md` sections 6.1, 14 |
| `Payer` | Duty | `spec-fiscal.md` sections 6.1, 14 |
| `Scope` | Duty | `spec-fiscal.md` sections 6.1.1, 14 |
| `RateBP` | Duty | `spec-fiscal.md` sections 6.1, 14 |
| `On` | Toll | `spec-fiscal.md` sections 7.1, 14 |
| `Mode` | Toll | `spec-fiscal.md` sections 7.1, 14 |
| `Amount` | Toll | `spec-fiscal.md` sections 7.1, 14 |
| `Target` | Payout (a `FlowTarget`) | `spec-fiscal.md` sections 3.2, 8.1, 14 |
| `Commodity` | Payout | `spec-fiscal.md` sections 8.1, 14 |
| `Amount` | Payout | `spec-fiscal.md` sections 8.1, 14 |
| `Level` | Payout | `spec-fiscal.md` sections 8.1, 14 |
| `Per` | Payout | `spec-fiscal.md` sections 5.1, 8.1, 14 |
| `NotIncome` | Payout | `spec-fiscal.md` sections 8.1, 14 |
| `Commodity` | Purchase | `spec-fiscal.md` sections 9.1, 14 |
| `Quantity` | Purchase | `spec-fiscal.md` sections 9.1, 14 |
| `LimitBP` | Purchase | `spec-fiscal.md` sections 9.1, 9.2, 14 |
| `Hold` | Purchase | `spec-fiscal.md` sections 9.1, 14 |
| `Commodity` | Release | `spec-fiscal.md` sections 9.1, 14 |
| `Quantity` | Release | `spec-fiscal.md` sections 9.1, 14 |
| `LimitBP` | Release | `spec-fiscal.md` sections 9.1, 9.2, 14 |
| `Reserve` | Release | `spec-fiscal.md` sections 9.1, 14 |
| `Jurisdiction` | FlowTarget (treasury only) | `spec-fiscal.md` sections 3.2, 14 (the type is `spec-money-supply.md` 4.1) |
| `Institutions` | Modules (module; nil = off) | `spec-institutions.md` section 9 |
| `Bodies` | Institutions | `spec-institutions.md` section 9 |
| `Rights` | Institutions | `spec-institutions.md` section 9 |
| `Controls` | Institutions | `spec-institutions.md` section 9 |
| `RateLimits` | Institutions | `spec-institutions.md` section 9 |
| `OwnUseBound` | Institutions | `spec-institutions.md` sections 5.5.2, 9 |
| `ID` | Body | `spec-institutions.md` sections 3.2.2, 9 |
| `Roles` | Body | `spec-institutions.md` sections 3, 9 |
| `Regions` | Body | `spec-institutions.md` sections 3, 9 |
| `Jurisdictions` | Body | `spec-institutions.md` sections 3, 9 |
| `Places` | Body | `spec-institutions.md` sections 3, 9 |
| `MaxTrainees` | Body | `spec-institutions.md` sections 3, 9 |
| `Needs` | Body (a settlement flag) | `spec-institutions.md` sections 3, 9 |
| `ID` | Right | `spec-institutions.md` sections 5, 9 |
| `Body` | Right | `spec-institutions.md` sections 5, 9 |
| `Activity` | Right | `spec-institutions.md` sections 5, 9 |
| `Subject` | Right | `spec-institutions.md` sections 5, 9 |
| `Region` | Right | `spec-institutions.md` sections 5, 9 |
| `Jurisdiction` | Right | `spec-institutions.md` sections 5, 9 |
| `Who` | Right | `spec-institutions.md` sections 5.1.2, 9 |
| `ID` | Control | `spec-institutions.md` sections 6.1, 9 |
| `Commodity` | Control | `spec-institutions.md` sections 6.1, 9 |
| `Region` | Control | `spec-institutions.md` sections 6.1, 9 |
| `Jurisdiction` | Control | `spec-institutions.md` sections 6.1, 9 |
| `CapBP` | Control | `spec-institutions.md` sections 6.1, 9 |
| `FloorBP` | Control | `spec-institutions.md` sections 6.1, 9 |
| `ID` | RateLimit | `spec-institutions.md` sections 6.1, 9 |
| `Region` | RateLimit | `spec-institutions.md` sections 6.1, 9 |
| `Jurisdiction` | RateLimit | `spec-institutions.md` sections 6.1, 9 |
| `CapBP` | RateLimit | `spec-institutions.md` sections 6.1, 9 |
| `Saving` | Modules (module; nil = off) | `spec-saving-and-investment.md` sections 3.2, 10 |
| `SafetyTicks` | Saving | `spec-saving-and-investment.md` sections 5.6.1, 10.2 |
| `HurdleSpreadBP` | Saving | `spec-saving-and-investment.md` sections 5.5.2, 10.2 |
| `CommitBandBP` | Saving | `spec-saving-and-investment.md` sections 5.5.3, 10.2 |
| `Saver` | Role (the saver block; nil = no saving behaviour) | `spec-saving-and-investment.md` sections 3.2, 10 |
| `PrecautionShareBP` | Saver | `spec-saving-and-investment.md` sections 4.2, 10.2 |
| `PrecautionCapTicks` | Saver | `spec-saving-and-investment.md` sections 4.2, 10.2 |
| `HurdleBP` | Saver | `spec-saving-and-investment.md` sections 5.5.1, 10.2 |
| `PaybackTicks` | Saver | `spec-saving-and-investment.md` sections 5.3, 10.2 |
| `RepairBelowBP` | Saver | `spec-saving-and-investment.md` sections 5.8, 10.2 |
| `Worth` | Saver (the rows) | `spec-saving-and-investment.md` sections 5.3.2, 10.2 |
| `Class` | ProjectWorth | `spec-saving-and-investment.md` sections 3.2, 10.2 |
| `PerTick` | ProjectWorth | `spec-saving-and-investment.md` sections 3.2, 10.2 |
| `Count` | ProjectWorth | `spec-saving-and-investment.md` sections 3.2, 10.2 |
| `Technology` | Modules (module; nil = off) | `spec-technology.md` section 9.1 |
| `Estates` | Modules (module; nil = off) | `spec-estates.md` section 2.6 (the pool policy; no field of its own, `spec-config.md` 6.2.4) |
| `Fields` | Technology | `spec-technology.md` section 9.1 |
| `ID` | TechField | `spec-technology.md` section 3.1 |
| `Recipes` | TechField | `spec-technology.md` sections 3.1, 9.2 |
| `CapBP` | TechField | `spec-technology.md` section 9.2 |
| `Base` | TechField (learning) | `spec-technology.md` section 9.2 |
| `DoublingBP` | TechField | `spec-technology.md` section 9.2 |
| `TrendBP` | TechField | `spec-technology.md` section 9.2 |
| `DiscoveryBP` | TechField | `spec-technology.md` section 9.2 |
| `StepBP` | TechField (discovery) | `spec-technology.md` section 9.2 |
| `CatchUpBP` | TechField | `spec-technology.md` section 9.2 |
| `Reach` | TechField (diffusion) | `spec-technology.md` section 9.2 |
| `Policy` | Modules (module; nil = off) | `spec-policy.md` sections 3.1, 10 |
| `Actors` | Policy | `spec-policy.md` sections 3.1, 10 |
| `ID` | PolicyActor | `spec-policy.md` sections 3.1, 10 |
| `Jurisdiction` | PolicyActor | `spec-policy.md` sections 3.2, 10 |
| `Policy` | PolicyActor (a registered name) | `spec-policy.md` sections 4.5, 10 |
| `Every` | PolicyActor | `spec-policy.md` sections 7.2, 10 |
| `ReadsRegions` | PolicyActor | `spec-policy.md` sections 5.4, 10 |
| `Levers` | PolicyActor | `spec-policy.md` sections 3.4, 10 |
| `Schedule` | PolicyActor | `spec-policy.md` sections 8.2, 10 |
| `Cycle` | PolicyActor | `spec-policy.md` sections 8.2, 10 |
| `Rules` | PolicyActor | `spec-policy.md` sections 8.3, 10 |
| `ID` | Lever | `spec-policy.md` section 3.4 |
| `Kind` | Lever | `spec-policy.md` sections 3.4, 6.3 |
| `Flow` | Lever | `spec-policy.md` sections 3.5, 6.3 |
| `From` | Lever | `spec-policy.md` sections 3.5, 6.3 |
| `To` | Lever | `spec-policy.md` sections 3.5, 6.3 |
| `Mode` | Lever | `spec-policy.md` sections 3.5, 6.3 |
| `Against` | Lever | `spec-policy.md` section 3.4.3 |
| `Min` | Lever | `spec-policy.md` section 3.4.1 |
| `Max` | Lever | `spec-policy.md` section 3.4.1 |
| `MaxStep` | Lever | `spec-policy.md` sections 3.4.1, 6.2.2 |
| `MayClose` | Lever | `spec-policy.md` sections 3.4.1, 6.2.1 |
| `Lever` | ScheduleStep | `spec-policy.md` section 8.2 |
| `At` | ScheduleStep | `spec-policy.md` section 8.2 |
| `BP` | ScheduleStep | `spec-policy.md` section 8.2 |
| `Closed` | ScheduleStep | `spec-policy.md` section 8.2 |
| `Lever` | Reaction | `spec-policy.md` section 8.3 |
| `Gauge` | Reaction | `spec-policy.md` section 8.3.2 |
| `Commodity` | Reaction | `spec-policy.md` section 8.3.2 |
| `MaxAge` | Reaction | `spec-policy.md` section 8.3.2 |
| `Target` | Reaction | `spec-policy.md` section 8.3.1 |
| `Deadband` | Reaction | `spec-policy.md` sections 8.3.1, 8.3.6 |
| `Span` | Reaction | `spec-policy.md` section 8.3.1 |
| `Move` | Reaction | `spec-policy.md` section 8.3.1 |
| `Raise` | Reaction | `spec-policy.md` section 8.3.1 |
| `Insurance` | Modules (module; nil = off) | `spec-insurance.md` sections 4.1, 13 |
| `ValueCapBP` | Insurance | `spec-insurance.md` sections 6.2, 13 |
| `PayoutTicks` | Insurance | `spec-insurance.md` sections 8.4, 13 |
| `MinSum` | Insurance | `spec-insurance.md` sections 7.4, 13 |
| `MaxPolicies` | Insurance | `spec-insurance.md` sections 7.4, 13 |
| `MaxCoverRequests` | Insurance | `spec-insurance.md` sections 5.2, 13 |
| `Cover` | Role (the cover block; nil = not a cover role) | `spec-insurance.md` sections 4.2, 13 |
| `Underwrite` | CoverTerms | `spec-insurance.md` section 4.2 |
| `Insure` | CoverTerms | `spec-insurance.md` section 4.2 |
| `LoadingBP` | Underwrite | `spec-insurance.md` sections 6.4, 13 |
| `StoreRateBP` | Underwrite | `spec-insurance.md` sections 6.4, 13 |
| `CoverBP` | Underwrite | `spec-insurance.md` sections 4.2, 13 |
| `CapBP` | Underwrite | `spec-insurance.md` sections 4.2, 13 |
| `ShareBP` | Underwrite | `spec-insurance.md` sections 10.3, 13 |
| `MinLossBP` | Insure | `spec-insurance.md` sections 4.2, 10.2, 13 |
| `RuinShareBP` | Insure | `spec-insurance.md` sections 10.2, 13 |
| `MaxLoadingBP` | Insure | `spec-insurance.md` sections 10.2, 13 |
| `MinCoverBP` | Insure | `spec-insurance.md` sections 10.2, 13 |
| `StoreShareBP` | Insure | `spec-insurance.md` sections 10.2, 13 |
| `MaxStoreRateBP` | Insure | `spec-insurance.md` sections 10.2, 13 |

## New domain terms to add to the glossary

Minor unit, denomination, ladder, formatter, precision, household, reference basket, welfare ratio, padded
history, observed wage, observed premium, working tick, commercial trip,
target-income rule, tool, durability, estate, pool, relief, carriage market,
trainee, trainer, traineeship, trainee-eligible recipe, account, boundary, faucet, sink, pack, layer, pool policy, feedback loop, damper, viability, steady state, subsistence floor, bid, ask, limit price, admission, reservation, crossing, tie-break key, market price, price age, reference price, weight, scale, group, effective configuration, configuration hash, pack, presentation-only field, warning, sourced, producible, year, commodity, unit, mass, bulk, load, container, capacity, binding dimension, chargeable size, lot, born tick, transfer, reason (of a transfer), claim, balance sheet, sector, flow total, pseudo-agent, earmark, named lot, audit, shelf life, age, spoil, durability (listed), durable, tool (listed), commodity class, raw, calendar, calendar day, absolute day, moment, season, season cycle, ordered list, job, execution, executor, principal, duration, source (recipe with no inputs), co-product, byproduct, discard (output), disposal, yield factor, effective factor, site (sited recipe), yield profile, yield shock, residue, draw (endowment draw), pending output, work in progress, trainee yield, unit cost, cost anchor, cost share, price belief, centre (of a belief), width (of a belief), belief step, effective range, outcome (of an order), adjustment gate, stock ratio, target stock, seed price, price window, reference band, cost band, re-clamp, upkeep, effective upkeep, fed, unfed, hunger, tradable set, tradable (commodity), role, strategy, intent, trade plan, job request, purpose (of a bid), agent view, outcome view, work share, rest band, survival rule, planned recipe, reserve (units), keep, gap, surplus, target (stock), use target, output target, cover period, tier (of a bid), reserve money, spend cap, favourability, external agent, directive, directed strategy, mailbox, production stop, household labour, activity, start capacity, mean factor, net vector, reference programme, surplus factor, labour share to subsist, available work share, warm start, seed table, virtual start offset, stagger (rule), supply gap, lean reserve, balanced census, lane, residual, drift, register entry, failure regime, pinned market, stability scenario, evidence offset, effective window, working household-tick, wear cost, work floor, evidence share, full cost, regional wage, anchor (reference), starting wage, full unit cost, construction phase, construction bucket, periodic season, stated lot key, held to work (by the floor), spawner, privation, ladder (failure ladder), bankruptcy, entry share, entry limit, review (occupation review), cooldown, viability (of a role), expected ratio, renewal, entrant, mortal (external agent), role change, want, rank, taste, schedule, choke price, ceiling (of a schedule), substitution share, want carry, discretionary budget, nominal seed, final demand (wants), variant, overlay, product line, balanced census, stable-state predicate, bands document, shock script, cold variant, report-only (scenario), region, region graph, edge, route (an edge), distance, `RegionID`, `EdgeID`, `NoRegion`, hop, shortest path, next hop, component (strongly connected), island, edge status, effective length, factor (of an edge), relocation, relocation totals, region override, effective reference price, `Graph`, money supply, money flow, grant, levy, top-up, account class, flow carry, flow scale, declared reason, holding charge, liquidity floor, spending threshold, closed economy, balanced pair, birth (of a household), death (of a household), schedule (birth, death), knot, replacement ratio, exposed household, natural increase, birth stake, household cap, death carry, birth carry, fixed factor, gain (of the population loop), money-limited population, mode, self-carried mode, hired mode, carrying unit, crew (of a mode), haul unit, haul span, traveller, shipper, consignee, shipment, dispatch, arrival, transit account, transit factor, away (agent), inbound, diverted (cargo), resource, endowment, stock (of an endowment), capacity (of an endowment), regeneration, shape (of a resource), carry (regeneration carry), refuge, extinct (stock), finite stock, abundance, abundance curve, plan share, plan stock, ration (of an endowment), grant (of a ration), request (of a draw), maximum flow, maximum sustainable draw, sustainable draw, runway, open access, binding (endowment), headroom, `ResourceID`, rumour, price record, observation (an entry of the price record), source (information source), vintage, lag (of a rumour), information speed, rumour error, horizon (of a rumour), informed (role), outside, outside node, gate, line (of an outside node), side (sell side, buy side), mid, quote, margin (of a side), counterparty, Outside (account kind), outside sector, import, export, balance of trade, list value, drift (of a walk), shock (of a line), availability window, capacity (of a side), cap (cumulative), sweep, top-up (of a counterparty), own-use recipe, own-use mode, covered set, cover (of upkeep), afford (ticks), secure ticks, coverage (of an own-use recipe), retry (the market retry), blocked (household), ramp (of an own-use job), migration, migrant, journey (of a migrant), earned ratio, settled (agent), pushed (household), estimate (of a destination), margin (of a move), width (of the margin), horizon (of a move), outflow cap, pipeline cap, provisioning (of a journey), `Fixed` (role flag), merchant, planned trip, lane (of a merchant), candidate (lane and mode), lot (of a merchant), lot cap, cargo set, trade inputs, fee share, bar, cautious revenue, break-even price, ask limit, lane lock, delivery offer, delivery take, payer, deliverer, contract round, delivery contract, escrow (of a contract), settlement (of a contract), plan notes, outbound, tick boundary, phase, position (of a tick), entry (of a tick), slot, between ticks, cadence, rotation, bucket (review bucket), Weyl value, keyed tie, failed world, word (of a stream), draw site, site constant, stream key, keyed stream, effective seed, the world's draw, creation draw, carried remainder, window (memory), pending totals, names table, volume class, recorder, level (of an instrument), delta instrument, region state, snapshot row, construction flow, run (of a bounded `Warn`), pattern A, pattern B, `warnRun`, `NoCommodity`, `WantID`, checkpoint, restore, format version, section (of a checkpoint), frame, header (of a checkpoint), trailer, digest (of a checkpoint), canonical encoding, one spelling, sparse row, construction flag, extension state, `Stateful`, entry (of an extension), reseed (of the seed), salt, branch, fixture (of a format), agent state, forced move, market state, slot state, query class, entry (of a setter), key (of a slot), indicator, preset, aggregator, tally, window (of ticks; the same term as EXP's window (sample)), household row, price table, base window, index (basis points of a base), Laspeyres, Paasche, Fisher, chain index, clamp (of an index), stale share, cohort share, value added (of a region), production approach, expenditure approach, income view, gross accumulation, written (goods), carriage (use of goods in moving goods), delivered price, freight, trade valuation, retirement basis, binding counters, reconciling item, stylised fact, fact (STY), band, prior, setting seeds, hold-out set, headroom (of a band), acceptance class, exact fact, ordinal fact, banded fact, report fact, pack-asserted fact, confidence (of a band), blind-spot field, detector, verdict, unknown (verdict), onset, healthy start, provoked run, separation protocol, stability check, stability record, verdict record, tier (of a test), experiment, factor, level, change, design (grid, one at a time, spread), point, replica, run key, unit (of work), plan, record (of a run), result, transient, final reading, series, measure, criterion, pass share, reduction, variant (of calls), continuation, aligned (prefix, horizon), stream fingerprint, pairing gain, effect range, elasticity, knob, damper strength, structural choice, registry (of knobs; the parameter-owner table is a different thing), search range, evidence anchor, derived condition, base value, grid (of a knob), neighbour (of a grid point), edge (of a grid; the edge of the region graph is REG's), screen, screened in, dormant, invisible, alias group, gauge (of a calibration; OBS's gauge is an instrument), composite (knob), search block, confirm seeds, feasible region, minimal change, grid distance, Pareto set, known miss, working target, working threshold, calibration overlay, calibration record, lineage, re-entry, stage (of a calibration), attempt, off value, structure group, scope token, overlay document, claim set, sensitivity matrix, scaling check, stage stop, program stop. Gate, hold-out, finding, stability record, elasticity and effect range are used as STY, EXP and IND define them and are not redefined by CLB.

Terms of the fourteen later-stage modules, each once, as each spec's N-017 list gives them. A word that an earlier spec or another module also uses is written in its qualified spelling. **Qualified spellings:** commodity class (a class of goods, `spec-goods.md`), account class (a class of accounts, a `FlowTarget` of MON and of FIS), store class (STO), settlement class (SET, always so written), facility class (CAP); claim is the ledger's (INS's is a payout claim); contract is MER's delivery contract (EMP's is a hire; a hire is not a contract of MER); facility is CAP's (EMP says the employer's tool); treasury is POL's account and pseudo-agent of a governed jurisdiction, never the region's pool; rent is LND's (ORG says premium, STO does not use it); levy is MON's (a transfer to a treasury is a tax); round is always "<module> round" (contract round, labour round, credit round, cover round, membership round) and a lane of such a round is a region ("lane of the credit round"), while "lane" alone stays MER's pair of a destination and a commodity; policy is PLC's interface `Policy` (an actor's policy is a name), an insurance policy is INS's, and a pool policy is the estates'; member is a direct member of a jurisdiction (POL) or a body member (ORG); ceiling is a role ceiling (SET), and the ceiling of a schedule is DEM's; reach is a jurisdiction's reach for a lever (PLC), a field's reach (TEC) and reachable (SAV); place is a body place (ORG), a trainee place (SKL) or an order place (WLD); need is a class need (a flag), a facility need (a recipe's list of facility classes) or a body need; cap and floor are a legal cap and a legal floor (ORG), a class floor (SET), a field cap (TEC) or a capacity cap (INS); headroom is qualified each time ("headroom of a role's ceiling", of a body, of a debtor); gate is a class gate (SET); limit is a class limit (SET, a count); hold and held are a key holder (PLC), a title holder (LND), a held change (SET) and a held completion (ORG); release is a standing order (FIS), a title release (LND) or the pool's `PoolRelease`; gain is a technology gain (TEC) or a layer gain (CAP); keep is a keep of a job (SAV) or a lender's keep (CRD); payout is a fiscal payout (FIS) or an insurance payout (INS); premium is an insurance premium (INS), a technology premium (TEC) or WEL's observed premium; window is a fiscal window (FIS); base is a base of a duty (FIS) or a base of a field (TEC); scale is always "<noun> scale" (risk scale, rent scale, rate scale, flow scale); schedule, directive, mailbox and gauge of an actor are PLC's ("gauge of a rule").

**STO:** store, room, mass room, bulk room, base room, limited (account), load, pledge, reserved room, free room, stored load, owns a store, fits (a load fits a room), commit (room is committed at the decision), upkeep room, cut of the room, buffer store, `StoreClassID`. **EMP:** employment, employer, worker (wage earner), hire (the one-job contract and its record), `HireID`, hire list, establishment, vacancy, positions, labour offer, reservation wage (not the ledger's reservation counter), employer's bid (not an order side), wage (money per household-tick; never a field), worker share, escrow (of a hire), payroll, unmatched (household), unemployment (a measure), hired yield, `open_hire`. **SET:** `SettlementClassID`, `NoSettlementClass`, enter threshold, leave threshold, band (of a class; the band of a stylised fact is STY's), dwell, settlement review (P7.2a), household count (of a region), flag (of a class), limit (of a class), gate (a reader's use of a flag or a limit), class floor, held change, headroom (of a role's ceiling), ceiling (of a role). **LND:** tenure, title (a claim of ownership of a share of an endowment, never a rank or an honorific), holder, commons share, entitlement, unused entitlement, let (verb), lessor, tenant, tier (of the ration), posted rent, rent in force, rent scale, closed (an endowment), enclosure, apportionment, `NoAgent`, priced out, forfeited. **POL:** jurisdiction, polity, root (of the tree), chain (of a region), covers, common jurisdiction, crossing (of a border; not the clearing crossing), exit, enter, border, border depth, seat, governed (jurisdiction), grouping, treasury sector, combining mode (innermost, added, scaled), `JurisdictionID`, `NoJurisdiction`. **CRD:** credit (the module; nothing else is called credit), loan, creditor, debtor, lender, borrower, request (for credit), offer (of credit; a delivery offer is MER's), posted rate, effective rate, rate scale, term, principal (of a loan; the principal of a job is PRO's), interest, owed, due tick, overdue, arrears, shield, settlement of claims (the pass of P1.7a; not EXT's sweep, not a settlement of a contract), waterfall, passed (claim), written off, forgiven, exposure cap, credit room (`CreditRoom`; not STO's room), cash gap, horizon (of a gap; not MIG's, RES's or INF's). **CAP:** facility, kind (of a facility class: site, route, ground, store), site, route (a facility's; an edge is REG's), ground improvement, store facility, condition, decay, use wear, effect fraction, ruin, repair, build (a job whose completion creates facilities), builder, repairer, commissioning, effect table, install (of a store facility), pending (a store facility not installed), durability and capacity (of the class). **RSK:** risk row, route row, store row, exposure (of goods; not POP's exposed household), strike, severity, common row, unit row, effective probability, loss share, kept share, stored fraction, expected loss (of a route), total loss, trip account. **FIS:** fiscal flow, authority, tax (head tax, holding tax, remittance), duty (sales tax, purchase tax, tariff), toll, in kind, standing order (purchase, release), pro rata cut, counterparty (of a trade; EXT's), payer (of a tax or a duty; MER's payer of a delivery contract is another), payee. **ORG:** body, outsider, eligible (household), places (of a body), seniority, held completion, right (not LND's title), activity (`produce`, `sell`, `buy`, `train`; not VIA's), subject (of a right), scope key, prohibition, in force (a right, a range), suspended (right), `Who` (principal, executor, both), legal price, control (a price control; not the controller of a loop), rate limit, legal range, clamp, `BodyID`, `ControlID`, `RightID`. **SAV:** saving (the module; money held back), saver (a role with the block), savings target, floor (of the target), precaution buffer, drawdown, goal, project, margin test, benefit, cost (of a project), job value, hurdle, hurdle spread, payback horizon, commit band, outlay, funded (a household for a project), worth (a pack's value of a whole facility), idle money, standing refusal, bar (the four reasons a project is not taken; MER's bar is another). **TEC:** technology (the module; not VIA's technology matrix), technology field, cell (of a field in a region), experience, learned gain, advance, technology factor, doubling (of the cumulative work), trend, discovery, key (of the module), contact, contact weight, frontier, gap, catch-up, practises (a cell). **PLC:** policy (the interface), policy actor, lever, mandate, decision (of an actor), due (actor), cadence (of an actor), reading, target (of a rule), deadband, span, move, step limit, rest (of a rule), reaction, step and cycle (of a schedule), `ActorID`, `LeverID`, `LeverKind`, `PolicyView`, `Settings`, `HoldPolicy`, `Against` (a lever's group). **INS:** cover (insurance; not the cover of upkeep), underwriter, insured, sum insured, maximum payout, cover share (`CoverBP`), loading, store rate, capacity (of an underwriter), open liability, unit value, value cap, loss value, due (of a policy), payout claim, shortfall, lane request (a mode and a destination; the lane of the cover round is a region).

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
| How does a player act inside the synchronous `Step`? | Through a directive left in a mailbox between steps, read by the `Directed` strategy, or through a non-blocking consumer strategy; no immediate action | `spec-agents.md` section 10, `spec-integration.md` sections 3.6 and 4.3 |
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
| Is a strategy name checked by `Validate`? | No, by `NewWorld`, after `Validate` succeeds: `Validate` takes no options, and the three name checks (strategy, spawner, clearing) return a `ValidationError` with the path and the rule | `spec-config.md` section 7.1.4, `spec-world.md` section 3.2.2 |
| Is `ExpectedWorkTicks` scaled when the year is short? | No, it is an error | `spec-config.md` section 13.5 |
| Is there a runtime stability controller? | No; stability comes from local rules, and the register records the loops and dampers | `spec-stability.md` sections 2.6, 5.1.3 |
| What anchors a staple's price? | The belief bounds, the production halt, buffer buying, recipe choice and exit; the halt alone does not anchor it | `spec-stability.md` section 6.2 run 1b |
| What is bankruptcy? | A leaky privation count read by the failure ladder, not money at 0 and not the one-tick `Hunger` | `spec-spawner.md` section 5.1 |
| In what order does the failure ladder act? | Rung 0 the survival rule, rung 1 role change, rung 2 relief (unconditional), rung 3 weight reduction by the unfed share and then removal, capped by `ExitShare` | `spec-spawner.md` section 5 |
| What damps occupation change? | The entry share (the damper that works in the model), with rotation of reviews, hysteresis and a cooldown | `spec-spawner.md` sections 6, 16.1 |
| Which role replaces a leaver? | Like for like if viable, else the best viable role, else none; not the most profitable | `spec-spawner.md` section 7.2 |
| Is `AgentBankrupted` the only removal event? | Yes, with `Cause` `insolvent`, `starved`, `consumer` or `population`; `WeightChanged` and `RoleChanged` are not removals | `spec-spawner.md` section 13.1.1 |
| Who owns births and deaths? | POP; the spawner has a gross hook (`PopBirths` per role, `PopDeaths` per agent) and does not write the demographic rates | `spec-spawner.md` section 8, `spec-population.md` 4 |
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
| Is a resource a commodity? | No: it has no price, no ledger account and no mass; a draw consumes endowment and the recipe's output is the commodity | `spec-resources.md` section 3.1 |
| Who owns the yield-table values? | RES; PRO keeps the format; there is no climate entity | `spec-resources.md` section 7 |
| How is a scarce endowment shared? | A proportional ration, leftovers by largest remainder then a stateless hash key; not first come, and not the rotation of production. The start loop is two passes: `Work` for all agents, the ration, then the starts | `spec-resources.md` sections 5.2, 5.3 |
| Is depletion permanent? | Data: a finite stock is, a refilling stock is not, a logistic stock is unless a refuge is set | `spec-resources.md` section 6.1 |
| What does the ledger see of a draw? | Nothing: no new reason, and an exact identity per endowment instead | `spec-resources.md` sections 6.2, 6.4 |
| What fixes the Malthusian loop? | An endowment that binds; whether the loop is stable depends on its shape | `spec-resources.md` section 8 |
| What sources a `Raw` good? | A recipe with no inputs, a sited recipe with `Draws` on an endowment that can start at least once (RES), or an import line with a sell side (EXT); the marker `Raw` still counts as sourced in the viability check, and either source satisfies `raw_without_source` | `spec-resources.md` section 9, `spec-external-sector.md` section 10.1 |
| What does an agent know about other regions? | Only its own region's market report for free; a remote price arrives as a rumour, at least 2 ticks old (the local report is 1) and blurred by an independent bounded error, only to roles flagged `Informed` | `spec-information.md` sections 3, 5, 6 |
| Who holds rumours? | The world holds one price record per source market (a ring of published prices); a rumour is computed when an informed agent's view is built; there is no per-agent or per-pair table | `spec-information.md` sections 4, 5.2 |
| Does news relay hop by hop, and does a trip carry news back? | No to both in version 1: the lag is `1 + ceil(shortest distance / InfoSpeed)` in one step, and a trip changes no record | `spec-information.md` sections 5.2, 7.5 |
| How does the outside trade? | Through counterparties: pseudo-agents with a pass-through account of the kind Outside, topped up in phase 3 with exactly what their order reserves and swept to zero in phase 4; six built-in reasons; clearing is unchanged, and the quote is a limit and not a peg. A boundary-backed admission was rejected (ADR 0004) | `spec-external-sector.md` sections 4, 6.7 |
| Where does the outside's money enter the pool? | It does not: it reaches the households first, and the pool only through the estates | `spec-external-sector.md` section 11.5 |
| Do outside goods ride the transport module? | No: a shipment to an outside node is refused with `outside_destination` and a relocation into one with `ErrOutsideNode` | `spec-external-sector.md` section 4.4, `spec-transport.md` section 5.4 |
| Is an outside node a source of rumours? | No in version 1: INF records only a market's published price and EXT has no declaration and no outside-price field for INF (N-112) | `spec-external-sector.md` section 9.2 |
| What is household production for own use? | An ordinary source recipe named on a list (a module default and a per-role row that replaces it), sited, with no inputs or tools; no flag on a recipe and no new recipe format | `spec-subsistence.md` section 3 |
| What is its output in the ledger? | An ordinary recipe execution into the household's account (`recipe_output`) eaten by the ordinary upkeep; no new reason, no in-kind channel | `spec-subsistence.md` section 6 |
| Why is the market preferred to the subsistence floor? | A bound on productivity (the coverage of an own-use recipe stays below the region's surplus factor, a warning) and an entry test on the household's own secure ticks | `spec-subsistence.md` sections 4.2, 4.4 |
| When does a household take up own-use work and leave it? | The world computes a flag from the household's own stock, money, `Hunger` and own-market price: enter below `EnterTicks` secure ticks, leave when stock cover reaches `ExitTicks`, a market retry every `RetryJobs` own-use jobs | `spec-subsistence.md` section 5 |
| How does own-use output count in the welfare ratio and the observed wage? | An in-kind receipt at creation, valued at the prices of `cost_t`; excluded from the observed wage with its ticks | `spec-subsistence.md` section 7 |
| Does the subsistence floor remove the pinned regimes? | No: it removes the shortage pin for households it can feed; the price can fall to the glut pin; it is not a damper of the target-income loop | `spec-subsistence.md` section 11 |
| What moves in a migration? | A whole agent, with all its households; weight 1 is the first consumer's case; a cohort moves whole and lockstep, a split is not in version 1 | `spec-migration.md` section 3.2 |
| What may a household read to decide to move? | Its own state, the structure of the world, and rumoured prices; no other region's wage, ratio, stock or count of households, and no regional average | `spec-migration.md` section 4 |
| What triggers a move? | A staggered review of a settled household whose own earned ratio (without relief) is below a gate, and a destination whose estimated income, discounted by the journey, beats its own measured wage by a margin | `spec-migration.md` sections 5, 6 |
| Does relief suppress migration? | No: relief is excluded from the earned ratio and the observed wage that the decision reads; what remains is the relief a poor arrival draws from the destination's pool (answers "Still open" 4) | `spec-migration.md` section 5.4 |
| Is the replacement ratio part of the migration signal? | No | `spec-migration.md` section 5.5 |
| Is a migration instant or timed? | Timed: a journey in the shape of a trip, the account staying in the origin until arrival, then the ledger-level `Relocate` with the cause `migration`; `on_journey` is a registered refusal; no `NoRegion` | `spec-migration.md` section 8.1 |
| What does a move cost and who pays? | The household, in time, in the job in progress and in the upkeep of the journey from its own stock; nothing is consumed, created or destroyed by the move | `spec-migration.md` sections 6.4, 7.4, 9 |
| How is an arrival funded? | It is not: the migrant carries its account, takes no stake and nothing from either pool | `spec-migration.md` section 9.2 |
| What damps migration? | A review rotation, a settling rule (at least the welfare window), a margin with a width, an outflow cap, a cap on the households on the way to a destination, a horizon and provisioning | `spec-migration.md` sections 5 to 7, 17.1 |
| Who receives a hired shipment? | The payer of a delivery contract | `spec-merchants.md` sections 8, 9 |
| How does a merchant decide between travelling and hiring? | One rule for every mode: the cost of carriage is at most a fee share of the cautiously read gross margin; the largest net gain wins | `spec-merchants.md` sections 5.5, 6.1 |
| How does a merchant estimate `expected_margin_per_haul_unit`? | `G / (n*s)`, with `G` from the rumour read as `min(Price, Mean)`, bounded by `SourceAge` (answers "Still open" 6) | `spec-merchants.md` sections 5.2, 5.6 |
| What is a trip's time worth? | The merchant's own upkeep cost times `TimeCostBP`, never a regional wage | `spec-merchants.md` section 5.4 |
| How are identical shippers kept from herding? | A lane lock, a lot cap, a fixed spread by agent ID (the Weyl offset), a `SourceAge` bound, no memory | `spec-merchants.md` section 7 |
| What does a delivery contract add to the book? | A buyer across a region boundary, a price fixed before the goods move, payment on arrival from escrow, a defined outcome of failure; offers and takes are matched by a first-fit round after the last plan of phase 3, not by `Clearing` | `spec-merchants.md` section 9.1, [ADR 0005](../adr/0005-delivery-contracts-are-matched-by-a-first-fit-round.md) |
| Who bears spoilage and a removal? | The deliverer bears spoilage; a removed deliverer refunds the payer; a removed payer is replaced by the estate | `spec-merchants.md` section 9.7 |
| Does MER price backhaul? | No; the double count stands and is measured (answers "Still open" 5) | `spec-merchants.md` section 10 |
| Does a traveller keep its beliefs? | No, they are reseeded on arrival, as a migrant's are | `spec-merchants.md` section 12.2, `spec-migration.md` section 8.5 |
| What state is a world in after `Step` returns an error? | Failed: every later call with an error result returns `ErrWorldFailed`; the readers with no error result still work (`Err`, `Tick`, `Config`, `ConfigHash` and the derived tables return a value; the readers of live state return the state the step left, with no guarantee, so read `Err` first); the consumer restores a checkpoint. No half-finished tick is observable | `spec-world.md` section 3.5, `spec-integration.md` section 6.1.2 |
| Can a call be made from inside a step? | No: `ErrInStep`, nothing changed; readers with no error result return their value | `spec-world.md` section 3.6 |
| How many random sources and how is a seed used? | One seed, and one `*rand.Rand` on one `rand.PCG` per world that is reseeded at each draw site from a key of (seed, site, tick, id) by SplitMix64 finalisations; every bounded draw is one `Uint64` word of its stream and the high word of its product with the bound (ADR 0008) | `spec-world.md` sections 7.1 and 7.2 |
| Is the count of draws fixed? | For each stream a function of the configuration and the state, never of an observer, a logger, or a module that is on and inactive; no stream depends on the count of another; between-ticks calls draw at the call from the stream of their row | `spec-world.md` sections 7.4 and 8 |
| How is a decision rate applied? | One of four devices: rotation by the bucket of the agent ID, a carried remainder over `TicksPerYear`, a window or a gate; a cadence that does not divide the year is legal and never rounded; no rate from an aggregate | `spec-world.md` section 9 |
| Who wins a tie? | In new specs a random key or a rotation, never the lower ID; the written specs that favour a low ID or an early role (the path ties by smallest region index among them) are the stated exception and are registered hazards | `spec-world.md` section 10 |
| Is `Work` one pass or two? | Always two passes, whatever modules are on (the ration of a scarce endowment needs it, `spec-resources.md` sections 5.2 and 5.3; the rule no longer depends on the module) | `spec-world.md` section 6.4.1 |
| What does the world add to a checkpoint? | The next agent ID and whether it is still in construction (the tick and the effective seed are in the header, and no random state is saved), and the pending flow totals of between-ticks calls | `spec-world.md` section 12 |
| What is a checkpoint? | An opaque immutable value of canonical bytes; no struct of state is exported | `spec-checkpoint.md` section 3 |
| How is a checkpoint encoded? | Minimal LEB128 varints (zigzag signed), no map, float or padding, framed sections in a fixed order, a SHA-256 trailer; accepted input re-encodes to the same bytes | `spec-checkpoint.md` sections 5.2 and 5.4 |
| How is a restore validated? | All or nothing, in a fixed order; the ledger's audit always runs; the configuration is not in the checkpoint and its hash must match | `spec-checkpoint.md` section 6.2 |
| Can a restore reseed? | `WithSeed` is refused (`ErrOptionContext`); `WithReseed(salt)` gives a branch its own randomness deterministically, by replacing the seed with `finalise(seed xor finalise(salt))` | `spec-checkpoint.md` sections 6.4 and 6.8 |
| How is an extension's state saved? | An optional `Stateful` pair per registered name with a version; an `Observer` never | `spec-checkpoint.md` section 7 |
| Are closed accounts compacted? | Never in version 1 | `spec-checkpoint.md` section 9.4 |
| Which formats restore? | Before 1.0 only the exact version; from 1.0 every v1 format, never a newer | `spec-checkpoint.md` sections 9.1 and 9.2 |
| Is a world checkpointed before its first step still in construction? | Yes: it is restored in construction and keeps its construction bucket; one checkpointed after its first step is not | `spec-checkpoint.md` sections 4.4 and 6.7, `spec-world.md` section 5.5.3 |
| Does a shock expire? | No: a slot holds its value until another call sets it; the consumer clears it (no timer, callback or queue) | `spec-integration.md` section 5.2.2 |
| Where does a band live? | STY defines the fact and its prior; the value a test asserts lives in `testdata/chains/bands/<pack>.toml` and is set only by the protocol of 8.4; regime thresholds are global, in `bands/regimes.toml` | `spec-stylised-facts.md` sections 3.5, 6.5 and 8.4 |
| What is a regime? | A property of a run that a detector classifies (none, detected, unknown), not a failed test; a run may be in a regime and pass; a threshold is looser than the band of every fact it backs | `spec-stylised-facts.md` section 6.4 |
| What may be called stable? | Only the verdict of one of three named stability checks over a stated set of packs and seeds; a default is stable only if the checks pass at the default and both grid neighbours | `spec-stylised-facts.md` sections 7.3 and 8.6 |
| Does the experiment harness draw randomness? | No: it is counter-based and stateless | `spec-experiments.md` section 4.5 |
| How does a point differ from the base? | By overlay documents through the loader | `spec-experiments.md` section 3.3 |
| Can a branch change a parameter? | No: the hash | `spec-experiments.md` section 6.2 |
| When is a paired comparison valid? | Exact only while the runs are aligned; the estimate of a difference of means is unbiased always; the gain is measured | `spec-experiments.md` section 7 |
| How is a failed run counted? | Never dropped | `spec-experiments.md` sections 8.3.1 and 9 |
| Where is the registry of knobs? | In `spec-calibration.md` 6; the owner table is not given a column (4.5, 12.2) | `spec-calibration.md` sections 4.1, 4.5 |
| What is a calibration for? | Feasibility and minimal change, never an optimum | `spec-calibration.md` section 3.4 |
| Which seeds may a calibration choose on? | Search blocks of 100 from 101, one choice each; the confirm on STY's setting seeds 1 to 50 (REF 8.2); never the gate; a hold-out 51 to 60 is a SHOULD of REF 8.2 and no band is set from it | `spec-calibration.md` section 7.1 |
| What transfers between settings? | The procedure, the conditions and the regime thresholds; defaults and priors as starting points; no band, record or table | `spec-calibration.md` section 9.3 |
| What is a store? | Room on the owner's account (a pair of mass and bulk added to its base room) | `spec-storage.md` section 3 |
| When is room tested? | At the decision | `spec-storage.md` section 5.1 |
| What does a store cost? | A fixed levy per store, a MON flow | `spec-storage.md` section 7 |
| Does a store slow ageing? | No | `spec-storage.md` section 8.1 |
| What does an owner of a store lose? | The freedom to move | `spec-storage.md` section 9 |
| Is labour a commodity? | No: a wage is money between accounts under a ledger reason | `spec-employment.md` section 3.5 |
| Who matches workers and employers? | A first-fit round at P2.5a | `spec-employment.md` section 5 |
| What sets a wage? | The midpoint of the two sides' own values, with a worker share | `spec-employment.md` sections 5.4, 4.3 |
| What limits hired employment in a closed economy? | The employers' spending | `spec-employment.md` section 11 |
| What is a settlement class? | An entry of an ordered list, ordered by its `Enter` threshold in households; the `SettlementClassID` is the index, and a setting with no sizes has one class | `spec-settlements.md` section 3.1 |
| What is a settlement class derived from? | The household count of the region, at one position (P7.2a), with a band and a dwell against flapping | `spec-settlements.md` sections 3.4, 4 |
| What may read a settlement class? | Only the rows of the table of hooks, each by its owner spec at a stated position; no view of a strategy carries a class | `spec-settlements.md` sections 6.2, 6.3 |
| What may a settlement class not do? | Change a price, a wage, a rate, a share, a probability, a period or a window; a gate bars entry and removes no household | `spec-settlements.md` sections 6.2, 7 |
| Is rent a debt, and who is paid? | No: a transfer of money per unit drawn from a let unit, from the principal that runs the job to the lessors, under the internal reason `rent`; not a MON flow, never a debt (a start that cannot pay is refused `rent_funds`) | `spec-land.md` sections 2 (positions 6, 10), 5 |
| Is the polity a tree or a graph? | A tree, not a DAG: a region's chain is one path to the root | `spec-polity.md` sections 2, 3.3 |
| What is the actor of a polity? | A treasury account and pseudo-agent of a governed jurisdiction only | `spec-polity.md` section 5 |
| Does a polity change a result by being on? | No, while no jurisdiction is governed: it is neutral | `spec-polity.md` FR-POL-019 |
| Does a region inherit a value through a jurisdiction? | No: readers use the three combining modes | `spec-polity.md` section 6.1 |
| Can membership change in a run? | No: it is static in version 1 | `spec-polity.md` section 7.4 |
| Does a loan create money? | No: the principal moves from the lender's account to the borrower's and a claim is formed; a setting that wants money created declares a MON grant | `spec-credit.md` sections 2, 3.4 |
| Is a loan renewed? | Never: the due claim is settled at P1.7a before any request of the tick is admitted at P3.3b, and a debtor with an overdue claim is refused | `spec-credit.md` sections 2, 6.1.3 |
| What does a removal do to claims? | A debtor's money pays its claims pro rata before the rest goes to the estate (the waterfall); the claims of a removed creditor pass to the estate | `spec-credit.md` sections 2, 9 |
| Is a facility a good? | No: module state | `spec-infrastructure.md` section 3.2.2 |
| How does a road speed an edge? | A layer multiplied into the transit factor | `spec-infrastructure.md` section 5.2.2 |
| Who builds? | A recipe with `Builds`, decided by a strategy or SAV | `spec-infrastructure.md` section 4.5 |
| Does a facility register a region-bound refusal? | No, except a store's | `spec-infrastructure.md` section 7.5 |
| What is a risk? | A row of data: a strike probability, a severity share and a `Common` flag, per directed edge and mode or per store class in a region; no actor, no macro state | `spec-risk.md` section 3 |
| When is a cargo's fate decided, and when is it told? | At dispatch (one word per unit row of the path; common rows once a tick at P3.1a), kept in the shipment as `LossBP`; applied at arrival, before the delivery | `spec-risk.md` sections 4, 5 |
| What does a strike take? | A share of every commodity, rounded half to even, from a Transit account and the traveller's own account, or the struck store's share of the account; never money, never a chosen set of goods | `spec-risk.md` sections 5.3, 6.3, 7.4 |
| What does a shipper see? | The expected loss of a route, in `Routes`; never an outcome, a draw or a scale as such | `spec-risk.md` section 8 |
| Is the merchant's margin net of the loss? | Yes, the cautious revenue is net of `LossBP` | `spec-risk.md` section 8.4 |
| How is the embargo modelled? | As REG's edge status, set by a consumer or a border; RSK adds no mechanism and no autonomous closure | `spec-risk.md` section 10.1 |
| Is a probability scaled by a slot? | Yes, the strike probability only; two slots; no timer | `spec-risk.md` section 9 |
| Does a row with probability 0 draw? | No: so the module on with every scale 0 equals the module off | `spec-risk.md` sections 3.4.3, 11.3 |
| Does an insurance policy create money, and what if the underwriter cannot pay? | No: a premium and a payout are transfers; the part an underwriter cannot pay is a payout claim that CRD holds, settles and passes to the estate | `spec-insurance.md` section 2 (positions 6, 7) |
| Does a fiscal flow create money? | No: every fiscal reason is internal between non-boundary accounts; a mint or a tribute is a MON grant or levy with the target kind `treasury`; six typed lists; a treasury that holds less than is due cuts its payouts pro rata | `spec-fiscal.md` section 2 (positions 2, 3, 9) |
| Is a price control a clamp, a rejection or a strategy rule? | A clamp in the admission of the market: a bid's limit is lowered to a cap, an ask's raised to a floor; no new rejection reason | `spec-institutions.md` sections 6.3, 6.4 |
| When is a right read? | At the start of the activity (a production start, the building of an order): one refusal reason, `not_permitted` | `spec-institutions.md` sections 5.2, 5.6 |
| How are the members of a body chosen? | By seniority: the longest in its role, then a stateless key, then the agent ID; a household whole or not at all; no draw | `spec-institutions.md` section 4.3 |
| What do a body's places limit? | A count of the households of its roles in each region, as one more limit on the headroom (a ceiling); it is not enforced on those already there | `spec-institutions.md` section 3.4 |
| Why does a household hold money back? | The reserve, a buffer from its own drawdown, a goal for a project | `spec-saving-and-investment.md` sections 2, 3, 4 |
| How does an agent decide to build or repair? | A project test with a hurdle spread by agent, in the default strategy for a role with a `Saver` block | `spec-saving-and-investment.md` section 5 |
| How is a project funded? | From the household's own money; the default asks for no loan | `spec-saving-and-investment.md` section 6 |
| Where is knowledge held? | In a cell, a region and a field, not an agent | `spec-technology.md` section 3.5 |
| How does a gain enter production? | As a term `tech` of the effective factor | `spec-technology.md` section 5.1 |
| Does a gain change the warm start? | No, it is 0 at tick 0 | `spec-technology.md` section 3.3.2 |
| How is discovery random? | A keyed hash of one construction word, no word in a tick | `spec-technology.md` section 4.4 |
| What does a gain do to the Malthusian fixed point? | Moves it; a logistic stock can be pushed past its maximum-yield point | `spec-technology.md` sections 13, 10.3 |
| Does a policy actor reuse `Strategy`? | No: its own interface `Policy` with one method, `Decide`; a fifth row of architecture.md 4.3 | `spec-policy.md` section 4, [ADR 0003](../adr/0003-version-1-scope.md) decision 3 |
| What can a policy actor change? | Only levers: a named, bounded, rate-limited handle on a flow scale, an edge status or a transit factor; the lever list is its mandate; one holder per key | `spec-policy.md` sections 3.4, 3.5, 6 |
| When does a policy actor act? | At P0.3a, the entry of the step, on a cadence by rotation of its index; its settings act as a consumer's call between the two steps would | `spec-policy.md` section 7 |
| What does a policy actor read? | The boundary state of the regions its jurisdiction covers, its treasury and its levers; no event, no indicator, no live state | `spec-policy.md` section 5 |
| Who is a player ruler? | A `directed` policy and `DirectPolicy`, with a mandate and a cadence; or INT's setters with neither | `spec-policy.md` sections 8.4, 9 |

## Still open

1. Which recipes in the reference chain are trainee-eligible, and how large is their output? Production supplies the lever (`TraineeEligible`, `TraineeYield`, `TrainingPays`, `spec-production.md` section 8.3). The chain's actual flags and yields are calibration for `spec-reference-chain.md` and `spec-viability.md`; the reference chain must contain one eligible recipe above and one below break-even.
2. (Answered 2026-10-10.) Time in the traineeship queue counts in the application rule as a delay discount: `spec-skill-barriers.md` section 4.5.

3. (Answered 2026-10-10.) The shortest traineeship that shows the throttle is derived in `spec-skill-barriers.md` section 4.9.

4. (Answered 2026-10-10.) Does relief (`spec-estates.md` section 2.6) suppress migration too much, since it raises
   the ratio that triggers it? No, by construction: the migration trigger reads the household's earned ratio, which excludes relief (`spec-migration.md` section 5.4). Whether the rest of the loop is stable is for the scenarios MIG-S1 to MIG-S6 (S61 to S66).
5. (Answered 2026-10-10.) How does the price of hired carriage behave with a backhaul? The cost of a haul unit
   includes the whole round trip of the producing recipe, which double counts when cargo
   is found for the return (`spec-carriage.md` 5.1). The double count stands and is measured: `spec-merchants.md` section 10.
6. (Answered 2026-10-10.) How does a merchant estimate `expected_margin_per_haul_unit` for the
   `MerchantFeeShare` ceiling? From the rumoured remote price read as `min(Price, Mean)`, bounded by `SourceAge`, and the local report: `spec-merchants.md` section 5.6 (5.2).
7. (Answered 2026-10-10.) The tolerance rule is `max(25% of the excess, 200 bp)`, raised to twice the observed spread: `spec-skill-barriers.md` section 3.

8. (Answered 2026-10-10.) The Justman and Beek (2015) paper was read; the two elasticities are reconciled in `evidence.md` (the 0.4 snippet is the rounded ordinary-least-squares wage value; the instrumented wage elasticity is about 1.0). The Statute of Artificers limit of three apprentices stays unverified.

