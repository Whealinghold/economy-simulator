# Spec: Saving and investment: a savings target above the reserve, a precaution buffer taken from the household's own cash cycle, a project test for a build or a repair, and the hoard as a measure

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-SAV-NNN` and
`NFR-SAV-NNN`, which tests cite (testing.md section 4). They are in section 18.

Layer: module ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md),
decision 2), later stage (`vision.md` section 4, "Later"). It is off unless
`Config.Modules.Saving` is present, and with it off nothing in this spec runs:
no target, no project, no event, no state, and the default strategy is exactly
the one of `spec-agents.md` 7. With it on and no role carrying a `Saver` block
the world is the world without it (FR-SAV-001). The core does not depend on it.
It needs `spec-agents.md` (AGT: the reserve and the spend cap of 7.4, the planned
recipe of 7.2, the work rule of 7.1, the views of 6.4), `spec-infrastructure.md`
(CAP: a facility, the build and repair clauses, the refusals; written beside it),
`spec-welfare-and-wages.md` (WEL: the ring of records that the buffer reads),
`spec-world.md` (WLD: the tick and its devices), and `spec-config.md` (CFG). It
reads `spec-demand.md` (DEM: the cap funds wants too), `spec-credit.md` (CRD: why
a project is self-financed), `spec-money-supply.md` (MON: the holding charge and
the identity), `spec-employment.md` (EMP: an employer's profit), `spec-land.md`
(LND: why a title is not bought here), `spec-storage.md` (STO) and
`spec-indicators.md` (IND: the measure). Every change it needs in another file
is in the handoff (section 17) and is not made here.

**Every number in this spec is a placeholder** except where a derivation is
shown. Nothing has run in the library, bazaarBot has not been run, and no
acceptance band exists (notes N-015, N-027). The scripts of section 15 are
stylised Docker Python models of the rules of this spec, not runs of code and
not a claim that any default is stable; the rules themselves are integer. No
default is set from a source: no historical fact about saving, hoarding or
investment was read or used (N-018), and every default is structural (no
precaution, no repair, a hurdle above 1) or a placeholder. The settings of the
worked examples (section 16), a pre-industrial realm and a space colony, are
**examples**, flagged as such. The normative text and its vocabulary are
setting-agnostic: `mill`, `grain`, `timber`, `cold cell` appear only in the
examples. A commodity, a recipe, a role or a facility class is data
(architecture.md 9.1): nothing in this spec branches on one.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2).
"Consumer" means the library consumer (a game or server), never a household
(N-001, N-017). The words **saving** and **savings** mean money a household
holds back from spending; **investment** means a build or a repair of a facility
(`spec-infrastructure.md`) and nothing else (a purchase of a title or a claim is
not investment here, 6.3); **hoard** is used only for the measure of 8.

## 1. Purpose and scope

Four specs point here for the same few things, and the household cannot do
them without it. `spec-agents.md` 7.4 has a **reserve** and a concave cap on
discretionary spending, so a household saves a little, by one fixed rule; it
has no reason to save for a purpose and no reason to hold more when its income
is uncertain. `spec-infrastructure.md` 4.5 builds nothing by default and leaves
"whether, when and where an agent decides to build or repair; the margin test,
the saving that funds it" to this spec. `spec-employment.md` 11.3 closes an
employer's circuit with "investment (the employers build): profit becomes demand
for goods made by workers". `spec-land.md` 6.2.1 and `spec-credit.md` 1 leave "a
decision to save, to hoard or to invest" here. This spec supplies the one
mechanism, and nothing more. It covers:

- what AGT's saving rule already does, what it does not do, and the **savings
  target** that replaces its reserve in one place (section 3);
- the **precaution buffer**: why a household holds more when its income is
  uncertain, measured on its own cash cycle (section 4);
- the **project**: how a household decides to build or repair a facility, the
  **margin test**, the **goal** that funds it and the request (section 5);
- how a project is **financed** and why the default asks for no credit, and why
  a title is not bought here (section 6);
- what the player and a consumer's strategy see and may do (section 7);
- **hoarding**, money velocity and the measure of idle money (section 8);
- the positions in the tick, the draws (none) and the cost (section 9);
- parameters, validation, observability, saved state (none), the register
  entries, the models run, worked examples, notes settled, requirements, open
  questions and the home in the code (sections 10 to 20).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| What a facility is, how it is built and repaired, its effects, its decay, the refusals of a build | `spec-infrastructure.md` (CAP); this spec decides **when** a household asks for one |
| A loan, interest, default; a loan to fund a project | `spec-credit.md` (CRD); 6.2 says why the default asks for none |
| Where money comes from, the holding charge that damps a hoard | `spec-money-supply.md` (MON) |
| A treasury that builds, a tax, a toll | `spec-fiscal.md` (FIS), `spec-policy.md` (PLC) |
| A purchase of a title, of a claim, of a facility for a price; a market for any of them | open questions 2 and 3; `spec-land.md` 6.3 and `spec-infrastructure.md` 11.3 give the transfers a later decision would call |
| Productivity growth from capital | `spec-technology.md` (TEC): a facility is not a technology; a gain of TEC raises the effective yield of a recipe, so `net_r` of 5.3.1 already includes it (`spec-technology.md` 5.3) |
| An interest rate on savings, a bank deposit, a safe asset that pays | not modelled: no faucet may be proportional to a balance (`spec-money-supply.md` 9.5), and a deposit is a claim on a lender (`spec-credit.md` 3.4) |
| Time preference, utility, expectations of the future | not modelled: the household reads its own state and the prices it expects (`spec-agents.md` 7.1.4), as everywhere in the library |
| The prices, the wage rule and the welfare ratio | `spec-clearing.md`, `spec-welfare-and-wages.md` |

## 2. Positions taken

The design pressures of the brief, each with a position and the reason. Later
sections give the rule.

| # | Question | Position | Why |
|---|----------|----------|-----|
| 1 | Why does a household hold money back? | Three reasons, and a fourth that is not a reason. **The reserve** (`ReserveTicks` of upkeep cost, AGT 7.4.2): already there. **A precaution buffer**: a share of the deepest hole its own cash flow has dug in its welfare window. **A goal**: the outlay of a project it has decided to take. **Idle money** is what is left above all three, and is a measure and not a decision | The brief asks for precaution and a target buffer; the first is the reserve, the second is new, the third is what turns saving into capital. The fourth is what the concave cap leaves, which AGT does not call a decision (3.1, 8.1) |
| 2 | What does AGT's rule not do? | It holds a flat reserve whatever the household's income looks like, and it saves for nothing. A household with a 20-tick job has the same reserve as one that is paid every tick (4.3). It has no purpose for savings | `spec-agents.md` 7.4, 11.5: "a long job cannot be funded by credit" and N-032 |
| 3 | Where does a target enter the rule? | In **one place**: the `e` of the cap (AGT 7.4.4) is the money left above the **savings target** `S*` instead of above the reserve. Tiers 1 and 2, the survival rule and the work share still read the flat reserve | The goal must stop buffers and wants, not food or inputs: a rule that withheld food or a job's inputs from a household saving for a mill is a poverty trap (AGT 7.4.3) |
| 4 | Precaution against what? | Against **its own** income stretches: the **drawdown**, the deepest fall from a peak of the cumulative `income - cost` over its window, read from the welfare ring. No distribution, no expectation, no aggregate | Local information only (`spec-stability.md` 5.1.3, `spec-world.md` 9.4.1); an integer, and it gives a long-job producer the working capital of its cycle (4.3) |
| 5 | How does an agent decide to build or repair? | A **project test** in the default strategy for a role with a `Saver` block: the benefit, in money, of the extra service of a facility over a payback horizon must exceed its cost by a **hurdle**. The benefit of a **site** is computed from the household's own recipes that need it; the benefit of any other kind is a number a pack states in a `Worth` row | A margin test as for MER (`spec-merchants.md` 5.5); a site is the one effect that accrues to its owner by construction (CAP 5.1.5), and the others accrue to others (CAP-A2), so the engine cannot value them (5.3) |
| 6 | Is it a second strategy or the default? | **The default**, for a role with a block, as DEM's tier 4 and CRD's request are. No Go registration, so a pack that builds its own capital needs configuration alone | FR-CAP-045 (a facility chain from a file alone); a wrapper would need the one Go line of MER 3.1.2 and could not alter the cap that sits inside the default |
| 7 | State? | **None.** The target, the goal, the test and the request are pure functions of the view of the tick. A household that holds the inputs of a project is recognised by its stock (the **commit band**, 5.5.3) and a project in progress by its `Job` (5.5.6) | No checkpoint section, no per-agent field (N-128), nothing to restore; the cost is a hysteresis that is a function of holdings and not of memory |
| 8 | How is a project funded? | By the household's own money: **funded** when `Money` is at least the outlay plus the floor of the target. A funded household plans the project, so the default's tier 2 buys the inputs; the default borrower of CRD then finds no gap and asks for nothing | A bullet loan that falls due at the end of a build, with no income from the facility yet, is the debt squeeze of CRD-A1 (6.2); self-financing is the safe default and a consumer may do otherwise |
| 9 | When does a household act? | Every tick: the test is a pure function and the request is made when the inputs are on hand. **No review cadence**: a gate on the request would starve a role whose job is longer than the gate (the model of 15) | Deletion over addition; the herd is damped by the spread of the hurdle and the dispersion of funds (5.5.2) |
| 10 | Is there a position, a draw, a state, a reason, a flow? | No to a draw, a state, a reason and a flow, and its one position adds no step. The module acts inside `Work` (P2.5) and `Trade` (P3.2) and adds one aggregate event at P3.6b, a letter of its own after STO's P3.6a, which runs nothing else | A module that adds a position needs an edit to the one authoritative tick (`spec-world.md` 6.1.10); this one's P3.6b is a row of that table (6.1.1 there) that emits an event and runs nothing |
| 11 | Does saving create income? | No. Saving is not income and investment is not output: `Σ spending = Σ income` holds in every tick (8.2). Saving moves money into balances and lowers the spending that would have been somebody's income | MON 8, N-054: a closed economy's identity holds; SAV adds no faucet and no sink |
| 12 | Who is exposed to the paradox of thrift? | Everyone above the reserve, bounded: the buffer and the goal act on the part of the cap that is a bounded function of the excess, so the cut in discretionary spending is at most `SpendMaxShare * SpendHalfTicks * UpkeepCost / 10 000` per household and tick (4.5) | The amplifier is a pulse and not a ratchet; the model shows a trough that does not deepen (15) |
| 13 | Is there a measure of hoarding? | Yes: **idle money**, `max(0, Money - S*)`, per role and region in the snapshot, and an IND preset that reads it with the velocity of MON 8.4. No rule reads it | Macro is measured, never set (ADR 0001 decision 7); STB's regime F2 needs it |
| 14 | What does the player get? | The same rules as any agent of its role, through the fallback; a consumer that wants the player free gives it a role with no block. A query returns the target and the project of an agent (7) | `spec-integration.md` minimal (N-131): one query, no call |
| 15 | New loops | Yes, seven, in section 14: precautionary saving as an amplifier, the goal as a pulse, the investment cobweb (the decision side of CAP-A1), flapping, the replacement echo, concentration by self-financing, and a hoard with no outlet | `spec-stability.md` 9: a spec that adds a loop and no entry is not done |
| 16 | Randomness | **None.** A per-agent spread is the Weyl value of the ID, with no creation draw | WLD 7.4.2, 9.1.1 |

## 3. What the default does now, and what this adds

### 3.1 The reserve and the cap today

3.1.1. The default strategy computes, for an agent with upkeep cost `UpkeepCost`
(AGT 7.4.2):

```
ReserveMoney = ReserveTicks * UpkeepCost
```

Tiers 1 (upkeep) and 2 (the inputs and tools of the next job) spend whatever
money allows and are **not** subject to saving (AGT 7.4.3). After them, with
`avail` the money left (AGT 7.4.4):

```
e   = max(0, avail - ReserveMoney)
H   = SpendHalfTicks * UpkeepCost
cap = SpendMaxShare * ( e * H / (e + H) ) / 10 000
```

Tiers 3 (buffers) and 4 (the wants of `spec-demand.md`, with the module on) share
the cap. The cap is increasing and concave in `e`, zero at `e = 0` and bounded by
`SpendMaxShare * H / 10 000`.

3.1.2. **What that saves, and what it does not.** A household below the reserve
buys no buffer and no want, so it rebuilds the reserve first (AGT-A4). A
household above it spends a decaying share of the excess, so what it holds
settles where the cap equals its surplus income. Three things are not in the
rule: the reserve is the same for a household with a steady income and one whose
income stops for twenty ticks; nothing makes a household save **for** anything,
so a facility that costs forty ticks of upkeep is not a thing it can plan toward; and
nothing in the library asks what a household with a large surplus does with it
(the cap is bounded, so the balance grows without limit, EMP-A3).

3.1.3. **What the rule is good at, and is kept.** It is a function of the
household's own state, it is integer, it never starves a tier 1 or tier 2
purchase and it never touches the reserve. All three are kept. This spec adds to
it and replaces nothing but the `ReserveMoney` inside `e`.

### 3.2 The module, the block and the target

3.2.1. The module is present when `Config.Modules.Saving` is non-nil
(`spec-config.md` 3.2: no separate flag). A role takes part when its
`Role.Saver` block is non-nil:

```go
// Saving is the module of saving and investment.
type Saving struct {
	SafetyTicks    int64 // ticks of upkeep cost added to the outlay of a project beyond its duration
	HurdleSpreadBP int64 // the half-open spread of the hurdle by agent, in the units of the hurdle
	CommitBandBP   int64 // the fall of the hurdle for a household that already holds the inputs
}

// Saver is the block of a role whose households save for a purpose and invest.
// A nil block is a role with the reserve of AGT 7.4 and no project.
type Saver struct {
	PrecautionShareBP  int64          // share of the drawdown held as a buffer; 0 is none
	PrecautionCapTicks int64          // the buffer is at most this many ticks of upkeep cost
	HurdleBP           int64          // the return a project must give per unit of its cost
	PaybackTicks       int64          // the horizon over which a facility's service is counted
	RepairBelowBP      int64          // a facility is considered for repair at or below this share of its durability; 0 never
	Worth              []ProjectWorth // per facility, per tick, for a class the rule cannot value
}

// ProjectWorth states what a whole facility of a class is worth to the role
// per tick, in minor units. It is a pack's assertion (5.3.2).
type ProjectWorth struct {
	Class   string // a facility class
	PerTick int64  // 1 to MaxWorthPerTick
	Count   int64  // the facilities of the class, held by one household, up to which the row counts; 1 to MaxWorthCount
}
```

In a file they are `[modules.saving]` and `[role.<id>.saver]`, with
`[[role.<id>.saver.worth]]` for a row (the form of `[modules.merchants]` and
`[role.<id>.merchant]`, N-102). The fields are in section 10; the block is
**data on the role** and is in the effective configuration and its hash.

3.2.2. **The savings target.** For an agent whose role has a block:

```
Floor   = max( ReserveMoney, Precaution )                 (4.2)
Goal    = max( 0, Outlay_p - HeldValue_p - Tier2_p )       for the project p chosen in 5.5.4 (5.6.3)
        = UpkeepCost * ( Remaining + 1 + SafetyTicks )     while a job of a project recipe is in progress (5.5.6)
        = 0                                                otherwise
