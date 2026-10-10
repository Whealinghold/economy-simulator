# Observability and Logging Rules

This document is normative. It binds every contributor, human or agent, who
writes or changes code in this module. Its purpose is to make sure that every
change considers observability and logging, so that a person running a
simulation built on this library can see what it is doing and why.

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT** and **MAY** are
used as in RFC 2119. To deviate from a SHOULD, write the reason in the PR
description.

This document extends section 11 and rule 12.5 of
[architecture.md](architecture.md). If the two conflict, `architecture.md` wins
and this document is corrected in the same PR. For tests of observability
behaviour, this document wins over [testing.md](testing.md).

Rules that are project decisions rather than restatements of a cited source are
marked **[project]**. Section 11 lists the sources.

## 1. Principles

1.1. Observability is part of the definition of done. A change that adds a
failure path, a state transition, a decision point or a loop is not complete
until it has considered logs, events and metrics (see section 9). **[project]**

1.2. The library decides *what* is observable. The client decides *where it
goes*. The library never chooses a sink, never writes to stdout or stderr, and
never reads global logging or telemetry state (architecture.md 11.1, 11.2).

1.3. Observability is optional and free when unused. With no logger and no
observer supplied, every call is a no-op and the hot paths allocate nothing
extra. **[project]**

1.4. Observability never changes results. A run with a logger and observer
attached **MUST** produce exactly the same output as a run without them
(architecture.md 11.3). Logging and observer code **MUST NOT** draw from the
random source, mutate world state or depend on wall-clock time to make
simulation decisions. **[project]**

## 2. Which channel to use

The library has two output channels. Choose deliberately. **[project]**

| Need                                         | Channel                                  |
|----------------------------------------------|------------------------------------------|
| Something a human should read when debugging | log, via the injected `*slog.Logger`     |
| An anomaly or degraded behaviour             | log at `Warn`                            |
| A fact that happened in the simulation, to be counted, graphed or traced | event, via `Observer` |
| A value for the caller to act on             | a returned value or error, never a log   |
| The state of the world at a point in time    | `Snapshot`                               |

2.1. If you are about to log something that someone will want to count or
graph, emit an event instead (or as well). Do not make operators parse log
lines to build a metric.

2.2. If you are about to log an error that you also return, stop. Handle each
error once (architecture.md 8.3). The caller decides whether to log it.

2.3. Anything a caller needs in order to behave correctly belongs in the return
value or error, not in a log line.

## 3. Logging

### 3.1 Mechanics

3.1.1. The only logging API is `*slog.Logger`, injected with `WithLogger`. The
default is `slog.New(slog.DiscardHandler)`. A nil logger from the caller means
the default (architecture.md 6.2, 11.1, 11.2).

3.1.2. Prefer `logger.LogAttrs(ctx, level, msg, attrs...)` with typed
`slog.Attr` values (`slog.Int`, `slog.String`, ...) over the loosely typed
key/value form. `LogAttrs` avoids allocation overhead, and typed attributes
prevent key/value mismatches.

3.1.3. The arguments to a log call are evaluated even if the record is
discarded. In any loop that runs per tick, per market, per agent or per order,
**MUST** check `logger.Enabled(ctx, level)` first when building the attributes
costs anything, or pass a `slog.LogValuer` that defers the work.

3.1.4. Scope a logger once with `logger.With(...)` or `WithGroup` when a
component logs repeatedly (for example `With("component", "market")`). Do not
repeat common attributes on every call. Use `WithGroup` when two subsystems
could otherwise produce the same key.

3.1.5. Context: pass the caller's `context.Context` to `LogAttrs` when one is
in scope. Where none is (short CPU-bound calls such as `Step`), use
`context.Background()`; never store a context in a struct (architecture.md
12.3).

### 3.2 Levels

`slog` defines `Debug` (-4), `Info` (0), `Warn` (4) and `Error` (8), and higher
means more severe. The meanings below are specific to this library.
**[project]**

| Level   | Use it for | Volume |
|---------|------------|--------|
| `Error` | An internal inconsistency the library detected and repaired, where continuing is safe and the method has no error result. If continuing could corrupt state, panic instead (architecture.md 8.1). If you can return an error, return it instead. | Should never occur in normal operation. A single occurrence is a bug report. |
| `Warn`  | The simulation is continuing but a market, a role or the configuration is degraded or surprising: a price clamped to a bound, a role with no agents left and no replacement, a configuration value that was defaulted. A problem with a single order, agent or trade is not a `Warn`: it is an event (such as `OrderRejected`) and, if useful, a `Debug` line. | Rare. At most one line per market or role per tick, so the volume is bounded by the configuration. |
| `Info`  | Lifecycle that an operator wants in a normal run: world created (with a configuration summary and seed), run started (ticks requested), run ended (ticks completed, whether it was cancelled). The library never measures wall-clock duration (architecture.md 7.3); the caller or the `otelobserver` adapter does. | A handful per run. Never per tick, per agent or per trade. |
| `Debug` | Detail for diagnosing behaviour: per-tick summaries, per-market clearing results, rejected orders, strategy decisions, bankruptcy and spawn decisions. | Allowed to be high; **MUST** be guarded by `Enabled` in hot paths. |

3.2.1. Choose the level by the audience and the action it implies, not by how
interesting the line seems. Ask: "should an operator be woken, told, or only
shown this on request?"

3.2.2. `Info` **MUST NOT** be emitted inside the per-tick, per-agent or
per-order loops. `Warn` **MAY** be emitted inside the per-tick loop only for a
condition about a market, a role or the configuration, so that its volume per
tick is bounded by the configuration; it **MUST NOT** be emitted per agent, per
order or per trade. `Error` has no volume rule because it should never fire.
Put per-entity detail at `Debug` and, when it is a fact, in an event.

