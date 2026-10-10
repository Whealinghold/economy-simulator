# Spec: Population scale, weights and the cost budget

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-SCL-NNN` and
`NFR-SCL-NNN`, which tests cite (testing.md section 4). They are in section 11.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs `spec-goods.md` for the unit of a commodity; nothing else. This spec is decided before `spec-agents.md` because it changes the agent
model (roadmap, tier 1).

**The cost figures in sections 6 and 11 are provisional.** The first consumer
has confirmed its world size (five regions of about 1 000 households and 30
commodities, vision.md section 2) but not its tick rate, and nothing has been run. They are arithmetic from assumed unit costs, not
measurements, and the slice must replace them.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

An agent is one **household** (`spec-currency.md` section 5.5). A realm of
hundreds of thousands of households cannot be simulated one agent each inside a
game loop. This spec decides how the library holds a large population at a cost
a consumer can afford. It covers:

- how an agent can stand for several identical households, its **weight**
  (section 2);
- how a configuration states the real population and the simulation derives the
  agents from it, at a chosen **scale** (section 3);
- what every other spec must do with a weight (section 4);
- what scaling costs in fidelity, and the floor that limits it (section 5);
- the cost model and the budget (section 6);
- what is deliberately left out: agents-free regions and changing the scale
  while a run is in progress (section 8).

Out of scope: births, deaths and migration (`spec-population.md`,
`spec-migration.md`), the outside world as a source of prices and volumes
(`spec-external-sector.md`), and parallel execution (architecture.md 12.2).

## 2. The weight

2.1. Every agent has a **weight**: a whole number of at least 1, the number of
identical households it stands for. An agent of weight 1 is an ordinary
household. The weight of an agent is part of its state.

2.2. An agent of weight `w` behaves as `w` identical households that act in
lockstep. Everything per household in the other specs is multiplied by `w` for
the agent: its upkeep, the inputs and outputs of a recipe execution, the tools
it needs, its orders. Its money and inventory are the sums over its households.
Because `w` is an integer and quantities are integers, the multiplication is
exact. There is no rounding anywhere in scaling a quantity up.

2.3. A weight is one mechanism for two needs, so there is no second one:

- a **scaled** region: many households, few agents, at a scale chosen to fit the
  cost budget;
- an **aggregated** distant region: the same, with a larger scale, because
  nothing there needs fine detail.

Agents with a weight of 1 and agents with a larger weight can sit in the same
region: a player or a named merchant is an individual (weight 1, section 4.7)
beside cohorts of farmers.

2.4. The weight is not a probability or a size. It does not change what one
household needs or earns. `spec-welfare-and-wages.md` measures a household's
income against a household's upkeep, and the weight cancels (section 4.3).

## 3. From a census to agents

The configuration states what the world is. The scale decides how it is
simulated. Changing the scale does not change the population.

### 3.1 Groups

A **group** is a role in a region. The configuration gives, for each group, its
number of `Households`: the real population being modelled. It may give a
`Scale` for the group, the number of households per agent. A world-level default
`Scale` applies to a group that gives none. The default `Scale` is 1, so a world
that says nothing is simulated one agent per household.

### 3.2 Deriving the agents

For a group with `H` households, scale `S` and floor `K` (section 5.2):

1. `n = ceil(H / S)`, the number of agents the scale asks for.
2. If `n < K`, `n = min(K, H)`: the floor wins over the scale, and a group is
   never made of fewer than `K` agents unless it has fewer than `K` households.
3. The weights are `H / n` (integer division) for every agent, and one more for
   the first `H mod n` agents of the group, in agent order.

So the weights add up to `H` exactly, and no two weights in a group differ by
more than 1. No random number is drawn: the same configuration always gives the
same agents (architecture.md 7.4).

Example: 2 000 households of one role, `S = 40`, `K = 10`. `n = ceil(2000 / 40)
= 50`. Each weight is `2000 / 50 = 40`, with remainder 0. A group of 105
households at the same scale: `n = ceil(105 / 40) = 3`, below `K`, so
`n = min(10, 105) = 10`; the weights are `105 / 10 = 10`, and the first
`105 mod 10 = 5` agents get 11: five of 11 and five of 10, total `55 + 50 = 105`.

### 3.3 The configuration, not an option

The scale is part of the `Config`, not a functional option. It changes the world
that is built and therefore the configuration hash in the checkpoint
(`adr/0002-persistence-by-checkpoint.md`). Observers, loggers and the seed are
options because they do not.

## 4. What the other specs must do with a weight

This section is the contract that `spec-agents.md`, `spec-production.md`,
`spec-ledger.md`, `spec-welfare-and-wages.md`, `spec-estates.md`,
`spec-spawner.md` and `spec-population.md` take on. Each lists it in its own
requirements.

4.1. **Upkeep and consumption.** An agent's upkeep per tick is its household
upkeep times `w` (`spec-agents.md` 5.2). A shortfall is a shortage of the agent's
total, not an error: consumption takes what is held and `Fed`, `Unfed` and `Hunger`
count the households that went without, exactly (`spec-agents.md` 5.3 to 5.5). How a
partial shortfall reduces the weight is `spec-population.md`'s.

4.2. **Production.** One recipe execution by an agent of weight `w` is `w`
parallel household executions: it consumes `w` times the inputs and produces `w`
times the outputs. If the stock of an input covers only `k < w` executions,
the agent runs `k` (the largest number the stock allows), not none and not `w`.
That is the natural partial result for a cohort, and it is the same as `k`
households working while the rest could not (`spec-production.md` 4.2;
`spec-agents.md` 9.4).

4.3. **Welfare.** The welfare ratio is a household's income over a household's
upkeep. For an agent, income and upkeep are both totals, so the ratio is
`income / (w * upkeep_per_household)` and does not depend on `w`. A cohort and a
single household in the same circumstances have the same ratio.

4.4. **Orders.** The quantities and limit prices of the orders of an agent are
whatever its strategy chooses. A strategy that sizes orders from need produces
orders `w` times larger. A limit price is per unit and never scaled. The default
strategy does exactly this: targets, reserves, job inputs and order quantities are
`w` times the household amounts, and prices, beliefs and bounds are per unit
(`spec-agents.md` 9.1, 9.2; `spec-beliefs.md` 9.1). The clearing spec needs no change: its quantities, reservations and overflow checks
are already `int64` and checked (`spec-clearing.md` sections 2.5 and 3).

4.5. **Bounds.** The configuration validator **MUST** reject a configuration in
which the largest weight times the largest per-household quantity of any order,
stock or recipe amount exceeds the configured maximum quantity, or in which the
corresponding money amounts can overflow `int64` (architecture.md 3.5, 8.5). A
heavy weight is the usual way to hit those bounds.

4.6. **Chance events.** A per-household event with a probability, such as a tool
wearing out, must give the right expectation for `w` households. Drawing `w`
independent trials costs `O(w)` and defeats the saving. The rule: the event is
resolved with integer arithmetic and only the world's random source, and the
expected number of households it affects in a tick equals `w * p` (with `p` in
basis points). A recommended method is an expected count `floor(w * p / 10_000)`
plus a carried remainder (the agent keeps the fractional part, and takes one more
when it reaches 1), which is exact and draws nothing; the agent's remainder
starts at a value drawn from the random source when it is created, so agents of
one group do not move in step. This spec states the rule and the recommended
method. Production uses the carried remainder at every weight, 1 included, for the
fraction of a yield (`spec-production.md` 6.6), and tool breakage is a durability
counter (`spec-skill-barriers.md` 2.1), so neither draws a chance trial and neither
needs the helper; the helper stays for a future event with a true per-household
chance (open question 3, settled).

4.7. **External agents.** A player agent, and any agent a consumer adds through
the integration interface (`spec-integration.md`), has weight 1. A weight above
1 is only for agents the configuration creates.

4.8. **Changing a weight.** A weight can change only through the spawner or the
population module, never silently. A change is a ledger operation with a `reason`
(`split`, `merge`, `births`, `starvation`, and `deaths` once the population module adds it), and for a reduction, a split or a merge the agent's holdings
change in proportion, so each household keeps what it had: for money or stock
`m`, a weight going from `w` to `w'` leaves the agent with
`m' = floor(m * w' / w)`. The difference `m - m'` goes to the regional pool
(`spec-estates.md`) when it is positive, and is drawn from the pool when it is
negative. No money or goods are created or lost except through that pool. The
weights of a group still add up to its household count (FR-SCL-003), and a
population module that changes the household count changes weights, not the
number of agents, unless the group needs more agents to stay at the floor. A weight can also rise by entry (`spec-spawner.md` 4.3): the new households bring `Role.StartMoney` each from the pool and no goods, the agent's other holdings do not change, and the per-household goods of the cohort dilute; the group's household count then differs from its configured `Households`, which FR-SCL-003 states at construction only.