S*      = Floor + Goal
```

and `S*` replaces `ReserveMoney` in the `e` of 3.1.1 and nowhere else:

```
e   = max(0, avail - S*)
```

For an agent whose role has no block, or with the module off, `S* = ReserveMoney`
and nothing changes (FR-SAV-001). `Outlay_p` and `HeldValue_p` are in 5.6.

3.2.3. **Why one place, and why the goal is not added to the reserve of the
survival rule.** `ReserveMoney` is read in the spend cap (AGT
7.4.4), in the survival rule that sets the work share to 10 000 (AGT 7.1.2, 7.1.3:
`Money < ReserveMoney`), in the warning `reserve_exceeds_money`, in the merchant's
trip planner (`spec-merchants.md` 6.2 and 8.2: `q_aff` and `afford`, what a trip
can pay) and in the budget values of the demand module (`spec-demand.md` 4.1 and
4.5, which follow the cap). A goal added to the survival rule would set a
household saving for a facility to work full time because it is "below its
reserve", which is not what the survival rule is for (survival comes before the
target, and a household saving up is not short). The goal and the buffer
therefore act on the cap only; the work share is still the target-income rule of
`spec-welfare-and-wages.md` 5, and saving is funded from the surplus that rule
leaves (3.3.3). The merchant's planner keeps the flat reserve too, so a saver
merchant's goal and buffer are invisible to its own trip planner: a trip is
planned from the money above the flat reserve, which is its working capital and
not a saving.

### 3.3 What the target does and does not change

3.3.1. **It changes** the cap of tiers 3 and 4 and nothing else in the budget: a
household below `S*` buys no buffer and no want. Tier 4 shares the cap
(`spec-demand.md` 4.5, FR-DEM-020), so a saver also buys fewer wants while it
saves; that is the point, and it is the paradox of thrift in its local form
(SAV-A1, 14).

3.3.2. **It does not change** tiers 1 and 2 (food and a job's inputs are bought
whatever the target), the production halt, the recipe choice, the survival rule,
the work share, the favourability of a buffer, the limit price, beliefs, or any
order's validity: the hard constraint of AGT 7.4.1 holds, since `S*` only lowers
the cap (FR-SAV-005).

3.3.3. **It does not create surplus.** A household saves from what is left after
its tier 1 and 2 spending. In a world where every household's income equals its
spending (the closed economy with upkeep only, STB-L46) there is none, and a goal
is never reached: the module redirects a surplus that exists. The target-income
rule gives a household a surplus when its target ratio is above 10 000 and the
economy can meet it (`spec-demand.md` 7, `spec-employment.md` 11): the savers
are the households of the roles that earn above their cost.

3.3.4. **A cohort.** The target, the buffer and the goal are totals of the agent
(weight included), like `UpkeepCost` and the reserve. Of the outlay, the
upkeep term scales with the weight (`UpkeepCost` is the cohort's); a build asks for
one facility whatever the weight (5.1.3), so its inputs are one execution's and
not `w` times; a repair's are those of `m_plan` executions (5.8). The precaution
is that of the sum of the cohort's ring (which is a total,
`spec-welfare-and-wages.md` 2.1).

## 4. The precaution buffer

### 4.1 The drawdown

4.1.1. Each agent keeps a ring of per-tick records for its effective window `T`
(`spec-welfare-and-wages.md` 2.1, 2.3.3). Record `i` holds `income_i`, which nets
the cash spent on inputs and the wear of tools, and `cost_i`, the cost of the
tick's effective upkeep. With `n` records, oldest first, `n` at most `T`:

```
net_i      = income_i - cost_i                      signed, checked
C_0        = 0;   C_i = C_(i-1) + net_i
Peak_0     = 0;   Peak_i = max(Peak_(i-1), C_i)
Drawdown   = max over i of ( Peak_i - C_i )          at least 0
```

The **drawdown** is the deepest fall from a running peak of the cumulative
`income - cost` in the window: the money the household would have needed to hold,
at its worst point, to go through the stretch without running down below where it
stood. It is 0 for a household that never ran a deficit in the window.

4.1.2. **Who computes it.** The world, as a pure function of the ring
(`internal/welfare`), for each agent whose role has a block, when it builds the
view, as it computes the floor count and the halt count (AGT 7.1.3, 7.1.5). A
consumer's strategy cannot hide or change it, and it is a function of saved state
(FR-OBS-043). The ring is saved state of WEL, so nothing is added.

4.1.3. **What it sees.** Because `income` is net of the cash spent on inputs, a
producer's own cycle is in it: the inputs bought at the start of a job are a
negative `income` in that tick, and the sales are a positive one at its end. And
because the effective window of a role is at least `WindowCycles` times its
longest job (`spec-welfare-and-wages.md` 2.3.3), the window spans two cycles of
the longest job on the role's list, or of its trips: the drawdown of a long-job
role sees a whole cycle (example B, 16.3).

4.1.4. **Cost.** `O(n)` per agent with a block per tick, with `n` at most
`MaxWelfareWindow`. At the default window of 30 and a first consumer's 5 000
agents that is 150 000 additions a tick, which the first slice measures (N-015);
a running maximum cannot replace it exactly because the window slides, and an
incremental form is open question 3.

### 4.2 The buffer

```
Precaution = min( PrecautionCapTicks * UpkeepCost,  PrecautionShareBP * Drawdown / 10 000 )
Floor      = max( ReserveMoney, Precaution )
```

with `UpkeepCost` the agent's own (AGT 7.4.2), the products in 128 bits and
truncation toward zero (the rule constant, architecture.md 3.5). With
`PrecautionShareBP` 0, the default, `Precaution` is 0 and `Floor` is the reserve:
the buffer is off. **The flat reserve is the floor of the buffer**: the buffer
only ever raises the target, never lowers it.

### 4.3 Reading it

4.3.1. **A steady income.** A household paid every tick above its cost has
`Drawdown` 0, so `Floor` is the reserve: AGT's rule, unchanged.

4.3.2. **Stretches.** A household whose income has bad stretches (a seller of
something seasonal or lumpy; a household between trips) digs holes. Its drawdown
is the worst of them it has lived through in the window, so its buffer is a share
of what its own history says it needs.

4.3.3. **A long job: the working capital of the cycle.** A producer with a job of
20 ticks, inputs of 300 at the start, upkeep of 100 a tick and sales of 2 500 at the
end has a drawdown of 2 200 over a window of 40 (16.3). With a share of 10 000
its floor is 2 200, against a flat reserve of 300: it holds the working capital
of its cycle before it buys a buffer or a want. This is the household's own
remedy for the gap that N-032 named, that AGT-A12 registers and that AGT open
question 15 asks about ("the reserve for a long job"), and that
`spec-credit.md` 10.2.3 fills with a loan: they are substitutes. A household with a buffer needs no loan (the default
borrower's gap, `spec-credit.md` 10.2, is `UpkeepCost * H + Tier2 - Money`, and
`Money` is high); a household with no surplus cannot build the buffer and borrows.

### 4.4 What it is not

Not an expectation: there is no forecast, no estimate of a variance and no
probability, so no draw. Not a response to prices, to the region or to any
aggregate (`spec-world.md` 9.4.1). Not a propensity to save out of income: the
share of income spent is the cap's. And not instant: a household that has not yet
run a hole has none, and learns only by living through one, so a young
household has the flat reserve.

### 4.5 Bounds on its effect

4.5.1. The buffer moves `S*` and so only lowers the cap, and the cap is bounded:
the cut in discretionary spending of one household in one tick is at most the
cap itself, `SpendMaxShare * SpendHalfTicks * UpkeepCost / 10 000` (500 at the
defaults of `spec-agents.md` 12 and an upkeep cost of 100). Tiers 1 and 2 are
untouched, so an income fall is never deepened by withholding food or inputs
(FR-SAV-005).

4.5.2. **It decays.** The drawdown leaves the window `T` ticks after the stretch
that made it, so the buffer is a pulse of at most `T` ticks after a deficit and
then returns to the reserve (the delay of SAV-A1).

4.5.3. **It is capped.** `PrecautionCapTicks` bounds it in ticks of upkeep cost
(SAV-D3).

4.5.4. **What the model shows** (15.1). One household with an income of 160 or 30
and an upkeep of 100: unfed ticks fall from 2 410 to 2 333, 2 141 and 1 862 of
20 000 at shares 0, 5 000, 10 000 and 20 000 (cap 30 ticks), the mean money held
rises from 410 to 576 and the mean discretionary spending falls from 36 to 34. A
representative household in a circular flow with a leak, whose autonomous income
stops for 30 ticks and for 60: the trough of discretionary spending is 0 at every
share and the time to 90% of the old level after the cut ends is 60 ticks at
shares 0 to 10 000 and 58 at 20 000 (62 at every share in the longer cut). The
household is limited by rebuilding its flat reserve, which the buffer does not
outlast, because the drawdown has left the window by then. So at the level of the
aggregate the amplifier is weak where the flat reserve already binds, and it acts
where households sit between the reserve and the buffer, bounded by the cap. It is
a model of the rule in an invented flow, not of the library.

## 5. Projects

### 5.1 What a project is

5.1.1. A **project** of an agent is a recipe of its role's list with a `Builds`
or `Repairs` clause (`spec-infrastructure.md` 4.1). Its **class** is the clause's
class and its **kind** is the class's kind (`site`, `route`, `ground`, `store`). A
role has the projects its list gives it: there is no new object and no list of
projects in the block. A role with a block and no such recipe has no project and
only the buffer (the warning `saver_no_project`, 11.2).

5.1.2. **What the rule can value and what it cannot.**

| Kind | What the benefit is | Source |
|------|---------------------|--------|
| `site` | the increase in the value of the household's best job when its capacity rises: computed from its own recipes that `Needs` the class (5.3.1) | the household's own beliefs and recipes |
| `route`, `ground`, `store` | `PerTick` of a `Worth` row of the role, per whole facility, for the ticks of service | a pack's assertion (5.3.2) |
| any kind with a `Worth` row | the row's `PerTick`, which replaces the computed value of a site, **for a household that holds fewer than the row's `Count` facilities of the class**; at `Count` held, 0 (5.3.2) | a pack's assertion |
| a class with no row that the rule cannot compute | **0: the project is never chosen** (counted `NoReturn`) | |

5.1.3. **One facility a project.** A build asks for one facility
(`Executions` 1), whatever the household's weight, and a household starts the
next project the next time one passes and it is funded: for a computed site while
`dJ` is above 0 (the capacity already held lowers it, and it reaches 0 at the
weight), for a class with a row while the household holds fewer than the row's
`Count`. A repair asks for the executions the world derives (`MaxExecutions`,
`spec-infrastructure.md` 4.2.2 limit 6).

### 5.2 The `Saving` group of the view

5.2.1. The view of an agent whose role has a block (the `Work` view and the
`Trade` view, `spec-agents.md` 6.4.1) gains one group of plain values, valid for
the call only:

| Field | Meaning |
|-------|---------|
| `Block` | a copy of the role's `Saver` block |
| `SafetyTicks`, `HurdleSpreadBP`, `CommitBandBP` | the module's values, so that a strategy need not know the configuration |
| `Drawdown` | the drawdown of 4.1, computed by the world |
| `Facilities` | the agent's own facilities in its region, ascending ID (`spec-infrastructure.md` 9.1, `FacilityView`); present in the `Trade` view too, which CAP 9.1 leaves without it (an owed edit). It is built from the **records**: a facility completed at P2.1 (P2.7 for a one-tick job) is in it in the same tick's `Work` and `Trade` views with the `Capacity` of its record, though its effect is in the table only from P8.3b (`spec-infrastructure.md` 4.3.3; 5.5.6) |
| `Classes` | for each class named by a clause on a recipe of the role: its kind, `Durability`, `Decay`, `WearPerUse`, `Capacity` |

5.2.2. The per-recipe group of the view carries each recipe's `Needs`, `Builds`
and `Repairs` (`spec-infrastructure.md` 4.1.1; an owed edit to AGT 6.4.1), besides
the `MaxExecutions` and the reason when it is 0 that it already has (AGT 6.4.1), and
for a recipe with a `Builds` or `Repairs` clause one more field, `Standing`: the
first refusal of `spec-infrastructure.md` 4.2.2 other than `no_input` and
`no_tool` that would apply were the inputs and tools held, or empty (an owed edit
to AGT 6.4.1 and CAP 4.2.2; 5.5.1 reads it, not the reason of `MaxExecutions`,
which the first reason `no_input` masks). The `Job` group says whether a job is active, its recipe
and `Remaining` (5.5.6). The
view carries no other agent's facility, no count of facilities of a region and no
figure of any market of capital: a strategy reads what the household has and what
it believes (the reasoning of N-106).

### 5.3 The benefit

5.3.1. **A site: the household's best job with and without the capacity.** For a
recipe `r` of the role's list the **value per tick**, scaled by 10 000 so that
two values are compared and subtracted in integers, is

```
pv_r = trunc( net_r * 10 000 / Duration_r )          net_r of spec-agents.md 7.1.4, at the agent's centres
```

A recipe is **valued** when it is eligible by AGT 7.2 ignoring stock: a positive
yield, `net_r > 0` or a source, and not halted. For a site class `c` let `B(c)` be
the valued recipes of the role that `Need` `c`, and `A(c)` the valued recipes
that do not need `c` and whose other needs the agent already holds with a
capacity above 0 in its region. Then

```
pv_b = max over B(c) of pv_r           (0 if none)        its recipe is b*
pv_a = max over A(c) of pv_r           (0 if none)
J(c, K) = max( pv_b * min(w, K),  pv_a * w )              the best job value per tick of an agent of weight w
                                                          with K executions of capacity of class c
```

An agent runs one job at a time and a job runs `k` executions in parallel
(`spec-production.md` 5.1.2), so the value of its best job is the better of the
job that needs `c` (limited to `K` executions) and the best job that does not. Let
`cap_now` be the sum of `cap(f)` over the agent's facilities of the class in its
region (`FacilityView.Capacity`). A new facility of the class adds `Capacity_c`
executions at full condition:

```
dJ_build = J(c, cap_now + Capacity_c) - J(c, cap_now)              at least 0
k        = min(w, cap_now + Capacity_c)
rate     = Decay_c + ceil( WearPerUse_c * k / Duration_b* )         condition units a tick
T_p      = min( floor( Durability_c / rate ), PaybackTicks )         PaybackTicks if rate is 0
Benefit  = dJ_build * T_p / 10 000                                   minor units, truncated
```

For a weight-1 household with no site, `dJ_build` is `max(0, pv_b - pv_a)`: the
gain per tick of being able to run the recipe that needs the site over its best
alternative. `T_p` is how long the facility serves, at most the payback horizon:
the rule never counts a return beyond the horizon or beyond the life of the
facility (SAV-D6).

5.3.2. **Another kind, or a site with a row.** With a `Worth` row for the class,
and `n_held` the facilities of the class that the agent holds in its region (the
`Facilities` of the view, ruined ones are gone):

```
rate     = Decay_c + WearPerUse_c * min(w, Capacity_c)              a duration of 1 tick is assumed
T_p      = min( floor( Durability_c / rate ), PaybackTicks )         PaybackTicks if rate is 0
Benefit  = PerTick * T_p        if n_held < Count
         = 0                    otherwise
```

With no row, a class that is not a site has `Benefit` 0. **Why `Count`.** The
gains of the effects add and are capped (`spec-infrastructure.md` 5.2.2, 5.4.3, CAP-D3),
so the second facility of a class is worth less than the first and a row with no
saturation would tell a household that every further one is worth the same until
the world's per-agent bound refuses it (`facility_cap`, a standing refusal, 5.5.1).
`Count` is the pack's statement of how many whole facilities of the class the row
is true for, a placeholder like `PerTick`. A repair is not limited by it (5.8).

5.3.3. **Why the engine values a site and not a route.** The owner of a site is its
only user (`spec-infrastructure.md` 5.1.5), so its benefit is in the owner's own
recipes. The gain of a route accrues to the shippers and of a ground improvement
to the holders and the drawers (CAP-A2); a rule that valued them from the owner's
recipes would value them at 0, and one that valued them from the region would read
an aggregate (`spec-world.md` 9.4.1). A pack that wants a merchant role to build
the route it ships over states the worth, which is the internalisation that CAP-A2
names, as data. The number is the pack's, a placeholder like every other.

### 5.4 The cost

```
Inputs_p = m * sum over inputs i of ( q_i * Centre_i )              the recipe's inputs and the wear of its tools,
                                                                    as cost_r of spec-agents.md 7.1.4, for m executions
