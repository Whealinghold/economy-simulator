# Spec: Resources: regional endowments, depletion and regeneration, the yield table, and the fixed factor of the population loop

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-RES-NNN` and
`NFR-RES-NNN`, which tests cite (testing.md section 4). They are in section 20.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
version 1 (`vision.md` section 4). The endowments are off unless configured: the
zero configuration has no resource, and a recipe with `Draws` fails validation
(`spec-production.md` 10.2). One part is not optional: the values of the yield
table, which this spec owns (section 7), belong to the core table of
`spec-production.md` 6 and work with the module off. Needs `spec-production.md`
(6, the yield table and the seam of 6.5; 4.2, the execution count; 5.5, the order
of starts), `spec-regions.md` (3, 5, the region and the override mechanism),
`spec-calendar.md` (4, 5, the season of a tick), `spec-goods.md` (6.2, a raw
good), `spec-ledger.md` (7.2, the reason registry, to which this spec adds
nothing), `spec-agents.md` (6.4, the agent view; 4, the tick) and
`spec-config.md` (3, 4, 7).

**Every number in sections 12, 16, 17 and 18 is a placeholder.** Nothing has been
run against the library, bazaarBot has not been run, and no acceptance band exists
(notes N-015, N-027). Section 17 describes stylised Docker Python models of the
rules of this spec, not runs of code and not a claim that any default is stable.
No default is set from a source: the sources for the shapes are general texts that
were not opened for this spec (section 21), and the defaults are structural
(a finite stock, a flat curve, a full start) or placeholders.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer, never a household (N-001, N-017). The examples of
sections 16 and 18 come from two unrelated invented settings and are flagged as
examples; the core text and its vocabulary are setting-agnostic. In the rules a
resource is a **stock** of **units**; the words for what it stands for (land,
forest, ore, ice) appear only in the examples.

## 1. Purpose and scope

A recipe that gathers or grows something takes it from somewhere. Until this spec,
that somewhere was either unlimited (a source recipe, limited by labour only,
`spec-production.md` 3.1.2) or a seam left open (`Draws`, 6.5). This spec fills the
seam and owns what lies behind it. It covers:

- what a **resource** is, how it differs from a commodity, and what a draw turns
  into (section 3);
- the state of an **endowment** (a resource in a region): a stock, a capacity, a
  regeneration with three shapes and a carried remainder, a seasonal profile and an
  **abundance curve** that makes yield fall as the stock does, all in integers
  (sections 3 and 4);
- the seam that production calls, the order in which agents compete for a scarce
  stock, and what a cohort gets (section 5);
- depletion that is permanent and depletion that is recoverable, how both become
  visible although the ledger sees neither, and the shocks a consumer may apply
  (section 6);
- who owns the values of the yield table, which settles note N-030 for RES
  (section 7);
- the **fixed factor** that the population loop needs (note N-098): how output per
  household falls as households rise, the stability condition, and what is left to
  population (section 8);
- the sourcing of a raw good (section 9, note N-021), open access and the hook for
  tenure (section 10), and the rule the steady-state solver uses to turn a
  regeneration into a sustainable draw (section 11).

Out of scope, with the spec that owns it: trade, prices and information, which a
resource does not have (MER, INF); the outside world as a source of goods (EXT,
section 9 says how the two meet); who owns an endowment, rent and tenure (LND,
section 10 leaves the hook); household production as a floor under market failure
(SUB, which may draw on an endowment); fixed capital that raises capacity, such as
irrigation (CAP, which would use the capacity slot of 6.3); the public calls that
set a shock (INT); the tick phases (WLD, section 5.2 states what this spec needs);
the bands that say what a healthy resource curve is (STY).

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is a resource, and is it a commodity? | A **resource** is a kind of regional endowment, defined as data. It is **not a commodity**: it has no price, no mass, no reference price, no ledger account, is never held by an agent and is in no recipe's `Inputs` or `Outputs`. A **draw** consumes units of an endowment; the recipe's output is a commodity. The two meet only in `Draws` | One owner per fact. A good that is bought and sold is a commodity (GDS); a limit on how much of it can be made from the ground is a resource. The ledger conserves what is in accounts, and an endowment is in none (3.1) |
| 2 | Is it a module? | Yes: `Modules.Resources`, present means on. The values of the yield table are owned here and stay core data that works with the module off | ADR 0001 decision 2; the zero configuration is the tested baseline. The yield table is needed by the first slice (one region, a seasonal crop) before any depletion exists (7) |
| 3 | What is the state? | Per (region, resource): a **stock** of whole units between 0 and a **capacity**, a regeneration **carry**, and a count of consecutive ticks in which a request was cut. All `int64` | Integer fixed point (architecture.md 3.5); the carry is what lets a slow regeneration work at a tick that is short against it (4.3) |
| 4 | How does a stock regenerate? | One of three **shapes**, a closed set: `constant` (a flow, zero for a finite stock), `refill` (the gap to capacity closes at a rate) and `logistic` (growth proportional to the stock and to the room left). Each has an annual rate in the unit of its shape, a region scale, and an optional factor by season | The three behave differently in the loop of section 8 (neutral, damped, tipping), so they are not three values of one formula. A formula chosen by the engine would be a setting's opinion; a shape is a rule |
| 5 | Is depletion permanent? | It is **data**. A `constant` stock with rate 0 is finite and its depletion is permanent. A `refill` stock with a positive rate always regrows. A `logistic` stock regrows unless it reaches 0 with no **refuge**; with a refuge it recovers slowly. The consumer may restock by an adjustment (6) | The settings differ (a finite deposit against a recovering stock), and the library should not choose. Two unrelated worked cases show both (16) |
| 6 | How does a draw reduce output besides running out? | Through the **abundance curve**: a list of knots from stock share to a yield factor, evaluated by integer interpolation, with a default of flat 10 000. The factor is the fourth factor of the effective yield of `spec-production.md` 6.3, read at job start from the stock as it stood when the production phase began | A hard quota is a cliff; a curve gives the soft landing that a real deposit or field has (a poorer grade, a thinner crop) and the smooth diminishing returns that the Malthusian loop needs (8). Flat by default keeps the quota alone |
| 7 | Who gets a scarce stock? | The **ration** of 5.3: demands are summed per (region, resource); if the stock covers them all, all are granted; if not, each agent gets its proportional share of executions, and the leftover units go one execution at a time to the agents with the largest fractions, ties by a stateless hash key of (tick, agent). It is fair over runs for requests of equal size and proportional in every tick for any mix. It replaces the rotation of `spec-production.md` 5.5.1 for draws only | The rotation is fair over a long run and not within a short one: in the model a household was denied for a whole year at half supply (17.2). A proportional share is weight-invariant (a cohort of 50 gets what 50 households would), takes no random draw and keeps no state |
| 8 | Which numbers does the seam expose? | `Endowment(region, resource)` returns the stock; `TakeEndowment(region, resource, n)` takes `n` units whole or fails with `ErrInsufficientEndowment`. The first is a public query, the second is internal to production. A consumer changes a stock only through the adjustment slot of 6.3 | The brief of PRO 6.5 fixes the two names. A public take would be a back door that moves a stock outside the tick (architecture.md 10.2) |
| 9 | Does the ledger see it? | **No**, and it must not. A draw and a regeneration are not transfers; no reason is added to the registry. Depletion is visible by events, a counter of the cut demand and a `Warn`, and an exact identity per endowment per tick that a test and the audit check | Goods in accounts are conserved; a stock in the ground is not goods. A new faucet or sink in the ledger would misstate matter that never was in an account (N-107). The identity is the conservation law of this state |
| 10 | Who owns the values of the yield table? | **This spec.** PRO keeps the format, the lookup and the validation errors. There is no climate entity and no function from richness to yield: a climate is the rows of the table, and richness is the capacity and the regeneration scale of an endowment | N-030: climate is data. A mapping would put a second table in front of the first and say the same thing |
| 11 | What fixes the Malthusian loop? | The fixed factor of section 8: an endowment that binds. With a flat curve, output per household is the endowment's flow over the households once they exceed `N_bind`; with a curve it falls as the stock does, and the equilibrium stock and population have a closed form. Stability depends on the shape (8.3) | N-098. The result is a condition the validator can warn about and a model that agrees with the linearisation (17.3) |
| 12 | How does a raw good get a source? | A source recipe with `Draws` on an endowment that can start at least once is sourced by this module; an import is sourced by EXT; a source recipe with no `Draws` is sourced by itself. The marker `Raw` alone stays a warning-bearing marker (9) | N-021. Validation sees the union and warns where a source can run out |
| 13 | Open access or owned? | Open access in version 1: any agent of the region whose role lists a recipe that draws may draw, and the ration is the only arbiter. The hook for tenure is a refusal and a per-claimant entitlement that LND adds (10) | LND is later (`vision.md` 4). The open-access result is the baseline that tenure is measured against |
| 14 | Is a draw returned when a job is abandoned? | No. The units were taken at the start, like the inputs, and an abandoned job is a loss of what it used (`spec-production.md` 5.4.1) | A refund would let an agent hold a stock for nothing and would need state |
| 15 | Which shocks? | Two world slots set between ticks, saved in a checkpoint: a scale on the capacity and a scale on the regeneration of one endowment (a sudden loss of capacity, a slowed recovery), and a one-off adjustment of the stock (a collapse, a discovery). INT names the calls | N-091, N-036. The failed harvest of `vision.md` 2 stays the yield shock of `spec-production.md` 6.8; these are the shocks of the ground |
| 16 | What does the solver use? | The **sustainable draw**: the regeneration at the **plan share** of the capacity for a renewing stock, and the starting stock over a stated horizon for a finite one. One linear row per (resource, region) and a constant abundance factor (11) | A steady state needs a flow; the stock level it holds is a choice the pack states |
| 17 | Does this add feedback loops? | Yes: the resource-population loop, a tipping point below the maximum sustainable draw, open-access overuse through recipe choice and entry, the cliff of a finite stock, the opening of a season against the stock, and scarcity through the cost anchor (19) | `spec-stability.md` 9: a spec that adds a loop and no entry is not done |

## 3. Resources and endowments

### 3.1 What a resource is

3.1.1. A **resource** is an entity of the configuration with an ID, a **shape** and
the data of section 12. It stands for anything a recipe can run out of that is not
a good in an account: anything that is a limited stock in the ground, in the water, in a
volume of space. The core says nothing about which.

3.1.2. **A resource is not a commodity.** It has no `Mass`, `Bulk` or reference price
(`spec-goods.md`), no account, no lot, no market and no price belief. It is never in
the `Inputs`, `Outputs`, `Requires` or `Upkeep` of anything, it is not in any
role's tradable set, and no agent holds it. A resource ID and a commodity ID are in
different namespaces (`spec-config.md` 3.3: unique within its kind), so a pack may
call both `ice`; the validator does not forbid it and the file layout keeps them
apart (section 12).

3.1.3. **What a draw turns into.** A recipe's `Draws` lists (resource, quantity)
pairs. At the start of a job of `k` executions the world takes `k` times each
quantity from the region's endowment of that resource (5). The job's outputs are
commodities, as always, created at completion by the ledger reason `recipe_output`
(`spec-production.md` 5.3). The relation between the two quantities is the pack's:
a draw of 30 units that yields 30 units of a commodity at an abundance of 10 000 is
a pack that measures its stock in the unit of its output; a draw of 1 that yields
1 500 is a pack that counts plots. The core does not relate them, and the effective
yield (`spec-production.md` 6.3, with the abundance factor of 4.7) scales the output
and never the draw.

3.1.4. **No code branches on a resource ID** (architecture.md 9.1). The behaviour
differs by shape, a closed enumerated set like the flow shapes of
`spec-money-supply.md` 4 and not an identifier, and by the numbers in the data.

### 3.2 An endowment and its state

3.2.1. An **endowment** is a resource in one region. Its state is plain data:

| Field | Meaning | Range |
|-------|---------|-------|
| `Stock` | units available now | 0 to the effective capacity |
| `Carry` | the fraction of a unit regenerated and not yet added, in units of `1/D` (4.3) | 0 to `D - 1` |
| `ExhaustedRun` | consecutive ticks in which a request on this endowment was cut | 0 or more |

The **capacity** is the most the stock can hold. The effective capacity is the
configured capacity times the capacity scale of the shock slot over 10 000,
truncated (3.4).

3.2.2. The endowments are a dense array indexed by (`RegionID`, `ResourceID`), built
at construction from the configuration (3.3), so a lookup in the tick loop is a read
and allocates nothing (`spec-regions.md` 5.3.5). A pair with no capacity has a
capacity of 0 and a stock of 0, and a draw on it is impossible.

3.2.3. **`ResourceID`** is the index of a resource in ascending order of its ID
string, bytewise, after canonicalisation (`spec-config.md` 3.4), in the way a
`RegionID` is (`spec-regions.md` 3.3):

```go
// ResourceID is the index of a resource in ascending order of its ID string.
// It is not stable across configurations; use the string ID to persist or exchange.
type ResourceID int
```

Every iteration over endowments, in the library and in every output ordered by
endowment, is by ascending `RegionID` and then ascending `ResourceID`, and no
iteration over a map may influence a result (architecture.md 7.5).

### 3.3 The module and its rows

The module is `Config.Modules.Resources`, a pointer: nil is off, non-nil is on
(`spec-config.md` 3.2). It needs the regions (always present) and production.

```go
// ResourcesConfig is the resources module. Plain data.
type ResourcesConfig struct {
	Resources   []Resource  // sorted by ID
	Endowments  []Endowment // sorted by (region, resource): the region-keyed table
	HorizonYears int64      // viability horizon for a finite stock (11)
}

// Resource defines a kind of regional endowment.
type Resource struct {
	ID          string
	Shape       string          // "constant" (default), "refill" or "logistic"
	RatePerYear int64           // constant: units a year; 0 is a finite stock
	RateBP      int64           // refill, logistic: annual rate in basis points
	Refuge      int64           // logistic: units that regrow even when the stock is 0
	Capacity    int64           // default capacity of an endowment of this resource, in units
	InitialBP   int64           // default starting stock as a share of the capacity
	PlanShareBP int64           // the stock share a steady state is assumed to hold (11)
	Abundance   []AbundanceKnot // stock share to yield factor; none is flat
	Seasons     []SeasonRegen   // regeneration factor by season; none is 10 000 always
	Label       string          // presentation only; not hashed
}

// AbundanceKnot is a point of the abundance curve, both in basis points.
type AbundanceKnot struct{ Share, Factor int64 }

// SeasonRegen is the regeneration factor of one season, in basis points.
type SeasonRegen struct {
	Season string
	BP     int64
}