3.2.3. Do not introduce custom levels. If `Debug` is not enough, the data
belongs in an event.

3.2.4. A `Warn` or `Error` **MUST** be actionable: the message and attributes
say what happened, where, and with which values, so the reader does not have to
reproduce the run to understand it.

### 3.3 Messages and attributes

3.3.1. The message is a short, constant, lowercase phrase that describes what
happened ("order rejected", "agent bankrupt", "world created"). It **MUST NOT**
interpolate values; values go in attributes. Constant messages can be searched,
grouped and counted.

3.3.2. Attributes use stable, lowercase keys in `snake_case`, defined once as
constants in an internal package and reused. The standard keys are:

| Key         | Meaning                              |
|-------------|--------------------------------------|
| `tick`      | simulation tick (integer)            |
| `ticks`     | ticks requested or completed by a `Run` |
| `seed`      | random seed                          |
| `commodity` | commodity ID                         |
| `role`      | role ID                              |
| `recipe`    | recipe ID                            |
| `agent`     | agent ID                             |
| `price`     | a price, in minor units              |
| `quantity`  | a quantity, in whole units           |
| `side`      | `bid` or `ask`                       |
| `reason`    | machine-readable reason (see below)  |
| `error`     | an error value (`slog.Any`)          |
| `region`    | region ID                            |
| `bound`     | which bound a price belief sits on: `floor` or `ceiling` |
| `agents`    | a count of agents (a summary, or the number at a bound) |
| `households`| a count of households, the sum of agent weights |
| `unfed`     | a count of households that went unfed |
| `path`      | the path of a configuration field, as in a validation error |
| `rule`      | the name of a configuration validation rule |
| `config_hash` | the first 8 bytes of the configuration hash |
| `prices`, `wages`, `groups`, `jobs` | counts of rows of the warm-start table, in the construction summary |
| `cause`      | machine-readable cause of a removal, a weight change or a role change (a constant, like `reason`) |
| `want`       | want ID |
| `commodities`, `recipes`, `roles`, `regions` | counts of the entities of each kind, in the construction summary (`region`, an ID, is a different key) |
| `modules`    | the names of the modules that are on, in the construction summary |
| `warm_start` | whether a warm start was present, in the construction summary (a bool) |
| `edges`, `components` | counts of the directed edges and of the strongly connected components of the region graph, in the construction summary |
| `from_region`, `to_region` | region IDs of the two ends of a relocation or an edge (`region` stays the key for a single region) |
| `distance` | an edge length, `int64`, in the pack's distance unit |
| `closed` | whether an edge is closed (a bool) |
| `flow` | money flow ID |
| `due`, `applied` | money, in minor units: the amount a money flow computed for a recipient or region, and the amount actually applied |
| `births`, `deaths` | counts of households born and of households that died |
| `shipment` | shipment ID |
| `mode` | transport mode ID |
| `dest` | destination region ID of a shipment (`region` is the origin) |
| `modes` | a count of transport modes, in the construction summary |
| `resource` | resource ID |
| `stock` | units in an endowment (`int64`) |
| `short` | units asked of an endowment and not granted (`int64`) |
| `rumours` | a count of rumours offered to one agent |
| `lag` | a lag in ticks, `int64` |
| `node` | region ID of an outside node (`region` stays the key for the gate region) |
| `value` | money in minor units (`int64`), as the money paid or received in the trades of an outside side |
| `nodes`, `counterparties` | counts of outside nodes and of counterparties, in the construction summary |
| `drift` | the drift of an outside price walk, in basis points (`int64`) |
| `own_use` | households in the own-use mode of the subsistence module, the sum of agent weights (`int64`) |
| `contract` | delivery contract ID |
| `offers`, `takes` | counts of admitted delivery offers and delivery takes of a round |
| `merchants` | a count of agents with a merchant block, in the construction summary (the module name `merchants` also joins the `modules` list) |

Add a key to this table in the same PR that introduces it. Prices, money and
quantities are logged as the integers they are (architecture.md 3.5), with
`slog.Int64`; never convert them to floating point for a log line. These keys
are for logs; metric attribute names are namespaced (5.1.1).

3.3.3. When logging a decision or rejection, include a `reason` attribute drawn
from a small fixed set of constants so it can be filtered, rather than a
free-form string.

3.3.4. Log an error under the key `error` with `slog.Any("error", err)`, not by
formatting it into the message.

3.3.5. Do not log whole configurations, snapshots or order books. Log
identifiers and summaries. Do not log values that could contain caller secrets;
the library should never be handed any.

## 4. Events

Events are how the simulation reports facts for metrics and traces. They reach
the client through the `Observer` interface (architecture.md 4.3).

4.1. An event is a plain data struct defined in the root package that
implements the sealed `Event` interface (architecture.md 4.8). It has an
exported `Tick` field and the identifiers and values describing the fact. It
**MUST NOT** contain pointers into live world state, functions, or wall-clock
timestamps (architecture.md 12.4). Slices in an event are copies, in a
deterministic order. Events **MUST NOT** contain maps: any consumer that
iterates one, including the golden-file renderer, gets a random order. Money,
prices and quantities use the fixed-point types of architecture.md 3.5. Event
types are named in the past tense for what happened (`TradeExecuted`,
`OrderRejected`), and do not embed other types (architecture.md 4.7).
**[project]**

