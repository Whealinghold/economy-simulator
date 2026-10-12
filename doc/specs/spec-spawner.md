# Spec: The spawner: bankruptcy, the failure ladder, replacement and the entry rule

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-SPN-NNN` and
`NFR-SPN-NNN`, which tests cite (testing.md section 4). They are in section 18.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs `spec-agents.md` (removal 3.6, weight change 3.7, `Unfed`, `Hunger`, external
agents 10), `spec-estates.md` (the pool, relief, 2.8), `spec-scale.md` (4.8),
`spec-ledger.md` (7.2), `spec-welfare-and-wages.md` (the ratio and the observed wage),
`spec-skill-barriers.md` (the capital test and the traineeship) and `spec-beliefs.md` 8.

**Every number in sections 11, 15 and 16 is a placeholder.** Nothing has been run
against the library, bazaarBot has not been run, and no acceptance band exists (notes
N-015, N-027). Section 16 is a stylised model of the rules of this spec, not a run of
code and not a claim that any default is stable.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer, never a household (N-001, N-017).

## 1. Purpose and scope

The `Spawner` is the fourth-listed extension point of architecture.md 4.3: it
"decides replacement and entry of agents". It owns who leaves the economy, who enters
it, and who changes occupation. This spec covers:

- the `Spawner` interface, the operations its decisions can name, what the world
  checks, and the default implementation (sections 3, 4);
- **bankruptcy** and the **failure ladder**: change role, relief, then death (5);
- the **entry rule** and the damped occupation choice (6);
- **replacement**, funding from the pool, and what a newcomer starts with (7);
- the boundary with the population module: what is done here in version 1 and the
  hook `spec-population.md` plugs into (8);
- external agents (9), determinism and the random source (10).

Out of scope: the births and deaths rates and the Malthusian fixed point (POP); where a
migrant goes (MIG); the traineeship queue, places and throttle (`spec-skill-barriers.md`
4); the mechanics of removing an agent (`spec-agents.md` 3.6) and of relief
(`spec-estates.md` 2.6); the tick phases (WLD).

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | The interface | One method, `Decide`, called once per region per tick in the entry-and-exit phase. It gets a plain-data view and the world's random source and appends decisions to a plan: remove, reduce, switch, apply, enter | Architecture 4.2 (small interface), 7.7 (randomness from the world). A plan, not a return value, so the world reuses scratch (same shape as `TradePlan`) |
| 2 | What is bankruptcy? | Not "money is 0" (bazaarBot). A household is bankrupt when its **privation** (a leaky count of unfed ticks) has reached `RemovePrivation`. The cause is recorded: `insolvent` if it could not have paid a tick of upkeep, `starved` if it had the money but no food could be bought | Money 0 with food in the larder is not failure, and one meal in seven is not fed. A leaky count catches both (model, 16.2) |
| 3 | The ladder | Rung 0 is AGT's survival rule (work full time). Rung 1 (privation `ChangePrivation`): change to a better viable role. Rung 2: relief, which the estate pays every tick to every agent below `ReliefBelow`, not conditionally. Rung 3 (privation `RemovePrivation`): weight reduced by up to `DeathStep` of the cohort per tick, removal when it reaches 0, and at most `ExitShare` of the region's households leave per tick | The roadmap order, made of measurable thresholds. Without the cap a role of identical weight-1 agents dies in one tick (model: 130 to 150 in one tick, against 2 to 6 with the cap) |
| 4 | Which role does a newcomer take? | The role the leaver held, if it is viable; else the best viable open role by expected ratio; else none. Not "the most profitable role" | Everyone entering the best role at once is the occupation cobweb (model, 16.1: amplitude 1.0 with no limit). Like-for-like replacement leaves the occupation shares alone; the adjustment is done by the damped switching of 6 |
| 5 | What damps occupation change? | A per-tick **entry share** (region and destination), reviews staggered by the bucket of the agent ID (6.2), a hysteresis band, and a cooldown | The model shows only the entry share removes the cobweb alone; the others are cheap and shrink it further. The condition is `EntryShare * window` small (16.1) |
| 6 | Funding | Every entrant household is funded from the pool, at most `EntryPoolShare` of the pool per tick, as many as the budget buys; the rest are not created and are counted. Entrants get money and no goods | Closes N-073. Money is never created. Relief keeps at least the rest of the pool |
| 7 | Births and population | SPN owns the operations and the funding; POP owns the number. The hook is gross: `PopBirths` per role and `PopDeaths` per agent in the view (both at least 0, `spec-population.md` 4.1), executed by the same operations | One executor, two rule owners |
| 8 | External agents | Never in any decision of the default spawner. The world refuses a decision on an external agent unless it was added mortal | N-022, `spec-agents.md` 10.6 |
| 9 | Randomness | The default spawner draws nothing. The `*rand.Rand` is passed for consumers' implementations | Every rule here has a deterministic, unique order, so no draw is needed and the stream does not depend on this module |

## 3. The `Spawner` interface

### 3.1 The contract

Defined in the root package, where it is consumed (architecture.md 4.1). It ships with
a default and a compile-time assertion (4.5):

```go
var _ Spawner = DefaultSpawner{}
```

```go
// Spawner decides who leaves, who enters and who changes role.
// rng is the world's random source; an implementation must not keep it.
type Spawner interface {
	// Decide is called once per tick for each region in region order (not for an outside node,
	// `spec-world.md` 6.1.11), in the entry-and-exit phase, after consumption, when no
	// reservation exists. It appends decisions to d in the order they are to be executed.
	Decide(v *RegionView, rng *rand.Rand, d *Decisions)
}
```

3.1.1. **Pure decisions.** `Decide` reads its arguments and appends plain data. It does
not call back into the world, keep a pointer into the view, or write to it.

3.1.2. **Randomness (architecture.md 7.7).** The method receives the world's `*rand.Rand`.
An implementation **MUST NOT** create its own source, use a global one, or keep the one
it was given after the call returns. The doc comment on `Spawner` states this. The call
order is fixed (regions in order), so the stream a spawner sees is a function of the
world's state. The world's own creation draws of a region follow that region's `Decide` and
precede the `Decide` of the next region (section 10, `spec-world.md` 6.9.1).

3.1.3. **Registration.** `WithSpawner(name string, s Spawner)` replaces the default.
The *name* is in the effective configuration and its hash; the code is not
(`spec-agents.md` 6.6, same limit). The name `"default"` is built in. An optional
state pair (ADR 0002 decision 7) saves a consumer spawner's state.

3.1.4. **Panics** are not recovered (architecture.md 12.5).

### 3.2 The view

A single instance per call, held in scratch and reused; valid for the call only. All
quantities are plain integers.

| Group | Fields |
|-------|--------|
| Region | `Tick`, `Region`, `TicksPerYear`, `PoolMoney` (the estate account's money), `Budget` (`PoolMoney * EntryPoolShare / 10 000` at view time; advisory, a ceiling that a consumer spawner may use to size its plan: the world takes the budget itself after the module's deaths, 7.3), the tunables of section 11 |
| Per role (canonical order) | `Role`, `Closed`, `Households` (sum of weights), `Agents`, `Wage` (the observed wage per working household-tick, `spec-welfare-and-wages.md` 4.3) and `HasWage`, `Upkeep` (cost of one household's effective upkeep per tick at the region's prices), `ExpectedWorkTicks`, `StartMoney`, `ToolPrice` (the market price of the tools its recipes require, 0 if none), `TrainingTicks`, `QueueAhead`, `Trainers`, `MaxTrainees`, `CapitalReturn`, `PopBirths` (section 8: households the population module asks to add; 0 when it is off) |
| Per agent (ID order, no pseudo-agents) | `Agent`, `Role`, `Weight`, `External`, `Mortal`, `Money`, `Unfed`, `Hunger`, `Privation`, `RoleSince`, `Surplus` (`S_u` of `spec-skill-barriers.md` 4.5), `Upkeep` (per household), `InTraineeship`, `PopDeaths` (section 8: households the population module asks to remove from this agent, at most `Weight - Unfed`; 0 when it is off), `OwnUse` (read-only: the agent is in the own-use mode of `spec-subsistence.md` 5.3; false when the module is off; the default spawner does not read it) |

3.2.1. The view carries data, not a formula: **expected ratio** `E` and **viability** `V`
of a role are computed from it (6.1) by a pure function in `internal/occupation`, so a
consumer spawner may reuse or ignore them.

### 3.3 Decisions

```go
type DecisionKind int
const (
	DecideRemove DecisionKind = iota // remove every household of an agent
	DecideReduce                     // lower an agent's weight by Households
	DecideSwitch                     // change an agent's role
	DecideApply                      // join a role's traineeship queue
	DecideEnter                      // add Households to a role in the region
)

