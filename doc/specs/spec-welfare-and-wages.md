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
relief `F_t` and the signed unearned sum `G_t` that are inside `income_t` (so that the earned ratio of
2.1.7 and the observed wage of 4.3 exist; `G_t` is defined in 2.1.9), the own-use receipt `K^own_t`, and
whether tick `t` was a working tick (4.3) with its working household-ticks and its
own-use household-ticks (section 13). The ring is also read by the saving module, whose drawdown is a pure function of it (`spec-saving-and-investment.md` 4.1):

- `income_t` is the **profit** of the tick (as in bazaarBot's `profit`), in signed
  minor units:

  ```
  income_t = R_t + Y_t + K_t + K^own_t + F_t + G_t - S_t(Input) - W_t - V_t - Z_t - I^out_t
  ```

  `R_t` is the money the agent received from trades in tick `t`; `K_t` the value of
  its in-kind receipts (section 6); `K^own_t` the value of the units that an own-use job
  of the subsistence module created at its completion in tick `t`, at the prices of
  `cost_t` (`spec-subsistence.md` 7.1, 7.2; a household that makes exactly its upkeep reads
  10 000 at any price); `F_t` relief paid to it
  (`spec-estates.md` 2.6); `Y_t` the wages it received (`wage_pay` into its account at P2.1 and P2.7, `spec-employment.md` 8.1; earned income) and `V_t` the wages it paid, released from the escrows of the hires of which it is the principal in the tick the working tick is taken (the escrow is not netted when it is made); `G_t` the **unearned sum**, signed, of the flows that are not a return to working (the grants, payouts, interest received, insurance result and rent received, less the taxes, duties and tolls paid, 2.1.9), stored as one field of the record; `S_t(Input)` the money it spent in tick `t` on bids whose
  purpose is `Input` (`spec-agents.md` 6.3, 7.3.6); `W_t` the **wear cost** of
  the tick, 2.1.4; `Z_t` the rent it paid in the tick for the use of an endowment (`spec-land.md` 5.7.3), a cost of production netted as an input is, in cash when paid (it is not a bid and has no purpose); and `I^out_t` the interest it paid (`spec-credit.md` 11.1). A term of a module that is off is 0. Spending on upkeep (purpose `Upkeep`), on a tool (`Tool`) and on
  anything else (`Other`) is **not** netted: upkeep is the `cost_t` below, and a tool
  is capital, whose use is netted as wear. A starting-money grant (`start_money`,
  `initial`) is not income, and neither is a grant with `NotIncome`. Which flow is netted, which is kept out of the observed wage and which is neither is one rule per flow, in 2.1.9. `income_t` may be negative (a tick of buying inputs and
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

2.1.4. **The wear cost** of tick `t` is recorded when the executions start and valued at P6.3,
with `cost_t` and at the same prices, not at the price of P2.7: the tick's market clears at P4.1,
after the wear is started. For each execution started in the tick that wears a tool it is
`ceil(WearPerUse * price / Durability)` at the region's market price of the tool in that tick (the
price of `spec-clearing.md` 5.4 published in the tick; the effective reference price if it has never
traded), summed over the executions. There is **one ceiling per execution**, then the product, so a
row of `n` executions of one tool costs `n * ceil(WearPerUse * price / Durability)` and never
`ceil(n * WearPerUse * price / Durability)` (`WearPerUse` 3, price 10, `Durability` 4, `n` 2: 2 * 8 =
16, against ceil(60 / 4) = 15). **The weight does not enter a second time:** an execution is one
household's and the `k` executions of a job are at most the weight (FR-SCL-007), so the sum over the
executions already counts the households. A hired execution is the principal's, and so are its `n`
(the tool is its capital, `spec-employment.md` 8.1). It is the `wear_cost` of `spec-production.md`
9.1 per execution, evaluated at the market price, and it uses no belief.

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
the relief taken out of the income. The unearned sum `G_t` (2.1.9) stays in. It is a derived read from the
records and not a second record, absent when the ratio is absent (2.2.3), and is the figure
the migration module reads as its trigger (`spec-migration.md` 5.3), so that relief does not
suppress migration (`spec-migration.md` 5.4).

2.1.8. **Own-use receipts.** `K^own_t` is counted at the creation of the units and nothing is
counted when they are eaten, whose cost is in `cost_t` as for bought goods
(`spec-subsistence.md` 7.1). So a household fed by its floor reads a ratio near what the floor
covers, and relief and the schedules of `spec-population.md` see it (`spec-subsistence.md`
7.4). A sale of the surplus counts again as cash (`spec-subsistence.md` 7.5, open question 2
there).

2.1.9. **One rule per flow.** Every flow that reaches the record is one of three things, and the
table gives each flow once, with the module that owns the flow. **Production** (in `income_t` and
in the observed wage of 4.3, which is the profit of a working tick: its receipts, and the costs of
making them, including the wages, the rent and the interest paid on the finance of it); **unearned**
(in `income_t` and in the signed `G_t`, so out of the observed wage and in the earned ratio of
2.1.7); or **not income** (in neither). The reason is one: the observed wage is the return to market
labour and the cost anchor reads it (`spec-production.md` 9.3), so a receipt or a charge that is not
part of the production of the tick stays out of it, while the target-income rule and relief read
what a household has left to live on, so a charge it must pay is netted.

| Flow | Term | Effect on `income_t` | Where | Owner |
|------|------|----------------------|-------|-------|
| Receipts of trades; in-kind receipts | `R_t`; `K_t` | + | production | 2.1, 6 |
| Wages received; wages paid (released from the escrow) | `Y_t`; `V_t` | +; - | production | `spec-employment.md` 8.1 |
| Spending on inputs; wear of a tool | `S_t(Input)`; `W_t` | -; - | production | 2.1.3, 2.1.4 |
| Rent paid for the use of an endowment | `Z_t` | - | production | `spec-land.md` 5.7.3 |
| Interest paid | `I^out_t` | - | production | `spec-credit.md` 11.1 |
| Own-use receipts | `K^own_t` | + | its own field (4.3) | `spec-subsistence.md` 7.1 |
| Relief | `F_t` | + | its own field | `spec-estates.md` 2.6 |
| A grant that is not `NotIncome` | in `G_t` | + | unearned | `spec-money-supply.md` 7.2 |
| A payout of a treasury that is not `NotIncome` (money, or in kind at the value of 6.2; it is not `K_t`) | `P_t`, in `G_t` | + | unearned | `spec-fiscal.md` 12.1 |
| Interest received | `I^in_t`, in `G_t` | + | unearned | `spec-credit.md` 11.1.1 |
| Insurance result: payouts, premiums and the principal of payout claims received less those paid | `N_t`, in `G_t` | + or - | unearned | `spec-insurance.md` 11.1 |
| Rent received by a holder | `Q_t`, in `G_t` | + | unearned | `spec-land.md` 5.7.3 |
| A tax, a duty or a toll paid to a treasury | `X_t`, in `G_t` | - | unearned | `spec-fiscal.md` 12 |
| A levy of the money-supply module | none | not netted | none | `spec-money-supply.md` 7.2 |
| The principal of a loan; a starting-money grant; a grant with `NotIncome` | none | none | none | `spec-credit.md` 11.1; 2.1 |

So `G_t = (grants) + P_t + I^in_t + N_t + Q_t - X_t`, which the record stores as one signed
integer (13): it is the field that already holds the grant, and no field is added. A tax is netted
because a household taxed a fifth that looked as rich as before would neither work more nor be
relieved (`spec-fiscal.md` 12.2); it is out of the wage because the statutory charge is not a cost
of the firm that hires, and a lower wage would lower the cost anchor by an incidence the engine
does not model. A levy is not netted in version 1: it moves money between sectors at a scale the
consumer or a policy actor sets and is not a charge for anything, and netting it is open question 7
of 15.2. A flow of `G_t` that moves after the record of its tick was pushed (P6.3 of
`spec-world.md` 6.8: the payout of a store cover at P8.1b, the principal a removal's waterfall pays
at P7.1) is added to that record by amending its last record (`spec-insurance.md` 11.1.3); an agent
with no record of the tick is not amended.

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

2.2.0. `U` is the effective upkeep valued at the prices published at the position of the read: P1.6 for relief, the view of P2.5, and phase 7 for the spawner and the estates, so the agent, estate and spawner rows agree on one figure. Padding is applied at read and never stored. The padded ratio equals `DefaultRatio` exactly when `n * U * d_bp` is divisible by 10 000 and is otherwise truncated (`pad_income` truncates): `n * U` of 1 with `d_bp` 15 000 pads 1 and reads 10 000, not 15 000.

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
window or income that validation can see would overflow is a configuration error caught at
validation (`ErrOverflow`, 12.2), not a runtime wrap. Rounding is truncation toward zero.

**What validation cannot see.** Validation bounds only the sales of a role's recipes against the
window (12.2). The receipts of a role with no recipes, relief and the unearned flows are not bounded
by it, and money up to `MaxMoney` times a window up to `MaxWelfareWindow` exceeds `int64`. An
overflow that a valid configuration and a consumer's strategy can reach is handled where it occurs
and never panics (`spec-world.md` 3.3.3; architecture.md 8.1, 13). Two rules, both deterministic,
neither changing a result below its bounds, neither an error of `Step`:

- **The clamp.** Storing a record clamps each of its fields to plus or minus
  `MaxInt64 / MaxWelfareWindow`, so that a running sum over at most `MaxWelfareWindow` records cannot
  overflow (the proof is that bound times the length of the ring, which is never above
  `MaxWelfareWindow`, 2.3.4, and stays so after the resize of 2.3.3). The bound is a parameter of
  the ring, because `MaxWelfareWindow` is a bound of the root, and the store reports that it clamped.
- **One rule for everything formed from the bounded sums.** The proof covers one ring's sums and
  nothing else: what is formed from several of them, or from several agents, can leave `int64` in a
  valid configuration. Every value of the tick that is formed from them is a **named value of the
  formulas** of this spec: a sum, a padded sum and `pad_cost` and `pad_income` (2.2), the per-agent
  numerator `income_t - F_t - G_t - K^own_t` summed (4.3), a set's numerator and denominator, a
  weighted sum (11.2), a quotient (the ratio, the earned ratio, the mean ratio, a wage, the premium),
  and what the world forms in the tick (the cost of a tick, the wear cost of 2.1.4 and the income of
  2.1). Each is computed by **saturating** arithmetic: a value that leaves `int64` is replaced by the
  nearest extreme, `MaxInt64` where the exact value is above it and `MinInt64` where below (for a sum,
  by the sign of the operands that overflowed, a difference by the sign of the exact result; for a
  quotient, by the sign of the exact quotient). A quotient is one 128-bit product followed by one
  truncating division, and **only its final value saturates**, so a ratio or a premium
  (`num * 10 000 / den`) is exact whenever it fits, however large the product by 10 000. The terms of a
  sum are taken in the order of the caller's slice (ascending agent slot), because a saturating sum is
  not associative. Nothing wraps, nothing panics, and nothing becomes "absent" for this reason: the
  only absent results are those of 2.2.3 and 4.3, applied to the saturated values unchanged. The
  functions of the tick return no `ok`, because the world cannot refuse at P6.3; the calibration
  formulas of 3.1 and 5, which run outside the tick, report an overflow (FR-WEL-010).

**The clamp is not silent** (observability-and-logging.md 3.2, 9.1; FR-STB-013, N-045): a household whose
ring holds a field at the bound is the evidence, a function of saved state, counted in
`RoleRow.ClampedHouseholds` and reported by the `Warn` "welfare record clamped" (11.3,
FR-WEL-042). `ErrOverflow` of validation (12.2) is unchanged.

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
with none of these, or only short ones, has the world's window. **A role change resizes the record and
keeps the history** (`spec-spawner.md` 4.2, `spec-agents.md` 3.8): with `T'` the new role's effective
window, the newest `min(held, T')` records are kept, and the ticks missing from `T'` are padded at read
as in 2.2 with the new role's `DefaultRatio`. A change of weight does not resize it: the window is a
function of the role and not of the weight (`spec-agents.md` 3.7). The window is **derived,
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
(a farm cannot answer a bad price faster than a crop) and is accepted. Credit (`spec-credit.md`, an optional module) funds the cash gap of a long job; it does not change how the ratio reads it. The record is
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
  working ticks from a per-agent scratch of the tick and not from that event (N-035): the
  world sets it at P2.1 for a job continued or completed and at P2.7 for a start, because
  the job is cleared at completion and a one-tick job is never seen at P6.3
  (`spec-world.md` 6.4, 6.8, P6.3; `spec-production.md` 5.2.1);
- the agent's role has no recipes (a trader or a carrier): it bought or sold
  in the tick (the `TradeExecuted` event), or it was in transit on a
  commercial trip, or it had an own-use job in progress (`spec-subsistence.md` 7.6; a
  role with no recipes may have an own-use list, 3.3.3 there).
- for a `Hireable` role (`spec-employment.md` 8.3.1) the "bought or sold" clause above does not apply: its working ticks are the ticks with a job in progress, its own or a hired one, and its working household-ticks are the `k` of that job (otherwise every tick of a worker, which buys its upkeep each tick, would be a working tick of its full weight and the floor of 5.3 would never bind);
- a tick in which the agent is the principal of a hired job in progress is a working tick for the floor of 5.3 and adds nothing to its working household-ticks (`spec-employment.md` 8.2.4); and the pooled regional wage below is taken over the agents of roles that have no employer block (`spec-employment.md` 8.4).

A **commercial trip** is travel carrying the agent's own goods to sell (a self-carried shipment of `spec-transport.md`). Its working ticks are the ticks from the dispatch tick to the tick before arrival, read from the shipment record (`spec-transport.md` 5.9, FR-TRN-021). A carrier that moves other people's goods is a producer of haul units, whose job is an ordinary recipe (`spec-transport.md` 8). Migration travel, ticks spent resting
under the target-income rule, and ticks with no activity are not working
ticks. Whether a role "has recipes" is read from the configuration, so no code
branches on a role identifier (`spec-currency.md` section 2.5).

Every tick of a self-carried trip is a working tick, so the observed wage over a trip is the income of the trip over its ticks.
Over the window `T`:

```
observed_wage = (sum of (income_t - F_t - G_t - K^own_t)) / (number of working household-ticks - number of own-use household-ticks)
```

Relief `F_t` and the unearned sum `G_t` (2.1.9: a grant, a payout, interest received, the insurance result and rent received, less a tax, duty or toll paid) are excluded: they are not earned by working (`spec-estates.md` 2.6; `spec-money-supply.md` 7.2). Wages received `Y_t` and paid `V_t`, rent paid `Z_t` and interest paid `I^out_t` stay in, as inputs and wear do (`spec-employment.md` 8.3). The
own-use receipt `K^own_t` and the own-use household-ticks are excluded too: the wage is the return
to market labour, and an own-use job is not market labour (`spec-subsistence.md` 7.3, which
gives the reason and the bias left, its open question 2). An own-use tick **is** a working tick for the
work floor of 5.3 and for the target-income rule; it is left out of the wage only. A
**working household-tick** is one household working for one tick: for an agent of
weight `w` that has a job of `k` executions in progress it is `k` (the other
`w - k` households rest, `spec-agents.md` 9.4). The **own-use household-ticks** of a tick are `k`
for an own-use job of `k` executions in progress, and are part of the working household-ticks.
For a role **with recipes** the working household-ticks are the `k` of its one job (an agent has at
most one job in progress, FR-PRO-014) and the own-use household-ticks are that `k` when the job is an
own-use job and 0 when it is a role job, so the difference is never negative. A role **without
recipes** has no job of its own to give a `k`: it works by trading, and an own-use job is a second
reason to work. Its working household-ticks are the weight when it traded or travelled; when it ran an
own-use job of `k` executions and did not trade they are `k`, and so are its own-use household-ticks;
when it traded and also ran an own-use job they are the weight and its own-use household-ticks are the
own-use `k`, so the denominator of the wage is the weight less `k`, never negative (`k` is at most
the weight). The world sets these once per tick and agent: a job continued or completed is seen at
P2.1 and a job started at P2.7, never both, because an agent starts no job in a tick in which it
continued or completed one (`spec-world.md` 6, P2.5; FR-PRO-014), so this spec states no rule for two
jobs in a tick. A working tick of a role with recipes on a commercial trip has no household-ticks rule
here and is left to `spec-merchants.md` (the merchants module).
So the observed wage is per working household, and does not depend on the weight (FR-SCL-009). An agent that earns while
resting under the target-income rule is not credited with working ticks.

The **pooled regional wage** is the same formula with the sum taken over every agent of the
region, whatever its role, each over its own effective window: the sum of
`income_t - F_t - G_t - K^own_t` over their records, divided by the sum of their working
household-ticks less their own-use household-ticks. It is the working-household-tick-weighted
mean of the role wages and is per household whatever the weights. It is the figure that the
cost anchor of `spec-production.md` 9.3 reads (as it stood at the end of the previous tick)
and that `spec-viability.md` 8.2.5 means by "the pooled observed wage". It is **unknown**
when no agent of the region has a working household-tick outside own-use in its window, or
when the pooled figure is not positive. **The starting wage** of a warm start
(`spec-viability.md` 8.2.5) is used whenever the pooled figure is unknown, and nothing is
latched: both are functions of the rings and the configuration, so they add no state and
a restore needs nothing for them (`spec-checkpoint.md` 4.5 F5).

The same formula, read over one agent's own effective window, is the **observed wage of the
agent**: the figure the migration module compares with an estimate of a destination
(`spec-migration.md` 6.3.2). It is a derived read from the records of that agent, per
household, and is absent when the agent had no working household-tick outside own-use in its
window. The formula per role and region is the sum over its agents' records.

A merchant has no recipes. Its purchases for a planned trip (the cargo, the haul units and the
feed) are bids with the purpose `Input` and are netted, so the income of a trip is its margin
(`spec-merchants.md` 3.4.2).

**Arithmetic and absence (D62).** Each wage of this section is one sum divided once: the numerator is
the sum of `income_t - F_t - G_t - K^own_t` and the denominator is the sum of the working
household-ticks less the sum of the own-use household-ticks, over the records of one agent, or over
the records of a set of agents (a role in a region, the unskilled agents of a region, a region), and
the quotient is **truncated toward zero** (architecture.md 3.5; the earned ratio and the mean ratio
have their rules in 2.1.7 and FR-WEL-036). The sums come first and the division last, so no agent's
wage is truncated before it is weighted: a set's wage is its summed numerators over its summed
denominators, and each household-tick weighs the same. A wage whose denominator is 0 (no working
household-tick outside own-use) is **absent**, not 0. Two reads differ on purpose. The **observed wage**
of an agent, or of a role in a region, is present whenever the denominator is above 0, whatever the
sign of the quotient, which may be zero or negative (the spawner's `HasWage` says only that someone
worked, FR-WEL-037). The **pooled regional wage** is known only when the denominator is above 0 and
the quotient is above 0; otherwise it is unknown and the starting wage is used (above). The function
that reads it returns the wage and whether it is known, and the world writes 0 for an unknown wage
into the cost-anchor table, which is the form `spec-production.md` 9.1.1 and FR-PRO-049 read. An
unskilled agent with no working household-tick outside own-use has no wage and adds nothing to the
unskilled wage below.

The **observed premium** of a role in a region is its observed wage divided by
the observed wage of the region's unskilled agents, in basis points:

```
premium_bp = role_wage * 10_000 / unskilled_wage
```

`unskilled_wage` is the observed wage of the set of the region's agents of roles marked
`Unskilled`, in the pooled form above: the sum of their numerators over the sum of their working
household-ticks less own-use, one truncating division. That is the mean of their wages weighted by
**households** (so that a cohort counts as the households it stands for, FR-SCL-015; the earlier
"agent count" would count a cohort of 40 as one), and it can differ from the mean of the
agents' truncated wages by up to one unit, which moves the premium. The same set function serves for
a role, so there is one implementation. If the region has no such agents, or the
role has no working ticks, or **the unskilled wage is not positive** (zero, which would divide by zero,
or negative, which would flip the sign), the premium is absent. Values are exposed per role
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
`TargetRatio * (10_000 + spread) / 10_000`. The range is the first row of the table of
`spec-world.md` 5.7.1 (`lo + draw(hi - lo + 1)`), which owns it. The word is taken for every agent at
creation, whatever the spread: it is the first of the agent's two creation words
(`spec-world.md` 5.7.1, FR-WLD-066), from the agent's own creation stream, which is
`(siteConstruction, 0, agent)` at construction, `(siteCreation, tick, new agent)` at P7.1(d) for a
replacement or a birth, and the call's stream for an external agent between ticks
(`spec-world.md` 7.3). Drawing even when the spread is 0 keeps the random stream
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
| Randomness | One creation word per agent, unused when the spread is 0 | The same word, now used. The stream does not depend on the spread, but golden output changes when a spread is configured |
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
10 000 whatever the ratio is, so the agent starts a job when it is free. `T` is the effective window
of 2.3.3 and not the number of records held, and the count is over that window: a missing tick (2.2)
is not a working tick, so a new agent is held to work until its window holds the share. The
comparison is `working < MinWorkShare * T / 10_000` with an integer division (truncation toward zero,
architecture.md 3.5). Own-use ticks count
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
held by it for 10 consecutive ticks, at the tenth tick and every tenth after (12). What "held", the count and the denominator are is stated once, in 11.3.

The snapshot field `RoleRow.HeldByFloor` is **evaluated at the boundary**, over the agents
with no job in progress on the welfare record as it stands after P6.3
(`spec-observability.md` 5.3.4): a function of saved state. It may differ from the count
the `Warn` used at P2.5, which saw a shorter ring and another set of agents, so the two
are the same quantity at two positions and not one number read twice. The `Warn` is
the signal; the field is the measure.

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
equals total upkeep spending and the **aggregate** welfare ratio (the sum of income over the sum of cost, over the same
agents and window) is **1** whatever `TargetRatio` is (`spec-viability.md` 4.7). The mean of the
per-agent ratios is not that figure: it differs from the aggregate by the covariance of the ratio
with the cost (`spec-indicators.md` 7.7.2, N-054), and `MeanRatioBP` (11.2) is such a mean. A target above 1 cannot then be met
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
the aggregate ratio is 1); `target_above_surplus` (`spec-viability.md` 10.2) warns when the mean
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
- **For the world:** the household-weighted mean target cannot exceed the ratio that the
  wants allow, `1 + Wref / U` at the reference prices and at most `1 + Wmax / U`
  (`spec-demand.md` 7.1), stated on the aggregate ratio and not on the mean of the per-agent ratios, which differs from it by the covariance with the cost (`spec-indicators.md` 7.7.2). A mean target above the first is the validation warning
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
| Welfare ratio and its window (section 2) | `welfare.go` (root, unexported) turns a tick's flows into a record at P6.3: the per-agent scratch of the tick (the trade receipts and the spending by purpose of `spec-agents.md` FR-AGT-057, the wear facts, the working flags), its conversion to a record, the cost of a tick, the effective window of a role and the per-(region, role) summaries; the per-agent ring buffer with running sums is `internal/welfare`; reported through `Snapshot`, never logged as float |
| The pure arithmetic and the ring: the ratio, the padding, the wage and ratio formulas of 3.1, the closed form of 5, the work share and the floor, the wear cost, the cost and income of a tick, the saturating arithmetic of 2.3.2, the record and the ring with its running sums and its resize | `internal/welfare` (stdlib and `internal/checked` only), over primitive slices and integers, converted inline at the call (architecture.md 2.9); named for what it provides |
| Observed wage and premium (section 4.3) | the formulas are in `internal/welfare`; `welfare.go` reads the rings of a set of agents and the root reports them in `Snapshot` per role and region |
| Calibration helper (sections 3.1, 5) | used by config tooling and tests; the running simulation does not depend on it |
| Role parameters (section 4.2) | plain structs in the root package; parsed by `config/` |

