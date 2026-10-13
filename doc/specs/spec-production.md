# Spec: Production, recipes, jobs, yield and tools

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-PRO-NNN` and
`NFR-PRO-NNN`, which tests cite (testing.md section 4). They are in section 17.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)).
Needs `spec-goods.md` (the fields of a commodity, `Durability`) and
`spec-ledger.md` (`Transform`, `Wear`, the reason registry). Three parts are
optional or belong to another spec and are marked as such: the endowment limit
(`spec-resources.md`, written: the seam of 6.5 is now completed there), the yield table's values
(owned by `spec-resources.md` 7; the format, the lookup and the validation stay here) and the
shock multiplier (`spec-integration.md`). The seams to them are defined here.

**The numbers in this spec are placeholders.** Every bound, default and example
price below is a guess that nothing has run against (N-015, N-027). No default
here is claimed to be stable. Section 11 lists them.

**Revised in round 3 (2026-10-10).** Four changes. (1) `UnitCost` now counts the
recipe's labour at the observed wage (section 9, N-055); the signature gains a
duration and a wage. (2) The strategy names the recipe through `JobRequest`, a
request off the role's list is refused with `not_on_role`, and `Role.Recipes` is
owned by `spec-agents.md` 3.3 (3.3, 4.1, 4.2, 14.2). (3) The two worked examples
are rescaled and completed so that they pass the viability checks of
`spec-viability.md` (section 12, N-057). (4) The open questions of section 18 that
the new specs answer are answered.

**Specs that must be re-checked after this revision.** `spec-agents.md` 7.3.6 passes
the cost anchor of 9.3 to the belief rule: that value now includes labour and is 0
(unknown) in a region that has no observed wage yet, so a role's first ticks have no
cost band. `spec-viability.md` 8.2.3 checks seeds against the cost band: with labour
counted at the solver's wage the seed over the unit cost is 1.00 for every active
activity of the prototype (section 9.5 here), so `SeedAboveCostCeiling` should no
longer fire, and the solver should also supply the starting wage (9.3). Stability
register entries STB-L02, STB-L09, STB-L10 and STB-L52 describe the floor, the
ceiling and the evidence offset against the old `UnitCost` and the old `CostFloor`.
`spec-welfare-and-wages.md` 4.3 must expose the pooled regional wage of 9.3 (its income already nets inputs and tool wear, 2.1).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

A **recipe** says how goods are turned into other goods by an agent's labour. This
spec defines what a recipe is as data, how an agent starts and finishes one, how
long it takes, what a region does to its yield, what a tool is for, and what
production exposes to price beliefs. It covers:

- the recipe format (section 3);
- how an execution starts, how many run, and what is consumed (section 4);
- multi-tick **jobs**, where unfinished work lives, and what interrupts it (section 5);
- **yield**: a region's and a season's effect on output, with no second recipe per
  region (section 6);
- byproducts, waste and disposal (section 7);
- trainee-eligible recipes (section 8);
- the production cost exposed to `spec-beliefs.md` (section 9);
- validation (section 10), parameters (section 11), and two worked examples in
  unrelated settings (section 12).

Out of scope: which recipe an agent chooses each tick (`spec-agents.md`, the
`Strategy`), who may enter a role (`spec-skill-barriers.md`, `spec-spawner.md`),
the regional endowment's own dynamics (`spec-resources.md`), shocks and who injects
them (`spec-integration.md`), fixed capital such as a mill (`spec-infrastructure.md`),
and labour as a good (`spec-goods.md` 7.3).

## 2. Positions taken

Each design pressure of the brief, with the position and the reason. Later sections
give the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | How expressive is the format, and is there a production hook? | Lists of inputs and outputs, a duration, required tools, endowment draws, a cost share and a discard flag per output, and flags for sites and trainees. **No hook.** Everything a rule needs is a field | Vision goal 5: client code inside production makes determinism and conservation harder to guarantee. Every case in section 12 fits the fields |
| 2 | How does a region or a season change yield without a second recipe per region? | A table of basis-point **yield factors** keyed by region, recipe and season, read when a job starts. 0 means the recipe cannot run there. RES fills it in later | N-030: climate is data. One recipe plus one table row per region keeps recipes few and lets a pack add a region without touching a recipe |
| 3 | What is an execution? | One all-or-nothing ledger transform. A recipe with no inputs is a source. An agent of weight `w` runs up to `w` at once | `spec-ledger.md` 8.1, 8.2; `spec-scale.md` 4.2 |
| 4 | Where does multi-tick work live? | In the agent, as one **job**: inputs are consumed at the start, outputs are created at the end, and in between the agent is busy. One job per agent | Keeps the ledger free of half-made goods; one household can do one thing at a time |
| 5 | Non-tradable goods, waste, disposal (N-003)? | **No `Tradable` flag.** An output may be marked **discard**: it is created and disposed in the same step, so it is measured and never held. A `disposal` sink is added to the ledger registry | A flag would add a clearing reject reason and a GDS field for one case; `Discard` covers waste, and unwanted tradable goods just fall to a low price |
| 6 | Variable yield and conditions? | Version 1: a deterministic yield factor by region and season, a shock multiplier for the integration spec, and the endowment limit. Deferred: random yield, conditions on a building, per-output factors, a harvest window at completion | See section 6.7 for what each deferral costs |
| 7 | Trainee-eligible recipes and output size? | A flag on the recipe, no tools allowed on it, and a `TraineeYield` factor (placeholder 5 000 basis points) that scales a trainee's output | It gives the author the lever that decides whether training pays; the answer for the reference chain is a calibration (section 8.4) |
| 8 | What does production add to validation? | Section 10, including a yield-table check. The N-031 worry (a recipe longer than its season) does not arise, because yield is read when a job starts (6.4) | |
| 9 | Is the format adaptable? | Two unrelated settings are written in section 12 with the same fields | Vision section 5, item 3 |
| 10 | Does production need the chance-event method of N-005? | **No chance at all.** A yield fraction is carried as a deterministic remainder at every weight, started at a seeded offset | The expected count is exact and uses no trials; N-005 is settled for production (section 6.6) |
| 11 | Does the cost anchor include labour? | **Yes, at the observed wage of the region, measured over the window that ended the previous tick.** A recipe's labour is `Duration` working ticks per execution. An unknown wage makes the whole unit cost unknown (0) | Without labour the zero-profit price of a chain is 3.7 to 19 times the unit cost (`spec-viability.md` 15), so the belief ceiling would sit far below the equilibrium price (N-055). The lag removes the circle: this tick's cost does not depend on this tick's prices. Section 9 |

## 3. The recipe

### 3.1 Fields

A `Recipe` is a plain struct in the root package, an entry of `Config.Recipes`
(`spec-config.md` 3.1). In a file it is `[recipe.<id>]` (`spec-config.md` 6.2).

| Field | Type | Meaning |
|-------|------|---------|
| `ID` | string | the recipe's ID (`spec-config.md` 3.3) |
| `Duration` | ticks | how many ticks one job takes (section 5); at least 1 |
| `Inputs` | list of (commodity, quantity) | consumed when a job starts, per execution |
| `Outputs` | list of output items (3.2) | created when a job completes, per execution |
| `Requires` | list of (commodity, `WearPerUse`) | tools: durables the executor must hold, each worn by use (section 4.4) |
| `Draws` | list of (resource, quantity) | endowment the recipe takes from its region, per execution (section 6.5); the resource is a `Resource` of `spec-resources.md` 3; module RES |
| `Needs` | list of site class IDs | facility classes of kind `site` the principal must hold a usable facility of in its region; at most `MaxRecipeNeeds` (module CAP, `spec-infrastructure.md` 4.1) |
| `Builds` | clause (`Class`, `Toward`) | the completion creates facilities of the class (module CAP, `spec-infrastructure.md` 4.1) |
| `Repairs` | clause (`Class`, `Units`) | the completion restores condition to facilities the principal owns (module CAP, `spec-infrastructure.md` 4.1) |
| `Sited` | bool | the recipe runs only where the yield table gives it a positive factor (6.2) |
| `TraineeEligible` | bool | a trainee may run it (section 8) |
| `TraineeYield` | basis points | a trainee's output as a share of a full execution (8.2) |
| `Label` | string | presentation only; not hashed (`spec-goods.md` 6.5 applies) |

3.1.1. `Inputs` and `Outputs` are lists of items. In Go they are `[]Item` and `[]Output`, and `Requires` is `[]Tool` (`spec-config.md` 3.7.2; no struct embeds another, so `Output` repeats the two fields of `Item`). Within one list a commodity
appears at most once. Each list is sorted by commodity in canonical order
(`spec-config.md` 3.4) and has at most `MaxRecipeItems` items (16). A quantity is
an integer from 1 to `MaxQuantity`. A commodity may appear in both lists: seed
grain that makes more grain is legal, and whether such a cycle sustains itself is
`spec-viability.md`'s.

3.1.2. A recipe with no `Inputs` is a **source**: it is how a gathered raw good
enters (`spec-goods.md` 6.2; `spec-ledger.md` 8.2). It is limited by labour (at most
`w` executions per job), by its tools, and by its endowment if it has `Draws`.
Nothing else bounds a source, so a source with no `Draws` and no `Sited` flag is an
unlimited faucet of labour output. That is legal and is the author's choice.

3.1.3. A recipe names no role, region, commodity class or setting. Which roles run
it is data on the role (3.3). Nothing in production branches on a recipe,
commodity or role identifier (architecture.md 9.1).

3.1.4. The three clauses `Needs`, `Builds` and `Repairs` are data like every other
field. Their values and their rules are `spec-infrastructure.md` 4.1 (a recipe has at
most one of `Builds` and `Repairs`, and a recipe with either has at least one input);
this spec keeps the format and the sentinels (10.2). With the infrastructure module off a
recipe with any of them is `ErrModuleDependency`.

### 3.2 Output items

An output item is a commodity, a quantity and two more fields:

| Field | Meaning | Default |
|-------|---------|---------|
| `CostShare` | the share of the recipe's input cost allocated to this output, in basis points (section 9) | 10 000 if the recipe has one non-discard output, otherwise **required** |
| `Discard` | the output is created and disposed in the same step (section 7.2) | false |

3.2.1. The `CostShare` of the non-discard outputs adds up to exactly 10 000. The
`CostShare` of a discard output is 0. A recipe with `Builds` or `Repairs` has no
non-discard output (3.2.2), so there is nothing to add up: its `CostShare`s are all 0 and
`ErrCostShares` is not raised for it.

3.2.2. A **byproduct** is not a separate kind of item. It is an output with a
`CostShare` of 0 that is not discarded: it is made by the same execution and is
sold if it can be, and no cost is allocated to it. An output with a positive
`CostShare` is a **co-product**. The engine has no code path that tells them apart
except the cost allocation of section 9. A recipe needs at least one non-discard
output (`ErrRecipeNoOutput`, `spec-config.md` 7.3), unless it has `Builds` or `Repairs`
(`spec-infrastructure.md` 4.1.2 B3): such a recipe creates facilities or restores
condition instead, and an output it has is a discard output.

### 3.3 Which agents can run a recipe

A role lists the recipes its agents may start (`Role.Recipes`, a list of recipe IDs,
sorted; the field is owned by `spec-agents.md` 3.3, which also says that an empty
list is legal). A role with an empty list has no recipes (a trader, a carrier). An
agent starts only a recipe on its role's list or on the role's effective own-use list
(`spec-subsistence.md` 3.3; FR-PRO-008); a request for any other
is refused with `not_on_role` (4.2). An own-use recipe is an ordinary recipe named on that
list (data, no flag on the recipe): it has no inputs and no `Requires`, and it is `Sited`
(`spec-subsistence.md` 3.4; a sited recipe runs only where the yield table lists it, 6.2).
It is not on `Role.Recipes`, and it sets no cost anchor (9.3). A trainee also runs the trainee-eligible recipes of its trainer's
role, for its trainer (section 8). A worker of the employment module also runs, for its
employer, a recipe on the hire list of the employer's role (`Role.Employer.Recipes`,
`spec-employment.md` 3.2); the request is made by the labour round, not by `Work`. A tool-less variant and a tool variant of the
same work are two recipes, and a role lists both; which to start is the strategy's
choice (4.1). This is how bazaarBot's "with tools yields 4, without yields 2"
(`bazaarbot-reference.md` 10) is written: two recipes, not a conditional.

## 4. Starting an execution

### 4.1 Who decides

Production is a mechanism; the **decision** to start a recipe belongs to the
agent's `Strategy` (`spec-agents.md` 6.2). The strategy's `Work` method returns a
`JobRequest` (`spec-agents.md` 6.3): the recipe, an optional maximum number of
executions (0 means as many as possible), or `NoJob`. Production checks whether the
request can start and how many executions, and starts them or reports a reason. The
default strategy's choice rule (the eligible recipe with the greatest net value per
tick of labour, at the agent's belief centres) is `spec-agents.md` 7.1.4, not this
spec's. Whether a resting agent (the target-income rule,
`spec-welfare-and-wages.md` section 5) starts is the same decision, taken only when
the agent has no active job (5.2): `Work` is not called for an agent that continued
or completed a job in the tick, nor for one that is away, on a migration journey or
teaching (`spec-world.md` P2.5).

### 4.2 How many executions

For an agent (the **executor**) of weight `w`, a recipe starts `k` executions, where
`k` is the least of:

- `w` (`spec-scale.md` 4.2), or the strategy's maximum if lower;
- for each input, the available stock divided by the input quantity, rounded down;
- for each `Requires` entry, the number of unreserved tool items of that commodity;
- for each `Draws` entry, the executions granted by the ration of `spec-resources.md` 5.3 (6.5);
- for each `Needs` class, the capacity of the principal's facilities of the class in its region, less the executions the same principal has already started or committed to the class in the tick (`spec-infrastructure.md` 4.2.2), evaluated again at P2.7;
- for a `Builds` or `Repairs` recipe, the gate, the room for facilities, the target, the store and what there is to repair (`spec-infrastructure.md` 4.2.2, limits 2 to 6);
- with the tenure module on, the executions the principal can pay the rent for (`spec-land.md` 4.5, 5.4);
- with the storage module on, `k_room`, the executions whose output fits the principal's free room (`spec-storage.md` 5.3.1, 5.3.2), evaluated at P2.5 as part of `k0` and again at P2.7, so the first start of a tick leaves the room the later ones see.

Before any of this, the request is checked against the role: a recipe that is not on
the agent's role list or its effective own-use list (or, for a trainee, on its trainer's, or,
for a hired household, on its employer's hire list, checked at admission,
`spec-employment.md` 5.2) is refused with the reason `not_on_role`, which is the world's
check on a strategy (`spec-agents.md` 6.5.1) and starts nothing. A recipe that a right in
force bars is then refused with `not_permitted` (`spec-institutions.md` 5.2, 5.3), for the
executor, the principal or both as the right says (the principal of a trainee's or a hire's
job is tested as the executor is). Both checks come before `k` is derived.

If `k` is 0 nothing is consumed, and the reason is the first of `zero_yield` (the yield
factor is 0, section 6), `no_site` (a `Sited` recipe with no positive factor), `no_tool`,
`no_facility` (`spec-infrastructure.md` 4.2.2), `no_input`, `no_endowment`, `no_title` and
`rent_funds` (`spec-land.md` 4.5.1), `not_unlocked`, `facility_cap`, `no_target`,
`store_refused` and `nothing_to_repair` (`spec-infrastructure.md` 4.2.2), and `no_room`
(`spec-storage.md` 5.3.1), in that order, `no_room` last. With the two checks of the
paragraph above in front, the whole **master order** of the reasons of a refused start is:
`not_on_role`, `not_permitted`, `zero_yield`, `no_site`, `no_tool`, `no_facility`,
`no_input`, `no_endowment`, `no_title`, `rent_funds`, `not_unlocked`, `facility_cap`,
`no_target`, `store_refused`, `nothing_to_repair`, `no_room`. This spec owns it and every
other file quotes its own slice and cites this one; a reason of a module that is off
cannot occur. This is the partial result of `spec-scale.md` 4.2: `k` households work and
the rest could not. A partial `k` is not an error and not a warning (FR-PRO-009).

### 4.3 The start step

4.3.1. At the start, one ledger `Transform` takes `k` times each input from the
executor's account to the `recipe_input` boundary and creates nothing
(`spec-ledger.md` 8.1). The transform is all or nothing; a refused transform leaves
the ledger unchanged and the job does not start (FR-PRO-010). Inputs are taken
oldest first, as for any move (`spec-ledger.md` 5.2.1). Production runs before
orders are placed in the tick, so no reservation exists and available is the whole
balance. If the world ever places orders first, available excludes reserved
(`spec-ledger.md` 6.3).

4.3.2. An input of a trainee's execution is taken from its **principal** (8.1), the
trainer, or, for a hired execution, the employer (`spec-employment.md` 6.1).

4.3.3. Money is not spent by production. Wages, fees and the price of inputs are
trades (`spec-clearing.md`); a recipe has no cash cost. (A hire's wage is paid by the
labour round from its escrow, `spec-employment.md` 7, not by production, and a rent is a
transfer of the start, `spec-land.md` 5.3.)

4.3.4. **The pledge.** With the storage module on, after the start's `Transform` the world
sets the principal's pledge to the exact load of the job's pending output plus the inbound
cargo (`spec-storage.md` 4.3, 5.3.3). A discard output, and the part of an output that
`MaxQuantity` disposes, take none, because they are never held (7.2, 5.3.2).

### 4.4 Tools and wear

4.4.1. A `Requires` entry is a commodity with `Durability` above 0 and a
`WearPerUse` from 1 to its `Durability`. The executor must hold `k` unreserved items
of it (for a hired execution the principal holds them and they wear:
`spec-employment.md` 6.1.2): each household of the cohort has its own tool (`spec-scale.md` 4.2,
FR-SCL-007).

4.4.2. At the start, each of the `k` executions wears a **distinct** item of each
required tool by `WearPerUse`, taking the `k` most worn items first
(`spec-ledger.md` 5.4.1). An item whose durability reaches 0 is destroyed, as a sink
with the reason `wear`. Wear happens at the start and is not repeated for the
duration of the job. The ledger today applies wear to one item at a time and does
not spread `k` hits over `k` items, so production asks for one more operation,
`WearItems(account, commodity, n, units)` (section 15 and the handoff).

4.4.3. A tool is not consumed and is not an input. It never appears in `Inputs`. A
durable that no recipe requires never wears, and validation warns (FR-GDS-023,
FR-PRO-040).

4.4.4. A tool's price is whatever the market sets (`spec-skill-barriers.md` 2.4).
Production fixes only the wear per use, which with `Durability` fixes the wear cost
used in section 9.

4.4.5. **A facility a recipe `Needs` is worn at the position of a tool**, at the start,
after `k` is final and before the inputs are taken (`spec-infrastructure.md` 5.1.3): the
`k` executions are assigned to the principal's facilities of the class in its region,
most worn first by the condition of the effect table, each taking at most its `cap(f)`
less the executions already assigned to it in the tick, and each facility loses its
executions times the class's `WearPerUse`. A facility is not an item of an account and is
not in `Requires`; the two clauses are separate (`spec-infrastructure.md` 4.1).

## 5. Duration and jobs

### 5.1 The job

5.1.1. A recipe has a `Duration` of `d` ticks, 1 or more. A started batch of `k`
executions is one **job**. A job is state of its executor:

```go
// Job is the one unit of work an agent has in progress. Plain data.
type Job struct {
	Recipe    int     // index into the canonical recipe list
	Remaining int64   // ticks left, 1 or more while active
	Executions int64  // k
	Principal int     // the agent whose goods were used and who receives the output
	Pending   []Item  // output quantities fixed at the start, per non-discard and discard output, in output order
}
```

The pending quantities are fixed when the job starts (yield is read then, 6.4) and
saved with the job (section 15).

5.1.2. **One job at a time.** An agent has at most one active job. An agent is a
household or a cohort of identical households (`spec-scale.md` 2); a household does
one task at a time, and the `w` households of a cohort do the same task in
lockstep. A recipe of duration 5 therefore gives an output once in five ticks, not
every tick. A pipeline of overlapping jobs would multiply the labour that the model
says one household has.

### 5.2 Timing

In each tick the production phase runs for each agent, in the order of 5.5:

1. If the agent has an active job, it **continues**: `Remaining` falls by 1. If it
   reaches 0 the job **completes** in this phase (5.3). The agent cannot start
   another job in the tick it continued.
2. Otherwise, the agent may **start** a job (section 4). The starts of a tick are resolved in two passes whatever modules are on (`spec-world.md` P2.5 to P2.7, ruling C9): `Work` for every free agent and then the starts, with a ration of scarce endowments between them (5.5.1). The job starts with
   `Remaining` equal to its `Duration`, and then takes this tick's step of (1): so a
   job of duration 1 starts and completes in the same phase, and a job of duration
   `d` started in tick `s` completes in the production phase of tick `s + d - 1`.

So a job occupies exactly `d` ticks, `s` to `s + d - 1`, and a new job can begin in
tick `s + d`. A duration of 1 is bazaarBot's instant production: outputs exist in
time to be offered in the same tick (`bazaarbot-reference.md` 4). Output created at
completion is available to the orders placed after the production phase.

5.2.1. **Every tick of a job is a working tick.** The start tick, each tick in
between and the completing tick count. `spec-welfare-and-wages.md` 4.3 now says so
("had a job in progress in the tick (started, continued or completed)"): the event
comes once at the end of the job, so the root reads working ticks from a per-agent
scratch set at P2.1 (continued or completed) and at P2.7 (started), not from the event
and not from the job, which is cleared at completion before the record is pushed at P6.3
(N-035; `spec-world.md` P6.3). For a job of `k` executions the working
household-ticks of each tick are `k`.

5.2.2. **A job is not interruptible by choice.** The target-income rule decides to
work or rest when the agent has no job. Once a job has started it runs to the end,
whatever the agent's welfare ratio. So the days an agent works come in blocks of at
least `Duration`, and it can overshoot its target by up to `Duration - 1` ticks. It
is a delay in the labour loop (section 16).

5.2.3. During a job the agent still has upkeep and still trades. Its inputs are
sunk, so a long job is a long period with cost and no sales. An agent that cannot
cover its upkeep over that period goes bankrupt (5.4); that is intended. The welfare
window of such a role is derived to be longer than its longest job
(`spec-welfare-and-wages.md` 2.3.3, N-032), and section 16 records the interaction.

### 5.3 Completion

5.3.1. At completion one ledger `Transform` creates the pending outputs in the
principal's account from the `recipe_output` boundary and takes no inputs
(`spec-ledger.md` 8.2). Each output lot has `Born` equal to the completing tick if
the commodity spoils, and a durable has full durability (`spec-ledger.md` 5.2.3). A
discard output is created and moved to the `disposal` boundary in the same step
(7.2). A recipe with no output at all (a `Builds` or `Repairs` recipe may have none,
3.2.2) makes no `Transform` at the completion: there is nothing to create, and a
`Transform` with both sides empty is refused (`spec-ledger.md` 8.2.1). With the storage
module on the outputs are credited from the pledge up to `RoomLeft`
(`spec-storage.md` 5.3.4): the room for them was decided at the start, so the credit
cannot fail for room.

5.3.2. If creating an output would push the principal's quantity of that commodity
above `MaxQuantity`, the part that fits is created and the rest is created and
disposed in the same step (reason `disposal`), and the event reports the amount
(FR-PRO-029). Production never fails and never holds a job open for lack of room.
With the storage module on, room is decided at the start (4.2, `k_room`), so a full
inventory is a refused start (`no_room`), not a lost output. The one exception is a cut
of the room while the job runs (`spec-storage.md` 5.5.3): the outputs are then credited in
commodity order, each up to `RoomLeft`, whole units, and the rest is created and disposed
in the same step under `disposal` and reported in `RecipeExecuted` as the `MaxQuantity`
overflow is (FR-PRO-029). N-002's update stands with that one exception.

5.3.3. `RecipeExecuted` is emitted at completion (section 14).

5.3.4. **A recipe with `Builds` or `Repairs`.** At the completion the world creates `k`
facilities (a `Builds`) or restores `k * Units` of condition (a `Repairs`), as
`spec-infrastructure.md` 4.3 says, besides the ordinary completion of the recipe's
outputs, which are all discard. `RecipeExecuted` is emitted as for any job with no
output row for the clause, and then the `FacilityChanged` events of
`spec-infrastructure.md` 14.1. The effects begin at the next P8.3b
(`spec-infrastructure.md` 4.3.3).

### 5.4 Interruption, bankruptcy and death

Inputs were consumed at the start and are already in the `recipe_input` boundary.
Nothing in the job is a ledger holding, so ending a job early moves no goods.

| Event | What happens to the job | Reason in the event |
|-------|-------------------------|---------------------|
| The executor goes bankrupt or dies | The job is **abandoned**: no output is created. The estate takes the agent's goods as usual (`spec-estates.md` 2.1). | `agent_removed` |
| The executor changes role, including a trainee at the completion of its traineeship | Abandoned. Production offers no way to carry a job across roles. At a completion the traineeship also ends, and this row, not the row of `principal_left`, applies: the order is `RecipeAbandoned` (`role_changed`), `TraineeshipCompleted`, then the `RoleChanged` it causes (`spec-world.md` 6.11) | `role_changed` |
| The executor's account changes region (`spec-regions.md` 6) | Abandoned, since the sited recipe and the yield of the new region differ. A self-carried trip cannot start while a job is in progress (`spec-transport.md` 5.4, `job_in_progress`), so the case does not arise for a traveller; it does for a migrant: the migration module abandons the job in progress at the departure of the journey, with this reason, before the account changes region at the arrival (`spec-migration.md` 8.2). An own-use job (`spec-subsistence.md` 5.8) is abandoned the same way, and so is a hired job when its worker moves (the hire ends with the cause `worker_moved`, `spec-employment.md` 6.4) | `region_changed` |
| The principal of a trainee's or a hired job leaves (the trainer is removed, or the traineeship ends without completing; the employer is removed or changes role, `spec-employment.md` 6.4) | Abandoned. A trainer's removal abandons its trainees' jobs first, in ascending agent ID, before the trainer's own events (`spec-world.md` 6.11 item 8; `spec-observability.md` 4.4.8). A completion is not this row (see the row above) | `principal_left` |
| The executor's weight falls from `w` to `w'` | Not abandoned. `Executions` becomes `min(k, w')` and each pending quantity is multiplied by the new `Executions` over the old, rounded down; the loss is not a ledger flow because the goods were never created. With the storage module on the world recomputes the pledge from the scaled pending output; the pledge is not scaled with the room and may exceed it, which 5.3.2 resolves at the completion (`spec-storage.md` 5.5.3) | none (a `RecipeExecuted` shows the smaller output) |
| The executor's weight rises | The job is unchanged; new households did not start it | none |
| A checkpoint is taken | Nothing; the job is saved (section 15) | none |

5.4.1. An abandoned job is a loss of the inputs' value. It is the right cost: work
in progress is worth something only if it is finished. Handing the pending output to
the estate would create goods that nobody made.

5.4.2. `RecipeAbandoned` is emitted (section 14).

### 5.5 Order of processing

5.5.1. Agents are processed in ID order starting from the agent whose position in
the region's agent list equals the tick number modulo the count of agents in the
region, then wrapping. The order matters only when agents compete for the same
resource: a regional endowment (6.5) or a trainer's inputs. A fixed order would give
the lowest IDs the whole endowment every tick, indefinitely. The rotation is
deterministic and gives each agent the first turn equally often. It uses no random
number.

**Revised by `spec-resources.md` (wave 3).** The rotation orders the starts and no longer arbitrates a scarce endowment: it gave fairness only over a long run, and in a model a household of weight 1 was denied for a whole year at half supply (`spec-resources.md` 17.2). The start phase is two-pass whatever modules are on (`spec-world.md` ruling C9): (1) the stocks are snapshotted after the continuations and completions; (2) `Work` is called for every free agent and each request gets `k0`, the executions it could start ignoring draws; (3) for each endowment that some request draws on, in (`RegionID`, `ResourceID`) order, the requests are rationed in proportion to what they ask, leftovers by largest remainder then a stateless hash key of (tick, agent) (`spec-resources.md` 5.3); (4) the starts proceed in the rotation order, each with `k` the least of `k0` and the grants, taking its draws whole or not at all. With the resources module off, or for a recipe with no draw, step (3) changes nothing and each request keeps its `k0`. The results of `Work` do not depend on the order of the calls (`spec-resources.md` FR-RES-015).

5.5.2. Completion and continuation of jobs, for all agents, run before any
starts, so a job that completes this tick cannot make an input available to a start
in the same pass out of order. (An agent that completes cannot start in the same
tick, 5.2, so this is only about other agents' goods: a completing seller's output
is not yet sold, since trading is after production.)

## 6. Yield: region, season and the endowment

### 6.1 The yield factor

The **yield factor** of a job is a basis-point multiplier on every output of the
recipe, read when the job starts. 10 000 is the recipe as written; 0 means the
recipe cannot run; a larger value is a richer site. The factor comes from a table
keyed by region, recipe and season. The table is data of the configuration. **The values are owned by `spec-resources.md` 7** (N-030): what a value means, how a pack states a climate, and the checks that read the whole table. This spec keeps the type, the lookup (6.2), the effective factor (6.3) and the validation errors (10.2), and the table works with the resources module off.

```go
// YieldProfile is a region's yield for one recipe. Plain data.
type YieldProfile struct {
	Region  string
	Recipe  string
	HasAll  bool         // All applies
	All     int64        // basis points, in all seasons without a season entry
	Seasons []SeasonYield // sorted by Season ID; empty means no seasonal entries
}

