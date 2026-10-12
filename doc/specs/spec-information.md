# Spec: Information: free local prices, and delayed, approximate rumours of remote ones

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-INF-NNN` and
`NFR-INF-NNN`, which tests cite (testing.md section 4). They are in section 16
(FR-INF-001 to 052, NFR-INF-001 to 004).

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
version 1 (`vision.md` section 4). Off unless configured: the zero configuration
has no `Modules.Information` and no agent ever learns anything about a region but
its own. Needs `spec-regions.md` (REG: the shortest distance between two regions
and the edge status, which this spec reads and nothing else), `spec-beliefs.md`
(BEL: the market price window, seeding), `spec-clearing.md` (CLR: the published
price and price age), `spec-agents.md` (AGT: the view a strategy reads and the
tick phases) and `spec-config.md` (the module switch and validation). It reads
`spec-transport.md` (TRN) only for the travel time, which it does not own.

**Every number in this spec is a placeholder** except where a derivation is shown.
Nothing has been run in the library, bazaarBot has not been run, and no acceptance
band exists (notes N-015, N-027). The two models of section 11 are stylised Docker
Python scripts of the rules, not claims about the library; the rules themselves are
integer. No default is claimed to be stable, and no historical figure about how fast
news travelled was read or used (N-018): the defaults of section 8 are guesses. The
settings of the worked examples (section 12) are **examples**, flagged as such; the
core and its vocabulary are setting-agnostic.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

Prices are the signal that moves goods between regions, and an agent in one region
can see the market of its own region and of no other. This spec decides what it may
learn about the others, how late and how exactly. It covers:

- what is known **for free**: the market report of the agent's own region, which the
  view already carries, and nothing else (section 3);
- the **price record**: for each market that can be a source of news, the last
  `RumourMaxAge` published prices, the only state this spec adds (section 4);
- the **rumour**: a plain-data reading of a remote market's price as it was some ticks
  ago, with an error, its **lag** from the distance and one speed, and the cases in
  which there is none (section 5);
- who is shown rumours: the roles flagged **informed**, in the view of the trade phase
  (section 6);
- where it happens in the tick, and what a closed road, a relocation and a trip do to
  it (section 7);
- parameters, validation, observability, state, the register entries and
  requirements (sections 8 to 10, 13, 14, 16).

Out of scope, with the spec that owns it: who ships, why, and how a shipper reads a
rumour (`spec-merchants.md`, MER); travel time, cost and the shipment (TRN); why and
when a household moves, and what it reads to decide (`spec-migration.md`, MIG); the
outside world as a source of prices, and how its price is seen (`spec-external-sector.md`,
EXT, section 4.5 states the seam); the tick phases as a whole (`spec-world.md`, WLD);
the events as a whole and the snapshot fields (`spec-observability.md`, OBS). This
spec sets no price, wage or rate and reads none into any rule of its own: it carries
what the markets published, late and blurred, to the agents that were told to listen.

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What does an agent know about another region for free? | Nothing. It sees the market report of its own region for the commodities of its tradable set (price, price age, the best unmatched quotes, the price window), which the view already carries. No new local field | N-106: no view carries a remote price unless INF says how late and how exact. The local report is free because the agent is in the market (3) |
| 2 | What is a rumour? | A reading of a remote market's published price, and of its window mean, as they were `Lag` ticks ago, multiplied by a bounded error, with the `Lag` and the source's price age beside them. Plain data: region, commodity, price, mean, lag, source age (5.1) | The shipper of MER needs a price to compare with its own, and needs to know how old it is. The age is exposed with every rumour so no strategy mistakes news for the present |
| 3 | Who holds rumours? | Nobody per agent. The world holds one **price record** for each source market, a short ring of what the market published; a rumour is computed from it when an informed agent's view is built | State grows with markets times the horizon, not with agents. A per-agent or per-region table of rumours would grow with agents or with regions squared (4.6, 7.8) |
| 4 | How does news travel? | Directly, in one step: the lag from a source region to a reader region is a function of REG's shortest distance from the one to the other and one speed. It does not relay hop by hop | One integer rule, independent of edge order and of rounding at each hop; the same graph serves goods and news. The cost is that an intermediate region plays no part (5.2, 17.2) |
| 5 | How late? | `Lag = 1 + ceil(Distance / InfoSpeed)` ticks, so at least 2. The local report of the trade phase is 1 tick old | N-106: a remote price is never as fresh as a local one, whatever the speed. The floor is a rule constant (INF-D1) |
| 6 | How inexact? | A bounded error in basis points, one integer draw per reader, source and commodity from the world's source, uniform and mean zero; no draw when the bound is 0. The draw order is fixed (5.5) | Independent errors are the only kind the model shows to help; a shared one hurts (11.2). A quantised price is the same for every reader, so it would herd them (5.5.4) |
| 7 | A staleness spread by agent (each agent reads a different vintage)? | Rejected | The model shows it lowers the stability limit of the shipping loop and does not reduce the swing of threshold shippers (11.1, 11.2). It would add an arbitrary rule on the agent ID for no gain |
| 8 | What if there is no news? | No rumour, and the reason is counted: no path (REG has none, or every route is closed), too old (the lag exceeds `RumourMaxAge`), not yet (the run is younger than the lag) | A closed road silences a region; that is the correct behaviour and must be visible (FR-STB-013) |
| 9 | Who is shown rumours? | A role flagged `Informed`, in the `Trade` view; the default is off, and then the module costs nothing per agent. `Work` and `Revise` do not carry them. One other reader: the migration pass asks for the rumours of the pushed household under evaluation whatever `Informed` says, through the same function and not through a view (6.1.3) | Almost every agent never ships; only a merchant needs them (6.1) |
| 10 | Does a trip carry news back? | No in version 1 | It needs a per-agent memory of what was seen where, which is state that grows with agents, regions and commodities (7.5, 17.1) |
| 11 | Is a cohort different? | No. A cohort is one agent: one view, one set of draws, prices per unit | `spec-scale.md` 4.4; the lockstep of a cohort is recorded (L13, L34) |
| 12 | Can the outside world be a source? | Only if EXT declares the pair; the record is then written from that market's published price. EXT never writes the record | The contract with EXT (4.5): one generic key, one writer |
| 13 | What does a rumour leave out? | Stock, volume, the best quotes, wages, welfare ratios. Prices only | N-106: no view carries a remote stock; a remote ratio is an aggregate MIG must justify (17.4) |
| 14 | Does a rumour reach a belief? | No. It is read by a strategy and by nothing else in the core | Beliefs are about the local market (`spec-beliefs.md` 3); a rumour in a belief would hide the loop it creates |
| 15 | Is this spec a faucet or a sink, or does it add a loop? | Neither a faucet nor a sink. It reads published prices and writes only its record, which no rule of the core reads, so it closes no loop of its own; it adds a delay and a blur to the loops of MER and MIG, recorded in section 13 | `spec-stability.md` 9.1 |

## 3. What an agent knows for free

3.1. **The local report.** In the trade phase the view of an agent carries, for each
commodity of its tradable set, the market report of its own region: the market price,
the price age, the best unmatched bid and ask with their flags, the effective
reference price, and the price window's mean, low and high with the position of the
price in it (`spec-agents.md` 6.4.1; `spec-clearing.md` 5.4, 5.5; `spec-beliefs.md`
7). That is the price of the previous tick's clearing, so the local report is **1 tick
old** when it is read. This spec adds nothing to it and takes nothing from it. It is
free because the agent is in the market; there is no delay, no error and no switch.

3.2. **Nothing else is free.** An agent is told nothing about the price of any
commodity in any other region, nothing about a commodity outside its tradable set even
in its own region, and nothing about the stock, the volume or the quotes of a remote
market. A rumour (section 5) is the only way it is told a remote price, and only if
its role is informed (6.1).

3.3. **A consumer is not an agent.** The `Snapshot` and the events carry every market's
true price (`MarketCleared`), and a consumer that wants to show them does so. The rule
of this spec is about what an *agent's strategy* is shown, because the strategy is
the agent (`spec-agents.md` 6.1.1). A strategy that a consumer writes receives only
its view; it has no handle to the world (`spec-agents.md` 6.5.4).

3.4. **The status of the local price as an input to the rest.** The local market price
is also what the beliefs of the region start from and are compared with
(`spec-beliefs.md` 8, 5.4), and nothing in this spec changes that.

## 4. The price record

### 4.1 Sources

4.1.1. A **source** is a pair (region, commodity) whose market can be the subject of a
rumour. The set of sources is derived at construction and fixed for the run:

- every pair (region, commodity) whose market is active, in a region that has at least
  one group (`spec-regions.md` 3.1.1: which markets are active is derived from the
  tradable sets of the roles that live there), and
- every pair that the external sector declares (4.5).

It is kept in ascending (`RegionID`, `CommodityID`) order. The number of sources `S`
is at most `MaxRegions * MaxCommodities`.

4.1.2. A pair that is not a source is never the subject of a rumour. In particular a
region with no agents is not a source unless EXT declares it (`spec-regions.md` 3.5).

### 4.2 The entry

4.2.1. For each source and each tick, the record holds one **entry**:

```go
// Observation is what a source market published in one tick. It is the unit of the
// price record. It holds no reader, no agent and no error: those are applied when a
// rumour is made (5.5).
type Observation struct {
	Price Price // the market price published that tick (spec-clearing.md 5.4)
	Mean  Price // the price window's mean after that tick's push (spec-beliefs.md 7.2); 0 if the window was empty
	Age   int64 // the price age published that tick: 0 if units traded, else ticks since they last did
}
```

`Price` is at least 1 whatever the tick, so `Mean` equal to 0 can mean only "no mean".
`Age` is the market's price age: a market that never traded publishes its seed price
(`spec-clearing.md` 5.4) with an age equal to the ticks run.

4.2.2. The entry of a tick in which nothing traded is the carried-forward price with a
positive age. It is stored all the same: the record is a record of what the market
published, and a stale price is told as stale (the age is in the rumour, 5.1).

### 4.3 When it is written

4.3.1. In phase 5 of the tick (`spec-agents.md` 4: outcomes), immediately after the
world pushes the price windows (`spec-beliefs.md` 7.4) and before the first outcome is
processed, the world appends one entry for tick `t` to the record of every source, in
ascending (`RegionID`, `CommodityID`) order. It reads the market's published price, price age
and the window mean as they stand after the push. It writes nothing else and emits no
event.

4.3.2. After the append of tick `t` the record of a source holds the entries of the
ticks `max(0, t - D + 1)` to `t`, with `D` = `RumourMaxAge`. A rumour made in phase 3
of tick `t + 1` can therefore read a vintage as old as `t + 1 - D`, which is a `Lag`
of `D` (5.3). A reader of phase 7 of tick `t` (the migration pass, 6.1.3) reads as of
tick `t + 1`: its vintage is `(t + 1) - Lag`, so the oldest vintage it can read is the
same `t + 1 - D` and the depth of the record is unchanged.

4.3.3. The record is empty at construction. It is **not** pre-filled with seed prices,
by a warm start or otherwise: until a market has published, there is nothing to tell
(5.3, `not_yet`). The first entry is tick 0's, written in phase 5 of tick 0.

### 4.4 Size

4.4.1. The record is a ring of `D` entries for each of `S` sources, in one slice
indexed by the position of the source and the tick modulo `D`. Its size does not grow
with the length of the run, the number of agents or the number of rumours made. For the
reference world of five regions with 17 active markets each and `D` = 60 it is 85 *
60 = 5 100 entries (under 125 KiB, section 12.3).

4.4.2. Validation bounds it: `S * D` is at most `MaxRumourCells` (9.2).

### 4.5 The seam with the external sector

4.5.1. `spec-regions.md` 3.5 and `spec-scale.md` 8 say that a region with supply and
demand functions and no agents is the outside world, a node of the region graph. Such a
region is a source only if EXT says so: EXT declares each (region, commodity) pair
it wants told, by a function of the construction that adds the key to the set of 4.1.1
(`Informs(region, commodity)`, internal, called once for each pair at construction). INF
provides the key and the record; EXT decides which keys.

4.5.2. A declared key must name a defined region and a defined commodity
(`ErrUnknownRegion`, `ErrUnknownCommodity`). A declaration for a pair that is already a
source changes nothing.

4.5.3. The record is written by the world from the **published market price of a
market**, whoever's orders fill it. EXT does not write an entry, a rumour or a lag, and
nothing outside the world can (FR-INF-006). If EXT presents the outside price to
agents in another way (a field of its own in the view), that field is EXT's and is not
in the record. What a rumour of an outside key means for the economy (an import price
that arrives late) is the same as for any other key.

4.5.4. The lag of a rumour whose source is an outside node uses the node's distance in
the region graph like any other region's (5.2). TRN moves goods to or from an outside
node only if EXT says so (EXT's brief); information does not depend on that.

### 4.6 What is not stored

No rumour is stored. No per-reader, per-agent or per-pair table exists: the lag of a
pair is a derived number (7.2), the rumour is computed at the moment a view is built,
and the only state of this spec is the record. That is what keeps the state bounded by
sources times the horizon and not by agents (position 3).

## 5. The rumour

### 5.1 The type

5.1.1. A **rumour** is plain data, made when a view is built and valid for the call
that receives it:

```go
// Rumour is what an informed agent has heard of a remote market. It is a reading of
// the price that market published Lag ticks ago, blurred by a bounded error; it is
// never the market's current price. It carries no stock, no volume and no quote.
type Rumour struct {
	Region    RegionID    // the region the news is about; never the agent's own
	Commodity CommodityID // the commodity
	Price     Price       // the market price at the vintage tick, with the error; 1 to MaxPrice
	Mean      Price       // the price window mean at the vintage tick, with the same error; 0 if there was none
	Lag       int64       // ticks from the vintage tick to this tick: at least 2 (5.2)
	SourceAge int64       // the source's price age at the vintage tick; 0 if units traded that tick
}
```

5.1.2. `Lag` and `SourceAge` are never changed by the error. They are the staleness
of the news, exposed with every rumour: `Lag` says how long ago the market published
it, and `SourceAge` how long before that a unit last traded (a carried-forward price
in a thin market has a positive `SourceAge`, and a market that has never traded has an
age equal to the ticks run). The total staleness of the underlying trade is
`Lag + SourceAge`.

5.1.3. A rumour has no hop count and no relay: news is not passed from region to
region (position 4). `Region` is the source.

### 5.2 The lag

5.2.1. For a source region `r` and a reader region `h` (different regions), let
`D = Distance(r, h)` be REG's shortest distance from `r` to `h` over the open edges
(`spec-regions.md` 4.3.1), in the pack's distance unit. Then

```
Lag(r, h) = 1 + ceil(D / InfoSpeed)  =  1 + (D + InfoSpeed - 1) / InfoSpeed
```

in integer arithmetic, with `InfoSpeed` in distance units per tick, at least 1
(section 8). Information therefore follows the direction of the edges, as goods do:
news from `r` reaches `h` only along a walk from `r` to `h`, and the lag from `r` to `h`
may differ from the lag from `h` to `r` (`spec-regions.md` 4.1.4).

5.2.2. `D` is at least 1 for different regions, so `ceil(D / InfoSpeed)` is at least 1
and `Lag` is at least **2**. The `1` is the tick of publication: a price cleared in
phase 4 of tick `t - 1` is the latest that anyone can read in phase 3 of tick `t`
(the migration pass of phase 7 of tick `t` reads as of tick `t + 1`, 4.3.2, so it too
finds that price the latest it can be offered), and it is what the local report is (3.1). A remote reading is therefore always at least
one tick older than the local one, however fast the news travels. The floor is a rule
constant and is not configurable (INF-D1 of section 13).

5.2.3. **Arithmetic.** `D` is at most `999 * MaxDistance * MaxEdgeFactorBP / 10 000` < 10^13
(`spec-regions.md` 4.2) and `InfoSpeed` at most `MaxInfoSpeed` = 10^9, so `D + InfoSpeed`
fits `int64` with a wide margin; the helper checks it all the same
(architecture.md 3.5). A result above `RumourMaxAge` is not narrowed: the comparison
is made on the `int64`.

5.2.4. **Why a lag from the shortest distance and not a relay.** The shortest distance
is a total-order, edge-order-independent function of the graph (`spec-regions.md`
4.3.1); a relay would need a rumour table for each pair of regions and commodity,
rounding at each hop, and a rule for which of two arriving rumours to keep. The price
of the choice is that news does not stop at an intermediate region: a trader in the
middle of a chain learns nothing from the news that passes over it (17.2).

5.2.5. **Distance is a proxy.** A distance is a length in the pack's unit, chosen for
goods (`spec-regions.md` 4.1.2). Where the unit is a real length, news at a stated
speed is a fair reading. Where it is an effort (the velocity change of the orbital
example of `spec-regions.md` 12.2), `InfoSpeed` is simply the pack's choice of how many
units of effort one tick of news covers; it has no other meaning. A pack whose news does
not follow its roads has no field for that in version 1 (17.3).

### 5.3 When there is no rumour

5.3.1. For an informed agent in region `h`, the **candidates** are the pairs (`r`,
`c`) with `r` different from `h`, `(r, c)` a source (4.1), and `c` in the agent's
tradable set (`spec-agents.md` 3.4). Each candidate is decided in this order, and the
first that applies wins:

| # | Case | Result | Count |
|---|------|--------|-------|
| 1 | `Distance(r, h)` is undefined: no walk over open edges from `r` to `h` | none | `NoPath` |
| 2 | `Lag(r, h)` is above `RumourMaxAge` | none | `TooOld` |
| 3 | `Lag(r, h)` is above the tick `t` of the reading (the current tick for a view; `t + 1` for the migration pass of phase 7, which reads as of the next tick, 4.3.2; the vintage `t - Lag` is before tick 0) | none | `NotYet` |
| 4 | otherwise | a rumour | `Offered` |

A closed edge is not in the graph (`spec-regions.md` 4.3.1), so a region cut off by a
closure has no path and no news from the other side; a region that is reachable only
by a longer way has a longer lag (7.2).

5.3.2. A candidate that gets no rumour consumes no less of the random stream than one
that does (5.5.2): the draw is taken for every candidate, so the stream does not depend
on the state of the graph or the age of the run.

5.3.3. The three counts are reported once per reader region per tick (10.1). An entry
the view does not carry is simply absent: a strategy looks it up with `Rumour(region,
commodity)` (6.7) and gets `false`.

### 5.4 The vintage

5.4.1. For an offered candidate the **vintage tick** is `v = t - Lag`, with `t` the
current tick (`t + 1` for the migration pass of phase 7, 4.3.2). The entry of tick `v` of the source's record is read (4.2): its `Price`,
`Mean` and `Age`. By 4.3.2 and the test `Lag <= RumourMaxAge`, the entry exists and has
not been overwritten.

5.4.2. The vintage depends on the reader's region and the source, and on nothing else:
two informed agents in one region read the same entry (and differ only by their
errors). The vintage moves when the lag changes (7.2).

### 5.5 The error

5.5.1. Let `E` = `RumourErrorBP`, in basis points, 0 to `MaxRumourErrorBP` (5 000).
For an offered candidate with entry `(p, m, a)`:

```
e      = -E + draw(2E + 1)                            uniform in [-E, E]; no draw if E = 0
Price  = clamp(halfEven(p * (10 000 + e) / 10 000), 1, MaxPrice)
Mean   = clamp(halfEven(m * (10 000 + e) / 10 000), 1, MaxPrice)    if m > 0, else 0
```

`halfEven` is the rounding of `spec-clearing.md` 5.2 (round half to even). The same
`e` multiplies both `Price` and `Mean`, so their ratio is kept to the rounding. Because
`E` is at most 5 000 and a price at most `MaxPrice` (10^9), `p * (10 000 + e)` is below
1.5 * 10^13 and fits `int64` with no helper; the clamp to 1 matters only for a price of
1 or 2. The error is unbiased in price (the mean of `e` is 0) and bounded:
`|Price - p| <= ceil(p * E / 10 000)`.

5.5.2. **The draw.** The error is the world's draw of `spec-world.md` 7.2.1 (one source
word, no loop, no rejection; architecture.md 7.3), one draw per candidate, **offered or not**, when `E` is above 0. The order is
fixed: the world builds the view of an informed agent immediately before it calls that
agent's `Trade`, and takes the draws at that moment in ascending `RegionID` of the
source and, within a region, ascending `CommodityID`. The migration pass of phase 7 is a
second reader and takes its own draws, in its own phase (6.1.3). Agents are visited in ID order
(`spec-agents.md` 4), so the whole sequence is a function of the configuration, the
world's state and the seed. The count of draws in a tick is the sum over the informed
agents given a view of their candidates (plus the migration pass's, 6.1.3), and with `E` = 0 it is zero: a world with no
informed role, or with `E` = 0, takes no draw for this module and its random stream is
the stream of a world without the module.

5.5.3. **Independent, not shared.** Each reader draws its own error. The model of
section 11.2 shows why: an error shared by all the shippers of a region in a tick
(one draw per tick) makes the swing of threshold shippers larger (2.2 to 2.4 against 1.8
times the freight with none), because it is a common shock; independent errors disperse
the shippers' decisions and shrink the swing, but only when `E` is a large part of the
margin they act on (0.4 times the freight cuts it by a quarter, 0.8 times by two thirds).
This is a weak damper and is recorded as one (INF-D2, INF-A2).

5.5.4. **Why not quantise.** Rounding a rumour to a band is deterministic and needs no
draw, but it gives every reader the same number, and a threshold shipper then sees
the same side of its threshold as its neighbours: it removes no herding, and it adds a
bias towards the band's centre. A deterministic offset by agent ID is the spread of
position 7, rejected. Quantisation is not offered; `E` = 0 is the deterministic mode.

5.5.5. The error is **not** a model of a liar. A rumour is the market's own published
price blurred by a symmetric error; nothing in this spec makes news systematically
wrong or lets an agent lie.

### 5.6 Order and uniqueness

A view carries at most one rumour for each (region, commodity), and its rumours are
sorted by ascending `Region` then `Commodity`. The number of candidates of an agent is
at most `MaxRumoursPerView` (9.2).

## 6. Who reads

### 6.1 The informed role

6.1.1. `Role.Informed` is a bool, default false, owned by this spec (`spec-agents.md`
3.3 lists the other fields of a role). With the module on, the agents of an informed
role are shown rumours; the agents of any other role are not, and nothing is computed
for them. Setting `Informed` while `Modules.Information` is nil is rejected
(`ErrModuleDependency`, 9.1): the flag would do nothing.

6.1.2. The default is off because almost no agent can act on a remote price. A
household that makes and sells in its own market has no use for one; a merchant that
may carry goods between regions does. The cost of the module is proportional to the
number of informed agents and not to the population (7.8).

6.1.3. **A second reader: the migration pass.** The migration module (`spec-migration.md` 4.3.1, 6.2) obtains rumours for the pushed household it is evaluating, for the sources of the commodities its role needs, **whatever `Informed` says**. It calls the same rumour function a view uses (the same lag, vintage, error and `RumourMaxAge`, 5.2 to 5.5) and builds no view: a strategy is shown nothing, and the household's role does not become informed. Its error draws are taken in the migration pass of phase 7, after the entry step of every region, among the pushed households in ascending agent ID, one per candidate (a source pair of a candidate region), in ascending source `RegionID` and then `CommodityID`, offered or not, and none when `E` is 0 or no household is pushed (`spec-migration.md` 6.2.1, 10.2.1; N-111). They are never taken in phase 3, so the view draws and the migration draws cannot interleave. `RumoursServed` does not count these calls (10.1). The statement of 6.1.1 that the agents of any other role are not shown rumours is about views.

### 6.2 The view

6.2.1. `AgentView` (`spec-agents.md` 6.4.1) gains one group:

| Group | Content |
|-------|---------|
| `Rumours` (only with the module on, in the view passed to `Trade` of an agent of an informed role) | the rumours of 5, a slice sorted by (`Region`, `Commodity`), a scratch copy valid for the call only |

It is empty for every other agent. The view passed to `Work` carries no rumours, and
the `OutcomeView` passed to `Revise` has none: a rumour is an input to the trade
decision, and no draw is taken for a call that does not get them. The field type is
`[]Rumour`, a slice in the world's scratch, reused for every agent and valid for the
call only like the other slices of the view (`spec-agents.md` 6.4.1); a strategy
**MUST NOT** keep it.

6.2.2. The default strategy ignores the group (it ships nothing, `spec-agents.md`
6.3). The MER strategy reads it with `Routes` and `Inbound` (`spec-transport.md` 9.1).
A consumer's strategy may read it for its own purposes.

### 6.3 Who is given a view

An agent that is away, or on a migration journey (`spec-transport.md` 10.5), is passed
no view (`spec-agents.md` 4, `spec-transport.md` 9.1), so it gets no rumours and no draw
is taken for it. The estate pseudo-agent has no role
and is never given a view (`spec-estates.md` 2.7.4). An external agent (a player) is
informed if its role is (`spec-agents.md` 10).

6.3.1. **An external agent's last rumours are kept.** A query cannot build rumours,
because building them takes draws (6.1.1). So the world keeps, for an external agent of
an informed role with the module on, a copy of the rumours of its last view and the
tick of that view, at most `MaxRumoursPerView` (`spec-integration.md` 4.7.3). They
influence no result, and they are saved in a checkpoint (14; `spec-checkpoint.md` 4.2
tag 22), because `Agent` of the integration spec answers from them.

### 6.4 Cohorts

An agent of weight `w` above 1 is one agent: one view, one set of candidates, one draw
for each. A rumour is a price, and a price is per unit; it is not multiplied by `w`
(`spec-scale.md` 4.4, `spec-beliefs.md` 9.1). A cohort of shippers reads one rumour, so
its households ship in lockstep by construction (L13, L34). The first consumer's world
needs no weight above 1 (N-030).

### 6.5 Limits

The number of candidates of an informed agent is at most its tradable set
(`spec-agents.md` 3.4) times the number of other regions, and fewer where a region has
no source for a commodity. Validation rejects a configuration in
which an informed role could exceed `MaxRumoursPerView` (9.2).

### 6.6 What a strategy must not assume

That the lag of a pair is constant (a status change moves it, 7.2); that the set of
rumours is the same from one tick to the next (a closure removes some, a reopening
restores them, a young run has none); that the error is small or that it is zero; that a
rumoured price is a price at which it can trade (it is an old reading of someone
else's market). These are properties of the news, not defects of the library.

### 6.7 Helper

```go
// Rumour returns the rumour about the commodity in the region, and false if the
// view carries none (no path, too old, not yet, not a source, or an agent that is not
// informed). It does not allocate.
func (v *AgentView) Rumour(region RegionID, c CommodityID) (Rumour, bool)
```

A binary search over the sorted slice. It is a method on a plain-data type and adds no
interface (architecture.md 4.2).

## 7. In the tick, and what changes it

### 7.1 Where it happens

Tick phases are `spec-world.md`'s (6); this spec depends on these points of the order
of `spec-agents.md` 4, which `spec-world.md` fixes:

| Phase | What this spec does |
|-------|---------------------|
| 3, trade decision | For each agent in ID order that is informed and neither away nor on a migration journey: build the rumours of its view (5.3 to 5.5, with the draws), then call `Trade`. After the last agent's plan is executed and before phase 4, emit `RumoursServed` for each reader region in ascending `RegionID` (10.1) |
| 5, outcomes | After the price windows are pushed and before the first `Revise`, append one entry to the record of every source (4.3) |
| 7, entry and exit | The migration pass reads the rumours of the pushed households (6.1.3), as of tick `t + 1`, with its own draws, after the entry step of every region (`spec-migration.md` 10.2) |

Phase 7 has a second reader (6.1.3): the migration pass asks for the rumours of each pushed household and takes their draws; it writes no entry. Nothing else in the tick touches this module. In particular no rumour is made in phase
1, 2, 6 or 8, and in phase 7 only by the migration pass, and the record is not read in phase 5 or 6 of the tick it is
written in: a rumour made in phase 3 of tick `t` reads entries of ticks `t - 2` and
earlier, and the migration pass of phase 7 of tick `t` reads as of tick `t + 1`, so its
entries are those of ticks `t - 1` and earlier (`spec-world.md` 6.12, C17).

### 7.2 A change of the graph

7.2.1. The lag of each ordered pair of regions is a table of `R * R` `int32` values
(`R` regions), derived from REG's distance table and `InfoSpeed` when the world is built
and again, once, after each successful `SetEdgeStatuses` (`spec-regions.md` 7.3.2,
4.4.4). A pair with no path holds a marker. The table is derived and is not saved
(14).

7.2.2. A status change takes effect at the next view built, the next phase 3. A
closed edge that removes the only walk from `r` to `h` turns every rumour from `r` to
`h` into `NoPath` at once; a closure that leaves a longer walk raises the lag, and the
**vintage steps back** (the example of 12.1: the lag from `middle` to `lower` goes from
2 to 5, so at tick 100 the vintage goes from 98 to 95); a reopening or a shorter edge
steps it forward. The reader sees the step as one tick's change of the news, not as a
gradual ageing. This is deterministic, and it is a hazard for a shipper that compares
rumours of two successive ticks (INF-A4).

7.2.3. A rumour already read is not changed by a later status change: the view is
built per tick. A shipment dispatched on a rumour is TRN's and is not affected.

### 7.3 Relocation, removal, role change

The reader region is the region of the agent's account at the moment its view is
built. A relocation (`spec-regions.md` 6) therefore changes the rumours the agent is
shown from the next view, with nothing to move or reseed: no per-agent state exists.
Removal and weight change touch nothing of this spec. A role change to or from an
informed role changes whether the agent is shown rumours from its next view.

### 7.4 Beliefs and orders

A rumour is not an input to `Revise`, to the belief update, to the limit price draw or
to the cost anchor; none of them reads it (position 14). It reaches the economy only
through what a strategy does with it: an order placed in the agent's own market, a
shipment request (TRN), a decision to move (MIG).

### 7.5 A trip carries no news back

A traveller (`spec-transport.md` 5) keeps its beliefs across a trip and remembers
nothing else (N-100, `spec-transport.md` 9.3). In version 1 its arrival changes no
record and no rumour. At the destination it is in the local market and reads the local
report like any agent there; it does not carry the origin's price with it, and on its
return it has no memory of what it saw. The reasons: a memory of prices seen is state
per agent, region and commodity, which grows with agents (position 3) and would have to
be saved in a checkpoint; and a traveller is the one agent that could otherwise read a
true remote price as of its visit, which N-106 forbids a view to carry. Whether MER
needs such a memory is open (17.1).

### 7.6 The start of a run, and a checkpoint

At tick 0 every record is empty, so every candidate is `NotYet` until the run is as old
as the lag (5.3, 4.3.3). A world restored from a checkpoint carries its records
(14) and continues exactly as if it had not stopped.

### 7.7 Changing the module's parameters

`InfoSpeed`, `RumourErrorBP` and `RumourMaxAge` are configuration and are in the hash
(`spec-config.md` 8). They are not changed in a run. Whether a shock should be able to
slow the news (a blackout) is INT's and is open (17.6).

### 7.8 Cost per tick

Placeholder counts, not timings (N-015). Appends: `S` entries per tick (85 for the
reference world). Views: an informed agent costs its candidates, at most
`MaxRumoursPerView`; the reference world with 40 informed agents of 8 tradable
commodities in 4 other regions has 1 280 candidates and, with `E` above 0, 1 280 draws
a tick (12.3). The lag table is read, not computed. A world with the module on and no
informed role costs `S` appends and nothing else; with the module off the cost is
zero.

## 8. Parameters

All are fields of `Config` with a default (or "required"), a bound, a doc line and an
owner (architecture.md 9.5 to 9.7). **Every default is a placeholder** (N-015, N-027):
no source for how fast or how exactly news travels was read. The bounds are validation
bounds in `bounds.go` (`spec-config.md` 4.1).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Information` | the module: present means on (`spec-config.md` 3.2) | nil (off) | |
| Information | `InfoSpeed` | distance units that news covers in one tick (5.2) | required | 1 to `MaxInfoSpeed` |
| Information | `RumourErrorBP` | half-width `E` of the error of a rumour, in basis points (5.5) | 500 (placeholder) | 0 to `MaxRumourErrorBP` |
| Information | `RumourMaxAge` | the horizon: the longest `Lag` that is offered, and the depth of the price record (4.4, 5.3) | 60 (placeholder) | 2 to `MaxRumourAge` |
| Role | `Informed` | the agents of the role are shown rumours (6.1) | false | a bool |

