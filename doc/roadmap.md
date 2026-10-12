# Roadmap: the specs to write

The goal is a reusable, adaptable economy simulation that works at the level of
a kingdom: several connected regions, each with its own market, not one market.
It takes ideas from bazaarBot (see `specs/bazaarbot-reference.md`) but does not
reproduce it: it need not be bit-equivalent, and where the original has defects
the port fixes them and records the difference. Nothing in the core assumes a
setting: currency, goods, roles and regions are data, and display names for
money are an optional formatter (`specs/spec-currency.md` section 4).

This file lists the design specs that cover the whole system, in the order to
write them. Which of them the first release contains is proposed in
[vision.md](vision.md). The index of the specs that exist, the area codes and the parameter
owners are in `specs/README.md`; this file is the plan, that one is the record.

Status values: **exists** (file written), **planned** (not written; the area
code is reserved in `specs/README.md`).

## Layers

Each spec has a layer ([ADR 0001](adr/0001-macro-economy-as-replaceable-layers.md)):

| Layer | Meaning |
|-------|---------|
| core | Always present; no opinion about the setting |
| module | Optional; off unless configured; built from configuration |
| extension | A consumer implements an interface to supply or replace it |
| fixture | A test fixture or acceptance scenario, not part of the library API |
| reference | Background material with no requirements |

