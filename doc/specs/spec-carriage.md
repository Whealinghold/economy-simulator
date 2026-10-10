# Spec: Carriage market and carrier pay

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CAR-NNN` and
`NFR-CAR-NNN`, which tests cite (testing.md section 4). They are in section 6
(FR-CAR-001 to 013, no NFR). Revised after `spec-transport.md` decided N-076
(its section 8): hired carriage is the haul unit of a hired mode, a produced good
traded in an ordinary market. Some requirements moved to the contract and merchant
spec (`spec-merchants.md`) or were superseded; their IDs are kept (section 6). The contract and merchant specs
are `spec-merchants.md`.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

This spec covers how hired carriage is priced and paid: carriers are paid by
merchants, and the fee is the market price of **haul units**, the commodity of a
hired mode of `spec-transport.md`. It is the seed of the contract spec (a bilateral
delivery with escrow, a deadline and failure), which is `spec-merchants.md` 9. Wages and
the target-income rule a carrier follows are in `spec-welfare-and-wages.md`.

## 2. Carriers are paid by merchants

A carrier is a role whose income is the sale of haul units to shippers, typically
merchants. The details of negotiated contracts belong to `spec-merchants.md` 9.
The currency consequences are:

2.1. The fee is the price of haul units bought in an ordinary `trade` (reason
`trade`): money passes from the shipper to the carrier at the trade, and no money is
created or destroyed. There is no escrow and no carrier custody in the market: the
goods travel alone (`spec-transport.md` 5.1) and the units are redeemed at dispatch
(`spec-transport.md` 3.3.3), so no carrier can fail to deliver. Escrow
(`escrow_in`, `escrow_out`, `escrow_refund`, `spec-ledger.md` 6.6) applies to a
bilateral contract with a negotiated fee, which is `spec-merchants.md` 9 (moved to 9.9) and does not go
through the book. A loss of cargo in transit is RSK's (`spec-transport.md` 10.1).

2.2. **Hired carriage is an ordinary commodity.** Decided by N-076
(`spec-transport.md` 8): the carriage unit is a produced good, so the fee is set by
neither a wage table nor negotiation between named parties but by the ordinary
market, cleared by the same `Clearing` as any commodity market, with no new
extension interface (architecture.md 4.3) and no change to `spec-clearing.md`.

- The traded unit is the **haul unit** of a hired mode (`Mode.Haul`,
  `spec-transport.md` 3.1): one haul unit carries one carrying unit's capacity over
  `HaulSpan` distance units. The market is the ordinary market of that commodity in
  the region where the carrier's households live, which is the origin region of the
  shipments (`spec-clearing.md` 2.1; a market is a region and a commodity). A route is
  an edge keyed by its `EdgeID` (`spec-regions.md` 4.7), and the destination is chosen
  at redemption, so the number of haul units a shipment redeems is
  `n * ceil(distance / HaulSpan)` (`spec-transport.md` 3.3.3).
- A load buys `n` carrying units by its binding fill,
  `max(1, ceil(mass / UnitMass), ceil(bulk / UnitBulk))` (`spec-transport.md` 3.2.1),
  and whether a load fits is decided by the pair of mass and bulk capacities
  (`spec-goods.md` 3.2), not by a chargeable size. The chargeable size and
  `BulkFactor` of `spec-goods.md` 3.3 are not used by hired carriage.
- A merchant bids a price per haul unit. Its ceiling is a share of the margin it
  expects per haul unit:
  `bid_max = expected_margin_per_haul_unit * MerchantFeeShare / 10_000`. It never pays
  more than a share of what it expects to earn. The bid comes from the merchant's price
  belief, updated after each cleared or failed bid as for any good, so fees follow the
  same adaptation as prices.
- A carrier makes haul units with an ordinary recipe: feed as inputs, a vehicle durable
  as a tool, and the working ticks of its households (`spec-transport.md` 3.3.4, 17.4).
  Its ask is anchored to that recipe's `UnitCost` (`spec-production.md` 9) by the cost
  band of `spec-beliefs.md` 3.2.2, so transport cost follows the basket (the upkeep and
  feed of the carriers) with no separate transport price. The earlier formula
  `fee_min = trip_ticks * U * (10_000 + m_bp) / 10_000` and its margin `m_bp` are
  deleted.

2.3. A carrier's income is the money it receives for haul units, recorded in the tick of
the trade like any seller's, and its wage is its observed wage as any producer's
(`spec-welfare-and-wages.md` 4.3): a haul recipe is a job, and every tick of a job in
progress is a working tick. The "commercial trip" of that section is a self-carried
trip of `spec-transport.md`, not carriage by a carrier role.

2.4. A carrier uses the target-income rule (`spec-welfare-and-wages.md` section 5): when
its welfare ratio reaches its `TargetRatio`, it stops starting haul jobs, which pushes
the price of haul units up until some carrier's ratio falls below target again.

## 3. Parameters

| Scope | Field          | Meaning                                                     | Default |
|-------|----------------|-------------------------------------------------------------|---------|
| Role  | `MerchantFeeShare` | ceiling on the share of its expected margin a merchant bids per haul unit, in basis points (section 2.2) | 5 000 |

`MerchantFeeShare` has the validation bound 0 to 10 000 (a share of a margin;
0 means the merchant never hires; above 10 000 it would pay more than it
expects to earn). The default and the bound are placeholders (N-015).

The carrier's margin `m_bp` of the earlier text is deleted (section 2.2), which closes
the registry gap it caused (N-016): there is no unnamed parameter.

3.1. **Rounding.** This spec states none, so architecture.md 3.5 applies (truncation
toward zero). `bid_max` truncates. The haul units of a shipment and the carrying units
of a load round up (`spec-transport.md` 3.2.1, 3.3.3). All products are checked.

## 4. Assumptions checked against the specs written since

Checked on 2026-10-10 against `spec-clearing.md`, `spec-beliefs.md`,
`spec-agents.md`, `spec-production.md` and `spec-goods.md`, and settled by
`spec-transport.md` and `spec-regions.md`.

4.1. **What holds.**

- Matching: the `Clearing` works on primitive orders and does not know what the
  market trades (`spec-clearing.md` 8.3), so haul units, an ordinary commodity, are
  cleared by the same extension with no new interface.
- Beliefs: a belief exists for every commodity in the agent's tradable set
  (`spec-beliefs.md` 3.1). A carrier's ask and a merchant's bid use it if the haul
  commodity is in both roles' tradable sets (`Role.Trades`, `spec-agents.md` 3.4). With
  no stock of haul units the target is 0 and the stock component is neutral
  (`spec-beliefs.md` 5.5). The cost anchor of the carrier's ask is the recipe's
  `UnitCost`, so the cost floor keeps the ask at or above 70% of cost (`CostFloor`,
  `spec-beliefs.md` 3.2.2) and the cost ceiling caps it: a spec decision, not a
  surprise.
- The carrier's upkeep is the effective upkeep of one household at the prices of its
  region (`spec-agents.md` 5.2, 5.6.2) and enters the cost of the haul unit through the
  observed wage (`spec-production.md` 9.3).
- The carrier's income is recorded in the tick of the trade
  (`spec-welfare-and-wages.md` 2.1.1) and every tick of its haul job is a working tick
  (`spec-welfare-and-wages.md` 4.3).
- A carrier driven by the default strategy follows the work share of
  `spec-agents.md` 7.1.2, which adds the rest band `RestBand` and the survival rule
  to the "stops at `TargetRatio`" of 2.4. Section 2.4 states the rule without the
  band; the band applies.

4.2. **What the earlier text left open, and where it is settled.**

- **A carriage market is not a (region, commodity) market of goods.** Resolved by
  `spec-transport.md` 8 (N-076): the haul unit is an ordinary commodity made by a
  recipe and sold in an ordinary market, so admission and settlement are
  `spec-clearing.md`'s unchanged, with all its guarantees.
- **The unit may be below the resolution floor.** Resolved in data: the pack chooses the
  carrying unit and the span, and the haul unit of the worked example costs 1 195
  against a floor of 50 (`spec-transport.md` 17.4). The remedy named by
  `spec-goods.md` 2.2, a coarser unit, is the pack author's choice.
- **Round-trip charging with a backhaul** (the README's open item 5, restated in
  section 5).
- **Which commodity, and where.** The market is the ordinary market of the haul
  commodity in the origin region (2.2). A route is an edge keyed by its `EdgeID`
  (`spec-regions.md` 4.7).
- **Carrier cohorts.** A carrier is an ordinary role. A carrier agent of weight `w` has
  quantities `w` times a household's and an unscaled limit price (`spec-scale.md` 4.4);
  the capacity it adds to the market is the output of its recipe, `w` times per
  execution, and no vehicle capacity is attached to its weight (`spec-transport.md`
  3.2.3).

## 5. Open items, restated

These were items 5 and 6 of "Still open" in the README. They wait because each
needs a spec that is not written, not because the answer is unknown.

5.1. **A carrier's reservation fee with a backhaul.** The cost of a haul unit includes
the whole round trip of the producing recipe, which double counts when cargo is found
for the return: each direction's capacity is made by that region's carriers
(`spec-transport.md` 8.4). A pack lever is to state the recipe's `Duration` and output
for one direction only. Whether a return leg is priced once, and whether a bilateral
contract prices it, is restated as open in `spec-transport.md` 20.2. Answered: `spec-merchants.md` 10 (not priced, the double count stands, measured). The effect today is over-charging, which errs towards dearer
carriage, and the loop it feeds is STB-L17. Answered (wave 4): README "Still open" 5 is answered by `spec-merchants.md` 10.

5.2. **How a merchant estimates `expected_margin_per_haul_unit`.** The ceiling
`bid_max` of 2.2 uses the merchant's rumoured remote prices less its local cost. The
rule is answered (README "Still open" 6) by `spec-merchants.md` 5.6, which needs `spec-information.md` for the
rumours (delayed, approximate prices) (`Rumour.Price` and `Rumour.Mean`, `spec-information.md` 5.1), and to `spec-transport.md` for the cost of
the trip (`Travel` and `Routes`, `spec-transport.md` 9). The ceiling is tested with FR-MER-016 (FR-CAR-004).

## 6. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. The
defaults and the bound are placeholders (N-015, N-027). A requirement whose test
needs a spec that is not written says so. After N-076 (`spec-transport.md` 8), a
requirement marked "moved" belongs to `spec-merchants.md` and keeps its ID, and
one marked "superseded" is replaced by the requirement named; neither is tested here.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CAR-001 | **Moved to `spec-merchants.md` 9.9**, restated there under this ID (tests cite it); not redefined here. For the market, the fee is the price of haul units in an ordinary `trade` (2.1). |
| FR-CAR-002 | **Moved to `spec-merchants.md` 9.9**, restated there under this ID (tests cite it); not redefined here. (No escrow exists in the market, 2.1.) |
| FR-CAR-003 | Hired carriage MUST be an ordinary commodity, the haul unit of a hired mode, traded in an ordinary market of the origin region and cleared by the same `Clearing` as any commodity market, and MUST NOT require a new extension interface or a change to `spec-clearing.md` (reworded; `spec-transport.md` FR-TRN-043). |
| FR-CAR-004 | A merchant's bid ceiling per haul unit MUST be `expected_margin_per_haul_unit * MerchantFeeShare / 10 000`, truncated, and the merchant MUST NOT bid above it. Tested with FR-MER-016; the merchant's share is `MerchantFeeShare` spread by agent ID (`spec-merchants.md` 5.7). |
| FR-CAR-005 | **Superseded by FR-TRN-010 and FR-TRN-012.** A carrier's ask is anchored to its recipe's `UnitCost` by the cost band (`spec-beliefs.md` 3.2.2), not to a reservation fee over a chargeable capacity. |
| FR-CAR-006 | **Superseded by the cost anchor** (`spec-production.md` 9, FR-PRO-035 to 037). `fee_min` and the margin `m_bp` are deleted. |
| FR-CAR-007 | **Superseded by FR-TRN-010 and FR-TRN-012.** The traded unit is the haul unit, and a load buys carrying units by its binding fill. |
| FR-CAR-008 | **Moved to FR-TRN-009.** Whether a cargo fits MUST be decided by the pair of mass and bulk capacities (`spec-goods.md` 3.2) and not by its chargeable size. |
| FR-CAR-009 | A carrier's wage MUST be its observed wage as any producer's (`spec-welfare-and-wages.md` 4.3), the haul recipe being a job (reworded: it was `fee / trip_ticks`). |
| FR-CAR-010 | A carrier's income MUST be recorded in the tick of the trade of haul units, and every tick of a haul job in progress MUST be a working tick (reworded: it was the tick the fee is released). |
| FR-CAR-011 | A carrier driven by the default strategy MUST stop starting haul jobs by the work share of `spec-agents.md` 7.1.2, the rest band included. |
| FR-CAR-012 | `MerchantFeeShare` MUST default to 5 000 and lie between 0 and 10 000, and a value outside the bound MUST be rejected with `ErrOutOfBounds`. |
| FR-CAR-013 | Fees follow the same adaptation as prices: a carrier's or merchant's belief for the haul unit MUST be revised once per admitted order after its market clears (`spec-beliefs.md` FR-BEL-007). |

### Non-functional

None. The text of this spec states no non-functional requirement, and none is invented here.

## 7. Entries for the stability register

Merged into `spec-stability.md` section 4 (STB-L16 a carrier's fee loop, STB-L17
carriage cost and the basket, whose `fee_min` form is replaced by the freight of the
haul unit, TRN-A4). The register is their home. Section 5.1 feeds STB-L17: the
over-charge of a backhaul raises the cost of delivered food.
