# Spec: Estates and the money pool

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-EST-NNN` and
`NFR-EST-NNN`, which tests cite (testing.md section 4). They are in section 5.

Layer: core for the estate account, its pseudo-agent, its listings and the use of the pool to fund an entrant; module for relief and the pool as a money-flow target (`spec-config.md` 6.2.4, [ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

When an agent dies or goes bankrupt, what it leaves must go somewhere so that
money and goods stay conserved. This spec covers the regional estate, its money
pool, how the pool funds new agents, and how it releases relief. Section 3 lists
the parameters it owns. How a dead agent's tool is priced for sale is in
`spec-skill-barriers.md` section 2.7.

## 2. Estates and the money pool

When an agent dies or goes bankrupt, what it leaves must go somewhere so that
money and goods stay conserved. Each region has an **estate**, which holds a
**pool** of money. This is a deliberately simple substitute for inheritance: no
heirs, no wills.

2.1. **What passes to the estate.** The agent's money balance (never
negative), less what its creditors take in the waterfall of `spec-credit.md` 9.1 (the claims
of the removed agent as debtor are paid from its balance first, pro rata, and the estate
receives what is left; the remainder of the escrow of a hire of a removed employer returns to
the agent's own account before the waterfall, `spec-agents.md` 3.6 step 2c, so that it is in
that balance and has no reason of its own); its tools, listed under `spec-skill-barriers.md` section 2.7; and
its other goods, listed as asks by the estate pseudo-agent (section 2.7) at the
region's effective reference price (`spec-goods.md` 6.3) with the same markdown.
Perishable goods spoil as usual. Sale proceeds go to the pool. These are ledger
transfers with the reasons `estate_money` and `estate_goods`
(`spec-ledger.md` 7.2), into the region's one estate account. The mechanics of a
removal, in order (the job abandoned, the trainee relationship ended, money and goods
moved, the account closed, `AgentBankrupted`, the only event that removes the agent, with
the events that accompany it in the order of `spec-world.md` 6.11 item 8), are
`spec-agents.md` 3.6; this spec owns only what the estate then does with them.

The steps of `spec-agents.md` 3.6 also pass to the estate what is not money or goods: the
titles of the agent (`spec-land.md` 6.4: they move to the pool of the title's region and no
money moves), the claims of which it is the creditor (`spec-credit.md` 9.2) and its site,
route or ground facilities (`spec-infrastructure.md` 7.1). Its open hires, covers and
delivery contracts end with no transfer of their own. `EstateOpened` counts the lots listed
and nothing else.

**What is counted.** `GoodsListed` counts **lots**: one per distinct lot key (commodity, born,
durability) of the holdings that moved to the estate, whatever the unit count; units travel
separately in `WeightChanged.Units`. `AgentBankrupted`, `EstateOpened` and `RoleChanged` carry
the same count. One function moves the money and then the lots, in key order, keeping the
keys (`spec-agents.md` 3.6 steps 3 and 4). It runs when no reservation is open, so a refusal is
an invariant failure, except a balance overflow of the estate account, which is returned as
`ErrBalanceOverflow`.

Goods also reach the estate by diversion: a cargo whose consignee is removed, outside the destination region or unable to hold it moves from its Transit account to the estate of the destination by `transit_in` (`spec-transport.md` 5.8), and the estate lists it like any other goods (2.7).

2.2. **Which region.** The region the agent was in when it died. A traveller
that dies in transit leaves its money to the pool of the region of its account, the region it departed from (`spec-transport.md` 5.9.2). Its cargo is not in its account: it is in a Transit account and arrives at its consignee, or, if the consignee is gone, is diverted to the estate of the destination by the reason `transit_in` and listed like any estate goods (`spec-transport.md` 5.8, 5.9.3). The `loss` sink is RSK's (`spec-risk.md`, with `Modules.Risk`; `spec-ledger.md` 7.2) and nothing is posted under it by this spec. Escrow for a
delivery contract is handled by `spec-carriage.md` section 2.1.

2.3. **Use of the pool.** The pool funds the starting money of agents created
in the region after the world exists (spawner replacements, newborns,
immigrants the spawner creates). A later agent of weight `w` receives
`min(w * Role.StartMoney, pool balance)`, moved from the pool with the reason
`start_money`. The draw is `min(want, pool)`, with `want` the executor's whole households times the amount; a draw below 1 is no transfer; one function serves `start_money` and `births`. The spawner limits this: it funds entrants only from `EntryPoolShare` of the pool per tick and only in whole households, and the households it cannot fund are not created (`spec-spawner.md` 4.3, 7.3), so a collapsing region does not create agents with part of a grant. `StartMoney` is a field of the role (`spec-agents.md` 3.3, 12), not of
the region, and is not a parameter of this spec. A later agent receives **no goods**
from the estate: it buys its tool and inputs (`spec-agents.md` 3.5.3), and the
estate's goods reach it only by sale. The default spawner grants none (`spec-spawner.md` 4.3); a goods grant by a consumer's spawner would be its own faucet with its own reason, and is not drawn from the estate's listings.

Initial agents at world creation receive their money by `initial` (`w *
Role.StartMoney`, or the warm start's `WarmGroup.Money`, `spec-agents.md` 3.5.1;
`spec-viability.md` 8.3), not from the pool. The pool itself starts empty unless a
warm start seeds it: `WarmStart.Pools` gives a region's pool an amount created by
`initial` into the estate account (`spec-viability.md` 8.3.3; default 0, so a
replacement in the first ticks may start with nothing). Money enters the world only
by `initial` (at construction, a pool seed, and an external agent added during a run,
`spec-agents.md` 3.5.4) and by the faucets of later specs, each a named flow of the
ledger (`spec-ledger.md` 7.3). Otherwise the total, agents plus pools plus escrow, is
constant. A configured grant to the pool of a region is a named faucet too (`spec-money-supply.md` 4.3.2, 7.4); a pool target needs this module on (`ErrModuleDependency`), although the estate account exists whether or not it is (`spec-regions.md` 3.6.2): the pool policy of 2.6 is the module's. A levy on a pool collects after relief in the same tick (P1.7 after P1.6) and before that tick's entry step, where the starting money of an entrant is funded at its creation, in phase 7 (`spec-money-supply.md` 7.5; `spec-world.md` ruling C1). A tax on a pool collects at P1.7b, after relief and after the settlement of claims at P1.7a (`spec-fiscal.md` 5.6); the treasury it is paid to is not the pool. The interest, the principal and the rent that the estate receives as a creditor or a holder of titles enter its account and so the pool (`spec-credit.md` 9.2.2; `spec-land.md` 5.3).

2.4. **Consequences.** A prosperous region accumulates a pool and funds
well-endowed newcomers. A collapsing region's pool runs dry, so new agents
start with nothing, cannot buy inputs or a tool, and fail fast. The collapse
can therefore reinforce itself. Recovery needs money from outside: immigrants
carrying their own money, or merchants. A pool that is not being used for
starting money is released as relief (section 2.6), so it does not sit idle.
The pool balance is exposed per region in `Snapshot` so it can be watched.

2.5. **Events and state.** An `EstateOpened` event is emitted when an agent
is removed, by every removal and whether or not the pool policy of 2.6 is on (agent,
region, balance, number of lots listed (2.1); `Balance` is the money the estate holds, the estate's pool after the intake; it accompanies `AgentBankrupted`, the event of
`spec-agents.md` 14 that removes the agent and carries the cause). The agent-spawned event
carries the starting money it received (`StartMoney`, `spec-agents.md` 14).

2.6. **Relief.** Each tick, in the start phase (`spec-agents.md` 4, phase 1), the
region releases a fraction of what is left in its pool after the starting-money draws of the previous tick's entry step (phase 7, where a new agent is created and funded, `spec-spawner.md` 4.3). The pool is therefore shared in time: entrants may draw at most `EntryPoolShare` of it in phase 7, and relief takes its fraction in the next phase 1 as relief to agents
living below subsistence. This is the answer to an idle pool, and it is
configurable per region: `PoolRelease` (basis points of the pool per tick,
default 100) and `ReliefBelow` (the welfare ratio under which an agent
qualifies, default 10 000, which is subsistence).

```
release   = pool * PoolRelease / 10_000
need_i    = U_i * (ReliefBelow - ratio_i) / 10_000      (for agents with ratio_i < ReliefBelow)
share_i   = release * need_i / sum(need)
payment_i = min(share_i, need_i)
```

`U_i` is the agent's effective upkeep cost this tick (`spec-agents.md` 5.2, 5.6.2),
weight included, so for a cohort `need_i` is the cohort's total and `ratio_i` is the
per-household ratio (`spec-scale.md` 4.3). `need_i` is the money that lifts the
agent's income this tick to the relief line. Relief is paid with the reason `relief`,
so it can be told from earned income. The products are computed with the checked
128-bit helper (`spec-currency.md` 7). All values are integers,
truncated, and overflow-checked. Amounts that are not paid out (truncation, or
`release` larger than the total need) stay in the pool. Each agent's payment
depends only on its own need and the totals, so no ordering of agents affects
the result.

- Relief counts as income in the welfare ratio (`spec-welfare-and-wages.md` section 2.1), but is excluded
  from the observed wage (`spec-welfare-and-wages.md` section 4.3), because it is not earned by working.
- Relief tops agents up to the relief line only, and the target-income rule
  (`spec-welfare-and-wages.md` section 5) still sees them below target, so they keep working.
- Relief does not enter the migration trigger or the migrating household's wage
  (`spec-migration.md` 5.4): the trigger reads the earned ratio, which leaves relief out. What
  remains is the relief a poor arrival draws from the destination's pool.
- Relief counts the receipts of an own-use job in the ratio (`spec-subsistence.md` 7), so a
  household that feeds itself draws less.
- Relief is in the balance that a debtor's settlement of claims reads at P1.7a, up to the shield
  (`spec-credit.md` 8.8): money the pool pays a poor household can be paid on to a creditor.
- A `ReliefPaid` event is emitted per region per tick when anything is paid
  (total paid, number of recipients, and `Households`, the sum of the recipients'
  weights). It carries no per-agent map. It is the pool policy's event: it is not
  emitted when the module is off, whereas `EstateOpened` (2.5) is core.

### 2.7 The estate as a pseudo-agent

The estate sells the goods it holds through the ordinary market, so that estate
goods meet buyers in the same `Clearing` as any other (`spec-clearing.md`). To do
that it acts as a **pseudo-agent**: something that places orders without being
a household.

2.7.1. Each region has exactly one estate pseudo-agent, created with the region,
before any household. The world creates it, with its account, in region order at construction (`spec-regions.md` 3.6, whether or not this module is on). It has one ledger account of the kind Estate
(`spec-ledger.md` 3.1) that holds **both** the pool's money and the unsold goods.
So the proceeds of a sale, a `trade` transfer into that account, are the pool's
money, with no separate path.

2.7.2. It places **asks only**. A bid from an estate is rejected with
`side_not_allowed` (`spec-clearing.md` 3.2). Each ask is backed by a reservation
of goods in its own account; it needs no cash, so it has no budget to check.

2.7.3. It places **one ask for each lot it holds**, naming the lot
(`spec-ledger.md` 6.7). This is the exception to the one-order rule of
`spec-clearing.md` 3.1.1: the price of a tool depends on its remaining durability
(`spec-skill-barriers.md` 2.7), so items of one commodity with different wear are
different asks. The price of an ask is

- for a durable: the region's effective reference price `ref` of the commodity times
  the lot's remaining durability `left` over the commodity's full durability `full`, and for
  other goods the effective reference price; this first price is `trunc(ref * left / full)`
  through the checked helper, never below the resolution floor (`spec-currency.md` 3.1);
  `full` is the commodity's `Durability`, and a commodity with `Durability` 0 is listed at the
  reference price with no factor;
- reduced, at P5.3 (`spec-world.md` ruling C6), for each tick in which the lot had an admitted
  ask and is still held (so it was not wholly sold): the new price is
  `max(floor, price - trunc(price * EstateMarkdown / 10 000))`, the product through the
  128-bit helper; a price at or below the floor stays where it is, and with `EstateMarkdown`
  0 nothing falls.

  An estate's ask is priced by this rule and not by a belief, so it may lie outside the belief interval
  of its market, and the published price of a tick in which an estate sold is bounded below by the
  resolution floor only (`spec-stability.md` FR-STB-008).

The ask is for all units of the lot. A row of the listing state (2.7.5) is created at the first
walk that sees the lot held with no row, and dropped when the lot is gone, at the P3.4 walk and
at P5.3. A lot that receives units keeps its price, and the joining units are listed at it. An
ask that the clearing rejects is not an ask, so its lot is not marked down.

The estate is exempt from a right (`spec-institutions.md` 5.5.2), and its asks are bound by a
legal price (6.3.3 there): a legal floor raises the limit of the admitted order and leaves the
listed price of the lot, which is state (2.7.5), unchanged. The "last price" of the markdown
above is that listed price.

2.7.4. It has no strategy, no price beliefs, no household, no weight, no upkeep
and no welfare ratio. It cannot die or go bankrupt, it draws no relief and is not
counted in the number of agents or households (`spec-scale.md` 3.2, FR-SCL-015). A treasury pseudo-agent (`spec-polity.md` 5.2) is likewise in no count. The estate may be the creditor of a claim by inheritance (`spec-credit.md` 9.2); it never lends, demands repayment or forgives.
No `AgentSpawned` event is emitted for it, and the snapshot has no row for it and nothing to mark: its identity is the `RegionID` of the region row and its money is `MoneyPool` (`spec-observability.md` 5.1.2). Its orders are otherwise ordinary: they are placed with the others before
any market clears, take a tie-break key like any order (`spec-clearing.md` 4.2),
and produce `OrderPlaced` and `TradeExecuted` events carrying its agent ID.

2.7.5. The estate keeps, for each lot it is selling, the lot's current ask price.
That price is state, so it is part of what this spec contributes to a checkpoint
(ADR 0002): the list of (lot key, current price) per region, in canonical order, for the
lots the estate **holds** that have a row. A held lot with no row is priced at the first price
(2.7.3) when the estate next lists it, which equals an uninterrupted run because the first
price is a function of the configuration. The `asked` mark of the tick is scratch and is not
saved. A restore refuses rows out of order or repeated, a price below 1 or above `MaxPrice`,
and a row whose key is not a lot the estate holds.

2.7.6. Pseudo-agent IDs are in the same ID space as agents, allocated first, one
per region in region order, as `spec-regions.md` 3.6.1 fixes (the pseudo-agent of the region with `RegionID` `r` has agent ID `r`), so every ordering that uses an agent ID
(`spec-clearing.md` 4.1) is unaffected by the estate's presence beyond the extra
orders it places. With the external module on, the counterparties of its outside nodes take the agent IDs that follow the estates (`spec-external-sector.md` 4.2.2), the treasuries of governed jurisdictions follow the counterparties (`spec-polity.md` 5.2.2), and the households' IDs follow them; the estate IDs stay `0..n-1`. The order is estates, counterparties, treasuries, households.

### 2.8 Weight changes and a failing cohort

2.8.1. When the spawner or the population module changes an agent's weight from `w`
to `w'`, the money difference passes through the estate account (`spec-scale.md`
4.8; `spec-agents.md` 3.7): the surplus goes to the pool when the weight falls, and
the shortfall is drawn from the pool when it rises, with the ledger reasons
`split`, `merge`, `births`, `starvation` or `deaths` (`spec-ledger.md` 7.2). A draw never takes
more than the pool holds (it is the draw of 2.3, `min(want, pool)`, and a draw below 1 is no transfer). The transfers that reduce a weight are the same function the other way: the money by an accept-money transfer, the goods by an accept-units transfer that keeps `floor(x * w' / w)` of each commodity and sends the difference to the estate (`spec-agents.md` 3.7; `spec-scale.md` 4.8), the units counted in `WeightChanged.Units`; a role change's goods pass as whole lots (2.1). A rise that the pool cannot fund is applied partially, in whole households, from a budget of `EntryPoolShare` of the pool taken once, after the population module's deaths and before the switches (`spec-world.md` P7.1(d), ruling C15), so that the estates of the dead fund the same tick's births; the rest is not created and is counted (`spec-spawner.md` 4.3, 7.3, N-073). A rise by entry brings money and no goods. A household born through the population module brings a stake of `Role.StartMoney * BirthStake / 10 000` from the pool with the reason `births`, money only (`spec-population.md` 4.3); a death by a schedule passes the money of the households that die to the pool with the reason `deaths` (`spec-population.md` 4.7); both reasons are rows of `spec-ledger.md` 7.2.