What the change does to the rest of the agent is `spec-agents.md` 3.7: beliefs,
residues, welfare history and `Hunger` are per household and unchanged; the job keeps
`min(k, w')` executions with its pending output scaled (`spec-production.md` 5.4);
targets, upkeep and budgets follow the new weight at once. The money difference passes
through the estate account and a draw never exceeds the pool (`spec-estates.md`
2.8). The spawner changes weights in version 1 (reductions by the failure ladder, entries into a cohort, `spec-spawner.md` 4.1, 4.3); a change of scale in a running world (split and merge) stays out of version 1 (section 8).

## 5. What scaling costs, and the floor

### 5.1 Costs in fidelity

A weight of `w` loses three things:

1. **Heterogeneity.** The `w` households are identical in belief, stock, money
   and fortune. Inequality inside a cohort is zero, so an inequality indicator
   (`spec-indicators.md`) can see only the differences between agents.
2. **Noise averaging.** With fewer agents, one agent's belief update moves a
   larger share of the market, and prices are noisier. The relative noise of a
   market average falls roughly as the square root of the number of traders, so
   scale raises it by about `sqrt(S)` for the same population.
3. **Lockstep.** `w` households cross a threshold together. The target-income
   rule already has this effect across agents (`spec-welfare-and-wages.md`
   section 5.1, `TargetSpread`) and the weight multiplies it. A group at the
   floor `K` keeps the per-agent spread working across at least `K` agents. The default