`InfoSpeed` has no default because a speed in a unit chosen by the pack has none that is
right for every setting (as `Mode.Speed` of `spec-transport.md` 11). A missing key is the
zero value, which is outside the bound (`ErrOutOfBounds`). `RumourErrorBP` of 500 (5%)
is of the order of the freight shares of the worked example of `spec-transport.md` 17.4
(6% for a full unit to 24% for a small cargo), so that the error is a fraction of the
margin a shipper acts on (11.2); it is a guess. `RumourMaxAge` of 60 is two months of
daily ticks or ten days of ticks of four hours; it is a number of ticks and is not tied
to the calendar. Both are to be calibrated by STY and CLB once a shipper exists.

**New bounds for `bounds.go`** (provisional, with the reasons):

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxInfoSpeed` | 10^9 | as `MaxSpeed` of `spec-transport.md`; `D + InfoSpeed` stays far below `int64` (5.2.3) |
| `MaxRumourErrorBP` | 5 000 | the factor `(10 000 + e) / 10 000` is at least 0.5, so a rumoured price is never 0 or negative, and `MaxPrice * 15 000` fits `int64` (5.5.1) |
| `MaxRumourAge` | 10 000 | a tenth of `MaxTicksPerYear`: a rumour older than that is not news, and the ring depth is bounded |
| `MaxRumourCells` | 4 000 000 | `S * RumourMaxAge`, the entries of the record; at 24 bytes an entry that is under 100 MB at the bound, and a realm of 100 regions with 30 commodities at depth 60 uses 180 000 |
| `MaxRumoursPerView` | 4 096 | the candidates of one informed agent: bounds the loop of a view and the draws it takes |

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the floor
of 2 on `Lag` (the `1 +` of 5.2.1); the uniform distribution of the error and the order
of the draws (5.5.2); the half-even rounding; the order of the candidates (5.3, 5.6);
the reason constants `no_path`, `too_old`, `not_yet` (counts of 5.3.1), exported
constants of the root package (`event.go`).

**Not parameters:** the lag table (7.2, derived), the vintage (derived), the sources
(derived, 4.1), the record (state, section 14).

The placeholders, so that none is mistaken for a result: the default of
`RumourErrorBP`, the default of `RumourMaxAge`, every bound above, and every number of
the worked examples (section 12) and of the two models (section 11).

## 9. Validation

Each rule has a sentinel or a warning name and a failing-case test (testing.md 3.9).
Errors name the path, for example `modules.information.info_speed`. The framework of
`spec-config.md` 7 runs them.

### 9.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | `InfoSpeed` below 1 (a missing key included) or above `MaxInfoSpeed`; `RumourErrorBP` above `MaxRumourErrorBP`; `RumourMaxAge` below 2 or above `MaxRumourAge` |
| `ErrTooMany` | `S * RumourMaxAge` above `MaxRumourCells`; an informed role whose candidates could exceed `MaxRumoursPerView` (9.2) |
| `ErrModuleDependency` | `Role.Informed` true while `Modules.Information` is nil |
| `ErrUnknownRegion`, `ErrUnknownCommodity` | a source declared by EXT (4.5.2) that names an undefined region or commodity; the rule is stated here so that the key has one meaning, and EXT's validation raises it |

No new sentinel. The module needs the region graph, which is core and always present,
so there is no dependency to check beyond the flag.

### 9.2 Derived checks

These are computed at construction from the configuration (`spec-config.md` 4.2 is
the home of the list):

- the number of sources `S` (4.1) times `RumourMaxAge` is at most `MaxRumourCells`;
- for each informed role, the largest number of candidates in any region, the size of
  the role's tradable set times the number of other regions (a safe upper bound that
  ignores which regions have a source), is at most `MaxRumoursPerView`.

### 9.3 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at
construction, at `Warn`, with the attributes `rule` and `path`; the volume is bounded
by the configuration.

| Warning | Condition |
|---------|-----------|
| `information_without_readers` | the module is on and no role is `Informed`: records are written and no one reads them |
| `information_without_routes` | the module is on and there are fewer than two regions or no edge: no pair of regions can have a rumour |
| `rumour_horizon_short` | at least one ordered pair of regions has a walk in the configured graph (no edge status applied) whose `Lag` exceeds `RumourMaxAge`, so no rumour is ever offered for it; carries `pairs`, the count |
| `informed_without_sources` | an informed role none of whose tradable commodities is a source in any region other than those where it has a group: it would never be shown a rumour |

## 10. Observability

10.1. **Event.** One, an aggregate in the style of `UpkeepShortfall`, a state of the
tick and not of one agent (observability-and-logging.md 4.1, 4.3). Plain data, no map,
no pointer, no wall-clock time, constructed only when an observer was supplied (4.5
there).

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `RumoursServed` (new) | once per reader region per tick in which the module is on and at least one informed agent was given a view in that region, in ascending `RegionID`, after the last plan of phase 3 is executed and before phase 4 | `Region` (the reader's), `Readers` (informed agents given a view), `Offered`, `NoPath`, `TooOld`, `NotYet` (counts of candidates by the result of 5.3.1, summed over the readers), `MaxLag` (the largest `Lag` among the offered, 0 if none) |

It counts the readers of views only: the calls of the migration pass (6.1.3) are not in `Readers` or in the counts; the migration module counts its own outcomes (`spec-migration.md` 13.1). It is bounded by the number of regions, so it is not per agent. It is the counter of
three silent limits: the closed road (`NoPath`), the horizon (`TooOld`) and the young
run (`NotYet`), as FR-STB-013 asks of every limit. A tick with no informed view in a
region emits nothing for it. The `Region` field carries the typed `RegionID`; the
strings come from the names table of `spec-observability.md` 3.2.

10.2. **Logs.** Constant lowercase messages, typed attributes, `Enabled` guards
(observability-and-logging.md 3.1.3).

- `Info`: nothing new. The construction summary line adds `information` to its
  `modules` list (the key exists).
- `Warn`: at construction, one per warning of 9.3, with `rule` and `path`. Nothing
  inside the tick.
- `Debug`: per informed view, "rumours served", with `agent`, `region`, `rumours` (the
  number offered) and `lag` (the largest offered). Guarded by `Enabled`: building the
  attributes costs nothing when the level is off. No line per draw and none per
  rumour.
- No `Error`.

10.3. **New log keys** for observability-and-logging.md 3.3.2: `rumours` (a count of
rumours offered to one agent) and `lag` (a lag in ticks). `region` and `agent` exist.

10.4. **Metric.** One instrument, fed by `RumoursServed`, proposed for the catalogue
of observability-and-logging.md 5.3:

| Name | Instrument | Unit | Attributes |
|------|-----------|------|------------|
| `economysim.information.rumours` | Counter | `{rumour}` | `economysim.outcome` |

`economysim.outcome` takes the four values `offered`, `no_path`, `too_old` and
`not_yet` and is a new attribute, bounded by a constant set (5.4.1 there lists the
attributes allowed; this one is added to the list). The counter is fed by the four
counts of `RumoursServed`. It has four series whatever the configuration. Nothing in
the metric names an agent, a region or a price.

10.5. **Snapshot.** Nothing is added. The true price of every market is already in the
snapshot and in `MarketCleared`; the record is state for a checkpoint, not for
observation (14). OBS may add the per-region count of sources if a consumer wants it.

10.6. **No observer, no logger.** The record is written, the lag table read and the
rumours made whether or not anyone watches; no event is constructed without an
observer. No log, event or metric takes a draw, reads the clock or changes a rumour
(observability-and-logging.md 1.4), and the draws of 5.5.2 do not depend on whether an
observer is attached.

## 11. What was run

Three scripts in `doc/specs/models/inf/`, run in Docker (`python:3.12-slim`, the
standard library only, `PYTHONDONTWRITEBYTECODE=1`), as the README of that folder says.
They are **prototypes of the rules, not of the library**: floating point inside the
model only, a linearised gap or a stylised shipper, no inventory, no belief, no quantity
sizing and no market. The shipper's own rule is MER's and is invented here for the model.

### 11.1 The lag in the shipping loop (`lag.py`, `lag.out`)

It extends the model of `spec-transport.md` 16 (the gap `x` between a dear region B and
a cheap region A, a shipper in A that dispatches `q = g * observed gap`, goods that reach
B `d` ticks later, a share `a` of the effect at dispatch). The shipper reads A's price
now and B's as a rumour. `ell` below is the extra lag of the rumour over the local
report, `ell = Lag - 1 = ceil(D / InfoSpeed)`. With `d` = 2 and all the effect at arrival
(`a` = 0) the largest gain `g` that decays, on a grid of 0.002:

| `ell` | 0 | 1 | 2 | 3 | 4 | 6 | 8 |
|-------|---|---|---|---|---|---|---|
| model | 0.618 | 0.444 | 0.346 | 0.284 | 0.240 | 0.184 | 0.148 |
| theory `2 sin(pi / (2 (2 (ell + d) + 1)))` | 0.618 | 0.445 | 0.347 | 0.285 | 0.241 | 0.185 | 0.149 |

So the rumour's lag adds to the transit time: the delay of the loop is `ell + d`, and
the limit on the gain falls as `pi / (2 (ell + d) + 1)` for long delays. With half the
effect at dispatch (`a` = 0.5; the purchase moves the origin price at once) the limit
stays at 1.998 for `ell` of 0, 1 and 2, 1.994 at 4 and 1.976 at 8: the local part of the
signal is immediate and dominates. **A spread of vintages** over the shippers (shipper `j`
reads `ell + j`, `j` = 0 to `S`, equal shares) does not help: with minimum `ell` = 1 the
limit is 0.444, 0.396, 0.360 and 0.308 for `S` of 0, 1, 2 and 4, between the limit of a
point lag at the mean (0.346 and 0.284 for `S` of 2 and 4) and at the minimum (0.444).
A distribution of lags is better than its mean and worse than its minimum, so a spread
lowers the limit that no spread has (position 7).

### 11.2 Threshold shippers and the error (`shippers.py`, `shippers.out`)

`N` = 50 shippers in A ship one lot when the rumoured gap exceeds the freight `f`; a lot
moves the gap by `Q / N` where `Q` = 0.5 `f` is what all of them shipping at once would
move in a tick; the lot arrives 2 ticks later; the rumour is 2 ticks old; the gap is
pulled back to 2 `f` by 10% a tick, so the steady flow that holds it at `f` is 0.1 `f` a
tick (10 lots of 50). The size of the gap is its peak-to-peak range over the last half of
6 000 ticks, in units of `f`:

| Shippers | Size of the gap |
|----------|----------------:|
| identical (no error, one vintage) | 1.80 `f` |
| a spread of vintages, `S` of 1, 2, 4, 8 | 1.89, 2.11, 2.15, 2.04 `f` |
| independent error `E` of 0.05, 0.1, 0.2, 0.4, 0.8 `f` | 1.99, 1.94, 1.75, 1.32, 0.60 `f` |
| one error shared by all, `E` of 0.2 and 0.8 `f` | 2.22, 2.43 `f` |
| a static spread `w` of the shippers' own bars, 0.1, 0.2, 0.4, 0.8 | 1.74, 1.59, 1.01, 0.04 `f` |
| identical, `Q` of 0.25, 0.1 `f` (a cap on what can be shipped) | 0.98, 0.41 `f` |
| independent error 0.4 `f`, longer delays (rumour 6, transit 4) | 3.04 `f` |

Over eight seeds the rows with an error vary by under 0.15 `f` (for example 1.21 to 1.32
at `E` = 0.4 `f`, 0.53 to 0.68 at 0.8 `f`). The number of shippers at a fixed `Q` hardly
matters (1.99 at 5, 1.66 at 1 000, error 0.2 `f`). What the run says:

1. Identical threshold shippers that read the same news swing the gap by almost twice
   the freight: that is herding (INF-A2). It is not removed by spreading what they read.
2. An independent error disperses the decisions of shippers, so the aggregate response
   to the gap has a slope `Q / (2E)` per unit of gap near the freight instead of a step.
   The swing falls to a settled gap only when that slope is below the limit of 11.1 for
   the total delay `n = ell + d`: `Q / (2E) < 2 sin(pi / (2 (2n + 1)))`, here 0.346. The
   static spread of bars shows it cleanly (slope 0.625 gives 1.01 `f`, slope 0.312 gives
   0.04 `f`); the redrawn error settles to a residual jitter of 0.25 to 0.6 `f` at
   N = 50 because it is redrawn each tick.
3. The cheap, strong levers are not in this spec: a cap on what can be shipped
   (`spec-transport.md` D39) and a spread of the shippers' own thresholds (MER, a fixed
   draw at creation, no cost per tick) reduce the swing far more than any error of a
   size that is still a rumour. INF's error is a weak damper (INF-D2), and a shared one is
   worse than none.

### 11.3 The arithmetic of the examples (`worked.py`, `worked.out`)

The integer arithmetic of section 12: shortest distances, the lag tables, the vintages,
the bounds of the error and its rounding, the sizes of the first consumer's record.

### 11.4 What they do not show

That the Go code is right (a test must); that any default is stable; the rule of any
real shipper; interaction with the beliefs of the markets involved or with the price of
the haul unit; seasons; more than two regions; the cost of the draws.

## 12. Worked examples

Two settings, each different from the other, chosen to show the module with a slow
channel and with a fast one. Both are **examples** with placeholder numbers; names are
invented. The arithmetic is in `models/inf/worked.py`. The graphs are those of
`spec-regions.md` 12.1 and 12.2.

### 12.1 Example A: a river valley (a pre-industrial setting; unit: one league)

Regions `lower` (0), `middle` (1), `upper` (2), with the edges of `spec-regions.md`
12.1 (shortest distances, row from, column to: `lower` 0, 55, 120; `middle` 30, 0, 70;
`upper` 70, 40, 0). One informed role, `broker`, trading `staple` and `fuel`.

```toml
[modules.information]
info_speed      = 40      # leagues of distance that news covers in one tick
rumour_error_bp = 500     # plus or minus 5%
rumour_max_age  = 60

