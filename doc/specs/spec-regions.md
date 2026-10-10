# Spec: Regions: the region, the directed graph between regions, and moving an account between them

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-REG-NNN` and
`NFR-REG-NNN`, which tests cite (testing.md section 4). They are in section 16.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)). A
world always has at least one region; a world with one region and no edges is the
degenerate case that every per-market spec already assumes. Needs
`spec-currency.md` (5, the region ID and the reference basket), `spec-goods.md` (6.3,
the effective reference price), `spec-ledger.md` (3, the account and its region),
`spec-estates.md` (2.7, one estate per region), `spec-scale.md` (3, 8, groups and
regions with no agents), `spec-config.md` (3.3, 3.4, 4, 7, ids, canonical order,
bounds, validation), `spec-production.md` (6, yield by region), `spec-agents.md`
(5.2, extra upkeep) and `spec-viability.md` (9, lanes).

**Every number in sections 4.2, 7, 8 and 12 is a placeholder** except where a
derivation is shown. Nothing has been run against the library, bazaarBot has not
been run, and no acceptance band exists (notes N-015, N-027). `models/reg/paths.py`
(section 14) checks the definitions of section 4.3 on small graphs; it measures no
speed and is not a claim that any default is stable. No default of this spec is a
macro constant: the spec sets no price, wage or rate, and reads none.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer, never a household (N-001, N-017). The examples of
section 12 come from three different settings and are flagged as examples; the
core and its vocabulary are setting-agnostic.

## 1. Purpose and scope

A realm is a set of places that each have their own market, linked by routes.
This spec fixes the places and the links, and nothing that moves along them. It
covers:

- what a **region** is: one settlement and one market, its identity and its order
  (section 3), and the creation of the estate that each region has (3.6);
- the **region graph**: **edges** that are directed and carry a **distance** in a
  generic unit, their bounds, and the rules for self, duplicate, asymmetric and
  one-way edges (4.1, 4.2);
- the deterministic queries over the graph, their total-order tie-break and their
  cost (4.3, 4.4), connectivity, islands and regions with no agents (4.5, 3.5), and
  the precise read-only view that the later specs take (4.6, 4.7);
- the **override** of a world or commodity value by region as one mechanism, and
  the one override this spec adds, the reference price (section 5);
- the operation that moves an account from one region to another, what it
  guarantees and the hook for goods in transit (section 6);
- the one dynamic part of the graph, the **edge status** that a shock sets: closed,
  or longer (section 7);
- parameters, validation, observability, checkpoint state, register entries and
  requirements (sections 8 to 11, 13, 16).

Out of scope, with the spec that owns it: **travel time**, speed, capacity,
vehicles, goods and travellers in transit, spoilage and wear on the road
(`spec-transport.md`, TRN); loss on the road (`spec-risk.md`, RSK); who ships and why
(`spec-merchants.md`, MER); what is known about another region and how late
(`spec-information.md`, INF); why and when a household moves (`spec-migration.md`,
MIG); the endowments of a region (`spec-resources.md`, RES); the size class of a
settlement (`spec-settlements.md`, SET); realms and jurisdictions over regions
(`spec-polity.md`, POL); the outside world as a source of prices and volumes
(`spec-external-sector.md`, EXT); the tick phases (`spec-world.md`, WLD). A distance
is not a duration: this spec has no word for time except where it says when a call
may happen.

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is a region? | One settlement and one market: for each commodity at most one market, one estate, one set of region fields. A town and its hamlets are separate regions joined by short edges | N-026, ADR 0003 decision 4. Clearing cost grows with regions times commodities, not with households (`spec-scale.md` 6), so the split is cheap; a split inside a region would break the one-price-per-market rule of `spec-goods.md` 7.1 |
| 2 | Can regions come and go in a run? | No. The set of regions is fixed at construction. Agents come and go; a region with no agents left is still a region | Every table keyed by region (the ledger's flow totals, the price windows, the estate pools) stays dense and the checkpoint stays small. A region founded in a run is a later extension |
| 3 | What is an edge? | A directed route `From`, `To`, `Distance`, with an integer `Distance` of at least 1 in a unit the pack chooses. `a > b` and `b > a` are different edges and may differ | The brief of the carriage market is one market per directed route (`spec-carriage.md` 2.2); asymmetry (a river, a gravity well, a one-way pass) is a fact of some settings and costs nothing to allow |
| 4 | Does a distance mean time? | No. It is a length in a generic unit. Travel time, cost and capacity are TRN's, derived from the distance and a mode | One owner per fact. The same graph serves a walking pack, a sailing pack and a pack of jump gates |
| 5 | Must the graph be connected? | No. Any set of edges is valid, including none. A region cut off is a closed economy, and a warning says so | A single region with no edges is the degenerate case (roadmap); a closed region is the scenario S14 and the `famine` variant. Forcing connectivity would forbid both |
| 6 | Which route between two regions? | The shortest by total distance. Ties are broken by the smallest region index at each step, so the path is the lexicographically smallest of the shortest paths. The rule is stated on the result, not on an algorithm | A total order that does not depend on how edges were listed, on a map or on the algorithm (architecture.md 7.5, 7.6) |
| 7 | How are regions ordered? | By ID string, bytewise, after canonicalisation. The index in that order is the `RegionID`; every iteration over regions is by ascending `RegionID` | `spec-config.md` 3.3, 3.4; clearing order is region then commodity (`spec-clearing.md` 7.2) |
| 8 | How does a region differ from the world's defaults? | By one mechanism: a row keyed by region and the subject in the table of the subject's owner, combined by replace, add or scale. This spec states the mechanism and owns the reference-price table | The overrides were scattered over five specs. The rule is the same for all of them, so it is written once (section 5) |
| 9 | Can an account change region? | Yes, an agent account, by one explicit operation, between ticks, with no reservation open. It moves no money and no goods between accounts, so it is not a transfer and has no reason. It is counted and reported | `spec-ledger.md` 3.5. The decision to move is MIG's and INT's; this spec guarantees what the move does to the ledger |
| 10 | Can the graph change in a run? | Only by an edge status: closed, or a length factor, set between ticks by a shock. Edges are not added or removed and base distances do not change | The first consumer wants a closed road (`vision.md` 2). Everything else static keeps the tables derived from configuration |
| 11 | Is this spec a faucet or sink, or does it add a feedback loop? | Neither. It reads no price, stock, income or count, so it closes no loop of its own; it adds hazards (section 13) and supplies the distances that the loops of TRN, INF and MIG will depend on | `spec-stability.md` 9.1 |

## 3. The region

### 3.1 What a region is

3.1.1. A **region** is a settlement with one market. It is an entity of the
configuration with an ID, a position in the graph and a set of parameters. It
holds:

- one **market** for each commodity: the double auction of `spec-clearing.md`,
  with its published price, price age and price window. A market exists for a
  commodity when some agent of the region can trade it; which markets are active is
  derived (`spec-reference-chain.md` 4.4.4 counts 17 per region for `five`) and is
  not this spec's;
- one **estate**, the pseudo-agent and the ledger account of `spec-estates.md` 2.7
  (3.6);
- the agents whose account is in it. An agent has no region field of its own: the
  region is on its ledger account (`spec-agents.md` 3, `spec-ledger.md` 3.5), and
  "the agents of a region" is derived from the accounts, in agent-ID order;
- the parameters of 3.2.

3.1.2. A region is not a country, a province or a polity. Nothing above a region
exists in this spec; a grouping of regions into jurisdictions is POL's and is a
table that names regions, never a field that regions read (5.4).

3.1.3. **One settlement per region (N-026, ADR 0003 decision 4).** There is no
second level inside a region: no ward, no hamlet that shares its market. A setting
that has a town with outlying hamlets writes the town and each hamlet as regions
joined by short edges (3.4). `spec-settlements.md` (later) only gives a region a size
class derived from its households; it neither splits nor merges markets.

### 3.2 Fields and who owns them

This spec owns the entity and the fields marked REG. The other fields of a region
are owned by the spec named; this spec only lists them, so that a reader has the
whole region in one place, and states how they combine with a default (5.1). A field
owned elsewhere is not restated: its meaning, default and bound are there.

| Field | Meaning | Owner |
|-------|---------|-------|
| `ID` | the region's identity: `[a-z][a-z0-9_]*`, at most 64 bytes, unique among regions | `spec-currency.md` 5 (the owner row of the parameter table); syntax `spec-config.md` 3.3; the index 3.3 here |
| `Label` (p) | display name; presentation only, not hashed | REG |
| `ReferenceBasket` | the cost of one household's baseline upkeep for one tick, at the region's effective reference prices | `spec-currency.md` 5 |
| `ExtraUpkeep` | goods added to every household's upkeep in the region | `spec-agents.md` 5.2 |
| `CapitalReturn` | the return a tool or training must beat to attract an entrant | `spec-skill-barriers.md` 2.5 |
| `EstateMarkdown`, `PoolRelease`, `ReliefBelow` | the estate's markdown, release rate and relief threshold | `spec-estates.md` 3 |
| reference-price overrides | `RegionPrices` rows for the region (section 5) | REG (values); the commodity's default is `spec-goods.md` 6.3 |
| yield profiles | `Yields` rows for the region | `spec-production.md` 6 (values move to `spec-resources.md`) |
| taste multipliers | `Modules.Demand.RegionTastes` rows for the region | `spec-demand.md` 3.2 |
| groups | `Groups` rows for the region: the households of each role | `spec-scale.md` 3 |
| warm start | `WarmStart` rows for the region | `spec-viability.md` 8 |
| edges | `Edges` rows with the region as `From` (4.1) | REG |

A region carries no coordinates, no area and no size: its position is only its
place in the graph. A setting that wants a map has its own data; the core does not
read it.

### 3.3 Identity and order

3.3.1. The regions of a `Config` are sorted by ID, bytewise, before a world is
built or a hash is taken (`spec-config.md` 3.4). The **`RegionID`** of a region is
its index in that order, from 0. It is a small named value type (architecture.md
3.3):

```go
// RegionID is the index of a region in ascending order of its ID string.
// It is not stable across configurations: adding a region can renumber the
// others. Use the string ID to persist or exchange a region.
type RegionID int

