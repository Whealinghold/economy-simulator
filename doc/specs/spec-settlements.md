# Spec: Settlements: the size class of a region, derived from its households, and what each class unlocks or limits

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-SET-NNN` and
`NFR-SET-NNN`, which tests cite (testing.md section 4). They are in section 16.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
later stage (`vision.md` section 4: "Later"). Off unless configured: the zero
configuration has no `Modules.Settlements`, no region has a class, and no rule of
any spec reads one. The core does not depend on it. Needs `spec-regions.md` (REG:
the region, `RegionID`, the one override mechanism of 5.1, N-104),
`spec-scale.md` (SCL: households are the sum of weights, 3, 9),
`spec-world.md` (WLD: one new position in phase 7, 5.1), `spec-observability.md`
(OBS), `spec-checkpoint.md` (CHK) and `spec-config.md` (CFG: the module switch and
validation). It is read by `spec-spawner.md` (SPN) and `spec-population.md` (POP)
through two gates (6.3) and by the specs that are not written yet (STO, ORG, RSK,
CAP), which find their hooks reserved in 6.5. It reads `spec-agents.md` (AGT: the
agent, its weight and region) and `spec-migration.md` (MIG: a migrant is counted
in its origin) and changes none of them. Every change it needs in another spec is
listed in its handoff.

**Every number in sections 3, 4, 8, 12 and 13 is a placeholder** except where a
derivation is shown. Nothing has been run against the library, bazaarBot has not
been run, and no acceptance band exists (notes N-015, N-027).
`models/set/band.py` (section 14) runs the review rule on an assumed household
count: it is a stylised model of the rules of 4, not of the library, and not a
claim that any default is stable. No default of this spec is a macro constant: the
module sets no price, wage, rate or share, and changes who may enter a role and
how many a role may hold, nothing else.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer, never a household (N-001, N-017). The class names in
the worked examples (`hamlet`, `village`, `town`, `city`, `outpost`, `station`,
`hub`) and the flag and limit names (`fair`, `hall`, `merchants`, `dock`) are
**examples** from invented settings and are flagged where they appear; they are
data, never branched on (architecture.md 9.1; `AGENTS.md`, setting-agnostic).

## 1. Purpose and scope

A region is one settlement and one market (`spec-regions.md` 3.1.3, ADR 0003
decision 4). Some settings want the settlement to have a rank: that a place of
forty households lacks some structures and has room for only a few carriers of a
trade, while a place of two thousand has them all (the examples of section 12 give
them names). This spec gives a region a **size
class** from its households and says, in one table, which other specs read it and
for what. It adds a layer and no behaviour of its own. It covers:

- the **class** as data: an ordered list with an enter threshold each, and what
  each carries: **flags** and **limits** (section 3);
- the **household count** that the class is derived from, the **band** and the
  **dwell** that keep it from flapping, the **class floor** of a region and the
  per-region scale of the thresholds, which follow the one override mechanism of
  `spec-regions.md` 5.1 (sections 3.5, 4);
- **where in the tick** the class is set, when it is read, and what a change
  emits (section 5);
- **what a class unlocks or limits**: two forms, the table of hooks with the owner
  spec of each, the rules for a reader, what is reserved for specs not written and
  what is refused (section 6);
- **why a count of households may gate a structure** in a library whose rule is
  that macro is measured and never set, and the guards that make it so (section 7);
- parameters, validation, observability, checkpoint state, register entries and
  requirements (sections 8 to 11, 13, 16).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The market of a region, the markets it holds, how a market clears | `spec-clearing.md` (CLR); REG |
| What happens when a role is closed to entry, how many may enter, the ladder, births | `spec-spawner.md` (SPN), `spec-population.md` (POP) |
| Stores, a store's capacity, its cost | `spec-storage.md` (STO, later) |
| Associations, charters, price controls | `spec-institutions.md` (ORG, later) |
| Defence, raids, loss | `spec-risk.md` (RSK, later) |
| Facilities as capital (a processing site, a terminal); who builds them | `spec-infrastructure.md` (CAP, later) |
| Realms, provinces and the jurisdiction of a region | `spec-polity.md` (POL, later) |
| The number of merchants or carriers a settlement supports **as an economy**: that is what the merchants earn | `spec-merchants.md` (MER), `spec-transport.md` (TRN); SET gives only a ceiling (6.3, H3) |
| The tick, the draws, the checkpoint format | WLD, CHK |

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is a class? | An entry of an ordered list, ordered by its `Enter` threshold (households), the first with `Enter` 0. The `ClassID` is the index. A setting with no sizes has one class | N-026 and ADR 0003: one settlement per region, so this is a thin layer. The names (hamlet, village, town, city) are data and only examples |
| 2 | What is it derived from? | The **household count** of the region: the sum of the weights of its agents (pseudo-agents excluded), the figure `RegionRow.Households` reports; a migrant on a journey is in its origin | One definition for the snapshot and the class, so a consumer can reproduce the class from what it can read (`spec-observability.md` 5.3.2) |
| 3 | Where is it set? | One position, **P7.2a**, after the migration pass of every region. The class is constant from the entry of a tick to P7.2 and every reader sees the one value | A reader at P2.5 and a reader at P3.2 must not see two classes. Phase 7 is where households come and go (`spec-world.md` 6.9) |
| 4 | What keeps it from flapping? | A **band** (a leave threshold below each enter threshold) and a **dwell** (a least number of ticks between two changes); a change that the dwell holds is counted | A count that crosses a threshold by one household must not open and close a gate every tick (model, 14.2) |
| 5 | What can a class carry? | **Flags** (a permission) and **limits** (a count of places or units), each named by the reader, never by this spec. There is no third form | The roadmap lists a market, a store, an association, a number of merchants, defence: each is a permission or a count |
| 6 | Who reads it? | Only the rows of the table of 6.3, each by its owner spec, each at a stated position. A reader outside the table is a defect | A class that anything can read is a macro aggregate that everything reacts to (position 11) |
| 7 | What may a gate do? | Say whether a role may be entered and how many households it may hold, and what a later spec says of its own structures. It never changes a price, wage, rate, share, probability, period or window | These are the "rates" that `spec-world.md` 9.4.1 keeps off every aggregate |
| 8 | Do agents see it? | No. No view of `Work`, `Trade` or `Revise` carries a class, a flag or a limit. The spawner's view carries the effect (an open role and its headroom) | MIG 4 and ADR 0001 decision 7: a household reads its own state, the structure of the world, and rumoured prices |
| 9 | Does a class remove what is there? | No. A gate bars entry. Households already in a role stay (the ladder alone removes them); a ceiling is not enforced on them | A closed gate that threw households out would be a shock the library caused |
| 10 | How does a region differ? | Two rows of the one mechanism of `spec-regions.md` 5.1: a **scale** of the thresholds and a **class floor** (replace) | N-104; a region where a settlement of a given rank is smaller (12.1 flags a mountain region as an example), or a place that has its rank by charter |
| 11 | Is this consistent with "macro is measured, never set"? | It is a stated exception with guards, and the one place where a count of households gates a structure. The conflict with `spec-world.md` 9.4.1 is recorded with a recommended carve-out (section 7) | The brief and the roadmap ask for it; the guards of 7.2 make it a step function, not a controller |
| 12 | Draws, ledger? | None. The review takes no random word, moves no money or goods, adds no reason | 4.7, 5.1 |
| 13 | Can a consumer set a class? | No slot and no call. A consumer changes the households, or the configuration | N-131: INT is minimal. A pin is the `Floor` row, which is configuration |
| 14 | A feedback loop? | Yes: a count gates a structure that moves the count (SET-A1, SET-A2). Registered in section 13 | `spec-stability.md` 9.1 |

## 3. The class

### 3.1 What a class is

```go
// ClassID is the index of a settlement class in ascending order of Enter. It is
// not stable across configurations: adding a class renumbers the others. Use
// the string ID to persist or exchange a class.
type ClassID int

// NoClass means "no class": the From of the event of a founding. A snapshot's
// Class is zero with the module off, and Modules says the module is off.
const NoClass ClassID = -1

// Settlements is the module's configuration: Config.Modules.Settlements, nil
// when the module is off.
type Settlements struct {
	Classes []SettlementClass // at least 1, at most MaxClasses
	BandBP  int64             // the band, in basis points of an enter threshold
	Dwell   int64             // least ticks between two changes of a region's class
	Regions []RegionSettlement
}

// SettlementClass is one size class.
type SettlementClass struct {
	ID     string       // [a-z][a-z0-9_]*, at most MaxIDLength bytes, unique
	Label  string       // presentation only, not hashed
	Enter  int64        // households at or above which the class is reached; 0 for the first
	Flags  []string     // flag IDs the class carries, each at most once
	Limits []ClassLimit // limit IDs the class carries, each at most once
}

// ClassLimit names a limit and gives its value for one class.
type ClassLimit struct {
	Name  string
	Value int64 // a count, 0 to MaxQuantity
}

// RegionSettlement is a row of the region-keyed table of 3.5.
type RegionSettlement struct {
	Region      string
	ThresholdBP int64  // scale of the thresholds in this region
	Floor       string // a class ID: the lowest class the region may have; "" for none
}
```

In a file (the tables merge by key, `spec-config.md` 6.4):

```toml
[modules.settlements]
band_bp = 1000
dwell   = 30

[modules.settlements.class.hamlet]            # example names, flagged
enter = 0
[modules.settlements.class.village]
enter = 300
flags = ["fair"]
[modules.settlements.class.village.limits]
merchants = 2
[modules.settlements.class.town]
enter = 800
flags = ["fair", "hall"]
[modules.settlements.class.town.limits]
merchants = 12

