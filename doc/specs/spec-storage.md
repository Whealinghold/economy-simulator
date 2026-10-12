# Spec: Storage: a store as room on an agent's account, capacity limits that act where a decision is made, the cost of a store as a named money flow, and the seams for risk, infrastructure and a large store held against shortage

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-STO-NNN` and
`NFR-STO-NNN`, which tests cite (testing.md section 4). They are in section 20.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md),
decision 2). It is off unless `Config.Modules.Storage` is present, and with it off
nothing in this spec runs, no account has a room, no state is added to an agent, no
reason is declared and no event is emitted: the economy is the one in which an agent can
hold any quantity up to `MaxQuantity` of a commodity (`spec-agents.md` 8). It needs the
ledger (`spec-ledger.md`) and the commodity attributes (`spec-goods.md` 3). It uses
hooks in the order book (`spec-clearing.md` 3), production (`spec-production.md` 4,
5), transport (`spec-transport.md` 5), the agent and its default strategy
(`spec-agents.md` 6, 7), the money flows (`spec-money-supply.md` 4, 10.3), the
relocation guard (`spec-regions.md` 6.5), the spawner (`spec-spawner.md` 4.2),
integration (`spec-integration.md` 4, 6), the checkpoint and observability, and it
reserves seams for risk (`spec-risk.md`), infrastructure (`spec-infrastructure.md`),
settlement size (`spec-settlements.md`), a ruler's reserve stores (`spec-fiscal.md`) and
policy. Every change it needs in another spec is listed in the handoff and is not made
here (section 19 states them).

Nothing has run in the library and no acceptance band exists (notes N-015, N-027).
Every number here is a **placeholder**: a default, a bound or a number of an example,
and none is called stable. The one script of section 17 is a stylised Docker Python
model of the rules of sections 5 and 6, not a claim about the library. The settings of
the worked examples (18) are invented **examples**, and the words in them (`grain`,
`grower`, `ration`, `steward`, `depot`) are flagged as such and are data, never
branched on (`AGENTS.md`, N-017). The first consumer calls a large store held against
a season of shortage a granary; that word is a pack's, and it appears in this spec in
this sentence and in the flagged examples only. The normative text says **buffer
store**.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

Until this module, an agent can hold any quantity of any good: `spec-agents.md` 8
decided that version 1 has no inventory capacity, because a capacity needs a rule of
what to discard, a cost and a container, and the container is this spec's. Every real
store is finite, and keeping goods in it costs something: that limit and that cost are
what this module adds to a ledger of quantities that otherwise lets an agent hold
anything for nothing. This spec covers:

- what a **store** is, and the **room** of an account (section 3);
- the ledger's part: the room, the load, the pledge and the refusal (section 4);
- **where room acts**: the commitment rule, and what it does to a bid, to the start of
  a job, to a shipment and to every other way goods reach an account (section 5);
- what an agent **sees and does** with its room: the view, the target and the default
  strategy's rules (section 6);
- the **cost of a store** as a named money flow (section 7);
- **spoilage, wear and loss** in storage, and the chargeable size of `spec-goods.md`
  3.3 (section 8);
- **region binding** (section 9);
- a **buffer store** and the seams to the later specs (section 10);
- the position in the tick, the parameters, validation, observability, saved state,
  the register entries, what was run, worked examples, requirements and open questions
  (sections 11 to 22).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| Who builds a store, what it costs to build and who pays; depreciation and maintenance of a facility | `spec-infrastructure.md` (CAP). This spec has the internal operation CAP calls (3.6) and the public call a consumer makes (3.6.2) |
| Renting room to another agent, a keeper's business, a lease with a price | not in version 1 (open question 2). It would be a contract of the kind of `spec-merchants.md` 9 over the same room, with `spec-land.md` or `spec-institutions.md` for tenure and charters |
| Theft, fire, a flood, a raid on stored goods; the `loss` sink | `spec-risk.md` (RSK). This spec names what RSK may read (8.3) |
| A ruler's treasury and the release of a buffer store as a policy | `spec-fiscal.md` (FIS), `spec-policy.md` (PLC). This spec supplies the hold, its room and its cost (10) |
| Which settlement size may have a store | `spec-settlements.md` (SET), through a registered gate (10.3) |
| A store that slows the ageing of goods (a cooled or sealed store) | not in version 1 (open question 1; 8.1 records why and the rule it would need) |
| A per-unit price of space | not in version 1 (7.4, open question 3) |
| What an agent buys or sells for a buffer store, and why | the agent's `Strategy`. The default strategy only respects its room (6) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | A store is **room added to its owner's own account**, not an account of its own. The goods stay in the agent's account and are used, sold, eaten and aged as ever; the room is a pair (mass, bulk) the account may hold, tested exactly by the pair of `spec-goods.md` 3.2 | 3 |
| 2 | An account is **limited** only if the agent's role has a base room; a role with none, and every pseudo-agent, is unlimited as before. The limit of an account is `w` times the base room of a household plus the capacity of its stores | 3.3 |
| 3 | A store has **one owner and one user**, the agent whose account it extends. No agent can use another's room in version 1; renting is a named seam | 3.5 |
| 4 | Room is **committed where a decision is made** and nothing is lost for lack of room, bar one case (15): a bid is admitted only with room for it, a job starts only with room for its output, a shipment is dispatched only with room at its consignee. After the decision every later step fits by construction | 5.1 |
| 5 | The **pledge**: the output of a job in progress and the cargo on its way hold room in the account that will receive them. It is derived from the job and the shipments, never saved | 3.4, 4.3 |
| 6 | **Production is all or nothing at the start, not at the completion.** A job starts with the largest number of executions whose output fits, or is refused with `no_room`; the completion does not fail and, with the room as it was at the start, disposes of nothing (`spec-production.md` 5.3.2 stands for `MaxQuantity`; position 15 is the other case) | 5.3 |
| 7 | A **target** is scaled to the room, so that the belief rule is not told to want what cannot be held (`spec-beliefs.md` 4.4); the default strategy sizes its bids to the free room in the order of its tiers, and discards surplus only to make room for upkeep | 6 |
| 8 | The cost of a store is a **fixed levy per store**: a `MoneyFlow` whose target is a class of stores, its reason the flow's ID, closed for a run. A sink, so no new faucet, no new built-in reason and no new money mechanism | 7 |
| 9 | **Ageing and wear are unchanged**: a store does not slow a clock and a durable wears by use. Preservation is not modelled; a rule that would model it is recorded | 8.1 |
| 10 | **The chargeable size and `BulkFactor` are not used**: capacity is a pair tested exactly, and the cost is per store, so no per-unit price of space exists and the over-charge of `spec-goods.md` 3.5 does not arise (N-011 applied) | 8.4 |
| 11 | A store is **tied to a region**: an agent that owns one is refused by the relocation guard with the `region_bound` refusal `has_store`, and a migrant or a forced move carries at most the base room (N-120 applied) | 9 |
| 12 | The module adds **no tick position, no draw and no ledger reason**. Its acts are inside positions that exist (11) | 11 |
| 13 | A role change that leaves goods above the new role's room moves the excess to the estate with the existing reason `estate_goods`, by the rule that already moves goods outside the new tradable set; a role change to a role with no base room ends the agent's stores | 5.5.2, 3.6.1 |
| 14 | The **upkeep room**: a job start and a dispatch leave free the load of one tick's upkeep of the agent that receives the goods, so a commitment never takes the room in which the household must buy its food | 3.4.5, 5.3.2, 5.4.1 |
| 15 | A **cut of the room** (a weight reduction, a role change) is the one case in which a commitment can outlive its room. The load is scaled or moved so that it fits, and what the pledge can no longer find room for is cut at the credit: a job's output is disposed (`disposal`) and a cargo is diverted to the estate, both counted | 5.3.4, 5.4.4, 5.5 |

## 3. The store and the room

### 3.1 Two sources of room

3.1.1. The **room** of an account is a pair of non-negative integers, a **mass
room** and a **bulk room**, in the mass units and bulk units of `spec-goods.md` 3.1. A
quantity of goods **fits** a room if and only if both sums of 3.4 are within it
(`spec-goods.md` 3.2: the pair, never a single number).

3.1.2. Room has two sources, and no other:

- the **base room** of a household of the agent's role, a pair per household. It is
  the household's own space (a home, a plot, a hold) and it goes with the household:
  it is not tied to a region and it is a function of the role (3.3), so a role change
  changes it (5.5.2);
- the **stores** the agent owns: facilities with a capacity pair, each of a **store
  class** (3.2), sited in the region of the account (9).

3.1.3. The base room is where the module starts: with it off, or on and no base room
given, an account has no limit (3.3). A pack that wants only a household's own limit
gives its roles a base room and no stores. A pack that wants a buffer store gives one
agent a store (3.6).

### 3.2 The store class

A **store class** is data: a capacity pair and a label, keyed by ID like every
configured entity (architecture.md 9.1; `spec-config.md` 3.3):

```go
// StoreClass is a kind of store. Its capacity is a pair (spec-goods.md 3.2).
type StoreClass struct {
	ID       string // matches the ID pattern of spec-config.md 3.3
	Mass     int64  // mass room one store of the class adds
	Bulk     int64  // bulk room one store of the class adds
	Needs    string // a flag of spec-settlements.md the region's class must carry; "" = none (3.2.3)
	Label    string // presentation only; not hashed
}
```

3.2.1. A class has no price, no durability and no cost field. What a store costs is a
flow of money (section 7) whose target names the class; what it takes to build one is
CAP's. A class with both capacities 0 adds nothing and is `ErrStoreShape`.

3.2.2. **No code names a class.** Nothing branches on a class ID (architecture.md 9.1).
A class is a key that a flow, a role row and a call name, and that a later spec
(RSK, SET) may key a rule on in data.

3.2.3. **The gate: `Needs`.** `Needs` names a **flag** of the settlement module
(`spec-settlements.md` 6.2: a permission a class of settlement carries), so that a
settlement's size can unlock a kind of store. The predicate is `regionHas(region, Needs)`,
read through the one function of that spec and for the value set at its last review, so
the value is the same at a call between ticks and at a position (`spec-settlements.md`
5.2, R1). It is tested at **`AddStore`** (refusal `ErrStoreLocked`, 3.6.2) and at the
**creation of an agent** that has the class in its role row: a store whose flag the
agent's region lacks is **not given**, and it is reported by the count `StoresLocked` of the
`AgentSpawned` event of that agent (14.1: an entry's creation is at P7.1(d), where no
per-tick counter survives to the next report) and warned at construction
(`store_locked_at_start`, 13.2). A store already owned is never taken
away when a class changes: a region that falls below the flag keeps the stores it has and
may not add the kind. With `Needs` empty there is no gate. With `Needs` non-empty and the
settlement module off the configuration is `ErrModuleDependency` (R5 there), and a flag
no class carries is `ErrUnknownSettlementFlag` (`spec-settlements.md` 9.1).

### 3.3 The room of an account

3.3.1. The agent's **base room** is, in order of precedence, the `Base` of the row for
its role in `Storage.Roles` (a row with no `Base` means no base room and so an unlimited
role), else `Storage.Base`, else none. An agent is **limited** if and only if its role
has a base room. A base room can be small, as small as the load of one tick's upkeep
that validation requires (`room_below_upkeep`, 13.1): the keeper of a buffer store has
that and a store (18.2).

3.3.2. For a limited agent of weight `w` that owns the stores `s_1 .. s_n`:

```
Room_d(a) = w * Base_d + sum_i Cap_d(class(s_i))          d in {mass, bulk}
```

with checked arithmetic (architecture.md 3.5, `spec-scale.md` 4.2: quantities of a
cohort are `w` times the household amounts, and a store is the agent's, not a
household's, so its capacity is not multiplied). The first consumer's world has weight 1
(N-030).

3.3.3. A **pseudo-agent** is never limited: an estate holds what the dead left and
what the pool receives, and the counterparties of the outside are swept to zero each
tick (`spec-estates.md` 2.7.4, `spec-external-sector.md` 4.2). They have no role and no
stores. A treasury of a jurisdiction (`spec-fiscal.md` 9.6) is a pseudo-agent of this kind: it is unlimited and owns no store. An agent of an unlimited role is a legal state and the default of every
configuration that does not name the module.

3.3.4. **Room is not state.** The room of an account is a function of its role, its
weight and its stores; it is rebuilt at a restore (15) and is read by every position
that needs it. A weight change changes it at once and a store call changes it at the
call.

### 3.4 Load, pledge, reserved and free

3.4.1. The **load** of a limited account is the pair of sums, over the commodities it
holds, of quantity times `Mass` and of quantity times `Bulk` (`spec-goods.md` 3.2),
checked. The ledger keeps it (4.1).

3.4.2. The **pledge** of an account is the pair of the room that goods not yet in it
will take: the pending output of the agent's job (`spec-production.md` 5.1.1, fixed
when the job starts) and the lines of the shipments that have the agent as consignee
(`spec-transport.md` 5.2; a self-carried trip to oneself has no lines, its goods stay in
the account, `spec-transport.md` 5.1). It is derived from the job and the shipments and
recomputed whenever one changes (4.3).

3.4.3. The **reserved room** of an account is the room of the bids it has admitted in
the current tick (5.2). It exists only inside a tick, like every reservation
(`spec-ledger.md` 6.5).

3.4.4. The **free room** of a limited account, per dimension, is

```
Free_d = Room_d - Load_d - Pledge_d - Reserved_d
```

and it is not negative at any commitment: a bid, a job start and a dispatch are tested
against it and made only with it non-negative afterwards (FR-STO-005). It is negative in
one case, after a **cut of the room** below what was already committed (a weight
reduction or a role change, 5.5); it is then read as 0 and every new commitment is
refused until the pledge is credited. The hard identity, true at every point, is that
`Load + Reserved` is within the room (4.4).

3.4.5. The **upkeep room** of an account is the pair

```
U_d(a) = sum_c Reserve_c(a) * size_c,d          d in {mass, bulk}
```

the load of the units its agent will eat at the end of the tick (`Reserve_c` is
`spec-agents.md` 7.3: `w` times the effective upkeep per household, 0 for a role with no
upkeep; the same quantity as `ResLoad_d` of 6.2.2). It is not reserved and is not a
second kind of load: it is the room that **a commitment must leave free**. A job start
(5.3.2) and a dispatch (5.4.1) are tested against `Free_d - U_d`, so that the room in
which the household buys its upkeep is never taken by a pledge, which nothing can
discard (6.2.3, STO-A3). A bid is not tested against it: the default strategy places its
tier 1 bids first (6.3.1) and discards surplus for them (6.3.2).

### 3.5 Who may own and use a store

3.5.1. **Owner and user are one agent.** A store extends the account of the agent that
owns it; only that account's goods are in it and only that agent decides about them.
Every agent of a limited role may own stores: a household agent, an external agent
(the player, `spec-integration.md` 4), a replacement or an entrant. There is no
ownership by an estate, by a region or by the pool, or by a treasury (`spec-fiscal.md` 9.6.1), because a pseudo-agent has no
strategy and no role (3.3.3). A buffer store of a ruler is owned by an agent that stands
for it (10.1); a granary of a treasury is FIS's buffer stock over the treasury's goods, not a store.

3.5.2. **Why not a store that several agents use.** Two agents in one room need a rule
for who gets the space when it is short, and the library has one such rule already for a
scarce endowment, the ration (`spec-resources.md` 5.3), which exists to prevent the
first-come ordering that `spec-world.md` 10 registers as a hazard. A shared store would
need a ration of space each tick, a record of who holds how much, and a price. That is a
keeper's business (open question 2). Version 1 gives each store one owner, so the
contention for room is inside one agent and is settled by its own plan order (5.2.3).

### 3.6 How an agent gets a store

3.6.1. **At creation.** An agent created in a role is given the stores that the role's
row lists (`RoleRoom.Stores`, 12.1), one per entry, at the creation and with no event:
the stores are a function of the role at creation, so an observer derives them from
`AgentSpawned` and the names table (14.1), except a class whose `Needs` flag the region's
class lacks, which is not given and is counted in that event (3.2.3).

3.6.1a. **A role change.** Stores are the agent's and the base room is the role's. A role
change **to a role whose resolved base room exists** (3.3.1) leaves the stores with the
agent and re-evaluates the room (5.5.2). A role change **to a role whose resolved base
room is nil** makes the account unlimited, and an unlimited account owns no store
(FR-STO-039): the stores end with it. Each is removed, in class order, with a
`StoreChanged` (`Kind` `removed`, `Cause` `role_changed`) emitted after the `RoleChanged`
of the same change, the levy of 7 ends with the next P1.7, and the refusal `has_store`
(9) ceases. The goods stay: an unlimited account holds anything. This is the one case in
which the library takes a store from an agent, and it is the case in which the store
would add no room; `spec-spawner.md` 4.2 gains the sentence (section 19).

3.6.2. **By a call between ticks.** The consumer adds and removes a store by two calls of
`spec-integration.md` (section 19 lists the edit), which act between ticks, are atomic,
emit `StoreChanged` at the call and are saved in a checkpoint (`spec-world.md` 8.1):

```go
// AddStore gives the agent one more store of the class.
func (w *World) AddStore(a AgentID, class StoreClassID) error
// RemoveStore takes one store of the class from the agent.
func (w *World) RemoveStore(a AgentID, class StoreClassID) error
```

The class is a `StoreClassID`, the typed ID of a configured entity (architecture.md 3.3;
`spec-integration.md` 3.4.1): its position in `Config().Storage.Classes`, in the sorted
order of the effective configuration. A consumer persists the class's string ID and
resolves it once from `Config()` (`spec-integration.md` 3.4.2). It is a named integer
type of the root package, in the names table of `WorldCreated` (14.1).

| Error | When |
|-------|------|
| `ErrModuleOff` | the module is off (`spec-integration.md` 11.2) |
| `ErrUnknownAgent`, `ErrAgentRemoved` | the ID names no live agent; a pseudo-agent is `ErrUnknownAgent` (`spec-integration.md` 7.1.1) |
| `ErrUnknownStoreClass` (new) | the class is not in the configuration |
| `ErrStoreLocked` (new) | `AddStore`: the class has a `Needs` that the agent's region lacks (3.2.3) |
| `ErrNotLimited` (new) | `AddStore`: the agent's role has no base room (a store on an unlimited account adds nothing, so it is refused: an input that is accepted and ignored is a trap, `spec-world.md` 12.3) |
| `ErrAgentAway` (new) | `AddStore`: the agent is away on a self-carried trip or on a migration journey. Its account stays in the origin region, so the call would succeed, and at the arrival `has_store` would refuse the relocation and hold it for ever (`spec-world.md` 6.12 C16: any refusal other than the arriving shipment's holds the arrival; P1.3, P1.5 with `skip = on_journey`), leaving the traveller or migrant stranded. The consumer calls again after the arrival; `RemoveStore` is not refused for it (it can only free the arrival) |
| `ErrOutOfBounds` | `AddStore`: the agent would own more than `MaxStoresPerAgent`; `RemoveStore`: the agent owns none of the class |
| `ErrNoRoom` (new) | `RemoveStore`: with the store gone the account's load plus pledge would exceed its room, or its pledge plus its upkeep room (3.4.5) would |
| `ErrStoreInUse` (new; asked by CAP) | `RemoveStore`: the agent's stores of the class would fall below its installed store facilities of the class (`spec-infrastructure.md` 5.4.2) |

3.6.3. **By CAP.** The facility spec (`spec-infrastructure.md` 5.4) calls the same internal
operation, `addStore`, at its P8.3b and only while the owner's account is in the facility's region
(`spec-infrastructure.md` 5.4.2), and `removeStore` when a facility is destroyed or decays.
`removeStore` has a **forced form** for a ruin: it never refuses `ErrNoRoom` but cuts the
room by 5.5.2.1 (whole lots to the estate under `estate_goods`) and 5.5.3 (the pledge). The away test of `ErrAgentAway` (3.6.2) is **in the operation**, so
a facility completed for an agent that is away or on a journey meets the same refusal and
gives it no store (it would be stranded by the C16 route, 9.2); CAP owns what its completion
does with the refusal (it keeps the facility and calls again after the arrival). This spec owns the operation and its effect on room, region
binding and state; CAP owns when it is called and what it costs.

3.6.4. **No store in the base case.** A configuration that names the module but gives
no role a store and no role a base room has no limited account: the module does
nothing and `Validate` warns (`no_limited_role`, 13.2), as the money module warns of a
module with no flows.

## 4. The ledger's part

The ledger is where the invariant lives. It knows no role, store or class: it sees an
account with an optional room and a table of per-commodity sizes it was given at
construction (architecture.md 2.9; `spec-ledger.md` 2.7).

### 4.1 What an account keeps

4.1.1. An Agent account has, beside its money and lots, three pairs and a flag:

```go
// internal/ledger: stdlib only, no root types.
type Room struct {
	Limited  bool
	Mass     int64 // the room, per dimension; meaningful only when Limited
	Bulk     int64
	LoadMass int64 // sum of quantity * Mass over the lots, maintained by every operation
	LoadBulk int64
	PledgeMass, PledgeBulk     int64 // set by the world (3.4.2); not a reservation
	ReservedMass, ReservedBulk int64 // room of the bids admitted this tick (3.4.3)
}
```

An account that is not limited has `Limited` false and none of the rest is read: every
operation of the ledger costs what it did (4.5). The sizes (`Mass`, `Bulk` of each
commodity) are a table of primitives given at construction, like shelf life and
durability.

4.1.2. **Operations added.**

| Operation | Effect | Refused with `ErrNoRoom` when |
|-----------|--------|-------------------------------|
| `SetRoom(account, limited, mass, bulk)` | sets the room (a weight change, a store call, creation) | `Load + Reserved` exceeds the new room. The pledge is not tested: a cut of the room may leave it above what the room can hold (3.4.4, 5.5.3), which is the credit's to cut; a caller that must keep it within the room (`RemoveStore`) tests first (4.2.2) |
| `SetPledge(account, mass, bulk)` | sets the pledge, absolute | never: the value is **not tested**. The callers test before a commitment (P2.7, P3.2, P3.3 against the free room less the upkeep room, 5.1.1; `RemoveStore` before `SetRoom`), as `RemoveStore` does for the room; and the calls that set a value above the room are legitimate: the recompute after a cut (5.5.3), the warm start's job after its test (13.1) and the rebuild at a restore (4.3, 15.3), which sets what the job and the shipments imply whether or not the room was cut. The check of the value is the audit's identity 4 (4.4) |
| `ReserveRoom(account, mass, bulk)` and `ReleaseRoom` | adds to or takes from the reserved room | reserve: it exceeds the free room. Release more than reserved is `ErrReservationExceeds` (`spec-ledger.md` 6.2) |
| `FreeRoom(account)` | the free room pair (3.4.4), 0 where it would be negative, read only; allocates nothing | |
| `RoomLeft(account)` | `Room - Load - Reserved`, per dimension, 0 where negative, read only: the room a credit from the pledge may use (4.1.3) | |

4.1.3. **A credit of goods** (`Move` into the account, `Transform` outputs, a creation)
takes one of three forms, as a debit of money takes the form "from reserved"
(`spec-ledger.md` 4.3):

- **unmarked**: it needs the quantity to fit the free room, else `ErrNoRoom`, and
  changes the load;
- **from reserved**: it draws on the reserved room (the settlement of a bid): the
  reserved room falls by what the load rises by, so the free room does not change;
- **from pledge**: it draws on the pledge (the completion of a job, the arrival of a
  cargo): the pledge falls by what the load rises by. The quantity of such a credit is
  bounded by the pledge **and** by `RoomLeft`. While the invariant holds the pledge is
  the smaller, and the credit is whole. After a cut of the room (5.5) `RoomLeft` may be
  the smaller: the caller then credits the part that fits, whole units, and the rest
  is not credited but disposed (a job's output, `disposal`, 5.3.4) or diverted (a cargo,
  5.4.4), and the pledge falls by the whole of the line either way.

A credit marked from reserved or from pledge that exceeds what it draws on is
`ErrReservationExceeds` (as a money transfer from reserved is, `spec-ledger.md` 6.3).
A debit (a move out, consumption, spoilage, wear, disposal, an estate transfer) lowers
the load and raises the free room, with no condition.

4.1.4. **What the ledger does not hold.** No new account kind, no new reason, no flow
total, no claim and no boundary: room is not a transfer and moves no money or goods
(`spec-ledger.md` 2.1 still holds for every balance and quantity). The pledge is a
counter of the same kind as the reserved room, set by the world and rebuilt after a
restore (15).

### 4.2 The refusal and who handles it

4.2.1. `ErrNoRoom` is a ledger-internal sentinel, mapped at the root to the exported
`ErrNoRoom` of 3.6.2 where a consumer sees it (N-075). It is returned by the
operations above and changes nothing (`spec-ledger.md` 2.6).

4.2.2. **Every place that can meet it tests first.** An input-derived path reads the
free room before it acts and turns a shortfall into its own refusal (a bid into
`no_room`, a job request into `no_room`, a shipment into `consignee_full`, an
`AddAgent` and a `RemoveStore` into `ErrNoRoom`). A completion and an arrival read
`RoomLeft` first and credit what fits (4.1.3), so they never meet it. A path that is
guaranteed room by construction (a settlement from reserved room: section 5) that still
meets `ErrNoRoom` has found a defect in the library, and it is a programmer error: it panics
with a message that says so (architecture.md 8.1, `spec-ledger.md` 12.2), or, with
`WithAudit`, fails the step with `ErrAuditFailed`. The library never loses goods
quietly.

### 4.3 Maintaining the pledge

The world sets the pledge, always as the absolute value derived from the state that
causes it (3.4.2), and sets it at these moments, none of which is a new position
(11.1):

| Moment | Position | New pledge of the account |
|--------|----------|---------------------------|
| the construction of a world, a warm start's job in progress | `NewWorld`, after `spec-world.md` 5.6.1 item 4 | the pending output of the job (load), fixed from its virtual start; no inbound cargo yet. It is **tested** at construction (13.1, `room_below_warm_job`), since no `Transform` of P2.7 sizes it |
| a job starts | P2.7 | the pending output of the job (load) plus the inbound cargo |
| a job completes, is abandoned or its executor is removed | P2.1, P7.1, P7.2, a call | without the job's part |
| a job's executions are scaled by a weight reduction | P7.1(d) | recomputed from the scaled pending output; it may exceed what the room can hold (5.5.3) |
| a shipment is dispatched with the account as consignee | P3.2, P3.3 | plus the load of the lines |
| a shipment arrives or is diverted | P1.3 | minus the load of its lines (delivered, spoiled or diverted alike) |

Because it is derived, it is not saved (15): a restore sets it for every limited
account from the jobs and the shipments in flight, after the load is rebuilt. The set
is not tested (4.1.2): the value is what the job and the shipments imply, and the
commitments that made it were each tested when made, so a pledge above the room after
a cut (18.4: room 700, load (100, 50), pledge (940, 940)) is set, restored and resumed
without a refusal.

### 4.4 The identities the audit adds

With `WithAudit` (`spec-ledger.md` 12.4) and the module on, the audit also checks, for
every limited account, in ID order, and names the first that fails:

1. `Load` equals the sum over its lots of quantity times size, per dimension;
2. `Load + Reserved` is at most the room, per dimension, at the end of every position;
   and `Load + Pledge + Reserved` is at most the room for every account that made a
   commitment in that position (the end of P2.7 and of P3.3: a commitment is tested
   against it). The pledge of an account whose room was cut (5.5) may be above it until
   it is credited, and the audit does not ask for more: a cut leaves no state to tell it
   from a defect, and a defect in a commitment is found at the commit;
3. at a tick boundary, `Reserved` is 0;
4. `Pledge` equals the load of the pending output of its job plus the load of the lines
   of the shipments that name it (so a pledge that nothing explains is found). This is the
   check of the value, since `SetPledge` tests nothing (4.1.2).

### 4.5 Cost

An operation on an account that is not limited costs one branch. On a limited account a
move updates two integers per lot touched, with checked arithmetic, and a credit tests
two comparisons. Ageing and spoilage move whole lots and update the load once per lot
that expires. Nothing is quadratic and nothing allocates (NFR-STO-001).

## 5. Where room acts

### 5.1 The rule: commit where the decision is made

5.1.1. **Decision.** Room is **committed at the point where an agent decides to
acquire goods**, and every later step of that acquisition fits by construction. Nothing
is ever created and then lost because there was no room, and nothing waits for room.

| An acquisition | Decided at | Tested against | Refused as | Commits |
|----------------|-----------|----------------|------------|---------|
| A bid | P3.2(c) | the free room | `no_room` (`OrderRejected`) | reserved room until P4.1 |
| The start of a job | P2.7 (advised at P2.5) | the free room less the upkeep room (3.4.5), with the inputs freed (5.3) | `no_room` (a refusal of `spec-production.md` 4.2) | pledge until completion |
| A shipment to another agent | P3.2(c), P3.3 | the consignee's free room less its upkeep room | `consignee_full` (`ShipmentRejected`) | pledge on the consignee until arrival |
| A self-carried trip to oneself | P3.2(c) | nothing: no goods move, they stay in the account (`spec-transport.md` 5.1) | | nothing |
| The creation of an agent with goods | construction, `AddAgent` | the room it would have | a validation rule; `ErrNoRoom` | nothing (the goods are in the account) |
| A job in progress made at construction (a warm start) | `NewWorld` | the base room plus the given stores (13.1.1), with the stock and the upkeep room (13.1) | a validation error at construction | pledge until completion |

5.1.2. **Why at the decision.** The three costs of a late test are all worse. A
bid that is admitted and cannot settle breaks the guarantee that settlement of a valid
trade cannot fail (`spec-clearing.md` 2.3, 3). A job whose output cannot be created
would have consumed its inputs already (`spec-production.md` 4.3: the start is where
nothing has yet been spent), and `spec-production.md` 5.3.2 refuses to hold a job open.
A cargo that cannot be delivered is diverted to the estate (`spec-transport.md` 5.8):
the shipper has paid for a journey that ends in a loss. Testing at the decision turns
each into a visible, counted refusal (14.1) at the moment nothing has been spent.

5.1.3. **What it costs, stated.** Room is committed early and for the whole life of
the commitment: a job of duration `d` holds its output's room for `d` ticks and a
cargo for its travel time, whether or not the room would have been wanted for anything
else in the meantime. A limited agent therefore holds less than its room most of the
time. The commitment is tight (the pledge is exactly the pending output, not a bound)
and the bid's reserved room is released at P4.1 for what did not trade.

### 5.2 Trades

5.2.1. **Admission.** `spec-clearing.md` 3.2 gains a last row. A bid of an agent whose
account is limited is rejected with `no_room` if its quantity times each size exceeds
the free room in either dimension. The check runs after `insufficient_stock`: money and
stock are decided first, so a bid that fails for two reasons reports the older one
(`spec-clearing.md` 3.2: the first that fails). A bid that is admitted reserves its room
(`ReserveRoom`) with its cash, in `spec-clearing.md` 3.3's step. A pseudo-agent has no
room and the check does not apply to it; neither does it apply to an ask.

5.2.2. **Settlement.** The goods a trade delivers are credited from reserved room
(4.1.3), so settlement of an admitted bid cannot fail for room (FR-STO-009). At the end
of P4.1 the room of every unmatched part is released with the unmatched money.

5.2.3. **Within one agent.** An agent's intents are admitted in plan order, and each
takes free room as it is admitted. An earlier intent therefore has the first claim on
the room (a bias registered as STO-A4, 16). The default strategy builds its plan in
the order of its tiers, so upkeep is served before inputs before buffers (6.3).

5.2.4. **An ask frees nothing the same tick.** The units an ask will sell leave the
account at settlement, after every admission. Their room is not available to a bid of
the same tick. It is conservative and it keeps the admission independent of the keys
that order the settlement (`spec-world.md` 10.1).

5.2.5. **What the market sees.** A bid rejected for room is not in the book and has no
outcome: it moves no belief and appears in no unmatched count. The unmet want is
invisible to the price, which is why a target is scaled to the room (6.2, STO-A1).

### 5.3 Production

5.3.1. **Where the room enters.** `spec-production.md` 4.2 finds `k0`, the executions
the inputs and tools allow with the draws ignored (`spec-world.md` P2.5; the ration of
P2.6 and the live inputs cut it further at P2.7: `k = min(k0, grants, live inputs)`), and
the reason if it is 0. With the module on, the executor's room gives a further bound
`k_room`, evaluated at P2.5 and **part of the `k0` the world derives there**, so that the
request that the ration of P2.6 shares is already the room-bounded one and a grant is not
sized on executions the room will refuse. The view of P2.5 carries `k_room` in
`MaxExecutions` (a strategy may read it), and **the start of P2.7 is the authority**: it
evaluates the bound again against the state at that moment, so that two jobs whose output
goes to one account (a trainer's trainees, `spec-production.md` 4.3.2) are sized in
rotation order, each against the room the earlier left; the second evaluation can only
lower `k`, and a grant above what `k` then uses is not drawn (the draws are whole or not
at all, `spec-resources.md` 5.4). If `k` is 0 and `k0` was not, the refusal is `no_room`,
last in the order of 4.2 there (the order is `spec-production.md` 4.2's; this spec quotes
its end): `no_endowment`, `no_title`, `rent_funds`, the build reasons of
`spec-infrastructure.md` 4.2.2, `no_room`.

5.3.2. **The bound.** Take the recipe's inputs `(q_i, size_i)` and outputs `(q_o,
size_o)` per execution, `E` the effective yield in basis points for this region and
season (`spec-production.md` 6.3), `D` = 10 000, and per dimension `d` of mass and bulk:

```
I_d = sum_i q_i * size_i,d          the load of the inputs of one execution
A_d = sum_o q_o * E * size_o,d      the load of the outputs of one execution, times D
c_d = A_d - D * I_d
R_d = D * (Free_d - U_d) - sum_o size_o,d * r'_o
r'_o = r_o if (q_o * E) mod D <> 0, else 0
```

`o` runs over the recipe's **non-discard** outputs. A discard output (`spec-production.md`
7.2) is created and moved to `disposal` in one `Transform` step at the completion and
never sits in the account, so it takes no room, is not in `A_d` and is not pledged; the
same holds for the part of an output that `MaxQuantity` disposes (5.3.4). `Free_d` is
the principal's free room before the start, `U_d` the principal's upkeep room (3.4.5; the
principal of a trainee's job is its trainer, 5.3.6), and `r_o` is the executor's
residue of output `o` (`spec-production.md` 6.6: `raw = k * q * E + residue`, `produced
= raw / D`), so the exact pending output of `k` executions is `floor((k * q_o * E + r_o)
/ D)` of `o`, which is at most `(k * q_o * E + r_o) / D` as a rational number. An output
whose `q_o * E` is a multiple of `D` carries no remainder that can add a unit
(`produced` is exactly `k * q_o * E / D`, 6.6.1), so its `r'_o` is 0. Taking the rational
bound for every output, the output of `k` executions fits once the inputs are freed if
`k * c_d <= R_d` in both dimensions (FR-STO-012). The inequality is linear in `k`, so the
largest `k` is found without a search:

