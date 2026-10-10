# Spec: Demand beyond subsistence: wants, tastes and the ladder

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-DEM-NNN` and
`NFR-DEM-NNN`, which tests cite (testing.md section 4). They are in section 18.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
version 1 (`vision.md` section 4). Off unless configured: the zero configuration
has no demand beyond upkeep. Needs `spec-agents.md` (the budget stack, the spend
cap, the tradable set, the order builder, consumption), `spec-beliefs.md` (the
belief centre, `Target`, `Stock`), `spec-clearing.md` (one order per commodity per
tick), `spec-ledger.md` (a goods sink reason), `spec-spawner.md` (role change and entry,
which the carries must follow, 4.9) and `spec-config.md` (the module switch and
validation).

**Every number in this spec is a placeholder.** Nothing has been run in the
library, bazaarBot has not been run, and no acceptance band exists (notes N-015,
N-027). The models of section 14 are stylised Docker Python models of the rules,
not claims about the library; they use floating point inside the models only, and
the rules themselves are integer. No default is claimed to be stable.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

In the written specs the only demand is upkeep, the inputs of jobs and buffers of
the same goods (`spec-agents.md` 11.7). Three consequences were recorded, and this
spec removes the first two and changes the third for non-staples:

- a closed economy has a mean welfare ratio of 1 whatever `TargetRatio` is
  (`spec-viability.md` 4.7; STB-L46; N-054);
- no flow answers to price in the interior of the price range, so nothing anchors
  the price level (N-043; STB-L51);
- a staple's demand elasticity is 0 to 1 and the labour-loop condition `e > 1`
  fails (`spec-agents.md` 11.6; STB-L14).

This spec adds **wants**: spending beyond upkeep that rises with wealth, falls with
price and can be substituted. It covers:

- what a want is as data, how a household's wants are arranged, and what a class
  is (sections 3, 5);
- the per-tick rule: price ratio, substitution, the schedule, whole units from a
  carried remainder, the allocation of the discretionary budget in ladder order,
  and the merge into the agent's single order per commodity (section 4);
- the Engel-style income dependence and the price response, with the numbers that
  were run (sections 5, 6);
- the closed-economy question and what DEM needs from the money specs (section 7);
- interactions with beliefs, the production halt, recipe choice and the work rule
  (section 8);
- parameters, validation, observability, checkpoint state, register entries,
  worked examples, prior art, the notes settled and the requirements (sections 9
  to 18).

Out of scope: upkeep and survival (`spec-agents.md` 5); where money comes from
(`spec-money-supply.md`, `spec-external-sector.md`, `spec-employment.md`,
`spec-land.md`, `spec-fiscal.md`); saving and investment decisions beyond the cap of
`spec-agents.md` 7.4.4 (`spec-saving-and-investment.md`); replacement demand for
durables (open question 4); quality grades (a grade is a separate commodity,
`spec-goods.md` 7.1); advertising, fashion and status (not modelled).

## 2. Positions taken

The design pressures of the brief, each with a position and the reason. Later
sections give the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is demand beyond subsistence, as data? | A **want** is a named need slot: a list of alternative commodities, a quantity per household per period, a rank, a price elasticity with a ceiling, and an integer substitution exponent. A household's **ladder** is the wants its role has a taste row for, in rank order. Nothing is named in code: no commodity, class or want has behaviour | Supply chains are data and setting-agnostic (architecture.md 9.1). A ladder is the discrete form of an Engel hierarchy (section 16) and needs no floating point |
| 2 | Which model family? | A **tiered (lexicographic) ladder funded by a concave wealth rule**, with a linear price schedule per want. Not Stone-Geary/LES, not an Almost Ideal system, not Cobb-Douglas or CES utility | LES makes every own-price elasticity at most 1 in absolute value (section 16), so it cannot supply `e > 1`; AIDS and Cobb-Douglas need logarithms and powers. The ladder gives Engel's law without a formula (5.3), and the wealth rule is the one AGT already has, Lengnick's concave consumption (section 16) |
| 3 | How does demand grow with income without floating point? | Through the **discretionary budget** of `spec-agents.md` 7.4.4, which is a concave function of the money held above the reserve, and through the rank order: a want of higher rank is funded only from what lower ranks leave. No new income formula | One saving rule, not two. Money above the reserve is the household's own record of past income (a permanent-income proxy), local information (`spec-stability.md` 5.1.3). The marginal propensity to spend is bounded below 1 (STB-L45) |
| 4 | How does demand fall with price? | A piecewise-linear **schedule** `g(r)` of the price ratio `r` (price over reference, basis points), with slope `Elasticity` at the reference, a **choke price** where it reaches 0, and a ceiling. Integer arithmetic. The price used is the household's own belief centre | The point elasticity is `eps * r / (10 000 * g)`: it rises with price, which is the stabilising direction (6.2). Beliefs are agent state, so the response is local and lagged by the belief rule |
| 5 | Staple against luxury | Unchanged for upkeep (elasticity 0 to 1, AGT 11.6). Wants carry their own `Elasticity`, default 15 000 (1.5), a placeholder. The asymmetry exists because a want can be skipped (choked) and upkeep cannot | Not inventing an elastic subsistence demand (AGT 11.6.1). A pack may list a staple as an item of a want, which adds an elastic discretionary part on top of the inelastic base |
| 6 | Are classes roles? | **Yes (N-001).** A class is a role; its tastes are the `Tastes` rows of that role. There is no separate class entity and no household type other than a role. A role with an empty `Recipes` list is legal (`spec-agents.md` 3.3.1) and its income is a transfer named by another spec | One concept, the one the scale, welfare and spawner specs already use. A change of class is a change of role |
| 7 | Where does the money of a non-producing class come from? | Not from DEM. DEM states what it needs: a recurring transfer from a boundary account with a registered reason, and a sink or return flow for the same money (7.3). Until MON, EXT, EMP, LND or FIS exist, only relief and starting money fund such a class, and the warning `class_without_income` says so | Conservation: every faucet and sink is named (`spec-ledger.md` 7.2); macro is measured, never set |
| 8 | Closed economy | With wants, the household-weighted mean welfare ratio is `1 + W / U` (W the want spending, U the upkeep cost) and can exceed 1 with no faucet. In the model of 14.2 it is 1.53 at the default target against exactly 1.0 with upkeep only, and the money needed is about the reserve plus the excess of 7.2. The expected ceiling is `1 + Wref / U` (the wants at the reference price) and the hard bound `1 + Wmax / U` (every want at its schedule ceiling) | An identity once the wants exist: income equals spending, and spending includes wants (7.1). MON and EXT are needed only for a non-producing class and for growth (7.3) |
| 9 | Beliefs, halt, recipe choice, work rule | A want's units join the agent's `Keep` and `Target` for the commodity, so beliefs see them (8.1). The halt and recipe choice are unchanged (8.2). For the work rule, a luxury role's labour loop is stable under two conditions, `e > 1` and at least 10% headroom in the budget (6.3), which DEM can supply for non-staples and cannot for a staple | The model of 14.1; `spec-stability.md` 6.1 |
| 10 | Rationing of tier 1 above a dear price (N-052) | **Rejected for version 1.** Upkeep stays a survival rule. The reason and the condition for reopening are in 17 | It would make hunger a choice and is a guess about households (AGT 11.6.1) |
| 11 | Randomness | None. Whole units come from a carried remainder per agent and want, started from the agent's ID, not from a draw | Determinism (architecture.md 7.3); the creation draw list of `spec-agents.md` 3.5.2 does not grow |
| 12 | Registers | Eight loops (DEM-A1 to DEM-A8), three dampers (D30 to D32), three scenarios (S23 to S25); the register ids are assigned by `spec-stability.md` | `spec-stability.md` 9 |

## 3. What a want is

### 3.1 The module

Demand is a **module** (ADR 0001): `Config.Modules.Demand` is a pointer; `nil` is
off. When off, nothing in this spec runs, no state is added to an agent and the
world behaves exactly as `spec-agents.md` describes (FR-DEM-001). When on, its
fields are the module's tables and parameters (`spec-config.md` 3.2). It needs
`spec-agents.md`, which is core, so `ErrModuleDependency` cannot arise from it. It
does not need the welfare module: the Engel variable is money, not the welfare
ratio (5.1).

### 3.2 The tables

```go
// Want is a need slot beyond upkeep. Plain data; an entry of Demand.Wants.
type Want struct {
	ID           string
	Items        []WantItem // alternatives, sorted by commodity, 1 to MaxWantItems
	Quantity     int64      // units per household per Period, at least 1
	Period       int64      // ticks, default 1
	Rank         int64      // ladder position, lower is funded first, default 0
	Elasticity   int64      // basis points; slope of the schedule at the reference, default 15 000
	Ceiling      int64      // basis points; most the schedule can reach, default 15 000
	Substitution int64      // integer exponent 0 to MaxSubstitution, default 2
	Label        string     // presentation only
}

// WantItem is one alternative that can satisfy a want.
type WantItem struct {
	Commodity string
	Weight    int64 // basis points; the taste for this alternative, default 10 000
}

// Taste is a role's taste for a want: the want is in the role's ladder.
type Taste struct {
	Role  string
	Want  string
	Scale int64 // basis points of Quantity, default 10 000; 0 removes the want
}

// RegionTaste scales a want in a region (a cold region wants more of a fuel
// luxury). Optional; absent means 10 000.
type RegionTaste struct {
	Region string
	Want   string
	Scale  int64 // basis points, default 10 000
}
```

In a file (`spec-config.md` 6.2) the module is a `[demand]` table, and its presence
switches the module on. Wants, tastes and region tastes are tables keyed by ID, so
documents merge by key:

```toml
[demand]
[demand.want.comfort]
items = [{ commodity = "cloth", weight = 10000 }]
quantity = 1
period = 4
rank = 1
elasticity = 15000

