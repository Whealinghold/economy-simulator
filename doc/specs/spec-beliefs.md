# Spec: Price beliefs and their update rule

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-BEL-NNN` and
`NFR-BEL-NNN`, which tests cite (testing.md section 4). They are in section 16.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs `spec-clearing.md` for the outcome of an order and the market report
(sections 5.4, 5.5 and 7.3 there). It takes one plain input from production, the
unit cost of section 4.4, and does not wait for `spec-production.md`.

**Every number in sections 11 and 15 is a placeholder.** Nothing has been run,
bazaarBot has not been run, and no acceptance band exists (notes N-015, N-027).
No default here is claimed to be stable. The defaults are chosen to be
plausible and easy to change, and the slice must replace them with calibrated
values. The numeric checks quoted in this spec (5.4, 3.2.2) are runs of a
stylised Python model of these rules with those placeholders (`doc/specs/models/pb/`),
not runs of the library.

**Revised in round 3 (2026-10-10).** (1) The cost band now sits on a unit cost that
counts labour (`spec-production.md` 9; N-055): `CostFloor` falls from 10 000 to
7 000 and 3.2.2 says why. (2) The evidence rule of 5.4 compares the order's limit
price, not the centre, to the target, never moves a centre past its target, and has
a share parameter, `BeliefEvidenceShare` (N-042). Both signals are needed (5.5,
N-044). (3) The contract with `spec-agents.md` is named (4.4, 10): `Target` and
`Stock`, `Revise`, `OutcomeView.Draw`. (4) The seed table is
`Config.WarmStart.Prices` (8.4). (5) The price window is kept by the market, pushed
once per tick after clearing, and defined here (7). (6) Section 17 points to the
register, and section 18 is answered where it can be.

**What a downstream reader must re-check after this revision.**
`spec-agents.md` 7.3.6 and 7.6 (the `UnitCost` it passes now includes labour and is
0 until the region has an observed wage; the default `Revise` gains no new input
because `BeliefEvidenceShare` is a world tunable). `spec-viability.md` 8.2.3 (seeds
against the cost band: with labour counted the seed is at 1.00 of the unit cost, so
it lies between the new floor of 0.70 and the ceiling of 3.00; the diagnostic
`SeedAboveCostCeiling` should not fire for the prototype). The register entries
STB-L02 (the offset is smaller, 5.7% instead of 11.3% in the model, 5.4), STB-L04
(the proportional signal is kept and re-tested), STB-L09 (the freeze threshold moves
with `CostFloor` and the wage lag, 3.2.2), STB-L10 (the gain around a cycle is now
`CostFloor / 10 000` times the full-cost share) and STB-L52 (resolved by the
labour-inclusive cost). `spec-skill-barriers.md` 2.4 ("cost acts through exit")
still needs rewording (N-039).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

Sources: [bazaarbot-reference.md](bazaarbot-reference.md) sections 3, 7, 11 and
12 describe what the original does and where it is defective. The Doran and
Parberry paper it is based on **has now been read in full** (the journal
version, section 19); where it differs from the bazaarBot source, section 14
says so.

## 1. Purpose and scope

An agent does not know the price of a commodity. It holds a **price belief**: a
guess, with a stated uncertainty, from which it draws the limit price of its
orders. After its market clears, the belief is moved by what happened to the
order. This is the learning loop at the heart of the bazaarBot approach, and
most of the stability of the result depends on its details. This spec defines:

- what a belief is, the invariant it keeps and the bounds that anchor it
  (section 3);
- what an update is given (section 4);
- the update rule: asynchronous, once per order, bounded in step, driven by the
  evidence of the market and by the agent's stock (section 5);
- how a limit price is drawn from a belief (section 6);
- a short memory of recent market prices (section 7);
- how beliefs are seeded, including a warm start (section 8);
- what a belief means for an agent that stands for many households (section 9);
- how beliefs relate to the `Strategy` extension point (section 10).

It settles the four bazaarBot defects the roadmap assigns here: the missing
lookback window (item 7), the price-scale threshold (item 9), the range that can
invert (item 10) and updates per match (item 11) (section 14).

Out of scope: how an agent chooses the quantity of an order, its ideal stock and
its tradable set (`spec-agents.md`); how the unit cost of a commodity is
derived from a recipe (`spec-production.md`); what the world does with the
outcome of an order other than belief updates (`spec-clearing.md` 7.3); the
tick phases (`spec-world.md`); and the stability register itself
(`spec-stability.md`), to which section 17 contributes.

## 2. Principles

2.1. **A belief is data, owned by the core.** It is a small plain value held by
the agent, saved in a checkpoint, and read by anyone who needs it. The rule that
updates it is a function of its inputs and nothing else (5.8).

2.2. **Integer arithmetic, basis points.** Every quantity is `int64`. Every
ratio, step and probability is in basis points (1/10 000) and is applied as
`v * bp / 10_000` with a checked multiply (architecture.md 3.5). No floating
point. The paper that bazaarBot comes from reports that its results changed by
about 20% after 500 rounds when a perturbation of 10^-9 was added to one agent,
because of floating-point error (section 14); integers do not remove that
sensitivity, but they make it reproducible.

2.3. **A scale-free rule.** No threshold is an absolute price. Every comparison
is a fraction of a price in basis points (defect 9). The same rule works for a
commodity that costs 100 minor units and one that costs 100 000.

2.4. **Prices are anchored and bounded.** A belief can never leave an interval
whose ends come from the commodity's reference price and, where known, from the
cost of producing it (3.2). This is the "prices anchored to cost with bounds"
duty of the roadmap.

2.5. **Slow, asynchronous and bounded.** An agent adjusts only with a stated
probability, by at most a stated fraction of its belief, at most once per order.
Slowness is a stability mechanism, not an inconvenience: Lengnick's baseline
model gets its self-organisation from adaptive, bounded, probabilistic price
changes, and its experiments find the economy breaks down only when price
adjustment becomes too rare (section 19).

2.6. **Evidence beats memory.** When nothing has traded, the published price is
old. The update uses the price age and the best unmatched quotes
(`spec-clearing.md` 5.4 and 5.5) to move off a stale price, instead of trusting
the old number.

2.7. **Nothing branches on a commodity, role or recipe** (architecture.md 9.1).
The rule has no commodity-specific case. What differs between commodities is
data: the reference price and the unit cost.

2.8. **Beliefs never change a result by being observed.** Events and logs read
the outcome of an update; they do not draw randomness or alter state
(observability-and-logging.md 1.4).

## 3. The belief

### 3.1 State

A belief is held per agent and per commodity in the agent's tradable set. The
tradable set is defined by `spec-agents.md`; this spec requires only that a
belief exists for every commodity in it from the moment the agent is created, so
that the number of beliefs does not depend on the order of events (defect 5).

```go
// internal/beliefs: stdlib only, no root types (architecture.md 2.9).
type Belief struct {
	Centre int64 // the belief's best guess, in minor units per unit; at least 1
	Width  int64 // half-width of the uncertainty, in basis points of Centre
}
```

Two numbers, not a low and a high. bazaarBot keeps `[low, high]` and updates
each end separately, which lets them cross (defect 10). A centre and a
half-width cannot invert: the range is derived from them (3.3) and is ordered by
construction. Two `int64` per belief is the same memory as the original's two
floats per agent and commodity (section 14).

### 3.2 Bounds

The centre is kept inside `[Floor, Ceiling]`. The bounds are not stored: they
are computed from the commodity's data each time they are needed, so a change in
the cost of production moves them at once, and nothing about them is in a
checkpoint (section 13).

Inputs: `Ref`, the effective reference price of the commodity in the agent's
region (`spec-goods.md` 6.3), at least 1; and `UnitCost`, the caller's unit cost
(4.4), which counts labour at the observed wage (`spec-production.md` 9), or 0 when
unknown.

```
refLow    = max(1, Ref * RefBandLow / 10_000)
refHigh   = min(MaxPrice, Ref * RefBandHigh / 10_000)
costLow   = UnitCost * CostFloor / 10_000        (0 when UnitCost is 0)
costHigh  = UnitCost * CostCeiling / 10_000      (MaxPrice when UnitCost is 0)

