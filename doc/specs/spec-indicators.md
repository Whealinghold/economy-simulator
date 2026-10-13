# Spec: Indicators: derived measures of a run, national-accounts-style measures, index numbers, inequality and the welfare distribution

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-IND-NNN` and
`NFR-IND-NNN`, which tests cite (testing.md section 4). They are in section 20.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)
decision 7: "macro is measured, never set"), version 1 (`vision.md` section 4).
It is the measure step of the loop of the roadmap's "Macro is measured, never
set": measure (IND), compare with the expected properties (STY), adjust a knob
(CLB), run again (EXP). It needs `spec-observability.md` (OBS: the snapshot, the
events, the names table), `spec-world.md` (WLD: the tick, the boundary at which
a measure exists), `spec-ledger.md` (LED: the reasons, the sectoral identities,
balance sheets), `spec-goods.md` (GDS: the commodity classes), `spec-scale.md`
(SCL: weights, the households count) and `spec-integration.md` (INT: the
per-agent record). It reads `spec-regions.md` (the relocation totals),
`spec-transport.md` (the delivered price), `spec-external-sector.md` (the
balance of trade), `spec-money-supply.md` (the supply and its identities),
`spec-welfare-and-wages.md` (the welfare ratio and the observed wage),
`spec-stability.md` (the failure regimes) and `spec-checkpoint.md` (a measure
across a restore), and asks several of them for small edits (section 22).

**Every number in this spec is a placeholder** except where a derivation is
shown. Nothing has run in the library, bazaarBot has not been run, and no
acceptance band exists (notes N-015, N-027): this spec fixes **what a measure
is**, never what its value should be; the bands are `spec-stylised-facts.md`'s.
The scripts of section 17 are stylised Docker Python models of the integer
rules, not of the library; they check the arithmetic against exact rationals and
the worked examples of section 18. The setting of the worked examples, a river
valley with two regions `alpha` and `beta` and four commodities, is an
**example**, flagged as such; the core and its vocabulary are setting-agnostic,
and a commodity, a class, a role or a mode is data (architecture.md 9.1):
nothing in this spec branches on one (a preset takes a class name as an
argument, from the consumer's data).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

The world decides what happens; OBS decides what it tells the outside; this spec
decides what a consumer **derives** from that telling, how, and with what
arithmetic, so that two consumers who ask for "the price index of food" or "the
Gini of money" get the same integer. It covers:

- **where an indicator lives and what it reads**: a pure function of a snapshot
  and of events, never of live state, drawing nothing; its package, its inputs,
  its cost classes, its errors, and how a consumer defines its own from the
  generic aggregators (sections 3 and 4);
- **the arithmetic**: fixed point only, index numbers in basis points with a
  stated base and rounding rule (Laspeyres, Paasche, Fisher with an integer
  square root, chain), the Gini coefficient and the quantiles over weighted
  households, and every overflow path as an error or a clamp with a flag
  (section 5);
- **households, not agents**: the weights, the pseudo-agents, the zero
  inequality within a cohort, and what a measure says about a world of weight
  above 1 (section 6);
- **the accounts**: gross domestic product by the production, the expenditure
  and the income view, the identity between them, the sectors, balance sheets
  valued at a caller's prices, what a price is for valuing (section 7);
- **regions and the realm**: the attribution of a flow, the reconciling item of
  a relocation, interregional trade, the price level of goods that travel
  (section 8);
- **the standard presets**: the named list, and for each its definition, inputs,
  units, cost, the failure it must not hide and the first test against a hand
  computation (section 9);
- **time**: windows, the calendar year and the season, annualisation, how a
  series is stored (by the consumer) and what a recorder keeps (section 10);
- **what IND hands** to STY, EXP and CLB, as requirements (section 11).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| What a snapshot or an event contains, and when it is emitted | `spec-observability.md` (OBS); this spec reads it |
| The acceptance bands, the stylised facts, the thresholds of a failure regime | `spec-stylised-facts.md` (STY); IND supplies the measure a band is stated in |
| A sweep, a calibration run, a series store | `spec-experiments.md` (EXP), `spec-calibration.md` (CLB) |
| A rule that reads a measure and changes a setting | `spec-policy.md` (PLC); no rule of the library calls `indicator` (3.1), and the built-in `rule` of PLC reads four gauges that are field reads with the definition of the matching preset (`spec-policy.md` 8.3.2) |
| The money supply as a mechanism (flows, the liquidity floor) | `spec-money-supply.md` (MON); IND measures it |
| The steady-state price a measure may be compared with | `spec-viability.md` (VIA); IND takes it as an input (9, `Offset`) |
| Taxes, tariffs, rent, interest, premiums, a net international position | FIS, LND, CRD, INS; their flows are ledger reasons and events that the tally and the sector table read (7.8.2); a net international position has no row in version 1 |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | An indicator is a **pure function of a snapshot (or two) and of the events a recorder kept**, with a price table and a class table as plain arguments. It takes no `*World`, no callback into one and no random number, and it is deterministic across platforms | 3.1, 3.3 |
| 2 | **Nothing in the world reads an indicator back.** This is a requirement with three tests: no package of the module imports `indicator` except through the consumer's own code; no exported function takes a world; a run with every preset computed every tick equals a run with none, byte for byte | 3.1, 20 |
| 3 | The home is a public package `indicator/` (it imports the root; the root, `config/`, `strategy/` and `internal/` never import it) over `internal/measure` (primitive checked arithmetic, standard library, `math/bits` only). No layered name, no interface | 3.2 |
| 4 | The **generic aggregators** are three: a value of flows at a price table, a weighted statistic over household rows, and a window statistic over a series. Every standard indicator is a **preset**: a named call of these with fixed arguments, and a consumer writes its own the same way | 3.6, 9 |
| 5 | Arithmetic is integer: sums in signed 128 bits, one rounding rule (half to even), indices in basis points against a base window of 10 000, the Fisher mean by an integer square root rounded to the nearest integer, the Gini coefficient exact in 128 bits and rounded once | 5 |
| 6 | An overflow is an **error** where the result cannot mean anything (a value that does not fit `int64`), and a **clamp with a flag** where the measurement matters most when the world is worst (an index of 2 147 483 647 bp or more, a runaway inflation) | 5.6 |
| 7 | **Households, not agents**: every population measure weights a row by the households it stands for, pseudo-agents are never rows, and every inequality measure reports beside it the share of households that sit in a cohort (where inequality is zero by construction) | 6 |
| 8 | GDP is stated **three ways that are one fact seen from three sources**: production (events: recipe output less recipe input, with work in progress), expenditure (events and two snapshots: consumption, change in stocks, losses, carriage, trade) and income (the households' own booked income). Production equals expenditure **exactly**, at the realm for any one price table and by region with named interregional terms; income is a diagnostic with named reconciling items and is never claimed to equal either | 7 |
| 9 | A **price for valuing** is always an argument. Three constructors: the window unit value of the published price, the effective reference price, and a delivered price (origin price plus freight). A base-window table gives constant prices | 4.3, 8.4 |
| 10 | By region: a flow belongs to the region of the account debited or credited by the ledger (LED 3.5, 7.4); the **reconciling items** of a region are the relocation (money exactly, goods by total units) and interregional trade (shipments and contract payments); the realm's identity has none | 8 |
| 11 | Twenty-five presets, listed once in 9, each with a first test that is a hand computation of the worked example of 18 | 9, 18 |
| 12 | A **window** is half-open ticks `[From, To)` bounded by two snapshots; a series is stored by the consumer as `[]Point`; the library keeps no history; a recorder keeps running sums, not ticks | 10 |
| 13 | A restore does not break a window: a **tally** is the consumer's plain value, events are not repeated across a restore, and a tally started at `WorldRestored` opens with that event's snapshot | 10.5 |

## 3. Where indicators live and what they read

### 3.1 The rule: measured, never set

3.1.1. **An indicator is read-only.** It is computed from copies of the world's
own output (events and snapshots, which are deep copies of plain data,
architecture.md 12.4) and writes nothing back. The world holds **no** value
that an indicator produced, no agent reads one, no rule, damper, cadence or
slot of the library takes one as input, and none of the library's code calls
into `indicator` (roadmap, "Macro is measured, never set"; ADR 0001 decision 7;
`spec-world.md` 9.4).

3.1.2. **Why this is stated as a requirement and tested.** The roadmap's rule
is a design principle, and a principle erodes by small steps: a warning that
"just reads the price index", a damper that "just uses the Gini". Three tests
make the erosion a failing build (FR-IND-001 to FR-IND-003): an import-graph
test (no package of the module other than `indicator` itself, its tests, the
separate modules `otelobserver` and `examples/` imports `indicator`); a
signature test (no exported identifier of `indicator` has a parameter or
result of type `*World`, `World`, or a function type that takes one); and a
non-interference test (the same seed, configuration and calls, run once with a
consumer that computes every preset after every step and once with none, give
the same checkpoint digest, event stream and snapshots).

3.1.3. **What a consumer may do** is its own affair: a game may read an
indicator and set a slot (INT 3.5, OBS-A1). That is a loop that exists in the
consumer, with a delay of one tick by construction, and it is registered as a
hazard (16, IND-A1), not as a mechanism of the library. A controller that is
part of a *setting's* rules is a policy actor (`spec-policy.md`, PLC), with its own
interface (`Policy`, architecture.md 4.3) and its own register entries.

3.1.4. **No draw, no time, no state.** An indicator takes no random number
(WLD 7.5), reads no clock (architecture.md 7.3), keeps no package-level state
(architecture.md 7.1) and starts no goroutine (12.2). The same inputs give the same output on
every platform (7.4). A **tally** (4.2) is state, but it is the consumer's
value, not the world's, and is not in a checkpoint (15).

### 3.2 The home

3.2.1. **Decision.** The public API is the package **`indicator`**, at
`<module>/indicator`. It imports the root package (it needs `Snapshot`, the
event types, `AgentState`, `RegionID`, `Price`, `Money`, `Quantity`), and the
root, `config/`, `strategy/` and `internal/` never import it, so the import
graph has one direction (`indicator` to root to `internal/`; architecture.md
2.7). Beneath it, **`internal/measure`** holds the checked arithmetic over
primitive types (the 128-bit sums, the half-to-even division, the integer
square root, the Gini accumulation, the nearest-rank quantile), stdlib only,
with no knowledge of a commodity, a region or an agent (architecture.md 2.9).
The root converts nothing: `indicator` converts at its own call into
`internal/measure`, inline.

3.2.2. **Why a package and not a root file.** The root is the world's API and
is already the widest file set of the module (architecture.md 2.2). An
indicator is not part of a world; a consumer that wants none does not import
the package, and a consumer that defines its own indicator imports it as one
unit. A package named `indicator` is named for what it provides, not for a
layer (architecture.md 2.3). It is not `util`, `helper` or `metrics` (2.4; and
`metrics` would be confused with the OpenTelemetry instruments of OBS 6).

3.2.3. **Why no interface.** Nothing in this spec has a second implementation
or a test seam that a function does not serve (architecture.md 4.2). A
consumer-defined indicator is a function that calls the aggregators (3.6), not
a type that implements an interface; the extension points of architecture.md
4.3 are unchanged.

3.2.4. **Names.** The package name is a single lowercase word, so exported names
do not repeat it: `indicator.Window`, `indicator.Tally`, `indicator.Prices`,
`indicator.StatGini`, `indicator.GDP`. The preset names of 9 are the exported
function names. `indicator.New` is not used: the package has several types
and no single primary type, and a constructor is named for what it builds
(`NewTally`, `NewPrices`; architecture.md 6.1).

### 3.3 What an indicator reads

| Input | Type (root package unless named) | Taken from | Used by |
|-------|----------------------------------|------------|---------|
| A snapshot at a boundary | `Snapshot` | `World.Snapshot()` between steps (OBS 5), or the one in `WorldRestored` | every preset |
| A pair of snapshots | the window's `Open` and `Close` | two calls, `Open.Tick < Close.Tick`, the same `ConfigHash` | flows against stocks (7.3), changes over a window |
| A tally | `indicator.Tally`, plain data, fed by `Observe` | the events of the steps between the two snapshots (4.2) | every flow measure |
| A price table | `indicator.Prices`, a price per (region, commodity) | 4.3 | every value measure |
| A class table | `indicator.Classes` from the commodity tags | `Config.Commodities[].Classes` (GDS 6.1), through the names table | every measure by class |
| Household rows | `[]indicator.Household{Weight, Total int64}` | INT `Agents(r)` and `Agent(a)` (`AgentState`), 4.5 | inequality, quantiles, shares |
| A series | `[]indicator.Point{Tick, Value int64}` | the consumer's own store (10.3) | window statistics, growth |
| The calendar | `Calendar` (`TicksPerYear`, `At`) | `World.Calendar()` (CAL 4) | annualisation, years, seasons |
| The region graph | `Graph` (components) | `World.Graph()` (REG 4.3) | `PriceGap.Connected` |
| A base | `indicator.Base`: the base window's `Prices` **and** its quantities by (region, commodity) for each `Weights` choice, frozen from a closed tally (`BaseOf(t, p)`) | the consumer's own store | `PriceIndex`, real values |
| Configuration data | `Config` fields named in the preset's row | `World.Config()` | basket, upkeep, mode haul commodity, durability |

A preset names in its row (9) the inputs it uses, so a consumer can see what to
keep. No input is a live reference: a `Snapshot` may be marshalled on another
goroutine (OBS 5.1.1), and so may every other row of the table.

3.3.1. **Tick resolution.** A tick is the finest resolution: no value exists
between `TickStarted` and `TickEnded` (WLD 13, OBS 10.1). An indicator is
defined at a tick boundary or over a window of whole ticks, and never "during"
a tick.

3.3.2. **The names table.** The typed IDs of an event and a snapshot are
positions in the configuration (OBS 3.2). An indicator keyed by an ID is valid
for the configuration it was computed against and carries the `ConfigHash` of
its inputs in its result, so a consumer that stores a series across
configurations keys it by the string ID from the names table (OBS 10.1, last
row). Two snapshots with different `ConfigHash` values are refused with
`ErrConfigMismatch` (3.5).

### 3.4 Cost classes

A preset states its cost class. The class numbers are IND's own and are **not**
INT 6.1.3's (INT's class 2 is a per-entity query). Counts, not timings (the library measures no
wall-clock time, architecture.md 7.3; N-015).

| Class | Cost | Which presets |
|-------|------|---------------|
| 1 | `O(rows of one snapshot)`: a copy or a fold of the aggregate rows (regions, roles, markets), independent of the number of agents (OBS 5.6) | `Sectors` (balances; its closing identity needs a tally and is class 2), `Households` and `Unfed` at a boundary, `BasketLevel`, `StockCover` at a boundary (over a window it is class 2), `RoleShares`, `RealWage`, `HouseholdIncome` (per snapshot), `Employment` at a boundary |
| 2 | `O(rows of a tally)`: one pass over the window's running sums, bounded by regions, reasons, commodities, lanes (never by ticks, orders or agents) | `GDP`, `ValueAdded`, `PriceIndex`, `DeliveredPrice`, `PriceGap`, `TradeBalance`, `Turnover`, `PriceRange`, `Pinned`, `Dispersion`, `Offset`, `Unemployment`, `WageBill`, `Losses`, and `Households`, `Unfed` and `Employment` over a window |
| 3 | `O(A log A)` for `A` household rows: a sort and a pass; the rows themselves cost the caller `O(A)` queries (4.5) | `WelfareDistribution`, `Inequality` |
| W | `O(points)` over a consumer's series | `Stats`, `Growth`, `Change`, `PriceRange` and `RoleShares` over a series |

A consumer that wants every preset every tick pays class 2 and class 1 (the
tally is updated per event at `O(1)` and a preset reads running sums) and
class 3 once per window or per frame as it chooses.

### 3.5 Errors

Sentinels that a caller may branch on (architecture.md 8.4); each is returned
wrapped with the preset's name and the field (8.2). An *undefined* measure is
not an error: it is a result with `OK` false (a zero base, an empty set, a
role with no upkeep), as OBS's `HasMeanRatio` is.

| Error | When |
|-------|------|
| `ErrConfigMismatch` | two snapshots, or a snapshot and a price table, from different `ConfigHash` values |
| `ErrWindowOrder` | `Open.Tick` not below `Close.Tick`; or a tally whose events do not cover `[Open.Tick, Close.Tick)` exactly once (a missing `TickEnded`, a failed step, an event before the window) |
| `ErrUnknownClass` | a class name that no commodity of the names table carries |
| `ErrRange` | an ID outside its slice of the names table; a negative weight or value where it is not allowed; a share above 10 000 |
| `ErrNegativeValue` | the Gini coefficient of a row set with a negative value (5.4.3) |
| `ErrOverflow` | a result that does not fit `int64`, or a guard of 5.6 |

### 3.6 A consumer's own indicator

A consumer defines an indicator by calling the three aggregators with its own
selection. They are the only generic part; each preset of 9 is a few lines
over them (9.2 shows the form).

```go
// Value is the sum over the selected (region, reason, commodity) rows of the
// window's flow totals, valued at p, in minor units. sel is a plain value.
func (t *Tally) Value(sel FlowSel, p Prices) (Money, error)

// Stat is a weighted statistic of household rows.
func Stat(rows []Household, op StatOp, arg int64) (StatResult, error)