4.2. Adding fields to an event is allowed. Removing or renaming a field, or
changing its meaning, is a breaking change (architecture.md 6.6). **[project]**

4.3. The library **MUST** emit at least the following events. Adding a new
state transition to the library without an event for it needs a written reason
in the PR. **[project]**

| Event type                | Emitted when                                         |
|---------------------------|------------------------------------------------------|
| `WorldCreated`            | the world is constructed and validated (seed, the whole configuration hash, whether a warm start was present) |
| `RunStarted`              | `Run` begins (ticks requested)                       |
| `RunEnded`                | `Run` returns, after its last completed tick (ticks completed, whether it was cancelled) |
| `TickStarted`, `TickEnded`| each `Step` begins and ends                          |
| `OrderPlaced`             | an agent submits a bid or ask (region, commodity, agent, side, limit price, quantity) |
| `OrderRejected`           | an order fails validation or cannot be accepted (region, commodity, agent, side, `reason`) |
| `MarketCleared`           | a market finishes matching, once per market per tick, also when nothing traded (price, price age, volume, unmatched bid and ask units, best unmatched bid and ask) |
| `TradeExecuted`           | a bid and an ask are matched                         |
| `RecipeExecuted`          | a job completes, and a one-tick job completes in its start tick (agent, recipe, start tick, executions, outputs, disposed quantities, effective yield) |
| `RecipeAbandoned`         | a job ends without output (agent, recipe, executions, `Reason`) |
| `AgentBankrupted`         | an agent is removed entirely; the only event of a full removal, whatever the cause (weight at removal, `Cause`: `insolvent`, `starved`, `consumer` or `population`; money and `GoodsListed`, the number of goods passed to the estate) |
| `AgentSpawned`            | an agent enters the world, including each initial agent during construction, after `WorldCreated` (region, `Role`, `Weight`, `StartMoney` received, `External`) |
| `UpkeepShortfall`         | once per region and role per tick in which at least one household went unfed, after the consumption phase; never when all are fed (households, unfed households, agents unfed, largest hunger) |
| `BeliefsUpdated`          | once per market per tick that had at least one order outcome, after the outcomes are processed (counts of outcomes, adjustments, raises, lowerings, beliefs at floor and ceiling, conflicts and clamps; mean, low and high centre) |
| `BoundaryFlow`            | once per tick for each (region, reason, money or commodity) with a non-zero total, after the tick's transitions (reason, faucet or sink, total quantity) |
| `EstateOpened`            | an agent is removed and its estate is opened, accompanying the removal event (agent, region, balance, number of goods listed) |
| `ReliefPaid`              | once per region per tick in which any relief is paid (total paid, number of recipients) |
| `TraineeshipStarted`, `TraineeshipCompleted`, `TraineeshipEnded` | a traineeship starts, completes, or ends without completing (identifiers only, and `reason` for the end) |
| `WeightChanged`           | an agent's weight changes without removing it: a reduction by the failure ladder, households added to an existing agent, a birth or death of the population module (agent, region, role, `From`, `To`, `Cause`) |
| `RoleChanged`             | an agent changes role (agent, region, `From`, `To`, weight, `Cause`, `GoodsListed`) |
| `EntryLimited`            | once per region per tick in which a limit or refusal of the spawner acted; never when all counts are zero (counts of role changes, entrants, removals, unfunded households, households with no viable role, and each cap, cooldown, hysteresis, exit-cap and refusal that acted) |
| `WantShortfall`           | once per region, role and want per tick in which a want was choked or unfunded for at least one household, after the trade phase (households, choked households, planned and funded units) |
| `AgentRelocated`          | an agent's account changes region, once per relocation, after it is complete (agent, role, `From`, `To`, weight, money, units of goods, `Cause`) |
| `EdgeStatusChanged`       | an edge's status changed, once per changed edge in edge order after the new graph is in place (`From`, `To`, `Closed`, `FactorBP`, `Cause`); only when something sets a status |
| `MoneyFlowLimited`        | once per money flow and region per tick in which any amount due was not applied; never when all was applied (flow, region, due, applied, and the households short, cut at the balance bound and cut by the limit) |
| `MoneyFlowEnded`          | a money flow reaches its effective limit and ends, once (flow, `Cause`: `limit` or `cap`, total) |
| `MoneyFlowScaled`         | a money flow's scale is changed between steps, once per change (flow, from, to) |
| `PopulationChanged`       | once per region and role per tick in which the population module is on and any count other than the exposed, unfed and no-ratio households is non-zero, after the region's plan and after `EntryLimited`; never when all are zero (region, role, exposed, unfed and no-ratio households, births and deaths due, deaths cut, deaths held, births capped, births unfunded, rates clamped, births and deaths done) |
| `ShipmentDispatched`      | a shipment is dispatched (shipment, mode, origin region, destination, shipper, traveller and whether hired, consignee, arrival tick, carrying units, haul units redeemed, lines) |
| `ShipmentArrived`         | a shipment arrives, after its delivery (shipment, mode, origin, destination, receiver, whether diverted, and per line the units shipped, delivered, diverted and spoiled) |
| `ShipmentRejected`        | a shipment request is refused (origin region, agent, mode, destination, `reason`) |
| `EndowmentChanged`        | once per endowment (region, resource) per tick, after its regeneration at the end of phase 8, in (`RegionID`, resource) order, if any of the units drawn, regenerated, short or lost is non-zero; never otherwise (stock after the tick, effective capacity, abundance of the tick, units requested, granted, drawn and short, regenerated, lost, agents denied, agents reduced) |
| `EndowmentAdjusted`       | a shock slot or an adjustment of an endowment is applied between ticks, once per call (kind `adjust` or `shock`, signed delta, the scales after, stock before and after, units lost) |
| `RumoursServed`           | once per reader region per tick in which the information module is on and at least one informed agent was given a view there, in ascending `RegionID`, after the last plan of phase 3 and before phase 4 (region, readers, and counts of candidates offered, with no path, too old and not yet, and the largest lag offered) |
| `OutsideTraded`           | once per (node, gate region, commodity, side) per tick in which the side wanted any unit or any unit was cut, after the sweep of phase 4; never when nothing was wanted or cut (node, gate region, commodity, side, mid, limit price, wanted, offered and traded units, money paid or received, list value, units cut by a closed edge, by a cap and by a price below 1, whether a price clamp acted) |
| `OutsideEnded`            | a side of an outside node reaches its cumulative cap and ends, once (node, commodity, side, `Cause`: `units` or `value`, cumulative units and list value) |
| `OutsideShocked`          | the price shock of an outside line is set between steps, once per call (node, commodity, from, to) |
| `OwnUseChanged`           | once per region and role per tick in phase 2, after the starts, in ascending `RegionID` then role order, if any of the households entered, left, started, blocked or retried, or the units output, is non-zero; never otherwise (households of the role, households in the own-use mode, entered, left, executions of own-use recipes started, blocked, retried, and the units created at the completion of own-use jobs, by commodity) |
| `MigrationStarted`        | a migration journey begins, once per departure, in the order of admission, after the departure is complete (agent, role, `From`, `To`, weight, arrival tick, distance, journey time, earned ratio, home figure, estimate, whether a job was abandoned) |
| `MigrationReviewed`       | once per region per tick in which the migration module is on and any household was pushed or held back by the settling rule, any arrival was held or any journey ended by a removal, in ascending `RegionID` after the `MigrationStarted` events of the pass; never when all are zero (households unsettled, reviewed, pushed, and the eleven outcomes started, no estimate, no destination, no news, too far, not viable, no gain, bound, out cap, unprovisioned, in cap, and arrivals held and journeys ended) |
| `ContractFormed`          | a delivery contract is formed, after its shipment's `ShipmentDispatched` (contract, payer, deliverer, origin region, destination, commodity, quantity, price, money escrowed, shipment, mode, arrival tick) |
| `ContractSettled`         | a delivery contract ends: in phase 1 after the arrivals of transport, or at the removal of its deliverer (contract, payer, deliverer, destination, commodity, quantity, units reached and spoiled, money paid and refunded, `Cause`: `delivered`, `short`, `payer_removed` or `deliverer_removed`) |
| `ContractRound`           | once per destination region per tick in which an offer or a take was submitted for it, in ascending `RegionID`, after the round (admitted and refused offers and takes, matches, contracts formed, failures by reason, units and value formed); never otherwise |
| `MerchantsPlanned`        | once per region per tick in which an agent with a merchant block was given a view there, after `ContractRound` and before `RumoursServed` (merchants, counts of the lanes considered by result, trips planned, requests, offers and takes, rests, stays and lots cut by the cap) |