[modules.settlements.region.west]
threshold_bp = 6000
[modules.settlements.region.north]
floor = "village"
```

and in a role (the owner of the gate is named in 6.3):

```toml
[role.merchant]
needs   = "fair"
ceiling = "merchants"
```

### 3.2 Order and identity

3.2.1. The classes are sorted by `Enter` ascending before a world is built or a hash
is taken. This is a **second exception** to the canonical order of `spec-config.md`
3.4 (every entity slice by ID) and to the assignment of typed IDs by sorted ID string
of 3.3 there; the edit to both is owed (handoff row 14). Two classes with one `Enter`
are an error (`ErrOutOfBounds`, rule `enter_order`), so the order is total. The
`ClassID` of a class is its index in that order, from 0: **the rank by `Enter`, not
the rank of the ID string** (in the example names of 12.1 `city` sorts before `hamlet` as a string and is the
last class). The class of index 0 has `Enter` 0 (rule `first_class_open`,
`ErrOutOfBounds`): every region has a class.

3.2.2. The order is by the number that gives it its meaning, not by the ID string.
This differs from regions, whose order is by ID, and from the months and seasons
of the calendar, which are ordered lists whose written order is data
(`spec-calendar.md` 3): the file is a table keyed by ID, so a layer may add or
change a class and the merge needs no rule for lists, and the canonical order is a
sort by `Enter`. Within a class, `Flags` are sorted by name and `Limits` by `Name`
before the hash (`spec-config.md` 3.4 gains them, with the classes, in the same
edit), so two configurations of equal content hash alike (`spec-config.md` 3.6).

3.2.3. The IDs are what a file, a log line and a checkpoint of a pack use. The
`ClassID` is what the engine's arrays use. The names table of `WorldCreated`,
`WorldRestored` and the snapshot gains the class IDs (handoff, `spec-observability.md`
3.2.4).

### 3.3 Flags and limits

3.3.1. A **flag** is a name. A class **carries** a flag by listing it. A reader that
asks "does the region have `fair`" gets true when the class of the region carries
it. A flag is a permission: it says that something may be done, not how much.

3.3.2. A **limit** is a name with a value per class that carries it. A reader that
asks "how many" gets the value of the region's class, or **no limit** when the class
has no row for the name. A limit is a count: of households, of agents, of units, of
places. It is never a rate, a share, a probability, a period or a price; the value
is `0` to `MaxQuantity` and its unit is the reader's (6.2, rule R2).

3.3.3. **This spec does not own the vocabulary.** It does not know what `fair`
means or what `merchants` counts. The set of flags is the union of the flags the
classes list, and the set of limits the union of the limit names; a reader names
one in its own configuration (`Role.Needs`, `Role.Ceiling`, 6.3). A reader that names
a flag no class carries is rejected (`ErrUnknownFlag`, 9.1), so a typo is an error
and not a gate that never opens.

3.3.4. **A flag need not be monotone.** A flag carried by one class and not by the
next is legal: the setting may have a common held by the smaller classes and given up by
the larger (an invented example). A pack author who wants monotone flags writes them in every class above.
No warning is raised for it.

3.3.5. **The absence of a limit row is no limit, and a row of `0` is a closed gate.**
A class that says nothing about `merchants` does not bound them; a class that says
`merchants = 0` supports none. So a limit is a statement about the small classes,
and the large ones omit it.

### 3.4 The thresholds, the band and the dwell

3.4.1. `Enter` of a class is the household count at or above which the class is
reached from below.

3.4.2. `BandBP` (basis points, 0 to `MaxBandBP`) sets, for every class but the
first, the **leave threshold**: the count below which the class is given up. With the
region's effective enter threshold `E_k` (4.2),

```
L_k = floor( E_k * (10 000 - BandBP) / 10 000 )          for k >= 1
```

so a class is reached at `E_k` and kept down to `L_k`. For `BandBP > 0` the leave
threshold is below the enter threshold by at least one household, whatever `E_k`
(4.2.2). With `BandBP` 0 there is no band: the class follows the count and only the
dwell limits it.

3.4.3. `Dwell` (ticks, 0 to `MaxWelfareWindow`) is the least number of ticks between
two changes of a region's class: a change that the review finds due, `t - Since <
Dwell`, is **held** and counted (4.4). `Dwell` 0 has no dwell.

3.4.4. The band answers a count that sits near a threshold and wanders by a few
households. The dwell answers a count that crosses a threshold and a band together
(a plague and its recovery, a harvest year) and would open and close a gate faster
than the structure behind it could respond. Section 14.2 shows each alone and both.
Neither is a claim about what a real settlement does.

### 3.5 Region rows (REG 5.1, N-104)

3.5.1. A region differs from the module's thresholds by **rows** of one table,
`Modules.Settlements.Regions`, the mechanism of `spec-regions.md` 5.1: one row per
region, combined with the default by replace or scale, in canonical order by
`RegionID`, with the errors of 5.3 there (`ErrUnknownRegion`, `ErrDuplicateID`,
`ErrOutsideNode` for an outside node). A row is added to the list of
`spec-regions.md` 5.1.3 (handoff):

| Subject | Table in `Config` | Key | Owner | Mode |
|---------|-------------------|-----|-------|------|
| thresholds of the classes | `Modules.Settlements.Regions` | region | `spec-settlements.md` 3.5 | scale (`ThresholdBP`) |
| the lowest class | `Modules.Settlements.Regions` | region | `spec-settlements.md` 3.5 | replace (`Floor`) |

3.5.2. **`ThresholdBP`** (default 10 000, 1 to `MaxThresholdBP`) scales every enter
threshold of the region but the first: `E_k = ceil(Enter_k * ThresholdBP / 10 000)`
(4.2). A region where a class is reached at half the households that the realm's other regions need has
`ThresholdBP` 5 000. The rounding is ceil, so a region is never given a class
earlier than the scale says.

3.5.3. **`Floor`** is the lowest class the region may have, by ID, default none (the
first class). The class of the region is never below it: not at construction and
not at any review. It models a place that is what it is by charter or by
geography, and is the one place the configuration pins a class. It is
configuration and not a slot (position 13).

3.5.4. An absent row is the default and draws no warning (`spec-regions.md`
5.3.3). The rows are static: no slot edits them (`spec-regions.md` 5.3.6).

### 3.6 A setting without sizes

3.6.1. A setting that has no sizes either leaves the module out or configures **one
class**: `Enter` 0, no flags, no limits. With the module out nothing exists. With one
class and no gate that names a flag or a limit, the module is **inert**: every
result of the run (the state, the events other than the module's own, the draws) is
that of the world without the module (FR-SET-015), apart from what says that the
module is configured: the configuration hash (`Snapshot.ConfigHash`,
`WorldCreated.ConfigHash`), `Modules`, the class IDs in the names table, the `settlement_classes`
count of the construction line, and the module's own snapshot fields and events.

3.6.2. One class that **carries flags** is the way to say "every settlement of this
pack has every flag that a gate names": a gate that names a flag is then open everywhere, and a
role field of a module that is on is not an error. The alternative, a role that
needs no flag, needs no module.

### 3.7 What a class is not

3.7.1. It is not a kind of market. A large and a small settlement clear by the same rules
(`spec-clearing.md`); only what other specs make of the class differs.

3.7.2. It is not a polity or a province (`spec-polity.md`, later) and it is not
inherited from one (`spec-regions.md` 5.4.1).

3.7.3. It is not a field of the region that other specs read as a number. There is
no table "value by class" of another spec's region field (a relief threshold, an
estate markdown): refused, 6.6.

## 4. Deriving the class

### 4.1 The household count

4.1.1. For a region `r` that is not an outside node, the **household count** `H_r`
is the sum of the weights of the live agents whose account is in `r`, pseudo-agents
(the estates, the counterparties of the outside) excluded. It is the figure
`RegionRow.Households` of `spec-observability.md` 5.3.2. A migrant on a journey and
an away traveller are in the region of their account, the region of departure, as
there (`spec-migration.md` 8.1, `spec-transport.md` 5.5). The households of a
region of no agents are 0. At construction the count is the census of the
region's groups, the sum of their `Households` (`spec-scale.md` 3.1), because the
weights of a group add up to its households exactly (`spec-scale.md` 3.2); afterwards
it is the sum of the live weights.

4.1.2. The count is read at P7.2a (5.1), after the weights of the tick are final:
the arrivals of P1.3 and P1.5, the births and deaths, the entries, the switches and
the removals of P7.1, and the departures of P7.2 (a departure leaves the count only
at the arrival, 4.1.1). A call between ticks (an external agent added, a
relocation, a removal) changes the count after the last review and is seen by the
next, so the class of a region is a function of its households **at its last
review**, not of the live count (5.3).

4.1.3. The cost is an integer sum of the weights of the region's agents. The world may
keep a counter per region, updated at every creation, removal, weight change and
relocation, or may scan the agents at P7.2a; the figure is the same and the choice is
the implementation's (NFR-SET-001). A counter is derived state and is rebuilt at a
restore.

### 4.2 The effective thresholds

4.2.1. For a region `r` with row `ThresholdBP` `s` (10 000 without a row), the
effective thresholds of the classes `k = 0 .. K-1` are

```
E_0 = 0                                   L_0 = 0
E_k = ceil( Enter_k * s / 10 000 )        L_k = floor( E_k * (10 000 - BandBP) / 10 000 )     for k >= 1
```

computed once at construction and held in a dense table of `K` entries per region
(`spec-regions.md` 5.3.5). The products are at most `MaxHouseholds * MaxThresholdBP
= 10^9 * 10^5 = 10^14`, far below `int64`; they go through the checked helper
(architecture.md 3.5), which cannot report overflow here.

4.2.2. **Bounds that follow.** With `BandBP` in `[1, MaxBandBP]` and `E_k >= 1`,
`L_k <= E_k - 1`, so a band is never empty; the band is one household whenever
`E_k * BandBP <= 10 000` (`E_k` 10 at 1 000 bp, `E_k` 5 at 1 000 bp, `E_k` 1 or 2 at
5 000 bp; the warning `band_one_household`, 9.2). `L_k >= floor(E_k / 2)` because
`MaxBandBP` is 5 000 (for odd `E_k` the floor division can fall below `E_k / 2`:
`E_k` 3 at 5 000 bp gives `L_k` 1). Two effective thresholds can be **equal** in a region with
`s < 10 000` (`Enter` 5 and 6 at 5 000 both give 3); the class between them is then
never held by a count, and the warning `threshold_collapsed` says so.

### 4.3 The review

4.3.1. A region has a **class** `c` (a `ClassID`), its `Since` (the tick of its last
change) and its class floor `f`. The **review** gives a target class `c'`:

```
c'  = c
while c' + 1 < K  and  H >= E[c' + 1]:     c' = c' + 1        up: every class whose enter threshold the count meets
if c' == c:
    while c' > f  and  H < L[c']:          c' = c' - 1        down: every class whose leave threshold the count is below
```

The two directions are exclusive: going up needs `H >= E[c+1]`, going down needs
`H < L[c]`, and `L[c] < E[c] < E[c+1]` (or `c` is the first or the top class). A
region may move **more than one class** in one review (a flood of migrants, a
plague), subject to the dwell.

4.3.2. **The result is inside its band, except at the floor.** After a review that is
not held, the class `c'` satisfies `H < E[c' + 1]` if `c' + 1 < K`, and satisfies
`L[c'] <= H` **unless `c'` is the floor `f`**: at the floor the count may be below
`L[f]` (4.3.3; example `north` of 12.1: floor `village`, 150 households, `L` of
`village` 270). A class is therefore a function of the count **and of the class it
replaces**: for `L[k+1] <= H < E[k+1]` both `k` and `k+1` are inside their band, and
the region keeps the one it has. That is the hysteresis, and the whole of the memory
the module holds is `c` and `Since`. The model checks the invariant, with and without
a floor, over every count of 0 to 3 000 (14.1).

4.3.3. **The floor.** The down loop stops at `f`, so `c' >= f` always. A region whose
count would give a class below its floor is **at its floor** (a snapshot field,
10.5): the floor acts silently in the sense that nothing is emitted when it does
and the field is how it is seen.

### 4.4 The dwell and a held change

4.4.1. If `c' != c` and `t - Since >= Dwell`, the class becomes `c'` and `Since`
becomes `t`, the tick in which the review ran (the class is read from `t + 1`,
5.2). If `c' != c` and `t - Since < Dwell` the change is **held**: nothing changes,
and the region has a `SettlementHeld` event for the tick (10.1). A held change is
not remembered: the next review recomputes the target from the count it then has,
so a plague that is over when the dwell ends leaves the class where it was
(example A, 12.1).

4.4.2. The dwell runs from the last **change**, not from the last review and not
from the first tick the target differed. A region that moves up and, `Dwell` ticks
later, down is allowed; one that moves up and down in less is not.

4.4.3. The first dwell of a world runs from tick 0 (4.5): a count that crosses a
threshold in the first `Dwell` ticks is held until tick `Dwell`. Regions founded
together are held together and released together; the coincidence is registered in
SET-A1 and open question 4.

### 4.5 Construction and restore

4.5.1. At construction, after the agents are created (`spec-world.md` 5.4) and
before the first step, the class of each region, in ascending `RegionID`, is the
highest `k` with `H >= E[k]`, and not below the floor: **no memory**, so the band
plays no part. `Since` is 0. The event `SettlementClassChanged` with `From`
`NoClass` and the cause `founded` is emitted for each region after the
`AgentSpawned` events (5.4).

