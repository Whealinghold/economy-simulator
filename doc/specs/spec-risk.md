# Spec: Risk: loss of cargo in transit and of stored goods as data, strikes drawn from the world's source, the `loss` sink, the embargo as an edge status, and the seams for insurance

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-RSK-NNN` and
`NFR-RSK-NNN`, which tests cite (testing.md section 4). They are in section 20.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md),
decision 2). It is off unless `Config.Modules.Risk` is present, and with it off
nothing in this spec runs, no draw is taken, no event is emitted, the reason `loss`
carries no flow and no field is added to a shipment: the economy is the one in which
every cargo that is dispatched arrives, less its spoilage, and every stored good stays
until it is used or spoils (`spec-transport.md` 5.6, `spec-storage.md` 8). It needs the
ledger (`spec-ledger.md`), the region graph (`spec-regions.md` 4), the transport module
for its route rows (`spec-transport.md`) and the storage module for its store rows
(`spec-storage.md`). It uses hooks in transport (5.5, 5.7, 9.1, 9.4), storage (8.3), the
tick (`spec-world.md` 6, 7.3, 8.2), integration (`spec-integration.md` 5), the merchant
(`spec-merchants.md` 5.2, 9.7), observability and the checkpoint, and it reserves seams
for insurance (`spec-insurance.md`), infrastructure (`spec-infrastructure.md`), a
conflict actor and settlement size. Every change
it needs in another spec is listed in the handoff and is not made here (section 19
states them).

Nothing has run in the library and no acceptance band exists (notes N-015, N-027).
Every number here is a **placeholder**: a default, a bound or a number of an example,
and none is called stable. The one script of section 17 is a stylised Docker Python
model of the arithmetic of sections 4 to 6, not a claim about the library. The settings
of the worked examples (18) are invented **examples**, and the words in them (`road`,
`bandit`, `flood`, `grain`, `hold`) are flagged as such and are data, never branched on
(`AGENTS.md`, N-017). The normative text says **strike**, **exposure** and **loss**.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

Until this module, a cargo that is dispatched arrives, less what spoils on the way, and
goods in a store stay until used. Both are kinder than any real road or any real
barn: goods are taken on the road, burnt in the barn, sunk with the ship. The ledger
reserves a sink for it, `loss` (`spec-ledger.md` 7.2), transport reserves the place
(`spec-transport.md` 10.1) and storage reserves the read (`spec-storage.md` 8.3), and
nothing posts under the reason. This spec covers:

- what a **risk row** is and what is **exposed** to it (section 3);
- the **route risk**: the strike fixed at dispatch, and the loss applied at arrival
  (sections 4 and 5);
- the **store risk**: the strike at the end of the tick, and the loss of the stored goods
  (section 6);
- the ledger's part, the sink `loss`, and the identities it adds (section 7);
- what an agent and a lane **see** of a risk (section 8);
- the **slots** that scale a strike probability between ticks (section 9);
- the **embargo** and the closed road as the edge status, and the hooks for the specs that
  come later (section 10);
- the position in the tick and the draws, the parameters, validation, observability, saved
  state, the register entries, what was run, worked examples, requirements and open
  questions (sections 11 to 22).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| An actor that raids, defends, escorts, pays tribute or profits from a loss: a strike destroys goods and gives them to no one | not in version 1. A consumer's controller sets the scale slots (9; N-131); a raider that gains would be an agent with an account and a strategy, and a loss that moves goods to it would be a transfer with its own reason (open question 3) |
| Insurance: a premium, a payout, a payout claim, moral hazard and adverse selection | `spec-insurance.md` (INS). This spec supplies the events and the expected loss of a route (10.2) |
| Defence by the size of a settlement | not in version 1 (open question 5): the hook H7 of `spec-settlements.md` 6.3 is declined; a consumer lowers a scale when a settlement class changes |
| A closure of an edge by chance, or a blockade that seizes cargo in flight | not in version 1 (open question 4). The embargo and the closed road are `spec-regions.md` 7's edge status, set by a consumer (`spec-integration.md` 5.1) or by a jurisdiction's border (`spec-polity.md` 6.3 H10, H13) |
| Loss of **money**: a purse taken from a traveller, a hoard burnt | not in version 1 (7.4): a money sink with no counterpart would shrink the money supply with no agent to receive it. A pack that wants it writes a levy (`spec-money-supply.md` 4.2, 4.5) |
| A facility (a bridge, a mill, a harbour) damaged by a strike | not in version 1 (open question 13): a facility is module state of `spec-infrastructure.md`, its condition is lowered by CAP's `AdjustFacilities` (11.2 there), which a consumer calls between ticks, and a strike by chance inside a step would need a position and CAP's internal form. A ruin leaves no goods and posts no `loss` (`spec-infrastructure.md` 6.2.3) |
| Loss of goods in a household's own base room (a burglary, a house fire) | not in version 1: only a store is exposed (6.1; open question 6) |
| A disaster that fails a harvest, empties an endowment or kills households | the slots of production (`spec-production.md` 6.8), resources (`spec-resources.md` 6.3) and population (`spec-population.md` 6.6), which a consumer sets between ticks. This module adds no generator of shocks |
| Loss on a migration journey | not in version 1: a migrant carries its account with no capacity (note N-120) and is not a shipment (open question 7) |
| Tools lost by wear, goods lost by spoilage | `spec-goods.md` 4, 5; `spec-ledger.md` 5.3, 5.4 (the sinks `spoilage` and `wear`, unchanged) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | A risk is a **row of data**: the probability of a strike and the share it takes, for a directed edge and a mode, or for a store class in a region. Nothing attacks: "bandits" and "a flood" are a pack's words for a row | 3 |
| 2 | **Two things are exposed and nothing else**: a shipment (its cargo, or for a trip to oneself the traveller's own account) and the stored part of a limited account | 3.1 |
| 3 | A strike takes a **share** of what is exposed, of every commodity (except the mode's `Vehicle` in a traveller's own account, 5.4.1), by one rounding rule (half to even); never a chosen set of goods, and never money | 5.3, 6.3, 7.4 |
| 4 | A route's strike is **fixed at dispatch** and kept in the shipment as `LossBP`; the loss is applied at **arrival**, from the Transit account, before the delivery | 4, 5 |
| 5 | A row may be **common**: one draw for everything exposed to it in the tick. Correlation is a flag on a row, not a second mechanism | 3.3, 4.2, 6.2 |
| 6 | A store's strike comes at the **end of the tick**, after ageing and before the flow totals are final, at a new position P8.1a | 6 |
| 7 | A row whose effective probability is 0 **takes no draw**, so a module that is on with every scale at 0 is, to the bit, the economy with it off | 3.4, 11.3 |
| 8 | Loss is the sink **`loss`**, goods only, from a Transit account or an agent's account to the boundary; its owner becomes this spec | 7 |
| 9 | A shipper reads the **expected loss** of a route in its view and nothing else of a risk; no strategy sees a strike, a draw or a scale | 8 |
| 10 | Two **slots**, scales of a strike probability, set between ticks, no timer, never edit a configured row | 9 |
| 11 | **The embargo is an edge status** (`spec-regions.md` 7). This spec adds no mechanism for it and no autonomous closure | 10.1 |
| 12 | **No macro state**: nothing is counted for the economy to react to; the state is the two slot lists and one field of a shipment | 15 |
| 13 | **No endogenous risk**: a probability does not depend on traffic, value carried, a price or a balance. A feedback of that kind belongs to a consumer's controller (N-131) | 16 |
| 14 | The module registers **no `region_bound` refusal**: what it exposes is already tied to a region by `open_shipment` (TRN) and `has_store` (STO) (N-090) | 3.5 |

## 3. Risk rows

### 3.1 What is exposed

3.1.1. An **exposure** is a set of goods that a strike can reach. There are three, and
no other:

| Exposure | What is in it | Owner of the goods | Struck at |
|----------|---------------|--------------------|-----------|
| a **cargo** | the goods in the Transit account of a shipment (`spec-transport.md` 7.1) | the consignee, on arrival (`spec-transport.md` 5.9.1); in no agent's account until then | arrival (5) |
| a **trip account** | the holdings of the traveller of a self-carried trip, whether or not it also consigns lines to another agent, **except units of the mode's `Vehicle` commodity** (`spec-transport.md` 3.2.2: a vehicle is not cargo; 5.1: a trip to oneself uses no Transit account, and the goods a traveller does not consign travel with it) | the traveller | arrival (5.4) |
| a **stored part** | the part of a limited account's load that sits above its base room, up to the capacity of its stores (`spec-storage.md` 8.3.2) | the owner of the stores | the end of the tick (6) |

3.1.2. **Not exposed:** goods in a household's base room (6.1), money in any account,
goods in an estate, goods in an Outside account (swept to zero each tick,
`spec-external-sector.md` 5.3), goods in the market (an estate's ask is still in the
estate's account), a reservation, a pending output (it is in no account,
`spec-ledger.md` 8.2.2), and an escrow (`spec-merchants.md` 9.6).

### 3.2 The route row and the store row

3.2.1. A **route row** is keyed by a configured directed edge and a transport mode:

```go
// RouteRisk is the risk a shipment of one mode runs on one directed edge.
type RouteRisk struct {
	From, To   string // region IDs of a configured edge (spec-regions.md 4.1)
	Mode       string // a transport mode ID (spec-transport.md 3.1)
	StrikeBP   int64  // probability of a strike, per shipment on the edge, 1 to 10 000
	SeverityBP int64  // share of each commodity a strike takes, 1 to 10 000
	Common     bool   // one draw per tick for every shipment dispatched on the row (3.3)
}
```

A direction and a mode are separate rows: a road that is dangerous both ways, or for a
slow mode only, is as many rows as it has directions and modes. A mode with no row on an
edge runs no risk there.

3.2.2. A **store row** is keyed by a region and a store class:

```go
// StoreRisk is the risk the stores of one class in one region run each tick.
type StoreRisk struct {
	Region     string // a region ID that is not an outside node
	Class      string // a store class ID (spec-storage.md 3.2)
	StrikeBP   int64  // probability that a store of the class is struck in a tick, 1 to 10 000
	SeverityBP int64  // share of the stored goods in the struck store a strike takes, 1 to 10 000
	Common     bool   // one draw per tick for every store of the class in the region (3.3)
}
```

3.2.3. Both are tables of the one override mechanism of `spec-regions.md` 5.1, in the
**replace** mode: one row per key (`ErrDuplicateID`), absence is "no risk" with no
warning, rows sorted by (region, then the other keys) before the hash and before a walk,
resolved once at construction into dense arrays and never saved (5.3.1 to 5.3.5 there).
The tables are rows of the list of `spec-regions.md` 5.1.3 (section 19).

### 3.3 Unit and common rows

3.3.1. A **unit** row (`Common` false) is drawn once for each exposure that meets it:
once per shipment that crosses the edge in the mode, and once per store of the class. Two
shipments on one lane in one tick are struck or spared independently.

3.3.2. A **common** row is drawn once per tick, whatever is exposed, and its outcome is
the outcome of every exposure that meets it that tick: the weather of a lane, the flood
of a valley. It is the only correlation the module has. It is a flag and not a second
kind of row so that a pack states the same hazard both ways without a new mechanism, and
a register entry (RSK-A3) can say what correlation does.

3.3.3. A common row **draws whether or not anything is exposed**: the hazard is a
property of the lane or the region, not of its traffic (position 13), and a count of
draws that depended on who happened to be on the road would be a function of a
strategy's choices that no paired comparison could keep aligned (11.4).

### 3.4 The effective strike probability and the draw

3.4.1. The **effective probability** of a row is

```
P = min( 10 000, rhe( StrikeBP * Scale, 10 000 ) )
```

with `Scale` the value of the row's slot (9), 10 000 when no call has set it, and `rhe(a,
b)` the quotient of a non-negative `a` by a positive `b` rounded half to even (the
tie rule of `spec-clearing.md` 5.2, which there rounds a midpoint; here it rounds a
quotient by 10 000, and is a rule constant of this module). `StrikeBP * Scale` is at most
`10 000 * MaxShockBP` = 10^9, below `int64`. A row with `P = 0` (a scale of 0) is not
drawn and strikes never (3.4.3).

3.4.2. **A strike** is the event that one word of the world's source, reduced by the
world's draw to `[0, 10 000)` (`spec-world.md` 7.2.1: the high word of `Uint64() *
10 000`), is **below** `P`. The probability is `P / 10 000` up to the bias of the
world's draw, at most `10 000 / 2^64` (7.2.1 there). A `P` of 10 000 strikes always and
still consumes its word (a bound is a bound: the stream does not depend on a value that
happens to be certain, `spec-world.md` 7.2.1, BEL 6).

3.4.3. **A row with `P = 0` takes no word.** This is the one place where the count of
draws depends on a value that a call between ticks can set, and `spec-world.md` 7.4.1
allows it: the count is a function of the configuration, the state, the extensions and
the calls made between ticks. The consequence is stated in 11.3: a module on with every
scale at 0 equals the module off, and a paired comparison across scales above 0 stays
aligned while a comparison with 0 does not.

3.4.4. **The severity is not scaled.** A slot scales the probability of a strike and not
what a strike takes (open question 2): a consumer that wants a worse storm sets a scale
above 10 000, which makes a strike likelier and a common one certain, and cannot make one
strike deeper. The reason is that two numbers a slot moves would be a pair that must be
read together, and the severity is a fact of the kind of hazard (a wreck takes the cargo,
a theft takes a part), where the probability is a fact of the season.

### 3.5 No `region_bound` refusal (N-090)

3.5.1. N-090's rule says a module that keeps an agent's account in a region while the
agent is away or committed registers a `region_bound` refusal. This module keeps nobody
anywhere: a cargo is exposed because it is in flight, which `open_shipment` already binds
(`spec-transport.md` 5.5.3), and a store is exposed because it stands, which `has_store`
already binds (`spec-storage.md` 9.2). Neither is made by this module, so none is
registered. A relocation is not exposed at all (open question 7).

## 4. Route risk at dispatch

### 4.1 What a dispatch does

4.1.1. A shipment is dispatched at P3.2(c) in plan order and at P3.3 in formation order,
by the application of `spec-transport.md` 5.5.1 after every refusal check has passed
(5.4 there). This spec inserts one step between the record (step 4) and the mark of
"away" (step 5):

> **4a. The risk.** With `Modules.Risk` present, walk the path of the shipment by
> `NextHop(from, to)` of `spec-regions.md` 4.3.1 on the graph as it stands (the edge
> statuses of the call last made), over the world's own next-hop table (`spec-regions.md`
> 4.4.1: a read that allocates nothing, where `Path` returns a fresh slice, 4.3.3), and for
> each consecutive pair `(a, b)` of it in path order, if a route row
> `(a, b, mode)` exists with `P > 0` (3.4.1): if the row is `Common`, its outcome is the
> outcome of P3.1a (4.2); otherwise take one word (**D14**) and the outcome is a strike
> if it is below `P`. For each struck row, in path order,
>
> ```
> kept  <- rhe( kept * (10 000 - SeverityBP), 10 000 )        kept starts at 10 000
> ```
>
> and `LossBP = 10 000 - kept` is stored in the shipment record.

4.1.2. A path with no row, and a shipment of a mode with no row on its path, has
`LossBP` 0 and takes no word. The path is the one of `spec-transport.md` 5.4 check 7
(`no_route`): the path that gave the travel time. Its edges at their statuses are the
rows' keys; a row on an edge that is closed is never met until it reopens (a closed edge
is not on a path).

4.1.3. **A refused request takes no word.** Every refusal of 5.4 there is before step
4a, so a request that is refused leaves no trace and takes no draw (`spec-transport.md`
6.4). A formation of the contract round that fails at dispatch likewise
(`spec-merchants.md` 9.5).

4.1.4. **Nothing is changed at dispatch but the record.** The cargo is moved to the
Transit account as before, the haul units or the feed are used as before, and the
shipment keeps its arrival tick. A strike does not delay a shipment and does not stop it:
what a strike does is at arrival (5).

### 4.2 The common rows: P3.1a

4.2.1. At **P3.1a**, a new position of phase 3 after P3.1 (`Inbound` and `Outbound` are
fixed) and before P3.2 (the first plan), with the module on, take one word (**D13**) for
each route row that is `Common` with `P > 0`, in ascending (`From`, `To`, `Mode`) as
`RegionID`, `RegionID` and the mode's index in the configuration's order. The outcome
(a strike or not) of each is kept for the tick in a scratch vector, derived and never
saved: a checkpoint is taken between ticks, and the vector is drawn again at the next
P3.1a.

4.2.2. A common row is drawn **for every tick**, so that a tick with no shipment takes the
same words as one with many (3.3.3). Its outcome is read by every dispatch of the tick
(4.1.1) on a path that crosses the row, whatever their order.

### 4.3 The combined share

4.3.1. The kept share is a product over the **struck** rows of a path, rounded at each
step. The order is the path's, and it fixes the rounding. Two rows of 2 500 and 4 000
basis points both struck keep `rhe(10 000 * 7 500, 10 000) = 7 500` then `rhe(7 500 *
6 000, 10 000) = 4 500`: `LossBP` 5 500. The intermediate `kept * (10 000 - SeverityBP)`
is at most 10^8.

4.3.2. **A strike cannot take more than the cargo.** `LossBP` is at most 10 000 − the
smallest `kept` reachable: with a `SeverityBP` of 10 000 on any struck row `kept` is 0
and `LossBP` is 10 000. A shipment with `LossBP` 10 000 is a **total loss**: everything
in its Transit account is lost at arrival (5.2). It is still dispatched and still
arrives as an event (a ship that sank is told at its arrival tick: the consignee learns
no sooner than the arrival, `spec-transport.md` 9.3).

4.3.3. **A strike with no loss.** A share that rounds to zero of every commodity
(5.3) is a strike with no loss, counted in `LossBP` (above 0) and showing in the event
with empty `Lost`. This is not an error: a small line meets a small share (5.3.2).

### 4.4 What the shipper does not see

4.4.1. `LossBP` is in the shipment record and in `ShipmentDispatched`, which an
observer reads. **No view carries it**: `Outbound` (`spec-transport.md` 9.1) lists the
quantity and the arrival tick of what is on the way, and not its fate. A strategy that
could see a strike at dispatch would also see it before the consignee does, and could
buy the loss's price effect ahead of the arrival; the information of a loss is the
arrival's (`spec-transport.md` 9.3). What a strategy does see is the **expected**
loss of the route before it ships (8).

## 5. Loss at arrival

### 5.1 The position

5.1.1. In P1.3, for each shipment with `Arrives = t` in ascending shipment ID, the
arrival of `spec-transport.md` 5.7 gains a **step 0, the loss**, before its step 1 (the
traveller's region change) and step 2 (the delivery):

> **0. The loss.** If `LossBP > 0`: for a shipment with a Transit account, for each
> commodity the account holds, in commodity order, `lost = rhe(held * LossBP, 10 000)`
> units move from the Transit account to the boundary under `loss`, oldest lots first.
> For a self-carried shipment whose traveller still exists, the same is then done over the
> traveller's own account (5.4). The arrival then goes on with what is left.

5.1.2. **Why before the delivery.** The consignee receives what survived. The check of
`spec-transport.md` 5.7 step 2 (whether the consignee can hold the quantity, else the
cargo is diverted) is then made on the surviving quantity, so a loss can make a
diversion unnecessary and never makes one necessary.

### 5.2 A cargo

5.2.1. `held` is what the Transit account holds at arrival, which is what was shipped
less what spoiled (`spec-transport.md` 5.6.1, 7.3.2): spoilage and loss do not compound
on a lot, because a lot that spoiled in a tick before the arrival is already gone, and a
lot that spoils in the arrival tick spoils at P8.1 of that tick, after the delivery.

5.2.2. The per-line identity of `spec-transport.md` 7.3.2 becomes exact (7):

```
shipped = delivered + diverted + spoiled + lost
```

with `spoiled = shipped - held` and `lost = rhe(held * LossBP, 10 000)` (5.1.1).

### 5.3 The rounding rule

5.3.1. The loss of a commodity is `rhe(held * LossBP, 10 000)`, half to even, a rule
constant. `held` is at most `MaxQuantity` (10^9) and `LossBP` at most 10 000, so the
product is at most 10^13, below `int64`.

5.3.2. **What it does to a small line, stated.** A line of 1 unit loses it if and only if
`LossBP > 5 000` (at exactly 5 000 the half goes to the even 0); a line of 3 units under
`LossBP` 2 500 loses `rhe(7 500, 10 000) = 1`; a line of 3 units under 1 500 loses
`rhe(4 500, 10 000) = 0`. The loss of a line of one unit is therefore a step function of the
share (0 below 5 000, 1 above) and not the share itself. For a line of `q` units the loss
is within half a unit of `q * share / 10 000`; over lines of many sizes the errors cancel
(the mean over `q` = 1 to 20 is at most 0.025 units a line, section 17 part 5), and over
many one-unit lines they do not. A rule that kept the expectation (a carried remainder, or a draw per unit)
would add state or draws to a module whose point is to add few; section 17 measures the
bias and open question 1 records the alternative.

5.3.3. **Which lots.** The ledger takes the oldest lots first and then the lowest
remaining durability (`spec-ledger.md` 5.2.1, FR-LED-012): a rule of the ledger and not a
choice here. A loss does not choose a spoilable lot to hit.

### 5.4 The traveller's own account

5.4.1. A self-carried trip carries the whole of the traveller's account
(`spec-transport.md` 3.2.2, 5.1). When the consignee is the traveller itself there is no
Transit account and no line, and the account is all the trip has; when it consigns lines
to another agent, the lines are in a Transit account and the rest of the account still
travels with it. In both cases the account is exposed: at arrival, after the loss of the
Transit account if there is one and before the region change, `lost = rhe(held *
LossBP, 10 000)` of **every commodity the traveller's account holds except the mode's
`Vehicle` commodity**, in commodity order, moves from the account to the boundary under
`loss`. The units of the `Vehicle` are not cargo (`spec-transport.md` 3.2.2: they are
left out of the load of the same trip), so a strike takes none of them: a vehicle is
capital, and the loss of capital is not in version 1 (1, 2 position 2, 10.3 CAP row). The
feed of the trip is used at dispatch under `transit_use` (`spec-transport.md` 3.3.2) and is
not in the account at arrival. The goods the traveller eats and every other good it holds
are exposed: a strategy that keeps its holdings small keeps its exposure small. The
holdings are **not** priced by the merchant (8.4.2). A hired shipment has no traveller and no such
step.

5.4.2. The traveller is then in the destination market in the arrival tick's phase 3
and buys what it lacks like any local agent, at the local price and with the money it
has; upkeep is eaten at P6.1, after P3.2. A loss can therefore leave a traveller short of
what it eats and unfed only if it cannot afford the difference, which is the ordinary
failure of any household (the privation count and the ladder, `spec-spawner.md` 5).

5.4.3. **A traveller that was removed** before arrival has its holdings in the origin
estate (`spec-transport.md` 5.9.2), so the step over its own account has nothing to take
and is skipped. Two cases, as TRN 5.9.2 has them: a **shipment to itself** is dropped, so
no loss is applied and no `ShipmentArrived` is emitted for it, as before; a **shipment
with lines** continues without a traveller, arrives at `Arrives` and emits its event, and
step 0 of 5.1.1 strikes its Transit account as for any shipment (5.5.1), with an empty
`AccountLost`.

### 5.5 A diverted cargo and a removed consignee

5.5.1. The loss is applied to the Transit account whatever happens to the consignee. If
the consignee was removed or cannot hold what survived, the surviving quantity is diverted
to the estate of the destination (`spec-transport.md` 5.8) as before. A cargo whose
`LossBP` is 10 000 has nothing to deliver and nothing to divert: the event says
`Diverted` false and every line's `Lost` equals its `Shipped` less `Spoiled`.

### 5.6 A delivery contract

5.6.1. The units that **reached** the payer are the delivered and the diverted
(`spec-merchants.md` 9.7), so a lost unit is a unit that did not reach. The contract's
table is unchanged in structure: `escrow_out` pays the deliverer `p * reached` and
`escrow_refund` returns `p * (q - reached)`; the units that did not reach are now the
spoiled **and** the lost. **The deliverer bears the loss** as it bears the spoilage
(`spec-merchants.md` 9.7, "the deliverer bears the spoilage"): it chose the lane, and
the expected loss of the lane is in its view (8.1). `ContractSettled` gains a field
`Lost` beside `Spoiled` (the cause stays `delivered` or `short` as MER defines them;
the edit is owed, section 19).

5.6.2. A shipment that is not a contract's (goods consigned to another agent, a merchant's
own cargo carried to itself) leaves the loss with whoever was to receive it: the consignee
receives less, and a traveller that is its own consignee arrives with less.

## 6. Loss in storage

### 6.1 What a strike on a store reaches

6.1.1. **Who is exposed.** A limited account (`spec-storage.md` 3.3) of an agent that
owns at least one store of a class for which a store row exists for the agent's region
with `P > 0`, and whose stored load is above 0 in at least one dimension at P8.1a. A
pseudo-agent owns no store (`spec-storage.md` 3.3.3). A household's base room is not
exposed (position 2; open question 6).

6.1.2. **What share of the account a store holds.** A good has no place inside an
account: the room is added to the account and the goods are one set of lots
(`spec-storage.md` 3.1, 8.1.2). The exposure of a store is therefore a **share** of every
commodity, never a chosen set of goods. For a limited account with the stores `s_1 ..
s_n`, `SumCap_d` the sum of their capacities in dimension `d` (mass or bulk), `Stored_d`
the stored load of `spec-storage.md` 8.3.2 and `Load_d` the account's load
(`spec-storage.md` 3.4.1), a store of class `k` holds the part of the account's load

```
x_d   = floor( Stored_d * Cap_d(k) / SumCap_d )        0 when SumCap_d = 0; 128-bit first product
phi_d = floor( x_d * 10 000 / Load_d )                 0 when Load_d = 0
phi   = max( phi_mass, phi_bulk )                      basis points, per store of the class
```

`Stored_d` is at most `SumCap_d` and `Cap_d(k)` is below 2^63, so the quotient of the
first line fits and the product is computed with `math/bits` (architecture.md 3.5, the
checked helpers of `internal/`). The second line is at most 10^12 times 10^4.

6.1.3. **Why the larger of the two.** The load is a pair and a good has no location, so
the share of the account that sits in a store is a pair of fractions with no single
answer. The larger is the cautious reading (it exposes more, not less) and it is a rule
constant; a pack whose stores are limited by one dimension only has one non-zero term.

### 6.2 The draws, in order

6.2.1. At **P8.1a** (11), after ageing and spoilage have run (P8.1) and before the flow
totals are final (P8.2), with the module on:

1. **The common rows.** For each store row with `Common` and `P > 0`, in ascending
   (`RegionID`, `StoreClassID`): one word (**D15**). The outcome is kept for the
   position in scratch.
2. **The stores.** For each limited agent in ascending agent ID, for each class it owns
   with a store row of its region with `P > 0` and `Common` false, in ascending
   `StoreClassID`, for each store of the class (an agent owns at most `MaxStoresPerAgent`,
   `spec-storage.md` 12.2), one word (**D15**) if the agent's stored load is above 0 in at
   least one dimension. A store whose owner has no stored load is not drawn and is not
   struck.

A struck store is a store drawn below `P`, or of a class whose row is `Common` with a
struck outcome. For a common row a store is struck if the agent has the store and its
stored load is above 0: the outcome is the row's and the word was taken in step 1.

6.2.2. The order and the count are a function of the configuration, the state at the
position and the slots, never of an outcome (`spec-world.md` 7.4.3): the strikes are
decided first and applied after (6.3.1), so the loss of an earlier agent does not change a
later agent's stored load or its words.

### 6.3 The loss

6.3.1. **Decide, then apply.** All outcomes of 6.2.1 are drawn before any loss is applied,
and the stored fractions of 6.1.2 are computed on the state at the start of P8.1a, so that
what one agent loses cannot change the draws or the share of another, and a second struck
class of one agent is applied to the quantity it had at the start.

6.3.2. For an agent with `n_k` struck stores of class `k`, the **share** of class `k` is

```
share_k = min( 10 000, n_k * rhe( phi_k * SeverityBP_k, 10 000 ) )          basis points
```

and for each commodity `c` the agent holds, with `q0_c` its quantity at the start of
P8.1a, the loss of class `k` is `rhe(q0_c * share_k, 10 000)`; the classes are applied in
ascending `StoreClassID`, each moving its units from the agent's account to the boundary
under `loss`, oldest lots first, and the total taken of a commodity is capped at `q0_c`.
The share of a strike is a share of the **account's** goods and not of the store's
capacity: the stored fraction `phi` already says how much of the account the store holds.

6.3.3. **What it does not do.** A strike does not destroy the store, change its capacity,
end its levy, change the owner's room or its refusal `has_store`. Goods are lost and the
room stays (a facility's destruction or decay is CAP's, `spec-storage.md` 3.6.3). The
agent's targets and beliefs are not told: next tick it holds less, its stock ratio falls
and its belief rule and its default strategy respond as they do to any shortfall
(RSK-A2). A loss is not an order outcome and is never an input to `Revise`.

6.3.4. **Cohorts.** The stores of an agent of weight `w` are the agent's, not multiplied
by `w` (`spec-storage.md` 3.3.2), and a strike takes a share of the agent's whole account,
so every household of a cohort loses the same share (a cohort has no within-group
inequality, `spec-scale.md` 5.1).

### 6.4 The event

6.4.1. For each (agent, class) with at least one struck store, in ascending agent ID then
class, `StoreLoss` is emitted after the whole position has been applied (14.1): the
agent, its region, the class, the number of stores struck, whether the row is common, the
share and the units lost by commodity (empty if every product rounded to 0, the strike
with no loss of 4.3.3).

## 7. The ledger's part

### 7.1 The reason `loss`

7.1.1. The registry of `spec-ledger.md` 7.2 already has the row

| Reason | Of | Kind | Between | Owner |
|--------|----|------|---------|-------|
| `loss` | goods | sink | Transit account to boundary; agent account to boundary | this spec |

with one change of meaning to the row as the ledger has it (section 19): the money form is
**not used** in version 1 (7.4). The row's owner is already this spec (`spec-ledger.md`
7.2); the sentence of `spec-transport.md` 7.2 that lists "owners `spec-transport.md` and
`spec-estates.md`" is the one to correct. The estate form (an estate account to the
boundary) is not used either: an estate's goods are on sale for the pool and a
pseudo-agent owns no store and carries no cargo.

7.1.2. A transfer under `loss` is always a **sink**: its counterpart is the boundary and
the boundary balance of the reason is `-Σ lost` over the run. It is a boundary flow like
`spoilage` and `wear`, reported by `BoundaryFlow` (`spec-ledger.md` 12.1) and counted by
the audit (`spec-ledger.md` 9.1) with no rule of this spec.

### 7.2 Attribution

7.2.1. The ledger attributes a flow to the region of the account on the non-boundary
side (`spec-ledger.md` 3.5). A cargo's loss is therefore attributed to the **origin**
region (the region of a Transit account is the origin of the shipment that opened it,
`spec-transport.md` 7.1.2, 7.3.3); a trip account's loss to the traveller's region at that
time, which is the origin (the region change comes after the loss, 5.4.1); a store's to
the region of the account. A report by region that wants the loss by destination reads
`ShipmentArrived`.

### 7.3 The identities this spec adds

For a commodity and a run, exactly and with no tolerance (testing.md 6.3):

1. **Per line**: `shipped = delivered + diverted + spoiled + lost` (5.2.2).
2. **Per commodity and run**: the boundary balance of `loss` for the commodity equals
   minus the sum of `Lost` over every `ShipmentArrived` line, `AccountLost` of every
   `ShipmentArrived` and `Lost` of every `StoreLoss`, over the events of the run.
3. **Per tick and region**: the `BoundaryFlow` of the reason `loss` for a (region,
   commodity) equals the sum of the same events attributed to that region by 7.2.1.

### 7.4 Money is not lost

7.4.1. A strike takes goods. Three reasons, each a decision: a sink of money with no
counterpart would shrink the money supply with no agent to receive it, which the first
consumer's players would read as theft by the world (and `spec-money-supply.md` 4.5
keeps every flow named and declared: a declared reason); no money is exposed to a row because the
accounts of a traveller and of a store owner hold money that the module has no way to
attribute to a place (money has no location, as goods in an account have none); and a
robbery that moves money from one household to another is a transfer between two
accounts, which needs an agent that robs (open question 3).

7.4.2. A pack that wants money destroyed by a hazard writes it as a levy of
`spec-money-supply.md` on the class that carries the money, with a rate and a limit, and
pairs it with a grant (a balanced pair, `spec-money-supply.md` 9.2). That is a rate, not a strike, and it
is MON's.

### 7.5 What the audit sees

7.5.1. Nothing new in the ledger: a transfer to the boundary keeps every identity of
`spec-ledger.md` 9.1, and the checks of `spec-transport.md` 7.4.1 hold after step 0 of
5.1.1 with the Transit account holding what the loss left. `World.Audit` also checks, with
the module on, that every in-flight shipment's `LossBP` is within `0 .. 10 000`. It cannot
check that a `LossBP` is the one the rows and the words would give: the words are the
stream's and the scales are not state of the shipment.

## 8. What an agent and a lane see of a risk

### 8.1 The expected loss of a route

8.1.1. The **expected loss** of a route `(from, to, mode)` is a share of a cargo in basis
points, a pure function of the configured rows, the scales and the graph at the call:

```
path   = the walk by NextHop(from, to) on the current graph (a `Path` may be used here:
         this is at a call, not in the tick, 8.1.3)
