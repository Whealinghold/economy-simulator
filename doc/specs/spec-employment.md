# Spec: Employment: hired labour, the labour round and the wage contract

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-EMP-NNN` and
`NFR-EMP-NNN`, which tests cite (testing.md section 4). They are in section 21.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
later stage (`vision.md` section 4, "Later"), off unless configured. A world
without `Modules.Employment` behaves as a world without this spec: every
household works for itself, as `spec-agents.md` says. Needs `spec-agents.md` (AGT:
the household, `Strategy`, `JobRequest`, the work share, the budget), `spec-welfare-and-wages.md`
(WEL: the welfare ratio, the observed wage, the target-income rule),
`spec-production.md` (PRO: the job, the executor and the principal, the yield
factor), `spec-ledger.md` (the Escrow account and the reason registry),
`spec-world.md` (WLD: the position in the tick and the draw table) and
`spec-config.md` (the module switch and validation). It reads `spec-merchants.md`
(MER: the first-fit round and the escrow it reuses), `spec-spawner.md` (SPN),
`spec-estates.md` (EST), `spec-regions.md` (REG), `spec-demand.md` (DEM) and
`spec-money-supply.md` (MON), and asks several of them for an edit (section 12).
Leaves capital to `spec-infrastructure.md` (CAP) and credit to `spec-credit.md` (CRD).

**Every number in this spec is a placeholder** except where a derivation is shown.
Nothing has been run in the library, bazaarBot has not been run, and no acceptance
band exists (notes N-015, N-027). The scripts of section 18 are stylised Docker
Python models of the rules, not claims about the library; the rules themselves are
integer. No default is claimed to be stable. No source on wage bargaining or on the
size of the labour share was read or used (N-018): the rule of the wage is a
convention of this library (the midpoint of the two sides' values, as the book and
the contract round price), with one lever that a pack states. The settings of the
worked examples (section 19), a river valley and a mining colony, are **examples**,
flagged as such; the core and its vocabulary are setting-agnostic, and a commodity,
a recipe, a role or a mode is data (architecture.md 9.1): nothing in this spec
branches on one.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game or server), never a household (N-001, N-017).

## 1. Purpose and scope

Until now every household in the library works for itself. A household of a role with
recipes starts a job on its own account, with its own inputs, and the output is its own;
a household of a role with none earns only what another spec pays it (a grant, relief,
`spec-agents.md` 3.3.1). The roadmap names two macro items that need more: *the wage
and demand loop* (DEM, EMP) and *unemployment* (EMP); `spec-welfare-and-wages.md` 10
lists "wages for employees working in someone else's recipe" as a non-goal until this
spec. This spec adds **hired labour**: one household works, for a wage, on the account
of another. It covers:

- what a **hire** is: a one-job contract between a **worker** and an **employer**, in
  which the worker is the executor of a job and the employer its principal (the
  mechanism of the trainee of `spec-production.md` 8, generalised), the wage is money
  per household-tick, and labour is still not a commodity (section 3);
- what a strategy returns to take part: a **labour offer** and **vacancies**, carried
  by `JobRequest`, and the default's rule for both (section 4);
- the **labour round**: admission, matching, the wage, formation and the escrow, once a
  tick between the two passes of the production phase (section 5);
- the **hired job**: its start, the wage paid each working tick, completion and every
  interruption, including default and the estate (section 6);
- the ledger form: the Escrow account, three reasons, no faucet and no sink (section 7);
- how the **target-income rule** treats a wage earner and an employer, and what the
  welfare record, the observed wage and the work floor do with a hired tick
  (section 8);
- the employer's budget and targets (section 9), **unemployment as a measure** and the
  wage bill (section 10), and the **wage and demand circuit**, with the closed form that
  says what limits employment in a world where the employers' profit has no outlet
  (section 11);
- the seams, the parameters, validation, observability, the state saved in a checkpoint,
  the register entries, the models run, two worked examples, the requirements and the
  open questions (sections 12 to 23).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| Tools, training and the skill premium | `spec-skill-barriers.md` (SKL); a hired household is unskilled by construction (3.3) |
| Capital: who owns a facility, and rent | `spec-infrastructure.md` (CAP), `spec-land.md` (LND); here a facility is a tool the employer holds (6.1) |
| Credit: an advance of wages, a loan to an employer, a wage claim | `spec-credit.md` (CRD); the escrow is cash only (7.3) |
| A minimum wage, a wage cap, a closed shop, a body's say over hiring | `spec-institutions.md` (ORG): a closed shop is the field `Who` of a `produce` right (5.3), a wage control is reserved with its form (6.8); `spec-policy.md` (PLC) |
| Standing employment: a service contract of a year, a retainer, a bond | not in version 1 (3.4.4) |
| Pay in kind | not in version 1 (8.6.1, open question 1) |
| What a household does for itself when no job is offered | `spec-agents.md` 7.1, `spec-subsistence.md` |
| Whether a household moves for work | `spec-migration.md` (MIG); it reads an observed wage and rumoured prices, never an offer |
| The level of prices and of money | the market (`spec-clearing.md`, `spec-beliefs.md`) and `spec-money-supply.md` |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | Hired labour is a module, off unless configured (`Modules.Employment`). A role is **hireable** (`Role.Hireable`) or **an employer** (`Role.Employer`, a block), never both. An employer is an agent of weight 1; a worker may be a cohort | 3.1, 3.6 |
| 2 | A hire is the **trainee mechanism generalised**: the worker is the *executor* of a job whose *principal* is the employer. Inputs and tools come from the principal's account, the output goes to it, the yield is the site's times a factor `HireYieldBP`. Production is unchanged except for that list of places where "the trainer" reads "the principal" | 3.4, 6.1, 12 |
| 3 | **Labour is not a commodity** and no recipe has labour as an input. A wage is money moved between accounts under a ledger reason; the cost of labour in a recipe stays `Duration` working ticks at the observed wage (`spec-production.md` 9.1). `spec-goods.md` 7.3 stands | 3.5 |
| 4 | A hire is **one job, a spot contract**. There is no standing employment, no notice and no firing: an employer "lets go" by not posting the vacancy again. The unit of the round is a household-tick of one job | 3.4 |
| 5 | A strategy takes part through `JobRequest`, which gains a **labour offer** and a **vacancy** as value fields. `Strategy` keeps its three methods. A consumer's strategy and the player's directive can offer and post like the default | 4.1, 4.4 |
| 6 | Matching is a **first-fit round** after every `Work` call and before the ration of scarce endowments (a new position, P2.5a): offers (workers, indivisible blocks of households, ascending reservation) against vacancies (employers, divisible, descending bid), ties by the stateless key `K(t, id)` of WLD 10.1.1 (no draw), in the form of the contract round of MER (ADR 0005) | 5 |
| 7 | The wage is **not configured and does not bid up**: it is the half-even midpoint, with a worker share `WorkerShareBP` of the employer's block (default 5 000, the midpoint), between the worker's reservation (the value of its own best alternative, else 0) and the employer's bid (the net value of one tick of the recipe). A shortage of labour moves it only through the prices that move the bid | 5.4, 4.3 |
| 8 | The wage of the whole job is **escrowed at formation** in an Escrow account (the form of MER 9.5) and released a tick at a time as the job is worked, so that neither side can default on a hire that exists, and a removal settles by a table (6.4). No credit, no claim | 5.5, 6.2, 7 |
| 9 | The **target-income rule** applies to a worker as it does to any household: the households it offers are its work share (`spec-agents.md` 7.1.2), and the wage counts in the ratio. To an employer it applies to its **own** labour only; its hires follow margin and capacity, because a rule that made an employer rest when rich would add a second unit-elastic link to the labour loop (L14) and make the establishment work in bursts of a window | 4.3, 8.2 |
| 10 | The observed wage of a worker is what its wage is. The **pooled regional wage** that anchors the cost of labour (`spec-production.md` 9.3) is taken over the agents of the roles that have no employer block, so that the profit of employers does not inflate the cost anchor. The rule is a function of the configuration and adds no state | 8.4 |
| 11 | An employer's profit has to go somewhere, or the circuit stops: in a closed economy whose employers only eat, the hired households settle at `E * u_e / ((1 - phi) * a)`, a number a model confirms (11.2). The spec says so, warns on a pack with an employer role and no outlet for its profit (`employer_no_outlet`), and leaves the cure to wants (DEM), the saving rule's cap (AGT), a flow of money (MON) or investment (SAV, `spec-saving-and-investment.md`) | 11 |
| 12 | A removal is settled **without a penalty and without a bond**: a worker removed mid-job ends the hire and the escrow returns to the employer; an employer removed, or changed to another role, abandons the hired jobs (`principal_left`) and the escrow goes back to the employer's own account (a removal: before its creditors' waterfall, 6.5.2). The worker keeps every tick it was paid | 6.4 |
| 13 | A principal with open hires is **bound to its region**: this spec registers the `region_bound` refusal `open_hire`, the rule that N-090 gave later modules | 6.5 |
| 14 | Unemployment is **measured, never set**: the households that offered and were not matched, by region and role, in the aggregate event `LabourRound` and in a snapshot row. Voluntary rest under the target-income rule is not unemployment. IND gets three asks | 10 |
| 15 | The module adds **no faucet, no sink and no ledger reason of kind faucet or sink**: three internal reasons (`wage_escrow`, `wage_pay`, `wage_refund`), money between non-boundary accounts only | 7 |
| 16 | The module's loops are registered with provisional IDs: the hired case of the target-income loop, the hire cobweb, the profit circuit, the payroll's drain on liquidity, the wage that follows the margin, and the employer's occupation signal | 17 |

## 3. Parties, data and what a hire is

### 3.1 The module and the two role fields

`Config.Modules.Employment` is a pointer: present means the module is on
(`spec-config.md` 3.2). It has one field of its own (13). A role gains two optional
fields, in the form of `Informed`, `Fixed` and `Merchant`:

| Field | Meaning | Default |
|-------|---------|---------|
| `Role.Hireable` | the households of the role may offer their labour for hire (4.1) | false |
| `Role.Employer` | the employer block (3.2); nil means the role does not hire | nil |

3.1.1. A role is hireable or an employer, not both (`ErrEmploymentShape`): one job at a
time and one side of the round per agent. A pack that wants a household to be a worker
on some days and an employer on others writes two roles and lets the spawner move it
between them (`spec-spawner.md` 6), or uses a hireable role with its own recipes (3.3).

3.1.2. With the module off, `Hireable` and an employer block are not silently ignored:
validation rejects them (`ErrModuleDependency`, 14). A field of a module that is off is a
mistake, as `Informed` is without `Modules.Information`.

### 3.2 The employer block

| Field | Meaning | Default | Bound |
|-------|---------|---------|-------|
| `Recipes` | the **hire list**: the recipes a hired household may run for the role, a subset of `Role.Recipes` | required | at least one, each on `Role.Recipes`, no duplicate |
| `MaxHires` | the most hired executions the agent has in progress at once: the **establishment** | 4 (placeholder) | 1 to `MaxEstablishment` |
| `WorkerShareBP` | the share of the margin between the worker's reservation and the employer's bid that the worker receives, in basis points (5.4) | 5 000 (the midpoint; placeholder) | 1 to 10 000 |
| `HireYieldBP` | the factor on the yield of a hired execution, in basis points: a hired household's output is not the principal's own (6.1) | 10 000 (placeholder) | 1 to `MaxYieldBP` |

3.2.1. The hire list is a subset of the role's own recipes, so an employer can always run
the work itself, and the same recipe has one yield table, one set of inputs and one
cost anchor whoever executes it. A recipe may need a tool (`Requires`) or draw on an
endowment: the tool is the principal's (6.1), and the draw is rationed with every other
request of the region (5.6).

3.2.2. A recipe on the hire list need not be `TraineeEligible`, and a trainee-eligible
recipe need not be on it. The two mechanisms are independent and an employer may be a
trainer (6.3.4).

### 3.3 The worker

A **worker** is a household (an agent of any weight) of a `Hireable` role, free at the
call of `Work`, not in the own-use mode of `spec-subsistence.md`, not a trainee. It may have
recipes or not:

- a role with **no recipe** has no alternative: its reservation is 0 and its
  only source of earned income is the wages of the round (and relief, a grant,
  `spec-agents.md` 3.3.1);
- a role **with recipes** keeps its own job as the fallback of its offer: if the offer is
  matched the own request is withdrawn, if not it stands (4.1). Its reservation is the value
  per tick of its own best executable recipe (4.3.2), so a household works for a wage when
  the wage beats its own recipe.

3.3.1. A hired household is **unskilled by construction**: it runs a recipe of the hire list
without the skill barriers of `spec-skill-barriers.md` (the tool is the employer's, there
is no traineeship), at the factor `HireYieldBP`. A role whose households are skilled
by tool or by training is an employer role or a self-employed one; a skilled worker is a
pack that gives the employer a high `HireYieldBP` and the worker role a higher
reservation (its own recipe). This is a choice of version 1 (open question 9).

3.3.2. A household in the own-use mode offers nothing (`in_own_use`): the mode is the
floor under a failed market and this spec does not change its entry or its exit
(`spec-subsistence.md` 5). Whether a household that cannot feed itself should take a wage
before it takes the floor is open question 8.

### 3.4 What a hire is

3.4.1. A **hire** is the record of one job of one worker for one employer, with an ID
(`HireID`, from a counter that is state, 16). It has: the worker and the employer
(agents), the recipe and the number of executions `k` (the households of the worker that
work, at most its weight), the **wage** per household-tick, the Escrow account that holds
the unpaid wages, the tick it was formed and the tick the job completes. The job itself is
the worker's job of `spec-production.md` 5.1, with `Principal` the employer: the hire is
not a second copy of it.

3.4.2. A hire exists from its formation at P2.5a (5.5) to the tick of its end (6.3, 6.4).
It has no negotiation, no counter-offer and no renewal: the next job of the worker is the
next round.

3.4.3. **One job at a time.** A worker has at most one job and so at most one hire; an
employer has as many hires as `MaxHires` allows (a hire of `k` executions counts `k`). A
worker is hired by one employer at a time: a cohort works its `k` households for one
principal and the rest of its households rest for the length of the job (the partial
support of `spec-production.md` 4.2, `spec-agents.md` 9.4). The offer of a cohort is
indivisible for that reason (5.4).

3.4.4. **No standing employment in version 1.** A worker that works every tick for one
employer does so by being matched again each time, with a new wage computed from the
prices of that tick. The cost is that a worker has no notice and no security and an
employer no retained staff; it is registered (EMP-A2). A service contract with a term, a
bond and a notice is a later extension (open question 3) and would reuse the record of
3.4.1 with a longer job.

### 3.5 Labour is not a commodity, and a wage is not configured

3.5.1. No recipe has labour as an input, no commodity is labour and no order or ask is for
labour. `spec-goods.md` 7.3 and `spec-production.md` 3 stand. What a recipe's labour costs
is `Duration` working ticks at the observed wage of the region (`spec-production.md` 9.1,
9.3); with hired labour that wage is a measurement of wages paid and of own earnings, as it
was before.

3.5.2. **No configuration field holds a wage** or a wage multiplier (`spec-currency.md`
2.6, FR-WEL-015). `WorkerShareBP` is a share of a margin that the employer's own beliefs
and the worker's own alternative produce; it is the one lever, and its effect on what a
household earns depends on the prices of the pack. The wage of a tick is derived from
state (4.3, 5.4) and is not saved.

### 3.6 Weight

3.6.1. **An employer has weight 1.** A cohort employer would need its establishment, its
places and its own labour to scale with the weight, and a partial admission of hires; this
version does not define that (the same limit as the merchant of `spec-merchants.md` 3.6
and the trainer of `spec-skill-barriers.md` 4.1.1). Validation rejects an employer role
with a group of scale above 1 or with more households than agents
(`ErrEmployerWeight`); the first consumer's world needs no weight above 1 (N-030).

3.6.2. **A worker may be a cohort** of weight `w`. It offers up to `w` households as one
block (4.1); a hire takes the whole block or none (5.4). The weight of a hired cohort may
fall during the job (the ladder, the population module): the executions are scaled by the
rule of `spec-production.md` 5.4 and the escrow remainder is refunded at the end (6.4).

## 4. What a strategy returns

### 4.1 The types

`JobRequest` (`spec-agents.md` 6.3) gains two value fields. A zero value means none, so
every existing strategy returns what it returned and is unaffected:

```go
// JobRequest names the recipe to start (AGT 6.3). The fields below are the employment
// module's (spec-employment.md 4.1); zero values mean none.
type JobRequest struct {
	NoJob      bool
	Recipe     RecipeID
	Executions int64

	Offer LabourOffer // a worker's offer of its households for hire; Households 0 means none
	Hire  HireRequest // an employer's vacancies; Positions 0 means none
}

