# Spec: Technology: a gain in yield held per region and field, raised by doing, by a trend, by discovery and by contact, and a bound on what it can do to the population loop

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-TEC-NNN` and
`NFR-TEC-NNN`, which tests cite (testing.md section 4). They are in section 18.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md),
decision 2), later stage (`vision.md` section 4, "Later"). It is off unless
`Config.Modules.Technology` is present, and with it off nothing in this spec runs:
no cell exists, no position is taken, no word is drawn, no event is emitted, no
state is saved, and a recipe's yield is the table of `spec-production.md` 6 and
nothing more. A world with the module on and no field configured has the state, the
events and the random stream of the world without it, and differs only in the
configuration hash, the module list and the names table that carry the module
(FR-TEC-001). The core does not depend on it. It needs
production (`spec-production.md` 3, 6.3, 6.4, 9.1, 10.3), the region graph
(`spec-regions.md` 3, 4.3, 7) and the calendar (`spec-calendar.md` 3.2, the
derived `TicksPerYear`). It uses hooks in the world (`spec-world.md` 6, 7, 8),
integration (`spec-integration.md` 3, 5), observability and the checkpoint. It
meets population and resources (`spec-population.md` 5, `spec-resources.md` 8)
in its stability section, and it reserves seams for infrastructure, skill
barriers, settlements, policy and indicators (section 8). Every change it needs
in another spec is listed in the handoff (section 17.2) and is not made here.

Nothing has run in the library and no acceptance band exists (notes N-015,
N-027). Every number here is a **placeholder**: a default, a bound or a number of
an example, and none is called stable. No source on the growth of productivity
was read for this spec (N-018): the shapes (a gain per doubling of cumulative
experience, a gain that spreads to a region that has less) are named in the
literature on learning curves and on diffusion, which was **not** read, and no
default comes from it. The four scripts of section 15 are stylised Docker Python
models of the rules of sections 3 to 7, 12 and 13, and an analysis; they are not
claims about the library. The settings of the worked examples (16), a farming
realm and a habitat, are invented **examples**, flagged as such, and the words in
them (`tillage`, `plain`, `fabrication`) are data and never branched on
(`AGENTS.md`, N-017). The normative text says **field**, **cell**, **gain** and
**experience**.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).
"Consumer" means the library consumer (a game or server), never a household
(N-001, N-017). **Technology** names this module and nothing in it is a recipe:
the word also appears in `spec-viability.md` for the matrix of activities of the
steady-state programme, which is a different thing and is written "the technology
matrix" there.

## 1. Purpose and scope

Until this module the productivity of a recipe is a table (`spec-production.md`
6): a number of basis points for each region, recipe and season, set by the pack
and moved only by a shock that a consumer sets and clears (`spec-production.md` 6.8) and by the
abundance of an endowment (`spec-resources.md` 4.7). Nothing in a run makes a
recipe yield more because it has been done, or because a region next door
learned how. A long run therefore has no growth and no learning, and the
population loop of `spec-population.md` 5 has a fixed point that only a shock
moves. This spec adds the one thing that was left open ("Productivity growth is
`spec-technology.md`; recipes are fixed for a run", `spec-production.md` 6.7):
a **gain** in yield that a region holds for a set of recipes, that rises by four
mechanisms a pack may switch on one by one, and that enters production as one
more factor of the effective yield. It covers:

- what a **field**, a **cell** and the **gain** are, and why the cell is a region
  and not an agent (section 3);
- the four **mechanisms**: learning by doing, a trend, discovery and diffusion,
  with their integer arithmetic and the order in which they act (section 4);
- how the gain **enters production** (section 5);
- the **position** in the tick, the **draws** (one word, at construction) and the
  one **call** (section 7), which is not a slot (6.4);
- what it asks of and gives to the other specs (section 8);
- parameters, validation, observability and saved state (sections 9 to 12);
- **growth against the Malthusian fixed point**, the target-income rule and
  deflation: an analysis with models (section 13);
- the register entries, what was run, worked examples, notes settled and needs,
  requirements, open questions and the home in the code (sections 14 to 20).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| A new recipe, a recipe that appears when a level is reached, a recipe that is withdrawn (an unlock) | not in version 1 (open question 3): `spec-production.md` 6.7 says "a pack replaces a recipe at construction"; the hook an unlock would need is named in 19 |
| Fixed capital: a processing site, a road, a drain; its build, its wear, its effect on a capacity or an edge | `spec-infrastructure.md` (CAP): "a facility is not a technology". A facility's effect is on a capacity, a rate or a limit and never on a yield (`spec-infrastructure.md` 3.1.3) |
| The skill of a household: tools, a traineeship, the skill premium | `spec-skill-barriers.md` (SKL). A technology gain is the region's; a skill is the household's. They compose (5.2) |
| Whether a person, a vehicle or a message carries knowledge from one region to another | not modelled (4.5.5): contact is a distance over the region graph. A carrier of knowledge is open question 2 |
| A research budget, a laboratory, a patent, a licence, a price for knowledge | not modelled. Knowledge has no price, no owner and no account; a charter or monopoly on a technique would be `spec-institutions.md` (ORG) and is not asked |
| What a controller does to growth (a subsidy, a ban) | a consumer's own controller, through the call of 7.1; a policy actor (PLC) has no lever on a gain (`spec-policy.md` 6.3.3) |
| The measure of productivity | `spec-indicators.md` (IND): a measure over `RecipeExecuted.EffectiveYield` and the snapshot rows (11.4); one preset is asked (section 17.2) |
| Prices, wages, the welfare ratio, the money supply | the owners; this module reads none of them (2, position 11) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | **Knowledge is held by a cell: a region and a field.** Not by an agent. A per-agent counter would be a hosted agent field, lost at a death, scaled by the weight, and it would duplicate what SKL owns (the household's skill). A cell is three integers and does not die | 3.2, 3.5 |
| 2 | **A field is a set of recipes that share what is learned.** A recipe is in at most one field; a recipe in no field never learns. A pack that wants one recipe to learn alone declares a field of one | 3.1 |
| 3 | **The yield table is the technology at tick 0.** Every gain is 0 at construction, so no parameter of this module moves the warm start, the viability check or the steady state of tick 0 (`spec-viability.md`), and a sweep of one needs no regenerated table (N-134) | 3.3, 8.7 |
| 4 | **The gain is a sum, capped.** `G = min(CapBP, g(X) + A)`, with `g` the learned part and `A` the advance (trend, discoveries, contact and the call). The factor of production is `10 000 + G`. Gains add and are capped, so capital in the sense of knowledge has a ceiling and no product of gains explodes (as `spec-infrastructure.md` 5.2.2) | 3.3, 4.6 |
| 5 | **Four mechanisms, each off at 0:** learning by doing (a gain per doubling of cumulative executions), a trend (a gain a year, a carried remainder), discovery (a step with a chance) and diffusion (a catch-up toward a frontier). The exogenous part is the trend, the call and discovery; the endogenous part is doing and contact | 4 |
| 6 | **Experience is physical.** It counts executions that completed, in the region where they ran, for a recipe of the field. It reads no price, wage, stock, count of households or measure: the module holds a stock that yield reacts to, like an endowment, and not an aggregate that anyone's rate is set from (WLD 9.4) | 4.1 |
| 7 | **Learning is logarithmic.** The gain is `DoublingBP` for each doubling of `(X + Base) / Base`, in integer arithmetic (a piecewise-linear log2 in 12-bit fixed point): exact at a doubling, within 0.086 of one between. Executions are bounded by households, so the learned gain grows ever slower: the loop through doing is self-limiting even before the cap | 4.2, 13 |
| 8 | **Discovery takes one word, once.** A discovery is a hit of a stateless keyed hash of (key, tick, cell); the key is one word drawn at construction. No word is drawn in a tick, so the stream of every later draw is unchanged by the module's activity (WLD 7.6.1), and a paired comparison across a parameter of this module stays aligned | 4.4, 6.3 |
| 9 | **Contact is a distance over the region graph.** A cell catches up toward the best gain of a region in contact, discounted by a weight that falls linearly with the shortest distance (REG 4.3) and is zero at `Reach`. The step reads a snapshot, so no order in which the cells are visited matters; a closed edge cuts contact; the receiver never passes the discounted frontier of the tick | 4.5 |
| 10 | **There is no forgetting except the call.** The gain never falls by itself. A consumer's call may lower `A` (a lost art, a forced loss): the module says so and does not model it (open question 4) | 7.1 |
| 11 | **No tick of feedback.** Experience accrues in scratch at P2.1 and P2.7 and is folded at one new position, **P8.3c**, at the end of the tick. Every position of a tick therefore sees one set of gains, and the cost anchor, the view and the start of a job agree | 6.1 |
| 12 | **The module creates and destroys nothing.** No ledger reason, no account, no flow, no goods and no money: a gain changes how many units a recipe makes, and the units are created by `recipe_output` as before | 5, 6 |
| 13 | **The effect on a recipe is multiplicative and sits right after the shock:** `E = site * shock * tech * trainee * abundance`. A running job keeps its output (`spec-production.md` 6.4) | 5.1 |
| 14 | **A gain can push a logistic stock past its maximum-yield point.** The fixed point of the population loop moves with the gain (`N*` proportional to it for a quota, a lower stock share for a stock), and a logistic stock is stable only above half its capacity. The validator checks the cap against the plan share; nothing in a run damps it | 13, 10.3 |

## 3. Fields, cells and the gain

### 3.1 The field

3.1.1. A **technology field** is data, keyed by ID like every configured entity
(architecture.md 9.1; `spec-config.md` 3.3):

```go
// TechField is a body of knowledge shared by a set of recipes. Plain data.
type TechField struct {
	ID          string   // matches the ID pattern of spec-config.md 3.3
	Recipes     []string // recipe IDs, sorted; each recipe in at most one field; 1 to MaxFieldRecipes
	CapBP       int64    // the largest gain of a cell, basis points; 1 to MaxGainBP; required
	Base        int64    // learning: executions already behind the field (4.2); 1 to MaxTechBase
	DoublingBP  int64    // learning: gain per doubling of (X + Base) / Base; 0 = no learning; 0 to MaxDoublingBP
	TrendBP     int64    // trend: gain added a year to every cell; 0 to MaxTechTrendBP
	DiscoveryBP int64    // discovery: expected discoveries a year for a cell that practises, in bp of one; 0 to MaxDiscoveryBP
	StepBP      int64    // discovery: the gain of one discovery; 1 to MaxStepBP
	CatchUpBP   int64    // diffusion: the share of the gap to the frontier closed a year; 0 to MaxCatchUpBP
	Reach       int64    // diffusion: the distance at which contact is lost, in the pack's distance unit; 1 to MaxDistance
	Label       string   // presentation only; not hashed
}
```

3.1.2. **Which fields a mechanism uses.** A mechanism is **on** when its rate is
above 0. A field that a mechanism that is off would use must be its zero value
(`ErrTechShape`), and one that an on mechanism requires must be set:

| Mechanism | On when | Requires | Must be zero while off |
|-----------|---------|----------|------------------------|
| learning by doing | `DoublingBP` > 0 | `Base` | `Base` |
| trend | `TrendBP` > 0 | | |
| discovery | `DiscoveryBP` > 0 | `StepBP` | `StepBP` |
| diffusion | `CatchUpBP` > 0 | `Reach` | `Reach` |

`ID`, `Recipes` and `CapBP` are always required. A field with every mechanism off
is legal and inert (the informational warning `field_inert`, 10.2); its cells hold
experience, which counts for nothing.

3.1.3. **A recipe is in at most one field** (`ErrTechRecipeOverlap`). A recipe may
be an ordinary recipe, an own-use recipe (`spec-subsistence.md` 3), a recipe a
trainee runs (`spec-production.md` 8) or one a hired worker runs
(`spec-employment.md` 6); it may be sited or not. It is the same recipe whoever
runs it and the field does not look at who. A recipe with `Builds` or `Repairs`
(`spec-infrastructure.md` 4.1) is legal in a field: its executions count, and its
yield factor is a test and not a quantity there (FR-CAP-011), so a gain changes
nothing it makes.

3.1.4. **`TechFieldID`** is the position of a field in the sorted order of the
effective configuration, a named integer type of the root package (architecture.md
3.3), in the names table of `WorldCreated` (11.1). A consumer persists the string ID
and resolves it once from `Config()` (`spec-integration.md` 3.4.2).

3.1.5. **Why a field and not a recipe.** One recipe per knowledge would make what is
learned in one recipe teach nothing to a recipe that uses the same tool. A region-wide pool
would make an unrelated recipe learn from a tool it never uses. The field is the author's choice of what
shares knowledge, and the smallest field is one recipe. It costs one list.

### 3.2 The cell

3.2.1. A **cell** is a (region, field) pair for every region that is not an outside
node (`spec-external-sector.md` 3.1.1: no agents, no market, no production) and
every field. Its state is three integers:

| Field | Meaning | Range |
|-------|---------|-------|
| `Experience` (`X`) | executions of the field's recipes completed in the region, cumulative | 0 to `MaxExperience` (2^50) |
| `Advance` (`A`) | gain added by the trend, discoveries, contact and the call of 7.1, basis points | 0 to the field's `CapBP` |
| `CatchUp` | the carried remainder of the diffusion step (4.5.4) | 0 to `10 000 * TicksPerYear - 1` |

A field has one more integer, `TrendCarry` (4.3), and the module has one, `Key`
(4.4.1), present when some field has `DiscoveryBP` above 0. The gain of a cell is
derived (3.3) and is not saved.

3.2.2. **Cells are created at construction**, in ascending (`RegionID`, `TechFieldID`)
order, all zero. A region added later does not exist (the region set is fixed for a
run, `spec-regions.md` 3), so the set of cells is fixed.

3.2.3. **What a cell is not.** It is not an account, a good, a lot or a recipe. It
has no price and no owner, it is in no ledger identity, and a household cannot hold,
sell or lose it. It is module state, like a facility (`spec-infrastructure.md` 3.2)
and an endowment's stock (`spec-resources.md` 3.2).

### 3.3 The gain

3.3.1. The **gain** of a cell, in basis points over 10 000, is

```
G = min(CapBP, g(X) + A)             0 to CapBP
```

with `g(X)` the learned part of 4.2 (0 when `DoublingBP` is 0). The **technology
factor** of a recipe in the field, in a region, is

```
Tech = 10 000 + G                    10 000 to MaxTechBP
```

and is 10 000 for a recipe in no field. It is one more term of the effective factor
of production (5.1).

3.3.2. **At construction `X` and `A` are 0, so `G` is 0 and `Tech` is 10 000.** The
yield table of `spec-production.md` 6 and `spec-resources.md` 7 is the technology
of tick 0: a region that is richer, or a field that is older, is a larger table
value and not a gain. There is no configured starting gain and no initial row
(N-104: the module adds no region-keyed table). The consequence is the one of
position 3: the viability check, the warm start and the steady state of tick 0 do
not read this module, and a sweep of any of its parameters starts every point
from the same table (`spec-experiments.md` 3.3.3; N-134).

3.3.3. **A gain does not fall by itself.** `g(X)` is non-decreasing because `X` is.
`A` is raised by the trend, a discovery and contact and lowered only by a call of 7.1.
A field is therefore **monotone** in a run with no call: `G` never falls
(FR-TEC-005). A pack that wants decline (a lost art, an obsolescence) has the call
or, at construction, a table value that is lower in a region.

3.3.4. **The stored clamp.** `A` is stored clamped to `[0, CapBP]`: a value above
the cap adds nothing to `G`, so keeping it would only hide a stock that a later
negative call would have to cross. With the clamp, a call that lowers `A` lowers
`G` at once whenever `G` is not already held up by `g(X)` (the identity of the
event, 11.1, is exact with it).

### 3.4 What counts as an execution

3.4.1. **Experience is `Executions` of a completed job.** At the completion of a
job of a recipe in the field (P2.1, and P2.7 for a job of duration 1), the number of
executions it completed (`RecipeExecuted.Executions`, the `k` of
`spec-production.md` 5.3) is added to the tick's scratch counter of the cell
(region of the job at its start, which equals its region at completion because a
relocation abandons the job, `spec-production.md` 5.4). The count is of executions
and not of households or of output: a cohort of weight `w` that runs `k` executions
adds `k`, so a world at a different scale (`spec-scale.md`) has the same experience
for the same households and the same work.

3.4.2. **What does not count:** an abandoned job (`RecipeAbandoned`: nothing was
made; open question 5 asks whether a part should), an execution refused at the start,
and a recipe in no field. The counter is read at the end of the tick, so a job that
completes at P2.1 and a job of one tick that completes at P2.7 count in the same tick.

3.4.3. **Trainee and hired executions count in full.** A trainee's execution makes
`TraineeYield` of a full one and a hired worker's makes `HireYieldBP` of it
(`spec-employment.md`); the execution is still done, and doing is what the field
counts. It is the simplest rule and the one with no extra parameter; a pack that
wants teaching to weigh less lowers `DoublingBP` of the field. It is the stated
bias of 13.7.

### 3.5 Why not per agent

A per-agent experience was the other reading of the brief. It is rejected for four
reasons, each testable:

1. It is **hosted agent state** (N-128): one more integer per agent and recipe in the
   agent record, so the checkpoint grows with the households (5 000 agents of the
   first consumer), and a cohort needs a rule for the experience of a weight change
   and a split (`spec-scale.md` 4.8) that the cell does not.
2. It is **lost at a death and at a removal**: a household of experience 40 000 dies
   and the knowledge goes with it, so the loop has a generational delay that the
   population loop already has (13) and a second one here would be an unmodelled
   cobweb.
3. It **duplicates SKL**: a household's skill is a tool and a traineeship
   (`spec-skill-barriers.md` 2, 4). A second household-level state would need a rule
   for how the two relate and, if it did not, a skilled worker and an experienced one
   would be two different things with no definition of either.
4. It makes **yield depend on who runs a job** and so adds a term to the view that a
   strategy reads (`spec-agents.md` 6.3) for every agent; the cell adds one factor
   that the existing field `E` already carries.

What is lost: an individual who practises more is no better than its neighbour, and
a migrant leaves its learning behind (a carrier of knowledge is open question 2).

## 4. The four mechanisms

The gain of a cell rises by four mechanisms, each of which a pack switches on by
giving its rate (3.1.2). They are defined here as arithmetic; the position and the
order are in 4.6 and 6. All of it is integer: no floating point appears in a rule
(architecture.md 3.5), and every product is checked against the bounds of 12.2
(12.3).

### 4.1 What accrues

Between ticks nothing changes. Within a tick the only thing that accrues is the
**execution counter** of each cell, a scratch value of the tick (3.4): the sum of the
`Executions` of the jobs of the field's recipes that completed in the region at P2.1
and at P2.7. It is cleared at P8.6 with the module's other per-tick counters; it is
not state (12.1). Everything below happens at P8.3c from that counter and from the
saved state.

### 4.2 Learning by doing

4.2.1. **The law.** With `v = X + Base` (at least `Base`, so at least 1) and `k =
bits.Len64(v) - 1` (the floor of its base-2 logarithm), the **fixed-point logarithm**
of `v`, in units of 1/4096 of a doubling, is

```
top12(v) = v >> (k - 12)      if k >= 12
         = v << (12 - k)      otherwise                 (4096 <= top12(v) < 8192)