strategy also softens it inside a cohort: the work share
`k = ceil(w * s / 10 000)` lets part of the households work as the ratio crosses the
rest band (`spec-agents.md` 7.1.2, 9.3).

A cohort does not fail all at once by itself: an agent with no money still exists and
goes unfed, with `Unfed` and `Hunger` as the signals (`spec-agents.md` 3.6.1, 5.5). The
removal, and the ladder before it, are `spec-spawner.md` 5: a cohort is reduced by the unfed share (at most `DeathStep` of its weight a tick, and at most `ExitShare` of the region's households leave a tick) and removed only when its weight reaches 0 (open question 4, answered).

### 5.2 The floor `K`

`K` (`MinAgents`) is the smallest number of agents a group may be scaled down
to. A group is never reduced below `K` agents (3.2) unless it has fewer than `K`
households, in which case every household is its own agent and nothing is
scaled. So a thin market caused by a small real population is allowed to be
thin, and a thin market caused only by scaling is not.

A group that has fewer than `K` households from the start (the smallest groups of the five-region reference census have 4 to 9, `spec-reference-chain.md` 6.5) is legal and silent: every household is its own agent, the snapshot reports agents and households per group, and no warning is raised. The warning `group_below_floor` (`spec-spawner.md` 12, 13.3) is only for a group with a weight above 1 whose agent count has fallen below `K` at run time, after exits; it names the region, role and agent count and is bounded per group.

`K` is a world-level parameter. Its default is a placeholder of 10: the
bazaarBot example uses 10 agents per role (bazaarbot-reference.md section 10),
the only configuration this project has any reference for, and it was never run.
Experiments (`spec-experiments.md`) decide the real value: the least `K` at which
the scale-consistency check of section 7 passes for the reference chain.

## 6. The cost model and the budget

The library is single-threaded and starts no goroutines (architecture.md 12.1,
12.2), so a budget is a budget on one core.

6.1. **Model.** The cost of a tick is the sum of three terms:

- the agents: `sum over agents of (c_0 + |T_a| * c_t)`, where `c_0` is the fixed
  cost of an agent's tick (the production phase and the choice of recipe, `Work`,
  consumption and the welfare record, `spec-agents.md` 4, 5.3, 7.1) and `c_t` the cost
  for each commodity in the agent's **tradable set** `T_a` (`spec-agents.md` 3.4): the
  targets and tiers, at most one order (7.3.3), its admission and earmark, its share of
  the clearing, the ledger moves of its trades, the outcome and the belief revision
  (`spec-beliefs.md` 5). Every agent holds one belief per commodity in its set
  (`spec-beliefs.md` 3.1), so the work and the state of an agent grow with `|T_a|`,
  and not only with the number of agents;