// LabourOffer offers households for a wage. It is indivisible: all of them, or none.
type LabourOffer struct {
	Households int64 // 1 to the agent's weight
	Reserve    Price // the least wage per household-tick it accepts, 0 to MaxPrice
}

// HireRequest posts vacancies of one recipe of the employer's hire list.
type HireRequest struct {
	Recipe    RecipeID
	Positions int64 // executions wanted, at least 1
	Bid       Price // the most it pays per household-tick, 1 to MaxPrice
}
```

4.1.1. **The own request is the fallback of an offer.** `Recipe` and `Executions` are the
job the agent would start for itself (`NoJob` for a role with none). If the offer is matched
the world withdraws that request before the ration of P2.6; if it is not, the request stands
and is started at P2.7 as always. An agent can therefore offer "my own job, or a wage that
beats it".

4.1.2. **An employer's `Hire` is independent of its own request.** The employer may start
its own job (`Recipe`, `Executions`) and post vacancies; the two share its stock and its
tools, and the world allots them in the order of 5.5. A vacancy is for one recipe, which
may differ from the one the employer runs itself.

4.1.3. `Work` is called only for an agent that is free (no job in progress, none continued
or completed this tick, not away, not on a journey, not teaching; `spec-world.md` 6.4,
P2.5). An employer that is busy with its own job posts no vacancy that tick: a limit of
version 1 (open question 4); an employer that lets its own labour rest is free every tick.

### 4.2 What the world checks

The world checks a returned `Offer` and `Hire` as it checks a `JobRequest`
(`spec-agents.md` 6.5.1): it refuses what it cannot take whole, with a reason, logs it at
`Debug` and leaves the rest of the request alone. The reasons are the admission reasons of
5.2. A request the checks refuse starts nothing from the offer or the vacancy and changes
nothing else.

### 4.3 The default strategy

`DefaultStrategy.Work` (`spec-agents.md` 7.1) gains one step, after its own decision of 7.1.2
to 7.1.7 and using its results. It reads the view of 9.1 (the employment group).

4.3.1. **A worker's offer.** If the role is `Hireable`, the agent is not in the own-use mode
and not a trainee, and the work share `s` of 7.1.2 (with the survival rule and the floor of
7.1.3 already applied) is above 0, the default offers

```
Households = ceil(w * s / 10 000)            the same count as 7.1.7 before the caps
Reserve    = the value per tick of its own best alternative (4.3.2), 0 if it has none
```

It does not offer a household that the target-income rule says should rest: the wage counts
in its ratio, so a household whose ratio reaches the top of the band offers nothing, and
its wage stays where it is (8.2).

4.3.2. **The reservation of a household with recipes.** The value per tick of its own best
executable recipe by the rule of 7.1.4, `max(0, floor(net_r / Duration_r))`, over the
recipes that pass 7.1.4 and 7.1.5 with `MaxExecutions >= 1`; 0 if there is none. It is the
agent's own price of a tick of its labour at its own beliefs, so the decision stays a
function of the agent's own state. A household in a role with no recipes has no
alternative and reserves 0.

4.3.3. **An employer's vacancies.** If the role has an employer block, the agent is free
and its hire list has a recipe that is **eligible** (the net value `net_r` of 7.1.4 positive,
or a source; not halted by 7.1.5), the default posts for the best such recipe `r` by
`net_r / Duration_r` (ties to the lower ID), valuing the output at the hired yield
(`E * HireYieldBP / 10 000`):

```
Bid       = floor(net_r / Duration_r)               the most a tick of a hired household is worth
Positions = min( MaxHires - Hired,                  the establishment, less the hired executions open
                 by_stock, by_tools, by_cash )      all at least 0
