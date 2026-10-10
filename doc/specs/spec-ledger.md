# Spec: The ledger, accounts, lots and conservation

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-LED-NNN` and
`NFR-LED-NNN`, which tests cite (testing.md section 4). They are in section 13.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs `spec-goods.md`. It is the base that clearing, production, agents, the
estate and the checkpoint all stand on.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

The **ledger** holds all the money and goods in the world and is the only place
that changes them. Every other spec asks the ledger to move something and gets
back either success or an error; none holds a balance of its own. This spec
defines:

- the one primitive that changes anything, a transfer between two **accounts**
  with a **reason** (section 2);
- accounts and **boundary** accounts (section 3);
- money (section 4) and goods held as **lots** (section 5);
- reservations that back orders (section 6);
- the registry of reasons and the flows built on it (section 7);
- how production, consumption, spoilage and wear are expressed (section 8);
- conservation, balance sheets and the sectoral identities (section 9);
- claims, left room for credit (section 10);
- the state it contributes to a checkpoint (section 11).

Out of scope: what a recipe is (`spec-production.md`), what an order is
(`spec-clearing.md`), who owns an account and why (the specs that open them:
`spec-agents.md`, `spec-estates.md`, `spec-carriage.md`), the checkpoint's
encoding (`spec-checkpoint.md`) and any money-creating mechanism
(`spec-money-supply.md`).

## 2. Principles

2.1. **One primitive.** The only operation that changes a balance or a quantity
is a **transfer**: an amount of money, or a quantity of one commodity, moves from
one account to another, with a **reason**. Production, consumption, spoilage,
wear, the arrival of an initial endowment, the creation of money by a mint and a
trade are all transfers. Nothing creates or destroys money or goods any other
way (ADR 0001, decision 1). So a conservation test never has to know about a
special case.

2.2. **Double entry, with signed boundaries.** A transfer debits one account and
credits another by the same amount. A **boundary** account stands for everything
outside the economy's conservation (a mint, a sink, the transformation of
recipes, the outside world). A boundary account may go **negative**; any other
account may not. Because every transfer is double-entry, the total over *all*
accounts, boundaries included, is always exactly zero. A faucet is a transfer
*from* a boundary (which goes more negative), and a sink is a transfer *to* one.

2.3. **Every flow has a name.** A transfer carries a reason from a registry (section 7) that is closed for a run: the built-in rows plus one row for each money flow the configuration declares (`spec-money-supply.md` 4.5, 7.2). Each reason says whether it is internal, a faucet or a
sink, and the ledger refuses a transfer that does not fit. So a report of every
source and drain of the economy is a sum over reasons.

2.4. **All state is plain data.** The ledger holds slices of structs, with no
maps in anything that influences results or is saved, no closures and no global
state (architecture.md 7.1, 7.5; ADR 0002, decision 6).

2.5. **Integers, checked.** Money and quantities are `int64`. Every addition and
multiplication is checked, and an operation that would overflow is refused and
changes nothing (architecture.md 3.5).

2.6. **Operations are atomic.** An operation either applies completely or leaves
the ledger unchanged and returns an error. A caller never has to undo half of
something.

2.7. **The ledger knows no setting and no identifiers.** It sees accounts,
commodity indexes, reasons and numbers. It lives in `internal/ledger`
(architecture.md 2.9) and receives the little it needs about commodities
(shelf life, durability) as a table of primitives at construction. Nothing in it
branches on a commodity (architecture.md 9.1).

## 3. Accounts

3.1. An **account** holds money, goods (as lots) and, for some kinds,
reservations. It has a kind, an owner reference and a region. Kinds:

| Kind | Holds | Opened by | May be negative |
|------|-------|-----------|-----------------|
| Agent | an agent's money and goods | the world, when an agent is created | no |
| Estate | a region's estate: the money pool **and** the goods left behind, awaiting sale; the account of the region's estate pseudo-agent (`spec-estates.md` 2.7) | the world, one per region | no |
| Escrow | money held for a delivery contract (`spec-merchants.md` 9.6): one open contract's money and nothing else; its region is the destination of the contract; reused by the next contract and never closed | the delivery contract (`spec-merchants.md` 9.6.2) | no |
| Transit | goods being carried between regions (`spec-transport.md`) | the transport module, when no free Transit account of the origin region exists; reused, not closed after a shipment (`spec-transport.md` 7.1) | no |
| Outside | the money and goods of a counterparty of the external sector (`spec-external-sector.md` 4.2): topped up from a boundary before admission and swept to zero after the markets clear; its region is the gate region; never closed, never relocated | the world, one per counterparty at construction, after the estate accounts and before any agent account | no |
| Boundary | the net amount created or destroyed under one reason | the world, one per reason and kind of thing | **yes** |

A kind can be added by a later spec without changing the ledger's rules: it is
one more entry in the table above, internal (not a boundary) unless it is a
boundary.

3.2. An account has an **ID**. IDs are assigned in order of creation, starting at
0, and are never reused in a run (architecture.md 7.5: iteration and tie-breaks
use the ID). The ledger iterates over accounts in ID order, never over a map.

3.3. An account is **closed** when its owner ceases to exist. A closed account
must have no money, no goods and no reservations: the owner's holdings are moved
out by transfers first (an agent's go to the estate, in the order of
`spec-agents.md` 3.6). A closed
account's slot stays so that IDs stay stable; its storage is reclaimed when a
checkpoint is compacted (open question 6).

3.4. Boundary accounts are one per **reason** and one per kind of thing (money,
or each commodity). The balance of a boundary account is therefore *the
cumulative net amount of that reason that has entered or left the economy*: how
much money a mint has created, how many units have spoiled. Reading the boundary
balances is how a report finds every faucet and sink. The boundary accounts of the reasons that configured money flows declare are created at construction, in flow-ID order, after those of the built-in reasons and before any other account (`spec-money-supply.md` 4.5.1).

3.5. An account has a **region**, except a boundary, which has none. Flows are
attributed to the region of the non-boundary side; an internal flow between two non-boundary accounts (`transit_out`, `transit_in`, `trade`, ...) is attributed to the region of the debited account (7.4). A region can change only by
an explicit operation (`relocate`, `spec-regions.md` 6), never as a side effect. The estate account of each region is created by `spec-regions.md` 3.6, in region order, before any agent account.

## 4. Money

4.1. A balance is an `int64` count of minor units. A non-boundary balance is
between 0 and `MaxMoney` (`spec-config.md` 4.1).

4.2. `Transfer(from, to, amount, reason)` moves `amount` of money, with
`amount >= 1`. It is refused with:

- `ErrInsufficientFunds`, if `from` is not a boundary and does not have `amount`
  **available** (balance minus reserved, section 6);
- `ErrBalanceOverflow`, if the result in `to` would exceed `MaxMoney`;
- `ErrReasonMismatch`, if `from` and `to` do not fit the reason's kind (7.2);
- `ErrUnknownAccount` or `ErrClosedAccount`;
- `ErrInvalidAmount`, if `amount < 1`.

4.3. A transfer may be marked **from reserved**, in which case it draws on an
existing reservation first (section 6.3). Nothing else may touch reserved money.

4.4. Money has no identity: a balance is a number. It has no age and does not
spoil.

## 5. Goods and lots

### 5.1 Lots

5.1.1. Goods in an account are held as **lots**. A lot has a commodity, a
quantity, a **born** tick and a **remaining durability**:

```go
// internal/ledger: stdlib only, no root types.
type Lot struct {
	Commodity  int   // opaque index
	Quantity   int64 // at least 1
	Born       int64 // tick of creation, signed; 0 for a commodity that does not spoil; negative only for a lot created at construction (5.2.4)
	Durability int64 // remaining; 0 for a commodity that does not wear
	Reserved   int64 // units earmarked by a reservation (6.4); 0 between ticks
}
```

The first four fields are the lot's **key** (with `Quantity` the amount).
`Reserved` is at most `Quantity` and is not part of the key. `Born` is a signed
tick: a lot made before tick 0 by a warm start (5.2.4) has a negative one. Every
rule below (age, canonical order, ageing, the hint) works unchanged on a negative
`Born`.

5.1.2. The **age** of a lot at the end of tick `now` is `now - Born + 1`, which
gives the rule of `spec-goods.md` 4.2: a lot made in tick `t` with shelf life `S`
has age `S` at the end of tick `t + S - 1`, and spoils then. Storing the tick of
creation, not a counter that every lot must increment, means a tick does not
touch a lot unless it expires.

5.1.3. **Canonical form.** In an account, lots with the same key (commodity,
born, durability) are merged into one, their quantities and earmarks added; a lot with quantity 0 does not exist; lots are
sorted by (commodity, born, durability). So equal holdings always have equal
representations, whatever sequence of operations made them. This is what lets a
checkpoint have equal bytes for equal states (ADR 0002, decision 5). For a
commodity that neither spoils nor wears, `Born` and `Durability` are 0, so its
holdings are a single lot.

5.1.4. A **durable** (`spec-goods.md` 5) is held as lots of items that share a
remaining durability. Items made new have the commodity's `Durability`. A lot of
quantity `n` and durability `d` is `n` interchangeable items each with `d` left,
which is the same as `n` items of quantity one (`spec-goods.md` 5.2) at a
fraction of the cost to store.

### 5.2 Moving goods

5.2.1. `Move(from, to, commodity, quantity, reason)` moves goods. It takes lots
from `from` in canonical order: **oldest first** (lowest `Born`), then lowest
remaining durability. So a seller offers its oldest stock first
(`spec-goods.md` 4.3) and a durable holder uses its most worn item first. The
moved lots keep their `Born` and durability in `to`: age and wear travel with the
goods, in a market, in transit and in an estate (`spec-goods.md` 4.4).

5.2.2. A move is refused with `ErrInsufficientStock` (not enough available),
`ErrBalanceOverflow` (the quantity in `to` would exceed `MaxQuantity`),
`ErrReasonMismatch`, `ErrUnknownAccount`, `ErrClosedAccount` or `ErrInvalidAmount`.
It may be marked from reserved (6.3).

5.2.3. **Creation.** Goods that enter the economy are a move *from* a boundary
account. The new lot's `Born` is the current tick if the commodity spoils, and its
`Durability` is the commodity's full durability if it wears. The boundary account
goes more negative by the quantity.

5.2.4. **Construction.** The world builds the ledger in a **construction phase**
that ends when the first `Step` begins; in it the current tick is 0. Only in this
phase may a creation (a move from a boundary under `initial`) state the lot key
instead of taking 5.2.3's defaults, so that a warm start (`spec-viability.md` 8.4.3,
8.6) can create:

- a lot of a spoiling commodity with a negative `Born`, from `-(ShelfLife - 1)` to
  0, so that it does not expire before tick 0 and the lots of a group do not
  expire together;
- items of a durable with a remaining durability from 1 to the full durability.

A stated key outside these ranges is refused with `ErrInvalidLot`, and a stated key
outside construction with `ErrNotConstruction`. The target may be any non-boundary
account, an estate account included. A creation that does not state a key behaves as
in 5.2.3 at any time (an external agent added during a run receives its money and
goods by `initial` with `Born` the current tick, `spec-agents.md` 3.5.4). The same
phase lets the world build a job in progress (`spec-viability.md` 8.5.3): grant the
job's inputs by `initial`, take them at once by `recipe_input` with a `Transform`
(8.1), and wear the tools with `WearItems` (5.4.3). Nothing else in the ledger
differs between construction and a run, and every operation keeps 2.6.

### 5.3 Ageing

5.3.1. At the end of each tick the ledger runs **ageing**: every lot whose
`Born + ShelfLife - 1` is at most the current tick is moved whole to the
`spoilage` boundary. This is a sink, with reason `spoilage`. It is deterministic,
uses no random numbers, and processes accounts in ID order and lots in canonical
order.

5.3.2. The cost of ageing must not grow with the number of live lots. The ledger
keeps, for each account, the earliest tick at which any of its lots expires, as a
**derived hint**: an account is visited only when that tick is reached. The hint
is recomputed from the lots after a restore and is not saved. A lot with a negative
`Born` (5.2.4) is counted like any other.

5.3.3. **The query `Expiring(account, commodity, tick)`.** It is read-only and
returns the number of units of the commodity in the account that are not earmarked
and that spoil by the end of `tick`: the lots with `Born + ShelfLife - 1` at most
`tick`. For the current tick these are the units that the ageing of 5.3.1 will move
to the `spoilage` boundary at the end of this tick, so a strategy can offer them
while they still have a value (`spec-agents.md` 7.3.5). It returns 0 for a commodity
that does not spoil and for a `tick` before the current one (the ageing has already
run), and it fails with `ErrUnknownAccount` or `ErrClosedAccount`. It uses the
earliest-expiry hint of 5.3.2: when the account's hint is after `tick`, it returns 0
without looking at a lot, and otherwise it reads the account's lots of that commodity
from the oldest, which canonical order puts first, and stops at the first that expires
after `tick`. It allocates nothing and changes nothing, so it affects no result.
Before the first order of a tick, no earmark exists (6.5), so the units not earmarked
are all the units that spoil.

### 5.4 Wear

5.4.1. `Wear(account, commodity, units)` reduces the remaining durability of items
of a durable. It applies to **the most worn item first** (lowest remaining
durability), one item at a time. An item whose durability reaches 0 or below is
destroyed: it is moved to the `wear` boundary, a sink of quantity 1. Wear units
beyond the destruction of one item are not carried to the next. This matches
`spec-skill-barriers.md` 2.1 and 2.7.

5.4.2. Wear changes no quantity until an item is destroyed, so it is not a
transfer. It is a change of the state of an item and is counted in the flow totals
as **wear units** by commodity (7.4), so a depreciation measure is possible
(`spec-indicators.md`). The units counted are the durability actually removed, so an
item destroyed with less durability left than the wear asked for counts what it had.

5.4.3. **`WearItems(account, commodity, n, units)`** wears `n` items at once, for a
cohort of `n` executions each using one tool (`spec-production.md` 4.4.2, N-033). It
applies to **the `n` most worn distinct items** (lowest remaining durability first,
then canonical order), selected **before any is worn**, so no item is worn twice in
one call. Each selected item loses `units`; an item whose durability reaches 0 or
below is destroyed to the `wear` boundary as in 5.4.1, and excess wear is not carried.
Items that survive are re-merged into the lots of their new durability (5.1.3).
`Wear(account, commodity, units)` is `WearItems(account, commodity, 1, units)`.

5.4.4. Both operations act on **available** items only, so an item earmarked for an
order is never worn (6.3). `WearItems` is atomic (2.6): it is refused, and changes
nothing, with `ErrInsufficientStock` if the account has fewer than `n` available
items of the commodity, `ErrInvalidAmount` if `n` or `units` is below 1,
`ErrUnknownAccount` or `ErrClosedAccount`. It is not a transfer except for the
destruction of an item, which is a move of quantity 1 to `wear`.

## 6. Reservations

A **reservation** earmarks part of an account's money or goods so that an order
placed now can be settled later, with no chance of failing for lack of funds
(`spec-clearing.md` 3.3).

6.1. `ReserveMoney(account, amount)` and `ReserveGoods(account, commodity,
quantity)` fail with `ErrInsufficientFunds` or `ErrInsufficientStock` if more than
the **available** amount is asked. Available is the balance minus the amount
reserved. A boundary account cannot reserve (`ErrBoundaryReserve`).

6.2. `ReleaseMoney` and `ReleaseGoods` return an amount to available. Releasing
more than is reserved is refused with `ErrReservationExceeds`.

6.3. A transfer or move marked from reserved draws on the account's reservation
for that money or commodity, reducing both the reservation and the balance. It
fails if the reservation is less than the amount. An unmarked transfer can use
only available money and stock and so can never spend an earmark.

6.4. **Representation.** A money reservation is a counter on the account. A goods
reservation is an **earmark**: the units reserved are marked inside the account's
lots (`Reserved`, 5.1.1), taken in canonical order, oldest first, so the units that
a later move will take are the ones earmarked. An unmarked move takes only
unearmarked units, in canonical order, so it can never take an earmarked unit
(6.3). The market needs one reservation per agent per commodity per tick, plus
one per named lot for the estate (`spec-clearing.md` 3.1), so nothing more is
needed than these marks.

6.5. Reservations exist only inside a tick. By the end of a tick, every market
reservation has been settled or released. **At a tick boundary the ledger holds no
reservation**, and so a checkpoint, which is taken between ticks (ADR 0002, open
point 4), saves none.

6.6. **Escrow is not a reservation.** Money held for a contract must outlive a
tick and be a claim of the contract, not an earmark of the payer. It is a transfer
into an Escrow account, and a transfer out when the contract settles
(`spec-merchants.md` 9.6; `spec-carriage.md` 2.1).

6.7. **Named lots.** `ReserveGoods`, `ReleaseGoods` and `Move`, with or without
from reserved, accept an optional lot key (commodity, born, durability). With a
key they act on exactly that lot, and fail with `ErrInsufficientStock` if it has
too few available units or `ErrUnknownLot` if there is no such lot. Without one
they act in canonical order. Named operations exist for the estate, which prices
each lot separately (`spec-estates.md` 2.7.3); an ordinary agent never needs
them.

## 7. Reasons and flows

### 7.1 The registry

A **reason** is a stable name from a closed list. Each has a **kind**:

- **internal**: both accounts are non-boundary (a trade, a relief payment);
- **faucet**: from a boundary into the economy;
- **sink**: from the economy into a boundary.

7.2. The ledger refuses a transfer whose accounts do not fit the reason's kind
(`ErrReasonMismatch`). A reason is declared once, in one place, with its kind and
the spec that owns it; a spec that needs a new flow adds a row, as a spec adds a
parameter. The starting registry:

| Reason | Of | Kind | Between | Owner |
|--------|----|------|---------|-------|
| `trade` | money, goods | internal | agent, estate or Outside account to agent, estate or Outside account | `spec-clearing.md` |
| `initial` | money | faucet | boundary to agent or estate | `spec-config.md`, `spec-world.md`; `spec-agents.md` 3.5; `spec-viability.md` 8.3 |
| `initial` | goods | faucet | boundary to agent or estate; a stated lot key only at construction (5.2.4) | `spec-agents.md` 3.5; `spec-viability.md` 8.4, 8.5.3, 8.6 |
| `recipe_input` | goods | sink | agent to boundary | `spec-production.md` |
| `recipe_output` | goods | faucet | boundary to agent | `spec-production.md` |
| `disposal` | goods | sink | agent to boundary | `spec-production.md` 7.3 (discard outputs, overflow); available to `spec-agents.md` 8 for a discard |
| `consumption` | goods | sink | agent to boundary | `spec-agents.md` |
| `want_consumption` | goods | sink | agent to boundary | `spec-demand.md` 4, 12 (consumption of a want beyond upkeep; separate from `consumption` so that the two are told apart in `BoundaryFlow`) |
| `spoilage` | goods | sink | any holder to boundary | `spec-goods.md` |
| `wear` | goods | sink | agent to boundary | `spec-goods.md` |
| `estate_money` | money | internal | agent to estate | `spec-estates.md` |
| `estate_goods` | goods | internal | agent to estate | `spec-estates.md` |
| `start_money` | money | internal | estate to agent | `spec-estates.md` |
| `relief` | money | internal | estate to agent | `spec-estates.md` |
| `outside_budget` | money | faucet | boundary to an Outside account (a buyer counterparty) | `spec-external-sector.md` 8.1 |
| `outside_unspent` | money | sink | Outside account (buyer) to boundary | `spec-external-sector.md` 8.1 |
| `outside_receipts` | money | sink | Outside account (seller) to boundary | `spec-external-sector.md` 8.1 |
| `outside_offer` | goods | faucet | boundary to an Outside account (a seller counterparty); born in the tick, at full durability | `spec-external-sector.md` 8.1 |
| `outside_unsold` | goods | sink | Outside account (seller) to boundary | `spec-external-sector.md` 8.1 |
| `outside_purchases` | goods | sink | Outside account (buyer) to boundary | `spec-external-sector.md` 8.1 |
| `escrow_in` | money | internal | agent to escrow: the payer's `price * quantity` at the formation of a delivery contract | `spec-merchants.md` 9.6 |
| `escrow_out` | money | internal | escrow to agent: the deliverer, for the units that reached the destination | `spec-merchants.md` 9.6 |
| `escrow_refund` | money | internal | escrow to agent, or escrow to the estate of the destination region when the payer no longer exists | `spec-merchants.md` 9.6 |
| `transit_out`, `transit_in` | goods | internal | agent to transit; transit to agent, or transit to estate (a diverted cargo) | `spec-transport.md` 7.2 |
| `haul_use` | goods | sink | agent to boundary: the redemption of haul units at dispatch | `spec-transport.md` 7.2 |
| `transit_use` | goods | sink | agent to boundary: the feed consumed by a trip | `spec-transport.md` 7.2 |
| `loss` | goods, money | sink | transit, agent or estate to boundary | RSK (`spec-risk.md`, later); no flow is posted under it in version 1 |
| a flow ID | money | faucet (a grant) or sink (a levy) | boundary to agent or estate; agent or estate to boundary | `spec-money-supply.md` 4.5: one row per configured money flow, reason = flow ID, declared at construction |
| `split`, `merge`, `births`, `starvation` | money, goods | internal | agent to estate, or back | `spec-scale.md` 4.8, `spec-agents.md` 3.7, `spec-spawner.md` 4.1, 4.3 (`starvation`: a reduction by the failure ladder; `births`: households added to an existing agent, money from the pool and no goods), `spec-population.md` 4.3 (every household born through the module, joining an existing agent or becoming a new one: money only, estate to agent) |
| `deaths` | money, goods | internal | agent to estate | `spec-population.md` 4.7 (a partial reduction by the schedule; an emptied agent is a removal with `estate_money` and `estate_goods`) |

Later specs add rows: money creation and destruction by configured flows (`spec-money-supply.md`: the rows are declared by the configuration, 4.5, 10.3),
taxes (`spec-fiscal.md`), imports and exports (`spec-external-sector.md` 8.1: the six `outside_*` reasons, declared when the external module is on), loans
(`spec-credit.md`). The names are the strings used in events and logs, and are the
values of the `reason` attribute (observability-and-logging.md 3.3.2).

7.2.1. `initial` has no limit in time: it creates the endowment at construction and
the money and goods of an external agent added during a run (`spec-agents.md`
3.5.4). Only the stated lot key is construction-only. `disposal` is a sink like
`recipe_input`, so a discard output is created by `recipe_output` and moved out
by `disposal` inside the one atomic completion step of the job (`spec-production.md`
5.3.1, 7.2); it is never held, and both flows show in the flow totals.

### 7.3 Faucets and sinks are the only exceptions

By 2.1 and 7.2, the only way for the total of non-boundary money, or of a
commodity, to change is a faucet or a sink, and each has a reason. A closed
economy (the zero configuration, ADR 0001 decision 2) has no money faucet or sink
at all after the initial endowment; goods still flow through recipe, consumption
and spoilage.

### 7.4 Flow totals

The ledger keeps, for the current tick, the total moved per (region, reason, money
or commodity) and the wear units per commodity. They are cleared at the start of
each tick, kept as a slice in a fixed order, and read by the observability and
indicator specs (`spec-observability.md`, `spec-indicators.md`). They are
aggregates, so their size is bounded by regions, reasons and commodities, not by
the number of agents. A flow between two non-boundary accounts is attributed to the region of the debited account. Beside them the ledger keeps the **relocation totals** (`spec-regions.md` 6.4): the money and the units moved with accounts that changed region, per ordered pair of regions, with the same lifetime and order; a relocation is not a transfer and is in no flow total.

7.4.1. **The construction bucket.** What is moved during the construction phase
(5.2.4) is totalled in a separate **construction bucket**, with the layout of the
flow totals, and not in the totals of tick 0. So the endowment and the work in
progress of a warm start (`spec-viability.md` 8.6 item 3) do not look like a flow of
the first tick, and a report of tick 0 shows only what tick 0 did. The bucket is
cleared by nothing, is read-only, and is available for the whole run; it is not saved
in a checkpoint (11), because a restored world is not in construction and its boundary
balances already hold the total. The audit includes it (12.4). Whether the world
emits it as a `BoundaryFlow` is for `spec-observability.md` (open question 9).

## 8. Production, consumption, spoilage and wear

8.1. **A recipe execution is a transformation.** `Transform(account, inputs,
outputs)` moves each input out of the account to the `recipe_input` boundary and
each output in from the `recipe_output` boundary, all or nothing. Goods are not
conserved by a recipe in the sense of quantities (flour is not wheat), and the
ledger does not pretend they are: a recipe is a pair of named sinks and faucets, so
the quantity of each commodity is conserved *except for* those flows, which is
exactly the statement of `testing.md` 6.3 ("goods are conserved except by
recipes").

8.2. **A recipe with no inputs is a source.** It is a transform with only outputs
and is how a gathered raw good enters (`spec-goods.md` 6.2).

8.2.1. **A `Transform` may have an empty side, but not both.** An empty input list
makes a source, or the completion of a job; an empty output list makes the start of a
job. A transform with both empty is refused with `ErrEmptyTransform`. This is how
`spec-production.md` uses the ledger: a job is two transforms, a start that takes the
inputs and creates nothing (4.3.1) and a completion that takes nothing and creates
the pending outputs (5.3.1), with no ledger holding between them.

8.2.2. **Work in progress.** The output of a job in progress is in no account: it is
fixed when the job starts and created at completion, so the ledger cannot see it. The
identities of 9.1 hold without it, because a flow is a ledger flow only when it moves.
What a test of "goods are conserved except by recipes" (testing.md 6.3) needs is
therefore the ledger's flows plus the pending output the snapshot reports
(`spec-production.md` 14.3): for each commodity, the units fixed at job starts equal
the units created by `recipe_output` plus the pending units plus the units dropped
from pending, which are never created: those of an abandoned job, and the share lost
when the executor's weight falls (`spec-production.md` 5.4).

8.3. **Consumption** of upkeep is a move to the `consumption` boundary, and the consumption of a want beyond upkeep (`spec-demand.md`) is a move to the `want_consumption` boundary. Any other use of goods that does not make others is a move to `consumption` unless the spec that owns it adds a reason. A role change that leaves goods outside the new role's tradable set moves them to the estate with the existing reason `estate_goods` (`spec-spawner.md` 4.2), so no reason is added for it.

8.4. Output lots are created with `Born` as in 5.2.3, and durable outputs with full
durability. A recipe's inputs are taken oldest first (5.2.1).

8.5. How many times a recipe runs, and what a weight does to it, are
`spec-production.md` and `spec-scale.md` 4.2. The ledger sees only the resulting
quantities.

## 9. Conservation, balance sheets and sectoral identities

### 9.1 The identities

These are true after every operation and are checked by the invariant tests
(testing.md 6.3):

1. **Zero total.** The total of money over all accounts, boundaries included, is
   0; so is the quantity of each commodity.
2. **Non-negativity.** No non-boundary account has a negative balance or quantity;
   reserved never exceeds the balance.
3. **Conservation by reason.** For money, and for each commodity, the change of
   the total over non-boundary accounts in a period equals the sum of its faucets
   minus the sum of its sinks over that period, reason by reason, and equals the
   negative of the change in the boundary accounts.
4. **Canonical form** (5.1.3).
5. **Exactness.** All of the above are asserted with exact equality, never a
   tolerance (testing.md 6.3).

The identities hold at every moment, the construction phase included: after
construction and before the first `Step`, the total of each thing over the
non-boundary accounts equals the faucets minus the sinks of the construction bucket
(7.4.1), reason by reason, and the boundary balances equal the negative of that
total. From the first tick on, the identity of item 3 is checked over each tick as
before, and the boundary balance of each reason equals the construction bucket plus
the totals of every tick since (in a run that began from construction; after a
restore the baseline is the restored balances).

### 9.2 Balance sheets

A **balance sheet** for a non-boundary account is a read-only view:

- assets: its money; its goods, by commodity, each valued at a price supplied by
  the caller (normally the commodity's market price in the account's region,
  `spec-clearing.md` 5.4); the **claims** it holds against others (10);
- liabilities: the claims against it;
- net worth: assets minus liabilities.

A balance sheet is *measurement only*. The valuation uses prices from the caller
and the engine never reads a balance sheet to make a decision (ADR 0001, decision
7). The arithmetic is checked. This is the "double-entry balance sheets per
agent" of the roadmap.

### 9.3 Sectors and the sectoral identity

Each account kind belongs to a **sector**: households (agent accounts), public
(estates), contract (escrow and transit), and boundary. In every tick, the
change in money held by the three non-boundary sectors adds up to the money
created minus the money destroyed by the boundary in that tick:

```
delta(households) + delta(public) + delta(contract) = faucets - sinks
```

and the same for each commodity. This is the stock-flow consistency check: no
sector's gain exists without another's loss or a named boundary flow. It is a
test, run every tick over the reference chain, over a range of seeds
(testing.md 6.2). With the external module on, the Outside accounts of its counterparties form a fourth non-boundary sector, the outside sector, whose money and goods are zero at every tick boundary (`spec-external-sector.md` 4.2.3, 8.4); the identity then holds with that sector at zero and the six `outside_*` reasons among the faucets and sinks.

## 10. Claims (room for credit)

A **claim** is a record of an amount that one account owes another: a creditor, a
debtor, a principal and terms. It is an asset of the creditor and a liability of
the debtor on their balance sheets, and the pair always balances (a loan is a
transfer of money from creditor to debtor *and* a claim). In version 1 nothing
creates a claim: the ledger has the list, empty, and the balance sheet reads it.
`spec-credit.md` defines the terms, interest and default. Keeping the list in the
ledger from the start means the checkpoint and the balance sheets do not change
shape when credit arrives (ADR 0002, "the ledger (accounts, balance sheets,
obligations)").

## 11. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002, consequences). The ledger's:

- the tick counter the lots' ages are measured against;
- the accounts, in ID order: kind, owner reference, region, whether closed,
  money balance, and lots in canonical order (`Born` is signed: a lot made before
  tick 0 by a warm start keeps its negative value);
- the boundary balances: the cumulative net per reason and thing;
- the claims, in a fixed order (empty in version 1).

It does **not** contain: reservations or earmarks (6.5: none exist between ticks,
so `Reserved` is 0 in every saved lot), flow totals
(cleared each tick), the construction bucket (7.4.1), the derived expiry hints
(recomputed on restore), pending output (it is in no account; `spec-production.md`
15 saves it with the job) or the table
of commodity attributes (configuration). All is plain data with no maps, sorted,
so equal states give equal bytes.

## 12. Observability

12.1. **Event.** Every faucet and sink has a name, but one event per transfer would
be far too many. The ledger instead feeds one aggregated event per tick, which is in
the table of observability-and-logging.md 4.3: `BoundaryFlow`, once per tick for
each (region, reason, money or commodity) with a non-zero total, after the tick's
transitions. This spec owns its fields: `Tick`, `Region`, `Reason` (the string name),
the commodity (or none for money), whether it is a faucet or a sink, and the total
quantity. It is emitted in the order of region, reason and commodity, with no map in
it. This is the "faucet-and-sink
event with the reason" of ADR 0001.

12.2. **Logs.** Nothing at `Info`, `Warn` or `Error`. Operation errors are returned
(architecture.md 8.3) and the root turns them into order rejections
(`spec-clearing.md` 3.2) or into a failure of the caller. At `Debug`, guarded by
`Enabled` in hot paths, the ledger may log a refused transfer with `reason`,
account IDs and amounts. A violated internal invariant found outside an enabled
audit (12.4) is a programmer error, so it panics with a message that says so
(architecture.md 8.1).

12.3. **Cost with no observer.** With the default no-op observer the ledger builds
no event and allocates nothing in its operations (NFR-LED-001;
observability-and-logging.md 4.5, 8.2). The flow totals (7.4) are plain integer
additions into preallocated slots, so they cost the same whether or not anyone
listens.

12.4. **Audit (public).** The ledger's `Audit` checks the identities of 9.1 (the
zero totals, non-negativity, reserved at most the balance, canonical form, and
the per-reason and sectoral conservation of 9.3) and reports the first one that
fails, naming the identity and the account or commodity. It is public in two
ways, both on the world:

- `World.Audit() error` runs it once, between ticks, and also on a world that has
  been constructed and not yet stepped, where it checks the construction bucket
  (9.1);
- the option `WithAudit()` runs it at the end of every `Step`, after the tick's
  events, and makes `Step` return an error wrapping `ErrAuditFailed` if it fails. With the money-supply module on it also checks, for each flow, that its `Total` equals the absolute boundary balance of its reason and has the right sign (`spec-money-supply.md` 3.5.3); with the transport module on it runs the checks of `spec-transport.md` 7.4.1; with the merchant module on it also checks that each open contract's Escrow account holds exactly `price * quantity`, that an unreferenced Escrow account is empty, that contract IDs are unique and that the shipment of each contract is in flight (`spec-merchants.md` 9.6, FR-MER-045).

With the external module on the audit also checks that every counterparty account is empty, that each of the six `outside_*` boundary balances has the sign of its kind, and that for each commodity the boundary balance of `outside_purchases` equals the sum of the units traded on its buy sides and `outside_offer` less `outside_unsold` equals the sum of the units traded on its sell sides (`spec-external-sector.md` 8.5, FR-EXT-050). With the resources module on it SHOULD also check the endowment identity of `spec-resources.md` 6.4 (FR-RES-025); an endowment is in no account and the ledger is unchanged by it.

Audit reads and never changes anything, so it affects no result, and the option is
not part of the effective configuration or its hash (`spec-config.md` 8.1). It is
off by default because it costs time proportional to the number of accounts and
lots on every tick. It follows architecture.md 6.2: an option
that validates its input and returns an error rather than panicking. With the option on, a failure is
returned so a consumer can log it and stop; with it off, a violated invariant
still panics (12.2). The invariant tests of testing.md 6.3 use it, and a consumer
who adds a `Strategy`, `Clearing` or `Spawner` can turn it on to check the
extension never leaves the ledger inconsistent.

## 13. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-LED-001 | A balance or a quantity MUST change only through a transfer between two accounts with a reason, `Wear`, or ageing; no other operation MUST create or destroy money or goods. |
| FR-LED-002 | The total of money over all accounts, boundaries included, and the total quantity of each commodity MUST be exactly 0 after every operation. |
| FR-LED-003 | A non-boundary account MUST NOT have a negative money balance or a negative quantity, and a boundary account MAY. |
| FR-LED-004 | Every transfer MUST carry a reason from the registry, and a transfer whose accounts do not fit the reason's kind (internal, faucet, sink) MUST be refused with `ErrReasonMismatch`. |
| FR-LED-005 | A transfer or move of an amount below 1 MUST be refused with `ErrInvalidAmount`. |
| FR-LED-006 | A transfer that would overflow `int64`, or take a non-boundary money balance above `MaxMoney` or a quantity above `MaxQuantity`, MUST be refused with `ErrBalanceOverflow`, and MUST NOT wrap. |
| FR-LED-007 | A refused or failed operation MUST leave the ledger unchanged. |
| FR-LED-008 | Account IDs MUST be assigned in order of creation and MUST NOT be reused in a run, and the ledger MUST iterate accounts in ID order. |
| FR-LED-009 | An account MUST NOT be closed while it holds money, goods or reservations, and an operation naming a closed account MUST be refused with `ErrClosedAccount`. |
| FR-LED-010 | Goods in an account MUST be held as lots of a commodity, a quantity of at least 1, a born tick and a remaining durability, with equal lots merged and ordered by commodity, born and durability. |
| FR-LED-011 | Equal holdings MUST have an identical representation whatever sequence of operations produced them. |
| FR-LED-012 | A move MUST take lots oldest first and then lowest remaining durability, and the moved lots MUST keep their born tick and remaining durability. |
| FR-LED-013 | Goods entering from a boundary MUST be born in the current tick if they spoil, and a durable MUST enter with its full durability, except for a creation at construction that states its lot key (FR-LED-043). |
| FR-LED-014 | At the end of each tick, every lot whose born tick plus shelf life minus 1 is at most that tick MUST be moved whole to the `spoilage` boundary, in a deterministic order and with no random numbers. |
| FR-LED-015 | `Wear` MUST reduce the remaining durability of the most worn item first, MUST destroy an item whose durability reaches 0 or below to the `wear` boundary, and MUST NOT carry excess wear to another item. |
| FR-LED-016 | Wear units applied MUST be counted by commodity in the flow totals. |
| FR-LED-017 | A reservation of money or goods MUST NOT exceed the available amount, and a boundary account MUST NOT reserve (`ErrBoundaryReserve`). |
| FR-LED-018 | A release MUST NOT exceed the reservation (`ErrReservationExceeds`). |
| FR-LED-019 | A transfer marked from reserved MUST draw on the reservation and fail if it is too small, and an unmarked transfer MUST NOT spend reserved money or goods. |
| FR-LED-020 | At every tick boundary the ledger MUST hold no market reservation. |
| FR-LED-021 | `Transform` MUST consume its inputs to the `recipe_input` boundary and produce its outputs from the `recipe_output` boundary all or nothing. |
| FR-LED-022 | Money held for a contract MUST be held in an Escrow account by transfers, and MUST NOT be a reservation. |
| FR-LED-023 | The ledger MUST keep, for the current tick, the total moved per region, reason and money or commodity, in a deterministic order. |
| FR-LED-024 | For money and for each commodity, the change of the total over non-boundary accounts in a tick MUST equal the faucets minus the sinks of that tick, reason by reason. |
| FR-LED-025 | In every tick, the change of the money held by the households, public and contract sectors MUST sum to the faucets minus the sinks of the boundary, and likewise for each commodity. |
| FR-LED-026 | A balance sheet MUST give a non-boundary account's money, goods valued at supplied prices, claims held and owed and net worth, using checked arithmetic, and the engine MUST NOT read it to make a decision. |
| FR-LED-027 | A change of an agent's weight MUST be applied by transfers whose reasons are `split`, `merge`, `births` or `starvation`, scaling every holding in proportion with the remainder going to the counterparty account, and conserving money and goods exactly (`FR-SCL-014`). |
| FR-LED-028 | The ledger MUST hold a list of claims, empty in version 1, and a balance sheet MUST include it. |
| FR-LED-029 | The ledger's saved state MUST be plain data in the order of section 11, with no maps, and MUST NOT contain reservations, flow totals, the construction bucket or expiry hints. |
| FR-LED-030 | After a restore, the derived expiry hints MUST be recomputed, and the run MUST continue exactly as an uninterrupted one. |
| FR-LED-031 | `BoundaryFlow` MUST be emitted once per tick for each region, reason and money or commodity with a non-zero total, in the order of region, reason and commodity, and MUST contain no map. |
| FR-LED-032 | The ledger MUST NOT log above `Debug`, and a violated internal invariant found when audit is not enabled MUST panic with a message that says it is a programmer error. |
| FR-LED-033 | Reserving, releasing and moving goods MUST accept an optional lot key and, with one, MUST act on exactly that lot, failing with `ErrInsufficientStock` or `ErrUnknownLot`. |
| FR-LED-034 | A goods reservation MUST earmark units in canonical order (or in the named lot), an unmarked move MUST take only unearmarked units, and a move from reserved MUST consume earmarked units. |
| FR-LED-035 | `World.Audit` MUST check every identity of section 9.1 and return an error naming the first that fails, and `WithAudit` MUST run it at the end of every `Step` and make `Step` return an error wrapping `ErrAuditFailed` on failure. |
| FR-LED-036 | Audit MUST NOT change any state, and a run with audit enabled MUST give results identical to one without, and the same configuration hash. |
| FR-LED-037 | The reason registry MUST contain `disposal` as a goods sink from an agent to a boundary, and a transfer under it from a boundary, or to a non-boundary account, MUST be refused with `ErrReasonMismatch`. |
| FR-LED-038 | The reason `initial` MUST be accepted for goods as well as money, from a boundary into any non-boundary account, an Estate account included, and MUST be a faucet. |
| FR-LED-039 | A `Transform` with an empty input list or an empty output list MUST be accepted and applied all or nothing, and one with both empty MUST be refused with `ErrEmptyTransform`. |
| FR-LED-040 | `WearItems(account, commodity, n, units)` MUST select the `n` most worn distinct available items before wearing any, MUST reduce each by `units`, MUST destroy to the `wear` boundary each item whose durability reaches 0 or below without carrying excess wear, and MUST merge the surviving items into the lots of their new durability. |
| FR-LED-041 | `WearItems` MUST be refused with `ErrInsufficientStock` when fewer than `n` available items exist and with `ErrInvalidAmount` when `n` or `units` is below 1, MUST leave the ledger unchanged when refused, and MUST NOT wear an earmarked item. |
| FR-LED-042 | `Wear(account, commodity, units)` MUST give the same result as `WearItems(account, commodity, 1, units)`, and both MUST count in the flow totals the durability actually removed, by commodity. |
| FR-LED-043 | During construction only, a creation under `initial` MAY state its lot key: a `Born` from `-(ShelfLife - 1)` to 0 for a spoiling commodity and a remaining durability from 1 to the full durability for a durable; a key outside these ranges MUST be refused with `ErrInvalidLot` and a stated key outside construction with `ErrNotConstruction`. |
| FR-LED-044 | A lot with a negative `Born` MUST follow the rules of ageing, age, canonical order, the expiry hint and the checkpoint exactly as any other lot, and MUST spoil at the end of the tick `Born + ShelfLife - 1`. |
| FR-LED-045 | What is moved during construction MUST be totalled in a separate construction bucket and not in the flow totals of tick 0, and the bucket MUST stay readable for the run. |
| FR-LED-046 | The audit MUST include the construction bucket: after construction and before the first `Step`, the total of each thing over the non-boundary accounts MUST equal the faucets minus the sinks of the bucket, reason by reason, with exact equality. |
| FR-LED-047 | The output of a job in progress MUST be in no account, and every identity of section 9.1 MUST hold with jobs in progress, including at tick 0 of a world with a warm start. |
| FR-LED-048 | `Expiring(account, commodity, tick)` MUST return the number of non-earmarked units of the commodity in the account whose lot has `Born + ShelfLife - 1` at most `tick`, MUST return 0 for a commodity that does not spoil and for a tick before the current one, and MUST change nothing. |
| FR-LED-049 | `Expiring` MUST return 0 without reading a lot when the account's earliest-expiry hint is after `tick`, MUST fail with `ErrUnknownAccount` or `ErrClosedAccount` for an unknown or closed account, and the sum of its results over the commodities for the current tick MUST equal what the ageing of the same tick moves to `spoilage` from that account, when nothing is earmarked. |
| FR-LED-050 | The registry MUST contain the goods sink `want_consumption`, agent to boundary, owner `spec-demand.md`, and a ledger with the demand module off MUST show no flow under it (`spec-demand.md` FR-DEM-001, FR-DEM-039). |
| FR-LED-051 | A weight reduction by the failure ladder MUST be applied with the reason `starvation`, a reduction by the population module with the reason `deaths`, and an entry of households into an existing agent with the reason `births`, each as transfers between the agent and the estate account that conserve money and goods exactly (`spec-spawner.md` 4.1, 4.3; FR-LED-027). |
| FR-LED-052 | The reason registry MUST be closed for a run and consist of the built-in rows plus one row for each configured money flow (reason equal to the flow ID, kind faucet for a grant and sink for a levy, money only), and the boundary accounts of the declared reasons MUST be created at construction in flow-ID order after those of the built-in reasons and before any other account (`spec-money-supply.md` FR-MON-003). |
| FR-LED-053 | A flow between two non-boundary accounts MUST be attributed in the flow totals to the region of the debited account, and a flow with a boundary to the region of the non-boundary side. |
| FR-LED-054 | The ledger MUST keep, for the current tick, the relocation totals of `spec-regions.md` 6.4 (money, and units per commodity, per ordered pair of regions) beside the flow totals, cleared each tick, not saved; a relocation MUST involve no transfer, reason, boundary account or flow total, and MUST leave the total of money and of each commodity unchanged (`spec-regions.md` FR-REG-040, FR-REG-042). |
| FR-LED-055 | The registry MUST contain `haul_use` and `transit_use` as goods sinks from an agent to a boundary, `transit_in` MUST also be accepted from a Transit account to an Estate account, `deaths` MUST be accepted as an internal transfer of money and goods from an agent to an estate account, and `births` as an internal transfer of money from an estate to an agent account; each other pairing MUST be refused with `ErrReasonMismatch` (`spec-transport.md` 7.2, `spec-population.md` 4.3, 4.7). |
| FR-LED-056 | With a money-supply module on, the audit MUST also check, for each flow, that its total equals the absolute boundary balance of its reason with the right sign (`spec-money-supply.md` FR-MON-026, FR-MON-027). |
| FR-LED-057 | With the external module on, the registry MUST also contain the six reasons `outside_budget` (money faucet), `outside_unspent` and `outside_receipts` (money sinks), `outside_offer` (goods faucet), `outside_unsold` and `outside_purchases` (goods sinks), declared at construction and closed for the run, and `trade` MUST be allowed between Agent, Estate and Outside accounts (`spec-external-sector.md` 8.1, FR-EXT-017). |
| FR-LED-058 | An Outside account MUST be a non-boundary account of the gate region that may not go negative, MUST be created at construction after the estate accounts and before any agent account, MUST NOT be closed or relocated, and MUST hold no money and no goods at any tick boundary (`spec-external-sector.md` 4.2.3, FR-EXT-012). |
| FR-LED-059 | With the external module on, the audit MUST also check the conditions of `spec-external-sector.md` 8.5 and report the node and commodity of the first that fails (FR-EXT-050). |
| FR-LED-060 | The reasons `escrow_in`, `escrow_out` and `escrow_refund` MUST be internal transfers of money only: `escrow_in` from an Agent account to an Escrow account, `escrow_out` from an Escrow account to an Agent account, and `escrow_refund` from an Escrow account to an Agent account or to an Estate account; each other pairing MUST be refused with `ErrReasonMismatch`, and a contract MUST add no faucet, no sink and no reason (`spec-merchants.md` 9.6.1). |
| FR-LED-061 | With the merchant module on, the audit MUST also check the conditions of `spec-merchants.md` 9.6 (each open contract's Escrow account holds exactly `price * quantity`, an unreferenced Escrow account is empty, contract IDs are unique, and the shipment of each contract is in flight) and report the contract of the first that fails (FR-MER-045). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-LED-001 | With the default logger and observer, ledger operations in a step of the reference chain MUST allocate nothing beyond scratch held by the world and reused across ticks. |
| NFR-LED-002 | The cost of ageing in a tick SHOULD be proportional to the number of accounts with an expiring lot plus the lots that expire, not to the number of lots. |
| NFR-LED-003 | The ledger SHOULD have benchmarks of a transfer, a move, ageing and a transform, with `b.ReportAllocs()`. |
| NFR-LED-004 | `Expiring` MUST NOT allocate, and `WearItems` and `Expiring` SHOULD have benchmarks with `b.ReportAllocs()`. |

## 14. Entries for the stability register

Merged into `spec-stability.md` section 4: hoarding by spoilage is STB-L25, money
that cannot leave is STB-L30, wealth stuck in an account is STB-L31, and no negative
balances is STB-L32. The text below is the source and is kept for reference; the
register is the home of the entries.

For `spec-stability.md`:

- **Hoarding by spoilage.** Spoilage is a sink of goods. A producer that cannot sell
  loses stock to the `spoilage` boundary, which is a damper on overproduction and
  a destabiliser if the sink is large: a glut that spoils is wealth destroyed,
  which then raises the next price. Dampers: oldest-first selling and the shelf
  life calibration (`spec-goods.md` section 10).
- **Money that cannot leave.** With no money sink, a faucet's cumulative total
  grows with no bound. The boundary balances make this visible per reason
  (3.4), which is the measure `spec-money-supply.md` needs to hold a faucet
  against a sink.
- **Wealth stuck in an account.** The estate pool and escrow hold money out of
  circulation. The pool's release rate (`spec-estates.md` `PoolRelease`) is the
  damper; the balance sheets and sector totals (9.3) show how much is held.
- **No negative balances is a constraint, not a stabiliser.** The ledger enforces
  non-negativity by refusing an operation, so a shortage shows as a refused order
  and a starving agent, never as a negative balance. The floor under that is
  `spec-subsistence.md` and relief.

## 15. Open questions

1. **Age or born.** Storing `Born` and computing age (5.1.2) is cheap, but a lot
   of a non-perishable commodity loses its age entirely (set to 0 so lots merge).
   If a measure of the age of stock is ever wanted for such a commodity, the
   canonical form must change. Recommend keeping it.
2. ~~The estate as a seller.~~ **Settled:** the estate is a pseudo-agent with one
   Estate account holding both its pool and its goods, placing one ask per lot
   (`spec-estates.md` 2.7, `spec-clearing.md` 3.1.2, section 6.7 here).
3. ~~A public audit option.~~ **Settled:** `World.Audit()` and `WithAudit()`
   (12.4). Open point: whether the audit should also be able to run a subset of
   the checks, to cost less on a very large world.
4. **Valuation of goods on a balance sheet.** The market price is used here. A
   cost basis (what the holder paid) needs per-lot cost, which this design does
   not keep. Do not add it unless an indicator needs it.
5. **Event volume.** One aggregated event per (region, reason, commodity) per tick
   is bounded by the configuration. A per-agent flow event is not provided; the
   Debug log and the balance sheets give agent-level detail.
6. **Closed accounts and checkpoint growth.** IDs are never reused (3.2), so a
   long run with many births and deaths accumulates closed slots. A compaction
   that renumbers accounts would have to update every reference, and belongs to
   `spec-checkpoint.md`. Every removal of an agent closes its account
   (`spec-agents.md` 3.6), so the growth is one slot per removal; a weight change
   (N-008) closes nothing. With the population module a death that empties an agent is a removal, so the growth also follows the death rate (`spec-population.md` 4.7). Still open (N-020).
7. **Goods valued as capital.** Durables are held as wear-able lots, but their wear
   is not a transfer (5.4.2). Whether depreciation should also appear as a named
   flow of value, for national-accounts style measures, is for `spec-indicators.md`.
8. **Region change of an account** (3.5). Answered: `spec-regions.md` 6.5 (cargo is in Transit accounts, not in the agent's; obligations tied to a region are `region_bound` refusals, `open_shipment` for transport, `spec-transport.md` 5.5; `NoRegion` is reserved and unused in version 1); migration is `spec-migration.md`'s: a timed journey, and the region change at arrival is the ledger-level `Relocate` with the cause `migration` (`spec-migration.md` 8.4).
9. **Reporting the construction bucket** (7.4.1). The bucket is readable, but whether
   the world emits it is for `spec-observability.md`. Recommend one `BoundaryFlow`
   set after `WorldCreated`, with `Tick` -1 (before tick 0), the only event with a
   negative tick; the alternative is a query and the snapshot only.
10. **Work in progress on a balance sheet** (9.2). Pending output is in no account
    (8.2.2), so a balance sheet cannot value it. The snapshot reports it by commodity
    (`spec-production.md` 14.3); valuing it, at cost or at a price, is for
    `spec-indicators.md`.

Settled by the new specs: the goods form of `initial`, the `disposal` reason, the empty
side of a `Transform`, `WearItems` (all N-033, applied in 5.4.3, 7.2 and 8.2.1), the
lots of a warm start (N-056, applied in 5.2.4, 7.4.1 and 9.1) and the query for
expiring lots (N-053, 5.3.3).

## 16. Where this lives in the code

| Concern | Home |
|---------|------|
| Accounts, lots, `Transfer`, `Move`, `Transform`, `Wear`, `WearItems`, `Expiring`, ageing, reservations, flow totals and the construction bucket, `Audit` | `internal/ledger` (stdlib only; architecture.md 2.9) |
| The registry of reasons and their kinds | `internal/ledger` (a fixed table), with the string names used by `internal/logkeys` |
| Checked add and multiply | the `internal/` checked-arithmetic helpers (`spec-currency.md` section 7) |
| Balance sheet valuation | `internal/ledger`, with the price table supplied by the root |
| The event `BoundaryFlow` | root package, `event.go` (`spec-observability.md`) |
| Opening and closing accounts for agents, estates, escrow and transit | the root package, in the specs that own them |
| Sentinels (`ErrInsufficientFunds` and the rest) | `internal/ledger`; the root maps them to rejection reasons (`spec-clearing.md` 3.2) and to its own errors |