2.8.2. A cohort does not fail by itself (`spec-agents.md` 3.6.1): an agent with no
money exists and goes unfed. The estate has no part in the decisions of the failure ladder (change
role, relief, partial reduction of the weight, removal). Relief is the estate's rung
(2.6), paid every tick and not conditional; the other rungs are `spec-spawner.md` 5 (privation, role change, reduction by the unfed share, removal), and the birth and death rates are `spec-population.md`'s.

## 3. Parameters

| Scope  | Field          | Meaning                                                     | Default | Bound |
|--------|----------------|-------------------------------------------------------------|---------|-------|
| Region | `EstateMarkdown` | per-tick price reduction of an unsold estate listing, in basis points (used in `spec-skill-barriers.md` section 2.7 and section 2.7.3 above) | 500 | 0 to 10 000 |
| Region | `PoolRelease` | share of the estate pool released as relief each tick, in basis points (section 2.6) | 100 | 0 to 10 000 |
| Region | `ReliefBelow` | welfare ratio below which an agent receives relief, in basis points (section 2.6) | 10 000 | 0 to `MaxBasisPoints` |

`EstateMarkdown` is built with the core; `PoolRelease` and `ReliefBelow` are appended to `Region` with the module.

All are plain data in the root package's config structs, and the owners are in the
parameter-owner table of `README.md`. **The defaults and the bounds are placeholders**
(N-015, N-027): nothing has been run. The starting money of a new agent is
`Role.StartMoney` (`spec-agents.md` 12) and the pool seed is `WarmPool.Money`
(`spec-viability.md` 8.1); neither is a parameter of this spec.