K      = 10 000
for each consecutive pair (a, b) of path in order, with a row (a, b, mode) with P > 0:
        f = 10 000 - rhe( P * SeverityBP, 10 000 )
        K = rhe( K * f, 10 000 )
LossBP = 10 000 - K
```

It is the integer approximation of `1 - Π(1 - p_i s_i)`, the exact expected kept share of
independent rows, rounded at each step; a common row has the same expectation as a unit
row (the correlation is between shipments and not within one). It is a number for a
decision and for a read, never for the ledger. No path, no module, or no row on the path:
0.

8.1.2. **It is shown to every agent**, as the route's travel time is: `Routes`
(`spec-transport.md` 9.1) gains `LossBP` for each (region, mode), 0 with the module off.
A road's danger is a structural fact of the world, like its length, and a strategy that
reads it learns nothing about another region's households, wages or stock
(`spec-migration.md` 4). It includes the scales, so a consumer's shock (a surge on a
lane) is in what the shipper reads at once: the shock is exogenous, and a strategy that
did not see it would price a lane on a dead number.

8.1.3. **It is derived and cached with the graph.** The table of expected losses by
(from, to, mode) is rebuilt at the call that changes an edge status or a scale (as the
graph tables and the lag table are, `spec-regions.md` 4.4.4) and nowhere else; a lookup
in the tick loop is a read and allocates nothing. It is not saved.

### 8.2 The read

8.2.1. The read is **TRN's** (`spec-transport.md` 9.2, beside `Travel`, whose formula is
8.1.1 here) and **INT lists it** in its table of queries (`spec-integration.md` 6.2, the row
of `InTransit`, `Travel` and `Away`; section 19):
`RouteLoss(from, to RegionID, mode ModeID) (lossBP int64, ok bool)`, `ok` false for no
path, an unknown key or the module off, a pure function of the world at a boundary that
draws nothing. Its cost class is 1: a lookup in the cached table of 8.1.3.

### 8.3 The lanes of the viability solver

8.3.1. `transport.Lanes` (`spec-transport.md` 9.4) derives for each hired mode and ordered
pair a lane "with no loss (RSK adds loss)". With the module present the lane's **loss**
is the expected loss of 8.1.1 computed from the configured rows at the scales of 10 000
(`Lanes` is a pure function of the configuration, so a scale slot is not read) over the
path of the configured graph with every edge open. The solver takes it as a share of the
lane's output that never arrives (`spec-viability.md` 9.4); `Lanes` stays a function of
the configuration and does not read this module's state. The edit is owed to TRN 9.4.

### 8.4 The merchant

8.4.1. The merchant of `spec-merchants.md` reads a rumour for the revenue and `Routes`
for the cost. The loss belongs to the revenue: of every unit it ships, the share
`LossBP / 10 000` does not arrive. The edit owed to `spec-merchants.md` 5.2.1 is

```
Rnet <- Rnet * (10 000 - LossBP) / 10 000          truncated, LossBP of Routes[to][mode]
```

applied after the consignee's cut (and for a self-carried mode to `Rnet = Rhat`), so
every quantity that `spec-merchants.md` derives from `Rnet` is net of the expected loss:
`G` and the fee-share test (5.5.1), `BreakEven`, the ceiling of the bid for the cargo
(5.5.3, 5.6) and `HaulCeiling` (5.6). `AskLimit = P_acq + margin_needed` (5.5.3) holds no
`Rnet` and is not changed: it is the cheapest delivered price at which the lot passes, a
function of the cost `C`, the acquisition price and the lot, and the loss reaches it only
through the size of the lot, which MER 6.2.2 sets with `BreakEven`. The loss is therefore **in the margin and not
in the cost of carriage**: a lane that loses a quarter of its cargo has a revenue a
quarter lower and the same fee. A merchant with a `LossBP` it cannot read (a consumer's
own strategy that ignores the field) ships at a margin that is not there, which is
RSK-A1's case.

8.4.2. **What is not priced.** The expected loss is that of the **lot**: the cargo of the
route. A self-carried trip also exposes the rest of the traveller's account (5.4.1), and
the merchant's cost `C` (`wear + feed + time`) holds no term for the same share of its
holdings. That exposure is not priced in version 1: a strategy that keeps its holdings
small keeps it small, as 5.4.1 says, and a pack that wants it priced raises the lane's
`StrikeBP` or `SeverityBP`. "A margin that survives its own expected loss" (RSK-A4) is
therefore a statement about the lot and not about everything the traveller carries.

### 8.5 What a view never shows

8.5.1. No view carries a shipment's `LossBP`, a strike, a draw, a row's `Common` flag or
`StrikeBP`, a slot's scale as such (it is in `LossBP`), or a store's expected loss. A
consumer holds the configuration (`Config()`) and can read the rows; an agent of the
library reads only `Routes.LossBP`.

## 9. Slots

### 9.1 The two calls

9.1.1. Two slots scale the strike probability of a row. They are world state set between
ticks, over the configured value, not configuration, not hashed, saved in a checkpoint
(`spec-world.md` 8.1, 8.2), and the public calls are INT's (`spec-integration.md` 5.1,
section 19 gives its rows):

```go
// RouteRiskScale scales the strike probability of the route row (From, To, Mode).
type RouteRiskScale struct {
	From, To RegionID
	Mode     ModeID
	BP       int64 // 0 to MaxShockBP; 10 000 restores the configured probability
}