type SeasonYield struct {
	Season string // a season ID of the Calendar
	BP     int64
}
```

A file writes it as `[region.<id>.yield.<recipe>]` with `all = N` and `seasons = {
<season id> = N }` (section 12). The key `all` is present exactly when `HasAll` is true (`spec-config.md` 6.2.2). `Config.Yields` is the list of profiles, sorted by
region and recipe. One profile per (region, recipe) (`ErrDuplicateID`).

### 6.2 The lookup

For the region of the executor, the recipe, and the season of the start tick
(`Calendar.At(tick).Season`, -1 for none; `spec-calendar.md` 4), the table gives:

1. the profile's season entry for that season, if there is a profile and the season
   has an entry; else
2. the profile's `All`, if it has one; else
3. 0, if the profile has season entries and none matches (a seasonal profile is
   complete: a season it does not list is a season the recipe cannot run in); else
4. if there is no profile: 10 000 for a recipe that is not `Sited`, and **0** for a
   `Sited` recipe.

With no seasons configured, `Season` is -1, rule 1 never applies, and a profile with
season entries is rejected (10.2).

`Sited` is the difference between a recipe that runs anywhere unless a region says
otherwise (baking) and one that runs only where a region says so (growing a crop,
extracting from a deposit). A pack that adds a region says nothing about baking and
lists each crop it can grow. There is no second recipe per region and no list of
regions on a recipe.

6.2.1. The seasonal profile is how climates differ (N-030). A region with the
opposite hemisphere lists the same seasons with the values reversed; a region with
two seasons where the world has four lists the four with repeated values; a region
where a crop cannot grow has no profile, or one with `all = 0`. No code names a
crop, a climate or a season (architecture.md 9.1).

### 6.3 The effective factor

The **effective factor** `E` of a job, in basis points, is computed with integer
arithmetic and truncation toward zero, in this order:

```
site  = table value (6.2)                           0 .. MaxYieldBP
E     = site * shock / 10 000                       shock: 6.8, default 10 000
E     = E * tech / 10 000                           tech: 10 000 plus the gain of the recipe's field in the executor's
                                                    region (`spec-technology.md` 5.1.1); 10 000 for a recipe in no field
