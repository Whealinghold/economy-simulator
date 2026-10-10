# Spec: Merchants: margin estimation, the decision to travel or to hire carriage, the consignee and the delivery contract

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-MER-NNN` and
`NFR-MER-NNN`, which tests cite (testing.md section 4). They are in section 21
(FR-MER-001 to 063, NFR-MER-001 to 005). It also restates `FR-CAR-001` and
`FR-CAR-002`, which `spec-carriage.md` moved here, under their old IDs (9.9).

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
version 1 (`vision.md` section 4). It has two parts, and both are off unless
configured. The **merchant strategy** is a second implementation of `Strategy`,
registered by name, that a role chooses with `Role.Strategy`; a role with no
merchant block and another strategy is untouched. The **delivery contract** is a
world mechanism switched on by `Modules.Merchants`; it works for any strategy that
returns offers and takes, not only the merchant. A world with neither behaves as a
world without this spec. Needs `spec-transport.md` (TRN: modes, `Routes`,
`Inbound`, the shipment and its refusals, haul units), `spec-information.md` (INF:
`Rumours`, `SourceAge`, the lag), `spec-carriage.md` (CAR: the fee is the price of
haul units, `MerchantFeeShare`), `spec-agents.md` (AGT: `Strategy`, the default
strategy this one wraps, the tick), `spec-ledger.md` (the Escrow account and its
three reasons), `spec-regions.md` (REG: the distance and the `region_bound`
refusal) and `spec-config.md` (the module switch and validation). It reads
`spec-external-sector.md` (EXT) and `spec-welfare-and-wages.md` (WEL) and asks each
for an edit (section 12).

**Every number in this spec is a placeholder** except where a derivation is shown.
Nothing has been run in the library, bazaarBot has not been run, and no acceptance
band exists (notes N-015, N-027). The scripts of section 18 are stylised Docker
Python models of the rules, not claims about the library; the rules themselves
are integer. No default is claimed to be stable. No historical fact about
merchants or freight was read or used (N-018). The settings of the worked examples
(section 19), a river valley and a mining colony, are **examples**, flagged as such;
the core and its vocabulary are setting-agnostic, and a commodity, a recipe, a role
or a mode is data (architecture.md 9.1): nothing in this spec branches on one.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

Goods are dear in one region and cheap in another, and the markets of two regions
never meet (`spec-clearing.md` 2.1). TRN moves goods, INF says what an agent may
know of a remote price, and neither decides who moves what, where, or why. This spec
is that decision, and the one mechanism that lets a sale cross a region boundary
without the seller being there. It covers:

- the **merchant**: a `Strategy` implementation registered by name, the role data it
  needs, what it overrides in `Work`, `Trade` and `Revise` and what it inherits from the
  default (section 3);
- what a merchant sees, and what it must not assume (section 4);
- **margin estimation** from a rumour and the local report with integer arithmetic,
  `SourceAge`, the freight and the spoilage, and the exact rule for
  `expected_margin_per_haul_unit` (section 5; README "Still open" 6);
- the decision to **travel** (self-carried, away, a gap in income) or to **hire**
  (buy haul units, redeem them at dispatch), with the cost comparison, the size of a
  lot and what the merchant buys beforehand (section 6);
- the **shipper's stability conditions** that N-100 and N-106 ask of it: count what is
  in flight, cap the dispatch, spread its own thresholds, use `SourceAge`, never
  difference rumours (section 7);
- the **consignee**: who receives a hired shipment, and how it decides what to take
  (section 8);
- the **delivery contract**: a bilateral sale of goods in transit with a price agreed
  before the goods move, escrow, settlement on arrival, failure and default; what it
  adds beyond the book; `FR-CAR-001` and `FR-CAR-002` restated; what is not in version
  1 (section 9);
- **backhaul** (README "Still open" 5, TRN 20.2) (section 10) and the **gate of the
  outside** (section 11);
- the seams with the specs around it, the parameters, validation, observability, the
  state saved in a checkpoint, the register entries, the models run, two worked
  examples, the requirements and the open questions (sections 12 to 23).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| Rumours, their lag, error and `SourceAge` | `spec-information.md` (INF) |
| The shipment mechanism, the 23 refusals, haul units as a market good, the Transit account | `spec-transport.md` (TRN) |
| The price of a haul unit | the market: carriers' recipes and the ordinary book (`spec-transport.md` 8) |
| Whether, when and where a household moves for good | `spec-migration.md` (MIG); a merchant on a trip is away and is not a migrant (12.4) |
| The welfare ratio and the window | `spec-welfare-and-wages.md` (WEL); this spec asks it for one edit (12.3) |
| Household production as a floor | `spec-subsistence.md` (SUB); no seam beyond the strategy registry |
| Loss in transit, theft | `spec-risk.md` (RSK) |
| Credit: an advance against a cargo, a bill, a merchant's loan | `spec-credit.md` (CRD); not in version 1 (9.10) |
| Warehouses and a cost of storing | `spec-storage.md` (STO) |
| Tolls and tariffs | `spec-fiscal.md` (FIS) |
| Associations of merchants, charters | `spec-institutions.md` (ORG) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | Two parts, both off unless configured: the merchant strategy (chosen per role by name) and the delivery contract (`Modules.Merchants`). The contract does not require the merchant, and the merchant needs the contract only for a hired mode | 3.1, 9 |
| 2 | The merchant is a second `Strategy` implementation, `strategy.Merchant`, written the way `spec-agents.md` 6.7 says: it holds a `DefaultStrategy` in a named field and delegates. `Work` is the default's (a role with no recipe requests no job), `Revise` is the default's, `Trade` is the default's with one change: the planned recipe is replaced by a **planned trip**, and the plan gains a shipment, an offer and takes | 3.3, 3.4 |
| 3 | A planned trip is to the merchant what a planned recipe is to a producer: its inputs (the cargo, the haul units or the vehicle and feed, the provisions) are the job's inputs of `spec-agents.md` 7.3, bought by the same three-tier budget and the same limit-price draw. No second set of budget rules | 3.4 |
| 4 | The merchant keeps no state. It values what it holds at what it would cost to replace (the local price), not at what it paid, so it needs no cost basis; the fixed spread of its thresholds is a function of its agent ID; it differences nothing across ticks because it remembers nothing | 3.5, 5.7 |
| 5 | The expected revenue of a lane is the destination's rumour read cautiously: the lower of the rumoured price and the rumoured window mean, and none at all if the rumour's `SourceAge` is above a bound. The expected cost is the local price | 5.2 |
| 6 | One rule decides every mode: the cost of carriage, in money or in time, may be at most a **fee share** of the expected gross margin. `MerchantFeeShare` is that share; the ceiling it puts on a bid for haul units is `FR-CAR-004`; the same share, applied to a self-carried trip, bounds the cost of a tradeoff in time | 5.5, 5.6 |
| 7 | The cost of a tick away is the merchant's own upkeep cost for it, times `TimeCostBP`. It is not a regional wage: a regional average handed to an agent would break the rule that macro is measured and never set | 5.4 |
| 8 | To travel or to hire is decided by the largest expected net gain among the modes that pass the rule of position 6, ties to the lower mode ID. A hired lane is discounted by the cut that a consignee takes | 6.1 |
| 9 | One cargo commodity, one destination and one mode per plan, at most `LotUnits` carrying units: the dispatch cap of TRN is a rule constant here, not a tunable | 6.2, 7.1 |
| 10 | The shipper's stability rests on five rules: a lane lock on its own cargo in flight, a lot cap, a fixed spread of the fee share, a bound on `SourceAge`, and no memory. The model shows the lock halves the spread needed and the spread settles a gap that identical shippers swing by almost twice the freight | 7 |
| 11 | The consignee of a hired shipment is the **payer** of a delivery contract: it buys the goods before they move. A merchant role with `Consigns` posts takes, sized against its stock and its `Inbound` | 8, 9 |
| 12 | A delivery contract is a spot-forward sale: offers and takes are matched in one **round** after the last plan of phase 3, the price is the midpoint, the goods are dispatched at once by the shipment mechanism, the payer's money is in escrow from formation, and the contract is settled on arrival. There is no forward term, no standing order and no credit in version 1 | 9 |
| 13 | The contract has two parties, a payer and a deliverer. The carrier of `spec-carriage.md` is the deliverer when it travels, and the market for haul units when it does not (N-076). `FR-CAR-001` and `FR-CAR-002` keep their IDs and are restated for this shape | 9.9 |
| 14 | Settlement pays the deliverer for the units that reached the destination, delivered or diverted, and refunds the payer for the units that spoiled. The removal of the deliverer refunds the payer in full; the removal of the payer leaves the contract standing with the estate of its region in its place | 9.7 |
| 15 | The contract adds no faucet, no sink and no ledger reason: it uses the three escrow reasons already in the registry, whose owner it takes over. It registers the `region_bound` refusal `open_contract` that `spec-regions.md` 6.5.2 foresaw | 9.6, 9.8 |
| 16 | Backhaul is not priced by this spec. The double count of the round trip stands, it errs towards dearer carriage, a pack states its recipe for one direction to remove it, and the model puts a number on it (a doubled smallest gap) | 10 |
| 17 | The gate of the outside is not a special case. A gate region is an ordinary region with an ordinary market that is a rumour source; an outside node is never a destination; a merchant buys imports in the book and ships them inland like any good | 11 |
| 18 | A traveller reseeds its beliefs on arrival, the default of MIG for a migrant, which TRN 9.3 left to this spec | 12.2 |
| 19 | A merchant has weight 1 and no recipe. It relocates when a trip ends, and it lives where it is: there is no home region | 3.6, 12.4 |
| 20 | This spec asks WEL to count the longest trip of a merchant's modes in the derived window of 2.3.3, which answers N-099 for merchants | 12.3 |
| 21 | The merchant closes loops, and each has a register entry: the price-gap loop with its three delays (L84, L85, L93, L94, L95), the importers' take loop, the derived demand for haul units with a choke price, the escrow, and the drift of merchants between regions | 17 |

## 3. The merchant strategy

### 3.1 Registration, name and home

3.1.1. The merchant is a `Strategy` (`spec-agents.md` 6.2) registered under the name
`"merchant"`. A role chooses it with `Role.Strategy = "merchant"`; the name is data, is
in the effective configuration and so in its hash (`spec-agents.md` 6.6), and an
unregistered name is `ErrUnknownStrategy` at `NewWorld`. The code behind the name
cannot be hashed (ADR 0002, open point 1), which is the consumer's to manage as for any
strategy.

3.1.2. **Where it lives, and what moves.** The root package cannot import `strategy/`
(architecture.md 2.7), so the library cannot register the merchant by itself.
`architecture.md` 2.6 creates `strategy/` "once there is a second strategy", and this
is it. Nothing that exists moves: `DefaultStrategy`, `Directed` and the contract of
`Strategy` stay in the root package (4.5 there). What is new:

- `strategy/merchant.go`: the type `strategy.Merchant`, a struct with one named field,
  `Default economysim.DefaultStrategy`, and the three methods; the function
  `strategy.Options()` returns the `economysim.Option` values that register the
  standard names of the package (`"merchant"` today), so a consumer writes
  `economysim.NewWorld(cfg, strategy.Options()...)`;
- `internal/merchant`: the pure rules over primitive slices (the margin, the fee share,
  the spread, the sizing of a lot, the matching and the settlement arithmetic), which
  imports the standard library and `internal/household` only (architecture.md 2.9);
- the root package keeps the world side: the contract round, the escrow, the plan
  fields and the view group (23).

A configuration that names `"merchant"` still needs one Go line, the registration: code
cannot be in a file (`spec-agents.md` 6.6). That is the only Go a world with merchants
needs beyond the world without them, and the configuration-only chain test
(testing.md 6.4) is about commodities, recipes and roles and is unaffected.

3.1.3. The strategy is **stateless** and takes its randomness from the world as every
strategy does (architecture.md 7.7). It draws exactly what the default would draw for
the same intents (one `uint64` per intent it keeps or drops, `spec-agents.md` 7.5.3) and
nothing of its own: the spread of 5.7 is a function of the agent ID, so it needs no
draw at creation and adds nothing to the list of `spec-agents.md` 3.5.2.

### 3.2 The role data

A merchant role carries a block `Role.Merchant` (in a file `[role.<id>.merchant]`), a
pointer: nil is not a merchant role. The role keeps its other fields: `Trades` lists the
commodities it deals in and the inputs of its modes (3.2.2), `Informed` must be true
(14.1), `Recipes` must be empty (3.6), `MerchantFeeShare` is the centre of the fee
share (`spec-carriage.md` 3).

| Field | Meaning | Default | Bound |
|-------|---------|---------|-------|
| `Modes` | the modes the merchant plans with, by ID | required, not empty | known modes, no duplicate, at most `MaxModes` |
| `LotUnits` | the carrying units a dispatch may use at most: the lot cap (7.1) | 1 | 1 to `MaxLotUnits` |
| `FeeShareSpreadBP` | the half-width of the fixed spread of the fee share by agent (5.7) | 2 500 | 0 to 5 000, at most `MerchantFeeShare`, and `MerchantFeeShare` plus it at most 10 000 |
| `MaxSourceAge` | the oldest `SourceAge` of a rumour the merchant acts on (5.2) | 2 | 0 to `MaxRumourAge` |
| `TimeCostBP` | the cost of one tick away, in basis points of the merchant's own upkeep cost for a tick (5.4) | 10 000 | 0 to 100 000 |
| `ConsigneeCutBP` | the share of the destination price that the merchant expects to give up to a consignee when it sells under a contract (6.1) | 500 | 0 to 5 000 |
| `Consigns` | the merchant posts takes: it buys delivered goods as a consignee (8) | false | a bool |
| `TakeMarginBP` | the discount to its local price at which a consignee takes (8.2) | 1 000 | 0 to 9 000 |
| `TakeUnits` | the units of a cargo commodity a consignee wants to hold or have on the way (8.2) | required with `Consigns` | 1 to `MaxQuantity` |

3.2.1. `MerchantFeeShare` keeps its owner, its row and its default of 5 000
(`spec-carriage.md` 3); this spec uses it and adds no second name for it. Every other
field is owned here and has a row in the parameter-owner table (13).

3.2.2. **The cargo set and the trade inputs.** From the block and the configuration the
world derives, once, two sets of commodities. The **trade inputs** of a merchant role are
the `Haul` commodity of each of its hired modes and the `Vehicle` and the `Feed`
commodities of each of its self-carried modes. The **cargo set** is `Trades` less the
trade inputs: the goods it buys to sell elsewhere. Validation requires the trade inputs to
be in `Trades` (they must be in the tradable set to be bought) and the cargo set not to be
empty (14.1). A commodity may be both cargo and upkeep: a merchant that trades the staple
it eats is the main case, and 3.4 handles it with no special rule.

### 3.3 What it inherits and what it overrides

| Method | The merchant's behaviour |
|--------|--------------------------|
| `Work` | the default's: a role with no recipe requests no job (`spec-agents.md` 7.1.1). Nothing is overridden |
| `Revise` | the default's: `DefaultStrategy.Revise`, one update per admitted order with the draw the world took (`spec-agents.md` 7.6). The merchant's beliefs about its own market are those of any agent |
| `Trade` | the default's, with three changes: (1) the planned unit of work is a **planned trip** instead of a planned recipe (3.4); (2) the limit of a bid for the cargo and for haul units is lowered to a ceiling the trip can bear (5.6); (3) the plan gains at most one **shipment request** or one **delivery offer** when the trip is ready (6.3), and the **takes** of a consignee (8) |

The tradable set, the targets, the three tiers, the saving rule, the favourability, the
urgency of a hungry household, the limit-price draw and the affordability rule of
`spec-agents.md` 7.3 to 7.5 are the default's, unchanged, and the merchant calls the same
pure functions of `internal/household`. A merchant therefore eats before it trades, keeps
its reserve, and sells the surplus of anything it holds that no trip needs, at the draws
the default would take.

### 3.4 The planned trip

3.4.1. In `spec-agents.md` 7.2 the **planned recipe** `r*` is what the agent expects to do
next, so that `Trade` can buy its inputs. For a merchant the thing it expects to do next
is a **trip**: a lane (a destination region `r` and a cargo commodity `c`), a mode `m`, and
a lot `q`, chosen by sections 5 and 6 from the view of the tick, or none. The trip is
chosen afresh every tick from the view and nothing of it is remembered (3.5).

3.4.2. The **inputs of a trip** are the `Job_c` of `spec-agents.md` 7.3, in whole units:

| Commodity | `Job_c` |
|-----------|---------|
| the cargo `c` | `q` |
| the `Haul` commodity (a hired mode) | `n * s` (3.3.3 of TRN): the carrying units the lot needs times the haul units per carrying unit |
| the `Vehicle` (a self-carried mode with one) | `n`, the carrying units the load needs |
| each `Feed` commodity | `quantity * n * k`, `k` the travel time |
| each upkeep commodity `c` (a self-carried mode) | `w * u_c * max(0, k - CoverTicks)`: the provisions that the check `unprovisioned` of `spec-transport.md` 5.4 asks for, less the `CoverTicks` of upkeep that the default already holds (`spec-agents.md` 7.3) |

where a commodity that is on two lines (the staple that is cargo and upkeep) takes the
sum. With these as `Job_c`, and `StartsIn` 1 (a merchant has no job), the default's
computation gives `Keep`, `Target`, the gap and the three tiers: the cargo and the haul
units are tier 2, the inputs of the trip, **not subject to the saving rule**, because a
merchant that could not buy its stock would be a poverty trap like a producer that could
not buy its inputs (`spec-agents.md` 7.4.3). Their purpose is `PurposeInput`
(`spec-agents.md` 7.3.6), so the money spent on them is netted from the merchant's income
when it is spent, and the sale, when it comes, is income: `spec-welfare-and-wages.md` 2.1
reads the merchant's profit and not its turnover. That is the right reading of a
merchant's cost of goods; the wording of `spec-agents.md` 7.3.6 ("the next job") needs the
edit of 12.1.

3.4.3. **Nothing is held back for a trip that is not planned.** With no trip, `Job_c` is
0 for every commodity, `Keep` is the household's own, and the default sells whatever the
merchant holds above it, the unsold cargo of an abandoned trip included. A merchant
therefore carries inventory only while a trip is planned, and the cost of an unplanned
inventory is bounded by one tick (it is offered at once).

### 3.5 No state

3.5.1. The merchant strategy keeps no state, so it needs no save and load pair
(ADR 0002 decision 7), and it cannot herd by memory. The consequences are decisions, not
accidents:

- **Replacement cost, not cost basis.** It values a unit it holds at the local price it
  would pay to buy it again (5.4). A sunk cost has no place in a decision about the
  future, and a cost basis would be per-agent per-commodity state that grows with
  merchants and commodities and must be saved. The price paid is not remembered; the loss
  of a trip that went wrong is not remembered either (open question 4);
- **No difference of rumours across ticks.** The strategy reads the rumours of the view
  and nothing older, so the step of the vintage when the graph changes (L95, INF-A4) cannot
  mislead it (7.1);
- **No cooldown, no contract book.** It knows no contract it is party to. It knows its
  cargo in flight through the view (4.1), which is the world's record.

3.5.2. What a merchant does with a failed offer is the same as what it does with an
unfilled order: nothing is kept, the tick's offer expired, and the next tick the trip is
chosen again from the then-current view.

### 3.6 A merchant has weight 1 and no recipe

3.6.1. A merchant role has `Recipes` empty and its groups have weight 1 (a `Scale` of 1,
so `Households` agents). The reasons: a self-carried trip takes a whole agent away and a
cohort travels whole (`spec-transport.md` 3.2.4), `spec-scale.md` 4.7 makes a named
merchant an individual, the first consumer's world needs no weight above 1 (N-030), and a
cohort of identical merchants would be the lockstep this spec's rules exist to prevent
(L13, L34). A recipe would make the trip's `job_in_progress` refusal and the working-tick
reading of `spec-welfare-and-wages.md` 4.3 depend on two things at once. Both are
validation errors (14.1), as `traineeship_needs_weight_one` is for a trainer. A producer
that also deals is two roles.

3.6.2. A merchant is an agent like any other for everything this spec does not change:
it is born with `StartMoney` and `StartStock` (`spec-agents.md` 3.5), its upkeep is
consumed in phase 6, it can go unfed, the ladder of `spec-spawner.md` 5 acts on it, and it
is removed with its goods to the estate (`spec-agents.md` 3.6), with the contract rule of
9.8 added.

## 4. What a merchant sees

### 4.1 The merchant group of the view

The `AgentView` of the trade phase (`spec-agents.md` 6.4.1) already carries what a
merchant mostly needs: the local market report of every commodity of its tradable set (the
price, its age, the best unmatched quotes, the window), the agent's money, upkeep and
beliefs, `Inbound` and `Routes` with the transport module on (`spec-transport.md` 9.1) and
`Rumours` for an informed role (`spec-information.md` 6.2). This spec adds one group, plain
data, a scratch copy valid for the call only like the others, present only in the view of
an agent whose role has a merchant block:

| Field | Content |
|-------|---------|
| `Params` | the role's block of 3.2 by value, and `MerchantFeeShare` |
| `Modes` | for each mode of `Params.Modes`, in mode order: hired or self-carried, `UnitMass`, `UnitBulk`, `Crew`, `Vehicle`, `WearPerTick`, the `Feed` entries, `Haul`, `HaulSpan`: the definitions of `spec-transport.md` 3.1 by value |
| `Goods` | for each commodity of the tradable set, in canonical order: `Mass`, `Bulk`, `ShelfLife`, `Durability` (the definitions of `spec-goods.md` 8), and `OldestAge`: the ticks since the born tick of the oldest lot the agent holds, or -1 when it holds none |
| `Outbound` | for each destination region and commodity with units on the way: the total quantity of the shipments the agent is the **shipper** of (a hired shipment, or a trip it is not on) and the earliest arrival tick, from the shipments dispatched before the start of phase 3, so that it does not depend on the order in which agents are called, as `Inbound` does not |

`Outbound` is the mirror of `Inbound` and is TRN's data: the same shipment list, read for
the other party. It is not an aggregate: it is the agent's own cargo, and an agent may
react to its own state (`AGENTS.md`, "Macro is measured"). The agent's own `Inbound` is its
pipeline as a consignee (8). Both are bounded by the commodities times the regions.
`Goods` is a static table by value, as the recipes of the view are, and costs a copy
bounded by the tradable set. An agent that is away is passed no view (`spec-transport.md`
5.5.2), so a merchant on a trip decides nothing until it arrives.

### 4.2 What a merchant must not assume

That a rumour is a price at which it can trade; that the lag of a pair is constant, or the
set of rumours the same from one tick to the next (`spec-information.md` 6.6); that its own
cargo has not moved the price it left behind (the purchase at the origin is the part of the
effect that comes at dispatch, 7.2); that a consignee exists (the offer may find no take);
that a haul unit it holds is worth what it paid (it is valued at the market price, and it
spoils with the commodity's shelf life). These are properties of the situation the rules
of sections 5 to 7 are written for.

## 5. Margin estimation

### 5.1 The lanes considered

5.1.1. A **lane** is a pair (destination region `r`, cargo commodity `c`). A **candidate**
is a lane with a mode `m` of `Params.Modes` that has a route to `r` (`Routes`, `Distance`
exists and the travel time `k` is at most `MaxTravelTicks`). The merchant evaluates every
candidate in ascending (`r`, `c`, `m`) order; the order decides ties and counts only.

5.1.2. Each candidate is decided by the first of these that applies; the count of
`MerchantsPlanned` (15.1) names the result:

| # | Case | Result |
|---|------|--------|
| 1 | the view has no rumour for (`r`, `c`) | none: `NoRumour` |
| 2 | the rumour's `SourceAge` is above `MaxSourceAge` | none: `Stale` |
| 3 | `Outbound` has units of `c` on the way to `r` | none: `Locked` (7.1) |
| 4 | the commodity has a shelf life and the travel time is above the life left (5.3) | none: `Perishable` |
| 5 | the cautious revenue, net of the consignee's cut for a hired mode, is not above the local price | none: `NoMargin` |
| 6 | the lot is 0 (5.8, 6.2) | none: `Unaffordable` |
| 7 | the cost of carriage is above the fee share of the gross margin (5.5) | none: `BelowShare` |
| 8 | otherwise | a passing candidate, with a score (6.1) |

The merchant plans the passing candidate with the greatest score, ties to the lower `r`,
then `c`, then `m`; with none, it plans no trip. At most one trip is planned: one cargo
commodity, one destination and one mode (6.2).

### 5.2 Reading a rumour

5.2.1. For the rumour `rho` of (`r`, `c`) (`spec-information.md` 5.1) the **cautious
revenue** per unit is

```
Rhat = rho.Price                        if rho.Mean = 0
Rhat = min(rho.Price, rho.Mean)           otherwise
```

and, for a hired mode, `Rnet = Rhat * (10 000 - ConsigneeCutBP) / 10 000` (truncated),
which is the revenue after the share a consignee keeps (6.1); for a self-carried mode
`Rnet = Rhat`. The window mean is the smoother of the two readings and the published price
the latest; the lower of them does not extrapolate a spike. The rumour is an old reading
blurred by an error (`spec-information.md` 5.4, 5.5): the merchant uses `Price` and `Mean`
as they are and does not undo the error, which has mean zero.

5.2.2. **`SourceAge`.** A source market that did not trade in the tick publishes its
carried-forward price with a positive age (`spec-information.md` 4.2.2). A gap against
such a price can persist for ever because nothing moves it (L94, INF-A3), and goods
shipped into it arrive to find no buyer at the rumoured price. The merchant therefore
acts on a rumour only if `SourceAge <= MaxSourceAge` (default 2, so a source that traded
within the last three ticks of its record). The condition INF-A3 states, "small against
the time its goods take", is a number the author sets: the default is a placeholder, and a
pack with a long route or a thin market sets it from the route (open question 5).

5.2.3. **The lag is not used as a number.** The rumour is `Lag` ticks old and the goods
need `k` more; the price may move in the `Lag + k` ticks, and no rule can know how. The
delay is dealt with where it matters, in the stability conditions (7.2): the lot lock
removes the pipeline the merchant itself created, the spread and the share keep the herd
from acting together on the same stale reading, and the share is the margin of safety.

### 5.3 Perishables

5.3.1. A unit that spoils is delivered only if it is still fresh on arrival
(`spec-transport.md` 5.6.2): a lot born at `b` with shelf life `S` shipped at `t` over `k`
ticks is delivered if and only if `t + k <= b + S - 1`. The ledger moves the oldest lots
first (`spec-ledger.md` 5.2.1), so a line of `q` units takes the oldest held lots, and the
line is safe only if the oldest survives. The view gives `OldestAge`, so the merchant
computes the **life left** of its cargo:

```
life = S - 1 - max(1, OldestAge)         for a commodity with S > 0
```

with `max(1, ...)` because goods bought in tick `t` settle in phase 4 and cannot be sent
before tick `t + 1`, so the youngest lot a merchant can ship is one tick old. A candidate is
`Perishable` if `S > 0` and `k > life`. A commodity with `S = 0` never spoils.

5.3.2. The merchant does not ship part of a line to save the young lots: it ships `c` or it
does not. A mixed-age stock whose oldest lot cannot survive blocks the commodity until the
oldest has spoiled or been sold (the default sells it as surplus, `spec-agents.md` 7.3.5).
This is a simplification (open question 6).

### 5.4 The cost of a trip

Let `n` be the carrying units the lot needs, `n = max(1, ceil(q * Mass / UnitMass),
ceil(q * Bulk / UnitBulk))` with a term whose unit capacity is 0 left out
(`spec-transport.md` 3.2.1), `s` the haul units per carrying unit (`Routes`), `k` the travel
time. Prices are the local market prices of the report (`Price`, the effective reference
price when the market has never traded), so every input of the trip is valued at what it
costs to buy now. The **cost of carriage** `C` is:

- **hired:** `C = n * s * P_haul`, `P_haul` the local price of the mode's `Haul` commodity;
- **self-carried:** `C = wear + feed + time`, where
  - `wear = ceil(n * WearPerTick * k * P_vehicle / Durability_vehicle)` (0 with no
    `Vehicle`), the wear `WearItems` will charge priced as `spec-production.md` 9 prices a
    tool's wear;
  - `feed = sum over Feed entries of  quantity * n * k * P_feed`;
  - `time = k * UpkeepCost * TimeCostBP / 10 000` (truncated), with `UpkeepCost` the cost of
    one tick of the agent's effective upkeep at its own belief centres
    (`spec-agents.md` 7.4.2).

All sums are checked. **The time of a trip is paid at the merchant's own cost of living,
not at a wage.** A merchant that goes away for `k` ticks gives up `k` ticks in which it
could eat from what it earns here; `TimeCostBP` of 10 000 (the default) asks the trip to
pay at least for the upkeep of the days it takes, and a higher value asks it to pay for a
standard of living above bare upkeep. The pooled observed wage of the region would be the
opportunity cost of a household with a recipe, but it is an aggregate, and an agent may
not react to an aggregate handed to it (position 7; the cost anchor of `spec-production.md`
9 is the world's, given to the agent as a number it did not ask for, and is a different
case: it anchors a price, not a decision to leave).

### 5.5 The fee-share test and the exact rules

5.5.1. For a candidate with a lot `q`, a cautious revenue `Rnet` and an acquisition price
`P_acq`, the local price of `c` in the report, the **gross margin** is

```
G = q * (Rnet - P_acq)                    positive (case 5 of 5.1.2)
```

and the agent's **fee share** `phi` is `MerchantFeeShare` with the agent's fixed spread
(5.7), in basis points. The candidate passes if and only if

```
C * 10 000 <= phi * G                        (128-bit intermediates, checked)
```

that is, if the cost of carriage is at most the fee share of the expected gross margin. It
is the rule `FR-CAR-004` states for haul units, applied to every mode, in money for a hired
mode and in time, wear and feed for a self-carried one. The share is a margin of safety on
a reading that is `Lag` ticks old and a delivery that is `k` ticks away: at 5 000 the trip
must promise twice its cost.

5.5.2. **The score** of a passing candidate is `G - C`, the expected net gain.

5.5.3. **The break-even and the ask.** Two prices follow from the same inequality, both
rounded against the merchant:

```
margin_needed = ceil(C * 10 000 / (phi * q))          per unit
BreakEven     = Rnet - margin_needed                 the dearest price at which the lot still passes
AskLimit      = P_acq + margin_needed                the cheapest delivered price at which it passes
```

`BreakEven` is the ceiling of the merchant's bid for the cargo (5.6); `AskLimit` is the
limit of its offer in a contract (9.2). A `BreakEven` below 1 means no bid.

### 5.6 `expected_margin_per_haul_unit`

This is the answer to the open item 6 of the README ("Still open") and to
`spec-carriage.md` 5.2. For a hired candidate with `n * s` haul units:

```
expected_margin_per_haul_unit = G / (n * s)                       truncated
HaulCeiling                   = expected_margin_per_haul_unit * phi / 10 000   truncated
```

with `G` net of the consignee's cut (5.2.1). The merchant's bid for haul units has the limit
`min(L, HaulCeiling)`, `L` the limit of the default's draw (`spec-agents.md` 7.5.3), and the
bid for the cargo has the limit `min(L, BreakEven)`. The ceiling is **`FR-CAR-004`**: the
merchant never bids for a haul unit more than the fee share of the margin it expects to earn
from it, and with the spread of 5.7 `phi` is the agent's own share, equal to
`MerchantFeeShare` when `FeeShareSpreadBP` is 0. The ceiling makes the merchant's demand for
haul units end at a price: above it, no merchant of the lane buys (MER-A4).

### 5.7 The fixed spread of the fee share

5.7.1. The fee share of agent `a` is

```
w(a)   = (a * 2 654 435 769) mod 2^32                      the Weyl offset of spec-population.md 3.8.3
u(a)   = floor( w(a) * 10 000 / 2^32 )                       in [0, 10 000)
phi(a) = clamp( MerchantFeeShare + FeeShareSpreadBP * (2 * u(a) - 10 000) / 10 000, 1, 10 000 )
```

with the division truncated toward zero. `w` is the Weyl offset that `spec-population.md` 3.8.3 uses for the start of
a death carry (Knuth's multiplier, 2^32 over the golden ratio), and the spread of margins of MIG is the same function of the ID:
there is one rule for a fixed per-agent spread in the library, and agents with consecutive IDs get values spread evenly over
the interval. The product is the 32-bit product of an unsigned integer and the multiplier, which wraps modulo 2^32 by
the definition of the type. **This is the one kind of place in the module where arithmetic is modular by design**: `w` is a
hash of an identifier, not a quantity, and nothing that affects money or goods wraps. The multiplier is a rule constant
(architecture.md 9.6), not a parameter; the helper belongs where the three users can share it (23).

5.7.2. **Why it is a function of the ID and not a draw at creation.** `spec-information.md`
11.2 and `spec-stability.md` L93 ask for "a fixed spread of the shippers' own thresholds,
drawn at creation, not per tick". The property that matters is that the spread is fixed per
agent and covers the interval evenly, and a function of the ID has both without a cost the
draw has: a draw adds an entry to the list of creation draws of `spec-agents.md` 3.5.2 for
every agent whatever its role, a field to the agent and to the checkpoint, and an unevenness
(ten uniform draws leave large gaps and clumps by chance, the Weyl values of consecutive IDs are closer to even, `models/mer/lane.out`). The
cost is that the spread of a role is the spread of the IDs it happens to have: with
consecutive IDs (the initial agents of a group) it is even; with IDs a few apart the largest
gap between neighbours is up to three times the ideal for ten merchants, which the model
shows and the `merchant_spread_narrow` warning does not see. A role that needs an exact spread
raises `FeeShareSpreadBP`. A draw at creation is open question 3.

5.7.3. The spread is **in the fee share, so in the bar**: a merchant with a lower share demands
a larger margin over its cost before it ships, a bar of `10 000 / phi` times the cost. The
widest the bars can be is from `10 000 / (MerchantFeeShare + FeeShareSpreadBP)` to
`10 000 / (MerchantFeeShare - FeeShareSpreadBP)`; with the defaults 1.33 to 4 times the cost,
a width of 2.67. A bar is never below 1 times the cost, because a share is at most 10 000: a
merchant never ships on an expected loss, which is where it differs from the model of
`spec-information.md` 11.2, whose bars straddle the freight.

### 5.8 Overflow and rounding

Every product above is checked (architecture.md 3.5). A candidate whose arithmetic does not
fit is dropped as `Unaffordable`. Rounding: divisions truncate, except `ceil` where written;
the costs are rounded up and the revenues down, so that every rounding is against the
merchant. The midpoint of a contract is half to even (9.4).

## 6. To travel or to hire

### 6.1 The cost comparison

6.1.1. The merchant compares modes by the **same rule on the same lane**: for a lane (`r`,
`c`), each mode `m` of `Params.Modes` with a route gives its own lot (6.2), cost `C` (5.4) and
revenue (5.2), and the passing candidate with the greatest `G - C` wins. A hired mode is
valued with the consignee's cut, so its gross margin is lower than a self-carried lane's by the cut, and a
hired lane is chosen over a self-carried one when its freight is less than the wear, the feed and the
time of the trip by more than the cut costs. The comparison uses nothing the merchant
cannot see: the market price of haul units and of the vehicle and the feed in its own market,
its own cost of living, and the routes.

6.1.2. **Where each wins.** A self-carried trip pays for time, and time grows with the distance
and does not depend on the size of the lot: it wins for a small, valuable lot and a short trip,
or for a trip with a return load (10). A hired trip pays by the haul unit, which grows with
the lot's binding fill and the distance in whole spans, and it wins for a large, bulky lot (the
numbers of 19.1, an example: a lot of 6 units of a bulky good on a small self-carried mode fails the test, the same lane by a hired mode passes). Neither
rule is a preference of the code; a pack that wants only one lists only that mode.

### 6.2 Sizing the lot

6.2.1. The lot is bounded in this order, and the smallest bound applies:

1. **The cap of the lot.** `q_cap = Units * per_unit`, with `Units = LotUnits` carrying units and
   `per_unit` the units of `c` that one carrying unit holds,
   `min(floor(UnitMass / Mass), floor(UnitBulk / Bulk))` with a term whose capacity is 0 left out.
   For a self-carried mode the pooled capacity is also used by everything else the agent holds
   (`spec-transport.md` 3.2.2), so `q_cap` is the most units of `c` that fit after the other
   holdings, by the same pair rule, and never more than `LotUnits`. A self-carried mode in a merchant block has `Crew` 1, because the
   merchant has weight 1 (3.6; 14.1), so the vehicles the lot needs are `n`, bought as an input (3.4.2). This is
   the cap of D39 applied to a merchant (7.1).
2. **What it can pay.** `q_aff = floor((Money - ReserveMoney - other) / LimitMax)`, with
   `ReserveMoney` the default's reserve (`spec-agents.md` 7.4.2), `other` the cost at their
   maximum limits of the trip's other inputs it does not yet hold, and `LimitMax` the
   dearest price at which it would bid for `c`, `min(High, BreakEven)`. A merchant never plans a
   lot that would leave it below its reserve.
3. **What it holds,** when it is ready to dispatch (6.3): the free units of `c`, the units above
   what the household keeps.

6.2.2. `BreakEven` depends on the lot (`margin_needed` falls as `q` grows), so the rule is two
passes: size with `q = min(q_cap, q_aff at LimitMax = High)`, compute `BreakEven` at that `q`,
recompute `q_aff` with `LimitMax = min(High, BreakEven)`, take the smaller `q`, and evaluate the
test and the prices at that final `q` once more. The result is deterministic and bounded; a
lot that is 0 after the second pass is `Unaffordable`.

6.2.3. **Lumpiness.** A hired lot pays for whole carrying units (`spec-transport.md` 8.4), so a
lot below one unit pays for one: the cap `q_cap` is a multiple of `per_unit` and a merchant that
can afford less ships a smaller, dearer lot or none. The freight of a small cargo is a large
share of its value, and the test says so.

### 6.3 Preparing and dispatching

6.3.1. A trip is **ready** when the merchant already holds all its inputs, in the lot it can
send: the cargo (at least one unit, `q_ready = min(q, free units)`), the `n * s` haul units or
the vehicles, the feed and the provisions. When it is not ready, it **prepares**: it buys what is
missing, which the default's tiers do from `Job_c` (3.4.2), and it sends nothing. The cargo and the
haul units arrive in phase 4 and can be sent in phase 3 of the next tick, so a trip takes two ticks
from the decision to the dispatch, the delay that a model must carry (7.2).

6.3.2. A ready trip is evaluated again at the lot it holds, `q_ready`, with the test of 5.5. If it
passes:

- **self-carried:** the plan gains the request `ShipmentRequest{Mode: m, To: r, ToSelf: true}`:
  the merchant travels with everything it holds, sells at the destination in the ordinary book
  (the default's surplus rule, there), and its account is in `r` when it arrives
  (`spec-transport.md` 5.7);
- **hired:** the plan gains the offer `DeliveryOffer{Mode: m, To: r, Commodity: c, Quantity:
  q_ready, Limit: AskLimit}` (9.2). The offer is matched in the round of the same tick, after the
  last plan.

If it does not pass, nothing is sent, and the default sells what is not needed (3.4.3).

6.3.3. A ready trip's inputs are not bought again: the gap of each input is 0, so its tier is 0.
A trip whose conditions fail between the decision and the dispatch leaves the merchant holding
inputs it paid for: the cargo is sold by the default as surplus, the haul units are sold or spoil
(a haul commodity has a short shelf life, `spec-transport.md` 8.4), and a vehicle is kept (it is a
durable of the role's modes; the default offers a tool only to a role that makes it,
`spec-agents.md` 7.3.5).

### 6.4 When a merchant stays

A merchant plans no self-carried trip, and counts the case `Waiting`, when:

- it has `Inbound` goods (as a consignee, 8): it would leave its region with its account still there, arrive
  elsewhere before the cargo, and find the cargo diverted to the estate of the region it left
  (`spec-transport.md` 5.8);
- it has a job in progress: a merchant has no recipe of its own but the household floor of SUB (`spec-subsistence.md`)
  may give a role with none an own-use list, and the default `Work` that a merchant inherits then starts an
  own-use job, which the shipment mechanism refuses a trip for (`job_in_progress`, `spec-transport.md` 5.4).

A merchant that is in the own-use mode of SUB (the flag the view carries beside `Hunger`) plans no trip at all and
posts no take: it is short of food, and the household feeds itself before it trades. A hired trip needs nothing
of the first two.

### 6.5 The work share

The merchant plans a trip and posts takes only if the work share of `spec-agents.md` 7.1.2 is
above 0 for it: the survival rule and the floor of 7.1.3 make it 10 000, and a ratio at or above
`T_hi` makes it 0. A merchant that has reached its target income rests as a producer does: it plans
no trip and posts no take, and still eats, sells its surplus and keeps its reserve. For an agent
of weight 1 the share is a switch (`spec-agents.md` 9.3). Case: `Rested`. This is the target-income
rule, with the same damper and the same loop (MER-A3).

## 7. The shipper's stability conditions

The price-gap loop (L84) is closed by the response of the shipper, and this is that
response. `spec-transport.md` 15 (TRN-A1) found that a shipper which responds to a gap with a
transit delay `k` is stable only below a gain; `spec-information.md` 11 (INF-A1, INF-A2) found
that a rumour adds its lag to the delay and that identical threshold shippers herd. N-100
and N-106 hand the conditions to the merchant. It meets them with rules, not with a gain:
a lot is shipped or it is not, and the stability comes from how many may be in flight and how
alike the merchants are.

### 7.1 The rules

| # | Rule | What it damps | Cost | Counted by |
|---|------|---------------|------|------------|
| R1 | **Lane lock.** No new lot on a lane (`r`, `c`) while the merchant has units of `c` on the way to `r` (`Outbound`). A self-carried merchant is away and decides nothing, which is the same rule | the invisible pipeline of its own: it cannot re-send what has not yet arrived (L85, L84) | one lot per lane per `k + 1` ticks | `Locked` |
| R2 | **Cap.** At most one trip per plan (one commodity, one destination, one mode) and at most `LotUnits` carrying units; the plan carries one request or offer, far below `MaxDispatchPerTick` | the size of the swing: the gap moves by at most the cap times the merchants (D39) | a large shipment is several lots over several ticks | `LotCapped` |
| R3 | **Fixed spread.** The fee share, and so the bar, differs by agent, fixed for the agent for ever (5.7) | herding of identical shippers on one reading (L93) | merchants with a high bar never ship the thin gaps | not silent: the bar is in every decision |
| R4 | **Source age and a cautious reading.** A rumour with a `SourceAge` above `MaxSourceAge` is not acted on; the revenue is the lower of the price and the mean (5.2) | the phantom gap of a thin market (L94) | a thin source market is never served | `Stale` |
| R5 | **No memory.** Only the rumours of the view are read, none is differenced with an earlier one (3.5) | the step of the vintage when the graph changes (L95) | none | not applicable: there is no state to count |

R1 and R2 are the **counts** that N-100 asks ("count `Inbound`, cap dispatches"): R1 counts the
merchant's pipeline as a shipper (`Outbound`), 8.2 counts it as a consignee (`Inbound`). Neither
counts anyone else's cargo: no agent sees another agent's pipeline, and the market-wide pipeline
stays invisible, which is what R2 and R3 are for.

### 7.2 The gain bound, with the delay of the rumour

7.2.1. Let `Lag` be the age of the rumour (`spec-information.md` 5.2), `k` the travel time, and `n = Lag + k + 1`
the total delay of the loop: the age of what the merchant reads, plus the tick between the purchase and the
dispatch (6.3.1), plus the journey. It is counted as `models/inf/shippers.py` counts it (the age of the rumoured
gap and the ticks from the decision to the arrival). INF-A1 counts the extra age of the rumour over the local
report, `ell = Lag - 1`, and the travel time, `ell + k`; the merchant has two ticks more, the tick of the local
report and the tick of preparation. A merchant that dispatches a lot raises the price at its origin at once by the part of the
effect that comes at dispatch (its purchase), and lowers the gap at the destination `k` ticks later
by the rest. Let `Q` be the change of the gap if every merchant of the lane shipped one lot in one tick,
and `W` the dispersion of the merchants' bars in units of the freight (5.7.3). In the linearised model
of `models/inf/shippers.py`, which `models/mer/lane.py` extends, the aggregate response to the gap is a
step of height `Q` for identical bars and a slope `Q / W` for bars spread evenly over `W`, and the gap
settles if the slope is below the limit of the delayed feedback:

```
Q / W  <  2 sin( pi / (2 (2 n + 1)) )         n = Lag + k + 1      (no lock)
```

(0.347 for `n` = 4, 0.241 for `n` = 6, 0.149 for `n` = 10). With the share of the effect at dispatch at
one half or more, or with the pipeline counted, the limit is about 2 for any `n` (`spec-transport.md`
15, TRN-A1; `models/trn/pipeline.out`).

7.2.2. **What the lane lock does to the condition.** It limits each merchant to one lot per `k + 1` ticks,
which caps the response and breaks the lockstep of merchants that shipped together. The model finds the
spread that settles the gap (a peak-to-peak size under a tenth of the freight) for `Q` = 0.5 f:

| Rumour age, decision to arrival | limit on `Q / W` | `W` needed by the formula | `W` found, no lock | `W` found, lock |
|---------------------|-----------------:|--------------------------:|-------------------:|----------------:|
| 2, 2 | 0.347 | 1.44 f | 1.45 f | 0.80 f |
| 4, 2 | 0.241 | 2.07 f | 1.85 f | 0.95 f |
| 2, 4 | 0.241 | 2.07 f | 1.85 f | 0.55 f |
| 6, 4 | 0.149 | 3.35 f | 2.55 f | 0.65 f |
| 8, 8 | 0.095 | 5.25 f | none up to 3 f | none up to 3 f |

The formula is the condition without the lock and the model agrees with it to the step of the grid;
the lock needs between about a half and a quarter of the spread. For a total delay of 16 ticks neither works
at a spread of 3 f and a lot of half the steady flow: **a pack with a rumour lag and a transit time that
add to many ticks must cut the lot (R2) or accept a swing** (7.3).

7.2.3. The condition cannot be checked at validation, because `Q` is the effect of a lot on a gap and
that needs the depth of two markets, which only a run shows. What validation can check is the width
`W` the configuration gives (`merchant_spread_narrow`, 14.2) and the lot cap. The scenario MER-S1 runs
the lane with the cap on and off and asserts the bound. Status: **model**, a linearised gap with no
inventory, belief, sizing or market, and a share at dispatch that is not derived for any pack.

### 7.3 Herding of many merchants on one stale gap

The model (`models/mer/lane.out`), 50 merchants, a rumour age of 2, a decision-to-arrival delay of 2, `Q` = 0.5 f, a steady
flow of 0.1 f that holds the gap at the freight; the size is the peak-to-peak range of the gap in the last
half of 6 000 ticks, in units of the freight; the mean gap is its mean:

| Merchants | Size | Mean gap |
|-----------|-----:|---------:|
| identical bars, no lock (the herd) | 1.80 f | 0.61 f |
| lot lock only | 1.17 f | 0.82 f |
| spread `W` = 1 f, no lock | 0.59 f | 1.13 f |
| spread `W` = 2 f, no lock | 0.04 f | 1.27 f |
| spread `W` = 1 f and lock | 0.03 f | 1.28 f |
| spread `W` = 1 f (Weyl of the agent ID) and lock | 0.05 f | 1.27 f |
| independent error 0.4 f, no lock (INF-D2 alone) | 1.32 f | 0.70 f |
| independent error 0.4 f and lock | 0.58 f | 0.85 f |
| identical, share at dispatch 0.5 | 1.20 f | 0.67 f |
| spread 1 f and lock, share at dispatch 0.5 | 0.06 f | 1.26 f |

With a longer delay (a rumour age of 6, a decision-to-arrival delay of 4) the herd swings by 3.35 f, the lock alone
brings it to 0.48 f, and spread and lock to 0.06 f. With a lot twice as large (`Q` = 1.0 f) the
herd swings by 3.85 f, lock and a spread of 1 f leave 0.60 f and a spread of 2 f settles it at 0.05 f;
a lot cap of 0.25 f or 0.1 f gives 0.98 f or 0.41 f for the herd and 0.06 f or 0.04 f with spread and
lock. At a fixed `Q` the size falls as the merchants grow in number (0.61 f at 5 to 0.10 f at 200 with
spread, lock and an error of 0.2 f). Over eight seeds the rows with an error vary by about 0.1 f at most.

Two readings. **First,** the mean gap of the herd is 0.61 f, below the freight: the herd overshoots,
ships into a gap its own arrivals have closed, and loses on average. The spread keeps the mean gap above
the freight (1.28 f), so a merchant that ships earns what its share promised. **Second,** the
spread is the strong damper, the lock the cheap one that halves the spread needed, and the independent
error of INF is weak next to either (INF-D2), as `spec-information.md` 11.2 found. What remains is a
pack with a long total delay and a lot large against the markets (7.2.2), and many merchants of
different regions on one lane whose spreads are not coordinated; MER-A1 records both.

### 7.4 What is counted

By FR-STB-013 and N-065 a damper that acts silently is counted. The strategy cannot emit events, so it
reports its decisions in the plan (`TradePlan.Notes`, a fixed set of counters, 15.1) and the world sums
them per region into one event, `MerchantsPlanned`. R1 is `Locked`, R2 is `LotCapped`, R4 is `Stale`; the
other cases of 5.1.2 and 6 are counted beside them, so a lane that never trades shows why.

## 8. The consignee

### 8.1 Who receives a hired shipment

8.1.1. A hired shipment needs a consignee in the destination (`spec-transport.md` 8.4, N-100). The
decision of this spec, among three options:

- **the merchant's own agent there:** rejected. Agents are independent accounts, a "house" of two accounts
  would be a relation the ledger does not know, and the goods would be given away between them;
- **a partner role with a standing order:** rejected for version 1. A standing order is a contract with a
  term (9.10) and needs a record of who is bound to whom, which is state the merchant does not have;
- **the payer of a delivery contract:** adopted. The consignee is an agent that **bought** the goods
  before they moved. Its payment is in escrow from formation (9.5), so the deliverer is paid when the goods
  arrive and the consignee is not charged for goods that do not.

8.1.2. Any agent may be a consignee: it posts a **take** (9.2). The merchant strategy posts takes if its
role has `Consigns`; a consumer's strategy posts them as it likes, and the world treats them the same.

### 8.2 The take

For a role with `Consigns` and a work share above 0 (6.5), for each commodity `c` of the cargo set in
ascending order that is not the cargo of the trip it plans this tick (a merchant does not buy what it is
about to send):

```
P        = the local price of c in the report
limit    = max(1, P * (10 000 - TakeMarginBP) / 10 000)                 truncated
free     = max(0, Stock_c - Keep_c)                                       units above the household's own keep
want     = TakeUnits - free - InboundTotal_c                              what it lacks, counting what is on the way
afford   = floor( max(0, Money - ReserveMoney - M) / limit )              M: money its orders of this plan reserve
qty      = min(want, afford, MaxQuantity)
```

and a take `{Commodity: c, Limit: limit, Quantity: qty}` if `qty >= 1`. **It counts its pipeline:**
`InboundTotal_c` is the total quantity of `c` on the way to it (`Inbound`), so a consignee that has a
shipment coming does not take again for the same need (L85, TRN-A2). It reads its own market's price, its
own stock and its own money, and nothing about any other region: a consignee needs no rumour. The take
lasts one tick, like an order.

### 8.3 What the consignee does with the goods

They arrive in phase 1 (`spec-transport.md` 5.7) and sit in its account. In phase 3 the default's rule sells
the surplus above its keep, as an ask at a limit drawn from its belief, unless the consignee has made them
the cargo of its own trip. The consignee's profit is the difference between what it sells at and the
contract price, and it bears the movement of the local price in the `k` ticks from formation to sale
(MER-A2). Its take margin is the compensation: with `TakeMarginBP` 1 000 it buys at 10% under the local
price it saw when it posted the take, and a fall of more than that, before it sells, is a loss. The market
tells it so through its income, and the work share and the ladder act on it as on any agent.

### 8.4 Several consignees, one offer

The round (9.4) matches each offer with the take of the highest limit that can take it whole, ties by agent
ID, so the consignee that values the goods most gets them and the others take what is left. A take that finds
no offer expires, and its money was never moved.

## 9. The delivery contract

### 9.1 What a contract adds beyond the book

The book (`spec-clearing.md`) is one market of one region in one tick: both parties are in the
region, the goods change hands at once, and the money is a reservation that lasts the tick
(`spec-ledger.md` 6.5). A hired shipment (`spec-transport.md` 8) moves goods without their owner,
so it needs a **buyer who is committed before the goods leave**; the book cannot supply one,
because the buyer is in another region. The contract is that commitment, and it adds, beyond the
book, exactly four things:

1. **A buyer across a region boundary.** The consignee of a hired shipment bought the goods; without
   it the goods would be a gift to a stranger (8.1).
2. **A price fixed before the goods move.** The deliverer bears no price risk of the destination, the
   payer bears it from formation to resale (MER-A2).
3. **Payment on arrival, from money that is committed and not merely reserved.** The payer's money is
   in an Escrow account from formation (`spec-ledger.md` 6.6), because a reservation does not outlive a
   tick; the deliverer is paid out of it when the goods arrive, for what arrived.
4. **A defined outcome when something goes wrong**: spoilage, the removal of a party, a closed road
   (9.7).

It does **not** add a new kind of market. `spec-clearing.md` is unchanged and every guarantee of its
2.3 stands for the book; the contract round has its own admission and its own guarantee (9.5), which
is the same in kind: everything the settlement needs is checked and moved before the contract exists,
so settlement cannot fail. It adds no negotiation (the matching is a rule, 9.4) and no credit (9.10).

### 9.2 Offers and takes

A strategy gains two plan fields. `TradePlan` (`spec-agents.md` 6.3) gets:

```go
// DeliveryOffer offers goods for delivery to another region. It is indivisible.
type DeliveryOffer struct {
	Mode      ModeID
	To        RegionID    // the destination region
	Commodity CommodityID
	Quantity  Quantity    // whole units, all or nothing
	Limit     Price       // the least delivered price per unit the deliverer accepts
}