4.5.2. A restore installs the saved `c` and `Since` of each region and runs no
review and no event (`spec-checkpoint.md` 6.7): the class is state and the world
continues as the uninterrupted one would (FR-SET-029). A checkpoint taken after a
call between ticks that changed the count keeps the old class, as the uninterrupted
world does until its next P7.2a.

### 4.6 What changes the count

For the reader's orientation only; each is another spec's mechanism, and none
reads the class except through a row of 6.3.

| Change of `H_r` | Position | Owner |
|-----------------|----------|-------|
| Births and deaths of the population module | P7.1(b), (d) | POP |
| Entry, replacement, rung-3 reductions and removals | P7.1(d) | SPN |
| A migration arrival (a departure changes no count) | P1.5 | MIG |
| A traveller's arrival moves its weight from one region's count to the other's | P1.3 | TRN |
| An external agent added, removed or relocated by a call | between ticks | INT |

### 4.7 What the review does not do

It does not read a price, a wage, a stock, an income or any region but its own. It
does not move an agent, a household, money or goods. It does not draw. It does not
call a consumer's code (the gates are read by the world's own positions, 6). It does
not set a rate, a period or a share.

## 5. In the tick

### 5.1 The position

One new position, in phase 7, after the migration pass of every region. It follows
the rules of `spec-world.md` 6.1.1 and FR-WLD-040: a stable ID, one owner, and a
letter after the position it follows. The position follows `P7.2`, so it is `P7.2a`
(as `P2.8a` and `P3.3a` are), and nothing is renumbered.

| ID | What runs | Owner | Reads / writes | Draws | Events |
|----|-----------|-------|----------------|-------|--------|
| P7.2a | The settlement review, with `Modules.Settlements` present, for each region that is not an outside node in ascending `RegionID`: the household count is taken (4.1), the class reviewed (4.3), and a change applied or held by the dwell (4.4) | SET 4 | reads the weights of the region's agents, the saved class, `Since`, the floor and the effective thresholds; writes the class and `Since` | none | per region in that order: `SettlementClassChanged` if the class changed, else `SettlementHeld` if a change was held; never both and never one for a region with nothing due |

5.1.1. **Why after P7.2.** The count of the tick is final only when the entry step of
every region (P7.1) and the migration departures (P7.2) have run; the arrivals are
at P1.5 and the removals at P7.1(d). A review at the end of phase 7 sees every
weight change of the tick. It is before phase 8 so that nothing in phase 8 (ageing,
flow totals, regeneration, the audit) reads a class that is about to change, and
the state is final before the checkpoint position (`spec-world.md` 8.4).

5.1.2. **Why not at the start.** A review at P1.1 would read the count before the
arrivals of P1.3 and P1.5, and would change the class between the end of one tick
and the start of the next with the between-ticks calls made in the gap: the same
one-tick lag, with the calls counted in the next review instead of the next but
one. The end of the tick is the one place where the class is a function of a
settled count.

### 5.2 When the class is read

5.2.1. **The class at tick `t` is the class after the review of tick `t - 1`** (the
class from construction at tick 0). Every position of tick `t` from P0.1 to P7.2
reads that value, and so does a call between ticks and a snapshot. P7.2a of tick `t`
is the one place it changes. A reader never sees two values in one tick.

5.2.2. **The delay is one tick plus the dwell.** A household that arrives at P1.5
of tick `t` is counted at P7.2a of tick `t`, and a gate it opens is read from P1 of
tick `t + 1`. This is the phase-order delay that `spec-world.md` 6.1.8 registers
once (WLD-A1): no entry of 13 counts it again.

### 5.3 Between ticks and after a restore

5.3.1. A call between ticks changes the household count and nothing else: the class
is unchanged until the next P7.2a. A consumer that adds a thousand external agents
sees the same class until a step has run. The snapshot carries both the households
and the class, and `ClassTarget` (10.5) says what a review at the current count
would give.

5.3.2. A checkpoint is taken between ticks and holds the class and `Since` of each
region (11). A restore installs them (4.5.2) and the next P7.2a is the one the
uninterrupted world would have run.

### 5.4 The order of events

5.4.1. **At construction**, after `WorldCreated` and the `AgentSpawned` events of the
initial agents (`spec-world.md` 5.8): one `SettlementClassChanged` per region in
ascending `RegionID`, `From` `NoClass`, cause `founded`, `Tick` 0. An observer then
has every class from events, as it has every agent.

5.4.2. **In a step**, in phase 7 after `MigrationReviewed` of the last region
(`spec-world.md` 6.11 item 8): the events of P7.2a in region order. They come before
phase 8's `BoundaryFlow` and `TickEnded`.

### 5.5 What this asks of WLD