E     = E * trainee / 10 000                        trainee: 10 000, or TraineeYield (8.2), or, for a hired job,
                                                    `Employer.HireYieldBP` (`spec-employment.md` 6.1), which replaces it
E     = E * abundance / 10 000                      abundance: `spec-resources.md` 4.7 (the least over the recipe's draws;
                                                    10 000 for a recipe with no draw or a flat curve)
```

The four truncations lose less than 1 basis point in each step; they are a rule
constant, not a tunable. `E = 0` means the job does not start (`zero_yield`,
`no_site` when the cause is a `Sited` recipe with no profile).

### 6.4 When the factor is read

The factor and the season are read **when the job starts**, and the output
quantities are fixed then. A job started in a season with a positive factor
completes whatever the season is at completion. This answers N-031 for production:

- A crop planted in a growing season and harvested two seasons later is one recipe
  with a long duration and a seasonal profile that is positive only in the planting
  season. A duration longer than the season is no problem.
- "A recipe whose production time exceeds its season cannot complete" would be true
  if the factor were read at completion; it is not, so validation need not detect it.
- A harvest window (a crop that must be taken in before it rots) is a **perishable
  intermediate**, not a condition: a `grow` recipe makes a standing crop with a
  `ShelfLife` equal to the window, and a `reap` recipe takes it. The cliff
  spoilage of `spec-goods.md` 4 does the rest. Section 12 shows it.

### 6.5 The endowment limit (module, `spec-resources.md`)

A recipe with `Draws` takes a quantity of a regional resource per execution, such as
a deposit's stock or a field's tilled area. The resource, its stock, its
regeneration and its depletion are `spec-resources.md`'s. Production defines only
the seam, two operations on a per-(region, resource) value that RES owns and defines in `spec-resources.md` 5.1 (typed IDs; `Endowment` is a public read, the take is internal to production and all-or-nothing):

```go
// Defined by RES; production calls them. Both are plain data operations on world state.
func (w *World) Endowment(region RegionID, resource ResourceID) int64        // available units, 0 or more
func (w *World) takeEndowment(region RegionID, resource ResourceID, n int64) error  // n <= Endowment, else ErrInsufficientEndowment
```

`k` is limited by the executions the ration grants (`spec-resources.md` 5.3; 4.2, 5.5.1); the start step
takes `k` times the draw quantity from the live stock, all draws or none. A draw is not a ledger flow: the endowment is not
a good in an account. RES must make its own depletion visible (an event, and a
regeneration flow), because the ledger will not see it. RES now makes its depletion visible (`spec-resources.md` 6.2, 14). While the resources module is off, a recipe
with `Draws` fails validation with `ErrModuleDependency`; with it on, an unknown resource is `ErrUnknownResource`. The limit is therefore the only thing in a recipe that depends on another
module, and it does so through a list that is empty by default.

### 6.6 Fractional yield without chance (N-005)

An output quantity of `q` at a factor `E` is `q * E / 10 000` of a unit per
execution, which is rarely whole. Production never draws a trial. Each agent keeps
a **residue** per (recipe, output item), an integer from 0 to 9 999 in units of 1/10
000 of a unit. For a job of `k` executions, for each output item:

```
raw      = k * q * E + residue
produced = raw / 10 000                  (integer division)
residue  = raw mod 10 000
```

The product `k * q * E` is computed with the checked 128-bit helper of
`spec-currency.md` 7, and the validator rejects a configuration in which `produced`
could exceed `MaxQuantity` (10.3).

Example: `k = 3`, `q = 1`, `E = 8 500`, starting residue 2 000. Tick one: `raw =
3 * 1 * 8 500 + 2 000 = 27 500`, produced 2, residue 7 500. Next job: `raw = 25 500 +
7 500 = 33 000`, produced 3, residue 3 000. Then 28 500 gives 2 (residue 8 500), then
34 000 gives 3 (residue 4 000). The mean is 2.55 per job, which is `3 * 0.85`
exactly in the long run, with no randomness.

6.6.1. At `E = 10 000` the residue never changes and `produced = k * q`: the
mechanism costs nothing for a recipe that has no yield variation.

6.6.2. **The starting residue is drawn once per agent.** When an agent is created
the world draws one integer in [0, 10 000) from its random source, whatever the
recipes of the agent's role (so the random stream does not depend on the
configuration of recipes, as in `spec-welfare-and-wages.md` 5.1), and sets every
residue of the agent to it. Without the draw, every agent would start at residue 0
and all would produce the same short and long years together, which would be a
lockstep that the market has to absorb (N-014). The draw is made in agent-creation
order, after the warm start is applied and after the agent's other creation draw (the
target ratio, `spec-agents.md` 3.5.2), in the order `spec-world.md` fixes (D10); it is one
source word of the world's draw (`spec-world.md` 7.2, ruling C8). A job created at
construction from a warm start has its pending output computed with residue 0, and the
agent's residues take the drawn value afterwards (ruling C4).

6.6.3. The method is the same at every weight, 1 included. A cohort of weight `w`
runs `k` up to `w` executions in one multiplication, so its cost is that of an agent
of weight 1. This is the answer to N-005 and to `spec-scale.md` open question 3 for
production: the carried remainder is used at every weight, because a deterministic
fraction is uniform across scales and a random trial per household is not needed. A
future event with a real per-household chance would use the helper of
`spec-scale.md` 4.6; production and the tool wear of `spec-skill-barriers.md` 2.1 do
not.

6.6.4. A residue is not a good. It is a number in the agent's state. When an agent
dies, its residue is dropped; the fraction of a unit it represents was never created.

### 6.7 What is deferred, and what it costs

| Deferred | What a pack does instead | Cost of the deferral |
|----------|--------------------------|----------------------|
| **Random yield** (Factorio's `probability`, `amount_min` and `amount_max`) | A mean yield through the factor and the residue (6.6) | No variance in a harvest from one agent to the next. A failed harvest is a shock (6.8), which is correlated and so more realistic for a region. A game that wants luck per agent adds it as a shock per agent through INT |
| **A condition on a building or other fixed capital** | Answered by `spec-infrastructure.md`: a facility class the recipe `Needs` (3.1), held by the principal in its region, with a capacity and wear by use and by time (4.4.5). A tool that stands for the building stays legal | A tool held by one miller cannot be shared, has no capacity and no location, which is why a facility is a separate clause and not an item |
| **A season condition at completion** | A perishable intermediate (6.4) | A harvest window needs two recipes and a perishable commodity instead of one condition |
| **A per-output factor** | One factor for every output; a recipe whose outputs vary differently is two recipes | A harvest where grain and straw vary independently cannot be one recipe. They normally move together |
| **A recipe that changes over time** (technology) | A pack replaces a recipe at construction | Productivity growth is `spec-technology.md`; recipes are fixed for a run |
| **Quality grades** | A grade is a separate commodity (`spec-goods.md` 7.1); the format assumes none | A recipe cannot make a better product from a better input except by being a different recipe |
| **A `Tradable` flag** | `Discard` for waste (7.2) | An author cannot forbid selling a good |
| **Overlapping jobs per agent** | One job at a time (5.1.2) | A role that really does several things at once (a farm worker tending two crops) is a cohort of weight above 1 or two roles |

### 6.8 The shock multiplier (seam for `spec-integration.md`)

The world holds a list of **yield shocks**, each a region, a recipe and a
basis-point multiplier (default 10 000, meaning none). A shock multiplies the site
factor in 6.3. The failed harvest of the vision (section 2) is a shock of 3 000 on
the crops of a region; it has no expiry or timer: the public call is `SetYieldShocks`
(`spec-integration.md` 5.2.2), the consumer sets and clears shocks between ticks. Production
owns the slot and the arithmetic; it never creates a shock. The list is saved in the checkpoint (section 15), sorted by region and recipe.
A shock multiplier is from 0 to `MaxYieldBP`.

The yield shock is one of several **world slots set between ticks**, each state saved in the
checkpoint and each given its public call by `spec-integration.md`: the edge status of
`spec-regions.md` 7, the transit factor of `spec-transport.md` 10.3, the scale of a money flow
(`spec-money-supply.md`) and the effective birth and death scales of a region
(`spec-population.md` 6.6, 11), and the capacity and regeneration scales and the one-off adjustment of an endowment (`spec-resources.md` 6.3). None edits a configured value.

## 7. Byproducts, waste and disposal (N-003)

7.1. **A byproduct is an output with no cost share** (3.2.2). A bran-like residue is
an item the executor holds and may offer. If nobody wants it, its price belief falls
to the floor, and it accumulates in stock until it spoils, if its commodity has a
`ShelfLife`, or until the executor's strategy discards it. Nothing in production
forces a market for it.

7.2. **A discard output** is for waste that has no use or no market: slag, ash,
exhaust. At completion it is created from the `recipe_output` boundary in the
executor's account and at once moved to the `disposal` boundary, in one
transform step. It never sits in inventory and never reaches a market. It is
measured in the flow totals (`recipe_output` and `disposal`, by commodity), so an
indicator can report waste produced (`Disp` in `Written` and in the `Disposed` of the
recipe table, `spec-indicators.md` 7.2.1), which is how a pollution measure would be built.
Work in progress is `Pending` there.
Its commodity needs a reference price like any commodity (`spec-goods.md` 6.3) but
the price is never used.

7.3. **The `disposal` reason** is a sink of goods, agent to boundary, owned by this
spec (`spec-ledger.md` 7.2 gains the row). It is used by discard outputs, by the
overflow rule of 5.3.2, and is available to `spec-agents.md` for an agent that
discards stock. The decision to discard stock is the strategy's (N-002).

7.4. **There is no `Tradable` flag** and no clearing rejection for it. A commodity
that a pack wants kept out of a market is not offered by any strategy and has no
buyer. A tool bound to its holder is handled by the skill-barriers rule
(`spec-skill-barriers.md` 2.7). If a case appears where a good must be forbidden from
trade by rule, add `Tradable` to `spec-goods.md` and a reject reason to
`spec-clearing.md` 3.2 then.

7.5. Resolution for `spec-goods.md` open question 4: byproducts and waste need no
attribute; the flag is rejected for version 1 with the reasons above.

## 8. Trainee-eligible recipes

8.1. A recipe with `TraineeEligible` may be run by a **trainee** (`spec-skill-barriers.md`
4.1) on behalf of its trainer. The trainee is the **executor**: its labour, its
working ticks, its weight bound `k`. The trainer is the **principal**: the inputs are
taken from the trainer's account and the output goes to it (4.3.2, 5.3.1,
`spec-skill-barriers.md` 4.2). The recipe must be on the trainer's role list (3.3). A
trainee-eligible recipe MUST NOT have `Requires`: a trainee needs "neither the tool
nor the skill", so a recipe that needs a tool is not eligible. It has none of `Needs`, `Builds` or
`Repairs` either (`spec-infrastructure.md` 4.1.2 B7): a facility is not divisible and a
trainee's job draws on its trainer's account, not on the trainer's site. A recipe on an
employer's hire list may require a tool: the principal holds it
(`spec-employment.md` 6.1.2); the rule above is about the trainee's own tool. An author who wants an
eligible variant of a tool recipe writes a second recipe without the tool and with a
smaller output (the `knead_dough` and `bake_loaf` of section 12.1 show the split).

8.2. **Trainee output.** `TraineeYield` is the share of a full execution that a
trainee produces, in basis points from 1 to 10 000, default 5 000. It is the factor
`trainee` of 6.3. The inputs consumed are unchanged: a trainee wastes material and
produces less. The default of 5 000 is a placeholder; the historical account that
trainees were not cheap labour (`evidence.md`, unverified) is the only support, and
no number in a source is used.

8.3. **What decides whether training pays.** The trainer takes a trainee only if
`gain >= 0` (`spec-skill-barriers.md` 4.3). In terms of this spec, with `value` the
market value of a full execution's output (the sum over outputs of quantity times
price), `input_cost` the market value of its inputs, and `T` the `TraineeYield`:

```
output_a = value * T / 10_000 - input_cost         per trainee-tick, for Duration 1
```

and `gain = output_a - U_a - TeachingLoad * W_trainer / 10_000` as in the skill-barriers
spec. The inequality gives, for a chain, the least `TraineeYield` at which training
pays; production exposes `TrainingPays(recipe, prices, upkeep, teachingCost) bool`
as a pure function over supplied values (the same kind of function as `UnitCost`,
9.4) for the entry rule and the calibration.

8.4. This settles the *lever*, not the *answer* of `specs/README.md`, "Still open" 1.
Which recipes of the reference chain are eligible and what `TraineeYield` the chain
uses are chain calibration, and wait for `spec-reference-chain.md` and
`spec-viability.md`. The reference chain must have at least one skilled role with an
eligible recipe whose `TraineeYield` is at or above the break-even of 8.3 and one
whose yield is below it, so that both outcomes are tested.

8.5. A trainee's job follows the same timing, weight and abandonment rules as any
job (section 5). If the traineeship ends while a job is running, the job is abandoned
(`principal_left`); at the completion of the traineeship it is abandoned with
`role_changed`, the trainee changing role (5.4).

## 9. The production cost exposed to price beliefs

`spec-beliefs.md` anchors a price to the cost of making the good, and receives that
cost as a plain value, `UnitCost`: money per unit of a commodity in minor units, an
`int64`, **0 meaning unknown or none** (so no anchor). Production computes it. It
reads no agent state and has no effect.

**Labour is counted (N-055).** The cost of a recipe is its inputs, the wear of its
tools **and its labour**: `Duration` working ticks of a household per execution,
valued at the observed wage of the region (9.3). Without labour, the zero-profit
price of a chain is 3.7 to 19 times the unit cost (`spec-viability.md` 15), so the
cost ceiling of `spec-beliefs.md` 3.2.2 would cap a seller below its equilibrium
price. With labour counted at the wage the solver computes, the price over the unit
cost is 1.00 for every active activity of the prototype (9.5).

### 9.1 For one recipe and one output

Given the prices of the commodities (a slice indexed by commodity, in minor units,
0 for unknown), the wage per working tick (9.3, 0 for unknown), a region and a
season:

```
input_cost  = sum over Inputs of  quantity * price                          (checked)
wear_cost   = sum over Requires of  ceil(WearPerUse * price_of_tool / Durability)
labour_cost = Duration * wage
rent_cost   = sum over Draws of  d * r(region, res)                         (`spec-land.md` 5.7.1; 0 with the tenure module off or a rent of 0)
cost        = input_cost + wear_cost + labour_cost + rent_cost
unit_cost   = ceil( cost * CostShare / (quantity * E) )                     (E from 6.3 for this region and season, trainee 10 000, shock, technology and abundance included: a thin stock makes a unit dearer, `spec-resources.md` 4.7.5)
```

With `CostShare` in basis points and `E` in basis points the factors of 10 000
cancel, as in the example below. The result is rounded **up**, so the anchor never
understates cost. The labour of an execution is `Duration` household-ticks whatever
the weight of the executor (`spec-viability.md` 3.1), so `wage` is per household.
A draw is priced at the posted rent of `spec-land.md` 5.2.2 when the tenure module is on
(`rent_cost`, `spec-land.md` 5.7.1: for every unit, a holder's own included, the rent being
the opportunity cost), and is not priced otherwise: the endowment is not bought, and a
scarce deposit then shows as a rent in the market price, not in the cost.

9.1.1. `unit_cost` is 0 (unknown) when: any input or tool price is 0; the wage is 0
or less; `E` is 0; `CostShare` is 0 (a byproduct or a discard output: no cost is
allocated); or the arithmetic would overflow. An unknown wage makes the whole unit
cost unknown rather than counting inputs alone: an input-only cost understates the
true cost by the labour share, which would put the ceiling below the price again.
The consequence is that, in a world with no observed wage (the first ticks of a run
with no warm-start wage), no commodity has a cost band and only the reference band
applies.

### 9.2 Worked example

A flour-milling recipe: 3 grain in, 2 flour out (cost share 10 000), a bran
byproduct (cost share 0), `Duration` 1, and a tool with `WearPerUse` 1 and
`Durability` 2 000. Grain costs 200, the tool 20 000 new, the wage is 300 per
working tick (placeholders):

```
input_cost  = 3 * 200 = 600
wear_cost   = ceil(1 * 20 000 / 2 000) = 10
labour_cost = 1 * 300 = 300
cost        = 910
```

At `E = 10 000`: `unit_cost(flour) = ceil(910 * 10 000 / (2 * 10 000)) = 455`. In a
poor region with `E = 8 000`: `ceil(910 * 10 000 / (2 * 8 000)) = ceil(568.75) = 569`.
The bran byproduct has `CostShare` 0, so its `unit_cost` is 0: no anchor. With an
unknown wage (0) the flour's `unit_cost` is 0 as well.

### 9.3 The wage, and the cost anchor of a commodity in a region

**The regional wage.** The wage per working tick of a region is the pooled observed
wage, which `spec-welfare-and-wages.md` 4.3 defines and owns: this spec has no formula
of its own for it (the figure of 4.3 excludes relief, grants and own-use receipts and
ticks), and it is read as it stood at the end of the previous tick. Income there already
nets the money spent on inputs (cash) and the wear of tools (accrual)
(`spec-welfare-and-wages.md` 2.1, 2.1.3, 2.1.4), so the observed wage is the return to
labour and the cost of 9.1 counts inputs and wear once. The window is each agent's
effective window, which is at least its longest job (2.3.3). The wage is 0 (unknown)
where `spec-welfare-and-wages.md` 4.3 says the pooled figure is unknown. Pooling over
the region keeps one noisy agent from moving the anchor, and using the previous tick's
value means that this tick's cost does not depend on this tick's prices, so there is no
circle. A warm start may supply the starting wage (`spec-viability.md` 8), used whenever
the pooled figure is unknown (a function of the welfare rings, not a latch;
`spec-checkpoint.md` F5).

The wage is the region's, not the role's: a skilled recipe is costed at the common
wage, with the skill premium left to the tool wear and the market (open question 10).
The own-use receipts and ticks of the subsistence module are left out of it
(`spec-welfare-and-wages.md` 4.3, `spec-subsistence.md` 7.3): the wage is the return to market
labour, so a region where households subsist does not pull the anchor toward 0.

**The anchor.** The cost anchor of a commodity in a region is the smallest
`unit_cost` over the recipes that make it and that can run in the region this tick
(`E` above 0 and some agent of the region has the recipe on its role's list),
ignoring those whose cost is 0. A recipe that is only on an own-use list never sets an
anchor (`spec-subsistence.md` 3.6): the cost of a household's own labour is not what a
seller's price is anchored to. It is 0 if there is none. Ties are impossible (a
number). The root computes the anchor of every (region, commodity) **once per tick**,
before orders, from the prices published by the previous clearing
(`spec-clearing.md` 5.4) and the regional wage above, into scratch held by the
world, and passes the value to the belief rule per agent and commodity. The table is
derived, not state (section 15).

### 9.4 The signature (for BEL)

In `internal/production`, over primitives only (architecture.md 2.9):

```go
// Term is a commodity index and a quantity.
type Term struct {
	Commodity int
	Quantity  int64
}