// DeliveryTake asks for goods to be delivered to the taker's region.
type DeliveryTake struct {
	Commodity CommodityID
	Quantity  Quantity    // at most; a take may be filled in parts
	Limit     Price       // the most per unit the taker pays
}

// TradePlan gains:
//	Offers []DeliveryOffer
//	Takes  []DeliveryTake
//	Notes  PlanNotes // 15.1: fixed counters a strategy fills; zero for the default
```

An offer names no consignee: the round chooses the payer. A take names no origin: it is for goods
from anywhere, and each contract it makes has the travel time of its own origin. Both last one tick, as
an order does. The default strategy appends none, so a world of default strategies has no offer and no
take, and a world with the module off refuses them all (9.3).

### 9.3 Admission

At the start of the round the world visits the offers and takes in agent-ID order and in plan order
within an agent and admits each or refuses it whole with a `reason`, in this order (constants of
`internal/logkeys`; a refusal changes nothing and is counted in `ContractRound`, 15.1, and logged at
`Debug`; it is not an error returned by `Step`):

| Of | # | `reason` | Condition |
|----|---|----------|-----------|
| both | 1 | `contracts_off` | `Modules.Merchants` is absent |
| offer | 2 | `too_many_requests` | with the plan's shipment requests, more than `MaxDispatchPerTick` (`spec-transport.md` 5.4, D39) |
| offer | 3 | `unknown_mode` | the mode is not configured |
| offer | 4 | `unknown_region`, `same_region`, `outside_destination` | as the same reasons of `spec-transport.md` 5.4 |
| both | 5 | `unknown_commodity`, `not_tradable` | as `spec-clearing.md` 3.2 |
| both | 6 | `invalid_quantity`, `invalid_price` | a quantity below 1 or above `MaxQuantity`; a limit below 1 or above `MaxPrice` |
| offer | 7 | `insufficient_stock` | the quantity exceeds the available (unreserved) units of the commodity |
| take | 8 | `duplicate_take` | the agent already has an admitted take for the commodity this tick |
| take | 9 | `overflow` | limit times quantity does not fit `int64` |
| take | 10 | `insufficient_funds` | limit times quantity exceeds the agent's available cash |

The payer's cash is its balance less what its orders of the tick reserved (`spec-clearing.md` 3.3), so
a take can never spend money an order has earmarked, and the stock of an offer is the stock unreserved by
its owner's asks. A strategy that double counts is refused, as `insufficient_stock` refuses an ask.

### 9.4 The round

9.4.1. **When.** Once per tick, in phase 3, after the last agent's plan has been executed and before the
estates' asks and the counterparties' orders (`spec-agents.md` 4; the position is proposed to WLD, 12.5).
It runs only with `Modules.Merchants` present; it is a no-op when there is no admitted offer.

9.4.2. **Lanes.** A **lane** of the round is a pair (destination region, commodity). The admitted offers are
those that name the region, the takes are those of agents whose account is in it. The round visits lanes in
ascending (region, commodity) order. In a lane:

- the offers are ordered by `Limit` ascending, equal limits by a key, then by admission position;
- the takes are ordered by `Limit` descending, equal limits by a key, then by admission position.

The **key** is one `uint64` drawn from the world's source for each admitted offer and then for each
admitted take, in admission order, the way `spec-clearing.md` 4.2 draws the tie-break key of an order: no
agent is favoured by its ID, the comparison is a total order (architecture.md 7.6), and the number of
draws is the number of admitted offers and takes, a function of state. A world with no admitted offer or
take takes none.

9.4.3. **Matching.** For each offer in order, find the first take in order whose `Limit` is at least the
offer's `Limit` and whose remaining quantity is at least the offer's quantity (an offer is indivisible, a take
is divisible); if there is none, the offer is **unmatched** (`no_taker`). If there is one, the pair is a
**tentative match** at the price

```
Price = halfEven( (offer.Limit + take.Limit) / 2 )
```

the midpoint of the two limits with the rounding of `spec-clearing.md` 5.2, which lies between them. The
midpoint splits the surplus between the two parties, as the book's does. The pair then goes through
formation (9.5). If formation succeeds the take's remaining quantity falls by the offer's; if it fails the
offer is dropped with the reason and the take is unchanged, and the round goes on to the next offer.

9.4.4. **Why not the book.** The offers are indivisible because a lot is a whole number of carrying units of a
shipment, and splitting one between two consignees would double the haul units it redeems
(`spec-transport.md` 8.4, lumpiness). So the round is not a double auction and does not call `Clearing`: it is
first-fit by price with a midpoint, which is simple, has no cycle and gives the trade to the deliverer with the
lowest limit and the taker with the highest, the rule the book follows. Whether a second extension point
for it is needed is open question 8.

### 9.5 Formation and the escrow

For a tentative match of the offer of a deliverer `D` with the take of a payer `P`, quantity `q`, price `p`,
destination `r`, the world checks and acts in this order. The first failure ends the attempt with its reason
and changes nothing.

1. **`too_many_contracts`:** the contracts in flight are `MaxOpen` (13).
2. **`payer_funds`:** `P`'s available cash is below `p * q` (its earlier contracts of the round may have spent
   what its take was sized on).
3. **Dispatch.** The world executes, as `D`, the shipment request `{Mode, To: r, Consignee: P, Lines: [(c, q)]}`
   with every check and every effect of `spec-transport.md` 5.4 and 5.5.1, and the event
   `ShipmentDispatched` of that spec. A refusal of transport (`insufficient_haul`, `capacity_exceeded`,
   `traveller_away`, `unprovisioned`, ...) is the reason of the failure and emits `ShipmentRejected` as it
   always does. A self-carried offer sends `D` itself on the trip, with the lot consigned to `P` and the rest of its
   holdings travelling with it.
4. **Escrow.** The world takes the lowest-ID open Escrow account of region `r` that no open contract uses,
   or opens one (kind Escrow, region `r`), and transfers `p * q` from `P` to it under `escrow_in`. It cannot
   fail: `P`'s cash was checked in step 2 and nothing since has moved it.
5. **The record** of 16, with the next contract ID, and the event `ContractFormed` (15.1).

Because dispatch precedes the escrow and the escrow cannot fail after it, a contract exists only if the goods
are on the way and the payment is committed: there is no state in which one is and the other is not. The
shipment arrives at `Dispatched + k` and `k` is fixed at dispatch (`spec-transport.md` 4.3), so **the
deadline of a contract is its arrival tick**, known at formation; a closure after dispatch does not move it
(`spec-transport.md` 6.3). There is no late delivery.

### 9.6 Ledger and accounts

9.6.1. **Reasons.** The registry (`spec-ledger.md` 7.2) already has `escrow_in`, `escrow_out` and
`escrow_refund` (money, internal), owned by `spec-carriage.md`. This spec takes over the owner and fixes the
accounts: `escrow_in` from an Agent account to an Escrow account; `escrow_out` from an Escrow account to the
deliverer's Agent account; `escrow_refund` from an Escrow account to an Agent account or, when the payer no
longer exists, to the Estate account of the payer's region. **The contract adds no faucet, no sink and no
reason:** it moves money between non-boundary accounts only, so every identity of `spec-ledger.md` 9.1 holds
and `BoundaryFlow` shows nothing of it (N-107: applied, not applicable).

9.6.2. **Accounts.** An Escrow account holds one open contract's money and nothing else. As Transit accounts are
(`spec-transport.md` 7.1) they are reused and never closed: a contract takes the lowest-ID open Escrow account of
the payer's region that no open contract references, or opens one, and releases it when it ends; the free list is
derived from the ledger and the contract list and is not saved. The reason is the same: IDs are never reused and
a closed account's slot stays, so closing one account per contract would add a slot per contract. The region of
an Escrow account is the region it was opened in, the destination `r`, so a contract's flows are attributed
to the destination (`spec-ledger.md` 3.5).

9.6.3. **Money in escrow is out of circulation** until the arrival: it is in the contract sector of
`spec-ledger.md` 9.3, counted in the total of money and in the money supply by sector (`spec-money-supply.md`
13.4), and in no agent's balance. The amount at any tick is at most `MaxOpen * MaxMoney`, and in practice the
value of the goods in flight. It is a drain on the payers' liquidity of the same size, which the take sizing
bounds (8.2) and MER-A5 records.

### 9.7 Settlement, failure and default

In phase 1, after all arrivals of the tick have been processed by transport (`spec-transport.md` 5.7), the world
settles the contracts whose shipment arrived, in contract ID order. For a contract with price `p` and quantity
`q`, with `reached` the units of the line that arrived and were delivered to the payer or, if the payer had been
removed, diverted to the estate of the destination (`spec-transport.md` 5.8), and `spoiled = q - reached`
(`ShipmentArrived`):

| Case | To the deliverer (`escrow_out`) | Back (`escrow_refund`) |
|------|--------------------------------|------------------------|
| everything arrived | `p * q` | 0 |
| `spoiled` units spoiled in transit | `p * reached` | `p * spoiled` to the payer |
| the payer was removed before arrival (the cargo was diverted) | `p * reached` | `p * spoiled` to the estate of region `r` |
| the deliverer was removed before arrival | 0: the contract ended at the removal | `p * q` to the payer (9.8) |

Then the Escrow account is empty and is released, and the event `ContractSettled` is emitted with the amounts
and the cause. Three decisions in the table:

- **The deliverer bears the spoilage.** It chose the lot and the lane, and the life-left rule of 5.3 exists to
  keep it from shipping what cannot arrive. The payer pays for what it receives.
- **A diverted cargo counts as reached.** It reached the destination; the deliverer did what it promised. A
  payer that was removed in transit is paid for out of its escrow, which would otherwise have gone to the estate
  with the rest of its money. The estate receives the goods in place of the money.
- **A failure to dispatch cannot happen after formation** (9.5), and a late arrival does not exist. The only
  defaults are the two removals. **There is no penalty and no bond:** a default costs the party that defaults
  what it loses by the removal itself, which is everything, and a surviving party loses nothing it did not
  choose to risk.

### 9.8 Relocation and removal

9.8.1. **The refusal `open_contract`.** With the module on, this spec registers the `region_bound` refusal
`open_contract` that `spec-regions.md` 6.5.2 foresaw for "CAR, MER", open for an agent that is the payer or the
deliverer of an open contract. The public relocation (MIG, INT) returns `ErrRegionBound` naming it, and changes
nothing. A payer or a self-carried deliverer is already refused by `open_shipment` (`spec-transport.md` 5.5.3),
since both are parties to the shipment of the contract; the refusal is registered all the same, because the contract
is the obligation and a later change of the transport rule must not free a party to it. This answers the CAR/MER part
of N-090.

9.8.2. **Removal.** `spec-agents.md` 3.6 gains one step, before the money of the removed agent moves to the estate:
for each open contract of the agent as **deliverer**, the escrow is returned in full to the payer (`escrow_refund`)
and the contract ends; the shipment is not touched (it arrives as `spec-transport.md` 5.9 says and the payer receives
the goods and its money back, the windfall of FR-CAR-002, whose price is borne by the deliverer's estate). For each open
contract of the agent as **payer**, nothing happens: the escrow is a separate account, and the contract is settled
on arrival with the estate of the payer's region standing in for it (9.7).

9.8.3. A role change, a weight change or a relocation by transport of a party does not end a contract. A contract's
parties are fixed at formation, and the only agent whose account changes region during one is a self-carried deliverer,
whose arrival is the arrival of the shipment.

### 9.9 FR-CAR-001 and FR-CAR-002 restated

`spec-carriage.md` moved two requirements here, keeping their IDs (N-076, `spec-transport.md` 8.5). Their text spoke of
a "fee" from "the merchant" to "the carrier", and the shape of this contract differs from it in one way that is a decision:
**there are two parties, not three.** The carrier of the CAR text is, in a contract, the **deliverer** that carries its own
goods (a self-carried offer) or buys the carriage in the market for haul units (a hired offer), where the fee is already the
price of haul units and needs no escrow (`spec-carriage.md` 2.1). A third-party carrier would add a third account, a custody
transfer of the goods into its hands and a second default (9.10). The two requirements, in the shape of this contract:

- **FR-CAR-001.** A contract payment MUST be a transfer of money from the payer to an Escrow account at formation (`escrow_in`),
  held there until the arrival of the contract's shipment, and then released from it to the deliverer for the units that
  reached the destination (`escrow_out`) and returned for the units that did not (`escrow_refund`); no contract MUST create or
  destroy money.
- **FR-CAR-002.** If the deliverer is removed before the arrival, the escrow MUST be returned in full to the payer
  (`escrow_refund`) and the contract MUST end.

Tests cite the old IDs. They are also in section 21, with the requirements of the contract that are new.

### 9.10 Not in version 1

A forward term (a delivery at a tick in the future), a standing or repeating contract, a third-party carrier and a custody
transfer of goods into escrow, a bond or a penalty, a renegotiation or a partial acceptance of an offer, a contract with
several lines, a price indexed to a market price, a take for goods from a named region, and every form of credit: an advance
against a cargo, a bill, a loan to a merchant. Credit is `spec-credit.md` (CRD), a planned extension; the ledger has a claims
list for it (`spec-ledger.md` 10) and nothing here depends on it. A merchant that cannot afford its lot is limited to a smaller
lot (6.2.1): that is the cost of the absence, and `Unaffordable` counts it.

## 10. Backhaul

10.1. **The question** (README "Still open" 5, `spec-carriage.md` 5.1, `spec-transport.md` 20.2): the cost of a haul unit
includes the whole round trip of the recipe that makes it, so when cargo is found for the return the capacity is paid for twice.

10.2. **Decision: this spec does not price backhaul, and the double count stands as a cost that errs towards dearer carriage.**
Under N-076 capacity is a good made by the carriers of the *origin* region for the shippers of that region, so each direction's
capacity is made and paid for there, and a joint product that sold capacity in another region's market is impossible, because an
account has one region (`spec-transport.md` 20.2). A rule in the merchant could not remove the double count without seeing the
carriers' recipe, which it does not. What the merchant does is to treat the price of a haul unit as it finds it (5.4) and the
share as its margin of safety (5.5).

10.3. **What it costs, with numbers.** `models/mer/backhaul.out` re-reads the example of `spec-transport.md` 17.4: a round-trip
recipe (Duration 4, 4 haul units) gives a unit cost of 1 195; the same carriers with a recipe stated for one direction
(Duration 2, 4 haul units) give 598; a shipment of 6 haul units costs 7 170 against 3 588. At a fee share of 5 000 the smallest gap
per unit on which a full carrying unit of 400 units of a good priced at 300 is shipped is 36 (12% of the price) against 18 (6%):
**the double count doubles the smallest gap a lane supports.** With both legs loaded and a recipe that counts the round trip, a carrier bears 598 a haul unit sold (an execution serves 8) while its cost anchor, and with it the floor of its ask, stays 1 195, so the lane is half as good as the real cost allows.

10.4. **The levers, in order of weight.** (1) The pack states the recipe for one direction (`Duration` and output of one leg),
which `spec-transport.md` 8.4 names. (2) A self-carried merchant is not charged twice: it makes the return trip a trip of its
own, loaded with the goods of the far region, and the time of the first trip is amortised; the model of 7 and the rule of 6.1
already make it choose that mode when the lane is short and the load is small. (3) The imbalance between the two directions
is **visible**: `haul_use` is attributed to the origin region (`spec-ledger.md` 3.5), so `BoundaryFlow` shows the haul units
redeemed in each direction, and a pack or a measure can read the ratio. A pack in which the flows are unbalanced sees the price of
haul units in the loaded direction rise and the other fall, the freight differential that real backhaul shows.

10.5. **What stays open** is the question of whether a carrier role can price one recipe for two directions once: it needs a
carrier that is in two regions, or a contract with a third party who carries (9.10). Open question 7.

## 11. The gate of the outside

11.1. **A gate region is an ordinary region.** An outside node has no market, no agents and no group, and trades through
counterparties in the market of its gate regions (`spec-external-sector.md` 4, 9.1). A merchant in a gate region sees the
outside's quote the way it sees any price: as an ordinary price and, if unfilled, as the best unmatched ask or bid of the market
report. It buys an import with an ordinary bid at the limit its trip can bear (5.6) and the clearing prices the trade between the
two limits (`spec-external-sector.md` 6.7: a quote is a limit, not a peg). Nothing in this spec reads an outside node, names
one or branches on one.

11.2. **An outside node is never a destination.** A shipment or an offer to a node is refused (`outside_destination`,
`spec-transport.md` 5.4, 9.3), and `Routes` reports no route to it, so a merchant never evaluates one. Goods reach the outside
by being sold in a gate market to the buy side's counterparty: a merchant exports by shipping its cargo to the gate region
(an ordinary lane) and selling there in the book, to the counterparty's bid or to anyone.

11.3. **An interior merchant does hear of the gate.** `spec-external-sector.md` 9.2 and N-112 say an outside node is not a
rumour source. The gate region's market is: its published price, which includes the outside's trades, is recorded like any
market's (`spec-information.md` 4.1). A merchant elsewhere therefore reads a rumour of the gate region's price and can plan the
lane gate to interior, buying at the gate and selling inland; it learns the outside's quote only as far as it shows in the gate's
published price, with the lag and the error of any rumour. This is the answer to the "never hears of an outside price" of
N-112, whose conclusion holds for the node and not for its gate.

11.4. **The outside's capacity is shared.** A counterparty's order is one order for the whole gate market, limited by the capacity
of its line (`spec-external-sector.md` 7.1), so the merchants of a gate compete for it as buyers of any scarce good do; a merchant's
unfilled bid is evidence for its belief (`spec-beliefs.md` 5.4) and its next lot is sized from its money. The arbitrage between
two gates with different scales (EXT-A8 as a case of L84) is a lane like any other and is damped by the same rules and by the
capacity (D63); the warning `outside_arbitrage` of EXT is the configuration-time guard.

## 12. Seams with the specs around it

### 12.1 Edits this spec asks of AGT and TRN

12.1.1. `spec-agents.md` 6.3 gains the plan fields `Offers`, `Takes` and `Notes` (9.2, 15.1); 6.4.1 gains the group of 4.1; 7.3.6 reads
"a bid of a trip's inputs" beside "the next job" (3.4.2); 3.6 gains the step of 9.8.2; 3.5.2 is unchanged (the spread needs no draw).
12.1.2. `spec-transport.md` 9.1 gains `Outbound` (4.1); 5.3.2 gains the round after the last plan; 5.9 notes that a contract may refer
to a shipment; 5.7 reseeds a traveller (12.2). The list in the handoff is exact.

### 12.2 Beliefs on arrival

`spec-transport.md` 9.3 left to this spec whether a traveller reseeds its beliefs on arrival. **It does**, for every
self-carried traveller of every role, by the rule of `spec-beliefs.md` 8.3: for each commodity of its tradable set the centre is
the destination's window mean (the market price if the window is empty), the width `BeliefWidthInit`, re-clamped to the bounds of
the new region. The reason is the one MIG gives a migrant (N-038): a belief about a price in another region is a belief about
the wrong market, and a merchant arriving with the belief of the origin would offer the goods it carried at the origin's
price, below the market it is in, and lose on every sale until the update caught up. The reseed happens at the arrival, in
phase 1, right after the `Relocate`, and takes no draw. It is a world operation and not the strategy's, because a belief is
the world's state; the edit is TRN 5.7 and BEL 8.3.

### 12.3 The welfare window (N-099)

A self-carried trip of `k` ticks is a gap in income: every tick of it is a working tick, the merchant earns nothing until it
sells, and the effective window of `spec-welfare-and-wages.md` 2.3.3 is derived from recipes only, so a merchant's ratio can read
as starving in the middle of a healthy trade (TRN-A6). This spec proposes the fix WEL left to "the first traders": the effective
window of a role with a merchant block is

```
T = max( WelfareWindow, WindowCycles * max( longest Duration, Kmax ) )
```

with `Kmax` the longest travel time over the configured graph, at construction, of the self-carried modes in `Params.Modes`.
The merchant's trip is its job and `Kmax` is its duration, so the same derivation (D29) covers it. This is a change to WEL 2.3.3
and to the warning `trip_longer_than_window` of TRN 12.2, which stays for roles with no block; it is WEL's to adopt, and until it
does the warning is the only guard. The cost is the one WEL names for a long job: the merchant's ratio reacts slowly.

### 12.4 MIG, SUB, SPN, EST and EXT

12.4.1. **A merchant on a trip is away, and is not a migrant.** TRN keeps its account in the origin region while it is away
(`spec-transport.md` 5.5.2), refuses a public relocation with `open_shipment` and `open_contract`, and changes its region at
arrival with the cause `transport`. MIG must respect both refusals and must not read a trip as a move; a merchant that ends a
trip in another region and stays there has relocated by trade, and the households-by-region count follows `AgentRelocated`. MIG's own rule for a role with no recipe is MIG's: this spec asks only that a merchant not be moved
for good while its trade is where it is, and the guard of the refusals is what makes that true while it is away.

12.4.2. **A merchant has no home.** It lives where its account is. The ladder and relief see the origin region until arrival
(`spec-transport.md` 5.9.4); a merchant that dies in transit leaves its money to the pool of the region it departed from
(`spec-estates.md` 2.2); its flows are attributed to the origin until arrival. MER-A6 records the drift of merchants between
regions this allows.

12.4.3. **SPN.** A merchant that goes unfed climbs the ladder like any agent (rung 1, a change of role: like for like if viable,
else the best viable role). A merchant has no recipes, so its observed wage is the income of its working ticks (`spec-welfare-and-wages.md`
4.3), which is what the spawner's occupation signal reads. A role change or a removal ends nothing of a contract (9.8).

12.4.4. **SUB.** The seam is 6.4. SUB lets a role with no recipe have an own-use list; the merchant's `Work` is the default's, so a merchant takes the mode as any household, and while it has an own-use job or is in the mode it
stays (6.4). Otherwise the two specs share only the strategy registry.

12.4.5. **EXT.** Section 11.

### 12.5 The positions in the tick (for WLD, N-009)

| Phase | What this spec does |
|-------|---------------------|
| 1 | After transport's arrivals and before relief and starting money: the settlement of contracts whose shipment arrived, in contract ID order (9.7); at a traveller's arrival the reseed of its beliefs (12.2) |
| 3 | For each agent in ID order, as before; a merchant's `Trade` follows the construction of its view (with the draws of `spec-information.md` 5.5.2); after the last agent's plan, the round: admission, matching, formation (9.3 to 9.5) with its draws, then `MerchantsPlanned`, then the estates' asks and the counterparties' orders, then `RumoursServed` |
| 7 | The removal of a party applies 9.8.2 |

## 13. Parameters

All are fields of `Config` with a default (or "required"), a bound, a doc line and an owner
(architecture.md 9.5 to 9.7). **Every default is a placeholder** (N-015, N-027): no source for how
merchants priced, shared or insured a risk was read. The bounds are validation bounds in `bounds.go`
(`spec-config.md` 4.1).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Merchants` | the delivery contract: present means on (`spec-config.md` 3.2); it needs the transport module | nil (off) | |
| Merchants | `MaxOpen` | the most contracts open at once (9.5) | 1 000 (placeholder) | 1 to `MaxContracts` |
| Role | `Merchant` | the merchant block: nil means the role is not a merchant (3.2) | nil | |
| Merchant | `Modes` | the modes the merchant plans with | required | known modes, no duplicate, at most `MaxModes` |
| Merchant | `LotUnits` | the lot cap in carrying units | 1 (placeholder) | 1 to `MaxLotUnits` |
| Merchant | `FeeShareSpreadBP` | half-width of the spread of the fee share by agent | 2 500 (placeholder) | 0 to 5 000, at most `MerchantFeeShare`, and the share plus it at most 10 000 |
| Merchant | `MaxSourceAge` | the oldest `SourceAge` acted on | 2 (placeholder) | 0 to `MaxRumourAge` |
| Merchant | `TimeCostBP` | cost of a tick away, in basis points of the agent's own upkeep cost | 10 000 (placeholder) | 0 to `MaxTimeCostBP` |
| Merchant | `ConsigneeCutBP` | the share of the destination price expected to be given up under a contract | 500 (placeholder) | 0 to 5 000 |
| Merchant | `Consigns` | the merchant posts takes | false | a bool |
| Merchant | `TakeMarginBP` | the discount to the local price at which a consignee takes | 1 000 (placeholder) | 0 to 9 000 |
| Merchant | `TakeUnits` | the units of a cargo commodity a consignee wants to hold or have on the way | required with `Consigns` | 1 to `MaxQuantity` |