// StoreRiskScale scales the strike probability of the store row (Region, Class).
type StoreRiskScale struct {
	Region RegionID
	Class  StoreClassID
	BP     int64 // 0 to MaxShockBP; 10 000 restores the configured probability
}

func (w *World) SetRouteRiskScales(scales []RouteRiskScale) error
func (w *World) SetStoreRiskScales(scales []StoreRiskScale) error
```

Both are atomic slices of entries, one per key, as every setter of INT 3.3 is.

9.1.2. **What a scale acts on.** It multiplies `StrikeBP` (3.4.1) and nothing else: not
the severity (3.4.4), not the `Common` flag, not the row's existence. It cannot create a
risk where there is no row: a key with no row is refused (9.2) so that a consumer's
mistake is found at the call, as INT 5.1.3 does for every slot.

### 9.2 Errors, in the order checked

| Call | Errors |
|------|--------|
| `SetRouteRiskScales` | `ErrModuleOff` (risk module off), `ErrUnknownRegion` (`From` or `To` outside `[0, n)`, `NoRegion` included), `ErrUnknownMode`, `ErrUnknownRiskRow` (new: the key is not a configured route row), `ErrOutOfBounds`, `ErrDuplicateID` |
| `SetStoreRiskScales` | `ErrModuleOff`, `ErrUnknownRegion`, `ErrUnknownStoreClass` (STO's), `ErrUnknownRiskRow`, `ErrOutOfBounds`, `ErrDuplicateID` |

An entry equal to the default (10 000) is cleared (`spec-integration.md` 5.2.1), a change
emits an event per changed key in key order (14.1), an entry equal to the current value
is accepted and is no event. No timer: a scale holds until another call sets it
(`spec-integration.md` 5.2.2).

### 9.3 When a scale is read

9.3.1. A route scale is read at **P3.1a** (the common rows' probability) and at the
dispatch record (4.1.1, the unit rows) and at the call that rebuilds the expected-loss
table (8.1.3). A store scale is read at **P8.1a**. A shipment in flight keeps the
`LossBP` it was given at dispatch (4): setting a scale does not touch what is on the
road (the rule of every slot, `spec-integration.md` 5.2.3).

### 9.4 What a consumer uses them for

9.4.1. A surge of banditry on a road for a season; a storm season, as a common row raised
and cleared; a flood: a store row's scale to 100 000; a patrol: a scale down to 2 000. The
scale is also the means by which a consumer's controller closes a loop (a patrol that
follows the loss rate): the engine guarantees conservation and determinism of every
setting and not the stability of the controller (N-131, L121).

## 10. The embargo, the closed road and the seams

### 10.1 The embargo is an edge status

10.1.1. A closed road, a blockade and an embargo are one thing in the library: an edge
whose status is closed, or slowed by a factor (`spec-regions.md` 7). Its public call is
`SetEdgeStatuses` of INT (`spec-integration.md` 5.1; the cause is `consumer`); a
jurisdiction's border gives the edges to set through `Between(g, a, b)` (`spec-polity.md`
6.3, H10 and H13). The internal form of the call keeps a `cause` (`spec-regions.md`
7.3.3) so that a later module can set a status with its own; this spec has none.

10.1.2. **What a closure does to cargo is TRN's**: a closed edge is left out of the paths, so a new request is refused `no_route` only
when no other path is open (`blocked` is the other refusal, a transit factor of 0,
`spec-transport.md` 6.3, FR-TRN-006); a shipment in flight keeps its arrival tick
(`spec-transport.md` 4.3) and a closure of the gate edge of an outside node cuts its
lines (`spec-external-sector.md` 7.5). This spec adds nothing: a closure does not seize what is
in flight, and does not raise the probability of a strike on the other roads (no
endogenous risk, position 13).

10.1.3. **No autonomous closure.** A road that closes by chance (a pass in the snow, a
bridge washed away) would be a status set inside a step, at a position that the tables
(`spec-world.md` 8.2: rebuilt at the call) do not have. It is open question 4.

### 10.2 Insurance (INS)

10.2.1. What `spec-insurance.md` may read, and when:

| Read | Source | When | Use |
|------|--------|------|-----|
| the expected loss of a route | `RouteLoss`, 8.2 | any time at a boundary, or at a position of INS | a premium |
| the loss of a shipment | `ShipmentArrived`: `Lost` of each line and `AccountLost` | P1.3, at the arrival | a loss report (a payout is INS's) |
| the loss of a store | `StoreLoss` | P8.1a | a loss report |
| the rows | `Config().Modules.Risk` | construction | the correlation: a `Common` row is a risk that strikes every insured at once |

10.2.2. **What INS must not read**: a shipment's `LossBP` before its arrival. The field is
in the record and in `ShipmentDispatched` for an observer, and it is the world's fact and
not an agent's information: an insurer that read it at dispatch would insure only the
cargo that is lost (adverse selection by construction). An insurance module reads the
expected loss and the arrival's event, and a test of INS must assert it (section 19).

10.2.3. **Correlation** is the property of insurance that this spec supplies as data: the
losses of unit rows diversify (their sum has a variance that grows with the count of
shipments and not its square), and the losses of a common row do not (the whole book of a
lane is struck at once, RSK-A3). A bust of every insurer at once is a common row.

### 10.3 The other seams

| Spec | What it may do | What this spec supplies | Status |
|------|----------------|-------------------------|--------|
| STO | owns the store and the stored load | the read of `Stored_d` and the stores by class (`spec-storage.md` 8.3.2); a store row needs `Modules.Storage` | used |
| TRN | owns the shipment and the path | step 4a, step 0, `Routes.LossBP`, `Lanes` (4, 5, 8) | used |
| MER | the shipper and the contract | the expected loss in the margin (8.4); `Lost` in `ContractSettled` (5.6) | edits owed |
| POL | a jurisdiction pair as a blockade | nothing: H10 and H13 call `SetEdgeStatuses` | used |
| SET | a size class as a defence | **declined** (open question 5): H7 of `spec-settlements.md` 6.3 stays reserved. R2 of SET (a limit is a count, never a rate) forbids reading a class as a probability; a flag could scale a row, which is a gate that removes a risk and not a barrier to entry | reserved |
| CAP | a facility that lowers a probability (a watch, a harbour); a facility lowered by a strike (a raid on a bridge) | nothing in version 1: CAP writes no slot and has no internal setter (`spec-infrastructure.md` 5.5 and 4.1: a facility is a layer, not a slot; a fifth kind of effect needs a spec that owns it, here RSK). The damage is **not** done by this module in version 1: `AdjustFacilities` stays the consumer's call (open question 13 gives the reserved shape, answering `spec-infrastructure.md` 21 item 7) | reserved |
| FIS | a protection payment, a toll | nothing: a levy and a strike are separate | none |
| LND | loss of a title by conflict | nothing in version 1: a title is not goods and this module has no actor; the call of `spec-land.md` 6.3 (`TransferTitles`) stays the consumer's, and the two LND texts that say "a cause RSK names" are edited (section 19) | none |
| INS | premium, payout | 10.2 and the scratch of the lost units of an arrival and of a strike (`spec-insurance.md` 5, 8), kept with or without an observer | used |
| PLC | a policy actor's levers | nothing: `SetRouteRiskScales` and `SetStoreRiskScales` are not levers (a hazard is a story shock, `spec-policy.md` 6.3.3) | none |
| EXT | the outside's lines | nothing: no shipment to an outside node, no store of the outside | none |

## 11. Position in the tick and draws

### 11.1 The positions

11.1.1. The module adds two positions to the table of `spec-world.md` 6 and one step to
two existing ones; nothing is renumbered (`spec-world.md` 6.1.1, 6.1.10). A module that
is off has none of them (6.1.5 there).

| ID | What runs | Reads / writes | Draws | Events |
|----|-----------|----------------|-------|--------|
| **P3.1a** (new, after P3.1) | the common route rows: for each with `P > 0`, ascending (`From`, `To`, `Mode`), the outcome of the tick (4.2) | the rows, the route scales; writes the tick's outcome vector (scratch, derived) | **D13**, one per row | none |
| P3.2(c), P3.3 (TRN 5.5.1 **step 4a**, new) | the risk of a dispatched shipment: the path's rows in path order, `LossBP` into the record (4.1.1) | the rows, the scales, the path, the P3.1a vector; writes `LossBP` | **D14**, one per unit row of the path with `P > 0` | `ShipmentDispatched` carries `LossBP` |
| P1.3 (TRN 5.7 **step 0**, new) | the loss of an arriving shipment: its Transit account, then the traveller's account (5.1, 5.4) | the shipment, the account; writes lots, boundary balance `loss` | none | `ShipmentArrived` carries the losses |
| **P8.1a** (new, after P8.1) | the stores: the common rows, then the stores of each limited agent in ID order; the losses applied after all are decided (6.2, 6.3) | store rows, store scales, accounts, `Stored_d`; writes lots, boundary balance `loss` | **D15** | `StoreLoss` per (agent, class), ascending |

11.1.2. **Why P3.1a.** The common outcome of a lane is the weather of a tick: known before
any agent's plan is executed, so that the order of dispatches within the tick gives no
shipment a different weather. P3.1 fixes `Inbound` and `Outbound` (derived from shipments
dispatched before the phase) and is untouched by a risk.

11.1.3. **Why P8.1a.** After ageing (P8.1), so that a strike takes what the clock left
and a lot that spoiled this tick is not lost twice; before P8.2 (the flow totals are
final), so that the tick's `BoundaryFlow` includes the loss of the tick; after
consumption and the welfare record (phase 6), so that an end-of-tick strike does not
change what a household ate and the record of the tick shows the tick as it was lived. The
goods are gone before P3.2 of the next tick, where the household decides on what it has. CAP's facility step is **P8.3b** (`spec-infrastructure.md` 6.3): after regeneration (P8.3) and after LND's rent step P8.3a (`spec-land.md` 8.1), before the audit. It follows P8.2 and claims nothing of P8.1a, and no letter of P8.3 clashes with a letter of P8.1.

### 11.2 The draws

Rows for `spec-world.md` 7.3, **inserted at their position in the order of a run under
the new IDs** (the table's heading is "in the order of a run"; an ID is not an order):
D13 between D3 (P2.5) and D4 (P3.2(a)); D14 after D5, its words of P3.2(c) falling inside each
agent's turn after that agent's D5 and its words of P3.3 after D6; D15 after D11. The IDs are final (`spec-world.md` 7.3: D13 to D15 are RSK's, D16 TEC's and D17 PLC's).

| ID | When and where | Taken by | How many | In what order | Owner |
|----|----------------|----------|----------|---------------|-------|
| D13 | P3.1a | the world, per route row with `Common` and `P > 0` | 1 | ascending (`From`, `To`, `Mode`) | RSK 4.2 |
| D14 | P3.2(c) and P3.3, at the record of a dispatched shipment | the world, per route row of the shipment's path with `Common` false and `P > 0` | 1 | path order; shipments in dispatch order (agents ascending, plan order; formation order) | RSK 4.1 |
| D15 | P8.1a | the world: (i) per store row with `Common` and `P > 0`; (ii) per store of a limited agent with a unit row with `P > 0` and a stored load above 0 | 1 | (i) ascending (`RegionID`, `StoreClassID`); (ii) agents ascending, class ascending, store instance | RSK 6.2 |

The terms of `spec-world.md` 7.3.1 gain, each at its position,

```
+ common route rows with P > 0                                   D13
+ unit rows on the paths of the shipments dispatched             D14
+ common store rows with P > 0 + stores drawn                    D15
```

each a function of the configuration, the slots and the state at its position (11.3).

### 11.3 The count of draws

11.3.1. **It is a function of the configuration, the state and the slots**, never of an
observer, a logger, `WithAudit`, a snapshot, the number of `Run` calls or a checkpoint
(`spec-world.md` 7.4.1). Specifically: D13 depends on the rows with `P > 0` (the
configuration and the route scales); D14 on the shipments dispatched and the rows on
their paths; D15 on the store rows with `P > 0`, the stores and whether each owner's
stored load is above 0.

11.3.2. **Neutrality.** A world with the module on and every scale of every row at 0 (or
no row) takes no word of D13, D14 or D15, writes `LossBP` 0 to every shipment and applies
no loss: the state of every tick, every snapshot and every event of the other specs equal
those of the world with the module off, bit for bit. What differs is the module's own
surface: the fields of 14.1 (0 or empty), the module's section of a checkpoint, and the
configuration hash (the module is in the effective configuration). A test asserts it over
seeds (FR-RSK-005).

11.3.3. **A module on that has nothing to do takes no word** (`spec-world.md` 6.1.5): no
row with `P > 0`, or a unit row with no exposure (no shipment crosses it, no owner has a
stored load). A **common** row is not a case of "nothing to do": it draws every tick by
configuration, as a walking line of the outside draws every tick (D2, `spec-world.md`
7.3), because the weather is a property of the lane and not of its traffic (3.3.3).

### 11.4 What it does to a paired comparison

11.4.1. Two runs of one seed that differ in a scale above 0 take the same words in the
same positions (the count does not depend on the value) and are aligned until the first
shipment or store whose outcome differs changes what an agent later does
(`spec-world.md` 7.6; `spec-experiments.md` 7). A comparison with the scale 0 is aligned
only up to the first position where a row would have drawn. EXP reads the first as it
reads any change of a parameter that moves no draw count (`spec-experiments.md` 7.3.1),
and the second as a change of the number of draws. The module's knobs are structural
(section 12.5) and its slots are not a sweep axis in the registry of CLB.

### 11.5 The order of the events

11.5.1. Rows to append to `spec-world.md` 6.11: in phase 1 `ShipmentArrived` carries the
loss and no event of its own comes between; in phase 3 `ShipmentDispatched` carries
`LossBP` and there is no event for P3.1a; in phase 8 the `StoreLoss` events (P8.1a)
come first, in ascending agent ID then class, then `BoundaryFlow`,
`EndowmentChanged`, `TickEnded` (item 9 there). A call between ticks emits
`RouteRiskScaled` or `StoreRiskScaled` at the call, before the next `TickStarted`
(`spec-world.md` 6.11, end).

## 12. Parameters

### 12.1 The module and its types

`Config.Modules.Risk` is a pointer; `nil` is off (`spec-config.md` 3.2). Plain data, rows
sorted by key, no maps (the row types are in 3.2):

```go
// Risk is the module. Its presence switches it on.
type Risk struct {
	Routes []RouteRisk // sorted by (From, To, Mode) as RegionID, RegionID, mode index (3.2.1)
	Stores []StoreRisk // sorted by (Region, Class) as RegionID and class index (3.2.2)
}
```

12.1.1. **Replace, one row per key, absence is no risk** (`spec-regions.md` 5.3): a second
row for a key is `ErrDuplicateID` in a `Config` built in Go (a file cannot have it).

12.1.2. **Presence.** The module is on when `Modules.Risk` is non-nil. A non-nil module
with no row is legal, does nothing and warns (`no_risk_rows`).

### 12.2 The tunables and their defaults

All are fields of `Config.Modules.Risk`, with an owner (this spec), a bound and a doc line
(architecture.md 9.5, 9.7). **Every number is a placeholder** (N-015, N-027). The default
of the section as a whole is none: no row is no risk.

| Scope | Field | Meaning | Default | Bound (`bounds.go`) |
|-------|-------|---------|---------|---------------------|
| Risk | `Routes` | the route rows | none | at most `MaxRiskRows`, one per key |
| RouteRisk | `From`, `To` | the directed edge | required | a configured edge, neither end an outside node |
| RouteRisk | `Mode` | the transport mode | required | an existing mode |
| RouteRisk | `StrikeBP` | the probability of a strike per shipment (3.2.1) | required | 1 to 10 000 |
| RouteRisk | `SeverityBP` | the share of each commodity a strike takes | required | 1 to 10 000 |
| RouteRisk | `Common` | one draw per tick for the row (3.3.2) | false | |
| Risk | `Stores` | the store rows | none | at most `MaxRiskRows`, one per key |
| StoreRisk | `Region` | the region of the stores | required | a region that is not an outside node |
| StoreRisk | `Class` | the store class | required | an existing class |
| StoreRisk | `StrikeBP` | the probability that a store is struck in a tick (3.2.2) | required | 1 to 10 000 |
| StoreRisk | `SeverityBP` | the share of the struck store's part of the account a strike takes | required | 1 to 10 000 |
| StoreRisk | `Common` | one draw per tick for the row | false | |

**Bounds** (validation bounds in `bounds.go`, not tunables; `spec-config.md` 4.1 gains one
row): `MaxRiskRows` **4 096** per table, a placeholder far above the first consumer's
36 route rows (12 directed edges, three modes) and a handful of store rows; the cost of a
tick is the rows with `P > 0` for the common draws and the rows on a path for a dispatch.
The scale bound is the existing `MaxShockBP` (`spec-config.md` 4.1: ten times).

### 12.3 What the module adds to other types

None: a store row names a class of `spec-storage.md` and a route row names a mode of
`spec-transport.md`, and neither owner's type changes. The slots are not configuration
(9).

### 12.4 Rows for the parameter-owner table

Every row has a field, a default and a validation test (testing.md 6.7). Section 19 lists
them for `specs/README.md`:

| Parameter | Scope | Owner |
|-----------|-------|-------|
| `Risk` | Modules (module; nil = off) | `spec-risk.md` section 12.1 |
| `Routes` | Risk | `spec-risk.md` sections 3.2.1, 12.1 |
| `Stores` | Risk | `spec-risk.md` sections 3.2.2, 12.1 |
| `From`, `To`, `Mode` | RouteRisk | `spec-risk.md` section 3.2.1 |
| `Region`, `Class` | StoreRisk | `spec-risk.md` section 3.2.2 |
| `StrikeBP`, `SeverityBP`, `Common` | RouteRisk; StoreRisk | `spec-risk.md` sections 3.2, 3.3, 3.4 |

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the
rounding rule half to even (3.4.1, 4.1.1, 5.1.1, 6.3.2, 8.1.1), the clamp of `P` at 10 000,
the test "draw below `P`", the order of the rows of a path, the larger of the two
fractions of 6.1.3, the order of the draws of 6.2.1 and of the classes of 6.3.2, the
decision that a strike takes a share of **every** commodity (the mode's `Vehicle` units of a
trip account excepted, 5.4.1) and never money, and that a cargo's loss precedes its delivery.

### 12.5 Classification for the registry of knobs (CLB)

12.5.1. For `spec-calibration.md` 6 (a registry row for each field of the table above): a
row's `StrikeBP`, `SeverityBP` and `Common`, and the existence of a row, are **structural
choices**: a pack states how dangerous its world is, and no search range exists for them
(a danger is a fact of a setting, not a quantity to be tuned to a fact of the economy).
None is a knob or a damper strength, and no field moves the steady state of the
viability programme (`spec-viability.md` 11): the programme reads a lane's loss only
through `transport.Lanes` (8.3), a derived input and not a path of the table. The two
slots are not configuration and are not rows (`spec-calibration.md` 3.5 item 1: a slot is
set between steps by a script that is data, never by a calibration rule).

## 13. Validation

### 13.1 Errors

Each is a `ValidationError` with a `Path` and a `Rule` (`spec-config.md` 7.1.1), and each
has a failing-case test (testing.md 3.9). The sentinels are new unless noted.

| `Rule` | Sentinel | Condition |
|--------|----------|-----------|
| `risk_module_dependency` | `ErrModuleDependency` | a route row and `Modules.Transport` is absent, or a store row and `Modules.Storage` is absent |
| `unknown_region` | `ErrUnknownRegion` | `From`, `To` or `Region` names no region |
| `unknown_edge` | `ErrUnknownEdge` (REG's) | the pair `(From, To)` is not a configured edge |
| `unknown_mode` | `ErrUnknownMode` (TRN's) | `Mode` is not a configured mode |
| `unknown_store_class` | `ErrUnknownStoreClass` (STO's) | `Class` is not a configured class |
| `risk_outside_node` | `ErrOutsideNode` (EXT's, REG's) | a route row's edge has an outside node at an end, or a store row's region is an outside node: no shipment crosses it and no account stands there (a row that cannot have an effect is refused, `spec-integration.md` 5.1.3) |
| `risk_row_shape` | `ErrRiskShape` | `StrikeBP` or `SeverityBP` outside `1 .. 10 000` |
| `duplicate_id` | `ErrDuplicateID` | a key twice |
| `risk_bound` | `ErrOutOfBounds` | more than `MaxRiskRows` rows in a table |

### 13.2 Warnings

Warnings do not fail validation (`spec-config.md` 7.2), are logged once at `Warn` with
`rule` and `path` at construction, and are returned by `World.Warnings()`.

| Warning | Condition |
|---------|-----------|
| `no_risk_rows` | the module is on and has no row: it does nothing |
| `risk_edge_off_path` | a route row's edge lies on no shortest path between two regions of the configured graph at the configured distances: a shipment crosses it only if a status changes the paths |
| `risk_store_class_unowned` | a store row's class is listed by no role row's `Stores` (`spec-storage.md` 12.1): it matters only if a call gives a store of the class |
| `risk_certain` | a row has `StrikeBP` 10 000: it strikes every time (a `Common` row strikes every tick); with a `SeverityBP` of 10 000 nothing crosses the lane or stays in the store |

### 13.3 Not validated here

A row on a mode that no strategy uses; a lane whose expected loss makes it unprofitable
for every merchant (a pack's intent); the effect of a scale (a call, 9.2).

## 14. Observability

### 14.1 Events

Three events and four fields, all gated by the module (an event of a module that is off
is not emitted, `spec-observability.md` 4.1). None has a map or a pointer, and all are
built only when an observer was supplied (observability-and-logging.md 4.5). Section 19
gives the rows for `spec-observability.md` 4.1 (N-126).

| Event | Position | Class | Fields (besides `Tick`) |
|-------|----------|-------|-------------------------|
| `StoreLoss` (new) | P8.1a, ascending agent ID then class, after the whole position is applied | E (per agent and class struck) | `Agent`, `Region`, `Class StoreClassID`, `Stores` (stores of the class struck, at least 1), `Common bool` (the row's flag), `ShareBP` (the share of 6.3.2, after the cap), `Lost []Amount` (a copy, in commodity order; empty when every product rounded to 0) |
| `RouteRiskScaled` (new) | the call, ascending (`From`, `To`, `Mode`) | C | `From`, `To` (regions), `Mode`, `BeforeBP`, `AfterBP` |
| `StoreRiskScaled` (new) | the call, ascending (`Region`, `Class`) | C | `Region`, `Class`, `BeforeBP`, `AfterBP` |
| `ShipmentDispatched` (a field added) | P3.2(c), P3.3 | E | `LossBP` (the share fixed at dispatch, 0 for no strike or the module off) |
| `ShipmentArrived` (two fields added) | P1.3 | E | per line `Lost Quantity` (beside `Shipped`, `Delivered`, `Diverted`, `Spoiled`); `AccountLost []Amount` (the loss of the traveller's own account, 5.4, in commodity order; empty for a hired shipment) |
| `ContractSettled` (a field added) | P1.4 | E | `Lost` (units of the contract that were lost in transit, beside `Spoiled`; `Reached = Quantity - Spoiled - Lost`) |

`StoreLoss` is **per entity** (an agent and a class), like `AgentRelocated` and
`ShipmentArrived`; its volume is bounded by the stores that exist and the probability of
a strike, never by orders or trades. There is no event for a strike of a route row: its
fact is in `ShipmentDispatched.LossBP` and its effect in `ShipmentArrived`. There is no
event for the common outcome of P3.1a or P8.1a as such: a common strike with nothing
exposed is a draw with no effect and needs no observer.

### 14.2 Reasons

None new. The ledger reason `loss` (7.1) is told apart in `BoundaryFlow` by its name.

### 14.3 Logs

Through the injected logger, constant lowercase messages, the keys of
observability-and-logging.md 3.3.2 (section 19: `share`, a share in basis points, and
`rows`, a count in the construction summary, are the new keys; `class` is STO's):

| Level | Message | Attributes | When |
|-------|---------|------------|------|
| `Info` | "world created" gains `rows` (route rows plus store rows) and the module name `risk` in `modules` | | once, at construction |
| `Warn` | the warnings of 13.2 | `rule`, `path` | once each at construction |
| `Debug` | "route struck" | `tick`, `shipment`, `mode`, `region`, `dest`, `share` | per shipment dispatched with `LossBP` above 0, guarded by `Enabled` |
| `Debug` | "store struck" | `tick`, `agent`, `class`, `share` | per (agent, class) struck, guarded |

No `Info` and no `Warn` is emitted inside the tick: a strike is a fact of an entity and
is an event (observability-and-logging.md 3.2.2); it is not an anomaly of the
simulation, so no `Warn` is added and no counterpart is owed (N-126).

### 14.4 Metrics

Derived from the events (observability-and-logging.md 5.3.1), integers, bounded
attributes (5.4). Section 19 gives the rows.

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.risk.strikes` | Counter | `{strike}` | `economysim.hazard` | `ShipmentDispatched` (1 for each with `LossBP` above 0, by `route`), `StoreLoss` (`Stores`, by `store`) |
| `economysim.transport.cargo.lost` | Counter | `{unit}` | `economysim.commodity` | `ShipmentArrived` (each line's `Lost` and each `AccountLost` amount) |
| `economysim.store.lost` | Counter | `{unit}` | `economysim.commodity`, `economysim.store_class` | `StoreLoss` (`Lost`) |

`economysim.hazard` is a new attribute with two constants (`route`, `store`), bounded by
the module (5.4.1 there); `economysim.store_class` is STO's. The units lost by commodity
are also in `economysim.commodity.flow` (the reason `loss`, the kind `sink`); the two
counters above tell the **source** apart, which the reason alone does not. Series for the
reference chain: 2, at most 30, and at most 30 times the classes.

### 14.5 Snapshot

OBS owns the names (`spec-observability.md` 5.3); this spec asks for these, all plain data
and all functions of saved state (N-125): `SlotState` (`spec-integration.md` 5.3) gains
`RouteRiskScales` and `StoreRiskScales`, the entries that differ from 10 000, in the order
a checkpoint saves them. The expected loss of a route is a derived read (8.2) and is not a
snapshot field.

### 14.6 Cost with no observer

With the default observer and logger no event is built and no log call allocates (the flag
of observability-and-logging.md 4.5). The per-dispatch work is a lookup of at most the
path's rows in the dense array and at most that many words; the per-tick work is the
common rows and the stores of the limited owners; neither allocates (NFR-RSK-001).

## 15. State contributed to a checkpoint

15.1. **Saved**, in a section of its own present when `Modules.Risk` is non-nil
(`spec-checkpoint.md` 4.2 gains a row, section 19), plain data in a fixed order:

- the route scales that differ from 10 000, ascending by key, each (`From` index, `To`
  index, mode index, `BP`);
- the store scales that differ from 10 000, ascending by key, each (region index, class
  index, `BP`).

and **one field in the section of another spec** that hosts it: the shipment record of
transport (`spec-checkpoint.md` 4.2 tag 17) gains `LossBP`, an unsigned integer 0 to
10 000, per shipment in flight, present when the module is on (a section of a module that is off is absent, `spec-checkpoint.md` 4.2.1; the field is
the module's own and is present only when the module is on, a fact the configuration
hash fixes, and the owed row for tag 17 says so, section 19).
The scales are sparse (a value of 10 000 is corrupt, R10 there).

15.2. **Not saved** (derived, `spec-checkpoint.md` 4.3): the common outcomes of P3.1a and
P8.1a (drawn afresh at the position of the next tick, from the saved stream), the table of
expected losses (rebuilt from the rows, the scales and the graph), the dense arrays of
rows, the effective probabilities.

15.3. **Restore validation** (`spec-checkpoint.md` 6.2 gains these): a scale's key names a
configured row, a scale is within `0 .. MaxShockBP` and not 10 000, keys are ascending
and unique, and a shipment's `LossBP` is within `0 .. 10 000`. A failure is
`ErrCheckpointCorrupt`.

15.4. **A checkpoint is taken between ticks** (`spec-world.md` 8.4), where no position of
this module is half done: P3.1a, a dispatch, an arrival and P8.1a all complete inside a
step, and a restored world draws the same words from the saved PCG state
(FR-CHK-051, `spec-checkpoint.md` 6.7).

15.5. **Format version.** The shipment record and the new section change shape with the
module on, so the format version changes with the first release that has the module
(`spec-checkpoint.md` 9.3), and a fixture of it joins `testdata/checkpoints/`.

## 16. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. The checklist of
`spec-stability.md` 9 was followed: 9.1 each rule of sections 4 to 6 was read for the
chain it creates; 9.6 the delays were checked against the neighbours, and each entry
names the ones that match: the transit time `k` and the rumour lag (`spec-transport.md`
4, `spec-information.md` 5.2), the effective welfare window
`T = max(WelfareWindow, WindowCycles * longest Duration)` (`spec-welfare-and-wages.md`
2.3.3, which a trip longer than it is already a hazard of, L19), a shelf life, and the
belief adjustment (`spec-beliefs.md` 5.2).

**What was looked for and not found.** The brief names a loop, "loss that raises price
that raises risk-taking". It is a loop of this module only if the probability or the
share of a strike depends on what the economy does: traffic on a lane, value carried, a
price. They do not (position 13), so the chain is *balancing and bounded* (RSK-A1) and
the reinforcing version needs a controller outside the engine (RSK-A6).

**RSK-A1 Price gap, shipping and loss.** *Kind:* loop, a case of L84, L85 and L93
(price-gap shipping with a transit delay; the sign structure is unchanged). *Loop:* the
gap at the destination (+) shipments (+) cargo arriving, `(1 - lambda)` of what left (-)
the destination price (+) the gap: one negative link, **B**, with the delay; and the
dispatch raises the origin price at once: shipments (+) the origin price (-) the gap, one
negative link, **B**, immediate. A loss `lambda` scales the arrival link: the loop gain
falls, so the loop is **more** stable and the shipper is **poorer**. *Delay:* the total
`n` of L84's MER case, `Lag + k + 1` (the age of the rumour, the tick between the
decision and the dispatch, and the transit time `k`; `spec-merchants.md` 7.2.1), or
`k` for a shipper that reads no rumour. *Owner:* RSK 4, 5, 8; MER 5.2, 7.
*Damper:* RSK-D1 (the expected loss in the merchant's margin), and MER's own: D38 (the
pipeline view, `Inbound`), D39 and D80 (the dispatch cap and the lot cap) and the lane
lock. *Undamped:* a
consumer's strategy that ships on the nominal gap and ignores `Routes.LossBP`.
*Condition:* the linearised condition of L84 with the gain scaled and the **total delay
`n`**, `g_eff < 2 sin(pi / (2 (2 n + 1)))`, `g_eff = g (1 - lambda)` for a shipper that
does not read the loss and `g (1 - lambda)^2` for one that does. The numbers are the
model of 17 part 7 with **none of the effect at dispatch** and `n` = 3: the largest gain
that decays is 0.444 with no loss, 0.592 (not read) and 0.790 (read) at `lambda` 2 500
basis points, 0.890 and 1.780 at 5 000. With **half of the effect at dispatch** (the
purchase moves the origin price at once, as L84 says of a merchant) the limit is about 2
with no loss, as L84 says, and the loss only raises it (17 part 7: 1.998, then 2.348
(not read) and 2.716 (read) at 2 500, 2.872 and 3.830 at 5 000); a loss does not turn a
loop that decays without it into one that does not, in the model. *Cost:* a shipper that does not read the loss ships until the
**nominal** gap has closed, at a real margin below 0: in the model (destination 180,
origin 100, freight 20, gain 0.05, `k` 3, `lambda` 2 500) it ships 68.6 units, lands 51.4
and ends at a margin of -38.6 a unit, where one that reads the loss ships 19.2 and ends
at 0. *Neighbours (9.6):* the rumour's `SourceAge` bound and the lag are near `k` by
design (MER-D4); no new match. The loop is the same chain as L84's MER case with the
arrival link scaled, so it is entered here as a case of it and not as a new loop. *Test:* RSK-S3. *Status:* model (17); a linear model of an
invented rule, not of MER's.

**RSK-A2 The leaky buffer.** *Kind:* loop. *Loop:* the stock (+) the units a strike
takes, a share of the stored part (-) the stock: one negative link, **B**. And the
refill: the stock (+) the stock ratio (-) the owner's limit price (the belief rule raises
the centre when the ratio falls, `spec-beliefs.md` 5.5) (+) its purchases up to `Target`
(`spec-storage.md` 6.2) (+) the stock: one negative link, **B**, toward the target,
with a permanent flow equal to the loss: to hold a stock `S` the owner must buy `S *
(share of a strike) * (probability)` a tick, so a store is a drain proportional to what it
holds. *Delay:* the belief adjustment (a few ticks, `spec-beliefs.md` 5.2), and the
purchase's own one-tick delay (`spec-world.md` 6.1.8). *Owner:* RSK 6. *Damper:* the
inventory buffer (D07), the budget stack and the saving rule's spend cap (D27:
`SpendMaxShare`, `SpendHalfTicks`, `spec-agents.md` 7.4), the room (STO-D4, a constraint of STO with no D number). *Undamped:* none: the loop balances, and an
owner whose refill exceeds its cap holds less than its target. *Condition:* the refill
bill, at the reference price, is within the spend cap. *Neighbours (9.6):* the belief
step against the shelf life of the goods stored (L25): a store that loses a share each
tick and spoils at the end of its shelf life has two sinks and one refill. *Test:*
RSK-S4. *Status:* hand.

