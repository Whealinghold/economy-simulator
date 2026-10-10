# Spec: Transport, travel time, carrying capacity and goods in transit

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-TRN-NNN` and
`NFR-TRN-NNN`, which tests cite (testing.md section 4). They are in section 19
(FR-TRN-001 to 046, NFR-TRN-001 to 004).

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
version 1 (`vision.md` section 4). Off unless configured: the zero configuration
has no transport mode and no goods ever travel. Needs `spec-regions.md` (REG:
the directed region graph, its distances and the relocation of an account; this spec
uses only what section 1.2 lists), `spec-goods.md` (mass, bulk, shelf life),
`spec-ledger.md` (the Transit account kind, transfers, ageing, `WearItems`),
`spec-production.md` (a haul unit is made by an ordinary recipe), `spec-agents.md`
(the plan a strategy returns, the views, removal) and `spec-config.md` (the module
switch and validation).

**Every number in this spec is a placeholder.** Nothing has been run in the
library, bazaarBot has not been run, and no acceptance band exists (notes N-015,
N-027). The two scripts of section 16 are stylised Docker Python models of the
rules, not claims about the library; the rules themselves are integer. No default
is claimed to be stable. Mode names such as `foot`, `wagon`, `coach` and
`shuttle` are examples in two invented settings and are flagged as examples
wherever they appear (AGENTS.md, setting-agnostic rule); the core knows no mode
by name.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).
Note that in this spec the agent that moves is the **traveller**; a **carrier** is
the role of `spec-carriage.md`, whose households make the capacity that other
agents hire.

## 1. Purpose and scope

Goods are made in one region and wanted in another. This spec is the mechanism by
which goods and agents get from one region to another, and the only one. It
decides how long a journey takes, how much can be carried, what a journey costs,
what happens to goods on the way, and what the world does when they arrive. It
does **not** decide who ships, what, where or why: that is the merchant spec
(`spec-merchants.md`, MER), the information spec (`spec-information.md`, INF) and
the strategy of an agent. Transport provides the mechanism and nothing in it
reads a price or chooses a route.

It covers:

- transport modes as data: speed, capacity, cost (section 3);
- travel time from a distance and a mode (section 4);
- the shipment: request, refusal, dispatch, transit, arrival and the cargo that
  cannot be delivered (section 5);
- queues, blocked shipments and capacity contention (section 6);
- goods in transit in the ledger, the audit and the balance sheets (section 7);
- the decision on N-076, the carriage unit (section 8);
- what transport exposes to strategies and to information (section 9);
- the hooks reserved for risk and infrastructure (section 10);
- parameters, validation, observability, checkpoint state, register entries, what
  was run, a worked example of three regions, the notes settled, the requirements
  and the open questions (sections 11 to 20).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The region graph, its edges and distances, the region entity, the region-change operation of an account | `spec-regions.md` (REG) |
| Who ships, what and why; the decision to travel or to hire | `spec-merchants.md` (MER), the agent's `Strategy` |
| Rumours and delayed prices | `spec-information.md` (INF) |
| Loss in transit: theft, wreck, disaster | `spec-risk.md` (RSK); this spec reserves the hook (10.1) |
| Roads, bridges, ships as fixed capital and how they change a travel time | `spec-infrastructure.md` (CAP); this spec reserves the hook (10.2) |
| Warehouses and storage cost | `spec-storage.md` (STO) |
| Migration of households | `spec-migration.md` (MIG) |
| Tolls and tariffs | `spec-fiscal.md` (FIS) |
| Bilateral delivery contracts, escrow of a negotiated fee | `spec-merchants.md` and the contract feature built on `spec-carriage.md` (8.5) |

### 1.1 The tick, in one paragraph

A journey that takes `k` ticks is dispatched in the trade phase of tick `t`
(phase 3 of `spec-agents.md` 4) and arrives in the first phase of tick `t + k`
(section 5.7). The goods are in no market in between, they age at the end of each
of the `k` ticks from `t` to `t + k - 1` (`spec-goods.md` 4.4), and a traveller
that moves with them is absent from its region's markets for `k` ticks.

### 1.2 What this spec assumes from REG

Transport uses these parts of `spec-regions.md` and nothing else:

- a region has an ID and the region graph has **directed** edges `From`, `To`,
  `Distance` (`spec-regions.md` 3.3, 4.1), with `Distance` an `int64` of at least 1
  in a configured generic distance unit; edges may be asymmetric; the unit has no
  travel-time meaning of its own;
- the query `Distance(from, to)` returns `(int64, bool)`: the length of the shortest
  directed path over open edges and whether one exists, with a total-order tie-break
  between equal paths (`spec-regions.md` 4.3). Transport uses the length only, never
  the path. A closed edge (`spec-regions.md` 7) is absent from the graph, so it lengthens
  the distance or removes the route (`no_route`); an edge factor changes the distance;
- one settlement per region and a region is one market (ADR 0003 decision 4); the
  region creates its estate pseudo-agent and account (`spec-estates.md` 2.7.1);
- the ledger-level `Relocate(account, region)` of `spec-regions.md` 6.1, which transport
  calls itself at an arrival (5.7) and which carries `cause` `transport` (a constant of
  `internal/logkeys`) and the event `AgentRelocated` (`spec-regions.md` 10.1);
- the hook of `spec-regions.md` 6.5.2: a spec that ties an obligation to an account's
  region registers a refusal `region_bound` with a name, and the public relocation
  (`spec-regions.md` 6.1.3, check 7) refuses an account with an open obligation
  (`ErrRegionBound`). `spec-regions.md` pre-names `open_shipment` for transport.
  Transport registers it when the module is on (5.5.3).

Transport does **not** use `NoRegion`, `depart` or `arrive` (`spec-regions.md` 6.5.3) in
version 1: a traveller's account stays in the origin region while it is away and changes
region at arrival. REG's reservation of those stays unused.

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | Transport is a module, off unless configured. A world with modes and strategies that never ask for a shipment behaves as a world without transport | 3.1, 5.3 |
| 2 | A mode is data: a speed, a capacity pair per carrying unit, and either the crew and goods a traveller needs (**self-carried**) or a commodity of **haul units** that a shipper buys and redeems (**hired**). No code branches on a mode | 3.1 |
| 3 | Transport charges goods and time and never money. A trip costs the traveller's time, wear of a durable vehicle and a list of consumed goods; hired capacity costs what its producers' recipe costs, through the ordinary market | 3.3 |
| 4 | Capacity is a pair, mass and bulk, per carrying unit, and a load fits if and only if both sums are within it (`spec-goods.md` 3.2). `BulkFactor` is not used for transport: a hired load is priced by its binding fill, which removes the over-charge of `spec-goods.md` 3.5 for transport | 3.2, 8.4 |
| 5 | Travel time is `ceil(distance / speed)` with a minimum of 1 tick, from the shortest distance of REG, and is fixed at dispatch | 4 |
| 6 | The traveller of a self-carried trip carries everything its account holds, counted against its capacity: no capacity can be bypassed by moving an account | 3.2, 5.5 |
| 7 | Goods in transit are in a Transit ledger account, one shipment per account, and age as in any account. Transit accounts are reused, never closed | 5.6, 7.1 |
| 8 | There are no queues. A request that cannot be met is refused whole with a reason and leaves no state | 6 |
| 9 | The carriage unit is a produced good (N-076, option 1): the haul unit is an ordinary commodity made by an ordinary recipe, traded in an ordinary market and redeemed at dispatch. No change to clearing, and every guarantee of `spec-clearing.md` 2.3 stands | 8 |
| 10 | Loss in transit is RSK's. The reason `loss` stays unused and reserved | 10.1 |

## 3. Modes

### 3.1 The mode as data

`Config.Modules.Transport` is present if the module is on (`spec-config.md` 3.2).
Its only content is the table `Modes`, keyed by ID like every entity
(`spec-config.md` 3.3). A mode has these fields:

| Field | Meaning | Default | Bound |
|-------|---------|---------|-------|
| `ID` | the mode's identifier | required | the ID pattern |
| `Speed` | distance units per tick, in the unit of REG's distances | required | 1 to `MaxSpeed` |
| `UnitMass` | the mass capacity of one **carrying unit**, in mass units (`spec-goods.md` 3.1); 0 means mass is not limited | required | 0 to `MaxCapacity` |
| `UnitBulk` | the bulk capacity of one carrying unit; 0 means bulk is not limited | required | 0 to `MaxCapacity` |
| `Crew` | households a carrying unit needs; self-carried modes only | 1 | 1 to `MaxWeight` |
| `Vehicle` | a durable commodity that each carrying unit needs one item of; self-carried modes only; empty means the households themselves are the carrying unit | empty | a known commodity with `Durability` above 0 |
| `WearPerTick` | wear units per vehicle per tick of travel; only with a `Vehicle` | 0 | 1 to the vehicle's `Durability` |
| `Feed` | a list of (commodity, quantity): goods consumed per carrying unit per tick of travel; self-carried modes only | empty | at most `MaxFeed` entries, each quantity 1 to `MaxQuantity`, no commodity twice |
| `Haul` | the commodity of haul units; setting it makes the mode **hired** | empty | a known commodity |
| `HaulSpan` | the distance one haul unit carries one carrying unit; hired modes only | required if `Haul` is set | 1 to `MaxHaulSpan` |
| `Label` | display name, presentation only | empty | at most 64 bytes |

`UnitMass` and `UnitBulk` are not both 0. A **self-carried** mode (`Haul` empty)
is one in which an agent moves with its own capacity; a **hired** mode (`Haul`
set) is one in which a shipper redeems haul units that other households made and
no agent of the shipper's moves. The shape rules are validation errors
(section 12): a hired mode has no `Crew`, `Vehicle`, `WearPerTick` or `Feed`, and
a self-carried mode has no `HaulSpan`. The mode IDs, the numbers and the shape
are part of the effective configuration and its hash (`spec-config.md` 8.1);
only `Label` is not.

3.1.1. A speed is per tick and is not rescaled when the calendar changes, as every
duration and per-tick quantity is not (`spec-calendar.md` 6.1, 6.2). A pack for a
tick of a day and a pack for a tick of an hour state their speeds for their own
tick.

3.1.2. The same table serves any setting. Two example settings, with placeholder
numbers and no claim about either:

| Setting (example) | Mode | Shape | Speed | Unit (mass, bulk) | Crew | Vehicle | Feed | Haul |
|-------------------|------|-------|------:|-------------------|-----:|---------|------|------|
| An agrarian realm | `foot` | self-carried | 20 | (40, 30) | 1 | none | none | none |
| An agrarian realm | `wagon` | self-carried | 30 | (1 200, 900) | 2 | `wagon`, wear 5 a tick | 3 `fodder` | none |
| An agrarian realm | `coach` | hired | 30 | (1 200, 900) | n/a | n/a | n/a | `haul`, span 10 |
| A mining colony | `crawler` | self-carried | 12 | (6 000, 4 000) | 3 | `crawler`, wear 20 a tick | 2 `cell` | none |
| A mining colony | `lift` | hired | 400 | (20 000, 20 000) | n/a | n/a | n/a | `slot`, span 200 |

### 3.2 Carrying units and what an agent can carry

3.2.1. **Self-carried.** A traveller of weight `w` (`spec-scale.md` 2) fields

```
units = floor(w / Crew)                   with no Vehicle
units = min(floor(w / Crew), v)           with a Vehicle, v = available vehicle items in its account
```

carrying units. `units` is 0 when `w < Crew` (reason `crew_short`) or when it holds
no available vehicle (`no_vehicle`). The pooled capacity of a trip is
`(units * UnitMass, units * UnitBulk)`, and the number of units a load needs is

```
n = max(1, ceil(load_mass / UnitMass), ceil(load_bulk / UnitBulk))
```

where a term with a unit capacity of 0 is left out. The trip is allowed if
`n <= units`, which is the same as the load fitting the pooled pair
(`spec-goods.md` 3.2): both sums within their capacity. Nothing is clamped: a load
that does not fit is refused whole (6.1). The cost of the trip is charged on the
`n` units used (3.3), not on all `units` fielded, and the whole agent is away
whatever `n` is. Pooling treats a cargo as divisible into whole units of goods
across vehicles; a good whose single item exceeds one unit's capacity could be
accepted although no packing exists (open question 4).

3.2.2. **The load of a self-carried trip is everything the traveller's account
holds.** The load is the pair of sums over every good in the traveller's account at
dispatch (the lines it consigns to another agent, if any, are part of them), except
units of the mode's `Vehicle` commodity (a vehicle is not cargo) and the feed that
the trip consumes (3.3.2). Money has no mass and no bulk. The reason is closure: the
traveller's account changes region at arrival (5.7), and a rule that checked only a
declared cargo would let an agent move any quantity of goods by declaring none. The
upkeep goods it holds for the journey are goods it carries and count.

3.2.3. **Hired.** A shipper of any weight, in any role, can ship by a hired mode.
The capacity is the haul units it redeems (3.3.3), not anything it holds or is; no
carrying unit is fielded. The units a load needs are the same `n` as in 3.2.1,
using the mode's `UnitMass` and `UnitBulk`.

3.2.4. **A cohort travels whole.** An agent of weight `w` that travels takes all
`w` households away together. Capacity uses `floor(w / Crew)` units even when the
load needs fewer. A part of a cohort cannot go: that would be a split of the
agent, which a running world does not do in version 1 (`spec-scale.md` 8). The
first consumer needs no weight above 1 (N-030), so the case is a stated
simplification and not a design target; a player and any named merchant are weight
1 (`spec-scale.md` 4.7).

### 3.3 The cost of a trip

Transport charges goods and time; it never moves money (position 3). Money costs
arise elsewhere: the price of hired capacity is the market's, a toll would be
FIS's, a loss would be RSK's.

3.3.1. **Time.** The traveller of a self-carried trip is away for `k` ticks
(section 4) and every one of them is a working tick for the welfare ratio: the
tick of dispatch and the `k - 1` ticks after it (`spec-welfare-and-wages.md` 4.3,
"in transit on a commercial trip"; 5.5). Its upkeep is consumed as in any tick
(5.5). The loss of its labour to production is the opportunity cost, and it is the
reason a trip is not free for a household with a recipe.

3.3.2. **Wear and feed (self-carried).** At dispatch, for the `n` units used and
`k` ticks of travel, in this order (5.5):

- if the mode has a `Vehicle`: `WearItems(traveller, Vehicle, n, WearPerTick * k)`
  (`spec-ledger.md` 5.4.3): the `n` most worn distinct vehicles lose that many wear
  units, and one that reaches 0 is destroyed to the `wear` boundary. The wear of a
  whole trip is charged at its start, as production wears a tool at the start of a
  job (`spec-production.md` 4.4.2); a vehicle that breaks does not interrupt the
  trip;
- for each `Feed` entry (commodity, `q`): `q * n * k` units move from the traveller
  to the boundary under the reason `transit_use` (a goods sink).

A mode with no `Vehicle` and no `Feed` costs only time. A traveller that lacks the
vehicles or the feed is refused (`no_vehicle`, `insufficient_feed`).

3.3.3. **Haul units (hired).** The shipper redeems `n * s` haul units, with

```
s = ceil(distance / HaulSpan)         distance = REG's shortest distance
```

moved from its account to the boundary under the reason `haul_use` (a goods sink;
section 7.2). It buys them beforehand in its region's market like any good. A
shipper with fewer is refused (`insufficient_haul`). One haul unit therefore is the
carriage of one carrying unit's capacity over `HaulSpan` distance units; the cost of
carriage grows in steps with distance (`s`) and with the binding fill of the load
(`n`), and not with the time the journey takes.

3.3.4. **Who pays for hired capacity.** The carrier role that makes haul units: its
recipe consumes feed, requires a vehicle durable as a tool, and takes the working
ticks of its households (`spec-production.md` 3, 4.4). The cost of a haul unit is
that recipe's `UnitCost` (`spec-production.md` 9) and the ask of a carrier is
anchored to it by the beliefs' cost band (`spec-beliefs.md` 3.2.2). No transport
price is configured and no fee formula is needed (section 8). A pack author sets the
recipe's `Duration` and output so that they agree with the mode's speed and span
(17.4); validation does not check the agreement (open question 3).

## 4. Travel time

4.1. For a directed pair of regions `(from, to)` with `from != to`, REG's shortest
distance `D = Distance(from, to)` and a mode with speed `v`, the **travel time** in
ticks is

```
k = max(1, ceil(D * 10_000 / (v * F)))
```

where `F` is the **transit factor** of (`from`, `to`, mode) in basis points, 10 000
by default. With the default factor this is `ceil(D / v)`. The product `D * 10_000`
and the division use the checked multiply-divide helper with a 128-bit intermediate
(`spec-currency.md` 7; architecture.md 3.5); `v * F` is at most
`MaxSpeed * MaxTransitFactorBP` and fits. The rounding is **up**: a journey never
takes less time than the distance needs at that speed, so the choice errs towards
slower transport and never lets goods arrive early. The minimum of 1 is because a
journey dispatched in phase 3 of tick `t` can arrive no earlier than the first phase
of tick `t + 1`.

4.2. If the pair has no path, the request is refused (`no_route`). If `F` is 0 the
pair is **blocked** for this mode (`blocked`; 10.3). If `k` exceeds `MaxTravelTicks`
the request is refused (`too_far`).

4.3. `k` is fixed at dispatch: the shipment records its arrival tick `t + k` and a
later change of the factor or of the graph does not move it (10.3).

4.4. Asymmetry is allowed and is the whole of REG's. In the example of section 17 the
shortest distance from `south` to `north` is 100 although the direct edge is 110,
because the path through `mid` is shorter; from `north` to `south` the direct edge of
90 wins. The traveller does not stop in `mid`: it is in no market on the way.

4.5. The travel time is a pure function of (distance, speed, factor), computed in
`internal/transit` over primitives (architecture.md 2.9), and it is available to a
strategy as a read-only query (9.2).

## 5. Shipments

### 5.1 Two kinds

| | Self-carried | Hired |
|---|--------------|-------|
| Who moves | the **shipper**, which is the **traveller**, with its households | nothing the shipper owns; the goods go alone |
| Capacity from | the traveller's crew and vehicles (3.2.1) | haul units it redeems (3.3.3) |
| Goods | all its holdings (3.2.2); lines only if the consignee is another agent | the lines, always |
| Consignee | the traveller itself, or another agent in the destination | another agent in the destination |
| In a Transit account | only the lines, if any | the lines |
| The shipper's region at arrival | the destination: the traveller always arrives there (5.7) | unchanged |
| Away | yes | no |

For a self-carried trip whose consignee is the traveller itself, no Transit account
is used: the goods stay in the traveller's own account and its account changes
region at arrival. They are the traveller's assets throughout, so a balance sheet
loses nothing during the trip (7.4).

### 5.2 The record

A shipment is plain data owned by the world:

```go
// Shipment is one journey. Plain data; no pointers into live state.
type Shipment struct {
	ID         ShipmentID // assigned in order of dispatch from 0; never reused
	Mode       ModeID
	From, To   RegionID   // From is the shipper's region at dispatch
	Shipper    AgentID
	Traveller  AgentID    // the shipper, for a self-carried trip; unused for a hired one
	Consignee  AgentID
	Hired      bool
	Dispatched int64      // the tick of dispatch
	Arrives    int64      // Dispatched + k
	Account    int        // the Transit ledger account; -1 for a trip to oneself
	Units      int64      // carrying units used (n)
	HaulUnits  int64      // haul units redeemed (n * s); 0 for a self-carried trip
	Lines      []Line     // commodity and quantity shipped, in commodity order
}
```

The quantity delivered is not stored: at arrival the shipment's Transit account holds
exactly what was not spoiled on the way (5.6), and the difference from `Lines` is the
spoiled quantity.

### 5.3 Where a request comes from and when it is processed

5.3.1. A strategy asks for a shipment in `Trade`, by appending to the plan:

```go
// ShipmentRequest asks the world to move goods, or the agent itself.
type ShipmentRequest struct {
	Mode      ModeID
	To        RegionID
	Consignee AgentID // the receiving agent; ignored when ToSelf is set
	ToSelf    bool    // self-carried only: the traveller itself is the consignee
	Lines     []Line  // goods to consign; empty when ToSelf
}