// NoRegion means "no region". A query returns it where a region does not exist,
// and a later spec may use it for an agent that is travelling (6.4).
const NoRegion RegionID = -1
```

3.3.2. Every iteration over regions in the library, and every output ordered by
region, is by ascending `RegionID`. This fixes: the order in which markets clear
(region, then commodity: `spec-clearing.md` 7.2), the order of events that carry a
region, the order of rows in a snapshot, the order of pseudo-agent IDs (3.6) and the
order of every table keyed by region in a hash or a checkpoint. No iteration over a
map of regions may influence a result (architecture.md 7.5).

3.3.3. The string ID is what a consumer, a file, a log line and a checkpoint of a
pack use. The `RegionID` is what the engine's arrays use. `Graph.Lookup` and
`Graph.Name` convert (4.3).

### 3.4 A town and its hamlets (N-026)

3.4.1. A town with outlying hamlets is a set of regions: the town and one region per
hamlet, joined by edges in both directions whose distance is short against the
setting's other edges (example C, section 12). Each hamlet has its own market and
price, its own estate and pool, its own groups and its own region parameters. Trade
between them is the trade between any two regions: it needs a carrier (TRN) and a
margin (MER), and costs at least the minimum travel time of the shortest mode.

3.4.2. This costs little. The work of clearing grows with regions times the
commodities each trades, and the work of agents with their number, not with the number
of households (`spec-scale.md` 6). A hamlet of a few households is a handful of
agents and a handful of small markets. What it costs is in section 13: a small
market is a thin one (REG-A2), and a hamlet's pool is its own, so a failing hamlet
cannot draw on the town's.

3.4.3. A setting that wants one price for the town and its hamlets does not need a
different structure: it gives them one region. The line between "one region" and
"two regions with a short edge" is the setting's; the engine offers both.

### 3.5 Regions with no agents

3.5.1. A region may have no group at construction, and a region that had agents may
lose them all in a run. Both are valid. Such a region is a node of the graph with a
market in which no one trades; its price is the carried-forward price from the
reference price (`spec-clearing.md` 5.4) and its price age grows. Its estate exists.

3.5.2. `spec-scale.md` 8 calls a distant region modelled only by supply and demand
functions, with no agents at all, the outside world, and gives it to
`spec-external-sector.md`. This spec makes it a node and no more: how such a node is
fed supply and demand, and whether the configuration marks it, are EXT's. Until EXT
exists, a region with no group draws the warning `region_without_agents` (9.2),
which says that nothing will happen there. A region that loses its agents at run time
draws no warning from this spec; the role-extinct and collapse warnings are SPN's and
POP's.

3.5.3. A household may be placed in a region with no agents by an entry (SPN) or a
relocation (section 6). The region's market then starts to clear from its carried
price.

### 3.6 Creation of the estate of each region

3.6.1. At construction, after the configuration is validated and canonicalised and
before any household, the world creates for each region, in ascending `RegionID`
order:

1. the **estate account**, a ledger account of kind Estate whose region is that
   region (`spec-ledger.md` 3.1, 3.5);
2. the **estate pseudo-agent** bound to that account (`spec-estates.md` 2.7.1).

The pseudo-agent of the region with `RegionID` `r` has the agent ID `r`: the
pseudo-agents take the lowest agent IDs, one per region in region order
(`spec-estates.md` 2.7.6, FR-AGT-001). The estate accounts precede every agent
account in account-ID order, in region order among themselves. How they are ordered
against the ledger's boundary accounts is WLD's and the ledger's (N-009); this spec
needs only the order among estates and that they precede all agent accounts.

3.6.2. The estate of a region is created whether or not the region has groups, and
whether or not the module of `spec-estates.md` is on: the account exists so that a
removal always has a place to put what it leaves (`spec-agents.md` 3.6); the pool
policy (release, relief, starting money) is the module's and is off when the module
is off.

3.6.3. An estate account and its pseudo-agent never change region, are never
removed, and are not counted as agents or households (`spec-estates.md` 2.7.4). The
relocation of section 6 refuses an estate account.

3.6.4. A region has exactly one estate. A second one is not creatable.

## 4. The region graph

### 4.1 Edges

4.1.1. An **edge** is a directed route from one region to another:

```go
// Edge is a directed route. It says that a carrier can go directly from From to
// To, over Distance. It says nothing about who, how fast or how much: those
// are the transport spec's.
type Edge struct {
	From, To string // region IDs
	Distance int64  // in the pack's distance unit, 1 to MaxDistance
}
```

`Config.Edges` is a list of edges. In a file an edge is the table
`[region.<from>.edge.<to>]` with the key `distance`, so a document adds an edge
without touching the region, and documents merge by key (`spec-config.md` 6.4).

4.1.2. **Distance.** A distance is an integer length in a unit the pack chooses
(a league, a kilometre, a velocity change, a jump): the core has no notion of the
unit and never converts it. `World.DistanceLabel` (p) is a presentation-only name
for the unit, empty by default, not hashed. The unit should be chosen so that the
shortest edge of the pack is at least a few tens of units: the differences between
edges are then representable to a few per cent (an edge of 8 against one of 9
differs by 12%, one of 800 against 801 by 0.1%). That is advice for a pack author,
not a rule.

4.1.3. **What is valid.**

- both ends are defined regions (`ErrUnknownRegion`);
- `From` differs from `To` (`ErrSelfEdge`): a region is already at distance 0 from
  itself;
- at most one edge for a pair `From`, `To` (`ErrDuplicateID`); the opposite
  direction is a different edge;
- `Distance` is at least 1 and at most `MaxDistance` (`ErrOutOfBounds`). A distance
  of 0 would put two places at the same point, which is one region, and would make a
  cycle of length 0, so that "the shortest path" and its tie-break would no longer
  be well defined.

4.1.4. **Asymmetry and one-way edges** are allowed. `a > b` and `b > a` may have
different distances (a river, a gravity well), and either may be absent (a
one-way pass). A missing reverse edge is not a fault; a graph that is not strongly
connected draws a warning (4.5).

4.1.5. **An edge longer than a detour** is allowed and warned about
(`edge_not_shortest`): the direct route from `a` to `b` may be a poor road that the
shortest path avoids. `Distance(a, b)` returns the shortest path, `Edge(a, b)`
returns the direct route, and a route of the transport spec (a carriage market per
directed route, `spec-carriage.md` 2.2) is an edge, not a shortest path.

4.1.6. **Canonical order.** Edges are sorted by `(From, To)` as `RegionID`s before a
world is built or a hash is taken. The **`EdgeID`** of an edge is its index in that
order, from 0. It does not change when an edge status changes (7).

```go
// EdgeID is the index of an edge in ascending order of (From, To).
type EdgeID int
```

### 4.2 Bounds and arithmetic

All bounds are constants in `bounds.go` (architecture.md 9.6). They are provisional
until the slice has run.

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxDistance` | 10^9 | one edge. A shortest path has at most `MaxRegions - 1` = 999 edges, and an effective edge (7.2) is at most `MaxDistance * MaxEdgeFactorBP / 10 000` = 10^10, so a path is at most `999 * 10^10`, below 10^13. That is far below `int64` (about 9.2e18), so path sums cannot overflow |
| `MaxEdges` | `20 * MaxRegions` = 20 000 | bounds the construction work (4.4): an average out-degree of 20. A full graph of 40 regions has 1 560 edges. A realm of 150 regions does not need a complete graph, because `Distance` goes through intermediate regions |
| `MaxEdgeFactorBP` | 100 000 | the largest length factor of an edge status: ten times (7.2) |
| `MaxRegionPrices` | `MaxRegions * MaxCommodities` | one override per region and commodity; derived, as `MaxSeedPrices` is |
| `MaxLabelBytes` | 256 | a label, for the presentation-only fields |

4.2.1. The additions in the graph code (a path sum, the effective length of an edge)
use the checked helpers of `internal/` and report overflow to the caller
(architecture.md 3.5), although the bounds make overflow impossible. A test runs the
largest case (999 edges of the largest effective length) to show it.

4.2.2. A later spec that multiplies a distance (a cost per distance unit, a rate, a
chargeable size) uses the checked 128-bit helper: `10^13` times `MaxQuantity` does not
fit `int64`. This spec's bound is chosen to leave the distance itself safe, not the
products of other specs.

### 4.3 The queries

4.3.1. **Definitions.** Let there be `n` regions with `RegionID`s `0` to `n-1`, and let
`w(a, b) >= 1` be the effective length of the open edge from `a` to `b` (its
distance, changed by its status, 7.2). A closed edge is not in the graph for the
queries. Then:

- `dist(a, a) = 0`. For `a != b`, `dist(a, b)` is the least total length of a walk
  from `a` to `b` along open edges, and is undefined when there is no walk. Because
  every length is at least 1, a shortest walk repeats no region.
- `NextHop(a, b)`, for `a != b` and `b` reachable from `a`, is the smallest
  `RegionID` `v` among the open out-neighbours of `a` with `w(a, v) + dist(v, b) =
  dist(a, b)`.
- `Path(a, b)` is `a`, then `NextHop(a, b)`, then the next hop of that region
  towards `b`, and so on to `b`. It is the lexicographically smallest of all shortest
  paths from `a` to `b`, compared as sequences of `RegionID`s. (Every shortest path
  starts with an edge that satisfies the equation above, and the paths that start
  with the smallest such `v` are the smallest class; every suffix of a shortest path
  is a shortest path, so the rule repeats at each step.)

The tie-break is therefore a total order that depends on region IDs and on nothing
else: not on the order in which the edges were listed, on a map, on the algorithm or
on the platform (architecture.md 7.5, 7.6). The model checks it against enumeration
of all paths on 3 000 random directed graphs with ties and unreachable pairs
(section 14).

4.3.2. **The `Graph`.** The queries are methods of an immutable value, built from a
configuration and held by the world:

```go
// Graph is the region graph. It is immutable: no method changes it, and a
// change of edge status gives the world a new Graph (7). A Graph a caller holds
// describes the world at the time it was taken and may be read from any goroutine.
type Graph struct { /* unexported */ }

func NewGraph(cfg Config) (*Graph, error) // runs the graph rules of 9, then builds
func (w *World) Graph() *Graph

func (g *Graph) Len() int                               // regions
func (g *Graph) Name(r RegionID) string                 // "" if r is not a region
func (g *Graph) Lookup(id string) (RegionID, bool)
func (g *Graph) Edges() int                             // configured edges, open or closed

func (g *Graph) Edge(from, to RegionID) (EdgeView, bool) // the direct route, closed or not
func (g *Graph) EdgeAt(e EdgeID) (EdgeView, bool)
func (g *Graph) Out(from RegionID) []Hop                // open edges, ascending To
func (g *Graph) In(to RegionID) []Hop                   // open edges, ascending From

func (g *Graph) Distance(from, to RegionID) (int64, bool) // shortest; (0, true) for from == to
func (g *Graph) NextHop(from, to RegionID) (RegionID, bool)
func (g *Graph) Path(from, to RegionID) ([]RegionID, bool) // from ... to; [from] when equal
func (g *Graph) Within(from RegionID, limit int64) []Hop   // others, ascending (Distance, Region)

func (g *Graph) Components() int
func (g *Graph) Component(r RegionID) (int, bool)

// Hop is a region and a length: the effective length of an edge in Out and In,
// the shortest distance in Within.
type Hop struct {
	Region   RegionID
	Distance int64
}

// EdgeView is the state of one edge.
type EdgeView struct {
	ID       EdgeID
	From, To RegionID
	Base     int64 // the configured distance
	Distance int64 // the effective length (7.2); equals Base when the status is default
	Closed   bool
	FactorBP int64 // 10 000 when the status is default
}
```