[demand.want.treat]
items = [{ commodity = "sweet_a", weight = 10000 }, { commodity = "sweet_b", weight = 10000 }]
quantity = 1
period = 5
rank = 2
elasticity = 20000
substitution = 2

[demand.taste.worker.comfort]
scale = 10000
[demand.taste.worker.treat]
scale = 5000
[demand.region_taste.north.comfort]
scale = 15000
```

The IDs above are neutral examples for the layout, not content of the core. The keys
are the ones the reference chain uses for its demand overlay (`spec-reference-chain.md`
4.4.6: `demand.want.<id>`, `demand.taste.<role>.<want>`, and by the same rule
`demand.region_taste.<region>.<want>`); the two specs agree on them and a test loads
the overlay (FR-DEM-043).

3.2.1. A want is **not** a commodity and not a recipe. It names no behaviour; only
its numbers and its items act. A commodity may be an item of several wants and may
also be upkeep, an input or an output (the staple "better bread" is an upkeep
commodity that a want also lists). Validation rejects an item that is durable
(`ErrWantDurable`, 10.1), because a want is consumed (4.7) and the replacement of a
durable is open question 4.

3.2.2. Canonical order: wants by ID; items by commodity; tastes by role then want;
region tastes by region then want. The **ladder** of a role is its wants sorted by
`Rank`, then by ID (a total order, architecture.md 7.6). A role with no `Tastes`
row has an empty ladder and no wants.

3.2.3. The items of the wants in a role's ladder join the role's tradable set
(`spec-agents.md` 3.4; a change AGT owns, section 17). A belief exists for each from
creation (FR-AGT-004). A want item that is in no role's set cannot exist, because
the taste is what puts it there.

### 3.3 Classes are roles

N-001 asked whether classes are roles with an empty recipe list or a separate
concept. They are roles. A class is the pair of a role's upkeep (`spec-agents.md`
5.2) and its ladder (3.2.2): the same word for the household type that the scale,
welfare, skill and spawner specs already use. Three kinds follow from data, with no
new concept:

| Class | Data | Where its income comes from |
|-------|------|------------------------------|
| A producing household | recipes, a ladder | its sales (the common case; the closed economy of 7.1) |
| A trading household | `Trades`, a ladder | its trades (`spec-merchants.md`) |
| A non-producing household | an empty `Recipes` list, a ladder | a transfer named by another spec, relief, or starting money (7.3) |

3.3.1. Different tastes for different classes are different `Tastes` rows for
different roles: a role that likes a want twice as much has `Scale` 20 000 for it.
Tastes are not learned and do not change in a run. A household that gets richer does
not become a different class; it funds more of its ladder (5.2). Changing class is a
role change by the spawner (`spec-spawner.md` 4.2, 6); 4.9 says what happens to the
carries.

3.3.2. No code branches on a role, a want, a commodity or a region (architecture.md
9.1; FR-DEM-002).

## 4. The rule, per tick

Demand adds a **fourth tier** to the budget stack of `spec-agents.md` 7.4.3 and a
fourth part to each bid. It runs inside `Trade` (the default strategy) and in the
consumption phase. A consumer's own strategy is free to ignore it; the library
function is exported for it, as `DefaultStrategy` is (`spec-agents.md` 6.7).

### 4.1 Inputs

For an agent of weight `w`, role `ρ`, in region `g`, for each want `k` in the ladder
of `ρ` and each item `j` of `k`:

- `Centre_j`: the agent's belief centre for the commodity (`spec-beliefs.md` 3.1);
- `Ref_j`: the effective reference price in `g` (`spec-goods.md` 6.3);
- `S_ρk` and `S_gk`: the taste scale and the region scale (3.2), 10 000 if absent;
- the agent's carry `C_k` (4.4);
- the budget values of `spec-agents.md` 7.4: `avail`, `ReserveMoney`, the cap, and
  the money tiers 1 to 3 already spent.

Every product uses the checked 128-bit helper (architecture.md 3.5). Rounding is
truncation toward zero unless stated.

### 4.2 The price ratio and the substitution shares

The ratio of an item's price to its reference, in basis points, floor:

```
r_j = Centre_j * 10 000 / Ref_j
```

The belief centre is bounded by the reference band (`spec-beliefs.md` 3.2), so `r_j`
lies between `RefBandLow` and `RefBandHigh`.

The **substitution shares** divide a unit of the want among its alternatives. With
`x_j = Ref_j * 10 000 / Centre_j` (the inverse ratio), the integer exponent `σ` =
`Substitution`, and `pow(x, σ)` computed by `σ` steps of `v = v * x / 10 000` from
`v = 10 000`:

```
t_j = Weight_j * pow(x_j, σ) / 10 000
s_j = t_j * 10 000 / sum(t)             floor; the remainder to the item with the greatest t, ties to the lower commodity ID
```

`σ = 0` gives shares proportional to the weights whatever the prices (fixed
proportions, no substitution); a larger `σ` moves the shares toward the item that is
cheap against its reference; `σ = 4` is the largest. A want with one item has
`s = 10 000`. The want's **price ratio** is the share-weighted mean

```
r_k = sum_j( s_j * r_j ) / 10 000
```

The exponent is an integer because a power with a fractional exponent needs
floating point (architecture.md 3.5). This is a CES-style share rule in integer form
(section 16), not a CES utility: the price ratio is a mean and not the exact CES
index, and nothing claims the exact elasticity of substitution.

### 4.3 The schedule

The **schedule** is the share of the base quantity that a household wants at price
ratio `r`, in basis points:

```
g(r) = clamp( 10 000 + trunc( Elasticity * (10 000 - r) / 10 000 ),  0,  Ceiling )
```

`g(10 000) = 10 000`: at the reference price the household wants its base quantity.
It rises as the price falls, to `Ceiling` (default 15 000, one and a half times the
base), and falls as the price rises, to 0 at the **choke price**

```
r_choke = 10 000 + 10 000 * 10 000 / Elasticity           (basis points; 16 667 for Elasticity 15 000)
```

A household whose price ratio is at or above the choke price skips the want that
tick; it is **choked** (counted, 11). The shape is a rule constant (a straight
line); its slope and its ceiling are fields (9). The point elasticity at ratio `r`
(natural units, not an input to any rule) is

```
e(r) = Elasticity * r / ( 10 000 * g(r) )                (zero where g is at Ceiling)
```

At the reference it equals `Elasticity / 10 000`. It rises with price (the
quantity of a dearer good is more sensitive) and falls below 1 when the price is low
enough, at `r = (10 000 + Elasticity) / (2 * Elasticity) * 10 000` for a schedule not
at its ceiling (6.2).

### 4.4 Whole units from a carried remainder

A want is a rate, and quantities are whole units (architecture.md 3.5). The desired
rate of an agent in this tick, in ten-thousandths of a unit:

```
Need_k = Q * w * S_ρk * S_gk * g(r_k) / ( P * 10^8 )          (Q = Quantity, P = Period; a 128-bit product)
C_k    = C_k + Need_k
units_k = C_k / 10 000          floor
C_k    = C_k mod 10 000
```

The **carry** `C_k` (an `int64` below 10 000) is state, one per taste row of the role
(12). It adds the fractions up exactly: a household that wants one unit every
four ticks places one bid unit every fourth tick, and a cohort of `w` places `w`
units, with no random draw and the same expected count at every weight. The carry of
a new agent starts at

```
C_k(0) = ( AgentIndex * 7 919 + k's position in the ladder * 2 777 ) mod 10 000
```

a deterministic spread (no draw: `spec-agents.md` 3.5.2 is unchanged) so that the
agents of a role do not all place a unit in the same tick (DEM-A5). Units the
household cannot fund are lost; the carry keeps only the fraction, so a rich tick
does not repay a poor one (no pent-up demand).

### 4.5 The budget and the ladder

After `spec-agents.md` 7.4.3 has allocated tiers 1 to 3, the **want budget** is

```
B = max(0, cap - spent_tier3)        and never more than the money left after tiers 1 to 3
```

where `cap = SpendMaxShare * (e * H / (e + H)) / 10 000` is the cap of `spec-agents.md`
7.4.4 (`e` the money left after tiers 1 and 2, less `ReserveMoney`). The same cap
funds buffers first and wants second, so the saving rule is one rule: below the
reserve nothing is spent on either, and the sum of tiers 3 and 4 never exceeds the
excess (FR-DEM-020). Buffers go first because they belong to survival and jobs
(food first, `spec-stability.md` STB-L47).

The wants are then funded in ladder order (3.2.2). For each want `k` with
`units_k > 0` and `g > 0`:

1. split the units among the items by the shares of 4.2, by largest remainder: each
   item gets the floor of `units_k * s_j / 10 000`, and the units left go to the
   items with the greatest remainders, ties to the lower commodity ID;
2. for each item `j` in commodity order, buy `q_j = min(units_j, B / L_j)` units,
   where `L_j` is the limit price of the agent's bid for that commodity
   (`spec-agents.md` 7.5.3), and subtract `q_j * L_j` from `B`;
3. units not bought are **unfunded** (counted); there is no re-split to a cheaper
   alternative within the tick, since the shares already follow the prices.

A want of higher rank is funded only from what the lower ranks leave: this is the
**ladder**. A household with a small budget buys the first want or two; one with a
large budget goes further up. Section 5.3 shows that this gives Engel's law.

### 4.6 The merge into the agent's order

Clearing admits one order per agent and commodity (`spec-clearing.md` 3.1.1), so
the want units do not make a second order. They become a part of the bid that
`spec-agents.md` 7.3 builds. With `Want_c` the sum over the wants funded in 4.5 of
the units of commodity `c`:

```
Keep_c    = Reserve + UseTgt_c + Want_c                      (AGT 7.3, with the third term new)
Target_c  = UseTgt_c + Want_c   if either is positive, else OutTgt_c        (AGT 7.3.1)
T4_c      = Gap_c - T1_c - T2_c - T3_c                       (at least 0; the part of the bid that is a want)
qty_c     = q1 + q2 + q3 + q4,    q4 <= T4_c
```

`Purpose` follows `spec-agents.md` 7.3.6: a commodity that is a want item only has
`PurposeOther`, so it is not netted from income (`spec-welfare-and-wages.md` 2.1:
only `Input` and tool wear are netted); a want item that is also upkeep keeps
`PurposeUpkeep`. The limit price is the one drawn for the intent (one `uint64` per
intent, as now). The want units do not add to the number of draws.

Because the budget of 4.5 is a part of the cap of AGT 7.4.4, the sum over an agent's
bids of limit times quantity stays within its money (FR-AGT-041, FR-DEM-021). A
want bid that is rejected by the market is a defect (FR-DEM-022).

### 4.7 Consumption

Wants are consumed. Without this a bought good would pile up and the demand would
end. In the consumption phase (`spec-agents.md` 4, phase 6), after upkeep is eaten
(`spec-agents.md` 5.3), for each commodity `c` with planned want units `Want_c` (the
scratch value of 4.6, which lives within the tick and is not state):

```
consumed_c = min( Want_c,  max(0, held_c - U_c) )        U_c = the effective upkeep of c, 0 if not upkeep
```

as a move to the `want_consumption` boundary, oldest first (`spec-ledger.md` 5.2.1).
The subtraction of `U_c` is the guard that a want never eats into the next tick's
upkeep of the same commodity (D32). Goods that the household did not receive are not
consumed: a want that is not met in a tick is simply not satisfied. There is no
penalty and no hunger for a want. `want_consumption` is a goods sink owned by this
spec; it is a new row in the ledger registry (`spec-ledger.md` 7.2, section 17) and is
reported in `BoundaryFlow` by region and commodity, which separates discretionary
from subsistence consumption for the indicators.

### 4.8 Cohorts

An agent of weight `w` has `w` times the units (4.4) and the same limit prices;
nothing else changes (`spec-scale.md` 4.4). The carry makes the expected count
exact at every weight. A weight change leaves the carries unchanged (they are per
household fractions), as beliefs are (`spec-agents.md` 3.7). A cohort of `w` places
all its units in one order, so a cohort's want is lumpier than `w` households, and
the floor `K` of `spec-scale.md` 5.2 bounds the effect (DEM-A5).

### 4.9 Role change, entry and removal

The spawner changes an agent's role (`spec-spawner.md` 4.2, rung 1 and the voluntary
review), enters households (4.3 there) and removes agents. The carries are per agent
and want, so:

- **A role change** gives the agent the ladder of the new role. A carry is kept for
  each want that is in both ladders (by want ID, so a reordered rank does not matter),
  a want only in the new ladder starts at the formula of 4.4 with its position in the
  new ladder, and a carry of a want only in the old ladder is dropped. Nothing is drawn.
  `spec-spawner.md` 4.2 gets a row for it (handoff). The goods a household bought for
  a want of the old role and still holds are treated by the spawner's rule for goods
  outside the new tradable set (kept if they are in it, else listed by the estate).
- **An entry into an existing cohort** (`Enter`, 4.3 there) adds households: the
  carries are per household fractions and are unchanged, like a weight change (4.8).
  A new agent starts its carries by the formula of 4.4.
- **A removal** drops the carries with the agent. A reduction of weight leaves them.
- **A class without income that climbs the ladder.** A non-producing class (3.3) with
  no transfer spends its money down to the reserve, buys no want, earns nothing and
  accrues `Privation`; the ladder then changes its role (rung 1), gives it relief
  (rung 2) or removes it (rung 3) (`spec-spawner.md` 5.2). DEM adds nothing to the
  ladder: a want is never a cause of hunger (FR-DEM-026), so wants cannot start the
  climb, and a household that stops buying wants because it is poor is not counted as
  in privation for that.

## 5. Income dependence, classes and Engel's law

### 5.1 What "income" is here

The variable that funds wants is the household's **money above its reserve**, `e`,
the same variable that funds buffers. Money is the integral of past income less past
spending, so a household whose income is higher holds more and funds more of its
ladder; a household whose income falls spends down its excess first. This is a
wealth rule in the sense of Lengnick (section 16): consumption rises with wealth at
a decaying rate. It needs no welfare ratio and no new formula, so the module works
with the welfare module off.

For a steady desired want spending `x` per tick, the excess that funds it is, from
the cap of AGT 7.4.4:

```
e* = x * H / ( SpendMaxShare * H / 10 000 - x )         valid only if x < SpendMaxShare * H / 10 000
```

with `H = SpendHalfTicks * UpkeepCost`. The cap is bounded by `SpendMaxShare * H /
10 000` (5 ticks of upkeep cost per tick at the defaults), so a want spending above
that can never be fully funded whatever the household holds (the warning
`wants_exceed_spend_cap`, 10.2).

### 5.2 The Engel curve of a household

Total want spending as a function of money held is the cap (a concave curve, rising
from 0 at the reserve to its bound); the composition is the ladder. The numbers of
the defaults (`UpkeepCost` 300, reserve 900, `SpendMaxShare` 5 000, `SpendHalfTicks`
10), from the script of section 14:

| Money held | Cap per tick | In ticks of upkeep cost |
|-----------:|-------------:|------------------------:|
| 900 | 0 | 0.00 |
| 1 200 | 136 | 0.45 |
| 1 800 | 346 | 1.15 |
| 2 400 | 500 | 1.67 |
| 3 200 | 650 | 2.17 |
| 4 800 | 847 | 2.82 |
| 9 000 | 1 094 | 3.65 |

The curve starts at 0 at the reserve, rises steeply and flattens: the marginal
propensity to spend out of excess money falls from `SpendMaxShare / 10 000` toward 0.
A household below its reserve buys no want at all. A richer household's extra money
goes up the ladder, to wants of higher rank, until each want is satiated at its
schedule and the budget no longer binds.

### 5.3 Engel's law holds

Engel's law says that the share of income spent on food falls as income rises
(section 16). Here upkeep spending is fixed at `UpkeepCost` (it does not depend on
income, `spec-agents.md` FR-AGT-069) and want spending rises with the cap, so the
upkeep share of total spending is `UpkeepCost / (UpkeepCost + cap)`: 0.69 at money
1 200, 0.38 at 2 400 and 0.22 at 9 000 with the table above and an upkeep cost of 300. The staple's budget
share falls with income and its income elasticity is below 1 (zero for upkeep), while
a want of high rank has an income elasticity above 1 as it enters the budget and 0
once it is satiated. This is a property of the rules, not a calibration (FR-DEM-029).

### 5.4 What this model does not do

It has no income effect on tastes (no habit formation, no status), no hoarding for a
goal (that is SAV), no credit and no price expectations beyond the belief. A
household that expects a price to rise does not buy ahead (belief centres are the
only expectation). A very rich household saves what the cap leaves, without limit;
that is the hoarding side of STB-L45 and is bounded only by its income.

## 6. Price response and the staple/luxury asymmetry

### 6.1 The asymmetry

| Flow | Own-price elasticity | Source |
|------|---------------------|--------|
| Upkeep (tier 1) | 0 while cash suffices, 1 when cash binds | `spec-agents.md` 11.6 |
| Job inputs (tier 2) | 0 inside a profitable recipe, a switch at the margin | `spec-agents.md` 11.6 |
| Buffers (tier 3) | elastic but transient | `spec-agents.md` 11.6 |
| A want (tier 4) | `e(r)` of 4.3: `Elasticity / 10 000` at the reference, rising with price, to a choke | here |

The asymmetry is by construction and by data: a pack that wants a good to behave as a
luxury lists it as a want item with an `Elasticity` above 10 000 and does not list it
as upkeep. A good that is both (the staple that a want upgrades) has a demand that is
the inelastic base plus an elastic increment, so its elasticity is between the two
(DEM-A8). DEM does not make a pure staple elastic.

### 6.2 The two sides of the schedule

At a price above the reference the schedule is at its most stabilising: `e(r)` rises
and the want vanishes at the choke price, so a luxury price cannot ratchet to the
ceiling of the belief band while households can choke (FR-DEM-010, FR-DEM-011). At a price below
the reference the elasticity falls (0.92 at `r = 8 000` for `Elasticity` 15 000) and
is 0 where `g` is at `Ceiling`, so a glut can still push the price down to a floor
if supply does not respond. Supply responds through the halt, recipe choice and exit
(STB-D24, STB-L48; SPN), not here. The lower edge of the basin of attraction in the
model of 14.1 lies between 60% and 70% of the reference price for `Elasticity` 15 000
to 30 000 and a ceiling of 15 000.

### 6.3 The labour loop for a luxury role (answering L14)

`spec-stability.md` 6.1 shows that a self-employed role under the target-income rule
has excess-demand slope `(1 - e) Q / p` and is stable only if the demand elasticity
`e` of its good exceeds 1. Wants supply `e > 1` for a luxury. The model of 14.1
(STB 6.1's own setting: 1 000 producers, a target of 150 days at the reference, a
work floor at 146 days, 30 seeds, perturbations of +10% and -10%) gives:

| Case | End price (reference 1 000) |
|------|------------------------------|
| `Elasticity` 5 000, start +10% | 1 033 to 1 074 (the work floor's stop) |
| `Elasticity` 5 000, start -10% | 100 (the floor of the band): unstable |
| `Elasticity` 10 000, either start | 100: neutral, falls to the floor |
| `Elasticity` 15 000, 20 000 or 30 000, either start | 988 to 1 015: converges to the reference |
| Fixed budget, no headroom (spending at most the equilibrium spend), any `Elasticity` | 100: the budget binds below the reference and the demand is neutral there |
| Budget headroom 110%, 130% or 160%, `Elasticity` 15 000 or 30 000 | 1 000: converges |

Two conditions, then, and both are needed: **the schedule elasticity above 1**, and
**headroom**, the household's discretionary budget at least about 10% above the
spending at the equilibrium price, so that a fall in price raises the quantity
bought instead of meeting a fixed budget (where the elasticity is exactly 1,
neutral). Headroom is a property of the money held above the reserve (5.1): the
warning `wants_exceed_spend_cap` and the scenario S23 check it. This is a stylised
model with no income feedback from the luxury role's customers; it shows the
condition, not that a pack is stable (N-027).

DEM cannot make the labour loop of a **staple** role stable, because a staple's
demand is upkeep (6.1). The staples are held by the production halt, the bounds, the
work floor and exit, as before (STB-L14, L51).

## 7. Money: the closed economy and what DEM needs

### 7.1 The identity with wants

In a closed economy (no faucet or sink, `spec-ledger.md` 7.3) household money is
conserved. Over a window in which holdings are unchanged on average, total income
(net of inputs) equals total spending on goods, which is upkeep plus wants (buffers
are consumed later as upkeep). With `U` the total upkeep cost and `W` the total
spending on want goods:

```
mean welfare ratio  =  (U + W) / U  =  1 + W / U
```

Wants spending counts as income of the producers of the want goods, so it is not a
leak. With upkeep alone `W = 0` and the ratio is 1 (STB-L46, N-054). With wants it
exceeds 1. Two bounds on it, with `Wref` the want spending per tick when every want is
funded at its reference price (schedule 10 000) and `Wmax` the spending when every want is
funded at its schedule ceiling: the ratio is expected to settle at or below `1 + Wref / U`
(prices near the reference), and cannot exceed `1 + Wmax / U` at all. Per household:

```
Wref = sum over the ladder:  Ref_k * Quantity * S / ( Period * 10^4 )
Wmax = sum over the ladder:  Ref_k * Quantity * S * Ceiling / ( Period * 10^8 )
```

`Ref_k` is the share-weighted mean reference price of the want's items and `S` the
product of the taste and region scales in basis points. The reference chain's demand
overlay has `Wref / U` of about 0.32 (`spec-reference-chain.md` 4.4.6), so its expected
ceiling is about 1.32 against the default target 1.5, and `target_above_demand` is the
expected warning there.

The ceiling `1 + Wmax / U` also bounds the replacement ratio of the population module: births cannot be sustained at a ratio the economy cannot show (`spec-population.md` 5.3, warning `replacement_above_closed_ceiling`).

### 7.2 What was run

Model B2 (14.2): 400 households, 200 farmers and 200 artisans, fixed reference
prices (staple 100, luxury 300), upkeep 3 staples (cost 300), one luxury want of
1 unit a tick at the reference, the cap of AGT with the defaults, the target-income
rule with a band, `TargetRatio` 15 000. Mean over the last 300 of 1 500 ticks:

| Case | Mean ratio | Work share | Luxury per household per tick |
|------|-----------:|-----------:|------------------------------:|
| All farmers, upkeep only (the identity) | 1.000 | 1.00 | 0 |
| Farmers and artisans, no wants | 0.500 | 1.00 | 0 |
| Wants, money 450 (half the reserve) | 0.500 | 1.00 | 0 |
| Wants, money 900 (the reserve) | 1.061 | 0.83 | 0.30 |
| Wants, money 1 800 (twice the reserve) | 1.532 | 0.60 | 0.77 |
| Wants, money 2 700 to 9 000 | 1.533 | 0.60 | 0.77 |
| Wants, target 12 000, money 2 700 | 1.230 | 0.62 | 0.61 |
| Wants, target 20 000, money 2 700 | 2.000 | 1.00 | 1.00 |

Reading it: with upkeep alone the ratio is exactly 1 and every household works every
tick; the second row shows that a luxury-making role with no buyers earns nothing
(its ratio 0 pulls the mean to 0.5); with wants the households reach the target
(1.53 at 15 000: the rest band keeps them slightly above it) and rest 40% of the
time; money below the reserve funds no want; and when the target is above what the
wants can absorb (20 000 against a ceiling of 2.0) the households work every tick
and the target is not met. That last row is the check `target_above_demand` (10.2).

The money needed. A household funds a steady want spending `x` from an excess
`e* = x H / (S H / 10 000 - x)` (5.1). In the model `x` is 231 (0.77 of 300) and `H`
is 3 000, so `e*` is 546 and the household holds about 1 446, or 1.6 times its
reserve, as the table shows: 900 is not enough, 1 800 is. **The money a closed
economy needs for wants is the sum over households of reserve plus `e*`.** If the
total is below the sum of the reserves, no household reaches its reserve and wants
are never bought (STB-L45; DEM-A7); the warning `reserve_exceeds_money` of
`spec-agents.md` 13 checks the start.

### 7.3 What DEM needs from MON, EXT and the other income specs

Stated precisely, so that those specs are written against it and DEM writes none of
it:

1. **A ratio above 1 needs no faucet.** The result above is for a closed economy.
   DEM needs nothing from MON or EXT for it.
2. **A non-producing class needs a recurring transfer.** A household with an empty
   `Recipes` list and no income runs down its money to the reserve and then to 0,
   and buys no want. The transfer is a ledger transfer from a boundary or another
   account, with a reason registered by the owning spec (`spec-ledger.md` 7.2):
   EXT for income from the outside world, MON for a stipend (a grant to households of the role, counted as income by default, `spec-money-supply.md` 4.2, 7.3), FIS for pay or relief
   from a treasury, LND for rent, EMP for wages. DEM requires only that the transfer
   exists, is a named reason and arrives before the household's `Trade` of the next
   tick (`spec-clearing.md` 3.3.1).
3. **A recurring faucet needs a sink or a return flow.** In the model, 50 non-producing
   households with a stipend of 300 a tick and no sink make household money grow
   without bound (47 700 per household at the end against 2 700 at the start; ratio
   1.47) and the mean ratio rises only because the money keeps growing. Money
   creation without a named sink is STB-L30 and MON's: `spec-money-supply.md` answers it with a levy (a fixed amount per household, or a rate on the balance above an exemption, whose class is bounded by `E + g/rho`, 9.3), a check that fixed grants and levies match (`pair_unbalanced`, 9.2, 12.2) and the boundary balance per reason as the measure. DEM neither creates nor
   destroys money (FR-DEM-023).
4. **Growth of wants over time** (rising incomes, new tastes) is not modelled. A
   consumer who wants it changes the data between runs or uses the player.

## 8. Interactions

### 8.1 Beliefs

A want's units are part of `Keep` and of `Target` (4.6). The buyer's stock ratio for a
want-only commodity is therefore units held over units planned: 10 000 if the bid is
filled and 0 if it is not, in the neutral and the shortage bands of `spec-beliefs.md`
5.5. An unfilled want bid pushes the buyer's centre up with the evidence rule (5.4
there), and a buyer who is choked places no bid and learns nothing from that tick.
The belief centre feeds back into the schedule (4.2), so the response of demand to
the market price is **lagged by the belief rule** (the bounded, asynchronous step):
a price jump reaches demand over several ticks, not at once. That lag is a damper of
the cobweb of the luxury market (D03) and a source of delay (DEM-A2). N-042's
finding on the evidence offset used one quantity-sizing rule; a schedule is a second,
price-responsive one, and the run of `spec-stability.md` 6.2 should be repeated with
it (open question 5).

### 8.2 The production halt and recipe choice

Neither changes. A seller of a want good offers its whole surplus (AGT 7.3.5) and the
halt (AGT 7.1.5) stops a recipe whose output stock is high. With wants, the halt's
dependence on the target is better behaved for a luxury than for a staple because the
demand is elastic: the glut lowers the price, the schedule rises toward its ceiling,
and the glut is absorbed up to `Ceiling` times the base quantity, then the halt acts.
Recipe choice by net value at the agent's beliefs (AGT 7.1.4) sees the luxury's price
and is the supply side of the occupation choice (STB-L48, SPN).

### 8.3 The work rule

The target-income rule reads the welfare ratio, which includes the receipts from
wants (the sellers' income) and not the want spending (`Purpose` `Other` is not
netted and is not upkeep), so wants spending raises the ratio of the sellers and
leaves the buyer's own ratio untouched. A producer of a want good that earns more than
its target rests (the rest band), and the labour loop of that role is stable under
the two conditions of 6.3. The work floor (`MinWorkShare`) applies to it as to any
role. The survival rule and the floor are unchanged.

### 8.4 The wage and demand loop

Wants spending creates income for the sellers and income raises their wants
spending: a reinforcing loop with the marginal propensity to spend as its gain (the
Keynesian multiplier). The gain is bounded below 1 by the concavity of the cap: its
slope is `SpendMaxShare * H^2 / (e + H)^2 / 10 000`, at most `SpendMaxShare / 10 000`
= 0.5 at the default and falling as the excess grows. The loop is DEM-A1.

## 9. Parameters

All are tunables with a default, a bound, a doc line and an owner (architecture.md
9.5 to 9.7). **Every default is a placeholder** (N-015, N-027). The bounds are
validation bounds in `bounds.go` (`spec-config.md` 4.1).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Demand | `Wants` | the want entities (3.2) | none | at most `MaxWants` |
| Want | `Items` | alternatives; each a commodity and a weight | required | 1 to `MaxWantItems`; weight 1 to `MaxBasisPoints`; no duplicate commodity; each commodity known and not durable |
| Want | `Quantity` | units per household per `Period` at the reference price | required | 1 to `MaxQuantity / MaxWeight` |
| Want | `Period` | ticks over which `Quantity` is wanted | 1 | 1 to `MaxWantPeriod` |
| Want | `Rank` | position on the ladder, lower is funded first | 0 | 0 to `MaxWantRank` |
| Want | `Elasticity` | slope of the schedule at the reference, basis points; 10 000 is unit-elastic | 15 000 | 0 to `MaxElasticityBP` |
| Want | `Ceiling` | most the schedule can reach, basis points | 15 000 | 10 000 to `MaxCeilingBP` |
| Want | `Substitution` | integer exponent of the substitution shares | 2 | 0 to `MaxSubstitution` |
| Taste | `Scale` | a role's quantity multiplier for a want, basis points; 0 removes it | 10 000 | 0 to `MaxTasteScale` |
| RegionTaste | `Scale` | a region's multiplier for a want, basis points | 10 000 | 0 to `MaxTasteScale` |

New bounds for `bounds.go` (provisional, with the reasons): `MaxWants` 1 000 (as
`MaxCommodities`), `MaxWantItems` 4 (the loop over alternatives), `MaxWantsPerRole`
32 (the per-agent loop and the carry state), `MaxWantPeriod` 100 000
(`MaxTicksPerYear`), `MaxWantRank` 1 000, `MaxElasticityBP` 100 000 (a slope of ten),
`MaxCeilingBP` 50 000 (five times the base), `MaxSubstitution` 4, `MaxTasteScale`
100 000 (ten times). The products of 4.4 go through the 128-bit helper; the largest
is `1 000 * 10^6 * 10^5 * 10^5 * 5 * 10^4` and does not fit `int64`, which is why
(architecture.md 3.5).

Read here and owned elsewhere (no row): `SpendMaxShare`, `SpendHalfTicks`,
`ReserveTicks`, `FavourFloor`, `CoverTicks` (`spec-agents.md` 12), `RefBandLow`,
`RefBandHigh` (`spec-beliefs.md` 11). The module adds no world-level scalar: the
budget is AGT's cap, and the schedule's shape is a rule constant.

The placeholders, so that none is mistaken for a result: all the defaults above and
the bounds; the numbers of the worked examples of section 15; the carry spread
constants 7 919 and 2 777 (primes, chosen to spread the start; any other would do).

## 10. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). Errors name the
path, for example `demand.want.treat.items[1].weight`.

### 10.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field outside its bound in section 9; a `Quantity` below 1; a `Ceiling` below 10 000; more than `MaxWantsPerRole` tastes for one role |
| `ErrDuplicateID` | a want ID twice; a (role, want) pair twice; a (region, want) pair twice; a commodity twice in one want |
| `ErrUnknownCommodity`, `ErrUnknownRole`, `ErrUnknownRegion`, `ErrUnknownWant` (new) | a reference in a want item, a taste or a region taste that is not defined |
| `ErrWantItems` (new) | a want with no item, or with all weights 0 |
| `ErrWantDurable` (new) | a want item with `Durability` above 0 (3.2.1) |
| `ErrOverflow` | the largest weight times `Quantity` times the scales times `Ceiling` over `Period` above `MaxQuantity`; the sum over a role's ladder of the largest want spending at the largest reference price above `MaxMoney` |

### 10.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2).

| Warning | Condition |
|---------|-----------|
| `want_unused` | a want in no taste row |
| `want_unsourced` | a want item that is not raw, that no recipe makes and that no `StartStock` supplies, so no one can sell it (the producibility check of `spec-config.md` 7.5 sees recipes and upkeep) |
| `want_inelastic` | a want with `Elasticity` of 10 000 or less whose item is made by a role under the target-income rule (the stability condition of 6.3 fails) |
| `wants_exceed_spend_cap` | a role's `Wmax` per tick (7.1) is at least `SpendMaxShare * SpendHalfTicks * UpkeepCost / 10 000`: the wants can never be fully funded (5.1) |
| `target_above_demand` | the mean `TargetRatio` of a role is above `10 000 + 10 000 * Wref / UpkeepCost`: the target is not reachable at reference prices and the target-income rule is expected not to rest (7.2, the 20 000 row); it can still be reached if prices fall, up to the hard bound `Wmax` |
| `class_without_income` | a role with a ladder, an empty `Recipes` list, no `Trades` and a `StartMoney` below `ReserveTicks` times its upkeep cost plus a year of its wants: it can fund no want unless another module pays it (7.3). Not raised for a role that a grant to households with `NotIncome` false targets (`spec-money-supply.md` 4.3, 7.3) |
| `region_taste_without_taste` | a region taste for a want that no role has a taste for |

The checks that a chain can supply the wants' goods are `spec-viability.md`'s (open
question 3).

## 11. Observability

11.1. **Event.** One aggregated event, bounded by regions times roles times wants,
in the style of `UpkeepShortfall`:

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `WantShortfall` (new) | once per region, role and want per tick in which the want was choked or unfunded for at least one household, after the trade phase, in order of region, role and want | `Region`, `Role`, `Want`, `Households`, `Choked` (households whose price ratio was at the choke), `Planned` (units from the carry), `Funded` (units placed in bids) |

It carries no map, no pointer and no wall-clock time (observability-and-logging.md
4.1), is constructed only when an observer was supplied (4.5 there) and is not
emitted when every household is fully funded, so a healthy world costs nothing. The
choke and the unfunded units are the counters of the two silent dampers D30 and D31
(FR-STB-013; N-045, N-065). The goods consumed are in `BoundaryFlow` under the reason
`want_consumption`, so no event per consumption exists.

11.2. **Logs.** Per agent, `Debug`, guarded by `Enabled`, constant messages: "want
skipped" with `agent`, `want` and `reason` (`choked`, `over_budget`, `no_budget`,
`no_stock`); "want consumed" with `agent`, `commodity` and `quantity`. One `Warn`
bounded by the configuration (observability-and-logging.md 3.2.2): "wants choked",
per region, role and want, when all the role's households have been choked for the
want in 10 consecutive ticks, at the tenth and every tenth after, with `region`,
`role`, `want` and `households`; it says that the price of the item is above the
choke price everywhere. No `Info` and no `Error`.

11.3. **New log key** for observability-and-logging.md 3.3.2: `want`. The `reason`
values are constants in `internal/logkeys`.

11.4. **Metric.** `economysim.want.unfunded`, a counter of `{unit}` with attributes
`economysim.want`, `economysim.role`, fed by `WantShortfall` (`Planned` less
`Funded`), now in the metric catalogue (`rules/observability-and-logging.md` 5.3, 5.3.1). Both
attributes are bounded by the configuration (5.4.1 there).

11.5. **Snapshot.** Per region, role and want: households, planned, funded and
choked in the last tick; per agent in a query: the carries. Bounded by the
configuration.

11.6. **Cost with no observer.** The want computation uses scratch held by the world
and allocates nothing (NFR-DEM-001). Observation never changes results.

## 12. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, plain data in a
fixed order with no maps (a role change re-forms the list of an agent by 4.9):

- for each agent whose role has a ladder, in agent-ID order: the carries `C_k`, one
  `int64` below 10 000 per taste row of the role in ladder order;
- for each (region, role, want), in that order: the run of consecutive ticks with all
  households choked that the `Warn` of 11.2 uses (diagnostic: it affects no result,
  and is saved so that a resumed run logs as the original did).

**Not saved:** the price ratios, the schedule values, the shares, the budget, the
units, the planned want units and the consumption plan: all derived from the saved
state and the configuration, and scratch within a tick. A world restored between
ticks gives the same bids and consumption as an uninterrupted one (FR-DEM-034).

## 13. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9. Fields as in its section
3. **Status is `hand` unless a model is cited.** The register owner assigns the
`STB-Lnn` numbers; entries here are `DEM-An`.

**Dampers this spec adds** (for the catalogue of `spec-stability.md` 5):

| ID | Damper | Home | Status |
|----|--------|------|--------|
| D30 | The demand schedule: a want falls with the price to a choke price and rises with a fall to a ceiling; the response is through the belief centre, so it is local and lagged. Its cost: a want vanishes at a dear price (counted by `WantShortfall.Choked`) | DEM 4.3 | specified |
| D31 | The ladder and the shared cap: a want is funded only from the part of the saving rule's cap that buffers leave, in rank order, so wants never touch the reserve, upkeep or job inputs. Its cost: in a poor household higher wants are never bought (counted by `Planned - Funded`) | DEM 4.5 | specified |
| D32 | The consumption guard: a want never eats into the next tick's upkeep of the same commodity | DEM 4.7 | specified |

**DEM-A1. The wage and demand loop.** *Kind:* loop. *Loop:* wants spending (+)
income of the want-good sellers (+) their money (+) their wants spending. R, gain the
marginal propensity to spend, at most `SpendMaxShare / 10 000` and falling (8.4).
*Delay:* a tick for the money, the window for the ratio. *Owner:* DEM 4.5, 8.4; merges
with the expected "DEM, EMP" row of `spec-stability.md` 4.8 and touches STB-L45.
*Damper:* D27 (the cap is concave and bounded by the excess), D31. *Undamped:* none
beyond the cap; the multiplier is finite by construction. *Condition:* the marginal
propensity is below 1, which the cap guarantees for `SpendMaxShare` below 10 000.
*Test:* S24. *Status:* model (14.2: the ratio converges to 1.53, not unboundedly).

**DEM-A2. A luxury market with an elastic demand and a target-income labour supply.**
*Kind:* loop (STB-L14 for a luxury). *Loop:* the loop of L14 (wage, income per
working day, days worked, supply, excess demand, price) with `e` the schedule's
elasticity: the slope of excess demand is `(1 - e) Q / p`. *Delay:* the welfare
window and the belief lag (8.1). *Owner:* DEM 6.3; merges with L14. *Damper:* D30,
D13, D04. *Condition:* `Elasticity` above 10 000 **and** budget headroom of about
10% (6.3). *Test:* S23 (new). *Status:* model (14.1). The lower edge of the basin of
attraction is between 60% and 70% of the reference price; below it the market goes
to the floor.

**DEM-A3. Food first and the fall of wants in a staple shortage.** *Kind:*
amplifier. *Loop:* staple price (+) upkeep spending (-) the excess (-) the cap (-)
wants spending (-) luxury sellers' income (-) their staple purchases. R, bounded by
the cap reaching 0. *Owner:* DEM 4.5; merges with L47 (AGT-A6). *Damper:* D31, D01.
*Undamped:* by design, wants fall first in a famine. *Test:* S04, S09. *Status:*
hand.

**DEM-A4. Substitution flips.** *Kind:* amplifier. *Loop:* an item's price (+) its
share (-) demand for it (-) its price; with `Substitution` high, a small price gap
moves most of the demand to the cheaper alternative, and with the belief lag the
shares can swing between the items. *Delay:* the belief step. *Damper:* D03, D02, an
integer exponent of at most 4. *Undamped:* a pair of identical goods with
`Substitution` 4. *Condition:* a want's alternatives differ in weight or the
exponent is small. *Test:* S25 (new, below). *Status:* hand.

**DEM-A5. Periodic wants and cohort pulses.** *Kind:* amplifier. *Loop:* none; a
want with a long `Period` is placed once per `Period` ticks, and a cohort places `w`
units at once. *Damper:* the carry start spread (4.4), the floor `K` of
`spec-scale.md` 5.2 (D10), different `Period`s. *Undamped:* a world with few agents
and a long period has pulses. *Test:* S10. *Status:* hand.

**DEM-A6. A target the wants cannot absorb.** *Kind:* hazard (N-054's other half).
*Loop:* none; an identity (7.1): the mean ratio is at most `1 + Wmax / U` and is expected at or below `1 + Wref / U`, so a
`TargetRatio` above the second is not met at reference prices, the work share stays at 10 000 and the sellers'
surplus is unsold (as L46). *Damper:* D24 bounds the glut; the warning
`target_above_demand`. *Test:* S24 (the 20 000 row). *Status:* model.

**DEM-A7. Money below the sum of the reserves.** *Kind:* hazard (merges L45). *Loop:*
if total money is below the sum of the reserves, no household has an excess, no
want is funded, the sellers earn nothing and the money does not move. *Condition:*
total money at least the sum over households of reserve plus `e*` (7.2). *Test:*
S25. *Status:* model (the 450 row: ratio 0.5).

**DEM-A8. A staple that is also a want item.** *Kind:* bias. The demand is the
inelastic base plus the elastic increment, so the staple's elasticity is higher than
AGT's 0 to 1 by the share of the increment and no more. *Owner:* DEM 6.1. *Test:*
S04 with the staple as an item. *Status:* hand.

**New scenarios proposed to the register.** S23: a self-employed luxury role under
the target-income rule with a want of `Elasticity` 5 000, 10 000, 15 000 and 30 000
and budget headroom 100%, 110% and 160%, perturbed up and down; asserts that the
price converges only when both conditions hold and that the lower edge of the basin
is reported (DEM-A2). S24: a closed economy with a ladder and the default
`TargetRatio`, run with the wants on and off and with a target above the ceiling;
asserts mean ratio near 1 with the wants off, above 1 with them on and at most
`1 + Wmax / U`, and a bounded glut with the target above it (DEM-A1, A6, N-054).
S25: total money at 0.5, 1 and 2 times the sum of the reserves, and two alternatives
with `Substitution` 4 and a price gap; asserts that wants are not bought below the
reserve and that the shares do not swing between the items without damping
(DEM-A4, A7).

**Loops this spec creates that nothing yet dampens:** the substitution flip with a
high exponent (A4), the pulses of a cohort with few agents (A5), and a role whose
luxury has no supply response (6.2: the lower side of the schedule).

**Checklist items (spec-stability.md 9.6, 9.7).** Delays that match a neighbour:
the carry's `Period` against `ShelfLife` of a perishable want item (a want bought for
the tick and consumed in it is not exposed, but a long `Period` with units bought on
the tick may waste goods only if the item spoils before the consumption of the tick,
which cannot happen); the belief lag (`BeliefAdjustProb` 5 000, step 200) against the
welfare window 30 (A2). Every damper parameter named here has a row in the parameter
table (section 9).

## 14. What was run

All were run in `python:3.12-slim` through Docker; scripts and outputs are in
`/tmp/economy-boss/dem/` (outside the repository, N-068). They are models of the
rules, not of code. Floating point is used inside them where a model needs a
division; the spec's rules are integer. They use placeholders (N-015). **Nothing here
supports a claim of stability (N-027).**

### 14.1 The luxury labour loop (`a_loop.py`, `a2.py`, `a3.py`)

The setting is `spec-stability.md` 6.1's: 1 000 self-employed producers of 10 units a
day, an income target of 1 500 000 (150 days at the reference price 1 000), the price
moving by 2% with probability one half toward excess demand, clamped to [100, 10 000],
a floor of 146 working days, 400 ticks, 30 seeds, starts at 110% and 90% of the
reference. Demand is the schedule of 4.3 times the equilibrium quantity, with a
ceiling of 15 000, and a budget that caps the quantity at `headroom * S0 * P0 / p`.
The results of 6.3 are these runs. The basin: with budget headroom 130% and ceiling
15 000, starts at 70% or more of the reference converge and starts at 60% or less go
to 100, for `Elasticity` 15 000, 20 000 and 30 000 (10 seeds each). The runs match
`spec-stability.md` 6.1 for `Elasticity` 5 000 (an upper stop at 1.03 to 1.07 times
the reference, a low pin) and add the converging cases. The model has no income
feedback from the luxury role's customers and no halt.

### 14.2 The closed economy (`b2.py`, `b2_allf.py`)

The model of 7.2. A first model with prices moving every tick (`b_closed.py`) was
dropped: its staple supply exceeded demand and the prices ran to their bounds, which
is the glut of L46 and answers nothing about the ratio. The reported model fixes the
prices at the reference and lets quantities adjust, which isolates the identity and
the money needed. It shows the three facts of 7.1 to 7.3: the identity at 1.000, the
ratio 1.53 with wants and the ceiling 2.0, the money threshold between 900 and 1 800,
and the unbounded growth of money with a stipend and no sink.

### 14.3 The integer rules (`worked.py`)

The functions of 4.2 to 4.5 as integer code, used for the examples of section 15 and
the table of 5.2 and the elasticity table of 4.3. Not library code.

## 15. Worked examples

All numbers are placeholders, chosen to be easy to follow, and were computed by
`worked.py`. Neutral commodity names.

### 15.1 One household, three tiers and two wants

A household (`w = 1`) with money 3 200, a staple stock that covers its upkeep (so
tiers 1 to 3 spend nothing), `UpkeepCost` 300, reserve 900. The cap: `e = 2 300`,
`H = 3 000`, `e * H / (e + H) = 1 301`, `cap = 5 000 * 1 301 / 10 000 = 650`.

Want `comfort`: one item (reference 150, centre 165), `Quantity` 1, `Period` 1,
`Elasticity` 15 000, `Ceiling` 15 000, rank 1. `r = 165 * 10 000 / 150 = 11 000`;
`g = 10 000 + 15 000 * (10 000 - 11 000) / 10 000 = 8 500`; `Need = 1 * 10 000 *
10 000 * 8 500 / 10^8 = 8 500`. With a carry of 4 000, `C = 12 500`: one unit, carry
2 500. The limit drawn is 160, so it buys 1 unit and the budget falls to 490.

Want `treat`: items `sweet_a` (reference 400, centre 380) and `sweet_b` (reference
500, centre 550), weights 10 000, `Substitution` 2, `Period` 5, `Elasticity` 20 000,
rank 2. Ratios 9 500 and 11 000; inverse ratios 10 526 and 9 090; powers (two steps)
11 079 and 8 262; shares 5 728 and 4 272; `r_k = (5 728 * 9 500 + 4 272 * 11 000) /
10 000 = 10 140`; `g = 10 000 + 20 000 * (10 000 - 10 140) / 10 000 = 9 720`; `Need =
1 * 10^8 * 9 720 / (5 * 10^8) = 1 944`. With a carry of 9 000: `C = 10 944`, one
unit, carry 944. The unit goes to `sweet_a` (share 5 728 against 4 272, largest
remainder). Its limit is 410, within the budget 490: bought, budget left 80.

Choke: if the centres were 700 and 800 the ratios would be 17 500 and 16 000, the
want ratio 16 859, above the choke price of `10 000 + 10^8 / 20 000 = 15 000`, so `g
= 0` and the household is choked.

### 15.2 The elasticity of a want

For `Elasticity` 15 000 and `Ceiling` 15 000, `e(r)` in basis points of an elasticity
(10 000 = 1): 9 230 at `r = 8 000` (below 1), 9 999 at 8 333 (the point where
`e = 1`), 11 739 at 9 000, 15 000 at 10 000, 19 411 at 11 000, 25 714 at 12 000, 52 500
at 14 000 and 240 000 at 16 000.

### 15.3 Engel's law

The table of 5.2 with an upkeep cost of 300: at money 1 200 the cap is 136, so the
upkeep share of spending is `300 / 436 = 0.69`; at 2 400 it is `300 / 800 = 0.38`; at
9 000 it is `300 / 1 394 = 0.22`. The share falls as money rises, as 5.3 says (the cap
is an upper bound on want spending, so these are lower bounds on the fall).

## 16. Prior art

What was opened, and what was taken. "Read" means the text was in front of the
writer; a fetch tool that summarises is marked. No default in this spec is set from
any source.

| Source | Status | Taken | Not taken |
|--------|--------|-------|-----------|
| Lengnick (2013), "Agent-based macroeconomics: a baseline model" | equations 11 and 12 and the surrounding text read from text extracted from the PDF: planned real consumption `c = (m / P)^alpha`, `0 < alpha < 1`, capped at `m / P`; "consumption expenditure increases with personal wealth but at a decaying rate"; consumption depends only on wealth; the planned amount is spread equally over the days of the month | A concave wealth rule for consumption (already AGT 7.4.4); the monthly plan spread over days (our carry spreads a rate over ticks) | The power function and `alpha` (floating point); his household has no upkeep and no ladder, and demand responds to price only by searching for a cheaper seller |
| Stone-Geary utility and the linear expenditure system, Wikipedia "Stone-Geary utility function" | opened through a summarising fetch: `U = prod (q_i - gamma_i)^beta_i`, demand `q_i = gamma_i + beta_i / p_i * (y - sum gamma_j p_j)`, `gamma` subsistence quantities, the bracket supernumerary income. The page says nothing about elasticities | The structure: upkeep is the subsistence part and what is left is spent on the rest. DEM's discretionary budget is the supernumerary income | The formula. A property worked out here and not read in the source: the own-price elasticity of the discretionary part is `-1` for a good with `gamma = 0` (a constant share of a fixed budget), so LES gives `e` of at most 1 for every good and cannot supply `e > 1`. This is a hand derivation (unverified against the literature) |
| Engel's law, Wikipedia "Engel's law" | opened through a summarising fetch: the share of income spent on food falls as income rises although food spending rises; Engel's own 1857 data gave an income elasticity of food of about 0.86; a necessity's Engel curve is concave; the UN benchmarks of 59% and 30% for the food share | The law as a property the ladder must show (5.3). The figure 0.86 is not used for any default | The benchmarks as numbers |
| Constant elasticity of substitution, Wikipedia | opened through a summarising fetch: `sigma = 1 / (1 - rho)`, goods are perfect substitutes as `sigma` goes to infinity and perfect complements as it goes to 0; the quantity ratio follows the price ratio raised to a power set by `sigma` | The idea of a share that follows the price ratio to a power; with an integer power it is computable in integers | The exact CES price index (it needs a fractional power); the claim that our `Substitution` is `sigma` (it is an integer approximation) |
| Victoria 3, `vic3.paradoxwikis.com/Goods` and `/Pops` | opened through a summarising fetch: goods are staple or luxury, "staple goods tend to be purchased in vast quantities by poor and middle class pops, while richer pops generally avoid them in favor of their luxury variants"; prestige goods have a substitution weight of +50% by default; pops with a higher standard of living have larger and more expensive needs; a good's price varies up to 75% either side of the base with supply and demand. The rules by which purchases depend on price were not in the page | The staple/luxury distinction as data, a substitution weight per alternative (our `Weight`), needs that grow with a standard of living (our ladder) | Standard of living as a state variable, the +50% constant, any buying rule (not read) |
| Dwarf Fortress, `dwarffortresswiki.org/index.php/Need` | opened through a summarising fetch: needs are met by activities tied to personality, are not tiered by wealth or price, involve no money; focus is the ratio of met to unmet needs | Nothing in the rules. It is a counter-example: a need system with no budget has no demand curve | The whole mechanism |
| abcEconomics `Household` page | opened through a summarising fetch: the only utility constructor documented there is Cobb-Douglas (`create_cobb_douglas_utility_function(exponents)`); `consume` takes a utility function and goods; `NotEnoughGoods` when short | That a household is a pure consumer with a utility function (the shape of a want) | Cobb-Douglas (fixed budget shares, elasticity 1, no choke price, needs a power) |
| USDA ERS, Seale, Regmi and Bernstein (2003), "International evidence on food consumption patterns" | found by search only; the summary page returned 404 and no figures were read. From the search summary: poorer countries spend more of the budget on necessities and respond more to price and income | Nothing | Every number. No elasticity default is set from it |

The Almost Ideal Demand System (Deaton and Muellbauer), Working-Leser Engel curves,
Friedman's permanent income and the Godley-Lavoie consumption function were **not
opened**; they are named only as the families compared in position 2 (AIDS and
Working-Leser need logarithms), and nothing is set from them. The comparison in
position 2 is therefore a design comparison on the stated grounds (integer
arithmetic, `e > 1`), not a literature review.

**Differences from bazaarBot.** bazaarBot has no demand beyond a role's fixed ideal
stock of the goods it uses (`bazaarbot-reference.md` sections 3 and 5); it has no
price-responsive demand and no classes. Everything in this spec is new to the port.

## 17. Notes settled

Per the rules of `doc/notes.md`, each open note tagged DEM or ALL, and the others the
brief named. The text for `doc/notes.md` is in the handoff; the boss applies it.

| Note | Disposition |
|------|-------------|
| N-001 | **Applied, close.** A class is a role; its tastes are the `Tastes` rows; unearned income is a transfer named by another spec (3.3, 7.3) |
| N-054 | **Applied, close.** With wants the closed economy's mean ratio is `1 + W / U` and exceeds 1 with no faucet; the expected ceiling is `1 + Wref / U` and the hard bound `1 + Wmax / U`; the money needed is the reserve plus `e*` per household (7.1, 7.2). The default `TargetRatio` stays 15 000 and the warning `target_above_demand` checks a pack against its own ceiling. A pack with no DEM still has the identity of N-054 |
| N-043 | **Partly applied, leave open.** DEM supplies a price-responsive flow for non-staples (D30). A pure staple is unchanged: the anchor is the halt, buffer buying, recipe choice and exit. Remove DEM from the areas; SPN and STB keep it |
| N-052 | **Rejected as framed, close.** Rationing of tier 1 above a dear price is not added: it makes hunger a choice and is a guess about households. DEM supplies an elastic demand for what is discretionary, including an elastic increment on a staple (a want listing the staple, DEM-A8). Reopen condition: if S04 or S16 shows a pinned staple market that the halt, exit and the work floor do not cure, raise a new note owned by AGT |
| N-042 | **Left open.** The evidence offset was found with one quantity-sizing rule; a schedule is a second. The run of `spec-stability.md` 6.2 run 2b should be repeated with it (open question 5). Added to the note |
| N-046 | **Applied for DEM.** The entries are in 13 |
| N-012, N-065 | **Applied.** DEM adds `WantShortfall`, the log key `want`, a metric and a Snapshot figure for the two silent dampers (11) |
| N-017 | **Applied.** New terms for the glossary: want, ladder, rank, taste, schedule, choke price, ceiling (of a schedule), substitution share, want carry, discretionary budget, supernumerary excess |
| N-016 | **Applied.** The parameters of section 9 join the parameter-owner table; no field is unnamed |
| N-027 | **Applied.** Every number is a placeholder (preamble, section 9) |
| N-015 | **Applied.** Placeholders listed in section 9 |
| N-058 | Not touched: no `math/big` here |
| N-077 | **Done for DEM.** `rules/observability-and-logging.md` now has the event `WantShortfall` (4.3), the log key `want` (3.3.2) and the metric `economysim.want.unfunded` (5.3, 5.3.1); the rules worker replaced the read-receipt tag. Nothing of DEM is left in N-077 |

**Why tier-1 rationing was rejected (N-052).** Upkeep is the survival rule: `Fed`
counts households that received their full basket (`spec-agents.md` 5.4). A rationing
rule would make a household choose to be unfed when the price is dear, which turns the
`Unfed` signal from an outcome into a decision and would feed the spawner's hunger
ladder with a choice. Whether households really do that in a famine is a claim about
behaviour that no source in the repository supports (Sen's entitlement argument and
the like were not opened). The anchor of a staple that remains is the halt and exit
(SPN), and `spec-stability.md` run 1b says plainly that this is not a cure.

## 18. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-DEM-001 | With `Modules.Demand` nil the world MUST behave exactly as without this spec: no want state, no extra order quantity, no `want_consumption` flow and no `WantShortfall` event. |
| FR-DEM-002 | No code of this spec MUST branch on a commodity, role, region, want or class identifier. |
| FR-DEM-003 | A want MUST have an ID, one to `MaxWantItems` distinct known non-durable items each with a weight, a `Quantity` of at least 1, a `Period` of at least 1, a rank, an `Elasticity`, a `Ceiling` of at least 10 000 and a `Substitution` of 0 to `MaxSubstitution`, each within its bound. |
| FR-DEM-004 | A role's ladder MUST be its wants with a taste row of nonzero scale, sorted by rank then ID, and a role with no taste row MUST have an empty ladder and no want state. |
| FR-DEM-005 | A class MUST NOT be an entity of its own: a household type MUST be a role, and a role with an empty `Recipes` list and a ladder MUST be valid. |
| FR-DEM-006 | The tradable set of a role MUST include the items of the wants in its ladder, and a belief MUST exist for each from creation. |
| FR-DEM-007 | The ratio `r_j` of an item MUST be the agent's belief centre times 10 000 over the effective reference price in the agent's region, rounded down. |
| FR-DEM-008 | The substitution shares of a want MUST be proportional to the weight times the inverse price ratio raised to the integer `Substitution` power, MUST add to 10 000, and with `Substitution` 0 MUST be proportional to the weights whatever the prices. |
| FR-DEM-009 | The want's price ratio MUST be the share-weighted mean of the item ratios, rounded down. |
| FR-DEM-010 | The schedule MUST be `clamp(10 000 + trunc(Elasticity * (10 000 - r) / 10 000), 0, Ceiling)`, MUST equal 10 000 at the reference price, MUST NOT increase as the price ratio rises, and MUST be 0 at and above the choke price. |
| FR-DEM-011 | An agent whose want price ratio gives a schedule of 0 MUST skip the want for the tick, MUST be counted as choked, and MUST place no bid for it. |
| FR-DEM-012 | The desired rate of a want MUST be `Quantity * weight * taste scale * region scale * schedule / (Period * 10^8)` ten-thousandths of a unit, computed with a checked 128-bit product, and the agent MUST want `floor((carry + rate) / 10 000)` units, keeping the remainder as its carry. |
| FR-DEM-013 | The carry of a new agent MUST be `(AgentIndex * 7 919 + position * 2 777) mod 10 000` and MUST NOT use the random source. |
| FR-DEM-014 | Over a long run the units wanted of a want MUST equal the rate times the number of ticks to within one unit per household, whatever the weight and whatever the changes of price. |
| FR-DEM-015 | Units that an agent does not fund in a tick MUST be lost and the carry MUST keep only the fraction. |
| FR-DEM-016 | The want budget MUST be the cap of `spec-agents.md` 7.4.4 less the tier 3 spending, floored at 0, and never more than the money left after tiers 1 to 3. |
| FR-DEM-017 | Wants MUST be funded in ladder order, and a want of higher rank MUST be funded only from the budget that wants of lower rank leave. |
| FR-DEM-018 | The units of a want MUST be split among its items by the shares, by largest remainder with ties to the lower commodity ID, and each item MUST be bought up to the lesser of its units and the budget over its limit price. |
| FR-DEM-019 | A want unit MUST be a part of the agent's single bid for the commodity, MUST add to `Keep` and to `Target` of that commodity, and MUST NOT create a second order. |
| FR-DEM-020 | The sum of the tier 3 and tier 4 spending of an agent MUST NOT exceed the excess of its money over its reserve money, so a household below its reserve MUST buy no want. |
| FR-DEM-021 | The sum over an agent's bids of limit price times quantity MUST NOT exceed its available money, with wants included. |
| FR-DEM-022 | With an accurate view the default strategy MUST NOT produce a want bid that the market rejects, over a range of seeds. |
| FR-DEM-023 | This module MUST NOT create or destroy money: its only effect on money is trades between accounts, and it MUST NOT add a faucet or a sink. |
| FR-DEM-024 | A want bid MUST have the purpose `Other` unless the commodity is also upkeep, a job input or a tool, in which case the precedence of `spec-agents.md` 7.3.6 applies. |
| FR-DEM-025 | In the consumption phase, after upkeep, each want commodity MUST be consumed up to the lesser of its planned want units and what is held above that tick's upkeep of the same commodity, as a move to the `want_consumption` boundary, oldest first. |
| FR-DEM-026 | A want MUST NOT cause hunger: an unmet or unfunded want MUST change no `Unfed`, `Hunger` or welfare record. |
| FR-DEM-027 | An agent of weight `w` MUST want `w` times the units of an agent of weight 1 in the same per-household state, with equal limit prices. |
| FR-DEM-028 | A change of weight MUST leave the carries unchanged. |
| FR-DEM-029 | With the default rules, the upkeep share of an agent's spending MUST fall as its money above the reserve rises, and the spending on a want of higher rank MUST be zero until the wants of lower rank are satiated or choked. |
| FR-DEM-030 | The mean welfare ratio of a closed economy in which households buy wants MUST be the total income over the total upkeep cost, and MUST exceed 1 when want spending is positive, with no faucet. |
| FR-DEM-031 | The mean welfare ratio MUST NOT exceed `1 + Wmax / U`, and a role whose mean `TargetRatio` is above `1 + Wref / U` MUST give the warning `target_above_demand`. |
| FR-DEM-032 | Validation MUST reject, with the sentinels of 10.1: a bound violation, a duplicate ID, an unknown commodity, role, region or want, a want with no item or all weights 0, a durable item, and an overflow. |
| FR-DEM-033 | Validation MUST warn, and MUST NOT fail, for `want_unused`, `want_unsourced`, `want_inelastic`, `wants_exceed_spend_cap`, `target_above_demand`, `class_without_income` and `region_taste_without_taste`. |
| FR-DEM-034 | The carries and the choke runs of section 12 MUST be saved as plain data in a fixed order with no maps, and a world restored between ticks MUST give the same bids and consumption as an uninterrupted one. |
| FR-DEM-035 | `WantShortfall` MUST be emitted once per region, role and want per tick with a choked or unfunded household, in order of region, role and want, MUST contain no map, and MUST NOT be emitted when all households are fully funded. |
| FR-DEM-036 | The `Warn` "wants choked" MUST be emitted per region, role and want only, never per agent, after 10 consecutive ticks with all households choked, at the tenth and every tenth after. |
| FR-DEM-037 | Every per-agent log line of this spec MUST be at `Debug` and guarded by `Enabled`. |
| FR-DEM-038 | A run with a logger and an observer attached MUST give results identical to a run without them. |
| FR-DEM-039 | Goods and money MUST be conserved after every tick of a run with wants, checked by audit, and the want consumption MUST appear in `BoundaryFlow` under `want_consumption` by region and commodity and nowhere else. |
| FR-DEM-040 | A world with the same seed, configuration and calls MUST give identical bids, units and carries on every run. |
| FR-DEM-041 | Each parameter of section 9 MUST have a default that passes validation, and a value outside its bound MUST be rejected with `ErrOutOfBounds` carrying its path, value and bound. |
| FR-DEM-042 | On a role change the carries of wants in both ladders MUST be kept by want ID, those only in the new ladder MUST start by the formula of 4.4, those only in the old ladder MUST be dropped, and nothing MUST be drawn; an entry into an existing cohort MUST leave the carries unchanged. |
| FR-DEM-043 | The keys `demand.want.<id>`, `demand.taste.<role>.<want>` and `demand.region_taste.<region>.<want>` MUST load the tables of 3.2, and the demand overlay of `spec-reference-chain.md` 4.4.6 MUST load without error and without an unexpected warning. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-DEM-001 | With the default logger and observer, the want computation of a tick for the reference chain MUST allocate nothing beyond scratch held by the world. |
| NFR-DEM-002 | The cost of the want computation in a tick SHOULD be proportional to the sum over agents of the length of the ladder times the number of items, and MUST NOT depend on the weights. |
| NFR-DEM-003 | The pure functions of `internal/household` for the ratio, the shares, the schedule, the carry and the ladder allocation SHOULD have benchmarks that call `b.ReportAllocs()`. |
| NFR-DEM-004 | Those functions MUST have fuzz tests asserting that no input within the bounds causes a panic or an overflow, that the shares add to 10 000, that the schedule does not rise with the price and that the budget is never exceeded. |

## 19. Open questions

1. **Substitution within upkeep.** A household that can eat either of two staples is
   still two roles or a list that includes both (`spec-agents.md` open question 11).
   An upkeep need that accepts alternatives would give a staple a cross-price
   elasticity and would be the real anchor of N-043; it changes `Fed` and so is
   AGT's. Not done: the arithmetic of 4.2 is reusable for it. Next step: AGT decides
   after S04 and S16 have run.
2. **The shape of the schedule.** A straight line was chosen for integer simplicity.
   A convex curve (constant elasticity) would not rise toward its choke so quickly.
   Next step: compare in the scenario S23 with a piecewise-linear curve of three
   segments, as data, if the line proves too stiff at the low end (6.2).
3. **Viability with wants.** `spec-viability.md` 4 checks that labour can produce the
   upkeep; the wants' goods are extra final demand in the programme. Next step: VIA
   adds a final-demand vector from the ladder at the reference prices and reports a
   `wants_exceed_labour` diagnostic. The reference chain's census for `one-demand` already
   treats the wants as a fixed extra demand at the reference (`spec-reference-chain.md`
   4.4.6), as a prototype reading until VIA does it.
4. **Replacement demand for durables.** A durable want (furniture, a coat) is a stock
   to hold and replace, not a flow. It needs wear by use or by time in the ledger
   (`spec-goods.md` 5) and is scenario S17's. Left out of version 1 by the rejection
   `ErrWantDurable`.
5. **The evidence offset with a schedule.** N-042 found an offset with one
   quantity-sizing rule. Repeat run 2b of `spec-stability.md` with a schedule demand.
6. **Headroom.** The rule gives wants 100% of the cap that buffers leave. Whether the
   cap's default `SpendMaxShare` 5 000 leaves the 10% headroom of 6.3 for typical
   wants is a property of the pack (the warning covers the extreme). Next step: S23.
7. **Rank ties and many wants.** Wants of equal rank are funded in ID order, so a
   poor household always prefers the earlier ID. A share-based split inside a rank
   would be fairer and costs a rule. Next step: decide when the reference chain has
   two wants of one rank.
8. **Taste drift.** Tastes are constant. Rising incomes lift wants only through
   money. A habit or demonstration effect is not modelled.
9. **The consumption guard (D32)** keeps one tick of upkeep. Whether `CoverTicks`
   should be kept instead is a question for the scenario with a staple that is also a
   want.

## 20. Where this lives in the code

| Concern | Home |
|---------|------|
| `Want`, `WantItem`, `Taste`, `RegionTaste`, `DemandConfig`, the `Modules.Demand` pointer | root package, `config.go` (types) and `defaults.go`; bounds in `bounds.go` |
| Validation, the sentinels `ErrUnknownWant`, `ErrWantItems`, `ErrWantDurable`, and the warnings | root package, `validate.go`, `errors.go` |
| Pure rules: the ratio (4.2), the shares (4.2), the schedule (4.3), the carry and units (4.4), the ladder allocation (4.5), the want consumption amount (4.7) | `internal/household` (stdlib only, over primitive slices, converted inline at the call, architecture.md 2.9); the same package as AGT's pure rules, because they share the budget |
| The merge into the bid and the consumption phase | root package, `strategy.go` (`DefaultStrategy.Trade`) and `world.go` |
| The event `WantShortfall` | root package, `event.go` |
| Reason constants (`choked`, `over_budget`, `no_budget`, `no_stock`) and the log key `want` | `internal/logkeys` |
| The ledger reason `want_consumption` | `internal/ledger` registry (`spec-ledger.md` 7.2) |

```go
// internal/household: stdlib only, no root types. A sketch, not final.
type WantItem struct {
	Centre, Ref, Weight int64
}

func PriceRatio(centre, ref int64) int64
func Shares(items []WantItem, substitution int) []int64   // adds to 10 000
func Schedule(ratio, elasticity, ceiling int64) int64
func Units(carry int64, quantity, weight, tasteBP, regionBP, sched, period int64) (units, newCarry int64)
func Allocate(units int64, shares []int64) []int64        // largest remainder
```