- if `c_d > 0`: no `k` of 1 or more fits unless `R_d >= 0`, and then `k <= floor(R_d / c_d)`;
- if `c_d <= 0` and `R_d >= 0`: every `k` fits in this dimension;
- if `c_d < 0` and `R_d < 0`: `k >= ceil(-R_d / -c_d)` (the start frees room as it goes:
  an input larger than the output); if `c_d = 0` and `R_d < 0`: none fits.

`k_room` is the largest `k <= k0` that meets every bound, or 0 if the lower bounds
exceed the upper. The bound is **sound**: the exact pending output of `k_room`
executions, with the executor's own residues, fits (the model of 17 checks it over
20 000 random cases, in one dimension; the pledge is the exact load, not the bound). It is
**conservative** by less than the load of one unit of each output in each dimension, and
the case where that shows is a recipe whose output load is within about one unit of its
input load, in a room with up to 5 units free (17: the 6 cases with a gap of 3
executions or more, whose output exceeds the input by 0 to 1.5 units): the bound asks for
the fraction of a unit its residue may add. A pack author who wants such a recipe to run
in a full room gives the room one unit of the output more (`room_below_recipe` is the
warning that names the nearest case).

5.3.3. **The start.** At the start the `Transform` takes the inputs (`recipe_input`)
and the job is created with its pending output fixed from `E` and the remainders
(`spec-production.md` 4.3, 6.4). The world then sets the pledge to the pending output's
load plus the inbound cargo (4.3). The pending output cannot exceed the bound, so the
pledge fits. A partial `k` is a partial start of a cohort, "`k` households work and the
rest could not" (`spec-production.md` 4.2, FR-PRO-009): not an error and not a
warning. It is counted (14.1).

