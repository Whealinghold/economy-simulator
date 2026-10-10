# Notes for future work

A register of things to take into account in work that has not been done yet:
constraints one spec places on another, decisions to revisit, traps found while
writing, and tasks that wait for a spec or for code. It is read **before writing
or changing a spec** (`AGENTS.md`), and it carries a read-receipt tag at the end
that an agent quotes when it starts that work.

It is not a second home for decisions or requirements. A decision lives in a spec
or an ADR, a requirement in a spec's Requirements section, and a spec's own open
questions stay in that spec. A note points at those and says *who needs to know,
and when*. If a note is only an open question of one spec, leave it there.

## How to use it

**Before writing or changing a spec.** Find the spec's three-letter area code in
the index below (the codes are in [specs/README.md](specs/README.md)). Read every
open note tagged with that code and every note tagged `ALL`. For each one, do one
of three things in your change, and say which in your summary:

- **apply** it (the spec now does what the note asks), then close the note;
- **reject** it, with the reason, then close the note;
- **leave it open**, if it is not yours to settle yet, and add what you learned.

**While working.** If you find something a future spec or a code change must take
into account, add a note in the same change. Do not leave it only in a chat or a
PR description.

**Rules for entries.**

- One note, one topic. Keep it to a few lines and link instead of copying.
- Tag it with the area codes of the specs that must read it. `ALL` is for every
  spec. `CODE` is for the first code change that touches the area. `EVD` is for
  the evidence file.
- Give a stable ID `N-NNN`. IDs are never reused. Never delete a note: close it,
  moving it to "Closed notes" with the resolution and where it was settled.
- Give the date it was raised (the format is `YYYY-MM-DD`) and where, so a reader
  can check it is still true. A note that names a section or a requirement may go
  stale when that changes: fix the pointer when you see it is wrong.
- Update the index in the same change, and replace the read-receipt tag at the
  end of the file with a fresh random value.

## Index by area code

| Area | Open notes |
|------|-----------|
| ALL | N-016, N-017, N-027, N-103 |
| AGT | N-043, N-067, N-081, N-087, N-101, N-111, N-117 |
| BEL | N-042, N-043 |
| CAL | N-030 |
| CAP | N-108 |
| CFG | N-016, N-061, N-075, N-093, N-102 |
| CHK | N-020, N-051 |
| CLB | N-015 |
| CLR | N-113 |
| CODE | N-019, N-089 |
| CRD | N-032, N-046, N-107 |
| DEM | N-080, N-089 |
| EMP | N-046 |
| EST | N-082, N-096, N-117, N-119 |
| EVD | N-018 |
| EXP | N-015 |
| EXT | N-108, N-112, N-113 |
| FIS | N-021, N-046, N-107 |
| GDS | N-031, N-039 |
| IND | N-013 |
| INF | N-110, N-112 |
| INT | N-022, N-031, N-036, N-051, N-079, N-090, N-091, N-116 |
| LED | N-021, N-107, N-117 |
| LND | N-046, N-104, N-108 |
| MER | N-120 |
| MIG | N-121 |
| MON | N-096, N-107 |
| OBS | N-012, N-031, N-035, N-040, N-045, N-065, N-067, N-072, N-081, N-087, N-088, N-094, N-109, N-111, N-115 |
| PLC | N-025, N-046 |
| POP | N-096, N-097, N-109 |
| PRO | N-030, N-031, N-039, N-043, N-090, N-118 |
| REF | N-027, N-030, N-089, N-092, N-114 |
| RSK | N-046 |
| SAV | N-046 |
| SCL | N-030 |
| SKL | N-039, N-071, N-090, N-110 |
| SPN | N-043, N-089, N-096, N-118 |
| STB | N-028, N-032, N-039, N-042, N-043, N-045, N-067, N-086, N-087, N-092, N-099, N-113 |
| STO | N-011, N-046, N-120 |
| STY | N-015, N-027, N-030, N-084, N-085, N-086, N-092, N-113 |
| SUB | N-121 |
| TRN | N-120 |
| VIA | N-015, N-061, N-071, N-080, N-085, N-093, N-109, N-114 |
| WEL | N-016, N-028, N-081, N-099, N-118, N-119 |
| WLD | N-009, N-031, N-036, N-046, N-051, N-056, N-074, N-082, N-111, N-117 |

## Open notes

### N-009 Decisions the world and tick phases must make

- **Areas:** WLD
- **Raised:** 2026-10-09, from CLR, LED, CFG and SCL.
- **Note:** `spec-world.md` fixes, and several specs already assume:
  - all orders of a tick are placed before any market clears, and markets clear in
    region then commodity order (`spec-clearing.md` 7.2);
  - estate pseudo-agents are created with their regions, before any household
    (`spec-estates.md` 2.7.6);
  - ageing runs at the end of the tick (`spec-ledger.md` 5.3);
  - `Step` surfaces `ErrInvalidTrade` and, with `WithAudit`, `ErrAuditFailed`
    (`spec-clearing.md` open question 5; `spec-ledger.md` 12.4);
  - a checkpoint is taken between ticks, with no reservation open
    (`spec-ledger.md` 6.5);
  - options that change results (such as `WithTargetSpread`) are part of the
    effective configuration and its hash (`spec-config.md` 8.1);
  - decisions run at different rates (roadmap, WLD row).