**RSK-A3 A common strike is a lockstep.** *Kind:* hazard, a case of L93 (herding of
threshold shippers) and of the lockstep of L34 (a cohort moving together). *Chain:* one word strikes every store of a class in a region, or every
shipment dispatched on a lane that tick: the stocks fall together, the stock ratios fall
together, and the owners' bids rise at the same P3.2 and the price moves by the sum.
*Polarity:* none of its own. The strike is an exogenous shock applied to every copy of
the **B** loop of RSK-A2 at once; each owner's response has A2's sign, and what is new
is the simultaneity, not a link. It is
the correlation that a unit row does not have, and the one that an insurer cannot
diversify (10.2.3). *Delay:* the belief adjustment and `BeliefAdjustProb` (asynchronous
by design). *Owner:* RSK 3.3. *Damper:* the asynchronous belief
adjustment, the per-agent spread of the target ratio drawn at creation
(`spec-welfare-and-wages.md` 5.1), the room. *Undamped:* a pack of identical owners and a
common row of a high probability. *Condition:* the product of the probability and the
number of owners of the class in the region is small against the market's depth, which
the pack's author states as a number of owners. *Test:* RSK-S4 (the common leg).
*Status:* untested.

**RSK-A4 Ruin by a run of losses.** *Kind:* hazard, a case of L19 (a trip is a gap in
income) and of L40 and L63 (relief and migration). *Chain:* a merchant or a deliverer
whose cargo is lost has paid for its inputs and its haul units and earned nothing: a
lost cargo (-) its welfare ratio (-) the privation count (+) the ladder: a role change,
relief, a weight reduction, removal (`spec-spawner.md` 5), which takes it off the lane.
*Polarity:* the loss is an input and not a link of a loop; the loop it acts on is
the lane's carriers (-) the lane's margin (+) entry (+) the carriers, one negative
link: **B**, with the privation count's delay. A selection effect: the merchants that remain
are the lucky ones, so an old merchant is a lucky merchant and a statistic over survivors
overstates the lane's return. *Delay:* the privation count, longer than the welfare
window by design. *Owner:* RSK 5; MER 5.5. *Damper:* the fee share's margin of safety
(`spec-merchants.md` 5.5.1: at 5 000 a trip must promise twice its cost), the
leaky privation, relief, and the effective window. *Undamped:* a lane whose loss is above
the margin of safety. *Condition:* the fee-share test of `spec-merchants.md` 5.5.1 passes on the revenue net of
the loss, `C * 10 000 <= phi * q * (Rnet * (10 000 - LossBP) / 10 000 - P_acq)`, so a lane
is worth a trip only while the margin of its lot survives the lot's own expected loss (8.4.2). *Neighbours (9.6):* the window `T` against a trip of `k`
ticks (L19, `trip_longer_than_window`). *Test:* RSK-S3 over seeds (the removal rate of
the merchants with and without the risk). *Status:* untested.