5.3.4. **The completion.** At P2.1 the `Transform` creates the pending output
**from the pledge** (4.1.3) and the pledge falls to what remains. With the room as it was
at the start, nothing is cut and the credit cannot fail. `spec-production.md` 5.3.2
stands for `MaxQuantity`: output that would take a quantity above `MaxQuantity` is still
created and disposed of in the same step. A job's completion never fails and never holds
a job open for lack of room. That settles what N-002's update left to this spec: the
start is where room is decided, and a job's output is lost for lack of room in one case
only, a **cut of the room** while the job runs (5.5.3: the weight fell, and the job's
pledge is not scaled with the room). The completion then credits the outputs in
commodity order, each up to `RoomLeft` (4.1.3), whole units, and creates the rest and
disposes of it in the same step under `disposal`, the way the `MaxQuantity` rule does;
`RecipeExecuted` reports the disposed quantities as it reports that overflow
(FR-PRO-029), and `BoundaryFlow` reports `disposal`. The case is seen there and needs no
counter of its own: `RoomLimited.Discarded` counts the strategy's discard of 6.3.2 only.

5.3.5. **Weight and removal.** A weight reduction scales the pending output
(`spec-production.md` 5.4) and the world recomputes the pledge from it; the pledge may
then exceed the room that is left (5.5.3), and 5.3.4 is where that ends. A removal
abandons the job and its account closes. A role change abandons the job
(`role_changed`), which releases its part of the pledge before 5.5.2 runs.

5.3.6. **Own-use and trainees.** An own-use job (`spec-subsistence.md`) is an ordinary
job and meets the same bound; `no_room` is one more reason for which a household is
**blocked** (`spec-subsistence.md` 5.5). A trainee's job puts its output into its
trainer's account, the principal, so the bound and the pledge are the trainer's. A hired job's principal is the employer (`spec-employment.md` 5.1), so its bound and pledge are the employer's.

5.3.7. **Output that has no room is not an input problem.** A job that is refused for
room consumed nothing. The unmade output is the cost, and it appears as a refusal
(`RoomLimited`, `StartsRefused`) and, by the missing supply, in the market (STO-A2).

### 5.4 Transport

5.4.1. **Dispatch.** `spec-transport.md` 5.4 gains a refusal after
`consignee_elsewhere` (#11): `consignee_full`, when the shipment has a consignee other
than the shipper, the consignee's account is limited, and the load of the lines (sum
of quantity times size, per dimension) exceeds the consignee's free room less its upkeep
room (3.4.5) in either dimension. The consignee's free room is read at the request, after every earlier
request of the tick has pledged its own (so two shippers to one consignee are sized in
dispatch order, 5.4.2). At dispatch (`spec-transport.md` 5.5.1) the world sets the
consignee's pledge up by the load of the lines. Haul units and feed are not cargo:
they are redeemed and consumed at dispatch and take no room.

5.4.2. **Contention.** Two shippers, or a shipper and a contract round, that send to one
consignee in one tick are served in the order of P3.2 (ascending agent ID) and P3.3
(formation order): the earlier takes room first. A formation that fails at dispatch
for `consignee_full` is `ShipmentRejected`, and the contract round counts it in the
column it has for a refusal of transport, `ContractRound.DispatchRefused`
(`spec-merchants.md` 9.5, 15.1; `spec-observability.md` 4.1.6: no new column). The bias belongs to
the consignee's own room, not to a pool, so it picks no winner among agents'
livelihoods (STO-A4).

5.4.3. **A self-carried trip.** A trip to oneself (`ToSelf`) moves no goods: the
holdings stay in the traveller's account, which changes region at arrival
(`spec-transport.md` 5.1), so its load and its room are unchanged and nothing is
tested. A self-carried trip with lines to another agent is a shipment with a consignee
and is tested as in 5.4.1. The traveller's account is limited while away, and it places
no order, so its free room is not used meanwhile (`spec-transport.md` 5.5.2). Whether a
store owner may travel at all is 9.3.

5.4.4. **Arrival.** At P1.3 each line is credited from the consignee's pledge, in line
order, and the pledge falls by the load of the line, whatever part was delivered, spoiled
or diverted. The rule of `spec-transport.md` 5.7 item 2 ("the move would not exceed
`MaxQuantity`") gains one clause, "or the room": the quantity credited is bounded by
`RoomLeft` (4.1.3). It is a **backstop** that the pledge makes unreachable for a consignee
that has kept its room, and it is reachable in one case: a **cut of the room** while the
cargo travelled, a weight reduction of the consignee or a role change to a role with a
smaller base room (5.5). The part that does not fit, whole units, is then diverted to
the estate of the destination under `transit_in`, as for any cargo the consignee cannot
hold (`spec-transport.md` 5.8), and `ShipmentArrived` says what was diverted. A trip
that was dispatched is not undone: the shipper has paid for it (5.1.2), and that is the
price of a room that shrank.

5.4.5. **Information.** The view's `Inbound` is unchanged (`spec-transport.md` 9.1).
The view's free room is already net of the pledge (6.1), so a consignee sizes its bids
against the room the cargo will not need.

### 5.5 Every other way goods reach an account

5.5.1. The list is closed, and each has its guarantee (FR-STO-016):

| Path | Reason | Why it fits |
|------|--------|-------------|
| a settlement | `trade` | reserved room (5.2.2) |
| a job's output | `recipe_output` | the pledge (5.3.4); after a cut, what does not fit is disposed |
| a cargo's arrival | `transit_in` | the pledge (5.4.4); after a cut, the backstop diverts |
| the endowment at construction, a warm start's stock | `initial` | validated before construction (13.1): `room_below_start_stock`; a warm job's pending output is pledged, and tested at construction (`room_below_warm_job`) |
| an external agent's money and goods | `initial` | `AddAgent` tests and returns `ErrNoRoom`, creating nothing |
| a weight reduction, a removal, a diverted cargo | `split`, `merge`, `starvation`, `deaths`, `estate_goods`, `transit_in` | goods go to the estate or are scaled down together with the room |
| a role change | `estate_goods` | 5.5.2 |
| a ruined store facility | `estate_goods` | the rule of 5.5.2.1 (`spec-infrastructure.md` 5.4.3) |
| a relocation | none | no goods move; stores refuse it (9) |
| spoilage, wear, consumption, disposal | sinks | the load only falls |

5.5.2. **A role change.** The base room is the role's. When the spawner or a
traineeship changes an agent's role (`spec-spawner.md` 4.2, `spec-skill-barriers.md`
4.4), after its job is abandoned and the goods outside the new tradable set have gone to
the estate, the world resolves the new room (3.3.1: a role with no base room makes the
account unlimited and ends its stores, 3.6.1a, and there is nothing to compare). For a
limited new role it compares the account's **load** with the new room. If the load
exceeds it in either dimension, goods move to the estate under the existing reason
`estate_goods` by **the rule of 5.5.2.1**, until the load fits. The number of goods listed
counts in `RoleChanged.GoodsListed` (`spec-observability.md` 4.1.4). It is the existing
rule extended by one condition: a role change is never refused for room and the library
never holds a load above its room. The cargo in flight is not moved and not tested here:
if the new room cannot hold it, the arrival diverts it (5.4.4). The cost is a loss of the
agent's goods to the pool, which is what a role change already costs for the goods the
new role cannot use.

5.5.2.1. **What goes first: one rule for 5.5.2 and for the discard of 6.3.2.** Let the
**shortfall** `S_d` be, per dimension, the amount by which the load must fall (5.5.2: the
load above the room; 6.3.2: the load of the tier 1 bids above the free room), `S_d` at
most 0 meaning it is met. While some `S_d` is above 0:

1. the **binding dimension** `d*` is the one whose shortfall is the largest *relative to
   the room*, `S_d / max(Room_d, 1)`, compared by cross-multiplication in the 128-bit
   helper (no division), ties to mass;
2. among the commodities that have units that may go (5.5.2: any held unit; 6.3.2: the
   surplus above `Keep'`) and a size above 0 in `d*`, take the one with the greatest
   load of those units in `d*`, ties by the lower `CommodityID`;
3. move that commodity: 5.5.2 moves **whole lots, oldest first**, until `S_{d*}` is met or
   the commodity has none left; 6.3.2 discards **whole units**, the least that meets
   `S_{d*}`, `ceil(S_{d*} / size)` capped at the surplus;
4. **recompute** every `S_d` (the other dimension may have been met or changed) and the
   binding dimension, and repeat. The ranking is therefore re-evaluated before each
   commodity and never inside one.

It ends: every pass lowers the load in a dimension that has a shortfall, and a shortfall
above 0 has a load behind it (5.5.2: the load is above the room; 6.3.2: 6.3.3). The
model of 17 runs it over 5 000 random role changes and 582 random discards, and 18.3 is
a worked case. A mass count is never compared with a bulk count (`spec-goods.md` 3.2:
the pair, never a single number): both are measured against their own room.

5.5.3. **A weight reduction** scales every holding by the new weight over the old
(`spec-scale.md` 4.8) and the base room by the same: a limited account's **load** never
rises above its room by it (FR-STO-017: `floor(q * w'/w)` summed is at most
`load * w'/w`, and the room falls by the same factor on the base part and not at all on
the stores; the model of 17 runs 20 000 random reductions). A weight rise adds
households and money, not goods, and raises the room.

The **pledge** is not scaled with the room. A job's pending output is scaled by
`min(k, w')/k` (`spec-production.md` 5.4), so a job of `k` below `w'` keeps all of it, and
an inbound cargo is not touched at all (`spec-transport.md` 5.9.4: the trip is
unchanged). A cohort of weight 10 with a base room of 100 per household, a store of 200
and a job of one execution that pledges 900 falls to weight 5: the room is 700, the load
is half of what it was, and the pledge of 900 may be above what is left. That is the cut
of the room, and it is not refused (a weight reduction is the ladder's and the
population's, never a consumer's input) and not repaired by moving goods (a pledge is
not goods): the commitment is cut where it is credited, the output of the job disposed
(5.3.4) and the cargo diverted (5.4.4), in whole units, and every new commitment is
refused meanwhile because the free room is read as 0 (3.4.4). The case of the model of
17 (part 4): of an output of 450 pending and a cargo of 40, the cargo is credited whole
and 280 of the 450 are credited and 170 disposed.

5.5.4. **A restore** rebuilds the load from the lots and the pledge from the jobs and
the shipments, and it is corrupt (`ErrCheckpointCorrupt`) if any limited account's load
plus reserved room exceeds its room (15.3). The pledge is not tested: after a cut it may
legitimately be above the room.

## 6. What an agent sees and does

`spec-agents.md` 8.3 said what a capacity would change: "a full store is a strategy
problem on the next tick, the view gains the free room, and discarding stays a strategy
decision". This section is that, with one correction: room is committed at the decision
(5), so a full account is not a problem the strategy meets after the fact, it is a
number the strategy reads before it plans.

### 6.1 The view

6.1.1. With the module on, the `AgentView` of a limited agent (`spec-agents.md` 6.4.1)
gains a group, plain data, valid for the call:

| Field | Meaning |
|-------|---------|
| `Limited` | whether the agent's account has a room |
| `Room`, `Load`, `Pledge`, `Free` | the pairs (mass, bulk) of 3.4, at the time the view is built; `Free` is `FreeRoom` of 4.1.2: `Room - Load - Pledge`, read as 0 per dimension where it would be negative after a cut (3.4.4), never a negative number (the reserved room is 0 before the first order of a tick) |
| `Stores` | the agent's stores as (class, count), sorted by class |
| per commodity in the tradable set: `Mass`, `Bulk` | the size of one unit (`spec-goods.md` 3.1) |

The view of `Work` carries `Free` as it is after the continuations and completions of
P2.1 and the per-recipe `MaxExecutions` with `k_room` applied (5.3.1), with the reason
`no_room` if it is 0. The view of `Trade` is built immediately before the agent's plan
(P3.2), so its pledge includes the cargo `Inbound` fixed at P3.1.

6.1.2. **No other agent's room is visible.** The view carries the agent's own room and
nobody else's, as it carries its own region's endowments only (`spec-resources.md`
5.5.1). A consumer's strategy that wants to know a consignee's room asks the shipment
mechanism, which refuses (`consignee_full`).

### 6.2 What a store changes in the target

6.2.1. `Target_c` is the number of units the belief rule is told the agent wants to
hold after this tick's upkeep (`spec-agents.md` 7.3.1). If it is above what the room
can hold, the belief rule reads a stock ratio that can never be met and raises the
centre of the price belief tick after tick (`spec-beliefs.md` 5.5), which pushes the
market price up with a want that cannot become a purchase: a bid rejected for room is
not in the book (5.2.5), so nothing tells the price the want is empty. This is
STO-A1 (16). The target is therefore **scaled to the room**, and `Keep` with it.

6.2.2. **The rule.** With the module on and the agent limited, after `UseTgt_c` and
`OutTgt_c` are computed (`spec-agents.md` 7.3) and before the tiers, per dimension `d`:

```
ResLoad_d  = sum_c Reserve_c * size_c,d              upkeep eaten at the end of this tick
Avail_d    = max(0, Room_d - Pledge_d - ResLoad_d)   what the targets may take
TL_d       = sum_c Target_c * size_c,d               the load of the targets, wants not yet in
f          = min over d of floor(Avail_d * 10 000 / TL_d), taken only where TL_d > Avail_d;
             f = 10 000 if no dimension binds (so f = 0 where Avail_d is 0)
Target'_c  = floor(Target_c * f / 10 000)         the number the belief rule receives
UseTgt'_c  = floor(UseTgt_c * f / 10 000)
Keep'_c    = Reserve_c + UseTgt'_c + Want_c        what the agent tries to hold now
```

