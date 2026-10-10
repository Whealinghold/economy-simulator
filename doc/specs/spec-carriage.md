# Spec: Carriage market and carrier pay

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CAR-NNN` and
`NFR-CAR-NNN`, which tests cite (testing.md section 4). They are in section 6
(FR-CAR-001 to 013, no NFR). Several of them depend on the contract,
transport and merchant specs, which are not written: section 4 says which.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

This spec covers how hired carriage is priced and paid: carriers are paid by
merchants, and the fee comes from a carriage market for each directed route. It
is the seed of the contract spec (escrow, delivery, failure), which is not yet
written. Wages and the target-income rule a carrier follows are in
`spec-welfare-and-wages.md`.

## 2. Carriers are paid by merchants

A carrier is a role whose income is fees paid by merchants for hired delivery.
The details of contracts belong to the contract feature spec. The currency
consequences are:

2.1. The fee is a transfer of money from the merchant to the carrier. It is
held in escrow when the contract is accepted and released to the carrier on
delivery. Money is conserved: no money is created or destroyed by a contract.
If the carrier dies in transit, the escrow returns to the merchant.

2.2. **Carriage market.** The fee is set by neither a wage table nor
negotiation between named parties. Hired carriage is traded in a **carriage
market** for each directed route (an edge of the region graph, one direction),
cleared by the same `Clearing` as any commodity market, so no new extension
interface is needed (architecture.md 4.3).

- The traded unit is one chargeable unit of cargo (`spec-goods.md` section 3.3:
  the greater of a good's mass and its bulk converted by `BulkFactor`) carried
  one way along the route. A cargo's chargeable size is the sum of its goods'
  chargeable sizes, so light bulky goods and heavy compact goods are both charged
  for whichever limit they would reach first. Whether a cargo physically fits a
  vehicle is decided by the pair of mass and bulk capacities, not by this
  number (`spec-goods.md` 3.2).
- A merchant bids a price per chargeable unit. Its ceiling is a share of the
  margin it expects per chargeable unit:
  `bid_max = expected_margin_per_unit * MerchantFeeShare / 10_000`. It never
  pays more than a share of what it expects to earn. The bid itself comes from
  the merchant's price belief, updated after each cleared or failed bid as for
  any good, so fees follow the same adaptation as prices.
- A carrier asks no less than its reservation fee, spread over its chargeable
  capacity (`fee_min / capacity` per chargeable unit), where `capacity` is the
  smaller of its mass capacity and its bulk capacity converted by `BulkFactor`
  (`spec-goods.md` 3.4; the vehicle capacities themselves belong to
  `spec-transport.md`):

```
fee_min = trip_ticks * U * (10_000 + m_bp) / 10_000
```

with `trip_ticks` the round-trip travel time, `U` the carrier's upkeep cost per
tick at the prices of its region, and `m_bp` a configured margin in basis
points. This ties transport cost to the basket: longer routes and dearer food
make hired carriage dearer, with no separate transport price.

2.3. A carrier's daily wage for a trip is `fee / trip_ticks`. This is the
value used in the observed wage of `spec-welfare-and-wages.md` section 4.3, and it is comparable to any
other role's observed wage.

2.4. A carrier uses the target-income rule (`spec-welfare-and-wages.md` section 5): when its welfare
ratio reaches its `TargetRatio`, it stops taking contracts, which pushes fees
up until some carrier's ratio falls below target again.

## 3. Parameters

| Scope | Field          | Meaning                                                     | Default |
|-------|----------------|-------------------------------------------------------------|---------|
| Role  | `MerchantFeeShare` | ceiling on the share of its expected margin a merchant bids for carriage, in basis points (section 2.2) | 5 000 |

`MerchantFeeShare` has the validation bound 0 to 10 000 (a share of a margin;
0 means the merchant never hires; above 10 000 it would pay more than it
expects to earn). The default and the bound are placeholders (N-015).

The carrier's margin `m_bp` in section 2.2 is a configured margin in basis
points. It is a parameter of the carrier role, to be named in the contract spec;
until it is named it has no row in the parameter-owner table, no default and no
bound, which is a known gap of the registry (N-016) and the reason FR-CAR-006
cannot be tested yet.

3.1. **Rounding.** This spec states none, so architecture.md 3.5 applies
(truncation toward zero) with one exception that is a labelled choice: the
carrier's ask per chargeable unit, `fee_min / capacity`, is rounded **up**, so
that a vehicle filled to capacity earns at least `fee_min`. `bid_max` and `fee_min`
truncate. All products are checked.

## 4. Assumptions checked against the specs written since

Checked on 2026-10-10 against `spec-clearing.md`, `spec-beliefs.md`,
`spec-agents.md`, `spec-production.md` and `spec-goods.md`. None of the three
newer specs (AGT, PRO, BEL) removes an assumption made here, but the check found
points that the contract, transport and merchant specs must settle.

4.1. **What holds.**

- Matching: the `Clearing` works on primitive orders and does not know what the
  market trades (`spec-clearing.md` 8.3), so a carriage market can be cleared by
  the same extension with no new interface.
- Beliefs: a belief exists for every commodity in the agent's tradable set
  (`spec-beliefs.md` 3.1). A carrier's ask and a merchant's bid can use it if
  the carriage unit is in both roles' tradable sets (`Role.Trades`,
  `spec-agents.md` 3.4). With no stock of carriage the target is 0 and the stock
  component is neutral (`spec-beliefs.md` 5.5). A carrier's `fee_min / capacity`
  passed as the unit cost makes the cost floor keep its ask at or above its
  reservation fee, as 2.2 asks, but the cost ceiling (default three times the
  cost) then also caps the ask: a spec decision, not a surprise.
- The carrier's upkeep `U` of 2.2 is the effective upkeep of one household at the
  prices of the carrier's region (`spec-agents.md` 5.2, 5.6.2).
- The carrier's income is recorded in the tick the fee is released
  (`spec-welfare-and-wages.md` 2.1.1) and every tick of a trip is a working tick
  (`spec-welfare-and-wages.md` 4.3).
- A carrier driven by the default strategy follows the work share of
  `spec-agents.md` 7.1.2, which adds the rest band `RestBand` and the survival rule
  to the "stops at `TargetRatio`" of 2.4. Section 2.4 states the rule without the
  band; the band applies.

4.2. **What does not yet hold, and who must settle it.**

- **A carriage market is not a (region, commodity) market of goods.**
  `spec-clearing.md` 2.1 identifies a market by a region and a commodity, admits an
  ask only for stock the agent has (`insufficient_stock`) and settles a trade by
  moving goods and money (section 6). Hired carriage moves no goods between the
  two parties, and a fee is held in escrow, not paid at once
  (`spec-ledger.md` 6.6, reasons `escrow_in`, `escrow_out`, `escrow_refund`).
  So "cleared by the same `Clearing`" is true of matching only; admission and
  settlement need either a carriage unit that exists as a good (for example
  produced by a trip as a job) or a contract-aware admission and settlement that
  keeps the guarantees of `spec-clearing.md` 2.3 (orders backed, settlement
  cannot fail). The choice belongs to the contract spec with `spec-transport.md`
  and `spec-merchants.md`; this spec does not make it.
- **The unit may be below the resolution floor.** The carriage unit needs a
  reference price like any commodity (`spec-goods.md` 6.3), at or above the
  resolution floor `ceil(10 000 / BeliefStep)`, 50 at the default
  (`spec-currency.md` 3.1). With placeholder numbers: `U` of 1 000 (the default
  `B_r`), a round trip of 2 ticks, `m_bp` of 2 000 and a chargeable capacity of 100
  give `fee_min = 2 * 1 000 * 12 000 / 10 000 = 2 400` and an ask of 24 per
  chargeable unit, below the floor. Such a chain would be rejected. The remedy is
  the one of `spec-goods.md` 2.2: a coarser carriage unit (a unit of several
  chargeable units) or a finer minor unit. The contract spec must pick the unit.
- **Round-trip charging with a backhaul** (the README's open item 5, restated in
  section 5).
- **Which commodity, and where.** Whether the market lives in the origin region
  (where both parties are present, `spec-clearing.md` 3.2 `unknown_agent`), and how
  an order for a route is a (region, commodity) key, are for `spec-regions.md` and
  `spec-transport.md`.
- **Carrier cohorts.** A carrier agent of weight `w` has quantities `w` times a
  household's and an unscaled limit price (`spec-scale.md` 4.4); how a vehicle's
  capacity relates to the weight is `spec-transport.md`'s.

## 5. Open items, restated

These were items 5 and 6 of "Still open" in the README. They wait because each
needs a spec that is not written, not because the answer is unknown.

5.1. **A carrier's reservation fee with a backhaul.** Section 2.2 charges the round
trip (`trip_ticks`) on each leg's ask, which double counts when cargo is found for
the return. Waits for the contract spec: whether a return leg is a second
contract, how a carrier prices a leg it expects to be able to fill, and whether
`trip_ticks` is the round trip or the leg are contract questions that depend on
the contract's lifecycle (escrow, delivery, failure). The effect today is
over-charging, which errs towards dearer carriage, and the loop it feeds is
STB-L17.

5.2. **How a merchant estimates `expected_margin_per_unit`.** The ceiling
`bid_max` of 2.2 uses the merchant's rumoured remote prices less its local cost.
The rule belongs to `spec-merchants.md`, which needs `spec-information.md` for the
rumours (delayed, approximate prices), and to `spec-transport.md` for the cost of
the trip. Until then the number has no source and the ceiling cannot be tested
beyond its formula (FR-CAR-004).

## 6. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. The
defaults and the bound are placeholders (N-015, N-027). A requirement whose test
needs a spec that is not written says so in section 4.2.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CAR-001 | A contract fee MUST be a transfer of money from the merchant to the carrier, held in an escrow account from acceptance and released to the carrier on delivery; no contract MUST create or destroy money. |
| FR-CAR-002 | If the carrier dies in transit, the escrow MUST return to the merchant. |
| FR-CAR-003 | Hired carriage MUST be traded in one carriage market for each directed route, cleared by the same `Clearing` as any commodity market, and MUST NOT require a new extension interface. |
| FR-CAR-004 | A merchant's bid ceiling per chargeable unit MUST be `expected_margin_per_unit * MerchantFeeShare / 10 000`, truncated, and the merchant MUST NOT bid above it. |
| FR-CAR-005 | A carrier's ask per chargeable unit MUST NOT be below `fee_min / capacity` rounded up, where `capacity` is the smaller of its mass capacity and its bulk capacity times `BulkFactor` divided by 10 000 rounded down (`spec-goods.md` 3.4). |
| FR-CAR-006 | `fee_min` MUST be `trip_ticks * U * (10 000 + m_bp) / 10 000`, with `trip_ticks` the round-trip travel time, `U` the carrier's effective upkeep cost per tick at the prices of its region and `m_bp` the carrier role's margin in basis points, using checked arithmetic. |
| FR-CAR-007 | The traded unit MUST be a chargeable unit of cargo, and a cargo's chargeable size MUST be the sum of its goods' chargeable sizes (`spec-goods.md` 3.3). |
| FR-CAR-008 | Whether a cargo fits a vehicle MUST be decided by the pair of mass and bulk capacities (`spec-goods.md` 3.2) and not by its chargeable size. |
| FR-CAR-009 | A carrier's wage for a trip MUST be `fee / trip_ticks`, and the observed wage of `spec-welfare-and-wages.md` MUST use it. |
| FR-CAR-010 | A carrier's income MUST be recorded in the tick the fee is released, and every tick of a delivery trip MUST be a working tick. |
| FR-CAR-011 | A carrier driven by the default strategy MUST stop taking contracts by the work share of `spec-agents.md` 7.1.2, the rest band included. |
| FR-CAR-012 | `MerchantFeeShare` MUST default to 5 000 and lie between 0 and 10 000, and a value outside the bound MUST be rejected with `ErrOutOfBounds`. |
| FR-CAR-013 | Fees follow the same adaptation as prices: a carrier's or merchant's belief for the carriage unit MUST be revised once per admitted carriage order after its market clears (`spec-beliefs.md` FR-BEL-007). |

### Non-functional

None. The text of this spec states no non-functional requirement, and none is invented here.

## 7. Entries for the stability register

Merged into `spec-stability.md` section 4 (STB-L16 a carrier's fee loop, STB-L17
carriage cost and the basket). The register is their home. Section 5.1 feeds
STB-L17: the over-charge of a backhaul raises the cost of delivered food.
