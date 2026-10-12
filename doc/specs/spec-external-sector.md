# Spec: The external sector: the outside world and the money and goods that cross the boundary

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-EXT-NNN` and
`NFR-EXT-NNN`, which tests cite (testing.md section 4). They are in section 21.

Layer: module, off unless `Config.Modules.External` is present
([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md) decisions 1 to 4;
`spec-config.md` 3.2). Version 1 (`vision.md` section 4). Needs `spec-regions.md` (the
region graph: an outside node is a region, its gates are found by edges),
`spec-money-supply.md` (the identities of 3, the rule of 10.1 that a counterparty whose
balance is not conserved is a Boundary account, the declared-reason mechanism of 4.5),
`spec-clearing.md` (admission and reservation, 2.3 and 3, which this spec leaves
unchanged), `spec-ledger.md` (the reason registry 7, boundary accounts 3, the identities
9, the audit 12.4), `spec-calendar.md` (the season index and `TicksPerYear`) and
`spec-config.md` (the module switch, bounds, validation, the hash).

**Every number in this spec is a placeholder.** Nothing has been run in the library,
bazaarBot has not been run, and no acceptance band exists (notes N-015, N-027). No
outside exists by default, so the spec sets no default macro constant. The two models
of section 19 are small integer models of the rules of this spec in Docker Python, not
claims about the library, and no default is claimed to be stable. The example settings
of section 18 are examples only; the core names no setting.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).
"Realm" means the regions that have agents; the **outside** is everything else.

## 1. Purpose and scope

A closed world can only recirculate what it has. A setting usually has an outside: a
neighbouring market that sells goods the realm cannot make, buys the surplus the realm
cannot eat, and pays or is paid in money. Without it a chain with a missing input does
not load, a glut has no buyer, and money can never enter or leave except through the
grants and levies of `spec-money-supply.md`. The roadmap calls EXT "the valve that lets a
setting import, export and gain or lose money". This spec decides:

- what the outside **is**, as data: an **outside node** (a region of the graph with no
  agents), with a **line** per commodity that carries a price path and up to two
  **sides**, one on which the outside sells to the realm and one on which it buys from it
  (sections 3, 6, 7);
- how the outside **trades**: through **counterparties**, pseudo-agents that place
  ordinary orders in the market of a **gate** region, backed by an ordinary account that
  is topped up from a Boundary account before admission and swept back to it after the
  markets clear (section 4). The decision between this, a boundary-backed admission and
  a third way is made there with reasons, and `spec-clearing.md` is left unchanged except
  for one wording;
- the tick: what runs in which phase and in what order (section 5);
- the ledger reasons, the identities, the **balance of trade** as a measure and what
  happens to the realm's money supply when it exports more than it imports (sections 8
  and 11);
- what an agent can see of the outside, and what it cannot (section 9);
- the goods side: how a `Raw` commodity is sourced by an import, how validation sees it
  and the relation to RES and TRN (section 10);
- how the viability check, the warm start and the demand ceiling change from a closed to
  an open economy (section 12);
- parameters, validation, observability, checkpoint state, the entries for the
  stability register, worked examples in two unrelated settings, the notes settled,
  requirements and open questions (sections 13 to 22).

Out of scope: taxes, tariffs and a treasury (`spec-fiscal.md`); credit and foreign debt
(`spec-credit.md`); who ships goods from a gate to the interior and why
(`spec-merchants.md`, MER); the time and cost of transport inside the realm
(`spec-transport.md`); rumours of remote prices (`spec-information.md`, INF); the
resource endowments of the realm (`spec-resources.md`, RES); a policy that opens or
closes trade in response to a measure (`spec-policy.md`); the tick phases
(`spec-world.md`); the public calls that set a shock (`spec-integration.md`).

## 2. Positions taken

The design pressures of the brief, each with a position and the reason. Later sections
give the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is the outside? | A region of the graph with no agents, **marked** by a row of this module (an **outside node**). It has no market, no households and no policy. It is data: lines of price paths and sides of quantities | `spec-scale.md` 8 and `spec-regions.md` 3.5.2 already call a region with supply and demand functions and no agents the outside world, a node; this spec marks it and gives it the functions. One mechanism for places (the graph) |
| 2 | How does it trade? | Through **counterparties**: pseudo-agents, one per (node, gate region, side), that place ordinary orders in the gate region's market. Each order is backed by an account of a new kind, **Outside**, that is topped up with a budget that covers the reservation from a Boundary account just before admission and swept to zero after every market has cleared (4) | Compared with a boundary-backed admission and a standing outside account in 4.1: this one keeps `spec-clearing.md` 2.3 and 3 true with no change of rule, every unit and minor unit moves by a named reason, and nothing is held across a tick |
| 3 | What changes in `spec-clearing.md`? | One wording: the row `side_not_allowed` and 3.1.2 name the counterparty beside the estate as a pseudo-agent that places one side only. No rule changes | The counterparty is an agent from the market's point of view (4.3) |
| 4 | How does money cross the boundary? | By six built-in reasons (8.1): the budget of a buying counterparty, its unspent part, the receipts of a selling counterparty, and the goods forms: offer, unsold, purchases. Money enters the realm only when the outside buys, and leaves only when the outside sells; the net is the **balance of trade** (8.3) | N-107: a trade with an external counterparty is the case for a built-in reason, not a grant or a levy |
| 5 | Fixed prices or drifting prices? | A line has a mid price path: a base, a table by season, a linear trend with a clamp, a bounded mean-reverting walk drawn from the world's source (draw order stated, 6.4), and a shock slot set between ticks (6.5). All integer. A pack with none of the last three has a fixed price | "Fixed or drifting by a deterministic rule from data" (brief); the walk is the only random element and has a stated, fixed draw count |
| 6 | Is a quote a price or a limit? | A **limit**. The counterparty's order carries the quote as its limit price; the trade price is whatever the `Clearing` sets between the two limits (the midpoint by default). The quote is therefore a corridor edge that bites, not a peg (6.7) | A peg needs the clearing rule to know which order is the outside's; that would change `spec-clearing.md` 5.1. Left open (22, question 1) |
| 7 | Can the outside be a pump? | No: on one line the ask is always above the bid by at least one basis point of the mid (validation `ErrCrossedQuotes`, 6.6), so no loop of buying and selling to the same outside earns money, and a counterparty buyer and seller cannot trade with each other | A money pump would be a faucet with no limit |
| 8 | Which regions touch the outside? | The **gates**: regions with a direct edge to or from the node. An edge from the node to a region carries the node's goods in (the outside sells there); an edge from a region to the node carries the realm's goods out (the outside buys there). A closed edge (`spec-regions.md` 7) stops that flow | The graph already says where a place connects and in which direction, and `SetEdgeStatuses` is an embargo for free |
| 9 | Do outside goods ride TRN? | No. They appear in the gate region's market as an ask and leave it as a bid. Moving them inward from the gate is the realm's, by TRN, as for any good. A shipment to an outside node is refused (4.4) | Keeps the outside out of TRN's accounts, consignees and travellers; the cost of crossing is in the quote |
| 10 | How does an agent learn an outside price? | As ordinary orders in the gate market: the trades enter the price window, the unmatched quote is a best unmatched bid or ask in the market report (CLR 5.4, 5.5). No view field. Elsewhere not at all in version 1: an outside node is not a rumour source (9.2) | N-106: no view carries a remote price; EXT never touches INF (INF records only a market's published price, and a node has no market) |
| 11 | What are the limits that stop a silent drain or flood? | Per tick: the line's capacity and its window. Over the run: a cap on units and on list value per side. Every cut is counted in an event. No rule reads the balance of trade (11) | "Macro is measured, never set"; MON-D1 is the model |
| 12 | `Raw` goods | An import line with a sell side makes a commodity sourced for validation, beside a recipe with no inputs and a RES-limited sited recipe. The warning `raw_without_source` is satisfied by it; a commodity sourced only by an import draws `import_only_source` (10) | N-021 |
| 13 | Closed to open | The steady state keeps the closed economy's identity with trade added as bounded columns and a value row of imports not above exports at the quotes; the demand ceiling of `spec-demand.md` 7 does not move; a persistent surplus raises the mean welfare ratio only while the money supply grows (12) | A balanced trade leaves the household identity unchanged |
| 14 | Randomness | One draw per line with a walk, per tick, in phase 1, in ascending (node, commodity) order; the count is fixed by the configuration. None at creation | Determinism (architecture.md 7.3, 7.4) |
| 15 | Region-keyed tables | The node rows and the line rows are keyed by region and commodity, one row per key, in canonical order; the gate scale is a **scale** row of `spec-regions.md` 5.1.2 | N-104 |

## 3. The outside node

### 3.1 A node is a region with no agents, marked by data

3.1.1. `Config.Modules.External` is a pointer; `nil` is off. When off, nothing in this
spec runs, no account of kind Outside is opened, no reason is declared, no state is
added and no event is emitted. Its `Nodes` is a list of rows, each of which names an
existing region by ID (`ErrUnknownRegion` otherwise) and **marks it as an outside node**.
Nothing else in `Config.Regions` changes: the region keeps its place in the graph, its
`RegionID`, its edges, its estate account and pseudo-agent (`spec-regions.md` 3.6.2).
The estate of an outside node exists because the rule of 3.6.1 there makes the
pseudo-agent ID equal to the `RegionID`; it never holds anything and is never used.

3.1.2. **An outside node has no agents.** No `Groups` row, no `WarmStart` group, pool or
wage row, no `Modules.Population.Regions` row, no `Modules.Demand.RegionTastes` row and no
money-flow target may name it (`ErrOutsideNodeInhabited`, naming the table). The world
refuses a relocation into it (4.4); an entry follows the groups, so none can occur. It therefore has no market: no
`MarketCleared` is emitted for it, its price is never published, and it needs no seed
price. The warning `region_without_agents` of `spec-regions.md` 9.2 is not raised for a
marked node.

3.1.3. **More than one node** is allowed (at most `MaxOutsideNodes`). Two nodes are two
independent outsides with their own lines, so an upward-sloping supply or a second buyer
at a lower price is written as a second node on the same gate, at a different quote
(there are no tiers inside a line: one counterparty places one order per commodity,
`spec-clearing.md` 3.1.1).

### 3.2 Gates

3.2.1. A **gate** of a node is a region that has a **direct** edge (`Edge(a, b)` of
`spec-regions.md` 4.1.5, not a shortest path) with the node as one end. The direction
decides the sides that can use it:

| Edge present | Gate for | Meaning |
|--------------|----------|---------|
| node to region | the node's **sell** sides (imports into that region) | goods leave the outside toward the region |
| region to node | the node's **buy** sides (exports from that region) | goods leave the region toward the outside |

A region with both edges is a gate for both. Gates are derived from the configured
edges at construction and never change; whether the edge is **open** is read each tick
from its status (7.5).

3.2.2. **The distance of the edge is not a price.** The quote is a price at the gate,
already delivered. The distance is used only by the warnings of 14.2; this spec does not turn a distance into a cost, because the
cost of crossing is the setting's and a unit such as a league or a velocity change has no
price in the core (`spec-regions.md` 4.1.2).

3.2.3. A gate region may be the gate of several nodes. Each (node, gate, side) has its
own counterparty, so two outsides that sell the same commodity into one region place two
asks there.

3.2.4. **Gate scale.** `Gates` rows, keyed by (node region, gate region), carry an
optional `ScaleBP` (default 10 000) that multiplies both quotes of that gate, with the
rounding of 6.6, so a far gate can be dearer to buy at and poorer to sell at (a **scale**
row of `spec-regions.md` 5.1.2, N-104). A row naming a region that is not a gate of the
node is `ErrNotAGate`.

### 3.3 Lines and sides

A **line** is the outside's dealing in one commodity at one node. It holds one price
path (the **mid**, 6) and up to two **sides**:

```go
// External is the module. Its presence switches it on.
type External struct {
	Nodes []OutsideNode // sorted by Region
}

// OutsideNode marks an existing region as the outside.
type OutsideNode struct {
	Region  string        // an existing region ID with no agents (3.1.2)
	Lines   []OutsideLine // sorted by Commodity
	Gates   []OutsideGate // sorted by Region; optional
	Label   string        // presentation only
}

// OutsideLine is the outside's dealing in one commodity.
type OutsideLine struct {
	Commodity     string
	Base          Price          // the mid at tick 0, 1 to MaxPrice; required
	PriceBySeason []SeasonFactor // optional; absent seasons are 10 000
	TrendBP       int64          // signed basis points of Base per year; default 0
	FloorBP       int64          // lower clamp of the trend and walk factor; default 1 000
	CeilingBP     int64          // upper clamp of the trend and walk factor; default 100 000
	WalkStepBP    int64          // the largest step of the walk per tick; default 0 (no walk)
	WalkRevertBP  int64          // the share of the drift undone per tick; default 0
	WalkBoundBP   int64          // the largest drift either way; default 0
	Sell          OutsideSide    // the outside sells to the realm (imports); Capacity 0 = absent
	Buy           OutsideSide    // the outside buys from the realm (exports); Capacity 0 = absent
}