Opp_p    = J(c, cap_now) * Duration_p / 10 000                       what the best job would earn in the ticks of the project
Cost_p   = Inputs_p + Opp_p
```

`m` is 1 for a build and `m_plan = ceil(missing / Units)` for a repair (5.8): the
planning ignores stock, as AGT 7.2 does. A tool of the recipe (`Requires`; CAP's
rules do not forbid one on a build recipe) is a purchase that stays an asset and
is not consumed, so it is not in the cost of the test beyond its wear (the wear is
in `Inputs_p`); the price of a tool the household does not hold is in the **outlay**
(5.6.1), where tier 2 buys it (`PurposeTool`, AGT 7.3.6). The opportunity cost is the value of the
job the household gives up: it has one job at a time, so while it builds it
produces nothing else (`spec-infrastructure.md` 4.4.2, CAP-A3). Its **upkeep is
not a cost of the test**, because it is eaten with or without the project; it is
in the outlay (5.6), because the household has no income during the job.

### 5.5 The test

5.5.1. **The test.** A project `p` **passes** when `Benefit_p` is above 0 and

```
Benefit_p * 10 000  >=  Hurdle_a * Cost_p                           128-bit products
```

with `Hurdle_a` the agent's hurdle (5.5.2), reduced by the commit band when the
household already holds the inputs (5.5.3). A hurdle of 10 000 is break-even; the
default is above it (10).

**Standing refusals.** A project is **weighed** only when the world would start it
once the household has bought what it lacks. The world's reason for a start of 0
is the **first** of the order of `spec-infrastructure.md` 4.2.2, and `no_input`
precedes `no_endowment`, `no_title`, `rent_funds`, `not_unlocked`, `facility_cap`,
`no_target`, `store_refused` and `nothing_to_repair`, so while the inputs are not
held the reason of `MaxExecutions` hides every one of them. The rule therefore
does not read that reason. It reads the per-recipe field **`Standing`** (5.2.2): the
first reason of the order, other than `no_input` and `no_tool`, that would refuse a
start of one execution **were the inputs and the tools held** (`zero_yield`,
`no_site`, `no_facility`, `no_endowment`, `no_title`, `rent_funds`, `not_unlocked`,
`facility_cap`, `no_target`, `store_refused`, `nothing_to_repair`), or empty. The
world derives it at P2.5 from the limits it already computes, none of which
reads the household's inputs (an owed edit, 17.2 row 4 and 9). A project with a
`Standing` that is not empty is **not weighed**, whatever its benefit: the
household would otherwise save for a start it is refused every tick and sit at
`Floor + Goal` for good, or buy the inputs and sell them back again and again. A
household none of whose projects is weighed has no goal and is counted `Standing`
(12.1), with the `Standing` reason in the `Debug` line. The settlement gate
(`not_unlocked`, `spec-infrastructure.md` 8) is one of them, so `Classes` carries no `Open` field.

`rent_funds` is a standing refusal here although money cures it: the outlay has no
term for a rent (a let draw's rent depends on the ration of the tick, which the
household cannot see; `spec-land.md` 4.5.1), so a household on a rent-bearing
class is not saved for and sees no project while its money does not cover the
rent; it is weighed again in the tick in which the world no longer refuses it. A
pack that needs the saving to include the rent has no way to say so in version 1
(open question 14). A start refused after the project was weighed for a
transient reason (`no_input`, `no_tool`) leaves the inputs held and the test is
evaluated again next tick (5.7); a start refused for a standing reason shows in
`Standing` at the next tick and ends the weighing of that project until it
changes, with no state kept (FR-SAV-047).

5.5.2. **The hurdle and its spread.**

```
Hurdle_a = HurdleBP + floor( Weyl32(AgentID) * HurdleSpreadBP / 2^32 )         in [HurdleBP, HurdleBP + HurdleSpreadBP)
```

`Weyl32` is the fixed per-agent spread of the library (`spec-world.md` 9.1.1; one
helper in `internal/ordering`): no creation draw, nothing saved, and the same
value at every call. The device is the merchant's fee-share spread
(`spec-merchants.md` 5.7) and the migration margin's. It turns a herd into a
**supply curve of investors**: at a given margin only the agents whose hurdle is
below it pass, and as capacity is built and the margin falls the later agents
stop passing. The rule of WLD 9.1.1 that two spreads acting on one decision must
not share a value is met, since no rotation acts here.

5.5.3. **The commit band.** A household is **committed** to a project when it
already holds at least half of its inputs by value:

```
Committed_p  <=>  2 * HeldValue_p >= Inputs_p       HeldValue_p = sum over inputs of min(Stock_i, q_i * m) * Centre_i
Hurdle'      =  Hurdle_a * (10 000 - CommitBandBP) / 10 000
```

It is a hysteresis that is a function of holdings and not of memory (7: no state):
a household that has spent its money on inputs would sell them back at a loss if
the test flipped with a belief, so it continues at a slightly lower bar. Without
it a price that wanders around the hurdle makes the household buy and sell the
same inputs (SAV-A4; the model has 1 to 3 flaps a run without the band and none
with it).

5.5.4. **Which project.** Of the projects that pass and are reachable (5.6.5), the
one with the greatest `Benefit_p * 10 000 / Cost_p` (a cross-multiplication in 128
bits); ties go to a repair before a build and then to the lower recipe ID. At most
one is chosen.

5.5.5. **No project while survival binds.** No project is chosen, and its goal is
0, for an agent with `Hunger` above 0, or with `Money` below `ReserveMoney` (the
survival rule's own tests, `spec-agents.md` 7.1.3), or in the own-use mode
(`spec-subsistence.md` 5): a household that is short does not save for a facility.
The event counts it (`Survival`, 12.1). The bar comes first: it also bars the keep
of 5.5.6.

5.5.6. **A project in progress.** The rule is stateless and reads the `Job` group.
For an agent not barred by 5.5.5 whose active job is of a recipe with a `Builds` or
`Repairs` clause, **no project is chosen**: the inputs were taken at the start
(P2.7), so `HeldValue_p` is 0 again and a second set of inputs would otherwise be
bought for a facility already being built. Its goal is the **keep of the rest of
the job**, `UpkeepCost * (Remaining + 1 + SafetyTicks)`, the money the outlay
held for the upkeep (5.6.1) at the ticks that are left, so that the goal does not
fall to 0 in the middle of the job and release the money for wants. It is counted
`Building`. At the tick the job completes the facility is in `Facilities` (the
view is built from the records, 5.2.1): for a computed site `cap_now` has risen
and `dJ` is 0 for a weight-1 household, for a class with a row `n_held` has
risen towards `Count`, and for a repair `missing` has fallen, so the project that
was just finished does not pass again unless it is worth another. The effect of the
facility on the recipes that need it begins at P8.3b (CAP 4.3.3) and does not change
this.

### 5.6 Funds, the goal and the planned recipe

5.6.1. **The outlay.** For the chosen project:

```
ToolBuy_p   = sum over the recipe's tools t of  max(0, m - Stock_t) * Centre_t      the tools it does not hold
Outlay_p    = Inputs_p + ToolBuy_p + UpkeepCost * ( Duration_p + 1 + SafetyTicks )
HeldValue_p = as in 5.5.3 (the held tools count at their centres, up to m)
```

The upkeep term is the money the household needs in hand to eat through the job
and the tick in which its first income can be seen (`spec-world.md` 6.1.8: money
earned at P4.1 is spendable at the next P3.2). `SafetyTicks` is the margin, and it
should be at least `SlackTicks` of the credit block when the role also borrows
(the warning `safety_below_slack`).

5.6.2. **Funded.** The household is **funded** for the project when

```
Money + HeldValue_p  >=  Outlay_p + Floor                  equivalently  Money >= Goal + Floor
```

5.6.3. **The goal.** `Goal = max(0, Outlay_p - HeldValue_p - Tier2_p)` for the
chosen project, funded or not, and 0 with no project (the job in progress has the
keep of 5.5.6). While saving, the cap is closed to tiers 3 and 4; once funded, the
target keeps the money in place until the inputs are bought. As the inputs are
bought `HeldValue_p` rises and `Goal` falls by the same amount, so the target does
not jump.

`Tier2_p` is the money that the plan of this very tick has committed to the
project's own inputs and tools as tier 2 (zero in every tick but the one in which
`StartsIn` is 1). `avail` of the cap (AGT 7.4.4) is already net of that spend, so
without the term the inputs would be counted twice in the tick they are bought,
once in `avail` and once in `Goal`, and `e` would be lowered by them again; with it
the target is the money that is still to be held (example A3, 16.5).

5.6.4. **The planned recipe.** When the household is funded, its planned recipe
`r*` of `spec-agents.md` 7.2 is the project's recipe and `k*` is `m`: then 7.3's
`Job_c` makes the project's inputs tier 2 (`PurposeInput`, not subject to saving),
and they are bought in the tick in which `StartsIn` is 1. When it is not funded,
the default's planned recipe stands: the household goes on with its ordinary work
and saves.

5.6.5. **Reach.** A goal is a standing withdrawal of the household's demand for
buffers and wants (3.3.1), so it is taken only when it can end. With `Need_p` what
the household still lacks, `n` the records of its ring (4.1.1) and `C_n` the
cumulative `income - cost` over them, and `T_p` the service time of the project
(`T_p` of 5.3 for a build, `T_rep` of 5.8.1 for a repair: the ticks of benefit the test counted):

```
Need_p   = max(0, Outlay_p - HeldValue_p + Floor - Money)          zero for a funded household
reachable  <=>  Need_p * n  <=  max(0, C_n) * T_p                    128-bit products; with n = 0 only Need_p = 0
```

that is, the household can save what it lacks, at its own mean surplus per tick,
within the ticks of service it is counting on. A project that passes and is not
reachable is not chosen; if none of the household's passing projects is reachable
it has no goal and is counted `Unreachable`, and it buys its buffers and wants as
a household with no project does. It is re-evaluated every tick with no state: a
household whose money has risen, or whose surplus has, becomes reachable and
then saves for a bounded time (at most the service time: `T_p`, at most `PaybackTicks`, for a build; `T_rep`, at most `PaybackTicks * n_f`, for a repair).
Because spending on wants is not in `cost_i`, the withholding of a goal does not
change `C_n`, so the test does not flap on the household's own saving. The bound is
a rule constant; a tunable for it is open question 13.

### 5.7 The request

For a free agent of a role with a block, `Work` returns:

1. the default's request, unchanged, when survival binds (5.5.5), when a job of
   a project recipe is already in progress (5.5.6; the agent is not free) or when no
   project is chosen;
2. otherwise, if the project's recipe has `MaxExecutions` of at least 1 (the
   world's `k0`: the inputs are on hand and no limit of `spec-infrastructure.md`
   4.2.2 is 0), `{Recipe: p, Executions: 1}` for a build and
   `{Recipe: p, Executions: MaxExecutions}` for a repair;
3. otherwise the default's request: the household works while its inputs arrive.

The request does **not** depend on the work share of the target-income rule: a
household whose ratio is above its target and which would rest has idle labour,
and a build is the use of it (the ratio then falls, since a job's inputs are
netted from income, and the rule asks for work in the ordinary way). The
opportunity cost of 5.4 is taken at full work, which is conservative. A refused
start (`spec-infrastructure.md` 4.2.2) leaves the inputs held; for a transient
reason the test is evaluated again next tick, and for a standing one the project
is no longer weighed (5.5.1).

### 5.8 Repair

5.8.1. For a class `c` of a recipe with `Repairs`, with `n_f` facilities of the
class held in the agent's region (the view), `missing` the sum of
`Durability - Condition` over them and `Units` the clause's:

```
gate     :  min over them of  Condition * 10 000 / Durability  <=  RepairBelowBP      (0 never repairs)
m_plan   =  ceil( missing / Units )
restored =  min( missing, m_plan * Units )
dJ_1     =  J(c, cap_now) - J(c, max(0, cap_now - Capacity_c))
T_rep    =  min( floor( restored / rate ),  PaybackTicks * n_f )
Benefit  =  dJ_1 * T_rep / 10 000                                    ( PerTick * T_rep with a Worth row )
```

`rate` is that of 5.3.1 for one facility, with `k = min(w, cap_now)`. Cost and
test are 5.4 and 5.5 with `m = m_plan`.

5.8.2. **Why this shape.** A site serves at least one execution while its
condition is above 0 (`spec-infrastructure.md` 3.3.2, the `ceil`), so the loss from
a worn site is not a capacity cut now: it is a ruin later. A repair therefore buys
**ticks of service** for the capacity, and its benefit is the value of one
facility's capacity for those ticks, summed over the facilities. **Why a gate.** A
repair of a nearly new facility voids units (CAP 4.3.2) and idles the household
for nothing; the threshold is the pack's. A pack that wants repairs never sets 0.

### 5.9 What the rule does not know

- **Others.** It does not see that others are building, because no view carries
  an aggregate or another agent's facility (5.2.2). The delay between a decision
  and the capacity is invisible to it: the cobweb of SAV-A3 is the cost of local
  information, damped by the spread, the funds and the decay, not removed.
- **Hires.** An employer's workshop serves its hires' executions
  (`spec-infrastructure.md` 5.1.2), and the rule counts the household's own
  weight. A pack for an employer role states a `Worth` row for its site class.
- **Its own effect.** It prices the output at its current beliefs: a facility that
  lowers the price it sells at is not seen.
- **Risk, tolls, taxes.** None is read. A ruin by decay is (it is `rate`); a loss
  to RSK or a levy of FIS is not.
- **The reserve of others' use.** A facility used by many is not valued by its
  owner for them (5.3.3).

## 6. Financing

### 6.1 Self-financed by default

6.1.1. A project is **funded from the household's own money**: it is planned only
when `Money + HeldValue_p >= Outlay_p + Floor` (5.6). A funded household buys the
inputs as tier 2 and has the upkeep of the job and one tick more in hand. The
default borrower of `spec-credit.md` 10.2 then finds no gap: its `need` is
`UpkeepCost * H + Tier2 - Money`, with `H = Duration + 1 + Late + SlackTicks`, a
funded household that buys at once has `Late` 0, and `Tier2` is valued at the
limits of the plan, which lie in the belief's range and may be above the centres
that `Inputs_p` uses. So `need` is not above 0 when
`Floor + UpkeepCost * (SafetyTicks - SlackTicks)` is at least that premium of the
limits over the centres, which a floor of the default's three ticks of upkeep and a
`SafetyTicks` at least `SlackTicks` give for a belief of ordinary width. The two
defaults therefore do not conflict, and a household with both a block and a credit
block saves for its project and asks for nothing (the warning `safety_below_slack`,
11.2, says when the margin is short).

6.1.2. **While it saves**, the household is not funded and its planned recipe is
the default's: if it has a credit block the default borrower reads that ordinary
plan, which is CRD's rule and is not changed.

### 6.2 Credit, and why it is not the default

6.2.1. A project's return arrives **per tick over its life**, and a loan of
`spec-credit.md` is a bullet: principal and interest fall due at the end of the
term, paid in full by one pass at P1.7a (`spec-credit.md` 8). A loan taken to
build is therefore due when the facility has served a few ticks of its life, with
the household's own income unchanged (the facility is not an output and a build
is booked as a cost with no return, `spec-infrastructure.md` 4.4.2). To repay it
the household must have accumulated the sum out of its surplus, which is saving
again, and one that has not goes into arrears and then into the failure ladder.
That is the debt-service squeeze of CRD-A1, and the default would give it to
every household with a project. So the default asks for no loan for a project.

6.2.2. **What a consumer may do.** Its strategy may append a `CreditRequest` of
the shortfall `Goal + Floor - Money` with a term of its own choosing, inside
`MaxCreditTerm`, and fund the project sooner (`spec-credit.md` 5.1, 10.4). The
rule of this spec still decides the project; the loan changes only when the
household is funded. What would make it safe is an **instalment claim**, a
schedule of payments in place of a bullet, which `spec-credit.md` does not have
and which is the first ask of this spec to it (open question 1; a note).

6.2.3. **What it costs to leave it out.** Only households that can accumulate the
outlay invest: a poor household never does, and capital goes to the households
that already have money (SAV-A6). A household that cannot reach it within the
service time does not withhold its demand for a goal it will not finish (5.6.5).
That is the price of safety in version 1, and the register says so.

### 6.3 Titles and other assets are not bought here

6.3.1. **A title** (`spec-land.md`) is not bought by this module. The note that
`spec-land.md` 18.1 (a) raises ("a decision to acquire a title is SAV's, with CRD's
credit, and calls `TransferTitles`; it needs a field in the plan of a
`Strategy`") is **left open, with what was learned**:

- **No observable return.** The household sees the rent in force (`Posted`,
  `spec-land.md` 4.6.1) but not what a title would earn: a holder is paid the rent
  on the let units that tenants draw and that the attribution of `spec-land.md`
  4.4.2 assigns to it, which depends on other households' requests and on the
  ration, and no view carries another's draws or shares. A margin test with an
  unobservable return is a guess, which the module must not make.
- **No seller.** The only holder that is not a household is the pool, which holds
  what removed holders leave and lets it (`spec-land.md` 6.4.2); nothing sells. A
  purchase from the pool would be a sale at a price the world would have to set,
  and money paid to the pool leaves households' hands for the relief the pool pays
  (STB-L63): a new loop that wants its own entry.
- **The shape, if a consumer asks.** A plan field `TitleBids`, admitted and
  matched in a round after the last plan, the way CRD's offers are
  (`spec-credit.md` 6), against the pool's shares at a price that is a stated
  multiple of the entitlement's rent at the reference price; the transfer is
  `TransferTitles` with `Price`. Open question 2.

6.3.2. **A claim** (CRD open question 7, `TransferClaims`) and **a facility for
a price** (`spec-infrastructure.md` 11.3 transfers by decree, no price) are the
same: a purchase of an existing asset between households needs a counterparty
that wants to sell and a price, and neither exists in version 1. What the module
does is build and repair, which needs neither.

6.3.3. **Goods as savings.** A household that holds goods in place of money
(a buffer) does so by the default's `Target` and the room of storage
(`spec-agents.md` 7.3, `spec-storage.md`); that is not a decision of this spec,
and a store facility is a project like any other (a `Worth` row).

## 7. What the player and a consumer's strategy see and do

7.1. **The fallback.** An external agent whose strategy is `Directed` has the
default as its fallback (`spec-agents.md` 10.3, 10.4): the directive's `Job`
overrides `Work` and the rest is the default's. A player whose role has a block
therefore saves and invests by 5 with nothing asked of the consumer, and its
goal withholds buffers and wants as any agent's does. **A consumer that wants its
player free gives it a role with no block**: a role is the unit and there is no
per-agent switch (the same reasoning as CRD 10.5). The directive gains no field.

7.2. **A consumer's strategy.** It receives the `Saving` group of 5.2. The two
rules are exported as pure functions of the view, as the default's rules are
(`spec-agents.md` 6.7; `spec-credit.md` 10.4):

```go
// SavingTarget returns the floor, the goal and the target S* of the agent.
func SavingTarget(v *AgentView) SavingTargetResult