**RSK-A5 Haul units are spent whatever arrives.** *Kind:* hazard, a case of L21 and of
MER-A4. *Chain (MER-A4's cobweb with a wrong price):* a hired dispatch redeems its haul units at
dispatch (`haul_use`) and the cargo may be lost at arrival, so the demand for haul units
does not follow the loss of the cargo unless the shipper reads it (RSK-D1 puts it in the ceiling `HaulCeiling` of
`spec-merchants.md` 5.6): the choke price of haul units falls by `(1 - lambda)` and the
carriers' output of an unread loss is a cobweb with a wrong price. *Delay:* the
recipe's `Duration` of the carrier. *Polarity:* **B**, as MER-A4's, the haul price (+)
the carriers' output after `Duration` (-) the haul price; the loss does not change its
sign, it moves the choke price the demand side reads. *Owner:* RSK 8.4; TRN 8. *Damper:* RSK-D1. *Undamped:*
a consumer's strategy that ignores the loss. *Test:* RSK-S3. *Status:* hand.

**RSK-A6 Endogenous risk is not modelled.** *Kind:* constraint (the absence of a loop).
*What it would be:* a probability that grows with the traffic on a lane, with the value
carried or with the price at the destination: raiders go where the cargo is. That chain
(the destination price (+) shipments (+) strike probability (+) loss (-) arrivals (-) the
destination price: two negative links) is **R**, reinforcing, and none of its links is in
this module (position 13). *Where it goes:* a
consumer's controller reads a measure (a loss measure over the events and the `Loss`
account of `spec-indicators.md` 7.1, or the traffic) and
sets a scale (9.4); the engine keeps conservation and determinism of every setting and
does not promise the controller's stability (N-131, L121). *Condition for a consumer:* a
controller's gain and delay are bounded as `spec-policy.md` 15 (PLC-A1) requires of a `rule`. *Test:* RSK-S5
(a scripted scale series with an audit). *Status:* constraint.