[role.broker]
informed = true
```

`Lag(r, h) = 1 + ceil(Distance(r, h) / 40)`, source `r` in the rows, reader `h` in the
columns:

| source \ reader | `lower` | `middle` | `upper` |
|-----------------|--------:|---------:|--------:|
| `lower`  | - | 3 | 4 |
| `middle` | 2 | - | 3 |
| `upper`  | 3 | 2 | - |

The lags are not symmetric, as the distances are not: news from `lower` takes 4 ticks to
reach `upper`, news from `upper` takes 3 to reach `lower`. All lags are far under the
horizon of 60, so there is no `rumour_horizon_short`.

**A reading.** The record of `staple` at `lower` holds, for ticks 90 to 99 (invented
figures; price, price age, window mean): 358/0/359, 360/0/359, 361/0/360, 365/0/361,
365/1/362 (a tick with no trade), 372/0/363, 370/0/365, 368/0/366, 371/0/368, 372/0/369.
At tick 100 a `broker` of `middle` reads the vintage 100 - 3 = 97: price 368, age 0, mean
366; a `broker` of `upper` reads the vintage 100 - 4 = 96: price 370, age 0, mean 365.
With `E` = 500 the broker of `upper` is shown a price between 352 and 388 and a mean
between 347 and 383 (370 * 9 500 / 10 000 = 351.5, which rounds half to even to 352;
370 * 10 500 / 10 000 = 388.5, which rounds to 388). If its draw were `e` = -230 it would
see price 361 and mean 357, with `Lag` 4 and `SourceAge` 0. The local price it sees at
`upper` is the previous tick's market price, 1 tick old; the news from `lower` is 4.

**A closure.** `SetEdgeStatuses(shock, [middle > lower closed])` gives the shortest
distances `lower` 0, 55, 120; `middle` 160, 0, 70; `upper` 90, 40, 0
(`spec-regions.md` 12.1). The lags into `lower` become `middle` 5 (was 2) and `upper` 4
(was 3). At tick 100 a broker of `lower` reads the vintage 95 about `middle` (was 98) and
96 about `upper` (was 97): the news steps back by 3 and 1 ticks at once (7.2.2, INF-A4).
Nothing is silenced: a path remains. If the edges `upper > middle` and `upper > lower`
were both closed, no walk would lead from `upper` to anywhere, and every broker would
have `NoPath` for `upper`.

### 12.2 Example B: an orbital colony (a space setting; unit: velocity change, m/s)

Regions `belt` (0), `orbital` (1), `surface` (2), with the edges of `spec-regions.md`
12.2. The distance is an effort, not a length, so `InfoSpeed` is the pack's choice of
how much effort one tick of news covers (5.2.5); here 4 000. The error is 0: this
colony's markets post their prices on a shared board and a rumour is exact but late, so
no draw is taken at all.

```toml
[modules.information]
info_speed      = 4000
rumour_error_bp = 0
rumour_max_age  = 6