type Decision struct {
	Kind       DecisionKind
	Agent      AgentID   // Remove, Reduce, Switch, Apply
	Role       RoleID    // Switch, Apply, Enter
	Households int64     // Reduce, Enter; at least 1
	Cause      Cause     // carried to the event
}

type Decisions struct{ List []Decision }
```

`Cause` is a named string type; its values are exported constants of the root package
(`event.go`; `internal/logkeys` holds the key strings only): `insolvent`, `starved`, `consumer`, `population` (removal and reduction), `distress`,
`review`, `traineeship` (role change), `renewal`, `births` (entry).

## 4. Operations and what the world checks

The world executes the plan in order. Each decision is checked; a refused decision
changes nothing, is counted in `EntryLimited` (13.1) and logged at `Debug` with its
`reason`. The checks protect the world; they are not the default spawner's policy
(caps, shares and the viability gate are policy, not checked here).

| Reason | Refused when |
|--------|--------------|
| `unknown_agent`, `unknown_role` | the ID does not exist, or the agent is in another region |
| `external_agent` | the agent is external and not mortal (9) |
| `same_role` | a switch to the agent's own role |
| `in_traineeship` | a switch or apply of a trainee, or a switch of a trainer with trainees. A removal is never refused for this: removing a trainee ends its traineeship (`spec-agents.md` 3.6 step 2) and removing a trainer returns its trainees to their own role (`spec-skill-barriers.md` 4.6) |
| `on_journey` | a `Switch` or `Apply` of an agent on a migration journey (`spec-migration.md` 8.3). A removal or a reduction is never refused for this. The constant is also the name of the `region_bound` refusal that the migration module registers (`spec-migration.md` 7.2.2), which is a different table (`spec-regions.md` 6.5.2) |
| `needs_weight_one` | an `Apply` for an agent of weight above 1 (N-064, `spec-skill-barriers.md` 4.1.1) |
| `bad_households` | `Reduce` by 0 or by more than the weight; `Enter` of fewer than 1 household |
| `unfunded` | an `Enter` or a weight rise whose money the pool cannot fund (see 4.3): the part that is funded proceeds |

### 4.1 Remove and reduce

A removal is the mechanics of `spec-agents.md` 3.6 (job abandoned with `agent_removed`,
trainee ended, money and goods to the estate with `estate_money` and `estate_goods`,
account closed, ID never reused). A reduction from `w` to `w'` is the weight change of
`spec-scale.md` 4.8 and `spec-agents.md` 3.7 with the ledger reason `starvation` (cause
`starved` or `insolvent`) or `deaths` (cause `population`, a reason POP adds to the
registry): money and each good become `floor(x * w' / w)` and the difference goes to the
estate. A reduction to 0 is a removal. Neither creates or destroys anything.

### 4.2 Role change

The **mechanics of a role change** are owned here and are used by the entry rule, the
ladder and the traineeship completion of `spec-skill-barriers.md` 4.1 (cause
`traineeship`). The agent keeps its account, so no account is opened or closed.

| State | Effect |
|-------|--------|
| Money | kept |
| Goods | those in the new role's tradable set (`spec-agents.md` 3.4) are kept; the others pass to the estate with `estate_goods` and are listed (a switching cost, counted: `GoodsListed`) |
| Job | abandoned, reason `role_changed` (`spec-production.md` 5.4); N-034 |
| Beliefs | kept for commodities in both sets; new ones start per `spec-beliefs.md` 8.3; the rest are dropped |
| Welfare history | kept (records stay with the agent, `spec-welfare-and-wages.md` 2.1.2) |
| Target ratio | `clamp(new_target + (old_target - old_role_target), new_target - spread, new_target + spread)`: its offset carries over, no draw |
| Residues | set to 0 (a fraction of a unit is lost; `spec-production.md` 6.6.4) |
| Want carries (`spec-demand.md` 4.4, when the demand module is on) | dropped with the old role's taste rows; the new role's carries start by the creation formula of `spec-demand.md` 4.4 (a function of the agent index, no draw) |
| `Hunger`, `Unfed`, `Privation` | kept; the removal clock uses the time since `RoleSince` (5.3) |
| `OwnUseJobs` (`spec-subsistence.md` 5.3) | reset to 0; `OwnUse` is kept and re-evaluated at the next phase 2 against the new role's list (`spec-subsistence.md` 5.8); an own-use job in progress is abandoned with `role_changed` like any job |
| Strategy | the new role's registered strategy |
| `RoleSince` | the current tick |

### 4.3 Enter and funding (N-073)

`Enter(role, n)` adds `n` households. Placement is the world's rule: into the agent of
the role in the region with the smallest weight (ties to the lower ID) while its weight
is below the group's `Scale`; otherwise a new agent of weight 1 (so at scale 1 every
entrant is a new agent, and at larger scales the agent count stays near `ceil(H / S)`). An agent on a migration journey is never chosen (`spec-migration.md` 8.3): it is skipped as if it were not in the region.
Each household brings `Role.StartMoney` from the pool (reason `start_money` for a new
agent that is an entrant, `births` for households added to an existing agent), and **no
goods** (`spec-agents.md` 3.5.3): the per-household goods of a cohort dilute. A household
born through the population module (cause `births`) brings instead the stake
`trunc(Role.StartMoney * BirthStake / 10 000)` (`spec-population.md` 4.3), with the reason
`births` whether it joins an existing agent or becomes a new one. The funded number
is `min(n, floor(budget / amount))`, `amount` being `Role.StartMoney` or the stake; an
amount of 0 is not limited. The
world also never draws more than the pool holds (FR-EST-022). The unfunded households
are **not created**, not deferred, and are counted. This closes N-073 for entry, and a
weight rise by POP uses the same rule. A new agent takes the two creation draws of
`spec-agents.md` 3.5.2.

### 4.4 Apply

`Apply(agent, role)` enters the agent in the traineeship queue of `spec-skill-barriers.md`
4.4. The queue, places and completion are SKL's; the completion calls the role change
of 4.2.

## 5. Bankruptcy and the failure ladder

### 5.1 Privation

AGT gives `Unfed` and `Hunger` (consecutive ticks with `Unfed` above 0). A household
that eats one tick in seven resets `Hunger` each time and never reaches any threshold
(model, 16.2: 36 000 unfed household-ticks and no rung reached). So the ladder uses its
own count, the **privation** of an agent, set by the world at P6.1 with `Hunger` (ruling
C2 of `spec-world.md` 6.12; the formula is AGT's, `spec-agents.md` 5.4.1, and is restated
here) and only read by the spawner:

```
Privation = Privation + 1   if Unfed > 0
Privation = max(0, Privation - 1)   otherwise
```

A household fed every other tick stays where it is; one fed less often climbs. It is
state of the agent (section 14). It is not reset by a weight change or a role change.

### 5.2 The ladder

| Rung | Trigger | Action | Owner |
|------|---------|--------|-------|
| 0 | `Hunger > 0` or money below the reserve | work full time (the survival rule); with the subsistence module on, the own-use mode of `spec-subsistence.md` is the household's own response at this rung (rung 0, not a new rung) | AGT 7.1.3, SUB |
| 1 | `Privation >= ChangePrivation` | change role (6.4) | here |
| 2 | welfare ratio below `ReliefBelow` | relief from the pool, every tick, not conditional on anything here | EST 2.6 |
| 3 | `Privation >= RemovePrivation` and `tick - RoleSince >= RemovePrivation` | reduce weight, then remove (5.3) | here |

5.2.1. Rung 2 is a rate, not a gate: relief flows from the first tick of poverty, so by
rung 3 it has had at least `RemovePrivation` ticks to act. The ladder does not ask
whether the pool was empty; an empty pool is what `Cause` and `EntryLimited` show.

5.2.2. `ChangePrivation` is below `RemovePrivation` (validation, 12), so rung 1 comes
first. The test `tick - RoleSince >= RemovePrivation` gives an agent that has just
changed role as long to establish itself as the threshold itself, whatever its
privation was.

5.2.2a. **`RoleSince` is the tick the agent took up its present role or place.** It is set at
creation, at a role change (4.2), at the arrival of a migration journey
(`spec-migration.md` 8.4) and at a forced `Relocate` of the integration interface
(`spec-integration.md` 4.5.2 step 5), so the test `tick - RoleSince >= RemovePrivation` gives a migrant
the grace it gives a role changer. The module that sets it for a move is MIG's; the ladder's
rules are otherwise unchanged. The review clock `RoleCooldown` (6.2) reads it too, and the
settling rule of `spec-migration.md` 5.2 reuses it.