**Dampers.** *RSK-D1* the expected loss in the shipper's margin and ceiling (8.4): a
lane's `LossBP` lowers the cautious revenue and so the margin, the bid for the cargo and
the ceiling of the bid for haul units (new, an edit to MER); *RSK-D2* the bounded
exposure, which is a constraint and not a stabiliser: a cargo is at most what a mode
carries (`spec-transport.md` 3.2), a stored part is at most the stores' capacity
(`spec-storage.md` 8.3.2) and a strike takes a share, so a strike cannot take more than
the account holds; *RSK-D3* no endogenous risk (position 13), a constraint.

**Scenarios proposed** (numbered by the supervisor; every one runs with the module on over
a range of seeds with `WithAudit` and asserts the identities of 7.3 at every tick):
*RSK-S1* neutrality: the module on with every scale at 0 against the module off, bit for
bit in the state and the events (FR-RSK-005); *RSK-S2* accounting: the reference chain
with the trade overlay, a route row and a store row, the per-line identity, and the
boundary balance of `loss` equal to the sum of the events; *RSK-S3* a merchant lane with
and without the reading of `LossBP` (the totals shipped, landed and wasted; the removal
rate of merchants; RSK-A1, A4, A5); *RSK-S4* a buffer store's owner under a unit row and
under a common row (the refill, the lockstep of several owners; RSK-A2, A3); *RSK-S5* a
scripted series of scales (a surge, a patrol) with events at the calls and the audit
(RSK-A6); *RSK-S6* a checkpoint at a boundary with shipments in flight that carry
`LossBP`: the restored run's strikes, losses and events equal the uninterrupted run's.

## 17. What was run

One script, in Docker with `PYTHONDONTWRITEBYTECODE=1`, `doc/specs/models/rsk/loss.py`
with its output `loss.out` (`doc/specs/models/README.md` gains a row, section 19). It is
a model of the **rules** of sections 3 to 6 and 8 with integers and placeholder numbers.
It shows arithmetic and one stylised loop. It does **not** show that any default is
stable, nor how the library behaves: nothing has run in it.

| Part | What | Result |
|------|------|--------|
| 1 | the first words of seeds 0, 1 and 42, reduced by the world's draw (the PCG of `models/wld/seed.py`, which matches Go 1.25) | seed 42: 3 819 and 1 485 (the words of example A) |
| 2 | example A: a common row and a unit row on a path, the combined share, four lines, a contract | `LossBP` 2 500; the lines of 18.1 |
| 3 | example B: a limited owner, two classes, `phi`, the shares and the losses | `phi` 3 272 and 4 000; 133 of 250 and 27 of 50 |
| 4 | example C: the traveller's own account, with the mode's `Vehicle` among its holdings | 3 of 12 and 1 of 3; the 2 vehicles are not exposed |
| 5 | the rounding bias of a line | a line of `q` units loses within half a unit of `q * share`; over `q` = 1 to 20 the mean error is at most 0.025 units a line, and for single-unit lines the loss is a step at a share of 5 000 |
| 6 | the expected loss against 500 000 simulated dispatches | 1 305 by the formula, 1 306.1 simulated (standard error 2.8); the frequency of a strike at `P` 3 000 is 0.3003 |
| 7 | the price-gap shipper with a loss: totals, and the largest gain that decays at a total delay `n` = 3, with none (a = 0) and with half (a = 0.5) of the effect at dispatch | 68.6 shipped and 17.1 wasted when the loss is not read against 19.2 and 4.8 when it is, at `lambda` 2 500; a = 0: 0.444, 0.592 and 0.890 (not read) and 0.444, 0.790 and 1.780 (read) at 0, 2 500 and 5 000; a = 0.5: 1.998, 2.348 and 2.872 (not read) and 1.998, 2.716 and 3.830 (read) |

**What the model does not show.** It does not show the library's draws (it takes the
words in the order the spec fixes and no other), the interaction of a strike with the
belief rule, the failure ladder or the merchant's lot cap, or any of RSK-A2 to A5: those
are hand entries. Part 7 is a linear model of an invented rule, as the model of
`spec-transport.md` 16 is.

## 18. Worked examples

The settings are invented **examples** (the words `wagon`, `grain`, `cloth`, `bin`, `pit`
are data, never branched on); every number is a placeholder. The arithmetic is
`models/rsk/loss.py`.

### 18.1 Example A: a hired shipment over two rows

Regions `north` (0), `mid` (1), `south` (2) with the edges `north` to `mid` and `mid` to
`south`; the hired mode `wagon`. Two rows:

| Row | `StrikeBP` | `SeverityBP` | `Common` |
|-----|-----------:|-------------:|----------|
| (`north`, `mid`, `wagon`) | 3 000 | 2 500 | false |
| (`mid`, `south`, `wagon`) | 1 000 | 6 000 | true |