`MerchantFeeShare` (Role, default 5 000, bound 0 to 10 000) is read here and owned by
`spec-carriage.md` 3 (no new row). Both of its extremes are legal: a share of 0 is clamped to 1 in 5.7.1 and
makes the merchant never ship in practice (`spec-carriage.md` 3: "0 means the merchant never hires").

`LotUnits` of 1 means one carrying unit a lot, so the lot is the load of one vehicle or of one carrying unit of a hired mode
and the freight is the lumpiness of one unit (6.2.3). `FeeShareSpreadBP` of 2 500 on a share of 5 000
gives shares from 2 500 to 7 500 and bars from 1.33 to 4 times the cost (5.7.3). `MaxSourceAge` of 2 is
a number of ticks and is tied to nothing. Each is to be calibrated by STY and CLB once a lane has
run: none of them is derived.

**New bounds for `bounds.go`** (provisional, with the reasons):

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxLotUnits` | 64 | with `MaxQuantity` and `MaxPrice` a lot's product fits `int64` through the 128-bit helper, and a lot of more than 64 carrying units is not a lot |
| `MaxContracts` | 1 000 000 | the contracts in flight; the Escrow accounts are at most this many |
| `MaxTimeCostBP` | 100 000 | ten times the upkeep for each tick away, far beyond any use |

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the multiplier of the spread
(5.7.1, shared with `spec-population.md` 3.8.3), the cautious reading `min(Price, Mean)` (5.2), one trip and at most one request or offer per plan
(7.1), the lane lock (7.1), the matching order and the midpoint with half-to-even rounding (9.4), the key
draw (9.4.2), the order of the checks of 9.3 and 9.5, the single line of a contract, the settlement table
(9.7), the reseed on arrival (12.2), the reason constants of 9.3, 9.5 and the decisions of 5.1.2 in
`internal/logkeys`.

**Not parameters:** the fee share of an agent (derived, 5.7), a lane's score, the trip, the lot, the ceilings
(derived each tick), the free list of Escrow accounts (derived), the contracts (state, 16).

The placeholders, so that none is mistaken for a result: every default and bound above, the thresholds of
the warnings of 14.2, and every number of the worked examples (19) and of the models (18).

## 14. Validation

Each rule has a sentinel or a warning name and a failing-case test (testing.md 3.9). Errors name the path,
for example `role.broker.merchant.modes`. The framework of `spec-config.md` 7 runs them.

### 14.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field of 13 outside its bound; `FeeShareSpreadBP` above `MerchantFeeShare` or with `MerchantFeeShare` plus it above 10 000 |
| `ErrUnknownMode` (new) | a mode ID in `Modes` that is not defined |
| `ErrDuplicateID` | a mode twice in `Modes` |
| `ErrMerchantShape` (new) | a merchant block on a role with a non-empty `Recipes`; `Strategy` `"merchant"` with no block; a block with `Informed` false; a trade input (3.2.2) that is not in `Trades`; an empty cargo set; a self-carried mode in `Modes` with a `Crew` above 1; `Consigns` with no `TakeUnits` |
| `ErrMerchantWeight` (new) | a merchant role with a group of scale above 1, or whose households exceed one per agent (an agent of weight above 1) |
| `ErrModuleDependency` | a merchant block while `Modules.Transport` or `Modules.Information` is absent; `Modules.Merchants` while `Modules.Transport` is absent; a hired mode in `Modes`, or `Consigns`, while `Modules.Merchants` is absent |
| `ErrOverflow` | a derived check of `spec-config.md` 4.2: `MaxLotUnits` times the largest `Mass` or `Bulk` times `MaxQuantity` does not fit, or `MaxOpen` times `MaxMoney` does not fit |

No new sentinel is needed for the contract round at run time: a refusal there is a counted reason, not an
error (9.3).

### 14.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at construction, at `Warn`,
with the attributes `rule` and `path`; the volume is bounded by the configuration.

| Warning | Condition |
|---------|-----------|
| `merchant_spread_narrow` | the width of the bars of a role, `10 000 * 10 000 / (MerchantFeeShare - FeeShareSpreadBP) - 10 000 * 10 000 / (MerchantFeeShare + FeeShareSpreadBP)`, in basis points of the cost, is below 10 000 (one cost): the merchants act together (L93). A placeholder threshold from the model of 7.3 |
| `merchant_fee_share_high` | `MerchantFeeShare + FeeShareSpreadBP` above 9 000: the top merchants ship on a bare margin of cost |
| `merchant_fee_share_low` | `MerchantFeeShare - FeeShareSpreadBP` below 500: the lowest bar is above twenty times the cost, so some merchants hardly ever ship |
| `merchant_no_consignee` | a merchant role lists a hired mode and no role has `Consigns` |
| `lane_without_consignee` | carries `pairs`: the ordered pairs of regions with a route in a hired mode of a merchant role, where the destination has no group of a role that `Consigns` |
| `cargo_never_delivered` | carries `pairs`: the (commodity, mode) pairs of a role with a cargo commodity of shelf life `S` above 0 for which every route has a travel time above `S - 2` (5.3.1): no lot can ever arrive fresh |
| `consigns_without_cargo` | a role with `Consigns` whose cargo set has no commodity that a merchant role of another region could carry to it: it would never be sent anything |
| `merchants_one_agent` | a merchant role with one agent in the whole world: no competition for the lane and no spread |

## 15. Observability

### 15.1 Events and plan notes

15.1.1. **Events.** Four, all plain data with no map, pointer or wall-clock time, constructed only when an
observer was supplied (observability-and-logging.md 4.1, 4.5). Two are per contract, like `ShipmentDispatched`
and `OrderPlaced`; two are aggregates, bounded by the regions, like `UpkeepShortfall`.

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `ContractFormed` (new) | when a contract is formed, after its shipment's `ShipmentDispatched` | `Contract`, `Payer`, `Deliverer`, `Region` (the deliverer's, the origin), `To`, `Commodity`, `Quantity`, `Price`, `Escrowed`, `Shipment`, `Mode`, `Arrives` |
| `ContractSettled` (new) | when a contract ends, at its arrival or at the removal of its deliverer | `Contract`, `Payer`, `Deliverer`, `To`, `Commodity`, `Quantity`, `Reached`, `Spoiled`, `Paid`, `Refunded`, `Cause` (`delivered`, `short`, `payer_removed`, `deliverer_removed`) |
| `ContractRound` (new) | once per destination region and tick in which an offer or a take was submitted for it, in ascending `RegionID`, after the last contract of the round | `Region` (the destination), `Offers` and `Takes` (admitted), `OffersRefused`, `TakesRefused`, `Matched`, `Formed`, `NoTaker`, `PayerFunds`, `TooMany`, `DispatchRefused`, `Units`, `Value` (of the contracts formed) |
| `MerchantsPlanned` (new) | once per region and tick in which at least one agent with a merchant block was given a view there, in ascending `RegionID`, after `ContractRound` and before `RumoursServed` | `Region`, `Merchants`, and the sums of the plan notes of 15.1.2 |

The order within a step: phase 1, after the `ShipmentArrived` events, the `ContractSettled` events in contract
ID order; phase 3, during the round the `ShipmentDispatched` and `ContractFormed` events of each contract in
formation order, then `ContractRound` and `MerchantsPlanned`. An event of a module that is off is not emitted.
`ContractSettled` for a deliverer's removal is emitted with the removal events of phase 7.

15.1.2. **Plan notes.** A strategy cannot emit events, and the merchant's dampers act silently, so FR-STB-013
and N-065 ask that they be counted. `TradePlan.Notes` is a fixed struct of counters, filled by the strategy and
zero for the default, that the world sums per region into `MerchantsPlanned`:

```go
// PlanNotes is what a strategy reports of its decisions, as counts. A strategy that
// has nothing to report leaves it zero. The names are those of the merchant (5.1.2, 6).
type PlanNotes struct {
	Considered, NoRumour, Stale, Locked, Perishable, NoMargin, Unaffordable, BelowShare int64 // candidates, by result
	Planned, Dispatched, Offered, Taken                                                  int64 // trips planned, requests, offers, takes
	Rested, Waiting, LotCapped                                                           int64 // a rest, a stay (goods inbound, a job, the floor), a lot cut by the cap
}
```

The notes influence nothing: the world reads them only to build the event, and a run with and without an observer
is identical (observability-and-logging.md 1.4). They are not state.

### 15.2 Logs

`Debug`, guarded by `Enabled`, constant messages: "contract formed" (`contract`, `agent` the deliverer, `region`,
`dest`, `commodity`, `quantity`, `price`), "contract settled" (`contract`, `reason`), "offer refused" and "take refused"
(`agent`, `reason`), "trip planned" (`agent`, `dest`, `commodity`, `mode`, `quantity`). No `Info` in the tick and no `Warn`
per contract or per agent: a contract and a refusal are per entity (observability-and-logging.md 3.2.2). The warnings
of 14.2 are reported once at construction, and the summary line gains `merchants` among its `modules` when the module is on.

**New log keys** for observability-and-logging.md 3.3.2: `contract` (the contract ID), `offers` and `takes` (counts of
admitted offers and takes), `merchants` (a count of agents with a merchant block). `region`, `dest`, `agent`,
`commodity`, `quantity`, `price`, `mode` and `reason` exist.

### 15.3 Metrics

All `Int64`, each derivable from the events (observability-and-logging.md 5.3):

| Name | Instrument | Unit | Attributes | Fed by |
|------|------------|------|------------|--------|
| `economysim.contract.formed` | Counter | `{contract}` | `economysim.commodity` | `ContractFormed` |
| `economysim.contract.value` | Counter | `{minor_unit}` | `economysim.commodity` | `ContractFormed` (`Escrowed`) |
| `economysim.contract.escrow` | UpDownCounter | `{minor_unit}` | none | `ContractFormed` (+ `Escrowed`), `ContractSettled` (- `Paid` and `Refunded`) |
| `economysim.merchant.decisions` | Counter | `{lane}` | `economysim.decision` | `MerchantsPlanned` (the case counts) |

`economysim.decision` is a new attribute with the constant set `no_rumour`, `stale`, `locked`, `perishable`, `no_margin`,
`unaffordable`, `below_share`, `planned`, `rested`, `waiting`, `lot_capped`; it is bounded by constants (5.4.1 there lists the
allowed attributes; this one is added). The series added for the reference chain: one per commodity that is ever contracted for
the first two (at most 30) and eleven for the last.

### 15.4 Snapshot

The contracts in flight (the record of 16 with the time left) and the money in escrow in total, deep copies bounded by `MaxOpen`
(architecture.md 12.4). Names are OBS's.

### 15.5 Cost with no observer

The margin arithmetic, the matching and the settlement allocate nothing beyond scratch held by the world
(NFR-MER-001); the events and the notes are not built without an observer (NFR-MER-004). Observation never changes a result.

## 16. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, plain data in a fixed order with no maps:

- the next contract ID;
- the contracts in flight, in ID order: ID, payer, deliverer, shipment ID, commodity, quantity, price, the Escrow account,
  the tick of formation, the destination region (the payer's region and the origin are derived from the accounts and the
  shipment).

**Not saved:** the merchant strategy's state (it has none), the fee share and the spread (a function of the agent ID), the
trip, the lot and the ceilings (derived each tick), `Outbound` and `Inbound` (derived from the shipment list), the free list of
Escrow accounts (derived from the ledger and the contract list), the plan notes and the counters. The Escrow accounts and the
money in them are the ledger's state (`spec-ledger.md` 11), and the shipments are TRN's (`spec-transport.md` 14). A world
restored between ticks settles the same contracts at the same arrivals and takes the same key draws (FR-MER-061). The configuration
of the module and of the block is in the hash, and so are the strategy's name and the parameters it reads.

## 17. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9 (FR-STB-003). Fields as in its section 3; the
register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn` numbers, and the entries here are `MER-An`,
`MER-Dn` and `MER-Sn`. **The merchant closes loops of its own**, by its response to a gap and by the consignee's
take, and the contract holds money out of circulation: each has an entry. Nothing here has been run except where a
status says `model`, and no default is claimed stable (N-027).