4.3.3. **Behaviour.** A `RegionID` outside `[0, n)` in any query gives the zero
result and `false`, never a panic: it can come from a caller's data (architecture.md
8.1). `Distance(a, a)` is `(0, true)`. `Within(a, limit)` lists every region other than
`a` whose shortest distance from `a` is at most `limit`, in ascending distance and,
for equal distances, ascending `RegionID`; a negative `limit` gives an empty list.
`Out`, `In`, `Within` and `Path` return a fresh slice that the caller may keep or
change; the graph never hands out its own storage (architecture.md 12.4), so the
queries that return a list allocate, and `Len`, `Name`, `Lookup`, `Edges`, `Edge`,
`EdgeAt`, `Distance`, `NextHop`, `Component` and `Components` do not.

4.3.4. **A `*Graph` crosses the API.** Architecture.md 12.4 asks that a value returned
across the API boundary be a deep copy of plain data with no pointer into live state,
so that a consumer can use it on another goroutine without a race. A `Graph` is
neither plain data nor a copy, but it serves that purpose: it never changes after it
is built, it holds no pointer into the world, and it has no mutating method. The
reading is flagged for the rules owner (open question 3).

4.3.5. **Components.** A **component** is a strongly connected component of the open
graph: a set of regions each reachable from each other. They are numbered from 0 in
ascending order of the smallest `RegionID` in each, so the numbering depends on the
IDs and on nothing else. `Component(r)` gives the number of `r`'s component. A graph
of one region has one component, a graph with no edges has `n`.

### 4.4 Representation and cost

4.4.1. The graph keeps, as derived data: a table of the shortest distance for every
ordered pair (`n * n` values of `int64`, -1 for unreachable), a table of the next hop
for every ordered pair (`n * n` values of `int32`), the component of every region,
and the open out and in lists in compressed form. All are derived from the
configuration and the edge statuses; none is saved in a checkpoint (11).

4.4.2. **Construction** runs one single-source search from each region (Dijkstra with
a binary heap ordered by `(distance, RegionID)`), `O(n (m + n) log n)` for `m` edges;
derives the next-hop table by scanning, for each source, its out-edges in ascending
order for each target, `O(n m)`; and derives the components from the distance table,
`O(n^2)`. It is done once at construction and again after a change of edge status.
The algorithm is not part of the spec: any exact algorithm gives the same tables,
because 4.3.1 defines them by their result.

4.4.3. **Operation counts** (not timings: nothing has been measured, N-015):

| Realm | `n` | `m` | Construction, order of | Tables |
|-------|----:|----:|------------------------|--------|
| The first consumer | 5 | at most 20 | under 10^3 steps | 300 bytes |
| A larger realm | 100 | 400 | 100 * 500 * 7, about 3.5 * 10^5 steps | 120 KB |
| The bound | 1 000 | 20 000 | 1 000 * 21 000 * 10, about 2 * 10^8 steps | 12 MB |

Even the largest case happens at construction or when a shock changes an edge, never
inside the tick loop. A query inside the loop is a table read (`O(1)`) or, for a list,
a copy of at most `n` entries.

4.4.4. A batch of edge statuses (7) rebuilds the tables once, not once per edge.

### 4.5 Connectivity, islands and warnings

4.5.1. **No connectivity is required.** A graph with no edges, with several
components or with one-way edges is valid. A region with no edge in or out is a
closed economy: nothing can be shipped to it or from it. This is the scenario of the
`famine` variant (`spec-reference-chain.md` 4.4.5) and of S14.

4.5.2. **Warnings** (9.2) tell the author what the shape is, because it is often not
what was meant:

- `region_isolated`: with at least two regions, a region with no edge in or out;
- `graph_not_strongly_connected`: with at least two regions, more than one component;
  the warning carries the count (`components`);
- `edge_not_shortest`: an edge longer than the shortest path between its ends
  through other edges (4.1.5);
- `region_without_agents`: a region with no group (3.5).

They never fail validation (`spec-config.md` 7.2). A world of one region and no edges
draws none of the first three.

4.5.3. **Import-dependent regions.** `spec-viability.md` 9.5 warns about a region
that cannot supply an upkeep good itself (`region_not_self_sufficient`) and assumes,
for the comparison it calls free trade, an edge between every pair of regions
(`spec-viability.md` 9.4). This spec supplies what that check lacks: whether the
import can arrive at all. Whether the viability check should use `Components` is for
VIA (open question 2); this spec only exposes it.

4.5.4. A closure at run time (7) changes the components. It is a shock, not a
configuration fault: no warning comes from this spec, and the event of 10.1 reports
it.

### 4.6 What the later specs read

The read-only view that the specs that depend on this one take. Every item is a
method of 4.3.2; none gives access to the world, none allocates unless it returns a
list, and none may be called back into the world during a tick (architecture.md
12.5).

| Spec | Reads | Must not assume |
|------|-------|-----------------|
| MIG | `Out`, `Within`, `Distance` to weigh a destination; `Component` to skip unreachable ones | symmetric distances; that every region is reachable |
| INF | `Distance`, `Within`; `Path` if a rumour passes region by region. The delay of a rumour is INF's function of the distance | that a distance is a number of ticks |
| TRN | `Edge`, `EdgeAt` for the direct route of a shipment (the effective length and `Closed`); `NextHop`, `Path` for a journey of several routes; `EdgeID` as a key. Travel time, cost and capacity are TRN's, derived from the effective length and a mode | that a shortest path is a direct edge; that a closed edge can be used |
| CAR, MER | `Out` for the set of directed routes; `EdgeID` as the key of a route (4.7) | that a route has a reverse |
| RES | `Len`, `Name`: the set of regions, so that an endowment is keyed by `RegionID` | any distance |
| VIA | `Len`, `Name`; `Out` as the candidate lanes of the solver's hook (`spec-viability.md` 9.4); `Component` (4.5.3) | that all pairs are linked |
| STY, IND | `Distance` for the price-convergence bands; region order for output | |
| EXT | `Len`; which regions have no agents (3.5) | |
| INT | `SetEdgeStatuses` (7) for the closed-road shock | |
| POL, SET (later) | `Len`, `RegionID`: a partition or a size class is a table keyed by region | that REG reads either |

### 4.7 Routes as keys: the hook for the carriage market

4.7.1. `spec-carriage.md` 4.2 leaves open "which commodity, and where" a carriage
market lives, and how an order for a route is keyed. This spec gives the key: a route
is an edge, and its identity is its `EdgeID` (or the pair `From`, `To`). The total
order of routes is ascending `EdgeID`, which is ascending `(From, To)`. If TRN places
the market of a route in the origin region, its sort key in the clearing order is
`(From, route, commodity)`, which is a total order; if it places it elsewhere, it
states the key. Which of the two is for TRN and CAR (N-076). This spec adds no market.

## 5. Overrides by region

### 5.1 One mechanism

5.1.1. Several values differ by region: the reference price of a commodity, the
reference basket, the extra upkeep, the yield of a recipe, the taste for a want. Each
was assigned to the spec of its subject, which is right: the owner of the subject
defines the meaning, the default and the bound. What was missing was one rule for how
a region's value meets the default, so that a pack author learns it once. This is
it.

5.1.2. **A region override is a value keyed by a region and a subject, held in a
table of the subject's owner, combined with the default in one of three ways:**

| Mode | Effective value | Used by |
|------|-----------------|---------|
| **replace** | the row's value if the row exists, else the default | the reference price (5.2); a region's scalar parameters, which are fields of the region whose default is `DefaultRegion()` (`ReferenceBasket`, `CapitalReturn`, `EstateMarkdown`, `PoolRelease`, `ReliefBelow`) |
| **add** | the default list plus the row's list, quantities of one commodity added | `ExtraUpkeep` (`spec-agents.md` 5.2) |
| **scale** | the default times the row's factor over 10 000, with the rounding rule of the owner | the yield factor (`spec-production.md` 6.3, default 10 000, or 0 for a `Sited` recipe); the taste scale of a region (`spec-demand.md` 3.2, default 10 000) |

5.1.3. The tables to which the mechanism applies today:

| Subject | Table in `Config` | Key | Owner of meaning and bound | Mode |
|---------|-------------------|-----|----------------------------|------|
| reference price | `RegionPrices` | region, commodity | REG for the row, `spec-goods.md` 6.3 for the default | replace |
| reference basket | `Regions[].ReferenceBasket` | region | `spec-currency.md` 5 | replace |
| extra upkeep | `Regions[].ExtraUpkeep` | region, commodity | `spec-agents.md` 5.2 | add |
| capital return | `Regions[].CapitalReturn` | region | `spec-skill-barriers.md` 2.5 | replace |
| estate parameters | `Regions[]` | region | `spec-estates.md` 3 | replace |
| yield | `Yields` | region, recipe | `spec-production.md` 6.1 | scale |
| taste | `Modules.Demand.RegionTastes` | region, want | `spec-demand.md` 3.2 | scale |

`Groups` (`spec-scale.md` 3) and `WarmStart` (`spec-viability.md` 8) are also keyed by
region but are populations and starting values, not overrides of a default.

### 5.2 The reference-price override

5.2.1. `Config.RegionPrices` is a list of

```go
// RegionPrice replaces, in one region, the reference price of one commodity.
type RegionPrice struct {
	Region    string
	Commodity string
	Price     Price // 1 to MaxPrice
}
```

In a file it is a table of the region, keyed by commodity:
`[region.<id>.reference_price]` with `<commodity> = <price>`. The mode is replace. A
region with no row for a commodity uses the commodity's `ReferencePrice`, which is
required (`spec-goods.md` 6.3).

5.2.2. The **effective reference price** of a commodity in a region is the row's price
if there is one, else the commodity's. Everything that `spec-goods.md` 6.3 lists as
using the reference price uses the effective one: the seed price and the centre of
the initial beliefs; the reference band; the basket `B_r`; the estate's listing price;
the fallback for a tool listing. A warm start (`spec-viability.md` 8.1) replaces the
seed price only; the reference band and the other uses stay on the effective
reference price.

5.2.3. **Relations that the validation already checks, now with the effective
price:** the cheapest tradable commodity's effective reference price in every region is
at least the resolution floor (`spec-config.md` 7.4.1, FR-BEL-029); a region's
`ReferenceBasket` matches the cost of its baseline upkeep at its effective reference
prices within `BasketTolerance` (`ErrBasketMismatch`, `spec-currency.md` 5.3). A
region whose override makes its staple dearer must therefore state a dearer
`ReferenceBasket` (example A, section 12).

5.2.4. No relation between an override and the commodity's own price is required: a
region may be dearer or cheaper by any factor within `[1, MaxPrice]`. A row equal to
the commodity's price draws the warning `override_equals_default`. The world resolves
the price once, into a dense array indexed by `RegionID` and commodity (5.3.5), and
exposes it as `World.ReferencePrice(r RegionID, c CommodityID) Price`.

### 5.3 Rules common to every region-keyed table

5.3.1. **Both keys exist.** A row naming an undefined region is rejected with
`ErrUnknownRegion`, one naming an undefined subject with the sentinel of that kind
(`ErrUnknownCommodity`, `ErrUnknownRecipe`, `ErrUnknownWant`).

