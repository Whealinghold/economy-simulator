# Spec: Polity: jurisdictions over regions, the borders between them and the treasury of a governed one

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-POL-NNN` and
`NFR-POL-NNN`, which tests cite (testing.md section 4). They are in section 16.

Layer: module, later stage ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md),
[ADR 0003](../adr/0003-version-1-scope.md)). It is off unless `Config.Modules.Polity`
is present, and the core does not depend on it. A world with the module off, or with
the module on and no jurisdiction governed, is exactly the world without it (FR-POL-019).
Needs `spec-regions.md` (3, 4, 5, 6, 7, the region, the graph, the override mechanism,
the refusals of a relocation, the edge status), `spec-ledger.md` (3.1, 7.2, 9.3, the
account kinds, the reason `initial`, the sector identity), `spec-world.md` (5.2, 5.3, 8,
9.4, the order of accounts and agent IDs, the calls between ticks, no rate from an
aggregate) and `spec-estates.md` (2.7, the pseudo-agent). It is read by the specs that
come after it: fiscal (FIS), institutions (ORG), policy (PLC), and, for a holder, land
(LND) and storage (STO).

**Every number in sections 4.4, 8 and 12 is a placeholder** except where a
derivation is shown. Nothing has been run against the library, bazaarBot has not been
run, and no acceptance band exists (notes N-015, N-027). `models/pol/chain.py`
(section 14) checks the definitions of section 4.1 on random forests; it measures no
speed and is not a claim that any default is stable. No default of this spec is a
macro constant: the spec sets no price, wage, rate or share, defines no law and reads
no price, stock, income or count.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer, never a household (N-001, N-017). The setting words of the
examples (`crown`, `province`, `kingdom`, `duchy`, `county`, `parish`, `federation`,
`station`) appear only in the **flagged examples** of section 12
and in sentences marked "for example"; they are data, never branched on (architecture.md
9.1; `AGENTS.md`, setting-agnostic). Several written specs use "realm" informally for
the whole set of regions of a world (`spec-regions.md` 1, `spec-external-sector.md`
1, `spec-indicators.md` 8); this spec does **not** give the word a type, and its own
tree has no level called that.

## 1. Purpose and scope

A region is one settlement and one market, linked to others by directed edges
(`spec-regions.md` 3.1, 4.1). Some settings group regions under authorities that nest
(for example a sovereign over provinces, a federation over member states, a station
authority over its decks: invented settings, flagged in section 12). The grouping matters to later specs because a levy, a toll, a charter or a
price control applies **where** an authority holds sway, and because a good crossing a
border passes from one authority to another. `spec-regions.md` 3.1.2 and 5.4 leave the
grouping to this spec and say only what it must not be: a table that names regions,
never a field that regions read. This spec fixes that table and the questions a reader
may ask of it. It adds a layer and no behaviour of its own. It covers:

- the **jurisdiction** as data: an ID, an optional parent, an optional seat, and the
  **membership** of regions, which together form one tree over the regions (section 3);
- the deterministic **queries**: the chain of jurisdictions of a region, whether a
  jurisdiction covers a region, the lowest one two regions share, the regions a
  jurisdiction covers, the **crossing** of an ordered pair of regions (what is left and
  what is entered) and the edges between two jurisdictions (section 4);
- the one **actor** a jurisdiction has, its **treasury**: what exists, where it sits in
  the ledger and the agent ID space, and what this spec does and does not do with it
  (section 5);
- the **hooks** by which later specs make a rule depend on a jurisdiction: how rows keyed
  by jurisdiction combine along a chain, a table of the readers and what each reads, and
  what is not offered (section 6);
- where in the tick the module acts (nowhere), what is fixed for a run, and what a
  restore needs (section 7);
- parameters, validation, observability, checkpoint state, register entries and
  requirements (sections 8 to 11, 13, 16).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The region, the edge, the distance, the shortest path, the edge status that closes or lengthens a route | `spec-regions.md` (REG) |
| What a law, a tax, a toll, a tariff or a grant **is**, its rate and its cap; where a treasury's money comes from and goes | `spec-fiscal.md` (FIS), with `spec-money-supply.md` (MON) for the flow mechanism |
| Associations, charters, monopolies, price controls: rules that bind who may do what | `spec-institutions.md` (ORG) |
| How a policy actor chooses a rate or a release over time: a policy actor, a schedule, a rule that reacts to a measure, a player | `spec-policy.md` (PLC) |
| Who holds an endowment, who owns a store | `spec-land.md` (LND), `spec-storage.md` (STO): they may take a jurisdiction as a holder through its treasury (5.4) |
| Loss, raids, conflict, a border that is fought over | `spec-risk.md` (RSK) |
| The size class of a settlement | `spec-settlements.md` (SET); SET classes regions, not jurisdictions |
| The outside world, its nodes and gates, a tariff on an import | `spec-external-sector.md` (EXT) for the mechanism; FIS for the tariff |
| Why a household moves, and whether a border stops it | `spec-migration.md` (MIG); 6.4 says what is not offered |
| Derived measures over the regions of a jurisdiction | `spec-indicators.md` (IND): a preset ask (section 15) |
| The tick, the draws, the options | `spec-world.md` (WLD) |

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is a jurisdiction? | A named node of one tree over the regions: an ID, an optional parent, an optional seat. It has no behaviour, no law and no state of its own | The grouping is a key that later specs read. Everything that makes it matter is theirs (`spec-regions.md` 3.1.2, ADR 0001 decision 7) |
| 2 | A tree or a DAG? | **A tree** (a forest: several roots are allowed). A jurisdiction has at most one parent and a region at most one direct jurisdiction | A second parent needs a rule for which of two rates applies and makes "the border between two regions" a set of paths. Overlap, such as a league that spans two top-level authorities, is a body with members, which is ORG's, never a second parent |
| 3 | What does a region belong to? | At most one jurisdiction, **directly**; the **chain** to the root is derived. A region with none is ungoverned | A table keyed by region, one row per region, is the mechanism of `spec-regions.md` 5.1 (N-104) |
| 4 | Does a value inherit down the tree? | **No** region field does (`spec-regions.md` 5.4.1): a reference price, a yield or an upkeep is compared with the world's default and never with an enclosing jurisdiction's. For rows that **a reader keeps** keyed by jurisdiction, this spec fixes how rows along a chain combine (6.1) | One owner per fact. The combining rule is written once, as `spec-regions.md` 5.1 wrote the override rule once |
| 5 | What is a border? | A property of an **ordered pair of regions**: the jurisdictions that cover the first and not the second (left, innermost first) and the second and not the first (entered, outermost first). It is derived, not stored, and has no configured attribute | A configured attribute of a border would be a law, which is FIS's or ORG's. An edge is REG's; this spec adds a question to ask of it |
| 6 | Who is the actor of a jurisdiction? | Its **treasury**: an account of a new kind Treasury in the region of its seat, with a pseudo-agent that gives it an agent ID. A jurisdiction has one only if it has a **seat** (it is **governed**) | FIS needs an account that holds money and does not obey the pool's policy; PLC needs one identity per governed jurisdiction to attach a policy to; LND and STO need a holder with an agent ID. An ungoverned jurisdiction is a label and creates nothing |
| 7 | Why not the seat region's pool? | The pool has an automatic policy (release as relief, funding of entrants, `spec-estates.md` 2.6, `spec-spawner.md` 7.3) that would spend a treasury without anyone deciding; it also takes the money of the dead of one region. A treasury is a balance that only a named rule moves | One rule per account. The pool stays the region's |
| 8 | Why not an external agent? | An external agent is created at run time by a call and is not in the configuration (`spec-integration.md` 4.1). A jurisdiction exists in the configuration, so its actor must too | A pack and its checkpoint must be a function of the configuration. A consumer may still pay an external agent from a treasury as that authority's household; that is FIS's flow |
| 9 | Can membership change in a run? | **Not in version 1.** The tree and the membership are configuration, hashed and fixed for a run. A transfer of a region is reserved with its exact shape (7.4, open question 1) | No consumer asks yet (the pattern of N-131). A transfer is cheap to add as a slot, and every reader that caches by jurisdiction would need the rule stated before it exists |
| 10 | Does the module change a result by being on? | **No, while no jurisdiction is governed**: no account, no pseudo-agent, no draw, no event, no flow, no ID shifts (FR-POL-019). A governed jurisdiction shifts household agent IDs by the number of treasuries (N-117, POL-A1) | A grouping for measurement and for flags must be safe to add to any pack. The cost of a treasury is stated, not hidden |
| 11 | Where is it in the tick? | Nowhere. It has no position, no draw, no event of its own and no slot (7) | It reads nothing that moves and writes nothing but a starting balance at construction |
| 12 | Are the levels named? | No. The depth of a jurisdiction is a number derived from the tree (a root has depth 0). A pack may give a jurisdiction a label | A setting's words for ranks belong to the pack. A rule that applies "at one level of the tree" is a row keyed by those jurisdictions, never a branch on a name |
| 13 | Is this spec a faucet or sink, or does it add a loop? | Neither. It reads no price, stock, income or count, and moves money only once, at construction, under the existing reason `initial` | `spec-stability.md` 9.1. Its hazards are in section 13 |

## 3. The jurisdiction

### 3.1 What it is

3.1.1. A **jurisdiction** is an entity of the configuration with an ID and a position
in a tree. It is a **key** that the readers of section 6 attach their rows to. It holds:

- its **parent** jurisdiction, or none (a **root**);
- the regions that are its **direct members** (3.4);
- optionally a **seat**, a region from which it is governed, which makes it
  **governed** (3.5).

3.1.2. A jurisdiction **covers** a region if the region is a direct member of it or of
one of its descendants. The regions it covers are derived (4.1).

3.1.3. A jurisdiction is not a region: it has no market, no estate, no distance, no
position in the graph and no households of its own; "the households of a jurisdiction"
is the households of the regions it covers, derived by the reader. A jurisdiction is
not a country or a state: nothing in the core is told that one is. A setting that
wants a roster of independent states writes several roots.

### 3.2 Identity and order

3.2.1. The jurisdictions of a `Config` are sorted by ID, bytewise, before a world is
built or a hash is taken (`spec-config.md` 3.4). The **`JurisdictionID`** of a
jurisdiction is its index in that order, from 0. It is a small named value type
(architecture.md 3.3):

```go
// JurisdictionID is the index of a jurisdiction in ascending order of its ID
// string. It is not stable across configurations: adding a jurisdiction can
// renumber the others. Use the string ID to persist or exchange one.
type JurisdictionID int