by_stock  = min over inputs i of floor((Stock_i - Keep_i) / q_i) - k_own      (keep-the-food: 7.1.6)
by_tools  = min over Requires t of (unreserved items of t) - k_own
by_cash   = floor( (Money - UpkeepCost) / (Bid * Duration_r) )
```

`k_own` is the executions of its own request (7.1.7). `UpkeepCost` is the one-tick cost of
7.4.2 at its beliefs: the employer leaves its tier 1 for this tick. If `Bid` is below 1 or
`Positions` is 0, it posts nothing. Because `Bid` is the net value *before* any wage, the
employer's margin on a hire is `Bid - wage`, which is at least `(1 - WorkerShareBP/10 000) *
(Bid - Reserve)` (5.4).

4.3.4. **The default does not satisfice its hires.** The number of positions is not scaled by
the work share: an employer whose ratio is above its target still posts its vacancies, and
only its own labour rests (8.2). The reason is in 8.2.2.

4.3.5. **No memory and no draw.** The default keeps no state and draws no word in `Work`
for an offer or a vacancy; the round breaks its ties by `K(t, id)` (5.3). An agent that offers and posts
nothing changes no result.

### 4.4 A consumer's strategy and the player

4.4.1. A consumer's `Strategy` returns the same two fields from `Work` and is checked the
same way; a role it serves must be `Hireable` or have an employer block for its fields to be
taken. The `Directed` strategy's directive carries a `*JobRequest` (`spec-agents.md` 10.3),
so a **player** can offer labour or post vacancies through the mailbox like any request,
subject to the same admission (a player whose role has no employer block gets
`not_employer`).

4.4.2. The world treats a consumer's offer as it treats the default's. A consumer that wants
a wage rule of its own (a bid that rises with unfilled vacancies, a fixed wage) writes it in
its strategy: the bid and the reservation are its numbers, and the round only enforces that
the wage lies between them (5.4) and that the money is there (5.5). The guarantee is
conservation and determinism, not the stability of its rule (L121).

## 5. The labour round

### 5.1 Where it runs

5.1.1. **Position P2.5a**, a new position of `spec-world.md` 6.4 (`6.1.10` there: a new
position takes a letter after the position it follows): after P2.5, the calls of `Work`
for every free agent, and before P2.6, the ration of scarce endowments. It is in the
middle of the two passes that ruling C9 fixed for production, so all requests exist when
it runs and no start has happened.

| Position | What runs | Reads | Writes | Draws | Events |
|----------|-----------|-------|--------|-------|--------|
| P2.5a | the labour round: admission, order, matching, formation (5.2 to 5.5) | the `Offer` and `Hire` of the requests of P2.5, stock, tools, cash, the snapshot of the endowments of P2.3, the yield table and its shocks | hires, Escrow accounts and balances, the withdrawn own requests of matched workers, the hired requests (scratch, which join P2.6), the committed executions of each employer (scratch) | none: ties are broken by the stateless key `K(t, id)` (5.3), which takes no word and adds no row to the draw table | `HireFormed`, per hire in formation order |
| P2.8b | the aggregate of the tick: offers, vacancies, matches, refusals, wages paid | the counts of P2.1, P2.5a and P2.7 | none | none | `LabourRound`, per region ascending, after `ProductionHalted` of P2.8a |

5.1.2. The round runs only with `Modules.Employment` present, and does something only where an
offer or a vacancy was admitted. The module takes **no word of the world's source at any time**
(5.3.1), so a world with the module on and no offer or vacancy leaves the stream as it would be
with the module off (the rule of `spec-world.md` 6.1.5 and 7.4.2), and so does a world with
offers: the stream is the same whatever the round does.

5.1.3. **The one-tick delay.** A hire formed at P2.5a starts at P2.7 of
the same tick, and its first wage is released there; a worker's wage is therefore in its
account before P3.2 of the tick, a tick earlier than money earned at P4.1 (`spec-world.md`
6.1.8). The delay of the circuit through the market is unchanged: the employer's sales are
at P4.1.

### 5.2 Admission

At the start of the round the world visits the `Offer` and `Hire` of the requests in
ascending agent ID (not in the rotation order of the calls, so that the result does not
depend on it) and admits each whole or refuses it with a `reason`, in this order. The
reasons are exported constants of the root package (`event.go`; `internal/logkeys` holds the
key strings only). A refusal changes nothing, is counted in `LabourRound` (15.1) and logged
at `Debug`; it is not an error returned by `Step`.

| Of | # | `reason` | Condition |
|----|---|----------|-----------|
| both | 1 | `employment_off` | `Modules.Employment` is absent. There is then no round and no `LabourRound`: the refusal is a `Debug` line only (15.2.1); with the module on this reason cannot arise and its count in `LabourRound` is not a field |
| offer | 2 | `not_hireable` | the agent's role is not `Hireable` |
| offer | 3 | `in_traineeship` | the agent is a trainee (it runs its trainer's recipes, `spec-skill-barriers.md` 4.2) |
| offer | 4 | `in_own_use` | the agent is in the own-use mode (3.3.2) |
| offer | 5 | `invalid_households` | `Households` below 1 or above the agent's weight |
| offer | 6 | `invalid_price` | `Reserve` below 0 or above `MaxPrice` |
| vacancy | 7 | `not_employer` | the agent's role has no employer block |
| vacancy | 8 | `not_on_hire_list` | `Recipe` is not on the role's hire list |
| vacancy | 9 | `invalid_positions` | `Positions` below 1 |
| vacancy | 10 | `invalid_price` | `Bid` below 1 or above `MaxPrice` |
| vacancy | 11 | `employer_full` | `MaxHires` less the hired executions the agent has open is 0; a larger `Positions` is **cut** to what remains and admitted |

5.2.1. An agent that returns both an offer and a vacancy (a consumer's, 4.4) has the one its
role allows admitted and the other refused (`not_hireable` or `not_employer`), since 3.1.1 makes
a role one side of the round.

### 5.3 Keys and order

5.3.1. **No draw.** The round takes no word of the world's source. A tie between offers of
equal reservation or between vacancies of equal bid is broken by the stateless key
`K(t, agent ID)` of `spec-world.md` 10.1.1 (smaller first), the rule for a new tie, so that a
lower ID is never systematically first (the reservation of a role with no recipe is 0 and the
tie is the usual case). An agent has at most one offer or one vacancy in a tick (3.1.1), so `K` and then the
ID is a total order. This replaces the drawn key of `spec-merchants.md` 9.4.2 for two reasons:
it follows the rule of WLD for a new tie without a stated reason to leave it, and a draw per
admitted offer would move every later word of the run whenever one household's offer
changed (`spec-world.md` 7.6), which the keyed tie does not. The module adds **no row to the
draw table** of `spec-world.md` 7.3 and no term to the formula of 7.3.1.

5.3.2. **Lanes.** A lane of the round is a region. The offers are those of agents whose
account is in the region, the vacancies those of employers whose account is. A worker and an
employer in different regions never meet: the lane is the region, so a hire is always within
one region and the yield, the season and the endowment of the job are those of one place.
The round visits lanes in ascending `RegionID`; the lanes share nothing (and `MaxOpen` is
**per region**, 13, so that no lane is first-come against another, the rule of
`spec-world.md` 10.1.1 for a new cap).

5.3.3. **Order within a lane.** The offers are ordered by `Reserve` ascending, equal
reserves by `K(t, id)` then by agent ID. The vacancies are ordered by `Bid`
descending, equal bids by `K(t, id)` then by agent ID. Each comparison is a total order
(architecture.md 7.6). No agent is favoured by its ID: the order is by value, and a tie by
the stateless key.

### 5.4 Matching and the wage

5.4.1. For each offer in order, in its lane, the round takes the first **open** vacancy in
order for which

```
Bid >= Reserve                      the pair can agree
Left >= Households                  the vacancy still has the positions (an offer is indivisible)
wage >= 1                           (the wage below)
```

and tries to form the hire (5.5). If formation succeeds the vacancy's `Left` falls by the
offer's `Households` and the next offer is taken. If no open vacancy qualifies, the offer's
households are **unmatched** this tick (`NoVacancy`); the offer is not tried again.

5.4.2. **The wage.** The wage per household-tick of the pair is

```
wage = Reserve + halfEven( WorkerShareBP * (Bid - Reserve) / 10 000 )
```

with `WorkerShareBP` of the employer's role block (default 5 000). The rounding is half to
even as in `spec-clearing.md` 5.2; the result lies in `[Reserve, Bid]`, so the worker never
gets less than it asked and the employer never pays more than it bid. At the default the wage
is the midpoint of the two limits, the price of a trade of the book and of a contract of the
round of MER (`spec-clearing.md` 5.1, `spec-merchants.md` 9.4.3). The share is one world-data
lever; the wage itself is derived.

5.4.3. **What the wage does not depend on.** It does not depend on how many offers or
vacancies there are: an unfilled vacancy does not raise its own bid and a crowd of offers
does not lower a reservation. The wage moves only with the two sides' own values, which are
set by their beliefs about prices (4.3). This is the position that keeps the module from a
wage spiral by bidding (EMP-D1) and it has a stated cost: wages do not clear the labour
market, so unemployment can persist while a wage stays (EMP-A5; the adjustment is the
spawner's, the ladder's and relief's, not the wage's).

5.4.4. **An offer is indivisible; a vacancy is not.** An offer of `m` households is matched
whole to one employer (5.5 makes all `m` executions or none); the vacancy's `Positions`
fall by `m`. This is the shape of `spec-merchants.md` 9.4.4 (the offer is a block, the take
is divisible) for the same reason: one job per agent (3.4.3).

### 5.5 Formation and the escrow

For the tentative pair of an offer of agent `W` (`m` households, reservation `R`) and a
vacancy of employer `P` (recipe `r`, `Duration` `d`) at a wage `u`, the world checks in this
order. The first failure ends the attempt with its reason, counted in `LabourRound`, and the
offer goes on to the next vacancy. What the failure closes depends on its kind. Check 1 closes
the whole lane for the rest of the round. Checks 2, 3 and 9 do not depend on the size of the
block and **close the vacancy**. The others (4 to 8 and 10 to 12) depend on the block: a failure for `m` households lowers
the vacancy's **fit** (the largest block it can still take) to `m - 1`, so a later offer of `m`
or more skips it without a new count and a smaller one may still be formed; a vacancy whose fit
reaches 0 is closed, and its remaining positions are counted once as unfilled with the reason
of the last failure. The wage below 1 of 5.4.1 belongs to the pair and skips the vacancy for
this offer only.

1. **`too_many_hires`**: the hires open in the lane's region are `MaxOpen` (13). The lane is
   closed for the rest of the round. It is the round's own check and is not a reason of a start
   (the order of the reasons of a start is `spec-production.md` 4.2's; the checks below that
   name one keep its order, and 6.1.3 states the cuts that happen after formation).
2. **`not_permitted`**: with the institutions module on, a right in force bars the recipe
   for the pair: the (executor, principal) pair is tested as a `Work` request is
   (`spec-institutions.md` 5.2, 5.3), the principal by default and the executor by the right's
   field `Who`; a traineeship's executor is exempt.
3. **`zero_yield`, `no_site`**: the effective factor `E` of `spec-production.md` 6.3 of `r` for
   the region and the season, with `HireYieldBP` for the trainee factor and `tech` (6.1.1), is 0.
4. **`no_tool`**: for some `Requires` tool `t` of `r`, the employer's unreserved items of `t`
   are below `committed + m`.
5. **`no_facility`**: with the infrastructure module on, for some class `c` in the recipe's
   `Needs`, `committed_c + m` is above `cap_c` of the employer in its region (`committed_c` the
   executions already committed to the class this tick by its own request and its hires, as
   `committed` is for a tool; `spec-infrastructure.md` 4.2.2).
6. **`no_input`**: for some input `i` of `r`, the employer's unreserved stock of `i` is below
   `(committed + m) * q_i`, with `committed` the executions of its own request, its trainees'
   and the hires formed so far this round.
7. **`no_endowment`**: with the resources module on, a draw of `r` on an endowment is above what
   the snapshot of P2.3 holds for `m` executions (`m * d_i > A_i`: a hire that could never draw
   even if it were alone). **No reservation order**: the check does not subtract the requests
   already made, so a hire is not first-come against the endowment or against the own requests
   of P2.5; a scarce endowment is shared by the ration of P2.6, proportionally, with every other
   request (`spec-resources.md` 5.3, `spec-world.md` 10.1), and a hire that passes this check may
   be cut there (5.5.1, 6.1.3).
8. **`no_title`, `rent_funds`**: with the tenure module on, by the rules of
   `spec-land.md` 4.7.2 as **bounds**: `no_title` by the rule of 4.5.1 (the stock and the
   holdings are fixed for the tick), and `rent_funds` only when the employer cannot pay even the
   least rent the job could owe (0 when its own entitlement covers the draw). The rent the
   ration really leaves is decided at P2.7 (6.1.3).
9. **`not_unlocked`, `no_target`, `store_refused`, `nothing_to_repair`**: with the
   infrastructure module on, the build reasons that do not depend on the size of the block
   (the same tests as limits 2 to 6 of `spec-infrastructure.md` 4.2.2, with the employer as
   principal, the running count of builds included). They **close the vacancy** (as checks 2 and 3).
10. **`facility_cap`**: the same source; it depends on `m` and lowers the vacancy's fit
   (as checks 4 to 8).
11. **`no_room`**: with the storage module on, the principal's room bound `k_room` of
   `spec-storage.md` 5.3.2, evaluated for the executions already committed to it plus `m`, is
   below `m` (the output of the job would not fit its store). The start of P2.7 is the authority,
   as for a trainee (`spec-storage.md` 5.3.1, 5.3.6): this check only avoids a hire that the
   start would cut.
12. **`employer_funds`**: the employer's available money is below the **bill**
   `u * d * m` (checked, 128 bits).

When all pass, the world, atomically:

1. takes the lowest-ID open Escrow account of the employer's region that no open contract or
   hire uses, or opens one (kind Escrow, in that region), and transfers the bill from `P` to it
   under `wage_escrow`; it cannot fail, the cash was checked in check 12;
2. creates the record of 3.4.1 with the next hire ID, `Left = d` ticks and the executions `m`;
3. withdraws `W`'s own request from the list of P2.5 (4.1.1) and adds a **hired request** to the
   list that the ration of P2.6 reads: executor `W`, principal `P`, recipe `r`, `m` executions
   (`k0` for the ration is `m`);
4. adds `m` to `P`'s committed executions;
5. emits `HireFormed`.

5.5.1. **Everything settlement needs is checked and moved before the hire exists**, as for
the contract of `spec-merchants.md` 9.5: the wages of the job are in escrow, the inputs and the
tools are in the employer's account and counted against its own and its trainees' requests, the
site yields. The only things that can still cut a hire are the ration of a scarce endowment
at P2.6 (a proportional share, with no priority for the hire: check 7 only refuses a hire that
could never draw), the rent the ration leaves and the room of the principal's store at the
start, which are not errors (6.1.3).

5.5.2. **No negotiation after formation and no late entry.** The wage is fixed for the whole
job at formation, and a worker that is matched cannot take another offer.

### 5.6 What the round does not do

- It does not touch the book. A hire is not an order, a bid or an ask, and `Clearing` is
  unchanged (ADR 0005 for the shape).
- It does not price labour in a market: there is no price of labour published and no belief
  about it. What a consumer or an indicator wants to call "the wage" is measured (8.3,
  10.2).
- It does not let a worker choose the recipe or the employer, and it does not let an employer
  choose a worker: the first-fit rule does (5.3, 5.4).
- It does not ration a scarce endowment among hires: a hired request joins the others in the
  ration of P2.6 (`spec-resources.md` 5.3), proportionally, with no priority for the employer.
- It does not change a belief, a target or a want: all of that happens in `Trade` and
  `Revise` as before.

## 6. The hired job

### 6.1 Start

6.1.1. **At P2.7**, in the rotation order with every other start, the hired request starts as
a job of its executor, with these differences from a self-employed one (`spec-production.md`
4.3, 4.4, 5.1, 6.3):

| Matter | A self-employed job | A hired job |
|--------|---------------------|-------------|
| Executor | the agent | the worker (its weight bounds `k`) |
| Principal | the agent | the employer |
| Inputs, `recipe_input` | from the executor's account | from the **principal's** account (as a trainee's, `spec-production.md` 4.3.2) |
| Tools held and worn | the executor's | the **principal's**: `k` unreserved items, each wearing `WearPerUse`, `wear` as always |
| Output at completion | to the executor | to the **principal** (`spec-production.md` 5.3.1: the principal's account) |
| Effective factor `E` | site * shock * tech * abundance | site * shock * tech * `HireYieldBP` * abundance (the `trainee` factor of 6.3 is replaced; `tech` is the technology factor, `spec-technology.md` 5.1.1, 1 when the module is off) |
| Facilities (`Needs`, `Builds`, `Repairs`) | the executor's | the **principal's**, judged in the principal's region (6.1.2) |
| Wage | none | paid each working tick from the hire's escrow (6.2) |

6.1.2. A **tool** on a hired recipe is the facility (`spec-production.md` 6.7, the "tool that
stands for the building"): the employer holds it, a hired household needs none, and one item
is worn per execution. A tool is not the only form of a facility: a class of
`spec-infrastructure.md` that the recipe `Needs`, `Builds` or `Repairs` is the principal's and is
judged in the principal's region. `FR-PRO-033` (a trainee-eligible recipe has no tool) does not apply:
that rule is about the trainee's own skill and tool; here the tool is the principal's.

6.1.3. **A ration, rent, facility or room cut.** If the ration of P2.6 grants fewer executions than `m` (a scarce
endowment), or the rent the ration leaves is more than the principal can pay (`spec-land.md` 4.7.2), or
the principal's facility capacity `k_cap` of `spec-infrastructure.md` 4.2.2 or its room bound `k_room` of
`spec-storage.md` 5.3 is below `m` at the start, the start has
`k = min(m, grant, live inputs, k_room, k_rent, k_cap)`. The unstarted executions' wages,
`u * d * (m - k)`, go back to the employer at once under `wage_refund`, and the record's
executions are `k`. No `HireEnded` is emitted (the hire goes on); the refund is counted in
`LabourRound.Refunded` of the tick (15.1) and is not repeated in the `HireEnded.Refunded` of the
end. If `k` is 0 the job does not start and the hire ends with the cause
`not_started`, the whole escrow refunded, the worker unmatched for the tick (it has no job
and does not start its own: its own request was withdrawn). This is the one place where a
matched worker ends a tick idle, and `LabourRound` counts it (`NotStarted`).

### 6.2 Pay

6.2.1. A hired job is paid **one tick at a time, at the position where the tick is taken**:
at P2.7 for the start tick and at P2.1 for each later tick including the completing one. The
amount is `wage * Executions` of the job at that moment (the executions fall if the worker's
weight falls, `spec-production.md` 5.4), transferred from the escrow to the worker under
`wage_pay`. It cannot fail: the escrow holds at least `wage * Executions * ticks left` by the
invariant of 7.4.

6.2.2. The wage is income in the tick it is paid (`Y_t` of 8.1), before the worker's `Trade`
of the same tick, and it is netted from the employer's income in that tick (`V_t`), not at
formation: the escrow is the employer's money in a different account, not yet a cost.

### 6.3 Completion

6.3.1. When the job completes (`Remaining` reaches 0 at P2.1, or at P2.7 for a one-tick
job) the output is created in the principal's account by the ordinary completion step
(`spec-production.md` 5.3), the last tick's wage is paid, any remainder of the escrow (zero
unless the executions fell during the job; a cut start of 6.1.3 was refunded at the start) is returned to the
employer under `wage_refund`, the Escrow account is released to the free list, and the hire
ends with the cause `completed` (`HireEnded`, 15.1).

6.3.2. The worker is free again at the next tick's P2.5: it offers again, or starts its own
job, by its own decision.

6.3.3. `RecipeExecuted` is the ordinary event of the completion (`spec-production.md` 14.1):
its `Agent` is the executor, the worker, as for a trainee. A measure of value added by
recipe is unaffected by who held the principal's account. **Order at a completion:**
`RecipeExecuted` first (the output is created), then `HireEnded` (`completed`), both at P2.1
(or P2.7 for a one-tick job), hires in ascending hire ID within the position; the `LabourRound`
of the tick follows at P2.8b.

6.3.4. **An employer that is also a trainer.** Its trainees' requests and its hires both take
inputs from its account; at formation the hires count the trainees' `k0` as committed
(5.5, check 6), so the sum of requests never exceeds the stock. A trainer with trainees may
be an employer; a trainee may not be hired (`in_traineeship`).

### 6.4 Interruptions and default

The inputs of a started job are already consumed and nothing in the job is a ledger holding
(`spec-production.md` 5.4), so ending a hire early moves only the escrow. There is no
penalty and no bond (as for a contract, `spec-merchants.md` 9.7): a party that defaults
loses what the removal itself costs it, and the other keeps every wage paid.

| Event | The worker's job | The escrow remainder | Cause of `HireEnded` |
|-------|------------------|----------------------|----------------------|
| The job completes | completes; output to the employer (6.3) | to the employer (`wage_refund`) | `completed` |
| The worker is removed (the ladder, the population module, a consumer's removal) | abandoned, `agent_removed` | to the employer | `worker_left` |
| The worker changes role (the spawner, a completed traineeship) | abandoned, `role_changed` | to the employer | `worker_role` |
| The worker's account changes region (a migration journey leaves, a forced `Relocate`) | abandoned, `region_changed` | to the employer | `worker_moved` |
| The worker's weight falls | not abandoned: `Executions` scaled by `spec-production.md` 5.4 | refunded at the end, with the rest | none |
| The employer is removed | abandoned, `principal_left` | to the **employer's own account** (`wage_refund`), before the waterfall of `spec-credit.md` 9.1 | `employer_left` |
| The employer changes role (the spawner) | abandoned, `principal_left` | to the employer | `employer_role` |
| Nothing could start (6.1.3) | none | the whole escrow to the employer | `not_started` |
| A checkpoint is taken | nothing; the hire and the job are saved (16) | | |

6.4.1. **Why the escrow goes to the employer's own account when the employer is removed.** The
escrow is a part of the employer's money that happened to be in another account. It returns to
the employer's account first (step 2c of `spec-agents.md` 3.6) so that the creditors' waterfall
of `spec-credit.md` 9.1 (step 2d) sees it, and what is left moves to the estate with the rest
of the account (step 3). The worker, which already holds every wage it earned, is free the next
tick. A worker is not paid for ticks it did not work.

6.4.2. **The worker's loss.** A worker whose job is abandoned has lost the rest of the job's
pay and the labour already put in beyond the paid ticks, which is none: it is paid a tick at
a time. It is free at the next P2.5. The employer's loss on a removed worker is the inputs
already consumed (`spec-production.md` 5.4.1: work in progress is worth something only if
finished).

6.4.3. **The weight of a worker rises** during the job: nothing changes (`spec-production.md`
5.4); the new households did not start the job.

### 6.5 Relocation and removal

6.5.1. **The refusal `open_hire`.** With the module on this spec registers the `region_bound`
refusal `open_hire` of `spec-regions.md` 6.5.2, open for an agent that is the principal of an
open hire. The relocation of MIG and INT returns `ErrRegionBound` naming it and changes
nothing. Its instance keys are the hire IDs. The reason is that the principal's account and
its Escrow accounts are tied to a region and the output of the job would appear in another
region without having been made there. This is the rule of N-090 for a module that keeps an
account committed. The worker needs no refusal: its job is abandoned when its account changes
region (`spec-production.md` 5.4, `spec-migration.md` 8.2), which is the row above, and the
`ErrRegionBound` of an agent that is on a migration journey stands (`on_journey`).

6.5.2. **Removal.** `spec-agents.md` 3.6 gains one step, **2c**, after LND's 2b (titles to the pool)
and before CRD's 2d and 2e, so before the money of the removed agent moves to the estate:

- as a **worker** with an open hire, its job is abandoned (step 1 there), the hire ends
  (`worker_left`) and the remainder of the escrow returns to the employer;
- as an **employer** with open hires, for each hire in ascending hire ID: the worker's job is
  abandoned with `principal_left`, the remainder of the escrow returns to **the employer's own
  account** (so that the waterfall of step 2d sees it), and the hire ends (`employer_left`).

The events of one removal, in the order of `spec-world.md` 6.11 item 8 (every event of a
removal precedes `AgentBankrupted`; the steps are 2b `TitleTransferred`, 2c here, then CRD's 2d
and 2e, INS's 2f and CAP's 2g): first, as 6.11 says,
the trainees of a trainer being removed; then the hires of an employer being removed, hire by
hire in ascending hire ID, each `RecipeAbandoned` of the worker's job (`principal_left`) and
then `HireEnded` (`employer_left`); then the agent's own list (`RecipeAbandoned` for its job,
`HireEnded` (`worker_left`) if it was a worker, `TraineeshipEnded`, `ContractSettled`,
`AgentBankrupted`, `EstateOpened`). A role change of an employer emits, for each hire in
ascending ID, `RecipeAbandoned` (`principal_left`) and `HireEnded` (`employer_role`), then its
own `RecipeAbandoned` and `RoleChanged`; that of a worker emits `RecipeAbandoned`
(`role_changed`), `HireEnded` (`worker_role`) and `RoleChanged`.

## 7. The ledger form

### 7.1 Reasons

Three reasons join the registry of `spec-ledger.md` 7.2, **all internal** (money between
non-boundary accounts), owned here:

| Reason | Of | Kind | Between | When |
|--------|----|------|---------|------|
| `wage_escrow` | money | internal | agent to Escrow: the employer's bill at the formation of a hire | P2.5a |
| `wage_pay` | money | internal | Escrow to agent: the worker, one tick's wage `wage * Executions` | P2.7 for the start tick, P2.1 for the others |
| `wage_refund` | money | internal | Escrow to agent (the employer; a removed employer is paid to its own account first, 6.4.1) | the end of a hire; 6.1.3 |

7.1.1. Reasons and money-flow IDs share one namespace (`spec-money-supply.md` 4.5): a flow
named `wage_pay` is a collision (`ErrReasonCollision`), as for the other built-ins.

### 7.2 The Escrow account

The kind Escrow of `spec-ledger.md` 3.1 is reused: "money held for a delivery contract" is
amended to "for a delivery contract or a hire". An account holds the money of **one** open
contract or hire and nothing else; its region is the employer's region at formation; it is
reused by the next contract or hire and never closed, for the reason Transit and the
contract's Escrow accounts give (a closed slot stays, so a closing per hire would add a slot
per hire). The free list is derived from the ledger, the contract list and the hire list and
is not saved.

### 7.3 Cash only

The escrow holds money and nothing else. There is no claim and no credit: an employer that
cannot fund the bill of a job does not hire (`employer_funds`), and the cost of that is
liquidity (EMP-A4). A wage advance, a loan to an employer and a wage claim on an estate are
`spec-credit.md`'s: a loan to an employer funds its upkeep and tier 2 only, and the wage bill is not in the default request (`spec-credit.md` 11.4).

### 7.4 Identities and the audit

7.4.1. **Conservation.** Every flow of the module is a transfer between two non-boundary
accounts, so every identity of `spec-ledger.md` 9.1 holds and the sectoral identity of 9.3
holds with the Escrow accounts in the contract sector: the money an employer moves into
escrow leaves the household sector (and so `MoneyContract` and `economysim.money.supply`
rise), and returns to it as wages and refunds.

7.4.2. **The audit** (`WithAudit`, and the restore audit of `spec-checkpoint.md` 6.2.2) checks,
for each open hire: the Escrow account exists, is of the employer's region and is used by no
other contract or hire; its balance is at least `wage * Executions * ticks left` of the
worker's job, the difference being only the refund due at the end; the worker's job names the
employer as principal and is the recipe of the hire; and the money in all Escrow accounts is
the sum of the open contracts' and hires' held amounts. A failure is `ErrAuditFailed`.

7.4.3. **No faucet and no sink and no `BoundaryFlow` row.** The module appears in
`BoundaryFlow` nowhere, and N-107 and N-021 are applied by this absence: a wage is not a
grant or a levy of `spec-money-supply.md` and creates and destroys nothing.

## 8. Welfare and the target-income rule

### 8.1 What the record gains

The welfare record of a tick (`spec-welfare-and-wages.md` 2.1, 13) is unchanged in shape. Two
terms join the definition of `income_t`:

```
income_t = R_t + Y_t + K_t + K^own_t + F_t + G_t - S_t(Input) - W_t - V_t
```

`Y_t` is the money the agent received as **wages** in tick `t`: the sum of `wage_pay` into its
account at P2.1 and P2.7. `V_t` is the wages it **paid** in the tick: the sum of `wage_pay` out
of the escrows of the hires of which it is principal, in the tick the working tick is taken
(6.2.2); the escrow itself is not netted when it is made. Wages received are earned income:
`Y_t` stays in the earned ratio (2.1.7 there) and in the observed wage (4.3 there), and is not
`F_t`, `G_t` or `K^own_t`. The wear cost `W_t` of a hired execution is the **principal's**
(2.1.4 there, "the executor" reads "the principal"): the tool is its capital.

### 8.2 The target-income rule

8.2.1. **A wage earner.** The rule of `spec-welfare-and-wages.md` 5 is the household's own and
is applied as before (`spec-agents.md` 7.1.2): the work share `s` is a function of its ratio
against its target and band, with the survival rule and the floor of 7.1.3 over it. What the
share now decides, for a household with no job of its own, is **how many of its households it
offers** (4.3.1). The wage is income in the ratio, so a higher wage means fewer working ticks
to reach the target: **the backward-bending supply of labour that the rule always had, now for
hired labour** (STB-L14, EMP-A1). The floor `MinWorkShare` counts a hired tick as a working tick,
so a worker that is hired often enough is not held to work by the floor, and one that is not
hired is held to *offer*: the floor makes it offer every tick, and the round decides whether it
works.

8.2.2. **An employer.** The rule applies to the employer's **own** labour exactly as before: its
own job is requested by its work share, and it rests when its ratio reaches the top of the band.
Its **hires are not governed by the rule** (4.3.4). The reasons:

- an establishment is not the household's labour: the hired households' ticks are another
  household's, and an employer whose ratio is above its target does not make the hired
  households' targets met;
- a rule that cut the vacancies of an employer at its target would make the demand for labour
  fall as the employer's margin rose, a second unit-elastic link beside the worker's (L14), in
  the same sign: a price rise lifts the margin, the employers rest, fewer households work, supply of
  the good falls and the price rises again. It would also put the whole establishment of an
  employer to rest and to work together at the pace of one window (a lockstep, L22, L34);
- the profit-seeking rule has its own stop, the capacity and the cash (4.3.3), and the cost it
  leaves is stated: an employer above its target keeps its profit and hoards it up to the cap of
  `spec-agents.md` 7.4.4 (EMP-A3, whose cure list `spec-saving-and-investment.md` 8.3 extends), which is the loop 11 is about.

8.2.3. **What an employer's ratio shows.** Its income is the sales of the whole establishment
less inputs and wages (`R_t - S_t(Input) - V_t`) over its own upkeep. An employer with four hired
households reaches its target ratio with a quarter of the labour of its own and so rests its own labour;
the establishment runs. That is the rule, not a defect.

8.2.4. **The floor for a principal.** A tick in which the agent is the principal of a hired
job in progress is a **working tick** for the floor of 5.3 there (the flag of the record), so an
employer that runs an establishment and rests its own bench is not held to work by the floor.
The working household-ticks that the observed wage divides by are the agent's **own**
executions only (8.3).

### 8.3 The observed wage

8.3.1. **A worker.** A hired household-tick is a working household-tick: for a worker with a
hired job of `k` executions in progress it is `k`. For a `Hireable` role **the clause of
`spec-welfare-and-wages.md` 4.3 for a role with no recipes ("it bought or sold in the tick")
does not apply**: that clause makes every tick of a trader a working tick, and a worker buys its
upkeep every tick, so with it a worker that was never hired would have a working tick of its full
weight each tick and the wage would be divided by weight rather than by hired executions. A tick
of a `Hireable` role is a working tick if it had a job in progress (its own, or hired), and its
working household-ticks are the `k` of that job. The observed wage of the agent is therefore the
wages it received over its hired (and own) household-ticks, and the observed wage of a hireable
role in a region is what its households were paid per working household-tick (own-use ticks
excluded as before). It is the occupation signal of the spawner for the role
(`spec-spawner.md` 6.1) with no change. The same text makes the floor of 8.2.1 true: a worker that
is not hired has no working tick, so the floor binds on it and holds it to offer.

8.3.2. **An employer.** Its observed wage is its income over its own working household-ticks, and
includes its profit on the establishment (EMP-A6). The spawner reads it as the role's occupation
signal like any other; the consequence is stated in the register entry, not hidden.

### 8.4 The pooled regional wage and the cost anchor

8.4.1. The pooled regional wage of `spec-welfare-and-wages.md` 4.3, which the cost anchor of
`spec-production.md` 9.3 reads, is the working-household-tick-weighted mean over **the agents of
the roles that have no employer block**. The agents of an employer role are left out of the sum
and of the count. Two reasons: the cost anchor is the price of a tick of labour and an employer's
income per own tick is a profit that includes the work of others; and with them in, the more an
employer hired the higher the anchor would read and the higher the cost floor of its sellers
(`spec-beliefs.md` 3.2.2), a loop through the belief bounds that this spec would otherwise
create (EMP-D6).

8.4.2. The rule is **stateless**: it is a function of the role's configuration and the rings, so
a restore needs nothing for it (`spec-checkpoint.md` 4.5 F5). The pooled figure is unknown, as
before, when no agent outside the employer roles has a working household-tick in its window, and
the starting wage of a warm start is used then.

8.4.3. **What the anchor now measures** in a region where an employer role hires: the wages the hired
households were paid, and the earnings of the self-employed. A region where every household is hired by an
employer has an anchor equal to the hired households' wage; the employer's profit is not in it. That is the
reading of `spec-production.md` 9.3: "the return to market labour".

### 8.5 The window

8.5.1. A hired job does **not** lengthen a worker's window: the worker is paid every working
tick, so a job is not a gap in its income (STB-L19 does not act on a hire). Its window is
`WelfareWindow` or the longest of its own recipes as before.

8.5.2. An employer's window covers its hire list, since the list is a subset of its recipes
(3.2.1), so a long hired recipe lengthens it as a long own recipe does: the sales of a hired
job's output are as late as any.

### 8.6 What this spec does not change in WEL

8.6.1. **`InKind`.** The share of income received as goods (`spec-welfare-and-wages.md` 6) is
not applied to wages in version 1 (open question 1); a `Hireable` role with `InKind` above 0 draws the
warning `inkind_wage_ignored`. The trainee's board of 6.3 there is unchanged.

8.6.2. **The role economics check** (`spec-welfare-and-wages.md` 7) is a report as before. For a
hireable role the figure it reports is the wage the round gives at the reference prices, `R +
phi * (Bid - R)` with `R` 0 and the bid of 4.3.3 at the reference prices of the employer roles'
hire lists; the validation warnings `hire_never_profitable` and `wage_below_upkeep` (14.2) state
the two cases that matter, a hire list nobody would post and a wage below the upkeep of a tick.

## 9. The employer's budget and targets

### 9.1 The view

The view of `Work` and `Trade` (`spec-agents.md` 6.4.1) gains a group, present only with the
module on and only for a role with the field or the block:

| Role | Fields |
|------|--------|
| `Hireable` | `Hireable` (true); the reservation of 4.3.2 is computed by the default from the per-recipe group already in the view |
| employer | `Employer`: the hire list (recipe IDs), `MaxHires`, `WorkerShareBP`, `HireYieldBP`; `Hired`: the executions of the hires it has open, and the number of hires; `Open`: the hires open in its region against `MaxOpen` (so that it does not post into a full lane) |

The per-recipe group already carries `MaxExecutions` and the reason for a recipe, the
effective yield `E`, and the beliefs' values; for a recipe of the hire list the group adds the
**hired yield** (`E * HireYieldBP / 10 000`) beside it, so a consumer's strategy does not
recompute it.

### 9.2 Planned executions and targets

9.2.1. The **planned recipe** `r*` and its executions `k*` (`spec-agents.md` 7.2) are, for an
employer, its own: `r*` by 7.1.4 among its recipes and `k* = w`. The hires add a second
planned job: the recipe `r_h` the default would post (4.3.3, evaluated for the next tick:
`StartsIn` 1) and `H*`, the number of positions it would post, bounded by cash as it is after
this tick's tier 1. The inputs and tools of the targets of 7.3 are then

```
Job_c = InputJobs * ( k* * q_c(r*) + H* * q_c(r_h) )     for an input c of either recipe, StartsIn = 1
      + k* * [c is a tool of r*] + H* * [c is a tool of r_h]