The parentheses name the main fields; the fields of each event, and the order
in which the events of one step are emitted, are fixed by the spec that owns
the transition (`doc/specs/`: production for the recipe events, agents for
`AgentSpawned`, the removal event and `UpkeepShortfall`, beliefs, clearing,
ledger, estates, skill barriers and configuration for the rest). Every field
obeys 4.1: no map, no pointer, fixed-point values. `BoundaryFlow`,
`BeliefsUpdated`, `UpkeepShortfall`, `EntryLimited`, `WantShortfall`, `MoneyFlowLimited`, `PopulationChanged`, `EndowmentChanged`, `RumoursServed`, `OutsideTraded`, `OwnUseChanged`, `MigrationReviewed`, `ContractRound` and `MerchantsPlanned` are aggregates: each is bounded by
regions, commodities, reasons, roles, wants, money flows, resources or outside nodes in the configuration, so none is per
agent, order or trade. `WeightChanged` and `RoleChanged` are per agent, like `AgentSpawned` and `AgentBankrupted`: each is a state transition of one agent. `AgentRelocated` and `MigrationStarted` are per agent too, and `ContractFormed` and `ContractSettled` are per contract, like `ShipmentDispatched`. `ShipmentDispatched`, `ShipmentArrived` and `ShipmentRejected` are per shipment or request, like `OrderPlaced` and `OrderRejected`: their `Lines` are copies in commodity order, bounded by `MaxShipmentLines`. `EdgeStatusChanged`, `MoneyFlowEnded`, `MoneyFlowScaled`, `EndowmentAdjusted`, `OutsideEnded` and `OutsideShocked` are per edge, flow, call or side. An event of a module that is switched off is not emitted: modules are off unless configured (ADR 0001), so `ReliefPaid`, `Traineeship*`, `WantShortfall`, `MoneyFlowLimited`, `MoneyFlowEnded`, `MoneyFlowScaled`, `PopulationChanged`, `ShipmentDispatched`, `ShipmentArrived`, `ShipmentRejected`, `EndowmentChanged`, `EndowmentAdjusted`, `RumoursServed`, `OutsideTraded`, `OutsideEnded`, `OutsideShocked`, `OwnUseChanged` (subsistence), `MigrationStarted` and `MigrationReviewed` (migration), `ContractFormed`, `ContractSettled` and `ContractRound` (merchants) and the events of any later module appear only when it is on, and `MerchantsPlanned` only when a role has the merchant block; `AgentRelocated` and `EdgeStatusChanged` depend on no module. The order of the events of a step follows the phases: arrivals of shipments in shipment order in phase 1, after them the `ContractSettled` events of the contracts whose shipment arrived in contract ID order and the migration arrivals (`AgentRelocated` with the cause `migration`) in ascending agent ID, in the order the tick positions of WLD fix (N-009); in phase 2 `OwnUseChanged` after the starts; in phase 3 each agent's shipment events in plan order before its `OrderPlaced` events, then, after the last plan, the `ShipmentDispatched` and `ContractFormed` events of each contract of the round in formation order, then `ContractRound` and `MerchantsPlanned`; in phase 7 `MigrationStarted` in the order of admission and then `MigrationReviewed`, after `EntryLimited` and `PopulationChanged` of the last region, and the `ContractSettled` of a deliverer's removal with the removal events. `RumoursServed` comes in phase 3 after the last plan, after `MerchantsPlanned` when the merchant module is on, and before phase 4, in ascending `RegionID`; the outside's events (`OutsideTraded`, `OutsideEnded`) come after the sweep at the end of phase 4; `EndowmentChanged` comes at the end of phase 8 in (`RegionID`, resource) order.