5.2.3. Trainees are exempt (their board comes from the trainer, `spec-skill-barriers.md`
4.2) and so are external agents (9).

### 5.3 Rung 3: reduction and removal

For each agent meeting the rung-3 trigger, in order of `Privation` descending then ID
ascending, until the region's allowance is used:

```
allowance      = ceil(Households * ExitShare / 10 000)          households that may leave in this region this tick
drop           = min(Unfed, max(1, ceil(w * DeathStep / 10 000)))     households of this agent
drop           = min(drop, remaining allowance)   (the agent that crosses the allowance is admitted whole: drop is cut, not the agent skipped)
w'             = w - drop;   w' = 0 is a removal
Cause          = insolvent if Money < Upkeep * w, else starved
```

So a weight-1 agent (`Unfed` 1) is removed whole, and a cohort of 40 with 7 unfed
households loses `min(7, 4) = 4` per tick. The unfed share is what is cut, never the
fed households. The agent that crosses the allowance is cut to what remains; if the
remainder is 0 it waits for the next tick. `ExitShare` 0 turns the rung off (no one
leaves by the ladder); a consumer who wants that sets it.

5.3.1. **Bankruptcy** is the rung-3 removal of an agent: the event `AgentBankrupted`
with the cause. It is the only event that removes the agent (13.1); the events that
accompany a removal are listed in `spec-world.md` 6.11 item 8.

### 5.4 Interaction with the estate and relief

The leaver's money and goods pass to the estate (AGT 3.6; the reduction of 4.1) and
become pool and listings; the pool then funds relief and entrants. The same pool funds
both, so entry is limited to `EntryPoolShare` of it per tick and relief keeps the rest
(STB-L39). Relief is paid in phase 1 and only relief is paid there; starting money moves at
creation, in phase 7 (`spec-world.md` ruling C1). So relief reads what the previous tick's
entry step left in the pool (`spec-estates.md` 2.6).

## 6. The entry rule and the occupation choice

### 6.1 The signals

Two numbers per role in the region, computed from the view (`Wage`, `Upkeep`,
`ExpectedWorkTicks`, `TicksPerYear`), both in basis points:

```
E = Wage * ExpectedWorkTicks * 10 000 / (Upkeep * TicksPerYear)      the ratio a household would reach working its expected ticks
V = Wage * 10 000 / Upkeep                                           the ratio it would reach working every tick
```

`E` ranks roles (the leisure a better wage buys is counted, as the target-income rule
counts it); `V` is the **viability** gate: a role is viable if `V >= EntryRatio`. A role
with no observed wage (`HasWage` false: nobody worked in the window) has no signal and
cannot be entered by the default spawner (6.5). Products use the checked 128-bit helper.
Roles with different upkeep are compared by ratio, never by money (`spec-currency.md`
5.6). Relief is not in the wage (`spec-welfare-and-wages.md` 4.3), so relief does not
make a role look better.

### 6.2 Voluntary review

Each agent **reviews** its role on the ticks where `(tick + bucket(AgentID, ReviewTicks))
mod ReviewTicks = 0`, with `bucket(id, P) = (Weyl2(id) * P) >> 32` of `spec-world.md` 9.1.1
(ruling C14): a fixed rotation with no random draw, so reviews are spread over the period
(the decision rate of D25, specified here for occupations). An agent reviews only if it is not
external, not a trainee, not already handled by rung 1, and `tick - RoleSince >=
RoleCooldown`. It looks at the open roles (`Closed` false) other than its own, in role
order, with a signal, and takes the one with the highest `E` (ties to the lower role
ID) that satisfies both

```
E_d * 10 000 >= E_o * (10 000 + RoleHysteresis)         hysteresis: ceil on the right
capital test of spec-skill-barriers.md 2.5 / 4.5          if the role needs a tool or training
```