// OutsideSide is one side of a line.
type OutsideSide struct {
	MarginBP   int64          // sell: the ask above the mid; buy: the bid below the mid
	Capacity   int64          // units offered or wanted per tick at most; 0 = the side is absent
	Window     []SeasonFactor // availability by season in basis points; absent seasons are 10 000
	TotalUnits int64          // cap on the units traded over the run; 0 = none beyond the bound
	TotalValue int64          // cap on the list value traded over the run (7.4); 0 = none beyond the bound
}

// SeasonFactor is a basis-point factor for one season of the calendar.
type SeasonFactor struct {
	Season string
	BP     int64
}

// OutsideGate overrides the scale of one gate of a node.
type OutsideGate struct {
	Region  string
	ScaleBP int64 // default 10 000
}
```

3.3.1. In a file a node is the table `[module.external.node.<region>]`, a line the table
`[module.external.node.<region>.line.<commodity>]` with its sides as the sub-tables `sell`
and `buy`, and a gate `[module.external.node.<region>.gate.<region>]`, so documents merge
by key (`spec-config.md` 6.4); the layout is proposed for 6.2 of that spec.

3.3.2. **Canonical order.** Nodes are sorted by region (as `RegionID`), lines by commodity
(as `CommodityID`), gates by region, season factors by season index, before a world is
built or a hash is taken (`spec-config.md` 3.4; `spec-regions.md` 5.3.4). A second row for
a key is `ErrDuplicateID`.

3.3.3. **A side with `Capacity` 0 is absent**, and so are its other fields: any of them
non-zero on an absent side is `ErrOutOfBounds` (a margin with no side is a mistake). A
line with both sides absent draws the warning `line_without_side`.

### 3.4 What a node is not

It is not a market (3.1.2), not a household, not an agent and not an account: no money or
goods are ever held "at" the node. It is not a policy: it has no rule that reads a
balance, a price of the realm or an aggregate (11.3). It is not a source of information
unless INF reads it (9.2). It is not a place for agents to go: TRN refuses it as a
destination and REG refuses a relocation into it (4.4).

## 4. How the outside trades

### 4.1 The decision

The question left open by `spec-money-supply.md` (open question 1) and `spec-clearing.md`
(every order is backed, 2.3 and 3) is how an outside bid or ask is backed. Three ways were
compared.

| Option | What it is | Verdict and reason |
|--------|-----------|--------------------|
| **(a) Boundary-backed admission** | the counterparty is a Boundary account; CLR admits its order against a cap, and settlement moves goods or money to or from the boundary under a reason | **Rejected.** The ledger refuses a boundary reservation (`spec-ledger.md` 6.1, `ErrBoundaryReserve`). It would need (i) a new kind of reservation backed by a cap and not by a balance, (ii) a settlement that moves money from a boundary to a non-boundary account as part of a `trade`, which is neither internal nor a faucet in the registry, (iii) a change of `spec-clearing.md` 3.3, of the guarantee in 2.3 ("every order is backed" becomes "backed by a balance or a cap") and of 6, and of the ledger's 6 and 7. Two specs and their requirements change, for a mechanism whose only gain is the absence of a gross top-up |
| **(b) A pass-through account** | the counterparty has an ordinary account of a new kind; just before admission it is topped up from a boundary with a budget that covers what the order reserves (`limit * quantity`; a legal price can only lower the reservation, `spec-institutions.md` 6.3.3); after every market has cleared it is swept to zero | **Chosen.** The order is an ordinary order backed by a real reservation; settlement is an ordinary `trade` transfer; the top-up and the sweep are transfers with named reasons (8.1). `spec-clearing.md` 2.3 and 3 hold with no change of rule (4.3). The cost: the boundary flows are gross (the budget offered, the part returned), so the net is a derived measure (8.3) |
| **(c) MON's top-up and sweep as flows** | the counterparty is a class of accounts topped up to a level by a top-up grant in phase 1 and swept by a rate levy of 10 000 basis points (`spec-money-supply.md` 10.2) | **Rejected as the mechanism, kept as the model.** A flow tops up to a fixed level per tick in phase 1, and the sweep runs in the next tick's phase 1: the account would hold money through the whole tick between, and the level cannot follow the order, which depends on the price, the window and the caps of each line. It cannot carry goods. (b) uses the same two rules, a Boundary account for what is not conserved (`spec-money-supply.md` 10.1, FR-MON-043) and a closed registry of declared reasons, in a form made for an order |

The answer to MON's open question 1 is therefore (b): a pass-through account owned by this
spec, with built-in reasons. The target kind `external` that MON reserved stays unused and
MON may drop the reservation (handoff, section h).

### 4.2 The counterparty

4.2.1. A **counterparty** is a pseudo-agent bound to one ledger account of the kind
**Outside**. There is one for each (node, gate, side) for which the node has at least one
line with that side present and the gate is a gate for that side (3.2.1). A counterparty
is a **buyer** (it places bids, for the buy sides) or a **seller** (asks, for the sell
sides). It places one order for each line of its node whose side is present, one per
commodity, so it never trades with itself (`spec-clearing.md` 3.1.1). The seller and the
buyer of one (node, gate) are different agents with different accounts, so they place an
ask and a bid for the same commodity without a conflict, and 6.6 guarantees they cannot
meet.

4.2.2. **Agent IDs.** The estate pseudo-agents take the lowest agent IDs, one per region
(`spec-estates.md` 2.7.6, FR-AGT-001). The counterparties follow them: the next IDs, in
ascending order of (node `RegionID`, gate `RegionID`, side with buy before sell). The
households' IDs follow. The counterparties are not agents or households (no weight, no
role, no strategy, no `AgentSpawned`), as the estate is not (`spec-estates.md` 2.7.4).

4.2.3. **The account.** Its kind is **Outside**: a non-boundary account that holds money
and goods and may not go negative, whose region is the gate region, created at
construction after the estate accounts and before any agent account, in agent-ID order
(`spec-regions.md` 3.6.1; the order against the Boundary accounts is the ledger's). It
never closes and never relocates (`spec-regions.md` 6.1.3, `not_relocatable`). Its sector
is the **outside** sector, whose money and goods are zero at every tick boundary (5.3),
so it adds nothing to `M_h`, `M_p` or `M_c` (`spec-money-supply.md` 3.1.1) and the
sectoral identity of `spec-ledger.md` 9.3 holds with the outside sector at zero.

4.2.4. **The Boundary accounts** of the six reasons of 8.1 are created at construction:
the money reasons first, then the goods reasons for each commodity that has a line side,
in commodity order, after the boundary accounts of the built-in reasons and of the
declared money flows and before any other account (`spec-ledger.md` 3.4; `spec-money-supply.md`
4.5.1).

### 4.3 The guarantees of spec-clearing 2.3 and 3 hold unchanged

For one order of a counterparty, `spec-clearing.md` 3.2 checks, in order:

| Check | Why it passes |
|-------|---------------|
| `unknown_agent` | the counterparty exists and its account is in the gate region, the market's region |
| `side_not_allowed` | a buyer places only bids and a seller only asks (a wording addition, 4.4) |
| `unknown_commodity` | the line's commodity is validated |
| `not_tradable` | does not apply: a pseudo-agent has no role (`spec-clearing.md` 3.2) |
| `not_permitted` | does not apply: a pseudo-agent holds no membership of a body, and the check lets it pass (`spec-clearing.md` 3.2, `spec-institutions.md` 5.6.2) |
| `invalid_price` | the quote is clamped to `[1, MaxPrice]` (6.6) |
| `invalid_quantity` | the quantity offered is at least 1 (a side that offers nothing places no order) and at most `Capacity`, which is at most `MaxQuantity` (13.1) |
| `duplicate_order` | one order per commodity per counterparty |
| `overflow` | price times quantity is at most `MaxPrice * MaxQuantity` = 10^18, which fits (`spec-config.md` 4.1) |
| `insufficient_funds` | a buyer is topped up with a budget of `limit * quantity` before admission (5.2), which is at least what the order reserves (a legal price can lower the reservation, never raise it) |
| `insufficient_stock` | a seller is topped up with exactly `quantity` units before admission |

So every order is admitted, and every admitted order is backed by a reservation in a real
account (`spec-clearing.md` 3.3). The reservation of a bid is at most `limit * quantity` (a legal price may lower it), and a trade
price never exceeds the limit, so settlement cannot fail (2.3). The part of the budget
that the price improvement leaves goes back to the buyer's available cash at settlement
(`spec-clearing.md` 6) and is swept at the end of phase 4. When the markets have finished,
every reservation is released (`spec-clearing.md` 6, `spec-ledger.md` 6.5) and **only then**
are the accounts swept, so no sweep ever meets a reservation. If an order of a counterparty
were ever rejected (a programmer error, not a configured case) the top-up would sit in the
account and the sweep would still return it: the accounts are empty at the tick boundary
whatever happened. `spec-clearing.md` 8.2 (the world validates the whole list of trades) is
unchanged: the buyer and the seller of a trade are different agents because a buyer
counterparty has no ask and a seller no bid.

### 4.4 What changes in other specs

| Spec | Change | Kind |
|------|--------|------|
| `spec-clearing.md` 3.1.2 and the `side_not_allowed` row of 3.2 | name the counterparty beside the estate: a pseudo-agent that may place one side only | wording, no rule |
| `spec-ledger.md` 3.1, 7.2, 9.3 | the kind Outside; the six reasons; `trade` between an Agent, Estate or Outside account; the outside sector, zero at a tick boundary | additions |
| `spec-regions.md` 6.1.3, 3.5.2, 9.2 | a relocation into an outside node is refused, a new row `outside_node` with `ErrOutsideNode` (an Outside account is already refused by `not_relocatable`, because it is not of kind Agent); `region_without_agents` is not raised for a node | additions |
| `spec-transport.md` 5.4 | a new refusal `outside_destination` between checks 6 and 7: the destination is an outside node; `Routes` reports no route to it | addition |
| `spec-estates.md` 2.7.6, `spec-agents.md` FR-AGT-001 | the counterparties follow the estates in the agent ID space | addition |
| `spec-money-supply.md` 3.1.1, 6.4.2, 10.2, 21 | the outside sector is zero at a boundary; the supply bound counts the outside; the target kind `external` is released; open question 1 is answered | edits |
| `spec-viability.md` 4.2.1, 4.6 and the programme | an import is a source; the open layer of 12.1 | addition |
| `spec-config.md` 3.7, 4.1, 6.2, 7.5 | the catalogue, the bounds, the layout, the source rule | additions |

The full list, with exact text, is in the handoff (section h).

## 5. The tick

The phase numbers are those of `spec-agents.md` 4, which `spec-world.md` will fix
(N-009). This spec asks for three positions.

### 5.1 Phase 1: the price state

5.1.1. In phase 1, after the money flows (`spec-money-supply.md` 5.1) and before the
directive, for each line with `WalkStepBP` above 0, in ascending (node `RegionID`,
`CommodityID`), the world draws once from its source and advances the drift (6.4). No
other draw is made by this module, at creation or later, so the count of draws in a tick
is the number of walking lines and does not depend on state (FR-EXT-021).

5.1.2. The shock slot (6.5) is world state set between steps; phase 1 reads it and does
not change it.

### 5.2 Phase 3: the orders

5.2.1. After the last agent's plan has been executed and the estate pseudo-agents have
placed their asks, the counterparties place their orders, in ascending agent ID
(4.2.2). For one counterparty, for each line of its node in commodity order whose side
is present, the following are computed from the tick's moment (season), the line's state
and the edge's status. They are a pure function of those (FR-EXT-023); no price,
inventory or balance of the realm is read.

1. **Wanted.** `want = floor((Capacity * Avail + carry) / 10 000)`; the carry advances (7.1).
2. **Open?** The edge from the node to the gate (a seller) or from the gate to the node (a
   buyer) is open. If it is closed, nothing is offered and the wanted units are counted
   as `CutClosed` (7.5).
3. **The limit.** The quote of the side at this gate (6.6). If it is below 1 (a bid whose
   rounding reaches 0) the side places nothing and the units are counted as `CutPrice`.
4. **Offered.** The wanted units less the cuts of the caps (7.3), which use the limit:
   `offered`, at least 0.
5. **Top-up.** If `offered` is at least 1: a buyer's account receives `limit * offered`
   from the boundary of `outside_budget`; a seller's account receives `offered` units of
   the commodity, born this tick and at full durability, from the boundary of
   `outside_offer`.
6. **The order.** The order is placed through the ordinary path, one for the (counterparty,
   commodity), and admitted by the checks of 4.3.

5.2.2. The admission order of the counterparties' orders is the last in the market's input
order (`spec-clearing.md` 4.2): estates first, then counterparties by agent ID. The
tie-break keys are drawn for every order as for any order, so the counterparties' orders
take their share of the clearing's draws.

5.2.3. **Why a top-up each tick.** An unfilled part of a bid or an ask is released when its
market finishes and is swept at once (5.3); nothing the outside brought stays in the
realm's ledger overnight. This is what makes the boundary flows gross (8.1). The top-up of a buyer is a **budget** computed from the quote; a legal price (`spec-institutions.md` 6.3.3) may lower the limit the order reserves, and what it does not reserve is swept back unspent at P4.2 under `outside_unspent`.

### 5.3 Phase 4: the sweep

5.3.1. When every market has cleared and every reservation has been settled or released
(`spec-clearing.md` 6, 7.2), before phase 5, for each counterparty in agent-ID order, its
account is emptied to the boundary, money then goods in commodity order:

| Account | What it holds after the markets | Moved to the boundary under |
|---------|--------------------------------|------------------------------|
| buyer, money | the budget less what it paid | `outside_unspent` |
| buyer, goods | what it bought | `outside_purchases` |
| seller, goods | what was not bought | `outside_unsold` |
| seller, money | what the realm paid | `outside_receipts` |

A quantity of 0 is not moved. A goods lot is moved whole with its born tick, so spoilage
and wear of a lot do not matter here: the goods leave the same tick they arrive.

5.3.2. After the sweep each side's state is updated: the cumulative units and list value
(7.4) and the ended flag (7.3). Then the events of 15.1 are emitted (architecture.md
12.5; observability-and-logging.md 4.4).

5.3.3. **Atomicity.** Each transfer is atomic (`spec-ledger.md` 2.6). A checkpoint is taken
between ticks, when every counterparty account is empty, so it never sees an order half
settled (ADR 0002 decision 4; FR-EXT-016).

## 6. Prices

### 6.1 The mid

Each tick the **mid** of a line, an integer price in minor units, is

```
F_trend = max(0, 10 000 + trunc(TrendBP * t / TicksPerYear))
F_walk  = 10 000 + Drift
F       = clamp(round_half_even(F_trend * F_walk / 10 000), FloorBP, CeilingBP)
S       = PriceBySeason[season(t)], or 10 000 for an absent season or no season
Mid     = clamp(round_half_even(Base * S * F * Shock / 10^12), 1, MaxPrice)
```

where `t` is the tick, `Shock` the shock slot (6.5, default 10 000) and `Drift` the walk
state (6.4, 0 without a walk). The product is formed with the checked 128-bit helper of
`spec-currency.md` 7: its largest value is `10^9 * 10^5 * 10^6 * 10^5 = 10^25`, below
`2^127`. `trunc` is division truncating toward zero and `round_half_even` rounds a
non-negative quotient half to even, the rule of `spec-clearing.md` 5.2. The mid is computed
when it is needed, from the tick, the drift and the shock; it is not stored (16).

6.1.1. A line with no season table, `TrendBP` 0, no walk and the shock at its default has
`Mid = Base` for all time: a **fixed** price. The first consumer's cheapest setting is
this one.

### 6.2 Seasons

The season of a tick is `Calendar.At(tick).Season`, an index, or -1 for no season
(`spec-calendar.md` 4.5). A table is keyed by season ID; no logic names a season
(architecture.md 9.1). An unknown season ID is `ErrUnknownSeason`, and so is any season ID
when the calendar has no seasons (`spec-calendar.md` 4.5 gives -1 and no names).
The same table form is used for the **window** of a side (7.2).

### 6.3 The trend and the clamp

`TrendBP` is a signed number of basis points of `Base` per year (`TicksPerYear` ticks): a
rising price for a positive value. It is linear, not compounding. The factor `F` is held
between `FloorBP` and `CeilingBP` (defaults 1 000 and 100 000: a tenth and ten times
`Base`), so a long run cannot take the price to 0 or to the bound. A tick in which the
clamp changes `F` or the mid is flagged `PriceClamped` in the event of 15.1, and the
clamp is the damper EXT-D5 (17).

### 6.4 The walk

6.4.1. A line with `WalkStepBP` above 0 has a **drift** `D`, an integer in
`[-WalkBoundBP, WalkBoundBP]` that starts at 0 and is part of the saved state. In phase 1
of each tick it moves:

```
d  = -WalkStepBP + draw(2 * WalkStepBP + 1)             (one draw, in [-step, step])
D' = clamp(D - trunc(D * WalkRevertBP / 10 000) + d, -WalkBoundBP, WalkBoundBP)
```

The draw is the world's draw of `spec-world.md` 7.2.1 (the high word of one source
word times the bound: one word, no loop, no rejection; architecture.md 7.3), taken in
the order of 5.1.1; the extension contract of architecture.md 7.7 does not apply because
the world itself draws. `WalkBoundBP` must be at least `WalkStepBP` and at most 9 999, so
`F_walk` stays positive.

6.4.2. **What it does.** The drift is a bounded, mean-reverting random walk (an
autoregression of order 1 with coefficient `1 - r`, `r = WalkRevertBP / 10 000`, and a
uniform step). Its stationary variance, before the bound, is `s(s + 1) / 3 / (1 - (1 - r)^2)`
for step `s` in basis points (the variance of an integer uniform on `[-s, s]` is
`s(s + 1) / 3`). For `s` 100 and `r` 5% it is about 34 500 and the standard deviation about
186 basis points; the model of 19.1 gives a range of -568 to 395 over 720 ticks, which is
consistent with it. With `WalkRevertBP` 0 the walk is free and reaches the bound in about
`(WalkBoundBP / WalkStepBP)^2 * 3` ticks. This is a derivation and a model, not a result
for the library.

6.4.3. **Against the realm's beliefs.** A belief moves by at most `2 * BeliefStep` of its
centre per adjustment and adjusts with probability `BeliefAdjustProb` (`spec-beliefs.md`
5.2, 5.6, 11). A mid that moves per tick by more than the beliefs can follow leaves the
realm trailing a moving anchor; the warning `walk_faster_than_beliefs` fires when
`WalkStepBP + ceil(|TrendBP| / TicksPerYear)` exceeds `BeliefStep` (14.2). It is a
heuristic: the drift per tick is compared with one step, and the condition for the
beliefs to keep up is not derived (EXT-A6).

### 6.5 The shock slot

The **shock** of a line is a factor in basis points, default 10 000, bound 1 to
`MaxShockBP` (100 000), that multiplies the mid (6.1). It is world state, not
configuration: it is changed only between steps by the world's operation
`SetOutsideShocks([]OutsideShock{Node, Commodity; BP})` (`spec-integration.md` 5.1;
N-091), applies from the next step, is saved in a checkpoint and is not part of the
configuration hash. The operation validates, in this order (`spec-integration.md`
5.1.3), the module (`ErrModuleOff`), the node (`ErrUnknownRegion`), the commodity
(`ErrUnknownCommodity`), the line (`ErrUnknownLine`, new: the node has no line for the
commodity), the bound (`ErrOutOfBounds`) and that no key repeats (`ErrDuplicateID`), and
emits `OutsideShocked` for each changed (node, commodity).
The engine never changes it. It is the means by which a consumer, a script or a future
policy actor moves an outside price (a supplier abroad fails, a rival market opens) without
a rule of the core reading anything. A closed edge (7.5) is the means to cut a flow.

### 6.6 The quotes

For a line with mid `m`:

```
ask  = ceil(m * (10 000 + MarginBP_sell) / 10 000)       (the outside sells)
bid  = floor(m * (10 000 - MarginBP_buy) / 10 000)       (the outside buys)
ask_g = ceil(ask * ScaleBP_g / 10 000),  bid_g = floor(bid * ScaleBP_g / 10 000)
```

then each is clamped to at most `MaxPrice`. A bid below 1 means the side places nothing
that tick (`CutPrice`). The rounding widens the spread, against the realm, so it can never
close it. The limit price of the counterparty's order is `ask_g` or `bid_g`.

6.6.1. **No pump.** Where both sides of a line are present, `MarginBP_sell + MarginBP_buy`
must be at least `MinQuoteSpreadBP` (1), else `ErrCrossedQuotes`. Then for every mid
`m >= 1`, `bid < m <= ask` or `bid <= m - 1 < m <= ask`, so `bid < ask`; the gate scale
keeps it, because `floor(x * S) <= x * S < y * S <= ceil(y * S)` for `x < y`; and the clamp
at `MaxPrice` keeps it, because a bid is below the mid and the mid is at most `MaxPrice`.
Hence a buyer and a seller counterparty of one gate never cross (`spec-clearing.md` 4.3),
and an agent that buys from the outside at its ask and sells to it at its bid at the same
gate loses at least the spread: no loop of trades with the outside earns money (FR-EXT-009).
Across two gates the scale can open a gap; that is an arbitrage that needs a carrier, and it
is warned about at construction (`outside_arbitrage`, 14.2) and measured.

6.6.2. **Against the realm's reference band.** A quote far outside the band of reference
prices of the gate (`spec-beliefs.md` 3.2.1, `RefBandLow` and `RefBandHigh` of the effective
reference price) can never be reached by a realm belief, which is clamped to the band, so
the side never trades or always does; the warning `quote_outside_reference_band` says so at
construction, for `Base` with the margin and the scale of each gate.

### 6.7 A quote is a limit, not a peg

The counterparty's order is an ordinary order, so the trade price is set by the `Clearing`
between the two limits: by default the midpoint (`spec-clearing.md` 5.1). A realm buyer with
a limit of 50 against an outside ask of 42 pays 46, not 42; a realm seller with an ask of 15
against an outside bid of 18 receives 16.5, rounded half to even to 16 (worked in 18.1). The
consequences, stated plainly:

- The quote bounds only what the outside will accept. The realm's price is **not** held at
  the quote: a realm buyer pays above the ask while its own belief is above it.
- The realm's beliefs are pulled toward the quote over ticks: a filled bid lowers its
  centre, an unfilled ask is evidence to lower it (`spec-beliefs.md` 5.4, 5.5), and the
  cheaper order is matched first. The speed and the size of the pull are unknown (EXT-S1).
- The outside gains the price improvement: a sell line earns at least its ask, a buy line
  pays at most its bid. The model of 19.2 shows the cost to the realm of cycling between the
  two quotes (EXT-A5).

A peg, a trade at the counterparty's own limit, would need the `Clearing` to know which
order is a counterparty's; `spec-clearing.md` 5.1 and 8 would change. It is the first open
question (22), and a consumer who needs it today supplies a `Clearing` that prices a pair
involving a counterparty at the counterparty's limit (the world checks only that the price
lies between the two limits, `spec-clearing.md` 8.2) and recognises the counterparties by the
ID range of 4.2.2.

## 7. Quantities, windows and limits

### 7.1 Capacity and its carry

`Capacity` is the most units a side offers or wants in a tick at full availability. The
wanted units of a tick are `floor((Capacity * Avail + carry) / 10 000)` with `Avail` the
window factor (7.2) and `carry` an integer in `[0, 10 000)` kept per side, starting at 0
and set to the remainder. A fraction of a unit is therefore not lost: a capacity of 3 at an
availability of 2 500 wants 0, 1, 1, 1, 0, 1, 1, 1 ... units, 3 in 4 ticks. The carry
advances by the wanted units whether or not they were offered, so a cut is forfeited, not
deferred (the rule of `spec-money-supply.md` 6.1.2). No draw is used. `Capacity * Avail` is
at most `10^9 * 10^4`.

### 7.2 The window

The **window** of a side is a table of availability in basis points by season, 0 to 10 000,
with absent seasons at 10 000. A season at 0 is a closed season: the side places no order
and nothing is counted as a cut, because the closure is the configuration's. It is how a
setting says the outside buys a good only in the season after its production, or sells nothing in
a season in which it cannot be reached (an availability window). A window that is 0 in every season
draws the warning `window_all_closed`.

### 7.3 Cumulative caps

A side may carry `TotalUnits` and `TotalValue`, caps over the run on the units traded and on
the **list value** traded (7.4). The effective list-value cap is the smaller of `TotalValue`
(when set) and `MaxLineTotal`, a bound of 13.2 that makes all the lines together able to move
at most `MaxMoney`; the effective unit cap is `TotalUnits` when set. Each tick the offer is
cut to the remaining units and to `floor(remaining list value / limit price)`, and the units
cut are counted as `CutCap`. When, after settlement, the remaining units are 0, or the
remaining list value is below the limit price of the tick, the side **ends** for good:
`OutsideEnded` is emitted once with the cause (`units` or `value`) and a `Warn` is logged
once. An ended side places nothing and draws no counterparty top-up. A price that later falls
does not revive it, so up to one quote price of the list value may stay unused. The cap is the
damper of a finite outside (EXT-D2), the model being MON-D1.

### 7.4 List value

The **list value** of a tick's trades on a side is the units traded times the limit price of
the counterparty's order, not the money that changed hands. On a buy side the realised money
is at most the list value, since a trade price does not exceed the bid; on a sell side it is
at least the list value (the midpoint favours the outside). The cap is therefore a bound on
what the outside can pay the realm, and an approximate one on what the realm can pay the
outside; the realised money is measured exactly by the reasons (8.1). The model of 19.1
shows a sell line with a list value of 11 988 against a cap of 12 000 that moved 13 394
minor units.

### 7.5 A closed edge

If the edge that a side needs (3.2.1) has the status closed (`spec-regions.md` 7), the side
places nothing, and the wanted units are counted as `CutClosed`. An edge's length factor has
no effect in version 1: the price at a gate is not a function of distance (open question 3).
The status is set between ticks by `SetEdgeStatuses` (`spec-regions.md` 7.3; INT), so an
embargo is a call, and the register's L66 (a region that cannot be reached) is its hazard
(EXT-A7).

### 7.6 Every cut is counted

There is no silent clamp. The units that a side wanted and did not offer are split into
`CutClosed` (7.5), `CutCap` (7.3) and `CutPrice` (6.6); a clamp of the price is `PriceClamped`
(6.3); an ended side is `OutsideEnded`. All are in `OutsideTraded` (15.1), the aggregate that
FR-STB-013 asks for.

## 8. Money and goods across the boundary

### 8.1 The six reasons

This spec adds six rows to the registry of `spec-ledger.md` 7.2, built in and declared only
when the module is on. They are not grants or levies (N-107): the amounts follow an order that
the market fills, and the account that carries them is the outside's counterparty.

| Reason | Of | Kind | Between | Meaning |
|--------|----|------|---------|---------|
| `outside_budget` | money | faucet | boundary to a buyer counterparty | the money the outside brings to bid for the realm's goods this tick (gross) |
| `outside_unspent` | money | sink | a buyer counterparty to boundary | the part of the budget not paid (unmatched bids, price improvement) |
| `outside_receipts` | money | sink | a seller counterparty to boundary | the money the realm paid for what the outside sold |
| `outside_offer` | goods | faucet | boundary to a seller counterparty | the units the outside brings to sell this tick (gross) |
| `outside_unsold` | goods | sink | a seller counterparty to boundary | the units not bought |
| `outside_purchases` | goods | sink | a buyer counterparty to boundary | the units the realm sold to the outside |

`trade` (`spec-clearing.md` 6) also moves between an Agent, an Estate and an Outside account
(the row of the registry is edited). The boundary flows of a tick are totalled per region of
the non-boundary side, which is the gate, so `BoundaryFlow` reports each reason per gate
(`spec-ledger.md` 7.4).

### 8.2 The identities

For every tick, exactly (FR-EXT-019, FR-EXT-020):

1. Every counterparty account holds no money and no goods at the tick boundary.
2. **Imports and exports in units.** For a commodity `c`, the units that the realm bought
   from the outside are `import_c = outside_offer_c - outside_unsold_c`, and the units it
   sold are `export_c = outside_purchases_c`. Each equals the sum of the units of the
   counterparty trades of that commodity.
3. **Imports and exports in money.** `export_value = outside_budget - outside_unspent` and
   `import_value = outside_receipts`; they are the money that the counterparties paid and
   received in trades.
4. **The supply.** `M_t = M_init + grants - levies + (other faucets - other sinks) +
   sum over ticks (export_value - import_value)`: the money supply of
   `spec-money-supply.md` 3.1.2 with the outside's net as its one new term.
5. **Goods.** For `c`, the change of the total over non-boundary accounts is its recipe
   outputs less its inputs, consumption, spoilage, wear, disposal, haul use and feed, plus
   `import_c - export_c`; goods are conserved except by recipes, consumption, spoilage and the
   other named flows and by these two (testing.md 6.3).

### 8.3 The balance of trade

The **balance of trade** of a node over a period is `export_value - import_value`: the money
the outside added to the realm (positive) or took from it (negative). It is a **measure**: it
is measured by `TradeBalance` of `spec-indicators.md` 9 row 13 (gross and net, units and
money, by node, gate and commodity, from `OutsideTraded` and `BoundaryFlow`), and not by a
Snapshot field; nothing in the engine reads it (11.3). The boundary accounts are per reason
and kind of thing (`spec-ledger.md` 3.4) and the money of a reason is attributed to the
gate region, so the per-gate figure comes from `BoundaryFlow` and `OutsideTraded`; per node
it is the sum over the node's gates, and per commodity for goods.
The cumulative balance over the run equals `M_t - M_init` less the contributions of the
other faucets and sinks (8.2, item 4).

### 8.4 The sectoral identity

The counterparty accounts are empty at every boundary (8.2, item 1), so the identity of
`spec-ledger.md` 9.3, `delta(households) + delta(public) + delta(contract) = faucets - sinks`,
holds per tick with the six reasons among the faucets and sinks and the outside sector at
zero. Money that the outside pays the realm raises the households' sector (the sellers are
households or the estate); money the realm pays the outside lowers it.

### 8.5 The audit

`World.Audit` and `WithAudit` (`spec-ledger.md` 12.4) also check, for the module, that every
counterparty account is empty, that the signs of the six boundary balances are those of their
kinds (a faucet's balance is not positive, a sink's is not negative), and that for every
commodity the boundary balance of `outside_purchases` equals the sum of the `Units` of its
buy sides and `outside_offer` less `outside_unsold` equals the sum of the `Units` of its sell
sides, with exact equality (FR-EXT-050). A violation is reported with the node and commodity.

### 8.6 Construction

Nothing of the module runs during construction. The construction bucket
(`spec-ledger.md` 7.4.1) contains no outside reason, and the counterparty accounts open
empty. A warm start gives them nothing (12.1).

## 9. What an agent sees of the outside

9.1. **In a gate region, as ordinary orders.** The counterparty's order is placed by the
ordinary path, so `OrderPlaced` is emitted for it (its agent is a counterparty, marked as a
pseudo-agent in the Snapshot); the trades enter the price window of the market
(`spec-clearing.md` 5.4); an unfilled ask or bid is the best unmatched quote of the market
report (5.5) that the strategy's `Revise` and the beliefs see (`spec-beliefs.md` 5.4). So a
realm agent learns the outside price in the gate region by trading there, as it learns any
price. No view field is added: the brief's alternative (a field EXT defines) is not needed.

9.2. **Elsewhere, not at all in version 1.** An agent outside a gate region is shown no outside
price: no view carries a remote price, true or not (N-106). An outside node is **not a rumour
source**: INF writes its rumour record only from a market's published price (`spec-information.md`
4.5), and a node has no market (3.1.2), because its counterparties trade in the gate region's
market. EXT adds no read function, no declaration and no field for INF and never writes INF's
table, an `AgentView` or a strategy input (FR-EXT-033). The open question 10 records the shape a
source would take. The market of a gate region is a rumour source like any market, so an interior merchant hears of the gate's published price, which includes the outside's trades, with the lag and the error of any rumour (`spec-merchants.md` 11.3); only the node itself is not a source.

9.3. **Routes.** TRN's `Routes` view lists the regions other than the agent's. A node
appears with no route, because a shipment to it is refused (`outside_destination`, 4.4). The
way to the outside is the gate's market, not a trip.

9.4. **A player.** An external agent in a gate region trades with the counterparties like any
agent, within the same admission rules (`spec-agents.md` 10).

## 10. The goods side

### 10.1 A `Raw` commodity and its source (N-021)

`spec-goods.md` 6.2 says a raw commodity is gathered, harvested or imported and that the
specs that supply real sources add a check. The real sources are three: a recipe with no
inputs (wild seed, salvage), a `Sited` recipe limited by a regional endowment (RES), and an
import line. This spec supplies the third.

10.1.1. **Sourced.** For the structural pass of `spec-config.md` 7.5, a commodity is sourced if
some node has a line for it with a sell side present and at least one gate of that node
for the sell side (3.2.1) and a window that is not 0 in every season. The cap does not
matter: a finite import is a source for validation and a risk for the run (10.1.3).

10.1.2. **How validation sees it.** The marker `Raw` still makes a commodity sourced
(`spec-goods.md` 6.2); the warning `raw_without_source` of `spec-viability.md` 4.2.1 is not
raised for a raw commodity that a sell side supplies, and for a raw commodity with none of the
three sources it stays. For V1 of the viability check an import makes the commodity producible
in the gate regions; the other regions get it through trade, which `spec-viability.md` 4.6
decides. A commodity sourced **only** by an import line draws `import_only_source` (14.2), so a
pack that depends on the outside for its upkeep sees it.

10.1.3. **What an import does not guarantee.** An import is bounded by the capacity, the
window, the cumulative caps and the edge status; when it stops the realm is a closed economy
for that commodity. The register entry EXT-A7 records the hazard and S2 the test.

### 10.2 A resource is not a commodity; an import is one

A resource of RES is a regional value that a recipe draws on (`spec-production.md` 6.5); a
draw is not a ledger flow. An import is a commodity that enters the ledger by a faucet reason
and is bought in a market. They do not interact: EXT reads no endowment and RES reads no line.
What a draw turns into is RES's. The choice between a resource and an import for a good is the
setting's: a limited local source is a resource, a good that comes from beyond is an import,
and a good that is both has a recipe with a draw and a line.

### 10.3 Tradability

A commodity with a sell side that no role of the realm lists in its tradable set
(`spec-agents.md` 3.4) can be bought by nobody; the warning `line_untraded` says so. The same
holds for a buy side.

### 10.4 Freshness of an import

An import is created fresh: its lot is born in the tick it is offered and a durable has its full
durability. An outside that sells goods already old or worn is not modelled (open question 5).

## 11. The money side

11.1. **A surplus is a faucet that is measured.** If the realm exports more than it imports, the
balance of trade is positive and `M` grows by it each tick (8.2, item 4). A constant surplus `g`
per tick gives `M(t) = M_0 + g t` and a rate of growth `g / M(t)` that falls as `1 / t`, the
unbalanced faucet of `spec-money-supply.md` 9.1, with the same reading for prices. Nothing
stops it except the cumulative caps of the buy sides (EXT-D2), the corridor itself (EXT-A1) and
the window; the warning `outside_faucet_without_sink` flags a world with buy sides and no sell
side anywhere, which can only be a surplus.

11.2. **A deficit is a sink that is measured.** If the realm imports more than it exports, `M`
falls by the balance, `M_h / H` falls toward the two thresholds of `spec-money-supply.md` 8.5
(the spending threshold first, then the liquidity floor), and households go unfed below the
floor. The deficit cannot go on for ever: the realm buys from the outside with the money it
holds, and an import is paid at placement of the bid by the buyer's reservation, so a buyer that
has run out of money places no bid it cannot fund (`spec-clearing.md` 3.2, `insufficient_funds`).
There is no credit to the outside in version 1 (CRD). The drain stops by exhaustion, after the
deflation of the stage before, unless the cap of a sell side stops it first (EXT-A4).

11.3. **No rule reads the balance.** Nothing in this module reads `M`, a balance of trade, a
price index or a count of agents. The limits of the outside are structural (capacity, window,
cumulative caps, the edge status) and its prices are a function of the tick, the drift and the
shock. A policy that reacts to the balance of trade, a tariff or a quota in response to a
deficit, is a policy actor (`spec-policy.md`) that acts through the shock slot (6.5), the edge
status (`spec-regions.md` 7) or a flow scale (`spec-money-supply.md` 10.4), as ADR 0001
decision 4 says.

11.4. **Deflation.** The closed economy's "deflation" is conditional (`spec-money-supply.md`
8.3, 8.4); the outside does not remove the condition, it adds a term to `M`: a surplus raises
it, a deficit lowers it. Whether prices fall is a result, not a rule (N-027).

11.5. **Hoarding of the outside's money.** Money the outside pays arrives in the accounts of
the realm's sellers, households or the estate, and stays there until spent. It does not reach a
region's pool directly: the pool receives only what the estate receives (`spec-estates.md` 2),
so a surplus raises `M_h` first and the pool only through deaths, relief returns and
uncollected goods' sales. This answers the question left by N-096 (how money entering a region
reaches its pool): by the households, and through them. A hoard of the outside's money is a
hoard like any (STB-L31, L45); the damper is the holding charge (D42) on the class that holds it.

11.6. **The spread.** The outside's margins are paid by the realm on any round trip. A realm
whose price swings between the bid and the ask pays about `ask - bid` on every unit it imports
at one end and exports at the other. The model of 19.2 loses about 83 minor units a tick over
400 ticks to a corridor of 45 to 75 (EXT-A5): the structural drain of a volatile realm,
measured by the balance of trade.

## 12. From the closed to the open economy

### 12.1 The steady state and the warm start

The closed steady state of `spec-viability.md` 5 conserves household money, which gave the
identity of 4.7 (the mean welfare ratio is 1 with upkeep only). With an outside, money crosses
the boundary, and a steady state needs the crossing to balance in value. This spec asks VIA for
an **open layer** on top of the programme, and defines its inputs; the solver's change is VIA's.

For each node, gate and line side present, in the order of the configuration, the layer gives
the solver: a column of imports (a sell side) or exports (a buy side) bounded above by the mean
capacity `Capacity * mean(Window) / 10 000` per tick; the price of the column, the quote at tick
0 with the gate's scale (6.6); and the **trade row**, `sum of import columns times their quotes
<= sum of export columns times their quotes`, so the realm does not finance an unbounded deficit
at the quotes. The dual of the programme then prices a traded commodity inside the corridor
where trade is active, at the ask where it is imported at the margin and at the bid where it is
exported. Because the quote is a limit and the realised price is the midpoint (6.7), the
realised balance of a run is worse for the realm than the one at the quotes; the difference is
measured in the boundary balances. The warm start applies nothing to counterparties (8.6), and
EXT supplies the inputs through a pure read function over the configuration, `Lines()`, in a
deterministic order, which the solver may call without a world. The checks of V1 treat an import
as a source in the gate regions (10.1.2).

### 12.2 The demand ceiling and the welfare ratio

With the demand module on, the closed mean welfare ratio is at most `1 + Wmax / U`, a ceiling of
the wants' physical quantities (`spec-demand.md` 7). An outside does not move it. A balanced
trade replaces domestic purchases by imports and domestic sales by exports at the same value,
so total income still equals total spending and the identity of `spec-viability.md` 4.7 holds
with the spending on imports among the purchases and the sales to the outside among the income.
A positive balance of trade raises the mean ratio above the closed value only while `M` grows,
and a negative one lowers it only while `M` falls (`spec-money-supply.md` 9.1): the outside is
not a source of a lasting ratio above 1, and it does not change the default `TargetRatio` of
15 000. N-054 is answered for EXT: no change.

### 12.3 What EXT does not give a non-producing class

A class with an empty `Recipes` list gets no income from the outside: a counterparty buys goods,
and a class that sells none receives nothing. A remittance or any other transfer from abroad is a recurring named
transfer, a grant of `spec-money-supply.md` 4.2, and is not provided here (N-078 is answered
there). A pack that wants one writes the grant and, for the pair, a sink (`pair_unbalanced`).

## 13. Parameters

### 13.1 The tunables

The tunables are the fields of the rows of 3.3. All are fields of
`Config.Modules.External`, with an owner (this spec), a bound and a doc line (architecture.md
9.5, 9.7), and each has a row in the parameter-owner table of `README.md`. The module has no
nodes unless the pack lists them, so there is no default macro constant (N-027). Every default
marked placeholder below is a placeholder.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Module | `Nodes` | the outside nodes, sorted by region (3.1) | none | at most `MaxOutsideNodes` |
| Node | `Region` | the region marked as outside | none | an existing region, no agents |
| Node | `Lines` | the node's dealings, sorted by commodity | none | at most `MaxOutsideLines` |
| Node | `Gates` | optional gate scales, sorted by region | none | gates of the node |
| Node | `Label` (p) | display name | "" | presentation only, not hashed |
| Gate | `Region` | a gate of the node | none | a gate (3.2.1) |
| Gate | `ScaleBP` | multiplier of both quotes at the gate (6.6) | 10 000 | 1 to `MaxGateScaleBP` |
| Line | `Commodity` | the commodity dealt in | none | an existing commodity |
| Line | `Base` | the mid at tick 0 (6.1) | none (required) | 1 to `MaxPrice` |
| Line | `PriceBySeason` | price factor by season (6.2) | absent (10 000) | each 1 to `MaxPriceFactorBP`, known seasons |
| Line | `TrendBP` | linear drift of the mid, basis points of `Base` a year (6.3) | 0 | `-MaxTrendBP` to `MaxTrendBP` |
| Line | `FloorBP` | lower clamp of the trend and walk factor (6.3) | 1 000 (placeholder) | 1 to 10 000 |
| Line | `CeilingBP` | upper clamp of the trend and walk factor (6.3) | 100 000 (placeholder) | 10 000 to `MaxFactorBP` |
| Line | `WalkStepBP` | the largest step of the drift per tick (6.4) | 0 (no walk) | 0 to `MaxWalkStepBP` |
| Line | `WalkRevertBP` | the share of the drift undone per tick (6.4) | 0 | 0 to 10 000 |
| Line | `WalkBoundBP` | the largest drift either way (6.4) | 0 | `WalkStepBP` to `MaxDriftBP` when a walk is on, else 0 |
| Side | `MarginBP` | sell: the ask above the mid; buy: the bid below the mid (6.6) | 0 | sell 0 to `MaxSellMarginBP`; buy 0 to `MaxBuyMarginBP`; the two together at least `MinQuoteSpreadBP` |
| Side | `Capacity` | units a tick at full availability (7.1); 0 = the side is absent | 0 | 0 to `MaxQuantity` |
| Side | `Window` | availability by season (7.2) | absent (10 000) | each 0 to 10 000, known seasons |
| Side | `TotalUnits` | cap on units traded over the run (7.3) | 0 (none) | 0 to `MaxLineTotal` (a unit is worth at least 1) |
| Side | `TotalValue` | cap on list value traded over the run (7.3, 7.4) | 0 (none beyond `MaxLineTotal`) | 0 to `MaxLineTotal` |
| Runtime | `Shock` | the line's price factor, state (6.5) | 10 000 | 1 to `MaxShockBP` |
| Runtime | `Drift`, `carry`, cumulative `Units` and list value, `Ended` | state (16) | 0 | within the bounds above |

No value of any field is a calibrated figure.

### 13.2 The bounds (`bounds.go`)

Bounds are constants, not tunables (`spec-config.md` 4). They are provisional until the slice
runs.

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxOutsideNodes` | 16 | counterparties and loops are bounded by the configuration; also a divisor of `MaxLineTotal` |
| `MaxOutsideLines` | 64 per node | the per-tick loop is nodes times gates times lines; 64 is a placeholder, well below `MaxCommodities` |
| `MaxCounterparties` | 256 | agent IDs, accounts and boundary accounts of the module are bounded |
| `MaxSellMarginBP` | 100 000 | an ask of at most eleven times the mid |
| `MaxBuyMarginBP` | 9 999 | a bid is at least a basis point of the mid (6.6) |
| `MinQuoteSpreadBP` | 1 | the smallest sum of the two margins of a line with both sides (6.6.1) |
| `MaxPriceFactorBP` | 100 000 | a seasonal factor of ten times |
| `MaxTrendBP` | 100 000 | ten times `Base` a year |
| `MaxFactorBP` | 1 000 000 | the largest `CeilingBP`: a hundred times `Base` |
| `MaxWalkStepBP` | 5 000 | half the mid in one tick is already a jump, not a walk |
| `MaxDriftBP` | 9 999 | keeps `F_walk` positive (6.4.1) |
| `MaxShockBP` | 100 000 | the shock slot, ten times |
| `MaxGateScaleBP` | 100 000 | ten times |
| `MaxLineTotal` | `MaxMoney / (2 * MaxOutsideNodes * 64)` = 488 281 250 000 000 | all sides together can move at most `MaxMoney` of list value; with the initial money and the flows of `spec-money-supply.md` 6.4.2 the money supply stays below `3 * MaxMoney` = 3 * 10^18, inside `int64` |