Because observers are supplied as options to `NewWorld`, they receive
`WorldCreated` and the initial `AgentSpawned` events. A consumer can rebuild
the total live agent count from `AgentSpawned` and `AgentBankrupted` alone. The
household count also needs `WeightChanged` (`To` less `From`) besides the `Weight` those two carry, and a count by role also needs `RoleChanged` (the agent and its weight leave `From` and join `To`). A count of households by region also needs `AgentRelocated` (the weight leaves `From` and joins `To`). Estate pseudo-agents are not
agents and are not counted in any of them.

4.4. Observers are called synchronously on the stepping goroutine and **MUST
NOT** call back into the `World` (architecture.md 12.5). The library **MUST
NOT** hold internal locks or leave state half-updated while calling an observer:
emit an event only after the transition it describes is complete. Observers
**MUST** ignore event types they do not recognise.

4.5. When the observer is the no-op default, constructing events in hot paths
**MUST NOT** allocate. Passing an event struct as an `Event` boxes it, so the
world records once, at construction, whether any observer was supplied, and
constructs and emits events only when it was. Prove this with an allocation
test (8.2). **[project]**

## 5. Metrics

Metrics are produced from events by the separate `otelobserver` module. The
core module defines events; it does not define or record metrics, and it does
not import OpenTelemetry (architecture.md 11.4). The rules in this section
apply to `otelobserver` and to anyone adding events that metrics will be built
from.

### 5.1 Naming