[role.freight_broker]
informed = true
```

Lags (source in the rows):

| source \ reader | `belt` | `orbital` | `surface` |
|-----------------|-------:|----------:|----------:|
| `belt`    | - | 2 | 3 |
| `orbital` | 2 | - | 2 |
| `surface` | 5 | 4 | - |

News from the well's floor takes longest to get out (5 ticks to `belt`) and news into it
is quick. With a practically instant channel (`info_speed` 10^9) every lag is 2: the
floor of 5.2.2, the one tick of publication and one tick that no speed removes. With
`rumour_max_age` of 4 the warning `rumour_horizon_short` fires with `pairs` = 1
(`surface` to `belt`, lag 5), and a broker in `belt` is told nothing about `surface`:
every candidate is `TooOld`. Without the edge `orbital > surface` (a colony that can
send and not receive), `surface` hears nothing from `belt` or `orbital` (`NoPath`),
while they still hear `surface` at lags 5 and 4.

### 12.3 The first consumer's world

Five regions with 17 active markets each: 85 sources and, at `RumourMaxAge` 60, 5 100
entries, about 119 KiB at an upper figure of 24 bytes an entry. If 40 agents of
informed roles each trade 8 commodities that are sources in the 4 other regions, there
are 1 280 candidates a tick and, with `E` above 0, 1 280 draws, which is nothing beside
the 5 000 agents' own work. The lag table is 25 values.

## 13. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9 (FR-STB-003). Fields as in
its section 3; the register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn`
numbers, and the entries here are `INF-An`, `INF-Dn` and `INF-Sn`.