## 10. Non-goals and future work

- Military roles and their pay. The 1344 pay scale in `evidence.md` is kept as a
  note for a later feature.
- Employers other than merchants hiring carriers: now `spec-employment.md` (an employer role with a hire list). Treasuries and taxation: now `spec-fiscal.md`.
- Wages for employees working in someone else's recipe (general wage labour): now `spec-employment.md`.

## 11. Observability

11.1. **Events.** This spec adds none. The welfare ratio, the observed wage and the
premium are measures, read from the `Snapshot`, not events (observability-and-logging.md
2). Relief is `ReliefPaid` (`spec-estates.md` 2.6).

11.2. **Snapshot.** Reports, per region and role: the observed wage and the observed
premium (4.3, absent as stated there), the role's effective window (2.3.3), the
households held to work by the floor (`RoleRow.HeldByFloor`: evaluated at the boundary, over the agents with no
job in progress on the record after P6.3, so it may differ from the count of the `Warn` in the tick, 5.3),
the households whose welfare ring holds a field at the clamp of 2.3.2 (`RoleRow.ClampedHouseholds`, 11.3), and the **household-weighted mean
welfare ratio** of the role (`MeanRatioBP`: the sum over its agents of weight times ratio over the sum of weights
of the agents whose ratio exists, `RatioHouseholds`; absent if none), and the same mean over all roles of the region, which a consumer forms from the role rows weighted by `RatioHouseholds`.
`MeanRatioBP` is rounded toward zero (FR-WEL-036); the mean of `spec-indicators.md` is rounded half to
even and is authoritative for the indicators (5.5.1 there), so the two may differ by 1 basis point.
The mean ratio is what the scenarios S19 and S24 of `spec-reference-chain.md` (section 9) and the
closed-economy check of `spec-demand.md` 7 measure; it needs no event and is computed from the
records of 2, so it adds no state. Per agent, in a query
(`spec-integration.md`): the welfare ratio and whether it exists, its target ratio
(after the spread), the earned ratio beside the ratio (2.1.7), and the working ticks in its window. All are bounded by the
configuration or are per agent in a query (observability-and-logging.md 5.4.1).

