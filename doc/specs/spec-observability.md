# Spec: Observability: the event catalogue, the snapshot, metrics, logs, the cost of observing and how all of it is tested

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (the rules win on any
conflict, and `rules/observability-and-logging.md` wins over `rules/testing.md`
for tests of observability behaviour). Its requirements carry IDs of the form
`FR-OBS-NNN` and `NFR-OBS-NNN`, which tests cite (testing.md section 4). They
are in section 17.

Layer: core. It is the design behind `rules/observability-and-logging.md` 4.3
(the event table), 3.3.2 (the log keys) and 5.3 (the metric catalogue): where the
rules list an event with its main fields in parentheses, this spec fixes every
field, its type and its place in the tick. It reads every spec of tiers 1 and 2
and `spec-world.md`, takes each event's fields from the spec that owns the
transition, and changes none of their rules; where two texts disagree it rules and
names the text that wins (section 4.5), and section 19 lists what must still be
edited elsewhere. It does **not** edit the rules: the exact edits are in section
19, for the supervisor to apply with fresh read-receipt tags.

Stage: this spec is **in the slice**, except its metric adapter. The events
and their observer, the options `WithLogger` and `WithObserver`, the log keys and
the golden rendering are built from Stage 0 (`doc/roadmap-v1.md`), the snapshot
and the world-dependent tests of section 9 in Stage 1, and `otelobserver/` in
Stage 2 (it was "version 1" in `vision.md` section 4 before the roadmap of the
code was written; the move is recorded there). The events are built in stages
too: each module's change declares its own events and fields, and the catalogue
test of 9.1.2 is staged to match (3.3.4, FR-OBS-003).