// Wear is a tool: commodity index, wear per use and full durability.
type Wear struct {
	Commodity  int
	WearPerUse int64
	Durability int64
}

// UnitCost returns the cost of one unit of an output, in minor units, rounded
// up, or 0 if unknown (section 9.1.1). prices is indexed by commodity index; 0
// means unknown. share and yieldBP are basis points; outQty is the output's
// quantity per execution; duration is the recipe's ticks and wage the money per
// working tick of one household; rent is the rent of the draws of one execution
// (`sum over Draws of d * r`, 0 with the tenure module off). It has no side effects and
// allocates nothing.
func UnitCost(inputs []Term, tools []Wear, prices []int64, share, outQty, yieldBP, duration, wage, rent int64) int64
```

The root's `UnitCost(region, commodity)` over a whole configuration, and
`TrainingPays`, are root-level wrappers that convert inline (architecture.md 2.9).
`spec-beliefs.md` takes `UnitCost int64` as a parameter and does not import
production.

### 9.5 What the numbers showed (N-055)

The prototype of `spec-viability.md` 15 (the rescaled bread chain, one region,
`BeliefStep` 200, seed prices from the exact duals, wage 2 980.5 per working tick),
re-run for this revision (`doc/specs/models/pb/cost.py`):

| Good | Seed | Input and wear | Labour per unit | Full cost | Seed over old cost | Seed over full cost |
|------|-----:|---------------:|----------------:|----------:|-------------------:|--------------------:|
| grain | 373 | 134.1 | 238.4 | 372.6 | 2.78 | 1.00 |
| flour | 342 | 93.4 | 248.4 | 341.8 | 3.66 | 1.00 |
| dough | 277 | 28.5 | 248.4 | 276.9 | 9.72 | 1.00 |
| loaf | 376 | 44.6 | 331.2 | 375.8 | 8.42 | 1.00 |
| hoe | 3 146 | 166.0 | 2 980.5 | 3 146.5 | 18.95 | 1.00 |

So with labour counted, the zero-profit price **equals** the unit cost. That is an
identity of the programme (value added equals wage times duration), and it moves the
difficulty from the ceiling to the floor: a `CostFloor` of 100% binds exactly at the
steady state. `spec-beliefs.md` 3.2.2 records the numeric check and lowers the
default to 7 000.

## 10. Validation

### 10.1 Bounds

Provisional, in `bounds.go` (`spec-config.md` 4.1), placeholders until the slice runs:

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxRecipeTicks` | 100 000 | a job's duration; equal to `MaxTicksPerYear` so a job of years is possible and a duration times a small count does not overflow |
| `MaxYieldBP` | 100 000 | a yield factor of 10 times; with `MaxWeight` (10^6) and a quantity it must still fit (10.3) |
| `MaxRequires` | 4 | tools of one recipe |
| `MaxDraws` | 4 | endowments of one recipe |
| `MaxRecipeNeeds` | 2 | site classes a recipe needs (`spec-infrastructure.md` 12.2) |