11.3. **Logs.** Two `Warn`s, bounded by the configuration (observability-and-logging.md
3.2.2): "work floor binding" and "welfare record clamped".

"Work floor binding", per region and role, never per agent, emitted when more
than half of the role's households have been held to work by `MinWorkShare` for 10
consecutive ticks, at the tenth tick and every tenth after, with `region`, `role` and
`households` (the number held). **Held** means: the agent is free (it has no job in progress when
`Work` would be called), the floor condition of 5.3 holds, and the share that the ratio alone gives is
below 10 000 (an absent ratio, or a ratio below `T_lo`, which already gives 10 000, is not held). The
count is in **households** (the weights of the held agents) and is taken at P2.5 when `Work` is called.
The denominator of "more than half" is every household of the role in the region, busy or free, so a
busy agent never raises the share: strictly more than half warns and exactly half does not
(`2 * held > households`). A role with no households has no run. It is actionable: a role that works because of the
floor and not its target is a pinned labour market (STB-L14). The count behind it
makes the floor, a damper that acts silently, observable (FR-STB-013, N-045); `RoleRow.HeldByFloor` is the same quantity evaluated at the boundary and may differ from it (5.3, 11.2). The
run of consecutive ticks is kept per (region, role) and is diagnostic state (13).

"Welfare record clamped" (pattern A: the first tick of a run and every tenth after), per region and
role, never per agent, emitted when at least one household of the role holds a field of its welfare
record at the clamp of 2.3.2, with `region`, `role` and `households` (the households, by weight, of
the agents whose ring holds a field at the bound). It is actionable: a field at the bound is a flow of
the size of `MaxMoney`, which a pack or a strategy should not produce, and every ratio read from that
ring is saturated. Its counterpart is `RoleRow.ClampedHouseholds`, derived from the saved records (the
count of rings holding a field at the bound is rebuilt on restore and never saved), and its run is
diagnostic state (13). FR-WEL-042.