**This spec closes no loop of its own.** By 9.1 of the checklist: it reads the published
price and the window mean of a market and the distance between two regions, and what it
writes (the record) is read by no rule of the core; a rumour reaches the economy only
through a strategy's orders, shipment requests and decisions to move. The loops that
follow belong to MER and MIG. This spec adds a delay and a blur to them, a rule that
removes any delay below two, one honest limit (the horizon) and one weak damper (the
error). The entries below record the delay it adds, the herding it permits, and three
hazards. The MIG and INF row of `spec-stability.md` 4.8 (migration on rumoured ratios) is
not answered here: this version carries prices only (17.4).

**INF-A1. The rumour lag in the price-gap loop.** *Kind:* loop (extends TRN-A1, STB-L84).
*Loop:* the gap between a dear and a cheap region (+) the rumour of the dear region's
price, `ell` ticks older than the local report (+) the quantity a shipper dispatches
(+, after the transit time `k`) the arrivals in the dear region (-) its price (+) the
gap; and the dispatch moves the origin price at once (-) the gap. Each path has one
negative link, so each is balancing (B); the arrival path has the delay `ell + k`.
*Delay:* `ell + k`, with `ell = Lag - 1 = ceil(D / InfoSpeed)` (5.2) and `k` TRN's travel
time. *Owner:* INF 5.2; the response (the gain) is MER's. *Damper:* D38 (a shipper that
counts `Inbound` removes `k` from its signal but not `ell`), D39 (the dispatch cap), D40
(the freight as a floor on the gap); INF-D3 bounds `ell`. *Undamped:* a shipper whose gain
is high and whose rumour is old. *Condition:* in the model of 11.1, with all the effect
at arrival, the gap decays if `g < 2 sin(pi / (2 (2 (ell + k) + 1)))`: for `k` = 2, 0.618
at `ell` 0, 0.444 at 1, 0.346 at 2, 0.240 at 4, 0.148 at 8; with half the effect at
dispatch the limit is about 2 up to `ell` of 8. *Test:* INF-S1. *Status:* model (11.1). It
does not show the library: no inventory, belief, sizing or market, and the share of the
effect at dispatch is not derived for any pack. *Neighbouring delays (checklist 9.6):*
`ell + k` against a perishable's `ShelfLife` (a gap that closes in fewer ticks than
`ell + k` is chased after it is gone; TRN-A5, STB-L86), against the belief adjustment (a
few ticks, `spec-beliefs.md` 5.2, 5.6) and against INF-A5.