// Line is a commodity and a whole quantity.
type Line struct {
	Commodity CommodityID
	Quantity  Quantity
}
```

`TradePlan` gains the field `Shipments []ShipmentRequest` (`spec-agents.md` 6.3). It
is a change of the plan and not of the interface: `Strategy` keeps its three
methods (architecture.md 4.2, 4.3). The default strategy appends none: it never
ships, so a world of default strategies has no shipment whatever the modes
(position 1). Shipping is the behaviour of a later strategy, MER's, and not of the
core.

5.3.2. **When.** In phase 3, for each agent in ID order, after `Trade` returns the
world executes, in this order: the plan's discards; its shipments, in plan order;
then its intents become orders (`spec-agents.md` 6.5.2). An agent's shipment can
therefore use the stock the strategy saw, and the orders that follow are admitted
against what is left (`insufficient_stock` if the strategy double-counted).
**Arrivals** are processed in phase 1 of the tick, first, after the moment is
computed and before relief and starting money are paid (5.7). An agent that is away
is not passed to `Work` or `Trade` (5.5).

5.3.3. A request is processed against the state at that moment; there is no
batching of requests across agents. Because each agent's goods are in its own
account and an arrival in phase 1 is already complete, one agent's dispatch cannot
change what another agent's request sees, with one exception: the `Inbound` of the
view is fixed at the start of phase 3 (9.1) so that it does not depend on agent
order.

### 5.4 Refusal reasons

A request is **refused** whole, with a `reason`, if the first of these holds. The
checks run in this order; the reasons are constants in `internal/logkeys`. A refusal
changes nothing, emits `ShipmentRejected` (section 13) and is not an error returned
by `Step`.

| # | `reason` | Condition |
|---|----------|-----------|
| 1 | `too_many_requests` | this is request number `MaxDispatchPerTick + 1` or later of the agent's plan |
| 2 | `unknown_mode` | the mode is not in the configuration (also every request when the module is off) |
| 3 | `traveller_away` | self-carried and the agent is already away (including by an earlier request of this plan) |
| 4 | `job_in_progress` | self-carried and the agent has a job in progress, started this tick or earlier (`spec-production.md` 5.2) |
| 5 | `same_region` | `To` is the shipper's own region |
| 6 | `unknown_region` | `To` is not a region |
| 6a | `outside_destination` | `To` is an outside node of the external module (`spec-external-sector.md` 3.1, 4.4); outside goods reach the realm through the gate region's market, not through a shipment |
| 7 | `no_route` | REG has no path from the shipper's region to `To` |
| 8 | `blocked` | the transit factor of (from, to, mode) is 0 |
| 9 | `too_far` | `k` above `MaxTravelTicks` |
| 10 | `unknown_consignee` | the consignee is not an existing agent (a pseudo-agent is not one) |
| 11 | `consignee_elsewhere` | the consignee is not in region `To`, except `ToSelf` of a self-carried request; a hired request with `ToSelf` is this reason |
| 12 | `lines_with_self` | `ToSelf` with a non-empty `Lines` |
| 13 | `empty_shipment` | a hired request, or a self-carried one to another agent, with no line |
| 14 | `too_many_lines` | more than `MaxShipmentLines` lines |
| 15 | `unknown_commodity`, `duplicate_line`, `invalid_quantity` | a line names an unknown commodity, names one twice, or has a quantity below 1 or above `MaxQuantity` |
| 16 | `vehicle_in_cargo` | a line names the mode's own `Vehicle` commodity |
| 17 | `insufficient_stock` | for some commodity, its line plus any redemption or feed of it exceeds the available (unreserved) stock |
| 18 | `crew_short`, `no_vehicle` | self-carried and `w < Crew`; or it holds no available vehicle item |
| 19 | `capacity_exceeded` | the load does not fit the pooled capacity (3.2.1, 3.2.2): `n > units` |
| 20 | `insufficient_haul` | hired and it holds fewer than `n * s` haul units (counting a line of the haul commodity) |
| 21 | `insufficient_feed` | self-carried and it lacks a feed quantity |
| 22 | `unprovisioned` | self-carried and, after the moves of dispatch, the traveller's stock of its upkeep goods is below its effective upkeep times `w` times `k` (`spec-agents.md` 5.2): it would be unfed on the way, and it cannot buy while away |
| 23 | `overflow` | a product of 3.2 or 3.3 does not fit `int64` |

A quantity beyond `MaxQuantity` in a feed or haul product (3.3) is reported as the
corresponding insufficiency, since no account can hold more.

### 5.5 Dispatch

5.5.1. After the checks pass the world applies, with no step able to fail (each was
checked; a failure here is a programmer error and panics, architecture.md 8.1,
`spec-ledger.md` 12.2):

1. **Account.** For a shipment with lines, take the open Transit account of the
   shipper's region with the lowest ID that no in-flight shipment uses, or open a
   new one (kind Transit, region `From`, `spec-ledger.md` 3.1). Transit accounts are
   never closed; they are reused (7.1).
2. **Lines.** For each line in commodity order, `Move(shipper, account, commodity,
   quantity)` under `transit_out`. The moved lots keep their born tick and remaining
   durability (`spec-ledger.md` 5.2.1): age and wear travel with the goods.
3. **Cost.** Self-carried: the feed and the wear of 3.3.2. Hired: the `n * s` haul
   units of 3.3.3 to the boundary under `haul_use`.
4. **The record.** The shipment is stored with the next ID. `Arrives` is `t + k`.
5. **Away.** For a self-carried trip the traveller is marked away until arrival.
6. **The event** `ShipmentDispatched` (section 13).

5.5.2. **The traveller while away.** From dispatch until the first phase of tick
`Arrives`:

- it is not passed to `Work` or `Trade` and places no order. It has no market to
  trade in: it is in none;
- its account keeps its region (`From`) and its holdings stay in the account and age
  as any lot does; its upkeep is consumed in phase 6 of each tick from them
  (`spec-agents.md` 5.3); the spawner acts on it as on any agent (5.9);
- every tick from the dispatch tick to tick `Arrives - 1` is a working tick (3.3.1);
- its account cannot be relocated by anyone else while it is away: see 5.5.3.

5.5.3. **The refusal `open_shipment`.** When the module is on, transport registers the
`region_bound` refusal named `open_shipment` (`spec-regions.md` 6.5.2). It is open for an
account whose agent is the traveller of an away trip, or the shipper or the consignee of
a shipment in flight (an agent whose goods or whose delivery is tied to a region). The
public relocation (MIG, INT) then returns `ErrRegionBound` naming `open_shipment`, and
changes nothing. Transport itself changes a traveller's region at arrival by calling the
ledger-level `Relocate(account, region)` directly, not the guarded public path, with the
cause `transport`; it emits `AgentRelocated` of `spec-regions.md` 10.1 (`Agent`, `Role`,
`From`, `To`, `Weight`, `Money`, `Units`, `Cause`) and no event of its own for the move.
A traveller dying in transit leaves its money to the pool of the region it departed from
(`spec-estates.md` 2.2) and, since its account is still in the origin region, that is the
region of the account at removal (5.9.2).

### 5.6 In transit

5.6.1. Goods in a Transit account are in a ledger account and so are subject to the
ledger exactly as any goods are. The ageing of `spec-ledger.md` 5.3 visits Transit
accounts like all accounts: a lot of a perishable commodity whose `Born + ShelfLife - 1`
is at most the current tick moves whole to the `spoilage` boundary at the end of the
tick, from the Transit account. **A lot that spoils in transit is lost to the
consignee**, and the arrival delivers the rest. Nothing is paused (`spec-goods.md`
4.4, FR-GDS-012). A durable cargo does not wear in transit: wear is by use
(`spec-goods.md` 5.3).

5.6.2. A lot of a perishable commodity born at tick `b` with shelf life `S` shipped
in tick `t` over `k` ticks is delivered if and only if `t + k <= b + S - 1`, that is
`k <= S - 1 - (t - b)`. A fresh lot (`b = t`) therefore travels at most `S - 1` ticks
and arrives with at least one tick of use left (17.2).

5.6.3. Nothing else happens to the cargo in version 1. Reserved for risk: a transfer
from the Transit account to the boundary under `loss` (10.1).

### 5.7 Arrival

In the first phase of tick `t`, after the moment is computed, the world processes the
shipments with `Arrives = t` in order of shipment ID:

1. **A self-carried trip.** If the traveller still exists, its account's region
   becomes `To` by the ledger-level `Relocate` with cause `transport` (1.2, 5.5.3) and it
   is no longer away. If it
   was removed, nothing moves and the shipment record is dropped when the traveller is
   removed (5.9).
2. **Delivery.** If the shipment has a Transit account: for each commodity in the
   account in commodity order, the whole quantity left moves to the consignee under
   `transit_in`, taking lots oldest first and keeping their born tick and durability.
   The consignee is the agent named in the shipment, if it still exists, is in region
   `To`, and the move would not exceed `MaxQuantity`; otherwise the quantity is
   **diverted** (5.8).
3. **Release.** The Transit account is empty. It stays open and joins the free list
   of its region (7.1).
4. **The event** `ShipmentArrived`, with for each line the shipped, delivered and
   spoiled quantity and whether any was diverted (section 13).

The first phase comes before relief and starting money, so a consignee has its goods
and a traveller its new region before anything else of the tick reads them, and
before `Work` and `Trade`.

### 5.8 Cargo that cannot be delivered

If the consignee has been removed, is not in `To`, or cannot hold the quantity, the
goods are **diverted**: they move to the estate account of region `To` under
`transit_in` (a transit-to-estate move; 7.2). The estate lists them for sale like any
estate goods (`spec-estates.md` 2.7), so their value goes to the regional pool. No
goods are created or destroyed, nothing waits, and the event says what was diverted.
The rule is a chosen default that errs towards conservation and a visible,
non-silent outcome; a different rule (return to the shipper) would need the shipper
to be reachable from the destination and is MER's to add (open question 5).

### 5.9 Removal, weight change and role change while away

5.9.1. **The shipper is removed** (a hired shipment; or a self-carried one whose
traveller is not the consignee): nothing happens to the shipment. The goods are in a
Transit account and belong to the consignee on arrival.

5.9.2. **The traveller is removed** (it starves, or the consumer removes an external
agent that is mortal, `spec-spawner.md` 9): its holdings in its own account go to the
estate of the account's region, `From`, as for any removal (`spec-agents.md` 3.6). A
shipment to itself is dropped. A shipment with lines continues without a traveller:
its cargo is in the Transit account and arrives at `Arrives`, delivered to the
consignee; only the traveller's region change is skipped.

5.9.3. **The consignee is removed** before arrival: the cargo is diverted (5.8).

5.9.4. **Weight and role.** A weight change of an away traveller does not change the
trip: the capacity was fixed at dispatch. A role change by the spawner does not end
a trip (it is not a job and `spec-spawner.md` 4.2 abandons only jobs). The ladder's
relief and entry rules see the agent in its account's region, `From`, until arrival.

## 6. Capacity contention, queues and blocked shipments

6.1. **There is no queue.** A request that cannot be dispatched at the moment is
refused whole (5.4), with no state kept. A strategy that wants the shipment next tick
asks again. This is a decision: a queue would be hidden state the strategy cannot see,
with a new lag in the loops of section 15 and a rule for who comes first. Nothing is
clamped either: a load over capacity is refused, not reduced to fit
(`spec-clearing.md` 3.2 for the same reason; sizing is the strategy's job).

6.2. **What competes for what.** Contention can arise only in these places, each with
a deterministic rule:

| Contention | Rule |
|------------|------|
| Several requests by one agent in one tick | processed in plan order against what is left; at most `MaxDispatchPerTick`; a self-carried traveller can only make one (it is away after the first, `traveller_away`) |
| Several agents wanting the same haul units | the market's: haul units are goods bought at a price; the book of `spec-clearing.md` 4 decides, not transport |
| Several agents wanting the same vehicle | none: a vehicle is a durable in one account |
| Agents dispatching on one edge | no limit in version 1. An edge with a limit (a bridge, a pass) is CAP's: a rule at dispatch in agent ID order, refusing with `edge_full` (10.2) |
| Several shipments arriving at one consignee in one tick | processed in shipment ID order; a quantity over `MaxQuantity` is diverted (5.8) |

6.3. **Blocked.** A pair blocked by the transit factor (4.2, 10.3) refuses new
requests (`blocked`) and does not affect shipments in flight. A closed road is
modelled as REG's edge status (closed, or a factor: `spec-regions.md` 7, which changes or removes the distance) or the transit factor going to 0, set between ticks
by the integration spec (`vision.md` section 2), not by transport.

6.4. **A refusal leaves no trace.** The only state a refusal could leave is the Transit
account of 5.5.1, and that is chosen only after every check has passed, so a refusal
opens no account and moves nothing.

## 7. Goods in transit in the ledger

### 7.1 Transit accounts

7.1.1. A Transit account holds the cargo of at most one in-flight shipment and nothing
else (FR-TRN-023). `spec-ledger.md` 3.1 says a Transit account is opened by a shipment;
this spec reuses accounts: a shipment takes the lowest-ID open Transit account of the
origin region that no in-flight shipment uses (5.5.1), and releases it at arrival. The
reason is the growth of closed accounts: IDs are never reused and a closed account's
slot stays until a checkpoint is compacted (`spec-ledger.md` 3.2, 3.3, open question 6),
so closing one account per shipment would add a slot per journey. Reuse bounds the
number of Transit accounts by the peak number of shipments in flight per origin region.

7.1.2. The **region** of a Transit account is the origin region of the shipment that
first opened it (`spec-ledger.md` 3.5), and a free account is reused only by shipments
from that region, so the region never changes. The consequence for flow attribution
is stated in 7.3.

7.1.3. The free list of a region is **derived** from the ledger and the shipment list:
the open Transit accounts of the region that no in-flight shipment references, in ID
order. It is not saved (14).

### 7.2 Reasons

The reasons below join the registry of `spec-ledger.md` 7.2. `transit_out` and
`transit_in` are already there with this spec as owner; the others are new.

| Reason | Of | Kind | Between | Owner |
|--------|----|------|---------|-------|
| `transit_out` | goods | internal | agent to transit | this spec |
| `transit_in` | goods | internal | transit to agent, or transit to estate (a diverted cargo) | this spec |
| `haul_use` | goods | sink | agent to boundary: the redemption of haul units | this spec |
| `transit_use` | goods | sink | agent to boundary: the feed of a trip | this spec |
| `loss` | goods | sink | transit to boundary | **RSK**; reserved, no flow under it in this spec |
| `spoilage`, `wear` | goods | sink | existing; a Transit account and a worn vehicle appear as holders | `spec-goods.md`, `spec-ledger.md` |

`haul_use` and `transit_use` are separate from `consumption` so that they are told
apart in `BoundaryFlow`, as `want_consumption` is (`spec-ledger.md` 7.2). `loss` is
listed in the registry with owners `spec-transport.md` and `spec-estates.md`; the
handoff proposes changing its owner to RSK.

### 7.3 Conservation and attribution

7.3.1. Every identity of `spec-ledger.md` 9.1 holds with Transit accounts: they are
non-boundary accounts, never negative. They are in the contract sector of 9.3 there.

7.3.2. **Per shipment.** For each line, the quantity shipped equals the quantity
delivered to the consignee plus the quantity diverted to the estate plus the quantity
that spoiled in transit (plus the quantity lost, which is 0 until RSK), exactly
(FR-TRN-028). The spoiled quantity is measured, not assumed: it is the line's shipped
quantity less what the Transit account held at arrival.

7.3.3. **Attribution.** The ledger attributes a flow to the region of the account on the
non-boundary side (`spec-ledger.md` 3.5). A spoilage or loss from a Transit account is
therefore attributed to the **origin** region. For `transit_out` and `transit_in`, both
sides non-boundary, the attribution rule is the ledger's: this spec does not set it, and
asks `spec-ledger.md` 7.4 to state it (unverified; open question 6). `BoundaryFlow`
reports only faucets and sinks (`spec-ledger.md` 12.1), so `transit_out` and
`transit_in` do not appear there.

### 7.4 Audit and balance sheets

7.4.1. **Audit.** `World.Audit` (`spec-ledger.md` 12.4) also checks, when the module is
on: every in-flight shipment with a Transit account references an open Transit account
of kind Transit and region `From` that holds only commodities of its lines and no
more of each than was shipped; every Transit account no shipment references is empty;
shipment IDs are unique and every `Arrives` is at or after the current tick; every
away traveller is the traveller of a live shipment and every live self-carried shipment
has an away traveller. A failure names the check and the shipment and wraps
`ErrAuditFailed`.

7.4.2. **Balance sheets are measurement only** (`spec-ledger.md` 9.2). Goods in a Transit
account are on no agent's sheet. A measure that wants them on the shipper's or the
consignee's reads the shipment list through `InTransit(agent)`, a read-only query that
returns the lines of the shipments in which the agent is shipper or consignee, and
values them at a price the caller supplies. A self-carried trip to oneself keeps its
goods in the traveller's own account, so they stay on its sheet. Indicators decide how
to present either (`spec-indicators.md`).

## 8. The carriage unit: the decision on N-076

### 8.1 The question

`spec-carriage.md` 2.2 says hired carriage is traded in a carriage market for each
directed route, cleared by the same `Clearing`. Its section 4.2 found that this is true
of matching only: `spec-clearing.md` 2.1 identifies a market by a region and a
commodity, admits an ask only for stock the agent holds (`insufficient_stock`) and
settles by moving goods; hired carriage moves none between the two parties, and its fee
is held in escrow. The unit may also fall below the resolution floor (an ask of 24
against a floor of 50). N-076 asks the transport spec to choose between:

- **Option 1:** the carriage unit is a produced good (a trip is a job); or
- **Option 2:** a contract-aware admission and settlement that keeps the guarantees of
  `spec-clearing.md` 2.3.

### 8.2 What each would take

| | Option 1: a produced good | Option 2: contract-aware market |
|---|---|---|
| Market key | unchanged: (region, commodity) | a route, or a (route, carriage) key: `spec-clearing.md` 2.1 changes |
| What backs a carrier's ask | goods it holds (haul units), the ordinary reservation | a capacity commitment that is not a ledger quantity: a new kind of reservation in `spec-ledger.md` 6, with its own release rule |
| Settlement | an ordinary `trade` of goods and money; cannot fail (`spec-clearing.md` 2.3) | a trade that creates a contract: money into escrow, a commitment on a trip; a new settlement path that must also be total |
| Guarantees of `spec-clearing.md` 2.3 | untouched | to be re-proved for the new admission and settlement |
| Beliefs, tradable set, events | a haul commodity is a commodity: nothing changes | a belief and a tradable-set entry for a pseudo-commodity, and `TradeExecuted` fields for a route |
| The price floor problem (4.2 of `spec-carriage.md`) | solved in data: the unit is chosen by the pack (a coarser carrying unit and span) | still there, and needs the same coarser unit |
| Cost of capacity | the producing recipe's `UnitCost`; the belief cost band already anchors an ask to it | the formula `fee_min` of `spec-carriage.md` 2.2 and a margin `m_bp` that has no name |
| Specs touched | none beyond the edits of 8.5 | CLR (2.1, 3.1 to 3.3, 6, 9), LED (6), BEL, AGT (tradable set), OBS and CAR |
| Escrow and delivery risk | none: the units are paid for at the trade and the carrier carries nothing | the point of the option: money held until delivery |
| Capacity timing | produced ahead of use, as a flow; unsold units spoil | committed per trip |

### 8.3 The decision

**Option 1.** The carriage unit is a produced good: the **haul unit** of a hired mode
(3.1), an ordinary commodity made by an ordinary recipe of a carrier role, traded in
the shipper's region's ordinary market, and redeemed by the shipment mechanism at
dispatch (3.3.3). The reasons, in order of weight:

1. It leaves `spec-clearing.md` unchanged and with every guarantee: orders backed at
   admission, settlement that cannot fail, markets independent of clearing order. A
   contract-aware market would touch at least six specs (the "specs touched" row of 8.2) before
   any shipment moved.
2. A trip is a job. The carrier's households work for `Duration` ticks, consume feed,
   wear a vehicle held as a tool, and are paid by selling what the job made. Working
   ticks, the welfare ratio, the failure ladder, entry and exit apply to carriers with
   no special case, and the price of capacity is anchored to its cost by the same
   cost band as any good (`spec-beliefs.md` 3.2.2; `spec-production.md` 9). No transport
   price is configured.
3. The unit is chosen in data. A carrying unit of a realistic capacity and a span of
   several distance units makes a haul unit cost far above the resolution floor
   (17.4: 1 195 against a floor of 50), which settles the second finding of
   `spec-carriage.md` 4.2.
4. The mixed-load over-charge of `spec-goods.md` 3.5 disappears for transport: a load
   buys `n` carrying units by its binding fill (3.2.1), which is exact for one vehicle
   of the mode, and `BulkFactor` is not needed.

### 8.4 What this costs, stated

- **Capacity is produced ahead and unsold capacity spoils.** The carrier bears the
  demand risk: it makes haul units before it knows who will buy, and a unit that is
  not bought before its `ShelfLife` is lost to `spoilage`. This is a production delay
  with a response to price: the cobweb of STB-L21 (entry TRN-A3, section 15). A pack
  gives the haul commodity a short `ShelfLife`; a haul commodity that never spoils is
  warned (12.2) because it lets capacity be banked.
- **There is no escrow and no carrier custody.** A shipper pays when it buys the units
  and the goods travel alone, so no carrier can fail to deliver, and `FR-CAR-001` and
  `FR-CAR-002` do not apply to hired carriage (8.5). The risk of cargo not arriving is
  RSK's `loss`, borne by the consignee's side. A bilateral contract with a
  negotiated fee, a deadline and an escrow remains possible and is MER's, built on the
  shipment mechanism and `spec-ledger.md` 6.6; it does not go through the book.
- **A hired shipment needs a consignee in the destination.** The shipper stays in its
  region, so the goods go to another agent there. A merchant that wants to sell in the
  destination itself travels (self-carried) or has a counterpart there. Which, and why,
  is MER's.
- **The backhaul double count stays.** `spec-carriage.md` 5.1 asks how a carrier
  prices a leg it expects to fill on the way back. Here a carrier's households make
  haul units for their own region's market, so each direction's capacity is made by
  that region's carriers and the whole round trip is in its cost. A pack lever is to
  state the recipe's `Duration` and output for one direction only. The question stays
  open (20.2).
- **Lumpiness.** A load pays for whole carrying units. A small cargo pays for a whole
  unit (17.4: the same 7 170 for 100 units of a staple as for 400): the freight share
  of a small cargo is high. Smaller carrying units are the pack's remedy.

### 8.5 What changes in the other specs

The edits are proposed in the handoff and not made here (the supervisor integrates
shared files). For `spec-carriage.md`:

- 2.1: a fee is the price of haul units bought in an ordinary trade (reason `trade`);
  escrow (`escrow_in`, `escrow_out`, `escrow_refund`) applies to a bilateral contract
  of MER's, not to the market. FR-CAR-001 and FR-CAR-002 move to the contract spec.
- 2.2: the carriage market per directed route becomes the haul commodity of a hired
  mode traded in the origin region's market. The unit is the haul unit; its chargeable
  size is replaced by the load's binding fill (3.2.1). The ask is anchored to the
  recipe's `UnitCost`; `fee_min` and `m_bp` go, closing the unnamed `m_bp` of N-016.
  FR-CAR-003 stays true in a weaker form (no new extension interface), FR-CAR-005,
  FR-CAR-006 and FR-CAR-007 are superseded, FR-CAR-008 moves here (FR-TRN-009).
- 2.3, 2.4: a carrier's wage is its observed wage as any producer's; the "commercial
  trip" of `spec-welfare-and-wages.md` 4.3 is a self-carried trip only. FR-CAR-009 and
  FR-CAR-010 are reworded, FR-CAR-011 and FR-CAR-013 stand with the haul unit.
- 3, 4.2, 5: `m_bp` and the finding about the carriage market are resolved by this
  section; the backhaul question (5.1) is restated as open (20.2).

`spec-clearing.md` needs no change. Its scope sentence "trade between regions
(`spec-transport.md`, `spec-merchants.md`)" stays true, and a haul unit is an ordinary
commodity. `spec-ledger.md` gains the rows of 7.2, the reuse rule of 7.1 and the audit
checks of 7.4.1. `spec-goods.md` 3.5 and its open question 1 are answered for transport
(8.3, point 4) and remain open for storage.

## 9. What transport exposes

### 9.1 To a strategy

`AgentView` (`spec-agents.md` 6.4.1) gains two groups, plain data, bounded by the
configuration:

| Group | Content |
|-------|---------|
| `Inbound` | per commodity with units on the way to this agent as consignee: the total quantity and the earliest arrival tick, from the shipments dispatched before the start of phase 3 of this tick (so it does not depend on the order in which agents are called) |
| `Routes` | for each region other than the agent's and each mode: whether there is a route, the travel time `k` (section 4), and for a hired mode the haul units per carrying unit `s` (3.3.3); an outside node is reported with no route (`outside_destination`, 5.4) |

`Inbound` exists so that a consignee does not buy again what is already on its way: a
buyer that ignores its pipeline orders twice (TRN-A2). Whether the default strategy's
`Target` counts inbound units is AGT's (open question 7); the field is the means.
An agent that is away is never passed a view.

### 9.2 Queries

Read-only functions of the world, available to a consumer and to later specs:
`Travel(from, to, mode)` returns the travel time and the haul units per carrying unit
or that there is no route; `InTransit(agent)` (7.4.2); `Away(agent)`. None changes
state or draws a random number.

### 9.3 The information consequence

INF owns rumours and delay. Transport adds exactly this, and no more:

- the **arrival tick is known at dispatch** and is deterministic, to the shipper and
  the consignee through `Inbound` and `ShipmentDispatched`;
- a **self-carried traveller is in the destination region from the first phase of its
  arrival tick**, so it sees that market as any local agent does; before that it is in
  no market. Transport does not carry a price from the origin with it; whether a
  traveller's remembered prices count as information, and how stale they are, is INF's
  and MER's;
- an **arrival is an event** (`ShipmentArrived`); no price, no rumour and no belief
  changes because of it. Beliefs of a traveller are kept across the move; whether a
  traveller reseeds them is MER's, as MIG's for a migrant (N-038).
- **INF defines rumours** (`spec-information.md` 5); a trip carries no news back (`spec-information.md` 7.5).

### 9.4 Lanes for the viability solver

`spec-viability.md` 9.4 takes a list of lanes as data and says transport decides who
supplies them. For each hired mode, ordered pair of regions with a route, and
commodity `c`, transport can derive one lane with no loss (RSK adds loss) and a labour
charge in household-ticks per 10 000 units of `c`:

```
f      = max(Mass_c / UnitMass, Bulk_c / UnitBulk)       the binding fill of one unit, a rational; a term whose unit capacity is 0 is left out
labour = ceil( 10_000 * f * s * Duration / Output )       Duration and Output of the cheapest recipe
                                                          that makes the haul commodity, Output at E = 10 000