// Endowment is the row of one region and one resource.
type Endowment struct {
	Region, Resource string
	HasCapacity      bool  // Capacity applies
	Capacity         int64 // replaces the resource's default capacity
	HasInitial       bool  // Initial applies
	Initial          int64 // replaces the default starting stock, in units
	HasRegenScale    bool  // RegenScaleBP applies
	RegenScaleBP     int64 // scales the regeneration of this resource here; 10 000 is the resource as written
}
```

3.3.1. **The rows are the override mechanism of `spec-regions.md` 5 (note N-104).**
`Endowments` is a region-keyed table: one row per (region, resource), owned by this
spec for meaning, default and bound. `Capacity` and `Initial` use the **replace** mode
(the row's value, else the resource's default capacity or its `InitialBP` share of
the capacity); `RegenScaleBP` uses the **scale** mode (the resource's rate times the
row's factor over 10 000, default 10 000; only the effective scale m of 3.4 is truncated, the rate is not). Absence is the default, with no
warning: a resource's default capacity is 0, so a region that has no row has none of
the resource, which is the intended way to say that a region has no field or no
deposit. A row is resolved once into the dense array of 3.2.2; the canonical order is
ascending `RegionID` then `ResourceID`, in the hash and the checkpoint; the errors
are `ErrUnknownRegion`, `ErrUnknownResource` and `ErrDuplicateID` (`spec-regions.md`
5.3.1, 5.3.2). The row needs a line in `spec-regions.md` 5.1.3 (handoff). The shock
slots of 6.3 edit no row (`spec-regions.md` 5.3.6).

3.3.2. **File layout** (`spec-config.md` 6.2; the spelling of the module table is
open until N-102 is closed): the module is the table `[resources]`, a resource is
`[resources.resource.<id>]` and a row `[resources.endowment.<region>.<resource>]`, in
the way `[demand.want.<id>]` and `[demand.region_taste.<region>.<want>]` are, so that
layers merge by key (`spec-config.md` 6.4). The yield table is unchanged:
`[region.<id>.yield.<recipe>]`.

3.3.3. **Canonical order:** resources by ID, knots by ascending share, season
entries in the calendar's season order, rows by (region, resource).

### 3.4 Effective values

The shock slot (6.3) holds, per endowment, a capacity scale and a regeneration scale
in basis points, 10 000 each by default. The values the rules use are

```
C   = min( MaxEndowment, trunc( capacity_row * capacity_scale / 10 000 ) )   effective capacity
m   = trunc( RegenScaleBP * regen_scale / 10 000 )             effective regeneration scale
```

with `capacity_row` the row's capacity or the resource's, and `RegenScaleBP` the row's
or 10 000. Both are computed with the checked 128-bit helper (`spec-currency.md` 7)
and are at most `MaxEndowment` and `MaxRegenScaleBP * MaxShockBP / 10 000`
respectively (4.6).

## 4. Regeneration and the abundance curve

### 4.1 When

4.1.1. Regeneration runs **once per endowment per tick, as the last step of phase 8**
(`spec-agents.md` 4: ageing and spoilage, flow totals, then this, then the optional
audit), in ascending (`RegionID`, `ResourceID`) order, using the season of that tick
(`Calendar.At(tick).Season`, -1 for none). The draws of a tick come earlier, in phase
2, so the stock that regenerates is the stock the draws left. The event of 14.1 is
emitted after it. The order is WLD's to adopt (N-009); this spec needs only that the
draws of a tick precede its regeneration and that no draw happens between them.

4.1.2. A world with the module off has none of this and no state (FR-RES-001).

### 4.2 The three shapes

Let `S` be the stock, `C` the effective capacity, `m` the effective regeneration
scale in basis points and `s` the factor of the tick's season in basis points (4.5).
Let `TPY` be `TicksPerYear` (`spec-calendar.md` 3.2). Every rate is **annual**, so
that a pack is stated once and the tick length follows the calendar, as for the
population rates (`spec-population.md` 3.8). The regeneration of one tick is a
**numerator** `n` over the fixed denominator

```
D = 10^12 * TPY                 (at most 10^17 for TPY at its bound, below int64)
```

| Shape | Meaning | Annual regeneration | Numerator `n` of one tick |
|-------|---------|---------------------|---------------------------|
| `constant` | a flow that does not depend on the stock; with rate 0 a finite stock | `RatePerYear * m * s / 10^8` units | `RatePerYear * m * s * 10^4` |
| `refill` | the gap to capacity closes at a rate | `(RateBP / 10^4) * (C - S) * m * s / 10^8` units | `RateBP * (C - S) * m * s` (0 if `S >= C`) |
| `logistic` | growth proportional to the stock and to the room left | `(RateBP / 10^4) * w * m * s / 10^8` units | `RateBP * w * m * s` |

where, for `logistic`, with `S' = max(S, min(Refuge, C - 1))` (a refuge is capped below the effective capacity, so a capacity shock cannot freeze regrowth),

```
w = ceil( S' * (C - S') / C )        if 0 < S' < C
w = 0                                if S' = 0 or S' >= C
```

The formulas of the middle column are the meaning and are analysis, not rules; the
integer rule is the last column and 4.3. The products can reach 10^31 (4.6) and use
the checked 128-bit helper.

4.2.1. **Why three.** `constant` is a source of flow, or a finite deposit when its rate is 0: its
regeneration does not respond to its stock, so with a population that responds to
the stock the linearised loop is neutral (8.3) and the nonlinear loop cycles with an
amplitude set by the start (17.3). `refill` is a stock that recovers toward a capacity: the more
is missing, the faster it comes back, so a displaced stock returns monotonically and
the loop is damped. `logistic` is a stock that grows from itself: growth is largest at half the capacity, falls on both sides, is 0 at 0
(extinction is absorbing unless there is a refuge) and has a maximum, so a draw above
the **maximum sustainable draw** `RateBP * C / (4 * 10^4)` a year (at `m = s = 10^4`)
collapses the stock whatever else happens (8.3).

4.2.2. A shape is a rule, not a table of data; adding a fourth shape is a change of
this spec. The set is closed so that every shape has a stability condition in 8.3.

### 4.3 The carried remainder and the rounding

4.3.1. The carry is added to the numerator, whole units are taken off, and the rest
is kept:

```
total = Carry + n
add   = total / D                 (integer division; all quantities are non-negative)
Carry = total mod D
S     = S + add
```

This is exact: over any run of ticks in which the capacity is not reached, the units
added are `floor((Carry_start + sum of the numerators) / D)` whatever path the stock
took (FR-RES-010; model 1 checks it on 400 random runs). There is no rounding bias
and no random draw. A rate as small as 1 basis point a year on a stock of a few units
is representable: the first unit comes after the time it takes.

4.3.2. **Why a carry.** Without it a slow regeneration is truncated to zero every tick
and never happens. In model 1 a logistic stock of 1 to 100 units at 5% a year on a
capacity of a million stays at its starting value for twenty years without the carry
and grows to 2 to 270 with it. A tick that is short against the regeneration (a
365-tick year) makes this the common case, not an edge.

4.3.3. **The logistic factor is rounded up.** `w` is a whole number of units rounded
**up**, so that any live stock of at least one unit and below the capacity has
`w >= 1` and can regrow. Rounding down gives `w = 0` at a stock of one unit, and one
unit below the capacity, where the true value is just under one: a single unit could
never grow, an absorbing state created by arithmetic and not by the pack. The error is
less than one unit in `w`, always in the direction of recovery, and it is a rule constant. `Refuge` (units, default 0) sets the stock the
growth acts on when the real stock is below it: with a refuge of `R`, a stock of 0
grows at the rate of a stock of `R`. A stock of 0 with no refuge is **extinct**: its
numerator is 0 for ever (permanent depletion, 6.1).

4.3.4. All arithmetic is integer with the rounding stated: floor for the add and the
carry, ceiling for `w`, floor for the effective capacity and scale (3.4). No
floating point affects a result; the real-number formulas in this spec are analysis.

### 4.4 The clamp at capacity

4.4.1. If `S + add >= C` the stock becomes `C` and the carry becomes 0: a full stock
banks no regeneration. Without this a stock held at capacity for a season would pay
out a lump when the first unit was drawn.

4.4.2. If `C` is 0 the stock is 0, the carry 0 and the numerator is not computed.

4.4.3. A stock above the effective capacity can arise only when a capacity shock
lowers it (6.3); the shock clamps it at once and the excess is reported as lost.

### 4.5 Seasons

4.5.1. A resource may carry a regeneration factor by season, `Seasons`, a list of
(season ID, basis points) from 0 to `MaxSeasonRegenBP`. The rule is the one of the
yield table (`spec-production.md` 6.2): **a profile is complete**: with a profile, a
season it does not list has a factor of 0; with no profile the factor is 10 000 in
every season, and with no seasons configured (`Season` -1) a profile is rejected
(`ErrUnknownSeason`) and the factor is 10 000.

4.5.2. This is how regeneration follows the year: a soil that recovers only in the
fallow half lists those seasons at twice the rate (example A, 16.1); a stock that
breeds in one season lists it alone. The annual total is the pack's choice and is
`sum over ticks of s / (10^4 * TPY)` of the flat rate, which the solver uses (11.1).

4.5.3. The season is global (`spec-calendar.md` 5.3): regions differ by their rows
(`RegenScaleBP`) and not by a cycle of their own, as for yields.

### 4.6 Bounds and overflow

All are constants in `bounds.go` (`spec-config.md` 4.1), provisional until the slice
has run, and chosen so that every product fits in 128 bits and every quantity that is
stored fits in `int64`:

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxResources` | 256 | resources in the module; with `MaxRegions` it bounds the dense array (256 000 entries) and the per-tick loops |
| `MaxEndowments` | `MaxRegions * MaxResources` | rows of the table; derived |
| `MaxEndowment` | 10^15 units | a capacity, a starting stock, an adjustment; larger than `MaxQuantity` because a stock in the ground can exceed any holding. With `MaxWeight` times a draw already at most `MaxQuantity` (`spec-production.md` 10.3), the draws of a tick fit |
| `MaxRegenRatePerYear` | 10^15 units | the rate of a `constant` stock |
| `MaxRegenBP` | 100 000 | the annual rate of `refill` and `logistic`: ten times a year |
| `MaxRegenScaleBP` | 100 000 | a row's regeneration scale |
| `MaxSeasonRegenBP` | 100 000 | a season's factor |
| `MaxShockBP` | 100 000 | a shock's scale, ten times |
| `MaxHorizonYears` | 1 000 | the viability horizon of a finite stock |

The largest numerator is `MaxRegenBP * MaxEndowment * (MaxRegenScaleBP * MaxShockBP / 10^4) * MaxSeasonRegenBP`,
about 10^31 (below 2^127, about 1.7e38) and the largest `D` is 10^17 (below 2^63);
model 1 checks the numerator for scales up to 10^5 (10^30) and the denominator; the bound above is the arithmetic of the product. The 128-bit intermediate cannot overflow, and `add` is clamped to `C - S` in 128 bits before it is narrowed to `int64` (it can exceed `int64` at `TPY` 1 otherwise); nothing is ever wrapped (architecture.md 3.5). A per-tick growth fraction above 1 would overshoot
the capacity in one step: the clamp of 4.4 makes it harmless, and the warning
`regen_per_tick_high` (13.3) flags an effective fraction above 2 000 basis points a
tick as a rate that is stiff against the tick.

### 4.7 The abundance curve

4.7.1. The **abundance** of an endowment is a yield factor from 0 to `MaxYieldBP`
(`spec-production.md` 10.1), read from the stock share. The share is

```
share = floor( Stock * 10 000 / C )          (C > 0)
```

and the factor is the curve evaluated at it. The curve is a list of 1 to `MaxKnots`
knots (`spec-config.md` 4.1) of (`Share`, `Factor`), shares strictly increasing from
0 to 10 000, factors from 0 to `MaxYieldBP`. **No knots means flat 10 000**, the
default, except that an endowment with C = 0 has abundance 0 whatever the curve (it can supply nothing). Outside the first and last knots it is flat at their factors; between knots
`i` and `i+1`, with `dx = x_{i+1} - x_i`,

```
factor = y_i + trunc( (y_{i+1} - y_i) * (share - x_i) / dx )
```

integer, truncation toward zero, products with the checked 128-bit helper. This is
the evaluation of `spec-population.md` 3.5.1, stated again because a change to a
rounding rule is a behaviour change (architecture.md 3.5); the code shares one
helper (`internal/curve`, 24).

4.7.2. **A curve need not rise, and the analysis of 8 assumes it does.** A curve that
falls somewhere is legal and warned (`abundance_not_monotone`, 13.3): a yield that
falls as a stock fills is a setting's choice (a crowded stand grows poorly), but the
fixed-point and stability formulas of 8 do not hold for it.

4.7.3. **Which stock.** The abundance a job uses is the one of the **snapshot** taken
when the production phase begins (5.2): the stock before any draw of the tick. It is
the same for every agent in the tick, so it does not depend on the order of the
starts, and it is the same value the agent's view shows when `Work` is called. It is
read when the job starts, with the season and the site factor, and the pending outputs
are fixed then (`spec-production.md` 6.4): a job that started at a high abundance
completes at that yield even if the stock has since fallen. The cost anchors of phase 1
(`spec-production.md` 9.3) read the same stock, because nothing changes it between the
regeneration that ends one tick and the snapshot that opens the next, apart from the shock
slots, which act between ticks, before phase 1.

4.7.4. **Several draws.** A recipe with several draws has the abundance of the
**least** of its resources, the minimum of the factors (the scarcest input limits the
yield). A product would compound the penalties of independent resources; the minimum
bounds the effect and is simpler to explain. It is a rule constant.

4.7.5. **What the factor does and does not do.** It scales the output (it is a factor
of the effective yield `E`, `spec-production.md` 6.3) and so enters the cost anchor
(`spec-production.md` 9.1: `unit_cost` divides by `E`): a poorer stock makes a unit
dearer to make, which is how scarcity reaches prices through the belief bounds
(register entry RES-A6, 19). It does **not** scale the draw: a job takes its full draw
whatever the abundance, so a depleted stock is drawn down as fast as a full one
(a grade story, not a catch-per-effort one; open question 2).

## 5. The seam with production

### 5.1 The two operations

```go
// Endowment returns the units of the resource available in the region now.
// It returns 0 for an unknown region or resource and for a pair with no
// endowment. It is a read, safe between steps.
func (w *World) Endowment(region RegionID, resource ResourceID) int64