**Merges into existing loops** (9.9 of the checklist): MER-A1 into L84 and L85 as the shipper's response, and into
L93 as its damper; it records that L94 and L95 are damped by the merchant's rules. MER-A3 into L14 and L16. MER-A4
into L16 and L21 (TRN-A3). MER-A2, A5, A6 and A7 are new.

**MER-A1. The merchant's response closes the price-gap loop.** *Kind:* loop (the response side of L84, L85, L93).
*Loop:* the rumoured gap (+) a merchant's trip passes the fee-share test (+) the lot it buys: the price at the
origin rises at once (-) the gap; and the lot arrives `k + 1` ticks after the decision (-) the destination price
(+) the gap. Each path has one negative link: B. *Delay:* `n = Lag + k + 1`: the age of the rumour, the tick between
the purchase and the dispatch, and the journey (7.2.1). *Owner:* MER 5, 6, 7; the mechanism is TRN's, the rumour INF's.
*Damper:* MER-D1 (the lane lock), MER-D2 (the lot cap, D39), MER-D3 (the fixed spread of the fee share), MER-D4 (the
bound on `SourceAge` and the cautious reading), MER-D8 (the fee share as a margin of safety, extending D40); D38 (the merchant counts
its own pipeline); D60 (the error of INF, weak). *Undamped:* a total delay of many ticks with a lot large against the two
markets; merchants of several regions on one lane whose spreads are not coordinated. *Condition:* the slope of the aggregate
response `Q / W` is below `2 sin(pi / (2 (2 n + 1)))` without the lock, and the spread needed is about a half to a quarter of that
with it (7.2.2). *Test:* MER-S1. *Status:* model (`models/mer/lane.py`): a linearised gap with no inventory, belief, sizing or market,
a share of the effect at dispatch that is not derived for any pack, a single rule and identical lots. *Neighbouring delays (9.6):*
`n` against a perishable's `ShelfLife` (the rule of 5.3 sets a lane aside when `k` exceeds the life left, but the rumour's age
is not in it), the belief adjustment (a few ticks), the welfare window (12.3) and the recipe `Duration` of the carriers (MER-A4).