Floor     = min(MaxPrice, max(refLow, costLow))
Ceiling   = max(Floor, min(refHigh, costHigh))
```

3.2.1. **The reference band** (`RefBandLow`, `RefBandHigh`) is an outer guard
rail around the calibration anchor: the reference price is the **anchor**, and the
band is asymmetric about it (from `RefBandLow` to `RefBandHigh` basis points of
it, `spec-goods.md` 6.3), not centred on it. It stops a belief running away to a
price a thousand times the anchor for a commodity whose cost is unknown. It is wide
by default (a tenth to ten times the reference price) because a shortage may
legitimately raise a price a long way; it is a placeholder, and open question 3
records why it is kept.

3.2.2. **The cost band** (`CostFloor`, `CostCeiling`) is the anchor to cost. With
a known unit cost, a seller's belief never falls below `UnitCost * CostFloor`,
so an agent does not sell far below what it costs to make (default: 70% of the
cost), and never rises above `UnitCost * CostCeiling`, so a producer does not
price itself out of any market by a margin larger than that (default: three times
the cost). This is Lengnick's rule that price is bounded relative to marginal
cost (`p_low = phi_low * mc`, `p_high = phi_high * mc`, with `1 < phi_low <
phi_high`, section 19), restated for price takers. Lengnick's multiples (1.025
and 1.15, as read from extracted text) belong to a model of firms with market
power and a fixed markup range; they are not adopted.

**The cost is the full cost, labour included** (`spec-production.md` 9, N-055).
That decides the defaults. At the zero-profit prices of a chain the price equals
the full unit cost (the prototype of `spec-viability.md` 15 gives price over cost
of 1.00 for grain, flour, dough, loaf and tool; with labour left out it gave 2.8,
3.7, 9.7, 8.4 and 19). Two consequences, both checked in the stylised model:

- *The ceiling is now sound.* A ceiling of three times a cost that includes labour
  leaves headroom of 3 over the equilibrium price. With labour left out it capped
  four of five goods below their equilibrium price, so a seller could not earn the
  wage. The default `CostCeiling` of 30 000 stays.
- *The floor must be below 100%.* At `CostFloor` 10 000 the floor equals the
  equilibrium price, so any lag in the wage that raises the cost puts the floor
  above what buyers will pay. In the model (one commodity, buyers capped at the
  equilibrium price 1 000, a shelf life of 3, six seeds, ticks 100 to 300), with the
  wage exact (cost 1 000), a floor of 10 000 gave trades in 0% of ticks and spoiled
  99% of the output; 9 000 traded in 94%; 8 000 and below in every tick. With the
  wage 10% high (cost 1 100) the floor had to be 8 000 or below to trade (97%), and
  with it 25% high (cost 1 250) 7 000 or below (98%; 8 000 gave 0%). A floor of
  7 000 therefore tolerates a wage that is up to about 40% too high. The loop
  price, income, wage, cost, floor, price (STB-L10) has gain `CostFloor / 10 000`
  where the floor binds: 1 at 10 000 (a wage-price spiral nothing damps), 0.7 at
  7 000. The default is 7 000, a placeholder. The price of the lower floor is that
  an agent may sell up to 30% below its full cost, which is a loss only if its own
  labour is valued at the wage; its cash cost, the inputs, is a small part of it.

The perishables freeze of STB-L09 and open question 1 are therefore not removed but
moved: the floor binds for a seller only when the market price is below 70% of its
full cost, which is a downturn, not the steady state.

3.2.3. **Precedence.** If the cost floor is above the reference band's upper
end (the cost has risen above ten times the reference), the floor wins and the
ceiling is raised to meet it. `Floor <= Ceiling` always holds. The event of
section 12 counts these conflicts.

3.2.4. A belief for a commodity the agent buys but does not make has a unit cost
of 0 and only the reference band. A merchant may supply its acquisition cost as
the unit cost: this spec does not care where the number came from.

### 3.3 The effective range and the invariant

The **effective range** is where an order's limit price is drawn (section 6):

```
half  = Centre * Width / 10_000                  (truncated)
Low   = max(Floor,   Centre - half)
High  = min(Ceiling, Centre + half)
```

**Invariant.** After every operation on a belief, and for every input:

1. `1 <= Floor <= Centre <= Ceiling <= MaxPrice`;
2. `BeliefWidthMin <= Width <= BeliefWidthMax`, and `BeliefWidthMax < 10_000`;
3. therefore `Floor <= Low <= Centre <= High <= Ceiling`, and `Low <= High`.

Item 3 follows from items 1 and 2: `Width < 10_000` gives `half < Centre`, so
`Centre - half >= 1`. The range cannot be empty, inverted or non-positive. This
is the definition of the range invariant that defect 10 asks for.

Because the bounds are recomputed from current data, a stored centre can fall
outside them without any update (the unit cost has risen). The belief is
**re-clamped** whenever it is read for a draw and whenever it is updated; a
re-clamp is not a step and is not limited by 5.6, because the floor it enforces
is the point of the floor.

## 4. Inputs to an update

An update happens for one order, after its market has cleared, and is a pure
function of the following. All are plain integers and flags, so the function
lives in `internal/beliefs` and the root converts at the call
(architecture.md 2.9).

### 4.1 The belief and the bounds

The belief before the update (3.1) and the bounds (3.2).

### 4.2 The outcome of the order

From `spec-clearing.md` 7.3 and FR-CLR-025: the side, the order's limit price,
the quantity ordered, and the units filled. `Unfilled = Quantity - Filled`. The
outcome is of the order as a whole, once, whatever the number of trades that
filled it. An order that was rejected at admission has no outcome and updates
nothing: it never reached the market.

The **fill share** is `Filled * 10_000 / Quantity` (truncated), in basis points.

### 4.3 The market report

From `spec-clearing.md` 5.4 and 5.5, for the order's market: the market price
`P`, the price age `Age`, and the best unmatched bid `B` and best unmatched ask
`A` with a flag each for "none".

Two facts about this report are used in section 5 and are guaranteed by the
matching rule (`spec-clearing.md` 4.3):

- when matching stops, every remaining bid is below every remaining ask, so for
  an unfilled bid with limit `b`, an unmatched ask `A` exists only if `A > b`;
- symmetrically, for an unfilled ask with limit `a`, an unmatched bid `B` exists
  only if `B < a`.

### 4.4 The agent's stock and the unit cost

Two inputs the caller supplies, because they belong to other specs:

- **Stock** and **Target**: the units of the commodity the agent holds, and the
  units it wants to hold (its **target stock**), both whole units, both
  totals for the agent (so both scale with its weight, section 9). Their ratio is
  the **stock ratio**, the inventory-to-demand ratio of the roadmap:
  `stock_ratio = Stock * 10_000 / Target` (basis points). Target is set by the
  strategy from the agent's recent demand: for a buyer of an input, the units
  its recipes will need over a cover period; for a seller, its recent sales over
  a cover period (Lengnick compares inventory with the most recent demand,
  section 19). `Target` of 0 means "no target", and the stock ratio is then
  treated as neutral (5.5). The definition of the target and of the cover period
  is `spec-agents.md` 7.3 (`Target`: the units to hold after this tick's upkeep,
  cover periods `CoverTicks`, `InputJobs` and `OutputCoverTicks`), and the
  definition of `Stock` is `spec-agents.md` 6.4.4: the units held after the trades,
  less the units reserved for this tick's upkeep, not below 0, so that a buyer
  holding exactly its target after eating has a ratio of 10 000. Both arrive in the
  `OutcomeView` of `Revise` (`Stock`, `Target`, `UnitCost`).
- **UnitCost**: money per unit of the commodity, an `int64` in minor units, which
  the caller (production, for a commodity the agent makes) derives from a recipe,
  the current prices of its inputs and the regional wage
  (`spec-production.md` 9.1, 9.3), so it includes labour. 0 means unknown, and it
  is 0 in a region with no observed wage yet. The strategy passes it only for a
  commodity the role can make (`spec-agents.md` 7.3.6). This spec assumes exactly
  that and nothing about how it is derived. If production passes a cost per
  execution, or a cost for a byproduct, it divides before calling.

### 4.5 One random draw

The only randomness the rule uses is one `uint64`, supplied by the caller, drawn
from the world's random source (architecture.md 7.7). Exactly one is drawn for
each outcome, whether or not it is used (5.2). The strategy receives it as
`OutcomeView.Draw` (`spec-agents.md` 6.4.4); the world, not the strategy, takes it
in the order of 5.1.1.

## 5. The update rule

### 5.1 When

5.1.1. All orders of a tick are placed before any market clears
(`spec-clearing.md` 7.2). After all markets have cleared, the world processes
the outcomes of that tick, in order of region, then commodity, then the order in
which the orders were admitted. This order is deterministic and is the order in
which the one draw per outcome is taken, so the random stream does not depend on
anything but the number and order of admitted orders.

5.1.2. Each admitted order gives exactly one update, however many trades filled
it. An agent with no order in a market gets no update there. This replaces the
original's update per matched trade plus one per unmatched order (defect 11).

5.1.3. The world applies the update when processing the outcome, before the next
tick's orders are placed. A belief changed here is the one the strategy sees at
the start of the next tick.

### 5.2 The adjustment gate

Each agent adjusts only some of the time, which keeps agents from moving in
step (asynchronous adjustment; Lengnick's firms change a price only with a
probability below one, section 19). The draw `u` of 4.5 is mapped to `[0,
10_000)` without rejection:

```
hi, _ := bits.Mul64(u, 10_000)
open  := hi < AdjustProb          // AdjustProb in basis points
```

If the gate is closed, the centre and the width are unchanged, apart from the
re-clamp of 3.3. The mapping uses one multiplication and no loop, so exactly one
`uint64` is consumed per outcome (the bias is at most `10_000 / 2^64`). A
closed gate also means that no evidence is processed: the agent simply did not
look this time.

### 5.3 The step

The **belief step** is `BeliefStep`, in basis points of the centre. One unit of
adjustment is

```
Δ = max(1, Centre * BeliefStep / 10_000)         (truncated)
```

The `max(1, ...)` stops a belief freezing when the product truncates to zero.
It is a backstop, not the design: the resolution floor of `spec-currency.md` 3.1
requires every tradable commodity's reference price to be at least
`ceil(10_000 / BeliefStep)`, which makes `Centre * BeliefStep / 10_000 >= 1` at
the reference price. `BeliefStep` is therefore the parameter named in
`spec-config.md` 7.4 and `spec-currency.md` 3.1. Below the floor (a belief can
fall to a tenth of its reference price) the step is one minor unit, which is a
coarser fraction of the price but never zero.

### 5.4 Evidence

The evidence component moves the centre toward a **target price** `T` that the
market has shown to be reachable. There is at most one target. It is chosen as
follows.

A price `P` is **fresh** when `Age <= BeliefMaxAge`. A stale price is not
evidence: it is the last price at which anything traded, carried forward
(`spec-clearing.md` 5.4).

For a **bid**:

- If it was **not fully filled** (`Unfilled > 0`): the candidates are `A`, if
  there is an unmatched ask, and `P`, if it is fresh and `P > limit`. The target
  is the smaller candidate. If there is none, there is no target. A bid that
  could not buy learns the cheapest price at which something was offered.
- If it was **fully filled**: the target is `P`, if it is fresh and `limit - P >
  Centre * BeliefSignificance / 10_000`. Otherwise there is no target. A bid
  that bought everything at a limit well above the market price learns that it
  overbid. The significance test is defect 9's replacement (section 14).

For an **ask**, symmetrically:

- If it was **not fully filled**: the candidates are `B`, if there is an
  unmatched bid, and `P`, if it is fresh and `P < limit`. The target is the
  larger candidate. If there is none, there is no target.
- If it was **fully filled**: the target is `P`, if it is fresh and `P - limit >
  Centre * BeliefSignificance / 10_000`. Otherwise there is no target.

**The evidence move** compares the target to the order's **limit price**, not to the
centre. The order failed or succeeded at its limit; the centre is only where the
limit was drawn from, and a limit drawn far from the centre says little about the
centre. Let `Δe = max(1, Δ * BeliefEvidenceShare / 10_000)`, `gL = T - Limit` and
`gC = T - Centre`:

```
evidence_move = 0                                   if there is no target, or gL = 0, or gC = 0,
                                                    or gL and gC have different signs