5.3.2. **One row per key.** A second row for the same key is `ErrDuplicateID`, in a
`Config` built in Go (a file cannot have it).

5.3.3. **Absence is the default**, with no warning. A region that says nothing is a
region that follows the world.

5.3.4. **Canonical order.** Rows are sorted by region (as `RegionID`), then by the
subject's ID, before a hash is taken or a table is walked, and the hash covers every
row (`spec-config.md` 8).

5.3.5. **Resolved once.** At construction the world resolves each table into a dense
array indexed by `RegionID` and the subject's index, so a lookup in the tick loop is a
read and allocates nothing. The array is derived and is not saved.

5.3.6. **Static.** An override is configuration. The two things that vary a regional
value in a run are separate slots with their own state: the yield shocks of
`spec-production.md` 6.8 and the edge status of section 7. Neither edits an override.

### 5.4 What is not an override

5.4.1. There is no hierarchy: a region's value is compared with the world's or the
commodity's, never with a province's or another region's. POL may group regions; it
does so with a table of its own that names regions, and no value is inherited through
it.

5.4.2. A World tunable (a belief step, a cover period) is world-wide. A setting that
needs it by region makes it a region field in its owner's spec, with the rules above.
This spec does not add one, and an owner that does adds a row to the table of 5.1.3.

5.4.3. A value that varies with the tick (a season) is not a regional override: it is
the owner's profile (a yield by season), keyed by region among its keys.

## 6. Changing an account's region

`spec-ledger.md` 3.5 says that an account's region "can change only by an explicit
operation, never as a side effect", and leaves its interaction with goods in transit
open (`spec-ledger.md` 15, item 8). This section is that operation. It decides what
the move does to the ledger and to the agent's state and where the rest is someone
else's. It does not decide who moves, when or why: that is MIG's, and INT's for an
external agent.

### 6.1 The operation

6.1.1. The world has one internal operation,

```go
// relocate moves the agent's ledger account to region `to`. It is not a
// transfer: nothing passes between accounts. MIG and INT expose it, each under
// its own rules.
func (w *World) relocate(a AgentID, to RegionID, cause Cause) error
```