The consolidated table is in the handoff. In short: the row P7.2a in 6.9; item 8 of
6.11; the construction events of 5.8; a row in the table of 9.2 (the dwell, a gate on
`t - Since`); the carve-out of 9.4.1 (section 7); and nothing in 7.3 (no draw), 8.2
(no slot) or 12 (the state is the module's, section 11 here).

### 5.6 Cost

P7.2a is `O(R)` with a counter per region and `O(A)` with a scan of the agents (4.1.3),
once per tick: no term is added that grows with households (`spec-world.md` 11.2).
The events are bounded by regions. The gates are table reads (6.1).

## 6. What a class unlocks or limits

### 6.1 Two forms, named by the reader

6.1.1. A reader names a **flag** or a **limit** in its own configuration (3.3.3).
At construction the world resolves each name to an index, and the classes to a dense
table: one word per class for the flags it carries (at most `MaxFlags` flags, so
one `uint64`), one word for which limits it has, and the values. The reads are

```go
// internal: resolved at construction, no allocation.
func (w *World) regionHas(r RegionID, f flagIndex) bool
func (w *World) regionLimit(r RegionID, l limitIndex) (int64, bool) // false: no limit
```

and read the class of the region as it stands (5.2). They are not exported: a
consumer reads the class and the class table from the snapshot and `Config()` (10.5).
This spec adds no public call beyond the type `ClassID` and the constant `NoClass`.

### 6.2 Rules for a reader

Binding on every row of 6.3 and on every spec that adds one (position 6).

| Rule | Statement |
|------|-----------|
| R1 | A reader reads the class only through `regionHas` and `regionLimit`, at the position its row states, and the value is the class set at the last P7.2a (5.2). It does not keep a copy across ticks |
| R2 | A limit is a count of units of something: households, agents, contracts, places, units of capacity. It is never used as a rate, a share, a probability, a period, a window, a price or a wage, and never multiplied into one |
| R3 | A gate bars **entry or creation**. It does not remove what exists (position 9). A reader that wants a removal says so in its own spec, with a register entry |
| R4 | Every refusal a gate causes is counted in an event of the reader or a bounded `Warn` (FR-STB-013, N-045) |
| R5 | A configuration field that names a flag or a limit while the module is off is rejected with `ErrModuleDependency`. No reader has a default of its own for "no module" |
| R6 | A gate that could close what a household needs to eat is a hazard and has a register entry of its own (SET-A2 is the pattern) |
| R7 | A reader that is added adds its row to 6.3 in the same change, as a slot adds a row to `spec-world.md` 8.2 |

### 6.3 The table of hooks

| # | Hook | Form | Reader's field (proposed) | Owner of the behaviour | Read at | Status |
|---|------|------|---------------------------|------------------------|---------|--------|
| H1 | Entry to a role | flag | `Role.Needs` | SPN (the default spawner); POP (births) | P7.1(b) births; P7.1(c) the view and the plan | proposed, an edit owed (6.4.1) |
| H2 | How many households a role may hold in a region | limit | `Role.Ceiling` | SPN; POP | P7.1(b), (c), (d) | proposed, an edit owed (6.4.2) |
| H3 | How many merchants or carriers a settlement supports | limit | `Role.Ceiling` on the merchant or carrier role; no field of MER or TRN | MER, TRN own the roles; SPN and POP the gate | as H2 | follows from H2 (6.4.3) |
| H4 | A market charter | flag | by ORG | ORG; CLR for admission | later | reserved (6.5) |
| H5 | Which store classes a region may have (a store is room for goods) | flag; a limit for how many stores of a class | `StoreClass.Needs` (proposed, STO 10.3 has the seam) | STO | `AddStore` and the construction of an agent with role stores, a call between ticks or a position before P7.2a | reserved; STO has the seam (`spec-storage.md` 10.3: a gate a later module registers, a predicate over (region, class)); its field `Needs`, the predicate `regionHas(region, class.Needs)` and the refusal `ErrStoreLocked` are owed by edit 31 of the handoff |
| H6 | An association | flag | by ORG | ORG (and the barriers of SKL) | later | reserved |
| H7 | Defence | limit | by RSK | RSK | later | reserved |
| H8 | A facility: a processing site, a terminal (of a transport mode) | flag | by CAP; `Mode.Needs` in TRN | CAP; TRN | later | reserved |
| H9 | A recipe condition | flag | by PRO | PRO | | **not offered** (6.6) |
| H10 | A market for a commodity | flag | by CLR | CLR | | **not offered** (6.6) |

The first three are the only readers of version 1 of this module: they use rules
that exist (`Closed`, the entry caps, `PopBirths`) and need one field each. Without
them the module has a class and nobody who reads it.

### 6.4 What H1 and H2 ask of their owners

6.4.1. **H1, `Role.Needs`** (a flag ID, empty for none). In a region whose class does
not carry the flag the role is **closed** in the sense of `Role.Closed`
(`spec-spawner.md` 6.2, FR-SPN-041; `spec-population.md` 3.7.3): the default
spawner places no entrant and no switcher in it, rung 1 does not enter it,
like-for-like replacement does not apply (`spec-spawner.md` 7.2 rule 1 needs the role
open) and the replacement goes to another open viable role or is dropped and counted
(`NoViableRole`), and births into it are not formed and its birth carry does not
advance. The spawner's view
carries the role as closed **for that region**. Households already in the role stay,
and are exposed to the schedule's deaths and to the ladder as before. A role may have
both `Closed` and `Needs`; it is closed if either says so. An external agent and a
consumer's own spawner may place a household in the role: as for `Closed`, a gate
is a rule of the default and not an invariant of the world.

6.4.2. **H2, `Role.Ceiling`** (a limit ID, empty for none). In a region whose class
has the limit, the role's **headroom** is `max(0, limit - households of the role in
the region)`; with no limit there is none (the view says `NoLimit`). The default
spawner admits an entrant or a switcher into the role only while the weight of the
agent fits the headroom, and counts a refusal (a new field `RoleCeiling` of
`EntryLimited`); births of the role are capped at the headroom and counted (a new field
`BirthsCeiling` of `PopulationChanged`, owed to POP and OBS; `BirthsCapped` stays the
count of `HouseholdCap`, `spec-population.md` 4.4.2). The households of the role are counted at the start of the
region's entry step and each admission in the plan reduces the headroom. A ceiling
is **not** enforced on what is there: a fall of the limit, a weight change or an
external agent can leave the role above it, and no household is removed for it.

6.4.3. **H3.** A merchant and a carrier are roles (`spec-merchants.md`,
`spec-transport.md` 8). "How many merchants a place can support" is H2 on the
merchant role with the limits of the small classes, and "whether a place has a
carrier at all" is H1, or H2 with a limit of 0. MER and TRN change nothing. How many
a place can **feed** is what the merchants earn and is MER's loop (L84, L113); the
ceiling is only an upper bound set from outside it.

6.4.4. **The effect on the viability check.** A group that exists at construction in
a role whose `Needs` the region's class does not carry is legal and is warned
(`needs_unmet_at_start`, 9.2): the role will not be refilled like for like and will not breed. The
solver's warm start and its steady state do not read the class (`spec-viability.md`
is unchanged); a pack that gates a role that the steady state needs is the author's
to check, and SET-A3 is the hazard.

6.4.5. **A first-come cap.** A ceiling that binds serves the entrants in the order of
the spawner's plan: need, then agent ID (`spec-spawner.md` 6.3). That is one more
place of the kind `spec-world.md` 10.2 lists (O2, O3), and the remedy is the same
(the keyed tie of 10.1.1, D87 there); it is SET-A5, and the counter is `RoleCeiling`.

### 6.5 What is reserved for specs not written

The four later specs that the roadmap names each need a size. The promise of this
spec to them is exactly this: the two reads of 6.1, at a position of their
choice before P7.2a of the tick (so they read the class of 5.2), a flag or a limit
named in **their own** configuration with the rule R5, and a row in 6.3 and an entry
of their own in the register. Nothing else: this spec does not say what a store
holds, what a charter allows, what a defending force does or what a processing site is.

| Spec | Likely use | The form |
|------|-----------|----------|
| STO (`spec-storage.md` 10.3) | which store classes a region may have: a flag named by a store class (`Needs`, owed to STO), tested at `AddStore` and at the construction of an agent with role stores, refusal `ErrStoreLocked` (to be defined by STO); a cap on the stores of a class in a region would be a limit | flag, limit |
| ORG | which associations a region may have; whether it may hold a market charter | flag |
| RSK | the strength of a defending force, fortification | limit, flag |
| CAP | which facilities may be built; a mode's terminal | flag |

### 6.6 What is not offered, and why

6.6.1. **A market per commodity (H10).** A flag on a commodity that closes its
market in a region whose class lacks it. Refused for version 1 of this module:
AGT's tradable set is per role and not per region (`spec-agents.md` 3.4), so a
household would plan orders that the market refuses every tick, and the strategy's
view would have to hide the commodities of the closed markets; beliefs
(`spec-beliefs.md` 3.1), the ladder of wants (`spec-demand.md`) and the warm start
assume a market where a role can trade; and the viability check judges sourcing by
region. A setting that wants a good sold only in larger places gates the **role**
that sells it (H1), or uses a charter (H4), whose owner says what an agent sees of a
closed market.

6.6.2. **A recipe condition (H9).** `spec-production.md` 6.7 defers building
conditions. The yield table already says where a recipe cannot run (a factor of 0),
the viability check reads that table (`spec-viability.md` FR-VIA-005), and a
facility with its own state belongs to CAP. A class-keyed recipe condition would
make the runnable set depend on a count the solver does not have.

6.6.3. **A class-keyed region field or rate.** A relief threshold, an estate
markdown, an entry share or a birth scale that depends on the class would make a
**rate** a function of a count, which is what `spec-world.md` 9.4.1 forbids and
what R2 excludes. A region already has its own row for each of them
(`spec-regions.md` 5.1.3).

6.6.4. **A limit on the households of a region.** POP's `HouseholdCap` or any bound on
the count that gives the class would make the class **absorbing**: a cap below the
next enter threshold can never be crossed. A ceiling is on a **role** (H2), whose
count is not the count of the region.

### 6.7 What an agent sees

6.7.1. Nothing. The views of `Work`, `Trade` and `Revise` carry no class, flag or
limit (position 8, FR-SET-018). A household reads its own state, its market's
report, the structure of the world and rumoured prices; a size class of its
settlement is a measure of the realm and would be a controller input in disguise.
A strategy that wants to behave by size can read the snapshot on the consumer's
side, outside the engine, and the register says what that does (L121).

6.7.2. The spawner is not an agent. Its view carries what the gates did: a role is
open or closed **in the region** and has a headroom (6.4); it does not carry the
class.

## 7. Why a count may gate

### 7.1 The conflict

ADR 0001 decision 7: "the engine holds no aggregate state that agents react to".
`spec-stability.md` 5.1.3: a damper "does not read a macro aggregate and feed it
back". `spec-world.md` 9.4.1: "No cadence, period, window length, gate probability
or share of the library is a function of a measured aggregate of the world: a price
level, a money supply, a count of households or agents". The class is a function of
a count of households, and a ceiling (H2) is a number that depends on it. This is
the one place in the library where that is true, and the roadmap asks for it.

Four accepted texts say the rule, not one: `spec-world.md` 9.4.1, ADR 0001 decision
7, `spec-stability.md` 5.1.3 and its requirement FR-STB-020 ("The library MUST NOT
read a macro aggregate and feed it back into an agent's decision or a market's
price"), and the roadmap's section "Macro is measured, never set". Read to the
letter, FR-STB-020 names an agent's decision and a market's price, and the spawner's
entry rule is neither; the spirit of all four is the same, and this spec does not
argue the letter. It asks each owner to name the one exception (7.4).

### 7.2 What makes it a different thing

| # | Guard | Where |
|---|-------|-------|
| G1 | **A step function.** `K - 1` steps, quantised, not a continuous response; a region can be moved up at most `K - 1` times without moving down | 3.1, 4.3 |
| G2 | **It gates a permission or a count of places**, never a rate, price, wage, share, probability, period or window (R2). The things `spec-world.md` 9.4.1 keeps off an aggregate stay off it | 6.2, 6.6.3 |
| G3 | **Damped.** A band, a dwell and a class floor; every held change is counted | 3.4, 4.4, 3.5.3 |
| G4 | **One position, one value** in a tick; the class is state, saved and restored | 5.2, 11 |
| G5 | **No agent reads it.** Views carry nothing; only the world's own positions read, and only through the table | 6.7, 6.3 |
| G6 | **Off by default and inert with one class.** The core does not depend on the module | 3.6, FR-SET-015 |
| G7 | **A closed set of readers**, each with an owner, a position and a register entry; a new one is an edit here | 6.2 R7, 6.3 |

### 7.3 What remains

The guards make the gate a rank the world has, like the season that a calendar
gives a tick: not a response to anything the household does. They do not remove the
loop that follows, in which the count moves the gate and the gate (through the role
it opens or closes) moves the count. That loop is registered: it can ratchet up at
most `K - 1` classes (SET-A1), and it can fall to the bottom class and stay (SET-A2).
Both are conditions on a pack, and the first is bounded by construction.

### 7.4 The recommendation

Recorded as a conflict between specs in the handoff, with the owners deciding. The
decision is **one decision with five edits**, not one sentence in WLD:

| Owner | Edit |
|-------|------|
| `spec-world.md` 9.4.1 | one sentence added to the paragraph: "The one exception is the settlement class of `spec-settlements.md`, a step function of the household count of a region with a band and a dwell, which gates entry to a role and a count of places and nothing else." |
| ADR 0001 decision 7 | an amendment, or a new ADR that supersedes the decision in this one respect: "the engine holds no aggregate state that agents react to" gains "; the settlement class is a derived structural rank, read by the world's own entry rules and by no agent" |
| `spec-stability.md` 5.1.3 and FR-STB-020 | one clause each: "...except the settlement class of `spec-settlements.md` (SET-A4), which is a rank and not a damper" |
| `roadmap.md` "Macro is measured, never set" | one sentence after the first paragraph naming the class as the single structural exception |

The alternative, if the owners do not accept a count that bounds a number, is to offer
**flags only**: a permission is a step function of a count with no number in it. The
module then has no `Limits` (3.3.2), H2 and H3 are dropped, and "how many merchants"
is a configured constant per role (open question 1). **The fallback needs the same
decision.** A flag still reads a count of households and changes who may enter a
role, so it is under the same four texts; it removes only the number that a count
would bound, not the reading of the count. If no owner accepts the reading, the
module has no class to give and the SET row of the roadmap is withdrawn.

## 8. Parameters

All are tunables (architecture.md 9.5) with a default (or "required"), an owner, a
bound and a doc line, fields of `Config.Modules.Settlements` unless stated, with
rows in the parameter-owner table. **Every default is a placeholder** (N-015,
N-027): none comes from a source, and no source for how large a settlement must be
to have a given structure was read (N-018). The values come from the model of section 14, not
from any society.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Settlements` | the module: present means on (`spec-config.md` 3.2); it needs no other module | nil (off) | |
| Settlements | `Classes` | the size classes (3.1) | required, at least one | 1 to `MaxClasses` |
| Settlements | `BandBP` | the band: how far below an enter threshold the count may fall before a class is given up, in basis points of the enter threshold (3.4.2) | 1 000 | 0 to `MaxBandBP` |
| Settlements | `Dwell` | least ticks between two changes of a region's class (3.4.3) | 30 | 0 to `MaxWelfareWindow` |
| Settlements | `Regions[]` | rows of the region-keyed table (3.5) | none | at most `MaxRegions` |
| SettlementClass | `ID` | the class's identity | required | `[a-z][a-z0-9_]*`, at most `MaxIDLength` bytes, unique |
| SettlementClass | `Enter` | households at which the class is reached (3.4.1) | required; 0 for the first | 0 to `MaxHouseholds`, strictly increasing |
| SettlementClass | `Flags` | the flags the class carries (3.3.1) | none | each at most once; at most `MaxFlags` distinct in the module |
| SettlementClass | `Limits` | the limits the class carries (3.3.2) | none | each name at most once; at most `MaxLimits` distinct in the module; `Value` 0 to `MaxQuantity` |
| SettlementClass | `Label` (p) | display name; presentation only, not hashed | empty | at most `MaxLabelBytes` |
| RegionSettlement | `Region` | the region the row is for | required | a defined region that is not an outside node |
| RegionSettlement | `ThresholdBP` | scale of the enter thresholds, mode scale (3.5.2) | 10 000 | 1 to `MaxThresholdBP` |
| RegionSettlement | `Floor` | the lowest class, mode replace (3.5.3) | none | a defined class ID |
| Role | `Needs` | H1: the flag the region's class must carry for the role to be open (6.4.1) | none | a flag some class carries; the module on |
| Role | `Ceiling` | H2: the limit that bounds the households of the role in a region (6.4.2) | none | a limit some class carries; the module on |