Both scales are 10 000, so `P` is 3 000 and 1 000. In a tick whose next words are the
first two of seed 42 (the real stream has taken others before; the example fixes the
position, not the seed's run):

1. **P3.1a (D13).** The common row (`mid`, `south`, `wagon`) takes the word
   `61c88529c9612c1b`, which reduces to 3 819. 3 819 is not below 1 000: the lane is
   spared this tick.
2. **A shipment of `north` to `south`** by `wagon` is dispatched at tick 12 and arrives at
   tick 15. At the record (step 4a, **D14**) its path is `north`, `mid`, `south`. The unit
   row takes the word `2608d8a3075aa168`, which reduces to 1 485, below 3 000: struck. The
   common row's outcome is "spared". `kept = rhe(10 000 * 7 500, 10 000) = 7 500`,
   `LossBP = 2 500`. `ShipmentDispatched.LossBP` is 2 500.
3. **Expected loss of the route** (8.1.1): `f1 = 10 000 - rhe(3 000 * 2 500, 10 000) =
   9 250`, `f2 = 10 000 - rhe(1 000 * 6 000, 10 000) = 9 400`, `K = rhe(10 000 * 9 250,
   10 000) = 9 250`, then `rhe(9 250 * 9 400, 10 000) = 8 695`: `LossBP` read by a shipper
   is 1 305.
4. **Arrival (P1.3, step 0) at tick 15.** Four lines, loss 2 500:

| Line | Shipped | Spoiled on the way | Held | Lost `rhe(held * 2 500, 10 000)` | Delivered |
|------|--------:|-------------------:|-----:|---------------------------------:|----------:|
| `grain` | 40 | 3 | 37 | `rhe(9.25) = 9` | 28 |
| `cloth` | 5 | 0 | 5 | `rhe(1.25) = 1` | 4 |
| `rope` | 2 | 0 | 2 | `rhe(0.5) = 0` (a tie, to the even 0) | 2 |
| `tar` | 6 | 0 | 6 | `rhe(1.5) = 2` (a tie, to the even 2) | 4 |

The per-line identity holds: `40 = 28 + 0 + 3 + 9`.

A delivery contract is another case. Had the line been a contract of 30 units at a
price of 20, with 3 spoiled and 5 lost, then `reached = 30 - 3 - 5 = 22`: the deliverer is
paid `20 * 22 = 440` and the payer is refunded `20 * 8 = 160` (5.6).

### 18.2 Example B: a limited owner with two classes

A household agent of weight 1 with a base room of 100 mass and 80 bulk owns two stores of
the class `bin` (capacity 200 mass, 100 bulk) and one of `pit` (100 mass, 200 bulk). It
holds 250 `grain` (mass 2, bulk 1) and 50 `cloth` (mass 1, bulk 3). Rows for its region:
`bin` with `StrikeBP` 500, `SeverityBP` 4 000, unit; `pit` with `StrikeBP` 200,
`SeverityBP` 10 000, common.

- Load: mass `250 * 2 + 50 * 1 = 550`, bulk `250 * 1 + 50 * 3 = 400`. `SumCap`: mass
  `2 * 200 + 100 = 500`, bulk `2 * 100 + 200 = 400`. Stored (`spec-storage.md` 8.3.2):
  mass `min(500, 550 - 100) = 450`, bulk `min(400, 400 - 80) = 320`.
- `bin`, one store: `x_mass = floor(450 * 200 / 500) = 180`, `phi_mass = floor(180 * 10 000
  / 550) = 3 272`; `x_bulk = floor(320 * 100 / 400) = 80`, `phi_bulk = floor(80 * 10 000 /
  400) = 2 000`; `phi = 3 272`. `pit`: `x_mass = 90`, `phi_mass = 1 636`; `x_bulk = 160`,
  `phi_bulk = 4 000`; `phi = 4 000`.
- P8.1a. Suppose the common `pit` row struck (its word came first, D15 (i)) and one of the
  two `bin` stores was struck by its own word (D15 (ii)). Shares: `bin`: `1 * rhe(3 272 *
  4 000, 10 000) = rhe(1 308.8) = 1 309`; `pit`: `1 * rhe(4 000 * 10 000, 10 000) = 4 000`.
- Losses, from the quantities at the start (`q0`): `grain` `bin` `rhe(250 * 1 309 / 10 000
  = 32.7) = 33`, `pit` `rhe(100.0) = 100`, total 133, left 117; `cloth` `bin`
  `rhe(50 * 1 309 / 10 000 = 6.545) = 7`, `pit` `rhe(20.0) = 20`, total 27, left 23. Two `StoreLoss` events, `bin` then `pit` (class order), each with its `Lost`.

### 18.3 Example C: the traveller's own account

A self-carried trip by `wagon` to oneself with `LossBP` 2 500, a mode whose `Vehicle` is
the commodity `cart`. At arrival the traveller's account holds 12 `grain`, 3 `cloth` and
2 `cart` (the vehicles that carried it): lost `rhe(12 * 2 500, 10 000) = 3` and
`rhe(3 * 2 500, 10 000) = rhe(0.75) = 1`; the 2 `cart` are left out (5.4.1), where a rule
over every commodity would have taken `rhe(2 * 2 500, 10 000) = rhe(0.5) = 0` of them
today and one at a share above 5 000 (5.3.2). `ShipmentArrived.AccountLost` is `grain` 3,
`cloth` 1; the account then changes region with its 2 `cart`.

### 18.4 Example D: the words of one tick

Rows: R1 (`north`, `mid`, `wagon`) common; R2 (`mid`, `south`, `wagon`) unit; R3 (`north`,
`mid`, `foot`) unit; S1 (`north`, `bin`) common; S2 (`north`, `pit`) unit; all with
`P > 0`. In one tick three shipments are dispatched: two by `wagon` from `north` to
`south` and one by `foot` from `north` to `mid`. In `north`, agent 7 owns two `bin` and one
`pit` with a stored load, and agent 9 owns one `pit` with a stored load of 0.

| Position | Words | Why |
|----------|------:|-----|
| P3.1a (D13) | 1 | R1 only (a common row); R2, R3 are unit |
| dispatches (D14) | 3 | each `wagon` shipment: R1's outcome is the tick's, R2 takes one; the `foot` shipment: R3 takes one |
| P8.1a (D15, common) | 1 | S1 |
| P8.1a (D15, stores) | 1 | agent 7's `pit` (S2); its two `bin` use S1's outcome; agent 9 has no stored load |

Total 6 words, a function of the configuration and the state (11.3). With S2's scale set
to 0 the total is 5 and every later word of the run moves by one place (WLD-A5): a
comparison across that call is a different stream from there (11.4).

## 19. Notes settled, and what this spec needs from the other specs

### 19.1 Notes

| Note | Disposition | Where |
|------|-------------|-------|
| N-046 (RSK) | **Applied for RSK**: entries RSK-A1 to A6, dampers RSK-D1 to D3, scenarios RSK-S1 to S6, provisional. Remove RSK from the areas; SAV, CRD, FIS, PLC stay | 16 |
| N-090 (PRO, SKL, INT) | **Applied**: this module registers no `region_bound` refusal, with the reason; the rule stands for CAP and ORG | 3.5 |
| N-016 (ALL, CFG, WEL) | **Applied (rows)**: the owner-table rows of 12.4; no unnamed parameter; the registry test is still to be written | 12.4 |
| N-017 (ALL) | **Applied**: terms in 19.3; setting words only in flagged examples | 19.3 |
| N-027 (ALL, REF, STY) | **Applied**: every number is a placeholder; no default is called stable (the module has no default) | 12.2, 17 |
| N-103 (ALL) | **Applied**: provisional IDs; seams in the handoff | 16 |
| N-126 (ALL) | **Applied**: rows for the event, key and metric tables in 14 and 19.2; no `Warn` is added | 14 |
| N-128 (ALL, CHK) | **Applied**: the one per-shipment field is saved once, in TRN's record; the module has no per-agent field | 15 |
| Read, not tagged RSK | N-011 (loss in transit is `loss`, owner RSK: the note's STO area is closed by STO); N-021 (no new reason: `loss` exists, owner changes); N-045 and N-065 (every strike is counted by an event); N-107 (no money flow, 7.4); N-120 (a migrant is not exposed, open question 7); N-124 (a paired comparison across a scale, 11.4); N-125 (a snapshot field is a function of saved state: yes, 14.5); N-131 (a controller is outside the engine, 9.4, RSK-A6); N-122 and WLD D87 (nothing here serves a candidate first: the draws are exchangeable, so the order of the dispatches gives no shipment a better chance) | |

### 19.2 What this spec needs from the other specs

| Spec | Section | Edit |
|------|---------|------|
| `spec-world.md` | 6.5, 6.10, 6.3 | row P3.1a; row P8.1a; P1.3 notes the loss step 0 of an arrival |
| | 6.11 | the `StoreLoss` events first in phase 8 |
| | 8.2 | two rows: the route risk scale (read at P3.1a and at the dispatch record) and the store risk scale (P8.1a) |
| | 11.1 | the cost rows: P3.1a `O(common route rows with P > 0)`; the step 4a of a dispatch in the cost of P3.2 and P3.3, `O(rows on the path)`; P8.1a `O(common store rows + stores of the limited owners)` |
| | 7.3, 7.3.1 (placement) | the rows D13 to D15 go **at their position** under the new IDs: D13 between D3 and D4, D14 after D5 (inside each agent's turn at P3.2(c), and at P3.3 after D6), D15 after D11 (the heading is "in the order of a run", so an ID is not an order) |
| `spec-transport.md` | 3.2.2, 5.2, 5.5.1, 5.7, 5.9.2 | field `LossBP`; step 4a; step 0; a sentence in 3.2.2 that the same units the load leaves out (the `Vehicle`) are not exposed to a strike (RSK 5.4.1); in 5.7 and 5.9.2 the traveller's own account is struck after the Transit account and a shipment with lines of a removed traveller is struck in its Transit account |
|  | 7.2, 7.3.2, 5.6.3, 10.1 | the owners sentence of 7.2 reads RSK; `loss` is used with the module on; the identity includes `lost`; 10.1 points here |
| | 9.1, 9.2, 9.4 | `Routes[].LossBP`; the read `RouteLoss` beside `Travel` (the formula of RSK 8.1.1); the lane's loss from 8.3 |
| | 13.1, 14, FR-TRN-028, 031, 041 | the event fields; `LossBP` saved; the requirements read "with the module off" |
| `spec-storage.md` | 8.3, 10.2 | pointer to section 6 here; the RSK row says used |
| `spec-infrastructure.md` | 5.5, 6.2.3, 21 (7) | "RSK needs to lower a condition" becomes "RSK lowers none in version 1; the call stays the consumer's; the reserved shape is `spec-risk.md` open question 13"; 6.2.3 and 5.5 name RSK's `loss` as goods only |
| `spec-merchants.md` | 5.2.1, 5.5.1, 5.5.3, 5.6, 9.7, 15.1.1, FR-MER-008, FR-MER-040 | `Rnet` net of `LossBP`, so `G`, `BreakEven` and `HaulCeiling` are (`AskLimit` holds no `Rnet` and is unchanged); the table of 9.7; `ContractSettled.Lost` |
| `spec-ledger.md` | 7.2 | the `loss` row: no longer "later", goods only in version 1 (the money form is reserved and unused); the owner is already RSK |
| `spec-regions.md` | 5.1.3, 6.5.4, 7.3.3 | two rows (`Modules.Risk.Routes`, `.Stores`, replace); a pointer; "RSK has none" |
| `spec-integration.md` | 5.1, 5.1.3, 5.3, 11.2 | two setters; their errors; `SlotState`; `ErrUnknownRiskRow` |
| | 6.2 | the row of `InTransit(a)`, `Travel(from, to, mode)`, `Away(a)` gains `RouteLoss(from, to, mode)`, owner TRN 9.2, class 1 (a cached lookup, 8.1.3); INT 6.2 says it asks the owners for nothing else, so this is a read INT lists and TRN defines |
| `spec-observability.md` | 4.1.5, 4.1.6, 4.1.8, 6.2, 6.5.2, 7 | `StoreLoss` in 4.1.5; `RouteRiskScaled` and `StoreRiskScaled` in 4.1.8; the fields of `ShipmentDispatched`, `ShipmentArrived` and `ContractSettled` in 4.1.6 (count 52 plus 3; classes E +1, C +2); three metrics and the attribute `economysim.hazard`; keys `share`, `rows`; in 6.5.2 the money row's "`loss` included" is stale (`loss` is goods only; the count 82 is a ceiling and stays valid) |
| `spec-checkpoint.md` | 4.2 (tag 17 row and a new row), 4.2.5, 6.2, 9.3 | tag 17's row states the field `LossBP`, present when `Modules.Risk` is; a new tag, 28, for the module's two scale lists (`spec-checkpoint.md` 4.2; RSK needs only that it follows 17); the sparse lists; the validation; no format bump while nothing has shipped (9.3) |
| `spec-config.md` | 3.7, 4.1, 7.3, 7.4, 6 | the fields; `MaxRiskRows`; `ErrRiskShape`; the rules and warnings of 13; the file layout |
| `spec-stability.md` | 4, 5, 8 | the entries, dampers and scenarios of 16, numbered after the ones already integrated and the provisional ones of STO, CAP, CRD and the rest; RSK leaves the hook row "STO, RSK" of 4.8 (section 10 has no RSK row, so none is removed there) |
| `spec-viability.md` | 9.4, 20 item 5 (FR-VIA-044 unchanged) | "loss 0 until RSK exists" becomes "the expected loss of `spec-risk.md` 8.3, 0 with the module off"; `WithFreeTrade` stays loss 0 by definition |
| `spec-land.md` | 6.3 (the call), the out-of-scope row and the seams row "RSK" | "Loss of a title by conflict ... `spec-risk.md` (RSK), through the call of 6.3" and "the call of 6.3 with a cause RSK names" become "not RSK's in version 1 (a title is not goods and RSK has no actor): the consumer's call of 6.3"; `spec-risk.md` 10.3 |
| `spec-indicators.md` | 7.1, 7.3, 7.8, 9.1 | the account `Loss` (reserved there, 0 in version 1) becomes non-zero with the module on, as the identities of 7.3 and 7.8 read it; and a new preset in 9.1: lost units and value by commodity, region and source (`route` or `store`), from the events |
| `spec-stylised-facts.md` | 5.1, 5.6 (P06, T02) | convergence within freight **and the expected loss** of the route |
| `spec-calibration.md` | 6 | the registry rows of 12.5 |
| `spec-settlements.md` | 6.3 | H7 stays reserved and RSK declines it (10.3) |
| `spec-polity.md` | 6.3 | H10: RSK adds no cause and no call |
| `spec-money-supply.md` | 10.3 and the later-specs row | RSK adds no money flow: the `loss` of RSK is goods only (10.1 there is boundary counterparties) |
| `spec-reference-chain.md` | 7, 9 | an overlay `overlay/risk.toml` and the scenarios of 16 |
| `specs/README.md`, `spec-roadmap.md`, `documentation_index.md`, `models/README.md`, `notes.md` | | the index row, the code row, the owner rows, the terms, the decisions; the status; the rows; the notes of the handoff |
| `rules/observability-and-logging.md` | 3.3.2, 4.3, 5.3, 5.4.1 | the keys, the events, the metrics, the attribute (a rules change, with a fresh tag) |

### 19.3 Terms (N-017)

**Risk row** (a route row or a store row), **exposure** (what a strike can reach: a
cargo, a trip account, a stored part; not POL's or POP's "exposed" household),
**strike**, **severity** (`SeverityBP`, the share a strike takes), **common row** and
**unit row**, **effective probability** (`P`), **loss share** (`LossBP`, the share fixed
at dispatch), **kept share**, **stored fraction** (`phi`), **expected loss** (of a route),
**risk scale** (always "risk scale" where it could meet RES's capacity scale or POP's
birth scale), **total loss**, **trip account**. "Hazard" is not used for a row: the
register's "hazard" is a kind of entry. Setting words (`road`, `bandit`, `flood`, `wagon`,
`grain`) appear only in the flagged examples of 18.

## 20. Requirements

### Functional

| ID | Requirement |
|----|-------------|
| FR-RSK-001 | The module MUST be on if and only if `Config.Modules.Risk` is non-nil, MUST be written as `[modules.risk]` with `[modules.risk.route.<from>.<to>.<mode>]` and `[modules.risk.store.<region>.<class>]` rows, MUST be part of the configuration hash, and with it off MUST take no draw, emit no event of section 14, post nothing under `loss`, add no field to a shipment and change no result (3). |
| FR-RSK-002 | A table MUST hold at most one row per key (`ErrDuplicateID`), be sorted by its key, be resolved at construction into dense arrays that are not saved, and treat a missing row as no risk with no warning (3.2.3). |
| FR-RSK-003 | Validation MUST reject, with the rule and sentinel of 13.1, a route row without transport or a store row without storage, an unknown region, edge, mode or class, a row at an outside node, a probability or severity outside 1 to 10 000, a duplicate key and more than `MaxRiskRows` rows. |
| FR-RSK-004 | Validation MUST warn, as in 13.2, `no_risk_rows`, `risk_edge_off_path`, `risk_store_class_unowned` and `risk_certain`, and the warnings MUST NOT fail validation. |
| FR-RSK-005 | A world with the module on and every effective probability 0 MUST equal the world with it off in every `Snapshot` at every tick and in every event of the other specs, apart from the fields of this module, which are 0 or empty (11.3.2). |
| FR-RSK-006 | The effective probability MUST be `min(10 000, rhe(StrikeBP * Scale, 10 000))`, and a strike MUST be the event that the world's draw of `10 000` is below it; a probability of 10 000 MUST strike always and still take its word (3.4). |
| FR-RSK-007 | A row with an effective probability of 0 MUST take no word and never strike (3.4.3). |
| FR-RSK-008 | At P3.1a the module MUST take one word for each route row with `Common` and `P > 0`, in ascending (`From`, `To`, `Mode`), every tick whatever is exposed, and keep the outcomes for the tick only (4.2). |
| FR-RSK-009 | At the record of a dispatched shipment (P3.2(c), P3.3) the module MUST visit the rows of its path in path order, use the tick's outcome for a common row and take one word for a unit row with `P > 0`, fold the struck severities by `kept <- rhe(kept * (10 000 - SeverityBP), 10 000)` from 10 000, and store `LossBP = 10 000 - kept` in the shipment (4.1.1). |
| FR-RSK-010 | A request that is refused MUST take no word of D14 (4.1.3). |
| FR-RSK-011 | `ShipmentDispatched` MUST carry `LossBP`, and no view MUST carry a shipment's `LossBP`, a strike, a probability, a `Common` flag or a scale as such (4.4, 8.5). |
| FR-RSK-012 | At the arrival of a shipment with `LossBP` above 0 the loss MUST be applied before the traveller's region change and the delivery: for each commodity of its Transit account in commodity order `rhe(held * LossBP, 10 000)` units MUST move to the boundary under `loss`, oldest lots first (5.1). |
| FR-RSK-013 | For a self-carried shipment whose traveller exists the same rule MUST then be applied to every commodity of the traveller's own account except the mode's `Vehicle` commodity, which a strike MUST NOT take (5.4.1); a removed traveller's account MUST be left to its removal, and a shipment with lines whose traveller was removed MUST still lose from its Transit account (5.4.3). |
| FR-RSK-014 | The per-line identity `shipped = delivered + diverted + spoiled + lost` MUST hold exactly for every line of every shipment (5.2.2, 7.3). |
| FR-RSK-015 | `ShipmentArrived` MUST carry each line's `Lost` and the `AccountLost` amounts in commodity order, and the diversion MUST be decided on the surviving quantity (5.2, 5.5). |
| FR-RSK-016 | In a delivery contract the units that did not reach MUST be the spoiled and the lost, the deliverer MUST bear both, and `ContractSettled` MUST carry `Lost` (5.6). |
| FR-RSK-017 | The stored fraction of a store MUST be computed as in 6.1.2 with checked 128-bit arithmetic, per dimension, and `phi` MUST be the larger of the two. |
| FR-RSK-018 | At P8.1a, after ageing and before the flow totals are final, the module MUST take the common rows' words first in ascending (`RegionID`, `StoreClassID`), then one word for each store of each limited agent with a unit row with `P > 0` and a stored load above 0 in agent, class and instance order (6.2.1). |
| FR-RSK-019 | All strikes of P8.1a MUST be decided before any loss is applied, the shares computed on the state at the start, and the share of a class MUST be `min(10 000, n * rhe(phi * SeverityBP, 10 000))` with the loss of a commodity `rhe(q0 * share, 10 000)` capped at `q0` over the classes (6.3). |
| FR-RSK-020 | A strike on a store MUST NOT destroy the store, change its capacity, its levy or its refusal `has_store` (6.3.3). |
| FR-RSK-021 | `StoreLoss` MUST be emitted once per (agent, class) with a struck store, in ascending agent ID then class, after the position is applied, with the fields of 14.1. |
| FR-RSK-022 | The module MUST post nothing under `loss` but goods, from a Transit account or an agent's account to the boundary, and MUST NOT take money (7.1, 7.4). |
| FR-RSK-023 | For every commodity and run the boundary balance of `loss` MUST equal minus the sum of the lost amounts of the events, and per tick and region `BoundaryFlow` MUST agree with them by 7.2.1 (7.3). |
| FR-RSK-024 | The expected loss of a route MUST be computed as in 8.1.1 from the rows, the scales and the graph at the call, MUST be 0 with the module off or no row on the path, and MUST be shown as `Routes[].LossBP` and by the read `RouteLoss` (8.1, 8.2). |
| FR-RSK-025 | The table of expected losses MUST be rebuilt at the call that changes an edge status or a scale, MUST be derived and not saved, and a lookup in the tick MUST NOT allocate (8.1.3). |
| FR-RSK-026 | `transport.Lanes` MUST take a lane's loss from the configured rows at the scales of 10 000 over the path of the configured graph with every edge open (8.3). |
| FR-RSK-027 | The merchant's cautious revenue MUST be net of the route's `LossBP`, `Rnet * (10 000 - LossBP) / 10 000` truncated, before every use of it (`G`, `BreakEven`, `HaulCeiling`); `AskLimit` MUST NOT read it (8.4). |
| FR-RSK-028 | `SetRouteRiskScales` and `SetStoreRiskScales` MUST be between-ticks calls, atomic, validated in the order of 9.2, a scale of 10 000 MUST clear the key, and each changed key MUST emit its event in key order at the call. |
| FR-RSK-029 | A scale MUST multiply the strike probability only, MUST NOT create a risk (a key with no row is `ErrUnknownRiskRow`), and MUST hold until another call sets it (9.1.2, 9.2). |
| FR-RSK-030 | A shipment in flight MUST keep the `LossBP` it was given at dispatch whatever a call between ticks sets (9.3). |
| FR-RSK-031 | The module MUST NOT close an edge, set an edge status or seize a cargo in flight (10.1). |
| FR-RSK-032 | The module MUST NOT register a `region_bound` refusal (3.5). |
| FR-RSK-033 | The count of draws at P3.1a, at a dispatch and at P8.1a MUST be a function of the configuration, the slots and the state at the position, and MUST NOT depend on an observer, a logger, `WithAudit`, a snapshot, the split of a run into `Run` calls or a checkpoint (11.3.1). |
| FR-RSK-034 | A module that is on MUST take no word for a row with `P = 0`, and no word for a unit row with no exposure (no shipment on it, no owner with a stored load); a common row with `P > 0` MUST take its word every tick (11.3.3). |
| FR-RSK-035 | The module MUST save its two scale lists sparse in ascending key order and the shipment's `LossBP`, MUST NOT save a common outcome or a table, and a restored world MUST draw, strike and lose as the uninterrupted one (15). |
| FR-RSK-036 | The restore MUST reject, with `ErrCheckpointCorrupt`, a scale of 10 000, one above `MaxShockBP`, an unordered or duplicate key, a key that names no configured row, and a `LossBP` above 10 000 (15.3). |
| FR-RSK-037 | No event of the module MUST contain a map, a pointer or a wall-clock time, and the events MUST be built only when an observer was supplied (14.1). |
| FR-RSK-038 | A strike of a cohort MUST take the same share of the whole account whatever its weight, and a store's capacity MUST NOT be multiplied by it (6.3.4). |
| FR-RSK-039 | No rule of the module MUST branch on a commodity, a mode, a region or a store class identifier (architecture.md 9.1). |
| FR-RSK-040 | Every product that this spec names MUST be computed with checked arithmetic, and an overflow MUST be refused as the operation that cannot be applied (architecture.md 3.5). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-RSK-001 | With the default logger and observer, P3.1a, the dispatch step 4a, the arrival step 0 and P8.1a MUST allocate nothing per call, and a full step of the reference chain MUST allocate no more than the baseline of the module off (observability-and-logging.md 8.2). |
| NFR-RSK-002 | The per-dispatch work MUST be at most the number of edges of the path in lookups and words, and the per-tick work of P8.1a at most the number of rows with `P > 0` plus the stores of the limited owners. |
| NFR-RSK-003 | A run MUST be bit-identical for a given seed, configuration and sequence of calls across platforms: no floating point, no map order, no clock (architecture.md 7.4). |
| NFR-RSK-004 | A test that runs over seeds MUST report the failing seed (testing.md 6.2). |

## 21. Open questions

1. **The rounding of a small line** (5.3.2): half to even makes the loss of a one-unit
   line a step at a share of 5 000. A carried remainder per exposure (state) or the stateless
   key `K(t, id)` of `spec-world.md` 10.1.1 compared with the fractional part (no state, no
   draw) would keep the expectation. Decide with the first run that shows a bias that
   matters, which is a statistic over few small lines.
2. **A slot that scales the severity** (3.4.4): one more number a call moves. Reopen when a
   consumer needs a deeper strike rather than a likelier one.
3. **A raider as an agent** (position 1; 7.4.1): a loss that gives the goods to an
   account, a pillage, needs an agent with a strategy and a reason of its own (a transfer,
   not a sink), and a setting with a standing army needs the conflict specs. Out of
   version 1.
4. **Autonomous closure and the seizure of cargo in flight** (10.1.3): an edge status set
   inside a step needs a position and a rebuild of the graph tables there (`spec-world.md`
   8.2.1 says why they are rebuilt at the call). A blockade that seizes cargo needs the
   path of a shipment kept and the evaluation at arrival.
5. **Defence by the size of a settlement** (10.3): the hook H7 of `spec-settlements.md`
   declined. A pack gets it with a consumer's controller (9.4).
6. **The goods in a household's base room** (position 2): a fire, a burglary. The stored
   fraction already says what is not in a store; a base-room row would key on a role.
7. **Journeys and migrants** (N-120): a migrant carries its account with no capacity and is
   not a shipment. A risk on a journey would be a row keyed by edge and a `LossBP` fixed at
   departure, as a shipment's is.
8. **Money** (7.4): a row that takes money needs the answer of open question 3.
9. **Draw anyway at a probability of 0** (3.4.3): taking the word keeps a sweep across 0
   aligned (11.4) and loses the exact neutrality of FR-RSK-005. The choice was neutrality;
   EXP may prefer the other for a design that sweeps through 0.
10. **Timing of a loss** (4, 5): the strike is fixed at dispatch and told at arrival. A loss
    along the way (a wreck on tick 2 of 5) would give a different `Inbound` to the
    consignee and a different moment for an insurer; it needs the path position of a
    shipment and one draw per tick in flight.
11. **The expected loss as seen by an agent includes the scales** (8.1.2). The
    alternative is to show the configured value only and let the shock be a surprise that
    arrives with the cargo.
12. **A per-class share of the stored part by capacity** (6.1.2) is one reading; a
    store could instead hold a share by the time its goods were put in, which needs
    location.
13. **A strike on a facility** (10.3; `spec-infrastructure.md` 21 item 7). The reserved
    shape: a third table `Facilities` keyed (region, facility class) with the same three
    fields (`StrikeBP`, `SeverityBP`, `Common`), one word per facility of the class in the
    region at CAP's P8.3b (a common row once a tick), and a lowering of the condition by
    `SeverityBP` of its current value through the internal form of `AdjustFacilities` with
    a cause `risk`; no goods are lost, so no `loss` and no ledger entry. Not in version 1
    because it adds a capital loss whose loops (capital accumulation, the free rider) are
    CAP's register entries and need a run first, and because a consumer can script it
    between ticks with `AdjustFacilities` today. Whether a facility is struck by its edge or
    by its region is a row key then.

## 22. Where this lives in the code

Nothing is written yet (architecture.md 2.6). When it is: `risk.go` in the root package
(`Risk`, `RouteRisk`, `StoreRisk`, `RouteRiskScale`, `StoreRiskScale`, `RouteLoss`, the
sentinels, the events `StoreLoss`, `RouteRiskScaled` and `StoreRiskScaled`) and
`internal/risk` (stdlib only, over primitive slices: the effective probability, the
fold of the kept share, the loss of a line, the stored fraction and the share of a class,
the expected loss of a path, with the checked 128-bit helper), the two scales as state of
the `World`, the dense arrays derived at construction, and the log keys `share` and
`rows` in `internal/logkeys`. The terms of 19.3 go to the `doc.go` glossary before code
uses them. No `Stateful`, no extension point and no new interface (architecture.md
4.2): the module is data and functions.