The products of the module are bounded by `MaxPrice * MaxQuantity` = 10^18 (a budget, a
receipt), the 128-bit product of 6.1 (at most 10^25) and `Capacity * Avail` (at most 10^13)
(`spec-config.md` 4.1, `spec-currency.md` 7).

## 14. Validation

### 14.1 Errors

Errors fail validation (`spec-config.md` 7.1). The root maps the sentinels (N-075).

| Error | Condition |
|-------|-----------|
| `ErrUnknownRegion` | a node or a gate row names an undefined region |
| `ErrUnknownCommodity` | a line names an undefined commodity |
| `ErrUnknownSeason` | a price or window table names an unknown season, or any season when the calendar has none |
| `ErrDuplicateID` | a second row for a node, a line, a gate or a season |
| `ErrOutOfBounds` | a field outside its bound in 13.1; a field set on an absent side (3.3.3); `WalkBoundBP` below `WalkStepBP` or set with no step; more than `MaxOutsideNodes` nodes, `MaxOutsideLines` lines, or `MaxCounterparties` counterparties |
| `ErrOutsideNodeInhabited` | a group, a warm-start row, a population region row, a taste row or a money-flow target names an outside node (3.1.2); the error names the table |
| `ErrCrossedQuotes` | a line with both sides whose margins sum to less than `MinQuoteSpreadBP` (6.6.1) |
| `ErrNotAGate` | a gate row names a region with no direct edge to or from the node (3.2.4) |
| `ErrReasonCollision` | a money flow ID equals one of the six reasons of 8.1 (`spec-money-supply.md` 4.5.2) |
| `ErrModuleDependency` | the module is on while the region graph is off (no `Edges`) |