lg12(v)  = 4096 * k + (top12(v) - 4096)
```

and the **learned gain** is

```
g(X) = DoublingBP * (lg12(X + Base) - lg12(Base)) / 4096        truncating toward zero
```

It is `0` at `X = 0`, non-decreasing in `X`, and **exactly `j * DoublingBP`** when
`X + Base = 2^j * Base` (models/tec/law.py part 1: exact at 21 doublings). Between
two doublings it is linear in `v` and so within 0.086 of a doubling of the true
logarithm (0.083 measured over 200 000 random `X`: 16.7 basis points of gain at
`DoublingBP` 200). The error is a rule constant of the law, not a tunable, and the
law is part of the golden output.

4.2.2. **What it says.** Each **doubling of the cumulative work** adds the same
gain: the first `Base` executions add `DoublingBP`, the next `2 * Base` add another,
the next `4 * Base` another. A field that is a hundred times older (a larger `Base`)
learns more slowly from the same work. With the placeholders `Base` 100 000 and
`DoublingBP` 200, a region of 1 000 households that completes one execution each
tick gains 200 basis points in 0.3 years, 400 in 0.8, 600 in 1.9 and 1 200 in 17
(models/tec/law.out): a fast start and an ever slower climb. The rate of the learned
gain falls as `1 / (X + Base)` while `X` can rise by at most the households a tick,
so **the loop through doing is self-limiting before the cap** (13.2).

4.2.3. **Why logarithmic, and why a gain and not a factor.** The shape is the one
that is usually called an experience curve; no source was read (N-018), and it is
chosen for two properties that can be tested: it is bounded by construction for any
run of finite length (a run of `10^9` executions has `lg12` of at most `62 * 4096`),
and it needs no division by anything but a power of two. It is an **additive** gain
over 10 000 and not a compounded factor, so that the gains of the four mechanisms add
and the sum is capped (position 4); for gains up to the cap the difference from a
geometric law is second order (`(1.02)^10` is 1.219 against 1.200 additive) and a
pack that wants a geometric law lowers `DoublingBP`.

4.2.4. **The experience saturates.** `X` is at most `MaxExperience` (2^50, about
`1.1 * 10^15`): a run of `10^6` executions a tick needs `10^9` ticks to reach it, so
the clamp is unreachable in any run and is not counted. It exists so that `X +
Base` fits an `int64` for every `Base` of the bound (12.2).

### 4.3 The trend

4.3.1. A field with `TrendBP` above 0 adds, to **every cell of the field**, an
amount of gain a year. The amount is applied by the carried remainder of
`spec-world.md` 9.1, held once for the field:

```
TrendCarry = TrendCarry + TrendBP
add        = TrendCarry / TicksPerYear
TrendCarry = TrendCarry mod TicksPerYear
A          = A + add                          for every cell of the field
```

At P8.3c each tick. Over any whole number of years the total is exactly `TrendBP`
a year (models/tec/law.out part 2: 411 over three years at 137, with no remainder).

4.3.2. **It is exogenous and linear in time.** It is the module's one *configured*
growth law, and ADR 0001 warns against hard-coded macro laws: this one is data, off
at 0, one field of a pack that wants a steady drift (a game that gives a region
half a percent a year) and a pack that wants any other shape uses the call of 7.1.
It adds to a cell where no recipe of the field is run; the gain is then idle until
one is (no cost, no effect). The trend is capped with everything else (4.6).

4.3.3. **A fractional rate is not a rate of the economy.** `TrendBP` is a number of
basis points of gain *over the table*, not a percentage of the current yield: 50 a
year is half a percent of the table's yield each year for ever, and a smaller share
of a larger yield as the gain grows. The growth rate of the factor falls as `1 /
(10 000 + G)`; 13.2 uses the form.

### 4.4 Discovery

4.4.1. **The key.** If some field has `DiscoveryBP` above 0, the world takes **one
word** from its source at step 8c of construction (`spec-world.md` 5.1.2),
after the creation draws of the initial agents (draw D1) and before the audit of
step 9, and keeps it as the module's `Key` (an unsigned 64-bit value, saved, 12.1).
It is the draw D16 of 6.3. Without a discovery field the module takes no word. This
word is a **module-level** word and not a creation draw of an agent, so it is not
what `spec-world.md` 5.7.2 forbids to make conditional; 6.3.4 says what that
section and 5.7.3 then need.

4.4.2. **The hash.** Let `C` be the number of cells and `c` the index of a cell in
ascending (`RegionID`, `TechFieldID`) order. For tick `t`:

```
K(t, c) = finalise( Key + (t * C + c + 1) * 0x9E3779B97F4A7C15 )      modulo 2^64
```

with `finalise` the SplitMix64 finaliser of `spec-world.md` 7.1.2 (the one of
`internal/ordering`). The index `c` follows the canonical order of the pack, so a pack
whose regions are renamed so that their order changes gives a cell another `c`, and
with discovery on other hits: the discovery of a cell is a function of (key, tick,
canonical position), and the same holds for the rest of the world, which is ordered by
`RegionID` (`spec-world.md` 6.1.2). The wrap of the modular product is the hash, not an overflow
(architecture.md 3.5 states it for `spec-resources.md` 5.3.3, whose `K(t, id)` has
the same shape: `t * C + c + 1` is distinct for every (tick, cell) of a run).

4.4.3. **The hit.** A cell **practises** in a tick when its execution counter is
above 0 (4.1). A practising cell of a field with `DiscoveryBP` above 0 **discovers**
when

```
high word of ( K(t, c) * (10 000 * TicksPerYear) )  <  DiscoveryBP
```

(the reduction of `spec-world.md` 7.2.1 applied to a key: one multiplication, no
loop, no rejection; its bias is at most `10^9 / 2^64`). The probability in a tick is
`DiscoveryBP / (10 000 * TicksPerYear)`, so a cell that practises through a year
discovers `DiscoveryBP / 10 000` times in expectation: 500 is one in twenty years,
10 000 is one a year. models/tec/law.out part 3: over 4.4 million cell-ticks the hit
count is within 1.6 standard deviations of the expectation at every rate tried, two
keys and two cells give independent hits.

4.4.4. **The effect.** A discovery adds `StepBP` to the cell's `A`. It is a **step**
(a lump of gain), the module's one discrete event, and it is the thing a world that
wants luck in progress turns on. It does not depend on how much the cell practises
(only that it does), so there is no scale effect of discovery: a larger region does
not discover faster. A scale effect is open question 6, and the learning of 4.2
already gives a larger region a faster climb.

4.4.5. **Why a keyed hash and not a draw.** A draw is positional: one extra word early
in a run changes every later word (`spec-world.md` 7.6.1), and the number of
practising cells in a tick depends on the economy, so a draw per cell would shift the
stream of the clearing keys and of every belief draw by an amount that is a function of
what the economy did. A keyed hash of (key, tick, cell) costs one word in the world's
life and nothing after: it is a function of the tick and the cell, as the ration's
tie-break of `spec-resources.md` 5.3.3 is, and it is seeded through `Key`, so two
seeds discover on different ticks. The cost is that a cell's chance in a tick is a
fixed function of (`Key`, tick, cell) and cannot be changed by an event in the run: a
discovery is not an outcome of the economy's own randomness. That is the sense of
"exogenous".

4.4.6. **A reseed.** A restore with `WithReseed(salt)` (`spec-checkpoint.md` 6.8)
gives the branch its own stream. A module that keeps a key must re-derive it in the
same way, or every branch discovers on the same ticks and the reseed would not
branch the module. The rule is `Key' = finalise(Key xor finalise(salt))`, taken at the
restore and saved with the branch; it is a rule constant and part of the golden
output. The edit is owed to CHK 6.8 (17.2).

### 4.5 Diffusion

4.5.1. **Contact.** For a field with `CatchUpBP` above 0 and `Reach` set, the
**contact weight** from a region `s` to a region `r` is, in basis points,

```
w(d) = (Reach - d) * 10 000 / Reach           for 1 <= d < Reach     truncating
     = 0                                      for d >= Reach, or no path
```

with `d` the shortest distance from `s` to `r` over the effective region graph
(`spec-regions.md` 4.3: the lexicographically smallest shortest path; 7: an edge
closed by a status is not an edge, an edge slowed counts its effective length), as
the world's `Distance(s, r)` returns it, and `ok` false meaning no path. A distance
is directed: a one-way road gives contact one way. `Reach` is in the pack's distance
unit (`spec-regions.md` 4.1). The weight is **not stored**: it is computed where it is used, from one read of the
graph's own distance table (`spec-regions.md` 4.3, rebuilt where that spec rebuilds it,
7.3, 4.4.4). A table of weights of the module would be `R^2` entries per field, 6.4 * 10^7
at `MaxRegions` 1 000 and `MaxTechFields` 64; computing it takes one multiplication and
one division, and the module keeps no state sized by regions squared. The product
`(Reach - d) * 10 000` is at most `10^13` (9.5).

4.5.2. **The frontier.** At the diffusion step (4.6), the **frontier** of a cell
`(r, f)` is

```
F(r) = max over every ordinary region s other than r of   G'(s) * w(d(s, r)) / 10 000
```

where `G'` is the gain of the cell of the same field in region `s` **as it stands
after the fold, the trend and the discoveries of this tick** (a snapshot taken once;
4.6). `F` is 0 when no region is in contact.

4.5.3. **The step reads a snapshot.** Every `F` is computed from `G'` before any
cell receives a catch-up, and the catch-ups are applied afterwards. No cell is
first and no order in which the cells are visited changes a result (models/tec/law.out
part 4: the added advance is the same for the six orderings of three cells, tested by
visiting the cells of the step in permuted orders, NFR-TEC-002). This is a property of
the diffusion step and not of the world: the world's own order of regions (the markets
clear in ascending `RegionID`, `spec-world.md` 6.1.2, and the keys of D7 are drawn in
that order) and the cell index of the discovery hash (4.4.2) follow the canonical
order of the pack.

4.5.4. **The catch-up.** If `F(r) - G'(r)` (the **gap**) is above 0, the cell's carry
takes a share of it per year by the carried remainder:

```
CatchUp = CatchUp + gap * CatchUpBP
add     = CatchUp / (10 000 * TicksPerYear)
CatchUp = CatchUp mod (10 000 * TicksPerYear)
A       = A + add
```

If the gap is 0 or below the cell takes nothing and its carry is unchanged. The
carry is state (a cell holds up to one tick's share of a gap that it has not yet
turned into a basis point).

4.5.5. **What it shows, and what it does not.** A lagging cell **approaches `w`
times the frontier from below and never passes it**: per tick it closes at most
`1 / TicksPerYear` of a gap that is itself below the leader's gain
(models/tec/law.out part 4: with the leader at 3 000, `w` 8 000 and 20% a year, the
near cell is at 435 after a year, 1 517 after five and 2 356 after twenty, against
the limit 2 400; the far cell, `w` 4 000, at 217, 758 and 1 178 against 1 200). It
cannot create knowledge: with every `w` below 10 000 a set of regions that all have
the same gain has no gap, and a region that leads is not changed by those behind it.
A gain obtained by contact is a gain, so a third region may use it as a frontier one
tick later (a relay). The relay is not a second mechanism, and it is bounded by the
**product of the weights of its legs**, not by the weight of the total distance: the
weight is linear in distance, so for legs `a` and `b` the product
`(1 - a / Reach) * (1 - b / Reach)` is larger than `w(a + b) = 1 - (a + b) / Reach`
(legs of 50 at `Reach` 100: 0.25 against 0). A relay therefore gives a region at most
the leader's gain times the product of the weights of the legs, which can exceed the
weight of the direct distance; in example A the relay's product (0.8 * 0.5)
happens to equal `w(60)` (models/tec/law.out part 4); with `plain`-`coast` at 70 it would give 0.4 against 0.3.

4.5.6. **Carriers are not modelled.** Contact does not use the rumours of
`spec-information.md`, the journeys of `spec-migration.md` or the trips of
`spec-merchants.md`. Rumours carry prices (N-110: version 1 rumours prices only, and
the information module is optional); a migrant or a merchant carries an account
and a plan, and a diffusion that required their modules would not exist in a world
that has neither; and the pack already states how far apart two regions are. What
this costs is stated in open question 2: a region that has no road to the leader and
many migrants from it gets nothing.

### 4.6 The cap and the order of the step

4.6.1. **The order inside P8.3c**, for every cell in ascending (`RegionID`,
`TechFieldID`) order within each step, and every step before the next:

1. **Fold.** `X = min(MaxExperience, X + counter)`; `Doing = g(X) - g(X_before)`.
2. **Trend.** Per field in ascending ID, the carry of 4.3; every cell of the field
   gets `add`.
3. **Discovery.** Per cell that practises, in cell order, the hit of 4.4.3; a hit adds
   `StepBP`; `Discoveries` counts them.
4. **Snapshot.** `G'` of every cell, `min(CapBP, g(X) + A)` with the `A` of steps 2 and
   3 (stored clamped).
5. **Diffusion.** Per field with `CatchUpBP` above 0, `F` and the gap from `G'` for
   every cell, then the catch-up of 4.5.4 for every cell.
6. **Clamp and events.** `A = min(CapBP, A)`; `G = min(CapBP, g(X) + A)`; the event
   `TechnologyAdvanced` for each cell in which anything accrued or was cut (11.1).

4.6.2. **The cap is the field's `CapBP`** and applies to the sum, not to each
mechanism: a field at the cap takes experience and counts it, and its `A` stays
at the cap. The cap is a **bound with a counter** (STB 5.1.5): the event carries the
basis points that the cap cut in the step (`Clamped`, 4.6.3), so a market whose
growth was held at the cap shows it, and a metric counts it (11.3).