evidence_move = sign(gC) * min(|gL|, |gC|, Δe)      otherwise
```

So the move points toward `T`, is never larger than the distance from the centre to
`T` (the centre cannot pass the target), is never larger than the distance from the
limit to `T`, and is zero when the failure cannot be blamed on the centre: an
unfilled bid whose centre is already above the cheapest ask lost because its limit
was drawn low, and lowering its centre toward that ask would be the wrong sign. At
most `Δe` is moved.

**Why (N-042).** The first draft moved the centre by `clamp(T - Centre, -Δ, +Δ)`.
In the stylised model of `spec-stability.md` 6.2 (20 sellers and 20 buyers, elastic
flows with both elasticities 0.5, eight seeds, ticks 400 to 800) that settled the
price at 888 (883 to 894) against 1 001 with the evidence off, 11.3% below the
price at which supply equals demand, with 242 units of need unmet per tick of 2 000.
A trace showed the cause: with the target taken from the centre, a failing order
often moved its centre the wrong way (for 'ask, filled' 8 929 moves were down and
2 318 up). Re-running the same model:

| Evidence rule | Mean price (median, range) | Unmet need per tick | Relative std | Shock return (ticks) | Shock peak |
|---------------|----------------------------|---------------------|--------------|----------------------|------------|
| first draft (target against the centre, step `Δ`) | 888 (883 to 894) | 242 | 4.8% | 23.5 | 1.69 |
| against the limit, step `Δ` | 930 (928 to 934) | 147 | 3.7% | 26 | 1.63 |
| against the limit, guarded as above, step `Δ` | 903 (895 to 906) | 218 | 4.5% | 23 | 1.67 |
| **guarded, step `Δ/2` (this rule, `BeliefEvidenceShare` 5 000)** | **944 (940 to 949)** | **118** | **3.6%** | **30** | **1.56** |
| against the limit, step `Δ/2`, no guard | 952 (948 to 956) | 101 | 3.7% | 38 | 1.61 |
| evidence off (stock only) | 1 001 (1 000 to 1 002) | 0 | 4.8% | 98 | 1.61 |

(The shock is consumption up 50% for 100 ticks, six seeds, as in 6.2 run 3; "return"
is the median ticks after the shock ends until the price stays within 10% of its
pre-shock mean.) The offset falls from 11.3% to 5.7% and the need unmet from 12% to
5.9%, and the recovery stays much faster than the stock signal alone gives (30
against 98 ticks). A smaller share removes more of the offset and slows the recovery
(a quarter of `Δ` gave 970 and 53 ticks, unguarded). The default of 5 000 is a
placeholder chosen between those. **The offset is reduced, not removed.** Both the
model's quantity rule (sellers ask all their stock, buyers bid only their shortfall,
which `spec-agents.md` 7.3.5 also does) and the pairing of the highest bids with the
lowest asks tilt the unfilled orders' evidence downward, and the remainder is
probably a property of both. Open question 12 gives the next step.

When nothing has traded for
longer than `BeliefMaxAge`, only the unmatched quotes can set a target, and that
is how a belief moves off a stale price: an unfilled bid and an unfilled ask in
the same market are each pulled toward the other side's best quote, and the
distance between them closes (FR-BEL-034).

If a side of the market is empty (no unmatched quote and no fresh price), there
is no target, and the evidence move is 0. Raising a bid does not conjure a
seller.

### 5.5 Stock

The stock component pushes the centre by one unit of adjustment according to how
the agent's stock compares to its target, whichever side the order was on:

```
stock_ratio = Stock * 10_000 / Target            (Target >= 1)
v = +1  if stock_ratio < BeliefStockLow          (short of stock)
v = -1  if stock_ratio > BeliefStockHigh         (long of stock)
v =  0  otherwise, and whenever Target is 0
```

The reasons: a buyer that is short must pay more to get goods and one that is
long need not chase; a seller that is short can hold out for more and one that is
long must cut. Both are the inventory rule of Lengnick (price considered up when
inventory is below the lower bar, down when above the upper bar) and, in the
failure case, of the paper (an agent very low on inventory whose bid was rejected
bids more aggressively, section 14). The stock component applies to the
centre `Δ`, so it moves the price even when the market has shown no target.
Where there is no supply at all this lets a shortage raise a price until it
reaches the ceiling, which is what a shortage should do and is why the ceiling
exists.

**Both signals are needed (N-044).** The stock ratio is the integral of the flow
imbalance, so a rule driven by it alone responds late: in the labour model of
`spec-stability.md` 6.1 a price moved only on the stock ratio did not converge (a
relative standard deviation of 6% to 29%, growing with `BeliefStep`; STB-L04). In the
belief model of 6.2 the stock signal alone does settle at the clearing price (1 001),
but it recovers from a demand shock in 98 ticks against 30 with the revised evidence
rule added (5.4), and the evidence rule is the signal that answers the imbalance of
this tick and not the accumulated one. The evidence rule alone has no restoring force
on the level and, as written in the first draft, offset it (5.4). So the stock rule
fixes the level and the evidence rule gives the speed and damping. Neither is removed
without re-running S01 and S04 (`spec-stability.md` 8).

### 5.6 The move

```
move   = evidence_move + v * Δ                    // |move| <= Δe + Δ <= 2Δ
Centre = clamp(Centre + move, Floor, Ceiling)
```

5.6.1. The two components are added. Before the clamp the move is at most
`Δe + Δ` in size, where `Δe` is `Δ * BeliefEvidenceShare / 10_000` (at most `Δ`): the
evidence part and one unit from the stock. This is the bounded step of the roadmap,
and it is a number of basis points of the centre (at most `2 * BeliefStep`), not of
an absolute price.

5.6.2. The clamp may reduce the move; it never lengthens one made from a belief
that was inside the bounds. (A re-clamp of a belief that was outside the bounds,
3.3, is a separate act.)

5.6.3. At the defaults (a step of 2%, an evidence share of one half and a gate of
one half), a belief moves by at most 3% in an adjustment and, on average, by at
most 1.5% a tick for an agent that has an order each tick.

### 5.7 The width

If the update was made (the gate was open), the width changes by a fixed step in
basis points:

- `fill share >= 5 000`: `Width = Width - BeliefWidthStep` (narrower: the agent
  was right more than not);
- otherwise: `Width = Width + BeliefWidthStep` (wider: the agent was wrong);

clamped to `[BeliefWidthMin, BeliefWidthMax]`. This is the paper's rule of
shrinking the range when at least half of an offer is filled and growing it
otherwise (section 14). Width is a fraction of the centre, so it scales with the
price. The minimum width keeps some exploration: a belief of width 0 would never
test a price other than its centre.

### 5.8 Properties

These follow from the rules above and are the basis of the tests:

1. **Pure.** The new belief and the result depend only on the arguments of 4
   and `u`. There is no hidden state, no map, no time and no float.
2. **Bounded.** The invariant of 3.3 holds after the update for any inputs.
3. **Bounded step.** Before the clamp, `|new Centre - old Centre| <= Δe + Δ <= 2Δ`.
4. **Toward the evidence.** The evidence part points toward the target, never passes
   it measured from the centre, and is zero when the limit and the centre lie on
   opposite sides of the target.
5. **Stationary without evidence.** With a closed gate, or with no target and a
   stock ratio inside the band, the centre does not change (apart from a
   re-clamp).
6. **Scale-free.** Multiplying `Centre`, `Floor`, `Ceiling`, every price in the
   inputs and the limit by a factor `k` (so that no truncation occurs) multiplies
   the new centre by `k` and leaves the width unchanged.
7. **Constant draw.** Exactly one `uint64` is consumed per outcome (the caller's
   responsibility, FR-BEL-008).

## 6. Drawing a limit price

The default strategy draws the limit price of an order uniformly from the
effective range (3.3), as bazaarBot does with its belief range. The helper is
part of this spec because the range is:

```
n   = High - Low + 1
hi, _ := bits.Mul64(u, uint64(n))
limit = Low + int64(hi)
```

One `uint64` is consumed per draw, including when `n` is 1, so the number of
draws depends only on the number of orders. The strategy may use another rule
(10.2) and then draws what it needs from the same source.

The result is a price between `Floor` and `Ceiling`. A seller whose cost floor
is above the market price therefore posts asks at or above its floor and sells
nothing until the market comes to it (open question 1).

## 7. The market price window

bazaarBot gives each agent an `observedTradingRange` that is the all-time
minimum and maximum of the agent's own trades, an array that grows without limit,
and a market average over `lookBack` (15) entries (defect 7). The paper
describes the market's public information as an average "within some
user-defined window" and gives a window of 8 to 15 rounds for a different
statistic, profitability; it does not window an agent's own range (section 14).
Neither says what a good window is.

Decision: the memory is **per market, not per agent**, and it is windowed.

7.1. Each market keeps the **price window**: the market prices of its last
`PriceWindow` ticks in which units traded (price age 0). The window is a ring held
by the market in the same record as its published price and price age
(`spec-clearing.md` 5.4), but it is **not** part of what `Clearing` sees or returns:
the clearing rule never reads or writes it. This spec owns its definition, its
statistics (7.2), the parameter `PriceWindow` and the checkpoint state (13). A tick
with no trade adds nothing, so a stale carried-forward price is never counted twice.
The window starts empty.

7.2. From the window, with the rounding stated: the **window mean**, the mean of
the stored prices rounded half to even (`spec-clearing.md` 5.2); the **window
low** and **window high**, the minimum and maximum. All three are absent while
the window is empty. The window has no more than `PriceWindow` entries, so its
memory is fixed and does not grow with the length of a run or the number of
agents.

7.3. **Position in the window.** The strategy may use where a price lies in the
window to scale an order's quantity (the bazaarBot "favorability"):

```
position_bp = (price - low) * 10_000 / (high - low)        clamped to [0, 10_000]
position_bp = 5_000                                         when high == low
```

An agent therefore sees the same public range as every other agent rather than a
private one, which costs nothing per agent and is the form the paper's text
describes. How the strategy uses it is `spec-agents.md`'s.

7.4. The window is pushed by the world once per market per tick, after the market
clears and before the outcomes are processed, and only if units traded in the tick
(the market price then has age 0), so every update of a tick sees the same window.
It is state and goes in a checkpoint (section 13).

## 8. Seeding and the warm start

8.1. **The seed price** of a commodity in a region is the price a new belief and
a new market start from. By default it is the effective reference price
(`spec-goods.md` 6.3), or the entry of `Config.WarmStart.Prices` for that region
and commodity when one exists (8.4). It is also the market's initial published
price: the price before the first trade is this seed price, with price age 0
(`spec-clearing.md` 5.4; N-038), so the beliefs and the market start from the same
number.

8.2. **Initial agents.** A belief starts at `Centre = clamp(seed price, Floor,
Ceiling)` and `Width = BeliefWidthInit`. The bounds use the agent's unit cost
at that time. No randomness is used: two agents of one role start with the same
belief, and differ after their first draws and updates. bazaarBot starts a belief
at half to one and a half times the price (a width of 50%); the default width
here is 20% (section 14).

8.3. **Agents that enter later** (a replacement, a newborn, a migrant) start at
`Centre = clamp(window mean, Floor, Ceiling)` if the window is not empty and
the market price otherwise, with `BeliefWidthInit`. An agent that changes role (`spec-spawner.md` 4.2) keeps the belief of every commodity that is in the tradable sets of both roles, with its centre and width unchanged apart from the re-clamp of 3.3 to the bounds of the new role's unit cost; a commodity new to its set gets a belief by this paragraph's rule; a belief for a commodity that left the set is dropped. This is bazaarBot's rule of
starting a new agent from the average price over its lookback (bazaarbot-reference.md section 8),
with the window of section 7. Whether a migrant keeps its old beliefs or is
reseeded in the new region is open (open question 7); the default is to reseed.

8.4. **The hook for a steady state.** `spec-viability.md` solves for the steady
state of a chain and so gives a price for each commodity in each region. The seed
price of 8.1 is given by that table, `Config.WarmStart.Prices`, a list of
(region, commodity, price) entries sorted by region and commodity
(`spec-viability.md` 8.1, 8.2), absent by default. When an entry exists it replaces
the reference price both as the starting belief and as the initial published price,
so that a run begins near its equilibrium rather than from beliefs that are wrong by
design; a commodity without an entry uses the effective reference price. An entry is
checked against the reference band of 3.2 and the bounds of `spec-config.md` 4.1 by
validation (`ErrWarmStartPrice`, `spec-viability.md` 8.2.2). The reference band and
cost band stay anchored on the reference price and the cost, not on the seed
(`spec-viability.md` 8.2.1). Nothing in this spec depends on how the table is
produced. Newcomers do not start from the table (8.3).

## 9. Cohorts

An agent of weight `w` stands for `w` identical households that act in lockstep
(`spec-scale.md` 2.2). For beliefs:

9.1. **One belief per agent and commodity**, whatever its weight. The belief is
a price, and a price is per unit: it is not multiplied by `w`
(`spec-scale.md` 4.4, FR-SCL-010).

9.2. **Quantities are totals.** The order quantity, the units filled, the stock
and the target are all `w` times the household amounts, so the fill share and
the stock ratio are the same as those of one household in the same
circumstances (`spec-scale.md` 4.3 makes the same point for the welfare ratio).

9.3. **One gate draw per outcome**, independent of `w`. A cohort adjusts or does
not adjust as a whole. In expectation the share of households that adjust is
`AdjustProb`, as it would be if each household drew for itself; the variance is
larger, because the households of a cohort move together. This is the lockstep
that `spec-scale.md` 5.1 predicts and N-014 asks the stability register to
record (section 17). The method of N-005 (a carried remainder) is not used here:
a belief update is a decision by one agent, not a chance event happening to each
household, and a cohort is one decision-maker.

9.4. The first consumer's world needs no weight above 1 (N-030), so this
section is about consumers that scale.

## 10. Beliefs and the Strategy

10.1. **Who owns what.** The belief value is data owned by the core (2.1). The
default update rule is in `internal/beliefs`. `Strategy` decides an agent's bids
and asks and, through `Revise`, its belief updates (architecture.md 4.3: "decides an
agent's work (`Work`), its bids and asks (`Trade`) and its belief updates
(`Revise`); the world checks what each returns"). So:
the default strategy reads the belief to draw limit prices (section 6) and the
world, when it processes an outcome (5.1.3), calls the strategy's `Revise` method
(`spec-agents.md` 6.2): `Revise(o *OutcomeView, rng *rand.Rand) Belief`. The
`OutcomeView` (`spec-agents.md` 6.4.4) carries the order and its outcome, the current
belief and bounds, the market report, `Stock`, `Target`, `UnitCost` and `Draw`, the
one `uint64` the world drew for this outcome (4.5). The default is
`DefaultStrategy.Revise`, which calls `beliefs.Update` and nothing else
(`spec-agents.md` 7.6).

10.2. **A strategy may supply its own rule.** A consumer's strategy may ignore
the default update and compute its own new belief, or ignore beliefs altogether
and price its orders another way. What it receives and returns follows
architecture.md 7.7: it gets the world's random source as a parameter and may
not keep it. It reads the belief and the inputs of section 4 but does not write
the belief store directly.

10.3. **The world protects the invariant.** Whatever belief a strategy returns,
the world clamps it to the invariant and the bounds of section 3 before it is
stored, and counts the clamps in the event of section 12. A faulty or
adventurous strategy can therefore not place a centre outside its bounds or
invert a range, in the same way the world checks the list of trades from a
`Clearing` (`spec-clearing.md` 8.2). The step bound of 5.6 is a property of the
default rule, not an enforced limit, so a strategy may jump.

10.4. **No new extension interface.** The update rule is not one of the four
sanctioned interfaces (architecture.md 4.3), and this spec does not propose a
fifth. Rule 4.2 asks for a real need first: the table already assigns "belief
updates" to `Strategy`, a strategy can call or replace the default function, and
there is no second implementation of a separate `Beliefs` interface to justify
one. A consumer who wants a different rule writes a strategy.

10.5. **The default function is exported only through the strategy.** The
internal functions of section 20 are not part of the public API
(architecture.md 2.8). A consumer writing a strategy that wants the default rule
calls `DefaultStrategy.Revise`, which is exported and stateless; its type holds a
`DefaultStrategy` in a named field and delegates, since a public struct may not embed
(architecture.md 4.7; `spec-agents.md` 6.7). This is the wrapper this spec asked for.

## 11. Parameters

All are World-level tunables with a default, a bound, a doc line and an owner
(architecture.md 9.5 to 9.7). **Every default is a placeholder** (the preamble;
N-015). The bounds are validation bounds checked at construction. A default is
defined once, in `defaults.go`.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| World | `BeliefStep` | one unit of adjustment, basis points of the centre (5.3); sets the resolution floor `ceil(10_000 / BeliefStep)` of `spec-currency.md` 3.1 | 200 | 1 to 2 500 |
| World | `BeliefAdjustProb` | probability that an agent adjusts after an outcome, basis points (5.2) | 5 000 | 1 to 10 000 |
| World | `BeliefSignificance` | how far a limit price must be beyond the market price, as basis points of the centre, before a fully filled order is corrected (5.4) | 500 | 0 to 5 000 |
| World | `BeliefStockLow` | stock ratio below which the stock component pushes up, basis points of the target (5.5) | 5 000 | 1 to `BeliefStockHigh` - 1 |
| World | `BeliefStockHigh` | stock ratio above which the stock component pushes down, basis points of the target | 15 000 | `BeliefStockLow` + 1 to `MaxBasisPoints` |
| World | `BeliefMaxAge` | ticks for which a market price is still fresh evidence (5.4) | 3 | 0 to 1 000 |
| World | `BeliefEvidenceShare` | the largest evidence move as a share of one unit of adjustment `Δ`, basis points (5.4) | 5 000 | 1 to 10 000 |
| World | `BeliefWidthInit` | starting half-width of a belief, basis points of the centre (8.2) | 2 000 | `BeliefWidthMin` to `BeliefWidthMax` |
| World | `BeliefWidthMin` | smallest half-width (5.7) | 200 | 1 to `BeliefWidthInit` |
| World | `BeliefWidthMax` | largest half-width; must keep the range ordered (3.3) | 5 000 | `BeliefWidthInit` to 9 000 |
| World | `BeliefWidthStep` | change of the half-width per adjustment, basis points (5.7) | 250 | 1 to `BeliefWidthMax` |
| World | `CostFloor` | the lowest belief for a commodity with a known unit cost (labour included), basis points of that cost (3.2.2) | 7 000 | 1 to `MaxBasisPoints`; a value of 10 000 or more draws the warning `cost_floor_at_full_cost` |
| World | `CostCeiling` | the highest belief for such a commodity, basis points of the cost | 30 000 | `CostFloor` to `MaxBasisPoints` |
| World | `RefBandLow` | the lowest belief, basis points of the effective reference price (3.2.1) | 1 000 | 1 to 10 000 |
| World | `RefBandHigh` | the highest belief, basis points of the effective reference price | 100 000 | 10 000 to `MaxBasisPoints` |
| World | `PriceWindow` | ticks of trading kept in a market's price window (section 7) | 15 | 1 to 1 000 |

Notes on the defaults, none of which is calibrated:

- `BeliefStep` 200 is the same order as Lengnick's maximum price change of 2% per
  adjustment (`vartheta` 0.02, read in the paper's Table 1 and equation 10; the
  steps there are uniform in `[0, 2%]`, so their mean is half of this). With the
  gate it makes a belief move by a few percent a tick at most. The floor it
  implies is 50 minor units, below the 100 that `spec-currency.md` 4.3 gives the
  RPG preset's smallest denomination.
- `BeliefAdjustProb` 5 000 is a guess. Lengnick's probability of changing a price
  appears in the extracted text as 0.75, but the symbol was lost in extraction
  and the mapping is unverified (section 19), so it is not used.
- `BeliefMaxAge` 3, the stock band (50% to 150% of the target) and the width
  figures are guesses. The original's constants are in the table of section 14.
- `PriceWindow` 15 is bazaarBot's `lookBack` default, and the paper's profitability
  window of 8 to 15 rounds is in the same range. Neither was tuned for this use.
- `CostFloor` 7 000 and `CostCeiling` 30 000 are placeholders chosen from the check in
  3.2.2: with labour counted the equilibrium price equals the full unit cost, so a
  floor at 100% freezes the market when the wage lags, and 7 000 tolerated a wage
  about 40% too high in the model. It is the most consequential default (open
  question 1).
- `BeliefEvidenceShare` 5 000 is a placeholder chosen between the offset and the
  recovery speed in the table of 5.4; nothing else supports the number.

Bound derivations: the products computed are at most `MaxPrice * MaxBasisPoints
= 10^9 * 10^6 = 10^15`, which fits `int64`, so the checked helper is not needed
for the bounds themselves; `Stock * 10_000` and `Quantity * 10_000` use it,
because stocks may reach `MaxQuantity`. Validation also requires that
`ceil(10_000 / BeliefStep)` does not exceed the cheapest tradable commodity's
reference price in any region, which is `spec-currency.md` 3.1.

## 12. Observability

Everything here follows observability-and-logging.md. A belief update is a state
transition; one event per agent would be heavy, so the event is aggregated per
market, as `BoundaryFlow` is for the ledger.

12.1. **Event `BeliefsUpdated`.** One per market per tick that had at least one
outcome, emitted after the outcomes of the tick are processed, in order of
region and then commodity. Fields besides `Tick`: region, commodity; `Outcomes`
(orders processed); `Adjusted` (gates open); `Raised` and `Lowered` (centres that
increased or decreased); `AtFloor` and `AtCeiling` (centres that ended on a
bound); `Conflicts` (updates for which the cost floor exceeded the band, 3.2.3);
`Clamped` (beliefs a strategy returned that the world had to clamp, 10.3);
`MeanCentre`, `LowCentre` and `HighCentre` (over the beliefs processed in this
tick: the mean rounded half to even, the minimum and the maximum). No map and no
slice; all fields are integers. It lets a designer see why a price moved
(vision.md goal 4) and lets an observer build the series of belief centres.

12.2. **Logs.** All are guarded by `Enabled` and have constant messages.

- `Debug`, "belief updated", per update: `commodity`, `agent`, `price` (the new
  centre), `reason` (one of `gate_closed`, `moved`, `held`, `at_floor`,
  `at_ceiling`).
- `Warn`, "price beliefs at bound", **per market**, never per agent: emitted when
  more than half of a market's outcomes in a tick end on a bound
  (`AtFloor + AtCeiling > Outcomes / 2`), and that has held for `BoundRun`
  consecutive ticks (a count the market keeps, section 13), at the tenth tick and
  every tenth after. Attributes `region`, `commodity`, `bound` (`floor` or
  `ceiling`, the one with more), `agents` (the number at that bound). So the volume is
  at most one line per market per ten ticks, bounded by the configuration
  (observability-and-logging.md 3.2.2). The line is actionable: it names where, which
  bound and how many.
- No `Info` and no `Error` line. An update that cannot be applied does not exist:
  every input is a bounded integer, and an input outside its bound is a
  configuration error caught at validation.

12.3. **Log keys.** `region`, `bound` (`floor` or `ceiling`) and `agents` are in the
table of observability-and-logging.md 3.3.2. The `reason` values above are constants
in `internal/logkeys`.

12.4. **No new metric** is defined here. The metrics adapter can build a gauge of
the mean centre per commodity from `BeliefsUpdated`; its name belongs to
`spec-observability.md`.

12.5. With the default no-op logger and observer, the update path allocates
nothing: events are constructed only when an observer was supplied
(observability-and-logging.md 4.5) and the update works on values
(NFR-BEL-001).

12.6. Observation does not change results. The draw of 4.5 is made whether or
not an observer is attached, and no log or event code draws randomness.

## 13. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002, consequences). This spec's:

- for each agent, in agent ID order, and for each commodity in its tradable set,
  in commodity ID order: the `Centre` and the `Width` (two `int64`);
- for each market, in region then commodity order: the price window (the count,
  the position of the next write, and the prices in ring order, up to
  `PriceWindow`) and `BoundRun`, the count of consecutive ticks used by the
  `Warn` of 12.2. The window lives in the market's record beside the published
  price and its age (`spec-clearing.md` 5.4), which are `spec-clearing.md`'s state;
  the window and `BoundRun` are this spec's.

It does **not** contain: the bounds `Floor` and `Ceiling` and the effective
range (all are computed from configuration and the caller's unit cost); the
gate draws (the state of the random source is saved by the world); the stock
ratio (computed from the ledger and the target). All is plain data with no maps,
in a fixed order, so equal states give equal bytes. `BoundRun` is diagnostic and
affects no result; it is saved so that a resumed run logs as the original did.
The seed table of 8.4, if present, is configuration.

## 14. Differences from bazaarBot and from the paper

The paper (section 19, journal version) and the repository's source do not agree
in detail. The source is what bazaarbot-reference.md describes; the paper
describes a rule in pseudocode with different mechanics and without the source's
constants. This spec follows the paper's ideas and the source's structure where
they agree, and its own integer rule where neither is adequate.

| Matter | bazaarBot source | The paper | This spec |
|--------|------------------|-----------|-----------|
| Belief | `[low, high]` floats; ends move separately | a lower and an upper bound, translated, shrunk or widened; a uniform draw inside | `Centre` and `Width` (basis points), a range derived from them: it cannot invert (defect 10, 3.3) |
| Update trigger | once per matched trade and once per unmatched order (defect 11) | text: "after each of the agent's offers has been resolved"; the pseudocode calls the update inside the trade loop and again for the remainder | once per admitted order, after its market clears (5.1.2) |
| Success rule | shift by half the gap to the market price if the gap exceeds an absolute 0.25 (`SIGNIFICANT`), then narrow by 5% of the mean (`wobble`) | at least half filled: move limits inward by a tenth of the upper limit | fill share of at least 50%: narrow by `BeliefWidthStep`; correct the centre only for a limit beyond the market price by `BeliefSignificance` basis points of the centre (5.4, 5.7) |
| Failure rule | shift by half the gap to the market price; double `wobble` if stock is very low or very high; else shift again by the last round's ask-bid imbalance (`SIG_IMBALANCE` 0.33); then widen | translate toward the mean by displacement, a fifth of the mean, 110% of the overbid, depending on supply, demand and filled share; very low stock and a rejected bid: translate up by the displacement | move toward the best unmatched quote or the fresh market price by at most `Δe`, half of `Δ` by default, measured from the limit and never past the target from the centre; the stock component adds `Δ` (5.4, 5.5); widen by `BeliefWidthStep` |
| Significance threshold | absolute 0.25, so it depends on the price scale (defect 9) | not stated as a constant | `BeliefSignificance`, basis points of the centre (2.3) |
| Imbalance signal | last round's total ask and bid units, threshold 0.33 | current round's supply against demand | the unmatched quotes and the stock ratio replace it; a unit-count imbalance is not used (open question 6) |
| Stock signal | `LOW_INVENTORY` 0.1 and `HIGH_INVENTORY` 2.0 of the ideal stock, used only to double `wobble` | "inventory < 1/4 capacity" for an aggressive bid | `BeliefStockLow` and `BeliefStockHigh` of the target, moving the centre by one `Δ`, on every outcome (5.5) |
| Lookback | `lookBack` 15 ignored when the agent's range is built: all-time minimum and maximum of its own trades, an array without limit (defect 7) | public averages over "some user-defined window"; an agent's own range is not windowed | a per-market window of `PriceWindow` traded ticks (section 7); no per-agent trade history |
| Minimum price | 0.01 absolute | not stated | `Floor`, at least 1 minor unit and normally the reference band or the cost floor (3.2) |
| Initial belief | `[0.5 p, 1.5 p]` of the average price | not stated | `[0.8 c, 1.2 c]` at the default width, from the seed price (8.2) |
| Cost | none: price is independent of the agent's cost; cost acts only through exit | "No seller will agree to sell a product below the cost to produce that product" (a statement of intent) | the cost floor (70% of the full cost, labour included) and ceiling (300%) (3.2.2) |
| Stale prices | the market price carries forward, updates only on a trade | "there will be no price updates" while nothing trades (the forest fire experiment) | price age and unmatched quotes move beliefs off a stale price (5.4) |
| Randomness | `Math.random`, unseeded | Mersenne Twister | one `uint64` per outcome and per draw from the world's source (5.2, 6) |

The paper also confirms two points that bear on the other specs and are reported
in the handoff: its offer-resolution pseudocode has no check that the bid reaches the
ask (so item 1 of the defects is in the paper as well as in the source,
`spec-clearing.md` open question 1), and it reports that a perturbation of 10^-9 of
one currency unit in one agent changed the price of some commodities by about 20%
after 500 rounds and bankrupted agents that would otherwise have survived. The
paper's authors attribute this to floating-point discretisation and call it
desirable; for this library the same sensitivity is a reason to expect golden
files to change with any change to the rule (testing.md 7.2), and not a reason to
keep floats.

## 15. A worked example

The parameters are the defaults of section 11, except where stated. A market in
which the effective reference price `Ref` is 1 000. Every gate in the example is
open. `Δe` is `Δ * 5 000 / 10 000`, truncated, at least 1 (5.4).

**Bounds.** With `UnitCost` 0: `refLow = 1 000 * 1 000 / 10 000 = 100`, `refHigh
= 1 000 * 100 000 / 10 000 = 10 000`; `Floor` 100, `Ceiling` 10 000.

**Case A: a bid that bought nothing, in a stale market.** Belief `Centre` 1 000,
`Width` 2 000. The bid was for 10 units at a limit of 940 and filled 0. The
market price is 1 000 with an age of 5 (stale, `BeliefMaxAge` is 3), there is an
unmatched ask at `A` = 1 100 and no unmatched bid. The agent holds 2 units
against a target of 10.

| Step | Value |
|------|-------|
| `Δ`, `Δe` | `max(1, 1 000 * 200 / 10 000) = 20`; `Δe = 10` |
| Candidates | `A` = 1 100; `P` is stale, so not a candidate. Target `T` = 1 100 |
| Gaps | `gL = T - limit = 160`; `gC = T - Centre = 100`; same sign |
| Evidence move | `+min(160, 100, 10) = +10` |
| Stock ratio | `2 * 10 000 / 10 = 2 000 < 5 000`: short, so `v` = +1, move +20 |
| New centre | `1 000 + 30 = 1 030` (inside `[100, 10 000]`) |
| Width | fill share 0, so `2 000 + 250 = 2 250` |
| New range | `half = 1 030 * 2 250 / 10 000 = 231`; `Low` 799, `High` 1 261 |

A stale price of 1 000 did not hold the belief: the unmatched ask pulled it up.

**Case B: an ask that sold everything, below the market.** Belief `Centre` 1 000,
`Width` 2 000. The ask was for 4 units at a limit of 940 and filled 4. The market
price is 1 010 with an age of 0. The agent holds 8 against a target of 8.

| Step | Value |
|------|-------|
| `Δ`, `Δe` | 20, 10 |
| Significance | `P - limit = 70`; the threshold is `1 000 * 500 / 10 000 = 50`; 70 > 50 |
| Target | `T` = 1 010 (fresh). `gL` = 70, `gC` = 10 |
| Evidence move | `+min(70, 10, 10) = +10` (the centre reaches the target and does not pass it) |
| Stock ratio | `8 * 10 000 / 8 = 10 000`: inside the band, `v` = 0 |
| New centre | 1 010 |
| Width | fill share 10 000, at least 5 000: `2 000 - 250 = 1 750` |

**Case C: an ask pinned at a cost floor.** The agent makes the commodity, with
`UnitCost` 1 600 (inputs, tool wear and labour at the wage, `spec-production.md` 9).
`Floor = max(100, 1 600 * 7 000 / 10 000) = 1 120` and `Ceiling = max(1 120,
min(10 000, 1 600 * 30 000 / 10 000 = 4 800)) = 4 800`. Belief `Centre` 1 140,
`Width` 2 000. The ask was for 5 units at a limit of 1 160 and filled 0. The market
price is 1 000 with an age of 0 and there is an unmatched bid at `B` = 990. The
agent holds 20 against a target of 10.

| Step | Value |
|------|-------|
| `Δ`, `Δe` | `1 140 * 200 / 10 000 = 22` (22.8 truncated); `Δe = 11` |
| Candidates | `B` = 990 (below the limit); `P` = 1 000 (fresh, below the limit). The larger is 1 000 |
| Gaps | `gL = 1 000 - 1 160 = -160`; `gC = 1 000 - 1 140 = -140`; same sign |
| Evidence move | `-min(160, 140, 11) = -11` |
| Stock ratio | `20 * 10 000 / 10 = 20 000 > 15 000`: long, `v` = -1, move -22 |
| Raw centre | `1 140 - 33 = 1 107`, below the floor |
| New centre | `1 120`, on the floor: counted in `AtFloor` |
| Width | fill share 0: `2 000 + 250 = 2 250` |

The seller wants to cut its price to a market that offers 1 000 and cannot, because
the market is below 70% of its full cost (1 120 against 1 000). If that holds for
ten ticks in most of the market's outcomes, the `Warn` of 12.2 is emitted. What
happens next is not this spec's: the seller sells nothing, runs down its money and,
through the spawner, leaves or changes role (section 17). At a market price of 1 600
(the full cost) the same seller would not be pinned: the floor is 30% below it.

**Case D: the guard.** Belief `Centre` 1 000, `Width` 2 000, `UnitCost` 0. A bid for
10 units whose limit was drawn low, at 880, filled 0; the best unmatched ask is
`A` = 950, the price is stale; stock 10 against a target of 10.

| Step | Value |
|------|-------|
| Target | `T` = 950 |
| Gaps | `gL = 950 - 880 = +70`; `gC = 950 - 1 000 = -50`: opposite signs |
| Evidence move | 0: the bid lost because its limit was drawn low, not because its centre was too low; lowering the centre toward the ask would be the wrong sign |
| Stock ratio | 10 000, `v` = 0 |
| New centre | 1 000 |
| Width | fill share 0: `2 000 + 250 = 2 250` |

## 16. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-BEL-001 | A price belief MUST consist of an integer centre and an integer half-width in basis points, and MUST exist for every commodity in the agent's tradable set from the agent's creation. |
| FR-BEL-002 | After every operation on a belief, the centre MUST lie between the floor and the ceiling and the width between `BeliefWidthMin` and `BeliefWidthMax`, for any inputs within the configuration bounds. |
| FR-BEL-003 | The effective range of a belief MUST satisfy `Floor <= Low <= Centre <= High <= Ceiling` and `Low >= 1`, and MUST NOT be empty or inverted, for any sequence of updates. |
| FR-BEL-004 | The floor MUST be the larger of the reference-band floor and the cost floor, capped at `MaxPrice`; the ceiling MUST be the smaller of the reference-band ceiling and the cost ceiling, raised to the floor if below it; so `Floor <= Ceiling` always. |
| FR-BEL-005 | A unit cost of 0 MUST apply no cost floor or cost ceiling. |
| FR-BEL-006 | Every product in the rule MUST use checked arithmetic, and no input within the configuration bounds MUST cause a wrap or a panic. |
| FR-BEL-007 | Each admitted order MUST give exactly one belief update after its market has cleared, however many trades filled it; a rejected order, and an agent with no order in a market, MUST give none. |
| FR-BEL-008 | The world MUST draw exactly one `uint64` for each outcome, in order of region, commodity and admission, whether or not the update uses it. |
| FR-BEL-009 | The adjustment gate MUST be open if and only if the high word of the product of the draw and 10 000 is below `BeliefAdjustProb`; when closed, the centre and the width MUST be unchanged, apart from re-clamping to the bounds. |
| FR-BEL-010 | One unit of adjustment `Δ` MUST be `max(1, Centre * BeliefStep / 10 000)`, truncated. |
| FR-BEL-011 | For a bid that was not fully filled, the target price MUST be the smaller of the best unmatched ask and the market price when fresh and above the limit; for an ask that was not fully filled, the larger of the best unmatched bid and the market price when fresh and below the limit; and there MUST be no target if neither candidate exists. |
| FR-BEL-012 | A market price MUST be treated as evidence only when its age is at most `BeliefMaxAge`. |
| FR-BEL-013 | For a fully filled order, there MUST be a target, the fresh market price, only if the limit lies beyond the market price on the unfavourable side by more than `BeliefSignificance` basis points of the centre. |
| FR-BEL-014 | The stock component MUST be +`Δ` when the stock ratio is below `BeliefStockLow`, -`Δ` when it is above `BeliefStockHigh` and 0 otherwise or when the target stock is 0, for both sides. |
| FR-BEL-015 | The total move before clamping MUST be the sum of the evidence move and the stock component, and MUST NOT exceed `Δe + Δ` in size, where `Δe` is `max(1, Δ * BeliefEvidenceShare / 10 000)`; the new centre MUST be the result clamped to the bounds. |
| FR-BEL-016 | After an open gate, the width MUST decrease by `BeliefWidthStep` if the fill share is at least 5 000 and increase by it otherwise, clamped to its bounds. |
| FR-BEL-017 | The update MUST be a pure function of the belief, the bounds, the outcome, the market report, the stock, the target and the draw, and MUST use no map, clock, global state or floating point. |
| FR-BEL-018 | Multiplying the centre, the bounds, the limit and every price in the inputs by a factor, where no truncation occurs, MUST multiply the new centre by that factor and leave the new width unchanged. |
| FR-BEL-019 | A limit price drawn from a belief MUST lie in its effective range and MUST consume exactly one `uint64` from the world's source, including when the range has one price. |
| FR-BEL-020 | A market MUST keep, in the record that holds its published price and price age but outside what `Clearing` sees, the market prices of its last `PriceWindow` ticks in which units traded, MUST push to it once per tick after clearing and before the outcomes are processed and only if units traded, and MUST report the window mean (rounded half to even), low and high, or none while the window is empty. |
| FR-BEL-021 | The position of a price in the window MUST be `(price - low) * 10 000 / (high - low)` clamped to `[0, 10 000]`, and 5 000 when `high` equals `low`. |
| FR-BEL-022 | An initial agent's belief MUST start at the seed price clamped to the bounds, with width `BeliefWidthInit`, and the start MUST draw no randomness. |
| FR-BEL-023 | An agent that enters later MUST start at the window mean clamped to the bounds, or at the market price while the window is empty, with width `BeliefWidthInit`. |
| FR-BEL-024 | When a seed-price table is configured, it MUST replace the reference price both as the starting belief centre and as the market's initial published price. |
| FR-BEL-025 | An agent MUST hold one belief per commodity whatever its weight, the limit price MUST NOT be scaled by the weight, and the update MUST draw one gate value per outcome whatever the weight. |
| FR-BEL-026 | Whatever belief a strategy returns, the world MUST clamp it to the invariant of FR-BEL-002 and FR-BEL-003 before storing it. |
| FR-BEL-027 | Each belief parameter, including `BeliefEvidenceShare`, MUST have a default that passes validation, and a value outside its bound in section 11 MUST be rejected with `ErrOutOfBounds` carrying its path, value and bound. |
| FR-BEL-028 | Validation MUST reject `BeliefWidthMin` above `BeliefWidthInit` or `BeliefWidthInit` above `BeliefWidthMax`, `BeliefWidthMax` of 10 000 or more, `BeliefStockLow` not below `BeliefStockHigh`, `CostCeiling` below `CostFloor`, `RefBandLow` above 10 000, `RefBandHigh` below 10 000, and an `AdjustProb` of 0. |
| FR-BEL-029 | Validation MUST reject a configuration in which the cheapest tradable commodity's reference price in any region is below `ceil(10 000 / BeliefStep)` (`spec-currency.md` 3.1). |
| FR-BEL-030 | `BeliefsUpdated` MUST be emitted once per market per tick with at least one outcome, after the outcomes are processed, in order of region and commodity, and MUST contain no map. |
| FR-BEL-031 | The `Warn` "price beliefs at bound" MUST be emitted per market and never per agent, only when more than half of a market's outcomes ended on a bound for `BoundRun` consecutive ticks (at the tenth and every tenth thereafter), and MUST carry the region, commodity, bound and agent count. |
| FR-BEL-032 | The update MUST emit no `Info` log; each `Debug` line MUST be guarded by `Enabled`. |
| FR-BEL-033 | A run with a logger and an observer attached MUST give identical beliefs to a run without them. |
| FR-BEL-034 | In a market with a stale price, an unfilled bid and an unfilled ask, each with an open gate, a stock ratio inside the band and a limit on the same side of the target as its centre, the bid's centre MUST rise by `min(Δe, A - limit, A - centre)` toward the best unmatched ask and the ask's centre MUST fall by the mirror amount toward the best unmatched bid, and neither MUST pass its target. |
| FR-BEL-035 | The checkpoint MUST contain each belief's centre and width and each market's price window and `BoundRun`, in a fixed order, and a resumed run MUST give beliefs identical to an uninterrupted one. |
| FR-BEL-036 | The evidence move MUST be `sign(gC) * min(|gL|, |gC|, Δe)` with `gL` the target minus the limit and `gC` the target minus the centre, and MUST be 0 when there is no target, when either gap is 0, or when the two gaps have different signs. |
| FR-BEL-037 | `BeliefEvidenceShare` MUST be an integer from 1 to 10 000, default 5 000, and `Δe` MUST be at least 1. |
| FR-BEL-038 | Validation MUST warn, and MUST NOT fail, with `cost_floor_at_full_cost` when `CostFloor` is 10 000 or more. |
| FR-BEL-039 | The price before a market's first trade MUST be the seed price of 8.1, with price age 0, and a market that has never traded MUST have an empty price window. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-BEL-001 | With the default logger and observer, an update, a draw and a window push MUST allocate nothing, and each event-emission and logging call site in the belief path MUST allocate nothing. |
| NFR-BEL-002 | The memory of beliefs MUST be two `int64` per agent and commodity plus a fixed window per market, and MUST NOT depend on the weights, the length of a run or the number of trades. |
| NFR-BEL-003 | An update SHOULD run in constant time, and SHOULD have a benchmark with `b.ReportAllocs()`. |

## 17. Entries for the stability register

Merged into `spec-stability.md` section 4. The nine entries this section held are
the following register entries, and the register is where their sign, delay, damper,
condition and test are kept (this spec no longer repeats them, so that the two cannot
disagree):

| This spec's old entry | Register id | Where this spec supplies the mechanism |
|-----------------------|-------------|-----------------------------------------|
| The price, belief and limit loop | STB-L01 | 5.2, 5.4, 5.6 |
| The stale price loop | STB-L03 | 5.4, FR-BEL-034 |
| The cobweb | STB-L21 | 5.2, 5.5 (an averaging expectation, and the buffer) |
| Herding and lockstep, including the cohort gate | STB-L13 | 5.2, 6, 9.3 |
| Cost-push ratchet | STB-L10 | 3.2.2 |
| A shortage with no supply | STB-L05 | 5.5 |
| Width collapse and blow-up | STB-L07 | 5.7 |
| A hard floor and perishable goods | STB-L09 | 3.2.2, open question 1 |
| Overconfident band | STB-L08 | 3.2.1 |

Entries the register added about this spec: STB-L02 (the evidence offset, N-042, 5.4),
STB-L04 (the stock signal as an integrator, N-044, 5.5), STB-L06 (the overhang ratchet
and the missing anchor, N-043) and STB-L52 (the cost ceiling against a labour-dominated
price, N-055, 3.2.2). What changed in this spec because of them:

- **L02:** the evidence rule is compared to the limit, guarded and scaled
  (`BeliefEvidenceShare`); the model offset falls from 11.3% to 5.7% (5.4).
- **L04:** both signals are kept and the reason is in 5.5; the test is S01 and S04.
- **L09 and L10:** the cost is the full cost with labour, `CostFloor` is 7 000, and the
  gain around a cost cycle is `CostFloor / 10 000` times the full-cost share (3.2.2).
- **L52:** closed by the labour-inclusive unit cost: the seed is 1.00 of it, inside the
  band of 0.70 to 3.00.

Scenarios S01 to S10 and S21 of `spec-stability.md` 8 carry the tests.

## 18. Open questions

Answered in round 3:

1. **Is a hard cost floor too hard?** Answered in part by 3.2.2. With labour counted,
   a floor at 100% of cost freezes a market whose wage lags (0% of ticks traded in the
   model at an exact wage, 99% of output spoiled), so the default is 70% (7 000), at
   which a market traded in 97% to 100% of ticks for a wage up to 25% too high. Option
   (b), releasing the floor in proportion to the stock ratio or the spoilage age, is
   not adopted for version 1: it adds a rule the lower floor makes unnecessary in the
   model. What is left is data from a run: the perishables scenario S07 with AGT's real
   quantity rule and a wage that moves. `spec-skill-barriers.md` 2.4 ("cost acts through
   exit") still needs rewording, because cost now also acts directly on a seller's price
   (N-041).
2. **Per-commodity or per-role parameters.** Decided: all parameters are world-level
   in version 1. No run or source asks for a per-commodity step. If STY's bands fail
   for one class of goods, a per-commodity override goes in `spec-goods.md`'s table
   with a row in the parameter-owner table.
3. **Is the reference band needed?** Decided: yes, kept for version 1. With labour
   counted, the cost band now constrains every commodity with a known cost, but a
   commodity the agent only buys has no cost and only the reference band keeps it from
   running away. The width (a tenth to ten times) is a placeholder that S09 checks
   (STB-L08). It is an anchor, and the band is asymmetric about it (3.2.1).
4. **Bernoulli gate or a carried remainder.** Decided: the gate stays, one draw per
   outcome at every weight. `spec-stability.md` 6.2 run 4 measured the price of this:
   with 20 households a side and weights 1, 2, 4 and 10 the relative standard
   deviation of the price rises from 4.7% to 9.1% and the return after a shock from 23.5
   to 169 ticks. The floor `K` of agents per group (`spec-scale.md` 5.2) is the damper.
   Revisit only if a scale-consistency check fails.
5. **One belief or two.** Decided: one belief per commodity for version 1 (the
   original's and the paper's form). Two would double the state and change the update,
   and no run shows a need; the offset of 5.4 is not a spread problem.
8. **The agent's own fill price.** Decided: not in version 1. The rule uses the market
   price. A volume-weighted fill price needs the money of an order's fills in the outcome
   of FR-CLR-025, which `spec-clearing.md` does not deliver. Revisit if S01's residual
   (open question 12) cannot be removed otherwise.

Still open:

6. **A unit-count imbalance signal.** The source uses the share of last round's asked
   against bid units (`SIG_IMBALANCE`). The clearing report can add the unmatched units
   per side (`spec-clearing.md` 5.5). Next step: if the residual offset of 12 remains
   in a run with AGT's quantity rule, test the unmatched units as the evidence in the
   model before adding the signal.
7. **A migrant's beliefs.** Reseeding on arrival is the default (8.3). Whether an agent
   that knows another region's prices should keep them for a while belongs to
   `spec-migration.md` and `spec-information.md`. Next step: those specs.
9. **What the consumer can read.** Whether `Snapshot` reports each belief's centre
   and range or only per-market aggregates is `spec-observability.md`'s (N-040);
   vision.md goal 4 argues for at least the aggregates. Next step: that spec.
10. **The paper's rule against this one.** The paper's own rule (section 14) and this
    one have not been compared by running either. Next step: when bazaarBot is run,
    compare it with a run of this rule on the one-region chain of
    `spec-reference-chain.md`.
11. **Calibration.** Every default is a placeholder. The least certain, and most worth
    checking first, are `CostFloor`, `BeliefAdjustProb`, `BeliefStep`,
    `BeliefEvidenceShare` and `BeliefStockLow`/`BeliefStockHigh`. Next step: the
    acceptance bands of `spec-stylised-facts.md` and scenarios S01, S04, S07, S09.
12. **The remaining evidence offset.** After the revision of 5.4 the model still
    settles 5.7% below the price that clears supply and demand, with 5.9% of need
    unmet (the model's quantity rule, and the pairing of the highest bids with the
    lowest asks, are the suspects). Next step: run S01 with AGT's real quantity rule and
    the clearing-residual check FR-STB-018; if the residual exceeds the band STY sets,
    try open question 6.
13. **A wage in the cost for a worker's own labour.** The cost includes labour at the
    regional wage (`spec-production.md` 9.3). Whether a per-role wage is needed is
    `spec-production.md` open question 10.

## 19. Sources

Marked "read in full" only where the whole source was opened. Nothing here sets
a default from a source that was not opened (N-018).

| Source | What was used | Status |
|--------|---------------|--------|
| Doran, J. and Parberry, I. (2012), "Emergent Economies for Role Playing Games", International Journal of Intelligent Games and Simulation 7(3), 35-48. <https://ianparberry.com/pubs/econ.pdf> | the belief representation; the bid and ask update pseudocode; the resolution loop; the public window statistics; the experiments and the stability criteria; the floating-point sensitivity result; the 200 000 agent updates a second | **read in full**, as text extracted from the PDF; the figures (graphs) were not seen and the extended example is not in this version. This is the journal paper. The technical report URL in the bazaarBot README (`larc.unt.edu/techreports/LARC-2010-03.pdf`) returned HTTP 404 and was not read |
| Lengnick, M. (2013), "Agent-based macroeconomics: A baseline model", JEBO 86, 102-120 | the inventory bounds against recent demand; price bounds relative to marginal cost; the step `(1 +/- epsilon)` with `epsilon` uniform in `[0, vartheta]`; the probability of changing a price; the finding that equilibrium breaks down only if price flexibility is too low | text extracted from the whole PDF; the sections on the firm's price rule, Table 1 and the price-flexibility experiment were read. The Greek symbols were lost in extraction. `vartheta` = 0.02 is explicit in the text. The mapping of the other Table 1 values to symbols (the bounds 1.15 and 1.025, the probability 0.75) is inferred from the order and the stated condition `1 < phi_low < phi_high`, and is **unverified**. Not read in full |
| Cobweb model, Wikipedia. <https://en.wikipedia.org/wiki/Cobweb_model> | the stability condition under naive expectations (demand more elastic than supply at the equilibrium) | opened through a summarising fetch, not read as a whole; the page does not give formulas for adaptive expectations |
| Cobweb-model literature on adaptive expectations (Ezekiel and later) | that averaging expectations can stabilise a cobweb | **snippet only** (search result text) |
| Cliff, D., ZIP traders. <https://eprints.soton.ac.uk/264078> | a margin updated by a learning rate and momentum (Widrow-Hoff) | **snippet only**: the page returned HTTP 403. The search summary was used, not the paper |
| Gode, D. and Sunder, S. (1993), zero-intelligence traders | only as cited in the Doran and Parberry paper | **not opened** |
| ASSUME documentation, "Custom unit example". <https://assume.readthedocs.io/en/stable/examples/03_custom_unit_example.html> | a bidding strategy as one `calculate_bids()` returning an order book; its example prices from a unit's marginal revenue and imposes no explicit bound | read directly (that page only). The bidding-agents page returned HTTP 404 and the base strategy class was not read |
| abcEconomics documentation | not used in this spec | the index page only was read earlier (`evidence.md`) |
| `bazaarbot-reference.md` | the source's rule, constants and defects | the repository's own record of reading the source (not run) |

The paper's stability criteria (section VII there) are the only prior
definition of "stable" for this family of models: every commodity is produced and
consumed; none goes more than 20% of rounds without trading; the mean trade
volume exceeds one unit a round; the price variance is between 0.025 and 7.5
times the largest trading price; the mean change of price and its variance lie
between 0.02 and 0.9; fewer than two price inflections a round on average. They
are stated for prices of order one, in an unrun reconstruction of the
paper's filter, and several are scale-dependent. They are a lead for
`spec-stylised-facts.md`, not a calibration (N-015, N-027). The paper reports
that 0.19% of 1.5 billion random rule sets passed them.

## 20. Where this lives in the code

| Concern | Home |
|---------|------|
| The belief per agent and commodity, the seed table (`Config.WarmStart.Prices`), the price window per market (beside the published price and age, outside `Clearing`), `BoundRun` | root package, `agent.go` and `market.go` |
| `Bounds`, `Update`, `Draw`, the window push and the position helper | `internal/beliefs` (stdlib only), over primitive structs: `Belief`, `Params`, `Bounds`, `Outcome`, `Quote`, `Inputs`, `Result`. The caller passes the one `uint64`; the package imports no random source |
| The parameters of section 11 | root package, `config.go` and `defaults.go`; bounds in `bounds.go` |
| Processing outcomes in order, the draw per outcome, the clamp of a strategy's belief | root package, the step logic of `spec-world.md`; `strategy.go` for the default strategy's use of the draw |
| The event `BeliefsUpdated` | root package, `event.go` |
| `reason` constants and the new log keys | `internal/logkeys` |
| Checked multiply and divide, half-to-even mean | the `internal/` checked-arithmetic helpers (`spec-currency.md` section 7) |

```go
// internal/beliefs: stdlib only, no root types. A sketch, not final.
type Belief struct{ Centre, Width int64 }

type Bounds struct{ Floor, Ceiling int64 }

type Params struct {
	Step, AdjustProb, Significance int64 // basis points
	EvidenceShare                 int64 // basis points of Step: the largest evidence move (5.4)
	StockLow, StockHigh, MaxAge   int64
	WidthMin, WidthMax, WidthStep int64
	CostFloor, CostCeiling        int64 // used by Limits
	RefBandLow, RefBandHigh       int64 // used by Limits
	MaxPrice                      int64 // the bound of spec-config.md 4.1, passed in (architecture.md 2.9)
}

type Outcome struct {
	Bid           bool
	Limit         int64
	Quantity      int64
	Filled        int64
}

type Quote struct { // the market report of spec-clearing.md 5.4 and 5.5
	Price, Age       int64
	BestBid, BestAsk int64
	HasBid, HasAsk   bool
}

type Result struct {
	Adjusted             bool
	AtFloor, AtCeiling   bool
	Conflict             bool
}

func Limits(ref, unitCost int64, p Params) Bounds
func Update(b Belief, bd Bounds, o Outcome, q Quote, stock, target int64, p Params, u uint64) (Belief, Result)
func Draw(b Belief, bd Bounds, u uint64) int64
```
