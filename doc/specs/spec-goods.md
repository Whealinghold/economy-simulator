# Spec: Goods, the attributes of a commodity

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-GDS-NNN` and
`NFR-GDS-NNN`, which tests cite (testing.md section 4). They are in section 9
(FR-GDS-001 to 023, NFR-GDS-001; no ID is added this revision, FR-GDS-002 and
FR-GDS-003 are reworded).

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs nothing. It is the lowest level of the tier 1 dependency picture.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

A **commodity** is a kind of good that agents make, hold, trade and consume. This
spec defines the *attributes* of a commodity, all of them data: what a unit of
it is, how big it is, whether it spoils, whether it wears out, how it is grouped
for measurement, whether it is a raw good, and its reference price. It covers:

- the unit and the choice of it (section 2);
- mass, bulk and loads, and the chargeable size used to price transport and storage (section 3);
- spoilage by shelf life (section 4);
- wear by durability (section 5);
- classes, raw goods and the reference price (section 6);
- what GDS deliberately does not model (section 7).

What a commodity *does* belongs elsewhere: how recipes use it
(`spec-production.md`), how lots are held, aged and conserved (`spec-ledger.md`),
how it travels (`spec-transport.md`), how it is priced (`spec-clearing.md`,
`spec-beliefs.md`). This spec supplies the fields those specs read.

## 2. The unit

2.1. A commodity has one **unit**: the whole item or measure that a `Quantity`
counts (architecture.md 3.5). Quantities are whole numbers of the unit. There are
no fractional quantities anywhere in the library, so a commodity is as
**divisible** as its unit is small: the author chooses the unit, and that choice
is the whole of "divisibility". A cart is one unit; grain is a small measure of
grain; a service is whatever slice of it can be bought.

2.2. The unit is chosen against two constraints that pull in opposite
directions. Both are checked by validation.

- **Upkeep is a whole number of units.** A household's upkeep of a commodity per
  tick is a whole number of units, at least 1 for each good it needs
  (`spec-scale.md` open question 6). A unit too coarse makes the smallest need
  larger than a household uses.
- **The price clears the floor.** The reference price per unit is at least the
  resolution floor for the belief step `BeliefStep` (`spec-currency.md` 3.1). A unit
  too fine makes the price too small for a belief step to move it.

A finer *minor unit of money* relaxes the second constraint without touching the
first (`spec-currency.md` 3.2). Section 2.3 shows the arithmetic.

2.3. Worked example. The reference basket is `B_r = 1 000` minor units
(`spec-currency.md` 5.2, its default). The default `BeliefStep` of 200 basis
points gives a resolution floor of 50; the example takes a stricter 1% step
(`BeliefStep` 100), whose floor is 100 (`spec-currency.md` section 3 table), so
that the floor binds. A household needs 3 units of a staple and 1 unit of a fuel
each tick.

| Commodity | Upkeep (units) | Reference price | Cost per tick |
|-----------|---------------:|----------------:|--------------:|
| staple | 3 | 300 | 900 |
| fuel | 1 | 100 | 100 |
| **Total** | | | **1 000** (equals `B_r`) |

Both prices are at or above the floor of 100. Choosing a finer unit of fuel,
10 units a tick at price 10, would make the same cost per tick but a price of
10, below the floor, and validation rejects it. With these numbers a basket of
more than ten distinct goods cannot clear the floor at `B_r = 1 000` (twenty at
the default step); a chain with a longer upkeep list needs a larger `B_r` or a
lower belief step (`spec-currency.md` 5.7).

2.4. Money is not a commodity. There is one currency and one unit of account
(`spec-currency.md` 2.1), and no commodity stands for it. Coins as goods are a
stated non-goal there (`spec-currency.md` section 6).

2.5. The unit has an optional display name, `UnitLabel`, free text that is
presentation only (section 6.5): a library with a setting says "bushel" or
"crate" there, and the core never reads it.

## 3. Mass, bulk and loads

Moving or storing goods is limited by two different things: how much they weigh
and how much room they take. A load of ingots is limited by what the cart can
bear; a load of fleeces by what it can hold. Transport and storage costs depend
on whichever limit is reached first, so a commodity has two attributes, not one.

3.1. A commodity has a **mass** and a **bulk**: two non-negative integers, the
amount of a container's load-bearing capacity and the amount of its room that one
unit uses, in abstract **mass units** and **bulk units**. Both default to 1. 0
means the commodity uses none of that dimension, so a good with no physical form
(a service, an item of information) has 0 for both. The attribute is called
`Mass`, not `Weight`, because `Weight` is the number of households an agent
stands for (`spec-scale.md` section 2).

3.2. A **load** is a pair: the sum over the goods of quantity times mass, and the
sum of quantity times bulk. A **container** is anything with a capacity: a
carrier's vehicle (`spec-transport.md`), a store (`spec-storage.md`). Its
capacity is also a pair, a **mass capacity** and a **bulk capacity**, owned by the
spec of that container. A load **fits** a container if and only if both sums are
within their capacity. Whichever is reached first is the **binding dimension**,
and it differs by cargo. The sums are computed with checked arithmetic
(architecture.md 3.5).

3.3. **Chargeable size.** Transport and storage are priced as ordinary markets,
by a price per unit (`spec-carriage.md` section 2.2), and that needs one number
per good. The **chargeable size** of one unit of a commodity is

```
charge = max(mass, ceil(bulk * BulkFactor / 10_000))
```

`BulkFactor` is a world parameter in basis points, default 10 000: at the default
one bulk unit is charged as one mass unit. A larger factor charges bulk more
heavily, as freight does when it bills the greater of the actual weight and the
space a parcel takes up. The chargeable size of a quantity is the quantity times
`charge`, checked. It is used *only* for pricing. Whether a load fits a container
is always decided by the pair of 3.2.

3.4. **Chargeable capacity.** A container with a mass capacity `Cm` and a bulk
capacity `Cb` carries at least `min(Cm, floor(Cb * BulkFactor / 10_000))`
chargeable units of any single commodity: a load of one commodity within that
number always fits. This is the capacity over which a carrier spreads its
reservation fee (`spec-carriage.md` section 2.2) and a store its charge. The
bound is exact when `Cm` equals `Cb * BulkFactor / 10_000`, a vehicle that is
balanced for the convention, and conservative otherwise.

3.5. **Mixed loads.** A load of several commodities can fit even when the sum of
their chargeable sizes is above the chargeable capacity, because goods that are
heavy and goods that are bulky use different limits. Pricing each good by its
chargeable size and adding them therefore *over-charges* a well-mixed cargo. That
is a stated simplification (open question 1): it keeps carriage and storage a
per-unit market, and it errs towards dearer transport, not cheaper.

3.6. Mass and bulk are properties of a unit, so a cohort of agent weight `w`
holds `w` times as much of each (`spec-scale.md` 4.2). Loads and capacities are
checked against the bounds of section 8.

3.7. Worked example. A cart has a mass capacity `Cm = 100` and a bulk capacity
`Cb = 40`; `BulkFactor = 25 000`, so one bulk unit is charged as 2.5 mass units.
Two commodities:

| Commodity | Mass | Bulk | Charge per unit |
|-----------|-----:|-----:|-----------------|
| ingot | 5 | 1 | `max(5, ceil(1 * 25 000 / 10 000)) = max(5, 3) = 5` |
| fleece | 1 | 4 | `max(1, ceil(4 * 25 000 / 10 000)) = max(1, 10) = 10` |

The chargeable capacity is `min(100, floor(40 * 25 000 / 10 000)) = min(100,
100) = 100`, a balanced cart.

- Ingots only: 20 fit (mass 100, bulk 20); a 21st breaks mass (105 > 100). The
  chargeable size of 20 is `20 * 5 = 100`, the chargeable capacity. Mass is
  binding.
- Fleeces only: 10 fit (mass 10, bulk 40); an 11th breaks bulk (44 > 40). The
  chargeable size of 10 is `10 * 10 = 100`. Bulk is binding.
- A mix: 16 ingots and 6 fleeces fit (mass `5 * 16 + 6 = 86 <= 100`, bulk
  `16 + 4 * 6 = 40 <= 40`). Their chargeable size is `16 * 5 + 6 * 10 = 140`, 40%
  above the chargeable capacity, although the cart carries them. The per-unit
  price would charge for 140, so this mix is over-charged (3.5).

## 4. Spoilage and shelf life

4.1. A commodity has a **shelf life**: the number of ticks a lot of it lasts. 0,
the default, means it never spoils. A lot is a quantity made or bought together
(`spec-ledger.md`), and has an **age** in ticks that starts at 0 when it is
created.

4.2. Aging is deterministic and uses no random numbers. At the end of every tick
the age of each lot of a perishable commodity goes up by 1. When the age reaches
the shelf life, the whole lot **spoils**: it is removed, as a sink of goods with
the reason `spoilage` and an event (`spec-ledger.md`). A lot is present at the
end of at most `ShelfLife` ticks counting the tick it was created in, so:

| Shelf life | A lot made in tick `t` is usable in ticks |
|-----------:|-------------------------------------------|
| 1 | `t` only; it spoils at the end of `t`, as a service that is consumed or lost does |
| 3 | `t`, `t + 1`, `t + 2`; it spoils at the end of `t + 2` |

4.3. When lots of the same commodity are sold or used, the oldest goes first.
That is a rule of the ledger (`spec-ledger.md`), stated here because it is what
makes shelf life a pressure to sell: a seller holding fresh and old lots offers
the old ones first.

4.4. Age continues while goods are in transit or in an estate listing
(`spec-estates.md` 2.1: perishable goods spoil as usual; `spec-transport.md`
for goods on a route). Nothing pauses a clock.

4.5. This is a cliff, not a decay: a lot is whole and then it is gone. A
gradual decay (a share of the stock lost each tick) is smoother and needs no
lots, but it spreads loss over every holder at every tick and hides who held the
goods that spoiled. The cliff is chosen for the ledger's lots and for a sink
event that says what spoiled and where. Open question 2 records the alternative.

## 5. Wear and durability

5.1. A commodity has a **durability**: the number of wear units one unit of it
has when new. 0, the default, means it does not wear. A commodity with a
durability above 0 is a **durable**.

5.2. A durable is held as items of quantity one, each carrying its remaining
durability (`spec-skill-barriers.md` section 2.7: "the ledger holds each tool as
a lot of quantity one, carrying its remaining durability"). Remaining durability
is between 1 and `Durability`. A new item has `Durability`. When use reduces it to
0, the item is destroyed, as a sink of goods with the reason `wear` and an event.

5.3. Wear happens by use. What counts as one use, and how much wear it causes
(`wear_per_use`, the field `WearPerUse` of a recipe's tool), belongs to the recipe
that requires the durable (`spec-production.md` 4.4, which also owns `Requires`:
wear falls on distinct items, most worn first, and is not random), because the same tool may wear faster in one recipe than
in another. This spec owns the *capacity* to wear (`Durability`); the production
spec owns the *rate* and the `Requires` field of a recipe. This settles which
spec owns which half of the field pair named in `spec-skill-barriers.md` 2.1.

5.4. A **tool** is a durable that some recipe requires. It is not a separate kind
of commodity: it is a commodity that is required by a recipe, and it is a durable
because it wears. A durable that no recipe requires never wears, and validation
warns about it (7.2).

5.5. Wear and spoilage are independent. A commodity may have both, and each runs
on its own counter: age by time, durability by use.

5.6. Durability is a use count, like the tool durability of
`spec-skill-barriers.md` section 2: a tool of durability 10 lasts 10 uses at
`wear_per_use` 1, with the same expected life as the original's 10% chance of
breaking, and with no randomness.

## 6. Classes, raw goods and the reference price

### 6.1 Classes

A commodity has a set of **classes**: short tags such as `food` or `fuel` or
`tool`, chosen by the author. They exist for measurement: the generic
aggregators of `spec-indicators.md` group by class (a price index per class, a
value added per class), and a consumer-defined measure can too. Rules:

- A class is a string matching the ID pattern of `spec-config.md` 3.3, and a
  commodity has at most 8, with no duplicates.
- Classes are sorted into canonical order with the rest of the configuration
  (`spec-config.md` 3.4).
- **No code may name a class.** Classes carry no behaviour in the engine; they
  only group for measurement and for rules a pack writes in data. The same rule
  as for identifiers (architecture.md 9.1) applies: branching on `"food"` is a
  defect. A module that needs groups (for example demand by class in
  `spec-demand.md`) takes class names as configuration.

### 6.2 Raw goods

A commodity may be marked **raw**: it can be supplied without being made from
other commodities. This is what `spec-config.md` 7.5 calls *sourced*. A raw
commodity is gathered, harvested or imported: its source is a recipe with no
inputs, a regional endowment (`spec-resources.md`), a faucet or an import
(`spec-external-sector.md`). Marking a commodity raw is what lets a chain
without a source for it validate; the specs that supply real sources add a check
that a raw commodity has one (open question 5). Until then the marker stands
alone. A commodity may be raw and also made by a recipe.

### 6.3 The reference price

Every commodity has a **reference price**: a price per unit, in minor units, that
the model uses where there is no recent market price. It is *required* (it has no
default, `spec-config.md` 5.3), and it is a calibration anchor, not a live value:

- the default **seed price**: the starting market price before the first trade
  (`spec-clearing.md` 5.4) and the default starting centre of every initial
  agent's price belief (`spec-beliefs.md` 8.1, 8.2, FR-BEL-022). A warm start
  (`spec-viability.md` 8.1) may replace the seed price with its own table; the
  reference price is then still the anchor below;
- the anchor of the **reference band** that bounds a belief: the band runs from
  `RefBandLow` to `RefBandHigh` basis points of the effective reference price
  (`spec-beliefs.md` 3.2, FR-BEL-004), so it is not symmetric about it;
- the basis of the reference basket `B_r` (`spec-currency.md` section 5);
- the price an estate lists goods at (`spec-estates.md` 2.1) and the last
  fallback for a tool listing (`spec-skill-barriers.md` section 2.7).

A region may override the price of a commodity for its own markets. The override
field belongs to `spec-regions.md`. The **effective** reference price of a
commodity in a region is the region's override if it has one, else the
commodity's `ReferencePrice`. This settles `spec-clearing.md` open question 6: the
commodity owns the default, the region owns the override.

### 6.4 Bounds on a price

The reference price is between 1 and `MaxPrice` (`spec-config.md` 4.1). The
cheapest tradable commodity's effective reference price in every region is at
least the resolution floor (`spec-currency.md` 3.1). Validation enforces both.

**Tradable** is defined by `spec-agents.md` 3.4: a commodity is tradable if it is
in the tradable set of at least one role, where the set of a role is its upkeep
commodities, the inputs, required tools and non-discard outputs of its recipes,
and its `Trades` list. A commodity in no role's set (for example one that exists
only as a discard output) is never traded, and the floor does not apply to it.
This spec does not define the set; it only uses the word as AGT does (N-048).

### 6.5 Presentation

`Label` (a display name) and `UnitLabel` (2.5) are free text of at most 64 bytes.
They are presentation only: they change no result, are not part of the
configuration hash (`spec-config.md` 8.3), and are not used by the core.

## 7. Not modelled

7.1. **Quality grades.** Two grades of grain are two commodities. This keeps
every unit of a commodity interchangeable, so one market clears one price. The
only per-item state is the remaining durability of a durable (5.2). A `Grade`
attribute, if a case ever needs one, is a later change to this spec and a new
commodity field, not a change to the unit.

7.2. **Warnings.** A durable that no recipe requires never wears, which is
legal but probably a mistake. Validation reports it as a warning
(`spec-config.md` 7.2, FR-GDS-023).

7.3. **Labour and land** are not commodities in version 1 (overview, known
gaps).

7.4. **Money** is not a commodity (2.4).

7.5. **A commodity that changes.** The attributes of a commodity are fixed for a
run. A technology or season that changes a shelf life, a mass or a bulk would be a
different design (`spec-technology.md`, `spec-calendar.md`).

## 8. Parameters

All are fields of a commodity in the configuration, in the file as
`[commodity.<id>]` (`spec-config.md` 6.2). The defaults are those of
`DefaultCommodity()`.

| Scope | Field | Meaning | Default |
|-------|-------|---------|---------|
| Commodity | `Mass` | load-bearing capacity one unit uses, in mass units (section 3); 0 uses none | 1 |
| Commodity | `Bulk` | room one unit takes up, in bulk units (section 3); 0 takes none | 1 |
| World | `BulkFactor` | how heavily bulk counts in a chargeable size, in basis points (section 3.3) | 10 000 |
| Commodity | `ShelfLife` | ticks a lot lasts before it spoils; 0 never spoils (section 4) | 0 |
| Commodity | `Durability` | wear units of a new unit; 0 does not wear (section 5) | 0 |
| Commodity | `Classes` | set of class tags for measurement (section 6.1) | none |
| Commodity | `Raw` | can be supplied without being made from other commodities (section 6.2) | false |
| Commodity | `ReferencePrice` | price per unit in minor units; required, no default (section 6.3) | none |
| Commodity | `Label` | display name, presentation only (section 6.5) | empty |
| Commodity | `UnitLabel` | display name of the unit, presentation only (section 6.5) | empty |

Bounds, which validation enforces with `ErrOutOfBounds`:

- `Mass` and `Bulk` each from 0 to 1 000 000, provisional. The largest agent
  weight (`MaxWeight`) times a mass or bulk times a quantity still has to fit
  (`spec-config.md` 4.2).
- `BulkFactor` from 1 to `MaxBasisPoints`.
- `ShelfLife` from 0 to 10 000 000 ticks, provisional.
- `Durability` from 0 to `MaxQuantity`.
- `ReferencePrice` from 1 to `MaxPrice`.
- at most 8 `Classes`, each matching the ID pattern.

## 9. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-GDS-001 | A commodity MUST have a unit, and every quantity of it MUST be a whole number of that unit. |
| FR-GDS-002 | The cheapest tradable commodity's effective reference price in every region MUST be at least the resolution floor, or the configuration MUST be rejected; a commodity is tradable if it is in the tradable set of at least one role (`spec-agents.md` 3.4), and a commodity in no role's set MUST NOT count. |
| FR-GDS-003 | The upkeep of a household for each commodity it needs MUST be at least 1 unit (the effective upkeep of `spec-agents.md` 5.1, 5.2, per household), or the configuration MUST be rejected. (Owned jointly with `spec-agents.md`.) |
| FR-GDS-004 | `Mass` and `Bulk` MUST each be an integer from 0 to their bound, defaulting to 1. |
| FR-GDS-005 | The load of a set of goods MUST be the pair of the sum of quantity times `Mass` and the sum of quantity times `Bulk`, computed with checked arithmetic, and a load MUST fit a container if and only if both sums are within the container's two capacities. |
| FR-GDS-006 | The chargeable size of one unit of a commodity MUST be the greater of its `Mass` and `Bulk` times `BulkFactor` divided by 10 000 (rounded up), and `BulkFactor` MUST default to 10 000 and lie between 1 and `MaxBasisPoints`. |
| FR-GDS-007 | The chargeable capacity of a container MUST be the smaller of its mass capacity and its bulk capacity times `BulkFactor` divided by 10 000 rounded down, and any load of a single commodity whose chargeable size is within it MUST fit. |
| FR-GDS-008 | `ShelfLife` MUST be an integer from 0 to its bound, where 0 means the commodity never spoils. |
| FR-GDS-009 | The age of every lot of a perishable commodity MUST increase by 1 at the end of each tick, and a lot MUST spoil, whole, at the end of the tick in which its age reaches the shelf life. |
| FR-GDS-010 | Spoilage MUST NOT use the random source, and MUST be a sink of goods with the reason `spoilage` and an event. |
| FR-GDS-011 | When lots of one commodity are sold or used, the oldest MUST go first. |
| FR-GDS-012 | A lot's age MUST continue to increase while the goods are in transit or in an estate listing. |
| FR-GDS-013 | `Durability` MUST be an integer from 0 to its bound, where 0 means the commodity does not wear. |
| FR-GDS-014 | A unit of a commodity with a durability above 0 MUST be held as an item of quantity one with a remaining durability between 1 and `Durability`, and a new item MUST have `Durability`. |
| FR-GDS-015 | An item whose remaining durability reaches 0 MUST be destroyed, as a sink of goods with the reason `wear` and an event. |
| FR-GDS-016 | A commodity MAY have at most 8 classes, each matching the ID pattern, with no duplicates, and classes MUST be in canonical order. |
| FR-GDS-017 | No code outside the aggregators and the data a pack supplies MUST name a class, and the engine's behaviour MUST NOT depend on a commodity's classes. |
| FR-GDS-018 | A commodity marked `Raw` MUST count as sourced in the producibility check of `spec-config.md` 7.5. |
| FR-GDS-019 | A commodity's reference price MUST be between 1 and `MaxPrice`, and a configuration that omits it MUST be rejected. |
| FR-GDS-020 | The effective reference price of a commodity in a region MUST be the region's override if it has one, else the commodity's `ReferencePrice`. |
| FR-GDS-021 | `Label` and `UnitLabel` MUST be at most 64 bytes, MUST NOT change any result, and MUST NOT be part of the configuration hash. |
| FR-GDS-022 | The attributes of a commodity MUST NOT change during a run. |
| FR-GDS-023 | Validation MUST warn about a durable that no recipe requires, and MUST NOT fail the configuration for it. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-GDS-001 | Reading the attributes of a commodity during a tick MUST NOT allocate. |

## 10. Entries for the stability register

Merged into `spec-stability.md` section 4 (STB-L25 spoilage and stockpiling, which
also records the whole-lot sawtooth of 4.5; STB-L26 durables and the replacement
cycle). The register is now their home; the text below is the source, kept for the
record.

Original entries:

- **Spoilage and stockpiling.** Shelf life makes holding a perishable costly, which
  limits the stock an agent can build against a shortage. A short shelf life on a
  staple is a destabiliser: a harvest glut spoils and the next shortage is
  deeper. Damper: the staple's shelf life is a calibration choice checked by the
  viability and acceptance tests, and the oldest-first rule makes sellers clear
  old stock before it spoils.
- **Durables and cycles.** A tool that wears slowly is capital: demand for
  replacements is low and lumpy until the stock ages together, then spikes (an
  echo of the replacement cycle). Damper: different ages across holders, set
  by the starting inventories in the chain's data
  (`spec-skill-barriers.md` section 2.8).

## 11. Open questions

1. **Pricing a mixed load.** Section 3.5 over-charges well-mixed cargo because
   each good is charged by its own chargeable size. A price based on the binding
   fill of the combined load would be exact for one vehicle, but it is not a
   per-unit price and so does not fit the carriage market. Revisit with
   `spec-transport.md`, and measure how much over-charging the reference chain
   actually has. A single world-level `BulkFactor` also means every vehicle type
   is priced with one convention; a per-mode factor is the next step if one is
   needed.
2. **Cliff or decay for spoilage.** A lot spoils whole at a fixed age (4.5), and
   the ledger implements it with lots stored by born tick (`spec-ledger.md` 5.1
   and 5.3). The alternative is a share of the stock lost each tick; revisit only
   if the sawtooth of whole-lot spoilage shows up in the stability tests.
3. **Inventory capacity in version 1.** Answered by `spec-agents.md` 8 (N-002): no
   inventory capacity in version 1. The original bounds an agent's whole
   inventory (`max_size`) and makes it discard goods when full; here a discard is
   a strategy decision executed as a move to the `disposal` boundary. `Mass` and
   `Bulk` stay for transport and storage (`spec-storage.md` is "later" in the
   vision).
4. **Goods that cannot or should not be traded.** Answered by
   `spec-production.md` 7.5 (N-003): there is no `Tradable` flag. A waste output
   is marked `Discard` (created and disposed of in one step, never held or
   offered), a byproduct is an output with `CostShare` 0, and the `disposal`
   reason is the goods sink. Bound tools are handled by the skill-barriers rule
   (`spec-skill-barriers.md` 2.7), not by an attribute here. If a case appears
   where a good must be forbidden from trade by rule, `spec-production.md` 7.4
   says to add `Tradable` here and a reject reason to `spec-clearing.md` 3.2.
5. **A source for a raw good.** Check that a commodity marked `Raw` has a real
   source once resources, the external sector or faucets exist, so the marker
   cannot hide a missing one (6.2).
6. **A commodity that is both a tool and a consumable.** Allowed (5.5). The ledger
   holds such items as lots keyed by born tick and remaining durability, so use
   and age are independent (`spec-ledger.md` 5.1, 5.4); whether any chain needs
   the combination is untested.
7. **Quality grades.** Separate commodities for now (7.1). Revisit if a pack needs
   hundreds of near-duplicates.

## 12. Where this lives in the code

| Concern | Home |
|---------|------|
| `Commodity`, `CommodityID`, `DefaultCommodity()` | root package, `commodity.go` |
| The fields of section 8 and their bounds | root package, `commodity.go`; bounds in `bounds.go` (`spec-config.md` 4.1) |
| Validation of a commodity, and the warning of 7.2 | root package, `validate.go` |
| Aging of lots, spoilage and wear | `internal/ledger` (`spec-ledger.md`) |
| `Requires` and `WearPerUse` | `spec-production.md` 4.4, `internal/production` |
| The tradable set that defines "tradable" | `spec-agents.md` 3.4 |
| The class aggregators | `spec-indicators.md` |