### 14.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at construction, at
`Warn`, with the attributes `rule` and `path`; the volume is bounded by the configuration.

| Warning | Condition |
|---------|-----------|
| `node_without_gate` | a node with no gate for any side it has |
| `gate_edge_wrong_direction` | a side whose needed edge is absent while the opposite edge exists (a sell side with only region-to-node, a buy side with only node-to-region) |
| `line_without_side` | a line with both sides absent |
| `window_all_closed` | a side whose window is 0 in every season |
| `quote_outside_reference_band` | a quote at `Base` with its margin and a gate's scale outside `[RefBandLow, RefBandHigh]` of the effective reference price of the gate (6.6.2) |
| `walk_faster_than_beliefs` | `WalkStepBP + ceil(|TrendBP| / TicksPerYear)` above `BeliefStep` (6.4.3) |
| `outside_faucet_without_sink` | buy sides exist and no sell side exists anywhere (11.1) |
| `outside_drain_without_source` | sell sides exist and no buy side exists anywhere (11.2) |
| `outside_arbitrage` | for a commodity, the lowest ask at a gate (scaled, at `Base`) is below the highest bid at another gate: a carrier would earn before freight (6.6.1) |
| `import_only_source` | a commodity that some recipe, upkeep or tool consumes and whose only source is a sell side (10.1.2) |
| `line_untraded` | a line whose commodity is in no role's tradable set (10.3) |