// NoJurisdiction means "none": a query returns it where a jurisdiction does not
// exist, or where no jurisdiction covers both of two regions.
const NoJurisdiction JurisdictionID = -1
```

3.2.2. The ID syntax is that of a region (`[a-z][a-z0-9_]*`, at most 64 bytes,
`spec-config.md` 3.3), unique among jurisdictions. Jurisdictions and regions are
separate namespaces: a jurisdiction may have the ID of a region (a city that is both),
and a file names them by different tables (8).

3.2.3. Every iteration over jurisdictions in the library, and every output ordered by
jurisdiction, is by ascending `JurisdictionID`; a list of the regions of a jurisdiction
is by ascending `RegionID`; the children of a jurisdiction are by ascending
`JurisdictionID`. The tree order (parent before child) is never used as an iteration
order, so no list depends on how a file nested its tables. No iteration over a map
influences a result (architecture.md 7.5).

### 3.3 The tree

3.3.1. `Parent` names another jurisdiction, or is empty. A jurisdiction whose parent
chain returns to itself is rejected (`ErrJurisdictionCycle`, 9.1). The depth of a
jurisdiction is the number of its ancestors; a root has depth 0. The bound is on the
number of **levels**: a depth is at most `MaxPolityDepth - 1` (so at most
`MaxPolityDepth` levels, 8, a placeholder), and a chain has at most `MaxPolityDepth`
entries; a jurisdiction of depth `MaxPolityDepth` or more is `ErrOutOfBounds` with the
path of the jurisdiction that exceeds it.

3.3.2. A jurisdiction may have no children, one or many. A jurisdiction with one
child and no member of its own is allowed and draws no warning: a setting may nest one
authority over a single subdivision. A jurisdiction that covers **no** region draws the warning
`jurisdiction_empty` (9.2), because a row keyed by it can never apply.

3.3.3. There is no level table. A rule that wants to apply "to the second level" is a
row keyed by each jurisdiction of that depth, or a reader that asks for the depth
(4.2). Nothing branches on a jurisdiction's ID (architecture.md 9.1).

### 3.4 Membership

3.4.1. **Membership** is a table keyed by region, one row per region: the region and
the jurisdiction it is a direct member of. It follows the one override mechanism of
`spec-regions.md` 5.1 in **replace** mode (absence is the default, which is "none"):
the row is added to the list of 5.1.3 of that spec (N-104), with the errors of its
5.3: a region that is not defined is `ErrUnknownRegion`, a jurisdiction that is not
defined is `ErrUnknownJurisdiction`, two rows for one region are `ErrDuplicateID`, and
rows are canonical by `RegionID`.

```go
// Member says that a region is a direct member of a jurisdiction. A region has at
// most one row.
type Member struct {
	Region       string
	Jurisdiction string
}
```

3.4.2. In a file the membership is one table keyed by region, so that an overlay
document reassigns a region without touching a jurisdiction:

```toml
[modules.polity.member]
port  = "coast"
mill  = "inland"
```

3.4.3. **Any region may be a member**, an outside node of the external module
included: a node assigned to a jurisdiction is a foreign power's land, and a node
assigned to none is simply outside every jurisdiction, which is the usual case. An
outside node has no agents, so assigning one adds nothing but its place in a chain
and in a crossing (4.3).

3.4.4. **Ungoverned regions.** A region with no row is in no jurisdiction. Its chain
is empty. The crossing between two ungoverned regions is empty: nothing is left and
nothing is entered. A world in which no region has a row is the world without the
module, for every reader (FR-POL-001).

### 3.5 The seat and the governed jurisdiction

3.5.1. A jurisdiction with a non-empty `Seat` is **governed**. The seat names a region
that the jurisdiction **covers** (`ErrSeatNotCovered` otherwise) and that is not an
outside node (`ErrSeatOutsideNode`): the seat is where the treasury's account sits
(5.1), and an outside node has no market and no one to hold it.

3.5.2. A governed jurisdiction has one treasury and no more (5.1). Two governed
jurisdictions may share a seat: an outer jurisdiction and the inner one that holds its
capital are each governed from that capital, and each has its own treasury in that region.

3.5.3. A jurisdiction with no seat is a **grouping**: it has a place in the tree, a
chain, a crossing and rows keyed by it, and no account. A grouping may be the parent
of a governed jurisdiction and the other way round; being governed is not inherited.

3.5.4. `StartMoney` is the money the treasury is created with, in minor units, default
0, replace mode, `0` to `MaxMoney` (`ErrOutOfBounds`). It is meaningful only with a
seat; a `StartMoney` without a seat is `ErrStartWithoutSeat`.

### 3.6 A setting without a polity

3.6.1. A setting that has no authorities above the region writes no `Modules.Polity`.
Every query of section 4 answers "none" and no reader finds a row. A setting with one
authority over everything writes one jurisdiction and one row per region; it may leave
it ungoverned, and a single jurisdiction has no border (4.3.3).

### 3.7 What a jurisdiction is not

3.7.1. It holds no aggregate (a population, a money supply, a rate of tax paid): those
are measures, computed by IND from the regions it covers (ADR 0001 decision 7).

3.7.2. It does not own regions in the sense of LND: it holds no endowment, no title and
no market. A claim on an endowment is LND's; a jurisdiction may be a holder through its
treasury once LND says so (5.4).

3.7.3. It has no strategy and decides nothing. A policy actor that chooses a rate over time is
PLC's, attached to the jurisdiction (governed or a grouping, `spec-policy.md` 3.2) and reading the treasury of a governed one (5.4).

## 4. The queries

### 4.1 Definitions

4.1.1. Let there be `J` jurisdictions with `JurisdictionID`s `0` to `J-1`, `p(j)` the
parent of `j` or none, and `m(r)` the jurisdiction that region `r` is a direct member
of, or none. Then:

- the **chain** of a region is `m(r)`, `p(m(r))`, `p(p(m(r)))`, and so on to a root,
  **innermost first**; it is empty if `m(r)` is none. Because each jurisdiction has one
  parent, the chain is a path and has no repeated element (3.3.1);
- a jurisdiction `j` **covers** `r` if `j` is in the chain of `r`;
- the **common** jurisdiction of two regions is the first element of the chain of the
  one that is also in the chain of the other, or `NoJurisdiction`. The common elements
  of two chains are a suffix of both, because chains are paths to a root, so the first
  is the innermost jurisdiction that covers both;
- the **crossing** of an ordered pair `(a, b)` has two lists. `Exit` is the elements of
  the chain of `a` that are not in the chain of `b`, innermost first. `Enter` is the
  elements of the chain of `b` that are not in the chain of `a`, **outermost first**.
  The crossing of a region with itself is empty, and so is the crossing of two regions
  with the same chain. The crossing is defined for any two regions, edge or not.

4.1.2. The orders are fixed so that a reader that applies a rule on leaving or on
entering sees the jurisdictions in the order a good would pass them: out through the
smallest holder first, and in through the largest first. They depend on region and
jurisdiction IDs and on nothing else: not on the order of rows in a file, on a map or
on the algorithm (architecture.md 7.5, 7.6). `models/pol/chain.py` checks the
definitions against a second formulation by walking parents, on 2 000 random forests
(section 14).

4.1.3. **Why a chain and not a set.** A reader needs the order: the innermost
jurisdiction is the one whose row replaces the others (6.1), and the exit and entry
orders are the order of the rows a good meets. A set would force every reader to sort
by depth, and a sort by depth with ties by ID is the rule written here once.

### 4.2 The `Polity` value

4.2.1. The queries are methods of a value that holds plain data only (slices and
integers: no pointer into the world, no function field):

```go
// Polity is the jurisdiction tree and the membership of the regions: plain data,
// immutable. It cannot change during a run (7.4), so the world returns an equal value
// every time. Like Graph, each call returns a fresh deep copy that shares no storage
// with the world, and so do the lists it returns (architecture.md 12.4).
type Polity struct { /* unexported fields: slices and integers only */ }

func NewPolity(cfg Config) (Polity, error) // runs the polity rules of 9, then builds
func (w *World) Polity() Polity            // a deep copy of the world's value; the empty Polity if the module is off

func (p Polity) Len() int                                   // jurisdictions
func (p Polity) Name(j JurisdictionID) string               // "" if j is not a jurisdiction
func (p Polity) Lookup(id string) (JurisdictionID, bool)

func (p Polity) Parent(j JurisdictionID) (JurisdictionID, bool)   // false for a root or an unknown j
func (p Polity) Children(j JurisdictionID) []JurisdictionID       // ascending
func (p Polity) Depth(j JurisdictionID) (int, bool)               // a root is 0

func (p Polity) Of(r RegionID) (JurisdictionID, bool)             // the direct jurisdiction; false if none
func (p Polity) Chain(r RegionID) []JurisdictionID                // innermost first; empty if none
func (p Polity) Covers(j JurisdictionID, r RegionID) bool
func (p Polity) Common(a, b RegionID) JurisdictionID              // NoJurisdiction if none
func (p Polity) Regions(j JurisdictionID) []RegionID              // covered, ascending
func (p Polity) Crossing(from, to RegionID) Crossing
func (p Polity) Between(g Graph, a, b JurisdictionID) []EdgeID    // edges from a region covered by a into one covered by b

func (p Polity) Seat(j JurisdictionID) (RegionID, bool)           // false for a grouping
func (p Polity) Governed() []JurisdictionID                       // ascending