// ChooseProject returns the project that passes the test of 5.5 and is the
// best of those that do, and whether there is one, with its benefit, cost,
// outlay and whether the household is funded for it.
func ChooseProject(v *AgentView) (ProjectChoice, bool)
```

A wrapper holds a `DefaultStrategy` in a named field (architecture.md 4.7) and may
call them, ignore them, or call the default's `Trade` and `Work` with a view of
its own making whose `Saving.Block` it has changed: the default reads its
tunables from the view (`spec-agents.md` 7).

7.3. **A query.** `Saving(agent)` (an owed row to `spec-integration.md` 6.2, query
class 2, `spec-integration.md` 6.1.3: `O(the entity)`) returns, for an agent whose
role has a block, `Floor`, `Precaution`, `Drawdown`, `Goal`, `Idle`, and the chosen
project with its `Benefit`, `Cost`, `Outlay` and `Funded`, or the keep of a job in
progress (`Building`); and the zero value with `ErrNotSaver` for any other. It is
a pure function of saved state, so it is the same on a restored world
(FR-SAV-040). The first consumer's interface can say "you are saving 600 for a
facility, 420 so far".

7.4. **Not offered** (N-131; INT is minimal): a call that sets a goal or moves
money into savings; a savings account; a call that makes a household build
now. A consumer that wants one sets a facility with `AddFacilities`
(`spec-infrastructure.md` 11.1) or gives the player a directive.

## 8. Hoarding, velocity and the measure

### 8.1 Idle money

8.1.1. The **idle money** of an agent is

```
Idle = max( 0, Money - S* )
```

in minor units, a total of the agent (weight included, as `Money` is). It is money
that **no rule of the household holds for a purpose**: not the reserve, not the
buffer, not the goal. It is a measure and not a decision: the household has not
chosen to hold it, and the default has no rule that reads it.

8.1.2. **Where it comes from.** The cap of AGT 7.4.4 spends a share of the excess
`e` each tick, and the share falls as `e` rises, with a bound at
`SpendMaxShare * H / 10 000`. A household whose surplus exceeds that bound
accumulates without limit; one whose surplus is below it settles where the cap
equals its surplus. The first kind is the household of an employer
(`spec-employment.md` 8.2.2, EMP-A3) or of a rent holder
(`spec-land.md` 7); the second holds a stock that is large but steady.

### 8.2 The identities

With the module on every identity of the ledger and of MON holds unchanged, and
two statements follow (FR-SAV-036):

8.2.1. **Saving is not income.** In each tick the money paid in trades equals the
money received in them, so `sum of spending = sum of receipts` over all agents,
and the welfare income of a household nets the cash it spends on inputs
(`spec-welfare-and-wages.md` 2.1). In a closed economy the household-weighted
mean ratio over a window is `1 + W / U` (DEM 7.1, MON 8.2 item 4), with `W` the
want spending and `U` the upkeep cost, **where stocks are unchanged over the
window**: buffers bought in the window are "consumed later as upkeep" and are not
in it. Where they are, a term `Other` is added: the spending that is neither upkeep
consumed nor netted (buffers added to stock, tools), so the ratio is
`1 + (W + Other) / U`. `Other` is **this spec's** addition for a window of
changing stocks, and is 0 over a window of unchanged ones (where DEM's and MON's
form holds as written). **A goal or a buffer lowers `W` and `Other` and so lowers the
mean ratio**; a project's inputs are `Input` spending, netted at the buyer and
income at the seller, so they add nothing to the aggregate ratio. SAV adds no
income.

8.2.2. **In a closed economy the households' saving is the others' dissaving.**
With a fixed `M` and no boundary flow, the sum over all accounts of the change in
balance is 0 in every tick (`spec-ledger.md` 9). A household that raises its
balance does so by taking money from another's: from a household whose income
falls, from the pool (relief paid less than the estates return) or from an
escrow. **The sum of the savings targets `S*` can exceed the money the households
hold**, and then no household reaches its target, buffers and wants are never
bought and the work rule runs at full (the condition of STB-L45, AGT-A4). The
warning `saving_target_exceeds_money` checks it at the reference prices (11.2).

### 8.3 What returns idle money to circulation

| Outlet | How | Owner |
|--------|-----|-------|
| The cap | a decaying share of the excess each tick, to buffers and wants | AGT 7.4.4 |
| Wants | tier 4, funded from the cap | DEM 4.5 |
| Projects | the inputs of a build or a repair, bought at once (a lump, SAV-A2) | this spec 5 |
| Lending | a lender role lends a share of what it holds above its own keep; the borrower spends it | CRD 10.3 |
| The holding charge | a rate levy on the balance above an exemption, with the bound `m* = E + g / rho`; the sink is returned by a grant elsewhere | MON 9.3, D42 |
| The pool and the estates | what a dead household held goes to the pool, which releases a share each tick as relief | EST 2.6, MON D12 |
| Outside trade | imports pay money out of the economy; exports bring it | EXT |

A household with no project on its list, no lending role, no wants and no charge
on its class holds its surplus for good: the **hoard with no outlet**, a hazard
and not a defect (SAV-A7). The module offers one outlet, projects, and only where
a project passes the test.

### 8.4 The measure, and velocity

8.4.1. `spec-money-supply.md` 8.4 defines the **velocity** `V` of a window as the
value of trade over `M` (IND `Turnover`, row 8) and says that hoarding lowers it.
This spec adds the quantity that says how much is hoarded: the sum of `Idle` over
the households of a role and region, and its share of `M_h`. It counts households
only: the money a treasury collects and does not spend is FIS's own hoard
(`spec-fiscal.md`, the warning `treasury_idle`), and the pool's is `MoneyPool`
(`spec-money-supply.md` 3.1), each its own row of `Sectors`. **It is partial.**
`Idle` exists only for the agents of a role that has a block, so a sum "in total"
is the sum over the roles with one, and the households of a role with no block are
not in it; a pack that wants a role measured gives it an inert block (all
defaults: no buffer, no `Worth`, a project list that need not exist, so the
warning `saver_no_project` is informational), which opts the role in to the
measure and, as 3.2 says, to the saver's rules.

8.4.2. **The snapshot.** For each (region, role) whose role has a block, the
snapshot's `RoleRow` (`spec-observability.md` 5.3.4) gains four fields, each a
pure function of saved state (N-125, FR-OBS-043), computed by the world by the
rules of this spec for the role's agents whatever strategy they run (as the
floor's count is). They are **not** a constant number of reads per agent, so they
are not class `S` of `spec-observability.md` 5.5.1: each is an `O(T)` drawdown and
a project test per agent with a block, at snapshot time, a new cost class `Q` (the letter `W` is OBS 3.3.9's volume class) that is
owed to 5.5.1 and 5.5.2 there (12.5; edit 15):

| Field | Meaning |
|-------|---------|
| `SavingHouseholds` | the households (the sum of weights) of agents with a goal above 0 |
| `GoalMoney` | the sum over agents of `Goal` |
| `BufferMoney` | the sum over agents of `max(0, Precaution - ReserveMoney)` |
| `IdleMoney` | the sum over agents of `Idle` |

8.4.3. **The IND preset.** A preset `Idle` (an owed row to `spec-indicators.md`
9, with the asks of its 21): per region and role and in total, `IdleMoney`, its
share of the households' money `M_h` (`Sectors`, row 7), the buffer and the goal as
shares, and the **velocity of the money that is not idle**,
`rdiv(10 000 * turnover, M - IdleMoney)`, beside the velocity of row 8. The
identity `turnover = M * V` is the definition and asserts nothing about prices
(`spec-money-supply.md` 8.4); the preset is a measure and no rule reads it. With
8.4.1's partial coverage, "the money that is not idle" is `M` less the idle money of
the roles with a block, so the households of the other roles count as not idle
in it; the preset says which roles were covered.

## 9. Position in the tick, draws and cost

9.1. **No new position.** Every act of this module is inside a position that
exists:

| ID | What this module does there | Owner of the position | Draws | Events |
|----|-----------------------------|-----------------------|-------|--------|
| P2.5 | `Work` of an agent with a block returns the project's request when the inputs are on hand (5.7); the world counts the requests of project recipes | AGT 6.2, 7.1; PRO 4.2 | none | none (a refusal is CAP's `Debug` line) |
| P3.2 (b) | `Trade` of an agent with a block: the target `S*` replaces the reserve in `e`; the planned recipe is the project when funded (5.6); the world evaluates the rules of 5 once per agent with a block when it builds the view (as it does the drawdown) and keeps the counts of 12.1 in scratch, whatever strategy runs | AGT 6.2, 7.4 | the default's, one per intent it considers (9.2) | none |
| P3.6b (the letter after STO's P3.6a, before FIS's P3.6c; `spec-world.md` 6.5) | the aggregate event `SavingPlanned`, per (region, role) ascending, **after `WantShortfall` and `RoomLimited`**; the order is `WantShortfall`, `RoomLimited` (P3.6a), `SavingPlanned`, then FIS's toll `FiscalFlowApplied` (P3.6c, money flows) | OBS | none | `SavingPlanned` |
| P8.6 | the per-tick counters of this module are cleared with the others | WLD | none | none |

9.2. **Draws.** The module takes **no** random number at any position, and a
module that is on with no block takes none either (`spec-world.md` 7.4.2). The
default strategy's per-intent draws are unchanged in rule: one for each intent it
considers. Two things move that count with no project planned. A target `S*`
above the reserve (a goal while saving, or a buffer above the reserve after any
drawdown) lowers the cap and so the budgets that admit an order with a positive
quantity, and WLD 7.3.1 counts "orders the agents' budgets admit" (the rows D7 and
D8); a planned project changes **which** intents are considered (the project's
inputs in place of the ordinary job's). The count of draws of a tick is therefore a
function of the configuration and of the state, as it is for any change of a
budget or a planned recipe (WLD 7.4), and it differs from the count of the same
world with no block as soon as some household of a role with a block has `S*`
above its reserve. The stream is positional (WLD 7.6.1): a paired comparison
across the module's parameters is exact only **until some household's `S*`
exceeds its reserve** (`spec-experiments.md` 7). Every field of 10.2 moves `S*` or
the chosen project, so each carries the flag `d` of `spec-calibration.md` 4.2.1.

9.3. **Cost.** Per agent with a block, per tick: the drawdown is `O(n)` with `n`
at most the window (4.1.4); the test is `O(recipes * inputs)` plus a scan of the
agent's facilities of the classes it names (at most `MaxFacilitiesPerAgent`). The
module allocates nothing per step beyond the scratch of the view (NFR-SAV-002).

9.4. **Order against other modules in a tick.** A facility completed in tick `t`
is a record from P2.1 (P2.7 for a job of one tick) and is **not in the effect
table until P8.3b** (`spec-infrastructure.md` 4.3.3). The `Facilities` of the view
are built from the records, so the decision of tick `t` itself, at P3.2 (and at
P2.5 for a job that completed at P2.1), already sees the new facility with its
`Capacity` (5.5.6); the recipes that `Need` the class still see `no_facility` in
tick `t` and run from `t + 1`. A repair restores the record at once and the effect
at P8.3b. No loop runs inside a tick. The sentence that CAP 9.1 gains for this is
owed (edit 9).

## 10. Parameters

### 10.1 The module and its types

10.1.1. The types are those of 3.2.1: `Saving` (the module, three tunables) and
`Saver` (the block of a role, six fields) with `ProjectWorth`. The module is on
when `Config.Modules.Saving` is non-nil, and with no role carrying a block it is on
and inert (FR-SAV-001).

10.1.2. **Where each lives, and why.** The three tunables the whole module shares
(`SafetyTicks`, `HurdleSpreadBP`, `CommitBandBP`) are the module's. The six that
describe a role's attitude (the buffer, the hurdle, the horizon, the repair gate and
the worth) are the role's, because a role that holds a site and a role that ships differ in them and
a pack states it by role (the form of `Role.Merchant`, `Role.Credit`).

### 10.2 The fields and the bounds

Every field is a tunable of the pack and has an owner, a bound and a doc line
(architecture.md 9.5, 9.7). **Every default is a placeholder** (N-015, N-027); the
off value of an amplifier (no buffer) is a structural default and the rest are
guesses.

| Field | Scope | Meaning | Default | Bound |
|-------|-------|---------|---------|-------|
| `SafetyTicks` | module | ticks of upkeep cost beyond the job and one tick that the outlay holds (5.6.1) | 2 | 0 to `MaxSafetyTicks` |
| `HurdleSpreadBP` | module | the half-open spread of the hurdle by agent (5.5.2) | 20 000 | 0 to `MaxHurdleSpreadBP` |
| `CommitBandBP` | module | the fall of the hurdle for a household that holds half its inputs (5.5.3) | 2 000 | 0 to `MaxCommitBandBP` |
| `PrecautionShareBP` | role | share of the drawdown held as a buffer (4.2) | 0 (no buffer) | 0 to `MaxPrecautionShareBP` |
| `PrecautionCapTicks` | role | the buffer is at most this many ticks of upkeep cost (4.2) | 30 | 0 to `MaxPrecautionCapTicks` |
| `HurdleBP` | role | the return per unit of cost a project must give; 10 000 is break-even (5.5.1) | 15 000 | 10 000 to `MaxHurdleBP` |
| `PaybackTicks` | role | the horizon over which a facility's service is counted (5.3) | 180 | 1 to `MaxPaybackTicks` |
| `RepairBelowBP` | role | the share of durability at or below which a facility is considered for repair; 0 never (5.8) | 5 000 | 0 to 10 000 |
| `Worth` | role | rows: the worth of a whole facility of a class per tick, for up to `Count` facilities held (5.3.2) | none | at most `MaxWorthRows`, each class once, `PerTick` 1 to `MaxWorthPerTick`, `Count` 1 to `MaxWorthCount` |

The bounds are in `bounds.go` (architecture.md 9.6), chosen so that every product
of this spec fits in 128 bits (11.3):

| Bound | Placeholder | Why |
|-------|-------------|-----|
| `MaxSafetyTicks` | 100 | a margin of ticks of upkeep, not a plan |
| `MaxPrecautionShareBP` | 20 000 | up to twice the worst hole; more is a hoard by design |
| `MaxPrecautionCapTicks` | 1 000 | at most a few years of upkeep, below the largest effective window |
| `MaxHurdleBP` | 100 000 | a return of up to ten times the cost |
| `MaxHurdleSpreadBP` | 100 000 | the same width |
| `MaxCommitBandBP` | 5 000 | a half of the hurdle at most is given up |
| `MaxPaybackTicks` | 36 500 | a hundred years of a 365-tick year; the facility's own life binds earlier |
| `MaxWorthRows` | 16 | a few classes per role |
| `MaxWorthPerTick` | `MaxMoney / MaxPaybackTicks` | so that `PerTick * T_p` stays within `MaxMoney` |
| `MaxWorthCount` | 16 | a few facilities of a class per household; the world's own bound (`MaxFacilitiesPerAgent`) binds beyond it |

### 10.3 Rule constants

Beside their code, not parameters (architecture.md 9.6): the drawdown (4.1); the
form `S* = max(Reserve, Precaution) + Goal` and its single place in the cap (3.2);
the scale 10 000 of a value per tick and truncation toward zero (5.3.1); the job
value `J` (5.3.1); the rate and the service time (5.3.1, 5.8); the one facility a
build asks for (5.1.3); the planning that ignores stock (5.3.1, 5.4); the survival
bar (5.5.5); the commit rule "at least half by value" (5.5.3); the choice by
benefit over cost with a repair first (5.5.4); the standing refusals and which
reasons cure by buying (5.5.1); the keep of a job in progress (5.5.6); the
`+ 1` tick of the outlay (5.6.1); the netting of the tick's tier 2 spend (5.6.3);
the reach bound, the service time (5.6.5); the reasons of 12.3.

### 10.4 Rows for the parameter-owner table

For `specs/README.md`; every field a row with its default and a validation test
(architecture.md 9.7, testing.md 6.7). All are owned by this spec:

| Parameter | Scope | Owner |
|-----------|-------|-------|
| `Saving` | Modules (module; nil = off) | `spec-saving-and-investment.md` section 3.2 |
| `SafetyTicks`, `HurdleSpreadBP`, `CommitBandBP` | Saving | section 10.2 |
| `Saver` | Role (the saver block; nil = no saving behaviour) | section 3.2 |
| `PrecautionShareBP`, `PrecautionCapTicks`, `HurdleBP`, `PaybackTicks`, `RepairBelowBP`, `Worth` | Saver | section 10.2 |
| `Class`, `PerTick`, `Count` | ProjectWorth | section 3.2, 10.2 |

**Classification in the registry of knobs** (`spec-calibration.md` 6; the handoff
has the rows): `PrecautionShareBP` is a knob and the **strength** of SAV-A1 (the
amplifier) with the off value 0; `PrecautionCapTicks` is a damper strength (SAV-D3);
`HurdleBP` and `PaybackTicks` are knobs; `HurdleSpreadBP` is the damper strength of
SAV-D2 and `CommitBandBP` of SAV-D1; `RepairBelowBP` is a knob (SAV-A5);
`SafetyTicks` is a structural choice with a derived condition (at least CRD's
`SlackTicks`); `Worth` rows are pack data with no range. No search range is known
(`spec-calibration.md` 4.3).

## 11. Validation

### 11.1 Errors

Each has a sentinel and a failing-case test (testing.md 3.9). A path names the
field, for example `roles.worker.saver.hurdle_bp`. The sentinels are root
sentinels mapped as N-075 asks.

| Sentinel | Rejects |
|----------|---------|
| `ErrOutOfBounds` | a field outside its bound (10.2, `Count` included); a `RepairBelowBP` above 10 000 |
| `ErrModuleDependency` | a `Saver` block with `Modules.Saving` absent; a `Worth` row with `Modules.Infrastructure` absent. There is no condition on the welfare ring: it exists for every role (`spec-config.md` 3.2: `WelfareWindow` and `TargetRatio` are plain fields, there is no module switch), so a buffer needs none |
| `ErrUnknownFacilityClass` | a `Worth` row that names a class not defined (CAP's sentinel) |
| `ErrDuplicateID` | a class twice in one role's `Worth` |
| `ErrNotSaver` (new) | the query `Saving(agent)` for an agent whose role has no block (7.3) |

### 11.2 Warnings

A warning never fails validation (`spec-config.md` 7.2). Each has a rule name and a
path.

| Warning | Condition |
|---------|-----------|
| `saver_no_project` | a role with a block and no recipe with `Builds` or `Repairs` on its list: only the buffer applies (informational) |
| `worth_unused` | a `Worth` row for a class that no recipe on the role's list builds or repairs |
| `worth_on_site` | a `Worth` row for a site class: it replaces the computed benefit (informational) |
| `project_below_hurdle` | at the reference prices no project of the role passes the test of 5.5, so it would never be chosen (informational; a pure function of the configuration) |
| `safety_below_slack` | a role with a `Saver` and a `Borrow` block whose `SafetyTicks` is below its `SlackTicks` (6.1.1) |
| `saving_target_exceeds_money` | the sum over roles with a block of households times `Floor` plus the outlay of the best project, at the reference prices, exceeds the money of the start (8.2.2); a check of the start, as `reserve_exceeds_money` is |
| `repair_never` | a role with a block and a `Repairs` recipe whose `RepairBelowBP` is 0: it never repairs (informational) |
| `spread_below_margin` | `HurdleSpreadBP` is below the ratio, in basis points, of the benefit to the cost of the role's best project at the reference prices less `HurdleBP`: every agent passes at once (informational; the model of 15.2 is the reason) |

### 11.3 What is not validated here

- **Overflow.** With the bounds of 10.2 and those of `spec-config.md` 4.1
  (`MaxMoney` 10^18, `MaxWeight` 10^6): a `net_r` is below `MaxMoney`, so `pv` is
  below 10^22; `J` multiplies it by a weight (below 10^28); the product of `dJ` and
  `T_p` (at most 36 500; a repair's `T_rep` at most 36 500 * 64 = 2.4 * 10^6, 5.8.1) is below 3 * 10^34. `Benefit` and `Cost` **saturate at
  `MaxMoney`** (a checked helper: a project whose benefit saturates passes, and one
  whose cost saturates fails), so `Benefit * 10 000` is below 10^22 and
  `Hurdle * Cost` below (`MaxHurdleBP` + `MaxHurdleSpreadBP`) * `MaxMoney` = 2 * 10^5 *
  10^18 = 2 * 10^23; the reach product `Need * n` is below 3 650 * 10^18 and
  `C_n * T_p` below 3 650 * 10^18 * 2.4 * 10^6 (8.8 * 10^27), all inside 1.7 * 10^38. Every product is
  therefore inside 128 bits at the largest bounds, **so this module raises no
  `ErrOverflow`** and has no failing-case test for one: the 128-bit helper
  (`spec-currency.md` 7) is used for each product and a saturation is never a wrap
  (architecture.md 3.5; FR-SAV-043, tested by fuzz at the bounds).
- **That a project is worth it.** The test judges the household's own numbers; a
  project that never pays in the setting is the pack's choice.
- **That the spread is wide enough.** It is a warning, not an error (the model of
  15.2: 5 000 does nothing and 40 000 staggers).

## 12. Observability

### 12.1 Events

One event, gated by the module (an event of a module that is off is not emitted,
`spec-observability.md` 4.1). It has no map and no pointer and is built only when an
observer was supplied (observability-and-logging.md 4.5). Section 17 gives the rows
for `spec-observability.md` 4.1 and 4.3 (N-126).

| Event | Position | Class | Fields (besides `Tick`) |
|-------|----------|-------|-------------------------|
| `SavingPlanned` | P3.6b, per (region, role) ascending, after `WantShortfall` and `RoomLimited` (the fiscal tolls follow at P3.6c); emitted only when one of the **triggering counts** is above 0 | A (aggregate) | `Region`, `Role`; **ride along, never trigger:** `Households` (the households of the role's agents that were given a `Trade` view), `Goal` (money, the sum of `Goal`), `Buffer` (money, the sum of the excess of the buffer over the reserve), `Funded` (of `Passed`, funded), `Committed` (of `Passed`, committed); **triggering:** `Saving` (households with a goal above 0), `Buffered` (households whose buffer is above the reserve), `Passed` (households with a project chosen), `Building` (households with a project job in progress, 5.5.6), `NoReturn` (households with a project weighed that did not pass or had no value), `Standing` (households none of whose projects was weighed because of a standing refusal, `not_unlocked` included, 5.5.1), `Unreachable` (households that had a project pass and none reachable, 5.6.5), `Survival` (households barred by 5.5.5), `Builds` (requests of a build recipe at P2.5, one per agent), `Repairs` (requests of a repair recipe at P2.5, one per agent) |

A household is in **at most one** of `Survival`, `Building`, `Passed`,
`Unreachable`, `NoReturn`, `Standing`, in that precedence; one with no project on its
list is in none (and may be in `Buffered`). The counts are computed **by the world**
from the pure rules of this spec (`SavingTarget`, `ChooseProject`) for the agents of
a role with a block, whatever strategy they run, as the floor's count is, so a
consumer's strategy that fills no notes still has them; `Builds` and `Repairs` are
the requests that came back from `Work` and count what was asked, not what started
(a refusal is CAP's `FacilityLimited`, a start is its `RecipeExecuted`).

It is **bounded by the configuration** (regions times roles), not by agents or
executions: an aggregate of the kind of `WantShortfall` and `MerchantsPlanned`, and
it is emitted on occurrence, as `WantShortfall` and `RoomLimited` are
(FR-STO-036 is the pattern). It is the counter of every silent rule of this module
(FR-STB-013, N-045): the goal that withholds buffers and wants (`Saving`, `Goal`),
the keep of a job in progress (`Building`), the buffer (`Buffered`, `Buffer`), a
project that was weighed and not taken (`NoReturn`, `Unreachable`, `Standing`), a
household barred by survival (`Survival`). A refusal of the world to start a
requested build is CAP's `FacilityLimited`, not repeated here.

### 12.2 Logs

Through the injected logger, constant lowercase messages, the keys of
observability-and-logging.md 3.3.2 (two new keys, `goal` and `buffer`, money in minor
units as `int64`):

| Level | Message | Attributes | When |
|-------|---------|------------|------|
| `Info` | "world created" gains the module name `saving` in `modules` | | once, at construction |
| `Warn` | the warnings of 11.2 | `rule`, `path` | once each at construction |
| `Debug` | "project chosen" | `tick`, `agent`, `recipe`, `goal`, `buffer` | per chosen project, guarded by `Enabled` |
| `Debug` | "project declined" | `tick`, `agent`, `recipe`, `reason` | per household and tick in which a project was weighed or barred and not taken, guarded; `reason` is a constant of 12.3, or for a standing refusal the world's reason of `spec-infrastructure.md` 4.2.2 |

No `Info` and no `Warn` is emitted inside the tick: `SavingPlanned` is the counter
(observability-and-logging.md 3.2.2; N-126), so no `Warn` is needed and none is
added.

### 12.3 Reasons

Constants of the root package (`event.go`), with their key strings in
`internal/logkeys` only (N-017): `no_return` (a project was weighed and did not pass
or had no value), `standing` (no project was weighed, 5.5.1), `unreachable` (a
project passed and none was reachable, 5.6.5) and `survival` (barred by 5.5.5). They
are the values of `reason` in the `Debug` line and the four constants of the metric
attribute `economysim.bar`; `Building` has no reason (a household in progress is
not declined). No `OrderRejected` reason and no ledger reason is added, and
`economysim.reason` is not extended (it is a closed set of flow, boundary and
order-rejection reasons, rules 5.4.1; `no_return` is a constant of `economysim.limit`
already).

### 12.4 Metrics

Derived from the event and the snapshot (observability-and-logging.md 5.3.1),
integers, bounded attributes (5.4). Section 17 gives the rows.

| Name | Instrument | Unit | Attributes | Fed by |
|------|-----------|------|------------|--------|
| `economysim.saving.goal` | Gauge | `{minor_unit}` | `economysim.role`, `economysim.region` | the snapshot, `RoleRow.GoalMoney` |
| `economysim.saving.buffer` | Gauge | `{minor_unit}` | `economysim.role`, `economysim.region` | the snapshot, `RoleRow.BufferMoney` |
| `economysim.saving.idle` | Gauge | `{minor_unit}` | `economysim.role`, `economysim.region` | the snapshot, `RoleRow.IdleMoney` |
| `economysim.saving.requests` | Counter | `{request}` | `economysim.role`, `economysim.project` | `SavingPlanned` (`Builds` as `build`, `Repairs` as `repair`) |
| `economysim.saving.barred` | Counter | `{household}` | `economysim.role`, `economysim.bar` | `SavingPlanned` (`NoReturn` as `no_return`, `Standing` as `standing`, `Unreachable` as `unreachable`, `Survival` as `survival`) |

Two new attributes: `economysim.project` with two constants (`build`, `repair`) and
`economysim.bar` with four (`no_return`, `standing`, `unreachable`, `survival`); both
are bounded by a closed set (5.4.1). The requests are counted and not starts
(a start is CAP's). For the reference chain on `three` with two roles that have a
block, the gauges add `2 * 3` series each, `requests` `2 * 2` and `barred` `2 * 4`.
The gauges are fed by a snapshot and are absent, not zero, until one is passed.

### 12.5 Snapshot

The four fields of 8.4.2 on `RoleRow`, all plain data and all functions of saved
state (N-125). There is no per-agent row: the per-agent figures are the query of 7.3.
**Cost class.** Each is a pass over the agents of the roles with a block, with an
`O(T)` drawdown and a project test per agent (`O(T + recipes * inputs +
facilities)`): class `Q`, not the constant reads of class `S`. With the placeholder
sizes of `spec-observability.md` 5.5.2 and 1 000 agents with a block at a placeholder
40 reads each, 40 000 reads, about 0.4 ms at 10 ns, which brings the snapshot from
about 1.1 ms to about 1.5 ms (3% of the tick of 53 ms): the row of class `Q` and the
total are owed to 5.5.1 and 5.5.2 there (edit 15). A world with no role with a
block pays nothing.

### 12.6 Cost with no observer

With the default observer and logger no event is built and no log call allocates
(observability-and-logging.md 4.5): the counts exist only to build the event and are
kept only under the flag that an observer was supplied; the rules themselves are
evaluated once per agent with a block (9.1) whatever the observer. The module's
steps allocate nothing per call in steady state (NFR-SAV-002).

### 12.7 The Observability entry (observability-and-logging.md 9)

1. New failure path or rejected input: none returned; a project the world refuses
   is CAP's, counted by its event (12.1). 2. New state transition: none by this
   module (a build is CAP's `FacilityChanged`); the event is an aggregate with no
   map. 3. New decision point in a strategy: yes (the target and the project): a
   `Debug` line with the inputs and a `reason` (12.2). 4. New loop: the drawdown
   and the test, guarded by `Enabled` for the log; the event is built only with an
   observer. 5. Changed event or metric: none; the instrument `requests`, the
   instrument `barred` and the attributes `economysim.project` and `economysim.bar`
   are new (5.4.1). 6. New metric attribute: `economysim.project` (two values) and
   `economysim.bar` (four). 7. `Info` or `Error`: none new. 8. Logging an error also
   returned: none. 9. Tests: FR-SAV-029 to FR-SAV-032, FR-SAV-038 to FR-SAV-040.

## 13. State contributed to a checkpoint

13.1. **None.** The module adds no state, no section and no per-agent field
(N-128). The target, the goal, the test and the request are pure functions of the
view of the tick: of the agent's money and stock, its belief centres, the ring of
its welfare records, its facilities, its active job (recipe and `Remaining`), the
configuration and the agent ID. Every one
of those is saved by its owner (the ledger, `spec-beliefs.md` 14, `spec-welfare-and-wages.md`
13, `spec-infrastructure.md` 15, the configuration hash).

13.2. **A restore** needs nothing of this module, and a restored world's plan,
request and snapshot rows are the original's at the same boundary (FR-SAV-040). The
module's name is in the configuration hash with the block and the module's three
fields, so a restore against a different block or module fails `ErrCheckpointConfig`
as any configuration change does (`spec-checkpoint.md` 5.1).

13.3. **Why no memory of a goal.** A goal that persisted (a project the household
has "decided on") would be per-agent state saved once in the agent record and a
rule for when it ends. The stateless form costs a hysteresis that is a function of
holdings (the commit band, 5.5.3) and a goal that follows the household's present
beliefs. The cost is that the goal moves with the prices of its inputs (a case of
SAV-A2); the gain is that nothing can be lost or stale across a restore.

13.4. **The inventory row** for `spec-checkpoint.md` 4.2 is "SAV: none, and why"
(an owed edit, section 17), the form of the rows for modules that hold no state.

## 14. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. The checklist of
`spec-stability.md` 9 was followed: 9.1 each rule of sections 3 to 6 was read for
the chain it creates; 9.6 the delays were checked against the neighbours (14.2); 9.7
each damper's tunable is a field with a row (10.4); 9.10 each is classified (10.4).

**SAV-A1 Precautionary saving amplifies a downturn.** *Kind:* loop (reinforcing,
bounded; a case of L45, AGT-A4). *Loop:* a fall in income (+) the drawdown of the
window (+) the buffer (-) the excess `e` over the target (+) discretionary spending
(+) other households' income, and so income. Two negative links, so polarity R.
*Delay:* a tick for the cash flow (WLD-A1) and up to the window `T` for the drawdown
to leave the ring. *Owner:* SAV 4; touches L31 and L45. *Damper:* SAV-D3 (the cap in
ticks), SAV-D4 (tiers 1 and 2 are outside the target, so food and inputs are never
withheld), the concavity and bound of the AGT cap (D27: the cut is at most
`SpendMaxShare * SpendHalfTicks * UpkeepCost / 10 000` per household and tick), D11
and D12. *Undamped:* none known; a high share with a long window and a large
discretionary share is the case to watch. *Condition:* the buffer moves a household
only where its money lies between the reserve and the buffer; in the model of 15.1
the trough of the aggregate is unchanged and the recovery is within 2 ticks, because
the household is limited by rebuilding its flat reserve, and the individual holds 40%
more money and has 23% fewer unfed ticks at a share of 20 000. *Neighbours (9.6):* the
window `T` against the job and the trip (`WindowCycles`). *Test:* SAV-S2. *Status:*
model (an invented flow, not the library).

**SAV-A2 The goal withdraws demand and gives it back as a lump.** *Kind:* bias (a
pulse), with a reinforcing part where goals coincide. *Loop:* a goal is set (+) the
target (-) buffers and wants bought (-) the sellers' income (-) their own surplus
and saving; at funding the inputs of the project are bought at once (+) demand for
the inputs (the lump of `spec-agents.md` 7.3.2). In the stateless rule the goal
also follows the prices of its inputs: a rise in them raises the outlay and the goal
while many households save, which is a small bubble in the input market (the case
that 13.3 names). *Delay:* the time to save, the outlay over the surplus (43 ticks in
example A, at most the service time by the reach rule of 5.6.5), and the build's `Duration`. *Owner:* SAV 5.6. *Damper:* SAV-D2 (the
spread of the hurdle) and the dispersion of the households' funds, which stagger the
funding; tiers 1 and 2 untouched (SAV-D4); one job at a time; the reach rule
(5.6.5), which takes a goal only where it can end within the ticks of service, so a
household with a passing project and a surplus too small for it is not held at
`Floor + Goal` for good (it is `Unreachable`, and buys its buffers and wants), and
the in-progress keep (5.5.6), which holds the upkeep of the job and not a second
outlay. *Undamped:* identical households of equal wealth in one region, all funded
in one tick; a pack whose surplus per tick times `PaybackTicks` is just above every
outlay (the loss of demand is then `PaybackTicks` ticks of the goal for every such
household, the bound of the reach rule, not a permanent one). *Condition:* the
funds are spread over more than the saving time, or the hurdles are. *Neighbours:*
the lockstep of L22 and L34. *Test:* SAV-S3. *Status:* hand.

**SAV-A3 The investment cobweb: the decision side of CAP-A1.** *Kind:* loop
(balancing with a delay: a cobweb, and a one-way ratchet because a facility is worn
and not unbuilt). *Loop:* the price of what a facility makes (+) the margin of the
recipe that needs it (+) projects that pass (+) builds started (+) capacity, after
`Duration` and the commissioning tick (+) output (-) the price; and decay (-)
capacity. *Delay:* the saving time, `Duration`, one tick of commissioning
(`spec-infrastructure.md` 4.3.3) and the belief lag. *Owner:* SAV 5 for the
decision, CAP 4 and 6 for the mechanism; merges with CAP-A1, which names the
decision as "the strategy's or SAV's". *Damper:* SAV-D2 (a spread of the hurdle is a
supply curve of investors), the funds test (SAV-D5: only those who have saved), the
horizon bound by life (SAV-D6), decay (CAP-D2), the belief rule's stock term (L06)
and the halt (D24). *Undamped:* identical households, all funded at once, with a
narrow spread: the model's K goes from 0 to the whole group; **the default
`HurdleSpreadBP` of 20 000 is below the condition below, so at the default the
damper does not damp the worst case** (the model: spread 20 000, everyone funded,
K reaches 60); a pack that wants the stagger sets the spread to the range of its
own margin, which `spread_below_margin` warns of. A class with a `Worth` row is
bounded by its `Count` per household (5.3.2) and by CAP's `facility_cap`, so a
row that does not diminish cannot build without limit. *Condition:* the spread
of the hurdle covers the range of the margin to cost ratio (here from 1.5 to 5.1 in
units of the cost, so a spread of about 40 000) and the funds are dispersed. In the
model of 15.2, 60 households and a site with a life of 300 ticks: with everyone
funded at tick 0 the number of working sites goes from 0 to 60 and the mean is 9
(106 builds, 86 ruins); with dispersed wealth and a spread of 40 000 and a commit
band of 2 000 it stays between 12 and 24 with a mean of 20 (109 builds, 86 ruins, no
flap); a spread of 5 000 does nothing. It is a model of the rule in an invented
market with identical beliefs (the worst case), not of the library. *Neighbours
(9.6):* the occupation cobweb of L36 at about two periods; `Duration` against the
effective window (a match by construction); the belief adjustment of a few ticks.
*Test:* SAV-S3. *Status:* model.

**SAV-A4 The project test flaps without memory.** *Kind:* hazard (chattering).
*Loop:* a belief near the hurdle (+) the test passes in one tick and fails in the
next (+) inputs bought and sold back (-) the household's money by the market's
spread (+) its need to save again. *Delay:* a tick. *Owner:* SAV 5.5.3 (cousin of
AGT-A7, a choice with no hysteresis). *Damper:* SAV-D1, the commit band, which is a
function of holdings and not of memory; the smoothing of beliefs (BEL 5). *Undamped:*
a household whose inputs are also its upkeep (the stock is eaten and not held).
*Cases with the same shape:* the **completion tick** (the project that has just
been built is in `Facilities`, so it does not pass again and its inputs are not
bought a second time, 5.5.6). A **standing refusal** that the world's first reason
`no_input` masks is not a case: the field `Standing` (5.5.1) carries it with the
inputs not held, so the project is never weighed and no inputs are bought for it.
*Condition:* a band above the typical move of the margin between ticks. In the model
the flaps per run are 1 to 3 without the band and 0 with it. *Test:* SAV-S3.
*Status:* model.

**SAV-A5 A replacement echo.** *Kind:* hazard (a lockstep of ruin). *Loop:* a cohort
of facilities built within a few ticks (+) ruins `Durability / rate` ticks later,
together (+) capacity falls at once (+) the margin and the price rise (+) many
builds at once, again. *Delay:* the facility's life (a period). *Owner:* CAP 6
(decay) with SAV 5.8 (repair). *Damper:* repair below `RepairBelowBP` (each facility
is repaired at its own condition, so a cohort's ruin is spread), SAV-D2 and the
dispersion of funds. *Undamped:* a pack with no repair recipe and one cohort of
builds. *Condition:* a role lists a repair recipe and `RepairBelowBP` is above 0 (the
warning `repair_never`). In the model with no repair, a start with everyone funded
and a wide spread still takes the number of sites to 0 at about tick 300. *Test:*
SAV-S3. *Status:* model.

**SAV-A6 Capital goes to those who can fund it.** *Kind:* loop (reinforcing,
concentration). *Loop:* money (+) the outlay is reached (+) a facility owned (+)
income (+) money; a household that cannot reach the outlay never invests. *Delay:*
the saving time. *Owner:* SAV 6.2 (the project is self-financed). *Damper:* the
reach rule (5.6.5) bounds the withholding of demand of a household that cannot
finish its goal, and nothing here bounds the concentration itself, by design:
credit-financed projects (CRD, open question 1) are the remedy; decay (CAP-D2) runs unrepaired capital down; MON's holding charge (D42) and
the pool's release (D12) move money. *Undamped:* in version 1, by design. *Condition:*
the outlay over a household's surplus per tick is below the facility's life. In the
model (an outlay of 3 000, savings of 8 to 16 a tick, a life of 300) the slowest
third of the households held 10 163 ownership ticks and the fastest 9 905: no
concentration at those numbers. *Neighbours:* LND-A5 (concentration of titles).
*Test:* SAV-S4. *Status:* model (a negative result at stated numbers), hand
otherwise.

**SAV-A7 A hoard with no outlet.** *Kind:* hazard (bias; a case of F2, L31 and L45).
*Loop:* a surplus above the cap's bound (+) idle money (-) money offered (-) velocity
`V` (-) sellers' income (-) the surplus of the next household. *Delay:* the
cap's own, tens of ticks. *Owner:* SAV 8 with MON 8.4 and EMP-A3. *Damper:* the cap
(a decaying share), wants (D30, D31), lending (CRD), projects (SAV 5), the holding
charge (D42, with the bound `m* = E + g / rho`), the pool's release (D12).
*Undamped:* a household with no project on its list, no lending role, no wants and no
charge on its class. *Condition:* an outlet exists for every class that earns above
the cap's bound; the measure is `IdleMoney` and its share of `M_h` (8.4). *Test:*
SAV-S5. *Status:* hand.

### 14.1 Dampers this spec adds or specifies

| ID | Damper | Where | Status |
|----|--------|-------|--------|
| SAV-D1 | The commit band: a household that holds half its inputs continues at a lower hurdle; a stateless hysteresis from holdings. Cost: a project is continued a little after it stopped being worth it | 5.5.3 | specified (a parameter, `CommitBandBP`) |
| SAV-D2 | The spread of the hurdle by agent (Weyl32): a supply curve of investors in place of a herd. Cost: a household with a high hurdle forgoes a project that would pay. **The default of 20 000 damps nothing in the worst case of the model** (it needs about 40 000, SAV-A3), and the warning `spread_below_margin` fires on example A itself (54 000 - 15 000 above 20 000): the default is a placeholder that leaves the damper weak, the pack sets it | 5.5.2 | specified (a parameter, `HurdleSpreadBP`), weak at the default |
| SAV-D3 | The cap of the buffer in ticks of upkeep cost. Cost: a long-job producer with a larger cycle is under-buffered | 4.2 | specified (a parameter, `PrecautionCapTicks`) |
| SAV-D4 | Saving acts only above tiers 1 and 2, through the concave cap: food and a job's inputs are never withheld and the cut is bounded. Cost: a household below its target buys no want | 3.2, 3.3, 4.5 | specified (rule) |
| SAV-D5 | The funds test: a project is planned only when the household can pay its outlay and eat through the job, and the default asks for no loan. Cost: the poor do not invest (SAV-A6); with the reach rule (5.6.5) a household that cannot finish the goal within the service time does not withhold its demand for it, and one that can withholds for at most the service time (`T_p` of a build, `T_rep` of a repair) | 5.6, 6.1 | specified (rule) |
| SAV-D6 | The horizon bound by life: a return is never counted beyond `PaybackTicks` or beyond what the facility's condition serves. Cost: a durable project is under-valued | 5.3 | specified (rule, a parameter, `PaybackTicks`) |

### 14.2 Delays near neighbours (checklist 9.6)

| Delay here | Neighbour | Relation | Entry |
|------------|-----------|----------|-------|
| the buffer's pulse, up to the window `T` | the effective welfare window and the migration settling rule (`spec-migration.md` 5.2) | the same number by construction; a household that saved through a long job sees its drawdown and its window leave together | SAV-A1 |
| the saving time (tens of ticks) | the review period of the spawner (`spec-spawner.md` 6.2) and the welfare window | near; a household that saves has an unchanged ratio, so neither rule acts on saving as such | SAV-A2 |
| a build's `Duration` and one commissioning tick | the effective window and `MoveCooldown`; the transit time `k` of a shipment (`spec-transport.md` 4) | a match by design (CAP-A3); a route completed changes `k` of new dispatches (CAP-A7) | SAV-A3 |
| a facility's life `Durability / rate` | `PaybackTicks`, the year of `TicksPerYear` | the horizon is below the life by design; a pack's `PaybackTicks` above the life is cut to it | SAV-A5, SAV-D6 |
| the belief adjustment (a few ticks) | the test's inputs (the centres) | the test reads smoothed prices, so a flip of the test is slower than a tick | SAV-A4 |
| `SafetyTicks` | `SlackTicks` of the credit block (`spec-credit.md` 10.2.1) | at least equal when a role has both blocks (`safety_below_slack`) | 6.1.1 |

### 14.3 Scenarios proposed

| ID | Scenario | Measure and pass |
|----|----------|------------------|
| SAV-S1 | The module on with no role carrying a block, over a range of seeds, against the same pack with it off | the state, the snapshot bytes, the events and the final state of the random source are equal (FR-SAV-001, FR-SAV-002). **Harness test** |
| SAV-S2 | A producer with a job of 20 ticks and a seller with bad stretches, with `PrecautionShareBP` 0, 10 000 and 20 000, on `one` with the welfare module on, and a cut of its trade partners' income | unfed household-ticks fall with the share and the money held rises; the trough of the aggregate discretionary spending and the recovery time are not worse than with 0 beyond one window (SAV-A1) |
| SAV-S3 | A role with a site recipe and a build recipe on `one` with CAP on, a price response, with the spread at 0, at 5 000 and at 40 000 and the band at 0 and 2 000, and with and without a repair recipe; a second variant in which the world would refuse the start for a standing reason with the inputs held (`facility_cap` reached, `no_target`) | the number of working sites and its swing (min and max over the last 700 ticks), builds, ruins and flaps against the model's order; with repair no simultaneous ruin of a cohort; no second set of inputs bought in the completion tick of a build; in the variant the project is never weighed, `Standing` is counted, no goal is held and the inputs bought per 100 ticks are 0 (SAV-A3, A4, A5) |
| SAV-S4 | The same with an outlay above the surplus a household can save within a life | ownership ticks by saving rate concentrate in the fastest savers; with an outlay below it they do not (SAV-A6); a household whose mean surplus times the service time is below what it lacks has no goal (`Unreachable`) and keeps buying buffers and wants (5.6.5) |
| SAV-S5 | An employer role with a large surplus, with no outlet, then with a project, then with a holding charge on its class (`overlay/faucet.toml`) | `IdleMoney` and its share of `M_h` rise without an outlet, fall with a project and are bounded by `m*` with the charge (SAV-A7) |
| SAV-S6 | A checkpoint taken with households saving, holding inputs and mid-build, restored and run | the continuation equals the uninterrupted run, the plan and the snapshot rows included (FR-SAV-040). **Harness test** |

## 15. What was run

Three scripts, in `doc/specs/models/sav/` (README beside them), Docker Python 3.12
with `PYTHONDONTWRITEBYTECODE=1`, standard library only, the output captured by the
host shell in `.out` files. They are models of the rules of this spec, not of the
library, with placeholders (N-015, N-027). Run, never with a local interpreter:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/sav":/w:ro -w /w \
  python:3.12-slim python precaution.py > doc/specs/models/sav/precaution.out
```