- the clearing: `sum over markets of n_m * log(n_m)`, `n_m` being the orders of
  market `m` (at most `sum of |T_a|` over the agents, spread over the markets),
  small beside the first term at any size considered here;
- the observers and logging, which cost nothing when they are the no-ops
  (observability-and-logging.md 1.3) and are the consumer's otherwise.

The cost does not depend on the number of households, only on the number of
agents and the sizes of their tradable sets (FR-SCL-016). This is the point of the
weight: the amounts of a cohort are `w` times the household's in the same operations
(`spec-agents.md` 9.1; `spec-production.md` 4.2). Building the world, the warm-start
solve of `spec-viability.md` and a checkpoint are not part of a tick; the solver's
time is outside this budget.

6.2. **Provisional budget.** A tick of the reference chain with the default
logger and observer costs at most `c_0 + |T_a| * c_t` per agent on one core of the
reference machine, with `c_0` of 2 microseconds and `c_t` of 1 microsecond
(NFR-SCL-002). For an agent whose tradable set holds the **reference size** of 8
commodities that is 10 microseconds, or 100 000 agent-ticks per second. Both costs
and the size are **placeholders**, chosen to be easy to beat: `c_t` stands for about
ten steps of tens of nanoseconds each (target, intent, admission and earmark, a share
of a sort, a match, two ledger moves, an outcome and a belief revision), counted, not
measured. They replace the earlier 5 microseconds per agent, which took no account of
the tradable set: with a belief and an order for each commodity an agent deals in, an
agent that deals in 8 costs about twice that (N-015, N-030). The slice must measure the
real costs and replace them, and the reference chain fixes the real `|T_a|` per role.
Benchmarks use `b.Loop()` and `b.ReportAllocs()` (testing.md 8.1).

6.3. **What this allows (arithmetic from the placeholder costs and the reference
size of 8, not a measurement):**

| World | Households | Scale | Agents | Tick at 10 us per agent |
|-------|-----------:|------:|-------:|------------------------:|
| One region, full detail | 2 000 | 1 | 2 000 | 20 ms |
| A realm, scaled | 200 000 | 40 | 5 000 | 50 ms |
| The same realm, full detail | 200 000 | 1 | 200 000 | 2 s |
| A realm, aggregated distant regions | 200 000 | 40 near, 400 far | 2 750 | about 28 ms |
| **The first consumer** (5 regions, 30 commodities) | 5 000 | 1 | 5 000 | 50 ms, plus the clearing below |

The third row is why this spec exists: two seconds a tick, where a tick is a day by default,
is too slow for a game that ticks several days a second, and the first consumer's
needs (vision.md section 2) rule it out. The fourth row assumes half the
population (100 000 households) is near at scale 40, which is 2 500 agents, and
half is distant at scale 400, which is 250 agents: 2 750 agents, 27.5 ms.

The last row is the check against the first consumer's world (N-030), now that the
agent, production and belief specs say what an agent does. It has 150 markets (5
regions of 30 commodities). At most 5 000 agents times 8 orders is 40 000 orders a tick,
about 270 for each market on average. A sort of that size is about 270 * 8 = 2 200
steps, so about 330 000 steps for the 150 markets, which at an assumed 10 nanoseconds a
step is about 3 ms: some 6% of the agent term, so the clearing stays small and the
first term decides. The result is about 53 ms a tick, or 19 ticks a second, against the
25 ms and 40 ticks a second that the 5 microsecond placeholder gave. The world still
fits at weight 1; a consumer that ticks faster than about 19 times a second would need a
scale above 1 (a scale of 4 on groups of 170 households or more keeps at least 42
agents each, above `K`), or a smaller tradable set per role. The recipe count and the
tick rate of the consumer are still not known.