where the capital test is evaluated exactly as SKL states it, with its `p_d * D_work`
read as the annual gain `(E_d - E_o) * Upkeep * TicksPerYear / 10 000` for this
household. A role that needs a tool is entered by a `Switch` provided the agent's money
covers `ToolPrice` (SKL's `tool_unaffordable`); the new role's `Trade` then buys the tool
(the agent cannot bid for it before, `not_tradable`). A role with `TrainingTicks` above 0
is entered only by `Apply`, and only by an agent of weight 1 (N-064); an agent of weight
above 1 is never offered it.

### 6.3 The caps

The reviewers that pass 6.2 and the rung-1 candidates of 6.4 are ordered by need (rung 1
first, by `Privation` descending; then by gap `E_d - E_o` descending), then by ID. They are
admitted in that order while both

```
changed in this region this tick   < ceil(Households * EntryShare / 10 000)
entered into role d this tick      < ceil(Households_d * EntryShare / 10 000)
```

hold; an agent whose best role is full takes its next best that still qualifies. The
agent that makes a count reach the cap is admitted (the cap is exceeded by at most one
agent's weight). `EntryShare` 0 turns role changes off, including rung 1. The cap of a
destination role counts only switchers and cross-role entrants: like-for-like
replacement (7.2) does not change a share and is not capped. This is the **entry limit**
(D14).

### 6.4 Rung 1

An agent with `Privation >= ChangePrivation` (not external, not a trainee, past its
cooldown) is a candidate each tick, not only on review ticks. Its destination is the open,
**viable** role with the highest `E` that beats its own (`E_d > E_o`, no hysteresis band)
and passes the capital test. If none exists it stays, and the ladder goes on to rung 3.

### 6.5 Roles with no agents

A role with no agents has no observed wage and no signal, so the default spawner never
re-seeds an extinct role. A role that never had a group and has no agent (the reference chain's `visitor`, a closed role that only external agents fill) is not extinct and is no fault: the warning is for a role that had at least one agent and has none, or that has a configured group (FR-SPN-042). The world emits a bounded `Warn` "role extinct" (13.3). A
consumer who wants a role back adds agents through the integration interface or writes a
spawner.

## 7. Replacement

### 7.1 Renewal

The households that leave in a region by rung 3 in a tick are replaced at the rate
`Renewal` (basis points; default 10 000, full replacement as bazaarBot did). While the
population module is on the effective `Renewal` is 0 and no household is replaced here
(`spec-population.md` 4.6, 8.3). Per role the
number of replacements is

```
carry_r = carry_r + removed_r * Renewal;   n_r = carry_r / 10 000;   carry_r = carry_r mod 10 000
```

with the carry kept per (region, role) so that a rate below 10 000 does not lose
fractions (section 14). With `Renewal` 5 000 and one removal per tick the entries are
0, 1, 0, 1 and so on.

### 7.2 Which role

For each role in role order, its `n_r` replacements go, in this order, to:

1. the same role, if it is open and viable (like for like; not capped, 6.3);
2. otherwise the open viable role with the highest `E` (ties to the lower role ID), under
   the caps of 6.3;
3. otherwise nowhere: the replacement is dropped and counted (`NoViableRole`).

Rule 3 is what keeps a famine a famine: where no role can feed an entrant, no entrant is
made, so the churn "starve, replace, starve" cannot run (model 16.2 ran it as the
bazaarBot policy: 316 deaths and a role share swinging between 0 and 1). The bazaarBot
rule (always the most profitable class, hard-coded by commodity name, 100 money created
from nothing, the old agent's stock destroyed; defects 13 and 14,
`bazaarbot-reference.md`) is replaced by 1 to 3, funding from the pool (4.3) and estate
listings.

### 7.3 Funding

The budget is `PoolMoney * EntryPoolShare / 10 000` taken once, after the population
module's deaths and before the switches (`spec-world.md` P7.1(d), ruling C15; the view's
`Budget` is advisory), so the order of roles does not change what any role gets, but
earlier roles in role order draw on it first. Each entrant household costs `Role.StartMoney`; the world
admits as many as the budget buys (4.3). The unfunded count of the tick is reported
(`Unfunded` in `EntryLimited`).

### 7.4 What a newcomer starts with

Money from the pool, no goods, an empty welfare record padded by the role's
`DefaultRatio` (`spec-welfare-and-wages.md` 2.2), beliefs from the price window mean or
the market price, not from the warm-start table (`spec-beliefs.md` 8.3,
`spec-viability.md` 8.8; closes the SPN part of N-038), `Hunger` and `Privation` 0, the
two creation draws of `spec-agents.md` 3.5.2, and `RoleSince` the current tick, so a
newcomer is not reviewed for `RoleCooldown` ticks.

## 8. Births and population: the boundary with POP

8.1. **What this spec owns in version 1:** the operations (`Enter`, `Reduce`, `Remove`,
`Switch`), their ledger mechanics and funding, renewal of households lost by the ladder,
and the occupation choice. **What POP owns:** how many households are born or die from
causes other than the ladder, as a function of welfare, and the Malthusian fixed point
(STB-L50). The ladder's deaths are SPN's; POP does not count them again.

8.2. **The hook.** The population module, if it is on, computes before `Decide`, from the
state at the start of the phase (`spec-population.md` 4.1), two non-negative numbers: per
role in the view `PopBirths`, the households to add, and per agent in the view `PopDeaths`,
the households to remove from it, at most `Weight - Unfed`; both are 0 when the module is
off. They are gross: births and deaths are not netted, so the money of the dead reaches the
pool and the stake of the newborn leaves it, at every scale. The default spawner executes
them:

- deaths: for each agent with `PopDeaths` above 0, in agent ID order, a `Reduce` by
  `PopDeaths` with cause `population` and ledger reason `deaths`; a reduction to 0 is a
  removal (4.1). They come after the rung-3 reductions and removals and before the budget
  of the entry step is taken (10.2), so the estates of the dead fund the same tick's births;
- births: for each role with `PopBirths` above 0, `Enter(role, PopBirths)` with cause
  `births`, last in the plan, under the funding of 7.3 with the placement of 4.3 and the
  stake of 4.3, not capped by `EntryShare` and not gated by viability (POP owns the rule).

A newborn household is an entrant with the stake (4.3) until POP changes its endowment.

8.3. **Renewal against births.** Decided (`spec-population.md` 4.6): while the population
module is on the effective `Renewal` is 0, whatever is configured, so the households the
ladder removes are not refilled and births are the only replacement. `renewal_with_population`
is a warning only when `Renewal` is set to a non-default value above 0 with the module on;
at the default it is the informational diagnostic `renewal_ignored` (`spec-population.md`
4.6.1). With the module off `Renewal` keeps its meaning (7.1).

## 9. External agents

9.1. The default spawner does not decide about an external agent: it never reduces,
removes, switches, applies or replaces one (N-022, `spec-agents.md` 10.6, FR-AGT-053).
External agents appear in the view (`External` true) so a consumer spawner can see them.

9.2. The world refuses any decision on an external agent with `external_agent`, from any
spawner, unless the agent is **mortal**, the flag `Mortal` of `NewAgent` in the integration
interface's `AddAgent`, default false (`spec-integration.md` 4.1 owns the field). A mortal
external agent is subject to the ladder only through a consumer's spawner. The consumer's
own `RemoveAgent` (`spec-integration.md` 4.4) removes any external agent, mortal or not:
`Mortal` limits what the library decides, not what the consumer asks. Whether a player can
starve is the consumer's choice (`spec-agents.md` 10.6). A mortal external agent may be a
trainee through a consumer spawner's `Apply`; a non-mortal one never (`spec-integration.md` 4.6).

9.3. External agents count in `Households` and in the role's wage like any agent.

## 10. Determinism and the random source

10.1. The default spawner draws nothing. Its decisions are functions of the view, with
total orders: regions in region order, roles in role order, agents in ID order, candidates
ordered by `Privation`, gap and then ID, destinations by `E` and then role ID. No map is
iterated (architecture.md 7.5) and every sort ends in a unique key (7.6).

10.2. **Order of the entry-and-exit phase.** For each region in region order: the world
builds the view (`Privation` was set at P6.1, 5.1); calls `Decide`; executes the plan in
plan order. The
default's plan is: rung-3 reductions and removals; then rung-1 and review switches and
applies; then renewal entries (none while the population module is on); then the births of `PopBirths`. The module's deaths (`PopDeaths`) follow the rung-3 reductions and removals and precede the switches, and the budget of 7.3 is taken after them. Creation draws
(`spec-agents.md` 3.5.2) are taken as each new agent is created, in plan order. A
consumer spawner's own draws of a region precede the creation draws of that region, because
`Decide` runs before execution, and follow those of the previous region (ruling C3,
`spec-world.md` 6.9.1). The migration pass of `spec-migration.md` 10.2 (a module) follows the whole entry step of every region, so its draws and its departures come after the creation draws of every region.

10.3. A spawner is called whether or not anything can happen, so a consumer's draws do not
depend on the module state.

## 11. Parameters

All are tunables (architecture.md 9.5) with a default, an owner, a bound and a doc line,
fields of `Config.World` unless stated, with rows in the parameter-owner table. **Every
default is a placeholder** (N-015, N-027); the values come from the model of section 16,
not from any source.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| World | `ReviewTicks` | period of the occupation review of an agent, in ticks (6.2) | 10 | 1 to 1 000 |
| World | `EntryShare` | most households that may change role per tick, in basis points of the region's households and of the destination's (6.3); 0 turns role change off | 50 | 0 to 10 000 |
| World | `RoleHysteresis` | the margin a role's expected ratio must exceed the agent's own by, in basis points (6.2) | 1 000 | 0 to `MaxBasisPoints` |
| World | `RoleCooldown` | ticks after entering a role or being created before an agent may change role (6.2) | 30 | 0 to `MaxWelfareWindow` |
| World | `EntryRatio` | the viability gate: lowest `V` of a role an entrant or a distressed agent may enter, in basis points (6.1) | 8 000 | 0 to `MaxBasisPoints` |
| World | `ChangePrivation` | privation at which rung 1 applies (5.2) | 3 | 1 to 1 000 |
| World | `RemovePrivation` | privation at which rung 3 applies (5.2) | 10 | 1 to 1 000 |
| World | `DeathStep` | largest share of an agent's weight that leaves per tick by rung 3, in basis points (5.3) | 1 000 | 1 to 10 000 |
| World | `ExitShare` | most households that may leave a region per tick by rung 3, in basis points of its households; 0 turns the rung off (5.3) | 100 | 0 to 10 000 |
| World | `Renewal` | share of the households lost by rung 3 that are replaced, in basis points (7.1) | 10 000 | 0 to 10 000 |
| World | `EntryPoolShare` | share of the pool that may fund entrants in one tick, in basis points (7.3) | 5 000 | 0 to 10 000 |
| Role | `Closed` | the default spawner places no entrant or switcher in this role (6.2) | false | boolean |

The placeholders and why: `EntryShare` 50 because in the model (16.1) shares of 200 and
above left the role swinging by 30 to 100% at a window of 30 and 120, and 50 held it at 0
(window 30) or 35% (window 120), where 20 held it at 8%; the condition is on `EntryShare`
times the effective window, not on `EntryShare` alone (17, SPN-A2). `RemovePrivation` 10
and `ChangePrivation` 3 are guesses. `ExitShare` 100 is the model's cap (16.2).

## 12. Validation

Each rule has a sentinel and a failing-case test (testing.md 3.9); errors name the path.

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a parameter of section 11 outside its bound; `ChangePrivation` not below `RemovePrivation` (rule `ladder_order`); an `EntryShare` of 0 with a `ChangePrivation` the pack relies on is not an error |
| `ErrUnknownStrategy`-style `ErrUnknownSpawner` | `WithSpawner` naming nothing registered (checked by `NewWorld`) |
| `ErrOverflow` | `Households * EntryShare`, `Households * ExitShare` or `Wage * ExpectedWorkTicks` products beyond `MaxQuantity` or `MaxMoney` at the largest weight |

Warnings (never fail validation):

| Warning | Condition |
|---------|-----------|
| `entry_share_window` | `EntryShare` times the longest effective welfare window exceeds 2 500 (the model's condition, a placeholder): occupation changes may swing (SPN-A2) |
| `renewal_with_population` | the population module on and `Renewal` set to a non-default value above 0 (8.3). At the default 10 000 only the informational diagnostic `renewal_ignored` is reported (`spec-population.md` 4.6.1) |
| `no_open_role` | every role is `Closed`: no entrant can be placed |
| `group_below_floor` | at run time, never at construction: a group with a weight above 1 whose agent count has fallen below `MinAgents` (13.3). A group that was below `MinAgents` from construction because it has fewer than `MinAgents` households (every household its own agent, `spec-scale.md` 5.2) is legal, silent here, and visible in the snapshot as agents and households per group |
| `traineeship_cohort` | a role with `TrainingTicks` above 0 and a group with a scale above 1 (already `ErrOutOfBounds` in `spec-skill-barriers.md` 4.1.1; listed so the spawner's refusal is not a surprise) |

## 13. Observability

13.1. **Events** (no map, no pointer, no wall-clock time; emitted after the transition).

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `AgentSpawned` (existing) | an agent is created, including entrants | `Agent`, `Region`, `Role`, `Weight`, `StartMoney`, `External` |
| `AgentBankrupted` (existing) | an agent is removed entirely, by rung 3 or at the consumer's request | `Agent`, `Region`, `Role`, `Weight` (at removal), `Cause`, `Money`, `GoodsListed` |
| `WeightChanged` (new) | an agent's weight changes without removal: a rung-3 reduction, an entry into an existing agent, a POP birth or death | `Agent`, `Region`, `Role`, `From`, `To`, `Cause`, `Money` (signed, positive to the agent) and `Units` (goods moved to the estate) |
| `RoleChanged` (new) | an agent changes role | `Agent`, `Region`, `From`, `To`, `Weight`, `Cause`, `GoodsListed` |
| `EntryLimited` (new) | once per region per tick in which any limit or refusal acted | `Region`, `Changed`, `Entered`, `Removed`, `Unfunded`, `NoViableRole`, `RegionCap`, `RoleCap`, `Cooldown`, `Hysteresis`, `ExitCap`, `Refused` |

13.1.0. With the population module on, `PopulationChanged` (`spec-population.md` 10.1) follows `EntryLimited` in the order of the events of a region's entry step.

13.1.1. **`AgentBankrupted` is the only event that removes the agent.** Its name is
historical: `Cause` says why (`insolvent`, `starved`, `consumer`, `population`), so
bankruptcy, starvation and death are one event with a cause, not three. A reduction that
leaves households is not a removal of an agent and has its own event, `WeightChanged`; a
role change is not a removal either. The rules now state this (observability-and-logging.md 4.3, 5.3.1):
the **agent count** is still rebuilt from `AgentSpawned` (+1) and `AgentBankrupted` (-1)
alone; the **agent count by role** also needs `RoleChanged` (-1 from, +1 to); the
**household count** needs `WeightChanged` (`To - From`) besides `Weight` on the two
events, and `RoleChanged` for the count by role. `economysim.agent.bankruptcies` counts
`AgentBankrupted`, all causes. `WeightChanged`, `RoleChanged` and `EntryLimited` are in the event table of observability-and-logging.md 4.3, and the feeds of `agent.count`, `household.count` and `role.changes` are in 5.3.1.

13.1.2. `EntryLimited` is the counter of the silent dampers (N-045, N-065, N-067: the
entry limit, the cooldown, the hysteresis band, the exit cap, the viability gate and the
funding limit). Each field is a count of households or decisions in the tick; it is not
emitted when everything is zero, so a healthy world costs nothing. Bounded by the number
of regions. `Refused` totals the world's refusals of section 4; their reasons are `Debug`.

13.2. **Metric proposals** (OBS decides): `economysim.role.changes`, a Counter `{change}`
by `economysim.role` (the destination) fed by `RoleChanged`; `household.count` and
`agent.count` as in 13.1.1.

13.3. **Logs.** Per agent, all `Debug`, guarded by `Enabled`, with constant messages:
"agent bankrupt" (`agent`, `role`, `reason`), "role changed" (`agent`, `role`, `reason`),
"decision refused" (`agent`, `reason`), "entry dropped" (`role`, `reason`: `no_viable_role`,
`unfunded`). `Warn`, bounded per region or role (observability-and-logging.md 3.2.2):
"role extinct" (a role that had at least one agent, or has a configured group, and has none in
the region, FR-SPN-042; at the first tick and each tenth after; `region`, `role`); "entry
unfunded" (a region whose pool funded no entrant for ten consecutive ticks while entrants were
due, each tenth after; `region`, `households`); "group below floor" (a group of weight above 1
whose agent count is below `MinAgents`, at run time, FR-SPN-047; at the first tick and each
tenth after; `region`, `role`, `agents`). No `Info`, no `Error`. New log keys,
added with the code: `households` (exists in AGT), `cause`.

13.4. **Snapshot.** Per region and role: households, agents, the expected ratio and
viability when defined, and the privation maximum. The counts of `EntryLimited` are the event,
not a snapshot field. Per region: the pool balance (EST) and the funding budget.

## 14. State contributed to a checkpoint

All plain data in a fixed order, no maps (ADR 0002):

- per agent: `Privation` and `RoleSince` (a signed tick; the tick the agent took up its role or
  place, 5.2.2a), together, hosted in the agent record and saved there in ID order
  (`spec-checkpoint.md` 4.2 tag 3); this spec lists no other per-agent state;
- per (region, role), in region then role order: the renewal `carry`;
- the population module's death carry per agent and birth carry per (region, role) are state of that module (`spec-population.md` 11), hosted in the agent for the first;
- the diagnostic runs for the three `Warn`s: per region the run of ticks with unfunded
  entrants; per (region, role) the run of empty ticks of a role and the run of ticks a group is
  below `MinAgents` (they affect no result and are saved so a resumed run logs as the original
  did);
- a consumer spawner's state, through the optional pair.

The default spawner has no other state. Not saved: the view, the plan, the signals `E` and
`V` (derived), the budget. A world restored between ticks makes the same decisions as an
uninterrupted one (FR-SPN-052). Closed accounts: only removals close accounts; reductions,
entries into a cohort and role changes close none, so the growth of the closed-account
list (`spec-ledger.md` open question 6, N-020) is one slot per leaver, and zero in a world
with no exits.

## 15. Worked examples

All numbers are placeholders chosen to be easy to follow, and were checked by a short
integer script (not code of the library).

15.1. **Signals.** A role with wage 1 200 per working household-tick, upkeep 1 000 per
tick, `ExpectedWorkTicks` 250 and `TicksPerYear` 365: `E = 1 200 * 250 * 10 000 / (1 000 *
365) = 8 219`, `V = 1 200 * 10 000 / 1 000 = 12 000`, viable at `EntryRatio` 8 000. An agent
whose own role has `E_o = 8 219` and `RoleHysteresis` 1 000 needs `E_d >= ceil(8 219 * 11 000
/ 10 000) = 9 041` to change.

15.2. **Review rotation.** `ReviewTicks` 10, agent ID 17: `Weyl2(17) = 17 * 0x6A09E667 mod
2^32 = 178 801 879` and `bucket(17, 10) = (178 801 879 * 10) >> 32 = 0`, so it reviews where
`(t + 0) mod 10 = 0`, that is ticks 0, 10, 20, 30. Consecutive IDs do not review on consecutive
ticks: the bucket spreads a role over the period, about one tenth of a large group each
tick (`spec-world.md` 9.1.1).

15.3. **The caps.** A region of 1 000 households, `EntryShare` 50: the region allows
`ceil(1 000 * 50 / 10 000) = 5` role changes per tick. A destination role with 50
households allows `ceil(50 * 50 / 10 000) = 1`, one with 400 allows 2. `ExitShare` 100:
`ceil(1 000 * 100 / 10 000) = 10` households may leave the region per tick.

15.4. **A cohort's reduction.** An agent of weight 40, money 12 000, 80 units of a good,
`Unfed` 7, `DeathStep` 1 000, privation at the threshold. `drop = min(7, max(1, ceil(40 *
1 000 / 10 000))) = min(7, 4) = 4`; `w' = 36`. Money becomes `floor(12 000 * 36 / 40) =
10 800` and 1 200 go to the pool (reason `starvation`); the good becomes `floor(80 * 36 /
40) = 72`, 8 units to the estate. The 36 households remain with 300 money each, as the 40
had. If it were weight 1 with `Unfed` 1 it would be removed whole.

15.5. **Funding.** Pool 30 000, `EntryPoolShare` 5 000: budget 15 000. With `StartMoney` 10 000
the tick admits `floor(15 000 / 10 000) = 1` household; the other entrants due are counted
in `Unfunded`. The pool keeps 20 000 for relief.

15.6. **Privation.** Unfed pattern 1,1,1,0,1,1,1,0,1,1,1,1,1,1 (1 = unfed): privation 1, 2,
3, 2, 3, 4, 5, 4, 5, 6, 7, 8, 9, 10. A household unfed three ticks in four climbs to the
threshold in 14 ticks, where `Hunger` never exceeded 6.

15.7. **Renewal carry.** `Renewal` 5 000, one household removed per tick: carry 5 000, 10 000,
5 000, 10 000; entries 0, 1, 0, 1.

## 16. A stylised model of the rules (not a result)

Two small Docker models of the rules of this spec. Scripts and outputs are in
`/tmp/economy-boss/spn/` (`occ.py`, `occ2.py`, `ladder.py`, `ex.py`, `*.out`), outside the
repository (N-068). They are **models of the rules, not of the library**; floating point is
used only for the demand curve; the rules are integer; the parameters are placeholders. They
show how the dampers behave, and no default is shown to be stable (N-027).

### 16.1 Model A: the occupation cobweb

1 000 weight-1 agents in 3 roles, each paying `a_r * (x_r / x0_r)^(-1/e)` per head; the signal
is the mean of that income over the last `T` ticks (the welfare window); reviews rotate by
agent ID; the switch rules are those of 6.2 and 6.3; start at shares 0.4, 0.3, 0.3; a 30%
income shock to role 0 at tick 1 500. (The model rotates reviews by raw agent ID; the rule
of 6.2 uses the bucket of `spec-world.md` 9.1.1, which spreads the same load. The rows below are
of the model, not re-run.) "Amplitude" is the largest minus the smallest share of
role 0 over the last 500 of 3 000 ticks (1.0 means it runs between empty and full).

| Variant (`e` 0.5, `T` 30) | Amplitude | Mean changes per tick |
|---------------------------|-----------|-----------------------|
| everyone, every tick, best role, no limit (the bazaarBot herd) | 1.00 | 73 |
| + review rotation 10 | 1.00 | 52 |
| + hysteresis 10% | 1.00 | 51 |
| + entry share 2% | 0.31 | 11 |
| + cooldown 30 | 0.29 | 12 |
| each alone: rotation, hysteresis | 1.00 | |
| entry share 2% alone | 0.30 | 14 |
| cooldown 30 alone | 0.35 | 33 |

With the full set (rotation 10, hysteresis 10%, cooldown 30), amplitude of role 0's share by
`EntryShare` in basis points, mean changes per tick after the slash:

| `e`, `T` | 500 | 200 | 100 | 50 | 20 | 10 |
|----------|-----|-----|-----|----|----|----|
| 0.5, 30 | 0.57 | 0.29 | 0.16 | 0.00 | 0.00 | 0.00 |
| 0.5, 120 | 1.00 | 0.90 | 0.43 | 0.35 | 0.08 | 0.07 |
| 2.0, 30 | 0.74 | 0.00 | 0.00 | 0.00 | 0.00 | 0.00 |
| 2.0, 120 | 1.00 | 0.99 | 0.42 | 0.26 | 0.00 | 0.00 |

What it shows. (1) Rotation, a hysteresis band and a cooldown **do not by themselves remove
the cobweb**; the entry share (a slew limit) does, as the production-delay model of
`spec-stability.md` 6.3 found for the production cobweb. (2) The share that works falls as
the signal's delay `T` grows: the pairs that settle have `EntryShare * T` of about 1 500 to
2 400 or less, those that do not have 6 000 or more. That is the basis of the warning
`entry_share_window` (2 500, a placeholder). (3) The cost is slow adjustment: at share 20 and
`T` 120, 1 changes per tick. What it does not show: no switching cost beyond the cooldown, no
tool or queue, a demand curve for each role instead of a market, mean-field income for the
role, and an equilibrium that exists. It is the occupation model of `spec-stability.md` 6.3
with discrete agents and the actual rules.

### 16.2 Model B: the ladder, exit and funded entry

300 households in 2 roles, upkeep 100, money kept as a balance, an indivisible meal (an agent
eats if its money reaches the upkeep), role 0's income falls to 35% (or 15%) of its price for
ticks 100 to 250. **Bazaar**: an agent unfed once is removed and replaced at once by an entrant
into the role with the best signal, funded from the pool. **Ladder**: rungs 1 and 3 of section
5, `EntryShare` 50, privation 3 and 10, the entry policy of 7.2, `EntryPoolShare` 5 000.

| Shock | Policy | Deaths | Largest exit in a tick | Lowest population | Unfed household-ticks | Role-0 share (min to max) |
|-------|--------|--------|------------------------|-------------------|-----------------------|---------------------------|
| 35% | bazaar | 316 | 166 | 34 | 316 | 0.00 to 1.00 |
| 35% | ladder, role change only | 0 | 0 | 300 | 36 094 | 0.35 to 0.50 |
| 15% | bazaar | 328 | 178 | 41 | 328 | 0.00 to 1.00 |
| 15% | ladder, no exit cap | 130 | 130 | 183 | 24 451 | 0.00 to 0.50 |
| 15% | ladder, `ExitShare` 200 | 72 | 6 | 236 | 24 530 | 0.17 to 0.50 |
| 15% | ladder, `ExitShare` 100 | 65 | 3 | 238 | 22 747 | 0.22 to 0.50 |
| 15% | ladder, `ExitShare` 50 | 52 | 2 | 250 | 28 238 | 0.25 to 0.50 |

What it shows. (1) The bazaar policy empties the failing role in one tick and fills the other
(share 0 to 1), the synchronised failure (STB-L35) and the herd. (2) The ladder replaces
deaths with role changes and hunger: with the milder shock nobody dies. The price is long
hunger: 36 000 unfed household-ticks against 316, because the rungs wait. That cost is real
and is the trade the ladder makes; it is paid in `Unfed` and shows in `UpkeepShortfall`. (3)
Without an exit cap identical agents reach the threshold together (130 leave in one tick); a
cap of 1 to 2% bounds it to 2 to 6 and halves the deaths. (4) The first version of the model
used `Hunger`, and nobody ever reached a rung: the reset on a fed tick is why privation is a
leaky count (5.1). What it does not show: no goods, no market clearing, incomes set by a
formula, the pool only a bank, no relief, no cohorts (15.4 has the arithmetic), and the
shock is exogenous.

## 17. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9; fields as its section 3. The
register owner assigns the `STB-Lnn` numbers; entries here are named `SPN-An` and say what
they merge with. Dampers `D33` to `D37` are new (numbers fixed by the boss in round 5, after DEM took D30 to D32; the
register owner merges them into `spec-stability.md` section 5). **Status is
`model` where section 16 supports it and `hand` otherwise; nothing here claims stability.**

**Dampers this spec adds or specifies:**

| ID | Damper | Home | Status |
|----|--------|------|--------|
| D09 | Hysteresis for a change of role: `RoleHysteresis` | SPN 6.2 | specified (it is not enough alone, 16.1) |
| D14 | Entry limit: `EntryShare` per region and per destination | SPN 6.3 | specified; the effective occupation damper in the model |
| D33 | Staggered occupation review: `ReviewTicks`, by the bucket of the agent ID (`spec-world.md` 9.1.1), no draw (a form of D25) | SPN 6.2 | specified |
| D34 | Cooldown after a role change: `RoleCooldown` | SPN 6.2 | specified |
| D35 | Graded exit: the privation count, the cohort `DeathStep` and the regional `ExitShare` | SPN 5 | specified; counted by `EntryLimited` |
| D36 | Entry gate: a newcomer enters only a viable role, `EntryRatio`; like for like first | SPN 6.1, 7.2 | specified |
| D37 | Funding limit: `EntryPoolShare` of the pool per tick, entrants funded in whole households | SPN 7.3 | specified |

**SPN-A1. Synchronised failure (merges STB-L35).** *Kind:* hazard. *Loop:* a shock lowers the income of
all agents of a role (+) they reach the same privation together (+) they leave together (-)
supply and demand of the role. *Delay:* the privation threshold. *Owner:* SPN 5. *Damper:* D35.
*Undamped:* a pool-free sudden loss of all food in a region leaves exits at the cap while the
rest starve. *Condition:* `ExitShare` small against the share failing at once. *Test:* S14, S26.
*Status:* model (16.2).

**SPN-A2. The occupation cobweb (merges STB-L36, extends the model of `spec-stability.md` 6.3).**
*Kind:* loop. *Loop:* a role's expected ratio (+) entry (-) its income per head (-) the ratio. B
with a delay: the welfare window `T`, the cooldown, the review period. *Owner:* SPN 6; SKL 2.5, 4.
*Damper:* D14, D09, D33, D34, D20. *Undamped:* any pack with `EntryShare * T` above about 2 500
swings (16.1). *Condition:* `EntryShare` times the effective window below about 2 000. *Test:*
S06, S26. *Status:* model.

**SPN-A3. Replacement by the most profitable role (the bazaarBot herd).** *Kind:* hazard. *Loop:*
the best role's ratio (+) all replacements into it (-) its ratio, with the window delay. B with a
delay, so a cobweb. *Owner:* SPN 7.2. *Damper:* D36 (like for like first), D14 (the cap on cross-role
entry). *Undamped:* a role that is both the best and the only viable one. *Condition:* as A2. *Test:*
S26. *Status:* model (16.2, the bazaar rows).

**SPN-A4. The churn of a famine.** *Kind:* hazard. *Loop:* an unviable region (+) deaths (+)
replacements funded from the dead agents' money (+) deaths. R, if replacements enter. *Owner:* SPN
7.2. *Damper:* D36 (no viable role, no entrant), D37. *Undamped:* a role that looks viable on a
stale wage, because `V` is a window mean. *Condition:* the window is short against the collapse.
*Test:* S27. *Status:* hand.

**SPN-A5. The pool serves two masters (extends STB-L39).** *Kind:* loop. *Loop:* a dry pool (+)
unfunded entrants (-) agents (-) households to relieve; and relief (-) pool (-) entrants. *Owner:*
SPN 7.3; EST 2.6. *Damper:* D37 reserves at least half of the pool for relief (D11, D12); D35.
*Undamped:* a pool that is empty and nothing arrives. *Condition:* money enters the region (immigrants,
merchants). *Test:* S14. *Status:* hand.

**SPN-A6. Census imbalance (extends STB-L54).** *Kind:* hazard. A configured census off the
balanced one starts a transient of role changes, bounded per tick by D14 and slowed by the window.
*Owner:* VIA 5; SPN 6. *Damper:* D14, D09. *Test:* S06 and VIA A1. *Status:* model (the transient
of 16.1 from shares 0.4, 0.3, 0.3).

**SPN-A7. Hunger, removal and demand (the SPN part of STB-L50).** *Kind:* loop. *Loop:* unfed
households (+) rung-3 exits (-) demand for the staple (-) its price (-) unfed. B, a long delay.
Renewal restores households and reopens the loop. *Owner:* SPN 5, 7; POP. *Damper:* D35, D36.
*Undamped:* POP's rates. *Condition:* in POP. *Test:* S14. *Status:* hand-off.

**SPN-A8. An extinct role is never re-seeded.** *Kind:* hazard. A role with no agents has no wage and
no signal, so no one enters it (6.5). *Owner:* SPN 6.5. *Damper:* none by design; a `Warn`. *Test:*
S27. *Status:* hand.

**SPN-A9. The cost of switching.** *Kind:* bias. Stranded goods pass to the estate (4.2) and the job
is abandoned, so a switch costs the agent; the cost is in the hysteresis in effect and in the
event's `GoodsListed`. *Test:* S06. *Status:* hand.

New scenarios proposed for section 8 of the register: **S26**, a role-income shock with the
caps on and off (asserts the role share stays within a band and the changes per tick within the
cap; entries SPN-A1 to A3), and **S27**, a region where no role is viable (asserts no entrant is
made, the `Warn` "role extinct" or the counter `NoViableRole` shows, and the households fall;
entries SPN-A4, A8). Their ids were assigned in round 5 (DEM has S23 to S25); REF names them S26 and S27.

## 18. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-SPN-001 | The root package MUST define `Spawner` with one method `Decide(v *RegionView, rng *rand.Rand, d *Decisions)`, a default `DefaultSpawner`, and the assertion `var _ Spawner = DefaultSpawner{}` (3.1; architecture.md 4.5). |
| FR-SPN-002 | The world MUST call `Decide` once per region per tick, in region order, in the entry-and-exit phase after consumption, with no reservation open, and MUST NOT call it for an outside node (`spec-world.md` 6.1.11) (3.1, 10.2). |
| FR-SPN-003 | A spawner MUST NOT create its own random source, use a global one, or keep the `*rand.Rand` after `Decide` returns, and the doc comment of `Spawner` MUST say so (3.1.2). |
| FR-SPN-004 | The world MUST execute the decisions of a region in plan order and MUST refuse, change nothing for and count each decision that fails a check of section 4, with its `reason`. |
| FR-SPN-005 | A removal MUST follow `spec-agents.md` 3.6 and a reduction MUST scale money and each good to `floor(x * w' / w)` with the difference to the estate, with the ledger reason `starvation`, `deaths` or `births`, conserving money and goods exactly (4.1). |
| FR-SPN-006 | A role change MUST keep the agent's account and money, keep goods in the new role's tradable set and pass the others to the estate with `estate_goods`, abandon the job with `role_changed`, keep beliefs for common commodities, keep the welfare history, carry the target-ratio offset without a draw, set `RoleSince` to the current tick and open or close no account (4.2). |
| FR-SPN-007 | `Enter` MUST place households in the role's agent of smallest weight below the group's `Scale`, else in a new agent of weight 1, and each household MUST bring `Role.StartMoney` from the pool and no goods (4.3). |
| FR-SPN-008 | The funded number of an `Enter` MUST be `min(n, floor(budget / StartMoney))`, the unfunded households MUST NOT be created or deferred, and the pool MUST never be overdrawn (4.3, 7.3; FR-EST-022). |
| FR-SPN-009 | A decision to `Apply` for an agent of weight above 1 MUST be refused with `needs_weight_one` (4.4; N-064). |
| FR-SPN-010 | The world MUST refuse a decision on an external agent with `external_agent` unless the agent is mortal, and the default spawner MUST NOT produce one (9). |
| FR-SPN-011 | The world MUST refuse a switch or apply of a trainee, and a switch of a trainer with trainees, with `in_traineeship`, and MUST NOT refuse a removal or reduction for that reason (4). |
| FR-SPN-012 | The spawner MUST read `Privation` as set by the world at P6.1: `Privation + 1` if `Unfed > 0` and `max(0, Privation - 1)` otherwise (5.1; `spec-agents.md` 5.4.1). |
| FR-SPN-013 | The default spawner MUST apply rung 3 to an agent with `Privation >= RemovePrivation` and `tick - RoleSince >= RemovePrivation` and to no other, and never to an external agent, a trainee or a pseudo-agent (5.2, 5.3). |
| FR-SPN-014 | A rung-3 reduction MUST be `min(Unfed, max(1, ceil(w * DeathStep / 10 000)))` households, cut to the region's remaining allowance `ceil(Households * ExitShare / 10 000)`, in order of `Privation` descending then ID ascending (5.3). |
| FR-SPN-015 | A reduction to weight 0 MUST be a removal with the event `AgentBankrupted`, with `Cause` `insolvent` if `Money < Upkeep * w` and `starved` otherwise (5.3). |
| FR-SPN-016 | `ExitShare` 0 MUST turn rung 3 off and `EntryShare` 0 MUST turn off all role changes, including rung 1 (5.3, 6.3). |
| FR-SPN-017 | `E` and `V` of a role MUST be `Wage * ExpectedWorkTicks * 10 000 / (Upkeep * TicksPerYear)` and `Wage * 10 000 / Upkeep` with checked arithmetic, and a role with no observed wage MUST have no signal and MUST NOT be entered by the default spawner (6.1, 6.5). |
| FR-SPN-018 | An agent MUST be reviewed by the default spawner only on ticks where `(tick + bucket(AgentID, ReviewTicks)) mod ReviewTicks = 0` (`bucket` of `spec-world.md` 9.1.1), and only if it is not external, not a trainee and `tick - RoleSince >= RoleCooldown` (6.2). |
| FR-SPN-019 | A review MUST choose the open role with the highest `E` (ties to the lower role ID) that satisfies `E_d * 10 000 >= E_o * (10 000 + RoleHysteresis)` and, for a role with a tool or training, the capital test of `spec-skill-barriers.md` 2.5 and 4.5 (6.2). |
| FR-SPN-020 | A role needing a tool MUST be entered by `Switch` only if the agent's money covers `ToolPrice`, and a role with `TrainingTicks` above 0 MUST be entered only by `Apply` (6.2). |
| FR-SPN-021 | The default spawner MUST admit role changes in order of need (rung 1 by privation, then gap, then ID) while fewer than `ceil(Households * EntryShare / 10 000)` have changed in the region and fewer than the same share of the destination have entered it, the agent that reaches a cap being admitted (6.3). |
| FR-SPN-022 | Like-for-like replacement MUST NOT count against the caps of 6.3 (6.3, 7.2). |
| FR-SPN-023 | A rung-1 candidate (`Privation >= ChangePrivation`) MUST be considered every tick, and its destination MUST be an open viable role with `E_d > E_o` that passes the capital test (6.4). |
| FR-SPN-024 | `Renewal` MUST replace the households lost by rung 3 with a carry per (region, role), so a rate below 10 000 loses no fraction (7.1). |
| FR-SPN-025 | A replacement MUST go to the leaver's role if it is open and viable, else to the open viable role of highest `E` under the caps, else nowhere, counted as `NoViableRole` (7.2). |
| FR-SPN-026 | The funding budget MUST be taken once per region per tick as `PoolMoney * EntryPoolShare / 10 000`, after the population module's deaths and before the switches and any entry (7.3; `spec-world.md` ruling C15). |
| FR-SPN-027 | A newcomer MUST start with beliefs from the price window mean or the market price (not the warm-start table), `Hunger` and `Privation` 0 and `RoleSince` the current tick (7.4; `spec-beliefs.md` 8.3). |
| FR-SPN-028 | A positive `PopBirths` of a role MUST be executed as an `Enter` with cause `births` under the funding of 7.3 with the stake of 4.3 and without the cap or the gate, and a positive `PopDeaths` of an agent as a `Reduce` of that agent with cause `population` and reason `deaths`, in agent ID order, at most `Weight - Unfed` (8.2). |
| FR-SPN-029 | The default spawner MUST draw nothing from the random source (10.1). |
| FR-SPN-030 | Every ordering of the default spawner MUST end in a unique key and MUST NOT depend on map iteration (10.1). |
| FR-SPN-031 | The default's plan order MUST be: reductions and removals, then the population module's deaths, then switches and applies, then renewal entries (none while the population module is on), then its births (10.2). |
| FR-SPN-032 | `AgentBankrupted` MUST be the only event that removes the agent (the events that accompany a removal are those of `spec-world.md` 6.11 item 8), with `Agent`, `Region`, `Role`, `Weight`, `Cause`, `Money` and `GoodsListed`, and MUST NOT be emitted for a reduction that leaves households or for a role change (13.1.1). |
| FR-SPN-033 | `WeightChanged` MUST be emitted for each weight change that is not a removal, with `From`, `To`, `Cause`, `Money` (signed, positive to the agent) and `Units` (goods moved to the estate; FR-OBS-019), and `RoleChanged` for each role change, with `From`, `To`, `Weight`, `Cause` and `GoodsListed` (13.1). |
| FR-SPN-034 | The agent count rebuilt from `AgentSpawned` (+1) and `AgentBankrupted` (-1) MUST equal the world's agent count at every tick, and the household count rebuilt from the `Weight` of those two events and `To - From` of `WeightChanged` MUST equal the sum of the weights (13.1.1, as the rules now state in 4.3 and 5.3.1). |
| FR-SPN-035 | `EntryLimited` MUST be emitted once per region per tick in which any limit or refusal acted, with the counts of 13.1, and MUST NOT be emitted when all are zero (13.1.2). |
| FR-SPN-036 | The `Warn`s "role extinct", "entry unfunded" and "group below floor" MUST be bounded per region or role and carry the attributes of 13.3 (13.3). |
| FR-SPN-037 | Every parameter of section 11 MUST be a field with the default and bound stated, and a value outside its bound MUST be rejected with `ErrOutOfBounds` (11, 12). |
| FR-SPN-038 | `ChangePrivation` MUST be below `RemovePrivation`, or construction fails with `ErrOutOfBounds` (rule `ladder_order`) (12). |
| FR-SPN-039 | The warnings `entry_share_window`, `renewal_with_population` and `no_open_role` MUST be reported and MUST NOT fail validation (12). |
| FR-SPN-040 | An unknown spawner name MUST be rejected by `NewWorld` with `ErrUnknownSpawner` (3.1.3). |
| FR-SPN-041 | A role marked `Closed` MUST receive no entrant and no switcher from the default spawner (6.2, 7.2). |
| FR-SPN-042 | The default spawner MUST NOT re-seed a role with no agents, and the world MUST emit the `Warn` "role extinct" for a role that had an agent or has a configured group and has none; a role that never had a group and has no agent (such as a closed role filled only by external agents) MUST NOT cause it (6.5). |
| FR-SPN-043 | The fields `Privation` and `RoleSince` (hosted in the agent record), the renewal carry and the three diagnostic runs MUST be in the checkpoint in a fixed order, and a restored world MUST make the same decisions as an uninterrupted one (14; FR-SPN-052; `spec-checkpoint.md` 4.2 tags 3 and 9). |
| FR-SPN-044 | Role changes, reductions and entries into an existing agent MUST open and close no account (14). |
| FR-SPN-045 | Money and goods MUST be conserved exactly by every decision (removal, reduction, role change, entry), and no money may enter except by `start_money` and `births` from the pool (4). |
| FR-SPN-046 | The expected ratio, the viability and the privation maximum per role and region MUST be in the snapshot; the counts of `EntryLimited` are the event (13.4). |
| FR-SPN-047 | A group of weight above 1 whose agent count falls below `MinAgents` MUST be reported by a bounded `Warn` and MUST NOT be rebuilt silently (13.3). |
| FR-SPN-048 | A role change of an agent whose role has a required tool held MUST pass that tool to the estate if it is not in the new role's tradable set (4.2). |
| FR-SPN-049 | A trainee's completion MUST use the role change of 4.2 with the cause `traineeship` (4.2, 4.4). |
| FR-SPN-050 | The default spawner MUST NOT remove, reduce or switch an agent in the tick it was created (7.4, 5.2). |
| FR-SPN-051 | A reduction or removal MUST leave the fed households' share of the agent's money per household unchanged (4.1, 15.4). |
| FR-SPN-052 | A world restored from a checkpoint taken between ticks MUST give the same `AgentSpawned`, `AgentBankrupted`, `WeightChanged` and `RoleChanged` events as an uninterrupted one (14). |
| FR-SPN-053 | `RoleSince` MUST be the tick the agent took up its present role or place: set at creation, at a role change, at the arrival of a migration journey and at a forced `Relocate` (`spec-integration.md` 4.5.2), so that rung 3 waits `RemovePrivation` ticks after an arrival as after a role change. (new, `spec-migration.md` 8.4) |
| FR-SPN-054 | A `Switch` or `Apply` of an agent on a migration journey MUST be refused with `on_journey`, a removal or a reduction MUST NOT be, and the placement of an `Enter` MUST NOT choose such an agent. (new, `spec-migration.md` 8.3) |
| FR-SPN-055 | A role change MUST reset `OwnUseJobs` to 0 and keep `OwnUse` for re-evaluation, and the per-agent view MUST carry `OwnUse` read-only. (new, `spec-subsistence.md` 5.8) |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-SPN-001 | With the default logger and observer, the view, the plan, the caps and the executions of a step of the reference chain MUST allocate nothing beyond scratch held by the world (observability-and-logging.md 4.5, 8.2). |
| NFR-SPN-002 | The cost of `Decide` for a region SHOULD be proportional to the agents of the region and not to the weights, and the sort of candidates SHOULD cost `O(k log k)` in the candidates `k`. |
| NFR-SPN-003 | Observation MUST NOT change results: a run with a logger and an observer attached MUST make the same decisions as one without. |

## 19. Open questions

1. **Does a household in a cohort change role whole?** A switch moves the whole agent. For
   weight above 1 this is a lockstep of up to `w` households. Splitting off the unfed share
   needs a `split` (level of detail, version 1 fixes the scale). The first consumer needs no
   weight above 1 (N-030). Next step: `spec-scale.md` open question 7.
2. **Stranded goods.** They pass to the estate without payment (4.2). A sale at the agent's
   risk, or leaving them dormant, is the alternative. Next step: measure the loss in S06.
3. **Renewal default.** 10 000 holds the census steady and matches bazaarBot. Answered for
   POP: while the population module is on `Renewal` is ignored (8.3). Whether the field should
   disappear when the module is on is left open.
4. **`Privation` is SPN's counter, not AGT's.** It could move to AGT beside `Hunger`; it is
   here because the ladder is the only reader.
5. **The relief order.** Answered by `spec-world.md` ruling C1: phase 1 pays relief only and
   starting money moves at creation in phase 7, so relief reads what the previous tick's
   entry step left (5.4).
6. **Entry gate on a stale wage.** `V` is a window mean, so a collapsing role looks viable for
   about half a window (SPN-A4). A shorter window for the gate would need a second window.
7. **`EntryShare` against the window.** The condition is a model's (16.1); the warning's 2 500 is
   a placeholder. Next step: S06 and S26 on the reference chain.
8. **A mortal external agent.** Answered: the field is `NewAgent.Mortal` of `AddAgent`
   (`spec-integration.md` 4.1) and the check of section 4 reads it.

## 20. Where this lives in the code

| Concern | Home |
|---------|------|
| `Spawner`, `DefaultSpawner`, `RegionView`, `Decisions`, `Decision`, `DecisionKind`, `Cause`, `WithSpawner` | root package, `spawner.go` (architecture.md 2.6) |
| The tunables of section 11 and `Role.Closed` | root package, `config.go`, `defaults.go`; bounds in `bounds.go` |
| Pure rules: `E`, `V`, the caps, the apportionment of allowances, the carry, the privation update | `internal/occupation` (stdlib only, over primitive slices and indexes, converted inline at the call, architecture.md 2.9) |
| Execution of decisions (removal, reduction, role change, entry, funding) | root package, `world.go`, calling `internal/ledger` inline |
| Events `WeightChanged`, `RoleChanged`, `EntryLimited` (and the extended `AgentBankrupted`) | root package, `event.go` |
| Reason and cause constants (the `reason` values of section 4 and the `Cause` values of 3.3) | exported constants of the root package, `event.go`; `internal/logkeys` holds the key strings only |