The scaling is **pro rata** over every commodity of the target, the job's inputs
included: a room that cannot hold the targets holds a share of each, in proportion, and
no commodity is chosen to go first. A target's share of the room is the strategy's, not
a rule of the library (a consumer's strategy ignores this and meets `no_room`), and the
default's proportional share is the simplest that keeps every want represented. The
wants of the demand module (`Want_c`) are funded last (`spec-demand.md` 4.5), and are
bounded by the free room that tiers 1 to 3 left, in their ladder order (6.3.1), so
they are not in `TL_d`.

6.2.3. **What it needs.** `Room_d - Pledge_d - ResLoad_d` is at least 0 whenever every
commitment was made with the upkeep room left free (3.4.5): `ResLoad_d` is `U_d`, a job
start and a dispatch are tested against `Free_d - U_d`, and `Free_d` already has the
load, so after a commitment `Pledge_d + U_d` is at most `Room_d - Load_d`, which is at
most `Room_d` (the model of 17, part 3, asserts `Pledge + U <= Room` after every
commitment of a 4 000-tick sequence). Validation requires that the base room hold one
tick's upkeep of a household (`room_below_upkeep`, an error, 13.1), so that the
inequality is possible. It is false in one case, a **cut of the room** (5.5.3), and
`Avail_d` is then clamped to 0 as written above: `f` is 0, the agent targets nothing
beyond what it eats and buys only what it needs this tick, until the pledge is
credited.

6.2.4. **What it does not do.** A target is a belief input and a purchasing want, never
a ceiling on stock: an agent may hold more than its target (a harvest, a cargo it asked
for) as long as it fits the room. The room is the hard limit; the target is scaled so
that wanting and holding agree.

### 6.3 Bids sized to the free room, and discards

6.3.1. **Sizing.** `spec-agents.md` 7.5.4 bounds each tier's quantity by its want, by
the money in the tier and (tiers 3 and 4) by the cap. With the module on, the quantities
are also bounded by the free room, which is allocated in the order of the money
stack: tier 1 for every commodity in canonical order, then tier 2, then 3, then 4 in its
ladder order. A bid's quantity is cut so that its load fits the free room that remains
after the quantities already allocated; a bid that is cut to 0 is not placed, and the
draw of its limit is taken all the same (`spec-agents.md` 7.5.3, `spec-world.md` 7.4.3:
the number of draws never depends on an outcome). The default therefore places no bid
that the market would reject for room; the rejection is the backstop for a consumer's
strategy.

6.3.2. **The discard that makes room to eat.** A limited household that holds goods it
does not need, in a room those goods fill, would be unable to buy its own upkeep (a
deadlock STO-A3 records). The default `Trade` therefore discards, **before** it forms the
asks, the units of surplus (`Surplus_c`, 7.3.3 there, above `Keep'_c`) that its tier 1
bids need room for, by the rule of 5.5.2.1 (the shortfall is the tier 1 load above the
free room; the commodity of the greatest surplus load in the binding dimension first,
relative to the room, ties by the lower `CommodityID`, whole units, re-evaluated before
each commodity), the minimum that lets the tier 1 quantities fit. The discard is a `Discard` of
the plan (`spec-agents.md` 6.5.2) executed first as a move to `disposal`, so it is counted
in `BoundaryFlow` (`spec-agents.md` 8.2). The surplus that is left is offered as before.
It discards nothing for tier 2 or beyond: a producer whose room is full of its own
unsold output does not throw it away to buy inputs; its production stops (5.3) and its
price signal does the rest (STO-A2).

6.3.3. **Why this can always be satisfied.** After the scaling of 6.2, the load of
`Keep'` fits `Room - Pledge` (the scaling takes `Avail` after `ResLoad`, and 6.2.3 shows
`Avail` is not clamped unless the room was cut); so with every unit above `Keep'`
discarded, the account's load is at most the load of `Keep'` and the free room is at
least what the tier 1 bids add (they raise a commodity to its `Reserve`, which is part of
`Keep'`). After a cut the clamp gives `f` = 0 and `Keep'` = `Reserve`, whose load is
within `Room - Pledge` only if the cut left that much; if it did not, the household
buys what the free room allows and may go unfed until the pledge is credited, which is
the cost of a cut (STO-A3, STO-A7).

### 6.4 A consumer's strategy

6.4.1. A strategy may ignore the view, the target and the discard. The world checks, as
it checks everything (`spec-agents.md` 6.5.2): a bid beyond the free room is rejected
`no_room` whole, a job request beyond room is refused with the reason `no_room`, a
shipment beyond a consignee's room is `consignee_full`; none is clamped. A directive's
intents (`spec-integration.md` 4.2) are bids like any other, and the player learns
by an `OrderRejected` with the reason `no_room`, or by reading the free room in the
query of the agent (`AgentState`, `spec-integration.md` 4.7.2, section 19).

6.4.2. **What the default does not do.** The default strategy holds what it uses: its
targets are cover periods of its own use and output (`spec-agents.md` 7.3). It does
not buy goods to hold for a price change, so a store gives the default no reason to
hoard, and a buffer store in a default agent's hand is room it does not fill. The keeper of one
is a strategy of its own (a consumer's, a directive, or a later spec's). That is
deliberate: version 1 makes the limit and the cost real and leaves the use of a large
room to the agent that chooses to use it.

## 7. The cost of a store

### 7.1 A fixed levy per store

7.1.1. The cost of owning a store is a **named money flow** of `spec-money-supply.md`:
a `MoneyFlow` of kind `levy`, shape fixed (`Amount`, `Per`, `Exempt`), whose **target
class is stores**. `spec-money-supply.md` 10.3 reserves exactly this path ("a spec that
wants a configured, scheduled faucet or sink may reuse `MoneyFlow` and its step and add a
target kind"), and N-107 asks every later flow to start there: a declared reason equal
to the flow ID, closed for a run, a boundary account created at construction, reported
in `BoundaryFlow`, checked by the audit, no faucet that reads an aggregate.

7.1.2. **The target kind.** `FlowTarget` (`spec-money-supply.md` 4.1) gains a kind
`"stores"` and a field `Store`:

```go
type FlowTarget struct {
	Kind   string // "households", "pool" or "stores"
	Region string // "" means every region
	Role   string // households only
	Store  string // stores only: a store class; "" means every class
}
```

The class of accounts is every open Agent account whose agent owns at least one store
of the class (all classes if `Store` is empty) in the region (every region if `Region`
is empty). **An amount is per store**: the account of an owner of `n` matching stores is
due `n` times the amount, not `w` times it, because a store is not a household
(`spec-money-supply.md` 4.3.1 differs here and says so in the edit). Membership is read
at the position of the flow, so a store added or removed between ticks counts from the
next P1.7.

7.1.3. **Shapes.** Only the fixed shapes are defined for stores: a **fixed levy** (the
owner pays `n * Amount * S / (P * 10^4)` with the carry of `spec-money-supply.md` 6.1, capped
at what it holds above `Exempt`, which for a stores target is an amount per account, not
per household) and a **rate grant** (a subsidy per store, the same due). `Level` and `RateBP` for a stores target are `ErrFlowShape`: a top-up and a rate
of a balance are not per store. The target kind needs the module (`ErrModuleDependency`,
`spec-money-supply.md` 12.1) and a class that exists (`ErrUnknownStoreClass`).

7.1.4. **Where, when, how often.** The flow runs at P1.7 with every other flow, after
relief and before production (`spec-world.md` 6.3), so a levy sees the balance the
owner starts the tick with and a store's upkeep is paid before the agent buys. It has
no new position (11).

### 7.2 What it is, and what it is not

7.2.1. **A sink, so no faucet.** The levy moves money from the owner to the flow's
boundary account under the flow's reason. The money supply falls by it, and it is a
sink in `BoundaryFlow` by reason (`spec-observability.md` 4.4.1). A pack that wants the
money to stay in the economy pairs it with a grant of its own (to a pool or to a class;
`spec-money-supply.md` 9.2) or leaves the sink as a deflation pressure that its money
settings answer. This spec adds **no** built-in reason and no faucet (N-021, N-107).

7.2.2. **A cost of capacity, not of use.** The levy is a function of the stores an agent
owns and nothing else: it does not read the load or what is held, and no rule reads an
aggregate (`spec-money-supply.md` 10, N-107). It makes a store a **fixed cost of owning room**, so an agent
with more room than it needs pays for it, and it has no effect on how much a store
holds (the room does that, 5). It is the cost of the capacity, and the number of stores
an agent keeps is the decision it bears.

7.2.3. **Nobody is foreclosed.** An owner that cannot pay is forgiven the unpaid part
and keeps the store (`spec-money-supply.md` 5.4: "short"), and the shortfall is counted in
`MoneyFlowLimited`. A store that is abandoned for want of money, or a facility that
decays for want of maintenance, is the credit and infrastructure specs' to say (open
question 4).

7.2.4. **The lever.** The flow scale (`spec-money-supply.md` 4.4.3; `SetFlowScales`) is
the one in-run lever on the cost of storage: a ruler's duty on stores, a relief during a
famine, a scripted schedule. The engine contains no rule that changes it in response to
a measure (ADR 0001 decision 7).

### 7.3 An example of the data

An invented setting (flagged: the words are data). A store class `bin` of mass 400 and
bulk 300, a levy of 12 minor units a tick per `bin`, and an optional grant to the
pool of the same size that returns the money to the economy:

```toml
[modules.storage.class.bin]
mass = 400
bulk = 300

[modules.money_supply.flow.bin_upkeep]
kind = "levy"
amount = 12
per = "tick"
target = { kind = "stores", store = "bin" }

[modules.money_supply.flow.bin_dues]      # an optional pair, the pack's choice
kind = "grant"
amount = 12
per = "tick"
target = { kind = "pool" }
```

The two have equal fixed amounts only when the number of stores equals the number of
pools, so the pair is not balanced in general (`spec-money-supply.md` 9.2 warns of
`pair_unbalanced` for fixed pairs); the example shows the form, not a recommendation.

### 7.4 What is not modelled: a price per unit of space

7.4.1. A **holding charge** per unit held (a cost that rises with what is stored) would
be the textbook shape, and it would make the marginal unit costly, which a fixed levy
does not. It is not in version 1, for three reasons. Its base is a stock of goods and
not a balance of money, so it is not one of the shapes of `spec-money-supply.md` 4.2 and
would need a new one; the chargeable size of `spec-goods.md` 3.3 that it would use is
known to over-charge a well-mixed load (3.5 there), and the pair is already the exact
test of the room, so a second, dearer measure of the same load would disagree with it at
the margin; and the default strategy does not read a cost of carry, so a marginal cost
would reach a decision only through a consumer's strategy. Open question 3 records the
shape it would take.

7.4.2. The effect of the **absence**: an agent with room pays the same whether it holds
one unit or the whole room. The incentive to leave room empty is the opportunity of a
bid (the free room is what a bid and a job are tested against), not a price.

## 8. Spoilage, wear, loss and the chargeable size

### 8.1 Spoilage: the clock is not slowed

8.1.1. **Decision.** Goods in a store age exactly as goods in any account: a lot made
at tick `t` with shelf life `S` spoils at the end of tick `t + S - 1`, from the account
that holds it, as the sink `spoilage` (`spec-ledger.md` 5.3, `spec-goods.md` 4.4:
"nothing pauses a clock"). A store adds room and nothing else; no store in version 1
cools, seals or otherwise slows the ageing of what it holds.

8.1.2. **Why.** Preservation needs the ledger to tell the goods in a store from the
goods in the agent's hand. With room on the account (3.1) they are one set of lots with
one clock. Telling them apart needs either a second kind of lot (a new dimension of the
lot key, `spec-ledger.md` 5.1.3: the canonical form and every checkpoint would change)
or a separate account for the store, which is the other design (position 1 of
section 2 chose against it) and is recorded below.

8.1.3. **The rule that would model it** (open question 1), for the record, so that a
pack that needs it later has a design to start from. A store class would carry a
`ShelfBP` of 10 000 or more (a shelf-life multiplier) and goods in it would be an
account of their own (kind Store, one per owner and class). A move into it rebases the
lot's `Born` so that its remaining life in the store's own clock is the same fraction of
a shelf life, and a move out rebases back, rounding down, so that a lot never gains
life by a round trip. It costs a new account kind, two reasons (`store_in`,
`store_out`) and a strategy that decides when to move goods in and out. That is a
different design from this one and it would replace section 3, not extend it.

### 8.2 Wear

A durable wears **by use** (`spec-goods.md` 5.3, `spec-production.md` 4.4). A tool in a
store is a tool in the agent's account and wears when a recipe uses it, and does not
wear otherwise. A store that decayed its contents would be a loss and is RSK's.

### 8.3 Loss in storage is reserved for RSK

8.3.1. The reason `loss` exists in the registry (`spec-ledger.md` 7.2: transit, agent or
estate to boundary) and no flow is posted under it in version 1. This spec reserves
the place and RSK specifies it (`spec-risk.md` 6): a theft, a fire or a flood of stored goods is a
transfer from the owner's account under `loss`, drawn from the world's source at the position
P8.1a (`spec-world.md` 6.10, 7.3). The strike takes the struck store's **share** of the account's
goods (`spec-risk.md` 6.1.2), not a chosen set of lots.

8.3.2. **What RSK may read.** For a limited agent: its stores by class and the region
(the facility's site); its room and its load; and the **stored load** per dimension, the
part of the load that sits above the base room,

```
Stored_d(a) = min( sum_i Cap_d(class(s_i)),  max(0, Load_d(a) - w * Base_d) )
```

a derived read with no attribution to commodities: the goods RSK takes are a share of the
account's goods by `spec-risk.md` 6.1.2 (a good has no location in an account, 8.1.2), not a chosen
set of lots: a rule over the lots in canonical order, which this spec's first text had, is
superseded. Nothing in this
spec says goods in stores are safer or riskier than goods in hand; that is RSK's data,
keyed by store class (8.3.3).

8.3.3. A store class is the key a risk row would name in data. RSK adds a field to the
risk table, not to `StoreClass`, so that this spec's data does not change when RSK
arrives (N-104 mechanism, one row per key).

### 8.4 The chargeable size is not used (N-011)

8.4.1. `spec-goods.md` 3.3 and 3.4 define the **chargeable size** and the
**chargeable capacity** "for pricing transport and storage" and leave their use for
storage open (open question 1 there; N-011). This spec does not use either. The room is
a pair tested exactly (3.1.1), the cost is per store (7), and nothing is priced per unit
of space, so the over-charge of a mixed load (`spec-goods.md` 3.5) does not arise in
storage as it does not in transport (`spec-transport.md` 3.2.1). `BulkFactor` stays as
a world parameter with one user fewer than `spec-goods.md` 8 expected; section 19 lists
the edit to the wording. A per-unit price of space (7.4) would use both.

8.4.2. **What the measurement N-011 asked for would be.** The note asks how much the
over-charge is on the reference chain, to decide a per-mode factor. For storage the
question does not arise in version 1. It arises with open question 3, and the
measurement is then the pair against the chargeable load over a run's load series.

## 9. Region binding

9.1. **A store is sited.** A store stands in the region where its owner's account is
when it is given (3.6) and nowhere else: it is a facility, and a facility does not move
with an account. The room of an account therefore depends on the account staying where
its stores are.

9.2. **The refusal.** With the module on, the world registers the `region_bound`
refusal **`has_store`** (`spec-regions.md` 6.5.2, N-090), open for an account whose
agent owns at least one store. It reports no instance keys (one obligation, like
`in_traineeship`). A guarded relocation returns `ErrRegionBound` naming it and changes
nothing (`spec-regions.md` 6.1.3, #7). The list of region-bound refusals of
`spec-regions.md` 6.5.2 gains a row (section 19).

9.3. **What it forbids, and who meets it.**

| Caller | What happens |
|--------|--------------|
| MIG's guard at admission | the household is not moved: the consideration ends with `bound` (`spec-migration.md` 7.2, 7.6), counted in `MigrationReviewed` |
| INT's `Relocate` of an external agent | `ErrRegionBound` naming `has_store`; the consumer calls `RemoveStore` for each store, then `Relocate`, both between ticks |
| TRN, a self-carried trip | the **dispatch is refused** with a new reason `has_store` (`spec-transport.md` 5.4, after `traveller_away`): a trip ends in an account that changes region at arrival, and the arrival would be held for ever by this refusal (`spec-world.md` 6.12 C16: any other refusal holds the arrival). A hired shipment moves goods, not the owner, and is unaffected |
| `AddStore` on an agent already away or on a journey | `ErrAgentAway` (3.6.2): the same arrival would be held for ever, from the other side. The refusal is in the operation `addStore` itself, which the public call and CAP's completion both use (3.6.3), so no arrival is ever held by `has_store` |
| MER's merchant | a merchant that owns a store must not plan a self-carried trip: the planner's choice between travelling and hiring (`spec-merchants.md` 6.1, position 8) drops the travelling option while the view's `Stores` is non-empty (an edit owed to MER, section 19). Without that edit the dispatch refusal is the backstop and is counted, each tick it happens, in `ShipmentRejected` (`has_store`) and `ContractRound.DispatchRefused` for an offer; it is not an error and it changes nothing. A merchant with a store can hire carriage and be a consignee |

9.4. **Why a trip is refused and not allowed to abandon the store.** The alternative is a
store that is lost when its owner moves, which would make a long trip destroy room and
leave the account over its room, or a store that stays and a room that shrinks while the
owner is away, which breaks the invariant of 4. A store that ties its owner to its site
is the plain reading of a facility, and the cost is stated: **an owner of a store does
not travel or migrate**. A role whose members must travel (a merchant) and must store
holds its goods at its base room and hires carriage, or removes the store before it
goes.

9.5. **N-120 applied.** A relocation carries the whole account for free (note N-120,
`spec-migration.md` 9.4). With the module on and the agent's role limited, a migrant or
a forced move carries at most its room, and an agent that owns a store cannot move at
all, so what it carries is at most `w` times the base room, per dimension. The leak the
note describes is closed for a limited role. For an unlimited role it stays as it was; a
pack that wants a carry limit gives the role a base room.

## 10. Stores in use: a buffer store and the seams

### 10.1 A buffer store

10.1.1. The first consumer needs a **buffer store**: a place where a ruler keeps food
against a season of shortage. In this spec a buffer store is
a store class of large capacity owned by one agent, and the use of it is the agent's
strategy. In the first consumer's game
the owner is the **player's agent**, an external agent with a directed strategy
(`spec-agents.md` 10, `spec-integration.md` 4):

1. the consumer gives the player's role a base room and the player a store of the class
   (`AddStore`), or the role lists it (`RoleRoom.Stores`);
2. in a season of plenty the player's directive buys the staple, with intents sized to
   its free room (the query of the agent returns `Free` in its `AgentState`, `spec-integration.md` 4.7.2);
   the bids are admitted only up to the room, and the cost is the levy of 7;
3. in a lean season it sells, or, with FIS and PLC, releases (10.2);
4. the effect is read, as every effect, from indicators (`spec-indicators.md`: price
   index, `StockCover`), never from a state the engine sets.

10.1.2. **What this gives, and what it does not.** It gives the player a limit that is
real (it cannot hoard past its room), a cost that is real (a levy it pays whether or not
it fills the room), and goods that spoil on the clock of their shelf life (8.1). It does
not give a buyer of last resort, a release rule or a price floor: those are policy and
belong to PLC and FIS. A buffer store of version 1 is a strategy's tool, not an institution.

10.1.3. **A worked case** is 18.2.

### 10.2 What is left to the later specs

| Later spec | What it adds | What this spec supplies |
|------------|--------------|-------------------------|
| CAP (`spec-infrastructure.md` 5.4) | builds, maintains and decays stores as facilities; who pays | the operation `addStore` and `removeStore` and its forced form (3.6.3); `ErrStoreInUse`; the cost flow (7); region binding (9) |
| RSK (`spec-risk.md`) | loss of stored goods, by store class | the stored load and the stores by class (8.3), used by `spec-risk.md` 6; the reason `loss` |
| INS (`spec-insurance.md`) | cover of stored goods and of routes | the owner's held value and its stores by class (8.3.2), read only; no change |
| SET (`spec-settlements.md`) | which size class may have which store | the field `Needs`, the gate of 3.2.3 and 10.3, `ErrStoreLocked` |
| FIS and PLC (`spec-fiscal.md`, `spec-policy.md`) | a treasury's buffer store and its release rules; a duty on storage | the flow scale (7.2.4). A treasury is unlimited and owns no store (3.3.3, 3.5.1, `spec-fiscal.md` 9.6); a policy actor has no agent (`spec-policy.md` 3.2.2) |
| LND, ORG (`spec-land.md`, `spec-institutions.md`) | leases, who may keep a store for others, a charter | the one-owner rule (3.5) and the seam of 10.4 |
| EMP (`spec-employment.md`) | a workshop that needs its own room | the room of any limited agent; no change |

10.3. **The gate.** SET unlocks a store class by a settlement's size through the field
`StoreClass.Needs` and the predicate `regionHas(region, Needs)` (3.2.3;
`spec-settlements.md` 6.3 H5, 6.5). The refusal `ErrStoreLocked` is this spec's. A cap on
the stores of one class in a region (a SET limit) is not offered in version 1 (open
question 13). With `Needs` empty, which is the default, no gate exists and every class
is allowed everywhere.

10.4. **A keeper and a lease.** A store that holds goods for others needs three
things this spec does not have: a room that several agents use (3.5.2), a price per
unit and tick, and a deposit that a keeper can fail to honour. The design to start from
is the delivery contract: an offer of room by a keeper and a take by a holder matched
by a first-fit round (ADR 0005), the price escrowed per tick, the room pledged to the
holder as a cargo's room is pledged to its consignee (4.3), so the invariant of 4
holds for both accounts without a change to the ledger. It is not written here because
a first consumer has no keeper and a library that guesses a contract form is a library
that has to keep it (open question 2). A right over keeping goods is ORG's and reserved (`spec-institutions.md` 5.8.2); a right over a recipe that requires a facility works without CAP.

## 11. Position in the tick and draws

11.1. **No new position.** The module acts inside positions of `spec-world.md` 6 that
exist; nothing is renumbered, and a module that is off has none of these acts (6.1.5
there).

| Act | Position | Where it is specified |
|-----|----------|-----------------------|
| the levy of stores | P1.7 | 7.1 |
| the arrival of a cargo is credited from the pledge, as far as the room goes; the pledge falls | P1.3 | 5.4.4 |
| a job completes: the output is credited from the pledge, as far as the room goes | P2.1 | 5.3.4 |
| the room is read for `Work`: `k_room` is part of `k0` (the view's `MaxExecutions`) | P2.5 | 5.3.1 |
| the start of a job: `k` sized with the upkeep room left free, the pledge set | P2.7 | 5.3.2, 5.3.3 |
| `Inbound` fixed; the pledge of cargo already in flight is the same state | P3.1 | 4.3 |
| a bid is admitted and reserves room; a shipment is dispatched and pledges | P3.2(c), P3.3 | 5.2, 5.4 |
| `RoomLimited`, once per region and role, after `WantShortfall` | **P3.6a** | 14.1 |
| the bids' room is released with their money | P4.1 | 5.2.2 |
| a role change: the excess goes to the estate; to a role with no base room the stores end | P7.1(a), (d) | 5.5.2, 3.6.1a |
| a store facility is installed or ruined: `addStore`, forced `removeStore` | P8.3b | 3.6.3 |
| `AddStore`, `RemoveStore`: the room changes | between ticks | 3.6.2 |

11.2. **No draw.** Nothing in this spec takes a random number: the sizing, the scaling,
the discard order and the levy are functions of the state, the configuration and the
tick. The draw table of `spec-world.md` 7.3 is unchanged and the module adds no row. The
**count** of draws is unchanged by the module while no scaling, cut or discard acts. It is
not unchanged once one does: the default strategy draws one word per intent it considers
(a commodity with a gap or a surplus, `spec-agents.md` 7.5.1), a scaled `Keep'` or a
discard changes the stock and so can give a commodity a surplus or take its gap away,
and the intents then differ. A paired comparison of a run with and without the module
(`spec-world.md` 7.6; `spec-experiments.md` 7) is aligned until the first limit binds and
is a different stream from then on; EXP reads that as it reads any change of the number
of draws.

11.3. **The order of the events.** `RoomLimited` follows `WantShortfall` in phase 3
(`spec-world.md` 6.11 item 4); `StoreChanged` is the event of a call between ticks and
comes with the other slot events, before the next `TickStarted` (6.11). A `StoreChanged`
with `Cause` `role_changed` (3.6.1a) comes in P7.1 after the `RoleChanged` of the same
agent, one per store in class order; a creation's `AgentSpawned` carries its own
`StoresLocked` (14.1).

## 12. Parameters

### 12.1 The module and its types

`Config.Modules.Storage` is a pointer; `nil` is off (`spec-config.md` 3.2). Plain data,
sorted by ID or role, no maps:

```go
// Storage is the module. Its presence switches it on.
type Storage struct {
	Base    *Capacity    // base room per household for every role without a row; nil = unlimited
	Roles   []RoleRoom   // sorted by Role; a row replaces Base for its role
	Classes []StoreClass // sorted by ID (3.2)
}

// Capacity is a pair of room (spec-goods.md 3.2).
type Capacity struct {
	Mass int64 // mass room
	Bulk int64 // bulk room
}

// RoleRoom is the storage row of one role, in the replace mode of spec-regions.md 5.1.
type RoleRoom struct {
	Role   string    // a role ID
	Base   *Capacity // the role's base room per household; nil = the role is unlimited
	Stores []string  // store classes each agent of the role is created with, in class order; a class may repeat
}
```

12.1.1. **Replace, and an empty row means none.** A row replaces the module's `Base`
for its role as a whole: a row with no `Base` makes the role unlimited even though the
module has a default, and a row with an empty `Stores` gives no store. A row that lists
stores therefore states its `Base` too, or its role is unlimited and the stores are
`ErrNotLimited`. One row per role
(`ErrDuplicateID`), in the one override mechanism of `spec-regions.md` 5.1 without a
region key: **the module has no region-keyed table**, and a regional difference in room
is a regional difference of a store given by a call or of a role's mix (N-104).

12.1.2. **Presence.** The module is on when `Modules.Storage` is non-nil. A non-nil
module with no limited role is legal, does nothing and warns (`no_limited_role`).

### 12.2 The tunables and their defaults

All are fields of `Config.Modules.Storage`, with an owner (this spec), a bound and a doc
line (architecture.md 9.5, 9.7). **Every number is a placeholder** (N-015, N-027).

| Scope | Field | Meaning | Default | Bound (`bounds.go`) |
|-------|-------|---------|---------|---------------------|
| Storage | `Base` | base room per household for every role without a row (3.3.1) | `nil`: unlimited | `Mass`, `Bulk` each 0 to `MaxRoom` |
| Storage | `Roles` | the role rows | none | at most one per role |
| RoleRoom | `Role` | the role | required | an existing role |
| RoleRoom | `Base` | the base room of this role per household | `nil`: unlimited | as `Base` |
| RoleRoom | `Stores` | store classes given to each agent of the role at creation | none | at most `MaxStoresPerAgent`, existing classes; not on an unlimited role |
| Storage | `Classes` | the store classes | none | at most `MaxStoreClasses` |
| StoreClass | `ID` | the class | required | the ID pattern of `spec-config.md` 3.3 |
| StoreClass | `Mass` | mass room one store adds | required | 0 to `MaxRoom`; not both 0 |
| StoreClass | `Bulk` | bulk room one store adds | required | 0 to `MaxRoom`; not both 0 |
| StoreClass | `Needs` | a flag of `spec-settlements.md` the region's class must carry for the store to be given (3.2.3) | empty: none | an existing flag; empty unless `Modules.Settlements` is present |
| StoreClass | `Label` | display name, presentation only | empty | at most 64 bytes; not hashed (`spec-config.md` 8.3) |

**Bounds** (validation bounds in `bounds.go`, not tunables; `spec-config.md` 4.1 gains
rows): `MaxRoom` **10^12** per dimension, so that `MaxWeight * MaxRoom` is 10^18 and the
largest room of an account (a base room of a cohort and `MaxStoresPerAgent` stores) is
below `int64`; `MaxStoresPerAgent` **8**; `MaxStoreClasses` **64**. A load is a sum
of a quantity (at most `MaxQuantity`) times a size (at most 10^6, `spec-goods.md` 8),
computed through the 128-bit helper and refused where it would exceed the room, so it
never overflows (architecture.md 3.5).

### 12.3 The cost flow's fields

The cost is a `MoneyFlow` (7). It adds one field to a type of another spec:
`FlowTarget.Store` (`spec-money-supply.md` 4.1), owner this spec, default empty (every
class), meaning only for the kind `stores`.

### 12.4 Rows for the parameter-owner table

Every row has a field, a default and a validation test (testing.md 6.7). Section 19
lists them for `specs/README.md`:

| Parameter | Scope | Owner |
|-----------|-------|-------|
| `Storage` | Modules (module; nil = off) | `spec-storage.md` section 12.1 |
| `Base` | Storage; RoleRoom | `spec-storage.md` section 12.1 |
| `Roles` | Storage | `spec-storage.md` section 12.1 |
| `Role` | RoleRoom | `spec-storage.md` section 12.1 |
| `Stores` | RoleRoom | `spec-storage.md` section 12.1 |
| `Classes` | Storage | `spec-storage.md` section 12.1 |
| `ID`, `Mass`, `Bulk`, `Needs`, `Label` | StoreClass | `spec-storage.md` sections 3.2, 12.1 |
| `Mass`, `Bulk` | Capacity | `spec-storage.md` section 12.1 |
| `Store` | FlowTarget | `spec-storage.md` section 7.1.2 (the type is `spec-money-supply.md` 4.1) |

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the
sufficient bound of 5.3.2 and its denominator 10 000, the scaling of 6.2.2, the discard
order of 6.3.2 and of 5.5.2, the names `no_room`, `consignee_full` and `has_store`, the
rule that an amount of a stores flow is per store (7.1.2), the order of the checks
of 5.2.1 and 5.4.1.

## 13. Validation

### 13.1 Errors

Each is a `ValidationError` with a `Path` and a `Rule` (`spec-config.md` 7.1.1), and each
has a failing-case test (testing.md 3.9). The sentinels are new unless noted; the root
maps the ledger's `ErrNoRoom` (N-075).

| `Rule` | Sentinel | Condition |
|--------|----------|-----------|
| `duplicate_id` | `ErrDuplicateID` | a class ID or a role row twice |
| `unknown_role` | `ErrUnknownRole` | a row names a role that is not configured |
| `store_class_shape` | `ErrStoreShape` | a class with both capacities 0, or a capacity outside `0 .. MaxRoom` |
| `room_bound` | `ErrOutOfBounds` | a `Base` outside `0 .. MaxRoom`; more than `MaxStoreClasses` classes; more than `MaxStoresPerAgent` stores in a row |
| `unknown_store_class` | `ErrUnknownStoreClass` | a row's `Stores`, or a stores target of a flow, names a class that is not configured |
| `needs_flag` | `ErrUnknownSettlementFlag` (`spec-settlements.md` 9.1) | a class's `Needs` names a flag that no settlement class carries |
| `needs_module_off` | `ErrModuleDependency` | a class's `Needs` is non-empty and `Modules.Settlements` is absent (`spec-settlements.md` R5) |
| `store_on_unlimited_role` | `ErrNotLimited` | a row lists stores and its role's resolved base room is `nil` |
| `room_below_upkeep` | `ErrRoomShape` (new) | a limited role's base room per household is below the load of one tick's effective upkeep (`spec-agents.md` 5.2) at the largest upkeep of any region: the household could not eat |
| `room_below_start_stock` | `ErrRoomShape` | a limited role's `StartStock` per household, or a warm-start group's stock (`spec-viability.md` 8.1), has a load above the base room plus the **given** stores of the row (13.1.1), taken for the largest weight of any group of the role (a store is not multiplied by the weight, 3.3.2) |
| `room_below_warm_job` | `ErrRoomShape` | a warm-start group (`spec-viability.md` 8.1) with a job in progress (`InProgressBP` above 0, 8.5) whose stock load plus the load of the job's pending output, at the effective yield of the first tick (`spec-world.md` 5.6.1 item 4: the season of that tick, the shock at its default, a residue of 0), is above the base room plus the given stores of its row (13.1.1), per household and for the largest weight of any group of the role, or whose pending output load plus the upkeep room (3.4.5) is. Checked at **construction**, once the warm start is applied, not at load of the file: the yield and the selection of agents are fixed there (it is still a configuration error and `NewWorld` returns it, architecture.md 8.5) |
| `stores_target_off` | `ErrModuleDependency` (`spec-money-supply.md` 12.1) | a flow targets stores and the module is off |

13.1.1. **The given stores.** The two rules above count only the stores the creation will
give (3.2.3): of a row, the entries whose class has an empty `Needs` or a `Needs` flag that
the settlement class of the group's region carries at construction (the value of its
review at construction, `spec-settlements.md` 5.2). A gated entry the region lacks is not
given and so adds no room to the test, and the warning `store_locked_at_start` (13.2) says
so. The class of a region is configuration, so the rule is evaluated at construction with
the regions of the groups; a group whose room is short only for want of a locked store is
`ErrRoomShape`, not an `ErrNoRoom` of the ledger.