6.4. **Memory and checkpoint.** State, and so a checkpoint
(`adr/0002-persistence-by-checkpoint.md`), grows with the number of agents and
their lots, and not with the number of households. A weight of 1 000 costs the
same as a weight of 1 (NFR-SCL-004). Each agent also holds a belief for each
commodity of its tradable set (`spec-beliefs.md` 13), a ring buffer of welfare
records (`spec-welfare-and-wages.md` 2.3.1, default window 30), residues and a job
(`spec-agents.md` 15), so the state per agent is a few hundred bytes plus its lots; for
5 000 agents that is of the order of megabytes. This is an estimate and not a
measurement.

6.5. **Near-linear.** Doubling the agents, with the same commodities, should
slightly more than double the tick at most (NFR-SCL-001 allows a factor of 2.2,
for the clearing term). A worse-than-linear term would put the budget out of
reach as the world grows.

6.6. **Not concurrent.** Markets are independent within a tick
(`spec-clearing.md` section 7), so a future parallel step can shard by region.
That is not planned and would need architecture.md 12.2 changed first.

## 7. Consistency across scales

The weight is only sound if a scaled world behaves like the world it stands for.
That cannot be exact: noise, lockstep and heterogeneity (5.1) differ. The
requirement is an acceptance check, not an identity.

7.1. **Exact invariants**, true at every scale and checked with equality: the
households of each group (Σ weights = `Households`), the total of goods and
money at start for the same per-household endowment, and the per-household
upkeep and recipe quantities.

7.2. **Scale-consistency check.** The reference chain is built with 200 households
per role at scales 1, 4 and 10, with `K = 10` (so 200, 50 and 20 agents per
role). Each world starts from the same warm start (`spec-viability.md` 8): the table is
per household (money, stock, seed prices) with a share of each group in progress, so it
is the same at every scale, and weight times the per-household amount gives the same
total of money and of goods at every scale. Over a window that begins at tick 0, with
no settling period, the time-averaged price of each commodity at scales 4 and 10 stays
within a tolerance of its value at scale 1. Two things still differ, and both are the
noise the check measures: the stagger gives agent `i` of `n` the phase
`floor(i * P / n)` (`spec-viability.md` 8.4.3), so a coarser scale has fewer distinct
phases, and the creation draws are made for each agent (`spec-agents.md` 3.5.2). The
agents in progress on a recipe are apportioned by largest remainder, so with 20 agents
a share is rounded to a twentieth. The tolerance is a placeholder of 10% until
the slice has run; the place it belongs is `spec-stylised-facts.md`. The same
check, run over a range of `K`, is how `K` is chosen (5.2). The reference chain's scenario S10 runs this check on the `one` pack with the overlay `h200` (200 households in each of six roles, `spec-reference-chain.md` 7.2, FR-REF-033), at `K` 10: 1 200, 300 and 120 agents; the roles of that pack have no group below `K` at scale 1.

## 8. Left out

- **Regions with no agents.** A distant region modelled only by supply and
  demand functions, with no agents at all, is the outside world, and belongs to
  `spec-external-sector.md`. A region with agents, however few and heavy, is
  scaled, and belongs here.
- **Changing the scale in a running world.** Giving the agents near a player more
  detail and the distant ones less (level of detail) needs split and merge
  operations on agents, with exact ledger operations (4.8) and a checkpoint that
  records them. The weight is mutable state and the ledger reasons exist, so the
  design allows it, but version 1 fixes the scale at construction.
- **Fractional weights.** A weight is a whole number. A cohort that loses 40% of
  its households is a smaller whole weight with the remainder handled by 4.8.

## 9. Observability

9.1. `AgentSpawned` and `AgentBankrupted` carry a `Weight` field
(observability-and-logging.md 4.3), so a consumer can
rebuild the household count by role from the events alone, as it can the agent
count today. A later weight-changing event belongs to `spec-population.md`.

9.2. The snapshot reports agents and households separately, per role and per
region.

9.3. No new log line. `World created` at `Info` gains the totals as attributes
(`agents`, `households`) so an operator can see the scale of a run. They are
summary counts, not per-entity values (observability-and-logging.md 3.3.5).

