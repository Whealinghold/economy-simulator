# Spec: Clearing, the per-region double auction

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CLR-NNN` and
`NFR-CLR-NNN`, which tests cite (testing.md section 4). They are in section 12
(FR-CLR-001 to 035, NFR-CLR-001 to 003).

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs `spec-ledger.md` for reservations and transfers (section 3.3 and ledger
section 6; the `trade` reason is in the ledger's registry, section 7.2).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

Sources: [bazaarbot-reference.md](bazaarbot-reference.md) sections 5, 6 and 11
describe what the original does and where it is defective. The Doran and
Parberry paper it is based on has been read (journal version, `evidence.md`); its
offer-resolution pseudocode has no crossing check either (open question 1).

## 1. Purpose and scope

A **market** is one commodity in one region. Each tick, the agents of that
region place orders for it, and the market matches them and sets the prices.
This spec defines:

- what an **order** is, which orders are accepted, and how an accepted order is
  backed by the agent's cash or stock (section 3);
- how orders are matched, in which order, and at what price (sections 4 and 5);
- how a trade is settled and what is conserved (section 6);
- the price the market publishes, including when nobody trades (section 5.4);
- the `Clearing` extension point and what the world enforces on it (section 8);
- the events and logs (section 9).

It settles the two bazaarBot defects the roadmap assigns here: matching without
a crossing check, and no budget check (section 10). Out of scope: how an agent
chooses its limit price and quantity (`spec-agents.md`, `spec-beliefs.md`),
trade between regions (`spec-transport.md`, `spec-merchants.md`), and the tick
phases (`spec-world.md`).

## 2. Principles

2.1. A market is identified by a region and a commodity. The matching code does
not know what the region or the commodity is. Orders in one market never meet
orders in another. Nothing in this spec branches on a commodity or region
identifier (architecture.md 9.1). In tier 1 there is one region, which is the
degenerate case.

2.2. Orders last one tick. Every order is placed before any market clears, and
every unmatched order expires when its market has cleared. The book is empty at
the start of each tick.

2.3. **Admission** and **matching** are separate. Admission (section 3) checks an
order and backs it with a reservation; it is a rule of the market and is the
same for every `Clearing`. Matching (sections 4 and 5) is the `Clearing`
extension. Because every order is backed at admission, settlement of any valid
trade cannot fail.

2.4. A market has no tunable parameters of its own. Price and quantity bounds
come from configuration validation (architecture.md 3.5; they are bounds in
`bounds.go`, not tunables: N-016). The seed price that starts the published price
comes from the commodity and region data, or from a warm start (section 5.4).

2.5. All arithmetic is `int64` fixed point with checked overflow and a stated
rounding rule (`spec-currency.md` section 2.3; architecture.md 3.5). Rounding in
this spec is half to even, not truncation (section 5.2).

## 3. Orders and admission

### 3.1 The order

An order has an agent, a commodity, a side (**bid** to buy, **ask** to sell), a
**limit price** and a quantity. The limit price is the most a bid will pay, or
the least an ask will accept, per unit, in minor units. The quantity is a count
of whole units. An ask may also name a **lot** (commodity, born tick and
remaining durability, `spec-ledger.md` 6.7), in which case its goods are exactly
that lot. Only the estate pseudo-agent names a lot (3.1.2).

3.1.1. An agent has at most one order per commodity, lot and side per tick, and an
order that names no lot is identified by its agent, commodity and side. This keeps
the book simple and keeps the property that an agent never trades with itself, by
the integer order of its own limits and not by refusing the second side: an
ordinary agent places one order per commodity (the world rejects a second intent
for a commodity while it builds the orders, `spec-agents.md` 6.5.2), so it places
one side, and a treasury (3.1.2) may place a bid and an ask of one commodity because the two
limits are strictly ordered, bid below ask, on the integers (`spec-fiscal.md`
9.4.1, FR-FIS-053), so the pair never crosses (4.3 requires the bid's limit not
below the ask's). The original has one order per agent per commodity
(bazaarbot-reference.md section 5).

3.1.2. **The estate exception.** A region's estate is a pseudo-agent
(`spec-estates.md` 2.7). It places only asks, never bids, and may place several
asks for one commodity in a tick, each naming a different lot, because an estate
item is priced by its remaining durability (`spec-skill-barriers.md` 2.7). The
rule of 3.1.1 therefore reads: at most one order per agent, commodity, named
lot and side, where an ordinary agent names none. An order is identified by that
quadruple, so a trade can say which ask it filled. Because the estate never bids, it cannot
trade with itself. A **counterparty** of the external sector (`spec-external-sector.md` 4.2) is a second pseudo-agent of this kind: a buyer counterparty places only bids and a seller only asks, one order per commodity, so the rule above reads the same and an order of a counterparty is an ordinary order backed by a reservation in its own account (`spec-external-sector.md` 4.3).
A **treasury** of a governed jurisdiction (`spec-polity.md` 5, `spec-fiscal.md` 9) is a third pseudo-agent of this kind. It places its standing orders at P3.4a (`spec-fiscal.md` 9.4.2) and may place a bid and an ask of one commodity in a tick, one of each side, with strictly ordered integer limits, validated at construction (`ErrFiscalShape`); two bids, or two asks, of one commodity of one treasury are never admitted from the configuration, and a second order of one side is `duplicate_order` all the same. A treasury never matches itself because the bid's limit is below the ask's, and the clearing may assume it (FR-FIS-053); the identity check of 8.2 stays as a guard. A treasury has no strategy and no belief, so its orders have no outcome (7.3).

### 3.2 Validation

An order is **rejected** (not admitted) with a `reason` if any of these holds.
Reasons are exported constants of the root package (`event.go`); `internal/logkeys` holds
the key strings only.

| `reason` | Condition |
|----------|-----------|
| `unknown_agent` | the agent (or pseudo-agent) does not exist or is not in the market's region |
| `side_not_allowed` | a pseudo-agent that may place one side only places the other (the estate, 3.1.2; a counterparty, `spec-external-sector.md` 4.2) |
| `unknown_commodity` | the commodity is not in the configuration |
| `not_tradable` | the commodity is outside the tradable set of the agent's role (`spec-agents.md` 3.4). The world rejects such an intent while it builds the orders (`spec-agents.md` 6.5.2), so it never reaches matching. A pseudo-agent has no role and no tradable set, and this check does not apply to it |
| `not_permitted` | the agent is not a member of the body that holds the `sell` right (for an ask) or the `buy` right (for a bid) in force on the commodity in its region (`spec-institutions.md` 5.6.2, FR-ORG-024). The world rejects such an intent while it builds the orders, beside `not_tradable` and after it, so it never reaches matching. A pseudo-agent holds no membership and this check does not apply to it |
| `invalid_price` | the limit price is below 1 or above `MaxPrice`, a validation bound of `spec-config.md` 4.1 (in `bounds.go`, not a configured tunable) |
| `invalid_quantity` | the quantity is below 1 or above `MaxQuantity`, a validation bound of `spec-config.md` 4.1 |
| `duplicate_order` | the agent already has an admitted order of this side for this commodity (and named lot, if any) this tick |
| `overflow` | limit price times quantity does not fit in `int64` |
| `insufficient_funds` | a bid whose limit price times quantity exceeds the agent's available cash |
| `insufficient_stock` | an ask whose quantity exceeds the agent's available stock |
| `no_room` | with the storage module on, a bid of an agent whose account is limited, whose quantity times each size exceeds the free room of the account in either dimension (`spec-storage.md` 5.2.1, FR-STO-008). Not applied to an ask or to a pseudo-agent |

Checks run in the order of the table; the first that fails gives the reason.
The `reason` of a second intent for one commodity (`duplicate_order`) and of one
outside the tradable set (`not_tradable`) is therefore decided by this order too.
An order is rejected whole. The market never reduces a quantity to fit a budget:
sizing an order is the strategy's job (`spec-agents.md`), and a silent clamp of a
quantity would hide a strategy defect. The one clamp the market makes is of a
price, to a legal cap or floor (3.2.1), and it is counted, never silent.

3.2.1. **The legal clamp.** With the institutions module on and a price control in
force for (region, commodity) (`spec-institutions.md` 6.3), an order that has passed
the checks of `invalid_price`, `invalid_quantity`, `duplicate_order` and `overflow`
is clamped before the checks of funds, stock and room: a bid whose limit price is
above the cap has its limit replaced by the cap, and an ask whose limit price is
below the floor has its limit raised to the floor. The admitted order carries the
replaced limit: its reservation is `limit * quantity` at it, `OrderPlaced` shows it,
`Clearing` receives it and the world's check of 8.2 compares with it. The clamp adds
no `reason`, reduces no quantity and changes no tie-break key (4.2). The overflow
test of 3.2 sees the limit before the clamp, which is harmless: a floor raise can
lift an ask above the product tested, but an ask reserves no money, and a cap only
lowers a bid. Every replaced limit is counted in `PriceControlled`
(`spec-institutions.md` 11.1).

### 3.3 Reservation

When a bid is admitted, **reservation** moves `limit price * quantity` of the
agent's cash to a reserved balance. When an ask is admitted, it reserves
`quantity` units of the agent's stock. The **available** cash or stock of an
agent is its balance minus what is reserved. A reservation is released when the
order expires or is filled (section 6).

A bid reserves at its limit price, the worst case: a trade price never exceeds
the bid's limit (section 5.1), so the buyer is never short at settlement. The limit
is the admitted limit, after the legal clamp (3.2.1). With the storage module on, a
bid of a limited account also reserves its room, `quantity` times each size
(`ReserveRoom`, `spec-storage.md` 4.1, 5.2.1), in the same step as its cash, and the
room is released as the cash is.

Reservation is a ledger operation (`spec-ledger.md` section 6): reserve,
release, and a transfer marked from reserved that carries a `reason` (`trade`).
The ledger owns the representation. Reservations last one tick and none exists
at a tick boundary (`FR-LED-020`).

3.3.1. Money spent on a bid is not available to other orders in the same tick,
and proceeds of a sale are not available until the next tick. So, given the
tie-break keys (4.2), the order in which a tick's markets are cleared cannot change
any result (section 7). The keys come from one stream in market order, so the fixed
order of the markets is part of the draw sequence (`spec-world.md` D7, ruling C22).

## 4. Matching

The default `Clearing`, `AverageClearing`, works on the admitted bids and asks of
one market.

4.1. **Order of the books.** Bids are sorted by limit price, highest first. Asks
are sorted by limit price, lowest first. Equal limit prices are ordered by a
**tie-break key**, then by the order's position in the input.

4.2. **Tie-break key.** Before sorting, the clearing draws one `uint64` from the
world's random source for each order, bids first and then asks, each in input
order. Lower keys come first. This replaces bazaarBot's shuffle with a draw that
is fair between agents (no agent is favoured by its position) and reproducible.
Because the input position is unique, the comparison (limit price, key, input
position) is a total order, as architecture.md 7.6 requires. The key is drawn
for every order, whether or not there is a tie, so the number of draws depends
only on the number of orders. In the root, `AverageClearing.Clear` takes the draws
(one source word per order, `spec-world.md` 7.2, ruling C8) and hands the keys to
`internal/clearing`, which takes no random source (8.3).

The input order is supplied by the world and must itself be deterministic (the
market lists orders in the order they were admitted, which follows the agents'
iteration order).

4.3. **Crossing.** Take the first bid and the first ask. If the bid's limit
price is lower than the ask's, matching stops: no bid and ask can trade, because
this pair has the highest bid and the lowest ask. Otherwise the pair **crosses**
and trades.

4.4. **Quantity.** A pair trades `min(bid remaining, ask remaining)` units. The
order with nothing remaining leaves its book. Repeat from 4.3 while both books
are non-empty.

This greedy rule trades the largest number of units any matching could: it pairs
the most generous buyer with the cheapest seller, and a unit stays unmatched
only if no remaining bid reaches any remaining ask.

4.5. Orders left in either book when matching stops, and the remainder of a
partly filled order, are **unmatched**. They are reported (section 5.5) and
expire (section 7).

## 5. Price

### 5.1 The trade price

The price of a trade is the midpoint of the two limit prices, per unit,
rounded half to even (5.2). Each trade has its own price, so the trades of one
market in one tick can have different prices. The price is between the ask's
limit and the bid's limit, inclusive, so a buyer never pays more than its limit
and a seller never receives less.

The midpoint splits the surplus of a trade equally. It is the bazaarBot rule
(bazaarbot-reference.md section 6, step 4) and it is the default rule only: a
`Clearing` can price differently, within the bounds the world enforces
(section 8.2).

### 5.2 Rounding

The midpoint of integer prices `a <= b` is `a + (b - a) / 2`. If `b - a` is
even this is exact. If it is odd the exact midpoint ends in one half, and the
result is the even one of the two neighbouring integers (**half to even**). The
form `a + (b - a) / 2` cannot overflow for non-negative prices.

Truncation always rounds towards the lower price, which would on average favour
buyers by half a minor unit per trade. Half to even removes that bias and keeps
the result inside the interval, because both neighbours lie in `[a, b]`.

### 5.3 Consequence for the minor unit

A half-unit bias matters only when prices are small. `spec-currency.md` section 3
already requires a minor unit small enough that the cheapest good is at or above
the resolution floor, so a half unit is a small fraction of any price.

### 5.4 The published price

After matching, the market publishes one **market price** for the tick:

- If at least one unit traded, it is the volume-weighted mean of the trade
  prices, `sum(price * quantity) / sum(quantity)`, rounded half to even. The sum
  is accumulated in 128 bits (`math/bits`) so no intermediate overflows; the
  result is between the lowest and highest trade price, so it fits.
- If nothing traded, it is the previous market price, carried forward.
- Before the first trade ever, the market price is the **seed price** of the
  commodity in the region (`spec-beliefs.md` 8.1). By default that is the
  effective reference price (`spec-goods.md` 6.3: the region's override, else the
  commodity's `ReferencePrice`); when a warm start is configured it is the
  `Config.WarmStart.Prices` entry for the region and commodity
  (`spec-viability.md` 8.1, 8.2.1). The beliefs of the initial agents start from
  the same number, so the market and the beliefs begin together. bazaarBot starts
  at 1.0 for every good, which is not a price in any setting.

The market also publishes the **price age**: the number of ticks since a unit
last traded. It is 0 when units traded this tick and counts up from there. A
market that has never traded has an age equal to the ticks run.

**The price window.** The market also keeps the **price window** of
`spec-beliefs.md` 7: a ring of the last `PriceWindow` market prices of ticks in
which units traded (age 0). Decision: the market keeps it, in the same record as
the published price and the age (`market.go`), because it is a record of what
this market did and nothing outside the market writes it. The `Clearing` does not
see or write it. The market pushes the market price into the ring once per tick,
after the market has cleared and before the outcomes are processed, and only when
units traded (`spec-beliefs.md` 7.1, 7.4). Beliefs own the definition of the
window and of its statistics (mean, low, high, position: `spec-beliefs.md` 7.2,
7.3) and `PriceWindow` itself (`spec-beliefs.md` 11); the ring is part of the
checkpoint under `spec-beliefs.md` 13. `internal/clearing` does not know it.

The published price is a record of what happened, not a bound on what may
happen. The market does not clamp it, and it does not smooth it. Bounds and
smoothing on beliefs are `spec-beliefs.md`'s business.

### 5.5 What an empty or thin market reports

Every tick the market also reports, for each side: the number of unmatched
units, and the best unmatched limit price (the highest unmatched bid, the lowest
unmatched ask), or "none" if that side has nothing unmatched.

These give a price signal when the published price is stale. If only asks
remain, the lowest of them is a ceiling on what a buyer needed to pay; if only
bids remain, the highest is a floor. A strategy and the belief update can use
them with the price age (stability, section 13).

## 6. Settlement

For each trade of `q` units at price `p` between buyer `B` and seller `S`:

1. `q` units move from `S` to `B`.
2. `p * q` money moves from `B` to `S`.
3. `B`'s reservation for the bid is reduced by `bid limit * q`, and the
   difference between that and `p * q` goes back to `B`'s available cash.
4. `S`'s reservation for the ask is reduced by `q`.
5. With the storage module on and a limited buyer, the `q` units are credited from
   the room the bid reserved (the credit marked from reserved, `spec-ledger.md`
   6.8), which cannot fail for room (`spec-storage.md` 5.2.2, FR-STO-009).

All four happen or none does. They are ledger transfers with the `reason`
`trade` (`spec-ledger.md`), so the faucets-and-sinks accounts show no creation or
destruction. `p * q` was checked for overflow at admission, because `p` is at
most the bid limit.

When a market has finished, the remaining reservation of every unmatched or
partly filled order is released, and with the storage module on so is the reserved
room of the part of a bid that did not trade (`ReleaseRoom`), at the end of P4.1.

6.1. Within a market, goods and money are conserved exactly: the units sold
equal the units bought, and the money paid equals the money received. Across the
world they are conserved except by recipes and by explicit faucets and sinks
(testing.md 6.3). Settlement is where the original is not conserved at the
level of one round (bazaarbot-reference.md section 11, items 2 and 6).

## 7. Tick-level properties

7.1. The market does not carry state between ticks other than the market report of
the last clearing (the published price, the price age, the volume, the unmatched units
of each side and the best unmatched bid and ask with their flags: 5.4, 5.5) and the
price window (5.4). Orders do not persist. The state is listed in section 16.

7.2. All orders of a tick are placed (admitted or rejected) before any market
clears. A market clears from its admitted orders only. Since each order is
already backed by a reservation (3.3), clearing the markets in any order gives
the same trades, prices and balances **given the tie-break keys** (4.2). The keys
come from one stream in market order, so the fixed order of the markets is part
of the draw sequence (`spec-world.md` D7, ruling C22). The world clears them in a
fixed order (by region, then commodity), which is also the order in which events
come out (section 9).

7.3. Unmatched orders are not an error and are not retried. The agent learns the
outcome of its order after the market clears (filled quantity and unfilled
quantity). The original updates beliefs once per matched trade and once
per unmatched order, so one agent can see several updates for one good in one
round (bazaarbot-reference.md section 11, item 11). Here the outcome goes to the
agent's strategy method `Revise` (`spec-agents.md` 6.2, 6.4.4) **once per admitted
order of a household agent**, however many trades filled it, together with the market report of the
order's market (the market price, the price age and the best unmatched bid and ask
with their flags: 5.4, 5.5). Outcomes are delivered after every market of the tick
has cleared and the price windows are updated, in order of region, then commodity,
then the order in which the orders were admitted (`spec-beliefs.md` 5.1.1,
FR-BEL-007, FR-BEL-008). A rejected order has no outcome: it never reached the
market. Nor has an order of a pseudo-agent: an estate's ask, a counterparty's
order and a treasury's order have no strategy or belief, so they receive no outcome
and take no outcome draw (`spec-world.md` D8, ruling C21; `spec-fiscal.md` 9.5). What `Revise` does with the outcome is `spec-beliefs.md` and
`spec-agents.md`; the market only supplies it.

## 8. The `Clearing` extension point

`Clearing` is one of the four sanctioned extension interfaces
(architecture.md 4.3). It is defined in the root package where it is consumed
(4.1) and ships with `AverageClearing` as its default, with a compile-time
assertion (4.5). A proposed shape (final form is decided with the code):

**The owner of `World.Clearing` (ruling C7, `spec-world.md` 4.2).** The field
`Config.World.Clearing` is a string that names the clearing a world uses. This section
owns it, as `spec-spawner.md` 3.1.3 owns `World.Spawner`: the default is `average`
(from the defaults of `spec-config.md` 5, so an empty value is an error and not a
default), `average` is `AverageClearing` and is built in, and a consumer's
implementation is registered under another name by `WithClearing(name, c)`
(`spec-world.md` 4.1). `NewWorld` rejects an empty name or one that is not registered
with `ErrUnknownClearing`, a name registered twice (`average` is built in, so
registering it is registering it twice) or the name `default` with `ErrDuplicateID`, as
for the other extensions (`spec-world.md` FR-WLD-002), and a nil `Clearing` with
`ErrNilExtension`.
The **name** is in the effective configuration and its hash and the code behind it is
not, so a world with a custom rule does not hash like one with the default
(`spec-config.md` 8.1). An implementation that keeps state between calls implements
`Stateful` (architecture.md 4.3, 7.7; ADR 0002 decision 7); `AverageClearing` keeps none.

```go
// Clearing matches the admitted orders of one market and prices the trades.
type Clearing interface {
	Clear(bids, asks []Order, rng *rand.Rand) []Trade
}
```

8.1. **Randomness.** `rng` is the world's random source. An implementation must
not create its own, use a global one, or keep `rng` after `Clear` returns
(architecture.md 7.7). The doc comment on `Clearing` states this.

8.2. **What the world checks.** The world validates the whole list of trades
before it settles any. A list is valid if, for every trade:

- the buyer holds a bid and the seller holds an ask in this market;
- the quantity is at least 1 and, summed per order, no more than the order's
  quantity;
- the price is at least the ask's limit price and at most the bid's limit
  price, both as admitted (after the legal clamp of 3.2.1);
- the buyer and the seller are different agents.

If the list is invalid, the world settles none of the trades of that market,
releases its reservations, and `Step` returns an error wrapping `ErrInvalidTrade`
at P4.1 (`spec-world.md` 3.5): the step fails and the world is failed, so every
later call with an error result returns `ErrWorldFailed`. The world does not also
log it (architecture.md 8.3). A faulty extension
therefore cannot create money or goods or break a budget; it can only fail
loudly. The default `AverageClearing` never produces an invalid list.

Because of this, the original's behaviour of trading when the top bid is below
the top ask cannot be reproduced by a `Clearing`: it would violate the price
bound. That is deliberate (section 10).

8.3. **Internal home.** The algorithm of `AverageClearing` is in
`internal/clearing`, over primitive structs, and the root converts to and from
them inline (architecture.md 2.9):

```go
// internal/clearing: stdlib only, no root types, no random source.
type Offer struct {
	Ref   int    // index into the caller's slice
	Price int64
	Qty   int64
	Key   uint64 // the tie-break key of 4.2, drawn by the root
}