Names follow the
[OpenTelemetry naming conventions](https://opentelemetry.io/docs/specs/semconv/general/naming/).

5.1.1. Metric names and metric attribute names are lowercase, namespaced with
dots, and use `snake_case` within a component:
`economysim.market.clearing_price`, `economysim.commodity`. Log keys (3.3.2)
are not namespaced; the adapter maps `commodity` to `economysim.commodity` and
so on.

5.1.2. All metric names and metric attribute names start with the
`economysim.` namespace. The `otel.*` namespace is reserved for OpenTelemetry
itself and **MUST NOT** be used.

5.1.3. Do not append `_total` to counters, and do not put the unit in the name.
Do not pluralise a name unless the value counts discrete instances of a
quantity (`economysim.market.trades` with unit `{trade}` is correct).

5.1.4. Durations are named `{operation}.duration` with unit `s`.

5.1.5. Every instrument **MUST** have a unit and a one-line description.

### 5.2 Choosing an instrument

Pick the instrument by the nature of the value, following the
[OpenTelemetry Metrics API](https://opentelemetry.io/docs/specs/otel/metrics/api/):

| Value                                   | Instrument                      |
|-----------------------------------------|---------------------------------|
| only ever increases (trades, bankruptcies) | Counter                      |
| goes up and down (live agent count)     | UpDownCounter                   |
| a distribution (step duration, trade price spread) | Histogram            |
| a point-in-time value that is not additive (last clearing price) | Gauge  |

### 5.3 Initial catalogue

These are the initial instruments. Changing a name, unit or attribute set
breaks users' dashboards and alerts and is a breaking change. Every instrument
except the duration histogram records integers, so use the `Int64` variants;
prices are recorded in minor units, unconverted (architecture.md 3.5).
**[project]**

| Name                                | Instrument      | Unit            | Attributes        |
|-------------------------------------|-----------------|-----------------|-------------------|
| `economysim.world.step.duration`    | Histogram       | `s`             | none              |
| `economysim.market.trades`          | Counter         | `{trade}`       | `economysim.commodity` |
| `economysim.market.volume`          | Counter         | `{unit}`        | `economysim.commodity` |
| `economysim.market.clearing_price`  | Gauge           | `{minor_unit}`  | `economysim.commodity` |
| `economysim.market.orders.unmatched`| Counter         | `{order}`       | `economysim.commodity`, `economysim.side` |
| `economysim.recipe.executions`      | Counter         | `{execution}`   | `economysim.recipe` |
| `economysim.agent.count`            | UpDownCounter   | `{agent}`       | `economysim.role` |
| `economysim.agent.bankruptcies`     | Counter         | `{agent}`       | `economysim.role` |
| `economysim.household.count`        | UpDownCounter   | `{household}`   | `economysim.role` |
| `economysim.household.unfed`        | UpDownCounter   | `{household}`   | `economysim.role` |
| `economysim.role.changes`           | Counter         | `{change}`      | `economysim.role` (the destination role) |
| `economysim.want.unfunded`          | Counter         | `{unit}`        | `economysim.want`, `economysim.role` |
| `economysim.household.relocations`  | Counter         | `{household}`   | `economysim.region.from`, `economysim.region.to` |
| `economysim.money.flow`             | Counter         | `{minor_unit}`  | `economysim.reason`, `economysim.kind` (`faucet` or `sink`) |
| `economysim.money.flow.unapplied`   | Counter         | `{minor_unit}`  | `economysim.reason` |
| `economysim.household.births`       | Counter         | `{household}`   | `economysim.role` |
| `economysim.household.deaths`       | Counter         | `{household}`   | `economysim.role` |
| `economysim.transport.shipments`    | Counter         | `{shipment}`    | `economysim.mode` |
| `economysim.transport.in_transit`   | UpDownCounter   | `{unit}`        | `economysim.commodity` |
| `economysim.transport.cargo.spoiled`| Counter         | `{unit}`        | `economysim.commodity` |
| `economysim.endowment.drawn`        | Counter         | `{unit}`        | `economysim.resource`, `economysim.region` |
| `economysim.endowment.regenerated`  | Counter         | `{unit}`        | `economysim.resource`, `economysim.region` |
| `economysim.endowment.short`        | Counter         | `{unit}`        | `economysim.resource`, `economysim.region` |
| `economysim.endowment.stock`        | Gauge           | `{unit}`        | `economysim.resource`, `economysim.region` |
| `economysim.information.rumours`    | Counter         | `{rumour}`      | `economysim.outcome` |
| `economysim.external.units`         | Counter         | `{unit}`        | `economysim.node`, `economysim.commodity`, `economysim.side` |
| `economysim.external.value`         | Counter         | `{minor_unit}`  | `economysim.node`, `economysim.commodity`, `economysim.side` |
| `economysim.external.price`         | Gauge           | `{minor_unit}`  | `economysim.node`, `economysim.commodity`, `economysim.side` |
| `economysim.household.own_use`      | Gauge           | `{household}`   | `economysim.role`, `economysim.region` |
| `economysim.own_use.units`          | Counter         | `{unit}`        | `economysim.commodity` |
| `economysim.migration.started`      | Counter         | `{household}`   | `economysim.region.from`, `economysim.region.to` |
| `economysim.migration.limited`      | Counter         | `{household}`   | `economysim.limit` |
| `economysim.migration.on_journey`   | UpDownCounter   | `{household}`   | none |
| `economysim.contract.formed`        | Counter         | `{contract}`    | `economysim.commodity` |
| `economysim.contract.value`         | Counter         | `{minor_unit}`  | `economysim.commodity` |
| `economysim.contract.escrow`        | UpDownCounter   | `{minor_unit}`  | none |
| `economysim.merchant.decisions`     | Counter         | `{lane}`        | `economysim.decision` |

### 5.3.1 Which event feeds which metric

| Metric                               | Fed by                                  |
|--------------------------------------|-----------------------------------------|
| `economysim.world.step.duration`     | `TickStarted` to `TickEnded`, measured by the adapter |
| `economysim.market.trades`           | `TradeExecuted`                         |
| `economysim.market.volume`           | `TradeExecuted` (quantity)              |
| `economysim.market.clearing_price`   | `MarketCleared`                         |
| `economysim.market.orders.unmatched` | `MarketCleared`                         |
| `economysim.recipe.executions`       | `RecipeExecuted` (its `Executions` field, not one per event) |
| `economysim.agent.count`             | `AgentSpawned` (+1, including initial agents), `AgentBankrupted` (-1); by role also `RoleChanged` (-1 for `From`, +1 for `To`) |
| `economysim.agent.bankruptcies`      | `AgentBankrupted` (every `Cause`)       |
| `economysim.household.count`         | `AgentSpawned` (+`Weight`, including initial agents), `AgentBankrupted` (-`Weight`), `WeightChanged` (`To` less `From`); by role also `RoleChanged` (-`Weight` for `From`, +`Weight` for `To`) |
| `economysim.household.unfed`         | `UpkeepShortfall` (`Unfed`, by role; see below) |
| `economysim.role.changes`            | `RoleChanged` (by `To`)                 |
| `economysim.want.unfunded`           | `WantShortfall` (`Planned` less `Funded`, by want and role) |
| `economysim.household.relocations`   | `AgentRelocated` (`Weight`, by `From` and `To`) |
| `economysim.money.flow`              | `BoundaryFlow` (money, by the reason and whether it is a faucet or a sink) |
| `economysim.money.flow.unapplied`    | `MoneyFlowLimited` (`Due` less `Applied`, by flow) |
| `economysim.household.births`        | `PopulationChanged` (`BirthsDone`, by role) |
| `economysim.household.deaths`        | `PopulationChanged` (`DeathsDone`, by role) |
| `economysim.transport.shipments`     | `ShipmentDispatched` |
| `economysim.transport.in_transit`    | `ShipmentDispatched` (+ each line), `ShipmentArrived` (- each line's units shipped) |
| `economysim.transport.cargo.spoiled` | `ShipmentArrived` (units spoiled, by line) |
| `economysim.endowment.drawn` | `EndowmentChanged` (units drawn) |
| `economysim.endowment.regenerated` | `EndowmentChanged` (units regenerated) |
| `economysim.endowment.short` | `EndowmentChanged` (units short) |
| `economysim.endowment.stock` | `EndowmentChanged` (stock after the tick) and `EndowmentAdjusted` (stock after); the adapter keeps the last value for a tick with no event |
| `economysim.information.rumours` | `RumoursServed` (`Offered`, `NoPath`, `TooOld`, `NotYet`, by outcome) |
| `economysim.external.units` | `OutsideTraded` (units traded) |
| `economysim.external.value` | `OutsideTraded` (money paid or received) |
| `economysim.external.price` | `OutsideTraded` (the limit price) |
| `economysim.household.own_use` | `OwnUseChanged` (households in the mode, by role and region); the adapter keeps the last value for a tick with no event |
| `economysim.own_use.units` | `OwnUseChanged` (units output, per line) |
| `economysim.migration.started` | `MigrationStarted` (`Weight`, by `From` and `To`) |
| `economysim.migration.limited` | `MigrationReviewed` (the counts other than started, pushed, reviewed and unsettled, by outcome: `no_estimate`, `no_destination`, `no_rumour`, `too_far`, `no_viable`, `no_gain`, `bound`, `out_cap`, `unprovisioned`, `in_cap`, `held`, `ended`) |
| `economysim.migration.on_journey` | `MigrationStarted` (+`Weight`), `AgentRelocated` with the cause `migration` (-`Weight`) and `MigrationReviewed` (- `Ended`) |
| `economysim.contract.formed` | `ContractFormed` |
| `economysim.contract.value` | `ContractFormed` (money escrowed) |
| `economysim.contract.escrow` | `ContractFormed` (+ money escrowed), `ContractSettled` (- money paid and refunded) |
| `economysim.merchant.decisions` | `MerchantsPlanned` (the case counts, by decision) |

`UpkeepShortfall` is not emitted for a region and role whose households are all
fed, so the adapter treats a tick with no such event for a region and role as
zero unfed there. `economysim.household.count` and `economysim.household.unfed`
count households (the sum of weights), never agents. `economysim.role.changes` counts changes, not agents, and `economysim.want.unfunded` counts units of goods not funded. `economysim.household.relocations`, `economysim.household.births`, `economysim.household.deaths`, `economysim.household.own_use` and the migration instruments count households (the sum of weights), never agents.

A new metric **MUST** be derivable from the existing events or come with the
event it needs. Do not add event fields only to avoid a lookup in the adapter.
The adapter measures durations with its own clock; events carry no wall-clock
time (4.1). Because observers run synchronously (4.4), a step duration includes
the time spent in every observer during the step.

### 5.4 Cardinality

5.4.1. Metric attributes **MUST** have a number of values bounded by the
configuration: `economysim.commodity`, `economysim.role`, `economysim.recipe`,
`economysim.side`, `economysim.want`, `economysim.region.from`, `economysim.region.to` (regions), `economysim.reason` and `economysim.kind` (the money flow IDs and the built-in reasons, and faucet or sink), `economysim.mode` (transport modes), `economysim.resource` and `economysim.region` (resources and regions of the configuration), `economysim.node` (outside nodes), `economysim.outcome` (the four constants `offered`, `no_path`, `too_old`, `not_yet`), `economysim.limit` (the twelve constants of the migration outcomes: `no_estimate`, `no_destination`, `no_rumour`, `too_far`, `no_viable`, `no_gain`, `bound`, `out_cap`, `unprovisioned`, `in_cap`, `held`, `ended`) and `economysim.decision` (the eleven constants of the merchant's lane decisions: `no_rumour`, `stale`, `locked`, `perishable`, `no_margin`, `unaffordable`, `below_share`, `planned`, `rested`, `waiting`, `lot_capped`). Metric attributes **MUST NOT** include agent IDs, order IDs,
ticks, prices, quantities or any other unbounded or per-entity value. Put those
in logs, traces or events.

5.4.2. A new attribute on an existing metric multiplies its series count. Say
how many series it adds for the reference chain in the PR.

## 6. Traces

6.1. Traces are coarse. Create a span for each `Run`, from `RunStarted` to
`RunEnded`, and for each `Step`, from `TickStarted` to `TickEnded`, with
`economysim.tick` as a span attribute. A step inside a run is a child of the
run's span; a `Step` called directly is a root span. **MUST NOT** create a span
per order, trade or agent. **[project]**

6.2. Notable occurrences within a step (a bankruptcy, a rejected order) are
span events or attributes on the step span, not child spans.

6.3. The `otelobserver` module uses the `TracerProvider` and `MeterProvider`
given by the client, and **MUST NOT** call the global providers. Its
instrumentation scope name is its module path. Without an SDK installed the
OpenTelemetry API is a no-op, so the adapter is safe to wire in unconditionally.

## 7. Failure and degradation

7.1. A failure in observability **MUST NOT** break the simulation. The library
does not return errors from logging or from events, and does not stop because a
sink is slow or absent. **[project]**

7.2. A slow observer is the client's problem, because observers run
synchronously. Document this on the `Observer` type and tell implementers to
return quickly and hand work to their own goroutines if needed.

## 8. Testing observability

8.1. Behaviour that is meant to be observable **MUST** be tested.
**[project]**
- **Events**: a recording `Observer` defined in a `_test.go` file asserts that
  the expected events, with the expected fields and order, were emitted for a
  scripted scenario.
- **Logs**: a capturing `slog.Handler` asserts the level, message and key
  attributes of every `Warn` and `Error` path, and that `Info` is not emitted
  inside the tick loop.
- **Defaults**: constructing a `World` with no logger and no observer works,
  produces no output and does not panic. Passing a nil logger or nil observer
  behaves the same.
- **Non-interference**: the same seed and configuration produce identical
  results with and without a logger and observer attached.
- **Cardinality**: in the core module, a test over the reference chain asserts
  that every event field that becomes a metric attribute (commodity, role,
  recipe, side) takes only values present in the configuration. In
  `otelobserver`, a test asserts the attribute sets recorded on each
  instrument, using a configuration built in Go inside its own tests, because
  it cannot read the core module's `testdata/` (testing.md 10.2).
- **Run lifecycle**: `Run` emits `RunStarted` and `RunEnded`, including when it
  is cancelled, and `RunEnded` reports the ticks actually completed.
- **Composition**: with several `WithObserver` options, every event reaches
  each observer in the order given. An observer that ignores unknown event
  types keeps working when a new event type is added.
- **Emission order**: for a scripted step, the events are emitted in a
  deterministic order, each after the state change it describes.

8.2. The cost of the no-op path **MUST** be covered by an ordinary test, not
only a benchmark, because a benchmark never fails CI. With the default logger
and observer, the test uses `testing.AllocsPerRun` to assert that:
- each event-emission and logging call site in the tick loop allocates nothing;
  and
- a full `Step` of the reference chain allocates no more than a baseline
  constant held in the test. Raising that constant is a reviewed change with a
  reason in the PR.

A benchmark with `b.ReportAllocs()` reports the trend alongside it
(testing.md 8). **[project]**

## 9. Considerations when writing code

Before opening a PR, answer these for your change. The answers go in the PR
description under an "Observability" heading. "None, because ..." is a valid
answer; an empty or missing section is not. **[project]**

1. **New failure path or rejected input?** Is it returned as an error? If it is
   survived rather than returned, is there a `Warn` with a `reason`?
2. **New state transition?** (an order, a trade, a production run, a spawn, a
   bankruptcy, a clearing step, a run.) Does it have an event? If not, why
   not? Does the event avoid maps and use fixed-point values (4.1)?
3. **New decision point in a strategy, clearing rule or spawner?** Would an
   operator need to see why it chose what it chose? If so, `Debug` log with
   the inputs and `reason`.
4. **New loop?** Which level is anything logged inside it, and is it guarded
   with `Enabled`? Is any `Warn` in it bounded per market or role (3.2.2)?
   Does event construction allocate when the observer is the no-op (8.2)?
5. **Changed event, metric name, unit or attribute?** This is a breaking change
   for users' dashboards. Is it versioned and called out in the changelog?
6. **New metric attribute?** Is it bounded by configuration (5.4)?
7. **Used `Error` or `Info`?** Re-read the table in 3.2 and confirm the level.
8. **Logging an error you also return?** Remove the log.
9. **Tests?** Does section 8 apply, and are the tests there?

## 10. Anti-patterns checklist

Reject a change that contains any of the following:

- `fmt.Println`, `log.Print*`, `print`, or any write to stdout or stderr
  (architecture.md 11.1)
- `slog.Default()` or a global OTel provider (architecture.md 11.2)
- a custom logger interface instead of `*slog.Logger`
- a log line that also returns the same error (2.2)
- `Info` inside a per-tick, per-agent or per-order loop, or `Warn` per agent,
  order or trade (3.2.2)
- a wall-clock duration measured by the library (3.2)
- an event containing a map (4.1)
- an interpolated log message such as `"agent " + id + " went bankrupt"` (3.3.1)
- unguarded expensive attribute construction in a hot path (3.1.3)
- a metric attribute with unbounded values (5.4.1)
- a metric named with `_total`, a unit suffix, or outside the `economysim.` namespace (5.1)
- a span per order, trade or agent (6.1)
- an event holding pointers into live state or wall-clock time (4.1)
- observability code that consumes randomness or alters results (1.4)
- an `Observer` callback that calls back into the `World` (4.4)
- a PR with no Observability section (9)

## 11. Provenance

| Rule(s)                | Source |
|------------------------|--------|
| 3.1.1, 3.1.2, 3.1.3, 3.1.4, 3.2 (level ordering) | [`log/slog` package documentation](https://pkg.go.dev/log/slog): standard levels and values, `LogAttrs`, `Enabled`, `DiscardHandler`, `LogValuer`, `WithGroup` |
| 2.2                    | Uber Go Style Guide, "Handle Errors Once"; architecture.md 8.3 |
| 3.1.5                  | Go Code Review Comments, "Contexts"; architecture.md 12.3 |
| 5.1, 5.2               | [OpenTelemetry naming conventions](https://opentelemetry.io/docs/specs/semconv/general/naming/); [OpenTelemetry Metrics API](https://opentelemetry.io/docs/specs/otel/metrics/api/) |
| 8.2                    | [`testing` package documentation](https://pkg.go.dev/testing): `AllocsPerRun`; benchmarks run only with `-bench` and report rather than fail |
| 6.3                    | [OpenTelemetry library guidelines](https://opentelemetry.io/docs/specs/otel/library-guidelines/): libraries depend on the API only, and without an SDK the API is a no-op |

The slog documentation fixes the level names and numeric values and says that
higher means more severe. It does not define what each level means for a
library, so the meanings in 3.2, the standard attribute keys in 3.3.2, the event
list in 4.3, the metric catalogue in 5.3 and the span policy in 6.1 are project
decisions. The `economysim.` namespace for metric and attribute names and the
specific cardinality rules are likewise project decisions built on the
OpenTelemetry naming guidance, which asks custom names to carry an application
prefix.

## Read receipt

If you are an agent, you quote the tag below in your first message, as
required by [AGENTS.md](../../AGENTS.md). Reaching it means you have read this
file in full. Do not guess it or reuse one from memory.

When you edit this file, replace the tag with a fresh random value in the same
change (for example `od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`). A stale
tag in an agent's first message means it read an older version.

RULE-TAG: 8522b7405b6c4b7edf3d2ce0