4.6.3. **The identity of the event.** With `Before` and `After` the gain `G` of the
cell before and after the step and `Doing`, `Trend`, `Discovery`, `CatchUp` the
amounts that the four mechanisms *would have added* with no cap (`Doing` the change
of `g(X)`; `Discovery` the sum of the steps of the hits; `CatchUp` the `add` of
4.5.4),

```
After = Before + Doing + Trend + Discovery + CatchUp - Clamped            Clamped >= 0
```

with `Clamped` defined by that equation, so the identity is exact; that `Clamped` is
never negative and `After` never below `Before` in a step is a property of the rules
(models/tec/law.out part 5, 100 000 random cells). The call of 7.1 is outside it and has its own event.

4.6.4. **Nothing in the step depends on a price, a wage, a stock, a count of
households or an order of agents.** The step reads the counters (executions), the
saved state, the graph and the key. It is a function of those and of the tick: the
purity that `spec-world.md` 6.1.9 asks of a step.

## 5. How the gain enters production

### 5.1 The effective factor

5.1.1. `spec-production.md` 6.3 computes the **effective factor** `E` of a job as the
site factor, the shock, the trainee factor and the abundance. This module adds one
term, in the position between the shock and the trainee factor, and nothing else in
that section changes:

```
site  = table value (spec-production.md 6.2)               0 .. MaxYieldBP
E     = site * shock / 10 000                              shock: 6.8, default 10 000
E     = E * tech / 10 000                                  tech: 10 000 + G (3.3.1) of the cell of the recipe's field in the
                                                            executor's region at the start; 10 000 for a recipe in no field
E     = E * trainee / 10 000                               trainee: 10 000, or TraineeYield; for a hired job HireYieldBP
E     = E * abundance / 10 000                             abundance: spec-resources.md 4.7
```

Each division truncates toward zero and loses less than one basis point; the order and
the truncation are a rule constant of the golden output. With the module off, or for a
recipe in no field, `tech` is 10 000 and `E` is as before.

5.1.2. **A gain does not open a closed site.** `site = 0` gives `E = 0` whatever
`tech` is: a `Sited` recipe that a region has no profile for does not start
(`no_site`) and a profile of 0 does not start (`zero_yield`). Climate is data
(N-030) and a technology gain is not a profile: contact never carries a site factor
(position 9 says what it carries).

5.1.3. **The yield is read when a job starts** (`spec-production.md` 6.4) and a
running job keeps its output. A gain set at P8.3c therefore acts on the jobs that
start from the next tick, and a job of `Duration` `d` that started the tick before
completes at the old output: the effect of a gain reaches the supply with a delay of
up to `d` ticks, a delay that the register records next to the recipe's own
(14, TEC-A1).

5.1.4. **The largest factor is bounded.** `spec-production.md` 10.3 rejects a
configuration in which the largest weight times a quantity times the product of the
largest factors could exceed `MaxQuantity`. The module's largest term is
`MaxTechBP = 10 000 + MaxGainBP` (110 000 at the placeholders), and the check of 10.3
reads it in place of 10 000 for a recipe in a field; the edit is owed
(`spec-production.md` 10.3, 17.2). No other bound changes.

### 5.2 Composition

| Other term | How it composes with `tech` |
|------------|-----------------------------|
| Shock (`SetYieldShocks`) | multiplies: a shock of 3 000 on the recipes of a region yields 30% of what the region's technique would have made. A shock set after a gain does not undo it, and a gain after a shock does not clear it (the module never writes the slot, as CAP does not, `spec-infrastructure.md` 5.2.2) |
| Trainee (`TraineeYield`) | the trainee makes a share of the technique's output, not of the table's: a region with a better technique teaches a better output |
| Hired worker (`HireYieldBP`, `spec-employment.md` 6.1) | the same place as the trainee factor, which it replaces; the technique is the region's and the hired worker has it |
| Skill (a tool, a traineeship, `spec-skill-barriers.md`) | independent: a skill gates who may run a recipe and what it costs to learn to; a gain changes what a started job makes. A skilled and an unskilled worker of one region have one `tech` |
| Abundance (`spec-resources.md` 4.7) | multiplies: a gain raises the yield per drawn unit and does not change the draw (`Draws` is the quantity of endowment per execution, unscaled), so a thin stock still limits `k` by `floor(stock / draw)` and a gain raises what each unit made yields (13.4) |
| A facility (`spec-infrastructure.md`) | independent: a site facility limits how many executions run (`k0`), a route speeds an edge, a ground facility raises a capacity; none changes `E`, and the module changes none of them |
| Own-use (`spec-subsistence.md`) | the same factor; an own-use recipe that is in a field learns and gains like any other (13.5 for the effect on the floor under a market) |

### 5.3 Where the gain is read

The gain of a cell changes at one position (P8.3c) and at one call (7.1), so every
position of a tick reads the same `tech` for a recipe and a region:

| Position | What reads it | Why it agrees |
|----------|---------------|---------------|
| P1.2 | the cost anchor: `UnitCost` divides by `E` (5.4) | the value at the end of the previous tick |
| P2.5 | `k0` and the `Work` view's `E` (`spec-agents.md` 6.3) | the same value; no execution of this tick has changed it |
| P2.7 | the start: the output of the job is fixed with `E` | the same value |
| tick 0 | `spec-viability.md` (the solver, the warm start) | `G` is 0 (3.3.2) |

A completion of this tick adds to the counter and changes nothing a position of this
tick reads (position 11, FR-TEC-010).

### 5.4 The unit cost

`UnitCost` (`spec-production.md` 9.1) is `ceil(cost * CostShare / (quantity * E))`, so
a gain lowers the cost anchor of every output of the recipe by the same factor, from
the tick after P8.3c (the anchor is read at P1.2). The belief bands of
`spec-beliefs.md` 3.2.2 (the cost floor and ceiling at 70% and 300% of the full cost)
follow it down. The module does not move a price and does not feed a measure back;
what happens to prices is what the market does with a lower cost (13.6).

### 5.5 What a strategy sees

No new field. The `Work` view of `spec-agents.md` 6.3 already carries "the effective
yield `E` for this region and season", which is the product of 5.1.1 and so includes
`tech`; the default strategy's choice among recipes (net value by the belief centres
at the effective yield, `spec-agents.md` 7.1.4) sees a gain from the tick after
P8.3c. A consumer's strategy has no forward view of a gain: it reads what is, and
the module offers it no forecast, no experience figure and no gain of another
region (the world exposes no remote state to a strategy, N-106).

## 6. Position in the tick, draws and the call

### 6.1 The position and the scratch

6.1.1. **One new position, P8.3c**, for `spec-world.md` 6.10, after CAP's P8.3b (the
facility step) and before P8.4 (the optional audit):

| ID | What runs | Owner | Reads / writes | Draws | Events |
|----|-----------|-------|----------------|-------|--------|
| P8.3c | The technology step, with the module on: fold the execution counters of the tick, the trend, discovery, the snapshot of the gains, diffusion, the clamp (4.6) | TEC 4.6 | the counters, the saved cells, the graph tables, the key; writes `X`, `A`, `CatchUp`, `TrendCarry` and the derived gains | none (the keyed hash of D16 takes no word) | `TechnologyAdvanced`, per cell in ascending (`RegionID`, `TechFieldID`) order, after the events of P8.3b and before `TickEnded` |

6.1.2. **Two rows of the table gain a clause** (an edit, 17.2): P2.1 "the executions
of a completed job of a recipe in a field are added to the counter of its cell",
and P2.7 the same for a job of duration 1. The counter is scratch, cleared at P8.6.

6.1.3. **Why the end of the tick.** If the gain were folded at the completion, the
cost anchor of P1.2 of the same tick, the view of P2.5 and the start of P2.7 would
disagree for the agents that start after a completion in the rotation order, and the
rotation would decide which household got a better yield: the rotation order would
become a source of bias that the world's 10.1 works to remove. At the end of the
tick every position reads one set (position 11). The price is a delay of one tick
between a completion and its effect, which is below every other delay of the loop
(14).

6.1.4. **Why after CAP's step.** A facility's ruin or build at P8.3b does not change a
gain, so the order between them is a convention; P8.3c is after it so that the step
is the last of the tick that changes module state and its events are the last before
`TickEnded`. The audit (P8.4) reads no gain.

### 6.2 What a restore and a reseed do

A restored world continues exactly: the cells, the carries and the key are saved
(12.1), and the counters are empty at a boundary (a tick's scratch is cleared at
P8.6). `WithReseed(salt)` re-derives the key (4.4.6). `WithSeed` on a restore is an
error as always (`ErrOptionContext`).

### 6.3 The draws

6.3.1. The module adds **one row** to the table of `spec-world.md` 7.3 (the number
follows the draw IDs D13 to D15 that RSK took; "draw D16" is a row of that table and
is not the damper D16 of the register, `spec-stability.md` 5):

| ID | When and where | Taken by | How many | In what order | Owner |
|----|----------------|----------|----------|---------------|-------|
| D16 | construction, step 8c (after the creation draws, draw D1, and before the audit of step 9), only when some field has `DiscoveryBP` above 0 | the world: the module's key | 1 | once | TEC 4.4.1 |

6.3.2. **The module takes no word in a tick**, in any configuration: the trend, the
learned gain, the diffusion and the discovery are functions of the saved state, the
graph and the key (4.6.4). The words of one tick (`spec-world.md` 7.3.1) are
unchanged, so a pair of runs that differ in any parameter of this module is aligned
in the sense of `spec-experiments.md` 7.3 as far as this module goes, from the first
tick after construction. The one exception is a pair that differs in whether a
discovery field exists at all: the second run has one more word at construction,
which shifts every later word by one (WLD-A5). An experiment sweeps `DiscoveryBP`
between positive values, or keeps a field with the smallest rate, to stay aligned
(N-124).

6.3.3. **A module that is on and does nothing takes no word**: with no field, or no
field with `DiscoveryBP` above 0, there is no key and no draw
(`spec-world.md` 7.4.2). The creation draws of the initial agents (draw D1) are the
same with the module on or off, because draw D16 comes after them.

6.3.4. **What `spec-world.md` 5.1, 5.7.2 and 5.7.3 then say** (an edit owed, 17.2).
They state that nothing in construction takes a random number before step 8, that a
spec that adds a creation draw never makes it conditional on a module, and that
construction takes `2 * H_agents` draws and nothing else, so that the first draw of
the first tick is the `2 * H_agents + 1`th. The key is neither an agent's creation
draw nor a per-tick term, and 7.4.2 of that spec states the converse (a module that is on and does nothing takes
none); the edits, which add D16 as the word of a module that has work, are: 5.1.2 has the sub-step 8c, "the technology key, one word, when some field has a
discovery rate (D16)"; 5.7.2 carves out of its rule a **module-level word** that is
taken once at construction, after the creation draws, and only by a module that is on
and has work to do (7.4.2), and says that a creation draw of an agent is still never
conditional; 5.7.3 reads "`2 * H_agents` draws, and one more when the technology
module has a discovery field (D16); the first draw of the first tick is the
`2 * H_agents + 1`th, or the `2 * H_agents + 2`th with it". A pair of runs that
differs in whether a discovery field exists is not aligned from the first tick (6.3.2,
WLD-A5), which is the price of a word that no step could take without shifting the
stream.

### 6.4 Between ticks

The call of 7.1 changes `A` of a cell **at the call**: the gain is derived from
`X` and `A` and is read by the next position that reads it. No table is rebuilt (the
contact weights depend on the graph and not on a gain). The call is not a slot
in the sense of `spec-world.md` 8.2: it holds no value over a configured one, it
edits state once, as `AdjustEndowments` does (`spec-resources.md` 6.3.2), so
`Slots()` does not list it and a clearing entry has no meaning. Its effect on the
flow totals is none: no money and no goods move.

### 6.5 The order of the events

For `spec-world.md` 6.11, item 9 (phase 8): `BoundaryFlow`, `EndowmentChanged`, the
events of P8.3a (LND) and P8.3b (CAP), then `TechnologyAdvanced` per cell in
ascending (`RegionID`, `TechFieldID`) order, then `TickEnded`. A call between ticks
emits `TechnologyAdjusted` in ascending (`RegionID`, `TechFieldID`) order at the
call, with the tick of the next step, before that step's `TickStarted`.

### 6.6 Cost