type Match struct {
	BidRef, AskRef int
	Price, Qty     int64
}

func Average(bids, asks []Offer) []Match
```

`AverageClearing.Clear` in the root draws one source word per order, bids first and then
asks, each in input order (4.2; `spec-world.md` 7.2, ruling C8), puts it in `Key`, and
calls `Average`, which sorts by (limit price, `Key`, position in the input) and draws
nothing. The key is a raw `uint64`, so it needs no reduction. The internal package does
not know about commodities, roles or agents. The
checked arithmetic it needs (the midpoint, the 128-bit volume-weighted mean) is
in the `internal/` checked-arithmetic helpers that `spec-currency.md` section 7
names.

## 9. Observability

The events are those of observability-and-logging.md section 4.3. Their fields
and order for this spec:

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `OrderPlaced` | when an order is admitted | region, commodity, agent, side, limit price (the admitted limit, after the legal clamp of 3.2.1), quantity |
| `OrderRejected` | when an order is not admitted | region, commodity, agent, side, `Reason` |
| `TradeExecuted` | after a trade is settled | region, commodity, buyer, seller, price, quantity |
| `MarketCleared` | once per market per tick, after its trades | region, commodity, market price, price age, volume, unmatched bid units, unmatched ask units, best unmatched bid and ask (with a flag for "none") |

9.1. `OrderPlaced` and `OrderRejected` are emitted in the order the orders were
submitted. Then, for each market in order of region and commodity, the
`TradeExecuted` events in the order the trades were made, and then that market's
`MarketCleared`. `MarketCleared` is emitted even when nothing traded or nothing
was offered, so an observer can see a stale price. No event has a map.

9.2. Logs (observability-and-logging.md section 3). Everything here is
`Debug`, guarded by `Enabled`, with constant messages: "order rejected"
(`commodity`, `agent`, `side`, `reason`) and "market cleared" (`commodity`,
`price`, `quantity` for the volume). There is no `Info`, `Warn` or `Error` line:
a rejected order is an event, not a degraded market; and the invalid-trade error
is returned, not logged (8.2). A price clamped to a bound is a belief concern
and is logged there, by the `Warn` "price beliefs at bound" of `spec-beliefs.md`
12.2 (one line per market per ten ticks), not by the market. The `not_tradable`
rejection of 3.2 is an `OrderRejected` event and the same `Debug` line.

9.3. With the default no-op observer and logger, the clearing path allocates
nothing beyond scratch slices held by the world and reused across ticks
(NFR-CLR-002; observability-and-logging.md 4.5 and 8.2).

## 10. Differences from bazaarBot

Item numbers are those of bazaarbot-reference.md section 11.

| Item | bazaarBot | This port |
|------|-----------|-----------|
| 1 | Matches until a list is empty, even when the bid is below the ask (the paper's pseudocode has no crossing check either) | Matches only crossing pairs (4.3, FR-CLR-009). **Decision: require crossing.** A trade at a price that one side's own limit forbids contradicts what a limit price means, and the budget and limit rule of the roadmap (stability duties) |
| 2 | No budget or stock check; money can go negative within a round | Every order is backed at admission (3.3, FR-CLR-005, FR-CLR-006) |
| 3 | The shuffle skips its last swap | Not ported. A seeded key per order (4.2) |
| 4 | Unstable sort; ties in an unspecified order | A total order ending in the input position (4.1, FR-CLR-010) |
| 6 | Consuming more than is held succeeds and is ignored | Not a clearing concern (`spec-ledger.md`); here an ask can only be admitted for stock the agent has |
| price | `Float`, mean of two floats | `int64`, midpoint rounded half to even (5.2) |
| history seed | 1.0 for every good | The seed price: the effective reference price, or the warm-start entry (5.4) |
| unmatched offers | Each triggers a failure update | Reported with the outcome of the order, once per admitted order (7.3); the update belongs to `spec-beliefs.md` |

What is kept: one order per good per agent per round, all orders placed before
any is cleared, best bid against best ask, the average of the two limits as the
trade price, and orders that last one round. The difference from the original is
therefore behavioural only in the rows above, and the one-region scenario of
`spec-reference-chain.md` is where the behaviour is compared, not bit for bit.

No "faithful mode" is provided: it would need trades outside the price bound,
which section 8.2 forbids for every `Clearing`. Open question 4.

## 11. A worked example

One market, one tick. Orders, each as limit price times quantity:

| Order | Side | Limit | Qty |
|-------|------|-------|-----|
| B1 | bid | 12 | 3 |
| B2 | bid | 10 | 2 |
| B3 | bid | 7 | 4 |
| A1 | ask | 6 | 2 |
| A2 | ask | 9 | 4 |
| A3 | ask | 11 | 1 |

All limits differ, so the tie-break key does not matter. Bids sorted highest
first are B1, B2, B3. Asks sorted lowest first are A1, A2, A3. Admission
reserves cash for the bids: B1 `12 * 3 = 36`, B2 `10 * 2 = 20`, B3 `7 * 4 = 28`;
and stock for the asks.

| Step | Pair | Crosses? | Qty | Price | Remaining after |
|------|------|----------|-----|-------|-----------------|
| 1 | B1 (12) and A1 (6) | 12 >= 6 | `min(3, 2) = 2` | `(12 + 6) / 2 = 9` | B1: 1; A1: 0 |
| 2 | B1 (12) and A2 (9) | 12 >= 9 | `min(1, 4) = 1` | `10.5`, half to even: 10 | B1: 0; A2: 3 |
| 3 | B2 (10) and A2 (9) | 10 >= 9 | `min(2, 3) = 2` | `9.5`, half to even: 10 | B2: 0; A2: 1 |
| 4 | B3 (7) and A2 (9) | 7 < 9, stop | | | |

In step 2 the neighbours of 10.5 are 10 and 11, and 10 is even. In step 3 the
neighbours of 9.5 are 9 and 10, and 10 is even.

Results:

- Volume `2 + 1 + 2 = 5` units. Value `2 * 9 + 1 * 10 + 2 * 10 = 48`.
- Market price: `48 / 5 = 9.6`, which rounds to 10 (`q = 9`, remainder 3,
  `2 * 3 = 6 > 5`, so round up).
- B1 paid `2 * 9 + 1 * 10 = 28` of its 36 reserved, so 8 is released. B2 paid 20
  of 20. B3 traded nothing, so its 28 is released at once.
- Unmatched bids: B3's 4 units, so the best unmatched bid is 7.
- Unmatched asks: A2 sold `1 + 2 = 3` of its 4 units (steps 2 and 3), leaving
  1, and A3's 1 unit did not trade, so 2 unmatched ask units; the best
  unmatched ask is 9. Price age 0.
- Conservation: sellers gave `2 + 3 = 5` units and buyers received
  `3 + 2 = 5` (B1 got 3, B2 got 2). Buyers paid `28 + 20 = 48` and sellers
  received `2 * 9 + 3 * 10 = 48` (A1 got 18, A2 got 30).

## 12. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CLR-001 | Orders MUST be matched only against orders of the same region and commodity. |
| FR-CLR-002 | An order whose limit price is below 1 or above `MaxPrice` MUST be rejected with `reason` `invalid_price`; one whose quantity is below 1 or above `MaxQuantity` (both bounded by `spec-config.md` 4.1) MUST be rejected with `invalid_quantity`. |
| FR-CLR-003 | An order for an unknown agent, an agent outside the market's region, or an unknown commodity MUST be rejected with `unknown_agent` or `unknown_commodity`. |
| FR-CLR-004 | A second order from the same agent for the same commodity, the same named lot (none, for an ordinary agent) and the same side in the same tick MUST be rejected with `duplicate_order`; a bid and an ask of one commodity from one agent are not by that fact a duplicate (3.1.1). |
| FR-CLR-005 | A bid MUST be admitted only if its limit price times its quantity does not exceed the agent's available cash, and on admission that amount MUST be reserved; otherwise it MUST be rejected with `insufficient_funds`. |
| FR-CLR-006 | An ask MUST be admitted only if its quantity does not exceed the agent's available stock of the commodity, and on admission that quantity MUST be reserved; otherwise it MUST be rejected with `insufficient_stock`. |
| FR-CLR-007 | If limit price times quantity overflows `int64`, the order MUST be rejected with `overflow`; no value MUST wrap and nothing MUST panic. |
| FR-CLR-008 | Every order MUST expire when its market has cleared, with its remaining reservation released, and no order MUST be present in the next tick's book. |
| FR-CLR-009 | A bid and an ask MUST NOT trade when the bid's limit price is below the ask's. |
| FR-CLR-010 | Bids MUST be matched highest limit first and asks lowest limit first, with equal limits ordered by a tie-break key drawn from the world's random source and then by input position, so the order is total. |
| FR-CLR-011 | The quantity of a trade MUST be the smaller of the two orders' remaining quantities. |
| FR-CLR-012 | Matching MUST continue with the best remaining bid and ask until one book is empty or the best pair does not cross. |
| FR-CLR-013 | The price of a trade MUST be the midpoint of the two limit prices, rounded half to even, and MUST lie between the ask's and the bid's limit price inclusive. |
| FR-CLR-014 | A buyer MUST NOT pay more per unit than its bid's limit price, and a seller MUST NOT receive less per unit than its ask's limit price, for any `Clearing` implementation. |
| FR-CLR-015 | The number of units matched in a market MUST equal the number of units that cross, computed independently from the sorted limit prices. |
| FR-CLR-016 | Settling a trade MUST move the quantity of goods from seller to buyer and price times quantity of money from buyer to seller as one atomic ledger transfer with `reason` `trade`, and MUST return the unspent part of the bid's reservation to the buyer. |
| FR-CLR-017 | In every market and tick, the units sold MUST equal the units bought, and the money paid MUST equal the money received, exactly. |
| FR-CLR-018 | The buyer and the seller of a trade MUST be different agents. |
| FR-CLR-019 | When units traded, the market price MUST be the volume-weighted mean trade price rounded half to even, MUST lie between the lowest and highest trade price, and MUST be computed without intermediate overflow. |
| FR-CLR-020 | When no unit traded, the market price MUST be the previous market price, and before the first trade ever it MUST be the seed price of the commodity in the region (the effective reference price, or the `WarmStart.Prices` entry when one is configured); the price age MUST be 0 after a tick in which units traded and otherwise one more than before. |
| FR-CLR-021 | Each tick a market MUST report, per side, the unmatched units and the best unmatched limit price, or that there is none. |
| FR-CLR-022 | The same seed, configuration and sequence of orders MUST give identical trades in an identical order, and `AverageClearing` MUST draw exactly one `uint64` per order, bids first, in input order. |
| FR-CLR-023 | Given the tie-break keys, the trades, prices and balances of a tick MUST NOT depend on the order in which its markets are cleared; the fixed order of the markets is part of the draw sequence of the keys (3.3.1, 7.2). |
| FR-CLR-024 | The world MUST validate the whole list of trades from a `Clearing` before settling any; if any trade breaks the rules of section 8.2 it MUST settle none of that market's trades, release its reservations and return an error wrapping `ErrInvalidTrade`. |
| FR-CLR-025 | The total outcome (filled units, unfilled units) of each admitted order of a household agent, with the market report of its market (market price, price age, best unmatched bid and ask with flags), MUST be delivered once to the agent's `Revise` after all markets of the tick have cleared, however many trades filled the order, in order of region, commodity and admission; a rejected order, and an order of a pseudo-agent (an estate, a counterparty or a treasury), MUST have no outcome and MUST take no outcome draw (7.3). |
| FR-CLR-026 | `OrderPlaced` MUST be emitted for each admitted order and `OrderRejected` with a `reason` from the table in section 3.2 for each rejected order, in the order submitted. The reason `no_room` is FR-STO-008 and the room of a settlement is FR-STO-009; the reason `not_permitted` is FR-ORG-024. |
| FR-CLR-027 | `TradeExecuted` MUST be emitted for each settled trade, in matching order and after settlement, and `MarketCleared` once per market per tick after that market's trades, even if nothing traded; markets MUST be ordered by region and then commodity. |
| FR-CLR-028 | A bid from a pseudo-agent that may only sell, such as an estate, MUST be rejected with `side_not_allowed`. |
| FR-CLR-029 | An ask that names a lot MUST reserve exactly the units of that lot, and settling a trade against it MUST move units of that lot and no other. |
| FR-CLR-030 | An intent for a commodity outside the tradable set of the agent's role MUST be rejected with `reason` `not_tradable` and an `OrderRejected` event, MUST NOT reach matching, and MUST NOT reserve cash or stock. |
| FR-CLR-031 | The checks of section 3.2 MUST run in the order of its table, so an order that fails two of them MUST be rejected with the first. |
| FR-CLR-032 | A market MUST push its market price into its price window once per tick, after it has cleared and before the outcomes are processed, if and only if units traded, and the `Clearing` MUST NOT read or write the window. |
| FR-CLR-033 | The clearing path MUST emit no `Info`, `Warn` or `Error` log line, and each `Debug` line MUST be guarded by `Enabled`. |
| FR-CLR-034 | The market report of the last clearing of each market (the published price, the price age, the volume, the unmatched units of each side and the best unmatched bid and ask with their flags) MUST be saved in a checkpoint, and a restored world MUST answer the same report as the original at the same boundary (7.1, 16). |
| FR-CLR-035 | A price control in force for (region, commodity) MUST replace, in admission and after the checks of price, quantity, duplicate and overflow and before those of funds, stock and room, the limit of a bid above its cap by the cap and the limit of an ask below its floor by the floor; the admitted order MUST carry the replaced limit in its reservation, in `OrderPlaced`, in the orders given to `Clearing` and in the check of 8.2, and the clamp MUST add no `reason`, change no quantity and draw nothing (3.2.1). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-CLR-001 | `AverageClearing` SHOULD run in time `O(n log n)` in the number of orders of a market. |
| NFR-CLR-002 | With the default logger and observer, a full clearing of the reference chain MUST allocate no more than a baseline held in the test, and each event-emission and logging call site MUST allocate nothing. |
| NFR-CLR-003 | Clearing SHOULD have a benchmark with `b.ReportAllocs()` over books of increasing size. |

## 13. Entries for the stability register

Merged into `spec-stability.md` section 4 (STB-L01 price, belief and limit loop;
STB-L03 stale price loop; STB-L11 budget loop; STB-L12 pairing bias). The
register is now the home of these entries, with their sign, delay, damper and
test; the text below is the source it was merged from and is kept for the record.

Original entries (each to be given its sign, delay, damper and test):

- **Price, belief and limit loop.** A high market price raises sellers' limits
  and then the next price; a midpoint trade price is a damper (the buyer's limit
  is above it and the seller's below), and bounded belief steps
  (`spec-beliefs.md`) are the other.
- **Stale price loop.** A price that nobody trades at stays published; beliefs
  anchor to it; orders stay uncrossed; the price stays stale. This is a
  reinforcing loop. Dampers: the price age, and the best unmatched quotes
  (5.5), which `spec-beliefs.md` must use to move off a stale price.
- **Budget loop.** An agent that cannot fund a bid cannot bid; with no purchases
  it cannot produce and cannot earn. The hard budget (3.3) prevents the
  negative-balance runaway of the original, and the cost is this loop; the floor
  under it is `spec-subsistence.md` and relief in `spec-estates.md`.
- **Pairing bias.** Greedy best-to-best matching with a random tie-break is
  neutral between agents with equal limits; a test over many seeds checks that
  no agent position wins more often than the others.

## 14. Open questions

1. **The paper.** Answered (2026-10-10). The paper (journal version, read in full)
   has no crossing check in its offer-resolution pseudocode either: it matches the
   highest bid to the lowest ask, as the source does. So defect 1 of
   bazaarbot-reference.md section 11 is in the paper as well as in the source. The
   decision in section 10 (require crossing) stands, because it follows from what a
   limit price means; this difference from the paper is recorded in section 10.
   The figures and the extended example of the paper were not seen, so whether its
   authors meant a crossing check without writing it is not known.
2. **One order per agent per commodity per tick.** This is simple and matches
   the original, but a strategy cannot post a ladder of prices. The estate is
   an exception (3.1.2), identified by its named lot, and a treasury may place
   one bid and one ask of one commodity, identified by their side (3.1.1). Relaxing the rule for
   ordinary agents needs a general order identifier in `Trade` and a rule for
   self-trades. Recommend keeping the restriction for version 1.
3. **Pairwise midpoint or one price.** A single clearing price for the whole
   market per tick is the textbook call auction, removes the dependence of price
   on pairing order, and is easier to reason about for stability. The roadmap and
   architecture.md name `AverageClearing`, so this spec keeps the pairwise rule;
   a uniform-price `Clearing` can be added as a second implementation, and
   `spec-stability.md` may recommend it.
4. **A "faithful" mode.** Not provided (section 10). If a comparison with the
   original is needed, run the original and compare the dynamics, not the
   trades.
5. **How `Step` reports `ErrInvalidTrade`.** Answered (`spec-world.md` 3.5): `Step`
   returns an error wrapping `ErrInvalidTrade` at P4.1 and the world is failed, as
   architecture.md 8.1 favours (8.2).
6. **The reference price's owner.** Settled in `spec-goods.md` 6.3: the
   commodity owns the default `ReferencePrice`, and a region may override it
   (the override field is `RegionPrices`, `spec-regions.md` 5.2). The price before the first trade
   is the seed price of `spec-beliefs.md` 8.1 (5.4), which is that effective
   reference price unless a warm start supplies one.
7. **`MarketCleared` for markets with no orders.** Emitting it every tick costs
   one event per market per tick when an observer is present. This spec emits it
   so a stale price is visible; revisit if a configuration has many idle
   markets.
8. **Who keeps the price window.** Settled (5.4): the market keeps it. The
   alternative, the world keeping a table of rings, adds nothing: the world would
   still write it from the market's price.

## 15. Where this lives in the code

| Concern | Home |
|---------|------|
| `Order`, `Trade`, `Clearing`, `AverageClearing`, `ErrInvalidTrade` | root package, `clearing.go`, `order.go`, `errors.go` |
| Admission, reservation, the market price, the price age and the price window | root package, `market.go` |
| The tie-break draw (one source word per order) and the registration of `World.Clearing` by name | root package, `clearing.go` (`AverageClearing.Clear`), `option.go` (`WithClearing`) |
| The matching algorithm over the keys it is given | `internal/clearing` (stdlib only, no random source; `Offer` with its `Key`, `Match`, `Average`) |
| Midpoint, half-to-even division, 128-bit volume-weighted mean, checked multiply | the `internal/` checked-arithmetic helpers |
| Rejection `reason` constants, including `not_tradable` | exported constants of the root package, `event.go`; `internal/logkeys` holds the key strings only |
| State contributed to a checkpoint (section 16) | root package, `market.go` |
| Reserve, release, `trade` transfers | `internal/ledger` (`spec-ledger.md`) |
| Events `OrderPlaced`, `OrderRejected`, `TradeExecuted`, `MarketCleared` | root package, `event.go` |

## 16. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). The market's, plain data in a
fixed order with no maps, saved in the Markets section of the checkpoint
(`spec-checkpoint.md` 4.2 tag 4), for each (region, commodity) of a region that is not an
outside node, in region order and then commodity order:

- the **market report of the last clearing**: the published price, the price age, the
  volume, the unmatched bid units and the unmatched ask units, the best unmatched bid
  (a present flag, then the price) and the best unmatched ask (a present flag, then the
  price) (5.4, 5.5).

The price window and its diagnostic run are the beliefs' state, saved beside the report
(`spec-beliefs.md` 13). **Not saved:** the orders, the reservations (none exists at a
boundary, 3.3), the tie-break keys and the list of trades of the last clearing (they are
a function of the tick). The report is read at the next tick by `Trade` (`spec-agents.md`
6.4.1), by the integration query `Market(r, c)` and by the snapshot, so a restored world
must answer the same report as the original at the same boundary (FR-CLR-034). The
implementation of an alternative `Clearing` that keeps state saves it through `Stateful`
(8), under its registered name.
