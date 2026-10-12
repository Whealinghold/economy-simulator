# Spec: Population: births, deaths, the Malthusian fixed point and regional collapse

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-POP-NNN` and
`NFR-POP-NNN`, which tests cite (testing.md section 4). They are in section 15.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
version 1 (`vision.md` section 4). Off unless configured: the zero configuration has
a fixed number of households, changed only by the failure ladder and the consumer.
Needs `spec-agents.md` (the agent, `Unfed`, the weight change 3.7, the creation of
agents 3.5), `spec-welfare-and-wages.md` (the welfare ratio 2.1, 2.2, the effective
window 2.3.3), `spec-estates.md` (the pool, 2.3, 2.8), `spec-spawner.md` (the
operations and the funding, 3.3, 4, 7.3, and the hook of 8), `spec-scale.md` (the
weight, 4.6, 4.8), `spec-ledger.md` (the reason registry 7.2) and `spec-calendar.md`
(`TicksPerYear`, 3.2).

**Every number in sections 8, 12 and 13 is a placeholder.** Nothing has been run
against the library, bazaarBot has not been run, and no acceptance band exists
(notes N-015, N-027). Section 13 describes stylised Docker Python models of the rules
of this spec, not runs of code and not a claim that any default is stable. No default
is set from a source: the source read for this spec (Clark, section 16) gives shapes
and no usable rates.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer, never a household (N-001, N-017). "Year" is
`TicksPerYear` ticks (`spec-calendar.md` 3.2).

## 1. Purpose and scope

The spawner (`spec-spawner.md`) owns who leaves through the failure ladder and who
changes occupation. It leaves one thing to this spec: how many households are
**born** and how many **die** for reasons other than the ladder, as a function of
welfare, so that a stable population exists (roadmap POP row; the Malthusian fixed
point). This spec covers:

- what a birth and a death are, and why there is no age structure (3.2, 3.3);
- the signal, the rate schedules and the exposed households (3.4 to 3.7);
- how a rate becomes a whole number of households, exactly and without a random
  draw (3.8);
- how births and deaths become weight changes: the hook of `spec-spawner.md` 8, the
  funding, the floors and the ledger reasons `births` and `deaths` (4);
- the interaction with the failure ladder and with `Renewal`, so that no
  household is counted twice (4.5, 4.6);
- the Malthusian fixed point: what makes it, when it is stable, and what this library
  does and does not provide for it (5);
- regional collapse, recovery, the floors and the closed region (6).

Out of scope: where a household goes when it leaves a region (`spec-migration.md`,
MIG); the failure ladder, the entry rule and the occupation choice
(`spec-spawner.md`); what makes the welfare ratio fall as households rise, which is a
fixed factor of production such as an endowment (`spec-resources.md`, RES, written; 5.2, 8 there); shocks that change a death schedule (`spec-integration.md`, INT; 6.6);
the tick phases (`spec-world.md`); the bands that say what a healthy population curve
is (`spec-stylised-facts.md`).

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is it? | An optional module, on when `Config.Modules.Population` is non-nil. It adds two rates, a birth rate and a death rate, each a function of a household's own welfare ratio | ADR 0001 decision 2. The zero configuration is the tested baseline |
| 2 | What is a birth, what is a death? | The formation and the dissolution of a household. A birth adds a household to the economy in the role of the households it grows from; a death dissolves a household and passes its holdings to the estate. The names are the ledger's (`births`, `deaths`, N-083) | The unit of the model is the household (`spec-currency.md` 5.5). People inside a household are not modelled |
| 3 | Age structure? | None. A household has no age. Rates are per household per year | The simplest model that gives a fixed point: the three assumptions of the Malthusian model need rates and a link from population to income, and nothing else (3.3, 16). Cohorts of identical households (`spec-scale.md`) cannot carry an age without splitting. The cost is the loss of generation lags and echo cycles (3.3) |
| 4 | The signal | The household's **own** welfare ratio (`spec-welfare-and-wages.md` 2.1, 2.2) | The rule is local. It reads no regional aggregate, so "macro is measured, never set" holds (roadmap; STB 5.1.3). The mean ratio is only measured |
| 5 | The rates | Two **schedules**, each a piecewise-linear list of knots of (ratio, annual rate in basis points), flat outside the knots, as data. Region scales shift them. The default birth schedule is flat and the default death schedule falls with the ratio | Clark's evidence supports a death rate that falls with income across households and a birth rate that hardly responds (16); the schedule is data so a setting can differ. The heights are placeholders |
| 6 | Who is exposed? | Only the **fed** households of non-external agents (`Weight - Unfed`). Unfed households are under the failure ladder and nothing else | One household is never counted by two rules. The ladder reduces `min(Unfed, ...)` (`spec-spawner.md` 5.3), so the two sets are disjoint by construction (4.5) |
| 7 | Rate to whole households | The exact carried remainder of `spec-scale.md` 4.6 with the denominator `10 000 * TicksPerYear`: per agent for deaths, per (region, role) for births. No random draw. The start of an agent's carry is a fixed function of its ID | Exact in expectation, deterministic, cheap, and it spreads deaths in time without a draw. A per-agent carry is needed because poorest-first allocation of a role count is fatal to pool-funded births (5.5, 13.2) |
| 8 | The hook | Per role a number of births (`PopBirths`) and per agent a number of deaths (`PopDeaths`), both at least 0. This replaces the single signed `PopDelta` per role of `spec-spawner.md` 3.2 and 8.2 | A signed net hides the gross flows and the money they move (4.2); an allocation by privation and poverty kills the poorest and starves the pool. Contract conflict C1 in the handoff |
| 9 | Funding of births | From the regional pool, in whole households, within the spawner's budget `EntryPoolShare` (`spec-spawner.md` 7.3). Each household receives a **stake**, `Role.StartMoney * BirthStake / 10 000` | Money is never created (N-073). A closed region's population is then bounded by its money, `M / stake` (5.5): a finding, not a design wish, so the stake is a parameter |
| 10 | `Renewal` | Ignored while the module is on (effective 0). A configured non-default `Renewal` above 0 is warned (`renewal_with_population`); the default is only noted (4.6.1) | A refill of every ladder removal would cancel the positive check. Births are the only replacement (4.6) |
| 11 | The fixed point | A replacement ratio `r*` where births equal deaths. It is stable if net growth rises with the ratio and the ratio falls with households. POP owns the first, RES the second. Without a fixed factor nothing bounds growth except the pool's money and `HouseholdCap` | Clark's three assumptions (16). The library has no fixed factor until RES exists (5.2) |
| 12 | Collapse | No state, no regime flag. Collapse is a fall in households that the measures show; recovery is births funded by the pool the dead filled. Floors: `MinHouseholds` stops the schedule's deaths from emptying a role; the ladder alone can end a role | Macro is measured, never set. A smooth schedule must not extinguish a small group by arithmetic (6.4) |
| 13 | Closed region | There is no immigrant. A region's households change by births, deaths and the ladder only, until the migration module (`spec-migration.md`) is on | Stated so the closed behaviour is testable (6.5) |

## 3. The module and the rates

### 3.1 The module

`Config.Modules.Population` is a pointer: nil is off, non-nil is on (`spec-config.md`
3.2). Its fields are in section 8: two schedules, `MinHouseholds`, `BirthStake` and a
table of per-region settings. The module needs no other module. It reads the welfare
ratio, which every agent keeps (`spec-agents.md` 3.1), whether or not the welfare
rules (the target-income rule) are used.

3.1.1. With the module off nothing here runs, no event of section 10 is emitted, no
state of section 11 exists, and `Renewal` keeps its meaning (`spec-spawner.md` 7.1).

3.1.2. With the module on, the households change by births (here), deaths (here),
ladder removals and reductions (`spec-spawner.md` 5), entry of the spawner that is not
renewal (none in the default spawner), and the consumer's own calls. Nothing else
creates or removes a household.

### 3.2 Births and deaths are the formation and the dissolution of households

A **birth** adds one household to a role in a region. It is an entry in the sense of
`spec-spawner.md` 4.3: it is placed in the agent of the role with the smallest weight
below the group's `Scale`, else in a new agent of weight 1, it brings money from the
pool (4.3) and no goods, and it has the newcomer's state of `spec-spawner.md` 7.4
(beliefs from the price window mean or the market price, an empty welfare record padded
by `DefaultRatio`, `Hunger` and `Privation` 0).

A **death** removes one household from an agent: the weight falls by 1, the money and
every good of the agent scale to `floor(x * w' / w)` and the difference goes to the
estate (`spec-scale.md` 4.8; ledger reason `deaths`, 4.7). When the weight reaches 0 it
is the removal of `spec-agents.md` 3.6 with the event `AgentBankrupted` and `Cause`
`population`.

Households are not conserved, money and goods are. A birth is not a transfer from
anywhere: it is a new unit that is funded from the pool. A death is not a loss: the
holdings go to the estate. The module creates and destroys no money and no goods
(FR-POP-022).

### 3.3 No age structure

A household has no age, so a death is not the end of an age and a birth is not the
birth of a person. The rate is a probability per household per year, applied in
expectation by the carry of 3.8.

Why this is enough. The Malthusian model of Clark (16) is three assumptions: a birth rate
that rises, or is flat, with living standards; a death rate that falls with them; and
living standards that fall with population. They give the fixed point, its location
(`r*`, fixed by the two schedules alone) and its stability (5.4). None needs an age.
Why it is the right size here. The agent is a household or a cohort of identical
households (`spec-scale.md` 2): one record cannot carry an age distribution without
splitting into cohorts by age, which multiplies the agents. The first consumer needs no
weight above 1 (N-030), but other consumers may, and the library does not add a field
and a rule that only some of them need.

What is lost, stated so nobody expects it:

- **No generation lag.** Real populations respond to income through people who must
  grow up. Here a household forms at once. The only delays in the loop are the welfare
  window and the economy's own lags (5.4). A population cycle of a generation's length
  (the echo of a past surge) does not arise.
- **No dependency.** A household that is a child, an elder or a worker is not
  distinguished. A birth does not reduce the labour of its parents.
- **No age at death.** Deaths come from a schedule on the ratio, not from old age. A
  household dies at about `1 / d` years after its carry's start, in expectation, and
  the starting carry is spread by the ID (3.8.3), so deaths are spread in time.

If a consumer needs a delay, the extension point is a maturity pipeline for births
(open question 1). The condition under which it matters is in 5.4: a delay turns the
approach to the fixed point into a cycle when gain times delay is about 1.

### 3.4 The signal: the household's own welfare ratio

For an agent `a` the signal is its welfare ratio in basis points, as
`spec-welfare-and-wages.md` 2.1 defines it over the agent's effective window and 2.2
pads it for a new agent. For a cohort it is the per-household ratio, which does not
depend on the weight (`spec-scale.md` 4.3). It is read after the consumption phase, so
it includes the tick just ended.

3.4.1. An agent whose ratio is absent (no upkeep, `spec-welfare-and-wages.md` 2.2.3) is
not exposed. Its households are counted as `NoRatio` (10.1). A world with no upkeep is
legal and warned (`no_upkeep`, `spec-agents.md` 13); in it the module is inert.

3.4.2. The ratio can be negative (a window of net losses). A negative ratio is below the
first knot of every schedule and takes the first knot's rate.

3.4.3. **Why the household's own ratio and not the region's mean.** A rule that read the
mean would be a controller over an aggregate. The household reads what the household
has: its own income against its own cost over its own window. The mean is a measure
(5.3, 10.4) and nothing acts on it. This is the same line `spec-estates.md` 2.6 and
`spec-spawner.md` 6.1 draw.

### 3.5 The schedules

A **schedule** is a list of **knots** `(Ratio, Rate)`, `Ratio` in basis points of the
welfare ratio and `Rate` in basis points per year (10 000 is every household per year).
Two schedules exist: `BirthSchedule` and `DeathSchedule`.

3.5.1. **Evaluation.** For a ratio `x`: below the first knot the rate of the first knot;
above the last, the rate of the last; between knots `i` and `i+1`, with
`dx = x_{i+1} - x_i` (positive),

```
rate = y_i + trunc( (y_{i+1} - y_i) * (x - x_i) / dx )
```

Integer, truncation toward zero (architecture.md 3.5; the rule is stated here because a
change of it is a behaviour change), products with the checked 128-bit helper. A
schedule of one knot is a constant.

3.5.2. **Why piecewise linear and not a formula.** A formula would fix a shape (a
power, an exponential) for every setting. A list of knots is data: a flat line, a ramp,
a curve with a knee, all in the same field, and the validator can read the whole shape
(9). Two knots make the ramp of the defaults.

3.5.3. **What the defaults say.** The default birth schedule is flat. The default death
schedule falls from a high rate at low ratios to a low rate at high ratios. They cross
at the **replacement ratio** `r*` (5.7), 12 000 by default. The shapes follow what Clark
found (a death rate that falls strongly with income across households, a birth rate
that hardly rises, 16); the heights are placeholders and are not the rates of any
society.

3.5.4. **Direction.** The Malthusian assumptions are a birth rate that does not fall and
a death rate that does not rise as the ratio rises. A schedule that breaks them is
legal (a setting may model a demographic transition) and is warned (9.2,
`schedule_direction`), because the fixed point is then not guaranteed.

### 3.6 Region scales

A region may carry two scales, `BirthScale` and `DeathScale`, in basis points (10 000
= the schedule as written). The effective annual rate of an agent is

```
rate_eff = min( 10 000, trunc( schedule(ratio) * scale / 10 000 ) )
```

The scale is the place where a region differs (a disease environment, a custom, a
climate) without a second schedule, and where a shock acts: Clark's point that a higher
death rate at a given income is a shift of the whole schedule (16) is a change of
`DeathScale`. A shock sets and clears the scale between ticks; its mechanism is INT's
(6.6). The clamp at 10 000 (every household per year) is counted
(`RateClamped`, 10.1).

### 3.7 The exposed households

For an agent `a` of weight `w` and `Unfed` households (`spec-agents.md` 5.4), the
**exposed households** are

```
H_a = w - Unfed          when the agent is not external, its ratio exists,
                         and (for births) its role is not closed
                         in the region (`Closed`, or a `Needs` its class lacks)
H_a = 0                  otherwise
```

3.7.1. **Unfed households are not exposed.** They are under the failure ladder
(`spec-spawner.md` 5). A household that cannot eat neither forms a household nor dies
by the schedule. This is what keeps one household from being counted twice (4.5). The
cost: in a severe shortage the schedule is silent for the unfed and the only check is
the ladder, which is capped by `ExitShare` and `DeathStep` (`spec-spawner.md` 5.3).
That is the intended grading, and the measure `Unfed` in `PopulationChanged` (10.1)
shows how many households the schedule did not see.

3.7.2. External agents are not exposed, for births or for deaths (`spec-spawner.md` 9).
A consumer who wants a mortal player or a player who has children writes its own
spawner.

3.7.3. **Births and a closed role.** A role marked `Closed`, or whose `Needs` flag the
region's class lacks (`spec-settlements.md` 6.4.1), receives no entrant from the default
spawner (`spec-spawner.md` 6.2, FR-SPN-041, FR-SPN-057). Its households are not exposed for
births, so the birth carry of the role does not advance. Its households are exposed for
deaths. A role that is open but has a ceiling (`spec-settlements.md` 6.4.2) or a body's places
(`spec-institutions.md` 3.4.3) is exposed, and its births are capped at its headroom (4.4.4).

### 3.8 From a rate to whole households: the carried remainder

The expected number of events in a tick for `H` exposed households at an annual rate of
`R` basis points is `H * R / (10 000 * TicksPerYear)`. The module keeps that number
exactly as an integer numerator over the denominator

```
D = 10 000 * TicksPerYear          (an integer; TicksPerYear is derived, spec-calendar.md 3.2)
```

3.8.1. **Deaths, per agent.** In the computation of a tick (4.1), for each agent `a` in
ID order with `H_a > 0`:

```
num_a      = H_a * d_a                        d_a the effective death rate of 3.6
carry_a    = carry_a + num_a
n_a        = carry_a / D        (integer division)
carry_a    = carry_a mod D
n_a        = min(n_a, H_a)                    a cut is counted (DeathsCut)
```

`carry_a` is the agent's **death carry**, state of the agent (11). An agent with `H_a` 0
keeps its carry.

3.8.2. **Births, per region and role.** In role order, the numerator is the sum over the
non-external agents of the role, in ID order, of `H_a * b_a` (the effective birth rate
of 3.6), and

```
carry_r  = carry_r + sum
births_r = carry_r / D;   carry_r = carry_r mod D
```

`carry_r` is the **birth carry** of the (region, role), state of the module (11). It is
one number per (region, role), so no agent is the "parent" of a birth, and the new
household is placed by the spawner's rule (3.2).

3.8.3. **The start of a death carry.** An agent's carry starts at

```
offset(id) = ( (id * 2 654 435 769) mod 2^32 ) * D / 2^32          (integer, 128-bit helper)
```

a fixed function of the ID (a Weyl sequence by Knuth's multiplier), in `[0, D)`. It
draws nothing, so the random stream does not depend on the module (3.1.1; unlike a
creation draw it adds none to the order of `spec-agents.md` 3.5.2). Without it every
agent with the same ratio would reach its first death on the same tick (the lockstep of
`spec-scale.md` 5.1). The offset is the helper's "random starting offset" of
`spec-scale.md` 4.6 made deterministic. A newborn agent takes the offset of its own ID. The same function of the ID is the fixed per-agent spread of the migration module (the share of `MoveWidth`, `spec-migration.md` 6.5) and of the merchant (the fee share, `spec-merchants.md` 5.7): one helper, whose home is `internal/ordering` (the package of deterministic ordering helpers), with one test of its evenness; a new per-agent spread reuses it and does not draw at creation.

3.8.4. **Why exact.** Cumulatively, over any run of ticks the number of events is
`floor((carry_start + sum of num) / D)` less the same at the start: within 1 of the
expectation whatever the weights or the ratio path. There is no rounding bias and no
draw. Rates as low as 1 basis point a year at one household are representable: the
first event comes after `D` ticks, never earlier.

3.8.5. **Overflow.** `num_a` is at most `MaxWeight * 10 000` = 10^10 and the role sum at
most `MaxHouseholds * 10 000` = 10^13, both below `int64`; `D` is at most `10 000 *
MaxTicksPerYear` (`spec-config.md` 4.1). Every sum is checked.

## 4. From counts to weight changes

### 4.1 When it runs and what it hands over

In the entry-and-exit phase (`spec-agents.md` 4, phase 7), for each region in region
order, after the `Privation` that P6.1 set and before `Decide` (`spec-spawner.md` 10.2):
the module computes, from the state at P7.1(b) of the region (`spec-world.md` 6.9: after
the completions of traineeships of step (a), whose role changes it therefore sees, and
before any removal, switch or entry of the step),

- per agent, `PopDeaths`: the households to remove, `n_a` of 3.8.1 after the floor of
  4.4;
- per role, `PopBirths`: the households to add, `births_r` of 3.8.2 after the cap of 4.4.

and puts them in the spawner's view (`spec-spawner.md` 3.2). The default spawner executes them (4.8). A consumer's spawner may ignore them;
the carries have already advanced, so a number it ignores is not carried over. The
computation draws nothing, and its cost is proportional to the agents of the region
(NFR-POP-002).

### 4.2 Why two hooks and not a signed number

`spec-spawner.md` 3.2 and 8.2 carry one signed `PopDelta` per role. This spec needs two
non-negative numbers, for two reasons that the models of section 13 make concrete.

- **Gross flows.** A role of 100 000 households has about 7 births and 7 deaths a tick
  in balance. A net number would execute none of them: no money would change hands, the
  pool would not see the estates of the dead nor fund the stake of the newborn, and the
  stake loop of 5.5 would vanish from the model for large roles and appear for small
  ones. The result would depend on the scale of the group. Gross is the same at every
  scale.
- **Allocation.** A count per role has to be allocated to agents. SPN 8.2 allocates a
  negative number to the agents by privation, then by the least money per household. In
  model 2 (13.2) that rule takes only the poorest households, which hold nearly no
  money, so the pool receives nearly nothing and cannot fund births: with births equal
  to deaths the population fell by 86% in 80 years because 99% of the births could not
  be funded. A death that falls on each agent in proportion to its own rate
  (3.8.1) takes households at the money they really hold.

### 4.3 Births and the stake

A birth is an `Enter(role, n)` with cause `births` (`spec-spawner.md` 3.3) executed
under the funding of `spec-spawner.md` 4.3 and 7.3 with one difference: each household
brings the **stake**

```
stake = trunc( Role.StartMoney * BirthStake / 10 000 )
```

and not `Role.StartMoney`. The pool pays it with the ledger reason `births`, whether the
household joins an existing agent or becomes a new agent of weight 1 (the spawner uses
`start_money` for the entrants it makes by renewal; with the module on there are none,
4.6). The household brings no goods. The funded number is
`min(n, floor(budget / stake))` with the budget of `spec-spawner.md` 7.3 taken once per
region per tick; a stake of 0 is not limited by the pool. The households the pool
cannot fund are **not created, not deferred**, and are counted (`BirthsUnfunded`).

4.3.1. **Why a stake at all.** A new household with no money cannot buy food. It would be
unfed within a tick and reach the ladder in `RemovePrivation` ticks: births and
starvation would be one churn. `Role.StartMoney` is what the library already treats as
the money a household needs to start (`spec-agents.md` 3.3).

4.3.2. **Why a separate `BirthStake`.** In a closed region the money that funds the
stake is the money of the dead. If the stake exceeds the money a household holds on
average, the pool drains and births are limited by money and not by food (5.5). A
parameter lets a pack set the stake to what its money supply can bear. The default is
the full `StartMoney` so that, without a reason to change it, a newborn is the same
household as any entrant.

### 4.4 The floors and the cap

4.4.1. **`MinHouseholds`.** Deaths are taken per agent in ID order. For each, the
households taken are cut so that the households of the (region, role) group do not fall
below `MinHouseholds`. The deaths that were due and are not taken are dropped (not
deferred) and counted as `DeathsHeld` (10.1). The default is 1. The floor stops the
schedule from extinguishing a small group by arithmetic (6.4). It does not stop the
ladder: a group that starves to its end ends (`spec-spawner.md` 6.5, 5.3).

4.4.2. **`HouseholdCap`.** A region may carry a cap on its households (0 = none, the
default). After the deaths of the tick, a birth is allowed only while the households of
the region stay at or below the cap; the births that would exceed it are dropped and
counted (`BirthsCapped`). It is a bound, in the sense of STB 2.3, not a damper: it stops
a runaway where nothing else does (5.2), and a market pinned at it is a failure regime
that the counter shows. It is a configured constraint, like `ExitShare`: no agent
decision and no rate reads the household count through it, so it is not a rule that
reacts to an aggregate.

4.4.3. **The per-agent cap.** The households taken from an agent are at most its fed
households, `Weight - Unfed`. A cut is counted (`DeathsCut`). After the ladder's
reductions of the same tick, which take only unfed households (`spec-spawner.md` 5.3),
the fed households of the agent are untouched, so the cap holds in the plan order of
4.8.

4.4.4. **The role ceiling.** The births of a role in a region are capped at its headroom: the
least of the headroom of its class limit (`Role.Ceiling`, `spec-settlements.md` 6.4.2) and of its
body's (`spec-institutions.md` 3.4.3), counted at the start of the region's entry step
(`spec-spawner.md` 6.3.1). The births that would exceed it are dropped and counted
(`BirthsCeiling`, a field of `PopulationChanged`; `BirthsCapped` stays the count of
`HouseholdCap`). SET owns the rule and the name; a body's places are a second source of the same
cap and add no field. The order of the cuts on the births of a tick is: `HouseholdCap` for the
region (4.4.2), then the role ceiling, then the pool's funding (4.3, `BirthsUnfunded`). It is a
bound, like 4.4.2, and removes no household when a limit falls.

### 4.5 The ladder and the schedule: no household is counted twice

| Households | Rule that may act | Why not the other |
|------------|-------------------|-------------------|
| Unfed (`Unfed` of an agent) | the failure ladder: rung 1 role change, rung 2 relief, rung 3 reduction and removal (`spec-spawner.md` 5) | not exposed to the schedule (3.7.1) |
| Fed | the schedule: births and deaths | the ladder takes `min(Unfed, ...)` and so never a fed household (`spec-spawner.md` 5.3) |

The two sets are disjoint in every tick, and a household that is fed one tick and unfed
the next is under one rule at a time. Starvation deaths are therefore the ladder's, and
the module's deaths are the deaths that come with hardship and risk at a given ratio
(Clark's death schedule, 16) among households that are eating. The graded exit of the
ladder (`DeathStep`, `ExitShare`) is the high end of the death rate in this library, and
the schedule is its gentle part. The cause on the event tells which: `starved` and
`insolvent` are the ladder's, `population` is this spec's.

4.5.1. **What this costs.** The positive check of a severe shortage is the ladder's,
which is capped at `ExitShare` of the region's households a tick and delayed by
`RemovePrivation` (`spec-spawner.md` 11). A schedule that rises steeply at low ratios
cannot act on households that are unfed. Where a pack wants a faster exit, it sets
`ExitShare` and `DeathStep`.

### 4.6 `Renewal`

With the module on the effective `Renewal` is 0, whatever is configured (a rule
constant, not a parameter). Reason: `Renewal` replaces the households the ladder
removes, by default one for one (`spec-spawner.md` 7.1). In the Malthusian loop the
removal of an unfed household is the check that brings a population back to its
carrying level; a refill would cancel it and the population would be held where it was
by the pool's money (5.5). Births are the only replacement.

4.6.1. With the module on and `Renewal` equal to its default (10 000) validation reports
only the informational diagnostic `renewal_ignored` (same level as `replacement_ratio`,
5.7): a pack need not set `Renewal` to 0 to be silent. The warning
`renewal_with_population` (`spec-spawner.md` 8.3, 12) fires only when `Renewal` is set to a
non-default value above 0, which is a deliberate setting that has no effect. When the
module is off nothing changes.

4.6.2. This answers N-083 and `spec-spawner.md` 8.3, and closes the question of
"net of renewal": births are never netted.

### 4.7 Ledger reasons and the estate path

| Reason | Of | Kind | Between | Owner |
|--------|----|------|---------|-------|
| `births` | money | internal | estate to agent | `spec-spawner.md` 4.3 (extended here: every household born through this module, joining an existing agent or a new one) |
| `deaths` | money, goods | internal | agent to estate | this spec (new): a death that leaves households in the agent |

A death that takes an agent to weight 0 is a removal, with the reasons `estate_money`
and `estate_goods` of `spec-estates.md` 2.1 and the event `AgentBankrupted`
(`Cause` `population`); the account closes (`spec-agents.md` 3.6). So `deaths` flows
only for a partial reduction, as `starvation` does for the ladder's; in a world of
agents of weight 1 (the first consumer, N-030) every death is a removal and the reason
`deaths` never appears. The count of deaths is then read from `AgentBankrupted` with
cause `population` and from `PopulationChanged` (10.1).

Every flow is internal: the module has no faucet and no sink. Money and goods are
conserved exactly by every birth and death (FR-POP-022). The household count is not
conserved; it is rebuilt from `AgentSpawned`, `WeightChanged` and `AgentBankrupted`
(`spec-spawner.md` 13.1.1, FR-SPN-034).

### 4.8 Order within the phase

The plan of a region (`spec-spawner.md` 10.2) becomes: (1) the ladder's reductions and
removals; (2) the module's deaths (`Reduce` with cause `population`, agents in ID
order); (3) switches and applies; (4) entries by renewal (none: 4.6); (5) the module's
births. The deaths go before the budget is taken, so the estates of the dead fund the
same tick's births. Births go last so the household count that the cap reads includes
every death of the tick. The events are the executed operations' own
(`WeightChanged`, `AgentSpawned`, `AgentBankrupted`), then `EntryLimited`, then one
`PopulationChanged` per role (10.1).

## 5. The Malthusian fixed point

### 5.1 The statement

Clark's model (16) has three assumptions. In this library's terms:

1. **Births.** A household forms at an annual rate that is a function of its ratio, flat
   or rising. (This spec: `BirthSchedule`.)
2. **Deaths.** A household dissolves at an annual rate that falls as its ratio rises.
   (This spec: `DeathSchedule`.)
3. **Living standards fall as households rise.** The mean ratio is a falling function of
   the number of households. (Not this spec: a fixed factor, 5.2.)

With these the number of households `N` obeys, per year,

```
dN/dt = N * g(r(N)) / 10 000,        g(r) = birth(r) - death(r)   (basis points a year)
```

with `r(N)` the ratio the households read. `g` is negative below the **replacement
ratio** `r*` and positive above it. `N` rises while `r(N) > r*` and falls while
`r(N) < r*`, so it settles where `r(N*) = r*`. The location `r*` depends on the two
schedules alone, as Clark notes (16: the subsistence income "is determined without any
reference to the production technology"). What the technology and the fixed factor
decide is `N*`, the number of households at that ratio.

### 5.2 What provides each assumption in this library

| Assumption | Provided by | Status |
|------------|-------------|--------|
| 1 and 2: the rates | this spec | specified |
| 3: the ratio falls with households | a **fixed factor**: an endowment that a recipe's executions draw from (`Draws`, `spec-production.md` 6.5), with a limited stock or flow. Owner RES | **provided by `spec-resources.md` 8** (N-098): an endowment that binds, with a stated mechanism, closed forms and a stability condition by shape. A recipe with `Draws` still fails validation while the resources module is off (`ErrModuleDependency`, `spec-production.md` 10.2); with it on the pack has a fixed factor. Nothing has been run against the library |
| (a backstop) the money of the region | the stake loop of 5.5 | specified here; an accident of funding, not a Malthusian limit |
| (a bound) a design cap | `HouseholdCap` (4.4.2) | specified here |

Without a fixed factor (a pack with no binding endowment) every recipe has constant returns to labour (`spec-production.md`
3.1.2: a source with no endowment is limited by labour only), so the ratio does not
fall as households rise, `g` keeps its sign at the ratio the economy settles at, and
the population grows or shrinks without limit. That is a consequence of the library as
specified, not a defect of this spec, and it has three consequences for the pack
author, each now with a cure in `spec-resources.md`:

- the module is testable as a loop with a RES endowment that binds (`spec-resources.md` 8, scenarios RES-S1), or with a test double that makes
  the ratio a function of households (the models of section 13 do exactly this, and isolate the population rules from the resource's own timescale);
- without a binding endowment the only bounds are the money of the region (5.5) and `HouseholdCap`;
- validation reports `no_fixed_factor` (9.2) so the author knows; a pack cures it by a recipe with `Draws` on an endowment whose maximum flow can bind.

### 5.3 The ratio a closed economy can show

The welfare ratio is bounded above in a closed economy with no demand beyond upkeep:
total income net of inputs equals total spending, which is at most the cost of the
upkeep required, so the household-weighted mean ratio is at most 1 (10 000), and is 1
exactly when everyone is fed (N-054; `spec-viability.md` 4.7; `spec-demand.md` 7.1). With
wants the mean is `1 + W/U`, at most `1 + Wmax/U`. Two consequences for the fixed point.

- **Without wants** (the demand module off, no faucet) the households read a mean ratio
  near 10 000 when everyone is fed, and a surplus does not show in the ratio: it is a glut, bounded by the
  production halt (`spec-agents.md` 11.7). The schedule cannot tell abundance from
  bare sufficiency. If `r*` is below 10 000, households grow until scarcity pulls the
  ratio down to `r*`, that is until about `(10 000 - r*) / 10 000` of the upkeep value
  is unmet: the fixed point is at **chronic hunger**, and the check is mostly the
  ladder's, not the gentle schedule. If `r*` is above 10 000, `g` is negative at every
  ratio the economy can show and the population falls to its floor. Both are traps. This
  is a derivation from the cited identity, not a run; scenario POP-S4 (14.4) is the test.
- **With wants** the ratio ranges above 1, up to the ceiling the pack's wants allow, and
  a fixed point can sit where households are fed, above subsistence and below the
  ceiling: `10 000 < r* < 10 000 * (1 + Wmax/U)`. The default `r*` of 12 000 is meant for
  a pack with wants whose `Wref / U` is above 0.2 (the reference demand overlay has about
  0.32, `spec-demand.md` 7.1).

Validation warns when `r*` is above 10 000 and the demand module is off
(`replacement_above_closed_ceiling`, 9.2); the pack may have a faucet (MON, EXT) that
lifts the ratio, which is why it is a warning.

### 5.4 Local stability and the condition

Let `r(N)` fall with elasticity `eta` (`eta = -N r'(N) / r`, 0 for constant returns, at
most 1; for a labour exponent `beta` of a fixed-factor production function `eta = 1 -
beta`). Linearising 5.1 at the fixed point, with `n = ln N - ln N*`:

```
dn/dt = - k * x,        k = r* * g'(r*) * eta / 10 000        (per year)
```

where `x` is the ratio signal the households read and `g'(r*)` is the slope of `g` at
`r*` (basis points a year per basis point of ratio). `k` is the **gain**. With no delay
`x = n`-driven and the approach is a decaying exponential with time constant `1 / k`
years. The signal is not instantaneous: it is the welfare window mean (a delay of about
`T / 2` ticks, `T` the effective window, `spec-welfare-and-wages.md` 2.3.3) of the
economy's ratio, which follows the households through prices and production with its own
lag `tau_e`. With total delay `tau = (T/2 + tau_e) / TicksPerYear` years:

- **`k * tau` small: monotone approach.** In every run of model 1 (13.1) the approach
  was monotone up to `k * tau` of 0.6 (0.7 for the shorter windows).
- **`k * tau` near 1: overshoot.** The first overshoot (a second crossing of `N*`)
  appeared at 0.9 with the longest window tried (`T` 1 095) and at 1.4 with `T` 365.
- **`k * tau` above about 1.5: a sustained cycle,** bounded by the flat ends of the
  rate schedules: 2% to 4% of `N*` at 1.8 to 2.9 (`T` 365), and with `T` 1 095 3% at 1.2,
  6% at 1.8, 12% at 2.4 and 17% from 3.6.

For comparison the textbook values for a first-order lag (oscillation above 1/4) and a
pure delay (instability above `pi / 2`) bracket these. The model's window is a boxcar
and the schedule is clipped at its flat ends, so its numbers are the model's, not a
law.

For the default pack: `r* = 12 000` and `g' = 400 / 8 000 = 0.05`, so `k = 12 000 *
0.05 * eta / 10 000 = 0.06 * eta` per year, 0.03 for `eta` 0.5, a time constant of about
33 years. With `T` 30 and `tau_e` 30 ticks, `tau` is 0.12 years
and `k * tau` is 0.004: monotone, by a wide margin. The delays that would matter (a
generation) are not in the model (3.3).

**The condition the validator can check.** The gain is bounded above by `eta = 1`, and
the window is known; the economy's lag is not. The warning `population_gain` (9.2) fires
when the worst-case gain times the half window exceeds 1/2:

```
r* * dRate * T  >  10 000 * dRatio * TicksPerYear          (dRate/dRatio the slope of g at r*)
```

It is necessary and not sufficient: a long economy lag can still produce a cycle. The
register entry POP-A1 (14.2) states the full condition and scenario POP-S1 tests it.

### 5.5 The stake loop and the money-limited population

Births take the stake from the pool; deaths fill the pool with what the dead hold. In a
region with no money entering or leaving (no faucet, no migrant, no merchant), the
total money `M` of households plus the pool is constant. Then:

- if households grow, the money per household falls (dilution);
- the pool funds a birth only from the money of the dead; the funded births per death
  are about (money per dying household) / (stake);
- so growth stops, and then reverses, when the money per household falls to the stake.

The population is bounded by

```
N_M = M / stake
```

whatever the food supply says. Model 2 (13.2) confirms it: with births 400 and deaths
250 basis points a year, 1 000 households and `M` = 12 000 000, the population settled at
about 1 160 for a stake of 10 000 (`N_M` 1 200), 2 350 for 5 000 (2 400) and about 4 800 for
2 500 (4 800). This is a **money-limited carrying capacity**: a spurious fixed point
unrelated to the Malthusian one. It is the right behaviour in one sense (a household
cannot start without money) and a trap in another (a pack with food for 5 000 and money
for 1 200 will stop at 1 200). Three ways out: a lower `BirthStake`, more money
(a faucet, MON; an inflow with migrants, MIG), or knowing the bound. The warning
`births_unfunded` (10.3) shows when births are due and the pool cannot pay.

5.5.1. **The poorest-first trap** (the reason for 4.2): in model 2, with deaths taken
poorest first and births equal to deaths, the population fell from 1 000 to 143 in 80
years and the pool never held money, because the dead held none. The per-agent rule of
3.8.1 does not have this effect (994 households at 80 years in the same conditions).

5.5.2. **The pool's other uses.** The stake draws from the same pool as relief
(`spec-estates.md` 2.6) and renewal (none, 4.6); the spawner's budget `EntryPoolShare`
keeps at least half for relief by default (D37). In a collapse the dead fill the pool,
so the pool is fullest when the region has most need of births and relief (6.3).

### 5.6 What the models show

Section 13 has the details; the statements that matter for the design:

1. With the defaults and a ratio function of diminishing returns, the population
   approaches `N*` monotonically from above and from below; the approach is slow
   (a time constant of about 35 years at the default gain), by design of the placeholders.
2. A fall of the economy's productivity by 40% for ten years takes the ratio from 12 000
   to 7 500, the population falls by 18% and then recovers over more than a hundred
   years; the ratio overshoots upward while the population is low (13 106 at year 35).
3. A death schedule four times higher for one year (a shock) takes the population down by
   6% and *raises* the ratio from 12 000 to 12 410 (Clark's counterintuitive result, 16:
   what raises the death rate raises living standards), after which births exceed deaths
   until the population is back.
4. The gain and delay condition of 5.4.
5. The stake loop of 5.5.

### 5.7 The replacement ratio and validation

The **replacement ratio** `r*` is the ratio at which `birth - death` changes sign from
negative to positive as the ratio rises, found on the knots of the two schedules in
integer arithmetic: for the segment between two consecutive ratios of the union of the
knots where `g` changes sign, `r* = x0 + trunc(-g0 * (x1 - x0) / (g1 - g0))`. If `g` is
never negative, or never positive, over the knots, there is no replacement ratio
(`no_replacement_ratio`, 9.2): the population always grows or always shrinks. The value
is reported once at validation as the informational `replacement_ratio` (like
`window_extended`, `spec-welfare-and-wages.md` 11.4) and kept in the snapshot (10.4).

## 6. Collapse, recovery and the floors

### 6.1 No regime is set

There is no state called collapse, no flag, no mode. The library holds no aggregate that
agents react to. A **collapse** is a fall in households that the measures show
(`spec-stability.md` 7, STB-F3: households fall more than a stated share in a window).
This spec adds one measure to the snapshot, the **natural increase** (births less deaths
of the window, per region and role, 10.4), and one per tick event, `PopulationChanged`,
so the regime test is computable from events and the snapshot alone (FR-STB-012).

### 6.2 Contraction

A region contracts when the households' ratios are below `r*` (the schedule) or when
households go unfed (the ladder). The two work together and in order: a shock lowers the
ratio first, deaths exceed births, households fall, the ratio of those who remain rises
(5.1, assumption 3). A shock that leaves households unfed engages the ladder
(`spec-spawner.md` 5), which is graded (D35). The dead's holdings fill the pool
(`spec-estates.md` 2.1), which funds relief to those who remain (`spec-estates.md` 2.6)
and the stake of the births that follow. The estate listings of the dead's goods
(`spec-estates.md` 2.7) add a supply of goods to the market while the demand falls. The
loop L39 (a dry pool makes a collapse worse) is damped, in this module, by the pool
being at its fullest when the dead are most numerous.

### 6.3 Recovery and its speed limit

Once the ratio is above `r*` the net rate is positive, births resume, funded by the pool
that the dead filled. The fastest possible growth is set by the schedules:

```
g_max = max(birth) - min(death)           (basis points a year)
```

For the defaults, `250 - 50 = 200` bp, 2% a year: a region that has fallen to a tenth of
`N*` needs at least `ln 10 / 0.02` = 115 years to regrow (analysis, not a run). A pack
that wants a quicker recovery steepens the schedule (a higher top birth rate, a lower
bottom death rate) or has migrants (MIG). The recovery also needs money: births stop when
the pool cannot pay the stake (5.5), and `BirthsUnfunded` says so.

### 6.4 The floors

**`MinHouseholds`** (4.4.1). A group (region, role) is never reduced below it by the
schedule. Why: at a household count of a handful, the carry arithmetic of 3.8 can take
the last households at the ticks their carries cross `D`, and a role that disappears is
never re-seeded by the default spawner (`spec-spawner.md` 6.5, STB-L64). The schedule is a
smooth rate; the extinction of a role should be the ladder's decision, visible as hunger.
The default is 1. A pack that accepts the extinction of a small role by chance sets it to
0.

**`MinAgents` (`K`).** A death reduces an agent's weight and removes the agent only at
weight 0 (`spec-scale.md` 4.8). The group's agent count falls below `K` only through
removals, which warn (`group_below_floor`, `spec-spawner.md` 12) and are not rebuilt
silently (FR-SPN-047). The births refill the agents of smallest weight below the scale
first (`spec-spawner.md` 4.3), so growth rebuilds the agent count at the pace of
`Scale`. At scale 1 every household is its own agent and the floor is not engaged.

**The last household of a region.** When the households of a region are 0 the region is
**extinct**. Nothing here can revive it: births need exposed households, and the default
spawner never re-seeds a role (`spec-spawner.md` 6.5). Extinct is absorbing until an
external agent, MIG or the consumer adds households. The world emits a bounded `Warn`
"region extinct" (10.3). The pool of an extinct region keeps its money (the estates of the
last households), which is not lost: it is available when households return.

### 6.5 A closed region

A region with no immigrant has only its own births, deaths and ladder. This is the
behaviour tested before MIG exists. Its properties, from 5.5 and 6.3: the population is
bounded by `M / stake`; recovery from a collapse needs the money the dead left; and a
region whose households are all dead stays so. A migrant arriving with money (MIG) lifts
all three, by what it carries; a household pushed to move is poorer than a mean one and dilutes the money per household of the region it enters (`spec-migration.md` 9.3, STB-L76). No stake is paid for it and nothing is taken from either pool (7).

### 6.6 Shocks

A shock that changes the death rate (a disease, or any hazard that is
not a lack of food) acts through the region's `DeathScale`; one that changes births
through `BirthScale`. Each is a multiplier the consumer sets between ticks and clears, in
the way the yield shock is (`spec-production.md` 6.8): the effective scale is world state
(saved, 11), and the configured value is its default (an unset scale is the configured
value, not 10 000: FR-POP-039). INT owns the calls that set them, `SetBirthScales` and
`SetDeathScales`, each over a list of `RegionScale{Region; BP}` (`spec-integration.md` 5.1);
this spec owns the fields and their effect, and each change is the event `PopulationScaled`. A shock that removes
households at once (a battle, a flood) is a consumer call of the spawner's kind, with
cause `consumer` (`spec-spawner.md` 3.3), and is not a schedule.

### 6.7 A trap at low density (a hazard, not modelled)

Nothing in the rules makes the ratio fall *less* as households fall; it falls with
diminishing returns, rising as households fall. But a real economy can have the
opposite at low density: too few households to run a chain (complementary roles, a
minimum market), so the ratio *falls* as the households fall and a low-density fixed
point is stable and `N*` is unstable above it (an Allee effect). The library can show it
in a pack where a role of the chain disappears (`spec-spawner.md` 6.5, STB-L64) and its
output is a need of the rest. This spec does not remove it. It is a register entry
(POP-A8, hand) and a scenario (POP-S2) shows whether a pack has it; `MinHouseholds` is the
only mechanism that bears on it.

## 7. Interactions

| With | What |
|------|------|
| `spec-welfare-and-wages.md` | the signal (3.4); the effective window is part of the delay (5.4). A longer window slows the response and lowers the gain tolerance |
| `spec-agents.md` | `Unfed` defines the exposed households (3.7); the agent gains the field `DeathCarry` (11); the weight change leaves it unchanged (`spec-agents.md` 3.7); creation sets it (3.8.3) |
| `spec-spawner.md` | the view, the plan order, the funding, `Renewal` (4) |
| `spec-settlements.md` | a role's `Needs` closes it for births in a class that lacks the flag (3.7.3) and its `Ceiling` caps its births at the headroom, counted in `BirthsCeiling` (4.4.4); SET owns the rule |
| `spec-institutions.md` | a body's places make a second source of the same headroom (3.4.3 there); no field is added |
| `spec-estates.md` | the dead's holdings fill the pool; births draw from it; relief shares it (4.7, 5.5.2). N-082: the order phase 1 relief, phase 7 entry holds |
| `spec-scale.md` | the carried remainder (3.8, `spec-scale.md` 4.6); births into cohorts dilute their goods (4.3) |
| `spec-demand.md` | with wants the ratio ranges above 1 (5.3). Without them the fixed point is at hunger |
| `spec-viability.md` | a warm start's census is not `N*`. A world that starts at its census and has a fixed factor drifts toward `N*` (open question 3) |
| `spec-migration.md` | the other way a household enters or leaves: the signal is not shared and no stake is paid; a migrant carries its account and draws nothing from the pool, so an arrival creates no household and no money (`spec-migration.md` 5.5, 9.2); the replacement ratio is not a migration signal (a pack with both modules may set `MoveBelow` near `r*` by hand); the money a migrant carries moves the bound `M / stake` (6.5) |
| `spec-resources.md` | provides the fixed factor (5.2; `spec-resources.md` 8): an endowment that binds, regeneration shapes, the ration |
| `spec-integration.md` | sets and clears the scales (6.6) |
| `spec-calendar.md` | `TicksPerYear` converts annual rates exactly (3.8); no seasonality of births or deaths in version 1 |

## 8. Parameters

All are tunables (architecture.md 9.5) with a default, an owner, a bound and a doc line,
fields of `Config.Modules.Population`, with rows in the parameter-owner table. **Every
default is a placeholder** (N-015, N-027); none comes from a source.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Module | `BirthSchedule` | list of knots; annual birth rate of a household by its ratio (3.5) | `[(8 000, 250), (16 000, 250)]` (flat) | 1 to `MaxKnots` knots; ratios strictly increasing, 0 to `MaxBasisPoints` |
| Module | `DeathSchedule` | list of knots; annual death rate by ratio | `[(8 000, 450), (16 000, 50)]` | as above |
| Knot | `Ratio` | welfare ratio, basis points | | 0 to `MaxBasisPoints` |
| Knot | `Rate` | annual rate, basis points | | 0 to 10 000 |
| Module | `MinHouseholds` | households a (region, role) group keeps against the schedule's deaths (4.4.1) | 1 | 0 to `MaxHouseholds` |
| Module | `BirthStake` | share of `Role.StartMoney` that each newborn household brings, basis points (4.3) | 10 000 | 0 to 10 000 |
| Region | `BirthScale` | multiplier of the birth schedule, basis points (3.6) | 10 000 | 0 to `MaxBasisPoints` |
| Region | `DeathScale` | multiplier of the death schedule | 10 000 | 0 to `MaxBasisPoints` |
| Region | `HouseholdCap` | most households a region may hold; births beyond are dropped; 0 = none (4.4.2) | 0 | 0 to `MaxHouseholds` |

`MaxKnots` is a bound of `spec-config.md` 4.1: **8** (provisional). The upper bound
10 000 on a `Rate` is not a tunable: it is the largest probability per year.

In a file the module is the table `[population]` with the two schedules as lists of
`{ ratio, rate }` and the regions as tables keyed by region ID, so layers merge by key
(`spec-config.md` 6). A region not listed has the defaults.

**Region-keyed table, in REG's sense.** `Modules.Population.Regions[]` is a region-keyed
table of the mechanism of `spec-regions.md` 5.1, one row per region, owned by this spec
for meaning, default and bound. `BirthScale` and `DeathScale` use the **scale** mode (the
effective rate is the world's schedule times the factor over 10 000, with the truncation
rule of 3.6); `HouseholdCap` uses **replace** (the row's value, else 0 = none). Absence is
the default, with no warning; a row is resolved once into a dense array by `RegionID`; the
canonical order is ascending `RegionID` (the region IDs bytewise, `spec-regions.md` 3.3.1
and 5.3.4); the errors are those of `spec-regions.md` 5.3.1 and 5.3.2 (`ErrUnknownRegion`,
`ErrDuplicateID`), not restated. The effective scales a shock sets are the separate
slot of 6.6 and edit no row (`spec-regions.md` 5.3.6). The table needs a row in
`spec-regions.md` 5.1.3.

Why these defaults, and what they are not. They cross at 12 000, between a bare
sufficiency (10 000) and the ceiling of the reference demand overlay (about 13 200,
`spec-demand.md` 7.1), at 250 basis points a year (2.5%), which with the 365-tick year
makes the arithmetic in 12 easy to follow. They are not the rates of any society; the
source read for this spec does not give usable ones (16). A pack calibrates them
against the stylised facts of `spec-stylised-facts.md` when it exists.

The rate scale of an example may use a historical figure, labelled (12.1 uses none).

## 9. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9); errors name the path.

### 9.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a schedule with no knots or more than `MaxKnots`; knot ratios not strictly increasing, or outside their bound; a `Rate` outside 0 to 10 000; `MinHouseholds`, `BirthStake`, `BirthScale`, `DeathScale` or `HouseholdCap` outside its bound |
| `ErrUnknownRegion`, `ErrDuplicateID` | the rows of `Regions` as `spec-regions.md` 5.3.1 and 5.3.2 say, cited and not restated |
| (none) | No rule needs `ErrOverflow`: every product is bounded by the constants of `spec-config.md` 4.1 (3.8.5), and `Role.StartMoney` times the largest weight is already checked (`spec-config.md` 4.2). The arithmetic is checked all the same |

### 9.2 Warnings

Never fail validation (`spec-config.md` 7.2).

| Warning | Condition |
|---------|-----------|
| `renewal_with_population` | the module is on and `Renewal` is set to a non-default value above 0 (4.6.1). Named by `spec-spawner.md` 8.3, 12. At the default it is only the informational `renewal_ignored` |
| `no_replacement_ratio` | `birth - death` is never negative or never positive over the knots (5.7): the population always grows or always shrinks |
| `schedule_direction` | a birth schedule falls, or a death schedule rises, anywhere (3.5.4) |
| `replacement_above_closed_ceiling` | `r*` is above 10 000 and the demand module is off (5.3) |
| `population_gain` | `r* * dRate * T > 10 000 * dRatio * TicksPerYear` with `T` the longest effective welfare window (5.4) |
| `no_fixed_factor` | no recipe draws on an endowment whose maximum flow can bind (the households at which the draw equals the sustainable draw, `spec-resources.md` 8.2, 11.3, below the census's households times a stated margin) and no region has a `HouseholdCap` (5.2). Before RES this was true of every pack; the margin is open (below) |
| `population_without_upkeep` | no role has an upkeep (the `no_upkeep` case of `spec-agents.md` 13): the module is inert |

The informational diagnostics `replacement_ratio` (5.7) and `renewal_ignored` (4.6.1) never fail validation. A warning never fails validation, but the **acceptance** of a pack by `spec-stylised-facts.md` reads `replacement_above_closed_ceiling` as a rejection of the pack for a scenario that expects a healthy start (STY C1: with the population module on, no demand module and no faucet, the default schedules have no healthy start). `replacement_ratio` names `r*` and the knot segment it
lies on, once at validation, and never fails.

## 10. Observability

### 10.1 Event: `PopulationChanged`

Emitted once per region per role per tick in which the module is on and at least one
count other than `Exposed`, `Unfed` and `NoRatio` is non-zero, after the region's plan
has been executed and after `EntryLimited`; never when all are zero, so a quiet tick
costs nothing. Roles in role order within a region. Bounded by regions times roles. No
map, no pointer, no wall-clock time.

| Field | Meaning |
|-------|---------|
| `Tick`, `Region`, `Role` | |
| `Exposed` | exposed households (3.7) summed over the role's agents |
| `Unfed` | households not exposed because unfed |
| `NoRatio` | households not exposed because their agent has no ratio |
| `BirthsDue`, `DeathsDue` | the whole households that the carries produced (before the floors and the cap) |
| `DeathsCut` | deaths cut to the fed households of an agent (4.4.3) |
| `DeathsHeld` | deaths dropped by `MinHouseholds` (4.4.1) |
| `BirthsCapped` | births dropped by `HouseholdCap` (4.4.2) |
| `BirthsCeiling` | births dropped by a role ceiling, a class limit or a body's places (4.4.4) |
| `BirthsUnfunded` | births the pool could not fund (4.3) |
| `RateClamped` | exposed households whose effective rate was clamped at 10 000 (3.6) |
| `BirthsDone`, `DeathsDone` | households actually added and removed by the module in the tick |

`DeathsDone` counts the households removed with cause `population`, whole removals and
reductions. The household count is still rebuilt from `AgentSpawned`, `WeightChanged` and
`AgentBankrupted`; this event is the aggregate for metrics and the regime measures.

10.1.1. **The executed operations keep their own events.** A death that leaves households
is a `WeightChanged` with `Cause` `population`, a birth into an existing agent a
`WeightChanged` with `Cause` `births`, a birth that makes a new agent an `AgentSpawned`,
and a death that empties an agent an `AgentBankrupted` with `Cause` `population`
(`spec-spawner.md` 13.1). A `WeightChanged` carries the money and the units of goods that
the change moved, `Money` and `Units` (`spec-observability.md` 4.1.4), so the money that a
birth takes from the pool as a stake and the money that a death returns to the estate are in
the event of the operation as well as in `BoundaryFlow`. `PopulationChanged` adds the aggregate and the counters of the
clamps, which no per-agent event can carry.

### 10.2 Metrics

Proposals (OBS decides), derived from `PopulationChanged` (observability-and-logging.md
5.3): `economysim.household.births` and `economysim.household.deaths`, Counters, unit
`{household}`, attribute `economysim.role`, fed by `BirthsDone` and `DeathsDone`. No
other attribute (cardinality: roles, 5.4.1). The unfunded and held counts are available
as events and in the snapshot; a gauge for them is OBS's call.

### 10.3 Logs

All with constant messages and the standard keys. `Warn`, bounded per region or role
(observability-and-logging.md 3.2.2), emitted in the first tick of the condition and every
tenth after:

| Message | Condition | Attributes |
|---------|-----------|------------|
| "region extinct" | a region has no households | `region` |
| "births unfunded" | births were due and the pool funded none | `region`, `births` (the due count of the last tick) |
| "deaths held" | `MinHouseholds` held deaths in a group | `region`, `role`, `deaths` |

The three conditions are of pattern A (`spec-observability.md` 7.3): the line is emitted at
the first tick of the condition and every tenth tick after it, as the header says, so the
run of consecutive ticks of each condition is state (11).

`Debug`, per region per tick, guarded by `Enabled`: "population step" with `region`,
`births`, `deaths`. No `Info`, no `Error`. New log keys, added with the code
(observability-and-logging.md 3.3.2): `births` and `deaths`, counts of households.

### 10.4 Snapshot

Per region and role: the household-weighted mean annual birth rate and death rate at the
agents' current ratios (basis points a year), their difference (the **natural
increase** rate), the birth carry and the unfed households. The fields are
`spec-observability.md`'s (5.3.4 and 5.3.10): the unfed households are `RoleRow.Unfed`, a
state at the boundary, not a count of the last tick (the last tick's are in
`PopulationChanged.Unfed`). Per region: the pool balance (EST) and, if set, the cap. Per
module: `r*` and the gain. These are measures; nothing reads them.

## 11. State contributed to a checkpoint

All plain data in a fixed order, no maps (ADR 0002):

- the **death carry** `carry_a` of each agent, an `int64` in `[0, D)`, is hosted in the agent
  record and saved there, once (`spec-agents.md` 3.1, 15; `spec-checkpoint.md` 4.2 tag 3,
  FR-CHK-041); this section does not list it again;
- per (region, role), in ascending `RegionID` then role order (`spec-regions.md` 3.3.2): the **birth carry** `carry_r`, in
  `[0, D)`;
- per region: the effective `BirthScale` and `DeathScale` when a shock has set them
  (INT's slot, 6.6);
- per region: the run of consecutive ticks of a region with no households ("region
  extinct") and the run of ticks of unfunded births, and per (region, role) the run of ticks
  of held deaths: the three runs of the three `Warn`s of 10.3 (diagnostic: they affect no
  result and are saved so a resumed run logs as the original did).

Not saved: the exposed households, `PopBirths` and `PopDeaths` (recomputed each tick),
`r*` and the gain (derived from the configuration). A world restored between ticks makes
the same births and deaths as one that was never stopped (FR-POP-035). The module adds no
account to the ledger and closes one per removal only (`spec-spawner.md` 14).

## 12. Worked examples

The numbers are placeholders chosen to be easy to follow and were checked by a short
integer script (`models/pop/worked.py`, not code of the library). Both examples are
settings of the author's invention; neither claims to describe a real society.

### 12.1 An agrarian realm (an example)

A realm with `TicksPerYear` 365, so `D = 10 000 * 365 = 3 650 000`, the default
schedules, `MinHouseholds` 1, `BirthStake` 5 000 (half of `StartMoney` 10 000), the
demand module on.

Rates by ratio:

| Ratio | Birth | Death | Net |
|-------|-------|-------|-----|
| 5 000 | 250 | 450 | -200 |
| 8 000 | 250 | 450 | -200 |
| 12 000 | 250 | 250 | 0 (the replacement ratio) |
| 14 000 | 250 | 150 | +100 |
| 20 000 | 250 | 50 | +200 |

At 14 000 the death rate is `450 + trunc(-400 * 6 000 / 8 000) = 450 - 300 = 150`.

**A cohort.** Agent 17 has weight 40, `Unfed` 4 and ratio 14 000. Its exposed
households are `40 - 4 = 36`, the four unfed ones being the ladder's. The death
numerator is `36 * 150 = 5 400` a tick. Its carry starts at
`offset(17) = ((17 * 2 654 435 769) mod 2^32) * 3 650 000 / 2^32 = 1 849 008`. After
tick 1 the carry is 1 854 408, after tick 2 1 859 808, after tick 3 1 865 208 (nothing
due). At tick 334 it is 3 652 608, which is at least `D`: one household dies, the carry
becomes 2 608. The agent goes to weight 39, money and goods scale by 39/40 and the
difference goes to the estate (reason `deaths`). The fed households of the agent (36)
bound the deaths of the tick at 36, so the ladder (which takes only the four unfed
households) and the schedule cannot take the same household.

**A role.** A role has 1 000 fed households at birth rate 250. The numerator is
`1 000 * 250 = 250 000` a tick, and the birth carry crosses `D` at tick 15
(`15 * 250 000 = 3 750 000`): one birth, the carry becomes 100 000. About 25 births in
a year, 2.5% of 1 000 households, as the rate says.

**Funding.** The birth is due with a pool of 30 000 and `EntryPoolShare` 5 000. The
budget is `30 000 * 5 000 / 10 000 = 15 000`. The stake is `10 000 * 5 000 / 10 000 =
5 000`, so up to `15 000 / 5 000 = 3` households a tick can be funded. The birth is
funded; the pool becomes 25 000 (reason `births`, to the smallest agent of the role
below the scale or a new agent of weight 1). With a pool of 3 000 the budget is 1 500,
which pays for no household: the birth is not created and `BirthsUnfunded` is 1.

**The gain check.** `r* * dRate * T = 12 000 * 400 * 30 = 144 000 000`, against `10 000
* dRatio * TicksPerYear = 10 000 * 8 000 * 365 = 29 200 000 000`: far below, no warning.

### 12.2 A habitat (an example)

A habitat in a different setting with `TicksPerYear` 640 (a calendar of 320 days at two
ticks a day, `spec-calendar.md`), so `D = 6 400 000`. Its birth schedule rises and its
death schedule falls steeply, a community that counts every household:

```
BirthSchedule = [(6 000, 300), (18 000, 700)]
DeathSchedule = [(6 000, 900), (18 000, 100)]
```

| Ratio | Birth | Death | Net |
|-------|-------|-------|-----|
| 4 000 | 300 | 900 | -600 |
| 9 000 | 400 | 700 | -300 |
| 12 000 | 500 | 500 | 0 (`r*`) |
| 15 000 | 600 | 300 | +300 |
| 18 000 | 700 | 100 | +600 |

At 9 000 the birth rate is `300 + trunc(400 * 3 000 / 12 000) = 400` and the death rate
`900 + trunc(-800 * 3 000 / 12 000) = 700`. With 400 fed households of weight 1 at
ratio 9 000 the numerators are `400 * 400 = 160 000` (0.025 births a tick) and `400 *
700 = 280 000` (0.044 deaths a tick): the habitat shrinks by 3% a year while the ratio
stays low.

**A shock.** A disease sets the region's `DeathScale` to 30 000 for a time. The death
rate at 9 000 becomes `min(10 000, 700 * 30 000 / 10 000) = 2 100`; the numerator
`840 000` (0.131 deaths a tick, about one every 8 ticks). When the shock is cleared the
scale returns to 10 000. The ratio of the survivors rises as the households fall
(5.1, assumption 3), and the net rate crosses zero at a lower number of households.

**The cap.** The habitat has life support for 600 households: `HouseholdCap` 600. At 598
households with 3 births due, `min(3, 600 - 598) = 2` are born and 1 is dropped and
counted (`BirthsCapped` 1). The cap acts as a bound, and the event shows when it does.

**The floor.** A role of 1 household with a death due: `MinHouseholds` 1 holds the death,
`DeathsHeld` is 1, and the household stays.

**The gain check.** `12 000 * 800 * 30 = 288 000 000` against `10 000 * 12 000 * 640 =
76 800 000 000`: no warning. The demand module is on in this habitat, so `r*` of 12 000
lies under its ceiling; with it off the warning `replacement_above_closed_ceiling`
would fire.

## 13. What was run

Three small Docker Python scripts in `models/pop/` (`docker run --rm -v "$PWD/doc/specs/models/pop":/w
-w /w python:3.12-slim python <script>`). They are **models of the rules of this spec, not
of the library**. The population rules (the schedules, the carry, the floor, the stake)
are integer and as in sections 3 and 4. Floating point is used only for the economy's
assumed ratio function and for the printouts. The economy is replaced by one function
(model 1) or by money alone (model 2). The numbers are placeholders and show no default
to be stable (N-027).

### 13.1 Model 1: the fixed point (`fixed_point.py`, `sweep.py`)

The mean ratio the households would read at `N` households is `R(N) = RA * (N0 / N)^eta`
(`RA` 16 000, `N0` 100 000, `eta` 0.5 unless stated: diminishing returns on a fixed
factor), followed by a first-order lag `tau_e` and averaged over a window `T`; births and
deaths as the spec; `N* = N0 * (RA / r*)^(1 / eta)` = 177 778.

1. **Approach.** With the defaults (`T` 30, `tau_e` 30), `N` starts at 100 000 and at
   300 000 and approaches `N*` monotonically: 118 145 and 264 772 after 10 years,
   175 041 and 180 532 after 110; a time constant of about 35 years, as the
   linearisation (5.4) says (33 years at `eta` 0.5).
2. **Gain and delay.** The death ramp is narrowed to give a gain `m` times the
   default's without changing the rates. For `T` 365, `tau_e` 365: monotone up to `k * tau` of
   0.7 (`m` 16); a second crossing from 1.4; cycles of 2% at 1.8, 4% at 2.2 and 7% at
   4.3. For `T` 1 095, `tau_e` 365: a second crossing from 0.9, cycles of 3% at 1.2, 6% at
   1.8, 12% at 2.4 to 3.0 and 16% to 17% from 3.6. The cycle is bounded by the flat ends
   of the schedules.
3. **A shock to productivity.** `RA` at 60% for years 20 to 30 (defaults): the ratio
   falls from 12 000 to about 7 500, `N` falls from 177 777 to 145 803 by year 30 and
   regrows: 164 494 at year 60, 175 632 at year 115; the ratio rises to 13 106 at year 35
   while `N` is low.
4. **A shock to deaths.** `DeathScale` 40 000 for year 20: `N` falls to 166 746, the ratio
   rises to 12 410, and `N` regrows to 177 529 by year 115: Clark's result (16) in the
   model.

What it does not show. No unfed households, no ladder, no money, no pool; the economy's
response is an assumed lag, not prices and production; a single role; `N` large so the
quantisation of whole households is negligible; the cycle thresholds belong to a boxcar
window and a clipped ramp. It does not show that any pack of the library has these
properties.

### 13.2 Model 2: the stake loop (`stake.py`)

1 000 households with exchanged money (a random re-split of each pair's money), a pool
filled by the dead and drawn by births, `EntryPoolShare` 5 000, no relief. Rates held
fixed.

| Case | Result |
|------|--------|
| births = deaths = 250, stake 10 000, deaths **poorest first** | population 1 000 to 139 in 80 years; 9 of 898 births funded |
| the same, stake 5 000 | 143 in 80 years; 18 of 902 funded |
| the same, deaths **spread** (each household in proportion) | 994 (stake 10 000) and 997 (stake 5 000); 2 230 of 2 236 and 2 240 of 2 243 funded |
| births 400, deaths 250, deaths spread, stake 10 000 | population settles at about 1 160 (`N_M` = 1 200) |
| the same, stake 5 000 | about 2 350 (2 400) |
| the same, stake 2 500 | about 4 750 (4 800) |

What it shows: 4.2 (the allocation) and 5.5 (the money-limited population). What it does
not show: the money exchange is an assumption and not the market; there is no relief,
which would return pool money to the households (the pool in the "spread" cases holds a
large part of the money because nothing releases it); no production, no prices.

### 13.3 The worked examples (`worked.py`)

The integer arithmetic of section 12.

## 14. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9; fields as its section 3. The
register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn` numbers. The entries here
are named `POP-An`, the dampers `POP-Dn` and the scenarios `POP-Sn` so that several
specs written at once do not collide. **Status is `model` where section 13 supports it and
`hand` otherwise; nothing here claims stability.**

### 14.1 Dampers this spec adds or specifies

| ID | Damper | Home | Status |
|----|--------|------|--------|
| POP-D1 | Bounded gain: the slopes of the two schedules set the gain `k`; the validator warns when the half-window alone makes `k * tau` exceed 1/2 (`population_gain`) | POP 3.5, 5.4 | specified (a warning; the economy's lag is not known to it) |
| POP-D2 | Disjoint rules: the schedule acts only on fed households and the ladder only on unfed ones, so no household is counted twice and the positive check is graded | POP 3.7, 4.5 | specified (a rule constant; counted by `Unfed` and `NoRatio` of `PopulationChanged`) |
| POP-D3 | No renewal: `Renewal` is 0 while the module is on, so the ladder's removals are not refilled | POP 4.6 | specified (a rule constant; a non-default value is warned) |
| POP-D4 | Floor of households: `MinHouseholds` against the schedule's deaths. Cost: a small group never dies out by the schedule | POP 4.4.1 | specified (counted by `DeathsHeld`) |
| POP-D5 | Cap of households: `HouseholdCap`. Cost: births dropped at the cap | POP 4.4.2 | specified (counted by `BirthsCapped`) |
| POP-D6 | Funding limit by the stake: births only as the pool pays, `BirthStake`. Cost: the money-limited population of 5.5 | POP 4.3, 5.5 | specified (counted by `BirthsUnfunded`) |
| POP-D7 | Death in proportion: deaths fall on each agent in proportion to its own rate, by a per-agent carry with a spread start, so deaths are neither synchronous nor concentrated on the poor | POP 3.8 | specified (model 2) |

### 14.2 Loops, hazards and amplifiers

**POP-A1. The Malthusian loop.** *Kind:* loop. *Loop:* households (+) crowding of the
fixed factor (-) the ratio (+) net growth `birth - death` (+) households. Polarity B
(an odd number of negative links), Malthusian by design, with a long delay.
*Delay:* the welfare window `T` (mean `T / 2`), the economy's own lag `tau_e`, and
nothing else (no generation lag, 3.3). *Owner:* POP 5; the third assumption is RES's;
extends L50 and L46. *Damper:* POP-D1, D2, D7, and the rate bounds (a rate is at most
10 000). *Undamped:* a long `tau_e`: the warning sees only the window. *Condition:*
`k * tau` below about 1 (model: monotone to 0.7, overshoot from 0.9 to 1.4, a cycle from
about 2); `k` from the schedules and `eta` of the pack. *Test:* POP-S1. *Status:*
model (13.1). *Needs a fixed factor (5.2); without one the loop has no feedback.*

**POP-A2. The stake loop.** *Kind:* loop. *Loop:* households (+) money per household (-)
funded births (+) households. Polarity B, a spurious Malthusian limit set by money.
*Delay:* the pool's turnover (a death to a birth). *Owner:* POP 4.3, 5.5; touches EST
2.3 and SPN 7.3, extends L63. *Damper:* POP-D6, D7; money from outside (MON, MIG).
*Undamped:* a closed region cannot grow past `M / stake`. *Condition:* `M / stake` above
the food-limited `N*`. *Test:* POP-S3. *Status:* model (13.2).

**POP-A3. Recovery needs the dead's money.** *Kind:* loop (damper of L39). *Loop:*
collapse (+) deaths (+) the pool's money (+) funded births and relief (-) the
collapse. Polarity B, with the delay of the births. *Owner:* POP 6.2, 6.3; EST 2.1, 2.6.
*Damper:* POP-D2, D6; D11, D12, D37. *Undamped:* an extinct region (absorbing). *Condition:*
households remain. *Test:* S14, POP-S2. *Status:* hand.

**POP-A4. Selection by role.** *Kind:* loop. *Loop:* a role's ratio (+) its births, net of
deaths (+) its share of households (-) its ratio (a wage falls with supply). Polarity
B, much slower than the spawner's role change (L36). Births
enter the parents' role (3.2), so roles with a higher ratio grow faster and the
spawner's damped switching acts on a population whose composition also drifts.
*Owner:* POP 3.2; SPN 6. *Damper:* D14, D09 for the switching; rates bounded. *Condition:*
the two rates of change are far apart (births 2.5% a year against switching per tick).
*Test:* S06, POP-S1. *Status:* hand.

**POP-A5. Renewal refills the check.** *Kind:* hazard. *Loop:* ladder removals (-)
households (+) renewal (+) households. Polarity R if enabled: a second positive loop on
the same flow. *Owner:* POP 4.6; SPN 7.1. *Damper:* POP-D3. *Undamped:* none while the
module is on. *Test:* POP-S4. *Status:* hand.

**POP-A6. No fixed factor.** *Kind:* hazard. Without a fixed factor the ratio does not
fall with households and the rate has the same sign for any `N` (5.2). *Owner:* POP 5.2;
RES. *Damper:* POP-D5, POP-D6 (the money). *Condition:* a fixed factor (a RES endowment that binds). *Test:*
POP-S1 with a RES endowment or a double. *Status:* hand.

**POP-A7. The closed ceiling.** *Kind:* hazard (N-054; L46). In a closed economy with
upkeep only the mean ratio is at most 1 (5.3): the schedule cannot separate abundance
from sufficiency, so the fixed point is at chronic hunger (`r*` below 10 000) or the
population dies out (`r*` above it). *Owner:* POP 5.3; DEM 7. *Damper:* wants (DEM) or a
money source; the warning `replacement_above_closed_ceiling`. *Condition:* the pack's
ceiling `1 + Wmax/U` above `r*`. *Test:* POP-S4. *Status:* hand (a derivation from the
cited identity).

**POP-A8. A low-density trap.** *Kind:* hazard. At low density the ratio may fall with
households (a chain without enough households to run), making a low fixed point stable
and `N*` unstable above it (6.7). *Owner:* POP 6.7. *Damper:* POP-D4 only. *Condition:* a
pack whose chain survives at the smallest group. *Test:* POP-S2. *Status:* hand.

**POP-A9. The ladder's cap bounds the positive check.** *Kind:* constraint. A severe
shortage is checked by the ladder, capped at `ExitShare` a tick and delayed by
`RemovePrivation` (4.5.1); the schedule does not see unfed households. *Owner:* POP 4.5;
SPN 5. *Damper:* D35. *Condition:* a pack wanting a faster check sets the ladder. *Test:* S14.
*Status:* hand.

### 14.3 Placeholders, delays against neighbours

Delays near each other (checklist 9.6): the window `T` (30 ticks) against the spawner's
window-based review (`entry_share_window`, `spec-spawner.md` 12) are both multiples of
the same window; the loop of this spec is much slower. The
generation lag, which would make POP-A1 oscillate, is not modelled (3.3).

### 14.4 Scenarios proposed

Numbers are assigned by the register owner; `POP-Sn` stands for them. Wave 3: POP-S1 to POP-S4 (S37 to S40 in the register) may now use RES for the fixed factor; none has been run, and the caveat of every number here stands. The bands of S37 to S40 are those of `spec-stylised-facts.md`: STY-W02 (S37), STY-W03 (S38), STY-W06 (S39), and the keys of its 5.11 (`w06.money_cap_bp`, `s40.hunger_bp`); this spec states what each scenario asserts and STY sets the numbers.

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| POP-S1 | A region with a fixed factor (a RES `refill` endowment, scenario RES-S1 of `spec-resources.md` 19.4, or a test double that makes the ratio a function of households), the demand module on, started at 0.6 and at 1.6 times `N*` | the population approaches `N*` within a band and, for `k * tau` below 0.5, without crossing it more than once; the mean ratio at the end is within a band of `r*` | A1, A4, A6 |
| POP-S2 | A yield shock that cuts the region's income for a time, then restores it, with the pool, relief and the ladder on | households fall and regrow; the ratio rises while they are low; the extinct-region `Warn` does not fire; and with a chain whose role disappears the run is reported as F3 and the trap is visible | A3, A8 |
| POP-S3 | A closed region with the pool empty, `BirthStake` 10 000, 5 000 and 0, and births above deaths | the population stops at about `M / stake`; `BirthsUnfunded` is non-zero when it does | A2 |
| POP-S4 | The default schedules with the demand module off and `r*` above 10 000, and with `r*` just below it; and a pack with `Renewal` set to a non-default value above 0 | the first dies out or is flagged; the second sits at a bounded hunger and the ladder's counters show it; `Renewal` is ignored, and warned only at the non-default value | A5, A7 |

## 15. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-POP-001 | The module MUST be on if and only if `Config.Modules.Population` is non-nil, and with it off no event, state or log of this spec MUST exist and `Renewal` MUST keep the meaning of `spec-spawner.md` 7.1 (3.1). |
| FR-POP-002 | The world MUST compute the module's births and deaths once per region per tick in the entry-and-exit phase, after the `Privation` that P6.1 set and before `Decide`, regions in ascending `RegionID` order (`spec-regions.md` 3.3.2), roles in role order, agents in ID order (4.1). |
| FR-POP-003 | The signal of an agent MUST be its own welfare ratio (`spec-welfare-and-wages.md` 2.1, 2.2), and an agent whose ratio is absent MUST NOT be exposed, its households being counted as `NoRatio` (3.4). |
| FR-POP-004 | A schedule MUST be evaluated by integer piecewise-linear interpolation, flat outside the first and last knots, with the increment truncated toward zero and the products checked (3.5.1). |
| FR-POP-005 | The effective annual rate MUST be `min(10 000, trunc(schedule(ratio) * scale / 10 000))`, and a household whose rate is clamped MUST be counted as `RateClamped` (3.6). |
| FR-POP-006 | The exposed households of an agent MUST be `Weight - Unfed` for an agent that is not external and whose ratio exists, and 0 otherwise; an unfed household MUST NOT be subject to the schedule (3.7). |
| FR-POP-007 | The exposed households for births MUST exclude those of a `Closed` role and of a role whose `Needs` flag the region's class lacks, and the birth carry of such a role MUST NOT advance (3.7.3). |
| FR-POP-008 | For each agent with exposed households the module MUST add `H_a * d_a` to the agent's death carry, take `carry / D` whole households, keep `carry mod D`, and cut the households to `H_a`, counting a cut as `DeathsCut`, with `D = 10 000 * TicksPerYear` (3.8.1). |
| FR-POP-009 | An agent with no exposed households MUST keep its death carry unchanged (3.8.1). |
| FR-POP-010 | The initial death carry of every agent, an initial agent or a later one, MUST be `((id * 2 654 435 769) mod 2^32) * D / 2^32`, computed with the checked 128-bit helper, and MUST NOT use the random source (3.8.3). |
| FR-POP-011 | For each (region, role) the module MUST add the sum over the non-external exposed agents of `H_a * b_a` to the birth carry, take `carry / D` whole births and keep `carry mod D` (3.8.2). |
| FR-POP-012 | Over any run of ticks the number of deaths of an agent, and of births of a role, MUST equal `floor((carry at the start + sum of numerators) / D)`, within 1 of the expectation, independent of the weight (3.8.4). |
| FR-POP-013 | Deaths MUST be taken per agent in ID order and cut so that the group's households do not fall below `MinHouseholds`; the deaths not taken MUST be dropped, not deferred, and counted as `DeathsHeld` (4.4.1). |
| FR-POP-014 | Births MUST be allowed only while the households of the region, after the tick's deaths, stay at or below `HouseholdCap` when it is above 0; those beyond MUST be dropped and counted as `BirthsCapped` (4.4.2). |
| FR-POP-015 | The households taken from an agent by the module MUST NOT exceed its fed households after the ladder's reductions of the same tick (4.4.3). |
| FR-POP-016 | A birth MUST be an `Enter` with cause `births`; a death MUST be a `Reduce` with cause `population`, and a reduction to weight 0 MUST be a removal with `Cause` `population` (4.3, 4.7). |
| FR-POP-017 | Each born household MUST bring `trunc(Role.StartMoney * BirthStake / 10 000)` from the pool with the ledger reason `births`, whether it joins an existing agent or becomes a new one, and no goods; the funded number MUST be `min(n, floor(budget / stake))`, a stake of 0 being unlimited, and the unfunded MUST NOT be created or deferred (4.3). |
| FR-POP-018 | A death that leaves households in an agent MUST scale its money and each good to `floor(x * w' / w)` with the difference to the estate and the ledger reason `deaths` (4.7). |
| FR-POP-019 | In a region's plan the module's deaths MUST follow the ladder's reductions and removals and precede the budget of the entry step, and its births MUST be the last operation (4.8). |
| FR-POP-020 | While the module is on the effective `Renewal` MUST be 0, and validation MUST warn `renewal_with_population` only if the configured value is a non-default value above 0, reporting the informational `renewal_ignored` when it equals the default (4.6.1). |
| FR-POP-021 | A born household MUST be placed by the rule of `spec-spawner.md` 4.3 in the role of the households it grows from, and MUST have the newcomer's state of `spec-spawner.md` 7.4 (3.2). |
| FR-POP-022 | Every birth and death MUST conserve money and goods exactly, the module MUST have no faucet and no sink, and no household MUST be created except by a funded birth (3.2, 4.7). |
| FR-POP-023 | An external agent MUST NOT be exposed for births or deaths (3.7.2). |
| FR-POP-024 | A change of weight or of role MUST leave an agent's death carry unchanged (7). |
| FR-POP-025 | Every parameter of section 8 MUST be a field with the default and the bound stated, and a value outside its bound MUST be rejected with `ErrOutOfBounds` naming the path (8, 9.1). |
| FR-POP-026 | A schedule with no knots or more than `MaxKnots`, or with knot ratios not strictly increasing, MUST be rejected with `ErrOutOfBounds` (9.1). |
| FR-POP-027 | The rows of `Regions` MUST obey `spec-regions.md` 5.3: an unknown region is `ErrUnknownRegion`, a second row for a region `ErrDuplicateID`, absence is the default without a warning, and rows are in ascending `RegionID` order in the hash and the checkpoint (8, 9.1). |
| FR-POP-028 | The warnings `no_replacement_ratio`, `schedule_direction`, `replacement_above_closed_ceiling`, `population_gain`, `no_fixed_factor` and `population_without_upkeep` MUST be reported and MUST NOT fail validation (9.2). |
| FR-POP-029 | The replacement ratio MUST be the ratio at which `birth - death` changes sign from negative to positive, found on the union of the knots in integer arithmetic, and MUST be reported once at validation and in the snapshot (5.7). |
| FR-POP-030 | `population_gain` MUST be reported when `r* * dRate * T > 10 000 * dRatio * TicksPerYear` with `T` the longest effective window and `dRate / dRatio` the slope of `birth - death` at `r*` (5.4). |
| FR-POP-031 | `PopulationChanged` MUST be emitted once per region and role per tick after the region's plan, when any of `BirthsDue`, `DeathsDue`, `DeathsCut`, `DeathsHeld`, `BirthsCapped`, `BirthsCeiling`, `BirthsUnfunded`, `RateClamped`, `BirthsDone` or `DeathsDone` is non-zero, and MUST NOT otherwise, with the fields of 10.1 and no map (10.1). |
| FR-POP-032 | `DeathsDone` MUST equal the households removed with cause `population` and `BirthsDone` the households added with cause `births`, in the same tick (10.1). |
| FR-POP-033 | The `Warn`s "region extinct", "births unfunded" and "deaths held" MUST be bounded per region or role, emitted at the first tick and each tenth after, and carry the attributes of 10.3. |
| FR-POP-034 | The snapshot MUST report per region and role the mean birth and death rates and the natural increase, and the module's `r*` and gain (10.4). |
| FR-POP-035 | The death carries (in the agent record, `spec-checkpoint.md` 4.2 tag 3), the birth carries, the effective scales and the three diagnostic runs MUST be in the checkpoint in a fixed order, and a world restored between ticks MUST make the same births and deaths as an uninterrupted one (11). |
| FR-POP-036 | The module MUST draw nothing from the random source, and every ordering MUST end in a unique key and MUST NOT depend on map iteration (3.8.3, architecture.md 7.5, 7.6). |
| FR-POP-037 | The household count rebuilt from `AgentSpawned`, `WeightChanged` and `AgentBankrupted` MUST equal the sum of the weights in every tick with the module on (4.7). |
| FR-POP-038 | A region with no households MUST get no birth from the module and MUST emit the `Warn` "region extinct" (6.4). |
| FR-POP-039 | The effective `BirthScale` and `DeathScale` of a region MUST be the configured values unless a shock has set them, and MUST be read each tick (6.6). |
| FR-POP-040 | The household-weighted mean ratio in the snapshot MUST NOT be read by any rule of this module (3.4.3). |
| FR-POP-041 | The births of a role in a region MUST be capped at its headroom (the least of its class limit and its body's, less its households, counted at the start of the entry step), the births dropped MUST be counted in `BirthsCeiling` and not in `BirthsCapped`, and the cuts MUST apply in the order `HouseholdCap`, role ceiling, funding (4.4.4; `spec-settlements.md` FR-SET-023). (new) |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-POP-001 | With the default logger and observer, the computation, the carries and the executions of a step of the reference chain MUST allocate nothing beyond scratch held by the world (observability-and-logging.md 4.5, 8.2). |
| NFR-POP-002 | The cost of the computation for a region SHOULD be proportional to its agents and not to its weights. |
| NFR-POP-003 | Observation MUST NOT change results: a run with a logger and an observer attached MUST make the same births and deaths as one without. |
| NFR-POP-004 | The random stream of a world MUST NOT depend on whether the module is on. |

## 16. Prior art and evidence

**Clark's Malthusian model: read.** G. Clark, *A Farewell to Alms* (2007). Read directly:
chapter 2, "The Logic of the Malthusian Economy" (the author's chapter PDF, text
extracted, figures not seen), and passages of chapters 4 and 5 of the author's October
2006 draft of the whole book (<https://faculty.econ.ucdavis.edu/faculty/gclark/papers/FTA2006-1.pdf>,
draft pages 90 to 93 and 100 to 103). Chapter 3 (material living standards) and the
rest were not read; the published edition was not checked against the draft. What the
text says, and what this spec takes from it:

| Statement | Where | Use |
|-----------|-------|-----|
| Three assumptions: the birth rate is determined by custom and increases with living standards; the death rate declines as living standards rise; living standards decline as population increases | chapter 2 | the structure of 5.1 |
| In a stable population births equal deaths; the income at which they do is the "subsistence income", fixed by the two schedules alone and independent of production technology | chapter 2 | `r*` depends on the schedules alone (5.1) |
| Population above or below the equilibrium returns to it; a higher death-rate schedule at a given income raises equilibrium income and lowers equilibrium population, and conversely | chapter 2 | scenario POP-S2; the shock result of 13.1 |
| Diminishing returns on a fixed factor (land) give the third assumption | chapter 2 | 5.2: the fixed factor is RES's |
| Surviving children per woman close to 2 in every society before 1800 (1.93 to 2.10 in his Table 2.1 for six European populations, c. 1300 to c. 1800) | chapter 2 | none: not used as a default |
| "Gross fertility did not rise significantly with income": in a sample of English testators c. 1620 the richer (about ten times the assets) married slightly more (12.7% unmarried against 7.8%) and the gross fertility is judged only 2% to 5% higher; "it does not do too much violence to reality to simply assume a given birth rate" | draft chapter 4 | the default birth schedule is flat (3.5.3) |
| Across households, income was a strong predictor of mortality: household income explained 62% of the variation in infant mortality across eight London parishes; for the poorest groups the death rate exceeded the birth rate, for the richest it was well below | draft chapter 5 | the default death schedule falls with the ratio |
| In England 1540 to 1800 no association between national mortality and income appears, which Clark attributes to shifts of the death schedule over time (disease environment, urbanisation, sanitation) | draft chapter 5 | a shift of the whole schedule is a region scale (3.6) |

What this evidence does not give: **no usable rate**. The figures in the text (a death rate
of 33 per thousand for a life expectancy of 30, birth rates of 28 to 40 per thousand) are
presented as examples of the arithmetic or as observed values of some societies. They are
not copied into the core, and the defaults of section 8 are placeholders; examples may
use historical figures, labelled, and 12 does not. The claims are Clark's, and the
interpretation is contested (for example the role of fertility control); this spec
takes the shapes, not the conclusion that the whole of pre-industrial history was
Malthusian. The earlier "unverified" entry of `evidence.md` (the VoxEU summary) is
superseded for the facts above; the proposed rows are in the handoff.

**Other prior art.** bazaarBot replaces a bankrupt agent at once (`bazaarbot-reference.md`
defects 13 and 14); the library's `Renewal` does the same by default, and 4.6 turns it off
under this module. Lengnick's baseline model has a fixed number of households and no
births or deaths (`evidence.md`), so it offers no precedent for the loop.

## 17. Notes settled

| Note | Resolution |
|------|------------|
| N-008 | Applied for POP: the ledger reasons `births` and `deaths` and the estate path (4.7); the ladder's `starvation` is SPN's. Close |
| N-029 | Applied for POP: Clark's own source read (16); chapters 3 and the published edition not read. The evidence rows are in the handoff. Close |
| N-046 | POP's entries delivered (14). Leave open for the other specs |
| N-083 | Applied: `deaths` added (4.7), `Renewal` ignored while the module is on (4.6). Close |
| N-073 | Confirmed for births with one change: the stake is `BirthStake` of `StartMoney` (4.3) |
| N-016, N-017, N-027 | N-016: the rows of 8 are owed to the parameter-owner table; N-017: new terms in the handoff; N-027: every number here is a placeholder |

## 18. Open questions

1. **A maturity delay for births.** A pipeline of pending newborn households of length
   `MaturityTicks` would add the generation lag the model lacks (3.3) and would turn the
   approach of 5.4 into a cycle when `k * tau` reaches about 1. The cost is state per
   (region, role) and a second window. Next step: decide when the first scenario needs
   a cycle (STY).
2. **Per-role schedules.** A class with its own custom (a role with different fertility)
   would take `Role` overrides of the two schedules. Version 1 has the world's schedules
   and region scales. Next step: the first pack that needs it.
3. **A warm start at `N*`.** A warm start's census (`spec-viability.md` 8) is not the
   Malthusian equilibrium; a world with a fixed factor drifts toward `N*`. The solver
   could report `N*` for a given pair of schedules if it knew the fixed factor (RES, `spec-resources.md` 8.2) and the full-draw ratio `r_full`, which is no field (`spec-resources.md` 23 item 6; note N-110 proposed). RES is written; VIA still has to add it.
4. **Fertility response to relief.** Relief raises the ratio (`spec-estates.md` 2.6) and
   so raises births and lowers deaths (STB-L40 touches the migration signal; this is the
   same effect for population). Next step: an experiment.
5. **The stake from the parents.** The stake is paid from the pool because that is the
   only place money can come from without a rule about kinship. A rule that took it from
   the exposed households of the role in proportion to their money would avoid the
   money-limited population (5.5) in a prosperous region and would move money from the
   living. Next step: measure the pool in S14.
6. **Seasonal births and deaths.** The rates have no season. A seasonal pattern would be
   a table by season as for yields (`spec-production.md` 6.1). Not needed in version 1.
7. **Cohorts and the death carry.** A cohort of weight `w` removes households one at a
   time from the same agent; a second agent of the same role and ratio has a different
   carry start (3.8.3). Whether the Weyl start spreads well for very large numbers of
   agents is untested. Next step: a test with 10^5 agents.
8. **The shock call. (Answered.)** The calls are `SetBirthScales` and `SetDeathScales` of
   `spec-integration.md` 5.1, each over `RegionScale{Region; BP}`; the fields stay and each
   change is the event `PopulationScaled`.

9. **The margin of `no_fixed_factor`.** 5.2 and 9.2 say an endowment "can bind" when `N_bind` is below the census's households times a stated margin; the margin is not stated. Next step: POP with VIA, from `Headroom` (`spec-resources.md` 11.3).

10. **`Renewal` and its default.** Ignored while the module is on, informational at the
   default and a warning only at a non-default value above 0 (4.6.1). Whether the
   spawner should drop `Renewal` as a field when the module is on is left to SPN.

## 19. Where this lives in the code

| Concern | Home |
|---------|------|
| `PopulationConfig`, `Knot`, `RegionPopulation`, the `Modules.Population` pointer | root package, `config.go` (types) and `defaults.go`; bounds in `bounds.go` |
| Validation, the warnings and the diagnostic | root package, `validate.go`, `errors.go` |
| Pure rules: the schedule interpolation, the effective rate, the carry, the offset, the replacement ratio and the gain test | `internal/population` (stdlib only, over primitive slices and indexes, converted inline at the call, architecture.md 2.9) |
| The computation each tick, the views, the cap and the floor | root package, `world.go` and `spawner.go` (the view fields) |
| The event `PopulationChanged` | root package, `event.go` |
| Reason and cause constants (`births`, `deaths`, `population`) | root package, `event.go` (exported constants, `spec-observability.md` 4.2.3) |
| The log keys `births`, `deaths` (key strings only) | `internal/logkeys` |
| The ledger reasons | `internal/ledger` registry (`spec-ledger.md` 7.2) |

```go
// internal/population: stdlib only, no root types. A sketch, not final.
type Knot struct{ Ratio, Rate int64 }

func Rate(knots []Knot, ratio, scaleBP int64) (rate int64, clamped bool)
func Step(carry, exposed, rate, denom int64) (events, newCarry int64)     // 3.8
func Offset(id, denom int64) int64                                        // 3.8.3
func Replacement(birth, death []Knot) (ratio int64, ok bool)              // 5.7
func GainWarn(rstar, dRate, dRatio, window, ticksPerYear int64) bool      // 5.4
```
