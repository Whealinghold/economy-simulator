# Spec: Migration: whether, when and where a household moves

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-MIG-NNN` and
`NFR-MIG-NNN`, which tests cite (testing.md section 4). They are in section 19.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)),
version 1 (`vision.md` section 4). Off unless configured: the zero configuration
has no `Modules.Migration` and no agent ever changes region. Needs
`spec-regions.md` (REG: the relocation of an account, its refusals and the
distance between two regions), `spec-information.md` (INF: the rumours of remote
prices, the only news of another region), `spec-welfare-and-wages.md` (WEL: the
household's own welfare ratio and observed wage), `spec-agents.md` (AGT: the
agent, its upkeep, the tick phases), `spec-spawner.md` (SPN: the clocks and
the viability gate it shares), `spec-production.md` (PRO: the recipe, the yield
table, the abandoned job), `spec-beliefs.md` (BEL: the reseeding of a newcomer's
beliefs) and `spec-config.md` (the module switch and validation). It reads
`spec-estates.md` (EST: relief and the pool), `spec-population.md` (POP: the
stake and the closed region) and `spec-transport.md` (TRN: the refusal
`open_shipment`) and changes none of them, except for the small edits listed in
the handoff.

**Every number in sections 5 to 8, 11, 15 and 16 is a placeholder** except where
a derivation is shown. Nothing has been run against the library, bazaarBot has not
been run, and no acceptance band exists (notes N-015, N-027). `models/mig/loop.py`
(section 16) runs the rules of sections 5 to 8 on two and three regions with an
assumed economy; it is a stylised model of the rules, not of the library, and not
a claim that any default is stable. No default of this spec is a macro constant:
the module sets no price, wage or rate, reads no regional aggregate, and moves
households only.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer, never a household (N-001, N-017). The two worked
examples of section 15 come from different settings and are flagged as examples;
the core and its vocabulary are setting-agnostic.

## 1. Purpose and scope

A household that cannot live on what its region gives it can stay and be helped
(relief, a role change, hunger and the ladder, `spec-spawner.md` 5), or it can go
somewhere else. This spec decides whether it goes, when, and where, with rules
that read only what the household itself has and what it has been told. It covers:

- the **unit** that moves: a whole agent, with all the households it stands for
  (section 3);
- what a household may **read** to decide, and what it may not (section 4);
- the **trigger**: a staggered review, a settling rule, and the household's own
  earned welfare ratio (section 5), with the decisions on relief and on the
  replacement ratio (5.4, 5.5);
- the **destination**: an estimate of the income the household's role would earn
  there from rumoured prices, the journey time, a horizon, a margin and a
  deterministic ranking (section 6);
- **admission**: the guard of REG's relocation, the caps on outflow and on the
  households on the way to one destination, and the provisioning of the journey
  (section 7);
- the **journey**: a timed move whose account stays in the region of departure
  until arrival, with what happens on the way, at departure and at arrival
  (section 8);
- what is created or destroyed (nothing), the pool, and the money-limited
  population (section 9);
- where it happens in the tick (section 10); parameters, validation,
  observability, checkpoint state, worked examples, the model, the register
  entries and the requirements (sections 11 to 19).

Out of scope, with the spec that owns it: the household's welfare ratio, its
window and its observed wage (WEL, `spec-welfare-and-wages.md` 2, 4.3); the
floor of household production outside the market (`spec-subsistence.md`, SUB,
written at the same time: this spec states the seam, 8.5); rumours (INF);
travel by a shipper or a merchant, and the haul units that carry goods (TRN,
`spec-merchants.md`, MER); relief, the pool and the estates (EST); births,
deaths and the replacement ratio (POP); the failure ladder and the occupation
choice (SPN); the public call that moves an external agent (INT, `spec-regions.md`
6.1.5); the tick phases as a whole (`spec-world.md`, WLD); the events as a whole
and the snapshot fields (`spec-observability.md`, OBS). A change of region by a
shock that moves households at once (a flood, an expulsion) is a consumer's call
to INT, not a decision of this module.

## 2. Positions taken

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | What is it? | An optional module, on when `Config.Modules.Migration` is non-nil. A rule of the world, not an extension interface: it has parameters in data and no code of a consumer | ADR 0001 decision 2. A new extension interface would need a row in architecture.md 4.3, a default and an assertion (4.5); a consumer who wants another mover moves its own agents through INT. See open question 1 |
| 2 | What does it need? | The information module (`ErrModuleDependency` otherwise), and the region graph, which is core. It does not need POP or TRN | The pull of a destination is a rumoured price (4.3); the roadmap row says "needs POP, REG, INF", and POP is a signal-sharing neighbour, not a dependency (5.5) |
| 3 | What moves? | A whole agent, with all its households. A cohort moves at once; moving part of one is a split first (`spec-scale.md` 4.8) and version 1 does not split. The first consumer's agents have weight 1 (N-030) | REG 6.1.4. The cost of a cohort is lockstep (L13, L34), so the caps count households and a warning flags a mobile role with a group scale above 1 (3.2, 12.3) |
| 4 | Who is never moved by it? | An external agent (REG 6.1.5, N-022), an agent of a role with `Fixed`, an agent that is away or has an open obligation of a registered refusal, an agent already on a journey | The default module leaves the player and the tied alone; the guard is REG's (7.2) |
| 5 | What makes a household consider moving? | Its own review, on a fixed rotation (no draw); it is settled (it has been where it is for at least the longer of `MoveCooldown` and its own welfare window); and its own **earned ratio** (the welfare ratio without relief) is below `MoveBelow` | A rule on the household's own state (N-110). The rotation and the settling are the dampers D25 and D34 applied to places (5.1, 5.2) |
| 6 | What may it read? | Its own state, the structure of the world (distances, recipes, the yield table, upkeep lists), and rumours of remote prices as INF defines them. No other region's wage, ratio, stock or count of households, and no regional average | N-110: a regional aggregate is a measure, not an input. A price is a market's own output, and it arrives late and blurred (4) |
| 7 | How does it value a destination? | The net value per working tick of the household's best recipe at the rumoured prices of the destination and the table's yield there, divided by the cost of its upkeep at the same prices, in basis points (6.3). A household with no recipe has no estimate | The shape of SPN's `V` (`spec-spawner.md` 6.1) and AGT's recipe choice (`spec-agents.md` 7.1.4), with rumoured prices in place of belief centres and of an observed wage nobody can see |
| 8 | How does distance matter? | Through the journey time `k = max(1, ceil(D / MoveSpeed))` and a **horizon**: the income of a journey of `k` ticks is discounted by `(H - k) / H`, and a destination with `k >= H` is not one | A move costs time and nothing else the ledger can count (position 10); the horizon turns the time into a number of the same unit as the gain |
| 9 | Instant or timed (N-105)? | Timed: a journey in the shape of TRN's trip. The account stays in the region of departure, the agent acts in no market, and the account changes region at arrival by REG's relocation. `NoRegion`, `depart` and `arrive` stay unused | 8.1 gives the reasons: a death, the ladder, relief and the estate all keep working with no new rule; no ledger reason; the cost of distance is real |
| 10 | What does a move cost, and who pays? | The household pays, in time (the ticks of the journey), in the job in progress (abandoned, `region_changed`), and in the upkeep of the journey, which it must hold: a household that cannot feed itself for `k` ticks does not go. No money and no goods are consumed by the move | Money and goods change only by a reason of the registry (`spec-ledger.md` 2); a cost in time and in the household's own stock needs none. N-107: the module is no faucet and no sink |
| 11 | How is an arrival funded (N-097)? | It is not: the migrant carries its account. It receives no stake, draws nothing from the pool of either region, and takes no part of `EntryPoolShare` | A relocation is not a transfer (REG 6.3). Births and entry keep the whole budget (9.2) |
| 12 | Is POP's replacement ratio a signal (N-097)? | No | The trigger is the household's own earned ratio against `MoveBelow`; `r*` is a property of two schedules, and reading it would tie this module to POP (5.5) |
| 13 | Does relief suppress migration (README Still open 4)? | No by construction: relief is excluded from the earned ratio and from the observed wage the trigger reads (5.4). What remains is relief's effect on the destination's pool (L63) | The comparison between home and away is between what work earns, and relief is not earned |
| 14 | What does the module do to a feedback loop? | It closes the migration cobweb and adds a pipeline, a herd and a poverty trap. Seven dampers: a review rotation, the settling rule, a margin with a width, an outflow cap, a cap on the households on the way, the horizon, and provisioning | Section 17. The model of section 16 shows which of them works alone and which does not |
| 15 | What happens to the job, the beliefs, the record? | The job is abandoned at departure; the beliefs are reseeded at arrival (BEL 8.3, N-038); the welfare record stays (WEL 2.1.2) and holds the journey as zero-income ticks; `RoleSince` is set at arrival | 8.5 gives the table |
| 16 | Is the role kept? | Yes. The destination must make the role viable by the estimate (`EntryRatio`); a change of role afterwards is SPN's | One decision at a time, and the spawner already has the damped rule for roles |
| 17 | Per-region differences? | A region-keyed table of the one mechanism of `spec-regions.md` 5.1: `Regions` rows with `OutShare` and `InShare` by replace (N-104) | A region that takes no immigrant, or lets none leave, is a row, not a field of its own |
| 18 | Are silent limits counted (N-045)? | Yes: every household that is pushed ends in exactly one counted outcome (7.6), in the event `MigrationReviewed` | FR-STB-013 |
| 19 | Does it draw random numbers? | Only through INF's rumour error, one draw per candidate of a household that is pushed, in a fixed order (6.2). The spread of margins is a function of the agent ID, with no draw | Determinism (architecture.md 7.3, 7.7); N-111 |
| 20 | Does it take a household's load into account? | No. A migrant carries its whole account, as REG's relocation does. TRN limits a traveller; this module does not | A stated hazard (MIG-A9), and open question 4 |

## 3. The module and the unit that moves

### 3.1 The module

`Config.Modules.Migration` is a pointer: nil is off, non-nil is on
(`spec-config.md` 3.2). Its fields are in section 11: a speed, a review period, a
gate, a margin with a width, a cooldown, a horizon, two caps and a table of
per-region caps. It needs `Config.Modules.Information` (`ErrModuleDependency`
otherwise), and so the region graph. `Role.Fixed` is a bool owned here, default
false; setting it while the module is off is allowed and does nothing.

3.1.1. With the module off nothing here runs, no event of section 13 is emitted,
no state of section 14 exists, no agent has a journey, and the phases of
`spec-agents.md` 4 are unchanged.

3.1.2. With the module on, an agent changes region by this module (the journey of
section 8), by the arrival of a transport trip (`spec-transport.md` 5.7), and by
the consumer's own call of `Relocate` (`spec-integration.md` 4.5.1, `spec-regions.md`
6.6), which moves any agent that is not a pseudo-agent, a household as well as an
external agent. Nothing else changes a region.

### 3.2 The unit: a whole agent

An agent is a household or a cohort of identical households that move together
(`spec-scale.md` 2). A move is the move of an agent: the rule is evaluated once
for the agent, on the per-household ratio and the per-household estimate (neither
depends on the weight, `spec-scale.md` 4.3), and if it fires, every household
leaves. With weight 1, the case of the first consumer, nothing else is possible.

3.2.1. The caps of 7 count **households**, the sum of weights. An agent whose
weight exceeds what is left of a cap is admitted whole when the cap is not yet
reached (the cap is exceeded by at most one agent's weight, as in
`spec-spawner.md` 6.3), so a cohort is never split to fit.

3.2.2. A cohort is a hazard for this module: all its households decide at once,
with one draw set, so it moves in lockstep and the caps cannot divide it. The
warning `migration_cohort` (12.3) names a mobile role with a group whose `Scale`
is above 1. Splitting a cohort to move part of it is `spec-scale.md` 4.8 and 8:
not in version 1 (open question 3).

### 3.3 Who is eligible

An agent is **eligible for review** in the migration pass of a tick when all hold:

1. it is not an external agent (`spec-agents.md` 10) and not a pseudo-agent;
2. its role is not `Fixed`;
3. it is not on a journey (8.3);
4. it is not away on a trip of the transport module (`spec-transport.md` 5.5.2);
5. its turn has come: `(tick + bucket(AgentID, MoveReviewTicks)) mod MoveReviewTicks = 0` (5.1);
6. it is settled (5.2).

An eligible agent is **pushed** when its earned ratio exists and is below
`MoveBelow` (5.3). The registered refusals of REG (7.2) are checked at admission,
not here, because they are state that other specs own and may change inside the
pass.

3.3.1. An agent that is away on a transport trip is in no market (`spec-transport.md`
5.5.2) and is not a migrant: a merchant on a trip stays a merchant on a trip. An
agent that is the consignee of a shipment in flight is refused at admission by
`open_shipment` (7.2). The module never moves a trip's traveller; the transport
module moves it at arrival.

3.3.2. A role with an empty recipe list is eligible and ends in the outcome
`no_estimate` when pushed (6.3.4): version 1 has no estimate of income for a
household that earns nothing from production. A merchant role (`spec-merchants.md`,
MER) normally has none.

## 4. What a household reads

The line drawn here is the answer to N-110 and to the roadmap's "macro is measured,
never set". A household's decision to move may use three kinds of input and no
fourth.

### 4.1 Its own state

Its welfare record (WEL): the earned ratio (5.3) and the **observed wage** per
working household-tick over its own effective window, without relief and grants
(`spec-welfare-and-wages.md` 4.3, evaluated for the agent and not for a role); its
effective upkeep (`spec-agents.md` 5.2) and the stock it holds; its role and the
role's recipes; its weight; and the tick it last took up its place (`RoleSince`,
SPN). All are the household's own.

### 4.2 The structure of the world

What is true of the world by its construction, and does not change with the state
of any market: the region graph, with the edge statuses of the moment (a closed
edge is not a way, `spec-regions.md` 4.3.1); the recipes and their yield factors
by region and season (`spec-production.md` 6.2); the upkeep lists, including the
region's extra upkeep (`spec-agents.md` 5.2); the calendar; and this module's own
parameters. A household can know that a crop does not grow in a place, as it can
know how far the place is. The module reads **the table value** of the yield
(`spec-production.md` 6.3, `site`) and not the shock or the abundance, which are
state of the destination (4.4).

### 4.3 Rumours

The only news of a remote market is a rumour as `spec-information.md` 5 defines
it: the price that market published `Lag` ticks ago, at least 2, blurred by an
independent bounded error, with the market's window mean beside it and the age of
its price. The module reads the **window mean** (`Mean`) of a rumour and no other
field of the market, and uses a rumour only if its mean exists (the market traded
within its price window); a price that is carried forward from long ago is not
news for a decision as costly as this one (6.2).

4.3.1. **Who is shown them.** `spec-information.md` 6.1 shows rumours to the
agents of roles flagged `Informed`, in the view of `Trade`. A household that is
reviewing a move is not in `Trade`, and flagging every household informed would
build a view for every household every tick. So this module asks INF's function
directly, for the agent under review and for the commodities of 6.1, whatever
`Informed` says. The call is the one a view makes (the same lag, the same error,
the same vintage, `spec-information.md` 5.2 to 5.5), the draws are taken in the
order of 6.2, and the event `RumoursServed` does not count it (the counts are
this spec's, 13.1). This is an edit of INF, listed in the handoff.

### 4.4 What it does not read

Another region's wage, welfare ratio, income, stock, endowment, abundance, count
of households, count of migrants or pool; the average of any of these; the price
of a market with no rumour. The estimate of 6.3 is built from prices and
structure, so a destination whose land is exhausted, or crowded, shows nothing of
it until the arrivals show it in prices, late (MIG-A3). The caps of 7 use
the counts of households of the two regions: they are limits the world applies to
the flow, as `EntryShare` and `ExitShare` are, and no estimate, gate or ranking
reads a count.

4.4.1. **Why a price and not a ratio.** The roadmap and INF 17.4 asked whether a
household may react to a regional mean welfare ratio. It may not: a mean ratio is
an indicator, a controller input in disguise, and its delay and error would have to
be invented. A price is a market's own output, which INF already delays and blurs;
an income derived from prices and structure is a household's own calculation, and
it can be wrong (it ignores crowding), which is the realistic failure and the one
the caps and the settling rule are for.

## 5. The trigger

### 5.1 The review

An agent reviews its place on the ticks where

```
(tick + bucket(AgentID, MoveReviewTicks)) mod MoveReviewTicks = 0
```

with `bucket(id, P) = (Weyl2(id) * P) >> 32` of `spec-world.md` 9.1.1 (ruling C14), a
fixed rotation with no random draw, so reviews are spread over the period. It is the
decision rate of D25 (migration and hiring decisions at slower stated rates) applied
to places, and the same device as the occupation review of `spec-spawner.md` 6.2.
The review runs in the migration pass of phase 7 (10), after consumption, so it sees
the tick just ended; its rumours are read as of tick `t + 1` (ruling C17,
`spec-world.md` P7.2).

5.1.1. **What the rotation does and does not do.** In the model (16.2) a period
alone does not stop the cycle: with a period of 30 or 60 ticks the whole region is
still reviewed once per period on the same lagged news. It does make a cycle slower
and fewer movers decide on the same tick. It is a damper only with the others
(MIG-D1).

### 5.2 Settling

An agent is **settled** at tick `t` when

```
t - RoleSince >= max( MoveCooldown, T_a )
```

with `T_a` the effective welfare window of its role (`spec-welfare-and-wages.md`
2.3.3, derived and never shorter than the longest job). `RoleSince` is the tick the
agent took up its present role or place: SPN sets it at creation and at a change of
role (`spec-spawner.md` 4.2, 7.4), this module sets it at **arrival**, and a forced
`Relocate` of INT sets it at the move (`spec-integration.md` 4.5.2 step 5). An agent
created in a run, a role changer and a migrant are therefore all left alone for at
least a window.

5.2.1. **Why the window.** The welfare record travels with the agent
(`spec-welfare-and-wages.md` 2.1.2), so for `T_a` ticks after a move its ratio is
made partly of the place it left and, after a journey, of `k` ticks with no income
(8.6). A trigger that read that record would see a household that is poor because
it has just moved and send it on. The window is the time the record needs to
forget. It is derived, like the window itself (D29), and `MoveCooldown` can only
lengthen it. In the model (16.2) a household free to move again at once keeps
crossing between two equal regions (239 reversals against 99 with the rule alone
and 2 with the width as well).

5.2.2. A created agent is settled after a window (its ratio is padded until then,
`spec-welfare-and-wages.md` 2.2). An initial agent of a world with a warm start
may carry a negative `RoleSince` (`spec-spawner.md` 14); it is settled at once.

5.2.3. **Counted.** The settling rule acts silently, so it is counted (N-045). An agent
whose turn comes and that meets items 1 to 4 of 3.3 but is not settled is tested against
the gate of 5.3 for the count only: if its earned ratio is below `MoveBelow` its
households are added to `Unsettled` of `MigrationReviewed` (13.1), and nothing else is
computed for it.

### 5.3 The push: the earned ratio

For an agent whose welfare ratio exists (`spec-welfare-and-wages.md` 2.2.3), the
**earned ratio** is the ratio of 2.1 of that spec with the relief of the window
taken out of the income:

```
earned_bp = ( sum over the window of (income_t - F_t) ) * 10 000 / ( sum over the window of cost_t )
```

with `F_t` the relief paid to the agent in tick `t` (`spec-estates.md` 2.6), and
everything else as WEL defines it: the same window, the same padding, the same
truncation. A grant to households (`G_t`) stays in `income_t`: it is the standing
income of the household's class. Whether it follows the household depends on the flow's
target (a class tied to a region does not, `spec-money-supply.md` 4.3.1), and the
estimate of 6.3 does not weigh it (open question 6). The agent is **pushed** when

```
earned_bp < MoveBelow
```

The ratio can be negative (a window of net losses) and is then below any
non-negative `MoveBelow`. If the ratio is absent, because the agent has no upkeep,
it is not pushed.

5.3.1. **Why a gate at all.** Without it the rule would be a labour-market
arbitrage at every wage: any household would move for any gain beyond the margin,
and the flow would equalise incomes above subsistence. That is a legitimate
behaviour and a pack that wants it sets `MoveBelow` high (up to `MaxBasisPoints`).
The default, 10 000 (a ratio of 1.0: the household's earnings do not cover its
upkeep), is the distress migration the roadmap row names, and it keeps the number
of households that evaluate a destination small. It is a placeholder.

5.3.2. **Independence of the target.** The gate is absolute, against subsistence,
and not against the household's own target ratio (`spec-welfare-and-wages.md`
5.1): a household that is below its aspiration and above its upkeep does not move
for that alone.

### 5.4 Relief (README Still open 4, STB-L40)

The question was whether relief suppresses migration because it raises the ratio
that triggers it. Decided: **no, by construction**. The trigger reads the earned
ratio (5.3), which has no relief in it, and the pull reads the observed wage, which
has none either (`spec-welfare-and-wages.md` 4.3). The household compares what its
work earns here with what its work would earn there; relief is neither.

5.4.1. What relief still does. It keeps a household fed in the place it is poor, so
the household is under the ladder's rung 2 and not rung 3 for longer
(`spec-spawner.md` 5.2); it does not change the decision to go. It does have a
cost to the place the household goes to: an arriving household whose record is
poor (and one that has just spent `k` ticks with no income, 8.6) draws relief from
the destination's pool if its ratio is below `ReliefBelow`. The pool serves a third
master then (L63, MIG-A4): the cap on the households on the way (7.5) bounds it.

5.4.2. The check is scenario MIG-S3: the same two-region run with relief on and
off must show the same outflow within a band that STY sets (17).

5.4.3. The edit that makes this possible is WEL's: the record must carry the relief
of each tick (or a running sum of it), which the observed wage of 4.3 already
needs (it takes `F_t` out of the income). The handoff lists it.

### 5.5 POP's replacement ratio (N-097)

POP's **replacement ratio** `r*` is the ratio at which births equal deaths
(`spec-population.md` 5.7). N-097 asked whether it is part of this module's signal.
Decided: **no**. Reasons: the signal is the household's own earned ratio against a
gate in this module's data; `r*` is a property of two schedules and exists only
when the population module is on; and a gate derived from it would make
the module depend on POP for no gain. A pack with both modules may set
`MoveBelow` near `r*` by hand, and the informational line of POP that reports
`r*` at validation is where it finds the number. The module does not read the
births and deaths of a region either: they are counts.

## 6. The destination

For each pushed agent, the module builds a ranked list of destinations. The list
is built from the agent's own state, the structure of the world and rumours (4),
with integer arithmetic and a total order.

### 6.1 Candidates and the commodities they need

A **candidate** is a region `d` other than the agent's region `h`, that is not an
outside node (`spec-external-sector.md` 3.1: a relocation into one is refused,
`ErrOutsideNode`), and for which REG has a way: `Distance(h, d)` is defined over the
open edges at the moment (`spec-regions.md` 4.3.1). A region with no agents is a
candidate as far as REG is concerned, and drops out at 6.2 if its markets give no
rumour.

The commodities the estimate needs are the set `K(role)`: the upkeep commodities
of the role in any region (the role's or the world's list, and every region's extra
upkeep, `spec-agents.md` 5.2), and, for every recipe on the role's list, its inputs,
its required tools and its non-discard outputs. It is a function of the role and
the configuration, in ascending commodity order. Validation bounds it (12.2).

### 6.2 Rumours and their draws

For each candidate `d` in ascending `RegionID` and each commodity `c` in `K(role)`
in ascending order such that `(d, c)` is a **source** of INF (`spec-information.md`
4.1), the module asks INF for the rumour of `(d, c)` as seen from `h`
(`spec-information.md` 5.3 to 5.5): the first of these applies and decides

| # | Case | Result |
|---|------|--------|
| 1 | no walk from `d` to `h` over open edges | none (`NoPath`) |
| 2 | `Lag(d, h)` is above `RumourMaxAge` | none (`TooOld`) |
| 3 | `Lag(d, h)` is above `t + 1`, the tick the pass reads as of (ruling C17, `spec-world.md` P7.2) | none (`NotYet`) |
| 4 | otherwise | a rumour |

and a **price** `p_c(d)` is taken from the rumour: its `Mean` if it is above 0, else
none (the market did not trade within its price window, so its price is old and
carried forward). A candidate is dropped at stage 1 (`no_rumour`, 6.5, 7.6) when any
upkeep commodity of the household in `d` has no price, or when no recipe of its role has
a price for every commodity in it (its inputs, required tools and non-discard outputs).

6.2.1. **Draws.** They are taken in the migration pass of phase 7 (10.2 step 2), among the reviewed households in ascending agent ID, one per candidate, and never in the view of phase 3: INF's view draws precede `Trade` in phase 3, these follow the entry step in phase 7, so the two streams cannot interleave. When `RumourErrorBP` is above 0, one draw is taken for every
`(d, c)` pair that is a source, in the order above, whether or not the case is 4,
exactly as for a view (`spec-information.md` 5.5.2, 5.3.2). The agents are visited in
ascending ID. The number of draws in a tick is therefore a function of the
configuration and of which agents are pushed (N-111): a world with no pushed
agent, or with `RumourErrorBP` 0, takes none, and the stream is that of a world
without the module.

6.2.2. The direction is the news's: a rumour about `d` reaches `h` along the walks
from `d` to `h`, and the journey goes the other way (`h` to `d`). The two
distances differ in general (`spec-regions.md` 4.1.4): in the river valley of 15.1 the
news of `lower` takes 4 ticks to reach `upper` and the journey from `upper` to
`lower` takes 4 as well, by coincidence of the numbers.

### 6.3 The estimate

For a candidate `d` and every recipe `r` on the role's list for which every
commodity of `r` (inputs, tools and non-discard outputs) has a price in `d`, let

```
S       = the table value of the yield of (d, r) in the season of the arrival tick  (spec-production.md 6.2)
value_r = sum over non-discard outputs o:  floor( q_o * S * p_o(d) / 10 000 )
cost_r  = sum over inputs i:  q_i * p_i(d)  +  sum over tools t:  ceil( WearPerUse_t * p_t(d) / Durability_t )
net_r   = value_r - cost_r                                  per execution
```

the formulas of `spec-agents.md` 7.1.4 with the rumoured price in place of the
belief centre, and `S` in place of the effective yield. The season of the arrival
tick is that of `t + 1 + k` (8.2), `Calendar.At(t + 1 + k).Season`, or -1 when the
calendar has none. A recipe is **eligible** at `d` if `S` is above 0 and either
`net_r > 0` or it is a source (no inputs). The household's **rumoured income per
working tick** at `d` is the greatest of `floor(net_r / Duration_r)` over the
eligible recipes, and 0 if there is none. The household's upkeep at `d` is the
effective upkeep of its role in `d` (the role's or the world's list and the
region's extra upkeep, `spec-agents.md` 5.2), per household, and its cost is

```
U_d = sum over upkeep commodities u:  q_u(d) * p_u(d)
```

and the **estimate** is

```
Vhat_d = rumoured income per working tick * 10 000 / U_d             basis points
```

computed with the checked 128-bit helper and truncated toward zero (architecture.md
3.5). It is the ratio the household would reach if it worked every tick: the same
shape as `V` of `spec-spawner.md` 6.1, `Wage * 10 000 / Upkeep`, with the wage
estimated from prices. Roles with different upkeep are compared by ratio and never
by money (`spec-currency.md` 5.6). A weight does not enter.

6.3.1. **Why a recipe's net value.** In this library a household earns by making
and selling (`spec-agents.md` 3.3); its wage is the net value of what it makes
divided by the ticks it works (`spec-welfare-and-wages.md` 4.3). The estimate
is that wage at the destination's prices, which is what a household that knew
only prices could compute. It does not include labour in the cost (the household's
own labour is what is earning), it does not include the cost of a tool it would
have to buy there beyond the wear (the household carries its tools), and it does
not include a fixed factor's limit (4.4).

6.3.2. **The home side is measured, not estimated.** The household's own income per
working tick is its observed wage over its window, without relief and grants
(`spec-welfare-and-wages.md` 4.3, for the agent), and its home figure is

```
V_a = observed wage * 10 000 / U_h
```

with `U_h` the cost of its effective upkeep in its own region at the local market
prices of tick `t`, the tick just ended (the window already holds the tick's push; the
price window mean if the window is not empty, else the market price;
`spec-clearing.md` 5.4), per household. If the agent had no
working household-tick in its window, `V_a` is 0. The local prices are free and
exact (`spec-information.md` 3.1), the rumoured ones are late and blurred, and the
comparison is between a measurement and an estimate: that is a feature (a household
trusts what it earned) and the reason for the margin (6.5).

6.3.3. **Why not the recipe at home too.** Using the same estimate for home and away
would compare like with like, and it would be blind to what a household cannot
see in prices and structure: a fixed factor that is cut by a ration, an exhausted
site, a closed market. Those are the reasons a household is poor, and they show in
its own measured wage. Home is measured for that reason.

6.3.4. **No recipe, no estimate.** A pushed agent whose role has no recipes has no
income to estimate, and ends in the outcome `no_estimate` (7.6). A household
that lives on a grant or on relief in version 1 does not move by this rule (open
question 6). A pushed agent whose recipes are all ineligible or priceless at every
candidate is not `no_estimate`: its candidates fall at the stages of 6.5.

### 6.4 The journey time and the horizon

The journey time from `h` to `d`, in ticks, is

```
k = max( 1, ceil( D / MoveSpeed ) )              D = Distance(h, d), REG's shortest over open edges
```

with an integer ceiling, as TRN's travel time with the transit
factor at its default (`spec-transport.md` 4.1). The **horizon** `H = MoveHorizon`
is the number of ticks over which the household values the move. A candidate with
`k >= H` is dropped (`too_far`, 6.5). Otherwise the estimate is discounted by the
share of the horizon that is left after the journey:

```
V_eff,d = Vhat_d * (H - k) / H                      truncated toward zero
```

The discount is the opportunity cost of the journey in the unit of the gain: a
household that works every tick of the horizon after arriving earns `H - k` ticks'
worth of `H`. It is also what makes distance matter when no money is charged
(position 10), and it makes the effect of a long journey asymmetric when the
distances are (15.2). A pack that wants no discount sets `MoveHorizon` to its
maximum (the discount is then below one part in `MaxTravelTicks`).

### 6.5 The gates, the margin and the ranking

Each candidate is tested in this order; the first test it
fails is its stage, and the household's outcome when no candidate survives is the
latest stage any candidate reached (7.6):

| Stage | Test | Fails with |
|-------|------|------------|
| 1 | the household's upkeep in `d` is fully priced and at least one recipe of its role is (6.2) | `no_rumour` |
| 2 | `k < H` | `too_far` |
| 3 | `Vhat_d >= EntryRatio` (the viability gate of `spec-spawner.md` 6.1, read, not owned) | `no_viable` |
| 4 | `V_eff,d * 10 000 >= V_a * (10 000 + g_a)` | `no_gain` |

where `g_a` is the agent's own **margin**

```
g_a = MoveGain + floor( u_a * MoveWidth )            u_a = Offset(AgentID) / 2^32, in [0, 1)
Offset(id) = ( id * 2 654 435 769 ) mod 2^32         POP 3.8.3: a Weyl sequence, a function of the ID, no draw
```

computed as `(Offset * MoveWidth) / 2^32` with the 128-bit helper. The margin is
the relative improvement the household needs to see: at least `MoveGain`, at most
`MoveGain + MoveWidth`, spread over the agents by their IDs. `MoveGain` is the
hysteresis band of D09; `MoveWidth` disperses the thresholds of the households of a
region, which is the "fixed spread of the agents' own thresholds" that N-106 and
INF 13 recommend for any agent that acts on one common signal. The comparison is by
cross-multiplication, not by division.

6.5.1. **The ranking.** The candidates that pass stage 4 are ranked by `V_eff,d`
descending and, for equal values, by `RegionID` ascending: a total order that
depends on the estimate and on the region IDs and on nothing else. The household's
list is that ranking, at most one entry per region.

6.5.2. **A destination is not a promise.** The estimate is a rumour of old prices.
Between the review and the arrival `Lag + 1 + k` ticks pass, and the destination's
prices change, partly because of the arrivals the estimate cannot see (MIG-A3). The
module does not re-test on arrival: a migrant that finds the place worse has moved,
is settled for a window, and may then move again, which is a decision it reaches by
its own measured wage.

6.5.3. **The flat-world case.** Two regions with the same structure and the same
prices give `V_eff,d < V_a` after the discount for any `H` finite, so nothing moves
unless the household's measured wage is below the estimate by more than the
discount and the margin. The model confirms it (the equal regions of 16.5 start
unbalanced, and stop moving inside the band).

## 7. Admission

The ranked households are admitted or not, in one order, against the guard, the caps
and the provisioning.

### 7.1 The order

The pushed agents that have a non-empty ranking are ordered by earned ratio
ascending (the poorest first) and, for equal ratios, by agent ID ascending. They are
considered in that order. This is the order of need, as `spec-spawner.md` 6.3 orders
its candidates, and it is a total order. The evaluations of 6 (and their draws) were
made earlier, in ascending agent ID (10.2).

### 7.2 The guard

The household is first tested against the refusals of `spec-regions.md` 6.1.3 by the
read-only query `World.RelocationRefusal` (7.2.1), in that order, with the destination at
the head of its ranking: a `region_bound` refusal that a later spec registered
(`open_shipment` for an agent that is the shipper or the consignee of a shipment in
flight or is away, `in_traineeship`, `open_contract`) ends the consideration with the
outcome `bound`. An agent already on a journey is not in the pass.
The other refusals cannot occur for a live, open agent account in a region, with
no reservation in phase 7 and a destination that is a region other than its own
(6.1); if one does, the world has a defect and it panics with the invariant
(architecture.md 8.1). The same refusals are tested again by the real relocation at
arrival (8.4), because an obligation can open on the journey.

7.2.1. **The query.** The refusals of REG 6.1.3 are asked, with no change of state, by

```go
// RelocationRefusal returns the reason (an exported constant of the root package) and the error of
// the first refusal of spec-regions.md 6.1.3 that would stop relocate(a, to, ...), or
// ("", nil) if none would. Registered region_bound obligations named in skip, by name or
// by instance key, are not tested. It changes nothing, draws nothing and allocates nothing.
func (w *World) RelocationRefusal(a AgentID, to RegionID, skip ...string) (string, error)
```

The guard calls it with no `skip`. The arrival of 8.4 calls it with `skip` = `on_journey`,
by name: an agent has at most one journey, so the name is enough (`spec-regions.md` 6.1.6).

7.2.2. **The refusal `on_journey`.** With the module on, this module registers the
`region_bound` refusal `on_journey` (`spec-regions.md` 6.5.2), open for an agent that has a
journey, as TRN registers `open_shipment` (`spec-transport.md` 5.5.3). One caller skips
it on purpose: the public `Relocate` of INT (`spec-integration.md` 4.5.2) runs the dry run
with `skip` = `on_journey`, cancels the journey (8.4.1) and moves the agent, in one atomic
call. Any other path of the guarded relocation returns `ErrRegionBound` naming `on_journey`
and changes nothing, so the account stays in the region of departure for the whole journey.
This module itself changes the region only at arrival, through the ledger-level `Relocate`
(8.4), as TRN does at the arrival of a trip.
### 7.3 The outflow cap

In a region `h`, at most

```
OutCap_h = ceil( Households_h * OutShare_h / 10 000 )
```

households start a journey in one tick, with `Households_h` the sum of the weights of
the agents whose account is in `h` at the start of the pass (those on a journey
included: their account is still there) and `OutShare_h` the `MoveOutShare` of the
world or the `Regions` row of `h`. An agent that makes the count reach the cap is
admitted; one considered after it is refused (`out_cap`). `OutShare` 0 means nobody
leaves `h`. This is the entry limit of D14 turned the other way, and the damper of
MIG-D4.

### 7.4 Provisioning

A household cannot buy on the journey, as a traveller cannot (`spec-transport.md` 5.4,
`unprovisioned`). For the candidate under consideration, with `k` its journey time,
the agent must hold, in its account at the end of phase 6 of the tick, at least

```
w * u_c * k            of each upkeep commodity c of its effective upkeep in its present region
```

(`spec-agents.md` 5.2: the account is in `h` for the whole journey, so the extra
upkeep of `h` applies and the consumption of phase 6 takes from the account as
usual). If it does not, the candidate is skipped (`unprovisioned` if no candidate is
left that passes). The test uses the stock now; goods that spoil before they are
eaten are not predicted.

7.4.1. **What it costs, and who it falls on.** The poorest households hold the
least, so the poor can go only where `k` is small: a pack with long journeys lets
the poor stay (MIG-A7). This is the price of charging the journey in the household's
own stock and not in a faucet or a sink, and it is a bound in the sense of D01 more
than a damper. A default household holds about `CoverTicks` of upkeep
(`spec-agents.md` 7.3); the warning `migration_provision` (12.3) names an edge
whose journey is longer than that.

### 7.5 The pipeline cap

For each destination `d`, the households on a journey to `d` (those already on the
way and those admitted earlier in this pass) must not exceed

```
InCap_d = ceil( Households_d * InShare_d / 10 000 )
```

with `Households_d` as above and `InShare_d` the `MoveInShare` of the world or the
row of `d`. A candidate whose destination is full is skipped (`in_cap` if no
candidate is left that passes). The household that makes the count reach the cap
is admitted. `InShare` 0 means nobody may enter `d` by this module. It counts the
**pipeline** and not the flow: an arrival that has landed leaves the count, and the
destination's own prices then show it (late).

7.5.1. **Why the stock on the way and not the flow per tick.** The movers on the way
are invisible to the next review: their effect on the destination's prices comes
after the journey and the rumour's lag, so a review that cannot see them sends more
(MIG-A3, the analogue of L85). The cap is the module's `Inbound`: a bound on what
can be in the pipeline, so the swing is bounded by the cap and the delay is not
amplified. In the model (16.3) a journey of 8 or 20 ticks without the cap swings the
region to 258 and 91 of an equilibrium of 500; with the cap (50 and 100 basis
points) to 505 and 508.

### 7.6 Every pushed household ends in one outcome

The outcomes partition the pushed households (weights), so a silent clamp cannot
hide:

| Outcome | Constant | The household ended there when |
|---------|----------|--------------------------------|
| started | `started` | it was admitted and its journey began |
| no estimate | `no_estimate` | its role has no recipe (6.3.4) |
| no destination | `no_destination` | no candidate region exists (REG has no way to any other region, or all are outside nodes) |
| no news | `no_rumour` | every candidate failed stage 1 |
| too far | `too_far` | the latest stage any candidate reached was 2 |
| not viable | `no_viable` | ... was 3 |
| no gain | `no_gain` | ... was 4 and none passed it |
| bound | `bound` | the guard refused (7.2) |
| out cap | `out_cap` | the outflow cap refused it (7.3) |
| unprovisioned | `unprovisioned` | every candidate that passed the gates failed 7.4 |
| in cap | `in_cap` | at least one passed 7.4 and every such candidate failed 7.5 |

The order of the tests at admission is the guard (7.2), the outflow cap (7.3), then
for each candidate in its ranking the provisioning (7.4) and the pipeline cap
(7.5); the first candidate that passes both is the destination. A fixed, documented
order, so that the counts are the same on every run. `held` (an arrival waiting for
an obligation, 8.4) is counted besides, and `ended` (journeys ended by a removal,
8.3) too: they are not outcomes of a push.

## 8. The journey

### 8.1 Instant or timed (N-105)

N-105 left the choice to this spec: an instant `relocate` (REG 6), a journey in the
shape of TRN's traveller, or REG's reserved `NoRegion` with `depart` and `arrive`
(`spec-regions.md` 6.5.3). Decided: **a timed journey in the shape of the traveller;
`NoRegion` is not used.**

| | Instant | Timed, account stays in the origin (chosen) | Timed with `NoRegion` |
|---|---------|---------------------------------------------|-----------------------|
| Cost of distance | none unless a charge is invented: a sink for the journey's upkeep, a new ledger reason, and no loss of time | the ticks of the journey: no income, the upkeep from the household's own stock | the same ticks |
| New ledger reason | one (the sink), or none and no cost of distance | none: the upkeep is `consumption`, as every tick | none |
| A death on the journey | none | the household is in its origin: its holdings go to the origin's estate (`spec-estates.md` 2.2, `spec-transport.md` 5.9.2) | needs a rule: the money goes to the region departed from (`spec-regions.md` 6.5.3 b) |
| The ladder, relief, POP | act as for any agent | act as for any agent, in the origin | an account in no region is in no estate, no pool and no list of a region: each of those needs a rule (6.5.3 c) |
| Flows | attributed to the region | attributed to the origin until arrival, as a transport trip's are (`spec-ledger.md` 3.5) | attributed to no region |
| State | none | one record per migrant | one record per migrant and a changed account |
| Precedent | REG | TRN 5.5 | reserved, unused |

The timed journey is the only one that makes distance cost something without a
new reason in the registry (position 10), and the one with a tested shape. It
keeps every other rule of the library working on the migrant unchanged: the
account is in one region at all times. Its price is a record and the rules of 8.3
for what the agent does on the way.

### 8.2 Departure

Departure is the last step of the migration pass of phase 7 for an admitted agent
(10.2). In this order, atomically (each step was checked; a failure is a defect and
panics, architecture.md 8.1):

1. **The job.** If the agent has a job in progress it is abandoned, creating
   nothing, with the reason `region_changed` (`spec-production.md` 5.4; the event
   `RecipeAbandoned`). The inputs it consumed at the start are lost
   (`spec-production.md` 5.4.1). That is a cost of the move that the household does not
   weigh, because its estimate counts no sunk input; a role with a long job has a
   longer window (`spec-welfare-and-wages.md` 2.3.3), which delays its first review.
2. **The floor.** The flag and the counter that the subsistence module keeps for the
   agent are dropped (`OwnUse` false, `OwnUseJobs` 0, `spec-subsistence.md` 5.8). SUB has
   no reason and no event for the drop, and an own-use job in progress was abandoned by
   step 1 like any job. A household on a journey does no work.
3. **The record.** The agent's `Journey` is set to `(To, Arrives)` with
   `Arrives = tick + 1 + k`: the journey takes the `k` ticks after the tick of
   departure, and the agent is in the destination from the first phase of tick
   `tick + 1 + k`. The arrival tick is fixed now and does not move if the graph
   changes later (as TRN 4.3).
4. **The event** `MigrationStarted` (13.1) is emitted at step 5 of the pass (10.2), after
   the last departure, in the order of admission. The `RecipeAbandoned` of step 1 is
   emitted per departure, in admission order, before it (`spec-world.md` P7.2).

A departure at the end of tick `t` therefore costs `k` ticks in which the household
makes nothing and trades nothing, and at least one (`k >= 1`).

### 8.3 On the journey

From the first phase of tick `t + 1` until the arrival, the agent is **on a
journey**: its account keeps the region of departure, and

| What | Rule |
|------|------|
| `Work`, `Trade`, `Revise`, orders | none: it is passed to none of them and places no order, as an agent that is away (`spec-transport.md` 5.5.2). It is in no market |
| Upkeep | consumed in phase 6 as in any tick, from its own account, with the effective upkeep of the origin region (`spec-agents.md` 5.2, 5.3); the provisioning of 7.4 made sure it can |
| Welfare record | pushed each tick as usual: no income from trade or work (relief apart: the earned ratio leaves it out), cost the upkeep at the origin's prices (`spec-welfare-and-wages.md` 2.1.1). The ticks are not working ticks (`spec-welfare-and-wages.md` 4.3: "migration travel ... [is] not [a] working tick") |
| Relief | the origin's pool pays relief to it if its ratio is below `ReliefBelow`, as to any agent of the region (its account is there, `spec-transport.md` 5.9.4). Phase 1 pays relief only; starting money moves at creation (`spec-world.md` ruling C1) |
| The ladder (SPN) | acts as on any agent: privation counts, and rung 3 may reduce or remove it. A role change or an application for a traineeship is **refused** while it is on a journey (`on_journey`, `spec-spawner.md` 4), and the placement of an entrant never chooses it (`spec-spawner.md` 4.3) |
| POP's deaths | act on its fed households as on any agent's |
| A removal | the holdings go to the origin's estate, the journey is dropped, and the removal event is the usual `AgentBankrupted` (`spec-agents.md` 3.6). The households are counted in `Ended` of the next `MigrationReviewed` (13.1). A cancel by INT's `Relocate` (8.4.1) is counted in `Ended` in the same way |
| A weight reduction | the journey goes on with the smaller weight (as a trip, `spec-transport.md` 5.9.4) |
| An edge that closes | the journey does not change: its arrival tick was fixed at departure (8.2). Rumours about the origin from the destination's side are INF's |
| A consignee | it may be the consignee of a shipment that arrives in the origin while it is on the way, and the goods are delivered to its account |

8.3.1. **Death on the journey, the ladder and the window (N-099, N-105).** A household
that starves on the way is removed at its origin by rung 3 after `RemovePrivation`
ticks of privation, like any other, and its holdings pass to the origin's pool: the
money is not lost, and the journey is not a place where anything is destroyed. The
welfare window sees the journey as `k` ticks of cost with no income. The effect is
bounded by `k / T_a` of the window and ends when those ticks leave it, which is what
the settling rule of 5.2 waits for; WEL may choose to exclude the ticks of a
journey from the window as it considers a long trip (open question 8).

### 8.4 Arrival

In the first phase of each tick, after the arrivals of the transport module
(`spec-transport.md` 5.7) and before relief (phase 1 pays relief only, `spec-world.md`
ruling C1), for each agent whose `Journey.Arrives` is the tick or earlier, in ascending
agent ID:

1. **The dry run.** The world asks `RelocationRefusal(agent, To, "on_journey")` (7.2.1):
   every refusal of REG 6.1.3 except this module's own, including any obligation opened
   on the way. If none holds, it calls the ledger-level `Relocate(account, To)` with the
   cause `migration`, not the public path (which `on_journey` would block). REG changes
   the account's region and emits `AgentRelocated` (`spec-regions.md`
   6.2, 10.1) with the cause `migration`, and the relocation totals take what the
   account holds.
2. **If a refusal holds** (`region_bound`: an obligation opened on the way: the
   agent became the consignee of a shipment that has not yet arrived, or a later
   spec's obligation) the arrival is **held**: the journey record stays, the agent
   stays on the journey, and the test is made again in the next tick. Releasing it
   against the refusal would make a delivery to its consignee land in a region the
   consignee has left (`spec-transport.md` 5.8: diverted). The count is `held` in
   `MigrationReviewed` (13.1), and a bounded `Warn` says a region has held arrivals
   for ten ticks (13.2). A hold lasts at most as long as the obligation, which
   TRN bounds by `MaxTravelTicks`.
3. **On success, the migrant's state** changes as 8.5 says: the journey record is
   cleared; the beliefs are reseeded (BEL 8.3: for every commodity of the agent's
   tradable set the centre becomes the clamp of the window mean of the destination's
   market, or its price if the window is empty, with the width `BeliefWidthInit`);
   `RoleSince` becomes the tick; and the agent is passed to `Work` and `Trade` from
   phase 2 of this tick.

The guard of 7.2 runs the same tests, so a refusal at arrival is rare.

8.4.1. **Cancelling a journey.** The module has an internal operation `cancelJourney(agent)`: it
drops the journey record, changes nothing in the ledger (the account never left the origin),
counts the households in `Ended` of the next `MigrationReviewed`, and leaves the abandoned job
abandoned. It has no public name. The cancel is a step inside INT's `Relocate`
(`spec-integration.md` 4.5.2): the dry run with `skip` = `on_journey`, then the cancel, then
the move, atomically, so a forced move of a migrating household is one call and a refusal by
another obligation leaves the journey as it was (question 16 is answered).

### 8.5 What moves, and what does not

The table of `spec-regions.md` 6.3, completed for this module. The first rows are
facts of REG, repeated so that a reader of this spec finds the whole.

| State | What happens | When |
|-------|--------------|------|
| Money and goods, tools, lots | stay in the account and travel with it: no transfer, no reason, no flow; the relocation totals record them (`spec-regions.md` 6.4) | at arrival (the account changes region) |
| Role, weight, strategy | unchanged | |
| Job in progress | abandoned, `region_changed` | at departure |
| The subsistence floor (SUB) | the flag `OwnUse` and the counter `OwnUseJobs` are dropped (`spec-subsistence.md` 5.8), with no reason and no event of SUB; the household is evaluated afresh at the first phase 2 after the arrival, from its stock, money and the new region's prices | at departure |
| Price beliefs | reseeded (default of N-038 made a rule): for every commodity of the tradable set, centre `clamp(window mean of the new market, Floor, Ceiling)` (the market price if the window is empty) and width `BeliefWidthInit`, as for any newcomer (`spec-beliefs.md` 8.3) | at arrival |
| Welfare record | kept (`spec-welfare-and-wages.md` 2.1.2), with the journey as zero-income ticks | |
| Target ratio, residues, want carries, `DeathCarry` | unchanged: per household or per unit | |
| `Hunger`, `Privation` | unchanged. The grace that SPN gives a role changer applies: `RoleSince` is the arrival tick, so rung 3 waits `RemovePrivation` ticks (`spec-spawner.md` 5.2) | |
| `RoleSince` | the arrival tick; also the tick of a forced move by INT's `Relocate` (`spec-integration.md` 4.5.2 step 5) | at arrival, or at the forced move |
| Traineeship | a trainer with trainees, and a trainee, are refused by the guard (`in_traineeship`, SKL) | at admission |
| Rumours | none are kept; the agent's rumours are those of its new region from its next view (`spec-information.md` 7.3) | |
| Cost anchors, effective upkeep, effective reference price | derived from the region each tick and change at the next computation (`spec-regions.md` 6.3) | |

8.5.1. **The seam with the subsistence module.** SUB's activity lowers the pressure
that triggers a move by the only route there is: what an own-use job makes is counted
as an in-kind receipt in the household's income (`spec-subsistence.md` 7), so the
earned ratio of 5.3 rises with it and the household is not pushed. The same receipts
and the ticks of the job are excluded from the observed wage that the pull reads (6.3.2),
which stays the wage of market work. This module reads WEL's figures and nothing of
SUB's state: SUB exposes `OwnUse` and `OwnUseJobs` in the view and a rule of this module
MAY read them; version 1 does not. What SUB keeps for a household is dropped at
departure and starts afresh after the arrival, as the table says.

### 8.6 The ticks of a journey and the welfare window

Stated once, for the register (MIG-A6): a journey of `k` ticks puts `k` records with
no earned income and the origin's upkeep into the record. The earned ratio of an agent
that has just arrived is lower by about `k / T_a` of its level. It is below the gate
for as long as those records are in the window, and the settling rule keeps the
agent from acting on it. The relief of the destination can be drawn on those
records (5.4.1). That is the cost of recording what happened.

## 9. Money and goods: nothing is created, nothing is destroyed

9.1. **Conservation.** A move is a relocation of an account (`spec-regions.md` 6.3):
no transfer, no reason, no boundary account, no flow. The total of money and of each
commodity over all accounts is unchanged by a departure, a journey and an arrival
(FR-MIG-033). The upkeep eaten on the journey is `consumption`, the library's own
sink for a household's upkeep, in the tick it is eaten. This module adds no ledger
reason and no faucet or sink (N-107, N-021).

9.2. **The pool (N-097).** Neither region's pool is touched by a move. The migrant
brings its account, so it needs no starting money and no stake. `EntryPoolShare`
(`spec-spawner.md` 7.3) is not shared: births and entry keep the whole budget. The
interplay is indirect, and stated so that nobody looks for a direct one:

- the money a migrant carries leaves the origin's households and enters the
  destination's (REG-A3, L68), and changes `M` of each region by that amount;
- a poor arrival draws relief from the destination's pool (5.4.1, L63);
- a migrant that dies in its origin during the journey leaves its money to the
  origin's pool (8.3.1);
- births in the destination are funded from its pool at POP's stake as before.

9.3. **The money-limited population (POP 5.5, L76).** In a closed region the
population is bounded by `M / stake`, with `M` the money of the households and the
pool. The bound moves with the money of the migrants: it falls where they leave and
rises where they arrive, by what they carry, divided by the stake. The households
pushed to move are the poor, so what they carry is less than a mean household's: an
inflow of poor households into a region lowers its money per household and, with
POP on, the funded births per death (L76). This is the loop POP's section 6.5
promised to MIG ("a migrant arriving with money lifts all three") stated with its
sign: lifts the bound by what it carries, and dilutes by the household it is.

9.4. **A migrant carries its whole account.** TRN prices the carriage of a
traveller's goods against a capacity (`spec-transport.md` 3.2.2: a self-carried
traveller carries everything its account holds, counted against the capacity). A
relocation has no capacity. A household's holdings are small and bounded by the
buffer rules of the default strategy (`spec-agents.md` 7.3), so the leak is small;
a consumer's strategy that stockpiles to ride a journey for free would bypass
carriage. It is a stated bias (MIG-A9) and an open question (4).

## 10. In the tick

### 10.1 Where it happens

| Phase | What this module does |
|-------|-----------------------|
| 1, start | after the arrivals of the transport module and before relief (phase 1 pays relief only): the arrivals of 8.4, in ascending agent ID |
| 2, 3 | none; an agent on a journey is passed to neither `Work` nor `Trade` (AGT 4, as an away agent) |
| 6, consumption | none; the upkeep of a migrant is consumed from its account and its record is pushed as for any agent |
| 7, entry and exit | after the entry step (the spawner and the population module) of **every** region has run: the migration pass of 10.2 |

The pass is after the whole entry step of all regions, so a household that the ladder
removes or switches this tick is not moved, a household that was born this tick is
not settled (5.2), and the counts that the caps read are those after the births and
deaths of the tick. It is a world-wide step and not a per-region one, because the
pipeline cap of a destination takes the departures of every origin.

### 10.2 The pass

1. **Counts.** For each region, the households of the agents whose account is there
   (journeys included), and for each region the households on a journey to it.
   These are taken once, at the start of the pass.
2. **Evaluation**, for each agent in ascending ID that is eligible (3.3), settled and
   pushed (5.3): the candidates, their rumours with their draws (6.2), the
   estimates, the gates and the ranking (6.3 to 6.5). Outcomes `no_estimate`,
   `no_destination`, `no_rumour`, `too_far`, `no_viable`, `no_gain` are recorded
   here.
3. **Admission** of the agents with a ranking, in the order of 7.1, against the
   guard, the caps and the provisioning (7.2 to 7.5). The running counts of the
   outflow of each origin and of the pipeline of each destination include the agents
   admitted earlier in the pass.
4. **Departures** of the admitted agents in the order of admission (8.2).
5. **Events**: `MigrationStarted` for each departure, in the order of admission; then
   `MigrationReviewed` for each region in ascending `RegionID` (13.1).

10.2.1. **Draws.** The only random numbers are those of 6.2.1, in step 2, in the order
of ascending agent ID and, within an agent, ascending destination and commodity.
Steps 3 to 5 take none. The order does not depend on the number of regions with
news, on the caps or on the guard. A consumer's spawner whose `Decide` draws is
called before the pass (`spec-spawner.md` 10.2), so its draws precede this
module's.

10.2.2. **Cost.** The pass visits the agents on their review tick: about
`Households / MoveReviewTicks` a tick. For each of those it reads the earned ratio
(a running sum) and, for the pushed ones only, evaluates at most
`MaxRumoursPerView` candidates. In the reference world of 5 000 agents at a period
of 30 that is 167 reviews a tick, a few of them pushed, each a few dozen
candidates. Counts, not timings (N-015).

10.3. **A consumer's call.** The public face of `relocate` is INT's `Relocate`
(`spec-regions.md` 6.6, `spec-integration.md` 4.5): it calls the guarded path with its own
cause, in the world's between-step slot, for any agent that is not a pseudo-agent; for a
household on a journey the cancel of 8.4.1 is a step inside the call. This module does not
move an external agent and offers no way to have it do so: the default module is for
households.

## 11. Parameters

All are tunables (architecture.md 9.5) with a default (or "required"), an owner, a
bound and a doc line, fields of `Config.Modules.Migration` unless stated, with rows in
the parameter-owner table. **Every default is a placeholder** (N-015, N-027): none
comes from a source, and no source for how far or how readily households move was read
(N-018). The values come from the model of section 16, not from any society.

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| Modules | `Migration` | the module: present means on (`spec-config.md` 3.2); needs `Modules.Information` | nil (off) | |
| Migration | `MoveSpeed` | distance units a household covers in one tick on a journey, in the unit of REG's distances (6.4) | required | 1 to `MaxSpeed` |
| Migration | `MoveReviewTicks` | period of the review of an agent, in ticks (5.1) | 30 | 1 to 1 000 |
| Migration | `MoveBelow` | the gate: an agent is pushed when its earned ratio is below this, in basis points (5.3) | 10 000 | 0 to `MaxBasisPoints` |
| Migration | `MoveGain` | the least relative improvement of the estimate over the household's own measured figure that it needs, in basis points (6.5) | 1 000 | 0 to `MaxBasisPoints` |
| Migration | `MoveWidth` | the width over which the agents' margins are spread above `MoveGain`, in basis points (6.5) | 5 000 | 0 to `MaxBasisPoints` |
| Migration | `MoveCooldown` | least ticks since `RoleSince` before an agent is settled; the effective figure is the longer of this and the agent's window (5.2) | 60 | 0 to `MaxWelfareWindow` |
| Migration | `MoveHorizon` | the ticks over which a move is valued; a journey of this many ticks or more is not a destination (6.4) | 180 | 2 to `MaxTravelTicks` |
| Migration | `MoveOutShare` | most households that may start a journey from one region in one tick, in basis points of the region's households (7.3); 0 means none | 50 | 0 to 10 000 |
| Migration | `MoveInShare` | most households that may be on a journey to one region at once, in basis points of its households (7.5); 0 means none | 100 | 0 to 10 000 |
| Migration | `Regions[]` | rows of the region-keyed table (11.2) | none | at most `MaxRegions` |
| Role | `Fixed` | the households of the role are never moved by the module (3.3) | false | a bool |

11.1. **The placeholders and why.** `MoveReviewTicks` 30 is the base welfare window
(`spec-welfare-and-wages.md` 2.3.1), so a household is looked at about once a window.
`MoveBelow` 10 000 is a ratio of 1.0. `MoveGain` 1 000 and `MoveWidth` 5 000 are the
smallest margin and the width that, with the settling rule and the caps, stopped every
cycle of the model for the journey times and elasticities of 16.5; the model shows that
the width and the margin are each enough alone only when they are large against the
swing the flow can cause (16.2). `MoveOutShare` 50 and `MoveInShare` 100 are the
caps of the model: 5 households a tick and 10 on the way for a region of 1 000. They
are small on purpose: they set how fast a region can empty, and their cost is time to
settle (about 240 ticks for a 30% imbalance in the model, 16.3). `MoveCooldown` 60 is
twice the base window. `MoveHorizon` 180 is six base windows, so a journey of 3 ticks
loses under 2% of the gain. No default is a claim about how people move.

11.2. **The table of per-region caps** is a region-keyed table of the one mechanism
of `spec-regions.md` 5.1 (N-104):

| Scope | Field | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| RegionMove | `Region` | the region the row is for | required | a defined region |
| RegionMove | `OutShare` | replaces `MoveOutShare` in this region as an origin | the world's | 0 to 10 000 |
| RegionMove | `InShare` | replaces `MoveInShare` in this region as a destination | the world's | 0 to 10 000 |

One row per region (`ErrDuplicateID`), a row naming a region that is not defined
(`ErrUnknownRegion`), rows in region order (`spec-regions.md` 5.3). An absent row is
the world's value and draws no warning. A row equal to the world's values is legal. A
region that takes no immigrant, or lets nobody leave, is a row with a 0. A row is a
**replace** row. The row is added to the list of `spec-regions.md` 5.1.3 (handoff).

11.3. **Rule constants** (beside their code, not parameters, architecture.md 9.6): the
rotation `(tick + bucket(AgentID, MoveReviewTicks)) mod MoveReviewTicks` with the bucket
of `spec-world.md` 9.1.1 (`Weyl2`, multiplier `0x6A09E667`); the Weyl multiplier
2 654 435 769 of `spec-population.md` 3.8.3 for the margin offset; the use of the window mean and the exclusion of an empty
one (6.2); the form of the estimate (6.3); the discount `(H - k) / H` (6.4); the
order of the stages and of the admission tests (6.5, 7.6); the ranking and its
tie-break (6.5.1); the arrival tick `t + 1 + k` (8.2); the factor 3 of the warning
`migration_flow_window` (12.3); the reason constants `region_changed`, `migration` and
the outcome names (13).

11.4. **Not parameters:** the effective window (WEL, derived), `EntryRatio`
(SPN's, read), `RumourErrorBP`, `RumourMaxAge` and `InfoSpeed` (INF's), `CoverTicks`
(AGT's), the journey and arrival ticks (derived), the caps per tick (derived from the
shares and the households).

## 12. Validation

Each rule has a sentinel or a warning name and a failing-case test (testing.md 3.9).
Errors name the path, for example `modules.migration.move_gain`. The framework of
`spec-config.md` 7 runs them.

### 12.1 Errors

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field of section 11 outside its bound, a `MoveSpeed` that is missing (the zero value) included, a row with a share above 10 000 |
| `ErrModuleDependency` | `Modules.Migration` while `Modules.Information` is nil |
| `ErrUnknownRegion` | a `Regions` row naming a region that is not defined |
| `ErrDuplicateID` | two rows for one region |
| `ErrTooMany` | more rows than `MaxRegions`; a mobile role for which `|K(role)|` times the number of other regions exceeds `MaxRumoursPerView` (12.2) |
| `ErrOverflow` | `Households * MoveOutShare` or `Households * MoveInShare` beyond `MaxQuantity` at the largest group; a product of the estimate that the 128-bit helper cannot hold (12.2) |

No new sentinel. `Role.Fixed` has no error.

### 12.2 Derived checks

Computed at construction from the configuration (`spec-config.md` 4.2 is the home):

- for each role that is not `Fixed` and has recipes, the size of `K(role)` (6.1) times
  the number of other regions is at most `MaxRumoursPerView`, the bound of INF for one
  view (`ErrTooMany`);
- the largest number of households a region can have times `MoveOutShare` and
  `MoveInShare` (and the rows') fits `MaxQuantity` (`ErrOverflow`);
- the estimate: the largest weight is not in it (per household), the largest
  output quantity times `MaxYieldBP` times `MaxPrice` is at most `MaxQuantity * MaxPrice`
  = 10^18 for one output, and the sum over the outputs of a recipe is bounded by
  `MaxRecipeItems` times that; the sum is taken with checked addition and the product
  of the estimate and the horizon with the 128-bit helper. A recipe whose estimate
  would overflow at the largest price is already rejected by the unit-cost check of
  `spec-production.md` 10.3.

### 12.3 Warnings

Warnings never fail validation (`spec-config.md` 7.2). Each is raised once at
construction, at `Warn`, with the attributes `rule` and `path`; the volume is bounded
by the configuration.

| Warning | Condition |
|---------|-----------|
| `migration_without_routes` | the module is on and there are fewer than two regions or no edge: no household can have a destination |
| `migration_no_mobile_role` | the module is on and no role has both a recipe and `Fixed` false: the module can move nobody |
| `migration_cohort` | a mobile role has a group whose `Scale` is above 1: a cohort moves whole (3.2.2) |
| `migration_provision` | the shortest journey between some pair of regions with a route has a journey time `k` above `CoverTicks` of AGT: a default household holds about that many ticks of upkeep and cannot be provisioned for it (7.4.1); carries `pairs`, the count |
| `migration_far` | at least one ordered pair of regions has a route whose journey time is `H` or more: it is never a destination (6.4); carries `pairs` |
| `migration_flow_window` | `MoveOutShare` times the longest effective welfare window of a mobile role exceeds 3 times `MoveGain` (placeholder, from the model of 16.6): the flow can reverse. Carries the product and `MoveGain` |
| `migration_uncapped` | `MoveOutShare` is at least `10 000 / MoveReviewTicks` and `MoveInShare` is at least 10 000: no cap can bind, and only the rotation limits the flow |

12.3.1. **The flow warning.** `migration_flow_window` is the analogue of
`entry_share_window` (`spec-spawner.md` 12, 17): a condition on a product of a share
and a window, found by a stylised model. In the model (16.6) a flow that moves
`MoveOutShare` of a region in each tick of a window reversed when the product was
more than about 5 times the margin and did not when it was 3 or less, with the width
at 0. It is a warning, and the placeholder is the model's. It does not know the
elasticity of income to households, so it takes the worst one that the diminishing
returns of a fixed factor can give (an elasticity of 1).

## 13. Observability

### 13.1 Events

Plain data, no map, no pointer, no wall-clock time (observability-and-logging.md 4.1),
emitted only after the transition they describe is complete, and constructed only
when an observer was supplied (4.5 there).

| Event | Emitted | Fields (besides `Tick`) |
|-------|---------|-------------------------|
| `MigrationStarted` (new) | a journey begins (8.2), once per departure, in the order of admission | `Agent`, `Role`, `From`, `To` (`RegionID`), `Weight`, `Arrives` (the arrival tick), `Distance` (REG's, `int64`), `Journey` (`k`), `EarnedRatio` (basis points, signed), `Home` (`V_a`), `Estimate` (`V_eff,d`, basis points), `JobAbandoned` (bool) |
| `MigrationReviewed` (new) | once per region per tick in which the module is on and any household was pushed or held back by the settling rule, any arrival was held or any journey ended by a removal; never when all are zero. Counts of **households**, attributed to the region of the agent's account | `Region`, `Unsettled` (households whose turn came, that the gate of 5.3 would have pushed and that were not settled, 5.2.3), `Reviewed` (households eligible and settled whose turn came), `Pushed`, then the outcomes of 7.6: `Started`, `NoEstimate`, `NoDestination`, `NoRumour`, `TooFar`, `NoViable`, `NoGain`, `Bound`, `OutCap`, `Unprovisioned`, `InCap`; and `Held` (arrivals held, 8.4), `Ended` (journeys ended by a removal, or cancelled by `Relocate` of INT, since the last pass, 8.3, 8.4.1) |

The arrival is the event `AgentRelocated` of REG, with the cause `migration`
(`spec-regions.md` 10.1): this module adds none for it. Pushed equals the sum of the
eleven outcomes (FR-MIG-025). `MigrationStarted` is per agent, like `AgentSpawned`
and `AgentRelocated`; `MigrationReviewed` is an aggregate bounded by the number of
regions, and is the counter of every silent limit of the module (N-045, N-065). The
`Region` fields carry the `RegionID`; the string is in the names table of `WorldCreated`
(`spec-observability.md` 3.2; answers `spec-regions.md` 17.4).

13.1.1. **Households on the way** can be rebuilt from events: `MigrationStarted`
(+ `Weight`), `AgentRelocated` with the cause `migration` (- `Weight`) and `Ended`
(- households). A count of households by region needs `AgentRelocated` as it
already does (`rules/observability-and-logging.md` 4.3): the households of a
migrant are in the origin until the arrival.

### 13.2 Logs

Constant lowercase messages, typed attributes, `Enabled` guards
(observability-and-logging.md 3.1.3).

- `Info`: nothing new. The construction summary adds `migration` to its `modules` list.
- `Warn`, at construction: one per warning of 12.3, with `rule` and `path`.
- `Warn`, bounded per region (observability-and-logging.md 3.2.2): "migration
  blocked", for a region in which households were pushed in each of ten consecutive
  ticks and none started, at the tenth tick and every tenth after, with `region`,
  `households` (pushed in the tick) and `reason` (the outcome that held most of them);
  "arrivals held", for a region with a held arrival in each of ten consecutive ticks,
  at the tenth and every tenth after, with `region` and `households`. Never per agent.
- `Debug`, per agent: "migration started" (`agent`, `from_region`, `to_region`,
  `households`, `lag`: the journey), "migration declined" (`agent`, `reason`: the
  outcome), "arrival held" (`agent`). All guarded by `Enabled`.
- No `Error`. No `Info` inside the tick.

No new log key: `agent`, `region`, `from_region`, `to_region`, `households`, `reason`
and `lag` exist. The values of `reason` are the outcome constants of 7.6 and
`held`.

### 13.3 Metrics

Three instruments for the catalogue of `rules/observability-and-logging.md` 5.3
(proposals for OBS), fed by the events above:

| Name | Instrument | Unit | Attributes |
|------|-----------|------|------------|
| `economysim.migration.started` | Counter | `{household}` | `economysim.region.from`, `economysim.region.to` |
| `economysim.migration.limited` | Counter | `{household}` | `economysim.limit` |
| `economysim.migration.on_journey` | UpDownCounter | `{household}` | none |

`economysim.migration.started` is fed by `MigrationStarted` (`Weight`);
`economysim.migration.limited` by the counts of `MigrationReviewed` other than
`Started` and `Pushed` and `Reviewed`, by outcome; `economysim.migration.on_journey`
by 13.1.1. All count households and never agents. `economysim.limit` is a new
attribute with the twelve constants `no_estimate`, `no_destination`, `no_rumour`,
`too_far`, `no_viable`, `no_gain`, `bound`, `out_cap`, `unprovisioned`, `in_cap`,
`held` and `ended`; its cardinality is constant (5.4.1 there lists the allowed
attributes: this one is added). The first instrument adds at most `MaxRegions` squared
series, 20 for the reference world, as `economysim.household.relocations` does.

### 13.4 Snapshot

Per region: the households on a journey from it and to it; the agents with a journey
(agent, destination, arrival tick), bounded by the agents and so by the scale. The
counts of `MigrationReviewed` are the event, not a snapshot field. The names are OBS's.

### 13.5 No observer, no logger

The pass runs whether or not anyone watches; no event is constructed without an
observer. No log, event or metric takes a draw, reads the clock or changes a result
(observability-and-logging.md 1.4); the draws of 6.2.1 do not depend on whether an
observer is attached.

## 14. State contributed to a checkpoint

Every spec lists the state it contributes (ADR 0002). This spec's, plain data in a
fixed order with no maps:

- per agent on a journey: the journey (the destination `RegionID` and the arrival tick),
  hosted in the agent record and saved there in ascending agent ID
  (`spec-checkpoint.md` 4.2 tag 3);
- per region, in ascending `RegionID`: the run of consecutive ticks with pushed
  households and no start, and the run of consecutive ticks with an arrival held,
  for the two `Warn`s of 13.2 (diagnostic: they affect no result and are saved so
  that a resumed run logs as the original did);
- the count of journeys ended by a removal or cancelled by `Relocate` since the last pass, per region
  (reported once and then zero: it is saved so that a resumed run reports as the
  original did).

`RoleSince` is SPN's state and is saved by it (`spec-spawner.md` 14). **Not saved**
(derived): the households by region, the households on a journey to a region, the
caps of a tick, the candidates, the rumours, the estimates and the rankings, the
outcomes of a pass. The random source's state is the world's. A world restored
between ticks makes the same decisions and the same draws as an uninterrupted one
(FR-MIG-041).

## 15. Worked examples

Two settings, each unlike the other, chosen to show the module with a short
journey and with a long, asymmetric one. Both are **examples** with placeholder
numbers and invented names; the graphs are those of `spec-regions.md` 12.1 and 12.2 and
the news is that of `spec-information.md` 12. The arithmetic is in
`models/mig/worked.py` (its output is `worked.out`).

### 15.1 Example A: a river valley (a pre-industrial setting; unit: one league)

Regions `lower` (0), `middle` (1), `upper` (2). Shortest distances, row from, column to
(`spec-regions.md` 12.1): `lower` 0, 55, 120; `middle` 30, 0, 70; `upper` 70, 40, 0.
One role, `cultivator`, whose recipe `grow_staple` takes 4 ticks, consumes 2 `seed` and
makes 12 `staple`; the yield table gives `upper` 8 000, `middle` 10 000, `lower` 12 000
(a poor soil upstream). The world's upkeep is 2 `staple`; `upper` adds 1 `fuel`.

```toml
[modules.information]
info_speed        = 40        # leagues of distance that news covers in one tick
rumour_error_bp   = 500
rumour_max_age    = 60