// takeEndowment takes n units whole, or none, and returns ErrInsufficientEndowment
// when n exceeds the stock. n must be 1 or more. Production alone calls it,
// during the start of a job; it is not exported.
func (w *World) takeEndowment(region RegionID, resource ResourceID, n int64) error
```

The signatures of `spec-production.md` 6.5 used `int`; the typed IDs are the same
indexes. `Endowment` is public so that a consumer can display and test a stock; the
only public writers are the slots of 6.3, which are INT's calls (N-091). A draw is
not a ledger flow (3.1, 6.2). `ErrInsufficientEndowment` is a root sentinel mapped
from the internal package (N-075).

### 5.2 The tick, for draws

The part of the production phase that concerns draws runs in this order, after the
continuations and completions of `spec-production.md` 5.5.2 and before the starts. It
adds two steps to a phase that already has `Work` and the starts, and changes no
other step of production.

1. **Snapshot.** For every endowment, in (`RegionID`, `ResourceID`) order, the world
   records the stock `A` and the abundance `a` (4.7). They are fixed for the tick.
2. **Work.** `Work` is called for every free agent, in the order production already
   uses. For each request the world computes the executions `k0` the agent could
   start **ignoring draws**: the least of its weight (or the strategy's maximum), the
   stock of each input, the unreserved tool items, from the state after the
   continuations and completions. A request with an effective yield of 0 (which
   includes an abundance of 0) is refused with `zero_yield` or `no_site`, and a
   request with `k0` of 0 for another reason is refused with that reason
   (`spec-production.md` 4.2). The results of `Work` do not depend on the order of
   the calls, because the only shared state they read, the endowment, is the
   snapshot (FR-RES-015).
3. **Ration.** For each endowment that some request draws on, in order, the requests
   are collected in agent-ID order and rationed (5.3). Each request gets, per draw, a
   number of executions it may start.
4. **Start.** The starts proceed in the rotation order of `spec-production.md` 5.5.1.
   The executions started are `k = min(k0, the grant of each draw)`, then limited by
   the live inputs as production already does. The job takes `k` times each draw from
   the live stock (5.4). If `k` is 0 for lack of a grant, the refusal reason is
   `no_endowment`, after `zero_yield`, `no_site`, `no_tool` and `no_input` in the
   order of `spec-production.md` 4.2.

The rotation of `spec-production.md` 5.5.1 therefore keeps ordering the starts and
no longer decides who gets a scarce stock. A start that takes nothing from an
endowment is not changed. WLD adopts the snapshot at the start of the production
phase and the ration before the starts (handoff, N-009).

### 5.3 The ration

5.3.1. **Input.** For one endowment with available stock `A`, the requests are `(agent,
w_i, d_i)`: `w_i` executions wanted and `d_i` units drawn per execution, both at least
1. The units wanted are `U = sum of w_i * d_i`, checked.

5.3.2. **Rule.**

1. If `U <= A`, every request is granted `w_i`.
2. Otherwise let `p_i = floor( w_i * A / U )` and `phi_i = (w_i * A) mod U`. Each agent
   is granted `p_i` executions, which uses `sum of p_i * d_i <= A` units; let `R` be
   what is left.
3. The agents with `p_i < w_i` are ordered by **descending `phi_i`**, then **ascending
   key** (5.3.3), then ascending agent ID. In that order each agent whose `d_i <= R`
   is granted one more execution and `R` falls by `d_i`; an agent whose `d_i` exceeds
   `R` is skipped and the walk goes on. One pass; what remains is not taken.
4. The grant of an agent is at most `w_i`, and the sum of the units granted is at most
   `A`.

The order of step 3 is a total order (it ends in a unique ID), so the result does not
depend on how the requests were listed (architecture.md 7.6). The products `w_i * A`
use the checked 128-bit helper.

5.3.3. **The key** is a stateless function of the tick and the agent, with no draw
from the world's random source and no state of its own:

```
x   = tick * 0x9E3779B97F4A7C15 + agent        (mod 2^64)
z   = x + 0x9E3779B97F4A7C15                   (mod 2^64)
z   = (z xor (z >> 30)) * 0xBF58476D1CE4E5B9   (mod 2^64)
z   = (z xor (z >> 27)) * 0x94D049BB133111EB   (mod 2^64)
key = z xor (z >> 31)
```

(the finaliser of the SplitMix64 generator). It is a rule constant. The random stream
of a world does not depend on whether the module is on (FR-RES-045, as FR-POP-036 and
NFR-POP-004 require of that module).

5.3.4. **Why this rule and not another.** The model of 17.2 compares three rules on
the same demand.

- *First come, first served in the rotation order*, the rule production already has: at
  half supply each agent is served in a contiguous half of a thousand-tick cycle, so a
  household can be denied for a whole year while its neighbour is served for all of it
  (the minimum share over a year is 0.000, the longest denial 365 ticks at 0.3 and 0.5
  supply, 100 at 0.9). A job-holding household that cannot restart for a year goes
  unfed, and the failure ladder (`spec-spawner.md` 5) begins on a rule's artefact.
- *Proportional share with leftovers to the rotation order*: for agents of weight 1
  every fraction is equal, so the leftovers go in the rotation order and the result is
  the first rule again.
- *The rule above*: for agents of weight 1 the leftovers go by the hash key, which is
  a different permutation each tick, so over a year each agent gets between 0.42 and
  0.58 of the demand at half supply, and the longest denial is 20 ticks. Cohorts get
  their proportional share (exactly when `w * A / U` is a whole number). It is fair in expectation and in runs for requests of equal size. **It is not fair in runs for a fixed mix of sizes**: the fractional order is the same every tick, so with 25 agents asking 2 and 50 asking 1 against a supply of 25 the larger requests take the leftovers every tick and the smaller get 0 for ever, below their proportional share of 0.25. The proportional part is exact in every tick; the leftover is the part that can be biased. The cost is bounded by the leftover (less than the number of agents), and a carried remainder per agent would remove it (open question 4). RES-S4 tests a mixed mix.

5.3.5. **Weight invariance.** Twenty cohorts of weight 50 and a thousand agents of
weight 1 with the same total demand get the same total, and a cohort's share per
household is its proportional share to within one execution per tick
(`spec-scale.md` 4.2, FR-SCL-007; FR-RES-021). First come, first served does not have
this property.

5.3.6. **What it costs.** The pass is `O(n log n)` in the requests on one endowment,
and only when the stock is short: with `U <= A` it is one sum. The fractional order
is by `phi_i`, so an agent that always asks for a small number of executions has a
smaller chance at the leftovers than one that asks for many; the bias is at most the leftover of the tick (fewer executions than agents) and is zero for equal requests, but it repeats every tick while the mix repeats (5.3.4). A carried remainder per
agent would remove it at the price of state; it is not used (open question 4).

5.3.7. **An example** (checked in `models/res/worked.py`, 17.5). Ten plots, five
agents want A 4, B 3, C 3, D 2, E 1 executions of a draw of 1: `U = 13`. The floor
shares are 3, 2, 2, 1, 0 (`w * 10 / 13`), using 8 of the 10; the remainders are
1, 4, 4, 7, 10 (of 13). The 2 left go to E (10) and D (7): the grants are A 3, B 2,
C 2, D 2, E 1.

### 5.4 Executions, cohorts and the start

5.4.1. **All or nothing.** The start of a job takes its draws with `takeEndowment`,
one call per draw, after checking that the live stock covers every one of them; if
any does not, nothing is taken, the job does not start, and the reason is
`no_endowment` (FR-RES-018). Because the grants never exceed the snapshot and the
starts only take what was granted, this cannot happen in a correct run; it is stated
so that a second spec that takes from an endowment keeps the same contract.

5.4.2. **Several draws.** A recipe with two draws is in two rations. Its grants are
independent, and `k` is the least of them. The units granted in the larger ration and
not used are not taken and stay in the stock for the next tick: the allocation does
not iterate to a joint optimum (open question 5). The loss is bounded by one pass.

5.4.3. **Cohorts.** A cohort of weight `w` requests `k0 <= w` executions; its grant is
proportional to its request like any other, so `k` households work and the rest
cannot, as `spec-scale.md` 4.2 describes. The draw is `k` times the quantity,
whatever the weight.

5.4.4. **Trainees.** A trainee's job draws from the endowment of the executor's region
and its yield is scaled by `TraineeYield` as before (`spec-production.md` 8); its
request is in the ration like any other.

5.4.5. **An abandoned job, or a job whose executions fall because the executor's weight falls**, does not return its draw (position 14); the units are
already counted as drawn in the tick of the start.

5.4.6. **A region change** after the start changes nothing: the draw was taken from
the region the executor was in at the start.

### 5.5 What a strategy sees and what production changes

5.5.1. **The agent view** (`spec-agents.md` 6.4) carries, per recipe, the effective
yield `E` and `MaxExecutions` with its reason. With this module `E` includes the
abundance (the snapshot of 4.7.3) and `MaxExecutions` is the least of the existing
limits and, for each draw, `floor(A / d)` of the snapshot (a bound that ignores the
competition of the ration), with `no_endowment` as the reason when that is 0. The
view also gains a group `Endowments` (only with the module on), **for the agent's own
region only**: the endowments that its role's recipes draw on, including the recipes of its effective own-use list (`spec-subsistence.md` 3.6), sorted by resource, each
with `Resource`, `Stock`, `Capacity` and `AbundanceBP`. It is local physical state, free like the local price, and it is not a
macro measure that agents react to (roadmap, "Macro is measured, never set"). **No
view carries the endowment of another region** (note N-106): a strategy that wants a
remote stock needs a rumour, which INF may define and which this spec does not.

5.5.2. **Edits to `spec-production.md`** (handoff h): 3.1 `Draws` row (the resource is
a `Resource` of this spec); 4.2 and FR-PRO-009 (the draw term is the grant of 5.2, not
the live availability); 5.5.1 (the rotation orders starts and does not arbitrate a
scarce stock); 6.3 (the effective factor gains `abundance`); 6.5 (the seam text points
here); 10.2 (`ErrUnknownResource`); 10.4 (`unrunnable_recipe` also covers a recipe
whose every draw exceeds the capacity where it has a positive yield); 10.3 (the output overflow check uses the largest yield factor times the largest abundance, up to `MaxYieldBP` over 10 000); 5.2 (continue or start per agent becomes continue and complete, `Work` for all, the ration, then the starts); 6.3 (the line `E = E * abundance / 10 000` after the trainee step, truncated like the others, so the order is site, shock, trainee, abundance); 16 and STB-L29.

## 6. Depletion, recovery and shocks

### 6.1 Permanent or recoverable: a matter of data

| Shape and data | Depletion | Why |
|----------------|-----------|-----|
| `constant`, `RatePerYear` 0 | permanent | nothing adds to the stock; a **finite stock**. The default when a resource names no rate |
| `constant`, `RatePerYear` above 0 | recoverable while the flow lasts | the flow adds regardless of the stock |
| `refill`, `RateBP` above 0 | recoverable | the gap is largest at 0, so regrowth is fastest there; it cannot be extinct |
| `logistic`, `Refuge` 0 | recoverable above 0, **permanent at 0** | a stock of 0 has `w = 0`; extinction is absorbing |
| `logistic`, `Refuge` above 0 | recoverable, slowly | at 0 the stock grows at the rate of a stock of `Refuge` |

In every case the consumer may restock by the adjustment of 6.3. Model 4 (17.4)
shows the four cases after a shock that sets the stock to 5% of the capacity: a
`refill` stock is back at 0.62 of the capacity after ten years, a `logistic` stock
with no refuge is at 0 for ever and its households fall to the floor, the same stock
with a refuge of 1% is back in about three centuries, and a `constant` flow does not settle: it cycles. The numbers are the model's. Two unrelated worked cases are in 16.

### 6.2 Visibility: the ledger does not see it

6.2.1. A draw and a regeneration change no account, post no transfer and need no
reason. The goods a job makes are created at completion by `recipe_output`, as always
(`spec-ledger.md` 7.2); what limits them lives here. **This spec adds no faucet, no
sink and no ledger reason** (note N-107: nothing here creates or destroys money or
goods; an endowment is not in an account).

6.2.2. Depletion is made visible by four means: the event `EndowmentChanged` (14.1),
once per endowment per tick in which a unit moved or a request was cut, with the stock
after the tick and the units requested, drawn, short, regenerated and lost; the
`Warn` of 14.2 at the first tick a request is cut and every tenth after; the counter
`Short` (and the agents denied and reduced), which is the counter that a halted recipe
needs (note N-045: a silent damper needs a counter); and the snapshot of 14.4. The
refusal of a start is not an event, as production says (`spec-production.md` 14.1: the
volume would be per agent per tick); the aggregate here is bounded by regions times
resources.

### 6.3 The shock slots

Two world slots (a capacity scale and a regeneration scale) and one adjustment of the
stock, set between ticks and saved in a checkpoint, with the public calls of
`spec-integration.md` 5.1 (note N-091): `SetCapacityScales` and `SetRegenScales`, whose
entries are `EndowmentScale{Region, Resource; BP}` (0 to `MaxShockBP`, default 10 000),
and `AdjustEndowments`, whose entries are `EndowmentAdjustment{Region, Resource; Delta}`.
None edits a configured row (`spec-regions.md` 5.3.6).

6.3.1. **The scales.** Per endowment, a capacity scale and a regeneration scale in
basis points, from 0 to `MaxShockBP`, default 10 000. A scale of 3 000 on the
regeneration slows recovery; a scale of 5 000 on the capacity halves what the endowment can hold. A change recomputes the effective values of 3.4. If the stock then
exceeds the capacity it is clamped to it and the excess is **lost**: it is reported in
`EndowmentAdjusted` (kind `shock`, with the stock before and after) and counted as
`Lost` in the identity of 6.4. A scale of 0 on the capacity empties the endowment and
makes every draw impossible.

6.3.2. **The adjustment.** A signed delta to the stock, clamped to [0, capacity]: a
negative delta is a collapse, a positive one a discovery or a restocking. The event is
`EndowmentAdjusted` (kind `adjust`). It is the only way a stock rises other than by
regeneration, and it creates no goods.

6.3.3. **The failed harvest stays a yield shock.** The multiplier of
`spec-production.md` 6.8 scales the output of a recipe in a region and leaves the
stock alone. The two are different shocks: a drought that halves the crop is a yield
shock; a drought that halves how fast the soil recovers is a regeneration scale; INT
may apply both.

### 6.4 The identity of an endowment

For every endowment and every tick, in integers,

```
Stock_end = Stock_start - Drawn + Regenerated - Lost + Adjusted
```

where `Drawn` is the units taken by starts, `Regenerated` the units actually kept after the clamp of 4.4 (not the units computed), `Lost` the units removed by a capacity clamp, and `Adjusted` the signed sum of the effective (post-clamp) adjustments applied between the ticks; the `Lost` and `Adjusted` of a call between ticks belong to the tick that follows it. A test asserts it over the reference packs and
a range of seeds, and `World.Audit` (`spec-ledger.md` 12.4) SHOULD check it, reporting
the endowment, tick and the difference (FR-RES-025). It does not need the ledger and
is the conservation law of the ground.

## 7. The yield table: who owns the values

### 7.1 The decision (note N-030)

7.1.1. **This spec owns the values of the yield table; production keeps the format.**
The table is `Config.Yields`, a list of `YieldProfile` keyed by region and recipe,
with an `All` factor and a factor by season, exactly as `spec-production.md` 6.1
defines it; its lookup (6.2), its effective factor (6.3), its validation errors
(10.2) and its type stay there. What moves here is the meaning of a value, its
authoring rules and the checks that read the whole table (7.4). The table is core
data: it works with the resources module off, and a pack with no endowment at all
still states its climates in it.

7.1.2. **No climate entity, no mapping.** A climate is not an object. It is the rows of
the table that belong to a region: the yield of each gathering or growing recipe, by
season. An alternative was considered and rejected: a function from a region's
"richness" or "climate class" to a yield, which would put a second table in front of
the first and say the same thing in a vocabulary that the core would have to define.
Richness is the same kind of data, in the place where it acts: the **capacity** and
the **regeneration scale** of an endowment (3.3). A rich region has a high capacity
and a quick scale; a favourable climate has a high yield in the seasons of the crop.

7.1.3. **Migration of the parameter-owner rows** (handoff b). The rows of
`specs/README.md` change owner:

| Parameter | Scope | Was | Becomes |
|-----------|-------|-----|---------|
| `YieldProfile` | Region and recipe | `spec-production.md` 6.1 (values move to `spec-resources.md` when it exists) | values and authoring: `spec-resources.md` 7; format and lookup: `spec-production.md` 6.1 |
| `Region`, `Recipe`, `All`, `Seasons` | Yield | `spec-production.md` 6.1 | `spec-resources.md` 7 (format: `spec-production.md` 6.1) |
| `Draws` | Recipe | `spec-production.md` 6.5 (resource values owned by RES) | the field: `spec-production.md` 3.1; the resources it names: `spec-resources.md` 3 |

The yield table has **no default and no placeholder in this spec**: absent means
10 000 for a recipe that is not `Sited` and 0 for one that is (`spec-production.md`
6.2), and every value is the pack's calibration.

### 7.2 What a value means

A value is a factor in basis points on every output of the recipe, read when the job
starts: 10 000 is the recipe as written, 0 is "cannot run", and 100 000 is the
bound (`MaxYieldBP`). Rules for authors, none of which the engine enforces beyond the
errors of `spec-production.md` 10.2:

1. **State the climate as ratios, not as absolutes.** The best region for a recipe is
   10 000 or above and the others are fractions of it, so that the recipe's written
   output is the best case and the numbers in a recipe stay readable. A pack that
   wants a richer land than the recipe says writes 15 000, as the invented iron seam
   of `spec-production.md` 12.1 does.
2. **A recipe that cannot run somewhere has no profile or `all = 0`.** An omitted
   profile on a `Sited` recipe is 0 as well; an omitted profile on a recipe that is not
   `Sited` is 10 000, which is probably not what was meant for a gathering recipe. The warning
   `unrunnable_recipe` (`spec-production.md` 10.4) catches a `Sited` recipe that runs
   nowhere.
3. **A seasonal profile is complete.** A season it does not list is a season in which
   the recipe cannot start (`spec-production.md` 6.2 rule 3). Reversed season orders,
   regions with fewer seasons and the rest are the lists of N-030 and `spec-calendar.md` 5.3;
   the world has one season cycle.
4. **A factor is not a limit.** How much of the stock a recipe may take is the
   endowment's, and how much the stock has been depleted is the abundance's; the yield
   table is the climate and nothing else (the site factor of `spec-production.md`
   6.3).

### 7.3 The seasons (note N-031)

7.3.1. A season is the index `Calendar.At(tick).Season`, or -1 for none, and every
table of this spec keyed by season is keyed by season ID: the regeneration profile
(4.5) and the yield table. No logic names a season (architecture.md 9.1). The RES
part of N-031 is applied: the table of yield by season, and now of regeneration by
season, is owned here; the WLD, OBS, INT and GDS parts stay with their specs.

7.3.2. The shock of a seasonal failed harvest is the yield shock of
`spec-production.md` 6.8 (INT's slot); the shocks of the ground are those of 6.3.

### 7.4 What the table must allow: comparative advantage

7.4.1. Differences in yield are what give trade a reason (`spec-viability.md` 9.1,
9.2; N-030: specialisation follows from the table). A table in which the regions are
scaled copies of one another gives none. For two regions `i` and `j` in one strongly
connected component of the region graph (`spec-regions.md` 4.3.5) and two `Sited`
recipes `a` and `b`, let `Ē(r, x)` be the mean over the year's ticks of the site
factor of recipe `r` in region `x` (`spec-production.md` 6.2 with the season of each
tick), in basis points, all four positive. The **cross ratio** is

```
Ē(a, i) * Ē(b, j) / ( Ē(a, j) * Ē(b, i) )
```

and it differs from 1 exactly when the regions have a comparative advantage in the two
recipes. The warning **`no_comparative_advantage`** (13.3) fires when the realm has at
least two such regions and two such recipes and no cross ratio, or its inverse, reaches
11 000 basis points (a relative difference of 10%). The threshold is a placeholder and
a rule constant, since a warning changes no result; so are the 2 000 bp of `regen_per_tick_high` and the cadence of the `Warn` (first and every tenth tick), which change no result. The check is `O(R^2 K^2)` for `R`
regions and `K` sited recipes and is bounded by the configuration;. `spec-reference-chain.md`
4.4.4 passes it by a wide margin (the cross ratio of `grow_staple` and `cut_fuel`
between `north` and `centre` is 1.46).

7.4.2. The check is advice to an author, not a claim that trade will occur: whether
anyone ships is MER's and the transport spec's.

## 8. The fixed factor of the population loop (note N-098)

### 8.1 What is needed

The Malthusian loop of `spec-population.md` 5 has three assumptions; the third is
that living standards fall as households rise. POP supplies the first two (the
schedules) and states that the third needs "a fixed factor": a limited, regenerating
thing that recipes draw on (5.2). That is an endowment. This section says how an
endowment makes the ratio fall with households, with closed forms, the stability
condition by shape, what the models show and what is left to POP. All formulas here
are **analysis** in real numbers with a stated rounding of the rules they describe;
the rules themselves are the integer rules of 4 and 5.

### 8.2 The mechanism

Let `N` be the households in a region that draw on one endowment; each draws `h` units
a tick when it works (executions a tick times the quantity drawn), so the demand is
`U = N * h` a tick, or `N * h * TPY` a year. Let `F(S)` be the annual regeneration at
stock `S` (the middle column of 4.2) and `F̂` the **maximum flow** the endowment can
give a year:

| Shape | `F̂` | At |
|-------|-----|----|
| `refill` | `(RateBP / 10^4) * C * mbar` | an empty stock |
| `constant` | `RatePerYear * mbar` | any stock (a finite stock has no steady flow) |
| `logistic` | `(RateBP / 10^4) * C * mbar / 4` | half the capacity |

with `mbar = (m / 10^4) * (mean of the season factor / 10^4)`. The households at which
the demand equals the maximum flow are

```
N_bind = F̂ / (h * TPY)
```

Two regimes follow, according to the abundance curve.

**Regime A: a flat curve (a pure quota).** Below `N_bind` the stock never runs out, the
ration grants every request, and output per household is the full output. Above it the
stock is spent every tick, the draw is the flow, each household is granted
`N_bind / N` of its request, and output per household is `y_full * N_bind / N`: an
elasticity of -1 with respect to the households, a kink at `N_bind`. If the ratio a
household reads is proportional to its output, `r = r_full * N_bind / N` above the
kink, and the fixed point at which births equal deaths (`r = r*`) is

```
N* = N_bind * r_full / r*          (r_full > r*; otherwise the households never outgrow N_bind
                                    and there is no growth fixed point)