// Window statistics of a series.
func Stats(points []Point) SeriesStats
```

`FlowSel` is `{Regions []RegionID; Reasons []Reason; Commodities []CommodityID}`
with a nil slice meaning "all"; a class is turned into its commodities by the
class table (`Classes.Of(name)`). `StatOp` is a closed set of constants (`StatSum`,
`StatMean`, `StatQuantile`, `StatShareBelow`, `StatGini`). These signatures are provisional
until the code (they fix the semantics, not the spelling), and the closed sets
are rule constants of the package (architecture.md 9.6). A consumer's measure
is deterministic by construction: it can use no map iteration, since the inputs
are slices in canonical order.

## 4. The inputs

### 4.1 The window and the two snapshots

4.1.1. **A window** is the half-open interval of ticks `[From, To)` that the
steps `From, From+1, ..., To-1` ran, bounded by a snapshot taken before step
`From` (`Open.Tick = From`) and one taken after step `To-1` (`Close.Tick = To`).
`Snapshot.Tick` is the tick the *next* step will run (OBS 5.3.1), so the two
snapshots of a window are `Tick` apart by the number of its steps, and the events
of the window are exactly those emitted by those steps.

4.1.2. **What the window must be.** A window of `n >= 1` steps, none failed
(`TickEnded.Failed` and `RunEnded.Failed` are errors: a failed step is not a
tick, WLD 3.5). The tally verifies this: it counts `TickEnded` events and the
first and last `Tick` it saw, and `Close` returns `ErrWindowOrder` when the
count is not `Close.Tick - Open.Tick` or an event carries a `Tick` outside the
window. The one exception is the construction bucket, `BoundaryFlow` with
`Tick` -1, which belongs to the window whose `Open.Tick` is 0 and is kept apart
(7.1.4).

4.1.4. **Calls between ticks, and events while closed.** One rule. A tally
**ignores every event while no window is open**, and `Open(s)` takes its
households from `s`. A consumer therefore takes the snapshot **after** the calls
it wants inside the opening state (`AddAgent`, `Relocate`, a slot), then `Open`s
with it; an event of a call made after `Open` has the tick of the next step,
inside the window, and is counted. Money and goods that a call moved are in the
balances at once and in a `BoundaryFlow` only at the next tick end
(`Snapshot.Pending`, OBS 5.3.3, WLD 8.1.3), so **the flows of a window are the
tally's rows less `Open.Pending` plus `Close.Pending`** (7.1.5): what `Open`
already holds is not counted again, and a call after the last step, which is in
`Close` and in no flow, is. A call between `Close(s)` and the next `Open` is in
neither window's events; `Open` with a snapshot taken after it is exact, and
reusing `s` taken before it gives a wrong household count (the tally cannot see
the event), which is why the rule says snapshot first.

4.1.3. **A window of one tick** is legal and is the finest: its flows are the
`BoundaryFlow` rows of that tick and its stocks the two boundaries. A measure
that needs a longer window (a Gini of income over a window, a velocity) says so
in its row.

### 4.2 The tally: what a recorder keeps

4.2.1. **Decision.** A **tally** is a value that accumulates the running sums of
a window from the events of its steps, and nothing else. It is the answer to
the brief's "what a recorder needs to keep": **running sums, not ticks**. It
implements `Observer` (`Observe(Event)`; architecture.md 4.8), ignores every
event type it does not use (rules 4.4), is not safe for concurrent use (a
consumer's observer runs on the stepping goroutine, architecture.md 12.5), and
**must not call the world** (12.5) - which it cannot, since it holds none.

4.2.2. **What it keeps.** Every row is keyed in the canonical order of its key
and its size is bounded by the configuration, never by agents, orders, trades
or ticks (FR-IND-004).

| Event | Kept, per | As |
|-------|-----------|----|
| `BoundaryFlow` | (region, reason, commodity or money) | the sum of `Total` over the window; the `Kind`; the rows of `Tick` -1 apart; and `Open.Pending`, copied from the opening snapshot (7.1.5) |
| `MarketCleared` | (region, commodity) | `SumPV` (128-bit sum of `Price * Volume`), `SumV`, the last `Price`, the lowest and highest `Price`, the ticks with volume, the last `PriceAge`, and the ticks in which units were left unmatched on **both** sides with no volume (`StuckTicks`, for `Pinned`) |
| `BeliefsUpdated` | (region, commodity) | `Outcomes`, `AtFloor`, `AtCeiling` summed; the last `LowCentre`, `HighCentre`, `MeanCentre`, and the lowest value over the window of `rdiv(10 000 * (HighCentre - LowCentre), MeanCentre)` (for `Dispersion`) |
| `ShipmentDispatched` | (origin, destination, mode) | shipments, `HaulUnits`, carrying `Units`, the units of the lines by commodity, and the shipments with `LossBP` above 0 (`Struck`, for `Losses`) |
| `ShipmentArrived` | (origin, destination, mode, commodity) | `Shipped`, `Delivered`, `Diverted`, `Spoiled`, `Lost` summed, and by commodity the `AccountLost` of a self-carried trip |
| `StoreLoss` | (region, store class, commodity) | `Stores` struck and the units `Lost` summed |
| `LabourRound` | region (and worker role) | `Offered`, `Hired`, `NoVacancy`, `NotStarted`, `HouseholdTicks`, `Paid` and `Refunded` summed, and the counts of formation refusals by reason (for `Unemployment`, `WageBill`, `Employment`) |
| `FiscalFlowApplied` | (flow, authority, region) | the `Applied` and the `Due` summed, with the `Kind` and the `Class` (for the regional terms of 8.2) |
| `ContractSettled` | (deliverer's `Region`, `To`) | `Paid`, `Refunded` summed (the event carries the deliverer's region, OBS 4.1.6, so no contract is remembered) |
| `ContractFormed` | (`Region`, `To`) | `Escrowed` summed |
| `ReliefPaid` | region | `Paid` and `Recipients` summed |
| `AgentRelocated` | (from, to) | `Money`, `Units`, `Weight` summed |
| `AgentSpawned`, `AgentBankrupted`, `WeightChanged`, `RoleChanged` | (region, role) | the running **household count**, seeded from `Open` (6.3) |
| `PopulationChanged` | (region, role) | `BirthsDone`, `DeathsDone` summed (and the counters of the last row) |
| `UpkeepShortfall` | (region, role) | `Unfed` summed and the largest `MaxHunger`; its own `Households` field is **not** the denominator (6.3, 9 row 12) |
| `OutsideTraded` | (node, gate, commodity, side) | `Traded`, `Value`, `ListValue` summed; the side is the counterparty's: a **bid** is the outside buying (an export of the realm), an **ask** the outside selling (an import) |
| `EntryLimited`, `PopulationChanged`, `MoneyFlowLimited`, `ContractRound`, `MigrationReviewed` | region (and role) | the counts of each cap, limit or refusal that acted: `RegionCap`, `RoleCap`, `Cooldown`, `Hysteresis`, `ExitCap`, `Unfunded`, `Refused`; `BirthsCapped`, `BirthsUnfunded`, `DeathsCut`, `DeathsHeld`; `Short`, `Limit`; `TooMany`, `PayerFunds`; `OutCap`, `InCap`, `Unprovisioned`, `Bound`, `Held`; summed: the **binding counters** of 8.7 |
| `RecipeExecuted` | (region, recipe) | `Executions` and `Outputs`, `Disposed` by commodity, summed (7.2.4) |
| `TickEnded`, `WorldCreated`, `WorldRestored`, `RunEnded` | the window | the step count, the failure flag, the opening snapshot of a restore (used only when no window is open, 4.2.5) |

4.2.3. **What it ignores, on purpose.** The per-order and per-trade classes
(`OrderPlaced`, `OrderRejected`, `TradeExecuted`; OBS volume classes O and R,
more than 99% of the events of the first consumer's world, OBS 8.3). A tally
that needed them would force every consumer to pay for them. `MarketCleared`
gives the turnover to within rounding (4.3.2), which is all an indicator
needs. A consumer that wants them filters its observer.

4.2.4. **Size.** For the first consumer's world (5 regions, 30 commodities, about
8 boundary reasons that are non-zero in a normal run, 85 active markets, 12
lanes) the tally holds of the order of a thousand rows, of the order of 30 KB,
and is the same for 50 agents or 200 000 (the placeholder sizes of
`spec-scale.md` 6.3; counted, not measured). The tally remembers no contract and
no shipment.

4.2.5. **Lifecycle.** `NewTally()` is empty. `Open(s Snapshot)` starts a window at
`s.Tick`, copies the households by (region, role) from `s.Roles` and
`s.Pending` (the pending flow totals), and clears the sums; it clears every
row, so nothing of an earlier window survives (4.1.4). `Close(s Snapshot) (Window, error)` verifies 4.1.2, freezes the window and
returns it, so that a result computed from it is a function of the frozen
value. `Open` may be called again on the same value to start the next window
(the usual loop: `Close` then `Open` with the same snapshot). A tally that is
fed `WorldRestored` while a window is open ignores it for that window: a restore
changes nothing in a window that straddles it (10.5.1, FR-IND-055). One with no
window open `Open`s a window at that event's snapshot (10.5.2).

4.2.6. **The fields of `Window`.** A `Window` is plain data with exported fields and
no tags (it is the consumer's value, 10.5.1): `From` and `To` (`int64`, the
ticks of 4.1.1, `To - From` steps), `ConfigHash` (the hash of the configuration of
the world the events came from, from `WorldCreated` or `WorldRestored`; two windows
of different hashes are not combined, 3.3.2), the number of steps, the two
snapshots' household counts by (region, role), `Open.Pending` and `Close.Pending`
(7.1.5), and the rows of 4.2.2 in canonical order. A measure takes a `Window`
and nothing else of the tally.

### 4.3 What a price is for valuing

A value is a quantity times a price, and the price is a choice: it is always an
**argument** (`indicator.Prices`), never read from the world, so that "GDP at
base-period prices" and "GDP at today's" are two calls and not two behaviours.

4.3.1. **`Prices`** is a table of `Price` per (region, commodity), plain data,
with `At(r, c)`. Rows exist for every (region, commodity) of a region that is
not an outside node; an outside node has none (an outside node has no market,
REG 3.5, and its trades are priced in the gate region's market).

4.3.2. **The window unit value** (`PricesOfWindow(t)`): for each market the
volume-weighted mean of the published prices of the ticks in which units
traded, `rdiv(SumPV, SumV)` half to even, and where `SumV` is 0 the **last
published price** carried forward, which is the price the market publishes in a
thin market (CLR 5.4: the previous price carried forward from the seed price).
`MarketCleared.Price` is itself the volume-weighted mean of that step's trades
rounded half to even (CLR 5.2 for the rounding of the midpoint, 5.4 for the published price), so `Price * Volume` differs from the exact
turnover `sum(p_i * q_i)` of the step by less than `Volume / 2`, so `SumPV`
differs from the exact turnover of the window by less than `SumV / 2` and the
unit value from the exact unit value of the window's trades by less than one
minor unit (half from the roundings of the steps, half from the last one). A
consumer who needs the exact value of trades has `TradeExecuted`
and pays class R (4.2.3).

4.3.3. **The published price at a boundary** (`PricesOfSnapshot(s)`): the
`MarketRow.State.Price` of each market (OBS 5.3.5), which is the price of the
last step.

4.3.4. **The effective reference price** (`PricesOfReference(cfg)`): the
commodity's `ReferencePrice`, or the region's override `RegionPrices` where it
has one (GDS 6.3, REG 5.2). It is the price of a commodity that never trades
(a discard-only output, GDS 6.4) and the base of the basket of 8.4.

4.3.5. **A base table.** A table taken from an earlier window (`PricesOfWindow`
of the base window) is the **constant price** table of real measures (5.3.6).
It is a value like any other; nothing marks it as "the" base except the
consumer's choice and its tick (`Prices` carries the `ConfigHash` and the
window it came from, so a mismatch is `ErrConfigMismatch`).

4.3.6. **The delivered price** is not a table: it is a per-lane price (origin
price plus freight) defined in 8.4.

4.3.7. **A price is never zero where a value matters.** A market's published
price is at least 1 (GDS 6.4: a reference price is between 1 and `MaxPrice`; CLR 5.4); a table built from
a world therefore has no zero price. A consumer's own table may hold zero and
the value of a quantity at it is zero; an index with a zero base value is
undefined (5.3.1).

### 4.4 Classes

4.4.1. **`Classes`** is built from the configuration: for each class tag, the
sorted set of commodity IDs that carry it (a commodity has at most 8, GDS 6.1).
`Classes.Of(name)` returns the set or `ErrUnknownClass`; `Classes.All()` is the
set of every commodity. A commodity in two classes is counted in each class's
measure and **once** in the whole-economy measure (a measure over all
commodities never sums class measures: 9, `PriceIndex`).

4.4.2. **No code names a class** (FR-GDS-017). Every preset that groups takes
the class name or the set as an argument; the standard names of this spec's
examples (`food`, `fuel`) are the example pack's data.

### 4.5 Household rows

4.5.1. A **household row** is `{Weight int64; Total int64}`: a weight (the
households an agent stands for, at least 1) and the agent's **total** of the
quantity, exactly as the world holds it (`AgentState.Money` is the cohort's sum,
SCL 2.2). **The households of a cohort share the total**: each holds `Total /
Weight`, which need not be an integer (a relief payment or a birth adds an amount
that is not a multiple of the weight), and no per-household integer is formed
until a result is stated. Hence `S = sum Total` is the exact total in the
snapshot (`Sectors.M_h`). For a per-household quantity that is a *state* of the
household and not an amount (a welfare ratio in basis points, a hunger count) the
row's `Total` is `Weight * value`, exact. Rows are the input of every
distribution measure (inequality, quantiles, shares); the rules over rows of
totals are 5.4.1 (the Gini coefficient, with no division before the last), 5.5.2
(a quantile returns `rdiv(Total, Weight)` of its row) and 5.5.3 (a share counts
the households of a row below `x` when `Total < x * Weight`).

4.5.2. **Source.** INT's `Agents(r)` lists the live agents of a region in
ascending ID without pseudo-agents (INT 6.2), and `Agent(a)` returns an
`AgentState` with `Weight`, `External`, `Money` and the welfare ratio and
whether it exists (INT 4.7.2). `indicator.RowsOf(states []AgentState, col Column,
opt RowOpt)` builds rows from them; `Column` is a closed set (`Money`, `WelfareRatio`,
`EarnedRatio`, `Hunger`, `Wealth`), and `RowOpt.ExcludeExternal` drops the agents with
`External` true (INT 9.4: a player is a household in the counts; a measure that
does not want it in a regional average filters it). A row whose value does not
exist (a ratio of an agent with no upkeep, OBS 5.3.4, WEL 2.2.3) is **left out**
and counted in the result's `Absent` field, with its weight (4.5.4).

4.5.3. **Cost.** `AgentState` is a query of the second class (INT 6.1.3), `O(|tradable
set| + |recipes| + |lots|)` and an allocation each: for the first consumer's
world (5 000 agents, a tradable set of 8, 7 recipes, 8 lots) about 23 reads
and a few hundred bytes of allocation per agent, of the order of 0.1 ms and a
few megabytes for the whole population (counted, not measured), which is why
these rows are built **once per window or per frame, not per tick**. A bulk read
of one column is the open question 1 of section 21.

4.5.4. **What a row set reports.** Every statistic over rows returns, with its
value, the `Households` (the sum of the weights used), the `Agents` (the rows),
the `Absent` households, and the `Cohort` households (the sum of the weights of
rows with `Weight > 1`). 6.2 says what `Cohort` is for.

4.5.5. **Money, and not wealth.** The distribution of **money** needs no price.
The distribution of **wealth** (money plus goods valued at a price table) needs
the holdings of each agent, which `AgentState` has (INT 4.7.2, Holdings) and
the balance-sheet valuation of LED 9.2; `RowsOf` offers `Wealth` as a column
that takes a `Prices` table and values an agent's holdings at its region's
prices, with checked arithmetic: a row `Total` that does not fit `int64` (a
holding of 1 000 commodities at the bounds can reach `10^21`) is `ErrOverflow`
for the call; it cannot occur for a realistic world and is not claimed away. Goods are valued as new: the snapshot and the
state do not track the remaining durability of an item as a quantity, so a
worn durable is valued at the full price (7.6).

## 5. Arithmetic

Everything here follows architecture.md 3.5: no floating point in anything an
indicator produces, checked arithmetic, one stated rounding rule, and an
overflow that is reported and never wrapped. A consumer converts a result to
floating point for display outside the package, as the rule says.
`indicator` and `internal/measure` import neither `math` nor any float type
(FR-IND-008); `math/bits` is allowed.

### 5.1 Types and units

| Type | Meaning | Unit |
|------|---------|------|
| `Money`, `Price`, `Quantity` | the root package's (architecture.md 3.3) | minor units; minor units per unit; whole units |
| `BP` | a ratio, an index, a share or a rate, `int64` | basis points: 10 000 is 1.0 |
| (count of households) | a sum of weights, `int64` | households |
| `Point` | `{Tick, Value int64}` | ticks; the value's own unit |

Every result struct has the fields of its row in 9 and, where it can be
undefined, an `OK bool`. A value with `OK` false is zero and must not be read.
Every result that depends on a table carries the `ConfigHash` of its inputs
(3.3.2).

### 5.2 Wide sums and one rounding rule

5.2.1. **Sums are 128 bits.** A sum of values is accumulated in a signed 128-bit
integer (`internal/measure.Int128`, two words, built with `math/bits.Add64`,
`Mul64` and `Sub64`), never in `int64`. The checked multiply and divide that
`spec-currency.md` 7 places in `internal/` are the same primitives: when that
package exists `internal/measure` uses its 128-bit helpers and adds only the
statistics (Gini accumulation, nearest rank, integer square root); the two are
one helper and not two (open question 11 if the package is not yet decided). The bound that makes this enough: a row
value is at most `MaxMoney` (10^18) in magnitude for money and at most
`MaxMoney * 10 000` for a ratio in basis points (a ratio of income to a cost of
at least 1 minor unit), a weight is at most `MaxWeight` (10^6), the sum of
weights is at most `MaxHouseholds * MaxRoles * MaxRegions` (10^15), and the sum
of money over the world stays below `3 * MaxMoney` (`spec-external-sector.md`
13.2). A sum is 128-bit, and **a sum that is later multiplied by 10 000** (every
ratio, share and Gini numerator) is checked against `2^127 / 10 000` (about
1.7 * 10^34) **before** the multiplication, and is `ErrOverflow` beyond it. For
money rows the sum is below `3 * 10^18` and the guard never acts. For ratio rows
the extreme bound (`10^15` households at `10^22`) is above the guard, so the
guard is an error path for a consumer's own rows or an absurd world, not a
claim that every valid world passes; no realistic world is near it (a mean ratio
of 10 000 bp over 5 000 households is a sum of `5 * 10^7`).

5.2.2. **One rounding rule.** A quotient `n / d` of two integers is rounded
**half to even**: to the nearest integer, and an exact half to the even integer,
symmetric for a negative `n` (`rdiv(-5, 2) = -2`, `rdiv(-3, 2) = -2`,
`rdiv(25, 10) = 2`, `rdiv(35, 10) = 4`). It is the rule of clearing's midpoint
(CLR 5.2) and of the other specs' rates, so a published price and a measure of
it round alike. A change of this rule is a behaviour change (architecture.md
3.5, 6.6).

5.2.3. **Rounded once.** A result is rounded at the **last** division and
nowhere before, with two named exceptions that are definitions and not
accidents: the Fisher mean takes the two rounded basis-point indices as its
inputs (5.3.4), and a chain multiplies the previous rounded link (5.3.5). A
sum of rounded parts is therefore never the rounded sum, and a table of shares
that is rounded cell by cell need not add to 10 000; every share is computed
from the exact sums (the shares of a distribution: 5.5.3). The rule holds for
every **derived** measure: a spread, a ratio of two levels, an annualised
rate or velocity is computed from the exact sums of its parts in one division,
never from a rounded measure (`Spread = rdiv(2 * 10 000 * |p_d - p_o|, p_d +
p_o)`, not a share of a rounded mean).

5.2.4. **Division by zero is not an error and not a panic.** A quotient whose
divisor is zero is *undefined*: the result has `OK` false (3.5). The divisor of
each ratio is the one its definition in 5 or 9 names. `OK` is **per field**:
`Laspeyres` and `Paasche` have their own flags (a zero base value at the base
quantities does not make the Paasche index undefined), `Fisher` is defined only
when both are; `Offset` is undefined for a reference price at or below 0; a
growth from a base at or below 0 is undefined; a share of an empty set is
undefined.

5.2.5. **Determinism.** Every sort is a total order that ends in the row's
position (architecture.md 7.6), no map is iterated, and the order of summation
does not matter because integer addition is associative (the sums are exact),
so the result is the same for any order of the rows: a test shuffles them
(FR-IND-010).

### 5.3 Index numbers

An **index** is the ratio of the value of a set of goods at the prices of this
window to its value at the prices of a **base** window, with the weights
(quantities) of one of the two, in basis points of the base.

5.3.1. **Base and set.** The base is a window the consumer names (4.3.5) and its
value is 10 000 by definition. The set `S` is a set of (region, commodity)
cells: a class (4.4), a region, or a pair; a class over the whole realm is the
union over regions. `OK` is false when the denominator is zero (an empty set,
or a set whose base value at the weights is zero), and the result then carries
no value.

5.3.2. **Weights.** A measure names its quantities; a preset states which. The
closed set `Weights`:

| Weights | Quantity of a cell in the window | Source |
|---------|----------------------------------|--------|
| `Traded` | the units traded: `SumV` | `MarketCleared.Volume` (tally) |
| `Consumed` | the units consumed by households: `consumption` plus `want_consumption` | `BoundaryFlow` |
| `Produced` | the units made: `recipe_output` | `BoundaryFlow` |
| `Basket` | the baseline upkeep of one household for one tick, `Upkeep` plus the region's `ExtraUpkeep`, times 1 | `Config` (CUR 5.1; AGT 5.2) |

`Basket` is the weight of the price level of a region (8.4), the **fixed
basket** that CUR 5 calls the reference basket `B_r`.

5.3.3. **Laspeyres and Paasche.** With `p0` the base prices, `p1` the prices of
this window, `q0` the base window's quantities and `q1` this window's, over the
cells of `S`:

```
Laspeyres_bp = rdiv(10 000 * sum(p1 * q0), sum(p0 * q0))
Paasche_bp   = rdiv(10 000 * sum(p1 * q1), sum(p0 * q1))
```

The sums are 128-bit (5.2.1) and each term is `Price * Quantity`, at most
`10^9 * 10^9 = 10^18` (`MaxPrice`, `MaxQuantity`), so a term fits `int64` and
the sum does not need to. The Laspeyres index uses fixed weights, and so reads a
price rise as the rise of the cost of the *old* basket (it overstates when
households substitute); Paasche reads it against the *new* basket (it
understates). The consumer chooses; the preset `PriceIndex` returns all three
(`Laspeyres`, `Paasche`, `Fisher`) and states which weights it used.

5.3.4. **Fisher, and its integer square root.** The Fisher index is the
geometric mean of the two rounded indices:

```
Fisher_bp = isqrt_nearest(Laspeyres_bp * Paasche_bp)
```

where `isqrt_nearest(n)` is the integer `r` that is nearest to the square root
of `n`: let `r0 = floor(sqrt(n))` (an integer Newton iteration, or a binary
search over 31 bits, on `uint64`), then `r = r0 + 1` if `n - r0 * r0 > r0` and
`r = r0` otherwise. There is **no tie to break**: `(r0 + 1/2)^2 = r0^2 + r0 +
1/4` is never an integer, so `n` is never exactly half-way, and half to even
is not needed here. The product fits: each of the two indices is at most
`MaxIndexBP` (5.3.7, `2^31 - 1`), so `n < 2^62`. The rounded inputs make the
result differ from the Fisher index of the exact ratios by at most one basis
point (each input is within half a basis point and the mean is no more
sensitive than its inputs); the model of 17 checks that the result is the
nearest integer to the square root for twenty thousand random products, and
the worked example of 18.4 computes `12 105 * 12 083 = 146 264 715`, whose
square root is 12 093.995, and returns 12 094.

5.3.5. **Chain.** A chain index links consecutive windows so that the weights
can change: with `I_k` the link index of window `k` against window `k-1`
(Laspeyres with the weights of `k-1`, or Fisher of the two), and `C_0 = 10 000`,

```
C_k = rdiv(C_{k-1} * I_k, 10 000)
```

Each link is rounded once, so the chain drifts from the exact product of its
links by at most `0.5` basis points per link multiplied by the growth of
the links after it: after `k` links the error is at most
`0.5 * sum over j of prod over i > j of (I_i / 10 000)`. For links near 1 it
is at most `0.5 k` basis points (the model: 400 random links between 0.98 and
1.03 drift by at most 4.75 basis points, far below the bound). **A direct
index against a fixed base has one rounding and no drift**; the chain is for a
long series whose weights must move, and the consumer chooses (10.4). The
product `C_{k-1} * I_k` is below `2^62` by the clamp of 5.3.7. A clamp is
sticky: once a link is clamped every later point of the chain carries
`Clamped`.

5.3.6. **Real value and the deflator.** The **real** value of a nominal total
`N_t = sum(p_t * q_t)` is its value at base prices, `R_t = sum(p_0 * q_t)`, and
the **deflator** is the Paasche-type ratio `rdiv(10 000 * N_t, R_t)` (the
implicit price index of the total). A real GDP, a real consumption and a real
wage are one call each with a base `Prices` table (7.2). Changes in volume are
read from the real series and changes in price from the deflator, and
`N_t = R_t * deflator / 10 000` up to one rounding (a test, FR-IND-013).

5.3.7. **The clamp.** An index, a ratio or a growth in basis points is held in
`BP` (`int64`), and a value of **`MaxIndexBP = 2^31 - 1 = 2 147 483 647`** or
more is **clamped to it and flagged** (`Clamped` true on the result), not
returned as an error. The reason is the one that the failure regimes give: a
runaway inflation (STB-F1) is the case in which an index is most worth reading,
and an error there would abort the measure exactly when the run is worst. An
index of 2 147 483 647 bp is 214 748 times the base, and nothing else in the
spec's arithmetic needs a number above it. The bound also keeps the products of
5.3.4 and 5.3.5 in `uint64`. A negative index cannot occur (prices and
quantities are non-negative); a growth in basis points is signed and is
clamped to `-MaxIndexBP` and `MaxIndexBP`.

5.3.8. **The stale share.** An index of prices that did not trade is the
carried-forward price (4.3.2). With every index the result reports `Stale`, the
share in basis points of the base value `sum(p0 * q)` of the cells of `S`
whose market had no volume in the window, so that an index computed mostly from
stale prices says so (STB-L03, a stale price loop). A cell with no volume is
still priced, at its carried-forward price: the alternative, to drop it,
would change the set between windows and break comparability.

### 5.4 The Gini coefficient and the other inequality measures

5.4.1. **Definition.** For `n` rows `(w_i, T_i)` of 4.5.1 with `w_i >= 1` and
`T_i >= 0`, the households of row `i` each holding `T_i / w_i`, sorted ascending
by `T_i / w_i` **by cross multiplication** (`T_i * w_j < T_j * w_i`, exact in 128
bits, so no division and no rounding decides the order; ties in any order: the
sum below does not depend on it), with `W = sum w_i`, `S = sum T_i` and the
running total `C_i = sum over j <= i of T_j` (`C_0 = 0`), the **population Gini
coefficient** of the households is

```
D = W * S
N = D - sum over i of w_i * (2 * C_{i-1} + T_i)
Gini_bp = rdiv(10 000 * N, D)
```

Every term is an integer product of exact totals, and the one division is the
last. A cohort whose total does not divide by its weight is therefore handled
exactly (the model checks it against the pairwise sum with exact fractions).

This is `1 - sum x_i (Y_{i-1} + Y_i)` of the Lorenz curve with the share of
households `x_i = w_i / W` and the cumulative share of value `Y_i = T_i / S`,
multiplied out so that no division happens before the last one. It equals the
mean absolute difference between two households over twice the mean,
`sum_i sum_j w_i w_j |T_i/w_i - T_j/w_j| / (2 W S)`, **with a row of weight `w`
counting as `w` households that each hold `T_i / w_i`**, which is what a cohort is (6.2): a row of weight
`w` and total `T` is the same as `w` rows of weight 1 each holding `T / w` (when it divides), so a measure over the
agents and a measure over the households they stand for agree (FR-IND-014;
the model checks the identity against the pairwise sum on 2 000 random row
sets). It is the *population* coefficient with no `n / (n - 1)` correction, so
its maximum for `W` households is `(W - 1) / W` (one household holds everything:
5 000 bp for two households, 9 000 for ten, 9 990 for a thousand), and a one-household
world has Gini 0.

5.4.2. **Exactness.** `D` and `N` are 128-bit; the division is an exact
unsigned 128-bit shift-and-subtract division (`10 000 * N` is below `2^127`
whenever the guard below holds, because `N <= D`), so the result is the exact rational rounded
once half to even, at most half a basis point from the true value. The guard
(5.6) is `10 000 * W * S < 2^127`: for money, `W = 10^15` and `S = 3 * 10^18`
give `3 * 10^37`, below the bound (the model prints it); for the `Wealth` column
and for ratio rows the guard can act and is `ErrOverflow` (5.2.1).

5.4.3. **When it is undefined and when it is an error.** `W = 0` or `S = 0`:
`OK` false (everyone holds nothing; the coefficient is not defined). A negative
`T_i`: `ErrNegativeValue`. The Gini coefficient is for non-negative values
(money, wealth, hunger); a welfare ratio can be negative (WEL 2.1), so the
ratio distribution is reported by quantiles, shares and means (5.5), never by
its Gini (9, `WelfareDistribution`). Clamping a ratio at zero to force a Gini
is not offered: it would hide the households in the worst position.

5.4.4. **The other measures.** From the same rows:

| Measure | Definition | Unit |
|---------|------------|------|
| `Top(q)` | the share of `S` held by the top `k = ceil(W * q / 10 000)` households (at least 1), the boundary row split by weight (`k'` of its `w` households hold `k' * T / w`): one division, `rdiv(10 000 * (w * (sum of the full rows) + k' * T), w * S)` | bp of the total |
| `Bottom(q)` | the same for the bottom `k` households | bp of the total |
| `P90` over `P10` | the ratio of the two quantiles (5.5.2), `rdiv(10 000 * P90, P10)`, `OK` false if `P10 <= 0` | bp |
| `CohortShare` | `Cohort / Households` (4.5.4) | bp |

`Top(1 000)` and `Bottom(4 000)` are the top decile and the bottom four
deciles; their ratio is the Palma ratio, which a consumer forms from the two.

### 5.5 Means, quantiles and shares

5.5.1. **Mean.** `Mean = rdiv(S, W)`, the household-weighted mean (an agent of
weight `w` counts `w` times). It is undefined for `W = 0`. IND's own rounding
(half to even, here) is authoritative for IND measures: OBS's `MeanRatioBP`
is rounded toward zero (FR-WEL-036) and is a convenience that may differ from
`WelfareDistribution.Mean` by 1 basis point. A mean of role means is weighted by
OBS's `RatioHouseholds`, never by `Households` or by agents (6.5).

5.5.2. **Quantile.** The **weighted nearest-rank quantile** `Q(q)` for `q` in
`[0, 10 000]` basis points: sort ascending by `T / w` (by cross multiplication,
5.4.1), let `k = max(1, ceil(q * W / 10 000))`, and return `rdiv(T, w)`
(half to even) of the first row whose cumulative weight is at least `k`. The
*rank* is exact; the **value** is a household's value of that row (no
interpolation: an interpolated value would be a number nobody has), and is
rounded once, to the integer, when the row's total does not divide by its
weight (a cohort of 2 holding 41 returns `rdiv(41, 2) = 20`, the even integer of
20.5), `Q(0)` is the minimum and `Q(10 000)` the
maximum. It does not depend on the order of ties: two rows of equal value give
the same result in either order.

5.5.3. **Shares.** `ShareBelow(x)` is the share of households whose value is
**strictly below** `x`, that is the rows with `T < x * w`, `rdiv(10 000 * (sum of w over those rows), W)`; a share is
computed from the exact sums and is in `[0, 10 000]`. `ShareAtOrAbove` is its
complement computed from the same sums (not `10 000 - rounded`, so the pair
need not add to 10 000 after rounding and each is within half a basis point).

### 5.6 Every overflow path

| Path | Bound that makes it safe | Outside the bound |
|------|--------------------------|-------------------|
| `Price * Quantity` of one cell | `MaxPrice * MaxQuantity = 10^18` | cannot occur for a table of a valid world; a consumer's own: `ErrRange` |
| A sum of values | 128-bit; the largest is below `2^127` (5.2.1) | a sum at `2^126`: `ErrOverflow` |
| Conversion of a total to `int64` | a money total below `3 * MaxMoney`; a quantity total below `MaxQuantity * rows` | the total does not fit: `ErrOverflow` |
| `10 000 * n / d` of a ratio | `n` below `2^127 / 10 000` (checked first, 5.2.1) | `n` above it: `ErrOverflow`; a quotient at or above `MaxIndexBP`: **clamp** to it, `Clamped` true |
| Fisher product | each factor at most `MaxIndexBP`, so below `2^62` | cannot occur after the clamp |
| Chain link `C * I` | the same | cannot occur after the clamp; a clamped link makes the chain `Clamped` |
| Gini `W * S * 10 000` | below `2^127`: always for money rows (5.4.2) | `ErrOverflow` (can act for `Wealth` and ratio rows) |
| Annualisation `x * TicksPerYear` | `TicksPerYear` at most `MaxTicksPerYear` = 100 000 | 128-bit, so cannot overflow; the result outside `int64`: `ErrOverflow` |
| A count of ticks or rows | `int` on the platform, and a window of at most `MaxTicksPerYear * 1 000` ticks by the tally's guard | `ErrWindowOrder` |

There is no wrap anywhere: every conversion from the wide type goes through one
checked function (`measure.ToInt64`), and a test asserts that no `int64`
arithmetic on a value that came from an input is unchecked (FR-IND-009 is the
requirement and a fuzz test the check).

### 5.7 Growth, and the annualisation of a total

5.7.1. **Growth** of a value from `a` to `b` is `rdiv(10 000 * (b - a), a)` in
basis points, signed, `OK` false for `a = 0`, clamped by 5.3.7.

5.7.2. **Annualisation.** A total over a window of `n` ticks is stated per year
as `rdiv(x * TicksPerYear, n)` (128-bit, half to even), which is exact up to
the one rounding, for any `TicksPerYear` and any `n`, including a window that
does not divide the year (WLD 9.3.2: a quantity per year is never converted to
an integer number of ticks). A rate per year (a birth rate, an inflation rate)
is annualised from the exact totals in one division (`rdiv(10 000 * x * TicksPerYear, base * n)`), not from a rounded window rate. A stock is not annualised.

## 6. Households, not agents

6.1. **Weights, counts and pseudo-agents.** An agent of weight `w` stands for
`w` households (SCL 2). Every population figure of this spec is a sum of
weights: `Households` is the sum over live agents, `Agents` the count of
agents, and they are separate fields everywhere (OBS 3.3.6, 5.1.2). **The
pseudo-agents** (the estates and the counterparties of the outside) are in
neither count and are never a row (FR-SCL-015, FR-OBS-007, N-013): a
`Households` row set built from `Agents(r)` has none, and a snapshot's
`Households` has none.

6.2. **Inequality within a cohort is zero by construction** (SCL 5.1,
N-013). An agent of weight `w` is `w` households that are identical in belief,
stock, money and fortune, so a cohort adds nothing to the dispersion: a measure
of inequality sees only the differences **between** agents. Three consequences
the spec states as requirements:

(a) **Every inequality figure carries the cohort coverage.** `Inequality`,
`WelfareDistribution` and every other measure over rows return `Cohort` and
`CohortShare` (5.4.4). A result with `CohortShare` above 0 says, in its own
fields, that the households of that share are held identical; a consumer that
compares two runs compares them at the same scale (SCL 7.2 is the
scale-consistency check) or says that it did not.

(b) **A mean and a count are exact at any scale**, because they are weighted
(5.5.1); a **quantile** is exact for the model as it is (the cohort's value
repeats `w` times in the order), and a **share** and a **Gini** are exact for
the model and, for a world that stands for a larger one, a *lower bound* on
the dispersion the same world would show at scale 1. The statement of this
spec is "exact for the world as built"; whether the built world is a faithful
miniature is SCL's check, not an indicator's.

(c) **A world of weight above 1 is never an error** and the measure is not
withheld: the coverage is reported and the decision is the consumer's. The
first consumer's world has weight 1 (N-030), so `Cohort` is 0 there and the
field is silent.

6.3. **Households rebuilt from events.** A consumer that keeps a tally rebuilds
the households of each (region, role) at every tick from the opening snapshot
by the rule of OBS 4.3.2: `AgentSpawned.Weight` adds, `AgentBankrupted.Weight`
subtracts, `WeightChanged` adds `To - From`, `RoleChanged` moves `Weight`
from `From` to `To`, and `AgentRelocated` moves `Weight` from `From` to `To`
(an agent on a migration journey is in its origin region until
`AgentRelocated` with the cause `migration`, OBS 4.3.2). The tally's running
count equals `Snapshot.Roles[].Households` at every boundary (FR-OBS-040;
FR-IND-011 is the same statement for the tally). It is what lets a measure over
`n` ticks use the households of *each* tick (the unfed share of 9) without a
snapshot per tick. **The denominator of a tick is the running count at its
`TickStarted`**, the previous tick's closing count: `UpkeepShortfall` is emitted
at P6.4, before the births, removals and relocations of P7 (`spec-world.md`
6.11), so a count taken after the tick's own P7 events would be the wrong
instant. It differs from the count at P6.4 only by relocations that arrive in
P1, which a convention at `TickStarted` makes the same for every tick whether
or not an `UpkeepShortfall` was emitted.

6.4. **An external agent** is a household in the counts (INT 4.1.5); `RowOpt`
filters it from a regional average where the consumer wants no player in it
(4.5.2). A tally has no such option: its counts are the world's.

6.5. **A value that does not exist.** The welfare ratio of an agent with no
upkeep is absent (WEL 2.2.3, OBS `HasMeanRatio`). A mean over such rows is
weighted by the households whose ratio exists, and the result reports the
`Absent` households (4.5.4). OBS's `RoleRow.MeanRatioBP` is already such a
mean, and OBS carries `RatioHouseholds` beside it, the households whose ratio
exists: a mean of role means is `sum(MeanRatioBP * RatioHouseholds) / sum(
RatioHouseholds)`, which a consumer forms from the snapshot with no further
ask. IND's own rounding is authoritative for IND measures (5.5.1).

## 7. The accounts

National-accounts-style measures over a window: what was produced, what it was
used for, and who held the change. The library has no firms: with the employment module off households are
self-employed (ADR 0001) and there is no wage flow, and with the fiscal, land,
credit and insurance modules off there is no tax, rent, interest or premium
either. The accounts are built from what the ledger records: goods made
and used by recipes, consumption, spoilage, losses, trade with the outside, and
the stocks; a module that is on adds flows of money between sectors (7.5.3, 7.8.2). The identities that follow are consequences of the ledger's
conservation (LED 9.1) and are asserted with exact equality, never a tolerance
(testing.md 6.3).

### 7.1 The quantities

Every term is a **window total of units** by (region, commodity), from the
tally's `BoundaryFlow` rows **corrected by the pending totals of the two
snapshots (7.1.5)**, or a **stock** from a snapshot. `r` is a region and
`c` a commodity.

| Symbol | Meaning | Source |
|--------|---------|--------|
| `Out` | units made: the output of recipes, discard outputs included | `BoundaryFlow` reason `recipe_output` |
| `In` | units used up as recipe inputs | `recipe_input` |
| `Cons` | units eaten as upkeep | `consumption` |
| `WCons` | units consumed as a want beyond upkeep | `want_consumption` |
| `Spoil` | units that spoiled | `spoilage` |
| `Disp` | units disposed of (a discard output, an overflow, a strategy's discard) | `disposal` |
| `Wear` | items retired because their durability ran out | `wear` |
| `Haul` | haul units redeemed at dispatch | `haul_use` |
| `Feed` | feed consumed by a trip | `transit_use` |
| `Loss` | units lost to a strike of the risk module (RSK): a route strike at the arrival of a shipment and a store strike at P8.1a; 0 with the module off | `loss` |
| `Offer`, `Unsold` | units the outside brought to sell and the part not bought | `outside_offer`, `outside_unsold` |
| `Purch` | units the outside bought | `outside_purchases` |
| `Init` | units created by `initial` in the window (an external agent added in the run) | `initial` at `Tick` 0 or later |
| `Held` | units held by agents, by the estate and by treasuries | `MarketRow.HeldByAgents + HeldByEstate + HeldByTreasury` (the last is 0 with no governed jurisdiction) |
| `Pend` | units of pending output of jobs in progress | `MarketRow.Pending` |
| `Transit` | units in Transit accounts attributed to the origin region | the lines of `Shipments` whose `Region` is `r` |

7.1.1. **Imports and exports in units** are `Imp = Offer - Unsold` and
`Exp = Purch` (EXT 8.2, items 2 and 3).

7.1.2. **Stocks and their changes.** `dHeld`, `dTransit`, `dPend` are the closing
snapshot's value less the opening snapshot's, by (region, commodity). The
*system* stock of a region is `Held + Transit` (a shipment's goods stay in a
Transit account of the **origin** region until they arrive, LED 3.1, 3.5);
pending output is in no account (LED 8.2.2) and is added separately.

7.1.3. **The identity of the ledger, in units.** For each commodity over the
whole realm, from LED 9.1 item 3 and 8.2.2, with every flow total as corrected in
7.1.5:

```
dHeld + dTransit  =  Out - In - Cons - WCons - Spoil - Disp - Wear - Haul - Feed
                     - Loss - Purch + Offer - Unsold + Init           (realm, per c)
```

(`dPend` does not appear: the units fixed at job starts left the held stock as
inputs and are created at completion into `Out`; pending output is a stock of
the realm's work in progress, in the production account below). A test of this
equation over a run is the "goods are conserved except by recipes and the named
flows" test of testing.md 6.3 restated with the snapshot's stocks, and the
first thing the accounts need to hold (FR-IND-016).

7.1.4. **The construction bucket is not in a window's flows.** `BoundaryFlow` of
`Tick` -1 is the endowment that exists in the **opening** snapshot of a window
that begins at tick 0 (the snapshot is taken after `NewWorld`, OBS 4.4.2), so
adding it would count the endowment twice. The tally keeps those rows apart
(`Construction`) and the accounts report them as a memo line. `initial` at
`Tick` 0 or later (an external agent added during the run, LED 7.2.1) is in the
window's flows and is the `Endow` term of 7.3.

7.1.5. **The window's flows are the tally's rows less `Open.Pending` plus
`Close.Pending`** (F1 of the review; goods and money, by (region, reason,
commodity)). A move made by a call between ticks (`AddAgent`, `Relocate`, a
slot) is in the balances at once and is totalled in the **next** tick's
`BoundaryFlow` (WLD 8.1.3; OBS 5.3.3 `Pending`: "what the next tick's
`BoundaryFlow` will include besides its own"; FR-OBS-042 states the same
correction for the boundary balances). So a window opened after such a call
holds the move in `Open` and would count it again in the first tick's flow:
`Open.Pending` is subtracted; and a call made after the last step is in `Close`
and in no flow of the window: `Close.Pending` is added. Every identity of 7 and
8 is stated with the corrected flows, and is **tested with a call before `Open`
and one after `Close`** (FR-IND-016). `Pending` is empty when no call was made,
and for a window that begins at tick 0 the construction bucket is the memo line
of 7.1.4.

### 7.2 The production approach

7.2.1. **Output, intermediate use and value added** of a region, at a price
table `p` (4.3):

```
Output_r        = sum over c of p_rc * Out_rc
Intermediate_r  = sum over c of p_rc * In_rc
Pending_r       = sum over c of p_rc * dPend_rc            (signed)
ValueAdded_r    = Output_r - Intermediate_r + Pending_r
```

`ValueAdded` is **gross domestic product by the production approach**, `GDP^P`.
The `Pending_r` term (work in progress) is the accrual: a job's inputs are
consumed when it starts and its output created when it completes
(PRO 5.3.1), so a window that begins with a long job in progress would show the
inputs of the job's start in an earlier window and its whole output in this one.
Adding `dPend` books each output as it is **fixed at the job start** (units fixed
at starts `= Out + dPend`, LED 8.2.2, net of the units dropped by an abandoned
job or a fall in weight), so the value added of a window is the value added of
the jobs **started** in it. The cash form `Output - Intermediate` is returned
beside it as `GDP^P_cash` and the preset says which it reports; a consumer with
no job longer than a tick sees no difference (`dPend` is zero), and a warm
start's staggered jobs (VIA 8.5) are the case where the accrual matters.

7.2.2. **What labour is.** Labour is not a commodity (GDS 7.3) and a recipe's
labour is the household's time, not an input: `ValueAdded` therefore contains
labour and profit together. With the employment module off the library has no
wage bill to separate them: a consumer who needs the labour share has the
observed wage (`RealWage`, 9) and the households' working ticks and forms it
itself. With the module on, what a hired household is paid is a flow of money
(`wage_pay`, EMP 7) that is not an input of the job and is not netted from
`ValueAdded`, and `WageBill` (9, row 23) reports it beside the value added
(7.5.1).

7.2.3. **By class.** `Output` and `Intermediate` are returned by commodity
class (4.4): a class's output is the value of the units of its commodities made,
and its intermediate use the value of its units used as inputs. A **class's
value added is not defined** from these two, since an input is not tied to the
output it made (a cell of the use table, not of the make table), and the
preset does not pretend: `ValueAdded` by class is the by-recipe table of 7.2.4,
or nothing.

7.2.4. **By recipe, on the completion basis.** The tally keeps `RecipeExecuted`
by (region, recipe): `Executions` and `Outputs`, `Disposed` by commodity.
For a recipe `k` the **job value added** is
`sum over c of p * (Outputs_c) - Executions * sum over c of p * Input_kc`, with
`Input_kc` the per-execution input of the recipe from `Config`, summed over
the executions that **completed** in the window. Its `Outputs` are valued gross
of `Disposed` (a discard output is produced and thrown away; its value is shown
as a loss in 7.3, not hidden). The table is grouped to a class by the output's
`CostShare` (PRO 3.2): the class of commodity `c` receives
`value(c) - CostShare_c * (input value) / 10 000` of the recipe, with the
rounding remainder of the shares going to the output with the largest share
(ties to the lower `CommodityID`). The sum of the by-recipe table (outputs of jobs
**completed**, inputs of jobs **completed**) differs from `GDP^P` (outputs
**fixed** at starts, inputs **consumed** at starts) by exactly

```
WIP = sum p * (In - InCompleted - dPend)
```

with `InCompleted = sum over recipes of Executions * Input_kc` from the table, so
the difference is the inputs of jobs started and not completed, less the change
of pending output; it is zero for a window much longer than the longest job and
in a stationary window. The preset returns it as the named row `WIP`, so the two
tables always reconcile, and the test is this formula, not "`WIP` equals
`-Pending`" (which holds only when the started and completed inputs are equal).

### 7.3 The expenditure approach

7.3.1. **Components** of a region, at the same `p`:

```
C_r        = sum p * (Cons + WCons)                      final consumption of households
StockChange= sum p * (dHeld + dTransit + dPend)          the physical change of stocks, valued
Written_r  = sum p * (Spoil + Disp + Wear + Loss)        goods that left stock unused
Carriage_r = sum p * (Haul + Feed)                       goods used in moving goods
NX_r       = sum p * (Purch - (Offer - Unsold))          exports less imports with the outside
Inter_r    = interregional net exports (8.3)             0 for a one-region world and for the realm at one price table
Endow_r    = sum p * Init                                goods created by `initial` in the window
```

and **gross domestic product by the expenditure approach** is

```
GDP^E_r = C_r + StockChange_r + Written_r + Carriage_r + NX_r + Inter_r - Endow_r
```

7.3.2. **Names, and what each is for.** `StockChange + Written` is the **gross
accumulation** of goods before spoilage and retirement: the units made and not
used for consumption, carriage, trade or loss. It is the "investment" of a
national account and the library does not call it that, because it holds
inventories and tools together and has no firm that invests (SAV and CAP are
later specs); a consumer that wants the **durable** part reads `Out` of the
commodities of a class (a `tool` class is a consumer's tag, 4.4). `Written` is
what a national account calls other changes in volume: stock that left the
accounts without being consumed.

7.3.3. **Imports and exports at two values.** `NX_r` values the units of trade
at the price table, so that the identity closes and the measure is of **volume
at a price**. The money actually paid is in the boundary balances of the six
`outside_*` reasons (`export_value = outside_budget - outside_unspent`,
`import_value = outside_receipts`, EXT 8.2 item 3), and `TradeBalance` (9)
reports both. They differ by the price improvement of the midpoint rule (EXT
6.7: a counterparty trades at the midpoint of its limit and the realm's), which
is a transfer between the realm and the outside and not production.

### 7.4 The identity

7.4.1. **Statement.** Three statements, in order of strength, over every
window of `n >= 1` steps, at any price table `p`:

(a) **The realm, at one price table for every region** (`p_rc = p_c`):

```
GDP^P  =  C + StockChange + Written + Carriage + NX - Endow          exactly, in minor units
```

with no interregional, valuation or relocation term: a unit that moves between
regions or with a relocated account leaves one region's stock and joins
another's, and at one price the two cancel (FR-IND-017).

(b) **A region**, at any table:

```
GDP^P_r  =  GDP^E_r - Reloc_r,     Reloc_r = sum p_rc * (RelocIn_rc - RelocOut_rc)
```

where `GDP^E_r` is 7.3.1 with `Inter_r` built from the shipments of the window
(8.3) and `Reloc_r` is the **value of the goods that moved with relocated
accounts**, the reconciling item of REG 6.4. `Reloc_r` is zero for a world with
no migration module, no self-carried trip and no INT relocation; where it is
not zero it is reported as the residual by commodity and **checked in units in
total** against the `Units` of `AgentRelocated` (8.2). It is the one term that
the events do not give by commodity.

(c) **The realm at regional prices:** (a) with `TradeValuation` and
`RelocValuation` added (7.4.2). `RelocValuation` is the value that the goods
which moved with relocated accounts have at their destination's prices over their
origin's, and the events give those goods in total units only: it is reported
**as a residual by value**, flagged `Residual`, with the check of the units in
total (8.2). The statement is exact in a world with no relocation (no migration
module, no self-carried trip, no INT relocation), and in a world with one it
is the identity with that one flagged residual, not a claim of equality.

All three are the ledger's conservation of units (7.1.3) multiplied by prices
and summed, so none needs a model: a **test over a run** (FR-IND-016,
FR-IND-017, FR-IND-019) asserts them for every window, every seed and every pack
of the reference chain, with and without every module.

7.4.2. **At regional prices.** With a price table that differs by region the realm
sum `sum over r of Inter_r` is not zero: a unit that leaves `alpha` at `p_alpha`
and arrives in `beta` at `p_beta` is worth `p_beta - p_alpha` more at its
destination. The realm identity is then

```
GDP^P = C + StockChange + Written + Carriage + NX - Endow + TradeValuation
TradeValuation = sum over shipments of (p_origin - p_destination) * (Delivered + Diverted)
```

(a negative number when goods go to dearer markets). `TradeValuation` is a
**measure of the price gap that the realm's trade closes**, not a production
and not a loss; the worked example of 18.2 has it at -27, the gap of 3 on 9
units. A consumer who wants a single realm figure uses a one-vector table
(`PricesOfWindow` of the realm's mean unit values, or the base table), and a
consumer who wants regional figures at regional prices reports `TradeValuation`
beside the realm's GDP.

7.4.3. **Why the two can differ in principle.** One side comes from the
**flows** of events (`recipe_output`, `consumption`, ...), the other from the
**stocks** of two snapshots. They are independent records of the same ledger;
if they disagree one of them is wrong (a missing event, a snapshot field that
is not a function of the saved state, a leak in a module). That is the value of
stating the identity: it is the check that no module creates or destroys goods
off the books, and it holds in the audit's own terms (`World.Audit`, LED 12.4).

### 7.5 The income view

7.5.1. **What the library has.** With the employment module on, a hired
household's wage is a flow of money: the internal reason `wage_pay`, from the
Escrow account of the hire to the worker (EMP 7), totalled in a tick as
`LabourRound.Paid`. It is no boundary row and is in no term of 7.1 to 7.4, since
it moves money between sectors and makes no goods; `WageBill` (9, row 23) reports
it, and the labour share is the wage bill over the value added of 7.2.1. What
the library has of the income view proper is the **household's own booked
income** `income_t` of WEL 2.1:

```
income_t = R_t + K_t + K^own_t + F_t + G_t - S_t(Input) - W_t
```

per agent per tick, in the welfare ring (saved state). The terms are WEL 2.1's:
a module that adds a term of income (wages, interest, rent, a payout) changes
WEL's formula and not this view, which reads their sum. Summed over the live
agents, weighted by their households, it is the **household income** of a region
and role for a tick, and the sum over the ticks of a window is the window's.
OBS's `RoleRow` does not carry it (it carries a mean ratio). **The fields
(`IncomeLast`, `ReliefLast`, `CostLast`, functions of the saved ring) are not
adopted for version 1**, and the wear event of 7.6.2 is not either: they remain
a **future ask** for OBS (22). Until then the income view is **not available**,
`HouseholdIncome` returns `OK` false in version 1, and the production and
expenditure views do not need it.

7.5.2. **The income view is a diagnostic, not a third equal.** `income_t` is a
cash flow at the **traded** prices of the agent's own trades, nets inputs when
they are **bought** (not when used), books tools at their wear, and includes
transfers (relief, grants, in-kind, own-use receipts). `GDP^P` is a flow of
**units** at a **table** price, with inputs netted when **used**. So

```
HouseholdIncome - GDP^P  =  Transfers                                relief, grants, in-kind: income, not product
                          + (input purchases - input use) at purchase prices      cash against accrual
                          + (traded - table price) * units traded     valuation
                          + (own-use receipts - own-use output at table)   SUB 7
                          - (wear booked - wear retired)             the tool convention
```

The terms after `Transfers` are not separately observable from the events (a
bid's purpose is not an event field, OBS 4.4.7), so the view returns
`HouseholdIncome`, the observable `Transfers` (relief from `ReliefPaid`, grants
from the boundary rows of the money flows that are not `NotIncome`), and
`Gap = HouseholdIncome - Transfers - GDP^P` as a **single named residual**. It is
not claimed to be zero and no test asserts it zero; the claim that does hold
is in a **stationary** window (constant stocks and prices, `dHeld = dPend = 0`
and every trade at the table price), where the three timing and valuation terms
are zero and the residual is the tool convention alone. The worked example of 18.2
does not include it; the stationary one-tick world of 18.9 is the test once the
fields exist (until then the first test is that the preset returns `OK` false).

7.5.3. **The money identity is exact and is the income side's equality.** In
money, by sector, over a window and for the realm (LED 9.3):

```
dM_h + dM_p + dM_c + dM_t  =  sum of money faucets - sum of money sinks
```

with `M_h`, `M_p`, `M_c` the money of households, of the estate pools and of
Escrow accounts (contracts and hires), and `M_t` the money of the treasuries of
governed jurisdictions (POL 10.5; 0 with none) (OBS 5.3.2; the outside holds
nothing at a boundary), and the
faucets and sinks the `BoundaryFlow` rows of kind faucet or sink with
`Commodity` `NoCommodity`. The relation to **goods** is the balance sheet of
LED 9.2: valued at one table `p`, the net worth of the sectors (the three of
LED 9.2 and the treasuries)

```
NW_h = M_h + sum p * HeldByAgents + sum p * Pend           (households, with work in progress)
NW_p = M_p + sum p * HeldByEstate                          (public: the pools and the estates' goods)
NW_c = M_c + sum p * Transit                               (contract: escrow and cargo)
NW_t = M_t + sum p * HeldByTreasury                        (treasuries: money and goods bought or held)
```

changes, in total, by exactly the **value of the boundary flows and of the
change of work in progress**:
`dNW_h + dNW_p + dNW_c + dNW_t = (money faucets - sinks) + sum p * (goods faucets -
sinks) + sum p * dPend`, the last term because work in progress is in `NW_h`
and in no flow, which is `(money faucets - sinks) + GDP^P - C - Written -
Carriage - Exp + Imp + Endow` with `GDP^P` in its accrual form (FR-IND-018). The
flows are corrected by 7.1.5. That is the stock-flow consistency of LED 9.3
in value, and it **is** the saving-and-investment identity: gross saving of the
realm is `GDP - C`, and it is held as the change of the sectors' net
worth plus what leaves as `Written`, `Carriage` and trade. A wage, a tax, a rent, an
interest payment or a premium moves money from one of these sectors to another
and makes no term of the equation: each is an internal reason of its module
(`spec-employment.md`, `spec-fiscal.md`, `spec-land.md`, `spec-credit.md`,
`spec-insurance.md`). The preset `Sectors` (9) returns `NW` and `dNW` per sector
and the closing of this equation, and the money moved by each such reason (7.8.2).

### 7.6 Depreciation, and net product

7.6.1. **What the library records.** An item of a durable that wears to nothing
is retired to the `wear` boundary (LED 5.4.1) and counted in `Wear`, one unit
per item. The **wear units** actually applied (durability removed) are counted
by the ledger in its flow totals (LED 5.4.2, FR-LED-016) and are **not** in an
event or a snapshot. The two are the same flow booked at two times: the item at
its end, the units as they happen.

7.6.2. **Decision.** The version-1 depreciation is **the retired items valued at
the table price** (`Written` includes it as `Wear`), a **retirement basis**: its
sum over an item's life is the item's price, and its timing is lumpy (an item
bought in one window may be retired in another). `NetProduct = GDP^P - sum p *
Wear` is returned with the flag `Basis: retirement`. An **accrual** basis, `sum
over c of WearUnits_c * p_c / Durability_c` rounded half to even once, needs the
wear units by commodity: **not adopted for version 1** (an aggregate event of the
wear units, or a field of `BoundaryFlow`, stays a future ask in 22), so
depreciation is retirement-basis only. Until then `HeldByAgents` values a worn durable
**as new** (the snapshot has no durability, 4.5.5), so a stock of tools is
overstated by its accumulated wear and `StockChange` is correct only because the
item's retirement is booked in `Written`: the **sum** of the two is right, the
split is not. This is stated in the result (`Basis`) and in the failure each
preset must not hide (9).

### 7.7 Real values and the closed economy

7.7.1. **Real GDP** is `GDP^P` at the base table (5.3.6), and its change over a
series is a **volume** change; the deflator is the Paasche index of the whole
output. The three views are valued at the same table, so a "real" account is
one call with the base `Prices`.

7.7.2. **A closed economy.** With no outside module and no money faucet or
sink, `NX = 0`, `Endow = 0` unless an external agent was added, and the money
identity of 7.5.3 reads `dM_h + dM_p + dM_c + dM_t = 0` (MON 8.2). The mean welfare
ratio is then **exactly 1 only if households buy only upkeep and every
household's income equals its upkeep cost**; the exact statement the library
makes is the household-weighted mean of the ratio over agents (WEL 2.1), which
`WelfareDistribution` reports as `Mean`, and the **aggregate ratio**
`sum income / sum cost` over the same agents and window, which `HouseholdIncome`
reports as its aggregate ratio (`OK` false until the fields of 7.5.1 exist), so that a test of "1 with upkeep only and
`1 + W/U` with wants" (VIA 4.7, DEM 7.1, N-054) is stated on the aggregate, the
quantity for which it was derived, and not on the mean of per-agent ratios,
which differs from it by the covariance of the ratio with the cost (the
mean of ratios is not the ratio of means; the worked example of 18.9 shows a
case).

### 7.8 What the accounts do not measure

7.8.1. **The trade margin and the service of a merchant.** A merchant has no
recipe (MER 3.3): it earns the margin between two prices, and the library
books no output for it. The haul units it redeems are production (a carrier's
recipe) and appear as `Carriage`. Goods moved by a trip are valued at the
destination's price on arrival by the table, which is the `TradeValuation` of
7.4.2. A consumer who wants distribution as a sector computes `TradeValuation +
freight` itself.

7.8.2. **Taxes, subsidies, rent, interest and premiums.** With the fiscal, land,
credit and insurance modules on each exists, as an internal transfer of money
between two accounts under its own reason (a levy or a payout between a
household or a pool and a treasury, a rent, an interest payment, a premium or a
payout). None is a boundary row and none is output, so the accounts have no row
for them and the identities of 7.4 and 7.5.3 stay as they are. They are read in
two places: the aggregate events of the modules, which the tally keeps where a
preset needs them (4.2.2), and the money of each sector in `Sectors` (9, row 7),
with the regional terms of 8.2 for the fiscal flows. A tax is not value added and
no preset adds one to `GDP`; a consumer that wants a figure at market prices
forms it from those rows and says so. The net worths of 7.5.3 hold money and
goods only: a claim is a row of the snapshot (`Claims`) and a consumer values it
itself.

7.8.3. **Household production for own use** is production: an own-use recipe's
output is `recipe_output` and is in `Out`, valued at the table price of the
household's own market (SUB 7). It is **not** marked in `GDP`; a consumer that
wants the non-market share lists the own-use recipes (`Config.Subsistence`) and
reads the `RecipeExecuted` rows of the tally for them (7.2.4).

7.8.4. **The estate's goods.** `HeldByEstate` is public-sector stock; whether
its value is part of "wealth" is the consumer's: the sector table keeps it
apart (`NW_p`).

## 8. By region and for the realm

8.1. **Attribution.** A flow is attributed to the region of the non-boundary
account it debits or credits (LED 3.5, 7.4): a recipe's output to the region of
the agent that made it, a consumption to the region of its household, a spoilage
to the region of the account that held the lot (a Transit account's region is
its origin). The tally keys by that region, so the **regional GDP is the value
added of what was made in the region**, wherever it is later consumed.

8.2. **The regional identity and its two reconciling items.** For a region `r`
and a commodity `c`, with the system stock `Held + Transit` (7.1.2):

```
d(Held + Transit)_rc = Out - In - Cons - WCons - Spoil - Disp - Wear - Haul - Feed - Loss
                       - Purch + Offer - Unsold + Init
                       + ArrIn_rc - ArrOut_rc            interregional trade (8.3)
                       + RelocIn_rc - RelocOut_rc        relocation (this section)
```

**Money**, by region: `d(M_h + M_p + M_c + M_t)_r = faucets_r - sinks_r +
RelocMoneyIn_r - RelocMoneyOut_r + SettledIn_r - SettledOut_r + FiscalIn_r -
FiscalOut_r`, where the relocation terms are the `Money` of `AgentRelocated` by
`From` and `To` (a household that relocates takes its money with it), and the
contract terms are the `Paid` of `ContractSettled` (money leaves the Escrow
account of the **destination** region and reaches the deliverer, in the origin
region, which is the event's own `Region`). `M_t` is the money of the treasuries
seated in `r` (`RegionRow.MoneyTreasury`, POL 10.5), a treasury's account being
in the region of its seat (POL 5.3.4). A tax paid by the households of one
region into a treasury seated in another, and a payout from a treasury to the
households of another, is a transfer between regions that none of the other
terms names: `FiscalOut_r` is the money (the `Applied` of the money rows of
`FiscalFlowApplied`) that leaves `r` for the seat of an authority outside it, a
payout counting as `FiscalOut` of the seat and `FiscalIn` of the recipient's
region, and `FiscalIn_r` is the money that comes in, the seat of each authority
read from the `Jurisdictions` rows of the snapshot (FIS 4.3.2, 16.1.3). Where no
authority is seated away from its payers and recipients both terms are 0, and a
world with no fiscal module has none. The flows are corrected by 7.1.5. **Every
term is observable**, so the
regional money identity is **exact** (FR-IND-019). In **goods**,
`AgentRelocated` carries only the **total units** moved (`Units`), not units by
commodity: the per-commodity relocation terms are the **reconciling item** of
REG 6.4, and the regional goods identity holds **in total**
(`sum over c of (RelocIn_rc - RelocOut_rc) = Units in - Units out` of the region)
and per commodity up to that item, which the preset reports as
`RelocatedUnits` (the residual by commodity) with the check of the totals.
Reporting the goods by commodity needs the `AgentRelocated` event to carry its
lines; the ask (optional, of event volume class E) is in 22 and the
recommendation is **not now**: a relocation moves the holdings of a household,
and the identity by commodity is a refinement of a measure no scenario needs.

8.3. **Interregional trade.** `ShipmentArrived` gives, by (origin, destination,
commodity), the units `Shipped`, `Delivered`, `Diverted` and `Spoiled`
(`Shipped = Delivered + Diverted + Spoiled`). The units that leave the origin's
system and join the destination's are `ArrOut = ArrIn = Delivered +
Diverted`; the spoiled units are a `spoilage` flow of the origin's Transit
account (a boundary row, attributed to the origin). Net exports of a region at
`p` are `Inter_r = sum over lanes and c of p_rc * ArrOut_rc - sum of p_rc *
ArrIn_rc` (a **volume at the region's own price**); a cargo still in flight at
the window's end is `dTransit` (a stock of its origin), not an export yet.
**Interregional payments** are `ContractSettled.Paid` between regions (8.2); a
self-carried trip is a relocation of its traveller at arrival (OBS 4.3.2) and
shows in the relocation terms.

8.4. **The price level of goods that travel, and the delivered price.**
N-013 asks that a price index of goods that travel say whether it is the local
price or the delivered one (TRN-A4). Both are offered and the result says which.

(a) **The local price** of a commodity in a region is its published price there
(4.3.2). A **regional price level** of a set of goods is a fixed-basket index
(5.3.3) with the `Basket` weights, `BasketLevel_r = rdiv(10 000 * sum p_rc *
q_c, Bcost_r)` where **`Bcost_r` is the computed cost of the baseline basket at
the region's effective reference prices** (`sum q_c * ref_rc`), so a level is
10 000 **exactly** at reference prices. The configured field `B_r` of CUR 5.1
is a different number within `BasketTolerance` of it (CUR 5.3: validation only
checks that the two agree within the tolerance, default 10%), so a level against
`B_r` could differ by up to that tolerance with the author's choice; it is not
the denominator, and a consumer who wants it divides by `B_r` itself. The
**ratio of two regions' levels** is the price level of one against the other,
computed from the exact sums in one division.

(b) **The delivered price** of a commodity on a lane (origin `o`, destination
`d`) is the origin's price plus the **freight per unit**:

```
Freight(o -> d, m) = rdiv(HaulValue, Units)                  per unit of cargo, hired mode m
HaulValue          = sum over the window's dispatches on (o, d, m) of
                     HaulUnits * p(origin, haul commodity of m)
Units              = sum over those dispatches of the units of their lines
Delivered_c(o -> d, m) = p_oc + Freight(o -> d, m)
```

with `HaulUnits` and the lines from `ShipmentDispatched` (so the numerator and
the denominator are of the same events and the same tick), the haul commodity
from the mode of `Config` (`Mode.Haul`, TRN 3.1) priced at the origin, and one
division. **The freight is per unit of cargo, whatever the commodity**: this is
a stated **convention**, exact for a single-line shipment. Transport buys
carrying units by the binding fill of mass and bulk (TRN 3.2.1; it does not use
the chargeable size, N-011), which cannot be apportioned to a line from the
events; a per-line carrying-unit figure on `ShipmentDispatched` would make it
exact and is routed to TRN and OBS (22). A **self-carried** lane has
no priced freight: its cost is the traveller's upkeep, wear and time, which are
not a price, and the lane is reported with `Hired` false and `Freight` 0. The
delivered price is what a unit costs **at the destination before the
destination's margin**, so that the comparison "is the gap in price larger than
the freight" (the arbitrage condition TRN-A4 and the merchant of MER 5 use) is
`p_dc - Delivered_c`, the `GapAfterFreight` of `PriceGap` (9).

(c) **Which a series should use.** For the *level* of what households pay: the
local price (that is what they pay where they are). For the *integration of two
markets*: the gap after freight (the law of one price holds when it is within a
band of zero). The delivered price is not a table (4.3.6) because it varies by
lane, and a consumer who wants a delivered price index forms it from the lanes
with its own weights and says so.

8.5. **The realm.** `GDP^P` of the realm is the sum of the regions' (7.4.1);
the money of the realm is the sum of the sectors; households are the sum of the
regions' (and a migrant on a journey is in its origin, 6.3). A realm figure of a
**price** is not a sum: it is an index over the cells of all regions (5.3.3),
with the weights of the realm, or a mean of regional levels weighted by
households, and the preset says which. An outside node has no row: its trade is
in the gate region's `NX`.

8.6. **A region that is cut off.** A closed edge (REG 7) changes no accounting
term: the shipments and settlements that did not happen are zero rows. The
**price gap** between the two ends rises (`PriceGap`) and the **households**
fall or not (`Households`), which is what S28 reads (STB 8).

8.7. **A regional difference is not real until the caps are checked.**
`spec-world.md` 10.2 says that a cap that binds picks who is served by an ID or
a region order, and "a statistic that groups by region or role would show it as
a difference between regions that the data did not put there. IND and STY must
not read a regional difference as real before checking whether a cap or a limit
was binding there". The tally keeps the **binding counters** of the region for all five places
of WLD 10.2 (4.2.2): O1 the entry budget and births (`EntryLimited.Unfunded`,
`PopulationChanged.BirthsCapped`, `BirthsUnfunded`); O2 the role-change and
entry caps and the admission of movers (`EntryLimited` caps, the
`MigrationReviewed` caps `OutCap`, `InCap`, `Unprovisioned`, `Bound`, `Held`);
O3 the ladder (`EntryLimited.ExitCap`); O4 a flow's total limit
(`MoneyFlowLimited`); O5 the contract round (`ContractRound.TooMany`,
`PayerFunds`). Every measure
that **compares regions or roles** (`PriceGap`, `Households` by region,
`RoleShares`, `Unfed` by region, `WelfareDistribution` by region,
`TradeBalance` by gate) carries the warning `Binding` with the counters of the
regions it compares when any of them is non-zero in the window. The warning
says "check before reading"; it does not withhold the measure.

## 9. The standard presets

A **preset** is a named, fixed call of the aggregators (3.6) with a stated
definition, input and unit. The list is closed for version 1: a later module's
request for a preset is an ask of the module's spec (22), not a row, and the
four rows 22 to 25 are the presets of the employment and risk modules that this
spec accepted. Each row is a requirement on the preset (FR-IND-030 to FR-IND-050,
and FR-IND-056 to FR-IND-059 for rows 22 to 25). A consumer adds its own the same
way (3.6); a preset is never *the* measure, only the one every consumer,
scenario and band means by its name.

### 9.1 The list

Columns: **Definition**, **Result fields** (the exported fields of the result,
in the spelling that STY, CLB and STB use for a measure, `Pinned.Share`,
`Turnover.Volume`; a field a band or a knob row names is in this column, and the
names are provisional until the code, 3.6), **Inputs** (3.3), **Unit**, **Cost** (3.4), **Must not
hide** (the failure a naive version of the measure would conceal, which the
result carries as a field), **First test** (a hand computation of 18).

| # | Preset | Definition | Result fields | Inputs | Unit | Cost | Must not hide | First test |
|---|--------|------------|---------------|--------|------|------|---------------|------------|
| 1 | `GDP` | the three views and the identity of 7: production (`ValueAdded`, accrual, with cash beside it), expenditure by component, the residual of the identity (which is zero) and, by region, `Inter`, `Reloc`, `Endow`; realm and regions | `ValueAdded`, `Cash`, `C`, `StockChange`, `Written`, `Carriage`, `NX`, `Inter`, `Reloc`, `Endow`, `TradeValuation`, `NetProduct`, `Basis`, `Residual`, `Construction` | `Open`, `Close`, tally, `Prices` | minor units | 2 | work in progress (`Pending`) shown apart from cash; discards in `Written`, not in `C`; `Basis` of depreciation; the construction bucket as a memo line | 18.2 |
| 2 | `ValueAdded` | output and intermediate use by commodity class, and value added by recipe on the completion basis with grouping by cost share and the `WIP` reconciling row | `Output`, `Intermediate`, `ByRecipe`, `ByClass`, `WIP` | tally (`RecipeExecuted`, `BoundaryFlow`), `Config`, `Prices`, `Classes` | minor units | 2 | that a class has no value added of its own (7.2.3); the `WIP` row; disposed output counted in gross output | 18.3 |
| 3 | `PriceIndex` | `Laspeyres`, `Paasche`, `Fisher` of a set (a class, a region, a pair, all) against a base window, with the weights named, and the chain (5.3) | `Laspeyres`, `Paasche`, `Fisher`, `Chain`, `Weights`, `Stale`, `Clamped` | a `Base` (prices and quantities of the base window, 3.3), the current `Prices` and quantities (tally), `Classes` | bp of the base | 2 | `Stale` share of the set (prices carried forward); `Clamped`; which weights; that a class index and the all-goods index are not sums of each other | 18.4 |
| 4 | `BasketLevel` | the cost of the baseline basket of a region at its published prices over its **computed** cost at its reference prices, `rdiv(10 000 * sum p * q, Bcost_r)` (8.4: not the configured `B_r`); and the level of one region against another from exact sums | `Level`, `Ratio`, `Stale` | `Snapshot` (or window `Prices`), `Config` (upkeep, `ExtraUpkeep`, reference prices) | bp (10 000 is the calibration) | 1 | the basket is **fixed**: it does not follow what households actually buy; a good with no trade is priced stale (`Stale`) | 18.5 |
| 5 | `DeliveredPrice` | per lane (origin, destination, commodity): the origin's window price plus the freight per unit of cargo of the lane's hired mode (8.4), with `Hired` and `Freight` | `Delivered`, `Freight`, `Hired`, `HaulValue`, `Units` | tally (`ShipmentDispatched`, `ShipmentArrived`), `Config` (modes), `Prices` | minor units per unit | 2 | a self-carried lane has no priced freight (`Hired` false, `Freight` 0): the lane's cost is not in the price | 18.5 |
| 6 | `PriceGap` | per connected pair and commodity: the absolute difference of the two prices as a share of the pair's mean, `Spread`; and the gap after freight `p_d - Delivered`, signed, as a share of the delivered price; and whether the pair is in one component | `Spread`, `GapLocal`, `GapAfterFreight`, `Connected` | `Prices`, `DeliveredPrice`, `Graph` components | bp | 2 | a gap in a pair with no route (a closed road) is not a failure of arbitrage; `Connected` is a field, and an unconnected pair is excluded from any average | 18.5 |
| 7 | `Sectors` | money by sector and per household (`M_h`, `M_p`, `M_c`, `M_t`, `M`, `M_h / H`), by region and in total, `M_t` being the money of the treasuries of governed jurisdictions (POL 10.5) with the fiscal flows reported beside it (`FiscalIn`, `FiscalOut` of 8.2 and the `Applied` of each fiscal flow by kind); net worth by sector (a treasury's with the goods it holds) and its change at a price table; the closing of the identity of 7.5.3 | `M_h`, `M_p`, `M_c`, `M_t`, `M`, `PerHousehold`, `NW`, `dNW`, `Closing`, `Boundary`, `FiscalIn`, `FiscalOut` | `Close` (and `Open`, and the tally for the closing and for `FiscalFlowApplied`), `Prices` | minor units; minor units per household (by region and total; **by role** only through household rows of `Money`) | 1 | money held out of circulation (pool, escrow, treasury) is its own row, not "the economy"; a tax moves money from households to a treasury and is no loss of money; the boundary balance of each reason (how much entered and left); the net worth of the estate is public | 18.6 |
| 8 | `Turnover` | the value of trade `sum(Price * Volume)` and the volume by market, region and class; the **velocity** `V = turnover / M` of the window, and annualised from the exact sums, `rdiv(10 000 * turnover * TicksPerYear, M * n)` | `Value`, `Volume`, `Velocity`, `Annualised`, `Active` | tally (`MarketCleared`), `Close.Money` | minor units; units; bp (and per year) | 2 | a thin market (few ticks with volume) is flagged (`Active` ticks); a velocity with a falling `M_h` and a rising pool is a different story than a falling turnover | 18.6 |
| 9 | `Households` | households and agents by region and role, their shares, and the **account of change**: spawned, bankrupted, weight changes, role changes, relocations, births and deaths done, over the window; the natural increase and its annual rate | `Households`, `Agents`, `Spawned`, `Bankrupted`, `WeightChanges`, `RoleChanges`, `Relocated`, `Births`, `Deaths`, `Rate`, `Residual` | `Open`, `Close`, tally | households; bp | 1 or 2 | agents are not households; a migrant on a journey is in its origin; the account closes exactly (`Residual` 0); an estate is in neither | 18.7 |
| 10 | `WelfareDistribution` | the distribution of the welfare ratio over households: `Mean`, `P10`, `P25`, `P50`, `P75`, `P90`, the share below 10 000 (a ratio under 1), the share below the role's target, and the same for the earned ratio | `Mean`, `P10`, `P25`, `P50`, `P75`, `P90`, `Below`, `BelowTarget`, `Earned`, `Absent`, `Cohort` | household rows of `WelfareRatio` and `EarnedRatio` | bp | 3 | negative ratios exist (no Gini); `Absent` households (no upkeep); `Cohort` coverage; a mean of ratios is not the aggregate ratio (7.7.2) | 18.8 |
| 11 | `Inequality` | the Gini of money (and of wealth with a table), `Top(1 000)`, `Bottom(4 000)`, `P90 / P10`, `CohortShare` | `Gini`, `Top`, `Bottom`, `P90P10`, `CohortShare` | household rows of `Money` (or `Wealth`), `Prices` | bp | 3 | a cohort is identical by construction (`CohortShare`); money is not wealth; the estate and pool are outside the households | 18.8 |
| 12 | `Unfed` | the share of households unfed in the last consumption (at a boundary) and the share of **household-ticks** unfed over a window, by region and role, and the largest `MaxHunger` | `Share`, `HouseholdTicks`, `MaxHunger` | `Close.Roles[].Unfed`, `Households`; or tally (`UpkeepShortfall` and the running households at each `TickStarted`, 6.3) | bp | 1 or 2 | households, not agents (an agent of weight 6 unfed is 6); `UpkeepShortfall` is not emitted when all are fed, so the denominator is the running count at the tick's start, not the events | 18.7 |
| 13 | `TradeBalance` | by node, gate and commodity: units imported and exported (`Imp`, `Exp`), the money value of each (`import_value`, `export_value`), the net, and the volume at a table; and the openness `(X + M) / GDP` | `Imp`, `Exp`, `ImportValue`, `ExportValue`, `Net`, `Volume`, `Openness` | tally (`OutsideTraded`, `BoundaryFlow` of the six reasons), `Prices`, `GDP` | units; minor units; bp | 2 | the outside's boundary flows are **gross** (budget, unspent), the net is derived (EXT 8.2, N-115); price improvement is a transfer, not output | 18.8 |
| 14 | `RealWage` | per (region, role): the observed wage per working household-tick over the cost of the role's upkeep at the region's price table, `rdiv(10 000 * wage, upkeep cost)`: how many ticks of upkeep a working tick buys | `Ratio`, `HasWage` | `Close.Roles[]` (`WageObserved`, `HasWage`), `Config` (effective upkeep), `Prices` | bp (10 000 is one tick of upkeep) | 1 | a role with no observed wage (`HasWage` false) is absent, not zero; wage is per working tick, and a role works less than every tick (it is not an income) | 18.8 |
| 15 | `StockCover` | units held (agents, estate and treasuries) over the units consumed per tick, per (region, commodity) or class: ticks of cover; and the unsold share `HeldByEstate / Held` | `Cover`, `Unsold` | `Close.Markets[]`, tally (`Cons`, `WCons`) | bp of one tick (10 000 is one tick of cover) | 1 or 2 | a glut (cover rising without bound, F2, F4) and an empty pantry (cover 0) are different failures; a commodity with no consumption has cover undefined (`OK` false), not infinite | 18.6 |
| 16 | `RoleShares` | households by role as a share of the region's (and the realm's), per window start and end, and the swing: `max - min` of the share over a series | `Share`, `Swing` | `Close.Roles[]` or the tally's running counts; a series | bp | 1; W | the swing of a role's share is the cobweb (F6, L36); a role that disappears has share 0 and must be reported, not dropped | 18.7 |
| 17 | `PriceRange` | per market and class: the lowest and the highest published price in the window, `High * 10 000 / Low`, and over a series of windows whether the range shrinks | `Low`, `High`, `Range`, `Active` | tally (`MarketCleared`), a series | bp (10 000: no variation) | 2; W | a market with no trade in the window has range 10 000 only because its price is carried (`Active` 0); the `Low` is a published price, not a limit | 18.5 |
| 18 | `Pinned` | the share of belief outcomes that ended on the floor or the ceiling of the reference band: `(AtFloor + AtCeiling) / Outcomes` per market, and the markets with unmatched units on both sides and no volume | `Share`, `Outcomes`, `AtFloor`, `AtCeiling`, `Stuck` | tally (`BeliefsUpdated`, `MarketCleared`) | bp | 2 | a pinned price is not a stable price (F5); `Outcomes` 0 means undefined, not zero | 18.5 |
| 19 | `Dispersion` | the spread of belief centres: `(HighCentre - LowCentre) / MeanCentre` per market, in the last tick of the window and the window's lowest | `Spread`, `LowCentre`, `HighCentre`, `MeanCentre`, `Lowest` | tally (`BeliefsUpdated`) | bp | 2 | synchronisation (F7) shows as a spread that falls below a floor; `MeanCentre` 0 is undefined | 18.5 |
| 20 | `Offset` | the difference between a market's window price and a reference price supplied by the caller (the steady-state price of VIA, a base table), `rdiv(10 000 * (p - ref), ref)`, signed, with `Unmet` need: the share of the household-ticks of the window that were unfed (the `Unfed` share of row 12, from `UpkeepShortfall`), in basis points | `Diff` (signed; the `Offset.abs` of CLB is its absolute value), `Unmet` | `Prices`, a `Prices` of the reference, tally | bp, signed | 2 | the offset (F8) is meaningful only against a reference computed for the same pack; `Unmet` is reported beside it | 18.5 |
| 21 | `HouseholdIncome` | the booked income of households (income, relief, cost), the aggregate ratio `sum income / sum cost`, and the `Gap` of 7.5.2; **`OK` false in version 1**: the fields are a future ask (7.5.1, 22) | `Income`, `Relief`, `Cost`, `Ratio`, `Gap` | per-tick snapshots after each step (`RoleRow.IncomeLast`, ... once they exist) | minor units; bp | 1 per tick | that it is cash at traded prices with inputs netted at purchase (7.5.2); that it is not GDP; the aggregate ratio is the quantity N-054 is about | 18.9 (that it returns `OK` false in version 1; the stationary world once the fields exist) |
| 22 | `Unemployment` | the share of the households that offered their labour for hire in a window and were not matched: `rdiv(10 000 * NoVacancy, Offered)` of the summed household-ticks (EMP 10.1.1), by region and worker role, with `Offered` and `Hired` beside it and the refusals of formation by reason (EMP 10.1.3) | `Rate`, `Offered`, `Hired`, `NoVacancy`, `ByReason` | tally (`LabourRound`), `Config` (`Role.Hireable`) | bp; households | 2 | households, not agents; a household that rested, was in the own-use mode, was a trainee, away or on a journey offered nothing and is not counted; a role with no offered household-tick has the rate undefined (`OK` false), not 0; the rate does not say whether the unemployment is structural or frictional, and the reasons are returned beside it | 18.11 |
| 23 | `WageBill` | the money paid to hired households in a window, the sum of `LabourRound.Paid` by region of the hire, with the `Refunded` of cut starts beside it, and the labour share `rdiv(10 000 * Paid, ValueAdded)` against the accrual form of 7.2.1 and against the cash form | `Paid`, `Refunded`, `Share`, `ShareCash` | tally (`LabourRound`), `GDP` (`ValueAdded`, `Cash`) | minor units; bp | 2 | that `ValueAdded` holds the wage and the employer's profit together (7.2.2), so the share is the wage bill over it and not a share of income; that a hire is paid by the tick and its value is booked at the job start, so a window that cuts a hire shows the two on different edges; the share is `OK` false for a value added at or below 0, and above 10 000 it is a result and not an error | 18.11 |
| 24 | `Employment` | the households of the hireable roles that were on a hired job: at a boundary `rdiv(10 000 * Employed, Households)` of the role from `RoleRow.Employed`, and over a window the `HouseholdTicks` paid over the running households at the start of each tick (6.3), by region and role; and `Employing`, the hired households working for agents of an employer role | `Rate`, `Employed`, `Employing`, `HouseholdTicks` | `Close.Roles[]` (`Employed`, `Employing`, `Households`), tally (`LabourRound`, the running households), `Config` (`Role.Hireable`) | bp; households | 1 or 2 | households, not agents; a household that works its own recipe is not employed in this sense, and employment is not "has income"; a role that is not hireable is absent, not 0; the denominator is the running count, as for `Unfed` | 18.11 |
| 25 | `Losses` | the units lost to strikes and their value at a table, by commodity, region and source: `route` (the `Lost` of each `ShipmentArrived` line and the `AccountLost` of a self-carried trip) and `store` (the `Lost` of `StoreLoss`), the strikes counted apart (`Struck`: the shipments with a `LossBP` above 0 and the stores struck), and the total against the boundary flow `loss` | `Units`, `Value`, `Struck`, `Source` | tally (`ShipmentArrived`, `ShipmentDispatched`, `StoreLoss`, `BoundaryFlow` of `loss`), `Prices` | units; minor units | 2 | a strike that took no unit (a line rounded down) is in `Struck` and not in `Units`; a loss is valued at the price of the region of the account that held it (an origin, for a cargo in transit, 8.1); the `Lost` of `ContractSettled` is the same units as a `ShipmentArrived` line and is not added again; the total equals the `loss` of the window, which is the check | 18.11 |

9.1.1. **Names.** The row's preset name is the exported function; the fields of
its result are the nouns of the row. The set of names and the closed sets they
use are rule constants of the package and a change of a name is a breaking change
(architecture.md 6.6). Two of the twenty-five (`WelfareDistribution`, `Inequality`) need household rows
and are class 3; the rest need only a snapshot or a tally.

9.1.2. **What a preset returns.** Every preset returns a result struct with plain
fields (no map; rows in canonical order), the `ConfigHash` of its inputs, the
`Window` it covers, an `OK` where it can be undefined, the clamp flag where it is
an index or a ratio, and its **warnings** as constants of a closed set
(`Stale`, `Clamped`, `Thin`, `Cohort`, `Absent`, `NoRoute`, `Basis`, `Binding`), so a band
that says "within 5% of the base" can also say "and not computed from stale
prices".

9.1.3. **Preconditions as errors.** A window that is not closed, two snapshots
from different worlds, a class that does not exist, a price table of another
configuration: each is an error (3.5), never a silent wrong number.

### 9.2 A consumer's own indicator

A consumer's indicator is a function of the same inputs, written with the three
aggregators (3.6). The one form every preset has:

```go
// ShareOfFoodInConsumption is the value of food consumed over all consumption,
// in basis points, at p. Food is a class of the consumer's own pack.
func ShareOfFoodInConsumption(t *indicator.Tally, cl indicator.Classes, p indicator.Prices) (indicator.BP, bool, error) {
	food, err := cl.Of("food")
	if err != nil {
		return 0, false, err
	}
	consumed := indicator.FlowSel{Reasons: []economysim.Reason{"consumption", "want_consumption"}}
	all, err := t.Value(consumed, p)
	if err != nil {
		return 0, false, err
	}
	consumed.Commodities = food
	part, err := t.Value(consumed, p)
	if err != nil {
		return 0, false, err
	}
	return indicator.Quotient(part, all) // rdiv(10 000 * part, all), OK false for all == 0
}
```

The example is a sketch of the semantics (the package's names are provisional,
3.6); the rules it shows are not: a class is an argument, a price table is an
argument, a quotient is `Quotient` and so rounds as 5.2.2, and an undefined result
is `ok` false.

## 10. Time

### 10.1 Windows and the finest resolution

A window is `[From, To)` in ticks (4.1.1). Every flow measure is over a window; every
stock measure is at a boundary or is the change between the two boundaries of a
window. There is no "rate at an instant". A per-tick **mean** is `rdiv(total, n)`
and a per-year figure is the annualisation of 5.7.2.

### 10.2 The calendar year and the season

10.2.1. **A year** is `TicksPerYear` ticks (CAL 3.2), derived from the calendar
and never configured. A **calendar-year window** is the ticks whose
`Moment.Year` is `y` (CAL 4): `YearWindow(cal, y)` is `[first tick of year y,
first tick of year y+1)` found by a binary search over `cal.At` (a monotone
function), `O(log)`. The first year of a world is **partial** when the calendar's
`StartDay` is not 0 (the world begins `StartDay` days into its year, CAL 3.1), and
the function says so: `Partial` true when the window has fewer than
`TicksPerYear` ticks, and an annualised figure of a partial window is exact in
the sense of 5.7.2 and is flagged.

10.2.2. **A season** is a set of ticks that repeats every year: `Moment.Season`
(CAL 4, -1 for none). `SeasonWindows(cal, from, to, s)` returns the list of
the windows of the season `s` that lie inside `[from, to)`, in order: each is a
**maximal run of consecutive ticks** with `Moment.Season == s`, clipped to
`[from, to)` (and flagged `Clipped`). A season may **wrap the year end** (CAL: the
default seasons, winter, cross it), and its window then crosses the year
boundary; it belongs to the year in which it starts. A measure over a season is
a series with one point per run, never one window over a non-contiguous set. A figure of one season **against the
same season of the previous year** is the way to read a seasonal economy: it is
a consumer's ratio of two points of that series (`Growth`, 5.7.1), and it is
not offered as a preset because the base is the consumer's.

10.2.3. **No cadence is aligned to a season** (WLD 9.3.3): a window that is a
year or a season is a **reading** choice and moves no tick of the world.

### 10.3 A series belongs to the consumer

10.3.1. **Decision.** The library keeps **no history**: not a series, not a
ring of snapshots, not a moving average (ADR 0002 decision 6 forbids hidden
state that a checkpoint cannot hold; a history in the world would be state that
must be saved, grows without bound and exists only for a reader). A
series is the consumer's `[]Point{Tick, Value}`, appended after each window, in
whatever store it has. This is the reason the arithmetic of 5 is over plain
slices and the tally is a value the consumer holds.

10.3.2. **What the library offers for a series** (the `W` class of 3.4): `Stats`
(first, last, lowest, highest, count, the mean rounded half to even, and the
ratio `High * 10 000 / Low`, `OK` false for `Low <= 0`), `Growth` between two
points (5.7.1) and `Change` over a series (the difference of the last and the
first). A trend fit, a moving average and a spectral measure are the consumer's.

### 10.4 What a recorder keeps, restated as a rule

A consumer that wants **every preset every window** keeps, per window, one
`Tally` (4.2), the two boundary snapshots (or only the closing one if it keeps
the previous as the next opening), the **`Base`** of its base window (the prices
**and the quantities**, which `Laspeyres` needs: `BaseOf(t, p)` freezes both from
the base window's closed tally, or the consumer keeps that tally), and, for
household measures, the rows of its last pass (4.5). That is the whole
requirement: of the order of tens of kilobytes plus the rows, **independent of
the length of the window**. A consumer that wants a **tick series** (a series of
one-tick windows) takes a snapshot per tick (class 1 and the scans of OBS 5.5,
about 1 ms at the first consumer's size, 2% of a tick, counted) or only the
tally's own rows, which need no snapshot, and keeps its points.

### 10.5 A window across a restore

10.5.1. **A restore is not a break in the run** (FR-CHK-050): a run split by a
restore is the same run, and events are not repeated across it. The tally is
the consumer's value and survives the restore as it is; if the process
restarts, the consumer's tally is **its** state and it keeps it as it keeps a
series (it is plain data with exported fields and no tags, and the consumer
encodes it as it likes). A restore therefore changes **nothing** in a window
that straddles it, provided the tally was open before and is fed the events of
the restored world.

10.5.2. **When there is no tally across.** A consumer that lost its tally opens
a new one at `WorldRestored`: the event carries the snapshot of the restored
boundary (OBS 4.4.5), and `Tally.Observe(WorldRestored)` on a tally with no
open window `Open`s it at that snapshot, with the households of its role rows
(6.3). A window never begins mid-step: the restore is at a boundary.

10.5.3. **What is not available after a restore:** the **construction bucket**
(LED 7.4.1: it is not saved, and a world restored after its first step has
none), so the memo line of 7.1.4 is empty; and the events of the steps before
the restore, which a consumer that needs them has from its own log.

## 11. What IND hands to STY, EXP and CLB

Stated as requirements on this spec's output, not as the content of theirs.

### 11.1 To STY (`spec-stylised-facts.md`)

A stylised fact is a band on a **named measure of a named window of a named
pack at a stated scale**. These are the measures a fact is expressed in; the
failure regimes of STB 7 map to presets as follows (OBS 10.2 gave the events;
this table gives the measure):

| Regime or fact | Preset and field | Window |
|----------------|------------------|--------|
| F1 runaway inflation | `PriceIndex.Fisher` by class (chain), `Sectors.M` and `Turnover.Velocity` | a window of at least one year, a series |
| F2 hoarding | `Sectors` shares of money by sector; `StockCover`; `Turnover.Volume` | a series |
| F3 collapse | `Households` change and `RoleShares` (a role at 0) | a series |
| F4 stagnation | `Turnover.Volume` against its first window | a series |
| F5 pinned market | `Pinned` (share) and the unmatched-both-sides count | a window |
| F6 cobweb blow-up | `PriceRange` over successive windows; `RoleShares` swing | a series |
| F7 synchronisation | `Dispersion` | a window |
| F8 offset | `Offset` against VIA's price | a window |
| price convergence between connected regions within transport cost | `PriceGap.GapAfterFreight` per connected pair | a window per season or year |
| recovery after a harvest shock | `GDP` real, `PriceIndex` of the shocked class, `Unfed` | a series from the shock |
| famine in a closed region | `Unfed` (household-ticks), `Households` change, `StockCover` | a series |
| the wage and demand loop | `RealWage`, `WelfareDistribution.Mean`, `HouseholdIncome`, and with the employment module on `Unemployment.Rate` and `WageBill.Share` | a series |
| R9 deflation with distress (proposed F9, `spec-stylised-facts.md` 6.2; the thresholds are there) | `PriceIndex.Fisher` chain and `Turnover.Volume` | a series |

11.1.1. **Requirements on STY's use.** A band is stated with the **scale**
(`CohortShare` is reported with every inequality figure, 6.2), the **window**
(a year, a season, a tick), the **price basis** (a table: window, reference,
base) and the **weights** of an index. A band that omits one of them is
ambiguous and IND returns what each preset actually used so that a test can
check the band was read the way it was written.

### 11.2 To EXP (`spec-experiments.md`)

| EXP may rely on | Where |
|-----------------|-------|
| every preset is a pure function of plain data that a harness already holds (a snapshot, a tally) and **takes no world**, so a sweep computes them on separate goroutines with no coordination (worlds share nothing, WLD 3.1.1) | 3.1, 3.3 |
| a result is keyed by the `ConfigHash` it carries, and by names, not indices, across configurations | 3.3.2 |
| two runs of one seed give equal results bit for bit, so a **paired comparison** of a measure is exact when the difference moves no draw count (WLD-A5), and the estimate of a difference of means is valid whether the runs are aligned or not (`spec-experiments.md` 7.4.1) | 3.1.4 |
| the series are the harness's: the library keeps none, and `Stats`, `Growth` and `Change` cover what a sweep summarises | 10.3 |
| a window is cheap: a tally is `O(rows)`, and a sweep of `K` runs of `T` ticks keeps `K` tallies, not `K * T` points | 4.2.4, 10.4 |

### 11.3 To CLB (`spec-calibration.md`)

| CLB may rely on | Where |
|-----------------|-------|
| each knob's **expected effect** can be stated as the sign of the change of one preset field (a rise in `CostFloor` raises `Pinned`, lowers `Turnover.Volume`), because a preset's field is a number with a unit and a window | 9 |
| the **targets** of a calibration are bands of presets (STY); the library never moves a knob by a preset (3.1) and CLB's procedure is a consumer's loop: measure, compare, adjust, run (roadmap) | 3.1.3 |
| a calibration that reads `Offset` needs VIA's price for the **same** configuration, and that is the caller's table | 9 (`Offset`) |

### 11.4 To INT, CHK and OBS (the seams, as asks)

| To | What IND asks or promises | Where |
|----|---------------------------|-------|
| OBS | fields: the last tick's income, relief and cost per role; the households whose ratio exists; wear units by commodity; optionally the lines of `AgentRelocated`. Each is a function of saved state or an aggregate event of class A | 7.5.1, 6.5, 7.6.2, 8.2, 22 |
| INT | nothing blocking; `Agents(r)` and `Agent(a)` are the household rows (4.5.2); a bulk column read is an open question | 4.5.3, 21 |
| CHK | nothing: a tally is not world state (15) | 10.5 |
| WLD | nothing: IND adds no position, no draw and no phase | 3.1 |

## 12. Parameters

This spec adds **no `Config` field, no default and no row** to the
parameter-owner table. An indicator changes no result (3.1), so nothing in it is
a tunable (architecture.md 9.5), and the registry test (testing.md 6.7) needs
nothing from it. A price table, a base window, a class name, a quantile and a
threshold are **arguments of a call**, not configuration.

**Rule constants**, beside their code (architecture.md 9.6): the basis-point
base 10 000; `MaxIndexBP = 2^31 - 1`; the rounding rule (half to even,
5.2.2); the Gini and quantile definitions (5.4, 5.5); the closed sets `Weights`,
`StatOp`, `Column` and the warning constants (9.1.2); the preset names; the
standard arguments of a preset (`Top(1 000)` and `Bottom(4 000)` of
`Inequality`, the ratio 10 000 of "a ratio below 1": a ratio of 1.0 is 10 000 by
the definition of the welfare ratio, WEL 2.1, not a setting). Changing a rule
constant changes a measure's value, so it is a behaviour change of the
package (architecture.md 6.6).

**Not parameters:** the window, the base table, the weights of an index (a
`Weights` constant argument), the class, the column of a row set.

## 13. Validation

No rule is added to `spec-config.md`: an indicator reads a configuration that
`Validate` has already accepted. The preconditions of a call are checked by the
call and are the errors of 3.5; each has a failing-case test (testing.md 3.9)
that asserts it with `errors.Is`. A **fuzz test** is required for the one
function family that parses nothing but takes arbitrary numbers (testing.md
7.4's spirit: a function that takes external input): the row statistics and the
index functions must never panic and must satisfy their properties for any rows
and tables (NFR-IND-003).

## 14. Observability

The Observability entry (observability-and-logging.md 9): **None, because**
indicators emit no event, write no log and register no instrument. They are
functions of the events and snapshots that OBS defines and consume them; the
library's events, metrics and logs are unchanged by this spec. Two things are
stated so that a reader does not look for them:

14.1. **No logger.** `indicator` has no `*slog.Logger`, no `WithLogger` and no
`Observer` of its own that logs: a warning of a result is a field (9.1.2), not
a log line, because a measure is called by a consumer's code and the consumer
decides what to log (rules 2, 2.3: anything a caller needs in order to behave
correctly belongs in the return value).

14.2. **Metrics from presets** are not defined here. `otelobserver` may expose a
preset as a gauge fed by a snapshot it is given (`RecordSnapshot`, OBS 6.2.2);
that is OBS's catalogue to extend, in the same change as a consumer that asks
(OBS 18 question 8), and a preset is the definition of the number such a gauge
would carry. The `otelobserver` module depends on `indicator` only for that,
and the core never does (3.2.1).

## 15. State contributed to a checkpoint

**None.** An indicator is a function and a tally is the consumer's value, not
the world's (3.1.4, 10.5): it is not saved by a checkpoint, a restore does not
read one, and the world's state is unchanged by a consumer computing every
preset. What a restore means for a window is 10.5. The names of the closed
sets are not state.

## 16. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. This spec
adds **no feedback loop inside the library**: by 3.1 nothing in the world reads
an indicator, and the checklist of `spec-stability.md` 9 was followed (9.1: no
rule of this spec reads a measure and changes a rate; the tests of 3.1.2 make
that a requirement). It adds three **hazards** that live outside the run or in
the reading of it, because a measure is the point at which a loop can be closed
by hand.

- **IND-A1. A consumer closes a loop through an indicator.** *Kind:* hazard.
  *Loop:* an indicator over a window (`PriceIndex`, `Inequality`, `Unfed`) of
  steps `t-n+1 .. t` (+) a consumer's rule that reads it (a game's reaction to
  inflation, a script that sets a flow scale) (+) a slot call (+) the state at
  `t+1`. Extends OBS-A1 with the **window**: a measure over `n` ticks is a
  moving average, so the loop's delay is `n/2` ticks beyond the one tick of the
  call, and a window as long as the loop's natural period turns a damped
  controller into an oscillator. *Owner:* INT 3.5; the consumer. *Damper:* none
  in the library, by design (ADR 0001 decision 7); a policy actor
  (`spec-policy.md`) is where a rule that reads measures is specified,
  with its own entries. *Undamped:* all of it. *Condition:* a controller whose
  gain times its total delay (`n/2 + 1` ticks) is small against 1, as for any
  loop with a delay (`spec-stability.md` 6.3). *Test:* none in the library; the
  non-interference test (FR-IND-003) shows only that the library does not close
  it. *Status:* hand.
- **IND-A2. A calibration targets a measure that has a blind spot.** *Kind:*
  hazard of method. *Loop:* knob (+) preset value (-) distance to a band (+)
  knob, run by the measure-compare-adjust-run loop of the roadmap (CLB, EXP).
  The loop can converge on a value of the measure by a route that the measure
  cannot see: an inequality band met by raising the weight of cohorts (within a
  cohort inequality is zero, 6.2); a price-index band met by letting markets
  carry stale prices (5.3.8); a GDP band met by output that is discarded (7.3.2).
  *Owner:* CLB, STY. *Damper:* the **field** that names each blind spot is part
  of the result (`CohortShare`, `Stale`, `Written`, `Absent`, `Binding`), and a
  band is stated with the scale, the weights, the window and the price table
  (11.1.1); the check at scale is SCL 7.2's. *Undamped:* a band that omits those
  fields. *Condition:* a band is a conjunction of a measure and its blind-spot
  field. *Test:* S10 (scales 1, 4, 10) reads `Inequality` with `CohortShare`.
  *Status:* hand.
- **IND-A3. A measure that is smooth for the wrong reason.** *Kind:* hazard of
  reading, a neighbour of STB-L03 (the stale price loop) and STB-F5. *Loop:*
  none: a price index over markets that did not trade is the carried-forward
  price, which does not move, so the index looks stable exactly when the markets
  are pinned or frozen (F4, F5). *Owner:* IND 5.3.8. *Damper:* `Stale` on every
  index, and `Pinned` and `Turnover` beside it in the same scenario (F5 is
  `Pinned`, F4 is `Turnover.Volume`). *Condition:* a stable-price claim cites
  `Stale` below a stated share. *Test:* S03 and S09 read `PriceIndex.Stale` and
  `Pinned` together. *Status:* hand.

No new damper and no new scenario (the checklist's items: none added). The tests
of this spec are unit and integration tests (section 20), not stability
scenarios.

## 17. What was run

Two Docker Python scripts (standard library only; `PYTHONDONTWRITEBYTECODE=1`;
never the host interpreter), archived in `doc/specs/models/ind/`. They are
models of the integer rules, **not of the library**, and nothing in them is a
default or a band (N-027).

| Script | What it checks | Result quoted in |
|--------|----------------|------------------|
| `arith.py` (`arith.out`) | half to even on both signs; the Gini rule of 5.4.1 against the pairwise sum with `Fraction` on 2 000 random row sets of up to 9 rows (equal, and within half a basis point of the rounded value), and, for rows of **totals** whose totals need not divide by the weights, on 3 000 more; the maximum `(W - 1) / W` for `W` of 1, 2, 3, 10 and 1 000; the vectors of the deflator, the clamp, a quantile and a share over totals; the nearest-integer property of the Fisher square root on 20 000 random products below `2^62`; a chain of 400 random links between 0.98 and 1.03 against the exact product (largest error 4.75 basis points); the bound `10 000 * W * S < 2^127` for `W` up to `10^15` and `S = 3 * 10^18` | 5.2, 5.3.4, 5.3.5, 5.4 |
| `worked.py` (`worked.out`) | every number of section 18: the production, expenditure and regional identities (and their equality), the recipe table with its `WIP` row, the freight and delivered price, the three indices, the Gini and the quantiles, the sectors, the households account, the distributions, the trade balance, the stationary world and the calendar windows | 18 |

What they do **not** show: that a measure is informative of a real run (nothing
has run), that the identities hold in the library (they are tests, section 20),
that any band is right, and the speed of the Go implementation (counted costs
only, 3.4).

## 18. Worked examples

An **invented** setting, flagged as an example (N-017): a river valley of two
regions `alpha` (`RegionID` 0) and `beta` (1), four commodities `grain` (0),
`loaf` (1), `wood` (2) and `cart` (3, the haul unit of a hired mode), the
default calendar (`TicksPerYear` 365) and a window of two steps, `[10, 12)`.
The words are data; nothing branches on them. All numbers are placeholders.
`worked.py` computes and asserts every figure below.

### 18.1 The world and the window

Window totals of units, from the tally (`BoundaryFlow`, by reason):

| Reason | alpha | beta |
|--------|-------|------|
| `recipe_output` | grain 100, loaf 30, wood 20, cart 6 | grain 50, wood 30 |
| `recipe_input` | grain 60 | |
| `consumption` | grain 10, loaf 20 | grain 35 |
| `want_consumption` | wood 5 | wood 10 |
| `spoilage` | grain 3 (one of them in transit) | |
| `disposal` | wood 2 | |
| `haul_use` | cart 4 | |
| `initial` | grain 5 (an external agent added at tick 11) | |
| `outside_offer`, `outside_unsold`, `outside_purchases` | | wood 20, wood 5, loaf 6 |

`ShipmentArrived` of the window: alpha to beta, grain, `Shipped` 10,
`Delivered` 9, `Diverted` 0, `Spoiled` 1. A second shipment of 6 grain from
alpha is still in flight at the close. Stocks (units, `Held` is agents plus
estate):

| | alpha open | alpha close | beta open | beta close |
|--|-----------|-------------|-----------|------------|
| grain, loaf, wood, cart held | 200, 40, 25, 3 | 217, 50, 38, 5 | 60, 10, 40, 0 | 84, 4, 75, 0 |
| pending output (loaf) | 12 | 8 | 0 | 0 |
| in flight (grain, origin alpha) | 0 | 6 | 0 | 0 |

Window unit values (the price table, 4.3.2): alpha grain 10, loaf 25, wood 6,
cart 5; beta grain 13, loaf 28, wood 8, cart 6. (Alpha's grain 10 is formed
from the market reports `(Price 10, Volume 30)` and `(11, 20)`:
`rdiv(10*30 + 11*20, 50) = rdiv(520, 50) = 10`, since 10.4 rounds to 10.) The closing held stock above follows from the ledger identity of 7.1.3, and the
second shipment's 6 units left `Held` for a Transit account. The 4 loaves by
which pending output fell are **dropped by an abandoned job** (`RecipeAbandoned`):
the 60 grain consumed at starts were for 30 loaves fixed, 30 created and 4 not
pending any more, so `Out + dPend = 26` is the output of started jobs net of
the dropped. No call between ticks was made in the window, so both `Pending`
totals are empty (7.1.5).

### 18.2 GDP: the three statements of 7.4

Alpha: `Output = 100*10 + 30*25 + 20*6 + 6*5 = 1 900`, `Intermediate = 60*10 =
600`, `Pending = (8 - 12) * 25 = -100`, so `ValueAdded = 1 900 - 600 - 100 =
1 200` (the cash form is 1 300). Expenditure: `C = (10*10 + 20*25) + 5*6 =
630`; `StockChange = (17*10 + 10*25 + 13*6 + 2*5) + 6*10 - 100 = 508 + 60 - 100 =
468`; `Written = 3*10 + 2*6 = 42`; `Carriage = 4*5 = 20`; `NX = 0`;
`Inter = 9*10 = 90`; `Endow = 5*10 = 50`. Then `630 + 468 + 42 + 20 + 0 + 90 -
50 = 1 200`.

Beta: `ValueAdded = 50*13 + 30*8 = 890`; `C = 35*13 + 10*8 = 535`;
`StockChange = 24*13 - 6*28 + 35*8 = 312 - 168 + 280 = 424`; `NX = 6*28 -
(20 - 5)*8 = 168 - 120 = 48`; `Inter = -9*13 = -117`; and `535 + 424 + 48 - 117 =
890`.

Realm at regional prices: `1 200 + 890 = 2 090 = C 1 165 + StockChange 892 +
Written 42 + Carriage 20 + NX 48 - Endow 50 + TradeValuation`, with
`TradeValuation = 90 - 117 = -27 = (10 - 13) * 9`. At **one** table (say alpha's
for every region) the interregional term is `9 * 10 - 9 * 10 = 0`: the realm's
identity has no term (7.4.1 a). The construction bucket is not in these flows
(7.1.4), and the `initial` of tick 11 is the `Endow` of alpha. *First test of
`GDP`:* equality `GDP^P = GDP^E` for each region with the two residuals (and the
realm's, at one table) exactly 0.

### 18.3 Value added by recipe

Alpha completed `grow` 50 executions (100 grain), `bake` 30 (30 loaf from 60
grain, `CostShare` of loaf 10 000), `cut` 10 (20 wood) and `make_cart` 6 (6
cart). Job values added: `grow 1 000`, `bake 30*25 - 30*2*10 = 150`, `cut
120`, `make_cart 30`, sum 1 300 = the cash form. By class (output class by cost
share): `food 1 150`, `fuel 120`, `carriage 30`. The `WIP` row is `1 300 -
1 200 = +100`: the jobs completed in the window were started earlier than the
jobs started in it, by the 4 loaves at 25 by which pending output fell. *First
test of `ValueAdded`:* the table sums to 1 300 and `WIP` equals `p * (In -
InCompleted - dPend)` = `10 * (60 - 60) - 25 * (8 - 12) = +100` (the started and
completed inputs are equal here, 60 grain each).

### 18.4 A price index of a class

Class `food` (grain and loaf), base prices `(8, 20)`, current `(10, 24)`, base
quantities `q0 = (12, 18)` and current `q1 = (10, 20)` (consumption):
`Laspeyres = rdiv(10 000 * (10*12 + 24*18), 8*12 + 20*18) = rdiv(10 000 * 552,
456) = 12 105` (12 105.26), `Paasche = rdiv(10 000 * (10*10 + 24*20), 8*10 +
20*20) = rdiv(10 000 * 580, 480) = 12 083` (12 083.33), and `Fisher =
isqrt_nearest(12 105 * 12 083) = isqrt_nearest(146 264 715)`: `r0 = 12 093` (`12
093^2 = 146 240 649`), the difference `n - r0^2 = 24 066` is above `r0`, so the
result is `r0 + 1 = 12 094` (`12 094^2 = 146 264 836`; the square root is
12 093.995). *First test of `PriceIndex`:* the three
values; the class `food` and the all-goods index are separate calls.

### 18.5 Prices across regions, freight and the market figures

Grain moves from alpha to beta by a hired shipment that redeemed 4 haul units
(cart, price 5 at alpha) and dispatched 10 units of grain: `HaulValue = 20`,
`Freight = rdiv(20, 10) = 2` per unit of cargo (one division, from the dispatch
event's own units), `Delivered = 10 + 2 = 12`, `GapLocal = 13 - 10 = 3`,
`GapAfterFreight = 13 - 12 = 1`. `Spread` is computed from the exact sums, one
rounding: `rdiv(2 * 10 000 * 3, 13 + 10) = rdiv(60 000, 23) = 2 609` (a share of a
rounded mean of 12 would give 2 500, 4% off: 5.2.3).

The **basket** of one household per tick is 2 grain and 1 loaf. At reference
prices alpha `(8, 24)` and beta `(12, 26)` its **computed** cost is `Bcost_alpha =
40` and `Bcost_beta = 50` (8.4(a); the configured `B_r` is another number within
the tolerance); at the window prices it costs 45 and 54, so `BasketLevel_alpha = rdiv(10
000 * 45, 40) = 11 250`, `BasketLevel_beta = rdiv(10 000 * 54, 50) = 10 800`,
and beta against alpha, from the exact sums, `rdiv(10 000 * 54 * 40, 50 * 45) =
rdiv(21 600 000, 2 250) = 9 600`.

Alpha grain over the two ticks: `Pinned = rdiv(10 000 * (6 + 1), 50) = 1 400`
(`Outcomes` 50, `AtFloor` 6, `AtCeiling` 1); `Dispersion = rdiv(10 000 * (12 -
9), 10) = 3 000` (last tick: `LowCentre` 9, `HighCentre` 12, `MeanCentre` 10);
`PriceRange = rdiv(10 000 * 11, 10) = 11 000` (published prices 10 and 11);
`Offset` against VIA's price 9 `= rdiv(10 000 * (10 - 9), 9) = 1 111`.
*First tests:* each number; `DeliveredPrice` has `Hired` true and `Freight` 2;
`PriceGap.Connected` true.

### 18.6 Money, velocity and cover

Region alpha at the close: `M_h = 5 000`, `M_p = 800`, `M_c = 200`, `M_t = 0` (no governed jurisdiction), `M = 6 000`,
100 households, so `M_h / H = 50`. Turnover of the window `sum(Price * Volume) =
3 000`, `V = rdiv(10 000 * 3 000, 6 000) = 5 000` per window, annualised from the
exact sums `rdiv(10 000 * 3 000 * 365, 6 000 * 2) = 912 500` (91.25 turnovers a
year). Grain cover at alpha:
held 217 units, consumed 10 over 2 ticks, `StockCover = rdiv(10 000 * 217 * 2,
10) = 434 000` (43.4 ticks), and with 37 of the 217 in the estate the unsold
share is `rdiv(10 000 * 37, 217) = 1 705`. A window total of 1 150 annualises to
`rdiv(1 150 * 365, 2) = 209 875`.

### 18.7 Households and the unfed share

Role `grower` at alpha opens with 40 households and `baker` with 10. In the
window: `WeightChanged` from 4 to 7 (+3, three births done), one agent of weight
1 is bankrupted, one of weight 1 changes role to `baker`, and an agent of weight
2 relocates to beta. Running counts: 40 at the start of tick 10, 42 at the start of tick 11 (after
+3 and -1 at the end of tick 10), 39 at the end of tick 11. The close is `grower` 39 at alpha and 2 at beta, `baker`
11; the account closes: `39 + 11 + 2 - (40 + 10) = +2 = 3 births - 1
bankruptcy`. `UpkeepShortfall` at alpha for `grower` is emitted at P6.4, before the P7
changes of its tick: `Unfed` 4 at tick 10 and 6 at tick 11. The denominators are
the running counts **at the start of each tick**, 40 and 42: the unfed share of
household-ticks is `rdiv(10 000 * (4 + 6), 40 + 42) = rdiv(100 000, 82) = 1 220`.
(Dividing by the counts after the tick's own P7 events, 42 and 39, would give
1 235: the wrong instant.) It is the **running** count that is the denominator,
not the households of the events (an event exists only when someone was
unfed).

### 18.8 Distributions, inequality, trade and the real wage

Money rows `(weight, total)`: `(1, 0)`, `(2, 40)`, `(4, 160)`, `(1, 60)` (the
cohorts of 2 and 4 hold 20 and 40 each): `W = 8`, `S = 260`, the exact money.
`Gini` over totals: `C` is 0, 0, 40, 200, 260 and `N = 8*260 - (1*(0+0) +
2*(0+40) + 4*(80+160) + 1*(400+60)) = 2 080 - (0 + 80 + 960 + 460) = 580`, `D = 2 080`, `Gini =
rdiv(10 000 * 580, 2 080) = 2 788`; the pairwise sum gives `1 160 / 4 160` the
same. Quantiles: `P10` has rank `ceil(0.8) = 1`, value 0; `P50` rank 4, value 40
(cumulative weights 1, 3, 7); `P90` rank `ceil(7.2) = 8`, value 60. Shares:
below 40 (`T < 40 * w`: the rows of 1 and 2), `rdiv(10 000 * 3, 8) = 3 750`. With
the cohort of 2 holding 41 (not divisible), the median row is the same and the
quantile at rank 2 returns `rdiv(41, 2) = 20`, the Gini over totals is 2 759. `Top(1 000)`: `k = ceil(0.8) = 1`
household, value 60 of 260, `2 308`; `Bottom(4 000)`: `k = ceil(3.2) = 4`
households (the row of 1, the row of 2, one of the four), value `0 + 40 + 40 =
80`, `3 077`; `P90 / P10` undefined (`P10 = 0`). **`CohortShare`**: the rows of
weight above 1 hold 6 of the 8 households, `7 500`: three quarters of the
households are held identical by construction and the result says so.

Welfare ratios `(1, -5 000)`, `(2, 8 000)`, `(4, 10 000)`, `(1, 25 000)`:
`Mean = rdiv(76 000, 8) = 9 500`, `P10 = -5 000`, `P50 = 10 000`, `P90 =
25 000`, share below 10 000 `rdiv(10 000 * 3, 8) = 3 750`; there is **no Gini**:
the first row is negative (5.4.3).

Trade at gate beta: `outside_budget 200`, `outside_unspent 32`,
`outside_receipts 120`: `export_value = 168`, `import_value = 120`, net `+48`
(the 6 loaves at 28 are exactly 168); units imported `20 - 5 = 15` wood, exported
6 loaf. Openness at the table: `(168 + 120) * 10 000 / 2 090 = 1 378`.

Real wage at alpha: observed wage 24 per working household-tick, upkeep cost
`2*10 + 1*25 = 45`, `rdiv(10 000 * 24, 45) = 5 333`.

### 18.9 The mean of ratios, the ratio of sums, and a stationary world

Two agents of weight 1 with `(income, cost)` `(20, 10)` and `(20, 40)` have
ratios 20 000 and 5 000: the **mean of ratios** is 12 500 and the **aggregate
ratio** `40 * 10 000 / 50 = 8 000`. A claim about "the mean welfare ratio is 1 in
a closed economy" (N-054) is stated on the aggregate (7.7.2).

A stationary one-tick world for the income view: a grower makes 4 grain and
sells all 4 to a baker at 10; the baker makes 2 loaf from them and sells both to
the grower at 25; the grower eats the 2 loaf; no stock changes. Incomes: grower
`4*10 = 40`; baker `2*25 - 4*10 = 10`; total 50. `GDP^P = 4*10 + 2*25 - 4*10 =
50`; `GDP^E = C = 2*25 = 50` with no stock change. The `Gap` of 7.5.2 is 0. *First
test of `HouseholdIncome`:* in version 1 it returns `OK` false (the fields are
not adopted); this world is its test once they exist.

### 18.10 Windows of a calendar year and of a season

`TicksPerYear` 365, `StartDay` 100, two seasons: `warm` (season 0, days 59 to
258) and `cold` (season 1, days 259 to 364 **and** 0 to 58: it wraps the year
end). The world starts on day 100 of year 0, so year 0 is `[0, 265)` (`Partial`
true: 265 ticks) and year 1 is `[265, 630)`. Season `warm` has windows `[0,
159)` and `[324, 524)`; `cold` has `[159, 324)`, which **crosses the year
boundary at 265** (106 ticks of year 0, 59 of year 1), and `[524, 630)` (clipped
at the end of year 1; `Clipped` true). *First test of
`YearWindow` and `SeasonWindows`:* these boundaries from `Calendar.At`.

### 18.11 Hired labour, and a lost cargo

A second window of the world of 18.1, with the employment and risk modules on
(it leaves 18.2 as it is). At alpha the `grower` role is hireable. In tick 10
`LabourRound` has `Offered` 12, `Hired` 9, `NoVacancy` 3, `HouseholdTicks` 9 and
`Paid` 270 (a wage of 30); in tick 11 `Offered` 10, `Hired` 8, `NoVacancy` 2,
`HouseholdTicks` 8 and `Paid` 240 (no start was cut, so nothing is refunded). `Unemployment = rdiv(10 000 * (3 + 2), 12 + 10) = rdiv(50 000, 22) =
2 273` (2 272.7). `WageBill`: `Paid = 270 + 240 = 510` against `ValueAdded` 1 200
of 18.2, `Share = rdiv(10 000 * 510, 1 200) = 4 250` exactly, and against the
cash form 1 300, `ShareCash = rdiv(5 100 000, 1 300) = 3 923` (3 923.08).
`Employment` over the window: the paid household-ticks 9 + 8 = 17 over the running
`grower` households at the start of the two ticks, 40 and 42 (18.7), `rdiv(170 000,
82) = 2 073` (2 073.2); at the close, if 8 of the 39 `grower` households at alpha
are on a hired job of more than one tick, `rdiv(80 000, 39) = 2 051` (2 051.3).

A shipment of 10 units of grain leaves alpha with `LossBP` 2 500 and arrives:
`Lost = rhe(10 * 2 500, 10 000) = rhe(2.5) = 2` (half to even), `Delivered` 8; and
a store strike takes 3 loaves at alpha. `Units = 2 + 3 = 5`, `Value = 2 * 10 + 3 *
25 = 95` at alpha's table (grain 10, loaf 25), `Struck = 1` shipment `+ 1` store,
and the boundary flow `loss` of the window is 5 units, the check. *First tests of
`Unemployment`, `WageBill`, `Employment` and `Losses`:* these numbers.

## 19. Notes settled

Per `AGENTS.md`: every open note tagged `IND` and every `ALL` note, and the
notes the brief names, each applied, rejected or left open with the reason.

| Note | Decision | Where |
|------|----------|-------|
| N-013 What indicators must not get wrong | **Applied** (and closed when this spec is accepted): within-cohort inequality is zero by construction and reported (`CohortShare`, 6.2); population indicators count households and exclude pseudo-agents (6.1); balance sheets value at a caller's price and are measurement only (4.3, 7.5.3); wear units back the depreciation measure only through an ask (7.6); classes are the grouping, and class value added is not defined from output and input (7.2.3); a report by region reconciles by the relocation totals (8.2) and households by region are rebuilt from events (6.3); the money supply by sector and the boundary balance per reason are `Sectors` (9); a price index says local or delivered (8.4) | 6, 7, 8 |
| N-081 Snapshot figures the scenarios need | **Applied** for IND: the mean ratio per role and region is read from `RoleRow` (9, `WelfareDistribution`), the unsold stock from `MarketRow` (`StockCover`), the production halt is `ProductionHalted` (not an indicator). `RatioHouseholds` already exists in OBS and is the weight of the mean (6.5) | 6.5, 9 |
| N-094 Snapshot fields of the wave 2 modules | **Applied** as a consumer: the fields OBS named are read by `Sectors`, `Households`, `RoleShares`; the fields IND still needs (income of the last tick, wear units) are **not adopted for version 1**, a future ask in 22 | 9, 22 |
| N-115 The boundary flows of the outside are gross | **Applied**: `TradeBalance` derives the net and reports gross and net, in units and in money, and the volume at a table (OBS added no net event, FR-OBS 4.4.7) | 7.3.3, 9 |
| N-054 The closed economy fixes the mean welfare ratio at 1 | **Applied**: the claim is stated on the aggregate ratio, not on the mean of ratios, and `HouseholdIncome` returns the aggregate; the mean of ratios is `WelfareDistribution.Mean` and the two differ by the covariance of the ratio with the cost (18.9) | 7.7.2, 18.9 |
| N-109 The closed forms of the Malthusian loop need the welfare ratio at a full draw | **Left open** for POP and VIA. What IND learned: `r_full` is a counterfactual, not a function of a snapshot (OBS 5.8), so no preset can compute it from a run; a **proxy** exists (the ratio of households that draw at full abundance in a run with the resource unbound), and it would be a consumer's measure over `RecipeExecuted` rows; the solver is the place for the figure | 21 |
| N-027 No default macro constant is stable yet | **Applied** as a standing constraint: this spec fixes what a measure is, not a band; every example number is a placeholder (17) | preamble |
| N-016 Keep the parameter registry true | **Applied**: no field, no default, no row (12) | 12 |
| N-017 Words and names | **Applied**: new terms listed below for the glossary; no setting word outside the flagged example | 18 |
| N-103 Running several spec writers in parallel | **Applied**: IND was written after OBS, with OBS, CHK and INT drafted and read-only; no conflict found with them (22 lists the asks) | the handoff |

New terms for the glossary (`specs/README.md`, `doc.go`): **indicator**,
**preset**, **aggregator**, **tally**, **window** (of ticks), **household row**,
**price table**, **base window**, **index** (basis points of a base),
**Laspeyres**, **Paasche**, **Fisher**, **chain index**, **clamp** (of an
index), **stale share**, **cohort share**, **value added** (of a region),
**production approach**, **expenditure approach**, **income view**, **gross
accumulation**, **written** (goods), **carriage** (use of goods in moving goods),
**delivered price**, **freight**, **trade valuation**, **retirement basis**,
**binding counters**, **reconciling item**.

New notes to add to `doc/notes.md` (areas in brackets), in the handoff: the
inputs for the income view [OBS, IND, WEL] and the wear units by commodity
[OBS, LED, IND], both future asks not adopted for version 1; a bulk read of
household rows [INT, IND]; goods lines on `AgentRelocated` for the regional goods
identity [OBS, REG, IND]; per-line carrying units for the delivered price [TRN,
OBS, IND]; the wear of a tool-using recipe in `RecipeExecuted` [PRO, IND].

## 20. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

FR-IND-030 to FR-IND-050 and FR-IND-056 to FR-IND-059 are one requirement per preset whose clauses (the
fields and the "must not hide" field of its row in 9) are asserted as separate
checks of that preset's worked-example test.

| ID | Requirement |
|----|-------------|
| FR-IND-001 | No package of the module other than `indicator`, its tests, `otelobserver` and `examples/` MUST import `indicator`: the root package, `config/`, `strategy/` and every package under `internal/` MUST NOT, and no package other than `indicator` MUST import `internal/measure` (3.1.2, 3.2.1). |
| FR-IND-002 | No exported identifier of `indicator` MUST have a parameter or a result of type `World`, `*World` or a function type that takes or returns one (3.1.2). |
| FR-IND-003 | The same seed, configuration, calls and observers MUST give the same checkpoint digest, event stream and snapshots whether or not a consumer computes every preset after every step (3.1.2). |
| FR-IND-004 | A tally's size MUST be a function of the configuration and of the open contracts and MUST NOT depend on the number of agents, orders, trades or ticks of the window (4.2.2). |
| FR-IND-005 | A tally MUST implement `Observer`, MUST ignore every event type it does not use, including `OrderPlaced`, `OrderRejected` and `TradeExecuted`, MUST ignore every event while no window is open, and MUST NOT call the world (4.1.4, 4.2.1, 4.2.3). |
| FR-IND-006 | `Close` MUST return `ErrWindowOrder` when the tally's `TickEnded` count is not `Close.Tick - Open.Tick`, when an event carries a tick outside the window other than the construction bucket, or when a step failed, and `ErrConfigMismatch` for snapshots of different `ConfigHash` (4.1.2, 3.5). |
| FR-IND-007 | The `BoundaryFlow` rows of `Tick` -1 MUST be kept apart and MUST NOT be in a window's flows or in `GDP`, and `initial` at `Tick` 0 or later MUST be in the window's flows as `Endow` (7.1.4). |
| FR-IND-008 | `indicator` and `internal/measure` MUST NOT use a floating-point type or import `math` (5). |
| FR-IND-009 | Every conversion of a wide value to `int64` MUST be checked and MUST return `ErrOverflow` where it does not fit, and no arithmetic on a value that came from an input MUST wrap (5.6). |
| FR-IND-010 | The result of a measure MUST be independent of the order of its input rows and of the order of summation, and MUST be identical on every platform (5.2.5). |
| FR-IND-011 | A tally's running households by (region, role) MUST equal `Snapshot.Roles[].Households` at every tick boundary of a run (6.3). |
| FR-IND-012 | A quotient MUST be rounded half to even, symmetric for a negative numerator, and an undefined quotient (a zero divisor) MUST return `OK` false and never panic (5.2.2, 5.2.4). |
| FR-IND-013 | The nominal value `N` of a total MUST satisfy `|N - rdiv(R * deflator, 10 000)| <= ceil(R / 20 000)`, with `R` its real value (the deflator is rounded to half a basis point) (5.3.6). |
| FR-IND-014 | The Gini coefficient of a row of weight `w` MUST equal that of `w` rows of weight 1, and MUST equal the pairwise definition for every row set (5.4.1). |
| FR-IND-015 | The Gini coefficient MUST be 0 for equal values and `(W - 1) / W` for one holder, MUST return `ErrNegativeValue` for a negative value and `OK` false for `W = 0` or `S = 0` (5.4.3). |
| FR-IND-016 | For each commodity of the realm, over each window, `dHeld + dTransit` MUST equal the units made less the units used and lost plus the units that the outside brought and `initial` created, exactly, with the flows corrected by `Close.Pending` less `Open.Pending`, for a window with a call between ticks before `Open` and one after `Close` as for one with none (7.1.3, 7.1.5). |
| FR-IND-017 | At one price table for every region, `GDP^P` of the realm MUST equal `C + StockChange + Written + Carriage + NX - Endow` exactly, for every window, seed and pack (7.4.1 a). |
| FR-IND-018 | Over a window the sum of the changes of the net worth of the sectors (households, pools, contracts and hires, treasuries) at one price table MUST equal the value of the money and goods boundary flows of the window plus `p * dPend`, exactly (7.5.3). |
| FR-IND-019 | For each region the change of its money MUST equal its money faucets less sinks plus the money relocated in and out, the contract payments settled in and out and the money that crosses a region boundary under a fiscal flow (`FiscalIn`, `FiscalOut`), exactly, the money of a region including its treasuries, and the units of the relocation item summed over commodities MUST equal the units of `AgentRelocated` in total (8.2). |
| FR-IND-020 | An index MUST be 10 000 against its own base, MUST be computed from the exact sums by the formulas of 5.3.3 with the weights it names, and MUST return `OK` false for a zero denominator. |
| FR-IND-021 | The Fisher index MUST be the integer nearest to the square root of the product of the rounded Laspeyres and Paasche indices (5.3.4). |
| FR-IND-022 | A chain index MUST follow `C_k = rdiv(C_{k-1} * I_k, 10 000)` from `C_0 = 10 000`, and a clamped link MUST make every later point `Clamped` (5.3.5). |
| FR-IND-023 | An index, ratio or growth at or above `MaxIndexBP` MUST be clamped to it with `Clamped` true and MUST NOT be an error, and every index MUST report its `Stale` share (5.3.7, 5.3.8). |
| FR-IND-024 | A quantile MUST be the weighted nearest-rank value of 5.5.2, MUST be a value of a row, and MUST not depend on the order of ties. |
| FR-IND-025 | A share MUST be computed from the exact sums over the rows strictly below the threshold (5.5.3). |
| FR-IND-026 | Every result over household rows MUST carry `Households`, `Agents`, `Absent`, `Cohort` and `CohortShare` (4.5.4, 6.2). |
| FR-IND-027 | A row set MUST contain no pseudo-agent, every population figure MUST be a sum of weights, and an `External` agent MUST be excluded only when the caller asks (6.1, 6.4). |
| FR-IND-028 | `PricesOfWindow` MUST return the volume-weighted mean of the published prices rounded half to even, and the last published price where no unit traded, with the `ConfigHash` of its window (4.3.2). |
| FR-IND-029 | A class name that no commodity carries MUST return `ErrUnknownClass`, and a commodity of two classes MUST be counted once in a whole-economy measure (4.4.1). |
| FR-IND-030 | `GDP` MUST return the production view with cash beside the accrual, the expenditure components, `Inter`, `Reloc`, `Endow`, the residual of the identity and the `Basis` of depreciation, by region and for the realm (9, row 1). |
| FR-IND-031 | `ValueAdded` MUST return output and intermediate use by class and value added by recipe on the completion basis with the `WIP` row, and MUST NOT define a class's value added from output and input (9, row 2). |
| FR-IND-032 | `PriceIndex` MUST return Laspeyres, Paasche and Fisher with its weights, `Stale` and `Clamped` (9, row 3). |
| FR-IND-033 | `BasketLevel` MUST be `rdiv(10 000 * the basket at the published prices, Bcost_r)`, `Bcost_r` the computed cost of the region's baseline upkeep at its effective reference prices, so that it is exactly 10 000 at reference prices (8.4, 9 row 4). |
| FR-IND-034 | `DeliveredPrice` MUST be the origin's price plus `rdiv(HaulValue, Units)` of the dispatches of the lane's hired mode, with `Hired` false and `Freight` 0 for a self-carried lane (8.4, 9 row 5). |
| FR-IND-035 | `PriceGap` MUST return the spread and the gap after freight with `Connected`, and MUST exclude an unconnected pair from any average (9, row 6). |
| FR-IND-036 | `Sectors` MUST return money by sector (households, pools, contracts and hires, treasuries) and per household, the boundary balance by reason and the change of net worth by sector with the closing of the identity of 7.5.3 (9, row 7). |
| FR-IND-037 | `Turnover` MUST return value and volume by market, region and class and the velocity of the window and per year, with `Active` ticks (9, row 8). |
| FR-IND-038 | `Households` MUST count households and agents separately and return the account of change of a window with a zero residual (9, row 9). |
| FR-IND-039 | `WelfareDistribution` MUST return the mean, the quantiles, the shares below the thresholds and the earned ratio, MUST NOT return a Gini and MUST report `Absent` and `Cohort` (9, row 10). |
| FR-IND-040 | `Inequality` MUST return the Gini of money and the other measures of 5.4.4 with `CohortShare` (9, row 11). |
| FR-IND-041 | `Unfed` MUST count households, MUST use the running count of households as the denominator of a window, and MUST report the largest hunger (9, row 12). |
| FR-IND-042 | `TradeBalance` MUST return gross and net, in units and in money, by node, gate and commodity, and the openness (9, row 13). |
| FR-IND-043 | `RealWage` MUST be `rdiv(10 000 * wage, upkeep cost)` and MUST be absent where the role has no observed wage (9, row 14). |
| FR-IND-044 | `StockCover` MUST return the cover in ticks and the unsold share and MUST be undefined where nothing is consumed (9, row 15). |
| FR-IND-045 | `RoleShares` MUST report a role with no households as 0 and not drop it (9, row 16). |
| FR-IND-046 | `PriceRange` MUST return the lowest and highest published price of the window and the ratio, with `Active` (9, row 17). |
| FR-IND-047 | `Pinned` MUST return `(AtFloor + AtCeiling) / Outcomes` and the markets with unmatched units on both sides and no volume (9, row 18). |
| FR-IND-048 | `Dispersion` MUST return `(HighCentre - LowCentre) / MeanCentre` of the last tick and the window's lowest (9, row 19). |
| FR-IND-049 | `Offset` MUST return the signed difference of the window price from the caller's reference price with `Unmet` need (9, row 20: the share of household-ticks of the window that were unfed, in basis points). |
| FR-IND-050 | `HouseholdIncome` MUST return `OK` false until the snapshot carries the income fields and, when it does, the booked income, relief, cost, the aggregate ratio and the `Gap` (9, row 21). |
| FR-IND-051 | A regional comparison MUST carry the warning `Binding` with the counters when a cap or a limit of a compared region acted in the window (8.7). |
| FR-IND-052 | `YearWindow` MUST return the ticks whose `Moment.Year` is `y` with `Partial` set where fewer than `TicksPerYear`, and `SeasonWindows` each maximal run of consecutive ticks of the season, across a year boundary where the season wraps, clipped to the range with `Clipped` set, in order (10.2). |
| FR-IND-053 | An annualised total MUST be `rdiv(x * TicksPerYear, n)` for a window of `n` ticks (5.7.2). |
| FR-IND-054 | `indicator` MUST keep no package-level mutable state, MUST start no goroutine, MUST draw no random number and MUST keep no history (3.1.4, 10.3). |
| FR-IND-055 | A tally started at `WorldRestored` MUST open at that event's snapshot, and a window that straddles a restore with the same tally MUST give the result of the uninterrupted run (10.5). |
| FR-IND-056 | `Unemployment` MUST be `rdiv(10 000 * NoVacancy, Offered)` of the household-ticks summed over the window by region and worker role, MUST return `Offered`, `Hired` and the refusals by reason beside it, and MUST be `OK` false where nothing was offered (9, row 22). |
| FR-IND-057 | `WageBill` MUST return the `Paid` and the `Refunded` of `LabourRound` summed over the window by region, and the shares of the value added in its accrual and cash forms, `OK` false for a value added at or below 0 (9, row 23). |
| FR-IND-058 | `Employment` MUST count households and not agents, MUST be `rdiv(10 000 * Employed, Households)` at a boundary and the paid household-ticks over the running households of the window otherwise, and MUST omit a role that is not hireable (9, row 24). |
| FR-IND-059 | `Losses` MUST return the units lost and their value by commodity, region and source, MUST count a strike apart from the units it took, MUST NOT add the `Lost` of `ContractSettled` to that of `ShipmentArrived`, and its total MUST equal the boundary flow `loss` of the window (9, row 25). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-IND-001 | The allocations of the class 1 and class 2 presets MUST be a function of the rows of the snapshot and the tally and MUST NOT depend on the number of agents, which a test over two scales of one configuration MUST show (3.4). |
| NFR-IND-002 | A tally's `Observe` MUST allocate nothing for an event of an existing row once the rows exist (`testing.AllocsPerRun`) (4.2.2). |
| NFR-IND-003 | The row statistics, the index functions and the accounts MUST have fuzz tests asserting that they never panic and that the properties of FR-IND-010, 014, 015, 020 and 024 hold for any input (13). |
| NFR-IND-004 | `indicator` MUST import only the standard library, the root package and `internal/measure`, and `internal/measure` only the standard library (architecture.md 5.1). |

## 21. Open questions

1. **A bulk read of household rows.** `Agents(r)` then `Agent(a)` builds an
   `AgentState` for each household, about 23 reads and a few hundred bytes each
   (4.5.3), once per window. A call that returns one column of `{Weight, Value}`
   for every agent without building the rest, in class 3 and one allocation,
   would cost about a tenth of that. Not adopted: it is additive to INT's
   surface and no measure shows the cost matters. Next step: the slice's measure
   of a `RowsOf` pass at the first consumer's size, then INT.
2. **The income view.** The fields of 7.5.1 are not adopted for version 1, so
   `HouseholdIncome` is `OK` false and the closed-economy claims of N-054 are
   stated but not measurable from a snapshot. Kept as a future ask: three
   `int64` per role row, a scan the snapshot already pays (OBS 5.5).
3. **A durability-weighted stock.** A snapshot values a worn durable as new
   (4.5.5, 7.6.2); retirement basis only for version 1. The accrual form needs the wear units by commodity. The
   alternative, a durability-weighted quantity in `MarketRow`, is a second field
   per market. The recommendation is the wear-units event (one aggregate, class
   A); the choice is OBS's.
4. **Value added by recipe and an input that is a tool.** A recipe's wear
   (`WearPerUse`) is a cost of the job that this spec's job value added
   (7.2.4) does not subtract, since the tool's wear is not in `RecipeExecuted`.
   The completion-basis table therefore **overstates** the value added of a
   tool-using recipe by the wear value; `GDP^P` is unaffected (the tool is
   retired at its end, `Written`). Next step: PRO's `RecipeExecuted` carries the
   wear units, or the table is net of a per-execution wear value from the
   recipe and the table price.
5. **Labour share and the wage bill.** Answered by EMP (`spec-employment.md`
   10.2.1): with the employment module on, the wage is a flow (`wage_pay`) and
   `WageBill` (9, row 23) reports it with the labour share of the value added;
   with it off there is no wage bill and the labour share is not defined. The
   compensation row of the income view waits for the income fields of 7.5.1.
6. **A metric from a preset.** Which presets, if any, an adapter should expose
   (14.2). Next step: the first dashboard (OBS 18 question 8).
7. **A default for `Weights`.** `Consumed` for a consumer price index, `Traded`
   for a market index; the spec offers both and names neither as "the" index
   (a default would be a claim, N-027). Answered by STY 4.2.2: there is no default
   weight set, and each fact names its own.
8. **A per-line freight.** `ShipmentDispatched` gives the haul units of the
   whole shipment, and the freight of a line is apportioned by chargeable size
   (8.4). Exact for a one-line shipment, a convention otherwise. Next step: TRN
   and OBS add `HaulUnits` to the line if a scenario needs it (X1, X2).
9. **A sample correction.** The Gini of 5.4.1 is the population coefficient,
   with no `n / (n - 1)` correction, so a two-household world has at most 0.5.
   A consumer that wants the sample estimator multiplies by `n / (n - 1)`
   itself. Not offered: the households of the library are the population.
11. **One checked 128-bit helper.** `spec-currency.md` 7 names `internal/`
    checked-arithmetic helpers without a package; `internal/measure` must not
    duplicate them. Next step: the first code decides the one home.
10. **A median of agents and a mean of agents.** Both are offered only as
    household-weighted measures. A measure over agents (the number of agents
    that hold less than a price) is a count of rows and is the consumer's.

## 22. Edits owed to other files

One consolidated table: file, section, exact change. None alters a result that
a test could depend on (nothing has run); rule files need a fresh read-receipt
tag each; and the **asks** (marked A) are requests to the owner, with this
spec's recommendation.

### The rules and the layout

| File | Section | Change |
|------|---------|--------|
| `doc/rules/architecture.md` | 2.6 | add `indicator/` (the public package of derived measures, imports the root, imported by nothing in the module) and `internal/measure/` (checked arithmetic over primitive types, 128-bit sums, nearest-rank, Gini) to the layout; they are created at the first code that needs them (2.1) |
| | read receipt | a fresh tag |
| `AGENTS.md` | Layout | the same two lines |
| `doc.go` (glossary) | | the terms of 19 |
| `doc/rules/observability-and-logging.md` | none | no change (14) |

### The specs

| File | Section | Change |
|------|---------|--------|
| `spec-observability.md` (future ask, **not adopted for version 1**) | 5.3.4 `RoleRow` | `IncomeLast`, `ReliefLast`, `CostLast` (`Money`, household-weighted sums of `income_t`, `F_t` and `cost_t` of the last record of the welfare ring: a function of saved state, FR-OBS-043); until adopted `HouseholdIncome` is `OK` false. `RatioHouseholds` is already in OBS and needs nothing |
| (future ask, **not adopted for version 1**) | 4.1.x, 4.4.1 | an aggregate event `WearApplied` (`Region`, `Commodity`, `Units`: the durability units removed in the tick; class A, never when zero), or the same as a field of `BoundaryFlow` for the reason `wear`; the ledger already counts them (FR-LED-016) |
| (A, optional) | 4.1.6 `AgentRelocated` | `Lines []Amount` (the goods that moved with the account, bounded by the tradable set); recommendation: not now (8.2) |
| | 10.1 | the IND row points to `spec-indicators.md` 3.3 and 9; 18 question 8 is answered: IND adds no event and no metric (14) |
| `spec-transport.md` (A) | 13.1 `ShipmentDispatched` | optional carrying units (the binding fill, TRN 3.2.1) per line, so that the freight per commodity is exact (8.4, open question 8); 15 TRN-A4: the delivered-price index is `spec-indicators.md` 8.4 |
| `spec-ledger.md` | 5.4.2, 15 questions 4, 7, 10 | the wear units are not in an event (the ask above); question 4 answered (the valuation is the caller's table, 4.3); question 7 answered (retirement basis, 7.6); question 10 answered (pending output is in `NW_h` at the table price, 7.5.3) |
| | 7.4 | "read by the observability and indicator specs" reads "reach them through `BoundaryFlow`" (WLD already asks) |
| `spec-goods.md` | 6.1, 9 | `spec-indicators.md` exists; "a value added per class" reads "an output and an intermediate use per class, and a value added by recipe grouped by cost share" (7.2.3) |
| `spec-scale.md` | 5.1, 9.2 | the inequality indicator reports `CohortShare` (6.2); the snapshot fields are OBS's |
| `spec-money-supply.md` | 3.1.3, 8.4, open question 12 | velocity and money per household are `Turnover` and `Sectors`; the identity `sum(p * q) = M * V` is the `Turnover` definition; question 12 answered |
| `spec-external-sector.md` | 8.3, 15.4 | the balance of trade is `TradeBalance` (gross and net, units and money) |
| `spec-welfare-and-wages.md` | 2.1.7, 7.2, 11.2 | the aggregate ratio and the mean of ratios are two measures (7.7.2); `MeanRatioBP`'s weight is the households whose ratio exists |
| `spec-stability.md` | 7, FR-STB-012, "The regime measures" row | the map of 11.1 gives the preset and field of each regime |
| `spec-reference-chain.md` | 7, 9 | the measures of the scenarios are the presets of 9; the income fields of 21 are the ask |
| `spec-regions.md` | 6.4.2, 10 | the reconciling item of a regional report is `spec-indicators.md` 8.2 |
| `spec-integration.md` | 9.4, 6.2 | the household rows are `Agents` and `Agent` (4.5.2); `RowOpt.ExcludeExternal` is the filter; a bulk read is an open question (21) |
| `spec-checkpoint.md` | 11 (STY, IND) | "an indicator that accumulates must carry its own state" reads: the tally is the consumer's plain value (10.5) |
| `spec-world.md` | 10.2, 13 (IND), 22 question 9 | `Binding` is the mechanism by which IND honours 10.2 (8.7); question 9 answered: a tick summary is derived from events |
| `spec-demand.md` | 5 (line on indicators) | `want_consumption` is `C` with `consumption` (7.1); the split is the reason's |
| `spec-production.md` | 7.3, 14.3 | waste produced is `Disp` in `Written` and in the `Disposed` of the recipe table; work in progress is `Pending` (7.2.1) |
| `spec-subsistence.md` | 14, open question 11 | the duty cycle is `OwnUseChanged.Started` over `RecipeExecuted` for the own-use recipes (7.8.3) |
| `spec-currency.md` | 5.1, 8 (not modelled) | the reference basket is the fixed basket of `BasketLevel`; an inflation index is a measure, not a behaviour |
| `specs/README.md` | index, code table, glossary | the IND row (exists, FR-IND-001 to 055, NFR-IND-001 to 004), the new terms |
| `doc/spec-roadmap.md` | existing specs | the IND row exists; "Suggested next specs" drops IND |
| `doc/documentation_index.md` | | the rows for the spec and for `specs/models/ind/` |
| `doc/specs/models/README.md` | contents | the `ind/` row (`arith.py`, `worked.py`) |
| `doc/notes.md` | open notes | in the handoff |

## 23. Where this lives in the code

| Concern | Home |
|---------|------|
| `Window`, `Tally`, `Prices`, `Classes`, `Household`, `Point`, the presets and their results, the errors | `indicator/` (one file per concern: `tally.go`, `prices.go`, `accounts.go`, `index.go`, `distribution.go`, `time.go`, `errors.go`) |
| 128-bit sums, `RoundDiv`, `Isqrt`, the Gini accumulation, nearest-rank, checked conversion | `internal/measure/` (primitive types only; architecture.md 2.9) |
| the tests: the import graph and signature tests, the identities over a run, the worked example, the fuzz tests | `indicator/*_test.go` (external test package), `testdata/golden/` for the rendering of a preset table over the reference chain |
| the models | `doc/specs/models/ind/` |