// Crossing is what a good leaves and enters in going from one region to another.
type Crossing struct {
	Exit  []JurisdictionID // covering the origin and not the destination, innermost first
	Enter []JurisdictionID // covering the destination and not the origin, outermost first
}
```

4.2.2. **Behaviour.** An ID outside its range in any query gives the zero result and
`false` (or `NoJurisdiction`, or an empty list), never a panic: it can come from a
caller's data (architecture.md 8.1). `Name`, `Lookup`, `Len`, `Parent`, `Depth`, `Of`,
`Covers`, `Common` and `Seat` allocate nothing; `Children`, `Chain`, `Regions`,
`Crossing`, `Between` and `Governed` return fresh slices (architecture.md 12.4).

4.2.3. **`Between(g, a, b)`** lists, in ascending `EdgeID`, the edges of the graph `g` whose
crossing (4.1.1) has `a` in `Exit` and `b` in `Enter`: an edge that leaves a region `a`
covers and enters one `b` covers, and is internal to neither. A closed edge is listed
(the status is on the `EdgeView`, `spec-regions.md` 4.3.2). Two nested jurisdictions
have no edge between them, since every region of the inner one is covered by the outer
one, and so does a jurisdiction with itself. It reads the graph's `EdgeAt` and this
polity's chains, so it changes nothing and costs `O(edges * depth)`.

4.2.4. **`World.Polity()`** returns a **deep copy** of the world's value, built fresh on
every call exactly as `World.Graph()` is (`spec-regions.md` 4.3.4), so it shares no
storage with the world and architecture.md 12.4 stands without an exception. The copy
is `O(J + n * depth)` and allocates: class 3 of `spec-integration.md` 6.1.3, with
`Graph`, so a consumer calls it once and keeps the value, which does not follow the
world (it cannot change, 7.4). It is a derived-table reader with no error result,
available on a failed world like `Graph` (`spec-world.md` 3.5.1); `spec-integration.md`
6.2 gets one row, class 3. The methods of the value are as 4.2.2 says.

### 4.3 Borders

4.3.1. A **border** is not a stored thing. For an ordered pair of regions it is the
crossing of 4.1, and it is "a border" when `Exit` or `Enter` is not empty. The
**border depth** of the pair is the depth of their common jurisdiction, or -1 when
there is none; it tells a reader at which level the two regions first belong together
(a depth of 0 means they share only a root, a larger depth a closer holder). It is a
derived read, `Depth(Common(a, b))`.

4.3.2. **An edge's attributes.** The attributes of an edge that this spec supplies are
all derived: whether it is a border, its `Exit` and `Enter` lists and its border
depth, from the chains of its two ends. They do not depend on the edge's status: a
closed or lengthened edge still crosses the same borders (`spec-regions.md` 7).

4.3.3. **A pair within one jurisdiction** that is the same chain has an empty crossing,
and a world with one jurisdiction has no border. A pair of regions in two jurisdictions
of one tree crosses every border between them: a good from a jurisdiction `c1` to a jurisdiction
`c2` whose parent `p2` differs from `c1`'s parent `p1` leaves `c1`, then `p1`, and enters
`p2`, then `c2`, in that order (the crossing is `Exit = [c1, p1]`,
`Enter = [p2, c2]`).

4.3.4. **A route of several edges** crosses the borders of each edge it traverses. A
reader that charges per edge iterates the path's edges (`Graph.Path`) and asks the
crossing of each; this spec does not offer the crossing of a whole route, because the
crossing of the two ends omits a jurisdiction the route leaves and re-enters.

4.3.5. **An outside node and its gate.** A gate region and an outside node are
connected by an edge (`spec-external-sector.md` 3.2). With the node in no jurisdiction,
the crossing of (gate, node) is the gate's whole chain as `Exit` and nothing as
`Enter`: what an import duty or an export duty would key on. The outside's trades are
made in the gate region's market through counterparties
(`spec-external-sector.md` 4), so a tariff is applied by its owner to those trades with
the gate's chain; this spec supplies the keys and no more.

### 4.4 Representation and cost

4.4.1. The polity keeps, as derived data built once at construction: the parent and
depth of each jurisdiction, the direct jurisdiction of each region, the chain of each
region (at most `MaxPolityDepth` entries), the children of each jurisdiction, and the
covered regions of each jurisdiction in ascending `RegionID`. All are derived from the
configuration; none is saved in a checkpoint (11).

4.4.2. **Operation counts** (not timings: nothing has been measured, N-015):

| Case | `J` | `n` regions | Depth | Build, order of | Tables |
|------|----:|------------:|------:|-----------------|--------|
| The first consumer | 4 | 5 | 2 | under 100 steps | under 200 bytes |
| A larger world | 40 | 100 | 4 | `n * depth + J log J`, about 650 steps | under 2 KB |
| The bound | 1 000 | 1 000 | 8 levels | `n * depth + J log J`, about 19 000 steps (9 000 for the tables, 10 000 for the sort) | about 60 KB |

A query is a table read (`O(1)`), a walk of one or two chains (`O(depth)`) or a copy of
at most `n` entries; `Between` is `O(edges * depth)`. None runs inside the tick loop
unless a reader (FIS, ORG, PLC, LND, STO) calls it, and each of those states its own cost.

4.4.3. A test builds a polity of the bound size (1 000 jurisdictions, 1 000 regions) and
a chain of exactly `MaxPolityDepth` entries (a deepest depth of `MaxPolityDepth - 1`), and asserts that the queries agree with the model
of section 14 (FR-POL-004).

## 5. The treasury

### 5.1 What exists

5.1.1. For each **governed** jurisdiction (3.5), at construction and in ascending
`JurisdictionID`, the world creates:

1. the **treasury account**, a ledger account of the new kind **Treasury**
   (`spec-ledger.md` 3.1), whose region is the region of the seat and whose owner
   reference is the `JurisdictionID`;
2. the **treasury pseudo-agent** bound to that account, with an agent ID (5.2).

A jurisdiction that is a grouping creates nothing (FR-POL-019).

5.1.2. A governed jurisdiction has exactly one treasury. The treasury is created
whether or not any other module is on; what it holds and does is the business of the
modules of 5.4. A world in which no module reads it has a set of idle accounts with
the money of `StartMoney` and nothing else.

### 5.2 Where it sits in the orders of construction

5.2.1. **Accounts.** `spec-world.md` 5.2 orders the accounts of a world. The treasury
accounts are a new group between the Outside accounts and the agent accounts: after the
estate accounts of every region and the Outside accounts of the counterparties, and
before the first household agent's account, in ascending `JurisdictionID`. The accounts
of the earlier groups are unchanged.

5.2.2. **Agent IDs.** `spec-world.md` 5.3 gives the estate pseudo-agents the IDs `0` to
`n-1`, the counterparties of the outside the next `m`, and the household agents the
rest. The treasury pseudo-agents take the `g` IDs after the counterparties, in ascending
`JurisdictionID`, and the household agents follow. With the module off, or with no
jurisdiction governed, `g` is 0 and no ID changes.

5.2.3. **Consequence (N-117).** A governed jurisdiction shifts every household's agent
ID by `g`, so every tie-break that reads an ID and every rotation phase changes with it,
exactly as adding a region or a gate does (`spec-world.md` 5.3.2). That is not a defect
and it is the reason a grouping that is not governed creates nothing: the effect on a
run is the price of a treasury and no more. The hazard is POL-A1 (13).

### 5.3 What it is in the ledger and the world

5.3.1. The treasury account is a **non-boundary** account: its balance is never
negative, every change of it is a transfer with a reason of the registry, and the
conservation identities hold with it counted in a further non-boundary sector, the
**treasury sector** (`spec-ledger.md` 9.3 gets one term; `spec-observability.md` 5.3
one field per region). It may hold goods: a lot in a treasury ages, spoils and wears
like any (`spec-ledger.md` 5.3, 5.4), because the account is an ordinary one.

5.3.2. It is never closed and never relocated: the relocation of `spec-regions.md` 6
refuses it with `ErrNotRelocatable`, as it refuses an estate (the kind is not Agent), so
this module registers **no** `region_bound` refusal (the rule of N-090 concerns a
module that keeps an agent's account in a region while the agent is away or committed;
a treasury is neither).

5.3.3. The pseudo-agent is not an agent or a household (`spec-estates.md` 2.7.4 is
restated here for the kind): it has no role, weight, strategy, beliefs, job or upkeep;
it is not counted in the agents or households of any region or role; it causes no
`AgentSpawned` and no `AgentBankrupted` event; it is never passed to a strategy
(FR-AGT-032) and never given relief; it is not in a class of a money flow
(`spec-money-supply.md` 4.3.1) and the ledger's `Relocate` refuses it. It places **no
order of its own**: nothing in this spec gives it a plan (the standing orders of
`spec-fiscal.md` 9 are that module's, with the admission rule of its 9.4), and the call of `spec-integration.md`
that names an agent treats its ID like a pseudo-agent's (`ErrUnknownAgent` for the
calls that need an agent, `ErrNotRelocatable` for `Relocate`, `spec-integration.md`
3.4.3, 4.5.1). A spec that gives it orders adds its admission rule to
`spec-clearing.md` 3.1.2, as EXT did for a counterparty and FIS did for a treasury (`spec-fiscal.md` 9.4).

5.3.4. Flows to and from the treasury are attributed to the seat's region
(`spec-ledger.md` 3.5). A levy paid into it by the households of **another** region is
therefore a transfer between regions that is none of the reconciling items of
`spec-indicators.md` 8.2 (a relocation, an interregional shipment or contract payment).
This spec moves no such money, so it creates no gap; the spec that does, FIS, either
adds the item or keeps tax in the region of the payer (5.4).

### 5.4 What this spec does with it, and what is left

5.4.1. **Only one thing.** At construction, if `StartMoney` is above 0, the money is
moved from the boundary account of the reason `initial` to the treasury account, in the
construction bucket, before the households are created. No other transfer is made by
this spec at any time. No faucet, no sink and no reason is added (FR-POL-018): `initial`
already accepts money into any non-boundary account (FR-LED-038), so the registry row
only widens its text from "agent or estate" to "agent, estate or treasury".

5.4.2. **What the treasury is for**, so that the next specs do not each invent one. It
is the one account a jurisdiction has. A later spec may use it as follows, each under
its own rules and its own reasons:

| Spec | Use | What it must add |
|------|-----|------------------|
| FIS | the account that levies, tolls and tariffs are paid into; the account that pays what a policy releases (relief, a wage, a purchase): `spec-fiscal.md` 3 to 9 | its reasons as internal transfers (a tax is a transfer from a household, not a sink); the reconciling item of 5.3.4 is `FiscalFlowApplied` (`spec-fiscal.md` 4.3); its loops (register) |
| MON | a **grant** whose target is the treasury (a stipend from a boundary), when FIS wants a treasury funded by a faucet | a flow target kind `treasury` (`spec-money-supply.md` 4.3.5; the `Jurisdiction` field of `FlowTarget`) |
| PLC | the identity a policy is attached to: one policy actor per jurisdiction, governed or a grouping (a grouping's actor has no treasury); `spec-policy.md` 3.2 | its interface, its position, its reads (IND measures) and its settings (slots): written in `spec-policy.md` |
| STO | the owner of a store: a store of staple goods held for release, for example a granary | the owner field naming a treasury agent ID: reserved (`spec-storage.md` 3.5.1, `spec-fiscal.md` 9.6); a granary is FIS's buffer stock over treasury goods |
| LND | a holder of a title: the land that a governed jurisdiction holds | `spec-land.md` 3.2.3 lists a jurisdiction as not a holder in version 1, so this stays reserved; it would take the treasury's agent ID |
| EXT | where a duty on an import or an export lands | nothing in EXT: the receipt is FIS's flow |

5.4.3. **What a treasury never does by itself.** It receives no relief, pays no upkeep,
takes no part in the failure ladder, does not spend and does not lend: those would be
rules, and a rule is FIS's, PLC's or ORG's. A treasury with money and no reader is
idle for the life of the run, and that is allowed.

### 5.5 What a consumer sees

5.5.1. The money of a treasury is a field of the snapshot (10.5) and a term of the
sector identity. `World.TreasuryAgent(j JurisdictionID) (AgentID, bool)` returns the
pseudo-agent's ID for a governed jurisdiction (`false` for a grouping or an unknown
`j`); with the account's balance in the snapshot it is all a consumer or a later spec
needs. The name is provisional and the query is INT's to list, class 1
(`spec-integration.md` 6.2: a scalar, `O(1)`, no allocation).

5.5.2. The treasury and its pseudo-agent are never removed. A jurisdiction exists for
the life of the run, so there is no dissolution, no escheat and no succession in this
spec (7.4).

## 6. Hooks: how a rule comes to depend on a jurisdiction

### 6.1 Rows keyed by jurisdiction, and how they combine along a chain

6.1.1. A spec after this one that makes a value depend on a jurisdiction keeps its own
table of rows keyed by a jurisdiction and its own subject (a commodity, a role, an
edge). This spec fixes how such a table is read, once, so a pack author learns it
once, as `spec-regions.md` 5.1 fixed the override of a region. The rules:

- **one row per key** (jurisdiction, subject): a second row is `ErrDuplicateID`;
- **both keys exist**: a row naming an undefined jurisdiction is
  `ErrUnknownJurisdiction`, and one naming an undefined subject has that kind's
  sentinel;
- **absence is the default**, with no warning;
- **canonical order**: rows are sorted by `JurisdictionID`, then by the subject's ID,
  before a hash is taken or a table is walked (architecture.md 7.5);
- **resolved once**: at construction the owner resolves its table into a dense array
  indexed by `RegionID` and the subject, so a lookup in the tick loop is a read and
  allocates nothing. The array is derived and not saved.

6.1.2. **The value of a region** is a function of the rows of the jurisdictions in its
chain and the owner's default. The owner declares one of three modes for each subject:

| Mode | Effective value of region `r` | Typical use |
|------|-------------------------------|-------------|
| **innermost** (replace) | the value of the row of the first jurisdiction in `Chain(r)` that has a row, else the default | a cap, a charter, a rule that the nearest authority sets |
| **along the chain, added** (add) | the default plus the sum of the additions of every row of the chain, outermost first | a rate that every authority above the region levies |
| **along the chain, scaled** (scale) | the default times the product of the factors of every row of the chain (each over 10 000), computed with the checked 128-bit helper and **rounded once** at the end by the owner's stated rule | a factor of a yield or a cost |

An ungoverned region (an empty chain) gets the default in all three. The rounding rule
and the bound are the owner's: this spec states the order and that the combination is
exact before the one rounding, so the result does not depend on the order of the chain.

6.1.3. **At a border.** A reader that applies a rule when a good crosses an edge takes
the lists of `Crossing(from, to)` and applies the rows of the jurisdictions it names,
in list order: of `Exit` (a duty on leaving), of `Enter` (a duty on entering), or of
both. Whether the rows of the list are added or only the first is used is the same
mode choice as above, stated by the owner. This spec fixes the two lists and their
orders and nothing about what is charged.

6.1.4. **No other inheritance.** Nothing else flows down the tree. In particular none
of the region fields of `spec-regions.md` 5.1.3 (reference prices, the basket, extra
upkeep, yields, tastes, shares) is read from a jurisdiction (FR-POL-024), and a
jurisdiction has no price level, no tax revenue and no population as a field. A
setting that wants a regional value to follow the nearest authority writes it as a
region's own row, or as a reader's table keyed by jurisdiction with a mode of 6.1.2.

### 6.2 Rules for a reader

6.2.1. A spec that reads a jurisdiction:

1. reads **structure only**: the chain, the covered regions, the crossing, the seat,
   whether a jurisdiction is governed, a treasury's identity. It never reads a count of
   households, a price level or a money supply of a jurisdiction to decide anything
   that changes how often anyone decides (`spec-world.md` 9.4.1): an aggregate is a
   measure (ADR 0001 decision 7), and a rule that reacts to a measure is a policy
   actor's (PLC);
2. names its mode of 6.1.2 for each subject, and what an ungoverned region and an
   outside node get;
3. adds its rows to the table of 6.3 in the change that adds them, with the position of
   the tick that reads them;
4. branches on no jurisdiction ID (architecture.md 9.1);
5. registers its own loops: a rate that differs across a border is a gap that trade
   will arbitrage, which is the reader's loop and not this spec's (13).

6.2.2. A reader never edits the tree. A jurisdiction's parent and members are
configuration (3.3, 3.4).

### 6.3 The table of hooks

Each row says who reads what. "Offered" means this spec supplies it today; the readers' specs are written (FIS, ORG, PLC,
and the others the rows name), and a row says whether that reader has taken it.

| # | Reader | Reads | Key form and mode | Position of the read | Status |
|---|--------|-------|-------------------|----------------------|--------|
| H1 | FIS: a levy on households or on trade inside a jurisdiction | `Chain(region of the account)` | rows keyed by (jurisdiction, subject), **added** along the chain or **innermost** | FIS's phase | offered; used by `spec-fiscal.md` 5 |
| H2 | FIS: the account a levy lands in, and the account that pays | `Seat`, `TreasuryAgent` | by jurisdiction | FIS's phase | offered; used by `spec-fiscal.md` 4, 5 |
| H3 | FIS: a toll on a route (a TRN shipment) | `Crossing` of each edge of the path | rows keyed by the jurisdiction exited or entered, mode by the owner | dispatch (P3.2, TRN) | offered; used by `spec-fiscal.md` 7 |
| H4 | FIS and EXT: a duty on an import or an export at a gate | `Chain(gate)`, `Crossing(gate, node)` | by jurisdiction of the gate | the outside's phase 3 and 4 positions (EXT) | offered; used by `spec-fiscal.md` 6 |
| H5 | ORG: who may do what where (a charter, an exclusive right) | `Chain(region)` | rows keyed by jurisdiction naming a role, a recipe or a commodity; **innermost** | ORG's refusal, as PRO's `not_on_role` | taken: `Right.Jurisdiction`, mode innermost (`spec-institutions.md` 5.4, 6.1) |
| H6 | ORG: a price control or a usury cap | `Chain(region)` | a cap keyed by jurisdiction, **innermost** | admission (P3, CLR 3.2) or the strategy | taken: `Control.Jurisdiction` and `RateLimit.Jurisdiction`, mode innermost; admission of the market and of the credit round (`spec-institutions.md` 6.3, 6.6) |
| H7 | PLC: the actor of a jurisdiction | `Governed`, `TreasuryAgent`, `Regions(j)` for the measures it reads | one policy actor per jurisdiction, governed or a grouping (a grouping's has no treasury) | PLC's position (P0.3a) | used by `spec-policy.md`: `Regions(j)` for the view, `TreasuryAgent` for the balance |
| H8 | LND: a jurisdiction as the holder of a title | `TreasuryAgent` | by jurisdiction | LND's ration (RES 5.3) | reserved: LND lists it as not offered in version 1 |
| H9 | STO: a store owned by a jurisdiction | `TreasuryAgent` | by jurisdiction | STO's | reserved |
| H10 | RSK: a blockade, a closed border | `Between(g, a, b)` into `SetEdgeStatuses` (the internal form, with a `cause`) | by jurisdiction pair | RSK's | offered through REG 7.3.3; RSK adds no cause and no call in version 1 |
| H11 | IND: a measure over a jurisdiction | `Regions(j)` | by jurisdiction | a snapshot | a preset ask (15) |
| H12 | MON: a flow whose target is a treasury | `TreasuryAgent` | by jurisdiction | P1.7 (money flows) | `FlowTarget.Kind` `treasury` (`spec-money-supply.md` 4.3.5), used by FIS |
| H13 | INT: a consumer's embargo | `Between` then `SetEdgeStatuses` | by jurisdiction pair | between ticks | offered, no new slot; or a lever of kind `edge` with `Against` (`spec-policy.md` 6.3.1) |
| H14 | SET: a size class | not offered (6.4) | | | |

### 6.4 What is not offered, and why

6.4.1. **A law table.** A jurisdiction carries no law, no rate and no flag of its own.
A law is a row of the spec that owns the thing it governs. Otherwise this spec would be
a second configuration surface for every later spec, and one owner per parameter would
fail (architecture.md 9.5, `specs/README.md` "Parameter owners").

6.4.2. **A stored border attribute.** A configured attribute on an edge (a duty class,
a passport, a closure) is a rule. The status that closes or lengthens an edge is REG's
and is already a slot; a border is closed by setting it on `Between`'s edges (H13).

6.4.3. **A law that forbids a move across a border.** `spec-migration.md` decides by
its own reads (4) and a refusal reason is REG's (6.1.3 of that spec) or a module's
`region_bound` registration. A closed edge already makes a destination unreachable for
a journey (`Distance` is undefined). A rule that forbids a move across a border for a
household that could otherwise make it needs a new refusal in `spec-regions.md` 6.1.3
and is not offered; reopen with the first consumer that asks (open question 4).

6.4.4. **A jurisdiction's own money, calendar or market.** A currency boundary, a local
calendar or a market of a jurisdiction would break the one unit of account of
`spec-currency.md` 2 and the one market per region. A duty is not a currency.

6.4.5. **Overlap.** A region in two jurisdictions of different trees, or a body whose
members are jurisdictions or households, is an association (ORG). It reads the chain
as any reader and keeps no membership table of this spec's kind: a body selects by role and
region (`spec-institutions.md` 3.3).

6.4.6. **A count-based rank.** What a settlement's size unlocks is SET's, per region.
A jurisdiction has no class, because a class is a function of a count of households and
SET already keeps that per region (`spec-settlements.md` 7); a reader that wants "the
largest settlement of a jurisdiction" reads SET's class of the regions in
`Regions(j)`.

## 7. In the tick, between ticks, and after a restore

### 7.1 No position, no draw, no event

7.1.1. The module acts **nowhere in the tick**. It has no position among the phases of
`spec-world.md` 6, reads nothing that moves, and the readers of 6.3 read it from their
own positions. It takes **no draw** (`spec-world.md` 7.3): the table of draws gains no
row, and the draw count of a step is the same with the module on or off
(FR-POL-020). It emits **no event of its own** at any time (10.1; one field of `WorldCreated`, 10.7), because the tree, the
membership and the treasury accounts do not change.

7.1.2. **Construction** is the one time it acts, in the steps of `spec-world.md` 5.1:
step 5 derives the `Polity` tables with the other derived tables; step 7 creates the
treasury accounts in the group of 5.2.1; step 8 creates the treasury pseudo-agents and
makes the `StartMoney` transfer (5.4.1) before the households are created and before
the warm start. A configuration that passes `Validate` cannot fail in these steps
(`spec-world.md` 5.1.1).

### 7.2 Between ticks

7.2.1. There is **no slot** and no call of this module (FR-POL-021). The tree and the
membership are configuration: part of the hash, fixed for a run, never edited by a
call (`spec-world.md` 8.2.1). A consumer reads them with `World.Polity()` and the
snapshot, on a healthy or a failed world (4.2.4).

7.2.2. An **embargo** or a blockade between two jurisdictions is `Between(g, a, b)` fed
to `SetEdgeStatuses` (`spec-integration.md` 5.1): both are existing mechanisms, and
this spec adds no slot for them (H13).

### 7.3 After a restore

7.3.1. A restore builds the `Polity` from the configuration, rebuilds the treasury
pseudo-agents from the configuration and the account list, and finds the treasury
accounts in the ledger section (11). The configuration hash covers the tree and the
membership, so a checkpoint cannot be restored into a different polity
(`spec-checkpoint.md` 6.2): the world is the same one.

### 7.4 What is reserved: a transfer of a region

7.4.1. A **transfer** moves a region from one jurisdiction to another (for example a conquest, a
secession, a sale) in the middle of a run. It is not offered in version 1 (position 9).
Its shape is fixed here so that no reader assumes a static tree by accident, and so
that adding it is a slot and not a redesign:

- the mechanism is this spec's and the public call is INT's: a plural setter of
  `(region, jurisdiction or none)` pairs, atomic and between ticks only, with the cause
  fixed by the setter (`spec-integration.md` 5.1.2);
- it acts **at the call**, like an edge status: the derived tables (chains, covered
  lists) are rebuilt for the whole batch once, so that no reader sees a half-built
  polity (`spec-world.md` 8.2.1);
- the state is the list of regions whose membership differs from the configured one,
  sparse and in ascending `RegionID`, saved in a new section of the checkpoint, not
  hashed (`spec-world.md` 8.1.1 item 5);
- the event is `RegionTransferred` (region, from, to), one per changed region in
  ascending `RegionID`, after the tables are in place;
- a region that is the seat of a governed jurisdiction cannot leave it (the seat stays
  covered, `ErrSeatNotCovered`), because a treasury is not relocatable (5.3.2);
- every reader that cached by jurisdiction reads its resolved array again at the call.

7.4.2. Reasons it is left open: it is a feedback target for a conflict spec (RSK) that
uses none in version 1 (`spec-risk.md` 10.3: RSK adds no cause and no call, and does not use
a transfer of a region); every reader of 6.3 would need to say what a transfer does to its
state (a tax carry, a charter), which no written reader (FIS, ORG, PLC) states for itself;
and the first consumer's five regions have no stated need (N-030). It is the open
question 1.

## 8. Parameters

All are fields of `Config` with a bound, a doc line and an owner (architecture.md 9.5
to 9.7), under the module entry `Modules.Polity` (a pointer; nil is off,
`spec-config.md` 3.2). The module has **no tunable with a default**: a tree and a
membership are structure, and the default of the section as a whole is "no
jurisdictions", which is the world without the module. The bounds are in `bounds.go`
(architecture.md 9.6).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Polity` | the module; present means on | nil (off) | |
| Polity | `Jurisdictions` | the jurisdictions | none | at most `MaxJurisdictions` |
| Polity | `Members` | the membership rows, keyed by region | none | one per region, at most `MaxRegions` |
| Jurisdiction | `ID` | the identity (3.2.2) | required | `[a-z][a-z0-9_]*`, at most 64 bytes, unique |
| Jurisdiction | `Parent` | the parent jurisdiction, or empty for a root | empty | a defined jurisdiction other than itself, no cycle, fewer than `MaxPolityDepth` ancestors |
| Jurisdiction | `Seat` | the region it is governed from, or empty for a grouping | empty | a region it covers that is not an outside node |
| Jurisdiction | `StartMoney` | the money its treasury is created with, in minor units | 0 | 0 to `MaxMoney`; only with a seat |
| Member | `Region` | the region that is a member | required | a defined region |
| Member | `Jurisdiction` | the jurisdiction it is a direct member of | required | a defined jurisdiction |