### 15.1 `precaution.py`

| Part | What it shows | What it does not show |
|------|---------------|-----------------------|
| A | One household, income 160 in a good stretch and 30 in a bad one (a two-state chain, mean bad stretch 6.7 ticks), upkeep 100, window 30, 20 000 ticks. Unfed ticks 2 410, 2 333, 2 141 and 1 862 at shares 0, 5 000, 10 000 and 20 000 (cap 30 ticks); mean money 410, 429, 482 and 576; mean discretionary spending 36, 35, 35 and 34. A cap of 5 ticks at a share of 10 000 gives 2 218 and 455 | other households, prices, the pool |
| B, B2 | A representative household in a circular flow with a leak (income is a share of last tick's spending plus an autonomous part), the autonomous income cut by 10 000 bp for 30 ticks (leak 2 000 bp) and for 60 (leak 3 000 bp). The trough of discretionary spending is 0 at every share; the time to 90% of the old level after the cut is 60 ticks at shares 0 to 10 000 and 58 at 20 000 (62 at every share in the longer cut); the peak buffer is 220, 440 and 880 (1 498, 2 996 and 3 000) | a heterogeneous population, a real flow, prices |
| C | The drawdown of a producer with a 20-tick job over a window of 40 records (two cycles by `WindowCycles` 2): 2 200 | the cost of a loan, the alternative |

The reading is in 4.5.4: the buffer helps the individual and does not deepen the
aggregate trough, because the household is limited by the flat reserve it must
rebuild, which the drawdown outlasts only by the window.

### 15.2 `invest.py`

60 households may each own one site; the price of the output falls with the number
of working sites (`p = (833 - 10 K) / 40`, from 20.8 to 5.8); a site's life is 300
ticks (`Durability` 600, `Decay` 2); a build takes 5 ticks and costs 3 000; every
household has the same belief of the price (the worst case); the rule is the test
of 5.5 with the funds condition of 5.6, a payback of 180 and a hurdle of 15 000.

| Case | Max K | Min and max K after tick 300 | Mean K | Builds | Ruins | Flaps |
|------|-------|------------------------------|--------|--------|-------|-------|
| everyone funded at tick 0, hurdle 15 000 | 60 | 0 and 60 | 9 | 106 | 86 | 0 |
| dispersed wealth, hurdle 15 000 | 60 | 0 and 60 | 37 | 177 | 151 | 2 |
| dispersed wealth, hurdle 10 000 | 60 | 0 and 60 | 37 | 183 | 153 | 3 |
| dispersed wealth, hurdle 25 000 | 58 | 0 and 58 | 39 | 182 | 141 | 0 |
| dispersed wealth, spread 5 000 | 60 | 0 and 60 | 40 | 199 | 155 | 3 |
| dispersed wealth, spread 5 000, band 2 000 | 60 | 0 and 60 | 42 | 200 | 153 | 0 |
| everyone funded, spread 5 000, band 2 000 | 60 | 0 and 60 | 9 | 106 | 86 | 0 |
| everyone funded, payback 60 | 60 | 0 and 60 | 6 | 92 | 66 | 0 |
| everyone funded, spread 20 000 | 60 | 0 and 60 | 8 | 101 | 82 | 1 |
| everyone funded, spread 40 000 | 55 | 0 and 55 | 10 | 108 | 88 | 2 |
| everyone funded, spread 80 000 | 27 | 0 and 27 | 21 | 114 | 94 | 2 |
| everyone funded, spread 40 000, band 2 000 | 55 | 0 and 55 | 10 | 107 | 88 | 0 |
| dispersed wealth, spread 40 000, band 2 000 | 24 | 12 and 24 | 20 | 109 | 86 | 0 |

The reading: a **narrow spread does nothing** (the margin to cost ratio ranges from
5.1 at the highest price to the hurdle at the margin, so a spread of 5 000 is
inside one step of it); a spread of the order of that range **and** dispersed
wealth together stagger entry (K between 12 and 24); everyone funded at once
overshoots at every spread up to 40 000 (K reaches 55 or 60) and is held to 27 only
by a spread of 80 000, and the facilities built together ruin together (the echo,
SAV-A5). The last line of the output: ownership ticks of the slowest third
of the households by saving rate 10 163, of the fastest 9 905 (SAV-A6: no
concentration at an outlay of 3 000 against 8 to 16 a tick over a life of 300).
What it does not show: heterogeneous beliefs, the library's clearing, hires, repair,
and a population that is not identical. **The entry model does not exercise the
rules of 5.5.1, 5.5.6 and 5.6.5**: its households have a state `building` in which they
are not re-tested (so the in-progress rule is assumed and not shown), no standing
refusal exists, and money accrues at a fixed rate whether or not a goal withholds
wants, so the reach rule does not act in it (it bounds the withholding of demand,
which the model does not have). Their arithmetic is in `worked.py` (15.3, 16.5);
their behaviour in a population is SAV-S3 and SAV-S4 in the library.

### 15.3 `worked.py`

The arithmetic of the examples of 16 (A, A2 with the build that no longer passes,
A3 with reach, the keep of a job in progress and the funding tick, B, C, C2 with
the saturation of a `Worth` row, D with the tool), the rules as written, so that the
numbers in the text are checked.

## 16. Worked examples

The settings are invented **examples** and flagged so; every number is a
placeholder.

### 16.1 Example A: a mill (a pre-industrial setting; unit: one grind)

A role `miller` of weight 1, upkeep cost 30 a tick, `ReserveTicks` 3 (reserve 90).
Recipes: `carry` (alternative; net 8 a tick, duration 1), `grind` (needs the site
class `mill`; 4 grain at 10 and 3 flour at 20, net `60 - 40 = 20`, duration 1) and
`build_mill` (30 timber at 8 and 10 stone at 12, duration 5, `Builds` the class
`mill`). The class: `Durability` 600, `Decay` 1, `WearPerUse` 1, `Capacity` 6. The
block: `HurdleBP` 15 000, `PaybackTicks` 180; the module's `SafetyTicks` 2.

**The test** (5.3, 5.4, 5.5; `worked.out`): `pv(grind)` = 20 * 10 000 / 1 =
200 000 and `pv(carry)` = 80 000; no mill, so `cap_now` 0 and
`J(mill, 0) = 80 000`, `J(mill, 6) = max(200 000 * min(1, 6), 80 000) = 200 000`,
so `dJ = 120 000`. `rate = 1 + ceil(1 * 1 / 1) = 2`, so the life is
`600 / 2 = 300` ticks and `T_p = min(300, 180) = 180`. `Benefit = 120 000 * 180 /
10 000 = 2 160`. `Inputs = 30 * 8 + 10 * 12 = 360`; `Opp = 80 000 * 5 / 10 000 =
40`; `Cost = 400`. The ratio is 5.4: `2 160 * 10 000 = 21 600 000 >= 15 000 * 400 =
6 000 000`. The project passes.

**Funds and the goal** (5.6): `Outlay = 360 + 30 * (5 + 1 + 2) = 600`; `Floor` is
the reserve, 90; the household is funded at `Money >= 690`. With 100 and a surplus of
14 a tick it reaches 702 at tick 43. Until then `Goal = 600` and `S* = 690`: it buys
no buffer and no want, and food and the inputs of its ordinary job are bought as
ever. At tick 43 its planned recipe is `build_mill`, tier 2 buys the timber and the
stone, and next tick, with the inputs on hand, `Work` requests the build with one
execution.

**The hurdle by agent** (5.5.2, a spread of 2 000): agent 3 has 16 708, agent 7
15 652 and agent 11 16 596; none is at the bound of 15 000. With the commit band of
2 000, a household that holds the inputs continues at `15 000 * 8 000 / 10 000 =
12 000`.

### 16.2 Example A2: the repair of the same mill

After a time the mill's condition is 120 of 600 (20%, at or below `RepairBelowBP`
5 000). `cap(f) = ceil(6 * 120 / 600) = 2`. The repair recipe restores 100 units for
4 timber (32), duration 1. `missing = 480`, `m_plan = 5`, `restored = 480`,
`rate` 2, `T_rep = min(480 / 2, 180 * 1) = 180`. `dJ_1 = J(mill, 2) - J(mill, 0) =
200 000 - 80 000 = 120 000`, `Benefit = 120 000 * 180 / 10 000 = 2 160`;
`Inputs = 5 * 32 = 160`, `Opp = 200 000 * 1 / 10 000 = 20`, `Cost = 180`: ratio 12.
The repair passes. A build of the same class does not compete: at this state
`cap_now` is 2, so for a weight-1 household `dJ = J(mill, 8) - J(mill, 2) =
200 000 - 200 000 = 0`, its benefit is 0 and it does not pass (`worked.out`); the
ratio 5.4 of 16.1 is that of a build at `cap_now` 0. The repair requests
`MaxExecutions` executions, which the world derives as the least number that
restores the missing units: 5.