```

computed with exact integer arithmetic (the rational compared by cross-multiplication,
128-bit intermediates). It is a pure function of the configuration, `transport.Lanes`,
that a consumer passes to the solver as an option; the solver does not call it, so the
viability check stays independent of this module. The lane charges labour only: the
feed and the vehicle wear of the carriers' recipe are not in the programme, which is a
stated limitation of the check (open question 8). Self-carried modes give no lanes in
version 1.

## 10. Hooks reserved

10.1. **Risk (RSK).** Reserved, not specified. The place is between dispatch and
arrival: a transfer from the Transit account to the boundary under `loss` before the
delivery of 5.7, which the arrival event would report as `Lost` (a field that is 0
until RSK adds the mechanism; adding a field is allowed, observability-and-logging.md
4.2). A loss that falls on a traveller's own holdings is RSK's to define. RSK decides
whether a risk is per edge, per mode or per shipment, and what it draws.

10.2. **Infrastructure (CAP).** Reserved: how a road, a bridge or a ship changes a
travel time. The place is the transit factor `F` of 4.1, per (from, to, mode), 10 000
by default and up to `MaxTransitFactorBP`; a faster road raises it. The same slot
could carry an edge limit (`edge_full`, 6.2). Nothing in this spec reads a road.

10.3. **The transit factor slot.** `F` is world state, not configuration: a table of
the factors that differ from 10 000, keyed by (from, to, mode), set between ticks by
the integration spec for a shock (a closed road is 0) or by CAP, and saved in a
checkpoint (14). It acts on new dispatches only; shipments in flight keep the arrival
tick they were given (4.3). The setting function is the integration spec's; this spec
owns the slot and its effect (as production owns the yield-shock slot,
`spec-production.md` 6.8).

10.4. **Storage (STO).** None: the destination is an account, not a warehouse. A cargo
that arrives is in the consignee's account and subject to whatever capacity rule STO
adds then.

10.5. **Migration (MIG).** A household's migration is not a shipment and is not a
working trip (`spec-welfare-and-wages.md` 4.3). It may use `Travel` to know how long a
journey takes. Its region change is the public relocation of `spec-regions.md` 6.1, refused
with `ErrRegionBound` (`open_shipment`) while the agent is away or has a shipment in flight.

## 11. Parameters

All are tunables of the mode table, with a default, a bound, a doc line and an owner
(architecture.md 9.5 to 9.7). **Every default is a placeholder** (N-015, N-027), and
most fields are required with no default because a speed or a capacity has none that
could be right for every setting. The bounds are validation bounds in `bounds.go`
(`spec-config.md` 4.1).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Transport | `Modes` | the mode entities (3.1) | none | at most `MaxModes` |
| Mode | `ID` | identifier | required | the ID pattern, no duplicate |
| Mode | `Speed` | distance units per tick | required | 1 to `MaxSpeed` |
| Mode | `UnitMass` | mass capacity of one carrying unit | required | 0 to `MaxCapacity`; not both 0 |
| Mode | `UnitBulk` | bulk capacity of one carrying unit | required | 0 to `MaxCapacity`; not both 0 |
| Mode | `Crew` | households per carrying unit (self-carried) | 1 | 1 to `MaxWeight` |
| Mode | `Vehicle` | durable commodity per unit (self-carried) | none | known, `Durability` above 0 |
| Mode | `WearPerTick` | wear per vehicle per tick | 0 | 1 to `Durability` with a vehicle, else 0 |
| Mode | `Feed` | goods consumed per unit per tick (self-carried) | none | at most `MaxFeed`; quantity 1 to `MaxQuantity` |
| Mode | `Haul` | the commodity of haul units (hired) | none | a known commodity |
| Mode | `HaulSpan` | distance one haul unit carries one unit | required with `Haul` | 1 to `MaxHaulSpan` |

New bounds for `bounds.go` (provisional, with the reasons): `MaxModes` 64 (a loop over
modes per request and per view); `MaxSpeed` 10^9 (with `MaxTransitFactorBP` the product
`v * F` fits `int64`); `MaxCapacity` 10^12 per carrying unit (times `MaxWeight` of 10^6
it is 10^18, which fits `int64`); `MaxFeed` 4 (as `MaxRequires`);
`MaxHaulSpan` 10^12; `MaxShipmentLines` 16 (the loop and the size of an event);
`MaxDispatchPerTick` 8 (requests an agent's plan may carry); `MaxTravelTicks` equal to
`MaxTicksPerYear` (100 000: a journey of a year of ticks, and a duration times a count
does not overflow); `MaxTransitFactorBP` 100 000 (a road ten times faster).

Read here and owned elsewhere (no row): `Mass`, `Bulk`, `ShelfLife`, `Durability`
(`spec-goods.md` 8); `Distance` (REG); `MaxWeight`, `MaxQuantity`, `MaxTicksPerYear`
(`spec-config.md` 4.1). The module adds no world-level scalar. `BulkFactor` is not used
by transport.

The placeholders, so that none is mistaken for a result: every default and bound above;
every number of the two example settings (3.1.2) and of the worked example (section 17).

## 12. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). Errors name the path,
for example `modules.transport.modes.wagon.vehicle`.

### 12.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field outside its bound in section 11; `UnitMass` and `UnitBulk` both 0; more than `MaxModes` modes or `MaxFeed` feed entries |
| `ErrDuplicateID` | a mode ID twice; a commodity twice in `Feed` |
| `ErrUnknownCommodity` | a `Vehicle`, `Feed` or `Haul` that is not defined |
| `ErrVehicleNotDurable` (new) | a `Vehicle` whose commodity has `Durability` 0 |
| `ErrModeShape` (new) | a hired mode (`Haul` set) with `Crew`, `Vehicle`, `WearPerTick` or `Feed`, or without `HaulSpan`; a self-carried mode with `HaulSpan`; `WearPerTick` without `Vehicle` or `Vehicle` without `WearPerTick` |
| `ErrModuleDependency` | the module on while the region graph (REG) is off |
| `ErrOverflow` | `MaxWeight` times `MaxCapacity` above `int64` (a derived check of `spec-config.md` 4.2, true at the bounds above) |

### 12.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2).

| Warning | Condition |
|---------|-----------|
| `transport_without_routes` | the module is on with fewer than two regions, or no modes |
| `haul_unsourced` | a hired mode's `Haul` commodity that no recipe makes and that is not raw: no one can sell it |
| `haul_never_spoils` | a `Haul` commodity with `ShelfLife` 0: capacity could be banked without cost |
| `haul_durable` | a `Haul` commodity with `Durability` above 0 |
| `vehicle_wears_in_one_trip` | `WearPerTick` times the longest travel time of the mode over the graph exceeds the vehicle's `Durability` |
| `trip_longer_than_window` | the longest travel time of a self-carried mode over the graph exceeds the base `WelfareWindow` (`spec-welfare-and-wages.md` 2.3.1): a trip is a gap in income like a long job (STB-L19, TRN-A6) and the window is derived from recipes only |
| `route_too_slow_for_shelf_life` | for a commodity with a `ShelfLife` and a mode, every route of the graph takes `k` of `ShelfLife` or more: nothing perishable could ever be delivered by that mode (5.6.2). A report, bounded by commodities times modes |

## 13. Observability

13.1. **Events.** Three, in the style of `OrderPlaced` and `OrderRejected`, each a
state transition of one shipment or one request (observability-and-logging.md 4.3):

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `ShipmentDispatched` (new) | when a shipment is dispatched | `Shipment`, `Mode`, `Region` (origin), `To`, `Shipper`, `Traveller` and `Hired`, `Consignee`, `Arrives`, `Units`, `HaulUnits`, `Lines` (commodity and quantity, a copy in commodity order) |
| `ShipmentArrived` (new) | when a shipment arrives, after its delivery | `Shipment`, `Mode`, `Region` (origin), `To`, `Receiver` (the consignee, or the destination estate pseudo-agent if diverted), `Diverted`, `Lines` (commodity, `Shipped`, `Delivered` to the consignee, `Diverted` to the estate, `Spoiled`) |
| `ShipmentRejected` (new) | when a request is refused | `Region`, `Agent`, `Mode`, `To`, `Reason` |

None has a map, a pointer or a wall-clock time (observability-and-logging.md 4.1).
`Lines` are slices of copies in a deterministic order, bounded by `MaxShipmentLines`.
The events are constructed only when an observer was supplied (4.5 there). An event of
a module that is off is not emitted. The order within a step: arrivals in shipment ID
order (phase 1), each `ShipmentArrived` after its delivery; then, in phase 3, for each
agent in ID order, its `ShipmentDispatched` and `ShipmentRejected` events in plan
order, before that agent's `OrderPlaced` events. The goods that a trip consumes and
the units redeemed are in `BoundaryFlow` under `transit_use` and `haul_use`; spoilage
and wear in transit are under `spoilage` and `wear`. No event is emitted per ledger
transfer.

13.2. **Logs.** `Debug`, guarded by `Enabled`, constant messages: "shipment dispatched"
(`shipment`, `mode`, `region`, `dest`, `agent`), "shipment arrived" (`shipment`,
`dest`), "shipment refused" (`agent`, `mode`, `reason`), "cargo diverted" (`shipment`,
`dest`, `agent` of the missing consignee). There is no `Info` line in the tick and no
`Warn` per shipment: a shipment and a refused request are per entity (observability-and-
logging.md 3.2.2), and their facts are events. At construction the summary line gains
`modes` (the count) and the module name in `modules`. The warnings of 12.2 are
reported through the configuration's warning channel (`spec-config.md` 7.2), not
logged per tick.

13.3. **New log keys** for observability-and-logging.md 3.3.2: `shipment` (the shipment
ID), `mode` (the mode ID), `dest` (the destination region ID; `region` is the origin),
and `modes` (a count in the construction summary). The `reason` values of 5.4 are
constants in `internal/logkeys`.

13.4. **Metrics** (observability-and-logging.md 5.3), all `Int64`, each derivable from the
events:

| Name | Instrument | Unit | Attributes | Fed by |
|------|------------|------|------------|--------|
| `economysim.transport.shipments` | Counter | `{shipment}` | `economysim.mode` | `ShipmentDispatched` |
| `economysim.transport.in_transit` | UpDownCounter | `{unit}` | `economysim.commodity` | `ShipmentDispatched` (+ each line), `ShipmentArrived` (- each line's `Shipped`) |
| `economysim.transport.cargo.spoiled` | Counter | `{unit}` | `economysim.commodity` | `ShipmentArrived` (`Spoiled`) |

`economysim.mode` is bounded by the configuration (5.4.1 there; at most `MaxModes`);
`economysim.commodity` already is. The series added for the reference chain: one per
mode and, for the two commodity metrics, one per commodity that is ever shipped
(5.4.2 there), at most 30.

13.5. **Snapshot.** The list of shipments in flight (the record of 5.2 with the lines
and the time left), the set of away agents, and the transit factors that differ from
10 000. Bounded by the shipments in flight and the configuration, a deep copy
(architecture.md 12.4).

13.6. **Cost with no observer.** Travel time, the fit test and the load sum use scratch
held by the world and allocate nothing (NFR-TRN-001); events are not built
(NFR-TRN-004). Observation never changes results.

## 14. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002 consequences). This spec's, plain
data in a fixed order with no maps:

- the next shipment ID;
- the shipments in flight, in ID order: every field of 5.2 (the mode and regions by
  their position in the canonical order of the configuration, the lines in commodity
  order);
- the transit factors that differ from 10 000, in the order of (from, to, mode).

**Not saved:** the away flags (derived: a traveller is away if it is the traveller of
an in-flight self-carried shipment), the free lists of Transit accounts (derived, 7.1.3),
the arrival schedule (derived from `Arrives`), the travel times, the routes and
`Inbound` (derived from the configuration, REG and the shipment list), and scratch. The
Transit accounts themselves, with their lots, are the ledger's state
(`spec-ledger.md` 11). A world restored between ticks continues exactly as an
uninterrupted one, with the same arrivals in the same order (FR-TRN-036). The modes are
configuration and are in its hash.

## 15. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9. Fields as in its section 3.
**Status is `hand` unless a model is cited.** The register owner assigns the
`STB-Lnn` numbers; entries here are `TRN-An`.

**TRN-A1. Price-gap shipping with a transit delay.** *Kind:* loop. *Loop:* the price
gap between a dear and a cheap region (+) the quantity shipped (+) the price in the
cheap region, at dispatch, where the shipper bought (-) the gap; and the gap (+) the
quantity shipped (+, after `k` ticks) the arrivals in the dear region (-) the price
there (+) the gap. Each path has one negative link, so each is balancing (B); the first
is immediate and the second has the delay.
*Delay:* the travel time `k` (section 4). *Owner:* TRN 5; the decision to ship is MER's.
*Damper:* the freight is a floor on the gap that triggers a shipment, so the loop closes
the gap to the freight cost and not to zero (TRN-A4); a shipper that counts `Inbound`
(9.1) removes the delay from its signal; a limit on what can be shipped per tick (the
capacity of one carrying unit, `MaxDispatchPerTick`) bounds the swing. *Undamped:*
shippers that ignore their pipeline and respond strongly. *Condition:* in the linearised
model of section 16, with `g` the fraction of the gap a shipper tries to close per tick
and none of the effect at dispatch, the gap decays if `g < 2 sin(pi / (2 (2k + 1)))`,
about `pi / (2k + 1)`: 1.0 at `k` = 1, 0.62 at 2, 0.35 at 4, 0.18 at 8; with at least half
of the effect at dispatch (the purchase in the origin region moves its price at once) or
with a shipper that counts `Inbound`, the limit is at least about 2 for any `k`. A cap on the
shipment per tick bounds the swing of the gap to about `k` times the cap. *Test:* a new
scenario (S28 proposed, below). *Status:* model (section 16). It does not show the
library: it has no inventory, no belief and no market, and the shares of the effect at
dispatch and at arrival are not derived for any pack.
INF-A1 (`spec-information.md` 13) adds the rumour's lag `ell` to the delay of a shipper that reads a rumour: the gap decays if `g < 2 sin(pi / (2 (2 (ell + k) + 1)))` (`spec-information.md` 11.1).

**TRN-A2. The pipeline is invisible.** *Kind:* amplifier. *Loop:* a consignee's stock
(excludes goods on the way) (-) its orders for the good (+) the goods on the way (+) its
stock on arrival: a buyer that sees only its stock orders again while its goods are in
transit, then holds twice what it wanted `k` ticks later and stops ordering, an
oscillation of period about `2k` that grows with the delay. R in the stock signal, with
the delay `k`. *Owner:* TRN 9.1 (the `Inbound` field); AGT's `Target` (open question 7).
*Damper:* the buffer cap and the production halt (D24) bound the overhang once it
exists; `Inbound` is the means to avoid it. *Undamped:* a strategy that does not count
`Inbound`. *Condition:* the stock a strategy sizes against includes `Inbound`. *Test:*
S28 with a consignee that buys and receives. *Status:* hand.

**TRN-A3. The haul capacity cobweb.** *Kind:* loop (a case of STB-L21). *Loop:* the price
of haul units (+) the expected price (+) carriers' starts of the haul recipe (delay: its
`Duration`, the round trip) (+) the later supply of haul units (-) their price. B with a
delay, the cobweb of STB-L21 and 6.3 of the register: the delay sets the period and the
elasticity condition `es < ed` sets the threshold. *Delay:* the haul recipe's `Duration`.
*Owner:* TRN 8.4, PRO 16, BEL 17. *Damper:* D02, D03, D14; the short `ShelfLife` of the
haul commodity ends banked capacity and so shortens the memory of the stock. *Undamped:*
the same as L21: a response to price of the carriers' starts that is above the shippers'
demand elasticity. *Condition:* as L21. A shipper's demand for haul units is derived
demand, inelastic in the haul price over the range where the goods' price gap exceeds the
freight, so the condition is likelier to fail here than for a final good: the register
should record the measured elasticity. *Test:* S05 with a haul recipe, S28. *Status:*
hand (merges with L21).

**TRN-A4. Freight and the delivered price.** *Kind:* loop (extends STB-L17 and replaces
its formula). *Loop:* the delivered price of a good (+) the carriers' upkeep and feed
cost (+) the cost of a haul unit (`UnitCost`) (+) the freight (+) the delivered price. R
across regions with a gain below 1 (the freight is a share of the delivered price).
*Owner:* TRN 8.3; CAR 2.2. *Damper:* none, but the gain is below one for any real
freight share. *Undamped:* a route with a high freight share and a high carrier feed
cost. *Condition:* the freight is a small share of the delivered price (17.4: 6% for a
full unit, 24% for a small cargo, placeholders). *Test:* S04 on a two-region chain,
X1. *Status:* hand. Replaces the `fee_min` form of L17 (`spec-carriage.md` 2.2).

**TRN-A5. Perishables cannot cross.** *Kind:* hazard. A commodity with `ShelfLife` `S`
cannot be delivered by a route of `k >= S` ticks (5.6.2), so a price gap in it never
closes and its shipments, if a strategy makes them, spoil whole. Not an instability: a
pinned gap, counted by `ShipmentArrived.Spoiled`. *Owner:* TRN 5.6. *Damper:* the
warning `route_too_slow_for_shelf_life` and the strategy's own use of `Routes`.
*Condition:* a strategy ships a perishable only if `k <= S - 1 - age`. *Test:* a unit
test of 5.6.2 and S07's perishable in a two-region pack. *Status:* hand.

**TRN-A6. A trip is a gap in income.** *Kind:* loop (an instance of STB-L19). *Loop:* a
self-carried trip (no income until the goods sell at the destination) (-) the welfare
ratio over the window (-) relief or a role change by the failure ladder (rung 1, 2)
(+) the traveller's absence from its trade. B. *Delay:* the trip, against the effective
window. *Owner:* TRN 3.3.1; WEL 2.3.3 (the window derives from recipes only). *Damper:*
D29 does not cover it. The warning `trip_longer_than_window` is the only guard; a
traveller that leaves with enough money and provisions (`unprovisioned`, 5.4) survives
it. *Undamped:* a trip longer than the window. *Condition:* the longest trip of a
travelling role is below its effective window. *Test:* S11's long-job pack with a
travelling role. *Status:* hand. A trader's labour is also lost to production while it is
away: the total supply of its household-ticks falls with every trip.

**TRN-A7. Vehicles are capital.** *Kind:* loop (merges with STB-L26). A fleet bought
together wears at the same rate and is replaced together; transport adds wear charged
per tick of travel at dispatch (3.3.2). *Damper:* D19 (a spread of ages by the starting
inventories). *Status:* hand.

**TRN-A8. Diversion transfers wealth to the pool.** *Kind:* hazard (a case of L63).
Cargo whose consignee is gone goes to the destination estate (5.8) and its value to the
regional pool, which relief and entrants share. Bounded by the shipments in flight at a
removal. *Damper:* D37. *Status:* hand.

**Dampers this spec adds** (for the catalogue of `spec-stability.md` 5):

| ID | Damper | Home | Status |
|----|--------|------|--------|
| D38 (proposed) | The pipeline view: `Inbound` lets a consignee or shipper count goods on the way; its cost is the field in every view | TRN 9.1 | specified (the field); the use is AGT's |
| D39 (proposed) | The dispatch cap: a plan carries at most `MaxDispatchPerTick` requests and a traveller makes one trip at a time; its cost is that a large shipment must be one request | TRN 5.4, 6.2 | specified (a bound) |
| D40 (proposed) | Freight as a floor on the gap: a good is shipped only if the gap exceeds the haul price, so shipping closes the gap to the freight and no further; its cost is a persistent gap | TRN 3.3.3, 8 | specified (emergent from the market price of haul units) |

**Scenarios proposed:** S28, *two regions with a route and a merchant that ships on a
price gap, with the shipper counting `Inbound` and not*: the gap closes to within the
freight plus a band and does not oscillate with a growing amplitude for `k` in a stated
range (TRN-A1, A2); S29, *a perishable with `ShelfLife` equal to the travel time and one
less*: the first is never delivered, the second always (TRN-A5); S30, *a hired mode with
a haul recipe whose `Duration` is the round trip, at three elasticities of the carriers'
starts*: the price of haul units converges or is reported as F6 (TRN-A3).

## 16. What was run

Two scripts in `doc/specs/models/trn/`, run in Docker (`python:3.12-slim`, the standard
library only), as the README of that folder says.

**`pipeline.py` and `pipeline.out`** model TRN-A1: the gap `x` between two regions,
a shipment `q[t] = g x[t]` each tick, a share `a` of its effect on the gap at dispatch
and `1 - a` at arrival `d` ticks later: `x[t+1] = x[t] - a q[t] - (1 - a) q[t-d]`. It
finds the largest gain `g` that decays, by a scan from 0 that stops at the first gain
that does not decay. Results:

| `d` (ticks) | 0 | 1 | 2 | 3 | 4 | 6 | 8 | 12 | 20 |
|-------------|---|---|---|---|---|---|---|----|----|
| `a = 0`: largest decaying `g` | 1.998 | 0.998 | 0.618 | 0.444 | 0.346 | 0.240 | 0.184 | 0.124 | 0.076 |
| theory `2 sin(pi / (2 (2d + 1)))` | 2.000 | 1.000 | 0.618 | 0.445 | 0.347 | 0.241 | 0.185 | 0.126 | 0.077 |
| `a = 0.5` | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 |
| shipper counts what is in flight, `a = 0` | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 | 1.998 |

The `a = 0` row agrees with the known limit of the delayed feedback
`x[t+1] = x[t] - g x[t-d]`, to the grid step of 0.002. With a share `a` of 0.8 the
limit is 3.33 at `d` = 1, 2.39 at 3 and 1.998 elsewhere (a large immediate share can
allow a gain above 2); the table does not list it. Above the limit, a cap on the shipment
per tick `c` bounds the swing of the gap: with `a = 0` the amplitude was about `1.96 c`
at `d` = 2 and about `5.1 c` to `5.7 c` at `d` = 6, i.e. roughly `d c` (at 1.5 and 3 times
the limit). **What it does not show:** a linearised gap with no inventory, no belief, no
quantity sizing and no market; a single shipper type; `a` is a free parameter and is not
derived for any pack; no noise, no capacity limit other than the cap, no seasons.

**`worked.py` and `worked.out`** check the integer arithmetic of section 17: shortest
distances, travel times, the fit tests, the spoilage timing and the haul unit cost.

## 17. Worked example

All numbers are placeholders in an example setting; the arithmetic is in `worked.py`.

### 17.1 The graph and the travel times

Three regions, `north`, `mid` and `south`. REG owns the graph; this is an example of
one (directed edges, distances in an abstract unit):

| Edge | `north`>`mid` | `mid`>`north` | `mid`>`south` | `south`>`mid` | `north`>`south` | `south`>`north` |
|------|---:|---:|---:|---:|---:|---:|
| Distance | 40 | 40 | 60 | 60 | 90 | 110 |

REG's shortest distances (4.4): `north`>`south` is 90 (the direct edge beats 40 + 60 =
100); `south`>`north` is 100 (through `mid`, 60 + 40, beats the direct 110). The modes
are those of 3.1.2: `foot` (speed 20), `wagon` (30), `coach` (30, hired). Travel times
`k = ceil(D / v)`:

| Pair | Distance | `foot` | `wagon` and `coach` |
|------|---------:|-------:|--------------------:|
| `north`>`mid` | 40 | 2 | 2 |
| `mid`>`north` | 40 | 2 | 2 |
| `mid`>`south` | 60 | 3 | 2 |
| `south`>`mid` | 60 | 3 | 2 |
| `north`>`south` | 90 | 5 | 3 |
| `south`>`north` | 100 | 5 | 4 |

Goods (placeholders): `staple` mass 3, bulk 2, price 300; `fuel` mass 2, bulk 6;
`fresh` mass 1, bulk 1, `ShelfLife` 3; `fodder` mass 2, bulk 3, price 40; `wagon`
a durable of `Durability` 400, price 6 000.

### 17.2 A self-carried trip on foot: capacity, the whole account and spoilage

A trader of weight 1 in `north` at tick 10 holds 10 `staple`, 1 `fuel` and 4 `fresh`
(born in tick 10), and its upkeep is 1 `staple` a tick. A `foot` unit is (40, 30), `Crew` 1,
so it fields 1 unit.

- Its holdings are all load (3.2.2): staple `10 * (3, 2)` = (30, 20), fuel (2, 6), fresh
  (4, 4): the load is (36, 30). It fits (40, 30) with bulk exactly at its limit: `n` = 1.
- With 3 `fuel` instead of 1 the load is (40, 42): mass fits, bulk 42 is over 30, so the
  request is refused `capacity_exceeded`. This is the pair rule: `BulkFactor` would have
  charged the mixture differently, and a pair is exact.
- Dispatched to `south`: `k = 5`, arrival at tick 15. The trader is away in ticks 10 to 14,
  five working ticks. It needs 5 `staple` of provisions after the dispatch and holds
  10 (`unprovisioned` would otherwise apply). It eats one a tick.
- The `fresh` has `Born` 10 and `ShelfLife` 3, so it spoils at the end of tick
  `10 + 3 - 1 = 12`, in the trader's account, to the `spoilage` boundary, attributed to
  `north`. At arrival it holds 5 `staple` and 1 `fuel`. With 5.6.2, a trip of 1 or 2 ticks
  would have delivered the `fresh` (arrival in tick 11 or 12, at most 12) and one of 3
  would not (13).

### 17.3 A self-carried trip by wagon

A pair of households (weight 2, `Crew` 2) with one `wagon` item in `north` ships to
`south` (`k = 3`). `units = min(floor(2 / 2), 1)` = 1. Cost: feed `3 * 1 * 3` = 9 `fodder`
to the boundary under `transit_use`; wear `WearItems(account, wagon, 1, 5 * 3)` takes the
wagon from durability 400 to 385. The pooled capacity is (1 200, 900).

### 17.4 Hired carriage and the cost of a haul unit

A shipper in `mid` sends 100 `staple` to a consignee in `south` by `coach`. The load is
(300, 200); the unit is (1 200, 900), so `n = max(ceil(300 / 1 200), ceil(200 / 900))` = 1.
The distance is 60 and the span 10, so `s = 6` and the shipment redeems `1 * 6` = 6
`haul` units, with `k = ceil(60 / 30)` = 2: dispatched in tick 20, arriving in tick 22.

Where the price of a haul unit comes from. A carrier role has a recipe `run_coach`:
`Duration` 4 (a round trip of 2 ticks each way at speed 30 over 40), inputs 12 `fodder`,
a tool `wagon` with `WearPerUse` 20, and an output of 4 `haul` per execution (one
carrying unit over the 40 of the loaded leg, divided by the span 10). With a wage of
1 000 per household-tick and a `wagon` price of 6 000 (`spec-production.md` 9.1):

```
input_cost  = 12 * 40                      = 480
wear_cost   = ceil(20 * 6 000 / 400)       = 300
labour_cost = 4 * 1 000                    = 4 000
cost        = 4 780
unit_cost   = ceil(4 780 * 10 000 / (4 * 10 000)) = 1 195
```

The cost anchor of a haul unit is 1 195, far above the resolution floor of 50
(`spec-currency.md` 3.1); the ask of 24 of `spec-carriage.md` 4.2 does not arise. The
freight at cost is `6 * 1 195` = 7 170. For 100 `staple` worth 30 000 that is 23.9%
(2 390 basis points); for a full unit of 400 `staple` (1 200, 800), the same 7 170 against
120 000 is 6.0% (597): the lumpiness of 8.4. The shipment's goods leave the shipper's
account under `transit_out` into a Transit account of region `mid`, the 6 haul units go
to the boundary under `haul_use`, and in tick 22 the 100 `staple` move to the consignee
under `transit_in`. If the consignee had been removed in tick 21 they would have gone to
`south`'s estate, flagged `Diverted`.

### 17.5 Which ledger operations happened

For 17.4 the flows of the tick are, by reason: `trade` (the shipper's purchase of haul
units, an earlier tick), `transit_out` 100 `staple` (internal), `haul_use` 6 `haul`
(sink, region `mid`), and in tick 22 `transit_in` 100 `staple` (internal). Goods are
conserved: shipped 100 = delivered 100 + diverted 0 + spoiled 0.

## 18. Notes settled

- **N-011** (transport, storage and carriage). Applied for TRN: the vehicle capacity is a
  pair per carrying unit and a load fits only if both sums are within it (3.2); the
  chargeable size and `BulkFactor` are not used by transport because a hired load buys
  `n` units by its binding fill (3.2.1, 8.3), which also answers "add a per-mode factor
  if the error matters": the unit pair is the per-mode factor; goods keep ageing in
  transit (5.6); cargo lost is `loss`, reserved for RSK (10.1); contract money is held
  in an Escrow account, not a reservation, and that is MER's for a bilateral contract
  (8.4). Left open for STO, CAR and MER.
- **N-046** (register entries owed). Left open for the other specs. TRN's entries are
  TRN-A1 to TRN-A8 of section 15.
- **N-076** (a carriage market is not a goods market). Decided: option 1, the haul unit
  (section 8), with the edits to CAR, LED and GDS listed in 8.5; close it when they land.
- **N-084** (scenarios that wait for trade). Left open: TRN supplies the mechanism for
  the dynamic parts of X1 and X2 (modes, the shipment, `Inbound`); the shipper's decision
  is MER's, rumours INF's. The static check of X1 can use the lanes of 9.4. S28 to S30 are
  new (15).
- **N-016, N-017, N-027** (ALL). N-016: every field of 11 has a row proposed for the
  parameter-owner table, and there is no unnamed parameter. N-017: the new terms are in
  the preamble list below and `traveller` is distinct from the carrier role. N-027: no
  default is claimed stable.
- Read and not touched: **N-010** (regions) is REG's; this spec adds that goods in transit
  are in Transit accounts, so a region change of an agent's account moves no cargo, and
  that an away traveller or a party to a shipment in flight cannot be relocated by anyone else (`open_shipment`, 5.5.3).

New terms for the glossary (README list and `doc.go`): mode, self-carried mode, hired
mode, carrying unit, crew (of a mode), haul unit, haul span, traveller, shipper,
consignee, shipment, dispatch, arrival, transit account, transit factor, away, inbound,
diverted (cargo).

## 19. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Defaults and
bounds are placeholders (N-015, N-027).

### Functional

| ID | Requirement |
|----|-------------|
| FR-TRN-001 | A mode MUST have a `Speed` from 1 to `MaxSpeed`, a `UnitMass` and a `UnitBulk` from 0 to `MaxCapacity` that are not both 0, and at most `MaxFeed` feed entries; a value outside a bound MUST be rejected with `ErrOutOfBounds`. |
| FR-TRN-002 | A mode with `Haul` set MUST have a `HaulSpan` from 1 to `MaxHaulSpan` and no `Crew`, `Vehicle`, `WearPerTick` or `Feed`; a mode without `Haul` MUST have no `HaulSpan`; a violation MUST be rejected with `ErrModeShape`. |
| FR-TRN-003 | A `Vehicle` MUST be a known commodity with `Durability` above 0 (`ErrVehicleNotDurable`), with a `WearPerTick` from 1 to its `Durability`, and a `Vehicle`, `Feed` or `Haul` that is not a defined commodity MUST be rejected with `ErrUnknownCommodity`. |
| FR-TRN-004 | With the module absent from the configuration, every shipment request MUST be refused with `unknown_mode`, no shipment, Transit account or transport event MUST exist, and a configuration with modes and strategies that request no shipment MUST give results identical to one without the module. |
| FR-TRN-005 | The travel time MUST be `max(1, ceil(D * 10 000 / (Speed * F)))` with `D` the shortest distance of the region graph and `F` the transit factor in basis points (default 10 000), computed with a 128-bit intermediate and checked arithmetic. |
| FR-TRN-006 | A request between regions with no path MUST be refused `no_route`, one with a transit factor of 0 `blocked`, and one whose travel time exceeds `MaxTravelTicks` `too_far`. |
| FR-TRN-007 | A shipment's arrival tick MUST be its dispatch tick plus its travel time at dispatch, and a later change of the transit factor or of the graph MUST NOT change the arrival tick of a shipment in flight. |
| FR-TRN-008 | The carrying units a self-carried traveller of weight `w` fields MUST be `floor(w / Crew)`, limited by the available vehicle items when the mode has a `Vehicle`; with none the request MUST be refused `crew_short` or `no_vehicle`. |
| FR-TRN-009 | A load MUST fit a trip if and only if its mass sum is within `units * UnitMass` and its bulk sum within `units * UnitBulk` (a capacity of 0 leaves that dimension unlimited), and a load that does not fit MUST be refused whole with `capacity_exceeded` and not reduced. |
| FR-TRN-010 | The carrying units a load needs MUST be `max(1, ceil(mass / UnitMass), ceil(bulk / UnitBulk))`, a term with a capacity of 0 left out, and the cost of a self-carried trip MUST be charged on those units and not on all the units fielded. |
| FR-TRN-011 | The load of a self-carried trip MUST be the pair of sums over every good in the traveller's account at dispatch, the request's lines included, except units of the mode's `Vehicle` commodity and the feed the trip consumes; money MUST NOT count. |
| FR-TRN-012 | A hired shipment MUST redeem `n * s` haul units from the shipper's account to the boundary under `haul_use`, with `s = ceil(distance / HaulSpan)`, and MUST be refused `insufficient_haul` if the shipper holds fewer. |
| FR-TRN-013 | At dispatch a self-carried trip with a `Vehicle` MUST apply `WearItems` to the `n` vehicles used for `WearPerTick` times the travel time, a vehicle that reaches 0 MUST be destroyed to the `wear` boundary, and the trip MUST NOT be interrupted by it. |
| FR-TRN-014 | At dispatch each `Feed` entry MUST move `quantity * n * k` units from the traveller to the boundary under `transit_use`, and a traveller holding less MUST be refused `insufficient_feed`. |
| FR-TRN-015 | Transport MUST NOT move money: no transfer of money MUST be made by dispatch, transit or arrival. |
| FR-TRN-016 | In phase 3 the world MUST execute, for each agent in ID order, the plan's discards, then its shipment requests in plan order, then turn its intents into orders; an agent MUST NOT make more than `MaxDispatchPerTick` requests in a plan, the rest being refused `too_many_requests`. |
| FR-TRN-017 | A request MUST be refused whole, with a reason of 5.4 chosen by the order of its checks, changing no state and opening no account, and MUST emit `ShipmentRejected`. |
| FR-TRN-018 | A request with a consignee MUST be refused `unknown_consignee` if the consignee is not an existing agent and `consignee_elsewhere` if it is not in the destination region, except a self-carried request to oneself; a request to oneself MUST have no lines (`lines_with_self`). |
| FR-TRN-019 | A self-carried request MUST be refused `traveller_away` if the agent is away and `job_in_progress` if it has a job in progress, and `unprovisioned` if after the dispatch's moves its stock of its upkeep goods is below its effective upkeep times its weight times the travel time. |
| FR-TRN-020 | Dispatch MUST move each line from the shipper to a Transit account under `transit_out`, with the lots keeping their born tick and remaining durability, and MUST take the lowest-ID open Transit account of the shipper's region that no shipment in flight uses, or open one. |
| FR-TRN-021 | A traveller MUST be away from its dispatch until the first phase of its arrival tick: not passed to `Work` or `Trade`, placing no order, with every tick from the dispatch tick to the tick before arrival a working tick, its account keeping its region and its upkeep consumed from its account. |
| FR-TRN-022 | With the module on, transport MUST register the `region_bound` refusal `open_shipment`, open for an agent that is an away traveller or the shipper or consignee of a shipment in flight, so that the public relocation returns `ErrRegionBound` for it; at arrival transport MUST change the traveller's region by the ledger-level `Relocate` with cause `transport` and emit `AgentRelocated` once, and MUST NOT use `NoRegion`, `depart` or `arrive`. |
| FR-TRN-023 | A Transit account MUST hold the cargo of at most one shipment in flight and nothing else, MUST NOT be closed, and MUST be empty when no shipment references it. |
| FR-TRN-024 | In the first phase of a tick, after the moment and before relief, the world MUST process the shipments whose arrival is that tick in order of shipment ID: change a self-carried traveller's region to the destination, move the cargo to the consignee under `transit_in`, release the Transit account and emit `ShipmentArrived`. |
| FR-TRN-025 | Goods in transit MUST age as in any account: a lot of a perishable commodity whose born tick plus shelf life minus 1 is at most the current tick MUST move whole to the `spoilage` boundary at the end of the tick from the Transit account, so that a lot is delivered if and only if the arrival tick is at most its born tick plus shelf life minus 1. |
| FR-TRN-026 | A durable cargo MUST NOT lose durability in transit. |
| FR-TRN-027 | A cargo whose consignee no longer exists, is not in the destination region, or could not hold the quantity MUST be moved to the destination region's estate account under `transit_in`, MUST be flagged `Diverted` in the event, and MUST NOT wait or be returned. |
| FR-TRN-028 | For each line of a shipment, the quantity shipped MUST equal the quantity delivered to the consignee plus the quantity diverted to the estate plus the quantity that spoiled in transit, exactly. |
| FR-TRN-029 | Removing a shipper MUST NOT change a shipment; removing a traveller MUST send its own holdings to the estate of its account's region, drop a shipment to itself, and leave a shipment with lines to arrive without the region change; removing a consignee MUST divert the cargo. |
| FR-TRN-030 | The ledger registry MUST contain `transit_out` (agent to transit), `transit_in` (transit to agent or estate), `haul_use` and `transit_use` (goods sinks, agent to boundary), and a transfer under one of them between accounts of the wrong kind MUST be refused with `ErrReasonMismatch`. |
| FR-TRN-031 | The reason `loss` MUST be unused by this module: no flow under it MUST exist with the module on. |
| FR-TRN-032 | `World.Audit` MUST also check, with the module on, that every Transit account holds only the commodities and at most the quantities of the one shipment that references it, that an unreferenced Transit account is empty, that shipment IDs are unique with arrival ticks not before the current tick, and that the away travellers and the live self-carried shipments correspond. |
| FR-TRN-033 | `AgentView` MUST carry `Inbound` (per commodity the total quantity on the way to the agent and the earliest arrival, from the shipments dispatched before phase 3 began) and `Routes` (per other region and mode the existence of a route, the travel time and for a hired mode the haul units per carrying unit), and an agent that is away MUST NOT be passed a view. |
| FR-TRN-034 | `InTransit(agent)` MUST return the lines of the shipments of which the agent is shipper or consignee, `Travel(from, to, mode)` the travel time and haul units per unit or the absence of a route, and neither MUST change state or draw a random number. |
| FR-TRN-035 | `ShipmentDispatched`, `ShipmentArrived` and `ShipmentRejected` MUST be emitted with the fields of 13.1, with no map, pointer or wall-clock time, in the order of 13.1, and MUST NOT be emitted when the module is off. |
| FR-TRN-036 | The shipments in flight, the next shipment ID and the transit factors that differ from 10 000 MUST be saved in a checkpoint in the order of section 14, the away flags and the free lists MUST be recomputed on restore, and the restored run MUST continue exactly as an uninterrupted one. |
| FR-TRN-037 | The modes MUST be part of the effective configuration and its hash, except `Label`. |
| FR-TRN-038 | Validation MUST emit the warnings of 12.2 without failing the configuration, and MUST reject the errors of 12.1 each with its sentinel. |
| FR-TRN-039 | The transit factor slot MUST default to 10 000 for every (from, to, mode), MUST affect new dispatches only, and a value above `MaxTransitFactorBP` MUST be refused. |
| FR-TRN-040 | A shipment's Transit account and its cargo MUST be attributed, for spoilage, to the origin region. |
| FR-TRN-041 | `transport.Lanes` MUST derive, for each hired mode, ordered pair of regions with a route and commodity, a lane with loss 0 and labour `ceil(10 000 * f * s * Duration / Output)`, `f` the binding fill of one unit of the commodity, from the cheapest recipe making the haul commodity, with exact integer arithmetic. |
| FR-TRN-042 | The module MUST log nothing above `Debug` per shipment, MUST have no `Info` line in the tick, and the keys `shipment`, `mode`, `dest` and `modes` MUST be used as in 13.3. |
| FR-TRN-043 | Hired capacity MUST be an ordinary commodity made by ordinary recipes and traded by ordinary orders: no market, admission or settlement rule of `spec-clearing.md` MUST depend on a mode, and a world with the module on MUST pass the clearing invariants unchanged. |
| FR-TRN-044 | Two requests with the same arguments from the same state MUST give the same result, shipment ID, Transit account and events: no iteration order of a map MUST influence a result. |
| FR-TRN-045 | A request MUST be refused `vehicle_in_cargo` if a line names the mode's `Vehicle` commodity, `duplicate_line`, `unknown_commodity` or `invalid_quantity` for a bad line, `empty_shipment` for a hired or consigned request with no line, and `too_many_lines` above `MaxShipmentLines`. |
| FR-TRN-046 | A request MUST be refused `insufficient_stock` when for any commodity the sum of its line, its feed and its haul redemption exceeds the available (unreserved) stock. |
| FR-TRN-047 | A request whose destination is an outside node of the external module MUST be refused with `outside_destination` after `unknown_region` and before `no_route`, changing no state, and `Routes` MUST report no route to such a node (`spec-external-sector.md` FR-EXT-034). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-TRN-001 | With the default logger and observer, the travel time, the fit test, the load sum and a dispatch MUST allocate nothing beyond scratch held by the world and reused across ticks. |
| NFR-TRN-002 | The cost of arrivals in a tick SHOULD be proportional to the shipments that arrive, not to the shipments in flight. |
| NFR-TRN-003 | The module SHOULD have benchmarks of a dispatch (with 1 and with `MaxShipmentLines` lines) and of a tick of arrivals, with `b.ReportAllocs()`. |
| NFR-TRN-004 | Events of the module MUST NOT be constructed when no observer was supplied, proved by an allocation test (observability-and-logging.md 4.5, 8.2). |

## 20. Open questions

1. **Subsets of a cohort.** A cohort travels whole (3.2.4) and its capacity rounds down.
   A trader needing a few of a cohort's households needs a split, which is out of
   version 1. Revisit if a consumer needs weights above 1 for travellers.
2. **Backhaul.** The round trip is in the cost of each direction's haul units (8.4). A
   joint product with a cost share, as `spec-production.md` 3.2 allows, cannot sell
   capacity in another region's market because accounts have one region. Whether a
   carrier's round trip can be priced once is CAR's 5.1 and MER's.
3. **Does the haul recipe agree with the mode?** Capacity-distance supplied should be
   about carrying units times speed times loaded time over span (17.4). Validation does
   not check it. A report that compares them, or a recipe template, could be added if
   packs get it wrong.
4. **Item bigger than a unit.** Pooling (3.2.1) can accept a load no packing realises
   when one item exceeds the capacity of one unit (a bulky good on a `foot` unit). A
   warning `item_exceeds_unit` (commodity, mode) would catch it at validation; not
   included because it is noisy for coarse goods.
5. **Return to sender.** A diverted cargo goes to the destination estate (5.8). A
   return to the shipper would need the shipper to be reachable and a second journey.
   MER may want it for contracts.
6. **Attribution of internal flows.** `transit_out` and `transit_in` have two
   non-boundary sides; `spec-ledger.md` 7.4 attributes a flow to "the region of the
   non-boundary side". The ledger should state which side an internal flow is attributed
   to (the debited account's region is suggested). Unverified; asked of LED.
7. **Does the default strategy count `Inbound`?** The field is provided (9.1). Whether
   `DefaultStrategy`'s `Target` of `spec-agents.md` 7.3 includes units arriving within
   `StartsIn` is AGT's; the default never ships but a consignee may be sent goods by
   others.
8. **The lanes ignore the carriers' inputs.** `transport.Lanes` charges labour only
   (9.4), so the steady-state check does not count the feed and vehicle wear of the
   carriers' recipe, and a pack could pass the check yet be short of fodder. VIA could
   take a lane with inputs.
9. **A window for travellers.** `spec-welfare-and-wages.md` 2.3.3 derives the window
   from recipes only. A traveller's trip is a gap in income (TRN-A6); either the window
   includes the longest self-carried trip of any role (it would lengthen every role's
   window) or a role says whether it travels. The warning is a stopgap.
10. **Tolls and tariffs** at an edge (FIS) would be a money cost at dispatch, charged to
    the shipper; the place is step 3 of 5.5.1. Not specified.
11. **Mid-journey events.** A closed road after dispatch does not affect a shipment in
    flight (4.3, 6.3). Whether a shock should delay or divert cargo in flight is
    INT's and RSK's.
12. **Several vehicles of unequal wear in one trip.** `WearItems` wears the `n` most worn
    distinct vehicles (`spec-ledger.md` 5.4.3), so the worst vehicle is used first and may
    break at dispatch while a new one is spared. Whether the least worn should travel is
    unmodelled.

## 21. Where this lives in the code

| Concern | Home |
|---------|------|
| `Mode`, `ModeID`, `Shipment`, `ShipmentID`, `ShipmentRequest`, `Line`, the module's state and the world's phase steps (arrivals, dispatch) | root package, `transport.go` |
| Travel time, units needed, the fit test, the lanes arithmetic | `internal/transit` over primitives (architecture.md 2.9): `Ticks(distance, speed, factorBP)`, `UnitsNeeded(mass, bulk, unitMass, unitBulk)`, `Fits(load, capacity)`, `Lane(...)` |
| Transit accounts, `Move`, `WearItems`, ageing, the audit | `internal/ledger` (`spec-ledger.md` 16); the root opens and reuses Transit accounts |
| The events and their fields | root package, `event.go` (`spec-observability.md`) |
| The fields of section 3.1 and their bounds | root package, `transport.go`; bounds in `bounds.go` (`spec-config.md` 4.1) |
| Validation and the warnings of section 12 | root package, `validate.go` |
| `transport.Lanes` | root package, `transport.go` (a pure function; the solver of `spec-viability.md` takes its result as an option) |
| The plan field `Shipments` and the view groups `Inbound`, `Routes` | root package, `strategy.go` (`spec-agents.md` 6.3, 6.4) |
| Reason constants (`unknown_mode`, `capacity_exceeded` and the rest of 5.4) and log keys | `internal/logkeys` |