9.4. The metric `economysim.agent.count` counts agents. The household count is
`economysim.household.count`, an up-down counter by role, fed by the `Weight` of the
same two events; both are in the catalogue of observability-and-logging.md 5.3 and
5.3.1, and neither counts estate pseudo-agents.

## 10. Stability register entries

Merged into `spec-stability.md` section 4.4: noise from fewer traders is STB-L33,
lockstep of a cohort is STB-L34 and synchronised failure is STB-L35. The text below is
the source, kept for reference.

For `spec-stability.md` (with sign, delay, damper and test when that spec is
written):

- **Noise from fewer traders.** Fewer agents mean larger swings from one belief
  update. Damper: the floor `K` (5.2), and the scale-consistency check (7.2).
- **Lockstep of a cohort.** Households of one agent cross thresholds together,
  adding to the lockstep of the target-income rule. Damper: `TargetSpread`
  across at least `K` agents (`spec-welfare-and-wages.md` section 5).
- **Synchronised failure.** A cohort whose money reaches 0 fails whole. Damper:
  the failure ladder in `spec-spawner.md` (relief, then partial weight
  reduction before removal). The agent side is done: a cohort with no money goes
  unfed, with `Unfed` and `Hunger` as signals, and is not removed by itself
  (`spec-agents.md` 3.6.1, 5.5).

## 11. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-SCL-001 | Every agent MUST have an integer weight of at least 1, and a configuration MUST be rejected that would give an agent a lower weight. |
| FR-SCL-002 | For a group with `H` households, scale `S` and floor `K`, the number of agents MUST be `ceil(H / S)`, raised to `min(K, H)` if that is less. |
| FR-SCL-003 | The weights of the agents of a group MUST add up to `H`, and no two weights in a group MUST differ by more than 1. |
| FR-SCL-004 | The agents and weights of a group MUST be derived without drawing from the random source, so the same configuration gives the same agents. |
| FR-SCL-005 | A group with an agent of weight above 1 MUST have at least `K` agents. |
| FR-SCL-006 | A group's `Scale` MUST default to the world's `Scale`, and the world's `Scale` MUST default to 1. |
| FR-SCL-007 | The upkeep, recipe inputs, recipe outputs and tools required by an agent MUST be its per-household amounts multiplied by its weight, exactly. |
| FR-SCL-008 | When an input covers only `k` of an agent's `w` recipe executions, the agent MUST run `k` executions. |
| FR-SCL-009 | The welfare ratio of an agent MUST equal its income divided by its weight times its household upkeep, so it MUST NOT change when the weight changes and the household's circumstances do not. |
| FR-SCL-010 | A limit price MUST NOT be scaled by a weight. |
| FR-SCL-011 | Configuration validation MUST reject a configuration in which the largest weight times the largest per-household quantity or amount exceeds the maximum quantity, or can overflow `int64`. |
| FR-SCL-012 | A per-household chance event for an agent of weight `w` and probability `p` MUST use only integer arithmetic and the world's random source, and its expected count per tick MUST equal `w * p`. |
| FR-SCL-013 | An agent added through the integration interface MUST have weight 1. |
| FR-SCL-014 | Any change of an agent's weight MUST be a ledger operation with a `reason`, MUST conserve goods and money exactly, and, when it lowers the weight or splits or merges, MUST scale its holdings in proportion with the remainder going to the pool; a rise by entry MUST bring money from the pool and no goods (`spec-spawner.md` 4.3). |
| FR-SCL-015 | The household count of a role or region MUST be the sum of the weights of its agents, and indicators of population MUST count households. |
| FR-SCL-016 | The cost of a tick MUST NOT depend on the weights: two worlds with the same agents and different weights MUST do the same work. |
| FR-SCL-017 | For the reference chain with 200 households per role (`spec-reference-chain.md` overlay `h200`, `one` pack) at scales 1, 4 and 10, with `MinAgents` 10, the time-averaged price of each commodity at scales 4 and 10 SHOULD be within the tolerance of section 7.2 of its value at scale 1. |
| FR-SCL-018 | `AgentSpawned` and `AgentBankrupted` MUST carry the agent's weight, and the snapshot MUST report agents and households separately. |
| FR-SCL-019 | At the end of construction, with or without a warm start, the weights of each group MUST add up to its `Households`, and the total money and the total quantity of each good MUST equal the households times the per-household amounts, exactly, at every scale (7.1). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-SCL-001 | Doubling the number of agents of the reference chain SHOULD increase the time of a step by a factor of at most 2.2, measured by benchmarks at three sizes. (A weight of w changes the household count, not the agent count: FR-SCL-016.) |
| NFR-SCL-002 | A step of the reference chain with the default logger and observer SHOULD cost no more than `c_0 + n_T * c_t` per agent on one core of the reference machine, with `c_0` of 2 microseconds and `c_t` of 1 microsecond, `n_T` being the size of the agent's tradable set (about 100 000 agent-ticks per second for a set of 8). Provisional placeholders, unmeasured (section 6.2); this replaces the earlier 200 000 agent-ticks per second. |
| NFR-SCL-003 | The benchmarks of NFR-SCL-001 and NFR-SCL-002 MUST call `b.ReportAllocs()`. |
| NFR-SCL-004 | The memory of a world, and the size of its checkpoint, MUST depend on the number of agents and their lots and MUST NOT depend on the weights. |