```

so an employer buys, for its next tick, the inputs of the establishment it expects to run. The
rest of 7.3 to 7.5 is unchanged: the same tiers, the same cap, the same limit-price draws.

9.2.2. `H*` is a function of the view only (no state): the number the same rule posts on the
next tick if nothing changes. A tick on which the employer is busy (`StartsIn` above 1) has
`H* = 0`, since a busy agent is not called (4.1.3).

### 9.3 Payroll

9.3.1. **The payroll is paid before `Trade`.** The escrow of a hire formed at P2.5a leaves the
employer's account before P3.2, so the `Money` the view shows `Trade` is net of the wages of
the jobs of the tick, and the budget stack (7.4.3) works on what is left: tier 1 (upkeep) and
tier 2 (inputs and tools) are paid from it as before. There is no payroll tier and no new
rule: the wage bill is a **cost of the job**, paid at its start, as an input is, and in the
same order of priority (a household's upkeep first, by the check `by_cash` of 4.3.3, which
leaves `UpkeepCost` in hand).

9.3.2. **The cash check is a guarantee, not a plan.** `by_cash` sizes the vacancies on the
money the employer has when it posts. It does not reserve money for the next tick's
vacancies: the next tick's positions are again cut by the cash then, and a poor employer hires
less (EMP-A4).

### 9.4 A pulse of inputs

An employer that expects `H*` hires buys their inputs a tick before: if the hires do not come
(no households offered, a closed lane) the inputs are stock, bought on the strength of an expected
margin. That is the buffer pulse of AGT-A9 with a second source, and the cost of the
design (EMP-A2): the stock is sold by the usual rule and the production halt keeps it from
growing (`spec-agents.md` 7.1.5). The inputs of unfilled vacancies are the employer's, not the
worker's.

## 10. Unemployment, the wage bill and the measures

### 10.1 What unemployment is here

10.1.1. A household is **unmatched** in a tick if it was offered (admitted) and no hire was
formed for it: the `NoVacancy` count of `LabourRound`. A household that rested under the rule
(its ratio at the top of the band), that was in the own-use mode, that was a trainee, away or on
a journey, offered nothing and is not unmatched. The **unemployment of a window** is the sum of
unmatched household-ticks over the sum of offered household-ticks, by region and role: a share
of those who wanted work and did not get it.

10.1.2. It is **measured, never set**: no rule of the library reads it. A policy actor that
wants it (a relief rule, a public work) reads it from IND (`spec-policy.md`), and the
engine holds no aggregate of it (ADR 0001 decision 7).

10.1.3. **What it is not.** It is not a count of households with no income (a household with a
recipe offers and falls back to its own job, and is not unmatched in income terms though it is
in count); it does not say whether the unemployment is **structural** (no vacancy at any wage:
`NoVacancy` with `Unfilled` of every employer at 0), **frictional** (a vacancy exists and its
employer lacked cash or inputs: `Unfilled` by reason `employer_funds`, `no_input`) or
**cyclical**; the counters by reason let a consumer split it. The rate counts households, not
agents.

### 10.2 What the library reports

| Where | What |
|-------|------|
| `LabourRound` (15.1), per region and tick | households offered, hired, unmatched; positions posted, filled, unfilled by reason; wages paid and refunded; rows by worker role and by employer role |
| `RoleRow` (15.4), per (region, role) at a boundary | `Employed` (households of the role on a hired job), `Employing` (hired households working for agents of the role) |
| `Hires` (15.4), by hire ID | the hires in flight |

10.2.1. **Asks of IND** (`spec-indicators.md`, section 12 here): three presets,
`Unemployment` (the rate of 10.1.1 over a window, by region and role, with `Offered` beside it
and the split by reason of 10.1.3), `WageBill` (the sum of `wage_pay` over a window by region and
the compensation row of the income view of `spec-indicators.md` 7.5, which says "when EMP exists
the income view gains a compensation row": the labour share is the wage bill over value added),
and `Employment` (the hired household-ticks over the households of the hireable roles). They
are measures of a window of events and one snapshot; none needs a field beyond those above.

10.2.2. **N-130 item 1** (`IncomeLast` and the other income fields) is not changed by this spec;
the wage bill is measurable from `LabourRound.Paid` without it.

## 11. The wage and demand circuit

This section is the employment side of the wage and demand loop of the roadmap and of
`spec-stability.md` L57 (whose demand side is DEM's). It states one identity, its closed form,
what closes the circuit in the library, and one lever.

### 11.1 The identity

In a closed economy with one tick of job, value `v` per execution net of inputs, `h` hired
executions per tick, a wage `phi * v` (the worker share `phi = WorkerShareBP / 10 000`, reservation
0) and the employers' profit `(1 - phi) * v` per hired execution, **money does not accumulate
anywhere in a steady state**: whatever the workers are paid they spend (they eat their
upkeep; they have no other use), so the value of sales equals the wages plus what the employers
spend:

```
sales = v * h  =  phi * v * h  +  S_e             (the input suppliers pass on what they get)
S_e   =  (1 - phi) * v * h                         what the employers spend equals what they keep
```

The employers' profit is spent or it piles up; if it piles up, the money in circulation falls
by the pile each tick and the circuit contracts until the profit it yields is what the employers
spend. **Hired employment is therefore set by the employers' spending, not by their
capacity.**

### 11.2 The closed form

With `E` employers, `u_e` units of the staple each of upkeep per tick, `a` units per execution
at price `p` (so `v = a * p`) and the employers' spending in units `S_e / p`:

```
h*  =  S_e / ((1 - phi) * a * p)  =  E * u_e / ((1 - phi) * a)                  employers eat their upkeep only
h*  <=  E * u_e * (1 + SpendMaxShare * SpendHalfTicks / 10 000) / ((1 - phi) * a)   the saving rule's cap as the outlet
```

capped at the places `E * MaxHires`. The second line is the cap of `spec-agents.md` 7.4.4: the
spending above upkeep of a household with a large excess tends to `SpendMaxShare *
SpendHalfTicks * UpkeepCost / 10 000`, an upper bound reached only for a large excess. The
bound holds with an input cost `c` as well, the suppliers passing on what they get.

The model of `doc/specs/models/emp/circuit.py` (200 workers, 10 employers of 10 places, `a` 6,
`u` 3, the halt of `spec-agents.md` 7.1.5, a fixed price) gives, over the last 3 000 of 6 000
ticks (`circuit.out`):

| `phi` (bp) | `SpendMaxShare` 0: hired / `h*` | 2 500 | 5 000 | 9 000 |
|-----------|----------------------------------|-------|-------|-------|
| 3 000 | 7.1 / 7.1 | 23.8 / 25.0 | 40.5 / 42.9 | 66.7 / 71.4 |
| 5 000 | 10.0 / 10.0 | 33.3 / 35.0 | 56.7 / 60.0 | 86.7 / 100.0 |
| 7 500 | 19.8 / 20.0 | 66.7 / 70.0 | 100.0 / 100.0 | 100.0 / 100.0 |
| 9 000 | 50.0 / 50.0 | 100.0 / 100.0 | 99.3 / 100.0 | 94.9 / 100.0 |

The first column is exact: employers who spend only their upkeep keep ten of a hundred places
filled at the midpoint (the halt posts no vacancy while their stock is unsold), so 190 of the
200 households are unmatched however many offer.
The others sit just under the bound, which the cap's concavity explains (the bound is the
asymptote). At `phi` 9 000 with a large cap the last cells are below places because the
model's hiring is bursty (the halt) and the workers' purchases are capped at their upkeep.

### 11.3 What closes the circuit in the library

| Piece | How it makes `S_e` large | Owner |
|-------|--------------------------|-------|
| The wants of the employer role | tier 4 of the budget: the ladder of `spec-demand.md` funded from what the cap leaves, so the employers' profit buys goods | `spec-demand.md` 4.5, `spec-agents.md` 7.4.3 |
| The saving rule's cap | the outlet is bounded by it; raising `SpendMaxShare` or `SpendHalfTicks` raises `h*` | `spec-agents.md` 7.4.4, 12 |
| A grant to the workers, or to the employers' pool | adds a term to the workers' spending, so `sales = phi v h + G + S_e` | `spec-money-supply.md` 4, 9 |
| A levy on the employers' hoard (a holding charge) | takes the pile and returns it by a grant elsewhere | `spec-money-supply.md` 9.3 |
| Exports | the outside buys: `sales = phi v h + S_e + X` | `spec-external-sector.md` |
| Investment (the employers build) | profit becomes demand for goods made by workers (SAV 8.3 lists the outlets) | written: a `Saver` block on the employer role and a `Worth` row for the workshop class (`spec-saving-and-investment.md` 5.3.2, 5.9) |
| The worker share `WorkerShareBP` | `1 - phi` is the share of the margin that must be spent | this spec, 3.2 |

11.3.1. **A pack with employers and no outlet** gets the first line of the table: hired
employment is bounded by the employers' upkeep. The validator warns (`employer_no_outlet`, 14.2):
an employer role with no taste row (the demand module off, or no want in its ladder) and no money
flow that targets it. The warning is informational; a pack may want that world (a small
employer class in a subsistence economy).

11.3.2. **The worker share is the lever with a cost.** Raising `WorkerShareBP` raises `h*` and
the workers' ratio at once (the workers' units per household-tick are `phi * a * h / N`) and
lowers the employers' margin per hire; at 10 000 the employer earns nothing on a hire and has no
reason to post it (the margin test of 4.3.3 is on the bid, not on the margin, so the default
still posts; a consumer's strategy would not). It is **a share of what the beliefs of the two
sides produce**, not a wage.

### 11.4 What this does not say

11.4.1. It does not say the real economy is a one-good circuit: the identity is an aggregate
statement (the value of sales is the value of wages, profit and inputs) that holds for any
number of goods; the closed form is for one good with identical employers, and a model with
several goods multiplies through the same way (`sales = wages + spending`).

11.4.2. It does not claim a price: the model fixes it. In the library the price level and the
relative prices are the beliefs'. The identity holds in value, and the units follow.

11.4.3. **The mean welfare ratio of 1** of N-054 and `spec-viability.md` 4.7 still holds for the
closed economy with upkeep as the only demand: it is the aggregate over all households. The
employers' ratio is above it and the workers' below it in the collapsed state (EMP-A3), and the
mean hides the split, which is why `spec-indicators.md` 7.7.2 asks for the ratio of sums by
role.

## 12. Seams with the specs around it

Each row is an edit this spec asks of another file, stated as the change; section 21 does not
repeat them and the handoff holds them in one table with the exact text.

| To | Where | The change |
|----|-------|------------|
| `spec-production.md` (PRO) | 3.3, 4.2, 4.3.2, 4.4.1, 5.4, 6.3, 8.1, 10.3 | 10.3: the "trainee factor" in the derived overflow check of the largest yield factor reads "the larger of the trainee factor and, for a recipe on an employer's hire list, `Employer.HireYieldBP` (up to `MaxYieldBP`, ten times the trainee's)", so that a hired execution's output cannot exceed `MaxQuantity` (14.1). A hired job: the recipe on the employer's hire list is not `not_on_role` for its worker; inputs and tools come from the principal as a trainee's; a tool on a hired recipe is the principal's; the factor `hired` (`HireYieldBP`) replaces `trainee` in the effective factor; 5.4 rows `principal_left` and `region_changed` name the hire; `FR-PRO-033` says that a trainee-eligible recipe has no tool, which is not the rule for a hired one |
| `spec-agents.md` (AGT) | 3.6, 6.3, 6.4.1, 6.5.1, 7.1, 7.2, 7.3, 14, 15 | removal steps for hires (6.5.2); `JobRequest` fields; the view group (9.1); the checks of 4.2; the default's offer and vacancies (4.3); the planned hires and the targets (9.2); the event rows; no per-agent state is added |
| `spec-welfare-and-wages.md` (WEL) | 2.1, 2.1.4, 4.3, 10, 15.1 | the two terms of 8.1; the principal's wear; the hired working tick and household-tick, with **the "bought or sold" clause of 4.3 not applying to a `Hireable` role** (8.3.1); the pooled wage excludes employer roles (8.4); the non-goal "wages for employees" is now this spec's |
| `spec-world.md` (WLD) | 6.4, 6.11, 8.2, 9.2, 10.2 | positions P2.5a and P2.8b; the event order of 6.5.2 and of a completion (6.3.3); the table of cadences gains "labour round, every tick, P2.5a"; `MaxOpen` per region keeps O5's hazard off. **No row in 7.3 and no term in 7.3.1**: the round takes no word (5.3.1); a note that the module's ties use `K(t, id)` (10.1.1) |
| `spec-spawner.md` (SPN) | 4.2 | a role change of a worker or an employer ends its hire (the rows of 6.4) |
| `spec-estates.md` (EST) | 2.1 | the escrow remainder of a removed employer goes to the estate of the region as part of its money |
| `spec-ledger.md` (LED) | 3.1, 7.2, 9.3, 12.4, FR-LED-022, FR-LED-060, FR-LED-061 | the Escrow row reads "for a delivery contract or a hire"; three reasons, with their pairings in the family of `ErrReasonMismatch` (`wage_escrow` Agent to Escrow, `wage_pay` Escrow to Agent, `wage_refund` Escrow to Agent); the contract sector includes hire escrow; 12.4, FR-LED-022 and FR-LED-061 read "an Escrow account that no open contract **or hire** references is empty" and "money held for a contract or a hire" (7.4.2) |
| `spec-regions.md` (REG) | 6.5.2, FR-REG-060 | `open_hire` joins the registered refusals |
| `spec-observability.md` (OBS) | 4.1, 4.3.3, 5.3.4, 5.3.8, 6.2, 7.3 | three events; the rebuild of household money; two `RoleRow` fields and the `Hires` slice; five metrics; two `Warn` rows |
| `spec-checkpoint.md` (CHK) | 4.2, 4.2.3, 4.2.5, 9.3 | a section for the module (position and version in 16): the next hire ID and the hires in flight; the employer and worker references are checked as live agents; its diagnostic runs are sparse |
| `spec-indicators.md` (IND) | 7.2.2, 7.5, 9.1 | "there is no wage flow" now has one, `wage_pay`; three presets (10.2.1) |
| `spec-demand.md` (DEM) | 10.2 | `class_without_income` is not raised for a `Hireable` role in a world with an employer role: its income is wages |
| `spec-integration.md` (INT) | 4.7.2, 4.3.2 | 4.7.2: `AgentState` gains the hire of an agent as a worker and as an employer. 4.3.2: the fields `Offer` and `Hire` of a directive's job request are admitted by the labour round (5.2), not by the call: the call's check 1 does not refuse them and FR-INT-025's list of errors does not change |
| `spec-config.md` (CFG) | 3.2, 3.7, 7.3, 7.4.2, 8.1 | the module and `MaxOpen`, two role fields and the four fields of the block, two sentinels, nine warnings |
| `spec-merchants.md` (MER) | 9.5, 9.6.2, FR-MER-039, FR-MER-045 | the free list of Escrow accounts is shared with hires: "references" means an open contract **or** an open hire in 9.6.2 and FR-MER-039 (a contract takes the lowest-ID account that no open contract or hire references), and the audit of FR-MER-045 does not fail on a hire's funded account (its check of "exactly `price * quantity`" is for a contract's account) |
| `spec-storage.md` (STO) | 5.3, 10.2 | the output of a hired job goes to the principal, so the room bound and the pledge are the principal's, as for a trainee's job (5.3.6); `no_room` is a formation check of 5.5 (11); its row of 10.2 stands: no change |
| `spec-infrastructure.md` (CAP) | 4.2.2, 5.5, 6.1 | the checks 5 (`no_facility`) and 9, 10 (the build reasons) of 5.5; the facility is the principal's (6.1.2); `k_cap` joins `k` (6.1.3); `LabourRound` names the reasons (15.1) |
| `spec-land.md` (LND) | 4.7.2, 4.5.1 | check 8 (`no_title`, `rent_funds`) as bounds; the rent cut at the start (6.1.3); `LabourRound.NoTitle`, `RentFunds` |
| `spec-institutions.md` (ORG) | 5.2, 5.3 | check 2 (`not_permitted`): the pair is tested as a `Work` request is, the executor by `Who`; a traineeship's executor is exempt |
| `spec-saving-and-investment.md` (SAV) | 5.3.2, 5.9, 8.3 | the outlet of an employer's profit (11.3, 14.2) |
| `spec-technology.md` (TEC) | 5.1.1 | the factor `tech` of a hired job (6.1.1, FR-EMP-028) |
| `spec-credit.md` (CRD) | 11.4 | a loan to an employer funds its upkeep and tier 2 only; the wage bill is not in the default request (7.3) |
| `spec-stability.md` (STB) | 4.12, 5, 8 | the entries of 17, the hook row "EMP" removed |
| `spec-calibration.md` (CLB) | 6 | registry rows for the four fields |
| `spec-viability.md` (VIA), `spec-reference-chain.md` (REF) | 4.7, 8 | a steady state with employers is not a zero-profit state (open question 7); an overlay with an employer role and wants |
| `spec-stylised-facts.md` (STY) | 5 | a report fact for the unemployment share and the employer's margin |
| `rules/observability-and-logging.md` | 3.3.2, 4.3, 5.3, 5.4.1 | one log key (`vacancies`), three events, five metrics, the set of `economysim.reason` |

## 13. Parameters

All are fields of `Config` with a default (or "required"), a bound, a doc line and an owner
(architecture.md 9.5 to 9.7). **Every default is a placeholder** (N-015, N-027): no source for
the size of an establishment, the share of a margin that workers receive or the productivity of
hired households was read. The bounds are validation bounds in `bounds.go` (`spec-config.md` 4.1).

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Employment` | the labour round and the hired job: present means on (`spec-config.md` 3.2) | nil (off) | |
| Employment | `MaxOpen` | the most hires open at once **in a region** (5.5 check 1) | 1 000 (placeholder) | 1 to `MaxHireRecords` |
| Role | `Hireable` | the role's households may offer their labour (3.1) | false | a bool; needs the module |
| Role | `Employer` | the employer block; nil means the role does not hire (3.2) | nil | needs the module |
| Employer | `Recipes` | the hire list, a subset of `Role.Recipes` | required | at least 1, known, no duplicate, on the role's list |
| Employer | `MaxHires` | the establishment: hired executions in progress at once | 4 (placeholder) | 1 to `MaxEstablishment` |
| Employer | `WorkerShareBP` | the worker's share of the margin between its reservation and the bid, in basis points | 5 000 (the midpoint; placeholder) | 1 to 10 000 |
| Employer | `HireYieldBP` | the factor on the yield of a hired execution, in basis points | 10 000 (placeholder) | 1 to `MaxYieldBP` |