`MaxRecipeItems` (16 per side) and `MaxRecipes` are in `spec-config.md` 4.1.

### 10.2 Rules

Each has a sentinel and a failing-case test (testing.md 3.9). A path names the field,
for example `recipes.mill_flour.outputs[1].cost_share`.

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a `Duration` outside [1, `MaxRecipeTicks`]; a quantity outside [1, `MaxQuantity`]; a `WearPerUse` outside [1, `Durability`]; a yield outside [0, `MaxYieldBP`]; a `TraineeYield` outside [1, 10 000]; more than `MaxRequires`, `MaxDraws` or `MaxRecipeItems` entries |
| `ErrDuplicateID` | a commodity twice in one list of a recipe; a resource twice in the `Draws` of a recipe; two yield profiles for one region and recipe |
| `ErrRecipeNoOutput` | a recipe with no non-discard output, unless it has `Builds` or `Repairs` (waived, `spec-infrastructure.md` 4.1.2 B3) |
| `ErrCostShares` | non-discard `CostShare`s that do not add to 10 000, a discard output with a share above 0, or a missing share where there are several non-discard outputs; not raised for a recipe with `Builds` or `Repairs` and no non-discard output (the sum is 0, 3.2.1) |
| `ErrToolNotDurable` | a `Requires` entry for a commodity with `Durability` 0 |
| `ErrEligibleHasTool` | a `TraineeEligible` recipe with `Requires`, `Needs`, `Builds` or `Repairs` |
| `ErrFacilityShape`, `ErrFacilityNoCost`, `ErrUnknownFacilityClass`, `ErrModuleDependency` | the clauses `Needs`, `Builds` and `Repairs` (`spec-infrastructure.md` 13.1) |
| `ErrUnknownCommodity`, `ErrUnknownRecipe`, `ErrUnknownRegion` | the references in recipes, role lists and yield profiles |
| `ErrUnknownSeason` | a yield entry for a season ID the calendar lacks, including any season entry when the calendar has no seasons |
| `ErrModuleDependency` | `Draws` while the resources module is off |
| `ErrUnknownResource` | a `Draws` entry that names a resource the resources module does not define (`spec-resources.md` 13.2) |
| `ErrOverflow` | the derived products of 10.3 |

### 10.3 Derived checks (added to `spec-config.md` 4.2)

For every recipe, validation computes with checked arithmetic and rejects with
`ErrOverflow` if any fails:

- the largest weight a role's group can get (`spec-scale.md` 3.2) times each input
  quantity, and times each output quantity times the largest yield factor the
  configuration can give the recipe (the largest table entry, times the largest
  shock bound, times the largest technology factor `MaxTechBP` (`spec-technology.md` 5.1.4;
  10 000 for a recipe in no field), times the larger of the trainee factor and, for a
  recipe on an employer's hire list, `Employer.HireYieldBP` (up to `MaxYieldBP`), times
  the largest abundance factor, `MaxYieldBP`, of `spec-resources.md` 4.7) divided by 10 000, is at most `MaxQuantity`;
- the same weight times each draw quantity is at most `MaxQuantity`;
- the sum over inputs of quantity times the largest reference price is at most
  `MaxMoney` (so `UnitCost` cannot overflow).

### 10.4 Warnings

A warning never fails validation (`spec-config.md` 7.2). Each has a rule name and a path.

| Warning | Condition |
|---------|-----------|
| `unused_recipe` | no role lists the recipe, it is not trainee-eligible and it is on no role's effective own-use list (`spec-subsistence.md` 3.6) |
| `unrunnable_recipe` | a `Sited` recipe with no profile that has a positive factor in any region, or a recipe whose every profile entry is 0, or a recipe whose every draw exceeds the effective capacity of its endowment in every region where its site factor is positive (`spec-resources.md` 13.3, `draw_impossible`), or a recipe whose `Needs` class has no initial facility in a region where it is otherwise runnable and no recipe builds it (`spec-infrastructure.md` 13.2, `facility_unbuildable`) |
| `durable_not_required` | a durable that no recipe requires (FR-GDS-023) |
| `source_unbounded` | a source (no inputs) with no `Draws` and not `Sited`: limited by labour only (3.1.2); informational |

The warning for a role whose starting money cannot buy its tool is already in
`spec-skill-barriers.md` 2.3. The former warning `job_longer_than_window` is removed:
the welfare window of a role is derived to exceed its longest job
(`spec-welfare-and-wages.md` 2.3.3, 2.3.5), which reports `window_extended` instead; `spec-config.md` replaced the warning by it.


### 10.5 What is not checked here

Whether a cycle yields a surplus and whether a household's labour can make its
upkeep is `spec-viability.md` (`spec-config.md` 7.6). The producibility fixpoint is
`spec-config.md` 7.5 and treats a recipe's non-discard outputs as made and a
recipe's `Draws` as a source (a `Sited` source with `Draws` is sourced). A recipe
whose only way to start is a season that does not occur is the `unrunnable_recipe`
warning, not an error.

### 10.6 Authoring note: close every cycle and keep sources tool-free

Two traps found by building the reference chain (`spec-reference-chain.md` 6):

1. **A cycle needs a way in.** A chain in which a good is made only from itself through
   other goods (a seed cycle, a tool that is needed to make the tools it is made from) has
   no producible good by the fixpoint of `spec-config.md` 7.5 and fails V1. Give the cycle
   a source outside it: a recipe with no inputs (wild seed, salvage), or a good marked
   `Raw` with a real source later. The source may be poor; the economy will use its own
   cycle once it runs.
2. **A source recipe must be tool-free if the tool's chain needs the source.** A required
   tool is an input of the fixpoint. If `dig_ore` requires a pick, the pick is made from
   `bar`, `bar` from `ore`, and `ore` from `dig_ore`, then nothing in the loop can start.
   Either the source needs no tool or the tool has its own source. The validator names the
   cycle in its message (`spec-config.md` 7.5).

## 11. Parameters

All are fields of the recipe or of the yield table in the configuration, in the file
as `[recipe.<id>]` and `[region.<id>.yield.<recipe>]` (`spec-config.md` 6.2). The
defaults are those of `DefaultRecipe()`. **Every default and bound here is a
placeholder** (N-015, N-027).

| Scope | Field | Meaning | Default |
|-------|-------|---------|---------|
| Recipe | `Duration` | ticks one job takes (section 5); at least 1 | 1 |
| Recipe | `Sited` | runs only where the yield table gives a positive factor (6.2) | false |
| Recipe | `TraineeEligible` | a trainee may run it (section 8) | false |
| Recipe | `TraineeYield` | a trainee's share of a full execution's output, in basis points (8.2) | 5 000 |
| Tool | `WearPerUse` | wear units one execution puts on a tool (4.4) | 1 |
| Output | `CostShare` | share of the input cost allocated to the output, in basis points (3.2) | 10 000 if it is the only non-discard output, otherwise required |
| Output | `Discard` | created and disposed in one step (7.2) | false |
| Recipe | `Draws` | endowment taken per execution, list of (resource, quantity) (6.5); the resources are `spec-resources.md` 3 | none |
| Recipe | `Needs` | site classes the principal must hold a facility of (3.1); values and rules `spec-infrastructure.md` 4.1 | none |
| Recipe | `Builds` | the facility class (and region of a route) the completion creates (3.1) | none |
| Recipe | `Repairs` | the facility class and the units of condition the completion restores (3.1) | none |
| Config | `Yields` | the profiles: a basis-point factor by region, recipe and season (6.1; the fields are the rows of the scope `YieldProfile`); values owned by `spec-resources.md` 7 (format and lookup: 6.1, 6.2) | none (10 000, or 0 for a `Sited` recipe) |

`Inputs`, `Outputs` and `Requires` are lists with no default (`Outputs` is required,
`ErrRecipeNoOutput`, except for a recipe with `Builds` or `Repairs`). `Label` is presentation only.

The placeholders, listed so that none is mistaken for a result: `Duration` 1 (a
structural default, instant production); `TraineeYield` 5 000; the bounds
`MaxRecipeTicks` 100 000, `MaxYieldBP` 100 000, `MaxRequires` 4, `MaxDraws` 4; the
example prices and durabilities of section 12 and 9.2.

## 12. Two settings, one format

Vision section 5, item 3: a new chain needs a configuration file only, and this holds
for a second, unrelated setting. The core text above names no setting. The examples
below are **marked setting-specific content**, excerpts of packs, with placeholder
numbers. They use the file layout of `spec-config.md` 6.2 and the keys of this spec.

**Rescaled and completed in round 3 (N-057).** The first draft's quantities were not
viable: example A as written needs 73 times the labour its households have (a
120-tick job that yields 6 grain, against a household that eats several units a
tick), and example B 12 times (`spec-viability.md` 4.4, 15). The quantities below are
the set that passes the viability checks in that prototype. The recipes that complete
the chains (`save_seed`, `cut_wood`, `mine_iron`, the tool recipes in A; `melt`,
`dig_regolith`, `refine_wafer`, `refine_solder`, `make_drill`, `make_fabricator` in B)
are **invented here, with placeholder numbers, so that each pack is closed**; they
are marked in the excerpts. The rule of thumb from the runs: the output per labour
tick of each step of a chain must exceed the household's need by the depth of the
chain, so a farm job that lasts a season must yield hundreds of units, not a handful.