No `Info`, no `Error`, and nothing per agent above `Debug`.

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
checked 128-bit helper). Validation bounds only the sales of a role's recipes against the window:
what it cannot see (the receipts of a role with no recipes, relief, the unearned flows) is handled at
run time by the clamp and the saturating arithmetic of 2.3.2, which return no error, and
`ErrOverflow` is unchanged. `window_extended` is informational (11.4).

## 13. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, hosted in the agent
(`spec-agents.md` 3.1), all plain data in a fixed order with no maps:

- for each agent, in agent ID order: the ring buffer of its window, oldest first, with
  for each record `income_t` (signed `int64`), `cost_t` (`int64`), the relief `F_t` and the
  unearned sum `G_t` of 2.1.9 that is inside `income_t` (signed `int64`, for the earned ratio of 2.1.7 and the observed wage of 4.3; one field, whose meaning widens with the modules that are on and whose shape is unchanged),
  the own-use receipt `K^own_t` (`int64`) and the own-use household-ticks, and the working flag
  and the working household-ticks; the number of records held (fewer than `T` for a
  new agent); and the target ratio drawn at creation (5.1). A role change resizes the ring and keeps the newest `min(held, T')` records (2.3.3), so the number of records held is at most the current effective window;
- for each (region, role), in region then role order: the two runs of consecutive ticks used
  by the two `Warn`s of 11.3, the work floor run and the clamp run, one for each (diagnostic: they
  affect no result and are saved so that a resumed run logs as the original did, sparse,
  `spec-checkpoint.md` 4.2 tag 7).