**New bounds for `bounds.go`** (provisional, with the reasons):

| Constant | Value | Reason |
|----------|-------|--------|
| `MaxEstablishment` | 1 000 | a hired execution per household of an employer's establishment; with `MaxQuantity` and `MaxPrice` the product of a bill (wage, `Duration`, executions) fits `int64` through the 128-bit helper, and an establishment above a thousand households is not one agent |
| `MaxHireRecords` | 1 000 000 | the hires open at once in a region; the Escrow accounts are at most this many (as `MaxContracts`) |

**Rule constants** (beside their code, not parameters, architecture.md 9.6): the position of the
round (P2.5a) and of the aggregate (P2.8b); the admission order and the reasons of 5.2; the
tie key `K(t, id)` (5.3.1) and the order of lanes, offers and vacancies (5.3); the wage formula and its
half-to-even rounding (5.4.2); the order of the formation checks and the closing of a vacancy
(5.5); the one release a working tick (6.2); the table of interruptions and its causes (6.4); the
destination of the escrow remainder (6.4.1); the rule that a pooled wage excludes the employer
roles (8.4); the reservation of 4.3.2 (the value of the agent's own best alternative); the three
reasons of 7.1; the refusal `open_hire` (6.5). The reasons and causes are exported constants of
the root package (`event.go`).

**Not parameters:** the wage of a hire (derived at formation and saved in the record), the
reservation and the bid (derived each tick), the positions, the committed executions, the free
list of Escrow accounts (derived), the hires (state, 16).

The placeholders, so that none is mistaken for a result: every default and bound above, the
thresholds of the warnings of 14.2, and every number of the worked examples (19) and of the
models (18).

## 14. Validation

Each rule has a sentinel or a warning name and a failing-case test (testing.md 3.9). Errors
name the path, for example `role.<id>.employer.recipes`. The framework of `spec-config.md` 7
runs them.

### 14.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field of 13 outside its bound |
| `ErrUnknownRecipe` | a hire-list entry that is not a recipe |
| `ErrDuplicateID` | a recipe twice in a hire list |
| `ErrEmploymentShape` (new, a root sentinel) | an employer block on a role with an empty `Recipes`; an empty hire list; a hire-list recipe that is not on `Role.Recipes`; a role that is both `Hireable` and an employer |
| `ErrEmployerWeight` (new, a root sentinel) | an employer role with a group of scale above 1, or whose households exceed one per agent (an agent of weight above 1) |
| `ErrModuleDependency` | `Hireable` or an employer block while `Modules.Employment` is absent |

The output of a hired execution is bounded by the derived check of `spec-production.md` 10.3 (`ErrOverflow`), which
with the edit of 12 takes `HireYieldBP` into the largest yield factor, so this spec adds no
overflow error of its own. No overflow error is needed at validation for the bill: a bill that does not fit makes the formation fail
with `employer_funds` (the bill is bounded by `MaxEstablishment`, `MaxPrice` and the longest
`Duration` of the hire lists, `MaxRecipeTicks`, whose product fits through the 128-bit helper),
and a refusal in the round is a counted reason, not an error (5.2).

### 14.2 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at construction, at
`Warn`, with the attributes `rule` and `path`; the volume is bounded by the configuration.

| Warning | Condition |
|---------|-----------|
| `employment_no_employer` | the module is on and no role has an employer block: no hire can form |
| `employment_no_worker` | the module is on and no role is `Hireable` |
| `no_employer_in_region` | a region has a group of a `Hireable` role and no group of an employer role: its households would never be hired there (carries `region`, `role`) |
| `employer_no_outlet` | an employer role has no taste row (the demand module off, or no want in its ladder) and no money flow targets it: the employers' profit has no outlet and hired employment is bounded by their upkeep (11.3.1). The cure list includes a `Saver` block with a `Worth` row for the workshop class (`spec-saving-and-investment.md` 5.3.2, 5.9) |
| `hire_never_profitable` | a recipe on a hire list whose net value per tick at the reference prices with `HireYieldBP` is below 1: the default would never post it |
| `wage_below_upkeep` | the wage of the round at the reference prices, with reservation 0 and the role's `WorkerShareBP`, is below one tick of the upkeep of a `Hireable` role: a hired household cannot reach a ratio of 1 by working every tick (informational) |
| `payroll_exceeds_money` | an employer role's `StartMoney` is below the bill of one full establishment at the reference prices (`MaxHires` times the longest `Duration` of its hire list times the wage above): it could not post its places |
| `inkind_wage_ignored` | a `Hireable` role has `InKind` above 0: the module pays in money only (8.6.1) |
| `worker_share_full` | an employer role has `WorkerShareBP` 10 000: it earns nothing on a hire |

## 15. Observability

### 15.1 Events

Three events, none with a map or a pointer. The fields follow `spec-observability.md` 4.1; the
rows are to be added there (N-126; section 12).

| Event | Emitted | Position | Class | Gate | Fields (besides `Tick`) |
|-------|---------|----------|-------|------|-------------------------|
| `HireFormed` | a hire is formed, in formation order | P2.5a | E | employment | `Hire HireID`, `Region`, `Employer`, `Worker` (agents), `EmployerRole`, `WorkerRole`, `Recipe`, `Households` (the executions), `Wage Money` (per household-tick), `Ticks int64` (the job's `Duration`), `Escrowed Money`, `Reserve Price`, `Bid Price` |
| `HireEnded` | a hire ends: at the completion, at a start that starts nothing, with a removal, a role change or a move | P2.1, P2.7; P7.1, P7.2 or a call, in the order of 6.5.2 | E | employment | `Hire`, `Region` (the employer's, at the event), `Employer`, `Worker`, `Recipe`, `Households` (the executions at the end), `Worked int64` (the working ticks paid), `Paid Money` (the wages paid over the hire), `Refunded Money` (the **remainder** of the escrow at the end only; the refunds of a cut start, 6.1.3, are in `LabourRound.Refunded` and not repeated here), `Cause` |
| `LabourRound` | once per region per tick in which the module is on and any offer, vacancy or hire in progress was counted, in ascending `RegionID`, after `ProductionHalted` | P2.8b | A | employment | `Region`, `Offered`, `Hired`, `NoVacancy`, `NotStarted` (households), `Posted`, `Filled`, `Unfilled` (positions), `OffersRefused`, `VacanciesRefused`, `Matched` (hires formed), refusals of formation `TooMany`, `NotPermitted`, `NoSite`, `ZeroYield`, `NoTool`, `NoFacility`, `NoInput`, `NoEndowment`, `NoTitle`, `RentFunds`, `NotUnlocked`, `FacilityCap`, `NoTarget`, `StoreRefused`, `NothingToRepair`, `NoRoom`, `EmployerFunds` (positions closed by each), `WageBelowUnit` (pairs skipped), `HouseholdTicks` (paid), `Paid Money` (the wages released in the tick, P2.1 and P2.7, to workers whose hire is in the region), `Refunded Money` (the refunds of a cut start made in the tick, 6.1.3, to employers of the region; the remainder of an ended hire is `HireEnded.Refunded`), `Workers []LabourRow` with `LabourRow{Role; Offered, Hired, NoVacancy}`, `Employers []VacancyRow` with `VacancyRow{Role; Posted, Filled, Unfilled}` |

The `Cause` of `HireEnded` is a closed set of constants of the root: `completed`,
`worker_left`, `worker_role`, `worker_moved`, `employer_left`, `employer_role`, `not_started`.
The refusal reasons of `HireFormed`'s failure are the counted fields of `LabourRound`, the
admission reasons of 5.2 are counted in `OffersRefused` and `VacanciesRefused`, and the `Debug`
lines (15.2) carry the reason.

15.1.1. **What an observer can rebuild.** The money of the household sector, by region
(`spec-observability.md` 4.3.3), gains four items in the order of reading: `HireFormed.Escrowed`
(household to contract sector), `LabourRound.Paid` (contract sector to households, the workers,
within the region of the round), `LabourRound.Refunded` (contract sector to the employers of the
region: the refund of a start cut by the ration or the room, 6.1.3, while the hire goes on) and
`HireEnded.Refunded` (the remainder at the end, to the employer (for a removed employer, to its own account first, 6.4.1)). Each refund is in exactly one of the last two. The counts of households by role and region are unchanged: a hire moves
no household.

15.1.2. **Volume.** `HireFormed` and `HireEnded` are per hire (class E, like `ContractFormed`);
`LabourRound` is per region and tick, bounded by the regions and the roles of the configuration,
never per agent. With the module on and an idle labour market none is emitted.

### 15.2 Logs

15.2.1. **`Debug`**, guarded by `Enabled`, constant messages: "offer refused" with `agent`,
`reason`; "vacancy refused" with `agent`, `reason`; "hire refused" with `region`, `reason` (a
closing of 5.5, once per vacancy). The reasons are the constants of 5.2 and 5.5.

15.2.2. **`Warn`**, bounded by the configuration, two lines, pattern B of
`spec-observability.md` 7.3 (the tenth tick of an uninterrupted run and every tenth after;
`WarnRun` is the rule constant):

| Message | Condition | Keys | Cells bounded by | Counterpart |
|---------|-----------|------|------------------|-------------|
| "labour unmatched" | more than half of a worker role's offered households in a region were unmatched in each of the last 10 ticks | `region`, `role`, `households` | regions x roles | `LabourRound.Workers[].NoVacancy`; `RoleRow.Employed` |
| "vacancies unfilled" | positions of an employer role in a region were unfilled in each of the last 10 ticks | `region`, `role`, `vacancies`, `reason` | regions x roles | `LabourRound.Employers[].Unfilled` and the refusal fields |

The `reason` of the second is the commonest cause of the closing in the run: one of the
refusal fields above or `no_labour` (no offer reached the vacancy). The runs are diagnostic state
(16). No `Info`, no `Error`: nothing here is a lifecycle or an inconsistency the library repaired.

15.2.3. **Log keys** (rules 3.3.2), to add in the same change: **one**, `vacancies` (a count of
positions, `int64`), used by the second `Warn` line. `households`, `region`, `role`, `reason` and
`agent` exist, and no line carries a hire ID, a wage or a count hired (none of the lines is about a
formed hire; those values are in the events).

### 15.3 Metrics

Fed by the events above, derivable from them (rules 5.3: a new metric must be). Names and units
follow `spec-observability.md` 6.1; attributes are bounded by the configuration (5.4.1 there).

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.hire.formed` | Counter | `{hire}` | `economysim.role` (the employer's) | `HireFormed` |
| `economysim.hire.wage` | Counter | `{minor_unit}` | `economysim.region` | `LabourRound.Paid` |
| `economysim.labour.unmatched` | Counter | `{household}` | `economysim.role` (the worker's) | `LabourRound.Workers[].NoVacancy` |
| `economysim.vacancy.unfilled` | Counter | `{position}` | `economysim.role` (the employer's), `economysim.reason` | `LabourRound.Employers[].Unfilled` and the refusal fields |
| `economysim.hire.escrow` | UpDownCounter | `{minor_unit}` | none | `HireFormed` (+ `Escrowed`), `LabourRound` (- `Paid`, - `Refunded`), `HireEnded` (- `Refunded`) |

`economysim.reason` gains the closed set of the labour refusals (`employer_funds`, `no_input`,
`no_tool`, `no_facility`, `no_endowment`, `no_title`, `rent_funds`, `not_permitted`, `not_unlocked`, `facility_cap`, `no_target`, `store_refused`, `nothing_to_repair`, `no_room`, `zero_yield`, `no_site`, `too_many_hires`, `no_labour`); the sum of
`economysim.hire.escrow` equals the part of `MoneyContract` that is hires'.

### 15.4 Snapshot

Added to `spec-observability.md` 5.3 (names are OBS's; the form is proposed):

| Slice or field | Row | Meaning | Cost |
|----------------|-----|---------|------|
| `RoleRow.Employed`, `RoleRow.Employing` | `int64` | households of the role on a hired job at the boundary; hired households working for agents of the role | S |
| `Hires []HireRow` | by hire ID | `Hire`, `Region`, `Employer`, `Worker`, `Recipe`, `Households`, `Wage`, `Formed`, `Ends` (the completion tick), `TicksLeft`, `Account` (the Escrow account), `Held Money` (its balance) | F |

The last tick's counts (offered, hired, unmatched) are the event's, not snapshot fields
(`spec-observability.md` 5.2.2). A hire is a function of saved state (16).

### 15.5 Cost with no observer

With the default logger and observer the round, the keys and the release allocate nothing beyond
scratch held by the world: the admitted lists, the lanes and the committed executions are
scratch, the events are constructed only when an observer was supplied
(observability-and-logging.md 4.5) and the logging is guarded (NFR-EMP-001). The cost of a tick
is proportional to the admitted offers and vacancies and to the hires in progress, not to the
agents: with `O` admitted offers, `V` vacancies and `H` open hires, the sort is `O log O + V log V`
and a tick's payments are `H`.

## 16. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, all plain data in a fixed
order with no maps, a new section of `spec-checkpoint.md` 4.2 (gate `Modules.Employment`). **Its
position** among the 23 sections is after Merchants (21) and before Integration (22), because a
hire names an Escrow account and agents that earlier sections hold (4.2.1 puts sections 12 to 22
in the order of the dependencies between modules): it takes tag 22 and Integration and Extensions
become 23 and 24, the numbering being CHK's. **A section added and a renumbering change the format
version** (`spec-checkpoint.md` 9.3), which the edit asks CHK to bump; the golden file of a world
with the module off does not change in content (the section is absent, 4.2.1) but its version
does:

- the **next hire ID**;
- the **hires in flight**, in ascending hire ID: the ID, the employer, the worker (agents), the
  wage per household-tick, the Escrow account, the formation tick, the region (by index); the
  recipe, the executions and the ticks left are **the worker's job**, saved once in the agent
  record by PRO (`spec-production.md` 15; `Principal` there names the employer), so the job
  record is not repeated (N-128);
- for each (region, role), in region then role order: the two runs for the `Warn` lines of
  15.2.2 (diagnostic: they affect no result and are saved so that a resumed run logs as the
  original did), sparse.

**Not saved (derived):** the free list of Escrow accounts, the committed executions, the
reservation and the bid, the positions, the committed scratch, the wage of a hire that has not
been formed, the effective factor, `Employed` and `Employing` (functions of the hires and the
jobs). A restore builds the free list from the ledger, the contract list and the hire list and
runs the audit of 7.4.2 (`spec-checkpoint.md` 6.2.2); a world restored between ticks forms the
same hires and pays the same wages as an uninterrupted one (FR-EMP-043).

## 17. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9. Provisional IDs (N-103): the
supervisor assigns the `STB-Lnn`, `Dnn` and `Snn`. Each entry has the fields of its section 3.
**Status is `hand` unless it says `model`:** the models of section 18 are stylised, and nothing
here has been run in the library. No default is claimed to be stable (N-027). The hook row "EMP"
of `spec-stability.md` 4.8 is answered by these entries and can be removed.

**Dampers this spec specifies** (for the catalogue of `spec-stability.md` 5):

| ID | Damper | Home | Status |
|----|--------|------|--------|
| EMP-D1 | No bidding up: the wage is a share of the employer's own margin and the worker's own alternative, and does not depend on the count of offers or of vacancies, so there is no wage spiral by bidding. Cost: wages do not clear the labour market, so unemployment can persist at a wage | 5.4.3 (`WorkerShareBP`) | specified (a rule constant; model `labour.py`, `circuit.py`) |
| EMP-D2 | The margin test and the capacity: a vacancy is posted only for a recipe with a positive net value that is not halted, at most `MaxHires`, and at most what stock, tools and cash support. Cost: the capacity binds before the demand does, so the demand for labour is a step | 4.3.3 (`MaxHires`) | specified (counted by `Posted`, `Unfilled`) |
| EMP-D3 | The escrow at formation: a hire exists only if its wages are in escrow, the inputs and tools are held and the site yields, so neither side defaults on a hire that exists. Cost: money out of circulation for the length of the job | 5.5, 7 | specified (counted by `EmployerFunds`; audit) |
| EMP-D4 | One job, no standing contract: a lay-off is a vacancy not posted, so no mass dismissal exists and a worker's income does not stop at a stroke. Cost: no security and no retained staff | 3.4 | specified (a rule constant) |
| EMP-D5 | The worker share as a visible lever: the share of the margin that the circuit needs to be spent is `1 - phi`, and the warnings say when it cannot be | 3.2, 11.3.2 (`WorkerShareBP`) | specified (a placeholder default) |
| EMP-D6 | The pooled wage excludes the employer roles: the cost anchor, and through it the cost floor of the sellers' beliefs, do not rise with the profit an employer makes on hired households. Cost: the anchor says nothing of an employer's own labour | 8.4 | specified (a rule constant) |
| EMP-D7 | `MaxOpen` per region: a cap on open hires that is not first-come across regions. Cost: some vacancies are closed (`TooMany`) | 5.5 (`MaxOpen`) | specified (counted by `TooMany`) |

**EMP-A1. The wage earner's labour loop: the hired case of L14.** *Kind:* loop (a case of
STB-L14). *Loop:* wage `W = R + phi (Bid - R)` (+) income per working tick (-) households offered
(the target is reached sooner) (+) households hired (more offered, more hired) (-) shortage of the
good (more hired, less shortage) (+) price (+) the bid `a p - c` (+) wage. Two negative links, an
even number: reinforcing. *Delay:* the welfare window and the gate,
and the one-tick delay of P4.1 (6.1.8 of `spec-world.md`); the wage itself reaches the worker
before P3.2, so no tick is added there. *Owner:* EMP 4.3.1, 8.2.1; WEL 5. **Two regimes**: when
the places are fewer than the households that offer (**vacancy-bound**), output is fixed at `a`
times the places, the labour channel is inert, and the price moves only through demand (L05, L06,
L51); when the households that offer are fewer than the places (**labour-bound**), the loop is
L14's. The elasticity of the days worked to the price depends on what the worker eats: a worker
that eats the good it makes has `D` proportional to `p / (a p - c)`, whose elasticity is
`-c / (a p - c)`, small when inputs are a small share of the value (wage and cost rise together),
and a worker that eats another good has `-a p / (a p - c)`, near unit-elastic for small `c`.
*Damper:* D13 (the floor counts a hired tick), D09 and D28 (the band and the proportional offer
of a cohort), D26 (not a damper), EMP-D2. *Undamped:* the interior equilibrium, as L14. *Condition:*
the demand for the good is elastic above 1 (L14) or the market is vacancy-bound. *Test:* S04 with
a hired labour role (EMP-S2). *Status:* hand, with `labour.py` for the regimes.

**EMP-A2. The hire cobweb.** *Kind:* loop (a case of STB-L21, L22 and AGT-A9). *Loop:* price (+)
the bid (+) positions posted (a step at a bid of 1, then the capacity, 4.3.3) (+) inputs bought a
tick earlier (+) output after `Duration` (-) price. Balancing with a delay of `Duration + 1`. The
demand for labour is a **step** in the price and then a cap, so the only elasticity of production
to price that the module adds is at the margin, like recipe choice (L48): `labour.py` shows
(`labour.out`) hired households per tick over the last 100 ticks of a glut running between 0
and all 120 places, with a mean of 59.3 of 120 (60.2 with the halt), and the price chattering
(200 to 205) at the employers' zero-margin price `c / a`. The hires of one employer in one tick start and end together
(a lockstep, L22), and an employer buys the inputs of the establishment it expects (9.2), a second
source of the buffer pulse (AGT-A9). *Damper:* D24 (the halt on stock), EMP-D2, D04 (the belief
bounds), D16 (spoilage). *Undamped:* the step. *Condition:* a price-responsive flow somewhere else
(a luxury's schedule, D30; exports; a staple has none: L51). *Test:* EMP-S4, S16. *Status:* hand,
`labour.py` (model).

**EMP-A3. Profit has no outlet: the circuit.** *Kind:* constraint (an identity) and hazard.
*Loop:* none; the identity of 11.1: what the employers keep is what they must spend for the
output to sell. With employers who spend only their upkeep, the hired households settle at
`E u_e / ((1 - phi) a)` (11.2), the rest of the places unfilled and the households unmatched, and the
money piles with the employers (L31, L45) while the workers are broke. *Owner:* EMP 11 (SAV 8.3 lists the outlets). A
companion of L57 (the demand side of the wage and demand loop), L46 and N-054 (a closed economy
with upkeep only). *Damper:* D30 and D31 (the employers' wants, funded from what the cap leaves),
D27 (the cap, which bounds the outlet at `SpendMaxShare * SpendHalfTicks` ticks of upkeep), D42
(a holding charge), `WorkerShareBP` (EMP-D5), exports and grants (MON, EXT). *Undamped:* an
employer role with upkeep only, and the cap's asymptote. *Condition:* `h*` of 11.2 is at least the
places. *Test:* EMP-S1 (the closed form). *Status:* model (`circuit.py`, 16 cells, the first column
exact).

**EMP-A4. The payroll's drain on liquidity.** *Kind:* loop (a sibling of MER-A5). *Loop:* hires
formed (+) money in escrow (-) the employers' cash (+) the positions the cash supports on the
next tick (+) hires. One negative link, an odd number: **balancing** while cash is the binding
limit, and bounded by `MaxHires`; the cash returns as sales (the wages are spent) after
`Duration` plus one tick. *Delay:*
`Duration + 1`. *Owner:* EMP 5.5, 7. *Damper:* EMP-D3 (the check), the positions by cash (4.3.3),
`MaxOpen` (EMP-D7), the warning `payroll_exceeds_money`. *Undamped:* an employer with a long job
and thin cash: the whole bill of the job is in escrow from its start; credit (CRD) is the remedy.
*Condition:* the employers' cash covers `MaxHires * Duration * wage`. *Test:* EMP-S1 with a
three-tick recipe. *Status:* hand.

**EMP-A5. The wage follows the margin.** *Kind:* bias and hazard. *Loop:* a glut (+) the price (-)
the bid (-) the wage (-) the workers' ratio (+) privation (+) the ladder, relief and exit; in a
shortage the wage rises with the price and not with the scarcity of workers. The wage is `R + phi
(Bid - R)`, so in a glut that destroys the margin it falls to the reservation, which for a
household with no alternative is 0 (`labour.py`: at the zero-margin price, 200 to 205 in the run, the bid is 0 to 30 and the
wage 0 to 15). *Owner:* EMP 5.4. *Damper:* D11 (relief), D15 (the floor: the own-use mode),
D35 (the ladder, graded), D36 (the entry gate), the reservation of a role with its own recipes
(4.3.2). *Undamped:* a role with no recipe and so no alternative: its wage is whatever the
margin leaves. *Condition:* the margin is positive at the reference prices (`hire_never_profitable`,
`wage_below_upkeep`). *Test:* EMP-S1, S4. *Status:* hand, `labour.py`.

**EMP-A6. The employer's occupation signal.** *Kind:* loop (a case of STB-L36). *Loop:* an employer
role's observed wage (its profit per own working tick, 8.3.2) (+) the expected ratio `E` (SPN 6.1)
(+) entrants to the role (+) employers (-) hired households per employer (-) profit per own tick (-) `E`.
Balancing with the window's delay, and a cobweb through the delay of entry. The signal is
inflated by the hired households' work and by the capital an employer holds (a tool), so the role attracts
households that lack the capital: the barrier is `ToolPrice` and `Closed`. *Delay:* the welfare
window and the review period. *Owner:* EMP 8.3.2; SPN 6. *Damper:* D14 (entry share), D33, D34,
D35, D36 (the viability gate), `Closed`, and the tool as a barrier (`spec-skill-barriers.md` 2).
*Undamped:* a role with no tool and no `Closed`. *Condition:* `EntryShare * T` below about 2 000
(L36). *Test:* S06, S26 with an employer role. *Status:* hand.

**Checklist items (spec-stability.md 9.6 and 9.7).** Delays that match a neighbour: the longest
hired `Duration` against the employer's window (8.5.2, both by the recipe's length); the inputs
bought a tick before against `CoverTicks` 3 and `BeliefMaxAge` 3 (a hire's inputs are bought on
the price the market had a tick ago, 9.4); the wage released at P2 against the sales at P4 (no
neighbour, a tick). Every damper parameter named here has a row in the parameter-owner table of
the handoff.

**New scenarios proposed to the register.**

| ID | Pack and perturbation | Asserts |
|----|-----------------------|---------|
| EMP-S1 | `one` with a hire list on an employer role and a `Hireable` role without recipes; run with the employers' wants off, then on | hired households settle at `h*` of 11.2 (within the cap's concavity) with the wants off; with them on the unfilled places fall, and the audit holds every tick |
| EMP-S2 | a labour-bound world (more places than offers) with the target-income workers, the floor at its default and at 0 | the price stays within its bounds and the role ends in a stated condition, the measured elasticity of the days worked recorded (EMP-A1) |
| EMP-S3 | an employer removed in the middle of a three-tick job, and a worker removed in the middle of another | the escrow returns to the employer's own account (and reaches the estate with the rest of its money) and to the employer respectively, every wage paid stays, conservation holds, the audit passes |
| EMP-S4 | a step in the price of the output (a shock to the yield of the employers' input) | positions fall and recover, `NoVacancy` rises and falls, the recovery time recorded (EMP-A2) |
| EMP-S5 | the invariants over a range of seeds with `WithAudit`; a module on takes no word, with or without offers | determinism, conservation and the escrow identity hold; the stream of draws equals the module off |

## 18. What was run

Four small Python scripts in `doc/specs/models/emp/`, run only in Docker with
`PYTHONDONTWRITEBYTECODE=1`, standard library only (README there). They are prototypes of the
rules, not of the library, and the numbers they print are placeholders and models only (N-027).

| Script | What it shows | What it does not show |
|--------|---------------|-----------------------|
| `round.py` | the round of 5 and 6 over 500 random scenarios (1 to 4 employers, 1 to 12 workers, reservations 0 to 100, shares 3 000 to 10 000, jobs of 1 to 5 ticks, a cap of open hires, random removals of a worker or an employer in mid-job): after every step the money of all accounts is constant, no balance is negative, the escrow holds `wage * k * ticks left`, the wage lies in `[Reserve, Bid]`, and every escrow is empty at the end (1 048 hires formed in the run, with refusals counted by reason; ties by the stateless key `K(t, id)`, no draw) | the ration and the refund of a cut start, the welfare record, the events |
| `circuit.py` | the identity and the closed form of 11 over 16 combinations of the worker share and the cap: with employers who eat only their upkeep the hired households settle at `E u_e / ((1 - phi) a)` (7.1, 10.0, 19.8 and 50.0 hired of 100 places at `phi` 3 000, 5 000, 7 500 and 9 000), and with the cap of AGT 7.4.4 just under the bound | a price (it is fixed), several goods, a want, the beliefs |
| `labour.py` | the price channel of 17 (EMP-A1, EMP-A2) over ten seeds: with an inelastic staple the price goes to the employers' zero-margin price `c / a` (200 of 1 000) in a glut, with the hired households per tick running between 0 and all 120 places over the last 100 ticks (the output prints the minimum and the maximum, not only the mean) and the wage with it, and to the ceiling in a shortage, where employment stays at the places (80); with cheap inputs the price falls to the bound | the stability of any default; it uses a caricature of the price rule and a fixed demand |
| `worked.py` | the arithmetic of the two examples of 19 and the closed form again | |

**What the models changed in the design.** The first form of the wage rule had a `BidShareBP`
(the share of the margin an employer bids) and a worker's reservation of 0. `circuit.py` showed
that with the employers' profit spent only on upkeep the wage bill equals what the employers
spend, so the workers' ratio is bounded by the employers' appetite and not by their capacity
(11.2); the lever that matters is therefore the *worker share of the margin*, `WorkerShareBP`,
which is what 3.2 now has, with the midpoint as its default.

## 19. Worked examples

Placeholders, chosen to be easy to follow, produced by `models/emp/worked.py`. The settings are
examples and flagged as such.

### 19.1 Example A: a river valley, a mill that hires hands (a pre-industrial setting)

Roles: `miller` (an employer block: hire list `[mill_grain]`, `MaxHires` 4, `WorkerShareBP` 5 000,
`HireYieldBP` 10 000; `Recipes` `[mill_grain]`), `hand` (`Hireable`, no recipes, upkeep 2 `grain`
a tick). Recipe `mill_grain`: `Duration` 1, input `grain` 3, output `flour` 2 (cost share 10 000).
The miller's beliefs: `flour` 700, `grain` 150. Its money 6 000, `grain` 12, upkeep 2 `bread` a
tick at 150 (cost 300; `bread` is not an input of the recipe, so the keep-the-food rule of
`spec-agents.md` 7.1.6 holds back no grain).

| Step | Value |
|------|-------|
| Value of an execution | `2 * 700 = 1 400` |
| Cost of its input | `3 * 150 = 450`; net `950`, per tick of a hired household (Duration 1) `950`: the bid |
| Reservation of a hand (no recipe) | 0 |
| Wage | `0 + halfEven(5 000 * 950 / 10 000) = 475` |
| The miller's own request | 1 execution (`k_own` 1), 3 grain |
| Positions | by stock `12 / 3 - 1 = 3`; by cash `(6 000 - 300) / 950 = 6`; `MaxHires` 4: **3** |
| Five hands offer one household each | offers ordered by reservation (all 0) then key; vacancy `Left` 3 |
| Round | three hires formed, two hands unmatched (`NoVacancy` 2) |
| Escrow | `3 * 475 = 1 425` moved at P2.5a; the miller's cash 4 575 |
| P2.7 | inputs `9` grain taken from the miller, three jobs of duration 1 start and complete: `6` flour to the miller; `1 425` wage paid; refund 0 |
| A hand's ratio on a working tick | `475 / 300 = 15 833` bp, above its target of 15 000: it reaches the target in `ceil(15 000 * 300 * 365 / (10 000 * 475)) = 346` of 365 ticks |

The miller's margin is `475` per hired execution and `1 425` for the three. In the circuit of 11
this is the amount that must be spent by the employers (the miller's wants) for the flour to
sell and the hands' wages to be spent back. The closed form of 11.2 is for **one** good that the
employers also eat and is **not** applied to this example: here the hands eat grain, the miller
eats bread and the mill makes flour, so no instance of `h*` is claimed.

### 19.2 Example B: a mining colony, a rig operator that hires crew (a space setting)

Roles: `operator` (hire list `[drill_ore]`, `MaxHires` 3, `HireYieldBP` 8 000; money 5 000, upkeep cost
300, three `rig` and three `charge` held, no execution of its own), `crew` (`Hireable`, recipes
`[gather_scrap]`). Recipe `drill_ore`: `Duration` 3, input `charge` 1, requires `rig`
(`WearPerUse` 1, `Durability` 500), output `ore` 5. Beliefs: `ore` 400, `charge` 120, the rig's
price 20 000. The crew's own recipe `gather_scrap` nets 180 per execution of 3 ticks.

| Step | Value |
|------|-------|
| Hired value of an execution | `floor(5 * 8 000 * 400 / 10 000) = 1 600` |
| Input and wear | charge 120; `ceil(1 * 20 000 / 500) = 40` |
| Net per execution, per tick of labour | `1 600 - 160 = 1 440`; `1 440 / 3 = 480`: the bid |
| Reservation of a crew household | its own alternative: `180 / 3 = 60` |
| Wage | `60 + halfEven(5 000 * (480 - 60) / 10 000) = 270`; a household with no recipe (reservation 0) would get 240 |
| Positions (4.3.3) | `min(MaxHires 3, by_stock 3, by_tools 3, by_cash floor((5 000 - 300) / (480 * 3)) = 3) = 3`; with money 3 000 `by_cash` would be 1 (`floor(2 700 / 1 440)`) and the cohort of 3 would never be matched |
| Bill of one household for the job; of a cohort of 3 | `270 * 3 = 810`; `810 * 3 = 2 430`, which the operator's money 5 000 covers (three positions, three rigs and three charges fit a block of 3, 5.5 checks 3, 4 and 6) |
| Margin per execution | `(480 - 270) * 3 = 630` |
| Pay | 270 at the start tick (P2.7), 270 at P2.1 of `s + 1`, 270 at P2.1 of `s + 2`; completion at `s + 2` refunds 0 |
| The offer of the cohort of 3 (one block, matched whole) whose weight falls to 2 after two ticks | paid `270 * 3 * 2 + 270 * 2 * 1 = 2 160` of an escrow of `2 430`; refund `270` at the end |

The worked `HireEnded` of the last row has `Households` 2, `Worked` 3, `Paid` 2 160 and
`Refunded` 270 with the cause `completed`.

## 20. Notes settled

| Note | Decision | Where |
|------|----------|-------|
| N-046 Register entries owed by specs that were not yet written | **Applied for EMP**: EMP-A1 to A6 and the dampers EMP-D1 to D7 and scenarios EMP-S1 to S5 are in 17; EMP leaves the areas | 17 |
| N-028 The target-income rule and a shortage | **Applied in part**: the hired case is EMP-A1; the two regimes and the elasticity of the days worked are stated; the note stays open for WEL and STB until S04 has run | 8.2, 17 |
| N-054 The closed economy fixes the mean ratio at 1 | **Closed note; companion stated**: with employers the identity holds in the aggregate and the split by role is EMP-A3 (11.4.3) | 11 |
| N-014 Stability register entries to collect | **Closed note; consistent** | 17 |
| N-016 Keep the parameter registry true | **Applied**: rows for the module, `MaxOpen`, `Hireable`, the employer block and its four fields; no unnamed parameter | 13 |
| N-017 Words and names | **Applied**: terms fixed in 23.2; setting words only in 19 | 23.2 |
| N-027 No default macro constant is stable | **Applied**: every number is a placeholder (13, 18, 19) | 13 |
| N-103 Running several writers | **Applied**: the seams were read against the finished specs; two slips of the host interpreter are in the handoff | |
| N-126 A new event or `Warn` adds a row | **Applied**: three events and two `Warn`s, rows given in 15 and the handoff | 15 |
| N-128 A hosted per-agent field is saved once | **Applied**: this spec adds no per-agent field; the worker's job is PRO's record (16) | 16 |
| N-078 What the demand module needs from the money and income specs | **Applied for EMP**: a class with no recipes (a `Hireable` role) gets its income from wages; the spending side is 11 | 3.3, 11 |
| N-090 What MIG and later modules must do with a relocation | **Applied**: the refusal `open_hire`; the worker's job is abandoned by the move | 6.5 |
| N-107 / N-021 New faucets and sinks | **Applied**: none; three internal reasons | 7 |
| N-122 First-come caps | **Applied**: `MaxOpen` is per region and the order is by value then the stateless key `K(t, id)` | 5.3 |
| N-130 What IND cannot measure | **Applied**: the wage bill is measurable from `LabourRound.Paid`; the income fields stay an ask | 10.2 |
| N-099 A trip is a gap in income | **Applied by contrast**: a hired job is paid per tick and is no gap | 8.5 |
| N-124 A paired comparison is exact only while aligned | **Stated**: the module takes no word at any time (5.3.1), so a parameter that changes the positions posted (`MaxHires`, `WorkerShareBP` through the cash check) moves no later word of the run and is not the flag `d` of CLB; the paired comparison stays aligned | 5.3.1 |

## 21. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Every requirement is
derived from text of sections 1 to 17.

### Functional

| ID | Requirement |
|----|-------------|
| FR-EMP-001 | The module MUST run if and only if `Config.Modules.Employment` is non-nil, and a world with it nil MUST have no position P2.5a or P2.8b, take no word for it and emit none of its events. |
| FR-EMP-002 | A role MUST have the fields `Hireable` (default false) and `Employer` (default nil), and a role that is both MUST be rejected with `ErrEmploymentShape`. |
| FR-EMP-003 | An employer block MUST have `MaxHires` 1 to `MaxEstablishment` (default 4), `WorkerShareBP` 1 to 10 000 (default 5 000) and `HireYieldBP` 1 to `MaxYieldBP` (default 10 000), and a value outside its bound MUST be rejected with `ErrOutOfBounds`. |
| FR-EMP-004 | The hire list MUST be non-empty, every entry MUST be a known recipe of the role's own `Recipes` and none MUST be repeated, rejected with `ErrEmploymentShape`, `ErrUnknownRecipe` and `ErrDuplicateID`. |
| FR-EMP-005 | An employer role MUST have only groups of scale 1 and no more households than agents, rejected with `ErrEmployerWeight`. |
| FR-EMP-006 | `Hireable` or an employer block MUST be rejected with `ErrModuleDependency` while `Modules.Employment` is absent. |
| FR-EMP-007 | No configuration field MUST hold a wage or a wage multiplier, and the wage of a hire MUST be derived at its formation. |
| FR-EMP-008 | A hire MUST be recorded with an ID, the worker, the employer, the wage, the Escrow account and the formation tick, and the job MUST be the worker's job with the employer as its `Principal`; no commodity, input or order MUST be labour. |
| FR-EMP-009 | A `JobRequest` with the zero value of `Offer` and `Hire` MUST be handled exactly as before this spec, for every strategy. |
| FR-EMP-010 | An offer MUST be admitted or refused whole, in the order `employment_off`, `not_hireable`, `in_traineeship`, `in_own_use`, `invalid_households`, `invalid_price`, with the first failing reason counted in `OffersRefused` when the module is on and logged at `Debug` only (not counted) for `employment_off`. |
| FR-EMP-011 | A vacancy MUST be admitted or refused whole, in the order `employment_off`, `not_employer`, `not_on_hire_list`, `invalid_positions`, `invalid_price`, `employer_full`, with the first failing reason counted in `VacanciesRefused` when the module is on and logged at `Debug` only for `employment_off`, where positions above `MaxHires` less the hired executions open MUST be cut and not refused. |
| FR-EMP-012 | The own request of a worker whose offer is matched MUST be withdrawn before the ration of P2.6, and the own request of a worker whose offer is not matched MUST stand. |
| FR-EMP-013 | The default strategy MUST offer `ceil(w * s / 10 000)` households when its work share `s` is above 0 and the role is `Hireable` and the agent is neither in the own-use mode nor a trainee, and nothing otherwise. |
| FR-EMP-014 | The reservation of the default's offer MUST be `max(0, floor(net_r / Duration_r))` of its best executable recipe by the rule of `spec-agents.md` 7.1.4 and 7.1.5, and 0 for a household with none. |
| FR-EMP-015 | The default strategy MUST post for the eligible hire-list recipe of the greatest `net_r / Duration_r` a bid `floor(net_r / Duration_r)` valued at the hired yield and positions `min(MaxHires - Hired, by_stock, by_tools, by_cash)`, and MUST post nothing when the bid is below 1 or the positions are 0. |
| FR-EMP-016 | The positions an employer posts MUST NOT be scaled by its work share, and its own labour MUST follow the work share of `spec-agents.md` 7.1.2 as before. |
| FR-EMP-017 | The labour round MUST run after the calls of `Work` for every free agent and before the ration of scarce endowments, as position P2.5a. |
| FR-EMP-018 | The round MUST take no word of the world's source, and MUST break a tie of equal reservations or equal bids by the stateless key `K(t, id)` of `spec-world.md` 10.1.1, smaller first, then by agent ID. |
| FR-EMP-019 | The lanes MUST be the regions in ascending `RegionID`; in a lane the offers MUST be ordered by reservation ascending then `K(t, id)` then agent ID, and the vacancies by bid descending then `K(t, id)` then agent ID. |
| FR-EMP-020 | An offer MUST be matched to the first open vacancy of its lane in order whose bid is at least its reservation and whose remaining positions are at least its households, and its households MUST be counted unmatched when there is none. |
| FR-EMP-021 | The wage of a pair MUST be `Reserve + halfEven(WorkerShareBP * (Bid - Reserve) / 10 000)`, MUST lie in `[Reserve, Bid]`, and a pair whose wage is below 1 MUST be skipped. |
| FR-EMP-022 | The wage MUST NOT depend on the number of offers or of vacancies. |
| FR-EMP-023 | The formation checks MUST run in the order `too_many_hires`, `not_permitted`, `zero_yield` or `no_site`, `no_tool`, `no_facility`, `no_input`, `no_endowment`, `no_title`, `rent_funds`, `not_unlocked`, `facility_cap`, `no_target`, `store_refused`, `nothing_to_repair`, `no_room` (each with its module on), `employer_funds`, the order of the reasons being `spec-production.md` 4.2's; `too_many_hires` MUST close the lane, `not_permitted`, `zero_yield`, `no_site` and the build reasons other than `facility_cap` MUST close the vacancy, and each of the others MUST lower the vacancy's fit to one household less than the block that failed, closing it at 0. |
| FR-EMP-024 | A formation MUST, atomically, move the bill `wage * Duration * k` from the employer to the lowest-ID open Escrow account of the region that no contract or hire uses (opening one if none) under `wage_escrow`, create the record, withdraw the own request and add the hired request. |
| FR-EMP-025 | The executions of an employer's own request, its trainees' requests and the hires formed so far in the round MUST be counted against its stock and tools at each formation. |
| FR-EMP-026 | A hired request MUST join the ration of P2.6; a start with `k` below the formed executions MUST refund `wage * Duration * (formed - k)` at once under `wage_refund`, and a start with `k` 0 MUST end the hire with the cause `not_started` and refund the whole escrow. |
| FR-EMP-027 | A hired job MUST take its inputs and wear its tools from the principal's account and create its output in the principal's account, with the employer's tool, `wear` as for any wear. |
| FR-EMP-028 | The effective factor of a hired job MUST be the site factor times the shock times `tech` times `HireYieldBP` times the abundance, with the `trainee` factor not applied. |
| FR-EMP-029 | A hired job MUST release `wage * Executions` from its escrow to the worker under `wage_pay` once per working tick, at P2.7 for the start tick and at P2.1 for each later tick, and never more than the escrow holds. |
| FR-EMP-030 | At the completion of a hired job the output MUST be created in the principal's account, the last tick's wage paid, the remainder of the escrow returned to the employer under `wage_refund`, the Escrow account released and the hire ended with the cause `completed`. |
| FR-EMP-031 | When a worker is removed its job MUST be abandoned with `agent_removed`, the remainder of the escrow returned to the employer and the hire ended with `worker_left`. |
| FR-EMP-032 | When a worker changes role its job MUST be abandoned with `role_changed` and the hire ended with `worker_role`, and when its account changes region its job MUST be abandoned with `region_changed` and the hire ended with `worker_moved`, the remainder returned to the employer in both. |
| FR-EMP-033 | When a worker's weight falls its job MUST NOT be abandoned, its executions MUST be scaled by `spec-production.md` 5.4, and the escrow not paid MUST be returned to the employer at the end. |
| FR-EMP-034 | When an employer is removed each hire's job MUST be abandoned with `principal_left`, the remainder of its escrow moved to the employer's own account under `wage_refund` at step 2c of `spec-agents.md` 3.6 (before the creditors' waterfall), and the hire ended with `employer_left`, hires in ascending hire ID. |
| FR-EMP-035 | When an employer changes role each hire's job MUST be abandoned with `principal_left`, the remainder returned to the employer and the hire ended with `employer_role`. |
| FR-EMP-036 | A worker MUST keep every wage paid before an interruption, and the wages paid plus the amounts refunded of a hire MUST equal the bill moved into escrow at its formation, so that no money is created or destroyed. |
| FR-EMP-037 | While an agent is the principal of an open hire the relocation of `spec-regions.md` 6 MUST refuse it with `ErrRegionBound` naming `open_hire`, with the hire IDs as its instance keys. |
| FR-EMP-038 | The events of one removal or role change MUST be in the order of 6.5.2: trainees, then the employer's hires in ascending hire ID (`RecipeAbandoned`, `HireEnded`), then the agent's own list. |
| FR-EMP-039 | `wage_escrow`, `wage_pay` and `wage_refund` MUST be internal reasons between non-boundary accounts, no flow of the module MUST appear in `BoundaryFlow`, and the sectoral identity of `spec-ledger.md` 9.3 MUST hold with the hire escrow in the contract sector. |
| FR-EMP-040 | An Escrow account MUST hold the money of one open contract or hire only, MUST be of the employer's region and MUST be reused and never closed, with the free list derived. |
| FR-EMP-041 | The audit MUST fail with `ErrAuditFailed` when an open hire's escrow is below `wage * Executions * ticks left`, is shared, or when the worker's job does not name the employer as principal. |
| FR-EMP-042 | A formation whose bill exceeds the employer's available money MUST fail with `employer_funds` and move nothing, and the module MUST create no claim and no credit. |
| FR-EMP-043 | A world restored between ticks MUST form the same hires, pay the same wages, take the same words and emit the same events as an uninterrupted run. |
| FR-EMP-044 | The checkpoint MUST save the next hire ID and the hires in flight in ascending ID (employer, worker, wage, Escrow account, formation tick, region) and the diagnostic runs, and MUST NOT save the free list, the committed executions or any derived quantity. |
| FR-EMP-045 | `income_t` MUST include the wages received `Y_t` and net the wages paid `V_t` in the tick the working tick is taken, and the escrow made at formation MUST NOT be netted. |
| FR-EMP-046 | A hired tick MUST be a working tick of the worker with `k` working household-ticks, and a tick in which an agent is the principal of a hired job in progress MUST be a working tick for the floor of the principal and MUST NOT add to its working household-ticks. |
| FR-EMP-047 | The pooled regional wage MUST be taken over the agents of the roles that have no employer block, and the rule MUST add no state. |
| FR-EMP-048 | A hired job MUST NOT lengthen the effective window of a worker's role. |
| FR-EMP-049 | The wear cost of a hired execution MUST be the principal's. |
| FR-EMP-050 | `InKind` of a `Hireable` role MUST NOT change the wage, and the warning `inkind_wage_ignored` MUST be raised. |
| FR-EMP-051 | `HireFormed` MUST be emitted at P2.5a per hire in formation order with the fields of 15.1. |
| FR-EMP-052 | `HireEnded` MUST be emitted at the end of every hire with its `Cause` from the closed set and the money paid and refunded. |
| FR-EMP-053 | `LabourRound` MUST be emitted once per region per tick in which any offer, vacancy or hire was counted, in ascending `RegionID`, after `ProductionHalted`, and MUST carry no map. |
| FR-EMP-054 | The money of the household sector rebuilt from the events MUST equal the snapshot with the four items of 15.1.1. |
| FR-EMP-055 | The snapshot MUST carry `Hires` and the `Employed` and `Employing` of each role row, each a function of saved state. |
| FR-EMP-056 | The two `Warn` lines of 15.2.2 MUST be emitted per region and role, never per agent, on the tenth tick of an uninterrupted run and every tenth after. |
| FR-EMP-057 | The five metrics of 15.3 MUST be derivable from the events and their attributes MUST be bounded by the configuration. |
| FR-EMP-058 | Offers and vacancies refused and formations closed MUST be logged at `Debug` only, guarded by `Enabled`, with constant messages and a `reason`. |
| FR-EMP-059 | The warnings of 14.2 MUST be raised once at construction and MUST NOT fail validation. |
| FR-EMP-060 | The households offered and not matched MUST be counted per region and role as `NoVacancy`, and a household that rested, was in the own-use mode or was a trainee MUST NOT be counted. |
| FR-EMP-061 | With the module on, whether or not anything is admitted, the module MUST take no word, so the stream MUST equal the module off, and a run with a logger and observer MUST give results identical to a run without. |
| FR-EMP-062 | In a closed economy whose employers spend only their upkeep and with the halt on, the mean hired households SHOULD settle at `E * u_e / ((1 - phi) * a)` within the burst of the halt. |
| FR-EMP-063 | `MaxOpen` MUST bound the open hires of a region and not of the world, and a vacancy MUST NOT be closed by the open hires of another region. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-EMP-001 | With the default logger and observer, a tick with the module on MUST allocate nothing beyond scratch held by the world for admission, the keys, the matching, the formation and the release. |
| NFR-EMP-002 | The cost of the round SHOULD be `O(O log O + V log V)` in the admitted offers and vacancies, and the cost of a tick's pay `O(H)` in the open hires, not in agents or households. |
| NFR-EMP-003 | The pure functions of `internal/employment` (the wage, the bill, the order, the positions) MUST have fuzz tests asserting that no input within the bounds panics or overflows and that the wage lies in `[Reserve, Bid]`. |
| NFR-EMP-004 | The round SHOULD have a benchmark that calls `b.ReportAllocs()`. |
| NFR-EMP-005 | The conservation, determinism and escrow identity tests of EMP-S5 MUST run over a range of seeds with `WithAudit`, reporting the failing seed. |

## 22. Open questions

Each with a next step.

1. **Pay in kind.** WEL 6 describes `InKind` and no mechanism but the trainee's board. A wage in
   goods needs a valuation (the published price), goods that the employer holds beyond its own
   use, and a settlement that cannot fail (goods are not escrowed): the rule that would work
   delivers the share in the worker's upkeep goods from the employer's unreserved stock after its
   inputs, falls back to cash from the escrow, and refunds the cash saved. Not adopted for
   version 1 (the `wage_in_kind` reason is reserved). *Next step:* WEL and EMP, when a pack needs
   a goods wage (the first consumer's agricultural labour is the likely case, an example only).
2. **A cohort employer.** The establishment, the places and the own labour would scale with the
   weight and a hire would be a partial admission. *Next step:* when a consumer scales an employer
   role (3.6.1).
3. **Standing employment.** A service contract with a term, a notice and a bond is a longer job
   with a retained worker; the record of 3.4.1 would carry it. *Next step:* ORG or CRD, with
   the claim of `spec-ledger.md` 10.
4. **A busy employer posts nothing** (4.1.3). Posting from `Trade`, or calling `Work` for an
   employer with a job in progress for its `Hire` only, would let an employer with a long own job
   run an establishment. It changes WLD 6.4 and AGT 6.2. *Next step:* the first scenario that shows
   it (S24).
5. **A wage that responds to scarcity.** A bid that rises with the unfilled vacancies of the last
   tick is a delayed-gain loop of the shipper's form (L84, L85); a stability condition on its
   gain would be needed. EMP-D1 refuses it for version 1. *Next step:* EXP with `labour.py`'s
   structure and a real chain.
6. **A reservation from the target wage.** `W = r_bp * B_r * D_year / (10 000 * D_work)`
   (WEL 3.1) as the reservation of a household with no alternative, in place of 0: it would keep a
   hand from taking a wage that cannot reach its target in its expected work ticks, and it would
   leave a glut with unemployment and no wage cut. *Next step:* STY, once S04 has run.
7. **A steady state with employers is not a zero-profit state.** `spec-viability.md` solves for
   prices equal to the cost with labour at the wage dual; with a margin shared by `phi` the
   employer keeps `1 - phi` of it, so the prices of a pack with employers sit above the solver's
   by that margin, and a warm start has no hire in flight. *Next step:* VIA and REF; an overlay
   with an employer role and wants (and the warm start's `Jobs` rows for a principal).
8. **A hungry household and the floor.** A household in the own-use mode offers nothing (3.3.2),
   so it takes the floor before a wage. Whether it should take a wage if one is offered is SUB's
   entry rule. *Next step:* SUB and EMP with the first scenario that has both.
9. **Skilled hired labour.** `HireYieldBP` is one factor per employer block, not a function of the
   worker's skill; a pack with skilled and unskilled households writes two blocks' worth of roles. *Next
   step:* SKL, when a pack has skilled wage labour.
10. **The employer's occupation signal** (EMP-A6) counts the profit on hired households as the employer's wage.
    A rule that left the hired execution's margin out of the signal would remove the inflation and
    the role's attraction. *Next step:* SPN, with S06 and S26 with an employer role.
11. **When the wage is paid.** Per tick (6.2) smooths the worker's income and exposes the employer
    to a removal of the worker only by the escrow; an advance or a payment at the end moves the risk.
    The choice is a rule constant. *Next step:* none unless a pack shows the sawtooth of a long job.
12. **A hire across regions.** The lane is the region (5.3.2), so a commuter between two
    regions is a migrant or a traveller, not a worker. A pack with hamlets joined by a short edge
    and a market town would want a commute. *Next step:* when the first consumer has the case; it
    would be a lane over an edge with a travel time and `open_hire` for both ends.

## 23. Where this lives in the code

### 23.1 Layout

| Concern | Home |
|---------|------|
| The module, the role fields, the employer block, the hire record, the sentinels, the admission, the formation, the release | the root package: `employment.go` (architecture.md 2.6; `AGENTS.md` layout gains the file) |
| The pure arithmetic: the wage, the bill, the order of offers and vacancies, the positions bound, the closed form of 11.2 | `internal/employment` (standard library only, over primitive slices; no commodity, role or agent type, architecture.md 2.9) |
| The default offer and vacancy | the root, `DefaultStrategy.Work` (`strategy.go`), calling `internal/employment` |
| The refusal `open_hire` and the removal steps | the root, with `relocate` and the removal of AGT 3.6 |
| The Escrow account and the reasons | `internal/ledger` (the kind and three rows) |

The package `internal/employment` is created when first needed (architecture.md 2.1). It does
not import the root, `config/` or `strategy/` and takes no random source: the tie keys `K(t, id)` are
computed by the root (`internal/ordering`'s finaliser) and are passed in as `uint64`. No public package and no new extension point:
`JobRequest` gains two value fields and `Strategy` keeps its three methods (architecture.md 4.3).

### 23.2 New domain terms

For the glossary of `specs/README.md` and `doc.go` (architecture.md 3.2): employment (the
module), employer, worker (wage earner), hire (the one-job contract and its record), `HireID`,
hire list, establishment (an employer's own labour and its hires), vacancy, positions, labour
offer, **reservation wage** (not the ledger's reservation counter, `Reserve` of `LabourOffer` is the field), **employer's bid** (not an order side; the `Bid` of `HireRequest`), wage (money per household-tick; never a field), worker share,
labour round, lane (of the round; MER's lane is a pair of a region and a commodity), escrow (of a
hire), payroll, unmatched (household), unemployment (a measure), hired yield, `open_hire`.
"Employer" and "worker" are roles of an agent in a hire, not entities; "wage" is measured and
derived, not configured (`spec-currency.md` 2.6).