## 12. Open questions

1. **Tick budget.** The first consumer's world is five regions of about 1 000
   households and 30 commodities (vision.md section 2): 5 000 households. At scale
   1 that is 5 000 agents, the "realm, scaled" row of 6.3 with no scaling. With the
   cost per tradable commodity of 6.2 that is about 53 ms a tick, or 19 ticks a second
   (the earlier figure was 25 ms and 40 ticks a second, before the agent, production
   and belief specs made the work depend on the tradable set). So
   version 1 does not need a scale above 1 for this world unless the consumer ticks
   faster than that, and a cohort is not needed to meet the budget. An agent dealing in
   up to 30 commodities costs more than the reference size of 8 (a merchant role), which
   the slice must measure. Still open: how often the consumer ticks. NFR-SCL-002 stays a
   placeholder until it is confirmed and measured.
2. **The value of `K`.** 10 is a placeholder from an unrun example. It is set by
   the scale-consistency check (7.2) over a range of `K`.
3. ~~**Chance events (4.6).**~~ **Settled for production and skill barriers
   (N-005):** the carried remainder is used at every weight, 1 included
   (`spec-production.md` 6.6.3), starting from one seeded draw per agent at creation;
   production draws no chance trial and tool breakage is a counter. The belief gate is
   one decision per outcome whatever the weight (`spec-beliefs.md` 9.3). A future event
   with a true per-household chance uses the helper of 4.6; whether it too uses the
   carried remainder at every weight is that spec's.
4. **A cohort that fails whole (5.1).** Answered. A cohort does not fail by itself: an agent with no money still exists and goes unfed (`spec-agents.md` 3.6.1), `Unfed` and `Hunger` are the signals (5.5), and the ladder is `spec-spawner.md` 5: change role, relief, then a reduction of the weight by the unfed share, removal at weight 0. The spawner reduces the weight first, as recommended; it does not split.
5. **A household-count metric (9.4).** Answered: it is in the catalogue
   (observability-and-logging.md 5.3).
6. **Unit size (GDS).** Settled in `spec-goods.md` 2: the author chooses each
   commodity's unit, validation requires that a household's upkeep of a good is
   at least one whole unit (FR-GDS-003) and that the price clears the resolution
   floor (FR-GDS-002).
7. **Level of detail.** Whether version 1 needs more detail near a player than
   far away (section 8). If it does, split and merge move into version 1 and
   the checkpoint must record them.

## 13. Where this lives in the code

| Concern | Home |
|---------|------|
| `Weight` on the agent, `Households`, `Scale`, `MinAgents` in the configuration | root package, `agent.go`, `world.go`; parsed by `config/` |
| Deriving the agents and weights from a group (3.2) | the root package, inline in world construction; a small pure function over integers, with a table-driven test |
| Scaling of production, upkeep and welfare | the specs listed in section 4, in `internal/production` and the root `Strategy` default |
| Bounds check (4.5) | configuration validation, with the checked-arithmetic helpers of `spec-currency.md` section 7 |
| The chance-event helper (4.6) | an `internal/` package over integers and the world's `*rand.Rand` |