Presentation-only, so exempt from the registry test and not hashed
(`spec-config.md` 8.3): `Jurisdiction.Label`, empty by default and at most
`MaxLabelBytes` bytes (the constant of `spec-regions.md` 4.2).

**Bounds** (placeholders, N-015): `MaxJurisdictions` 1 000 (the value of
`MaxRegions`: a jurisdiction that covers no region draws a warning, so more is pointless)
and `MaxPolityDepth` 8 (the bound is on **levels**: a chain of at most 8 entries, a depth of at most 7; eight levels of authority is more than the examples of section 12 use:
three, one and one). `MaxMoney` is the world's (`spec-config.md`
4.1).

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the order of
the chain (innermost first), of `Exit` (innermost first) and `Enter` (outermost first)
(4.1); the order and the single rounding of the combination along a chain (6.1.2); the
placement of the treasury accounts and agent IDs (5.2); the name `Treasury` of the
account kind.

**Not parameters:** the `JurisdictionID`, the depth, the chain tables and the covered
lists (derived), the treasury accounts and agent IDs (derived from the configuration),
and every reader's rows (theirs).

The placeholders, so that none is mistaken for a result: `MaxJurisdictions`,
`MaxPolityDepth`, the operation counts of 4.4.2 and the numbers of the worked examples
of section 12.

