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
| `Warn`  | The simulation is continuing but an entity of the configuration (a market, a role, a region, an endowment, a body) or the configuration itself is degraded or surprising: a price clamped to a bound, a role with no agents left and no replacement, a configuration value that was defaulted. A problem with a single order, agent or trade is not a `Warn`: it is an event (such as `OrderRejected`) and, if useful, a `Debug` line. | Rare. At most one line per entity of the configuration per tick, so the volume is bounded by the configuration. |
| `Info`  | Lifecycle that an operator wants in a normal run: world created (with a configuration summary and seed), run started (ticks requested), run ended (ticks completed, whether it completed, was cancelled or failed), world restored (from a checkpoint), warm start applied. The library never measures wall-clock duration (architecture.md 7.3); the caller or the `otelobserver` adapter does. | A handful per run. Never per tick, per agent or per trade. |
| `Debug` | Detail for diagnosing behaviour: per-tick summaries, per-market clearing results, rejected orders, strategy decisions, bankruptcy and spawn decisions. | Allowed to be high; **MUST** be guarded by `Enabled` in hot paths. |

3.2.1. Choose the level by the audience and the action it implies, not by how
interesting the line seems. Ask: "should an operator be woken, told, or only
shown this on request?"

3.2.2. `Info` **MUST NOT** be emitted inside the per-tick, per-agent or
per-order loops. `Warn` **MAY** be emitted inside the per-tick loop only for a
condition about any entity of the configuration (a market, a role, a region, an
endowment, a body) or the configuration itself, so that its volume per tick is
bounded by the configuration; it **MUST NOT** be emitted per agent, per
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
| `reason`    | machine-readable reason (see below); the values of the "run ended" line are `completed`, `cancelled` and `failed`, the closed set of that one line (log-only values, with no event field); `not_member` is a log-only reason of the institutions module |
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
| `cause`      | machine-readable cause of a removal, a weight change, a role change, a relocation, an edge status change, the end of a money flow or of an outside side, or a slot set by the consumer or by a policy actor (a constant, like `reason`; each event has its own closed set, listed by the spec that owns the event) |
| `want`       | want ID |
| `commodities`, `recipes`, `roles`, `regions` | counts of the entities of each kind, in the construction summary (`region`, an ID, is a different key) |
| `modules`    | the names of the modules that are on, in the construction summary (the lower-case name of each `Modules` key, such as `storage` or `insurance`) |
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
| `format` | the checkpoint format version, logged as `slog.Int64(int64(f))` |
| `bytes` | the size of an encoded checkpoint, in bytes (`int64`) |
| `from`, `to` | a value before and after a change, in basis points (`int64`), as in a money flow scaled (`from_region` and `to_region` stay the keys of two regions) |
| `account`, `amount` | a ledger account ID (`int64`) and a sum of money in minor units (`int64`), in the line of a refused transfer (`agent` is the key of an agent, which an estate, escrow or transit account is not) |
| `store_class`, `store_classes` | a store class ID (string), and a count of store classes in the construction summary |
| `settlement_class`, `from_settlement_class`, `to_settlement_class`, `target_settlement_class`, `settlement_classes` | settlement class IDs (`int64`: the class, and the two ends and the target of a change), and a count in the construction summary |
| `vacancies` | a count of positions posted by employers |
| `rent`, `share` | `rent`: money per unit of an endowment, in minor units (`int64`); `share`: a share in basis points (`int64`), as the share of a title or of a strike |
| `jurisdictions`, `governed` | counts of jurisdictions and of governed jurisdictions, in the construction summary |
| `claim`, `claims`, `lenders`, `borrowers` | a claim ID, and counts of claims, lenders and borrowers |
| `rate`, `term` | a rate in basis points (`int64`) and a term in ticks (`int64`) |
| `principal`, `owed`, `arrears`, `lost` | money in minor units (`int64`): the principal of a claim (not the agent of a production job), the amount owed, the owed on overdue claims, and the amount lost |
| `facility`, `facility_class`, `facilities`, `facility_classes` | a facility ID, a facility class ID (string), and counts of facilities and of facility classes in the construction summary |
| `condition` | the wear of a facility, in units of its durability (`int64`) |
| `rows` | a count of risk rows, in the construction summary |
| `authority`, `fiscal_flows` | a jurisdiction ID (string) and a count of fiscal flows in the construction summary |
| `body`, `right`, `control`, `activity` | IDs of a body, a right and a control, and the activity a right gates |
| `places`, `waiting`, `bodies`, `rights`, `controls` | counts: places of a body, households waiting for one, and bodies, rights and controls in the construction summary |
| `goal`, `buffer` | money in minor units (`int64`), in the two `Debug` lines of the saving module |
| `field`, `fields`, `gain` | a technology field ID, a count of fields in the construction summary, and a gain in basis points (`int64`) |
| `actor`, `lever`, `reading` | IDs of a policy actor, of one of its levers and of the reading a rule used |
| `entries`, `changed`, `actors`, `levers` | counts: entries a policy actor returned, settings changed, and actors and levers in the construction summary |
| `cover_policy`, `sum`, `premium`, `payout` | an insurance cover's policy ID, and its sum, premium and payout in minor units (`int64`) |
| `underwriters`, `insured` | counts of underwriters and of insured agents, in the construction summary |
| `outcomes`, `previous` | `outcomes`: a count of order outcomes (`int64`), the denominator of the "price beliefs at bound" line; `previous`: a price in minor units (`int64`), the centre of a belief before an update (`price` is the one after) |

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

A field that names a configured entity (a region, a commodity, a role, a
recipe, a resource, a transport mode, a want) carries the typed integer ID of
its kind, and an agent is named by its `AgentID`. A reason, a cause, a kind and
a side are constants of a closed set, exported from the root package
(`event.go`). The strings of the IDs are in the names table that `WorldCreated`,
`WorldRestored` and the snapshot carry. A money flow and a fiscal flow are the
exception: neither has a typed ID, and each is named by its string ID, which is
also its ledger reason (the two kinds share one ID namespace). An
event of a transition of one agent carries the agent's region at the time of the
event, so that an observer that keeps no agent table can attribute it.
**[project]**

4.2. Adding fields to an event is allowed. Removing or renaming a field, or
changing its meaning, is a breaking change (architecture.md 6.6). A field is
appended at the end of the event, and a field that a module appends to an event
that exists without it **MUST** be a count, an amount, a bool, a constant of a
closed set or a slice, never a typed ID: the golden rendering omits a field at
its zero value, so a world with the module off renders as it did before the
module existed (architecture.md 2.10.5, `spec-observability.md` 9.5.2).
**[project]**

4.3. The library **MUST** emit at least the following events. Adding a new
state transition to the library without an event for it needs a written reason
in the PR. The table is the catalogue of the finished library. The code is
built in stages (`doc/roadmap-v1.md`), so the catalogue test checks the events
that are declared and holds the rest in one explicit `notBuilt` list, in the
staged form of architecture.md 9.7: "emit at least" binds the events that are
built, and an event is declared in the change that builds its owner.
**[project]**