```

`r_full` is the ratio a household reads at its full draw and full abundance. It is a
property of the pack's economy (prices, wants, money) and not a field of this spec: the
validator cannot compute it and a run measures it. A `logistic` stock with a flat
curve has no soft regime: above `N_bind` (the maximum sustainable draw) the stock
collapses and the output goes with it (8.3).

**Regime B: a curve.** With an abundance curve, output per household is the full
output times the granted share `f` times the abundance `a(sigma)` at the stock share
`sigma`. In the interior (`f = 1`) the ratio falls smoothly as the stock does, and the
fixed point has a stock share `sigma*` and a population `N*` given by

```
a(sigma*) = 10^4 * r* / r_full             the stock share at which a household reads exactly r*
N*        = F(sigma* * C) / (h * TPY)       the households whose draw equals the regeneration there
```

The stock settles where the regeneration equals the draw, and the population settles
where the stock gives the ratio at which births equal deaths. The two settle together
(8.3). Model 3 confirms regime A to within 0.2% of the predicted `N*` for a refilling
stock (17.3).

### 8.3 Stability

Linearise regime B around `(N*, S*)` with the stock in units and time in years. Let

- `kappa = r* g'(r*) / 10^4` per year be the gain of the population loop with an
  elasticity of 1 (`spec-population.md` 5.4: `g'` is the slope of birth minus death in
  basis points a year per basis point of ratio; `kappa` is 0.06 for the default
  schedules);
- `eps = sigma a'(sigma) / a(sigma)` at `sigma*` be the elasticity of the abundance
  curve (1 for a curve through the origin, 0 for a flat one);
- `F' = dF/dS` at `S*`, and `Gamma = kappa * eps * F(S*) / S*`.

The Jacobian of `(dN/dt, dS/dt) = (N G(S), F(S) - N h TPY)` has the characteristic
equation `lambda^2 - F' lambda + Gamma = 0`: **the equilibrium is stable if and only
if `F' < 0`** (the stock's own regeneration falls as the stock rises), the approach is
monotone if `F'^2 >= 4 Gamma` and an oscillation that decays if `F'^2 < 4 Gamma`.

| Shape | `F'` | `Gamma` | Stable | Monotone iff |
|-------|------|---------|--------|--------------|
| `constant` | 0 | `kappa * eps * F / S*` | **no: neutral**, a cycle whose amplitude is set by the start | never |
| `refill` | `-R` with `R = RateBP * mbar / 10^4` | `kappa * eps * R * (1 - sigma) / sigma` | always | `R >= 4 * kappa * eps * (1 - sigma) / sigma` |
| `logistic` | `-r_s * (2 * sigma - 1)` with `r_s = RateBP * mbar / 10^4` | `kappa * eps * r_s * (1 - sigma)` | iff `sigma > 1/2` | `r_s * (2 * sigma - 1)^2 >= 4 * kappa * eps * (1 - sigma)` |

Four things follow.

1. **A slowly refilling stock overshoots; a quickly refilling one does not.** With the
   default population gain (`kappa` 0.06) and `sigma*` of 0.6 the threshold for a
   refilling stock is `R >= 0.16` a year: a stock that closes less than 16% of its gap
   a year makes the households overshoot `N*` and return in a damped oscillation, and
   the model agrees on both sides of the threshold (undershoot 15.6% of `N*` at 0.02,
   7.0% at 0.05, 1.4% at 0.10, none at 0.16 and above). With the welfare window and a
   lag of a year, the four rates tried (0.16, 0.5, 1, 3) each undershoot by about 4% (17.3); POP's own estimate of the delay is small (5.4), so this is probably the nonlinearity of the 1.6 `N*` start and not a pure delay effect, and the model does not separate them. Both the delay of `spec-population.md` 5.4 and the refill rate apply.
2. **A `logistic` stock is stable only above the maximum-yield point.** If the equilibrium
   stock share `sigma*` is at or below 0.5 the fixed point is unstable (a saddle or a repelling focus) and the model collapses
   from both sides of it (17.3). Above 0.5 it is stable, and its **basin is narrow**:
   the draw at `N*` is 0.96 of the maximum sustainable draw `F_max` for `sigma*` 0.6,
   and a start whose draw exceeds `F_max` for long enough empties the stock before the
   households can respond. Model 4 (17.4) finds the edge at 1.17 times `N*` with the
   welfare window at its default and 1.15 with a slower population. **A logistic stock
   with a refuge recovers; one without does not** (6.1).
3. **A `constant` flow with a population that responds to the stock is neutral.** The
   model gives a sustained cycle of about 10% of `N*` either side from a 10%
   displacement, and a swing from 0.56 to 1.62 times `N*` from a 60% one (17.3). A pack
   that wants a stable fixed point uses `refill`.
4. **A flat curve removes the stock feedback** (`eps` = 0): the loop is POP's alone,
   with an elasticity of 1 above the kink, and `spec-population.md` 5.4 applies with
   `eta` = 1. It is the simplest fixed factor and the sharpest (a kink). A `logistic`
   stock should not have a flat curve.

8.3.1. **What the validator can check.** The condition needs `r*`, `kappa` and `sigma*`.
`r*` and the slope come from the population module's schedules (`spec-population.md`
5.7) when it is on; `sigma*` is not known, because it depends on `r_full`. The
validator therefore evaluates the conditions at the **plan share** of the resource
(`PlanShareBP`, the stock share the pack states as its operating point, 11.1), as
integer inequalities with the checked 128-bit helper, with `eps` the elasticity of the curve at the plan share, `eps_bp = sig * slope / a(sig)` with the slope of the segment that contains `sig` (factor over share, rounded to the nearest basis point), 0 for a flat curve or when `a(sig)` is 0 (the condition is then not evaluated). With `RateBP_eff = RateBP * m / 10^4 * (mean season factor) / 10^4` the annual rate in basis points, `dR` and
`dX` the slope of birth minus death (rate over ratio) at `r*`, `sig` the plan share and
`eps_bp` the elasticity in basis points:

| Warning | Fires when |
|---------|-----------|
| `resource_overshoot`, shape `refill` | `RateBP_eff * 10^4 * dX * sig < 4 * r* * dR * eps_bp * (10^4 - sig)` |
| `resource_overshoot`, shape `logistic` with `sig > 5 000` | `RateBP_eff * (2 * sig - 10^4)^2 * dX < 4 * r* * dR * eps_bp * (10^4 - sig)` |
| `resource_unstable_plan`, shape `logistic` | `sig <= 5 000` and `eps_bp > 0` |
| `resource_unstable_plan`, shape `constant` | `eps_bp > 0` |

Check, for the defaults (`r*` 12 000, `dR` 400, `dX` 8 000) with a refill of 5 000
basis points a year, `sig` 6 000 and a curve through the origin (`eps_bp` 10 000): the
left side is `5 000 * 10^4 * 8 000 * 6 000 = 2.4e15` and the right `4 * 12 000 * 400 *
10 000 * 4 000 = 7.68e14`, so no warning (0.5 a year is above the threshold 0.16). At
1 600 basis points the two sides are equal. The warnings fire only with the population
module on; without it there is no loop to oscillate.

### 8.4 What the models show

Section 17.3 and 17.4 have the runs. In one sentence each, with the caveats of 17: a
refilling stock with a curve through the origin approaches `N*` monotonically above the
threshold and with a damped oscillation below it, as the linearisation says; a logistic
stock above the maximum-yield point is stable inside a narrow basin and collapses
outside it; below the point it collapses from both sides; a constant flow cycles; a
flat curve with a refilling stock converges to the predicted `N*` from above and below
(99 999 and 99 833 of 100 000, one crossing at most); a finite stock does not have a
fixed point at all: the population peaks, the stock is spent, and the households fall
to the floor (17.4).

### 8.5 What is left to POP

1. `spec-population.md` 5.2, row 3 of the table, changes from "RES is not written" to
   "provided by `spec-resources.md` 8"; the sentence "no pack can have a fixed factor
   today" goes (handoff h).
2. The warning `no_fixed_factor` (`spec-population.md` 9.2) is POP's: it should now
   read "no recipe draws on an endowment whose maximum flow can bind (`N_bind` below
   the census's households times a stated margin) and no region has a `HouseholdCap`".
   VIA can report the households at which the draw equals the sustainable draw at the plan stock, `N_plan` (the census's households times `Headroom`, 11.3); `N_bind` of 8.2 uses the maximum flow at an empty stock and is larger for `refill` by 1 over (1 minus the plan share).
3. POP-S1 (`spec-population.md` 14.4; S37 in the register) can run on a `refill`
   resource instead of a test double. The double remains useful to isolate the
   population rules from the resource's own timescale.
4. The warm start's census is not the Malthusian `N*` (`spec-population.md` open
   question 3); with this spec the solver can report `N_bind` and, if the pack states
   `r_full`, `N*`. `r_full` is not a field today; adding one is POP's or VIA's
   decision (open question 6).
5. `r_full` has to exceed `r*` for the population to outgrow `N_bind`; a pack whose
   full-draw ratio is below the replacement ratio shrinks whatever the endowment
   (`spec-population.md` 5.3 covers the related closed ceiling).

## 9. Raw commodities and their sources (note N-021)

9.1. A commodity marked `Raw` "can be supplied without being made from other
commodities" (`spec-goods.md` 6.2). Four things can supply it, and each belongs to
one spec:

| Source | What it is | Spec | Limit |
|--------|-----------|------|-------|
| a recipe with no inputs and no `Draws` | gathering that only labour limits | `spec-production.md` 3.1.2 | labour |
| a recipe with no inputs and `Draws` on an endowment | gathering from a stock | this spec | the endowment's flow or stock |
| an import | goods that arrive from beyond the realm | EXT | its volume and money |
| a money faucet | not goods | `spec-money-supply.md` | not applicable to goods |

The rule for a pack author: *the endowment is for what the realm can make and runs
short of; the import is for what it cannot make at all.* A raw good with both is
produced at home up to the endowment and bought beyond it, which is the arbitrage that
EXT's prices decide.

9.2. **The predicate.** A commodity is **endowment-sourced** if some recipe with no
inputs outputs it as a non-discard item, has at least one `Draws` entry, and there is a
region `x` where (a) the recipe's site factor is positive in some season and (b) for
every draw `(res, d)` the endowment of `res` in `x` can supply at least one execution:
its effective capacity is at least `d`, and its starting stock is positive or its
shape can regrow (`refill` or `logistic` with a positive rate and, for `logistic`, a
positive starting stock or refuge; `constant` with a positive rate or a positive starting
stock). This is the "exists in a region" test of `spec-viability.md` 4.2 with the
endowment conjunct added (handoff h), and the structural pass of `spec-config.md` 7.5,
which is blind to yield and region, keeps counting a `Sited` source with `Draws` as
sourced.

9.3. **How validation sees it.** The marker `Raw` still makes a commodity sourced in
both passes (`spec-viability.md` 4.2.1). The warning `raw_without_source`
(`spec-viability.md` 10.2) becomes: a raw commodity that no recipe with no inputs
outputs, that is not endowment-sourced and that no import or faucet supplies. This spec
adds the warning **`raw_source_exhaustible`** (13.3): a raw commodity whose only sources
are endowments of finite stocks (`constant` with rate 0, or a `refill` or `logistic` stock with `RateBP` 0). Such a chain stands until the
stock is spent. The run-down time at the steady-state flow is the informational
diagnostic `endowment_runway`, reported with the solver's diagnostics (`spec-viability.md` 10.3; 11.3 here).

9.4. **EXT** supplies its own predicate, `import-sourced`, in the same two places;
validation takes the union. This spec does not decide whether a region with no
endowment of a good imports it: that is EXT's data. An endowment can be a node in the
same graph as an outside region (`spec-regions.md` 3.5.2), but the two never share a
field.

## 10. Open access and tenure (the hook for LND)

10.1. In version 1 every endowment is **open access**: any agent in the region whose
role lists a recipe that draws on it, or has one on its effective own-use list (`spec-subsistence.md` 3.6), may draw, subject only to the ration of 5.3. There
is no owner, no price for the use of the stock and no rent. This is the baseline that
tenure is measured against. It has a known consequence, recorded as RES-A5 (19): when
the abundance feeds the net value of a recipe (`spec-agents.md` 7.1.4 reads the
effective yield), more households choose and enter the drawing role as it pays, and the
stock is drawn down by the sum of choices that none of them internalises.