**INF-A2. Herding of threshold shippers.** *Kind:* amplifier. *Loop:* every informed
shipper of a region reads the same news with at most a blur: the rumoured gap (+) the
number of shippers whose reading exceeds the freight (a step, not a slope) (+) the lots
dispatched (-) the gap, after the delay of INF-A1. Without dispersion the aggregate
response to the gap is a step of height `Q` (all shippers at once), which is a limit
cycle. *Owner:* INF 5.5; the thresholds are MER's. *Damper:* INF-D2 (an independent error,
weak: it settles the cycle only when `Q / (2E)` is below the limit of INF-A1, so `E` must
be a large part of the margin); D39 and the cap on the dispatch; a spread of the
shippers' own thresholds (MER, a function of the agent ID, `spec-merchants.md` 5.7.2; far stronger in the model, 0.04 `f` against 1.8 `f` for
identical shippers at `w` = 0.8, 11.2); D38. *Undamped:* identical shippers with a large
`Q` against the steady flow: a swing of almost twice the freight in the model (`Q` 0.5
`f`, steady flow 0.1 `f`). *Condition:* `Q / (2E) < 2 sin(pi / (2 (2n + 1)))` with `n` the
total delay, where `Q` is the change of the gap if every shipper ships and `E` the
half-width of the dispersion of what they act on (an error, or their own bars); or a cap
that keeps `Q` near the steady flow. A common error is worse than none (11.2), so the
error is independent per reader (5.5.3). *Test:* INF-S1. *Status:* model (11.2), a single
shipper type and rule, `N` = 50.

**INF-A3. A phantom gap from a stale source price.** *Kind:* hazard (the cost of
STB-L67, REG-A2). A thin source market that rarely trades publishes its carried-forward
price with a growing `Age`; a rumour of it carries that price and a `SourceAge`, and a
gap against it can persist for ever: shippers ship into a market whose price does not
move because no one trades there, and the goods arrive to sell below the rumoured price or
to find no buyer. *Loop:*
none by itself; it feeds INF-A1 a signal that is not news. *Owner:* INF 5.1.2 (the age is
exposed); the use is MER's. *Damper:* none in this spec by design: the `SourceAge` field
is the means, and a strategy that discounts or ignores a rumour with a large age is the
damper (MER). `Lag` is also exposed. *Undamped:* a strategy that ignores `SourceAge`.
*Condition:* a shipper acts on a rumour only if `SourceAge` is small against the time
its goods take. *Test:* INF-S2. *Status:* hand.

**INF-A4. The vintage steps when the graph changes.** *Kind:* hazard. A status change
(`spec-regions.md` 7) moves the lag of every pair whose shortest walk it touches, and a
reader's vintage jumps by the difference at once (12.1: from 98 to 95). A shipper that
differences two successive rumours reads a spurious price move: a step back looks like a
fall or a rise depending on the direction of the trend, for one tick. *Owner:* INF 7.2.
*Damper:* none; the event `EdgeStatusChanged` says when, and a strategy that compares
rumours across ticks is warned (6.6). *Undamped:* a strategy that uses the change of a
rumour as a signal. *Condition:* a strategy does not difference rumours of successive
ticks. *Test:* INF-S3. *Status:* hand.

**INF-A5. A lag near half a seasonal period.** *Kind:* hazard (a case of STB-L23). If the
price of a commodity swings with a season (a harvest), a rumour whose lag is a large
share of the period shows the opposite phase: at half a period the sign of the gap is
reversed, and shippers move goods the wrong way. The phase error is `2 pi * Lag /
period`, under a quarter turn (45 degrees) when `Lag` is under an eighth of the period.
*Delay:* the lag against `TicksPerYear / seasons`. *Owner:* INF 5.2; CAL 4.5.1 for the
period. *Damper:* INF-D3 (the horizon) bounds the lag; nothing else. A pack with a coarse
tick (a month) and a slow speed can have a lag of half a year. *Undamped:* any pack
whose largest lag is near half a season. *Condition:* the largest `Lag` in the realm is
under an eighth of the shortest seasonal period of a traded commodity. *Test:* INF-S5.
*Status:* hand (a phase argument, untested). There is no validation for it in version 1: it
needs the seasonal profile of prices, which no spec holds.

**Dampers this spec adds** (for the catalogue of `spec-stability.md` 5; each meets
5.1):

| ID | Damper | Home | Status |
|----|--------|------|--------|
| INF-D1 | The lag floor: a remote reading is at least 2 ticks old, one older than the local report, whatever `InfoSpeed`. Cost: no instant arbitrage information, even with a practically infinite speed | INF 5.2.2 | specified (a rule constant; visible in every `Lag`) |
| INF-D2 | The independent error: each reader's own bounded blur disperses threshold shippers, and a shared blur is worse than none. Cost: draws, and shipments made on noise; weak unless `E` is a large part of the margin | INF 5.5 (`RumourErrorBP`) | specified (counted: nothing silent, the error is in the rumour; a draw count is in 5.5.2) |
| INF-D3 | The horizon: no rumour older than `RumourMaxAge` is offered, and the record is that deep. Cost: far regions and slow channels are invisible | INF 5.3 (`RumourMaxAge`) | specified (counted by `TooOld`, warning `rumour_horizon_short`) |

**Scenarios proposed** (for section 8 of the register; REF builds them):

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| INF-S1 | Two regions with a route, an informed merchant role that ships on a rumoured gap, the dispatch cap on and off, a rumour lag `ell` of 1, 2 and 4 and several travel times, and with and without an error | the gap does not oscillate with growing amplitude for a gain under the limit of INF-A1; with a gain above it, the cap bounds the swing; with `E` large against the margin the swing falls as INF-A2 says. Waits for MER and TRN | INF-A1, INF-A2, L84 |
| INF-S2 | A thin source market (its price age grows) and a dear remote region | the rumour carries the source's age; a merchant that ignores it ships and its goods arrive without moving the price; one that reads it does not. The first assertion is runnable now with a recording strategy | INF-A3 |
| INF-S3 | A three-region chain; a closure for a stated number of ticks, then a reopening; a recording strategy that logs the rumours it is shown | the rumours from a cut-off region are absent while it is cut off and `NoPath` is counted; the vintage steps by the change of lag at the change; after the reopening the rumours return with the old lags; `EdgeStatusChanged` and `RumoursServed` agree. Runnable now | INF-A4, REG-A1 (STB-L66) |
| INF-S4 | The reference world with an informed role and `E` = 500, run for 200 ticks, checkpointed at 80 and resumed | the resumed run is bit-identical; the draws taken equal the sum of the candidates of the informed views; with `E` = 0 no draw is taken and the stream equals the module-off stream. Runnable now | determinism, FR-INF-014, FR-INF-028 |
| INF-S5 | A seasonal price swing in the source region, a lag of an eighth, a quarter and half of the period | the phase of the rumoured gap against the true gap is shifted by the lag; at half a period the sign is reversed (a measurement of the signal, not of the economy). Runnable with a recording strategy | INF-A5, L23 |

**Checklist 9.7 and 9.8.** The tunables of the dampers are `RumourErrorBP` and
`RumourMaxAge` (section 8: default, owner, bound, doc line, row in the parameter-owner
table, placeholder note). The new entries are handed over in the handoff for the register
owner to merge; INF-A1 should be merged with STB-L84 and STB-L85 (the delay) as a case, and
INF-A5 with STB-L23.

## 14. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, plain data in a fixed
order with no maps:

- for each source in ascending (`RegionID`, `CommodityID`) order, the number of entries
  held (0 to `RumourMaxAge`) and the entries from the oldest to the newest: `Price`,
  `Mean`, `Age`;
- for each external agent of an informed role that has had a view, in ascending agent
  ID, the tick of its last view and the rumours given to it, in the order of the view
  (`spec-checkpoint.md` 4.2 tag 22; 6.3.1).