`room_below_upkeep` is an error and not a warning because a limited household that
cannot hold a tick's upkeep cannot be fed, and a configuration that makes every
household starve is not a setting but a mistake (architecture.md 8.5). It checks only
the base room: a role that eats from its stores must still be able to hold, at its base
room, the goods it eats in a tick (6.3.3 needs it).

### 13.2 Warnings

Warnings do not fail validation (`spec-config.md` 7.2), are logged once at `Warn` with
`rule` and `path` at construction, and are returned by `World.Warnings()`.

| Warning | Condition |
|---------|-----------|
| `no_limited_role` | the module is on and no role is limited (no `Base` anywhere): it does nothing |
| `room_below_cover` | a limited role's base room per household is below the load of `CoverTicks` of its upkeep (`spec-agents.md` 12): the default strategy's targets will always be scaled (6.2) |
| `room_below_recipe` | a limited role's base room plus its stores' room is below one tick's upkeep room plus the load of the non-discard outputs of one execution of one of its recipes (at `E` of 10 000, per household): the recipe can never start, because a start leaves the upkeep room free and its inputs are already in the load (5.3.2) |
| `store_class_unused` | a class is named by no row and no flow: it can be given only by a call |
| `store_locked_at_start` | a role row lists a class whose `Needs` flag the class of a region of one of the role's groups lacks at construction: the stores are not given there (3.2.3) |
| `store_levy_below_upkeep` | a flow that targets stores has an `Exempt` below one tick's upkeep cost, at the reference prices, of a limited role: a levy could take an owner's food money (STO-A5, `spec-money-supply.md` 9.4) |

### 13.3 Not validated here

A role with a base room and no store; a store class no role lists (a call may give
one); an unlimited role with an agent that owns a store through a call is refused at the
call (3.6.2); a role change into an unlimited role ends the stores of the agent at the
change (3.6.1a); the fill of a room at any tick is state, not configuration.

## 14. Observability

### 14.1 Events

Two events and one field, all gated by the module (an event of a module that is off is not
emitted, `spec-observability.md` 4.1). Neither event has a map or a pointer, and both are
built only when an observer was supplied (observability-and-logging.md 4.5). Section 19
gives the rows for `spec-observability.md` 4.1 (N-126).

| Event | Position | Class | Fields (besides `Tick`) |
|-------|----------|-------|-------------------------|
| `RoomLimited` | P3.6a, after `WantShortfall`, per region ascending then role; never when `StartsCut`, `StartsRefused`, `BidsCut`, `BidsRejected`, `ShipmentsRefused` and `Discarded` are all 0 (`LimitedHouseholds` and `FullHouseholds` ride along and do not trigger it) | A (aggregate) | `Region`, `Role`, `LimitedHouseholds` (households of limited agents given a view), `FullHouseholds` (households whose free room was 0 in a dimension when their view was built), `StartsCut` (executions not started for room though inputs allowed them), `StartsRefused` (requests refused whole, `no_room`), `BidsCut` (intents the default sized down or dropped for room), `BidsRejected` (orders rejected `no_room`), `ShipmentsRefused` (`consignee_full` of shippers of the group), `Discarded` (units discarded by 6.3.2) |
| `StoreChanged` | the call (`AddStore`, `RemoveStore`); P7.1 for a role change (3.6.1a); P8.3b and `AddFacilities` / `AdjustFacilities` for a facility (steps 2 and 3 of `spec-infrastructure.md` 6.3.2), before CAP's `FacilityChanged` | E (per transition) | `Agent`, `Region`, `Role`, `Class StoreClassID`, `Kind` (`added` or `removed`), `Cause` (`consumer` for a call, `role_changed` for a role change, `facility` for CAP's install and forced removal; constants of `event.go`), `Stores` (the agent's stores of the class after), `Room` and `Load` (the account's, as pairs, after) |
| `AgentSpawned` (a field added) | construction, P7.1(d), `AddAgent` | E | `StoresLocked` (the stores of the agent's role row not given because the region's class lacks the `Needs` flag, 3.2.3; 0 with no gate) |

`RoomLimited` is **bounded by the configuration** (regions times roles), not by agents,
orders or shipments (observability-and-logging.md 4.3 note): it is an aggregate of the
kind of `WantShortfall`. `StoreChanged` is per agent per call, like `AgentRelocated`.
Stores given by a role row at creation have no `StoreChanged`: they are a function of the
role in the configuration and the creation, which `AgentSpawned` reports (3.6.1).

**Why `StoresLocked` is not in `RoomLimited`.** A creation at P7.1(d) is followed by
P8.6, which clears the per-tick counters of every module (`spec-world.md` P0.2, P8.6), so
a counter fed there would be cleared before the next P3.6a and never reported. The
entry's own event carries the count at the creation, where it is a fact of one agent, and
nothing is kept across the boundary (a counter that did survive would be state for the
checkpoint). The count of locked stores is a sum over `AgentSpawned`, which is what an
observer that keeps no table does.

### 14.2 Reasons

Constants of the root package (`event.go`), with their key strings in `internal/logkeys`
only (`spec-ledger.md` 16; N-017):