**MER-A2. The consignee's take loop.** *Kind:* loop (new). *Loop:* the local price at the destination (+) the consignee's
take limit (+) the contracts it makes, in units (+, after `k`) the goods it holds and sells (-) the local price. B with the
delay of the shipment, the sale and the belief adjustment. The take is sized against the consignee's stock and its `Inbound`
(8.2), which removes the pipeline from the signal. The consignee bears the price risk from formation to sale. *Delay:* `k`
plus the sale. *Owner:* MER 8, 9. *Damper:* MER-D5 (sizing against stock and `Inbound`), MER-D6 (cash only, a cap on open
contracts), D01 (the hard budget). *Undamped:* consignees whose `TakeUnits` is large against the market and whose take margins are
equal. *Condition:* `TakeUnits` times the number of consignees is small against the units the destination market clears in a
tick. *Test:* MER-S2. *Status:* hand.

**MER-A3. The merchant at its target income.** *Kind:* loop (a case of L14, L16). A merchant whose ratio reaches the top of its
band plans no trip and posts no take (6.5): the supply of arbitrage falls as its margin rises, as a carrier's does (L16). The
demand it answers is the gap, which closes as lots arrive: the elasticity of the gap to the flow is that of the two markets,
below 1 for a staple. *Delay:* the effective window, which 12.3 proposes to extend by the longest trip. *Owner:* MER 6.5; WEL 5.
*Damper:* D09, D13, D26 to D29 as for any agent; the lock. *Undamped:* as L14. *Condition:* as L14. *Test:* S04 with a merchant
role. *Status:* hand.