## 9. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). Errors name the path,
for example `modules.polity.jurisdictions[2].parent` or `modules.polity.members[0].region`.
The framework of `spec-config.md` 7 runs them, and `NewPolity` runs these alone.

### 9.1 Errors

The rules run in this order, so that the first error is the same for the same
mistake: syntax and duplicates; counts; parents (unknown, then cycle, then depth);
members (unknown region, unknown jurisdiction, duplicate region); seats (unknown
region, not covered, outside node); `StartMoney` (without seat, then range).

| Sentinel | Rejects |
|----------|---------|
| `ErrInvalidID`, `ErrDuplicateID` | a jurisdiction ID that does not match `spec-config.md` 3.3, or twice; two membership rows for one region |
| `ErrTooMany` | more than `MaxJurisdictions` jurisdictions |
| `ErrUnknownJurisdiction` (new) | a `Parent` or a member row naming an undefined jurisdiction |
| `ErrUnknownRegion` | a member row or a `Seat` naming an undefined region |
| `ErrJurisdictionCycle` (new) | a parent chain that returns to the jurisdiction, a self-parent included |
| `ErrOutOfBounds` | a jurisdiction with `MaxPolityDepth` or more ancestors (more than `MaxPolityDepth` levels); a `StartMoney` outside `[0, MaxMoney]`; a label above `MaxLabelBytes` |
| `ErrSeatNotCovered` (new) | a `Seat` that is not a direct member of the jurisdiction or of one of its descendants |
| `ErrSeatOutsideNode` (new) | a `Seat` that is an outside node of the external module (arises only with that module on) |
| `ErrStartWithoutSeat` (new) | a `StartMoney` above 0 on a jurisdiction with no seat |

The sentinels are root sentinels matched with `errors.Is`. Nothing at run time returns
an error from this module: there is no call (7.2.1). Note N-075 records the mapping
for the root.

### 9.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at
construction, at `Warn`, with the attributes `rule` and `path`, through the
"configuration warning" line of `spec-observability.md` 7.3 (row 1); the volume is
bounded by the configuration.

| Warning | Condition |
|---------|-----------|
| `jurisdiction_empty` | a jurisdiction that covers no region: a row keyed by it could never apply (3.3.2) |
| `region_unassigned` | the module is on, at least one region has a member row, and a region that is not an outside node has none (3.4.4): one warning per such region |

## 10. Observability

