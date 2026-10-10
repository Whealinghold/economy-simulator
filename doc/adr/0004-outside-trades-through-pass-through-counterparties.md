# 0004. The outside trades through pass-through counterparties

- Status: Accepted (by the supervisor of the wave 3 integration; human review pending)
- Date: 2026-10-10
- Supersedes: none

## Context

The economy has an outside: a market beyond the realm that sells goods the realm
cannot make, buys its surplus and pays or is paid in money. `spec-scale.md` 8 and
`spec-regions.md` 3.5 call a region with supply and demand functions and no
agents the outside world. `spec-money-supply.md` reserved two primitives for it:
a counterparty whose balance is not conserved must be a Boundary account
(10.1), and a top-up plus a sweep can serve as a pass-through account (10.2); its
open question 1 left the choice to the external sector spec.

The choice is how an outside bid or an outside ask is backed. `spec-clearing.md`
2.3 and 3 guarantee that every admitted order is backed by a reservation of cash
or stock, so settlement of a valid trade cannot fail. The ledger cannot reserve
on a Boundary account (`spec-ledger.md` 6.1, `ErrBoundaryReserve`), and money and
goods change only through a transfer with a reason of a closed registry
(`spec-ledger.md` 2.1, 2.3).

## Decision

The outside trades through **pass-through counterparties** (`spec-external-sector.md`
4). The outside is a region of the graph with no agents, marked by a row of
`Modules.External`. For each outside node, gate region (a region with a direct
edge to or from the node) and side (buy or sell), the world has one
**counterparty**, a pseudo-agent bound to an account of a new ledger kind,
**Outside**. In phase 3 the counterparty's account is topped up from a Boundary
account with exactly what its order reserves, and the order is placed through the
ordinary admission; in phase 4, after every market has cleared and every
reservation is released, the account is swept to zero. Six built-in reasons name
the flows: `outside_budget`, `outside_unspent`, `outside_receipts` (money) and
`outside_offer`, `outside_unsold`, `outside_purchases` (goods). The counterparty
holds nothing at a tick boundary.

`spec-clearing.md` changes by one wording (the counterparty is named beside the
estate as a pseudo-agent that places one side only) and no rule. A quote is a
limit, not a peg: trades are priced by the unchanged clearing.

## Consequences

- **Ledger.** A new account kind and an outside sector that is zero at every
  boundary, six registry rows, `trade` allowed between Agent, Estate and Outside
  accounts, and extra audit checks. The boundary flows of the outside are
  **gross** (the budget offered and the part returned); imports, exports and the
  balance of trade are derived measures.
- **Regions.** An outside node has no market and no group; a relocation into it is
  refused (`outside_node`).
- **Estates and agents.** The counterparties take the agent IDs that follow the
  estates (`0..n-1`, equal to the `RegionID`), then the households; adding a node
  or a gate shifts the households' IDs, as adding a region does.
- **Clearing.** None of the guarantees of 2.3 and 3 changes; a custom `Clearing`
  that wants a peg must recognise the counterparties by the ID range.
- **Transport.** A shipment to an outside node is refused (`outside_destination`);
  outside goods appear in the gate region's market and the realm moves them inward.
- **Information.** An outside node is not a rumour source in version 1: INF records
  only a market's published price and a node has no market.

## Alternatives considered

- **A boundary-backed admission.** The counterparty would be a Boundary account and
  clearing would admit its order against a cap. Rejected: the ledger refuses a
  boundary reservation, so it needs a new kind of reservation backed by a cap, a
  settlement that moves money from a boundary as part of a trade (neither internal
  nor a faucet in the registry), and a change of `spec-clearing.md` 2.3, 3.3 and 6
  and of the ledger's 6 and 7. Two specs and their requirements would change.
- **The top-up and sweep of `spec-money-supply.md` 10.2 as the mechanism.** Rejected:
  a flow tops an account up to a fixed level in phase 1 and sweeps in the next
  phase 1, so the account holds money through the whole tick; the level cannot
  follow an order that depends on the price, the window and the caps of each line;
  and it cannot carry goods. The two rules behind it (a Boundary account for what is
  not conserved; a closed registry of declared reasons) are kept.

## Open points

- A peg: trading at the counterparty's own limit needs a flag on `Order` and a rule
  in `spec-clearing.md` 5.1 and 8 (`spec-external-sector.md` 22, question 1).
- The gross flows may deserve a net event or metric (OBS).
- An outside node as a remote information source (INF would need a second record
  source).

## Related

[spec-external-sector.md](../specs/spec-external-sector.md),
[spec-money-supply.md](../specs/spec-money-supply.md) (10, open question 1),
[spec-clearing.md](../specs/spec-clearing.md) (2.3, 3),
[spec-ledger.md](../specs/spec-ledger.md) (3, 6, 7),
[spec-regions.md](../specs/spec-regions.md) (3.5),
[spec-estates.md](../specs/spec-estates.md) (2.7.6),
[ADR 0001](0001-macro-economy-as-replaceable-layers.md).