**MER-A4. Derived demand for haul units has a choke price.** *Kind:* loop (a case of L16 and of the cobweb L21, TRN-A3). The merchants'
demand for haul units is derived: it is `n * s` units a lot whatever the price, up to the ceiling `HaulCeiling` (5.6), and zero above it.
The price of haul units (+) the carriers' starts after the recipe `Duration` (+) later supply (-) the price is the cobweb of TRN-A3; the
ceiling bounds the upswing at a price the merchants can bear and ends the lane above it. *Owner:* MER 5.6; TRN 8.4. *Damper:* the
ceiling itself, D02, D03, D14 and the short shelf life of the haul commodity. *Undamped:* a ceiling above the carriers' cost band
ceiling, which then does not bind. *Condition:* as L21, with the derived demand elastic at the ceiling and inelastic below it.
*Test:* S32 (TRN) with merchants. *Status:* hand.

**MER-A5. The escrow holds money out of circulation.** *Kind:* hazard (a neighbour of the "wealth stuck in an account" of
`spec-ledger.md` 14). Money in an Escrow account is in no balance until the arrival: at most the value of the goods in flight
(`MaxOpen` times the price), and it is a drain on the payers' liquidity that cannot starve them, because a take is sized on the
money above the reserve (8.2). A region with many importers shows a lower money supply in circulation by the escrowed amount while
the shipments are on the way. *Owner:* MER 9.6. *Damper:* MER-D6. *Undamped:* by design. *Condition:* the escrowed amount is small against the
money supply of the destination. *Test:* MER-S2 (the identity of `spec-ledger.md` 9.1 and the audit hold with money in escrow). *Status:* hand,
and the conservation identity run in `models/mer/contract.py`.

**MER-A6. Merchants drift between regions.** *Kind:* hazard (an amplifier of L66, L67 and the pool loops L39, L63). A self-carried merchant changes
region at the end of each trip and has no home (12.4.2). The merchants of a lane accumulate in the region where the next lane is best, the
origin region loses its buyers (a region of few merchants has a larger gap, a lane with more) and the destination gains sellers, and the counts of
households by region, the pools (relief and entrants) and the failure ladder follow. *Owner:* MER 12.4; TRN 5.7. *Damper:* none by design; the
event `AgentRelocated` with the cause `transport` counts every move, and the lock bounds the pace. *Undamped:* all of it. *Condition:* the merchants
are few against the households of a region. *Test:* MER-S4. *Status:* hand.

**MER-A7. A stranded preparation.** *Kind:* hazard (a case of L84). A merchant buys a lot and haul units in one tick and sends them in the next; if the
rumour, the price or the haul price has moved by then, the trip fails the test and the merchant holds inputs it paid for (6.3.3). The default sells the
cargo as surplus and the haul units spoil. The loss is bounded by one lot per merchant and the lock, and it is a cost of the two-tick preparation, not a
loop. *Owner:* MER 6.3. *Damper:* MER-D8 (the margin of safety). *Condition:* the share is large enough that a move of the inputs of the
order of the price's own noise does not fail the test. *Test:* MER-S1. *Status:* hand.

**Dampers this spec adds** (for the catalogue of `spec-stability.md` 5; each meets 5.1):

| ID | Damper | Home | Status |
|----|--------|------|--------|
| MER-D1 | The lane lock: no new lot on a lane while an own cargo of it is on the way. Cost: one lot per lane per `k + 1` ticks | MER 7.1 R1 | specified (a rule constant; counted by `Locked`; model) |
| MER-D2 | The lot cap and the single trip: one request or offer a plan, at most `LotUnits` carrying units a lot (D39 for a merchant). Cost: a large shipment is several lots | MER 6.2, 7.1 R2 (`LotUnits`) | specified (counted by `LotCapped`; model) |
| MER-D3 | The fixed spread: the fee share, and so the bar, differs by agent, by a function of the agent ID, and is the same every tick. Cost: the merchants with a high bar never ship a thin gap | MER 5.7 (`FeeShareSpreadBP`) | specified (a visible parameter; model; the warning `merchant_spread_narrow`) |
| MER-D4 | The bound on `SourceAge` and the cautious reading `min(Price, Mean)`. Cost: a thin source market is never served | MER 5.2 (`MaxSourceAge`) | specified (counted by `Stale`) |
| MER-D5 | The consignee counts its pipeline: a take is sized against its free stock and its `Inbound`. Cost: a consignee does not take what it already expects | MER 8.2 | specified |
| MER-D6 | Cash only and a cap: a take is sized on the money above the reserve, the payer's cash is checked again at formation, and at most `MaxOpen` contracts are open. Cost: some takes are not filled | MER 8.2, 9.5 (`MaxOpen`) | specified (counted by `PayerFunds` and `TooMany` of `ContractRound`) |
| MER-D7 | The reseed on arrival: a traveller's beliefs are reseeded from the destination's market, so it does not offer its cargo at the origin's price. Cost: what the belief had learned is lost | MER 12.2 | specified (a rule constant) |
| MER-D8 | The fee share as a margin of safety: carriage is paid for only if it costs at most a share of the expected gross margin (extends D40, the freight as a floor on the gap). Cost: a persistent gap of the freight over the share | MER 5.5 (`MerchantFeeShare`) | specified (counted by `BelowShare`) |

**Scenarios proposed** (for section 8 of the register; REF builds them, with the packs they need):

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| MER-S1 | Two regions with a route, a merchant role of 20 and then 50 agents, the lane lock and the spread on and off, `FeeShareSpreadBP` at 0, 1 000 and 2 500, with rumours of a lag of 1, 2 and 4 and several travel times (it is S46 and S30 with the merchant) | the peak-to-peak size of the gap in the last half of the run does not exceed the bound of 7.3 for the settings that satisfy 7.2; with identical merchants and no lock it swings by more than the freight; the cap bounds the swing; the units shipped per tick never exceed the merchants over `k + 1` | MER-A1, MER-A7, L84, L93 |
| MER-S2 | A lane with consignees (a role with `Consigns`) and brokers, a spoiling commodity and a closed edge; the removal of a deliverer and of a payer scripted in transit | every contract is settled in the table of 9.7; money is conserved with money in escrow and the audit holds; the escrowed total returns to 0 when nothing is in flight; the refusal reasons are counted | MER-A2, MER-A5, FR-CAR-001, FR-CAR-002 |
| MER-S3 | A thin source market whose price age grows and a dear remote region (S47 with the merchant) | a merchant with `MaxSourceAge` 2 ships nothing on the stale rumour; a merchant with the bound at its maximum ships and its cargo arrives without moving the price | L94, MER-D4 |
| MER-S4 | Three regions in a chain, a hired mode and a self-carried one, the price of haul units raised stepwise | the mode chosen switches at the haul price the rule of 6.1 gives; merchants relocate along the lanes and `AgentRelocated` with the cause `transport` counts them | MER-A6, MER-A4 |
| MER-S5 | A gate region with an outside node (S52 and S55 with the merchant) | merchants buy imports at the gate and sell inland; no shipment or offer names the node; the outside's capacity is shared | EXT-A8, L84 |
| MER-S6 | The reference world with merchants and consignees, 200 ticks, a checkpoint at 80 and a resume | the resumed run is bit-identical, the key draws equal the admitted offers and takes, and the contracts in flight at the checkpoint settle at the same ticks | determinism, FR-MER-061, FR-MER-062 |

## 18. What was run

Four scripts in `doc/specs/models/mer/`, run in Docker (`python:3.12-slim`, the standard library only,
`PYTHONDONTWRITEBYTECODE=1`), as the README of that folder says. They are **prototypes of the rules, not of the
library**: floating point inside `lane.py` only, a linearised gap or integer arithmetic, no inventory, belief, sizing
or market.

- **`lane.py` and `lane.out`** (7.2, 7.3, MER-A1): it extends `models/inf/shippers.py`. `N` merchants ship a lot when
  the rumoured gap exceeds their bar; the lot arrives after the delay; the gap is pulled back to twice the freight.
  Variants: identical bars, the lane lock, a one-sided spread of the bars (even and by the Weyl value of the agent
  ID), an independent error, a share of the effect at dispatch, longer delays, a larger lot, a lot cap, the number of
  merchants, eight seeds, the smallest spread that settles the gap against the formula, and how even the Weyl spread
  of consecutive and spaced IDs is. The row of identical bars with no lock reproduces `shippers.py` (1.80 f).
- **`contract.py` and `contract.out`** (9, MER-A5): the matching of offers and takes on a lane, the formation and the
  settlement in four cases (everything arrived, units spoiled, the payer removed, the deliverer removed), with the
  check after each step that the money of all accounts is constant and the escrow account is empty. Integer arithmetic.
- **`worked.py` and `worked.out`** (19): the arithmetic of the two worked examples, the fee shares of four agents, the
  Weyl values and the cost comparison.
- **`backhaul.py` and `backhaul.out`** (10): the unit cost and the smallest gap in three readings of the carrier's recipe.

**What they do not show:** that the Go code is right (a test must); that any default is stable; the rule of any
real merchant; the interaction of the lots with the beliefs of the markets or with the price of haul units; seasons;
more than one lane; the cost of the draws. In `lane.py` the key tie-break of 9.4.2 is not modelled and the matching
of `contract.py` breaks ties by agent ID, which does not change any number quoted.

## 19. Worked examples

Two settings, each different from the other, chosen to show the hired mode with a contract and the self-carried mode
with a gate. Both are **examples** with placeholder numbers; names are invented and no claim is made about either
setting. The arithmetic is in `models/mer/worked.py` and `contract.py`; the outputs are `worked.out` and `contract.out`.

### 19.1 Example A: a river valley (a pre-industrial setting; hired mode and a contract)

Two regions, `upper` and `lower`, with the edges `upper` to `lower` of 60 and `lower` to `upper` of 70. One hired mode,
`barge`: speed 30, a carrying unit of (1 200 mass, 900 bulk), haul commodity `haul`, span 10. `fuel` has mass 2 and bulk 6
and never spoils. The information module has `info_speed` 40. The merchants are four `broker`s in `upper` (agents 32 to 35);
the consignees are the `broker`s in `lower` with `consigns`.

```toml
[modules.transport.modes.barge]
speed = 30
unit_mass = 1200
unit_bulk = 900
haul = "haul"
haul_span = 10

[modules.information]
info_speed = 40
rumour_error_bp = 500
rumour_max_age = 60

[modules.merchants]
max_open = 1000

[role.broker]
strategy = "merchant"
informed = true
trades = [ "fuel", "haul" ]
merchant_fee_share = 5000

[role.broker.merchant]
modes = [ "barge" ]
lot_units = 1
fee_share_spread_bp = 2500
max_source_age = 2
time_cost_bp = 10000
consignee_cut_bp = 500
consigns = true
take_margin_bp = 1000
take_units = 150
```

and, in Go, `economysim.NewWorld(cfg, strategy.Options()...)`.

**The lane.** `upper` to `lower`: `k = ceil(60 / 30)` = 2, `s = ceil(60 / 10)` = 6 haul units per carrying unit. The news from `lower`
to `upper` has the lag `1 + ceil(70 / 40)` = 3. A carrying unit holds `min(1 200 / 2, 900 / 6)` = 150 units of fuel (bulk
binds), so a lot of one unit is 150 units and `n` = 1. At tick 100 the rumour about `lower` is: price 310, mean 305, `SourceAge` 0,
within the bound of 2. The cautious revenue is `min(310, 305)` = 305; a hired lane is valued net of the cut of 500 basis points,
`305 * 9 500 / 10 000` = 289 (truncated). The local price of fuel is 200 and a haul unit costs 700, so the freight is
`1 * 6 * 700` = 4 200 and the gross margin is `150 * (289 - 200)` = 13 350.

**Four brokers, four fee shares.** The spread of 2 500 on a share of 5 000 gives:

