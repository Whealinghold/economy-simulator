# Spec: Currency, the reference basket and denominations

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CUR-NNN` and
`NFR-CUR-NNN`, which tests cite (testing.md section 4). They are in section 8
(FR-CUR-001 to 023, NFR-CUR-001).

Layer: core for the minor unit, the resolution floor and the reference basket; the denomination formatter is optional, and the RPG ladder is a pack ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

Prices, wages and balances in the library are single `int64` counts of one
**minor unit** (architecture.md 3.5). The library is agnostic about what that
unit is called, how it is grouped and what it is worth in the setting: it could
be pence, credits, grain or a game's coins. This spec covers:

- how large the minor unit must be so that integer arithmetic does not freeze
  the simulation (section 3);
- an optional **denomination** formatter that turns a count of minor units into
  text for a setting's coins, and a preset for a fantasy setting (section 4);
- how a reference household basket anchors the price level, per region
  (section 5).

Section 2 holds the principles shared by every spec in this directory. Welfare,
wages, skill barriers, estates and carriage are in the sibling specs listed in
the README. What is out of scope here is in section 6. Money stays a single
unit of account.

## 2. Principles

2.1. There is one currency and one unit of account: the minor unit. The core
knows no names for it. A **denomination** is a name for a multiple of the
minor unit and is a display concern only.

2.2. Denominations never appear in the core module's types, events, snapshots,
logs or configuration. Those carry `Money`, `Price` and `Quantity` as integers.
Nothing in the core imports the formatter of section 4, and removing it changes
no simulation result. This keeps architecture.md 3.5 true: how many minor units
make one displayed unit is the consumer's concern.

2.3. All derivations use integer arithmetic with checked overflow and a stated
rounding rule (truncation toward zero unless stated). Ratios are basis points
(1/10 000).

2.4. A **tick** is the unit of simulated time. A **year** is `TicksPerYear`
ticks, the period in which annual rates (a required return, a yearly income)
are stated. `TicksPerYear` is derived from the world's calendar
(`spec-calendar.md` 3.2): the days in a year times the ticks in a day, divided
by the days in a tick. At the Earth default it is 365. A setting with a
different calendar, or a tick longer or shorter than a day, sets the calendar
and changes no code. `TicksPerYear` is not a parameter of its own; the calendar
fields are owned by `spec-calendar.md`. In this and the sibling specs a "day" is
a tick, and "daily" means per tick; the words are kept because the sources and
worked examples are in days. A calendar day is a different unit
(`spec-calendar.md` 2.5).

Quantities stated per tick (a household's upkeep, the reference basket `B_r` of
section 5) are data for the chosen tick length: changing the calendar does not
rescale them. Seasons and harvests within a year are `spec-calendar.md`, which
also defines `TicksPerYear`.

2.5. Nothing in this spec is branched on by code. Skill levels, wage targets
and basket contents are configuration data (architecture.md 9.1).

2.6. Wages are not configured. A wage is what the market pays: the net revenue
of a self-employed role, or the fee a merchant pays a carrier. The formulas in
`spec-welfare-and-wages.md` section 3 are used to calibrate a chain and to predict behaviour, and the
simulation measures the result (`spec-welfare-and-wages.md` section 4.3).

## 3. The minor unit and price resolution

Price beliefs move by basis-point steps: `step = price * bp / 10_000`. Integer
division truncates, so a price that is too small gives a step of zero and the
belief freezes. The step is the **belief step**, the world parameter
`BeliefStep`, in basis points of the belief's centre (owner `spec-beliefs.md`
5.3 and 11; default 200, bound 1 to 2 500). For a step of `s` basis points to
move a price by at least one minor unit, the price must satisfy:

```
price >= ceil(10_000 / s)
```

| Belief step | Smallest price that still moves |
|-------------|---------------------------------|
| 5% (500 bp)          | 20                              |
| 2% (200 bp, the default) | 50                          |
| 1% (100 bp)          | 100                             |
| 0.1% (10 bp)         | 1 000                           |

3.1. The **resolution floor** is `ceil(10_000 / BeliefStep)` minor units; the
default `BeliefStep` of 200 gives a floor of 50. Configuration validation
(architecture.md 8.5) **MUST** reject a chain whose cheapest tradable commodity
has an effective reference price, in any region, below the resolution floor. The effective reference price is the region's override or, absent one, the commodity's (`spec-regions.md` 5.2, `spec-goods.md` 6.3).
**Tradable** is defined by `spec-agents.md` 3.4: a commodity in the tradable set
of at least one role (N-048). The floor is a documented rule derived from a
named parameter, not a heuristic and not a constant of its own: it moves when
`BeliefStep` does. The same check is FR-BEL-029 and, in the goods spec,
FR-GDS-002; `spec-config.md` 7 lists it once. A seed price from a warm start
must clear the same floor (`spec-viability.md` 8.2.2).

3.2. The minor unit is therefore chosen small, so that the cheapest tradable
good is still at or above the floor. How small is a calibration choice made
with the reference prices and the basket of section 5; the library imposes no
scale beyond 3.1. The RPG preset of section 4.3 uses 100 minor units for its smallest
denomination, which satisfies 3.1 for the default step (floor 50) and for a 1%
step (floor 100) when the cheapest good costs one of them.

## 4. Denominations (optional display)

A **ladder** is an ordered list of denominations, each with a name, a symbol
and a size in minor units. A **formatter** is built from a ladder and writes a
`Money` value in those denominations. Both are optional: a consumer that
prints plain integers, or has its own formatting, never uses them.

### 4.1 The ladder

A ladder has at least one denomination. Each has a non-empty name and symbol,
unique across the ladder, and a size in minor units of at least 1 and at most
10^12. Sizes are strictly decreasing from the largest denomination to the
smallest. A size need not divide the next larger one (a ladder may be
irregular); the formatter fills from the largest down and keeps the remainder.
A ladder that breaks these conditions is rejected at construction with an
error, never at format time.

The ladder is a free choice of the consumer. A setting with 12 pence to a
shilling and 20 shillings to a pound, a decimal currency, or a single unit
called "credits" changes only the ladder, never the simulation.

### 4.2 Formatting

Formatting is a pure function from `Money` to text.

1. A negative amount is formatted as `-` followed by the amount's magnitude.
   Compute the magnitude in `uint64`, so `math.MinInt64` does not overflow.
2. Taking denominations from largest to smallest: `count = r / size`, then
   `r = r % size`, starting with `r` as the magnitude.
3. Omit a denomination whose count is zero, but print `0` of the smallest
   denomination for a zero amount. The largest denomination is unbounded:
   1 000 of a 10 000-minor-unit coin prints as `1000`, not as a larger unit.
4. The remainder `r` left after the smallest denomination is smaller than its
   size. By default it is truncated. With a **precision** `p` (0 to 6, default
   0), the smallest denomination is printed with `p` decimal places:
   `frac = r * 10^p / size`, truncated and zero-padded to `p` digits, written
   after the smallest denomination's count (`4.60`). The smallest denomination
   is printed when its count or `frac` is non-zero, or when the whole amount is
   zero. The product `r * 10^p` fits in `uint64` because `size <= 10^12` and
   `p <= 6`.
5. If nothing was printed by steps 3 and 4 (a non-zero amount smaller than the
   smallest denomination, at precision 0), the formatter prints `0` of the
   smallest denomination, and a negative sign is written only when some printed
   count or fraction is non-zero, so the output is never empty and never `-0`.
   This step closes a gap in the text above, where such an amount printed
   nothing; it is a labelled choice (see the handoff) and reversible.

### 4.3 The RPG preset

The preset is a ready-made ladder, not part of the simulation. It is the
default for the example consumer and the setting this project was designed
with.

| Denomination | Symbol | Minor units | Relation    |
|--------------|--------|-------------|-------------|
| Copper       | cp     | 100         | 1/10 silver |
| Silver       | sp     | 1 000       | 10 copper   |
| Gold         | gp     | 10 000      | 10 silver   |

In this setting, a silver piece is about one household's day of subsistence
(the default basket of section 5.2), a copper is a small fraction of it, and a
gold is a large infrequent purchase (a cart, a month of skilled work). These
are guidance for choosing prices, not rules the code enforces.

| Minor units | Default            | Two-decimal precision |
|-------------|--------------------|-----------------------|
| 0           | `0 cp`             | `0.00 cp`             |
| 100         | `1 cp`             | `1.00 cp`             |
| 1 460       | `1 sp 4 cp`        | `1 sp 4.60 cp`        |
| 12 345      | `1 gp 2 sp 3 cp`   | `1 gp 2 sp 3.45 cp`   |
| -2 920      | `-2 sp 9 cp`       | `-2 sp 9.20 cp`       |

A second preset, a single denomination of size 1 named for the consumer, shows
that no ladder structure is required. A test builds an irregular ladder (for
example 240 pence to the pound and 12 to the shilling) to prove the formatter
makes no assumption of powers of ten.

## 5. The reference basket and the price level

The absolute price level is arbitrary: doubling every price and every balance
changes nothing but the digits. Calibration therefore needs an anchor, and
because regions differ, the anchor is per region.

5.1. `B_r` is the **reference basket** cost of region `r`: the cost in minor
units of one household's subsistence for one tick in that region, at that
region's effective reference prices (`spec-goods.md` 6.3; the override is `spec-regions.md` 5.2). It is the sum, over
the **baseline upkeep** of the household, of quantity times effective reference
price. The basket itself is a field of the region, one of the region-keyed values of `spec-regions.md` 5.1.3 (replace mode); its meaning, default and check stay here. The baseline upkeep is the world's upkeep list plus the region's extra
upkeep, and the upkeep itself (the list of goods a household must consume each
tick to stay alive, a "day" being a tick) is defined by `spec-agents.md` 5.1 and
5.2, which owns the **effective upkeep** an agent actually consumes. A role with
an upkeep list of its own pays its own cost; `B_r` stays the cost of the baseline
(`spec-agents.md` 5.6.1). The reference basket is also the **fixed basket** of the
indicator `BasketLevel` (`spec-indicators.md` 8.4): that measure divides the cost of the
baseline basket at the published prices by its **computed** cost at the reference prices,
not by the configured `B_r`, which validation only checks against the computed cost
within `BasketTolerance` (5.3). No simulation decision reads `B_r` (FR-CUR-023).

5.2. The default is `B_r = 1 000` minor units for every region that does not
set its own. The number is chosen so that a basket spread over several goods
stays above the resolution floor (5.7), not because of any currency: it is a
plain amount of the minor unit. In the RPG preset (4.3) it is exactly one of the middle
denomination, a "bread standard", and the preset describes it that way. A region sets its own `B_r` in
configuration when its conditions justify it: a cold region whose upkeep
includes fuel, or a region where staple food is dearer because of distance or
poor land.

5.3. A region's extra upkeep adds items to the baseline upkeep (data, for example
fuel in a cold region: `spec-agents.md` 5.2). `B_r` is computed from the combined
baseline. The configuration states `B_r` explicitly and validation checks that it
matches the cost of that baseline at the region's effective reference prices
within the tolerance `BasketTolerance`, so the two cannot drift apart silently.
`BasketTolerance` is a world parameter in basis points of `B_r`, default 1 000
(10%), bound 0 to 10 000, owned by `spec-agents.md` 12 and 5.6.1; a mismatch is
`ErrBasketMismatch` (FR-AGT-018). This names the tolerance that this section left
unnamed (N-016). The default of 5.2 is a default for the field when a region does
not state one; the check applies to it as well, so a chain whose baseline does not
cost about 1 000 in a region must state that region's `B_r`.

5.4. `B_r` is a calibration anchor, not a live value. The simulation does not
read it to make decisions. Live decisions use the agent's own effective upkeep at
the prices of its region (`spec-agents.md` 5.2, 5.6.2; `spec-welfare-and-wages.md`
section 2).

5.5. One agent represents one **household**, or `w` of them when its weight is
`w` (`spec-scale.md`). Its upkeep per household is the household's
daily basket. This avoids the family-size conventions in the literature
(`evidence.md`) inside the simulation: the household is whatever the upkeep says
it is, and the welfare ratio divides by that upkeep.

5.6. Because money is a single currency, nominal wages and prices are not
comparable across regions with different `B_r`. Compare regions by welfare
ratio (dimensionless), never by nominal wage. Reports that compare regions
state values in multiples of the regional basket.

5.7. Choosing `B_r` and the reference prices **MUST** respect section 3: with
`B_r = 1 000` and a household basket spread over several goods, the cheapest
good should still be at or above the resolution floor `ceil(10_000 / BeliefStep)`
(for the default step of 200 bp, 50 minor units; for a 1% step, 100, the smallest
denomination of the RPG preset). With `B_r = 1 000` and a floor of 50 a basket of
at most twenty distinct goods can clear it.

## 6. Non-goals and future work

- Physical coins: coins as commodities with a face value, money supply from
  mining and minting, carrying and theft.
- Exchange rates between metals or regions, and debasement.
- A price level that varies over time by design (a drifting basket or a scheduled inflation). An
  inflation index as a **measure** is not a non-goal: it is the fixed-basket `BasketLevel` of
  `spec-indicators.md` 8.4, a function of the published prices, and it changes no behaviour.
- Parsing formatted text (such as `1 sp 4 cp` in the RPG preset) back into `Money`. The formatter writes only.

The design keeps these open: `Money` stays an `int64` count of the minor unit.
The non-goals of the other topics are in their own specs.

## 7. Where this lives in the code

| Concern | Home |
|---------|------|
| `Money`, `Price`, minor unit | root package (architecture.md 3.3, 3.5) |
| Checked multiply and divide, used by every spec | `internal/` checked-arithmetic helpers |
| `ReferenceBasket` (`B_r`, section 5) | a plain struct field in the root package; parsed by `config/` |
| Denomination ladder, formatter and the RPG preset | an optional package `denomination/` (name provisional), standard library only; imported by consumers and examples, never by the root, `internal/`, `config/` or `strategy/` |

The formatter is a library feature, so the PR that adds `denomination/` also adds
it to the layout in architecture.md 2.6 and `AGENTS.md`, and the new terms to the
glossary in `doc.go` (architecture.md 3.2, 14.2). Until then the rules' layout
does not list it. A consumer that does not import it pays nothing and changes no
result (2.2).

## 8. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Every
number is a default or a bound owned by the spec named; none is calibrated
(N-015, N-027). Sections 2.5, 5.5, 5.6, the guidance on the RPG preset in 4.3 and
the worked choice of `B_r` in 5.2 and 5.7 are explanation, not requirements.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CUR-001 | Every amount of money and every price in the core module's types, events, snapshots, logs and configuration MUST be a single `int64` count of one minor unit, and no denomination name or symbol MUST appear in any of them. |
| FR-CUR-002 | No package of the core module (the root, `internal/`, `config/`, `strategy/`) MUST import the denomination package, and a build and a run without it MUST give identical results. |
| FR-CUR-003 | Every derivation of this spec MUST use integer arithmetic with checked overflow; where no rounding rule is stated the result MUST truncate toward zero; ratios MUST be basis points. |
| FR-CUR-004 | A quantity stated per tick (a household's upkeep, a region's reference basket) MUST NOT be rescaled when the calendar changes. |
| FR-CUR-005 | The configuration MUST NOT contain a wage field; wages MUST be measured from the market. |
| FR-CUR-006 | The resolution floor MUST be `ceil(10 000 / BeliefStep)` minor units, computed from the configured `BeliefStep` and not held as a constant, so it is 50 at the default `BeliefStep` of 200, 100 at 100 and 1 000 at 10. |
| FR-CUR-007 | Validation MUST reject a configuration in which the cheapest tradable commodity's effective reference price, in any region, is below the resolution floor, where tradable has the meaning of `spec-agents.md` 3.4. (The same check as FR-BEL-029 and FR-GDS-002.) |
| FR-CUR-008 | A denomination ladder MUST have at least one denomination, each with a non-empty name and a non-empty symbol that are unique across the ladder, and a size in minor units from 1 to 10^12. |
| FR-CUR-009 | The sizes of a ladder MUST be strictly decreasing from the largest denomination to the smallest, and a size MUST NOT be required to divide the next larger one. |
| FR-CUR-010 | A ladder, or a precision outside 0 to 6, that breaks these conditions MUST be rejected with an error at construction of the formatter, and formatting MUST NOT return an error. |
| FR-CUR-011 | Formatting MUST be a pure function of the ladder, the precision and the amount. |
| FR-CUR-012 | A negative amount MUST be formatted as `-` followed by its magnitude, the magnitude being computed in `uint64`, so `math.MinInt64` MUST format without overflow. |
| FR-CUR-013 | The formatter MUST fill denominations from the largest to the smallest with `count = r / size` and `r = r % size`, starting with `r` as the magnitude, and the largest denomination MUST be unbounded. |
| FR-CUR-014 | A denomination whose count is zero MUST be omitted, except that a zero amount MUST print `0` of the smallest denomination. |
| FR-CUR-015 | With a precision `p` from 0 to 6 (default 0), the smallest denomination MUST be printed with `p` decimal places, `frac = r * 10^p / size` truncated and zero-padded to `p` digits; with `p` of 0 the remainder left after the smallest denomination MUST be truncated; and `r * 10^p` MUST NOT overflow `uint64`. |
| FR-CUR-016 | The smallest denomination MUST be printed when its count or fraction is non-zero or the whole amount is zero; if nothing else was printed, `0` of the smallest denomination MUST be printed, and the sign MUST be written only when a printed count or fraction is non-zero, so the output is never empty and never `-0`. |
| FR-CUR-017 | The RPG preset (denominations of 100, 1 000 and 10 000 minor units) MUST format 0, 100, 1 460, 12 345 and -2 920 as the table of 4.3 shows, at the default precision and at precision 2. |
| FR-CUR-018 | A ladder of one denomination of size 1 MUST format every amount as that count of the denomination. |
| FR-CUR-019 | An irregular ladder (for example 240 pence to the pound and 12 to the shilling) MUST format correctly, so the formatter MUST make no assumption of powers of ten. |
| FR-CUR-020 | The reference basket `B_r` of a region MUST be the sum, over the baseline upkeep (the world's upkeep list plus the region's extra upkeep), of quantity times the commodity's effective reference price in that region, with checked arithmetic. |
| FR-CUR-021 | A region that states no reference basket MUST have `B_r` of 1 000 minor units, and the check of FR-CUR-022 MUST apply to that default as to a stated value. |
| FR-CUR-022 | Validation MUST reject, with `ErrBasketMismatch`, a region whose `B_r` differs from the cost of FR-CUR-020 by more than `BasketTolerance` basis points of `B_r`; `BasketTolerance` MUST default to 1 000 and lie between 0 and 10 000. (The same check as FR-AGT-018.) |
| FR-CUR-023 | No simulation decision MUST read `B_r`: changing a region's `B_r` within the tolerance, with validation passing, MUST NOT change any trade, price or balance. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-CUR-001 | Formatting MUST NOT panic for any `int64` amount, any valid ladder and any precision from 0 to 6, and SHOULD be covered by a fuzz test with those inputs. |