backed by `Relocate(account, region)` in `internal/ledger`, which takes the region as
a plain index and does not know what a region is (architecture.md 2.9). The root
checks that the index names a region. `cause` is a constant of `internal/logkeys`
that the caller supplies (`migration` is MIG's; INT supplies its own); it is carried
to the event and the log and affects nothing else.

6.1.2. **When.** A relocation is legal only when the account has no reservation, and
reservations exist only inside a tick, between admission and settlement
(`spec-ledger.md` 6.5). So it is legal at a tick boundary, which is also when a
consumer's call to the world may be made, and at the points of the tick that WLD fixes
after settlement and before the next admission. It is never legal between the
admission of an order and its settlement. Where in the tick MIG's relocations happen
is WLD's and MIG's to fix; the recommendation is the entry-and-exit phase
(`spec-agents.md` 4), so that an agent places its next orders in its new region.

6.1.3. **Refusals.** The checks run in this order, before anything changes. A refused
relocation changes nothing (`spec-ledger.md` 2.6) and returns the error shown:

| # | Reason | The relocation is refused when | Error |
|---|--------|--------------------------------|-------|
| 1 | `unknown_region` | `to` is not in `[0, n)` (so `NoRegion` is refused in version 1, 6.5) | `ErrUnknownRegion` |
| 2 | `unknown_agent` | the ID names no agent | `ErrUnknownAgent` |
| 3 | `closed_account` | the agent's account is closed | `ErrClosedAccount` (`spec-ledger.md` 4.2) |
| 4 | `not_relocatable` | the account is not of kind Agent: an estate, an escrow, a transit or a boundary account | `ErrNotRelocatable` |
| 5 | `same_region` | the account is already in `to` | `ErrSameRegion` |
| 6 | `open_reservation` | the account has a reservation of money or goods | `ErrOpenReservation` |
| 7 | `region_bound` | an obligation that a later spec registered is tied to the account's region (6.5) | `ErrRegionBound`, which names the obligation |

The reason is a constant of `internal/logkeys`, used in the log and in the count a
caller keeps. The caller decides whether to log the error it gets (architecture.md
8.3); the world does not log an error it returns.

6.1.4. **The whole agent moves.** The account is the agent's, and an agent has a
weight (`spec-scale.md` 4). A relocation moves the agent with all the households it
stands for. Moving part of a cohort is a split first (the ledger reason `split`,
`spec-scale.md` 4.8) and then a relocation of the new agent; version 1 does not split
(`spec-scale.md` 8), and the first consumer has weight 1 throughout (N-030).

6.1.5. **An external agent** may be relocated: the consumer asked for it (INT). The
checks of 6.1.3 are the only ones. The default MIG does not relocate an external agent
on its own (that is MIG's rule, N-022).

### 6.2 What the operation does

On success, in this order, atomically:

1. the account's region becomes `to` (`spec-ledger.md` 3.5);
2. the relocation totals of the current tick are increased by what the account holds
   (6.4);
3. the derived list of the agents of each region (3.1.1) reflects the change: the
   agent leaves the list of the old region and enters the list of the new one, each
   kept in agent-ID order;
4. after the change is complete, the event `AgentRelocated` is emitted (10.1).

Flows after the move are attributed to the new region (`spec-ledger.md` 3.5); flows
before it stay with the old one. Nothing else changes in the ledger.

### 6.3 What does not move, and who decides

The money and the goods stay in the same account: lots keep their born tick,
durability and the account's expiry hint, because the account does not change. There
is no transfer, no reason, no boundary account and no flow, so the identities of
`spec-ledger.md` 9.1 are unaffected. Everything else the agent has is untouched by this
operation. Where the new region would make it stale, the caller decides:

| State | What this operation does | Who decides, and the default that spec states or proposes |
|-------|--------------------------|----------------------------------------------------------|
| Role, weight, strategy | nothing | unchanged by the move |
| Job in progress | nothing: its outputs were fixed at its start (`spec-production.md` 6.4, FR-PRO-021) and it completes where the agent then is | MIG. A `Sited` job would complete away from its site, so MIG should abandon the job first with a reason of `spec-production.md` 5.4; this spec proposes `region_changed` (open question 4) |
| Price beliefs | nothing | MIG; the default is to reseed on arrival (`spec-beliefs.md` 8.3, N-038) |
| Welfare record | nothing: it travels with the agent (`spec-welfare-and-wages.md` 2.1.2) | WEL |
| `Hunger`, `Privation` | nothing | SPN's leaky count is the agent's; MIG may reset it |
| Traineeship | nothing | SKL: a trainer with trainees or a trainee is registered as a `region_bound` refusal (6.5) |
| Directive in the mailbox | nothing | INT |
| Effective upkeep, effective reference prices, cost anchor | nothing: they are derived from the region each tick (`spec-agents.md` 5.2, `spec-production.md` 9.3) and change at the next computation | |

### 6.4 Accounting: the relocation totals

6.4.1. The ledger keeps, for the current tick, the **relocation totals**: for each
ordered pair of regions `(from, to)` with a relocation in the tick, the money moved
with the accounts and, for each commodity, the units. They have the layout, the
lifetime and the order of the flow totals (`spec-ledger.md` 7.4): cleared at the start
of each tick, kept as a slice in a fixed order (ascending `from`, then `to`, then money
before the commodities in commodity order), bounded by the regions and the commodities
and never by the number of agents, and read-only. They are not saved in a checkpoint.

6.4.2. They are the reconciling item of a regional stock report. The money held in
a region changes by the flows attributed to it and by the relocations in and out, and
the totals say how much came with accounts that moved. They are the only way a report
by region can tell a household that arrived with money from a faucet. The global
identities need no such item, since a relocation moves nothing between accounts.

6.4.3. A relocation is not a boundary flow. It appears in no `BoundaryFlow` and no
reason of the registry (`spec-ledger.md` 7.2) is added for it.

### 6.5 Goods in transit and travellers: the hook for TRN

6.5.1. **Cargo is not the agent's account.** Goods being carried are in a Transit
account that belongs to a shipment (`spec-ledger.md` 3.1), not in the account of the
agent that ships or receives them. A relocation of an agent therefore neither moves
nor re-homes cargo. A shipment's origin and destination regions are fixed by TRN when
it is created, from the edges of this spec.

6.5.2. **A shipment tied to a region.** Delivery credits a recipient's account, and the
recipient's account is in some region at that time. If the recipient has moved away
from the destination, goods would appear in another region without having travelled
there. This spec forbids that by a hook, not by a rule of its own: a spec that creates
an obligation tied to the region of an account **registers a refusal** of reason
`region_bound` with a name, and the relocation of 6.1.3 refuses an account for which
any registered obligation is open. The refusals foreseen, none of them in version 1,
and the owner of each:

| Name | Obligation | Owner |
|------|------------|-------|
| `open_shipment` | the agent owns, or is the recipient of, a shipment in flight | TRN |
| `open_contract` | the agent is a party to an open carriage contract on a route | CAR, MER |
| `in_traineeship` | the agent is a trainer with trainees, or a trainee | SKL |

The list is closed at construction by the modules that are on (a module that is off
registers nothing), so it is part of the world's behaviour and not a callback into
consumer code. In version 1 it is empty.

6.5.3. **An agent that is itself travelling.** The relocation of 6.1 is instantaneous:
the agent is in the old region until the call and in the new one after it. A journey
that takes time is TRN's. This spec reserves `NoRegion` (3.3.1) as the region of an
account that is between regions, and requires of TRN, if it uses it: (a) it adds two
operations, `depart` (to `NoRegion`) and `arrive` (from `NoRegion` to a region), with
the preconditions of 6.1.3 and the event of 10.1; (b) it records the departure region,
because a death in transit leaves the money to the region departed from
(`spec-estates.md` 2.2) and flows of the account are attributed to a region
(`spec-ledger.md` 3.5); (c) an account with no region is in no market, no estate and no
agent list of a region and places no order. In version 1 `relocate` refuses
`NoRegion`, and an agent is always in a region.

6.5.4. **What is lost in transit** (cargo or a traveller) is a `loss` sink and is RSK's
(`spec-ledger.md` 7.2). This spec reserves the place and specifies nothing.

### 6.6 Who calls it

MIG calls it for a household that moves; INT for an external agent the consumer moves
(the player on a journey, in version 1 without a duration); TRN for the arrival of a
traveller, if it models one. The default spawner does not call it: an entry creates a
household in a region, it does not move one (`spec-spawner.md` 4.3). The public
face of the operation, its name and its phase in the tick belong to those specs.

## 7. Edge status

### 7.1 What it is for

7.1.1. The first consumer needs a closed road (`vision.md` 2), and the stylised facts
need a shock to the links as well as to the harvest (`spec-stylised-facts.md`, STY).
The graph of section 4 is configuration and does not change. This section adds the one
dynamic part: for each edge, a **status** that a shock sets and clears between ticks.
It follows the shape of the yield shocks of `spec-production.md` 6.8: world state, not
configuration; set and cleared between ticks; default off.

7.1.2. The status of an edge is two values: whether it is **closed**, and a **factor**
in basis points that scales its configured distance. The default is open at 10 000,
which is the configured distance. No edge is added or removed and no base distance
changes. A pack with no shocks never has a non-default status, and then the graph is
built once and never again.

### 7.2 The effective length

7.2.1. The **effective length** of an open edge is

```
w = max(1, ceil(Base * FactorBP / 10 000))
```

with the product computed by the checked 128-bit helper, `Base` the configured distance
and `FactorBP` in `[1, MaxEdgeFactorBP]`. The rounding is up, so a lengthening never
rounds to a shorter edge and an effective length is never 0 (4.1.3). A factor below
10 000 shortens an edge (a new road, a favourable season); one above lengthens it (a
damaged road). A closed edge has no effective length and is not in the graph for the
queries (4.3.1), but `Edge` still returns it, with `Closed` set.

7.2.2. Examples, from the model: `Base` 30 at 15 000 gives 45; at 10 000 gives 30;
`Base` 7 at 10 001 gives 8; `Base` 1 at 1 gives 1.

### 7.3 The call

```go
// EdgeStatus is the status a shock gives an edge. The zero Closed and FactorBP
// 10 000 are the default.
type EdgeStatus struct {
	From, To RegionID
	Closed   bool
	FactorBP int64 // 1 to MaxEdgeFactorBP
}

// SetEdgeStatuses applies the changes to the world's graph, all or none. It is
// called between ticks and never inside a Step.
func (w *World) SetEdgeStatuses(cause Cause, changes []EdgeStatus) error
```

7.3.1. Every change names a configured edge (`ErrUnknownEdge`), has a factor in
range (`ErrOutOfBounds`), and no edge appears twice in one call (`ErrDuplicateID`). If
any change fails, none is applied. A change equal to the edge's current status is
accepted and is not an event.

7.3.2. On success the world builds a new `Graph` once for the whole batch (4.4.4) and
replaces the old one. A `Graph` that a caller holds keeps describing the world as it
was (4.3.2). Then `EdgeStatusChanged` is emitted for each edge whose status changed,
in ascending `EdgeID`, after the new graph is in place (10.1).

7.3.3. The public face is INT's (a closed-road shock), and CAP, RSK and POL may call it
with their own `cause` later. The cause is a constant of `internal/logkeys`, carried
to the event and the log.

### 7.4 What a status does not do

It does not move, delay, strand or destroy anything: what a shipment on a route that
closes does is TRN's, and what a household in a region that is cut off does is the
ordinary behaviour of a closed region (4.5.1). It sets no price and reads none. In
version 1, before TRN, INF and MIG read the graph, a status changes the answers of the
queries and emits its event, and nothing else.

## 8. Parameters

All are fields of `Config` with a bound, a doc line and an owner (architecture.md
9.5 to 9.7). The spec adds no tunable with a default: a distance and a price have no
sensible default, and the default of the section as a whole is "no edges, no
overrides", which is a closed economy. The fields owned by other specs that a region
holds are in 3.2 and are not restated. The bounds are in `bounds.go` (4.2).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Edge | `From` | the region the route leaves | required | a defined region |
| Edge | `To` | the region the route enters | required | a defined region other than `From` |
| Edge | `Distance` | the length of the route in the pack's unit (4.1.2) | required | 1 to `MaxDistance` |
| RegionPrice | `Region` | the region whose reference price differs | required | a defined region |
| RegionPrice | `Commodity` | the commodity whose reference price differs | required | a defined commodity |
| RegionPrice | `Price` | the reference price in that region, in minor units | required | 1 to `MaxPrice` |
| Config | `Edges` | the edges | none | at most `MaxEdges` |
| Config | `RegionPrices` | the reference-price overrides | none | at most `MaxRegionPrices` |

Presentation-only, so exempt from the registry test and not hashed
(`spec-config.md` 8.3): `Region.Label` and `World.DistanceLabel`, both empty by
default and at most `MaxLabelBytes` bytes.

Not fields: the edge status of section 7 (world state, saved in a checkpoint, set by
a call), the `cause` of a call, and the `RegionID` and `EdgeID` (derived).

The placeholders, so that none is mistaken for a result: every bound of 4.2
(`MaxDistance`, `MaxEdges`, `MaxEdgeFactorBP`, `MaxLabelBytes`), the operation counts
of 4.4.3, and the numbers of the worked examples of section 12.

## 9. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). Errors name the
path, for example `edges[3].distance` or `region_prices[1].price`. The framework of
`spec-config.md` 7 runs them, and `NewGraph` runs the graph rules alone.

### 9.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrNoRegions` (new) | a configuration with no region |
| `ErrInvalidID`, `ErrDuplicateID` | a region ID that does not match `spec-config.md` 3.3, or twice |
| `ErrTooMany` | more than `MaxRegions` regions, `MaxEdges` edges or `MaxRegionPrices` overrides |
| `ErrUnknownRegion` | an edge, an override or any region-keyed row (5.3.1) naming an undefined region |
| `ErrUnknownCommodity` | an override naming an undefined commodity |
| `ErrSelfEdge` (new) | an edge whose `From` equals its `To` |
| `ErrDuplicateID` | two edges with the same `From` and `To`; two overrides for one region and commodity |
| `ErrOutOfBounds` | a `Distance` below 1 or above `MaxDistance`; an override `Price` outside `[1, MaxPrice]`; a label above `MaxLabelBytes` |

Rules of other specs that now read the effective reference price are unchanged in
their text and take the effective price: the resolution floor and `ErrBasketMismatch`
(5.2.3).

The errors of the runtime calls: `ErrUnknownRegion`, `ErrUnknownAgent` (new),
`ErrClosedAccount`, `ErrNotRelocatable` (new), `ErrSameRegion` (new),
`ErrOpenReservation` (new) and `ErrRegionBound` (new) for `relocate` (6.1.3);
`ErrUnknownEdge` (new), `ErrOutOfBounds` and `ErrDuplicateID` for `SetEdgeStatuses`
(7.3.1). They are root sentinels matched with `errors.Is`; the ledger's own
(`ErrClosedAccount`) maps to the same value (note N-075 records this mapping for the
ledger's sentinels).

### 9.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at
construction, at `Warn`, with the attributes `rule` and `path`; the volume is bounded
by the configuration.

| Warning | Condition |
|---------|-----------|
| `region_isolated` | at least two regions, and a region with no edge in or out (4.5.2) |
| `graph_not_strongly_connected` | at least two regions and more than one component; carries `components` |
| `edge_not_shortest` | an edge longer than the shortest path between its ends through other edges (4.1.5); one warning per such edge |
| `region_without_agents` | a region with no group (3.5.2) |
| `override_equals_default` | a reference-price override equal to the commodity's price (5.2.4) |

## 10. Observability

10.1. **Events.** Both are plain data, with no map, no pointer and no wall-clock time
(observability-and-logging.md 4.1), emitted only after the transition they describe is
complete, and constructed only when an observer was supplied (4.5 there).

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `AgentRelocated` (new) | an agent's account changes region (6.2), once per relocation, in the order the relocations were applied | `Agent`, `Role`, `From`, `To` (`RegionID`), `Weight`, `Money` (what the account held), `Units` (the total units of goods it held), `Cause` |
| `EdgeStatusChanged` (new) | an edge's status changed (7.3.2), once per changed edge, in ascending `EdgeID`, after the new graph is in place | `From`, `To`, `Closed`, `FactorBP` (the new status), `Cause` |

`AgentRelocated` is per agent, like `AgentSpawned`: it is a transition of one agent.
The fields that name a region carry the `RegionID`; whether events carry the region's
string ID instead, as other events may, is for OBS to settle for all of them at once
(open question 5). A consumer that tracks households by region needs `AgentRelocated`
as well as `AgentSpawned`, `AgentBankrupted`, `WeightChanged` and `RoleChanged`
(`rules/observability-and-logging.md` 4.3): a household leaves `From` and joins `To`.
The estate pseudo-agents are not agents and cause no event (`spec-estates.md` 2.7.4).

10.2. **Logs.** Constant lowercase messages, typed attributes, `Enabled` guards.

- `Info`, once, in the existing "world created" line: two attributes join the summary,
  `edges` and `components` (counts, with the existing `regions`). No other `Info`.
- `Warn`, at construction, one per warning of 9.2, with `rule` and `path`.
- `Debug`, per relocation: "agent relocated" with `agent`, `from_region`, `to_region`,
  `cause` and `households`; "relocation refused" with `agent`, `to_region` and
  `reason`. Per status change: "edge status changed" with `from_region`, `to_region`,
  `closed` and `cause`. All guarded by `Enabled` (3.1.3 there).
- No `Error`. No `Info` inside the tick: a relocation is a per-agent event and so is
  `Debug` and an event (3.2.2 there).

10.3. **New log keys** for observability-and-logging.md 3.3.2: `edges` and
`components` (counts, in the construction summary), `from_region` and `to_region`
(region IDs; `region` stays the key for a single region), `distance` (an `int64` in
the pack's unit), `closed` (a bool), and `households`, `cause` and `reason`, which exist;
their values are constants in `internal/logkeys`.

10.4. **Metric.** One instrument, fed by `AgentRelocated`, for the catalogue of
`rules/observability-and-logging.md` 5.3:

| Name | Instrument | Unit | Attributes |
|------|-----------|------|------------|
| `economysim.household.relocations` | Counter | `{household}` | `economysim.region.from`, `economysim.region.to` |

It counts households (`Weight`), not agents, like the other household metrics. Its
cardinality is bounded by the configuration: ordered pairs of regions, so 20 series
for the five-region reference world and at most `MaxRegions` squared in the worst case
(5.4.2 there). An edge status is not a metric; the snapshot reports it. The names and
attributes are proposals for OBS.

10.5. **Snapshot.** Per region: the households and agents (`spec-scale.md` 9.2) and the
pool (`spec-estates.md` 2.4) are already there; this spec adds the edges whose status
is not the default (from, to, closed, factor) and the number of components. Bounded by
the edges. The names of the fields are OBS's.

10.6. **No observer, no logger.** The graph is built once and the queries allocate
nothing except the lists (4.3.3); no event is constructed without an observer. Observation
never changes a result: no log, event or metric draws a random number or reads the
wall-clock (observability-and-logging.md 1.4).

## 11. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, plain data in a
fixed order with no maps:

- the list of edges whose status is not the default, in ascending `EdgeID`: `From`,
  `To`, `Closed`, `FactorBP`. Empty in a world with no shocks.

Already saved by others: the region of every account is in the ledger state
(`spec-ledger.md` 11); the estates are `spec-estates.md`'s (2.7.5).

**Not saved** (derived): the `RegionID` and `EdgeID`, the distance, next-hop and
component tables, the adjacency lists, the resolved override arrays (5.3.5), the
relocation totals (6.4.1, like the flow totals) and the lists of the agents of a
region. A restore builds the graph from the configuration and the saved statuses and
gets the same tables, because they are defined by their result (4.3.1); a world
restored between ticks answers every query as an uninterrupted one does (FR-REG-052).
The configuration hash covers `Regions`, `Edges` and `RegionPrices` in canonical
order, every field except the labels, so a change of any edge or override changes the
hash (`spec-config.md` 8).

## 12. Worked examples

Three regions each, from three different settings. The settings are **examples**, chosen
to show that the core has no setting in it; the names and numbers are placeholders. The
tables are the output of `models/reg/paths.py` (section 14). Region order is by ID
string, so `RegionID` 0 is the first name in each table's row order.

### 12.1 Example A: a river valley (a pre-industrial setting; unit: one league)

Regions `lower` (0), `middle` (1), `upper` (2). Upstream is longer than downstream.

```toml
[region.upper.edge.middle]  distance = 40
[region.middle.edge.upper]  distance = 70
[region.middle.edge.lower]  distance = 30
[region.lower.edge.middle]  distance = 55
[region.upper.edge.lower]   distance = 90     # a mountain track
[region.lower.edge.upper]   distance = 120
```

Shortest distance (row from, column to):

| from \ to | `lower` | `middle` | `upper` |
|-----------|--------:|---------:|--------:|
| `lower`   | 0 | 55 | 120 |
| `middle`  | 30 | 0 | 70 |
| `upper`   | 70 | 40 | 0 |

- `Distance(upper, lower)` is 70, by `upper > middle > lower` (40 + 30), not the track
  of 90: the track draws `edge_not_shortest`. `Distance(lower, upper)` is 120, the
  direct edge, since the detour `lower > middle > upper` is 55 + 70 = 125. The
  distances are not symmetric: 70 down, 120 up.
- One component: every region reaches every other.
- **Overrides.** The world's upkeep is 2 `staple` (reference price 300), and `fuel`
  costs 400. `upper` adds `fuel` ×1 to its upkeep (`ExtraUpkeep`, mode add): its
  baseline is 2 × 300 + 400 = 1 000, the default `ReferenceBasket`. `middle` adds
  nothing: its baseline is 600, so it must state `reference_basket = 600`; left at the
  default 1 000 it would differ by 400, more than the 100 allowed (10% of 1 000 at
  `BasketTolerance` 1 000), and be rejected with `ErrBasketMismatch`. `lower` has a
  `RegionPrices` row `staple = 360` (mode replace): its effective `staple` price is 360,
  its baseline 2 × 360 = 720, and it states `reference_basket = 720`.
- **A status.** `SetEdgeStatuses(shock, [middle>lower closed])` gives the new
  distances `lower` 0, 55, 120; `middle` 160, 0, 70; `upper` 90, 40, 0: from `upper`
  the track (90) is now the shortest to `lower`, and `middle` reaches `lower` only by
  `middle > upper > lower`, 70 + 90 = 160. A factor of 15 000 instead of a closure
  makes the edge 45; `upper > middle > lower` is then 40 + 45 = 85, still shorter than
  the track.

### 12.2 Example B: an orbital colony (a space setting; unit: velocity change, m/s)

Regions `belt` (0), `orbital` (1), `surface` (2). There is no edge between `surface` and
`belt`. Going up the gravity well costs more than coming down.

```toml
[region.surface.edge.orbital]  distance = 9400
[region.orbital.edge.surface]  distance = 2000
[region.orbital.edge.belt]     distance = 3000
[region.belt.edge.orbital]     distance = 3000
```

| from \ to | `belt` | `orbital` | `surface` |
|-----------|-------:|----------:|----------:|
| `belt`    | 0 | 3000 | 5000 |
| `orbital` | 3000 | 0 | 2000 |
| `surface` | 12400 | 9400 | 0 |

`Path(surface, belt)` is `surface > orbital > belt` (9 400 + 3 000). If the edge
`orbital > surface` were absent (a colony that can send but not receive), `surface`
would reach the others (9 400 and 12 400) and nothing would reach it: two components,
`{belt, orbital}` as 0 and `{surface}` as 1, and the warning
`graph_not_strongly_connected` with `components` 2. The distance here is an effort, not
a time: how long a manoeuvre takes is TRN's.

### 12.3 Example C: a town and two hamlets (N-026; unit: ten metres)

Regions `hamlet_east` (0), `hamlet_west` (1), `town` (2): a market town and two
hamlets, each its own market, estate and groups, joined by short edges in both
directions (80 is 800 m) and a cross-country track (160).

```toml
[region.town.edge.hamlet_east]        distance = 80
[region.hamlet_east.edge.town]        distance = 80
[region.town.edge.hamlet_west]        distance = 80
[region.hamlet_west.edge.town]        distance = 80
[region.hamlet_east.edge.hamlet_west] distance = 160
[region.hamlet_west.edge.hamlet_east] distance = 160
```

| from \ to | `hamlet_east` | `hamlet_west` | `town` |
|-----------|-------------:|--------------:|-------:|
| `hamlet_east` | 0 | 160 | 80 |
| `hamlet_west` | 160 | 0 | 80 |
| `town` | 80 | 80 | 0 |

- **The tie.** `hamlet_east` to `hamlet_west` is 160 both directly (160) and through
  the town (80 + 80). `NextHop(hamlet_east, hamlet_west)` is the smallest `RegionID`
  among the neighbours on a shortest path: `hamlet_west` is 1 and `town` is 2, so the
  direct track. The track is not longer than the detour, so it draws no warning
  (`edge_not_shortest` is for a strictly longer edge).
- **It depends on the IDs and on nothing else.** Rename `town` to `a_town` and it
  sorts first (0): the same tie now resolves through it, `hamlet_east > a_town >
  hamlet_west`. Listing the edges in any other order changes nothing (the model asserts
  it).
- **Agents and IDs.** Three regions give three estate pseudo-agents with agent IDs 0,
  1 and 2, in the order `hamlet_east`, `hamlet_west`, `town`, before any household
  (3.6). Each hamlet has its own pool: a hamlet that fails cannot draw on the town's
  (REG-A2).

### 12.4 Use by the reference chain (a proposal for REF)

`spec-reference-chain.md` 7.6 says the dynamic parts of X1 and X2 wait for an
`overlay/trade.toml` that "those specs define". The edge part of it is this spec's.
The variants today have no edges: `one` stays so (a single region, the degenerate
case), and the others are sets of regions that do not trade. A proposal, with
distances that are placeholders until TRN fixes what a unit of distance costs and how
fast a mode moves:

| Variant | Edges of the trade overlay | Shows |
|---------|-----------------------------|-------|
| `two` | `north` and `south` in both directions, 100 each | the minimal route; X1 |
| `three` | `south` to each of `north` and `far_south`, both directions, 100 each; no edge between `north` and `far_south` | a shortest path of two routes (200 between `north` and `far_south`), and a middle region; X2 |
| `five` | `centre` to each of `east`, `north`, `south`, `west`, both directions, 100 each; `north` and `east`, and `south` and `west`, both directions, 130 each | one component, paths of one and two routes; X3 and a non-trivial `Within`. East to west is 200 through `centre`; the model prints the table |

IDs sort as `far_south` 0, `north` 1, `south` 2 in `three`, and `centre` 0, `east` 1,
`north` 2, `south` 3, `west` 4 in `five`. The overlay is a document of tables
`[region.<from>.edge.<to>]`, layered after the variant like any overlay
(`spec-config.md` 6.4).

## 13. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9 (FR-STB-003). Fields as in
its section 3; the register owner assigns the `STB-Lnn` numbers, and the entries here
are `REG-An`.

**This spec closes no feedback loop.** By 9.1 of the checklist: it reads no price,
inventory, income, count of agents or share, and what it writes (a region of an
account, a status of an edge) is read by no rule of this spec. The distances it
supplies are inputs to the delays of the loops of INF, TRN, MIG and MER (a rumour, a
journey, a migration, an arbitrage), and those specs enter them. By 9.6, no delay of
this spec exists to compare with a window or a season. It adds three hazards and no
damper (9.3: there is nothing here that limits a loop).

**REG-A1. A region that cannot be reached.** *Kind:* hazard. *Loop:* none. A region
that imports an upkeep good (VIA `region_not_self_sufficient`) and has no path from a
region that produces it is a closed economy that starves, and the cause is the graph,
not the economy. A closure (7) can create the situation in a run. *Owner:* REG 4.5, 7.
*Damper:* none by design; the warnings `region_isolated` and
`graph_not_strongly_connected` (9.2), VIA's warning, and the event `EdgeStatusChanged`
make it visible. *Undamped:* a closure that partitions the realm. *Condition:* every
region that imports has a path from a region that supplies it, at construction; in a
run the condition is the shock's. *Test:* a new scenario S28 (below). *Status:* hand.

**REG-A2. A small market is a thin market.** *Kind:* hazard (the cost of N-026).
*Loop:* none of its own. Splitting a town from its hamlets gives each hamlet a market
with a few agents; a thin market's price is mostly its carried-forward price
(`spec-clearing.md` 5.4), so the belief and price loops (L01 to L09) run on few orders,
and each hamlet's pool and failure ladder are its own (L39, L63). Trade between them
needs a carrier and a margin (TRN, MER), which do not exist yet. *Owner:* REG 3.4.
*Damper:* none in this spec; the short edge makes the arbitrage cheap once TRN exists,
and a setting that does not want the effect writes one region. *Condition:* a region
holds enough agents to keep the group floor `MinAgents` (D10) or is accepted as thin.
*Test:* a new scenario S29 (below). *Status:* hand.

**REG-A3. A household takes its money with it.** *Kind:* bias. *Loop:* none of its own.
A relocation moves money and goods between regions with no flow (6.3). The money of an
arriving household enters the destination's demand and the sender's loses it; the loop
that follows (migration raises demand and prices in the destination, which lowers the
ratio that triggered it) is MIG's, and the hook is in the register (4.8, MIG row). The
bias here is that a report of money by region changes with no flow, which the
relocation totals (6.4) reconcile. *Owner:* REG 6. *Damper:* none needed. *Condition:*
none. *Test:* S14 with a household that moves. *Status:* hand.

**Scenarios proposed** (for section 8 of the register; REF builds them):

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| S28 | A realm of three regions in a chain, with a closure of the middle link for a stated number of ticks and then a reopening, on a pack with trade on | the cut region's households fall no faster than in the closed-region variant (F3) while cut; the price gap between the two ends rises while cut and returns inside a stated band after the reopening; the event `EdgeStatusChanged` is emitted at both changes. Waits for TRN, MER and INT | REG-A1, L62 |
| S29 | A town as one region, and the same town as a town and two hamlets joined by a short edge, with the same total households | the thin hamlet markets are reported (price age, unmatched units); the dispersion of the three prices stays within a stated band once trade is on; the one-region run is the control. Waits for TRN and MER | REG-A2 |

## 14. What was run

`models/reg/paths.py` (Docker, standard library only; see its README). It is a
prototype of the definitions of 4.3 and 4.3.5, not of the library:

- **The tie-break definition.** On 3 000 random directed graphs of 2 to 7 regions with
  lengths 1 to 6 (so ties are common) and an edge density of about 45% (so asymmetry and
  unreachable pairs are common), `dist`, `NextHop` and `Path` as defined in 4.3.1 equal
  the shortest distance and the lexicographically smallest shortest path found by
  enumerating every simple path. 2 750 of the pairs had a tie at the first step and 9 531
  were unreachable. The strongly connected components are labelled as 4.3.5 says on
  every graph.
- **The tables of section 12**, including the closure, the factor, the renamed town and
  the five-region proposal, and the order independence of the edge list.
- **The bound of 4.2**: 999 edges of 10^9 are below 10^12 and far below `int64`.

It does not show that the Go implementation is correct (a test in the module must), and
it measures no speed: the operation counts of 4.4.3 are counts, not timings (N-015).
It is not a model of an economy, and no number in it is a result about stability.

## 15. Notes settled

For `AGENTS.md`: each open note tagged `REG` or `ALL`, and the others that name a
region, is applied, rejected or left open here, with the text to append in the
handoff.

| Note | Decision | Where |
|------|----------|-------|
| N-010 Regions | **Applied; close.** The reference-price override is 5.2; one settlement per region is 3.1; the estate of each region is created in 3.6; changing an account's region is section 6, with the hook for goods in transit in 6.5. What remains is MIG's and TRN's: when and why a household moves, and the journey | 3, 5, 6 |
| N-026 One settlement per region | **Applied; close.** 3.1.3 and 3.4, with the cost in REG-A2 and example C | 3.1, 3.4, 12.3, 13 |
| N-030 Five regions with different climates | **Leave open**, with this learned: the regions of `five` need edges for the trade overlay, and a proposal is in 12.4. The yield by region is PRO's and the values RES's; REG adds nothing to them but the rules of 5.3 | 12.4 |
| N-084 Scenarios that wait for trade | **Leave open**, with this learned: REG supplies the edge format and a proposal for the edge part of `overlay/trade.toml` (12.4), and two scenarios, S28 and S29. X1 and X2 still wait for TRN, MER and INF | 12.4, 13 |
| N-016 Keep the parameter registry true | **Applied** for this spec: the new fields have rows (8, handoff b), and no unnamed parameter was found; the bounds are in `bounds.go` terms | 8 |
| N-017 Words and names | **Applied**: `region`, `edge`, `distance`, `component`, `status`, `route` are the terms; "settlement" is the word for what a region is, not a type; a distance is not a duration; the setting words appear only in flagged examples | 1, 4.1.2, 12 |
| N-027 No default macro constant is stable | **Applied**: the spec sets no default that affects a price, wage or rate, and every bound is labelled provisional | preamble, 8 |
| N-009 What the world and tick phases must decide | **Leave open**, with this added: estate accounts and pseudo-agents are created with the regions, in region order, before any household (3.6.1); clearing and events by ascending `RegionID` (3.3.2); a relocation needs no reservation open and is placed after settlement and before the next admission (6.1.2) | 3.6, 6.1.2 |
| N-012 Events, metrics and log keys | **Leave open**, with this added: the events `AgentRelocated` and `EdgeStatusChanged`, the log keys `edges`, `components`, `from_region`, `to_region`, `distance`, `closed`, and the metric `economysim.household.relocations` (10) | 10 |
| N-020 What each spec contributes to a checkpoint | **Leave open**, with this added: REG saves the non-default edge statuses only (11) | 11 |
| N-021 Anything that creates or destroys money or goods | **Leave open**, with this added: REG adds no faucet or sink and no ledger reason; a relocation is not a flow (6.4.3) | 6.4 |
| N-036 Production's draws, phases and shocks | **Leave open**, with this added: the yield-shock slot and the endowment keys use `RegionID`; the edge status of section 7 is a second slot of the same shape, for INT | 5.3.6, 7 |
| N-038 Seed prices, newcomers and migrants | **Leave open**, with this added: `relocate` leaves a migrant's beliefs untouched, so MIG's default of reseeding on arrival is MIG's to execute (6.3) | 6.3 |
| N-076 A carriage market is not a goods market | **Leave open**, with this added: a route is an edge and its key is the `EdgeID` (4.7); where the market lives is still TRN's and CAR's | 4.7 |
| N-046 Register entries owed by unwritten specs | **Leave open**, with this added: REG has its entries (13) and the rows for TRN, MIG and INF in `spec-stability.md` 4.8 now depend on the distances of 4.3 | 13 |

Proposed new notes are in the handoff.

## 16. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-REG-001 | A configuration with no region MUST be rejected with `ErrNoRegions`, and a world of one region and no edges MUST be valid (2, 9.1). |
| FR-REG-002 | Regions MUST be sorted by ID, bytewise, before a world is built or a hash taken; the `RegionID` of a region MUST be its index in that order from 0; and two configurations that differ only in the order of their regions, edges or overrides MUST give equal graphs, equal hashes and equal runs (3.3.1). |
| FR-REG-003 | Every iteration over regions, and every output ordered by region (clearing order, events, snapshot rows, tables in a hash or a checkpoint), MUST be by ascending `RegionID` (3.3.2). |
| FR-REG-004 | `Name` and `Lookup` MUST invert each other for every region, `Lookup` of an unknown ID MUST give `false`, and `Name` of a `RegionID` outside `[0, n)` MUST give the empty string (4.3.2). |
| FR-REG-005 | At construction the world MUST create, for each region in ascending `RegionID` order and before any household, one estate account of kind Estate in that region and one estate pseudo-agent bound to it; the pseudo-agent of the region with `RegionID` `r` MUST have the agent ID `r`; and the estate accounts MUST precede every agent account in account-ID order (3.6.1). |
| FR-REG-006 | A region MUST have exactly one estate, created whether or not the region has a group and whether or not the estates module is on (3.6.2, 3.6.4). |
| FR-REG-007 | A region with no group MUST be valid and draw the warning `region_without_agents`, and a region that loses all its agents in a run MUST remain a region with its market and its estate (3.5). |
| FR-REG-008 | The set of regions MUST NOT change after construction (2). |
| FR-REG-009 | An edge whose `From` or `To` is not a defined region MUST be rejected with `ErrUnknownRegion` (4.1.3). |
| FR-REG-010 | An edge whose `From` equals its `To` MUST be rejected with `ErrSelfEdge` (4.1.3). |
| FR-REG-011 | Two edges with the same `From` and `To` MUST be rejected with `ErrDuplicateID`, and the edges `a > b` and `b > a` MUST be valid together and independent, with different distances if so configured (4.1.3, 4.1.4). |
| FR-REG-012 | An edge `Distance` below 1 (zero and negative included) or above `MaxDistance` MUST be rejected with `ErrOutOfBounds` naming its path (4.1.3, 9.1). |
| FR-REG-013 | More than `MaxEdges` edges MUST be rejected with `ErrTooMany` (4.2). |
| FR-REG-014 | Edges MUST be sorted by `(From, To)` as `RegionID`s, the `EdgeID` of an edge MUST be its index in that order, and a change of edge status MUST NOT change any `EdgeID` (4.1.6). |
| FR-REG-015 | A configuration with no edge MUST be valid and give one component per region, whatever the number of regions (4.3.5, 4.5.1). |
| FR-REG-016 | `Distance(a, a)` MUST be `(0, true)`; for `a != b` `Distance(a, b)` MUST be the least total effective length of a walk along open edges from `a` to `b`, and `(0, false)` when there is none (4.3.1). |
| FR-REG-017 | `Distance(a, b)` and `Distance(b, a)` MUST be allowed to differ when the edges do, and one may be defined while the other is not (4.1.4; example A of 12.1 and the one-way variant of 12.2). |
| FR-REG-018 | `NextHop(a, b)` MUST be, for `a != b` with `b` reachable, the smallest `RegionID` `v` among the open out-neighbours of `a` with `w(a, v) + dist(v, b) = dist(a, b)`, and `false` otherwise (4.3.1). |
| FR-REG-019 | `Path(a, b)` MUST be `a` followed by repeated `NextHop` to `b`, which is the lexicographically smallest of all shortest paths as sequences of `RegionID`s; `[a]` when `a` equals `b`; `(nil, false)` when unreachable (4.3.1). |
| FR-REG-020 | `Distance`, `NextHop` and `Path` MUST NOT depend on the order in which the edges and regions were listed, on map iteration or on the algorithm, and MUST equal the result of enumerating every simple path on every directed graph of up to seven regions in the test (4.3.1, 4.4.2). |
| FR-REG-021 | A tie between two shortest paths MUST resolve by `RegionID`: renaming a region so that it sorts differently MUST be able to change the path and listing the edges differently MUST NOT (example C of 12.3). |
| FR-REG-022 | `Edge(a, b)` MUST return the direct route, closed or not, with its configured and effective distance; and an edge longer than the shortest path through other edges MUST be valid, with `Edge(a, b).Distance` greater than `Distance(a, b)` (4.1.5). |
| FR-REG-023 | `Out(a)` and `In(b)` MUST list the open edges in ascending `To` and ascending `From` with their effective length, and a closed edge MUST NOT appear (4.3.2). |
| FR-REG-024 | `Within(a, limit)` MUST list every region other than `a` at shortest distance at most `limit`, in ascending distance and, for equal distances, ascending `RegionID`, and MUST be empty for a negative `limit` (4.3.3). |
| FR-REG-025 | A `RegionID` or `EdgeID` outside its range in any query MUST give the zero result and `false` and MUST NOT panic (4.3.3). |
| FR-REG-026 | A list returned by a query MUST be a fresh slice: changing it MUST NOT change the graph or any later answer (4.3.3). |
| FR-REG-027 | Components MUST be strongly connected components of the open graph, numbered from 0 in ascending order of the smallest `RegionID` in each; `Component(r)` MUST give the number of `r`'s component and `Components()` the count (4.3.5). |
| FR-REG-028 | A `Graph` MUST NOT change after it is built: a `Graph` taken before `SetEdgeStatuses` MUST give the same answers after it, and `World.Graph()` MUST return the new one (4.3.2, 7.3.2). |
| FR-REG-029 | Path sums and effective lengths MUST use checked arithmetic and report overflow, and the largest case allowed by the bounds (999 edges of the largest effective length) MUST NOT overflow (4.2, 4.2.1). |
| FR-REG-030 | `NewGraph(cfg)` MUST run the graph rules of section 9 and build the graph without a `World`, and the world MUST use the same function (4.3.2). |
| FR-REG-031 | The warnings `region_isolated`, `graph_not_strongly_connected` (carrying the count of components) and `edge_not_shortest` (for an edge strictly longer than the detour) MUST be raised as described, MUST NOT fail validation, and a world of one region and no edge MUST draw none of the first three (4.5.2, 9.2). |
| FR-REG-032 | The effective reference price of a commodity in a region MUST be the `RegionPrices` row's price if there is one, else the commodity's `ReferencePrice`, and every use that `spec-goods.md` 6.3 lists MUST use the effective price (5.2.2). |
| FR-REG-033 | A `RegionPrice` naming an undefined region or commodity, a second one for the same region and commodity, a price outside `[1, MaxPrice]`, or more than `MaxRegionPrices` of them MUST be rejected with `ErrUnknownRegion`, `ErrUnknownCommodity`, `ErrDuplicateID`, `ErrOutOfBounds` and `ErrTooMany` respectively (5.2, 9.1). |
| FR-REG-034 | A `RegionPrice` equal to the commodity's price MUST draw the warning `override_equals_default` and MUST NOT fail validation (5.2.4). |
| FR-REG-035 | A row of any region-keyed table (5.1.3) naming an undefined region MUST be rejected with `ErrUnknownRegion`, a second row for one key with `ErrDuplicateID`, an absent row MUST give the default without a warning, and rows MUST be sorted by region then subject (5.3). |
| FR-REG-036 | `World.ReferencePrice(r, c)` MUST equal the effective reference price for every region and commodity, MUST be resolved at construction and MUST NOT allocate (5.2.4, 5.3.5). |
| FR-REG-037 | The resolution-floor rule and the basket rule MUST use the effective reference price: a region whose override puts its cheapest tradable commodity below the floor MUST be rejected with `ErrOutOfBounds`, and a region whose `ReferenceBasket` differs from the cost of its baseline at its effective prices by more than `BasketTolerance` with `ErrBasketMismatch` (5.2.3; example A). |
| FR-REG-038 | A warm-start seed price MUST replace the seed price only: the reference band and the other uses of the reference price MUST stay on the effective reference price (5.2.2). |
| FR-REG-039 | `relocate` MUST check, in this order, the region, the agent, the account being open, the kind, the region being different, the absence of a reservation and the registered obligations, MUST return the error of 6.1.3 for the first that fails, and a refused call MUST leave the ledger and the agent unchanged (6.1.3). |
| FR-REG-040 | A successful relocation MUST set the account's region and change no balance, no lot and no reservation; the total over all accounts of money and of each commodity MUST be unchanged; and no transfer, reason, boundary account or flow total MUST be involved (6.2, 6.3). |
| FR-REG-041 | Flows after a relocation MUST be attributed to the new region and flows before it MUST stay with the old one (6.2). |
| FR-REG-042 | The ledger MUST keep, for the current tick, the money and the units of each commodity moved with relocated accounts for each ordered pair of regions, in ascending `from`, then `to`, then money before commodities in commodity order, cleared at the start of each tick and not saved in a checkpoint (6.4.1). |
| FR-REG-043 | An estate, escrow, transit or boundary account MUST be refused with `ErrNotRelocatable`, and the estate pseudo-agent MUST never change region (3.6.3, 6.1.3). |
| FR-REG-044 | In version 1 `relocate` MUST refuse `NoRegion` with `ErrUnknownRegion` and an agent MUST always be in a region (6.1.3, 6.5.3). |
| FR-REG-045 | The derived list of the agents of a region MUST reflect a relocation at once, and each list MUST be in ascending agent ID (3.1.1, 6.2). |
| FR-REG-046 | `AgentRelocated` MUST be emitted once per successful relocation after it is complete, with the agent, role, `From`, `To`, weight, money, units and cause, MUST NOT be emitted for a refused call or for a pseudo-agent, and MUST contain no map (10.1). |
| FR-REG-047 | When a registered region-bound obligation is open for an account, `relocate` MUST refuse with `ErrRegionBound` naming it, and with no obligation registered, as in version 1, it MUST never do so (6.5.2). |
| FR-REG-048 | A relocation MUST leave the agent's role, weight, strategy, job, beliefs, welfare record, `Hunger`, `Privation`, trainee state and mailbox unchanged (6.3). |
| FR-REG-049 | The effective length of an open edge MUST be `max(1, ceil(Base * FactorBP / 10 000))` computed with checked arithmetic, a closed edge MUST be absent from `Out`, `In`, `Distance`, `NextHop`, `Path`, `Within` and the components, and `Edge` MUST still return it with `Closed` set (7.2). |
| FR-REG-050 | `SetEdgeStatuses` MUST be atomic: an edge that is not configured MUST be rejected with `ErrUnknownEdge`, a factor outside `[1, MaxEdgeFactorBP]` with `ErrOutOfBounds` and an edge twice with `ErrDuplicateID`, and when any change fails none MUST be applied (7.3.1). |
| FR-REG-051 | A successful batch MUST rebuild the tables once, replace the world's graph, and emit `EdgeStatusChanged` once per edge whose status changed, in ascending `EdgeID`, after the new graph is in place, and none for a change equal to the current status (7.3.2, 4.4.4). |
| FR-REG-052 | The checkpoint MUST contain the edges whose status is not the default, in ascending `EdgeID`, and nothing else of this spec; and a world restored between ticks MUST answer every query of 4.3.2 as an uninterrupted one does (11). |
| FR-REG-053 | The configuration hash MUST cover `Regions`, `Edges` and `RegionPrices` in canonical order, every field except `Region.Label` and `World.DistanceLabel`, so that changing a distance or a price changes the hash and changing a label does not (11). |
| FR-REG-054 | A configuration read from a file and the same configuration built in Go MUST have equal edges and overrides, equal hashes and equal runs, and the file MUST write an edge as `[region.<from>.edge.<to>]` with `distance` and the overrides as `[region.<id>.reference_price]` keyed by commodity; an unknown key MUST be rejected with `ErrUnknownKey` and a non-integer distance with `ErrNotInteger` (4.1.1, 5.2.1). |
| FR-REG-055 | The bounds `MaxDistance`, `MaxEdges`, `MaxEdgeFactorBP`, `MaxRegionPrices` and `MaxLabelBytes` MUST be constants in `bounds.go` with the values of 4.2 (4.2). |
| FR-REG-056 | The "world created" line MUST carry `edges` and `components` beside `regions`, each warning of 9.2 MUST be logged once at `Warn` with `rule` and `path` at construction, a relocation MUST be logged at `Debug` only and guarded by `Enabled`, and no `Info` MUST be logged inside the tick (10.2). |
| FR-REG-057 | The events `AgentRelocated` and `EdgeStatusChanged` MUST be constructed only when an observer was supplied, and a run with and without a logger and observer MUST give equal results (10.1, 10.6). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-REG-001 | `Len`, `Name`, `Lookup`, `Edges`, `Edge`, `EdgeAt`, `Distance`, `NextHop`, `Component`, `Components` and `World.ReferencePrice` MUST NOT allocate (4.3.3, 5.2.4). |
| NFR-REG-002 | Building the graph of 100 regions and 400 edges SHOULD take under 10 ms and of the bounds (1 000 regions, 20 000 edges) under 5 s. Both figures are placeholders from the operation counts of 4.4.3 and are unmeasured (N-015). |
| NFR-REG-003 | The graph validator and the queries MUST have fuzz tests (testing.md 7.4) asserting that they never panic for any edge list, region ID or limit, and that for every valid graph `Distance` obeys the triangle inequality, `Path` has the length `Distance` and each consecutive pair of `Path` is an open edge. |
| NFR-REG-004 | The graph algorithms MUST live in `internal/graph` over primitive slices, import the standard library only, and know no region name, commodity or role (architecture.md 2.9, 9.1). |

## 17. Open questions

1. **Is the edge status REG's or INT's?** The brief gives REG the graph and INT the
   shocks. Section 7 puts the mechanism (the status, the rebuild, the event) here and
   leaves the public call to INT, because a closed road is a change to the graph and the
   tables are REG's. If the supervisor prefers INT to own it, section 7 moves with
   `FR-REG-049` to `FR-REG-051`. Next step: the supervisor.
2. **Free trade and components.** `spec-viability.md` 9.4 adds a lane between every pair
   of regions for the pooled check. With edges, a pooled solution across components that
   cannot reach each other is optimistic. Whether the check uses `Components` is VIA's.
   Next step: `spec-viability.md`.
3. **A `*Graph` across the API.** 4.3.4 argues that an immutable `Graph` meets the aim of
   architecture.md 12.4 although it is not plain data. If the rules owner disagrees, the
   alternative is a plain-data copy of the tables (12 MB at the bound) or query methods
   on `World` only. Next step: rules owner.
4. **An abandon reason for a relocated job.** 6.3 proposes `region_changed` in
   `spec-production.md` 5.4 and leaves the decision to abandon to MIG. Next step: PRO
   and MIG.
5. **Region in an event: ID or name.** 10.1 uses the `RegionID`; the `otelobserver`
   adapter and a consumer need the name for an attribute. Next step: OBS decides for every
   event at once.
6. **Founding or merging regions in a run.** Out of scope for version 1 (2). A consumer
   who needs a region that appears (a colony founded by the player) needs the tables to
   grow, the ledger to open an estate and a checkpoint to record it. Not designed.
7. **A shorthand for a symmetric pair.** A file with many symmetric routes writes two
   tables for each. A key such as `both = true` would halve the lines and is presentation
   in the loader, not a core feature. Next step: `spec-config.md` 6.
8. **Append forms.** `AppendOut` and `AppendWithin` would let a per-tick caller avoid the
   allocation of 4.3.3. They can be added without a break (architecture.md 6.6); add them
   if MIG or INF shows the need.
9. **Where in the tick a relocation happens, and in what order.** 6.1.2 recommends the
   entry-and-exit phase. When several relocations in one tick compete for one thing (a
   pool, a place in a traineeship), their order is MIG's: this spec requires only the
   total order of agent ID if the caller has no better one.
10. **Distances from coordinates.** A pack could give each region a position and derive
    the distances. That belongs in pack tooling, not in the core (3.2).

## 18. Where this lives in the code

| Concern | Home |
|---------|------|
| `Region` (the entity, with the fields of other specs), `RegionID`, `NoRegion`, `Edge`, `EdgeID`, `RegionPrice`, `EdgeView`, `Hop`, `EdgeStatus`, `Graph`, `NewGraph`, `World.Graph`, `World.ReferencePrice`, `World.SetEdgeStatuses` | root package, `regions.go` |
| The estate creation at construction, `relocate`, the lists of agents by region | root package, `world.go` |
| The single-source searches, the next-hop table, the components, the lengths with a factor, over primitive slices | `internal/graph` (standard library only, converted inline at the call, architecture.md 2.9) |
| `Relocate(account, region)` and the relocation totals | `internal/ledger` |
| `DefaultRegion()` | root package, `defaults.go` |
| The bounds of 4.2 | root package, `bounds.go` |
| Validation, the sentinels `ErrNoRegions`, `ErrSelfEdge`, `ErrUnknownEdge`, `ErrUnknownAgent`, `ErrNotRelocatable`, `ErrSameRegion`, `ErrOpenReservation`, `ErrRegionBound` | root package, `validate.go`, `errors.go` |
| Events `AgentRelocated`, `EdgeStatusChanged` | root package, `event.go` |
| Reason and cause constants | `internal/logkeys` |
| The file shape of an edge and an override | `config/` (`spec-config.md` 6) |