The subsistence floor (`spec-subsistence.md` 9) is such a draw: an own-use recipe draws on an endowment through `Draws` like any recipe, with no priority and no quota over the market recipes, so its requests share the ration of 5.3 with theirs; it is a third path to the same draw-down, beside recipe choice and entry (RES-A5, `spec-stability.md` L90; the floor's own entry is L104). A closed commons makes a household in the own-use mode `Blocked` and counts `own_use_not_available`.

10.2. **The hooks LND will use**, stated so that nothing in this spec has to change:

- a **refusal**: a spec that ties access registers a reason in the ordered list of
  production's refusals (`spec-production.md` 4.2), after `no_endowment`, by the same idea as the
  refusals that other specs register with the relocation (`spec-regions.md` 6.5.2), but in the list of production;
- an **entitlement**: a per-claimant cap on the executions of a request, applied to
  `w_i` before the ration, so that a holder's share is its own and the remainder is
  shared as today; a request carries its recipe, so an entitlement may treat an own-use request differently from a market one (`spec-subsistence.md` 9);
- a **rent**: a recurring levy on the claimant by the money flows of
  `spec-money-supply.md` 4 (a named reason equal to the flow ID), valued at the dual
  of the resource row (11.3), which is the scarcity price of a unit of the stock.

10.3. Until LND exists, nothing in the core reads who has drawn. The ration is a
function of the requests of the tick and keeps no memory of past draws, so a tenure
rule that wants a history keeps its own.

## 11. Viability: the sustainable draw

11.1. **The rule.** The steady-state solver (`spec-viability.md` 3) is a linear
programme of flows per tick. An endowment becomes one row per (resource, region), the
**sustainable draw**, in units a tick:

```
sum over activities (r, x) drawing res:  d(res, r) * x(r, x)   <=   Phi(res, x)
```

with `x(r, x)` the executions a tick of the activity (`spec-viability.md` 3.2) and
`d(res, r)` the quantity drawn per execution. `Phi` is computed in exact rational
arithmetic as the regeneration at the **plan stock** `S_p = PlanShareBP * C / 10^4`
(the share of the capacity a steady state is assumed to hold), over a year, divided by
`TPY`:

| Shape | `Phi` a year |
|-------|--------------|
| `refill` | `(RateBP / 10^4) * (C - S_p) * (m / 10^4) * sbar` |
| `logistic` | `(RateBP / 10^4) * (S_p * (C - S_p) / C) * (m / 10^4) * sbar` (exact, not the rounded-up `w`) |
| `constant` | `RatePerYear * (m / 10^4) * sbar + Stock_0 / HorizonYears` |

where `sbar` is the mean over the year's ticks of the season factor over 10 000
(`spec-calendar.md` 4.5.1: the periodic season of each tick), exact, and `Stock_0` is the
starting stock. **A renewing stock is sustained by its flow; a finite stock is a
reserve drawn down over `HorizonYears`** (module field, default 50, a placeholder): a
pack that must be viable for fewer years than its deposits last is, and one that needs
longer is not. A `constant` stock with a positive rate gets both terms. **The check is optimistic**: it holds the plan stock and so the plan abundance, while the run draws a deposit down. Example B passes at a horizon of 15 years by 1% (313 units a tick against 309 once the abundance at the plan share is counted) while the run of 16.2 exhausts the deposit in 13.8 years. A finite stock should be given a horizon with margin.

11.2. **The abundance in the programme.** A recipe with draws has its mean factor `E`
multiplied by the abundance at the plan share, `a_p = Eval(curve, PlanShareBP)`, the
minimum over its draws (4.7.4), a constant. The programme stays linear. This is the
assumption that the steady state holds the plan stock; the stock a run settles at is
decided by the loop of 8 and may differ.

11.3. **What the solver reports** (handoff h, the edits to `spec-viability.md`):

- `Headroom[res][region]`, the sustainable draw over the draw at the reference
  programme at `lambda = 1`; above 1 the endowment is slack, below 1 it binds and the
  warm start's census is above `N_bind`;
- `Rent[res][region]`, the dual of the row, the value of one more unit of draw per
  tick, 0 when the row is slack. It is a report, the scarcity price that LND's rent
  would charge (10.2);
- the error **`resource_short`**: `lambda* < 1` with a tight resource row, which takes
  the place of `labour_short` when the binding row is a resource (V3), and the
  per-region warning `region_not_self_sufficient` when the cause is a resource;
- the informational **`endowment_runway`**: for a finite stock, `Stock_0` over the draw
  at the optimum, in years, and the warning `raw_source_exhaustible` when a raw good
  depends on it (9.3). Example B in 16.2 gives a runway of 16.9 years against a
  horizon of 50;
- the warning **`endowment_season_peak`**: for an activity whose starts are confined to
  windows (`spec-viability.md` 3.1: the start capacity), the draws made in one window
  exceed `min(C, S_p + the regeneration of that window)`. A seasonal crop whose starts
  are all in the first twenty days of a season draws a year's plots at once; the annual
  flow can be sufficient and the stock still short at the opening. It is an
  approximation (starts are assumed uniform over a window) and a warning, and it is
  register entry RES-A7.

11.4. **The warm start.** The solver's steady state holds the plan stock, and the
default starting stock is `InitialBP` of the capacity (10 000, full). For a renewing
stock a warm-started pack should begin at the plan stock; the author sets `InitialBP`
to `PlanShareBP`, or `spec-viability.md` 8 adds the stock to the warm-start table, one
row per endowment (handoff h; open question 3). A cold start from a full stock is legal
and draws the stock down to its equilibrium over the time constants of 8.3.

## 12. Parameters

All are tunables (architecture.md 9.5), fields of `Config.Modules.Resources`, each with
a default, a bound, an owner and a doc line, and a row in the parameter-owner table
(handoff b). **Every default is a placeholder** (N-015, N-027) or structural; none comes
from a source. The module field `HorizonYears` and the fields of a `Resource` and an
`Endowment` row:

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Module | `HorizonYears` | years a finite stock must last in the viability check (11.1) | 50 | 1 to `MaxHorizonYears` |
| Resource | `ID` | the resource's identity: `[a-z][a-z0-9_]*`, at most 64 bytes, unique among resources | required | `spec-config.md` 3.3 |
| Resource | `Shape` | `constant`, `refill` or `logistic` (4.2) | `constant` | the three names |
| Resource | `RatePerYear` | units a year of a `constant` stock; 0 is a finite stock | 0 | 0 to `MaxRegenRatePerYear`; must be 0 for the other shapes |
| Resource | `RateBP` | annual rate of `refill` and `logistic`, basis points | 0 | 0 to `MaxRegenBP`; must be 0 for `constant` |
| Resource | `Refuge` | units a `logistic` stock regrows from even at 0 | 0 | 0 to the capacity; `logistic` only |
| Resource | `Capacity` | default capacity of an endowment of the resource, units | 0 (none) | 0 to `MaxEndowment` |
| Resource | `InitialBP` | default starting stock as a share of the capacity, basis points | 10 000 | 0 to 10 000 |
| Resource | `PlanShareBP` | stock share a steady state holds (11.1, 8.3.1) | 6 000 | 1 to 10 000 |
| Resource | `Abundance` | knots of (`Share`, `Factor`), basis points (4.7) | none (flat 10 000) | 0 to `MaxKnots` knots; shares strictly increasing, 0 to 10 000; factors 0 to `MaxYieldBP` |
| Resource | `Seasons` | regeneration factor by season, basis points (4.5) | none (10 000 always) | each 0 to `MaxSeasonRegenBP`; a season listed once |
| Resource | `Label` (p) | display name; presentation only, not hashed | empty | `MaxLabelBytes` |
| Endowment | `Region`, `Resource` | the key of the row | required | an existing region and resource |
| Endowment | `Capacity` | replaces the resource's default capacity here | the resource's | 0 to `MaxEndowment` |
| Endowment | `Initial` | replaces the default starting stock here, units | `InitialBP` of the capacity | 0 to the capacity |
| Endowment | `RegenScaleBP` | scales the regeneration here, basis points | 10 000 | 0 to `MaxRegenScaleBP` |

`PlanShareBP` is at least 1 so that the plan stock is positive. The shock slots of 6.3
(a capacity scale and a regeneration scale per endowment) are world state, not fields.

Why these defaults and what they are not. A resource with no data is a **finite
stock of capacity 0**: nothing can be drawn until a pack states a capacity, so an
omission fails safe (a recipe that draws it cannot run, and the warning
`draw_impossible` says so) and not into an unlimited source. A flat curve and a full
start are the neutral choices. A plan share of 6 000 is a placeholder that sits above the
maximum-yield point of a `logistic` stock (5 000, where its loop is on the edge of
instability, 8.3). `HorizonYears` of 50 is a placeholder that a pack with a deposit must
read: example B shows a pack that fails it.

## 13. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9); errors name the
path, for example `modules.resources.resources.woodland.abundance[2].share`.

### 13.1 Bounds

The constants of 4.6, plus `MaxKnots` (8) and `MaxYieldBP` from `spec-config.md` 4.1.

### 13.2 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field outside the bound of 12; more than `MaxResources` resources or `MaxEndowments` rows; a `Capacity`, `Initial` or `Refuge` above `MaxEndowment` |
| `ErrDuplicateID` | two resources with one ID; a resource twice in the `Draws` of one recipe; two rows for one (region, resource); a season twice in a profile |
| `ErrUnknownResource` | a draw of a recipe (`spec-production.md` 3.1) or an endowment row that names a resource that is not defined; **new** |
| `ErrUnknownRegion` | an endowment row that names an undefined region |
| `ErrUnknownSeason` | a season in a regeneration profile that the calendar lacks, including any season entry when the calendar has no seasons |
| `ErrResourceShape` | a shape that is not one of the three; a rate field that the shape does not use set above 0 (`RatePerYear` with `refill` or `logistic`, `RateBP` with `constant`); a `Refuge` on a shape other than `logistic`, or above the effective capacity of any of its endowments; an `Initial` above its capacity; **new** |
| `ErrCurve` | abundance knots whose shares are not strictly increasing or lie outside 0 to 10 000, or whose factors are outside 0 to `MaxYieldBP`; **new** |
| `ErrModuleDependency` | (`spec-production.md` 10.2, unchanged) a recipe with `Draws` while the module is off |
| `ErrOverflow` | the derived checks of `spec-production.md` 10.3 (the largest weight times a draw is at most `MaxQuantity`), and the product of a capacity and the largest shock scale over 10 000 above `MaxEndowment` |

A recipe that names a resource and a region whose row is absent is **not** an error:
the pair has capacity 0 and the recipe cannot run there (`no_endowment`).

### 13.3 Warnings

A warning never fails validation (`spec-config.md` 7.2). Each has a rule name and a
path. The conditions that need the solver are computed by `spec-viability.md` and
reported through it.

| Warning | Condition |
|---------|-----------|
| `resource_unused` | a resource that no recipe draws on |
| `endowment_unused` | an endowment with a capacity in a region where no role with a group lists a recipe that draws on it, on its `Recipes` or on its effective own-use list (`spec-subsistence.md` 3.6) |
| `draw_impossible` | a recipe with a positive site factor in a region whose endowment of a drawn resource has an effective capacity below the draw (so it can never run there) |
| `abundance_not_monotone` | an abundance curve whose factor falls as the share rises (4.7.2) |
| `regen_per_tick_high` | a `refill` or `logistic` effective per-tick fraction, `RateBP * m / (10^4 * TPY)`, above 2 000 basis points |
| `raw_source_exhaustible` | a raw commodity whose only sources are endowments of finite stocks, that is `constant` with rate 0 or `refill` or `logistic` with `RateBP` 0 (9.3) |
| `resource_inert` | a `refill` or `logistic` resource with `RateBP` 0: it never regenerates, so it is a finite stock of the wrong shape |
| `resource_overshoot` | the inequalities of 8.3.1, with the population module on |
| `resource_unstable_plan` | the conditions of 8.3.1, with the population module on |
| `endowment_season_peak` | the condition of 11.3 |
| `no_comparative_advantage` | the cross-ratio condition of 7.4.1 |

Reported by the solver, not by validation proper: `resource_short` (an error, 11.3),
`region_not_self_sufficient` with a resource cause, and the informational
`endowment_runway`.

## 14. Observability

14.1. **Events.** Two new events, both plain data with no map and no pointer, both
emitted after the transition they describe (observability-and-logging.md 4.1, 4.4):

| Event | Emitted when | Fields |
|-------|--------------|--------|
| `EndowmentChanged` | once per endowment per tick, after its regeneration at the end of phase 8, in (`RegionID`, `ResourceID`) order, **if any of `Drawn`, `Regenerated`, `Short` or `Lost` is non-zero**; never otherwise | `Tick`, `Region`, `Resource`, `Stock` (after the tick), `Capacity` (effective), `AbundanceBP` (the snapshot of the tick), `Requested` (units asked, the sum of `w * d`), `Granted` (units the ration granted), `Drawn` (units the starts took, at most `Granted`), `Short` (`Requested - Granted`), `Regenerated`, `Lost` (units removed by a capacity clamp this tick), `AgentsDenied` (requests granted 0), `AgentsReduced` (requests granted less than asked but more than 0) |
| `EndowmentAdjusted` | a shock slot or an adjustment is applied between ticks (6.3), once per changed endowment of a call | `Tick`, `Region`, `Resource`, `Kind` (`adjust` or `shock`), `Delta` (signed, for `adjust`), `CapacityBP` and `RegenBP` (the scales after, for `shock`), `StockBefore`, `StockAfter`, `Lost` |

`Region` and `Resource` are the typed IDs `RegionID` and `ResourceID`; the strings are
in the names table of `spec-observability.md` 3.2. Both events are aggregates bounded
by regions times resources (observability-and-logging.md 4.3), or per changed endowment
of a call. `EndowmentChanged` is the
counter that a halted recipe needs (note N-045): `Short`, `AgentsDenied` and
`AgentsReduced` say that production did not start for want of the stock, by region and
resource, with no per-agent event. An event of a module that is off is not emitted.

14.2. **Logs.** New keys (observability-and-logging.md 3.3.2, handoff g): `resource` (the
resource ID), `stock` (units in an endowment) and `short` (units asked and not granted);
`region`, `tick` and `quantity` are existing. At `Warn`: `endowment exhausted`, with
`tick`, `region`, `resource`, `stock` and `short`, emitted for an endowment in a tick in
which `Short` is above 0 and its `ExhaustedRun` is 1 or a multiple of 10, so the first
cut and then every tenth consecutive tick (bounded by regions times resources, as
observability-and-logging.md 3.2.2 requires). At `Debug`, guarded by `Enabled`, per
endowment per tick in which the ration had to cut: `draw rationed` with `tick`,
`region`, `resource`, `quantity` (requested), `short`. No `Info`, no `Error`. A
validation warning is not a log: it is returned (`spec-config.md` 7.2).

14.3. **Metrics** (proposed; observability-and-logging.md 5.3, handoff g). Two new metric
attributes, `economysim.resource` and `economysim.region`, both bounded by the
configuration (5.4.1):

| Name | Instrument | Unit | Attributes | Fed by |
|------|------------|------|------------|--------|
| `economysim.endowment.drawn` | Counter | `{unit}` | `economysim.resource`, `economysim.region` | `EndowmentChanged.Drawn` |
| `economysim.endowment.regenerated` | Counter | `{unit}` | same | `EndowmentChanged.Regenerated` |
| `economysim.endowment.short` | Counter | `{unit}` | same | `EndowmentChanged.Short` |
| `economysim.endowment.stock` | Gauge | `{unit}` | same | `EndowmentChanged.Stock` and `EndowmentAdjusted.StockAfter`; the adapter keeps the last value otherwise |

The series added for the five-region proposal of 18 are 5 regions times 3 resources,
that is 15 per instrument and 60 in all.

14.4. **Snapshot.** Per endowment in (`RegionID`, `ResourceID`) order: `Stock`,
`Capacity` (effective), `AbundanceBP`, `ExhaustedRun`, and the two shock scales when
non-default. With the population module on, OBS may add the measures of 8 (`N_bind`
where VIA has computed it). The field names are OBS's (note N-094).

14.5. **Cost with no observer.** Event construction is guarded by the world's "has
observer" flag and allocates nothing otherwise; the ration works in scratch slices
held by the world (observability-and-logging.md 4.5, 8.2; NFR-RES-001).

## 15. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This module's, all plain data,
sorted, no maps:

- for each endowment with a configured capacity above 0 (so a depleted stock is saved as 0 and not restored as the starting stock), in (`RegionID`, `ResourceID`) order: `Stock`, `Carry`, `ExhaustedRun`;
- the shock scales that are not 10 000, as (region, resource, `CapacityBP`, `RegenBP`) in
  the same order.

**Not saved**, because derived: the effective capacity and scale (a function of the
configuration and the shock), the abundance snapshot and the grants of the tick (recomputed
at the start of the production phase), the season factor (a function of the tick), the
dense index arrays, the key of the ration (a function of the tick and the agent) and the
sustainable-draw rows of the solver. A world restored between ticks regenerates, rations
and reports exactly as an uninterrupted one (FR-RES-042). The configuration hash covers
the resources and their rows; it does not cover the shocks, which are state.

## 16. Worked examples

Two settings of the author's invention, flagged as examples (N-017). The numbers are
placeholders; the arithmetic is checked in `models/res/worked.py` (17.5). Neither claims
to describe a real land, forest or mine.

### 16.1 Example A: a farming realm (an example)

The Earth calendar (365 ticks a year, one tick a day, seasons `spring` 92, `summer` 92,
`autumn` 91, `winter` 90 days, `spec-calendar.md` 7.4) and the bread chain of
`spec-production.md` 12.1, whose three sited sources gain a limit. Excerpt, not a
complete valid pack:

```toml
[resources]
horizon_years = 50

# Land that recovers in the fallow half of the year: a refill of 100% of the gap a year,
# paid at twice the rate in autumn and winter and not at all in spring and summer.
[resources.resource.arable]
shape = "refill"
rate_bp = 10000
initial_bp = 8500
seasons = [ { season = "autumn", bp = 20000 }, { season = "winter", bp = 20000 } ]

# A stand that grows from itself: 30% a year, a refuge of 20 units, and a poorer yield
# when it is thin.
[resources.resource.woodland]
shape = "logistic"
rate_bp = 3000
refuge = 20
abundance = [ { share = 0, factor = 0 }, { share = 2000, factor = 6000 },
              { share = 5000, factor = 9000 }, { share = 10000, factor = 10000 } ]

# A seam that does not renew: rate_per_year is 0, so it is finite and its depletion is
# permanent. A poorer grade as it empties.
[resources.resource.iron_seam]
abundance = [ { share = 0, factor = 4000 }, { share = 10000, factor = 10000 } ]

[resources.endowment.north.arable]
capacity = 2000
[resources.endowment.south.arable]
capacity = 1600
regen_scale_bp = 12000          # richer soil recovers a fifth faster
[resources.endowment.north.woodland]
capacity = 50000
[resources.endowment.north.iron_seam]
capacity = 400000
[resources.endowment.far_south.iron_seam]
capacity = 900000
```

and in the recipes of `spec-production.md` 12.1, one line each:

```toml
[recipe.grow_grain]   # a plot-use per job
draws = [ { resource = "arable", quantity = 1 } ]
[recipe.cut_wood]
draws = [ { resource = "woodland", quantity = 12 } ]
[recipe.mine_iron]
draws = [ { resource = "iron_seam", quantity = 18 } ]
```

**Arithmetic of a tick.** In `north`, `arable` has a capacity of 2 000 and a starting
stock of `8 500 * 2 000 / 10 000 = 1 700`. Suppose 380 growers start a job in the first
20 ticks of spring, 19 a tick: the ration grants every request (19 is below the stock),
the stock falls to 1 320, and it stays there through the summer (no regeneration). In
the first autumn tick the numerator is `10 000 * (2 000 - 1 320) * 10 000 * 20 000 =
1.36e15` over `D = 10^12 * 365 = 3.65e14`: 3 units are added and the carry becomes
`1.36e15 - 3 * 3.65e14 = 2.65e14`. In the second the stock is 1 323, the numerator
`1.354e15`, the total `1.619e15`: 4 units are added and the carry is `1.59e14`. The stock
at the start of each spring is 1 700, 1 748, 1 766, 1 773 in the first four years,
settling at 1 777; a 380-plot spring takes 19% of the capacity and the fallow
half returns it (the table is `worked.out`, A1). The abundance of `woodland` at a stock of
41 609 of 50 000 is the share `41 609 * 10 000 / 50 000 = 8 321` and the factor
`9 000 + trunc((10 000 - 9 000) * (8 321 - 5 000) / 5 000) = 9 664`.

**Three kinds of depletion** (A2 in `worked.out`). `woodland`'s maximum sustainable
draw is `0.3 * 50 000 / 4 = 3 750` units a year. A steady draw of 3 000 holds the stock
toward 36 180 (share 0.72; the table of `worked.out` shows 36 268 at year 35); of 3 600 (0.96 of the maximum) toward 30 000
(share 0.60; 30 900 at year 35); of 4 500, above it, the stock falls to 11 963 after thirty years and is 0
at thirty-five, where it stays: with a refuge of 20 the regrowth at 0 is
`0.3 * 20 = 6` units a year against a draw of 4 500. The iron seam never regrows: at
18 units an execution its 400 000 units are 22 000 executions, and the poorer grade as it
empties (a factor of 4 000 at the last ton) is a rule of the data and not of the code. All
three are the same format; no code names iron, wood or land (architecture.md 9.1).

### 16.2 Example B: a habitat (an example)

A calendar of 320 days at two ticks a day (`TicksPerYear` 640) with two seasons, `lit`
120 days (240 ticks) and `dark` 200 days (400 ticks); the colony of `spec-production.md`
12.2. 1 000 households in `crater_rim`, whose `cut_ice` yields 6 000 (`spec-production.md`
12.2).

```toml
[resources]
horizon_years = 15              # a pack that must last fifteen years, not fifty

# A deposit with no renewal: finite, permanent depletion.
[resources.resource.ice_field]
abundance = [ { share = 0, factor = 2000 }, { share = 2000, factor = 8000 },
              { share = 10000, factor = 10000 } ]

# Culture vats: used up and cleaned, a refill of 200% of the gap a year.
[resources.resource.vat_volume]
shape = "refill"
rate_bp = 20000
plan_share_bp = 5000

[resources.endowment.crater_rim.ice_field]
capacity = 3000000
[resources.endowment.deep_crater.ice_field]
capacity = 12000000
[resources.endowment.crater_rim.vat_volume]
capacity = 30000
```

with `cut_ice` drawing 30 `ice_field` and `grow_algae` drawing 1 `vat_volume`.

**The runway.** The households need 1 water a tick each and a `melt` turns 1 ice into 6
water, so 166.7 ice a tick; a `cut_ice` execution makes `30 * 6 000 / 10 000 = 18` ice, so
9.26 executions a tick and 278 units drawn a tick, 177 778 a year. A deposit of 3 000 000
lasts 16.9 years at a flat yield and **13.8 years** with the curve above, because the
yield per execution falls as the deposit empties (80% left after 3.3 years, 50% after 7.9,
20% after 12.1, 5% after 13.6). The viability row (11.1) for `crater_rim`: the sustainable
draw is `Stock_0 / HorizonYears`; with the default horizon of 50 years it is 60 000 a year,
94 a tick, against 278 needed: the pack fails with `resource_short`. With `horizon_years =
15` it is 200 000 a year, 313 a tick, and passes; with 10, 469. The warning
`raw_source_exhaustible` fires if `ice` is marked `Raw` and `cut_ice` is its only source.
The same data in another setting would be a well, a quarry or a stock of fuel.

**The fixed factor.** `grow_algae` yields 120 biomass at a site factor of 10 000 and its
mean over the year is `(240 * 10 000 + 400 * 2 500) / 640 = 5 312.5`, so 63.75 biomass an
execution and, for 2 biomass a tick each, 31.4 executions a tick (20 078 a year) for the
1 000 households. With the vats (a capacity of 30 000, refill 20 000 basis points) the
maximum flow is `2.0 * 30 000 = 60 000` a year at an empty stock and 30 000 at the plan
share: headroom 2.99 and 1.49, and the vats bind at `N_bind` of 2 988 households (1 494 at
the plan share). With the population module on, a habitat of 1 000 households grows until
the vats ration it, and the ratio then falls as `N_bind / N` (8.2): this is the fixed
factor that `spec-population.md` 5.2 lacks. It is in the same format as the farm's land,
with different words.

## 17. What was run

Five small Docker Python scripts in `models/res/` (`docker run --rm -e
PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/res":/w -w /w python:3.12-slim python
<script>`). They are **models of the rules of this spec, not of the library**: the
regeneration, the ration, the abundance curve and the population rules are the integer
rules of the specs; the economy is replaced by one assumed function (the ratio a household
reads is proportional to the share of its draw granted and the abundance). Floating point
appears in the printouts and the predictions only. The numbers are placeholders and show no
default to be stable (N-027).

### 17.1 Regeneration (`regen.py`)

The units added over a run equal `floor((carry0 + sum of numerators) / D)` and the carry
equals the remainder, on 400 random runs of the three shapes with tick counts from 1 to
100 000 a year: **no violation**. A logistic stock of 1, 2, 5, 20 and 100 units at 5% a year
on a capacity of a million: without a carry the stock is stuck at 1, 2, 5, 20, 100 after
twenty years; with the carry and the round-up factor it is 2, 4, 12, 53, 270 (the
exponential at 5% for twenty years is 2.7). The largest numerator is 10^30 and the largest
denominator 10^17, inside 128 bits and `int64`.

### 17.2 The ration (`allocate.py`)

1 000 agents of weight 1, one execution each wanted every tick for a year, and a supply of
a given share of the demand:

| Supply | Rule | Least share of its demand one agent got | Most | Longest run of ticks denied |
|-------:|------|---:|---:|---:|
| 0.3 | first come, in the rotation order | 0.000 | 0.822 | 365 |
| 0.3 | proportional share, hash key (the spec) | 0.236 | 0.384 | 40 |
| 0.5 | first come, in the rotation order | 0.000 | 1.000 | 365 |
| 0.5 | proportional share, hash key | 0.416 | 0.581 | 20 |
| 0.9 | first come, in the rotation order | 0.726 | 1.000 | 100 |
| 0.9 | proportional share, hash key | 0.841 | 0.945 | 5 |

Proportional share with the rotation as tie-break gives the same as first come for agents
of weight 1. Twenty cohorts of weight 50 get exactly 0.500 per household at half supply;
1 000 agents of weight 1 get 0.416 to 0.581 over the year (sampling), the same mean. A
mixed case of draws of 5, 2 and 1 units against 14 available takes exactly 14. What it
does not show: one-tick jobs only (a multi-tick job keeps its holder out of the ration for
its duration), no strategy reacting to a denial, no money.

### 17.3 The loop with a stock (`coupled.py`)

The Malthusian rules of `spec-population.md` with its default schedules (`kappa` 0.06, `r*`
12 000), `N*` of 100 000 by construction, the ratio `r = 20 000 * f * a`.

| Case | Prediction | Run |
|------|-----------|-----|
| `refill`, curve through the origin, window 1, start 1.6 `N*`, rate 0.02, 0.05, 0.10, 0.16, 0.30, 0.5, 1, 3 a year | oscillates below 0.16 a year | undershoot 15.6%, 7.0%, 1.4% of `N*`; none from 0.16 up |
| the same with a window of 365 and a lag of 365 | delay dominates (`spec-population.md` 5.4) | undershoot 4.6%, 4.4%, 4.3%, 3.8% for all rates |
| `logistic`, `sigma*` 0.6, r_s 1, starts 1.02 to 1.6 `N*` | stable in a narrow basin | recovers up to 1.1; collapses at 1.6 (the draw is then 1.54 `F_max`) |
| `logistic`, `sigma*` 0.4, start 0.98 and 1.02 `N*` | unstable | collapses from both |
| `constant`, curve through the origin, starts 0.9, 1.1, 1.6 | neutral | sustained cycles: +-10%, +-10%, 0.56 to 1.62 `N*` |
| flat curve, `refill`, rates 0.3 and 3, start 1.6 and 0.6 `N*` | `N* = N_bind * r_full / r*` = 100 000 | 99 999 and 99 833 |

### 17.4 Recovery and the basin (`recovery.py`)

A blight at year 100 sets the stock to 5% of the capacity; cells are `N / N*` and `S / C`:

| Case | year 110 | year 150 | year 250 | year 400 | year 599 |
|------|---------:|---------:|---------:|---------:|---------:|
| `refill` 0.5 a year | 0.94 / 0.62 | 0.99 / 0.61 | 1.00 / 0.60 | 1.00 / 0.60 | 1.00 / 0.60 |
| `logistic`, no refuge | 0.82 / 0.00 | 0.37 / 0.00 | 0.05 / 0.00 | 0.00 / 0.00 | 0.00 / 0.00 |
| `logistic`, refuge 1% of the capacity | 0.82 / 0.00 | 0.37 / 0.00 | 0.05 / 0.00 | 0.35 / 0.91 | 1.00 / 0.60 |
| `constant` | 0.82 / 0.08 | 0.46 / 0.70 | 1.01 / 0.20 | 0.76 / 0.26 | 1.01 / 1.00 |

A finite deposit of 10^9 units with a start at 0.5 `N*`: the population peaks at 0.66 `N*`
in year 20, the stock is spent by year 50, and the households are 0.14 `N*` at year 100 and
0.00 at year 399. The basin of the logistic stock of 17.3, found by bisection: the largest
start that recovers is 1.17 `N*` (1.15 with the window and the lag of 365).

### 17.5 The worked examples (`worked.py`)

The integer arithmetic of 16, of the ration of 5.3.7 and of the sizing of 18. Every number
quoted in those sections is in `worked.out`.

### 17.6 What the models do not show

No prices, no money, no market, no failure ladder, no pool, no spawner: the economy is
one function, the population is large enough that whole households do not matter, the
signal is a boxcar window, and a single draw and a single role. They do not show that any
pack of the library has these properties, and the thresholds belong to the placeholder
schedules. They show that the integer rules agree with the linearisation they were
derived from.

## 18. The five-region pack and what REF needs

18.1. **What the base chain draws on.** `spec-reference-chain.md` 4.2 has three sited
sources with no limit: `cut_fuel`, `dig_mineral`, `gather_seed` (and the line gatherers
`gather_a` to `gather_g`). With this module REF can give some of them a limit, and should
leave the rest unlimited, since an unlimited source is legal and an overlay is the
mechanism for variants (`spec-reference-chain.md` 4.5). The proposal for a new overlay,
`overlay/resources.toml`:

| Recipe | Draws | Resource and shape | Reason |
|--------|-------|--------------------|--------|
| `grow_staple` | 1 per execution | `arable`, `refill` 10 000 basis points a year, `plan_share_bp` 5 000 (the sizing of 18.2 assumes it; the default is 6 000) | the land of a staple; the fixed factor of the population scenarios (S37) |
| `dig_mineral` | 3 per execution | `deposit`, `constant` with rate 0 | a finite stock; the runway and the cliff (RES-S3) |
| `cut_fuel` | 14 per execution | `stand`, `logistic` 3 000 basis points a year, refuge | the maximum sustainable draw and the tipping point (RES-S2) |
| `gather_seed`, `gather_a` to `gather_g` | none | unlimited | not changed: wild gathering stays labour-limited |

The names are neutral and are REF's to choose.

18.2. **Capacity from the solver, not by hand.** The capacity of an endowment follows
from the draw of the reference programme (`spec-viability.md` 3.2) and a chosen
**load**: `Capacity = draw a year / (load * (RateBP / 10^4) * (1 - share))` (with `share` 0.5 and a rate of 1 below) for a `refill` stock at the
plan share `share` (so that its sustainable draw is the draw divided by the load). The proposal uses
two data sets: a **slack** one with a sustainable flow of 2.2 times the draw at
`lambda = 1`, so that no autarky `lambda*` of `spec-reference-chain.md` 4.4.4 (1.64 to
1.92) is lowered by a resource row, and a **binding** one at 1.1 times, so that the
fixed factor acts and the population scenarios can show the approach to `N*`. The figures
below are hand-derived from the census of `five`, the mean site factor of each region's
`grow_staple` (a mean over its start seasons) and an upkeep of 2.25 `staple` a household a
tick (2 and 0.25 through `prepare`), net of the 40 seed an execution returns through
`save_seed`; the solver's exact flows replace them (`worked.out`, D). They are an order of
magnitude, not a calibration:

| Region | Households | Mean site factor | Executions a tick | A year | Capacity, slack (2.2x) | Capacity, binding (1.1x) |
|--------|-----------:|-----------------:|------------------:|-------:|-----------------------:|-------------------------:|
| `north` | 1 200 | 10 000 | 1.53 | 560 | 2 464 | 1 232 |
| `east` | 1 000 | 9 500 | 1.35 | 491 | 2 162 | 1 081 |
| `south` | 1 000 | 10 000 | 1.28 | 467 | 2 054 | 1 027 |
| `west` | 800 | 7 000 | 1.46 | 533 | 2 347 | 1 174 |
| `centre` | 1 000 | 8 000 | 1.60 | 583 | 2 567 | 1 284 |

Richness differs by region through the `RegenScaleBP` and the capacity (the table above is
the capacity at a scale of 10 000); a pack that wants a fertile `east` and a thin `west`
raises one and lowers the other and re-runs the solver. The seasonal profile of the
yield (`grow_staple` in spring) puts a year's starts into a window: the `arable` stock
must hold a spring's draw, which the capacities above do (a capacity is 4.4 times the
spring's executions in the slack set and 2.2 times in the binding set), and
`endowment_season_peak` checks it.

18.3. **What the scenarios need** (N-092): the overlay as above; `Renewal` 0 and the
demand overlay on for the population scenarios (`spec-population.md` 14.4); the
population module's schedules; for RES-S2 and RES-S3 a one-region pack (`one`) with the
`stand` and the `deposit`. REF's warm start applies the plan stock by `InitialBP =
PlanShareBP` or by the warm-start row of 11.4.

## 19. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9; fields as its section 3. The
register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn` numbers (the next free are
L87, D53 and S41 at the time of writing). The entries here are `RES-An`, the dampers
`RES-Dn` and the scenarios `RES-Sn`, so that specs written at once do not collide.
**Status is `model` where section 17 supports it and `hand` otherwise; nothing here claims
stability.**

### 19.1 Dampers this spec adds or specifies

| ID | Damper | Home | Status |
|----|--------|------|--------|
| RES-D1 | Proportional ration with a stateless key: a scarce stock is shared in proportion to what is asked, weight-invariant and fair over runs, not first come. Cost: a household can be cut by a small share every tick | RES 5.3 | specified (counted by `Short`, `AgentsDenied`, `AgentsReduced`; model 2) |
| RES-D2 | Abundance curve: output falls as the stock does, before it runs out, so the draw's feedback on the ratio is smooth and the loop of 8 has the soft regime. Cost: yield lost before the stock is spent | RES 4.7 | specified (data; default flat) |
| RES-D3 | The clamp at capacity and the carry: a full stock banks nothing, and a slow regeneration is not truncated to zero, so no state is absorbing by arithmetic. Rule constants | RES 4.3, 4.4 | specified |
| RES-D4 | Refuge: a logistic stock of 0 regrows, so a collapse can end. Cost: a stock that never fully dies | RES 4.3.3, 6.1 | specified (data; default 0, which is permanent) |
| RES-D5 | The tick snapshot: the stock and abundance every agent sees and uses are those of the start of the production phase, so a result does not depend on the order of `Work` or of the starts. A rule constant | RES 5.2 | specified |
| RES-D6 | The resource row of the viability check: a pack whose census draws more than the sustainable draw fails `resource_short`, and the warnings of 8.3.1 flag an overshoot or an unstable plan. Cost: a pack that would be viable on a longer horizon is rejected | RES 11, 8.3.1 | specified (the check is VIA's solver; the conditions are RES's) |

### 19.2 Loops, hazards and amplifiers

**RES-A1. The resource-population loop.** *Kind:* loop. *Loop:* households (+) draw on the
stock (-) the stock (+) the abundance and the granted share (+) the ratio (+) net growth
`birth - death` (+) households. One negative link, so polarity B, with the regeneration as
the second state. *Delay:* the regeneration timescale `1 / R` (a year for a refill of
10 000 basis points), the welfare window `T` and the economy's lag (POP-A1). *Owner:* RES
8; the other half is POP-A1 (L75), which this loop closes (L80 no longer applies to a
pack with an endowment that binds). *Damper:* RES-D2, D6; POP's D46 to D52. *Undamped:* a
`constant` flow, which is neutral (8.3). *Condition:* `F' < 0` and, for a monotone
approach, `F'^2 >= 4 Gamma` (8.3 table); the delay condition of POP 5.4 on top. *Test:*
RES-S1. *Status:* model (17.3).

**RES-A2. Tipping below the maximum sustainable draw.** *Kind:* loop. *Loop:* a draw above
`F_max` (+) falls in the stock (-) the regrowth `F(S)` at a stock below half the capacity
(+) falls in the stock. Two negative links make it reinforcing (R): a logistic stock below
half the capacity regrows less the less it has. *Delay:* the time to spend the stock at
the excess draw. *Owner:* RES 4.2.1, 8.3. *Damper:* RES-D4 (a refuge ends it), D6 (a plan
share above one half), a population that responds before the stock empties (basin 1.17
`N*`). *Undamped:* a logistic stock with no refuge at 0 is extinct for good. *Condition:*
the draw stays below `F_max` and the plan share above 5 000. *Test:* RES-S2. *Status:*
model (17.3, 17.4).

**RES-A3. A finite stock has no fixed point.** *Kind:* hazard. A stock with no
regeneration is spent: the abundance falls, the ratio falls, the households fall, and the
chain that depends on the stock (a raw good with that as its only source, 9.3) stops. The
`cliff` is a flat curve; a curve makes it a slope. *Owner:* RES 6.1, 9.3, 11. *Damper:*
RES-D2 (a slope), D6 (`HorizonYears`, `raw_source_exhaustible`), the failure ladder and
exit for the households. *Undamped:* by design. *Condition:* the runway exceeds the
horizon the pack needs. *Test:* RES-S3. *Status:* model (17.4).

**RES-A4. Contested endowments.** *Kind:* hazard (the entry STB-L29 of `spec-production.md`
16, made specific). A scarce stock is shared by a rule; the rotation gave fairness over
long runs only, and a household could be denied for a year. *Owner:* RES 5.3; PRO 5.5.
*Damper:* RES-D1, D5. *Undamped:* a household that is cut a little every tick and has no
other recipe goes unfed slowly. *Condition:* the cut is spread in proportion. *Test:*
RES-S4. *Status:* model (17.2). This entry merges into L29.

**RES-A5. Open-access draw-down through recipe choice and entry.** *Kind:* loop. *Loop:*
draws (+) a lower stock (-) a lower abundance (-) output per execution (-) supply of the
commodity, so a higher price when demand is inelastic (+) the net value of the drawing
recipe (`spec-agents.md` 7.1.4) when the price rises more than the yield falls (+) agents
choosing and entering the drawing role (`spec-spawner.md` 6) (+) draws. The sign of the
price link is positive for an inelastic demand (a staple, `spec-agents.md` 11.6): the loop
is then **reinforcing (R)** and a tragedy of open access; for an elastic demand the net
value falls and it is balancing (B). *Delay:* the belief step, the review interval and the
entry share. *Owner:* RES 10; AGT 7.1.4; SPN 6. *Damper:* SPN's entry limit (D14) and
cooldown (D34), the ration (RES-D1) and the plan row (RES-D6) as warnings; tenure is LND's.
*Undamped:* the incentive itself. *Condition:* demand elasticity above 1, or a stock whose
regeneration exceeds the entry rate. *Test:* RES-S2 with the spawner on. *Status:* hand.

**RES-A6. Scarcity through the cost anchor.** *Kind:* loop. *Loop:* a lower stock (-) a
lower abundance (-) a lower effective yield `E` (+) a higher `UnitCost` (`spec-production.md`
9.1 divides by `E`) (+) the belief cost band, floor and ceiling (`spec-beliefs.md` 3.2.2)
(+) the asking price. This is how scarcity reaches the price, the point of putting the
abundance in `E`. The hazard is that a floor above what buyers can pay freezes the market
(L09, N-039), more likely the thinner the stock. *Owner:* RES 4.7.5; PRO 9; BEL 3.2.2.
*Damper:* the cost floor of 7 000 and the ceiling (D04), exit. *Undamped:* a market frozen
by a cost floor. *Condition:* `CostFloor` times the cost at the thinnest stock stays below
the buyers' limit; an abundance of 0 makes `E` 0 and the cost unknown, which lifts the floor.
*Test:* RES-S3 with perishables. *Status:* hand.

**RES-A7. The opening of a season against the stock.** *Kind:* hazard. A crop whose starts
are confined to the first days of a season draws a year's units at once, while the stock
was filled by a flow spread over the year, so the annual flow can be sufficient and the stock
short at the opening; the ration then cuts the late starters. The start is a lockstep (L22)
and the endowment a bound on it. *Owner:* RES 11.3; PRO 12.1; STB L22. *Damper:* RES-D1 (the
cut is shared), the capacity sized to a season's draw (18.2), `endowment_season_peak`.
*Undamped:* a pack that ignores the warning. *Condition:* the capacity at least a window's
draws. *Test:* RES-S4. *Status:* hand (the arithmetic of 16.1).

### 19.3 Delays near neighbours (checklist 9.6)

The regeneration timescale of a refilling stock (a year at 10 000 basis points) is near the
season cycle and the welfare window of POP (which sets the population's delay): the loop of
RES-A1 has both. A `refill` of 16% a year has a time constant of six years against about 17 for the population (`spec-population.md` 5.4 gives 33 years at an elasticity of 0.5; it is half that at 1): the stock is faster than the population in the default
placeholders, which is what makes the approach monotone. A shorter window makes the stock
slower in comparison and an overshoot more likely.

### 19.4 Scenarios proposed

RES-S1 to RES-S5 are `spec-reference-chain.md`'s S41 to S45, in the same order; their
bands are in `spec-stylised-facts.md` 5.11 (S43 within `s43.runway_bp`, 10%; S44 within
`s44.grant_band_bp` and `s44.denied_ticks`).

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| RES-S1 | A region with a `refill` resource and the population module on, started at 0.6 and 1.6 times the households at which the stock binds, over rates above and below the threshold of 8.3 | the population approaches `N*` within a band; for a rate above the threshold without crossing it more than once, below it with a damped oscillation of the size the table predicts; `EndowmentChanged.Short` is non-zero while it binds | A1 |
| RES-S2 | A `logistic` resource with a draw below and above the maximum sustainable draw, with and without a refuge, the spawner on | the stock holds below the maximum and collapses above it; without a refuge it stays 0 and the `Warn` fires bounded; with a refuge it recovers; the recipe reports `no_endowment` and the role fails by the ladder | A2, A5 |
| RES-S3 | A finite deposit that is the only source of a raw good | the run-down time agrees with `endowment_runway` within 10%; the dependent chain stops at exhaustion; the F-regime of collapse is reported | A3, A6 |
| RES-S4 | A scarce `refill` resource, a thousand agents of weight 1 and 20 cohorts of weight 50, a crop whose starts are in a window | the grant per household is within the band of the ration; no agent is denied for more than a stated run; the cohorts and the units get the same mean; the Warn bound holds | A4, A7 |
| RES-S5 | A blight (stock to 5%) and a drought (regeneration scale 3 000) on each shape | a `refill` stock recovers, a `logistic` one with no refuge does not, a `constant` one as its rate allows; the identity of 6.4 holds in every tick | A1, A2 |

## 20. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-RES-001 | The module MUST be on if and only if `Config.Modules.Resources` is non-nil; with it off no endowment, event, log or state of this spec MUST exist, a recipe with `Draws` MUST be rejected with `ErrModuleDependency`, and the yield table MUST work unchanged (2, 3.3). |
| FR-RES-002 | A resource MUST be a plain struct with an ID and a shape from the closed set `constant`, `refill`, `logistic`; no code MUST branch on a resource ID; resource IDs and commodity IDs MUST be separate namespaces; and no resource MUST appear in an account, a lot, a price, a tradable set or any recipe list other than `Draws` (3.1). |
| FR-RES-003 | Resources MUST be ordered by ID string bytewise to give `ResourceID`, and every iteration over endowments, in state, events, the snapshot and the checkpoint, MUST be in ascending `RegionID` then `ResourceID` (3.2.3). |
| FR-RES-004 | `Endowments` MUST be a region-keyed table of the mechanism of `spec-regions.md` 5 with one row per (region, resource); `Capacity` and `Initial` MUST be replace and `RegenScaleBP` scale; absence MUST be the default without a warning; an unknown region or resource MUST be rejected with `ErrUnknownRegion` or `ErrUnknownResource` and a second row with `ErrDuplicateID` (3.3.1). |
| FR-RES-005 | The stock of an endowment MUST be an integer from 0 to its effective capacity, `trunc(capacity * capacity scale / 10 000)`, at every observable point, and a pair with no capacity MUST have a stock of 0 (3.2, 3.4). |
| FR-RES-006 | The starting stock MUST be the row's `Initial`, else `trunc(Capacity * InitialBP / 10 000)`, and MUST NOT exceed the capacity (3.3, 13.2). |
| FR-RES-007 | Regeneration MUST run once per endowment per tick as the last step of phase 8 in ascending (`RegionID`, `ResourceID`) order, after the tick's draws, using the season of that tick (4.1). |
| FR-RES-008 | The numerator of a `constant` stock MUST be `RatePerYear * m * s * 10^4`, of a `refill` stock `RateBP * (C - S) * m * s` (0 when `S >= C`), and of a `logistic` stock `RateBP * w * m * s`, over `D = 10^12 * TicksPerYear`, with `m` the effective scale and `s` the season factor, in checked 128-bit arithmetic (4.2). |
| FR-RES-009 | For a `logistic` stock `w` MUST be `ceil(S' * (C - S') / C)` for `0 < S' < C` and 0 otherwise, with `S' = max(S, min(Refuge, C - 1))`, so that a stock of at least one unit below the capacity regrows (4.3.3). |
| FR-RES-010 | Regeneration MUST add `(Carry + n) / D` units and keep `(Carry + n) mod D` as the carry, and over any run in which the capacity is not reached the units added MUST equal `floor((Carry_start + sum of the numerators) / D)` whatever path the stock took (4.3.1). |
| FR-RES-011 | When the stock would reach or exceed the effective capacity it MUST become the capacity and the carry MUST become 0, and a capacity of 0 MUST give a stock and a carry of 0 (4.4). |
| FR-RES-012 | A regeneration profile MUST be complete: with a profile an unlisted season MUST have the factor 0, with none the factor MUST be 10 000 in every season, and a season entry when the calendar has no seasons MUST be rejected with `ErrUnknownSeason` (4.5.1). |
| FR-RES-013 | The abundance MUST be the curve evaluated at `floor(Stock * 10 000 / C)` by integer interpolation with truncation toward zero, flat outside the first and last knots, 10 000 when there are no knots, and 0 when `C` is 0 (4.7.1). |
| FR-RES-014 | A recipe with several draws MUST take the least of the abundance factors of its resources, and the abundance MUST be a factor of the effective yield of `spec-production.md` 6.3, read when the job starts, so that pending output is fixed then (4.7.3, 4.7.4). |
| FR-RES-015 | The abundance and the stock every agent sees and uses in a tick MUST be those of a snapshot taken at the start of the production phase, and the results of `Work` MUST NOT depend on the order of the calls or of the starts (5.2, 4.7.3). |
| FR-RES-016 | `Endowment(region, resource)` MUST return the stock, and 0 for an unknown region or resource or a pair with no endowment; the take MUST be all or nothing and MUST fail with `ErrInsufficientEndowment` when it exceeds the stock; and the take MUST NOT be exported (5.1). |
| FR-RES-017 | The ration MUST grant every request when the units asked do not exceed the stock; otherwise it MUST grant `floor(w_i * A / U)` executions to each request, then give one more execution at a time, to agents ordered by descending remainder, then ascending key, then ascending agent ID, to those whose draw fits what is left, in one pass (5.3.2). |
| FR-RES-018 | The start of a job MUST take every draw or none, and a start that cannot MUST not begin and MUST be refused with `no_endowment` (5.4.1). |
| FR-RES-019 | The key of the ration MUST be the SplitMix64 finaliser of `tick * 0x9E3779B97F4A7C15 + agent` as stated in 5.3.3, MUST keep no state and MUST NOT draw from the world's random source (5.3.3). |
| FR-RES-020 | The grant MUST never exceed the request, the units granted MUST never exceed the stock, and the result MUST NOT depend on the order in which the requests were listed (5.3.2). |
| FR-RES-021 | A cohort of weight `w` and `w` agents of weight 1 with the same total request MUST receive total grants that differ by less than the number of agents, and each request MUST receive at least `floor(w * A / U)` executions (5.3.5). |
| FR-RES-022 | The executions started MUST be the least of the executions the agent could start ignoring draws and the grant of each of its draws; when that is 0 for lack of a grant the refusal MUST be `no_endowment`, after `zero_yield`, `no_site`, `no_tool` and `no_input` (5.2). |
| FR-RES-023 | A draw MUST NOT be returned when a job is abandoned, and a change of the executor's region after the start MUST NOT change the draw (5.4.5, 5.4.6). |
| FR-RES-024 | A draw and a regeneration MUST post no ledger transfer, MUST create or destroy no money or goods, and MUST add no reason to the registry (6.2.1). |
| FR-RES-025 | For every endowment and tick, `Stock_end` MUST equal `Stock_start - Drawn + Regenerated - Lost + Adjusted`, a test MUST assert it over the reference packs and a range of seeds, and `World.Audit` SHOULD report a difference (6.4). |
| FR-RES-026 | `EndowmentChanged` MUST be emitted once per endowment per tick, after its regeneration, in (`RegionID`, `ResourceID`) order, if and only if `Drawn`, `Regenerated`, `Short` or `Lost` is non-zero, with the fields of 14.1 and no map (14.1). |
| FR-RES-027 | `Requested - Granted` MUST equal `Short`, `Drawn` MUST not exceed `Granted`, and `AgentsDenied` and `AgentsReduced` MUST count the requests granted 0 and the requests granted less than asked but more than 0 (14.1). |
| FR-RES-028 | The `Warn` "endowment exhausted" MUST be emitted for an endowment in a tick with `Short` above 0 and `ExhaustedRun` equal to 1 or a multiple of 10, at most once per endowment per tick, and `ExhaustedRun` MUST count consecutive ticks with `Short` above 0 and be saved (14.2, 15). |
| FR-RES-029 | The shock slot MUST hold per endowment a capacity scale and a regeneration scale from 0 to `MaxShockBP`, default 10 000, set only between ticks by `SetCapacityScales` and `SetRegenScales`, MUST recompute the effective values, and when the stock then exceeds the capacity MUST clamp it and count the excess as lost (6.3.1). |
| FR-RES-030 | The adjustment, `AdjustEndowments`, MUST change the stock by a signed delta clamped to [0, capacity], MUST emit `EndowmentAdjusted`, and MUST be counted in the identity of FR-RES-025 (6.3.2). |
| FR-RES-031 | A `constant` stock with `RatePerYear` 0 MUST never gain stock except by an adjustment, a `logistic` stock at 0 with `Refuge` 0 MUST stay at 0, one with `Refuge` above 0 MUST regrow from 0, and a `refill` stock with a positive rate MUST regrow from 0 (6.1). |
| FR-RES-032 | The errors of 13.2 MUST be reported with the sentinel and the path of the field, and `ErrUnknownResource`, `ErrResourceShape` and `ErrCurve` MUST be defined as root sentinels (13.2). |
| FR-RES-033 | The warnings of 13.3 MUST be reported and MUST NOT fail validation, `draw_impossible` for a positive-yield region whose capacity is below the draw, and the conditions that need the solver MUST be reported through it (13.3). |
| FR-RES-034 | Every field of 12 MUST have the default and the bound stated, and a value outside its bound MUST be rejected with `ErrOutOfBounds` naming the path (12, 13.2). |
| FR-RES-035 | The owner of the yield-table values MUST be this spec and the owner of the format, the lookup and the validation errors `spec-production.md` 6; the yield table MUST NOT depend on the module, and there MUST be no climate entity and no function from richness to yield (7.1). |
| FR-RES-036 | `no_comparative_advantage` MUST be reported when at least two regions of one strongly connected component and two sited recipes have positive mean site factors and no cross ratio, or its inverse, reaches 11 000 basis points (7.4.1). |
| FR-RES-037 | A recipe with `Draws` MUST count as existing in a region for the viability check only if its site factor is positive in some season and every draw is at most the effective capacity of the region's endowment, which can start (9.2, 13.3). |
| FR-RES-038 | A commodity MUST be endowment-sourced under the predicate of 9.2, and `raw_source_exhaustible` MUST be reported for a raw commodity whose only sources are finite stocks (9.2, 9.3). |
| FR-RES-039 | The solver's resource row MUST be `sum of d * x <= Phi` per (resource, region), with `Phi` the regeneration at the plan stock over a year divided by `TicksPerYear` for `refill` and `logistic`, and the regeneration plus `Stock_0 / HorizonYears` for `constant`, in exact rational arithmetic, and the abundance at the plan share MUST multiply the mean factor of the recipes that draw (11.1, 11.2). |
| FR-RES-040 | The report MUST give `Headroom` and `Rent` per (resource, region), `resource_short` when `lambda*` is below 1 with a tight resource row, and `endowment_runway` for a finite stock (11.3). |
| FR-RES-041 | The agent view MUST carry, for the agent's own region only, the stock, capacity and abundance of the endowments its role's recipes draw on, and no view MUST carry the endowment of another region (5.5.1). |
| FR-RES-042 | The state of 15 MUST be saved as plain data in a fixed order with no maps, and a world restored between ticks MUST regenerate, ration and report exactly as an uninterrupted one. |
| FR-RES-043 | Every agent of a region whose role lists a recipe that draws on an endowment MUST be able to draw on it, with the ration as the only arbiter, and the ration MUST keep no memory of past draws (10). |
| FR-RES-044 | `resource_overshoot` and `resource_unstable_plan` MUST be reported by the integer inequalities of 8.3.1 when the population module is on, with the plan share, and MUST NOT be reported when it is off (8.3.1). |
| FR-RES-045 | The module MUST draw nothing from the random source, and the random stream of a world MUST NOT depend on whether the module is on (5.3.3). |
| FR-RES-046 | Observation MUST NOT change results: a run with a logger and an observer attached MUST give the same stocks, grants and draws as one without. |
| FR-RES-047 | The snapshot MUST report per endowment the stock, the effective capacity, the abundance and `ExhaustedRun`, and the shock scales when non-default (14.4). |
| FR-RES-048 | The effective yield used by `UnitCost` and by the agent view MUST include the abundance of the snapshot, so that the cost anchor and `MaxExecutions` agree with the start (4.7.5, 5.5.1). |
| FR-RES-049 | A configuration that is a pack for a new setting MUST load, validate and run with resources and no Go code specific to it, and the two examples of 16, completed with the elided commodities, MUST load, validate and run with the invariants of testing.md 6.3 holding (architecture.md 9.4). |
| FR-RES-050 | `EndowmentChanged` and `EndowmentAdjusted` MUST contain no map and no pointer, their count per tick MUST be bounded by regions times resources, and each metric attribute of 14.3 MUST take only values present in the configuration. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-RES-001 | With the default logger and observer, the snapshot, the ration and the regeneration of a step of the reference chain MUST allocate nothing beyond scratch held by the world (observability-and-logging.md 4.5, 8.2). |
| NFR-RES-002 | The cost of a tick for this module SHOULD be proportional to the endowments plus the requests that draw, not to the households or the weights, and the ration SHOULD be a single sum when the stock covers the requests. |
| NFR-RES-003 | The regeneration, the abundance curve, the ration and the validators MUST have fuzz tests (testing.md 7.4) asserting that they never panic, that the stock stays in [0, C], that the grant never exceeds the request or the stock, and that the identity of FR-RES-025 holds. |
| NFR-RES-004 | The regeneration, the ration and the abundance SHOULD have benchmarks that call `b.ReportAllocs()`. |

## 21. Prior art and evidence

**General models of renewable resources, not opened for this spec.** Logistic growth, the
maximum sustainable yield `rC/4`, the collapse of a stock under a constant quota above it,
the catch-per-effort (Schaefer) form and the open-access equilibrium (Gordon) are standard
results in the economics of renewable resources, known from general knowledge and **not
opened** for this spec: unverified in the sense of `evidence.md` and N-018. They supply the
structure (three shapes, `F_max`, the tipping point, the loop A5) and **no number**. The
linearisation of 8.3 is derived here from the stated rules and checked by the models of 17.

**Clark** (`spec-population.md` 16): the third Malthusian assumption, diminishing returns on
a fixed factor, is read there; this spec provides the factor and takes the shape of the
assumption and none of Clark's rates. **Games**: a resource patch that depletes, and one that
loses yield to a floor as it empties (Factorio's finite and infinite patches are the example
in mind), are familiar; the pages were not opened and only the idea that a yield may fall
with depletion is used, as the abundance curve. `bazaarbot-reference.md` has gatherers limited
by labour only (`spec-production.md` 3.1.2). No default of this spec is set from a source.

## 22. Notes settled

| Note | Resolution |
|------|-----------|
| N-030 | Applied for RES: the yield-table values are owned here, the format stays in production, there is no climate entity and no mapping, richness is the capacity and regeneration scale of an endowment (7). The five-region endowments are proposed in 18. STY and REF keep their parts. Remove RES from the areas |
| N-031 | Applied for RES: tables keyed by season ID, the season index of a tick, regeneration by season (4.5, 7.3). WLD, OBS, INT and GDS parts unchanged. Remove RES from the areas |
| N-036 | Applied for RES: `Endowment` and the all-or-nothing take, the definitions, depletion and regeneration, and depletion visible by event (5, 6, 14). WLD (the snapshot and the ration before the starts, 5.2) and INT (the slots of 6.3) keep their parts. Remove RES from the areas |
| N-046 | RES's entries delivered (19). Remove RES from the areas |
| N-098 | Applied: the mechanism, the closed forms, the stability condition by shape and the models (8). POP must make the edits of 8.5. Close when POP has |
| N-104 | Applied for RES: `Endowments` is a region-keyed table of the one mechanism, with replace and scale modes (3.3.1). Remove RES from the areas; the others stay |
| N-021 | Applied for RES: the sourcing of a raw good by an endowment and the warning for an exhaustible source (9). EXT's part and FIS's stay open |
| N-045 | Applied for RES: the cut demand is counted by `Short`, `AgentsDenied` and `AgentsReduced` of `EndowmentChanged` and a bounded `Warn` (6.2.2, 14) |
| N-106 | Applied: the agent view carries the endowments of its own region only (5.5.1) |
| N-107 | Applied: this spec adds no faucet, no sink and no ledger reason (6.2.1) |
| N-091 | Two slots added for INT: the scales and the adjustment (6.3) |
| N-092 | The overlay for the resource scenarios is proposed (18) |
| N-016, N-017, N-027, N-103 | N-016: the rows of 12 are owed to the parameter-owner table and none is unnamed; N-017: new terms in the handoff; N-027: every number here is a placeholder; N-103: this spec was written by the process of that note |

## 23. Open questions

1. **Occupying draws (a lease).** A draw consumes. Land held for the duration of a job and
   then released is a concurrency limit, not a flow; a refill with a fallow approximates it.
   Next step: LND or CAP, when a pack needs a stock held.
2. **Removal that follows the yield (catch per effort).** The draw is fixed per execution (a
   grade story). A stock whose removal is proportional to its abundance has a self-damping
   term and, derived by hand, is stable for any effort below the growth rate, with
   extinction only above it (the Schaefer form). A `Removal` flag on the resource would
   switch it. Not in version 1: the fixed form is the more cautious. Next step: the first
   pack with a fishery or herd.
3. **The warm-start stock.** 11.4 proposes a row in `spec-viability.md` 8 for the stock of
   each endowment, or `InitialBP = PlanShareBP`. VIA decides.
4. **A carry in the ration.** The bias against a small request is at most one execution per
   agent per tick (5.3.6); a per-agent carried remainder would remove it at the price of
   state. Next step: measure it in RES-S4.
5. **Joint grants for a recipe with several draws.** The grants are independent and the unused
   part stays (5.4.2). Next step: a pack with two draws that both bind.
6. **`r_full` as a measure.** The closed forms of 8.2 need the ratio at the full draw; a run
   can measure it, and whether the library reports it (snapshot or VIA) is OBS's and VIA's.
7. **Per-region shapes and curves, regeneration that depends on something else (labour, a
   tool), a stock shared between regions.** Not modelled: a region differs in capacity,
   starting stock and regeneration scale; the rest is a second resource or CAP, TEC and a
   use case.
8. **The cost-anchor interaction.** The abundance in `E` makes scarcity reach prices through
   the belief bounds (RES-A6) and may freeze a market with a cost floor. Next step: RES-S3
   with perishables.

## 24. Where this lives in the code

| Concern | Home |
|---------|------|
| `ResourcesConfig`, `Resource`, `AbundanceKnot`, `SeasonRegen`, `Endowment`, `ResourceID`, `DefaultResource()` | root package, `resources.go`; defaults in `defaults.go`; bounds in `bounds.go` |
| Validation, the sentinels `ErrUnknownResource`, `ErrResourceShape`, `ErrCurve`, `ErrInsufficientEndowment`, and the warnings | root package, `validate.go`, `errors.go` |
| The regeneration (the three numerators and the carry), the abundance, the ration, the key, the fixed-factor inequalities | `internal/endowment`: pure functions over primitive slices and indexes (architecture.md 2.9); stdlib only, `math/bits` for the 128-bit products |
| Piecewise-linear evaluation of a knot list, shared with the population schedules | `internal/curve` (a new package with one purpose; `spec-population.md` 19 may use it) |
| The endowments array, the snapshot, the take, the shock slots and the identity | root package, `world.go` and `resources.go` |
| The events `EndowmentChanged` and `EndowmentAdjusted` | root package, `event.go` |
| The log keys `resource`, `stock`, `short` | `internal/logkeys` (key strings only) |
| The reason constants | root package, `event.go` (exported constants) |
| The sustainable-draw rows, `Headroom`, `Rent`, `resource_short`, `endowment_runway` | `internal/steady`, called through `SolveSteadyState` (`spec-viability.md` 6) |

```go
// internal/endowment: stdlib only, no root types. A sketch, not final.
type Shape int // Constant, Refill, Logistic

// Regen returns the new stock, the new carry and the units added for one tick.
func Regen(shape Shape, stock, capacity, rate, refuge, scaleBP, seasonBP, carry, denom int64) (newStock, newCarry, added int64)

// Abundance returns the yield factor, in basis points, for a stock and a capacity.
func Abundance(knots []curve.Knot, stock, capacity int64) int64

// Request is one agent's demand on one endowment: w executions of a draw of d units.
type Request struct {
	Ref        int // index into the caller's slice, returned in the grant
	Executions int64
	Draw       int64
	Key        uint64
}

// Ration fills grants[i] with the executions granted to reqs[i].
func Ration(available int64, reqs []Request, grants []int64)

// Key is the stateless tie-break key of the ration (5.3.3).
func Key(tick, agent int64) uint64
```
