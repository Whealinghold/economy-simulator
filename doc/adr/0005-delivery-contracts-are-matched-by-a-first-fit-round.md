# 0005. Delivery contracts are matched by a first-fit round

- Status: Accepted (by the supervisor of the wave 4 integration; human review pending)
- Date: 2026-10-10
- Supersedes: none

## Context

N-076 asked how a carriage fee or a delivery between two regions meets the guarantees of
the order book. `spec-clearing.md` 2.1 identifies a market by a region and a commodity:
both parties are in the region, goods change hands at once, and the money is a
reservation that lasts the tick (`spec-ledger.md` 6.5). A hired shipment moves goods
without their owner (`spec-transport.md` 8), so it needs a buyer in another region who is
committed before the goods leave, a price fixed before they move, payment on arrival from
money that outlives a tick, and a defined outcome when something goes wrong. The book cannot
supply any of these. N-076 offered two options: the carriage unit as a produced good, or a
contract-aware admission and settlement that keeps the guarantees of `spec-clearing.md`
2.3. `spec-transport.md` 8 took the first for the carriage market (the haul unit) and left
the bilateral contract with escrow to MER (N-011, N-100).

`spec-clearing.md` guarantees that every admitted order is backed by a reservation of cash
or stock, that settlement of a valid trade cannot fail, and that markets are independent of
the order in which they clear (2.3, 3). Escrow is not a reservation (`spec-ledger.md` 6.6): it
is a transfer into an Escrow account that outlives the tick.

## Decision

Delivery contracts are matched by a **first-fit round** that runs once per tick in phase 3,
after the last agent's plan and before the estates' asks (`spec-merchants.md` 9.4), and not
by `Clearing`.

- A strategy appends **delivery offers** (goods for delivery to a region, indivisible, with a
  least price) and **delivery takes** (goods wanted in the taker's region, divisible, with a
  most price) to its plan (`spec-merchants.md` 9.2). The round admits or refuses each whole,
  with a `reason`, then matches in each lane (destination region, commodity): for each offer
  in ascending limit, the first take in descending limit whose limit is at least the offer's
  and whose remaining quantity covers it. The price is the midpoint of the two limits, rounded
  half to even. Ties break by a key drawn from the world's source.
- Formation checks and acts in a fixed order, so that a contract exists only if the goods are
  on the way and the payment is committed: the cap on open contracts, the payer's cash, the
  dispatch of the shipment at once (with every check of `spec-transport.md` 5.4), then the
  transfer of the payer's `price * quantity` into an Escrow account (`escrow_in`). The escrow
  cannot fail after the dispatch.
- The shipment arrives at a tick fixed at dispatch, which is the deadline. In phase 1, after
  transport's arrivals, the contract is settled in contract ID order: the deliverer is paid
  from the escrow for the units that reached the destination (`escrow_out`), and the rest is
  returned to the payer or, if the payer was removed, to the estate of the destination
  (`escrow_refund`).
- `Clearing` is unchanged. The contract adds no faucet, no sink and no ledger reason; it uses
  the three escrow reasons already in the registry, whose owner is now MER.

## Consequences

- **Clearing.** No change, and every guarantee of `spec-clearing.md` 2.3 and 3 stands for the
  book. The round has its own admission and its own guarantee of the same kind: everything
  settlement needs is checked and moved before the contract exists.
- **Ledger.** Escrow accounts are reused and never closed, as Transit accounts are; money in
  escrow is out of circulation until arrival (a drain on the payers' liquidity, bounded by the
  take sizing and by `MaxOpen`, register entry MER-A5). The audit checks each open contract's
  escrow exactly.
- **Transport.** A shipment can be requested by the round and is dispatched at once; the
  view gains `Outbound`; a traveller's beliefs are reseeded on arrival.
- **Determinism.** The round draws one key per admitted offer and take, in admission order, a
  function of state; a world with none takes none.
- **Indivisible offers.** An offer is all or nothing because a lot is a whole number of
  carrying units of a shipment and splitting it would double the haul units it redeems. A
  take is divisible. The round is therefore not a double auction.
- **A new matching path.** A second place where trades are made exists beside the book, with
  its own tests (FR-MER-030 to 039).

## Alternatives considered

- **A contract-aware admission and settlement inside `Clearing`** (N-076 option 2). Rejected:
  it touches at least six specs (`spec-clearing.md` 2.1, 3, 6 and 9, `spec-ledger.md` 6,
  beliefs, the tradable set, observability and carriage), needs a new kind of reservation for
  a capacity commitment that is not a ledger quantity, a settlement path that must also be
  total, and a re-proof of the guarantees of 2.3. The contract's buyer is in another region and
  the book is one region.
- **The book alone.** Rejected: a buyer in the destination cannot bid in the origin's book, a
  bid reserves money for one tick only, and nothing commits the buyer before the goods leave
  or defines an outcome for spoilage or a removal. Hired carriage itself stays in the book as
  the haul unit (`spec-transport.md` 8.3); only the delivery of goods to a named buyer needs
  the round.
- **A divisible offer matched as a double auction per lane.** Not taken in version 1: it
  would split a lot between consignees, and a second extension point for matching would be
  needed (`spec-merchants.md` 22, question 8).

## Open points

- Whether a double auction per lane with divisible lots would be the better long-run shape
  (`spec-merchants.md` 22, question 8).
- Forward and standing contracts, a third-party carrier, a bond or a penalty, and credit
  (`spec-merchants.md` 9.10).
- The position of the round in the tick is WLD's to fix (N-009).

## Related

[spec-merchants.md](../specs/spec-merchants.md) (9),
[spec-transport.md](../specs/spec-transport.md) (8),
[spec-carriage.md](../specs/spec-carriage.md) (2.1, 6),
[spec-clearing.md](../specs/spec-clearing.md) (2.3, 3),
[spec-ledger.md](../specs/spec-ledger.md) (3.1, 6.6, 7.2),
[ADR 0001](0001-macro-economy-as-replaceable-layers.md),
[ADR 0004](0004-outside-trades-through-pass-through-counterparties.md).