**Not saved (derived):** the running sums and the count of fields at the clamp (recomputed from the buffer on restore), the effective
window (derived from the configuration), the work share and the ratio (derived), the pooled
regional wage and the choice of the starting wage in its place (4.3: a function of the rings
and the configuration, with no latch, `spec-checkpoint.md` 4.5 F5).

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

7. **Netting a levy (2.1.9).** A tax paid to a treasury is netted and a levy of the money-supply module is not (`spec-money-supply.md` 7.2, open question 5; `spec-fiscal.md` 12.2): the two differ in kind, and a pack that wants a levy netted too asks for the one rule. Next step: the first scenario in which a levy and a tax are both on and the target-income rule is read (STB-L14); the change is one term (`X_t`-like) in `G_t` and a format-neutral edit.

## 16. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Every
requirement is derived from text of sections 1 to 15. Requirements marked (new) state
behaviour decided in the third round; the rest restate text that already existed.

### Functional

| ID | Requirement |
|----|-------------|
| FR-WEL-001 | The welfare ratio of an agent MUST be the sum of `income_t` times 10 000 divided by the sum of `cost_t` over the agent's window, in basis points, truncated toward zero, computed with checked arithmetic. |
| FR-WEL-002 | `income_t` MUST equal the trade receipts plus the in-kind receipts plus the own-use receipts `K^own_t` (`spec-subsistence.md` 7.1) plus the wages received `Y_t`, plus the relief of the tick, plus the unearned sum `G_t` (the grants of the money-supply module that are not `NotIncome`, the payouts of a treasury that are not `NotIncome`, interest received, the insurance result and rent received, less the taxes, duties and tolls paid, 2.1.9), less the money spent on bids with the purpose `Input`, the wear cost of the tick, the wages paid `V_t`, the rent paid `Z_t` and the interest paid `I^out_t`, in signed minor units, and MUST NOT net spending with the purpose `Upkeep`, `Tool` or `Other`, a levy, the principal of a loan, a starting-money grant or a grant with `NotIncome`. (new) |
| FR-WEL-003 | The wear cost of a tick MUST be, for each execution started that wears a tool, `ceil(WearPerUse * price / Durability)` at the region's market price of the tool in that tick (its effective reference price if it has never traded), one ceiling per execution and then the sum over the executions (a row of `n` executions costs `n` times one ceiling), recorded when the executions start and valued at P6.3 at the prices of the tick, with no second factor of the weight (an execution is one household's; the `n` of a hired execution are the principal's), equal to the `wear_cost` of `spec-production.md` 9.1 per execution, and a tool purchase MUST NOT be netted. (new) |
| FR-WEL-004 | `cost_t` MUST be the cost of the agent's effective upkeep (`spec-agents.md` 5.2) at the prices of the region the agent was in during tick `t`. (new) |
| FR-WEL-005 | While an agent has no region (reserved, unused in version 1) or is away on a trip, its upkeep MUST be valued at the prices of the last region it was in and its income MUST be zero unless it receives some in kind. |
| FR-WEL-006 | An agent's records MUST stay with it when it changes region. |
| FR-WEL-007 | An agent with fewer than `T` records MUST have the missing ticks padded with `pad_income = n * U * d_bp / 10 000` and `pad_cost = n * U`, with `d_bp` the role's `DefaultRatio`, default 10 000, `U` the effective upkeep at the prices published at the position of the read, and `pad_income` truncated toward zero (so the padded ratio is exactly `DefaultRatio` only when `n * U * d_bp` is divisible by 10 000, 2.2.0). |
| FR-WEL-008 | The ratio MUST be reported as absent, and not as zero or infinity, when the agent has no upkeep. |
| FR-WEL-009 | The effective window of an agent MUST be the greater of `WelfareWindow` and `WindowCycles` times the longest of the `Duration` among the recipes on its role's list, the `Duration` among the recipes on its role's effective own-use list, and, for a role with a merchant block, the longest travel time of the self-carried modes of that block, `WelfareWindow` MUST default to 30 and `WindowCycles` to 2, and the effective window MUST be at most `MaxWelfareWindow`. (new) |
| FR-WEL-010 | A configuration in which a window, an income or a calibration product would overflow MUST be rejected with `ErrOverflow`, and no multiplication in this spec MUST wrap. |
| FR-WEL-011 | Nothing MUST clamp an agent's income to keep its ratio above 1: the ratio is an output and a decision input and not a constraint. |
| FR-WEL-012 | The wage for a target ratio MUST be `r_bp * B_r * D_year / (10 000 * D_work)` with the multiplication before the single truncating division, and the ratio for a wage MUST be `W * D_work * 10 000 / (B_r * D_year)`. |
| FR-WEL-013 | `D_year` MUST be the derived `TicksPerYear` and `D_work` MUST be the role's `ExpectedWorkTicks`, never a constant in code. |
| FR-WEL-014 | `ExpectedWorkTicks` MUST be an integer from 1 to the derived `TicksPerYear`, default 250, and a larger value MUST be rejected with `ErrOutOfBounds` carrying `TicksPerYear`, never scaled silently. (new, with FR-CFG-048) |
| FR-WEL-015 | No configuration field MUST hold a wage or a wage multiplier, and a skill premium MUST be measured, never set. |
| FR-WEL-016 | The role fields `Unskilled`, `TargetRatio`, `TargetSpread`, `DefaultRatio`, `InKind` and `ExpectedWorkTicks` MUST have the defaults and bounds of section 12. |
| FR-WEL-017 | For a role with recipes, a tick MUST be a working tick if the agent had a job in progress in it (started, continued or completed) or was in transit on a commercial trip, and for a role without recipes if it traded or was in transit on a commercial trip, and a tick with an own-use job in progress MUST be a working tick for a role with or without recipes; for a `Hireable` role a tick MUST be a working tick if a job, its own or a hired one, was in progress (its working household-ticks are the `k` of that job), and a tick in which the agent is the principal of a hired job in progress MUST be a working tick for the floor of 5.3 and add no working household-tick; migration travel, rest under the target-income rule and idle ticks MUST NOT be working ticks. (new for the job wording) |
| FR-WEL-018 | The observed wage over a window MUST be the sum of `income_t - F_t - G_t - K^own_t` (`G_t` the unearned sum of 2.1.9) divided by the number of working household-ticks less the number of own-use household-ticks, and the same figure MUST be available for one agent over its own effective window, where a working household-tick of an agent with a job of `k` executions in progress is `k`, and for a role without recipes it is its weight when it traded or travelled, and `k` when it ran an own-use job of `k` executions and did not trade (a tick with both has the weight as working household-ticks and the own-use `k` as own-use household-ticks, so the divisor is never negative); the quotient MUST be truncated toward zero, the sums taken before the one division, and the wage absent when the divisor is 0 and, for a role or an agent, present whatever the sign of the quotient. (new) |
| FR-WEL-019 | The observed premium of a role in a region MUST be its observed wage times 10 000 divided by the mean observed wage of the region's agents of `Unskilled` roles weighted by households, and MUST be absent if the region has no such agents or the role has no working ticks. The mean of the unskilled agents MUST be the pooled form (the sum of their numerators over the sum of their working household-ticks less own-use, one truncating division), the premium MUST be truncated toward zero, and the premium MUST be absent when the unskilled wage is not positive. The pooled regional wage MUST be unknown when its divisor is not above 0 or its quotient is not above 0. (new for the weighting) |
| FR-WEL-020 | The `Snapshot` MUST report the observed wage and premium per role and region, the effective window per role, the households held to work by the floor (`HeldByFloor`, evaluated at the boundary over the agents with no job in progress after P6.3, so it may differ from the count the `Warn` of FR-WEL-033 used), the household-weighted mean welfare ratio per role and region with the households whose ratio exists (`MeanRatioBP` and `RatioHouseholds`; absent when no agent has a ratio), each bounded by the configuration. |
| FR-WEL-021 | The work share MUST be 10 000 if the ratio is below `T_lo`, 0 if it is at or above `T_hi = T_lo * (10 000 + RestBand) / 10 000`, and `(T_hi - ratio) * 10 000 / (T_hi - T_lo)` between, with `T_lo` the agent's own target, and the executions requested MUST be `ceil(weight * share / 10 000)`. (new for the band and the share) |
| FR-WEL-022 | With `RestBand` 0 the days worked per year in steady state MUST be `min(D_year, ceil(r_bp * B_r * D_year / (10 000 * W)))`. |
| FR-WEL-023 | An agent MUST be able to change its work or rest decision only when it has no job in progress. |
| FR-WEL-024 | Each agent's target MUST be drawn once at creation uniformly between `TargetRatio * (10 000 - spread) / 10 000` and `TargetRatio * (10 000 + spread) / 10 000` with the first of the agent's two creation words (`spec-world.md` 5.7.1, FR-WLD-066), taken from its own creation stream for every agent whatever the spread (`spec-world.md` 7.3), in the creation order of `spec-agents.md` 3.5.2, and MUST be stored with the agent. |
| FR-WEL-025 | Validation MUST reject a role whose lowest possible target is below 10 000, or whose `TargetSpread` is 10 000 or more. |
| FR-WEL-026 | `WithTargetSpread(bp)` MUST override every role's spread, MUST return an error and not panic on an invalid value, and MUST distinguish "not given" from "given as 0". |
| FR-WEL-027 | If the working ticks in an agent's window are fewer than `MinWorkShare * T / 10 000` (integer division, `T` the effective window and not the records held, a missing tick not being a working tick), its work share MUST be 10 000 whatever its ratio, and with `MinWorkShare` 0 the floor MUST be off. (new) |
| FR-WEL-028 | Validation MUST reject a role whose `MinWorkShare * TicksPerYear / 10 000` exceeds its `ExpectedWorkTicks` with `ErrOutOfBounds`. (new) |
| FR-WEL-029 | A role with `InKind = k_bp` MUST receive `income * k_bp / 10 000` of its income as upkeep goods valued at the prices of its region, and in-kind receipts MUST count in `income_t` at that value. |
| FR-WEL-030 | A trainee's upkeep taken from its trainer's account MUST count as an in-kind receipt of the trainee and MUST be netted from the trainer's income in the tick of consumption. (new) |
| FR-WEL-031 | The role economics check MUST NOT fail validation or world construction, and MUST be produced as a report with a number per role. (new) |
| FR-WEL-032 | The default `TargetRatio` of 15 000 MUST NOT be changed by this spec's rules, with the demand module on or off; in a world with no spending beyond upkeep the check of FR-WEL-031 is then a report of an unmet aspiration. (new) |
| FR-WEL-033 | The `Warn` "work floor binding" MUST be emitted per region and role, never per agent, only after more than half of the role's households have been held to work by the floor for 10 consecutive ticks, at the tenth tick and every tenth after, with the region, role and households held, where an agent is held when it is free, the floor condition of FR-WEL-027 holds and the share its ratio alone gives is below 10 000, the count is in households taken at P2.5, the denominator is every household of the role in the region and the run requires `2 * held > households`, kept per (region, role). (new) |
| FR-WEL-034 | The state of section 13 MUST be saved as plain data in a fixed order with no maps, and a world restored between ticks MUST give the same ratios, working flags and decisions as an uninterrupted run. |
| FR-WEL-035 | A run with a logger and an observer attached MUST give results identical to a run without them. |
| FR-WEL-036 | The household-weighted mean welfare ratio of a role in a region MUST be the sum of weight times ratio over the agents of the role whose ratio exists, divided by the sum of their weights, rounded toward zero, and MUST be absent when no such agent exists. The observed wage, the pooled wage and the premium truncate toward zero likewise (4.3). (new, `spec-reference-chain.md` 9) |
| FR-WEL-037 | The observed wage of 4.3 MUST be available per role and region as a value per working household-tick with a flag saying whether anyone worked in the window, for the spawner's occupation signal, and MUST exclude relief. (new, `spec-spawner.md` 6.1) |
| FR-WEL-038 | With the demand module on, the role economics check of 7.3 MUST remain a report that never fails validation or construction, and the warning `target_above_demand` MUST be the demand module's. (new) |
| FR-WEL-039 | The earned ratio of an agent MUST be `sum(income_t - F_t) * 10 000 / sum(cost_t)` over the same effective window, with the same padding and truncation as the ratio, MUST be absent when the ratio is absent, and MUST be available per agent in a query. (new, `spec-migration.md` 5.3) |
| FR-WEL-040 | Each record MUST hold the relief `F_t` and the signed unearned sum `G_t` of 2.1.9 (one field, with no field added for interest, insurance, rent or a fiscal flow) inside `income_t`, the own-use receipt `K^own_t` and the own-use household-ticks, so that the earned ratio and the observed wage per agent are derived from the records alone. (new, `spec-migration.md` 5.4.3, `spec-subsistence.md` 7.3) |
| FR-WEL-041 | Each flow of 2.1.9 MUST enter the record by its one rule: wages received and paid, rent paid and interest paid in `income_t` and in the observed wage; a payout, interest received, the insurance result and rent received in `income_t` and in `G_t`; a tax, a duty or a toll paid netted in `G_t`; a levy of the money-supply module and the principal of a loan in neither; and a flow that moves after the record of its tick was pushed MUST be added to that record. (new, 2.1.9) |
| FR-WEL-042 | A record MUST be stored with each field clamped to plus or minus `MaxInt64 / MaxWelfareWindow` so that no running sum over a window overflows, and the clamp MUST be reported; every value of the tick formed from the sums MUST be computed by saturating arithmetic (never wrapping, never panicking, never absent for overflow); and a household whose ring holds a field at the clamp MUST be counted in `RoleRow.ClampedHouseholds` and reported by the `Warn` "welfare record clamped", per region and role, never per agent, with the region, role and households, by pattern A (the first tick of a run and every tenth after), its run saved as diagnostic state (2.3.2, 11.3, 13). (new) |

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