The ring's write position is not saved: it is the tick modulo the depth, and the entries
are written oldest first, so a restore rebuilds the same ring. The set of sources is
derived from the configuration (and EXT's declarations), so it is not saved; a restore
whose derived set differs from the saved one is rejected as a configuration mismatch
(the hash covers the configuration, `spec-config.md` 8.1).

**Not saved** (derived): the lag table (7.2), the sources, the candidates and the
rumours themselves, except the last rumours of an external agent (6.3.1), and the
scratch slice of a view. The error needs no state: the draws
come from the world's random source, whose state the world saves. A world restored
between ticks makes the same rumours with the same errors as an uninterrupted one
(FR-INF-028). The module's parameters and `Role.Informed` are configuration and are in
its hash.

## 15. Notes settled

For `AGENTS.md`: each open note tagged `INF` or `ALL`, and the others that name rumours,
is applied, rejected or left open here, with the text to append in the handoff.

| Note | Decision | Where |
|------|----------|-------|
| N-046 Register entries owed by unwritten specs | **Leave open** for the other specs, with this added: INF has its entries (13) | 13 |
| N-084 Scenarios that wait for trade | **Leave open**, with this learned: INF supplies the rumours, so X1 and X2 (dynamic) now wait for MER and EXT only; the overlay needs `modules.information` and an informed merchant role (12.1 shows the keys) | 12, 13 |
| N-100 What transport leaves to MER and INF | **Applied for INF; leave open** for MER and CAR: a traveller keeps its beliefs and remembers nothing else, and a trip carries no news back (7.5); rumours are defined (5) | 5, 7.5 |
| N-104 Region-keyed tables use the one override mechanism | **Leave open** (the others still owe it), with this added: INF adds no region-keyed table. The speed of news is world-wide; a region that is slower is expressed by longer distances; a per-region speed would be a row in the table of `spec-regions.md` 5.1.3 in scale mode, keyed by the reader's region (17.9) | 8, 17.9 |
| N-106 What a strategy may see of other regions | **Applied for INF; leave open** for MER and TRN: no view carries a true remote price (the lag is at least 2 and the price blurred, 5.2.2, 5.5); the field is `Rumours` of the trade view of an informed role (6.2); MER's shipper reads rumours and `Routes` and counts `Inbound` | 5, 6 |
| N-016 Keep the parameter registry true | **Applied**: rows for `Modules.Information`, `InfoSpeed`, `RumourErrorBP`, `RumourMaxAge` and `Role.Informed` (8, handoff b); no unnamed parameter | 8 |
| N-017 Words and names | **Applied**: rumour, price record, vintage, lag, information speed, informed role, source; setting words appear only in the flagged examples | 5, 12 |
| N-027 No default macro constant is stable | **Applied**: the spec sets no default that is a price, wage or rate, and every default is a labelled placeholder | preamble, 8 |
| N-103 Running several spec writers in parallel | **Applied**: no shared file was edited, proposals are in the handoff, provisional IDs are used (`INF-A1` to `INF-A5`, `INF-D1` to `INF-D3`, `INF-S1` to `INF-S5`); the model ran in Docker with no bytecode | 13 |
| N-107 New money faucets and sinks start from MON | **Reject as not applicable**: INF creates and destroys no money and no goods and adds no ledger reason | 2 (position 15) |
| N-012 Events, metrics and log keys | **Leave open**, with this added: the event `RumoursServed`, the log keys `rumours` and `lag`, the metric `economysim.information.rumours` with the new attribute `economysim.outcome` (10) | 10 |
| N-020 What each spec contributes to a checkpoint | **Leave open**, with this added: the price record per source (14) | 14 |
| N-021 Anything that creates or destroys money or goods | **Leave open**, with this added: INF adds no faucet, sink or reason | 2 |
| N-038 Seed prices, newcomers and migrants | **Leave open**, with this added: a migrant has no rumour state to reseed or keep; its rumours are those of its new region from its next view (7.3) | 7.3 |
| N-045 and N-065 Dampers that act silently | **Leave open**, with this added: the three limits of this spec are counted by `RumoursServed` (`NoPath`, `TooOld`, `NotYet`) | 10.1 |
| N-009 What the world and tick phases must decide | **Leave open**, with this added: the view of an informed agent is built, with its draws, immediately before its `Trade`; the record is appended in phase 5 after the windows are pushed (7.1) | 7.1 |
| N-091 The world slots INT must expose | **Leave open**, with this added: INF has no slot; a blackout shock would be one (17.6) | 17.6 |
| N-015 Placeholders to calibrate | **Leave open**, with this added: `RumourErrorBP` 500, `RumourMaxAge` 60, the bounds of 8 and every number of 11 and 12 | 8 |

Proposed new notes are in the handoff.

## 16. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-INF-001 | With `Modules.Information` nil, the world MUST keep no record, make no rumour, take no draw for this module and emit no `RumoursServed`, and a configuration with `Role.Informed` true MUST be rejected with `ErrModuleDependency` (6.1.1, 9.1). |
| FR-INF-002 | The sources MUST be the pairs (region, commodity) whose market is active in a region with at least one group, plus the pairs declared by EXT, in ascending (`RegionID`, `CommodityID`) order; a region with no group and no declaration MUST contribute none (4.1). |
| FR-INF-003 | A declared source naming an undefined region or commodity MUST be rejected with `ErrUnknownRegion` or `ErrUnknownCommodity`, and a declaration of a pair that is already a source MUST change nothing (4.5.2). |
| FR-INF-004 | In phase 5, after the price windows are pushed and before the first outcome is processed, the world MUST append exactly one observation to the record of every source, in ascending (`RegionID`, `CommodityID`) order, including in a tick in which nothing traded (4.2.2, 4.3.1). |
| FR-INF-005 | An observation MUST hold the market price published that tick, the price window's mean after the push (0 if the window was empty) and the published price age (4.2.1). |
| FR-INF-006 | The record MUST be written only by the world from a market's published price: no exported function, option, strategy method or other module MUST be able to append to it or change it (4.5.3). |
| FR-INF-007 | The record MUST hold at most the last `RumourMaxAge` observations of each source, MUST overwrite the oldest, MUST be empty at construction and MUST NOT be filled from seed prices or a warm start (4.3.2, 4.3.3). |
| FR-INF-008 | The lag from a source region `r` to a reader region `h` MUST be `1 + ceil(D / InfoSpeed)` with `D` the shortest effective distance from `r` to `h`, in checked integer arithmetic, and MUST be at least 2 (5.2.1, 5.2.2, 5.2.3). |
| FR-INF-009 | The lag MUST follow the direction of the edges: on a graph whose distances from `r` to `h` and from `h` to `r` differ, the lags MUST differ accordingly (5.2.1). |
| FR-INF-010 | A candidate with no walk over open edges from the source to the reader's region MUST get no rumour and be counted as `NoPath`, and a closed edge MUST NOT carry news (5.3.1). |
| FR-INF-011 | A candidate whose lag exceeds `RumourMaxAge` MUST get no rumour and be counted as `TooOld` (5.3.1). |
| FR-INF-012 | A candidate whose lag exceeds the current tick MUST get no rumour and be counted as `NotYet` (5.3.1). |
| FR-INF-013 | A candidate MUST be decided by the first of the cases of 5.3.1 that applies and counted exactly once; a candidate with no path and a lag above the horizon MUST be `NoPath` (5.3.1). |
| FR-INF-014 | When `RumourErrorBP` is above 0, the world MUST take one draw `Int64N(2E + 1) - E` from its source for every candidate of an informed agent, offered or not, in ascending source `RegionID` and then `CommodityID`, immediately before that agent's `Trade`, with agents in ID order; and when it is 0 it MUST take none (5.5.2, 5.3.2). The draws of the migration pass are separate (FR-INF-051). |
| FR-INF-015 | A rumour's `Price` MUST be `halfEven(p * (10 000 + e) / 10 000)` clamped to `[1, MaxPrice]` and its `Mean` the same with the same `e` when the entry's mean is above 0 and 0 otherwise, so that `|Price - p|` is at most `ceil(p * E / 10 000)` (5.5.1). |
| FR-INF-016 | `Lag` and `SourceAge` of a rumour MUST be the lag of 5.2 and the observation's age, unchanged by the error (5.1.2). |
| FR-INF-017 | A rumour MUST be made from the observation of the vintage tick `t - Lag` of the source's record (5.4.1). |
| FR-INF-018 | No rumour in any view MUST have a `Lag` below 2 or a vintage later than `t - 2`, whatever `InfoSpeed` (5.2.2, N-106); a rumour read by the migration pass of phase 7 of tick `t`, which reads as of tick `t + 1`, MUST have a `Lag` of at least 2 against `t + 1` and so a vintage not later than `t - 1` (4.3.2). |
| FR-INF-019 | The `Rumour` type MUST have exactly the fields `Region`, `Commodity`, `Price`, `Mean`, `Lag` and `SourceAge`, and no stock, volume, quote or hop count (5.1.1, position 13). |
| FR-INF-020 | A view MUST carry at most one rumour per (region, commodity), sorted by ascending `Region` then `Commodity`, never about the agent's own region and only about commodities of its tradable set (5.3.1, 5.6). |
| FR-INF-021 | Only the `Trade` view of an agent of an informed role MUST carry rumours; the `Work` view, the `OutcomeView` and the views of other agents MUST carry none and cause no draw (6.2.1). The migration pass obtains rumours by the function and not through a view (FR-INF-051). |
| FR-INF-022 | An agent that is away or on a migration journey, and an estate pseudo-agent, MUST be given no view, no rumour and no draw (6.3). |
| FR-INF-023 | An agent of weight above 1 MUST get one view and one set of draws, and its rumoured prices MUST NOT be multiplied by its weight (6.4). |
| FR-INF-024 | `AgentView.Rumour(region, commodity)` MUST return the view's rumour and `true`, or the zero value and `false` when there is none, and MUST NOT allocate (6.7). |
| FR-INF-025 | With a strategy that ignores rumours, a world with the module on MUST give the same trades, prices and balances as one with it off, apart from the configuration hash and the saved record, and no belief update, limit-price draw or cost anchor MUST read a rumour (6.2.2, 7.4). |
| FR-INF-026 | After a successful `SetEdgeStatuses` the lag table MUST be rebuilt once with the graph's tables, the next phase 3 MUST use the new lags, and a rumour already built MUST NOT change (7.2). |
| FR-INF-027 | A relocation MUST change only the reader region of the rumours from the agent's next view, and move or reseed no state (7.3). |
| FR-INF-028 | A trip, an arrival and a shipment MUST change no record entry and no rumour (7.5). |
| FR-INF-029 | The checkpoint MUST contain, per source in ascending order, the number of observations held and the observations from the oldest to the newest, and nothing else of this spec; and a world restored between ticks MUST make the same rumours and take the same draws as an uninterrupted one (14). |
| FR-INF-030 | The lag table, the set of sources, the candidates and the rumours MUST NOT be saved in a checkpoint (14). |
| FR-INF-031 | An `InfoSpeed` outside `[1, MaxInfoSpeed]` (a missing key included), a `RumourErrorBP` above `MaxRumourErrorBP` and a `RumourMaxAge` outside `[2, MaxRumourAge]` MUST be rejected with `ErrOutOfBounds` naming their path (8, 9.1). |
| FR-INF-032 | An absent `RumourErrorBP` MUST default to 500 and an absent `RumourMaxAge` to 60, and `Role.Informed` MUST default to false (8). |
| FR-INF-033 | A configuration whose sources times `RumourMaxAge` exceed `MaxRumourCells` MUST be rejected with `ErrTooMany` (9.2). |
| FR-INF-034 | A configuration with an informed role whose bound on candidates (tradable set times other regions) exceeds `MaxRumoursPerView` MUST be rejected with `ErrTooMany` (9.2). |
| FR-INF-035 | The warning `information_without_readers` MUST be raised when the module is on and no role is informed (9.3). |
| FR-INF-036 | The warning `information_without_routes` MUST be raised when the module is on and there are fewer than two regions or no edge (9.3). |
| FR-INF-037 | The warning `rumour_horizon_short`, carrying `pairs`, MUST be raised when an ordered pair of regions has a walk in the configured graph whose lag exceeds `RumourMaxAge`, with `pairs` the number of such pairs (9.3). |
| FR-INF-038 | The warning `informed_without_sources` MUST be raised for an informed role none of whose tradable commodities is a source in a region other than where it has a group (9.3). |
| FR-INF-039 | The warnings of this spec MUST NOT fail validation, MUST each be logged once at `Warn` with `rule` and `path` at construction, and MUST NOT be logged inside the tick (9.3, 10.2). |
| FR-INF-040 | `RumoursServed` MUST be emitted once per reader region per tick in which at least one informed agent was given a view there, in ascending `RegionID`, after the last plan of phase 3 and before phase 4, and never otherwise (10.1). |
| FR-INF-041 | The counts `Offered`, `NoPath`, `TooOld` and `NotYet` of `RumoursServed` MUST add up to the candidates of the informed views of that region and tick, `Offered` MUST equal the rumours placed in them and `MaxLag` the largest of their lags (10.1). |
| FR-INF-042 | `RumoursServed` MUST contain no map or pointer, MUST be constructed only when an observer was supplied, and a run with and without a logger and an observer MUST give equal results (10.1, 10.6). |
| FR-INF-043 | Each informed view MUST be logged at `Debug` only, as "rumours served" with `agent`, `region`, `rumours` and `lag`, guarded by `Enabled`, and no `Info` MUST be logged inside the tick (10.2). |
| FR-INF-044 | The configuration hash MUST cover the fields of `Modules.Information` and `Role.Informed`, so that changing any of them changes the hash (8, 14). |
| FR-INF-045 | The file MUST write the module as `[modules.information]` with the keys `info_speed`, `rumour_error_bp` and `rumour_max_age`, and the flag as `informed` in the role table; an unknown key MUST be rejected with `ErrUnknownKey`, a non-integer value with `ErrNotInteger`, and a file and the same configuration built in Go MUST have equal hashes and runs (8). |
| FR-INF-046 | For a given seed, configuration and sequence of calls, the rumours and the draws MUST be identical across runs and platforms (architecture.md 7.4), checked by a golden test on a world with an informed role and `RumourErrorBP` above 0 (5.5.2). |
| FR-INF-047 | With `RumourErrorBP` 0, or with no informed role, the module MUST take no draw, and the world's random stream and every result MUST equal those of a world with the module off, apart from the hash and the saved record (5.5.2). |
| FR-INF-048 | The bounds `MaxInfoSpeed`, `MaxRumourErrorBP`, `MaxRumourAge`, `MaxRumourCells` and `MaxRumoursPerView` MUST be constants in `bounds.go` with the values of section 8 (8). |
| FR-INF-049 | The module MUST create or destroy no money or goods and add no ledger reason: with the module on, every conservation check and every flow total MUST equal that of a world with it off (position 15). |
| FR-INF-050 | Two informed agents in one region at one tick MUST read the same observation for a pair (the same vintage, `Lag` and `SourceAge`) and MAY differ only in `Price` and `Mean` (5.4.2). |

### Non-functional

| ID | Requirement |
|----|-------------|
| FR-INF-051 | The migration pass MUST be able to obtain the rumours of the pushed household under evaluation through the rumour function whatever `Informed` says, with the same lag, vintage, error and `RumourMaxAge` as a view, MUST take its error draws only in phase 7, among the pushed households in ascending agent ID and, for each, one per candidate in ascending source `RegionID` then `CommodityID`, offered or not, and none when `RumourErrorBP` is 0 or no household is pushed; it MUST NOT build a view or make the role informed (6.1.3). |
| FR-INF-052 | `RumoursServed` MUST NOT count the calls of the migration pass (6.1.3, 10.1). |
| NFR-INF-001 | Appending the record, building the rumours of a view and `AgentView.Rumour` MUST NOT allocate in a tick with the default logger and observer (observability-and-logging.md 8.2). |
| NFR-INF-002 | The per-tick cost SHOULD be `O(S)` for the append and `O(P)` for an informed agent's view (`P` its candidates, at most `MaxRumoursPerView`), and the draws and rumours of the 40 informed agents of the reference world (1 280 candidates) SHOULD take under 0.5 ms a tick. The figure is a placeholder and unmeasured (N-015). |
| NFR-INF-003 | The lag, the distortion and the ring MUST live in `internal/rumour` over primitive slices, import the standard library only, and know no region, commodity or role (architecture.md 2.9, 9.1). |
| NFR-INF-004 | The lag function and the distortion MUST have fuzz tests (testing.md 7.4) asserting that they never panic for any distance, speed, price or error, that the lag is at least 2 for a distance of at least 1, that the distorted price is within `[1, MaxPrice]` and within the bound of FR-INF-015, and that overflow is reported and not wrapped. |

## 17. Open questions

1. **Does a trip carry news back?** Not in version 1 (7.5). A trader that visits a market
   and returns has seen the true price, which a rumour does not give. A bounded form is
   possible: a record, per informed agent, of the last price seen in each region for its
   tradable set (at most `MaxRumoursPerView` entries, saved in a checkpoint) and its
   tick. Decided in `spec-merchants.md` 3.5: the merchant keeps no memory in version 1.
2. **Relay through intermediate regions.** News goes from the source to the reader in
   one step (5.2.4). A relay would make a region on the way a source for its neighbours
   and let a hub learn earlier than a leaf; it needs a table per pair of regions and
   commodity. Next step: add it if MER or a scenario shows that a hub's advantage
   matters; it would be a second module mode, not a change of the first.
3. **News that does not follow the roads.** The lag uses the distance of goods (5.2.5).
   A pack whose roads are slow and whose couriers are not (or the reverse) has no field
   for it. A field `InfoLength` on an edge, or a second graph for news, would be REG's
   and INF's together. Next step: when a pack needs it.
4. **Other signals.** Only prices are rumoured. MIG wanted the welfare ratio elsewhere
   (`spec-welfare-and-wages.md` 5.1, "rumoured conditions elsewhere") and SKL wants the
   remote premium (`spec-skill-barriers.md` 2.5). Both are aggregates measured in a region.
   A second record kind keyed by (region, signal) with the same lag and error would carry
   them; whether an agent may react to a regional aggregate is a question for the roadmap's
   "macro is measured, never set": a price is the market's own output, a mean ratio is an
   indicator. Answered for MIG (wave 4): a household does not use a regional mean ratio; it derives a destination's income from rumoured prices (`spec-migration.md` 4.4.1, 6.3); INF stays prices only. SKL's premium stays open.
5. **What a stateless strategy can do with a noisy rumour.** A strategy keeps no state
   (`spec-agents.md` 6.1.4), so it cannot average several ticks of news; `Mean` helps
   a little. MER reads `min(Price, Mean)` only (`spec-merchants.md` 5.2). A smoothed field (a mean of the last `n` observations of the
   source before the vintage), which the record can give for free, is not needed by MER or MIG in version 1.
   Linked: a fixed spread of the shipper's own thresholds is a stronger damper than any
   error (11.2) and is MER's: a function of the agent ID, not a draw at creation (`spec-merchants.md` 5.7.2).
6. **A blackout shock.** A slot to slow or cut the news for a time (a war, a storm) would
   be a world slot like the edge status (`spec-regions.md` 7), set between ticks, with an
   event. Not in version 1; INT's, with N-091.
7. **Informed by role or by agent.** The flag is on the role. A player agent that should
   see rumours while its role's other agents should not needs a flag per agent, which is
   `AddAgent`'s (INT, N-022). Next step: INT.
8. **A start with news.** The record is empty at tick 0, so a run starts blind for the
   length of the longest lag (4.3.3). A warm start could pre-fill it with the seed prices
   and an age; the first ticks would then show the reference-price gaps as news. Decided
   against for version 1; revisit if S22 (a warm start against a cold one) shows the blind
   start matters.
9. **A region that is slower.** A per-region speed would be a row in the table of
   `spec-regions.md` 5.1.3 in scale mode, keyed by the reader's region (N-104). Not added:
   a longer distance does the same for goods and news together.
10. **Measure the cost.** The cost figures (7.8, NFR-INF-002) are counts. Measure them
    in the slice with an informed role of the size of the reference world's merchants.

## 18. Where this lives in the code

| Concern | Home |
|---------|------|
| `Rumour`, `Observation` (unexported), the module's configuration (`Information` and its three fields), `Role.Informed`, the sources, the lag table, the append in phase 5, the building of a view's rumours in phase 3, `AgentView.Rumour` | root package, `information.go` |
| The view group `Rumours` | root package, `strategy.go` (`spec-agents.md` 6.4) |
| The lag from a distance and a speed, the distortion with half-even rounding and the clamp, the ring of observations, over primitives | `internal/rumour` (standard library only, converted inline at the call, architecture.md 2.9) |
| The event `RumoursServed` | root package, `event.go` |
| The bounds of section 8 | root package, `bounds.go` |
| Validation, the derived checks of 9.2 and the warnings of 9.3 | root package, `validate.go` |
| The reason constants `no_path`, `too_old`, `not_yet` and the log keys `rumours` and `lag` | `internal/logkeys` |
| The file shape | `config/` (`spec-config.md` 6) |