This spec adds **no `Config` field and no default** (section 11). The only numbers
in it are rule constants (the run length of a bounded `Warn`, 7.3) and the example
numbers of sections 8 and 16, which are labelled placeholders (N-015, N-027).
Nothing has run in the library, and no model was run for this spec: the one place
that needs a number (the events of a tick of the first consumer's world, 8.3) is
arithmetic on the placeholder sizes of `spec-scale.md` 6.3, shown in full. The
worked examples (16) use invented words (`alpha`, `grain`, `baker`), flagged as
examples, which are data and never branched on (`AGENTS.md`, N-017).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

The world decides what happens in a tick; this spec decides what the world tells
the outside about it, and how little that may cost. It covers:

- the **event catalogue**: every event of the library as one table, with its
  fields, its owner, its position in the tick, its volume class and the module that
  gates it (section 4);
- the **snapshot**: the plain data a consumer reads between ticks, with every
  field's name, type and unit, and a guide to what each costs (section 5);
- the **metric catalogue** and the adapter that derives it from events and
  snapshots (section 6);
- **logs**: every `Warn` and `Info` path of the library, its keys, its reason and
  the bound on its volume, and every log key against the rules (section 7);
- **the cost of observing**: what the world records at construction, what an
  event costs to build, what the snapshot costs against `spec-scale.md` 6
  (section 8);
- **how it is tested**: the tests of `rules/observability-and-logging.md` 8 as
  requirements, and the golden rendering of events (section 9);
- **what OBS hands** to IND, STY, EXP, CHK and INT (section 10).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| Whether a transition happens, and the rule behind an event's numbers | the spec that owns the transition (section 4 names it in each row) |
| The position of an event in the tick, and the order of the events of a step | `spec-world.md` 6 and 6.11 (this spec checks them, 4.5, and asks for two new positions and one order) |
| The encoding, version and restore of a checkpoint; the state a checkpoint holds | `spec-checkpoint.md` (CHK). The snapshot is **not** the checkpoint (5.7) |
| The public names, errors and bounds of the calls a consumer makes between ticks, and of the per-agent query | `spec-integration.md` (INT); this spec defines the event each call emits and the record a query returns |
| The OpenTelemetry instruments' implementation, the span code, the SDK wiring | the separate module `otelobserver`; this spec says what it must keep and must not keep (6.6) |
| The measures a consumer derives (GDP, price indices, inequality) | `spec-indicators.md` (IND), built on 4 and 5 |
| Acceptance bands, scenarios | STY, EXP; `spec-reference-chain.md` 9 lists what the scenarios need of this spec |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | There are three outputs and each fact goes to one: a **fact that happened** is an event, a **state at a boundary** is a snapshot field, a **thing for a human** is a log line. A metric is derived, never a fourth source of truth | 3.1 |
| 2 | One catalogue of 89 events: the 47 of the first rules, `WorldRestored`, `ProductionHalted`, `YieldShocked`, `TransitFactorChanged`, `PopulationScaled` and `WearApplied` (53), and the 36 of the fourteen later-stage modules (4.1.9 to 4.1.12 and the groups they name). Every event has one owner, one position, one volume class and one module gate | 4 |
| 3 | An event carries the **typed integer ID** of every configured entity (`RegionID`, `CommodityID`, `RoleID`, ...) and the agent ID; a reason, a cause and a side are constants of a closed set. The strings come from one **names table**, carried by `WorldCreated`, `WorldRestored` and the snapshot, and by `World.Config()`. This answers REG open question 4, RES 14.1 and N-012 | 3.2 |
| 4 | An event of a transition of one agent carries the agent's region (and role where the metric needs it) so that a consumer with no agent table can attribute it | 3.3 |
| 5 | `BoundaryFlow` is only for the reasons whose kind is faucet or sink. The internal reasons (`trade`, `relief`, `start_money`, `estate_*`, `escrow_*`, `transit_*`, the weight reasons) have their own events or are in the snapshot | 4.4.1 |
| 6 | The construction bucket is a `BoundaryFlow` with `Tick` -1, emitted at the entry of the first step of a world still in its construction phase, before `TickStarted` (answers N-072) | 4.4.2 |
| 7 | `TickEnded.Failed` and `RunEnded.Failed` are accepted. `TickEnded` carries nothing else, `RunEnded` carries the ticks completed and whether it was cancelled or failed | 4.4.3 |
| 8 | Three slots get an event (`YieldShocked`, `TransitFactorChanged`, `PopulationScaled`) and a restore gets `WorldRestored`. A directive, a season and a net trade get none, with reasons | 4.4.4 to 4.4.8 |
| 9 | `WorldRestored` carries the whole snapshot at the restored boundary, so that an adapter built after a restore can seed the instruments that are fed by deltas | 4.4.5, 6.4 |
| 10 | The snapshot is a **deep copy of aggregates**, bounded by the configuration and by what is in flight, and never by the number of agents. Per-agent data is a record returned by a query that INT names | 5.1, 5.6 |
| 11 | Agents and households are reported separately everywhere, pseudo-agents are never in either, and a count of households is the sum of weights | 5.1 |
| 12 | A metric is fed by an event or by a snapshot that the consumer passes to the adapter between steps. The metrics that are fed by a snapshot are named as such, and the rule that every metric is derivable from events is restated to say so | 6.2, 6.3 |
| 13 | `economysim.agent.bankruptcies` is renamed `economysim.agent.removals` and gains the attribute `economysim.cause` (answers N-088 4) | 6.3 |
| 14 | Every `Warn` is bounded by an entity of the configuration (a market, a role, a region, an endowment, a body, a side or, row 19, with no run, a recipe), and repeats on every tenth consecutive tick of its condition, and pattern A also on the first; the `10` is one rule constant. There is no `Error` path in the library | 7 |
| 15 | With no observer the world constructs no event; the cost with an observer is counted per event class, and the per-order and per-trade classes are 92% to 99% of the events | 8 |
| 16 | The golden rendering of events is defined here, in text, field by field in declaration order; the renderer is test code | 9.5 |

## 3. Principles

### 3.1 Which output carries which fact

The rules fix the channels (`rules/observability-and-logging.md` 2). This section
adds the test that decides between the three outputs of the core for a new fact.

3.1.1. **An event** is a transition that happened, at a point of a tick, to be
counted, graphed or traced. It exists once and is delivered once. A consumer that
missed it cannot recover it, so an event is the right form for something that
the consumer must have seen to make its numbers right (a trade, a removal, a
faucet).

3.1.2. **A snapshot field** is a state at a boundary: a stock, a level, a count of
things that exist. It can be asked for at any boundary and costs nothing to
anyone who does not ask. It is the right form for a quantity that is the *sum
of past events* and that would otherwise have to be rebuilt by the consumer from
the first event of the run (the money held by the households, the pending output
of the jobs in progress).

3.1.3. **A log line** is for a human who is debugging or operating. It is never the
source of a metric (rules 2.1).

3.1.4. **The rule.** If a consumer could not compute a value from the events it has
seen plus the snapshot it asked for, the value is missing from one of them, and it
is added to the one that costs less *for the consumer who does not use it*: a
snapshot field costs nothing when not asked for, an event costs a construction
and a call per occurrence for every consumer with an observer. So **a state goes
in the snapshot, a transition goes in an event, and an event is never added only
to spare a consumer a lookup** (rules 5.3.1).

3.1.5. **No fact is in two places by design**, but a count that a consumer can
derive from events is also kept in the snapshot when it would otherwise need the
whole history (the household count: the events of 4.3 rebuild it, and the
snapshot gives it directly). The two must agree at every boundary, and a test
asserts it (FR-OBS-040).

### 3.2 Identifiers: an index in the event, a string in the names table

3.2.1. **Decision (REG open question 4, RES 14.1, INF 10.1, MIG 13.1, N-012).**
Every field of an event or a snapshot row that names a configured entity carries
the **typed integer ID** that the owning spec defines: `RegionID` (the index in the
canonical region order, `spec-regions.md` 3.3), `CommodityID`, `RoleID`,
`RecipeID`, `ResourceID`, `ModeID` and `WantID`, each an index in the
canonical order of its kind (`spec-config.md` 3.4, 3.5; `WantID` is a type this
spec asks `spec-demand.md` to name, section 19), a season index (`Moment.Season`,
-1 for none), and `AgentID` (an `int64`, never reused). The later-stage modules add
the typed IDs `StoreClassID` (STO), `SettlementClassID` (SET), `FacilityClassID`
(CAP), `TechFieldID` (TEC), `JurisdictionID` (POL), `BodyID`, `RightID` and
`ControlID` (ORG) and `ActorID` (PLC), each an index in the canonical order of its
kind, and a claim, a hire, a cover policy and a facility are named by the ID that
its owner defines (`HireID`, `FacilityID`; a claim and a cover policy are `int64`
counters of the world, as a shipment is, 3.2.5). A **money
flow** and a **fiscal flow** are the exception, as in `spec-integration.md` 3.4.1:
each is named by its string ID, which is also its ledger reason
(`spec-money-supply.md` 4.5; the two kinds share one ID namespace), and has no typed
ID. A **reason**, a **cause**, a **kind**
(faucet or sink) and every other member of a closed set of constants is a named
string type (`Reason`, `Cause`, `Kind`, `ScaleKind`) **exported from the root
package** (`event.go`) with its constants, so that a consumer and the separate
`otelobserver` module can name them (architecture.md 2.9: domain types are the
root's). `Side` is a root type declared in `event.go`, beside the events that carry it
(`Bid`, `Ask`; AGT's `Intent` and `Order` use it, `spec-agents.md` 6.3, and it is
in the root package whatever file declares it). It has a `String` method for the
golden rendering (`bid`, `ask`). `internal/logkeys`
holds the **key strings** of the logs only and imports nothing: it never aliases
or imports a root type (architecture.md 2.9); the root owns the types and the
constants and passes their values to the log calls.

3.2.2. **Why an index.** Three reasons. An event is built in the hot path, and an
integer field is a word where a string is two and a pointer into the
configuration; a consumer keys an array by it; and the cardinality test of
section 9.4 is a range check (`0 <= id < len(Names.X)`). The cost the choice puts
on the consumer is one table, which it already holds (3.2.4).

3.2.3. **Why not the string.** A string would survive a reordering of the
configuration; an index does not. That is not a defect: an index depends on the
configuration exactly as an agent ID does (`spec-world.md` 5.3.2), a different
configuration has a different hash, and no checkpoint restores across it. A
consumer that needs identity across configurations (a long-lived dashboard, a
database row) maps through the names table at the edge, as WLD 5.3.2 already tells
it to do for roles, regions and commodities.

3.2.4. **The names table.** One plain-data value, the same in every place it
appears:

```go
// Names maps each typed ID of a world to the string ID of its configuration.
// The slice of a kind is indexed by that kind's ID type. Plain data: a deep copy.
type Names struct {
	Regions, Commodities, Roles, Recipes, Resources, Modes, Wants, Seasons []string
	// Kinds of the later-stage modules; each is empty with its module off.
	StoreClasses, SettlementClasses, FacilityClasses, TechFields []string
	Jurisdictions, Actors, Bodies, Rights, Controls              []string
}
```

The slice of a later module's kind is named by the plural of the ID type, one field
for each kind (the settlement classes are not the commodity classes of IND, so the
field says `SettlementClasses`), and it holds the string of every typed ID of the
module in the canonical order of the module's own data (STO, SET, CAP, TEC, POL,
PLC, ORG). A fiscal flow and a money flow have no slice: each is its own string.
It is carried by `WorldCreated`, by `WorldRestored` and by the `Snapshot`, and is
what `Config()` would give by walking its canonical order. It is built once at
construction (it is derived from the configuration, 8.1) and copied into each
event that carries it, so the cost is paid once per world and once per snapshot.
An adapter that sees `WorldCreated` or `WorldRestored` holds the table and never
needs the configuration (6.6.1).

3.2.5. **What is not an index.** The shipment ID and the contract ID are `int64`
counters of the world, as are the claim ID, the hire ID, the cover policy ID and the
facility ID of the later modules. The ledger's `Reason` is the string of its registry, and
for a money flow, and for a fiscal flow, it equals the flow's string ID (`spec-money-supply.md` 4.5), so
`BoundaryFlow.Reason` joins to the `Flow` field of `MoneyFlowLimited`, `MoneyFlowEnded`
and `MoneyFlowScaled` by equality, with no table. An outside node has a `RegionID` like any region (`spec-regions.md` 3.3):
`OutsideTraded` carries it in `Node` and the gate in `Region`, and the `Region`
of a `BoundaryFlow` under an `outside_*` reason is the gate.

### 3.3 What an event is

These rules hold for every event of section 4. Rules 3.3.1 to 3.3.4 restate
`rules/observability-and-logging.md` 4.1; 3.3.5 to 3.3.9 are new.

3.3.1. An event is a plain struct of the root package that implements the sealed
`Event` interface (architecture.md 4.8), is passed **by value**, and has an
exported `Tick int64` field first.

3.3.2. No pointer into live state, no function, no map, no wall-clock time, no
float. Money, prices and quantities use the fixed-point types of architecture.md
3.5. A slice in an event is a copy in a deterministic order, **owned by the
event**: the library never reuses its backing array, so an observer may keep the
event (a recorder does).

3.3.3. An event is emitted **after** the transition it describes is complete, in
the order of `spec-world.md` 6.11, and **only when an observer was supplied**
(rules 4.4, 4.5). A module that is off emits none of its events; a module that is
on and has nothing to do emits none either, and each aggregate says when it is
silent (the "never when all are zero" of the rules).

3.3.4. Adding a field is allowed; removing or renaming one, or changing its
meaning, is a breaking change (rules 4.2). Once code exists a new field goes at the end of its
event, so that a positional rendering of an older recording is a prefix. In this
draft the fields marked `+` in 4.1 sit beside the identity they belong with
(`Region` after `Agent`), because no recording exists yet; the golden rendering
follows the declaration order of the code, whatever it is.

**Staged form (decision D3 of the plan of Stage 0).** The table of 4.1 is the
catalogue of the finished library, and the code is built in stages
(`doc/roadmap-v1.md`), so an event declares the fields the specs give the core,
that is every field no module of `Config.Modules` owns, and each module's change
appends its own events and its own fields to the end of an existing event, in
module order (architecture.md 2.10.5). The declaration order of a built event is
therefore the order of its row in 4.1, less the fields not built yet. A module's
change moves its fields to the end of the row in 4.1 in the same change, so that
from then on the row order is the declaration order. An appended field is a
count, an amount, a bool, a constant of a closed set or a slice, never a typed
ID, so that a world with the module off renders as it did before (9.5.2,
FR-OBS-091). The catalogue test of 9.1.2 is the form of architecture.md 9.7: it
checks the events and fields that are built, and one explicit `notBuilt` list
names the rest (FR-OBS-003, FR-OBS-004).

3.3.5. **`Tick` is the tick the event belongs to.** For an event of a step it is
the tick that step runs. For an event emitted by a call between ticks it is the
tick the next step will run (`spec-world.md` 8.1). For an event of construction
it is 0. The one exception is the construction bucket (4.4.2), `Tick` -1, which comes
after events of `Tick` 0 (`WorldCreated`, the initial `AgentSpawned`, a consumer's
calls before the step): **the stream is not monotone in `Tick`**, and an observer
must not assume it is. A
restored world's `WorldRestored` carries the tick it restored to.

3.3.6. **Counts of households and counts of agents are different fields and are
named so.** A field named `Households` (or any count of what an agent stands for)
is the sum of weights; a field named `Agents` is a count of agents. An event never
has a bare `Count`. Pseudo-agents are in neither (`spec-estates.md` 2.7.4).

3.3.7. **A transition of one agent carries the agent's region at the time of the
event, and its role where a metric or a count by role needs it**, so that an
observer that keeps no agent table can attribute the event. This is a field of
the event, never a reference to look up. It is why `RecipeExecuted`,
`RecipeAbandoned`, the three `Traineeship*` events and `ContractSettled` gain a
`Region` here (4.1); for `ContractSettled` it is the region of the deliverer's
account at the event.

3.3.8. **Signed amounts** (a net change, a delta) are signed `int64`; amounts that
are magnitudes are non-negative. A field that is "absent" has a companion
`bool` (`HasBestBid`), never a sentinel value of the quantity, except where a
sentinel is already a named constant of an ID type (`NoCommodity` for money in
`BoundaryFlow`, 4.4.1).

3.3.9. **Volume classes** say what the number of events of a kind is bounded by.
They are the only guide a consumer has to the cost of an observer (8):

| Class | Code | Bounded by | Examples |
|-------|------|------------|----------|
| world or run | W | once per world, run or step | `WorldCreated`, `RunStarted`, `TickStarted` |
| aggregate | A | the configuration: regions, markets, roles, wants, flows, resources, nodes, reasons, facility classes, bodies, rights, controls, technology fields, authorities, policy actors, never agents, orders or trades | `MarketCleared`, `UpkeepShortfall`, `BoundaryFlow`, `EndowmentChanged` |
| entity | E | one transition of one agent, shipment, contract, journey, edge, flow, claim, hire, title, facility, store, cover policy or region (a settlement class), however many there are | `AgentSpawned`, `AgentBankrupted`, `ShipmentDispatched`, `ContractFormed` |
| order | O | one per order | `OrderPlaced`, `OrderRejected` |
| trade | R | one per trade | `TradeExecuted` |
| call | C | one per changed slot cell, at a call between ticks, at P0.3a when a policy actor writes it (cause `policy`), or at P8.3b for a facility's change of an endowment (at most one per endowment, bounded by regions times resources) | `YieldShocked`, `EdgeStatusChanged`, `OutsideShocked` |

The classes O and R are the volume of the library (8.3): they grow with the
number of agents and the number of tradable commodities, and every other class is
small beside them.

## 4. The event catalogue

### 4.1 The table

Every event of the library. **Position** is the position of `spec-world.md` 6
that emits it (a call between ticks is "call"). **Class** is 3.3.9. **Gate** is
the module whose absence silences it (ADR 0001: modules are off unless
configured); "core" means no module gates it. **Owner** is the spec whose section
fixes the transition and, with this table, the fields. A field marked `+` is one
this spec adds to the owner's list (the edit is in section 19). Types: `Money`,
`Price` and `Quantity` are architecture.md 3.3's, every count and every
basis-point value is `int64`, `Region` is a `RegionID`, an agent is an `AgentID`.
`Amount` is `{Commodity CommodityID; Quantity Quantity}`.

#### 4.1.1 Lifecycle

| Event | Owner | Position | Class | Gate | Fields (besides `Tick`) |
|-------|-------|----------|-------|------|-------------------------|
| `WorldCreated` | WLD 16.1, CFG 10.3 | construction, first (WLD 5.8.2) | W | core | `Seed uint64`, `ConfigHash [32]byte`, `WarmStart bool`, `Names Names`+, `Jurisdictions []JurisdictionRow`+ (one `JurisdictionRow{Jurisdiction; Seat; Agent; Money}` for each governed jurisdiction, `Money` its `StartMoney`; empty with none, POL 10.7) |
| `WorldRestored` + | OBS 4.4.5; CHK 6.7.3, 14.1 | the restore, first | W | core | `ConfigHash [32]byte`, `Format uint32` (the checkpoint's, as `CheckpointInfo.Format`; the log key is `slog.Int64(int64(f))`), `Reseeded bool`, `Salt uint64` (meaningful when `Reseeded`), `Names Names`, `Snapshot Snapshot` (the state at the restored boundary; its header holds the original seed, the agents and the households) |
| `RunStarted` | WLD 3.4, 16.1 | `Run`, before its first tick | W | core | `Ticks int64` (requested) |
| `RunEnded` | WLD 3.4.2, 16.1 | `Run`, before it returns | W | core | `Completed int64`, `Cancelled bool`, `Failed bool`+ |
| `TickStarted` | WLD 6.2 | P0.3 | W | core | none |
| `TickEnded` | WLD 3.5.6, 6.10 | P8.5, and at the failure of P4.1 | W | core | `Failed bool`+ |

#### 4.1.2 Market and orders

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `OrderPlaced` | CLR 9 | P3.2(c), P3.4, P3.5 | O | core | `Region`, `Commodity`, `Agent`, `Side`, `Limit Price`, `Quantity` |
| `OrderRejected` | CLR 9, 3.2 | P3.2(c), P3.4, P3.5 | O | core | `Region`, `Commodity`, `Agent`, `Side`, `Reason` |
| `TradeExecuted` | CLR 9 | P4.1 | R | core | `Region`, `Commodity`, `Buyer`, `Seller`, `Price`, `Quantity` |
| `MarketCleared` | CLR 9, 5.4 | P4.1, once per market, also with no trade | A | core | `Region`, `Commodity`, `Price` (the published price), `PriceAge int64`, `Volume Quantity`, `UnmatchedBid Quantity`, `UnmatchedAsk Quantity`, `BestBid Price`, `BestAsk Price`, `HasBestBid bool`, `HasBestAsk bool` |
| `BeliefsUpdated` | BEL 12.1 | P5.4, after the last outcome of the market | A | core | `Region`, `Commodity`, `Outcomes`, `Adjusted`, `Raised`, `Lowered`, `AtFloor`, `AtCeiling`, `Conflicts`, `Clamped`, `MeanCentre Price`, `LowCentre Price`, `HighCentre Price` |

#### 4.1.3 Production, work and wants

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `RecipeExecuted` | PRO 14.1 | P2.1, P2.7 | E | core | `Agent`, `Region`+, `Recipe`, `StartTick int64`, `Executions`, `Outputs []Amount`, `Disposed []Amount`, `EffectiveYield` (bp) |
| `RecipeAbandoned` | PRO 14.1 | P7.1, P7.2, a removal or a `Relocate` (`region_changed`) at a call | E | core | `Agent`, `Region`+, `Recipe`, `Executions`, `Reason` |
| `ProductionHalted` + | OBS 4.4.6; AGT 7.1.5 | P2.8a (new) | A | core | `Region`, `Role`, `Free`, `Halted`, `Skipped []HaltRow` with `HaltRow{Recipe RecipeID; Households int64}` |
| `OwnUseChanged` | SUB 14.1 | P2.8 | A | subsistence | `Region`, `Role`, `Households`, `OnOwnUse`, `Entered`, `Left`, `Started`, `Blocked`, `Retried`, `Output []Amount` |
| `UpkeepShortfall` | AGT 14.1 | P6.4 | A | core | `Region`, `Role`, `Households`, `Unfed`, `Agents`, `MaxHunger` |
| `WantShortfall` | DEM 11.1 | P3.6 | A | demand | `Region`, `Role`, `Want`, `Households`, `Choked`, `Planned`, `Funded` |
| `RoomLimited` | STO 14.1 | P3.6a, after `WantShortfall`, region then role ascending; never when `StartsCut`, `StartsRefused`, `BidsCut`, `BidsRejected`, `ShipmentsRefused` and `Discarded` are all 0 | A | storage | `Region`, `Role`, `LimitedHouseholds`, `FullHouseholds` (both ride along and do not trigger it), `StartsCut`, `StartsRefused`, `BidsCut`, `BidsRejected`, `ShipmentsRefused`, `Discarded` |

#### 4.1.4 Agents, weight, roles, entry and exit

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `AgentSpawned` | AGT 14.1, SPN 13.1 | construction after `WorldCreated`; P7.1(d); call | E | core | `Agent`, `Region`, `Role`, `Weight`, `StartMoney Money`, `External bool`, `StoresLocked` (the stores of the role row not given because the region's class lacks the flag, STO 14.1; 0 with the module off) |
| `AgentBankrupted` | AGT 14.1, SPN 13.1 | P7.1(d); call; the one event that removes the agent (4.4.8) | E | core | `Agent`, `Region`, `Role`, `Weight` (at removal), `Cause`, `Money Money`, `GoodsListed int64` |
| `EstateOpened` | EST 2.5 | with the removal, after `AgentBankrupted` (4.4.8) | E | core | `Agent`, `Region`, `Balance Money`, `GoodsListed int64` |
| `WeightChanged` | SPN 13.1 | P7.1(d) (no consumer call changes a weight, WLD 8.3) | E | core | `Agent`, `Region`, `Role`, `From`, `To` (weights), `Cause`, `Money Money`+ (signed: positive to the agent), `Units int64`+ (goods moved to the estate) |
| `RoleChanged` | SPN 13.1 | P7.1(a), (d) | E | core | `Agent`, `Region`, `From`, `To` (roles), `Weight`, `Cause`, `GoodsListed int64` |
| `EntryLimited` | SPN 13.1 | P7.1, end of the region's step | A | core | `Region`, `Changed`, `Entered`, `Removed`, `Unfunded`, `NoViableRole`, `RegionCap`, `RoleCap`, `RoleCeiling`+ (a refusal by the class ceiling of a role, SET 6.4), `Cooldown`, `Hysteresis`, `ExitCap`, `Refused` |
| `PopulationChanged` | POP 10.1 | P7.1, after `EntryLimited`, roles ascending | A | population | `Region`, `Role`, `Exposed`, `Unfed`, `NoRatio`, `BirthsDue`, `DeathsDue`, `DeathsCut`, `DeathsHeld`, `BirthsCapped`, `BirthsCeiling`+ (births cut by a role ceiling), `BirthsUnfunded`, `RateClamped`, `BirthsDone`, `DeathsDone` |
| `TraineeshipStarted` | SKL 4.7 | P7.1(a) | E | skill barriers | `Trainee`, `Trainer`, `Role`, `Region`+ |
| `TraineeshipCompleted` | SKL 4.7 | P7.1(a) | E | skill barriers | `Agent`, `Region`+, `From`, `To` (roles) |
| `TraineeshipEnded` | SKL 4.7, 8.1 | with a removal (4.4.8) | E | skill barriers | `Agent`, `Region`+, `Reason` |
| `StoreChanged` | STO 14.1 | a call (`AddStore`, `RemoveStore`); P7.1 after the `RoleChanged` that ended the store; P8.3b for a store facility | E | storage | `Agent`, `Region`, `Role`, `Class StoreClassID`, `Kind` (`added`, `removed`), `Cause` (`consumer`, `role_changed`, `facility`), `Stores` (the agent's stores of the class after), `Room`, `Load` (the account's, as pairs, after) |

#### 4.1.5 Money, estates, boundary

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `BoundaryFlow` | LED 12.1 | P8.2; and P0.2a with `Tick` -1 (4.4.2) | A | core | `Region`, `Reason`, `Kind`, `Commodity` (`NoCommodity` for money), `Total int64` |
| `WearApplied` | LED 12.1, N-130 | P8.2, after the `BoundaryFlow` rows, ascending commodity; and P0.2a with `Tick` -1, after the construction `BoundaryFlow` rows (4.4.2) | A | core | `Commodity`, `Units int64` (the durability removed from the commodity's items, at least 1) |
| `ReliefPaid` | EST 2.6, FR-EST-020 | P1.6 | A | estates (pool policy) | `Region`, `Paid Money`, `Recipients int64` (agents), `Households int64`+ |
| `MoneyFlowLimited` | MON 13.1 | P1.7, after the step, flow order | A | money supply | `Flow string`, `Region`, `Due Money`, `Applied Money`, `Short`, `Headroom`, `Limit` (households) |
| `MoneyFlowEnded` | MON 13.1 | P1.7, after `MoneyFlowLimited` of the same flow | E | money supply | `Flow string`, `Cause`, `Total Money` |
| `MoneyFlowScaled` | MON 13.1; PLC 12.1.2 | call; P0.3a when a policy actor writes it | C | money supply (or fiscal, for a fiscal flow) | `Flow string`, `From`, `To` (bp), `Cause`+ (`consumer`, `policy`) |
| `StoreLoss` | RSK 14.1 | P8.1a, ascending agent ID then class, after the whole position is applied | E | risk | `Agent`, `Region`, `Class StoreClassID`, `Stores` (stores of the class struck, at least 1), `Common bool` (the row's flag), `ShareBP` (after the cap), `Lost []Amount` (a copy, in commodity order; empty when every product rounded to 0) |

#### 4.1.6 Regions, transport, contracts, migration

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `AgentRelocated` | REG 10.1 | P1.3, P1.5, call | E | core | `Agent`, `Role`, `From`, `To` (regions), `Weight`, `Money Money`, `Units int64`, `Cause` |
| `EdgeStatusChanged` | REG 10.1 | call, edge order, after the graph is in place; P0.3a for a policy actor's write (cause `policy`) | C | core | `From`, `To` (regions), `Closed bool`, `FactorBP`, `Cause` (gains `policy`) |
| `ShipmentDispatched` | TRN 13.1 | P3.2(c), P3.3 | E | transport | `Shipment`, `Mode`, `Region` (origin), `To`, `Shipper`, `Traveller`, `Hired bool`, `Consignee`, `Arrives int64`, `Units`, `HaulUnits`, `Lines []Amount`, `Toll Money`+ (the total toll of the shipment, 0 when none, FIS 16.1), `LossBP`+ (the loss share fixed at dispatch, 0 for no strike or the module off, RSK 14.1) |
| `ShipmentArrived` | TRN 13.1 | P1.3 | E | transport | `Shipment`, `Mode`, `Region`, `To`, `Receiver`, `Diverted bool`, `Lines []ArrivedLine` with `ArrivedLine{Commodity; Shipped, Delivered, Diverted, Spoiled, Lost Quantity}` (`Lost`+: RSK 14.1), `AccountLost []Amount`+ (the loss of the traveller's own account, in commodity order; empty for a hired shipment) |
| `ShipmentRejected` | TRN 13.1 | P3.2(c), P3.3 | E | transport | `Region`, `Agent`, `Mode`, `To`, `Reason` |
| `TransitFactorChanged` + | OBS 4.4.4; TRN 10.3 | call, (`From`, `To`, `Mode`) order; P0.3a for a policy actor's write | C | transport | `From`, `To` (regions), `Mode`, `BeforeBP`, `AfterBP`, `Cause`+ (`consumer`, `policy`; PLC 12.1.2) |
| `ContractFormed` | MER 15.1.1 | P3.3, formation order, after the contract's `ShipmentDispatched` | E | merchants | `Contract`, `Payer`, `Deliverer`, `Region`, `To`, `Commodity`, `Quantity`, `Price`, `Escrowed Money`, `Shipment`, `Mode`, `Arrives` |
| `ContractSettled` | MER 15.1.1 | P1.4; with a removal of the deliverer (4.4.8) | E | merchants | `Contract`, `Payer`, `Deliverer`, `Region`+ (the deliverer's account at the event), `To`, `Commodity`, `Quantity`, `Reached`, `Spoiled`, `Lost`+ (units lost in transit, RSK 14.1; `Reached = Quantity - Spoiled - Lost`), `Paid Money`, `Refunded Money`, `Cause` |
| `ContractRound` | MER 15.1.1 | P3.3, destination region ascending | A | merchants | `Region`, `Offers`, `Takes`, `OffersRefused`, `TakesRefused`, `Matched`, `Formed`, `NoTaker`, `PayerFunds`, `TooMany`, `DispatchRefused`, `Units`, `Value Money` |
| `MerchantsPlanned` | MER 15.1.1, WLD P3.3a | P3.3a, region ascending | A | merchant block | `Region`, `Merchants`, `Considered`, `NoRumour`, `Stale`, `Locked`, `Perishable`, `NoMargin`, `Unaffordable`, `BelowShare`, `Planned`, `Dispatched`, `Offered`, `Taken`, `Rested`, `Waiting`, `LotCapped` |
| `MigrationStarted` | MIG 13.1 | P7.2, admission order | E | migration | `Agent`, `Role`, `From`, `To`, `Weight`, `Arrives int64`, `Distance int64`, `Journey int64`, `EarnedRatio` (bp, signed), `Home`, `Estimate` (bp), `JobAbandoned bool` |
| `MigrationReviewed` | MIG 13.1 | P7.2, region ascending, after the departures | A | migration | `Region`, `Unsettled`, `Reviewed`, `Pushed`, `Started`, `NoEstimate`, `NoDestination`, `NoRumour`, `TooFar`, `NoViable`, `NoGain`, `Bound`, `OutCap`, `Unprovisioned`, `InCap`, `Held`, `Ended` |

#### 4.1.7 Resources, information, the outside

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `EndowmentChanged` | RES 14.1 | P8.3, (`Region`, `Resource`) order | A | resources | `Region`, `Resource`, `Stock`, `Capacity`, `AbundanceBP`, `Requested`, `Granted`, `Drawn`, `Short`, `Regenerated`, `Lost`, `AgentsDenied`, `AgentsReduced` |
| `EndowmentAdjusted` | RES 14.1; CAP 14.1 | call; P8.3b for a facility's change (`Kind` `facility`) | C | resources | `Region`, `Resource`, `Kind` (`adjust`, `shock`, `facility`), `Delta` (signed), `CapacityBP`, `RegenBP`, `StockBefore`, `StockAfter`, `Lost` |
| `RumoursServed` | INF 10.1 | P3.6, region ascending | A | information | `Region`, `Readers`, `Offered`, `NoPath`, `TooOld`, `NotYet`, `MaxLag` |
| `OutsideTraded` | EXT 15.1 | P4.2, (`Node`, gate, `Commodity`, buy before sell) | A | external | `Node`, `Region` (the gate), `Commodity`, `Side`, `Mid Price`, `Price` (the limit), `Wanted`, `Offered`, `Traded`, `Value Money`, `ListValue Money`, `CutClosed`, `CutCap`, `CutPrice`, `PriceClamped bool` |
| `OutsideEnded` | EXT 15.1 | P4.2, after `OutsideTraded` | E | external | `Node`, `Commodity`, `Side`, `Cause`, `Units`, `ListValue Money` |
| `OutsideShocked` | EXT 15.1 | call | C | external | `Node`, `Commodity`, `From`, `To` (bp) |

#### 4.1.8 Slots that had no event

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `YieldShocked` + | OBS 4.4.4; PRO 6.8 | call, (`Region`, `Recipe`) order | C | core (the slot is core data) | `Region`, `Recipe`, `BeforeBP`, `AfterBP` |
| `PopulationScaled` + | OBS 4.4.4; POP 3.6, 6.6 | call, region order; one event per changed key, so the call of births and the call of deaths each emit theirs | C | population | `Region`, `Which ScaleKind` (`birth` or `death`), `BeforeBP`, `AfterBP` |

| `RouteRiskScaled` | RSK 14.1 | call, ascending (`From`, `To`, `Mode`) | C | risk | `From`, `To` (regions), `Mode`, `BeforeBP`, `AfterBP` |
| `StoreRiskScaled` | RSK 14.1 | call, ascending (`Region`, `Class`) | C | risk | `Region`, `Class StoreClassID`, `BeforeBP`, `AfterBP` |

The two risk scales are the slot events of their module and sit here because RSK
asks for this group (the other slot events of the later modules are in 4.1.9 to
4.1.12, beside the module that owns the slot).

#### 4.1.9 Employment

Gate: `Modules.Employment` (a role with the block). The fields are EMP 15.1's.

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `HireFormed` | EMP 15.1 | P2.5a, formation order | E | employment | `Hire`, `Region`, `Employer`, `Worker`, `EmployerRole`, `WorkerRole`, `Recipe`, `Households` (the executions), `Wage Money` (per household-tick), `Ticks int64` (the job's duration), `Escrowed Money`, `Reserve Price`, `Bid Price` |
| `HireEnded` | EMP 15.1, 6.5.2 | P2.1, P2.7; P7.1, P7.2 or a call, in the order of 4.4.8 (a removal: 2c, the worker's side and then the employer's) | E | employment | `Hire`, `Region` (the employer's at the event), `Employer`, `Worker`, `Recipe`, `Households`, `Worked int64` (the working ticks paid), `Paid Money`, `Refunded Money` (the remainder of the escrow at the end; the refunds of a cut start are in `LabourRound`), `Cause` (`completed`, `worker_left`, `worker_role`, `worker_moved`, `employer_left`, `employer_role`, `not_started`) |
| `LabourRound` | EMP 15.1 | P2.8b, region ascending, after `ProductionHalted`; once per region per tick in which any offer, vacancy or hire in progress was counted, never otherwise | A | employment | `Region`, `Offered`, `Hired`, `NoVacancy`, `NotStarted` (households), `Posted`, `Filled`, `Unfilled` (positions), `OffersRefused`, `VacanciesRefused`, `Matched`, `TooMany`, `NoSite`, `ZeroYield`, `NoInput`, `NoTool`, `NoEndowment`, `NoRoom`, `EmployerFunds` (positions closed by each), `WageBelowUnit`, `HouseholdTicks`, `Paid Money`, `Refunded Money`, `Workers []LabourRow` with `LabourRow{Role; Offered, Hired, NoVacancy}`, `Employers []VacancyRow` with `VacancyRow{Role; Posted, Filled, Unfilled}` |

#### 4.1.10 Credit

Gate: `Modules.Credit`. The fields are CRD 16.1's. A claim has an `Origin`, `loan` or
`payout` (an insurance payout that the underwriter could not pay in full,
`spec-insurance.md` 8.3). The credit module forms only claims of origin `loan`; a
claim of origin `payout` is formed by INS (told by `CoverEnded.Short` and its
`Claim`) and ends through `LoanEnded` with `Origin` `payout`, so a reader separates
the two by that field. The credit instruments and the credit `Warn` rows count the
claims of origin `loan` only (6.2.1; `rules/observability-and-logging.md` 5.3.1): a payout claim
is counted by the insurance instruments. Whether the sums of `CreditSettled` hold
the payments of a payout claim, which the rebuild of 4.3.3 needs, or only those of
a loan, is CRD's to say, and the edit is owed (19).

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `LoanFormed` | CRD 16.1 | P3.3b, formation order | E | credit | `Claim`, `Creditor`, `Debtor`, `Region` (the debtor's), `Principal`, `Interest`, `RateBP`, `Term`, `Due` |
| `LoanEnded` | CRD 16.1; INS 15.4 | P1.7a, ascending claim ID; with a removal (2d); at the call of `ForgiveClaims` | E | credit | `Claim`, `Origin`+ (`loan`, `payout`), `Creditor`, `Debtor`, `Region` (the debtor's at the event), `CreditorRegion`, `ToEstate bool`, `Principal`, `PaidPrincipal`, `PaidInterest` (the waterfall's payment at a removal, 0 for other causes), `LostPrincipal`, `LostInterest`, `Cause` (`repaid`, `written_off`, `debtor_removed`, `forgiven`) |
| `LoanPassed` | CRD 16.1 | with a removal (2e), ascending claim ID | E | credit | `Claim`, `Creditor` (the removed agent), `Debtor`, `Region` (the estate's), `Owed` |
| `CreditRound` | CRD 16.1 | P3.3b, region ascending, after the last `LoanFormed`; never when all counts are 0 | A | credit | `Region`, `Requests`, `Offers`, `RequestsRefused`, `OffersRefused`, `InArrears`, `Exposure` (subsets of the refusals), `Formed`, `Principal`, `Interest`, `Unfilled`, `TooMany`, `Overflow` |
| `CreditSettled` | CRD 16.1 | P1.7a, region ascending, after the `LoanEnded` events of the tick; never when all counts and sums are 0 | A | credit | `Region`, then for its debtors `Due`, `Repaid`, `Defaulted`, `WrittenOff`, `PrincipalPaid`, `InterestPaid`, `Lost`, `Shielded`, `Headroom`, `Arrears`; and for its creditors `Received`, `ReceivedByEstate` |
| `CreditRateScaled` | CRD 16.1 | call, ascending `RegionID` | C | credit | `Region`, `BeforeBP`, `AfterBP` |

#### 4.1.11 Technology

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `TechnologyAdvanced` | TEC 11.1 | P8.3c, (`Region`, `Field`) order; for a cell in which anything accrued | A | technology | `Region`, `Field TechFieldID`, `Executions`, `Experience`, `Discoveries`, `BeforeBP`, `AfterBP`, `DoingAddedBP`, `TrendAddedBP`, `DiscoveryAddedBP`, `CatchUpAddedBP`, `ClampedBP` (the cut of the cap), `AtCap bool` |
| `TechnologyAdjusted` | TEC 7.1 | call, (`Region`, `Field`) order | C | technology | `Region`, `Field`, `DeltaBP` (signed, asked), `BeforeBP`, `AfterBP`, `LostBP` (the part the bounds cut) |

#### 4.1.12 The other later-stage modules

Each row is the owner's, with the names the rulings of the integration fixed (a
settlement class is a `SettlementClassID`, a cover policy is INS's, the key of a
store class is `store_class`).

| Event | Owner | Position | Class | Gate | Fields |
|-------|-------|----------|-------|------|--------|
| `SettlementClassChanged` | SET 10.1 | construction after the initial `AgentSpawned` events (`founded`); P7.2a | E | settlements | `Region`, `From`, `To` (`SettlementClassID`; `From` is `NoSettlementClass` at the founding), `Households`, `Cause` (`founded`, `grew`, `shrank`) |
| `SettlementHeld` | SET 10.1 | P7.2a, region ascending, instead of `SettlementClassChanged` | A | settlements | `Region`, `Class`, `Target`, `Households`, `TicksLeft` |
| `RentPaid` | LND 13.1 | P8.3a, (`Region`, `Resource`) order, after `EndowmentChanged`; for an endowment with a title, when any tier count, idle, held or unserved unit, refusal, cut, payment or forfeit is non-zero or the posted rent changed | A | tenure | `Region`, `Resource`, `Rent`, `Entitled`, `OwnUnits`, `CommonsUnits`, `LetUnits`, `Offered`, `Idle`, `Held`, `Unserved`, `PricedOut`, `Tenants`, `Lessors`, `Paid Money`, `Forfeited Money`, `NoTitle`, `RentFunds`, `CutUnits`, `Exempt`, `RentAfter`, `ToPool Money`, `From []RegionMoney`, `To []RegionMoney` with `RegionMoney{Region; Money}` |
| `TitleTransferred` | LND 13.1 | a removal, before `AgentBankrupted` (2b); or a call | E | tenure | `Region`, `Resource`, `From`, `To` (agents; `NoAgent` for the commons), `ShareBP`, `Price Money`, `Cause` (`removal`, `consumer`), `FromAfter`, `ToAfter`, `FromRegion`, `ToRegion`, `FromPool`, `ToPool` |
| `RentScaled` | LND 13.1 | call, (`Region`, `Resource`) order | C | tenure | `Region`, `Resource`, `From`, `To` (bp) |
| `FacilityChanged` | CAP 14.1 | construction after the last `AgentSpawned`; P2.1, P2.7; P7.1 (a removal, 2g; a role change); P8.3b; a call | E | infrastructure | `Facility`, `Class FacilityClassID`, `Region`, `To` (`NoRegion` unless a route with a target), `Owner`, `Kind` (`added`, `built`, `repaired`, `installed`, `transferred`, `ruined`), `Cause` (`recipe`, `initial`, `consumer`, `decay`, `use`, `role_changed`, `owner_removed`), `Before`, `After` (the condition), `Effect`, `GoodsListed` |
| `FacilityLimited` | CAP 14.1 | P8.3b, region ascending then class; never when `Cut`, `Refused`, `Voided` and `Pending` are all 0 | A | infrastructure | `Region`, `Class`, `Served`, `Cut`, `Refused`, `Voided`, `Pending` |
| `FiscalFlowApplied` | FIS 16.1 | P1.7b (taxes, remittances, payouts), P3.6c (tolls), P4.1a (duties); in the order of the flows, per flow and region ascending; for a due or an exempt trade | A | fiscal | `Flow string`, `Kind` (`tax`, `duty`, `toll`, `payout`), `Authority JurisdictionID`, `Region`, `Class` (`households`, `pool`, `treasury`), `Commodity`, `Due`, `Applied`, `Parties`, `Short`, `Cut`, `Headroom`, `Exempt` |
| `TreasuryOrdered` | FIS 16.1 | P3.4a, once per authority after its orders; when a standing order was active with a wish of at least 1 | A | fiscal | `Authority`, `Region` (the seat), `Bids`, `Asks`, `WantedBuy`, `PlacedBuy`, `WantedSell`, `PlacedSell`, `CutCash`, `CutHold`, `CutStock` |
| `MembershipChanged` | ORG 11.1 | P7.2b, (`RegionID`, `BodyID`) order; never when all counts are zero | A | institutions | `Region`, `Body`, `Members`, `Places`, `Eligible`, `Admitted`, `LapsedRole`, `LapsedRegion`, `Skipped`, `Barred`, `Waiting`, `Held` |
| `RightRefused` | ORG 11.1 | P7.2b, (`RegionID`, `RightID`) order | A | institutions | `Region`, `Right`, `Produce`, `Sell`, `Buy`, `Train` |
| `PriceControlled` | ORG 11.1 | P7.2b, (`RegionID`, `ControlID`) order | A | institutions | `Region`, `Control`, `Commodity` (`NoCommodity` for a rate limit), `Bids`, `Asks`, `Units` |
| `LegalLevelSet` | ORG 11.1 | call, `ControlID` order | C | institutions | `Control`, `CapBefore`, `CapAfter`, `FloorBefore`, `FloorAfter` (bp) |
| `RightSuspensionSet` | ORG 11.1 | call, `RightID` order | C | institutions | `Right`, `Suspended` (after) |
| `SavingPlanned` | SAV 12.1 | P3.6b, (`RegionID`, `RoleID`) order, after `RoomLimited`; when a triggering count is above 0 | A | saving | `Region`, `Role`; riding: `Households`, `Goal`, `Buffer`, `Funded`, `Committed`; triggering: `Saving`, `Buffered`, `Passed`, `Building`, `NoReturn`, `Standing`, `Unreachable`, `Survival`, `Builds`, `Repairs` |
| `PolicyDecided` | PLC 12.1 | P0.3a, per due actor in ascending `ActorID`, after the slot events of its application; if any count is non-zero | A | policy | `Actor`, `Jurisdiction`, `Entries`, `Applied`, `Unchanged`, `NotHeld`, `Duplicate`, `NoClose`, `Clamped`, `Stepped`, `InBand`, `NoReading` |
| `CoverFormed` | INS 15.1 | P3.3c, formation order | E | insurance | `Policy`, `Kind` (`route`, `store`), `Insured`, `Underwriter`, `Region`, `UnderwriterRegion`, `Shipment` (`NoShipment` for a store), `Mode`, `To`, `Sum`, `MaxPayout`, `Premium`, `RateBP`, `CoverBP` |
| `CoverEnded` | INS 15.1 | P1.3a; P8.1b; with a removal (2f); a `no_loss` store cover is not an event | E | insurance | `Policy`, `Kind`, `Insured`, `Underwriter`, `Region`, `UnderwriterRegion`, `Cause` (`no_loss`, `claimed`, `insured_removed`, `underwriter_removed`), `LossValue`, `Due`, `Paid`, `Short`, `Claim` (the payout claim, -1 when none), `Dropped`, `Cut`, `Capped bool` |
| `CoverRound` | INS 15.1 | P3.3c, region ascending, after `CreditRound`; never when all counts are 0 | A | insurance | `Region`, `Requests`, `Offers`, `RequestsRefused`, `OffersRefused`, `InArrears`, `Exposure`, `Candidates`, `Formed`, `Sum`, `Premium`, `Unfilled`, `NoShipment`, `NoRisk`, `NoOffer`, `BelowMinimum`, `Funds`, `TooMany`, `Overflow`, `ValueCapped` |
| `CoverSettled` | INS 15.1 | P1.3a and P8.1b, region ascending, after the `CoverEnded` events of the position; stage `claim`: P1.7a, region ascending, after CRD's `CreditSettled`; never when all counts are 0 | A | insurance | `Region`, `Stage` (`arrival`, `store`, `claim`), `Ended`, `Claimed`, `LossValue`, `Due`, `Paid`, `Short`, `Dropped`, `Cut`, `Capped`, `ValueCapped`, `Shielded`; at stage `claim` only `ClaimEnded` (payout claims ended repaid), `ClaimPaid` (principal paid by the debtors of the region) and `ClaimReceived` (principal received by the insureds of the region), the other fields being 0 |

Three further changes of events that the table of 4.1.6 and 4.1.8 already shows are
the later modules': the `Cause` of `MoneyFlowScaled`, `TransitFactorChanged` and
`EdgeStatusChanged` gains the constant `policy` (PLC), and the fields `Toll`,
`LossBP`, `Lost` and `AccountLost` are the fiscal and risk modules' (FIS, RSK).

The count is 89: the 47 of the first table of `rules/observability-and-logging.md`
4.3, the five marked `+` as new events and `WearApplied` (53), and the 36 of the later modules
(`StoreChanged` and `RoomLimited` in 4.1.3 and 4.1.4, `StoreLoss` in 4.1.5, the two
risk scales in 4.1.8, three in 4.1.9, six in 4.1.10, two in 4.1.11 and twenty in
4.1.12: 2 + 1 + 2 + 3 + 6 + 2 + 20 = 36). Volume class by event, for the cost model of
section 8:

| Class | Events |
|-------|--------|
| W (6) | `WorldCreated`, `WorldRestored`, `RunStarted`, `RunEnded`, `TickStarted`, `TickEnded` |
| A (35) | `MarketCleared`, `BeliefsUpdated`, `ProductionHalted`, `OwnUseChanged`, `UpkeepShortfall`, `WantShortfall`, `EntryLimited`, `PopulationChanged`, `BoundaryFlow`, `WearApplied`, `ReliefPaid`, `MoneyFlowLimited`, `ContractRound`, `MerchantsPlanned`, `MigrationReviewed`, `EndowmentChanged`, `RumoursServed`, `OutsideTraded`, `RoomLimited`, `LabourRound`, `SettlementHeld`, `RentPaid`, `CreditRound`, `CreditSettled`, `FacilityLimited`, `FiscalFlowApplied`, `TreasuryOrdered`, `MembershipChanged`, `RightRefused`, `PriceControlled`, `SavingPlanned`, `TechnologyAdvanced`, `PolicyDecided`, `CoverRound`, `CoverSettled` |
| E (31) | `RecipeExecuted`, `RecipeAbandoned`, `AgentSpawned`, `AgentBankrupted`, `EstateOpened`, `WeightChanged`, `RoleChanged`, `TraineeshipStarted`, `TraineeshipCompleted`, `TraineeshipEnded`, `MoneyFlowEnded`, `AgentRelocated`, `ShipmentDispatched`, `ShipmentArrived`, `ShipmentRejected`, `ContractFormed`, `ContractSettled`, `MigrationStarted`, `OutsideEnded`, `StoreChanged`, `StoreLoss`, `HireFormed`, `HireEnded`, `LoanFormed`, `LoanEnded`, `LoanPassed`, `SettlementClassChanged`, `TitleTransferred`, `FacilityChanged`, `CoverFormed`, `CoverEnded` |
| O (2) | `OrderPlaced`, `OrderRejected` |
| R (1) | `TradeExecuted` |
| C (14) | `MoneyFlowScaled`, `EdgeStatusChanged`, `TransitFactorChanged`, `EndowmentAdjusted`, `OutsideShocked`, `YieldShocked`, `PopulationScaled`, `RentScaled`, `CreditRateScaled`, `RouteRiskScaled`, `StoreRiskScaled`, `LegalLevelSet`, `RightSuspensionSet`, `TechnologyAdjusted` |

The six classes add to 89 (6 + 35 + 31 + 2 + 1 + 14). The first tables are the
authority and this one is for reading: a test (FR-OBS-003) derives the class of
each event from the tables of 4.1 and checks that every event type of the
catalogue appears once.

### 4.2 Conventions of the fields

4.2.1. **Names.** A field has the name its owner gave it, in the form of 3.3. The
convention that `From` and `To` name the two ends of a move (regions, roles) or,
for a quantity that changed, the value before and after (`WeightChanged`,
`MoneyFlowScaled`, `OutsideShocked`), is the owners' and is kept; a new event that
has both a pair of places and a pair of values names the values `BeforeBP` and
`AfterBP` (`TransitFactorChanged`, `YieldShocked`, `PopulationScaled`), so a field
name never has two meanings in one event.

4.2.2. **Money and quantities** of an event are the `Money`, `Price` and
`Quantity` types; a count of households is `int64` and is the sum of weights
(3.3.6). A basis-point value is `int64` and has `BP` or `bp` in its name or its
comment, as the owners wrote it.

4.2.3. **A `Reason`, a `Cause`, a `Kind` and a `ScaleKind`** are named string types
exported from the root with their constants (`event.go`); `internal/logkeys` holds
key strings only and does not alias them (architecture.md 2.9). Their constants are **the values of the
`reason` and `cause` log keys** (rules 3.3.2). `Kind` is a closed set **for each
event**: `faucet` and `sink` in `BoundaryFlow`; `adjust`, `shock` and `facility` in
`EndowmentAdjusted` (its owner's names); `added`, `built`, `repaired`, `installed`,
`transferred` and `ruined` in `FacilityChanged`; `added` and `removed` in
`StoreChanged` (`added` is shared with `FacilityChanged`, a different set of a
different field); `tax`, `duty`, `toll` and `payout` in `FiscalFlowApplied`
(whose `Class` is a set too: `households`, `pool`, `treasury`); `route` and `store`
in `CoverFormed` and `CoverEnded`; `ScaleKind` is `birth` and `death`. A `Cause` that
a policy actor sets is the constant `policy` (beside `consumer`) on
`MoneyFlowScaled` and `TransitFactorChanged`, and a member of the set of
`EdgeStatusChanged`. The constants of each event's field are listed
by the owning spec (the rejection reasons of `OrderRejected` in
`spec-clearing.md` 3.2; the causes of a removal in `spec-spawner.md` 13.1.1; the
abandon reasons in `spec-production.md` 14.1). A **consumer-supplied** free text
is never a `Cause`: INT's calls take a constant of a closed set, so that a cause
can be a metric attribute with a bounded number of values (5.4.1 of the rules).

4.2.4. **Unsigned counts that can be zero** are present in every event of their
kind even when zero: a recording of an aggregate has the same fields every time,
and "never emitted when all are zero" is the rule of the event, not of a field.

### 4.3 What an observer can rebuild

The rules state the rebuild of the agent and household counts (rules 4.3, last
paragraph). This spec adds what the new fields make possible, and requires it
(FR-OBS-040, FR-OBS-041).

4.3.1. **Agents by role and region** from `AgentSpawned` (+1), `AgentBankrupted`
(-1), `RoleChanged` (-1 `From`, +1 `To`) and `AgentRelocated` (-1 `From`, +1 `To`).

4.3.2. **Households by role and region** from the same with the weights:
`AgentSpawned.Weight`, `AgentBankrupted.Weight`, `WeightChanged` (`To` less
`From`), `RoleChanged.Weight` and `AgentRelocated.Weight`. An agent on a migration
journey is in its origin region until `AgentRelocated` with the cause `migration`
(`spec-migration.md` 13.1.1).

4.3.3. **Money of the household sector, of the pool and of the treasury sector, by
region.** Start from the first snapshot (or from nothing at construction) and add,
tick by tick, in this order of reading (the treasury sector is the money of the
treasuries seated in the region, `RegionRow.MoneyTreasury`, zero with no governed
jurisdiction):

1. the **money reasons that move the household or the pool sector across the
   boundary**: `initial` and the money-flow IDs, from `BoundaryFlow` (a flow's
   target, household class or pool, is its configuration's,
   `spec-money-supply.md` 4.3). The six `outside_*` reasons are **not** among them:
   they move the Outside accounts, a fourth sector that is zero at every
   boundary (`spec-ledger.md` 9.3), and the money a household gains or loses by
   trading with a counterparty is counted once, by `TradeExecuted`;
2. the money of `TradeExecuted` (an estate pseudo-agent has an `AgentID` below the
   number of regions, equal to its `RegionID`, `spec-world.md` 5.3; a counterparty
   has one from the number of regions to the end of the counterparties,
   `spec-world.md` 5.3; **a treasury** has one after the end of the counterparties'
   range, `spec-polity.md` 5.2.2, whose seat region is that of its row in
   `WorldCreated.Jurisdictions` and whose money is the **treasury sector** of the
   seat's region: the money of a `TradeExecuted` with a treasury as buyer or seller
   moves between the household sector of the trade's region and that sector, and a
   trade between a treasury and a counterparty or an estate moves between that sector
   and the other's, never into a household sector, and needs no `FiscalFlowApplied`);
3. `ReliefPaid.Paid`, pool to households;
4. `AgentSpawned.StartMoney` of an entrant that the pool funded (a spawner entrant
   or a household born into a new agent). An agent created by `initial` (every
   `AgentSpawned` before the first `TickStarted`, and every one with `External`
   true) is in the `initial` row of 1, not here. **The pool's own seed** (a warm
   start's `WarmStart.Pools`, `spec-estates.md` 2.3) is part of the region's
   `initial` money row too: the pool's seed is that row less the `StartMoney` of
   the `AgentSpawned` of the same region that `initial` funded and less the
   `Money` of each row of `WorldCreated.Jurisdictions` whose seat is the region (the
   `StartMoney` of a treasury, which the same row funded and which is in the
   treasury sector, not the household or pool sector), because `BoundaryFlow` has
   no holder;
5. `AgentBankrupted.Money` (household to pool) and `WeightChanged.Money` (signed,
   to or from the pool);
6. `AgentRelocated.Money` (between regions);
7. `ContractFormed.Escrowed` (household to contract sector), `ContractSettled.Paid`
   (to the deliverer, in the region `Region` of its account at the event) and
   `.Refunded` (to the payer, whose region is `To`, or, with `Cause`
   `payer_removed`, to the pool of `To`). `ContractSettled` carries `Region` for
   this (a field added here; the edit is MER's, section 19). The sector holds the
   escrow of the hires too (`MoneyContract` includes it, 5.3.2), by four readings of
   EMP 15.1.1: `HireFormed.Escrowed` (household to contract sector), `LabourRound.Paid`
   (contract sector to the workers' households, within the region of the round),
   `LabourRound.Refunded` (contract sector to the employers of the region: the refund
   of a start cut by the ration or the room) and `HireEnded.Refunded` (the remainder
   at the end, to the employer, a removed employer's own account first: EMP 6.4.1), each
   refund in exactly one of the last two;
8. **rent and the price of a title** (LND 13.1.1a): at a `RentPaid` the households of
   region `g` lose `From[g]` and gain `To[g]` and the pool of `Region` gains
   `ToPool` (`Paid` is the sum of `From`, a pool is never a payer); at a
   `TitleTransferred` with `Price` above 0 the payer's side of `ToRegion` (the
   household sector, or the pool when `ToPool`) loses it and the payee's side of
   `FromRegion` (the household sector, or the pool when `FromPool`) gains it; a
   removal moves no money and its `Price` is 0;
9. **a loan** at P3.3b (`LoanFormed`): `Principal` moves between two households of
   `Region` (a lane is a region) and changes the money of no region;
10. **a payment** at P1.7a (`CreditSettled`): the households of `Region` lose
   `PrincipalPaid + InterestPaid`, and the households of the region gain `Received` and
   its pool `ReceivedByEstate`;
11. **a waterfall payment** (`LoanEnded` with `Cause` `debtor_removed`):
   `PaidPrincipal + PaidInterest` from the households of its `Region` to those of its
   `CreditorRegion`, or to that region's pool when `ToEstate`;
12. **a fiscal flow** (`FiscalFlowApplied` of money): `Applied` from the household
   sector or the pool of `Region` (by `Class`) to the treasury sector of the seat of
   `Authority`, for the kinds `tax`, `duty` and `toll`; the reverse for a `payout`
   (from the treasury sector of the seat to the sector of `Class` in `Region`); the
   units of an in-kind payout move between the treasury's holdings and the households
   of the seat region;
13. **a cover payout** (`CoverEnded.Paid`): from the households of
   `UnderwriterRegion` to those of `Region`; a premium (`CoverFormed.Premium`) moves
   between two households of one region (a lane is a region, INS 7.2.1) and changes no
   regional total; the payout claim of `Short` is a claim of origin `payout`; its later payments
   at P1.7a (`CoverSettled` of stage `claim`) move `ClaimPaid` from the households of
   the debtors' `Region` and `ClaimReceived` to those of the insureds' `Region`
   (the credit instruments do not count them), and at a removal they are item 11.

A call between ticks has moved money that no event has reported yet: it is in
`Snapshot.Pending` until the tick that reports it, and the rebuild at such a
boundary adds `Pending` (FR-OBS-041). The pool is the estate account's money. The
`WeightChanged.Money` and `.Units` fields are added for this: without them a
reduction by the failure ladder, a birth into an existing agent and a death by
the population module move money between the household sector and the pool with
no event that says how much.

4.3.4. **What cannot be rebuilt** from events and is in the snapshot instead:
goods held (lots, ages and wear are the ledger's), beliefs, welfare records, and
anything that is a function of those. A consumer that needs them asks for a
snapshot or the per-agent record (5).

### 4.4 Rulings on the questions the specs left to OBS

#### 4.4.1 What `BoundaryFlow` covers

**Decision.** `BoundaryFlow` is emitted for a (region, reason, money or
commodity) with a non-zero total whose reason has the kind **faucet** or **sink**
(`spec-ledger.md` 7.1). The reasons of kind internal are not in it. The ledger
keeps flow totals for every reason (`spec-ledger.md` 7.4), so the totals exist,
but a faucet-and-sink event is what ADR 0001 asks for ("a faucet-and-sink event
with the reason"), and an internal flow is already told by a more specific event.

| Reason (ledger 7.2) | Kind | In `BoundaryFlow` | Told instead by |
|---------------------|------|-------------------|------------------|
| `initial`, `recipe_input`, `recipe_output`, `disposal`, `consumption`, `want_consumption`, `spoilage`, `wear`, `haul_use`, `transit_use`, `loss`, the six `outside_*`, and every money flow ID | faucet or sink | yes | |
| `trade` | internal | no | `TradeExecuted` |
| `start_money` | internal | no | `AgentSpawned.StartMoney` |
| `relief` | internal | no | `ReliefPaid` |
| `estate_money`, `estate_goods` | internal | no | `AgentBankrupted.Money`, `GoodsListed`, `EstateOpened`, `RoleChanged.GoodsListed` |
| `escrow_in`, `escrow_out`, `escrow_refund` | internal | no | `ContractFormed`, `ContractSettled` |
| `transit_out`, `transit_in` | internal | no | `ShipmentDispatched`, `ShipmentArrived` (a diverted cargo: `Lines[].Diverted`) |
| `split`, `merge`, `births`, `starvation`, `deaths` | internal | no | `WeightChanged` (`Money`, `Units`), `AgentBankrupted`; the stake of a household born into a new agent is `AgentSpawned.StartMoney` |
| `rent`, `title_sale` (LND) | internal | no | `RentPaid`, `TitleTransferred` |
| `wage_escrow`, `wage_pay`, `wage_refund` (EMP) | internal | no | `HireFormed`, `LabourRound`, `HireEnded` |
| `loan_out`, `loan_repay`, `loan_interest` (CRD) | internal | no | `LoanFormed`, `CreditSettled`, `LoanEnded` |
| `premium`, `payout` (INS) | internal | no | `CoverFormed`, `CoverEnded` |
| every fiscal flow ID (FIS; it shares the namespace of the money flow IDs) | internal | no | `FiscalFlowApplied` |

This answers `spec-reference-chain.md` 9, which lists `start_money` and `relief`
among the reasons of `BoundaryFlow` for S14: they are told by `AgentSpawned` and
`ReliefPaid`, and the edit to REF is in section 19.

4.4.1.1. **Fields.** `Region` is the ledger's attribution of the flow
(`spec-ledger.md` 3.5, 7.4: the region of the non-boundary account). `Kind` is
`faucet` or `sink`, a property of the reason, never of the amount. `Commodity` is
a `CommodityID`, or the constant `NoCommodity` (-1) for money: the type of a
`BoundaryFlow` is therefore one event for money and goods, as the rules have it.
`Total` is a magnitude, **non-negative and non-zero**; the direction is `Kind`. It
is an `int64` and not a `Money` or a `Quantity` because the one field holds money
for one row and units of a commodity for another, which `Commodity` tells apart.
For an `outside_*` reason `Region` is the **gate** (the region of the non-boundary
side, `spec-external-sector.md` 8.1), not the node; the node is in `OutsideTraded`.

4.4.1.2. **Order.** Within a tick: ascending `Region`, then the reason by the
**order of this list**: the built-in reasons in the order of `spec-ledger.md` 7.2,
then the money flows in ID order, then the six `outside_*` in the order money
reasons first and then goods reasons (a definition of its own: it is not the
creation order of the accounts, which for the `outside_*` goods reasons is
commodity-major, `spec-external-sector.md` 4.2.4), then money before goods, then
ascending `Commodity`. (`spec-money-supply.md` 13.4 and FR-MON-042 say "sorted by
reason"; the order here is the same list.) No map is iterated.

4.4.1.3. **Both forms of `initial`.** `initial` has a money row and a goods row
(`spec-ledger.md` 7.2); each is a `BoundaryFlow` of its own with its `Commodity`.

#### 4.4.2 The construction bucket (N-072)

**Decision.** The construction bucket is emitted as `BoundaryFlow` events with
`Tick` -1, **once**, at the entry of the **first** step of a world that is still in its
construction phase (built by `NewWorld`, or restored from a checkpoint taken
before its first step, which keeps its bucket, CHK), before that step's
`TickStarted`, in the order of 4.4.1.2 over the bucket's rows.

Why at the first step and not at construction. The construction phase ends when
the first `Step` begins (`spec-world.md` 5.5.1), and the consumer's calls between
`NewWorld` and the first `Step` (`AddAgent`, the slots) are inside it and add to
the bucket (5.5.2). An event emitted at construction would be incomplete the
moment the consumer adds a player. The bucket is final at the entry of the first
step, which is where a position for it is added: **P0.2a** (a request to WLD,
section 19; it takes a letter after P0.2, `spec-world.md` 6.1.1).

Why a `BoundaryFlow` and not a new event. The consumer's question is the same
("how much money and goods entered, by reason and region") and the adapter's
instrument is the same; `Tick` -1 says "before tick 0" and is the one value an
adapter has to treat apart. The alternative, an event type of its own, would
duplicate the fields for one reader's convenience (3.1.4).

Consequences. (a) The conservation identity of `spec-ledger.md` 9.1 (boundary
balance = bucket + the totals of every tick since) can be checked from the events
alone. (b) A world that never steps emits no bucket event; the snapshot has the
bucket (5.3.3). (c) A world restored after its first step has no bucket and emits none
(`spec-ledger.md` 7.4.1): its `WorldRestored` carries the levels instead (4.4.5).
(d) `economysim.money.flow` counts the endowment at `Tick` -1 like any flow
(6.3.2); a consumer who does not want the one-off spike in a rate graph drops the
first collection.

#### 4.4.3 `TickEnded` and `RunEnded`

**Decision.** `TickEnded` carries `Tick` and `Failed` and nothing else
(`spec-world.md` 3.5.6, 16.1). `RunEnded` carries `Completed`, `Cancelled` and
`Failed`. Both flags are accepted as added fields (rules 4.2).

4.4.3.1. **`TickEnded` is not a summary** (`spec-world.md` 22.9). A count a
consumer wants of a tick (orders, trades, agents) is derived from the events of
the step, which every observer receives, and the standing counts are in the
snapshot. A field on `TickEnded` would put a number in two places and would
need the world to keep counters for the sake of an observer that may not exist
(3.1.4).

4.4.3.2. **`RunEnded.Tick`** is the world's `Tick()` when `Run` returns. `Completed`
is the number of steps that finished without error; `Cancelled` is true only for
the context's cancellation; `Failed` is true only for a failed step; the two flags
are never both true (`spec-world.md` 3.4.2).

#### 4.4.4 The slots that had no event

**Decision.** `YieldShocked`, `TransitFactorChanged` and `PopulationScaled` are
added, each in the shape of `MoneyFlowScaled`: **one event per slot cell whose
value actually changed**, emitted at the call after the whole call has been
applied (a call is atomic, `spec-world.md` 8.1.1), in the order of the key, with
`Tick` the tick the next step will run. A call that sets a cell to the value it
already has emits nothing for that cell. The three slots are saved by their owners
and are not configuration, so the event is the only trace of a shock in the
event stream; without it a recording of a run with a failed harvest does not show
when the harvest failed.

#### 4.4.5 `WorldRestored` (CHK's seam)

**Decision.** A restore emits one `WorldRestored` as its first event, to the
observers given to the restore, with `Tick` the restored tick, the whole
configuration hash, the checkpoint's format version, whether the run was reseeded
and with what salt (CHK 6.8), the names table and the **snapshot at the restored
boundary**, whose header carries the original seed and the totals of agents and
households (they are in the snapshot once). It is not a replay: no `AgentSpawned` is emitted for the restored
agents, so an adapter that was never given the history has no levels. The
snapshot is the levels (6.4.2): every instrument that is fed by deltas starts
from it. CHK says when the restore calls the observers and what a restore does
with a failed checkpoint (no event); OBS fixes the fields. The event has class W
and is emitted only when an observer was supplied, like any other: the world
builds the snapshot for it only in that case (8.2).

#### 4.4.6 `ProductionHalted` (N-067, N-087)

**Decision.** One aggregate per region and role per tick in which at least one
free household's job request was declined by the **production halt**
(`spec-agents.md` 7.1.5): `Free` is the sum of weights of the role's agents in the
region that were passed to `Work` at P2.5 (the **free** households: a different
denominator from `Households` of `UpkeepShortfall`, which is the whole role, and
so a different name; it is not POP's `Exposed`), `Halted` the households of
those agents that requested no job and for which at least one recipe was skipped
by the halt (AGT defines "declined for the halt" in 7.1.5: an agent skipped on
recipe A that runs recipe B is not halted, and for a cohort the count is its
weight), and `Skipped` one row per recipe of the role that the halt skipped for
at least one free household, with the households for which it did (a household
may appear under several recipes, so the rows do not sum to `Halted`; the metric
fed by the rows counts a household once per recipe skipped, 6.2.1). The rows are in ascending `RecipeID`; a role
has at most as many as its list. It is never emitted when `Halted` is zero.

It answers N-087's question by commodity: the recipe row names the output that
was overstocked. The mechanism is AGT's and is settled (`spec-agents.md` 7.1.5): the
**world** computes the count, as a pure function of the view, for each free
household and each recipe of its role, from the agent's own stock and `OutputTarget`,
and sums it per (region, role). `JobRequest` and `Work` are unchanged: a strategy
cannot hide the halt or report a different count, so every strategy, the library's
and a consumer's, is counted alike. A `rest` is **not** counted here: it is the target-income rule working as designed,
not a clamp, and the households held to work by the floor are in the snapshot
(5.3.4). The position is P2.8a, after `OwnUseChanged`, in ascending (`RegionID`,
`RoleID`). `ProductionHalted` is the counter of the damper D24 (the production halt,
`ProduceHaltRatio`; `spec-stability.md` register, K-AGT09 of
`spec-calibration.md`): a halt that acts is visible here and nowhere else
(FR-STB-013).

#### 4.4.7 Events that are not added, with the reason

| Candidate | Decision | Reason |
|-----------|----------|--------|
| A season event (N-031, CAL open question 4) | none | `Calendar.At(tick)` is a pure function; the moment is in the snapshot (5.3.1); a consumer finds the tick of a season boundary by comparing `At(t)` with `At(t-1)`. An event would be a duplicate of a function of `Tick` |
| A directive event (`Direct`) | none | a directive is the consumer's own call; it changes no state until P1.9 and is saved by AGT; a replay uses the consumer's call log. Per agent per tick, a directive would be class E for no information the consumer lacks. A `Debug` line says it was set |
| A net trade event (N-115) | none | `OutsideTraded` carries `Traded` and `Value` per side, so the balance of trade per node is `sum(Value of buy sides) - sum(Value of sell sides)`, and per commodity `Traded` of the buy sides less the sell sides; the gross reasons of `BoundaryFlow` agree (`spec-external-sector.md` 8.2). The snapshot carries the cumulative units per commodity (5.3.9); the money per node is the sum of `OutsideTraded.Value` |
| A `CheckpointTaken` event | none | a checkpoint is a read (`spec-world.md` 8.4); a read changes nothing and emits nothing, so that reading cannot change the stream |
| An event per consumed or produced unit | none | `BoundaryFlow` under `consumption`, `recipe_output` and the rest (`spec-ledger.md` 12.1) |
| A per-agent welfare or belief event | none | class E per agent per tick would be the volume of the library; `BeliefsUpdated` and the snapshot's welfare rows are the aggregates |

#### 4.4.8 The events of one removal, and of the entry step (C11)

**Decision.** The order of `spec-world.md` 6.11 item 8 is adopted as a
requirement (FR-OBS-020): a removal emits, in this order, `RecipeAbandoned`
(reason `agent_removed`) for its job, `TraineeshipEnded` if it was in a
traineeship, `ContractSettled` for each contract of which it was the deliverer
(`deliverer_removed`), then the events of the removal steps of `spec-agents.md` 3.6
in step order, each only with its module on: `TitleTransferred` for each title moved
(2b, LND), `HireEnded` for each hire (2c, EMP: the worker's side and then the
employer's; the remainder of the escrow has returned to the employer's account
before the waterfall), `LoanEnded` for each claim ended (2d, CRD, in ascending claim
ID) and `LoanPassed` for each claim passed to the estate (2e, CRD, in ascending claim
ID), `CoverEnded` for each cover ended (2f, INS, unpaid, with the premium kept) and
`FacilityChanged` (`ruined`, cause `owner_removed`) for each facility (2g, CAP); and
last `AgentBankrupted`, `EstateOpened`. Every event of a removal precedes
`AgentBankrupted`. `AgentBankrupted` is the
only event that removes the agent: an observer decrements its agent count on
`AgentBankrupted` and only on it; the others accompany the removal and are not a
second removal.

Two gaps that 6.11 leaves. (1) The **traineeship events** of P7.1(a)
(`TraineeshipEnded` is placed by item 8 with a removal; the other two are not).
**Decision** (a request to WLD, section 19): within a region's entry step, step
(a) emits, in queue order, each completion as the `RecipeAbandoned` of its job (if
any, reason `role_changed`), then `TraineeshipCompleted`, then the `RoleChanged`
(cause `traineeship`) it causes, and then each place filled as
`TraineeshipStarted`; these precede the events of steps (b) to (d) (population
and spawner operations). The reason: the completion is the cause and the role
change its effect, the same order as `RecipeAbandoned` before `RoleChanged`.
(2) **A trainer's removal** ends its trainees' traineeships (`TraineeshipEnded`,
`trainer_left`, `spec-skill-barriers.md` 8.1) and abandons their jobs
(`RecipeAbandoned`, `principal_left`, `spec-production.md` 5.4). Decision: they
come **first**, trainee by trainee in ascending agent ID, each trainee's
`RecipeAbandoned` then its `TraineeshipEnded`, and then the trainer's own list
above; the owners (PRO 5.4, SKL 4.2) state it (section 19).

#### 4.4.9 The order of the events of the later modules' positions

`spec-world.md` 6.11 gives the merged order and the owners' specs the events of their
positions; this paragraph lists the three places where the order of events is not
that of the position's number alone.

(1) **P0.3a, the policy step.** After `TickStarted` and before the first event of
phase 1, for each due policy actor in ascending `ActorID`: the slot events of its
application in the order of its levers (`MoneyFlowScaled`, `TransitFactorChanged`,
`EdgeStatusChanged`, each with the cause `policy`), then its `PolicyDecided`. An
observer that wants to know which actor caused a slot event reads the next
`PolicyDecided`; one that wants only whether it was a policy reads the cause.
(2) **P7.1, a store ended by a role change.** `StoreChanged` (`removed`, cause
`role_changed`) comes after the `RoleChanged` that ended it; a store given by the role row at creation has no
event (`AgentSpawned.StoresLocked` reports the ones not given).
(3) **P8.3b, the facility step.** After `BoundaryFlow`, `EndowmentChanged` and
`RentPaid` (P8.3a): per ruined facility in ascending ID, `StoreChanged` (`removed`,
cause `facility`) for a store and then `FacilityChanged` (`ruined`); then per
installed store facility in ascending ID, `StoreChanged` (`added`, cause
`facility`) and then `FacilityChanged` (`installed`); then `EndowmentAdjusted`
(kind `facility`) in ascending (`RegionID`, `ResourceID`); then `FacilityLimited`;
then `TickEnded`. The state results of P8.3a and P8.3b are independent, so a golden
stream differs only by this order, which is written once here and in CAP 14.1.

### 4.5 Checks against the order of the tick

`spec-world.md` 6.11 gives the merged order; this spec checked each event of 4.1
against it and against the rules' paragraph on order (4.3 there). Findings:

| # | Finding | Ruling | Edit |
|---|---------|--------|------|
| K1 | Every event of 4.1 has a position in `spec-world.md` 6 or is emitted at a call or at construction, with the exceptions K2 to K4 | none | |
| K2 | `TraineeshipStarted` and `TraineeshipCompleted` have no place in 6.11 (item 8 places `TraineeshipEnded` with a removal), and the events of a trainer's removal are not ordered against its trainees' | 4.4.8 | WLD 6.11 item 8; PRO 5.4, SKL 4.2 |
| K3 | `BoundaryFlow` of the construction bucket has no position | P0.2a, 4.4.2 | WLD 6.2 |
| K4 | `ProductionHalted` and the four new events of 4.4.4 and 4.4.5 have no position | P2.8a; at the call; the restore | WLD 6.4, 6.11, 8.2 |
| K5 | WLD 3.4.2's row "Step `k+1` failed" says OBS may add a flag | accepted: `RunEnded.Failed` | WLD 3.4.2 |
| K6 | The rules' 4.3 says `BoundaryFlow` is "once per tick for each (region, reason, money or commodity) with a non-zero total" and `spec-reference-chain.md` 9 names `start_money` and `relief` as reasons of it | 4.4.1: boundary reasons only | REF 9; rules 4.3 |
| K7 | A flow that is both limited and ended in one tick: WLD 6.3 says "`MoneyFlowLimited` and `MoneyFlowEnded`, in flow ID order" without saying the order within a flow | `MoneyFlowLimited` then `MoneyFlowEnded` of the same flow | WLD 6.3 P1.7, 6.11 item 2 |
| K8 | The rules' paragraph names `MerchantsPlanned` "only when a role has the merchant block" and WLD P3.3a says "also with the module absent" | consistent: the gate is the block, not the module; a role with the block and no module is a validation question (`spec-merchants.md`), not this spec's | none |
| K9 | `RecipeAbandoned` can be emitted at P7.2 (a departure) and between ticks (a removal); 6.11 item 8 lists both | none | |
| K10 | `OrderRejected` can come from P3.4 (an estate's ask rejected) and P3.5 | listed in 4.1.2 | none |
| K11 | The events of the later modules have positions that `spec-world.md` 6 names: P0.3a (`PolicyDecided` and the policy-caused slot events, after `TickStarted`), P1.3a (`CoverEnded`, `CoverSettled`), P1.7a (`LoanEnded`, `CreditSettled`, then `CoverSettled` of stage `claim`), P1.7b (`FiscalFlowApplied` of taxes and payouts), P2.5a (`HireFormed`), P2.8b (`LabourRound`), P3.3b (`LoanFormed`, `CreditRound`), P3.3c (`CoverFormed`, `CoverRound`), P3.4a (`TreasuryOrdered`), P3.6a (`RoomLimited`), P3.6b (`SavingPlanned`), P3.6c (the toll `FiscalFlowApplied`), P4.1a (the duty `FiscalFlowApplied`), P7.2a (`SettlementClassChanged`, `SettlementHeld`), P7.2b (`MembershipChanged`, `RightRefused`, `PriceControlled`), P8.1a (`StoreLoss`), P8.1b (`CoverEnded`, `CoverSettled`), P8.3a (`RentPaid`, after `EndowmentChanged`), P8.3b (`FacilityChanged`, `StoreChanged` and `EndowmentAdjusted` of kind `facility`, then `FacilityLimited`) and P8.3c (`TechnologyAdvanced`); the removal steps are 2b to 2g (4.4.8) | the order is WLD 6.11, as amended by the owners' rows; this table only lists it | WLD 6, 6.11 |


## 5. The snapshot

### 5.1 What it is

```go
// Snapshot returns the state of the world at a tick boundary as plain data.
func (w *World) Snapshot() (Snapshot, error)
```

5.1.1. A `Snapshot` is a deep copy of plain data: no pointer into live state, no
function, no map (architecture.md 12.4). It can be marshalled on another goroutine
while the world steps. It is built **between ticks** on a healthy world, and
nowhere else: during a step the call returns `ErrInStep`, on a failed world
`ErrWorldFailed` (`spec-world.md` 3.5.1, 3.6). It draws no random number, changes
no state and takes no lock.

5.1.2. **Agents and households are separate fields everywhere**, and a count of
households is the sum of weights (FR-SCL-015, N-013). The pseudo-agents (the
estates and the counterparties of the outside) are in neither count and appear
only where a row is about them (the pool of a region, 5.3.2; the counterparties,
5.3.9).

5.1.3. **A field absent for want of a module is zero**, and `Modules` lists the
modules that are on, so a consumer tells "zero" from "not measured". A row that
exists only for a module (a shipment, a contract, an edge) is a row of a slice
that is empty with the module off.

5.1.4. **Order.** Every slice is in the canonical order of its key: ascending
`RegionID`, then `RoleID`, `CommodityID`, `RecipeID`, `ResourceID` and so on as the
row's key lists, with the ledger's registry order for a reason (4.4.1.2).

5.1.5. **The type names** of rows in this section are the root package's. They
are plain structs with exported fields and no embedded type (architecture.md
4.7). Money, prices and quantities are `Money`, `Price` and `Quantity`; a count
is `int64`.

### 5.2 The rule: a snapshot is a function of the saved state

5.2.1. **Rule.** Every field of a snapshot is a function of the checkpoint (the
state that CHK saves) and the configuration, and of nothing else. It follows that
a world restored from a checkpoint gives, at the same tick, a snapshot equal to
the one the uninterrupted world gives, field for field (FR-OBS-043; this is the
observable half of FR-WLD-063). The test needs no knowledge of the world's
inside: it compares two plain values.

5.2.2. **What that excludes.** A count of what happened in the **last tick** and
is not saved is not a snapshot field: it is the event of that tick. The owners of
five modules asked for such figures (`spec-spawner.md` 13.4, `spec-migration.md`
13.4, `spec-demand.md` 11.5, `spec-subsistence.md` 14.4, `spec-population.md`
10.4); 5.8 lists each and what became of it. The principle is 3.1.2: a state in
the snapshot, a transition in an event. A consumer that wants "the unfed
households of the last tick" has `UpkeepShortfall`, and also has the households
unfed *now* (`RoleRow.Unfed`), which equals the last `UpkeepShortfall.Unfed` unless a
removal, a role or weight change or an entry at P7.1 intervened (an agent's `Unfed`
is state and is recomputed at the next consumption, AGT 3.7).

5.2.3. **What it includes** that a naive reading would leave out: derived
figures that are a pure function of saved state (the household-weighted mean
welfare ratio of a role, the households held to work by the work floor, the
effective welfare window, the expected ratio of a role). They cost a scan
(5.5), not a counter, and the snapshot is where the scan is paid.

5.2.4. **The one exception that is not an exception.** The construction bucket
(`spec-ledger.md` 7.4.1) is saved only while the world is in its construction
phase (CHK), so `Snapshot.Construction` holds rows only in a snapshot taken
before the first `Step`. After the first step the bucket is the `BoundaryFlow`
events of `Tick` -1 (4.4.2). That differs from `spec-ledger.md` 7.4.1's "available
for the whole run" in the snapshot (the ledger itself keeps it for its audit,
`spec-ledger.md` 9.1); the edit to LED says so.

### 5.3 The sections

Each row type below lists its fields. **Cost** is the class of 5.5: **K** a figure
the owner already keeps, **D** derived from the configuration, **S** a scan over
agents, **L** a scan over lots, **B** a scan over beliefs, **Q** a scan over
the agents of the roles with a block that does more than a constant number of reads
per agent, **F** a copy of an in-flight set. **Src** is the spec that owns the meaning.

#### 5.3.1 The header

| Field | Type | Meaning | Src | Cost |
|-------|------|---------|-----|------|
| `Tick` | `int64` | the tick the next step will run (`World.Tick()`) | WLD 3.7.2 | K |
| `Seed` | `uint64` | the seed the world was built with, **for information**; a restore carries it (`spec-world.md` 12.3) | WLD 16.5 | K |
| `ConfigHash` | `[32]byte` | the configuration hash | CFG 8 | K |
| `Moment` | `Moment` | `Calendar.At(Tick)`, a pure function of the tick (N-031) | CAL 4 | D |
| `Names` | `Names` | the names table of 3.2.4 | OBS | D |
| `Modules` | `[]string` | the modules that are on, in the order of `spec-config.md` 3.2 | CFG | D |
| `Agents` | `int64` | live agents; pseudo-agents excluded | SCL 9 | K |
| `Households` | `int64` | the sum of the weights of the live agents | SCL 9 | K |
| `Money` | `Money` | money in non-boundary accounts, estates, escrow and treasuries included (the treasuries in `MoneyTreasury`, 5.3.2), counterparties excluded (they hold nothing at a boundary) | LED 9, POL 10.5 | K |

#### 5.3.2 Regions: `Regions []RegionRow`, one per region by `RegionID`

An outside node has a row with `Outside` true and every other field zero
(`spec-regions.md` 3.5): a consumer indexes by `RegionID` without a hole.

| Field | Type | Meaning | Src | Cost |
|-------|------|---------|-----|------|
| `Region` | `RegionID` | | REG | |
| `Outside` | `bool` | an outside node (no market, no agents, no pool) | REG, EXT | |
| `Agents`, `Households` | `int64` | agents of the region and their weights; a migrant on a journey is in its origin | SCL, MIG 13.1.1 | S |
| `MoneyHouseholds` | `Money` | `M_h`: the money of the agents of the region | MON 13.4 | S |
| `MoneyPool` | `Money` | `M_p`: the money of the region's estate account, which is the pool | EST 2.4 | K |
| `MoneyContract` | `Money` | `M_c`: the money in Escrow accounts attributed to the region by the ledger, the escrow of the hires (`Hires[].Held`) included | MON 13.4, LED 9.3, EMP 15.4 | F |
| `MoneyTreasury` | `Money` | `M_t`: the money of the treasuries seated in the region (0 with none) | POL 10.5 | K |
| `Class`, `ClassSince`, `ClassTarget`, `ClassAtFloor` | `SettlementClassID`, `int64`, `SettlementClassID`, `bool` | the settlement class of the region as of the last review, the tick of its last change (0 at construction), the class a review at the current `Households` would give ignoring the dwell, and whether the class is the region's floor and the count alone would give a lower one (settlements on; `Class` and `ClassSince` are saved, the other two are derived) | SET 10.5 | K, D |
| `EntryBudget` | `Money` | the budget that the next entry step's view would carry from the current pool, `MoneyPool * EntryPoolShare / 10 000`; **advisory**: the pool budget is taken in P7.1(d) after the deaths (WLD C15) and is not a ceiling on any draw | SPN 13.4, WLD C15 | K |
| `HouseholdCap` | `int64` | the region's `HouseholdCap` of the population module, 0 for none | POP 10.4 | D |
| `OnJourneyFrom`, `OnJourneyTo` | `int64` | households on a migration journey that leave this region and that head for it | MIG 13.4 | F |

#### 5.3.3 Money, the boundary, the flows, the pending totals and the bucket

| Slice | Row | Fields | Meaning | Src | Cost |
|-------|-----|--------|---------|-----|------|
| `Boundary []BoundaryRow` | one per reason with a boundary account, in the order of 4.4.1.2, money then goods by commodity | `Reason`, `Kind`, `Commodity` (`NoCommodity` for money), `Balance int64` (signed, as the account is) | the boundary balance per reason: how much has entered (negative of a faucet's balance) or left | LED 7, MON 13.4 | K |
| `Flows []FlowRow` | one per configured money flow, in flow ID order | `Flow string`, `Total Money` (cumulative), `ScaleBP int64`, `Ended bool` | each flow's state | MON 13.4 | K |
| `Pending []FlowTotal` | the totals that calls between ticks added since the last tick end, in the order of 4.4.1.2 | `Region`, `Reason`, `Kind`, `Commodity`, `Total` | what the next tick's `BoundaryFlow` will include besides its own (`spec-world.md` 8.1.3); before the first step the same moves are in `Construction` and `Pending` is empty | WLD 8.1.3 | K |
| `Construction []FlowTotal` | the construction bucket, only before the first step (5.2.4) | as `Pending` | the endowment at construction | LED 7.4.1 | K |

`M_h / H` and the velocity are computed by a consumer or by IND from these
(`spec-money-supply.md` 13.4); no field is a ratio of two others except where a
role row says so.

#### 5.3.4 Roles: `Roles []RoleRow`, one per (region, role) of a region that is not an outside node

| Field | Type | Meaning | Src | Cost |
|-------|------|---------|-----|------|
| `Region`, `Role` | IDs | | | |
| `Agents`, `Households` | `int64` | | SCL | S |
| `Unfed` | `int64` | households not fully fed in the last consumption: the sum over the role's agents of their `Unfed` (a count of households, not of weights; a cohort of weight 40 with `Unfed` 4 is 4; saved agent state, AGT 5.4) | AGT 14.6 | S |
| `UnfedAgents` | `int64` | agents with `Unfed` above 0 | AGT 14.1 | S |
| `MaxHunger`, `MaxPrivation` | `int64` | the largest over the role's agents | AGT 5.4, SPN 13.4 | S |
| `HeldByFloor` | `int64` | households that the work floor `MinWorkShare` would hold to work **at the boundary**, evaluated over the agents with no job in progress on the welfare record as it stands after P6.3: a function of saved state, which may differ from the count the "work floor binding" `Warn` used at P2.5 (a shorter ring and another set; AGT 7.1.3, WEL 5.3 count it "in the last tick", routed, section 19) | WEL 11.2, AGT 14.6 | S |
| `MeanRatioBP`, `HasMeanRatio`, `RatioHouseholds` | `int64`, `bool`, `int64` (the households whose ratio exists) | the household-weighted mean welfare ratio (the sum over agents of weight times ratio over the sum of weights of the agents whose ratio exists); absent if none (N-081 item 1) | WEL 11.2 (rounded toward zero, FR-WEL-036) | S |
| `WageObserved`, `HasWage` | `Money`, `bool` | the observed wage of the role in the region | WEL 4.3 | S |
| `PremiumBP`, `HasPremium` | `int64`, `bool` | the observed premium | WEL 4.3, SKL 8.2 | S |
| `Window` | `int64` | the role's effective welfare window in ticks | WEL 2.3.3 | D |
| `ExpectedRatioBP`, `Viable`, `HasExpected` | `int64`, `bool`, `bool` | the spawner's expected ratio and viability signals, when defined | SPN 13.4 | S |
| `OnOwnUse` | `int64` | households with the own-use flag (subsistence on) | SUB 14.4 | S |
| `Places`, `PlacesTaken`, `Queue`, `Trainers` | `int64` | the traineeship throttle: places, taken, applicants waiting, trainers (skill barriers on) | SKL 8.2 | S |
| `BirthRateBP`, `DeathRateBP`, `NaturalIncreaseBP` | `int64` | household-weighted mean annual rates at the agents' current ratios, and their difference (signed) (population on) | POP 10.4 | S |
| `BirthCarry` | `int64` | the (region, role) birth carry, below its denominator | POP 11 | K |
| `Employed`, `Employing` | `int64` | households of the role on a hired job at the boundary; hired households working for agents of the role (employment on) | EMP 15.4 | S |
| `LimitedHouseholds`, `RoomMass`, `RoomBulk`, `LoadMass`, `LoadBulk` | `int64` | the households of the limited agents of the role in the region, and the sums over them of the room and the load in each dimension, the weight folded into the room (storage on); `economysim.room.fill` is `Load * 10 000 / Room` of a dimension | STO 14.5 | S |
| `SavingHouseholds`, `GoalMoney`, `BufferMoney`, `IdleMoney` | `int64`, `Money` | the households of agents with a goal above 0, and the sums over agents of `Goal`, of `max(0, Precaution - ReserveMoney)` and of `Idle` (saving on, a role with the block; a drawdown and a project test per agent, not a constant number of reads) | SAV 8.4.2 | Q |

The `MeanRatio` per region over all roles is not a field: it is the household-
weighted mean of the role rows, and a consumer computes it as the sum of `MeanRatioBP` times `RatioHouseholds` over
the sum of `RatioHouseholds` (the households whose ratio does not exist are left
out as the owner says). That is one more place where a field would be a function of other fields.

#### 5.3.5 Markets and jobs

`Markets []MarketRow`, one per (region, commodity) of a region that is not an
outside node, in that order:

| Field | Type | Meaning | Src | Cost |
|-------|------|---------|-----|------|
| `State` | `MarketState` | the market report that INT's query `Market(region, commodity)` returns (`spec-integration.md` 6.3): `Region`, `Commodity`, the published `Price`, its `Age`, the `Volume` and the `UnmatchedBid` and `UnmatchedAsk` units of the last clearing, `BestBid`, `BestAsk` with `HasBid` and `HasAsk`, and the price window's `WindowLen`, `WindowMean`, `WindowLow` and `WindowHigh`, under INT's names. One type, so the snapshot and the query cannot disagree; the event `MarketCleared` carries the same values under its owner's names (`PriceAge`, `HasBestBid`, `HasBestAsk`). It is saved state: CLR 5.4 and 5.5 and BEL 7 (the last report is a request to CHK, section 19) | CLR 5.4, 5.5; INT 6.3 | K |
| `HeldByAgents`, `HeldByEstate`, `HeldByTreasury` | `Quantity` | the stock at the boundary in the households' accounts, in the estate's and in the treasuries seated in the region: what no order of the tick sold (N-081 item 2). Goods in Transit accounts are in `Shipments` | OBS, FIS 16.3 | L |
| `Lots` | `int64` | the number of lots of the commodity held by the accounts the three `HeldBy` fields count (`spec-ledger.md` 5.1.3: one per distinct key, so a spoiling good fragments by born tick and a durable by remaining durability). A count, a function of the saved lots (FR-OBS-043); 0 where nothing is held. Lots of Transit and Escrow accounts are not counted: their goods are in `Shipments` | LED 5.1.3, N-130 | L |
| `Pending` | `Quantity` | the pending output of the jobs in progress in the region: in no account, so this is the only place it is measured | PRO 14.3 | S |
| `Beliefs`, `MeanCentre`, `LowCentre`, `HighCentre`, `AtFloor`, `AtCeiling` | `int64`, `Price`, `Price`, `Price`, `int64`, `int64` | the beliefs of the region's agents for the commodity: how many, the mean (half to even), the lowest and the highest centre, and how many sit at the lower or upper bound of their effective range (the reference band or the cost band, BEL 12.1) (N-040) | BEL 12.1 | B |

`Jobs []JobRow`, one per (region, recipe) with at least one job, in that order:
`Region`, `Recipe`, `Agents` (agents with a job of the recipe), `Executions`
(the sum of their executions), `Households` (the sum of their weights). It is
`PRO 14.3`'s "number of agents with a job, per recipe", by region. Cost S.

#### 5.3.5a The region state

INT's query `Region(r)` returns the part of the snapshot whose key has the region `r` (`spec-integration.md` 6.5), so that a frame that needs one region does not build the whole snapshot:

```go
// RegionState is the snapshot restricted to one region. Plain data.
type RegionState struct {
	Region     RegionRow
	Roles      []RoleRow      // the region's role rows, by RoleID
	Markets    []MarketRow    // by CommodityID
	Jobs       []JobRow       // by RecipeID
	Endowments []EndowmentRow // by ResourceID
	Shipments  []ShipmentRow  // those that leave or reach the region, by ID
	Journeys   []JourneyRow   // those that leave or reach it, by agent ID
	Facilities []FacilityRow  // by FacilityClassID (CAP on)
	Stores     []StoreRow     // by StoreClassID (STO on)
	Bodies     []BodyRow      // by BodyID (ORG on)
	Tech       []TechRow      // by TechFieldID (TEC on)
}
```

Its cost is the region's share of the scans of 5.5 (`O(agents of the region)` for the
S rows, the region's lots for L and its beliefs for B), the second class of
`spec-integration.md` 6.1.3, and it equals the filter of a full snapshot by region
(FR-OBS-044). The world-level fields (`Boundary`, `Flows`, `Pending`, `Outside`)
are not in it, nor are the in-flight sets that have two regions (`Contracts`, `Hires`,
`Claims`, `Policies`, `Away`) or one value for the world (`Credit`, `Insurance`,
`Jurisdictions`, `Actors`, `LegalLevels`, `Suspended`).

#### 5.3.6 Resources

`Endowments []EndowmentRow`, one per (region, resource) in that order:
`Region`, `Resource`, `Stock`, `Capacity` (effective), `AbundanceBP`,
`ExhaustedRun`, `CapacityScaleBP`, `RegenScaleBP` (both 10 000 when not shocked); with the
tenure module on and tenure data for the endowment also `Holders` (the number of
titles), `PoolBP` (the pool's share), `CommonsBP`, `TopBP` (the largest share of an
agent), `Posted` (the posted rent), `ScaleBP` (the rent scale), `Closed`, and the two
diagnostic runs `NoTitleRun` and `RentFundsRun` (LND 13.4; the per-holder list is the
query `Holders`, not a snapshot field). Source RES 14.4. Cost K (the abundance is a
function of stock and capacity). The
measures of 8 there that need the full-draw ratio (`N_bind`, N-109) are **not**
fields: see 5.8.

#### 5.3.7 The graph

The other slots (yield shocks, birth and death scales, flow scales, endowment scales, outside shocks, rent scales, credit rate scales, route and store risk scales) are **not** snapshot fields: the non-default ones are
INT's `Slots()` (`spec-integration.md` 5.3). The edges and the transit factors are
in the snapshot because REG and TRN asked, and they are the same rows as in
`Slots()`.

| Slice or field | Row | Meaning | Src | Cost |
|----------------|-----|---------|-----|------|
| `Edges []EdgeRow` | `From`, `To`, `Closed bool`, `FactorBP int64` | only the edges whose status is not the default, by `EdgeID` | REG 10.5 | K |
| `Components int64` | | strongly connected components of the graph as it is now | REG 10.5 | K |
| `TransitFactors []TransitFactorRow` | `From`, `To`, `Mode`, `BP` | the non-default factors, by (from, to, mode) | TRN 13.5 | K |

#### 5.3.8 What is in flight

| Slice | Row | Fields | Src | Cost |
|-------|-----|--------|-----|------|
| `Shipments` | by shipment ID | `Shipment`, `Mode`, `Region`, `To`, `Shipper`, `Traveller`, `Hired`, `Consignee`, `Dispatched`, `Arrives`, `TicksLeft`, `Account` (the Transit account, TRN 5.2), `Units`, `HaulUnits`, `Lines []Amount` | TRN 13.5 | F |
| `Away` | by agent ID | `[]AgentID`: the agents away on a trip | TRN 13.5 | F |
| `Journeys` | by agent ID | `Agent`, `Role`, `From`, `To`, `Arrives`, `Weight` | MIG 13.4 | F |
| `Contracts` | by contract ID | `Contract`, `Payer`, `Deliverer`, `Region`, `To`, `Commodity`, `Quantity`, `Price`, `Escrowed`, `Shipment`, `Formed` (the tick, MER 16), `Arrives`, `TicksLeft` | MER 15.4 | F |
| `Escrow` | one `Money` | the money held for contracts in total (the contracts' part of `MoneyContract`; the hires' part is the sum of `Hires[].Held`) | MER 15.4 | F |
| `Hires` | by hire ID | `Hire`, `Region`, `Employer`, `Worker`, `Recipe`, `Households`, `Wage`, `Formed`, `Ends` (the completion tick), `TicksLeft`, `Account` (the Escrow account), `Held` (its balance) | EMP 15.4 | F |
| `Claims` | by claim ID | `Claim`, `Origin`+ (`loan`, `payout`), `Creditor` (an `AgentID`, or the estate's agent ID, which is its `RegionID`), `Debtor`, `Region` (the debtor's), `Principal`, `Interest`, `PrincipalOwed`, `InterestOwed`, `RateBP`, `Formed`, `Due`, `Overdue` | CRD 16.4, INS 15.4 | F |
| `Credit` | one value | `Lent`, `Repaid`, `InterestPaid`, `InterestFixed`, `LostPrincipal`, `LostInterest`, `Awarded`+ (the payout claims awarded by INS, whose `Short` is read from it) | CRD 16.4, INS 15.4 | F |
| `Policies` | by cover policy ID | `Policy`, `Insured`, `Underwriter`, `Region`, `Shipment`, `Sum`, `MaxPayout`, `CoverBP`, `Premium`, `Formed` (route covers only; a store cover opens and ends inside one tick) | INS 15.4 | F |
| `Insurance` | one value | `Formed`, `Premiums`, `Due`, `Paid`, `Dropped`, `Cut` (`Short` is `Credit.Awarded`, read, not saved twice) | INS 15.4 | F |

Each is bounded by what is in flight, and so by `MaxOpen`, `MaxClaims`, `MaxPolicies`,
the shipments in flight, the hires and the agents (5.6). `Claims` of origin `payout`
are the insurance claims; the credit instruments count `loan` only (6.2.1).

#### 5.3.9 The outside

With the external module on: `Outside OutsideState`, whose slices are in the order
of the configuration.

| Slice | Row | Fields | Meaning |
|-------|-----|--------|---------|
| `Lines` | (`Node`, `Commodity`) | `Mid Price`, `DriftBP`, `ShockBP` | the line's mid and the state of its walk and shock |
| `Sides` | (`Node`, `Gate`, `Commodity`, `Side`) | `Quote Price`, `Carry`, `Units`, `ListValue Money`, `Ended bool` | each side: its quote, its carry, its cumulative units and list value, whether it ended |
| `Trade` | (`Commodity`) | `Imports Quantity`, `Exports Quantity` | cumulative over all nodes: the sum of the `Units` of the sell sides and of the buy sides of the commodity (saved per side by EXT 16) |
| `Counterparties` | by agent ID | `Agent`, `Node`, `Gate`, `Side`, `Account int64` | the pseudo-agents (FR-EXT-046) |

The six boundary balances per reason are rows of `Boundary` (5.3.3). Source EXT 15.4.
`Trade` is computed from the sides' cumulative `Units` at the call, so it adds no
state. **There is no per-node money balance**: the boundary accounts are one per
reason (`spec-ledger.md` 3.4), so the money of one node is not a function of saved
state, and `ListValue` is not the money that changed hands (`spec-external-sector.md`
7.4); the balance of trade in money is the world's, from `Boundary` (`outside_budget`
less `outside_unspent`, and `outside_receipts`), and per node it is the sum of
`OutsideTraded.Value` over the events. Cost K.

#### 5.3.10 The population module

`Population PopulationState`: `ReplacementRatioBP` (`r*`, the ratio where births
equal deaths) and `GainBP` (the module's gain, in the unit of `spec-population.md`
5), both functions of the configuration. Cost D. The region and role rows above
carry the rest (`BirthRateBP`, `DeathRateBP`, `NaturalIncreaseBP`, `BirthCarry`,
`HouseholdCap`). Source POP 10.4.

#### 5.3.11 The rows of the other later-stage modules

Each is a slice, empty with its module off, in the canonical order of its key, and a
function of saved state (5.2.1). The rows that the earlier groups gain (`MoneyTreasury`,
the settlement class fields, `Employed`, the room, load and saving fields,
`HeldByTreasury`, the tenure fields of an endowment) are in 5.3.2 to 5.3.6.

| Slice | Row | Fields | Src | Cost |
|-------|-----|--------|-----|------|
| `Jurisdictions` | one per **governed** jurisdiction, ascending `JurisdictionID` | `Jurisdiction`, `Seat`, `Agent` (the treasury pseudo-agent), `Money` | POL 10.5 | K |
| `Actors` | one per policy actor, ascending `ActorID` | `Actor`, `Jurisdiction`, `Pending` (the settings in the mailbox of a `directed` actor, 0 for another policy) | PLC 12.3 | K |
| `Bodies` | (`Region`, `Body`) of the scope, ascending | `Region`, `Body`, `Members`, `Eligible`, `Places` (households; `Waiting` is `Eligible - Members`, derived by the reader) | ORG 11.5.1 | S |
| `LegalLevels` | one per control or rate limit whose level differs from the configured one, ascending `ControlID` | `Control`, `CapBP`, `FloorBP` | ORG 11.5.2 | K |
| `Suspended` | the suspended rights, ascending | `[]RightID` | ORG 11.5.2 | K |
| `Facilities` | (`Region`, class) with a non-zero count | `Region`, `Class`, `Facilities` (a count), `ConditionBP` (the sum of `Condition * 10 000 / Durability`), `Effect`, `Pending` (store facilities not installed) | CAP 14.5 | S |
| `Stores` | (`Region`, class) with a non-zero count | `Region`, `Class`, `Stores` (a count of stores, not of agents or households) | STO 14.5 | S |
| `Tech` | (`Region`, field) with `Experience` or `AdvanceBP` above 0 | `Region`, `Field`, `Experience`, `AdvanceBP`, `GainBP` | TEC 11.4 | K |

The snapshot's header `Modules` lists the module names that are on, so a consumer
tells an empty slice from a module that is off (5.1.3). The scale lists of the credit, rent and risk
modules are INT's `Slots()` and are not snapshot rows (5.3.7), and an adjustment of a
technology gain is not a slot (INT 5.1).

### 5.4 The per-agent record

Per-agent data is **not** in the snapshot (5.6). The record of one agent is
INT's `AgentState` (`spec-integration.md` 4.7.2), returned by its query
`Agent(id)`; OBS fixes which fields the specs asked for, and INT's table covers
them (INT 4.7.2):

| Asked by | Field | In `AgentState` |
|----------|-------|-----------------|
| AGT 14.6 | `Unfed`, `Hunger`, `Privation` | yes |
| WEL 11.2 | the welfare ratio and whether it exists, the target ratio, the working ticks in the window | yes |
| WEL 11.2 | the earned ratio beside the ratio (`spec-welfare-and-wages.md` 2.1.7) | yes (INT 4.7.2, Standing) |
| SUB 14.4 | `OwnUse`, `OwnUseJobs`, `Cover`, `Secure` | yes |
| DEM 11.5, 12 | the carries of the wants of the agent's ladder | yes (INT 4.7.2, Wants) |
| BEL | the centre, the width and the bounds per commodity of the tradable set (N-040) | yes |
| MIG 13.4 | the agent's journey (destination, arrival tick) | yes |
| TRN 13.5 | whether the agent is away | yes |
| STO 14.5 | the group Room: `Room`, `Load`, `Pledge`, `Free`, `Stores` | yes (INT 4.7.2, Room) |
| EMP 15.4 | `Employer`, `HasEmployer`, `Hired` (the households on a hired job) | yes (INT 4.7.2) |
| CRD 16.4 | the claims of the agent, as creditor and as debtor | yes (INT 4.7.2, Credit) |
| INS 15.4 | the cover policies of the agent and its liability as underwriter | yes (INT 4.7.2, Policies and Liability) |

A query is not a snapshot: it is a read of one agent or one market, it may
allocate, and it costs the second class of `spec-integration.md` 6.1. The rows of
5.3.5 are what the queries `Market` and `Region` return for one key, so a
consumer's tool and an adapter read the same field names (INT 6.3 and 6.5 adopted this).

### 5.5 What it costs

5.5.1. **The classes.** `K` and `D` are copies of figures the owners keep or can
compute from tables: `O(rows)`. `S` is one pass over the live agents, each
touched once, with a constant number of reads per agent: `O(A)`. `L` is one
pass over the lots of the accounts of households and estates: `O(accounts +
lots)`. `B` is one pass over the beliefs: `O(sum over agents of |T_a|)`. `Q` is
one pass over the agents of the roles that have a block (saving), each with a
drawdown over a window and a project test: `O(sum over those agents of (T + recipes
* inputs + facilities))`, paid only by a world with such a role. `F` is
a copy of what is in flight. The snapshot is the sum, plus the allocation of the
deep copy.

5.5.2. **Against the budget of `spec-scale.md` 6.** Using the placeholder sizes of
`spec-scale.md` 6.3 for the first consumer (5 000 agents, a tradable set of 8, 30
commodities, 5 regions, 20 roles) and a placeholder 10 nanoseconds a read:

| Class | Count (placeholder sizes) | At 10 ns |
|-------|---------------------------|----------|
| S | 5 000 agents, a placeholder 6 reads each: 30 000 | 0.3 ms |
| L | 5 000 accounts, 8 lots each: 40 000 | 0.4 ms |
| B | 5 000 times 8: 40 000 | 0.4 ms |
| Q | 1 000 agents with a block, a placeholder 40 reads each: 40 000 (SAV 12.5) | 0.4 ms |
| K, D, F | under 1 000 rows | under 0.01 ms |
| allocation | rows of 5.3 times about 200 bytes (rows of 14 to 25 fields): 150 market rows, 100 role rows, 5 region rows and the in-flight sets, about 300 rows, about 60 KB | not counted |
| **Total** | | **about 1.1 ms** without class Q, **about 1.5 ms** with it |

Against the tick of about 53 ms of the same budget that is about 2% for one
snapshot (about 3% with class Q), so a consumer may take one every tick; a consumer that only needs the
cheap rows pays for the scans anyway, and a call with groups to skip is the
addition this spec leaves open (section 18, question 3). These numbers are
counted, not measured; the library measures no wall-clock time
(architecture.md 7.3).

### 5.6 What bounds its size

5.6.1. The size of a snapshot is `O(R * (roles + commodities) + resources *
regions + modes * regions^2 + edges)` for the configuration (and the rows of the
later modules of 5.3.11, bounded by regions times classes, fields and bodies), plus
the in-flight sets of 5.3.8 (`Shipments`, `Away`, `Journeys`, `Contracts`, `Hires`,
`Claims`, `Policies`). It is **never proportional to the number of agents**, except
through those sets, each of which is a set of agents that are doing something rare
(a trip, a journey, a hire, a claim, a cover) or a configured cap (`MaxOpen`,
`MaxClaims`, `MaxPolicies`). For the first consumer's world it is of the order of 300 rows
(5.5.2).

5.6.2. A world of 200 000 agents (`spec-scale.md` 6.3's third row) has the same
snapshot as one of 5 000 with the same configuration, which is what lets IND and
STY read a snapshot every tick of a long run.

### 5.7 The snapshot is not the checkpoint

| | Snapshot | Checkpoint |
|--|----------|------------|
| Purpose | observation: a summary a consumer reads | resumption: the complete state that continues the run exactly (ADR 0002) |
| Contents | aggregates and in-flight sets, derived figures (5.2.3), the names table | every account, lot, agent, belief, job, record, carry and the random state |
| Size | bounded by the configuration (5.6) | proportional to the world, agents and lots included |
| Derived values | yes: means, rates, effective values | no: derived values are rebuilt on restore |
| Can resume a run from it | **no**: it has no lots, no beliefs, no random state | yes |
| Encoding | plain Go values; the consumer marshals them as it likes; there are no struct tags (architecture.md 12.6) | one canonical binary encoding with a version and a hash (CHK) |
| Taken | between ticks, on a healthy world | between ticks, on a healthy world |
| Relation | a function of the checkpoint and the configuration (5.2.1) | the input of that function |

A snapshot cannot be turned into a checkpoint and has no format version: its
shape may gain fields in a minor version (3.3.4), and nothing validates one on
input because nothing reads one back.

### 5.8 What the owners asked for, and where it went

| Spec | Asked | Decision |
|------|-------|----------|
| REG 10.5 | edges with a non-default status, the component count | 5.3.7 |
| MON 13.4 | `M_h`, `M_p`, `M_c`, the world's `M` and `H`, the boundary balance per reason, each flow's `Total`, `Scale`, `Ended` | 5.3.2, 5.3.3 |
| POP 10.4 | mean birth and death rates, natural increase, the birth carry, the unfed households of the last tick; per module `r*` and the gain | 5.3.4, 5.3.10; the unfed households are `RoleRow.Unfed` (state), not a last-tick count |
| TRN 13.5 | shipments in flight, away agents, non-default transit factors | 5.3.7, 5.3.8 |
| RES 14.4 | stock, effective capacity, abundance, `ExhaustedRun`, scales when non-default; `N_bind` where VIA computed it | 5.3.6; `N_bind` is **not** a field (N-109, below) |
| EXT 15.4 | mid, quotes, drift, shock, carry, cumulative units and list value, the balance of trade, the six boundary balances, the counterparties | 5.3.9, 5.3.3; the balance of trade per node in money is **not** a field (it is not saved state, 5.3.9): the world's is `Boundary`, the node's is the events |
| SUB 14.4 | households in the mode; households blocked in the last tick; per agent `OwnUse`, `OwnUseJobs`, `Cover`, `Secure` | `OnOwnUse` in 5.3.4; **`Blocked` is an event** (`OwnUseChanged`), because it is a count of the last tick that SUB does not save; the per-agent fields are in 5.4 |
| MIG 13.4 | households on a journey from and to a region, the agents with a journey, the last tick's `MigrationReviewed` counts | 5.3.2, 5.3.8; **the counts are the event** |
| MER 15.4 | contracts in flight, the money in escrow | 5.3.8 |
| DEM 11.5 | per region, role and want: households, planned, funded, choked in the last tick; per agent the carries | **the counts are `WantShortfall`**; the carries are in 5.4 (INT 4.7.2) |
| SPN 13.4 | per region and role: households, agents, expected ratio and viability, the privation maximum, the last tick's `EntryLimited` counts; per region the pool and the funding budget | 5.3.2, 5.3.4; **the counts are `EntryLimited`** |
| WEL 11.2 | observed wage and premium, the effective window, the households held by the floor, the mean welfare ratio per role and per region | 5.3.4 |
| SKL 8.2 | places, taken, queue, trainers, observed premium | 5.3.4 |
| AGT 14.6 | households and agents, fed and unfed, the largest hunger, the households held by the floor | 5.3.4 |
| PRO 14.3 | pending output per (region, commodity); agents with a job per recipe | 5.3.5 |
| REF 9 | the mean ratio per role and region; the unsold stock per commodity per region, agents and estate separately; a count of the production halt | 5.3.4; 5.3.5; `ProductionHalted` (4.4.6) |
| WLD 8.1.3, 16.5 | the pending flow totals; the tick and the seed | 5.3.3; 5.3.1 |
| N-031 | the moment of the calendar | `Moment` in 5.3.1 |
| N-040 | whether beliefs are in the snapshot | aggregates per market in 5.3.5; per agent in the query |
| N-109 | whether a run measures the welfare ratio at a full draw | **No.** `r_full` is a counterfactual (what a household would earn at a full draw and a full stock), not a state, so it is not a snapshot field; it is a property of the pack's economy, and the steady-state solver of `spec-viability.md` is the place to compute it (a recommendation to POP and VIA, who own the decision whether a warning `population_cannot_grow` exists and whether the closed forms of `spec-resources.md` 8 read it from the solver). The note stays open for them |
| IND 7.5.1, 7.7, 7.6.2 | per role the last tick's income, relief and cost (`IncomeLast`, `ReliefLast`, `CostLast`); the wear units removed, by commodity; the lines (commodity and units) of `AgentRelocated` | **The income fields are not adopted in version 1.** They would be three `Money` fields of `RoleRow` (a function of the welfare ring, so of saved state, FR-OBS-043) and are a future ask; until then IND's `HouseholdIncome` is `OK` false. **The wear units are adopted** (2026-10-14, N-130 item 2): the aggregate event `WearApplied` of 4.1.5 carries them by commodity. They are a different quantity from the items retired that `wear` of `BoundaryFlow` counts (4.4.1): an item is retired only when its durability reaches 0, so the retirement is lumpy and the wear units are the rate. IND's adoption of the accrual form of its depreciation measure is IND's decision (N-130). `Lines` of `AgentRelocated` would be new and is not needed |
| EMP 15.4, STO 14.5, SET 10.5, LND 13.4, CRD 16.4, CAP 14.5, ORG 11.5, SAV 8.4.2 and 12.5, TEC 11.4, POL 10.5, PLC 12.3, INS 15.4, FIS 16.3 | the rows and fields of the later-stage modules: role, region, market and endowment fields, the in-flight sets `Hires`, `Claims` and `Policies`, and the new slices | **Adopted** as asked, in 5.3.2 to 5.3.6, 5.3.8 and 5.3.11; a count of the last tick of any of them is the module's event (5.2.2); the cost of the saving fields is the new class `Q` (5.5.1, SAV 12.5) |

## 6. Metrics

Metrics are produced by the separate module `otelobserver` from the events of
section 4 and, for fourteen instruments, from a snapshot that the consumer passes to
it (6.2.2). The core defines none and does not import OpenTelemetry
(architecture.md 11.4). This section reconciles the catalogue of rules 5.3 with
the events, resolves the five conflicts of N-088, and says what the adapter keeps.

### 6.1 Principles

6.1.1. **A metric is derived, never a source.** Every instrument is a function of
the events and snapshots of this spec. A value that no event or snapshot carries
is a request to add the event or the field (3.1.4), not to give the adapter a
back door.

6.1.2. **Two feeds.** An instrument is fed by **events** (the 4.1 table's fields)
or by a **snapshot** (5.3). The rule of `rules/observability-and-logging.md` 5.3.1
("a new metric must be derivable from the existing events or come with the event
it needs") is restated in 19 to read "from the events, or from a field of the
snapshot, and the table says which". An instrument fed by a snapshot is
refreshed only when the consumer passes one (6.2.2): between two snapshots it
holds its last value.

6.1.3. **The instrument follows the nature of the value** (rules 5.2): a count of
things that happened is a Counter; a level that moves by steps the events carry
is an UpDownCounter; a level that is read at a point is a Gauge; a duration is a
Histogram. Every instrument has a unit and a one-line description (rules 5.1.5).
All but the duration histogram record `int64` (architecture.md 3.5): prices are
recorded in minor units, unconverted.

6.1.4. **Household and agent counts are never mixed**: every instrument whose
name says `household` counts the sum of weights; `economysim.agent.count` and
`economysim.agent.removals` count agents (N-013).

6.1.5. **Estates and counterparties are not agents.** No instrument counts them
(`spec-estates.md` 2.7.4; N-012).

### 6.2 The catalogue

The rules' catalogue of 5.3 is **kept** except for the rows marked changed. A
metric name or unit that changes here is not a breaking change: no code exists
(rules 5.3 says a change breaks dashboards; there are none).

#### 6.2.1 Fed by events

| Name | Instrument | Unit | Attributes | Fed by | Status |
|------|-----------|------|------------|--------|--------|
| `economysim.world.step.duration` | Histogram | `s` | none | `TickStarted` to `TickEnded`, with the adapter's own clock; **not recorded when `Failed`** (the step did not complete) | kept |
| `economysim.world.step.failed` | Counter | `{step}` | none | `TickEnded` with `Failed` true, one per failed step (6.3.1) | **new**: the duration is not recorded for a failed step and the span is a sampled trace, so without it a world that failed leaves only a flat line |
| `economysim.market.trades` | Counter | `{trade}` | `economysim.commodity` | `TradeExecuted` | kept |
| `economysim.market.volume` | Counter | `{unit}` | `economysim.commodity` | `TradeExecuted` (`Quantity`); equals the sum of `MarketCleared.Volume` (FR-OBS-045) | kept |
| `economysim.market.clearing_price` | Gauge | `{minor_unit}` | `economysim.commodity`, `economysim.region` | `MarketCleared` (`Price`) | **changed**: gains `economysim.region`; a gauge of one commodity across regions has no meaningful last value |
| `economysim.market.price_age` | Gauge | `{tick}` | `economysim.commodity`, `economysim.region` | `MarketCleared` (`PriceAge`) | **new**: `clearing_price` is a held gauge (6.3.6), so a market that stopped trading shows a flat, healthy price; the age is the direct signal, and `market.volume` has no `region` to show which market it is |
| `economysim.market.unmatched` | Counter | `{unit}` | `economysim.commodity`, `economysim.side` | `MarketCleared` (`UnmatchedBid`, `UnmatchedAsk`) | **changed**: was `economysim.market.orders.unmatched` in `{order}`; the event carries units, not orders |
| `economysim.market.orders` | Counter | `{order}` | `economysim.commodity`, `economysim.region`, `economysim.side` | `OrderPlaced` | **new**; `economysim.region` added so that the orders of one market (its book size) can be read, which is what the admission scan of `spec-clearing.md` 3.2 grows with (N-241) |
| `economysim.market.orders.rejected` | Counter | `{order}` | `economysim.commodity`, `economysim.reason` | `OrderRejected` | **new**: the hard budget's refusals, counted (FR-STB-013) |
| `economysim.belief.centre` | Gauge | `{minor_unit}` | `economysim.commodity`, `economysim.region` | `BeliefsUpdated` (`MeanCentre`) | **new** (answers `spec-beliefs.md` 12.4, N-088 3): event-fed, so it needs no snapshot |
| `economysim.belief.at_bound` | Gauge | `{basis_point}` | `economysim.commodity`, `economysim.region`, `economysim.bound` (`floor`, `ceiling`) | `BeliefsUpdated` (`AtFloor`, `AtCeiling`, each `× 10 000 / Outcomes`, truncated toward zero; 3.5); the adapter holds the last value for a tick with no event (6.3.6) | **new**: the share the `Warn` "price beliefs at bound" tests (`spec-beliefs.md` 12.2, row 3 of 7.3), so a consumer with no logger sees it and the mean centre cannot say "converged" from "pinned" |
| `economysim.recipe.executions` | Counter | `{execution}` | `economysim.recipe` | `RecipeExecuted` (`Executions`, not one per event) | kept |
| `economysim.production.halted` | Counter | `{household}` | `economysim.role`, `economysim.recipe` | `ProductionHalted` (`Skipped` rows: a household counts once for each recipe skipped, so the sum over recipes is not the households halted; that figure, `Halted`, is in the event) | **new** (N-067) |
| `economysim.agent.count` | UpDownCounter | `{agent}` | `economysim.role` | `AgentSpawned` (+1), `AgentBankrupted` (-1), `RoleChanged` (-1 `From`, +1 `To`) | kept |
| `economysim.agent.removals` | Counter | `{agent}` | `economysim.role`, `economysim.cause` | `AgentBankrupted` | **renamed** from `economysim.agent.bankruptcies` (6.3.4) |
| `economysim.household.count` | UpDownCounter | `{household}` | `economysim.role` | `AgentSpawned` (+`Weight`), `AgentBankrupted` (-`Weight`), `WeightChanged` (`To` less `From`), `RoleChanged` (-`Weight` `From`, +`Weight` `To`) | kept |
| `economysim.household.unfed` | UpDownCounter | `{household}` | `economysim.role` | `UpkeepShortfall` (`Unfed`), with the adapter's per-(region, role) last value (6.3.1) | kept, rule made exact |
| `economysim.household.exits` | Counter | `{household}` | `economysim.role`, `economysim.cause` (`insolvent`, `starved`, `consumer`) | `AgentBankrupted` (`Weight`) and `WeightChanged` (`From` less `To`, for a reduction); the cause `population` is left out, because `household.deaths` counts it (FR-OBS-059) | **new**: rung 3 of the ladder cuts a cohort by the unfed share and removes it only at weight 0 (`spec-spawner.md` 5.3), so `agent.removals` stays at 0 while households leave, and `household.count` nets the exits against entries |
| `economysim.role.changes` | Counter | `{change}` | `economysim.role` (the destination) | `RoleChanged` (by `To`) | kept |
| `economysim.entry.limited` | Counter | `{case}` | `economysim.entry_limit` | `EntryLimited` (`Unfunded`, `NoViableRole`, `RegionCap`, `RoleCap`, `RoleCeiling`, `Cooldown`, `Hysteresis`, `ExitCap`, `Refused`) | **new** |
| `economysim.want.unfunded` | Counter | `{unit}` | `economysim.want`, `economysim.role` | `WantShortfall` (`Planned` less `Funded`) | kept |
| `economysim.household.relocations` | Counter | `{household}` | `economysim.region.from`, `economysim.region.to`, `economysim.cause` | `AgentRelocated` (`Weight`) | **changed**: gains `economysim.cause`, because a merchant's trips and a migrant's move are not one thing |
| `economysim.money.flow` | Counter | `{minor_unit}` | `economysim.reason`, `economysim.kind` | `BoundaryFlow` (money); includes the bucket at `Tick` -1 (6.3.2) | kept |
| `economysim.commodity.flow` | Counter | `{unit}` | `economysim.reason`, `economysim.kind`, `economysim.commodity` | `BoundaryFlow` (goods) | **new**: production, consumption, spoilage and wear, by commodity |
| `economysim.commodity.wear` | Counter | `{unit}` | `economysim.commodity` | `WearApplied` (`Units`); includes the bucket at `Tick` -1 | **new**: the durability removed from the items of a commodity, a smooth measure of tool depreciation; `economysim.commodity.flow` under `wear` counts only the items retired |
| `economysim.money.flow.unapplied` | Counter | `{minor_unit}` | `economysim.reason` | `MoneyFlowLimited` (`Due` less `Applied`) | kept |
| `economysim.household.births` | Counter | `{household}` | `economysim.role` | `PopulationChanged` (`BirthsDone`) | kept |
| `economysim.household.deaths` | Counter | `{household}` | `economysim.role` | `PopulationChanged` (`DeathsDone`) | kept |
| `economysim.transport.shipments` | Counter | `{shipment}` | `economysim.mode` | `ShipmentDispatched` | kept |
| `economysim.transport.in_transit` | UpDownCounter | `{unit}` | `economysim.commodity` | `ShipmentDispatched` (+ each line), `ShipmentArrived` (- each line's `Shipped`) | kept |
| `economysim.transport.cargo.spoiled` | Counter | `{unit}` | `economysim.commodity` | `ShipmentArrived` (`Spoiled`) | kept |
| `economysim.endowment.drawn`, `.regenerated`, `.short` | Counter | `{unit}` | `economysim.resource`, `economysim.region` | `EndowmentChanged` (`Drawn`, `Regenerated`, `Short`) | kept |
| `economysim.endowment.stock` | Gauge | `{unit}` | `economysim.resource`, `economysim.region` | `EndowmentChanged` (`Stock`), `EndowmentAdjusted` (`StockAfter`); last value held (6.3.6) | kept |
| `economysim.information.rumours` | Counter | `{rumour}` | `economysim.outcome` | `RumoursServed` (`Offered`, `NoPath`, `TooOld`, `NotYet`) | kept |
| `economysim.external.units` | Counter | `{unit}` | `economysim.node`, `economysim.commodity`, `economysim.side` | `OutsideTraded` (`Traded`) | kept |
| `economysim.external.value` | Counter | `{minor_unit}` | `economysim.node`, `economysim.commodity`, `economysim.side` | `OutsideTraded` (`Value`) | kept |
| `economysim.external.price` | Gauge | `{minor_unit}` | `economysim.node`, `economysim.region`, `economysim.commodity`, `economysim.side` | `OutsideTraded` (`Price`) | **changed**: gains `economysim.region` (the gate), since the quote differs by gate scale |
| `economysim.household.own_use` | Gauge | `{household}` | `economysim.role`, `economysim.region` | `OwnUseChanged` (`OnOwnUse`); last value held (6.3.6) | kept |
| `economysim.own_use.units` | Counter | `{unit}` | `economysim.commodity` | `OwnUseChanged` (`Output`, per line) | kept |
| `economysim.migration.started` | Counter | `{household}` | `economysim.region.from`, `economysim.region.to` | `MigrationStarted` (`Weight`) | kept |
| `economysim.migration.limited` | Counter | `{household}` | `economysim.limit` | `MigrationReviewed` (the counts other than `Started`, `Pushed`, `Reviewed`, `Unsettled`) | kept |
| `economysim.migration.on_journey` | UpDownCounter | `{household}` | none | `MigrationStarted` (+`Weight`), `AgentRelocated` with the cause `migration` (-`Weight`), `MigrationReviewed` (-`Ended`) | kept |
| `economysim.contract.formed` | Counter | `{contract}` | `economysim.commodity` | `ContractFormed` | kept |
| `economysim.contract.value` | Counter | `{minor_unit}` | `economysim.commodity` | `ContractFormed` (`Escrowed`) | kept |
| `economysim.contract.escrow` | UpDownCounter | `{minor_unit}` | none | `ContractFormed` (+`Escrowed`), `ContractSettled` (-`Paid`, -`Refunded`) | kept |
| `economysim.merchant.decisions` | Counter | `{lane}` | `economysim.decision` | `MerchantsPlanned` (the case counts) | kept |
| `economysim.storage.limited` | Counter | `{case}` | `economysim.region`, `economysim.role`, `economysim.room_limit` | `RoomLimited` (`StartsCut` as `start_cut`, `StartsRefused` as `start_refused`, `BidsCut` as `bid_cut`, `BidsRejected` as `bid_rejected`, `ShipmentsRefused` as `shipment_refused`) | **new** (STO) |
| `economysim.hire.formed` | Counter | `{hire}` | `economysim.role` (the employer's) | `HireFormed` | **new** (EMP) |
| `economysim.hire.wage` | Counter | `{minor_unit}` | `economysim.region` | `LabourRound` (`Paid`) | **new** |
| `economysim.labour.unmatched` | Counter | `{household}` | `economysim.role` (the worker's) | `LabourRound` (`Workers[].NoVacancy`) | **new** |
| `economysim.vacancy.unfilled` | Counter | `{position}` | `economysim.role` (the employer's), `economysim.reason` | `LabourRound` (`Employers[].Unfilled`, and the refusal fields by reason) | **new** |
| `economysim.hire.escrow` | UpDownCounter | `{minor_unit}` | none | `HireFormed` (+`Escrowed`), `LabourRound` (-`Paid`, -`Refunded`), `HireEnded` (-`Refunded`); the sector `contract` of `economysim.money.supply` includes it | **new** |
| `economysim.settlement.class` | Gauge | `{class}` | `economysim.region` | `SettlementClassChanged` (`To`); the adapter keeps the last value, which the events of construction set | **new** (SET) |
| `economysim.settlement.changes` | Counter | `{change}` | `economysim.region` | `SettlementClassChanged`, except the cause `founded` | **new** |
| `economysim.rent.paid` | Counter | `{minor_unit}` | `economysim.resource`, `economysim.region` | `RentPaid` (`Paid`) | **new** (LND) |
| `economysim.rent.units` | Counter | `{unit}` | `economysim.resource`, `economysim.region`, `economysim.tier` | `RentPaid` (`OwnUnits`, `CommonsUnits`, `LetUnits` as `own`, `commons`, `let`) | **new** |
| `economysim.rent.rate` | Gauge | `{minor_unit}` | `economysim.resource`, `economysim.region` | `RentPaid` (`RentAfter`) and `RentScaled` (`To`); the adapter holds the last values (6.3.6) | **new** |
| `economysim.rent.refusals` | Counter | `{request}` | `economysim.resource`, `economysim.region`, `economysim.reason` | `RentPaid` (`NoTitle`, `RentFunds` as `no_title`, `rent_funds`) | **new** |
| `economysim.title.transfers` | Counter | `{transfer}` | `economysim.resource`, `economysim.region`, `economysim.cause` | `TitleTransferred` (by `Cause`: `removal`, `consumer`) | **new** |
| `economysim.credit.formed` | Counter | `{claim}` | none | `LoanFormed` | **new** (CRD; origin `loan` only) |
| `economysim.credit.principal` | Counter | `{minor_unit}` | none | `LoanFormed` (`Principal`) | **new** |
| `economysim.credit.interest` | Counter | `{minor_unit}` | none | `CreditSettled` (`InterestPaid`) and `LoanEnded` (`PaidInterest`), of origin `loan` | **new** |
| `economysim.credit.outstanding` | UpDownCounter | `{minor_unit}` | none | `LoanFormed` (+`Principal`), `CreditSettled` (-`PrincipalPaid`), `LoanEnded` (-`PaidPrincipal` and `LostPrincipal`), of origin `loan`; seeded at a restore (6.4.2) | **new** |
| `economysim.credit.defaults` | Counter | `{claim}` | none | `CreditSettled` (`Defaulted`) | **new** |
| `economysim.credit.lost` | Counter | `{minor_unit}` | `economysim.cause` | `LoanEnded` (`LostPrincipal` plus `LostInterest`, by `Cause`), of origin `loan` | **new** |
| `economysim.credit.refused` | Counter | `{request}` | none | `CreditRound` (`RequestsRefused` plus `OffersRefused`) | **new** |
| `economysim.credit.limited` | Counter | `{case}` | `economysim.credit_limit` | `CreditRound` (`InArrears`, `Exposure`, `TooMany`, `Overflow` as `in_arrears`, `exposure`, `too_many_claims`, `overflow`; the first two are also in `refused`, so the two instruments are not added) | **new** |
| `economysim.facility.count` | UpDownCounter | `{facility}` | `economysim.facility_class`, `economysim.region` | `FacilityChanged` (+1 for `added` and `built`, -1 for `ruined`; the other kinds change no count) | **new** (CAP) |
| `economysim.facility.limited` | Counter | `{execution}` | `economysim.facility_class`, `economysim.region`, `economysim.facility_limit` | `FacilityLimited` (`Cut` as `cut`, `Voided` as `voided`) | **new** |
| `economysim.facility.refused` | Counter | `{start}` | `economysim.facility_class`, `economysim.region` | `FacilityLimited` (`Refused`) | **new** |
| `economysim.risk.strikes` | Counter | `{strike}` | `economysim.risk_kind` | `ShipmentDispatched` (1 for each with `LossBP` above 0, as `route`), `StoreLoss` (`Stores`, as `store`) | **new** (RSK) |
| `economysim.transport.cargo.lost` | Counter | `{unit}` | `economysim.commodity` | `ShipmentArrived` (each line's `Lost` and each `AccountLost` amount) | **new** |
| `economysim.store.lost` | Counter | `{unit}` | `economysim.commodity`, `economysim.store_class` | `StoreLoss` (`Lost`) | **new** |
| `economysim.fiscal.collected` | Counter | `{minor_unit}` | `economysim.reason` (the fiscal flow ID) | `FiscalFlowApplied` (`Applied` of the kinds `tax`, `duty`, `toll`) | **new** (FIS) |
| `economysim.fiscal.paid` | Counter | `{minor_unit}` | `economysim.reason` | `FiscalFlowApplied` (`Applied` of the kind `payout`, money) | **new** |
| `economysim.fiscal.in_kind` | Counter | `{unit}` | `economysim.reason`, `economysim.commodity` | `FiscalFlowApplied` (`Applied` of an in-kind `payout`) | **new** |
| `economysim.fiscal.unapplied` | Counter | `{minor_unit}` | `economysim.reason` | `FiscalFlowApplied` (`Due` less `Applied` of the fiscal flows that move money) | **new** |
| `economysim.fiscal.order.cut` | Counter | `{unit}` | `economysim.clamp` | `TreasuryOrdered` (`CutCash`, `CutHold`, `CutStock` as `cash`, `hold`, `stock`) | **new** |
| `economysim.right.refusals` | Counter | `{refusal}` | `economysim.right`, `economysim.activity` | `RightRefused` (`Produce`, `Sell`, `Buy`, `Train` as `produce`, `sell`, `buy`, `train`) | **new** (ORG) |
| `economysim.control.clamped` | Counter | `{clamp}` | `economysim.control`, `economysim.side` | `PriceControlled` (`Bids` as `bid`, `Asks` as `ask`) | **new** |
| `economysim.saving.requests` | Counter | `{request}` | `economysim.role`, `economysim.project` | `SavingPlanned` (`Builds` as `build`, `Repairs` as `repair`) | **new** (SAV) |
| `economysim.saving.barred` | Counter | `{household}` | `economysim.role`, `economysim.bar` | `SavingPlanned` (`NoReturn`, `Standing`, `Unreachable`, `Survival` as `no_return`, `standing`, `unreachable`, `survival`) | **new** |
| `economysim.technology.gain` | Gauge | `{basis_point}` | `economysim.field`, `economysim.region` | `TechnologyAdvanced` and `TechnologyAdjusted` (`AfterBP`); the adapter holds the last value for a tick with no event (6.3.6) and seeds the level from the technology rows of `WorldRestored`'s snapshot (6.4.2) | **new** (TEC) |
| `economysim.technology.experience` | Counter | `{execution}` | `economysim.field`, `economysim.region` | `TechnologyAdvanced` (`Executions`) | **new** |
| `economysim.technology.discoveries` | Counter | `{discovery}` | `economysim.field` | `TechnologyAdvanced` (`Discoveries`) | **new** |
| `economysim.technology.clamped` | Counter | `{basis_point}` | `economysim.field` | `TechnologyAdvanced` (`ClampedBP`, the part cut by the cap) | **new** |
| `economysim.policy.decisions` | Counter | `{decision}` | `economysim.actor` | `PolicyDecided` (one per event) | **new** (PLC) |
| `economysim.policy.entries` | Counter | `{entry}` | `economysim.actor`, `economysim.entry_outcome` | `PolicyDecided` (`Applied`, `Unchanged`, `NotHeld`, `Duplicate`, `NoClose` as `applied`, `unchanged`, `not_held`, `duplicate`, `no_close`) | **new** |
| `economysim.policy.cut` | Counter | `{entry}` | `economysim.actor`, `economysim.policy_cut` | `PolicyDecided` (`Clamped`, `Stepped` as `clamped`, `stepped`) | **new** |
| `economysim.policy.held` | Counter | `{reaction}` | `economysim.actor`, `economysim.policy_hold` | `PolicyDecided` (`InBand`, `NoReading` as `in_band`, `no_reading`) | **new** |
| `economysim.insurance.policies` | Counter | `{policy}` | `economysim.cover_kind` | `CoverFormed` (by `Kind`) | **new** (INS) |
| `economysim.insurance.premium` | Counter | `{minor_unit}` | none | `CoverFormed` (`Premium`) | **new** |
| `economysim.insurance.payout` | Counter | `{minor_unit}` | none | `CoverSettled` (`Paid`) | **new** |
| `economysim.insurance.shortfall` | Counter | `{minor_unit}` | none | `CoverSettled` (`Short`) | **new** |
| `economysim.insurance.unpaid` | Counter | `{minor_unit}` | `economysim.cover_loss` | `CoverSettled` (`Dropped`, `Cut` as `dropped`, `cut`) | **new** |
| `economysim.insurance.liability` | UpDownCounter | `{minor_unit}` | none | `CoverFormed` (+`MaxPayout`, route covers only), `CoverEnded` (-`MaxPayout` of a route cover); seeded at a restore (6.4.2) | **new** |
| `economysim.insurance.claims` | UpDownCounter | `{minor_unit}` | none | `CoverEnded` (+`Short`), `LoanEnded` of origin `payout` (-`Principal`, the claim at face; the part paid is `CoverSettled.ClaimPaid`); seeded at a restore (6.4.2) | **new** |
| `economysim.insurance.refused` | Counter | `{request}` | none | `CoverRound` (`RequestsRefused` plus `OffersRefused`) | **new** |
| `economysim.insurance.limited` | Counter | `{case}` | `economysim.cover_limit` | `CoverRound` (`InArrears`, `Exposure`, `NoRisk`, `NoOffer`, `BelowMinimum`, `Funds`, `TooMany`, `Overflow` as `in_arrears`, `exposure`, `no_risk`, `no_offer`, `below_minimum`, `funds`, `too_many`, `overflow`) | **new** |

#### 6.2.2 Fed by a snapshot

The consumer passes a snapshot to the adapter between two steps (a method of the
adapter, `RecordSnapshot(Snapshot)`, which the adapter documents; the world is not
involved, so an observer never calls back into it, architecture.md 12.5). The
adapter records each instrument below once per call, from the rows named.

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.production.pending` | Gauge | `{unit}` | `economysim.commodity`, `economysim.region` | `Markets[].Pending`: the output of jobs in progress, the work in progress that no account holds (N-088 3) |
| `economysim.market.stock` | Gauge | `{unit}` | `economysim.commodity`, `economysim.region`, `economysim.holder` (`agents`, `estate`, `treasury`) | `Markets[].HeldByAgents`, `HeldByEstate`, `HeldByTreasury` (N-081 2) |
| `economysim.ledger.lots` | Gauge | `{lot}` | `economysim.commodity`, `economysim.region` | `Markets[].Lots`: the lots of the commodity in the accounts counted by the three `HeldBy` fields. The sum over the series is the ledger's lot count, which the cost of ageing, of a move, of the audit and of the snapshot's `L` pass grows with (`spec-ledger.md` 5.1, 5.3, NFR-LED-002) |
| `economysim.money.supply` | Gauge | `{minor_unit}` | `economysim.region`, `economysim.sector` (`households`, `pool`, `contract`, `treasury`) | `Regions[].MoneyHouseholds`, `MoneyPool`, `MoneyContract` (hires' escrow included), `MoneyTreasury` |
| `economysim.welfare.ratio` | Gauge | `{basis_point}` | `economysim.role`, `economysim.region` | `Roles[].MeanRatioBP` where `HasMeanRatio` (N-081 1) |
| `economysim.store.count` | Gauge | `{store}` | `economysim.region`, `economysim.store_class` | `Stores[]` rows, by region and store class |
| `economysim.room.fill` | Gauge | `{basis_point}` | `economysim.region`, `economysim.role`, `economysim.dimension` | `Roles[]` room and load, `Load * 10 000 / Room` of a dimension (`mass`, `bulk`), absent where the room is 0 |
| `economysim.credit.arrears` | Gauge | `{minor_unit}` | `economysim.region` | `Claims[]` rows of origin `loan` that are overdue, `PrincipalOwed + InterestOwed`, by region (a level that changes with no `CreditSettled`: a debtor removed in phase 7, `ForgiveClaims`, a relocation) |
| `economysim.facility.pending` | Gauge | `{facility}` | `economysim.facility_class`, `economysim.region` | `Facilities[]` rows, `Pending` (store facilities not installed) |
| `economysim.facility.condition` | Gauge | `{basis_point}` | `economysim.facility_class`, `economysim.region` | `Facilities[]` rows, `ConditionBP` summed over the row |
| `economysim.body.members` | Gauge | `{household}` | `economysim.body`, `economysim.region` | `Bodies[].Members`, by body and region |
| `economysim.body.waiting` | Gauge | `{household}` | `economysim.body`, `economysim.region` | `Bodies[]` (`Eligible - Members`), by body and region |
| `economysim.saving.goal` | Gauge | `{minor_unit}` | `economysim.role`, `economysim.region` | `Roles[].GoalMoney` |
| `economysim.saving.buffer` | Gauge | `{minor_unit}` | `economysim.role`, `economysim.region` | `Roles[].BufferMoney` |
| `economysim.saving.idle` | Gauge | `{minor_unit}` | `economysim.role`, `economysim.region` | `Roles[].IdleMoney` |

An instrument of this table with no snapshot passed is absent, not zero.

### 6.3 The conflicts of N-088, resolved

6.3.1. **Item 1: `economysim.household.unfed` is fed by an event that is absent
when everyone is fed.** The adapter keeps `lastUnfed[region][role]`, an array of
`int64` sized by the names table. On `UpkeepShortfall` it adds
`Unfed - lastUnfed` to the instrument for the role and stores `Unfed`, and marks
the cell seen. At `TickEnded` it adds `-lastUnfed` for every cell it did not see
in the step with a nonzero last value, sets the cell to 0 and clears the marks.
Equivalently the instrument is the sum over regions of the unfed households of
the last step. On `WorldRestored` it is seeded from `Snapshot.Roles[].Unfed`
(6.4). **A failed step.** At a `TickEnded` with `Failed` true the adapter runs
no reset and no other correction: the levels are those after the events it saw
(a step that failed at P4.1 saw none of phases 6 to 8, so `household.unfed` is that of
the last complete step, while the levels that move earlier (`in_transit`, `escrow`,
`on_journey`) already include phases 1 to 4; one that failed its audit at P8.4
saw all of them). The
next world is restored, not continued (`spec-world.md` 3.5), so an adapter
attached to it is seeded again. The failed step itself is counted:
`economysim.world.step.failed` adds 1 at that `TickEnded` (FR-OBS-058).

6.3.2. **Item 2: the construction bucket.** Decided in 4.4.2: `BoundaryFlow` with
`Tick` -1. The adapter records it into `economysim.money.flow` and
`economysim.commodity.flow` like any flow. A consumer who does not want the one-off
spike in a rate graph drops the first collection or the series' first point; the
adapter does not suppress it, because the identity "boundary balance = bucket +
totals" needs it.

6.3.3. **Item 3: gauges fed by the snapshot.** Named in 6.2.2, and the beliefs'
mean centre is event-fed (`economysim.belief.centre`). The conflict with rules
5.3.1 is resolved by 6.1.2.

6.3.4. **Item 4: `economysim.agent.bankruptcies` counts every cause.** Renamed
`economysim.agent.removals`, with the attribute `economysim.cause` taking the four
constants of `AgentBankrupted.Cause` (`insolvent`, `starved`, `consumer`,
`population`). The name was misleading because a removal by the population module
is a death and one at the consumer's request is neither. A dashboard that wants
only the failures sums `insolvent` and `starved`.

6.3.5. **Item 5: whether `ReliefPaid` and `EstateOpened` belong to a module that
can be off.** `EstateOpened` is emitted by every removal, with the pool module on
or off, because the estate account and pseudo-agent exist in every world
(`spec-regions.md` 3.6.2, `spec-world.md` 5.3): it is **core**. `ReliefPaid` is
the pool policy's (relief is paid only when the estates module's pool policy is
on, `spec-estates.md` 2.6): it is **gated by that module**. No instrument of 6.2
is fed by either; both are in the event table for their own consumers, and a
dashboard of relief is `ReliefPaid.Paid` summed.

6.3.6. **The gauges that hold a last value** (`economysim.endowment.stock`,
`economysim.household.own_use`, `economysim.external.price`,
`economysim.market.clearing_price`, `economysim.belief.centre`, `economysim.belief.at_bound`, `economysim.rent.rate`,
`economysim.settlement.class`, `economysim.technology.gain`): the adapter keeps
the last recorded value per series and re-records every held series at
`TickEnded`, so that a series whose event was silent in a tick is not stale at
collection. The cost is bounded by the number of series of the gauge, never by
agents.

### 6.4 Levels, and how an adapter built after a restore gets them

6.4.1. **A level instrument** (an UpDownCounter, or a Gauge that holds a last
value) is a running sum of deltas that events carry. It is correct only if the
adapter saw every delta from the start of the world's life, which is true of an
adapter attached at `NewWorld` (it sees `WorldCreated` and every
`AgentSpawned`) and false of one attached at a restore, which sees only the
events after it.

6.4.2. **`WorldRestored` carries the levels** as the snapshot at the restored
boundary (4.4.5). The adapter seeds its levels from it, once:

| Instrument | Seed from |
|------------|-----------|
| `economysim.agent.count` | `Roles[].Agents` by role |
| `economysim.household.count` | `Roles[].Households` by role |
| `economysim.household.unfed` | `Roles[].Unfed` by (region, role), and `lastUnfed`; it may differ from the original adapter's level by what P7.1 did, until the end of the next complete step, from which the level is a function of that step's events alone |
| `economysim.transport.in_transit` | the sum of `Shipments[].Lines` by commodity |
| `economysim.migration.on_journey` | the sum of `Journeys[].Weight` |
| `economysim.contract.escrow` | `Escrow` |
| `economysim.endowment.stock` | `Endowments[].Stock` |
| `economysim.household.own_use` | `Roles[].OnOwnUse` |
| `economysim.external.price` | `Outside.Sides[].Quote` |
| `economysim.market.clearing_price` | `Markets[].State.Price` |
| `economysim.hire.escrow` | the sum of `Hires[].Held` |
| `economysim.facility.count` | the sum of `Facilities[].Facilities` by (class, region) |
| `economysim.settlement.class` | `Regions[].Class` |
| `economysim.rent.rate` | `Endowments[].Posted` and `ScaleBP` (the rent in force) |
| `economysim.credit.outstanding` | the sum of `PrincipalOwed` over the `Claims[]` rows of origin `loan`, equal to `Credit.Lent - Credit.Repaid - Credit.LostPrincipal` |
| `economysim.technology.gain` | `Tech[].GainBP`, a cell with no row being 0 |
| `economysim.insurance.liability` | the sum of `Policies[].MaxPayout` |
| `economysim.insurance.claims` | the sum of `PrincipalOwed` over the `Claims[]` rows of origin `payout` |

`economysim.belief.centre` and `economysim.belief.at_bound` are **not** seeded: `Markets[].MeanCentre` is over all
beliefs and `BeliefsUpdated.MeanCentre` over those processed in the tick (BEL 12.1),
so a series starts at the first event after the restore. The sparse held gauges (`endowment.stock`, `household.own_use`, `external.price`, `rent.rate`, `settlement.class`, `technology.gain`) are seeded for every key, whereas the original adapter has a series only for the keys that emitted, and `external.price` is seeded from the current quote where the original holds the last traded limit: they agree from the first event of each series, not before. A Counter is **not** seeded: it counts what this adapter saw, and a restore
starts it at 0, as a counter does when a process restarts.

6.4.3. **Why a snapshot and not a list of levels.** A separate "levels" value
would repeat 5.3 under another name and diverge from it. The cost is that the
snapshot is built once for a restore when an observer was supplied (8.2), which
is once per restore.

### 6.5 Attributes and cardinality

6.5.1. **The attributes** of rules 5.4.1 stand. This section adds four and
extends the use of a fifth, and lists the attributes of the later modules (the rules
name them in 5.4.1). Each is bounded by the configuration or by a constant set:

| Attribute | Values | Added for |
|-----------|--------|-----------|
| `economysim.cause` | the removal causes `insolvent`, `starved`, `consumer`, `population` (`economysim.agent.removals`) and the relocation causes `transport`, `migration`, `consumer` (`economysim.household.relocations`); and, for `economysim.household.exits`, `insolvent`, `starved` and `consumer`; a closed set of constants (4.2.3) | removals, relocations, exits |
| `economysim.entry_limit` | `unfunded`, `no_viable_role`, `region_cap`, `role_cap`, `role_ceiling`, `cooldown`, `hysteresis`, `exit_cap`, `refused` (nine) | `economysim.entry.limited` |
| `economysim.holder` | `agents`, `estate`, `treasury` | `economysim.market.stock` |
| `economysim.sector` | `households`, `pool`, `contract`, `treasury` | `economysim.money.supply` |
| `economysim.bound` | `floor`, `ceiling` | `economysim.belief.at_bound` |
| `economysim.region` | the regions of the configuration (an existing attribute of RES) | `economysim.market.clearing_price`, `economysim.belief.centre`, `economysim.external.price`, the snapshot gauges |
| `economysim.store_class`, `economysim.facility_class`, `economysim.body`, `economysim.right`, `economysim.control`, `economysim.field`, `economysim.actor` | the entities of the configuration of the module (store classes, facility classes, bodies, rights, controls, technology fields, policy actors) | STO, CAP, ORG, TEC, PLC |
| `economysim.dimension` | `mass`, `bulk` | `economysim.room.fill` |
| `economysim.room_limit` | `start_cut`, `start_refused`, `bid_cut`, `bid_rejected`, `shipment_refused` | `economysim.storage.limited` |
| `economysim.tier` | `own`, `commons`, `let` | `economysim.rent.units` |
| `economysim.credit_limit` | `in_arrears`, `exposure`, `too_many_claims`, `overflow` | `economysim.credit.limited` |
| `economysim.facility_limit` | `cut`, `voided` | `economysim.facility.limited` |
| `economysim.risk_kind` | `route`, `store` | `economysim.risk.strikes` |
| `economysim.clamp` | `cash`, `hold`, `stock` | `economysim.fiscal.order.cut` |
| `economysim.activity` | `produce`, `sell`, `buy`, `train` | `economysim.right.refusals` |
| `economysim.project`, `economysim.bar` | `build`, `repair`; `no_return`, `standing`, `unreachable`, `survival` | `economysim.saving.requests`, `economysim.saving.barred` |
| `economysim.entry_outcome`, `economysim.policy_cut`, `economysim.policy_hold` | `applied`, `unchanged`, `not_held`, `duplicate`, `no_close`; `clamped`, `stepped`; `in_band`, `no_reading` | `economysim.policy.entries`, `.cut`, `.held` |
| `economysim.cover_kind`, `economysim.cover_loss`, `economysim.cover_limit` | `route`, `store`; `dropped`, `cut`; `in_arrears`, `exposure`, `no_risk`, `no_offer`, `below_minimum`, `funds`, `too_many`, `overflow` | `economysim.insurance.*` |

`economysim.reason` takes, besides the money flow IDs and the built-in boundary
reasons of rules 5.4.1, the **rejection reasons** of `OrderRejected` (twelve
constants, `spec-agents.md` 14.2, and `not_permitted` and `no_room` of the later
modules) for `economysim.market.orders.rejected`, the goods boundary reasons of 4.4.1
for `economysim.commodity.flow`, the **fiscal flow IDs** (which share the namespace
of the money flow IDs) for `economysim.fiscal.*`, the refusal reasons of a vacancy
(`employer_funds`, `no_input`, `no_tool`, `no_endowment`, `no_room`, `zero_yield`,
`no_site`, `too_many_hires`, `no_labour`) for `economysim.vacancy.unfilled` and of a
rent request (`no_title`, `rent_funds`) for `economysim.rent.refusals`. `economysim.cause`
also takes, per instrument, the closed set of the event that feeds it (`removal` and
`consumer` for `economysim.title.transfers`; `repaid`, `written_off`,
`debtor_removed` and `forgiven` for `economysim.credit.lost`). No attribute is an
agent ID, an order ID, a tick, a price, a quantity, a contract or a shipment ID
(rules 5.4.1).

6.5.2. **Series for the reference world**, from the placeholder sizes of N-030 and
`spec-reference-chain.md` 4.4.4 (5 regions, 30 commodities, 45 recipes, **20 roles**),
`spec-resources.md` 18 (3 resources), 3 modes (a placeholder), one outside node with
2 gates, 6 wants and 4 money flows (placeholders). The rule is the **product** of
the attribute sets, every cell that could occur:

| Group | Series |
|-------|-------|
| market (trades 30, volume 30, unmatched 30 x 2, orders 30 x 2, rejected 30 x 10, clearing price 5 x 30, belief centre 5 x 30, price age 5 x 30, beliefs at bound 5 x 30 x 2) | 1 230 |
| goods flow (14 goods reasons x 2 kinds x 30 commodities) | 840 |
| production (executions 45, halted 20 roles x 45 recipes) | 945 |
| agents and households (count 20, removals 20 x 4, household count 20, unfed 20, changes 20, births 20, deaths 20, exits 20 x 3) | 260 |
| wants (6 x 20) and entry (9) | 129 |
| money (flow 8 reasons x 2 kinds, unapplied 4; `loss` is a goods reason, 4.4.1) and relocations (20 ordered pairs x 3 causes) | 80 |
| transport (3 + 30 + 30), contracts (30 + 30 + 1) and merchants (11) | 135 |
| endowment (4 x 15), own use (20 x 5 + 30), outside (units and value 30 x 2 each, price with 2 gates 120), information 4, migration (20 + 12 + 1) | 467 |
| snapshot-fed (pending 150, stock 300, money supply 15, welfare ratio 100) | 565 |
| duration and failed steps | 2 |
| the instruments of the later modules (6.2.1 from `economysim.storage.limited`, and 6.2.2 from `economysim.store.count`) | not counted: no reference pack has their sizes yet; the product rule applies to each, and each spec counts its series for its own placeholder sizes (LND 13.3: 90) |
| **Total** | **about 4 700** (4 653), for the instruments of the earlier modules |

This is a ceiling of cells that could occur by the product rule: most pairs (a
reason with a commodity, a role with a recipe it does not have) never occur, so
the series that exist are a small fraction. The point is that it does not depend
on the number of agents or households, and that the adapter records each series
**once per tick** (6.6.2).

### 6.6 The adapter

6.6.1. **What it must keep.** State bounded by the configuration, and nothing
else: the names table (from `WorldCreated` or `WorldRestored`); one `int64` per
series of each UpDownCounter and each held Gauge; `lastUnfed[region][role]` and its
marks (6.3.1); the start of the open step and of the open run, in its own clock;
and the per-tick accumulators of 6.6.2.

6.6.2. **What it must do with the volume classes.** A class-O or class-R event
(`OrderPlaced`, `OrderRejected`, `TradeExecuted`) arrives tens of thousands of
times a tick (8.3). The adapter does **not** call an instrument for each: it adds
into an array indexed by the typed IDs (commodity, side, reason) and records each
non-zero cell once, at `TickEnded`. The OpenTelemetry SDK is then called at most
as often as there are series (6.5.2), not as often as there are orders. The same
holds for any class-A or class-E event that feeds a Counter. A class-C or
class-W event is recorded at once.

6.6.3. **What it must not keep.** A table of agents, orders, trades, shipments or
contracts; any value keyed by an `AgentID` (an agent ID is unbounded, rules
5.4.1); a reference to an event's slice beyond the call that received it (the
slices are the event's, and a recorder keeps them, but an adapter has no use for
them); and anything that depends on the wall clock other than the duration of a
step.

6.6.4. **What it must not do.** Call the `World` (architecture.md 12.5); read a
global provider (it uses the `TracerProvider` and `MeterProvider` given to it,
rules 6.3); block or start a goroutine on the stepping path; panic on an ID out
of the names table (it drops the sample: such an ID is a library defect that the
cardinality test of 9.4 would have caught); and put anything unbounded in an
attribute.

6.6.5. **Traces** (rules 6): a span for each `Run` from `RunStarted` to
`RunEnded` and one for each step from `TickStarted` to `TickEnded`, with
`economysim.tick` as an attribute; a step span ends with an error status when
`TickEnded.Failed`; a run span likewise when `RunEnded.Failed`. A class-E event emitted **between `TickStarted` and `TickEnded`**
is a span event of the step span (`AgentBankrupted` of phase 7, `OutsideEnded`,
and the others) with its small fields as attributes, since their number is the
number of rare transitions. An event emitted **outside a step** (a class-C slot
event, a call's `AgentSpawned`, `AgentBankrupted`, `AgentRelocated` or
`RecipeAbandoned`, the bucket at `Tick` -1, `WorldCreated`, `WorldRestored`) has
no step span to attach to: it is counted or logged, not a span event, and an
adapter that wants it in a trace attaches it to the run span if one is open. A class-O or
class-R event is **not** a span event: the step span carries the *counts* of
orders placed, rejected and trades and the traded volume as attributes, from the
accumulators of 6.6.2. This refines rules 6.2, which names "a rejected order" as
a span event: a rejected order is class O and would be tens of thousands of span
events a tick. `WorldCreated` and `WorldRestored` are not span events; they seed
the adapter's state.

6.6.6. **Instrumentation scope** is the module path of `otelobserver` (rules 6.3).
Without an SDK the API is a no-op, so the adapter is safe to wire unconditionally;
its per-event cost then is the accumulator add, which is a few instructions.

## 7. Logs

The rules fix the mechanics and the levels (`rules/observability-and-logging.md`
3). This section collects every `Warn` and `Info` path of the written specs, the
keys they use and the bound on their volume, and says what the rules' key table
lacks. A log line is never the only record of a fact (rules 2.1): every `Warn`
has an event or a snapshot counterpart (7.4), and the test of 9.2 checks it.

### 7.1 The levels in use

| Level | Paths | Where |
|-------|-------|-------|
| `Error` | **none**. No written spec has an inconsistency that the library repairs and survives; a violated internal invariant panics with a message that says so (architecture.md 8.1) or, with `WithAudit`, returns `ErrAuditFailed` | |
| `Warn` | 25 messages, 7.3 | construction (configuration warnings) and the tick (bounded by an entity of the configuration: a market, role, region, endowment, body, side or, row 19, recipe) |
| `Info` | 5 messages, 7.2 | construction, `Run`, a restore; **none inside a step** |
| `Debug` | per entity, guarded by `Enabled`, 7.5 | the tick |

7.1.1. **A new `Error` path** needs the justification of rules 3.2: an internal
inconsistency that the library detected, repaired and can safely continue from,
in a method with no error result. Version 1 has none, and a spec that adds one
says so in its Observability entry.

### 7.2 `Info`

| Message | Keys | When | Owner |
|---------|------|------|-------|
| "world created" | `seed`, `config_hash`, `commodities`, `recipes`, `roles`, `regions`, `agents`, `households`, `modules`, `warm_start`, and for the modules that are on `prices`, `wages`, `groups`, `jobs`, `edges`, `components`, `nodes`, `counterparties`, `modes`, `merchants`, `store_classes`, `settlement_classes`, `facility_classes`, `facilities`, `fields`, `jurisdictions`, `governed`, `fiscal_flows`, `bodies`, `rights`, `controls`, `actors`, `levers`, `lenders`, `borrowers`, `underwriters`, `insured`, `rows` | once, at the end of construction | WLD 5.8.1, CFG 10.1 |
| "warm start applied" | `regions`, `prices`, `wages`, `groups`, `jobs` | once, at construction, with a warm start | VIA 17.2 |
| "run started" | `ticks` | `Run`, once | WLD 16.3 |
| "run ended" | `ticks` (completed), `reason` (`completed`, `cancelled`, `failed`) | `Run`, once; the error is returned, not logged | WLD 16.3 |
| "world restored" | `tick`, `seed`, `config_hash`, `agents`, `households`, `format`, `bytes` | a restore, once, after the configuration's warnings | CHK 6.7.4, 14.2 (this spec lists it and the two new keys) |

None is in the tick loop (rules 3.2.2). "Run started" and "run ended" are
emitted whenever `RunStarted` and `RunEnded` are, so the two never disagree.

### 7.3 `Warn`

**Run rule.** Every `Warn` of the tick that is about a condition that persists
is emitted by one of two patterns over the **run** of consecutive ticks in which
the condition held, a count its owner keeps and saves (diagnostic: it affects no
result, and is saved so that a resumed run logs as the original did; rows 8 and 9 need a run that SPN 14 and POP 11 do not save yet, section 19):

- **pattern B** (the tenth): when the run reaches 10 and at every multiple of 10
  after;
- **pattern A** (the first and the tenth): when the run is 1 and at every
  multiple of 10 after.

The 10 is one **rule constant**, `warnRun` (unexported: it is not part of the API, and earlier drafts wrote `WarnRun`), a constant of the root
package (`observer.go`; it is not a key and not a reason, so not in
`internal/logkeys`), passed as a parameter to any `internal/` package that keeps a
run (architecture.md 2.9), shared by every
owner (architecture.md 9.6: a rule constant beside the code that owns it, not a
tunable, since it changes no result). A tick in which the condition does not hold
resets the run, so a condition that flaps fires pattern A again at each new run: **the
bound of 7.3.1 is per uninterrupted run**. The two patterns are the owners' and are kept; the table says
which each uses.

| # | Message | Pattern, condition | Keys | Cells bounded by | Counterpart (rules 2.1) | Owner |
|---|---------|--------------------|------|------------------|-------------------------|-------|
| 1 | "configuration warning" | once per warning, at construction and at a restore | `rule`, `path` | the configuration | `Warnings()` (INT 6.4) | CFG 10.2 |
| 2 | "households unfed" | B: a nonzero `Unfed` in the (region, role) | `region`, `role`, `households`, `unfed` | regions x roles | `UpkeepShortfall`; `RoleRow.Unfed` | AGT 14.3 |
| 3 | "price beliefs at bound" | B: more than half of a market's outcomes ended on a bound (`BoundRun`) | `region`, `commodity`, `bound`, `agents` | markets | `BeliefsUpdated.AtFloor`, `AtCeiling`; `MarketRow` | BEL 12.2 |
| 4 | "work floor binding" | B: more than half of a role's households held by `MinWorkShare` | `region`, `role`, `households` | regions x roles | `RoleRow.HeldByFloor` (the same quantity evaluated at the boundary, which may differ from the count the `Warn` used, 5.3.4) | WEL 11.3 |
| 5 | "no places open" | B: a role with applicants and no place open because `gain` is negative | `region`, `role`, `reason` | regions x roles | `RoleRow.Places`, `Queue` | SKL 8.3 |
| 6 | "role extinct" | A: a role in a region that had at least one agent or has a configured group and has none; a closed role that only external agents fill is not extinct (FR-SPN-042) | `region`, `role` | regions x roles | `RoleRow.Agents` | SPN 13.3 |
| 7 | "entry unfunded" | B: a region whose pool funded no entrant while entrants were due | `region`, `households` | regions | `EntryLimited.Unfunded` | SPN 13.3 |
| 8 | "group below floor" | A (SPN gives no run; this spec reads it so, 7.3.3): at run time, never at construction, a group with a weight above 1 whose agent count has fallen below `MinAgents` (FR-SPN-047) | `region`, `role`, `agents` | regions x roles | `RoleRow.Agents` | SPN 13.3 |
| 9 | "region extinct" | A: a region with no households | `region` | regions | `RegionRow.Households` | POP 10.3 |
| 10 | "births unfunded" | A (POP 10.3: "first tick of the condition and every tenth after"): births due and the pool funded none | `region`, `births` | regions | `PopulationChanged.BirthsUnfunded` | POP 10.3 |
| 11 | "deaths held" | A (POP 10.3): `MinHouseholds` held deaths in a group | `region`, `role`, `deaths` | regions x roles | `PopulationChanged.DeathsHeld` | POP 10.3 |
| 12 | "endowment exhausted" | A: `Short` above 0; for an endowment with tenure data `Short - min(Short, Held + Idle)` above 0 (LND 13.1.4) | `tick`, `region`, `resource`, `stock`, `short` | endowments | `EndowmentChanged.Short`; `RentPaid.Held`, `Idle`; `EndowmentRow` | RES 14.2, LND 13.1.4 |
| 13 | "outside side ended" | once per side, when it ends | `node`, `commodity`, `side`, `cause` | sides | `OutsideEnded` | EXT 15.2 |
| 14 | "role on own use" | B: more than half of a role's households in the mode | `region`, `role`, `households`, `own_use` | regions x roles | `OwnUseChanged.OnOwnUse`; `RoleRow.OnOwnUse` | SUB 14.2 |
| 15 | "own use blocked" | B: at least one household in the mode blocked | `region`, `role`, `households`, `reason` | regions x roles | `OwnUseChanged.Blocked` | SUB 14.2 |
| 16 | "migration blocked" | B: households pushed in each tick and none started | `region`, `households`, `reason` | regions | `MigrationReviewed` | MIG 13.2 |
| 17 | "arrivals held" | B: a held arrival in each tick | `region`, `households` | regions | `MigrationReviewed.Held` | MIG 13.2 |
| 18 | "wants choked" | B: all of a role's households choked for the want | `region`, `role`, `want`, `households` | regions x roles x wants | `WantShortfall.Choked` | DEM 11.2 |
| 19 | "recipe output disposed" | **no run**: the overflow rule of PRO 5.3.2, at most one per recipe per tick (PRO 14.2), exempt from FR-OBS-071 and FR-OBS-072 and bounded by the recipes of the configuration | `recipe`, `commodity`, `quantity` | recipes | `RecipeExecuted.Disposed` | PRO 14.2 |
| 20 | "labour unmatched" | B: more than half of a worker role's offered households in a region were unmatched in each tick of the run | `region`, `role`, `households` | regions x roles | `LabourRound.Workers[].NoVacancy`; `RoleRow.Employed` | EMP 15.2 |
| 21 | "vacancies unfilled" | B: positions of an employer role in a region were unfilled in each tick of the run | `region`, `role`, `vacancies`, `reason` | regions x roles | `LabourRound.Employers[].Unfilled` and the refusal fields | EMP 15.2 |
| 22 | "tenure refusal" | A: requests of an endowment were refused for a reason (`no_title` or `rent_funds`) in the tick, the run of such ticks (saved by LND) being 1 or a multiple of 10 | `tick`, `region`, `resource`, `rent`, `agents`, `reason` | endowments x 2 reasons | `RentPaid.NoTitle`, `RentFunds` | LND 13.2 |
| 23 | "claims overdue" | B: the debtors of a region owe overdue claims of origin `loan` at the end of P1.7a | `region`, `claims`, `arrears` | regions | `CreditSettled.Arrears`; the `Claims` rows (`Overdue` at or above 0) | CRD 16.2 |
| 24 | "claim cap reached" | A: a request was cut because `MaxClaims` open claims exist | `claims` | the world | `CreditRound.TooMany`; the length of `Claims` | CRD 16.2 |
| 25 | "body has waiting households" | B: the households waiting for a place of a body in a region were above 0 in each tick of the run | `region`, `body`, `waiting`, `places` | regions x bodies | `MembershipChanged.Waiting`; `BodyRow` (`Eligible - Members`) | ORG 11.2 |

7.3.1. **The bound.** Lines 2 to 18 and 20 to 25 each have a cell per entity of the
configuration and emit at most once per `warnRun` ticks per cell **within an uninterrupted run**
(pattern A adds the first tick of a run), so on average a tick emits the number of
cells over `warnRun`, and at most the number of cells in the worst tick. Line 19 is bounded per tick by
the number of recipes, and line 13 by the number of sides over the whole run. For
the reference world (placeholder sizes of 6.5.2: 5 regions, 20 roles, 150 markets,
15 endowments, 6 wants) the cells of lines 2 to 18 are about 1 700 (eight kinds
over 100 (region, role) cells, 600 of line 18, 150 markets, 25 regions, 15
endowments, 120 sides of 1 node, 2 gates, 30 commodities and 2 sides), and those of
lines 20 to 25 about 250 more (200 of lines 20 and 21 over the 100 (region, role)
cells, 30 of line 22, 5 of line 23, 1 of line 24 and, for a placeholder of two
bodies, 10 of line 25), so about 1 950. **The per-tick bound is the number of cells**: every run
can start in the same tick (a famine, a shock, all groups at construction) and
pattern A fires on the first tick, so one tick may emit about 1 950 lines, plus
the 45 of line 19. **Over a window of `w` ticks** the bound is the cells times `ceil(w / 2)` for pattern A
(a condition true on alternate ticks starts a new run every other tick) and
`ceil(w / warnRun)` for pattern B, which is unbroken-run bound only; about 170 a
tick is the mean, not the bound. The volume does not depend on the number of agents (rules 3.2.2, 10).

7.3.2. **A `Warn` is actionable** (rules 3.2.4): each names where (`region`,
`role`, `commodity`, `resource`, `node`), what (the message) and how many (the
count key). The condition behind each is a **counter that exists as an event or a
snapshot field**, column "Counterpart", so a consumer who turned the logger off
loses the line and not the fact.

7.3.3. **Line 8 and its run.** SPN lists "group below floor" without a run
pattern. It is a condition on a count of agents that changes only at a removal or
entry, so the rule is: emitted when a group's agents first fall below `MinAgents`
and at every multiple of `warnRun` ticks it stays below; it is pattern A.

### 7.4 Every `Warn` has a counterpart: the check

The column "Counterpart" of 7.3 is a table a test can read (FR-OBS-051): for each
message, the named event or snapshot field exists in sections 4 and 5. A spec
that adds a `Warn` adds a row, with its counterpart or the reason there is none
(observability-and-logging.md 9, item 1).

### 7.5 `Debug`

7.5.1. Every `Debug` line is guarded by `logger.Enabled(ctx, slog.LevelDebug)`
when building its attributes costs anything (rules 3.1.3), has a constant message
and uses the keys of 7.6. With the default logger the guard is the only cost.

7.5.2. The lines are the owners' and are listed in their observability sections;
this spec adds none except WLD's "tick ended" (`tick`, `agents`, `households`,
`spec-world.md` 16.3). By owner: CLR ("order rejected", "market cleared"), BEL
("belief updated"), AGT ("work declined", "intent dropped", "upkeep unmet",
"discard refused"), PRO ("recipe not started"), SPN ("agent bankrupt", "role
changed", "decision refused", "entry dropped"), POP ("population step"), MON
("money flow limited", "money flow ended", "money flow scaled"), REG ("agent
relocated", "relocation refused", "edge status changed"), TRN ("shipment
dispatched", "shipment arrived", "shipment refused", "cargo diverted"), RES
("draw rationed"), INF ("rumours served"), EXT ("outside side", "outside drift"),
SUB ("own use entered", "own use left", "own use retry", "own use blocked"), MIG
("migration started", "migration declined", "arrival held"), MER ("contract
formed", "contract settled", "offer refused", "take refused", "trip planned"),
DEM ("want skipped", "want consumed"), SKL ("traineeship refused"), LED (a
refused transfer), CHK ("checkpoint taken": `tick`, `bytes`), INT ("agent added", "directive set", "agent removed", and one constant message per setter: "yield shocks set", "edge statuses set", "transit factors set", "flow scales set", "birth scales set", "death scales set", "capacity scales set", "regen scales set", "endowments adjusted", "outside shocks set", with `tick` and `cause`: `spec-integration.md` 12.2), and, for the later modules, EMP ("offer refused", "vacancy refused", "hire refused"), STO ("bid cut for room", "job refused for room", "store added", "store removed"), SET ("settlement class changed", "settlement class held"), LND ("rent charged"), CRD ("loan formed", "loan ended", "claim passed", "claim overdue", "request refused", "offer refused"), CAP ("facility built", "facility repaired", "facility ruined", "facility installed", "facility start refused"), RSK ("route struck", "store struck"), FIS ("fiscal flow applied", "treasury ordered"), ORG ("start refused" and "order rejected" with the reason `not_permitted`, "place not open"), SAV ("project chosen", "project declined"), TEC ("technology advanced", "technology adjusted"), PLC ("policy decided", "policy entry refused", "rule moved lever") and INS ("cover formed", "cover ended", "cover refused"); POL has none.

7.5.3. A `Debug` line that records a *fact* is also an event or a snapshot field
(a rejected order, a relocation, a contract): the line is the human's view of it.
A `Debug` line that records a *decision's inputs* (a work declined, an intent
dropped) is not an event, because its volume would be per agent per tick
(class O or worse, 3.3.9); it is the one place where a per-agent explanation
lives, and its counterpart in aggregate is `ProductionHalted` for the halt and
`EntryLimited` for the spawner's.

### 7.6 The log keys against the rules

`rules/observability-and-logging.md` 3.3.2 lists the standard keys. Compared with
every key used in the observability sections of the written specs, **every key is
in the rules' table except four**: two that the checkpoint spec introduces and two that
MON's Debug line "money flow scaled" uses (`from` and `to`, which are not the
rules' `from_region` and `to_region`):

| Key | Type | Meaning | In the rules' table | Used by |
|-----|------|---------|---------------------|---------|
| `format` | `int64` | the checkpoint format version | **no: add** | "world restored" (CHK) |
| `bytes` | `int64` | the size of an encoded checkpoint | **no: add** | "world restored", "checkpoint taken" (CHK) |
| `from`, `to` | `int64` | a value before and after, in basis points (a flow scale) | **no: add**, or MON renames them (routed, section 19) | "money flow scaled" (MON 13.2) |

The rules' table has since gained these four, `account` and `amount` (the line of a
refused transfer, LED 12.2), and the keys of the later modules, which the table
below lists with the lines that use them.

The keys in the rules' table and where each is used (the counts are of
messages in 7.2, 7.3 and 7.5, for the reader who wants to know whether a key is
dead):

| Key | Used by |
|-----|---------|
| `tick`, `ticks`, `seed` | "world created", "run started", "run ended", "tick ended", "endowment exhausted" |
| `commodity`, `role`, `recipe`, `agent`, `price`, `quantity`, `side`, `region` | the Debug lines of CLR, BEL, AGT, PRO, SPN and the Warns of lines 3, 4, 6, 8, 11 to 19 |
| `reason` | every refusal line, "agent bankrupt", "role changed", "contract settled"; "run ended" (`completed`, `cancelled`, `failed`) |
| `error` | none in the library (an error is returned, not logged: architecture.md 8.3) |
| `bound` | "price beliefs at bound" |
| `agents`, `households`, `unfed` | "world created", "tick ended", the unfed and floor Warns |
| `path`, `rule`, `config_hash` | "configuration warning", "world created", "world restored" |
| `prices`, `wages`, `groups`, `jobs` | "warm start applied" and "world created" |
| `cause` | "agent relocated", "edge status changed", "money flow ended", "outside side ended" |
| `want` | "want skipped", "wants choked" |
| `commodities`, `recipes`, `roles`, `regions` | "world created" (counts); "warm start applied" (`regions`) |
| `modules`, `warm_start` | "world created" |
| `edges`, `components`, `from_region`, `to_region`, `distance`, `closed` | REG's lines and the construction summary |
| `flow`, `due`, `applied` | MON's lines |
| `births`, `deaths` | POP's lines |
| `shipment`, `mode`, `dest`, `modes` | TRN's lines and the construction summary |
| `resource`, `stock`, `short` | RES's lines |
| `rumours`, `lag` | INF's and MIG's lines |
| `node`, `value`, `nodes`, `counterparties`, `drift` | EXT's lines and summary |
| `own_use` | SUB's Warn |
| `contract`, `offers`, `takes`, `merchants` | MER's lines and summary |
| `store_class`, `store_classes` | STO's lines and the summary (the key `class` of the first draft is `store_class`, so as not to collide with a commodity class or a settlement class) |
| `settlement_class`, `from_settlement_class`, `to_settlement_class`, `target_settlement_class`, `settlement_classes` | SET's lines and the summary |
| `vacancies` | the "vacancies unfilled" Warn (row 21) |
| `rent`, `share` | LND's "tenure refusal" and "rent charged"; RSK's "route struck" and "store struck" |
| `claim`, `claims`, `lenders`, `borrowers`, `rate`, `term`, `principal`, `owed`, `arrears`, `lost` | CRD's lines and summary, and rows 23 and 24 |
| `facility`, `facility_class`, `facilities`, `facility_classes`, `condition` | CAP's lines and summary |
| `rows` | RSK's summary |
| `authority`, `fiscal_flows` | FIS's lines and summary |
| `body`, `right`, `control`, `activity`, `places`, `waiting`, `bodies`, `rights`, `controls` | ORG's lines, summary and row 25 |
| `goal`, `buffer` | SAV's "project chosen" |
| `field`, `fields`, `gain` | TEC's lines and summary |
| `actor`, `lever`, `reading`, `entries`, `changed`, `actors`, `levers` | PLC's lines and summary |
| `cover_policy`, `sum`, `premium`, `payout`, `underwriters`, `insured` | INS's lines and summary (the key `policy` of the first draft is `cover_policy`, so as not to collide with PLC's module, cause and `Policy`) |
| `jurisdictions`, `governed` | POL's summary |

7.6.1. **Rules about keys** that this spec restates for the owners: a key is
lower-case `snake_case`, defined once as a constant in `internal/logkeys` and
reused; a count of agents and a count of households are different keys; money,
price and quantity are logged with `slog.Int64`, never as a float (rules 3.3.2);
a key is added to the rules' table in the change that introduces it; log keys are
not metric attribute names (rules 5.1.1), and the adapter maps `commodity` to
`economysim.commodity` and so on.

7.6.2. **The values of `reason` and `cause`** are the constants of the events'
fields (4.2.3), so a log filter and a metric attribute use the same strings. The
set is closed: no log line takes a `reason` or a `cause` from a consumer.

## 8. The cost of observing

Observation is free when unused (rules 1.3) and is the consumer's cost when used
(`spec-scale.md` 6.1). This section says what the world records, what an event
costs to build, and what the volume of the library is.

### 8.1 What the world records at construction

8.1.1. **A flag, once**: whether any observer was supplied (`WithObserver` was
given a non-nil observer at least once). It is a boolean on the world, never
changed after construction, and every event-construction site is guarded by it
(rules 4.5). With no observer no event struct is built, no slice is allocated for
one and nothing is boxed.

8.1.2. **The observers, in order**, as a slice (rules 6.2): one delivery loop per
event, no allocation per delivery. A nil observer given to `WithObserver` is
ignored and never stored. The default observer is the **unexported**
`discardObserver` of `observer.go`, with the assertion
`var _ Observer = discardObserver{}` (architecture.md 4.5): a consumer has no
need of a name for "no observer", which is no option or a nil one, so the type is
not exported. No event is built for it, because the flag of 8.1.1 is false when
only the default exists.

8.1.3. **Tables derived once**: the names table (3.2.4), whose strings are the
configuration's own (no copy of the string data); and, per module that emits
aggregates, the arrays of counters that the aggregate events are built from, sized
by the configuration and reused every tick. They are scratch, saved nowhere.

8.1.4. **What it does not record**: the logger's level. `slog.Handler.Enabled` may
depend on the context or on a `LevelVar` that changes at run time, so the world
calls `Enabled` at each guarded site (rules 3.1.3). With the default
`DiscardHandler` the call returns false at once.

### 8.2 The cost of an event

8.2.1. **Without an observer: nothing beyond what the owners' own rules need.**
A guarded site is a test of one boolean. A counter that an owner's own rule needs
(the flow totals of the ledger, the runs of a `Warn`, the spawner's caps) is an
integer addition into a preallocated slot and costs the same whether or not anyone
listens (`spec-ledger.md` 12.3); a counter that exists **only** to build an event is
built only under the flag of 8.1.1, and an owner states which of its counters is
which (NFR-OBS-001).

8.2.2. **With an observer:** per event, one struct construction (the event is a
value of a few words to a few dozen), one boxing into the `Event` interface (an
allocation, since the value escapes into an interface that the observer may keep),
and, for an event with a slice (`RecipeExecuted` outputs, `Shipment*` lines, the
snapshot of `WorldRestored`), one allocation for the slice (3.3.2, the slice is
the event's). Then one call per observer. An aggregate event also pays the pass
over its counters once per tick.

8.2.3. **No hot-path allocation without an observer** is an enforced test
(NFR-OBS-001, NFR-OBS-002, 9.6): every emission and logging site of the tick loop allocates
nothing with the default logger and observer, and a full step of the reference
chain stays under a constant baseline of allocations (rules 8.2).

### 8.3 The volume of the first consumer's world

All numbers are placeholders from `spec-scale.md` 6.3 and N-030 (5 000 agents, a
tradable set of 8, 30 commodities, 5 regions, 20 roles); they are counted, not
measured (N-015).

| Class | Events per tick | How |
|-------|-----------------|-----|
| O | up to 40 000 | agents times tradable set: `5 000 * 8` placed or rejected orders (`spec-scale.md` 6.3) |
| R | up to about 40 000; a placeholder typical 20 000 | a pair trades `min` of the two remainders (`spec-clearing.md` 4.4), so a trade exhausts at least one order and trades per market can reach its orders less one |
| A | ceiling about 3 600 (the terms below sum to 3 585); a typical tick a few hundred | markets 150 (`MarketCleared`) and 150 (`BeliefsUpdated`); `BoundaryFlow` up to 5 x 14 x 30 = 2 100 rows, a few hundred that occur; `UpkeepShortfall`, `OwnUseChanged`, `PopulationChanged` up to 100 each; `WantShortfall` up to 5 x 20 x 6 = 600; `ProductionHalted` up to 100; `OutsideTraded` up to 2 gates x 30 x 2 = 120; the rest under 50 ; the later modules add aggregates bounded by their own counts of the configuration (`RoomLimited` and `SavingPlanned` by regions times roles, up to 100 each at these sizes; `TechnologyAdvanced` by regions times fields; the rest by regions times a small count of classes, bodies, flows or actors), a few hundred more at most, so the ceiling rises by that and stays far below O and R |
| E | tens to a few hundred | entries, exits, role changes, recipe completions (up to one per agent per tick for a one-tick recipe: up to 5 000), shipments, contracts, hires, claims, covers, titles and facilities |
| W | 2 | `TickStarted`, `TickEnded` |
| C | 0 | between ticks |

The orders and trades are **about 99% of the events** of a tick of the
reference world at the typical figures (60 000 of about 60 500) and **about 92%**
if every agent completes a one-tick recipe (`RecipeExecuted` is class E, one per
agent: 60 000 of 65 500). An observer that wants only the aggregates still
receives and ignores 60 000 events; each costs a boxing allocation. At a placeholder of 50 nanoseconds per
event that is about 3 ms a tick, about 6% of the 53 ms of `spec-scale.md` 6.3.

8.3.1. **What a consumer can do about it** with the options as they are: an
observer that returns at once on the order and trade types costs the boxing and
the call, which is the floor. The option that would remove the boxing for the
classes the consumer does not want, `WithObserver(o, classes...)`, is a change to
architecture.md 6.2 and to the options table of `spec-world.md` 4.1 and is **left
open** (section 18, question 1) until a measure shows the 6% to matter: it needs
no change to the catalogue, since a class is a property each row of 4.1 already
carries.

### 8.4 The cost of the snapshot and of `WorldRestored`

The snapshot is on request and costs what 5.5 says: about 1.1 ms of counted work
for the first consumer's world (about 1.5 ms with the saving fields of class Q, 5.5),
2% to 3% of a tick, and `O(configuration + in-flight)` in size (5.6). `WorldRestored` builds one, once, and only when an observer was
supplied. A world that is never snapshotted pays nothing for the snapshot: the snapshot
keeps no counter of its own (5.2), and the S, L, B and Q classes are scans at the call.

## 9. How observability is tested

`rules/observability-and-logging.md` 8 requires the tests below, and this section
makes each a requirement with its own ID (section 17) and says what the test
asserts and over what. They follow the pyramid of testing.md 2: the unit level
tests one event's construction and one log site; the integration level runs a
world with a recording observer and a capturing handler; the end-to-end level is
the golden rendering. No test uses a mock library, a sleep or a wall clock
(testing.md 3.6, 10.2). Tests are table-driven and name the seed of a failing
run (testing.md 6.2).

### 9.1 The recorder and the events

9.1.1. **The recorder.** A hand-written `Observer` in a `_test.go` file, which
appends every event it receives, by value, to a slice and does nothing else. It
needs no library support (architecture.md 10.3). Because the slices of an event
are the event's own (3.3.2), the recorder may keep them.

9.1.2. **What it asserts**, for a scripted scenario on a fixed seed:

| Test | Asserts | Requirement |
|------|---------|-------------|
| the catalogue is complete, **in the staged form** | every event type declared in the root package is a row of the tables of 4.1; every row of 4.1 is declared or named by the test's one explicit `notBuilt` list (event types, and `Event.Field` entries for the fields of a built event that its module has not appended yet); the test fails when a listed type or field is declared (a stale entry) and when a row is neither declared nor listed (a typo); every declared type is seen at least once over the scenarios of 9.7 (the **all-modules** scenario of 9.7 exists for this, and in a stage it holds the modules built); the classes and gates match (a plain text parse of 4.1, as testing.md 6.7 does for the parameter table). The list is empty when the last module is built | FR-OBS-003 |
| event shape | every event has `Tick` first, no pointer, no map, no float (by reflection over the type: a test helper, not production code) | FR-OBS-001 |
| slices are the event's | mutating a recorded slice changes no later event and no world state | FR-OBS-002 |
| order | for a scripted step, the sequence of event types equals the order of `spec-world.md` 6.11 as amended by 4.4.8 and 4.5, and each event comes after the state change it describes (a snapshot taken after the step agrees with the last event of each kind) | FR-OBS-022 |
| fields | for each declared event, its fields are the fields of its row in 4.1 less those of `notBuilt`, in the order of the row, and in the scenario they equal the values the scenario's arithmetic gives (a worked example per event, with the arithmetic in a comment: testing.md 5.5) | FR-OBS-004 |
| pseudo-agents | no `AgentSpawned`, `AgentBankrupted`, `WeightChanged` or `RoleChanged` has an estate or a counterparty as subject, and no count includes one | FR-OBS-007 |
| region on agent events | `RecipeExecuted`, `RecipeAbandoned` and the `Traineeship*` events carry the agent's region, equal to the region of its account at the event | FR-OBS-018 |
| weight change money | the money and units of a `WeightChanged` equal the change of the agent's and the estate's balances | FR-OBS-019 |
| entry step order | the traineeship events come in the order of 4.4.8 | FR-OBS-021 |
| silent aggregates | an aggregate with "never when all are zero" is absent in a tick in which all its counts are zero | FR-OBS-009 |
| module off | with a module off, none of its events is emitted, and the stream of the other events equals that of a world in which the module was never configured | FR-OBS-008 |
| appended field | with a module off, the rendering of every event and row is byte-equal to that of the tree before the module was added (the events file of `testdata/golden/off/` does not move in a module's change), and a reflection test asserts that no field a module appended is of a typed ID type | FR-OBS-091 |
| slot events | each slot call emits one event per changed cell and none for an unchanged one, in key order, with the next step's `Tick`, before its `TickStarted` | FR-OBS-017 |
| removal order | one removal emits its events in the order of 4.4.8 | FR-OBS-020 |
| bucket | `BoundaryFlow` with `Tick` -1 appears once, before the first `TickStarted`, with the totals of `Snapshot.Construction` taken before the step | FR-OBS-015 |
| `BoundaryFlow` scope | no internal reason appears; the order is that of 4.4.1.2 | FR-OBS-014 |
| `ProductionHalted` | emitted only with `Halted` above 0, with a `Skipped` row for each recipe skipped | FR-OBS-016 |

### 9.2 The capturing handler and the logs

9.2.1. **The handler.** A hand-written `slog.Handler` in a `_test.go` file that
records each record's level, message and attributes and reports `Enabled` for
every level (so the guarded sites run).

9.2.2. **What it asserts.**

| Test | Asserts | Requirement |
|------|---------|-------------|
| each `Warn` path | for each row of 7.3, a scenario that makes its condition hold produces a record with the level, the constant message and the keys of the row, at the tick its pattern says (the 1st and the 10th for A; the 10th for B), and again at the next multiple of 10 | FR-OBS-071 |
| bounded | in each uninterrupted run of N ticks no cell (row 19 excepted: one per recipe per tick) emits more than `ceil(N / warnRun) + 1` lines of one message | FR-OBS-072 |
| counterpart | for each row of 7.3, the event or snapshot field of its column "Counterpart" exists in sections 4 and 5 and, in the scenario, carries the number the line reports (a plain text parse of 7.3) | FR-OBS-051 |
| no `Info` in the tick | between `TickStarted` and `TickEnded` of any step of any scenario, no record at `Info` or above other than a row of 7.3 | FR-OBS-070 |
| no `Error` | no record at `Error` in any scenario of 9.7 | FR-OBS-074 |
| keys | every attribute key of every record is a key of the rules' table 3.3.2 (a plain text parse) | FR-OBS-073 |
| constant messages | for each call site, all records have the same message | FR-OBS-075 |
| construction | "world created" has the keys of 7.2 with the values of the scenario's configuration | FR-OBS-076 |

### 9.3 Defaults and non-interference

| Test | Asserts | Requirement |
|------|---------|-------------|
| defaults | a world with no logger and no observer builds, steps and runs, writes nothing to stdout or stderr, and does not panic; a nil logger and a nil observer behave the same | FR-OBS-090 |
| non-interference | the same seed and configuration, with and without a logger, an observer and `WithAudit`, give identical snapshots at every tick, identical `ConfigHash`, identical random state, identical event streams between two observed runs, and identical draw counts (`spec-world.md` FR-WLD-064) | FR-OBS-023 |
| a panic | an observer that panics marks the world failed and propagates the panic (architecture.md 12.5); a world that was observed by a recorder is unaffected | FR-OBS-024 |

### 9.4 Cardinality

9.4.1. **In the core module**, a test over the reference chain asserts that
every event field that becomes a metric attribute takes only values present in
the configuration: for each typed ID field, `0 <= id < len(Names.X)` (and
`Region` of an outside node is a node); for each `Reason`, `Cause` and `Side`
field, the value is a constant of its closed set. The list of such fields is the
attribute column of 6.2.1: commodity, role, recipe, side, want, region, node,
mode, resource, reason, cause, kind, outcome (the counts of `RumoursServed`),
limit and decision (the counts of `MigrationReviewed` and `MerchantsPlanned`),
entry limit (the counts of `EntryLimited`), and the attributes of the later modules
of 6.5.1: store class, facility class, body, right, control, field and actor
(a typed ID in the range of its names slice), and the closed sets of the room limit,
the tier, the credit and cover limits, the risk kind, the clamp, the activity, the
project, the bar, the entry outcome, the policy cut and hold, the cover kind and the
cover loss. Requirement FR-OBS-005 and FR-OBS-006.

9.4.2. **In `otelobserver`**, a test asserts the attribute sets recorded on each
instrument, using a configuration built in Go inside its own tests (it cannot read
the core module's `testdata/`, testing.md 10.2), against the series bound of
6.5.2 for that configuration (FR-OBS-052).

### 9.5 The golden rendering of events

9.5.1. **What it is for.** The golden file of testing.md 7.1 compares "a canonical
rendering of the result (prices, volumes, inventories, event counts)". This
section defines the rendering of **events**, so that every consumer of the golden
means the same thing, and so that a change of the library's behaviour or of the
order of events is a visible diff.

9.5.2. **Format.** One line per event, in emission order:

```
t=<Tick> <Type> <Field>=<value> <Field>=<value> ...
```

- `Tick` is rendered as `t=` first; every other field follows in **declaration
  order**; a field is `Name=value` with no spaces inside the value.
- **A field at its zero value is not rendered** (decision D5 of the plan of
  Stage 0). A field whose value is the zero value of its type (`0`, `false`, the
  empty string, an empty or nil slice) is omitted from the line, whichever event
  it belongs to and whether or not a module appended it, so the rule is one test
  and not a list. Never omitted: `Tick` and the type name, and every field of a
  typed ID type (3.2.1) or of `Side`, because zero names a region, a commodity or
  an agent (`Region=alpha` is `RegionID` 0 and must show). The effect is the one D5
  asks: a field that a module appends to an event is a count, an amount, a bool, a
  constant or a slice (3.3.4), so with the module off it is zero and the line is
  byte-equal to what it was before the module existed, and
  `testdata/golden/off/` does not move when a module is added (testing.md
  7.1.1). A reader reads an absent field as zero.
- An integer, a `Money`, a `Price` and a `Quantity` are decimal; a `bool` is `true`
  or `false`; a `[32]byte` hash is the first 8 bytes in lower-case hex.
- A typed ID is the **string ID** from the names table (`Region=north`,
  `Commodity=grain`); an ID outside the table is `#<n>`; `NoCommodity` is `-`. An
  `AgentID` is its integer. A reason, a cause and a side are their constants.
- A slice is `[` rows `]` separated by spaces, a row is `{Name=value ...}`; an
  empty slice is omitted (above), and the fields of a row follow the same
  zero rule (`{Commodity=grain Quantity=3}` has no `Disposed`).
- `Names` and `Snapshot` (in `WorldCreated` and `WorldRestored`) are **omitted**
  from the line; `WorldRestored` renders `SnapshotRows=<n>` instead, the number of
  rows of the snapshot.
- Nothing else: no timestamp, no address, no map order.

**Worked rendering of an appended field (an example).** `AgentSpawned` is declared
by the slice with `Agent`, `Region`, `Role`, `Weight`, `StartMoney` and
`External`. The storage module's change appends `StoresLocked` at the end. For
the household `AgentID` 1 of 16.1, role `baker`, weight 1, start money 100, the
slice renders and the tree with the storage module added but off renders the same
line, byte for byte:

```
t=0 AgentSpawned Agent=1 Region=alpha Role=baker Weight=1 StartMoney=100
```

(`External=false` and `StoresLocked=0` are zero and absent). With the module on
and two stores of the role row withheld by the class of the region, the line is

```
t=0 AgentSpawned Agent=1 Region=alpha Role=baker Weight=1 StartMoney=100 StoresLocked=2
```

and only the `testdata/golden/storage/` and `all/` files carry it.

9.5.3. **The file** `testdata/golden/events-<pack>.txt` holds, for the reference
chain on a fixed seed and a fixed number of ticks (a placeholder, REF 8.2):

1. the **full rendering** of the first 5 ticks of the `one` pack (small enough to
   read), after the events of construction;
2. one line per tick of the whole run: `t=<n> events=<count> digest=<16 hex>`,
   the first 8 bytes of the SHA-256 of that tick's rendered lines (a test may use
   a hash: it is not randomness);
3. the **count of each event type** over the run, in the order of 4.1; **a type
   with a count of 0 is not listed**, so an event of a module that is off adds no
   line (D5);
4. the final snapshot rendered in the same style, with the same zero rule: one
   line per row with its type name and fields, **a section with no rows is not
   rendered**, and a snapshot field a module appends is omitted at zero as an
   event field is.

The renderer is test code (a `_test.go` helper using reflection over the
exported fields), not an exported function: a library function that renders
would be a back door that exists for tests (architecture.md 10.2). A consumer
that wants the same rendering writes its own to this specification; EXP may ship
one in its harness.

9.5.4. **Update.** The file is updated only with `-update`, and a PR that changes
it says which behaviour changed and which requirement or fix it serves
(testing.md 7.2). A change of the order of events or of a field's meaning is a
visible diff and needs that explanation. Requirement FR-OBS-080 and FR-OBS-081.

### 9.6 The allocation tests

9.6.1. With the default logger and observer, a test using `testing.AllocsPerRun`
asserts, as rules 8.2 requires, that (a) each event-emission and logging call
site that the tick loop reaches allocates nothing, and (b) a full `Step` of the
reference chain allocates no more than a baseline constant held in the test.
Raising the constant is a reviewed change with a reason (NFR-OBS-001,
NFR-OBS-002).

9.6.2. **Every site is reached.** The test of (a) runs the **all-modules**
scenario of 9.7, the same that FR-OBS-003 requires to emit every declared event type with
an observer, so that the sites exercised by the allocation test and the
types emitted by the recorder are the same set: a site that no scenario reaches
would pass the allocation test and be untested.

9.6.3. **A benchmark** with `b.ReportAllocs()` and `for b.Loop()` reports the
trend of a step with no observer, with a recorder and with the adapter's
accumulators (NFR-OBS-005 for the last), beside the test (testing.md 8).

### 9.7 Run lifecycle, composition, restore

| Test | Asserts | Requirement |
|------|---------|-------------|
| run lifecycle | `Run` emits `RunStarted` and `RunEnded` when it completes, when its context is cancelled (`Completed` the ticks that finished, `Cancelled` true) and when a step fails (`Failed` true, `Completed` the ticks before it); a negative `ticks` and a failed or in-step world emit neither | FR-OBS-012 |
| tick bracketing | every step that emitted `TickStarted` emits `TickEnded`; a step that fails at P4.1 or at the audit has `Failed` true; a guard that fails at P0.1 emits neither; a step that ends in a panic emits no `TickEnded` and the adapter's span stays open until the world is restored (WLD 3.5.4) | FR-OBS-013 |
| composition | with several `WithObserver` options every event reaches each observer in the order given, and an observer that type-switches on a subset sees exactly that subset of the full recorder's stream | FR-OBS-024 |
| `WorldCreated` | carries the seed, the whole hash, the warm-start flag and the names, before the first `AgentSpawned`, with `Tick` 0, and each initial agent has one `AgentSpawned` | FR-OBS-010 |
| restore in construction | a world restored from a checkpoint taken before its first step emits the bucket's `BoundaryFlow` at its first step, as the original would | FR-OBS-015 |
| `WorldRestored` | a restore emits it first, with the restored tick, the hash, the names and a snapshot equal to the original world's at that tick; and no `AgentSpawned` | FR-OBS-011 |

The **scenarios** that the tests share, kept in `testdata/` (the pack and its
overlays, `spec-reference-chain.md` 4):

1. `one` with a seed range, for orders, trades, beliefs, upkeep;
2. the **all-modules** pack: the reference chain with every module on and
   configured, run for a fixed number of ticks with a consumer's script that makes
   one call of every slot and adds and removes an external agent, so that every
   event type occurs (the script is a list of calls between steps, in the test);
3. `famine` for `UpkeepShortfall`, the failure ladder and the `Warn`s of the
   unfed and the extinct;
4. a faulty `Clearing` (a hand-written fake at an extension point,
   architecture.md 10.3) for `TickEnded.Failed` and `RunEnded.Failed`.

### 9.8 The snapshot tests

| Test | Asserts | Requirement |
|------|---------|-------------|
| deep copy | mutating a snapshot changes nothing in the world or in the next snapshot; a snapshot taken, then the world stepped, is unchanged; concurrent marshalling under `-race` has no race | FR-OBS-030 |
| refusals | `Snapshot` returns `ErrInStep` from inside a step (an observer that calls it) and `ErrWorldFailed` on a failed world | FR-OBS-031 |
| agents and households | `Agents` and `Households` are different; pseudo-agents are in neither; a cohort of weight 40 counts 1 and 40; the sum over `Roles` equals the header | FR-OBS-032 |
| bounded | the number of rows of a snapshot of one configuration is equal at scale 1 and at scale 40, with the in-flight sets empty | FR-OBS-033 |
| rebuild | the agents and households by role and region rebuilt from the events equal the snapshot at every boundary of the all-modules run, over a range of seeds | FR-OBS-040 |
| money | the money of the households, of the pool and of the treasury sector of each region rebuilt from the events (4.3.3) and `WorldCreated.Jurisdictions`, plus `Construction` before the first step and `Pending`, equals `MoneyHouseholds`, `MoneyPool` and `MoneyTreasury` at every boundary, also after the scenario's calls between steps, with a governed jurisdiction whose `StartMoney` is above 0 and a standing order and a tax in different regions | FR-OBS-041 |
| identity | the boundary balance of each reason equals `Construction` plus the sum of `BoundaryFlow` of every tick plus `Pending`, as a signed total, at every boundary, also after the scenario's calls between steps | FR-OBS-042 |
| restore equality | the snapshot after a restore at tick `n` equals the snapshot of the uninterrupted world at `n`, field for field, for a range of seeds and each module on | FR-OBS-043 |
| region state | `Region(r)`'s `RegionState` equals the rows of the full snapshot whose key has `r` | FR-OBS-044 |
| consistency | the sum of `TradeExecuted.Quantity` of a market in a step equals `MarketCleared.Volume`; the `MarketState` of the query equals the snapshot's | FR-OBS-045 |

### 9.9 The adapter's tests

In `otelobserver`, against the OpenTelemetry SDK's in-memory exporters (testing.md
6.6, 9.1), each with its own configuration built in Go:

| Test | Asserts | Requirement |
|------|---------|-------------|
| catalogue | every instrument of 6.2 has the name, unit, description and kind there, in the `economysim.` namespace, with no `_total` and no unit in the name | FR-OBS-050 |
| attributes | the attribute sets are within the bounds of 6.5 (9.4.2) | FR-OBS-052 |
| unfed | the sequence of 16.4 gives the level of 6.3.1 at every tick | FR-OBS-053 |
| levels after a restore | an adapter attached at a restore and one attached at construction have equal level instruments after the same further steps | FR-OBS-054 |
| removals | `economysim.agent.removals` by cause sums to the `AgentBankrupted` count | FR-OBS-057 |
| step failed | over a scenario with one step failing at P4.1 and one failing its audit at P8.4, `economysim.world.step.failed` is 2 and, with the recordings of the duration, equals the `TickEnded` count | FR-OBS-058 |
| exits | the cohort of 40 with 7 unfed households of `spec-spawner.md` 5.3 raises the instrument by 4 a tick while `economysim.agent.removals` stays at 0, then by the weight left at its removal; a population death is in `economysim.household.deaths` and not here | FR-OBS-059 |
| price age, beliefs at bound | a market with no trade for `k` ticks reads an age of `k`; beliefs all on the ceiling read 10 000 for `ceiling` and 0 for `floor`; a tick with no `BeliefsUpdated` holds the last value | FR-OBS-060 |
| volume | the SDK is called at most once per series per tick (6.6.2), with a counting meter | NFR-OBS-005 |
| event-only counters | with no observer the counters that exist only for an event are not built (a scenario with a test hook on the flag in an internal test) | NFR-OBS-003 |
| snapshot allocation | the allocations of `Snapshot` at scale 1 and at scale 40 of one configuration are equal | NFR-OBS-004 |
| no callback | the adapter holds no `World`, and a run with the adapter and one with a recorder give the same world state | FR-OBS-056 |
| failed step | a step that fails at P4.1 and one that fails its audit at P8.4 record no duration, end their span with an error status, run no reset and leave the levels as the events seen put them | FR-OBS-055 |
| the snapshot is validated | `FuzzRecordSnapshot`: for any `Snapshot` value (IDs out of range, negative counts, nil slices) `RecordSnapshot` neither panics nor records a series outside the bounds of 6.5 (testing.md 7.4) | FR-OBS-052 |

## 10. What OBS hands to IND, STY, EXP, CHK and INT

Stated as requirements on this spec's output, not as their content.

### 10.1 To IND (`spec-indicators.md`)

| IND may rely on | Where |
|-----------------|-------|
| a measure is a function of the events of the steps it covers and of a snapshot at a boundary; no value exists between `TickStarted` and `TickEnded` | 3.1, 5.1 |
| agents and households are separate; population indicators count households, never agents; pseudo-agents are in neither | 3.3.6, 5.1.2 |
| the money of the households, the pool and contracts by region, and the boundary balance by reason, are in every snapshot; the flows of a tick reach IND through `BoundaryFlow` only, and the bucket at `Tick` -1 | 5.3.3, 4.4.1, 4.4.2 |
| IND's reads of the snapshot and the events are `spec-indicators.md` 3.3 and 9; the household rows of a measure are INT's `Agents` and `Agent` (5.6) | 5.6, 10.5 |
| a price index of a commodity class needs the commodity class tag (`spec-goods.md` 6.1) from the configuration and the prices of `MarketCleared` or `MarketRow.State.Price`; whether the index is the local price or the delivered price (TRN-A4) is the consumer's choice, since the freight is in the shipment events | 4.1.2, 5.3.5 |
| value added and production and destruction value need `BoundaryFlow` under `recipe_output`, `recipe_input`, `consumption` and `spoilage` by commodity, and a price the consumer supplies (`spec-ledger.md` 9.2: a balance sheet is measurement only) | 4.4.1 |
| the typed IDs are positions in the configuration; a measure that must survive a change of configuration keys by the string ID from the names table | 3.2 |

### 10.2 To STY (`spec-stylised-facts.md`)

The failure regimes of `spec-stability.md` 7 are computable from the events and
the snapshot alone (FR-STB-012), and this is the map STY and IND use:

| Regime | Computed from |
|--------|---------------|
| F1 runaway inflation | the price per commodity (`MarketCleared.Price`, `MarketRow.State.Price`) by class; the money supply (`Regions[].MoneyHouseholds`, `MoneyPool`, `MoneyContract`) and `BoundaryFlow` money |
| F2 hoarding | the shares of money by sector and region (`MoneyPool` and `MoneyContract` by region, against `M`); `HeldByAgents` and `HeldByEstate` by market; for `Inequality.Top`, the household money rows of INT's `Agents` and `Agent` (not a snapshot field, 5.6); `MarketCleared.Volume` falling |
| F3 collapse | `Households` by region and role (the events of 4.3.2, or the snapshot); a role with `Agents` 0 (`RoleRow`) |
| F4 stagnation | `MarketCleared.Volume` and the count of `TradeExecuted` against their start |
| F5 pinned market | `BeliefsUpdated.AtFloor`, `AtCeiling`, `Outcomes`; `MarketCleared` with unmatched units on both sides and no volume; the `Warn` "price beliefs at bound" |
| F6 cobweb blow-up | the range of `MarketCleared.Price` over a window; role shares from `RoleRow.Households` |
| F7 synchronisation | `BeliefsUpdated.LowCentre`, `HighCentre`, `MeanCentre`; the distribution of `RecipeExecuted.StartTick` |
| F8 offset | `MarketCleared.Price` against the solver's price (`spec-viability.md`), a consumer's computation |
| R9 deflation with distress (STY 6.2, proposed there as F9 of the regimes of STB 7) | the `PriceIndex` chain over `MarketCleared.Price` by class (IND `Fisher`) and the volume of `MarketCleared`; no new event and no new field |

A scenario's `Run(ctx, n)` leaves a world that can be snapshotted (`spec-world.md`
13); the golden rendering of 9.5 is the regression record of a run.

### 10.3 To EXP (`spec-experiments.md`)

| EXP may rely on | Where |
|-----------------|-------|
| worlds share nothing, so a sweep builds many and snapshots each between ticks; the snapshot's size and cost do not depend on the number of agents | 5.5, 5.6 |
| the recorder of 9.1 is the pattern for a harness's collector; the format of a diff between two runs of a sweep is `spec-experiments.md` 8.2.2 (`Render`), not 9.5.2 | 9.1, 9.5 |
| a sweep with no measure that has an `Observer` registers none (`spec-experiments.md` 3.4.4); the world then builds no event (observability-and-logging.md 4.5) | 4.5 |
| `ConfigHash` and the names table key a result; an index is not stable across configurations | 3.2 |
| a failed world is detected by `TickEnded.Failed` and `RunEnded.Failed`, and by `Err()` | 4.4.3 |
| the cost of observing is counted per class (8.3); a sweep that wants no per-order events passes an observer that ignores them, or waits for the open option (18, question 1) | 8 |

### 10.4 To CHK (`spec-checkpoint.md`)

| CHK is asked | Why |
|--------------|-----|
| to save the **last market report** of each market (the `Volume`, `UnmatchedBid` and `UnmatchedAsk` units and the best quotes with their flags) as one unit, since `MarketState` and the snapshot's rows read it and FR-OBS-043 needs a restored snapshot to equal the original | 5.3.5 |
| to emit `WorldRestored` as the first event of a restore to the observers given to the restore, with `Reseeded` when `WithReseed` was used, and to build the snapshot for it only when an observer was supplied | 4.4.5 |
| to keep the construction bucket while the world is in its construction phase and to emit its `BoundaryFlow` at the first step, so that a world restored before its first step behaves as the original | 4.4.2, 5.2.4 |
| to save the pending flow totals that calls between ticks added (`spec-world.md` 12.1), since `Snapshot.Pending` is a field | 5.3.3 |
| the log message "world restored" and the keys `format` and `bytes` | 7.2, 7.6 |
| that the snapshot is not the checkpoint and has no version | 5.7 |

### 10.5 To INT (`spec-integration.md`)

| INT is asked | Why |
|--------------|-----|
| each slot call emits one event per changed cell, in key order, at the call, with the next step's `Tick`; the names are `YieldShocked`, `TransitFactorChanged`, `PopulationScaled` (INT 5.1 already uses them) | 4.4.4 |
| `Market(r, c)` returns the `MarketState` that `Snapshot.Markets[].State` holds, and `Region(r)` returns the `RegionState` of 5.3.5a (adopted by INT 6.3, 6.5; check only) | 5.3.5, 5.3.5a |
| every `cause` a call gives an event is a constant of a closed set (`consumer`), never consumer text, so that it can be a metric attribute | 4.2.3, 6.5.1 |
| `AgentSpawned` for an external agent carries `External` true and the starting money by `initial` is counted in the construction bucket or the next tick's `BoundaryFlow` | 4.4.2 |

## 11. Parameters

This spec adds **no `Config` field, no default and no row** to the
parameter-owner table. Observation changes no result (rules 1.4), so nothing in
it is a tunable (architecture.md 9.5). The registry test (testing.md 6.7) needs
nothing from it.

**Rule constants**, beside their code (architecture.md 9.6): the catalogue of
events and their field order; the closed sets of `Reason`, `Cause`, `Kind` and
`ScaleKind` constants (and `Side`, declared in `event.go`); `NoCommodity` (-1); `warnRun` (10), the run length of a bounded
`Warn` (7.3); the golden rendering format (9.5.2); the order of the names table
(canonical order, 3.2.4); the metric names, units and attributes (6.2). The
number 10 was a literal in several owners' text; it is one constant here, shared.

**Not parameters:** the flag "an observer was supplied" (8.1.1), the names table
(derived), the series bound (6.5.2).

## 12. Validation

No rule is added to `spec-config.md`. An option's argument is validated by its
option (`spec-world.md` 4.1: a nil observer and a nil logger are the defaults, not
errors). The one construction-time check that touches observation is a test, not a
validator: the all-modules scenario of 9.7 must emit every declared event type
(FR-OBS-003).

## 13. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. This spec
adds **no feedback loop**: observation never changes a result (rules 1.4;
FR-OBS-023), and the checklist of `spec-stability.md` 9 was followed (9.1: no rule
of this spec reads a measure and changes a rate, since the library has none that
reads an event or a snapshot).

- **OBS-A1. A consumer closes a loop through the observer.** *Kind:* hazard.
  *Loop:* events of step `t` (+) a consumer's rule (a game's reaction to a price,
  a script that sets a shock) (+) a slot call (+) the state at `t + 1`. The loop
  exists in the consumer, not in the library, and has a delay of one tick by
  construction (the events of a step reach the consumer after the step; its call
  acts at the next step, `spec-world.md` 8.1). *Owner:* INT 3.5; the consumer.
  *Damper:* none in the library, by design ("macro is measured, never set",
  ADR 0001 decision 7); `spec-policy.md` (PLC) is where a rule that reads
  measures and sets a slot is specified, with its own entries. *Undamped:* all of
  it. *Condition:* a consumer's controller with a gain that is small against the
  delay, as for any loop with a delay (`spec-stability.md` 6.3). *Test:* none in
  the library; STY's shock scenarios exercise the slots with a fixed script, not a
  controller. *Status:* hand.
- **OBS-A2. An observer that is slow stretches the step but changes no result.**
  *Kind:* hazard of cost, not of dynamics. A slow observer lengthens the wall
  time of a step (rules 4.4) and nothing else: the world is single-threaded and
  has no timeouts, no deadline and no clock, so a slow observer cannot make a
  result differ. *Owner:* WLD 3.1.3. *Damper:* none needed. *Test:* FR-OBS-023
  (the stream and the state are the same with a slow and a fast observer).
  *Status:* hand.

No new damper (the checklist's item for dampers: none added) and no new scenario.

## 14. State contributed to a checkpoint

None. The snapshot is derived (5.2), the names table is derived from the
configuration (3.2.4), the flag of 8.1.1 is rebuilt from the options of the
restore, and the observers, the logger and the adapter's state are not world
state (`spec-world.md` 12.1). The runs that the `Warn`s of 7.3 use are the owners'
diagnostic state and are listed in their checkpoint sections (AGT 15, BEL 13, WEL
13, SKL 9, SPN 14, POP 11, RES 15, SUB 15, MIG 14, DEM 12, EMP 16, LND 14, CRD 17, ORG 12); they are saved so that
a resumed run logs as the original did, and nothing here adds to them.

What a restore needs from this spec is two things: the names table to put in
`WorldRestored`, which it builds from the configuration, and the snapshot of the
restored boundary (4.4.5), which it builds by the same function as `Snapshot()`
(FR-OBS-043: a restored snapshot equals the original).

## 15. What was run

Nothing. The numbers of this spec are arithmetic on the placeholder sizes of
`spec-scale.md` 6.3 and the reference chain (5.5.2, 6.5.2, 7.3.1, 8.3), shown in
full in the sections that use them; no model was needed and none was written
(`doc/specs/models/obs/` does not exist). No number here is a default, and none is
claimed to be stable (N-027). The measures that would turn the counted costs into
real ones are the slice's (N-015): the allocation per event, the cost of a
snapshot at the first consumer's size, and the real volume of orders and trades.

## 16. Worked examples

Invented settings, flagged as examples (N-017): one region `alpha` and two
commodities `grain` and `loaf`, in the canonical order of their IDs; the numbers
are placeholders chosen to be easy to follow, and the arithmetic is by hand (15).

### 16.1 One tick in the golden rendering (an example)

The world has one region `alpha` (`RegionID` 0, whose estate is `AgentID` 0), two
roles `baker` (0) and `grower` (1), two households `AgentID` 1 (a baker) and 2 (a
grower), each of weight 1, and the commodities `grain` (0) and `loaf` (1). Tick 7,
in the order of `spec-world.md` 6.11, in the format of 9.5.2:

```
t=7 TickStarted
t=7 RecipeExecuted Agent=2 Region=alpha Recipe=grow StartTick=7 Executions=1 Outputs=[{Commodity=grain Quantity=3}] EffectiveYield=10000
t=7 OrderPlaced Region=alpha Commodity=grain Agent=1 Side=bid Limit=45 Quantity=2
t=7 OrderPlaced Region=alpha Commodity=grain Agent=2 Side=ask Limit=40 Quantity=3
t=7 TradeExecuted Region=alpha Commodity=grain Buyer=1 Seller=2 Price=42 Quantity=2
t=7 MarketCleared Region=alpha Commodity=grain Price=42 Volume=2 UnmatchedAsk=1 BestAsk=40 HasBestAsk=true
t=7 MarketCleared Region=alpha Commodity=loaf Price=90 PriceAge=4
t=7 BeliefsUpdated Region=alpha Commodity=grain Outcomes=2 Adjusted=1 Raised=1 MeanCentre=44 LowCentre=40 HighCentre=47
t=7 BoundaryFlow Region=alpha Reason=recipe_output Kind=faucet Commodity=grain Total=3
t=7 BoundaryFlow Region=alpha Reason=consumption Kind=sink Commodity=grain Total=2
t=7 TickEnded
```

Reading it. The grower completes a one-tick recipe in its start tick, so
`RecipeExecuted` comes in phase 2 (P2.7). Its bid and ask are placed in phase 3 in
agent order (1 before 2: the bid's agent has the lower ID). They cross, 45 against
40, and trade at the midpoint `(40 + 45) / 2 = 42.5`, rounded half to even to 42
(`spec-clearing.md` 5.2), for the smaller quantity 2; one unit of the ask stays
unmatched, so `UnmatchedAsk=1` and the best unmatched ask is 40. The `loaf` market
cleared with no order: its price is the one carried forward, 90, four ticks old
(`spec-clearing.md` 5.4), and `MarketCleared` is emitted anyway; every count and
flag that is zero or false is absent from the line (9.5.2), so that line is short
and `TickEnded` of a step that did not fail has no field. No
`UpkeepShortfall`: both households were fed, so none is emitted (3.3.3). The
`BoundaryFlow` rows are the two boundary reasons with a non-zero total in the
registry order of 4.4.1.2: 3 units of `grain` created by `recipe_output`, 2 eaten
(`consumption`, one unit for each household's upkeep); `trade` is not there, being
internal (4.4.1). No event for the modules that are off.

### 16.2 Households rebuilt from the events (an example)

Construction: `AgentSpawned Agent=1 Role=baker Weight=1` and `AgentSpawned Agent=2
Role=grower Weight=1`. Then, in later ticks:

| Event | Agents baker / grower | Households baker / grower |
|-------|-----------------------|---------------------------|
| after construction | 1 / 1 | 1 / 1 |
| `WeightChanged Agent=2 Role=grower From=1 To=2 Cause=births Money=4000` | 1 / 1 | 1 / 2 |
| `RoleChanged Agent=1 From=baker To=grower Weight=1` | 0 / 2 | 0 / 3 |
| `AgentBankrupted Agent=2 Role=grower Weight=2 Cause=starved` | 0 / 1 | 0 / 1 |

The household count is `To - From` on a weight change, `-Weight` and `+Weight`
on a role change (the agent and its weight leave `From` and join `To`), and
`-Weight` on a removal; the agent count changes only on a spawn, a removal and a
role change. The 4000 of the second row is the money the pool lent the born
household (`WeightChanged.Money`, positive to the agent): the pool of `alpha`
falls by 4000 and the household sector rises by 4000, which is how FR-OBS-041
rebuilds `MoneyHouseholds` and `MoneyPool`.

### 16.3 A restore (an example)

A world stopped at tick 50 has, in its snapshot, `Roles` rows
`(alpha, baker)`: `Agents=3 Households=3 Unfed=0` and `(alpha, grower)`: `Agents=5
Households=7 Unfed=2`, one shipment in flight with a line of 4 `grain`, and no
journey. An adapter attached to the restore receives `WorldRestored` with that
snapshot and seeds `economysim.agent.count` at 3 for `baker` and 5 for `grower`,
`economysim.household.count` at 3 and 7, `economysim.household.unfed` at 2 for
`grower` (and `lastUnfed[alpha][grower]` at 2), and `economysim.transport.in_transit`
at 4 for `grain`. Its counters (`economysim.market.trades`, ...) start at 0. At tick
50 an `UpkeepShortfall Unfed=3` for (alpha, grower) adds `3 - 2 = +1`, as it would
have for an adapter that had been attached from the start: the two adapters
agree on the UpDownCounters from the end of the first complete step after the
restore (FR-OBS-054).

### 16.4 The unfed level (an example)

The rule of 6.3.1 for (alpha, grower), with `lastUnfed` starting at 0:

| Tick | Event | Adapter adds | Level of `economysim.household.unfed` for `grower` |
|------|-------|--------------|-----------------------------------------------------|
| 5 | `UpkeepShortfall Unfed=4` | +4 | 4 |
| 6 | `UpkeepShortfall Unfed=6` | +2 (`6 - 4`) | 6 |
| 7 | none (everyone fed) | -6 at `TickEnded` (the cell was not seen) | 0 |
| 8 | `UpkeepShortfall Unfed=1` | +1 | 1 |
| 9 | (alpha, grower) `Unfed=3` and (beta, grower) `Unfed=2` | +2 and +2 | 5 |

The level is the sum over regions of the unfed households of the last step,
which is what the instrument means. The same numbers are `RoleRow.Unfed` of the
snapshots of those ticks unless a removal, a role change or an entry at P7.1
intervened in the tick (5.2.2).

### 16.5 A snapshot excerpt (an example)

The world of 16.1 after tick 7 (a placeholder state, `grain` held 4 by agents and
none by the estate, a grain job pending none):

```
Header Tick=8 Agents=2 Households=2 Money=9000 Modules=[]
Regions {Region=alpha Outside=false Agents=2 Households=2 MoneyHouseholds=9000 MoneyPool=0 MoneyContract=0 EntryBudget=0 HouseholdCap=0 OnJourneyFrom=0 OnJourneyTo=0}
Roles {Region=alpha Role=baker Agents=1 Households=1 Unfed=0 MaxHunger=0 MaxPrivation=0 HeldByFloor=0 HasMeanRatio=true MeanRatioBP=10400 ...}
Markets {State={Region=alpha Commodity=grain Price=42 Age=0 Volume=2 ...} HeldByAgents=4 HeldByEstate=0 Pending=0 Beliefs=2 MeanCentre=44 LowCentre=40 HighCentre=47 AtFloor=0 AtCeiling=0}
```

`Tick=8` because the snapshot is taken between ticks and `Tick` is the tick the
next step will run. `MoneyHouseholds` is the sum of the two agents' money, and the
sum of `HeldByAgents` and `HeldByEstate` over the two commodities is the goods the
ledger holds in non-boundary accounts, which the audit can check against the
boundary balances.

## 17. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. The
requirements of `spec-world.md` that bear on observation (FR-WLD-053, 054, 063,
064, NFR-WLD-001) are not repeated; these are the ones this spec adds.

### Functional

| ID | Requirement |
|----|-------------|
| FR-OBS-001 | Every event MUST be a plain struct of the root package that implements the sealed `Event` interface, MUST have `Tick int64` as its first field, and MUST contain no pointer, function, map, float or wall-clock time (3.3). |
| FR-OBS-002 | A slice in an event MUST be a copy owned by the event, in a deterministic order, and the library MUST NOT reuse its backing array after the event is emitted (3.3.2). |
| FR-OBS-003 | The event types declared in the root package MUST be rows of the tables of 4.1, each with an owner, a position, a volume class and a gate; every row that is not declared MUST be named by one explicit `notBuilt` list of the catalogue test, which MUST fail when a listed type is declared and when a row is neither declared nor listed (the staged form of architecture.md 9.7, 3.3.4); and the all-modules scenario MUST emit every declared type at least once. The list is empty when the last module is built. |
| FR-OBS-004 | Each declared event MUST carry exactly the fields that 4.1 lists for it, less those of the same `notBuilt` list (the fields of a module that is not built), in the order of its row, and each value MUST be the one the owning spec's rule gives. A module's change appends its fields at the end of the event and moves them to the end of the row. |
| FR-OBS-005 | Every field of an event or snapshot row that names a configured entity MUST carry the typed integer ID of its kind, in the range of the corresponding slice of the names table (3.2.1). |
| FR-OBS-006 | A reason, a cause, a side, a kind and every other member of a closed set in an event MUST be a constant of that set, and no call of the library MUST accept consumer text for one (4.2.3). |
| FR-OBS-007 | A pseudo-agent MUST NOT be counted in any `Agents` or `Households` field and MUST NOT be the subject of `AgentSpawned`, `AgentBankrupted`, `WeightChanged` or `RoleChanged` (3.3.6, 5.1.2). |
| FR-OBS-008 | A module that is off MUST emit none of its events, and the events of the other modules MUST equal those of a world in which it was never configured (3.3.3). |
| FR-OBS-009 | An aggregate event that the table marks "never when all are zero" MUST NOT be emitted in a tick in which all its counts are zero. |
| FR-OBS-010 | `WorldCreated` MUST be the first event of a constructed world, with `Tick` 0, the seed, the whole configuration hash, whether a warm start was present, the names table and the `Jurisdictions` rows, and MUST precede the `AgentSpawned` of each initial agent. |
| FR-OBS-011 | A restore MUST emit `WorldRestored` as its first event to the observers given to it, with the restored tick, the hash, the format, the reseed flag and salt, the names table and the snapshot at the restored boundary, and MUST emit no `AgentSpawned`, `AgentBankrupted` or `BoundaryFlow` for events that happened before the restore (the bucket of a world restored in construction is emitted at its first step, FR-OBS-015) (4.4.5). |
| FR-OBS-012 | `Run` MUST emit `RunStarted` and, whenever it did, `RunEnded` with the steps completed and the flags `Cancelled` and `Failed`, never both true; a call refused before its first step MUST emit neither. |
| FR-OBS-013 | Every step that emitted `TickStarted` MUST emit `TickEnded`, with `Failed` true when it returned an error, and a guard that refuses a step before `TickStarted`, and a step that ends in a panic (`spec-world.md` 3.5.4), MUST emit no `TickEnded`; a `Run` that ends in a panic emits no `RunEnded` and the run span stays open until the world is restored. |
| FR-OBS-014 | `BoundaryFlow` MUST be emitted only for a (region, reason, money or commodity) whose reason has the kind faucet or sink and whose total is non-zero, in the order of region, registry order of the reason, money before goods, then commodity, with `Commodity` `NoCommodity` for money (4.4.1). |
| FR-OBS-015 | The construction bucket MUST be emitted once as `BoundaryFlow` with `Tick` -1, at the entry of the first step of a world that is still in its construction phase and before its `TickStarted`, and MUST NOT be emitted by a world restored after its first step (4.4.2). |
| FR-OBS-016 | `ProductionHalted` MUST be emitted once per region and role per tick in which at least one household's job request was declined by the production halt, with one `Skipped` row for each recipe the halt skipped, MUST be counted by the world from the view and not reported by a strategy (`JobRequest` and `Work` are unchanged, `spec-agents.md` 7.1.5), and MUST NOT be emitted when `Halted` is 0 (4.4.6). |
| FR-OBS-017 | A call that sets a slot MUST emit one event per cell whose value changed, in the order of the key, after the whole call is applied, with the `Tick` of the next step and before its `TickStarted`, and MUST emit none for a cell set to the value it holds (4.4.4). |
| FR-OBS-018 | `RecipeExecuted`, `RecipeAbandoned` and the three `Traineeship*` events MUST carry the region of the agent at the time of the event (3.3.7). |
| FR-OBS-019 | `WeightChanged` MUST carry the money (signed, positive to the agent) and the units of goods that the change moved between the agent and the estate. |
| FR-OBS-020 | One removal MUST emit, in this order, `RecipeAbandoned` for its job, `TraineeshipEnded` if it was in a traineeship, `ContractSettled` for each contract it delivered, then the events of the removal steps of `spec-agents.md` 3.6 in step order (`TitleTransferred` 2b, `HireEnded` 2c, `LoanEnded` 2d, `LoanPassed` 2e, `CoverEnded` 2f, `FacilityChanged` 2g, each with its module on), `AgentBankrupted`, `EstateOpened` (and, for a trainer, its trainees' `RecipeAbandoned` and `TraineeshipEnded` first, 4.4.8), and `AgentBankrupted` MUST be the only event that removes the agent (4.4.8). |
| FR-OBS-021 | Within a region's entry step the completions of traineeships MUST be emitted as `TraineeshipCompleted` then the `RoleChanged` it causes, then each place filled as `TraineeshipStarted`, before the events of the population module and the spawner's plan (4.4.8). |
| FR-OBS-022 | The events of a step MUST be emitted in the order of `spec-world.md` 6.11 as amended by 4.4 and 4.5, each after the state change it describes. |
| FR-OBS-023 | A run MUST give identical snapshots, configuration hash, random state, draw count and event stream whether or not a logger, an observer or `WithAudit` is attached, and whatever the observer's speed. |
| FR-OBS-024 | With several `WithObserver` options every event MUST reach each observer in the order given, and a panicking observer MUST leave the world failed with the panic propagated (architecture.md 12.5). |
| FR-OBS-030 | A `Snapshot` MUST be a deep copy of plain data with no pointer into live state, no function and no map, and changing it MUST NOT change the world or a later snapshot. |
| FR-OBS-031 | `Snapshot` MUST return `ErrInStep` during a step and `ErrWorldFailed` on a failed world, and MUST draw no random number and change no state. |
| FR-OBS-032 | A snapshot MUST report agents and households as separate fields, with pseudo-agents in neither and a count of households the sum of weights. |
| FR-OBS-033 | The number of rows of a snapshot MUST depend on the configuration and on the sets in flight (shipments, away agents, journeys, contracts) and MUST NOT depend on the number of agents or households. |
| FR-OBS-040 | The agents and households by role and by region rebuilt from `AgentSpawned`, `AgentBankrupted`, `WeightChanged`, `RoleChanged` and `AgentRelocated` MUST equal the snapshot at every tick boundary. |
| FR-OBS-041 | The money of the households, of the pool and of the treasury sector of each region rebuilt from the events of 4.3.3 (with `WorldCreated.Jurisdictions`), plus `Snapshot.Construction` (before the first step) and `Snapshot.Pending`, MUST equal `MoneyHouseholds`, `MoneyPool` and `MoneyTreasury` at every tick boundary, a boundary after a call between ticks included. |
| FR-OBS-042 | The boundary balance of each reason MUST equal the construction bucket plus the sum of the `BoundaryFlow` events of every tick since, with exact equality, plus `Snapshot.Pending`, at every tick boundary of a run from construction, a boundary after a call between ticks included. |
| FR-OBS-043 | The snapshot of a world restored at tick `n` MUST equal, field for field, the snapshot of the uninterrupted world at tick `n`, and no field of a snapshot MUST be a function of anything that a checkpoint does not save (5.2.1). |
| FR-OBS-044 | `RegionState` of a region MUST equal the rows of the full snapshot whose key has that region (5.3.5a). |
| FR-OBS-045 | For each market in each step the sum of `TradeExecuted.Quantity` MUST equal `MarketCleared.Volume`, and the `MarketState` returned by the query MUST equal the snapshot's row. |
| FR-OBS-050 | Every instrument of 6.2 MUST have the name, kind, unit and attribute set listed there and a non-empty one-line description in the adapter's code (the descriptions are written with it; this catalogue fixes the rest), MUST be in the `economysim.` namespace, MUST NOT end in `_total` and MUST NOT carry its unit in its name. |
| FR-OBS-051 | Every `Warn` of 7.3 MUST have a counterpart event or snapshot field that exists in sections 4 and 5. |
| FR-OBS-052 | The attribute sets recorded on each instrument MUST be within the bounds of 6.5 for the configuration, and MUST contain no agent ID, order ID, tick, price, quantity, contract or shipment ID. |
| FR-OBS-053 | `economysim.household.unfed` MUST equal, at every tick, the sum over regions of the households unfed in the last complete step, by the rule of 6.3.1, including the tick in which no event is emitted. |
| FR-OBS-054 | An adapter attached at a restore MUST seed its level instruments from `WorldRestored` and MUST then agree with an adapter attached at construction, from the end of the first complete step after the restore, on every UpDownCounter and on every gauge that is fed every tick (`economysim.market.clearing_price`); a sparse held gauge (`economysim.endowment.stock`, `economysim.household.own_use`, `economysim.external.price`, `economysim.belief.centre`, `economysim.belief.at_bound`, `economysim.rent.rate`, `economysim.settlement.class`, `economysim.technology.gain`) agrees from the first event of its series (6.4.2). |
| FR-OBS-055 | A failed step MUST record no duration, MUST end its span with an error status and MUST run no reset: the level instruments are those after the events the adapter saw (6.3.1). |
| FR-OBS-056 | The adapter MUST NOT call the `World` and MUST NOT keep a value keyed by an agent ID, an order, a trade, a shipment or a contract. |
| FR-OBS-057 | `economysim.agent.removals` by `economysim.cause` MUST sum to the number of `AgentBankrupted` events. |
| FR-OBS-058 | `economysim.world.step.failed` MUST equal the number of `TickEnded` events with `Failed` true, and the recordings of `economysim.world.step.duration` plus it MUST equal the number of `TickEnded` events. |
| FR-OBS-059 | `economysim.household.exits` by `economysim.role` and `economysim.cause` MUST equal, with exact equality, the sum of `AgentBankrupted.Weight` and of `From` less `To` of each `WeightChanged` with `To` below `From`, over the events whose cause is not `population`. |
| FR-OBS-060 | `economysim.market.price_age` MUST equal `MarketCleared.PriceAge` of the last step for each (commodity, region), and `economysim.belief.at_bound` MUST equal `AtFloor × 10 000 / Outcomes` and `AtCeiling × 10 000 / Outcomes` of the last `BeliefsUpdated` of the market, truncated toward zero, and hold that value in a tick with no such event. |
| FR-OBS-070 | The library MUST emit no log record at `Info` or above inside a step other than the `Warn` rows of 7.3, and MUST emit no `Error` record. |
| FR-OBS-071 | Each `Warn` of 7.3 MUST have the level, the constant message and the keys of its row, and, except row 19, MUST be emitted by the pattern (A or B) of its row over the run of consecutive ticks its owner keeps. |
| FR-OBS-072 | In each uninterrupted run of `N` ticks no cell of a `Warn` other than row 19 MUST emit more than `ceil(N / warnRun) + 1` records of one message (and over any window `ceil(w / 2)` for pattern A), and row 19 MUST emit at most one per recipe per tick. |
| FR-OBS-073 | Every attribute key of every log record MUST be a key of the table of `rules/observability-and-logging.md` 3.3.2. |
| FR-OBS-074 | No scenario MUST produce a record at `Error`; an inconsistency the library cannot repair MUST panic or, with `WithAudit`, return `ErrAuditFailed`. |
| FR-OBS-075 | A log message MUST be a constant lowercase phrase with no interpolated value, and all records of one call site MUST have the same message. |
| FR-OBS-076 | "world created", "warm start applied", "run started", "run ended" and "world restored" MUST be emitted once each, at the point and with the keys of 7.2. |
| FR-OBS-080 | The golden rendering of an event MUST follow 9.5.2: `t=<Tick>`, the type, the fields in declaration order, typed IDs as their string IDs, no timestamp and no map order; and a field at its zero value (not a typed ID and not `Side`) MUST NOT be rendered, so that a field a module appends at its off value leaves no trace. |
| FR-OBS-081 | `testdata/golden/events-<pack>.txt` MUST hold the rendering of the first ticks of the `one` pack, one digest line per tick, the count of each event type that occurred (a type with a count of 0 is not listed), and the final snapshot, and MUST change only with `-update` and an explanation in the PR. |
| FR-OBS-090 | A world with no logger and no observer, and one given a nil logger or a nil observer, MUST build, step and run, write nothing to stdout or stderr and not panic. |
| FR-OBS-091 | A field that a module appends to an event or to a snapshot row MUST be a count, an amount, a bool, a constant of a closed set or a slice, MUST NOT be a typed ID, and MUST be zero when the module is off; the rendering of every event and row of a world with the module off MUST be byte-equal to that of the tree before the module was added (9.5.2; architecture.md 2.10.5). |
| FR-OBS-092 | `WearApplied` MUST be emitted once per tick for each commodity whose wear units of the tick are above 0, in ascending commodity order, after the tick's `BoundaryFlow` rows, with `Units` equal to the ledger's wear units of the commodity; the construction bucket MUST be emitted in the same form with `Tick` -1 after the construction `BoundaryFlow` rows; it MUST NOT be emitted for a tick with no wear. |
| FR-OBS-093 | `Markets[].Lots` MUST equal the number of lots of the commodity in the accounts counted by `HeldByAgents`, `HeldByEstate` and `HeldByTreasury` of the region, and a restored world's `Lots` MUST equal the original's (FR-OBS-043). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-OBS-001 | With the default logger and observer, each event-emission and logging call site that the tick loop reaches MUST allocate nothing (`testing.AllocsPerRun`), over the all-modules scenario. |
| NFR-OBS-002 | With the default logger and observer, a full `Step` of the reference chain MUST allocate no more than a baseline constant held in the test, and raising it MUST be a reviewed change with a reason. |
| NFR-OBS-003 | A counter that exists only to build an event MUST be built only when an observer was supplied, and an owner MUST state in its Observability entry which of its counters is which. |
| NFR-OBS-004 | The allocations of a `Snapshot` MUST be a function of the rows of 5.3 and not of the number of agents, which a test over two scales of one configuration MUST show. |
| NFR-OBS-005 | The adapter MUST call the OpenTelemetry SDK at most once per series per tick for the classes O and R, and MUST NOT call it per order or per trade. |

## 18. Open questions

1. **Event classes at the observer.** The classes O and R are 92% to 99% of the
   events of the first consumer's world and cost a boxing allocation each (8.3).
   `WithObserver(o, classes...)` would build and deliver only the classes an
   observer wants, and with no observer wanting a class the world would build none.
   It changes architecture.md 6.2 and the options table of `spec-world.md` 4.1 and
   needs the class of each event to be API (3.3.9 makes it a property of each row
   already). Not adopted: no measure shows the placeholder 6% to matter. Next step:
   the slice measures the allocation per event (N-015), then the supervisor decides.
2. **The bucket at construction.** The bucket event is emitted at the first step
   (4.4.2) because the consumer's calls before it add to the bucket. The
   alternative is an event at the end of `NewWorld` and one per call of `AddAgent`
   and of the slots that move goods, each with `Tick` -1. It lets an observer see
   the endowment before the first step, at the cost of several events and of the
   same position question for each call. Next step: the first consumer that asks
   to log the endowment at construction time.
3. **A snapshot that skips groups.** The scans of 5.5 (S, L and B) are paid by
   every `Snapshot()`, though most consumers read the cheap rows. A call that takes
   the groups to skip (`Snapshot(SkipBeliefs, SkipHoldings)`) is additive to the
   public API. Not adopted: about 2% of a tick at the placeholder sizes. Next step:
   the slice's measure of the snapshot at the first consumer's size.
4. **`WorldRestored` repeats the snapshot.** The levels of an adapter could be
   seeded by a smaller value than the whole snapshot (the rows of 6.4.2 only). The
   whole snapshot was kept so that there is one definition of the levels (6.4.3).
   Next step: a measure of the event's size for a world with many in-flight
   shipments.
5. **A rest counter.** `ProductionHalted` counts the halt and not the rest of the
   target-income rule (4.4.6). A study of the labour loop L14 may want the
   households that rested by commodity. Next step: scenario S04's first run.
6. **Presentation labels in the names table.** `Region.Label` and
   `World.DistanceLabel` are presentation-only fields outside the hash
   (`spec-config.md` 8.3); the names table carries IDs only, and a consumer that
   wants a label reads `Config()`. If a dashboard needs labels in events' tables,
   the names table gains a `Labels` section with the same indexing.
7. **The unit of `economysim.welfare.ratio`.** `{basis_point}` follows the basis
   points the welfare ratio is held in (`spec-welfare-and-wages.md` 2.1); a
   consumer that wants a ratio of 1.0 divides by 10 000 outside the core
   (architecture.md 3.5). Next step: the first dashboard.
8. **The rule that every event has a counterpart in a metric.** Some events feed
   no instrument (`EstateOpened`, `ReliefPaid`, `WeightChanged` beyond the counts,
   `ShipmentRejected`, `RecipeAbandoned`, `EndowmentAdjusted`). They are for a
   recorder, a trace or a log. If one is missed on a dashboard it is a new row of
   6.2, derivable from the event already there (3.1.4). Answered: IND adds no event and no
   metric (`spec-indicators.md` 14).

## 19. Edits owed to other files

One consolidated table: every file, section and change. The supervisor applies
them; the handoff is derived from this table. None alters a result that a test
could depend on (nothing has run), except where the row says an event or a metric
changes shape. Rule files need a fresh read-receipt tag each.

### The rules

| File | Section | Change |
|------|---------|--------|
| `doc/rules/observability-and-logging.md` | 3.3.2 | add the keys `format` (the checkpoint format version; logged as `slog.Int64(int64(f))`), `bytes` (the size of an encoded checkpoint, `int64`), and `from` and `to` (a value before and after, in basis points, `int64`; MON's "money flow scaled", unless MON renames them, routed), in the same change as the code that logs them; add to the `reason` row that the values of the "run ended" line are `completed`, `cancelled` and `failed` (WLD) |
| | 4.1 | add a paragraph: every field that names a configured entity carries the typed integer ID of its kind, a reason, a cause and a side are constants of a closed set, and the strings of the IDs are in the names table that `WorldCreated`, `WorldRestored` and the snapshot carry (3.2 here); add that an event of a transition of one agent carries the agent's region (3.3.7) |
| | 4.3 table | add the rows `WorldRestored`, `ProductionHalted`, `YieldShocked`, `TransitFactorChanged` and `PopulationScaled` with the fields and positions of 4.1 here; change `WorldCreated` (+ the names table), `RunEnded` (+ whether the run failed), `TickEnded` ("each `Step` that began ends, a failed one with `Failed`"), `BoundaryFlow` ("faucet or sink reasons only; the construction bucket at `Tick` -1"), `RecipeExecuted`, `RecipeAbandoned` and the three `Traineeship*` (+ the region), `WeightChanged` (+ the money and units moved), `ReliefPaid` (+ the households) |
| | 4.3 paragraph on aggregates and on order | add `ProductionHalted` to the aggregates; add `PopulationScaled` (population) and `TransitFactorChanged` (transport) to the gated events; add to the order: the bucket's `BoundaryFlow` at the entry of the first step before `TickStarted`, `ProductionHalted` after `OwnUseChanged` in phase 2, the traineeship events first in a region's entry step, and the slot events at the call |
| | 4.4, 6.1 | the wording WLD 22 owes: "after the state change it describes" reads "after the end of the step" for a failed step; 6.1 closes the span at `TickEnded` (FR-OBS-013) |
| | 4.5 | a sentence that a counter that exists only to build an event is built only under the flag (NFR-OBS-003) |
| | 5.2 | a Gauge is "not additive, or a level set from a snapshot" (6.1.3) |
| | 5.3 | the catalogue of 6.2.1 and 6.2.2 of this spec: rename `economysim.agent.bankruptcies` to `economysim.agent.removals` (+ `economysim.cause`); rename `economysim.market.orders.unmatched` to `economysim.market.unmatched` in `{unit}`; add `economysim.region` to `economysim.market.clearing_price` and `economysim.external.price`; add `economysim.cause` to `economysim.household.relocations`; add the new instruments `economysim.market.orders`, `economysim.market.orders.rejected`, `economysim.belief.centre`, `economysim.production.halted`, `economysim.entry.limited`, `economysim.commodity.flow`, and the four snapshot-fed gauges |
| | 5.3.1 | the feed table follows; the sentence "a new metric must be derivable from the existing events" reads "from the events or from a snapshot field, and the table says which"; the rule of 6.3.1 for `economysim.household.unfed`; the snapshot-fed instruments and `RecordSnapshot` |
| | 5.4.1 | the attributes `economysim.cause`, `economysim.entry_limit`, `economysim.holder` and `economysim.sector` with their constants (6.5.1); `economysim.reason` also takes the rejection reasons and the goods boundary reasons |
| | 6.2 | a class-O or class-R event is not a span event; the step span carries the counts as attributes (6.6.5) |
| | 8.1 | add the tests of section 9: restore equality of the snapshot, the rebuild of counts and money from events, `WorldRestored`, the counterpart of each `Warn`, the golden rendering of events, the adapter's calls once per series |
| | read receipt | a fresh tag |
| `doc.go` (glossary) | | **names table**, **volume class**, **recorder**, **level** (of an instrument), **delta instrument**, **region state**, **snapshot row**, **pending totals**, **construction flow**, **run** (of a bounded `Warn`), **pattern A** and **pattern B**, `warnRun`, `NoCommodity`; the public types `Names`, `Kind`, `ScaleKind`, `HaltRow`, `ArrivedLine`, `MarketRow`, `RegionState`; `WantID` as a typed ID |
| `doc/rules/architecture.md` | 2.6 (and 2.9 stands) | the description of `internal/logkeys` reads "slog attribute key strings; it imports nothing and holds no type of the root"; the reason and cause constants and the event field types are root exports (`event.go`) (R3, ARCH 2.9 wins); 12.5 as the rules 4.4 row above |

### The specs

| File | Section | Change |
|------|---------|--------|
| `spec-world.md` | 3.4.2 | the row "Step `k+1` failed": `RunEnded` has `Failed` true (the note "OBS may add a flag" is answered) |
| | 6.2 | a position **P0.2a**: at the entry of the first step of a world still in its construction phase, before `TickStarted`, the construction bucket as `BoundaryFlow` with `Tick` -1 (4.4.2) |
| | 6.4 | a position **P2.8a** after P2.8: `ProductionHalted`, region then role ascending (4.4.6) |
| | 6.11 | item 1: the bucket's `BoundaryFlow` before `TickStarted` in the first step; item 3: `ProductionHalted` after `OwnUseChanged`; item 8: the traineeship events first in a region's entry step (4.4.8); item 2: `MoneyFlowLimited` then `MoneyFlowEnded` of one flow (K7) |
| | 8.2 | the "Event at the call" column of the yield shock, the transit factor and the birth and death scales names `YieldShocked`, `TransitFactorChanged`, `PopulationScaled` |
| | 12.2 | the event of a restore is `WorldRestored` (4.4.5) |
| | 13 (OBS row), 16.1, 16.2, 22 (questions 5 and 9) | answered: `RunEnded.Failed` and `TickEnded.Failed` accepted; `TickEnded` is not a summary; the three slot events exist; `WorldCreated` gains the names table |
| `spec-ledger.md` | 7.4.1 | the construction bucket is emitted as `BoundaryFlow` with `Tick` -1 at the first step (open question 9 answered); the snapshot holds its rows only before the first step (5.2.4) |
| | 12.1, FR-LED-031 | `BoundaryFlow` is for the reasons of kind faucet or sink; `Commodity` is `NoCommodity` for money; the total is a magnitude and the kind says the direction; the order is the registry order |
| `spec-production.md` | 14.1 | `RecipeExecuted` and `RecipeAbandoned` gain `Region` |
| `spec-skill-barriers.md` | 4.7, 8.1, FR-SKL-021 | the three events gain `Region`; the order of 4.4.8 |
| `spec-estates.md` | 2.5, 2.6, FR-EST-020, FR-EST-021 | `ReliefPaid` gains `Households`; `EstateOpened` is emitted by every removal, with the pool policy on or off; `ReliefPaid` is the pool policy's |
| `spec-spawner.md` | 13.1, 13.4 | `WeightChanged` gains `Money` and `Units`; the snapshot's last-tick `EntryLimited` counts are the event, not snapshot fields (5.8) |
| `spec-agents.md` | 14.6 | the snapshot list reads `spec-observability.md` 5.3.4 here |
| `spec-agents.md` | 6.3, code-home row | `Side` is declared in `event.go` (not in `agent.go` or `strategy.go`) and has a `String` method (`bid`, `ask`) for the golden rendering of events (F8); applied 2026-10-12 |
| `spec-demand.md` | 11.5, 3 or 9 | the per-want counts are `WantShortfall`, not snapshot fields; the type `WantID` is named |
| `spec-subsistence.md` | 14.4 | `Blocked` is the event `OwnUseChanged`, not a snapshot field |
| `spec-migration.md` | 13.4 | the last tick's `MigrationReviewed` counts are the event, not snapshot fields |
| `spec-population.md` | 10.4 | the snapshot list reads 5.3.4 and 5.3.10 here; `WeightChanged.Money` and `.Units` |
| `spec-money-supply.md` | 13.4 | the snapshot list reads 5.3.2 and 5.3.3 here |
| `spec-regions.md` | 10.1, open question 4 | answered: events carry the `RegionID`; the strings are the names table; the relocation causes are `transport`, `migration`, `consumer` |
| `spec-resources.md`, `spec-information.md`, `spec-migration.md` | 14.1; 10.1; 13.1 | the question of the index or the string ID is answered (3.2) |
| `spec-reference-chain.md` | 9 | the rows marked "not stated" and "missing" are answered (5.3.4, 5.3.5, `ProductionHalted`, 4.4.6); `start_money` and `relief` are told by `AgentSpawned` and `ReliefPaid`, not `BoundaryFlow` (4.4.1) |
| `spec-stability.md` | 12.1, 12.2, FR-STB-012 | the production halt is counted by `ProductionHalted`; the map of regimes to measures is 10.2 here |
| `spec-beliefs.md` | 12.4 | the metric is `economysim.belief.centre` (6.2.1) |
| `spec-calendar.md` | 8.2, open question 4 | answered: no season event (4.4.7) |
| `spec-config.md` | 10.3 | `WorldCreated` gains the names table |
| `spec-clearing.md` | 5.4, 5.5, 7.1 (a request to CLR and CHK) | the market's last report (`Volume`, `UnmatchedBid`, `UnmatchedAsk`, the best quotes and their flags) is state saved in a checkpoint |
| `spec-checkpoint.md` | restore, the bucket, the log keys | 10.4 |
| `spec-scale.md` | 9.2 | the snapshot fields are 5.3.2 and 5.3.4 here |
| `spec-world.md` (also) | 5.5.3, 8.4.1, 12.1 and 12.2 (the clause "a restored world is never in construction"), FR-WLD-101, the note of P0.3 "`TickStarted` the first event of the step" | a world restored from a checkpoint taken before its first step is in its construction phase and keeps its bucket (CHK FR-CHK-026, 044); P0.2a precedes `TickStarted` in that step; the bucket is not saved only after the first step |
| `spec-world.md` (also) | 6.5 P3.4 and 6.11 item 4 | the estates' asks emit `OrderPlaced` and may emit `OrderRejected` (listed in 4.1.2) |
| `spec-ledger.md` (also) | 11 | the bucket is saved while the world is in its construction phase (CHK 4.4) |
| `spec-checkpoint.md` (also) | 14.1 | `WorldRestored.Format` is the `uint32` of `CheckpointInfo.Format` (R12; settled here, no CHK change); the "log key `format`" is `slog.Int64(int64(f))` |
| `spec-checkpoint.md` (also) | 4.2 tags 9 and 14, FR-CHK-043 | with SPN 14 and POP 11 below, the runs of "group below floor" and "region extinct" are saved |
| `spec-agents.md` (also) | 7.1.5, 6.3, 14.3 | **one** request: the halt count for `ProductionHalted` is the **recipe IDs** the halt skipped, per household that `Work` was called for ((computed by the world as a pure function of the view, from stock and `OutputTarget`, as it computes the floor in 7.1.3; `JobRequest` and `Work` are unchanged, AGT 7.1.5)); "declined for the halt" is AGT 14.3's reason `halted`; an agent with no work share (`rest`) is `rest`, not `halted`; households are the agent's weight, not its `Executions` |
| `spec-agents.md`, `spec-welfare-and-wages.md` (also) | 7.1.3, 14.6, FR-AGT-072; 5.3, 11.2 | say that the count of the households held by the work floor at the boundary (`RoleRow.HeldByFloor`) is evaluated over the agents with no job in progress on the record after P6.3 and may differ from the count in the "work floor binding" `Warn` (F16) |
| `spec-external-sector.md` (also) | 8.3, 15.4 | 8.3 "the boundary accounts are per reason and region" reads "one per reason" (`spec-ledger.md` 3.4); 15.4's cumulative balance of trade per node is dropped from the snapshot and `import_c`, `export_c` are the sums of the sides' `Units` per commodity (F2) |
| `spec-merchants.md` (also) | 15.1.1 | `ContractSettled` gains `Region` (the deliverer's); "An event of a module that is off is not emitted" says that `MerchantsPlanned` is gated by the merchant block (F4, K8) |
| `spec-money-supply.md` (also) | 13.2; 13.4, FR-MON-042; 3.1.3 | "money flow scaled" uses the keys `from` and `to`: either the rules gain `from` and `to` (above) or MON uses other words, MON decides; "sorted by reason" reads "in the order of `spec-observability.md` 4.4.1.2"; the per-role money per household of 3.1.3 cannot be computed from the snapshot, a gap in 13.4 |
| `spec-production.md`, `spec-skill-barriers.md` (also) | PRO 5.4; SKL 4.2, 8.1 | the order of a trainer's removal (4.4.8): its trainees' `RecipeAbandoned` (`principal_left`) and `TraineeshipEnded` (`trainer_left`) first, in agent ID order; a completion's `RecipeAbandoned` (`role_changed`), then `TraineeshipCompleted`, then `RoleChanged` |
| `spec-spawner.md` (also) | 13.3, FR-SPN-046, FR-SPN-036, FR-SPN-047 | FR-SPN-046 ("the snapshot MUST hold the last tick's `EntryLimited` counts") reads "the event"; "group below floor" says it is pattern A with the qualifier "weight above 1, at run time"; "role extinct" carries the qualifier of FR-SPN-042 |
| `spec-subsistence.md` (also) | FR-SUB-046 | "households blocked in the last tick" are the event, not a snapshot field |
| `spec-integration.md` (also) | 14.3 | the worked examples write `YieldShocked{BeforeBP, AfterBP}` and not `{From, To}` |
| `spec-clearing.md`, `spec-spawner.md`, `spec-production.md`, `spec-regions.md`, `spec-integration.md` (also) | CLR 3.2 and its "where this lives"; SPN 3.3, 13.1.1 and its "where this lives"; PRO 14.2 and its "where this lives"; REG 6.1, 7.3, 10.3 and its "where this lives"; INT 12.2, 20 | every "constant of `internal/logkeys`" reads "exported constant of the root package (`event.go`); `internal/logkeys` holds the key strings only" (R3); INT's cause `consumer` is a root constant |
| `AGENTS.md` | layout | `internal/logkeys` is described as "slog attribute key strings" |
| `spec-spawner.md` (also) | 14 | save the run of ticks of "group below floor" per (region, role) (R7), or say the `Warn` is a function of saved state |
| `spec-population.md` (also) | 11; 10.3 | save the run of ticks of "region extinct" per region (R7); 10.3 conditions say "for ten consecutive ticks" while the header says pattern A: drop the clause from the conditions of "births unfunded" and "deaths held", or say pattern B in the header (R11); OBS follows what POP says |
| `spec-integration.md` (also) | 12.2 "slot set"; 6.3, 6.5 | the Debug line uses keys the rules' table lacks (`rule` naming a call, a count of keys): restrict to the table's keys or add them with the code; `MarketState.WindowLen` is an `int64` (every count is); `Region(r)` returns `RegionState` (5.3.5a) and for an outside node `ErrOutsideNode` |
| `spec-ledger.md` (also) | 12.2 | the Debug line of a refused transfer uses account IDs and amounts: keys `account` and `amount` are not in the rules' table, use `agent`/`quantity`/`value` or add them |
| `spec-credit.md` | 16.1, 16.3 | say whether `CreditSettled` counts the payments of a claim of origin `payout` (the rebuild of `spec-observability.md` 4.3.3 item 10 needs them) or only those of a loan, and give a field for the other if the instruments `economysim.credit.*` are to count origin `loan` only (the rules 5.3.1); `Origin` on `LoanEnded` and on the `Claims` row (4.1.10, 5.3.8) |
| `spec-insurance.md` | 15 | the events are `CoverFormed` and `CoverEnded`, the key `cover_policy`, the Debug lines "cover formed" and "cover ended"; INS counts its own outstanding payout claims in an instrument of its own (not in the catalogue of 6.2.1 yet) and feeds `economysim.credit.outstanding` by nothing |
| `doc/notes.md` | the open notes of this spec | in the handoff |
| `doc/specs/README.md`, `doc/spec-roadmap.md`, `doc/documentation_index.md` | the OBS row, the existing-specs table, the parameter owners (none) | OBS exists |

## 20. Where this lives in the code

| Concern | Home |
|---------|------|
| the event types, the sealed `Event` interface, `Names`, `NoCommodity` | root package, `event.go` |
| `Observer`, `ObserverFunc`, the delivery loop, the "has observer" flag and the default observer, the unexported `discardObserver` with `var _ Observer = discardObserver{}` (architecture.md 4.5) | root package, `observer.go` |
| `Snapshot`, its rows, `RegionState`, `MarketRow`, the scans and the builder shared with `WorldRestored` | root package, `snapshot.go` |
| `Reason`, `Cause`, `Kind`, `ScaleKind` and their constants (and `Side` with its `String` method); `warnRun` | root package, `event.go`, `observer.go` |
| the log key strings (and nothing else; no alias, no import) | `internal/logkeys` |
| the instruments, the accumulators, `RecordSnapshot`, the spans | `otelobserver/` (separate module) |
| the golden rendering (a test helper) and the golden files | `*_test.go` of the root package, `testdata/golden/events-<pack>.txt` |
| the recorder and the capturing handler | `_test.go` files (architecture.md 10.3) |
| no model | `doc/specs/models/obs/` does not exist (15) |