**Closed in round 5 (finding 1 of `spec-reference-chain.md` 6.1).** Both packs, as
first completed, had a cycle with no source outside it: in A the seed cycle (`grow_grain`
takes `seed`, `save_seed` makes it from `grain`) and in B the tool cycle (`fab_board`
needs a `fabricator`, `make_fabricator` needs boards). The viability prototype of round 3
did not run check V1 and so did not see it. Run in round 5 with V1 (Docker, exact
fractions): A as written fails V1 (`unproducible_in_realm`: seed, grain, flour, dough,
loaf), also with `seed` marked raw if the marker is not taken as a source (`spec-viability.md`
4.2 now takes it, with a warning); B as written fails V1 (fabricator, board, drill, ice, water,
biomass). Each pack now has a **source** that opens its cycle, marked below; with them V1
passes in both. The examples are checked by V1 to V4 from now on (FR-PRO-046).

"Passes viability" means `lambda*` of the reference programme is at least 1 with
every household working every tick (`spec-viability.md` 3.2): a necessary condition.
It does not show that a run is stable; no default here is claimed to be stable
(N-015, N-027).

### 12.1 Example A: a bread chain (a farming setting)

Calendar: the Earth default, so seasons are `spring`, `summer`, `autumn`, `winter`
(`spec-calendar.md` 7.4). Upkeep per household per tick: 2 `loaf` and 1 `fuel`;
1 000 households in `north`. Tools have a durability of 2 000. Excerpt, not a
complete valid pack (the commodity table is elided):

```toml
# A crop: planted in spring, ready 120 ticks later. The hoe is required. Straw is a
# byproduct (cost share 0). Seed is an input and, through save_seed, an output of the cycle.
[recipe.grow_grain]
duration = 120
sited = true
inputs   = [ { commodity = "seed", quantity = 60 } ]
outputs  = [ { commodity = "grain", quantity = 1500, cost_share = 10000 },
             { commodity = "straw", quantity = 100, cost_share = 0 } ]
requires = [ { commodity = "hoe", wear_per_use = 1 } ]

# Invented to close the seed cycle: one grain becomes one seed.
[recipe.save_seed]
inputs  = [ { commodity = "grain", quantity = 1 } ]
outputs = [ { commodity = "seed", quantity = 1, cost_share = 10000 } ]

# Invented (round 5): the source that opens the seed cycle. A cycle whose only way in is
# itself cannot start (`spec-config.md` 7.5); wild seed is gathered at a poor rate, so the
# farm saves its own seed and this recipe is the inactive alternative. It has no inputs and
# no profile, so it runs anywhere, and `source_unbounded` (10.4) is informational.
[recipe.gather_seed]
outputs = [ { commodity = "seed", quantity = 2, cost_share = 10000 } ]

# Grain to flour: bran is a byproduct; the millstone is a tool.
[recipe.mill_flour]
inputs   = [ { commodity = "grain", quantity = 3 } ]
outputs  = [ { commodity = "flour", quantity = 12, cost_share = 10000 },
             { commodity = "bran", quantity = 6, cost_share = 0 } ]
requires = [ { commodity = "millstone", wear_per_use = 1 } ]

# What a trainee baker can do: no tool, half the output (8.2).
[recipe.knead_dough]
trainee_eligible = true
trainee_yield = 5000
inputs  = [ { commodity = "flour", quantity = 1 } ]
outputs = [ { commodity = "dough", quantity = 12, cost_share = 10000 } ]

[recipe.bake_loaf]
duration = 2
inputs   = [ { commodity = "dough", quantity = 2 }, { commodity = "fuel", quantity = 1 } ]
outputs  = [ { commodity = "loaf", quantity = 18, cost_share = 10000 } ]
requires = [ { commodity = "oven", wear_per_use = 1 } ]

# Invented sources and tool recipes that complete the chain.
[recipe.cut_wood]
outputs = [ { commodity = "fuel", quantity = 12, cost_share = 10000 } ]
[recipe.mine_iron]
sited = true
outputs = [ { commodity = "iron", quantity = 18, cost_share = 10000 } ]
[recipe.make_hoe]
inputs  = [ { commodity = "iron", quantity = 1 } ]
outputs = [ { commodity = "hoe", quantity = 1, cost_share = 10000 } ]
[recipe.make_millstone]
inputs  = [ { commodity = "iron", quantity = 2 } ]
outputs = [ { commodity = "millstone", quantity = 1, cost_share = 10000 } ]
[recipe.make_oven]
inputs  = [ { commodity = "iron", quantity = 2 } ]
outputs = [ { commodity = "oven", quantity = 1, cost_share = 10000 } ]

# Climates (N-030): the same recipe, different regions. A northern region has one
# planting season; a southern one plants in spring and again in autumn at a lower yield;
# an arid region cannot grow grain, so it has no profile and the recipe is Sited.
[region.north.yield.grow_grain]
seasons = { spring = 10000 }
[region.south.yield.grow_grain]
seasons = { spring = 12000, autumn = 7000 }
# A reverse-hemisphere region: the same four seasons, values shifted.
[region.far_south.yield.grow_grain]
seasons = { autumn = 10000 }

# Iron is a sited source whose richness differs by region (the three-region check).
[region.north.yield.mine_iron]
all = 10000
[region.south.yield.mine_iron]
all = 5000
[region.far_south.yield.mine_iron]
all = 15000
```

Viability, from the prototype of `spec-viability.md` 15 (one region, 1 000
households), re-run in round 5 with V1: V1 passes with `gather_seed` (and fails without
it, see above), V2 passes (the seed cycle's surplus `s*` is 1.92 with the source, 0.92
without), `lambda*` 2.981 (2.980 before the source, which adds a near-idle activity), the
households work 33.5% of their ticks to subsist. With three regions (autarky) `lambda*` is 2.980, 2.517 and 3.124, and the
pooled value 3.124. The unrescaled draft gave 0.0137.

Reading it: `grow_grain` is `Sited`, so the arid region, with no profile, cannot run it
(`no_site`). In `north` a job started in spring has `E = 10 000`; one started in
summer has no entry, so by 6.2 rule 3 the factor is 0 and nothing starts. The job
lasts 120 ticks and completes in summer or autumn whatever the season then is (6.4).
Only about a quarter of a year's starts are possible, so a household that farms can
start one job a year (`spec-viability.md` 3.1, the start capacity). `bake_loaf` has no
profile anywhere and is not `Sited`, so it runs at 10 000 in every region. The harvest
window of the grain is the `ShelfLife` of the `grain` commodity, not a condition on
the recipe. The trainee kneads at half output with no oven; a skilled baker then bakes
(`spec-skill-barriers.md` 4.2).

### 12.2 Example B: a space colony (an electronics setting)

The resources `ice_field` and `vat_volume` that its draws name are defined, with their endowments, in `spec-resources.md` 16.2.

Calendar: a planet with two seasons `lit` and `dark` (`spec-calendar.md` 7.4 style).
The viability prototype used the Earth year of 365 ticks with `lit` for spring and
summer, so the figures below hold for that split only. Upkeep per household per tick:
2 `biomass` and 1 `water`; 1 000 households in `crater_rim` (600 and 400 in the
two-region case). Drills have a durability of 500, fabricators of 1 000. Excerpt:

```toml
# Ice from a deposit: a source with no inputs, limited by the deposit (module RES).
# The viability solver treats a draw as unlimited; the size of the deposit is RES's.
[recipe.cut_ice]
sited = true
outputs  = [ { commodity = "ice", quantity = 30, cost_share = 10000 } ]
requires = [ { commodity = "drill", wear_per_use = 2 } ]
draws    = [ { resource = "ice_field", quantity = 30 } ]

# Invented to close the chain.
# Invented (round 5): the source that opens the tool cycle. A fabricator is needed to make
# boards and boards are needed to make fabricators, so without a way in the loop cannot start;
# `salvage` recovers one from the landing kit in 50 ticks. Its labour cost is far above
# `make_fabricator`, so it is inactive once the colony runs.
[recipe.salvage]
duration = 50
outputs = [ { commodity = "fabricator", quantity = 1, cost_share = 10000 } ]
[recipe.melt]
inputs  = [ { commodity = "ice", quantity = 1 } ]
outputs = [ { commodity = "water", quantity = 6, cost_share = 10000 } ]
[recipe.dig_regolith]
outputs = [ { commodity = "regolith", quantity = 30, cost_share = 10000 } ]
[recipe.refine_wafer]
inputs  = [ { commodity = "regolith", quantity = 3 } ]
outputs = [ { commodity = "wafer", quantity = 1, cost_share = 10000 } ]
[recipe.refine_solder]
inputs  = [ { commodity = "regolith", quantity = 2 } ]
outputs = [ { commodity = "solder", quantity = 1, cost_share = 10000 } ]
[recipe.make_drill]
inputs  = [ { commodity = "board", quantity = 1 } ]
outputs = [ { commodity = "drill", quantity = 1, cost_share = 10000 } ]
[recipe.make_fabricator]
inputs  = [ { commodity = "board", quantity = 3 }, { commodity = "solder", quantity = 1 } ]
outputs = [ { commodity = "fabricator", quantity = 1, cost_share = 10000 } ]

# A board: three ticks of fabrication. Slag is waste: discarded, never held (7.2).
[recipe.fab_board]
duration = 3
inputs   = [ { commodity = "wafer", quantity = 2 }, { commodity = "solder", quantity = 1 } ]
outputs  = [ { commodity = "board", quantity = 1, cost_share = 10000 },
             { commodity = "slag", quantity = 1, cost_share = 0, discard = true } ]
requires = [ { commodity = "fabricator", wear_per_use = 1 } ]

# Algae in light: twelve ticks, grown from water; the yield follows the light.
[recipe.grow_algae]
duration = 12
sited = true
inputs  = [ { commodity = "water", quantity = 4 } ]
outputs = [ { commodity = "biomass", quantity = 120, cost_share = 10000 } ]

[region.crater_rim.yield.grow_algae]
seasons = { lit = 10000, dark = 2500 }

[region.deep_crater.yield.grow_algae]
all = 0          # permanent shadow: the recipe cannot run here

[region.crater_rim.yield.cut_ice]
all = 6000
[region.deep_crater.yield.cut_ice]
all = 10000
```

Viability (same prototype, re-run in round 5 with V1, which passes with `salvage` and
fails without it): `crater_rim` alone `lambda*` 1.966, labour share 50.9%;
with two regions and free trade 1.911, where `deep_crater` alone cannot grow biomass
(`lambda*` 0) and depends on trade. The unrescaled draft gave 0.082.

Reading it: the same fields as the bread chain with no field that is a farming word.
`fab_board` is a three-tick job that wears one fabricator item and produces a board
and, at the same step, one unit of slag that goes straight to the `disposal`
boundary, appearing in the flow totals but not in anyone's inventory. `cut_ice` is a
source: no inputs, a drill, and a `Draws` entry that makes the deposit run out and
needs the resources module. `grow_algae` has a profile that varies by season in one
region and is 0 everywhere in the other: no second recipe for the crater. The pack
needs no Go code (architecture.md 9.4, FR-PRO-046).

### 12.3 What the two examples show

| Feature | Example A | Example B |
|---------|-----------|-----------|
| Several inputs | `bake_loaf` | `fab_board` |
| Several outputs and a byproduct | `mill_flour` (bran), `grow_grain` (straw) | none |
| Waste that is never held | none | `fab_board` (slag, `discard`) |
| Multi-tick duration | 120, 2 | 3, 12 |
| Required tool and wear | hoe, millstone, oven | drill, fabricator |
| Source (no inputs) | `cut_wood`, `mine_iron` | `cut_ice`, `dig_regolith` |
| Endowment limit | none | `cut_ice` draws from `ice_field` |
| Climate by region and season | spring / autumn profiles, iron richness | lit / dark and permanent shadow |
| A recipe that runs anywhere | `bake_loaf` | `fab_board` |
| Trainee-eligible | `knead_dough` | none |
| Seed cycle | `grow_grain` and `save_seed`, opened by the source `gather_seed` | none |
| Tool cycle | none | `fab_board` and `make_fabricator`, opened by the source `salvage` |

## 13. Prior art

What was opened, and what was taken. "Opened" means the page was fetched; the
fetch tool returned a summary of it, not the page text, so none of these counts as
read in full.