### 16.3 Example B: the buffer of a producer with a long job

A role with a job of 20 ticks: inputs of 300 bought at its start, no income for 19
ticks, sales of 2 500 at the end, upkeep 100 a tick. The cycle earns 2 200 against a
cost of 2 000, so it is viable. The effective window is `max(30, 2 * 20) = 40` (two
cycles, `WindowCycles` 2). `net_i` is `-400` in the first tick of a cycle, `-100` in
the next 18 and `+2 400` in the last, so after a cycle the running sum is `+200`,
a new peak. The deepest fall from a peak is inside a cycle: `-400 - 18 * 100 =
-2 200`. With `PrecautionShareBP` 10 000 and `PrecautionCapTicks` 30:
`Precaution = min(30 * 100, 10 000 * 2 200 / 10 000) = 2 200`, and
`Floor = max(300, 2 200) = 2 200`. At a share of 5 000 it is 1 100; with a cap of 10
ticks 1 000. The household holds the working capital of its cycle before it buys a
buffer or a want; a household with a `Borrow` block and no surplus borrows the gap
instead (`spec-credit.md` 20.1).

### 16.4 Example C: a cold cell (a space-colony setting; unit: one cell)

A role `provisioner` with a recipe `build_cell` (200 of one input, duration 4,
`Builds` a class `cold_cell` of kind `store`, `Durability` 400, `Decay` 0). The
role's `Worth` row: `cold_cell`, `PerTick` 15, `Count` 4; the horizon 100 ticks; `Decay` 0 so
`T_p = 100`. `Benefit = 15 * 100 = 1 500`; `Inputs = 200`; the alternative earns 5 a
tick, so `Opp = 5 * 1 * 4 = 20`; `Cost = 220`. The ratio is 6.8 and the project
passes a hurdle of 15 000 (1 500 * 10 000 >= 15 000 * 220). With a `PerTick` of 3,
`Benefit` is 300 and `300 * 10 000 = 3 000 000 < 3 300 000`: it does not pass
(`NoReturn`). Nothing in the rule names a store, a cell or a colony: the class is data
and the worth is the pack's. A household that holds `Count` cold cells has `Benefit` 0 for another (`worked.out`, C2, runs it with a `Count` of 2).