- **Update (2026-10-10, AGT):** AGT assumes the phase order of `spec-agents.md` 4: start (moment, cost anchors, relief and starting money, directive), production (continue and complete, then `Work`, then starts), `Trade` and orders, clearing, outcomes and `Revise`, consumption, entry and exit, end (ageing). Creation draws in the order target ratio then residue (3.5.2). VIA: the warm start is applied before the agents' creation draws (`spec-viability.md` 8.6 item 5).
- **Update (2026-10-10):** a construction phase precedes tick 0 and ends at the first `Step` (`spec-ledger.md` 5.2.4, N-074).
- **Update (2026-10-10):** AGT phase 6 includes `Privation` and want consumption; phase 7 is detailed in `spec-spawner.md` 10.2 (relief in phase 1 from what the previous tick's entry step left, `spec-estates.md` 2.6; see N-082).
- **Update (2026-10-10, REG):** WLD takes from `spec-regions.md`: estate accounts and pseudo-agents are created with the regions, in ascending `RegionID`, before any household (3.6.1; the order against the ledger's boundary accounts is WLD's and the ledger's); every iteration over regions is ascending `RegionID` (3.3.2); a relocation needs no reservation open, so it runs after settlement and before the next admission, and the recommended phase is entry and exit (6.1.2); `SetEdgeStatuses` is a between-ticks call (7.3).
- **Update (2026-10-10, MON):** the money flows run in phase 1 after relief and starting money and before production (`spec-money-supply.md` 5.1).
- **Update (2026-10-10, TRN):** transport assumes: arrivals in phase 1, first, after the moment and before relief and starting money, in shipment ID order; dispatch in phase 3 for each agent in ID order after `Trade` returns, in this order: discards, shipments, intents; an agent that is away is skipped in phases 2 and 3 (`spec-transport.md` 5.3, 5.5); `Inbound` is fixed at the start of phase 3.
- **Update (2026-10-10, POP):** in phase 7 the POP deaths run before the entry budget is taken and births come last (`spec-population.md` 4.8).
- **Update (2026-10-10, wave 2 integration):** the phase 1 order is now: the moment and the cost anchors; transport arrivals; relief and starting money; money flows; the directive. Phase 7: ladder reductions and removals, POP deaths, switches and applies, births last. WLD fixes both.
- **Update (2026-10-10, wave 2 integration):** the order of account creation at construction is now fixed by the specs together and WLD adopts it: the built-in boundary accounts, then the boundary accounts of the declared money-flow reasons in flow-ID order (`spec-ledger.md` FR-LED-052), then each region's estate account in ascending `RegionID` (`spec-regions.md` 3.6.1, FR-REG-005), then the agents. The agent IDs of the estate pseudo-agents are `0..n-1`, so a pseudo-agent's ID equals its `RegionID`.
- **Update (2026-10-10, RES):** RES assumes: phase 2 takes a snapshot of every stock after continuations and completions and before `Work`, calls `Work` for every free agent, rations the draws per endowment in (RegionID, ResourceID) order, then starts in the rotation order; phase 8 ends with regeneration of every endowment in the same order, then the events `EndowmentChanged`, before the optional audit. Shock slots act between ticks, before phase 1 (`spec-resources.md` 4.1, 5.2).
- **Update (2026-10-10, INF):** WLD takes from INF: in phase 3 the view of an informed agent is built, with its error draws, immediately before its `Trade`, agents in ID order; `RumoursServed` is emitted after the last plan of phase 3, before phase 4; in phase 5 the price record is appended after the price windows are pushed and before the first `Revise` (`spec-information.md` 7.1).
- **Update (2026-10-10, EXT):** EXT asks WLD for three positions (`spec-external-sector.md` 5): phase 1, after the money flows and before the directive, the walk draws (one per walking line, ascending node then commodity); phase 3, after the agents' plans and the estates' asks, the counterparties' top-ups and orders in ascending agent ID; phase 4, after every market has cleared and every reservation is settled or released and before outcomes, the sweep of every counterparty account. Agent IDs: the estates first, then the counterparties (ascending node, gate, buy before sell), then the households. Account creation: after the estate accounts, the Outside accounts in agent-ID order; the boundary accounts of the six reasons after those of the declared flows (money reasons first, then goods reasons by commodity).
- **Update (2026-10-10, wave 3 integration):** the positions asked by RES, INF and EXT, in one list with the earlier ones (no two claim the same slot; where two act in one phase the order is the one listed). Phase 1: the moment and the cost anchors; transport arrivals; relief and starting money; money flows; the outside's walk (EXT); the directive. Phase 2: continuations and completions; the snapshot of every endowment's stock and abundance; `Work` for every free agent; the ration of each endowment that is drawn on; the starts in the rotation order (RES, W2: a two-pass start). Phase 3: for each agent in ID order, the rumours of an informed agent's view are built (INF, with the error draws) immediately before its `Trade`, then its plan is executed; the estates' asks; the counterparties' top-ups and orders in ascending agent ID (EXT); then `RumoursServed`, before phase 4. Phase 4: every market clears; then the sweep of every counterparty account (EXT). Phase 5: price windows are pushed, then the price record is appended (INF), then `Revise`. Phases 6 and 7 as before. Phase 8: ageing and spoilage, flow totals, regeneration of every endowment, the events `EndowmentChanged`, then the optional audit (RES). Construction order: the built-in boundary accounts, the boundary accounts of the declared money flows, those of EXT's six reasons, each region's estate account, the Outside accounts, then the agents. WLD fixes all of it.
- **Update (2026-10-10, SUB):** SUB assumes in phase 2: continuations and completions, the snapshot of the endowments (RES), then the flag step of SUB for every eligible agent in agent-ID order, then `Work` for every free agent, the ration, the starts; the event `OwnUseChanged` follows the starts, in (RegionID, role) order (`spec-subsistence.md` 5.4, 14.1). SUB draws nothing.
- **Update (2026-10-10, MIG):** MIG asks WLD for two positions (`spec-migration.md` 10.1): in phase 1, the arrivals of the migration journeys, in ascending agent ID, after the arrivals of the transport module and before relief and starting money; in phase 7, one migration pass for the whole world, after the entry step (the spawner and the population module) of EVERY region has run, with its error draws (6.2.1, 10.2.1). An agent on a journey is passed to neither `Work` nor `Trade` (as an away agent).
- **Update (2026-10-10, MER):** MER asks: phase 1, after transport's arrivals and before relief and starting money, the settlement of contracts in contract ID order, and the reseed of an arriving traveller's beliefs; phase 3, after the last agent's plan, the contract round, then the events `ContractRound` and `MerchantsPlanned`, then the estates' asks and the counterparties' orders, then `RumoursServed` (`spec-merchants.md` 12.5).
- **Update (2026-10-10, wave 4 integration, X12):** the positions asked by SUB, MIG and MER, in one list with the earlier ones. Where two specs act at the same point of a phase the order is the one listed; WLD fixes all of it. Phase 1: the moment and the cost anchors; transport arrivals (`ShipmentArrived`) and, at a traveller's arrival, the reseed of its beliefs (MER 12.2); the settlement of delivery contracts whose shipment arrived, in contract ID order, after the `ShipmentArrived` events (MER); the arrivals of migration journeys in ascending agent ID (MIG; MER's settlement and MIG's arrivals claim the same slot after transport's arrivals, and the order here is settlement first, since a party to an open contract cannot be on a journey, `spec-migration.md` 7.2); relief and starting money; money flows; the outside's walk (EXT); the directive. Phase 2: continuations and completions; the snapshot of every endowment's stock and abundance (RES); the flag step of the subsistence module for every eligible agent in agent-ID order (SUB, a pure function, no draw); `Work` for every free agent (an agent that is away or on a journey is skipped); the ration of each endowment that is drawn on; the starts in the rotation order (RES, a two-pass start); then the event `OwnUseChanged` in (RegionID, role) order. Phase 3: for each agent in ID order, the rumours of an informed agent's view are built (INF, with the error draws) immediately before its `Trade`, then its plan is executed; after the last agent's plan the contract round (MER: admission, first-fit matching, formation, with its draws), then `ContractRound` and `MerchantsPlanned` in ascending `RegionID`; the estates' asks; the counterparties' top-ups and orders in ascending agent ID (EXT); then `RumoursServed`, before phase 4. Phase 4: every market clears; then the sweep of every counterparty account (EXT). Phase 5: price windows are pushed, then the price record is appended (INF), then `Revise`. Phase 6 as before. Phase 7: ladder reductions and removals, POP deaths, switches and applies, births last (the entry step of every region), then ONE migration pass for the world with its error draws in ascending agent ID (MIG), taking its counts once at its start, then `MigrationStarted` and `MigrationReviewed`. Phase 8: ageing and spoilage, flow totals, regeneration of every endowment, the events `EndowmentChanged`, then the optional audit (RES). Construction order as in the wave 3 entry.

### N-011 Transport, storage and carriage

- **Areas:** STO
- **Raised:** 2026-10-09, `spec-goods.md` section 3.
- **Note:** A vehicle's and a store's capacity is a pair, a mass capacity and a
  bulk capacity, and a load fits only if both are within it
  (`spec-goods.md` 3.2). Pricing uses the chargeable size (`BulkFactor`, 3.3, 3.4),
  which over-charges well-mixed cargo (3.5). Measure how much on the reference
  chain, and add a per-mode factor if the error matters. Goods keep aging in
  transit and in transit accounts; cargo lost is a `loss` sink
  (`spec-ledger.md` 7.2). Contract money is held in an Escrow account, not a
  reservation (`spec-ledger.md` 6.6). `spec-carriage.md` has no Requirements
  section yet, and its open items (`specs/README.md`, "Still open" 5 and 6) belong
  to the contract and merchant specs.
- **Update (2026-10-10, CAR):** `spec-carriage.md` now has Requirements (FR-CAR-001 to 013), restates its open items 5 and 6 with the reason they wait, and records a finding in 4.2 (see N-076).
- **Update (2026-10-10, TRN):** applied for TRN in `spec-transport.md`. A mode's capacity is a pair per carrying unit and a load fits only if both sums are within it (3.2). Transport does not use `BulkFactor` or the chargeable size: a hired load buys `n = max(ceil(mass / UnitMass), ceil(bulk / UnitBulk))` carrying units by its binding fill, which removes the over-charge of `spec-goods.md` 3.5 for transport and is the per-mode factor the note asked for (3.2.1, 8.3). Goods keep ageing in Transit accounts and a lot spoils whole from them (5.6); cargo lost is the reason `loss`, reserved for RSK (10.1). Still open for STO (the chargeable size for storage), CAR and MER (a bilateral contract and its escrow, `spec-transport.md` 8.4).
- **Update (2026-10-10, MER):** applied for MER in `spec-merchants.md` 9: contract money is held in an Escrow account by transfers, not in a reservation (9.6); the open items of CAR about the bilateral contract and the merchant's margin are settled there (9, 5.6, 10). Still open for STO (the chargeable size for storage).

### N-012 Events, metrics and log keys to define

- **Areas:** OBS
- **Raised:** 2026-10-09, collected from LED, SCL, CFG and EST.
- **Note:** `spec-observability.md` must define or accept:
  - `BoundaryFlow`, one aggregated event per region, reason and commodity per tick
    (`spec-ledger.md` 12.1);
  - `Weight` on `AgentSpawned` and `AgentBankrupted`, and agents and households
    reported separately in the snapshot (`spec-scale.md` 9);
  - the configuration hash on `WorldCreated` (`spec-config.md` 10.3);
  - `EstateOpened` and `ReliefPaid` (`spec-estates.md` 2.5, 2.6);
  - a household-count metric, `economysim.household.count`, as a change to the
    metric catalogue (`spec-scale.md` 9.4);
  - the log keys `path`, `rule` and `config_hash`, and the ledger reasons as the
    values of the `reason` key.
  Estate pseudo-agents must not count in `economysim.agent.count`.
- **Update (2026-10-10):** AGT adds `UpkeepShortfall` (one per region and role per tick, `spec-agents.md` 14.1); `AgentSpawned` gains `Role`, `StartMoney`, `External`; the removal event gains `Cause`; log keys `households` and `unfed` (14.4); a metric `economysim.household.unfed` is proposed (14.5). STB asks for counters on dampers that act silently (N-045).
- **Update (2026-10-10):** OBS also decides the name of the Snapshot field for the households held to work by the floor, and whether the construction bucket is a `BoundaryFlow` with `Tick` -1 (N-072).
- **Update (2026-10-10):** the events, keys and metrics of the written specs are now in the rules (`rules/observability-and-logging.md` 3.3.2, 4.3, 5.3). Still open for OBS: the Snapshot fields (N-065, N-081), `Tick` -1 for the construction bucket (N-072), the snapshot-fed gauges, and the remaining conflicts of the rules edit (N-088).
- **Update (2026-10-10, REG):** REG adds the events `AgentRelocated` and `EdgeStatusChanged`, the log keys `edges`, `components`, `from_region`, `to_region`, `distance` and `closed`, the metric `economysim.household.relocations` (attributes `economysim.region.from` and `.to`), snapshot fields for the edges with a non-default status and the component count, and a question for OBS: events carry the `RegionID` or the string ID (`spec-regions.md` 10, open question 4).
- **Update (2026-10-10, MON):** three events (`MoneyFlowLimited`, `MoneyFlowEnded`, `MoneyFlowScaled`), three log keys (`flow`, `due`, `applied`) and two proposed metrics (`economysim.money.flow`, `economysim.money.flow.unapplied`) in `spec-money-supply.md` 13; Snapshot fields for the money supply by sector, the money boundary balance per reason and each flow's state in 13.4 (names are OBS's).
- **Update (2026-10-10, POP):** POP adds the event `PopulationChanged` (one per region and role per tick, never when all counts are zero), the log keys `births` and `deaths`, and two proposed metrics (`economysim.household.births`, `economysim.household.deaths`) (`spec-population.md` 10).
- **Update (2026-10-10, TRN):** transport adds the events `ShipmentDispatched`, `ShipmentArrived` and `ShipmentRejected`, the log keys `shipment`, `mode`, `dest` and `modes`, and the metrics `economysim.transport.shipments`, `economysim.transport.in_transit` and `economysim.transport.cargo.spoiled` with a new metric attribute `economysim.mode` (`spec-transport.md` 13). Snapshot: the shipments in flight, the away agents and the non-default transit factors (13.5).
- **Update (2026-10-10, wave 2 integration):** the events, log keys and metrics of REG, MON, POP and TRN are now in `rules/observability-and-logging.md` 3.3.2, 4.3 and 5.3. Still open for OBS: the Snapshot fields (N-094), whether events carry the `RegionID` or the string ID, and the conflicts of N-088.
- **Update (2026-10-10, RES):** RES adds the events `EndowmentChanged` and `EndowmentAdjusted`, the log keys `resource`, `stock` and `short`, four proposed metrics (`economysim.endowment.drawn`, `.regenerated`, `.short`, `.stock`) and two new metric attributes `economysim.resource` and `economysim.region` (`spec-resources.md` 14). Whether events carry the string ID or the integer index is OBS's (as for REG).
- **Update (2026-10-10, INF):** INF adds the event `RumoursServed` (one per reader region per tick with an informed view), the log keys `rumours` and `lag`, and the metric `economysim.information.rumours` with a new attribute `economysim.outcome` (`offered`, `no_path`, `too_old`, `not_yet`) (`spec-information.md` 10).
- **Update (2026-10-10, EXT):** EXT adds the events `OutsideTraded`, `OutsideEnded` and `OutsideShocked`, the log keys `node`, `value`, `nodes`, `counterparties` and `drift`, and three metrics `economysim.external.units`, `.value` and `.price` with the attribute `economysim.node` (`spec-external-sector.md` 15). Snapshot fields in 15.4 (names are OBS's).
- **Update (2026-10-10, SUB):** SUB adds the event `OwnUseChanged` (one per region and role per tick, never when all counts are zero), the log key `own_use`, and two proposed metrics (`economysim.household.own_use`, a Gauge, and `economysim.own_use.units`, a Counter) with no new metric attribute (`spec-subsistence.md` 14).
- **Update (2026-10-10, MIG):** MIG adds the events `MigrationStarted` (per departure) and `MigrationReviewed` (one per region per tick in which any household was pushed or held back, any arrival was held or any journey ended by a removal), three proposed metrics (`economysim.migration.started`, `.limited`, `.on_journey`) and a new metric attribute `economysim.limit` with twelve constants (`spec-migration.md` 13). No new log key. Arrivals are REG's `AgentRelocated` with the cause `migration`.
- **Update (2026-10-10, MER):** MER adds the events `ContractFormed`, `ContractSettled`, `ContractRound` and `MerchantsPlanned`, the log keys `contract`, `offers`, `takes` and `merchants`, the metrics `economysim.contract.formed`, `.value`, `.escrow` and `economysim.merchant.decisions`, and a new metric attribute `economysim.decision` (`spec-merchants.md` 15). Snapshot: the contracts in flight and the money in escrow (15.4); names are OBS's.

### N-013 What indicators must not get wrong

- **Areas:** IND
- **Raised:** 2026-10-09, from SCL, LED and GDS.
- **Note:** Inequality within a cohort is zero by construction, so an inequality
  measure sees only differences between agents (`spec-scale.md` 5.1). Population
  indicators count households (the sum of weights), never agents, and exclude
  pseudo-agents (FR-SCL-015). Balance sheets value goods at a price the caller
  supplies and are measurement only (`spec-ledger.md` 9.2). Wear units are counted
  by commodity and could back a depreciation measure (`spec-ledger.md` 5.4.2, open
  question 7). Commodity classes are the grouping for price indices and value
  added (`spec-goods.md` 6.1).
- **Update (2026-10-10, wave 2 integration):** a report by region does not balance from flows alone: a household that relocates takes its money and goods with it, and the ledger's relocation totals are the reconciling item (`spec-regions.md` 6.4). Rebuild the count of households by region from `AgentSpawned`, `AgentBankrupted`, `WeightChanged`, `RoleChanged` and `AgentRelocated`. The money supply by sector and the boundary balance per reason (one per money flow) come from the Snapshot fields of N-094. A price index of goods that travel should say whether it is the local price or the delivered price, which includes the freight (`spec-transport.md` 15, TRN-A4).

### N-015 Placeholders to calibrate once something has run

- **Areas:** VIA, STY, CLB, EXP
- **Raised:** 2026-10-09.
- **Note:** These numbers are guesses, not results, and nothing has been run:
  `MinAgents` of 10 (`spec-scale.md` 5.2); the 10% scale-consistency tolerance
  (7.2); 200 000 agent-ticks a second (6.2); the bounds in `spec-config.md` 4.1;
  the validation time in NFR-CFG-001; the per-field bounds in `spec-goods.md` 8;
  the `TargetSpread` default of 0 (`spec-welfare-and-wages.md` 5.2). The Doran and
  Parberry paper is unread and bazaarBot has not been run, so "stable" has no
  reference yet (`bazaarbot-reference.md` section 12; `vision.md` section 6).
- **Update (2026-10-10):** the Doran and Parberry paper is read (journal version; use `ianparberry.com/pubs/econ.pdf`, the LARC URL 404s), so "stable" has a prior definition: the filter criteria of the paper's section VII, listed in `spec-beliefs.md` section 19. Still true: bazaarBot has not been run and no number has been validated. New placeholders: the 15 belief parameters of `spec-beliefs.md` section 11 (most importantly `CostFloor` 10 000, `BeliefAdjustProb` 5 000, `BeliefStep` 200 and the stock band 5 000 to 15 000), and `TraineeYield` 5 000 and the production bounds of `spec-production.md` 11.
- **Update (2026-10-10):** AGT placeholders (`spec-agents.md` 12): `CoverTicks` 3, `InputJobs` 1, `OutputCoverTicks` 1, `FavourFloor` 2 000, `ReserveTicks` 3, `SpendMaxShare` 5 000, `SpendHalfTicks` 10, `RestBand` 500, `ProduceHaltRatio` 30 000, `BasketTolerance` 1 000, `StartMoney` 10 000, `MaxUpkeepItems` 16, and the worked examples of section 16. VIA placeholders: `StockCoverTicks` 5, `MoneyBufferTicks` 10, `ResidualWarnBP` 1 000, `MaxViabilityPivots` 100 000, `MaxViabilityCells` 4 000 000 and the prototype numbers of `spec-viability.md` 15; the Go speed of the exact solver is not measured. STB adds only placeholders: every threshold of the failure regimes, the example values of its proposed parameters, and every number of its models.
- **Update (2026-10-10, round 3):** new placeholders: `CostFloor` 7 000, `BeliefEvidenceShare` 5 000, `WelfareWindow` 30, `WindowCycles` 2, `MinWorkShare` 4 000, `MaxWelfareWindow` 3 650; scale cost model constants 2 us, 1 us and 8; the rescaled example quantities of `spec-production.md` 12; the skill-barrier tolerance 200 bp and queue discount; and the bounds of `spec-config.md` 4.1 and `spec-estates.md` 3.
- **Update (2026-10-10, round 4 and 5):** new placeholders: every default of `spec-demand.md` 9, `spec-spawner.md` 11 (`EntryShare` 50, `ExitShare` 100, `Renewal` 10 000, `EntryPoolShare`), `spec-reference-chain.md` (every pack quantity, the censuses, seeds and run lengths) and the thresholds of `spec-stability.md` 7 and 11.
- **Update (2026-10-10, RES):** new placeholders: `HorizonYears` 50, `PlanShareBP` 6 000, `InitialBP` 10 000, the bounds `MaxResources` 256, `MaxEndowment` 10^15, `MaxRegenBP` 100 000, `MaxRegenScaleBP` and `MaxShockBP` 100 000, `MaxHorizonYears` 1 000, the warning thresholds 11 000 bp (`no_comparative_advantage`) and 2 000 bp a tick (`regen_per_tick_high`), every number of the worked examples (16), of the models (17) and of the five-region sizing (18).
- **Update (2026-10-10, INF):** new placeholders: `RumourErrorBP` 500, `RumourMaxAge` 60, the bounds `MaxInfoSpeed`, `MaxRumourErrorBP`, `MaxRumourAge`, `MaxRumourCells`, `MaxRumoursPerView`, the cost counts of `spec-information.md` 7.8 and every number of its models and examples.
- **Update (2026-10-10, SUB):** new placeholders: `EnterTicks` 2, `ExitTicks` 3, `RetryJobs` 10, the bounds `MaxOwnUseRecipes` 8, `MaxOwnUseTicks` 1 000 and `MaxRetryJobs` 1 000, every number of the worked examples (16) and of the model (11.2, 17).
- **Update (2026-10-10, MIG):** new placeholders: the nine defaults of `spec-migration.md` 11, the factor 3 of the warning `migration_flow_window`, and every number of the worked examples (15) and of the models (16).
- **Update (2026-10-10, MER):** new placeholders: `MaxOpen` 1 000, `LotUnits` 1, `FeeShareSpreadBP` 2 500, `MaxSourceAge` 2, `TimeCostBP` 10 000, `ConsigneeCutBP` 500, `TakeMarginBP` 1 000, the bounds `MaxLotUnits` 64, `MaxContracts` 1 000 000 and `MaxTimeCostBP` 100 000, the warning thresholds of 14.2 and every number of the models and examples.

### N-016 Keep the parameter registry true

- **Areas:** ALL, CFG, WEL
- **Raised:** 2026-10-09, `spec-config.md` 9.3.
- **Note:** Every new configuration field needs a row in the parameter-owner table
  (architecture.md 9.5, 9.7). Known gaps: the welfare window `T` (default 30,
  `spec-welfare-and-wages.md` 2.3.1) has no row; the tolerance for matching a
  region's `ReferenceBasket` to its reference prices (`spec-currency.md` 5.3) has no
  name or default; `spec-clearing.md` calls `MaxPrice` and `MaxQuantity`
  "configured" although they are bounds in `bounds.go` (`spec-config.md` 4.1).
  A role's default `ExpectedWorkTicks` of 250 becomes invalid if the calendar gives
  fewer ticks a year (`TicksPerYear` is derived, `spec-calendar.md` 3.2;
  `spec-config.md` open question 5). The calendar fields have rows in the
  parameter-owner table since 2026-10-10.
- **Update (2026-10-10):** production added nine rows and beliefs fifteen to the parameter-owner table; no new unnamed parameter. `YieldProfile` has two owners over time (PRO now, RES later); PRO is listed as owner until RES exists.
- **Update (2026-10-10):** AGT adds 17 rows and names `BasketTolerance`, closing one gap of the known list (the unnamed basket tolerance of `spec-currency.md` 5.3); VIA adds `WarmStart`. VIA's solve options (`StockCoverTicks`, `MoneyBufferTicks`, `ResidualWarnBP`, `MaxPivots`, `Lanes`) are not `Config` fields and are NOT in the parameter-owner table, because the registry test of testing.md 6.7 requires a field behind every row; they live in `spec-viability.md` 11. STB adds no field; its four proposals (`MinWorkTicks`, `ProdStopRatio`, `EntryShare`, `RoleHysteresis`) need rows if adopted.
- **Update (2026-10-10, round 3):** `spec-config.md` 3.7 lists every field and 9.2 says a row is a `Config` field; the parameter-owner table now has a row for each (generated from 3.7, plus `WelfareWindow`, `WindowCycles`, `MinWorkShare`, `BeliefEvidenceShare`), and the solve options of `spec-viability.md` 11 are deliberately not rows. Closed: the `BasketTolerance` name, the `MaxPrice` and `MaxQuantity` wording (they are bounds), `ExpectedWorkTicks` against a derived `TicksPerYear` (an error, FR-CFG-048). Remaining: a carriage `m_bp` has no name; a test must verify the table against the code once it exists.
- **Update (2026-10-10, round 4 and 5):** the parameter-owner table gained the demand and spawner fields (`World.Spawner`, `Role.Closed`) and a row for every field of `spec-config.md` 3.7, regenerated after the new modules. No unnamed parameter was found in AGT, WEL, DEM, SPN or REF. A test must still verify the table against the code once it exists.
- **Update (2026-10-10, REG):** REG adds `Edge.From`, `.To`, `.Distance` and `RegionPrice.Region`, `.Commodity`, `.Price` to the parameter-owner table, and the presentation-only fields `Region.Label` and `World.DistanceLabel` to the allow-list of `spec-config.md` 8.3. No unnamed parameter was found. REG adds no tunable with a default (a distance and a price have none; the default of the sections is "no edges, no overrides"). The edge status of `spec-regions.md` 7 is world state, not a field, and has no row.
- **Update (2026-10-10, MON):** the fields of `MoneyFlow` have rows in the parameter-owner table; no unnamed parameter; the runtime `Scale` is state and not a row.
- **Update (2026-10-10, POP):** POP adds nine rows (`BirthStake` has an explicit name); no gap found. The registry test is still to be written.
- **Update (2026-10-10, TRN):** TRN adds `Modes` and ten fields of a mode, each with a default or "required", a bound and an owner. No unnamed parameter; it removes the unnamed carriage margin `m_bp` of `spec-carriage.md` (the haul unit's price is anchored to the producing recipe's `UnitCost`, `spec-transport.md` 8.5).
- **Update (2026-10-10, RES):** RES adds the rows of the module, 11 fields of `Resource` and 5 of `Endowment` to the parameter-owner table and changes the owner of the yield rows and of `Draws`; no unnamed parameter was found. The registry test is still to be written.
- **Update (2026-10-10, INF):** INF adds five rows (`Modules.Information`, `InfoSpeed`, `RumourErrorBP`, `RumourMaxAge`, `Role.Informed`), each with a default or "required", a bound and an owner; no unnamed parameter.
- **Update (2026-10-10, EXT):** EXT adds its rows to the parameter-owner table; no unnamed parameter. The shock, drift, carry and cumulative totals are runtime state and not rows.
- **Update (2026-10-10, SUB):** SUB adds rows for the module, its default list, the role rows, the key and list of a row, and three tunables (`spec-subsistence.md` 12); no unnamed parameter was found. The registry test is still to be written.
- **Update (2026-10-10, MIG):** MIG adds rows for the module, nine tunables, the region-keyed rows and `Role.Fixed` (`spec-migration.md` 11); no unnamed parameter was found. The registry test is still to be written.
- **Update (2026-10-10, MER):** MER adds rows for `Modules.Merchants`, `MaxOpen` and the nine fields of the block `Role.Merchant` (`spec-merchants.md` 3.2, 13); `MerchantFeeShare` keeps its row. No unnamed parameter was found.

### N-017 Words and names

- **Areas:** ALL
- **Raised:** 2026-10-09.
- **Note:** Names are setting-neutral (`AGENTS.md`). Traps already hit: `Weight` is
  the households an agent stands for, so a commodity has `Mass`, not weight; a
  "day" in the specs means a tick and a "year" is `TicksPerYear` ticks
  (`spec-currency.md` 2.4, now derived from the calendar); a calendar day is a
  different unit, since a tick may be a part of a day or several days
  (`spec-calendar.md` 2.5); "consumer" means the library consumer (N-001); a "pack"
  is configuration, a "module" is an optional mechanism. Add new terms to the glossary
  list in `specs/README.md`, and to `doc.go` before code uses them.
- **Update (2026-10-10, REG):** terms fixed: region, edge, route (an edge), distance (a length, never a duration), component, edge status, relocation. "Settlement" names what a region is, not a type. Setting words appear only in the flagged examples of `spec-regions.md` 12.
- **Update (2026-10-10, POP):** a birth and a death are the formation and dissolution of a household, not of a person; "stake" is the money a born household brings, not an investment; "year" is `TicksPerYear` ticks.
- **Update (2026-10-10, TRN):** the agent that moves is the *traveller*; a *carrier* is the role of `spec-carriage.md` whose households make haul units. Example mode names (`foot`, `wagon`, `coach`, `crawler`, `lift`) are examples in two invented settings and are flagged so in the spec.
- **Update (2026-10-10, RES):** terms fixed: resource (not a commodity), endowment (a resource in a region), stock, capacity, regeneration, shape, refuge, abundance, plan share, ration, sustainable draw. Setting words (land, forest, ore, ice) appear only in the flagged examples of `spec-resources.md` 16 and 18. "Draw" now has a precise meaning.
- **Update (2026-10-10, INF):** terms fixed: rumour, price record, observation, vintage, lag (of a rumour), information speed, horizon, informed (role), source.
- **Update (2026-10-10, EXT):** terms fixed: outside, outside node, gate, line, side (of a line), mid, quote, margin, counterparty (buyer, seller), Outside (account kind), outside sector, import, export, balance of trade, list value, drift, shock, availability window, capacity, cap, sweep. Setting words appear only in the flagged examples of `spec-external-sector.md` 18.
- **Update (2026-10-10, SUB):** terms fixed: subsistence floor (the whole mechanism), own-use recipe, own-use mode (the flag `OwnUse`), covered set, cover (stock in ticks of upkeep), afford, secure ticks, retry (the market test), coverage (`Cov`), `FloorBP`, blocked. The word floor has six uses in the specs (work floor, cost floor, resolution floor, `MinWorkShare`'s floor, the floor of agents, this one): SUB says "subsistence floor" for the mechanism and "own-use" for its parts. Setting words (glean, grain, common, tray) appear only in the flagged examples of `spec-subsistence.md` 16.
- **Update (2026-10-10, MIG):** terms fixed: migration (this module's moves), migrant (an agent on a journey), journey, earned ratio (the welfare ratio without relief), settled, margin and width (of a move), horizon, estimate (of a destination), pushed (household), outflow cap, pipeline cap, provisioning. "Road" is not used (a journey). Setting words appear only in the flagged examples of `spec-migration.md` 15.
- **Update (2026-10-10, MER):** terms fixed: merchant, planned trip, lane (of a merchant), candidate, lot, lot cap, cargo set, trade inputs, fee share, bar, cautious revenue, break-even, lane lock, delivery offer, delivery take, payer, deliverer, round (of contracts), delivery contract, settlement. Setting words appear only in the flagged examples of `spec-merchants.md` 19.

### N-018 Unverified evidence

- **Areas:** EVD
- **Raised:** 2026-10-09.
- **Note:** Several facts behind defaults are unverified summaries
  (`specs/evidence.md`). The traineeship elasticities disagree and come from unopened
  snippets (`specs/README.md`, "Still open" 8). Do not set a default from such a
  fact without checking the source.

### N-019 Housekeeping for the first code change

- **Areas:** CODE
- **Raised:** 2026-10-09.
- **Note:** When the first code lands, in the same PR:
  - update the layout in architecture.md 2.6 and `AGENTS.md` for `config.go`,
    `defaults.go`, `bounds.go`, `validate.go`, `hash.go`, `commodity.go`, `calendar.go`, the
    optional `denomination/` package (`spec-currency.md` 7), `internal/ledger` and
    `internal/clearing`;
  - add the new terms to the glossary in `doc.go` and the new options
    (`WithAudit`, `WithTargetSpread`) to the options list;
  - add the log keys of N-012 to observability-and-logging.md 3.3.2, and any new
    event to its table 4.3;
  - record an ADR choosing the TOML library, with the justification of
    architecture.md 5.3 (`spec-config.md` 6.1), and check its strict decoding;
  - amend architecture.md 4.3 and 7.7 for the checkpoint and optional extension
    state, as ADR 0002 says, with a fresh read-receipt tag.
- **Update (2026-10-10):** the log keys and events of the written specs are in `rules/observability-and-logging.md`. The rest of this note is unchanged: the architecture.md 4.3 and 7.7 amendment for the optional extension state (ADR 0002) waits for the first code.
- **Update (2026-10-10, REG):** at the first code also add `internal/graph/` (the shortest-path algorithms over primitive slices) and `regions.go` to the layout of architecture.md 2.6 and `AGENTS.md`, and the REG terms to the glossary in `doc.go` (`Region` is used by every spec and is not in the vocabulary list of architecture.md 3.2).
- **Update (2026-10-10, wave 2 integration):** the four new specs name these homes, to add to the layout of architecture.md 2.6 and `AGENTS.md` when each is first needed: `internal/graph` (REG, shortest paths over primitive slices), `internal/moneyflow` (MON), `internal/population` (POP) and `internal/transit` (TRN), each with the root file of its concept and, for TRN, the root types `Mode`, `Shipment` and `ShipmentRequest`. Each also needs its terms in the `doc.go` glossary and its sentinels mapped at the root (N-075). The Python models of the four specs are archived in `doc/specs/models/{reg,mon,pop,trn}/`.
- **Update (2026-10-10, wave 3 integration):** the three new specs name these homes, to add to the layout of architecture.md 2.6 and `AGENTS.md` when each is first needed: `internal/endowment` (RES: regeneration, abundance, the ration, the key; stdlib only), `internal/curve` (piecewise-linear evaluation, shared with the population schedules), `resources.go` (RES), `internal/rumour` and `information.go` (INF: the lag, the distortion, the ring, over primitive slices) `internal/outside` and `external.go` (EXT: the mid, the walk, the quotes and the caps, over primitive slices; the account kind Outside and the six reasons are rows of `internal/ledger`, `spec-external-sector.md` 23). Each also needs its terms in the `doc.go` glossary and its sentinels mapped at the root (N-075). The Python models of the three specs are archived in `doc/specs/models/{res,inf,ext}/`.
- **Update (2026-10-10, SUB):** at the first code add `subsistence.go` and `internal/ownuse` (the flag step, `Cov`, the ramp, over primitive slices, standard library only) to the layout of architecture.md 2.6 and `AGENTS.md`, and the SUB terms to the `doc.go` glossary. The Python models are in `doc/specs/models/sub/`.
- **Update (2026-10-10, MIG):** at the first code add `migration.go` and `internal/migration` (the estimate, the journey time, the margin and its offset, the ranking, the admission against the caps and the partition of outcomes, over primitive slices, standard library only) to the layout of architecture.md 2.6 and `AGENTS.md`, and the MIG terms to the `doc.go` glossary. The Python models are in `doc/specs/models/mig/`.
- **Update (2026-10-10, MER):** MER names these homes, to add when each is first needed: `strategy/` (it now has a second member, so it is created with the first code: `merchant.go`, `options.go`), `internal/merchant` (the pure rules over primitive slices), `contract.go` (root). The Python models are in `doc/specs/models/mer/`.
- **Update (2026-10-10, wave 4 integration, X1 and X13):** the homes of the three wave 4 specs in one list: `subsistence.go` and `internal/ownuse` (SUB), `migration.go` and `internal/migration` (MIG), `strategy/` with `Merchant` registered as "merchant", `internal/merchant` and `contract.go` (MER). One shared helper: the Weyl offset of the agent ID, `(id * 2 654 435 769) mod 2^32`, which `spec-population.md` 3.8.3 defines and MIG and MER reuse as a fixed per-agent spread with no creation draw, lives in `internal/ordering` (the package of deterministic ordering helpers) with one test of its evenness; a new per-agent spread reuses it and does not draw at creation. Each spec's terms go to the `doc.go` glossary and its sentinels are mapped at the root (N-075).

### N-020 What each spec contributes to a checkpoint

- **Areas:** CHK
- **Raised:** 2026-10-09.
- **Note:** Every spec lists the state it adds (ADR 0002). Known so far: the ledger
  (`spec-ledger.md` 11) and the estate's listing prices (`spec-estates.md` 2.7.5).
  Not saved: reservations, flow totals and the derived expiry hints. The
  configuration hash includes result-affecting options (`spec-config.md` 8.1). Open:
  compaction of closed accounts (N-008), the extension-state interface, and the
  compatibility policy after 1.0 (ADR 0002, open points).
- **Update (2026-10-10, LED):** `Born` is signed in the saved state; the construction bucket and pending output are not in the ledger state (`spec-ledger.md` 11). The agent state AGT contributes is in `spec-agents.md` 20, the beliefs' in `spec-beliefs.md` 14.
- **Update (2026-10-10):** the spawner's state (`Privation`, `RoleSince`, the renewal carry, two diagnostic runs) is in `spec-spawner.md` 14; the demand module's carries are in `spec-demand.md` 12.
- **Update (2026-10-10, REG):** REG saves only the edges whose status is not the default (From, To, Closed, FactorBP), in `EdgeID` order (`spec-regions.md` 11). The graph tables, the resolved override arrays and the relocation totals are derived and not saved.
- **Update (2026-10-10, MON):** state in `spec-money-supply.md` 14 (per flow `Total`, `Scale`, `Ended`; carries per open account; pool carries per region).
- **Update (2026-10-10, POP):** POP contributes a death carry per agent, a birth carry per (region, role), the effective `BirthScale` and `DeathScale` per region when a shock set them, and two diagnostic runs (`spec-population.md` 11). Deaths close one account per removed agent and none for a partial reduction; with POP on the list of closed accounts grows by one slot per death in a weight-1 world (ledger open question 6).
- **Update (2026-10-10, TRN):** the transport module's state is the next shipment ID, the shipments in flight and the non-default transit factors (`spec-transport.md` 14); away flags and the free lists of Transit accounts are derived.
- **Update (2026-10-10, RES):** RES contributes per endowment `Stock`, the regeneration `Carry` and `ExhaustedRun`, and the non-default shock scales (`spec-resources.md` 15); effective values, the abundance snapshot, grants and keys are derived.
- **Update (2026-10-10, INF):** INF saves the price record: per source in ascending order the observations held, oldest first (`spec-information.md` 14). The lag table, sources and rumours are derived; the error needs no state.
- **Update (2026-10-10, EXT):** EXT saves per line the drift (with a walk) and the shock, per side the carry, the cumulative units and list value and the ended flag; the counterparty accounts are empty at a boundary and are not special state (`spec-external-sector.md` 16).
- **Update (2026-10-10, SUB):** SUB contributes per agent `OwnUse` and `OwnUseJobs`, and per (region, role) two diagnostic runs for its `Warn`s (`spec-subsistence.md` 15). The welfare ring gains two integers per tick (WEL).
- **Update (2026-10-10, MIG):** MIG contributes per agent on a journey its destination and arrival tick, and per region two diagnostic runs and a pending count of journeys ended by a removal (`spec-migration.md` 14). `RoleSince` is SPN's state. The welfare ring needs one more field per tick (relief).
- **Update (2026-10-10, MER):** MER saves the next contract ID and the contracts in flight in ID order; the merchant strategy has no state; the Escrow accounts are the ledger's (`spec-merchants.md` 16).

### N-021 Anything that creates or destroys money or goods

- **Areas:** FIS, LED
- **Raised:** 2026-10-09.
- **Note:** A new faucet or sink is a row in the ledger's reason registry, with its
  kind and owner (`spec-ledger.md` 7.2); there is no other way to change a total.
  The boundary balance per reason is the measure of how much has entered or left.
  Credit will use the ledger's claims list (`spec-ledger.md` 10). A commodity marked
  `Raw` needs a real source once resources, the external sector or faucets exist
  (`spec-goods.md` open question 5).
- **Update (2026-10-10, LED):** the reason registry gained `disposal` and the goods form of `initial`; `initial` is not construction-only (an external agent mid-run, `spec-ledger.md` 7.2.1).
- **Update (2026-10-10, REG):** REG adds no faucet, no sink and no ledger reason. A relocation moves nothing between accounts; the ledger keeps relocation totals as the reconciling item of a report by region (`spec-regions.md` 6.4).
- **Update (2026-10-10, MON):** applied for money. The reason registry is the built-in rows plus one row per configured money flow (reason = flow ID), closed for a run, so a new money faucet or sink is a registry row (`spec-money-supply.md` 4.5, 10.3); any counterparty whose balance is created or destroyed outside conservation must be a Boundary account (10.1, FR-MON-043). Still open for EXT, FIS and RES: the goods side (a `Raw` commodity's real source, imports and exports, goods faucets).
- **Update (2026-10-10, POP):** POP creates and destroys no money or goods; births and deaths are internal flows between the pool and agents.
- **Update (2026-10-10, RES):** RES applies its part in `spec-resources.md` 9: a raw commodity is endowment-sourced under a stated predicate (9.2: a source recipe with `Draws` on an endowment that can start at least once), validation takes the union with source recipes and EXT's imports, the warning `raw_source_exhaustible` flags a raw good whose only sources are finite stocks, and `endowment_runway` reports the run-down time. RES adds no faucet, no sink and no ledger reason (6.2.1).
- **Update (2026-10-10, INF):** INF adds no faucet, no sink and no ledger reason.
- **Update (2026-10-10, EXT):** applied for EXT in `spec-external-sector.md`. A trade with an external counterparty is the case for built-in reasons (8.1: `outside_budget`, `outside_unspent`, `outside_receipts`, `outside_offer`, `outside_unsold`, `outside_purchases`), and the counterparty's account is a pass-through account of the kind Outside swept to zero each tick (4). The goods side: a `Raw` commodity has three real sources, a recipe with no inputs, a RES-limited sited recipe and an import line; an import line with a sell side makes a commodity sourced for `spec-config.md` 7.5 and satisfies `raw_without_source`; a commodity sourced only so draws `import_only_source` (10.1). Still open for FIS (taxes) and LED (the ledger rows and kind).
- **Update (2026-10-10, SUB):** SUB adds no faucet, no sink and no ledger reason: the output of an own-use job is created by `recipe_output` (`spec-subsistence.md` 6).
- **Update (2026-10-10, MIG):** MIG adds no faucet, no sink and no ledger reason: a move is a relocation (no transfer), and the upkeep eaten on the journey is `consumption` (`spec-migration.md` 9.1).
- **Update (2026-10-10, MER):** MER adds no faucet, no sink and no ledger reason (`spec-merchants.md` 9.6.1).

### N-022 The player and other external agents

- **Areas:** INT
- **Raised:** 2026-10-09.
- **Note:** An agent added through the integration interface has weight 1
  (`spec-scale.md` 4.7). It has an ordinary agent account and is bound by the same
  admission and reservation rules as any agent (`spec-clearing.md` 3).
- **Update (2026-10-10, AGT):** `spec-agents.md` 10. An external agent has weight 1, an `External` flag, and a strategy that is either a non-blocking consumer `Strategy` or the library's `Directed`, which reads a `Directive` left in the agent's mailbox between steps (applied to the next tick, discarded unless `Sticky`, saved in a checkpoint). No action happens inside a step; an "act now" operation is not provided, and INT may add one with its own rules. INT owns `AddAgent` and `Direct`. The spawner must not remove or replace an external agent unless the consumer asks.
- **Update (2026-10-10, SPN):** an external agent is refused by the world for removal or replacement unless it is `Mortal` (`spec-spawner.md` 9, FR-SPN-010; `spec-agents.md` 10.6).
- **Update (2026-10-10, MIG):** the default migration module never reviews, moves or draws for an external agent; INT moves one by the guarded `relocate` (`spec-migration.md` 3.3, 10.3).

### N-025 Do policy actors reuse `Strategy`?

- **Areas:** PLC
- **Raised:** 2026-10-10, `roadmap.md` Open 2.
- **Note:** Open: whether a policy actor (a ruler choosing tax rates, minting,
  granary release, price controls) reuses the `Strategy` extension point or needs
  its own interface. A new interface means a change to the table in
  architecture.md 4.3, which also requires a default implementation and a
  compile-time assertion (4.5), and receives its randomness from the world (7.7).
  Decide when writing `spec-policy.md`.
- **Answer (proposed, 2026-10-10):** a separate interface, written when PLC is.
  ASSUME reuses one `calculate_bids()` interface for generators, demand and custom
  units, which supports one `Strategy` for every actor that *bids*. A policy actor
  does not bid: it reads measures and returns settings (rates, release amounts), so
  the signature and the returned type differ. A ruler that also trades (buying grain
  for a granary) uses `Strategy` for its orders and the policy interface for its
  settings. Do not create the interface before PLC (architecture.md 4.2).
- **Update (2026-10-10):** ADR 0003 decision 3 records the answer (its own interface, written with `spec-policy.md`). Close this note when PLC is written.

### N-027 No default macro constant is stable yet

- **Areas:** ALL, REF, STY
- **Raised:** 2026-10-10, `roadmap.md` Open 4 (ADR 0001, decision 5).
- **Note:** Stable defaults cannot be claimed until `spec-reference-chain.md` and
  the acceptance bands in `spec-stylised-facts.md` exist. Until then no spec states
  a default macro constant as stable. Related: the guessed numbers in N-015.
- **Answer (2026-10-10):** this is a standing constraint, not a question. Keep the
  note open so every spec reads it; close it when REF and STY exist.
- **Update (2026-10-10, RES):** every number of `spec-resources.md` is a placeholder or structural (a finite stock, a flat curve, a full start); the models show agreement with the linearisation, not stability of any default. INF and EXT set no default macro constant; the module has no default nodes.
- **Update (2026-10-10, SUB):** every number of `spec-subsistence.md` is a placeholder (`EnterTicks` 2, `ExitTicks` 3, `RetryJobs` 10, the bounds 8 and 1 000, the example quantities); the staple model shows structure and sensitivity, not stability of any default.
- **Update (2026-10-10, MIG):** every number of `spec-migration.md` is a placeholder (`MoveReviewTicks` 30, `MoveBelow` 10 000, `MoveGain` 1 000, `MoveWidth` 5 000, `MoveCooldown` 60, `MoveHorizon` 180, `MoveOutShare` 50, `MoveInShare` 100, the factor 3 of `migration_flow_window`, every number of the examples and of the models); the models show structure and which damper works alone, not the stability of any default.
- **Update (2026-10-10, MER):** every default of `spec-merchants.md` is a labelled placeholder and the models of `doc/specs/models/mer/` show agreement with the linearised delay condition, not the stability of any default.

### N-028 Can the target-income rule make a shortage worse?

- **Areas:** WEL, STB
- **Raised:** 2026-10-10, `roadmap.md` Open 5. See also N-014.
- **Note:** The rule in `spec-welfare-and-wages.md` may make labour supply fall as
  the wage rises, so a shortage could worsen itself. It is the first entry for
  `spec-stability.md`. Analyse it numerically when that spec is written, and decide
  whether the rule needs a damper or only a condition on demand. A cohort
  multiplies the lockstep of the rule (N-014).
- **Analysis (2026-10-10, derived by hand from section 5, not yet run):** days
  worked are `D = K/W`, so labour supply `N*K/W` is unit-elastic. A self-employed
  role has output `N*K/p`, the same curve in the goods price. With demand
  `Q = A*p^-e`, excess demand has slope `(N*K/p^2) * (1 - e)` at equilibrium, so a
  shortage corrects itself only if demand elasticity `e` is above 1. Staples have
  `e` below 1, so for them a shortage raises the price, days worked fall and the
  shortage deepens. Only three things stop it: the price bounds, households running
  out of money, and the one-day minimum. A spread above 0 smooths the steps but
  keeps the bend.
- **Answer (proposed):** keep the rule, because the backward bend is evidenced
  (Dyer, via Humphries and Weisdorf) for labourers. Make the damper a condition
  plus a test in `spec-stability.md`: bounded, asynchronous price steps and an
  inventory buffer (BEL), and a numeric run with a staple role that injects a
  shortage and asserts the price stays in bounds and recovers. Decide a work floor
  (a producer also works while its household's needs are short; survival before the
  target) from that run. A search snippet says inflection points in labour supply
  give multiple unstable equilibria; it is unverified (`specs/evidence.md`).
- **Update (2026-10-10, STB):** run in `spec-stability.md` 6.1 (stylised model, integer rule). The slope of excess demand is `(1 - e) Q/p`, exact to four places; stable only if `e > 1`; `e = 1` (a fixed budget) is neutral. For `e = 0.5` every seed ends pinned: at the price ceiling after a rise, or at the price where all households work the full year after a fall. An inventory buffer and `TargetSpread` 3 000 do not remove it. A work floor (minimum working days, D13) replaces the ceiling by a price set by the floor. The survival-floor reading is unit-elastic too and is not a damper by itself. The model has no income feedback, so it does not show that a real pack is unstable. **AGT (2026-10-10):** a staple's demand elasticity is 0 to 1, never above 1 (`spec-agents.md` 11.6), so the stability condition `e > 1` fails; the default strategy applies the rule with a rest band and a proportional work share for a cohort (7.1.2, 9.3). Leave open until WEL decides D13 (`MinWorkTicks`).
- **Update (2026-10-10, WEL and STB):** WEL adopted the floor as `MinWorkShare` (`spec-welfare-and-wages.md` 5.3, default 4 000 bp). `spec-stability.md` 6.1 gives the upper outcome as `P0 * (D0 / DMIN)^(1/e)`: about 1.06 times the reference for D0 = 150 days and the default floor, 2.9 for D0 = 250. The instability remains; scenario S04 records the outcome with the floor at its default and at 0. If a pinned market is still unacceptable, N-052 is the next lever. Leave open until S04 has run.
- **Update (2026-10-10, STB):** DEM answers for a luxury: stable only with schedule elasticity above 10 000 and about 10% budget headroom (`spec-demand.md` 6.3; a model), scenario S23. A staple is unchanged. Open until S04 and S23 have run.
- **Update (2026-10-10, SUB):** SUB gives the households that cannot afford a staple an own-use recipe (`spec-subsistence.md` 11). In a stylised model (`models/sub/staple.py`) it removes the shortage pin and 97% to 100% of the unfed household-ticks when the floor covers the upkeep; it does not change the elasticity of the labour supply of the households that can afford the staple, and the price falls to the lower pin (the glut) when the floor covers up to the upkeep and stays in an interior band when it covers more. It is not a damper of L14. Leave open until S04 has run on the library, with the module on and off.

### N-030 Five regions with different climates

- **Areas:** CAL, PRO, SCL, STY, REF
- **Raised:** 2026-10-10, the first consumer's world size (`vision.md` section 2).
- **Note:** The first consumer needs five regions of varying size and resource
  richness, with different climates, so a crop grows well in some and poorly or not
  at all in others. Consequences:
  - Climate is not an entity or an enum. It is data: a region's yield for each
    gathering or growing recipe, as a basis-point factor with a bound, optionally
    with a seasonal profile from `spec-calendar.md`. A factor of 0 means the recipe
    cannot run there. Nothing branches on a crop or a climate name (architecture.md
    9.1, `AGENTS.md` setting-agnostic rule). Richness is the same kind of data: the
    capacity and regeneration of an endowment (`spec-resources.md`).
  - So RES owns the yield factor and the endowment limit, CAL owns the seasonal
    profile, and PRO's recipe format must let a region modify a recipe's yield
    without a second recipe per region (N-003). Check this against a second,
    unrelated setting, as `vision.md` section 5 item 3 requires.
  - Differences in yield produce comparative advantage, which is what drives trade,
    price convergence within transport cost and specialisation. STY and REF need a
    scenario with at least three regions whose yields differ, not only the
    two-region arbitrage test (`roadmap.md`, REF row).
  - Varying size means households differ by region, so SCL's weights, `MinAgents`
    floor and cost model are checked at the smallest and largest region (N-015).
  - Size, confirmed 2026-10-10: about 1 000 households per region (5 000 in all)
    and 30 commodities, so 150 markets. At scale 1 that is 5 000 agents, within
    the placeholder budget of `spec-scale.md` 6.3 (about 25 ms a tick). Weights
    above 1 are not needed for this world, which also avoids the cohort lockstep of
    N-014 and the chance-event method of N-005 for the first consumer. SCL stays
    core because other consumers may be larger. Missing: the recipe count and the
    tick rate.
  - Region parameters override commodity reference prices (N-010), so each climate
    needs its own prices in the data pack.
  - Seasons, 2026-10-10: `spec-calendar.md` 5.3 defines one season cycle for the
    world. Regions with different climates share it through a table of yield by
    season (a reverse hemisphere reverses the table, a two-season region repeats
    values), not through a cycle of their own. RES owns that table.
- **Update (2026-10-10):** PRO applied its part in `spec-production.md` 6: the yield profile (`all` and by season, keyed by region and recipe), `Sited` for recipes that need a site, 0 = cannot run, no second recipe per region; two worked climates in 12.1 and 12.2. Remaining for RES (the table values and endowments, the `Draws` seam 6.5), REG, STY and REF (a three-region scenario). Still missing: the recipe count and tick rate of the first consumer.
- **Update (2026-10-10, VIA):** `spec-viability.md` 9 and 13.2 (A3, A5): comparative advantage from yield differences, a three-region check (prototype: autarky surplus 2.980, 2.517 and 3.124; pooled 3.124), scenario A3 handed to REF. REF must choose yield ratios large enough to show specialisation.
- **Update (2026-10-10, SCL):** with AGT, PRO and BEL describing the per-agent work, the cost model of `spec-scale.md` 6 is per tradable commodity (`c_0 + n_T * c_t`, placeholders 2 us, 1 us and a reference of 8 commodities): the first consumer's world of 5 000 agents is about 53 ms a tick (about 19 ticks a second), against the earlier 25 ms. Unmeasured; still missing: the recipe count and the tick rate.
- **Update (2026-10-10, REF):** `five` in `spec-reference-chain.md` 4.4.4: five regions with yields by season and recipe, 5 000 households, 30 commodities, 45 recipes (the missing recipe count), 85 active markets (17 per region, not 150), a mean tradable set of 7.7. Each region is viable alone; pooled 2.039. Still missing: the consumer's tick rate. Trade between regions waits for REG, TRN, MER and INF.
- **Update (2026-10-10, REG):** REG gives the region set and the graph. The five regions of `spec-reference-chain.md` 4.4.4 have no edges yet; `spec-regions.md` 12.4 proposes the edge part of `overlay/trade.toml` (a hub-and-pairs layout, 12 directed edges, distances placeholders until TRN fixes a unit's cost and speed). Yields by region keep the rules of `spec-regions.md` 5.3 (both keys exist, one row per key, canonical order). Remaining for RES, STY and REF as before.
- **Update (2026-10-10, POP):** the per-agent death carry works at weight 1, the first consumer's case; no weight above 1 is needed.
- **Update (2026-10-10, RES):** applied for RES in `spec-resources.md` 7: RES owns the VALUES of the yield table, PRO keeps the format, the lookup and the validation errors (`spec-production.md` 6); the table is core data and works with the resources module off. There is no climate entity and no function from richness to yield (rejected, 7.1.2): a climate is the rows of the table, richness is the capacity and the regeneration scale of an endowment (3.3). A new warning `no_comparative_advantage` checks the cross ratio of mean site factors (7.4). Endowments for the five-region pack are proposed in 18. Remaining: CAL, PRO, SCL, STY and REF as before.
- **Update (2026-10-10, MIG):** weight 1 is the first consumer's case, so no cohort moves; a cohort would move whole and lockstep (`spec-migration.md` 3.2, warning `migration_cohort`).

### N-031 What the calendar hands to other specs

- **Areas:** PRO, WLD, OBS, INT, GDS
- **Raised:** 2026-10-10, `spec-calendar.md`.
- **Note:**
  - RES and PRO: a season is the index `Calendar.At(tick).Season`, or -1 for none.
    A yield or a recipe condition by season is a table keyed by season ID; no logic
    names a season (architecture.md 9.1). A recipe whose production time exceeds its
    season cannot complete, and validation does not see it (`spec-calendar.md` 9);
    PRO or VIA should.
  - WLD: compute the moment once per tick, before orders are placed. It is derived,
    not state, and adds nothing to a checkpoint (`spec-calendar.md` 6.5).
  - OBS: decide whether the world emits a season event and whether the snapshot
    carries the moment (`spec-calendar.md` open question 4). Events already carry
    `Tick`.
  - INT: a consumer converts ticks to dates with `Calendar.At`; a shock by season
    (a failed harvest) needs the season index.
  - GDS and the other duration fields (`ShelfLife`, `TrainingTicks`): durations are
    in ticks and are not rescaled when the calendar changes, so a pack for another
    planet restates them (`spec-calendar.md` 6.1).
- **Update (2026-10-10):** PRO reads the season and the yield at **job start** (`spec-production.md` 6.4), so a recipe longer than its season completes; "cannot complete" does not arise for production and needs no validation. A harvest window is a perishable intermediate. INT: the yield-shock slot of 6.8 is where a seasonal failed harvest goes. RES, WLD, OBS and GDS parts unchanged.
- **Update (2026-10-10, VIA):** VIA reads the calendar as periodic (the season of tick `-o` is that of `(-o) mod TicksPerYear`) for the virtual start of initial jobs (`spec-viability.md` 8.5.2) and adds check V4 (supply gap against shelf life) for seasonal storage (4.5). Neither changes CAL; CAL may want to state the periodic extension (`At` rejects negative ticks).
- **Update (2026-10-10, CAL):** `spec-calendar.md` 4.5.1 adds the periodic helper `Calendar.SeasonAt(tick)` (the season of `floorMod(tick, TicksPerYear)`), separate from `At`; `spec-viability.md` 8.5.2 uses it. The "season shorter than a delay" entry (9) does not arise for production.
- **Update (2026-10-10, RES):** applied for RES in `spec-resources.md` 4.5 and 7.3: the season of a tick is the index `Calendar.At(tick).Season`, the yield table and the new regeneration profile are keyed by season ID, a profile is complete (an unlisted season is 0), no logic names a season. WLD, OBS, INT and GDS parts unchanged.

### N-032 Welfare window against recipe duration

- **Areas:** STB, CRD
- **Raised:** 2026-10-10, `spec-production.md` 5.2.3 and 16.
- **Note:** A job consumes inputs at its start and sells output at its end. The welfare ratio sums income over a window `T` (default 30, `spec-welfare-and-wages.md` 2.3.1), so a role whose recipe lasts `T` ticks or more shows costs and no income and looks like it is starving while healthy. That could trigger relief, migration and bankruptcy. Either the window must exceed the longest job plus selling time (PRO warns, `job_longer_than_window`) or income must be attributed to the job it came from. Credit is the real remedy for long jobs.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** `spec-welfare-and-wages.md` owns the window and the fix (it owns `T`), `spec-stability.md` records the loop. Not settled until WEL is revised.
- **Update (2026-10-10):** STB records it as STB-L19 and STB-L20 (`spec-stability.md` 4.2) with scenario S11 (a healthy producer with a job longer than the window is not flagged as starving). VIA 8.3.1 sets starting money to cover the longest job. The fix is still WEL's.
- **Update (2026-10-10, WEL):** decided in `spec-welfare-and-wages.md` 2.3.3: the window of a role is derived, `T = max(WelfareWindow, WindowCycles * longest Duration)`, `WindowCycles` default 2 (see N-062). Income attribution was rejected: it needs a valuation of work in progress or a rewrite of past records. The cost is a slower response of the ratio for long-job roles. Credit remains the real remedy (CRD); STB keeps scenario S11 (a healthy producer with a long job is not flagged as starving).

### N-035 Working tick with multi-tick jobs

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-production.md` 5.2.1.
- **Note:** `spec-welfare-and-wages.md` 4.3 defines a working tick by the `RecipeExecuted` event. With jobs, `RecipeExecuted` comes at completion and every tick of a job is a working tick, so the root reads working ticks from the agent's job state. Change the wording of 4.3 to "had a job in progress in the tick (started, continued or completed)". OBS: `RecipeExecuted` gains `StartTick`, `Executions`, outputs, `Disposed`, `EffectiveYield`; `RecipeAbandoned` is new. This is a change to the event table in `rules/observability-and-logging.md` 4.3, which needs a new read-receipt tag when made.
- **Update (2026-10-10, WEL):** applied for WEL in `spec-welfare-and-wages.md` 4.3: a working tick is a tick with a job in progress (started, continued or completed); the root reads it from the agent's job state. OBS: `RecipeExecuted` fields and `RecipeAbandoned` remain.

### N-036 Production's draws, phases and shocks in the world

- **Areas:** WLD, INT
- **Raised:** 2026-10-10, `spec-production.md` 5.5, 6.5, 6.6.2, 6.8.
- **Note:** WLD: production runs before orders; continuations and completions before starts; starts rotate by tick modulo the region's agent count (count taken at the start of the phase); at agent creation one draw in [0, 10 000) for production's residues, after the other creation draws in an order WLD fixes. INT: shocks are set and cleared between ticks through the yield-shock slot (a multiplier by region and recipe, default 10 000). RES: provide `Endowment(region, resource)` and `TakeEndowment(region, resource, n)`, own the resource definitions, depletion and regeneration, and make depletion visible by event since the ledger does not see it.
- **Update (2026-10-10, VIA):** applied for VIA in `spec-viability.md` 8.5, which staggers initial jobs with a fixed rule and answers `spec-production.md` open question 2; WLD must apply the warm start before the agents' creation draws (8.6 item 5).
- **Update (2026-10-10, REG):** the yield-shock slot and RES's `Endowment(region, resource)` key on `RegionID` (`spec-regions.md` 3.3). The edge status of `spec-regions.md` 7 is a second slot of the same shape (world state, set between ticks) for INT's closed-road shock; the other slots are listed in N-091.
- **Update (2026-10-10, RES):** applied for RES in `spec-resources.md` 5, 6 and 14: the seam is `Endowment` (public read) and an internal all-or-nothing take; RES owns the resource definitions, depletion and regeneration; depletion is visible by the events `EndowmentChanged` and `EndowmentAdjusted`, a bounded `Warn` and the counter `Short`. WLD: take a snapshot of every stock at the start of the production phase, after continuations and completions, ration the draws before the starts (5.2), and run regeneration as the last step of phase 8 before the optional audit (4.1). INT: two more slots for N-091, the scales and the adjustment (6.3). The ration replaces the rotation of `spec-production.md` 5.5.1 for contention on an endowment (decided at the wave 3 integration); the rotation still orders the starts.

### N-039 A hard cost floor

- **Areas:** STB, GDS, PRO, SKL
- **Raised:** 2026-10-10, `spec-beliefs.md` 18.1.
- **Note:** `CostFloor` 10 000 (100% of unit cost) stops sales below cost, so perishables can spoil instead of selling cheaply and a market can freeze with buyers below the floor. The stability tests need a perishables-glut scenario. `spec-skill-barriers.md` 2.4 ("cost acts through exit") needs rewording, because cost now also acts directly on a seller's price.
- **Update (2026-10-10, STB):** `spec-stability.md` 6.2 run 5 gives numbers for the perishables-glut scenario (S07): sellers with cost 1 100, buyers capped at 1 000, shelf life 3 ticks. `CostFloor` 10 000: 0% of ticks with a trade, 99% of output spoiled; 9 000: 7% and 98%; 8 000: 97% and 69%; 5 000: 100% and 21%. The market freezes whenever the floor is above what buyers can pay (STB-L09, regime F5). BEL decides its open question 1.
- **Update (2026-10-10, BEL and STB):** `CostFloor` is now 7 000 on a cost that includes labour (`spec-beliefs.md` 3.2.2). `spec-stability.md` 6.2 run 5: 7 000 trades in every tick (45% spoiled with the model's mismatched flows); 10 000 and 9 000 still freeze, and a wage 10% to 25% above the model needs 8 000 or less. `spec-skill-barriers.md` 2.4 is reworded.
- **Update (2026-10-10, RES):** with RES the abundance is a factor of the effective yield, so a thin stock raises `UnitCost` (`spec-production.md` 9.1 divides by `E`) and with it the cost band of beliefs (`spec-resources.md` 4.7.5, RES-A6, L91). A cost floor above what buyers pay can freeze a market whose stock is thin; scenario RES-S3 (S43) with perishables is the test.

### N-040 Belief event, log keys and snapshot

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-beliefs.md` 12 and 18.9.
- **Note:** `BeliefsUpdated` (one aggregated event per market per tick, 12.1) and the log keys `region`, `bound` and `agents` (12.3) join the list of N-012. Whether `Snapshot` carries beliefs is open.

### N-042 The evidence rule may offset the price

- **Areas:** BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2.
- **Note:** In a stylised model of the rules of `spec-beliefs.md` (`spec-stability.md` 6.2), the evidence component of 5.4 settles the price 11% to 13% below the price at which supply equals demand and leaves about 12% of need unmet; with the evidence off, the stock component alone settles at 1 001 of 1 000 with nothing unmet. A trace shows that unfilled bids and asks mostly move their centre down, because the target (the other side's best unmatched quote) can lie on the wrong side of a centre that the limit was drawn away from. Restricting the move to the direction of the failure removes only part of it. This is a finding about a model of the rule as written, with one quantity-sizing rule and no money loop. BEL should reproduce it or revise 5.4 (compare the evidence to the limit, not the centre); FR-STB-018 (clearing residual) is the test.
- **Update (2026-10-10, BEL and STB):** reproduced and revised. `spec-beliefs.md` 5.4 compares the evidence with the order's limit, guarded, scaled by `BeliefEvidenceShare` (FR-BEL-036, 037). In the stylised model the offset falls from 11.3% to 5.7%, with 118 of 2 000 need unmet; `spec-stability.md` 6.2 run 2b reproduces this independently. Reduced, not removed: `spec-beliefs.md` open question 12 owns the rest (candidate remedy: a unit-count imbalance signal). Test: S01 with FR-STB-018.
- **Update (2026-10-10, DEM and STB):** a DEM schedule is a second quantity-sizing rule; run 2b of `spec-stability.md` 6.2 has not been repeated with it (`spec-demand.md` open question 5; register open question 6).

### N-043 Nothing anchors the price level if flows ignore price

- **Areas:** AGT, PRO, SPN, BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2 (runs 1 and 2).
- **Note:** With price-inelastic production and consumption, the belief rules have no interior attractor: an early inventory overhang ratchets the price to the floor of the reference band, a shortage to the ceiling, and a small change in a production-stop threshold decides which. A price-responsive flow must exist somewhere (a consumption rule that depends on price or money, a production choice by margin, an inventory capacity or production stop, exit through the spawner). The register proposes a production stop D24 and records the elasticity condition `es < ed` for cobwebs (6.3).
- **Answer (2026-10-10, AGT):** `spec-agents.md` 11.6 states that a staple's demand elasticity is 0 to 1, never above 1, so `e > 1` and `es < ed` fail for a staple; the responsive flows are buffer buying, recipe choice by margin, the production halt D24 (now specified with `ProduceHaltRatio`) and exit via the spawner. An elastic subsistence demand is not invented; if S04 or S16 shows a pinned market that the halt and exit do not cure, see N-052. Leave open for SPN (exit) and the scenarios.
- **Update (2026-10-10, STB):** `spec-stability.md` 6.2 run 1b: with AGT's target the default halt removes the floor pin and leaves a shortage cycle (price 1.5 to 3.1 times the reference, 12% of need unmet), so the halt bounds the glut but is not an anchor. Open for SPN (exit) and scenarios S04 and S16; see N-067.
- **Update (2026-10-10, DEM and SPN):** DEM supplies a price-responsive flow for non-staples (D30). SPN's entry and exit are slow, capped flows (`EntryShare` 50 bp, `ExitShare` 100) and are not an anchor for a pinned staple (`spec-spawner.md` 16). A pure staple is unchanged: halt, buffer buying, recipe choice, work floor. Scenarios S04 and S16 decide whether this is enough.
- **Update (2026-10-10, EXT):** an outside quote anchors the price of a traded good at the edges of its corridor, never in the interior, and weakly under the midpoint rule (`spec-external-sector.md` 6.7, EXT-A2, extended into L21). In the stylised model of 19.2 a corridor of 45 to 75 confines a cobweb that cycles between 20 and 120 alone, provided the capacity covers the swing. It is an anchor only for goods the outside deals in.
- **Update (2026-10-10, SUB):** SUB is a price-responsive flow for a staple that is not an invented demand: a household whose secure ticks fall below `EnterTicks` feeds itself and its bids shrink with its stock (`spec-subsistence.md` 11.1; SUB-A2, L102). It anchors the shortage side for the households that can take the mode and not the interior of the price. A pure staple role with no own-use recipe is unchanged.

### N-045 Dampers that act silently

- **Areas:** OBS, STB
- **Raised:** 2026-10-10, `spec-stability.md` (FR-STB-012, FR-STB-013).
- **Note:** FR-STB-013 requires every clamp, refusal, floor or limit to be counted in an event or a bounded `Warn`. Counters exist for belief clamps and order rejections (`BeliefsUpdated`, `OrderRejected`). Missing: the cap or floor on days worked, entry limits, the production stop. FR-STB-012: the regime measures must be computable from events and the snapshot alone: price, volume and unmatched per market, at-floor and at-ceiling per market, weights on spawn and bankruptcy, boundary flows, pool balance, pending output.
- **Update (2026-10-10):** counters now exist for the work floor (Snapshot and bounded `Warn`, `spec-welfare-and-wages.md` 5.3, 11.3; `spec-agents.md` 7.1.3, FR-AGT-072) and for the traineeship queue and places (`spec-skill-barriers.md` 8.3). Still missing: the production halt (a per-agent `Debug` line only, N-067) and a spawner entry limit.
- **Update (2026-10-10):** counters now exist for the demand schedule and ladder (`WantShortfall`) and for the spawner's limits (`EntryLimited`). Still missing: the production halt (N-067).
- **Update (2026-10-10, MON):** `MoneyFlowLimited` is the counter for the silent clamps of MON (a levy found too little, a grant cut at `MaxMoney`, the limit).
- **Update (2026-10-10, POP):** the silent clamps of POP are counted in the event `PopulationChanged`: `DeathsCut`, `DeathsHeld`, `BirthsCapped`, `BirthsUnfunded`, `RateClamped` (`spec-population.md` 10.1).
- **Update (2026-10-10, RES):** the cut demand on an endowment is counted: `EndowmentChanged` carries `Short`, `AgentsDenied` and `AgentsReduced` (aggregate per endowment per tick) and a bounded `Warn` "endowment exhausted" (`spec-resources.md` 6.2.2, 14). The production halt and the spawner entry limit are still as before.
- **Update (2026-10-10, INF):** the closed road, the horizon and the young run are counted by `RumoursServed` (`NoPath`, `TooOld`, `NotYet`). EXT counts its silent cuts the same way: `CutCap`, `CutClosed` and `PriceClamped` in its events (`spec-external-sector.md` 15, EXT-D2, D4, D5).
- **Update (2026-10-10, SUB):** the silent rules of SUB are counted in `OwnUseChanged` (`Entered`, `Left`, `Started`, `Blocked`, `Retried`, `Output`) and by two bounded `Warn`s (`spec-subsistence.md` 14.1, 14.2).
- **Update (2026-10-10, MIG):** every silent limit of the migration module is counted in `MigrationReviewed`: the settling rule (`Unsettled`), the gates and the caps (`NoRumour`, `TooFar`, `NoViable`, `NoGain`, `OutCap`, `InCap`, `Unprovisioned`, `Bound`), held arrivals (`Held`) and journeys ended by a removal (`Ended`); every pushed household ends in exactly one outcome (`spec-migration.md` 7.6, 13.1), and two bounded `Warn`s say when a region is blocked or holds arrivals for ten ticks.
- **Update (2026-10-10, MER):** the merchant's silent dampers (the lane lock, the lot cap, the source-age bound, the rest, the stay) are counted by `MerchantsPlanned` through the plan notes, and the contract round's limits by `ContractRound` (`spec-merchants.md` 15.1).

### N-046 Register entries owed by unwritten specs

- **Areas:** EMP, SAV, CRD, STO, RSK, FIS, PLC, LND, WLD
- **Raised:** 2026-10-10, `spec-stability.md` 4.7 and 9.
- **Note:** Each adds its entries by the checklist of `spec-stability.md` 9 and states its dampers by `STB-Dnn`. The expected entries are in section 4.7 (SPN: the failure ladder, entry limits, replacement by the most profitable role; POP: the Malthusian fixed point; and so on). A spec that adds a loop and no entry is not done (FR-STB-003). Undamped now: see `spec-stability.md` section 10.
- **Update (2026-10-10):** the DEM and SPN entries are merged into the register (L35, L36, L45, L46, L47, L50, L54 and L57 to L65).
- **Update (2026-10-10, REG):** REG has its entries (REG-A1 to A3, `spec-regions.md` 13), now L66 to L68; the TRN, MER, MIG and INF rows of `spec-stability.md` 4.8 depend on the distances of `spec-regions.md` 4.3.
- **Update (2026-10-10, MON):** MON-A1 to MON-A7, dampers MON-D1 to MON-D5 and scenarios S13 (defined) and MON-S1 to MON-S4 are in `spec-money-supply.md` 16; in the register MON-A1 is L69, MON-A2 merges into L30, MON-A3 to A7 are L70 to L74, the dampers are D41 to D45 and the scenarios S33 to S36.
- **Update (2026-10-10, POP):** the POP entries are in `spec-population.md` 14 (POP-A1 to A9, dampers POP-D1 to D7, scenarios POP-S1 to S4); in the register they are L75 to L83, D46 to D52 and S37 to S40. POP's row in 4.8 can be removed.
- **Update (2026-10-10, TRN):** TRN's entries are TRN-A1 to TRN-A8 in `spec-transport.md` 15 (price-gap shipping with a transit delay, the invisible pipeline, the haul capacity cobweb, freight and the delivered price, perishables that cannot cross, a trip as a gap in income, vehicles as capital, diversion to the pool); in the register TRN-A1, A2 and A5 are L84 to L86, A3 merges into L21, A4 into L17, A6 into L19, A7 into L26 and A8 into L63; the dampers are D38 to D40 and the scenarios S30 to S32.
- **Update (2026-10-10, wave 2 integration):** the entries of REG, MON, POP and TRN are merged into the register, so POP, MON and TRN leave the areas of this note. Still owed: MIG, INF, EMP, EXT, SAV, CRD, MER, STO, RSK, FIS, PLC, RES, LND, SUB, WLD.
- **Update (2026-10-10, wave 2 integration):** the register now has 86 loops, 52 dampers and 39 scenarios (S01 to S40, S15 unused). The next free numbers are **L87, D53 and S41**. A writer uses provisional IDs (`<CODE>-A1`, `<CODE>-D1`, `<CODE>-S1`) in its spec, and the supervisor assigns the final numbers when the entries are merged (N-103); do not reuse a number a spec quotes provisionally.
- **Update (2026-10-10, wave 3 integration):** the entries of RES, INF and EXT (`spec-resources.md` 19, `spec-information.md` 13, `spec-external-sector.md` 17) are merged into the register, so RES, INF and EXT leave the areas of this note. New loops: RES-A1 to A3 are L87 to L89 and RES-A5 to A7 are L90 to L92 (RES-A4 merges into L29); INF-A2 to A4 are L93 to L95 (INF-A1 merges into L84 and L85 as a case, INF-A5 into L23); EXT-A1 is L96 and EXT-A3 to A7 are L97 to L101 (EXT-A2 extends L21, EXT-A8 merges into L84 as a case). Dampers: RES-D1 to D6 are D53 to D58, INF-D1 to D3 are D59 to D61, EXT-D1 to D5 are D62 to D66. Scenarios: RES-S1 to S5 are S41 to S45, INF-S1 to S5 are S46 to S50, EXT-S1 to S5 are S51 to S55. The register now has 101 loops, 66 dampers and 54 scenarios (S01 to S55, S15 unused). The next free numbers are **L102, D67 and S56**. Still owed: MIG, EMP, SAV, CRD, MER, STO, RSK, FIS, PLC, LND, SUB, WLD.
- **Update (2026-10-10, SUB):** SUB-A1 to SUB-A7, dampers SUB-D1 to SUB-D5 and scenarios SUB-S1 to SUB-S5 are in `spec-subsistence.md` 18. SUB-A1 merges into L11 (and names L39 and L63 as beneficiaries), SUB-A7 into L18, L40 and L46; SUB-A2 to SUB-A6 are new loops and hazards (L102 to L106).
- **Update (2026-10-10, MIG):** MIG-A1 to MIG-A9, dampers MIG-D1 to MIG-D7 and scenarios MIG-S1 to MIG-S6 are in `spec-migration.md` 17. MIG-A4 merges into L40 (relief and migration), MIG-A5 into L68 and L76, MIG-A6 into L19; MIG-A1, A2, A3, A7, A8 and A9 are new (L107 to L109, L110 to L112).
- **Update (2026-10-10, MER):** MER's entries are MER-A1 to MER-A7 in `spec-merchants.md` 17 (the merchant's response to a price gap, the consignee's take loop, the merchant at its target income, derived demand for haul units with a choke price, the escrow, the drift of merchants between regions, a stranded preparation); MER-A1 merges into L84, L85 and L93, MER-A3 into L14 and L16, MER-A4 into L16 and L21, and MER-A2, A5, A6 and A7 are new (L113 to L116). Dampers MER-D1 to MER-D8, scenarios MER-S1 to MER-S6. The MER hook row of `spec-stability.md` 4.8 can be removed.
- **Update (2026-10-10, wave 4 integration):** the entries of SUB, MIG and MER are merged into the register, so SUB, MIG and MER leave the areas of this note. Dampers: SUB-D1 to D5 are D67 to D71, MIG-D1 to D7 are D72 to D78, MER-D1 to D8 are D79 to D86. Scenarios: SUB-S1 to S5 are S56 to S60, MIG-S1 to S6 are S61 to S66, MER-S1 to S6 are S67 to S72. The register now has 116 loops, 86 dampers and 71 scenarios (S01 to S72, S15 unused). The next free numbers are **L117, D87 and S73**. Still owed: EMP, SAV, CRD, STO, RSK, FIS, PLC, LND, WLD.

### N-051 What the Strategy contract needs from other specs

- **Areas:** INT, CHK, WLD
- **Raised:** 2026-10-10, `spec-agents.md` 6, 10.
- **Note:** INT: `AddAgent`, `Direct` and the mailbox. CHK: the strategy names are hashed, the code is not (`spec-agents.md` open question 4), and the directive is state. WLD: the phase order of `spec-agents.md` 4, `Work` between continuations and starts, `Trade` after production, `Revise` per outcome with the world's draw.
- **Update (2026-10-10, SPN):** SPN needs from WLD the phase-7 position, regions in order and the creation draws after all `Decide` calls (`spec-spawner.md` 10.2); from INT the `Mortal` flag of `AddAgent` (N-079); from CHK the spawner name hashed and its state pair.

### N-056 What the ledger and world need for a warm start

- **Areas:** WLD
- **Raised:** 2026-10-10, `spec-viability.md` 8.6.
- **Note:** (1) `initial` for goods into any account at construction, including estate accounts; (2) lots with a negative `Born` and durable items with remaining durability below full at construction; (3) a construction bucket in the flow totals before tick 0, included in the audit; (4) a job created at construction with `Remaining < Duration` and pending output fixed from a virtual past start; (5) the warm start is applied before the agent creation draws (N-009, N-036).
- **Update (2026-10-10, LED):** items 1 to 3 are applied in `spec-ledger.md` 5.2.4, 7.4.1 and 9.1 (stated lot key at construction, signed `Born`, construction bucket in the audit). Items 4 and 5 stay with WLD (a job created at construction; the warm start before the creation draws); see N-074.

### N-061 The load-time size gate is a guess

- **Areas:** VIA, CFG
- **Raised:** 2026-10-10, `spec-config.md` 4.1.
- **Note:** `MaxViabilityCells` and `MaxViabilityPivots` are placeholders and the speed of the exact solver in Go is unmeasured. Measure in the slice so that NFR-CFG-001 holds with the programmes on.
- **Update (2026-10-10, REF):** `five` runs 202 pivots in the Python prototype; still unmeasured in Go.

### N-065 Silent dampers need counters

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-agents.md` 7.1.3, `spec-welfare-and-wages.md` 5.3.
- **Note:** The work floor is counted by the world (a pure function of the welfare record, so a consumer's strategy cannot hide it): a Snapshot field and a bounded `Warn` "work floor binding" (`spec-agents.md` FR-AGT-072). The traineeship queue has its own `Warn`. OBS names the Snapshot fields. The production halt and a spawner entry limit are still uncounted (N-067).
- **Update (2026-10-10, POP):** the silent clamps of POP are counted in `PopulationChanged` (`spec-population.md` 10.1); the production halt is still uncounted (N-067).
- **Update (2026-10-10, INF):** the closed road, the horizon and the young run are counted by `RumoursServed` (`spec-information.md` 10.1).
- **Update (2026-10-10, SUB):** as N-045: the mode, the retry and the blocked households are counted by an aggregate event per region and role.
- **Update (2026-10-10, MIG):** as N-045: every pushed household ends in one counted outcome of `MigrationReviewed`.
- **Update (2026-10-10, MER):** as N-045: the merchant's dampers are counted by `MerchantsPlanned` and `ContractRound`.

### N-067 The production halt is not an anchor and has no counter

- **Areas:** AGT, OBS, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2 run 1b.
- **Note:** With inelastic flows the default halt (30 000 bp of a one-tick target) gives a shortage cycle, not a stable price, and it is counted only by a per-agent `Debug` line. AGT and OBS: an aggregate per region and role per tick in the style of `UpkeepShortfall`. Scenario S16 must set the threshold.
- **Update (2026-10-10):** `spec-reference-chain.md` 9 repeats the request for an aggregate counter; `spec-demand.md`'s handoff proposes the name `ProductionHalted` (Region, Role, Households, Recipes). OBS decides.

### N-071 One wage per region and skilled recipes

- **Areas:** VIA, SKL
- **Raised:** 2026-10-10, `spec-viability.md` open question 12.
- **Note:** One wage per region costs a skilled recipe at the common wage. Decide whether the solver reports a wage per recipe class.

### N-072 The construction bucket as an event

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-ledger.md` 7.4.1 and open question 9.
- **Note:** Decide whether the construction bucket is emitted as a `BoundaryFlow` with `Tick` -1 (recommended in the ledger).

### N-074 The construction phase ends at the first Step

- **Areas:** WLD
- **Raised:** 2026-10-10, `spec-ledger.md` 5.2.4.
- **Note:** A stated lot key (negative `Born`, a worn durable) is allowed only during construction; WLD ends the phase at the first `Step`. With N-056 items 4 and 5: a job created at construction, and the warm start before the creation draws.

### N-075 Mapping the ledger's sentinels

- **Areas:** CFG
- **Raised:** 2026-10-10, `spec-ledger.md` 5.2.4 and 8.2.1.
- **Note:** `ErrInvalidLot`, `ErrNotConstruction` and `ErrEmptyTransform` are ledger-internal; the root needs a mapping to public sentinels. Bounds for the estate fields are in `spec-estates.md` 3.
- **Update (2026-10-10):** the sentinels `ErrUnknownSpawner`, `ErrUnknownWant`, `ErrWantItems` and `ErrWantDurable` also need a root mapping.
- **Update (2026-10-10, MON):** MON adds the sentinels `ErrFlowShape` and `ErrReasonCollision`; the root needs a mapping.
- **Update (2026-10-10, REG):** REG adds the root sentinels `ErrNoRegions`, `ErrSelfEdge`, `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation` and `ErrRegionBound`; the last three of the relocation may be ledger-level if `Relocate` is in `internal/ledger`. `ErrUnknownAgent` may collide with INT's.
- **Update (2026-10-10, RES):** RES adds the root sentinels `ErrUnknownResource`, `ErrResourceShape`, `ErrCurve` and `ErrInsufficientEndowment`; the last is mapped from `internal/endowment`.
- **Update (2026-10-10, EXT):** EXT adds the sentinels `ErrOutsideNodeInhabited`, `ErrCrossedQuotes`, `ErrNotAGate` and, with REG, `ErrOutsideNode`; the root needs a mapping.
- **Update (2026-10-10, SUB):** SUB adds the sentinels `ErrOwnUseBand`, `ErrOwnUseNotSource`, `ErrOwnUseNotSited` and `ErrOwnUseOverlap`; the root needs a mapping.
- **Update (2026-10-10, MER):** MER adds the root sentinels `ErrUnknownMode`, `ErrMerchantShape` and `ErrMerchantWeight`; the root needs a mapping.

### N-079 External agents are mortal only when the consumer says so

- **Areas:** INT
- **Raised:** 2026-10-10, `spec-agents.md` 10.6, `spec-spawner.md` 9.
- **Note:** `AddAgent` carries a `Mortal` flag (default false); the world refuses decisions on an external agent unless it is mortal.
- **Update (2026-10-10, MIG):** as N-022: a mortal external agent is not moved by the default module.

### N-080 Wants in the steady-state programme use equal-price shares

- **Areas:** VIA, DEM
- **Raised:** 2026-10-10, `spec-viability.md` 3.2.
- **Note:** Wants enter the programme at the reference weights, fixed and unscaled by the surplus factor. Revisit with scenario S25 (a substitution flip changes the shares).

### N-081 Snapshot figures the scenarios need

- **Areas:** OBS, WEL, AGT
- **Raised:** 2026-10-10, `spec-reference-chain.md` 9.
- **Note:** (1) the mean welfare ratio per role and region (`spec-welfare-and-wages.md` 11.2, FR-WEL-036: specified); (2) unsold stock per commodity; (3) a production-halt counter (N-067).

### N-082 Relief and entry share the pool in time

- **Areas:** WLD, EST
- **Raised:** 2026-10-10, `spec-spawner.md` 7.3, `spec-estates.md` 2.6.
- **Note:** Relief is paid in phase 1 from what the previous tick's phase-7 entry step left, so the pool is shared in time. WLD must keep the order: phase 1 relief, phase 7 entry.
- **Update (2026-10-10, POP):** births draw from the pool in phase 7 inside `EntryPoolShare`; the POP deaths are executed before the budget is taken so the estates of the dead fund the same tick's births (`spec-population.md` 4.8). WLD must keep phase 1 relief, phase 7 entry. The pool is shared more widely now: see N-096.
- **Update (2026-10-10, SUB):** the subsistence floor draws nothing from the pool and lowers the draw on it (`spec-subsistence.md` 8.4); a floor-fed household still draws relief on a ratio that includes its receipts. Whether relief should pay a household that feeds itself is `spec-subsistence.md` open question 8 (N-119).
- **Update (2026-10-10, MIG):** the migration module draws nothing from any pool; its indirect effects are the relief a poor arrival draws from the destination's pool (L63, MIG-A4) and the money a migrant leaves to the origin's pool if it dies on the journey (`spec-migration.md` 9.2).

### N-084 Scenarios that wait for trade, money and the coarse tick

- **Areas:** STY
- **Raised:** 2026-10-10, `spec-reference-chain.md` 7.6.
- **Note:** S13 (the faucet overlay), the dynamic parts of X1 and X2 (the trade overlay) and the coarse-tick leg of S12 wait for these specs.
- **Update (2026-10-10, REG):** REG supplies the edge format and a proposal for the edge part of `overlay/trade.toml` (`spec-regions.md` 12.4), and two scenarios, S28 and S29 (in the register S28 and S29). X1 and X2 (dynamic) still wait for MER and INF.
- **Update (2026-10-10, MON):** S13 is defined: `overlay/faucet.toml` (and `overlay/faucet-sunk.toml`) are in `spec-money-supply.md` 17.6, and the pass is exact for the accounting (boundary balance = `-Total`, `M = M_init + issued`) and report-only for the price drift until STY sets a band. Still open: the dynamic parts of X1 and X2, the coarse-tick leg of S12.
- **Update (2026-10-10, TRN):** the transport mechanism for the dynamic parts of X1 and X2 exists (modes, the shipment, `Inbound` and `Routes`: `spec-transport.md` 3 to 9). An `overlay/trade.toml` needs a `modules.transport.modes` table, a hired mode's haul commodity and a carrier recipe (17.4 is a model), a consignee in the destination, MER's shipper and INF's rumours. The static check can use `transport.Lanes` (9.4). Scenarios S30 to S32 are proposed in the register (TRN 15 called them S28 to S30).
- **Update (2026-10-10, wave 2 integration):** REG, MON and TRN now exist, so only EXT, MER, INF and STY are left in the areas. The overlays are N-092.
- **Update (2026-10-10, INF):** INF now supplies the rumours (`spec-information.md` 5). The overlay `overlay/trade.toml` needs a `[modules.information]` table (`info_speed`, `rumour_error_bp`, `rumour_max_age`) and an informed merchant role (`informed = true`); 12.1 there shows the keys.
- **Update (2026-10-10, EXT):** EXT exists (`spec-external-sector.md`). Its invariant scenario EXT-S5 (S55) runs the reference chain with an external overlay (an outside node on a gate region, 18.2 is a model of the file layout) over a range of seeds with `WithAudit`; REF builds the pack. EXT gives the dynamic parts of X1 and X2 the gate as the place where imports appear and the counterparty's ask as the price a shipper reads there.
- **Update (2026-10-10, wave 3 integration):** INF and EXT now exist, so only MER and STY are left in the areas. The dynamic parts of X1 and X2 wait for MER (the shipper); the overlays are N-092.
- **Update (2026-10-10, MER):** MER exists. The dynamic parts of X1 and X2, S28 to S32, S46 and S47 can run with a merchant role: `overlay/trade.toml` needs `[modules.merchants]`, a role with `strategy = "merchant"`, `informed = true`, a `[role.<id>.merchant]` block and `consigns = true` for the consignee role (`spec-merchants.md` 19.1 shows the keys), plus the Go line `economysim.NewWorld(cfg, strategy.Options()...)`.

### N-085 Nominal table against the full-labour table

- **Areas:** VIA, STY
- **Raised:** 2026-10-10, `spec-viability.md` 5.3.4.
- **Note:** When the wage dual is 0 the warm start uses the nominal programme. Run both tables on `famine` and compare.

### N-086 Re-run the demand and spawner models under the register

- **Areas:** STB, STY
- **Raised:** 2026-10-10, `spec-stability.md` 6.5 and open question 6.
- **Note:** The numbers quoted in 6.5 are from the owners' models. STY or EXP should re-run the luxury model with DEM's schedule and the occupation model with SPN's discrete rules, and set the `entry_share_window` threshold from the run.

### N-087 Wants against the production halt

- **Areas:** AGT, OBS, STB
- **Raised:** 2026-10-10, `spec-stability.md` open question 8.
- **Note:** Does the halt cut the supply of want goods in a downturn and deepen the wage and demand loop (L57)? Run S24 with the halt on and off.

### N-088 Conflicts left by the rules edits

- **Areas:** OBS
- **Raised:** 2026-10-10, `rules2` report.
- **Note:** (1) `household.unfed` is an up-down counter fed by an event that is absent when everyone is fed: the adapter keeps per-region and role state and resets at the end of the tick; (2) the construction bucket as a `BoundaryFlow` with `Tick` -1 is undecided (N-072); (3) the snapshot-fed gauges (production work in progress, the beliefs' mean centre) have no name and conflict with 5.3.1 ("derivable from events"); (4) `agent.bankruptcies` counts every cause of `AgentBankrupted`, including `consumer` and `population`, though the name says bankruptcies; (5) whether the estate's `ReliefPaid` and `EstateOpened` belong to a module that can be off.

### N-089 Model scripts of the demand, spawner and reference-chain specs

- **Areas:** DEM, SPN, REF, CODE
- **Raised:** 2026-10-10, `spec-demand.md` 14, `spec-spawner.md` 16, `spec-reference-chain.md` 14.
- **Note:** The scripts behind these specs are still in `/tmp/economy-boss/` (`dem/`, `spn/`, `ref/`). Archive them under `doc/specs/models/` like the others, and update the specs' paths.

### N-090 What MIG and the specs that tie obligations to a region must do with a relocation

- **Areas:** PRO, SKL, INT
- **Raised:** 2026-10-10, `spec-regions.md` 6.
- **Note:** `relocate` moves only the account's region (`spec-regions.md` 6.2). MIG decides the job (abandon it first; the abandon reason `region_changed` is proposed for `spec-production.md` 5.4, REG open question 3), the beliefs (reseed on arrival, N-038), `Privation` and the traineeship. SKL registers an `in_traineeship` refusal and CAR or MER an `open_contract` refusal (`region_bound`, 6.5.2). INT owns the public face of `relocate` for an external agent (6.6). The part for TRN is done: it registers `open_shipment` and calls `Relocate` with cause `transport` at an arrival.
- **Update (2026-10-10, SUB):** SUB's part: the mode and its counter are dropped at a relocation and at the departure of a timed move; SUB has no reason and no event for the drop; an own-use job is abandoned with `region_changed` as any job (`spec-subsistence.md` 5.8). A household on a journey or a trip is not evaluated.
- **Update (2026-10-10, MIG):** MIG's part is applied in `spec-migration.md` 7.2 and 8.2 to 8.5: the job is abandoned at departure with `region_changed` (adopted in `spec-production.md` 5.4); the beliefs are reseeded at arrival; `Privation` and `Hunger` are kept and `RoleSince` is set at arrival, which gives the ladder's grace of `spec-spawner.md` 5.2; a trainer or a trainee, an away traveller and a party to an open contract are refused by the guard, the read-only query `World.RelocationRefusal(agent, to, skip...)` (`spec-migration.md` 7.2.1; REG 6.1), run again at arrival with `skip` = `on_journey`, where a refusal HOLDS the arrival; the region then changes by the ledger-level `Relocate`, cause `migration`. MIG registers the `region_bound` refusal `on_journey`. Still open: SKL (register `in_traineeship`), INT (the public face of `relocate` for an external agent, and the cancel of a journey before a forced move, `spec-migration.md` 8.4.1).
- **Update (2026-10-10, MER):** MER registers the `region_bound` refusal `open_contract` (open for the payer and the deliverer of an open contract, `spec-merchants.md` 9.8.1). A merchant on a trip is away, is refused by `open_shipment` and `open_contract`, and is not a migrant (12.4.1). Removal of a party: 9.8.2.
- **Update (2026-10-10, wave 4 integration, X2):** the full list of region-bound refusals is `open_shipment` (TRN), `on_journey` (MIG), `open_contract` (MER) and `in_traineeship` (SKL). A timed move has the shape of TRN's traveller: the account stays in the origin while away and the region changes at arrival through the ledger-level `Relocate`; REG gains a read-only dry run of the relocate refusals with an optional `skip` of one named refusal.

### N-091 The world slots INT must expose

- **Areas:** INT
- **Raised:** 2026-10-10, `spec-regions.md` 7, `spec-money-supply.md` 10.4, `spec-population.md` 3.6 and 6.6, `spec-transport.md` 9.1 and 10.3.
- **Note:** Several mechanisms are world slots set between ticks, saved in a checkpoint, with a public face that belongs to INT (decided at the wave 2 integration: the mechanism belongs to the spec that owns it, the public call to INT): the yield shock (`spec-production.md` 6.8), the edge status, closed or slowed (`SetEdgeStatuses`, `spec-regions.md` 7), the transit factor (`spec-transport.md` 10.3), the flow scale (`SetFlowScale`, `spec-money-supply.md` 10.4) and the birth and death scales (`BirthScale`, `DeathScale`). INT also owns `relocate` for an external agent. For each, INT states the public name, its error values and its bound, between steps only. Each slot emits its own event and none is configuration.
- **Update (2026-10-10, RES):** RES adds two slots for INT: per endowment a capacity scale and a regeneration scale (basis points, 0 to `MaxShockBP`, default 10 000), and a one-off signed adjustment of the stock (`spec-resources.md` 6.3). Both are set between ticks, saved in a checkpoint, emit `EndowmentAdjusted`. The failed harvest stays the yield shock.
- **Update (2026-10-10, INF):** INF has no world slot. A blackout shock (slow or cut the news) would be one, set between ticks (`spec-information.md` 17 (open question 6)).
- **Update (2026-10-10, EXT):** one more slot: the outside price shock per (node, commodity), a factor in basis points, set between steps by `SetOutsideShock` (name provisional), saved, not hashed, with the event `OutsideShocked` (`spec-external-sector.md` 6.5). An embargo is the edge status of `spec-regions.md` 7. A capacity slot is left open (EXT open question 6). See N-116.
- **Update (2026-10-10, MIG):** MIG has no slot. A forced move (a flight, an expulsion) is INT's call of the guarded `relocate` with a cause of its own (`spec-regions.md` 6.6, `spec-migration.md` 10.3); for a household on a migration journey INT must cancel the journey first (MIG has an internal cancel, `spec-migration.md` 8.4.1; its public name is INT's), because MIG registers the refusal `on_journey`.
- **Update (2026-10-10, MER):** MER has no world slot.

### N-092 Overlays and scenarios that the wave 2 specs hand to REF

- **Areas:** REF, STB, STY
- **Raised:** 2026-10-10, `spec-regions.md` 12.4, `spec-money-supply.md` 17.6, `spec-population.md` 14.4 and 18, `spec-transport.md` 15 and 17.
- **Note:** REF builds: the edge part of `overlay/trade.toml` (REG 12.4, with a `modules.transport.modes` table, a hired mode's haul commodity and a carrier recipe, TRN 17.4); `overlay/faucet.toml` and `overlay/faucet-sunk.toml` (MON 17.6); a population overlay (a fixed-factor test double or RES, the demand overlay on, `Renewal` 0; a warm start's census is not the Malthusian `N*`, POP 18). The new scenarios are S28 and S29 (REG), S30 to S32 (TRN), S33 to S36 (MON) and S37 to S40 (POP); S13 is defined. STY sets the bands for natural increase, recovery time and the money-limited population.
- **Update (2026-10-10, RES):** `spec-resources.md` 18 proposes `overlay/resources.toml` (an `arable` refill resource for `grow_staple`, a finite `deposit` for `dig_mineral`, a logistic `stand` for `cut_fuel`; the line gatherers stay unlimited), a slack and a binding data set sized from the solver's flows, and the scenarios RES-S1 to RES-S5 (S41 to S45, 19.4). Capacities in 18.2 are hand-derived; REF re-derives them from the solver.
- **Update (2026-10-10, wave 3 integration):** EXT's scenarios EXT-S1 to S5 (S51 to S55) need an external overlay on a gate region of `one` or `three` (`spec-external-sector.md` 18.2); INF's S46 to S50 need `[modules.information]` and an informed merchant role; INF-S1 (S46) and the rest of INF-S2 (S47) wait for MER.
- **Update (2026-10-10, SUB):** `spec-subsistence.md` 18.3 proposes the scenarios SUB-S1 to SUB-S5 (S56 to S60). REF builds an overlay `overlay/own-use.toml` (an own-use recipe on `one`, with a yield row and, for SUB-S4, a refill endowment) and uses `famine` for SUB-S2.
- **Update (2026-10-10, MIG):** `spec-migration.md` 17.2 proposes MIG-S1 to MIG-S6 (S61 to S66) and 15.3 an overlay `overlay/migration.toml` (`[modules.information]`, `[modules.migration]`, `move_speed` chosen against the trade overlay's edge lengths so that the shortest journey is 1 or 2 ticks and the longest under `CoverTicks`). REF builds it.
- **Update (2026-10-10, MER):** the trade overlay needs a merchant role, a consignee role and `[modules.merchants]` (`spec-merchants.md` 19.1); scenarios MER-S1 to MER-S6 (S67 to S72) are proposed in 17.

### N-093 Inputs that the wave 2 specs offer the viability check

- **Areas:** VIA, CFG
- **Raised:** 2026-10-10, `spec-regions.md` 4.5.3, `spec-transport.md` 9.4, `spec-money-supply.md` 15.4.
- **Note:** (1) `WithFreeTrade` links every pair of regions; with edges only regions in one strongly connected component can trade (`Graph.Components`), so the pooled check may use it (REG open question 2). (2) `transport.Lanes` derives lanes from hired modes with labour only; the carriers' feed and vehicle wear are not in the programme (TRN 9.4, 20.8), and VIA may want a lane with inputs. (3) With a rate levy a class's holdings are bounded by `m* = E + g/rho` (MON 9.3); VIA may seed a hoarder class at `min(Money, m*)` (MON 15.4), optional.

### N-094 Snapshot fields of the wave 2 modules

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-regions.md` 10.5, `spec-money-supply.md` 13.4, `spec-population.md` 10.4, `spec-transport.md` 13.5.
- **Note:** OBS names the fields. REG: the edges with a non-default status and the component count. MON: the money supply by sector, the money boundary balance per reason and each flow's state. POP: per region and role the mean birth and death rates, the natural increase, the birth carry and the unfed households of the last tick; per module `r*` and the gain. TRN: the shipments in flight, the away agents and the non-default transit factors.
- **Update (2026-10-10, RES):** snapshot fields per endowment: stock, effective capacity, abundance, `ExhaustedRun`, and the shock scales when non-default; with the population module on OBS may add `N_bind` where VIA has computed it (`spec-resources.md` 14.4). Names are OBS's. EXT's fields are in `spec-external-sector.md` 15.4 and INF has none beyond the event.
- **Update (2026-10-10, SUB):** snapshot fields: per region and role the households in the mode and the households blocked in the last tick; per agent `OwnUse`, `OwnUseJobs`, `Cover` and `Secure` (`spec-subsistence.md` 14.4). Names are OBS's.

### N-096 The regional pool serves births, entry, relief and money flows

- **Areas:** MON, POP, EST, SPN
- **Raised:** 2026-10-10, `spec-money-supply.md` 8.6 and open question 4, `spec-population.md` 4.3, 4.8 and 5.5.
- **Note:** Four things draw on or pay into a region's pool: relief (phase 1), money flows with the pool as a target (phase 1; a pool target needs the estates module, `spec-money-supply.md` 4.3.2; the estate account exists without it, the pool policy does not), entry and births (phase 7, inside `EntryPoolShare`, deaths first). A per-household grant couples issuance to the household count (MON-A5): POP decides whether income from a grant raises births. In a closed region the population is bounded by M / stake; a faucet or an inflow with migrants lifts the bound. MON and EXT should say how money entering a region reaches its pool.
- **Update (2026-10-10, EXT):** money that the outside pays reaches the households first (the sellers of the goods the outside buys) and the pool only through the estates (deaths, relief returns, sales by the estate); EXT adds no pool channel and no grant to a pool (`spec-external-sector.md` 11.5). A surplus therefore raises `M_h` before `M_p`, and a hoard of the outside's money is a hoard like any (STB-L31, L45; damper D42).
- **Update (2026-10-10, MIG):** the migration module draws nothing from any pool (as N-082); a migrant takes no stake and neither region's pool is touched (`spec-migration.md` 9.2).

### N-097 Migration and the population loop

- **Areas:** POP
- **Raised:** 2026-10-10, `spec-population.md` 6.5.
- **Note:** MIG is the other way households enter and leave. It must use the same stake rule for arrivals and not create money, say whether the replacement ratio is part of its signal, and share the pool budget `EntryPoolShare` with births. A closed region (no immigrant) is the behaviour tested before MIG. Relief raises the ratio and so births and lowers deaths (POP open question 4): the same effect as L40.
- **Update (2026-10-10, MIG):** applied for MIG in `spec-migration.md` 5.5 and 9: a migrant carries its account, so no household is created, no money is created and no stake is paid; neither region's pool is touched and `EntryPoolShare` is not shared (births and entry keep the whole budget); the replacement ratio is NOT part of the signal (the trigger is the household's own earned ratio against `MoveBelow`; a pack with both modules may set `MoveBelow` near `r*` by hand); the closed-region behaviour is unchanged. The money a migrant carries moves the money-limited bound `M / stake` of each region by what it carries (9.3, MIG-A5, L76). Still open for POP: its section 7 (the MIG row) and 6.5.

### N-099 A trip is a gap in income

- **Areas:** WEL, STB
- **Raised:** 2026-10-10, `spec-transport.md` 15 (TRN-A6) and 20.9.
- **Note:** The effective welfare window derives from recipes only (`spec-welfare-and-wages.md` 2.3.3), so a self-carried trip longer than the window looks like starving. The validation warning `trip_longer_than_window` is a stopgap; WEL decides whether the window counts the longest trip of a role (WEL open question 9, left open at the wave 2 integration). The working ticks of a self-carried trip run from dispatch to the tick before arrival.
- **Update (2026-10-10, MIG):** a migration journey is also a gap in income: `k` records of zero income and the origin's upkeep, not working ticks (`spec-welfare-and-wages.md` 4.3 already says so). The settling rule (`spec-migration.md` 5.2) waits a window before the record can trigger a move.
- **Update (2026-10-10, MER):** MER proposes the answer (`spec-merchants.md` 12.3): the effective window of a role with a merchant block is `max(WelfareWindow, WindowCycles * max(longest Duration, Kmax))`, `Kmax` the longest travel time over the graph of the self-carried modes of its block. Until WEL closes its open question 9, `trip_longer_than_window` is the only guard.
- **Update (2026-10-10, wave 4 integration, X4):** ruled in the one combined edit of `spec-welfare-and-wages.md`: the effective window (2.3.3) lists the recipes on the role's list, the effective own-use list and the longest trip of the role's modes; a recipe-less role with an own-use job, and a journey or a trip, are handled in 4.3; a migrant's journey is zero-income ticks in the window. The note stays open until WEL's open question 9 is closed.

### N-101 Does the default Target count Inbound?

- **Areas:** AGT
- **Raised:** 2026-10-10, `spec-transport.md` 9.1 and 20.7.
- **Note:** The field `Inbound` exists in the agent view. The default strategy never ships, but a consignee can be sent goods (TRN-A2): decide whether the default `Target` counts what is in flight to the agent.
- **Update (2026-10-10, MER):** the merchant counts its `Inbound` as a consignee (`spec-merchants.md` 8.2) and its `Outbound` as a shipper (7.1). The default strategy is never a consignee unless a consumer's strategy sends it goods, so its `Target` is still AGT's.

### N-102 Unify the module table names before the loader is written

- **Areas:** CFG
- **Raised:** 2026-10-10, the wave 2 integration (the specs of that wave were not asked to share one spelling).
- **Note:** The specs name the table of a module differently in the file: `[module.estates]`, `[demand]`, `[module.money_supply]`, `[population]` and transport's own. `spec-config.md` records each as its spec says and leaves this as an open question. Unify before the loader is written.
- **Update (2026-10-10, RES):** RES writes its module tables as `[resources]`, `[resources.resource.<id>]` and `[resources.endowment.<region>.<resource>]`, the way `[demand.want.<id>]` is written (`spec-resources.md` 3.3.2).
- **Update (2026-10-10, INF):** INF writes its table as `[modules.information]` (as transport's `modules.transport.modes`) and the role flag as `informed` in the role table.
- **Update (2026-10-10, SUB):** SUB writes `[modules.subsistence]` and, for a role row, `[modules.subsistence.role.<id>]` (`spec-subsistence.md` 3.2).
- **Update (2026-10-10, MER):** MER writes `[modules.merchants]` and the block `[role.<id>.merchant]` (`spec-merchants.md` 19.1), the form of `[modules.information]`.

### N-103 Running several spec writers in parallel

- **Areas:** ALL
- **Raised:** 2026-10-10, the wave 2 run (REG, MON, POP and TRN written at once, then integrated).
- **Note:** What worked, so the next run repeats it, and what bit.
  - Each writer had its own worktree and branch and could create only its spec file and `doc/specs/models/<code>/`. It handed its edits to shared files over as a written handoff (notes, parameter rows, glossary, ledger reasons, config fields, register entries, events and keys, edits to other specs, checkpoint state, the `Testing` and `Observability` entries). The shared files collide when several writers edit them: `notes.md`, `specs/README.md`, `roadmap.md`, `documentation_index.md`, `spec-ledger.md`, `spec-config.md`, `spec-stability.md` and the rules. After the writers finished, one supervisor wrote the rulings and the final numbering, and integrators, one per disjoint set of files, applied the handoffs. The merges were conflict-free.
  - Fix the interface between concurrent specs in the brief (who owns the region graph, the transit hook, the ledger reasons) and read the finished specs against each other. Two writers still disagreed with the specs around them: TRN had assumed an error that REG did not have, and POP showed that the spawner's signed hook could not work. The cheapest fix was a review message to the writer, with the owner of the interface winning. When one spec depends on another's interface, write the owner first if the schedule allows.
  - Writers use provisional IDs for register entries, dampers and scenarios (`REG-A1`, `POP-D3`, `TRN-S2`) and their proposals clash. The supervisor assigns the final loop, damper and scenario numbers (see N-046) and the note IDs, and each register entry names the provisional ID in brackets so the mapping stays traceable.
  - Checks to run mechanically after integration (the script lived in `/tmp` and is gone, so rebuild it with `perl` and `grep`, never the host Python): every `spec-x.md N.N` reference resolves (to a heading, a numbered paragraph or a table row); no requirement ID is defined twice; every file under `doc/` has a `documentation_index.md` row; the notes index equals the areas of the open notes; no `.pyc` is committed; the read-receipt tags changed only for the files that were edited. Run it on the base first: one reference was already broken (`spec-welfare-and-wages.md` to `spec-config.md` 13.5).
  - Traps. Say in the brief that models run only in Docker (three of four writers ran the host `python3` for small things anyway). A model run in Docker leaves a root-owned `__pycache__` that makes removing a worktree fail and invites a committed `.pyc`: set `PYTHONDONTWRITEBYTECODE=1`, or clean up with a container. A new worktree makes Claude Code ask whether to trust the folder, which is the user's decision. In herdr a pane ID contains a colon and an agent name must match `[a-z][a-z0-9_-]*`. On this machine `grep` is `ugrep` and rejects some patterns. Notifications from a monitor of an earlier session arrive as if new: check what the supervisor started before acting on one.
- **Update (2026-10-10, RES):** RES followed the process: own worktree, only the spec and `models/res/` created, shared-file edits handed over. Slip: the host `python3` was called twice (an empty stdin script and `--version`), nothing else; all models ran in Docker with `PYTHONDONTWRITEBYTECODE=1`.
- **Update (2026-10-10, EXT):** wave 3 followed the method. What bit: a model run first assumed a price rule that made the cobweb vanish (a midpoint with cost-based asks has a fixed point), so the cobweb model was rewritten around a uniform price and says so; a numeric claim copied from a first model run had to be recomputed when the cap was redefined on list value. Re-run the arithmetic of an example after any change of definition.
- **Update (2026-10-10, wave 3 integration):** wave 3 (RES, INF and EXT written at once, then integrated) added two lessons. (1) A seam between two concurrent specs stayed invisible to both writers: INF described a function `Informs` that EXT was to call, and EXT described an `OutsidePrice` read for INF; neither writer saw the other's side until the supervisor compared the finished specs. The supervisor's review fix removed both from EXT (no requirement ID reused or renumbered), and the decision that an outside node is not a rumour source in version 1 (N-112). (2) The fix reached only one side: after it `spec-information.md` still describes the declaration (4.5, FR-INF-002 and others), so when a review removes one half of a seam, send the message to the other writer too and grep the other spec for the removed name. RES was also revised after an independent review of its spec (commit 883eff5); the findings it accepted are written into the spec.
- **Update (2026-10-10, supervisor, wave 3):** what the supervisor learned running wave 3, for the next run.
  - Count the register IDs from the spec files, not from the handoff prose: grep each spec for its provisional `<CODE>-A`, `-D` and `-S` IDs, then decide the merges. EXT's handoff text implied four scenarios and its spec had five, which would have made two final numbers collide. The final numbers must be fixed before the integrators start, because the notes editor needs them before the stability editor has assigned them.
  - Look at a worker pane's footer before sending briefs. Claude Code prints a usage warning there ("You've used 91% of your weekly limit"). A run of three writers and four integrators is large; if usage is high, ask the user before launching. A writer that stops at the limit leaves its branch and handoff on disk, so work is not lost, only stalled.
  - Choosing integrators: reuse the writers' panes for the files whose edits need the spec's context (they finished with no conflicts), and give the long reads (`notes.md`, the roadmap and the indexes) to a fresh agent on a new branch off `main` when a writer's context is high (the RES pane was at 70%).
  - Add to the mechanical checks: every spec file on `main` has an `exists` row in the roadmap (the tier table and the existing-specs table), a row in the code table of `specs/README.md` and a documentation-index row; and read the roadmap diff, because the script cannot see a count that is out of date. The user requires the roadmap to be updated in the same change that closes a spec.
  - Re-read what an integrator wrote in the rules and in any spec it edited. In wave 2 an integrator passed text through shell quoting and left the literal sequence `'"'"'` where an apostrophe belonged, in `rules/observability-and-logging.md` and `spec-ledger.md`; it was found only on the next re-read. Grep the whole of `doc/` for it after every integration (`grep -rF "'\"'\"'" doc`).
  - Merging without extra branches (a standing preference of the user: no branches beyond `main`): merge the integrators' branches in a detached worktree (`git worktree add --detach <path> main`), run the checks there, fast-forward `main` to the verified commit, then remove the worktrees and delete the merged branches. Before removing a worktree check for uncommitted changes and for root-owned files (a Docker `__pycache__`), or the removal fails halfway and leaves an empty workspace.
  - A launcher script that splits a pane ID such as `w4:p1` on the colon sends the brief to a target that does not exist (`agent target w4 not found`); the first launch of wave 2's integrators sent nothing. Read the result file of every prompt before waiting.
- **Update (2026-10-10, wave 4 integration):** the seam check of the brief worked: the writers of SUB, MIG and MER read one another's drafts and fixed their mismatches while all three existed (MIG expected an "ending reason" of SUB that SUB does not have; MER adopted POP's Weyl offset instead of inventing a second spread). Slip: the host `python3` was called with no-op scripts; every model ran in Docker with `PYTHONDONTWRITEBYTECODE=1`.

### N-104 Region-keyed tables use the one override mechanism

- **Areas:** LND
- **Raised:** 2026-10-10, `spec-regions.md` 5.1 and 5.3.
- **Note:** A new table keyed by region (a yield, a price, a scale, a cap, a rate) uses the mechanism of `spec-regions.md` 5.1: one row per key in the owner's table, combined with the default by replace, add or scale, in canonical order by `RegionID`, with the errors of 5.3, and a row in the list of 5.1.3. `RegionPrices` and POP's `Regions[]` are the two examples. Do not invent a per-region field with its own rules. The same holds for SET, CAP and later specs.
- **Update (2026-10-10, RES):** applied for RES in `spec-resources.md` 3.3.1: `Endowments` is a region-keyed table of the mechanism of `spec-regions.md` 5.1, one row per (region, resource), `Capacity` and `Initial` by replace, `RegenScaleBP` by scale, absence is the default (a resource's default capacity is 0). The row in the list of `spec-regions.md` 5.1.3 is added at the wave 3 integration. SUB, MIG and LND stay.
- **Update (2026-10-10, INF):** INF adds no region-keyed table. The speed of news is one world value; a slower region is expressed by longer distances; a per-region speed would be a row of `spec-regions.md` 5.1.3 in scale mode keyed by the reader's region (`spec-information.md` 17 (open question 9)).
- **Update (2026-10-10, EXT):** EXT's tables follow the one mechanism: the node rows are keyed by region, the line rows by (region, commodity), one row per key, in canonical order, with the errors of `spec-regions.md` 5.3; the gate scale (`OutsideGate.ScaleBP`) is a **scale** row (`spec-external-sector.md` 3.2.4). Rows are added to the list of `spec-regions.md` 5.1.3 at the wave 3 integration.
- **Update (2026-10-10, SUB):** applied for SUB in `spec-subsistence.md` 3.2.1 and 3.5: the module has no region-keyed table and no per-region field. Its role-keyed rows follow the replace mode (one row per role, absence is the default, an empty row means none); regional differences are rows of the yield table and the endowment rows, which already follow the mechanism.
- **Update (2026-10-10, MIG):** applied for MIG in `spec-migration.md` 11.2: `Regions` rows with `OutShare` and `InShare`, one row per region, by replace, in region order, with the errors of `spec-regions.md` 5.3; a row is added to the list of `spec-regions.md` 5.1.3.
- **Update (2026-10-10, MER):** MER adds no region-keyed table. A lane is not a row. LND stays.

### N-107 New money faucets and sinks start from MON

- **Areas:** FIS, CRD, MON, LED
- **Raised:** 2026-10-10, `spec-money-supply.md` 4.5, 9.5 and 10.1.
- **Note:** Start from MON's mechanism: a grant or a levy on a class, a declared reason equal to the flow ID, closed for a run, with its boundary account created at construction, reported in `BoundaryFlow` and checked by the audit. Add a built-in ledger reason only for a mechanism that is not a grant or a levy (an interest flow, a trade with an external counterparty). MON's positions bind the later specs: no faucet reads an aggregate or grows with a balance (only a sink may be proportional to a balance), and a counterparty whose balance is created or destroyed outside conservation is a Boundary account. Tax, credit and the external sector are the likely first uses. A policy that changes a rate over time is PLC's, through the flow-scale slot (N-091).
- **Update (2026-10-10, RES):** RES adds no faucet, no sink and no ledger reason: a draw and a regeneration are not transfers and an endowment is in no account (`spec-resources.md` 6.2.1). Depletion is made visible by events and an exact per-endowment identity (6.4), which a test and the audit check.
- **Update (2026-10-10, EXT):** applied for EXT: a trade with an external counterparty is the case the note names for built-in reasons (`spec-external-sector.md` 8.1); the counterparty's account is a Boundary account's partner and holds nothing across a tick (5.3); no rule reads an aggregate (11.3); a policy that changes trade acts through the shock slot (6.5), the edge status (`spec-regions.md` 7) or a flow scale.
- **Update (2026-10-10, SUB):** applied for SUB: no faucet, no sink, no ledger reason; the form of the output is the ordinary recipe execution, and the in-kind channel was refused with reasons (`spec-subsistence.md` 6.2).
- **Update (2026-10-10, MIG):** MIG adds no faucet or sink (`spec-migration.md` 9.1).
- **Update (2026-10-10, MER):** the delivery contract moves money between non-boundary accounts and adds no faucet, no sink and no reason (`spec-merchants.md` 9.6.1); the three escrow reasons already in the registry change owner to MER.

### N-108 What the resources module leaves to land, subsistence, infrastructure, the outside and migration

- **Areas:** LND, CAP, EXT
- **Raised:** 2026-10-10, `spec-resources.md` 10, 9.4 and 23.
- **Note:** Endowments are open access: any recipe with `Draws` on an endowment of its region may take from it, and the ration shares a scarce stock (5.3). LND adds the tenure through three hooks: a refusal after `no_endowment`, a per-claimant entitlement applied to the request before the ration, and a rent as a named money flow valued at the dual of the resource row (`Rent`, `spec-resources.md` 11). SUB: household production may draw on an endowment through `Draws` like any recipe. CAP: a facility that raises capacity uses the capacity-scale slot (6.3). EXT: the import predicate is the second half of the sourcing of a raw good (9.4). MIG: nothing in RES moves households; the signal of a depleted region is the welfare ratio.
- **Update (2026-10-10, SUB):** applied for SUB in `spec-subsistence.md` 9: an own-use recipe draws on an endowment through `Draws` like any recipe, open access, the ration of `spec-resources.md` 5.3, no priority and no quota; its requests carry the recipe, so LND's entitlement may treat them differently, and a closed commons makes the household `Blocked` and draws `own_use_not_available` (9.5). LND, CAP and EXT stay.
- **Update (2026-10-10, MIG):** applied for MIG in `spec-migration.md` 4.4 and 6.3.3: nothing in RES moves households; the signal of a depleted region is the household's own earned ratio and measured wage; a destination's stock, endowment or abundance is state and is not read by an estimate (the estimate uses the table value of the yield). LND, CAP and EXT stay.

### N-109 The closed forms of the Malthusian loop need the welfare ratio at a full draw

- **Areas:** POP, VIA, OBS
- **Raised:** 2026-10-10, `spec-resources.md` 8.2, 8.5 and 23 (item 6).
- **Note:** `N* = N_bind * r_full / r*` (flat curve) and the stock share `a(sigma*) = 10 000 r* / r_full` (curve) need the welfare ratio a household reads at its full draw and full abundance, `r_full`. It is a property of the pack's economy and not a field. Decide whether a run measures it (a snapshot figure), whether VIA computes it from the programme with wants, and whether a warning `population_cannot_grow` (`r_full <= r*`) belongs to POP or VIA.
- **Update (2026-10-10, SUB):** with the module on the welfare ratio of a floor-fed household includes its receipts (`spec-subsistence.md` 7.4): `r_full` for such a household is the floor's ratio, below the replacement ratio of a market-fed one, so the schedules form fewer households and dissolve more among them.

### N-110 Version 1 rumours prices only

- **Areas:** SKL, INF
- **Raised:** 2026-10-10, `spec-information.md` 17 (open question 4).
- **Note:** MIG's rumoured welfare ratio (`spec-welfare-and-wages.md` 5.1) and SKL's rumoured premium (`spec-skill-barriers.md` 2.5) have no source: they are regional aggregates, so whether an agent may react to one is a question for "macro is measured, never set" (roadmap). Decide in `spec-migration.md`.
- **Update (2026-10-10, MIG):** decided for MIG in `spec-migration.md` 4: a household does not react to a regional mean welfare ratio (an indicator, a controller input in disguise); it reads its own state, the structure of the world (graph, recipes, yield table, upkeep lists) and rumoured PRICES, from which it derives a destination's income (6.3). INF 17.4 is answered. The same decision applies to SKL's rumoured premium (`spec-skill-barriers.md` 2.5), which stays open for SKL.

### N-111 A view takes draws

- **Areas:** WLD, AGT, OBS
- **Raised:** 2026-10-10, `spec-information.md` 5.5.2.
- **Note:** The view of an informed agent is built immediately before its `Trade` and, with `RumourErrorBP` above 0, takes draws from the world's source in a fixed order (one per candidate). AGT's view building and WLD's phase 3 must keep that order; the draw count is a function of the configuration and of the informed agents present. `spec-agents.md` 6.1.2 (the stream a strategy sees is a function of the world's state) still holds because the order is fixed. OBS: `RumoursServed` and its metric.
- **Update (2026-10-10, MIG, X3):** a second reader. The migration pass of phase 7 reads rumours for the household under review through INF's read function, whatever the role's `Informed` flag says, and takes INF's error draws itself: among the reviewed households in ascending agent ID, one draw per source pair (candidate) whether or not a rumour is offered, none when `RumourErrorBP` is 0 or no agent is pushed (`spec-migration.md` 6.2.1, 10.2.1). They belong to the migration pass of phase 7 and never to a view of phase 3, so INF's stream (before each `Trade`) and MIG's (after the entry step) cannot interleave, and the draw count in a tick stays a function of the configuration and of which agents are pushed. `spec-information.md` (the reader, 5.5.2, 7, and the requirement that rumours reach only `Informed` views) carries the matching edit. AGT, WLD and OBS keep the fixed order of both; `RumoursServed` does not count MIG's calls.

### N-112 An outside node as a remote information source

- **Areas:** EXT, INF
- **Raised:** 2026-10-10, `spec-external-sector.md` 9.2 and open question 10, `spec-information.md` 4.5 and 17.
- **Note:** In version 1 an outside node is not a rumour source: INF records only the published price of a market and a node has no market, and EXT has no `Informs` declaration and no outside-price field (FR-EXT-033). A merchant away from a gate therefore never hears of an outside price. The minimal shape, if a setting needs it, is a second record source in INF (a quote function that EXT supplies as a pure read) and an EXT declaration of the (region, commodity) pairs it serves. The INF hook `Informs` stays reserved and unused. `spec-information.md` 4.5, 4.5.2 and FR-INF-002 still describe a declaration by EXT and must be read as reserved until INF is revised.
- **Update (2026-10-10, MER):** the market of a gate region is a rumour source like any market, so an interior merchant hears of the gate's published price (which includes the outside's trades) with the lag and the error of any rumour; only the node itself is not a source (`spec-merchants.md` 11.3).

### N-113 A quote is a limit, not a peg

- **Areas:** CLR, STB, STY, EXT
- **Raised:** 2026-10-10, `spec-external-sector.md` 6.7.
- **Note:** A counterparty's trade executes at the midpoint between its limit and the realm's, so the realm's price is pulled toward the quote and not held at it; the outside keeps the price improvement. A trade at the counterparty's own limit needs a flag on `Order` and a rule in `spec-clearing.md` 5.1 and 8. Run scenario EXT-S1 (S51: the pull of beliefs toward the quote at the default clearing); if it is too weak for a setting that wants a fixed price, CLR adds the flag and the rule.

### N-114 The open layer of the steady state

- **Areas:** VIA, REF
- **Raised:** 2026-10-10, `spec-external-sector.md` 12.1.
- **Note:** VIA adds optional import and export columns per gate and side (bounded by `Capacity` times the mean window), priced at the quotes at tick 0 with the gate scale, and a trade row (imports at the quotes not above exports at the quotes); EXT supplies `Lines()` as a pure read of the configuration. An import counts as a source in the gate regions for V1. REF builds an external overlay (EXT-S5, S55; N-084).

### N-115 The boundary flows of the outside are gross

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-external-sector.md` 5.2.3 and 8.1.
- **Note:** `BoundaryFlow` shows the budget and the part returned, not the net. The net is derived (`import_c = outside_offer - outside_unsold`, `export_c = outside_purchases`, `export_value = outside_budget - outside_unspent`, `import_value = outside_receipts`). OBS decides whether to add a net event or metric of the balance of trade and names the Snapshot fields of `spec-external-sector.md` 15.4.

### N-116 The outside's slots

- **Areas:** INT
- **Raised:** 2026-10-10, `spec-external-sector.md` 6.5 and 7.5.
- **Note:** INT names `SetOutsideShock` (node, commodity, basis points, between steps) and uses `SetEdgeStatuses` for an embargo. A capacity multiplier (a partial embargo) is not provided; decide when a consumer asks. Part of the slots of N-091.

### N-117 The agent ID space and the account order with an outside

- **Areas:** AGT, EST, WLD, LED
- **Raised:** 2026-10-10, `spec-external-sector.md` 4.2.
- **Note:** With the module on the estates take the lowest agent IDs (`0..n-1`, equal to the `RegionID`), then the counterparties (ascending node, gate, buy before sell), then the households, so adding a node or a gate shifts every household's ID, tie-break keys and checkpoints, as adding a region does. WLD fixes the construction order (the estate accounts, the Outside accounts, then the agents) and the positions of N-009; AGT and EST restate FR-AGT-001 and FR-EST-005.

### N-118 One job per agent limits the subsistence floor

- **Areas:** PRO, WEL, SPN
- **Raised:** 2026-10-10, `spec-subsistence.md` 5.7, 10.3, 8.3 and 21 (items 1, 4, 7).
- **Note:** A household in a long role job cannot take the own-use mode until the job ends, which is the lean season of a long-job role; a cohort cannot work an own-use recipe and a role recipe at once; the default spawner's rung 1 abandons a running own-use job. Decide, with the first scenario that shows the case, between a second light job slot, an interruption with an abandon reason, and a rule in the spawner that skips a household with a running own-use job.

### N-119 The receipts of a household that feeds itself

- **Areas:** WEL, EST
- **Raised:** 2026-10-10, `spec-subsistence.md` 7.5, 8.4 and 21 (items 2, 8).
- **Note:** The units of an own-use job are counted in the welfare ratio at creation, and a sale of the surplus is counted again as cash; relief may pay a household whose ratio is below the line only because its floor covers less than its upkeep. Decide a provenance, a consumption-side count, or accept the bound.

### N-120 A migrant carries its whole account for free

- **Areas:** TRN, MER, STO
- **Raised:** 2026-10-10, `spec-migration.md` 9.4 (MIG-A9).
- **Note:** TRN limits a traveller's load by a mode's capacity (`spec-transport.md` 3.2.2); a relocation has no capacity. The default strategy keeps holdings small, so the leak is small, but a consumer's strategy that stockpiles could ride a journey for free carriage. Close it with a carry limit (mass and bulk per household) if a case appears.

### N-121 A role with no recipe: merchants, the floor and migration

- **Areas:** MIG, SUB
- **Raised:** 2026-10-10, `spec-merchants.md` 6.4 and 12.4.
- **Note:** SUB lets a role with no recipe have an own-use list; the default `Work` a merchant inherits then starts an own-use job and the shipment mechanism refuses a trip (`job_in_progress`). MER keeps the merchant from planning a trip meanwhile and while it is in the own-use mode (the SUB qualifier on "a role with no recipe requests no job" is a supervisor erratum of `spec-merchants.md` 3.3). MIG ends the review of a no-recipe role in `no_estimate`, so no merchant is moved by MIG in version 1; if MIG gains an income estimate for such roles it must leave a merchant alone, since a merchant's place is where its trade is.

## Closed notes

When a note is closed, move it here and add **Closed:** with the date, the
resolution (applied or rejected, and why) and where it was settled.

### N-023 Calendar

- **Areas:** CAL, WEL
- **Raised:** 2026-10-09.
- **Note:** `spec-calendar.md` builds on `TicksPerYear` and does not redefine it
  (`spec-currency.md` 2.4). Per-tick quantities (upkeep, the reference basket) are
  not rescaled when the tick length changes. `ExpectedWorkTicks` is an expectation
  for calibration and the skill entry rule, and must be at most `TicksPerYear`
  (`spec-welfare-and-wages.md` 3.3, 4.2).
- **Closed:** 2026-10-10. Applied in part, rejected in part. Applied: per-tick
  quantities and durations are not rescaled (`spec-calendar.md` 6.1, 6.2) and
  `ExpectedWorkTicks` is at most `TicksPerYear` (6.3). Rejected: the first sentence.
  To make time configurable, the calendar defines `TicksPerYear` as a derived
  quantity (`spec-calendar.md` 3.2) and `spec-currency.md` 2.4 now says so, so it is
  no longer a separate world parameter.

### N-003 Non-tradable goods, disposal and the recipe format

- **Areas:** PRO, GDS, LED
- **Raised:** 2026-10-09, `spec-goods.md` open question 4 and `vision.md` goal 5.
- **Note:** Open for the production spec: a byproduct with no buyer, a waste
  product, whether a `Tradable` flag and a disposal sink are needed. Quality
  grades are separate commodities (`spec-goods.md` 7.1), so the recipe format must
  not assume a grade. Still undecided: variable yield, and conditions on a recipe
  (a season, a building), which touch `spec-calendar.md` and
  `spec-infrastructure.md`. Decide the recipe format against a second, unrelated
  setting, as `vision.md` section 5, item 3 requires.
- **Closed:** 2026-10-10. Applied in `spec-production.md` 7 and 3.2: no `Tradable` flag (rejected for version 1); an output may be `Discard` (created and disposed in one step, measured in flow totals); a `disposal` goods sink is needed in the ledger registry (N-033); byproducts are outputs with `CostShare` 0. Variable yield and conditions: deterministic yield factor by region and season, season read at job start; building conditions and random yield deferred with costs in 6.7. The recipe format assumes no grade and was checked against two unrelated settings (section 12). `spec-goods.md` open question 4 is answered by `spec-production.md` 7.5.

### N-004 What production takes on from other specs

- **Areas:** PRO
- **Raised:** 2026-10-09, collected while writing GDS, SCL and LED.
- **Note:** The production spec inherits these, already stated elsewhere:
  - it owns `Requires` on a recipe and `wear_per_use`; GDS owns `Durability`
    (`spec-goods.md` 5.3), and wear falls on the most worn item first
    (`spec-ledger.md` 5.4);
  - a recipe execution is one all-or-nothing ledger transform; a recipe with no
    inputs is a source (`spec-ledger.md` 8.1, 8.2; `spec-goods.md` 6.2);
  - an agent of weight `w` runs up to `w` executions, and `k` if inputs cover
    only `k` (`spec-scale.md` 4.2, FR-SCL-007, FR-SCL-008);
  - a durable no recipe requires must warn (FR-GDS-023);
  - which recipes are trainee-eligible and how large their output is sets
    whether training pays (`specs/README.md`, "Still open" 1).
- **Closed:** 2026-10-10. `Requires` and `WearPerUse` are owned in `spec-production.md` 4.4 (wear on distinct items, most worn first; it asks the ledger for `WearItems`, N-033); one all-or-nothing transform and source recipes in 4.3 and 3.1.2; weight `w` runs up to `w`, `k` if inputs cover `k`, in 4.2; the durable-not-required warning is FR-PRO-040; the trainee-eligible lever is in section 8. The chain's actual values stay open (`specs/README.md`, "Still open" 1).

### N-005 How chance events resolve for a cohort

- **Areas:** PRO, SKL
- **Raised:** 2026-10-09, `spec-scale.md` open question 3.
- **Note:** A per-household chance event for an agent of weight `w` must have the
  right expected count with integer arithmetic and the world's random source
  (`spec-scale.md` 4.6, FR-SCL-012). The recommended method is a carried
  remainder with a random starting offset. Tool breakage is no longer random (it
  is a durability counter, `spec-skill-barriers.md` 2.1), so this may only matter
  for other events. Decide whether the method is used at every weight or only
  above 1.
- **Closed:** 2026-10-10. Settled for production and skill barriers in `spec-production.md` 6.6: yield fractions use a deterministic carried remainder at every weight (including 1), started from one seeded draw per agent at creation; production draws no chance trials; tool breakage is a counter (`spec-skill-barriers.md` 2.1). The belief gate is a decision with one draw per outcome whatever the weight, not a per-household chance event (`spec-beliefs.md` 9.3). A future spec with a true per-household chance uses the helper of `spec-scale.md` 4.6. This also settles `spec-scale.md` open question 3 for production.

### N-006 What price beliefs must do with clearing's outputs

- **Areas:** BEL
- **Raised:** 2026-10-09, `spec-clearing.md` sections 5.4, 5.5, 7.3 and 13.
- **Note:** `spec-beliefs.md` must:
  - use the **price age** and the **best unmatched quotes** to move off a stale
    price, since clearing publishes a carried-forward price when nothing trades;
  - own the **belief step** that sets the resolution floor
    (`spec-currency.md` 3.1; named in `spec-config.md` 7.4), bound its steps, and
    adjust asynchronously;
  - fix the bazaarBot defects assigned to it (`bazaarbot-reference.md` section 11):
    7 (the lookback window), 9 (a threshold that depends on the price scale; use
    basis points of the mean), 10 (the range can invert; define the invariant) and
    11 (update per order outcome, once, not per match; `spec-clearing.md` 7.3).
  - the Doran and Parberry paper has not been read (N-015).
- **Closed:** 2026-10-10. Applied in `spec-beliefs.md`. Price age and the best unmatched quotes move a belief off a stale price (5.4, FR-BEL-011, FR-BEL-012, FR-BEL-034). `BeliefStep` is the belief step and sets the resolution floor `ceil(10 000 / BeliefStep)` (5.3, 11, FR-BEL-029); steps are bounded to `2 * BeliefStep` and asynchronous (5.2, 5.6). Defect 7: a per-market window, no per-agent history (7). Defect 9: `BeliefSignificance` in basis points of the centre (5.4). Defect 10: centre and half-width, invariant in 3.3. Defect 11: one update per admitted order (5.1.2). The Doran and Parberry paper is now read (journal version, `specs/evidence.md`); it uses no window for an agent's own range and its pseudocode has no crossing check.

### N-002 Does an agent have an inventory capacity?

- **Areas:** AGT, LED
- **Raised:** 2026-10-09, writing `spec-goods.md` (open question 3).
- **Note:** bazaarBot bounded an agent's whole inventory (`max_size`) and made it
  discard goods when full (`bazaarbot-reference.md` sections 3 and 10). GDS gives
  `Mass` and `Bulk`; whether an agent has a capacity in version 1 is undecided.
  If it does, a discard needs a goods sink reason in the ledger registry
  (`spec-ledger.md` 7.2) and an event, and the choice of what to discard is a
  strategy decision, not a rule.
- **Learned (2026-10-10, `spec-production.md` 5.3.2):** production never refuses an output for lack of room; overflow beyond `MaxQuantity` is disposed with reason `disposal`. If AGT adds an inventory capacity, a full inventory is the strategy's problem on the next tick, not a reason to lose output at completion. `disposal` is available to AGT for discards.
- **Closed:** 2026-10-10. Applied in `spec-agents.md` 8: no inventory capacity in version 1 (bazaarBot's `max_size` is the source of defects 6 and 8, storage is later, production never refuses output, overstock is limited by spoilage, the buffer cap, the production halt and the belief stock rule). A discard is a strategy decision executed as a move to the `disposal` boundary, so it is counted in `BoundaryFlow`; the default never discards. `spec-ledger.md` needs nothing beyond `disposal` (N-033).

### N-007 What the strategy takes on

- **Areas:** AGT
- **Raised:** 2026-10-09, `spec-clearing.md` section 3 and `spec-scale.md` section 4.
- **Note:** Order sizing is the strategy's job: the market rejects an unaffordable
  order whole and never clamps it (`spec-clearing.md` 3.2). One order per
  commodity per tick (3.1.1). A hard liquidity constraint, and a consumption rule
  that saves when wealth is low (roadmap, AGT row). Define the tradable set
  explicitly and a missing ideal stock as 0 (`bazaarbot-reference.md` item 5).
  The default strategy receives the world's random source and may not keep it
  (architecture.md 7.7). Quantities of an agent of weight `w` are `w` times the
  household amounts, and a limit price is never scaled (`spec-scale.md` 4.4).
- **Closed:** 2026-10-10. Applied in `spec-agents.md`: order sizing by the strategy with a budget that never gets an order rejected (7.4, FR-AGT-044); one order per commodity per tick (7.3.3); the hard liquidity constraint and the saving rule (7.4); the tradable set explicit (3.4) and a missing ideal as target 0 (7.3.1); the strategy gets the world's random source and may not keep it (6.1.2); quantities scale by the weight and a limit price never (9.1, 9.2).

### N-014 Stability register entries to collect

- **Areas:** STB
- **Raised:** 2026-10-09; each spec left a section for it.
- **Note:** Entries are waiting in `spec-clearing.md` section 13,
  `spec-scale.md` section 10, `spec-config.md` section 12, `spec-goods.md`
  section 10 and `spec-ledger.md` section 14. The roadmap's first entries are the
  target-income labour loop, the occupation-choice cobweb, and the delays in
  production, travel and traineeship (roadmap, STB row). Also record that a cohort
  multiplies the lockstep of the target-income rule (`spec-scale.md` 5.1) and that
  whole-lot spoilage makes a sawtooth (`spec-goods.md` 4.5).
- **Entries ready (2026-10-10):** `spec-beliefs.md` section 17 has nine (price/belief/limit loop, stale price loop, cobweb, herding and lockstep including the cohort gate, cost-push ratchet, shortage with no supply, width collapse, hard floor with perishables, reference band). `spec-production.md` section 16 has its own. The register itself is not written.
- **Closed:** 2026-10-10. Applied in `spec-stability.md`: all entries waiting in `spec-clearing.md` 13, `spec-scale.md` 10, `spec-config.md` 12, `spec-goods.md` 10, `spec-ledger.md` 14, `spec-beliefs.md` 17 (nine), `spec-production.md` 16 and `spec-calendar.md` 9 are in section 4, merged where two specs named one loop; loops the other specs leave unlabelled (WEL, SKL, EST, CAR) are added; the cohort lockstep of the target-income rule (L34) and the whole-lot spoilage sawtooth (L25) are recorded. The roadmap's first entries are L14 (target-income), L36 (occupation cobweb) and L21 (production, travel and traineeship delays). Future specs add entries by the checklist of section 9 (FR-STB-002, FR-STB-003). The thirteen entries of `spec-agents.md` 18 (AGT-A1 to A13) still have to be merged into `spec-stability.md` 4.7 (see N-059).

### N-037 What the strategy must supply to beliefs

- **Areas:** AGT, PRO
- **Raised:** 2026-10-10, `spec-beliefs.md` 4.4 and 10.
- **Note:** Beliefs need three inputs per order from the strategy or production: `Stock`, `Target` (units the agent wants to hold, from its recent demand over a cover period; 0 means none) and `UnitCost` (money per unit, 0 unknown). AGT must define `Target` and the cover period. PRO supplies `UnitCost` per (region, commodity) through `internal/production.UnitCost` (`spec-production.md` 9.4: input cost of one unit, rounded up, the minimum over recipes that can run there, byproducts and labour excluded) and the root hands beliefs a plain `int64`; the two specs agree on this. AGT creates a belief for every commodity in the tradable set at creation (`spec-beliefs.md` 3.1), draws limit prices with `beliefs.Draw` or its own rule, and uses the market price window (7) for quantity favourability. Watch: "a missing ideal stock as 0" (N-007) now has two readings, a zero ideal quantity for sizing and `Target = 0` meaning no stock signal for beliefs.
- **Closed:** 2026-10-10. Applied in `spec-agents.md` 7.3: `Target` is the units to hold after this tick's upkeep, `U * CoverTicks` plus the next job's inputs for a commodity used, `ceil(Make * OutputCoverTicks)` for one only made, 0 otherwise; cover periods `CoverTicks`, `InputJobs`, `OutputCoverTicks`; the `Stock` passed to `Revise` is stock after the trades less this tick's upkeep reserve (6.4.4); `UnitCost` is passed only for a commodity the role can make (7.3.6); beliefs for the whole tradable set at creation (3.4.1). The two readings of a missing ideal are one rule (7.3.1).

### N-033 Ledger changes production needs

- **Areas:** LED
- **Raised:** 2026-10-10, `spec-production.md` 4.4.2, 7.3, 15.
- **Note:** (1) add reason `disposal` (goods sink, agent to boundary, owner PRO); (2) allow `Transform` with an empty input list or an empty output list, not both (start step = inputs only, completion = outputs only); (3) add `WearItems(account, commodity, n, units)` that wears the `n` most worn distinct items by `units` each, destroying those that reach 0, with the same semantics as `Wear` otherwise (needed for a cohort of `k` executions); (4) the invariant "goods conserved except by recipes" holds with work in progress: pending output is in no account, so the snapshot reports it.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** accept all four. Not yet applied to `spec-ledger.md`; do it in the next change that touches that spec.
- **Closed:** 2026-10-10. Applied in `spec-ledger.md`: the `disposal` reason and the goods form of `initial` (7.2), a `Transform` with an empty side (8.2.1), `WearItems` (5.4.3) and work in progress (8.2.2); FR-LED-037 to 049.

### N-053 Ledger query for expiring lots

- **Areas:** LED
- **Raised:** 2026-10-10, `spec-agents.md` 7.3.5.
- **Note:** The default `Trade` offers units that spoil at the end of the tick. It needs a read-only query `Expiring(account, commodity, tick)`; `spec-ledger.md` 5.3.2 keeps an earliest-expiry hint per account but no per-commodity query.
- **Closed:** 2026-10-10. Applied in `spec-ledger.md` 5.3.3: the read-only `Expiring(account, commodity, tick)`.

### N-050 Spending classification for the welfare income

- **Areas:** WEL, AGT
- **Raised:** 2026-10-10, `spec-agents.md` FR-AGT-057.
- **Note:** `spec-welfare-and-wages.md` 2.1 nets inputs out of income. AGT's `Intent.Purpose` (`Upkeep`, `Input`, `Tool`, `Other`) classifies each bid; the world totals money received and money spent by purpose per agent per tick. A commodity that is both an input and upkeep is classed as an input. WEL decides what is netted (and whether tools are capex or recipe cost) and amends its 4.3 per N-035.
- **Closed:** 2026-10-10. Applied in `spec-welfare-and-wages.md` 2.1: `income_t` nets money spent on bids with the purpose `Input` and the wear cost of tools; upkeep, tool purchases and other spending are not netted; a tool is capital whose use is netted as wear (2.1.3); relief and in-kind count as income, a starting-money grant does not; a commodity that is both input and upkeep is classed as input and the ratio is understated by at most the upkeep bought of it (2.1.5, conservative).

### N-055 The belief cost ceiling against the equilibrium price

- **Areas:** BEL, PRO, STB
- **Raised:** 2026-10-10, `spec-viability.md` 8.2.3 and 15.
- **Note:** `UnitCost` (`spec-production.md` 9.1) excludes labour. At zero-profit prices the seed is 3.7, 9.7, 8.4 and 19 times unit cost for flour, dough, loaf and a tool in the prototype, against `CostCeiling` of 3 times (`spec-beliefs.md` 3.2.2). BEL clamps a belief to the ceiling, so sellers could not price at the wage and the seed would be clamped at creation. Options: count recipe labour at the wage in `UnitCost`; raise `CostCeiling`; drop the ceiling where the labour share is high. The solver reports `SeedAboveCostCeiling` meanwhile.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** count labour in `UnitCost` at the observed wage (`spec-welfare-and-wages.md` already defines an observed wage), measured from the previous period so there is no circularity. Reason: Lengnick's price bounds are multiples of marginal cost, and his marginal cost includes wages; the ceiling then stays near the equilibrium price instead of far below it. Cost: with labour counted, `CostFloor` 10 000 binds at the zero-profit price, which makes the freeze of N-039 more likely, so the floor must be re-examined together. Not applied: it changes `spec-production.md` 9 and `spec-beliefs.md` 3.2.2, and needs a revision pass of both specs.
- **Closed:** 2026-10-10. Resolved in design. `spec-production.md` 9 counts labour at the pooled observed wage of the previous tick (FR-PRO-035 to 037, 047 to 049); `spec-beliefs.md` 3.2.2 sets `CostFloor` 7 000 and `CostCeiling` 30 000 on the full cost, because a 100% floor freezes markets with perishables (`spec-stability.md` 6.2 run 5); `spec-viability.md` 8.2.3 and 15 re-ran the seed check against the full unit cost: ratio 1.000, minimum 0.991 (whole-unit rounding). A starting wage per region closes the first-ticks gap (`WarmStart.Wages`, `spec-viability.md` 8.2.5). Residual: the wage loop has gain `CostFloor / 10 000` times the labour share (STB-L10).

### N-057 The worked examples of `spec-production.md` 12 are not viable as written

- **Areas:** PRO, REF
- **Raised:** 2026-10-10, `spec-viability.md` 15.
- **Note:** Example A (6 grain per 120-tick job) needs 73 times the available labour; example B needs 12 times. They are packs with placeholder numbers, but FR-PRO-046 says they must "run to a stable state". Rescale them before REF or any test uses them. A set that passes (with invented completing recipes) is in `spec-viability.md` 15. Rule of thumb from the runs: the output per labour tick of each step of the chain must exceed the household's need by the depth of the chain.
- **Closed:** 2026-10-10. Applied in `spec-production.md` 12: both examples rescaled to the viable set of `spec-viability.md` 15 (invented completing recipes labelled as such); `spec-viability.md` confirms both pass check V3.

### N-059 Merge the AGT entries into the stability register

- **Areas:** STB, AGT
- **Raised:** 2026-10-10, `spec-agents.md` 18 and `spec-stability.md` 4.7.
- **Note:** The thirteen entries `AGT-A1` to `AGT-A13` of `spec-agents.md` 18 were written against the checklist of `spec-stability.md` 9 while that spec was being written. Merge them into `spec-stability.md` 4.7 and remove the AGT hook row. D24 and D09 become specified (`spec-agents.md` 7.1.5 and 7.1.2); D26 (survival rule, not a damper of L14), D27 (budget stack and saving rule) and D28 (proportional work share) are new; scenarios S19 and S20 are new. D13 (`MinWorkTicks`) is not adopted; WEL decides.
- **Closed:** 2026-10-10. Applied in `spec-stability.md` 4.7: the thirteen AGT entries merged (L44 to L51; five merged into existing loops), D24 and D09 recorded as specified, D26 to D28 added, S19 and S20 added; D13 is adopted by WEL as `MinWorkShare` and D29 is the derived window.

### N-044 The stock signal alone is an integrator

- **Areas:** BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.1.
- **Note:** A price rule driven only by the stock ratio gives a sustained oscillation of 6% to 29% of the price in the model; the amplitude scales with `BeliefStep`. The evidence rule is the proportional term that stops this, and the combination is stable in the model, so keep both and re-test when the evidence rule is revised (N-042). Test: S01, S04.
- **Closed:** 2026-10-10. Applied in `spec-beliefs.md` 5.5: both signals are needed. Re-tested with the revised evidence rule in `spec-stability.md` 6.2 run 2b (stock-only recovers in 98 ticks, combined in 30).

### N-049 Order reasons the strategy contract adds

- **Areas:** CLR, PRO
- **Raised:** 2026-10-10, `spec-agents.md` 6.5.
- **Note:** `not_tradable` (an intent outside the agent's tradable set) joins the table of `spec-clearing.md` 3.2; `not_on_role` (a job request off the role's list) joins the refusal reasons of `spec-production.md` 4.2, with a `Debug` log. Both are constants in `internal/logkeys`.
- **Closed:** 2026-10-10. Applied: `not_tradable` in `spec-clearing.md` 3.2 (FR-CLR-030, checked after `unknown_commodity`, pseudo-agents exempt) and `not_on_role` in `spec-production.md` 4.2 and 14.2 (FR-PRO-008, FR-PRO-048, checked first).

### N-048 The tradable set gives "tradable" a meaning

- **Areas:** GDS, CUR, CFG
- **Raised:** 2026-10-10, `spec-agents.md` 3.4.
- **Note:** `spec-goods.md` 6.4 and `spec-currency.md` 3.1 speak of the cheapest "tradable" commodity without defining it. `spec-agents.md` 3.4 defines it: a commodity in the tradable set of at least one role. Edit both specs' wording and FR-GDS-002.
- **Closed:** 2026-10-10. Applied: "tradable" is defined by `spec-agents.md` 3.4 and used in `spec-goods.md` 6.4 and FR-GDS-002, `spec-currency.md` 3.1 and `spec-config.md` 7.4.1.

### N-041 Edits to other specs owed by PRO and BEL

- **Areas:** LED, CFG, CLR, CUR, GDS, SKL, WEL, SCL, CAL, AGT, WLD, EST, PRO, BEL, STB, VIA
- **Raised:** 2026-10-10, handoffs of the workers that wrote `spec-production.md` and `spec-beliefs.md`.
- **Note:** The two new specs need changes in specs that already exist. None is made yet; make each in the next change that touches that spec, and close this note when all are done.
  - `spec-ledger.md`: 7.2 add the `disposal` reason; 8.1 allow an empty side on `Transform`; 5.4 add `WearItems` (N-033).
  - `spec-welfare-and-wages.md`: 4.3 working tick = job in progress (N-035); a note on the window against job length (N-032).
  - `spec-skill-barriers.md`: 2.1 and 4.2 refer to `spec-production.md` 4.4 and 8 (trainee-eligible recipes have no tool; trainee output = `TraineeYield`; breakage is not random); 2.4 reword (N-039).
  - `spec-config.md`: 4.1 bounds `MaxRecipeTicks`, `MaxYieldBP`, `MaxRequires`, `MaxDraws` (provisional); 4.2 the recipe checks of `spec-production.md` 10.3; 3.1 a `Yields` section; 7.3 and 7.4 the sentinels `ErrCostShares`, `ErrToolNotDurable`, `ErrEligibleHasTool`, `ErrUnknownSeason` and the PRO rules, and the 15 BEL parameters and rules (FR-BEL-027 to 029); 7.5 a `Sited` source with `Draws` counts as sourced; 6.2 the example keys; 7.4 name `BeliefStep`.
  - `spec-clearing.md`: 5.4 the seed price; 7.3 and FR-CLR-025 the outcome goes to beliefs once per admitted order; 5.4 or 7.1 who keeps the price window; open question 1 is also true of the paper's pseudocode; 9.2 cross-reference to the BEL `Warn`.
  - `spec-currency.md`: 3.1 name `BeliefStep` (floor `ceil(10 000 / BeliefStep)`; the default 200 gives 50).
  - `spec-goods.md`: 6.3 the effective reference price is also the default seed price and the centre of the reference band; open question 4 is answered.
  - `spec-scale.md`: open question 3 is settled for production.
  - `spec-calendar.md`: 9 the "season shorter than a delay" entry does not arise for production.
  - Future specs: AGT (N-034, N-037), WLD (N-009, N-036), OBS (N-012, N-035, N-040), STB (N-014), VIA (N-036, N-038).
  - `rules/observability-and-logging.md` 4.3: `RecipeExecuted` at completion with new fields, `RecipeAbandoned`, and later `BeliefsUpdated`. A rules change: it needs a new read-receipt tag and is not made here.
- **Added by round 2 (2026-10-10).** Further edits owed, none yet made:
  - `spec-clearing.md` 3.2, 9: the rejection reason `not_tradable` (N-049); 7.3 and FR-CLR-025: the outcome goes to `Revise` with the market report once per admitted order.
  - `spec-production.md` 3.3, 4.1, 4.2, 14.2: the strategy names the recipe through `JobRequest`; refusal reason `not_on_role`; `Role.Recipes` is owned by `spec-agents.md` 3.3; 12 and 18: rescale the worked examples (N-057), question 2 is answered by `spec-viability.md` 8.5; 9: the decision on labour in `UnitCost` (N-055).
  - `spec-beliefs.md` 3.2.2 (cost ceiling, N-055), 4.4, 10.1, 10.5 (`Target` and `Stock` are defined in `spec-agents.md` 7.3 and 6.4.4; the method that receives the outcome is `Revise`), 5.4 (the evidence move, N-042), 5.5 and 17 (the stock rule alone is an integrator, N-044; section 17 can point to STB-L01, L03, L21, L13, L10, L05, L07, L09, L08), 8.4 (the table is `Config.WarmStart.Prices`).
  - `spec-ledger.md` 7.2, 5.2.3, 7.4, 8.3: the goods form of `initial` at construction, lots with negative `Born`, a construction bucket in the flow totals (N-056); the query `Expiring` (N-053).
  - `spec-estates.md` 2.3, 3: `StartMoney` is `Role.StartMoney`; a later agent receives no goods; removal mechanics are `spec-agents.md` 3.6; pool seeding by `WarmStart.Pools` (`spec-viability.md` 8.3.3).
  - `spec-scale.md` 4.1, 4.4, 4.8, 7.2: cross-references to `spec-agents.md` 3.7, 5, 9; the scale-consistency check can use a warm start.
  - `spec-welfare-and-wages.md` 2.1, 4.3, 5, 5.1, 7: `cost_t` is the effective upkeep of `spec-agents.md` 5.2; income netting by `Purpose` (N-050); 5.1's "left to the strategy" is answered by `spec-agents.md` 7.1.2; WEL decides D13; the role economics check fails in a closed upkeep-only economy (N-054); note that labour supply is unit-elastic (`spec-stability.md` 6.1).
  - `spec-skill-barriers.md` 4.2: a trainee's upkeep is consumed from the trainer's account (`spec-agents.md` 5.7).
  - `spec-currency.md` 5.1, 5.3: the effective upkeep and `BasketTolerance`. `spec-goods.md` 6.4, FR-GDS-002, FR-GDS-003: "tradable" is defined by `spec-agents.md` 3.4 (N-048).
  - `spec-config.md` 3.1, 4.1, 4.2, 6.2, 7.3 to 7.6, 8, 9.2: the `WarmStart` section and its hash, bounds `MaxViabilityPivots`, `MaxViabilityCells`, `MaxLanes`, `MaxUpkeepItems`, the sentinels `ErrNotViable`, `ErrWarmStartPrice`, `ErrWarmStartJob`, `ErrUnknownStrategy`, `ErrBasketMismatch`, the viability rules and warnings, FR-CFG-033 satisfied by FR-VIA-004 to 008 and 011, the cycle-gain check (FR-STB-017).
  - `spec-calendar.md` 4.5, 9: state the periodic extension used by `spec-viability.md` 8.5.2 (optional).
  - The sources' own register sections (`spec-clearing.md` 13, `spec-scale.md` 10, `spec-config.md` 12, `spec-goods.md` 10, `spec-ledger.md` 14, `spec-calendar.md` 9, `spec-production.md` 16): a line "merged into `spec-stability.md` section 4" at the next touch.
  - `rules/architecture.md` 3.5 (allow `math/big` in `internal/steady`, N-058) and 4.3 (the `Strategy` row could say "Work, Trade and Revise") and `rules/observability-and-logging.md` 4.3 (`UpkeepShortfall`, `AgentSpawned` fields): rules changes, each needs a fresh read-receipt tag; not made.
- **Closed:** 2026-10-10. All owed edits were made in round 3: ledger, estates, scale and calendar; clearing, goods, currency and carriage; welfare and wages and skill barriers; config; production and beliefs; stability; and the agents and viability sweeps. Rules changes remain and are N-077.

### N-060 A starting wage for the cost anchor

- **Areas:** VIA, PRO, AGT, WEL
- **Raised:** 2026-10-10, `spec-production.md` 9.3 and 18.11, `spec-agents.md` 19.
- **Note:** The cost anchor is 0 while a region has no observed wage (`spec-production.md` 9.1.1), so beliefs have only the reference band in the first ticks.
- **Closed:** 2026-10-10. `spec-viability.md` 8.2.5 adds `WarmStart.Wages` (a starting wage per region from the labour-row dual, used only until the first working household-tick is observed; FR-VIA-049 to 052); `spec-welfare-and-wages.md` 4.3 exposes the observed wage per working household-tick.

### N-062 The derived welfare window

- **Areas:** WEL, CFG, PRO, STB
- **Raised:** 2026-10-10, `spec-welfare-and-wages.md` 2.3.3.
- **Note:** The effective window is `T = max(WelfareWindow, WindowCycles * longest Duration)`. CFG needs `WindowCycles`, `MaxWelfareWindow` and the effective window in the overflow check; PRO's `job_longer_than_window` becomes obsolete; STB's L19 damper is the window.
- **Closed:** 2026-10-10. Applied in `spec-config.md` (3.7, 4.1, 4.2, 7.4; FR-CFG-050 to 054), `spec-production.md` 10.4 and FR-PRO-040 (the warning is replaced by the informational `window_extended`) and `spec-stability.md` (D29).

### N-063 Estate ask anchor

- **Areas:** EST, SKL
- **Raised:** 2026-10-10, `spec-skill-barriers.md` 2.7.
- **Note:** `spec-skill-barriers.md` 2.7 anchored the estate's ask to the region's recent average price while `spec-estates.md` 2.7.3 anchors it to the effective reference price.
- **Closed:** 2026-10-10. Decided by the coordinating agent: the effective reference price, times remaining over full durability for a durable, less the per-tick markdown. It is deterministic and does not couple the estate to belief state. A market-anchored rule (the price-window mean, then the cost floor, then the reference price) is an experiment for later. Recorded in `specs/README.md` decisions.

### N-066 The cycle-gain warning formula

- **Areas:** CFG, STB
- **Raised:** 2026-10-10, `spec-stability.md` 4.1 (L10).
- **Note:** `spec-config.md` 7.6 item 3 computed the cost-loop gain from inputs only, but labour is now in the unit cost.
- **Closed:** 2026-10-10. Restated in `spec-config.md` 7.6 (FR-CFG-044): the material loop gain is `CostFloor / 10 000` times the full-cost share of the cycle's own inputs, which at the solver's prices is `q_in / q_out`; the wage loop is flagged by `cost_floor_at_full_cost`. `spec-stability.md` FR-STB-017 follows.

### N-069 Warning rule names

- **Areas:** CFG, CAL, SKL
- **Raised:** 2026-10-10, `spec-config.md` 7.4.2.
- **Note:** Three warnings were stated by other specs without a name: `inexact_hour`, `season_shorter_than_tick` (`spec-calendar.md` 7.3) and `tool_unaffordable` (`spec-skill-barriers.md` 2.3).
- **Closed:** 2026-10-10. Adopted in `spec-calendar.md` 7.3 and `spec-skill-barriers.md` 2.3 under the names given by `spec-config.md` 7.4.2.

### N-001 Households are the consumers; avoid the word

- **Areas:** DEM
- **Raised:** 2026-10-09, in a question about simulating consumers.
- **Note:** There is no separate consumer population. Every agent is a household
  that earns, pays upkeep and buys wants (`spec-currency.md` 5.5; overview).
  `spec-agents.md` owns upkeep orders and the consumption rule. `spec-demand.md`
  (version 1) must decide whether it needs households that do not produce
  (people on unearned income) or whether classes are only roles. The wage and
  demand loop (roadmap, "Macro is measured, never set") depends on this.
- **Watch for:** "consumer" in these docs means the *library consumer* (a game or
  server). Call the household a household or the demand side.
- **Update (2026-10-10, AGT):** a class of non-producing households is a role with an empty `Recipes` list (`spec-agents.md` 3.3.1). Upkeep, the consumption rule and the saving rule are in `spec-agents.md` 5 and 7. Spending beyond upkeep does not exist in version 1 (11.7); DEM must supply it (see N-054).
- **Closed:** 2026-10-10. Applied: a class of non-producing households is a role with an empty `Recipes` list (`spec-agents.md` 3.3.1, `spec-demand.md` 3.3); its income is a transfer named by another spec (`spec-demand.md` 7.3, warning `class_without_income`).

### N-052 A price-responsive flow for a staple

- **Areas:** STB, DEM, STY
- **Raised:** 2026-10-10, `spec-agents.md` 11.6.
- **Note:** A staple's demand elasticity is 0 to 1. If S04 or S16 shows a pinned market that the halt and exit do not cure, add the rationing of tier 1 above a dear price (11.6.1) or an elastic demand in DEM.
- **Closed:** 2026-10-10. Rejected as framed (`spec-demand.md` 17): tier-1 rationing makes hunger a choice and is an unsupported guess. DEM supplies elastic demand for discretionary goods, including an elastic increment on a staple listed as a want item. Reopen condition: S04 or S16 shows a pinned staple market that the halt, exit and the work floor do not cure; then raise a new note for AGT (upkeep alternatives).

### N-054 The closed economy fixes the mean welfare ratio at 1

- **Areas:** WEL, DEM, STB, MON
- **Raised:** 2026-10-10, `spec-viability.md` 4.7.
- **Note:** With no money faucet or sink and upkeep as the only purchase, household money is conserved, so total income equals total upkeep spending and the household-weighted mean welfare ratio is 1 whatever `TargetRatio` is. The default 15 000 cannot be met by all roles; the target-income rule never rests; the surplus of the steady state is produced and unsold. The role economics check of `spec-welfare-and-wages.md` 7 fails for every role in such a world. DEM, MON or EXT must supply the spending or money that lets a ratio above 1 exist. Same loop as N-028 from the other side. STB: record it, with a test that runs the default target with no DEM and asserts a bounded glut.
- **Answer (2026-10-10, AGT):** version 1 has no spending beyond upkeep; the glut is accepted and bounded by the production halt (`spec-agents.md` 11.7, FR-AGT-070). Open for WEL: whether the default `TargetRatio` stays 15 000 in a world without DEM, MON or EXT.
- **Update (2026-10-10):** WEL made the role-economics check a report, never an error (`spec-welfare-and-wages.md` 7.1). Decided by the coordinating agent: the default `TargetRatio` stays 15 000. In a world whose only demand is upkeep the mean ratio is 1, agents work at the floor-bounded maximum and the glut is bounded by the production halt; lowering the default would hide the missing demand, money-supply and external-sector specs. Revisit when DEM, MON or EXT exist. STB records it as STB-L46 with scenario S19.
- **Closed:** 2026-10-10. Answered by `spec-demand.md` 7: with wants the closed economy's mean welfare ratio is `1 + W/U` (model: 1.53 at target 15 000, against exactly 1.000 with upkeep only), up to `1 + Wmax/U`; no money faucet is needed; the money needed is the reserve plus about `e*` per household. The default `TargetRatio` stays 15 000 with the module on or off (`spec-welfare-and-wages.md` 7.2); the warning `target_above_demand` compares a pack's target with its own ceiling. STB-L46 stays for a pack without the demand module.
- **Update (2026-10-10, MON):** with MON the answer stands: a balanced pair leaves the mean ratio unchanged, an unbalanced faucet raises it only while the supply grows (`spec-money-supply.md` 8.2, 9.1); the default `TargetRatio` stays 15 000, so "revisit when DEM, MON or EXT exist" is answered for MON.
- **Update (2026-10-10, POP):** the closed ceiling matters for the population: with upkeep as the only demand a fed household reads a mean ratio near 10 000, so the Malthusian fixed point is at chronic hunger or the population dies out (`spec-population.md` 5.3, POP-A7).

### N-047 What the reference chain must contain

- **Areas:** REF, STY, VIA, STB
- **Raised:** 2026-10-10, `spec-stability.md` 8.
- **Note:** To build scenarios S04 to S20 the chain needs a self-employed staple role with demand elasticity below 1, a perishable, a recipe cycle (for the cost-push gain), a long recipe or trip (the cobweb), a durable with a replacement cycle, a role with a job longer than the welfare window, and a region that can lose its income (pool, relief, failure ladder). STY sets the bands and the regime thresholds; VIA staggers the initial jobs (S18).
- **Update (2026-10-10):** scenarios now run to S22 (S19 closed economy, S20 reserve, S21 seeds against the cost band, S22 warm against cold start). The chain also needs a world with upkeep as the only demand, a role with a job longer than the base window, and a warm start. The three-region run of VIA used wood yields by region that `spec-production.md` 12.1 does not list (N-070).
- **Closed:** 2026-10-10. Applied in `spec-reference-chain.md` 3 and 4: features C01 to C17 (a self-employed staple role, a perishable, a seed cycle, a 60-tick job, durables, a job longer than the window, a famine variant with estates, a warm start, an upkeep-only world, trainee recipes above and below break-even); the scenarios are assigned to packs in `spec-stability.md` 8. Bands are STY's.

### N-070 Regional yields of the three-region run

- **Areas:** VIA, REF
- **Raised:** 2026-10-10, `spec-viability.md` 15.
- **Note:** The three-region run used wood yields by region that `spec-production.md` 12.1 does not list. REF adds them when it builds scenario A3.
- **Closed:** 2026-10-10. The yields of the three-region run are listed in `spec-reference-chain.md` 4.4.3.

### N-034 Role-to-recipe membership and the choice of recipe

- **Areas:** SPN
- **Raised:** 2026-10-10, `spec-production.md` 3.3, 4.1, 5.1.2.
- **Note:** `Role.Recipes` (sorted list of recipe IDs) is a field AGT owns. An agent has one job at a time, starts only when free, and the `Strategy` names the recipe and an optional maximum number of executions. A role change abandons the job (reason `role_changed`). The default rule for choosing among executable recipes (for example by expected net value using `UnitCost` and the expected price) is AGT's. The spawner must expect a job to be lost when it moves an agent to another role.
- **Update (2026-10-10, AGT):** `Role.Recipes` is in `spec-agents.md` 3.3; the default recipe choice is the eligible recipe with the greatest net value per tick at the agent's belief centres, ties to the lower ID, net value positive or a source (7.1.4); the world refuses a request off the role's list with `not_on_role` (6.5.1). The spawner must expect a job to be lost when it moves an agent to another role.
- **Closed:** 2026-10-10. Applied: `Role.Recipes` in `spec-agents.md` 3.3; the default recipe choice in 7.1.4; a role change abandons the job with `role_changed` (`spec-spawner.md` 4.2).

### N-064 Traineeship is for weight 1

- **Areas:** SPN, SKL, SCL
- **Raised:** 2026-10-10, `spec-skill-barriers.md` 4.1.1.
- **Note:** A trainer and a trainee are agents of weight 1; a role with `TrainingTicks` above 0 needs groups of scale 1 (validation `traineeship_needs_weight_one`). SPN: the entry rule must not offer a traineeship to a cohort.
- **Closed:** 2026-10-10. Applied in `spec-spawner.md` 4.4 and FR-SPN-009 (the world refuses `Apply` for weight above 1, `needs_weight_one`) and `spec-skill-barriers.md` 4.1.1.

### N-073 A weight rise the pool cannot fund

- **Areas:** SPN, POP, EST
- **Raised:** 2026-10-10, `spec-estates.md` 2.8.1 and open question 2.
- **Note:** A weight rise draws money from the estate pool; refuse it or apply it partially when the pool is short. Goods drawn for a rise are unspecified.
- **Closed:** 2026-10-10. Applied for entry in `spec-spawner.md` 4.3 and 7.3 and `spec-estates.md` 2.8.1: partial, in whole households, money only, within the whole-pool budget `EntryPoolShare`; entrants get no goods. POP confirms for births.
- **Update (2026-10-10, POP):** confirmed for births with one change: each born household brings `Role.StartMoney * BirthStake / 10 000` (`spec-population.md` 4.3), with the ledger reason `births` for a new agent as well as for households added to an existing one. In a closed region the stake caps the population at M / stake (`spec-population.md` 5.5, model 2).

### N-058 Exact big integers in the solver

- **Areas:** ALL, CODE
- **Raised:** 2026-10-10, `spec-viability.md` 6 and open question 7.
- **Note:** `internal/steady` uses `math/big` (standard library). architecture.md 3.5 bans floats and requires checked arithmetic where `int64` can overflow; unbounded integers cannot overflow. A sentence in 3.5 would remove doubt. That is a rules change and needs a fresh read-receipt tag; not made.
- **Closed:** 2026-10-10. Applied in `rules/architecture.md` 3.5: `math/big` is allowed in `internal/steady` only, with the conversion rules and no `math/big` type in anything exported.

### N-077 Rules changes owed by the specs

- **Areas:** ALL, CODE
- **Raised:** 2026-10-10, handoffs of rounds 2 and 3.
- **Note:** None is made; each needs a fresh read-receipt tag. (1) `rules/architecture.md` 3.5: allow `math/big` in `internal/steady` (N-058). (2) 4.3: the `Strategy` row could say "`Work`, `Trade` and `Revise`". (3) `rules/observability-and-logging.md` 4.3: `RecipeExecuted` at completion with new fields, `RecipeAbandoned`, `UpkeepShortfall`, `AgentSpawned` fields and the removal event's `Cause`, and later `BeliefsUpdated`; 3.3.2: the log keys `path`, `rule`, `config_hash`, `region`, `bound`, `agents`, `households`, `unfed`.
- **Closed:** 2026-10-10. Applied in two edits: `rules/architecture.md` 3.5 and 4.3, and `rules/observability-and-logging.md` 3.3.2 (log keys), 4.3 (events including `WeightChanged`, `RoleChanged`, `EntryLimited`, `WantShortfall`) and 5.3 (metrics including `household.unfed`, `role.changes`, `want.unfunded`), with fresh read-receipt tags. Remaining conflicts of the rules edit are N-088.

### N-068 Model scripts are outside the repository

- **Areas:** STB, VIA, CODE
- **Raised:** 2026-10-10, `spec-stability.md` 15.
- **Note:** The numbers of `spec-stability.md` 6, `spec-viability.md` 15 and the belief revision come from small Docker Python models in `/tmp/economy-boss/` (`stb/`, `via/`, `pb/`), which will vanish with that directory. Decision for the human: archive them under `doc/specs/` (with a documentation index row; the project would then hold Python) or accept that the specs' descriptions are the record.
- **Closed:** 2026-10-10. Decided by the human: archive. The scripts are in `doc/specs/models/` (`stb/`, `via/`, `pb/`, `agt/`) with a README and a documentation index row; the specs cite the new paths. The scripts of the demand, spawner and reference-chain specs are still in `/tmp/economy-boss/` (see N-089).

### N-024 Which of FIS, INT and PLC are in the first release

- **Areas:** FIS, INT, PLC
- **Raised:** 2026-10-10, `roadmap.md` Open 1.
- **Note:** `spec-fiscal.md`, `spec-integration.md` and `spec-policy.md` are
  candidates to defer past the first release, and the decision is open. CHK is
  core (ADR 0002). SCL and DEM change how agents work and are costly to retrofit,
  so they stay in the core. Settle it against `vision.md` section 6 (version 1
  scope). Note that INT carries the player agent (N-022) and external shocks, so
  deferring it also defers those.
- **Answer (proposed, 2026-10-10):** follow `vision.md` section 4. FIS and PLC are
  later. INT is in version 1 only as shocks and a player agent, because STY needs a
  shock (a failed harvest) and the first consumer needs a player. A game can inject
  taxes as shocks until FIS exists. It binds only when the vision is accepted and
  recorded in an ADR; close this note then.
- **Closed:** 2026-10-10. Decided by the human (ADR 0003): FIS and PLC are later; INT is in version 1 only as shocks and a player agent.

### N-008 Weight changes, cohort failure and the estate

- **Areas:** POP
- **Raised:** 2026-10-09, `spec-scale.md` 4.8 and open question 4, `spec-ledger.md`.
- **Note:** A cohort whose money reaches 0 fails whole. Recommended: reduce its
  weight by the share it cannot support before removing it (`spec-scale.md` 5.1),
  inside a failure ladder (change role, relief, then death). A weight change is a
  ledger operation with reason `split`, `merge`, `births` or `starvation`, scaling
  holdings in proportion with the remainder going to the estate
  (`spec-scale.md` 4.8; `spec-ledger.md` 7.2). Account IDs are never reused, so a
  run with many deaths accumulates closed accounts (`spec-ledger.md` open
  question 6; N-020). The estate pseudo-agent is not an agent: it has no weight and
  is not counted in agents or households (`spec-estates.md` 2.7.4).
- **Update (2026-10-10, AGT):** a cohort does not fail by itself; an agent with no money still exists and goes unfed (`spec-agents.md` 3.6.1). AGT supplies `Unfed` and `Hunger` (5.4, 5.5) and the effect of a weight change on agent state (3.7). The ladder and the partial weight reduction by the unfed share are SPN and POP's.
- **Update (2026-10-10, EST):** weight changes pass money through the estate (`spec-estates.md` 2.8.1); a rise the pool cannot fund is open (N-073). The ladder is SPN and POP's.
- **Update (2026-10-10, SPN and EST):** applied for SPN and EST in `spec-spawner.md` 5 (the ladder, reduction by the unfed share, the estate path) and `spec-estates.md` 2.8. POP owns the demographic rates (`PopDelta` hook, `spec-spawner.md` 8; the ledger reason `deaths`).
- **Update (2026-10-10, POP):** applied for POP in `spec-population.md` 4.7: the ledger reasons are `births` (extended to every household born through the module) and `deaths` (new, a partial reduction; a death that empties an agent is a removal with `estate_money` and `estate_goods` and cause `population`). The ladder's `starvation` reduction stays SPN's.
**Closed:** 2026-10-10. Applied in `spec-population.md` 4.7 (reasons) and 4.5 (no household counted by two rules).

### N-010 Regions

- **Areas:** REG, MIG
- **Raised:** 2026-10-09, from GDS, EST and LED.
- **Note:** `spec-regions.md` owns the per-region override of a commodity's
  reference price (`spec-goods.md` 6.3) and decides whether a region holds one
  settlement or several (roadmap, Open 3). Each region creates its estate
  pseudo-agent and account (`spec-estates.md` 2.7). Changing an account's region
  is an explicit operation (`spec-ledger.md` 3.5) and its interaction with goods in
  transit is open (`spec-ledger.md` open question 8; `spec-migration.md`).
- **Update (2026-10-10, TRN):** goods in transit are in Transit ledger accounts, so relocating an agent moves no cargo (`spec-transport.md` 7; `spec-regions.md` 6.5.1). TRN registers the `region_bound` refusal `open_shipment` (`spec-regions.md` 6.5.2) when the module is on: an away traveller, or the shipper or consignee of a shipment in flight, is refused by the public relocation with `ErrRegionBound`. At arrival TRN changes the traveller account's region by the ledger-level `Relocate` with cause `transport` and emits REG's `AgentRelocated`; it does not use `NoRegion`, `depart` or `arrive` in version 1 (the traveller stays in the origin region while away). It uses REG's `Distance(from, to)` (`(int64, bool)`, 4.3). Answers `spec-ledger.md` open question 8 for transport (`spec-transport.md` 1.2, 5.5.3).
**Closed:** 2026-10-10. Applied in `spec-regions.md` (REG): the reference-price override is `RegionPrices` (5.2), one case of a single override mechanism (5.1); one settlement per region (3.1); each region's estate account and pseudo-agent are created in region order before any household, the pseudo-agent's agent ID equal to the `RegionID` (3.6); changing an account's region is `relocate` (6): ordered refusals, no transfer and no reason, relocation totals in the ledger, the event `AgentRelocated`. The hook for goods in transit is 6.5: cargo is never the agent's account, a spec that ties an obligation to a region registers a `region_bound` refusal (TRN registers `open_shipment`), and `NoRegion`, `depart` and `arrive` are reserved and unused in version 1. What MIG, PRO, SKL and CAR must do with a relocation is N-090.

### N-026 One settlement per region, or several?

- **Areas:** REG, SET
- **Raised:** 2026-10-10, `roadmap.md` Open 3. See also N-010.
- **Note:** One settlement per region is simpler and makes `spec-settlements.md` a
  thin layer; several lets a region have a large settlement and its smaller ones.
  Decide when writing `spec-regions.md`, before `spec-settlements.md`, because it
  fixes where market, estate and size class live.
- **Answer (decided, 2026-10-10):** one settlement per region, and a region is one
  market. A town and its hamlets are separate regions joined by short edges, which
  costs little because clearing cost grows with regions and commodities, not
  households (`spec-scale.md`). `spec-settlements.md` (later) then only gives a
  region a size class from its households, which is the thin layer its roadmap row
  describes. Victoria 3 treats the market as the unit and varies a state's access to
  it instead of splitting prices inside a state (dev diary 9, snippet only). Apply
  this in `spec-regions.md` and close the note there.
- **Update (2026-10-10):** ADR 0003 decision 4 records the answer. Apply it in `spec-regions.md` and close this note there.
**Closed:** 2026-10-10. Applied in `spec-regions.md` 3.1.3 and 3.4: a region is one settlement and one market, and a town and its hamlets are separate regions joined by short edges (example C, 12.3). The cost is recorded as REG-A2 in the register (a small market is a thin market, and each hamlet has its own pool). `spec-settlements.md` (later) only adds a size class to a region.

### N-029 How deep a review of other economy libraries?

- **Areas:** POP, EVD
- **Raised:** 2026-10-10, `roadmap.md` Open 6. See also N-015, N-018.
- **Note:** Open before `spec-agents.md` and `spec-population.md`. Only bazaarBot
  has been read in full. abcEconomics, Prehistoric, the Space Project, ASSUME and
  Crusoe are known from search snippets, and two documentation pages failed to
  load. Record what is read in `specs/evidence.md`, and do not set a default from
  an unread source.
- **Answer (proposed, 2026-10-10):** a targeted review, not a survey. Opened today:
  the abcEconomics index (Firm and Household archetypes; the platform moves goods,
  as our ledger does; message passing) and ASSUME's custom-unit example (a unit plus
  a bidding strategy with one `calculate_bids()` reused across actor types; dispatch
  stays in the unit). Both support a single `Strategy` and physical handling outside
  it. Still to read before AGT: the Doran and Parberry bazaarBot paper (N-015), the
  abcEconomics Firm and Household pages, ASSUME's base strategy class, and the rest
  of Lengnick (only pages 3 to 7 are read). Before POP: Clark's own Malthusian
  source, not the VoxEU summary. Skip Prehistoric, the Space Project and Crusoe
  unless a specific question needs them. The evidence rows are added to
  `specs/evidence.md`.
- **Update (2026-10-10):** the Doran and Parberry journal paper is read in full (text only). Lengnick's firm price rule, Table 1 and the price-flexibility experiment are read from extracted text with the Greek symbols lost. ASSUME's custom-unit example re-read; its bidding-agents page (404) and the ZIP paper (403) failed to load. Factorio, Victoria 3 and abcEconomics production pages were opened through a summarising fetch only. Still to read before AGT: the abcEconomics Firm and Household pages and ASSUME's base strategy class.
- **Update (2026-10-10, AGT):** the targeted review for AGT is done. Read: Lengnick's household section (equations 1 to 4, 11, 12, Tables 1 and 2, the search part of 2.2; text extracted from the PDF, symbols partly lost); ASSUME's `base.py` (`BaseStrategy`, `BaseUnit`); abcEconomics `Household`, `Firm` and `Trade` pages through a summarising fetch. ASSUME's bidding-agents page still 404. Still to read before POP: Clark's own source.
- **Update (2026-10-10, POP):** Clark's own source read; see the closing line. Evidence rows added to `specs/evidence.md`; the VoxEU row is superseded for the structure of the model.
**Closed:** 2026-10-10. Applied for POP in `spec-population.md` 16: Clark's own source read (chapter 2 of *A Farewell to Alms*, author's chapter PDF, text extracted, figures not seen; draft chapters 4 and 5 of the October 2006 draft of the whole book, draft pp. 90-93 and 100-103). Chapter 3 and the published edition not read. It gives shapes (birth rate flat or rising with income, death rate falling, living standards falling with population) and no usable rates; no default is set from it. The evidence rows are in `specs/evidence.md`.

### N-076 A carriage market is not a goods market

- **Areas:** CAR, MER, TRN, CLR, LED
- **Raised:** 2026-10-10, `spec-carriage.md` 4.2.
- **Note:** `spec-clearing.md` admits asks only for stock and settles by moving goods; hired carriage moves none and uses escrow. The contract, transport and merchant specs must pick: the carriage unit as a produced good (a trip is a job), or a contract-aware admission and settlement that keeps the guarantees of `spec-clearing.md` 2.3. The carriage unit's price may fall below the resolution floor (an ask of 24 against a floor of 50), so choose a coarser unit.
- **Update (2026-10-10, REG):** a route is an edge, keyed by `EdgeID`, in ascending (From, To) (`spec-regions.md` 4.7).
- **Update (2026-10-10, TRN):** decided in `spec-transport.md` 8: option 1, the carriage unit is a produced good, the *haul unit* of a hired mode, made by an ordinary recipe of a carrier role, traded in the origin region's ordinary market and redeemed by the shipment at dispatch. `spec-clearing.md` needs no change and keeps the guarantees of 2.3. The unit price in the worked example is 1 195 against a floor of 50 (17.4), so the coarser unit is a data choice. Costs: capacity is produced ahead and unsold units spoil (a cobweb, TRN-A3); no escrow and no carrier custody for the market; a bilateral contract with escrow stays MER's.
**Closed:** 2026-10-10. Decided by TRN in `spec-transport.md` 8 (option 1, the haul unit; accepted at the wave 2 integration). `spec-clearing.md` is unchanged. `spec-carriage.md` is edited to match: `FR-CAR-001` and `FR-CAR-002` move to the future contract spec keeping their IDs, `FR-CAR-005` to `FR-CAR-007` are superseded by `FR-TRN-009`, `FR-TRN-010` and `FR-TRN-012`, and the unnamed margin `m_bp` is deleted; `spec-ledger.md` and `spec-goods.md` are edited as `spec-transport.md` 8.5 lists. A bilateral contract with escrow and the backhaul double count stay MER's (N-100).

### N-078 What the demand module needs from the money and income specs

- **Areas:** MON, EXT, EMP, LND, FIS
- **Raised:** 2026-10-10, `spec-demand.md` 7.3.
- **Note:** A non-producing class needs a recurring named transfer of money and a paired sink or return flow. In the model a stipend of 300 a tick to 50 households with no sink left household money at 47 700 against 2 700 at the start.
- **Update (2026-10-10, MON):** applied in `spec-money-supply.md`: the recurring named transfer is a grant (4.2) to a role with an empty `Recipes` list (7.3), counted as income by default (7.2); the paired sink or return flow is a levy, with the validation check `pair_unbalanced` for fixed pairs (9.2, 12.2) and a derived bound for a rate levy (9.3, `m* = E + g/rho`). The model of DEM (a stipend of 300 to 50 households with no sink) is the unbalanced case (9.1). `spec-demand.md` 10.2 `class_without_income` gets its edit in the integration.
**Closed:** 2026-10-10. Applied in `spec-money-supply.md` 4.2, 7.2, 7.3, 9.1 to 9.3 and 12.2: a grant and a levy as data, counted as income by default (the welfare edit of the wave 2 integration, `spec-welfare-and-wages.md` 2.1), with a derived bound for the rate levy.

### N-083 Population adds the `deaths` reason and decides renewal against births

- **Areas:** POP, LED
- **Raised:** 2026-10-10, `spec-spawner.md` 4.1 and 8.2.
- **Note:** POP adds the ledger reason `deaths` and decides `Renewal` against births (warning `renewal_with_population`).
**Closed:** 2026-10-10. Applied in `spec-population.md` 4.6 and 4.7: `deaths` is added to the ledger registry (money and goods, internal, agent to estate, a partial reduction) and `births` is extended to every household born through the module; `Renewal` is ignored (effective 0) while the population module is on, with the warning `renewal_with_population` only for a non-default positive `Renewal` and the informational `renewal_ignored` at the default (4.6.1); births are never netted.

### N-095 The external sector's hooks in money and clearing

- **Areas:** EXT, MON, CLR
- **Raised:** 2026-10-10, `spec-money-supply.md` 10.1 and 10.2.
- **Note:** MON reserves the target kind `external`, the top-up plus sweep as the pass-through counterparty, and the rule that a counterparty whose balance is not conserved is a Boundary account. EXT decides between a boundary-backed admission in CLR and the pass-through account (MON open question 1).
- **Closed:** 2026-10-10. Decided in `spec-external-sector.md` 4.1: a pass-through account (an account of the kind Outside, topped up before admission, swept after the markets) with six built-in reasons. `spec-clearing.md` changes by one wording only (the counterparty beside the estate in 3.1.2 and the `side_not_allowed` row of 3.2); a boundary-backed admission was rejected because the ledger refuses a boundary reservation and two specs would change. MON's target kind `external` is released (`spec-money-supply.md` 10.2, open question 1). The decision is recorded in ADR 0004.

### N-098 The Malthusian feedback needs a fixed factor from RES

- **Areas:** RES, POP
- **Raised:** 2026-10-10, `spec-population.md` 5.2.
- **Note:** The third Malthusian assumption (the ratio falls as households rise) can be provided only by an endowment that recipes draw from (`Draws`, `spec-production.md` 6.5), which fails validation while RES is off. Until RES exists POP's loop has no feedback; the only bounds are the pool's money (M / stake) and `HouseholdCap`. RES must provide a limited, regenerating endowment and say how it makes the ratio fall with households, so RES is written before the POP scenarios can run (recorded in the roadmap). A test double (a ratio function of households) is used until then.
- **Update (2026-10-10, RES):** applied in `spec-resources.md` 8: an endowment that binds is the fixed factor. With a flat abundance curve output per household is the endowment's maximum flow over the households above `N_bind` (an elasticity of -1, a kink); with a curve the stock and the population have closed forms; the stability condition is the characteristic equation `lambda^2 - F' lambda + Gamma = 0`, stable iff `F' < 0`, which gives: `constant` neutral, `refill` always stable and monotone above a rate (0.16 a year at the default population gain), `logistic` stable only above the maximum-yield point and in a narrow basin. Models in `doc/specs/models/res/` agree with the linearisation.
- **Closed:** 2026-10-10. Applied in `spec-resources.md` 8 (the mechanism, the closed forms, the stability condition by shape, the validator's warnings `resource_overshoot` and `resource_unstable_plan`) and in `spec-population.md` 5.2 and 9.2 (the third assumption is provided by RES; `no_fixed_factor` reads RES). The POP scenarios S37 to S40 may now use RES; nothing has run. The closed forms need the welfare ratio at a full draw (N-109).

### N-038 Seed prices, newcomers and migrants

- **Areas:** MIG
- **Raised:** 2026-10-10, `spec-beliefs.md` 8.
- **Note:** A single seed price per region and commodity starts both the beliefs and the market's initial price (`spec-beliefs.md` 8.1, 8.4); VIA may supply a table. CLR: the price before the first trade is that seed price (`spec-clearing.md` 5.4). SPN: replacements and newborns start from the window mean or the market price (8.3). MIG decides whether a migrant keeps or reseeds its beliefs (default reseed).
- **Update (2026-10-10, VIA):** the seed table exists: `Config.WarmStart.Prices` with groups for money, stock and jobs (`spec-viability.md` 8.1, 8.2); newcomers use the window (8.8). It is a start, not a standing anchor.
- **Update (2026-10-10):** the CLR part is applied (`spec-clearing.md` 5.4: the price before the first trade is the seed price). The BEL part is applied (`spec-beliefs.md` 8.1, 8.4).
- **Update (2026-10-10, SPN):** newcomers use `spec-beliefs.md` 8.3, not the warm-start table; role changers keep common beliefs (`spec-spawner.md` 4.2, 7.4). MIG decides whether a migrant keeps or reseeds its beliefs (default reseed).
- **Update (2026-10-10, REG):** `relocate` leaves a migrant's beliefs untouched (`spec-regions.md` 6.3), so MIG's default of reseeding on arrival is MIG's to execute. A warm-start seed replaces the seed price only; the reference band stays on the effective reference price (5.2.2).
- **Update (2026-10-10, POP):** a born household takes the newcomer's beliefs of `spec-beliefs.md` 8.3 (`spec-spawner.md` 7.4); MIG still decides for a migrant. A traveller of the transport module keeps its beliefs across a trip (`spec-transport.md` 9.3).
- **Update (2026-10-10, INF):** a migrant has no rumour state to keep or reseed: its rumours are those of its new region from its next view (`spec-information.md` 7.3).
- **Update (2026-10-10, MER):** a self-carried traveller reseeds its beliefs on arrival by the rule of `spec-beliefs.md` 8.3, taking no draw (`spec-merchants.md` 12.2), as MIG does for a migrant. One helper should serve both.
- **Closed:** 2026-10-10. Applied in `spec-migration.md` 8.4 and 8.5: a migrant is reseeded at arrival as any newcomer (`spec-beliefs.md` 8.3): for every commodity of its tradable set the centre is the clamp of the window mean of the new market (the market price if the window is empty) and the width is `BeliefWidthInit`; it has no rumour state to keep or reseed. A traveller of the transport module is reseeded by the same rule (`spec-merchants.md` 12.2, MER).

### N-100 What transport leaves to MER and INF

- **Areas:** MER, CAR
- **Raised:** 2026-10-10, `spec-transport.md` 8.4, 9.3, 20.2 and 20.5.
- **Note:** A hired shipment needs a consignee in the destination region (otherwise `consignee_elsewhere`); a diverted cargo goes to the destination estate; a traveller keeps its beliefs and remembers nothing else; the backhaul double count of `spec-carriage.md` 5.1 stands; a bilateral contract with escrow is MER's and does not use the book. The shipper's decision is MER's and rumours are INF's.
- **Update (2026-10-10, wave 2 integration):** a shipper that responds to a price gap with a transit delay of `k` ticks is stable in the linearised model only if its gain per tick is below `2 sin(pi / (2 (2k + 1)))` (1.0 at `k` = 1, 0.35 at `k` = 4), unless it counts `Inbound` or a cap bounds what it dispatches per tick, in which case the limit is about 2 whatever `k` is (`spec-transport.md` 15, 16; L84, L85). A model of a linear rule, not of the library: MER should count `Inbound` and test the margin with scenario S30.
- **Update (2026-10-10, INF):** rumours are defined (`spec-information.md` 5). A traveller keeps its beliefs and remembers nothing else; a trip carries no news back in version 1 and an arrival changes no record and no rumour (7.5, FR-INF-028); a per-traveller memory is MER's to ask for (`spec-information.md` 17, open question 1). A shipper's reading of a rumour and the condition on its gain are in INF-A1 and INF-A2 (13; L84, L85 and L93).
- **Update (2026-10-10, MIG):** a migrant is not a traveller and a traveller that is away is not a migrant; the migration guard respects `open_shipment` and `open_contract` (`spec-migration.md` 3.3.1, 7.2).
- **Closed:** 2026-10-10. Applied in `spec-merchants.md`: the consignee of a hired shipment is the payer of a delivery contract (8, 9); a diverted cargo counts as reached and the estate of the destination stands in for a removed payer (9.7); a traveller reseeds its beliefs on arrival (12.2); the backhaul double count stands and is measured (10); the bilateral contract with escrow is built on the shipment and does not use the book (9.1); the shipper's stability rules are in 7 and the model in `doc/specs/models/mer/`. The matching of delivery offers and takes is a first-fit round after the last plan of phase 3, not `Clearing` ([ADR 0005](adr/0005-delivery-contracts-are-matched-by-a-first-fit-round.md)).

### N-105 Instant or timed migration

- **Areas:** MIG, TRN
- **Raised:** 2026-10-10, `spec-regions.md` 6.5.3, `spec-transport.md` 5.5.
- **Note:** `relocate` is instantaneous (`spec-regions.md` 6). A migration that takes time has two ways. It can use TRN's traveller: the account stays in the origin region while away and changes region at arrival, and TRN's `open_shipment` refusal means MIG cannot relocate that agent meanwhile. Or it can use REG's reserved `NoRegion`, `depart` and `arrive`, which 6.5.3 binds to three rules: the departure region is recorded, a death leaves the money to the region departed from, and an account with no region is in no market. MIG picks one and states what a death, a ladder rung and the welfare window mean for a migrant on the road (the window problem of N-099 applies).
- **Closed:** 2026-10-10. Decided in `spec-migration.md` 8.1: a timed journey in the shape of TRN's traveller, the account staying in the region of departure until arrival and changing region at arrival through the ledger-level `Relocate` with the cause `migration`, after a dry run of every other refusal; MIG registers the `region_bound` refusal `on_journey`; REG's `NoRegion`, `depart` and `arrive` stay reserved and unused. A death on the journey leaves the money to the origin's pool, the ladder, relief and POP act as on any agent (8.3, 8.3.1); the window records `k` zero-income, non-working ticks (8.6) and the settling rule (5.2) keeps them from triggering a second move. TRN is unchanged except the wording of `spec-transport.md` 10.5.

### N-106 What a strategy may see of other regions

- **Areas:** MER
- **Raised:** 2026-10-10, `spec-transport.md` 9.1 and 15 (TRN-A1, L84).
- **Note:** The agent view carries `Routes` (for each other region and mode: whether there is a route, the travel time, and for a hired mode the haul units per carrying unit) and `Inbound` (goods on the way to this agent). No view carries a remote price or a remote stock, and none should until INF defines rumours with a delay and an error: otherwise a shipper sees the gap it is meant to estimate, and the transit-delay loop is hidden. MER's shipper reads rumoured prices and `Routes` and counts `Inbound`.
- **Update (2026-10-10, RES):** the agent view carries the endowments of its OWN region only (`spec-resources.md` 5.5.1); no view carries another region's stock.
- **Update (2026-10-10, INF):** the view field is `Rumours` of the `Trade` view of a role with `Informed` (`spec-information.md` 6.2). No view carries a true remote price: a rumour is at least 2 ticks old (the local report is 1) and blurred by an independent error, whatever the speed (5.2.2, 5.5, FR-INF-018). What MER's shipper must do with it: read `Rumours` with `Routes`; count `Inbound` (L84) and cap what it dispatches per tick (D39); use `SourceAge`, since a rumour of a carried-forward price is not news (INF-A3, L94); not difference rumours of successive ticks, because a status change steps the vintage (INF-A4, L95); give shippers a fixed spread of their own thresholds, drawn at creation: in the model of `spec-information.md` 11.2 a spread of 0.8 times the freight settles the gap (0.04 f) where an independent information error of the same size leaves 0.6 f and identical shippers 1.8 f (a model of an invented rule, not of MER's). Condition on the gain: INF-A1 and INF-A2 (L84, L85, L93).
- **Update (2026-10-10, EXT):** applied: no view carries an outside price (`spec-external-sector.md` 9.1, 9.2). In a gate region the price is learned from ordinary orders; elsewhere not at all in version 1; an outside node is not a rumour source (INF records only a market's published price). The shape of a source is EXT open question 10 (N-112).
- **Update (2026-10-10, MIG):** the migration pass reads rumours through INF's function for the agent under review, not through a view, and shows a strategy nothing (`spec-migration.md` 4.3.1); MER's shipper is unchanged.
- **Closed:** 2026-10-10. Applied in `spec-merchants.md` 4, 5, 7: the merchant reads `Rumours` with `Routes`, counts its `Inbound` as a consignee and its own `Outbound` as a shipper, caps the dispatch, spreads its thresholds by agent ID, acts only on a `SourceAge` within a bound and differences nothing. No view carries a remote price.

## Read receipt

If you are an agent about to write or change a spec, you quote the tag below in
your first message about that work, as required by
[AGENTS.md](../AGENTS.md). Reaching it means you have read this file in full. Do
not guess it or reuse one from memory.

When you edit this file (add, update or close a note), replace the tag with a
fresh random value in the same change (for example
`od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`). A stale tag in an agent's
message means it read an older version, and may have missed a note.

NOTES-TAG: 1d6d3f4df410c8c96211b65e