A spec marked "core, module" has a core rule and optional parts; its text says
which is which. Opinions (calibrated values, a setting's content) live in
configuration packs, not in a layer of the engine.

## Existing specs

| Spec | Code | Layer | Status | Note |
|------|------|-------|--------|------|
| `specs/spec-currency.md` | CUR | core | exists | Minor unit, optional denomination formatter with an RPG preset, the reference basket |
| `specs/spec-ledger.md` | LED | core | exists | Accounts and signed boundaries, the transfer primitive and the reason registry, lots, reservations, flow totals, conservation and sectoral identities, balance sheets, claims, checkpoint state. Has a Requirements section |
| `specs/spec-goods.md` | GDS | core | exists | Commodity attributes as data: unit and its choice, mass and bulk with loads and chargeable size, shelf life, durability, classes, raw goods, reference price. Has a Requirements section |
| `specs/spec-config.md` | CFG | core | exists | The framework only, written ahead of the specs it needs: shape, bounds, defaults, the proposed file format and layering, validation machinery and structural rules, the configuration hash, and the parameter registry. Its field catalogue grows with each spec. Has a Requirements section |
| `specs/spec-calendar.md` | CAL | core, module | exists | The structure of time (Earth by default, any planet by configuration), the derived `TicksPerYear`, the moment of a tick, and an optional season cycle. Has a Requirements section |
| `specs/spec-scale.md` | SCL | core | exists | Weights, deriving agents from a census at a scale with a floor, the contract other specs take on, the cost model and a provisional budget. Has a Requirements section |
| `specs/spec-clearing.md` | CLR | core | exists | The double auction: admission and reservation, crossing check, tie-break, half-to-even midpoint, published price in thin markets, the `Clearing` contract. Has a Requirements section |
| `specs/spec-production.md` | PRO | core | exists | Recipes as data (several inputs and outputs, cost shares, byproducts, discard), multi-tick jobs, required tools and wear, yield by region and season (the seam to RES), trainee-eligible recipes, the cost anchor for BEL. Has a Requirements section |
| `specs/spec-beliefs.md` | BEL | core | exists | Price beliefs (centre and width, bounds, asynchronous bounded update, price window, seeding, warm-start hook). Has a Requirements section |
| `specs/spec-agents.md` | AGT | core | exists | Households, upkeep, the Strategy extension (three methods), the default strategy with a budget stack and saving rule, no inventory capacity, cohorts, the directed external agent. Has a Requirements section |
| `specs/spec-stability.md` | STB | core | exists | The feedback-loop register: entry format, 142 loops, the catalogue of 98 dampers, three numeric analyses, failure regimes, scenarios, the checklist for authors. Has a Requirements section; grows with every spec |
| `specs/spec-viability.md` | VIA | core | exists | Viability (sourced and runnable, productive cycles, labour sufficient, storable), the steady state by an exact linear programme and its dual, the warm-start table, regions and lanes. Has a Requirements section |
| `specs/spec-spawner.md` | SPN | core | exists | The spawner extension, bankruptcy by privation, the failure ladder, replacement and the damped entry rule, pool-funded entrants, the hook for POP. Has a Requirements section |
| `specs/spec-demand.md` | DEM | module | exists | Wants beyond upkeep as data, roles as classes, the price schedule, the ladder, the closed-economy ratio above 1. Has a Requirements section |
| `specs/spec-reference-chain.md` | REF | fixture | exists | The standard fixture as TOML packs, the variants, the scenarios S01 to S83 (S77 and S78 are harness tests that REF does not build) and A1 to A10, the invariants, the golden rendering. Has a Requirements section |
| `specs/spec-welfare-and-wages.md` | WEL | module | exists | Welfare ratio, wages, target-income rule |
| `specs/spec-skill-barriers.md` | SKL | module | exists | Tools and traineeship |
| `specs/spec-estates.md` | EST | core, module | exists | Estates, regional pool, relief |
| `specs/spec-carriage.md` | CAR | module | exists | Carrier pay: the fee is the market price of haul units (N-076, decided by `spec-transport.md`); `FR-CAR-001` and `FR-CAR-002` moved to `spec-merchants.md` 9.9 and its open items 5 and 6 are answered there |
| `specs/spec-regions.md` | REG | core | exists | The region as one settlement and market, the directed graph with integer distances and deterministic queries, one override mechanism, the relocation of an account and its hook for goods in transit, the edge status. Has a Requirements section |
| `specs/spec-money-supply.md` | MON | core, module | exists | Money flows in data (grants and levies on a class or a pool), declared reasons, the money supply and its identities, the closed economy and its liquidity floor, the primitives and rules reserved for EXT (EXT took the pass-through counterparty). Has a Requirements section |
| `specs/spec-population.md` | POP | module | exists | Births and deaths as household formation and dissolution driven by the welfare ratio, the hooks of the spawner, the stake, the Malthusian fixed point (the fixed factor comes from RES), collapse and recovery. Has a Requirements section |
| `specs/spec-transport.md` | TRN | module | exists | Modes as data, travel time from a REG distance, the shipment and goods in transit, the carriage unit as a produced good (N-076), hooks for RSK and CAP. Has a Requirements section |
| `specs/spec-resources.md` | RES | module | exists | Resources as data (not commodities), endowments per region with a stock, a capacity and three regeneration shapes, the abundance curve, the seam to production and the proportional ration of a scarce stock, depletion and recovery, the owner of the yield-table values (core data), the fixed factor of the population loop, the sourcing of raw goods, the sustainable draw for the solver. Has a Requirements section |
| `specs/spec-information.md` | INF | module | exists | What an agent knows of other regions: its own market report for free and delayed, blurred rumours of remote prices for informed roles, from a price record per source market. Has a Requirements section |
| `specs/spec-external-sector.md` | EXT | module | exists | The outside world as an agentless outside node, its lines (price path, capacity, windows, caps), trade through pass-through counterparties with an Outside account swept to zero each tick and six built-in ledger reasons, the balance of trade, the sourcing of raw goods by imports. Has a Requirements section |
| `specs/spec-subsistence.md` | SUB | module | exists | Household production for own use as the floor under market failure: own-use recipes as ordinary source recipes named on a list, the bound that keeps the market preferred, the mode a household enters and leaves, the default `Work`, the receipts in the welfare ratio and out of the observed wage, the commons. Has a Requirements section |
| `specs/spec-migration.md` | MIG | module | exists | Whether, when and where a household moves: a rule on the household's own earned ratio and measured wage and on rumoured prices only, a staggered review, an estimate of the income at each destination, a margin, a timed journey, caps on outflow and on the households on the way, no money or goods created. Has a Requirements section |
| `specs/spec-merchants.md` | MER | module | exists | The merchant strategy (margin estimation from rumours, travel or hire, the lane lock and the lot cap), the consignee and the delivery contract (offers and takes matched in a first-fit round, escrow, settlement on arrival); restates `FR-CAR-001` and `FR-CAR-002`. Has a Requirements section |
| `specs/spec-world.md` | WLD | core | exists | The one authoritative tick (eight phases and their positions), the random stream and its draw table, construction and the agent ID space, the options and the errors of `Step`, the between-ticks seam, cadences, order biases, what the world hands to OBS, CHK, INT, IND, STY and EXP. Has a Requirements section |
| `specs/spec-observability.md` | OBS | core | exists | The event catalogue with fields and positions, the snapshot, the metric catalogue and its feed, logs and keys, the cost of observing, and how all of it is tested. Has a Requirements section |
| `specs/spec-checkpoint.md` | CHK | core | exists | The checkpoint: the inventory of saved state, one canonical binary encoding with a version and the configuration hash, validated restore, extension state, bit-identity of a resumed run, compatibility and compaction. Has a Requirements section |
| `specs/spec-integration.md` | INT | core | exists | The minimal version 1 surface (ADR 0003): the player and other external agents, the slots set between ticks (shocks), queries, errors and guards. Has a Requirements section |
| `specs/spec-indicators.md` | IND | core | exists | Derived measures as pure functions of snapshots and events: GDP by three approaches, price indices, inequality, welfare distribution, windows, in fixed point. Has a Requirements section |
| `specs/spec-stylised-facts.md` | STY | core | exists | The testable macro facts and their bands (priors, not validated), the failure regimes and their detectors, acceptance versus report, how a fact becomes a test. Has a Requirements section |
| `specs/spec-experiments.md` | EXP | core | exists | The harness for sweeps and calibration runs: experiments as data, deterministic designs and seeds, branching from a checkpoint, paired runs and when they are valid, fixed-point results. Has a Requirements section |
| `specs/spec-calibration.md` | CLB | core | exists | The registry of tuning knobs with owners, ranges and expected effects, and the procedure for calibrating a setting against the stylised facts using sweeps. Has a Requirements section |
| `specs/evidence.md` | none | reference | exists | Sources behind the numbers |
| `specs/bazaarbot-reference.md` | none | reference | exists | What bazaarBot does and where its defects are |

Every existing spec now has a Requirements section (as of 2026-10-10). RES, INF and EXT, and then SUB, MIG and MER, were written on 2026-10-10 and, like every spec before them, are drafts: nothing in them has been run on the library. The tier 3 specs WLD, OBS, CHK, INT, IND, STY, EXP and CLB were written on 2026-10-11, each reviewed against the specs around it; they are drafts too, and the bands in STY and the ranges in CLB are priors, not results. The edits that each new spec owes to the other existing specs are part of the work below. For the eight tier 3 specs they were written in each spec's own edits table and were applied on 2026-10-11 to the other specs, the notes, the README, the rules and the ADRs. The edits that CLB owes were applied last, once its review was closed, to the README, the rules, the notes, the indexes and `spec-config.md`, `spec-viability.md`, `spec-stability.md` and `spec-scale.md`; its asks to `spec-stylised-facts.md`, `spec-indicators.md` and `spec-observability.md` (`spec-calibration.md` 23) are still open; [notes.md](notes.md) tracks the earlier ones.

## Tier 1: the per-region core

What bazaarBot provides, written with the region as a parameter from the start,
so that a single region with no edges is the degenerate case.

| Spec | Code | Layer | Status | Covers | Needs |
|------|------|-------|--------|--------|-------|
| `spec-goods.md` | GDS | core | exists | Commodity attributes as data: unit, divisibility, weight and bulk, shelf life, durability, commodity class | none |
| `spec-ledger.md` | LED | core | exists | Lots, spoilage, conservation of goods and money, checked arithmetic; accounts, boundary accounts, and transfers that carry a reason code, so every faucet and sink is named. Stability duties: double-entry balance sheets per agent and tests of the sectoral balance identities, not only conservation | GDS |
| `spec-production.md` | PRO | core | exists | Recipes, multi-tick duration, byproducts, required tools, trainee-eligible recipes | GDS, LED |
| `spec-clearing.md` | CLR | core | exists | The per-region, per-commodity double auction; the two bazaarBot defects to decide (matching when the top bid is below the top ask, and no budget check). Stability duties: a hard budget and limit-price rule (an agent never bids above its cash or sells below its limit), what price exists in a thin or empty market, and unbiased rounding | LED |
| `spec-beliefs.md` | BEL | core | exists | Price beliefs and update rules in fixed point. Stability duties: prices anchored to cost with bounds, adjustment driven by the inventory-to-demand ratio, asynchronous adjustment (each agent adjusts with a probability) and bounded step sizes | CLR |
| `spec-agents.md` | AGT | core | exists | Households, order generation, inventory targets, upkeep, the `Strategy` extension. Stability duties: a hard liquidity constraint and a consumption rule that saves when wealth is low | LED, PRO, CLR, BEL |
| `spec-spawner.md` | SPN | core | exists | Bankruptcy, replacement, the entry rule. Stability duties: damped occupation changes (entry limits, smoothing, hysteresis) and a failure ladder: change role, relief, then death | AGT, EST |
| `spec-scale.md` | SCL | core | exists | Population scale: how many households a region holds, aggregated populations for distant regions, the cost budget. Decide before `spec-agents.md`, because it can change the agent model | GDS |
| `spec-demand.md` | DEM | module | exists | Demand beyond subsistence: luxuries, classes, tastes, how demand grows with income | AGT |
| `spec-stability.md` | STB | core | exists | The feedback-loop register: every feedback loop in the design with its sign, delay, damper and test, and the catalogue of dampers (bounds, floors, asynchronous adjustment, entry limits). A cross-cutting spec that grows with every other spec. First entries: the target-income labour loop (supply falls as the wage rises), the occupation-choice cobweb, and production, travel and traineeship delays | none |
| `spec-viability.md` | VIA | core | exists | Whether a chain can sustain itself: each cycle yields a surplus, and a household's labour can produce its upkeep. A steady-state solver for calibration and for a warm start, so a run begins near equilibrium instead of from random beliefs | GDS, PRO, WEL |
| `spec-reference-chain.md` | REF | fixture | exists | The standard test fixture and the acceptance scenarios: one region as a bazaarBot-style chain, two-region arbitrage, starvation collapse | all of tier 1 |

`spec-clearing.md` was written first: it settles the rounding rule, the
tie-break and the two defects, and everything else depends on its answers.

Start `spec-stability.md` and `spec-viability.md` at the same time and keep
them open while the others are written: every spec adds its feedback loops to
the first, and the second decides whether a chain is worth simulating at all.
The research behind them is summarised in "Stability" below.

### Tier 1 dependencies

Each spec sits below everything it needs, taken from the "Needs" column above.
An asterisk marks a spec that already exists.

```
 level 0   GDS goods            STB stability (needs nothing; grows with every spec)
            |
            +--------------------------+
            v                          v
 level 1   LED ledger                SCL scale   (decide before AGT: it can change the agent model)
            |
            +------------+
            v            v
 level 2   CLR clearing   PRO production (needs GDS, LED)
            |            |
            v            |        WEL* welfare and wages
 level 3   BEL beliefs   |            |
            |            +-----+------+
            |                  v
            |             VIA viability (needs GDS, PRO, WEL)
            v
 level 4   AGT agents (needs LED, PRO, CLR, BEL)
            |
            +----------------+
            v                v
 level 5   SPN spawner       DEM demand
           (needs AGT, EST*)
            |                |
            +-------+--------+
                    v
 level 6   REF reference chain (needs all of tier 1)
```

Dependency order and write order differ in one place. CLR needs only LED, but it
is written first among the behaviour specs (above), because its rounding rule,
tie-break and two defects settle questions that BEL and AGT inherit. Write LED
as far as CLR needs it, then CLR, and finish LED afterwards. STB and VIA stay open
throughout.

Tier 2 hangs from this picture in four groups: the region graph (REG) and what it
carries (TRN, INF, MER, EXT); the people (POP, MIG, SUB, EMP); the land and
capital (RES, LND, CAP, STO); and the institutions that sit above them (POL, SET,
ORG). Tier 3 needs tiers 1 and 2. Within it the order is WLD, then OBS and CHK,
then INT, then IND, then STY and EXP, then CLB; CFG needs every other spec. The "Needs"
column of each table is the authority; this picture is only a reading aid.

## Tier 2: the realm

What bazaarBot lacks.

| Spec | Code | Layer | Status | Covers | Needs |
|------|------|-------|--------|--------|-------|
| `spec-regions.md` | REG | core | exists | The region graph, edge distance, per-region parameters | tier 1 |
| `spec-settlements.md` | SET | module | exists | Settlement size: classes derived from the households in a settlement by configured thresholds (class names are configuration; hamlet, village, town and city are only examples), and what each size unlocks or limits: a market, warehouses, associations, buildings, defence, how many carriers and merchants it can support. Settings that need no sizes use a single class. Draft, reviewed and re-verified. Has a Requirements section | REG, POP || `spec-polity.md` | POL | module | planned | Who rules: realms, provinces and the polity as a whole as nested jurisdictions over regions, borders, which laws and levies apply where | REG |
| `spec-subsistence.md` | SUB | module | exists | Household production outside the market (growing or making basics for oneself) as the floor under market failure, so a failed market does not mean starvation. Own-use recipes are ordinary source recipes on a list; land and rent (LND) are later and the spec leaves a hook. Has a Requirements section | PRO, RES, AGT, WEL |
| `spec-resources.md` | RES | module | exists | Regional endowments such as farmland and mines: yield, depletion, regeneration, the ration of a scarce stock, the owner of the yield-table values and the fixed factor of the population loop. Has a Requirements section | REG, PRO |
| `spec-calendar.md` | CAL | core, module | exists | The structure of time, Earth by default (hours in a day, days in a week, months, ticks in a day or days in a tick), the derived `TicksPerYear`, and an optional cycle of seasons. Has a Requirements section | none |
| `spec-population.md` | POP | module | exists | Births, deaths, starvation, regional collapse. Stability duties: birth and death rates that depend on welfare so a stable population exists (the Malthusian fixed point) | AGT, WEL, EST |
| `spec-migration.md` | MIG | module | exists | Migration triggers on the household's own earned ratio, destination choice from rumoured prices, the timed journey, interaction with relief (relief does not enter the trigger). Has a Requirements section | REG, INF, WEL, AGT, SPN |
| `spec-information.md` | INF | module | exists | Free local prices; delayed, approximate rumours of remote ones. Has a Requirements section | REG, BEL |
| `spec-transport.md` | TRN | module | exists | Travel time, carrying capacity (backpack, horse, cart), goods in transit, spoilage in transit | REG, GDS, LED |
| `spec-storage.md` | STO | module | exists | Warehousing, capacity limits, storage cost. Draft, reviewed once, revisions pending. Has a Requirements section | LED, GDS || `spec-risk.md` | RSK | module | planned | Loss and conflict: banditry, raids, shipwreck, disasters, loss in transit and in storage, embargo | TRN, STO |
| `spec-land.md` | LND | module | exists | Who owns the endowments of `spec-resources.md`: tenure, rent, tenancy, tenure obligations between landholder and tenant. Draft, not yet reviewed. Has a Requirements section | RES || `spec-institutions.md` | ORG | module | planned | Associations (a guild is one example), market charters, monopolies, price controls and usury laws; how associations gate the traineeship and tool barriers of `spec-skill-barriers.md` | POL, SKL, SET |
| `spec-infrastructure.md` | CAP | module | planned | Fixed capital beyond personal tools: mills, smithies, ships, roads, bridges, granaries. Who builds them, who pays, depreciation and maintenance, and how a road changes an edge's travel time | REG, PRO, RES, SET |
| `spec-employment.md` | EMP | module | exists | Hired labour: workshops, farm labour, wage contracts, and how they interact with the target-income rule. Draft, reviewed once, revisions pending. Has a Requirements section | AGT, WEL || `spec-merchants.md` | MER | module | exists | Margin estimation, the decision to travel or hire a carrier, delivery contracts. Has a Requirements section | INF, TRN, CAR, AGT |
| `spec-external-sector.md` | EXT | module | exists | The outside world: what lies beyond the realm, how it buys and sells (fixed or drifting prices, volumes), and how money and goods flow in and out. The valve that lets a setting import, export and gain or lose money. Has a Requirements section | REG, MON |
| `spec-saving-and-investment.md` | SAV | module | planned | Why a household holds money back (hoarding, precaution), what turns savings into capital, and how an agent decides to build or expand | DEM, CAP, MON |
| `spec-technology.md` | TEC | module | planned | What can raise productivity (learning by doing, discovery, diffusion) and what is exogenous; a setting without growth switches it off | PRO, CAP, SKL |

## Tier 3: the platform

Money supply, indicators, policy, stylised facts, calibration and experiments are the macro layer and its tuning loop (see "Macro is measured, never set" below).

| Spec | Code | Layer | Status | Covers | Needs |
|------|------|-------|--------|--------|-------|
| `spec-world.md` | WLD | core | exists | Tick phases, scheduler, determinism, options. Stability duties: decisions at different rates (daily trade; slower price, hiring, production and migration decisions), the rates being data. Has a Requirements section | tiers 1 and 2 |
| `spec-config.md` | CFG | core | exists | Config format, validation rules, bounds. Stability duty: reject a chain that fails the viability check of `spec-viability.md` | all |
| `spec-observability.md` | OBS | core | exists | Events, snapshot contents, metrics; a faucet-and-sink event that carries the reason. Has a Requirements section | WLD |
| `spec-checkpoint.md` | CHK | core | exists | The checkpoint: complete world state as plain data with a format version and a configuration hash; validated restore; one canonical binary encoding; explicit extension state; every other spec lists the state it contributes. A resumed run is bit-identical to an uninterrupted one (`adr/0002-persistence-by-checkpoint.md`) | WLD |
| `spec-indicators.md` | IND | core | exists | Derived measures: GDP, price index, inequality, welfare distribution, used by the acceptance checks. Built on generic aggregators over commodity classes and account sets; the standard indicators are presets, and a consumer can define its own; national-accounts-style measures (value added, production and destruction value, price indices per commodity class, which needs the commodity class tag in `spec-goods.md`) | OBS, WLD, LED, GDS, SCL, INT |
| `spec-integration.md` | INT | core | exists | What a game needs: queries, player agents, external shocks (a failed harvest, a closed road, a plague) | WLD |
| `spec-money-supply.md` | MON | core | exists | Where money enters (mints, mines) and leaves; a closed economy deflates. Every creation or destruction of money has a named reason and an event. Core, because the macro layer depends on it | LED, CUR |
| `spec-fiscal.md` | FIS | module | planned | Taxes, tolls, tariffs, a ruler's treasury, granaries. An optional extension; the core does not depend on it | MON, REG, POL |
| `spec-policy.md` | PLC | extension | planned | How a ruler or other policy actor chooses its settings over time (tax rates, minting, granary release, price controls): a fixed schedule, a rule that reacts to measures, or a player. Open: whether it reuses the `Strategy` extension point or needs its own (architecture.md 4.3) | POL, FIS, MON, IND |
| `spec-stylised-facts.md` | STY | core | exists | The testable macro properties of a healthy run and their acceptance bands (for example the pre-industrial wage and population cycle, price convergence between connected regions within transport cost, response to a harvest shock, famine in a closed region), plus the failure regimes: runaway inflation, hoarding, collapse, stagnation. Includes the stability checks: bounded prices, no cobweb blow-up, and a warm start that stays near the steady state | IND, STB, REF, VIA, WLD, INT |
| `spec-calibration.md` | CLB | core | exists | The registry of tuning knobs with classes, owners, kinds of range and expected effects (it adds no row to the parameter-owner table in `specs/README.md`: the registry is a section of its spec and holds no default or bound), and the procedure for calibrating a setting against `spec-stylised-facts.md` | STY, EXP, IND, STB, VIA, CFG, REF, WLD |
| `spec-experiments.md` | EXP | core | exists | A harness for parameter sweeps and calibration runs, such as `TargetSpread` experiments. Core, because tuning depends on it | WLD, CHK, INT, IND |

`spec-fiscal.md` and `spec-policy.md` are deferred past the first release, and `spec-integration.md` is in
version 1 only as shocks and a player agent (ADR 0003). `spec-money-supply.md`, `spec-indicators.md` and
`spec-experiments.md` are core: the macro layer and tuning depend on them.

## Extensions

Optional layers on top of the core. The core must not depend on them, but must
leave room for them: the ledger can hold an obligation between two parties
(`spec-ledger.md`), and risk events exist before insurance does.

| Spec | Code | Layer | Status | Covers | Needs |
|------|------|-------|--------|--------|-------|
| `spec-credit.md` | CRD | module | planned | Loans, interest, default, moneylenders and working capital; seasonal credit for farmers | LED, MON |
| `spec-insurance.md` | INS | module | planned | Insuring cargo and goods against the losses of `spec-risk.md` | CRD, RSK |

## Macro is measured, never set

Macroeconomic behaviour (price levels, cycles, unemployment, inequality, trade
balances) is meant to emerge from the micro systems. The engine holds no
aggregate state that agents react to: no inflation rate, GDP target or macro
equation drives anything. Aggregates are measured (`spec-indicators.md`) and
compared with stated expectations (`spec-stylised-facts.md`). Where a macro
item cannot emerge without a structural piece, that piece is a spec:

| Macro item | Specs |
|------------|-------|
| Money creation and destruction | MON, EXT |
| Saving and investment | SAV, CAP, CRD |
| The wage and demand loop | DEM, EMP |
| Unemployment | EMP |
| Long-run growth | TEC |
| Government | FIS, POL, PLC |

Tuning is a loop: measure (IND), compare with the expected properties (STY),
adjust a knob (CLB), run again (EXP). In-run adjustment of a faucet or a tax by
a rule is a policy actor (PLC), not engine logic.

## Stability

The aim is a stable and coherent economy, not only a feature list. The review
of other agent-based models (Lengnick's baseline model, stock-flow consistent
models, zero-intelligence traders, cobweb models, game post-mortems) found that
stability comes from a few kinds of mechanism, each of which has a home above:

| Kind | Where |
|------|-------|
| Every positive feedback loop named and damped | STB |
| Prices anchored to cost, adjusted by inventory ratios, asynchronously and in bounded steps | BEL, CLR |
| Hard budget and limit-price rules | CLR, AGT |
| Balance sheets and flow counterparts for every transfer | LED |
| A floor under market failure (home production, relief) | SUB, EST |
| A chain that can sustain itself, and a warm start | VIA |
| Demographic closure and damped occupation changes | POP, SPN |
| Decisions at different rates | WLD |

The sources are in `specs/evidence.md`. Details of Lengnick's rules were read
from pages 3 to 7 of the paper only; the other sources were read as summaries.

## Deliverables that are not specs

| Deliverable | Status | Note |
|-------------|--------|------|
| The RPG kingdom data pack: commodities, roles, recipes, regions, settlements and prices, as a configuration, with the RPG denomination preset | planned | Lives in `examples/`, not in the core. It is content, kept separate from the reference chain, which is a test fixture (`spec-reference-chain.md`). It proves the engine is setting-agnostic |
| A `NOTICE` file for third-party attribution | planned | Needed before any bazaarBot code or text is copied (MIT) |

## Work on existing specs

1. Done 2026-10-10: every existing spec has a Requirements section using its area code.
2. Move the open questions that belong to planned specs (README "Still open",
   items 1, 5 and 6) to the roadmap rows that own them: item 1 to PRO. Done
   2026-10-10: items 5 and 6 are answered by `spec-merchants.md`, and item 4 by
   `spec-migration.md`.
3. Add an unbiased rounding rule to `specs/spec-currency.md` section 2.3 (truncation
   toward zero biases prices over many operations), and record the stability
   duties above in the specs that own them when each is written.
4. When the first code is written, add the optional `denomination/` package to
   the layout in `rules/architecture.md` 2.6 and `AGENTS.md`
   (`specs/spec-currency.md` section 7).

## Decisions

| Question | Decision |
|----------|----------|
| Is bit-equivalence with bazaarBot a goal? | No. bazaarBot is inspiration; equivalence is behavioural, tested by the one-region scenario in `spec-reference-chain.md` |
| Does the core know what the money is called? | No. Denominations are an optional formatter with an RPG preset |
| Is the model single-market? | No. A realm of regions; every per-market spec says "per region" |
| Are stock markets in scope? | No. Equities do not fit a household-level model with no firms. Association shares or merchant partnerships may come later as an extension |
| Are banking and insurance in scope? | As deferred extensions (`spec-credit.md`, `spec-insurance.md`). The core leaves room for obligations and risk but implements neither |
| Is there a notion of settlement size? | Yes, as `spec-settlements.md`. A size class comes from population and gates what a settlement can have; a setting without sizes uses one class |
| Is the macro economy set or measured? | Measured only. No aggregate state is an input to agents. Where a macro item cannot emerge, a structural spec provides it (section "Macro is measured, never set") |
| Are the money supply and the experiment harness core? | Yes. The macro layer and tuning depend on them |
| How is the macro layer shaped? | Core primitives, optional modules off by default, a consumer hook, stable default constants, opinions in packs. See `adr/0001-macro-economy-as-replaceable-layers.md` |
| How is state persisted? | The library never persists. It produces a complete checkpoint of plain data with one canonical binary encoding; the consumer stores it. See `adr/0002-persistence-by-checkpoint.md` |
| Is stability a requirement? | Yes. Stability is a design requirement with its own specs (STB, VIA, SUB) and acceptance tests (STY), not an outcome we hope for. See "Stability" |
| Is a region one settlement or several? | One settlement per region, and a region is one market. A town and its hamlets are separate regions joined by short edges. `spec-settlements.md` (later) adds a size class to a region and does not split its market. Apply in `spec-regions.md` (N-026) |
| What is in version 1? | The staging of `vision.md` section 4: the slice, then version 1, then later; FIS and PLC later; INT only as shocks and a player agent. Accepted by the human | [ADR 0003](adr/0003-version-1-scope.md) |

## Open

1. (Answered 2026-10-10, ADR 0003.) FIS and PLC are later; INT is in version 1 only as shocks and a player agent. CHK is core (ADR 0002); SCL and DEM stay in the core.
2. (Answered 2026-10-10, ADR 0003.) A policy actor gets its own interface, written with `spec-policy.md`, because it returns settings and does not bid. Do not create it before PLC.
3. (Answered 2026-10-10, ADR 0003, N-026.) One settlement per region; a region is one market. Applied in `spec-regions.md` 3.1 and 3.4.
4. Stable defaults (ADR 0001, decision 5) cannot be claimed until `spec-reference-chain.md` and the acceptance bands in `spec-stylised-facts.md` exist. Both now exist as drafts, but no band has been checked against a run (the bands are priors and every default is a placeholder), so no spec states a default macro constant as stable. This is a standing constraint, not a question (N-027).
5. (Analysed 2026-10-10, N-028.) Labour supply under the target-income rule is unit-elastic, so a shortage corrects itself only if demand elasticity exceeds 1; a staple's is below 1 (`specs/spec-stability.md` 6.1, `specs/spec-agents.md` 11.6). The rule is kept; WEL adopted a floor on working ticks (`MinWorkShare`, `specs/spec-welfare-and-wages.md` 5.3), which sets the pinned price but does not remove the instability. Open until scenario S04 has run; DEM may supply elastic demand for non-staples, and SUB gives the households that cannot afford a staple an own-use recipe, which removes the shortage pin for them but is not a damper of the labour loop (`specs/spec-subsistence.md` 11).
6. (Done for AGT, 2026-10-10, N-029.) The targeted review before `spec-agents.md` is complete (Lengnick's household section, ASSUME's base strategy class, abcEconomics Household, Firm and Trade pages; see `specs/evidence.md`). Clark's own Malthusian source is now read for `spec-population.md` (chapter 2 and draft chapters 4 and 5; `specs/evidence.md`).
7. (Answered 2026-10-10, after RES was written.) The Malthusian loop of `spec-population.md` needs a fixed factor (land or another limit); `spec-resources.md` 8 provides it as an endowment that binds, and its stability depends on the shape of the endowment. The POP scenarios (S37 to S40) may now use RES instead of a test double. Caveat: nothing has run, so no scenario has shown the loop stable; the closed forms need the welfare ratio at a full draw, which is a property of a pack (N-109).

### Suggested next specs

Every version 1 spec is now written: WLD, OBS, CHK, INT (minimal: shocks and a player agent), IND, STY, EXP and CLB (2026-10-11, in the order of "Tier 3" above), after REG, MON, POP, TRN, RES, INF, EXT, SUB, MIG and MER (2026-10-10). The later-stage specs (STO, EMP, SET, LND, CAP, RSK, ORG, POL, FIS, PLC, SAV, TEC, CRD, INS) stay planned. What remains before code is the few asks of CLB to STY, IND and OBS (`spec-calibration.md` 23), then a first slice that runs the reference chain, which is what the placeholders and bands wait for.

## Updating this file

Update it in the same change that adds a spec, renames one, drops one, merges
two, or changes an area code's status. When a spec is written, change its
status to **exists** and make sure `specs/README.md` (the code table) and
`documentation_index.md` agree. A dropped or merged spec keeps its row, marked
as such, because area codes are never reused (`specs/README.md`).