### 16.5 Example A3: reach, a job in progress and the funding tick

Example A's numbers (outlay 600, floor 90, upkeep 30, `SafetyTicks` 2, `T_p` 180;
`worked.out`).

**Reach** (5.6.5). Money 100, a ring of 30 records with `C_n` 420 (a mean surplus of
14): `Need = 600 + 90 - 100 = 590`, `590 * 30 = 17 700 <= 420 * 180 = 75 600`:
reachable, it saves (43 ticks). The same household with `C_n` 60 (a mean of 2 a
tick): `17 700 > 60 * 180 = 10 800`, **not reachable**: no goal, `S* = Floor`, it
buys its buffers and wants, and is counted `Unreachable`. A household with money 690
needs nothing and is reachable whatever its surplus.

**A job in progress** (5.5.6). The mill's build lasts 5 ticks. In the tick in which
the job has `Remaining` 3, the household's goal is `30 * (3 + 1 + 2) = 180`, and
`S* = 90 + 180 = 270`: the money held for the rest of the job's upkeep is not
released to buffers and wants, and no second set of inputs is planned. At the tick
the job completes, the mill is in `Facilities` (`cap_now` 6) and the build has
`dJ = 0`: it does not pass.

**The funding tick** (5.6.3). Money 700; the plan buys the timber and stone, 360, as
tier 2, so `avail = 340`. With the full goal, `S* = 90 + 600 = 690` and `e = 0`;
with the goal net of the tier 2 spend, `S* = 90 + 240 = 330` and `e = 10`. The
inputs are counted once.

**A tool** (5.6.1). A build recipe that `Requires` a tool of 50 that the household
does not hold: `Outlay = 360 + 50 + 30 * 8 = 650`, and the household is funded at
`Money >= 740`.

## 17. Notes settled, and what this spec needs from the other specs

### 17.1 Notes

Every note tagged `SAV` or `ALL`, and the others this spec touched, with what was
done. The text to append to each is in the handoff.