| Event type                | Emitted when                                         |
|---------------------------|------------------------------------------------------|
| `WorldCreated`            | the world is constructed and validated (seed, the whole configuration hash, whether a warm start was present, the names table of the IDs, which also holds the store classes, settlement classes, facility classes, technology fields, jurisdictions, policy actors, bodies, rights and controls of the modules that are on) |
| `WorldRestored`           | a checkpoint is restored and validated: the first event to the observers given to the restore, and none when the restore fails (the configuration hash, the format version, whether the run was reseeded and the salt, the names table, and the snapshot at the restored boundary) |
| `RunStarted`              | `Run` begins (ticks requested)                       |
| `RunEnded`                | `Run` returns, after its last completed tick (ticks completed, whether it was cancelled, whether it failed) |
| `TickStarted`, `TickEnded`| each `Step` begins; every step that began ends, a failed one with `Failed` true (a panic emits nothing) |
| `OrderPlaced`             | an agent submits a bid or ask (region, commodity, agent, side, limit price, quantity) |
| `OrderRejected`           | an order fails validation or cannot be accepted (region, commodity, agent, side, `reason`) |
| `MarketCleared`           | a market finishes matching, once per market per tick, also when nothing traded (price, price age, volume, unmatched bid and ask units, best unmatched bid and ask, the cost anchor in force) |
| `TradeExecuted`           | a bid and an ask are matched                         |
| `RecipeExecuted`          | a job completes, and a one-tick job completes in its start tick (agent, region, recipe, start tick, executions, outputs, disposed quantities, effective yield, and the overflow: the part of the disposed quantities that is not a discard output) |
| `RecipeRefused`           | once per region and recipe per tick in which the job request of at least one free household was refused with a count of 0, or was not made because the first recipe that could not run was blocked, after `ProductionHalted` in phase 2; never when none was (households refused for each reason of the master order of the production spec: not on the role's list, zero yield, no site, no tool, no input) |
| `RecipeAbandoned`         | a job ends without output (agent, region, recipe, executions, `Reason`) |
| `ProductionHalted`        | once per region and role per tick in which the production halt declined the job request of at least one free household, or at least one free household rested or found no recipe worth running, after `OwnUseChanged` in phase 2; never when none did (free households, halted households, and per recipe skipped the households for which it was, rested households, households with no recipe worth running) |
| `AgentBankrupted`         | an agent is removed entirely; the only event of a full removal, whatever the cause (weight at removal, `Cause`: `insolvent`, `starved`, `consumer` or `population`; money and `GoodsListed`, the number of lots passed to the estate) |
| `AgentSpawned`            | an agent enters the world, including each initial agent during construction, after `WorldCreated` (region, `Role`, `Weight`, `StartMoney` received, `External`, and `StoresLocked`, the stores of the role row not given because the region's class lacks the flag) |
| `UpkeepShortfall`         | once per region and role per tick in which at least one household went unfed, after the consumption phase; never when all are fed (households, unfed households, agents unfed, largest hunger, and per upkeep commodity the units short) |
| `BeliefsUpdated`          | once per market per tick that had at least one order outcome, after the outcomes are processed (counts of outcomes, adjustments, raises, lowerings, beliefs at floor and ceiling, conflicts and clamps; mean, low and high centre) |
| `BoundaryFlow`            | once per tick for each (region, reason, money or commodity) with a non-zero total, for the reasons of kind faucet or sink only, after the tick's transitions (reason, faucet or sink, total quantity); the construction bucket of a world that has not yet stepped is emitted once, with `Tick` -1, at the entry of its first step, before `TickStarted` |
| `WearApplied`             | once per tick for each commodity whose wear units of the tick are above 0, in ascending commodity order, after the tick's `BoundaryFlow` rows; never when no item wore; the construction bucket's wear units are emitted in the same form with `Tick` -1 after the construction `BoundaryFlow` rows (commodity, units: the durability actually removed, which is not the items retired that the `wear` reason of `BoundaryFlow` counts) |
| `EstateOpened`            | an agent is removed and its estate is opened, accompanying the removal event (agent, region, balance, number of lots listed) |
| `ReliefPaid`              | once per region per tick in which any relief is paid (total paid, number of recipients, households) |
| `TraineeshipStarted`, `TraineeshipCompleted`, `TraineeshipEnded` | a traineeship starts, completes, or ends without completing (identifiers and region only, and `reason` for the end) |
| `WeightChanged`           | an agent's weight changes without removing it: a reduction by the failure ladder, households added to an existing agent, a birth or death of the population module (agent, region, role, `From`, `To`, `Cause`, the money and the units of goods moved) |
| `RoleChanged`             | an agent changes role (agent, region, `From`, `To`, weight, `Cause`, `GoodsListed`) |
| `EntryLimited`            | once per region per tick in which a limit or refusal of the spawner acted; never when all counts are zero (counts of role changes, entrants, removals, unfunded households, households with no viable role, and each cap, cooldown, hysteresis, exit-cap, role ceiling and refusal that acted) |
| `WantShortfall`           | once per region, role and want per tick in which a want was choked or unfunded for at least one household, after the trade phase (households, choked households, planned and funded units) |
| `AgentRelocated`          | an agent's account changes region, once per relocation, after it is complete (agent, role, `From`, `To`, weight, money, units of goods, `Cause`) |
| `EdgeStatusChanged`       | an edge's status changed, once per changed edge in edge order after the new graph is in place (`From`, `To`, `Closed`, `FactorBP`, `Cause`, which includes `policy` for a policy actor's write at P0.3a); only when something sets a status |
| `MoneyFlowLimited`        | once per money flow and region per tick in which any amount due was not applied; never when all was applied (flow, region, due, applied, and the households short, cut at the balance bound and cut by the limit) |
| `MoneyFlowEnded`          | a money flow reaches its effective limit and ends, once (flow, `Cause`: `limit` or `cap`, total) |
| `MoneyFlowScaled`         | a money flow's scale is changed between steps, or by a policy actor at P0.3a, once per change (flow, from, to, `Cause`: `consumer` or `policy`) |
| `PopulationChanged`       | once per region and role per tick in which the population module is on and any count other than the exposed, unfed and no-ratio households is non-zero, after the region's plan and after `EntryLimited`; never when all are zero (region, role, exposed, unfed and no-ratio households, births and deaths due, deaths cut, deaths held, births capped, births cut by a role ceiling, births unfunded, rates clamped, births and deaths done) |
| `ShipmentDispatched`      | a shipment is dispatched (shipment, mode, origin region, destination, shipper, traveller and whether hired, consignee, arrival tick, carrying units, haul units redeemed, the toll paid, the loss in basis points, lines) |
| `ShipmentArrived`         | a shipment arrives, after its delivery (shipment, mode, origin, destination, receiver, whether diverted, and per line the units shipped, delivered, diverted, spoiled and lost, and the amounts lost to an account) |
| `ShipmentRejected`        | a shipment request is refused (origin region, agent, mode, destination, `reason`) |
| `EndowmentChanged`        | once per endowment (region, resource) per tick, after its regeneration at the end of phase 8, in (`RegionID`, resource) order, if any of the units drawn, regenerated, short or lost is non-zero; never otherwise (stock after the tick, effective capacity, abundance of the tick, units requested, granted, drawn and short, regenerated, lost, agents denied, agents reduced) |
| `EndowmentAdjusted`       | a shock slot or an adjustment of an endowment is applied between ticks, once per changed key of a call, or a facility changes its endowment at P8.3b (kind `adjust`, `shock` or `facility`, signed delta, the scales after, stock before and after, units lost) |
| `RumoursServed`           | once per reader region per tick in which the information module is on and at least one informed agent was given a view there, in ascending `RegionID`, after the last plan of phase 3 and before phase 4 (region, readers, and counts of candidates offered, with no path, too old and not yet, and the largest lag offered) |
| `OutsideTraded`           | once per (node, gate region, commodity, side) per tick in which the side wanted any unit or any unit was cut, after the sweep of phase 4; never when nothing was wanted or cut (node, gate region, commodity, side, mid, limit price, wanted, offered and traded units, money paid or received, list value, units cut by a closed edge, by a cap and by a price below 1, whether a price clamp acted) |
| `OutsideEnded`            | a side of an outside node reaches its cumulative cap and ends, once (node, commodity, side, `Cause`: `units` or `value`, cumulative units and list value) |
| `OutsideShocked`          | the price shock of an outside line is set between steps, once per changed key of a call (node, commodity, from, to) |
| `OwnUseChanged`           | once per region and role per tick in phase 2, after the starts, in ascending `RegionID` then role order, if any of the households entered, left, started, blocked or retried, or the units output, is non-zero; never otherwise (households of the role, households in the own-use mode, entered, left, executions of own-use recipes started, blocked, retried, and the units created at the completion of own-use jobs, by commodity) |
| `MigrationStarted`        | a migration journey begins, once per departure, in the order of admission, after the departure is complete (agent, role, `From`, `To`, weight, arrival tick, distance, journey time, earned ratio, home figure, estimate, whether a job was abandoned) |
| `MigrationReviewed`       | once per region per tick in which the migration module is on and any household was pushed or held back by the settling rule, any arrival was held or any journey ended by a removal, in ascending `RegionID` after the `MigrationStarted` events of the pass; never when all are zero (households unsettled, reviewed, pushed, and the eleven outcomes started, no estimate, no destination, no news, too far, not viable, no gain, bound, out cap, unprovisioned, in cap, and arrivals held and journeys ended) |
| `ContractFormed`          | a delivery contract is formed, after its shipment's `ShipmentDispatched` (contract, payer, deliverer, origin region, destination, commodity, quantity, price, money escrowed, shipment, mode, arrival tick) |
| `ContractSettled`         | a delivery contract ends: in phase 1 after the arrivals of transport, or at the removal of its deliverer (contract, payer, deliverer, the deliverer's region, destination, commodity, quantity, units reached and spoiled, money paid and refunded, units lost, `Cause`: `delivered`, `short`, `payer_removed` or `deliverer_removed`) |
| `ContractRound`           | once per destination region per tick in which an offer or a take was submitted for it, in ascending `RegionID`, after the round (admitted and refused offers and takes, matches, contracts formed, failures by reason, units and value formed); never otherwise |
| `MerchantsPlanned`        | once per region per tick in which an agent with a merchant block was given a view there, after `ContractRound` and before `RumoursServed` (merchants, counts of the lanes considered by result, trips planned, requests, offers and takes, rests, stays and lots cut by the cap) |
| `YieldShocked`            | the yield shock of a (region, recipe) is set between ticks, once per changed key of a call (region, recipe, the factor before and after, in basis points) |
| `TransitFactorChanged`    | the transit factor of an (origin, destination, mode) is set between ticks, or by a policy actor at P0.3a, once per changed key of a call (origin, destination, mode, the factor before and after, in basis points, `Cause`: `consumer` or `policy`) |
| `PopulationScaled`        | a birth or death scale of a region is set between ticks, once per changed key of a call (region, which scale, the value before and after, in basis points) |
| `RoomLimited`             | once per region and role per tick at P3.6a, in ascending `RegionID` then role order, when the storage module is on and a room cut or refused a start, a bid or a shipment; never when the six counts are zero (limited and full households, starts cut and refused, bids cut and rejected, shipments refused, units discarded) |
| `StoreChanged`            | a store is added to an agent or removed from it, once per store: at a call, in phase 7 after the `RoleChanged` that ended it, or at the facility step of phase 8 for a store facility (agent, region, role, store class, `Kind`: `added` or `removed`, `Cause`: `consumer`, `role_changed` or `facility`, stores, room, load) |
| `HireFormed`              | a hire is formed, in formation order, at P2.5a (hire, region, employer, worker, their roles, recipe, households, wage, ticks, money escrowed, reserve and bid price) |
| `HireEnded`               | a hire ends, once: at P2.1 or P2.7, in phase 7 with a removal, a role change or a move, or at a call (hire, region, employer, worker, recipe, households, working ticks paid, money paid, remainder of the escrow refunded and whether it went to the estate, `Cause`: `completed`, `worker_left`, `worker_role`, `worker_moved`, `employer_left`, `employer_role` or `not_started`) |
| `LabourRound`             | once per region per tick at P2.8b, in ascending `RegionID` after `ProductionHalted`, in which the employment module is on and any offer, vacancy or hire was counted (households offered, hired and unmatched; positions posted, filled and unfilled; offers and vacancies refused; formations refused by reason; wages paid and money refunded; rows by worker role and by employer role) |
| `SettlementClassChanged`  | a region's settlement class changes: at construction after the initial `AgentSpawned` events (the founding) and at P7.2a (region, `From`, `To`, households, `Cause`: `founded`, `grew` or `shrank`) |
| `SettlementHeld`          | once per region at P7.2a in which the dwell rule holds a class change, instead of `SettlementClassChanged` (region, class, target, households, ticks left) |
| `RentPaid`                | once per endowment (region, resource) that has a title per tick at P8.3a, after `EndowmentChanged`, in (`RegionID`, resource) order, if any tier count, idle, held or unserved unit, refusal, cut, payment or forfeit is non-zero or the posted rent changed (rent in force, units by tier, let supply offered and idle, unserved and priced out, tenants and lessors, money paid and forfeited, requests refused for no title and for rent funds, the posted rent after the step) |
| `TitleTransferred`        | a title (a share of an endowment) moves: with a removal, or at a call, once per title moved (region, resource, `From`, `To`, share, price, `Cause`: `removal` or `consumer`, the shares after, and the regions and pools that pay and are paid) |
| `RentScaled`              | the rent scale of an endowment is set between ticks, once per changed key of a call (region, resource, the scale before and after, in basis points) |
| `LoanFormed`              | a claim is formed, in formation order, at P3.3b (claim, creditor, debtor, debtor's region, principal, interest, rate, term, due tick) |
| `LoanEnded`               | a claim ends, once: at P1.7a in ascending claim ID, with a removal, or at a call (claim, creditor, debtor, the regions of the two accounts, whether the creditor is an estate, principal, principal and interest paid, principal and interest lost, `Origin`: `loan` or `payout`, `Cause`: `repaid`, `written_off`, `debtor_removed` or `forgiven`) |
| `LoanPassed`              | a removed creditor's claim passes to its estate, once per claim in ascending claim ID, with the removal (claim, creditor, debtor, the estate's region, amount owed) |
| `CreditRound`             | once per region per tick at P3.3b, in ascending `RegionID` after the last `LoanFormed`, in which a request or an offer was submitted; never when all counts are zero (requests and offers admitted and refused, with the arrears and exposure refusals as subsets, claims formed, principal and interest formed, principal unfilled, fills dropped for too many claims and for overflow) |
| `CreditSettled`           | once per region per tick at P1.7a, in ascending `RegionID` after the `LoanEnded` events of the tick, if a count or a sum is non-zero (for its debtors: claims due, repaid, newly overdue and written off, principal and interest paid, amounts lost, kept out by the shield and cut at a creditor, arrears; for its creditors: money received by agents and by the estate) |
| `CreditRateScaled`        | the rate scale of a region is set between ticks, once per changed key of a call (region, the scale before and after, in basis points) |
| `FacilityChanged`         | a facility is added, built, repaired, installed, transferred or ruined, once per transition: at construction after the initial `AgentSpawned` events, at P2.1 or P2.7, at P7.1 with a removal or a role change, at P8.3b, or at a call (facility, class, region, `To`, owner, `Kind`, `Cause`, condition before and after, effect, units listed to the estate by a forced removal of a store) |
| `FacilityLimited`         | once per region and facility class per tick at P8.3b, in ascending `RegionID` then class order; never when executions cut, starts refused, executions voided and store facilities pending are all zero (executions served, cut for capacity, refused whole and voided, and facilities pending) |
| `StoreLoss`               | once per agent and store class struck at P8.1a, in ascending agent ID then class order, after the whole position is applied (agent, region, store class, stores struck, whether the row's share is common, share in basis points, and the units lost, in commodity order) |
| `RouteRiskScaled`, `StoreRiskScaled` | a route or store risk scale is set between ticks, once per changed key of a call (the route or the region and class, the scale before and after, in basis points) |
| `FiscalFlowApplied`       | once per fiscal flow and region per tick in which a due or an exempt trade occurred, in the order of the flows, at P1.7b (taxes and payouts), P3.6c (tolls) or P4.1a (duties) (flow, `Kind`: `tax`, `duty`, `toll` or `payout`, authority, region, class of the payer or recipient, commodity, due and applied, parties, and the amounts short, cut and exempt) |
| `TreasuryOrdered`         | once per authority per tick at P3.4a in which a standing order was active with a wish of at least 1 (authority, seat region, orders placed, units wanted and placed on each side, units cut for cash, for the hold and for stock) |
| `MembershipChanged`       | once per region and body per tick at P7.2b, in (`RegionID`, body) order, if any household was admitted, lapsed, skipped, barred, left waiting or held; never otherwise (members, places, eligible households, and the counts of each outcome) |
| `RightRefused`            | once per region and right per tick at P7.2b, in (`RegionID`, right) order, if any count is non-zero (starts, sell and buy orders and training places refused to a non-member) |
| `PriceControlled`         | once per region and control per tick at P7.2b, in (`RegionID`, control) order, if any order was lowered to a cap or raised to a floor (commodity, orders lowered and raised, units) |
| `LegalLevelSet`, `RightSuspensionSet` | the cap and floor of a control, or the suspension of a right, is set between ticks, once per changed key of a call (the control with its cap and floor before and after, or the right and whether it is suspended) |
| `SavingPlanned`           | once per region and role per tick at P3.6b, in ascending `RegionID` then role order, after `RoomLimited`, if any triggering count is above zero (households saving, buffered, with a project chosen, building, or barred, by bar; goal and buffer money; requests funded and committed) |
| `TechnologyAdvanced`      | once per region and field per tick at P8.3c, in (`RegionID`, field) order, for each cell in which a counter, a trend, a discovery, a catch-up or a cut occurred; never when the module is off (executions, experience, discoveries, the gain before and after, what each mechanism added, the part cut by the cap, whether at the cap) |
| `TechnologyAdjusted`      | the gain of a (region, field) cell is adjusted between ticks, once per changed key of a call (region, field, the delta asked, the gain before and after, the part the bounds cut) |
| `PolicyDecided`           | once per due policy actor per tick at P0.3a, after `TickStarted` and after the slot events of its application, if any count is non-zero; never when the actor returned nothing and held nothing for a reason (actor, jurisdiction, entries returned, and the counts applied, unchanged, not held, duplicate, with no close, clamped, stepped, held in band and held for no reading) |
| `CoverFormed`             | an insurance cover is formed, in formation order, at P3.3c (cover policy, `Kind`: `route` or `store`, insured, underwriter, their regions, shipment, mode, destination, sum, maximum payout, premium, rate, cover in basis points) |
| `CoverEnded`              | a cover ends, once: at P1.3a, at P8.1b, or with a removal (2f) (cover policy, `Kind`, insured, underwriter, their regions, `Cause`: `no_loss`, `claimed`, `insured_removed` or `underwriter_removed`, loss value, due, paid, short, the payout claim, dropped, cut, whether capped) |
| `CoverRound`              | once per region per tick at P3.3c, in ascending `RegionID` after `CreditRound`, in which a request or an offer was submitted; never when all counts are zero (requests and offers admitted and refused, candidates, covers formed, sum and premium, sum unfilled, and the failures by reason) |
| `CoverSettled`            | once per region and position (P1.3a or P8.1b) in which a cover was settled, in ascending `RegionID` after the `CoverEnded` events of the position; never when all counts are zero (covers ended and claimed, loss value, due, paid, short, dropped, cut, capped and kept out by the shield) |

`GoodsListed`, in `AgentBankrupted`, `EstateOpened` and `RoleChanged`, counts
lots: the number of lots passed to the estate, one per distinct lot key whatever
the unit count. Units travel separately in `WeightChanged.Units`.

The parentheses name the main fields; the fields of each event, and the order
in which the events of one step are emitted, are fixed by the spec that owns
the transition (`doc/specs/`: production for the recipe events, agents for
`AgentSpawned`, the removal event and `UpkeepShortfall`, beliefs, clearing,
ledger, estates, skill barriers and configuration for the rest). Every field
obeys 4.1: no map, no pointer, fixed-point values. `BoundaryFlow`, `WearApplied`,
`BeliefsUpdated`, `UpkeepShortfall`, `EntryLimited`, `WantShortfall`, `MoneyFlowLimited`, `PopulationChanged`, `EndowmentChanged`, `RumoursServed`, `OutsideTraded`, `OwnUseChanged`, `MigrationReviewed`, `ContractRound`, `MerchantsPlanned`, `ProductionHalted`, `RecipeRefused`, `RoomLimited`, `LabourRound`, `SettlementHeld`, `RentPaid`, `CreditRound`, `CreditSettled`, `FacilityLimited`, `FiscalFlowApplied`, `TreasuryOrdered`, `MembershipChanged`, `RightRefused`, `PriceControlled`, `SavingPlanned`, `TechnologyAdvanced`, `PolicyDecided`, `CoverRound` and `CoverSettled` are aggregates: each is bounded by
regions, commodities, reasons, roles, wants, recipes, money flows, resources, outside nodes, facility classes, fiscal flows, bodies, rights, controls, technology fields or policy actors in the configuration, so none is per
agent, order or trade. `WeightChanged` and `RoleChanged` are per agent, like `AgentSpawned` and `AgentBankrupted`: each is a state transition of one agent. `AgentRelocated` and `MigrationStarted` are per agent too, `ContractFormed` and `ContractSettled` are per contract, like `ShipmentDispatched`, and `HireFormed` and `HireEnded` are per hire, `LoanFormed`, `LoanEnded` and `LoanPassed` per claim, `CoverFormed` and `CoverEnded` per cover, `TitleTransferred` per title, `StoreChanged` per store, `FacilityChanged` per facility transition, `StoreLoss` per agent and store class struck and `SettlementClassChanged` per region: each is a state transition of one entity. `ShipmentDispatched`, `ShipmentArrived` and `ShipmentRejected` are per shipment or request, like `OrderPlaced` and `OrderRejected`: their `Lines` are copies in commodity order, bounded by `MaxShipmentLines`. `EdgeStatusChanged`, `MoneyFlowEnded`, `MoneyFlowScaled`, `EndowmentAdjusted`, `OutsideEnded`, `OutsideShocked`, `YieldShocked`, `TransitFactorChanged`, `PopulationScaled`, `RentScaled`, `CreditRateScaled`, `RouteRiskScaled`, `StoreRiskScaled`, `LegalLevelSet`, `RightSuspensionSet` and `TechnologyAdjusted` are per edge, flow, side or changed key of a call. An event of a module that is switched off is not emitted: modules are off unless configured (ADR 0001), so `ReliefPaid`, `Traineeship*`, `WantShortfall`, `MoneyFlowLimited`, `MoneyFlowEnded`, `MoneyFlowScaled` (the money supply module or the fiscal module), `PopulationChanged`, `ShipmentDispatched`, `ShipmentArrived`, `ShipmentRejected`, `EndowmentChanged`, `EndowmentAdjusted`, `RumoursServed`, `OutsideTraded`, `OutsideEnded`, `OutsideShocked`, `OwnUseChanged` (subsistence), `MigrationStarted` and `MigrationReviewed` (migration), `ContractFormed`, `ContractSettled` and `ContractRound` (merchants) and the events of any later module appear only when it is on, `MerchantsPlanned` only when a role has the merchant block, and likewise `HireFormed`, `HireEnded`, `LabourRound`, `LoanFormed`, `CreditRound`, `SavingPlanned`, `CoverFormed` and `CoverRound` only when a role has the block of their module; `PopulationScaled` appears only with the population module and `TransitFactorChanged` only with the transport module; `AgentRelocated`, `EdgeStatusChanged`, `YieldShocked` (the yield shock is core data), `ProductionHalted` and `RecipeRefused` depend on no module. The order of the events of a step follows the phases: after `TickStarted`, at P0.3a, the slot events of each due policy actor's write (cause `policy`) and then its `PolicyDecided`, in ascending actor order; arrivals of shipments in shipment order in phase 1, after them the `ContractSettled` events of the contracts whose shipment arrived in contract ID order and the migration arrivals (`AgentRelocated` with the cause `migration`) in ascending agent ID, in the order the tick positions of WLD fix (N-009), then the positions of the later modules: P1.3a the `CoverEnded` events of the route covers of the shipments that arrived in ascending cover ID and then `CoverSettled` per region, P1.7a `LoanEnded` in ascending claim ID and then `CreditSettled` per region, and P1.7b `FiscalFlowApplied` for taxes and payouts; in phase 2 `HireEnded` at P2.1 and P2.7, `HireFormed` at P2.5a, `OwnUseChanged` after the starts, `RecipeRefused` at P2.8a after `ProductionHalted`, and `LabourRound` at P2.8b after `RecipeRefused`; in phase 3 each agent's shipment events in plan order before its `OrderPlaced` events, then, after the last plan, the `ShipmentDispatched` and `ContractFormed` events of each contract of the round in formation order, then `ContractRound` and `MerchantsPlanned`, and at P3.3b `LoanFormed` in formation order then `CreditRound`, at P3.3c `CoverFormed` in formation order then `CoverRound` after `CreditRound`, at P3.4a `TreasuryOrdered`, and in the tail P3.6a `RoomLimited`, P3.6b `SavingPlanned` and P3.6c the toll `FiscalFlowApplied` events, after `WantShortfall`; in phase 7 `SettlementClassChanged` or `SettlementHeld` at P7.2a and `MembershipChanged`, `RightRefused` and `PriceControlled` at P7.2b, and `MigrationStarted` in the order of admission and then `MigrationReviewed`, after `EntryLimited` and `PopulationChanged` of the last region, and the `ContractSettled` of a deliverer's removal with the removal events; the duties `FiscalFlowApplied` come at P4.1a, `StoreLoss` at P8.1a, `CoverEnded` of the paid store covers then `CoverSettled` at P8.1b, and in phase 8 `RentPaid` at P8.3a after `EndowmentChanged`, the `FacilityChanged`, store `StoreChanged` and `EndowmentAdjusted` (kind `facility`) events then `FacilityLimited` at P8.3b, and `TechnologyAdvanced` at P8.3c. `RumoursServed` comes in phase 3 after the last plan, after `MerchantsPlanned` when the merchant module is on, and before phase 4, in ascending `RegionID`; the outside's events (`OutsideTraded`, `OutsideEnded`) come after the sweep at the end of phase 4; `EndowmentChanged` comes at the end of phase 8 in (`RegionID`, resource) order. The construction bucket's `BoundaryFlow` comes at the entry of the first step, before `TickStarted`; at construction `SettlementClassChanged` (the founding) and `FacilityChanged` (the initial facilities) come after the initial `AgentSpawned` events; `ProductionHalted` comes after `OwnUseChanged` in phase 2; in a region's entry step the traineeship events come first (a completion emits the `RecipeAbandoned` of its job, if any, then `TraineeshipCompleted`, then the `RoleChanged` it causes, and each place filled `TraineeshipStarted`); the slot events (`YieldShocked`, `TransitFactorChanged`, `PopulationScaled` and the others of a call) come at the call, after the whole call has been applied, except a policy actor's write, which comes at P0.3a after `TickStarted`. A removal emits `RecipeAbandoned` for its job, `TraineeshipEnded` if it was in a traineeship, `ContractSettled` for each contract of which it was the deliverer, then the events of the removal steps of `spec-agents.md` 3.6 in step order: `TitleTransferred` for each title moved (2b), `HireEnded` for each hire (2c, the worker's side and then the employer's), `LoanEnded` for each claim ended (2d) and `LoanPassed` for each claim passed to the estate (2e), `CoverEnded` for each cover ended (2f) and `FacilityChanged` for each facility ruined (2g), and last `AgentBankrupted` and `EstateOpened`, in that order (the removal of a trainer first ends its trainees' traineeships, in ascending agent ID): every event of a removal precedes `AgentBankrupted`; `AgentBankrupted` is the only event that removes the agent, and the others accompany it.

Because observers are supplied as options to `NewWorld`, they receive
`WorldCreated` and the initial `AgentSpawned` events. A consumer can rebuild
the total live agent count from `AgentSpawned` and `AgentBankrupted` alone. The
household count also needs `WeightChanged` (`To` less `From`) besides the `Weight` those two carry, and a count by role also needs `RoleChanged` (the agent and its weight leave `From` and join `To`). A count of households by region also needs `AgentRelocated` (the weight leaves `From` and joins `To`). Estate pseudo-agents and the treasuries of governed jurisdictions are not
agents and are not counted in any of them. An observer given to a restore receives no `AgentSpawned` for the restored agents: it starts from `WorldRestored`, whose snapshot holds the agents and the households by role and region at the restored boundary, and adds the events after it. The money of the household sector and of the pool is rebuilt from the same snapshot, its pending totals (`Pending`) and, for a world that has not stepped, the construction bucket (`Construction`). The money of a treasury is in the `Jurisdictions` rows and `MoneyTreasury` of the snapshot, and the treasury's agent ID is classed by the `Jurisdictions` rows of `WorldCreated`; the escrow of the hires is in the contract sector; the sets in flight at a restore (open hires, claims and covers, with the contracts and shipments) are rows of the same snapshot.

4.4. Observers are called synchronously on the stepping goroutine and **MUST
NOT** call back into the `World` (architecture.md 12.5). The library **MUST
NOT** hold internal locks or leave state half-updated while calling an observer:
emit an event only after the transition it describes is complete. A failed step
is the one exception: the world is marked failed before `TickEnded` is emitted,
and the event describes the end of the step (`spec-world.md` 3.5.6). Observers
**MUST** ignore event types they do not recognise.

4.5. When the observer is the no-op default, constructing events in hot paths
**MUST NOT** allocate. Passing an event struct as an `Event` boxes it, so the
world records once, at construction, whether any observer was supplied, and
constructs and emits events only when it was. A counter that exists only to
build an event is built only under the same flag, and a restore records the flag
once and builds the `WorldRestored` snapshot only when an observer was supplied.
Prove this with an allocation test (8.2). **[project]**

## 5. Metrics

Metrics are produced from events, and for some instruments from a snapshot that
the consumer passes to the adapter (`RecordSnapshot`), by the separate
`otelobserver` module. The core module defines events and the snapshot; it does
not define or record metrics, and it does not import OpenTelemetry
(architecture.md 11.4). The rules in this section apply to `otelobserver` and to
anyone adding events or snapshot fields that metrics will be built from.

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
| a point-in-time value that is not additive, or a level set from a snapshot (last clearing price) | Gauge  |

### 5.3 Initial catalogue

These are the initial instruments. Changing a name, unit or attribute set
breaks users' dashboards and alerts and is a breaking change, once there is a
release: until the first release the catalogue may change freely, because no
code exists and no dashboard depends on it. Every instrument
except the duration histogram records integers, so use the `Int64` variants;
prices are recorded in minor units, unconverted (architecture.md 3.5).
**[project]**

| Name                                | Instrument      | Unit            | Attributes        |
|-------------------------------------|-----------------|-----------------|-------------------|
| `economysim.world.step.duration`    | Histogram       | `s`             | none              |
| `economysim.world.step.failed`      | Counter         | `{step}`        | none              |
| `economysim.market.trades`          | Counter         | `{trade}`       | `economysim.commodity` |
| `economysim.market.volume`          | Counter         | `{unit}`        | `economysim.commodity` |
| `economysim.market.clearing_price`  | Gauge           | `{minor_unit}`  | `economysim.commodity`, `economysim.region` |
| `economysim.market.price_age`       | Gauge           | `{tick}`        | `economysim.commodity`, `economysim.region` |
| `economysim.market.unmatched`       | Counter         | `{unit}`        | `economysim.commodity`, `economysim.side` |
| `economysim.market.orders`          | Counter         | `{order}`       | `economysim.commodity`, `economysim.region`, `economysim.side` |
| `economysim.market.orders.rejected` | Counter         | `{order}`       | `economysim.commodity`, `economysim.reason` |
| `economysim.belief.centre`          | Gauge           | `{minor_unit}`  | `economysim.commodity`, `economysim.region` |
| `economysim.belief.at_bound`        | Gauge           | `{basis_point}` | `economysim.commodity`, `economysim.region`, `economysim.bound` |
| `economysim.belief.outcomes`        | Counter         | `{outcome}`     | `economysim.commodity`, `economysim.region` |
| `economysim.belief.limited`         | Counter         | `{outcome}`     | `economysim.commodity`, `economysim.region`, `economysim.belief_limit` |
| `economysim.recipe.executions`      | Counter         | `{execution}`   | `economysim.recipe` |
| `economysim.recipe.refused`         | Counter         | `{household}`   | `economysim.recipe`, `economysim.reason` |
| `economysim.recipe.abandoned`       | Counter         | `{execution}`   | `economysim.recipe`, `economysim.reason` |
| `economysim.recipe.overflow`        | Counter         | `{unit}`        | `economysim.commodity` |
| `economysim.recipe.in_progress`     | Gauge           | `{execution}`   | `economysim.recipe`, `economysim.region` |
| `economysim.production.cost_anchor` | Gauge           | `{minor_unit}`  | `economysim.commodity`, `economysim.region` |
| `economysim.production.halted`      | Counter         | `{household}`   | `economysim.role`, `economysim.recipe` |
| `economysim.work.declined`          | Counter         | `{household}`   | `economysim.role`, `economysim.work_decline` |
| `economysim.agent.count`            | UpDownCounter   | `{agent}`       | `economysim.role` |
| `economysim.agent.removals`         | Counter         | `{agent}`       | `economysim.role`, `economysim.cause` |
| `economysim.household.count`        | UpDownCounter   | `{household}`   | `economysim.role` |
| `economysim.household.unfed`        | UpDownCounter   | `{household}`   | `economysim.role` |
| `economysim.upkeep.short`           | Counter         | `{unit}`        | `economysim.commodity`, `economysim.role` |
| `economysim.household.exits`        | Counter         | `{household}`   | `economysim.role`, `economysim.cause` |
| `economysim.role.changes`           | Counter         | `{change}`      | `economysim.role` (the destination role) |
| `economysim.entry.limited`          | Counter         | `{case}`        | `economysim.entry_limit` |
| `economysim.want.unfunded`          | Counter         | `{unit}`        | `economysim.want`, `economysim.role` |
| `economysim.household.relocations`  | Counter         | `{household}`   | `economysim.region.from`, `economysim.region.to`, `economysim.cause` |
| `economysim.money.flow`             | Counter         | `{minor_unit}`  | `economysim.reason`, `economysim.kind` (`faucet` or `sink`) |
| `economysim.commodity.flow`         | Counter         | `{unit}`        | `economysim.reason`, `economysim.kind`, `economysim.commodity` |
| `economysim.commodity.wear`         | Counter         | `{unit}`        | `economysim.commodity` |
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
| `economysim.external.price`         | Gauge           | `{minor_unit}`  | `economysim.node`, `economysim.region`, `economysim.commodity`, `economysim.side` |
| `economysim.household.own_use`      | Gauge           | `{household}`   | `economysim.role`, `economysim.region` |
| `economysim.own_use.units`          | Counter         | `{unit}`        | `economysim.commodity` |
| `economysim.migration.started`      | Counter         | `{household}`   | `economysim.region.from`, `economysim.region.to` |
| `economysim.migration.limited`      | Counter         | `{household}`   | `economysim.limit` |
| `economysim.migration.on_journey`   | UpDownCounter   | `{household}`   | none |
| `economysim.contract.formed`        | Counter         | `{contract}`    | `economysim.commodity` |
| `economysim.contract.value`         | Counter         | `{minor_unit}`  | `economysim.commodity` |
| `economysim.contract.escrow`        | UpDownCounter   | `{minor_unit}`  | none |
| `economysim.merchant.decisions`     | Counter         | `{lane}`        | `economysim.decision` |
| `economysim.production.pending`     | Gauge           | `{unit}`        | `economysim.commodity`, `economysim.region` |
| `economysim.market.stock`           | Gauge           | `{unit}`        | `economysim.commodity`, `economysim.region`, `economysim.holder` (`agents`, `estate` or `treasury`) |
| `economysim.estate.sold`            | Counter         | `{unit}`        | `economysim.commodity`, `economysim.region` |
| `economysim.estate.at_floor`        | Gauge           | `{unit}`        | `economysim.commodity`, `economysim.region` |
| `economysim.ledger.lots`            | Gauge           | `{lot}`         | `economysim.commodity`, `economysim.region` |
| `economysim.money.supply`           | Gauge           | `{minor_unit}`  | `economysim.region`, `economysim.sector` (`households`, `pool`, `contract` or `treasury`) |
| `economysim.welfare.ratio`          | Gauge           | `{basis_point}` | `economysim.role`, `economysim.region` |
| `economysim.welfare.ratio.households` | Gauge         | `{household}`   | `economysim.role`, `economysim.region` |
| `economysim.welfare.floor_held`     | Gauge           | `{basis_point}` | `economysim.role`, `economysim.region` |
| `economysim.wage.observed`          | Gauge           | `{minor_unit}`  | `economysim.role`, `economysim.region` |
| `economysim.household.privation.max` | Gauge          | `{tick}`        | `economysim.role`, `economysim.region` |
| `economysim.storage.limited`        | Counter         | `{case}`        | `economysim.region`, `economysim.role`, `economysim.room_limit` |
| `economysim.store.count`            | Gauge           | `{store}`       | `economysim.region`, `economysim.store_class` |
| `economysim.room.fill`              | Gauge           | `{basis_point}` | `economysim.region`, `economysim.role`, `economysim.dimension` |
| `economysim.hire.formed`            | Counter         | `{hire}`        | `economysim.role` (the employer's) |
| `economysim.hire.wage`              | Counter         | `{minor_unit}`  | `economysim.region` |
| `economysim.labour.unmatched`       | Counter         | `{household}`   | `economysim.role` (the worker's) |
| `economysim.vacancy.unfilled`       | Counter         | `{position}`    | `economysim.role` (the employer's), `economysim.reason` |
| `economysim.hire.escrow`            | UpDownCounter   | `{minor_unit}`  | none |
| `economysim.settlement.class`       | Gauge           | `{class}`       | `economysim.region` |
| `economysim.settlement.changes`     | Counter         | `{change}`      | `economysim.region` |
| `economysim.rent.paid`              | Counter         | `{minor_unit}`  | `economysim.resource`, `economysim.region` |
| `economysim.rent.units`             | Counter         | `{unit}`        | `economysim.resource`, `economysim.region`, `economysim.tier` |
| `economysim.rent.rate`              | Gauge           | `{minor_unit}`  | `economysim.resource`, `economysim.region` |
| `economysim.rent.refusals`          | Counter         | `{request}`     | `economysim.resource`, `economysim.region`, `economysim.reason` |
| `economysim.title.transfers`        | Counter         | `{transfer}`    | `economysim.resource`, `economysim.region`, `economysim.cause` |
| `economysim.credit.formed`          | Counter         | `{claim}`       | none |
| `economysim.credit.principal`       | Counter         | `{minor_unit}`  | none |
| `economysim.credit.interest`        | Counter         | `{minor_unit}`  | none |
| `economysim.credit.outstanding`     | UpDownCounter   | `{minor_unit}`  | none |
| `economysim.credit.arrears`         | Gauge           | `{minor_unit}`  | `economysim.region` |
| `economysim.credit.defaults`        | Counter         | `{claim}`       | none |
| `economysim.credit.lost`            | Counter         | `{minor_unit}`  | `economysim.cause` |
| `economysim.credit.refused`         | Counter         | `{request}`     | none |
| `economysim.credit.limited`         | Counter         | `{case}`        | `economysim.credit_limit` |
| `economysim.facility.count`         | UpDownCounter   | `{facility}`    | `economysim.facility_class`, `economysim.region` |
| `economysim.facility.limited`       | Counter         | `{execution}`   | `economysim.facility_class`, `economysim.region`, `economysim.facility_limit` |
| `economysim.facility.refused`       | Counter         | `{start}`       | `economysim.facility_class`, `economysim.region` |
| `economysim.facility.pending`       | Gauge           | `{facility}`    | `economysim.facility_class`, `economysim.region` |
| `economysim.facility.condition`     | Gauge           | `{basis_point}` | `economysim.facility_class`, `economysim.region` |
| `economysim.risk.strikes`           | Counter         | `{strike}`      | `economysim.risk_kind` |
| `economysim.transport.cargo.lost`   | Counter         | `{unit}`        | `economysim.commodity` |
| `economysim.store.lost`             | Counter         | `{unit}`        | `economysim.commodity`, `economysim.store_class` |
| `economysim.fiscal.collected`       | Counter         | `{minor_unit}`  | `economysim.reason` (the fiscal flow ID) |
| `economysim.fiscal.paid`            | Counter         | `{minor_unit}`  | `economysim.reason` |
| `economysim.fiscal.in_kind`         | Counter         | `{unit}`        | `economysim.reason`, `economysim.commodity` |
| `economysim.fiscal.unapplied`       | Counter         | `{minor_unit}`  | `economysim.reason` |
| `economysim.fiscal.order.cut`       | Counter         | `{unit}`        | `economysim.clamp` |
| `economysim.body.members`           | Gauge           | `{household}`   | `economysim.body`, `economysim.region` |
| `economysim.body.waiting`           | Gauge           | `{household}`   | `economysim.body`, `economysim.region` |
| `economysim.right.refusals`         | Counter         | `{refusal}`     | `economysim.right`, `economysim.activity` |
| `economysim.control.clamped`        | Counter         | `{clamp}`       | `economysim.control`, `economysim.side` |
| `economysim.saving.goal`            | Gauge           | `{minor_unit}`  | `economysim.role`, `economysim.region` |
| `economysim.saving.buffer`          | Gauge           | `{minor_unit}`  | `economysim.role`, `economysim.region` |
| `economysim.saving.idle`            | Gauge           | `{minor_unit}`  | `economysim.role`, `economysim.region` |
| `economysim.saving.requests`        | Counter         | `{request}`     | `economysim.role`, `economysim.project` |
| `economysim.saving.barred`          | Counter         | `{household}`   | `economysim.role`, `economysim.bar` |
| `economysim.technology.gain`        | Gauge           | `{basis_point}` | `economysim.field`, `economysim.region` |
| `economysim.technology.experience`  | Counter         | `{execution}`   | `economysim.field`, `economysim.region` |
| `economysim.technology.discoveries` | Counter         | `{discovery}`   | `economysim.field` |
| `economysim.technology.clamped`     | Counter         | `{basis_point}` | `economysim.field` |
| `economysim.policy.decisions`       | Counter         | `{decision}`    | `economysim.actor` |
| `economysim.policy.entries`         | Counter         | `{entry}`       | `economysim.actor`, `economysim.entry_outcome` |
| `economysim.policy.cut`             | Counter         | `{entry}`       | `economysim.actor`, `economysim.policy_cut` |
| `economysim.policy.held`            | Counter         | `{reaction}`    | `economysim.actor`, `economysim.policy_hold` |
| `economysim.insurance.policies`     | Counter         | `{policy}`      | `economysim.cover_kind` |
| `economysim.insurance.premium`      | Counter         | `{minor_unit}`  | none |
| `economysim.insurance.payout`       | Counter         | `{minor_unit}`  | none |
| `economysim.insurance.shortfall`    | Counter         | `{minor_unit}`  | none |
| `economysim.insurance.unpaid`       | Counter         | `{minor_unit}`  | `economysim.cover_loss` |
| `economysim.insurance.liability`    | UpDownCounter   | `{minor_unit}`  | none |
| `economysim.insurance.claims`       | UpDownCounter   | `{minor_unit}`  | none |
| `economysim.insurance.refused`      | Counter         | `{request}`     | none |
| `economysim.insurance.limited`      | Counter         | `{case}`        | `economysim.cover_limit` |

The instruments fed by a snapshot, not by events, are the rows of the second table of 5.3.1; the sector `contract` of `economysim.money.supply` also holds the escrow of the hires.

### 5.3.1 Which event feeds which metric

| Metric                               | Fed by                                  |
|--------------------------------------|-----------------------------------------|
| `economysim.world.step.duration`     | `TickStarted` to `TickEnded`, measured by the adapter; not recorded when `TickEnded` has `Failed` true |
| `economysim.world.step.failed`       | `TickEnded` with `Failed` true (one per failed step) |
| `economysim.market.trades`           | `TradeExecuted`                         |
| `economysim.estate.sold`             | `TradeExecuted` (`Quantity`, where `Seller` is below the region count, so an estate's: the forced seller, kept apart from producers' supply and the pool's inflow from sales) |
| `economysim.market.volume`           | `TradeExecuted` (quantity)              |
| `economysim.market.clearing_price`   | `MarketCleared`                         |
| `economysim.market.price_age`        | `MarketCleared` (`PriceAge`, in ticks; the event is per market per tick, so the series is never stale) |
| `economysim.market.unmatched`        | `MarketCleared` (`UnmatchedBid`, `UnmatchedAsk`, units); the ask side includes the estate's unsold lots, which it asks again every tick, so read it with `economysim.market.stock` under `holder` `estate` |
| `economysim.market.orders`           | `OrderPlaced`                           |
| `economysim.market.orders.rejected`  | `OrderRejected`                         |
| `economysim.belief.centre`           | `BeliefsUpdated` (the mean centre)      |
| `economysim.belief.at_bound`         | `BeliefsUpdated` (`AtFloor`, `AtCeiling`, each `× 10 000 / Outcomes`, by `bound`: `floor`, `ceiling`; the adapter holds the last value for a tick with no event) |
| `economysim.belief.outcomes`         | `BeliefsUpdated` (`Outcomes`): the denominator of `economysim.belief.at_bound`, and the only way to tell a market with no orders from one that is pinned |
| `economysim.belief.limited`          | `BeliefsUpdated` (`Conflicts` as `conflict`, `Clamped` as `clamped`, by `belief_limit`) |
| `economysim.recipe.executions`       | `RecipeExecuted` (its `Executions` field, not one per event) |
| `economysim.recipe.refused`          | `RecipeRefused` (each count, by its reason: `not_on_role`, `zero_yield`, `no_site`, `no_tool`, `no_input`; households, so not the same count as `economysim.production.halted`, which counts requests the halt declined before any start was tried) |
| `economysim.recipe.abandoned`        | `RecipeAbandoned` (`Executions`, by `Reason`: `agent_removed`, `role_changed`, `principal_left`, `region_changed`) |
| `economysim.recipe.overflow`         | `RecipeExecuted` (`Overflow` rows, units, by commodity): the discard outputs, which are waste by design, are in `Disposed` and not here, so this is the anomaly and `economysim.commodity.flow` under `disposal` is the whole |
| `economysim.production.cost_anchor`  | `MarketCleared` (`CostAnchor`; 0 is an unknown anchor) |
| `economysim.production.halted`       | `ProductionHalted` (the households of each recipe skipped; a household counts once for each recipe skipped) |
| `economysim.work.declined`           | `ProductionHalted` (`Rested` as `rest`, `NoRecipe` as `no_recipe`; free households that requested no job, disjoint from `production.halted` and from `recipe.refused`) |
| `economysim.agent.count`             | `AgentSpawned` (+1, including initial agents), `AgentBankrupted` (-1); by role also `RoleChanged` (-1 for `From`, +1 for `To`) |
| `economysim.agent.removals`          | `AgentBankrupted` (every `Cause`)       |
| `economysim.household.count`         | `AgentSpawned` (+`Weight`, including initial agents), `AgentBankrupted` (-`Weight`), `WeightChanged` (`To` less `From`); by role also `RoleChanged` (-`Weight` for `From`, +`Weight` for `To`) |
| `economysim.household.unfed`         | `UpkeepShortfall` (`Unfed`, by role; see below) |
| `economysim.upkeep.short`            | `UpkeepShortfall` (`Short` rows, units by commodity and role: what the role's households needed and did not hold, which `household.unfed` does not say) |
| `economysim.household.exits`         | `AgentBankrupted` (`Weight`) and `WeightChanged` (`From` less `To`, for a reduction), by role and `Cause`; the cause `population` is left out, because `economysim.household.deaths` counts it |
| `economysim.role.changes`            | `RoleChanged` (by `To`)                 |
| `economysim.entry.limited`           | `EntryLimited` (the counts of the limits and refusals that acted, by limit) |
| `economysim.want.unfunded`           | `WantShortfall` (`Planned` less `Funded`, by want and role) |
| `economysim.household.relocations`   | `AgentRelocated` (`Weight`, by `From`, `To` and `Cause`) |
| `economysim.money.flow`              | `BoundaryFlow` (money, by the reason and whether it is a faucet or a sink; includes the construction bucket at `Tick` -1) |
| `economysim.commodity.flow`          | `BoundaryFlow` (goods, by the reason, the kind and the commodity) |
| `economysim.commodity.wear`          | `WearApplied` (`Units`, by commodity; includes the construction bucket at `Tick` -1) |
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
| `economysim.external.price` | `OutsideTraded` (the limit price, by node, gate region, commodity and side) |
| `economysim.household.own_use` | `OwnUseChanged` (households in the mode, by role and region); the adapter keeps the last value for a tick with no event |
| `economysim.own_use.units` | `OwnUseChanged` (units output, per line) |
| `economysim.migration.started` | `MigrationStarted` (`Weight`, by `From` and `To`) |
| `economysim.migration.limited` | `MigrationReviewed` (the counts other than started, pushed, reviewed and unsettled, by outcome: `no_estimate`, `no_destination`, `no_rumour`, `too_far`, `no_viable`, `no_gain`, `bound`, `out_cap`, `unprovisioned`, `in_cap`, `held`, `ended`) |
| `economysim.migration.on_journey` | `MigrationStarted` (+`Weight`), `AgentRelocated` with the cause `migration` (-`Weight`) and `MigrationReviewed` (- `Ended`) |
| `economysim.contract.formed` | `ContractFormed` |
| `economysim.contract.value` | `ContractFormed` (money escrowed) |
| `economysim.contract.escrow` | `ContractFormed` (+ money escrowed), `ContractSettled` (- money paid and refunded) |
| `economysim.merchant.decisions` | `MerchantsPlanned` (the case counts, by decision) |
| `economysim.storage.limited` | `RoomLimited` (each of the five counts, by `room_limit`: `start_cut`, `start_refused`, `bid_cut`, `bid_rejected`, `shipment_refused`) |
| `economysim.hire.formed` | `HireFormed` (by the employer's role) |
| `economysim.hire.wage` | `LabourRound` (money paid) |
| `economysim.labour.unmatched` | `LabourRound` (households offered and not hired, by the worker's role) |
| `economysim.vacancy.unfilled` | `LabourRound` (positions unfilled and the formation refusals, by the employer's role and the reason) |
| `economysim.hire.escrow` | `HireFormed` (+ money escrowed), `LabourRound` (- money paid, - money refunded), `HireEnded` (- remainder refunded); the sector `contract` of `economysim.money.supply` includes it |
| `economysim.settlement.class` | `SettlementClassChanged` (`To`); the adapter keeps the last value, which the events of construction set |
| `economysim.settlement.changes` | `SettlementClassChanged`, except the cause `founded` |
| `economysim.rent.paid` | `RentPaid` (money paid) |
| `economysim.rent.units` | `RentPaid` (units by tier: `own`, `commons`, `let`) |
| `economysim.rent.rate` | `RentPaid` (the posted rent after the step) and `RentScaled` (the scale after); the adapter holds the last values |
| `economysim.rent.refusals` | `RentPaid` (requests refused, by reason `no_title` and `rent_funds`) |
| `economysim.title.transfers` | `TitleTransferred` (by `Cause`: `removal`, `consumer`) |
| `economysim.credit.formed` | `LoanFormed` |
| `economysim.credit.principal` | `LoanFormed` (principal) |
| `economysim.credit.interest` | `CreditSettled` (interest paid) and `LoanEnded` (interest paid) |
| `economysim.credit.outstanding` | `LoanFormed` (+ principal), `CreditSettled` (- principal paid), `LoanEnded` (- principal paid and lost); seeded at a restore from the claims of the snapshot |
| `economysim.credit.defaults` | `CreditSettled` (claims newly overdue) |
| `economysim.credit.lost` | `LoanEnded` (principal and interest lost, by `Cause`) |
| `economysim.credit.refused` | `CreditRound` (requests and offers refused) |
| `economysim.credit.limited` | `CreditRound` (arrears, exposure, too many claims, overflow, by `credit_limit`; the first two are also in `refused`, so the two instruments are not added) |
| `economysim.facility.count` | `FacilityChanged` (+1 for `added` and `built`, -1 for `ruined`) |
| `economysim.facility.limited` | `FacilityLimited` (executions cut and voided, by `facility_limit`: `cut`, `voided`) |
| `economysim.facility.refused` | `FacilityLimited` (starts refused) |
| `economysim.risk.strikes` | `ShipmentDispatched` (1 for each with a loss above 0, by `route`), `StoreLoss` (stores struck, by `store`) |
| `economysim.transport.cargo.lost` | `ShipmentArrived` (each line's units lost and each amount lost to an account) |
| `economysim.store.lost` | `StoreLoss` (units lost) |
| `economysim.fiscal.collected` | `FiscalFlowApplied` (applied money of the kinds `tax`, `duty` and `toll`, by flow) |
| `economysim.fiscal.paid` | `FiscalFlowApplied` (applied money of the kind `payout`) |
| `economysim.fiscal.in_kind` | `FiscalFlowApplied` (applied units of an in-kind `payout`, by flow and commodity) |
| `economysim.fiscal.unapplied` | `FiscalFlowApplied` (due less applied of the flows that move money) |
| `economysim.fiscal.order.cut` | `TreasuryOrdered` (units cut for cash, for the hold and for stock, by `clamp`) |
| `economysim.right.refusals` | `RightRefused` (starts, sell and buy orders and training places, by activity) |
| `economysim.control.clamped` | `PriceControlled` (orders lowered as `bid`, raised as `ask`) |
| `economysim.saving.requests` | `SavingPlanned` (builds as `build`, repairs as `repair`) |
| `economysim.saving.barred` | `SavingPlanned` (households barred, by `bar`: `no_return`, `standing`, `unreachable`, `survival`) |
| `economysim.technology.gain` | `TechnologyAdvanced` and `TechnologyAdjusted` (the gain after); the adapter holds the last value and takes the level from the technology rows of `WorldRestored`'s snapshot |
| `economysim.technology.experience` | `TechnologyAdvanced` (executions) |
| `economysim.technology.discoveries` | `TechnologyAdvanced` (discoveries) |
| `economysim.technology.clamped` | `TechnologyAdvanced` (the part cut by the cap) |
| `economysim.policy.decisions` | `PolicyDecided` (one per event) |
| `economysim.policy.entries` | `PolicyDecided` (applied, unchanged, not held, duplicate, with no close, by `entry_outcome`) |
| `economysim.policy.cut` | `PolicyDecided` (clamped, stepped, by `policy_cut`) |
| `economysim.policy.held` | `PolicyDecided` (held in band, held for no reading, by `policy_hold`) |
| `economysim.insurance.policies` | `CoverFormed` (by kind) |
| `economysim.insurance.premium` | `CoverFormed` (premium) |
| `economysim.insurance.payout` | `CoverSettled` (paid) |
| `economysim.insurance.shortfall` | `CoverSettled` (short) |
| `economysim.insurance.unpaid` | `CoverSettled` (dropped and cut, by `cover_loss`) |
| `economysim.insurance.liability` | `CoverFormed` (+ maximum payout, route covers only), `CoverEnded` (- maximum payout of a route cover); seeded at a restore from the covers of the snapshot |
| `economysim.insurance.claims` | `CoverEnded` (+ `Short`, the payout claim the underwriter owes), `LoanEnded` of origin `payout` (- the claim ended, paid or written off); seeded at a restore from the claims of origin `payout` in the snapshot |
| `economysim.insurance.refused` | `CoverRound` (requests and offers refused) |
| `economysim.insurance.limited` | `CoverRound` (arrears, exposure, no risk, no offer, below minimum, funds, too many, overflow, by `cover_limit`) |

The instruments fed by a snapshot are recorded by the adapter once per
snapshot the consumer passes to it, between two steps, from these fields:

| Metric                               | Fed by                                  |
|--------------------------------------|-----------------------------------------|
| `economysim.production.pending`      | `Markets[].Pending`: the output of jobs in progress, which no account holds |
| `economysim.recipe.in_progress`      | `Jobs[].Executions`: the executions of the jobs in progress, by recipe and region |
| `economysim.market.stock`            | `Markets[].HeldByAgents`, `HeldByEstate` and `HeldByTreasury`, by holder |
| `economysim.estate.at_floor`         | `Markets[].EstateAtFloor`: the units of the estate's listings whose listed price is at the resolution floor, where a markdown cannot lower them; with `economysim.market.stock` under `holder` `estate` it tells a slow markdown from a good nobody wants |
| `economysim.ledger.lots`             | `Markets[].Lots`: the lots of the accounts those three fields count; the sum over the series is the ledger's lot count |
| `economysim.money.supply`            | `Regions[].MoneyHouseholds`, `MoneyPool`, `MoneyContract` and `MoneyTreasury`, by sector |
| `economysim.welfare.ratio`           | `Roles[].MeanRatioBP`, where the role has a mean ratio |
| `economysim.welfare.ratio.households` | `Roles[].RatioHouseholds`, where the role has a mean ratio: the denominator of `economysim.welfare.ratio`, and the only way to tell a mean over one household from a mean over four hundred, or to weight a region's mean across roles |
| `economysim.welfare.floor_held`      | `Roles[].HeldByFloor * 10 000 / Households`, absent where `Households` is 0: the share of the role's households held to work by `MinWorkShare` at the boundary, the quantity the `Warn` "work floor binding" tests against 5 000, but visible below the threshold and between the `Warn`'s tenth-tick lines |
| `economysim.wage.observed`           | `Roles[].WageObserved`, where `HasWage`: the observed wage of the role, in minor units per working household-tick; not the pooled regional wage the cost anchor reads |
| `economysim.household.privation.max` | `Roles[].MaxPrivation`: the largest privation of an agent of the role, the leaky count that the failure ladder reads; a snapshot gauge, because `UpkeepShortfall` is not emitted once everyone is fed again and privation is still falling then |
| `economysim.store.count`             | `Stores[]` rows of the snapshot, by region and store class |
| `economysim.room.fill`               | `Roles[]` room and load, `load * 10 000 / room`, absent where the room is 0 |
| `economysim.credit.arrears`          | `Claims[]` rows of origin `loan` that are overdue, `PrincipalOwed + InterestOwed`, by region |
| `economysim.facility.pending`        | the facility rows, `Pending` (store facilities not installed) |
| `economysim.facility.condition`      | the facility rows, `Condition * 10 000 / Durability` summed over the row |
| `economysim.body.members`            | `Bodies[].Members`, by body and region |
| `economysim.body.waiting`            | `Bodies[]` (`Eligible - Members`), by body and region |
| `economysim.saving.goal`             | `Roles[].GoalMoney` |
| `economysim.saving.buffer`           | `Roles[].BufferMoney` |
| `economysim.saving.idle`             | `Roles[].IdleMoney` |

An instrument fed by a snapshot is absent, not zero, until a snapshot is passed,
and holds its last value between two snapshots.

`UpkeepShortfall` is not emitted for a region and role whose households are all
fed, so the adapter keeps the last unfed count of each region and role, adds the
change to `economysim.household.unfed` when the event comes, and at `TickEnded`
sets to zero every cell that had no event in the step. At a `TickEnded` with
`Failed` true it makes no correction. `economysim.household.count` and `economysim.household.unfed`
count households (the sum of weights), never agents. `economysim.role.changes` counts changes, not agents, and `economysim.want.unfunded` counts units of goods not funded. `economysim.household.exits` counts the households that left by the failure ladder or by a consumer's removal, the weight of each `AgentBankrupted` and each reduction of a `WeightChanged`; a cohort that is cut by the unfed share shows here and not in `economysim.agent.removals`, and the cause `population` is left out because `economysim.household.deaths` counts it. A step that fails is counted by `economysim.world.step.failed` and records no duration. `economysim.market.price_age` is in ticks, and `economysim.belief.at_bound` is the share of a market's order outcomes whose centre ended on a bound, `AtFloor × 10 000 / Outcomes` rounded toward zero (architecture.md 3.5), the quantity the `Warn` "price beliefs at bound" tests. `economysim.household.relocations`, `economysim.household.births`, `economysim.household.deaths`, `economysim.household.own_use` and the migration instruments count households (the sum of weights), never agents. `economysim.hire.escrow` and the other money instruments count minor units. The credit instruments (`economysim.credit.*`) and the `Warn` lines of the credit module count only the claims whose `Origin` is `loan`: a claim of origin `payout` is an insurance claim, counted by the insurance instruments.

A new metric **MUST** be derivable from the existing events or from a field of
the snapshot, and the tables above say which, or come with the event or field it
needs. Do not add event fields only to avoid a lookup in the adapter.
The adapter measures durations with its own clock; events carry no wall-clock
time (4.1). Because observers run synchronously (4.4), a step duration includes
the time spent in every observer during the step.

### 5.4 Cardinality

5.4.1. Metric attributes **MUST** have a number of values bounded by the
configuration: `economysim.commodity`, `economysim.role`, `economysim.recipe`,
`economysim.side`, `economysim.want`, `economysim.region.from`, `economysim.region.to` (regions), `economysim.reason` and `economysim.kind` (the money flow IDs and the fiscal flow IDs, which share one namespace, and the built-in reasons, the goods boundary reasons, the rejection reasons of an order, the refusal reasons of a vacancy and of a rent request, the refusal reasons of a start and the abandon reasons of a job, and faucet or sink), `economysim.cause` (the closed sets of constants of the removal causes `insolvent`, `starved`, `consumer` and `population`, of the relocation causes `transport`, `migration` and `consumer`, and, per instrument, the closed set of the event that feeds it: `removal` and `consumer` for `economysim.title.transfers`, `repaid`, `written_off`, `debtor_removed` and `forgiven` for `economysim.credit.lost`, `insolvent`, `starved` and `consumer` for `economysim.household.exits`), `economysim.bound` (`floor`, `ceiling`), `economysim.belief_limit` (`conflict`, `clamped`), `economysim.work_decline` (`rest`, `no_recipe`), `economysim.entry_limit` (the nine constants `unfunded`, `no_viable_role`, `region_cap`, `role_cap`, `role_ceiling`, `cooldown`, `hysteresis`, `exit_cap`, `refused`), `economysim.holder` (`agents`, `estate`, `treasury`), `economysim.sector` (`households`, `pool`, `contract`, `treasury`), `economysim.mode` (transport modes), `economysim.resource` and `economysim.region` (resources and regions of the configuration, also on `economysim.market.clearing_price`, `economysim.market.orders`, `economysim.belief.centre`, `economysim.external.price` and the snapshot gauges), `economysim.node` (outside nodes), `economysim.outcome` (the four constants `offered`, `no_path`, `too_old`, `not_yet`), `economysim.limit` (the twelve constants of the migration outcomes: `no_estimate`, `no_destination`, `no_rumour`, `too_far`, `no_viable`, `no_gain`, `bound`, `out_cap`, `unprovisioned`, `in_cap`, `held`, `ended`) and `economysim.decision` (the eleven constants of the merchant's lane decisions: `no_rumour`, `stale`, `locked`, `perishable`, `no_margin`, `unaffordable`, `below_share`, `planned`, `rested`, `waiting`, `lot_capped`). The attributes of the later modules are bounded in the same way: `economysim.store_class`, `economysim.facility_class`, `economysim.body`, `economysim.right`, `economysim.control`, `economysim.field` and `economysim.actor` by the entities of the configuration, and `economysim.dimension` (`mass`, `bulk`), `economysim.room_limit` (`start_cut`, `start_refused`, `bid_cut`, `bid_rejected`, `shipment_refused`), `economysim.tier` (`own`, `commons`, `let`), `economysim.credit_limit` (`in_arrears`, `exposure`, `too_many_claims`, `overflow`), `economysim.facility_limit` (`cut`, `voided`), `economysim.risk_kind` (`route`, `store`), `economysim.clamp` (`cash`, `hold`, `stock`), `economysim.activity` (`produce`, `sell`, `buy`, `train`), `economysim.project` (`build`, `repair`), `economysim.bar` (`no_return`, `standing`, `unreachable`, `survival`), `economysim.entry_outcome` (`applied`, `unchanged`, `not_held`, `duplicate`, `no_close`), `economysim.policy_cut` (`clamped`, `stepped`), `economysim.policy_hold` (`in_band`, `no_reading`), `economysim.cover_kind` (`route`, `store`), `economysim.cover_loss` (`dropped`, `cut`) and `economysim.cover_limit` (`in_arrears`, `exposure`, `no_risk`, `no_offer`, `below_minimum`, `funds`, `too_many`, `overflow`) by their closed sets. Metric attributes **MUST NOT** include agent IDs, order IDs,
ticks, prices, quantities or any other unbounded or per-entity value. Put those
in logs, traces or events.

5.4.2. A new attribute on an existing metric multiplies its series count. Say
how many series it adds for the reference chain in the PR.

## 6. Traces

6.1. Traces are coarse. Create a span for each `Run`, from `RunStarted` to
`RunEnded`, and for each `Step`, from `TickStarted` to `TickEnded`, with
`economysim.tick` as a span attribute. A step inside a run is a child of the
run's span; a `Step` called directly is a root span. A span closes at its
`TickEnded` or `RunEnded`, and takes an error status when that event has
`Failed` true. **MUST NOT** create a span per order, trade or agent.
**[project]**

6.2. Rare notable occurrences within a step (a bankruptcy) are span events or
attributes on the step span, not child spans. An event that is per order or per
trade (a rejected order, a trade) is not a span event: the step span carries the
counts of orders placed, orders rejected and trades, and the traded volume, as
attributes.

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
  recipe, side, region, cause, reason) takes only values present in the
  configuration or in its closed set of constants. In
  `otelobserver`, a test asserts the attribute sets recorded on each
  instrument, using a configuration built in Go inside its own tests, because
  it cannot read the core module's `testdata/` (testing.md 10.2).
- **Run lifecycle**: `Run` emits `RunStarted` and `RunEnded`, including when it
  is cancelled or a step failed, and `RunEnded` reports the ticks actually
  completed.
- **Restore**: the snapshot of a restored world equals the snapshot of the
  original at the same boundary; an observer given to a restore receives
  `WorldRestored` first and nothing when the restore fails; the counts of agents
  and households and the money of the household sector and of the pool, rebuilt
  from `WorldRestored` and the events after it, equal the snapshot.
- **Counterpart**: every `Warn` has an event or a snapshot field that tells the
  same fact, and a test checks it.
- **Golden rendering**: the events of a scripted run, rendered in a fixed text
  form, equal a file in `testdata/golden/` (testing.md 7.1, 7.2). A field at its
  zero value is omitted from the rendering, so a field a module appends leaves
  the files of `testdata/golden/off/` unchanged (`spec-observability.md` 9.5.2).
- **Adapter calls**: in `otelobserver`, each series is recorded once per tick,
  however many orders and trades the tick had.
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
   for users' dashboards once there is a release (5.3). Is it versioned and
   called out in the changelog?
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

RULE-TAG: f3926b9ad4ec89afcd72eb54