With the module on and fields configured, P8.3c is one pass over the `R * F` cells
(`R` ordinary regions, `F` fields): the fold, the clamp and the event. Diffusion adds
`R^2` reads of the distance table, each one weight computed on the fly, per field with `CatchUpBP`
above 0 (`MaxRegions` 1 000 with 64 such fields is 6.4 * 10^7 reads a tick, a bound and
not a first consumer's number); discovery adds one hash per practising cell. For the first consumer's world (5 regions, say 8 fields: 40 cells,
and 8 * 20 = 160 pair reads) the step is well below a microsecond-scale budget per
tick and the module's allocation with the no-op observer is none (NFR-TEC-003). The
counter is one integer add at a completion (O(1)). The only structure that grows with
the economy is none: the module's state is sized by regions times fields.

## 7. Calls

### 7.1 `AdjustTechnology`

7.1.1. The one public call (`spec-integration.md` 5, a one-off adjustment of module
state, between ticks only, in the shape of 3.3 there: a slice, one entry per key,
atomic, an event per changed key in canonical order):

```go
// TechnologyAdjustment is a signed change of the advance of one cell. Plain data.
type TechnologyAdjustment struct {
	Region  RegionID
	Field   TechFieldID
	DeltaBP int64 // signed; at most MaxGainBP in magnitude
}

// AdjustTechnology changes the advance of cells between ticks. It is atomic.
func (w *World) AdjustTechnology(changes []TechnologyAdjustment) error
```

7.1.2. **Rule.** For each entry, `A' = min(CapBP, max(0, A + DeltaBP))`. The part of
the delta that the bounds did not allow is **lost** and reported. The effect is at
the call (6.4); the cell's experience, carry and the other cells are untouched; the
cap is the field's. The call moves no money and no goods and creates no account.

7.1.3. **Errors, in the order checked** (`spec-integration.md` 3.2): `ErrInStep` and
`ErrWorldFailed` first, as every call, then `ErrModuleOff` (naming `technology`), then
for each entry in the order given `ErrUnknownRegion` (a `RegionID` outside `[0, n)`),
`ErrOutsideNode` (an outside node has no cell), `ErrUnknownTechField` (new, a
`TechFieldID` outside `[0, F)`), `ErrOutOfBounds` (`|DeltaBP|` above `MaxGainBP`),
`ErrDuplicateID` (two entries for one key). The argument errors are typed
`*ValidationError`s with a `Path` that names the entry (`changes[2].field`)
(`spec-integration.md` 3.2.2). No state changes on an error.

7.1.4. **The event** `TechnologyAdjusted` is emitted for each entry whose `A`
changed (11.1). An entry with `DeltaBP` 0, or one that bounds cut completely, emits
none.

7.1.5. **What it is for.** A scripted discovery (a game's research reward, a
scenario of STY or EXP: "region `r` learns a technique at tick 100"), a lost art
or a forced loss (a negative delta after a loss of the people who practise it), and a consumer's
consumer's own controller that subsidises a field (a policy actor sets slot keys only and has no lever on a gain, `spec-policy.md` 6.3.3). It is the module's whole exogenous
interface, and the trend, the discovery and the call are the three ways a gain is
set from outside the economy.

7.1.6. **What is not offered** (N-131): setting experience directly (a gain by doing
is earned); a duration or a decay of an advance (a slot has no timer,
`spec-integration.md` 5.2.2); a call that makes a region learn a recipe it does not
run; a bundle of fields; and a query of a cell, which the snapshot gives (11.4).
Each is reopened when a consumer asks.

### 7.2 Reading

A consumer reads the gains from the snapshot's `TechRow` list (11.4) and the
configuration (`Config()`), and the agent's own `E` from its `Work` view. There is no
new query: a read of a cell is a read of a snapshot, a second-class query of the cost
`O(cells)` (`spec-integration.md` 6.1).

## 8. What this module asks of and gives to the others

### 8.1 Production

The one term of 5.1.1, the overflow check of 5.1.4, the two clauses of 6.1.2 and the
table row of `spec-production.md` 6.7 ("A recipe that changes over time:
`spec-technology.md`"), which this spec answers as: a gain, not a recipe; recipes are
still fixed for a run.

### 8.2 Infrastructure

"A facility is not a technology" (`spec-infrastructure.md` 1, 3.1.3) holds both ways:
the module has no facility kind and CAP has no yield effect. A recipe whose output
grows with a gain and whose throughput is limited by a site facility
(`spec-infrastructure.md` 5.1) shows both: the site limits `k`, the gain raises what
each execution makes. **An unlock** (a facility class or a recipe that a level makes
available) would be a `Needs`-like clause on the class or on the recipe naming a field
and a gain; it needs a refusal reason in PRO's order and a field on a CAP class, and
is not asked (open question 3).

### 8.3 Skill barriers

No change. A trainee's executions count (3.4.3) and `TraineeYield` multiplies the
technique's output (5.2). The skill premium of SKL 3 is a wage and is not read here;
a gain that raises output per household changes the welfare ratio and the observed
wage through the market and not by a rule of this module. Skill-biased change (a gain
that only skilled households get) is not modelled and would be a second factor keyed
by the executor's role.

### 8.4 Population and resources

The module's reason to exist in a world that has the population module is also its
main hazard: a gain raises output per household, the households respond, and the
fixed point moves. The analysis is section 13; what it gives the neighbours is:

- **RES** (`spec-resources.md` 8): `N*` (a quota) is proportional to `tech`; a stock's
  fixed share `sigma*` falls by `1 / tech` for a linear curve; a logistic stock is
  stable only above 1/2. 8.3.1 of RES's validator evaluates at the plan share and
  does not know of a gain: the module's warning `technology_past_maximum_yield`
  (10.3) is the same inequality with the cap.
- **POP** (`spec-population.md` 5): the loop gain `k` is unchanged by `tech` (it
  scales `r_full`, not the slope), so the stability conditions of 5.4 hold; the
  fixed point moves and the premium of 13.2 appears.

### 8.5 Information, migration, merchants

No dependency in either direction (4.5.6). Rumours still carry prices only; a migrant
and a merchant carry what they carried. The one coupling is indirect: the households
that a gain attracts (births, or a migrant's earned ratio, `spec-migration.md` 5) are
the existing responses.

### 8.6 Settlements

No hook is used. A size class could gate learning or discovery by a flag a field
names (`Needs`, the form of `spec-infrastructure.md` 3.1, 8): a settlement class for a place of
research. It would be one string on `TechField` and one reader in the settlements table
(`spec-settlements.md` 6.3); it is not asked and a gate never removes what exists
(SET R3). Open question 7.

### 8.7 Viability, the stability checks, experiments, calibration

- **VIA** (`spec-viability.md`): tick 0 is the table (3.3.2), so the solver, the warm
  start and the viability checks are unchanged, and no parameter of this module is in
  the list of paths that move the steady state (`spec-viability.md` 11): the path
  `modules.technology.*` is `no` in the list, with the reason "a gain is 0 at tick 0".
- **STY** (`spec-stylised-facts.md`): the stability check "a warm start that stays near
  the steady state" (`spec-stylised-facts.md` 7.3) assumes a steady state that does not drift, so a claim of
  stability is made **with the module off** (TEC-S1: on with no field equals off).
  Every price fact of STY is relative to a flow-balancing price, a base window or a
  freight (4.2), which survives a drifting steady state; a run with the module on is
  held to the growth scenarios of 14 and not to the check of the drift of the table.
- **EXP** (`spec-experiments.md`): a sweep of any parameter of the module needs no
  regenerated warm start (position 3); a paired comparison across a rate of the module
  is aligned from construction (6.3.2) except when it turns discovery on or off.
- **CLB** (`spec-calibration.md`): every field of `TechField` is a row of the registry
  with the class proposed in 9.4 (CLB decides, 4.6); the four rates default to 0, which
  is the owner's statement that the mechanism is off (9.1.2): a calibration that moves a
  rate away from 0 does so because a fact shows a loop, never the other way.

### 8.8 Money and the stylised regime of deflation

A gain lowers the unit cost of a recipe's outputs (5.4) and, with the money stock
fixed, a larger quantity of goods sells for the same money: the price level of those
goods falls. `spec-stylised-facts.md` 6.2 (the row R9) defines **R9 deflation with distress** as: `link_k` of the price index
is at most `10 000 - fall_bp` in each of `windows` consecutive windows **and** the
later window's `Turnover.Volume` (the units traded, `SumV`: `spec-indicators.md` 7) is
below the earlier's. Growth raises the volume (units) while it lowers the
price, so the second conjunct fails and the regime is not detected; but the first
conjunct holds, and a detector or a reader that looks at the price index alone would
call growth deflation distress. Two statements for STY, in the handoff: R9's second
conjunct is the discriminator and must stay a clause on **units**, not on value (the
value `Turnover` can fall with the price index), and a scenario with growth on
(TEC-S2) is the test that R9 stays silent in it.

### 8.9 Indicators

`RecipeExecuted.EffectiveYield` is already in the event (the owner's list in
`spec-observability.md` 4.1.3), so the productivity of a recipe in a region is a
consumer's measure: the executions-weighted mean of `EffectiveYield` over a window,
in basis points of the table. IND may want it as a preset (`Yield`, a tally of
`RecipeExecuted`, class 2 of its cost table); it is asked in 17.2 and is not needed
for any rule. The gain of a cell is a snapshot row (11.4); the aggregate "growth" of a
an economy is a measure and never an input (macro is measured, never set).

### 8.10 Employment, land, saving

They read `E` through the view and the events and change no rule: the hired worker's
factor of EMP 6.1 composes (5.2); LND's rent a unit of endowment can bear rises with
the yield per drawn unit (`spec-land.md` 7, "changes no rule here"); SAV's project
test values a site facility by the best job with and without it at the household's
view, which includes `tech` (a better technique makes a site more valuable, with no
rule in SAV). RSK's embargo as an edge status cuts contact (4.5.1) with no rule here.

## 9. Parameters

### 9.1 The module and its types

9.1.1. The module is present when `Config.Modules.Technology` is non-nil
(`spec-config.md` 3.2: no separate flag):

```go
// Technology is the module of productivity growth.
type Technology struct {
	Fields []TechField // at most MaxTechFields, keyed by ID, sorted by ID
}
```

In a file it is `[modules.technology]` and `[modules.technology.field.<id>]`, with
`recipes = ["..."]`, the form of `[modules.infrastructure]` and `[modules.merchants]`
(N-102). `TechField` is the type of 3.1.1.

9.1.2. **The module has no tunable of its own and no default field.** An empty
`Fields` is legal: the module is then on and inert (FR-TEC-001). Every number is a
field of a `TechField`, which a pack authors, and every rate has the zero value as
its default, which turns the mechanism off. `CapBP` is required and has no default,
because a default would be a claim about how much a technique may grow
(position 4).

### 9.2 The fields and the bounds

Every field is a tunable of the pack and has an owner, a bound and a doc line
(architecture.md 9.5, 9.7). The defaults and bounds are placeholders (N-015, N-027).

| Field | Scope | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| `Recipes` | field | the recipes that share what is learned | **required** | 1 to `MaxFieldRecipes`, each a recipe, none in another field |
| `CapBP` | field | the largest gain of a cell | **required** | 1 to `MaxGainBP` |
| `Base` | field (learning) | executions already behind the field | **required** with `DoublingBP` > 0 | 1 to `MaxTechBase` |
| `DoublingBP` | field | gain per doubling of `(X + Base) / Base` | 0 (off) | 0 to `MaxDoublingBP` |
| `TrendBP` | field | gain added a year | 0 (off) | 0 to `MaxTechTrendBP` |
| `DiscoveryBP` | field | expected discoveries a year for a practising cell, in bp of one | 0 (off) | 0 to `MaxDiscoveryBP` |
| `StepBP` | field (discovery) | the gain of one discovery | **required** with `DiscoveryBP` > 0 | 1 to `MaxStepBP` |
| `CatchUpBP` | field | the share of the gap to the frontier closed a year | 0 (off) | 0 to `MaxCatchUpBP` |
| `Reach` | field (diffusion) | the distance at which contact is lost | **required** with `CatchUpBP` > 0 | 1 to `MaxDistance` |
| `Label` | field | a human-readable name; presentation only, not hashed | empty | at most `MaxLabelBytes` (`spec-config.md` 4.1), `ErrOutOfBounds` |

The bounds are in `bounds.go` (architecture.md 9.6), chosen so that every product of
this spec fits in 128 bits (9.5):

| Bound | Placeholder | Why |
|-------|-------------|-----|
| `MaxTechFields` | 64 | fields are authored by hand; a field is a body of knowledge |
| `MaxFieldRecipes` | 64 | at most the recipes of a pack that a field could share |
| `MaxTechBase` | 10^12 | an author's count of executions; `X + Base` fits `int64` with `MaxExperience` |
| `MaxDoublingBP` | 10 000 | a doubling of the cumulative work doubles the table's yield at most |
| `MaxTechTrendBP` | 10 000 | a year adds at most the table's yield again |
| `MaxDiscoveryBP` | 100 000 | ten discoveries a year |
| `MaxGainBP` | 100 000 | a technique raises the table's yield at most elevenfold: `(10 000 + 100 000) / 10 000` |
| `MaxStepBP` | `MaxGainBP` | a discovery is at most the cap |
| `MaxCatchUpBP` | 10 000 | at most the whole gap in a year (the per-tick share is then below the gap) |
| `MaxTechBP` | `10 000 + MaxGainBP` | the largest `tech` |
| `MaxExperience` | 2^50 | unreachable (4.2.4) |
| `MaxDistance` | `spec-config.md` 4.1 | the region graph's own bound |

### 9.3 Rule constants

Beside their code, not parameters (architecture.md 9.6): the log law and its 12 bits
of fraction (4.2.1); the position of `tech` in the effective factor (5.1.1) and its
truncations; the linear contact weight (4.5.1); the frontier as a maximum over a
snapshot and the closing by a carried remainder (4.5); the hash, its multiplier
`0x9E3779B97F4A7C15`, the reduction and the reseed rule (4.4, 4.4.6); the order of the
steps of P8.3c (4.6.1); the stored clamp of `A` (3.3.4); the rule that a recipe is in
at most one field (3.1.3); the position P8.3c (6.1) and the draw D16 (6.3). The
experience that a job adds is its executions (3.4.1): a rule, not a weight.

### 9.4 Rows for the parameter-owner table

For `specs/README.md`, every field a row with its default and a validation test
(architecture.md 9.7, testing.md 6.7). All are owned by this spec:

| Parameter | Scope | Owner |
|-----------|-------|-------|
| `Technology` | Modules (module; nil = off) | `spec-technology.md` section 9.1 |
| `Fields` | Technology | section 9.1 |
| `ID` | TechField | section 3.1 |
| `Recipes`, `CapBP`, `Base`, `DoublingBP`, `TrendBP`, `DiscoveryBP`, `StepBP`, `CatchUpBP`, `Reach`, `Label` | TechField | sections 3.1, 9.2 |

The fields are **proposed for the registry of knobs** of `spec-calibration.md` 6 in
the same change (`spec-stability.md` 9.10); the class of each is CLB's decision
(`spec-calibration.md` 4.6), and the proposal is: `DoublingBP`, `TrendBP`, `DiscoveryBP`,
`StepBP` and `CatchUpBP` are **class K** knobs (drivers); all but `StepBP` have the flag `z` (the owner's
default is 0, so the grid is built on the pack's non-zero value, CLB 4.2.1; `StepBP`'s default is `required`, so its base is the pack's non-zero value and it takes no `z`, CLB decides) and an unknown
search range (`U`). The flag `off=<v>` is for class D only (CLB 4.2.1), so no `off=` is
claimed here: that a default of 0 means "off" is the owner's statement of 9.1.2, and
whether CLB's ordered test of 4.3 places a rate that no damper counts in S before K is
its own to settle. `CapBP` is the **damper strength**
of TEC-A1 and TEC-A3 (the one bound on what the module can do; required, so it has no
off value); `Reach` is a **knob with a derived condition** (it is meaningful only
relative to the distances of the pack: it must be above the shortest and, for the
module to do anything, below the longest finite path); `Base` and `Recipes` are
**structural choices**; `ID` and `Label` are presentation. No search range is
claimed; CLB adds the rows. The loops each knob acts on are in 14.

### 9.5 Products and bounds

Every product of the module, with the placeholders, fits `int64` and the checked
128-bit helper (`spec-currency.md` 7) where it would not:

| Expression | Largest | Where |
|------------|---------|-------|
| `X + Base` | `2^50 + 10^12 < 2^63` | 4.2.1 |
| `lg12(X + Base) - lg12(Base)` | `62 * 4096 = 253 952` | 4.2.1 |
| `DoublingBP * (that)` | `10^4 * 253 952 = 2.54 * 10^9` | 4.2.1 |
| `gap * CatchUpBP + carry` | `10^5 * 10^4 + 10 000 * MaxTicksPerYear (10^5) = 2 * 10^9` | 4.5.4 |
| `TrendCarry + TrendBP` | `MaxTicksPerYear + 10^4` | 4.3.1 |
| `K(t, c) * (10 000 * TicksPerYear)` | a 128-bit product (`bits.Mul64`); the high word is below `10^9` | 4.4.3 |
| `G * w / 10 000` | `10^5 * 10^4 / 10^4` | 4.5.2 |
| `(Reach - d) * 10 000` | `Reach` at most `MaxDistance` 10^9, so `10^9 * 10^4 = 10^13` | 4.5.1 |
| `site * shock * tech * trainee (or hired) * abundance` | `10^5 * 10^5 * 1.1 * 10^5 * 10^5 * 10^5 = 1.1 * 10^25`, with the factor of a hired job up to `MaxYieldBP` 10^5 | 5.1.4: the checked helper |

(models/tec/worked.out, the last part.)

## 10. Validation

### 10.1 Errors

Each has a failing-case test that breaks only that rule (testing.md 3.9). All need the
root mapping of N-075 and a row of `spec-config.md` 7.3.

| Sentinel | When | Where |
|----------|------|-------|
| `ErrTechShape` | a field that an off mechanism does not use is non-zero, or one that an on mechanism requires is 0 (3.1.2); `Recipes` is empty | `Validate` |
| `ErrTechRecipeOverlap` | a recipe is in two fields (3.1.3) | `Validate` |
| `ErrUnknownTechField` (new) | `TechFieldID` outside `[0, F)` in a call | 7.1 |
| `ErrUnknownRecipe` (reused) | a recipe of a field is not a configured recipe | `Validate` |
| `ErrDuplicateID` (reused) | two fields with one ID, or two entries of one key in a call | `Validate`, 7.1 |
| `ErrOutOfBounds` (reused) | a field outside its bound in 9.2 (a `Label` above `MaxLabelBytes` included), or `\|DeltaBP\|` above `MaxGainBP` | `Validate`, 7.1 |
| `ErrTooMany` (reused) | more than `MaxTechFields` fields or `MaxFieldRecipes` recipes in one | `Validate` |
| `ErrModuleOff` (reused), `ErrUnknownRegion`, `ErrOutsideNode` (reused) | a call with the module off, or naming a region that does not exist or is an outside node | 7.1 |

### 10.2 Warnings

Informational or advisory; a warning never fails validation (`spec-config.md` 7.2).
Each is a `Warn` at construction with `rule` and `path`.

| Rule | Fires when |
|------|-----------|
| `field_inert` | every mechanism of a field is off (3.1.2): the field is legal and its cells count experience for nothing. Informational |
| `field_unused` | no recipe of the field is on any role's `Recipes` list or effective own-use list: no execution can ever be counted, so the field never learns (a trend and a discovery still add gain that nothing uses) |
| `diffusion_no_contact` | `CatchUpBP` above 0 and, at construction, no ordered pair of ordinary regions has a finite distance below `Reach` (or there are fewer than two ordinary regions): the diffusion step can do nothing. A closed edge later can make this true again; it is checked at construction only |
| `technology_past_maximum_yield` | the inequality of 10.3 holds for a logistic resource a recipe of the field draws on, with the population module on |
| `trend_outpaces_schedule` | the inequality of 10.4 holds, with the population module on |

### 10.3 The logistic stock and the cap

10.3.1. **The check.** A recipe of a field draws on a resource of shape `logistic`
(`spec-resources.md` 3, 4.2) through `Draws`; the population module is on; `sig` is
the resource's plan share (`PlanShareBP`, RES 11.1) and the curve `a` its abundance
curve (RES 4.7) at `sig` has a slope above 0 (`eps_bp` > 0, RES 8.3.1: a flat curve
has no stock feedback and RES warns of it separately). With `a` in basis points the
warning fires when

```
a(5 000) * (10 000 + CapBP)  >=  a(sig) * 10 000
```

evaluated with the checked 128-bit helper, `a` read from the curve by the integer
interpolation of RES 4.7.

10.3.2. **Why this inequality.** At the plan share the pack says a household reads
the replacement ratio `r*`, so `r*` is `r_full * a(sig) / 10 000`. With a gain a household
reads `r_full * a(sigma) * (10 000 + G) / 10^8` at stock share `sigma`, so the fixed
point is where `a(sigma*) * (10 000 + G) = a(sig) * 10 000`. A logistic
stock is stable only for `sigma* > 5 000` (RES 8.3, row `logistic`), and `a` is
non-decreasing, so `sigma* <= 5 000` exactly when `a(5 000) * (10 000 + G) >= a(sig) *
10 000`. The check uses `G = CapBP`, the largest the field can reach. For a linear
curve `a(sigma) = sigma` and a plan share of 6 000 it is `CapBP >= 2 000`, which is the
`T_crit = 1.2` of models/tec/malthus.out part C.

10.3.3. **What it assumes, and which way it errs.** The proportionality of the ratio
to output is RES 8.2's own assumption. A household that takes a gain as rest
(13.5), or a price that falls with output (13.6), gives a ratio that rises less than
`tech`, and a smaller shift of `sigma*`. The check therefore **over-warns**: the
direction of safety. It is a sufficient
condition of the stock's instability under the assumption and not a statement that a
run below it is safe (the basin of a logistic stock is narrow even
above one half, RES 8.3 item 2: the model of part C collapsed at `T` 1.25 and cycled at
1.20 and was stable, with a damped swing, below 1.18).

10.3.4. **For a `refill` stock** the fixed point is always stable (RES 8.3) and the
stock share falls with the gain; the overshoot threshold `R >= 4 kappa eps (1 -
sigma) / sigma` rises as `sigma` falls (0.16 a year at 0.6, 0.24 at 0.5, 0.56 at 0.3,
models/tec/malthus.out part B). RES's `resource_overshoot` is evaluated at the plan
share and does not know the gain; a pack with a trend or a learned gain evaluates it at
`sigma` of the cap by hand, and the validator reports the share as the informational
diagnostic `technology_stock_share` (a field's smallest `sigma*` over its refilling
resources at `CapBP`), found by walking the segments of the curve in integer
arithmetic. It never fails.

### 10.4 The trend and the schedules

For the population module on, with `r*` the replacement ratio, `dR` and `dX` the rise
and the run of the segment of `birth - death` at `r*` (RES 8.3.1, POP 5.7) and
`knot_max` the highest ratio of the birth schedule's knots, the warning
`trend_outpaces_schedule` fires when

```
r* + TrendBP * dX / dR  >  knot_max
```

evaluated in integers as `TrendBP * dX > (knot_max - r*) * dR` (no division, no
rounding). The left side is the premium of 13.2 at `G` = 0 and `eta` = 1, the ratio at
which a population that tracks a trend of `TrendBP` a year settles. That is the
**largest** premium in `G` (it falls as `10 000 / (10 000 + G)`) and the **smallest**
in `eta` (a smaller elasticity gives a larger premium). The check therefore
over-warns for a pack whose gain has grown (at a `G` of 3 000 the premium is 23% below the one tested), and under-warns for a curve with `eta`
below 1: it is neither necessary nor sufficient, an advisory. Above the highest knot
the schedules are flat: the births cannot rise further, and the population trails the
fixed point by more than the linearisation says. With the defaults (`r*` 12 000, `dR`
400, `dX` 8 000, `knot_max` 16 000) the threshold is `TrendBP` 200 a year.

### 10.5 What is not validated here

That a `Base` is a sensible count; that a cap is a sensible gain; that the field's
recipes are the ones a pack means (a recipe that is a leaf of no chain is legal);
that a trend is plausible (no source, N-018). That a gain does not make the viability
check of `spec-viability.md` fail: it cannot, because tick 0 is the table.

## 11. Observability

### 11.1 Events

For `spec-observability.md` 4.1 (a new group, 4.1.9) and the rules' table 4.3. Both
are gated by `Modules.Technology`; neither has a map, a pointer or a float.

| Event | Owner | Position | Class | Gate | Fields (besides `Tick`) |
|-------|-------|----------|-------|------|-------------------------|
| `TechnologyAdvanced` | TEC 11.1 | P8.3c, (`Region`, `Field`) order | A (bounded by regions times fields) | technology | `Region`, `Field`, `Executions` (the cell's counter this tick), `Experience` (after), `Discoveries`, `BeforeBP`, `AfterBP` (the gain `G` before and after), `DoingAddedBP`, `TrendAddedBP`, `DiscoveryAddedBP`, `CatchUpAddedBP` (what each mechanism added, before the cap), `ClampedBP` (the cut of the cap: 4.6.3), `AtCap bool` (after) |
| `TechnologyAdjusted` | TEC 7.1 | call, (`Region`, `Field`) order | C (one per changed key of a call) | technology | `Region`, `Field`, `DeltaBP` (signed, asked), `BeforeBP`, `AfterBP` (the gain), `LostBP` (the part of the delta the bounds cut) |

11.1.1. **`TechnologyAdvanced` is emitted for a cell in which anything accrued** in
the step: a counter above 0, a trend, a discovery, a catch-up or a cut; it is not
emitted for a cell with nothing to report (so a module that is on and has no
practising region and no trend emits none, FR-TEC-024). Every count is present in
every event even when zero (OBS 4.2.4). `AfterBP = BeforeBP + DoingAddedBP + TrendAddedBP +
DiscoveryAddedBP + CatchUpAddedBP - ClampedBP` (4.6.3), a test. The four `...AddedBP`
fields are amounts added in the step; they are not the rates `TrendBP`, `DiscoveryBP`
and `CatchUpBP` of `TechField`.

11.1.2. **Volume.** At most one event per cell per tick: `R * F` (40 for the first
consumer's world with eight fields), which is below one percent of the order and
trade events of such a world (`spec-observability.md` 8.3); it is class A and
silent when the module is off or has no field.

11.1.3. **The names table.** `WorldCreated` and `WorldRestored` carry the field IDs
in the names table (`spec-observability.md` 3.2): one string per `TechFieldID`, in
the sorted order.

### 11.2 Reasons and causes

None. The events carry no `Reason` or `Cause`: a field is advanced by four fixed
mechanisms whose amounts are separate fields, and the adjustment has one cause (the
call).

### 11.3 Metrics

For the rules' table 5.3 and 5.3.1, with one new attribute `economysim.field` (the
fields of the configuration: bounded, rules 5.4.1):

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.technology.gain` | Gauge | `{basis_point}` | `economysim.field`, `economysim.region` | `TechnologyAdvanced` and `TechnologyAdjusted` (`AfterBP`); the adapter keeps the last value for a tick with no event and takes the level from `WorldRestored`'s snapshot rows: a row of the seeding table of `spec-observability.md` 6.4.2, seeded from `TechRow.GainBP`, a cell with no row being 0 (owed, 17.2) |
| `economysim.technology.experience` | Counter | `{execution}` | `economysim.field`, `economysim.region` | `TechnologyAdvanced` (`Executions`) |
| `economysim.technology.discoveries` | Counter | `{discovery}` | `economysim.field` | `TechnologyAdvanced` (`Discoveries`) |
| `economysim.technology.clamped` | Counter | `{basis_point}` | `economysim.field` | `TechnologyAdvanced` (`ClampedBP`) |

For the reference chain on `five` with three fields: 15 series of the gauge, 15 of
the experience counter, 3 and 3 (36). The gain is a level, so it is an event-fed
gauge and not a delta instrument.

### 11.4 The snapshot

A **`TechRow`** per (region, field) with `Experience` above 0 or `AdvanceBP` above 0
(`spec-observability.md` 5.3, a new group): `Region`, `Field`, `Experience`,
`AdvanceBP`, `GainBP`. It is a function of the saved state (N-125): the gain is
derived from the experience and the advance, and the field's cap is the configuration.
A snapshot taken at a boundary (the only place one exists) holds the gains that the
next tick reads.

### 11.5 Logs and keys

- **`Info`**: the construction summary ("world created", `spec-observability.md` 7.2)
  gains `fields` (the count of fields), and `technology` joins the `modules` list.
  Nothing at `Info` in a tick.
- **`Debug`**, guarded by `Enabled`: "technology advanced" per cell with `region`,
  `field`, `gain` and the counts; "technology adjusted" per changed key of a call.
- **No `Warn` in the tick.** A bound that acts (the cap) is counted by
  `TechnologyAdvanced.ClampedBP` and `economysim.technology.clamped`, so the
  checklist of `spec-observability.md` 7.4 (every `Warn` has a counterpart) has
  nothing to add. The construction warnings of 10.2 are the configuration's.
- **New log keys** (for the rules' table 3.3.2): `field` (a field ID, the string of
  the names table), `fields` (a count in the construction summary), `gain` (a gain,
  basis points, `int64`). Messages are constant and lowercase (3.3.1 of the rules).

### 11.6 Cost with no observer

Events are built only when an observer was supplied (rules 4.5); the counters are
integers; the step allocates nothing. With the default logger and observer a `Step`
of the module allocates no more than the world's baseline (NFR-TEC-003).

## 12. State contributed to a checkpoint

### 12.1 What is saved

A module section "technology", gated by `Modules.Technology`, in the dependency
order of `spec-checkpoint.md` 4.2 after the last later-stage module present (the tag is 33,
`spec-checkpoint.md` 4.2):

| Item | Content | Order |
|------|---------|-------|
| cells | per cell: `Experience`, `Advance`, `CatchUp` (three varints) | ascending (`RegionID`, `TechFieldID`); the count is `R * F` from the configuration and is not written |
| fields | per field: `TrendCarry` | ascending `TechFieldID` |
| key | `Key` (an unsigned 64-bit value), **present only when some field has `DiscoveryBP` above 0** | |

The size for the first consumer's world is a few hundred bytes. **Not saved**
because derived: the gain of a cell (a function of `X`, `A` and the configuration),
the contact weights (not stored: computed from the graph tables, 4.5.1), and the per-tick counters (cleared at P8.6, so empty at every
boundary). **No per-agent field**, so nothing is added to the agent record (N-128).

### 12.2 Restore

Validation, in the fixed order of `spec-checkpoint.md` 6.2, each failure
`ErrCheckpointCorrupt`: the cell count matches the configuration (the hash is
checked first); `Experience` in `[0, MaxExperience]`; `Advance` in `[0, CapBP]` of its
field; `CatchUp` in `[0, 10 000 * TicksPerYear)`; `TrendCarry` in `[0, TicksPerYear)`;
the key present exactly when a discovery field exists. A restored world continues
exactly as an uninterrupted one: the same gains, the same discoveries on the same
ticks (the key is saved), the same carries (FR-TEC-028). The format version changes
with the first release that has the module, and a fixture joins `testdata/checkpoints/`.

### 12.3 The hash

The module and its fields are in the configuration hash (`spec-config.md` 8.1), in
ID order; `Label` is presentation only (it joins the allow-list of presentation-only
fields of `spec-config.md` 8.3, as `Region.Label` is). The key is **not**
configuration: it is state drawn from the stream, like the PCG. The hash therefore
differs between a world with the module present and one without, and so do the
`ConfigHash` and `Names` fields of the snapshot header and of `WorldCreated`, and
`Modules`: FR-TEC-001 excepts exactly these.

## 13. Growth against the Malthusian fixed point, the target-income rule and prices

This section is **analysis**, in real numbers with a stated rounding of the integer
rules it describes; the models of 15 run the rules. It answers the brief's stability
concern: a gain raises output per household, and a population that responds to the
ratio it reads has a fixed point that moves (`spec-population.md` 5, `spec-resources.md`
8). Nothing here shows that a default is stable (N-027).

### 13.1 What a gain moves

With the population module on and a fixed factor that binds (`spec-resources.md` 8.1
to 8.2), let `T = (10 000 + G) / 10 000` and `r_full` the ratio a household reads at its
full draw at `T = 1` (N-109). The ratio is proportional to output, which RES 8.2
assumes and 13.5 qualifies.

| Fixed factor | Fixed point before | Fixed point with a gain |
|--------------|--------------------|-------------------------|
| a quota (flat curve, regime A) | `N* = N_bind * r_full / r*` | `N* = N_bind * r_full * T / r*`: **proportional to `T`**. `N_bind` is unchanged: a gain changes what a unit drawn yields and not how many are drawn |
| a stock with a curve (regime B) | `a(sigma*) = 10 000 * r* / r_full` | `a(sigma*) * T = 10 000 * r* / r_full`: the stock share **falls**; for a linear curve `sigma* = sigma_0 / T`; and `N* = F(sigma* C) / (h * TicksPerYear)` rises as the regeneration at the lower share does |

That is Clark's result of `spec-population.md` 5 (`evidence.md`: the income at which
births equal deaths is fixed by the two schedules alone, independent of technology),
stated for this library: the gain is taken as people, at the replacement ratio, and
the standard of living returns to `r*`.

### 13.2 The population follows, with a premium and a time constant

With `tau` the fractional growth of `T` a year, `eta` the elasticity of the ratio to
the households (1 for a quota, below 1 for a curve) and `k = r* g' eta / 10 000` the
gain of `spec-population.md` 5.4 (0.06 a year at `eta` 1 for the defaults: `g' = 400 /
8 000`), the linearised loop is

```
dn/dt = - k n - tau / eta          n = ln N - ln N*(T), the distance from the fixed point of the day
```

(a population that is off its fixed point decays toward it at `k`, and a fixed point
that moves at `tau / eta` pulls it). Three statements follow, each against the model
(models/tec/malthus.out part A):

1. **A steady drift is tracked at a premium.** The population settles at a constant
   distance below the moving fixed point and the households read a ratio above `r*`
   by
   ```
   premium (bp of ratio) = r* * tau / k = 10 000 * TrendBP / ((10 000 + G) * g' * eta)
   ```
   For `TrendBP` 50 a year and `G` 2 000 the prediction is 833 basis points above
   `r*`; the model gives 818 at year 40 (the table is below). The premium needs about
   one time constant `1 / k` (16.7 years at `eta` 1) to form, so it is smaller before.
2. **A step is absorbed with the time constant `1 / k`.** A discovery of +1 000 takes
   the ratio from 12 000 to 13 200 at once, and the gap between the new fixed point
   (110 000) and the population falls to `1/e` of its start at year 17 (the
   linearisation: 16.7); by year 80 the ratio is 12 019.
3. **A cap pays the premium back, without overshoot.** With `TrendBP` 50 and `CapBP`
   1 000 the gain stops growing at year 20; the ratio falls from 12 677 to 12 368 at
   year 30, 12 203 at 40 and 12 019 at 120, and the largest `N / N*'` over the run is
   0.998: a monotone approach. The loop is first order and its delay is small against
   its time constant (`k * tau_delay` of 5.4 is 0.004), so the condition of POP 5.4
   holds with a wide margin.

| `TrendBP` | year | `G` | `N / N*` | ratio | premium | predicted |
|-----------|------|-----|----------|-------|---------|-----------|
| 25 | 40 | 1 000 | 0.965 | 12 437 | 437 | 455 |
| 50 | 40 | 2 000 | 0.936 | 12 818 | 818 | 833 |
| 100 | 40 | 4 000 | 0.891 | 13 465 | 1 465 | 1 429 |
| 50 | 60 | 3 000 | 0.937 | 12 809 | 809 | 769 |

Agreement is within 4% at year 40 for every `TrendBP` tried, 2% to 9% at year 60 (5.2% at `TrendBP` 50, 9% at 100, where `G` is 6 000 and the linearisation is poorest), and **22% to 27% below the formula at year 20** (the premium forms over more than one time constant: 348, 677 and 1 297 against 476, 909 and 1 667). The **premium is the quantity
the register watches**: a trend that is large against `g'` puts the ratio past the
highest knot of the birth schedule (10.4), where the births are flat and the
population trails the fixed point by more than the linearisation says.

### 13.3 What a gain can do to a stock: the refilling stock and the logistic stock

13.3.1. **A refilling stock** (`spec-resources.md` 4.2) is always stable and its
share falls with the gain (models/tec/malthus.out part B: `sigma*` 0.545 at `T` 1.1,
0.500 at 1.2, 0.400 at 1.5, within 0.1% of `0.6 / T`, and the population
`N*` of 113 636, 125 000 and 150 000 within 0.1%). The threshold for a monotone
approach rises as the share falls: `R >= 4 kappa eps (1 - sigma) / sigma` is 0.16 a
year at 0.6, 0.24 at 0.5, 0.36 at 0.4 and 0.56 at 0.3. A refill that is quick enough
today may overshoot after a gain; RES's warning does not see it (10.3.4).

13.3.2. **A logistic stock** is stable only above half its capacity. The gain moves
the fixed point toward that point, and **below it the fixed point is a saddle**: the
stock regenerates less the less it has (RES-A2). From the fixed point of `T` = 1
(`sigma*` 0.6, a linear curve, so the critical gain is 20%):

| `T` (gain) | `sigma*` | `N*` of `T = 1` | smallest `N` | outcome (300 years) |
|------------|----------|-----------------|--------------|---------------------|
| 1.00 | 0.600 | 1.000 | 1.000 | monotone |
| 1.05 | 0.571 | 1.020 | 1.000 | damped oscillation, undershoot 2% |
| 1.10 | 0.545 | 1.033 | 1.000 | damped oscillation, undershoot 3% |
| 1.15 | 0.522 | 1.040 | 1.000 | damped oscillation, undershoot 4% |
| 1.18 | 0.508 | 1.041 | 0.982 | small residual swing; stock 0.517 |
| 1.20 | 0.500 | 1.042 | 0.881 | sustained cycle; stock 0.399 |
| 1.25 | 0.480 | 1.040 | 0.004 | **collapse** |
| 1.40 | 0.429 | 1.020 | 0.004 | **collapse** |

The edge is where the inequality of 10.3 says it is, and the swing below it is the
narrow basin of RES 8.3 (a damped oscillation of a few percent: the stock itself is
close to the point at which it stops recovering).

13.3.3. **A ramp crosses it late and is silent until it does.** A trend that raises
`T` from 1 with no cap crosses 1.2 at year 20, 40 or 80 (`TrendBP` 100, 50, 25).
The stock is exhausted (below 2% of its capacity) **21, 75 and 118 years after the
crossing**; ten years before it households read 10 817, 16 874 and 12 981 (replacement ratio `r*`
12 000). The fastest ramp is already below `r*` ten years before and the population
is shrinking, so it gives some early sign; for **the two slower ramps the ratio is at
or above `r*` ten years before and looks healthy (the middle one a boom, 16 874) until
the stock goes**. The slower the ramp the longer the system sits on the unstable side
while the fixed point it tracks has gone. A run that is watched through the welfare ratio
and the household count alone does not see it coming for the slower ramps; the stock share is in the
snapshot (`spec-observability.md` 5.3) and the validator's warning is the only
earlier signal (10.3). No rule of the module damps it, and none should: it is the
population's response to a gain, and a pack that wants a logistic stock and growth
chooses a cap below the threshold, a refuge, or a refilling stock.

13.3.4. **Without a fixed factor** (no resources module, or no recipe that draws
on a binding endowment: `no_fixed_factor` of `spec-population.md` 9.2) there is no
fall of the ratio with the households, so a gain is a gain in the ratio for good while
births exceed deaths: the population grows until the money of the pool is spent on
stakes (`N_M = M / stake`, POP 5.5) or `HouseholdCap` binds. The module adds no
warning of its own: POP's applies, and a pack that turns growth on with the
population module and no fixed factor has chosen a population that is bounded by
money.

### 13.4 Depletion and a gain

A thin stock lowers `E` through the abundance and raises `UnitCost` (RES-A6); a gain
raises `E`. They multiply (5.2), so a gain **compensates a thin stock in yield per
drawn unit, not in the stock**: the ration (`spec-resources.md` 5.3) still cuts `k` to
`floor(stock / draw)` per start, and the draw per execution is unchanged. A field
whose recipes draw on a depleting stock therefore grows the yield of what it can still
draw and not the draws, and the stock's trajectory is the households' (13.3).

### 13.5 The target-income rule: a gain is rest first

The rule of `spec-welfare-and-wages.md` 5 makes a household work until it has earned
its target, so a rise in what a day yields is met by fewer days (the backward bend of
`spec-stability.md` 6.1). The consequence for a gain is an analysis, not a run
(models/tec/farmer.out): with `n` identical producers, a demand `Q = A p^-e`, a
target `K` and a work floor `D_min` (`MinWorkShare`):

- **Interior** (`D = K / (p y T)` between the floor and the maximum): the price is
  `(n K / A)^(1 / (1 - e))`, **independent of `T`**; output per tick is `n K / p`,
  also independent; **the days worked fall as `1 / T`**. A 10% gain is a 9% cut in
  work and nothing else. In the example (150 days at `T` 1) a 50% gain is 100 days
  and the floor binds.
- **At the floor** the days are `D_min`, output is `n D_min y T`, and income per
  household changes as **`T^(1 - 1/e)`**: it rises for `e > 1`, is flat for `e = 1`
  and **falls for `e < 1`**. For a staple (`e` 0.5: `spec-agents.md` 11.6) a gain of 50%
  past the floor cuts income by a third (0.667), and doubling cuts it to 0.5. The
  market is pinned there for `e < 1` anyway (the interior equilibrium is unstable,
  STB-L14), at a price that falls with the gain.

Three consequences. (1) The premium of 13.2 is an **upper bound** for a population
with the rule (the ratio rises less than output while days can fall); the check of
10.3 inherits the direction of safety. (2) **For a staple, productivity growth is
immiserising once the floor binds**: the household's income is lower though its yield
is higher, the failure ladder (`spec-spawner.md` 5) and exit (damper D14) act on it, and
the producers who leave raise the price: the loop of TEC-A4. (3) An own-use floor
(`spec-subsistence.md`) changes the marginal household and not this arithmetic.

### 13.6 Prices

5.4 lowers the cost anchor of the field's outputs by the gain. In a closed economy
with the money stock and its velocity unchanged, more goods for the same money is a
lower price level, which the markets reach through the belief bands and the
clearing, with the delays of `spec-beliefs.md` 5 and the cost anchor's one tick. It is
**deflation caused by growth**, not by a sink, a hoard or a levy with no paired grant
(`spec-money-supply.md` 9). The register's regime R9 is *deflation with distress*;
8.8 says what keeps the two apart (the volume clause), and TEC-A5 asks the test. A
pack that does not want the price level to fall has a money flow (`spec-money-supply.md`
4) or a policy actor (PLC): the module sets no money and reads none.

### 13.7 What the analysis does not show

- **One function of the ratio.** The ratio is taken proportional to output, with no
  price, no wage and no target-income response in the models. 13.5 says the direction
  of the bias (the premium and the shift of `sigma*` are upper bounds).
- **No price feedback in the models of 13.2 and 13.3.** A closed economy's mean ratio
  is fixed at `1 + W/U` (N-054) and a gain in the household's own output changes what
  it can sell and what it can buy at once; the library's economy is what the models
  replace by one assumed function.
- **A trainee's executions weigh as a full one** (3.4.3), so a region that trains many
  learns faster than one that has the same output: a bias, with no parameter to
  correct it but `DoublingBP`.
- **One field.** The models have one field and one resource. Several fields on
  recipes of one chain give gains that differ by recipe (a gain in the intermediate
  and none in the final), and the chain's relative prices move; nothing was run.
- **The cap is a plateau, not a ceiling in reality.** It is a bound for the model's
  sake (position 4); a setting that wants unbounded growth raises `MaxGainBP` in a
  build of its own and meets the overflow check of PRO 10.3 first.

## 14. Entries for the stability register

By the checklist of `spec-stability.md` 9 (FR-STB-003). Provisional IDs: the supervisor
numbers them (N-103) and merges those that name a loop the register has. Dampers are
`TEC-Dn`, scenarios `TEC-Sn`. Every number is a placeholder (N-015, N-027).

### 14.1 Loops, hazards and amplifiers

**TEC-A1. Doing, gain and the households that do it.** *Kind:* loop. *Loop:* the
executions of the field's recipes (+) experience (+) the gain (+) output per household
(+) the ratio the households read (+) births less deaths (POP) (+) households (+)
executions. Every link is positive, so polarity **R**, closed through the population
loop; the fixed factor's own balancing loop (RES-A1, a lower ratio as households
rise) is the other half and is what makes the pair stable. *Delay:* the population's
time constant `1 / k` (16.7 years at `eta` 1, `spec-population.md` 5.4), plus a job's
`Duration` for a gain to reach supply (5.1.3) and the welfare window of the role.
*Owner:* TEC 4.2, 13; the other half is POP-A1 (L75) and RES-A1. *Damper:* **TEC-D1**
(the logarithm: the gain per execution falls as `1 / (X + Base)` while executions
rise by at most the households a tick), **TEC-D2** (the cap), RES's fixed factor
(a lower ratio as households rise: RES-D2), `HouseholdCap` and the money-limited
population `M / stake` (POP 5.5). *Undamped:* a pack with the population module on, no
fixed factor and a large cap: the population is bounded by money and by
`HouseholdCap` only, and the gain is paid for in people (13.3.4). *Condition:* a
finite cap (always true) and a fixed factor that binds (RES 8) so that the premium of
13.2 is finite; the delay condition of POP 5.4 on top. *Test:* TEC-S2. *Status:* model
(malthus.out part A for the loop with the fixed point; the reinforcing half by hand).
A neighbour of CAP-A4 (a ground improvement absorbed by the population) and of
RES-A1, which it feeds.

**TEC-A2. The premium of a moving fixed point.** *Kind:* hazard (a transient, no loop).
While a trend or a run of learning raises `T`, the households read a ratio above `r*` by
`10 000 * TrendBP / ((10 000 + G) * g' * eta)` basis points and the population trails
`N*`; a step decays with the time constant `1 / k`; a cap pays it back monotonically.
*Delay:* `1 / k`, 16.7 years or about 6 100 ticks at 365 ticks a year, about 200 windows of the welfare window (30 ticks, `spec-population.md` 5.4). *Owner:* TEC 13.2; POP 5.
*Damper:* none needed: the loop is first order; the ratio's rise is bounded by the
highest knot of the birth schedule (10.4 warns). *Undamped:* by design (a gain is meant
to raise living standards for the time it takes the population to follow). *Condition:*
`TrendBP * dX / dR` below the highest knot minus `r*` (10.4). *Test:* TEC-S2.
*Status:* model (malthus.out part A: premium within 4% of the formula at year 40, within 9% at year 60 and 22% to 27%
below it at year 20, the premium forming over more than one time constant; e-fold at year 17 against 16.7; no overshoot after a cap).

**TEC-A3. A gain pushes a logistic stock past its maximum-yield point.** *Kind:* loop
(reinforcing below the point). *Loop:* a gain (+) the ratio at a given stock (+)
households (+) the draw (-) the stock (-) when the stock is below half its capacity the
regeneration `F(S)` falls with it (+): two negative links in the second half make it
**R**. *Delay:* years to decades after the crossing: the stock is exhausted 21, 75 and
118 years after a ramp crosses (13.3.3), and, for the two slower ramps, the ratio is at or above `r*` ten years before
(the fastest is already below it, 13.3.3), so the validator's warning is the only early
signal for those. *Owner:* TEC 10.3, 13.3; the mechanism is RES-A2 (L-number of the register), moved by
the gain. *Damper:* none at run time. At construction **the warning
`technology_past_maximum_yield`** (10.3: not a damper, a check); a cap below the
threshold; a refuge (RES-D4); a refilling stock (always stable). *Undamped:* a logistic
stock with no refuge and a cap above the threshold: extinct for good. *Condition:*
`a(5 000) * (10 000 + CapBP) < a(sig) * 10 000`. *Test:* TEC-S4. *Status:* model
(malthus.out part C: stable below `T` 1.15, a cycle at 1.20, collapse at 1.25, ramps as
above).

**TEC-A4. A gain is rest, and past the floor it is a loss for a staple.** *Kind:*
hazard, a case of L14 (the target-income rule, N-028). *Loop:* a gain (+) yield per day
(-) the days worked, because the target is met sooner (-) output (stays: interior) and,
once the work floor binds, output (+) the supply (-) the price when demand is inelastic
(+) income per household: `T^(1 - 1/e)`, falling for `e` below 1; then the failure
ladder and exit (SPN) act on the producers (13.5). *Delay:* the welfare window, the
review interval and the entry share. *Owner:* TEC 13.5; WEL 5; STB L14, L36. *Damper:*
the work floor (damper D13, `MinWorkShare`), the production halt (damper D24), the entry limit (damper D14),
the subsistence floor (damper D15). *Undamped:* the loss of income itself: a staple's demand
is inelastic (`spec-agents.md` 11.6). *Condition:* demand elasticity above 1, or a
non-staple (`spec-demand.md` 6.3). *Test:* TEC-S5. *Status:* hand (farmer.out).

**TEC-A5. Deflation from growth.** *Kind:* hazard. A gain lowers the cost anchor (5.4)
and so the belief bands; with the money stock fixed the price index of the field's
outputs falls (13.6). A detector that reads the price index alone would call it R9.
*Owner:* TEC 8.8, 13.6; STY 6.2. *Damper:* R9's second clause (the units traded do
not fall) separates it; a money flow or a policy actor lifts the price level if a pack
wants it. *Undamped:* the fall itself, which is the price of growth. *Condition:* R9
stays a clause on units (8.8). *Test:* TEC-S2 (R9 silent). *Status:* hand. A neighbour
of L70, L71 and L98 (the causes R9 names).

**TEC-A6. Contact levels the gains and leaves the table alone.** *Kind:* bias. A
region that is behind closes a share of its gap to the discounted frontier of the tick
and never passes that frontier; because a gain obtained by contact is itself a
frontier one tick later, a relay through a third region is bounded by the product of
the weights of its legs, which can exceed the weight of the direct distance (4.5.5);
a closed edge (an embargo, RSK) stops it; the yield table's site factors,
which are where comparative advantage lives (`spec-resources.md` 7.4), are untouched.
The effective yields of two regions for one recipe approach each other as far as the
gains differ, so the ratio of the two effective yields narrows toward the ratio of the
table values. *Owner:* TEC 4.5. *Damper:* **TEC-D3** (the weight and the snapshot).
*Undamped:* by design. *Condition:* none. *Test:* TEC-S3. *Status:* model
(law.out part 4).

**TEC-A7. A discovery is a correlated supply pulse.** *Kind:* amplifier (a case of L21
and L22). A step of `StepBP` reaches every household of the field in the region at the
next tick's starts, so the supply of that recipe rises together `Duration` ticks
later: a lockstep with a delay. *Owner:* TEC 4.4. *Damper:* the step is bounded by
`StepBP` and the cap; beliefs adjust in bounded steps (the damper of `spec-beliefs.md` 5.6); the
production halt (damper D24); jobs that started before the step complete at the old output
(5.1.3), which staggers the pulse by the `Duration` spread. *Undamped:* a pack with
one `Duration` and one large step. *Condition:* `StepBP` small against the belief step
times the `Duration`. *Test:* TEC-S2 with discovery on. *Status:* hand.

### 14.2 Dampers

| ID | Mechanism | Acts on | Where | Status |
|----|-----------|---------|-------|--------|
| TEC-D1 | the log law: the gain per execution falls as `1 / (X + Base)`, and executions are bounded by households | TEC-A1 | 4.2 | specified (a rule; counted by the event's `DoingAddedBP`) |
| TEC-D2 | the cap `CapBP` on the sum of the four mechanisms | TEC-A1, A3 | 4.6 | specified (a bound; counted by `ClampedBP` and a metric) |
| TEC-D3 | the contact weight and the snapshot: a catch-up never passes the discounted frontier of the tick and no order in which the cells are visited matters | TEC-A6 | 4.5 | specified (a rule) |
| TEC-D4 | the position P8.3c: no same-tick feedback, one set of gains per tick | the order bias of the rotation (WLD 10) | 6.1 | specified (a rule) |

No new damper acts on a market. TEC-D1 to D4 bound the module's own growth; the
dampers of the loops that a gain reaches (the register's dampers D13, D14, D15 and D24) are the owners'.

### 14.3 Delays near neighbours (checklist 9.6)

- The **job's `Duration`** against the step: a gain set at P8.3c reaches supply after at
  most `Duration` ticks (5.1.3), which is the recipe's own delay and not a second one.
- The **welfare window** (`spec-welfare-and-wages.md` 2.3.3): the ratio the households
  read follows a gain with half a window of delay, 15 ticks at the default, against a
  population time constant of years: 15 ticks against about 6 100 (1 : 400 at `eta` 1; 1 : 800 at
  `eta` 0.5, 12 200 ticks), no cycle.
- The **population's time constant** `1 / k`, 16.7 years at `eta` 1, against the doubling
  time of learning (years to decades, 4.2.2): they are comparable for the first
  doublings of a small `Base`, which is where TEC-A1's loop is strongest; the cap and the
  log law bound it. A pack with a `Base` below the households of its region learns a
  doubling in less than a time constant: warned in the notes of 19 (open question 8),
  not by a rule.
- The **year** against the trend: the trend is applied by a carry, one basis point
  every `TicksPerYear / TrendBP` ticks; with a calendar that has seasons the unit is
  small enough that no season aliases it.
- **Discovery's expected interval** `10 000 / DiscoveryBP` years against `1 / k`: if a
  field discovers faster than the population follows, the premium accumulates, and the
  equivalent trend is `DiscoveryBP * StepBP / 10 000` basis points a year (a rule of
  thumb, to be used with the warning of 10.4).

### 14.4 Scenarios proposed

For `spec-stability.md` 8 (numbered by the supervisor); REF builds the overlay
`overlay/technology.toml` (a field on the growers of `one` and `three`, learning only
and each other mechanism in turn).

| ID | Scenario | Pack | Reads |
|----|----------|------|-------|
| TEC-S1 | the module on with no field, and on with every rate 0, equals the module off in state, events, snapshot rows and fields and final PCG state, with and without observer and logger, the snapshot header's `ConfigHash`, `Modules` and `Names` and `WorldCreated`'s `ConfigHash` and `Names` excepted (they carry the module: FR-TEC-001). A **harness test** of the class of S77 and S78, outside the band rule of FR-STY-098 | `one`, a range of seeds | the invariants |
| TEC-S2 | learning only on `one` with the population module and a refilling stock: the gain follows `g(X)` from the executions counted; the premium and the time constant of 13.2; R9 and F4 silent; a discovery variant for TEC-A7 | `one` (with the population overlay) | `TechRow`, `RoleRow`, `PriceIndex`, `Turnover` |
| TEC-S3 | diffusion on `three`: the lagging region reaches `w` times the leader; an edge closed by `SetEdgeStatuses` stops it; the comparative-advantage facts P06 and T02 hold | `three` with the trade overlay | `TechRow`, `PriceGap` |
| TEC-S4 | a logistic stock on `one`: a cap below and above the threshold of 10.3, a step and a ramp; the stock share in the snapshot | `one` with the resources and population overlays | `EndowmentRow`, `RegionRow` |
| TEC-S5 | a staple with inelastic demand: income per household against the gain with `MinWorkShare` at its default and at 0, and the production halt on and off | `one` | `RoleRow` (`HeldByFloor`, the ratio) |
| TEC-S6 | a **harness test**: a checkpoint mid-run with a discovery key and non-zero carries restores bit for bit; a branch with `WithReseed` discovers on other ticks | `one` | the checkpoint |
| TEC-S7 | the call: a negative advance (a lost art) lowers the gain at once and the population follows it down at `1 / k`; `LostBP` when the bound cuts | `one` | `TechnologyAdjusted`, `TechRow` |

## 15. What was run

All four scripts are in `doc/specs/models/tec/` (README there); they run only in
Docker (`python:3.12-slim`, `PYTHONDONTWRITEBYTECODE=1`), with the outputs saved
beside them. They are stylised models of the **rules** of this spec (and, in
`malthus.py`, of the rules of RES and POP through `models/res/coupled.py`), not of the
library, and they show agreement with the stated arithmetic and linearisation, not that
any default is stable.

| Script | What it shows | What it does not |
|--------|---------------|------------------|
| `law.py` | the log law is exact at a doubling and within 0.083 of one between (200 000 random `X`), monotone, and far from overflow at the bounds; the trend is exact over whole years; the keyed hash hits at `DiscoveryBP / (10 000 * TicksPerYear)` over 4.4 million cell-ticks (within 1.6 standard deviations) and two keys, two cells give independent hits; the diffusion step approaches `w` times the leader from below, is independent of the order of the cells (6 permutations) and does nothing over a closed edge; the identity of the event holds and `Clamped` is never negative (100 000 random cells) | the economy: the executions are given; no price, no agent |
| `malthus.py` | the premium of a trend against `10 000 * TrendBP / ((10 000 + G) g' eta)` (within 4% at year 40, 9% at year 60 and 22% to 27% below the formula at year 20: the premium forms over more than one time constant); the e-fold of a step at 17 years against 16.7; no overshoot after a cap; the stock share of a refilling stock against `0.6 / T` and `N*` within 0.1%; a logistic stock stable and swinging below `T` 1.18, a cycle at 1.20, collapse at 1.25 and 1.40; ramps that exhaust the stock 21, 75 and 118 years after crossing | the economy: the ratio is one assumed function of the granted share, the abundance and `T`; no price, wage or target-income response; one field, one resource |
| `farmer.py` | the interior equilibrium of the target-income rule is independent of `T` and the days fall as `1 / T`; at the floor income per household changes as `T^(1 - 1/e)` (0.667 at `T` 1.5 and `e` 0.5) | a run: it is a closed form; the interior equilibrium for `e < 1` is unstable in the library (STB-L14) and is shown to say what the floor-pinned outcome is relative to |
| `worked.py` | the arithmetic of example A (three cells over five years, one step by hand) and the bounds of the products of 9.5 | anything beyond the arithmetic |

## 16. Worked examples

Invented **examples**, flagged as such. The words (`tillage`, `plain`, `hills`,
`coast`, `fabrication`, `habitat`) are data and are never branched on. Every number is
a placeholder.

### 16.1 Example A: a farming realm of three regions (an example)

One field `tillage` over the recipes `grow_staple` and `grow_pulse`. `Base` 100 000,
`DoublingBP` 200, `CapBP` 3 000, `CatchUpBP` 2 000, `Reach` 100, no trend and no
discovery. Three regions `plain`, `hills`, `coast` with distances `plain`-`hills` 20,
`plain`-`coast` 60, `hills`-`coast` 50 (both ways); the contact weights are 8 000 for 20,
5 000 for 50 and 4 000 for 60. The regions complete 900, 400 and 150 executions a tick.

**Learning and contact over five years** (models/tec/worked.out):

| After | `X` (plain, hills, coast) | learned `g` | advance `A` | gain `G` |
|-------|---------------------------|-------------|-------------|----------|
| 1 tick | 900, 400, 150 | 2, 1, 0 | 0, 0, 0 | 2, 1, 0 |
| 1 year | 328 500; 146 000; 54 750 | 421, 270, 130 | 0, 10, 5 | 421, 280, 135 |
| 2 years | 657 000; 292 000; 109 500 | 583, 393, 214 | 0, 22, 10 | 583, 415, 224 |
| 5 years | 1 642 500; 730 000; 273 750 | 827, 611, 379 | 0, 38, 10 | 827, 649, 389 |

`plain` leads and is not changed by the others. `hills` practises less (its own doing
is 611 at year 5) and receives 38 from contact; its limit is `827 * 8 000 / 10 000 =
661`, which it approaches from below. `coast` is far (`w` 4 000, limit 330) and
practises least; its advance stops at 10 because its own learning has already passed
the discounted leader (`G` 389 against 330): there is no gap to close. That is the
module's one visible oddity (4.5.5): contact never takes a region down.

**One step of P8.3c by hand, at the end of year 1.** Take the state at tick 364 as
`X` = 327 600; 145 600; 54 600 and `A` = 0; 10; 5 with every carry 0, and the
counters 900, 400, 150 (an assumption of the example: the real carries are in the
script).

1. *Fold.* `X` becomes 328 500; 146 000; 54 750. `lg12(X + Base)` is 76 327, 73 223,
   70 371 and `lg12(Base)` is 67 690, so `g` is `200 * 8 637 / 4 096 = 421`,
   `200 * 5 533 / 4 096 = 270`, `200 * 2 681 / 4 096 = 130`; the cell `hills` moved from
   269 to 270 (`DoingAddedBP` 1), the others did not move.
2. *Trend, discovery.* None (both off).
3. *Snapshot.* `G'` = 421; 280; 135 (`270 + 10`, `130 + 5`).
4. *Diffusion.* For `hills`: from `plain` `421 * 8 000 / 10 000 = 336`, from `coast`
   `135 * 5 000 / 10 000 = 67`; `F` = 336; gap `336 - 280` = 56; carry `56 * 2 000` =
   112 000 of 3 650 000: `add` 0, so the cell waits about 32 ticks for its next basis
   point (11 a year). For `coast`: from `plain` `421 * 4 000 / 10 000 = 168`, from
   `hills` `280 * 5 000 / 10 000 = 140`; `F` = 168; gap 33; carry 66 000; `add` 0.
   `plain` has no region with a higher gain: gap 0.
5. *Clamp and events.* No cell is at the cap; `TechnologyAdvanced` for the three cells
   (`Executions` 900, 400, 150; `DoingAddedBP` 0, 1, 0; `CatchUpAddedBP` 0, 0, 0).

### 16.2 Example B: a habitat with a trend, a discovery and a lost art (an example)

One region, a calendar of 400 ticks a year, one field `fabrication` with `TrendBP` 40,
`DiscoveryBP` 1 000 (one in ten years for a cell that practises), `StepBP` 300 and
`CapBP` 2 000; learning and diffusion off.

- **The trend**: `TrendCarry` rises by 40 a tick; at tick 10 it is 400 and the cell
  gets `add` 1 and the carry returns to 0: one basis point every 10 ticks, 40 a year,
  exactly. After 12 years, 480.
- **Discovery**: the probability in a tick is `1 000 / (10 000 * 400)` = 1 in 4 000;
  over a year of practice the chance of one or more is about 9.5%. A hit adds 300: the
  gain goes from, say, 600 to 900 in one tick, and the production of the next tick's
  starts is 3% larger: the pulse of TEC-A7.
- **The call**: `AdjustTechnology([{Region 0, Field 0, DeltaBP -500}])` on a cell with
  `A` 900 gives `A'` 400, `LostBP` 0; on a cell with `A` 300 it gives `A'` 0 and
  `LostBP` 200. The event is `TechnologyAdjusted` (`BeforeBP` 900, `AfterBP` 400);
  nothing else changes; the population module follows at `1 / k`.

### 16.3 Example C: the validator on a logistic stock and a trend (an example)

A woodland of shape `logistic` with a linear abundance curve and a plan share of 6 000,
and a field with `CapBP` 1 500. `a(5 000) * (10 000 + 1 500) = 5 000 * 11 500 =
57 500 000` against `a(6 000) * 10 000 = 60 000 000`: no warning. With `CapBP` 1 999:
59 995 000, none. With 2 000: 60 000 000, which is not below: the warning fires
(`technology_past_maximum_yield`). At the defaults of POP (`r*` 12 000, `dR` 400, `dX`
8 000, `knot_max` 16 000) a trend of 200 gives `12 000 + 200 * 8 000 / 400 = 16 000`,
no warning; 201 gives 16 050, and `trend_outpaces_schedule` fires.

## 17. Notes settled, and what this spec needs from the other specs

### 17.1 Notes

Every open note tagged `TEC` (there is none in the index) and every note tagged `ALL`,
read before writing (`AGENTS.md`); the others that this spec touches are listed with
what it did.

| Note | Decision |
|------|----------|
| N-016 Keep the parameter registry true (ALL) | **Applied:** ten rows (9.4: the nine fields and `Label`); no unnamed parameter was found |
| N-017 Words and names (ALL) | **Applied:** terms in 17.3 |
| N-027 No default macro constant is stable (ALL) | **Applied (stands):** every number is a placeholder; models show agreement with the arithmetic |
| N-103 Running several writers (ALL) | **Followed.** Slips are in the progress file and the handoff |
| N-126 A new event or `Warn` adds a row (ALL) | **Applied:** two events (11.1), no `Warn` in the tick |
| N-128 A hosted per-agent field is saved once (ALL, CHK) | **Applied:** no per-agent field (3.5, 12.1) |
| N-015 Placeholders (VIA, STY, CLB, EXP) | **Update line:** the defaults and bounds of 9.2 and every number of 16 |
| N-098 / N-109 The fixed factor and `r_full` (POP, VIA) | **Read:** a gain scales `r_full` and moves the fixed point (13.1) |
| N-124 A paired comparison is exact only while aligned (EXP, STB, CLB, WLD) | **Stated:** the module takes no word in a tick; one at construction (6.3.2) |
| N-134 A sweep that moves the steady state (VIA, EXP, CLB) | **Applied:** no parameter moves the tick-0 table (3.3.2, 8.7) |
| N-104 Region-keyed tables (LND) | **Applied:** the module adds none (3.3.2) |
| N-090 What a relocation must do | **Applied:** the module holds no obligation tied to a region and registers no `region_bound` refusal |
| N-091 The world slots INT must expose | **Applied:** no slot; one call (6.4, 7.1) |
| N-107, N-021 New faucets and sinks | **Applied:** none; no ledger reason, no flow (position 12) |
| N-045 Dampers that act silently (STB) | **Applied:** the cap is counted (4.6.2) |
| N-125 A snapshot is a function of saved state (CHK) | **Applied:** `TechRow` (11.4) |
| N-131 Not in version 1 (INT, FIS, PLC) | **Respected** (7.1.6) |
| N-054 The closed economy fixes the mean ratio at 1 | **Read:** in a closed economy a gain moves the ratio only through the way 13.5 describes; no new claim |
| N-028 Target-income rule (WEL, STB) | **Update line:** a gain is rest first (13.5), TEC-A4 |
| N-110 Version 1 rumours prices only | **Respected:** contact does not use rumours (4.5.6) |

**New notes to add** (area codes; the supervisor numbers them):

1. **A gain is not a recipe** (areas PRO, CAP): an unlock (a recipe or a facility class
   that appears at a level) needs a refusal reason in PRO's order and a `Needs`-like
   clause on a facility class; not asked (open question 3).
2. **R9's volume clause is on units** (areas STY, IND): growth deflation must stay
   silent (8.8).
3. **A module that keeps a key re-derives it at a reseed** (areas CHK): the rule of 4.4.6
   is the first; a later module with a keyed hash does the same.
4. **The stability check is made with the module off** (areas STY, EXP): 8.7.

### 17.2 What this spec needs from the other specs

Every edit is owed and is **not made here**. The full table with exact text is in
`.supervisor/tec.handoff.md`; in brief:

| File | Where | The change |
|------|-------|-----------|
| `spec-production.md` | 6.3, FR-PRO-022, 6.7, 10.3 | add the term `E = E * tech / 10 000` between the shock and the trainee factor; the row "A recipe that changes over time" answered by this spec; the overflow check reads `MaxTechBP` |
| `spec-world.md` | 5.1, 5.7.2, 5.7.3, 6.4 (P2.1, P2.7), 6.10, 6.11, 7.3, 7.4.2, 8.2 | the sub-step 8c of construction (5.1.2: the key, draw D16); the carve-out of a module-level word in 5.7.2 and the count of 5.7.3 (6.3.4); the counter clauses; P8.3c; the order of the events; the row D16; the call as a row of 8.2 |
| `spec-integration.md` | 5.1, 5.3, 7.1, 11.2 | the call `AdjustTechnology`; not listed by `Slots()`; `ErrUnknownTechField`; the module name |
| `spec-observability.md` | 3.2, 3.3.9, 4.1, 5.3, 6.2, 6.4.2, 6.5, 7.2, 7.5, 7.6 | the events, the names table, `TechRow`, the metrics and the attribute, the seeding row of the gain gauge, the keys, the `Info` summary's `fields`; 5.3.1 and 4.1.1: a note that the header fields `ConfigHash`, `Modules`, `Names` and the event fields `ConfigHash`, `Names` are what a module present and inert changes (FR-TEC-001, FR-CAP-001) |
| `spec-checkpoint.md` | 4.1, 4.2, 4.3, 6.2, 6.8, 9.3 | the module section; the restore checks; the reseed rule for the key; the format version |
| `spec-config.md` | 3.2, 3.7, 4.1, 6.2, 7.3, 7.4, 8.1, 8.3 | the module, the fields, the bounds (and `Label` at `MaxLabelBytes`), the errors and warnings, the hash, the allow-list of presentation-only fields (`TechField.Label`) |
| `spec-resources.md`, `spec-population.md` | 8.3.1, 10; 5.4, 9.2 | informational pointers to 10.3 and 13 |
| `spec-agents.md`, `spec-employment.md` | 6.3; 6.1, FR-EMP-028 | the wording of the effective yield; `tech` in the hired job's factor |
| `spec-infrastructure.md` | FR-CAP-001, CAP-S1 and the lead (line 15) | the same defect as FR-TEC-001 had: "indistinguishable from the module absent" omits the header fields; reword as FR-TEC-001 (the same state, events, snapshot rows and fields other than `ConfigHash`, `Modules` and `Names`, and the same final random state) |
| `spec-land.md`, `spec-saving-and-investment.md`, `spec-polity.md` | 7; 5.3.1 and 1; 5.4.2 and 1 | pointers only, no rule: LND's rent per drawn unit rises with `E` (8.10); SAV's `net_r` includes `tech`; POL's jurisdiction carries no lever on a gain (a consumer's call, 7.1.5) |
| `spec-viability.md`, `spec-stylised-facts.md`, `spec-experiments.md`, `spec-calibration.md`, `spec-stability.md`, `spec-reference-chain.md`, `spec-indicators.md` | 11; 6.2, 7.3; 7.3; 4.3, 6; 4, 5, 8, 9.10; 7, 8; 7 | the path flag `no`; R9's units clause and the growth scenario; the note on alignment; the registry rows; the entries, dampers and scenarios; the overlay; one preset ask |
| `doc/rules/observability-and-logging.md`, `doc/rules/architecture.md`, `AGENTS.md` | 3.3.2, 4.3, 5.3, 5.4.1; 2.6, 3.3; Layout | the keys, the events, the metrics and the attribute; `internal/technology/` and `technology.go`, `TechFieldID`; a fresh read-receipt tag on each rule file |
| `doc/specs/README.md`, `doc/roadmap.md`, `doc/documentation_index.md`, `doc/notes.md` | index, code table, owner table, glossary, decisions; TEC row; rows; updates, the new notes, the index, a fresh tag | the usual |

### 17.3 Terms (N-017)

**technology** (this module; not the technology matrix of VIA), **field** (a set of
recipes that share what is learned; "field" is also a Go struct field, the term is
always "technology field" when there is doubt), **cell** (a region and a field),
**experience** (the cumulative executions of a cell), **learned gain**, **advance** (the
sum of trend, discoveries, contact and the call), **gain**, **technology factor**
(`tech`), **doubling** (of the cumulative work), **base** (executions already behind a
field), **trend**, **discovery** (a step with a chance), **key** (of the module),
**contact**, **contact weight**, **reach**, **frontier**, **gap**, **catch-up**, **cap**
(of a field), **practises** (a cell that completed an execution in the tick), **premium**
(the ratio above the replacement ratio while a gain is tracked). "Experience", "gain",
"cap" and "advance" repeat words of other specs (`experience` of nothing else,
`gain` of CAP's layer and MIG's `MoveGain`) and are qualified where they meet.

## 18. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs in a
`Requirement:` line (testing.md 4.2). The level of each test is in the handoff.

### Functional

| ID | Requirement |
|----|-------------|
| FR-TEC-001 | With `Modules.Technology` present with no field, or with every mechanism of every field off, the world MUST have the same state, the same events and the same snapshot rows and fields as the world with the module absent, and the same final state of the random source, with and without an observer and a logger, **except** the fields that carry the module itself: `ConfigHash`, `Modules` and `Names` of the snapshot header (`spec-observability.md` 5.3.1) and `ConfigHash` and `Names` of `WorldCreated` and `WorldRestored` (TEC-S1). |
| FR-TEC-002 | A field MUST be rejected with `ErrTechShape` when `Recipes` is empty, `CapBP` is 0, a field that an off mechanism does not use is non-zero, or a field that an on mechanism requires is 0 (3.1.2). |
| FR-TEC-003 | A recipe in two fields MUST be rejected with `ErrTechRecipeOverlap`, an unknown recipe with `ErrUnknownRecipe`, a duplicate field ID with `ErrDuplicateID`, a value outside its bound with `ErrOutOfBounds` and more than `MaxTechFields` fields or `MaxFieldRecipes` recipes with `ErrTooMany`. |
| FR-TEC-004 | A cell MUST exist for each ordinary region and each field, in ascending (`RegionID`, `TechFieldID`) order, with `Experience`, `Advance` and `CatchUp` 0 at construction, so that the gain is 0 and `tech` is 10 000 for every recipe at tick 0 (3.2, 3.3.2). |
| FR-TEC-005 | The gain of a cell MUST be `min(CapBP, g(X) + A)`, `A` MUST be stored in `[0, CapBP]`, and the gain MUST NOT fall in a run in which no call of 7.1 is made (3.3). |
| FR-TEC-006 | The learned gain MUST be `DoublingBP * (lg12(X + Base) - lg12(Base)) / 4096` truncating toward zero, with `lg12` as in 4.2.1, and MUST equal `j * DoublingBP` when `X + Base = 2^j * Base`. |
| FR-TEC-007 | Experience MUST increase only by the `Executions` of jobs of recipes of the field that completed in the region of the job, trainee, hired and own-use executions included and abandoned jobs excluded, folded at P8.3c from the counters of P2.1 and P2.7, and MUST NOT depend on the weight of the executor (3.4). |
| FR-TEC-008 | The trend MUST add to every cell of the field, by the carried remainder `TrendCarry` over `TicksPerYear`, exactly `TrendBP` basis points of advance over any whole number of years (4.3). |
| FR-TEC-009 | A practising cell of a field with `DiscoveryBP` above 0 MUST discover when the high word of `K(t, c) * (10 000 * TicksPerYear)` is below `DiscoveryBP`, with `K` as in 4.4.2, and a discovery MUST add `StepBP` to `Advance` (4.4). |
| FR-TEC-010 | Every position of a tick MUST read the same gain for a recipe in a region: the gain MUST change only at P8.3c and at a call of 7.1 (5.3, 6.1). |
| FR-TEC-011 | The module MUST take no random word in a tick; it MUST take exactly one word at construction, at step 8c of construction (`spec-world.md` 5.1), after the creation draws of the initial agents (draw D1) and before the audit of step 9, if and only if some field has `DiscoveryBP` above 0; and the creation draws of the agents MUST be the same with the module on or off (6.3). |
| FR-TEC-012 | The key MUST be saved with the module's state, and a restore with `WithReseed(salt)` MUST replace it with `finalise(Key xor finalise(salt))` (4.4.6). |
| FR-TEC-013 | Diffusion MUST compute every frontier from the snapshot of 4.6.1 (after the fold, the trend and the discoveries) before any catch-up is applied, and MUST NOT depend on the order in which the cells are visited (4.5.3). |
| FR-TEC-014 | The contact weight from a region `s` to a region `r` MUST be `(Reach - d) * 10 000 / Reach` for `1 <= d < Reach`, truncating, and 0 for `d >= Reach` or when there is no path, with `d` the shortest distance from `s` to `r` over the effective graph (4.5.1). |
| FR-TEC-015 | A catch-up MUST add `floor((CatchUp + gap * CatchUpBP) / (10 000 * TicksPerYear))` to `Advance` and keep the remainder in `CatchUp`, and a cell with no positive gap MUST add nothing and keep its carry (4.5.4). |
| FR-TEC-016 | A cell that has no experience, trend or discovery MUST NOT, by catch-up alone, reach a gain above the largest `w * G'` of the other cells of its field (4.5.5). |
| FR-TEC-017 | The sum of the four mechanisms MUST be capped at `CapBP`, and the event MUST report the part cut as `ClampedBP` such that `AfterBP = BeforeBP + DoingAddedBP + TrendAddedBP + DiscoveryAddedBP + CatchUpAddedBP - ClampedBP` with `ClampedBP` not negative (4.6.3). |
| FR-TEC-018 | The effective factor of a job MUST include `tech` between the shock and the trainee (or hired) factor, each division truncating toward zero, `tech` MUST be 10 000 for a recipe in no field, and `E` MUST be 0 when the site factor is 0 whatever the gain (5.1). |
| FR-TEC-019 | The output of a job MUST be fixed at its start with the gain at the start, and a gain set after the start MUST NOT change a running job (5.1.3). |
| FR-TEC-020 | The overflow check of `spec-production.md` 10.3 MUST read `MaxTechBP` for a recipe in a field, and a configuration in which a job's output could exceed `MaxQuantity` MUST be rejected with `ErrOverflow` (5.1.4). |
| FR-TEC-021 | `AdjustTechnology` MUST be atomic, MUST set `A' = min(CapBP, max(0, A + DeltaBP))` at the call, MUST leave the experience and the carries unchanged, MUST emit `TechnologyAdjusted` for each entry whose `A` changed in ascending (`RegionID`, `TechFieldID`) order, and MUST move no money and no goods (7.1). |
| FR-TEC-022 | `AdjustTechnology` MUST return `ErrInStep` and `ErrWorldFailed` first, as every call does, then `ErrModuleOff`, then for each entry in the order given `ErrUnknownRegion`, `ErrOutsideNode`, `ErrUnknownTechField`, `ErrOutOfBounds`, `ErrDuplicateID`, the argument errors MUST be typed as `spec-integration.md` 3.2.2 says, and it MUST change nothing on an error (7.1.3). |
| FR-TEC-023 | `TechnologyAdjusted` MUST carry the tick of the next step, the asked delta, the gain before and after and the part lost, and MUST NOT be listed by `Slots()` (6.4, 11.1). |
| FR-TEC-024 | `TechnologyAdvanced` MUST be emitted at P8.3c, in ascending (`RegionID`, `TechFieldID`) order, for each cell in which a counter, a trend, a discovery, a catch-up or a cut occurred, and MUST NOT be emitted for a cell with none of them, nor when the module is off (11.1). |
| FR-TEC-025 | The names table of `WorldCreated` and `WorldRestored` MUST carry one string per `TechFieldID`, and every value of the metric attribute `economysim.field` MUST be a field of the configuration (11.3). |
| FR-TEC-026 | The snapshot MUST hold a `TechRow` for each cell with `Experience` or `Advance` above 0, as a function of the saved state, and the snapshot of a restored world MUST equal the original's at the same boundary (11.4). |
| FR-TEC-027 | The checkpoint MUST contain exactly the state of 12.1, no per-agent field and no counter, and a restore MUST validate it as 12.2 says and return `ErrCheckpointCorrupt` for each failure. |
| FR-TEC-028 | A world restored from a checkpoint MUST continue exactly as the uninterrupted world: the same gains, the same discoveries on the same ticks and the same carries (12.2). |
| FR-TEC-029 | The warnings of 10.2 MUST be raised at construction with `rule` and `path`, MUST NOT fail validation, and the informational `field_inert` MUST NOT be raised for a field with a mechanism on. |
| FR-TEC-030 | `technology_past_maximum_yield` MUST fire if and only if the population module is on, a recipe of the field draws on a `logistic` resource whose abundance curve has a positive slope at the plan share, and `a(5 000) * (10 000 + CapBP) >= a(sig) * 10 000` (10.3). |
| FR-TEC-031 | `trend_outpaces_schedule` MUST fire if and only if the population module is on and `TrendBP * dX > (knot_max - r*) * dR` (10.4). |
| FR-TEC-032 | The module MUST create or destroy no money and no goods, add no ledger reason, no account and no money flow, and the audit of `WithAudit` MUST hold over a run with every mechanism on (position 12). |
| FR-TEC-033 | The step of P8.3c MUST be a function of the counters, the saved state, the graph tables, the key and the tick only: the same inputs MUST give the same result whatever the prices, the stocks, the observers and the logger (4.6.4). |
| FR-TEC-034 | No parameter of the module MUST change the yield table at tick 0, the warm start or the result of the viability check (3.3.2). |
| FR-TEC-035 | `Experience` MUST saturate at `MaxExperience` and MUST NOT wrap (4.2.4). |
| FR-TEC-036 | An outside node MUST have no cell, and contact MUST stop across an edge that a status has closed and resume when it is opened (4.5.1). |
| FR-TEC-037 | The module MUST emit no `Info` or `Warn` inside the tick, and its `Debug` lines MUST be guarded by `Enabled` (11.5). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-TEC-001 | The gain, the discoveries and the carries MUST be bit-identical across runs and platforms for a seed, a configuration and a sequence of calls, with the log law, the hash and the contact weight in the golden output (architecture.md 7.4). |
| NFR-TEC-002 | No step of P8.3c MUST depend on the order in which cells are visited: the advance added by a diffusion step MUST be the same for every order of the cells (4.5.3). The cell index of the discovery hash is a function of the canonical order of the pack and is outside this requirement (4.4.2). |
| NFR-TEC-003 | With the default logger and observer, P8.3c and the counter increment at a completion MUST allocate nothing (`testing.AllocsPerRun`). |
| NFR-TEC-004 | The cost of P8.3c MUST be `O(R * F)` plus `O(F * R^2)` for fields with diffusion on and `O(practising cells)` for discovery, and its state MUST be sized by regions times fields, with no table sized by regions squared (4.5.1, 6.6). |
| NFR-TEC-005 | Every product of 9.5 MUST be computed in `int64` or the checked 128-bit helper and MUST NOT overflow for any value inside the bounds. |

## 19. Open questions

1. **Per-agent experience.** Position 1 and 3.5 reject it. Reopen if SKL (or a consumer)
   wants a household's own practice to count: it would be a hosted agent field (N-128)
   with its own rule for a weight change, and the cell would then be the sum.
2. **Carriers of knowledge.** Contact is a distance (4.5.6). A migrant's arrival
   (`AgentRelocated`, cause `migration`, P1.5), a trader's trip (`spec-merchants.md`)
   or a rumour could add to the destination cell the origin's gain times a share:
   the hook is the arrival position, the state is none (the transfer is a
   catch-up with a one-off weight), and the cost is that diffusion would then depend
   on modules that may be off. The shape stays open until a consumer's pack has a
   region with no road and many migrants.
3. **Unlocks.** A recipe or a facility class that exists only above a level: a clause on
   the recipe (`Unlocked`: a field and a gain) and a refusal reason `not_unlocked`
   already used by CAP for a settlement flag (`spec-infrastructure.md` 4.2.2). Not
   asked: `spec-production.md` 6.7 keeps recipes fixed for a run.
4. **Forgetting.** A gain that decays when a field is not practised (a lost art by
   neglect, an obsolescence). Version 1 has the call only (7.1). A decay would be a
   rate on `A`, a carry, and a second source of a falling gain: it would change
   FR-TEC-005.
5. **Partial credit for an abandoned job.** Nothing completed, so nothing counts
   (3.4.2). A share of the executions of a job abandoned late (a removal, a role
   change) would be a lesson learned; it has no data behind it.
6. **A scale effect in discovery.** The chance of a practising cell does not grow with
   its households (4.4.4), so a large region does not discover faster than a small
   one. A chance proportional to the cell's executions would give the Kremer loop
   (more people, more ideas) a second positive link; it is the first thing to add if
   a setting needs it, with a register entry and a cap.
7. **A settlement class that gates learning or discovery.** One string on `TechField`
   and one reader in SET's table (8.6).
8. **A warning for a small `Base`.** When a doubling takes less than a population time
   constant (14.3), TEC-A1's loop is strongest. The check needs the executions a tick
   of the pack (the census and the recipes' durations): VIA could report it; no rule is
   written here.
9. **A compounded law.** A geometric gain per doubling (4.2.3) is the other usual
   form. It changes FR-TEC-006 and the bound of 9.5 and is a second `Law` field.
10. **Technology from trade.** An outside node as a frontier (the import of a technique)
    would make EXT a source of gain: `spec-external-sector.md` has no state for it and
    this spec reads no outside quote (N-112).
11. **A cap per region.** The cap is the field's (a pack with a rich region and a poor
    one has one cap). A row keyed by region would be a region-keyed table under the one
    override mechanism (N-104, `spec-regions.md` 5.1).
12. **The productivity preset.** IND's `Yield` (8.9): the executions-weighted mean of
    `EffectiveYield`, by region and recipe. A consumer builds it today from the events.

## 20. Where this lives in the code

| Part | Home | Why |
|------|------|-----|
| the types `Technology`, `TechField`, `TechFieldID`, `TechnologyAdjustment`, the call `AdjustTechnology`, the events | `technology.go` of the root package (one file per concept, architecture.md 2.6) | public API |
| the log law `lg12` and the learned gain, the contact weight and the frontier, the catch-up and the trend carries, the keyed hit, over primitive slices | `internal/technology/` (stdlib only; imports `internal/ordering` for the finaliser; never the root, architecture.md 2.9) | pure rules, unit-testable on their own (the model scripts are the first test tables) |
| the module's fields in the loader | `config/` (a `[modules.technology]` table, strict decoding like the others) | the only package that may import a parser |
| the state in the checkpoint | the checkpoint section of 12.1, written by `checkpoint.go` over `internal/codec` | one canonical encoding |
| the fixture | `testdata/checkpoints/` (a world with a discovery key and non-zero carries), `testdata/chains/overlay/technology.toml` (REF) | the testing rules |

No package for constants (architecture.md 2.4); the bounds are in `bounds.go`; the rule
constants are beside the code of `internal/technology`. The module imports nothing
that the root does not already import. The root converts at the call into
`internal/technology` inline (architecture.md 2.9): the executions counters and the
distance table go in as slices of `int64`, and the gains and carries come back.