[modules.migration]
move_speed        = 20        # leagues a tick, on foot
move_review_ticks = 30
move_below        = 10000
move_gain         = 1000
move_width        = 5000
move_cooldown     = 60
move_horizon      = 180
move_out_share    = 50
move_in_share     = 100
```

**The journeys.** `k = max(1, ceil(D / 20))`: from `upper`, `middle` is 40 leagues, so
`k = 2`; `lower` is 70, so `k = 4`. The news that reaches `upper` is that of INF's
table: from `middle` a lag of 3, from `lower` a lag of 4.

**A household.** Agent 1042, a `cultivator` of weight 1 in `upper`, is settled and
reviews this tick. Its local report (window means) is `staple` 298 and `fuel` 405, so
its upkeep costs `2 * 298 + 405 = 1 001`. Its own observed wage is 640 a working
household-tick, so `V_a = 640 * 10 000 / 1 001 = 6 393`. Its earned ratio is 5 100,
below `MoveBelow`: it is pushed. Its rumours, window means after the blur: `middle`
`staple` 321, `seed` 80; `lower` `staple` 366, `seed` 82.

| | `middle` | `lower` |
|---|---------:|--------:|
| `k` | 2 | 4 |
| value of 12 `staple` at the table's yield | `floor(12 * 10 000 * 321 / 10 000)` = 3 852 | `floor(12 * 12 000 * 366 / 10 000)` = 5 270 |
| cost of 2 `seed` | 160 | 164 |
| net per execution; per tick (over 4) | 3 692; 923 | 5 106; 1 276 |
| upkeep there (2 `staple`) | 642 | 732 |
| `Vhat` | `923 * 10 000 / 642` = 14 376 | `1 276 * 10 000 / 732` = 17 431 |
| `V_eff` with `H` = 180 | `14 376 * 178 / 180` = 14 216 | `17 431 * 176 / 180` = 17 043 |

The agent's margin is `g_a = 1 000 + floor(u * 5 000)` with `Offset(1042)` =
4 258 099 970, so `u` = 0.991 and `g_a` = 5 957; it needs `V_eff` at least
`6 393 * 15 957 / 10 000` = 10 201. Both pass, both are viable (above `EntryRatio`
8 000), and the ranking is `lower` (17 043), then `middle` (14 216).

**Admission.** The guard finds no obligation. `upper` has 600 households, so its
outflow cap is `ceil(600 * 50 / 10 000)` = 3 and two households have been admitted
before this one: it is the third, and is admitted. Provisioning: the household
holds 6 `staple` and 3 `fuel`. To `lower` (`k` = 4) it needs `4 * 2` = 8 `staple`
and 4 `fuel` of `upper`'s upkeep: **unprovisioned**. To `middle` (`k` = 2) it needs 4
and 2: it has them. `middle` has 1 000 households, a pipeline cap of
`ceil(1 000 * 100 / 10 000)` = 10, and 3 on the way: there is room. It starts for
`middle`.

**Departure and arrival.** It departs at the end of tick `t`. A job of `grow_staple`
begun in tick `t - 1` is abandoned (`region_changed`) and its 2 `seed` are lost. In
ticks `t + 1` and `t + 2` it is on the journey: it makes and trades nothing, eats 2
`staple` and 1 `fuel` a tick from its account (the account is in `upper`, whose extra
upkeep applies), and its record gets two ticks of zero income. In the first phase of
tick `t + 3` the account changes region (`AgentRelocated`, cause `migration`), its
beliefs are reseeded to `middle`'s window means, `RoleSince` becomes `t + 3`, and it
is not reviewed again before `t + 63` (the longer of 60 and its window). Another
household that held 9 `staple` and 4 `fuel` would have been provisioned for `lower`
and, ranking it first, would arrive in tick `t + 5`.

**Accounting for the tick.** Of the 600 households of `upper`, 40 were eligible and
settled and 12 were pushed. Their outcomes (7.6): 3 `no_gain`, 2 `unprovisioned`, 4
`out_cap`, 3 `started`; `3 + 2 + 4 + 3` = 12.

### 15.2 Example B: an orbital colony (a space setting; unit: velocity change, m/s)

Regions `belt` (0), `orbital` (1), `surface` (2), with the edges of
`spec-regions.md` 12.2 (shortest distances, row from, column to: `belt` 0, 3 000,
5 000; `orbital` 3 000, 0, 2 000; `surface` 12 400, 9 400, 0). INF's `info_speed` is 4 000
with no error (a rumour is exact but late), `rumour_max_age` 6. The household role is
`technician`.

```toml
[modules.migration]
move_speed        = 3000      # effort units a tick
move_review_ticks = 24
move_below        = 11000
move_gain         = 1500
move_width        = 3000
move_cooldown     = 48
move_horizon      = 6
move_out_share    = 100
move_in_share     = 200
```

`k = ceil(D / 3 000)` and the discount `(6 - k) / 6`:

| from > to | `belt>orbital` | `belt>surface` | `orbital>belt` | `orbital>surface` | `surface>belt` | `surface>orbital` |
|-----------|---:|---:|---:|---:|---:|---:|
| `k` | 1 | 2 | 1 | 1 | 5 | 4 |
| `(H - k) / H` (basis points) | 8 333 | 6 666 | 8 333 | 8 333 | 1 666 | 3 333 |

The well is a trap for the migration module because it is a trap for the journey:
up costs 4 or 5 of the 6 ticks of the horizon and down costs 1.

- **Coming down.** Technician 2203 of `orbital`, `V_a` 9 000, estimates `surface` at
  `Vhat` 15 000: `V_eff = 15 000 * 5 / 6` = 12 500. Its margin is `1 500 + floor(u * 3 000)`
  = 3 086, so it needs `9 000 * 13 086 / 10 000` = 11 777. It passes and goes
  (`k` = 1, arrives in `t + 2`).
- **Going up.** Technician 2204 of `surface`, `V_a` 9 000, estimates `orbital` at 21 000
  (more than twice its own figure) and `belt` at 30 000. `V_eff` is `21 000 * 2 / 6` =
  7 000 and `30 000 * 1 / 6` = 5 000; its margin is 1 940, so it needs 10 746. Both
  fail at stage 4: `no_gain`, with the estimate above its own by more than two
  times. A pack that wants the colony's surface to empty into the orbit sets
  `move_horizon` larger.
- **News.** With `rumour_max_age` 4 a household of `belt` hears nothing of `surface`
  (lag 5 is too old): every `surface` candidate fails at stage 1 (`no_rumour`). With
  the edge `orbital>surface` closed (`spec-regions.md` 7), `surface` has no way from `belt`
  or `orbital`: it is not a candidate (6.1, `no_destination` if it was the only
  one), while its households can still go up (their edge is open).

### 15.3 Use by the reference chain (a proposal for REF)

The reference chain's `five` variant has five regions, 5 000 households, no edges
until the trade overlay (`spec-reference-chain.md` 4.4.4, `spec-regions.md` 12.4). A
migration overlay adds `[modules.information]` and `[modules.migration]` to the trade
overlay, with `move_speed` chosen against the edge lengths of that overlay so that
the shortest journey is 1 or 2 ticks and the longest under `CoverTicks`; the roles
are not `Fixed`. Scenarios MIG-S1 to MIG-S6 (17) run on it. REF builds the file; the
numbers of this section are not its.

## 16. What was run

One script in `doc/specs/models/mig/`, run in Docker (`python:3.12-slim`, the standard
library only, `PYTHONDONTWRITEBYTECODE=1`), as the README of that folder says, and
one for the arithmetic of the examples. `loop.py` is a **prototype of the rules, not
of the library**. The rules are the integer rules of sections 5 to 8: the rotation,
the settling rule, the push gate, the estimate from a rumoured income, the horizon,
the margin with a width from the Weyl offset, the outflow cap, the pipeline cap,
the timed journey with the arrival at `t + 1 + k`, and a rumour with an
independent bounded error. The economy is **one assumed function**: the income per
working tick of a region, in basis points of the upkeep, is `A_r * (n0 / n)^eta`,
with `n` the households of the region that are not on a journey: a fixed factor with
diminishing returns, as in the models of POP. The household observes the mean of
that income over the working ticks of its own window of 30; a rumour is the income as
it was `Lag` ticks ago; a journey is a record and no goods. Floating point is used in
that function and in the printouts, and nowhere else. There is no inventory, no
price, no belief, no market and no provisioning (every household is assumed
provisioned), and no recipe: **the model does not show the library**. The default
parameters of section 11 are those of the model; nothing here calibrates them
(N-015, N-027). The output is `loop.out`.

### 16.1 The linearised loop is a delayed integrator

Two regions with the same structure and the households of one in excess by a share `z`:
the flow out is `(1 + z) P / R` a tick, with `P` the share of the reviewing households
that pass the gate, which rises linearly across the width of the margins, and the gap
that decides is lagged by the delay `tau`. Linearised, `dz/dt = -kappa z(t - tau)`
with `kappa = 2 eta / (R W)`, `W` the width of the margins as a share and `eta` the
elasticity of income to households. The first run integrates that equation: the
solution is monotone for `kappa tau` up to `1/e` (0.368), decays with overshoot up to
`pi/2` (1.571) and grows above it (at 1.56 each quarter of the run has 0.37 of the amplitude
of the one before, at 1.58 it has 2.3 times). This is the classical result for the delayed
integrator and the model confirms it. Its use is the
direction: the gain falls with the review period and the width, and the delay is
`T/2` (the lag of the household's own observed wage) plus the rumour's lag plus the
journey plus `R/2` (the stagger). With the defaults (`R` 30, `W` 0.5, `eta` up to
1) `kappa tau` is above `pi/2`: the **linear** condition is not met, and the
loop is held by what the linear model leaves out, the margin (a dead zone inside
which nobody moves) and the caps (a bound on the flow). The next runs show that.

### 16.2 Each damper alone

Two equal regions, 1 000 households started 800 and 200 (the equilibrium is 500),
`eta` 1, a margin of 300 (3%), a review period of 10, no width, no caps, no settling,
no horizon (the undamped base). "Lowest" is the least households the first region
reaches after it first reaches 500; "recrossings" counts reversals through 500
after that, over 2 400 ticks.

| Case | Lowest | Recrossings | Moves needed for the balance |
|------|-------:|------------:|-----------------------------:|
| undamped base | 80 | 239 | 640 times |
| review period 30 / 60 | 288 / 368 | 112 / 63 | 154 / 54 times |
| width 5 000 on the margin | 121 | 239 | 588 times |
| a **relative** spread of the margin (+-90% of 300) | 76 | 239 | 640 times |
| margin 2 000 | 126 | 239 | 627 times |
| the caps (outflow 50, on the way 100) | 489 | 0 | 25 times |
| the settling rule | 80 | 99 | 115 times |
| the horizon (`H` 60: 5% off a journey of 3) | 80 | 239 | 640 times |
| the defaults together (R 30, margin 1 000, width 5 000, caps, settling, `H` 180) | 514 | 0 | 0.95 times |

With the settling rule on, each other damper alone: the review period leaves 98 and 63
reversals; a width of 2 000 leaves 10 and of 5 000 leaves 2; a margin of 1 000 leaves
99 and of 2 000 leaves 1; the horizon leaves 98; the caps leave 0 (with a slow balance:
about 240 ticks). What the runs say, and what they do not: **the caps are the one
damper that works alone**, the settling rule is necessary and not enough, a width and a
margin work with the settling rule when they are large against the swing the flow
can cause, and the rotation and the horizon do not stop a cycle. The width has to
be an **absolute** width: a spread that is a share of a small margin (the 'relative'
row) disperses nothing, because the thresholds it spreads are all near the margin. That
is why `MoveWidth` is added to the margin and not multiplied (6.5).

### 16.3 The pipeline, the caps and the time to settle

The journey time `k`, with a margin of 1 000, a review period of 30, no width, and the
caps on or off:

| Journey `k` | No caps: lowest, recrossings, moves needed | Caps (50 / 100): lowest, recrossings, moves needed |
|------------:|:-------------------------------------------|:---------------------------------------------------|
| 1 | 370, 2, 2.5 times | 492, 0, 1.03 times |
| 3 | 339, 6, 4.7 times | 499, 0, 1.00 times |
| 8 | 258, 78, 87 times | 505, 0, 0.98 times |
| 20 | 91, 2, 9 times | 508, 0, 0.97 times |

A journey the next review cannot see makes the swing larger with its length and the
pipeline cap removes it (MIG-A3). The caps are not free: with the width at 0 and the
outflow at 50 the balance of a 300-household imbalance is reached in 242 ticks with
the pipeline cap at 100 and in 104 at 300, and does not happen within 2 400 ticks
with a pipeline cap of 50. A pipeline cap tighter than the outflow times the journey
throttles the flow below what the information would let; the defaults are
chosen so that the pipeline cap is about the outflow times a journey of 2.

### 16.4 Herding

Three regions: the first is poor and holds 1 000 of 1 500 households, the other two
are equally good. Every household of the first sees the same ranking. Without
caps, with exact news and identical margins, the households go to one destination and
the next wave to the other: the difference between the two destinations swings by 674
households (peak to peak) and 4 143 journeys are made. With an independent rumour
error of 500 the swing is 0 and the journeys 1 854; with the width as well 1 030.
With the caps and exact news the swing is 9 and the journeys 595; with the defaults
(caps, error 500, width 5 000) 8 and 582, and at most 5 households are on the way to
one destination at a time (against 84 without caps). The independent error and the
caps each remove the herd; the identical household with exact news is the case that
makes it (MIG-A2).

### 16.5 The defaults over seeds, elasticities and journeys

Two equal regions started 800 and 200, the defaults of section 11, 5 seeds, `eta`
0.5 and 1.0, journeys of 1, 3 and 8 ticks: the first region never falls to 500. It
stops between 505 and 559, inside the dead zone of the margin, after 241 to 295 journeys
(the minimum to the balance is 300), in every one of the 30 runs: no cycle and no
overshoot. An unequal pair (income factors 10 000 and 13 000, `eta` 0.5, equilibrium 372
of 1 000) stops at 410 to 414. The imbalance that remains is the cost of the margin
and the width: a household stays where the gain is under about 10% to 60%. That is the
hysteresis band of the design and the reason the equilibrium is not reached; a pack
that wants a closer balance lowers `MoveGain` and `MoveWidth` and pays in stability
(16.2).

### 16.6 When the flow reverses

With the width at 0, no pipeline cap, `eta` 1 and a period of 30, the outflow share
times the window against the margin:

| `MoveGain` | Outflow share times window / margin: reversals 0 up to | first reversals at |
|-----------:|:------------------------------------------------------|:-------------------|
| 300 | 2.0 | 5.0 (119) |
| 1 000 | 3.0 (and 1.5) | 6.0 (2) |
| 2 000 | 4.5 | none up to 4.5; 1 at the uncapped extreme |
| 4 000 | all | none |

So a flow of `MoveOutShare` of a region in each tick of a window, more than about 3 to
5 times the margin, reverses; at 3 or less it did not in any run. The warning
`migration_flow_window` (12.3) uses 3. The condition is conservative for a large margin
and tight for a small one, and the elasticity is that of the worst case.

### 16.7 What the models do not show

The economy is one function: no price, no belief, no stock, no inventory, no
provisioning, no crowding of a fixed factor beyond the function, no relief, no
second effect of the arrivals on the destination's other markets, no destination
whose land is exhausted. The household observes the regional income exactly and
the rumour is exact but for the blur. The margin is the only trigger of a stop. The
model has two or three regions and one role; with five regions, a chain and a
price-formation loop of its own (L01 to L09), the delay of the loop grows by the
belief lag and the economy's own. It shows that the dampers can stop a cycle in this
stylised loop and which of them do so alone. It does not show that any default is
stable in a pack (N-027).

## 17. Entries for the stability register

For `spec-stability.md`, by the checklist of its section 9 (FR-STB-003). Fields as in
its section 3; the register owner assigns the `STB-Lnn`, `STB-Dnn` and `STB-Snn`
numbers, and the entries here are `MIG-An`, `MIG-Dn` and `MIG-Sn`. A section number
that names no spec is a section of this one.

**This spec closes a loop.** By 9.1 of the checklist: it reads a household's own
measured wage and ratio and rumoured prices of other regions, and it changes the
households of two regions, and so their incomes and prices, which the next review
reads, through a rumour's lag, a journey and a window. The loop is the migration
cobweb (A1). It adds three amplifiers on it (A2, A3, A8), and two hazards of its own
(A7, A9); three further entries are cases of loops that exist and are merged into
them (A4, A5, A6). The MIG row of `spec-stability.md` 4.8 is answered here: the
rumoured ratio it expected is a rumoured price (4.4.1); "a flow toward a region lowers
its ratio; relief and births raise it" is A1, A4 and A5; "whether a migrant keeps or
reseeds its beliefs" is decided (8.5).

**MIG-A1. The migration cobweb.** *Kind:* loop. *Loop:* the income gap between a dear
and a poor region (+) the households that pass the gate and start (+) the households
of the dear region, after the journey (-) its income per head (-) the gap; and the same
starts (-) the households of the poor region (-) its income per head (+) the gap.
Each path has an odd number of negative links, so each is balancing (B). *Delay:*
`tau = T_a / 2 + ell + k + R / 2`, the lag of the household's own observed wage (half
its effective window), the rumour's lag `ell = Lag - 1 = ceil(D / InfoSpeed)`, the
journey `k` and half the review period (16.1); 32 to 40 ticks for the defaults and a
window of 30. *Owner:* MIG 5 to 8. *Damper:* MIG-D1 to MIG-D7, of which the pipeline cap
(D5), the outflow cap (D4) and the settling rule (D2) are decisive and the margin and
its width (D3) act with them (16.2); D59 to D61 (INF's lag floor, error and horizon)
shape the signal. *Undamped:* a flow whose swing exceeds the margin: a small margin
with a large outflow share and no pipeline cap (the base of 16.2: 239 reversals,
the region falling to 80 of 500). *Condition:* the linear condition `kappa tau < pi/2`
with `kappa = 2 eta / (R W)` (16.1) is **not** met by the defaults and is not the
guarantee; the guarantee is that the swing the caps allow stays inside the margin:
`MoveOutShare * T_a < 3 * MoveGain` (the warning `migration_flow_window`, 16.6),
together with the pipeline cap. *Test:* MIG-S1. *Status:* model (16.2, 16.3, 16.5,
16.6). It does not show the library: no price, belief or market, one role, and the
economy is one function (16.7). *Neighbouring delays (checklist 9.6):* the welfare
window `T_a` (30 or more) is the larger part of `tau`, as in L14, L19 and L36;
`RoleCooldown` 30 and `ReviewTicks` 10 of SPN (L36) act on roles, and `MoveCooldown` 60
and `MoveReviewTicks` 30 act on places, at a similar timescale: a pack changes both
only after reading both; `RemovePrivation` 10 against a journey `k`: a migrant that
starts unfed with a privation near 10 can be removed on the way, in its origin;
POP's population time constant (L75) is decades and far from `tau`; a journey near a
season's length (CAL) would carry the household across the yield profile of the
arrival (6.3 reads the arrival's season).

**MIG-A2. Herding on one destination.** *Kind:* amplifier (neighbours L93, L36). *Loop:*
every pushed household of a region reads the same structure and nearly the same news:
the ranking (+) the number that choose the same destination (+) its pipeline and, after
`k`, its arrivals (-) its estimate (-) the number that choose it next; the next wave
chooses the other, a cobweb across destinations. B with the delay of A1. *Owner:* MIG 6.
*Damper:* D60 (INF's independent error, which breaks the tie between destinations that
differ by less than the error), MIG-D3 (the width), MIG-D5 (the pipeline cap).
*Undamped:* identical households with exact news and no cap: the difference between two
equal destinations swings by 674 households of 1 500 (16.4). *Condition:* an independent
rumour error, or a width, or the pipeline cap, as in L93: a common error would be worse
than none (a rumour's error is independent per reader, `spec-information.md` 5.5.3).
*Test:* MIG-S2. *Status:* model (16.4).

**MIG-A3. The pipeline of movers is invisible.** *Kind:* amplifier (the analogue of L85,
TRN-A2). *Loop:* the destination's estimate, built from prices that do not yet show
the households on the way (+) the starts (+) the households on the way (+, after `k`
and the rumour's lag) the arrivals (-) the destination's price of its outputs and (+)
of its inputs (the estimate falls): a reviewer that sees only old prices sends more
while the earlier movers are still on the way, and the region ends up with more than
it wanted `ell + k` ticks later. R in the stock signal, with the delay `ell + k +` the
belief lag. *Owner:* MIG 7.5. *Damper:* MIG-D5 (the pipeline cap counts the
households on the way, the module's `Inbound`), MIG-D4. *Undamped:* `MoveInShare` of
10 000 or more. *Condition:* the cap on the households on the way is below what the
margin tolerates. *Test:* MIG-S1. *Status:* model (16.3: a journey of 8 or 20 ticks swings the region to
258 and 91 of 500 without the cap, 505 and 508 with it).

**MIG-A4. Relief and migration.** *Kind:* hazard; **merges into L40** (README Still
open 4; L40 was "untested"). *What was decided:* relief does not suppress migration,
because the trigger and the pull read earned figures (5.3, 5.4). *What remains:* an
arriving household whose record is poor, or that has just spent `k` ticks with no
income, draws relief from the destination's pool if its ratio is below `ReliefBelow`:
the pool serves a third master (L63), and a destination that receives many poor
arrivals sees its pool fall. *Damper:* the pipeline cap (D5) bounds the arrivals;
D37 keeps part of the pool for relief. *Condition:* the destination's pool covers the
arrivals' relief; the cap is small against it. *Test:* MIG-S3 (outflow equal with
relief on and off) and S14. *Status:* untested; the decision is by construction.

**MIG-A5. A migrant carries its money.** *Kind:* bias; **merges into L68 (REG-A3) and L76**
as cases. The money of an arriving household enters the destination's demand and the
origin's loses it, with no flow; the relocation totals reconcile it (`spec-regions.md`
6.4). The households pushed to move are the poor, so the money a flow carries is less
than a mean household's: the destination's money per household falls, and with POP on
its funded births per death with it (L76, the stake loop); a flow of rich households
would raise it. The loop the REG entry handed to MIG ("migration raises demand and
prices in the destination, which lowers the ratio that triggered it") is the
balancing half of A1. *Damper:* none needed; D51 (the stake) shows. *Test:* S14 with
immigrants. *Status:* hand.

**MIG-A6. A journey is a gap in income.** *Kind:* hazard; **merges into L19** as a case
(TRN-A6). A journey of `k` ticks puts `k` records of zero income and the origin's
upkeep into the window (8.6). The earned ratio is lower by about `k / T_a` for `T_a`
ticks after arrival. *Damper:* D29 (the window is derived), MIG-D2 (the settling rule
waits `T_a` before the record can trigger anything). *Undamped:* the relief it can
draw (A4) and the reading of the ratio by POP's schedules (L75) for those ticks.
*Condition:* `k` small against `T_a`. *Test:* S11 extended with a journey. *Status:*
hand.

**MIG-A7. The poverty trap of the journey.** *Kind:* hazard (a bound). *Loop:* none. A
household that has little (+) cannot hold the upkeep of a journey (+) is
`unprovisioned` (-) moves: the poor go only where `k` is small. In a realm with long
edges a failing region keeps its poor, who are then under relief and the ladder
(L39, L35, L63) and not on the move. *Owner:* MIG 7.4. *Damper:* none by design; it
is the price of charging a journey in the household's own stock and not in a faucet
or a sink; it is counted (`Unprovisioned`, the bounded `Warn` "migration blocked") and
the warning `migration_provision` names the edge. *Condition:* `CoverTicks` of AGT at
least the longest journey a pack expects poor households to make, or a pack that
accepts the trap. *Test:* MIG-S4. *Status:* hand.

**MIG-A8. Emptying a chain.** *Kind:* hazard (extends L64, L82). *Loop:* households leave a
region (-) the households of its roles, which are the demand and the supply of the
chain's other roles (+) the incomes of those who remain, when the chain is incomplete
(-) the remaining households' earned ratio (+) the pushed (+) the households that
leave. R, bounded by the caps. *Delay:* the window and the rumour. *Owner:* MIG 7.3.
*Damper:* MIG-D4 limits how fast a region can empty; MIG-D1 and D2 slow it; `Role.Fixed`
keeps a role in place; POP's `MinHouseholds` (D49) protects only against the
schedule's deaths, not against a flow. SPN does not re-seed an extinct role (L64).
*Undamped:* a chain role emptied by the flow, then the others. *Condition:* a pack whose
chain survives at the smallest group it will have after the flow, or roles marked
`Fixed`. *Test:* MIG-S5. *Status:* hand.

**MIG-A9. A mover carries its whole account.** *Kind:* bias. Goods that a household
holds travel with it for free, where a trip's traveller is limited by a capacity
(`spec-transport.md` 3.2.2). A consumer's strategy that stockpiles to ride a journey
for the carriage gets a free shipment. *Damper:* the buffer rules of the default
strategy keep holdings small (`spec-agents.md` 7.3). *Condition:* the default
strategy, or a pack with `Fixed` set on stockpiling roles. *Test:* the invariants of
MIG-S6 (conservation is exact; the leak is of value, not of units). *Status:* hand.
Open question 4.

### 17.1 Dampers this spec adds or specifies

| ID | Damper | Mechanism | Home | Status |
|----|--------|-----------|------|--------|
| MIG-D1 | Staggered migration review | a household decides only on its turn, `(tick + bucket(AgentID, MoveReviewTicks)) mod MoveReviewTicks`, no draw (a form of D25 and D33); alone it does not stop the cycle (16.2). Cost: slow response | 5.1 (`MoveReviewTicks`) | specified |
| MIG-D2 | Settling | a household is not reviewed until the longer of `MoveCooldown` and its own effective window have passed since it took up its place, so a record made partly of the place it left cannot trigger a second move (a form of D34 and D29). Cost: a mistaken move stays for a window | 5.2 (`MoveCooldown`) | specified (counted by `Unsettled`) |
| MIG-D3 | Margin with a width | a household moves only for a gain beyond its own margin, `MoveGain` plus a share of `MoveWidth` by agent ID, so the thresholds are spread (a form of D09 and D08, and the "fixed spread" of INF 13). Cost: a persistent income gap of the order of the margin (16.5) | 6.5 (`MoveGain`, `MoveWidth`) | specified (counted by `NoGain`) |
| MIG-D4 | Outflow cap | at most `MoveOutShare` of a region's households start a journey in a tick (the opposite of D14). Cost: a region empties slowly | 7.3 (`MoveOutShare`) | specified (counted by `OutCap`) |
| MIG-D5 | Pipeline cap | at most `MoveInShare` of a destination's households may be on the way to it at once, so the movers the next review cannot see are bounded (the module's `Inbound`, D38). Cost: time to settle; a destination can refuse | 7.5 (`MoveInShare`) | specified (counted by `InCap`) |
| MIG-D6 | Horizon | the gain is discounted by the share of the horizon the journey leaves, and a journey as long as the horizon is no destination. Cost: the far and the high are out of reach (15.2) | 6.4 (`MoveHorizon`) | specified (counted by `TooFar`) |
| MIG-D7 | Provisioning | a household goes only if it holds the upkeep of the journey. Cost: the poverty trap (A7) | 7.4 | specified (counted by `Unprovisioned`) |

None of them reads an aggregate: the caps read the household counts of the two regions
as limits of the world (4.4), as `ExitShare` does.

### 17.2 Scenarios proposed

For section 8 of the register; REF builds them (15.3). All use fixed seeds and report
the failing seed (testing.md 6.2).

| ID | Scenario | Asserts | Entries |
|----|----------|---------|---------|
| MIG-S1 | Two regions with the same structure, a yield shock on one set between ticks and cleared, the module on with the defaults | the households of the shocked region settle with no cycle: the range of each successive window of its household count shrinks or stays inside the margin (F6 not met); the households on the way to a region never exceed its pipeline cap; the number of journeys is within a stated factor of the minimum | MIG-A1, A3 |
| MIG-S2 | Three regions: a poor one and two equal ones | the arrivals divide between the equal ones within a stated band, and the difference of their household counts does not alternate in sign (F6) | MIG-A2 |
| MIG-S3 | MIG-S1 with relief on and off, same seeds | the households that start a journey differ by less than a stated band; the destination's pool is reported | MIG-A4 (L40) |
| MIG-S4 | A realm with an edge whose journey exceeds `CoverTicks`, a region that loses its income | the pushed households of that region are counted `unprovisioned`, the `Warn` "migration blocked" is emitted at the tenth tick, and relief and the ladder act on them | MIG-A7 |
| MIG-S5 | A region with a chain of three roles and a better neighbour | no role of the chain falls below a stated share of its households in a window (F3), or the run is reported with the counts; with `Fixed` on the chain's roles nothing leaves | MIG-A8 |
| MIG-S6 | The invariants over a range of seeds with `WithAudit`, journeys and a closure during a journey | money and each commodity are conserved exactly across departures, journeys and arrivals; the relocation totals reconcile a report by region; the household count by region rebuilt from the events equals the snapshot at every tick; a household is in one region at all times; the outcomes of every pass add to `Pushed` | MIG-A5, A9, all |

The existing scenario S14 (a region that loses its income; "the immigrants restore it")
now has its mechanism; S28 (a closure) gets a migration leg; S11 gets a journey leg
(MIG-A6). The provisional IDs this spec defines: loops MIG-A1 to MIG-A9, dampers
MIG-D1 to MIG-D7, scenarios MIG-S1 to MIG-S6.

## 18. Notes settled

For `AGENTS.md`: each open note tagged `MIG` or `ALL`, and the others that name
migration, is applied, rejected or left open here, with the text to append in the
handoff. The notes file is not edited by this spec.

| Note | Decision | Where |
|------|----------|-------|
| N-038 Seed prices, newcomers and migrants | **Applied; close.** A migrant is reseeded at arrival as any newcomer: for every commodity of its tradable set the centre is the clamp of the window mean of the new market (the price if the window is empty), width `BeliefWidthInit` (`spec-beliefs.md` 8.3). A migrant has no rumour state to keep | 8.4, 8.5 |
| N-046 Register entries owed by unwritten specs | **Leave open** for the other specs, with this added: MIG has its entries (`MIG-A1` to `A9`, `MIG-D1` to `D7`, `MIG-S1` to `S6`) | 17 |
| N-090 What MIG and the specs that tie obligations to a region must do with a relocation | **Applied for MIG; leave open** for PRO (adopt the reason `region_changed`), SKL and CAR or MER (register the refusals), INT (the public face). The job is abandoned at departure (`region_changed`), the beliefs reseeded at arrival, `Privation` kept with the grace of `RoleSince`, the traineeship left to the guard | 7.2, 8.2 to 8.5 |
| N-097 Migration and the population loop | **Applied for MIG; leave open** for POP's edit of its table 7. No money is created and no stake is paid (a migrant carries its account); `EntryPoolShare` is not shared because nothing is drawn; the replacement ratio is not a signal; the closed-region behaviour is unchanged | 5.5, 9 |
| N-104 Region-keyed tables use the one override mechanism | **Applied for MIG; leave open** for SUB and LND. `Regions` rows with `OutShare` and `InShare` by replace, canonical order, the errors of `spec-regions.md` 5.3 | 11.2 |
| N-105 Instant or timed migration | **Applied; close.** Timed, in the shape of TRN's traveller; the account stays in the origin; `NoRegion`, `depart` and `arrive` stay unused; death, ladder, relief and the window on the way are stated | 8.1, 8.3, 8.6 |
| N-108 What the resources module leaves to land, subsistence, infrastructure, the outside and migration | **Applied for MIG; leave open** for LND, SUB, CAP and EXT. Nothing in RES moves households; the signal of a depleted region is the household's own earned ratio and its measured wage; a destination's endowment is state and is not read | 4.4, 6.3.3 |
| N-110 Version 1 rumours prices only | **Applied for MIG; leave open** for SKL. A household does not react to a regional mean ratio; it reads its own earned figures and rumoured prices, and derives the destination's income from them and the structure | 4 |
| N-016 Keep the parameter registry true | **Applied**: rows for `Modules.Migration`, the nine fields, `Regions[]` and `Role.Fixed` (11, handoff b); no unnamed parameter | 11 |
| N-017 Words and names | **Applied**: journey, migrant, earned ratio, settled, margin, width, horizon, outflow cap, pipeline cap, provisioning; setting words only in the flagged examples | 3, 5 to 7, 15 |
| N-027 No default macro constant is stable | **Applied**: every default is a labelled placeholder and no claim of stability is made | preamble, 11, 16 |
| N-103 Running several spec writers in parallel | **Applied**: no shared file was edited, proposals are in the handoff, provisional IDs are used, the model ran in Docker with no bytecode | 17 |
| N-107 New money faucets and sinks start from MON | **Reject as not applicable**: the module creates and destroys no money or goods and adds no ledger reason | 9.1 |
| N-106 What a strategy may see of other regions | **Leave open** for MER, with this added: the module reads rumours through INF's function for the agent under review, not through a view, and shows a strategy nothing | 4.3.1 |
| N-111 A view takes draws | **Leave open**, with this added: the migration pass takes INF's draws for the pushed agents in phase 7, in ascending agent ID, destination and commodity (6.2.1) | 6.2.1, 10.2.1 |
| N-099 A trip is a gap in income | **Leave open** for WEL, with this added: a journey records `k` zero-income, non-working ticks; the settling rule waits a window; WEL may exclude them | 8.6 |
| N-100 What transport leaves to MER and INF | **Leave open**, with this added: a migrant is not a traveller; a traveller that is away is not a migrant; the guard respects `open_shipment` | 3.3.1, 7.2 |
| N-082, N-096 The regional pool | **Leave open**, with this added: the module draws nothing from a pool; the indirect effects are relief to arrivals (L63) and the money a migrant leaves on its death | 9.2 |
| N-022, N-079 External agents | **Leave open**, with this added: the default module never moves an external agent | 3.3, 10.3 |
| N-045, N-065 Dampers that act silently | **Leave open**, with this added: every limit of the module is counted in `MigrationReviewed` | 7.6, 13.1 |
| N-009 What the world and tick phases must decide | **Leave open**, with this added: arrivals in phase 1 after the transport arrivals, the migration pass in phase 7 after every region's entry step | 10.1 |
| N-012 Events, metrics and log keys | **Leave open**, with this added: the events `MigrationStarted` and `MigrationReviewed`, three metrics and the attribute `economysim.limit` | 13 |
| N-020 What each spec contributes to a checkpoint | **Leave open**, with this added: the journeys and two diagnostic runs | 14 |
| N-021 Anything that creates or destroys money or goods | **Leave open**, with this added: MIG adds no faucet, sink or reason | 9.1 |
| N-091 The world slots INT must expose | **Leave open**, with this added: MIG has no slot; a forced move is INT's call of the guarded `relocate` | 10.3 |
| N-030 Five regions | **Leave open**, with this added: weight 1 is the first consumer's case, so no cohort moves | 3.2 |
| N-015 Placeholders to calibrate | **Leave open**, with this added: the defaults of section 11 and every number of 15 and 16 | 11 |

Proposed new notes are in the handoff.

## 19. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs.

### Functional

| ID | Requirement |
|----|-------------|
| FR-MIG-001 | The module MUST be on if and only if `Config.Modules.Migration` is non-nil, MUST be written as the table `[modules.migration]`, MUST be part of the configuration hash, and with it off MUST emit no event of section 13, hold no journey and change no result (3.1). |
| FR-MIG-002 | `Modules.Migration` while `Modules.Information` is nil MUST be rejected with `ErrModuleDependency` (3.1, 12.1). |
| FR-MIG-003 | A field of section 11 outside its bound, a missing `MoveSpeed` included, MUST be rejected with `ErrOutOfBounds` naming its path (11, 12.1). |
| FR-MIG-004 | A `Regions` row naming an undefined region MUST be rejected with `ErrUnknownRegion`, two rows for one region with `ErrDuplicateID` and more than `MaxRegions` rows with `ErrTooMany`; an absent row MUST give the world's values with no warning; rows MUST be in region order (11.2). |
| FR-MIG-005 | An agent of a role with `Fixed` set, an external agent, an agent on a journey and an agent that is away on a transport trip MUST NOT be reviewed, and a mobile agent MUST be reviewed only on the ticks where `(tick + bucket(AgentID, MoveReviewTicks)) mod MoveReviewTicks` is 0, with the bucket of `spec-world.md` 9.1.1 (3.3, 5.1). |
| FR-MIG-006 | An agent MUST be settled at tick `t` if and only if `t - RoleSince >= max(MoveCooldown, T_a)` with `T_a` its effective welfare window; an arrival MUST set `RoleSince` to the arrival tick (5.2, 8.4). |
| FR-MIG-007 | The households of an agent whose turn comes, that is eligible, not settled and whose earned ratio is below `MoveBelow` MUST be counted in `Unsettled` and nothing else MUST be computed for them (5.2.3). |
| FR-MIG-008 | The earned ratio MUST be the welfare ratio of `spec-welfare-and-wages.md` 2.1 with the relief of the window taken out of the income, MUST be absent when the ratio is absent, and an agent MUST be pushed if and only if it is settled and eligible and the earned ratio exists and is below `MoveBelow` (5.3). |
| FR-MIG-009 | Two runs that differ only in the relief paid MUST give an agent the same earned ratio and the same observed wage for the decision, and the same decisions of the pass; no rule of this module MUST read POP's replacement ratio, a birth or a death (5.4, 5.5). |
| FR-MIG-010 | The decision of an agent MUST NOT read another region's wage, welfare ratio, income, stock, endowment, abundance, count of households, count of migrants or pool, nor an average of any: a test that changes each of these leaves every estimate, gate and ranking unchanged (4.4). |
| FR-MIG-011 | The candidates MUST be the regions other than the agent's that are not outside nodes and to which `Distance` is defined over the open edges, in ascending `RegionID`; and `K(role)` MUST be the upkeep commodities of the role in any region and the inputs, required tools and non-discard outputs of its recipes, in ascending commodity order (6.1). |
| FR-MIG-012 | The module MUST obtain a rumour for each source `(d, c)` with `c` in `K(role)` by INF's function with INF's lag, vintage and error, for an agent of a role whether or not `Informed`, MUST use `Mean` only when it is above 0, and MUST drop a candidate at stage 1 as `no_rumour` when an upkeep commodity of the household in it has no price or no recipe of the role has a price for all its commodities (4.3.1, 6.2). |
| FR-MIG-013 | With `RumourErrorBP` above 0 the module MUST take one draw per source pair for each pushed agent, in ascending agent ID and, within an agent, ascending destination and commodity, whether or not a rumour is offered, and with `RumourErrorBP` 0 or no pushed agent it MUST take none (6.2.1, 10.2.1). |
| FR-MIG-014 | The estimate MUST be `floor(net_r / Duration_r) * 10 000 / U_d`, truncated toward zero with 128-bit intermediates, with `net_r` as in 6.3 from the rumoured means and the table's yield at the season of the arrival tick, the greatest over the eligible recipes, 0 if there is none, and `U_d` the cost of the household's effective upkeep in `d` at the rumoured means (6.3). |
| FR-MIG-015 | The household's home figure MUST be its observed wage per working household-tick, without relief and grants, times 10 000 over the cost of its effective upkeep at the local report, and MUST be 0 when it had no working household-tick in its window (6.3.2). |
| FR-MIG-016 | The journey time MUST be `max(1, ceil(D / MoveSpeed))`; a candidate with `k >= MoveHorizon` MUST be dropped as `too_far`; and the discounted estimate MUST be `Vhat * (H - k) / H` truncated toward zero (6.4). |
| FR-MIG-017 | A candidate MUST be tested at stages 1 to 4 in that order (every price present; `k < H`; `Vhat >= EntryRatio`; `V_eff * 10 000 >= V_a * (10 000 + g_a)` by cross-multiplication), with the margin `g_a = MoveGain + floor(Offset(AgentID) * MoveWidth / 2^32)` and `Offset(id) = (id * 2 654 435 769) mod 2^32`, and the surviving candidates MUST be ranked by discounted estimate descending and, for equal values, by `RegionID` ascending (6.5). |
| FR-MIG-018 | A pushed agent whose role has no recipes MUST end in `no_estimate`; one with no candidate region in `no_destination`; and one whose candidates all fail the gates in the outcome of the latest stage any reached (3.3.2, 6.3.4, 7.6). |
| FR-MIG-019 | The agents with a ranking MUST be admitted in order of earned ratio ascending and, for equal ratios, agent ID ascending (7.1). |
| FR-MIG-020 | The guard MUST test the refusals of `spec-regions.md` 6.1.3 by the read-only query `RelocationRefusal` and a `region_bound` refusal MUST end the consideration with `bound`; the query MUST change nothing, draw nothing and allocate nothing, and `skip` MUST leave the named obligations untested (7.2, 7.2.1). |
| FR-MIG-021 | At most `ceil(Households_h * OutShare_h / 10 000)` households MUST start a journey from region `h` in a tick, with the households counted at the start of the pass including those on a journey; an agent that makes the count reach the cap MUST be admitted, a later one refused with `out_cap`; and `OutShare` 0 MUST mean none (7.3). |
| FR-MIG-022 | A candidate MUST be skipped as `unprovisioned` unless the agent holds, at the end of phase 6, `w * u_c * k` of each commodity `c` of its effective upkeep in its present region (7.4). |
| FR-MIG-023 | The households on a journey to `d`, those admitted earlier in the pass included, MUST NOT exceed `ceil(Households_d * InShare_d / 10 000)` when a household is admitted to it; the household that makes the count reach the cap MUST be admitted; a candidate over the cap MUST be skipped as `in_cap`; and `InShare` 0 MUST mean none (7.5). |
| FR-MIG-024 | The tests at admission MUST be, in order, the guard, the outflow cap, then for each candidate in its ranking provisioning and the pipeline cap, the first candidate that passes both being the destination; the outcome of an agent with none MUST be `unprovisioned` if all failed provisioning and `in_cap` otherwise (7.6). |
| FR-MIG-025 | In every pass, for every region, `Pushed` MUST equal the sum of `Started`, `NoEstimate`, `NoDestination`, `NoRumour`, `TooFar`, `NoViable`, `NoGain`, `Bound`, `OutCap`, `Unprovisioned` and `InCap`, all counted in households (7.6, 13.1). |
| FR-MIG-026 | Departure MUST, in this order, abandon a job in progress with the reason `region_changed`, end the subsistence activity of the agent, record `Journey = (To, tick + 1 + k)`, and the pass MUST then emit `MigrationStarted` for each departure in the order of admission, after the last departure and after the `RecipeAbandoned` events of the departures (8.2, 10.2); the arrival tick MUST NOT change if the graph changes later. |
| FR-MIG-027 | An agent on a journey MUST NOT be passed to `Work`, `Trade` or `Revise` and MUST place no order; its upkeep MUST be consumed in phase 6 from its own account with the effective upkeep of the region of departure; and its welfare record MUST receive income 0 and the origin's upkeep as non-working ticks (8.3). |
| FR-MIG-028 | An agent on a journey MUST remain in the region of departure for the relief, the ladder, the population module and every removal; a removal MUST send its holdings to that region's estate and drop its journey; a `Switch` or `Apply` of the spawner MUST be refused with `on_journey`; and no placement of an entrant MUST choose it (8.3). |
| FR-MIG-029 | In the first phase of a tick, after the arrivals of the transport module and before relief, the module MUST process in ascending agent ID each agent whose `Arrives` is the tick or earlier, after `RelocationRefusal(agent, To, "on_journey")` finds no refusal, by the ledger-level `Relocate` with the cause `migration` (8.4). |
| FR-MIG-030 | On a successful arrival the journey MUST be cleared, the beliefs of every commodity of the agent's tradable set MUST be reseeded by `spec-beliefs.md` 8.3, `RoleSince` MUST be the tick, and `AgentRelocated` MUST be emitted by REG with the cause `migration` (8.4). |
| FR-MIG-031 | An arrival for which `RelocationRefusal` finds a refusal other than `on_journey` MUST be held, the agent MUST stay on the journey, the test MUST be made again in the next tick, and the household MUST be counted in `Held` (8.4). |
| FR-MIG-032 | A move MUST leave the agent's role, weight, strategy, welfare record, target ratio, residues, want carries, `DeathCarry`, `Hunger` and `Privation` unchanged (8.5). |
| FR-MIG-033 | The total of money and of each commodity over all accounts MUST be unchanged by a departure, a journey tick (apart from the upkeep consumed under the existing reason `consumption`) and an arrival; the module MUST add no ledger reason, boundary account or flow; and neither region's estate account MUST change because of a move (9.1, 9.2). |
| FR-MIG-034 | The migration pass MUST run once per tick in phase 7 after the entry step of every region, and MUST consist of the counts, the evaluation in ascending agent ID, the admission, the departures and the events, in that order (10.1, 10.2). |
| FR-MIG-035 | `MigrationStarted` MUST be emitted once per departure, in the order of admission, after the last departure of the pass (10.2 step 5), with the fields of 13.1, and MUST contain no map. |
| FR-MIG-036 | `MigrationReviewed` MUST be emitted once per region per tick, in ascending `RegionID`, after the `MigrationStarted` events, when any household was pushed or held back by the settling rule, any arrival was held or any journey ended by a removal or was cancelled by `Relocate`, and never otherwise; its counts MUST be households (13.1). |
| FR-MIG-037 | The households on the way MUST be rebuildable from `MigrationStarted`, `AgentRelocated` with the cause `migration` and the `Ended` count of `MigrationReviewed` (13.1.1). |
| FR-MIG-038 | Construction MUST log each warning of 12.3 once at `Warn` with `rule` and `path`; the pass MUST log at `Warn` only the bounded "migration blocked" and "arrivals held" lines, at `Debug` per agent guarded by `Enabled`, and at no `Info` inside the tick (13.2). |
| FR-MIG-039 | The warnings `migration_without_routes`, `migration_no_mobile_role`, `migration_cohort`, `migration_provision`, `migration_far`, `migration_flow_window` and `migration_uncapped` MUST be raised as described, MUST NOT fail validation, and `migration_flow_window` MUST fire when `MoveOutShare` times the longest effective window of a mobile role exceeds 3 times `MoveGain` (12.3). |
| FR-MIG-040 | A mobile role for which `|K(role)|` times the number of other regions exceeds `MaxRumoursPerView` MUST be rejected with `ErrTooMany`, and a `Households * MoveOutShare` or `Households * MoveInShare` beyond `MaxQuantity` with `ErrOverflow` (12.2). |
| FR-MIG-041 | The checkpoint MUST contain the journeys, hosted in the agent record and saved there in ascending agent ID with destination and arrival tick (`spec-checkpoint.md` 4.2 tag 3), the two diagnostic runs per region and the pending `Ended` count per region, and a world restored between ticks MUST make the same decisions and draws as an uninterrupted one (14). |
| FR-MIG-042 | For a given seed and configuration the movers, destinations and arrival ticks MUST be identical on every run and platform, and independent of the order in which regions, edges, roles and `Regions` rows were listed (6.5.1, 7.1, 10.2). |
| FR-MIG-043 | A cohort MUST move whole; the caps MUST count households; and `migration_cohort` MUST be raised for a mobile role with a group scale above 1 (3.2). |
| FR-MIG-044 | Every field of section 11 MUST have a row in the parameter-owner table, a default (or "required") that passes validation and a failing-case validation test (11; testing.md 6.7). |
| FR-MIG-045 | The default module MUST NOT review, move or draw for an external agent, and a move that the consumer asks for MUST be INT's `Relocate`, which cancels any journey inside the call and moves any agent that is not a pseudo-agent (3.1.2, 3.3, 8.4.1, 10.3). |
| FR-MIG-046 | A candidate region that is an outside node MUST NOT be a destination, and a destination with no rumour for the commodities of the estimate MUST be dropped at stage 1 (6.1, 6.2). |
| FR-MIG-047 | With the module on, `on_journey` MUST be registered as a `region_bound` refusal open for an agent with a journey, so that a guarded relocation that does not skip it returns `ErrRegionBound` naming it and changes nothing, INT's `Relocate` being the one caller that skips it and cancels the journey; the account MUST stay in the region of departure until the module's own arrival or the cancel (7.2.2). |
| FR-MIG-048 | Cancelling a journey MUST drop the record, change no balance and no region, and count the households in `Ended`; a household moved by INT MUST have its journey cancelled inside the same atomic `Relocate`, not by a separate public call (8.4.1). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-MIG-001 | With the default logger and observer the migration pass of a step of the reference chain MUST allocate nothing beyond scratch held by the world (observability-and-logging.md 4.5, 8.2). |
| NFR-MIG-002 | The cost of a pass MUST be proportional to the agents whose turn comes and to the candidates of the pushed ones, and a tick in which no turn comes MUST cost no more than the counts of 10.2 step 1 (10.2.2). |
| NFR-MIG-003 | The rules of 5 to 7 MUST be integer arithmetic with the checked 128-bit helper, MUST use no floating point, and MUST live in `internal/migration` over primitive slices, importing the standard library only and knowing no region, commodity or role name (architecture.md 2.9, 3.5, 9.1). |
| NFR-MIG-004 | The estimate, margin, ranking and admission functions MUST have a fuzz test (testing.md 7.4) asserting that they never panic, that overflow is reported and not wrapped, that the outcomes of 7.6 partition the pushed households and that no cap is exceeded by more than one agent's weight. |

## 20. Open questions

1. **A replaceable mover.** The rule is a module with parameters (position 1). A consumer
   who wants another rule has nothing to implement. A new extension interface needs a
   row in architecture.md 4.3 and a default and assertion (4.5). Next step: only if a
   consumer shows a rule that data cannot express.
2. **Other signals.** Only prices are rumoured (4.3, N-110, INF 17.4). A second record
   kind keyed by (region, signal) with the same lag and error would carry a wage; whether
   an agent may react to a regional aggregate stays decided as no (4.4.1). Next step:
   only if a pack needs news that prices cannot give.
3. **Splitting a cohort.** A cohort moves whole (3.2). Moving part of it needs a split
   (`spec-scale.md` 4.8, 8) and the matching rule for the households that stay. Next step:
   when a consumer needs weights above 1.
4. **A carry limit.** A migrant takes its whole account (9.4, MIG-A9). A mass and a
   bulk capacity per household, `0` for none, and a refusal `overloaded` would close the
   leak. Next step: when TRN, STO or MER shows a case.
5. **A job in progress.** A departure abandons it (8.2). Deferring the departure to the
   job's last tick would save the inputs and delay the move by up to a duration; with
   long jobs it might never come. Next step: the scenario that shows the cost.
6. **Households with no recipe.** A household on relief, a grant or a trade has no
   estimate (6.3.4) and does not move by the rule. A rule for them needs a destination
   income that prices cannot give. Next step: EMP, and MER for its merchants.
7. **The season of the yield.** The estimate reads the table at the arrival tick's
   season (6.3). The mean over a year would be calmer for a household that arrives in the
   lean season. Next step: a helper of PRO or CAL if a seasonal pack shows the bias.
8. **The journey in the window.** WEL may exclude the ticks of a journey from the
   window as it may the ticks of a long trip (8.3.1, 8.6, N-099). Next step: WEL.
9. **A mode for a migrant.** The journey uses `MoveSpeed` and no vehicle or feed. A TRN
   mode (a self-carried mode with a speed, a crew and feed) would charge the vehicle's
   wear and the feed, and tie the module to transport. Next step: if a pack needs
   migration by wagon or ship.
10. **A scout.** A household that visits a place before it moves would learn its true
    price. A trip carries no news back (`spec-information.md` 7.5, 17.1). Next step: MER,
    if it asks for the memory.
11. **Buying provisions.** A household that cannot be provisioned could buy first and go
    on a later review. Today it waits for the next review with whatever it then holds.
    Next step: a scenario that shows the poverty trap to be too costly (MIG-S4).
12. **Population caps.** POP's `HouseholdCap` limits births (`spec-population.md` 4.4.2)
    and not immigrants. Whether `InShare` of 0 or a cap on arrivals should follow it is
    left to the pack by a `Regions` row. Next step: POP and a scenario.
13. **A forced move.** A flight from a disaster is a shock that moves households at
    once. It is INT's call of the guarded `relocate` with a cause of its own
    (`spec-regions.md` 6.6); the module has no such trigger. Next step: INT.
14. **Chain migration.** A household attracted by the presence of its own kind
    would read a count of households in the destination: an aggregate (4.4). Not
    designed; if needed it enters as news (a rumour kind), not as a count.
15. **Calibration.** Every default is a placeholder (N-015). The bands for the flow,
    the settling time and the imbalance that remains (16.5) are STY's; the defaults
    are CLB's.
16. **Cancelling a journey.** Answered: the cancel has no public name; it is a step inside
    INT's `Relocate` (`spec-integration.md` 4.5.2, 4.5.3; 8.4.1).

## 21. Where this lives in the code

| Concern | Home |
|---------|------|
| The module's configuration (`Migration` and its nine fields, `RegionMove`, `Role.Fixed`), the `Journey` record of an agent, the events `MigrationStarted` and `MigrationReviewed`, the migration pass and the arrival step | root package, `migration.go` (the events in `event.go`) |
| The estimate, the journey time, the discount, the margin and its offset, the stages and the ranking, the admission against the caps, the outcomes, over primitive slices | `internal/migration` (standard library only, converted inline at the call, architecture.md 2.9) |
| The rumour for the agent under review | `internal/rumour` (INF), called from the pass |
| The bounds of 11 (all existing) and the derived checks of 12.2 | root package, `bounds.go`, `validate.go` |
| The warnings of 12.3 | root package, `validate.go` |
| The reason and outcome constants (`migration`, `region_changed`, `on_journey`, `no_estimate`, `no_destination`, `no_rumour`, `too_far`, `no_viable`, `no_gain`, `bound`, `out_cap`, `unprovisioned`, `in_cap`, `held`, `ended`) | exported constants of the root package, `event.go`; `internal/logkeys` holds the key strings only |
| The file shape (`[modules.migration]`, `[modules.migration.region.<id>]`) | `config/` (`spec-config.md` 6) |