| Agent | `u` | `phi` | `phi * G / 10 000` against the freight 4 200 | Haul ceiling (`G / 6 = 2 225`) | Break-even buy | Offer limit |
|------:|----:|----:|---------------------------------------------:|-------------------------------:|---------------:|------------:|
| 32 | 7 770 | 6 385 | 8 524: passes | 1 420 | 245 | 244 |
| 33 | 3 951 | 4 476 | 5 975: passes | 995 | 226 | 263 |
| 34 | 131 | 2 566 | 3 425: **fails** | 570 | 179 | 310 |
| 35 | 6 311 | 5 655 | 7 549: passes | 1 258 | 239 | 250 |

The same lane, the same rumour, and one broker of four does not ship: the bars are 1.57, 2.23, 3.90 and 1.77 times the freight
(`10 000 / phi`). The haul ceiling is `FR-CAR-004`: agent 32 bids at most 1 420 for a haul unit although the market asks 700. The
break-even buy price 245 is the dearest price at which agent 32's lot still passes; its offer limit 244 is the least delivered
price at which it ships (`200 + ceil(4 200 * 10 000 / (6 385 * 150))` = 200 + 44).

**The contract.** At tick 100 agent 32 prepares: it buys 150 fuel (its bid is the draw of its belief, lowered to 245 if above) and 6 haul units
(lowered to 1 420), tier 2 of its budget, both arriving in phase 4. At tick 101 it is ready and offers 150 fuel to `lower` by `barge` at
244. In `lower` the importers post takes with a limit of `300 * (10 000 - 1 000) / 10 000` = 270 (their local price is 300). The round of
tick 101, on the lane (`lower`, fuel), with three offers and three takes (`contract.out`):

| Offers | Limit | Units | | Takes | Limit | Up to |
|-------:|------:|------:|-|------:|------:|------:|
| agent 32 | 244 | 150 | | agent 61 | 270 | 200 |
| agent 35 | 250 | 100 | | agent 62 | 255 | 150 |
| agent 33 | 263 | 150 | | agent 63 | 240 | 150 |

Offers in order of limit: 32, 35, 33. Offer 32 takes 150 of take 61 at `(244 + 270) / 2` = 257; take 61 has 50 left. Offer 35 (100 units,
indivisible) does not fit the 50 left and goes to take 62 at `(250 + 255) / 2` = 252.5, half to even 252. Offer 33 (limit 263, 150 units)
finds no take: 61 has 50 left and 62 is below its limit. Agent 34 made no offer. The two contracts are formed, each with a dispatch by the
shipment mechanism and an escrow. For agent 32: 6 haul units go to the boundary under `haul_use` (region `upper`), the 150 fuel to a Transit
account of `upper` under `transit_out`, and `150 * 257` = 38 550 moves from agent 61 to an Escrow account of `lower` under `escrow_in`. The shipment
arrives at `101 + 2` = 103, and the lane (`lower`, fuel) is locked for agent 32 in ticks 101 and 102 (`Outbound`).

**Settlement at tick 103**, phase 1, after the arrival of the shipment (`worked.out`, `contract.out`):

| Case | `escrow_out` to agent 32 | `escrow_refund` |
|------|-------------------------:|----------------:|
| all 150 reached | 38 550 | 0 |
| 128 reached, 22 spoiled (a spoiling cargo) | `257 * 128` = 32 896 | `257 * 22` = 5 654 to agent 61 |
| none reached | 0 | 38 550 to agent 61 |
| agent 32 removed in transit | 0 | 38 550 to agent 61 at the removal |
| agent 61 removed in transit (the cargo diverted to the estate) | 38 550 | 0 (the estate of `lower` stands in for 61) |