10.1. **Events: none of its own.** One field joins an existing event (10.7). The module has no transition: the tree, the membership and the
treasury accounts are fixed for a run, and a treasury's money moves only by the
transfers of later specs, which report them (`BoundaryFlow` for a faucet or a sink, the
owner's events for the rest). The treasury pseudo-agent causes no `AgentSpawned`
(5.3.3). A restore emits `WorldRestored` as always. An adapter that groups by
jurisdiction does so from the names table and the configuration it holds (10.3), not
from an event.

10.2. **Logs.** Constant lowercase messages, typed attributes, `Enabled` guards.

- `Info`, once, in the existing "world created" line: two attributes join the summary,
  `jurisdictions` and `governed` (counts), and `polity` joins the `modules` list. No
  other `Info`.
- `Warn`, at construction, one per warning of 9.2, with `rule` and `path`.
- No `Debug`, no `Error`: nothing in this module runs per tick.

10.3. **The names table** (`spec-observability.md` 3.2.4) gains a slice,
`Jurisdictions []string`, indexed by `JurisdictionID`, in the same order as every other
kind. An event or a row that names a jurisdiction (the later specs') carries the typed
`JurisdictionID` and the string is read from the table. The cardinality test of
`spec-observability.md` 9.4 gets the range check `0 <= id < len(Names.Jurisdictions)`.

10.4. **New log keys** for observability-and-logging.md 3.3.2: `jurisdictions` and
`governed` (counts, in the construction summary). The key `region` and the existing
keys are reused. The strings are in `internal/logkeys`.

10.5. **Snapshot.** Two fields, names provisional (OBS's): per region,
`RegionRow.MoneyTreasury`, the money of the treasuries seated there (0 with none), the
further term of the sector identity of `spec-ledger.md` 9.3; and
`Snapshot.Jurisdictions`, one `JurisdictionRow{Jurisdiction, Seat, Agent, Money}` for each
**governed** jurisdiction in ascending `JurisdictionID`: the jurisdiction, its seat, the
treasury pseudo-agent's `AgentID` and the treasury's money. The agent rows carry no
treasury: a pseudo-agent is in no count and has no per-agent row, as an estate has none
(`spec-observability.md` 5.3.1, FR-OBS-007); the `Agent` field is what identifies it.
Bounded by the governed jurisdictions. `RegionRow.MoneyTreasury` and the rows are also
the terms that `spec-indicators.md` 8.2 and 9 need beside households, pool and contract
(an `M_t` term of the regional identity: it is constant while no later spec moves
treasury money, and 15 asks for it).

10.6. **Metric.** No new instrument. `economysim.money.supply` takes the value
`treasury` for its `economysim.sector` attribute (`spec-observability.md` 6; the
cardinality list of observability-and-logging.md 5.4.1 gains one constant), fed by
`RegionRow.MoneyTreasury` like the other sectors; absent, not zero, until a snapshot is
passed.

10.7. **The start money in `WorldCreated`.** The `StartMoney` of a treasury is in the
`initial` row of `BoundaryFlow` at `Tick` -1 of the seat's region, and
`spec-observability.md` 4.3.3 item 1 counts that row as household or pool money. An
observer that keeps no configuration (`spec-observability.md` 3.2.4) would then rebuild
a pool too high by the treasury's money, and FR-OBS-041 would fail with a governed
jurisdiction. So `WorldCreated` gains `Jurisdictions []JurisdictionRow` (the shape of
10.5, `Money` the `StartMoney`, empty with no governed jurisdiction), and 4.3.3 item 4
subtracts each row's money from the `initial` money of the row's seat region, as it
subtracts the `StartMoney` of the initial `AgentSpawned`; `WorldRestored` carries the
snapshot, which has the rows with the money at the restore (FR-POL-035).

10.8. **No observer, no logger.** The queries allocate nothing but their lists
(4.2.2); no event is constructed. Observation never changes a result: no log, event or
metric draws a random number or reads the wall-clock (observability-and-logging.md 1.4).

## 11. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's:

- **none of its own.** No section of its own is added to the frame, for any
  configuration: the tree and the membership are in the configuration, which the header's
  hash covers (`spec-checkpoint.md` 5.4), and nothing about them changes in a run.
- The **treasury accounts** are in the ledger section (tag 2, `spec-checkpoint.md` 4.2)
  like every account that is not a boundary: kind, closed flag, region, owner reference,
  money and lots. The kind enumeration of 4.2.4 there gains `5` Treasury (the order of
  `spec-ledger.md` 3.1), and the owner reference of a Treasury account is the
  `JurisdictionID`. The pseudo-agent is rebuilt from the configuration and the account
  list (`spec-checkpoint.md` 4.2.2). A restore checks that the account list holds one
  Treasury account for each governed jurisdiction, in ascending `JurisdictionID`, with
  the seat's region, and rejects a checkpoint that does not as corrupt.

**Not saved** (derived, and a row owed to `spec-checkpoint.md` 4.3 for FR-CHK-045): the
`Polity` tables (parent, depth, chains, children, covered lists), the `JurisdictionID`,
the resolved arrays of the readers, the treasury pseudo-agent IDs and their account
index. A world restored between
ticks answers every query as an uninterrupted one does (FR-POL-032). The configuration
hash covers `Modules.Polity` in canonical order, every field except the labels
(`spec-config.md` 8), so a change of the tree, of a membership, of a seat or of
`StartMoney` changes the hash.

## 12. Worked examples

Three settings, flagged as **examples**. Names and numbers are placeholders; the
tables are the output of `models/pol/worked.py` (section 14). Region and jurisdiction
order is by ID string, bytewise, so the first name in a table's row order has ID 0.

### 12.1 Example A: a kingdom of five regions (an early-modern-like setting)

Regions `capital` (0), `farm` (1), `harbour` (2), `hills` (3), `mill` (4). Jurisdictions
`coast` (0), `crown` (1), `inland` (2), `uplands` (3). The crown and the inland province
are governed; the coast and the uplands are groupings.

```toml
[modules.polity.jurisdiction.crown]
seat = "capital"
start_money = 50000

[modules.polity.jurisdiction.coast]
parent = "crown"

[modules.polity.jurisdiction.inland]
parent = "crown"
seat = "mill"
start_money = 5000

[modules.polity.jurisdiction.uplands]
parent = "inland"

[modules.polity.member]
capital = "crown"
farm    = "inland"
harbour = "coast"
hills   = "uplands"
mill    = "inland"
```

**Chains** (innermost first): `capital`: [crown]; `farm`: [inland, crown]; `harbour`:
[coast, crown]; `hills`: [uplands, inland, crown]; `mill`: [inland, crown].

**Crossings** (the common jurisdiction, and its depth as the border depth of 4.3.1):

| from > to | `Exit` | `Enter` | common | border depth |
|-----------|--------|---------|--------|-------------:|
| `hills` > `harbour` | uplands, inland | coast | crown | 0 |
| `harbour` > `hills` | coast | inland, uplands | crown | 0 |
| `farm` > `mill` | (none) | (none) | inland | 1 |
| `farm` > `capital` | inland | (none) | crown | 0 |
| `hills` > `farm` | uplands | (none) | inland | 1 |
| `capital` > `mill` | (none) | inland | crown | 0 |

The first two rows are the reversal of one another (4.1.2): what a good leaves going one way it
enters going the other, in the opposite order. `farm > mill` is not
a border: both are in the same province.

**Edges** (REG's canonical order by (`From`, `To`) as `RegionID`s): 0 `capital>harbour`,
1 `capital>mill`, 2 `farm>mill`, 3 `harbour>capital`, 4 `harbour>mill`, 5 `hills>mill`,
6 `mill>capital`, 7 `mill>farm`, 8 `mill>harbour`, 9 `mill>hills`.
`Between(g, coast, inland)` is [4] (`harbour>mill`) and `Between(g, inland, coast)` is
[8]; `Between(g, coast, crown)` is empty because the two are nested. An embargo of the
coast against the interior is `SetEdgeStatuses` closing edges 4 and 8 (H13), and it
changes no jurisdiction.

**IDs.** With five regions and no counterparty the estates are agents 0 to 4. The
treasuries are agent 5 (`crown`, `JurisdictionID` 1) and agent 6 (`inland`, 2), and the
households start at 7, two later than with the module off (5.2.3). The treasury
account of `crown` is in region `capital` with 50 000 minor units, that of `inland` in
`mill` with 5 000, both created in the construction bucket by `initial`.

**Combining modes** (6.1.2), for three hypothetical readers with their own rows, in
basis points or minor units (all placeholders; FIS and ORG key their own tables in the same way):

| Reader and rows | `capital` | `farm` | `harbour` | `hills` | `mill` |
|-----------------|----------:|-------:|----------:|--------:|-------:|
| a levy, **added**: crown 200, inland 100, uplands 50 | 200 | 300 | 200 | 350 | 300 |
| a price cap, **innermost**: crown 90, inland 80 | 90 | 80 | 90 | 80 | 80 |
| a factor, **scaled** over a base of 1 000: crown 9 000, inland 8 000 | 900 | 720 | 900 | 720 | 720 |

`hills` is in the uplands, which has no row of the cap, so the cap is the inland's, 80;
`harbour` is in the coast, which has none either, so it is the crown's, 90. The scaled
value of `farm` is `1 000 * 9 000 * 8 000 / 10 000^2 = 720` with one rounding at the
end (a half up, the owner's rule), the same in whatever order the chain is read.

**Warnings:** none (every region has a row; every jurisdiction covers a region).

### 12.2 Example B: two crowns, a free city and an outside node (a trading-league setting)

Regions `ashford` (0), `brook` (1), `city` (2), `delta` (3), `far` (4, an outside node
whose gate is `delta`). Jurisdictions `east` (0) and `west` (1), both roots and both
governed (seats `ashford` and `brook`). `ashford` is in `east`, `brook` and `delta` in
`west`; `city` and `far` have no row.

| from > to | `Exit` | `Enter` | Reading |
|-----------|--------|---------|---------|
| `delta` > `far` | west | (none) | an export duty keys on `west`: the node is in no jurisdiction (4.3.5) |
| `far` > `delta` | (none) | west | an import duty keys on `west` |
| `city` > `ashford` | (none) | east | the free city's goods enter `east` |
| `ashford` > `brook` | east | west | two roots: they share nothing, border depth -1 |
| `brook` > `delta` | (none) | (none) | one jurisdiction: not a border |
| `city` > `far` | (none) | (none) | two ungoverned regions |

With the external module on, one node and one gate with both sides give two
counterparties, so the IDs are: estates 0 to 4, counterparties 5 and 6, treasuries 7
(`east`) and 8 (`west`), households from 9. Warnings: `region_unassigned` for `city`
(`far` is an outside node and draws none).

### 12.3 Example C: a station authority (a space setting; a household is a crew)

Regions `deck` and `dock`; one jurisdiction `authority`, a root with no seat, and a row
for each region. There is no border (4.3.3), no treasury, no account and no pseudo-agent:
the module is a label that a reader may key a charter or a measure to, and the world is
the same as the world without it but for the list of FR-POL-019 (the configuration
hash, the module list, the names table, the construction summary and the warnings). A setting that wants the authority to hold money adds `seat = "dock"` and pays the
price of 5.2.3.

### 12.4 Use by the reference chain (a proposal for REF)

12.4.1. `overlay/polity.toml` for `five` (`spec-reference-chain.md` 4.4.4): a crown over
two provinces, one of three regions and one of two, ungoverned, with no money, so that
every pack with it is exactly neutral and the scenario POL-S1 can assert it. A second
overlay, `overlay/polity-governed.toml`, adds a seat and `start_money` to the crown, for
POL-S2. REF builds both; this spec only names them.

## 13. Entries for the stability register

13.1. **The module adds no feedback loop.** By the checklist of `spec-stability.md` 9.1:
it reads no price, inventory, income, count or share, and changes nothing a rule
reads after construction. No rule of this spec reads a jurisdiction; the readers of 6.3 are the specs it names
(FIS, ORG, PLC and the reserved ones). The loops that a rate or a cap keyed by jurisdiction
creates (a gap at a border that a merchant arbitrages, a cap that causes a shortage,
a levy that moves a role) are the loops of FIS, ORG and PLC, and each registers its own
(6.2.1 item 5).

13.2. **Hazards** (kind `hazard`, a process or configuration trap and not a loop; provisional IDs, the supervisor numbers
them, N-103):

| ID | Hazard | Owner | Damper or mitigation | Condition | Test | Status |
|----|--------|-------|----------------------|-----------|------|--------|
| POL-A1 | **A governed jurisdiction shifts household IDs and account IDs.** Every tie-break key, rotation phase and cadence offset that reads an ID changes, so a run with a treasury is not a bit-identical neighbour of the run without it, although no rule changed (5.2.3). A paired comparison "treasury on, treasury off" is therefore not aligned from the first clearing (N-124); a comparison over seeds is valid | POL 5.2 | an ungoverned grouping creates nothing and is exactly neutral (FR-POL-019); a consumer that wants a polity for measurement only writes no seat | a comparison over seeds, never a paired one (N-124), or a pack with no seat | POL-S1, POL-S2 | `hand`: the shift is an integer offset (`worked.py`); no run of the library exists, and the effect size is not measured |
| POL-A2 | **A treasury is attributed to one region.** A levy that the households of another region pay into it is a transfer between regions, which no reconciling item of IND 8.2 names; a regional money report would show the payer's region losing and the seat's region gaining money that no flow explains (5.3.4) | POL 5.3.4; FIS, IND | Resolved by design: the receipt is in the seat's region and `FiscalFlowApplied` (`spec-fiscal.md` 4.3) is the reconciling item beside the treasury sector `M_t` of IND 8.2 (open question 5) | the module reports the fiscal flows beside the treasury sector, or keeps the receipt in the payer's region | POL-S2 (no levy of POL's own; the identity with the `M_t` term) | `untested` |

13.3. **Scenarios proposed** (provisional IDs; both are harness tests of the library's
own procedure, in the class of S77 and S78, outside the band rule of FR-STY-098):

| ID | Scenario | Pass |
|----|----------|------|
| POL-S1 | `five` with `overlay/polity.toml` (ungoverned) against `five`, over seeds 1 to 10, 200 ticks: events, snapshot and golden rendering compared | every event, every snapshot field and every golden line identical, except the names table, the configuration hash, `Snapshot.Modules`, the construction line and the polity's warnings (exact) |
| POL-S2 | `five` with `overlay/polity-governed.toml` against POL-S1's run, seeds 1 to 10: the treasury accounts exist, hold `start_money`, never change, and the audit passes | the treasury balance equals `start_money` at every boundary; `WithAudit` passes; the household IDs are shifted by `g` and the mean measures of the two runs are reported, not asserted (POL-A1) |

## 14. What was run

`models/pol/` (Docker, standard library only, `PYTHONDONTWRITEBYTECODE=1`, a read-only
mount; the commands are in its README):

- `chain.py`: the definitions of 4.1.1 by chains and sets against a second formulation
  that climbs to the lowest common ancestor, on **2 000 random forests** (1 to 30
  jurisdictions, 1 to 40 regions, at most 8 levels, a random share of regions with none),
  1 140 712 ordered pairs of regions, 963 342 of them a border. Every pair agreed;
  the reversal symmetry (`Crossing(b, a)` is the reversal of `Crossing(a, b)` with the
  two lists swapped), the disjointness of `Exit` and `Enter`, the empty crossing of a
  region with itself, the common suffix of two chains, the covered lists by descent
  against by chain, and `Between` against its definition on random edge lists all held,
  and a chain at the bound (8 levels, a deepest depth of 7) has 8 entries.
- `worked.py`: the arithmetic of section 12.

**What they do not show.** They check that the definitions are what 4.1 says and have
the properties claimed. They measure no cost (4.4.2 counts operations), say nothing
about any economy and are not evidence that any rule keyed by a jurisdiction is stable:
no such rule is written.

## 15. Notes settled

| Note | What this spec does |
|------|---------------------|
| N-016 (the parameter registry) | **Applied.** The module adds the rows of section 8 and no unnamed parameter; `StartMoney` is the only value and has a bound and an owner. The registry test is still to be written |
| N-017 (words and names) | **Applied.** The terms of the handoff are fixed (jurisdiction, polity, member, chain, covers, common, crossing, exit, enter, border, border depth, seat, governed, grouping, treasury, treasury sector, combining mode). `realm` stays informal in the other specs and has no type here (header) |
| N-027 (no default is stable) | **Applied.** The spec sets no default macro constant: `StartMoney` is 0, the bounds are structural placeholders and the model checks definitions |
| N-090 (the rule for a module that keeps an account in a region) | **Applied, answered by no registration.** A treasury is neither away nor committed, and its kind is not relocatable (5.3.2) |
| N-091 (the slots) | **Applied: none.** The module has no slot; the transfer of a region is reserved with its shape (7.4) |
| N-103 (running writers in parallel) | **Followed, with one slip.** Only the spec, `models/pol/` and `.supervisor/pol.*` were created; every model ran in Docker; the host `python3` was invoked once as a no-op (`python3 -c 1`) that read and wrote nothing |
| N-104 (region-keyed tables use one mechanism) | **Applied.** The membership is a table keyed by region in replace mode, with a row for the list of `spec-regions.md` 5.1.3 (3.4.1) |
| N-107 (new faucets and sinks start from MON) | **Applied: none added.** `StartMoney` uses `initial`; the reserved MON target name for a treasury is `treasury` (5.4.2, H12) |
| N-108 (what RES leaves to LND, and others) | **Left open for LND.** A jurisdiction may be a holder through its treasury's agent ID; LND lists it as not offered in version 1 (H8) |
| N-117 (the agent ID space with an outside) | **Applied.** The treasuries follow the counterparties and precede the households (5.2.2); POL-A1 |
| N-126 (a new event or `Warn` adds a row) | **Applied: none.** No event and no new `Warn` message; the two warnings are rules of the "configuration warning" line (row 1 of `spec-observability.md` 7.3), with a row owed to `spec-config.md` 7.4.2 |
| N-128 (a hosted per-agent field is saved once) | **Not applicable.** The module hosts no per-agent field |
| N-131 (not offered in version 1) | **Extended** with the transfer of a region and a law that forbids a move across a border (6.4.3, 7.4) |
| N-122, N-045, N-124 | read: no cap, no silent damper; N-124 is applied to POL-A1 |

**New notes** (areas in brackets; to add in the integration): a treasury is a
region-attributed account, so a levy across regions needs a reconciling item (FIS, IND,
LED); readers keyed by jurisdiction use the combining modes of 6.1.2 (FIS, ORG, PLC, LND,
STO); a transfer of a region is reserved with its shape (INT, RSK); a polity with a
treasury is not a paired comparison with one without (EXP, CLB); "treasury" is no longer
the region's pool (MON, FIS).

**Asks of other specs** (not adopted by this change; they are in the handoff): IND, the `M_t` term of the regional identity of 8.2 and a `Sectors` row (10.5), and an
aggregator whose input is a set of regions, so that a measure by jurisdiction is a fold
over `Regions(j)` and no new measure is needed (H11); OBS, the names slice, the snapshot
fields, the sector value and the `WorldCreated.Jurisdictions` field with the rule of
4.3.3 item 4 (10.7); LED and WLD, the account kind and the two orders of 5.2;
REG, one row in 5.1.3 and one sentence in 5.4.1; CHK, the row of 4.3 (11); LND, the
keyed-by-jurisdiction wording of its 1 and 7.7 (6.4.1).

## 16. Requirements

IDs are stable (testing.md 4.1). A requirement is a single testable statement.

### Functional

| ID | Requirement |
|----|-------------|
| FR-POL-001 | With `Modules.Polity` nil, or with no member row, every query MUST answer "none" (an empty chain, `NoJurisdiction`, empty lists) and no result, event or snapshot field of any world MAY differ from a world without the module except the names table, the configuration hash, `Snapshot.Modules`, the construction summary and the polity's configuration warnings (3.4.4, 3.6, FR-POL-019). |
| FR-POL-002 | Jurisdictions MUST be sorted by ID bytewise before a world is built or a hash is taken, and the `JurisdictionID` MUST be the index in that order; every list of jurisdictions MUST be ascending by it and every list of regions ascending by `RegionID` (3.2). |
| FR-POL-003 | A jurisdiction MUST have at most one parent; a `Parent` naming an undefined jurisdiction MUST be `ErrUnknownJurisdiction` and a parent chain that returns to itself, a self-parent included, `ErrJurisdictionCycle` (3.3.1, 9.1). |
| FR-POL-004 | The number of jurisdictions MUST be at most `MaxJurisdictions` (`ErrTooMany`) and a jurisdiction MUST have fewer than `MaxPolityDepth` ancestors, so a chain has at most `MaxPolityDepth` entries (`ErrOutOfBounds` otherwise); a chain of exactly `MaxPolityDepth` entries and a polity of the bound size MUST build and agree with the model, and one of `MaxPolityDepth + 1` MUST be rejected (3.3.1, 4.4.3). |
| FR-POL-005 | The membership MUST have at most one row per region, canonical by `RegionID`; a row naming an undefined region MUST be `ErrUnknownRegion`, one naming an undefined jurisdiction `ErrUnknownJurisdiction`, and two rows for one region `ErrDuplicateID` (3.4.1). |
| FR-POL-006 | `Chain(r)` MUST be the direct jurisdiction of `r` followed by its ancestors, innermost first, with no repeated element, and empty for a region with no row (4.1.1). |
| FR-POL-007 | `Covers(j, r)` MUST be true exactly when `j` is in `Chain(r)` (4.1.1). |
| FR-POL-008 | `Common(a, b)` MUST be the first element of `Chain(a)` that is in `Chain(b)`, or `NoJurisdiction` (4.1.1). |
| FR-POL-009 | `Regions(j)` MUST list the regions covered by `j` in ascending `RegionID`, the direct members of its descendants included (3.1.2). |
| FR-POL-010 | `Crossing(a, b)` MUST give `Exit`, the elements of `Chain(a)` not in `Chain(b)` innermost first, and `Enter`, the elements of `Chain(b)` not in `Chain(a)` outermost first; both MUST be empty for `a == b`, and the result MUST be defined for any two regions whether or not an edge joins them (4.1.1, 4.3.1). |
| FR-POL-011 | `Between(g, a, b)` MUST list in ascending `EdgeID` the edges of `g`, closed ones included, whose crossing has `a` in `Exit` and `b` in `Enter`; for `a` and `b` nested or equal it MUST be empty (4.2.3). |
| FR-POL-012 | A query given an ID outside its range MUST return the zero result and `false`, `NoJurisdiction` or an empty list, and MUST NOT panic (4.2.2). |
| FR-POL-013 | A `Polity` MUST hold plain data with no mutating method; the lists it returns MUST be fresh; `World.Polity()` MUST return an equal value on every call for the life of the world, as a deep copy that shares no storage with it (architecture.md 12.4), and MUST be readable on a failed world (4.2). |
| FR-POL-014 | A jurisdiction is governed exactly when its `Seat` is non-empty; the seat MUST be a region the jurisdiction covers (`ErrSeatNotCovered`) and not an outside node (`ErrSeatOutsideNode`), and a `StartMoney` above 0 without a seat MUST be `ErrStartWithoutSeat` (3.5, 9.1). |
| FR-POL-015 | The world MUST create, for each governed jurisdiction in ascending `JurisdictionID`, one account of kind Treasury in the seat's region with the `JurisdictionID` as owner reference, after the Outside accounts and before the first agent account; it MUST never close or relocate it (5.1, 5.2.1, 5.3.2). |
| FR-POL-016 | The treasury pseudo-agents MUST take the agent IDs after the counterparties and before the households, in ascending `JurisdictionID`; a treasury MUST NOT be counted in any agents or households, MUST NOT cause `AgentSpawned` or `AgentBankrupted`, MUST NOT be passed to a strategy and MUST NOT place an order of its own (a standing order is `spec-fiscal.md` 9's) (5.2.2, 5.3.3). |
| FR-POL-017 | A `StartMoney` above 0 MUST be moved from the boundary account of the reason `initial` to the treasury account in the construction bucket before the households are created; 0 MUST make no transfer (5.4.1). |
| FR-POL-018 | The module MUST move no money or goods after construction, and MUST add no reason, faucet or sink to the registry (5.4.1). |
| FR-POL-019 | With the module on and no jurisdiction governed, the world MUST create no account and no pseudo-agent, MUST NOT shift any ID and MUST produce every result, event field and snapshot field of the world without the module, except: the names table, the configuration hash (so the hash carried by `WorldCreated` and `WorldRestored` and by the checkpoint header), `Snapshot.Modules`, the construction summary and the polity's configuration warnings (`jurisdiction_empty`, `region_unassigned`) (5.1.1, 5.2.2). |
| FR-POL-020 | The module MUST take no draw from the world's source, and the number of draws of a step and of construction MUST be the same with it on or off (7.1.1). |
| FR-POL-021 | The module MUST have no slot and no call between ticks; the tree and the membership MUST NOT change in a run (7.2.1). |
| FR-POL-022 | The configuration hash MUST cover the jurisdictions and the members in canonical order, every field except the labels (11). |
| FR-POL-023 | A table keyed by jurisdiction MUST combine along a chain by one of the three modes of 6.1.2: innermost (the first row of the chain), added (the sum over the chain) or scaled (the product over the chain, computed exactly and rounded once); an ungoverned region MUST get the default (6.1.2). |
| FR-POL-024 | No field of a region (`spec-regions.md` 5.1.3) MUST be read from a jurisdiction (6.1.4). |
| FR-POL-025 | The construction summary MUST carry `jurisdictions` and `governed` and the module name `polity` in `modules` when the module is on (10.2). |
| FR-POL-026 | The warning `jurisdiction_empty` MUST be raised for a jurisdiction that covers no region and `region_unassigned` for each region that is not an outside node and has no row when at least one region has one; both at construction, never failing validation (9.2). |
| FR-POL-027 | The names table MUST carry `Jurisdictions` indexed by `JurisdictionID` (10.3). |
| FR-POL-028 | The snapshot MUST carry `MoneyTreasury` per region and one row per governed jurisdiction in ascending `JurisdictionID` with its seat, the treasury pseudo-agent's `AgentID` and the treasury's money; the agent rows MUST carry no treasury (10.5). |
| FR-POL-029 | A treasury account MUST be a non-boundary account that cannot go negative; the sector identity MUST hold with the treasury sector, and the audit MUST pass (5.3.1). |
| FR-POL-030 | The module MUST add no section to a checkpoint; the treasury accounts MUST be in the ledger section with the kind `5`, and a restore MUST reject as corrupt a checkpoint whose account list lacks, duplicates or misplaces one (11). |
| FR-POL-031 | `relocate` of a treasury MUST be refused with `ErrNotRelocatable`; a treasury MUST NOT be in a class of a money flow nor receive relief (5.3.2, 5.3.3). |
| FR-POL-032 | A world restored between ticks MUST answer every query of section 4 as an uninterrupted one (11). |
| FR-POL-033 | The crossing, the chain and the border depth MUST NOT depend on an edge's status (4.3.2). |
| FR-POL-034 | Goods in a treasury MUST age, spoil and wear as in any account (5.3.1). |
| FR-POL-035 | `WorldCreated` MUST carry `Jurisdictions`, one row for each governed jurisdiction with its `StartMoney`, and an observer's rebuild of `MoneyHouseholds` and `MoneyPool` MUST equal the snapshot with a governed jurisdiction whose `StartMoney` is above 0 (10.7). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-POL-001 | With the default logger and observer, `Name`, `Lookup`, `Len`, `Parent`, `Depth`, `Of`, `Covers`, `Common` and `Seat` MUST allocate nothing, and construction MUST emit no event of its own (4.2.2, 10.8). |
| NFR-POL-002 | Construction MUST be `O(n * depth + J log J)` (the sort of 3.2.1 is the log) and a query `O(depth)` or a copy of at most `n` entries; `Between` `O(edges * depth)` (4.4). |
| NFR-POL-003 | Observation MUST NOT change a result: the same seed and configuration give identical results with and without a logger and observer (10.8). |
| NFR-POL-004 | Every output MUST be bit-identical across runs and platforms: no iteration over a map and no sort without a total order influences a result (3.2.3). |

## 17. Open questions

1. **A transfer of a region** (7.4). Reopen when a consumer or RSK needs a region to
   change hands. The shape is fixed there. The part that needs a decision first is what
   each reader does with its state (a tax carry, a charter) when its key changes.
2. **More than one tree.** A body whose members cross jurisdictions is an association
   (ORG). **Answered** (`spec-institutions.md` 3.3): a body selects by role and region and
   keeps no membership table of this spec's kind.
3. **More than one treasury per jurisdiction.** Two governed jurisdictions with one
   seat give two treasuries; a single jurisdiction with a second purse and a treasury
   does not. No consumer needs it.
4. **A law that forbids a move across a border** (6.4.3). A new refusal in
   `spec-regions.md` 6.1.3, or a registration by ORG. Reopen with the first consumer that
   asks.
5. **The region of a treasury.** The seat is the simplest place and makes a levy across
   regions an interregional transfer (POL-A2). `NoRegion` would remove the gap but
   leaves an account that belongs to no region's report, and `spec-regions.md` 6.5.3
   reserves it for an account between regions. **Answered** (`spec-fiscal.md` 4.3): the seat
   region, and `FiscalFlowApplied` is the reconciling item.
6. **Whether a treasury places orders.** A store held for release or a purchase needs admission rules
   for a pseudo-agent that both buys and sells (`spec-clearing.md` 3.1.2). **Answered**
   (`spec-fiscal.md` 9): a treasury places standing orders of purchase and release under the
   admission rule of its 9.4.
7. **The bounds.** `MaxJurisdictions` 1 000 and `MaxPolityDepth` 8 are guesses; the
   first consumer's world needs two or three levels.

## 18. Where this lives in the code

Nothing is written; this is the proposed layout for the first code, checked against
architecture.md 2.6 and `AGENTS.md`.

- `polity.go` (root): `JurisdictionID`, `NoJurisdiction`, `Jurisdiction`, `Member`,
  `Polity`, `Crossing`, `NewPolity`, `World.Polity`, `World.TreasuryAgent`, and the
  creation of the treasury accounts and pseudo-agents in the construction steps of
  `spec-world.md` 5.1. One file per concept (architecture.md 2.6).
- `internal/polity/` (standard library only, primitive slices, no root type,
  architecture.md 2.9): the build of the tables from `parent []int32` and
  `member []int32`, the chain, the common jurisdiction, the crossing and the covered
  lists, and the three combiners of 6.1.2 over primitive slices, using the 128-bit
  checked product of whichever package N-019's first-code decision picks
  (`internal/measure`, `internal/reduce` or a shared `internal/checked`), not a third
  copy. The root converts at the call, inline.
- `config/`: the tables `[modules.polity]`, `[modules.polity.jurisdiction.<id>]` and
  `[modules.polity.member]`, and the rules of 9. No other package reads a file format.
- `testdata/chains/overlay/polity.toml` and `polity-governed.toml` (REF, 12.4).
- Tests: table-driven unit tests in `internal/polity` with the cases of the model;
  external `_test` tests of `NewPolity`, `World.Polity` and the neutrality of FR-POL-019
  over seeds; a failing-case test for each sentinel of 9.1 (testing.md 3.9); a
  checkpoint round trip with two treasuries; a fuzz test on the polity tables of the
  loader (testing.md 7.4); an `AllocsPerRun` test for NFR-POL-001; and an `Example`
  test of `Polity.Crossing` with example A.
- Rule files, at the first code (owed rows, with a fresh tag each): architecture.md
  2.6 (`polity.go`, `internal/polity/`) and 3.3 (`JurisdictionID` among the typed IDs);
  observability-and-logging.md 3.3.2 (the keys `jurisdictions` and `governed`) and
  5.4.1 (the sector constant `treasury`). The glossary words go to `doc.go` (N-017).
