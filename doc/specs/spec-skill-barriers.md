# Spec: Skill barriers: tools, premium checks and training

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-SKL-NNN` and
`NFR-SKL-NNN`, which tests cite (testing.md section 4). They are in section 12.
Sections 8 to 12 were added in the third round of the specs (observability, state,
the register entries, the decisions taken and the requirements); sections 1 to 7 keep
their numbers.

**The numbers in this spec are placeholders** where they are not definitions: the
tool prices and durabilities of 2.9, the defaults of section 5, the tolerance of 3 and
the magnitudes of 2.8 and 4.8 (N-015, N-027). No default is claimed to be stable.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)). The barriers apply only to a role that declares a required tool or a traineeship.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).

## 1. Purpose and scope

A skilled role earns a premium over an unskilled one only if something limits
entry to it. This spec covers the two barriers the model uses:

- a required tool, made by a supply chain and priced by the market (section 2);
- how the resulting premium is checked (section 3);
- a traineeship (apprenticeship in a medieval setting) under a trainer (section 4).

Section 5 lists the parameters this spec owns, section 6 how to calibrate the
barriers and section 7 what is out of scope. How wages and premiums are
measured is in `spec-welfare-and-wages.md` section 4. What happens to a tool
when its holder dies uses the estates in `spec-estates.md`.

## 2. Entry barrier: a required tool

If any agent may enter any role freely, a well-paid role attracts entrants
until its income falls to that of the others. bazaarBot does this directly: a
bankrupt agent is replaced by the class with the best recent profit. A premium
therefore persists only when something limits entry. In this design that is a
**tool**: a commodity a skilled role must hold to run its recipes. Tools are
finite, made by supply chains, and cost real money.

2.1. A recipe may require a tool. The agent must hold it to run the recipe.
Use wears the tool down by a deterministic **durability** counter: each
execution reduces it by `wear_per_use`, and at zero the tool is destroyed (an
event). A counter replaces bazaarBot's "break the tool with 10% chance", which
needs random numbers: a tool with durability 10 has the same expected life with
no randomness. Wear is not random at all: it happens at the start of a job, on
distinct items, the most worn first (`spec-production.md` 4.4). The data fields are
split between two specs: `Durability` on a commodity (the capacity to wear) belongs to
`spec-goods.md` section 5, and `Requires` on a recipe and `WearPerUse` (the rate) belong
to `spec-production.md` 4.4. A tool is never an input and never appears in `Inputs`.

2.2. A tool is made by a recipe, like any other commodity. A toolmaking role
turns inputs (for example metal and wood in the example chain) into tools, and
those inputs come from other roles. No tool appears except by a recipe, apart
from the tools in agents' starting inventories. So the number of skilled agents
is bounded by the tools that exist, and the supply of new tools is bounded by
the toolmakers' output and their inputs. Scarce tools mean a dearer tool and a
persistent premium; plentiful tools shrink it. If the toolmakers die out,
skilled work stops, which is an intended collapse path.

2.3. An agent entering a skilled role (as a spawner replacement into the role it replaces, or by a voluntary change of role, `spec-spawner.md` 6.2, 7.2) must acquire a tool on the market. Until it does, it cannot run the
recipe and earns nothing. Validation warns when a role's starting money cannot
buy a tool at the reference price, since such a role could never be entered. The
warning's rule name is `tool_unaffordable` (`spec-config.md` 7.4.2); it never fails
validation.

2.4. **Cost is emergent.** No tool price is configured. The market sets it,
and the supply chain sets what it has to cover. A toolmaker survives only if
its sales cover its inputs and its own upkeep, so in the long run the price
tends to a cost floor plus a scarcity rent:

```
P_floor = inputs_per_tool + labour_days_per_tool * W_toolmaker
```

- `inputs_per_tool` is the market value of the recipe's inputs per tool.
- `labour_days_per_tool` is the toolmaker's working ticks per tool: the recipe
  duration divided by its output per execution.
- `W_toolmaker` is the toolmaker's daily wage: the observed wage (`spec-welfare-and-wages.md` section 4.3),
  or the target wage from `spec-welfare-and-wages.md` section 3.1 when calibrating.

When tools are scarce relative to demand, the price rises above the floor (a
rent). When they are plentiful, it falls toward the floor, and below it
toolmakers go bankrupt and leave. Cost acts in **two** ways. bazaarBot's own price
beliefs do not price at cost, but the beliefs of `spec-beliefs.md` do, directly: a
seller's belief never falls below `UnitCost * CostFloor / 10 000` (3.2.2 there), so a
toolmaker does not offer below the cost of its inputs and the wear of its tools. That
unit cost counts inputs and wear and **not the toolmaker's labour**
(`spec-production.md` 9.1), so the belief floor is below the labour-inclusive `P_floor`
above; the part of cost that is labour still acts only through exit, until the labour
cost is counted in `UnitCost` (N-055, undecided in `spec-production.md`). With the hard
floor a market can freeze when buyers cannot pay it (STB-L09, N-039): a tool market is
such a market, and a toolmaker that cannot sell leaves.

The author's levers are therefore the recipe (inputs, duration, output per
execution) and the toolmakers' target ratio, never the price. To make a heavy
tool such as an anvil dear, make it take many toolmaker days and many inputs.

The emergent price is then judged as an **output check**, in days of the
skilled wage:

```
tool_days = tool_price / W_skilled
```

Heavy capital tools cost many days of wages and hand tools a few. From the
sources in `evidence.md` (unverified, see there): a carpenter earned about 3d a
day in the late 13th century and 4d to 6d later; a pair of chisels cost about
8d, roughly one to two days; a blacksmith's anvil cost 20s (240d) and bellows
30s (360d), which at 3d to 6d a day is about 40 to 80 days of wages for the
anvil and 60 to 120 for the bellows. Trades differ
by an order of magnitude or two, so the plausible range is set per tool, not
once. A result far outside it means the recipe's inputs or labour content need
changing.

2.5. **Entry rule.** An agent outside a skilled role buys the tool and enters
the role when the return on the tool beats the return required on capital. Let
`p_d` be the role's expected net premium per working day
(`W_skilled - W_unskilled`, from the observed values of `spec-welfare-and-wages.md` section 4.3, or the
rumoured ones for a remote region), `D_work` its expected working days per
year (the role's `ExpectedWorkTicks`, `spec-welfare-and-wages.md` section 4.2), `P` the tool's ask in the region, and `CapitalReturn` the region's
required annual return in basis points. The agent enters when:

```
p_d * D_work * 10_000 >= P * CapitalReturn
```

Wear is already a recipe cost netted from the wage (section 2.6), so only the
return on the capital tied up is tested here. `CapitalReturn` is a region
parameter, default 1 000 (10% a year). Clark's evidence from land rents puts
real returns at about 10% around 1300, falling to about 5% by 1400 and 4% to 5%
before industrialisation (`evidence.md`). A chain modelled on the later period
sets 500. A higher value makes entry harder and the premium larger. The rule
is applied by the default spawner (`spec-spawner.md` 6.2, 6.4): it reads `p_d * D_work` as the annual gain between the two roles' observed earnings for this household, `(E_d - E_o) * Upkeep * TicksPerYear / 10 000`, switches the agent when its money covers the tool's price, and leaves the purchase to the new role's `Trade`. This spec fixes the form of the inequality so that the premium check (section 3) can use it; the spawner adds the hysteresis band, the review rotation, the caps and the cooldown (`spec-spawner.md` 6).

For a role that requires training (section 4), the capital on the right of
the rule is `P + C_train`, where `C_train` is the cost of the traineeship
defined in section 4.5.

2.6. **Wear cost.** Each execution costs `wear_per_use / durability` of a
tool, valued at the current market price. This is a recipe cost, so it reduces
the role's income (`spec-welfare-and-wages.md` section 2.1). In steady state the skilled population creates
a replacement demand for the toolmakers of:

```
tools_per_tick = skilled_agents * uses_per_tick / durability
```

The author checks that the toolmakers' capacity covers it; if not, the tool
price rises and the premium with it.

As a share of the wage, a tool costing `d` days of wages with `U` uses of
durability costs `d / U` per working day in wear, plus the return on capital,
`d * CapitalReturn / (10_000 * D_work)`. A tool costing 100 days of wages and
lasting 5 000 uses costs 2% in wear; at 10% and 250 days the capital return is
another 4%. Even a dear, long-lived tool takes only about 6% of a skilled
worker's income, and it creates only a small flow of orders for toolmakers: a
few toolmakers can supply a large skilled population. That is a consequence of
durable capital, not a defect.

2.7. **Used tools and death.** While its holder lives, a tool is bound to it
and cannot be resold: a resale market would need a price for each remaining
durability, and one uniform price would invite selling nearly worn tools. When
the holder dies or goes bankrupt, the tool goes to the **estate** of the region
where it died, which lists it for sale on that region's open market.

An estate listing is an ordinary ask, placed by the region's estate pseudo-agent
(`spec-estates.md` 2.7), with a deterministic price:

```
ask = ref_price * durability_left / durability_full
```

`ref_price` is the region's **effective reference price** of the tool commodity
(`spec-goods.md` 6.3), as `spec-estates.md` 2.7.3 and FR-EST-008 state; it is the
estate's rule and this spec follows it. An earlier version of this section anchored
the ask to the region's recent average price, falling back to the cost floor of
section 2.4 and then to the reference price. That chain is dropped to remove a
conflict with the estate spec, and it costs little: an unsold listing falls by
`EstateMarkdown` each tick (below), so a listing priced above the market reaches it
within a few ticks, and one priced below it sells at once to the cheapest-ask rule,
the estate's pool gaining less. If experiments show that the starting anchor matters,
the better anchor is the mean of the market's price window (`spec-beliefs.md` 7.2),
then the cost floor, then the reference price; it is a change to the estate spec
(open question 4 of section 11). The result is truncated and never below the
resolution floor (`spec-currency.md` section 3).

- Clearing matches buyers to the cheapest asks first, so used tools sell before
  new ones, and the buyer pays the midpoint of its bid and the ask (half to even, `spec-clearing.md` 5.1). The buyer
  receives the tool with its remaining durability.
- Each tick an unsold listing is reduced by `EstateMarkdown` basis points
  (configuration), never below the resolution floor, so a listing does not sit
  forever.
- The sale only moves the tool, so goods conservation holds. The proceeds go to
  the region's estate pool (`spec-estates.md` section 2).
- **How "bound to its holder" is enforced.** The ledger does not mark a tool as
  unsaleable, and the market has no such rule (`spec-production.md` 7.4 rejects a
  `Tradable` flag). The default strategy enforces it: it asks for a commodity that some
  recipe requires only in a role that **makes** it (`spec-agents.md` 7.3.5), so a user
  of a tool never lists it, while a toolmaker sells its new output. A consumer's
  strategy may resell a tool, which is the consumer's choice and shows in the flows.
- The ledger holds each tool as a lot of quantity one, carrying its remaining
  durability. That is specified in `spec-ledger.md` 5.1.4 and 5.4: a lot of
  quantity `n` and remaining durability `d` is `n` such items, and wear falls on
  the most worn item first.

A chain with no tool requirement, or free tools, has no barrier, and its
observed premium drifts toward 10 000 (no premium). That is a valid result and
a finding for the author, not a bug.

2.8. **How much premium a tool can explain.** Little, at realistic prices.
The entry rule (section 2.5) gives a break-even premium per working day of
`p_d = P * CapitalReturn / (10_000 * D_work)`, so:

```
premium_bp = 10_000 + P * CapitalReturn / (D_work * W_unskilled)
```

With `W_unskilled` = 2 190 (`spec-welfare-and-wages.md` section 3.2), `D_work` = 250 and `CapitalReturn` =
1 000, a tool costing 100 days of the unskilled wage (`P` = 219 000) gives
`p_d` = 87.6 and a premium of 10 400, which is 4%. For a premium of 1.5
(15 000) the tool would have to cost `0.5 * 2 190 * 250 * 10 = 2 737 500`, which
is 1 250 days of the unskilled wage (5 working years). At a 5% return it would
be 2 500 days.

So a tool is a real barrier to entry and a real source of income for
toolmakers, but it is not what historical skill premiums mostly paid for. A
premium of 1.4 to 2.0 is closer to the size of a long training period: a
seven-year traineeship is about 1 750 working days of gross unskilled wages
(`evidence.md`). Training is therefore modelled as well (section 4), and the
two together are quantified in section 4.8. A chain with a tool but no
training is expected to report a premium well below the historical band. That
is a finding, not a defect.

2.9. **Starting values for the reference chain.** These are placeholders to
tune by experiment (`spec-welfare-and-wages.md` section 5.2 describes the method for the spread). The
sources give no use counts, so durability is an assumption.

| Tool class | Plausible price, in days of skilled wage | Durability, in working uses | Basis |
|------------|------------------------------------------|-----------------------------|-------|
| Hand tool (chisel, hammer) | 1 to 10 | 500 (two years of daily work) | A pair of chisels cost about 8d, one to two days (unverified). Hammers wear out sooner than anvils |
| Heavy tool (anvil, bellows) | 40 to 120 | 5 000 (twenty years of daily work) | Anvils last decades to centuries with care (unverified) |

## 3. Checking the premium

The engine does not aim at any premium, so a check on the reference chain's
observed premium has to say what it is checking. Four options were considered.

| Option | Check | For | Against |
|--------|-------|-----|---------|
| A. One historical band | Observed premium within 14 000 to 20 100 basis points (Clark's English series, 1220 to 1900: low 1.40 in 1460, high 2.01 in 1290) | Simple and sourced | Too wide to catch a chain that lands at 1.9 when its tools imply 1.5. Ties the fixture to English building trades |
| B. An era band | 1400 to 1900: 14 000 to 16 300. 1220 to 1350: 15 600 to 20 100 | Tighter | Claims the chain is medieval or early modern, which it is not |
| C. Ordinal properties | With a tool requirement the premium is above 10 000. It rises with the tool's price, and with free tools it is within a small tolerance (initially 500) of 10 000 | Tests the mechanism, depends on no absolute number | Says nothing about the size |
| D. Derived equilibrium | Premium within a tolerance of the value predicted by the chain's own tool price and the entry rule (below) | Self-consistent for any chain | Needs the entry rule (section 2.5), which is specified but not yet built |

Decision:

- **C is required** and always runs. It is what the tool mechanism claims.
- **A is reported, not asserted.** The calibration test logs the observed
  premium against the whole-series band. A value outside it is a prompt to
  look at the recipe economics, not a test failure.
- **D replaces A as the numeric check once the entry rule is built.** At
  break-even the premium follows from the tool's price (section 2.8):

```
premium_bp = 10_000 + (tool_price + C_train) * CapitalReturn / (D_work * W_unskilled)
```

  `C_train` is the training cost of section 4.5, and is 0 for a role with no
  training.

  The tolerance is `max(25% of the predicted excess over 10 000, 200 basis points)`,
  and it is raised to twice the coefficient of variation of the observed premium across
  seeds, as measured on the reference chain, if that is larger. Both numbers are
  **placeholders**: the 25% was a guess before training was included. With training the
  predicted excess is larger (2 733 for the figures of 4.8, a tolerance of 683, a
  premium band of about 0.07), so a relative tolerance is looser in absolute terms, and
  the 200 basis points keeps it from vanishing for a tool-only chain (predicted excess
  400, 25% = 100). The rule for tuning it is the one just stated, run by
  `spec-experiments.md` on the first working chain; no number is fixed from a source. This
  checks that entry behaves as the rule says. It does not check that the tool price is
  right: that is the cost-floor check of section 2.4.

- **B is rejected.**

Changing a check later is cheap. No band is part of the library's API: bands
and tolerances live in the reference chain's calibration data under
`testdata/` and are read by the tests. Changing one is a fixture edit with a
reason in the PR (testing.md 7.2), and no version bump. The exception is the
definition of the observed premium itself: it is exposed in `Snapshot`, so
changing how it is computed is a behaviour change under architecture.md 6.6.

## 4. Training: becoming a skilled worker

A skilled role can require a traineeship. This section defines the pathway
and its costs. It adds a second barrier to entry beside the tool (section 2),
and it is what gives the model a premium of historical size (section 4.8).

4.1. **Pathway.** A role with `TrainingTicks` above 0 can only be entered
through a traineeship. An agent in any other role may apply. If it is
accepted it becomes a **trainee** of a **trainer**, a living agent holding
the target role, for `TrainingTicks` ticks. At completion it changes to the
target role by the role change of `spec-spawner.md` 4.2 (cause `traineeship`): it keeps its account and money and the goods its new role can trade, and goods outside the new role's tradable set pass to the estate. The application is made by the default spawner (`spec-spawner.md` 4.4, 6.2) and only for an agent of weight 1. It still needs the tool (section 2)
before it can run the role's skilled recipes. A role with `TrainingTicks` of 0
has no training barrier, only the tool.

The agent's role does not change while it is a trainee. Traineeship is a
status. Until completion the agent does not run its own role's recipes and is
not counted among the target role's agents, so it does not affect that role's
observed wage or premium (`spec-welfare-and-wages.md` section 4.3).

4.1.1. **Weight 1 only (version 1).** A trainer and a trainee must both be agents of
weight 1. A cohort of weight `w` is `w` identical households working in lockstep
(`spec-scale.md` 2), and a place, a teaching load and a board are per household; to
make a cohort a trainer would need places that scale with the weight and a partial
admission of its households, which this version does not define. Validation rejects a
role with `TrainingTicks` above 0 that has a group with a scale above 1
(`ErrOutOfBounds`, rule `traineeship_needs_weight_one`), and an agent of weight above 1
does not apply. The world refuses an `Apply` for such an agent (`needs_weight_one`, `spec-spawner.md` 4). An external agent has weight 1 (`spec-scale.md` 4.7) and may be a trainee when the consumer directs it; the default spawner never applies for an external agent, and a trainee is exempt from the failure ladder (`spec-spawner.md` 5.2.3), so a mortal external trainee is consistent with weight 1. The first consumer's world needs no weight above 1 (N-030). Lifting
the limit is an open question (section 11).

4.2. **Board, work and teaching.** During training:

- The trainer supplies the trainee's upkeep in kind, buying the goods like
  its own. Mechanically (`spec-agents.md` 5.7, FR-AGT-068): in the consumption phase the
  trainee's upkeep is **consumed from the trainer's account** with the reason
  `consumption`, and the trainer's own use, and so its targets, include its trainees'
  upkeep. The trainee's income is the value of that board (`spec-welfare-and-wages.md`
  6.3) and no cash, so its welfare ratio is 10 000 while it is boarded, and it does not
  starve while its trainer lives. The same value is a cost netted from the trainer's
  income (`spec-welfare-and-wages.md` 2.1.6). Historically, trainers provided food,
  lodging and clothing.
- The trainee runs the trainer's role's **trainee-eligible recipes**
  (`spec-production.md` 8): recipes that have no tool (`ErrEligibleHasTool`), so the
  trainee needs neither the tool nor the skill. The trainee is the **executor** and the
  trainer the **principal**: the inputs are taken from the trainer's account and the
  outputs go to it, and the output is `TraineeYield` of a full execution
  (`spec-production.md` 8.2, a placeholder of 5 000 basis points) while the inputs are
  not reduced. A job in progress when the traineeship ends is abandoned
  (`principal_left`, `spec-production.md` 5.4).
- Each trainee takes `TeachingLoad` basis points of the trainer's working
  time. A per-trainer accumulator adds the sum of its trainees' loads every
  tick. When it reaches 10 000, it subtracts 10 000 and that tick is a teaching
  tick, in which the trainer does not run its skilled recipe. No randomness is
  used.

4.3. **Trainer's incentive.** A trainer takes a trainee only if the
expected net gain per trainee-tick is not negative:

```
gain = output_a - U_a - TeachingLoad * W_trainer / 10_000  >= 0
```

`output_a` is the net value of the trainee's eligible output per tick, `U_a`
the trainee's upkeep cost, and `W_trainer` the trainer's observed daily wage.
The historical economic account is that trainers recouped the cost of training
from trainees working at below-market wages, and that trainees were not
cheap labour (`evidence.md`). The author sets the eligible recipes so that
training is or is not profitable at the chain's prices. If `gain` is negative,
no places open and entry stops. This version has no cash fee and no credit:
historical premiums paid by parents are not modelled (section 7).

4.3.1. **The gain is read when a place opens.** It decides whether a trainer takes a
new trainee. It does not end a traineeship in progress: a trainee already accepted
stays until the course ends or the trainer dies (4.6). So a small price change that
turns the gain negative closes new places and does not throw out those who have started,
which softens the incentive cliff (STB-L37). Whether `output_a` for the test uses the
function `TrainingPays` of `spec-production.md` 8.3 over supplied prices is the
implementation's choice; the inequality is the same.

4.4. **Places and queue.** A trainer holds at most `MaxTrainees` at once.
The default is 3, the limit set by the English Statute of Artificers of 1563
(`evidence.md`, a search snippet and unverified; the default is a placeholder). Applicants queue per role per region, ordered by the tick they
applied and then by agent identifier. A place opens when a trainer has fewer than
the maximum and passes section 4.3. The supply of new skilled workers is bounded
by:

```
skilled_per_tick <= trainers * MaxTrainees / TrainingTicks
```

When there are more applicants than places, this throttle, not the cost, keeps
the premium up.

4.5. **Application rule.** An agent applies when the return on its
investment beats the required return (section 2.5), with the capital being
the tool plus the cost of training:

```
p_d * D_work * 10_000 >= (P + C_train) * CapitalReturn
C_train = TrainingTicks * S_u
```

`S_u` is the agent's observed income per tick above its own upkeep, floored at
0: the surplus it gives up while in training. It gives up only the surplus,
because it is fed.

**The queue counts (STB-L41, README "Still open" 2).** Time in the queue does not cost
the agent its surplus, since it goes on working in its own role, but it **delays** the
premium. The premium stream starts `q_w` ticks later, where `q_w` is the expected wait:
the applicants queued ahead of the agent for the role in its region times
`TrainingTicks`, divided by `n_max` (4.9) and rounded up, or no application if there are
no trainers. With a linear discount at `CapitalReturn` per year (a placeholder
simplification; exact for small delays) the rule becomes

```
p_d * D_work * (10_000 - CapitalReturn * q_w / TicksPerYear) >= (P + C_train) * CapitalReturn
```

with the bracket floored at 0, and no application when it is 0. Every quantity is
local: the queue ahead of the agent and the count of trainers in its region. A long
queue therefore lowers the appeal of joining it, which is the correction that the
earlier version of this rule lacked. It adds no parameter.

4.6. **Interruptions.** If a trainer dies, its trainees keep the ticks
they have served, return to their own role, and rejoin the queue in order of
ticks served, then application tick. A trainee that dies goes to the estate
(`spec-estates.md` section 2). The default spawner never removes a trainee by the failure ladder (its board comes from the trainer, `spec-spawner.md` 5.2.3), so a trainee leaves the world only through a consumer's removal, which ends the traineeship first (`spec-agents.md` 3.6). A trainer is subject to the ladder like any agent. Traineeships can also end early in this model only through
the trainer's death: voluntary early termination, which was common in the
historical record, is not modelled.

4.7. **Events.** `TraineeshipStarted` (trainee, trainer, role),
`TraineeshipCompleted` (agent, from role, to role) and `TraineeshipEnded`
(agent, reason). They carry identifiers only. Counts per role are bounded by
the configuration, so they are safe as metric attributes.

4.8. **Magnitude.** `B_r` = 1 000, an unskilled wage of 2 190 at a ratio of
1.5 over 250 days, `CapitalReturn` 1 000, and a seven-year traineeship of
`TrainingTicks` = 2 555:

- The unskilled agent's income per tick is `1.5 * 1 000 = 1 500`, so its
  surplus is `S_u` = 500.
- `C_train = 2 555 * 500 = 1 277 500`, which is about 583 days of the
  unskilled wage. A tool costing 100 days (219 000) is added to give a capital
  of 1 496 500.
- The break-even premium is
  `10 000 + 1 496 500 * 1 000 / (250 * 2 190) = 12 733`, a premium of 1.27.
  Training alone gives 12 333, and tool alone gives 10 400.

So at a 10% required return, the cost of tool and training together explains a
premium of about 1.27, against the historical 1.4 to 2.0. The rest is expected
to come from the throttle of section 4.4 (a queue for places), and from
trainers' teaching cost. At a 5% return the capital-cost part falls to about 1.14.
These figures are placeholders to check by experiment, not tuned values.

A seven-year traineeship is 2 555 ticks, longer than a short test run. A
chain used in fast tests may shorten `TrainingTicks`, which changes both the
cost and the throughput, and is recorded as a deviation in the fixture.

4.9. **The shortest traineeship that shows the throttle** (README "Still open" 3). The
throttle of 4.4 is visible only when it binds, that is when applicants would arrive
faster than places turn over. Let `n_max = trainers * MaxTrainees` be the places in a
region and `a` the flow of applicants per tick that the application rule (4.5) would
produce if there were no limit. The throttle binds if and only if

```
a > n_max / TrainingTicks          that is        TrainingTicks > n_max / a
```

so the shortest `TrainingTicks` that shows it is just above `n_max / a`. A chain that
wants it **clearly** visible chooses `TrainingTicks = 2 * n_max / a` (capacity half the
demand, the queue growing by `a / 2` a tick), and runs for at least three
`TrainingTicks`, so that the flow of completions settles at the capacity. Example, with
placeholder numbers: 10 trainers, `MaxTrainees` 3, so `n_max` = 30, and `a` = 0.5 a
tick: the throttle binds above 60 ticks, is clear at 120 and needs a run of 360 ticks.
A shorter course also shrinks the cost: `C_train = TrainingTicks * S_u`, so with 120
ticks the training part of the premium is `120 * 500 * 1 000 / (250 * 2 190) = 109`
basis points (1.1%) against 12 333 - 10 000 = 2 333 at 2 555 ticks. A fast chain shows
the throttle and almost none of the training cost, and the premium check D stays
self-consistent because it uses the course's own `C_train`; the fixture records the
deviation. The length of the run for the fixture is `spec-reference-chain.md`'s.

## 5. Parameters

| Scope  | Field          | Meaning                                                     | Default |
|--------|----------------|-------------------------------------------------------------|---------|
| Role   | `TrainingTicks` | ticks of traineeship needed to enter the role; 0 means no training (section 4.1) | 0 |
| Role   | `MaxTrainees` | trainees one trainer may hold at once (section 4.4) | 3 |
| Role   | `TeachingLoad` | share of a trainer's working time one trainee takes, in basis points (section 4.2) | 1 000 |
| Region | `CapitalReturn` | required annual return on a tool, in basis points (section 2.5) | 1 000 |

All are plain data in the root package's config structs. The tool a skilled role
requires is a recipe requirement and a commodity attribute, specified with
production.

Bounds, enforced by validation with `ErrOutOfBounds` (all provisional placeholders):
`TrainingTicks` from 0 to 1 000 000 (arbitrary and provisional; the product with `S_u`
must fit `int64`, checked);
`MaxTrainees` from 1 to 100; `TeachingLoad` from 0 to 10 000; `CapitalReturn` from 0 to
`MaxBasisPoints`. A role with `TrainingTicks` above 0 also needs every group of the role
at scale 1 (4.1.1).

## 6. Calibrating skill barriers

For each skilled role, an author:

1. gives it a required tool with a durability, and a toolmaking recipe whose
   inputs, duration and output set the tool's cost floor (section 2.4);
2. checks that the emergent price in days of wage is plausible, that the payback
   is acceptable and that the toolmakers' capacity covers replacement
   (section 2);
3. gives it a training period with trainee-eligible recipes, and a
   profitable-or-not choice for trainers (section 4);
4. runs the chain to steady state and checks the observed premium against the
   checks in section 3.

The wider calibration steps are in `spec-welfare-and-wages.md` section 8 and
`spec-currency.md` section 5.

## 7. Non-goals and future work

- Cash premiums paid by a trainee's family, association rules beyond the
  trainee limit, intermediate-stage years (journeyman years in a medieval
  setting), and voluntary early termination.
  Section 4 models a single traineeship pathway funded by the trainer and
  repaid in the trainee's work.

## 8. Observability

8.1. **Events.** `TraineeshipStarted` (trainee, trainer, role), `TraineeshipCompleted`
(agent, from role, to role) and `TraineeshipEnded` (agent, reason) of 4.7 are the only
events of this spec. They carry the `Tick`, identifiers and the reason, no map, and are
emitted after the transition (observability-and-logging.md 4.1, 4.4); they are
constructed only when an observer was supplied (4.5 there). The reasons of
`TraineeshipEnded` are `trainer_left` (the trainer died or went bankrupt), `agent_left`
(the trainee did), and `role_removed`. A tool's wear and destruction are already
`BoundaryFlow` under `wear` (`spec-ledger.md` 12.1) and the estate's listing is
`EstateOpened`.

8.2. **Snapshot.** Per region and role: the places, the places taken, the queue length,
the trainers, and the observed premium (`spec-welfare-and-wages.md` 4.3). The
throttle and the queue are therefore observable (FR-STB-013). All are bounded by the
configuration.

8.3. **Logs.** Nothing above `Debug`. At `Debug`, guarded by `Enabled`: "traineeship
refused" with `agent`, `role` and `reason` (`weight`, `no_trainer`, `gain_negative`,
`queue_too_long`). One `Warn`, bounded by the configuration: "no places open", per
region and role, emitted when a role with applicants has had no place open for 10
consecutive ticks because `gain` is negative (STB-L37), at the tenth tick and every
tenth after, with `region`, `role` and `reason`. It is the counter of the incentive
cliff. No `Info`, no `Error`.

## 9. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, all plain data in a
fixed order with no maps:

- for each trainee, in agent ID order: its trainer, the target role, the ticks served,
  and the tick it applied;
- for each trainer, in agent ID order: the teaching accumulator (4.2) and the list of
  its trainees;
- for each (region, role), in region then role order: the queue of applicants, in queue
  order (each: agent, application tick, ticks served), and the run of ticks for the
  `Warn` of 8.3 (diagnostic, saved so that a resumed run logs as the original did).

The remaining durability of a tool is in the ledger's lots (`spec-ledger.md` 11). The
estate's listing prices are the estate's (`spec-estates.md` 2.7.5). **Not saved:** the
places, the throttle and the expected wait (derived), and the entry decision.

## 10. Entries for the stability register

Merged into `spec-stability.md` section 4 (`STB-L36` the occupation-choice cobweb,
`STB-L37` the trainer's incentive cliff, `STB-L41` the traineeship queue, and with them
`STB-L26` durables and the replacement cycle and `STB-L27` a single tool, which come
from the tool of section 2). The dampers this spec specifies are D14 (`MaxTrainees` and
the queue), D19 (the spread of ages of tools, 2.8 and 2.9) and D20 (the training
throttle, 4.4). Changes of this revision for the register owner:

- **STB-L41 is resolved in design.** The expected wait is in the application rule (4.5),
  so a long queue lowers the appeal of joining. Status moves from untested bias to
  `hand`.
- **STB-L37 is softened.** The gain closes new places and does not end a traineeship in
  progress (4.3.1). It is still a cliff for new entry; the counter is the `Warn` of 8.3.
- **STB-L36 delay.** The expected wait adds a term to the delay; the occupation cobweb's
  swings in a role with a throttle and a queue should be smaller. *Test:* S06.

## 11. Decisions taken and open questions

11.1. **Decisions** of this revision, each stated where it is made and reversible:
the ask of a used tool follows the estate's rule, the effective reference price (2.7);
"cost acts through exit" is reworded because beliefs now floor a seller's price at its
unit cost, without labour (2.4); the tolerance of the derived premium check is
`max(25%, 200 basis points, twice the coefficient of variation)` (3); a trainer and a
trainee have weight 1 (4.1.1); the gain does not end a started traineeship (4.3.1);
the queue's expected wait is in the application rule (4.5); the throttle test is
derived (4.9).

11.2. **Answers to the open items of the README ("Still open")** that belong here:

| Item | Answer |
|------|--------|
| 2. Should time in the queue count in the application rule? | **Yes.** It delays the premium, so it enters as a discount of `CapitalReturn * q_w / TicksPerYear` (4.5); it does not cost the surplus, since the agent keeps working. Closed. |
| 3. The shortest traineeship that shows the throttle? | **Derived** (4.9): `TrainingTicks > n_max / a`, clear at `2 * n_max / a`, with a run of at least three courses; a fast chain shows the throttle but little of the training cost. The numbers for the reference chain are `spec-reference-chain.md`'s. Closed as a rule; the values wait for the chain. |
| 7. Is a 25% tolerance on the derived premium reasonable now that training is included? | **Replaced** by `max(25% of the predicted excess, 200 basis points, twice the observed coefficient of variation)` (3): the 25% is kept as the relative part, and the measured spread sets it when larger. Both are placeholders; the procedure is `spec-experiments.md`'s on the first working chain. |
| 8. The two traineeship-premium elasticities disagree | **Resolved by opening the source.** See 11.3. |

11.3. **The elasticities (README "Still open" 8, `evidence.md`).** The source is
Justman and Beek, "Market forces shaping human capital in eighteenth-century London",
Economic History Review 68(4), 2015, from Campbell's *The London Tradesman* (1747),
opened at <https://tzin.bgu.ac.il/~kvdbeek/papers/Campbell%20EcHR_2015.pdf> (a version
of the paper; text extracted from the PDF and read: the abstract, the description of
apprenticeship, the discussion of premiums and net costs, and the regression results).
The two figures in `evidence.md` were not in conflict about one estimate; they are two
estimates of the same paper. The **ordinary least squares** estimate of the elasticity
of the premium with respect to journeymen's wages is **0.35**, with respect to the
master's set-up costs **0.25**; after instrumenting wages (the Wu and Hausman tests
indicate that wages are endogenous) the elasticity with respect to wages is **about
1.0** (the abstract says "unitary") and with respect to set-up costs 0.23 to 0.25.
The earlier snippet figure of 0.4 for wages is the rounded OLS value and the "one for
one" snippet is the instrumented one. The paper also says: apprenticeship was usually
seven years; the master provided board, lodging and teaching; the premium was paid in
cash to the master at the beginning, to cover the master's net cost early in the term
(room and board, the master's time, wasted materials), when the apprentice is less
productive, and to guard against early departure. Typical premiums were 10 to 20
pounds. **Use in this spec:** none sets a default. The finding that the premium moves
one for one with the wage is consistent with the form of the trainer's gain
(`TeachingLoad * W_trainer` rises with the trainer's wage, 4.3), and the cash premium is
a mechanism this spec does not model (section 7). `evidence.md` should replace the
row's "unverified, the two snippets disagree" by "read: OLS 0.35 and 0.25, IV about 1.0
and 0.23 to 0.25".

11.4. **Open questions**, each with a next step:

1. **Traineeships for cohorts** (4.1.1): places, load and board per household. Needed
   only if a consumer scales a skilled role; next step, when a consumer needs it.
2. **Voluntary early termination** (4.6, section 7), common in the record
   (`evidence.md`): not modelled; the Justman and Beek account says the premium guards
   against it. Next step, a later module with cash premiums.
3. **A cash premium and credit** (section 7): `spec-credit.md`.
4. **The anchor of an estate's ask** (2.7): the effective reference price today; the
   price window mean is better. Next step, the estate spec's owner.
5. **Tool durability and prices** (2.9) remain assumptions; the evidence rows are
   unverified (`evidence.md`).
6. **`TeachingLoad` default of 1 000** is a guess; the trainer's gain depends on it.

## 12. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. Every
requirement is derived from text of sections 1 to 11; (new) marks behaviour decided in
the third round.

### Functional

| ID | Requirement |
|----|-------------|
| FR-SKL-001 | An agent MUST run a recipe that requires a tool only when it holds the tool, and a tool MUST NOT appear in a recipe's inputs. |
| FR-SKL-002 | Tool wear MUST be a deterministic durability counter reduced by `WearPerUse` per execution, most worn item first, and MUST use no random number. |
| FR-SKL-003 | A tool MUST come into being only by a recipe or in an agent's starting inventory, and no tool price MUST be configured. |
| FR-SKL-004 | A role whose starting money cannot buy its required tool at the reference price MUST produce the validation warning `tool_unaffordable` and MUST NOT fail validation. |
| FR-SKL-005 | An agent MUST enter a skilled role by buying the tool only when `p_d * D_work * 10 000 >= P * CapitalReturn`, or, for a role that requires training, `p_d * D_work * (10 000 - CapitalReturn * q_w / TicksPerYear) >= (P + C_train) * CapitalReturn`, with the bracket floored at 0 and no application when it is 0. (new for the queue) |
| FR-SKL-006 | `CapitalReturn` MUST be a region parameter in basis points with default 1 000. |
| FR-SKL-007 | A tool MUST be bound to its holder: the default strategy MUST NOT place an ask for a commodity that a recipe requires unless the agent's role makes it. (new) |
| FR-SKL-008 | When a holder dies or goes bankrupt, its tool MUST go to the estate of the region where it died, and the estate's ask for it MUST be the effective reference price times the remaining durability over the full durability, truncated and never below the resolution floor. (new for the anchor) |
| FR-SKL-009 | An unsold estate listing MUST be reduced each tick by `EstateMarkdown` basis points, never below the resolution floor, and the sale MUST only move the tool so that goods conservation holds. |
| FR-SKL-010 | A role with `TrainingTicks` above 0 MUST be enterable only through a traineeship, and a role with `TrainingTicks` of 0 MUST have no training barrier. |
| FR-SKL-011 | A trainee MUST keep its role during the traineeship, MUST NOT run its own role's recipes and MUST NOT be counted among the target role's agents for the observed wage or premium; at completion it MUST change to the target role keeping its money and goods. |
| FR-SKL-012 | Only an agent of weight 1 MUST be a trainer or a trainee, and a configuration with a role of `TrainingTicks` above 0 that has a group of scale above 1 MUST be rejected with `ErrOutOfBounds`. (new) |
| FR-SKL-013 | While boarded, a trainee's upkeep MUST be consumed from the trainer's account with the reason `consumption`, and the trainer's use MUST include it. (new) |
| FR-SKL-014 | A trainee MUST run only trainee-eligible recipes of the trainer's role, as executor with the trainer as principal, taking inputs from and creating outputs in the trainer's account, and its output MUST be `TraineeYield` of a full execution. (new for the references) |
| FR-SKL-015 | Each tick a trainer's teaching accumulator MUST increase by the sum of its trainees' `TeachingLoad`, and when it reaches 10 000 it MUST subtract 10 000 and that tick MUST be a teaching tick in which the trainer does not run its skilled recipe. |
| FR-SKL-016 | A trainer MUST take a trainee into a free place only if `output_a - U_a - TeachingLoad * W_trainer / 10 000 >= 0`, and a negative gain MUST NOT end a traineeship in progress. (new for the second clause) |
| FR-SKL-017 | A trainer MUST hold at most `MaxTrainees` trainees, default 3, and applicants MUST queue per role and region ordered by application tick and then agent ID. |
| FR-SKL-018 | `C_train` MUST equal `TrainingTicks` times the agent's observed income per tick above its upkeep, floored at 0. |
| FR-SKL-019 | The expected wait `q_w` MUST be the applicants queued ahead times `TrainingTicks` divided by `trainers * MaxTrainees`, rounded up, and an agent MUST NOT apply where there is no trainer. (new) |
| FR-SKL-020 | When a trainer dies its trainees MUST keep the ticks served, return to their own role and rejoin the queue ordered by ticks served and then application tick. |
| FR-SKL-021 | `TraineeshipStarted`, `TraineeshipCompleted` and `TraineeshipEnded` MUST carry only identifiers and the reason and no map, and MUST be emitted after the transition. |
| FR-SKL-022 | The `Warn` "no places open" MUST be emitted per region and role, never per agent, only after 10 consecutive ticks with applicants and no place because the gain is negative, at the tenth tick and every tenth after. (new) |
| FR-SKL-023 | The premium check C MUST always run: with a tool requirement the observed premium MUST be above 10 000 and rise with the tool's price, and with free tools it MUST be within the tolerance of 10 000. |
| FR-SKL-024 | The whole-series band of check A MUST be reported and MUST NOT fail a test, and the derived check D, once the entry rule exists, MUST accept a premium within `max(25% of the predicted excess over 10 000, 200 basis points)` of `10 000 + (tool_price + C_train) * CapitalReturn / (D_work * W_unskilled)`, raised to twice the observed coefficient of variation if larger. (new for the floor) |
| FR-SKL-025 | Bands and tolerances MUST live in the reference chain's data under `testdata/` and MUST NOT be part of the library's API. |
| FR-SKL-026 | `TrainingTicks`, `MaxTrainees`, `TeachingLoad` and `CapitalReturn` MUST have the defaults and bounds of section 5, and a value outside its bound MUST be rejected with `ErrOutOfBounds`. |
| FR-SKL-027 | The state of section 9 MUST be saved as plain data in a fixed order with no maps, and a world restored between ticks MUST give the same traineeships, queues and teaching ticks as an uninterrupted run. |
| FR-SKL-028 | No rule of this spec MUST branch on a commodity, recipe or role identifier, and nothing in it MUST use a random number. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-SKL-001 | With the default logger and observer, the entry decision, the teaching accumulator and the queue operations of a tick MUST allocate nothing beyond scratch held by the world. |
| NFR-SKL-002 | The cost of the entry decision SHOULD be proportional to the applicants and trainers of a role in a region and not to the weights. |

### Behaviour found with no clear statement (flagged)

- Whether a started traineeship survives a negative gain: not stated; FR-SKL-016 says it
  does.
- What a cohort does as trainer or trainee: not stated; FR-SKL-012 restricts to weight 1.
- The anchor of an estate's ask: two specs disagreed; FR-SKL-008 follows the estate.
- How "bound to its holder" is enforced: no mechanism was stated; FR-SKL-007 puts it in
  the default strategy.
- The teaching accumulator's initial value and whether a trainer's own `TeachingLoad`
  carries across a change of trainees: not stated; it is state and is saved (section 9).