| Constant (value) | Where it is a `reason` |
|------------------|------------------------|
| `no_room` | an `OrderRejected` (`spec-clearing.md` 3.2, the last row) and a job refusal (`spec-production.md` 4.2, last: after `no_endowment`, `no_title`, `rent_funds` and CAP's reasons) |
| `consignee_full` | a `ShipmentRejected` (`spec-transport.md` 5.4, after `consignee_elsewhere`) |
| `has_store` | a `ShipmentRejected` for a self-carried trip (5.4, after `traveller_away`) and the name of the `region_bound` refusal (`spec-regions.md` 6.5.2) |

The sentinel `ErrNoRoom` is the ledger's and the call's (4.2.1, 3.6.2).

### 14.3 Logs

Through the injected logger, constant lowercase messages, the keys of
observability-and-logging.md 3.3.2 (section 19: `store_class`, a store class ID, and `store_classes`,
a count of classes in the construction summary, are the new keys):

| Level | Message | Attributes | When |
|-------|---------|------------|------|
| `Info` | "world created" gains `store_classes` and the module name `storage` in `modules` | | once, at construction |
| `Warn` | the warnings of 13.2 | `rule`, `path` | once each at construction |
| `Debug` | "bid cut for room" | `tick`, `agent`, `commodity`, `quantity` (cut) | per cut intent, guarded by `Enabled` |
| `Debug` | "job refused for room" | `tick`, `agent`, `recipe`, `reason` | per refusal, guarded |
| `Debug` | "store added", "store removed" (two constant messages) | `tick`, `agent`, `store_class`, `cause` (`consumer` for a call, `role_changed`, `facility`) | per call and per store ended by a role change |

No `Info` and no `Warn` is emitted inside the tick: `RoomLimited` is the counter of the
limits (observability-and-logging.md 3.2.2; N-045, FR-STB-013), so no `Warn` is needed
and none is added (N-126).

### 14.4 Metrics

Derived from the events and the snapshot (observability-and-logging.md 5.3.1), integers,
bounded attributes (5.4). Section 19 gives the rows.

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.storage.limited` | Counter | `{case}` | `economysim.region`, `economysim.role`, `economysim.room_limit` | `RoomLimited` (each count, by `start_cut`, `start_refused`, `bid_cut`, `bid_rejected`, `shipment_refused`) |
| `economysim.store.count` | Gauge | `{store}` | `economysim.region`, `economysim.store_class` | the snapshot's store rows |
| `economysim.room.fill` | Gauge | `{basis_point}` | `economysim.region`, `economysim.role`, `economysim.dimension` | `RoleRow` room and load, `load * 10 000 / room`, absent where the room is 0 |

New attribute values are bounded by the configuration: `economysim.store_class` (the
classes), `economysim.dimension` (`mass`, `bulk`) and `economysim.room_limit` (the five
constants above). The two gauges are fed by a snapshot and are absent, not zero, until
one is passed (observability-and-logging.md 5.3.1).

### 14.5 Snapshot

OBS owns the names (`spec-observability.md` 5.3); this spec asks for these, all plain
data and all functions of saved state (N-125):

- `RoleRow` gains `LimitedHouseholds`, `RoomMass`, `RoomBulk`, `LoadMass`, `LoadBulk`:
  sums over the limited agents of the role in the region (the pairs of 3.4, the weight
  folded into the room). The row exists per (region, role), so the size is bounded by
  the configuration;
- a `StoreRow` per (region, class) with a non-zero count: `Stores` (a count of stores, not of agents or households).

INT's `AgentState` (the result of `Agent(a)`) gains a group "Room" with `Room`, `Load`, `Pledge`, `Free` (`FreeRoom`: 0 where negative after a cut, 6.1.1) and `Stores`
(`spec-integration.md` 4.7.2 and OBS 5.4, section 19); `Market(r, c)` is unchanged.

### 14.6 Cost with no observer

With the default observer and logger, no event is built and no log call allocates (the
flag of observability-and-logging.md 4.5). The counters that exist only to build
`RoomLimited` are kept only under the same flag. The module adds one branch per
operation on an unlimited account (4.5) and no allocation to a step (NFR-STO-002).

## 15. State contributed to a checkpoint

15.1. **Saved: the stores of each agent**, once, in the agent record of
`spec-checkpoint.md` 4.2 tag 3, which hosts it (N-128, F3 there): when the module is on,
a count `n` and `n` pairs (class index in the configuration's class order, count),
ascending by class index, a count of 0 being the common case. The module has **no
section of its own**: the agent record is its home, and the shape of the record depends
on whether the module is on, which the configuration hash already fixes
(`spec-checkpoint.md` 4.2.1). The state of the cost flow (`Total`, `Scale`, `Ended`,
the carries) is MON's, in tag 13 (`spec-money-supply.md` 14), and a carry of a stores
target is per account as for households: tag 13 lists, for each flow with a **stores**
target and a carry shape, the (account, carry) pairs in ascending account ID, open
accounts only, a third case beside the household and pool targets (a row owed to
`spec-checkpoint.md` 4.2, section 19).

15.2. **Derived and rebuilt, never saved** (`spec-checkpoint.md` 4.3): the room of an
account (a function of the role, the weight and the stores), its load (the sum over its
lots), its pledge (from the job's pending output, saved with the job in tag 6, and the
shipments in flight, saved in tag 17), the reserved room (0 at a boundary), the stored
load (8.3.2), the table of sizes and the free room.

15.3. **Restore validation** (`spec-checkpoint.md` 6.2 gains these, section 19): a class
index is below the count of classes; the stores of an agent are at most
`MaxStoresPerAgent`; an agent of an unlimited role owns no store (a role change into one
ends the stores, 3.6.1a, so a world never saves that state); and for every limited
account `Load` is within the room in both dimensions. The pledge is not tested: after a
cut of the room it may be above it (5.5.3). A failure is `ErrCheckpointCorrupt`.

15.4. **Format version.** The agent record changes shape with the module on, so the
format version changes with the first release that has the module
(`spec-checkpoint.md` 9.3), and a fixture of it joins `testdata/checkpoints/`.

15.5. **Not state:** the room's commitments (the pledge is derived; the reservation is
inside a tick), the events, the warnings.

## 16. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. The checklist of
`spec-stability.md` 9 was followed: 9.1 each rule of sections 5 and 6 was read for the
chain it creates; 9.6 the delays were checked against the neighbours, and each entry
names the ones that match: a job's `Duration` and the effective welfare window
`T = max(WelfareWindow, WindowCycles * longest Duration)` (`spec-welfare-and-wages.md`
2.3.3), a shelf life, the transit time `k`, `ReviewTicks` and `RoleCooldown`
(`spec-spawner.md` 6.2).

**STO-A1 A want above the room is invisible to the price and pins the stock signal.**
*Kind:* loop. *Loop:* an agent's target above what its room can hold (-) the stock
ratio stays below 1 (-) the belief centre rises (`spec-beliefs.md` 5.5) (+) its limit
price (+) the market price, which other buyers read through the window and the
evidence (+) their targets' cost and the unmet signal. R, with no stop but the belief's
reference band, because the want cannot become a purchase: the bid above the room is
rejected and has no outcome, so nothing in the book says the want is empty (5.2.5).
*Delay:* the belief adjustment (a few ticks, `spec-beliefs.md` 5.2). *Owner:* STO 6.2.
*Damper:* STO-D1, the target scaled to the room. *Undamped:* a consumer's strategy that
reports a target above its room to its own belief rule. *Condition:* the target the
belief rule receives is within the room (6.2.2). *Neighbours (9.6):* the delay is a few
ticks and below `T` and the price window; no match. *Test:* STO-S1 (a limited staple
household with a target above its room: the centre of its belief stays inside the
reference band and the market price does not drift). *Status:* hand.

**STO-A2 Production stopped by a full room.** *Kind:* loop. *Loop:* output (+) the
stock (+) the room taken (-) the free room (+) the starts permitted (+) output; and the
stock is lowered by sales and spoilage (-). One negative link, so balancing (B), with
a delay that makes a cobweb: a stock that fills the room stops the starts, the room
drains by sales and spoilage and the starts resume a `Duration` later. *Delay:* the
recipe's `Duration` plus the time the stock takes to clear. *Owner:* STO 5.3 (a
refusal at the start). *Damper:* D24, the production halt (`spec-agents.md` 7.1.5),
which acts at a threshold of 30 000 basis points of a one-tick target, usually before
the room does and without a refusal; STO-D2, a cohort's executions cut to what fits
(a partial start, 5.3.3); the stock rule of the belief, which lowers an asking price
as stock rises (L06). *Undamped:* a weight-1 producer, whose start is all or nothing
and whose halt threshold is above its room. *Condition:* the halt acts first when the
room is above the load of the halt threshold, `ProduceHaltRatio / 10 000` times the
output target (`spec-agents.md` 7.1.5), **plus the load of one whole job's output** (the
start needs the lump free, `Duration` times the one-tick output, and not the one-tick
output), plus the upkeep room (3.4.5): `Room_d > H_d + lump_d + U_d`; otherwise the
room refuses the start before the halt acts, and the refusal is counted
(`StartsRefused`) where the halt's is not. The warning `room_below_recipe` is the
nearest check and tests the empty-stock case only. *Neighbours (9.6):* the period of the
cobweb is about two `Duration`s, which is near the effective window
`T = max(WelfareWindow, WindowCycles * longest Duration)` by construction
(`spec-welfare-and-wages.md` 2.3.3), a match, so the welfare record spans the cycle; and
a commodity whose shelf life is near the job's `Duration` clears the room by spoilage at
the same period (L25). *Test:* STO-S3. *Status:* hand; the model of 17 checks the sizing
bound only and shows no dynamics.

**STO-A3 A full room blocks the purchase of upkeep.** *Kind:* hazard. *Loop:* unsold
goods fill the room (+) free room falls (-) a tier 1 bid is cut (-) the household is
unfed (+) privation (+) the ladder (`spec-spawner.md` 5): relief, role change, removal.
A poverty trap through a wall. *Delay:* the privation count. *Owner:* STO 6.3.
*Damper:* STO-D3, the discard that makes room to eat (6.3.2) for a room full of goods;
STO-D6, the upkeep room that a job start and a dispatch leave free (3.4.5) for a room
full of a pledge, which nothing can discard; the error `room_below_upkeep` (13.1); the
scaling of 6.2, which leaves the room of `Reserve`.
*Undamped:* a producer's tier 2: inputs are never made room for, by design, so a
producer whose room is full of unsold output does not buy inputs and produces nothing
until it sells (STO-A2). *Condition:* a base room at least one tick's upkeep load, and
every commitment tested against `Free - U` (5.3.2, 5.4.1). *Neighbours (9.6):* the
privation count against `ReviewTicks` (10) and `RoleCooldown` (30) of the ladder; no
match at the placeholders. *Test:* STO-S1. *Status:* hand.

**STO-A4 First come inside one room and at one consignee.** *Kind:* bias. *Loop:* none:
it chooses which of one agent's bids, or which of several shippers' cargo, is served
when the room is short. Inside an agent the order is its plan's, which the default
builds in tier order (5.2.3); at a consignee it is P3.2's ascending agent ID and P3.3's
formation order (5.4.2). *Owner:* STO 5.2.3, 5.4.2 with `spec-world.md` 6.5 (the order
is the world's, not this spec's tie-break: WLD-A2 is the same hazard). *Damper:* none:
the room is the consignee's own, a refused shipper retries the next tick, and nothing is
lost. *Undamped:* by design. *Condition:* a statistic that groups by agent ID does not
read a difference in shipments served as real before `RoomLimited.ShipmentsRefused` is
checked. *Test:* STO-S4. *Status:* hand.

**STO-A5 The store levy is a drain with no return.** *Kind:* hazard, a case of
STB-L69 and L71 (`spec-money-supply.md` 9.1, 9.4). *Loop:* the levy (+) the owner's
money removed (-) its balance (-) upkeep bought (+) unfed (+) privation. *Owner:* STO 7,
MON-A4. *Damper:* D43, the `Exempt` of the flow; the warning `store_levy_below_upkeep`;
a pair with a grant. *Undamped:* an unpaired levy deflates by design. *Condition:*
`Exempt` of at least one tick's upkeep cost. *Test:* STO-S5. *Status:* hand.

**STO-A6 Storage arbitrage is bounded by the room.** *Kind:* constraint. *Loop:* none of
its own: a strategy that buys cheap and holds for a dear season is a balancing loop of
its own making (a price gap closing, with the delay of the holding), and the room is
what limits the quantity it can hold, so the swing it can cause is bounded by the sum
of the room of such strategies. *Owner:* STO 10; the strategy is the consumer's.
*Damper:* the room (STO-D4); the shelf life (L25). *Condition:* the consumer's
strategy reads its free room and respects it (6.4). *Test:* STO-S2. *Status:* untested.

**STO-A7 A cut of the room under a pledge.** *Kind:* hazard. *Loop:* households unfed
(-) the weight of the agent falls through the ladder or the population module (`w`)
(+) its base room falls (+) its free room, clamped at 0 while the pledge is above it
(+) the food it can buy (-) households unfed. The two negative links make it
reinforcing (R): a cohort that loses households loses room and the pledge, which is
not scaled with it (5.5.3), keeps the rest of the room until it is credited. *Delay:* the
job's remaining `Duration` or the cargo's transit time, whichever pledge is over the
room; the privation count. *Owner:* STO 5.5.3. *Damper:* STO-D5, the pledge is cut at the
credit (5.3.4, 5.4.4), so the hazard lasts at most the pledge's own life; the weight
reduction is itself bounded by the ladder's rungs (`spec-spawner.md` 5). *Undamped:*
a long job (a year-long `Duration`) with a pledge near the whole room, in a cohort that is
losing households. *Condition:* a base room per household that holds the upkeep, and a
job whose lump is well below the room (the warning `room_below_recipe` names the
neighbour of this). *Neighbours (9.6):* the delay is a `Duration`, near `T` by
construction (a match: the welfare record of the cohort spans the hazard).
*Test:* STO-S6. *Status:* hand.

**Dampers.** *STO-D1* the target scaled to the room (6.2.2); *STO-D2* the start cut to
the executions that fit, so a cohort smooths and a weight-1 agent does not (5.3.3);
*STO-D3* the discard that makes room for upkeep (6.3.2), together with the error
`room_below_upkeep` that makes it always possible (6.3.3); *STO-D4* the hard room,
which is a constraint and not a stabiliser (5.1, FR-STO-005): it bounds and does not
steer, and it turns an excess into a counted refusal; *STO-D5* the pledge cut at the
credit after a cut of the room (5.3.4, 5.4.4); *STO-D6* the upkeep room left free by
every commitment (3.4.5).

**Scenarios proposed** (S91 and on, numbered by the supervisor; every one runs with the
module on over a range of seeds with `WithAudit` and asserts the invariant of 4.4 at every
tick): *STO-S1* a limited staple household and a producer with a base room just above
one tick's upkeep plus one job's load: no household goes unfed for room, `RoomLimited`
counts the cuts; *STO-S2* a buffer store's owner with a directed strategy fills in a plenty
season and releases in a lean one: the free room bounds the fill, the levy is paid, and
the goods spoil on their clock; *STO-S3* a producer glut: a weight-1 producer with a
full room and a price at the floor stops, recovers when it sells; *STO-S4* two shippers
and one consignee with room for one cargo: the first is served, the second is
`consignee_full` and retries; *STO-S5* the levy against its pair and its exemption: the
boundary balance of the flow equals its total and no owner is unfed by it; *STO-S6* a
cohort of weight 10 with a job of one execution and a cargo inbound, whose weight is
cut to 5 by the ladder: the load fits, the output is credited as far as the room goes and
the rest disposed, the cargo is credited or diverted, and the audit holds (the case of
17, part 4, as a run).

## 17. What was run

One script, in Docker with `PYTHONDONTWRITEBYTECODE=1`, `doc/specs/models/sto/room.py`
with its output `room.out` (`doc/specs/models/README.md` gains a row, section 19). It is
a model of the **rules** of sections 5 and 6 with integers and placeholder numbers. It
shows arithmetic and an invariant. It does **not** show that any default is stable, and
it has no price, belief or market, so it cannot show STO-A1 or STO-A2. Part 1 has **one
dimension** (the exact test and the closed form of 5.3.2 for a pair are the same
inequality per dimension); the pair is exercised in parts 2 to 6.

| Run | What it shows | Result |
|-----|---------------|--------|
| 1. the start size, 20 000 random recipes (0 to 3 input lines of 0 to 5 units, 1 to 3 output lines of 1 to 8 units, sizes 0 to 4, `E` 5 000 to 12 500, a random residue, free room 0 to 80, `k0` 1 to 12), one dimension | the closed form of 5.3.2 against the exact largest `k`: soundness (the exact pending output of its `k`, with the executor's own residues, fits) and its distance from the exact value. The free room it takes is `Free - U`, a number the caller subtracts | sound in all 20 000; equal to the exact value in 18 964, below it in 1 036, the largest gap 11 executions. 6 cases have a gap of 3 or more: gaps 11 and 8 with `c` of 0 (the output load equals the input load) and free room 1 and 2; gaps 5, 3, 3 and 3 with `c` of 2 000, 15 000, 8 000 and 7 500 (output load above input load by 0.2 to 1.5 units) and free room 1, 5, 2 and 4. So a large gap needs an output load within about one unit of the input load, in a room with up to 5 units free (5.3.2) |
| 2. the target scaling, two cases | 6.2.2: targets 40 and 30 in a room (100, 100) with pledge (10, 10) and a reserve of 6 of the first; and the clamp where the pledge leaves a negative available room | scaled by 6 461 basis points to 25 and 19, the load 69 and 82 within the available 78 and 84; with pledge (95, 10) the available mass room is negative, clamped to 0, `f` 0 |
| 3. a random sequence, 4 000 ticks, one limited account (room (60, 60), three commodities, the first eaten 2 units a tick, jobs of 2 to 4 ticks, cargoes of 1 to 3 ticks, bids of three commodities, consumption and spoilage) | the invariant under bids with reserved room, job starts and dispatches that leave the upkeep room free (5.3.2, 5.4.1), completions and arrivals from the pledge, and the tier 1 bid always placed | `Load + Pledge + Reserved` never above the room and `Pledge + U` never above the room after a commitment (both asserted after every step); 2 997 bids admitted and all settled whole; 7 826 bids, 258 job starts and 407 dispatches refused for room; 245 tier 1 bids fitted (0 units discarded in this run: its surplus never filled the room, and the discard is exercised by part 6); the household is never unfed for room; the highest committed load 58 of 60 |
| 4. a reduction of the weight | 5.5.3: 20 000 random reductions with `Load + Pledge` within the room before; and the case of a job pledge and a cargo over the cut | the scaled load fits the new room in all; the case: weight 10 to 5, base 100 and a store of 200, room 1 200 to 700, load (200, 100) to (100, 50), pledge (940, 940) above what is left; the cargo of 40 is credited whole, 280 of the 450 of output are credited and 170 disposed (5.5.3, 18.4) |
| 5. role changes, 5 000 random cases | 5.5.2.1: the excess always ends within the room (a four-commodity, two-dimension random load, lots of 1 to 12) | all fit; the worked case of 18.3: 20 units of the second commodity and 10 of the first go to the estate |
| 6. the discard, 582 random cases with `Pledge + Keep` within the room | 6.3.3: the tier 1 units always fit after the discard of 6.3.2 | all fit (433 units discarded in all); the case: room (60, 60) with 10 units of commodity 1 and 6 of commodity 2 held and 2 units of commodity 0 wanted: 1 unit of commodity 1 is discarded |
| the arithmetic of 18.1 and 18.2 | the two worked examples | the numbers of 18 |

**What it does not show:** a price effect of a binding room (no market); the
interaction with the production halt; the cost of the discard rule; any cohort larger
than 1 in parts 1 and 3 (one execution at a time; part 4 has a cohort of 10 and a job of
one execution); the trainer's case of two jobs on one account; and the dynamics of
STO-A2, STO-A3 and STO-A7, which need the library.

## 18. Worked examples

Both settings are invented and flagged: the words in them are data and no code branches
on them (architecture.md 9.1, `AGENTS.md`). Every number is a placeholder. The
arithmetic is in `models/sto/room.py` (17).

### 18.1 Example A: a barn on a farm (a pre-industrial setting)

**Data.** A commodity `grain` of mass 2 and bulk 1. A role `grower` with a base room of
mass 400 and bulk 200 per household (a barn), weight 1, an upkeep of 3 grain a tick
(load `(6, 3)`, within the room, so `room_below_upkeep` passes). A source recipe `sow`
with no inputs and an output of 120 grain per execution, `Duration` 100, `E` = 10 000.
No store class: the room is the household's own.

**The load of one harvest.** One execution creates 120 grain, a load of `(240, 120)`.

**The start (5.3.2)**, for a grower with a residue of 4 000. The upkeep is 3 grain, so
the upkeep room `U` is `(6, 3)` (3.4.5). `I` = 0; `A_mass` = 120 * 10 000 * 2 = 2 400 000
and `A_bulk` = 1 200 000, so `c` = `(2 400 000, 1 200 000)`; `q * E` = 1 200 000 is a
multiple of 10 000, so `r'` = 0 and `R_d = 10 000 * (Free_d - U_d)`.

| Grain held | Free room | Free less `U` | `R` (mass, bulk) | `k` by mass, by bulk | Start |
|-----------:|-----------|---------------|------------------|----------------------|-------|
| 0 | (400, 200) | (394, 197) | (3 940 000, 1 970 000) | 1, 1 | `k` = 1 |
| 100 | (200, 100) | (194, 97) | (1 940 000, 970 000) | 0, 0 | refused, `no_room` |
| 160 | (80, 40) | (74, 37) | (740 000, 370 000) | 0, 0 | refused, `no_room` |

**What the barn does.** After a harvest of 120 grain the grower holds (say) 120 grain,
load `(240, 120)`, free `(160, 80)`: the next sowing needs free `(240, 120)` and the
upkeep room `(6, 3)`, that is a load of at most `(154, 77)`, 77 grain. The grower must
sell or eat 43 of the 120 before it may sow again, whatever the date: the room is the
barn, a harvest needs one empty and the grower's own food is never crowded out by it.
Its asks are all of the surplus above `Keep'` (`spec-agents.md` 7.3.5), so it normally
holds far less; the case is a glut, which holds the stock and so holds the next sowing
(STO-A2).

**The target (6.2).** `CoverTicks` 3 (placeholder): `UseTgt` of grain is `3 * 3 = 9`;
`Reserve` 3; `Keep` 12, load `(24, 12)`, far below the room; `f` = 10 000, no scaling.
A store is unnecessary for this household: the room is set by one harvest.

**The discard (6.3.2).** The grower also eats `salt` (mass 1, bulk 1, upkeep 1) and
holds 200 grain, a load of `(400, 200)`: free `(0, 0)`. Its tier 1 bid for 1 salt needs
`(1, 1)`. `Surplus` of grain is `200 - 12 = 188`. The shortfall is `(1, 1)`; relative to
the room it is 1 of 200 in bulk against 1 of 400 in mass, so bulk binds (5.5.2.1), and
grain is the only commodity with a surplus. One unit of grain frees `(2, 1)`, which
covers `(1, 1)`, so the plan discards 1 grain (`disposal`), bids for 1 salt and asks for
the remaining 187 units of surplus, as ever.

### 18.2 Example B: the keeper of a buffer store in a colony (a space setting)

**Data.** A commodity `ration` of mass 1 and bulk 2. A role `steward` with a base room
of mass 40 and bulk 60 per household (its quarters, enough for the 2 rations a tick of
its upkeep, load `(2, 4)`), and one store class `depot` of mass 6 000 and bulk 9 000.
The steward is the player's agent: an external agent with a directed strategy
(`spec-integration.md` 4). A levy flow `depot_upkeep` of 40 minor units a tick per
`depot`. Nothing in the data names a granary: that is what the player makes of a store.

**Giving the store.** `AddStore(steward, depotID)` between ticks, `depotID` being the
`StoreClassID` the consumer resolved from `Config()`: the room is
`(40 + 6 000, 60 + 9 000) = (6 040, 9 060)`, the owner's account is now refused by the
relocation guard with `has_store` (9), `StoreChanged{Kind: added, Cause: consumer,
Stores: 1}` is emitted, and from the next P1.7 the owner pays 40 a tick.

**What fits.** `min(6 040 / 1, 9 060 / 2) = 4 530` rations: bulk binds.

| Step | Request | Free room before | Result | Free room after |
|------|---------|------------------|--------|-----------------|
| 1 | bid for 4 500 rations, load `(4 500, 9 000)` | `(6 040, 9 060)` | admitted, room reserved, settles at P4.1 | `(1 540, 60)` |
| 2 | bid for 40, load `(40, 80)` | `(1 540, 60)` | rejected `no_room`: bulk 80 above 60 | `(1 540, 60)` |
| 3 | a shipment of 20 rations to the steward is dispatched: load `(20, 40)` pledged | `(1 540, 60)`, less the upkeep room `(2, 4)`: `(1 538, 56)` | dispatched (`(20, 40)` is within `(1 538, 56)`: `consignee_full` not raised) | `(1 520, 20)` |
| 4 | bid for 11, load `(11, 22)` | `(1 520, 20)` | rejected `no_room` | `(1 520, 20)` |
| 5 | bid for 10, load `(10, 20)` | `(1 520, 20)` | admitted | `(1 510, 0)` |

**The levy.** `depot_upkeep`: `n` = 1, `Amount` 40: due 40 a tick with no carry (a whole
amount per tick). The boundary balance of the flow's reason falls by 40 a tick and
`BoundaryFlow` reports a sink of 40 (`spec-observability.md` 4.4.1).

**Leaving.** `Relocate(steward, other)` is refused `ErrRegionBound` naming `has_store`.
`RemoveStore(steward, depotID)` is refused `ErrNoRoom` while the load is `(4 500, 9 000)`
plus a pledge, because the base room of `(40, 60)` cannot hold it; the call tests the
load plus the pledge, and the pledge plus the upkeep room `(2, 4)`, against the room
without the store (3.6.2). After the steward has sold down to a load plus pledge within
`(40, 60)` the removal succeeds, `StoreChanged{Kind: removed,
Cause: consumer, Stores: 0}` is emitted, the levy stops at the next P1.7 and the guard
no longer refuses.

**What a default strategy would do.** Nothing with the depot: its targets are cover
periods of its own use (`spec-agents.md` 7.3: with `CoverTicks` 3 and an upkeep of 2,
6 rations), and it would offer anything above `Keep`. The depot is the player's: its
directive's intents fill it (`Directed`, `spec-agents.md` 10).

### 18.3 Example C: what goes first (the rule of 5.5.2.1)

An invented pair of commodities: `A` of mass 1 and bulk 3, `B` of mass 3 and bulk 1. An
account with a room `(100, 100)` holds 30 of `A` in three lots of 10 and 40 of `B` in four
lots of 10, oldest first. Its role changes to one with that base room.

| Pass | Load | Shortfall `S` | Binding (relative to the room) | Commodity (greatest load in it) | Moved |
|-----:|------|---------------|--------------------------------|---------------------------------|-------|
| 1 | `(150, 130)` | `(50, 30)` | mass: 50 of 100 against 30 of 100 | `B`: 120 in mass against `A`: 30 | two lots of `B`, 20 units, `(60, 20)`, until `S_mass` is met |
| 2 | `(90, 110)` | `(0, 10)` | bulk | `A`: 90 in bulk against `B`: 20 | one lot of `A`, 10 units, `(10, 30)` |
| end | `(80, 80)` | `(0, 0)` | | | 20 of `B` and 10 of `A` go to the estate under `estate_goods` and are listed in `RoleChanged.GoodsListed` as that field counts them |

The second pass is the re-evaluation: after the first commodity the binding dimension is
the other one, and the ranking is taken again, not continued. Compared absolutely, the
mass count and the bulk count would not have been comparable (`spec-goods.md` 3.2); the
model of 17 (part 5) reproduces the table.

### 18.4 Example D: a cut of the room under a pledge (5.5.3)

The data of 18.1 with a store: a cohort of weight 10 with a base room of `(100, 100)` per
household and one store of `(200, 200)`: the room is `(1 200, 1 200)`. It holds 100 of `A`
(mass 2, bulk 1: load `(200, 100)`), has one execution of a job pending 450 of `B` (mass
2, bulk 2: load `(900, 900)`) and a cargo of 40 of `C` (mass 1, bulk 1) inbound: the
pledge is `(940, 940)` and, with the load, `(1 140, 1 040)`, within the room `(1 200,
1 200)`, as every commitment must be. The role of the example has no upkeep, so the
upkeep room is 0.

The ladder reduces the weight to 5. The room is `(500 + 200, 500 + 200) = (700, 700)`, the
load is `(100, 50)` (50 of `A`), and the pledge is still `(940, 940)`: the job's
`Executions` is `min(1, 5) = 1`, so nothing is scaled, and the cargo is not touched. The
free room is `700 - 100 - 940`, negative, read as 0: every new commitment of the cohort is
refused (3.4.4) and the default strategy's `f` is 0 (6.2.2).

At P1.3 the cargo is credited line by line up to `RoomLeft = (600, 650)`: 40 of `C` fit,
whole. At P2.1 the output is credited up to what is left, `(560, 610)`, 2 per unit in each
dimension: `min(450, 560 / 2, 610 / 2) = 280` of `B`; the other 170 are created and
disposed in the same step under `disposal`, and `RecipeExecuted` reports them. Nothing is
lost silently and the audit holds (4.4).

## 19. Notes settled, and what this spec needs from the other specs

### 19.1 Notes

| Note | Disposition |
|------|-------------|
| N-011 (storage, carriage and the chargeable size) | **Applied for STO.** The chargeable size and `BulkFactor` are not used by storage: capacity is a pair tested exactly and the cost is per store (8.4), so the over-charge of `spec-goods.md` 3.5 does not arise; N-011's remaining parts were CAR's and MER's and are closed there |
| N-046 (register entries owed) | **Applied for STO** (16): STO-A1 to A7, dampers STO-D1 to D6 and scenarios STO-S1 to S6, provisional; the supervisor numbers them. Remains open for EMP, SAV, CRD, RSK, FIS, PLC and LND |
| N-120 (a migrant carries its whole account for free) | **Applied for a limited role** (9.5): a store owner cannot move, so a relocation carries at most the base room; **left open** for an unlimited role, which a pack closes by giving it a base room |
| N-016 (the parameter registry) | the rows of 12.4; no unnamed parameter; the registry test of testing.md 6.7 is still to be written |
| N-017 (words and names) | terms in 19.3; setting words only in the flagged examples of 18 and the one flagged sentence of the header (a pack's word for a buffer store) |
| N-027 (no default is stable) | every number is a placeholder (12.2, 17, 18); no default is called stable |
| N-103 (several writers in parallel) | provisional IDs, and the seams listed in the handoff |
| N-126 (a new event or `Warn` adds a row) | the rows of 14.1 and the metric and key rows of 14.3 and 14.4 are in the handoff; no `Warn` is added |
| N-128 (a hosted per-agent field is saved once) | the stores are saved once, in the agent record (15.1) |
| Not tagged STO and read: N-002 (closed), N-021, N-045 and N-065, N-090, N-101, N-104, N-107, N-122, N-125, N-130 | N-002: its update said "production never refuses an output for lack of room"; it stands with one exception (5.3.4): the room decision moves to the start, and a job's output is disposed for lack of room only after a cut of the room (5.5.3). N-021: no faucet, sink or reason added. N-045, N-065: `RoomLimited` counts every silent limit, and `AgentSpawned.StoresLocked` the stores not given at a creation (14.1). N-090: `has_store` (9.2). N-101: the pledge of `Inbound` is part of the free room, and whether the default `Target` counts `Inbound` is still AGT's. N-104: no region-keyed table (12.1.1). N-107: the cost is a MON flow (7). N-122: the order inside one room and at one consignee is first come, registered STO-A4. N-125: every snapshot field is a function of saved state (14.5). N-130: IND gains one ask, a storage-use measure (handoff) |

### 19.2 What this spec needs from the other specs

None of this is made here. Each is a row of the handoff's table
(`.supervisor/sto.handoff.md`), with the exact change.

| Spec | What |
|------|------|
| `spec-ledger.md` | a section for room (4 here): the account's room, load, pledge and reserved room; the operations including `RoomLeft`; the credit forms and the credit from the pledge bounded by `RoomLeft`; the audit identities; `ErrNoRoom` |
| `spec-clearing.md` | 3.2: the last reason `no_room`; 3.3: a bid reserves room |
| `spec-production.md` | 4.2: `k_room` (with the upkeep room left free) and the reason `no_room`; 4.3: the pledge; 5.3.2: the sentence that production loses output for lack of room only after a cut of the room; 5.4, weight row: the pledge may then exceed the room |
| `spec-transport.md` | 5.4: `consignee_full` and `has_store`; 5.5.1: the pledge at dispatch; 5.7 item 2 and 5.8: the clause "or the room"; 5.9.4: the cargo is not scaled with a weight reduction, so the arrival may divert; 10.4 |
| `spec-agents.md` | 6.4.1: the room group of the view; 7.3: the scaling and `Avail` clamped; 7.5.4: the free room in the quantity; 7.3.5: the discard; 8: the module on |
| `spec-money-supply.md` | 4.1, 4.3: the target kind `stores`, the field `Store`, the amount per store; 12.1: the dependency and the unknown class; 14: the carry of a stores target |
| `spec-regions.md` | 6.5.2: the row `has_store` |
| `spec-spawner.md`, `spec-skill-barriers.md` | 4.2: the excess goods of a role change go to the estate (5.5.2), and a role change to a role with no base room ends the agent's stores (3.6.1a); SKL 4.4 cites it |
| `spec-subsistence.md` | 5.5: `no_room` is a reason for blocked |
| `spec-merchants.md` | 6.1: a merchant whose view has stores plans no self-carried trip; 9.5: `consignee_full` is a failure of the formation counted in `DispatchRefused` |
| `spec-integration.md` | 3.6, 4 (4.7.2, new 4.8), 5, 11.2: `AddStore`, `RemoveStore` with `StoreClassID`, the group of `AgentState` (4.7.2), the sentinels including `ErrAgentAway` |
| `spec-world.md` | 6.3 (P1.3: the credit of an arrival bounded by the room), 6.4 (P2.1, P2.5, P2.7: `k` also bounded by the room; the credit of a completion bounded by the room), 6.5 (P3.6a), 6.11, 8.3, 13 |
| `spec-checkpoint.md` | 4.2 tag 3, tag 13 (the stores target carry), 6.2, 9.3 |
| `spec-observability.md`, the rules | 3.2 (`StoreClassID`), 4.1 (events, and the field `StoresLocked` of `AgentSpawned`), 5.3, 5.4, 6, 7; `observability-and-logging.md` 3.3.2, 4.3, 5.3 |
| `spec-goods.md` | 3.3, 3.4, 11 (open question 1): storage does not use the chargeable size |
| `spec-config.md` | 3.2, 3.7, 4.1, 7.3, 7.4 |
| `spec-viability.md` | 8.5: a warm job is tested against the room at construction (13.1, `room_below_warm_job`) |
| `specs/README.md`, `spec-stability.md`, `spec-calibration.md` | the owner rows, the register entries, the registry rows |

### 19.3 Terms (N-017)

**Store** (a facility that adds room to its owner's account), **store class**, **room**
(the pair an account may hold), **base room** (the household's own room, a function of the
role), **limited** (an account with a room), **load**, **pledge** (the room that goods
not yet in the account will take), **reserved room**, **free room**, **upkeep room** (the
load of one tick's upkeep, left free by every commitment), **cut of the room** (a fall of
the room below what a pledge needs: a weight reduction or a role change), **stored load**
(the part of the load above the base room, up to the stores), **buffer store** (a store
of large capacity held against shortage), **owns a store**. "Home" is not used (a
household's region is its region). A pack's own word for a kind of store (the first
consumer's is named at the top of this spec) is not a term: the library has stores. "Capacity" in `spec-goods.md` 3.2 and
here is a pair; the **chargeable capacity** of `spec-goods.md` 3.4 is not used.

## 20. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-STO-001 | With `Modules.Storage` absent, nothing of this spec MUST run: no account MUST have a room, no state MUST be added to an agent, no reason or event MUST exist, and a run MUST be bit-identical to one built without the module. With it present and no role limited, the results MUST be identical to a run with it absent. |
| FR-STO-002 | A store class MUST have an ID, a mass and a bulk capacity each from 0 to `MaxRoom`, not both 0, and a label that is not hashed; no code MUST branch on a class ID. |
| FR-STO-003 | An agent MUST be limited if and only if its role's resolved base room (the row of its role, else `Storage.Base`) is non-nil; a pseudo-agent MUST NOT be limited. |
| FR-STO-004 | The room of a limited account of weight `w` MUST be `w` times the base room plus the sum of the capacities of its stores, in each dimension, with checked arithmetic; a store's capacity MUST NOT be multiplied by the weight. |
| FR-STO-005 | For every limited account, at every point of a tick and at every boundary, `Load + Reserved` MUST be at most the room in both dimensions; a commitment (an admitted bid, a job start, a dispatch) MUST be made only if `Load + Pledge + Reserved` with it is at most the room, so the free room is never negative at a commitment. The pledge MAY exceed what the room can hold only after a cut of the room (FR-STO-017), and the free room is then read as 0 and no new commitment MUST be made. |
| FR-STO-006 | The load of an account MUST be the sum over its lots of quantity times `Mass` and of quantity times `Bulk`, maintained by every operation that changes a lot, and a commodity of mass 0 and bulk 0 MUST take no room. |
| FR-STO-007 | A goods credit to a limited account that is unmarked MUST be refused with `ErrNoRoom` unless it fits the free room, a credit marked from reserved MUST draw on the reserved room, and a credit marked from pledge MUST draw on the pledge and be bounded by `RoomLeft` (`Room - Load - Reserved`); a refused operation MUST change nothing. |
| FR-STO-008 | A bid of a limited agent MUST be rejected with the reason `no_room` if its quantity times each size exceeds the free room in either dimension, checked after `insufficient_stock`, and an ask and a pseudo-agent's order MUST NOT be subject to it. |
| FR-STO-009 | An admitted bid MUST reserve its room with its cash, the settlement of its trades MUST draw on the reserved room and MUST NOT fail for room, and the room of the part that did not trade MUST be released at the end of P4.1. |
| FR-STO-010 | At every tick boundary the reserved room of every account MUST be 0. |
| FR-STO-011 | The start of a job of a limited principal MUST use `k = min(k0, k_room)` executions, with `k_room` part of the `k0` derived at P2.5 and evaluated again at P2.7, MUST be refused with the reason `no_room`, consuming nothing, when `k` is 0 and `k0` was not, and `no_room` MUST be last in the order of the reasons: after `no_endowment` and, where a module adds them, after `no_title`, `rent_funds`, `no_facility` and the build reasons of `spec-infrastructure.md` 4.2.2 (the order is `spec-production.md` 4.2's). |
| FR-STO-012 | `k_room` MUST be the largest `k` up to `k0` with `k * c_d <= R_d` in both dimensions (5.3.2), `R_d` taking `Free_d - U_d` and the non-discard outputs only, with the executor's own residues and the exemption for an output whose `q * E` is a multiple of 10 000, and the exact pending output of the `k` chosen MUST fit the free room less the upkeep room once the inputs are taken. |
| FR-STO-013 | The start MUST set the principal's pledge to include the exact load of the job's pending output (a discard output and a `MaxQuantity` overflow take none); the completion MUST credit the output from the pledge, up to `RoomLeft`, and with the room as it was at the start MUST NOT fail; what a cut of the room leaves no room for MUST be created and disposed in the same step under `disposal`, in commodity order and whole units; the `MaxQuantity` rule of `spec-production.md` 5.3.2 MUST still apply. |
| FR-STO-014 | A shipment to a consignee other than its shipper MUST be refused with `consignee_full`, after `consignee_elsewhere`, if the load of its lines exceeds the consignee's free room less its upkeep room in either dimension, and a dispatched one MUST pledge the load of its lines on the consignee. |
| FR-STO-015 | The arrival of a cargo MUST credit each line from the consignee's pledge, in line order and up to `RoomLeft`, and MUST lower the pledge by the load of the line whatever part was delivered, spoiled or diverted, and cargo that does not fit MUST be diverted to the estate of the destination, whole units. |
| FR-STO-016 | Every credit of goods to a limited account MUST be one of a settlement from reserved room, a job's output or a cargo's arrival from the pledge, a creation validated before it, or a move to the estate; any other MUST NOT exist, and a credit that meets `ErrNoRoom` where room was committed MUST be a programmer error that panics, or fails the step with `ErrAuditFailed` under `WithAudit`. |
| FR-STO-017 | A reduction of an agent's weight MUST NOT take its load above its room. The pledge is not scaled with the room and MAY then exceed what is left; it MUST be cut at the credit (FR-STO-013, FR-STO-015) and the reduction MUST NOT be refused for it. |
| FR-STO-018 | A role change to a role with a base room that leaves the account's load above the new room MUST move goods to the estate under `estate_goods` by the rule of 5.5.2.1 (the binding dimension is the one whose shortfall is the largest relative to the room, the commodity of the greatest load in it first, ties by `CommodityID`, whole lots oldest first, re-evaluated before each commodity) until the load fits, and MUST NOT be refused for room; a cargo in flight is not moved. |
| FR-STO-019 | `AddAgent` MUST return `ErrNoRoom` and create nothing if the money-and-goods it would create exceed the agent's room. |
| FR-STO-020 | The pledge MUST be derived from the job's pending output and the shipments in flight, MUST be recomputed at every change of either, MUST NOT be saved, and MUST be rebuilt after a restore. |
| FR-STO-021 | The view of a limited agent MUST carry its room, load, pledge, free room, stores and the size of each commodity of its tradable set, and the view of `Work` MUST carry `MaxExecutions` bounded by `k_room` with the reason `no_room` when it is 0. |
| FR-STO-022 | The default strategy MUST scale the targets of its limited agent by `f = min over d of floor(Avail_d * 10 000 / TL_d)` where a dimension binds, `Target' = floor(Target * f / 10 000)`, so that `Keep'` fits `Room - Pledge`. |
| FR-STO-023 | The default strategy MUST size its bids to the free room in the order of its money stack, MUST place no bid that the market would reject for room, and MUST take the draw of every intent it considers whether or not the bid is placed. |
| FR-STO-024 | The default strategy MUST discard, before its asks, only the surplus above `Keep'` that its tier 1 bids need room for, in the order of 6.3.2, as a `Discard` executed as a move to `disposal`, and MUST discard nothing for tier 2 or beyond. |
| FR-STO-025 | A limited role whose base room per household is below one tick's effective upkeep load MUST be rejected by validation (`room_below_upkeep`), and with it the discard of FR-STO-024 MUST always be able to make room for tier 1, provided every commitment leaves the upkeep room free (FR-STO-046). |
| FR-STO-026 | A `MoneyFlow` MAY target stores (`Kind` `"stores"`, optional `Store`, optional `Region`); its amount MUST be per store, only its fixed levy and rate grant shapes MUST be defined (`Level` and `RateBP` are `ErrFlowShape`), a target of an unknown class MUST be `ErrUnknownStoreClass` and a target with the module off `ErrModuleDependency`. |
| FR-STO-027 | The module MUST add no ledger reason, no faucet and no sink of its own: the cost of a store MUST be a flow of `spec-money-supply.md` under its own reason. |
| FR-STO-028 | A lot in a limited account MUST age and wear as in any account; a store MUST NOT change a shelf life or a durability. |
| FR-STO-029 | Storage MUST NOT use the chargeable size or `BulkFactor`: whether a load fits is decided by the pair. |
| FR-STO-030 | With the module on the world MUST register the `region_bound` refusal `has_store`, open for an agent that owns at least one store, with no instance key. |
| FR-STO-031 | A self-carried shipment request of an agent that owns a store MUST be refused with the reason `has_store`, after `traveller_away`. |
| FR-STO-032 | `AddStore` MUST take the class as a `StoreClassID`, MUST act between ticks, atomically, and MUST return `ErrModuleOff`, `ErrUnknownAgent`, `ErrUnknownStoreClass`, `ErrStoreLocked`, `ErrNotLimited`, `ErrAgentAway` (the agent is away on a trip or on a migration journey) or `ErrOutOfBounds` for the cases of 3.6.2, changing nothing; on success MUST emit `StoreChanged` at the call. |
| FR-STO-033 | `RemoveStore` MUST take the class as a `StoreClassID`, MUST return `ErrStoreInUse` (the stores of the class would fall below the installed store facilities), MUST return `ErrNoRoom` and change nothing if, without the store, the account's load plus pledge would exceed the room or its pledge plus its upkeep room would, and `ErrOutOfBounds` if the agent owns none of the class; on success MUST emit `StoreChanged`. It MUST NOT be refused for an agent that is away. |
| FR-STO-034 | An agent created in a role MUST be given the stores of its role's row, in class order, at its creation and with no `StoreChanged`, except a class gated by `Needs` that the region lacks (FR-STO-044). A role change to a role with a base room MUST NOT take them; a role change to a role with no base room MUST end them, in class order, each with a `StoreChanged` of `Kind` `removed` and `Cause` `role_changed` after the `RoleChanged`, and MUST NOT be refused. |
| FR-STO-035 | `StoreChanged` MUST carry the agent, region, role, class (`StoreClassID`), the kind `added` or `removed`, the cause (`consumer` for a call, `role_changed`, `facility`), `Stores` (the count of the class after) and the account's room and load after, MUST be emitted at the call or at P7.1, and MUST contain no map. |
| FR-STO-036 | `RoomLimited` MUST be emitted at P3.6a once per region and role, in ascending region then role, only when one of `StartsCut`, `StartsRefused`, `BidsCut`, `BidsRejected`, `ShipmentsRefused` or `Discarded` is non-zero, with `LimitedHouseholds` and `FullHouseholds` riding along, bounded by regions times roles, and MUST contain no map. No counter of it MUST survive a P8.6. |
| FR-STO-037 | The module MUST add no tick position, no random draw and no row of the draw table, and the number of draws in a tick MUST be unchanged by the module being on **while no scaling, cut or discard acts**; once one does the number MAY differ (11.2). |
| FR-STO-038 | Validation MUST reject, each with its own failing case and sentinel, the rules of 13.1, and MUST warn, without failing, the rules of 13.2. |
| FR-STO-039 | The saved state MUST be the stores of each agent, once, in the agent record, in class order, with no section of its own; a restore MUST reject a class index out of range, more than `MaxStoresPerAgent` stores, a store on an agent of an unlimited role, and a limited account whose load exceeds its room, with `ErrCheckpointCorrupt`; it MUST NOT reject a pledge above the room. The pledge MUST be rebuilt from the jobs and the shipments with no room test (`SetPledge` is untested, 4.1.2). |
| FR-STO-040 | With `WithAudit` and the module on, the audit MUST check, for every limited account, that the load equals the sum of its lots, that load plus reserved is within the room at the end of every position and load plus pledge plus reserved is within it for an account that committed in that position, that reserved is 0 at a boundary and that the pledge equals what its job and shipments imply, and MUST name the first that fails. |
| FR-STO-041 | The module MUST NOT log above `Debug` inside a tick and MUST NOT construct an event or a counter that exists only for an event when no observer was supplied. |
| FR-STO-042 | The stored load of an account MUST be `min(capacity of its stores, max(0, load - w * base room))` per dimension, a read with no attribution to commodities. |
| FR-STO-043 | A restored world MUST continue bit-identically to an uninterrupted one with the module on, including its stores, loads and pledges. |
| FR-STO-044 | A store class with a `Needs` MUST be given, by `AddStore` and at the creation of an agent with the class in its role row, only if `regionHas(region, Needs)` holds at that moment; `AddStore` MUST return `ErrStoreLocked` and a creation MUST give no such store and report it in `StoresLocked` of that agent's `AgentSpawned`; a store already owned MUST NOT be taken when the region's class changes; a non-empty `Needs` MUST be `ErrModuleDependency` with the settlement module off and `ErrUnknownSettlementFlag` for an unknown flag. |
| FR-STO-045 | A job in progress made at construction by a warm start MUST have its pending output pledged, and construction MUST fail with `ErrRoomShape` (rule `room_below_warm_job`) if its group's stock load plus the load of that output is above the base room plus the given stores of the row (13.1.1: a gated store the region lacks is not counted), or if that output's load plus the upkeep room is, per household and for the largest weight of any group of the role; no `ErrNoRoom` of the ledger MUST reach construction. |
| FR-STO-046 | A job start and a dispatch MUST be tested against the free room less the upkeep room (the load of one tick's effective upkeep of the principal, or of the consignee, per dimension), so that after every commitment `Pledge + U` is at most `Room`, and a role with no upkeep MUST have an upkeep room of 0. A bid MUST NOT be tested against it. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-STO-001 | An operation of the ledger on a limited account MUST cost a constant number of integer operations per lot touched and MUST allocate nothing, and on an account that is not limited MUST cost one branch. |
| NFR-STO-002 | With the default logger and observer, a full `Step` of a limited pack MUST allocate no more than the baseline of observability-and-logging.md 8.2 plus nothing for this module. |
| NFR-STO-003 | The bid admission, the start sizing, the target scaling and the discard order SHOULD have benchmarks with `b.ReportAllocs()`. |
| NFR-STO-004 | A run with a limited pack MUST be bit-identical across platforms, with and without an observer and a logger (architecture.md 7.4). |

## 21. Open questions

1. **Preservation.** Should a store slow the ageing of goods in it (a cooled or sealed store)? It
   needs a store to be an account of its own, a rebase of `Born` at the doors and a
   decision to move goods (8.1.3), which is a different design from this one. Decide when
   a pack needs it and a consumer's strategy exists to use it.
2. **A keeper and a lease.** Room for hire needs a shared room, a ration of it, a price
   per unit and tick and a default (3.5.2, 10.4). The path is a contract of MER's kind
   over this room with the pledge as its commitment. Not built because the first consumer
   has no keeper.
3. **A holding charge per unit.** The base of a per-unit fee is the stored load in
   chargeable units, `max(mass, ceil(bulk * BulkFactor / 10 000))` per commodity, summed,
   less the base room's chargeable capacity, capped at the stores'. It needs a new shape
   in `spec-money-supply.md` 4.2, a chargeable measure beside the pair (the over-charge
   of `spec-goods.md` 3.5) and a strategy that reads a cost of carry. Decide with a
   scenario that shows the fixed levy is not enough.
4. **An owner that cannot pay.** Version 1 forgives and the store stays (7.2.3). Whether a
   store decays, is repossessed or is sold belongs to CAP and CRD.
5. **A hard room or a soft one.** The room is a wall (5.1). The alternative is a room
   that may be exceeded at a charge, which makes the cost marginal and the invariant of 4
   conditional. The wall was chosen for the exactness of the pair, the guarantee of
   settlement and the simplicity of a single rule; revisit if a scenario (STO-S3) shows
   pinned markets that a soft room would not have.
6. **A role-level cover.** The default holds `CoverTicks` of its use. The keeper of a buffer store
   is a strategy; whether the default needs a per-role cover for a role whose function
   is to hold (a trader that keeps stock for resale) is AGT's, with N-101.
7. **A regional base room.** Room at home is a property of a place. A base room keyed by
   region would use the override mechanism of `spec-regions.md` 5.1 and a row in its
   list. Not added: a regional difference is a store given in the region.
8. **The pro rata target.** Scaling every target by one factor (6.2.2) is the simplest
   share of a short room. A priority (inputs before buffers, a strategy's choice) would
   change which want is unmet and is a consumer's strategy's to make.
9. **The discard.** `spec-agents.md` 8.2 said the default never discards; with the module
   on it does, for upkeep only (6.3.2). If the rule is unwelcome, the alternative is a
   tier 1 bid that may exceed the room by what it eats in the tick, which costs the
   invariant at P4.1 and a rule in the ledger.
10. **A trip refused for a store.** An owner of a store does not travel (9.4). A portable
    store (a cart that is room as well as carriage) would be a vehicle with a room, and
    belongs to TRN's `Vehicle` with this spec's pledge.
11. **A role change that loses goods.** The excess goes to the estate (5.5.2), which
    sells it to the pool. The alternative, a refusal of the change, costs the spawner and
    the traineeship a refusal path. Measure how often with STO-S1.
12. **Own-use under a room.** An own-use job that is refused for room blocks the
    household (`spec-subsistence.md` 5.5); the household's floor is then as strong as
    its own room. Whether a floor should be exempt from room (a household may always grow
    its own food) is SUB's with this spec's pledge.

13. **A cap on the stores of a class in a region.** `spec-settlements.md` 6.5 offers a
    limit as well as a flag: the most stores of a class a region may hold (one in a town, a
    few in a city). Not offered: the count of stores of a class in a region is a
    scan of the agents, and it is a policy a pack can write as a flag until a case shows
    it needs the count.

14. **A cut of the room under a pledge.** A weight reduction or a role change can leave
    a pledge above the room (5.5.3), and the library cuts it at the credit: output
    disposed, cargo diverted. The alternatives are to scale the pledge with the room (a
    job's pending output would then be cut at the reduction, by a rule PRO would own, and
    a cargo cannot be cut in flight), or to refuse the reduction (impossible: it is the
    ladder's and the population's). Measure how often the cut acts with STO-S6 and STO-S1
    before adding a rule.
15. **A store that ends with a role.** A role change to a role with no base room ends the
    agent's stores (3.6.1a). The alternative is to keep a store that adds no room (an
    account with a store and no limit), which costs a rule at every reader of the room
    and an unlimited account that pays a levy; the spawner's role-change table (SPN 4.2)
    is where a pack would see the loss.

## 22. Where this lives in the code

| Concern | Home |
|---------|------|
| `Storage`, `Capacity`, `RoleRoom`, `StoreClass`, `StoreChanged`, `RoomLimited`, `AddStore`, `RemoveStore` | root package, `storage.go`; the sentinels in `errors.go`; the reasons in `event.go` |
| The bounds `MaxRoom`, `MaxStoresPerAgent`, `MaxStoreClasses`, the validation of 13 | root package, `bounds.go`, `validate.go` |
| The room, load, pledge, reserved room, `SetRoom`, `SetPledge`, `ReserveRoom`, `FreeRoom` and the credit forms | `internal/ledger` (stdlib only; the sizes are a table of primitives) |
| The bound of 5.3.2, the target scaling of 6.2.2, the discard order of 6.3.2 and 5.5.2, the stored load of 8.3.2 | `internal/room` (new: pure rules over primitive slices, stdlib only; architecture.md 2.9). The root converts inline at the call (architecture.md 2.9) |
| The stores of an agent | the agent record of the root package (`agent.go`), saved by `checkpoint.go` |
| The target kind `stores` of a money flow | `moneyflow` of `spec-money-supply.md` (`internal/moneyflow`) with the class table passed in |
| The refusal `has_store` | the registry of `region_bound` refusals of `spec-regions.md` 6.5.2 |

`internal/room` and `storage.go` are added when the module is first built, with the
layout of architecture.md 2.6 and `AGENTS.md` (N-019); the terms of 19.3 go to `doc.go`
first. The model `doc/specs/models/sto/room.py` is not part of the Go module.