## 15. Observability

### 15.1 Events

Three events are added to the table of `rules/observability-and-logging.md` 4.3. Each is a
plain struct with a `Tick`, no map, no pointer, fixed-point values (4.1 there).

| Event | Emitted when | Fields |
|-------|--------------|--------|
| `OutsideTraded` | once per (node, gate, commodity, side) per tick in which the side wanted any unit or any unit was cut, after the sweep, in ascending (node, gate, commodity, buy before sell); never when nothing was wanted or cut | `Node`, `Region` (the gate), `Commodity`, `Side` (`bid` for a buy side, `ask` for a sell side), `Mid`, `Price` (the limit), `Wanted`, `Offered`, `Traded`, `Value` (money paid or received in the trades), `ListValue`, `CutClosed`, `CutCap`, `CutPrice`, `PriceClamped` |
| `OutsideEnded` | a side reaches its cap and ends, once | `Node`, `Commodity`, `Side`, `Cause` (`units` or `value`), `Units`, `ListValue` (cumulative) |
| `OutsideShocked` | the shock slot is set between steps, once per changed (node, commodity) of a call | `Node`, `Commodity`, `From`, `To` |

`OutsideTraded` is an aggregate: it is bounded by nodes, gates and lines in the configuration
and is not per agent, order or trade. The events of a module that is off are not emitted. The
order of the events of a step follows the phases: the walk (phase 1) emits nothing, the orders
of the counterparties emit `OrderPlaced` with the others in phase 3, and the three events above
come after the sweep of phase 4. The counterparty trades appear as `TradeExecuted` with the
counterparty's agent ID.

### 15.2 Logs