## 4. Stability register

Merged into `spec-stability.md` section 4.5: the estate listing markdown is STB-L38
(2.7.3), the pool-dry collapse is STB-L39 (2.4) and relief suppressing migration is
STB-L40 (2.6). The dampers are `PoolRelease` (D12) and the resolution floor.

## 5. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Each is
derived from the section named.

### Functional

| ID | Requirement |
|----|-------------|
| FR-EST-001 | When an agent is removed, all its money, less what its creditors take in the waterfall of `spec-credit.md` 9.1, MUST pass to its region's estate account with the reason `estate_money` and all its goods with `estate_goods`, conserving money and goods exactly (2.1; `spec-agents.md` 3.6). |
| FR-EST-002 | The region of the estate that receives a removed agent's holdings MUST be the region of the agent's account when it is removed (2.2). |
| FR-EST-003 | A traveller that dies in transit MUST leave the money of its account to the pool of the region of that account, the region it departed from, and a cargo whose consignee is gone MUST reach the estate of the destination by `transit_in` and be listed like any estate goods (2.2; `spec-transport.md` 5.8, 5.9). |
| FR-EST-004 | Each region MUST have exactly one estate pseudo-agent, with one ledger account of the kind Estate that holds both the pool's money and the unsold goods, created with the region and before any household (2.7.1). |
| FR-EST-005 | Pseudo-agent IDs MUST be allocated first in the agent ID space, one per region in region order, the counterparties of the external module (when on) following them, then the treasuries of governed jurisdictions, and the households after (2.7.6; `spec-external-sector.md` 4.2.2; `spec-polity.md` 5.2.2). |
| FR-EST-006 | The estate MUST place asks only, and a bid from it MUST be rejected with `side_not_allowed`; each ask MUST be backed by a reservation of goods in the estate's account and need no cash (2.7.2). |
| FR-EST-007 | The estate MUST place one ask for each lot it holds, for all units of the lot, naming the lot; each lot's price MUST be kept as state and a lot that receives units MUST keep its price; an ask the clearing rejects MUST NOT count as an ask (2.7.3, 2.7.5). |
| FR-EST-008 | The first ask price of a durable lot MUST be the region's effective reference price times the lot's remaining durability over the full durability, and of any other lot the effective reference price, truncated through the checked helper (`trunc(ref * left / full)`) and never below the resolution floor, and a commodity with `Durability` 0 MUST be listed at the reference price (2.7.3). |
| FR-EST-009 | At P5.3, for each lot that had an admitted ask in the tick and is still held, its ask price MUST become `max(floor, price - trunc(price * EstateMarkdown / 10 000))` through the 128-bit product, a price at or below the floor MUST stay, and with `EstateMarkdown` 0 nothing falls (2.7.3). |
| FR-EST-010 | The estate pseudo-agent MUST NOT be counted in the number of agents or households, MUST NOT cause an `AgentSpawned` event, and MUST have no row among the agents of a snapshot (its identity is the `RegionID` of the region row and its money is `MoneyPool`); a treasury, like an estate, has no per-agent row and is in no count (its identity is `JurisdictionRow.Agent`) (2.7.4). |
| FR-EST-011 | The estate MUST NOT die, go bankrupt, receive relief or be passed to a strategy (2.7.4). |
| FR-EST-012 | The estate's orders MUST be placed with the others before any market clears, MUST take a tie-break key like any order, and the events they produce MUST carry the estate's agent ID (2.7.4). |
| FR-EST-013 | The proceeds of a sale by the estate MUST be a `trade` transfer into the estate account, and the pool balance reported per region in the snapshot MUST equal that account's money balance (2.4, 2.7.1). |
| FR-EST-014 | An agent created after the world exists, of weight `w`, MUST receive `min(w * Role.StartMoney, pool balance)` from the pool with the reason `start_money` (a draw below 1 is no transfer; households added to an existing agent use `births`), and MUST receive no goods from the estate (2.3). |
| FR-EST-015 | An initial agent MUST receive its money by `initial` and not from the pool, and the pool MUST start at the amount of `WarmStart.Pools` for its region, created by `initial` into the estate account, or 0 (2.3). |
| FR-EST-016 | Money MUST enter or leave the sum of agents, pools and escrow only by a named faucet or sink of the ledger registry (2.3). |
| FR-EST-017 | Each tick, in the start phase and after the starting-money draws of the previous tick's entry step, a region MUST release `pool * PoolRelease / 10 000` as relief, truncated, to the agents whose welfare ratio is below `ReliefBelow`, with the reason `relief` (2.6). |
| FR-EST-018 | The relief of an agent MUST be `min(share_i, need_i)` with `need_i = U_i * (ReliefBelow - ratio_i) / 10 000` and `share_i = release * need_i / sum(need)`, with `U_i` the effective upkeep cost of the agent, using integer arithmetic checked for overflow, and what is not paid MUST stay in the pool (2.6). |
| FR-EST-019 | An agent's relief MUST depend only on its own need and the totals, so that the order of agents does not change any payment (2.6). |
| FR-EST-020 | `ReliefPaid` MUST be emitted once per region per tick when anything is paid, with the total paid, the number of recipients and the households (the sum of their weights), and no map (2.6). |
| FR-EST-021 | `EstateOpened` MUST be emitted for every removal of an agent, with the pool policy on or off, accompanying `AgentBankrupted`, with agent, region, balance and the number of lots listed (2.1), and the spawn event MUST carry the starting money received (2.5). |
| FR-EST-022 | A draw from the pool for a weight change MUST NOT exceed the pool's balance, and a weight reduction MUST send the money and the difference `x - floor(x * w' / w)` of each commodity to the estate (2.8.1). |
| FR-EST-023 | `EstateMarkdown`, `PoolRelease` and `ReliefBelow` MUST be region fields with the defaults and bounds of section 3, and a value outside its bound MUST be rejected at construction with `ErrOutOfBounds` (3). |
| FR-EST-024 | The state this spec contributes to a checkpoint MUST be the list of (lot key, current ask price) per region in canonical order for the held lots that have a row, a restore MUST refuse rows out of order or repeated, a price outside 1 to `MaxPrice` and a key that is not a held lot, and a restored world MUST list and price the same lots as an uninterrupted one (2.7.5). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-EST-001 | With the default logger and observer, the estate's listing, repricing and relief in a step of the reference chain MUST allocate nothing beyond scratch held by the world (observability-and-logging.md 4.5, 8.2). |

## 6. Open questions

1. **Gaps found while writing section 5 (behaviour with no clear statement).** (a)
   Whether an external agent (`spec-agents.md` 10) can draw relief: 2.6 says "agents
   living below subsistence" and 2.7.4 excludes only the estate. Decided yes: it is a household like any and counts in the households of its role (`spec-spawner.md` 9.3); a consumer who does not want it keeps the agent's ratio above `ReliefBelow`. (b) Whether relief can go to an
   agent whose `Unfed` is 0 and whose ratio is low only because it is saving. The rule
   reads the ratio, so yes (`spec-agents.md` 11.1). (c) Two lots of one commodity with
   the same key are one lot (`spec-ledger.md` 5.1.3), so "one ask for each lot" is one
   per distinct key.
2. **A rise in weight that the pool cannot fund (2.8.1).** Answered (N-073): the rise is applied partially, in whole households, from a budget of `EntryPoolShare` of the pool, and the unfunded households are not created and are counted (`spec-spawner.md` 4.3, 7.3). A rise gives money and no goods. `spec-population.md` confirms it for births.
3. **The failure ladder and the estate (N-008).** Answered: the ladder is `spec-spawner.md` 5 (rung 1 role change, rung 2 this spec's relief, rung 3 reduction by the unfed share then removal). The rates of births and other deaths are `spec-population.md`'s.
4. **Estate goods for a new agent.** Answered: no. The default spawner grants none (`spec-spawner.md` 4.3); the estate sells, and a grant by a consumer's spawner would be a faucet with its own reason.