In each case the sum of the two is 38 550 and the total money of all accounts is constant. For the case of full delivery, agent 32's account
shows `38 550 - 150 * 205 - 4 200` = 3 600, if it paid 205 on average for the fuel: less than the 13 350 expected margin, because the rumoured
305 became a price of 257 (the consignee's discount of 10% from 300, then the midpoint) and the estimate allowed for a cut of 5%; the estimate
is a reading and not a promise, and the share of 6 385 is the margin that keeps it from losing. Agent 61 sells the 150 fuel in `lower` at about 300 and earns
`45 000 - 38 550` = 6 450 if the price held.

**A perishable on the same lane.** With a shelf life of 3, a fresh lot bought at tick `t` and sent at `t + 1` has age 1 and the life left is
`3 - 1 - 1` = 1 tick; `k` = 2 is above it, so the lane is `Perishable` and set aside. With a shelf life of 4 the life left is 2 and the lane
passes. **A self-carried mode on the same lane.** A `porter` (speed 20, a unit of (60, 40), crew 1) takes `ceil(60 / 20)` = 3 ticks and carries
`min(60 / 2, 40 / 6)` = 6 units of fuel: a gross margin of `6 * (305 - 200)` = 630 against a time cost of `3 * 200 * 10 000 / 10 000` = 600
(the agent's own upkeep cost is 200 a tick). Agent 32's `phi * G / 10 000` is 402: the porter fails and the barge, which passes, is the only mode
(6.1).

### 19.2 Example B: a mining colony (a space setting; a self-carried mode and a gate)

Three regions: `hub`, which is a gate of the outside node `far` (the outside sells `ration` into `hub`, ask 55); `pit`; and the node. `hub` to `pit`
is 250 and `pit` to `hub` is 250. One self-carried mode, `skiff`: speed 100, a carrying unit of (2 000, 1 500), crew 1, a vehicle `skiff` of durability 400
(price 4 000), wear 5 a tick, feed 1 `cell` a unit a tick. `ration` has mass 1 and bulk 1 and is also the upkeep (1 a tick). The information module has
`info_speed` 100. The merchant is a `trader` of weight 1 in `hub`, agent 18: `u` = 1 246, so `phi` = `5 000 + 2 500 * (2 * 1 246 - 10 000) / 10 000` = 3 123.

**The lane.** `k = ceil(250 / 100)` = 3; the news from `pit` has the lag `1 + ceil(250 / 100)` = 4. The rumour: price 80, mean 78, `SourceAge` 0:
the cautious revenue is 78 and, being self-carried, there is no cut. The local price of `ration` at `hub` is 56 (the outside's unmatched ask of 55 is in
the market report). A carrying unit holds 1 500 rations, so the cap is far above what the trader can pay for: with 6 000 in money, the reserve left out, and 3
cells at up to 35 as the other inputs, it affords `floor((6 000 - 105) / 58)` = 101, and the lot is 100.

**The cost, in time, wear and feed.** `wear = ceil(1 * 5 * 3 * 4 000 / 400)` = 150; `feed = 1 * 1 * 3 * 30` = 90 (3 cells at 30); `time = 3 * 56 * 10 000 /
10 000` = 168 (3 ticks of the agent's upkeep cost of 56): the cost `C` = 408. The gross margin is `100 * (78 - 56)` = 2 200, the test is
`408 * 10 000 = 4 080 000 <= 3 123 * 2 200 = 6 870 600`: it passes, and the trip costs 18.5% of the gross margin. The break-even buy price is
`78 - ceil(408 * 10 000 / (3 123 * 100))` = 78 - 14 = 64: the trader's bid for rations carries a limit of at most 64. The outside's ask is 55 and clearing prices a
pair between the two limits: a bid at 60 against the ask at 55 trades at the midpoint, 57.5, half to even 58 (a quote is a limit, not a peg).

**The trip.** The inputs are 100 rations, 1 vehicle and 3 cells; provisions need nothing more, since the trip of 3 ticks is not above the default's `CoverTicks`
of 3. The trader prepares at tick `t` and sends at `t + 1`: `ShipmentRequest{Mode: skiff, To: pit, ToSelf: true}`. It is away in ticks `t + 1` to `t + 3`,
three working ticks, and its welfare window, if WEL adopts 12.3, covers `WindowCycles` times 3. At tick `t + 4` it is in `pit`, its beliefs reseeded to the window
mean of `pit` (12.2), and the default's surplus rule sells the rations in `pit`'s book. Its account shows the sale at the price of `pit` and not at the price of `hub`.
The gate shows in two places: the rations were bought at a price the outside set the lower limit of, and the rumour of `hub`'s price that a smelter-region merchant
reads is `hub`'s published price, which includes the outside's trades, and is its way of learning of the import (11.3).

**The Weyl spread** of the first twelve agent IDs from 1, in basis points: 6 180, 2 360, 8 541, 4 721, 901, 7 082, 3 262, 9 442, 5 623, 1 803, 7 983, 4 164.
Consecutive IDs are spread over the interval with the largest gap between neighbours of 1 459 for ten merchants (the ideal is 1 000); IDs seven apart are
less even (2 836 for ten), `lane.out`.

## 20. Notes settled

For `AGENTS.md`: each open note tagged `MER` or `ALL`, and the others that name merchants, is applied, rejected or left
open here, with the text to append in the handoff.

| Note | Decision | Where |
|------|----------|-------|
| N-011 Transport, storage and carriage | **Applied for MER; leave open for STO.** The contract money is held in an Escrow account by transfers, not in a reservation (9.6); the open items of CAR and the contract of MER are settled here | 9 |
| N-046 Register entries owed by unwritten specs | **Leave open** for the other specs, with this added: MER has its entries (17) | 17 |
| N-084 Scenarios that wait for trade, money and the coarse tick | **Leave open for STY.** MER now exists: the dynamic parts of X1 and X2 and S28 to S32, S46, S47 can run with a merchant role in `overlay/trade.toml` (19.1 shows the keys); the scenarios of 17 are proposed | 17, 19 |
| N-090 What MIG and the specs that tie obligations to a region must do with a relocation | **Applied for MER and CAR:** `open_contract` is registered (9.8.1); a merchant on a trip is refused by `open_shipment` and `open_contract` and is not a migrant (12.4.1). Leave open for MIG, PRO, SKL and INT | 9.8, 12.4 |
| N-100 What transport leaves to MER and INF | **Applied; close.** The consignee is the payer of a contract (8, 9); a diverted cargo counts as reached (9.7); a traveller reseeds its beliefs on arrival (12.2); the backhaul double count stands and is measured (10); the contract is built on the shipment and does not use the book (9.1); the shipper's rules are in 7 | 7 to 10, 12 |
| N-106 What a strategy may see of other regions | **Applied; close.** The merchant reads `Rumours` with `Routes`, counts `Inbound` and its own `Outbound`, caps the dispatch, spreads its thresholds by agent, uses `SourceAge` and differences nothing (5, 7); no view carries a remote price | 4, 5, 7 |
| N-016 Keep the parameter registry true | **Applied:** rows for `Modules.Merchants`, `MaxOpen` and the nine fields of the block (13, handoff b); `MerchantFeeShare` keeps its row; no unnamed parameter | 13 |
| N-017 Words and names | **Applied:** merchant, lane, candidate, trip, lot, cargo set, trade inputs, fee share, bar, lane lock, cautious revenue, offer, take, round, payer, deliverer, contract, escrow (contract); setting words appear only in the flagged examples of 19 | 19 |
| N-027 No default macro constant is stable | **Applied:** every default is a labelled placeholder, nothing is claimed stable, and the models are not claims about the library | preamble, 13, 18 |
| N-103 Running several spec writers in parallel | **Applied:** no shared file was edited, proposals are in the handoff, provisional IDs are used (`MER-A1` to `A7`, `MER-D1` to `D8`, `MER-S1` to `S6`), the models ran in Docker with no bytecode. The seam check against SUB and MIG is in the handoff | 17 |
| N-099 A trip is a gap in income | **Leave open for WEL and STB**, with this proposed: the effective window of a merchant role counts the longest trip of its modes (12.3) | 12.3 |
| N-101 Does the default Target count Inbound? | **Leave open for AGT**, with this added: the merchant counts its `Inbound` as a consignee (8.2) and its `Outbound` as a shipper (7.1); the default strategy is never a consignee unless a consumer's strategy sends it goods | 8 |
| N-107 New money faucets and sinks start from MON | **Reject as not applicable:** the contract moves money between non-boundary accounts and adds no faucet, sink or reason (9.6.1) | 9.6 |
| N-104 Region-keyed tables use the one override mechanism | **Leave open** (the others still owe it), with this added: MER adds no region-keyed table; a lane is not a region-keyed row | 13 |
| N-012 Events, metrics and log keys to define | **Leave open**, with this added: the events `ContractFormed`, `ContractSettled`, `ContractRound` and `MerchantsPlanned`, the log keys `contract`, `offers`, `takes` and `merchants`, the metrics `economysim.contract.formed`, `.value`, `.escrow` and `economysim.merchant.decisions`, and a new metric attribute `economysim.decision` (15) | 15 |
| N-020 What each spec contributes to a checkpoint | **Leave open**, with this added: the contracts in flight and the next contract ID (16) | 16 |
| N-021 Anything that creates or destroys money or goods | **Leave open**, with this added: MER adds no faucet, sink or reason | 9.6.1 |
| N-038 Seed prices, newcomers and migrants | **Leave open for MIG**, with this added: a self-carried traveller reseeds its beliefs on arrival (12.2) | 12.2 |
| N-045 and N-065 Dampers that act silently | **Leave open**, with this added: the merchant's dampers are counted by `MerchantsPlanned` through the plan notes (15.1) | 15.1 |
| N-009 What the world and tick phases must decide | **Leave open**, with this added: the positions of 12.5 (settlement in phase 1 after the arrivals, the round in phase 3 after the last plan) | 12.5 |
| N-112 An outside node as a remote information source | **Leave open**, with this added: the gate region's market is a rumour source, so an interior merchant hears of the gate's price (11.3) | 11 |
| N-092 Overlays and scenarios handed to REF | **Leave open**, with this added: `overlay/trade.toml` gains a merchant role, a consignee role and `[modules.merchants]` (19.1); the scenarios MER-S1 to S6 (17) | 17, 19 |
| N-015 Placeholders to calibrate | **Leave open**, with this added: the defaults and bounds of 13, the warning threshold of 14.2 and every number of 18 and 19 | 13 |
| N-018 Unverified evidence | **Applied:** no source was read for this spec, and none is used for a default | preamble |

Proposed new notes are in the handoff.

## 21. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Defaults and bounds are placeholders (N-015, N-027).

### Functional

| ID | Requirement |
|----|-------------|
| FR-MER-001 | The merchant strategy MUST be registered by the name `"merchant"` through `WithStrategy`, `strategy.Options()` MUST return the options that register the standard names, and a role naming an unregistered strategy MUST be rejected with `ErrUnknownStrategy` at `NewWorld` (3.1). |
| FR-MER-002 | `strategy.Merchant` MUST hold a `DefaultStrategy` in a named field, its `Work` and `Revise` MUST be the default's, and its `Trade` MUST differ from the default's only as 3.3 says (3.3). |
| FR-MER-003 | A role whose `Strategy` is `"merchant"` MUST have a merchant block, `Informed` true, `Recipes` empty, every group of scale 1 and no self-carried mode of `Crew` above 1 in its `Modes`, and a violation MUST be rejected with `ErrMerchantShape` or `ErrMerchantWeight` (3.2, 3.6, 14.1). |
| FR-MER-004 | Each trade input of a merchant role (the `Haul` commodity of a hired mode, the `Vehicle` and the `Feed` commodities of a self-carried mode) MUST be in its `Trades`, and its cargo set MUST NOT be empty, or the configuration MUST be rejected with `ErrMerchantShape` (3.2.2). |
| FR-MER-005 | The merchant strategy MUST keep no state, MUST take no random draw other than those the default strategy takes for the same intents, and MUST compute the fee share of an agent from its agent ID alone (3.1.3, 3.5). |
| FR-MER-006 | The inputs of the planned trip MUST be the `Job_c` of 3.4.2, bought in tier 2 with the purpose `PurposeInput`, and with no trip planned `Job_c` MUST be 0 for every commodity so that the default's surplus rule sells what the merchant holds (3.4). |
| FR-MER-007 | Each candidate (lane and mode) MUST be decided by the first applicable case of 5.1.2 and counted exactly once. |
| FR-MER-008 | The cautious revenue MUST be the rumoured `Price` when the rumoured `Mean` is 0 and `min(Price, Mean)` otherwise, and `Rnet` MUST be `Rhat * (10 000 - ConsigneeCutBP) / 10 000` truncated for a hired mode and `Rhat` for a self-carried mode (5.2.1). |
| FR-MER-009 | A rumour whose `SourceAge` is above `MaxSourceAge` MUST NOT be acted on: the candidate MUST be set aside as `Stale` (5.2.2). |
| FR-MER-010 | A candidate for a commodity with a shelf life `S` above 0 MUST be set aside as `Perishable` if and only if the travel time `k` is above `S - 1 - max(1, OldestAge)` (5.3.1). |
| FR-MER-011 | The cost of carriage of a hired candidate MUST be `n * s * P_haul`, with `n` the carrying units the lot needs, `s` the haul units per carrying unit and `P_haul` the local price of the haul commodity (5.4). |
| FR-MER-012 | The cost of carriage of a self-carried candidate MUST be the wear, the feed and the time of 5.4, with the time at `k` times the agent's own upkeep cost times `TimeCostBP` over 10 000 and not at a regional wage or any aggregate (5.4). |
| FR-MER-013 | A candidate MUST pass if and only if `C * 10 000 <= phi * G`, with `G = q * (Rnet - P_acq)` positive, evaluated with checked 128-bit arithmetic (5.5.1). |
| FR-MER-014 | Among the passing candidates the merchant MUST plan the one with the greatest `G - C`, ties to the lower region, then commodity, then mode, and with none it MUST plan no trip (5.1.2, 5.5.2). |
| FR-MER-015 | `BreakEven` MUST be `Rnet - ceil(C * 10 000 / (phi * q))` and `AskLimit` MUST be `P_acq + ceil(C * 10 000 / (phi * q))`, both with the lot of the final pass (5.5.3, 6.2.2). |
| FR-MER-016 | `expected_margin_per_haul_unit` MUST be `G / (n * s)` truncated, the limit of a merchant's bid for haul units MUST NOT exceed `expected_margin_per_haul_unit * phi / 10 000` truncated, and the limit of its bid for the cargo MUST NOT exceed `BreakEven` (5.6; this is `FR-CAR-004` with the agent's own fee share). |
| FR-MER-017 | The fee share of agent `a` MUST be `clamp(MerchantFeeShare + FeeShareSpreadBP * (2 * u(a) - 10 000) / 10 000, 1, 10 000)` with `u(a)` the Weyl value of 5.7.1 computed by `math/bits.Mul64`, and it MUST equal `MerchantFeeShare` when `FeeShareSpreadBP` is 0. |
| FR-MER-018 | A product of the margin arithmetic that overflows MUST drop the candidate as `Unaffordable`, and every rounding MUST be against the merchant: costs up, revenues down (5.8). |
| FR-MER-019 | Modes MUST be compared on the same lane by the same rule, each with its own lot, and the passing mode with the greatest `G - C` MUST be chosen, ties to the lower mode ID (6.1). |
| FR-MER-020 | The lot MUST be the least of the lot cap, what the merchant can pay above its reserve and, when the trip is ready, what it holds, sized in the two passes of 6.2.2. |
| FR-MER-021 | A trip MUST be ready only when the merchant holds all its inputs; a ready trip that passes the test MUST append the request `{ToSelf: true}` (self-carried) or one `DeliveryOffer` with limit `AskLimit` (hired), and a trip that is not ready MUST append neither (6.3). |
| FR-MER-022 | A merchant with goods on the way to it (`Inbound`) or with a job in progress MUST plan no self-carried trip, and a merchant in the own-use mode MUST plan no trip and post no take; each MUST be counted as `Waiting` (6.4). |
| FR-MER-023 | A merchant whose work share is 0 MUST plan no trip and post no take, and MUST still buy its upkeep and sell its surplus (6.5). |
| FR-MER-024 | A candidate MUST be set aside as `Locked` while the merchant has units of the commodity on the way to the destination (`Outbound`) (7.1). |
| FR-MER-025 | A merchant's plan MUST carry at most one shipment request or one delivery offer, of at most `LotUnits` carrying units (7.1). |
| FR-MER-026 | The merchant MUST read only the rumours of the view of the tick: two calls with equal views and equal draws MUST return equal plans whatever happened in earlier ticks (3.5, 7.1). |
| FR-MER-027 | `AgentView` MUST carry the merchant group of 4.1 for an agent whose role has a merchant block, with `Outbound` fixed at the start of phase 3 so that it does not depend on the order of agents (4.1). |
| FR-MER-028 | A consignee's take MUST be sized by 8.2: the limit from its local price and `TakeMarginBP`, the want from `TakeUnits` less its free stock and the total on the way to it, the units it can afford above its reserve, and the least of them. |
| FR-MER-029 | A take MUST be posted only by a role with `Consigns` and a work share above 0, and only for a cargo commodity that is not the cargo of the trip the agent plans this tick (8.2). |
| FR-MER-030 | With `Modules.Merchants` absent every offer and take MUST be refused with `contracts_off`, and no contract, Escrow account or contract event MUST exist (9.3). |
| FR-MER-031 | Admission MUST apply the checks of 9.3 in their order, refuse an offer or a take whole with the first reason that holds, change nothing, and count the refusal in `ContractRound`. |
| FR-MER-032 | The round MUST run once per tick in phase 3 after the last agent's plan and before the estates' asks, MUST visit the lanes in ascending (destination region, commodity) order, and MUST do nothing when no offer was admitted (9.4.1, 9.4.2). |
| FR-MER-033 | The round MUST draw one `uint64` key per admitted offer and then per admitted take in admission order, and no other random number, and a tick with no admitted offer or take MUST draw none (9.4.2). |
| FR-MER-034 | Matching MUST visit the offers of a lane by limit ascending, then key, then admission position; for each it MUST take the first take, by limit descending then key then admission position, whose limit is at least the offer's and whose remaining quantity is at least the offer's quantity; an offer MUST be indivisible and a take divisible (9.4.3). |
| FR-MER-035 | The price of a match MUST be the midpoint of the two limits rounded half to even, and it MUST lie between them (9.4.3). |
| FR-MER-036 | Formation MUST apply the checks of 9.5 in order, `too_many_contracts` then `payer_funds` then the dispatch, a failure ending the attempt with its reason, dropping the offer, leaving the take unchanged and changing nothing else. |
| FR-MER-037 | Formation MUST dispatch the shipment as the deliverer by the shipment mechanism with all its checks and effects and the payer as consignee, and only then MUST it transfer `price * quantity` from the payer to an Escrow account under `escrow_in`. |
| FR-MER-038 | A contract MUST exist only if its dispatch and its escrow both took place; its deadline MUST be the arrival tick of its shipment, and a later change of the transit factor or of the graph MUST NOT move it (9.5). |
| FR-MER-039 | An Escrow account MUST hold the money of at most one open contract and nothing else, MUST be the lowest-ID open Escrow account of the destination region that no open contract references or a new one, MUST NOT be closed, and MUST be empty when no contract references it (9.6.2). |
| FR-MER-040 | In phase 1, after the arrivals of transport, the contracts whose shipment arrived MUST be settled in contract ID order by the table of 9.7: `escrow_out` of `price * reached` to the deliverer, `escrow_refund` of `price * spoiled` to the payer or, if it no longer exists, to the estate of the destination region. |
| FR-MER-041 | If the deliverer of an open contract is removed, the contract MUST end at the removal with the whole escrow returned to the payer, or to the estate of its region if the payer no longer exists, and the shipment MUST NOT be touched (9.8.2). |
| FR-MER-042 | If the payer of an open contract is removed, the contract MUST stand and be settled at the arrival with the estate of the destination region standing in for the payer, and the deliverer MUST be paid for the units diverted as for the units delivered (9.7, 9.8.2). |
| FR-MER-043 | With the module on, `open_contract` MUST be registered as a `region_bound` refusal, open for the payer and the deliverer of an open contract, so that the public relocation returns `ErrRegionBound` for them (9.8.1). |
| FR-MER-044 | A contract MUST create or destroy no money or goods and add no ledger reason, and every identity of `spec-ledger.md` 9.1 MUST hold with money in escrow (9.6.1). |
| FR-MER-045 | `World.Audit` MUST also check, with the module on, that each open contract's Escrow account holds exactly `price * quantity`, that an unreferenced Escrow account is empty, that contract IDs are unique, and that the shipment of each contract is in flight (9.6). |
| FR-MER-046 | No more than `MaxOpen` contracts MUST be open at once: a formation that would exceed it MUST fail with `too_many_contracts` (9.5, 13). |
| FR-MER-047 | At the arrival of a self-carried traveller its beliefs MUST be reseeded by the rule of `spec-beliefs.md` 8.3 for every commodity of its tradable set, taking no random draw (12.2). |
| FR-MER-048 | The haul units redeemed by a hired shipment MUST be attributed in the flow totals to the origin region (`spec-transport.md` 7.3.3), so that the haul units of the two directions of a pair are readable separately (10.4). |
| FR-MER-049 | A merchant MUST NOT name an outside node in a request, an offer or a candidate, and an offer naming one MUST be refused with `outside_destination` at admission (11.2). |
| FR-MER-050 | Each error of 14.1 MUST be rejected with its sentinel and the path of the field, each with a failing-case test. |
| FR-MER-051 | Each warning of 14.2 MUST be raised without failing validation, once, at `Warn` with `rule` and `path`, and MUST NOT be logged inside the tick. |
| FR-MER-052 | The fields of 13 MUST have the defaults and bounds there: an absent field takes its default, a required field that is absent or a value outside its bound MUST be rejected with `ErrOutOfBounds` naming its path. |
| FR-MER-053 | The configuration hash MUST cover `Modules.Merchants`, the merchant block of each role and `Role.Strategy`, so that changing any of them changes the hash. |
| FR-MER-054 | The file MUST write the module as `[modules.merchants]` and the block as `[role.<id>.merchant]` with the snake case keys of the fields, an unknown key MUST be rejected with `ErrUnknownKey`, a non-integer value with `ErrNotInteger`, and a file and the same configuration built in Go MUST have equal hashes and runs. |
| FR-MER-055 | `ContractFormed`, `ContractSettled`, `ContractRound` and `MerchantsPlanned` MUST be emitted with the fields of 15.1.1, with no map, pointer or wall-clock time, only when an observer was supplied and only with the module on (for the first three), and `ContractRound` MUST be counted once per destination region and tick in which an offer or a take was submitted. |
| FR-MER-056 | The events of a step MUST come in the order of 15.1.1: in phase 1 the `ContractSettled` events after the `ShipmentArrived` events in contract ID order, in phase 3 during the round the `ShipmentDispatched` and `ContractFormed` of each contract in formation order, then `ContractRound` and `MerchantsPlanned` in ascending `RegionID`, then `RumoursServed`. |
| FR-MER-057 | `MerchantsPlanned` MUST carry per region the sums of the plan notes of the agents given a view there, and a run with and without an observer MUST give equal results: the notes MUST influence nothing (15.1.2). |
| FR-MER-058 | The module MUST log nothing above `Debug` per contract, refusal or agent, MUST have no `Info` line in the tick, and the keys `contract`, `offers`, `takes` and `merchants` MUST be used as in 15.2. |
| FR-MER-059 | The metrics of 15.3 MUST be derivable from the events, and `economysim.decision` MUST take only the eleven constants of 15.3. |
| FR-MER-060 | The checkpoint MUST contain the next contract ID and the contracts in flight in ID order with the fields of 16, and nothing else of this spec; the fee share, the trip, `Outbound`, `Inbound` and the free list MUST NOT be saved. |
| FR-MER-061 | A world restored between ticks MUST settle the same contracts at the same ticks, take the same key draws and give the same events as an uninterrupted one. |
| FR-MER-062 | For a given seed, configuration and sequence of calls, the offers, matches, contracts, settlements and events MUST be identical across runs and platforms (architecture.md 7.4), checked by a golden test on a world with merchants and consignees. |
| FR-MER-063 | With `Modules.Merchants` absent and no role with a merchant block, results MUST equal those of a world without this spec, and with the module on and every strategy the default, results MUST equal those of a world with it off apart from the hash. |

**Restated from `spec-carriage.md` (9.9), under their old IDs:**

- **FR-CAR-001.** A contract payment MUST be a transfer of money from the payer to an Escrow account at formation (`escrow_in`), held there
  until the arrival of the contract's shipment, and then released from it to the deliverer for the units that reached the destination
  (`escrow_out`) and returned for the units that did not (`escrow_refund`); no contract MUST create or destroy money. (FR-MER-037, FR-MER-040,
  FR-MER-044.)
- **FR-CAR-002.** If the deliverer is removed before the arrival, the escrow MUST be returned in full to the payer (`escrow_refund`) and the
  contract MUST end. (FR-MER-041.)

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-MER-001 | With the default logger and observer, the margin arithmetic of a candidate, the matching of a lane, the formation and the settlement MUST allocate nothing beyond scratch held by the world and reused across ticks. |
| NFR-MER-002 | The cost of a merchant's `Trade` SHOULD be `O(R * C * M)` in the regions `R`, the cargo commodities `C` and the modes `M` of its block, and the round `O(O log O + T)` in the admitted offers `O` and takes `T`; the figures are counts and unmeasured (N-015). |
| NFR-MER-003 | The margin, the fee share, the sizing and the matching MUST live in `internal/merchant` over primitive slices, import the standard library and `internal/household` only, and know no region, commodity, role or mode (architecture.md 2.9, 9.1). |
| NFR-MER-004 | The events and the plan notes of the module MUST NOT be constructed when no observer was supplied, proved by an allocation test (observability-and-logging.md 4.5, 8.2). |
| NFR-MER-005 | The fee share, the margin arithmetic, the matching and the settlement MUST have fuzz tests (testing.md 7.4) asserting that they never panic for any prices, quantities, shares or limits, that the price of a match lies between the limits, that `escrow_out` plus `escrow_refund` equals `price * quantity`, and that overflow is reported and not wrapped. |

## 22. Open questions

1. **Forward and standing contracts, and a third-party carrier.** The contract is a spot-forward sale (9.10). A term, a repeating contract, or a
   carrier that takes custody of goods are all possible and each needs a record of who is bound, a deadline and a default with a cost. Next step: when
   a setting needs a courier trade, with CRD for the credit that a term makes natural.
2. **A cost basis.** The merchant values what it holds at the replacement cost (3.5.1), so it can sell at a loss and forgets that it did. A landed-cost anchor
   for the cost band of its asks (`spec-beliefs.md` 3.2.4 allows it) needs a per-agent, per-commodity cost saved in the checkpoint. Next step: run the
   merchants and see whether the losses matter.
3. **A spread drawn at creation.** The Weyl function of the agent ID (5.7) departs from the draw at creation that INF and the register name. A draw adds an
   entry to the list of 3.5.2 of AGT, a field and checkpoint state. Next step: the first run with merchants whose IDs are far apart.
4. **A memory of a failed trip.** A merchant that lost on a lane tries the same lane the next tick on the same reading. A cooldown per lane is state per
   agent and commodity. Next step: measure how often a lane is chosen again after a loss (MER-A7).
5. **`MaxSourceAge` from the route.** INF-A3 asks for a `SourceAge` small against the time the goods take. The default of 2 ignores the route. A bound of the
   form `k` or half of `k` is a rule that needs the route in the check. Next step: MER-S3 with several routes.
6. **A partial lot of a mixed-age perishable.** The merchant ships a line only if the oldest lot survives (5.3.2). Choosing the lots would need a named-lot
   request in TRN. Next step: when a perishable lane is common.
7. **Pricing a backhaul once.** A carrier role that serves both directions of a pair, or a third-party courier (question 1), would price the round trip once.
   Next step: with question 1.
8. **A second extension point for matching.** The round does not call `Clearing` (9.4.4). A custom matching (a pro-rata allocation, a call by region pair)
   would be an extension point and needs a second implementation to justify it (architecture.md 4.2).
9. **A basket.** One cargo commodity a trip (5.1). A mixed lot pays for one carrying unit once and is cheaper per unit, and the sizing of a mixed load
   against a pair of capacities is a knapsack. Next step: measure the lumpiness of 6.2.3 on the reference chain.
10. **Several lanes a tick.** One trip a plan is a rule constant (7.1). A larger world may want several; the lock and the shared stock make it a different
    sizing problem. Next step: the first consumer's tick rate and the number of merchants.
11. **The take margin has no spread.** A consignee's `TakeMarginBP` is the same for all of a role (8.2). The importers' take loop (MER-A2) may herd like
    the shippers; the spread of 5.7 could be applied to it. Next step: MER-S2.
12. **The split of the surplus.** The midpoint divides the surplus of a contract equally whatever the bargaining power. A price rule that weights the
    scarcer side is a different matching and a different spec (CLR's open question on the price rule applies).
13. **Takes are not seen.** A shipper never sees a take: it prices from a rumour of the destination's market, and the consignee's discount makes its
    estimate optimistic (19.1). A rumour of takes would be a second record kind of INF.
14. **A home region.** A merchant has none (12.4.2). A merchant that returns to a base to restock, or whose household is tied to a place, would need a field and a
    rule. Next step: MER-A6.

## 23. Where this lives in the code

| Concern | Home |
|---------|------|
| `Merchant` (the strategy type), `Options` | `strategy/merchant.go`, `strategy/options.go` (the package exists once a second strategy exists, architecture.md 2.6) |
| The margin, the fee share and the Weyl spread, the cost of carriage, the sizing of a lot, the two-pass break-even, the matching and the midpoint, the settlement arithmetic | `internal/merchant` over primitives (architecture.md 2.9): `FeeShare(base, spread, id)`, `Margin(...)`, `Fit(...)`, `Match(offers, takes, keys)`, `Settle(price, quantity, reached)`; it imports `internal/household` for the shared tiers |
| The Weyl offset `w(a)` of 5.7.1 | one helper, `Weyl32(id)`, in `internal/ordering` (deterministic helpers over primitives), shared with the death-carry offset of `spec-population.md` 3.8.3 and the spread of margins of MIG; `internal/merchant` imports it |
| `DeliveryOffer`, `DeliveryTake`, `PlanNotes`, `Contract`, `ContractID`, the round, the formation, the settlement, the module's state and its steps in phases 1 and 3 | root package, `contract.go` |
| The plan fields `Offers`, `Takes`, `Notes`; the view group `Merchant` and `Outbound` | root package, `strategy.go` (`spec-agents.md` 6.3, 6.4); `Outbound` beside `Inbound` in `transport.go` |
| `Role.Merchant`, `Modules.Merchants` and their bounds | root package, `config.go` and `defaults.go`; `bounds.go` (`MaxLotUnits`, `MaxContracts`, `MaxTimeCostBP`) |
| Validation, the sentinels `ErrUnknownMode`, `ErrMerchantShape`, `ErrMerchantWeight` and the warnings of 14.2 | root package, `validate.go`, `errors.go` |
| The events `ContractFormed`, `ContractSettled`, `ContractRound`, `MerchantsPlanned` | root package, `event.go` |
| Escrow accounts, `escrow_in`, `escrow_out`, `escrow_refund`, the audit checks | `internal/ledger` (`spec-ledger.md` 16); the root opens and reuses the accounts |
| The `open_contract` refusal | root package, `contract.go`, registered at construction |
| The reseed of a traveller at arrival | root package, `transport.go`, calling `internal/beliefs` |
| The reason constants (9.3, 9.5), the decision constants (5.1.2) and the log keys | `internal/logkeys` |
| The file shape | `config/` (`spec-config.md` 6) |
| The models | `doc/specs/models/mer/` |
