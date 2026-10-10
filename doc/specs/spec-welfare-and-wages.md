# Spec: Welfare ratio and wages

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-WEL-NNN` and
`NFR-WEL-NNN`, which tests cite (testing.md section 4). They are in section 16.
Sections 11 to 16 were added in the third round of the specs (observability, parameters,
state, the register entries, the decisions taken and the requirements); sections 1 to 10 keep
their numbers.

**The numbers in this spec are placeholders** where they are not definitions: the
defaults of 4.2 and 12, the window of 2.3 and the worked examples (N-015, N-027).
No default is claimed to be stable.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)). The welfare ratio is a measure the core can compute. The wage formulas are calibration tools, and the target-income rule is an optional agent behaviour, off unless configured. None of them is engine behaviour.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

This spec covers how a standard of living turns into wages and behaviour:

- how the welfare ratio is defined and computed, including for agents with no
  history (section 2);
- how a daily wage relates to a target standard of living (section 3);
- how wages differ by skill, as an emergent market outcome rather than a
  configured multiplier, and how wages and premiums are measured (section 4);
- how days worked emerge from a target-income rule (section 5);
- in-kind payment (section 6) and a role economics check (section 7).

The principles shared by every spec (one unit of account, integer arithmetic,
a tick is the unit of time and a year is `TicksPerYear` ticks, data not code, wages are not configured) are in
`spec-currency.md` section 2. The reference basket `B_r` is in
`spec-currency.md` section 5. The barriers that limit entry to skilled roles are
in `spec-skill-barriers.md`, and carrier fees are in `spec-carriage.md`.

## 2. The welfare ratio

The **welfare ratio** compares what a household earns to what it must spend to
stay alive. It follows Allen's definition: a ratio of 1 earns just enough for
the baseline basket, and higher values leave more discretionary income.

### 2.1 Definition

Each agent keeps a ring buffer of per-tick records for the last `T` ticks (the
window of 2.3, which is derived per role). Record `t` holds `income_t`, `cost_t`, the
relief `F_t` and the grant `G_t` that are inside `income_t` (so that the earned ratio of
2.1.7 and the observed wage of 4.3 exist), the own-use receipt `K^own_t`, and
whether tick `t` was a working tick (4.3) with its working household-ticks and its
own-use household-ticks (section 13):

- `income_t` is the **profit** of the tick (as in bazaarBot's `profit`), in signed
  minor units:

  ```
  income_t = R_t + K_t + K^own_t + F_t + G_t - S_t(Input) - W_t
  ```

  `R_t` is the money the agent received from trades in tick `t`; `K_t` the value of
  its in-kind receipts (section 6); `K^own_t` the value of the units that an own-use job
  of the subsistence module created at its completion in tick `t`, at the prices of
  `cost_t` (`spec-subsistence.md` 7.1, 7.2; a household that makes exactly its upkeep reads
  10 000 at any price); `F_t` relief paid to it
  (`spec-estates.md` 2.6); `G_t` the money received under a grant to households that is not `NotIncome` (`spec-money-supply.md` 7.2), an unearned income like relief; `S_t(Input)` the money it spent in tick `t` on bids whose
  purpose is `Input` (`spec-agents.md` 6.3, 7.3.6); and `W_t` the **wear cost** of
  the tick, 2.1.4. Spending on upkeep (purpose `Upkeep`), on a tool (`Tool`) and on
  anything else (`Other`) is **not** netted: upkeep is the `cost_t` below, and a tool
  is capital, whose use is netted as wear. A starting-money grant (`start_money`,
  `initial`) is not income, and neither is a grant with `NotIncome`. A levy of the money-supply module is not netted from `income_t` in version 1 (`spec-money-supply.md` 7.2, open question 5). `income_t` may be negative (a tick of buying inputs and
  selling nothing).
- `cost_t` is the cost of one tick of the agent's **effective upkeep**
  (`spec-agents.md` 5.2: the role's list or the world's, plus the region's extra,
  times the weight), at the prices of the region the agent was in during tick `t`.
  This is the agent's own upkeep, not the region's reference basket.

Over a full window:

```
ratio_bp = (sum of income_t) * 10_000 / (sum of cost_t)
```

The result is in basis points: 10 000 is a ratio of 1.0, 25 000 is 2.5. It is
negative if the window's profit is negative.

2.1.1. While an agent is away on a trip (`spec-transport.md` 5.9: its account keeps its region, and `NoRegion` is reserved and unused in version 1, `spec-regions.md` 6.5.3), its upkeep is valued at the
prices of the region of its account, and its income for those ticks is zero
unless it receives some in kind. A carrier paid a fee on delivery has its
income recorded in the tick the fee is released. A migration journey of `k` ticks
(`spec-migration.md` 8) is recorded the same way: `k` records with no earned income and the
upkeep of the origin region, which are not working ticks (4.3); the settling rule of
`spec-migration.md` 5.2 keeps them from triggering a second move. They stay in the window
(`spec-migration.md` 8.6); N-099 stays open.

2.1.2. Records stay with the agent when it migrates, so a migrant's ratio
reflects its recent history across regions. Because all regions share one
currency, the sums are directly comparable.

2.1.3. **Why cash on inputs and accrual on tools.** Inputs are netted when bought,
not when consumed. A buffer bought in one tick is consumed in later ones, and over a
window the two differ only at its edges; cash is what the ledger records and needs no
valuation. A tool is netted when it **wears**, not when it is bought, because its
purchase is capital that lasts `Durability` uses (`spec-skill-barriers.md` 2.6 already
calls wear "a recipe cost"). Netting both the purchase and the wear would count a tool
twice. This answers N-050.

2.1.4. **The wear cost** of tick `t` is, for each execution started in the tick that
wears a tool, `ceil(WearPerUse * price / Durability)` at the region's market price of
the tool in that tick (the price of `spec-clearing.md` 5.4; the effective reference
price if it has never traded), summed over executions, with the executor's weight
applied as for any amount. It is the `wear_cost` of `spec-production.md` 9.1 evaluated
at the market price, and it uses no belief.

2.1.5. **A commodity that is both an input and upkeep** (the seed that is also eaten)
is classed as an input by the strategy (`spec-agents.md` 7.3.6), so the money spent on
it is netted as an input even for the part that is eaten, and the upkeep cost is also
in `cost_t`. The ratio of such a household is understated, by at most the value of the
upkeep it buys of that commodity. The bias is conservative: an understated ratio makes
the household work more and qualify for relief earlier, never less. It is accepted
rather than split, because an order carries one purpose; a per-order job share would
fix it and is open question 3 of section 15.

2.1.6. **A trainer** boarding a trainee consumes the trainee's upkeep from its own
stock (`spec-agents.md` 5.7, `spec-skill-barriers.md` 4.2). The value of those goods at
the region's prices is a cost of training and is netted from the trainer's income in
the tick of consumption, as an input would be. The trainee receives the same value as
an in-kind receipt (6.2), so its ratio stays at 1.0 while it is boarded.

2.1.7. **The earned ratio.** `earned_bp = sum(income_t - F_t) * 10 000 / sum(cost_t)` over the
same effective window with the same padding and truncation as the ratio of 2.1: the ratio with
the relief taken out of the income. A grant `G_t` stays in. It is a derived read from the
records and not a second record, absent when the ratio is absent (2.2.3), and is the figure
the migration module reads as its trigger (`spec-migration.md` 5.3), so that relief does not
suppress migration (`spec-migration.md` 5.4).

2.1.8. **Own-use receipts.** `K^own_t` is counted at the creation of the units and nothing is
counted when they are eaten, whose cost is in `cost_t` as for bought goods
(`spec-subsistence.md` 7.1). So a household fed by its floor reads a ratio near what the floor
covers, and relief and the schedules of `spec-population.md` see it (`spec-subsistence.md`
7.4). A sale of the surplus counts again as cash (`spec-subsistence.md` 7.5, open question 2
there).

### 2.2 Agents without a full history

A new agent (an initial agent, a spawned replacement or a newborn) has fewer
than `T` records. Missing ticks are **padded** with a default, so the ratio is
always defined and no special "absent" state is needed.

Let `n` be the number of missing ticks (`T` minus the records held), `U` the
agent's upkeep cost in the current tick, and `d_bp` its default ratio in basis
points. The padded values for the missing ticks are:

```
pad_income = n * U * d_bp / 10_000
pad_cost   = n * U
ratio_bp   = (sum of income_t + pad_income) * 10_000 / (sum of cost_t + pad_cost)
```

2.2.1. `d_bp` is the role's `DefaultRatio`, falling back to a global default of
10 000 (a ratio of exactly 1.0). A ratio of 1.0 is neutral: it neither makes
an agent migrate nor starve before real data arrives. A role may set a higher
default if its members should start confident.

2.2.2. As records accumulate, the padding shrinks and real data takes over. By
tick `T` the agent has no padding left. A side effect is that a newborn in a
collapsing region reads as neutral for a while. That delays its migration
signal, and it is intended: a new agent has not had time to learn conditions.

2.2.3. If an agent has no upkeep (`cost_t` and `U` are zero), the ratio is
undefined and is reported as absent, not as zero or infinity.

### 2.3 Window and arithmetic

2.3.1. The world's window `WelfareWindow` is a configuration value, long enough to
smooth days without work (section 5). It defaults to 30 ticks (a placeholder). It has
a row in the parameter-owner table (N-016), owner this spec, bound 1 to 1 000
(provisional, `spec-config.md` 3.7). Its name is `World.WelfareWindow`; "the window `T`"
of the formulas is the effective window of 2.3.3.

2.3.2. The multiplications are overflow-checked (architecture.md 3.5). A
window or income that would overflow is a configuration error caught at
validation, not a runtime wrap. Rounding is truncation toward zero.

2.3.3. **The window of a role is derived, and is never shorter than its longest job
(N-032).** A job consumes its inputs when it starts and its output is sold some time
after it ends (`spec-production.md` 5.2.3), so a role whose recipe lasts as long as the
window shows costs and no income, and a healthy producer would read as starving. That
would trigger relief, migration and bankruptcy for a farm in the middle of a crop. The
effective window `T` of an agent is

```
T = max( WelfareWindow,  WindowCycles * L )
```

where `L` is the longest of: the `Duration` of the recipes on its role's list; the `Duration`
of the recipes on its role's effective own-use list (`spec-subsistence.md` 3.3, so the window
holds the lump of an own-use job); and, for a role with a merchant block, `Kmax`, the longest
travel time over the configured graph of the self-carried modes of that block
(`spec-merchants.md` 12.3, a trip being the merchant's job). `WindowCycles` is a world
parameter, default 2 (a placeholder), bound 1 to 100. A role
with none of these, or only short ones, has the world's window. The window is **derived,
not configured per role**, so an author cannot set it wrong; it appears in the
`Snapshot` as the role's window.

Why this and not the alternative. The other fix considered was to **attribute income to
the job it came from**, spreading a job's net value over its ticks. It was rejected:
output is sold after the job ends at a price nobody knows when the job starts, so
attribution needs either a valuation of work in progress by beliefs or a rewrite of
past records, and both put a guess in a measure that is meant to be measured. A longer
window needs neither, and a window of two cycles contains two starts and two completions
whatever the phase of the job within it, which removes the saw-tooth that a window of
about one and a half cycles would give. The cost is slower response: the ratio of a
long-job role reacts over `T` ticks to the migration trigger, to relief and to the
target-income rule, which only decides when the agent is free. That slowness is real
(a farm cannot answer a bad price faster than a crop) and is accepted. **Credit is the
real remedy for a long job** (`spec-credit.md`); version 1 has none. The record is
STB-L19 and STB-L20, scenario S11.

2.3.4. The effective window is at most `MaxWelfareWindow` (3 650 ticks, a provisional
bound of `spec-config.md` 4.1, ten Earth years of daily ticks); a configuration in
which `WindowCycles` times the longest duration of a role (`L` of 2.3.3) exceeds it is rejected with
`ErrOutOfBounds`. State is the integers of section 13 per tick of the window per agent,
kept with running sums, so the cost of a ratio is constant (NFR-WEL-001).

2.3.5. The warning `job_longer_than_window` of `spec-production.md` 10.4 cannot occur
under 2.3.3 (the window always exceeds the longest job). It is replaced by the
informational `window_extended` of 11.4, which says that a role's window is
longer than the world's, and why. For a role with a merchant block the window also
exceeds its longest trip, so the warning `trip_longer_than_window` of `spec-transport.md`
12.2 stays for roles with no block only.


### 2.4 Uses

The welfare ratio is an output and a decision input (for example the trigger
for migration, and the target-income rule in section 5). It is not a
simulation constraint: nothing clamps an agent's income to keep the ratio
above 1.

## 3. Wages and the target standard of living

A **daily wage** `W` is the money earned on one working tick. This section
relates it to the welfare ratio. It is used to calibrate a chain and to predict
steady-state behaviour. It does not set any wage in the running simulation
(`spec-currency.md` section 2.6).

### 3.1 Derivation

Over a year the worker needs `B_r * D_year` to subsist and earns
`W * D_work`. Setting the welfare ratio `r` equal to income over need:

```
r = (W * D_work) / (B_r * D_year)
```

Solved for the wage, in integer form with `r` in basis points and
multiplication before division so truncation happens only once:

```
W = (r_bp * B_r * D_year) / (10_000 * D_work)
```

Solved for the ratio a given wage produces:

```
r_bp = (W * D_work * 10_000) / (B_r * D_year)
```

Rounding is truncation toward zero. `D_year` is the world's `TicksPerYear`
(derived from the calendar, `spec-calendar.md` section 3.2; 365 at the Earth default). `D_work` is the number of ticks
worked per year: for calibration and prediction it is the role's
`ExpectedWorkTicks` (section 4.2). `B_r` is the basket of the worker's region, so
the same target ratio gives a different nominal wage in each region (`spec-currency.md` section 5.6).

### 3.2 Worked examples

`B_r = 1 000` minor units, `D_year = 365`. Annual income is `r * 365 * B_r` regardless of
`D_work`, for example 730 000 minor units at `r = 2.0`.

| Welfare ratio `r` | `W` at 250 working days | `W` at 150 working days |
|-------------------|-------------------------|-------------------------|
| 1.0 (10 000 bp)   | 1 460                   | 2 433                   |
| 1.5 (15 000 bp)   | 2 190                   | 3 650                   |
| 2.0 (20 000 bp)   | 2 920                   | 4 866                   |
| 3.0 (30 000 bp)   | 4 380                   | 7 300                   |

Arithmetic for the first row: at 250 days, `10 000 * 1 000 * 365 /
(10 000 * 250) = 1 460`. At 150 days the exact value is 2 433.33, truncated to
2 433. Because of truncation, `W * 150` is slightly below the exact annual
figure; the error is under one minor unit per working day.

### 3.3 How many days a worker works, historically

The conventional value for `D_work` in the literature is 250 (Allen, a
five-day week with two weeks' holiday). It is an assumption, and contested.
Humphries and Weisdorf estimate under 150 days a year for medieval day
labourers, and over 300 during the Industrial Revolution. Annual salaries in
1361-62 were paid for as few as 21 working weeks (105 to 126 days).

`D_work` **MUST** be a parameter of any calculation in this section, never a
constant in code: it is the role's `ExpectedWorkTicks` (section 4.2), whose
default is the 250 above, and different roles may differ, as the evidence does.
The running simulation does not impose it: the days an agent works emerge from
the target-income rule (section 5). `ExpectedWorkTicks` is the expectation used
where a number is needed before there is history to measure: calibration
(sections 7 and 8) and the entry rule of `spec-skill-barriers.md` section 2.5.
A well-calibrated chain has a measured working time near it (section 5).

## 4. Skill and market wages

### 4.1 No configured premium

Skilled roles earn more per working day than unskilled ones, but this spec does
not configure by how much. The difference is a market outcome:

- A self-employed role (example roles: farmer, blacksmith, refiner) earns the net revenue of
  its recipes: the price of its output times the quantity it sells, less the
  inputs it buys. Skill appears as a recipe that turns cheap inputs into a
  dearer output, and as the scarcity of agents who can run it.
- A carrier earns the fee a merchant pays for a delivery (`spec-carriage.md` section 2).
- The price of each good, and so each role's income, comes from the market:
  supply, demand and the agents' price beliefs.

There is no `SkillPremium` field, and no wage multiplier anywhere in the
configuration.

### 4.2 Role parameters

| Scope | Field          | Meaning                                                     | Default |
|-------|----------------|-------------------------------------------------------------|---------|
| Role  | `Unskilled`    | marks a role as the baseline for measuring skill premiums (section 4.3) | false |
| Role  | `TargetRatio`  | the welfare ratio the role's agents aim for (section 5), in basis points | 15 000 |
| Role  | `TargetSpread` | spread of per-agent targets around `TargetRatio`, in basis points (section 5.1) | 0 |
| Role  | `DefaultRatio` | starting welfare ratio for agents without history (section 2.2) | 10 000 |
| Role  | `InKind`       | share of income received as upkeep goods, in basis points (section 6) | 0 |
| Role  | `ExpectedWorkTicks` | ticks the role is expected to work in a year (`D_work`, section 3.3); at least 1 and at most `TicksPerYear`. A value above the derived `TicksPerYear` (the default 250 with a calendar of fewer ticks a year) is an error, `ErrOutOfBounds` carrying `TicksPerYear`; it is never scaled silently (`spec-config.md` 13.5, FR-CFG-048) | 250 |
| Role  | `MinWorkShare` | the least share of the window's ticks that must be working ticks, in basis points; 0 turns the floor off (section 5.3) | 4 000 (a placeholder) |

World-level parameters of this spec (a row each in the parameter-owner table):

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| World | `WelfareWindow` | the world window `T` of 2.3.1, in ticks | 30 | 1 to 1 000 (provisional, `spec-config.md` 3.7) |
| World | `WindowCycles` | how many of its longest job a role's window must hold (2.3.3) | 2 | 1 to 100 |

All are plain data in the root package's config structs. None of them is a
wage. The parameters owned by the other specs are indexed in the README. The role's
`TargetRatio` is a goal of the household, not a wage; the rest band `RestBand` that
the target-income rule uses is owned by `spec-agents.md` 12.

### 4.3 Measuring wages and premiums

The simulation measures what each role earns so that results can be checked and
reported.

A tick is a **working tick** for an agent in these cases:

- the agent's role has recipes: it **had a job in progress in the tick** (started,
  continued or completed, `spec-production.md` 5.2.1; an own-use job of the subsistence
  module is such a job), or it was in transit on a
  commercial trip. `RecipeExecuted` comes once, at the end of a job, so the root reads
  working ticks from the agent's job state and not from that event (N-035);
- the agent's role has no recipes (a trader or a carrier): it bought or sold
  in the tick (the `TradeExecuted` event), or it was in transit on a
  commercial trip, or it had an own-use job in progress (`spec-subsistence.md` 7.6; a
  role with no recipes may have an own-use list, 3.3.3 there).

A **commercial trip** is travel carrying the agent's own goods to sell (a self-carried shipment of `spec-transport.md`). Its working ticks are the ticks from the dispatch tick to the tick before arrival, read from the shipment record (`spec-transport.md` 5.9, FR-TRN-021). A carrier that moves other people's goods is a producer of haul units, whose job is an ordinary recipe (`spec-transport.md` 8). Migration travel, ticks spent resting
under the target-income rule, and ticks with no activity are not working
ticks. Whether a role "has recipes" is read from the configuration, so no code
branches on a role identifier (`spec-currency.md` section 2.5).

Every tick of a self-carried trip is a working tick, so the observed wage over a trip is the income of the trip over its ticks.
Over the window `T`:

```
observed_wage = (sum of (income_t - F_t - G_t - K^own_t)) / (number of working household-ticks - number of own-use household-ticks)
```

Relief `F_t` and a grant `G_t` are excluded: they are not earned by working (`spec-estates.md` 2.6; `spec-money-supply.md` 7.2). The
own-use receipt `K^own_t` and the own-use household-ticks are excluded too: the wage is the return
to market labour, and an own-use job is not market labour (`spec-subsistence.md` 7.3, which
gives the reason and the bias left, its open question 2). An own-use tick **is** a working tick for the
work floor of 5.3 and for the target-income rule; it is left out of the wage only. A
**working household-tick** is one household working for one tick: for an agent of
weight `w` that has a job of `k` executions in progress it is `k` (the other
`w - k` households rest, `spec-agents.md` 9.4), and for a role without recipes it is
the weight when the agent traded or travelled. The **own-use household-ticks** of a tick are `k`
for an own-use job of `k` executions in progress, and are part of the working household-ticks.
So the observed wage is per working household, and does not depend on the weight (FR-SCL-009). An agent that earns while
resting under the target-income rule is not credited with working ticks.

The same formula, read over one agent's own effective window, is the **observed wage of the
agent**: the figure the migration module compares with an estimate of a destination
(`spec-migration.md` 6.3.2). It is a derived read from the records of that agent, per
household, and is absent when the agent had no working household-tick outside own-use in its
window. The formula per role and region is the sum over its agents' records.

A merchant has no recipes. Its purchases for a planned trip (the cargo, the haul units and the
feed) are bids with the purpose `Input` and are netted, so the income of a trip is its margin
(`spec-merchants.md` 3.4.2).

The **observed premium** of a role in a region is its observed wage divided by
the observed wage of the region's unskilled agents, in basis points:

```
premium_bp = role_wage * 10_000 / unskilled_wage
```

`unskilled_wage` is the mean observed wage of the agents of roles marked
`Unskilled`, weighted by **households** (the sum of the weights, so that a cohort
counts as the households it stands for, FR-SCL-015; the earlier "agent count" would
count a cohort of 40 as one). If the region has no such agents, or the
role has no working ticks, the premium is absent. Values are exposed per role
and region in `Snapshot` and are bounded by the configuration (observability
rule 5.4.1).

The observed wage of a role in a region is the **occupation signal** of the spawner: `spec-spawner.md`
6.1 reads it as `Wage` (per working household-tick, relief excluded) with `HasWage` false when
nobody of the role worked in the window, and ranks roles by it (`E`, `V`). Nothing here changes for
it; two consequences are stated so that they are not found later. The wage is a window mean over the
effective window of 2.3.3, so the signal lags by about half a window (`spec-spawner.md` 19, question 6),
and a role that sells only to wants (`spec-demand.md`) shows a wage only in the ticks its households
work, which the target-income rule makes fewer when the wage is high (5).
## 5. Days worked emerge from the target-income rule

Each tick an agent that is free decides whether to work. Under the
**target-income rule**, an agent works while its windowed welfare ratio (section 2) is
below its `TargetRatio`, and rests when it is at or above, with a band above the
target (5.1) so that it does not chatter. The rule is defined here and applied by the
default `Strategy` as the **work share** of `spec-agents.md` 7.1.2:

```
T_lo = TargetRatio of the agent                       (its own, after the spread of 5.1)
T_hi = T_lo * (10_000 + RestBand) / 10_000
share = 10_000                                        if ratio < T_lo
share = 0                                             if ratio >= T_hi
share = (T_hi - ratio) * 10_000 / (T_hi - T_lo)       between
executions requested = ceil(weight * share / 10_000)
```

`RestBand` is a world parameter owned by `spec-agents.md` 12 (default 500, a
placeholder). With `RestBand` 0 the rule is exactly "work below the target, rest at or
above it". A household (weight 1) works whenever the share is above 0, so for it the
rule is a threshold with a band; a cohort works in proportion, so its labour supply
changes in steps of one household and not all at once (`spec-agents.md` 9.3). The
agent's decision is taken only when it is free: a job once started runs to its end
(`spec-production.md` 5.2.2). Two rules override the share (5.3): the survival rule of
`spec-agents.md` 7.1.3 and the floor `MinWorkShare`. This answers what an earlier
version of this section left to the strategy. This spec states the consequence the
rule is designed to produce.

In steady state, with an observed wage `W`, the number of days worked per year
is the smallest whole number that reaches the target, capped at the year (this
is the emergent value that the role's `ExpectedWorkTicks` should be near; it is
not itself configuration). The formula is exact at `RestBand` 0; with a band the
household works until its ratio is `RestBand` above the target, so the days are
slightly more than the formula gives (by about the band's share), and a test of the
closed form uses `RestBand` 0:

```
D_work = min(D_year, ceil(r_bp * B_r * D_year / (10_000 * W)))
```

This is the wage formula in section 3.1 solved for `D_work`, rounded up so the
target is reached, not just approached. If `W` is too low to reach the target
even working every day, the agent works every tick and its ratio is
`W * 10_000 / B_r` basis points, below target.

Illustrative values for `B_r = 1 000` and a target ratio of 1.5:

| Observed daily wage `W`       | Days worked per year | Resulting ratio |
|-------------------------------|----------------------|-----------------|
| 1 000                         | 365 (capped)         | 1.00            |
| 1 500                         | 365                  | 1.50            |
| 2 190                         | 250                  | 1.50            |
| 3 285                         | 167                  | 1.50            |
| 4 380                         | 125                  | 1.50            |
| 6 570                         | 84                   | 1.50            |

Arithmetic: the numerator is `15 000 * 1 000 * 365 / 10 000 = 547 500`, so days
= `ceil(547 500 / W)`: at 2 190 that is exactly 250, at 3 285 it is 166.67,
rounded up to 167. A higher wage means fewer days worked at the same ratio: the
backward-bending labour supply described by Dyer for post-plague England
(`evidence.md`).

Roles may have different targets. A skilled role with a higher `TargetRatio`
works more days at a given wage than one with a lower target.

### 5.1 One target or many

`TargetRatio` is one value per role. An optional `TargetSpread` (basis points,
default 0) gives each agent its own target, drawn once when the agent is
created, uniformly between `TargetRatio * (10_000 - spread) / 10_000` and
`TargetRatio * (10_000 + spread) / 10_000`, from the world's random source.
One draw is made for every agent at creation, whatever the spread, in
agent-creation order. Drawing even when the spread is 0 keeps the random stream
identical across spread values, so an experiment can compare runs that differ
only in the spread (a paired comparison). The drawn target is stored with the
agent and appears in `Snapshot`. Validation requires the lowest possible target
to be at least 10 000 (a ratio of 1.0): an agent never aims below subsistence.

The choice has these effects:

| Concern | One value per role (spread 0) | Many (spread above 0) |
|---------|-------------------------------|-----------------------|
| Aggregate labour supply | All agents of a role cross the threshold together. The role works and rests in lockstep, so supply moves in steps | The marginal worker changes continuously, so days worked respond smoothly to the wage |
| Price stability | Lumpy supply swings prices, which swing wages, and can set up limit cycles (bazaarBot already oscillates) | Spread damps the swings |
| Calibration and tests | The closed form in section 5 holds exactly for the whole role | The formula holds per agent. The role's total is a mean over draws, so a test uses a fixed seed and a tolerance, or spread 0 for exact checks |
| Randomness | One draw per agent at creation, unused when the spread is 0 | The same draw, now used. The stream does not depend on the spread, but golden output changes when a spread is configured |
| Inequality inside a role | None from this source | Low-target agents rest more and earn and save less |
| Survival | Targets are at least 1.0, so below subsistence every agent works every day | Same, given the validation above |

Two further points hold either way:

- **Migration does not depend on the target.** The trigger is the absolute
  ratio against subsistence (the earned ratio of 2.1.7) and the household's estimate of a
  destination's income from rumoured prices (`spec-migration.md` 4, 6),
  so an ambitious agent below its aspiration does not migrate merely for that. (Version 1 of `spec-information.md` rumours prices only; a rumoured ratio is not used, `spec-migration.md` 4.4.1.)
- **Chattering.** A rule "work below target, rest at or above" switches every
  time the ratio crosses the target. The band `RestBand` above the target avoids it
  (STB-L15, damper D09): the rule is now part of the definition above and is not left
  to the strategy. A band of 0 restores the plain threshold.

### 5.2 Experiments with the spread

The spread is meant to be experimented with, so it is configurable at two
levels:

- **Per role**, in configuration: the `TargetSpread` field (section 4.2).
- **For the whole world**, as an option: `WithTargetSpread(bp)` overrides every
  role's spread. It follows architecture.md 6.2: it validates its input and
  returns an error rather than panic. The options struct records whether the
  option was given, so "not given" (use the configuration) is distinct from
  "given as 0" (no spread anywhere). A golden test that depends on it passes it
  explicitly.

A sweep runs the same chain and seeds at several spreads (for example 0, 500,
1 500 and 3 000) over many seeds. Because the draw is made whatever the spread
(section 5.1), the runs are paired. The measures to compare, per role:

- the coefficient of variation of units produced per tick (supply lumpiness);
- price volatility of the role's output;
- the distribution of days worked, and the dispersion of welfare ratios;
- the number of work and rest cycles per role, and time to steady state.

The default stays 0 until an experiment gives a reason to change it. The
outcome and the chosen default are recorded in the PR that changes it.

### 5.3 A floor on working ticks (D13)

**Decision (N-028): adopt a floor, as the tunable `MinWorkShare`.** A role has
`MinWorkShare`, in basis points, the least share of the ticks of an agent's window
(2.3.3) that must be working ticks (4.3). The rule: if the working ticks in the agent's
window are fewer than `MinWorkShare * T / 10_000`, the work share of section 5 is
10 000 whatever the ratio is, so the agent starts a job when it is free. Own-use ticks count
as working ticks here (4.3, `spec-subsistence.md` 7.3). 0 turns the
floor off. The default is 4 000, **a placeholder**: about 146 ticks of a 365-tick year,
which is the same order as the lowest figure in the evidence (labourers under 150 days a
year in the medieval period, Humphries and Weisdorf, `evidence.md`) but is not derived
from it. Validation requires `MinWorkShare * TicksPerYear / 10_000` to be at most
`ExpectedWorkTicks` (a floor above the expected work would make the target moot).

It is expressed as a share of the window, and not as the working ticks per year that
the register proposed (`MinWorkTicks`, STB-D13), because the ring buffer of 2.1 already
counts working ticks over the window, so no state is added; a count over a year would
need a year of flags per agent.

Why adopt it. `spec-stability.md` 6.1 shows that labour supply under the rule is
unit-elastic in the wage, that the labour loop is stable only if the demand for the
good has elasticity above 1, and that for a staple (elasticity between 0 and 1,
`spec-agents.md` 11.6) every run ends pinned: at the price ceiling after a rise, or at
the price where all households work the whole year after a fall. A floor replaces the
ceiling as the stop in the first case by a price set by a parameter, in the model
`P0 * (S0 / S_floor)^(1/e)`: with an expected work of 250 ticks, the default floor of
about 146 and `e` = 0.5, about 2.9 times the reference price, against the ceiling's ten
times. That is a stylised-model result without income feedback, not a prediction.

What it does not do. It does **not** remove the instability: the interior equilibrium
stays a repeller and the market is still pinned, at a milder price. It also weakens the
claim that days worked are purely emergent: below the floor they are customary. The
survival rule of `spec-agents.md` 7.1.3 is a different rule, which `spec-stability.md`
6.1 finds is unit-elastic too and so not a damper by itself; it stays as a household
rule. The alternative, to reject the floor and accept the pinned outcome at the
ceiling, was considered and refused: the ceiling is ten times the reference price, a
famine price by any reading, and the vision asks for bounded, plausible prices
(`vision.md` 3). Setting `MinWorkShare` to 0 restores the unfloored rule, and scenario
S04 should record both.

The floor acts silently (a household works though its ratio says rest), so by
FR-STB-013 it is counted: the `Snapshot` reports, per region and role, the households
held to work by the floor in the last tick, and a `Warn` "work floor binding" is
emitted once per region and role when more than half of the role's households have been
held by it for 10 consecutive ticks, at the tenth tick and every tenth after (12).

### 5.4 Stability of the labour loop (STB-L14, L15, N-028)

The rule makes labour supply unit-elastic in the wage (`D = K / W`, 5). A shortage of a
staple raises its price, the households reach their target in fewer days, supply falls
and the shortage deepens. The loop is stable only if the demand elasticity of the good
exceeds 1 at the equilibrium (`spec-stability.md` 6.1; the slope of excess demand is
`(1 - e) Q / p`). The default strategy gives a staple a demand elasticity between 0 and
1 (`spec-agents.md` 11.6), so the condition fails for it. What this spec provides is
the band and the proportional share (5, STB-D09 and D28), which smooth the steps, the
floor (5.3, D13), which sets where a pinned market stops, and `TargetSpread` (5.1,
D08). **None of them is claimed to stabilise the loop.** The test is scenario S04: a
staple role of self-employed households with a shortage injected, asserting that the
price stays within its bounds and that the role ends in a stated condition, with the
measured elasticity recorded. A pack author whose staple role uses the rule should
check the staple's demand elasticity, and should expect a pinned outcome if it is below
1.

## 6. In-kind payment (optional)

6.1. A role with `InKind = k_bp` receives `income * k_bp / 10_000` of its
income as upkeep goods, valued at the prices of its region, and the rest in
cash.

6.2. In-kind receipts count in `income_t` at the value they have in the
agent's region (section 2.1), so replacing a fixed amount of cash with an
equally valued amount of goods does not change the welfare ratio. The agent's
upkeep need is reduced by what it already holds in goods, as for any
inventory.

6.3. **A trainee's board** is an in-kind receipt: the goods its trainer's account
supplies for its upkeep (`spec-agents.md` 5.7, `spec-skill-barriers.md` 4.2), valued at
the region's prices, are `K_t` of the trainee, and the same value is netted from the
trainer's income (2.1.6). A boarded trainee's ratio is therefore 10 000.

## 7. Role economics check

For a self-employed role to reach its target, its output must cover the income
it targets plus its inputs on the days it works. With `W_target` the wage that
section 3.1 gives for the role's target ratio and expected working days:

```
net_revenue_per_working_day >= W_target
```

Calibration (section 8) checks this for each role.

7.1. **It is a report, not an error (N-054).** The check never fails validation or
world construction. It is a calibration aid that produces a number per role
(`WageBelowTarget` in the solver's report, `spec-viability.md` 10.3). The reason is an
identity. In a closed economy (no money faucet or sink, `spec-ledger.md` 7.3) in which
households buy nothing but their upkeep, household money is conserved, so total income
equals total upkeep spending and the household-weighted mean welfare ratio is **1**
whatever `TargetRatio` is (`spec-viability.md` 4.7). A target above 1 cannot then be met
by every role, the check fails for roles on average, and the target-income rule never
rests (every household works every tick, `spec-agents.md` 11.7), producing a surplus
that is bounded only by the production halt. Neither is an error of the pack.

7.2. **What the default `TargetRatio` means in that world.** The default of 15 000
(4.2) is kept; this spec does not change it, and the decision is now taken (the open
question 1 of section 15 is answered, 15.2): it stays 15 000 with the demand module on and off.
Without spending beyond upkeep (the demand module off) and without money from a boundary
(`spec-external-sector.md`) or from a configured grant that is not matched by a levy (`spec-money-supply.md` 9.1), a `TargetRatio` above 10 000 is an **aspiration** that the
economy as a whole cannot meet: the households of a role work as the rule says and
the ratio they reach is whatever the money in circulation allows. A pack that wants
roles to rest in a closed upkeep-only world sets targets at 10 000, the only value
the whole population can meet on average (roles above 1 rest, roles below work, and
the mean is 1); `target_above_surplus` (`spec-viability.md` 10.2) warns when the mean
target is physically out of reach. The default does not fall to 10 000 when the
module is off: lowering it would hide the missing demand and change every example, and a pack
that wants roles to rest in an upkeep-only world sets its targets itself. With the module on, the
ceiling of the mean ratio is `1 + Wref / U` (7.3), and the default stays an aspiration above it.

7.3. **With demand beyond upkeep** (`spec-demand.md` on) the check has its original meaning, and
this is what it means. Two statements, one per role and one for the world:

- **Per role:** at the solved prices the role's net revenue per working day, from its sales to the
  upkeep and the wants of the households (the sellers' income of `spec-demand.md` 7.1), reaches
  `W_target`. The wants make sales to other roles possible, so a luxury role's revenue is
  positive where, with upkeep alone, it had no buyers (the second row of `spec-demand.md` 7.2).
- **For the world:** the household-weighted mean target cannot exceed the mean ratio that the
  wants allow, `1 + Wref / U` at the reference prices and at most `1 + Wmax / U`
  (`spec-demand.md` 7.1). A mean target above the first is the validation warning
  `target_above_demand` (`spec-demand.md` 10.2), owned by the demand module: the roles do not
  all rest, as in 7.1, but now the bound is the pack's own wants and not 1. Roles above the mean rest, roles below
  work, and the mean is whatever the wants and the money in circulation allow.

The check is evaluated for each role at the solved prices (`spec-viability.md`) and again on a
run. The solver does not yet include the wants (`spec-viability.md` open question 3,
`spec-demand.md` open question 3), so until it does the report counts them as a fixed extra
demand at the reference prices, as the prototype of `spec-reference-chain.md` 4.4.6 does.

## 8. Calibrating wages

Given a new supply chain, an author:

1. marks the baseline roles `Unskilled` and sets each role's `TargetRatio`,
   `DefaultRatio` and `InKind` (section 4.2);
2. uses section 3.1 to find the daily income each role needs, and checks the
   recipe economics allow it (section 7, a report and not an error, 7.1);
3. runs the chain to steady state and checks the measured welfare ratio
   (section 2), the days worked (section 5) and the observed premium
   (section 4.3) against the targets, and against the premium checks in
   `spec-skill-barriers.md` section 3.

The reference basket and prices are calibrated in `spec-currency.md` section 5,
and tools and training in `spec-skill-barriers.md` section 6.

Historical ratios are targets, not facts about the simulation (see
`evidence.md`): about 1 for bare subsistence, 1.5 to 3 for building labourers
in 1700s England, and up to about 4 for the best-off households.

## 9. Where this lives in the code

| Concern | Home |
|---------|------|
| Welfare ratio and its window (section 2) | computed by the root from ledger totals, the per-tick receipts and spending by purpose of `spec-agents.md` (FR-AGT-057) and a per-agent ring buffer with running sums; reported through `Snapshot`, never logged as float |
| The pure arithmetic: the ratio, the padding, the wage and ratio formulas of 3.1, the closed form of 5, the work share and the floor | `internal/welfare` (stdlib only), over primitive slices and integers, converted inline at the call (architecture.md 2.9); named for what it provides |
| Observed wage and premium (section 4.3) | computed by the root; reported in `Snapshot` per role and region |
| Calibration helper (sections 3.1, 5) | used by config tooling and tests; the running simulation does not depend on it |
| Role parameters (section 4.2) | plain structs in the root package; parsed by `config/` |

## 10. Non-goals and future work

- Military roles and their pay. The 1344 pay scale in `evidence.md` is kept as a
  note for a later feature.
- Employers other than merchants hiring carriers, treasuries and taxation.
- Wages for employees working in someone else's recipe (general wage labour).

## 11. Observability

11.1. **Events.** This spec adds none. The welfare ratio, the observed wage and the
premium are measures, read from the `Snapshot`, not events (observability-and-logging.md
2). Relief is `ReliefPaid` (`spec-estates.md` 2.6).

11.2. **Snapshot.** Reports, per region and role: the observed wage and the observed
premium (4.3, absent as stated there), the role's effective window (2.3.3), the
households held to work by the floor in the last tick (5.3), and the **household-weighted mean
welfare ratio** of the role (the sum over its agents of weight times ratio over the sum of weights
of the agents whose ratio exists; absent if none), and the same mean over all roles of the region.
The mean ratio is what the scenarios S19 and S24 of `spec-reference-chain.md` (section 9) and the
closed-economy check of `spec-demand.md` 7 measure; it needs no event and is computed from the
records of 2, so it adds no state. Per agent, in a query
(`spec-integration.md`): the welfare ratio and whether it exists, its target ratio
(after the spread), the earned ratio beside the ratio (2.1.7), and the working ticks in its window. All are bounded by the
configuration or are per agent in a query (observability-and-logging.md 5.4.1).

11.3. **Logs.** One `Warn`, bounded by the configuration (observability-and-logging.md
3.2.2): "work floor binding", per region and role, never per agent, emitted when more
than half of the role's households have been held to work by `MinWorkShare` for 10
consecutive ticks, at the tenth tick and every tenth after, with `region`, `role` and
`households` (the number held). It is actionable: a role that works because of the
floor and not its target is a pinned labour market (STB-L14). The count behind it
makes the floor, a damper that acts silently, observable (FR-STB-013, N-045). The
run of consecutive ticks is diagnostic state (13). No `Info`, no `Error`, and nothing
per agent above `Debug`.

11.4. **Informational diagnostic** `window_extended`: reported once per role at
validation, not a warning of the library's `Warn` level, naming the role, its effective
window and what fixed it: a recipe, an own-use recipe or the longest trip (2.3.3, 2.3.5). It never fails validation.

11.5. With the default no-op logger and observer the welfare record of a tick, the
running sums and the work-share arithmetic allocate nothing (NFR-WEL-001).

## 12. Parameters and validation

12.1. Parameters owned by this spec, with defaults that are **placeholders**
(N-015, N-027):

| Scope | Field | Default | Bound |
|-------|-------|---------|-------|
| World | `WelfareWindow` | 30 | 1 to 1 000 (provisional, `spec-config.md` 3.7) |
| World | `WindowCycles` | 2 | 1 to 100 |
| Role | `Unskilled` | false | |
| Role | `TargetRatio` | 15 000 | 10 000 to `MaxBasisPoints` |
| Role | `TargetSpread` | 0 | 0 to 9 999, and the lowest target `TargetRatio * (10 000 - spread) / 10 000` at least 10 000 |
| Role | `DefaultRatio` | 10 000 | 0 to `MaxBasisPoints` |
| Role | `InKind` | 0 | 0 to 10 000 |
| Role | `ExpectedWorkTicks` | 250 | 1 to the derived `TicksPerYear` |
| Role | `MinWorkShare` | 4 000 | 0 to 10 000, and at most `ExpectedWorkTicks * 10 000 / TicksPerYear` |
| World option | `WithTargetSpread` | not given | 0 to 9 999; hashed (`spec-config.md` 8.1) |

12.2. **The rules added to validation** (each has a sentinel and a failing-case test;
`spec-config.md` 7.3 absorbs them): `ErrOutOfBounds` for a parameter outside its bound
in 12.1 and for an effective window above `MaxWelfareWindow` (2.3.4), a new bound of
`spec-config.md` 4.1 (3 650, provisional); `ErrOverflow` for a window times the largest
daily income, or the calibration products of 3.1, that do not fit (2.3.2, using the
checked 128-bit helper). `window_extended` is informational (11.4).

## 13. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, hosted in the agent
(`spec-agents.md` 3.1), all plain data in a fixed order with no maps:

- for each agent, in agent ID order: the ring buffer of its window, oldest first, with
  for each record `income_t` (signed `int64`), `cost_t` (`int64`), the relief `F_t` and the
  grant `G_t` that are inside `income_t` (signed `int64`, for the earned ratio of 2.1.7 and the observed wage of 4.3),
  the own-use receipt `K^own_t` (`int64`) and the own-use household-ticks, and the working flag
  and the working household-ticks; the number of records held (fewer than `T` for a
  new agent); and the target ratio drawn at creation (5.1);
- for each (region, role), in region then role order: the run of consecutive ticks used
  by the `Warn` of 11.3 (diagnostic: it affects no result and is saved so that a resumed
  run logs as the original did).

**Not saved:** the running sums (recomputed from the buffer on restore), the effective
window (derived from the configuration), the work share and the ratio (derived).

## 14. Entries for the stability register

Merged into `spec-stability.md` section 4 (`STB-L14` target-income labour loop,
`STB-L15` work and rest chattering, `STB-L16` a carrier's fee loop, `STB-L17` carriage
cost and the basket, `STB-L18` in-kind valuation, `STB-L19` a job is a gap in income,
`STB-L20` an uninterruptible job overshoots the target, and `STB-L40` relief
suppresses migration). The dampers this spec specifies are D08 (the spread of targets,
5.1), D09 (the rest band, 5), D13 (the floor on working ticks, 5.3) and D28 (the
proportional work share, 5). New loops introduced by this revision, for the register
owner:

- **The window against the job (L19, L20).** The derived window of 2.3.3 damps L19.
  *Delay:* `T` is now `max(WelfareWindow, WindowCycles * longest job)`, so the lag of the
  ratio grows with the job. *Undamped:* a household whose cycle is longer than a
  window's worth of savings (credit). *Test:* S11.
- **The floor and the target.** Between the target and the floor the household works
  more than its target needs. *Test:* S04 with `MinWorkShare` on and off.

## 15. Decisions taken and open questions

15.1. **Decisions** of this revision (reversible; each is stated where it is made): the
income of 2.1 (cash on inputs, accrual on tools, relief and in-kind included, upkeep
and capital not netted); the derived window of 2.3.3 over income attribution; the floor
`MinWorkShare` of 5.3 adopted at a placeholder 4 000; the check of 7 as a report; the
observed wage per working household (4.3). Added in wave 4 (one edit for the subsistence,
migration and merchant specs): the record carries `F_t`, `G_t`, `K^own_t` and the own-use
household-ticks (2.1, 13); the earned ratio (2.1.7); the observed wage per agent, without
own-use receipts and ticks (4.3); the effective window counts the own-use list and the
longest trip of a merchant role (2.3.3).

15.2. **Open questions**, each with a next step:

1. **The default `TargetRatio` in a world with no spending beyond upkeep (7.2). (Answered.)**
   Kept at 15 000, decided by the coordinating agent and confirmed with DEM in place: with the
   module on the ceiling of the mean ratio is `1 + Wref / U` (7.3), and `target_above_demand`
   warns a pack whose target is above it.
2. **The value of `MinWorkShare` and `WindowCycles`** are guesses; the numeric run of S04
   (floor on and off) and S11 (a job longer than 30 ticks) set them.
3. **A per-order job share** would remove the bias of 2.1.5. It needs the order to carry
   the part bought for the job; `spec-agents.md` owns the order.
4. **Wage per working household** with a cohort that works partly (4.3) assumes the
   executions in progress are the working households. If a role has several jobs per
   agent (it cannot: one job per agent, `spec-production.md` 5.1.2) this changes.
5. **Hours.** A tick shorter than a day makes `ExpectedWorkTicks` a number of ticks and
   not days (`spec-calendar.md` 2.5); nothing here depends on hours.
6. **A window that counts trips. (Answered for a merchant role.)** A self-carried trip (4.3) is a gap in income like a long job (STB-L19, STB-L20 and `spec-stability.md` TRN-A6 merged into L19), and a trip longer than the base window can make a healthy trader look poor. `spec-merchants.md` 12.3 proposed and 2.3.3 adopts that the longest trip of a role with a merchant block joins the derivation (`Kmax`); `spec-transport.md` 12.2 still warns with `trip_longer_than_window` for a role with no block. N-099 stays open: a migration journey is recorded as zero-income, non-working ticks and is not excluded from the window (2.1.1, `spec-migration.md` 8.6, its open question 8).

## 16. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Every
requirement is derived from text of sections 1 to 15. Requirements marked (new) state
behaviour decided in the third round; the rest restate text that already existed.

### Functional

| ID | Requirement |
|----|-------------|
| FR-WEL-001 | The welfare ratio of an agent MUST be the sum of `income_t` times 10 000 divided by the sum of `cost_t` over the agent's window, in basis points, truncated toward zero, computed with checked arithmetic. |
| FR-WEL-002 | `income_t` MUST equal the trade receipts plus the in-kind receipts plus the own-use receipts `K^own_t` (`spec-subsistence.md` 7.1) plus the relief of the tick, plus the grants of the money-supply module that are not `NotIncome`, less the money spent on bids with the purpose `Input` and the wear cost of the tick, in signed minor units, and MUST NOT net spending with the purpose `Upkeep`, `Tool` or `Other`, a levy, a starting-money grant or a grant with `NotIncome`. (new) |
| FR-WEL-003 | The wear cost of a tick MUST be, for each execution started that wears a tool, `ceil(WearPerUse * price / Durability)` at the region's market price of the tool in that tick (its effective reference price if it has never traded), times the executor's weight, and a tool purchase MUST NOT be netted. (new) |
| FR-WEL-004 | `cost_t` MUST be the cost of the agent's effective upkeep (`spec-agents.md` 5.2) at the prices of the region the agent was in during tick `t`. (new) |
| FR-WEL-005 | While an agent has no region (reserved, unused in version 1) or is away on a trip, its upkeep MUST be valued at the prices of the last region it was in and its income MUST be zero unless it receives some in kind. |
| FR-WEL-006 | An agent's records MUST stay with it when it changes region. |
| FR-WEL-007 | An agent with fewer than `T` records MUST have the missing ticks padded with `pad_income = n * U * d_bp / 10 000` and `pad_cost = n * U`, with `d_bp` the role's `DefaultRatio`, default 10 000. |
| FR-WEL-008 | The ratio MUST be reported as absent, and not as zero or infinity, when the agent has no upkeep. |
| FR-WEL-009 | The effective window of an agent MUST be the greater of `WelfareWindow` and `WindowCycles` times the longest of the `Duration` among the recipes on its role's list, the `Duration` among the recipes on its role's effective own-use list, and, for a role with a merchant block, the longest travel time of the self-carried modes of that block, `WelfareWindow` MUST default to 30 and `WindowCycles` to 2, and the effective window MUST be at most `MaxWelfareWindow`. (new) |
| FR-WEL-010 | A configuration in which a window, an income or a calibration product would overflow MUST be rejected with `ErrOverflow`, and no multiplication in this spec MUST wrap. |
| FR-WEL-011 | Nothing MUST clamp an agent's income to keep its ratio above 1: the ratio is an output and a decision input and not a constraint. |
| FR-WEL-012 | The wage for a target ratio MUST be `r_bp * B_r * D_year / (10 000 * D_work)` with the multiplication before the single truncating division, and the ratio for a wage MUST be `W * D_work * 10 000 / (B_r * D_year)`. |
| FR-WEL-013 | `D_year` MUST be the derived `TicksPerYear` and `D_work` MUST be the role's `ExpectedWorkTicks`, never a constant in code. |
| FR-WEL-014 | `ExpectedWorkTicks` MUST be an integer from 1 to the derived `TicksPerYear`, default 250, and a larger value MUST be rejected with `ErrOutOfBounds` carrying `TicksPerYear`, never scaled silently. (new, with FR-CFG-048) |
| FR-WEL-015 | No configuration field MUST hold a wage or a wage multiplier, and a skill premium MUST be measured, never set. |
| FR-WEL-016 | The role fields `Unskilled`, `TargetRatio`, `TargetSpread`, `DefaultRatio`, `InKind` and `ExpectedWorkTicks` MUST have the defaults and bounds of section 12. |
| FR-WEL-017 | For a role with recipes, a tick MUST be a working tick if the agent had a job in progress in it (started, continued or completed) or was in transit on a commercial trip, and for a role without recipes if it traded or was in transit on a commercial trip, and a tick with an own-use job in progress MUST be a working tick for a role with or without recipes; migration travel, rest under the target-income rule and idle ticks MUST NOT be working ticks. (new for the job wording) |
| FR-WEL-018 | The observed wage over a window MUST be the sum of `income_t - F_t - G_t - K^own_t` divided by the number of working household-ticks less the number of own-use household-ticks, and the same figure MUST be available for one agent over its own effective window, where a working household-tick of an agent with a job of `k` executions in progress is `k`, and its weight for a role without recipes that traded or travelled. (new) |
| FR-WEL-019 | The observed premium of a role in a region MUST be its observed wage times 10 000 divided by the mean observed wage of the region's agents of `Unskilled` roles weighted by households, and MUST be absent if the region has no such agents or the role has no working ticks. (new for the weighting) |
| FR-WEL-020 | The `Snapshot` MUST report the observed wage and premium per role and region, the effective window per role, the households held to work by the floor, the household-weighted mean welfare ratio per role and region (absent when no agent has a ratio), each bounded by the configuration. |
| FR-WEL-021 | The work share MUST be 10 000 if the ratio is below `T_lo`, 0 if it is at or above `T_hi = T_lo * (10 000 + RestBand) / 10 000`, and `(T_hi - ratio) * 10 000 / (T_hi - T_lo)` between, with `T_lo` the agent's own target, and the executions requested MUST be `ceil(weight * share / 10 000)`. (new for the band and the share) |
| FR-WEL-022 | With `RestBand` 0 the days worked per year in steady state MUST be `min(D_year, ceil(r_bp * B_r * D_year / (10 000 * W)))`. |
| FR-WEL-023 | An agent MUST be able to change its work or rest decision only when it has no job in progress. |
| FR-WEL-024 | Each agent's target MUST be drawn once at creation uniformly between `TargetRatio * (10 000 - spread) / 10 000` and `TargetRatio * (10 000 + spread) / 10 000` from the world's random source, one draw for every agent whatever the spread, in the creation order of `spec-agents.md` 3.5.2, and MUST be stored with the agent. |
| FR-WEL-025 | Validation MUST reject a role whose lowest possible target is below 10 000, or whose `TargetSpread` is 10 000 or more. |
| FR-WEL-026 | `WithTargetSpread(bp)` MUST override every role's spread, MUST return an error and not panic on an invalid value, and MUST distinguish "not given" from "given as 0". |
| FR-WEL-027 | If the working ticks in an agent's window are fewer than `MinWorkShare * T / 10 000`, its work share MUST be 10 000 whatever its ratio, and with `MinWorkShare` 0 the floor MUST be off. (new) |
| FR-WEL-028 | Validation MUST reject a role whose `MinWorkShare * TicksPerYear / 10 000` exceeds its `ExpectedWorkTicks` with `ErrOutOfBounds`. (new) |
| FR-WEL-029 | A role with `InKind = k_bp` MUST receive `income * k_bp / 10 000` of its income as upkeep goods valued at the prices of its region, and in-kind receipts MUST count in `income_t` at that value. |
| FR-WEL-030 | A trainee's upkeep taken from its trainer's account MUST count as an in-kind receipt of the trainee and MUST be netted from the trainer's income in the tick of consumption. (new) |
| FR-WEL-031 | The role economics check MUST NOT fail validation or world construction, and MUST be produced as a report with a number per role. (new) |
| FR-WEL-032 | The default `TargetRatio` of 15 000 MUST NOT be changed by this spec's rules, with the demand module on or off; in a world with no spending beyond upkeep the check of FR-WEL-031 is then a report of an unmet aspiration. (new) |
| FR-WEL-033 | The `Warn` "work floor binding" MUST be emitted per region and role, never per agent, only after more than half of the role's households have been held to work by the floor for 10 consecutive ticks, at the tenth tick and every tenth after, with the region, role and households held. (new) |
| FR-WEL-034 | The state of section 13 MUST be saved as plain data in a fixed order with no maps, and a world restored between ticks MUST give the same ratios, working flags and decisions as an uninterrupted run. |
| FR-WEL-035 | A run with a logger and an observer attached MUST give results identical to a run without them. |
| FR-WEL-036 | The household-weighted mean welfare ratio of a role in a region MUST be the sum of weight times ratio over the agents of the role whose ratio exists, divided by the sum of their weights, rounded toward zero, and MUST be absent when no such agent exists. (new, `spec-reference-chain.md` 9) |
| FR-WEL-037 | The observed wage of 4.3 MUST be available per role and region as a value per working household-tick with a flag saying whether anyone worked in the window, for the spawner's occupation signal, and MUST exclude relief. (new, `spec-spawner.md` 6.1) |
| FR-WEL-038 | With the demand module on, the role economics check of 7.3 MUST remain a report that never fails validation or construction, and the warning `target_above_demand` MUST be the demand module's. (new) |
| FR-WEL-039 | The earned ratio of an agent MUST be `sum(income_t - F_t) * 10 000 / sum(cost_t)` over the same effective window, with the same padding and truncation as the ratio, MUST be absent when the ratio is absent, and MUST be available per agent in a query. (new, `spec-migration.md` 5.3) |
| FR-WEL-040 | Each record MUST hold the relief `F_t` and the grant `G_t` inside `income_t`, the own-use receipt `K^own_t` and the own-use household-ticks, so that the earned ratio and the observed wage per agent are derived from the records alone. (new, `spec-migration.md` 5.4.3, `spec-subsistence.md` 7.3) |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-WEL-001 | With the default logger and observer, recording a welfare tick, updating the running sums and computing the ratio and the work share MUST allocate nothing beyond scratch held by the world, and the cost of a ratio SHOULD be constant in the window. |
| NFR-WEL-002 | The pure functions of `internal/welfare` MUST have fuzz tests asserting that no input within the bounds panics or overflows and that the work share lies in [0, 10 000]. |
| NFR-WEL-003 | The ratio, the padding, the closed form and the work share SHOULD have benchmarks that call `b.ReportAllocs()`. |

### Behaviour found with no clear statement (flagged)

- The weighting of the unskilled wage was "by agent count" and is now by households
  (FR-WEL-019); the earlier text would count a cohort as one.
- `income_t` had no statement of what a "recipe cost" is; FR-WEL-002 and FR-WEL-003
  now define it.
- No statement said what the ratio is when the window's profit is negative; it is
  negative (FR-WEL-001), and relief reads it as below the line.
