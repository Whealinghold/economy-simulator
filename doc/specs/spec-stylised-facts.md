# Spec: Stylised facts: the testable macro properties of a healthy run, the acceptance bands, the failure regimes and the stability checks

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-STY-NNN` and
`NFR-STY-NNN`, which tests cite (testing.md section 4). They are in section 19.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)
decision 5 and 7), version 1 (`vision.md` section 4). It is a **fixture in
nature**: it defines what the library is tested against, and adds no API, no
`Config` field and no event. It is the compare step of the loop of the roadmap's
"Macro is measured, never set": measure (IND), compare with the expected
properties (STY), adjust a knob (CLB), run again (EXP). It needs
`spec-indicators.md` (IND: every measure a fact is stated in, by preset name and
field), `spec-stability.md` (STB: the register, the failure regimes F1 to F8 and
the scenarios S01 to S165, of which this spec takes the bands of the economic ones and
not of those of the class "harness test" (the list is in 5.11)),
`spec-reference-chain.md` (REF: the packs and the form of a band file),
`spec-viability.md` (VIA: the steady-state prices a fact is compared with),
`spec-world.md` (WLD: the run is a function of configuration, seed and calls) and
`spec-integration.md` (INT: the calls that inject a shock). It reads
`spec-observability.md` (OBS: events and snapshot fields), `spec-population.md`,
`spec-resources.md`, `spec-money-supply.md`, `spec-external-sector.md`,
`spec-transport.md`, `spec-merchants.md`, `spec-regions.md` and
`spec-checkpoint.md`, and asks several of them for small edits (section 21).

**Nothing has run.** The library does not exist, bazaarBot has not been run and
no band has ever been compared with a run (notes N-015, N-027). **Every number in
this spec that is not a derivation is a labelled placeholder**, and every band is
a **prior**: a value taken from a published fact, from a model in the specs or
from a hand derivation, stated with its basis so a reviewer can see how much to
trust it. A prior is a place to start, not a result. No default is called stable
here and this spec does not close N-027 (section 8.6 says what closes it). The
scripts of section 16 are stylised Docker Python models of the *definitions* of
this spec (a detector, a seed rule, a band-setting rule), run on synthetic series;
they are not runs of the library and show no property of it. The setting of the
worked examples (section 17) is invented and flagged as such; the core and its
vocabulary are setting-agnostic (N-017) and nothing here branches on a
commodity, a recipe or a role (architecture.md 9.1): a class name, a role name
and a region name are arguments, from the pack's data.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

The specs say how the parts of the economy behave; STB says how they can go
wrong; IND says what to measure. None of them says **what a healthy run looks
like**, in numbers, so that a build can be told apart from a broken one. This
spec does. It covers:

- **the form of a fact**: one entry format (the property in words, the measure, the
  window, the scale, the pack and scenario, the seeds, the band, what a pass and a
  fail look like, the source and the confidence), the classes of fact, the
  lifecycle of a band from a prior to a reviewed value, and the one home of a band
  (section 3);
- **what a fact is stated in**: windows, price bases, weights, the blind-spot
  fields, run lengths and the four tiers of test (section 4);
- **the facts**, in seven groups: price level and dynamics, the wage and population
  loop, shocks, distribution, money, trade and the warm start, each with the
  modules that must be on, and the places where two facts cannot both hold in one
  pack (section 5);
- **the failure regimes** as detectors over indicators, each with a threshold, a
  window and the loops of the register it exposes, and the difference between a
  regime and a failed test (section 6);
- **the stability checks**: bounded prices, no cobweb blow-up and a warm start that
  stays near the steady state, and what the word "stable" is allowed to mean
  (section 7);
- **acceptance versus report**, the statistics of a pass over several seeds, and
  the protocol that sets a band without tuning it to the code under test
  (section 8);
- **the tests** as requirements in the library's pyramid (section 9), and **what
  STY hands to EXP and CLB** (section 10).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| What a measure is and how it is computed, rounded and clamped | `spec-indicators.md` (IND); this spec names a preset and a field and adds none |
| The feedback loops, the dampers and the scenarios as such | `spec-stability.md` (STB); this spec takes the scenarios' bands and the regimes' thresholds and adds a regime (deflation) and no loop |
| The packs, the overlays and the shock scripts | `spec-reference-chain.md` (REF); this spec names a pack and an overlay and asks for no new one that REF has not already listed, except where section 21 says so |
| A sweep, a design, a paired run, a series store | `spec-experiments.md` (EXP); section 10 says what STY needs from it |
| The registry of knobs and the procedure that moves them against a band | `spec-calibration.md` (CLB); section 10 says what STY hands it |
| The events and snapshot fields the measures read | `spec-observability.md` (OBS) |
| The calls that inject a shock and their bounds | `spec-integration.md` (INT) |
| A rule of the library that reads a measure and acts | none: no rule does (IND 3.1); a policy actor is PLC's (later) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | A **fact** is a statement about a named measure of a named window of a named pack at a stated scale, with a band, a class, a source and a confidence. A fact with no source says "an internal requirement" and a band with no basis is a placeholder (N-027). No default is called stable | 3, 8.6 |
| 2 | Five classes: **X** exact (an identity or an invariant, no band), **O** ordinal (a sign, an order or a monotone relation, no magnitude), **B** banded (a magnitude inside a band), **R** report (recorded for calibration, never asserted), **P** pack-asserted (asserted only for the pack that names it). X and O are asserted where REF lists the check as exact (FR-REF-023) and are otherwise reported until REF's list is extended (owed row); B is asserted only once its key has a reviewed value; R never | 3.2 |
| 3 | **One home** for a band: STY defines the fact and its prior in this spec; the **value** a test asserts lives in `testdata/chains/bands/<pack>.toml` (REF 4.1, 7.1.5) and is set only by the protocol of 8.4. The regime thresholds are global, in `bands/regimes.toml`, because a regime must mean the same thing in every pack | 3.5, 6.5 |
| 4 | **Bands use REF's seed sets** (FR-REF-024, FR-REF-025): a band is set from a reviewed run over the nightly range 1 to 50 of REF 8.2, never from one seed and never from the code under test without review. The gate seeds 1 to 10 are inside that range, so the gate is a **regression** test, not an out-of-sample one; a hold-out set (51 to 60) is an owed REF row with a recommendation, and nothing here relies on it | 8.2, 8.4 |
| 5 | A **regime** is a property of a run that a detector classifies (none, detected or unknown); a **failed test** is an assertion on a fact that does not hold. A run may be in a regime and pass (a shock scenario that expects it), and may fail a fact with no regime. A regime threshold is looser than the band of every fact it backs | 6.4 |
| 6 | **Deflation with distress** is a new regime, added to STB as F9 (STY-R9); the other eight are STB's F1 to F8 with thresholds | 6.3 |
| 7 | The pass statistic over seeds is one of three, by the fact's kind: all seeds (X, O and the regimes of STY-G01), at least nine in ten (every B fact), or the median (report). Each is justified by exact binomial arithmetic in 8.3 | 8.3 |
| 8 | Four tiers of test: **S** short (unit; runs under `-short`), **G** gate (integration and end to end, seeds 1 to 10, the lengths of REF 7.2 and 8.2: at most 1 800 ticks), **N** nightly (seeds 1 to 50), **L** long (longer than 1 800 ticks, a build tag, never in the merge gate) | 4.5, 9 |
| 9 | A detector and a seed rule are **pure functions of plain series** and take no world, so they are unit-tested under `-short` against synthetic series (S tier), and the same code classifies a real run | 6.1, 9.1 |
| 10 | A band is stated **with its blind-spot fields**: a stable-price claim cites `Stale`, an inequality claim `CohortShare`, a regional comparison `Binding` (IND-A2, IND-A3). A band that omits one is ambiguous (IND 11.1.1) | 4.3 |
| 11 | "Stable" is the verdict of one of three named **stability checks** over a stated set of packs and seeds, and of nothing else. A default is called stable only when its checks pass at the default and at both neighbours on its knob's grid | 7, 8.6 |

## 3. The form of a fact

### 3.1 The entry

Every fact is written in this form. The field names are fixed so that a text
check of the spec can find them (FR-STY-001).

| Field | Content |
|-------|---------|
| ID | `STY-<group><nn>`: group `P` price, `W` wage and population, `S` shock, `D` distribution, `M` money, `T` trade, `H` warm start, `G` the umbrella fact. Never reused. The requirement that carries it is `FR-STY-nnn` (section 19) |
| Property | the statement, in words, of what is true of a healthy run |
| Measure | the IND preset and field (`spec-indicators.md` 9), the **window**, the **region scope** (a region, a connected pair, the realm), the **weights** of an index and the **price basis** (IND 11.1.1) |
| Pack | the REF pack and overlays that produce it and the scenarios (S, X, A of STB 8 and REF 7) that run it; the **modules** that must be on or off (5.10) |
| Seeds | the seed set and the pass statistic (8.2, 8.3) |
| Band | the band keys (REF 7.1.5: `<id>.<name>`), their unit and the **prior** value; "none" for an X or O fact |
| Pass and fail | what each looks like, in the measure, and what a fail is *not* (a regime, a defect, a finding) |
| Source | the evidence row (`evidence.md`) or "internal requirement"; "model" with the spec and section; "hand derivation" with the section |
| Confidence | one of the five of 3.3 |
| Class | X, O, B, R or P (3.2), and the tier of 4.5 |
| Loops | the register entries (`STB-Lnn`) the fact exposes |

### 3.2 The classes

| Class | What is asserted | When | A failure means |
|-------|------------------|------|-----------------|
| **X** exact | an identity or an invariant, with exact equality and no band (testing.md 6.3) | where REF lists the check as exact (FR-REF-023); otherwise reported until REF's list is extended (an owed row asks that every X check of this spec be asserted) | a defect in the library or in the fixture; never a band to widen |
| **O** ordinal | a sign, an order, a monotone relation or a derived bound: no magnitude | as X: where REF lists it among the exact checks, else reported until REF's list is extended (owed row) | a finding: either a defect, or the derivation of the fact is wrong; a human decides (the test is never edited to pass) |
| **B** banded | a measure inside a band | only when every key of the fact has a reviewed value (REF 7.1.5); until then the test **reports** the measure and the prior | the code moved the measure, or the band was set from too little; a human reviews, and the band is moved only by 8.4 |
| **R** report | nothing: the value is recorded in the verdict record and in the `t.Logf` line | never asserted | nothing; it is data for CLB |
| **P** pack | a B or O fact asserted only for the pack that names it | when the pack runs | as B or O, for that pack |

A fact may have parts of different classes (an O part and a B part); each part
has its own requirement line when they differ. **Acceptance** (what a build
must pass) is every X and O fact, and every B fact whose keys have values, for
the packs the fact names. Everything else is **report**. 8.1 restates it.

### 3.3 Confidence

A closed set, so that a reader can see how much a band is worth without reading
its source:

| Level | Meaning | Example |
|-------|---------|---------|
| **I** internal | an identity or an invariant of the design; exact; needs no outside evidence | the money of a closed economy is constant |
| **H** hand | derived by hand from the written rules, with the derivation in a spec; not run | a stationary closed economy has a constant mean price (MON 8.4) |
| **M** model | taken from a stylised model of a spec (a number or a shape); not run on the library | the evidence rule settles about 5.7% below the clearing price (STB 6.2 run 2b) |
| **E** evidence | a published fact read directly (`evidence.md`, status "read directly" or "read in full"), giving the shape, the order or the rough size, for another place and time | the craftsman-to-labourer wage ratio in England, 1.41 to 1.93 (Clark 2005) |
| **P** placeholder | a guess: nothing supports the number but the wish for a bound | `freeze_ticks` 20 |

A fact states the **highest** level its band has and, where the shape and the
size differ, both ("shape E, size P"). An evidence row that is a snippet, a
summary or unverified is **not** E: it is a lead and the band is P (N-018).
Level E never means "calibrated to": the published figure is for England or
Italy, for a time and a basket that are not the pack's. It bounds a sanity range
and gives a sign.

### 3.4 The life of a band

A band has four states and moves one way, by review:

1. **prior**: a value in this spec, with its basis. Nothing asserts it. A test
   that reads the band's key finds it unset and reports the measure beside the
   prior (REF 7.1.5);
2. **observed**: a reviewed run over the setting seeds (REF's 1 to 50, 8.4) has produced the
   values; they are recorded in the verdict record (8.5) and are the evidence for
   the next state;
3. **set**: the key has a value in `bands/<pack>.toml`, with `set_by` naming the
   run (REF 7.1.5). The gate asserts it;
4. **held**: the gate has passed on the seeds 1 to 10 and the nightly on 1 to 50
   (8.2) in later builds. A band that a later build misses is **not** widened to
   pass: the review of 8.4.5 decides whether the band, the code or the fact is
   wrong.

A band is never moved to make a test pass without a review that says which of the
three it was (testing.md 7.2 applies to bands as it does to a golden file).

### 3.5 One home

REF said "bands are STY's" and left a key with no value (REF 7.1.5). STB listed
placeholder thresholds in its 7 and 8. This spec settles where each lives
(position 3):

| What | Defined by | Value lives in |
|------|------------|----------------|
| a fact and its prior | this spec, section 5 | this spec (the prior) |
| a band a scenario asserts | this spec; the key is REF's name, mapped in 3.6 | `testdata/chains/bands/<pack>.toml`, one table per scenario (REF 7.1.5) |
| a regime threshold | this spec, section 6 | `testdata/chains/bands/regimes.toml`, one file for every pack |
| the seeds and the run lengths | REF 8.2, restated in 4.5 and 8.2 | test constants (REF 10) |
| the verdict of a run | this spec, 8.5 | `testdata/golden/stylised/<pack>.txt` (9.4) |

STB 7 and 8 keep the *meaning* of the regimes and the scenarios (what a regime
is, what a scenario injects) and drop their placeholder thresholds in favour of
this spec's (section 21). REF's scenario entries keep their setup and measure and
refer to the band table of 3.6 for the value.

### 3.6 The band keys of REF, mapped

REF named 37 band keys (7.2 to 7.4). Each is the key of one part of one fact; no
key is renamed, and the table is the one place the mapping is written. The prior
is in 5; a key whose prior says "none" is exact.

| REF key | Fact | Unit |
|---------|------|------|
| `s01.band_bp`, `s01.recover_ticks` | STY-S01 | bp, ticks |
| `s01.offset_bp` | STY-P04 | bp |
| `s03.pinned_ticks`, `s04.pinned_ticks` | STY-P02 | ticks |
| `s04.elasticity_max_bp` | STY-W05 | bp |
| `s05.range_factor`, `s05.range_growth_bp`, `s12.range_factor` | STY-P03 | bp |
| `s06.share_band_bp`, `s06.swing_bp` | STY-D03 | bp |
| `s06.free_tool_tol_bp` | STY-W04 | bp |
| `s07.freeze_ticks`, `s07.spoiled_bp`, `s16.pinned_share_bp` | STY-P02 | ticks, bp, bp |
| `s08.band_bp`, `s08.settle_ticks` | STY-P05 | bp, ticks |
| `s10.tolerance_bp`, `s10.dispersion_min_bp` | STY-P09 | bp |
| `s14.recover_band_bp`, `s14.recover_ticks`, `s14.fall_bp` | STY-S02 | bp, ticks, bp |
| `s17.spike_factor`, `s17.control_gap_bp` | STY-H04 | bp |
| `s19.ratio_band_bp` | STY-M03 | bp |
| `s19.stock_bound_bp` | STY-M05 | bp |
| `s20.ticks` | STY-M06 | ticks |
| `a1.price_band_bp` | STY-H01 | bp |
| `x1.warmup`, `x1.margin_bp` | STY-P06 | ticks, bp |
| `x2.share_band_bp` | STY-T02 | bp |
| `x4.trade_share_bp` | STY-P08 | bp |
| `x4.pinned_share_bp` | STY-P02 | bp |
| `x4.households_bp` | STY-G01 | bp |
| `x4.range_factor` | STY-P03 | bp |
| `x3.ms_per_tick`, `a10.seconds` | none: a cost budget, SCL's and VIA's, not a stylised fact | |

Two keys are STY's own and name unmet need, which no REF entry carries: `s01.unmet_bp`
(STY-S01's bound on the share of household-ticks unfed, prior 1 500) and `r8.unmet_bp`
(R8's threshold on the same share, prior 3 000, a held key of 6.5.3). They are
band keys of the bands files like the rest (registry rule of 9.1) and are not
used by STY-P06, whose worst-lane band is `x1.worst_bp` alone.

The scenarios that STB added after REF's 7.2 (S23 to S165, the harness tests aside) have no key yet; each
fact that backs one defines its keys in 5 (`s28.reopen_ticks` and so on) and REF
adds them to its band files when it builds the pack (section 21).

## 4. What a fact is stated in

IND 11.1.1 requires a band to be stated with the scale, the window, the price
basis and the weights, and IND returns what each preset actually used so a test
can check the band was read as written. This section fixes those choices once, so
the entries of 5 need only name them.

### 4.1 Windows and the transient

4.1.1. **A window** is half-open, in whole ticks, `[From, To)` (IND 4.1). The
windows of a fact are one of four kinds, named in its entry:

| Kind | Definition | Used for |
|------|------------|----------|
| **year** | a calendar-year window of `TicksPerYear` ticks (`YearWindow`, IND 10.2.1), partial years excluded | levels, shares, indices of a seasonal economy |
| **season** | the window of one season in each of several successive years (`SeasonWindows`, IND 10.2.2), one point per year | a delay or a cobweb in a pack whose prices have a seasonal swing |
| **run** | the whole of a run after the transient | maxima, minima, longest runs |
| **tick** | a window of one step, the finest resolution (IND 3.3.1) | a boundary identity, a count |

4.1.2. **The transient.** A band fact on a warm start excludes the first
calendar year of the run (REF 8.2: "the first lets the transient out, the last
two are the window"); a fact on a **cold** start excludes the first two. A fact
about the start itself (STY-H01, STY-H04) is stated over the ticks it names. The
exclusion is part of the fact, not of the test: a band set with a transient in
it would be a band of the transient. **A detector** (6) is the exception: it
reads whole years from tick 0 of a warm run, the first included, because a
regime is a gross failure and the first year is part of the run (and because
REF's 1 095 ticks then give the three windows it needs), for a warm run and for
a cold one alike (S22's cold leg is 1 095 ticks and must be classifiable); only a
**band fact** on a cold run excludes the first two. A recovery time, a scale run of 365 ticks from tick 0 (STY-P09), a base mean over the 100 ticks before a
shock (REF S01, S08: ticks 300 to 399, partly inside the transient year) and the
last 100 ticks of a run are **named exceptions** of their entries, with the
ticks REF's scripts fix; the transient exclusion does not move REF's scripts.
A boundary measure (a class 1 or 3 preset: `WelfareDistribution`, `Inequality`,
`RoleShares`) is read **at the close** of a year window, once per window, not per
tick (IND 4.5.3).

4.1.3. **A seasonal swing is not a cobweb.** A window shorter than a year, taken
at an arbitrary tick, reads a seasonal price swing as a range. A fact about a
range or a trend therefore uses the **year** kind or the **season** kind, and
never a bare run of 120 ticks, in a pack whose commodity has a seasonal supply
(`grow_staple` of `one` starts only in spring, REF 4.2). The model of 16
(`detectors.py`, 17.8) reads a stationary series that has only a seasonal swing
both ways. By year windows no pair of successive windows grows. By windows of 120
ticks, 3 to 6 of 23 successive pairs grow by more than 1 000 bp, and at a swing
of plus or minus 60% five pairs fail both conjuncts of STY-P03 (growing and above
a factor of 2.0) although nothing in the series is a cobweb; at 25% to 50% the
growth appears and the range stays under the factor. The harm is real but
conditional on a large seasonal swing, which a staple made in one season can
have. REF 7.2's S05 reads four successive windows of 120 ticks and is edited to
the season kind (section 21).

### 4.2 Price basis and weights

4.2.1. **A level** is read at the **local published price**. Two readings are
used and each fact says which: the **window unit value** (IND 4.3.2, volume
weighted, `PricesOfWindow`) where a value or an index is built, and the **mean
price** (the plain integer mean, half to even, of `MarketCleared.Price` over the
ticks of a window) where a fact compares prices over a few ticks (P05, P09, S01,
H01). They differ with volume and the two are never mixed in one band. A
**delivered price** is used only where the fact says so
(STY-P06) and the result says `Hired` (IND 8.4).

4.2.2. **A price index** is the **Fisher** index of a set against a base window
(IND 5.3.4), with the weights named in the entry: `Consumed` for what a
household pays, `Traded` for a market, `Basket` for the fixed basket of a
region (IND 5.3.2). The base is the first year window of the run for a detector and the first after the
transient for a band fact.
IND's open question 7 asks STY for a default of weights; this spec names none:
each fact that needs an index names its weights, and "the" price index is not a
defined term here.

4.2.3. **An offset** is read against the steady-state price of VIA for the same
pack (`Offset`, IND 9 row 20), taken from the pack's **warm-start table**, which
is the solver's prices (REF 4.4.1; `expected/` holds the report, not prices,
FR-REF-006). It is meaningful only where nothing else sets the price: a pack with an outside
corridor or a faucet has no closed steady-state price (N-114), and STY-P04 is
not asserted there (5.9, C7).

4.2.4. **A price table for values** (a GDP, a real wage) is the base window's
unit values (IND 4.3.5); a fact that uses one says it.

### 4.3 The blind-spot fields

A band is the conjunction of a measure and the fields that name its blind spot
(IND-A2). A B or O fact lists the fields it conjoins; a result with the field
outside its bound is **undetermined**, not passed:

| Field | Source | Bound | Why |
|-------|--------|-------|-----|
| `Stale` | every `PriceIndex` (IND 5.3.8): P07, R1, R9 only | at most `stale_max_bp`, prior 1 000 (P); `Stale` can turn a `none` into `unknown`, never a `detected` into `unknown` (an understated rise is still a rise) | an index of carried-forward prices looks stable because the market is frozen (IND-A3, STB-L03); a level read from `MarketCleared.Price` (P01, P04, P05, S01, H01) has no `Stale` field and is guarded by `Active` |
| `Clamped` | every index, ratio and growth (IND 5.3.7) | false | a clamped value is "at least 214 748 times": the band is exceeded by construction |
| `Active` ticks | `Turnover`, `PriceRange` (IND 9): the guard of R6, R8 and every level or range fact | at least `active_min_bp` of the window, prior 5 000 (P). **Not** a guard of R2, R4, R5b or R9's volume conjunct: a fall of activity is their subject | a range or a velocity of a market that did not trade is the carried price |
| `Outcomes` | `Pinned`, `Dispersion` | above 0 | a share of nothing is undefined (IND 9 row 18) |
| `Absent` | household rows (IND 4.5.4) | reported; a mean over rows that exist is weighted by them | a mean ratio over households whose ratio does not exist is a different population (IND 6.5) |
| `CohortShare` | `Inequality`, `WelfareDistribution` | reported with every figure; the first consumer's world has 0 | inequality within a cohort is zero by construction (IND 6.2) |
| `Binding` | a comparison of regions or roles (IND 8.7) | reported; a regional difference is not read as real while a cap acted there | a cap picks who is served by ID or region (WLD 10.2) |
| `Connected` | `PriceGap` (IND 9 row 6) | true for a pair that enters an average | a gap across a closed edge is not a failure of arbitrage |
| `Basis`, `Written` | `GDP` (IND 9 row 1) | reported | a GDP met by discarded output (IND-A2) |

A band that needs `stale_max_bp` or `active_min_bp` uses the same value
everywhere (they are shared keys of `regimes.toml`, 6.2, placeholders like the
rest), so that a pack cannot make its price look stable by loosening them.

### 4.4 Scale

4.4.1. Every fact is stated at **weight 1** (REF 2, position 6: the first
consumer's world needs no cohort, N-030). The scale-consistency fact (STY-P09)
is the one place a scale above 1 is run, with its tolerance.

4.4.2. A fact about inequality or a distribution reports `CohortShare` and is
silent when it is 0. A fact is **not** claimed at a scale where the cohort share
is above a stated value without saying so (IND 6.2(a)).

### 4.5 Run lengths, seeds and the four tiers

REF 8.2 fixes the packs, lengths and seeds of the existing scenarios; they are
test constants and placeholders (REF 10). This spec adds the **tier**, because a
fact's run length decides where it can run:

| Tier | Level (testing.md 2) | Seeds | Simulated length | Skipped with `-short` | Where it runs |
|------|----------------------|-------|------------------|-----------------------|---------------|
| **S** short | unit | none (synthetic series) | none | no | every build: detectors, seed rules, the band registry |
| **G** gate | integration and end to end | 1 to 10 | REF's lengths, 400 to 1 800 ticks (S14 is the longest; X4 and S22 are 1 095) | yes | the full suite of CI |
| **N** nightly | end to end | 1 to 50 | as G | yes | a scheduled run of CI (REF 8.2 "Nightly repeat") |
| **L** long | end to end | 1 to 10 | longer than 1 800 ticks | yes, and a build tag `stylised_long` | a scheduled run, never the merge gate |

4.5.1. The **cost of L** is counted, not measured. At REF 8.2's placeholder of
about 4 ms a tick for `one` (600 agents of 6.5 us), a hundred years of 365 ticks
(36 500 ticks) is about 146 seconds a seed, so ten seeds are about 24 minutes;
the Malthusian approach of STY-W02 has a time constant of about 33 years at
POP's default gain (POP 5.4), so a run that shows it is a hundred years or more.
That is why such a fact is tier L and why its band is set from 10 setting seeds
and not 40 (8.2, 8.4.3). The figures are placeholders (N-015).

4.5.2. A test of tier L is an ordinary Go test with a build tag, skipped with
`-short` like the others (testing.md 10.5, FR-STB-007). The library has no
timer, and no test sleeps (testing.md 10.2).

### 4.6 The modules a fact needs

A fact is asserted **only where its modules are present**, because a module that
is off has none of its positions and leaves every result as it would be without
it (WLD 6.1.5). 5.10 is the matrix. A fact whose modules are absent is not
failed: it is **not applicable**, and the verdict record says so (8.5).

**One rule for the later modules.** A fact holds on a pack with the fourteen modules
of the later stage (settlements, storage, employment, tenure, credit, risk,
infrastructure, insurance, polity, fiscal, institutions, policy, saving and
technology) **off**, unless the *On* column of its row in 5.10 names the module; the
*Off* column says what else must be absent. A pack that turns one on is its own pack
and needs its own record (7.4). Two cases are named because a fact would otherwise be
read where it cannot hold:

- **Technology.** A claim of stability is made with `Modules.Technology` off: a gain
  makes the steady state drift by design (`spec-technology.md` 8.7). A run with growth
  is held to TEC-S2 to S5 (S149 to S152), not to the facts of this section.
- **A binding control or right.** A pack with a price control in force (a cap or a
  floor) or a right that refuses an activity is **report-only** (class R) for the facts
  that assume free markets: P02, P04, P05, P06, S01 and S03. The exact facts (M01,
  M02) stay asserted. A control that binds nothing (the inert overlay) is not binding.

## 5. The facts

Each fact is in the form of 3.1 and in the order the roadmap row asks for:
price level and dynamics (5.1), the wage and population loop (5.2), shocks
(5.3), distribution (5.4), money (5.5), trade (5.6) and the warm start (5.7).
5.8 states the umbrella fact, 5.9 lists the places where two facts cannot both hold in one pack and 5.10 the
modules each needs. Every number marked "prior" is a placeholder or a stated
derivation; the *basis* says which. A prior of "none" means the fact is exact.

Reading the entries: **Measure** names IND's presets (`spec-indicators.md` 9) by
the exported name and the field; a *window* is one of the four kinds of 4.1.1;
**Seeds** gives the statistic of 8.3 (SR1 `all`, SR2 at least `ceil(0.9 N)` of the `N` seeds the pack runs, or SR3 `median`) and the
tier of 4.5.

### 5.1 Price level and dynamics

**STY-P01 A published price stays inside its bands.** *Property:* no tradable
commodity's published price leaves the interval its reference band and, where a
unit cost is known, its cost band allow, at any tick, and no balance or inventory
is negative. *Measure:* `MarketCleared.Price` of every market and tick against the
bounds of `spec-beliefs.md` 3.2 (`RefBandLow`, `RefBandHigh`, `CostFloor`,
`CostCeiling`); window: tick. *Pack:* every pack, every scenario (STB FR-STB-008).
*Seeds:* all, tier G (and in the invariants of REF 7.5). *Band:* none. *Pass and
fail:* a price outside is a defect of clearing or beliefs, never a regime; a pack
with an outside node reads the interval as `spec-external-sector.md` fixes it
(open question 1). *Source:* internal requirement (FR-STB-008). *Confidence:* I.
*Class:* X, tier G. *Loops:* STB-L05, L06, L09.

**STY-P02 A price is not pinned and a market does not freeze.** *Property:* in a
healthy run, the share of belief outcomes that end on a floor or a ceiling is
small, no market sits without a trade while both sides have unmatched orders for
long, and a perishable glut does not spoil most of its output. *Measure:*
the share of ticks of the **year** window in which the market is **pinned**
(a tick whose `Pinned.Share`, IND 9 row 18, is at or above `r5.pinned_bp`, 5 000: more than half of its
belief outcomes at a bound, the condition of the `Warn` of FR-BEL-031; this is
REF S16's "share of ticks at a bound" and is read from one-tick windows of
`Pinned`), the longest run of consecutive pinned ticks and the longest run of
consecutive frozen ticks (a tick with unmatched units on both sides and no
volume: `StuckTicks` of a one-tick window is 0 or 1, and the test computes the
run, IND 4.2.2 keeps only the count), and for the perishable the `spoilage` flow
over the output of `RecipeExecuted` (`BoundaryFlow`; STB S07). *Pack:* `one` (S03, S04, S09, S16), `one` +
`overlay/glut-prepared` at four `cost_floor` values (S07), `second` (X4).
*Seeds:* SR2, tier G; the S07 curve over the four floors is **report**.
*Band:* `s16.pinned_share_bp` prior 1 000 and `x4.pinned_share_bp` prior 1 000
(shares of ticks of the year, P); `s03.pinned_ticks` and `s04.pinned_ticks` (the
longest pinned run) prior 30, three periods of the `Warn` (P); `s07.freeze_ticks`
(the longest frozen run) prior 30 (P); `s07.spoiled_bp` prior 6 000
(shape M, STB 6.2 run 5 reports 45% spoiled at the default floor with mismatched
flows; size P). *Pass and fail:* the same measure is R5a's: a pinned run of
`r5.pinned_ticks` ticks is classified R5a, and a frozen run of `r5.freeze_ticks`
R5b (6.3); the regime implies the fact fails (60 ticks are 1 644 bp of a year,
above 1 000, 6.5.2); a pack that cannot meet
the band for a **staple** is the open question of N-028 (5.9, C2), not a silent
fail. *Source:* the `Warn` of FR-BEL-031 and the criterion "none untraded for
over 20% of rounds" (`spec-beliefs.md` 19, Doran and Parberry (2012), read in
full; scale-dependent, a lead). *Confidence:* shape I, size P. *Class:* B, tier G
(S07's curve R). *Loops:* STB-L05, L06, L09, L14, L44, L51.

**STY-P03 There is no cobweb blow-up.** *Property:* the range of a price over
successive windows does not grow without bound after a delayed supply response.
*Measure:* `PriceRange` (IND 9 row 17) of the commodity over **season** windows
of successive years (a seasonal pack, 4.1.3) or **year** windows, three or more (REF S12 gives three after the first), the
first year included; the conjunct `Active` of 4.3 (`PriceRange` carries no
`Stale`). *Pack:* `one`
(S05, S12 `long-grow`), `one` + the haul overlay (S32), the merchant lanes (S46,
S67). *Seeds:* SR2, tier G (four seasonal years from tick 0 end near tick 1 610,
under REF's 1 800; the second spring of REF 7.2 S04 and S05 is ticks 424 to 515); a cobweb in a healthy start is
R6, which STY-G01 asserts over all seeds. *Band:*
`s05.range_factor` and `s12.range_factor` and `x4.range_factor` prior 20 000
(High over Low of at most 2.0; basis: the cost band alone spans `CostCeiling /
CostFloor` = 30 000 / 7 000, 4.29 times, so 2.0 is a strict reading of it, P);
`s05.range_growth_bp` prior 1 000 (each window's range at most 10% above the one
before; P). *Pass and fail:* three REF forms, kept as REF has them. **S05:** the fact passes when a window is within
the factor **or** grows by at most the growth, and fails when a window is above
the factor **and** above the previous by more than the growth; **R6** is the same
two conditions at the looser thresholds of 6.2 in `windows` successive windows.
**S12** (calendar-year ranges) passes when each year after the first is within
`s12.range_factor` **and** grows by at most `s05.range_growth_bp` (no key of its
own). **X4(f)** is a bare bound: the range of each commodity in the window is
under `x4.range_factor`, with no growth escape. R6 is the S05 pair and no other. *Source:* the stability condition of the
cobweb under averaging expectations (`spec-stability.md` 6.3, a model of the
register; the literature is unverified: the cobweb rows of `evidence.md`).
*Confidence:* shape M, size P. *Class:* B, tier G/L. *Loops:* STB-L21, L36, L48,
L51, L84, L85.

**STY-P04 The long-run price is near the flow-balancing price.** *Property:*
with no faucet and no outside, the price a market settles at is near the price at
which flows balance, and the unmet need is small. *Measure:* `Offset` (IND 9 row
20) of the last 100 ticks of the run against the pack's steady-state price
(the pack's warm-start table), signed, with the `Unfed` share of row 12 beside it; window: the last 100 ticks after
a shock of S01, the year window of a no-shock run. *Pack:* `one` (S01 legs a and
b), every viable variant without an outside or a faucet. *Seeds:* SR2,
tier G. *Band:* `s01.offset_bp` prior 1 500 (basis: the revised evidence rule
settles 5.7% below the clearing price with 118 of 2 000 of need unmet in the
stylised model, `spec-stability.md` 6.2 run 2b and N-042; the prior is about 2.7
times STB's 5.6% (BEL's model gives 5.7%); size M); unmet need is the share of
household-ticks unfed of the window, the `Unfed` preset of IND 9 row 12 (row 20's
`Unmet` is the same figure; an owed IND row gives it its unit), key
`s01.unmet_bp` prior 1 500 (P: the model's 5.9% is units of need, another unit). *Pass and
fail:* a signed offset beyond the band with unmet need above its bound is **R8**
(F8); a negative offset alone is the known bias of the evidence rule reduced but
not removed (BEL open question 12) and fails the fact without a regime. *Source:*
model. *Confidence:* M. *Class:* B, tier G. *Loops:* STB-L01, L02, L04.

**STY-P05 A cost shock passes through a chain and settles.** *Property:* a fall of
the supply of an input raises its price while it lasts, the prices of what is
made from it follow and not before it, and every price settles near its earlier
level once the shock ends. *Measure:* the price series of the input and of each
product (`MarketCleared.Price`), the first tick at which each leaves
`s08.band_bp` of its mean over the 100 ticks before the shock, and the first tick
after the end of the shock at which it is back inside and stays. *Pack:* `one`
(S08: `dig_mineral` at 5 000 bp from tick 400 to 519; the series `mineral`,
`refined`, `tool_a`, `tool_b`), the demand shock leg of S01. *Seeds:* all for the
O parts, SR2 for the band, tier G. *Band:* `s08.band_bp` prior 1 000
(P); `s08.settle_ticks` prior 90, three times the 30 ticks to recover that the
combined belief rule shows in the register's model (`spec-stability.md` 6.2 run
2b; shape M, size P). *Pass and fail:* **O**: the price of the input is above its
band during the shock and the first excursions along the chain are in order
(input, intermediate, product): a product meets the shock only through the cost anchor
(`spec-production.md` 9.3, FR-PRO-037) and the market for the input, and both
move with the input, so the product cannot leave its band first. *Source:* hand derivation for the order, model
for the settling. *Confidence:* H (order), M (settling). *Class:* O and B, tier
G. *Loops:* STB-L10, L28.

**STY-P06 Connected regions' prices converge within the cost of carriage.**
*Property:* with trade on, no connected lane leaves a profit larger than a margin
after freight (in a pack with route rows, `spec-risk.md`, the cost of carriage is
freight and the expected loss of the route, `Routes.LossBP`, `spec-risk.md` 8.3: a
lane's gap is read net of both), and the relative prices of two trading regions lie within the
range of their autarky ratios. *Measure:* `PriceGap` (IND 9 row 6) per directed
connected lane and traded commodity, `GapAfterFreight` as a share of the
delivered price (`DeliveredPrice`, IND 8.4), the **median over the lanes that dispatched** of the positive part
`max(gap, 0)` over the **year** window (a lane with no dispatch has no
`Delivered`; a negative gap is a lane on which carriage does not pay); `Spread` against the same run without the trade overlay as a
control; `Connected` true; the conjunct `Binding` of 4.3; in a pack with route rows the expected loss of the route, which IND's
`DeliveredPrice` and `PriceGap` (IND 9 rows 5 and 6) carry once IND adds it (until then the
measure reads freight only, and the loss is a part of the gap that `x1.margin_bp` is read
with). *Pack:* `two` and
`three` with `overlay/trade.toml` (X1, X2 dynamic; S30, S46, S67), the control
the same pack without it. *Seeds:* SR2, tier N (the dynamic part waits for
MER and the code, REF 7.6). *Band:* `x1.margin_bp` prior 1 000 (10% of the
delivered price), `x1.warmup` prior 365 ticks (P); the INF model gives the
reason for a margin and its limit: shippers that spread their thresholds settle
the gap at 0.04 of the freight, an independent information error of the same size
leaves 0.6 of it and identical shippers 1.8 of it (`spec-information.md` 11.2; an
invented shipper rule, M). *Pass and fail:* **O**: the median `Spread` of the
trading pair is below the control's, and each traded relative price lies between
the two autarky ratios of VIA A3 (`ratio_outside_autarky` is a finding). **B**:
the median of the positive parts is at most the margin, and the **worst** lane at
most `x1.worst_bp`, prior 3 000 (P), in each of two successive years; a median
alone would hide one lane that never closes (17.6). A gap above the margin with
`Connected` false is not a failure (IND 8.6). *Source:* hand (the terms of trade
lie between the autarky ratios, `spec-viability.md` 9 and A3) and model. *Confidence:*
H (order), M (size). *Class:* O and B, tier N. *Loops:* STB-L84, L85, L93, L116.

**STY-P07 A stationary closed economy has no price trend.** *Property:* with
constant households, no faucet, no outside, no shock and settled behaviour, the
price level does not drift. *Measure:* the `PriceIndex.Fisher` of the whole
economy with `Consumed` weights, the link between two successive **year**
windows, as growth in basis points; conjuncts `Stale`, `Clamped`. *Pack:* `one`
and `second` (no demand overlay is needed), the links of the year windows after
the transient of a 1 095-tick run (one link).
*Seeds:* SR2, tier G. *Band:* `p07.drift_bp` per year, prior 500 (P);
the **sign** and the zero are derived: `sum(p * q) = M * V` with `M` fixed and
real volume constant gives a constant mean price (`spec-money-supply.md` 8.4).
*Pass and fail:* a drift beyond the band fails the fact; it is **R1** or **R9**
only above their own thresholds and persisting as 6.2 says (a drift of 10% is
neither). *Source:* hand derivation.
*Confidence:* H. *Class:* B, tier G. *Loops:* STB-L30, L69.

**STY-P08 Markets trade.** *Property:* every commodity that has a producer trades
in most ticks, and the mean volume of an active market is at least one unit.
*Measure:* the share of ticks of the **year** window with volume above 0
(`MarketCleared.Volume`) per commodity with a producer, and the mean volume per
active market (`Turnover`, IND 9 row 8). *Pack:* `second` (X4, the
configuration-only test), reported on every other pack. *Seeds:* SR2 (at least `ceil(0.9 N)`: all four of four on `second`), tier G. *Band:* `x4.trade_share_bp` prior 8 000 (the paper's 80%);
mean volume at least 1. *Pass and fail:* a commodity below the share is a failed
fact and, with households falling, R3. *Source:* the criteria of section VII of
Doran and Parberry (2012): every commodity produced and consumed, none untraded
over 20% of rounds, mean volume above one unit a round (`spec-beliefs.md` 19;
read in full; the other criteria are for prices of order one and are scale
dependent, and are **not** used). *Confidence:* shape E, size P (a lead, not a
calibration). *Class:* P (asserted for `second`), R elsewhere, tier G. *Loops:*
STB-L03, L09.

**STY-P09 The economy at scale 4 and 10 behaves as at scale 1.** *Property:*
the time-averaged prices at a scale above 1 stay within a tolerance of those at
scale 1, and the dispersion of belief centres does not collapse. *Measure:* the
time-averaged `MarketCleared.Price` per commodity over 365 ticks at scales 1, 4
and 10 of `one` + `overlay/h200`, and `Dispersion` (IND 9 row 19); conjunct:
`CohortShare` reported. *Pack:* S10. *Seeds:* SR2, tier G. *Band:*
`s10.tolerance_bp` prior 1 000 (SCL 7.2's placeholder of 10%, P);
`s10.dispersion_min_bp` prior 100 (P), stricter than the regime threshold
`r7.disp_bp` (6.3). *Pass and fail:* **X** parts first: the weights of a group add
to 200 and money and goods at tick 0 are equal at the three scales (S10's exact
checks). *Source:* internal requirement (FR-SCL-007, FR-SCL-010). *Confidence:*
I for the exact parts, P for the bands. *Class:* X and B, tier G. *Loops:*
STB-L13, L33, L34.

### 5.2 The wage and population loop

The roadmap row asks for "the pre-industrial wage and population cycle". What
was read (`evidence.md`: Clark 2007 chapter 2 and the 2006 draft chapters 4 and
5, read directly) gives the **logic of the equilibrium**: births flat or rising
with income, deaths falling with it, living standards falling as households
rise, so a stable population sits where the two schedules cross and a higher
death schedule *raises* living standards. It gives no period and no amplitude of
a cycle, and `spec-population.md` 5.4 shows that the default schedules approach
the fixed point **monotonically** (gain times delay of 0.004, a time constant of
about 33 years): a cycle appears only above a gain times delay of about 1.5,
which the default avoids. The facts below are therefore the equilibrium, the
approach to it and the sign of the shock response, which have a source, and the
cycle is **report only**, with no band (W02, part c).

**STY-W01 The real wage is within the range the evidence shows.** *Property:* in
a viable pack with demand beyond upkeep, the household-weighted mean welfare ratio
of the unskilled roles is at or above subsistence and below the best of the
published pre-industrial figures. *Measure:* `WelfareDistribution.Mean` (IND 9 row
10) of the roles marked unskilled, over the **year** window, household-weighted
(`RatioHouseholds`), with `Absent` reported; the share below 10 000 beside it.
*Pack:* `one-demand` (REF 4.1 has no other demand pack; `five` with the overlay is a pack REF would add, S24); in the closed
upkeep-only packs the value is 10 000 by identity (STY-M03) and is not this
fact. *Seeds:* SR2, tier G. *Band:* `w01.mean_low_bp` prior 10 000 and
`w01.mean_high_bp` prior 30 000. *Basis:* Allen (2013), read directly: English
labourers 1700-60 about 3 (London), 2 (southern England), 1.5 (northern England)
and 1 (northern Italy), about 2.5 by 1820 and at least 3 by 1850, a ratio of a
household's income to a basket's cost, the same concept as the library's ratio
(`spec-welfare-and-wages.md` 2). The pack's own ceiling, `1 + Wmax / U`, is
tighter than 3 (`1 + Wref / U` is about 13 200 for `one-demand`, REF 4.4.6, the
expectation; `Wmax` scales higher, DEM 7) and is STY-M04. The upper bound cannot
bind in any pack the demand module can build, so it is a sanity range and the
lower bound is the assertion. The fact is about the real wage as the evidence
states it (ratio of income to a basket); `RealWage` of IND 9 row 14 is the wage
per working tick over the upkeep cost and is reported beside it. *Pass and
fail:* below 10 000 on average is a failed fact and, with unfed households, a
sign of R3; the upper end is a sanity bound, not a target. *Source:* evidence E,
for another place, time and basket: it bounds a range and calibrates nothing.
*Confidence:* shape and size E, as a range. *Class:* B, tier G. *Loops:*
STB-L46, L57.

**STY-W02 The population approaches its fixed point and the ratio sits at the
replacement ratio.** *Property:* with the population module, a binding
endowment and wants on, households started above and below the fixed point end
at one level, approach it without crossing it more than once while the gain times
the delay is small, and the mean ratio there is the replacement ratio `r*` of the
schedules. *Measure:* (a) the `Households` series of the region and its crossings
of the end level; (b) the difference of the two end levels over the end level;
(c) `PopulationChanged` births and deaths and the `WelfareDistribution.Mean`
series, reported as a period if the series cycles. *Pack:* `one-demand` +
`overlay/resources.toml` (the binding `refill` set) + a population overlay with
`Renewal` 0, started at 0.6 and 1.6 of the households at which the stock binds
(S37, S41); the fixed point is **observed**, never assumed (RES 8.5: `r_full` is
a property of the pack and not a field). *Seeds:* all for (a), SR2 for
(b), tier L (a hundred years or more, 4.5.1). *Band:* `w02.n_band_bp` prior 500,
`w02.ratio_band_bp` prior 500 of `r*` (P; the RES model agreed with the closed
form to 0.2% for a flat refilling curve and POP's stake model to 3%, M, so the
priors are 2 to 3 times those). *Pass and fail:* **O**, derived: no more than one
crossing when validation raised neither `population_gain` nor
`resource_overshoot` (POP 5.4: monotone up to a gain times delay of 0.6 to 0.7;
RES 8.3: a refill stock closing at least 0.16 of its gap a year is monotone). Two
end levels further apart than the band are a failed fact; a population that
reaches its floor is R3. Part (c) is **R**: no band, because no source read gives
a period. *Source:* Clark (2007) ch. 2 (shape); the models of POP 13 and RES 17
(size). *Confidence:* shape E, size M and P. *Class:* O and B, R for (c), tier L.
*Loops:* STB-L75, L78, L80, L87, L88.

**STY-W03 A shock moves households and the ratio of survivors in opposite
directions.** *Property:* after a shock that lowers households, the mean ratio of
those who remain rises above its earlier level while households are below theirs,
and the two return together. *Measure:* from the tick of the trough to the tick
households are back within `s14.recover_band_bp` of their earlier level, the sign
of the sum over ticks of `(dH_t * dR_t)`, with `H` the `Households` series and
`R` the household-weighted `WelfareDistribution.Mean` (exact integers: no
correlation coefficient and no division); the shock is `SetDeathScales` or a
yield shock (`spec-integration.md` 9.1). *Pack:* the S37 and S38 packs of W02
(the fixed factor must bind, or there is no feedback, POP 5.2), `famine` for the
ladder's leg. *Seeds:* all, tier L. *Band:* none; the sign is the fact.
*Pass and fail:* a non-negative sum means the loop of assumption 3 is absent in
the pack (`no_fixed_factor`, `spec-population.md` 9.2): a finding about the pack,
not a regime. *Source:* Clark (2007) ch. 2, read directly: "what raises the death
rate raises living standards"; the model of `spec-population.md` 5.6 item 3
(12 000 to 12 410 after a death schedule four times higher for a year).
*Confidence:* shape E, H. *Class:* O, tier L. *Loops:* STB-L75, L77, L82.

**STY-W04 Skilled work earns a premium, as the evidence says it does.**
*Property:* with the skill barrier, a role that needs a tool or a traineeship
earns more than the unskilled roles beside it, and with the tool free the premium
vanishes. *Measure:* `PremiumBP` of the role rows (OBS 5.3.4; WEL 4.3), the
observed premium of a skilled role over the unskilled roles of its region, over
the **year** window; the four combinations of S06. *Pack:* `one` +
`overlay/imbalance-cook` with `overlay/skill` and `overlay/notool` (S06).
*Seeds:* SR2, tier G. *Band:* **O**: with the tool the premium is above
10 000 (`HasPremium`) and rises with the tool's price; with the tool off it is
within `s06.free_tool_tol_bp` of 10 000, prior 1 000 (P). **R**: the historical
range of 14 100 to 19 300 (a recorded reference, not a band). *Pass and fail:*
the whole-series band is reported only (FR-SKL-024). *Source:* Clark (2005)
table A2 via Our World in Data, read directly: the craftsman-to-labourer wage
ratio in English building trades, twelve points 1220 to 1900: 1.56, 1.93, 1.93,
1.79, 1.50, 1.44, 1.47, 1.44, 1.60, 1.51, 1.52, 1.41, minimum 1.41, median 1.515,
maximum 1.93 (`worked.py` of 16 recomputes them); the decision of README "How the
premium is checked": ordinal required, historical band reported. *Confidence:*
size E for the range (another trade, place and time), shape H for the order.
*Class:* O and B, R for the range, tier G. *Loops:* STB-L36, L37.

**STY-W05 Days worked emerge and a staple's labour supply is not elastic.**
*Property:* the share of ticks households work is not set and sits below the
expected working share when the target is met (it cannot be in a closed
upkeep-only pack, STY-M03, where every household works to the floor-bounded
maximum); the measured demand elasticity of the staple is at most 1. *Measure:* the working-tick share of the households of the unskilled
roles over the **year** window (the working ticks in the window that INT's
`Agent(a)` returns, 4.7.2, which are those of the role's effective welfare window,
so the query is polled once per effective window length and the readings are
summed, never once a year), and the **arc elasticity**
`rdiv(10 000 * (q1 - q0) * (p1 + p0), (p1 - p0) * (q1 + q0))` between the
ten-tick mean price and quantity before and during the shock (an integer
midpoint form; REF S04 says "slope of the log", which needs a regression no
indicator offers, and an owed REF row asks for this form). *Pack:* `one` and `one` + `overlay/nofloor` (S04). *Seeds:*
SR2, tier G. *Band:* `s04.elasticity_max_bp` prior 10 000 (AGT 11.6 states
a staple's elasticity is 0 to 1, a design statement, H); the working share is **R**
beside two reference points, `ExpectedWorkTicks` over `TicksPerYear` (68.5% at the
defaults) and the 150 days of Humphries and Weisdorf (2019), "under 150 days a
year" for medieval labourers, 4 110 bp of 365 (read directly; one source, another
economy). *Pass and fail:* an elasticity above 1 for a staple contradicts the
design and is a finding for AGT. *Source:* hand, evidence. *Confidence:* H for the
bound, E for the reference. *Class:* B for the bound, R for the share, tier G.
*Loops:* STB-L14, L15, L16, L51.

**STY-W06 Natural increase is bounded by the schedules and the closed population
by its money.** *Property:* the households formed in a year never exceed the
schedules' maximum rate, near the fixed point natural increase is near zero, and
in a closed region with an empty pool the population stops at about the money
over the stake. *Measure:* `PopulationChanged.BirthsDone` and `DeathsDone` summed
over the **year** window against the households (`Households`, IND 9 row 9, the
account of change) and `NaturalIncreaseBP` of the role rows; the `Households`
level against `M / stake` (`Sectors`, MON 13.4). *Pack:* the POP overlay (S37,
S39). *Seeds:* all for the bound, SR2 for the bands, tier L. *Band:* **X**:
births done in a year at most `max(birth schedule)` in basis points of the
households plus the rounding of the carry (derived, `spec-population.md` 6.3
`g_max`). **B**: `w06.fixed_point_bp` prior 100 bp a year (the realised net rate of the account of change, `Households` IND 9 row 9,
not the schedule's `NaturalIncreaseBP`, within the band near the end level, P); `w06.money_cap_bp` prior 1 000 below `M / stake`
(the model of POP 13.2: about 1 160 of 1 200, 2 350 of 2 400 and about 4 750 of 4 800). *Pass
and fail:* `BirthsUnfunded` non-zero at the stop is the signature (S39); a
population above `M / stake` is a defect of the stake rule. *Source:* hand
(POP 5.5) and model. *Confidence:* I for the bound, M for the bands. *Class:* X
and B, tier L. *Loops:* STB-L75, L76, L79.

**STY-W07 Unemployment and the employers' margin are reported.** *Property:* with
the employment module on, the share of the households that offered work and were
not hired and the margin that employers keep on the hired output are recorded; no
value is a defect and no band exists. A steady state with employers is not a
zero-profit state (`spec-viability.md` 4.7), so the margin is expected above 0.
*Measure:* `Unemployment` (IND 9 row 22: the unmatched household-ticks over the
offered household-ticks by region and role, with `Offered` beside it; EMP 10.1.1)
over the **year** window, and the employers' margin as 10 000 less the labour share
of `WageBill` (IND 9 row 23, EMP 10.2.1), by region. *Pack:* `one` +
`overlay/employment.toml` (REF 4.5; S97 to S100); the module on, 5.10. *Seeds:* every
seed the pack runs, recorded and not asserted, tier G. *Band:* none. *Pass and fail:*
nothing; the values go to the verdict record and to the `t.Logf` line, and a region
with no household that offered has no value, not 0. *Source:* internal requirement
(EMP 10, the finding of VIA 4.7). *Confidence:* P (no value is claimed). *Class:* R,
tier G. *Loops:* STB-L14, L150 (EMP-A5).

### 5.3 Shocks

Every shock is injected through INT's calls, at a boundary between steps
(`spec-integration.md` 9.1; FR-STB-014), and cleared through the same call; none
edits state. The calls the facts use: `SetYieldShocks` (a failed or a bumper
harvest), `SetEdgeStatuses` (a closed or slowed road, an embargo),
`SetDeathScales` and `SetBirthScales`, `SetOutsideShocks`, `SetFlowScales`,
`AdjustEndowments`, `AddAgent` with `Direct` (a buyer).

**STY-S01 A supply or demand shock moves the price the right way and it
returns.** *Property:* a fall of supply raises the price while it lasts, a demand
step raises it, and after the shock the price returns within a band in a stated
time. *Measure:* `MarketCleared.Price` of the commodity in the region; `P0` the
mean of ticks 300 to 399; the recovery time the first tick after the shock at
which the ten-tick mean is within `s01.band_bp` of `P0` and stays (REF 7.2).
*Pack:* `one`, legs (a) `cut_fuel` at 5 000 bp from tick 400 to 459 and (b) a
visitor buys `fuel` at 5% of its market's mean volume for 60 ticks. *Seeds:* all
for the sign, SR2 for the time, tier G. *Band:* **O**: during the shock
the mean price is above `P0` in both legs. **B**: `s01.band_bp` prior 1 000,
`s01.recover_ticks` prior 90 (the same basis as STY-P05: three times the 30
ticks of the register's combined rule, M, size P). *Pass and fail:* a price that
falls under a supply cut is a defect of beliefs or clearing, not a regime; a time
beyond the band with the price pinned is **R5a**. *Source:* hand (sign), model
(time). *Confidence:* H, M. *Class:* O and B, tier G. *Loops:* STB-L01, L02, L04.

**STY-S02 A failed harvest is recovered from, or the collapse is of a stated
kind.** *Property:* for each depth of a failed harvest in `famine`, the region's
households either return to near their earlier level in a time bound by the speed
at which the modules in force can regrow them, or the run is a collapse (R3) whose
kind is stated: the fall is bounded, the ladder's rungs act in order and no
accounting identity breaks. The depth below which the region does not recover is
**reported**. *Measure:* `Households` as a share of the start, the longest run of
ticks with any unfed (`Unfed`), relief and the pool balance, removals by cause,
the first tick back within `s14.recover_band_bp`; the **speed limit** `t_min`:
the ticks the households need to regrow from the trough to the **lower edge of
the recovery band**, `10 000 - s14.recover_band_bp` of the earlier level (the
level the observed time is measured to), at the fastest rate the modules in force allow, by integer iteration **per tick with a carried remainder** over
`10 000 * TicksPerYear` (the rule of `spec-population.md` 3.8; whole-year
compounding is a coarser reading and is not used) with `g_max = max(birth) - min(death)` for
the population module (`spec-population.md` 6.3), and, without it, by replacement alone: at most
`Renewal` times the leavers of the tick (`spec-spawner.md` 7.1), which the caps of
`spec-spawner.md` 6.3 do not limit (FR-SPN-022), and as many as the pool funds,
`EntryPoolShare` of it per tick (`spec-spawner.md` 8.2); the per-tick cap
`ceil(Households * EntryShare / 10 000)` is not the limit of regrowth (open question 2,
answered). *Pack:* `famine` with the
estates overlay, depth `m` in `{7 000, 5 000, 2 500, 0}` bp from tick 424 to 515
(S14), 1 800 ticks; the longer horizon of a population-module recovery is tier L.
*Seeds:* all for O and X, SR2 for B, tier G (L for POP on). *Band:*
**O**: the observed recovery time is at least `t_min` (a faster recovery is an
accounting defect). **B**: `s14.recover_ticks` prior `4 * t_min` (P),
`s14.recover_band_bp` prior 1 000, `s14.fall_bp` prior 5 000 (a collapse of the
stated kind falls by at most half; P). **X**: no balance or inventory is
negative, goods and money are conserved at every tick, the pool is never
overdrawn (FR-EST-022), the rungs act in order (FR-SPN-013 to 015). *Pass and
fail:* a depth that neither recovers within the bound nor collapses in the stated
kind fails the fact; the *threshold depth* is R. *Source:* hand (the speed limit
from the schedules), model for the ladder (SPN 16.2). *Confidence:* H for O and
X, P for B. *Class:* O, X and B, R for the threshold, tier G/L. *Loops:* STB-L11,
L31, L35, L38, L39, L50.

**STY-S03 A closed road widens the gap and a reopening closes it.** *Property:*
while an edge is closed the price gap between its ends rises, the households of
the cut region fall no faster than in a closed region of the same size, and after
the reopening the gap returns within a band in a stated time. *Measure:*
`PriceGap.Spread` of the pair (`Connected` is false while cut and the pair is
excluded from the P06 average then), `Households` of the cut region against the
closed-region variant, the events `EdgeStatusChanged` at both changes. *Pack:* a
three-region chain with `overlay/trade.toml`, the middle link closed for a stated
number of ticks (S28, REG-A1). *Seeds:* all for O and X, SR2 for B, tier
N (waits for TRN, MER and INT). *Band:* **O**: `Spread` at the end of the closure
is at least its value at the start; **X**: `EdgeStatusChanged` is emitted at
both changes. **B**: `s28.band_bp` prior 1 500, `s28.reopen_ticks` prior 180 (P).
*Pass and fail:* the cut region falling faster than its closed-region control
means the closure costs more than isolation, a finding for REG and MER. *Source:*
hand. *Confidence:* H, P. *Class:* O, X and B, tier N. *Loops:* STB-L66, L84.

**STY-S04 An embargo or a lost import shows as hunger no earlier than the stock
allows.** *Property:* with an import-only upkeep good and its edge closed, a
household is not unfed before it has used its stock, and the first unfed tick is
recorded. *Measure:* the first tick with any unfed household in the region
(`UpkeepShortfall`), and each household's holdings of the good at the cut tick
(INT's `Agent(a)`, Holdings, 4.7.2; `StockCover` is an aggregate and does not
show a household). *Pack:* the S52 pack
(`overlay/external.toml`, `SetEdgeStatuses` on the gate edge). *Seeds:* all, tier
N. *Band:* **O**: for a non-perishable good the first unfed tick of a household
is at least the cut tick plus `floor(held / need)` at the cut, because holdings
can fall by at most the need a tick (derived from the consumption rule,
`spec-agents.md` 5.3; a spoiling good falls faster and is excluded). **R**: the
tick itself and the ladder's response (`CutClosed` counters, EXT D2). *Source:*
hand. *Confidence:* H. *Class:* O and R, tier N. *Loops:* STB-L101, L66.

**STY-S05 A faucet's accounts are exact and its price effect is reported.**
*Property:* the money a named flow creates is exactly its boundary balance and
the money supply moves by exactly it, and the price level's response is a
recorded series. *Measure:* `Sectors` boundary balance by reason against `-Total`
of the flow, `M = M_init + issued`, the audit; the `PriceIndex.Fisher` chain,
`Turnover.Velocity` and `M_h / H` in the **year** windows, with `SetFlowScales` 0
at a stated tick. *Pack:* `one` + `overlay/faucet.toml` and
`overlay/faucet-sunk.toml` (S13, S33). *Seeds:* all for X, median for R, tier G.
*Band:* **X**: the accounting (FR-MON-043 and the identities of
`spec-ledger.md` 9.1). **R**: the price drift has **no band**: neither a source
nor a model fixes the direction robustly, because velocity and hoarding can
absorb a faucet (MON 8.4: nominal trade value moves only if velocity or real
volume does), and the neutrality of money in the long run is a claim of
Lengnick's introduction whose results were not read (`evidence.md`, the row on
the baseline model's reproduced properties).
*Source:* internal for X, none for R. *Confidence:* I, none. *Class:* X and R,
tier G. *Loops:* STB-L30, L69, L70.

**STY-S06 The order of a famine is the ladder's.** *Property:* hunger precedes
removal, removal follows a stated privation, and nothing is created or lost.
*Measure:* the events `UpkeepShortfall`, `RoleChanged`, `ReliefPaid`,
`WeightChanged` and `AgentBankrupted` with `Cause` over the S14 runs, and each
agent's `Unfed` and `Privation` from INT's `Agent(a)`, polled every tick by the
test on `famine` (600 agents: 600 queries a tick of about 23 reads each, IND 4.5.3;
counted, not measured). No event carries an agent's first unfed tick or its
privation; if the cost is too high the poll is once per effective window and the
bound below holds to within that window. IND 4.5.3 says rows are built "once per
window or per frame, not per tick"; this is the one use that needs a tick poll, and
FR-REF-027 ("from the events and the snapshot") gets an owed row to name INT's
queries. *Pack:*
`famine`, the four depths. *Seeds:* all, tier G. *Band:* none. *Pass and fail:*
**X**: a removal for hunger (`starved`, `insolvent`) of an agent comes at least
`RemovePrivation` unfed ticks after its first unfed tick, since privation rises
by 1 a tick when unfed and falls by 1 otherwise (`spec-spawner.md` 5.1); relief
is paid before reduction and removal (rung 2 before rung 3, SPN 5); `AgentBankrupted`
is the only event that removes an agent (WLD C11). *Source:* internal requirement
(FR-SPN-013 to 015, 032). *Confidence:* I. *Class:* X, tier G. *Loops:* STB-L35,
L39, L50.

### 5.4 Distribution

No source read gives a distribution of money or wealth for a pre-industrial
economy: `evidence.md` has wages, ratios and rates, not shares. The distribution
facts are therefore one banded fact on hunger, one on the shares of roles, and one
that is report only.

**STY-D01 A healthy run is fed.** *Property:* a viable pack run from its warm
start with no shock has almost no unfed household-ticks after the transient.
*Measure:* `Unfed` (IND 9 row 12): the share of **household-ticks** unfed over the
**year** window per region and role, the running household count as the
denominator, and the largest `MaxHunger`. *Pack:* every viable variant except
`famine` (whose autarky `lambda*` of 1.315 puts the labour share to subsist at
76%, above the 68.5% that `ExpectedWorkTicks` allows, REF 4.4.5): `one`, `two`,
`three`, `five`, `second`, `one-demand`, **asserted today** on `one` (S19, 1 095 ticks) and `second` (X4, 1 095 ticks, seeds 1 to 4); the others **need a
longer run** (REF 7.5 runs `three` and `one-demand` 400 ticks on seeds 1 to 4,
`five` 400 on seeds 1 and 2 and `two` not at all: owed REF 8.2 rows). No shock. *Seeds:* SR2,
tier G (a regime in a healthy start is G01's, over all seeds). *Band:*
`d01.unfed_bp` prior 100 (1% of household-ticks; P). *Pass and fail:* above the
band with households falling is **R3**; above it with households steady is a
failed fact (an under-supplied role, a thin market). *Source:* internal
requirement: `vision.md` 3 goal 1, "no collapse from a healthy start".
*Confidence:* shape I, size P. *Class:* B, tier G. *Loops:* STB-L11, L39, L45.

**STY-D02 Inequality is measured with its blind spot.** *Property:* the
distribution of money of the households is recorded with its cohort coverage.
*Measure:* `Inequality` (IND 9 row 11): the Gini of money, `Top(1 000)`,
`Bottom(4 000)`, `P90 / P10` and `CohortShare`, over the **year** window.
*Pack:* every pack, every run. *Seeds:* median, tier G. *Band:* none; no source
read supports a number (5.4 preface). *Pass and fail:* nothing is asserted;
`CohortShare` is 0 in the first consumer's world, and a figure at a scale above 1
is a lower bound (IND 6.2(b)). *Source:* none. *Confidence:* none. *Class:* R,
tier G. *Loops:* STB-L11, L13.

**STY-D03 The shares of the roles do not swing.** *Property:* in a run with a
balanced census, the share of a role in its region stays within a band, and a role
that starts off its balanced share returns toward it without a swing larger than
a band. *Measure:* `RoleShares` (IND 9 row 16): the share of each role at the
start and end of the **year** windows and the swing `max - min` over the last
730 ticks, a role with no households reported as 0. *Pack:* `one`, and `one` +
`overlay/imbalance-cook` (the cooks start 163 against the balanced 136, 2 267 bp,
S06), with `overlay/skill` and `overlay/notool` as the four combinations.
*Seeds:* SR2, tier G. *Band:* `s06.share_band_bp` prior 200 (the end share
within 2 points of the balanced one) and `s06.swing_bp` prior 300 (P); the model
of the spawner's entry share shows amplitude 0.00 at 50 bp over a window of 30,
`spec-spawner.md` 16.1 (M, with no switching cost, no tool, a demand curve per
role). *Pass and fail:* a swing above `s06.swing_bp` fails the fact; one above
`r6.swing_bp` in two windows is **R6**; a share that does not return fails the
fact only. *Source:* model. *Confidence:* shape M,
size P. *Class:* B, tier G. *Loops:* STB-L36, L37, L54, L61.

### 5.5 Money

**STY-M01 A closed economy's money is constant.** *Property:* with no faucet or
sink after construction, the money of the economy equals its start at every
tick, and the sectors add up to it. *Measure:* `Sectors.M` against `M_init`;
`M_h + M_p + M_c + M_t = M` (IND 9 row 7); window: tick. *Pack:* every pack with no
money flow and no outside. *Seeds:* all, tier G (REF 7.5's range). *Band:* none.
*Source:* internal (`spec-money-supply.md` 8.2 items 1 and 2). *Confidence:* I.
*Class:* X, tier G. *Loops:* STB-L30.

**STY-M02 The ledger's identities hold.** *Property:* goods are conserved except
by recipes, consumption, spoilage, wear and disposal; money by named flows; the
production and expenditure views of GDP are equal; the regional money identity
holds with its reconciling items. *Measure:* the identities of IND 7 and 8.2
(FR-IND-016 to 019) over every **tick** and **run**; the invariants of REF 7.5.
*Pack:* every pack of REF 7.5, `WithAudit`. *Seeds:* all of REF 7.5's sets, tier
G. *Band:* none. *Source:* internal (testing.md 6.3, FR-LED-*, FR-IND-*).
*Confidence:* I. *Class:* X, tier G.

**STY-M03 In a closed upkeep-only economy the aggregate ratio is 1.** *Property:* with
no demand beyond upkeep, no faucet and no outside, the household-weighted mean
welfare ratio is 10 000 up to the accounting of stock changes. *Measure:* the
**aggregate ratio** `sum income / sum cost` of `HouseholdIncome` (IND 9 row 21,
7.7.2: the quantity for which the identity was derived, N-054) over the **year**
window; `OK` is false until OBS adds the fields. `WelfareDistribution.Mean` with
`RatioHouseholds` is **recorded beside it** and is not bound by the identity: it
differs from the aggregate by the covariance of the ratio with the cost (IND
7.7.2). *Pack:* `one` with the halt on and
off (S19), any variant with no demand overlay. *Seeds:* SR2, tier G.
*Band:* `s19.ratio_band_bp` prior 500 (P). *Pass and fail:* the identity is exact
for flows in a window with holdings unchanged on average, so a miss measures the
change in stock over the window; a mean far above 10 000 in such a pack means
money or goods entered by a path the identity excludes and is a **finding for
LED**. *Source:* hand derivation (`spec-viability.md` 4.7; `spec-money-supply.md`
8.2 item 4; N-054). *Confidence:* H. *Class:* **R** while `HouseholdIncome` is `OK` false, then B,
tier G. *Loops:* STB-L46, L44.

**STY-M04 Wants lift the ratio, and not beyond their ceiling.** *Property:* with
the demand module, the aggregate ratio of a closed economy is above 10 000 and at
most `10 000 * (1 + Wmax / U)`. *Measure:* as STY-M03 (the aggregate, not the
mean of ratios), in `one-demand`, and the
ceiling the validator computes (`target_above_demand`, `spec-demand.md` 7); with
the target above the ceiling the glut stays bounded (STY-M05). *Pack:* S24.
*Seeds:* all, tier G. *Band:* none: both ends are derived (`spec-demand.md` 7:
`1 + W / U` expected, `1 + Wmax / U` at most; the model's 1.53 at target 15 000 is
for a wider ceiling than `one-demand`'s 1.32 and is not a band). *Source:* hand
derivation and model. *Confidence:* H. *Class:* **R** until the aggregate exists, then O, tier G.
*Loops:* STB-L46, L57.

**STY-M05 Nothing is hoarded and the unsold stock is bounded.** *Property:* in a
no-shock run the share of money held out of circulation stays below a band, and
with the production halt on the unsold stock of each produced good stays near the
halt's threshold, while with it off the stock grows past it. *Measure:*
`Sectors` share `(M_p + M_c + M_t) / M`; `StockCover` and the unsold share
`HeldByEstate / Held` (IND 9 rows 7 and 15); S19's stock as a share of the halt
threshold. *Pack:* `one` with the halt on and off (S19), the glut pack of S07.
*Seeds:* SR2, tier G. *Band:* `m05.pool_share_bp` prior 3 000 (P);
`s19.stock_bound_bp` prior 15 000 of the halt threshold (the threshold is 30 000
bp of one tick's target; 150% allows the job in progress to complete: H for the
shape, P for the size). **O**: with the halt off the stock of at least one good
passes the bound (the measure can see the glut, FR-STB-021). *Source:* hand. 
*Confidence:* H, P. *Class:* B and O, tier G. *Loops:* STB-L44, L46, L11.

**STY-M06 A poor start does not reach the reserve in a closed economy, and the
world says so.** *Property:* in a closed upkeep-only economy the mean money per
household of a region cannot rise, so a start below `ReserveTicks * U_c` is
reached by a group only at another's expense, and the world reports the
unreachable reserve. *Measure:* the mean money per household of each group
(`Sectors`, `M_h / H`) and the first tick it reaches `ReserveTicks * U_c`
(3 000 in `one`); the warning `reserve_exceeds_money` of FR-AGT-060, which
validation raises when a start is below the reserve (FR-STB-022 says "the world
reports the unreachable reserve": this is the report, and it is made at
construction, not per run). *Pack:* `one` + `overlay/poor`
(S20). *Seeds:* all for the O part, tier G. *Band:* **X**: the warning
`reserve_exceeds_money` is raised at construction for `overlay/poor`; **O**: the
region's mean cannot exceed its start (`spec-money-supply.md` 8.2 item 3: it
changes only by `H` or by money moving to the pool); **B**, tier N until REF has a pack in which a group can start below the reserve
and earn: a group that is a net earner (its mean money is above its start at
the end) reaches the reserve within `s20.ticks`, prior 120 (P). The B part can fail: the warning does not
excuse a net earner that never reaches the reserve. *Pass and fail:* a net earner
below the reserve at `s20.ticks` fails the fact (FR-STB-022).
*Source:* hand derivation. *Confidence:* H. *Class:* X and O, tier G; B, tier N
(waits for a pack, section 21). *Loops:* STB-L45, L11.

### 5.6 Trade

**STY-T01 The outside's accounts close.** *Property:* with an outside node, the
units and money across the boundary satisfy the identities of EXT 8.2 and the
counterparties are empty at every boundary. *Measure:* `TradeBalance` (IND 9 row
13), the six reasons of `BoundaryFlow`, the Outside accounts at the boundary.
*Pack:* the S55 invariant run, `WithAudit`, a range of seeds. *Seeds:* all, tier
G. *Band:* none. *Source:* internal (`spec-external-sector.md` 8.2, 5.3, 16;
N-115). *Confidence:* I. *Class:* X, tier G. *Loops:* STB-L96, L98.

**STY-T02 Trade moves occupation toward comparative advantage.** *Property:*
with trade on, the occupation shares of each region move toward the pooled
solution's census. *Measure:* the sum over roles of the absolute difference
between a region's `RoleShares` and the share the pooled solution of VIA assigns
(`spec-viability.md` 9, the pooled census), at the start and end of four years,
and the best-yield region's own activities. *Pack:* `three` + `overlay/trade.toml`
(X2 dynamic). *Seeds:* SR2, tier N (four years, 1 460 ticks, waits for the overlay). *Band:* **O**: the distance at the end
is below the start; **B**: `x2.share_band_bp` prior 1 500 (P). *Pass and fail:*
a distance that grows means specialisation does not follow the yields, a finding
for MIG, SPN and MER. *Source:* hand (VIA A3: the activities active in a strict
subset of regions). *Confidence:* H. *Class:* O and B, tier N. *Loops:* STB-L36,
L84, L108.

**STY-T03 A quote pulls the price and does not peg it.** *Property:* with an
outside corridor at the default midpoint clearing, the realm's price is pulled
toward the quote and the outside keeps the improvement. *Measure:* the share of
ticks of the **year** window in which the market price is within
`t03.quote_band_bp` of the line's quote while the side is present
(`OutsideTraded`, `MarketCleared`), and the price range with and without the
outside (S51). *Pack:* the S51 pack. *Seeds:* median, tier G. *Band:* none: this
is the **decision input of N-113**. If the share is below `t03.pull_bp` in a
setting that wants a fixed price, CLR adds the flag and rule of N-113; the priors
are `t03.quote_band_bp` 1 000 and `t03.pull_bp` 5 000 (P). *Source:* none.
*Confidence:* none. *Class:* R, tier G. *Loops:* STB-L21, L96, L99.

### 5.7 The warm start

**STY-H01 A warm start keeps its prices near their seeds.** *Property:* for the
first 30 ticks (the welfare window) each price of a warm run stays within a band
of its seed, and a cold run does not do better. *Measure:* the maximum over the
first 30 ticks of `|price - seed| / seed` per market (`MarketCleared.Price`
against the table), warm and cold; window: the 30 ticks named. *Pack:* `one` warm
and cold (A1), every variant. *Seeds:* SR2 for the band, all for the
order, tier G. *Band:* `a1.price_band_bp` prior 1 500 (P; no source says how far
a seeded belief wanders in 30 ticks, and the belief step bounds one move at 400 bp
of the centre, `spec-beliefs.md` 5.3). **O**: the warm run's maximum deviation is
at most the cold run's. *Source:* internal requirement (`spec-viability.md` 13.2
A1). *Confidence:* P, I for the order. *Class:* B and O, tier G. *Loops:*
STB-L53, L22.

**STY-H02 A warm start stays near the steady state.** *Property:* a viable pack
started from its warm start, with no shock, does not leave the neighbourhood of
the steady state over the windows after the transient. *Measure:* per market the maximum over the
**year** windows of `|Offset|` against the pack's steady-state price; the least
households over the run; `Sectors` `M_h / H` against its start; `RoleShares`
swing. *Pack:* `one`, `two`, `three`, `five`, `second`, with the population
module **off** (5.9, C5), on `one` and `second` today and on the others when REF
runs them 1 095 ticks (see D01); the famine pack's nominal and full-labour tables are
run side by side as **R** (N-085). *Seeds:* SR2, tier G. *Band:*
`h02.offset_bp` prior 2 000, `h02.money_bp` prior 2 000 (P), `x4.households_bp`
(STY-G01), and STY-D03's swing. *Pass and fail:* outside the band without a
regime fails the fact; leaving it for good is **R8** or **R6**. *Source:*
internal requirement (`spec-roadmap.md`, STY row: "a warm start that stays near the
steady state"). *Confidence:* P. *Class:* B, tier G. *Loops:* STB-L53, L02, L04.

**STY-H03 A warm start does not hide an instability.** *Property:* a perturbed
chain started from its (stale) warm table and from a cold start ends in the same
regime. *Measure:* the regime classification of 6 in the last year of each run.
*Pack:* `one` + `overlay/perturbed`, with `warm-one` and without (S22). Both legs are **perturbed** chains: S22 is in neither the healthy set H of G01 and D01 nor of 6.5.3, and its warm (stale-table) leg is R8's provoked run there. *Seeds:*
all, tier G. *Band:* none. *Pass and fail:* two different classes fail the fact
(FR-STB-024); the two paths of the first 100 ticks are recorded. *Source:*
internal requirement. *Confidence:* I. *Class:* O, tier G. *Loops:* STB-L53,
L14, L21.

**STY-H04 A warm start does not synchronise the replacement of durables or the
completions of jobs.** *Property:* the replacement orders of a durable are spread
and the completions of a long job are staggered. *Measure:* the number of
`OrderPlaced` bids for `tool_b` per tick, counted by a recording observer in
`_test.go` (FR-REF-027; the tally ignores `OrderPlaced`, IND 4.2.3), and the
ratio of its maximum to its mean over ticks 0 to 119, warm and cold (S17); the completions per tick of `refine` and
`make_tool_b` over the first 3 ticks (S18). *Pack:* `one` warm, `one` cold as the
control. *Seeds:* all, tier G. *Band:* `s17.spike_factor` prior 30 000 (a maximum
at most 3 times the mean, P) and `s17.control_gap_bp` prior 5 000 (the cold ratio
larger by 50%, P); **X**: the stagger of S18 holds (no tick above twice the
mean, VIA A8). The remaining durability of the items at tick 0
(`8 - floor(i * 8 / n)`, FR-VIA-038) is VIA's exact test and **not** STY's: no
snapshot tracks it (IND 4.5.5). *Source:* internal requirement (FR-VIA-037,
FR-VIA-038). *Confidence:* I for X, P for B. *Class:* X and B, tier G. *Loops:*
STB-L22, L26, L27.

### 5.8 The umbrella fact

**STY-G01 A healthy start stays healthy.** *Property:* every shipped pack that
passes the viability check, run from its warm start with no shock for the length REF gives it (1 095 ticks:
three **year** windows counted from tick 0, the first included, and two links;
4.1.2),
is classified by every detector of section 6 as `none` in the last year, and its
households stay near their start. *Measure:* the detectors R1 to R9 (nine regimes, ten rows) over the
**year** windows; `Households` against the start (`x4.households_bp`). *Pack:*
every variant of REF 4.1 except `famine`: asserted today on `one` (1 095 ticks,
S19) and `second` (X4); `two`, `three` and `one-demand` (400 ticks, or
none, in REF) and `five` (400 ticks, seeds 1 and 2: one year window) **need a
longer run**, owed REF 8.2 rows: 1 095 ticks on seeds 1 to 4 for `two`, `three`
and `one-demand`, and on seeds 1 and 2 for `five` (about 58 s a seed at REF's 53
ms a tick, 116 s for the two). *Seeds:* all of 1 to 10 (a regime in a healthy start is a defect),
tier G; SR1 at every tier (8.3). *Band:* `x4.households_bp`
prior 9 000 (households at least 90% of the start; P); the detectors' thresholds
are `regimes.toml` (6.5). *Pass and fail:* any detector `detected` fails the
fact; `unknown` (insufficient windows, or stale prices) is **not** a pass and is
reported. *Source:* internal requirement: `vision.md` 3 goal 1 and 5 item 1.
*Confidence:* shape I, size P. *Class:* B (over the regimes), tier G/N. *Loops:*
all of STB's regimes (6.6).

### 5.9 Facts that cannot both hold in one pack

The brief asks that two facts that cannot both hold be said, and where. Each is
a place where a pack that asserts both would fail by its construction, not by a
defect.

| # | Facts | Why they cannot both hold | What a pack does |
|---|-------|---------------------------|------------------|
| C1 | STY-M03 (aggregate ratio 1) and STY-W02, STY-G01 with the population module on | with upkeep as the only demand the aggregate ratio is 10 000 (IND 7.7.2: the mean of ratios is not bound by it but cannot be expected above), below the default replacement ratio 12 000: households fall to the floor or sit at chronic hunger (`spec-population.md` 5.3, "both are traps") | a pack with the population module has the demand module or a faucet, or `r*` at or below 10 000; the warning `replacement_above_closed_ceiling` is read as a rejection by acceptance (section 21) |
| C2 | STY-P02 (not pinned) for a **staple** and the labour loop of N-028 | a staple's elasticity is below 1, so the loop is unstable and ends pinned at the price the work floor sets (`spec-stability.md` 6.1; `spec-agents.md` 11.6) | the staple's pinned outcome is **recorded** by S04 with the floor at its default and at 0; until S04 has run its pinned-share key stays unset (a prior, not a promise); the pack is rejected (F5) only if the pinned run exceeds the band after review |
| C3 | STY-P03 (no cobweb) and a seasonal swing | a window shorter than a year reads a seasonal swing as a range | **year** or **season** windows only (4.1.3); the model of 16 shows the two readings disagree |
| C4 | STY-D01 (almost no hunger) and `famine` | `famine` has a labour share to subsist of 76%, above what `ExpectedWorkTicks` allows (REF 4.4.5) | STY-D01 is not asserted on `famine`; STY-S02 and S06 are |
| C5 | STY-H02 (households stay near the start) and STY-W02 (households approach `N*`) | a warm start's census is not the Malthusian `N*` (`spec-population.md` open question 3); with a fixed factor the households drift toward `N*` | H02 is asserted with the population module off; with it on H02 asserts prices and money only and W02 governs households |
| C6 | STY-P06 (convergence within a margin) and identical shippers with no spread | the INF model leaves a gap of 1.8 of the freight for identical shippers, 0.6 for an independent error and 0.04 for a spread of thresholds (`spec-information.md` 11.2; units of the freight, which P06's band, a share of the delivered price, converts only for a stated freight share) | P06 is asserted with the lane lock and the fee-share spread of MER on (D79, D81); S67's runs without them are **expected to fail** and say so |
| C7 | STY-P04 (offset) and STY-S05, STY-T03 (a faucet or an outside) | the closed steady state is not the price when money enters or a quote pulls (N-114) | P04 is asserted without a faucet and without an outside; with them it is **R** |
| C8 | STY-M03 and STY-M04 | the first is exactly 10 000, the second above it | they are asserted on different packs: with and without the demand overlay (5.5) |

### 5.10 The modules each fact needs

"On" means the module's entry is in `Config.Modules` (`spec-config.md` 3.2);
"off" means it must be absent for the fact to be the fact. A core mechanism
(clearing, beliefs, agents, spawner, estates' pool when `[modules.estates]` is
present) is not listed unless the fact names it. *Trade set* is the region
graph's edges with transport, information and the merchant strategy
(`overlay/trade.toml`).

The fourteen modules of the later stage are off for every row unless its *On* column
names them, and a binding control or right is absent where the *Off* column says so
(4.6).

| Fact | On | Off | Pack |
|------|----|-----|------|
| P01, P03, P08, P09 | none | none | any |
| P02, P05 | none | a binding control or right (4.6) | any |
| P04 | none | external, money flows, a binding control or right (4.6) | any viable |
| P06 | trade set | a binding control or right (4.6) | `two`, `three`, `five` |
| P07 | none | population, money flows, external | `one`, `second` |
| W01 | demand | none | `one-demand`, `five` + demand |
| W02, W03 | population, resources (binding), demand | `Renewal` 0 | `one-demand` + `overlay/resources.toml` + population overlay |
| W04 | skill barriers | none | `one` + `overlay/skill` |
| W05 | none | none | `one` |
| W06 | population (resources for the fixed point) | migration, money flows (for the money cap) | the population overlay |
| W07 | employment | none | `one` + `overlay/employment.toml` |
| S01 | none | a binding control or right (4.6) | `one` |
| S05 (accounts) | money flows | none | `one` + `overlay/faucet` |
| S02, S06 | estates (pool, relief, ladder) | none; population for the tier L leg | `famine` + `overlay/estates` |
| S03 | trade set | a binding control or right (4.6) | a three-region chain + trade overlay |
| S04, T01, T03 | external | none | `overlay/external.toml` |
| D01, D02, D03 | none | none | any viable (not `famine` for D01) |
| M01, M03 | none | money flows, external, demand (M03) | `one` |
| M02 | any | none | the packs of REF 7.5 |
| M04 | demand | none | `one-demand` |
| M05, M06 | none | money flows (M06) | `one`, `one` + `overlay/poor` |
| T02 | trade set | none | `three` + trade overlay |
| H01, H03, H04 | warm start | none | `one` warm and cold |
| H02 | warm start | population | `one`, `two`, `three`, `five`, `second` |
| G01 | the pack's own | none | every variant but `famine` |

### 5.11 Where each scenario's assertion lives

FR-STB-005 asks that every scenario of `spec-stability.md` 8 be covered, and the
brief asks that a scenario's bands have one home. The table gives, for every
scenario S01 to S165 (S15 is unused) except those of the class "harness test"
(S77, S78, S84 to S90 and the list below the table), the **fact** that holds its band, or the
class of what it asserts when no band is involved. A scenario with an assertion of
its own owner (an exact check, or a formula the owner derives) keeps it; STY adds
only the key and the prior where the scenario named "a stated band" and no fact
above holds it. New keys are `s<nn>.<name>` in the style of REF's; every prior is
a placeholder (P) unless a basis is given.

| Scenario | Held by | Class | New key and prior |
|----------|---------|-------|-------------------|
| S01 | STY-S01, STY-P04 | O, B | |
| S02 | exact (`n <= ceil(g / Delta)`, FR-BEL-034) | X | |
| S03 | STY-P02 (`s03.pinned_ticks`); the `Warn` at the tenth tick is exact | B, X | |
| S04 | STY-P02, STY-W05 | B, R | |
| S05, S12 | STY-P03 | B | |
| S06 | STY-D03, STY-W04 | B, O | |
| S07 | STY-P02 | B, R | |
| S08 | STY-P05; the cycle gain below 1 is exact | O, B, X | |
| S09 | exact (the price reaches the ceiling and no further; one `Warn` per ten ticks) | X | |
| S10 | STY-P09 | X, B | |
| S11 | exact (no relief and no unfed grower; the window is 120) | X | |
| S13 | STY-S05 | X, R | |
| S14 | STY-S02, STY-S06 | O, B, X | |
| S16 | STY-P02 (`s16.pinned_share_bp`) | B | |
| S17, S18 | STY-H04 | X, B | |
| S19 | STY-M03, STY-M05 | B, O | |
| S20 | STY-M06 | O, B | |
| S21 | exact (every seed inside the cost band, FR-VIA-027) | X | |
| S22 | STY-H03 | O | |
| S23 | STB-L14 for a luxury | O, B | `s23.band_bp` 1 000: the price within 10% of the reference after the transient; **O**: it converges only when the schedule elasticity is above 10 000 and the headroom about 10% (`spec-demand.md` 6.3, a model: M); the lower edge of the basin is **R** |
| S24 | STY-M04, STY-M05 | O, B | |
| S25 | exact (no want bought below the reserve); the swing of the shares | X, B | `s25.swing_bp` 2 000: the share of an alternative swings at most 20 points over a year |
| S26 | STY-D03 for the share; the changes per tick are exact (FR-STB-029) | X, B, O | `s26.share_band_bp` 500; with the caps off the swing is larger (O) |
| S27 | expects R3; no entrant is made and `NoViableRole` shows (exact) | X, O | |
| S28 | STY-S03 | O, X, B | `s28.band_bp` 1 500, `s28.reopen_ticks` 180 |
| S29 | thin markets of the small regions reported; the three prices' dispersion | R, B | `s29.disp_bp` 2 000 once trade is on |
| S30, S46, S67 | STY-P06, STY-P03 (R6 over the `Spread` series); the gain bounds of L84 and `spec-merchants.md` 7.3 are derived by their owners | O, B | |
| S31 | exact (the two cases of TRN-A5, delivery or not at the two sides of the boundary between the travel time and the shelf life) | X | |
| S32 | STY-P03 | B | |
| S33, S36 | exact (the class stays below `m*`; the net of a pair is the difference of its amounts times ticks; only the named flow changes) | X | |
| S34 | with `Exempt` no household goes unfed because of the levy (O); without it the warning and the count | O, X | |
| S35 | exact (the thresholds are crossed at the derived ticks) | X | |
| S37, S41 | STY-W02, STY-W06 | O, B | |
| S38 | STY-W03; expects R3 for the pack whose role disappears | O | |
| S39 | STY-W06 (`w06.money_cap_bp`); `BirthsUnfunded` non-zero | B, O | |
| S40 | the first pack dies out or is flagged (expects R3); the second sits at a bounded hunger the ladder's counters show; `Renewal` ignored (exact) | O, X | `s40.hunger_bp` 3 000: the unfed share at the fixed point (the closed ceiling of `spec-population.md` 5.3) |
| S42 | the stock holds below the maximum sustainable draw and collapses above it; a refuge recovers (derived, `spec-resources.md` 8.3); the basin edge is **R** | O, R | |
| S43 | the run-down time agrees with `endowment_runway`; expects R3 after exhaustion | B, O | `s43.runway_bp` 1 000 (the 10% of the scenario) |
| S44 | the grant per household within a band of the ration; no agent denied for longer than a run | B | `s44.grant_band_bp` 2 000 of the mean (the model's 0.416 to 0.581 of demand at half supply, `evidence.md` row 142: M); `s44.denied_ticks` 30 |
| S45 | exact (the identity of `spec-resources.md` 6.4 each tick); a `refill` stock recovers, a `logistic` one with no refuge does not (derived) | X, O | |
| S47 | a merchant that ignores `SourceAge` ships on a stale rumour and one that reads it does not | O | |
| S48, S49 | exact (the rumours cut off while cut off; bit-identical resume) | X | |
| S50 | **R**: the shift of the rumoured gap against the true gap by the lag is a measurement of the signal | R | |
| S51 | STY-T03 | R | |
| S52 | STY-S04 | O, R | |
| S53 | **R**: the realm's lag behind the mid, and the warning `walk_faster_than_beliefs` | R | |
| S54 | exact (`OutsideEnded` at the cap); the export cliff and the glut are **R** | X, R | |
| S55 | STY-T01 | X | |
| S56 | the unfed household-ticks fall when the floor covers the upkeep (O); the size is **R** | O, R | `s56.unfed_drop_bp` 9 000 when the floor covers the upkeep (the model of `spec-subsistence.md` 11.2: 97% to 100%, M) |
| S57 | households unfed, removals and the pool's balance are lower with the mode (O) | O | |
| S58 | exact (rung 1 abandons an own-use job only when `Duration - 1` reaches `ChangePrivation`; the warning fires for exactly those) | X | |
| S59 | exact (the ration holds); the cohorts and the units get the same mean (O) | X, O | |
| S60, S66, S72 | exact (the audit, the bit-identical resume, the reconciliation) | X | |
| S61 | the household count settles: R6 on it `none`; the pipeline cap is never exceeded (exact); the journeys within a factor of the minimum | O, X, B | `s61.journeys_factor` 20 000 (twice) |
| S62 | the arrivals divide between equal destinations within a band and the difference does not alternate in sign | B, O | `s62.split_band_bp` 2 000 |
| S63 | the journeys differ by less than a band between relief on and off | B | `s63.journeys_band_bp` 1 000 |
| S64 | exact (`unprovisioned` counted, the `Warn` at the tenth tick) | X | |
| S65 | no role of the chain is detected R3; with `Fixed` nothing leaves (exact) | O, X | `s65.role_share_bp` 5 000 of its start |
| S68, S71 | exact (the contract table, conservation with escrow, the audit; the node is named by no shipment) | X | |
| S69 | a merchant with `MaxSourceAge` 2 ships nothing on a stale rumour; one with the bound at its maximum ships and arrives without moving the price | O | |
| S70 | the mode switches at the haul price the rule of `spec-merchants.md` 6.1 gives | O | |
| S73 | tier S, no seeds, a world built in Go (WLD-S1): the admissions of each of 100 tied candidates under a cap of 5 over 1 000 ticks, as a share of its expected share; none below half or above twice it, a bound the keyed tie of `spec-world.md` 10.1.1 derives (O); with the lower ID first the first five take all admissions, the control (X); the nightly flag of STY-A4 reads L133 | O, X | none, **reported, no band**: the half and the twice are WLD's derived bound and not a measured band, so no run sets them |
| S74 | tier S, no seeds (WLD-S2): the reviews per tick of 1 000 and of 5 000 agents at periods 10 and 30 over every stride of 1 to 31 and three starting IDs; no tick above `ceil(1.2 * n / P)` (O, `spec-world.md` 9.1.1); below about a thousand agents the load is **reported** and no bound is claimed | O, R | none, **reported, no band**: the 1.2 is the rule's own bound with its derivation in `spec-world.md` 9.1.1, and the small-group burst is a property of the period (L119) |
| S75 | the warm leg is STY-H04 (`s17.spike_factor`, `s17.control_gap_bp`) as for S18; the cold leg, tier G, is **R**: the completions per tick of the first 40 ticks of a chain with a `Duration` 10 recipe without the warm start, a pulse that is measured and recorded; the warning `cold_start_lockstep` is raised for the cold run (exact); R7 reads the same series and is not asserted here | X, R | none, **reported, no band** for the cold leg: the pulse is the design of a cold start (L22) and a band on it would assert a property that the library has by construction; the warm leg's keys are STY-H04's |
| S76 | tier S for the unit model and tier G for the two runs of `one` that differ by one draw in the first tick (WLD-S4): the share of later draws that differ is **reported** (the model gives 98.0% for one stream and 2.0% for a stream per tick); a paired comparison is claimed only for a change that moves no draw count (L120, L124) | R | none, **reported, no band**: the share is a property of one positional stream (`spec-world.md` 7.6) and is read by EXP's paired designs (`spec-experiments.md` 7) |
| S79 | STY-P01 (every published price inside the belief bounds, each tick) and STY-M02 (goods and money conserved each tick), both tier G and exact; the recovery of the shocked region's staple price is STY-S01's measure (the ten-tick mean back within `s01.band_bp` of `P0`, 5.3) after `SetYieldShocks` of 3 000 on the staple recipe of one region of `three` or `five` for `k` ticks, then cleared | X, B | `s79.recover_ticks` 365 (P): ticks after the shock clears; the staple job starts only in spring (REF 4.2), so a shock that clears after a spring shows in the harvest of the next one; no other basis |
| S80 | STY-S03 (the gap opens while the edge is closed and closes after the reopening), tier N, with S28's two keys and no new one: the same fact on a pack of three or more regions; a consumer's rule that scales a flow from a published price shows in `MoneyFlowScaled` and in the flow's next tick (exact; it is not a claim that the rule is stable, L121) | O, X, B | `s28.band_bp`, `s28.reopen_ticks` (STY-S03's, 5.3) |
| S81 | tier G, exact: the same seed, configuration and calls give the same state at every tick and the same events (the golden rendering of REF 8.5); two orders of calls on different keys give the same state; a run with `AddAgent` shows `initial` in `BoundaryFlow` and passes the audit (STY-M02) | X | none: an exact check, asserted whether or not a band exists (FR-REF-023) |
| S82 | tier G, exact: the mailbox, the slots and `Agent` of the restored world equal the original's and the result equals an uninterrupted run (9.5, FR-STY-088); O: a non-mortal player at hunger stays and a mortal one is removed by a consumer's spawner | X, O | none: exact and ordinal checks, asserted whether or not a band exists |
| S83 | tier G, exact: a `Relocate` of a household on a journey moves it and ends its journey (`Ended` counts it); one with an open contract is refused with `ErrRegionBound`; goods and money are conserved (STY-M02) | X | none: exact checks, asserted whether or not a band exists |
| S91 to S165 (the fourteen later modules, REF 7.12) | the owner's own assertion in the scenario table of its spec (exact accounting, an invariant or a derived bound), asserted whether or not a band exists (FR-REF-023); the measures that spec reports stay reported until a reviewed run sets a key (8.4); the pass of PLC-S2 (S156) is a band for STY to set: the unfed share returns inside the band | X, O, R | none yet |

A scenario that is **exact** is asserted whether or not any band exists
(FR-REF-023), so STY has nothing to set for it. The scenarios that **expect a
regime** (S14's deep depth, S27, S38, S40, S43) name it in their band table
(6.4).

**Scenarios of the class "harness test"** (S77, S78 and S84 to S90, and S101, S105,
S106, S112, S113, S119, S125, S132, S141, S142, S147, S148, S153, S155, S158 and S159 of
`spec-reference-chain.md` 7.12, of `spec-stability.md` 8; FR-STB-005) are outside this table and outside the coverage
rule of FR-STY-098. Each is a test of a harness or of a procedure against a model of
its own rules and asserts nothing about prices, households or money, so no fact of
this spec holds it and no band exists for it; it is covered by a test in the spec that
owns the harness (`spec-checkpoint.md` 15, `spec-experiments.md` 17,
`spec-calibration.md` 11 and the specs of the later modules named in REF 7.12).

## 6. The failure regimes

A **failure regime** is a way a run goes wrong that a stability test must be able
to recognise (STB 7). STB defined eight and left every threshold a placeholder
(STB open question 3: "STY sets them from the first run of the reference chain").
This section defines each as a **detector**: a pure function of series IND
produces, with a threshold, a window and the loops it exposes. The thresholds are
**priors**, and a prior detector asserts nothing until its keys are set by the
separation protocol of 6.5.

### 6.1 The form of a detector

6.1.1. **A detector is a pure function** `Detect(series, thresholds) ->
Verdict`. Its input is plain series of integers (`{Tick, Value}` slices that the caller
copies from the result fields of the presets of IND 9 over successive windows, so
the package imports nothing of `indicator`), a `rate` argument for R3 (the speed
of STY-S02, computed by the caller from the pack) and the keys of `regimes.toml`. It takes
no world, no snapshot and no event, draws nothing and reads no clock; it is the
same code for a synthetic series in a unit test and for a series of a run
(position 9, FR-STY-080).

6.1.2. **A verdict** has three classes and a record:

| Class | Meaning |
|-------|---------|
| `none` | the series is long enough, its blind-spot fields are inside their bounds and the rule did not fire |
| `detected` | the rule fired; the verdict carries the **onset** (the first window or tick at which it did), the measured value and the threshold |
| `unknown` | the rule could not be evaluated: fewer windows than it needs (`min_windows`), a `Stale` share above `stale_max_bp` where the rule would otherwise return `none` (R1, R9), a threshold key unset, `Active` below `active_min_bp` (R6 and R8 only, 4.3), a module or reference the rule needs absent (R8 with no steady-state price) |

`unknown` is **not** `none`: a test that needs a verdict of `none` (STY-G01) fails
on `unknown` and says why. Silence is not health (IND-A3).

6.1.3. **Windows.** A detector reads whole **year** windows counted from
tick 0 (4.1.2: the first year included, warm or cold) except where its row says
one-tick windows, because a detector over a window that
is not a whole year reads the seasonal swing as a regime (4.1.3). A series is
`min_windows` year windows long at least (prior 3 counted from tick 0: two
links; REF's 1 095 ticks give exactly three, with no slack). `min_windows` does **not** apply to R5a, R5b or R7's completions test, which
read one-tick windows: each needs a series at least as long as its own count key
(`pinned_ticks`, `freeze_ticks`, `first_ticks`) and returns `unknown` for a
shorter one, so S16's 400-tick provoked run is classifiable. A detector whose rule
needs more than the run has says **needs a longer run** (`unknown`), and the
owed REF row (section 21) names the run.

### 6.2 The detectors (nine regimes, ten rows)

Each rule is stated in integers: indices in basis points of 10 000, a share in
basis points, a count in ticks or windows. `k` indexes the **year** windows counted from tick 0
(4.1.2).

| ID | Regime (STB) | Rule | Keys (prior) | Loops it exposes |
|----|--------------|------|--------------|------------------|
| **R1** | runaway inflation (F1) | `g_k = link_k - 10 000` of the `PriceIndex.Fisher` chain with `Consumed` weights, of the whole economy and of each class of `Classes` (any one detected is detected), is at least `rise_bp` in each of `windows` consecutive windows; **or** `Sectors.M` is at least `m_factor_bp / 10 000` times `M_init` at two successive closes; **or** any link is `Clamped` | `r1.rise_bp` 3 000, `r1.windows` 2, `r1.m_factor_bp` 30 000 (P) | L05, L10, L14 (upper outcome), L30 (a faucet with no sink), L69, L98 |
| **R2** | hoarding (F2) | trade volume falls (`Turnover.Volume` of the last window is below `vol_bp` of the first) **and** one of: `(M_p + M_c + M_t) / M` is at least `share_bp` at `windows` consecutive closes; the unsold share `HeldByEstate / Held` of a commodity with consumption is at least `unsold_bp` and rising over `windows`; `Inequality.Top(1 000)` of household money is at least `top_bp` | `r2.vol_bp` 8 000, `r2.share_bp` 5 000, `r2.unsold_bp` 5 000, `r2.top_bp` 7 000, `r2.windows` 2 (P) | L31, L11, L25, L45, L30, L73 |
| **R3** | collapse (F3) | the households of the realm, or of a region with a positive start, fall to at most `10 000 - fall_bp` of their start **and** are not back at `10 000 - recover_bp` within `horizon_factor / 10 000` times `t_min` ticks of the minimum (`t_min`: the speed limit of STY-S02, from the trough to `10 000 - s14.recover_band_bp` of the earlier level, the same figure and the same level as R3's `recover_bp`; the two priors are 1 000 and a registry check keeps them equal; at least `horizon` ticks); **or** a role with positive start households has none for `role_ticks` consecutive ticks; a run too short for the horizon is `unknown` (**needs a longer run**) | `r3.fall_bp` 2 000, `r3.recover_bp` 1 000, `r3.horizon_factor` 50 000 (5 times: above S02's band of 4), `r3.horizon` 730, `r3.role_ticks` 365 (P) | L39, L35, L19, L11, L62, L63, L64, L66, L77, L82, L88 to L90, L101, L104, L105, L111 |
| **R4** | stagnation (F4) | `Turnover.Volume` is below `vol_bp` of the first window in each of `windows` consecutive windows **while** households are above `10 000 - fall_bp` of their start (no collapse) | `r4.vol_bp` 5 000, `r4.windows` 2 (P) | L03, L09, L11, L45, L46, L73 |
| **R5a** | pinned market (F5) | in one-tick windows, `Pinned.Share` of a market is at or above `pinned_bp` for at least `pinned_ticks` consecutive ticks | `r5.pinned_bp` 5 000, `r5.pinned_ticks` 60 (P) | L05, L06, L09, L14, L51, L86, L91, L21, L102, L106 |
| **R5b** | frozen market (F5, second half) | in one-tick windows a market has unmatched units on both sides and no volume for at least `freeze_ticks` consecutive ticks | `r5.freeze_ticks` 60 (P) | L09, L03, L91 |
| **R6** | cobweb blow-up (F6) | the `PriceRange` ratio of the **season** or **year** window exceeds `factor` and is above the previous window's by more than `growth_bp` in `windows` successive windows; **or** a role's `RoleShares` swing is at least `swing_bp` in two windows | `r6.factor` 30 000, `r6.growth_bp` 2 500, `r6.windows` 2, `r6.swing_bp` 1 000 (P) | L21, L36, L04, L58, L61, L84, L85, L93, L97, L100, L107 to L109 |
| **R7** | synchronisation, lockstep (F7) | the window's lowest `Dispersion` of a market is below `disp_bp`; **or** in one-tick windows of the first `first_ticks` ticks the completions of a recipe on one tick exceed `spike_factor / 10 000` times their mean | `r7.disp_bp` 50, `r7.spike_factor` 20 000, `r7.first_ticks` 60 (P) | L13, L15, L22, L34, L35, L59, L92, L93 |
| **R8** | offset (F8) | the absolute value of `Offset` of a market is at least `offset_bp` **and** the `Unfed` share of household-ticks of the window is at least `unmet_bp` | `r8.offset_bp` 3 000, `r8.unmet_bp` 3 000 (bp of household-ticks unfed, as `s01.unmet_bp`; P) | L02 |
| **R9** | deflation with distress (**new**, STB-F9) | `link_k` is at most `10 000 - fall_bp` in each of `windows` consecutive windows **and** the later window's `Turnover.Volume` is below the earlier's | `r9.fall_bp` 3 000, `r9.windows` 2 (P) | L70, L71 (a sink or a levy with no paired grant), L98; the floor of the reference band |

Shared keys: `stale_max_bp` prior 1 000 and `active_min_bp` prior 5 000 (4.3),
`min_windows` prior 3 (P).

### 6.3 Reading the rules

6.3.1. **Why R9 is new.** STB's F1 is inflation only. ADR 0001 and the roadmap
say a closed economy that grows "deflates", and `spec-money-supply.md` 8.4 says
what that means: the mean price falls exactly when real volume grows by more than
velocity does. A fall of the index with a **growing** real volume is that
pressure and is not a failure (a population that grows on a fixed `M`, MON 8.6).
R9 therefore fires only with a **falling** volume: a price that falls because
nobody buys. The reference band's floor (`RefBandLow`, 1 000 bp of the reference
price) is the lowest the price can reach, and a market at it is also R5a.

6.3.2. **R2's first conjunct.** The share of money outside the households is a
regime only with trade falling; a pool that fills in a collapse is working as
designed (`spec-population.md` 6.2: it is fullest when need is greatest), and that
run is R3, not R2. The share is `(M_p + M_c + M_t) / M`, every sector but the households, with `M = M_h + M_p + M_c + M_t` (IND 9 row 7, which lists pool, escrow and treasury as money held out of circulation, and STB F2, "held by a sector or by the pool"): a treasury that keeps what it collects is a hoard like a pool that is not released, and with no governed jurisdiction `M_t` is 0 and the form is the pool and escrow share alone. A pack whose treasuries are meant to hold a buffer sets `r2.share_bp` above it (a key of `regimes.toml`, 6.2).

6.3.3. **R5 has two parts** because STB's F5 has two readings: the beliefs of a
market sit at a bound (the `Warn` "price beliefs at bound", FR-BEL-031, emitted at
the tenth consecutive pinned tick and every tenth after) and a market has orders
on both sides and no trade. They are caused by different loops (L05, L06 against
L09, L03) and S07's perishable glut shows the second without the first.

6.3.4. **R6 and STY-P03 share one rule at two strengths.** The fact fails when a
window is above the factor **and** above the previous by more than the growth
(REF 7.2 S05: it passes when the range "stays under `range_factor` or grows by
less than `range_growth_bp`"); R6 is that pair twice running at the looser
thresholds. STB's F6 says the range "does not shrink"; a steady, non-decaying
oscillation that never grows is not caught by REF's S05 rule, and a stricter
reading would make the regime fire while the fact passes (which 6.5.2 forbids).
STB rewords F6 to "grows" (section 21, owed row).

6.3.6. **Where a detector departs from STB's wording**, and why: R1 reads the whole
economy and each class (STB: "a commodity class") and adds a clamped link, since
IND clamps an index and does not return an error (IND 5.3.7); R2 adds the share of
the top decile of household money, which STB's F2 does not name, as a hoarding by
a class of households (flagged: STB may reject it); R3 measures the fall against
the start and not within one window, because a collapse is a fall from where the
run began; R6 follows REF S05's growth rule and not F6's "does not shrink"; R9 is new. STB's text is the
meaning, STY's the measure, and a disagreement is for STB to rule (section 21).

6.3.5. **R7 reads two things**, the belief centres of a market and the starts of a
long job, because F7 is the same disease in two places (`spec-stability.md` 7:
"the dispersion of belief centres or of targets falls below a stated minimum, or
a role works and rests in lockstep"). The first is a window figure; the second
needs one-tick windows of the start of the run, where the `RecipeExecuted` rows
of the tally are exact (IND 4.2.2).

### 6.4 A regime is not a failed test

The two words are not interchangeable and the verdict record keeps them apart.

| Case | Regime | Test result | Example |
|------|--------|-------------|---------|
| a healthy-start run, a detector says `detected` | yes | **fail** of STY-G01 | a viable pack that collapses with no shock |
| a healthy-start run, a detector says `unknown` | no verdict | **undetermined**: not a pass, reported | too short a run; stale prices |
| a shock scenario that *expects* a regime | yes, of the stated kind | **pass** if the kind, the fall and the recovery are as the fact says | S27 expects R3; S43 expects R3 after exhaustion; S14's deep depth |
| a run misses a band and no detector fires | no | **fail** of that fact only | a negative offset inside R8's threshold (STY-P04) |
| a run is in a regime the scenario did not expect | yes | **fail** of the scenario | S16: R5a means the pack is rejected for this default set (REF 7.2) |
| a provoked run used to set a threshold (6.5) | yes | none: it is data | S16, S07 at a 10 000 floor |

A scenario that expects a regime says so in its band table as `expects =
["r3"]`; an expected regime that does **not** occur is also a failure of the
scenario (the shock was not felt). **A regime is a property of a run; a failed
test is a property of an assertion.**

### 6.5 Thresholds, their file and how they are set

6.5.1. **One file for every pack**: `testdata/chains/bands/regimes.toml`, because
a regime must mean the same in every pack (position 3). A pack may not set its own
threshold. The file has one table per key, with the shape of REF 7.1.5, and a
key with **no `value` line is unset** (a value of 0 is a value):

```toml
[r1.rise_bp]
unit   = "bp"
# value = 3000       absent until a reviewed run sets it
# set_by = "commit=<id>;config=<hash>;seeds=1-50"
```

A detector with an unset key returns `unknown` with the reason "threshold unset",
and G01 reports it (so the suite runs before STY has data, as REF 7.1.5 intends).

A calibration may use the priors of 6.2 as **labelled working thresholds**
(`spec-calibration.md` 3.3 item 2): they are outside acceptance, a verdict that
rests on one says so, and `unknown` for an unset key is unchanged.
6.5.2. **The ordering rule.** A regime threshold is **looser** than the band of
every fact it backs: for an upper bound the band is at most the threshold, for a
lower bound at least. The pairs are `s07.freeze_ticks` against `r5.freeze_ticks`, `s03.pinned_ticks`
and `s04.pinned_ticks` against `r5.pinned_ticks`, `s05.range_factor`,
`s12.range_factor` and `x4.range_factor` against `r6.factor`, `s05.range_growth_bp` against `r6.growth_bp`,
`s06.swing_bp` against `r6.swing_bp`, `s01.offset_bp` and `h02.offset_bp` against
`r8.offset_bp`, `s01.unmet_bp` against `r8.unmet_bp`, `p07.drift_bp` against
`r1.rise_bp` and `r9.fall_bp`, `s14.recover_ticks` (`4 * t_min`) against
`r3.horizon_factor` times `t_min` (4 against 5 of the **same** `t_min`), `s10.dispersion_min_bp` against `r7.disp_bp` (a floor: at least),
`m05.pool_share_bp` against `r2.share_bp`, and `x4.households_bp` against
`10 000 - r3.fall_bp`; and the **implication** pair that a pinned run of
`r5.pinned_ticks` ticks is at least `s16.pinned_share_bp` and
`x4.pinned_share_bp` of a year (60 * 10 000 / 365 = 1 643 by integer division,
about 1 644, against 1 000). The shares of ticks of the facts are not paired with
`r5.pinned_bp`, which is a share of outcomes within one tick. A registry test checks every pair (FR-STY-072). The
facts that bound an **expected** regime are exempt: `s14.fall_bp` 5 000 bounds a
collapse of the stated kind, which R3 detects by design, and `s17.spike_factor`
is a ratio of replacement bids, not of completions, so it is not paired with
`r7.spike_factor`. With the
priors above they hold; the point of the rule is that a regime is a gross failure
and a failed fact is a smaller one, so a pack cannot be "in regime" while its
facts pass.

6.5.3. **The separation protocol.** A threshold is set from two sets of reviewed
runs over the **setting seeds** (8.4):

| Set | What | Statistic |
|-----|------|-----------|
| **H** healthy | the warm, no-shock runs of the shipped viable packs | `t_hi` the largest value of the detector's statistic over H |
| **B** provoked | runs that cause the regime (below) | `t_lo` the smallest value over B |

Each detector has one **separated key** and **held keys** that stay at their priors
and are reviewed by hand (a held key is never set from H and B). The table names
them, with the statistic of the separated key (the value the series reaches for
the run to count as the regime, the held keys at their priors; for a detector
whose bad side is low the statistic is negated, below):

| Detector | Separated key | Statistic | Held keys |
|----------|---------------|-----------|-----------|
| R1 | `rise_bp` | the largest rise held for `windows` links | `windows`, `m_factor_bp` |
| R2 | `share_bp` | the largest pool, escrow and treasury share held for `windows` closes | `vol_bp`, `unsold_bp`, `top_bp`, `windows` |
| R3 | `fall_bp` | the deepest fall | `recover_bp`, `horizon_factor`, `horizon`, `role_ticks` |
| R4 | `vol_bp` (low is bad) | the lowest volume share held for `windows` windows | `windows` |
| R5a | `pinned_ticks` | the longest pinned run | `pinned_bp` |
| R5b | `freeze_ticks` | the longest frozen run | none |
| R6 | `factor` | the largest range reached while growing by `growth_bp` | `growth_bp`, `windows`, `swing_bp` |
| R7 | `disp_bp` (low is bad) | the lowest dispersion; prior only until REF has a provoked run | `spike_factor`, `first_ticks` |
| R8 | `offset_bp` | the largest absolute offset with the unmet share at its prior | `unmet_bp` |
| R9 | `fall_bp` | the deepest link held for `windows` windows | `windows` |

A run **counts** for R5a or R5b when its statistic is at or above the key; the
key is an integer and the threshold is the integer midpoint of `t_hi` and `t_lo`
**rounded up**, then, for a tick count, up to the next multiple of 5 and, for a
share or a ratio, to two significant digits, in both cases kept within `[t_hi + 1,
t_lo]`. If `t_lo = t_hi + 1` the threshold is `t_lo`. If `t_hi >= t_lo` the
detector does not separate. The comparison of a detector is **at or above**, so
a threshold equal to `t_hi` would flag the healthiest run and is never chosen
(the lower edge `t_hi + 1` forbids it). Bands are
set **after** thresholds (10.3) and must be at most the threshold they back
(8.4.4); a threshold is never set from a band. A detector that does not separate is recorded as unusable for that pack set, its key stays unset, and G01 reports
the detector as `unknown`. **No threshold is set from H alone**: a threshold that
only the healthy runs support is a guess at where bad begins. A detector whose bad side is low (R4's volume share, R7's
dispersion) is stated on the **negated** statistic, so that every detector reads
"badness at or above the threshold": `t_hi` is then the largest badness over H and
`t_lo` the smallest over B, and "separates" always means `t_hi < t_lo`. The shared keys `stale_max_bp` and `active_min_bp` are held too; `r7.disp_bp`
has no provoked run in REF and is **prior only** until one exists. R3 also takes the `rate` of its
speed limit from the caller (6.1.1). Where no provoked
run exists the key stays unset (the prior of 6.2 is a starting point for the
reviewer, not a value).

| Regime | Provoked runs in REF today | Provoked runs that wait for an overlay |
|--------|----------------------------|-----------------------------------------|
| R1 | S13 `overlay/faucet` (no sink), with a large `SetFlowScales` (INT) | S08 with a cycle gain above 1 is a rejected pack (STB-L10), not a run |
| R2 | S19 with the halt off (the unsold stock grows) | S35 with the pool starved of its grant |
| R3 | `famine` at depth 0 (S14); S27 (REF 7.7) | |
| R4 | S25 with money at 0.5 of the reserves (REF 7.7) | |
| R5a | S16 `overlay/overhang`; S09 no sellers (the price at the ceiling) | |
| R5b | S07 at `cost_floor` 10 000 and 9 000 | |
| R6 | S05 with the visitor on `one` + `overlay/long-grow` (S12's pack) | S32 and S46 (a haul recipe at a high supply elasticity; a gain above the limit of L84) need the trade overlay |
| R7 | the cold `one` of S17 (the completions of a long job on one tick) | |
| R8 | S22's warm leg with the stale table (a perturbed chain, not in H) | |
| R9 | none | **a new overlay** `overlay/sink.toml` (a rate levy on the households with no paired grant; section 21) |

S10 at scale 10 is the healthy run of STY-P09 and is **not** a provoked run of R7.

### 6.6 Which loops each regime exposes

The last column of 6.2 is the list of STB 7 (and L102 to L116 as the later specs
registered them). A regime that fires names its loops in the verdict, so a reader
goes from "R5a in the staple market of `one`" to "L05, L06, L14: check the
elasticity (STY-W05), the halt (L44), the floor (D13)" without searching.
Section 15 adds the four hazards of this spec to the register.

## 7. The stability checks

The roadmap's STY row ends with "the stability checks: bounded prices, no cobweb
blow-up, and a warm start that stays near the steady state", and `vision.md` 5
item 1 makes passing them part of "version 1 is done". Each is a **named
conjunction** of facts (5) and detectors (6) with one verdict, so that "stable" is
the output of a procedure and not a word.

### 7.1 The three checks

| Check | Conjunction | Packs | Waits for |
|-------|-------------|-------|-----------|
| **C-BP bounded prices** | STY-P01 (X) **and** STY-P02 (B) **and** R5a and R5b `none` in every healthy run | every variant | nothing |
| **C-NC no cobweb blow-up** | STY-P03 (B) **and** STY-D03 (B) **and** R6 `none`; where trade is on, R6's rule over the `Spread` series of each connected lane and STY-P06's worst-lane band | `one` (delayed response S05, S12), `one` + haul overlay (S32), the merchant lanes (S46, S67) | the trade overlay for the lane part |
| **C-WS a warm start stays near the steady state** | STY-H01, STY-H02, STY-H03, STY-H04 **and** STY-P04 (B) **and** R7 and R8 `none` | `one`, `two`, `three`, `five`, `second` | nothing for the closed packs |

R7 and R8 are read on the unperturbed runs of a pack only. STY-H03's two legs are
perturbed chains, judged by H03 alone; its warm (stale-table) leg is R8's provoked
run (6.5.3), so R8 `none` is never asserted on it.

### 7.2 The verdict of a check

A check has three results:

- **STABLE**: every conjunct is asserted (X and O, and every B key set), every
  detector returned `none`, over the gate seeds (SR1 for the detectors and SR2
  for the bands, 8.3) and, where the nightly has run, over its seeds;
- **NOT STABLE**: a conjunct failed, or a detector returned `detected`; the
  verdict lists which, with the seed;
- **UNDETERMINED**: a B key is unset, a detector returned `unknown`, or a needed
  module's overlay does not exist. Until a band has a reviewed value no check can
  be STABLE: **this is the state of every check today**.

7.2.1. A scenario that expects a regime (6.4) is not in a stability check; it is
a shock fact (5.3) whose verdict is its own.

7.2.2. A check never returns STABLE for a pack with a blind-spot field outside
its bound (4.3): a price index computed from stale prices, a gap across a closed
edge, an inequality figure at a scale of cohorts.

### 7.3 What "stable" is allowed to mean

N-027 and ADR 0003 decision 6 say no default is called stable until the bands
exist. This spec makes the rule exact:

7.3.1. The word **stable**, applied to a default, a pack or a run, MAY be used
only for a default that is STABLE under all three checks of 7.1 for every shipped
pack that sets it, on the gate seeds, with every key those checks read **set** by
review (8.4), **and** that is also STABLE at the two neighbouring points of the
default's knob on the grid CLB gives it. A default at the edge of the stable
region is not stable: a one-step change of a knob that flips the verdict is a
property of the default, and the neighbour rule is how it is seen
(`spec-calibration.md` owns the grid; section 10). A claim of stability says
`in sample` unless REF provides a hold-out and the claim was checked on it
(`spec-calibration.md` 7.4.2).
A default at an **edge** of its grid has one neighbour: it is **not called stable**, and
the record's `claim` line says `one_neighbour` (`spec-calibration.md` 7.4.3, FR-CLB-039). A
claim of stability is made with `Modules.Technology` off (4.6).

7.3.2. The word **stable** is not used of a configuration in a spec, a doc
comment, a log line or a release note outside this procedure; "no regime was
detected in the runs of the verdict record" is what a run supports. REF's "stable
state" (X4's predicate, FR-REF-031) and testing.md 6.4's wording are predicates
of their own over a window and claim nothing about a default.

7.3.3. A default has a **stability record**: the verdict of the three checks at
the default and at its two neighbours, the commit and configuration hash they
were set from, and the seeds. It is a text file beside the golden file
(`testdata/golden/stylised/<pack>.stability.txt`, beside the verdict record
`<pack>.txt`, 8.5).

### 7.4 What the checks do not claim

- **They are not a proof**: a pass is a statement about these packs, these seeds,
  these lengths and this scale (weight 1), with these thresholds, and nothing
  else (STB 15 open question 4: the coupling of loops is not analysed).
- **They do not cover combinations of modules**: ADR 0001 says "we do not test
  every module combination": the zero configuration, each module alone and each
  shipped pack against the bands. A pack that turns on two modules whose loops
  couple is its own pack and needs its own record.
- **They are not a calibration to a real economy**: a published figure in a
  fact's source is a sanity range (3.3, level E).

## 8. Acceptance, report and the statistics of a pass

### 8.1 What a build must pass

The merge gate and the scheduled runs map to the classes of 3.2 and the tiers of
4.5:

| Set | Contents | Asserted by | When |
|-----|----------|-------------|------|
| **Acceptance, always** | every X fact (P01, P09 exact parts, W06 bound, S02 and S06 exact parts, S03's `EdgeStatusChanged`, S05 accounts, M01, M02, M06's warning, T01, H04's stagger) and every O fact (P05 order, P06 order, W02 (a), W03, W04 order, S01 sign, S02's speed limit, S03 order, S04, M05's halt-off growth, M06, T02 order, H01 order, H03; M04 once the aggregate ratio exists), each **where REF lists the check as exact or STY's owed row lands** (3.2) | tiers S and G (N and L for the facts of those tiers) | every full build |
| **Acceptance, banded** | every B fact whose keys have a reviewed value, for the packs the fact names | tier G, with the nightly N | every full build once set |
| **Pack-asserted** | P08 on `second`; the dynamic parts of P06 and T02 once their overlays exist | tier G or N | when the pack runs |
| **Report** | R facts: D02, T03, W05's share, S04's tick, S05's drift, the threshold depth of S02, the historical premium range, W02(c); and every B fact whose key is unset, with its prior beside it | `t.Logf` and the verdict record | always, asserted never |

8.1.1. **The version 1 criteria** of `vision.md` 5 item 1 are these facts, which
**MUST** have their keys set before version 1 is called done: bounded prices
(C-BP: P01, P02), no cobweb blow-up (C-NC: P03), price convergence within
transport cost (P06), recovery after a harvest shock (S01, S02), a famine in a
closed region (S02, S06) and a stable state from the warm start (C-WS and G01).
Every other B fact may stay report-only through version 1 without making it
incomplete; each says so by staying unset.

### 8.2 Seed sets

| Name | Seeds | Used for |
|------|-------|----------|
| **gate** | 1 to 10 (REF 8.2: the band scenarios) | the merge gate's assertion of a set band |
| **setting** | 1 to 50 (the nightly range of REF 8.2, 50 seeds); for tier L, REF gives no range: 1 to 10 | the reviewed run that sets a band or a threshold (8.4) |
| **nightly** | 1 to 50 | the scheduled repeat (REF 8.2): the whole range |
| **invariant** | the ranges of REF 7.5 (1 to 16 for `one`, 1 to 4 for `three`, `famine`, `second`, `one-demand`, 1 and 2 for `five`) | the X facts |
| **configuration-only** | 1 to 4 on `second` | X4 (STY-P08) |
| **search block** | 101 and up, 100 seeds each (CLB 7.1; outside every set of REF) | one choice of a calibration; never a band, never a gate assertion |

REF fixes the sets above (FR-REF-024, FR-REF-025) and this spec does not change them; the search blocks are a calibration's own and lie outside every REF set, and the gate is never an input to a calibration's choice.
The gate seeds lie inside the setting range, so a band set from 1 to 50 is not
asserted out of sample by the gate: the gate catches a **change** of the code,
not a band that does not generalise. What stands in for the hold-out in the meantime: (1) a calibration's own
separation of search and confirm seeds (CLB 7.1: choices on blocks of seeds from 101
up, confirm on 1 to 50; this protects a chosen point, not a band); (2) the verdict
record's list of seeds outside a band and its recurrence flag across nightlies
(8.3.2); (3) a later change of the code is what the gate sees. None of the three
is out-of-sample evidence for a band. The honest cure is a hold-out set. The
recommendation (REF 8.2, a SHOULD, **not relied on**): REF 8.2 adds the seeds 51 to 60
as a hold-out that no band and no calibration choice is set from, and FR-REF-024
stands as written. Until it is run the model's figure for a
hold-out (16, `seeds.py` part C) is what a hold-out *would* give, not what the
gate gives.

### 8.3 The pass statistic

Three rules, by the kind of fact; **n** is the number of seeds the tier runs.

| Rule | Passes when | Used for |
|------|-------------|----------|
| **SR1 all** | every one of the `n` seeds satisfies the fact | X facts, O facts, and STY-G01 (the detectors over a healthy run), at every tier |
| **SR2 nine in ten** | at least `ceil(9n / 10)` seeds are inside the band (9 of 10, 36 of 40, 45 of 50, all 4 of 4); the seeds outside are **listed** in the record | every B fact, at every tier |
| **SR3 median** | the median of the `n` values is inside, with the minimum and maximum recorded | R facts that are given a reference value; a B fact's central value beside its SR2 |

8.3.1. **Why these three.** The seeds are fixed, so the test of a fixed code is
deterministic and its outcome has no sampling noise; what the rule decides is how
much a band set from some seeds can say about others, and how many seeds a change
may move before the build fails. With `q` the chance that one seed lies outside a
band for the code as it is, the chance a build passes is exact (the model of 16,
`seeds.py`, in basis points):

| `n` | `q` | SR1 all | SR2 nine in ten |
|----:|----:|--------:|----------------:|
| 10 | 1% | 9 044 | 9 957 |
| 10 | 2% | 8 171 | 9 838 |
| 10 | 5% | 5 987 | 9 139 |
| 10 | 20% | 1 074 | 3 758 |
| 10 | 30% | 282 | 1 493 |
| 50 | 2% | 3 642 | 9 995 |
| 50 | 5% | 769 | 9 622 |
| 50 | 20% | 0 | 480 |
| 50 | 30% | 0 | 7 |

A band that misses one seed in fifty (`q` 2%) makes SR1 fail 18% of builds at 10
seeds and 64% at 50, which makes it unusable as a gate for a **typical** quantity;
SR2 fails 1.6% and 0.05% of them. A regression that moves a fifth of the seeds
out is caught by SR2 with probability 0.62 at `n` 10 and 0.95 at `n` 50, and one
that moves three tenths with 0.85 and 0.999. SR1 is kept for the facts and the
detectors where **one** bad seed is a defect to be looked at (an identity that
fails on one seed is a bug; a regime on one seed of a healthy start is one),
because there it has no band to be wrong about.

8.3.2. **A listed seed is not forgiven.** SR2 lets one seed in ten sit outside a
band **without failing the build**; the record lists it and the nightly shows
whether the same seed recurs. A seed outside by more than the band's own width is
reported with a flag (the band is then likely mis-set, 8.4.5).

8.3.3. **A failed world** (a `Step` error, an audit failure, a validation error
at a point: EXP 9, 12.2.1) is a seed **outside** every band and a fail of SR1,
listed with its cause; it is never inside, and a pack whose runs fail does not
pass by having fewer runs to judge (the number of seeds `n` is the number
planned, not the number completed).

8.3.4. **Integer arithmetic.** Every value compared with a band is an integer in
the unit the key states (basis points, ticks, households, minor units); a ratio
is a quotient of integers rounded half to even (IND 5.2.2) before the comparison;
a statistic of seeds (a median of an even count) is the half-to-even mean of the
two middle values; no floating point is used anywhere in a verdict
(architecture.md 3.5).

### 8.4 Setting a band without tuning it

8.4.1. **Preconditions.** The run is of a named commit and configuration hash,
of a pack that passes viability and has its `expected/` file; no step failed; the
exact facts (X and O) of every seed pass. A band is **never** set on a run that
breaks an identity.

8.4.2. **The seeds.** The setting seeds of 8.2: the nightly range 1 to 50 of REF
(FR-REF-024: "at least the seeds of 8.2"), and for tier L the seeds 1 to 10.
A band is never set from one seed. The chance that a new seed lies outside the
envelope of `m` seeds is `2 / (m + 1)`, 1 818 bp for 10 and 392 bp for 50 (the
model of 16, `seeds.py`, part B, exact for small `m`), which is why a band is
set from the whole nightly range and carries headroom.

8.4.3. **The rule.** From the `m` observed values `v` of one fact under the
pack and scenario, for an upper bound `ub = round_up(max + h * (max - min))`, for
a lower bound `lb = round_down(min - h * (max - min))`, both for a two-sided
band; `h` is the **headroom**: 2 500 bp of the observed range when `m >= 40`,
5 000 bp when `10 <= m < 40`; and the result is rounded **outward** to **two
significant digits** (to the next multiple of the power of ten two places below
the value's leading digit, up for an upper bound and down for a lower: 201 becomes
210, 184 becomes 190; for a negative value the upper bound moves toward zero
(-201 becomes -200) and the lower away from it (-210); a value under 100 is kept). Rounding to
1, 2 or 5 times a power of ten was tried and dropped: it turns 201 into 500. The
model (part C, synthetic bell-like and one-sided shapes, **with** this rounding)
measures what a hold-out of ten new seeds would give: all ten inside in 98.5% and 99.95% of trials for `m` 50 and `h` 2 500 (97.7%
and 99.9% for 40), nine in ten in about 100%, and 90% and 93% all inside for `m`
10 and `h` 5 000 (`seeds.out`). Against REF's seeds the same band is a regression tolerance,
not a generalisation claim. `h` is a **placeholder** (P): it comes from a model
of the procedure on synthetic values, and a first real run may need another.

8.4.4. **The bounds the rule may not cross.** A band is never set looser than the
regime threshold it backs (6.5.2): if `ub` exceeds it, the band **is not set** and
the finding goes to review (the healthy runs sit beyond what the detector calls
failure: the detector, the pack or the code is wrong). A band is never set looser
than a derived bound of an O fact.

8.4.5. **The review.** A human reads the verdict record of the setting run: the
values per seed, the `unknown`s, the regimes, the seeds that sit apart. The band
is set with `set_by` naming the run (commit, configuration hash, seeds). Later
gate runs are then the check. A band the gate misses is **not** widened to pass. The review decides which of three it is: the band was
set from an unrepresentative run (re-set with the procedure and an explanation),
the code regressed (fix it), or the fact is wrong (edit this spec). The decision
is written in the pull request (testing.md 7.2).

8.4.6. **Never.** A band is never set from one seed, from a run
of fewer seeds than 8.4.2, from the code under test without review, by rounding
inward, or by editing a test until it passes. A regime threshold is never set
from healthy runs alone (6.5.3).

### 8.5 The verdict record

A run through the stylised tests produces a **verdict record**, plain text of
integers, in `testdata/golden/stylised/<pack>.txt`, regenerated only with
`-update` and explained as testing.md 7.2 asks. Its rows, in a fixed order (pack,
scenario, fact ID, then region and role in ascending ID), are:

```
pack one  scenario S01  seeds 1..10  commit <id>  config <hash>
fact STY-S01  class O  sign        ok 10/10
fact STY-S01  class B  s01.recover_ticks  band unset  prior 90
                       values min 42 median 58 max 117  outside-prior 1 (seed 7)