8.1. **The bounds** (`bounds.go`, provisional, architecture.md 9.6): `MaxClasses` 16
(a ranking of settlements is short, and each region holds `K` thresholds);
`MaxFlags` 64 and `MaxLimits` 64 (so a class's flags fit one `uint64`, 6.1);
`MaxBandBP` 5 000 (a leave threshold is at least `floor(E_k / 2)`, so a region that
has just reached a class is not a few households from losing it);
`MaxThresholdBP` 100 000 (ten times, as `MaxYieldBP`). `MaxHouseholds`,
`MaxQuantity`, `MaxWelfareWindow`, `MaxIDLength`, `MaxRegions` and `MaxLabelBytes`
are the existing bounds of `spec-config.md` 4.1.

8.2. **The placeholders and why.** `BandBP` 1 000 and `Dwell` 30 come from the model of
14.2. With a mean-reverting count whose standard deviation is 2% of the threshold, a
band of 500 bp still flips about once in 3 650 ticks and a band of 1 000 never does;
at 4% a band of 1 000 flips about three times and one of 2 000 never; with no band and
no dwell the same count flips 477 to 1 330 times (1 330 at 0.9%, 817 at 2.0%, 557 at
3.9%, 477 at 7.6%). The dwell alone brings the 1 330 to 115. `Dwell` 30 is the base
welfare window (`spec-welfare-and-wages.md` 2.3.1), so a class cannot change faster
than the incomes that the gate it sets would move are measured: that is the hand
argument of SET-A1 (13), not a result of the model, and the warning
`dwell_below_window` (9.2) is raised when a gated role has a longer window. The thresholds of the examples (12) are invented.
No default is a claim about how settlements grow.

8.3. **Rule constants** (beside their code, architecture.md 9.6): the two directions of
the review and their order (4.3); the rounding, ceil for an effective enter
threshold and floor for a leave threshold (4.2); the first dwell from tick 0 (4.4.3);
the cause constants `founded`, `grew` and `shrank`; the position P7.2a.

8.4. **Not parameters:** the `ClassID`, the effective thresholds (derived from
the class, the band and the row), `Since` (state), the household count (derived), a
role's headroom (derived), and the household floor of a class (it is `Enter`).

## 9. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9). Errors name the
path, for example `modules.settlements.classes[2].enter`. The framework of
`spec-config.md` 7 runs them; the three rules that read a role are run with the
roles.

### 9.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | an `Enter` outside `[0, MaxHouseholds]`; the first class with an `Enter` other than 0 (rule `first_class_open`); two classes with one `Enter`, or `Enter` not strictly increasing in the sorted order (rule `enter_order`); `BandBP`, `Dwell` or `ThresholdBP` outside its bound; a limit `Value` outside `[0, MaxQuantity]` |
| `ErrInvalidID` | a class ID, a flag or a limit name that does not match `[a-z][a-z0-9_]*` or is longer than `MaxIDLength` |
| `ErrDuplicateID` | a class ID twice; a flag twice in one class; a limit twice in one class; two rows for one region |
| `ErrTooMany` | more than `MaxClasses` classes, `MaxFlags` distinct flags, `MaxLimits` distinct limits or `MaxRegions` rows; a module with no class (rule `no_class`) |
| `ErrUnknownRegion` | a row naming an undefined region |
| `ErrOutsideNode` | a row naming an outside node of the external module (it has no household and no class) |
| `ErrUnknownSettlementClass` (new) | a `Floor` naming a class that is not defined |
| `ErrUnknownFlag` (new) | a `Role.Needs` naming a flag that no class carries |
| `ErrUnknownLimit` (new) | a `Role.Ceiling` naming a limit that no class carries |
| `ErrModuleDependency` | a `Role.Needs` or `Role.Ceiling` while `Modules.Settlements` is nil (R5) |

No `ErrOverflow` arises: the largest product is `MaxHouseholds * MaxThresholdBP`
(4.2.1). The new sentinels are root sentinels matched with `errors.Is`
(note N-075: the root needs a mapping, in the handoff).

### 9.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at
construction, at `Warn`, with the attributes `rule` and `path`; the volume is bounded
by the configuration. None is raised in a tick.

| Warning | Condition |
|---------|-----------|
| `needs_unmet_at_start` | a role with a `Needs` that has a group in a region whose class at construction does not carry the flag (6.4.4, SET-A3) |
| `ceiling_below_start` | a role with a `Ceiling` whose group in a region has more households than the limit of the region's class at construction: the role starts above its ceiling and nobody is removed for it |
| `dwell_below_window` | `Dwell` below the effective welfare window of a role that has a `Needs` or a `Ceiling` (the delay argument of SET-A1; `spec-welfare-and-wages.md` 2.3.3) |
| `band_one_household` | a class whose effective band `E_k - L_k` is one household in some region (4.2.2) |
| `threshold_collapsed` | two classes with equal effective enter thresholds in a region (4.2.2) |
| `class_unreachable` | a class whose smallest effective enter threshold over the regions exceeds the households of the whole realm at construction, with the population module off |
| `flag_unused`, `limit_unused` | a flag or a limit that no role names (dead data; informational, since a later spec may name it) |

## 10. Observability

10.1. **Events.** Both are plain data, with no map, no pointer and no wall-clock time
(observability-and-logging.md 4.1), emitted only after the transition they describe is
complete, and constructed only when an observer was supplied (4.5 there). The fields
that name a region or a class carry the integer ID; the strings come from the names
table (`spec-observability.md` 3.2).

| Event | Owner | Position | Class | Gate | Fields (besides `Tick`) |
|-------|-------|----------|-------|------|-------------------------|
| `SettlementClassChanged` (new) | SET 4, 5.4 | construction, after the `AgentSpawned` events (`founded`); P7.2a | E | settlements | `Region`, `From`, `To` (`ClassID`; `From` is `NoClass` at the founding), `Households int64` (the count the review used), `Cause` (`founded`, `grew`, `shrank`) |
| `SettlementHeld` (new) | SET 4.4 | P7.2a, region ascending, instead of the above | A | settlements | `Region`, `Class`, `Target` (`ClassID`), `Households int64`, `TicksLeft int64` (`Dwell - (t - Since)`, at least 1) |

`SettlementClassChanged` is per region, like `AgentRelocated` is per agent: it is a
transition of one region, at most once per region per `Dwell` ticks (once per tick
with `Dwell` 0). `SettlementHeld` is an aggregate of the kind of `EntryLimited`: once
per region per tick in which the dwell held a change, never otherwise; it is the
counter of the damper (FR-STB-013). Neither is emitted with the module off. The
count of events in `spec-observability.md` 4.1 rises by two, and the volume classes
E and A by one each.

10.2. **Logs.** Constant lowercase messages, typed attributes, `Enabled` guards.

- `Info`, once, in the existing "world created" line: `settlement_classes` (a count) joins the
  summary beside `regions`, and `settlements` joins the `modules` list. No other
  `Info`.
- `Warn`, at construction, one per warning of 9.2, with `rule` and `path`.
- `Debug`, per change: "settlement class changed" with `region`, `from_settlement_class`,
  `to_settlement_class`, `households` and `cause`; per held change: "settlement class held" with
  `region`, `settlement_class`, `target_settlement_class` and `households`. Both guarded by `Enabled`.
- No `Error`. No `Info` or `Warn` inside the tick: a class change is a fact, so it is
  an event (`rules/observability-and-logging.md` 3.2.2).

10.3. **New log keys** for observability-and-logging.md 3.3.2: `settlement_class`, `from_settlement_class`,
`to_settlement_class`, `target_settlement_class` (class IDs, `int64`) and `settlement_classes` (a count, in the
construction summary). The keys carry the word `settlement` because "class" already
names a commodity class (`Commodity.Classes`, and the `ErrUnknownClass` of the
`indicator` package): the term of this spec is always **settlement class** where the
two could meet, and the root sentinel is `ErrUnknownSettlementClass`. `region`,
`households` and `cause` exist. The values of
`cause` here are `founded`, `grew` and `shrank`, exported constants of the root
package (`event.go`); `internal/logkeys` holds the key strings only.

10.4. **Metrics.** Two instruments for the catalogue of
`rules/observability-and-logging.md` 5.3 (names and attributes are proposals for OBS):

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.settlement.class` | Gauge | `{class}` | `economysim.region` | `SettlementClassChanged` (`To`); the adapter keeps the last value, which the construction events set |
| `economysim.settlement.changes` | Counter | `{change}` | `economysim.region` | `SettlementClassChanged`, except the cause `founded` |

Cardinality is bounded by the regions (5.4.1 there): five series each for the first
consumer's world. A held change is an event, not a metric; its count over a run is
derivable from `SettlementHeld`.

10.5. **Snapshot.** The `RegionRow` of `spec-observability.md` 5.3.2 gains four fields,
none of them state of its own beyond what 11 saves:

| Field | Type | Meaning | Cost |
|-------|------|---------|------|
| `Class` | `ClassID` | the class of the region as of the last review (5.2) | K |
| `ClassSince` | `int64` | the tick of its last change; 0 at construction | K |
| `ClassTarget` | `ClassID` | the class a review at the current `Households` would give, ignoring the dwell (4.3): a pure function of the count, the class and the configuration | D |
| `ClassAtFloor` | `bool` | the class is the region's `Floor` and the count alone would give a lower one (4.3.3) | D |

A field absent for want of the module is zero and `Modules` says which are on
(`spec-observability.md` 5.1.3). INT's query `Region(r)` returns the row
(`spec-integration.md` 6.5), so the public face needs no call of its own. A snapshot
is a function of the saved state (`spec-observability.md` 5.2.1): `Class` and
`ClassSince` are saved, `ClassTarget` and `ClassAtFloor` are derived.

10.6. **No observer, no logger.** No event is constructed without an observer, and the
review allocates nothing (NFR-SET-002). Observation never changes a result
(observability-and-logging.md 1.4).

## 11. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, plain data in a
fixed order with no maps, for each region that is not an outside node in ascending
`RegionID`:

- `Class`, an unsigned index into the sorted classes;
- `Since`, the tick of the last change (unsigned: 0 at construction).

A **dense** section, present when `Modules.Settlements` is configured and absent
otherwise (`spec-checkpoint.md` 4.2.1), written with the other module sections. A
hosted per-agent field: none (N-128). A diagnostic run: none (a held change is
counted by an event, and the dwell is a function of `Since`).

The decoder checks (`spec-checkpoint.md` 5.3): `Class` is below the number of
classes, is not below the region's `Floor`, and `Since` is at most the checkpoint's
tick less 1 (0 at tick 0). The effective thresholds, the flag and limit tables, the
household counter and the headroom are derived and rebuilt. The class of a restored
world is the saved one even when a call between ticks has moved the count across a
threshold since the last review (4.5.2). The configuration hash covers `Classes`
(`ID`, `Enter`, `Flags`, `Limits`), `BandBP`, `Dwell` and `Regions` in canonical
order (3.2: classes by `Enter`, `Flags` by name, `Limits` by `Name`, rows by region),
every field except `Label` (`spec-config.md` 8).

## 12. Worked examples

Three settings, flagged as **examples**: the names and numbers are placeholders, chosen
to show that the core has no setting in it. The arithmetic of the tables is the output
of `models/set/worked.py` (section 14).

### 12.1 Example A: a realm of five regions (an early-modern-like example setting)

Classes `hamlet` 0, `village` 300, `town` 800 and `city` 2 000 households, a band of
1 000 bp and a dwell of 30. The flag `fair` is carried by `village` and above, `hall`
by `town` and above; the limit `merchants` is 2 for a `village` and 12 for a `town`
and absent for a `city`. A role `merchant` has `needs = "fair"` and
`ceiling = "merchants"`. Two region rows: `west` is a mountain region with
`threshold_bp` 6 000, and `north` has the floor `village`. The regions are those of
`spec-reference-chain.md` 4.4.4 (`five`), with invented households.

```toml
[modules.settlements]
band_bp = 1000
dwell   = 30
[modules.settlements.class.hamlet]
enter = 0
[modules.settlements.class.village]
enter = 300
flags = ["fair"]
[modules.settlements.class.village.limits]
merchants = 2
[modules.settlements.class.town]
enter = 800
flags = ["fair", "hall"]
[modules.settlements.class.town.limits]
merchants = 12
[modules.settlements.class.city]
enter = 2000
flags = ["fair", "hall"]
[modules.settlements.region.west]
threshold_bp = 6000
[modules.settlements.region.north]
floor = "village"
```

| Region | `ThresholdBP` | Floor | Effective `E` (village, town, city) | Leave `L` | Households | Class at construction |
|--------|--------------:|-------|-------------------------------------|-----------|-----------:|------------------------|
| `centre` | 10 000 | none | 300, 800, 2 000 | 270, 720, 1 800 | 1 400 | `town` |
| `east` | 10 000 | none | 300, 800, 2 000 | 270, 720, 1 800 | 900 | `town` |
| `north` | 10 000 | `village` | 300, 800, 2 000 | 270, 720, 1 800 | 150 | `village` (the count alone gives `hamlet`; `ClassAtFloor` is true) |
| `south` | 10 000 | none | 300, 800, 2 000 | 270, 720, 1 800 | 1 100 | `town` |
| `west` | 6 000 | none | 180, 480, 1 200 | 162, 432, 1 080 | 240 | `village` (the count alone gives `hamlet` at the realm's scale) |

- **A change and a held change.** `centre` gains 3 households a tick from 1 400: at
  tick 200 it has 2 000, the review gives `city`, `200 - 0 >= 30`, so the class
  changes and `Since` is 200 (read from tick 201). A plague at tick 205 takes the
  count to 1 790, below `L` of `city` (1 800): the target is `town`, `205 - 200 = 5 <
  30`, so the change is **held**, and `SettlementHeld` is emitted every tick from 205
  with `TicksLeft` 25, then 24, down to 1 at tick 229. If the count is back at 1 810 by
  tick 230 the target is `city` again, nothing changes and no event says so beyond the
  end of the holds; if it is still 1 790 the class changes to `town` at tick 230 and
  `Since` is 230. A band alone would have held the class at 1 800 and above; the
  dwell is what held it between 1 790 and tick 230.
- **The gates.** `east` is a `town`: its `merchant` role has the limit 12. With 9
  merchant households its headroom is 3; four entrants apply, three are admitted and
  one is refused and counted in `RoleCeiling` of `EntryLimited`. `north`, a `village`
  by floor, carries `fair`, so the role is open there with the limit 2. Had `north`
  no floor it would be a `hamlet`, which carries no `fair`: the role would be closed
  to entry in `north`, its `merchant` households there would stay and would not be
  replaced like for like (6.4.1), and the warning `needs_unmet_at_start` would have said so.
- **The scale row.** `west` has 240 households. At the realm's scale a `village`
  needs 300 and the region is a `hamlet`; the row makes it 180 and the region is a
  `village`, with its leave threshold at 162.

### 12.2 Example B: an orbital colony (a space setting; a household is a crew)

Classes `outpost` 0, `station` 20, `hub` 150, a band of 1 000 bp. `station` carries
`dock` and the limit `berths` 3; `hub` carries `dock` and `yard` and `berths` 12;
`outpost` carries nothing. A role `hauler` has `needs = "dock"`, so no outpost can
add one. Effective thresholds 20 and 150, leave thresholds 18 and 135.

| Households at construction | Class |
|---------------------------:|-------|
| 5, 19 | `outpost` |
| 20, 149 | `station` |
| 150 | `hub` |

A review of a `hub` with 135 households keeps it (`135 >= L`); with 134 it gives
`station`. The same colony after a hull breach that takes 20 crews from a `hub` of
150 (to 130) is `station` after the dwell, loses `yard`, and its `berths` falls from
12 to 3: the haulers it already has stay.

### 12.3 Example C: a setting with no sizes (a trading-card world)

One class `all`, `Enter` 0, no flags and no limits (or the module absent). The review
gives class 0 for 0, 1 and 10^9 households and no change is ever due. The world is
the world without the module (FR-SET-015), and the configuration hash differs.

### 12.4 Use by the reference chain (a proposal for REF)

The packs of `spec-reference-chain.md` have no sizes and keep the module off. An
overlay `overlay/settlements.toml` for `five` carries the classes of 12.1 with a
`merchant` role gated by `fair` and a ceiling, and serves the scenarios SET-S1 to SET-S3
of 13. It needs the merchant module and a role with a group in each region whose
class carries or lacks the flag; `famine` is the base for SET-S3. The proposal is in
the handoff.

## 13. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9 (FR-STB-003). Fields as in
its section 3; the register owner assigns the `STB-Lnn`, `-Dnn` and `-Snn` numbers,
and the entries here are `SET-An`, `SET-Dn` and `SET-Sn`.

**What the checklist found (9.1).** The review reads a count of households and
changes what a role may do; what a role may do changes the count. That is a loop
(SET-A1, SET-A2), and a rule that reads a count of agents is the checklist's first
example. The reads of 6.3 are the whole of it. No other rule is added.

**SET-A1. The staircase.** *Kind:* loop. *Loop:* households (+) class (+) a role is
open to entry or its ceiling rises (+) households of that role and what they make and
earn (+) welfare ratio of the region (+) births, entries and arrivals (+) households.
All links are +, so **R**, reinforcing, and **saturating**: there are `K - 1` steps.
*Delay:* one tick (P7.2a to the next P1) plus the dwell plus the delays of what follows (the
welfare window `T`, a job's duration, a journey). *Owner:* SET 4, 6. *Damper:* SET-D1,
SET-D2, the finite number of classes, SPN's entry share (D14) and POP's carried births.
*Undamped:* the step itself: a region at a threshold gains a role in one tick; and
regions founded together are held together and released together at tick `Dwell`
(4.4.3), which can open many gates in one tick (open question 4). *Condition:* the
ratchet is bounded at `K - 1` steps whatever else (the model of 14.3 shows two steps
on a four-class ladder). The band and the dwell do not damp the staircase, which is a
saturating reinforcing loop: they act on **flapping** (14.2) and on SET-A2. `Dwell`
at least the effective welfare window `T` of the gated roles (the warning
`dwell_below_window`) is the argument that a gate does not open and close faster than
the incomes it moves are measured; `BandBP` above the noise of the count is the
condition for no flapping (14.2). *Test:* SET-S1. *Status:* hand for the delay
condition (`Dwell` against `T`; no model has a gated role or a window); the bound of
`K - 1` steps is shown by the model (14.3, an assumed growth rate by class).

**SET-A2. The absorbing class.** *Kind:* loop (a trap). *Loop:* households (+) class
(+) a role is open to entry or its ceiling is high (+) what the role makes and the
households' welfare (+) households. Every link has the same direction (fewer
households, a lower class, a closed role, less made, fewer households), so all are `+`
and the polarity is **R** (`spec-stability.md` 3), running downward to the bottom class.
*Delay:* as A1. *Owner:* SET 4, 6. *Damper:* SET-D1, SET-D2, SET-D3 (the class
floor), SET-D4 (a gate bars entry and removes no one), and SUB's floor under market
failure (D15) when the role feeds the household. *Undamped:* a role that the upkeep
needs, gated by a flag that the bottom class lacks, in a region with no floor: the
bottom class absorbs (14.3: with no floor 60 households fall to 25 in 2 150 ticks;
with the floor of class 1 they grow back). *Condition:* every role whose goods are in
the upkeep of a region is open in every class (no `Needs`, or a flag the first class
carries), or the region has a floor at a class that carries it. *Test:* SET-S3.
*Status:* model (14.3: the trap and the floor on an assumed rate by class; the
dynamics of a pack are not shown).

**SET-A3. A gate that no region meets at the start.** *Kind:* hazard. *Loop:* none of
its own. A role with a `Needs` that its groups' regions do not carry at construction
is closed there from the first tick: its households are not replaced like for
like and do not breed, and it runs down by deaths and the ladder (SPN 7.2, POP 3.7.3).
*Owner:* SET 6.4.4. *Damper:* none by design; the warning `needs_unmet_at_start`.
*Condition:* no group of a gated role in a region whose class lacks the flag, or the
author wants the run-down. *Test:* SET-S3 (the base). *Status:* hand.

**SET-A4. A count that gates is not a controller.** *Kind:* constraint. *Loop:* none
of its own; it is the exception of section 7. *Owner:* SET 7. *Damper:* the guards
G1 to G7. *Undamped:* a consumer's own controller that reads the snapshot's class
and sets slots (L121, `spec-policy.md` later). *Condition:* the readers are the rows
of 6.3 and nothing else; no view carries a class; a limit is a count (R2). *Test:*
FR-SET-018 and FR-SET-019 (a static check over the readers), and review for R2.
*Status:* hand.

**SET-A5. A ceiling is first-come.** *Kind:* hazard (a bias, not a loop). *Loop:* none.
When a ceiling binds, the entrants are served in the order of the spawner's plan
(need, then agent ID), so a lower ID is systematically first (`spec-world.md` 10.2,
O2 and O3; N-122). *Owner:* SET 6.4.5 with SPN 6.3. *Damper:* none; the counter
`RoleCeiling` shows when it binds, and the keyed tie of `spec-world.md` 10.1.1 (D87,
proposed there) is the remedy. *Condition:* the unfairness is accepted, or D87 is
adopted. *Test:* S73 (the fairness of a binding cap) with a ceiling. *Status:* hand.

**Dampers of this spec**

| ID | Damper | Mechanism | Home | Status | Cost |
|----|--------|-----------|------|--------|------|
| SET-D1 | Band | a leave threshold `BandBP` below each enter threshold; the class is kept inside its band | SET 3.4.2, 4.3 | specified (a placeholder; not counted: a held class is the class) | the class lags the count by up to the band, which is the point |
| SET-D2 | Dwell | no two changes in less than `Dwell` ticks; a held change is counted | SET 3.4.3, 4.4 | specified (counted by `SettlementHeld`) | the class lags a real change by up to `Dwell` ticks |
| SET-D3 | Class floor | a region's class is never below its `Floor` row; the field `ClassAtFloor` shows it | SET 3.5.3, 4.3.3 | specified (a bound, in the sense of STB 2.3, not a damper) | a region that the count would have demoted keeps what it cannot use |
| SET-D4 | Entry-only gate | a gate bars entry and creation and removes no one; a ceiling is not enforced on what is there | SET 2 (position 9), 6.2 R3 | specified (a rule) | a role can sit above its ceiling, and a closed role runs down slowly rather than at once |

**Scenarios proposed** (for section 8 of the register; REF builds them, with the
overlay of 12.4):

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| SET-S1 | `five` with a growing region (the population module on, a refilling endowment) and the gated `merchant` role | the class sequence of the region is non-decreasing while the count grows; no two changes are less than `Dwell` apart; the number of changes is at most `K - 1`; the role is closed before and open after, in the tick after the change; `SettlementClassChanged` is emitted once per change with the count the review used | SET-A1 |
| SET-S2 | a region whose count is held on a threshold by a noisy supply of entrants (a consumer adds and removes external agents between ticks) | with the default band and dwell the class never changes in the run; with `BandBP` 0 and `Dwell` 0 it changes; the held count of `SettlementHeld` is non-zero when the dwell binds | SET-D1, SET-D2 |
| SET-S3 | `famine` with a gated role that the upkeep needs, the first class lacking the flag, with and without a floor | without a floor the region ends in the bottom class, its role closed and its households falling; with a floor of the class that carries the flag the role is open and the region recovers; the warning `needs_unmet_at_start` is raised at construction | SET-A2, SET-A3 |
| SET-S4 | the module on with one class and no gate, against the module off, over a range of seeds | every `Snapshot` field other than the module's own, the hash, `Modules` and the names table, every event other than the module's own and the hash, names table and `settlement_classes` of the construction events, and the final state of the PCG are equal at every tick (inertness) | SET-A4 |
| SET-S5 | a checkpoint taken during a held change, restored and run on | the class, `Since` and the tick at which the hold ends are those of the uninterrupted world, with the same events | SET-D2 |

SET-S4 and SET-S5 test the library's own procedures against toy economies: they are of
the class "harness test" of `spec-stability.md` 8 and outside the band rule of
FR-STY-098, as S77, S78 and S84 to S90 are.

**Checks of the checklist (9.6 to 9.10).** *Delays against neighbours.* The default
dwell of 30 equals `WelfareWindow` (30), `RoleCooldown` (30) and `MoveReviewTicks`
(30): by design, since the delay argument of SET-A1 is `Dwell >= T`, but it is the place
where cycles form (L19, L119), and SET-S1 is the test. *Tunables.* The rows are in 8;
the registry rows for `spec-calibration.md` 6 are in the handoff: `BandBP` and `Dwell`
are damper strengths (SET-D1, SET-D2, acting on SET-A1 and SET-A2), and `Enter`, the
class floor, the scale, the flags, the limits, `Needs` and `Ceiling` are structural
choices, since each defines the setting's size ladder and no search should move them.

## 14. What was run

`models/set/band.py` and `worked.py` (Docker, standard library only; see the README
of the folder). They are prototypes of the rules of 4 on an assumed household count,
not of the library.

14.1. **The rule** (`band.py rule`). For classes 0, 500 and 2 500 and every count of 0
to 3 000, with a band of 0 and of 1 000 bp, and from every class: the class after a
review is inside its band, `L[c] <= H < E[c + 1]`, in all cases without a floor
(0 states outside); with a floor of 0, 1 and 2 and from every class at or above it
(18 006 reviews for each band) the class is never below the floor, `H < E[c + 1]`
always, and `L[c] <= H` except at the floor itself (0 states outside), where 500
counts (band 0) and 450 counts (band 1 000) of 0 to 3 000 are below `L` of a floor of
class 1 (4.3.2, 4.3.3). At class 1 (`E` 500, `L` 450) a count of 449 gives class 0
and 450 keeps class 1; a region can move two classes at once up and down; a floor
stops the fall. The effective thresholds for `Enter` 5 at 5 000 bp are `E` 3, `L` 2:
a band of one household (4.2.2); `Enter` 3 at 5 000 bp gives `L` 1, below 1.5; `Enter`
5 and 10 at 1 000 bp give a band of one household and `Enter` 11 gives two.

14.2. **The band and the dwell against a noisy count** (`band.py noise`). A count of
500 households that reverts to 500 by a twentieth of its distance each tick and moves
by an integer uniform in `[-s, s]`, so it sits on the threshold `E` = 500 (the worst
case). Class changes in 3 650 ticks, mean over 40 seeds (maximum in brackets):

| `s` (sd of the count) | band 0 dwell 0 | band 0 dwell 30 | band 500 | band 1 000 | band 2 000 |
|-----------------------|---------------:|----------------:|---------:|-----------:|-----------:|
| 2 (0.9%) | 1 330 (1 411) | 115 (116) | 0 | 0 | 0 |
| 5 (2.0%) | 817 (912) | 108 (111) | 1.2 (4) | 0 | 0 |
| 10 (3.9%) | 557 (666) | 100 (108) | 56 (70) | 2.9 (8) | 0 |
| 20 (7.6%) | 477 (531) | 97 (104) | 151 (181) | 60 (78) | 3.9 (10) |

(The columns of a band are with a dwell of 0; with a dwell of 30 the figures are the
same or a little lower: 1.1, 0, 0 for `s` = 5; 44.5, 2.9, 0 for `s` = 10; 78, 47, 3.8
for `s` = 20, as `band.out` shows.) The dwell caps the changes at one in `Dwell` ticks (3 650 / 30
= 121 at most); the band removes the changes whose noise it exceeds; a band that is
small against the noise (500 bp at 3.9%, 1 000 at 7.6%) still flips tens of times, and
the dwell then limits the changes to one in 30 ticks.

14.3. **A staircase and an absorbing class** (`band.py loop`). An assumed net growth of
households per tick by class, in basis points, `(-4, +12, +20, +4)` for classes of
`Enter` 0, 100, 400, 1 600, from 120 households, over 3 650 ticks. Growth alone:
class 1 to 2 at tick 1 005 (400 households), 2 to 3 at tick 1 699 (1 602): the
staircase of SET-A1 has two steps and no dwell binds. A shock to 60 households at
tick 1 500 from class 2: without a floor the region goes to class 0 at once (60 is
below the leave threshold 360 and 90), its rate is negative, and it ends at 25
households after 2 150 ticks (the absorbing class of SET-A2); with a floor of class 1
it falls to class 1 and no lower, grows back at +12 bp a tick and is class 2 again at
tick 3 083. A shock to 380 at tick 1 500, when the region is in class 2 (`E` 400, `L`
360): with the band of 1 000 and the dwell the class stays 2, because 380 is above
the leave threshold 360; with no band and no dwell it falls to class 1 at 380 (below
400) and returns at tick 1 543, two extra changes (a threshold crossed from above and
from below by one shock).

14.4. **What they do not show.** The rates by class are assumed: they are not derived
from a recipe, a wage or a birth schedule, so the model shows the sequence of classes
and the trap, not the dynamics of a pack. The noise is a mean-reverting walk, not the
count of the library, whose births and deaths are a carried remainder with a
fixed spread (`spec-population.md` 3.8). No result is a claim that a default is
stable, or that 1 000 bp is the right band for any setting.

14.5. **The arithmetic of the examples** (`worked.py`): the effective thresholds of
the five regions, the floor, the trace of `centre` with and without the recovery, the
colony's classes and the one-class setting.

## 15. Notes settled

For `AGENTS.md`: each open note tagged `SET` or `ALL`, and the others that name this
spec, is applied, rejected or left open here, with the text to append in the handoff.
No open note carries the code `SET` (the index of `notes.md` has no such row).

| Note | Decision | Where |
|------|----------|-------|
| N-026 One settlement per region | **Already closed** (2026-10-10, REG). Confirmed: this spec is the thin layer the note describes: a class per region from its households, no second level, no split of the market, no field a region reads | 1, 2 (position 1), 3.7 |
| N-104 Region-keyed tables use the one override mechanism | **Applied; the SET part is done; LND stays.** `Modules.Settlements.Regions` is one row per region, `ThresholdBP` by scale and `Floor` by replace, with the errors of `spec-regions.md` 5.3; two rows join the list of 5.1.3 (handoff) | 3.5, 8 |
| N-016 Keep the parameter registry true | **Applied** for this spec: every field has a row (8) and a default or "required", a bound and an owner; no unnamed parameter was found | 8 |
| N-017 Words and names | **Applied**: the terms are settlement class, enter threshold, leave threshold, band, dwell, settlement review, household count (of a region), flag, limit, gate, class floor (qualified, "floor" has six uses already), held change, headroom, ceiling, `ClassID`, `NoClass`. The setting words (`hamlet`, `village`, `town`, `city`, `outpost`, `station`, `hub`, `fair`, `hall`, `merchants`, `dock`, `berths`, `yard`) appear only in the flagged examples of 3.1, 12 and the file listings | 3, 12 |
| N-027 No default macro constant is stable | **Applied**: `BandBP` 1 000 and `Dwell` 30 and every number of the examples and models are placeholders; the models are of the rules on an assumed count; the spec sets no price, wage, rate or share | preamble, 8.2, 14 |
| N-045 Dampers that act silently | **Applied**: the band and the class floor act without an event of their own (a held class is the class; the floor is the field `ClassAtFloor`); the dwell is counted by `SettlementHeld`; the refusals of a gate are counted by the reader (R4, `RoleCeiling`, `BirthsCeiling`, `NoViableRole`) | 6.2, 10 |
| N-046 Register entries owed by unwritten specs | **Applied for SET**: SET-A1 to A5, SET-D1 to D4, SET-S1 to S5 (13); SET is not in the areas of the note, so nothing is removed from it | 13 |
| N-090 What modules that tie an obligation to a region must do with a relocation | **Applied**: SET keeps no account away or committed and registers no `region_bound` refusal; a household that relocates is counted in the new region at the next review | 4.1 |
| N-122 First-come caps | **Applied**: a ceiling that binds is a first-come cap, a sixth place for the list of `spec-world.md` 10.2 (SET-A5, 6.4.5); D87 is the rule for a new spec and is not adopted for it here, since the order of the entrants is SPN's | 6.4.5, 13 |
| N-103 Running several spec writers in parallel | **Followed**: the models ran in Docker with no bytecode and a read-only mount; the host `python3` was not run; the spec creates only its file and `models/set/` | 14 |
| N-126 A new event or `Warn` adds a row | **Applied**: two events and no runtime `Warn`; the rows for `spec-observability.md` 4.1 and the catalogue are in the handoff (10.1) | 10 |
| N-128 A hosted per-agent field is saved once | **Applied**: the module hosts no per-agent field; its state is per region (11) | 11 |
| N-130, N-131 What IND cannot measure; not in version 1 | **Left open, with this added**: a consumer-set class (a slot) is not offered (position 13); the class is read from the snapshot, and `Class` and `ClassTarget` are the fields an indicator needs | 2, 10.5 |
| N-023, N-030, N-084, N-092 | not read by this spec: none names a size class | |

Proposed new notes are in the handoff.

## 16. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-SET-001 | With `Modules.Settlements` nil no region MUST have a class, no review, event or checkpoint section MUST exist, the snapshot fields of 10.5 MUST be zero, and a `Role.Needs` or `Role.Ceiling` MUST be rejected with `ErrModuleDependency` (2, 9.1). |
| FR-SET-002 | The module MUST have at least one class and at most `MaxClasses`; the classes MUST be sorted by `Enter` ascending before a world is built or a hash taken (not by ID: an exception to `spec-config.md` 3.4); the `ClassID` of a class MUST be its index; the first class MUST have `Enter` 0 (rule `first_class_open`) and the `Enter` values MUST be strictly increasing (rule `enter_order`), each violation `ErrOutOfBounds`, and no class `ErrTooMany` (3.2, 9.1). |
| FR-SET-003 | An `Enter` outside `[0, MaxHouseholds]`, a `BandBP` outside `[0, MaxBandBP]`, a `Dwell` outside `[0, MaxWelfareWindow]`, a `ThresholdBP` outside `[1, MaxThresholdBP]` and a limit value outside `[0, MaxQuantity]` MUST be rejected with `ErrOutOfBounds` naming the path (8, 9.1). |
| FR-SET-004 | A class, flag or limit ID that does not match `[a-z][a-z0-9_]*` or exceeds `MaxIDLength` MUST be rejected with `ErrInvalidID`; a class ID, a flag or a limit twice in one class MUST be rejected with `ErrDuplicateID`; more than `MaxFlags` distinct flags or `MaxLimits` distinct limits MUST be rejected with `ErrTooMany` (3.3, 9.1). |
| FR-SET-005 | The flags of the module MUST be the union of the flags the classes list, and the limits the union of their names; a limit a class does not list MUST read as no limit, and a listed value of 0 MUST read as a limit of 0 (3.3.3, 3.3.5). |
| FR-SET-006 | A region row MUST follow `spec-regions.md` 5.3: a row naming an undefined region MUST be rejected with `ErrUnknownRegion`, one naming an outside node with `ErrOutsideNode`, a second row for one region with `ErrDuplicateID`; rows MUST be sorted by region; an absent row MUST give the defaults with no warning (3.5). |
| FR-SET-007 | A `Floor` naming an undefined class MUST be rejected with `ErrUnknownSettlementClass`, and the class of a region MUST never be below its floor, at construction or at any review (3.5.3, 4.3). |
| FR-SET-008 | The effective enter threshold of class `k` of a region MUST be `ceil(Enter_k * ThresholdBP / 10 000)` (0 for the first class) and its leave threshold `floor(E_k * (10 000 - BandBP) / 10 000)`; for `BandBP` above 0 and `E_k` at least 1 the leave threshold MUST be at most `E_k - 1` (4.2). |
| FR-SET-009 | The household count of a region MUST be the sum of the weights of the live agents whose account is in the region, pseudo-agents excluded, a migrant on a journey and an away traveller counted in the region of their account, and MUST equal `RegionRow.Households` of the same boundary (4.1.1). |
| FR-SET-010 | A review MUST give the highest class `c'` reached upward from `c` with `H >= E[c' + 1]` at each step, and, only if none was, the class reached downward with `H < L[c']` at each step, not below the floor; both directions MUST NOT hold at once; and after a review that is not held the class MUST satisfy `H < E[c' + 1]` (if `c' + 1 < K`) and `L[c'] <= H` unless `c'` is the region's floor (4.3.2). |
| FR-SET-011 | A review that finds a change due with `t - Since < Dwell` MUST leave the class and `Since` unchanged and emit `SettlementHeld`; a held change MUST NOT be remembered, and the next review MUST recompute the target from the count it then has (4.4.1). |
| FR-SET-012 | `Since` MUST be the tick of the last change, the dwell MUST run from it and not from the last review, and in a new world `Since` MUST be 0 (4.4.2, 4.4.3, 4.5.1). |
| FR-SET-013 | At construction the class of each region MUST be the highest `k` with `H >= E[k]`, not below the floor, with no band, and one `SettlementClassChanged` with `From` `NoClass` and the cause `founded` MUST be emitted per region in ascending `RegionID` after the `AgentSpawned` events (4.5.1, 5.4.1). |
| FR-SET-014 | The review MUST run once per tick at P7.2a after P7.2, for each region that is not an outside node in ascending `RegionID`, and the class of every region MUST be the same value at every position from P0.1 to P7.2 of a tick (5.1, 5.2.1). |
| FR-SET-015 | A world with `Modules.Settlements` of one class with no flag and no limit, and no `Role.Needs` or `Role.Ceiling`, MUST give every result of the world without the module: every field of the snapshot other than the module's own fields, the configuration hash, `Modules` and the class IDs of the names table, every event other than the module's own and the hash, names table and `settlement_classes` count that `WorldCreated`, `WorldRestored` and the construction line carry, and the final state of the random source, at every tick (3.6.1). |
| FR-SET-016 | The review and every read MUST take no random word and MUST NOT transfer money or goods, add a ledger reason or change any agent (4.7, 2). |
| FR-SET-017 | No call, slot or option MUST set a class or its `Since`; a call between ticks MUST NOT change a class (2 position 13, 5.3.1). |
| FR-SET-018 | No view of `Work`, `Trade` or `Revise`, and no outcome, MUST carry a class, a flag or a limit; the spawner's view MUST carry a role's openness and headroom in the region and no class (6.7). |
| FR-SET-019 | The readers of the class MUST be exactly the positions of the rows of 6.3 that are adopted, checked by a static test over the packages that use the module's reads (6.2 R1, R7). |
| FR-SET-020 | Flags and limits MUST be resolved at construction to dense tables indexed by class, and a read MUST NOT allocate (6.1.1). |
| FR-SET-021 | A `Role.Needs` naming a flag no class carries MUST be rejected with `ErrUnknownFlag`, a `Role.Ceiling` naming a limit no class carries with `ErrUnknownLimit`, and either while the module is off with `ErrModuleDependency` (6.2 R5, 9.1). |
| FR-SET-022 | A role with a `Needs` the region's class does not carry MUST be closed in that region as `Role.Closed` is: no entrant, no switcher, no like-for-like replacement, no birth and no advance of its birth carry; its households MUST stay and MUST NOT be removed because of the gate (6.4.1). |
| FR-SET-023 | A role with a `Ceiling` MUST have, in a region whose class has the limit, the headroom `max(0, limit - households)`; the default spawner MUST NOT admit an entrant or a switcher beyond it and MUST count the refusal; births MUST be capped at it and counted; no household MUST be removed for a ceiling (6.4.2). |
| FR-SET-024 | `SettlementClassChanged` MUST be emitted once per region per change after the change is complete, with `Region`, `From`, `To`, `Households` and `Cause` (`grew` when `To` is above `From`, `shrank` when below, `founded` at construction), MUST contain no map, and MUST NOT be emitted with the module off (10.1). |
| FR-SET-025 | `SettlementHeld` MUST be emitted once per region per tick in which a change was held, with `Region`, `Class`, `Target`, `Households` and `TicksLeft`, MUST NOT be emitted in a tick that emitted `SettlementClassChanged` for the region, and MUST NOT be emitted when nothing was due (10.1). |
| FR-SET-026 | `RegionRow` MUST carry `Class`, `ClassSince`, `ClassTarget` and `ClassAtFloor`; `Class` and `ClassSince` MUST be the saved state, `ClassTarget` the review's target at the current `Households` ignoring the dwell, and `ClassAtFloor` true exactly when the class is the floor and the count alone gives a lower class; a restored world's snapshot MUST equal the original's (10.5). |
| FR-SET-027 | The "world created" line MUST carry `settlement_classes` and `settlements` among the modules, each warning of 9.2 MUST be logged once at `Warn` at construction with `rule` and `path`, a change and a held change MUST be logged at `Debug` only and guarded by `Enabled`, and nothing MUST be logged at `Info` or `Warn` inside the tick (10.2). |
| FR-SET-028 | The warnings of 9.2 MUST be raised as described, MUST NOT fail validation, and MUST NOT be raised in a tick. |
| FR-SET-029 | The checkpoint MUST hold, with the module on, per region that is not an outside node in ascending `RegionID`, the class and `Since`, and nothing else of this spec's state, and MUST have no section with the module off; the decoder MUST reject a class below the number of classes or below the floor and a `Since` above the tick less 1; a world restored between ticks MUST run its next review as the uninterrupted one does (11). |
| FR-SET-030 | The configuration hash MUST cover `Classes` (`ID`, `Enter`, `Flags`, `Limits`), `BandBP`, `Dwell` and `Regions` in canonical order (classes by `Enter`, a class's `Flags` by name and its `Limits` by `Name`, rows by region) and no `Label`, so two configurations of equal content have one hash; and a configuration read from a file and the same built in Go MUST be equal, the file writing `[modules.settlements]`, `[modules.settlements.class.<id>]`, `[modules.settlements.class.<id>.limits]` and `[modules.settlements.region.<id>]`; an unknown key MUST be rejected with `ErrUnknownKey` (3.1, 11). |
| FR-SET-031 | The two metrics of 10.4 MUST be derivable from `SettlementClassChanged` alone, and the construction events MUST give the gauge its first value (10.4). |
| FR-SET-032 | Every product of the thresholds MUST use checked arithmetic and the largest case allowed by the bounds MUST NOT overflow (4.2.1). |
| FR-SET-033 | For a given seed, configuration and sequence of calls the classes, the events and their order MUST be identical across runs and platforms, and no iteration over a map MUST influence a result (architecture.md 7.4, 7.5). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-SET-001 | The review SHOULD cost `O(R)` per tick with a counter per region and `O(A)` with a scan, and the module MUST add no term that grows with households beyond a scan of the agents once per tick (5.6, 4.1.3). Counts, not timings (N-015). |
| NFR-SET-002 | With the no-op observer a `Step` of a world with the module on MUST allocate no more than a world with it off beyond the module's tables, and no event MUST be constructed; an `AllocsPerRun` test enforces it (observability-and-logging.md 8.2). |
| NFR-SET-003 | The thresholds and the review MUST live in `internal/settlement` over primitive slices, import the standard library only, and know no region, role, flag name or class name (architecture.md 2.9, 9.1). |
| NFR-SET-004 | The validator and the review MUST have fuzz tests (testing.md 7.4) asserting that they never panic for any thresholds, band, scale, floor, class or count, and that the review's result is in `[floor, K)`, satisfies `H < E[c' + 1]` and satisfies `L[c'] <= H` unless it is the floor (4.3.2). |

## 17. Open questions

1. **A count that bounds a number.** Section 7 asks WLD, ADR 0001, STB and the
   roadmap to carve the class out of the rule (7.4). If they refuse a count that
   bounds a number, the module keeps flags and drops limits (`ClassLimit`, H2 and
   H3), and "how many merchants" becomes a constant of the role; the flags-only
   fallback still needs the carve-out, since a flag reads the same count. The
   recommendation is the carve-out: a count of places is the thing a settlement's rank
   gives. Next step: the owners, with the supervisor.
2. **A gate the world enforces.** `Needs` and `Ceiling` bind the default spawner and
   births only, as `Closed` does (6.4.1); an external agent or a consumer's spawner
   may ignore them. A world-enforced gate would add a refusal reason to `Enter` and
   to `AddAgent`. Not taken: the first consumer's spawner is the default. Next step:
   SPN and INT if a consumer asks.
3. **A class that is never reviewed.** A setting that wants sizes fixed at
   construction can set a `Dwell` of `MaxWelfareWindow` or write one class per region
   by the floors; a flag `Static` would say it. Not added: no case.
4. **The first release of regions founded together.** Every region is held from
   tick 0 to tick `Dwell` and then released together (4.4.3). A start of
   `-bucket(RegionID, Dwell)` with the rule bucket of `spec-world.md` 9.1.1 would
   spread the releases with no draw. Not taken: five regions need no spread, and the
   bucket is not claimed for fewer than a thousand.
5. **The viability check and the floor.** A role that the steady state needs and a
   gate that the first class lacks is the author's to check (SET-A3). VIA could read
   the class at construction and the floor of each region and fail `role_cannot_work`
   for such a group. Next step: `spec-viability.md` 4 and 10 if a pack hits it.
6. **Which households count.** All of them, whatever the role. A setting where the
   class should count only the households of the resident roles (not a defending force, not a
   traveller's crew) needs a flag on the role. Not added.
7. **A class from another measure.** The count of agents, of money or of the
   output would be a different, and worse, input: each is a measured aggregate with
   a price in it. Households are the one count that the world keeps without a price.
8. **One event or two.** `SettlementHeld` repeats each tick of a hold. An event at the
   start and one at the end of a hold needs a state of the hold (a tick) and
   would change the checkpoint. Not taken; the aggregate is bounded by regions.
9. **A class in the strategy's view.** A setting whose households should behave by size
   can put it in a consumer's own strategy from the snapshot. The library does not
   (6.7). Next step: the first consumer.
10. **The mountain region and the one scale.** `ThresholdBP` scales every threshold
    by one factor. A region where only the town threshold differs needs a per-class
    row, which is the second key of a table that REG 5.1 does not have. Not added.

## 18. Where this lives in the code

| Concern | Home |
|---------|------|
| `ClassID`, `NoClass`, `Settlements`, `SettlementClass`, `ClassLimit`, `RegionSettlement`, the construction of the dense tables, the review's call at P7.2a, `regionHas` and `regionLimit` | root package, `settlements.go` |
| The effective thresholds and the review over primitive slices | `internal/settlement` (standard library only, converted inline at the call, architecture.md 2.9) |
| `Role.Needs` and `Role.Ceiling` | root package, with the role (`spec-agents.md` 3.3); read by `spec-spawner.md` and `spec-population.md` code |
| Events `SettlementClassChanged`, `SettlementHeld`; the cause constants `founded`, `grew`, `shrank` | root package, `event.go`; `internal/logkeys` holds the key strings only |
| The snapshot fields | root package, `snapshot.go` |
| The bounds `MaxClasses`, `MaxFlags`, `MaxLimits`, `MaxBandBP`, `MaxThresholdBP` | root package, `bounds.go` |
| The defaults `BandBP` 1 000 and `Dwell` 30 | root package, `defaults.go` |
| Validation, the sentinels `ErrUnknownSettlementClass`, `ErrUnknownFlag`, `ErrUnknownLimit` | root package, `validate.go`, `errors.go` |
| The checkpoint section | `checkpoint.go` with the other module sections |