| Source | Status | Taken | Not taken |
|--------|--------|-------|-----------|
| bazaarBot (`bazaarbot-reference.md` 10, 11) | the reference file read in full; the Haxe source itself was read when that file was written | Role logic as recipes; the need to make tool and no-tool variants explicit | Random tool breakage; fines for an idle role (defect 13); instant production only |
| Factorio recipe prototype (lua-api.factorio.com, `RecipePrototype`) | opened, summary only | Lists of ingredients and results with quantities; a crafting time (`energy_required`); a main-product notion (our non-discard output with cost share); an `ignored_by_productivity` idea for outputs a factor does not scale | `probability` and `amount_min`/`amount_max` (random yield, deferred, 6.7); crafting categories (machines are capital, later) |
| Victoria 3 production methods (paradoxwikis) | opened, summary only | A building produces and consumes goods per level, scaled by workforce (our `k` of at most `w`); a secondary output at a reduced cost (our byproduct) | Method groups that swap with technology (`spec-technology.md`); throughput modifiers |
| abcEconomics `produce` (abce.readthedocs.io, Firm page) | opened, summary only | A production function with fixed input quantities and an immediate transform that raises an error if inputs are short (our all-or-nothing transform, but we start fewer executions instead) | Cobb-Douglas and CES functions (our recipes are fixed coefficients, Leontief); no delay in abcE, so no model for jobs |
| ASSUME (custom unit example) | search snippet only (the unit model with efficiencies and start-up times was not opened) | Nothing; a claim that a unit has operating constraints is not used | Everything |

No default in this spec is set from a source. The bounds and defaults are
structural or placeholders (section 11).

## 14. Observability

14.1. **Events.** The table in observability-and-logging.md 4.3 has `RecipeExecuted`
(a job completes, and a one-tick job completes in its start tick) and
`RecipeAbandoned` (a job ends without output), with their main fields in
parentheses. This spec owns the full field lists and fixes them:

| Event | Emitted when | Fields |
|-------|--------------|--------|
| `RecipeExecuted` | a job completes (and a duration-1 job completes in its start tick) | `Tick`, `Agent`, `Region`, `Recipe`, `StartTick`, `Executions`, the outputs created as a slice of (commodity, quantity) in output order, `Disposed` as a slice of (commodity, quantity) (the part above `MaxQuantity` and, with the storage module on, the part that a cut of the room left no room for, 5.3.2), `EffectiveYield` in basis points |
| `RecipeAbandoned` | a job ends without output (5.4) | `Tick`, `Agent`, `Region`, `Recipe`, `Executions`, `Reason` (`agent_removed`, `role_changed`, `principal_left`, `region_changed`) |

They carry no map, no pointer, and only identifiers and fixed-point values, and
they are emitted after the transition (observability-and-logging.md 4.1, 4.4). No
event is emitted when a start is refused: the volume would be per agent per tick. The
metric `economysim.recipe.executions` is fed by `Executions` of `RecipeExecuted`,
which is why that field is fixed. A new metric for work in progress is a gauge of the
pending quantity by commodity, fed by the snapshot, not by an event.