regime R5a none   R5b none   R6 none   R9 unknown (windows)
```

It records, per fact, the class, the statistic, the band or `unset` with its
prior, the minimum, median and maximum and the **seeds outside**; per detector
the verdict, its onset and its `unknown` reason; and `not applicable` for a fact
whose modules are absent (4.6). It carries no wall-clock time and no
floating-point number. It is the **calibration data** CLB reads: a band that is
`unset` has its observed values there.

### 8.6 What closes N-027

N-027 says no default macro constant is called stable until REF and STY exist.
Both now exist **as specifications**, which is not what the note's standing
constraint needs: a default is called stable by 7.3.1 and by nothing else. The
note **stays open**, with this closing text: *closed when the first reviewed
setting run has set the keys of the version 1 criteria (8.1.1), the gate has
passed them (and, once REF adds it, the hold-out set has), and at least one shipped pack has a stability
record (7.3.3) in which all three checks are STABLE at its defaults and their
neighbours. Until then "stable" is not claimed, and the specs' own sentence "a
default that has been shown to work" is read as "a default no run has shown to
fail".*

## 9. The tests

A fact becomes a test in the library's pyramid (testing.md 2). The point of the
split is that the **judging** is unit-testable under `-short` and the **running**
is not: a detector, a seed rule and the band-setting rule are pure functions of
plain series, so they are tested against series built to show them (tier S), and
the same code classifies a real run in tiers G, N and L.

### 9.1 Tier S: unit tests, run under `-short`

The pure functions live in `internal/verdict/` (standard library only, primitive
series, no knowledge of a commodity, a region or an agent; architecture.md 2.9).
The fact tests, in the external test package of `indicator` (FR-IND-001), call
them with series converted inline from IND's results (architecture.md 2.9,
"convert only at the call"); the package itself imports neither the root nor
`indicator`. The unit tests
are:

| Test (table-driven, testing.md 3.1) | What it asserts | Requirement |
|-------------------------------------|-----------------|-------------|
| each detector over series built to show its regime | `detected` with the right onset; `none` on a healthy seasonal series; at the threshold, one below and one above (the comparison is "at or above") | FR-STY-061 to FR-STY-069 |
| the three verdict classes | `unknown` for a series shorter than `min_windows`, for `Stale` above `stale_max_bp`, for an unset key, for `Active` below `active_min_bp`; `unknown` is never `none` | FR-STY-060 |
| the seed rules | SR1, SR2 and SR3 for `n` from 1 to 50: the count `ceil(9n/10)`, the listed seeds, the median of an even count by half to even | FR-STY-076 |
| the band-setting rule | the headroom by `m`, the outward rounding to two significant digits (negatives away from zero), refusal below 10 seeds, refusal above a regime threshold | FR-STY-077, FR-STY-078 |
| the ordering rule | every pair of 6.5.2 holds for the priors, and a violating file is rejected | FR-STY-072 |
| the registry (a text parse of this spec, like testing.md 6.7) | every fact has the fields of 3.1; every B key is in the table of 3.6 or in its fact; every key of the bands files is a key of a fact; the `set_by` shape and seeds; the priors `s14.recover_band_bp` and `r3.recover_bp` are equal | FR-STY-001 to 005, 007, 009, 010, 072 |
| the verdict record's rendering | the same input gives the same bytes; no float, no map order, no wall-clock text | FR-STY-085, NFR-STY-002 |
| the unit order of 10.3 | no key of a later step of 10.3 has a value while a key of an earlier step is unset (a text check of the bands files) | FR-STY-091 |
| the version 1 keys | with the flag `-stylised.release` the keys of 8.1.1 must all have values; without it the test reports the unset ones | FR-STY-012 |
| the word "stable" | no doc comment of an exported identifier, no log message and no error string of the module contains "stable" as a claim outside `internal/verdict`'s own verdict names (a text check) | FR-STY-096 |
| the integer-only rule and the windows (`TestVerdicts_IntegerOnly`, `TestWindows_YearOrSeason`) | no float type in `internal/verdict`; a range fact is given a year or season series | FR-STY-006, FR-STY-008 |
| the acceptance set (`TestAcceptanceSet`) | R facts and unset keys are reported, not asserted | FR-STY-011 |
| thresholds' statistics | each magnitude key has a statistic and each secondary key is listed (6.5.3) | FR-STY-074 |
| purity | no exported identifier of `internal/verdict` takes or returns a `World` or a function of one (as FR-IND-002) | FR-STY-080 |

9.1.1. **Fuzz.** The detectors and the seed rules take arbitrary integers, so a
fuzz test asserts that they never panic, that a verdict of `detected` has an
onset inside the series, that SR2's outcome is monotone in the number of seeds
outside, and that no verdict depends on the order of equal values (NFR-STY-003).

### 9.2 Tier G: the gate

An integration or end-to-end test per **group of facts** of one pack (not one test
per fact: the run is the expensive part), skipped with `-short`
(testing.md 10.5; FR-STB-007), with the seeds 1 to 10 (8.2). The test:

1. builds the pack and overlays from `testdata/chains/` (REF 4.1) and the shock
   script of the scenario (REF 7.1.3, through INT's calls);
2. runs it with `WithAudit` under a recording observer in a `_test.go` file
   (FR-REF-027) and, between steps, takes the windows IND needs;
3. computes each fact's measure from the events and the snapshot only
   (FR-STB-012) and the detectors from the series;
4. asserts the X and O facts and each B fact whose key has a value, reports the
   rest, and writes the verdict record;
5. fails with a message of the form of testing.md 3.3 that names the **fact ID,
   the pack, the seed, the measure, the value and the band** (`got`, `want`)
   (FR-STY-086).

The doc comment of each test ends with the `Requirement:` line listing the
`FR-STY` IDs of the facts it asserts and the `FR-STB` and `FR-REF` IDs of the
scenario (testing.md 4.2).

### 9.3 Tiers N and L

Tier N is tier G with the seeds 1 to 50, run on a schedule (REF 8.2 "Nightly
repeat"), asserted by SR1 and SR2 of 8.3 and recording the recurrence of listed
seeds (8.3.2). The seed range is selected by a test flag of the package's test
binary (a `flag` in a `_test.go` file; no environment variable and no file is
read, testing.md 10.2). Tier L is the facts whose run is longer than 1 800
ticks: ordinary Go tests with the build tag `stylised_long`, seeds 1 to
10, run on a schedule. Neither is in the merge gate.

### 9.4 Golden

The verdict record (8.5) is the golden file of this spec: one per pack, in
`testdata/golden/stylised/`, regenerated only with `-update`; a change is
explained as testing.md 7.2 says, and it changes only when a fact's value, a band
or a detector's verdict changes, so it is a record of what moved.

### 9.5 Determinism, restore and non-interference

- **Determinism**: two runs of one pack, scenario and seed give the same verdict
  record (FR-STY-087); on two architectures the records are byte-identical (REF
  8.6).
- **Restore**: a run split by a restore is the same run (FR-CHK-050), so the
  verdict record of `one` run 200 ticks, restored and run 200 more equals that of
  400 uninterrupted ticks (FR-STY-088; X5 of REF).
- **Non-interference**: classifying a run reads series and changes nothing
  (IND 3.1); the same seed gives the same checkpoint digest whether the
  stylised tests run or not (as FR-IND-003).

### 9.6 What runs where

| Tier | Runs | Skipped with `-short` | Cost shape |
|------|------|-----------------------|------------|
| S | detectors, seed rules, band rule, registry, record rendering, fuzz | no | counts of series length; milliseconds |
| G | the facts of one group per pack, seeds 1 to 10, at most 1 800 ticks | yes | 10 seeds of at most 1 800 ticks of `one` is about 72 s at REF 8.2's 4 ms a tick (a placeholder) |
| N | as G with 50 seeds | yes | five times G |
| L | the Malthusian and population facts | yes, and a build tag | 24 minutes for 10 seeds of 100 years of `one` (4.5.1) |

## 10. What STY hands to EXP and CLB

### 10.1 To EXP (`spec-experiments.md`)

A fact needs these of the harness, in the words of its own brief (design,
replication, running, results):

| STY needs | Why |
|-----------|-----|
| a **run** of a pack and overlays over a named seed set (gate 1 to 10, setting and nightly 1 to 50, tier L 1 to 10), with shocks as INT calls at ticks, returning the IND series and the events a verdict reads as plain data | 8.2, 9.2 |
| a **one-at-a-time sweep** of a knob over a grid that CLB supplies, at the default and its two neighbours, on the gate seeds, with the **verdict of each fact and detector** as the response | 7.3.1, 10.2 |
| a **comparison of two points as distributions over seeds**, not as a pair, for any knob that changes a draw count (`RumourErrorBP` from 0, a changed number of informed agents or intents, a changed number of admitted orders): `spec-world.md` 7.6 and WLD-A5 say the stream is positional and a change that moves a count makes the runs unpaired. A knob that moves no count (a threshold of a decision that still draws, WLD-D2) allows a paired run | 8.3 |
| the **verdict record** (8.5) as the unit of a result, keyed by the `ConfigHash` it carries (IND 3.3.2) | 8.5 |
| the **failed world** recorded and skipped, not fatal (WLD 3.5), so a sweep over a knob that breaks a point still gives the others | 7.2 |
| a **branch from a checkpoint** for the shock facts: run to the boundary, restore many times and apply a different call at the branch (CHK) | STY-S01, S02, S03 |

The sweeps each fact group needs, by the knobs the register names as its
dampers (the owner is in the parameter-owner table of README):

| Facts | Knobs (owner) | Loops |
|-------|---------------|-------|
| P02, P04, P05, S01 | `BeliefStep`, `BeliefAdjustProb`, `BeliefEvidenceShare`, `BeliefStockLow`, `BeliefStockHigh`, `BeliefMaxAge`, `CostFloor`, `CostCeiling`, `RefBandLow`, `RefBandHigh`, `PriceWindow` (BEL) | L01 to L09 |
| P03, D03 | `ProduceHaltRatio`, `RestBand` (AGT); `MinWorkShare`, `TargetSpread` (WEL); `EntryShare`, `ReviewTicks`, `RoleHysteresis`, `RoleCooldown` (SPN); a recipe's `Duration` (PRO) | L14, L21, L36, L44, L48 |
| P06, S03 | `InfoSpeed`, `RumourErrorBP`, `RumourMaxAge` (INF); `FeeShareSpreadBP`, `MaxSourceAge`, `LotUnits` (MER); `MaxDispatchPerTick`, a mode's `Speed` and `HaulSpan` (TRN) | L84, L85, L93, L116 |
| W02, W03, W06 | the slopes of `BirthSchedule` and `DeathSchedule`, `BirthStake`, `MinHouseholds`, `HouseholdCap` (POP); a resource's `Shape`, `RateBP`, `PlanShareBP`, `Capacity` (RES) | L75, L76, L79, L87, L88 |
| S02, S06 | `ChangePrivation`, `RemovePrivation`, `DeathStep`, `ExitShare`, `EntryPoolShare` (SPN); `PoolRelease`, `ReliefBelow` (EST); `EnterTicks`, `ExitTicks`, `RetryJobs` (SUB) | L11, L35, L39, L50, L63 |
| M03, M04, M05, M06 | `TargetRatio` (WEL); `SpendMaxShare`, `ReserveTicks` (AGT); a want's `Elasticity`, `Ceiling`, `Rank` (DEM) | L45, L46, L57 |
| H01 to H04 | the solver's options `StockCoverTicks`, `MoneyBufferTicks` (VIA, not `Config` fields) and `WelfareWindow`, `WindowCycles` (WEL) | L22, L26, L53 |
| the regime thresholds | none: they are data of this spec; a sweep **re-classifies** stored series with a threshold changed, which needs no run | 6.5 |

### 10.2 To CLB (`spec-calibration.md`)

| STY hands | Content |
|-----------|---------|
| **targets** | the bands whose keys are set (the version 1 criteria first, 8.1.1) and, for the rest, the observed values of the verdict record beside the prior |
| **expected signs** | the register's expected effect of a knob on a fact, as the sign of the change of one preset field (IND 11.3): `CostFloor` up raises `Pinned.Share` and lowers `Turnover.Volume` (STB-L09; the cost-floor curve of `spec-stability.md` 6.2 run 5); `BeliefStep` up raises the oscillation of a price (STB-L04); `EntryShare` up raises the role-share swing (SPN 16.1); `RumourErrorBP`: sign `?` for the converged gap of P06 (INF 11.2: an independent error narrows it from 1.80 to 0.60 of the freight, and only a shared error widens it); `BirthStake` down raises the money-limited bound `M / stake` (POP 5.5); `ProduceHaltRatio` down lowers the unsold stock (STB-L44). A measured sign against the expected one is a finding |
| **the stop rule** | the procedure stops when the three checks of 7.1 are STABLE at the defaults and both neighbours for the shipped packs (7.3.1), not when one band is met, and the chosen point is **confirmed on seeds it was not chosen on** (EXP 4.6.4 and 12.4.2: another `Seed0`, which is the calibration's own separation of search and confirm; it is not the bands' hold-out, which REF's seed sets provide only as the recommendation of 8.2, so a claim says `in sample` unless REF has one, `spec-calibration.md` 7.4.2) |
| **the blind spots** | a calibration targets a band **and** its blind-spot fields (4.3, IND-A2): a knob is not turned until a price index looks flat when `Stale` has risen |

### 10.3 The order in which bands are set

Once something has run, in this order, each step's runs being the evidence for
the next:

| Step | What | Packs | Needs |
|------|------|-------|-------|
| 0 | the X and O facts are asserted; nothing is set | all | the code |
| 1 | the healthy baseline: the per-seed statistics of every detector on the warm, no-shock runs (`t_hi`) | `one`, `second`, `two`, `three` | step 0 passing |
| 2 | the provoked runs and the thresholds of `regimes.toml` (6.5.3); detectors that do not separate stay unset | the provoked list of 6.5.3 | step 1 |
| 3 | the closed single-region bands: P02, P03, P04, P07, D01, D03, M03, M05, H01 to H04 | `one`, `second` | step 2 |
| 4 | the shock bands: P05, S01, S02 (population module off) | `one`, `famine` | step 3 |
| 5 | scale: P09 | `one` + `overlay/h200` | step 3 |
| 6 | the larger world: the same facts on `three` and `five`, W01 and M04 on the demand packs, W04 on the skill packs | `three`, `five`, `one-demand` | step 3 |
| 7 | trade: P06, S03, T02 | `two`, `three` + `overlay/trade.toml` | the trade overlay, MER, the code |
| 8 | the open layer: S04, S05, T01, T03 | `overlay/external.toml`, `overlay/faucet.toml` | the overlays |
| 9 | the long facts: W02, W03, W06, the L tier of S02 | the population and resources overlays | step 6, tier L |

**Five facts that are in no step above** are searched at the stages of
`spec-calibration.md` 7.3 and have their bands set with the group of their fact:
M06 is searched at C4b (its X and O parts; its B part is tier N) and set with step 3, P08
at C4a and set with step 3, S06 at C5 and set with step 4, G01 at C2 and again at C7b and
set once after step 6, and W05 at C5 and set with step 4.

### 10.4 What STY owes the next specs

STY adds one regime (R9) and no loop, so no entry of the register is created
by it except the hazards of 15. It asks STB for F9 and for the replacement of its
placeholder thresholds by this spec's (section 21), REF for the bands directory
layout and the overlay for R9, and OBS for nothing new: every measure is
computable from the events and snapshot fields that IND and OBS name, from INT's
`Agent(a)` queries polled by the test (class 2 cost, IND 4.5.3: STY-S04, S06,
W05) or from a recording observer in a `_test.go` file (FR-REF-027: the bids of
STY-H04). Where a measure is none of these the fact says so; one part of
STY-H04 (the remaining durability) is VIA's test and not STY's.

## 11. Parameters

This spec adds **no `Config` field, no default and no row** to the
parameter-owner table. Nothing in it changes a result of a run (a detector reads
series and a band is compared after the run), so nothing in it is a tunable
(architecture.md 9.5), and the registry test (testing.md 6.7) needs nothing from
it. The numbers it fixes are of three kinds:

- **band values and regime thresholds**: *data of tests*, in
  `testdata/chains/bands/` (3.5), set by review (8.4), each with a prior in this
  spec; they are not configuration of the library;
- **rule constants**, beside their code in `internal/verdict` (architecture.md
  9.6): the three verdict classes, the closed set of fact classes (X, O, B, R,
  P), the seed rules (`ceil(9n / 10)`), the outward rounding to two
  significant digits, the half-to-even quotient;
- **placeholders of the procedure**: the headroom `h` of 8.4.3 (2 500 bp for 40
  or more setting seeds, 5 000 bp for 10 to 39), `min_windows` 3, `stale_max_bp`
  1 000 and `active_min_bp` 5 000, which are **P** and which the first reviewed
  run may move (changing one is a behaviour change of the tests and is explained
  as testing.md 7.2 asks).

The seed sets of 8.2 are test constants (REF 10).

## 12. Validation

No rule is added to `spec-config.md`. The checks of this spec are checks of its
**own data**, run as unit tests (tier S):

| Check | Fails when |
|-------|-----------|
| fields | a fact entry of 5 lacks a field of 3.1, or an ID is repeated |
| keys | a key of a `bands/<pack>.toml` or of `regimes.toml` belongs to no fact or detector, or a B fact has no key in 3.6 or its entry |
| shape | a key with a `value` has an empty `set_by`, or a `set_by` that does not name a commit, a configuration hash and a seed set |
| seeds | a `set_by` seed set is not REF's setting range (1 to 50, or 1 to 10 for tier L) or has fewer than 10 seeds |
| ordering | a fact band is looser than the regime threshold it backs (6.5.2), or a regime threshold is set while its detector is recorded as not separating |
| coverage | a scenario S01 to S165 (S15 and those of the class "harness test" of 5.11 excepted) has no row in 5.11 (FR-STB-005, both directions) |
| closed sets | a class, a confidence or a verdict class outside its set |

`set_by` has the form `commit=<id>;config=<hash>;seeds=<a>-<b>`; no wall-clock
time is stored.

## 13. Observability

The Observability entry (observability-and-logging.md 9): **None, because** this
spec adds no failure path that is survived, no state transition, no decision
point in a strategy, clearing or spawner, no loop and no metric: it reads the
events and the snapshot that OBS defines through IND, and judges them in test
code. Two things so that a reader does not look for them:

13.1. **No logger and no event.** `internal/verdict` has no `*slog.Logger`; a
verdict is a return value (observability-and-logging.md 2.3: anything a caller
needs belongs in the return value), and the tests print it with `t.Logf`.

13.2. **No metric.** A regime is not an instrument: the counters that make a
damper's action visible (FR-STB-013) are the owners'. A consumer who wants a
regime on a dashboard classifies the series it already keeps.

## 14. State contributed to a checkpoint

**None.** A detector, a seed rule and a verdict record are functions of series and
files; they hold no state of the world, and a restored world is classified from
its series as the original was (FR-CHK-050). A band, a threshold and a record are
test data, not part of a checkpoint.

## 15. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. The checklist
of `spec-stability.md` 9 was followed: 9.1 no rule of this spec reads a measure
and changes a rate (a detector returns a verdict and nothing reads it back,
IND 3.1), so **no feedback loop of the library is added**; 9.4 the new regime has
a test (6.5.3); the new entries are hazards of **method**, the way IND-A2 is.

- **STY-A1. A band is tuned to the code.** *Kind:* hazard of method. *Loop:* a
  calibration of a knob (+) a measure (-) the distance to a band (+) the knob, as
  IND-A2, with a second path: the band itself is moved by a reviewer until the
  test passes (+) the band (-) the failures, which closes without the library
  moving. *Delay:* a review cycle. *Owner:* STY 8.4, CLB. *Damper:* REF's seed
  sets and a hold-out set once REF adds it (8.2), headroom (8.4.3), outward rounding,
  the ordering against the regime threshold (6.5.2), a review that must say which
  of three it was (8.4.5), and the record of every set value with its run.
  *Undamped:* the reviewer's judgement. *Condition:* every band's `set_by` names a
  setting run on REF's seeds. *Test:* the registry check of 12.
  *Status:* hand.
- **STY-A2. A threshold at the edge of healthy.** *Kind:* hazard of method.
  *Loop:* none: a regime threshold that the healthy runs approach flags a healthy
  pack, or one that a bad run approaches passes a bad pack. *Owner:* STY 6.5.
  *Damper:* the separation protocol (6.5.3) sets a threshold only where healthy
  and provoked runs are separated, and records a detector that does not as unusable
  (its key unset, the verdict `unknown`). *Undamped:* a regime with no provoked
  run (R9 until `overlay/sink.toml` exists). *Condition:* `t_hi < t_lo` for every
  threshold set. *Test:* S16, S07 at a 10 000 floor and the other provoked runs of
  6.5.3. *Status:* hand.
- **STY-A3. Silence is read as health.** *Kind:* hazard of reading (a neighbour
  of IND-A3). *Loop:* none: a detector over a series that is too short, whose
  prices are stale or whose key is unset returns "no regime", and a verdict of
  `none` is then read as `stable`. *Owner:* STY 6.1.2. *Damper:* the third verdict
  class, `unknown`, which a check treats as UNDETERMINED and not as STABLE (7.2).
  *Undamped:* a consumer that reads only `detected`. *Condition:* a check cites its
  `unknown`s. *Test:* the unit tests of 9.1 (FR-STY-060). *Status:* hand.
- **STY-A4. A listed seed that recurs.** *Kind:* bias (a neighbour of WLD-A2).
  *Loop:* none: SR2 lets one seed in ten sit outside a band, and if the same seed
  or agent position is the one that sits outside on every nightly, the cause is an
  ID, a region order or a cap that favours it (WLD 10.2: "IND and STY must not read
  a regional difference as real before checking whether a cap or a limit was
  binding"). *Owner:* STY 8.3.2, WLD 10.2. *Damper:* the record lists the seeds
  and the nightly flags a recurring one; the `Binding` field of IND 8.7 is read
  with it. *Undamped:* a cap whose bias is small. *Condition:* no seed is outside
  in more than a stated share of nightlies. *Test:* WLD-S1 and the nightly record.
  *Status:* hand.

**New regime.** STY added **STB-F9, deflation with distress** (STY-R9, 6.3.1),
with its threshold, its measure and its loops (L70, L71, L98), and it is now in
`spec-stability.md` 7. It adds no loop: it names a state of loops that exist.

## 16. What was run

Three scripts, standard library only, archived in `doc/specs/models/sty/`, run
only in Docker (`python:3.12-slim`, `PYTHONDONTWRITEBYTECODE=1`, never the host
interpreter). They are models of the **definitions** of this spec, run on
**synthetic** series; nothing in them is a run of the library, a default or a
band (N-027).

| Script | Output | What it does |
|--------|--------|--------------|
| `seeds.py` | `seeds.out` | (A) the probability that a build passes under SR1 and SR2 for 10 and 50 seeds by the chance `q` that a seed is outside, exact with fractions; (B) the chance that a new seed lies outside the envelope of `m` exchangeable seeds, 2 / (m + 1), by enumeration for `m` 3, 5, 7; (C) the pass rate of ten new seeds against the band-setting rule (headroom and two-digit rounding) by `m` and the headroom on a bell-like and a one-sided integer distribution. Quoted in 8.3 and 8.4 |
| `detectors.py` | `detectors.out` | the detectors R1 to R9 in integer arithmetic on built series: the verdict of each on its regime, on a healthy seasonal series and on a series that is too short or stale; the ordering rule on the priors; the year and 120-tick windows on a stationary seasonal series. Quoted in 4.1.3, 6 and 17 |
| `worked.py` | `worked.out` | every number of the worked examples of 17, asserted (imports the two others) |

What they do **not** show: that a measure or a detector is informative of a real
run, that any threshold or band is right, that the headroom rule is the best
rule, or that the library behaves as any fact says. They show that the
definitions are well formed, that the detectors separate the series built to
separate, and what the seed rules and the setting rule do in arithmetic.

## 17. Worked examples

An **invented** setting (N-017): the numbers are synthetic, from a fixed integer
generator; no word of a setting is used. `worked.py` computes and asserts every
figure; the two others compute the rest.

### 17.1 Setting a band

The fact is the recovery time of STY-S01 (`s01.recover_ticks`), an upper bound.
Fifty setting seeds (1 to 50) give recovery times with minimum 70, median 116 and
maximum 161 ticks (range 91). With 50 seeds the headroom is 2 500 bp of the
range: `161 + 91 * 2 500 / 10 000 = 161 + 23 = 184`, rounded outward to two
significant digits, **190 ticks**. The ten gate seeds (1 to 10) are among the
fifty, so all ten are inside by construction (the model's own run); SR1 and SR2
both pass, and what they show later is a change of the code. The prior was 90, and the
observed envelope lies above it: the review decides whether 90 was a bad prior or
the code is slower than the register's model, and **never** writes 90 over the
data (8.4.5).

### 17.2 The seed rules on one gate run

Ten gate values of a recovery time against an upper bound of 90: 41, 55, 58, 60,
62, 64, 66, 70, 74 and 103. Nine are inside. **SR1** (all) fails; **SR2** (at least
`ceil(9 * 10 / 10)` = 9) passes, with seed 103 **listed** in the record; the
median 63 is inside. A build is not failed by one seed in ten and is not told it
is healthy either: the record shows the seed and the nightly shows whether it
recurs (STY-A4).

### 17.3 Classifying a series

A one-commodity economy whose price drifts up 40% a year (and has a 25% seasonal
swing). The yearly means are 9 994, 13 993, 19 589 and 27 417; the links are
14 001, 13 999 and 13 996 bp. R1's rule (a rise of at least 3 000 bp in each of 2
consecutive links) fires at the **second link**, the third year window: `detected`,
onset window 2 of the model (the first year, the transient, is included: a
detector reads whole years from tick 0, 4.1.2).
The same series without the drift gives R1 `none` and R6 (year windows) `none`,
although its price swings by a factor of 1.67 within every year: the seasonal swing
is not a regime (4.1.3).

### 17.4 The speed limit of a recovery

A region that fell from 1 000 households to 820 (an 18% fall) in a pack with the
default schedules of POP: `g_max = max(birth) - min(death) = 250 - 50 = 200` bp a
year. By integer iteration per tick with a carried remainder over `10 000 *
TicksPerYear`, the households need regrow to 900, the lower edge of `s14.recover_band_bp` (the level the observed
time is measured to), by `1 700` ticks (4 years and 240 ticks); whole-year
compounding in exact fractions gives 5 years, the coarser reading, not used. The
observed recovery of STY-S02 must be at least 1 700 ticks (O), its band is prior
`4 * t_min` = 6 800 ticks and R3's horizon `5 * t_min` = 8 500. S14's 1 800 ticks cannot
show it: with the population module on the leg is **tier L** (4.5).

### 17.5 The historical premium

The twelve points of Clark's table A2 in basis points: 15 600, 19 300, 19 300,
17 900, 15 000, 14 400, 14 700, 14 400, 16 000, 15 100, 15 200 and 14 100.
Minimum **14 100**, median **15 150** (the half-to-even mean of the two middle
values, 15 100 and 15 200), maximum **19 300**. They are the range STY-W04 reports;
no band is built from them (the model is another trade, place and time).

### 17.6 The gap after freight

Three directed lanes with delivered price (origin price plus freight per unit)
112, 99 and 120 and destination prices 118, 104 and 141 give gaps of 6, 5 and 21,
that is 536, 505 and 1 750 bp of the delivered price. The **median** 536 is under
the margin `x1.margin_bp` of 1 000, and the **worst** lane, 1 750, is under
`x1.worst_bp` of 3 000, so STY-P06's B part passes; a worst-lane prior of 1 500
would have failed it, which is what the second band is for.

### 17.7 The earliest unfed tick

A household holds 14 units of an import-only non-perishable upkeep good and needs
2 a tick; the edge is closed at tick 400. Holdings fall by at most the need a
tick, so the household is not unfed before tick `400 + floor(14 / 2) = 407`
(STY-S04, O).

### 17.8 A seasonal series read by year and by 120-tick windows

A stationary series with a triangular seasonal swing of plus or minus `amp` and 2%
noise, eight years, by year windows and by windows of 120 ticks (`detectors.py`
part 4):

| `amp` (bp) | Year windows: highest range | grow | fact | R6 | 120-tick windows: pairs | grow | fact | R6 run |
|-----------:|---------------------------:|-----:|-----:|----|------------------------:|-----:|-----:|--------|
| 2 500 | 17 153 | 0 | 0 | none | 23 | 3 | 0 | none |
| 4 000 | 23 979 | 0 | 0 | none | 23 | 6 | 0 | none |
| 5 000 | 30 802 | 0 | 0 | none | 23 | 4 | 0 | none |
| 6 000 | 41 015 | 0 | 0 | none | 23 | 5 | 5 | none |

*Grow*: a window's range above the previous by more than 1 000 bp; *fact*: grows
**and** is above 20 000 (fails both conjuncts of STY-P03). Year windows never grow
on a stationary series. 120-tick windows read growth in 3 to 6 of 23 pairs, and
at a swing of 60% five pairs fail STY-P03 although nothing is a cobweb. R6's
two-in-a-row rule at a factor of 30 000 and growth of 2 500 does not fire on
either reading: the regime's rule is stricter than the fact's, as 6.5.2 requires.

## 18. Notes settled

Read: every open note tagged STY (N-015, N-027, N-030, N-084, N-085, N-086,
N-092, N-113), every `ALL` note (N-016, N-017, N-027, N-103), and N-028, N-042,
N-043, N-054, the loops that decide what a band can claim.

| Note | Disposition | Where |
|------|-------------|-------|
| N-015 placeholders | **Left open**, updated: STY's priors, the thresholds of 6.2, the headroom and `min_windows`, `stale_max_bp` and `active_min_bp`, and every number of the models are placeholders | 3.3, 6.2, 11 |
| N-027 no default is stable | **Left open** with a new closing text: closed by the first reviewed setting run that sets the version 1 criteria and a stability record in which the three checks are STABLE at the defaults and their neighbours | 7.3, 8.6 |
| N-030 five regions | **Applied for STY**: facts are stated at weight 1 on `five` and `three` (comparative advantage, P06, T02) and on `one` for scale (P09); the smallest groups of `five` (4 to 9 households) are why S10 uses `one`. Remains for the tick rate (REF, SCL) | 4.4, 5.1, 5.6 |
| N-084 scenarios waiting for trade, money and the coarse tick | **Applied in part**: S13 is STY-S05 (exact accounting, report-only drift, with the reason), X1 and X2 dynamic have bands (P06, T02) that wait for the overlay; the coarse-tick leg of S12 stays open for CAL and REF | 5.3, 5.6 |
| N-085 nominal table against the full-labour table | **Applied**: a report side by side on `famine` in H02; the comparison needs a run | 5.7 |
| N-086 re-run the demand and spawner models under the register | **Left open** for EXP: STY's models are of its own definitions and are no re-run of the luxury model with DEM's schedule or the occupation model with SPN's discrete rules; the experiment is described by S23 and S26 and their keys in 5.11 | 5.11, 10 |
| N-092 overlays and scenarios | **Applied in part**: the bands for natural increase (W06), recovery time (S02) and the money-limited population (W06) are STY's; the overlays stay REF's (and `overlay/sink.toml` is new) | 5.2, 5.3, 21 |
| N-113 a quote is a limit | **Applied**: STY-T03 is the decision input (the share of ticks near the quote, `t03.pull_bp`); CLR adds the flag if it falls below it for a setting that wants a peg | 5.6 |
| N-016 parameter registry | **Applied**: this spec adds no `Config` field and no row; bands are test data | 11 |
| N-017 words | **Applied**: setting-agnostic names; new terms below; the words of the examples are flagged | glossary |
| N-103 parallel writers | **Applied**: own files only, handoff, models in Docker; one slip is recorded in the handoff (the host `python3 --version`) | handoff |
| N-028 target-income loop | **Read**: a staple's band cannot claim "not pinned" (C2); STY-W05 bounds the elasticity and P02 records the outcome | 5.9 |
| N-042 evidence offset | **Read**: P04's prior is 2.6 times the revised rule's 5.7% and says so | 5.1 |
| N-043 nothing anchors the level | **Read**: no absolute price level is banded; every price fact is relative to the flow-balancing price, a seed, a base window or a freight | 4.2 |
| N-054 the closed economy's ratio | **Applied**: M03 is its identity, M04 its ceiling, C1 its consequence for the population module | 5.5, 5.9 |

**New notes to add** (IDs are the supervisor's):

1. *STB, REF, IND, OBS*: the failure regimes now have thresholds (`regimes.toml`);
   a ninth regime, deflation with distress (F9), is added to the register; the scenario bands
   have one home (STY 3.5, 5.11). Section 21 lists the edits.
2. *CFG, POP, STY*: a pack with the population module on, no demand module and no
   faucet has no healthy start at the default schedules (C1); the warning
   `replacement_above_closed_ceiling` is read by acceptance as a rejection.
3. *CODE*: the home `internal/verdict/` (detectors, seed rules, the band-setting
   rule) and the build tag `stylised_long`; `architecture.md` 2.6 and `AGENTS.md`
   gain the line when the package is first needed.
4. *EXP, CLB*: the needs of 10.1 and the expected signs of 10.2.
5. *IND*: open question 7 (a default for `Weights`) is answered by 4.2.2: STY
   names none; each fact names its weights.
6. *EVD*: no source read gives a distribution of money or wealth, nor a period of
   the pre-industrial wage and population cycle (5.2, 5.4); the next step is to
   read one before D02 or W02(c) is given a band.

## 19. Requirements

Each row is one requirement (testing.md 4.1) and tests cite its ID. A fact's row
says that the fact holds as stated in its entry, with the class and tier of that
entry; where the entry has parts (an X part, an O part, a B part) the row's
clauses are those parts and **each clause is one testable statement**, which a
test asserts separately under the one ID.

### Functional

| ID | Requirement |
|----|-------------|
| FR-STY-001 | Every fact MUST be written with the fields of 3.1 and an ID `STY-<group><nn>`, and an ID MUST NOT be reused. |
| FR-STY-002 | Every fact, and every part of a fact of a different class, MUST have one class of X, O, B, R or P, and a tier of S, G, N or L stated for it (a fact that runs in two tiers states both). |
| FR-STY-003 | Every fact MUST state its confidence as one of I, H, M, E or P (or "none" for a report fact with no band), a band with no basis MUST be labelled a placeholder, and a source that is a snippet, a summary or unverified MUST NOT be given level E. |
| FR-STY-004 | A band MUST have one definition (a fact of this spec) and one value (a key of `bands/<pack>.toml`), every key of a band file MUST belong to a fact, and every B fact MUST have its keys named. |
| FR-STY-005 | A band value MUST be set only by the procedure of 8.4, from REF's setting seeds (1 to 50; 1 to 10 for tier L), and MUST NOT be set from one seed, from fewer than ten seeds, or from the code under test without review. |
| FR-STY-006 | Every value compared with a band, every band and every verdict MUST be an integer, with quotients rounded half to even, and no floating-point type MUST be used (architecture.md 3.5). |
| FR-STY-007 | A B or O fact MUST be asserted together with its blind-spot fields (4.3), and a result with a field outside its bound MUST be undetermined and not passed. |
| FR-STY-008 | A fact about a range, a level or a trend of a seasonal pack MUST use year or season windows, and the transient of 4.1.2 MUST be excluded. |
| FR-STY-009 | A fact whose modules are absent (5.10) MUST be recorded as not applicable and MUST NOT be passed or failed. |
| FR-STY-010 | The places where two facts cannot both hold (5.9) MUST be kept, and a pack MUST NOT assert both facts of a pair. |
| FR-STY-011 | Acceptance MUST be every X and O fact and every B fact whose keys are set, and an R fact, or a fact still of class R because its measure does not yet exist (M03, M04), MUST NOT be asserted. |
| FR-STY-012 | The keys of the version 1 criteria of 8.1.1 MUST be set by review before version 1 is called done. |
| FR-STY-020 | STY-P01: no published price MUST leave the interval its reference and cost bands allow, and no balance or inventory MUST be negative, at any tick, for every seed. |
| FR-STY-021 | STY-P02: the share of ticks of the year in which a market is pinned MUST be within `s16.pinned_share_bp` and `x4.pinned_share_bp`, the longest pinned run within `s03.pinned_ticks` and `s04.pinned_ticks`, the longest frozen run within `s07.freeze_ticks`, and the spoiled share of a perishable glut within `s07.spoiled_bp`. |
| FR-STY-022 | STY-P03: the price range over successive year or season windows MUST satisfy REF's form of the scenario: S05 within `s05.range_factor` or growing by at most `s05.range_growth_bp`, S12 within `s12.range_factor` and growing by at most `s05.range_growth_bp`, X4 under `x4.range_factor`. |
| FR-STY-023 | STY-P04: the signed offset of the last 100 ticks MUST be within `s01.offset_bp` of the steady-state price with unmet need below its bound, without a faucet and without an outside. |
| FR-STY-024 | STY-P05: a fall of an input's supply MUST raise its price during the shock, the first excursions along the chain MUST be in order, and every price MUST settle within `s08.settle_ticks` inside `s08.band_bp`. |
| FR-STY-025 | STY-P06: with trade on, the median `GapAfterFreight` MUST be within `x1.margin_bp` and the worst lane within `x1.worst_bp`, the trading pair's `Spread` MUST be below its control's, and each traded relative price MUST lie between the autarky ratios. |
| FR-STY-026 | STY-P07: with constant households and no faucet, outside or shock, the yearly link of the price index MUST be within `p07.drift_bp` of 10 000. |
| FR-STY-027 | STY-P08: on `second`, each commodity with a producer MUST trade in at least `x4.trade_share_bp` of the ticks of the last year and the mean volume of an active market MUST be at least one unit. |
| FR-STY-028 | STY-P09: the group weights MUST add up and money and goods at tick 0 MUST be equal at scales 1, 4 and 10, and the averaged prices MUST be within `s10.tolerance_bp` and the dispersion above `s10.dispersion_min_bp`. |
| FR-STY-029 | STY-W01: with demand beyond upkeep, the household-weighted mean ratio of the unskilled roles MUST be within `w01.mean_low_bp` and `w01.mean_high_bp`. |
| FR-STY-030 | STY-W02: with a binding fixed factor, households started above and below MUST end within `w02.n_band_bp` of each other with the mean ratio within `w02.ratio_band_bp` of `r*`, crossing at most once when no gain warning is raised. |
| FR-STY-031 | STY-W03: after a shock that lowers households, the sum of the products of the changes of households and of the mean ratio over the regrowth MUST be negative. |
| FR-STY-032 | STY-W04: with the tool the observed premium MUST exceed 10 000, with the tool free it MUST be within `s06.free_tool_tol_bp` of 10 000, and the historical range MUST be reported. |
| FR-STY-033 | STY-W05: the measured elasticity of the staple MUST be at most `s04.elasticity_max_bp` and the working share MUST be reported. |
| FR-STY-034 | STY-W06: births done in a year MUST NOT exceed the schedule's maximum, and a closed region with an empty pool MUST stop within `w06.money_cap_bp` of `M / stake` with `BirthsUnfunded` non-zero. |
| FR-STY-035 | STY-S01: a supply cut and a demand step MUST raise the price while they last, and the price MUST return within `s01.band_bp` within `s01.recover_ticks`. |
| FR-STY-036 | STY-S02: each failed harvest MUST be recovered from in not less than the speed limit and within `s14.recover_ticks`, or be a collapse whose fall is within `s14.fall_bp`, with the identities exact. |
| FR-STY-037 | STY-S03: the spread of a closed edge's ends MUST not fall while it is closed, `EdgeStatusChanged` MUST be emitted at both changes, and the spread MUST return within `s28.band_bp` within `s28.reopen_ticks`. |
| FR-STY-038 | STY-S04: a household MUST NOT be unfed before its holdings of a non-perishable import-only good could be used up. |
| FR-STY-039 | STY-S05: the boundary balance of a flow's reason MUST equal minus its total and `M` MUST equal `M_init` plus it, and the price drift MUST be reported with no band. |
| FR-STY-040 | STY-S06: a removal for hunger MUST come at least `RemovePrivation` unfed ticks after the first unfed tick, and relief MUST precede reduction and removal. |
| FR-STY-041 | STY-D01: the unfed share of household-ticks of a viable pack from its warm start MUST be within `d01.unfed_bp`. |
| FR-STY-042 | STY-D02: the inequality of money MUST be reported with its `CohortShare` and MUST NOT be asserted. |
| FR-STY-043 | STY-D03: the end share of a role MUST be within `s06.share_band_bp` of its balanced share and its swing within `s06.swing_bp`. |
| FR-STY-044 | STY-M01: in a closed economy `M` MUST equal `M_init` at every tick and the sectors MUST add up to it. |
| FR-STY-045 | STY-M02: the ledger's identities of IND 7 and 8 MUST hold exactly for every tick and seed of REF 7.5. |
| FR-STY-046 | STY-M03: the aggregate ratio (`HouseholdIncome`) of a closed upkeep-only pack, when it is `OK`, MUST be within `s19.ratio_band_bp` of 10 000; until then the mean of ratios MUST be reported beside it and not asserted. |
| FR-STY-047 | STY-M04: with wants the aggregate ratio, when `OK`, MUST be above 10 000 and at most `10 000 * (1 + Wmax / U)`; until then it MUST be reported. |
| FR-STY-048 | STY-M05: the share of money in the pool, the escrow and the treasuries MUST be within `m05.pool_share_bp`, and with the halt on the unsold stock within `s19.stock_bound_bp`, and with it off the stock of at least one good MUST pass it. |
| FR-STY-049 | STY-M06: the mean money per household of a closed region MUST NOT exceed its start, and validation MUST raise `reserve_exceeds_money` for a start below the reserve, and a group that is a net earner MUST reach the reserve within `s20.ticks` (tier N, when REF has the pack of section 21). |
| FR-STY-050 | STY-T01: the identities of EXT 8.2 MUST hold and the counterparties MUST be empty at every boundary. |
| FR-STY-051 | STY-T02: with trade the distance of a region's role shares to the pooled census MUST fall over four years (REF X2) and end within `x2.share_band_bp`. |
| FR-STY-052 | STY-T03: the share of ticks near the quote MUST be reported against `t03.pull_bp`. |
| FR-STY-053 | STY-H01: each price of a warm run MUST stay within `a1.price_band_bp` of its seed for 30 ticks and the cold run MUST NOT do better. |
| FR-STY-054 | STY-H02: with the population module off, each market's offset, the households and the money per household of a warm run MUST stay within `h02.offset_bp`, `x4.households_bp` and `h02.money_bp` over the windows after the transient. |
| FR-STY-055 | STY-H03: a warm and a cold start of a perturbed chain MUST end in the same regime. |
| FR-STY-056 | STY-H04: the replacement bids of a durable MUST be within `s17.spike_factor` of their mean in a warm start and the cold control MUST show a larger ratio by `s17.control_gap_bp`; the completions of a long job MUST be staggered. |
| FR-STY-057 | STY-G01: every detector MUST return `none` for a viable pack run from its warm start with no shock, for every seed, and households MUST stay above `x4.households_bp` of their start. |
| FR-STY-058 | STY-W07: with the employment module on, the unemployment share and the employers' margin MUST be recorded in the verdict record and MUST NOT be asserted. |
| FR-STY-060 | A detector MUST return one of `none`, `detected` or `unknown`; it MUST return `unknown`, never `none`, for too few windows, stale prices (R1, R9), low activity (R6, R8), an unset key or a missing reference, and the volume detectors R2, R4, R5b and R9 MUST NOT be made `unknown` by low activity. |
| FR-STY-061 | R1 MUST detect a rise of at least `r1.rise_bp` in each of `r1.windows` consecutive windows, or money at `r1.m_factor_bp` of its start, or a clamped link. |
| FR-STY-062 | R2 MUST detect falling volume with a pool, escrow and treasury share, an unsold share or a top-decile share at or above its threshold for `r2.windows` windows. |
| FR-STY-063 | R3 MUST detect households at or below `10 000 - r3.fall_bp` of their start not recovered to `10 000 - r3.recover_bp` within `r3.horizon_factor` times the speed limit `t_min` of STY-S02 (at least `r3.horizon` ticks), or a role at 0 for `r3.role_ticks`. |
| FR-STY-064 | R4 MUST detect volume below `r4.vol_bp` of the first window in `r4.windows` windows with no collapse. |
| FR-STY-065 | R5a MUST detect a pinned share at or above `r5.pinned_bp` for `r5.pinned_ticks` ticks and R5b a run of `r5.freeze_ticks` ticks with unmatched units on both sides and no volume. |
| FR-STY-066 | R6 MUST detect a range above `r6.factor` growing by more than `r6.growth_bp` in `r6.windows` windows, or a role-share swing of `r6.swing_bp` in two windows. |
| FR-STY-067 | R7 MUST detect a lowest dispersion below `r7.disp_bp` or completions above `r7.spike_factor` times their mean in one tick. |
| FR-STY-068 | R8 MUST detect an offset of at least `r8.offset_bp` with unmet need of at least `r8.unmet_bp`. |
| FR-STY-069 | R9 MUST detect a fall of at least `r9.fall_bp` in `r9.windows` windows with falling volume. |
| FR-STY-070 | The regime thresholds MUST be in one file, `bands/regimes.toml`, and no pack MUST set its own. |
| FR-STY-071 | A threshold MUST be set only where healthy and provoked runs are separated, and a detector that does not separate MUST be recorded as such and left unset. |
| FR-STY-072 | A fact band MUST be at least as strict as the regime threshold it backs, and a file that violates the ordering MUST be rejected. |
| FR-STY-073 | A scenario that expects a regime MUST name it, and an expected regime that does not occur MUST fail the scenario. |
| FR-STY-074 | A threshold key's statistic, its count keys and the rounding within `[t_hi, t_lo]` MUST be as 6.5.3 states, and bands MUST be set after thresholds. |
| FR-STY-075 | The gate seeds MUST be 1 to 10 and the setting seeds 1 to 50 (1 to 10 for tier L), as REF 8.2 fixes them (FR-REF-024, FR-REF-025). |
| FR-STY-076 | SR1 MUST pass only when every seed passes, SR2 when at least `ceil(9n / 10)` seeds are inside with the others listed, and SR3 on the median with the minimum and maximum recorded. |
| FR-STY-077 | A band MUST be `extreme +- h * range` rounded outward to two significant digits, with `h` 2 500 bp for 40 or more setting seeds and 5 000 bp for 10 to 39. |
| FR-STY-078 | A band MUST NOT be set from fewer than 10 seeds, and MUST NOT be set looser than the regime threshold or a derived bound. |
| FR-STY-079 | A band missed by a later gate, nightly or hold-out run MUST NOT be widened to pass, and the review MUST record whether the band, the code or the fact was wrong. |
| FR-STY-080 | The detectors, seed rules and band rule MUST be pure functions of plain series that take no world, draw nothing and read no clock. |
| FR-STY-081 | The unit tests of the detectors, the seed rules, the band rule and the registry MUST run under `-short`. |
| FR-STY-082 | The facts of one group and pack MUST be asserted by one integration or end-to-end test, skipped with `-short`, with the `Requirement:` line of its facts. |
| FR-STY-083 | Tier N MUST run the gate tests with seeds 1 to 50 and tier L MUST be behind the build tag `stylised_long`; neither MUST be in the merge gate. |
| FR-STY-084 | The verdict record MUST be a golden file per pack under `testdata/golden/stylised/`, regenerated only with `-update` and explained. |
| FR-STY-085 | The verdict record MUST contain no floating point, no map order and no wall-clock time, and MUST list the seeds outside a band. |
| FR-STY-086 | A failure message MUST name the fact ID, the pack, the seed, the measure, the value and the band (testing.md 3.3, 6.2). |
| FR-STY-087 | Two runs of one pack, scenario and seed MUST give the same verdict record, on two architectures. |
| FR-STY-088 | The verdict record of a run split by a restore MUST equal that of the uninterrupted run. |
| FR-STY-089 | Running the stylised tests MUST NOT change the checkpoint digest of a run. |
| FR-STY-090 | The sweeps of 10.1 MUST be runnable as a harness's one-at-a-time sweeps at a default and its two neighbours, with the verdict as the response. |
| FR-STY-091 | Bands MUST be set in the order of 10.3. |
| FR-STY-092 | A knob's measured sign on a fact MUST be compared with its expected sign of 10.2 and a difference MUST be a finding. |
| FR-STY-095 | A stability check MUST be a conjunction as 7.1 states and MUST return STABLE, NOT STABLE or UNDETERMINED. |
| FR-STY-096 | The word stable MUST be used of a default only under 7.3.1. |
| FR-STY-097 | A default MUST have a stability record (7.3.3) before it is called stable. |
| FR-STY-098 | Every scenario S01 to S165 except S15 and those of the class "harness test" (the list of 5.11) MUST have a row in 5.11 holding its band or naming what it asserts. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-STY-001 | The time and allocations of a detector MUST be a function of the length of its series and MUST NOT depend on the number of agents of the run. |
| NFR-STY-002 | A verdict and a record MUST be identical on every platform (architecture.md 7.4). |
| NFR-STY-003 | The detectors and the seed rules MUST have fuzz tests asserting that they never panic, that an onset lies inside the series, that SR2 is monotone in the number of seeds outside and that equal values in any order give one verdict. |
| NFR-STY-004 | `internal/verdict` MUST import only the standard library (architecture.md 5.1). |

## 20. Open questions

1. **P01 with an outside node.** A counterparty's quote is a limit and the
   midpoint of the realm's limit and a quote can leave the realm's reference band
   if the quote does (`spec-external-sector.md` 6.7). Does EXT bound the quote to
   the band (D66's `FloorBP`, `CeilingBP`), or is the interval of STY-P01 widened
   for a pack with an outside? *Next step:* EXT.
2. **The speed limit without the population module.** Answered by SPN: like-for-like
   replacement is not under the caps of SPN 6.3 (FR-SPN-022, 7.2), so the
   per-tick cap `ceil(Households * EntryShare / 10 000)` is not the limit of regrowth.
   STY-S02 uses `Renewal` times the leavers of the tick (SPN 7.1) and the funding
   (`EntryPoolShare` of the pool per tick, SPN 8.2). No SPN edit.
   *Next step:* none.
3. **Where the detectors live for a consumer.** They are in `internal/verdict`
   here, so a consumer cannot call them; CLB's procedure is a consumer's loop
   (IND 11.3). *Next step:* CLB decides whether they join `indicator` (IND would
   gain a closed set of verdicts) or a public package; STY needs only that the
   code is one. Answered in part: CLB's open question 1 recommends a public
   `verdict` package when the first consumer runs its own calibration; the
   supervisor decides.
4. **A healthy pack with a warning it expects.** `one-demand` expects
   `target_above_demand` (its ceiling is 1.32, below the target 1.5, REF 4.4.6).
   Is a pack that carries a warning by design "healthy" for G01? *Next step:*
   REF lists the expected warnings (FR-REF-005) and G01 allows exactly those.
5. **The thresholds' scale.** Every threshold is for weight 1 and for the packs
   of REF; a pack with a large number of agents has a smaller noise. *Next step:*
   the first run of `five` against `one`.
6. **The headroom.** `h` of 2 500 and 5 000 bp comes from synthetic shapes. *Next
   step:* the first reviewed setting run measures the miss rate on real series, on the hold-out set once REF adds
   it (section 21).
7. **Coupled loops.** The facts are one at a time (STB open question 4). *Next
   step:* S22 and S05 with the halt on; no earlier.
8. **A distribution of money.** D02 has no band because no source read gives one.
   *Next step:* EVD reads one (Allen, Clark and others) before a band is proposed.
9. **The period of a wage and population cycle.** W02(c) is report-only for the
   same reason. *Next step:* EVD, and a pack with a generation lag
   (`spec-population.md` open question 1).
10. **The share of seeds SR2 forgives.** One in ten is a placeholder. *Next step:*
    the first nightly shows how often a seed is outside and whether it recurs.

## 21. Edits owed to other files

One consolidated table: file, section, exact change. None alters a result that a
test could depend on (nothing has run); the rule files need a fresh read-receipt
tag each. The **asks** (A) are requests to the owner, with this spec's
recommendation.

### The rules and the layout

| File | Section | Change |
|------|---------|--------|
| `doc/rules/architecture.md` | 2.6 | add `internal/verdict/` (detectors, seed rules and the band-setting rule over primitive series, standard library only), created at the first code that needs it (2.1) |
| | read receipt | a fresh tag |
| `AGENTS.md` | Layout | the same line |
| `doc.go` (glossary) | | the terms of this spec's glossary list (below) |
| `doc/rules/testing.md` | none | no change: the tier tags and the flag are ordinary test constructs (10.2, 10.5) |

### The specs

| File | Section | Change |
|------|---------|--------|
| `spec-stability.md` | 7 | add the row **STB-F9 deflation with distress** (measure: `PriceIndex.Fisher` link and `Turnover.Volume`; causes L70, L71, L98); replace "Thresholds are placeholders that STY sets." with a pointer to STY 6.2 and `bands/regimes.toml`; keep the meaning of F1 to F8 |
| | 8 | in the scenario table, the "Asserts" bands of the scenarios of 5.11 (S01 to S165 but S15 and the harness tests) point to STY 5.11 as their home; the rows keep their setup; open question 3 ("the thresholds of the regimes") is **closed** by STY 6 |
| | 11 | no parameter changes; the thresholds are test data |
| `spec-reference-chain.md` | 4.1 | add to the layout `bands/regimes.toml` and `testdata/golden/stylised/<pack>.txt` and `<pack>.stability.txt` |
| | 7.1.5 | the band file shape gains `expects = [...]` per scenario table and the form `set_by = "commit=<id>;config=<hash>;seeds=<a>-<b>"`; FR-REF-024 keeps "at least the seeds of 8.2"; STY adds nothing to it and relies on no change |
| | 7.2 S05 | the four successive windows of 120 ticks become **season** windows of successive years (STY 4.1.3), with the run lengthened to four seasonal years (tier L) |
| | 7.2, 7.4, 7.7, 7.9 | add the band keys of STY 5.11 that REF's entries lack (`s23.band_bp`, `s25.swing_bp`, `s26.share_band_bp`, `s28.band_bp`, `s28.reopen_ticks`, `s29.disp_bp`, `s40.hunger_bp`, `s43.runway_bp`, `s44.grant_band_bp`, `s44.denied_ticks`, `s56.unfed_drop_bp`, `s61.journeys_factor`, `s62.split_band_bp`, `s63.journeys_band_bp`, `s65.role_share_bp`) and the keys STY adds to its own facts (`s01.unmet_bp`, `r8.unmet_bp`, `x1.worst_bp`, `p07.drift_bp`, `w01.mean_low_bp`, `w01.mean_high_bp`, `w02.n_band_bp`, `w02.ratio_band_bp`, `w06.fixed_point_bp`, `w06.money_cap_bp`, `d01.unfed_bp`, `m05.pool_share_bp`, `h02.offset_bp`, `h02.money_bp`, `t03.quote_band_bp`, `t03.pull_bp`) |
| | 4.5 | add `overlay/sink.toml`: a rate levy on the households with no paired grant, the provoked run of R9 (6.5.3) |
| | 7.6 | "All bands: STY" becomes "the keys of STY 3.6 and 5.11, set by 8.4"; open question 1 (bands) is answered by STY 8 |
| `spec-stability.md` | 7, F6 | reword F6 from "does not shrink" to "grows" (STY 6.3.4: REF S05's rule and the ordering of 6.5.2 need it); F1 "of a commodity class" is read as "of the whole economy and of each class" (STY 6.2); F2 may take the top-decile share of household money or reject it (STY 6.3.6) |
| | FR-STB-022 | "the world reports the unreachable reserve" is the validation warning `reserve_exceeds_money` of FR-AGT-060, raised at construction; a closed economy's mean money cannot rise (`spec-money-supply.md` 8.2 item 3) |
| `spec-transport.md` (A) | 15, TRN-A5 (STB S31) | reword S31: the two cases are the travel time equal to the shelf life and one tick inside it; as written the second cannot be "always delivered" |
| `spec-reference-chain.md` | 8.2 | the **hold-out** recommendation, not relied on: add the seeds 51 to 60 as a hold-out set that no band is set from, and FR-REF-024 "set from 1 to 50, checked on 51 to 60"; tier L gets the seeds 1 to 10 (REF gives none) |
| | FR-REF-022, FR-REF-023, 7.1.5 | an O check (a sign, an order, a derived bound) is asserted like an exact check while its key is unset, for the scenarios STY names (S01, S05, S06, S08, S14, S19, S20, S22, S28, S37 to S40); until then STY's O facts are reported |
| | 7.2 S04 | "the slope of the log of the quantity on the log of the price" becomes the integer arc elasticity of STY-W05 |
| | 7.2 S14, 8.2 | S14 is tier G at 1 800 ticks (STY 4.5 allows REF's lengths up to 1 800); the population-module leg is tier L |
| | 7.2 S22, X4, 7.5 | the detectors need three year windows from tick 0: REF's 1 095 ticks give exactly three; `five` at 400 ticks gives one, so STY-G01 on `five` **needs a longer run**: 1 095 ticks on seeds 1 and 2 (about 58 s a seed at REF's 53 ms) |
| | 7.2 S05, S12 | S05's growth key is STY-P03's `s05.range_growth_bp` for S12 too (no key of its own) |
| `spec-reference-chain.md` | 7.2 S20, X rows | the net-earner B part of STY-M06 needs a pack in which a group starts below the reserve and earns (a closed economy cannot raise the mean): until REF has one the B part is tier N and waits |
| `spec-observability.md` (A) | 5.3.5, FR-STB-012 list | the fields R2 reads from the snapshot: `MoneyPool`, `MoneyContract`, `MoneyTreasury` by region, `HeldByEstate` and `HeldByAgents` by market, and the household money rows for `Inequality.Top`; add them to the FR-STB-012 list |
| `spec-calibration.md` (A) | 7.3.5, X4 | detectors read whole years from tick 0 and do not exclude a first-year transient (STY 4.1.2); STY agrees to CLB's "working threshold" X4 only as the priors of 6.2 until a setting run sets them |
| `spec-reference-chain.md` | 17 | the file and package for tests that need `indicator`: `indicator/stylised_*_test.go` (STY) and `indicator/scenario_*_test.go` (REF), package `indicator_test`, sharing `stylised_helpers_test.go` (the recording observer), which STY owns |
| `spec-experiments.md` (A) | 12.2.3 | "the gate 1 to 10, the setting 11 to 50, the nightly 1 to 50 (STY 8.2)" becomes "the gate 1 to 10, the setting and nightly 1 to 50, tier L 1 to 10 (STY 8.2)"; STY does not rely on the edit |
| `spec-calibration.md` (A) | 7.1, 7.4.4, FR-CLB-021 and the lines that name `internal/regime` | the confirm seeds are STY's setting seeds 1 to 50 (tier L 1 to 10) as FR-CLB-021 now says; no "11 to 50" and no "held-out gate" remains; `internal/regime` is `internal/verdict` (STY 22) |
| `spec-indicators.md` | 9 row 20, FR-IND-049 | give `Unmet` its unit: the `Unfed` share of household-ticks of the window, in basis points; STY reads it as that |
| `spec-reference-chain.md` | 7.2 S19, S24 | "the mean ratio" becomes "the aggregate ratio (`HouseholdIncome`) when `OK`, the mean of ratios beside it" (STY-M03, M04) |
| | 7.2 S20, FR-STB-022 | the B part is "a net earner reaches the reserve within `s20.ticks`"; the warning `reserve_exceeds_money` is a separate exact check (STY-M06) |
| | FR-REF-023 | extend the closed list to the exact checks STY 5.11 marks X: S10, S13, S14, S19, S33, S36, S45, S54, S55 and the rest of that column |
| | FR-REF-027 | "from the events and the snapshot" adds "and INT's `Agent(a)` queries" (STY-W05, S04, S06) |
| | 8.2 | the runs of 1 095 ticks STY-D01, G01 and H02 need: `two`, `three`, `one-demand` seeds 1 to 4, `five` seeds 1 and 2 |
| | 17, 3 | scenario tests live in `economysim_test`; STY's fact tests live in `indicator`'s external test package and **re-run the same pack and seed** (a run is a function of configuration and seed, so the two agree), duplicating a small recording observer rather than importing across packages |
| `spec-indicators.md` | 11.1 | add the R9 row (deflation with distress: `PriceIndex.Fisher` chain and `Turnover.Volume`) and cite STY 6.2 for the thresholds; 21 question 7 is **answered** by STY 4.2.2 (no default weights; each fact names its own) |
| `spec-observability.md` (A) | 10.2 | add the R9 row; no new event or field |
| `spec-money-supply.md` | 17.6, S13 | "the price drift is report-only until STY sets a band" reads "report-only: STY-S05 gives no band, and why (neither a source nor a model fixes the direction)" |
| `spec-population.md` | 14.4, 18 | the bands of S37 to S40 are STY-W02, W03, W06 and 5.11; the warning `replacement_above_closed_ceiling` is read by acceptance as a rejection (C1) |
| `spec-resources.md` | 19.4 | the bands of S41 to S45 are in STY 5.11 (S43's `s43.runway_bp`, S44's `s44.grant_band_bp`) |
| `spec-skill-barriers.md` | FR-SKL-024, FR-SKL-025 | the historical premium range and the bands of S06 are STY-W04 and STY-D03 |
| `spec-viability.md` (A) | 13.2 A1 | "within the STY band" is `a1.price_band_bp` of STY-H01 |
| `spec-spawner.md` (A) | FR-SPN-021, 022 | **met, no SPN edit**: like-for-like replacement is not capped (FR-SPN-022), so STY-S02 uses `Renewal` times the leavers and the funding as the speed limit without the population module |
| `spec-external-sector.md` (A) | 6.7, 13 | say whether a quote is bounded to the realm's reference band (open question 1) |
| `spec-world.md` | 13 (STY row) | no change; STY relies on it as it stands |
| `spec-checkpoint.md` | 11 (STY) | no change; STY-H uses FR-CHK-050 |
| `spec-experiments.md` (A) | the section that lists what STY needs | the six rows of 10.1 |
| `specs/README.md` | index, code table, glossary | the STY row (exists; FR-STY-001 to 098, NFR-STY-001 to 004), the new terms; **Decisions**: "Where does a band live?", "What is a regime?", "What may be called stable?" |
| `doc/spec-roadmap.md` | existing specs; "Suggested next specs" | the STY row exists; "Suggested next specs" drops STY; Open 4 gets the closing text of 8.6 |
| `doc/documentation_index.md` | | the rows for the spec and for `specs/models/sty/` |
| `doc/specs/models/README.md` | contents | the `sty/` row (`seeds.py`, `detectors.py`, `worked.py`) |
| `doc/notes.md` | open notes | in the handoff: N-015, N-027, N-030, N-084, N-085, N-086, N-092, N-113 updated and six new notes |

New terms for the glossary: stylised fact, fact (STY), band, prior, setting seeds,
hold-out set, headroom, acceptance class, exact fact, ordinal fact, banded
fact, report fact, pack-asserted fact, confidence (of a band), blind-spot field,
detector, verdict, unknown (verdict), onset, healthy start, provoked run,
separation protocol, stability check, stability record, verdict record, tier.

## 22. Where this lives in the code

| Concern | Home |
|---------|------|
| the detectors, the seed rules, the band-setting rule: pure functions over primitive series | `internal/verdict/` (`detect.go`, `seed.go`, `band.go`), standard library only; unit tests in the package, and the fact tests of `indicator`'s external test package that call it |
| the fact tests of tiers G, N and L | **the external test package of `indicator`** (`indicator/stylised_*_test.go`, package `indicator_test`), the only place besides `examples/` and `otelobserver` allowed to import `indicator` (FR-IND-001); they import the root and `internal/verdict`; skipped with `-short`; tier L behind the build tag `stylised_long` |
| the recording observer and the measure helpers | `_test.go` files (testing.md 3.6), shared with REF's |
| the bands and the thresholds | `testdata/chains/bands/<pack>.toml` and `bands/regimes.toml` |
| the verdict records | `testdata/golden/stylised/<pack>.txt` |
| the registry checks of 12 | a test in `internal/verdict`'s own package (reading `../../doc/specs` and `../../testdata` by relative path, testing.md 10.4) that parses this spec and the bands files, as the parameter registry test of testing.md 6.7 does |
| the models | `doc/specs/models/sty/` |