| Note | Status here | Where |
|------|-------------|-------|
| N-046 (EMP, SAV, CRD, STO, RSK, FIS, PLC, LND) | **Applied for SAV; remove SAV from the areas.** Entries SAV-A1 to SAV-A7, dampers SAV-D1 to SAV-D6 and scenarios SAV-S1 to SAV-S6 are in 14, owed to the register | 14 |
| N-016 (ALL, CFG, WEL) | Applied: the rows of 10.4 for the parameter-owner table; no unnamed parameter was found. The registry test is still to be written | 10.4 |
| N-017 (ALL) | Applied: the terms of 17.3; setting words only in the flagged examples | 17.3, 16 |
| N-027 (ALL, REF, STY) | Respected: every number is a placeholder or structural (a hurdle above 1, no buffer by default), the model shows structure and sensitivity and not the stability of any default | all |
| N-103 (ALL) | Followed: the spec, its models and `.supervisor/sav.*` only; the edits handed over. Three slips of the host `python3` (a version check and an empty `-` at the first writing; one more empty `-` in the revision after review), nothing read or written; every model ran in Docker | |
| N-126 (ALL) | Applied: the row of `SavingPlanned` and the metric rows are owed to OBS (12.1, 12.4) and the rules' catalogue changes in the same change | 12 |
| N-128 (ALL, CHK) | Applied: no per-agent field and no section (13) | 13 |
| N-032 (STB, CRD) | **Update line:** a household's own remedy for the cash gap of a long job is the buffer of 4.3.3, beside credit; they are substitutes (AGT-A12 and AGT open question 15 are what it answers in part). The note stays open for STB | 4.3.3 |
| N-054 (WEL, DEM, STB, MON) | **Update line:** saving is not income; a goal or a buffer lowers the want spending and so the mean ratio (`1 + W / U` over a window of unchanged stocks, `1 + (W + Other) / U` with `Other` this spec's term for stocks that change), and a project's inputs add nothing to it (8.2.1) | 8.2 |
| N-021, N-107 (FIS, LED, MON) | **Applied for SAV:** no faucet, no sink, no ledger reason, no money flow (`spec-money-supply.md` 9.5: no return on savings) | 2, 9 |
| N-090 (PRO, SKL, INT) | **Applied:** no `region_bound` refusal; the module holds no obligation tied to a region (a facility is CAP's and stays where built, `spec-infrastructure.md` 7.5) | 13 |
| N-091 (INT) | **Applied:** no slot and no call; one query (7.3) | 7 |
| N-131 (INT, FIS, PLC) | **Respected:** nothing offered beyond the query (7.4) | 7.4 |
| N-045 (STB) | **Applied:** every silent rule of the module is counted in `SavingPlanned` (12.1) | 12.1 |
| N-125 (CHK, LED, WLD, CLR, OBS) | **Applied:** the snapshot rows of 8.4.2 are functions of saved state | 8.4.2 |
| N-130 (OBS, IND, WEL, LED, INT, TRN, PRO) | **Update line:** the preset `Idle` needs no new field beyond the four of 8.4.2; the fields of the note (`IncomeLast` and the others) are not needed | 8.4 |
| N-108 (LND, CAP, EXT) | Read: a ground facility is built by a project like any other; its gain accrues to the holders and drawers (CAP 5.3.4), and the rule values it only by a `Worth` row (5.3.3) | 5.3 |
| N-122 (SPN, MIG, MON, MER, STB) | **Respected:** no cap is added and no place added to `spec-world.md` 10.2; the one bound that binds is CAP's on facilities, refused in the rotation of the starts | 5.7 |
| N-124 (EXP, STB, CLB, WLD) | **Stated:** the module's parameters change the count of draws as soon as some household's `S*` exceeds its reserve (a goal or a buffer lowers the budgets that admit an order) and once a household plans a project (9.2), so a paired comparison across them is exact only until then; each carries the flag `d` | 9.2 |
| N-015 (VIA, STY, CLB, EXP) | Placeholders added: every default of 10.2, the bounds of 10.2 and the numbers of the examples and the models | 10 |
| The note of `spec-land.md` 18.1 (a) (SAV, LND, CRD, AGT) | **Left open, with what was learned:** no observable return and no seller (6.3.1) | 6.3 |

**New notes to add** (area codes in the handoff): (a) **instalment claims** for a
project financed by credit: a claim with a schedule (SAV, CRD); (b) **a title or a
facility bought for a price needs an observable return and a seller** (SAV, LND,
CAP, INF); (c) **a `Facilities` group in the `Trade` view and the clauses in the
per-recipe group** (AGT, CAP); (d) **`FR-CAP-018` carries a carve-out for a role with
a `Saver` block** (CAP, SAV); (e) **the cost of the drawdown is `O(T)` per agent and
tick** (SCL, CODE); (f) **a default for `HurdleSpreadBP` rests on one invented
model** (CLB, STY).

### 17.2 Edits owed to other files

None of these edits is made here (`writer-common.md` 1); the handoff has one table
with the file, the section and the exact change. In short:

| # | File | Section | Change |
|---|------|---------|--------|
| 1 | `spec-agents.md` | 7.4.2, 7.4.4 | the `e` of the cap is `max(0, avail - SavingTarget)`, with `SavingTarget` equal to `ReserveMoney` unless the module is on and the role has a block (`spec-saving-and-investment.md` 3.2); 7.1.2 and 7.1.3 keep the flat reserve |
| 2 | `spec-agents.md` | 7.2, 7.1 | a funded project is the planned recipe (5.6.4); a new 7.1.10 for the request of 5.7 before 7.1.4, independent of the work share, barred by survival and the own-use mode |
| 3 | `spec-agents.md` | 7.1.4 | a note: a recipe with `Builds` or `Repairs` has no output value and is requested only by 7.1.10 (and the default of a role with no block never requests one, `spec-infrastructure.md` 4.5) |
| 4 | `spec-agents.md` | 6.4.1 | the `Saving` group (5.2); the `Facilities` group in the `Trade` view of an agent whose role has a block; `Needs`, `Builds`, `Repairs` and **`Standing`** (the first refusal of CAP 4.2.2 other than `no_input` and `no_tool` that would apply were the inputs and tools held, or empty) in the per-recipe group |
| 5 | `spec-agents.md` | 6.3 | **withdrawn**: the counts of `SavingPlanned` are computed by the world from the pure rules (12.1), so `PlanNotes` gains nothing |
| 6 | `spec-agents.md` | 3.3, 12, 13 | `Role.Saver`; the rows of 10.4; the validation of 11 |
| 7 | `spec-agents.md` | 6.7, 18, 21 (15) | the exported `SavingTarget` and `ChooseProject`; AGT-A4 names SAV-A1 as a case; AGT-A12 (a job longer than the reserve) and open question 15 point to `spec-saving-and-investment.md` 4.3.3 as the household's own remedy |
| 8 | `spec-infrastructure.md` | 4.5.1, 4.5.2, FR-CAP-018 | the default builds and repairs nothing **for a role with no `Saver` block**; with one the project rule of `spec-saving-and-investment.md` 5 requests a build or repair; FR-CAP-018 is amended the same way |
| 9 | `spec-infrastructure.md` | 4.2.2, 9.1, 9.2, 21 (9) | **4.2.2: one sentence**: the world also derives, for a `Builds` or `Repairs` recipe, the per-recipe field `Standing`: the first reason of the order, other than `no_input` and `no_tool`, that would refuse a start of one execution were the inputs and tools held (limits 1 to 6 read no input; `rent_funds` is judged at the money held), or empty, so that a strategy sees a standing refusal that `no_input` masks; the `Facilities` field in the `Trade` view too, for an agent whose role has a block; **one sentence**: the field is built from the records, so a facility completed at P2.1 (P2.7 for a one-tick job) is in it in the same tick with the `Capacity` of its record, while its effect is in the table only from P8.3b (4.3.3, 10.4); question 9 (a default repair rule) is answered by 5.8 |
| 10 | `spec-infrastructure.md` | 16 (CAP-A1, CAP-A5) | the decision is SAV's, SAV-A3 and the repair rule of 5.8 |
| 11 | `spec-credit.md` | 10.3 | a lender with a `Saver` block keeps `max(KeepTicks * UpkeepCost, Floor)` (optional); 10.2 notes that a funded project leaves no gap |
| 12 | `spec-credit.md` | 23, 1 | a new open question: an instalment claim (6.2.2); and in 1 the sentence that expects SAV to supply "the choice to buy a title or a store on credit": SAV's default asks for no loan for a project (6.2) and leaves a purchase of an asset open (6.3) |
| 13 | `spec-demand.md` | 4.5 | one sentence: the cap's excess is over `SavingTarget` |
| 14 | `spec-world.md` | 6.5 (P3.6), 6.11, 9.1.1, 7.4 | P3.6b: the order `WantShortfall`, `RoomLimited` (STO, P3.6a), `SavingPlanned` (P3.6b) and the fiscal tolls (FIS, P3.6c) is ruled; 6.11 the order of events; 9.1.1 "the saver's hurdle spread" among the users of `Weyl32`; 7.3 no row; a note in 7.4 that a target above the reserve and a planned project change the budgets and intents considered |
| 15 | `spec-observability.md` | 4.1, 4.4, 5.3.4, 5.5.1, 5.5.2, 5.8, 6, 7 | the event row `SavingPlanned` (triggering and riding counts of 12.1); the four `RoleRow` fields; the cost class `Q` and its row in the budget of 5.5.2 (12.5); the five metrics with their feeds (`requests`, `barred`); the keys `goal` and `buffer`; the attributes `economysim.project` and `economysim.bar` and the reasons `no_return`, `standing`, `unreachable`, `survival`; no `Warn` row; FR-OBS-003 input row |
| 16 | `rules/observability-and-logging.md` | 3.3.2, 4.3, 5.3, 5.3.1, 5.4.1 | the two keys; the event row (an aggregate; in the "aggregates" sentence and in the modules-off sentence); the five instruments and their feeds (the three gauges are fed by a snapshot); the attributes `economysim.project` (`build`, `repair`) and `economysim.bar` (`no_return`, `standing`, `unreachable`, `survival`) with their constants; `economysim.reason` is not extended; a fresh read-receipt tag (supervisor) |
| 17 | `spec-indicators.md` | 9, 21 | the preset `Idle` (8.4.3) and the asks |
| 18 | `spec-integration.md` | 6.2, 7.1 | the query `Saving(agent)` (class 2, `O(the entity)`) and `ErrNotSaver` |
| 19 | `spec-config.md` | 3.2, 3.7, 4.1, 6.2, 7.3, 7.4, 8.1 | the module, the fields, the bounds, the file keys, the errors, the warnings, the hash |
| 20 | `spec-checkpoint.md` | 4.2 | a row "SAV: none" |
| 21 | `spec-calibration.md` | 6 | the nine rows of 10.4 |
| 22 | `spec-stability.md` | 4, 5, 8, 10 | the entries, the dampers, the scenarios; the backlog row for SAV is removed |
| 23 | `spec-money-supply.md` | 9.5, 8.4 | "the savings returns of SAV" becomes "SAV pays no return on savings and has no faucet"; the measure of hoarding is `Idle` |
| 24 | `spec-employment.md` | 11.3, 14.2 | the row "Investment" is written (a `Worth` row for the workshop class); the cure list of `employer_no_outlet` gains it |
| 25 | `spec-welfare-and-wages.md` | 2.1 | one sentence: the ring is read by the saving module's drawdown (a pure function) |
| 26 | `spec-land.md` | 6.2.1, 18.1 | a pointer: the decision to buy a title is left open by `spec-saving-and-investment.md` 6.3 |
| 27 | `spec-reference-chain.md` | 7 | the overlay `overlay/saving.toml` for SAV-S2 to SAV-S5 |
| 28 | `spec-subsistence.md` | 5 | one sentence: the own-use mode bars a project (5.5.5) |
| 29 | `specs/README.md` | index, code table, owners, glossary, decisions | the index row, the code row, the owner rows, the glossary, the decisions |
| 30 | `roadmap.md`, `documentation_index.md` | the SAV row; the models | the roadmap row `exists`; the rows for the spec and `doc/specs/models/sav/` |
| 31 | `notes.md` | index, open notes | the note updates and the new notes; a fresh read-receipt tag |
| 32 | `rules/architecture.md` 2.6 and `AGENTS.md` layout | at the first code (N-019) | `internal/saving` and `saving.go`; the SAV terms to the `doc.go` glossary |
| 33 | `spec-stylised-facts.md` | 5.11, FR-STY-098 | rows in 5.11 for SAV-S2 to SAV-S5; SAV-S1 and SAV-S6 are two more scenarios of the class "harness test" in the list of FR-STY-098 (S77, S78, S84 to S90 become eleven) |

### 17.3 Terms (N-017)

**Saving** (the module; also money held back), **saver** (a role with the block),
**savings target** (`S*`, 3.2.2), **floor** (of the target: the larger of the reserve
and the buffer), **precaution buffer** (4.2), **drawdown** (4.1), **goal** (money set
aside for a project), **project** (a build or repair recipe of the role), **margin
test** (5.5), **benefit** and **cost** (of a project), **job value** (`J`),
**hurdle** (and its **spread**), **payback horizon**, **commit band**, **outlay**,
**funded** (a household for a project), **worth** (a pack's value of a whole
facility), **idle money** (8.1). The word **investment** is used for a build or a
repair only (6.3), and **hoard** only for the measure of 8. `Floor` is also a word of
the specs (work floor, cost floor, resolution floor, `MinWorkShare`'s floor, the
floor of agents, the subsistence floor, N-017): here "floor of the target" says what
it is. `Goal`, `Floor` and `Benefit` repeat names of other specs in a different
sense and are qualified in the text. They go to the glossary of `doc.go` before code
uses them.

## 18. Requirements

IDs are stable; a requirement that moves keeps its ID (testing.md 4.1). All numbers
are placeholders (N-015, N-027).

### Functional

| ID | Requirement |
|----|-------------|
| FR-SAV-001 | With the module off, or on with no role carrying a block, the world MUST be identical to the world without it: the same state, snapshot bytes, events and final state of the random source. A `Saver` block with the module off MUST be `ErrModuleDependency`. |
| FR-SAV-002 | The module MUST take no random number at any position or call. A world with the module on and no block MUST draw exactly the words of the world without it. |
| FR-SAV-003 | For an agent whose role has a block the savings target MUST be `max(ReserveMoney, Precaution) + Goal` and MUST replace `ReserveMoney` in the excess `e` of the spend cap (`spec-agents.md` 7.4.4) and nowhere else: the survival rule, tiers 1 and 2 and the work share MUST read the flat reserve. |
| FR-SAV-004 | For an agent whose role has no block, or with the module off, the target MUST be `ReserveMoney`. |
| FR-SAV-005 | The target MUST NOT change tiers 1 and 2, the production halt, the choice among recipes other than by 5.7, beliefs or the validity of any order; the sum of limit times quantity of an agent's bids MUST stay at most its available money, and the cut in the cap MUST be at most `SpendMaxShare * SpendHalfTicks * UpkeepCost / 10 000`. |
| FR-SAV-006 | The drawdown MUST be the deepest fall from a running peak of the cumulative `income - cost` over the agent's ring, 0 when there is none, by checked arithmetic. |
| FR-SAV-007 | `Precaution` MUST be `min(PrecautionCapTicks * UpkeepCost, PrecautionShareBP * Drawdown / 10 000)`, 0 with a share of 0, and `Floor` MUST be `max(ReserveMoney, Precaution)`. |
| FR-SAV-008 | The world MUST compute the drawdown from the ring and hand it in the view; a strategy MUST NOT be able to change it. |
| FR-SAV-009 | The value per tick of a recipe MUST be `trunc(net_r * 10 000 / Duration_r)` with `net_r` of `spec-agents.md` 7.1.4 at the agent's centres, and a recipe MUST be valued when it is eligible ignoring stock (a positive yield, `net_r > 0` or a source, not halted). |
| FR-SAV-010 | The job value `J(c, K)` MUST be `max(pv_b * min(w, K), pv_a * w)`, and the gain of a build `J(c, cap_now + Capacity) - J(c, cap_now)`, at least 0. |
| FR-SAV-011 | The rate MUST be `Decay + ceil(WearPerUse * k / Duration_b*)` and the service time `T_p = min(floor(Durability / rate), PaybackTicks)`, `PaybackTicks` for a rate of 0. |
| FR-SAV-012 | The benefit of a site MUST be `dJ * T_p / 10 000`; of a class with a `Worth` row `PerTick * T_p` while the agent holds fewer than the row's `Count` facilities of the class in its region, and 0 at `Count`; of any other class 0; and a project of benefit 0 MUST NOT be chosen. |
| FR-SAV-013 | The cost MUST be the inputs and tool wear of the recipe for `m` executions plus the opportunity `J(c, cap_now) * Duration / 10 000`, planned ignoring stock, and MUST NOT include upkeep. |
| FR-SAV-014 | A project MUST pass only when its benefit is above 0 and `Benefit * 10 000 >= Hurdle * Cost` in 128 bits; benefit and cost MUST saturate at `MaxMoney`. |
| FR-SAV-015 | The hurdle MUST be `HurdleBP + floor(Weyl32(AgentID) * HurdleSpreadBP / 2^32)`, a function of the agent ID alone. |
| FR-SAV-016 | A household that holds at least half of a project's inputs by value MUST use the hurdle lowered by `CommitBandBP`. |
| FR-SAV-017 | Of the projects that pass the one with the greatest benefit over cost MUST be chosen, ties to a repair and then to the lower recipe ID, and at most one. |
| FR-SAV-018 | No project MUST be chosen and the goal MUST be 0 for an agent with `Hunger` above 0, with `Money` below `ReserveMoney` or in the own-use mode (the keep of FR-SAV-046 included); a project with a `Standing` MUST NOT be weighed (FR-SAV-047, the settlement gate included). |
| FR-SAV-019 | The outlay MUST be the inputs plus the price at the centres of the tools the household does not hold plus `UpkeepCost * (Duration + 1 + SafetyTicks)`; the goal `max(0, Outlay - HeldValue - Tier2)` with `Tier2` the plan's own tier 2 spend on the project's inputs and tools in the tick; and a household MUST be funded when `Money + HeldValue >= Outlay + Floor`. |
| FR-SAV-020 | A funded household's planned recipe MUST be the project's, so that its inputs are tier 2 and `PurposeInput` and are bought in the tick in which `StartsIn` is 1. |
| FR-SAV-021 | `Work` of a free agent MUST request a chosen project's recipe when its `MaxExecutions` is at least 1, with 1 execution for a build and `MaxExecutions` for a repair, whatever the work share; otherwise the default's request. |
| FR-SAV-022 | A repair MUST be considered only when the lowest `Condition * 10 000 / Durability` of the agent's facilities of the class in its region is at most `RepairBelowBP` (0 never), with `m_plan`, `T_rep` and the benefit of 5.8. |
| FR-SAV-023 | The default strategy MUST NOT append a `CreditRequest` for a project: with `SafetyTicks` at least `SlackTicks` and the premium of the plan's limits over the centres within `Floor`, a funded household's default borrower MUST find no gap. |
| FR-SAV-024 | A role with no block MUST be unaffected in every respect: its target, its planned recipe and its request are the default's. |
| FR-SAV-025 | The target, the buffer and the outlay MUST be totals of the agent, weight included, and a build MUST ask for one facility whatever the weight. |
| FR-SAV-026 | The module MUST keep no state: no section, no per-agent field; the target, the goal, the test and the request MUST be functions of the view, and a restored run MUST equal the uninterrupted one. |
| FR-SAV-027 | The module MUST add no position, no ledger reason, no money flow, no slot and no `region_bound` refusal. |
| FR-SAV-028 | The `Saving` group MUST be present in the `Work` and `Trade` views only of an agent whose role has a block, and the `Facilities` group MUST be in both. |
| FR-SAV-029 | `SavingPlanned` MUST be emitted at P3.6b once per (region, role) in ascending order when one of `Saving`, `Buffered`, `Passed`, `Building`, `NoReturn`, `Standing`, `Unreachable`, `Survival`, `Builds` or `Repairs` is above 0, never otherwise (`Households`, `Goal`, `Buffer`, `Funded` and `Committed` ride along and do not trigger it), with no map or pointer, and built only when an observer was supplied. |
| FR-SAV-030 | The counts of `SavingPlanned` MUST satisfy `Funded <= Passed`, `Committed <= Passed`, `Saving` the households with a goal above 0 (the keep of a job in progress included) and `Survival` the households barred by 5.5.5; a household MUST be in at most one of `Survival`, `Building`, `Passed`, `Unreachable`, `NoReturn`, `Standing`; all in households (weights) and never agents; and they MUST be computed by the world from the rules of this spec whatever strategy runs, `Builds` and `Repairs` counting the requests that `Work` returned. |
| FR-SAV-031 | The snapshot rows `SavingHouseholds`, `GoalMoney`, `BufferMoney` and `IdleMoney` MUST be functions of saved state, computed for the agents of a role with a block whatever strategy they run. |
| FR-SAV-032 | `Idle` MUST be `max(0, Money - S*)` for each agent. |
| FR-SAV-033 | With the module on every identity of the ledger MUST hold; in a closed economy the sum of the changes of all balances MUST be 0 in every tick. |
| FR-SAV-034 | The errors of 11.1 MUST be raised at construction by the rules stated there, each with a failing-case test; the module MUST raise no `ErrOverflow` (every product is inside 128 bits at the bounds, 11.3). |
| FR-SAV-035 | The warnings of 11.2 MUST be raised by the conditions stated there and MUST never fail validation. |
| FR-SAV-036 | A goal or a buffer MUST lower no income as such: with the module on the mean ratio of a closed economy MUST equal `1 + W / U` over a window of unchanged stocks and `1 + (W + Other) / U` otherwise (`Other` of 8.2.1, this spec's term), and a project's inputs MUST add nothing to it. |
| FR-SAV-037 | The query `Saving(agent)` MUST return the floor, precaution, drawdown, goal, idle money and the chosen project with its benefit, cost, outlay and funded flag, and `ErrNotSaver` for an agent whose role has no block. |
| FR-SAV-038 | With a logger and an observer attached the run MUST be identical to one without; no event MAY be built without an observer. |
| FR-SAV-039 | The module MUST emit no `Info` or `Warn` inside the tick; its warnings are at construction, and a `Debug` line MUST be guarded by `Enabled`. |
| FR-SAV-040 | A checkpoint taken with households saving, holding inputs and mid-build MUST restore a world whose plans, requests, snapshot rows and query results equal the original's. |
| FR-SAV-041 | A world with a facility chain and a saver role MUST be able to build its own capital from a configuration file alone (architecture.md 9.4). |
| FR-SAV-042 | The exported `SavingTarget` and `ChooseProject` MUST return what the default strategy uses for the same view. |
| FR-SAV-043 | Every product MUST be computed with checked arithmetic within the bounds of 10.2; an overflow MUST NOT wrap. |
| FR-SAV-044 | A start of a project that the world refuses for a transient reason (`no_input`, `no_tool`) MUST leave the inputs held and MUST be evaluated again in the next tick with no state kept. |
| FR-SAV-045 | A project MUST be chosen only if reachable: `max(0, Outlay - HeldValue + Floor - Money) * n <= max(0, C_n) * T` (`T` the service time: `T_p` of a build, `T_rep` of a repair, 5.8.1) over the agent's ring (with `n` 0 only a need of 0); a household with passing projects none of which is reachable MUST have no goal and MUST be counted `Unreachable`. |
| FR-SAV-046 | For an agent not barred by survival whose active job is of a recipe with `Builds` or `Repairs` no project MUST be chosen and the goal MUST be `UpkeepCost * (Remaining + 1 + SafetyTicks)`; a facility completed in the tick MUST be in the `Facilities` of the same tick's `Work` and `Trade` views, and the project just completed MUST NOT buy a second set of inputs. |
| FR-SAV-047 | A project MUST be weighed only if its per-recipe `Standing` is empty (the first refusal of `spec-infrastructure.md` 4.2.2 other than `no_input` and `no_tool` that would apply were the inputs and tools held, `rent_funds` and `not_unlocked` included); a project with a `Standing` MUST NOT be weighed, MUST NOT give a goal and MUST be counted `Standing` (the reason in the `Debug` line); the rule MUST NOT read the reason of `MaxExecutions` for it, since `no_input` masks the standing ones. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-SAV-001 | Every iteration MUST be over a sorted slice or an array indexed by a typed ID, never over a map (architecture.md 7.5). |
| NFR-SAV-002 | With the default observer and logger, `Work` and `Trade` of an agent with a block MUST allocate nothing per call in steady state (`testing.AllocsPerRun`, observability-and-logging.md 8.2). |
| NFR-SAV-003 | The drawdown, the job value, the benefit and the test SHOULD have fuzz tests asserting no panic, no overflow within the bounds and the invariants (a drawdown of at least 0, a benefit that does not fall as the capacity rises). |
| NFR-SAV-004 | The drawdown SHOULD be `O(n)` for `n` records and have a benchmark that calls `b.ReportAllocs()` at `MaxWelfareWindow`. |
| NFR-SAV-005 | The pure rules SHOULD live in `internal/saving` over primitive slices and be tested without the world (architecture.md 2.9). |

## 19. Open questions

Each with a next step.

1. **Instalment claims** (6.2.2). A project financed by credit needs a claim with a
   schedule, which CRD does not have. *Next step:* a note to CRD (a new open question
   there); until then a consumer's strategy appends its own request.
2. **A title, a claim or a facility for a price** (6.3). The decision needs an
   observable return and a seller. *Next step:* a plan field `TitleBids` matched in a
   round, when a consumer asks; LND would need to publish what a holder earned in a
   window as a rumour or a view field.
3. **A running drawdown** (4.1.4). `O(T)` per agent and tick. *Next step:* measure
   at the first slice; an incremental form needs a monotone queue per agent saved as
   state.
4. **Heterogeneous beliefs.** The model gives every household one belief, the worst
   case for herding. *Next step:* SAV-S3 in the library, where beliefs move
   asynchronously.
5. **What the buffer should read.** The drawdown reads `income - cost`, with inputs
   netted, so a producer's buffer includes its working capital. A variant that
   leaves the inputs out would give a buffer for income variance only. *Next step:*
   run SAV-S2 with both.
6. **Working harder to save.** A household with a goal does not raise its target
   ratio. A rule that did would add a gain to the labour loop (L14). *Next step:*
   SAV-S2 and SAV-S5 show whether the surplus is enough.
7. **A default for `HurdleSpreadBP`.** 20 000 is a guess from one invented model
   (15.2). *Next step:* CLB's screen (`spec-calibration.md` 7.3) once a pack has
   a site recipe.
8. **An employer's hires.** The rule counts the household's own weight, so an
   employer's workshop is valued by a `Worth` row. *Next step:* EMP's vacancies as an
   input of `J`, if a run shows the row to be a poor stand-in.
9. **A route valued from the owner's own shipments.** A merchant that ships
   over its own route earns the route's gain on its own lanes. The rule would need the
   merchant's lane volumes, which are not state. *Next step:* MER's plan notes as an
   input, or a `Worth` row (as now).
10. **A purpose other than a project.** A stake for a new household is the pool's
    (`spec-population.md` 4.3); a reserve for a trip, or for a durable good a
    household wants to own, is a goal that this module has no object for. *Next step:* none until a consumer asks.
11. **A cadence.** None, by decision (position 9). *Next step:* if SAV-S3 shows a
    herd that the spread and the funds do not damp, the rotation of WLD 9.1 is the
    device; it needs a rule that does not starve a long job.
12. **Repair of a facility that is not the owner's** (CAP open question 5) is ORG's
    or FIS's.
13. **A tunable for the reach bound** (5.6.5). The bound is the service time the
    benefit counts (`T_p`, or `T_rep` of a repair); a pack may want a shorter one (a household should not go
    without wants for most of a horizon). *Next step:* SAV-S4 shows how many
    households sit just inside the bound; a field `ReachTicks` with a row if they
    do.

14. **Rent in the outlay** (5.5.1). A project on a rent-bearing class (a let draw,
    `spec-land.md` 4.5) is not weighed while the world refuses it for `rent_funds`,
    and the household does not save for the rent. *Next step:* a view field with
    the rent the start would owe, read by the outlay, when LND is built and a pack
    has a build recipe that draws a let resource.

## 20. Where this lives in the code

| Concern | Home |
|---------|------|
| `Saving`, `Saver`, `ProjectWorth`, the view group, `SavingTarget`, `ChooseProject`, `SavingTargetResult`, `ProjectChoice`, the query `Saving` and its result, the reasons and the event `SavingPlanned` | root package, `saving.go`; the event in `event.go` |
| The pure rules: the drawdown over primitive records, the job value, the benefit, the cost, the test, the funds and the goal | `internal/saving`, primitives only; the drawdown beside the ring in `internal/welfare` |
| The Weyl value of the ID | `internal/ordering` (shared, N-019) |
| The bounds of 10.2 | root package, `bounds.go` |
| Validation, the sentinels and the warnings of 11 | root package, `validate.go`, `errors.go` |
| Log keys `goal` and `buffer` | `internal/logkeys` |
| Tests | the external `_test` package for the default's behaviour; a table-driven unit test for each rule of `internal/saving`; a golden rendering of the events of the scripted run of 16; fuzz tests under the package's `testdata/fuzz/` |
| The overlay for SAV-S2 to SAV-S5 | `testdata/chains/overlay/saving.toml` (REF) |