14.2. **Logs.** At `Debug`, guarded by `Enabled` in the per-agent loop: `recipe not
started` with `agent`, `recipe` and `reason` from the set of 4.2, which includes
`not_on_role` (a request off the role's list, `spec-agents.md` 6.5.1; N-049), `not_permitted`,
`no_facility`, `no_title`, `rent_funds` and `no_room` and the other reasons of the master
order of 4.2. No `Info`. A `Warn`
for the overflow rule of 5.3.2 (`recipe output disposed`, with `recipe`,
`commodity`, `quantity`; also for the part that a cut of the room left no room for, 5.3.2), at most one per recipe per tick, a bound that comes from
the configuration (observability-and-logging.md 3.2.2). The reason constants are exported
constants of the root package (`event.go`); `internal/logkeys` holds the key strings only.
The keys are existing ones (`recipe`, `agent`, `commodity`,
`quantity`, `reason`).

14.3. **Snapshot.** Reports, per region and commodity, the quantity of **pending
output** of jobs in progress, and the number of agents with a job, per recipe. Both
are aggregates bounded by the configuration (observability-and-logging.md 5.4.1).
Pending output is not in any account, so a balance sheet cannot see it; the snapshot
makes it measurable (work in progress for `spec-indicators.md`).

14.4. **Cost with no observer.** Event construction in the production phase is
guarded by the world's "has observer" flag and allocates nothing otherwise
(observability-and-logging.md 4.5; NFR-PRO-001).

## 15. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). Production's, all plain data,
sorted, with no maps:

- for each agent, in agent ID order: its active job, if any (`Recipe`, `Remaining`,
  `Executions`, `Principal`, `Pending` quantities in output order), and its residues,
  one per (recipe on its role's list or its effective own-use list, output item) in
  canonical order (an own-use recipe is not on `Role.Recipes`, 3.3, and produces through the
  same factor and residue);
- the yield shocks, sorted by region and recipe.

**Not saved**, because derived: the yield table (configuration), the cost anchors of
9.3 (recomputed each tick), the season (a function of the tick), the rotation
offset of 5.5.1 (a function of the tick), and the count of pending quantities of a job,
which is the number of its recipe's outputs. The layout and order are those of
`spec-checkpoint.md` 4.2 tag 6. The recipe list and its canonical order
come from the configuration, whose hash a restore checks. A restored world completes
its jobs on the same ticks and produces the same quantities as an uninterrupted
one (FR-PRO-043).

**Ledger additions.** Production asks the ledger for three things, listed in the
handoff because LED is not mine to edit: the `disposal` row in the reason registry;
`Transform` with an empty input list or an empty output list (either, not both);
and `WearItems` (4.4.2).

## 16. Entries for the stability register

Merged into `spec-stability.md` section 4: the delay and the cobweb (STB-L21), the
lockstep of jobs (STB-L22), a job as a gap in income (STB-L19), an uninterruptible
job (STB-L20), a single tool (STB-L27), self-feeding recipes (STB-L28) and contested
endowments (STB-L29); the cost anchor against the equilibrium price is STB-L52. The
entries below are kept as the source text.

Original entries (a sign, a delay, a damper and a test are given in the register):

- **Production delay and the cobweb.** A duration is a delay between the price that
  prompts a start and the supply that follows. For a staple with a long duration
  and a short-lived price signal, supply response can overshoot (the cobweb of
  `evidence.md`, unverified). Dampers: beliefs adjust asynchronously and in bounded
  steps (`spec-beliefs.md`), entry is limited (`spec-spawner.md`), and the
  inventory ratio drives adjustment.
- **Lockstep of jobs.** Agents that are free at tick 0 all start together, and a
  crop that all plant in spring all complete together. A seasonal glut is realistic;
  a glut of a year-round recipe is an artefact. The random starting residue
  (6.6.2) decorrelates yield fractions, not start times. Damper: the warm start
  places initial jobs with staggered `Remaining` by a fixed rule
  (`spec-viability.md` 8.5, which answers the former open question 2). A world may
  start a job with an initial pending output at construction; production allows it
  because a job's state is plain data.
- **A job is a gap in income.** Inputs are paid at the start and output is sold at
  the end, so over a long job the agent shows costs and no revenue. The welfare ratio
  uses a window `T` (`spec-welfare-and-wages.md` 2.3.1, default 30) and reads a
  negative net income, so a role with a job longer than `T` appears to starve while
  it is working. The ratio would trigger migration, relief and bankruptcy for a
  healthy farm. This was N-032; it is settled by `spec-welfare-and-wages.md` 2.3.3,
  which derives the window to exceed the longest job (the warning is removed).
  Credit (`spec-credit.md`, an optional module) is the real answer for a long job; it is in force only where a pack turns it on.
- **A job cannot be interrupted** (5.2.2): the target-income rule overshoots its
  target by up to `Duration - 1` ticks and a cohort makes it `w` households at once.
- **Contested endowments.** Several agents on one deposit: the rotation of 5.5.1
  gave fairness over time, not within a tick. Replaced for draws by the ration of
  `spec-resources.md` 5.3 (RES-A4, merged into STB-L29). A deposit that runs out takes the
  whole role with it. The depletion dynamics are RES's (`spec-resources.md` 4, 6).
- **A single tool.** A recipe that requires a tool stops when the tool breaks. The
  toolmakers' capacity is the damper (`spec-skill-barriers.md` 2.6).
- **Self-feeding recipes.** Seed grain that makes grain is a positive loop that a
  poor yield breaks (a famine of seed). The yield shock of 6.8 is how STY tests it.
- **Haul recipes.** A hired mode's haul unit is made by an ordinary recipe whose `Duration`
  is the round trip (`spec-transport.md` 8.3); its capacity cobweb is STB-L21 with the owner
  `spec-transport.md` 8.4 (TRN-A3).

## 17. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-PRO-001 | A `Recipe` MUST be a plain struct in the root package, with no function field and no embedded type, and no code in production MUST branch on a recipe, commodity, role or class identifier. |
| FR-PRO-002 | The `Inputs` and `Outputs` of a recipe MUST each be a list of at most `MaxRecipeItems` items, each quantity from 1 to `MaxQuantity`, with no commodity twice in one list, in canonical commodity order. |
| FR-PRO-003 | A recipe with no inputs MUST be valid and MUST act as a source, and a recipe with no non-discard output MUST be rejected with `ErrRecipeNoOutput`, unless it has `Builds` or `Repairs` (`spec-infrastructure.md` 4.1.2 B3). |
| FR-PRO-004 | The `CostShare`s of a recipe's non-discard outputs MUST add to exactly 10 000, a sole non-discard output MUST default to 10 000, and a discard output's share MUST be 0, or the configuration MUST be rejected with `ErrCostShares`; a recipe with `Builds` or `Repairs` and no non-discard output has no share to add up and MUST NOT be rejected for it. |
| FR-PRO-005 | `Duration` MUST be an integer from 1 to `MaxRecipeTicks`, default 1, and changing the calendar MUST NOT rescale it. |
| FR-PRO-006 | A `Requires` entry MUST name a commodity with `Durability` above 0 (`ErrToolNotDurable`) and a `WearPerUse` from 1 to its `Durability`, with at most `MaxRequires` entries and no commodity twice. |
| FR-PRO-007 | A recipe with `Draws` MUST be rejected with `ErrModuleDependency` while the resources module is off, and each `Draws` entry MUST be at most `MaxDraws`. |
| FR-PRO-008 | An agent MUST start only a recipe on its role's list or on the role's effective own-use list, or, as a trainee, a trainee-eligible recipe on its trainer's role list, or, as a hired household, a recipe on its employer's hire list (checked at admission, `spec-employment.md` 5.2), and a request for any other recipe MUST start nothing and be refused with the reason `not_on_role`; a request that a right in force bars MUST start nothing and be refused with `not_permitted` immediately after (`spec-institutions.md` FR-ORG-020). |
| FR-PRO-009 | The number of executions started MUST be the least of the weight (or the strategy's maximum), the available stock over each input quantity, the unreserved tool items of each required tool, and (with the resources module on) the executions granted by the ration of `spec-resources.md` 5.3 for each draw, each rounded down, and, for a recipe that needs a facility, builds or repairs, with the capacity and the limits of `spec-infrastructure.md` 4.2.2, and (with the storage module on) `k_room`, and when it is 0 nothing MUST be consumed and the reason MUST be the first applicable of `zero_yield`, `no_site`, `no_tool`, `no_facility`, `no_input`, `no_endowment`, `no_title`, `rent_funds`, `not_unlocked`, `facility_cap`, `no_target`, `store_refused`, `nothing_to_repair`, `no_room` (the order of 4.2). |
| FR-PRO-010 | The start of a job MUST be one all-or-nothing ledger `Transform` of `k` times each input to the `recipe_input` boundary, and a refused transform MUST leave the ledger unchanged and the job unstarted. |
| FR-PRO-011 | At the start of a job each of the `k` executions MUST wear a distinct item of each required tool by `WearPerUse`, most worn first, destroying an item whose durability reaches 0 with the reason `wear`, and wear MUST NOT be applied again for the rest of the job. |
| FR-PRO-012 | At completion the pending outputs MUST be created in one ledger `Transform` from the `recipe_output` boundary with `Born` equal to the completing tick for a commodity that spoils and full durability for a durable. |
| FR-PRO-013 | A job of duration `d` started in tick `s` MUST complete in the production phase of tick `s + d - 1`, so a job of duration 1 completes in its start tick, and its outputs MUST be available to the orders placed after that phase. |
| FR-PRO-014 | An agent MUST have at most one active job, and MUST NOT start a job in a tick in which it continued or completed one. |
| FR-PRO-015 | Every tick from a job's start tick to its completing tick inclusive MUST be a working tick of the executor, and a resting decision MUST NOT interrupt a started job. |
| FR-PRO-016 | In each tick the production phase MUST run before orders are placed, and MUST process continuations and completions of all agents before starts. |
| FR-PRO-017 | (Amended by `spec-resources.md` 5.2: the rotation no longer arbitrates a scarce endowment, the ration does.) Agents in a region MUST be processed for starts in ID order beginning at the position equal to the tick modulo the count of agents in the region, with no random number. |
| FR-PRO-018 | A job MUST be abandoned, creating no output and moving no goods, when its executor is removed, changes role, or when the principal of a trainee's job leaves, and a `RecipeAbandoned` event MUST name the reason. |
| FR-PRO-019 | When the executor's weight falls to `w'`, the job's `Executions` MUST become `min(k, w')` and each pending quantity MUST be multiplied by the new over the old `Executions` rounded down, and a rise in weight MUST NOT change the job. |
| FR-PRO-020 | The site factor MUST be, for a region, recipe and the season of the start tick, the profile's season entry if present, else its `All`, else 0 if the profile has season entries, else 10 000 for a recipe that is not `Sited` and 0 for one that is. |
| FR-PRO-021 | The season and the factor MUST be read when the job starts and the pending outputs fixed then, and a job started with a positive factor MUST complete whatever the season is at completion. |
| FR-PRO-022 | The effective factor MUST be `site * shock / 10 000` then `* tech / 10 000` (`spec-technology.md` 5.1.1; 10 000 for a recipe in no field) then `* trainee / 10 000` (for a hired job `HireYieldBP` replaces it, `spec-employment.md` 6.1) then `* abundance / 10 000` (`spec-resources.md` 4.7; 10 000 for a recipe with no draw), each truncated, and a job with an effective factor of 0 MUST NOT start. |
| FR-PRO-023 | The yield shock of a region and recipe MUST default to 10 000, MUST multiply the site factor, MUST be from 0 to `MaxYieldBP`, and MUST be saved in the checkpoint. |
| FR-PRO-024 | The quantity of an output created by a job of `k` executions MUST be `(k * q * E + residue) / 10 000` and the residue MUST become the remainder of that division, computed with checked 128-bit arithmetic. |
| FR-PRO-025 | Over any number of jobs at a constant effective factor `E`, the total output MUST differ from `k * q * E / 10 000` per job by less than one unit, at every weight including 1. |
| FR-PRO-026 | When an agent is created the world MUST draw exactly one integer in [0, 10 000) from its random source (one source word, `spec-world.md` ruling C8), after the warm start and after the agent's target-ratio draw, whatever its recipes, and set every residue of the agent to it, and production MUST use no other random number; the pending output of a job created at construction from a warm start MUST use residue 0 (ruling C4). |
| FR-PRO-027 | A job's start MUST take `k` times each draw quantity from the region's endowment through the all-or-nothing take of `spec-resources.md` 5.1 (`TakeEndowment`, internal), and production MUST NOT create or restore an endowment. |
| FR-PRO-028 | A recipe with a yield profile MUST need no second recipe for another region or season: a recipe MUST run in a new region, or stop running in one, by a change of the yield table alone. |
| FR-PRO-029 | If creating an output would push the principal's quantity of a commodity above `MaxQuantity`, the part that fits MUST be created and the rest created and disposed with the reason `disposal`, the amount MUST be reported in `RecipeExecuted`, and the job MUST complete; with the storage module on the same MUST hold for the part of an output that a cut of the room left no room for (5.3.2). |
| FR-PRO-030 | A discard output MUST be created and moved to the `disposal` boundary in the same step, MUST NOT appear in any account afterwards, and MUST appear in the flow totals under `recipe_output` and `disposal`. |
| FR-PRO-031 | The ledger reason registry MUST contain `disposal` as a goods sink from an agent to a boundary, owned by production. |
| FR-PRO-032 | Production MUST NOT consult any `Tradable` attribute, and a byproduct (an output with a `CostShare` of 0 that is not discarded) MUST be handled by the same code as any other output. |
| FR-PRO-033 | A trainee-eligible recipe MUST have no `Requires`, `Needs`, `Builds` or `Repairs` (`ErrEligibleHasTool`), and a trainee's job MUST take its inputs from, and create its outputs in, the principal's account. |
| FR-PRO-034 | `TraineeYield` MUST be an integer from 1 to 10 000, default 5 000, applied as the trainee factor of the effective factor, and a trainee's inputs MUST NOT be reduced; for a hired job the factor MUST be the employer's `HireYieldBP` in its place (`spec-employment.md` 6.1). |
| FR-PRO-035 | `UnitCost` MUST equal the sum of quantity times price over the inputs, plus the sum of `ceil(WearPerUse * tool price / Durability)` over the tools, plus `Duration` times the wage, plus (with the tenure module on) the sum over the draws of quantity times the posted rent (`spec-land.md` 5.7.1), times `CostShare`, divided by the output quantity times the effective factor, rounded up, in checked integer arithmetic. |
| FR-PRO-036 | `UnitCost` MUST be 0 when any input or tool price is 0, the wage is 0 or less, the effective factor is 0, the cost share is 0, or the arithmetic would overflow, and MUST have no side effect and allocate nothing. |
| FR-PRO-037 | The cost anchor of a commodity in a region MUST be the smallest non-zero `UnitCost` over the recipes that make it and can run there this tick, or 0 if none, computed once per tick before orders from the previous clearing's published prices and the regional wage of FR-PRO-047. |
| FR-PRO-038 | A yield profile that names an unknown region, recipe or season MUST be rejected with the sentinel for that kind or `ErrUnknownSeason`, and a duplicate profile with `ErrDuplicateID`, and a value outside [0, `MaxYieldBP`] with `ErrOutOfBounds`. |
| FR-PRO-039 | A configuration in which the largest weight times an input, a draw, or an output times the largest possible effective factor exceeds `MaxQuantity`, or in which the cost of the inputs can exceed `MaxMoney`, MUST be rejected with `ErrOverflow`; the largest possible effective factor includes `MaxTechBP` for a recipe in a technology field and the larger of the trainee factor and `HireYieldBP` for a recipe on a hire list (10.3). |
| FR-PRO-040 | Validation MUST warn, and MUST NOT fail, for an unused recipe, an unrunnable recipe and a durable no recipe requires. |
| FR-PRO-041 | `RecipeExecuted` MUST be emitted at completion with the fields of section 14.1, and `RecipeAbandoned` when a job is abandoned, each after the state change and with no map. |
| FR-PRO-042 | The snapshot MUST report the pending output by region and commodity and the number of agents with a job by recipe. |
| FR-PRO-043 | The state of section 15 MUST be saved as plain data in a fixed order with no maps, and a world restored mid-job MUST complete its jobs on the same ticks with the same quantities as an uninterrupted run. |
| FR-PRO-044 | A recipe with a yield below 10 000 for a job MUST NOT create or destroy goods outside the `recipe_input`, `recipe_output`, `disposal` and `wear` flows. |
| FR-PRO-045 | Goods MUST be conserved except by those flows after every tick of a run that contains multi-tick jobs and abandoned jobs, checked by audit. |
| FR-PRO-046 | The two example packs of section 12, completed with the elided commodities and with their sources (`gather_seed`, `salvage`), MUST load, validate, pass the viability checks V1 to V4 of `spec-viability.md` (V1 included: a cycle with no source outside it is rejected, `unproducible_in_realm`; `lambda*` of at least 1 with every household working every tick) and run for a stated number of ticks with the invariants of testing.md 6.3 holding, with no Go code specific to either; that the run reaches a stable state is the claim of `spec-stylised-facts.md`, not of this requirement. |
| FR-PRO-047 | The regional wage used for `UnitCost` MUST be the pooled observed wage as `spec-welfare-and-wages.md` 4.3 defines it (the definition and the cases in which it is unknown are that spec's: FR-WEL-018, FR-WEL-020), as it stood at the end of the previous tick, and MUST be 0 when that wage is unknown. |
| FR-PRO-048 | A request for a job MUST be checked against the role's recipe list and effective own-use list before any other check (`not_on_role`), then against the rights in force (`not_permitted`) and before `k0` is derived, for the principal of a trainee's or a hire's job as for the executor as the right says, and the cost anchors of 9.3 MUST be computed once per tick before `Work` is called. |
| FR-PRO-050 | A job in progress MUST be abandoned with the reason `region_changed` when the executor's account is to change region by a migration journey, at its departure, and `RecipeAbandoned` MUST carry that reason. (new, `spec-migration.md` 8.2) |
| FR-PRO-051 | A recipe that is only on an own-use list MUST NOT set a cost anchor, and MUST count as a use for `unused_recipe`. (new, `spec-subsistence.md` 3.6) |
| FR-PRO-049 | In a region whose wage is 0, the cost anchor of every commodity MUST be 0, so that only the reference band applies to beliefs there. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-PRO-001 | With the default logger and observer, the production phase of a step of the reference chain MUST allocate nothing beyond scratch held by the world. |
| NFR-PRO-002 | The cost of the production phase in a tick SHOULD be proportional to the number of agents, not to the number of households or the weights. |
| NFR-PRO-003 | The yield lookup, `UnitCost` and the execution count SHOULD have benchmarks that call `b.ReportAllocs()`. |
| NFR-PRO-004 | The yield-profile validator and the lookup MUST have fuzz tests (testing.md 7.4) asserting they never panic and that the factor is within [0, `MaxYieldBP`] for every valid input. |

## 18. Open questions

Answered in round 3 (kept so that a reader of the first draft can find them):

1. **Same-tick completion and the order of markets.** Answered: the phase order of
   `spec-agents.md` 4 (production, then `Trade`, then clearing) is the order this spec
   assumes, and N-009 hands it to `spec-world.md`. If the world changes it, 5.2
   changes with it.
2. **Staggered initial jobs.** Answered by `spec-viability.md` 8.5: a fixed
   stagger rule, no random number, and construction books the inputs of a job in
   progress through `initial` and `recipe_input` (N-056).
3. **Income attribution over a long job.** Answered: the welfare window is derived to
   exceed the longest job (`spec-welfare-and-wages.md` 2.3.3); attribution was
   rejected there.
4. **`WearItems`.** Decided (N-033, accepted by the coordinating agent): the ledger
   gains `WearItems`, `Transform` with an empty side and the `disposal` reason. Wear by
   `k` calls of `Wear` is wrong for a cohort and is not used. Waiting for
   `spec-ledger.md` to be edited.
5. **Who picks the recipe.** Answered by `spec-agents.md` 7.1.4 (greatest net value per
   tick of labour at the agent's belief centres, ties to the lower ID).
6. **Per-output yield factors and a harvest-window condition** (6.7). Decided: not in
   version 1. No pack in section 12 needs one, and a perishable intermediate does the
   job. Add only if a pack needs one.
8. **A yield above 1 for a trainee.** Decided: `TraineeYield` stays capped at 10 000;
   no case needs more.
9. **Rotation within a region.** Decided (labelled choice): the count of agents is taken
   at the start of the production phase, as N-036 asks of `spec-world.md`.

Still open:

7. **A tool shared by several households.** A cohort of weight `w` needs `w` tool
   items. Whether a pack wants a "workshop" tool used by many is
   `spec-infrastructure.md`'s. **Answered (2026-10-12):** a workshop is a facility class
   that a recipe `Needs` (3.1); its capacity bounds the executions of the principal's starts
   in a tick, and a facility is the owner's alone, with no sharing and no queue
   (`spec-infrastructure.md` 5.1.2, 5.1.5). A tool that stands for a workshop stays legal.
10. **A wage per role for the cost anchor.** 9.3 costs every recipe at the region's
    common wage. A skilled role earns more, so its recipes are under-costed. Next step:
    run the reference chain (`spec-reference-chain.md`) with the pooled wage and compare
    the anchor with the zero-profit price per recipe; if a skilled recipe's price over
    its anchor is far from 1, cost it at the wage of the roles that list it.
11. **The starting wage.** 9.3 asks `spec-viability.md` to supply a regional starting
    wage so that the cost band exists from tick 0. Until it does, a run without a warm
    start has no cost band in its first ticks (9.1.1). Next step: VIA adds the field.

## 19. Where this lives in the code

| Concern | Home |
|---------|------|
| `Recipe`, `Item`, `RecipeOutput`, `Requirement`, `Draw`, `YieldProfile`, `SeasonYield`, `DefaultRecipe()` | root package, `recipe.go`; defaults in `defaults.go` (`spec-config.md` 5.1) |
| The bounds of 10.1 | root package, `bounds.go` |
| Validation, the sentinels and the warnings of section 10 | root package, `validate.go`, `errors.go` |
| The execution-count rule (4.2), the yield lookup (6.2), the residue arithmetic (6.6), `UnitCost` and `TrainingPays` | `internal/production`: pure functions over primitive slices and indexes (architecture.md 2.9) |
| The job, the residues and the production phase | root package, `agent.go` and `world.go`, calling `internal/production` and `internal/ledger` inline |
| The reason `disposal` | `internal/ledger` registry (`spec-ledger.md` 7.2) |
| The events `RecipeExecuted` (extended) and `RecipeAbandoned` | root package, `event.go` |
| Reason constants for refused starts | exported constants of the root package, `event.go`; `internal/logkeys` holds the key strings only |