New log keys (`rules/observability-and-logging.md` 3.3.2): `node` (an outside node's region ID),
`value` (money in minor units, `slog.Int64`), `nodes` and `counterparties` (counts in the
construction summary). The reason values are the six of 8.1.

| Level | Where | Message | Attributes |
|-------|-------|---------|------------|
| `Info` | construction summary | "world created" | adds `nodes`, `counterparties` |
| `Warn` | a side ends (7.3) | "outside side ended" | `node`, `commodity`, `side`, `cause` |
| `Warn` | construction | the warnings of 14.2 | `rule`, `path` |
| `Debug` | each side per tick, guarded by `Enabled` | "outside side" | `node`, `region`, `commodity`, `side`, `price`, `quantity`, `value` |
| `Debug` | the walk step | "outside drift" | `node`, `commodity`, `drift` |

`Info` is not emitted in the tick loop and `Warn` only for an ended side, at most once for each
side of the configuration (observability-and-logging.md 3.2.2). The key `drift` (an integer, basis
points) joins the table with the keys above.

### 15.3 Cost with no observer

With the default logger and observer the step builds no event, formats no attribute and
allocates nothing beyond scratch held by the world and reused across ticks: the per-side
records, the counterparty list and the trade totals are slices of fixed size, sized at
construction (observability-and-logging.md 4.5, 8.2; NFR-EXT-001).

### 15.4 Snapshot

The Snapshot carries, as plain data (names are `spec-observability.md` 5.3, N-094; the balance of trade is not a field, 8.3): for each node, line and gate the
mid, the quote of each side, the drift, the shock and the carry; for each side the cumulative
units, the list value and whether it ended; per commodity the cumulative `import_c` and
`export_c` of 8.2 (the sums of the sides' `Units` per commodity); the six boundary balances
per reason (money, and goods per commodity); and the counterparties as pseudo-agents with their account IDs and gate regions
(FR-EXT-046).

### 15.5 The Observability entry (observability-and-logging.md 9)

1. *New failure path or rejected input?* The configuration errors of 14.1 are returned at
   construction; `SetOutsideShocks` returns its errors; a cut of an order is survived and
   counted in `OutsideTraded`, with the cause as a field and no `Warn` per tick.
2. *New state transition?* A top-up, a sweep, a drift, a shock and an end: the first two are
   ledger transfers reported in `BoundaryFlow` and `OutsideTraded`; the others have events.
3. *New decision point?* None in a strategy. The order of a side is a pure function (5.2) and
   its inputs are in the `Debug` line.
4. *New loop?* The per-tick loop over nodes, gates and lines logs at `Debug` only, guarded, and
   allocates nothing with the no-op observer (NFR-EXT-001).
5. *Changed event, metric or attribute?* None changed; three events, five keys and three metrics
   are added.
6. *New metric attribute?* `economysim.node`: at most `MaxOutsideNodes` values, bounded by
   configuration (5.4.1 there).
7. *`Error` or `Info`?* `Info` only in the construction summary.
8. *Logging an error also returned?* No.
9. *Tests?* FR-EXT-036 to FR-EXT-039 and NFR-EXT-001 (section 21).

Metrics (for `rules/observability-and-logging.md` 5.3, built by `otelobserver`):

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.external.units` | Counter | `{unit}` | `economysim.node`, `economysim.commodity`, `economysim.side` | `OutsideTraded` (`Traded`) |
| `economysim.external.value` | Counter | `{minor_unit}` | `economysim.node`, `economysim.commodity`, `economysim.side` | `OutsideTraded` (`Value`) |
| `economysim.external.price` | Gauge | `{minor_unit}` | `economysim.node`, `economysim.commodity`, `economysim.side` | `OutsideTraded` (`Price`) |

The money crossing the boundary is also in `economysim.money.flow` through `BoundaryFlow`, by
reason, so a dashboard of the balance of trade is `outside_budget` less `outside_unspent` less
`outside_receipts`.

## 16. State contributed to a checkpoint

ADR 0002 asks every spec for the state it adds. With the module on, in the order of the
configuration (nodes by region, lines by commodity):

| State | Per | Saved as |
|-------|-----|----------|
| `Drift` | line with a walk | `int64` |
| `Shock` | line | `int64` (10 000 when never set) |
| `carry` | side | `int64` in `[0, 10 000)` |
| cumulative `Units` | side | `int64` |
| cumulative list value | side | `int64` |
| `Ended` | side | boolean |

The state saved is therefore a few integers per line and no map. **Not saved:** the mid and
the quotes (functions of the tick, the drift and the shock), the gates and the counterparty
list (derived from the configuration and the edges' existence), and the counterparty accounts
(empty; the ledger saves them as accounts of the kind Outside with a zero balance). The edge
status is REG's. A restored world continues exactly (FR-EXT-016). The seed and the world's
source are saved by WLD and CHK, so the walk's next draw is the same.

The effective configuration hash covers every field of 13.1 but `Label`, with nodes, lines,
gates and season factors in canonical order; the shock and the other runtime state are not in
the hash (`spec-config.md` 8).

## 17. Entries for the stability register

Each entry follows the checklist of `spec-stability.md` 9. Provisional IDs: the supervisor
assigns the final numbers (N-103). Status `model` means a run of section 19; nothing was run in
the library. `STB-L74` (a top-up peg) was written for MON's top-up as a budget fixed in money,
whose demand is unit-elastic; this spec's counterparty places a fixed **quantity** at a limit
price, so L74's neutral case does not arise here and its entry can be closed with a pointer to
EXT-A2.

**EXT-A1 The price-specie flow.** *Kind:* loop. *Loop:* the money supply `M` (+) the realm's
prices (via the budgets of households) (+) imports and (-) exports, so (-) the balance of trade
(+) `M`. One negative link: B. *Delay:* the belief adjustment (`spec-beliefs.md` 5.2, a few
ticks), the welfare window `T` and the production delay. *Owner:* EXT 6.6, 8.3, 11. *Damper:*
EXT-D1, EXT-D2. *Undamped:* the loop acts only where the realm's price reaches a quote; a
realm whose price sits at the bid with a standing glut exports at capacity whatever `M` is,
and `M` keeps rising. *Condition:* the realm's prices respond to `M` through demand within the
cap's horizon. *Test:* EXT-S1. *Status:* hand.

**EXT-A2 The corridor confines a cobweb and pins a price.** *Kind:* constraint. *Loop:*
excess demand in the realm (+) the price (-) the quantity the outside trades at an edge of the
corridor; inside the corridor the outside is silent. Not a loop of its own: a bound on L21
(the production-delay cobweb) and, at the edge, a pin like F5. *Delay:* the production delay.
*Owner:* EXT 6.6, 6.7. *Damper:* itself (EXT-D1). *Undamped:* a capacity below the swing of the
excess demand, and the midpoint rule, which lets the realm's price leave the corridor (6.7).
*Condition:* `Capacity` at least the swing of the realm's excess demand inside the corridor. In
the stylised model (19.2), with a supply slope twice the demand slope, the realm alone cycles
between the bounds of the reference band, 20 and 120, with 19.5 units made and unsold a tick;
with an outside ask of 75 and bid of 45 it cycles between 45 and 75 with none unsold when the
capacity is 10 or 40, and between 20 and 120 again when the capacity is 3. A tight corridor of
55 to 65 confines it to 55 to 65. It also answers N-043 in part: an outside quote anchors the
price level of a traded good at its edges, never in the interior. *Test:* EXT-S1. *Status:*
model (a uniform-price stand-in for the clearing and belief rules).

**EXT-A3 The capacity cliff.** *Kind:* hazard. *Loop:* a standing outside bid up to the
capacity `C` (+) producers' expansion (+) output after the production delay; beyond `C` the
extra supply meets the realm's market alone, and the bid disappears at once when a window
closes, a cap is reached or an edge closes. *Delay:* the recipe `Duration`. *Owner:* EXT 7.
*Damper:* EXT-D4 (the window is known in advance), EXT-D2 (the end is counted). *Undamped:* an
end that producers cannot see coming (a shock, a closure), and a window shorter than the
production delay (the seasonal output of L23 meets a seasonal bid). *Condition:* the window is longer
than the delay of the recipes that serve it, or the realm holds stock for the closed season.
*Test:* EXT-S4. *Status:* hand.

**EXT-A4 The drift of the balance of trade.** *Kind:* loop with drift (neighbour of STB-L30,
L69, L73). *Loop:* a standing surplus (+) `M` (the unbalanced faucet of `spec-money-supply.md`
9.1); a standing deficit (-) `M`, toward the spending threshold and the liquidity floor
(`spec-money-supply.md` 8.5). *Delay:* none in the accounting; the price response is EXT-A1's.
*Owner:* EXT 8.3, 11. *Damper:* EXT-A1; EXT-D2 (the caps); the realm cannot borrow, so a
deficit ends by exhaustion; the warnings `outside_faucet_without_sink` and
`outside_drain_without_source`. *Undamped:* a persistent imbalance with no price response, by
design and measured. *Condition:* the balance of trade is small against `M` over the horizon, or
EXT-A1 acts. *Test:* EXT-S1, EXT-S4. *Status:* hand, and the accounting identity run in 19.1.

**EXT-A5 The spread is a tax on volatility.** *Kind:* bias. *Loop:* none. A realm whose price
cycles between the bid and the ask buys at the ask and sells at the bid, and pays the spread
on every round trip; the outside also keeps the price improvement of the midpoint. In 19.2 the
cycle of 45 to 75 loses about 83 minor units a tick over 400 ticks (imports 3.5 units a tick at
75, exports 4.0 at 45). *Owner:* EXT 6.6, 11.6. *Damper:* none by design (the spread is data).
*Undamped:* all of it, and it is measured by the balance of trade. *Condition:* none. *Test:*
EXT-S1. *Status:* model.

**EXT-A6 A drifting outside price.** *Kind:* amplifier (an exogenous forcing). *Loop:* the walk,
trend or season moves the mid (+) the quote (+) the realm's belief, with a lag (+) the realm's
trades at the gate; a seasonal mid forces a seasonal cycle on the exports (L23). *Delay:* the
belief step. *Owner:* EXT 6.1 to 6.4. *Damper:* EXT-D5 (the clamps and the bound of the walk),
D02 and D03 of the beliefs. *Undamped:* a drift per tick above what beliefs follow. *Condition:*
the largest change of the mid per tick is at most one `BeliefStep` (the warning
`walk_faster_than_beliefs`, a heuristic). *Test:* EXT-S3. *Status:* hand.

**EXT-A7 A single source.** *Kind:* hazard (neighbour of STB-L66). *Loop:* none. A commodity
that the realm can only import is lost when a window closes, a cap is reached, an edge is closed
(7.5) or a shock raises the price beyond the realm's reach; the households then go unfed and the
failure ladder starts (`spec-spawner.md` 5). *Owner:* EXT 10.1.3. *Damper:* the warning
`import_only_source`; the counted ends (EXT-D2, EXT-D4). *Undamped:* by design. *Condition:* a
second source, a stock cover (`spec-agents.md` 7.3) or acceptance. *Test:* EXT-S2. *Status:* hand.

**EXT-A8 Arbitrage through the outside.** *Kind:* loop with a delay (neighbour of STB-L84). *Loop:*
the gap between a gate's landed ask and the realm's interior price (+) the units a carrier buys
at the gate (-) the gap, after the carrier's trip. B, with the delay of the trip `k`. At one gate
no loop pays (6.6.1); across two gates with different scales one can, before freight. *Delay:*
the travel time (`spec-transport.md` 4). *Owner:* EXT 6.6.1; MER's decision. *Damper:* the
outside's capacity, which every buyer at the gate shares; the freight as a floor (D40); the
warning `outside_arbitrage`. *Condition:* the shipper counts `Inbound` (L84). *Test:* S30 with an
outside. *Status:* hand.

Dampers proposed for the catalogue of `spec-stability.md` 5:

| ID | Damper | Mechanism | Home | Status |
|----|--------|-----------|------|--------|
| EXT-D1 | Quote corridor | the ask is above the bid by at least a basis point, and a quote is a limit that the realm's price meets at the edge; cost: the price signal is lost at the edge and the outside keeps the spread | EXT 6.6, 6.7 | specified |
| EXT-D2 | Capacity and cumulative caps | `Capacity` per tick and `TotalUnits`, `TotalValue` over the run; a side that reaches them ends and says so; cost: an outside that runs out | EXT 7.1, 7.3 | specified (counted by `CutCap`, `OutsideEnded`) |
| EXT-D3 | Sweep to zero | a counterparty holds nothing across a tick, so the outside's money and goods cannot accumulate or compound in the ledger | EXT 5.3 | specified (a rule constant) |
| EXT-D4 | Closure and window | a closed edge or a closed season stops a flow at once; cost: a cliff for producers (EXT-A3) | EXT 7.2, 7.5 | specified (counted by `CutClosed`) |
| EXT-D5 | Price clamps | `FloorBP`, `CeilingBP`, `MaxPrice` and `WalkBoundBP` bound the path | EXT 6.3, 6.4 | specified (counted by `PriceClamped`) |

Scenarios proposed (the packs are REF's):

| ID | Scenario | Measure |
|----|----------|---------|
| EXT-S1 | A pack with a staple and an outside corridor, a production-delay cobweb injected, capacity above and below the swing, at the default clearing (midpoint) | the price range with and without the outside; the balance of trade; the share of ticks at a quote; the pull of beliefs toward the quote (6.7) |
| EXT-S2 | An import-only upkeep good and a closed edge from tick `t` (`SetEdgeStatuses`) | the time to the first unfed households; the ladder; the counters of `CutClosed` |
| EXT-S3 | A walk and a trend on a traded good, the step above and below one belief step | the realm's lag behind the mid; the warning `walk_faster_than_beliefs` |
| EXT-S4 | A buy side with `TotalUnits` reached at tick `t` and a seasonal window | the export cliff; the glut after the end; `OutsideEnded` |
| EXT-S5 | The invariant run: the reference chain with the overlay of 18.2, over a range of seeds and with `WithAudit` | exact equality of 8.2; the counterparties empty at every boundary |

## 18. Worked examples

Both are examples in invented settings and are flagged so. The arithmetic of 18.1 and 18.3 is
checked by `models/ext/cycle.py` (19.1). The names (a coast, a ring, grain, salt, ore) are the
examples' and appear nowhere in the core.

### 18.1 Example A, one tick (a river valley; unit: one league)

Two regions, sorted by ID: `coast` (the outside node, agent ID 0 for its unused estate) and
`lowland` (agent ID 1); edges `coast` to `lowland` and `lowland` to `coast` of 30 leagues. The
counterparties are agent 2 (the buyer at `lowland`) and agent 3 (the seller at `lowland`);
households start at agent 4.

Two lines at `coast`: `salt` (Base 40, sell side, `MarginBP` 500, `Capacity` 30) and `grain`
(Base 20, buy side, `MarginBP` 1 000, `Capacity` 50). In tick 0 the mids are 40 and 20; the ask
for salt is `ceil(40 * 10 500 / 10 000)` = 42 and the bid for grain is
`floor(20 * 9 000 / 10 000)` = 18.

*Grain (an export).* The buyer's account is topped up with `18 * 50` = 900 (`outside_budget`).
The realm's sellers ask 15 (20 units), 17 (15), 19 (10) and 21 (30). The book matches the bid of
18 with the cheapest ask first: 20 units at the midpoint of 15 and 18, 16.5, rounded half to even
to 16 (value 320); then 15 units at the midpoint of 17 and 18, 17.5, rounded to 18 (value 270).
The bid of 18 does not reach the ask of 19. Traded 35 units for 590. The sweep returns 310
(`outside_unspent`) and takes the 35 grain (`outside_purchases`).

*Salt (an import).* The seller's account receives 30 units (`outside_offer`). The realm's buyers
bid 50 (10 units), 45 (10) and 41 (10). The ask of 42 is matched with the highest bid first: 10
units at 46 (460), then 10 at the midpoint of 42 and 45, 43.5, rounded half to even to 44 (440);
the bid of 41 is below the ask. Traded 20 units for 900. The sweep takes 10 unsold salt
(`outside_unsold`) and the 900 (`outside_receipts`).

| Reason | Thing | Amount this tick |
|--------|-------|------------------|
| `outside_budget` | money | 900 |
| `outside_unspent` | money | 310 |
| `outside_receipts` | money | 900 |
| `outside_offer` | salt | 30 |
| `outside_unsold` | salt | 10 |
| `outside_purchases` | grain | 35 |

The balance of trade of the tick is `(900 - 310) - 900` = -310: the realm exported grain worth
590 and imported salt worth 900, and its money fell from 3 000 to 2 690 (the model prints it).
`BoundaryFlow` reports the six rows for `lowland`, and the audit finds both counterparty
accounts empty.

### 18.2 Example A, the configuration

```toml
[module.external.node.coast]

[module.external.node.coast.line.salt]
base = 40
trend_bp = -300

[module.external.node.coast.line.salt.sell]
margin_bp = 500
capacity = 30

[module.external.node.coast.line.grain]
base = 20
walk_step_bp = 100
walk_revert_bp = 500
walk_bound_bp = 2000

[module.external.node.coast.line.grain.buy]
margin_bp = 1000
capacity = 50
window = [{ season = "winter", bp = 6000 }]
```

Salt's price falls 3% of `Base` a year; grain's mid does a walk with a stationary standard
deviation of about 186 basis points, bounded at 20% (6.4.2); the outside buys grain at 60% of
its appetite in winter. Over two years of 360 ticks with four seasons of 90, the model (19.1)
moves 24 420 grain out and 21 520 salt in, exports worth 409 560 and imports worth 951 790, a
balance of -542 230 (the realm paid more for the salt than it earned: the model's buyers bid
high); the walk stays between -568 and 395 basis points. The model's realm orders are invented
and only the arithmetic and the identities are the point.

### 18.3 Example B: a cap that ends a line (an orbital colony; unit: metres per second of delta-v)

Regions `inner_system` (the node) and `ring` (the gate), with edges both ways of 2 400. Two lines:
`ore` (Base 120, buy side, `MarginBP` 2 000, `Capacity` 200, `TotalUnits` 1 500) and `reagent`
(Base 300, `TrendBP` -200, sell side, `MarginBP` 800, `Capacity` 10, `TotalValue` 12 000).

The bid for ore is `floor(120 * 8 000 / 10 000)` = 96; the colony's miners ask 90, so each trade
is at the midpoint 93. The outside takes 200 units a tick; after 7 ticks it has taken 1 400 and
in tick 7 the offer is cut to the remaining 100 (`CutCap` 100) and the side ends (`OutsideEnded`
with the cause `units`): 1 500 units for 139 500. The ask for reagent is 324 (the trend lowers the factor by a basis point or two, and the mid stays 300 after
rounding). The
colony's buyers bid 400, so each trade is at 362. The line offers 10 units in each of ticks 0 to 2
(list value 3 240 a tick), and in tick 3 the remaining 2 280 of the cap allows `floor(2 280 / 324)`
= 7 units (`CutCap` 3); the list value is then 11 988 of 12 000, the remaining 12 is below the
quote and the side ends (cause `value`). The colony paid 13 394 for 37 units, more than the list
value, because the midpoint favours the outside (7.4). The balance of trade of the run is
139 500 - 13 394 = +126 106: the colony gained money from ore the outside no longer buys, and
the realm's miners now face the glut of EXT-A3 (S4).

## 19. What was run

Two small models, in `doc/specs/models/ext/`, both run in Docker (`python:3.12-slim`) with their
output in `cycle.out` and `corridor.out`. They are prototypes of the rules of this spec in Python
integers, not of the library: the 128-bit intermediates are not exercised, Python's `random` stands
in for the world's source (its draws are not the library's) and every number is a placeholder.

### 19.1 `cycle.py`: the counterparty cycle

An integer ledger with signed boundaries, a copy of the average clearing (sort by limit with ties
by position, midpoint half to even, a crossing check) and the cycle of sections 5 to 8 for lines
with a walk, a trend, a window, a capacity with a carry and cumulative caps. It asserts after every
tick that the total of money and of each commodity over all accounts and boundaries is zero and
that both counterparty accounts are empty, and that `M_t - M_0` equals the sum of
`budget - unspent - receipts`. It shows: the worked tick of 18.1 (35 grain for 590, 20 salt for
900, change -310, the six flows above); two years of example A (the cumulative flows of 18.2,
balance -542 230 equal to the change of the money in the realm, the drift range -568 to 395); and
the caps of example B (the sides ending at ticks 3 and 7, list value 11 988 against realised money
13 394, balance +126 106).

### 19.2 `corridor.py`: a cobweb with and without an outside

A stylised model that replaces the belief and clearing dynamics by a **uniform** price (the
smallest integer price at which demand does not exceed supply, held to a reference band of 20 to
120), so it shows the shape of the effect of a corridor and not its size in the library. Realm
demand falls by 1 unit per 2 of price; supply is decided one tick ahead with a slope of 1, a
cobweb multiplier of 2 in magnitude. Results over ticks 100 to 399: alone, the price alternates
between 20 and 120 and 19.5 units a tick are made and unsold; with the outside at an ask of 75 and
a bid of 45 and a capacity of 10 or 40 it alternates between 45 and 75, nothing is unsold, imports
average 3.5 units and exports 4.0 a tick and the cumulative money balance is -33 225 over 400
ticks; with a capacity of 3 the corridor is not enough and the price is back to 20 to 120; with an
ask of 65 and a bid of 55 it stays in 55 to 65. What it does **not** show: the midpoint rule
(which lets the realm's price leave the corridor, 6.7), the belief rules, the realm's response of
money to prices, or any behaviour of the library.

## 20. Notes settled

| Note | What this spec does |
|------|---------------------|
| N-021 | **Applied for EXT, leave open for FIS and LED.** A new faucet or sink is a registry row: the six reasons of 8.1; a counterparty whose balance is created outside conservation is a Boundary account (4.2, 4.3). The goods side: an import line is a real source of a `Raw` commodity (10.1); how validation sees it (10.1.2) |
| N-046 | **Leave open; EXT's entries delivered** (17): EXT-A1 to A8, dampers EXT-D1 to D5, scenarios EXT-S1 to S5 |
| N-084 | **Leave open for MER, INF, STY.** An external overlay (18.2) can be layered on a pack for the invariant scenario S5; the dynamic parts of X1 and X2 wait for MER and INF; EXT gives them the gate as the place where imports appear |
| N-095 | **Applied, close.** The decision between a boundary-backed admission and a pass-through account is made with reasons (4.1): the pass-through account; `spec-clearing.md` changes by one wording only (4.4) |
| N-096 | **Applied for EXT, leave open for POP, EST, SPN.** Money the outside pays reaches the households first and the pool only through the estates (11.5); EXT adds no pool channel |
| N-104 | **Applied for EXT, leave open for the other specs.** The node and line rows are keyed by region and commodity, one row per key, in canonical order with the errors of `spec-regions.md` 5.3; the gate scale is a scale row (3.2.4); the rows are to be listed in 5.1.3 there |
| N-107 | **Applied for EXT, leave open for FIS, CRD, MON, LED.** A trade with an external counterparty is the case for built-in reasons, not a grant or a levy (8.1); MON's positions bind: no rule reads an aggregate (11.3), a counterparty is a Boundary account's partner (4.2) |
| N-054 (closed) | **Read.** EXT is not what lets the ratio exceed 1 for long (12.2) |
| N-078 (closed) | **Read.** EXT gives a non-producing class no income (12.3); a remittance is a grant |
| N-016 | **Applied.** Every field of 13.1 has a row proposed for the parameter-owner table (handoff, b); no parameter is unnamed |
| N-017, N-027 | **Applied.** Names are neutral (outside, node, line, side, gate, counterparty, quote); no default macro constant is stated |
| N-103 | **Applied.** The worktree, the scope fence, the handoff and the provisional IDs follow it |
| N-106 | **Applied.** No view carries an outside price (9.1, 9.2); an outside node is not a rumour source in version 1 (9.2) |
| N-009, N-012, N-020, N-075, N-091 | **Leave open for WLD, OBS, CHK, CFG, INT.** The three positions of 5, the events and metrics of 15, the state of 16, the sentinels of 14.1 and the shock slot of 6.5 |
| N-018 | **Observed.** No default is set from an unread source |

## 21. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Every number is a bound
or a placeholder owned by this spec; none is calibrated (N-015, N-027).

### Functional

| ID | Requirement |
|----|-------------|
| FR-EXT-001 | With `Config.Modules.External` nil, the ledger's registry MUST hold none of the six reasons of 8.1, no account of the kind Outside MUST be opened, no outside state MUST be kept or saved and no outside event MUST be emitted. |
| FR-EXT-002 | A node row MUST name an existing region, otherwise `ErrUnknownRegion`, and MUST mark it without changing its `RegionID`, its edges or its estate account. |
| FR-EXT-003 | A group, a warm-start group, pool or wage row, a population region row, a taste row or a money-flow target naming an outside node MUST be rejected with `ErrOutsideNodeInhabited` naming the table. |
| FR-EXT-004 | An outside node MUST have no market: no `MarketCleared` MUST be emitted for it, and `region_without_agents` MUST NOT be raised for it. |
| FR-EXT-005 | A gate of a node MUST be a region with a direct edge to or from it; the edge from the node MUST make it a gate for the sell sides and the edge to the node a gate for the buy sides, derived at construction from the configured edges. |
| FR-EXT-006 | A gate row's `ScaleBP` MUST default to 10 000 and multiply both quotes of the gate, and a gate row naming a region that is not a gate of the node MUST be rejected with `ErrNotAGate`. |
| FR-EXT-007 | Nodes, lines, gates and season factors MUST be in canonical order before a world is built or a hash is taken, and a second row for a key MUST be rejected with `ErrDuplicateID`. |
| FR-EXT-008 | A side with `Capacity` 0 MUST be absent, and a non-zero field on an absent side MUST be rejected with `ErrOutOfBounds`. |
| FR-EXT-009 | A line with both sides MUST have margins summing to at least `MinQuoteSpreadBP`, otherwise `ErrCrossedQuotes`, and in every tick at every gate its bid MUST be strictly below its ask. |
| FR-EXT-010 | The ask MUST be the mid times `10 000 + MarginBP` over 10 000 rounded up, the bid the mid times `10 000 - MarginBP` over 10 000 rounded down, each then scaled by the gate with the same rounding and held to `MaxPrice`; a bid below 1 MUST place no order and the units MUST be counted as `CutPrice`. |
| FR-EXT-011 | The counterparties MUST take the agent IDs after the estate pseudo-agents in ascending (node, gate, buy before sell), MUST NOT be counted as agents or households, and MUST NOT cause `AgentSpawned`. |
| FR-EXT-012 | A counterparty MUST have exactly one account of the kind Outside in the gate region, opened at construction after the estate accounts and before any agent account, never closed and never relocated, and MUST hold no money and no goods at any tick boundary. |
| FR-EXT-013 | A counterparty's order MUST be placed by the ordinary admission after the estates' orders, in ascending agent ID, MUST be admitted for every configured case, MUST take a tie-break key like any order, and a buyer MUST place only bids and a seller only asks. |
| FR-EXT-014 | Before admission a buyer's account MUST receive exactly `limit * offered` under `outside_budget` (the order may reserve less under a legal price, 5.2.3) and a seller's exactly `offered` units, born in that tick and at full durability, under `outside_offer`. |
| FR-EXT-015 | After every market has cleared and every reservation has been settled or released, and before outcomes are processed, each counterparty account MUST be emptied to the boundary under `outside_unspent`, `outside_purchases`, `outside_unsold` or `outside_receipts` according to 5.3.1. |
| FR-EXT-016 | A checkpoint taken between ticks MUST find every counterparty account empty, and a restored world MUST place the same orders and apply the same sweeps as an uninterrupted one. |
| FR-EXT-017 | With the module on, the registry MUST hold the six reasons with the kinds of 8.1, closed for the run, and a money flow ID equal to one of them MUST be rejected with `ErrReasonCollision`. |
| FR-EXT-018 | The Boundary accounts of the six reasons MUST be created at construction, the money reasons first and the goods reasons by commodity in commodity order, after those of the built-in reasons and of the declared flows and before any other account. |
| FR-EXT-019 | In every tick, `import_c` MUST equal `outside_offer_c - outside_unsold_c` and the units the seller counterparties traded, `export_c` MUST equal `outside_purchases_c` and the units the buyer counterparties traded, `export_value` MUST equal `outside_budget - outside_unspent` and `import_value` MUST equal `outside_receipts`. |
| FR-EXT-020 | After every tick `M` MUST equal `M_init` plus the grants and other faucets applied, less the levies and other sinks, plus the sum over ticks of `export_value - import_value`, with exact equality, and the sectoral identity of `spec-ledger.md` 9.3 MUST hold with the outside sector at zero. |
| FR-EXT-021 | The module MUST draw once per line with `WalkStepBP` above 0 per tick, in phase 1, in ascending (node, commodity), MUST draw nothing else, and the number of draws in a tick MUST NOT depend on the state of the world. |
| FR-EXT-022 | The drift MUST be updated as `D - trunc(D * WalkRevertBP / 10 000) + d`, with `d` uniform on `[-WalkStepBP, WalkStepBP]`, held to `[-WalkBoundBP, WalkBoundBP]`, and `WalkBoundBP` MUST be at least `WalkStepBP` and at most `MaxDriftBP`. |
| FR-EXT-023 | A side's offered quantity and limit MUST be a function of the tick, the season, the line's state and the edge's status only, and MUST NOT change when any balance, price, inventory or count of the realm changes. |
| FR-EXT-024 | The mid MUST be `clamp(round_half_even(Base * S * F * Shock / 10^12), 1, MaxPrice)` with `F` as in 6.1, MUST equal `Base` for a line with no season table, no trend, no walk and the default shock, and MUST be computed from the tick, the drift and the shock without being stored. |
| FR-EXT-025 | A season table naming an unknown season, or any season when the calendar has none, MUST be rejected with `ErrUnknownSeason`, and a tick with no season MUST use the factor 10 000. |
| FR-EXT-026 | The trend factor MUST be linear and truncated toward zero, `F` MUST be held between `FloorBP` and `CeilingBP`, and a tick in which a clamp changes `F` or the mid MUST set `PriceClamped`. |
| FR-EXT-027 | The shock MUST default to 10 000, MUST be changed only between steps by `SetOutsideShocks` within 1 to `MaxShockBP`, MUST apply from the next step, MUST be saved and not hashed, and MUST emit `OutsideShocked`; the engine MUST NOT change it. |
| FR-EXT-028 | The wanted units MUST be `floor((Capacity * Avail + carry) / 10 000)` with the carry set to the remainder, advancing by the wanted units whether or not they are offered. |
| FR-EXT-029 | A season with availability 0 MUST place no order and MUST NOT be counted as a cut; a window that is 0 in every season MUST draw `window_all_closed`. |
| FR-EXT-030 | The offered units MUST NOT exceed the remaining `TotalUnits` or `floor(remaining list value / limit)`; the units cut MUST be counted as `CutCap`; a side that reaches a cap MUST end for good, emit `OutsideEnded` once with the cause and place nothing afterwards. |
| FR-EXT-031 | The list value of a tick's trades on a side MUST be the units traded times the limit price of the counterparty's order, and the effective list-value cap MUST be the smaller of `TotalValue` (when set) and `MaxLineTotal`. |
| FR-EXT-032 | A side whose needed edge has the status closed MUST place no order and the wanted units MUST be counted as `CutClosed`; an edge's length factor MUST have no effect. |
| FR-EXT-033 | No view, strategy input or INF table MUST carry or receive an outside price from this module, and the module MUST declare no field and offer no function for INF. |
| FR-EXT-034 | A shipment request whose destination is an outside node MUST be refused with `outside_destination`, and a relocation into one with `ErrOutsideNode`; neither MUST change any state. |
| FR-EXT-035 | A commodity with a sell side that has a gate and a window not closed in every season MUST count as sourced for the structural check of `spec-config.md` 7.5, `raw_without_source` MUST NOT be raised for it, and a commodity sourced only so MUST draw `import_only_source`. |
| FR-EXT-036 | `OutsideTraded` MUST be emitted once per (node, gate, commodity, side) per tick in which any unit was wanted or cut, after the sweep, in ascending (node, gate, commodity, buy before sell), with the fields of 15.1, no map and no pointer, and MUST NOT be emitted when nothing was wanted or cut. |
| FR-EXT-037 | `OutsideEnded` MUST be emitted once per side that ends and `OutsideShocked` once per changed (node, commodity) of a call of `SetOutsideShocks`; neither MUST be emitted with the module off. |
| FR-EXT-038 | The tick MUST NOT log at `Info` or `Error`, MUST log `Warn` only for an ended side and the warnings of construction, and its `Debug` lines MUST be guarded by `Enabled`, have constant messages and use the keys of 15.2. |
| FR-EXT-039 | The reasons of 8.1 MUST appear in `BoundaryFlow` under their names for the gate region with the kind faucet or sink. |
| FR-EXT-040 | A field outside its bound, a side set on an absent side, a `WalkBoundBP` below the step, or more than `MaxOutsideNodes` nodes, `MaxOutsideLines` lines or `MaxCounterparties` counterparties MUST be rejected with `ErrOutOfBounds` naming the path. |
| FR-EXT-041 | The warnings of 14.2 MUST NOT fail validation, and each MUST name its path. |
| FR-EXT-042 | The module MUST be rejected with `ErrModuleDependency` while the region graph is off. |
| FR-EXT-043 | Every field of 13.1 MUST have a default that passes validation, a bound and a row in the parameter-owner table; the hash MUST cover every field but `Label`, with nodes, lines, gates and season factors in canonical order, and MUST NOT cover the runtime state. |
| FR-EXT-044 | The module MUST draw no number at agent creation, and the draws of `spec-agents.md` 3.5.2 MUST NOT depend on whether it is on. |
| FR-EXT-045 | Nothing of the module MUST run during construction, and the construction bucket MUST contain no outside reason. |
| FR-EXT-046 | The Snapshot MUST carry the node, line and side state of 15.4 and the counterparties marked as pseudo-agents, as plain data. |
| FR-EXT-047 | `Lines()` MUST return the configured lines with their gates, quotes at tick 0 and mean capacities in a deterministic order, and MUST be callable without a world. |
| FR-EXT-048 | Two nodes with the same gate MUST place two orders for the same commodity from different counterparties, and a buyer and a seller of one gate MUST NOT trade with each other. |
| FR-EXT-049 | Under `AverageClearing` a realm buyer with a limit of 50 against a counterparty ask of 42 MUST trade at 46, and a realm seller with an ask of 15 against a counterparty bid of 18 at 16 (half to even): the quote is a limit and not a peg. |
| FR-EXT-050 | The audit MUST report, with the node and commodity, a counterparty account that is not empty, a boundary balance with the wrong sign for its kind, or a boundary balance that disagrees with the cumulative units of its sides. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-EXT-001 | With the default logger and observer, the module's steps in a tick of a reference-chain run MUST allocate nothing beyond scratch held by the world and reused across ticks (observability-and-logging.md 8.2). |
| NFR-EXT-002 | The cost of the module per tick SHOULD be proportional to the number of nodes times their gates times their lines, and SHOULD have a benchmark with `b.ReportAllocs()`. |
| NFR-EXT-003 | Validation of the module SHOULD have a fuzz test with random node tables (never panics; a valid table gives the same effective configuration and hash built in Go or read from a file; an invalid one returns a sentinel). |
| NFR-EXT-004 | Over a range of seeds the drift of a walking line MUST stay within `[-WalkBoundBP, WalkBoundBP]` and the mid within `[1, MaxPrice]`, and a run MUST be bit-identical across runs and platforms (architecture.md 7.4). |

## 22. Open questions

1. **A peg.** A counterparty's trade executes at the midpoint, so the quote is a limit and the
   realm's price is not held at it (6.7). A trade at the counterparty's own limit needs the
   `Clearing` to know which order is a counterparty's, for example a flag on `Order`, and a rule
   in `spec-clearing.md` 5.1 and 8. *Next step:* run EXT-S1; if the pull of beliefs toward the
   quote is too weak for a setting that wants a fixed price, CLR adds the flag and the rule.
2. **Gross flows.** The boundary flows are gross (5.2.3, 8.1). A reader of `BoundaryFlow` sees a
   large budget and a large return. **Answered:** OBS adds no net event or metric
   (`spec-observability.md` 4.4.7), and the net is `TradeBalance` of `spec-indicators.md` 9
   row 13; the derivation is in 8.2 and 15.5.
3. **A price by distance.** The quote is delivered at the gate (3.2.2). An edge's length or its
   status factor could change the price. *Next step:* TRN or INF, once a cost per distance unit
   exists.
4. **A supply curve inside one line.** Several tiers on one side would need several
   counterparties per node (3.1.3 gives a second node instead). *Next step:* add tiers if packs
   routinely use more than four nodes on a gate.
5. **Imports that are not fresh.** An import is born in the tick it is offered (10.4); an outside
   that sells old or worn goods, or buys only fresh ones, is not modelled. *Next step:* a field of
   the side, `Age` or `Condition`, if a pack needs it.
6. **A capacity slot.** The shock slot moves a price (6.5); a partial embargo would multiply a
   capacity. *Next step:* INT, with the other slots of N-091; a closed edge covers the full case.
7. **Credit and remittances.** The realm cannot borrow from the outside, and the outside pays
   nothing that is not a purchase (11.2, 12.3). *Next step:* CRD for debt, MON's grant for a
   remittance.
8. **Tariffs and tolls.** A levy on the trades with a counterparty belongs to FIS. The
   counterparty's trades are ordinary `trade` transfers, so FIS can attach to the agent ID range
   of 4.2.2. *Answered by FIS (`spec-fiscal.md` 6):* a tariff is a duty with `Scope` `import` or `export` on the trades of a gate region, charged to the realm's party, since a counterparty never pays (6.4 there); the sweep of 5.3 and the sectoral identity of 8.4 are unchanged, because the duty is applied at P4.1a, after the trade and before the sweep, and moves money from a household or pool account to a treasury, never from a counterparty account. This spec is unchanged.
9. **Where the walk's draw sits.** Phase 1, after the money flows (5.1.1). WLD may prefer another
   place; the order inside the step is all this spec needs.
10. **An outside node as a remote information source.** Not provided in version 1: INF records a
    rumour only from a market's published price, and an outside node has no market. The minimal shape
    would be INF accepting a **quote function** as a second record source: a pure read of the
    current mid of a line (6.1) keyed by (node, commodity), recorded like a published price, with the
    delay from the REG distance between the node and the region. EXT would then add a declaration
    on the node and the read function; the true price it returns is a true price of a place, as a
    market price is. *Next step:* INF, if a setting needs distant regions to hear of outside prices.
11. **Agent IDs shift.** Adding a node or a gate adds counterparties and shifts every household's
    agent ID, as adding a region shifts them by an estate (4.2.2). It changes tie-break keys and
    checkpoints between configurations. *Next step:* accepted; a pack that wants stable IDs adds
    its nodes first.
12. **A node's estate.** An outside node keeps an unused estate (3.1.1). *Next step:* REG may
    exempt marked regions if the agent ID rule is relaxed.

## 23. Where this lives in the code

| Concern | Home |
|---------|------|
| `External`, `OutsideNode`, `OutsideLine`, `OutsideSide`, `OutsideGate`, `SeasonFactor` and the `Modules.External` pointer | root package, `config.go` (types) and `defaults.go`; bounds in `bounds.go` |
| Validation, the sentinels `ErrOutsideNodeInhabited`, `ErrCrossedQuotes`, `ErrNotAGate`, `ErrOutsideNode` and the warnings | root package, `validate.go`, `errors.go` |
| The pure rules: the mid (6.1), the walk (6.4), the quotes (6.6), the wanted units with carry (7.1) and the cuts (7.3) | `internal/outside` (stdlib only, over primitive slices, converted inline at the call, architecture.md 2.9) |
| The counterparties, the tick steps (5) and the line state | root package, `external.go`, with the phase-1, phase-3 and phase-4 calls in `world.go` |
| The account kind Outside and the six reasons | `internal/ledger` (a kind and registry rows, built at construction from the configuration) |
| `SetOutsideShocks`, `Lines` | root package, `integration.go` for the first (`spec-integration.md` 5.1) |
| The events `OutsideTraded`, `OutsideEnded`, `OutsideShocked` | root package, `event.go` |
| Log keys `node`, `value`, `nodes`, `counterparties`, `drift` | `internal/logkeys` (key strings only) |

```go
// internal/outside: stdlib only, no root types. A sketch, not final.
// Mid returns the mid price and whether a clamp acted (6.1). All products use the
// checked 128-bit helper; ok is false on a result that cannot be represented.
func Mid(base, season, trendBP, tick, ticksPerYear, drift, floorBP, ceilBP, shock, maxPrice int64) (mid int64, clamped, ok bool)

// Walk returns the next drift (6.4); d is the draw already shifted into [-step, step].
func Walk(drift, revertBP, d, bound int64) int64

// Quote returns the limit price of a side at a gate (6.6); sell rounds up and buy down.
func Quote(mid, marginBP, scaleBP, maxPrice int64, sell bool) int64

// Want returns the wanted units of a tick and the new carry (7.1).
func Want(capacity, avail, carry int64) (units, newCarry int64)
```
